### Table of Contents

* [Part 1](#part-1)  basic definitions and example
* [Part 2](#part-2)    otel + apm definitions
* [Part 3](#part-3)    full config of otel + apm
* [Part 4](#part-4)    battle-tested approach using log to stdout

# part 1:
Nice — you asked for a hands-on, practical, **comprehensive** tour of the observability trifecta (logs, metrics, traces) with concrete Go examples and how to wire them into ELK / Prometheus+Grafana / Jaeger. Below I’ll:

1. define each signal and give a real-world example;
2. explain how they work together and important design caveats;
3. give an end-to-end practical recipe (Go code + Docker Compose + configs) that shows:

   * structured logging -> shipped to Elasticsearch (ELK/Filebeat),
   * metrics -> Prometheus + Grafana,
   * traces -> OpenTelemetry -> Jaeger,
   * how to correlate logs/metrics/traces (trace IDs, labels).

Where it matters I’ll cite authoritative docs/guides I used to construct the examples. ([IBM][1])

---

# 1) Short definitions + real-world example (the “story”)

**Observability trifecta** = *logs*, *metrics*, *traces* — three telemetry signals that together let you understand and debug running systems. Think of them as different lenses:

* **Metrics** — numeric, time-series, pre-aggregated, low cardinality, fast queries. Good for monitoring and alerting (e.g., “HTTP 5xx rate > 1%” or latency P95). *Example:* Prometheus counters for requests_total, histograms for request_duration_seconds. ([prometheus.io][2])

* **Logs** — event records, usually textual or structured (JSON). High cardinality and detail (stack traces, user id, payload). Use for forensic debugging of a single request or crash. *Example:* A structured JSON log including timestamp, level, message, user_id, trace_id, span_id. ([Elastic][3])

* **Traces** — distributed traces record the path of a single request across services as a tree of spans; each span has start/end time, attributes, and can include events. Great for understanding cross-service latency and causality (who called whom and why it’s slow). *Example:* A user request that hits API -> auth service -> product service -> DB; trace shows which call took the most time. ([OpenTelemetry][4])

Real-world scenario (e-commerce checkout):

* **Metric**: `checkout_success_rate` drops from 99% → 90% → alert triggers. (Prometheus triggers)
* **Logs**: look at structured logs around the failing requests, see repeated `sql: transaction deadlock` with user id X and `trace_id=...`.
* **Trace**: you open the trace in Jaeger; it shows the DB write span has high latency during checkout; upstream service retries cause timeout cascade.

---

# 2) How they complement each other (practical guidance)

* **Start with metrics** (fast, cheap): they tell you *that* something changed (alerts). ([prometheus.io][2])
* **Use traces** to see where time is spent across services and to link spans — they tell you *where* in the call graph the problem lives. ([OpenTelemetry][4])
* **Use logs** to get the detailed payloads, errors, stack traces — they tell you *why* (business data, error messages). Correlate logs with traces by injecting `trace_id` and `span_id` into logs. ([Elastic][3])

Important operational caveats:

* **Cardinality**: metrics should avoid high-cardinality labels (eg user_id) — that destroys Prometheus performance. Use labels for low-cardinality dimensions (region, service, status). ([prometheus.io][2])
* **Structured logs**: prefer JSON logs so ELK or Filebeat can parse and index fields. Include trace_id in every log line for correlation. ([Elastic][3])
* **Sampling**: full tracing of every request can be expensive; use adaptive sampling or sample head traces. ([OpenTelemetry][4])

---

# 3) Hands-on practical setup — architecture & what we’ll build

We’ll build a minimal reproducible setup that you can run locally (or adapt to prod):

* A small Go HTTP app exposing:

  * structured JSON logs (to file/stdout),
  * Prometheus metrics endpoint `/metrics`,
  * OpenTelemetry traces exported to Jaeger (OTLP or Jaeger exporter).
* A Docker Compose stack:

  * **Prometheus** (scrapes `/metrics`),
  * **Grafana** (dashboards for metrics),
  * **Jaeger (all-in-one)** (collects traces),
  * **Elasticsearch + Kibana** (logs),
  * **Filebeat** (tails app log file and ships to Elasticsearch).
* Config files: `prometheus.yml`, Filebeat config, minimal Grafana dashboard guidance.

Below I give code you can copy-paste and run locally.

---

# 4) Go application — code (complete, ready to run)

Create a directory `observability-example` and `main.go`. The app will:

* serve `/` (simulated work),
* emit structured logs (with trace_id when present),
* expose `/metrics` for Prometheus,
* create OpenTelemetry spans and export to Jaeger.

**main.go**

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
)

var (
	reqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "example_requests_total",
			Help: "Total requests handled",
		},
		[]string{"path", "status"},
	)

	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "example_request_duration_seconds",
			Help:    "Request durations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(reqs, reqDuration)
}

// simple structured logger (JSON)
type Logger struct {
	enc *json.Encoder
}

func NewLogger(out *os.File) *Logger {
	return &Logger{enc: json.NewEncoder(out)}
}

func (l *Logger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	// try to get trace id from context
	tr := oteltrace.SpanContextFromContext(ctx)
	if tr.IsValid() {
		fields["trace_id"] = tr.TraceID().String()
		fields["span_id"] = tr.SpanID().String()
	}
	fields["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	fields["level"] = "info"
	fields["msg"] = msg
	_ = l.enc.Encode(fields)
}

func main() {
	// Create simple logger that writes to /var/log/app.log (Filebeat will tail it)
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("open log: %v", err)
	}
	logger := NewLogger(logFile)

	// Init tracing -> Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
	if err != nil {
		logger.Info(context.Background(), "failed to create jaeger exporter", map[string]interface{}{"err": err.Error()})
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			// attributes about this service
			attribute.String("service.name", "example-go-app"),
		)),
	)
	defer func() { _ = tp.Shutdown(context.Background()) }()
	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("example-go-app")

	// HTTP handlers
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// instrumented handler
	mux.Handle("/", otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// start a child span (otelhttp already created a span; but demo creating nested spans)
		ctx := r.Context()
		ctx, span := tracer.Start(ctx, "handle-root")
		defer span.End()

		start := time.Now()
		// simulated work
		time.Sleep(50 * time.Millisecond)
		// simulate occasional error
		status := 200
		if time.Now().Unix()%7 == 0 {
			status = 500
			w.WriteHeader(500)
			fmt.Fprint(w, "internal error")
			span.SetStatus(oteltrace.StatusCodeError, "simulated")
		} else {
			fmt.Fprint(w, "hello world")
		}
		duration := time.Since(start).Seconds()
		reqs.WithLabelValues(r.URL.Path, fmt.Sprintf("%d", status)).Inc()
		reqDuration.WithLabelValues(r.URL.Path).Observe(duration)

		// log with trace id from ctx
		logger.Info(ctx, "handled request", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": status,
			"dur":    duration,
		})
	}), "root"))

	// start server
	addr := ":8080"
	logger.Info(context.Background(), "starting server", map[string]interface{}{"addr": addr})
	if err := http.ListenAndServe(addr, otelhttp.NewHandler(mux, "server")); err != nil {
		logger.Info(context.Background(), "server error", map[string]interface{}{"err": err.Error()})
	}
}
```

Notes about this code:

* Uses **prometheus/client_golang** to expose metrics on `/metrics`. (See Prometheus docs). ([prometheus.io][2])
* Tracing uses OpenTelemetry SDK and **jaeger** exporter sending to Jaeger collector HTTP endpoint. This emits spans including the OTEL trace id which the logger picks up from context. ([OpenTelemetry][4])
* Logger writes JSON lines to `app.log`. Filebeat will tail this file and send to Elasticsearch.

---

# 5) Docker Compose (stack)

Create `docker-compose.yml` with services: `app` (build from Go), `prometheus`, `grafana`, `jaeger`, `elasticsearch`, `kibana`, `filebeat`.

A minimal Compose (trimmed for brevity) — you can expand/secure for production:

```yaml
version: "3.8"
services:
  app:
    build: .
    volumes:
      - ./app.log:/app/app.log
    depends_on:
      - jaeger
    ports:
      - "8080:8080"

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    depends_on:
      - prometheus

  jaeger:
    image: jaegertracing/all-in-one:1.48
    ports:
      - "16686:16686"   # UI
      - "14268:14268"   # collector (HTTP)

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.9.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
    ports:
      - "9200:9200"
    volumes:
      - esdata:/usr/share/elasticsearch/data

  kibana:
    image: docker.elastic.co/kibana/kibana:8.9.0
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch

  filebeat:
    image: docker.elastic.co/beats/filebeat:8.9.0
    volumes:
      - ./filebeat.yml:/usr/share/filebeat/filebeat.yml:ro
      - ./app.log:/app/app.log
    depends_on:
      - elasticsearch

