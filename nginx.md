# Comprehensive Guide to Nginx: From Basics to Advanced Topics

> **Purpose:** A full, production-ready guide for DevOps teams to design, deploy, operate, and troubleshoot Nginx at scale. This document is modular so you can pick sections for training, runbooks, or automation.

---

## Table of contents

1. Introduction to Nginx
2. Nginx architecture and internals
3. Installing Nginx — quick start and package choices
4. Request forwarding and proxying
5. URL rewriting and redirects
6. Request mirroring
7. WebSocket support
8. Conditionals and variables
9. Log formats and variables (observability)
10. SSL/TLS configuration and security hardening
11. HTTP/2 and QUIC (HTTP/3)
12. Load balancing and high availability
13. Caching strategies and CDN integration
14. Security best practices and access control
15. Performance optimization and tuning
16. Docker Compose and containerized Nginx
17. Kubernetes deployment patterns (Ingress / DaemonSet / Sidecar)
18. Production-ready config example (modular)
19. Best-practice templates and folder structure
20. Troubleshooting, debugging, and runbook
21. Appendix: Useful commands, references, and checklist

---

## 1. Introduction to Nginx

**Description:** Short background, where Nginx fits in a modern stack, and when to choose it.

* **What is Nginx?** A high-performance HTTP server, reverse proxy, and load balancer. Also commonly used as a TLS terminator, cache, and API gateway component.
* **When to use Nginx:** Static content, TLS termination, reverse proxying to microservices, edge caching, HTTP/2, WebSockets, and high-concurrency scenarios.
* **Advantages:** asynchronous event-driven architecture → handles many concurrent connections with low memory; mature ecosystem; many modules (official and third-party); stable production footprints.

---

## 2. Nginx architecture and internals

**Description:** Deep-dive into how Nginx works so operators understand performance and failure modes.

* **Master process:** reads config, manages worker processes, reloads gracefully (signal-based), controls binary upgrades with `nginx -s reload` and `nginx -s reopen`.
* **Worker processes:** run event loops and handle actual connections. Usually one worker per CPU core.
* **Event model:** uses `epoll` (Linux), `kqueue` (BSD), or `select/poll` fallback. Non-blocking I/O means Nginx can multiplex thousands of connections.
* **Modules:** core, event, HTTP (and sub-modules), stream (TCP/UDP proxying), mail. Modules may be built-in or dynamic.
* **Configuration flow:** hierarchical — `nginx.conf` → `http` → `server` → `location` and directives inherit from parents.
* **Worker connections and limits:** `worker_processes`, `worker_connections`, `worker_rlimit_nofile` are critical. Calculate max concurrent connections ≈ worker_processes * worker_connections.

**Key operational notes:**

* Graceful reload: `nginx -t && nginx -s reload`.
* Binary upgrades (zero-downtime): `nginx -s quit`/`kill -USR2` → advanced workflows exist (see appendix).

---

## 3. Installing Nginx — quick start and package choices

**Description:** Options for installing (OS package, official repository, compiled, or Nginx Plus).

* **Debian/Ubuntu (official package):** `apt install nginx`
* **Official Nginx repo:** enables newer versions, `apt-key` and `sources.list` configuration.
* **RHEL/CentOS/Fedora:** `yum`/`dnf` or official RPM repo.
* **Compiling from source:** when you need custom modules (e.g., `ngx_brotli`, `google_perftools`, third-party). Keep reproducible build scripts and versions.
* **Nginx Plus:** commercial with additional features (dashboard, advanced health checks, dynamic reconfiguration).

**Verification:** `nginx -v`, `nginx -V` (shows compiled modules and flags), `systemctl status nginx`.

---

## 4. Request forwarding and proxying

**Description:** How to safely proxy requests to backend apps, pass headers, manage timeouts, and use buffering.

### Basic proxying

```nginx
location / {
    proxy_pass http://backend.example.internal:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_redirect off;
}
```

**Notes:** always set `X-Forwarded-*` headers if backend needs client IP / proto awareness.

### Timeouts & buffering