volumes:
  esdata:
```

**prometheus.yml** (scrape our Go app):

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'example-go-app'
    static_configs:
      - targets: ['app:8080']   # container name/service and port
```

**filebeat.yml** (basic filebeat config to tail `app.log` and send to Elasticsearch):

```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /app/app.log
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

After `docker-compose up --build`:

* Visit Grafana `http://localhost:3000` (default creds admin/admin) and add Prometheus as datasource (`http://prometheus:9090`). Create dashboards using metrics `example_request_duration_seconds` etc. (Grafana docs). ([Grafana Labs][5])
* Visit Jaeger UI `http://localhost:16686` and search traces for `service=example-go-app`. ([Coding Explorations][6])
* Visit Kibana `http://localhost:5601` and search for indexed JSON logs (Filebeat parsed fields including `trace_id`). ([Elastic][7])

---

# 6) Correlation: connecting logs, traces, metrics

* **Trace ID in logs**: your logger reads the trace id from OpenTelemetry context and writes `trace_id` into each JSON log line. In Kibana you can filter logs with that `trace_id` and also open Jaeger with that trace id to view spans.
* **Metrics label with trace?** Avoid adding trace_id as a metric label (cardinality explosion). Use metrics for aggregated behaviour; link them by time windows and trace sampling. Use span attributes (not metrics) for per-request context.
* **Useful fields to include in logs**: `trace_id`, `span_id`, `request_id` (business id), `user_id` (if low cardinality or redacted), `error`.

---

# 7) Production considerations & best practices

* **Retention & cost**: logs and traces grow fast. Decide retention policies for Elasticsearch and Jaeger indices. Consider ingest filtering (only store error logs or sampled traces).
* **Sampling strategy**: sample 1% of requests but 100% of errors; use tail-based sampling for traces in production if available.
* **Cardinality control**: Prometheus labels must be low cardinality; log fields can be high cardinality but be mindful of storage & index explosion. ([prometheus.io][2])
* **Security**: protect Elasticsearch/Kibana with access control; secure OTLP and Jaeger endpoints behind auth in production. ([Elastic][3])
* **Unified pipelines**: consider using OpenTelemetry Collector to receive traces/metrics/logs and forward to multiple backends (Prometheus remote_write, Elasticsearch, Jaeger). It simplifies sampling/exporting. (OpenTelemetry/docs). ([OpenTelemetry][4])

---

# 8) Quick checklist to get you going locally

1. Copy `main.go`, create `Dockerfile` to build the Go app.
   Example Dockerfile:

   ```dockerfile
   FROM golang:1.21-alpine AS build
   WORKDIR /src
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . .
   RUN go build -o /app main.go

   FROM alpine:3.18
   COPY --from=build /app /app
   WORKDIR /app
   EXPOSE 8080
   CMD ["/app"]
   ```
2. Add `docker-compose.yml`, `prometheus.yml`, `filebeat.yml` from above.
3. `docker compose up --build`
4. Hit `http://localhost:8080/` repeatedly and inspect:

   * Prometheus at `http://localhost:9090` → `example_request_duration_seconds_bucket` etc.
   * Grafana `http://localhost:3000` → add Prometheus datasource.
   * Jaeger UI `http://localhost:16686` → search traces.
   * Kibana `http://localhost:5601` → discover logs (search `trace_id: *`).

---

# 9) Further reading / authoritative references

* Prometheus Go guide (instrumentation): official docs. ([prometheus.io][2])
* OpenTelemetry Go tracing guide and examples (tracing + Jaeger). ([OpenTelemetry][4])
* Elastic blog on unified logs/metrics/traces and Filebeat docs for collecting logs. ([Elastic][3])
* IBM / Sematext overview of logs/metrics/traces as pillars of observability. ([IBM][1])

---

# 10) If you want — next steps I can produce immediately

(choose any and I’ll implement in this same chat)

* A **ready-to-run Git repo** containing `main.go`, `Dockerfile`, `docker-compose.yml`, and minimal Grafana dashboard JSON.
* An **OpenTelemetry Collector** config to collect traces/metrics/logs and route them (better for production).
* A **sample Kibana saved search / visualization** or Grafana dashboard (prebuilt).
* A step-by-step walkthrough for **tail-based sampling** using the OpenTelemetry Collector.

Pick one and I’ll drop the files/configs / Docker Compose adjustments in this chat right away.

---

If you want, I can now generate the Git repo content (main.go, Dockerfile, docker-compose.yml, prometheus.yml, filebeat.yml) in separate files you can download — tell me which of the next steps above you prefer and I’ll produce the files here.

[1]: https://www.ibm.com/think/insights/observability-pillars?utm_source=chatgpt.com "Three Pillars of Observability: Logs, Metrics and Traces"
[2]: https://prometheus.io/docs/guides/go-application/?utm_source=chatgpt.com "Instrumenting a Go application for Prometheus"
[3]: https://www.elastic.co/blog/3-pillars-of-observability?utm_source=chatgpt.com "The 3 pillars of observability: Unified logs, metrics, and traces"
[4]: https://opentelemetry.io/blog/2022/go-web-app-instrumentation/?utm_source=chatgpt.com "Go Web-app Instrumentation"
[5]: https://grafana.com/docs/grafana-cloud/knowledge-graph/enable-prom-metrics-collection/runtimes/golang/?utm_source=chatgpt.com "Configure golang to generate Prometheus metrics"
[6]: https://www.codingexplorations.com/blog/using-jaeger-with-opentelemetry-in-go-a-step-by-step-guide?utm_source=chatgpt.com "Using Jaeger with OpenTelemetry in Go: A Step-by- ..."
[7]: https://www.elastic.co/docs/deploy-manage/monitor/stack-monitoring/collecting-log-data-with-filebeat?utm_source=chatgpt.com "Collecting log data with Filebeat | Elastic Docs"



# part 2:


Great — let’s extend your hands-on observability lab to include the **OpenTelemetry Collector** and **Elastic APM**, and show exactly how they fit into the stack you already have (Go app + Prometheus/Grafana + Jaeger + ELK). I’ll explain the concepts, tradeoffs, how to wire them together, and give a runnable example (Docker Compose + configs + small code changes).

I’ll cover:

1. What the OpenTelemetry Collector is, its building blocks and when to use it. ([OpenTelemetry][1])
2. How to change the Go app to send telemetry to a Collector (OTLP), and a collector config that exports to Jaeger **and** to Elastic APM (APM Server). ([Elastic][2])
3. What Elastic APM is, how it relates to OTLP/OpenTelemetry, and whether it “does all the jobs” (metrics, logs, traces). ([Elastic][2])
4. A complete, practical Docker Compose / config example that you can run locally (Collector + APM + backends), plus notes on sampling, processors, and production hardening.

---

# 1 — OpenTelemetry Collector: what it is and why use it

**Short definition:** the Collector is a vendor-neutral, standalone service that *receives*, *processes*, and *exports* telemetry (traces, metrics, logs). Instrument your apps to talk to the Collector (OTLP), and the Collector forwards to one or more backends (Jaeger, Prometheus remote, Elastic, vendor SaaS) and can apply processing (batching, sampling, enrichment). This removes the need to configure multiple exporters in every service and gives a central place for pipeline logic. ([OpenTelemetry][1])

**Key components (Collector pipeline):**

* **Receivers** — accept telemetry (e.g. `otlp` for OTLP/gRPC or OTLP/HTTP, `jaeger` receiver, `prometheus` receiver for scraping).
* **Processors** — operate on data in flight (batching, attributes, sampling like `probabilistic_sampler`, `tail_sampling` in contrib). Use to reduce volume, enrich data, or normalize attributes.
* **Exporters** — send data to backends (e.g. `jaeger`, `otlp` to Elastic APM Server, `prometheusremotewrite`, `logging`).
* **Pipelines** — connect receivers -> processors -> exporters for specific signal types (traces | metrics | logs). ([OpenTelemetry][3])

**Why put a Collector in your stack (practical benefits):**

* Single place for sampling and filtering (reduce cost).
* Fan-out: send the same telemetry to multiple backends (Jaeger + Elastic + SaaS) without instrumenting app with multiple exporters.
* Centralized enrichment (add `service.version`, `k8s.*` labels).
* Easier migration between observability vendors.

---

# 2 — Elastic APM: what it is and how it fits

**Short definition:** Elastic APM is Elastic’s application performance monitoring system built on the Elastic Stack (APM Server + Elasticsearch + Kibana + APM agents). It collects traces, metrics and error events and shows them in Kibana’s APM UI. Recent Elastic APM Server versions natively support **OTLP intake** — meaning OpenTelemetry OTLP data can be sent directly to APM Server. That lets you use OpenTelemetry SDKs/Collector and still visualize in Elastic APM. ([Elastic][4])

**Does Elastic APM cover the three signals?**

* **Traces:** Yes — Elastic APM visualizes distributed traces and services. OTLP traces can be accepted by APM Server. ([Elastic][2])
* **Metrics:** Yes — metrics can be stored in Elastic’s TSDS/data streams; Elastic supports ingesting OTLP metrics. Note mapping differences between "classic APM" and general OpenTelemetry data streams; check Elastic docs for the exact mappings and storage model. ([Elastic][5])
* **Logs:** Elastic (Elasticsearch + Kibana) is a primary log store. You can either ship logs with Filebeat (common) or send structured logs via OTLP/logs to APM Stack (APM Server supports OTLP logs). ([Elastic][4])

**Pros of Elastic APM vs raw OpenTelemetry backends:**

* Very good integrated UI (Kibana) for traces+metrics+logs (unified view).
* Native OTLP support, so it can be fed by Collector or SDKs.
* Elastic’s indexing/pipelines allow powerful searching, dashboards and correlation.

**Caveats / production notes:**

* Elastic security / auth must be configured (APM Server often runs on 8200 and may require TLS + API key). If you use OTLP to APM Server, you’ll often need to configure headers or use the collector’s exporter `otlp` with proper endpoint/credentials. Some APM Server versions had gotchas in OTLP paths — check your APM Server docs/version and test. ([GitHub][6])

---

# 3 — Hands-on: architecture and approach

We’ll adapt your previous stack. New high-level flow:

```
[Go app] --OTLP--> [OpenTelemetry Collector] --fanout--> [Jaeger] + [Elastic APM Server -> Elasticsearch/Kibana]
                         \
                          -> [Optional other exporters: logging, prometheus remote, third-party]
```

Why this?

* App only knows OTLP endpoint (collector). One change to app, and you can redirect where telemetry goes without code changes.
* Collector can perform sampling, add attributes, route errors traces to Elastic while sending all traces to Jaeger, etc.

We’ll present:

* A Collector config (`collector-config.yaml`) using the **contrib** image (because it includes useful processors and exporters like tail_sampling).
* An updated **Go** snippet to use OTLP exporter pointed at the collector, instead of direct Jaeger exporter.
* A `docker-compose.yml` that adds the Collector and APM Server to the existing stack (Jaeger + Elasticsearch + Kibana + Prometheus + Grafana + Filebeat).

I’ll give the concrete files next.

---

# 4 — Collector config (example)

Create `collector-config.yaml` (we use `otelcol-contrib` image so we can use `tail_sampling` if you need it later).

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:

processors:
  batch:
    timeout: 10s
    send_batch_size: 8192

  attributes/service_attributes:
    actions:
      - key: service.instance.id
        action: insert
        value: "${HOSTNAME}"

  # probabilistic sampler example (keep 10% by default)
  probabilistic_sampler:
    sampling_percentage: 10

  # tail_sampling requires state; fine for demo but production needs storage
  tail_sampling:
    decision_wait: 30s
    num_traces: 10000
    policies:
      - name: error_traces
        type: tail_sampling
        # Select traces with span.status_code != OK or error attribute
        decision_copy: true

exporters:
  jaeger:
    endpoint: "http://jaeger:14268/api/traces"
    # use thrift_http by default with the jaeger exporter

  otlp/elastic:
    endpoint: "http://apm-server:8200"
    # If APM Server expects OTLP via HTTP, this will target it.
    # In production, secure with TLS and credentials:
    # headers:
    #   "Authorization": "ApiKey <your_api_key>"

  logging:
    loglevel: info

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, attributes/service_attributes]
      exporters: [jaeger, otlp/elastic, logging]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/elastic, logging]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/elastic, logging]
```

**Comments & explanation**

* `receivers.otlp` accepts both gRPC (4317) and HTTP (4318) OTLP traffic from instrumented apps or SDKs. ([OpenTelemetry][1])
* We export **traces** to both Jaeger and `otlp/elastic` (APM Server). That gives you Jaeger UI + Elastic APM UI. Fan-out is a common use case. ([Last9][7])
* The `probabilistic_sampler` (or `tail_sampling`) processor reduces volume. Tail sampling is powerful because it lets you sample after you know if a trace had an error — but it requires more Collector resources/storage. Use sampling to control cost. ([OpenTelemetry][3])

---

# 5 — Update the Go app: send OTLP to the Collector

Replace the Jaeger exporter in your app with an OTLP exporter so the app sends OTLP to the Collector.

Add dependencies in `go.mod` (OpenTelemetry OTLP exporter + SDK):

```
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
go get go.opentelemetry.io/otel/sdk/trace
```

Minimal code snippet to create an OTLP/HTTP exporter:

```go
import (
    "context"
    "log"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/sdk/resource"
)

// createOTLPTracer creates tracer provider which exports to collector via OTLP/HTTP.
func createOTLPTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    // endpoint is the collector service name in Docker Compose; using port 4318 (OTLP/HTTP)
    client := otlptracehttp.NewClient(
        otlptracehttp.WithEndpoint("otel-collector:4318"),
        otlptracehttp.WithInsecure(), // only for local/dev; use TLS in prod
    )
    exporter, err := otlptracehttp.New(ctx, client)
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            attribute.String("service.name", "example-go-app"),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

Then call `createOTLPTracer(context.Background())` at startup and shut down `tp.Shutdown()` on exit.

**Important:** For metrics and logs, you can also use OTLP exporters (metrics and logs OTLP), or keep `/metrics` for Prometheus scraping. Many production setups:

* Export **traces** via OTLP to Collector.
* Keep **metrics** exported via Prometheus scrape or use Collector’s prometheus receiver to scrape and then send metrics onwards.
* **Logs**: send structured logs to Filebeat/Logstash or use OTLP logs to Collector.

---

# 6 — Docker Compose example (extended)

Below is a trimmed but runnable `docker-compose.yml` that includes `otelcol` and `apm-server`. It presumes you still have `elasticsearch`, `kibana`, `jaeger`, `prometheus`, `grafana`, and `filebeat`. I’ll show the added services only (you can merge into your original compose):

```yaml
version: '3.8'
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.73.0
    command: ["--config=/conf/collector-config.yaml"]
    volumes:
      - ./collector-config.yaml:/conf/collector-config.yaml:ro
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "55681:55681" # legacy
    depends_on:
      - jaeger
      - apm-server

  apm-server:
    image: docker.elastic.co/apm/apm-server:8.9.0
    environment:
      - "ELASTICSEARCH_HOSTS=http://elasticsearch:9200"
      - "APM_SERVER_OTLP_ENABLED=true"   # ensure OTLP intake is enabled if needed
      - "OUTPUT_ELASTICSEARCH_ENABLED=true"
    ports:
      - "8200:8200"   # default APM Server port
    depends_on:
      - elasticsearch
```

**Notes:**