```nginx
proxy_connect_timeout 5s;   # time to establish TCP connection
proxy_send_timeout 30s;     # time to send request to backend
proxy_read_timeout 60s;     # time to read response from backend
proxy_buffering on;         # on by default (tune for streaming)
proxy_buffers 8 16k;
proxy_busy_buffers_size 32k;
```

### Health checks (open-source)

* Passive health checks: `max_fails` + `fail_timeout` in upstream definitions.
* Active health checks: available with Nginx Plus or using `nginx-module-vts` / external scripts (Consul, keepalived, custom sidecars).

---

## 5. URL rewriting and redirects

**Description:** How to rewrite URLs safely and use redirects for SEO / client routing.

* `rewrite` changes the URI and optionally issues redirect codes.
* `return` is preferred for simple redirects (fast path).
* Use `break` to stop rewrite processing and `last` to jump to location search.

**Examples**

```nginx
# Permanent redirect for SEO
location = /old { return 301 /new; }

# Internal rewrite for internal routing
location /api/ {
    rewrite ^/api/v1/(.*)$ /v1/$1 break;
    proxy_pass http://backend;
}
```

**Pitfall:** avoid complex `if` blocks. Use `map` or `try_files` where possible.

---

## 6. Request mirroring

**Description:** Mirror real production traffic to a staging/test backend to validate changes without impacting users.

* Use `mirror` and `mirror_request_body` to copy inbound requests to an internal location that forwards to a mirror backend.
* Mirrors must be `internal` and should be lightweight; mirrored responses are discarded.

**Example**

```nginx
location / {
    mirror /mirror;
    proxy_pass http://primary-backend;
}

location /mirror {
    internal;
    proxy_pass http://mirror-backend;
    proxy_set_header X-Real-IP $remote_addr;
}
```

**Operational notes:** watch traffic cost and backend load; limit mirroring for specific paths only.

---

## 7. WebSocket support

**Description:** Proxying long-lived WebSocket connections requires `Upgrade` headers and `proxy_http_version 1.1`.

```nginx
location /ws/ {
    proxy_pass http://ws-backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
}
```

**Notes:** tune `proxy_read_timeout` as WebSockets are long-lived.

---

## 8. Conditionals and variables

**Description:** Use variables and minimal conditionals for routing decisions while avoiding `if` pitfalls.

* Common variables: `$remote_addr`, `$host`, `$uri`, `$args`, `$scheme`, `$request_method`, `$status`.
* `map` is preferred to `if` for branching based on variables.

**Example with `map`**

```nginx
map $http_user_agent $block_ua {
    default 0;
    ~*Firefox 1;
}

server {
    if ($block_ua) { return 403; }
}
```

**Pitfall:** `if` is evaluated during request processing and can cause unexpected flow; avoid when possible.

---

## 9. Log formats and variables (observability)

**Description:** Design access_log and error_log formats for debugging, analytics, and security.

**Custom log format**

```nginx
log_format json_combined '{
  "time":"$time_iso8601",
  "remote_addr":"$remote_addr",
  "host":"$host",
  "request":"$request",
  "status":$status,
  "bytes":$body_bytes_sent,
  "request_time":$request_time,
  "upstream_time":"$upstream_response_time",
  "ua":"$http_user_agent"
}';

access_log /var/log/nginx/access.log json_combined;
```

**Integration:** forward logs to central logging (Fluentd, Filebeat, Vector) and correlate with traces. Use `$request_time` and `$upstream_response_time` for latency analysis.

---

## 10. SSL/TLS configuration and security hardening

**Description:** TLS termination, modern cipher suites, HSTS, OCSP stapling, and rate-limiting TLS parameters.

**Production-grade TLS**

```nginx
ssl_protocols TLSv1.2 TLSv1.3;
ssl_prefer_server_ciphers on;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 1d;
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:...';
ssl_ecdh_curve X25519:P-256:P-384;
ssl_session_tickets off; # mitigate session ticket issues
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
```

**OCSP stapling**

```nginx
ssl_stapling on;
ssl_stapling_verify on;
resolver 8.8.8.8 8.8.4.4 valid=300s;
resolver_timeout 5s;
```