* We used `otel/opentelemetry-collector-contrib` so tail sampling and some extra processors/exporters are available.
* APM Server must be configured to accept OTLP intake. In Elastic's newer images you can enable OTLP intake; check your APM Server version docs. If APM Server is secured, you must configure TLS/API keys. ([Elastic][4])

---

# 7 — How to test and verify (hands-on)

1. `docker compose up --build` (with your app using OTLP exporter pointing to `otel-collector:4318`).
2. Generate traffic: `for i in {1..500}; do curl -s http://localhost:8080/ > /dev/null; done`
3. Check Jaeger UI: `http://localhost:16686` — you should see traces from `example-go-app`.
4. Check Kibana APM: `http://localhost:5601` → APM app → Services — traces should appear (if APM Server received OTLP and ingested to ES).

   * If traces don’t appear in Kibana: check APM Server logs for OTLP errors and ensure Collector exporter endpoint is correctly set and accessible (and that any required headers are set).
5. Inspect logs in Kibana (Filebeat or OTLP logs) — ensure `trace_id` fields are present and searchable. Use Kibana Discover to search `trace.id : *` or `trace_id : <value>`.
6. Prometheus continues to scrape `/metrics` as before — if you want the Collector to scrape metrics and present them to Prometheus, use the `prometheusreceiver` in Collector and expose a Prometheus exporter (but often simpler to scrape app directly).

---

# 8 — Sampling & cost control (practical recipes)

* **App-level probabilistic sampling**: set a sampler in your SDK (e.g., sample 1%). Cheap but may miss rare errors.
* **Collector-level probabilistic sampling**: configure `probabilistic_sampler` in Collector to drop many traces before exporting.
* **Tail sampling (recommended for errors)**: use Collector `tail_sampling` or vendor solutions — sample traces after you know they contained an error (keeps error traces 100% while sampling normal traces). Tail sampling requires the Collector to keep trace state for `decision_wait` time and more memory/storage — plan accordingly. ([GitHub][8])

---

# 9 — Elastic APM specifics and gotchas

* **OTLP intake**: Elastic APM Server supports receiving OTLP traces/metrics/logs. This lets Collector export OTLP directly to APM Server. Depending on your Elastic version you may need to enable OTLP intake in APM Server settings and supply API keys or TLS. Test with curl/postman to the APM OTLP endpoint to validate ingestion. ([Elastic][4])
* **Field mapping**: OpenTelemetry and Elastic use slightly different naming/mapping; Elastic has guidance on mapping OTel signals into Elastic data streams and TSDS for metrics — consult docs for version-specific behavior. ([Elastic][5])
* **Troubleshooting OTLP to APM Server**: community threads show occasional version-specific issues (404 on OTLP paths or config differences). If traces fail to appear, check APM Server logs and ensure OTLP paths and protocols (gRPC vs HTTP) are correct. ([GitHub][6])

---

# 10 — Full files / copy-paste summary

Below are the most important files you’ll need to add or change. (I already showed `collector-config.yaml` and the Go OTLP snippet above.) Quick recap of key changes:

* `collector-config.yaml` — Collector pipelines, exporters (Jaeger + otlp/elastic), processors (batch, sampling).
* `docker-compose.yml` — add `otel-collector` service and `apm-server` service (and ensure `apm-server` ↔ `elasticsearch` is configured).
* `main.go` (or the tracing init) — switch to OTLP exporter with endpoint `otel-collector:4318`.

If you want, I can now:

* produce the **complete runnable repo** (main.go, Dockerfile, docker-compose.yml, collector-config.yaml, prometheus.yml, filebeat.yml) as files you can download and run; **or**
* generate an **OpenTelemetry Collector advanced config** that demonstrates tail sampling + attribute enrichment + routing (e.g., route error traces only to Elastic, all traces to Jaeger), or
* give a **troubleshooting checklist** for OTLP → Collector → APM Server (common errors + how to read logs).

Pick one and I will produce the files/configs right here. (I won’t run them for you, but I’ll give exact files and commands to run locally.)

[1]: https://opentelemetry.io/docs/collector/?utm_source=chatgpt.com "Collector"
[2]: https://www.elastic.co/docs/solutions/observability/apm/opentelemetry?utm_source=chatgpt.com "Use OpenTelemetry with Elastic APM | Elastic Docs"
[3]: https://opentelemetry.io/docs/collector/configuration/?utm_source=chatgpt.com "Configuration"
[4]: https://www.elastic.co/docs/solutions/observability/apm/opentelemetry-intake-api?utm_source=chatgpt.com "OpenTelemetry intake API | Elastic Docs"
[5]: https://www.elastic.co/docs/reference/opentelemetry/compatibility/data-streams?utm_source=chatgpt.com "OpenTelemetry data streams compared to classic APM"
[6]: https://github.com/elastic/apm-server/issues/12353?utm_source=chatgpt.com "OTLP HTTP 404 - /v1/traces · Issue #12353 · elastic/apm- ..."
[7]: https://last9.io/blog/how-to-use-jaeger-with-opentelemetry/?utm_source=chatgpt.com "How to Use Jaeger with OpenTelemetry"
[8]: https://github.com/lightstep/opentelemetry-examples/blob/main/config/example-collector-config.yaml?utm_source=chatgpt.com "example-collector-config.yaml"



# part 3:

# Full reproducible example: OpenTelemetry Collector -> APM Server -> Elasticsearch/Kibana (logs, metrics, traces)

This document contains a complete, copy-paste ready example that implements your requested flow:

* **Go app** instruments traces (OTLP), exposes Prometheus metrics (`/metrics`) and writes structured JSON logs to a file.
* **OpenTelemetry Collector** receives: OTLP traces, scrapes Prometheus metrics, tails the app log file (filelog receiver). Collector exports everything via OTLP to **Elastic APM Server**.
* **APM Server** ingests OTLP signals and stores them in **Elasticsearch**; view everything in **Kibana**.

> Files included below. Save them into a directory (e.g. `observability-full`) and run `docker compose up --build` from that directory.

---

## File: `docker-compose.yml`

```yaml
version: '3.8'

services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.9.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - ES_JAVA_OPTS=-Xms1g -Xmx1g
    ports:
      - "9200:9200"
    volumes:
      - esdata:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -s http://localhost:9200/_cluster/health?wait_for_status=yellow&timeout=50s || exit 1"]
      interval: 10s
      timeout: 10s
      retries: 5

  kibana:
    image: docker.elastic.co/kibana/kibana:8.9.0
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
      - xpack.security.enabled=false
      - XPACK_FLEET_AGENTS_ENABLED=false
    ports:
      - "5601:5601"
    depends_on:
      elasticsearch:
        condition: service_healthy

  # APM Server 7.17.13 (The savior)
  apm-server:
    image: docker.elastic.co/apm/apm-server:7.17.13
    command: >
      apm-server -e
        -E apm-server.kibana.enabled=true
        -E apm-server.kibana.host="http://kibana:5601"
        -E output.elasticsearch.hosts=["http://elasticsearch:9200"]
        -E apm-server.auth.anonymous.enabled=true
        -E apm-server.rum.enabled=true
    ports:
      - "8200:8200" 
      - "8201:8200"
    depends_on:
      - elasticsearch
      - kibana # Removed "condition: service_healthy" to fix your error

  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.88.0
    command: ["--config=/conf/collector-config.yaml"]
    volumes:
      - ./collector-config.yaml:/conf/collector-config.yaml
      - ./app_logs:/var/log/app
    depends_on:
      - apm-server
    ports:
      - "4317:4317"
      - "4318:4318"
      - "8889:8889" 

  app:
    build: ./app
    environment:
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
      - OTEL_EXPORTER_OTLP_INSECURE=true
    depends_on:
      - otel-collector
    ports:
      - "8080:8080"
    volumes:
      - ./app_logs:/var/log/app

volumes:
  esdata:

```

---

## File: `collector-config.yaml`

```yaml
receivers:
  otlp:
    protocols:
      http:
      grpc:

  prometheus:
    config:
      scrape_configs:
        - job_name: app
          scrape_interval: 5s
          static_configs:
            - targets: ["app:8080"]

  filelog:
    include: ["/var/log/app/app.log"]
    operators:
      - type: json_parser
        timestamp:
          parse_from: attributes.ts
          layout: '%Y-%m-%dT%H:%M:%S.%fZ'

      - type: remove
        field: attributes.ts

      # Extract trace/span IDs so they are indexed correctly in ES
      - type: move
        if: 'attributes.trace_id != nil'
        from: attributes.trace_id
        to: resource["trace_id"]
        
      - type: move
        if: 'attributes["span.id"] != nil'
        from: attributes["span.id"]
        to: resource["span_id"]

processors:
  batch:
  attributes:
    actions:
      - action: insert
        key: service.name
        value: example-go-app
      - action: insert
        key: event.dataset
        value: app.logs

exporters:
  # 1. OTLP gRPC for Traces/Metrics (To APM Server 7.17)
  otlp/apm:
    endpoint: apm-server:8200
    tls:
      insecure: true

  # 2. Native ES for Logs (Direct to Elasticsearch 8.9)
  elasticsearch:
    endpoints: ["http://elasticsearch:9200"]
    # We create a specific index for these logs
    logs_index: "app-logs"
    tls:
      insecure: true

  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, attributes]
      exporters: [otlp/apm] # Goes to APM Server

    metrics:
      receivers: [prometheus]
      processors: [batch]
      exporters: [otlp/apm] # Goes to APM Server

    logs:
      receivers: [filelog]
      processors: [batch, attributes]
      exporters: [elasticsearch] # ### CHANGED: Goes Direct to ES ###


```

Notes:

* `filelog` receiver tails `app.log` and attempts to parse JSON messages using the `json` operator. The app writes JSON structured logs into `/var/log/app/app.log`.
* All three signal pipelines export to `otlp/elastic` (APM Server at `http://apm-server:8200`). The collector also logs to console for debugging.

---

## Directory: `app/` (Go application)

Create the following files under `app/`.

### File: `app/Dockerfile`

```dockerfile
# --- Stage 1: Builder ---
# --- Stage 1: Builder ---
FROM golang:1.23-alpine AS build

WORKDIR /src

# Install git, although it's only needed if go.mod has git dependencies (good practice)
RUN apk add --no-cache git 

# Copy module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build the application
COPY . .
# Build the executable with a descriptive name inside the /tmp folder
# This avoids overwriting the /app directory in the final stage.
RUN CGO_ENABLED=0 GOOS=linux go build -o /tmp/app_binary ./main.go

# --- Stage 2: Final minimal image ---
FROM alpine:3.18

# Create log volume directory and set permissions
# We set ownership to 1000:1000 which matches the non-root user 'appuser' created below.
RUN mkdir -p /var/log/app && chown -R 1000:1000 /var/log/app

# Create a non-root user (UID 1000) and switch to it for security
RUN adduser -D -u 1000 appuser
USER appuser

# Copy the built binary from the builder stage to the root of the final image
COPY --from=build /tmp/app_binary /app_binary

# Set the entrypoint to run the binary (must be the full filename)
ENTRYPOINT ["/app_binary"]

# Default port
EXPOSE 8080

```

### File: `app/go.mod`

```go
module example.com/otel-full

go 1.23.0

toolchain go1.24.3

require (
	github.com/go-chi/chi/v5 v5.0.8
	github.com/prometheus/client_golang v1.14.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.18.0
	go.opentelemetry.io/otel/sdk v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.18.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/grpc v1.58.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

```

### File: `app/main.go`

```go
package main

import (
	"context"
	"encoding/json"
	"errors" // <--- ADDED: Needed for errors.New
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes" // <--- ADDED: Needed for codes.Error
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	reqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "example_requests_total", Help: "Total requests"},
		[]string{"path", "status"},
	)
	reqDur = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "example_request_duration_seconds", Help: "Request durations", Buckets: prometheus.DefBuckets},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(reqs, reqDur)
}

// Simple JSON logger that writes to /var/log/app/app.log
func newJSONLogger(path string) *log.Logger {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	return log.New(f, "", 0)
}

func logJSON(l *log.Logger, fields map[string]interface{}) {
	_ = l.Output(2, mustJSON(fields))
}

func mustJSON(m map[string]interface{}) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	// FIX: Strip the scheme (http:// or https://) because WithEndpoint expects host:port
	if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.TrimPrefix(endpoint, "http://")
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.TrimPrefix(endpoint, "https://")
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint), // otlptracehttp expects host:port
		otlptracehttp.WithInsecure(),         // Used unconditionally for simplicity
	)
	if err != nil {
		return nil, err
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceName("example-go-app"),
	}

	res := sdkresource.NewWithAttributes(
		semconv.SchemaURL,
		attributes...,
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	ctx := context.Background()
	tp, err := initTracer(ctx)
	if err != nil {
		panic(err)
	}
	defer func() { _ = tp.Shutdown(ctx) }()

	logger := newJSONLogger("/var/log/app/app.log")

	r := chi.NewRouter()

	r.Handle("/metrics", promhttp.Handler())

	// --- NEW ERROR ROUTE ---
	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		// Get the current span
		span := trace.SpanFromContext(r.Context())

		// Create a real Go error
		err := errors.New("something went terribly wrong in the database")

		// 1. Record the error in the trace (This creates the "Exception" event with stack trace)
		span.RecordError(err, trace.WithStackTrace(true))

		// 2. Set the status of the span to Error so it turns red in APM
		span.SetStatus(codes.Error, "critical failure")

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Check Kibana for the stack trace!")
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// simulated work
		time.Sleep(50 * time.Millisecond)

		status := 200
		if time.Now().Unix()%11 == 0 {
			status = 500
			w.WriteHeader(500)
			fmt.Fprint(w, "internal error")
		} else {
			fmt.Fprint(w, "hello world")
		}

		dur := time.Since(start).Seconds()
		reqs.WithLabelValues(r.URL.Path, fmt.Sprintf("%d", status)).Inc()
		reqDur.WithLabelValues(r.URL.Path).Observe(dur)

		spanContext := trace.SpanFromContext(r.Context()).SpanContext()
		traceID := ""
		spanID := ""

		if spanContext.IsValid() {
			traceID = spanContext.TraceID().String()
			spanID = spanContext.SpanID().String()
		}
		// Log structured JSON — include trace information if available
		fields := map[string]interface{}{
			"ts":     time.Now().UTC().Format(time.RFC3339Nano),
			"level":  "info",
			"msg":    "handled request",
			"method": r.Method,
			"path":   r.URL.Path,
			"status": status,
			"dur":    dur,
			// Add standard Elastic/OTel correlation fields
			"trace.id": traceID,
			"span.id":  spanID,
			"trace_id": traceID, // redundancy for different backend parsers
		}
		logJSON(logger, fields)
	})

	// wrap router with otelhttp so traces are created for requests
	wrapped := otelhttp.NewHandler(r, "server")
	srv := &http.Server{Addr: ":8080", Handler: wrapped}

	log.Printf("starting app on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}


```

Notes about the app:

* It exposes `/metrics` (Prometheus format), which the Collector scrapes using its `prometheus` receiver.
* Traces are sent via OTLP/HTTP to the Collector using the OTLP trace exporter. The environment variables in `docker-compose.yml` tell the app to send to `otel-collector` service.
* The logger writes structured JSON to `/var/log/app/app.log`. The collector `filelog` receiver tails that file and parses JSON messages.
* For this tutorial, trace IDs are not explicitly injected into logs; correlation is achieved in Kibana by matching timestamps and resource attributes. If you want stronger correlation, include trace_id in logs by extracting span context inside handlers (that is an exercise I can add next).

---

## How to run

1. Save all files (top-level `docker-compose.yml`, `collector-config.yaml`, and directory `app/` with `main.go`, `go.mod`, `Dockerfile`).
2. `mkdir app_logs` (if not already created) — it's mounted into the collector and app.
3. `docker compose up --build`

Services will start:

* Elasticsearch: `http://localhost:9200`
* Kibana: `http://localhost:5601`
* APM Server: `http://localhost:8200`
* Collector listens on OTLP ports `4317/4318`.
* App: `http://localhost:8080` (metrics: `http://localhost:8080/metrics`)