**Certificates:** use ACME (Certbot) or an automated CA (HashiCorp Vault, step-ca) when generating certs. Automate renewals and health checks.

**TLS notes:** prefer TLSv1.3 where supported, avoid weak ciphers, and verify with testssl.sh or SSL Labs.

---

## 11. HTTP/2 and QUIC (HTTP/3)

**Description:** HTTP/2 support is widely available; QUIC/HTTP/3 support requires specific builds (nginx-quic or 3rd party builds) or vendor edition.

**HTTP/2 example**

```nginx
server {
    listen 443 ssl http2;
    server_name example.com;
}
```

**HTTP/3 notes**

* Nginx mainline supports QUIC/HTTP3 in specific builds (with BoringSSL or quic-enabled forks). Production use requires careful testing.
* QUIC uses UDP; configure firewall and monitoring accordingly.

---

## 12. Load balancing and high availability

**Description:** Load balancer configuration strategies, session affinity, sticky sessions, and connection reuse.

**Upstream options**

```nginx
upstream backend {
    least_conn;             # or round-robin (default), ip_hash, hash
    server app1:8080 max_fails=3 fail_timeout=10s;
    server app2:8080 max_fails=3 fail_timeout=10s;
    keepalive 32;
}
```

**Sticky session example (cookie-based)**

```nginx
upstream backend {
    sticky cookie srv_id expires=1h path=/;
    server app1:8080;
    server app2:8080;
}
```

**Active health checks** require Nginx Plus; for OSS, rely on service discovery + probe orchestration (Kubernetes readiness/liveness, Consul, or external health-checkers).

**HA Patterns**

* Pair Nginx instances behind VIP (keepalived) or DNS round-robin with health checks.
* Use configuration management (Ansible, Salt, Terraform) with immutable images and controlled rollouts.

---

## 13. Caching strategies and CDN integration

**Description:** Edge caching with Nginx and how to integrate Nginx with CDNs for global delivery.

**Proxy cache example**

```nginx
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=mycache:100m max_size=10g inactive=60m use_temp_path=off;

location / {
    proxy_cache mycache;
    proxy_cache_key "$scheme://$host$request_uri";
    proxy_cache_valid 200 301 302 1h;
    proxy_cache_bypass $cookie_nocache;
    proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
    add_header X-Cache-Status $upstream_cache_status;
    proxy_pass http://backend;
}
```

**CDN notes:** Set appropriate `Cache-Control` and `Expires` headers in app or Nginx. Use `stale-while-revalidate` patterns where supported. Offload global traffic to Cloudflare/AKAMAI/Cloud CDN and keep Nginx as origin.

---

## 14. Security best practices and access control

**Description:** Hardening Nginx, minimizing attack surface, and protecting from common web threats.

* Run Nginx as non-root worker user.
* Remove server tokens: `server_tokens off;`
* Limit allowed methods: `if ($request_method !~ ^(GET|HEAD|POST)$) { return 405; }`
* Rate limiting: `limit_req_zone` + `limit_req` to protect upstreams from abusive clients.
* Connection limiting: `limit_conn_zone` + `limit_conn`.
* Use `auth_basic` for internal admin locations and protect endpoints with IP whitelists where possible.
* WAF: integrate ModSecurity (nginx-modsecurity) or vendor WAF in front of Nginx for deep inspection.

**Example rate limit**

```nginx
limit_req_zone $binary_remote_addr zone=one:10m rate=10r/s;

server {
    location /api/ {
        limit_req zone=one burst=20 nodelay;
    }
}
```

---

## 15. Performance optimization and tuning

**Description:** OS and Nginx tuning for latency and throughput.

**Nginx worker tuning**

```nginx
worker_processes auto;
worker_rlimit_nofile 65535;
worker_connections 16384;
use epoll;
```

**OS tuning**

* Increase `nofile` ulimits for user running Nginx.
* `/etc/sysctl.conf`: `net.core.somaxconn`, `net.ipv4.tcp_tw_reuse`, `net.ipv4.ip_local_port_range` tuning.
* Use `tcp_nopush` and `tcp_nodelay` where appropriate.

**Compression and static file optimizations**