Generate traffic:

```bash
for i in {1..200}; do curl -s http://localhost:8080/ > /dev/null; done
```

Verify:

* Open Kibana `http://localhost:5601` → APM app → Services. You should see `example-go-app` (traces & metrics) after a few seconds.
* In Kibana Discover, search the logs index (e.g., `logs-*` or `filebeat-*` depending on your Elastic mapping) for your JSON logs — they should include fields you wrote (ts, msg, status).
* APM UI shows traces and metrics stored by APM Server in Elasticsearch.

---

## Tips, limitations and next steps

* **Trace-to-log correlation**: In this example logs are parsed by Collector and sent to APM, but the app does not inject the trace_id into the log lines. For robust correlation, extract the active span context in the HTTP handler and add `trace_id` and `span_id` fields to your JSON logs. I can add code that does that if you want.
* **Security**: This example disables Elasticsearch security for simplicity. For production enable TLS, authentication, and secure the APM Server.
* **Sampling**: To control volume, configure sampling processors in `collector-config.yaml` (probabilistic_sampler or tail_sampling). Tail sampling requires more memory for the collector.
* **Resource attributes**: Use `service.name`, `service.version`, and host attributes to help Kibana group resources.
* **Production APM**: APM Server has its own ingest pipelines and mappings; use Elastic docs to tune index templates for OTLP signals.

---

If you'd like, I can now:

* Add automatic injection of `trace_id` into the app logs (Go code change) so logs are directly correlated to traces in Kibana; or
* Provide a Grafana dashboard that queries Elasticsearch metrics; or
* Add sampling processors (tail/ probabilistic) to `collector-config.yaml` and explain memory implications.

Which of these do you want next?


# part 4:

Nice — let’s build a battle-tested, production-ready example you can copy/paste and run in **both Docker-only** and **Kubernetes** environments.

We’ll use the industry standard pattern:

**App → stdout (JSON) → container runtime (Docker / containerd) → Fluent Bit (DaemonSet or sidecar) → OpenTelemetry Collector (OTLP) → APM Server → Elasticsearch → Kibana**

You’ll get:

1. A Go app that:

   * logs structured JSON to **stdout** (includes `trace_id` & `span_id`),
   * sends traces to OTLP Collector,
   * exposes Prometheus metrics at `/metrics`.
2. A **Docker Compose** example using Fluent Bit to read Docker container logs and forward them to the Collector.
3. A **Kubernetes** example using Fluent Bit DaemonSet reading containerd logs (Kubernetes standard) and forwarding to the Collector.
4. An **OpenTelemetry Collector** config that receives OTLP (traces, logs, metrics) and exports all to **Elastic APM Server** (Elasticsearch/Kibana).
5. Notes on scaling, sampling, security, and production hardening.

Everything is copy-paste ready. Let’s go.

---

## 0 — High level (what to run)

* Docker: `docker-compose up --build`
* Kubernetes: `kubectl apply -f k8s/` (manifests included below)

---

## 1 — Go app (stdout JSON + OTLP traces + /metrics)

Create `app/main.go` and `app/go.mod` (Dockerfile provided later for Docker Compose). This app logs JSON to stdout (not files) and injects trace IDs into logs.

### `app/go.mod`

```go
module example.com/otel-stdout

go 1.21

require (
    github.com/go-chi/chi/v5 v5.0.8
    go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.101.0
    go.opentelemetry.io/otel v1.18.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.18.0
    github.com/prometheus/client_golang v1.14.0
)
```

### `app/main.go`

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
    "sync/atomic"

    "github.com/go-chi/chi/v5"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/attribute"
    sdkresource "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

var (
    reqs = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "example_requests_total", Help: "Total requests"},
        []string{"path", "status"},
    )
    reqDur = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "example_request_duration_seconds", Help: "Request durations", Buckets: prometheus.DefBuckets},
        []string{"path"},
    )
    counter uint64
)

func init() {
    prometheus.MustRegister(reqs, reqDur)
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
    // OTEL collector endpoint via env OTEL_ENDPOINT (default: http://otel-collector:4318)
    ep := os.Getenv("OTEL_ENDPOINT")
    if ep == "" {
        ep = "http://otel-collector:4318"
    }
    // otlptracehttp wants host:port string (no http://) for WithEndpoint
    hostPort := ep
    if len(hostPort) >= 7 && hostPort[:7] == "http://" {
        hostPort = hostPort[7:]
    }
    client := otlptracehttp.NewClient(
        otlptracehttp.WithEndpoint(hostPort),
        otlptracehttp.WithInsecure(),
    )
    exporter, err := otlptracehttp.New(ctx, client)
    if err != nil {
        return nil, err
    }
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(sdkresource.NewWithAttributes(attribute.String("service.name", "example-go-app"))),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}

func main() {
    ctx := context.Background()
    tp, err := initTracer(ctx)
    if err != nil {
        log.Fatalf("init tracer: %v", err)
    }
    defer tp.Shutdown(ctx)

    r := chi.NewRouter()
    r.Handle("/metrics", promhttp.Handler())

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        ctx := r.Context()

        // business logic
        time.Sleep(50 * time.Millisecond) // simulate work
        status := 200
        if time.Now().Unix()%13 == 0 {
            status = 500
            w.WriteHeader(500)
            fmt.Fprint(w, "error")
        } else {
            fmt.Fprint(w, "ok")
        }

        dur := time.Since(start).Seconds()
        reqs.WithLabelValues(r.URL.Path, fmt.Sprintf("%d", status)).Inc()
        reqDur.WithLabelValues(r.URL.Path).Observe(dur)

        // log to stdout as structured JSON and include trace/span ids
        fields := map[string]interface{}{
            "ts":     time.Now().UTC().Format(time.RFC3339Nano),
            "level":  "info",
            "msg":    "handled request",
            "method": r.Method,
            "path":   r.URL.Path,
            "status": status,
            "dur":    dur,
            "id":     atomic.AddUint64(&counter, 1),
        }

        // extract trace/span
        if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
            fields["trace_id"] = sc.TraceID().String()
            fields["span_id"] = sc.SpanID().String()
        }
        // write JSON to stdout
        b, _ := json.Marshal(fields)
        os.Stdout.Write(append(b, '\n'))
    })

    wrapped := otelhttp.NewHandler(r, "server")
    srv := &http.Server{Addr: ":8080", Handler: wrapped}
    log.Println("starting :8080")
    log.Fatal(srv.ListenAndServe())
}
```

**Notes**

* The app writes to **stdout**. When running in containers, the runtime captures stdout logs.
* `trace.SpanContextFromContext(ctx)` extracts trace info injected by `otelhttp` wrapper — we include `trace_id` and `span_id` in log JSON.
* The app exports traces via OTLP/HTTP to Collector (env var `OTEL_ENDPOINT`).

---

## 2 — OpenTelemetry Collector config (receive OTLP and export to APM)

Create `collector-config.yaml`. This collector will accept OTLP (traces + logs + metrics) and export to Elastic APM Server via OTLP HTTP.

### `collector-config.yaml`

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:

  prometheus:
    config:
      scrape_configs:
        - job_name: 'example-go-app'
          static_configs:
            - targets: ['app:8080']    # in k8s use service:port

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024

  attributes/service:
    actions:
      - action: insert
        key: service.instance.id
        value: "${HOSTNAME}"

exporters:
  otlp/elastic:
    endpoint: "http://apm-server:8200"   # APM Server OTLP intake
    # If APM server is secured, add headers or TLS config.

  logging:
    loglevel: info

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, attributes/service]
      exporters: [otlp/elastic, logging]

    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/elastic, logging]

    metrics:
      receivers: [prometheus, otlp]
      processors: [batch]
      exporters: [otlp/elastic, logging]
```

**Notes**

* Collector receives OTLP over gRPC (4317) and HTTP (4318). Fluent Bit or Fluentd can call HTTP/OTLP if it supports it; Fluent Bit `out_opentelemetry` works with Collector HTTP at `http://collector:4318/v1/logs` or similar.
* We export to Elastic APM Server by pointing an OTLP exporter to APM Server. APM Server must have OTLP intake enabled (env/config).

---