```nginx
gzip on;
gzip_comp_level 5;
gzip_vary on;
gzip_proxied any;
expires max;
add_header Cache-Control "public";
```

**Keepalive tuning**

```nginx
keepalive_timeout 65;
keepalive_requests 100;
```

---

## 16. Docker Compose and containerized Nginx

**Description:** Example Docker Compose for a reverse proxy + app backends. Includes log forwarding and volume layout.

**Folder layout**

```
project/
├─ docker-compose.yml
├─ nginx/
│  ├─ conf.d/
│  │  └─ default.conf
│  ├─ nginx.conf
│  └─ ssl/
└─ app/
   └─ Dockerfile
```

**docker-compose.yml**

```yaml
version: '3.8'
services:
  nginx:
    image: nginx:stable
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - /var/log/nginx:/var/log/nginx
    depends_on:
      - app1
      - app2

  app1:
    build: ./app
    environment:
      - PORT=8080

  app2:
    build: ./app
    environment:
      - PORT=8081
```

**Notes:**

* Use bind mounts for config in development, and baked images (ConfigMap/Secrets) in production.
* Forward container logs to stdout/stderr to let the runtime collect them, or mount `/var/log/nginx` and run a log shipper.

---

## 17. Kubernetes deployment patterns (Ingress / DaemonSet / Sidecar)

**Description:** Multiple ways to run Nginx in Kubernetes depending on role: ingress controller, edge proxy, or as sidecar.

### Option A — Nginx Ingress Controller

* Use `kubernetes/ingress-nginx` (controller maintained by Kubernetes community) or `NGINX Ingress Controller` by Nginx Inc.
* Configure `Ingress` resources to map host/path to services.

**Ingress example**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: frontend-svc
            port:
              number: 80
```

### Option B — DaemonSet at Edge

* Run Nginx on every node as DaemonSet for node-local caching or TCP/UDP ingress.

### Option C — Sidecar

* Attach Nginx as sidecar for app-specific TLS termination, caching, or WAF.

**Kubernetes deployment tips**

* Use `ConfigMap` for Nginx config and `Secret` for TLS certs.
* Use readiness/liveness probe for the app, not Nginx.
* Prefer immutable config images + rolling update strategy.

---

## 18. Production-ready config example (modular)

**Description:** A realistic production-ready Nginx layout split into modular files with includes.

**Folder structure**

```
/etc/nginx/
├─ nginx.conf
├─ conf.d/
│  ├─ global.conf
│  └─ upstreams.conf
├─ sites-enabled/
│  └─ example.com.conf
├─ snippets/
│  ├─ tls.conf
│  └─ proxy.conf
└─ ssl/
   ├─ example.com.crt
   └─ example.com.key
```

**/etc/nginx/nginx.conf**

```nginx
user www-data;
worker_processes auto;
worker_rlimit_nofile 65536;
error_log /var/log/nginx/error.log crit;
pid /run/nginx.pid;

events { worker_connections 16384; use epoll; }

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$request_time" "$upstream_response_time"';

    access_log /var/log/nginx/access.log main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites-enabled/*;
}
```

**/etc/nginx/snippets/proxy.conf**

```nginx
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_connect_timeout 5s;
proxy_send_timeout 30s;
proxy_read_timeout 60s;
proxy_buffering on;
proxy_buffers 16 16k;
proxy_busy_buffers_size 64k;
proxy_temp_file_write_size 64k;
```

**/etc/nginx/sites-enabled/example.com.conf**

```nginx
upstream backend_nodes {
    server 10.0.0.11:8080 max_fails=3 fail_timeout=10s;
    server 10.0.0.12:8080 max_fails=3 fail_timeout=10s;
    keepalive 32;
}

server {
    listen 80;
    server_name example.com;
    # redirect http -> https
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name example.com;

    include snippets/tls.conf;

    location / {
        include snippets/proxy.conf;
        proxy_cache my_cache;
        proxy_cache_valid 200 302 1h;
        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
        proxy_pass http://backend_nodes;
    }

    location /static/ {
        root /var/www/example;
        expires 30d;
        add_header Cache-Control "public";
    }
}
```

**/etc/nginx/snippets/tls.conf**

```nginx
ssl_certificate /etc/nginx/ssl/example.com.crt;
ssl_certificate_key /etc/nginx/ssl/example.com.key;
ssl_protocols TLSv1.2 TLSv1.3;
ssl_prefer_server_ciphers on;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 1d;
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:...';
ssl_stapling on;
ssl_stapling_verify on;
resolver 1.1.1.1 8.8.8.8 valid=300s;
resolver_timeout 5s;
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
```

**Cache config (conf.d/cache.conf)**

```nginx
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=my_cache:200m max_size=50g inactive=60m use_temp_path=off;
```

**Operational checklist for production**

* Linter & test: `nginx -t` after changes.
* Use CI to build and sign config artifacts.
* Automated deployment: Ansible/Terraform/Flux/ArgoCD.
* Secrets: store TLS certs in Vault/Secrets Manager and populate as Kubernetes Secrets or OS-managed files.

---

## 19. Best-practice templates and folder structure

**Description:** Reproducible layout for a team to follow, with automation hooks.

**Recommended repo layout** (git)

```
infra/nginx/
├─ ansible/
│  └─ roles/nginx/
├─ k8s/
│  └─ nginx-ingress/
├─ docker/
│  └─ nginx/
├─ config/
│  ├─ nginx.conf
│  └─ sites/
│     └─ example.com.conf
└─ README.md
```

**CI / CD pipeline steps**

1. Lint config & run `nginx -t` in a containerized environment.
2. Run integration tests (smoke test endpoints via test harness).
3. Build artifacts (Docker image with config baked or config package for OS deployment).
4. Deploy to canary hosts (10%) with health checks.
5. Rollout to 100% if metrics are OK.

**Idempotent Ansible role tasks**

* Ensure package present (or use pre-built image)
* Upload systemd unit & configure user limits
* Deploy config files (template) and `nginx -t && systemctl reload`
* Configure log rotation and monitoring

---

## 20. Troubleshooting, debugging, and runbook

**Description:** Steps to follow when incidents occur.

**Common issues & checks**

* `nginx -t` fails → syntax error + check included files.
* 502/504 from Nginx → backend unreachable or timeouts. Check `upstream` health, `proxy_*_timeout`, and backend logs.
* SSL handshake failures → check cert validity, ciphers, `ssl_protocols`, and `nginx -V` for TLS libs.
* High connection usage → check `netstat -tnp | grep nginx`, `ss -s`, and tune `worker_connections`.

**Debugging tips**

* Turn on debug logging for a short period: `error_log /var/log/nginx/error.log debug;` (very verbose)
* Use `curl -v`/`openssl s_client` to inspect TLS and header flows.
* Use `nginx -T` to dump effective config (after includes) for review.

**Runbook: 5-minute checklist for 502 errors**

1. `systemctl status nginx` and `journalctl -u nginx -n 200`.
2. `nginx -t`.
3. Check backend `curl http://127.0.0.1:8080/health`.
4. Check upstreams in `upstream` block for DNS or IP drift.
5. Rollback recent config change if necessary.

---

## 21. Appendix: Useful commands, references, and checklist

**Commands**

* Test config: `nginx -t`
* Dump config: `nginx -T`
* Reload gracefully: `nginx -s reload` or `systemctl reload nginx`
* Stop: `systemctl stop nginx`
* Start: `systemctl start nginx`
* Check version & build flags: `nginx -V`

**Checklist before production deploy**

* [ ] `nginx -t` OK
* [ ] Certificates valid and auto-renewal configured
* [ ] Logging and monitoring pipelines configured
* [ ] Health checks / readiness probes working
* [ ] Rate limiting and security rules applied
* [ ] Performance tuning ops reviewed

**References & tools**

* Official docs: [https://nginx.org/en/docs/](https://nginx.org/en/docs/)
* Nginx Ingress Controller (k8s): [https://kubernetes.github.io/ingress-nginx/](https://kubernetes.github.io/ingress-nginx/)
* SSL testing: [https://www.ssllabs.com/ssltest/](https://www.ssllabs.com/ssltest/)
* testssl.sh for TLS testing

---