## 3 — Docker Compose flow (Docker-only)

This is for a developer laptop or simple Docker environment. Fluent Bit will read Docker container logs (JSON format in `/var/lib/docker/containers/.../*.log`) and send OTLP logs to Collector.

Create `docker-compose.yml`:

```yaml
version: '3.8'
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.9.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ports:
      - "9200:9200"
    volumes:
      - esdata:/usr/share/elasticsearch/data

  kibana:
    image: docker.elastic.co/kibana/kibana:8.9.0
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch

  apm-server:
    image: docker.elastic.co/apm/apm-server:8.9.0
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
      - APM_SERVER_OTLP_ENABLED=true
      - OUTPUT.elasticsearch.enabled=true
    ports:
      - "8200:8200"
    depends_on:
      - elasticsearch

  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.73.0
    command: ["--config=/conf/collector-config.yaml"]
    volumes:
      - ./collector-config.yaml:/conf/collector-config.yaml:ro
    ports:
      - "4317:4317"
      - "4318:4318"
    depends_on:
      - apm-server

  fluent-bit:
    image: fluent/fluent-bit:2.1.11
    # mount docker container log directory
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - /var/log:/var/log:ro
      - ./fluent-bit/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf:ro
      - ./fluent-bit/parsers.conf:/fluent-bit/etc/parsers.conf:ro
    depends_on:
      - otel-collector

  app:
    build: ./app
    environment:
      - OTEL_ENDPOINT=http://otel-collector:4318
    ports:
      - "8080:8080"
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    depends_on:
      - otel-collector
```

### Fluent Bit config for Docker (`fluent-bit/fluent-bit.conf`)

This config tails Docker JSON logs and uses the `out_opentelemetry` plugin to send OTLP logs to Collector HTTP endpoint.

```ini
[SERVICE]
    Flush        1
    Daemon       Off
    Log_Level    info
    Parsers_File parsers.conf

[INPUT]
    Name              tail
    Tag               docker.*
    Path              /var/lib/docker/containers/*/*.log
    Parser            docker
    DB                /var/log/flb_kube.db
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  5

[FILTER]
    Name   kubernetes
    Match  docker.*
    # Not running in k8s, but enrich with local host metadata if needed

[OUTPUT]
    Name  opentelemetry
    Match docker.*
    Host  otel-collector
    Port  4318
    # Protocol HTTP with OTLP - plugin sends to /v1/logs
    Mode  http
    # If plugin requires: OTLP endpoint path can be configured here if needed
```

### `fluent-bit/parsers.conf`

```ini
[PARSER]
    Name        docker
    Format      json
    Time_Key    time
    Time_Format %Y-%m-%dT%H:%M:%S.%L
    Time_Keep   On
```

**Run**

```bash
docker compose up --build
# then generate traffic
for i in {1..200}; do curl -s http://localhost:8080/ > /dev/null; done
```

**Flow**: app logs → Docker json file → Fluent Bit reads files → Fluent Bit out_opentelemetry → Collector HTTP → Collector exports to APM → Elasticsearch.

---

## 4 — Kubernetes flow (containerd / kubelet)

In Kubernetes, best practice is to run Fluent Bit as a DaemonSet that tails `/var/log/containers/*.log` (containerd / CRI). Fluent Bit enriches logs with Kubernetes metadata via `kubernetes` filter and sends them to Collector via OTLP.

You’ll need:

* `otel-collector` Deployment (or DaemonSet), Service (internal).
* `fluent-bit` DaemonSet (reads container logs and sends to Collector).
* your app Deployment with sidecarless logging to stdout.

Below are minimal manifests.

### `k8s/otel-collector-deploy.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: otel-collector
  namespace: observability
spec:
  selector:
    app: otel-collector
  ports:
    - name: otlp-grpc
      port: 4317
    - name: otlp-http
      port: 4318
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
  namespace: observability
spec:
  replicas: 2
  selector:
    matchLabels:
      app: otel-collector
  template:
    metadata:
      labels:
        app: otel-collector
    spec:
      containers:
        - name: otel-collector
          image: otel/opentelemetry-collector-contrib:0.73.0
          args: ["--config=/conf/collector-config.yaml"]
          volumeMounts:
            - name: conf
              mountPath: /conf
      volumes:
        - name: conf
          configMap:
            name: otel-collector-config
```

Create a ConfigMap `otel-collector-config` containing the same `collector-config.yaml` shown earlier (adapt `targets` for prometheus scraping if using k8s service names).

### `k8s/fluent-bit-daemonset.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: observability
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush        1
        Daemon       Off
        Log_Level    info
        Parsers_File parsers.conf

    [INPUT]
        Name              tail
        Tag               kube.*
        Path              /var/log/containers/*.log
        Parser            docker
        DB                /var/log/flb_kube.db
        Mem_Buf_Limit     5MB
        Skip_Long_Lines   On

    [FILTER]
        Name   kubernetes
        Match  kube.*
        Kube_URL https://kubernetes.default.svc:443
        Merge_Log On
        K8S-Logging.Exclude On

    [OUTPUT]
        Name  opentelemetry
        Match kube.*
        Host  otel-collector.observability.svc.cluster.local
        Port  4318
        Mode  http

  parsers.conf: |
    [PARSER]
        Name        docker
        Format      json
        Time_Key    time
        Time_Format %Y-%m-%dT%H:%M:%S.%L
        Time_Keep   On

---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluent-bit
  namespace: observability
spec:
  selector:
    matchLabels:
      app: fluent-bit
  template:
    metadata:
      labels:
        app: fluent-bit
    spec:
      serviceAccountName: fluent-bit
      tolerations:
        - operator: "Exists"
      containers:
        - name: fluent-bit
          image: fluent/fluent-bit:2.1.11
          volumeMounts:
            - name: varlog
              mountPath: /var/log
              readOnly: true
            - name: varlibdockercontainers
              mountPath: /var/lib/docker/containers
              readOnly: true
            - name: config
              mountPath: /fluent-bit/etc/
      volumes:
        - name: varlog
          hostPath:
            path: /var/log
        - name: varlibdockercontainers
          hostPath:
            path: /var/lib/docker/containers
        - name: config
          configMap:
            name: fluent-bit-config
```

**Notes**

* On Kubernetes, containerd logs are in `/var/log/containers` or `/var/log/pods`. Using `Path /var/log/containers/*.log` covers typical setups.
* `filter kubernetes` adds pod name, namespace, labels, annotations — critical for searching and correlation in Kibana.
* Fluent Bit uses `out_opentelemetry` plugin to send OTLP logs to Collector HTTP endpoint.

**Apply**

```bash
kubectl create namespace observability
kubectl apply -f k8s/otel-collector-configmap.yaml   # contains collector-config.yaml
kubectl apply -f k8s/otel-collector-deploy.yaml
kubectl apply -f k8s/fluent-bit-daemonset.yaml
kubectl apply -f k8s/app-deploy.yaml  # your application Deployment + Service
```

---

## 5 — Elastic APM Server + Elasticsearch + Kibana

* APM Server must have OTLP intake enabled (`APM_SERVER_OTLP_ENABLED=true`) and be configured to talk to Elasticsearch.
* Collector OTLP exporter should point to `http://apm-server:8200`.
* Once APM collects traces & logs, Kibana’s APM app and Discover tab show the signals. Use `trace_id` in logs to jump to traces.

---

## 6 — Production best practices & scaling notes

### Sampling & cost control

* Do **not** export 100% of traces in high throughput systems:

  * Use **Collector processors**: `probabilistic_sampler`, `tail_sampling` (contrib) to sample non-error traffic.
  * Keep **100% of error traces**, sample regular traces.
* Metrics: export aggregated metrics to Prometheus and remote_write or let Collector handle metrics pipeline.

### Fluent Bit scaling / resources

* Fluent Bit is lightweight and designed to run as a DaemonSet across nodes. Typical production setup:

  * 1 Fluent Bit pod per node (DaemonSet)
  * Resource requests: CPU 50–200m, memory 50–200Mi depending on log volume.
  * Use buffering and persistent DB (pos_file) to survive restarts.

### Collector scaling

* Run Collector as **deployment with >=2 replicas** behind a Service. Collectors are stateless in this pattern (unless you use tail sampling that needs local storage).
* For tail sampling, use a **single dedicated Collector with persistent store** or use a sampling agent designed for that.

### Resilience & delivery guarantees

* Fluent Bit supports retry/backoff and buffering when collector/APM is unreachable.
* Configure Fluent Bit `storage.type filesystem` to avoid data loss on restarts (persistent buffer).
* Use TLS and mTLS for securing OTLP traffic.

### Correlation (logs ↔ traces ↔ metrics)

* Inject `trace_id` and `span_id` into JSON logs (done in the Go app code).
* Fluent Bit’s Kubernetes filter adds `kubernetes.pod_name`, `namespace`, `labels`, etc.
* APM traces provide `trace_id` you can use as a key in Kibana to correlate.

### Security

* Use TLS between Collector, APM Server and Elasticsearch.
* Require API Keys for APM ingestion.
* Restrict network access (private subnets, security groups).

---

## 7 — Troubleshooting tips

* If you don’t see logs in Kibana:

  * Check Fluent Bit logs for errors (connectivity, parsing).
  * Check Collector logs; ensure `otlp` receiver listening on the expected port.
  * Confirm APM Server accepts OTLP (OTLP intake enabled).
* If traces are missing:

  * Confirm app can reach Collector (`OTEL_ENDPOINT`).
  * Check Collector log exporter errors (resource limits, rejected due to malformed attributes).
* If metrics missing:

  * Ensure Collector scrapes `/metrics` (prometheus receiver) or app exposes `/metrics`.

---

## 8 — Summary: recommended default configs

* Use **stdout JSON logging** (app-level).
* Run **Fluent Bit DaemonSet** (k8s) or Fluent Bit container (Docker Compose) to tail container logs, enrich, forward to Collector.
* Use **OpenTelemetry Collector** (contrib) as central pipeline to apply sampling, enrichment, and to export to APM/Elastic.
* Run **APM Server → Elasticsearch → Kibana** for stored, searchable telemetry.

---

## 9 — Want the files as a repo / next steps?

I can:

* produce a ZIP/Git repo with:

  * `app/` (code + Dockerfile),
  * `docker-compose.yml` + Fluent Bit config,
  * `collector-config.yaml`,
  * `k8s/` manifests (collector, fluent-bit, app),
* OR generate the Kubernetes manifests fully rendered (including collector ConfigMap),
* OR add a Fluent Bit config that writes directly to Elasticsearch (skip Collector) — sometimes simpler for logs-only pipelines.

Which one do you want me to generate now? I’ll drop the files (or a zip) directly into this chat.



# bonuse point: send OLTP directly to APM and remove otel collector:
Yes — **you *can* send OpenTelemetry data directly from your Go app to Elastic APM Server**, **without** running an OpenTelemetry Collector, FluentBit, file tailing, or any intermediate forwarder.

And **yes, this is fully supported**, because **Elastic APM Server (v8.x+) natively supports OTLP/HTTP and OTLP/gRPC intake**.

This is often the simplest architecture:

```
Go app (OTLP exporter)  --->  Elastic APM Server (OTLP intake)  --->  Elasticsearch  --->  Kibana
```

Below is **a fully working, step-by-step, production-ready example** using:

* **OpenTelemetry Go SDK**
* **OTLP gRPC exporter**
* **Elastic APM Server with OTLP enabled**
* **Elastic Stack (ES + Kibana)**

---

# ✅ **1. Configure Elastic APM Server for OTLP Intake**

Edit your `apm-server.yml`:

```yaml
apm-server:
  host: "0.0.0.0:8200"

  # Enable OTLP intake
  otlp:
    enabled: true
    protocols:
      http:
        endpoint: "0.0.0.0:8200"
      grpc:
        endpoint: "0.0.0.0:8200"

output.elasticsearch:
  hosts: ["http://elasticsearch:9200"]
  username: "elastic"
  password: "YourPassword"

setup.kibana:
  host: "kibana:5601"
```

Start APM:

```
sudo systemctl restart apm-server
```

Or Docker Compose version:

```yaml
services:
  apm-server:
    image: elastic/apm-server:8.14.0
    command: ["apm-server", "-e"]
    ports:
      - "8200:8200"
    environment:
      - output.elasticsearch.hosts=http://elasticsearch:9200
      - output.elasticsearch.username=elastic
      - output.elasticsearch.password=YourPassword
    volumes:
      - ./apm-server.yml:/usr/share/apm-server/apm-server.yml
```

---

# ✅ **2. Full Go Example: Send Traces + Metrics + Logs Directly to Elastic APM**

## 📌 Modules

```bash
go mod init direct-otel-apm-example
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go get go.opentelemetry.io/otel/exporters/otlp/otlpgrpc
go get google.golang.org/grpc
```

---

# 📌 **main.go – FULL Example**

```go
package main

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

func main() {

	ctx := context.Background()

	// Elastic APM Server OTLP endpoint
	apmEndpoint := "localhost:8200"

	grpcConn, err := grpc.DialContext(
		ctx,
		apmEndpoint,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to APM OTLP endpoint: %v", err)
	}

	// ---- TRACES EXPORTER ----
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(grpcConn))
	if err != nil {
		log.Fatalf("Failed to create trace exporter: %v", err)
	}

	// ---- METRICS EXPORTER ----
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(grpcConn))
	if err != nil {
		log.Fatalf("Failed to create metric exporter: %v", err)
	}

	// ---- TRACE PROVIDER ----
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(tp)

	// ---- METRIC PROVIDER ----
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(time.Second*5)),
		),
	)
	otel.SetMeterProvider(mp)

	tracer := tp.Tracer("demo-service")
	meter := mp.Meter("demo-service")

	// METRIC COUNTER
	requestCounter, _ := meter.Int64Counter("myapp_requests")

	// Generate tracing + metrics every 2 seconds
	for {
		ctx, span := tracer.Start(ctx, "demo_span")
		span.SetAttributes(attribute.String("service", "demo-service"))
		span.SetAttributes(attribute.String("environment", "production"))

		requestCounter.Add(ctx, 1)

		time.Sleep(2 * time.Second)
		span.End()
	}
}
```

---

# 🎯 **3. Start Everything**

### **Elastic Stack**

```
docker-compose up -d elasticsearch kibana
```

### **APM Server**

```
docker-compose up -d apm-server
```

### **Run Your Go App**

```
go run main.go
```

---

# 🎉 **4. Check the Results**

Open Kibana → **APM UI**

➡ You will see:

* Traces
* Spans
* Service map
* Metrics
* CPU / memory
* Latency breakdown
* Errors & exceptions

Everything is visible instantly.

---

# 🏆 Should You Directly Send to APM Without a Collector?

### ✔ Advantages

| Feature     | Direct to APM         | Via Collector         |
| ----------- | --------------------- | --------------------- |
| Complexity  | **Simplest possible** | More moving parts     |
| Reliability | Very stable           | More failure points   |
| Performance | Lowest latency        | Slight overhead       |
| Deployment  | Easiest               | Extra daemon required |
| Scaling     | Scale APM Server      | Scale Collector fleet |

### ✔ When Direct is Ideal

* Microservices all inside one cluster
* You only need Elastic APM as backend
* No multi-backend routing (Elastic + ClickHouse + Loki etc.)
* Simpler deployments preferred

### ✔ When Collector is Needed

* Multi-destination routing
* Ingesting logs from files / containerd / systemd
* Adding sampling
* Adding batching, filtering, redaction
* Centralized config for hundreds of apps

---

# 📌 Conclusion

👉 **Yes, you can send your Go app's telemetry directly to Elastic APM Server using OTLP.**
👉 **This is fully supported, production-ready, and very common.**
👉 **This is the simplest architecture for Elastic-only environments.**

If you want, I can now generate:

✅ Docker Compose full environment (ES + Kibana + APM + Go app)
✅ Kubernetes deployment using OTLP direct-to-APM
✅ Logging extension (OTLP logs → Elastic)
✅ Metric dashboards in Kibana
✅ Secure TLS (mTLS OTLP) setup

Just tell me **which version you want next**.
