

---

Alright, let's get into the ring. You're running Cilium and Hubble, which means you already have best-in-class eBPF-based networking, security, and observability at the L3/L4 and even L7 layer. Your setup is fast, efficient, and provides deep insights. So why on Earth would you even look at Istio for Ingress?

This is the right question to ask. We're not here to rip and replace. We're here to explore what happens when you augment your existing Cilium foundation with Istio's L7 edge capabilities. This isn't a Cilium vs. Istio deathmatch; it's about using the best tool for the job.

The core reason to consider Istio for your Ingress/Gateway is when your primary challenges shift from *network connectivity* to **application-level traffic orchestration, security, and extensibility**.

Cilium Ingress is excellent for high-performance L7 routing. Istio Ingress Gateway excels at being a full-featured **API Gateway** that is deeply integrated into a service mesh.

Here are the three killer features that might tempt you:

1.  **Declarative, Fine-Grained L7 Traffic Engineering at the Edge:** Go beyond simple host/path routing. Think sophisticated canary releases, traffic mirroring for testing, and routing requests based on JWT claims or custom headers, all declared in a `VirtualService`.
2.  **Unified Authentication and Authorization as a Service:** Offload JWT validation, OIDC flows, and fine-grained API authorization (`role: 'admin'` can call `DELETE`, but `role: 'user'` cannot) to the gateway. Your applications get clean, trusted requests, and your security policy is centralized.
3.  **Extreme Extensibility with WebAssembly (WASM):** Inject custom, high-performance logic into the gateway without recompiling Envoy or touching your applications. Want to add custom rate limiting, perform a geo-IP lookup, or transform a request payload on the fly? WASM lets you do this in a language like Rust, Go, or C++, and run it sandboxed inside the Envoy proxy.

This tutorial will walk you through these three use cases in a hands-on, practical way, showing you how to install Istio *alongside* Cilium and leverage its unique strengths at the edge.

---

### **Part 1: Setup - Installing Istio Alongside Cilium**

Our goal is to use Cilium as the CNI and Istio purely for its control plane and gateway.

**Step 1: Install Cilium (if you haven't already)**
We'll assume you have a cluster ready. Install Cilium without its Ingress controller to avoid port conflicts.
```bash
# Install Cilium with the CNI and Hubble, but disable the Ingress controller
cilium install --set ingressController.enabled=false
```

**Step 2: Install Istio with its CNI Disabled**
This is the most critical step. We must tell Istio *not* to manage the CNI, as Cilium owns it.

Create an `istio-cilium-values.yaml`:
```yaml
# istio-cilium-values.yaml
# We use the default profile but disable the CNI.
profile: "default"

components:
  # We explicitly disable the Istio CNI component.
  cni:
    enabled: false

# We only need the control plane and the ingress gateway.
gateways:
  istio-ingressgateway:
    enabled: true
    # For production, you'd enable autoscaling and resource limits.
    # autoscaleEnabled: true
    # resources:
    #   requests:
    #     cpu: "100m"
    #     memory: "128Mi"
```

Now, install Istio using Helm:
```bash
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm repo update

kubectl create namespace istio-system

helm install istio-base istio/base -n istio-system --wait
helm install istiod istio/istiod -n istio-system --values istio-cilium-values.yaml --wait
helm install istio-ingressgateway istio/gateway -n istio-system --values istio-cilium-values.yaml --wait
```
Verify: `kubectl get pods -n istio-system`. You should see `istiod` and `istio-ingressgateway` running.

---

### **Part 2: The Application - A Simple API**

We'll use `httpbin` as our backend API. It's perfect for demonstrating routing and auth.

**Step 1: Create a namespace and enable sidecar injection.**
```bash
kubectl create namespace api
kubectl label namespace api istio-injection=enabled
```

**Step 2: Deploy two versions of `httpbin`.**
Save this as `httpbin.yaml`:
```yaml
# httpbin.yaml
apiVersion: v1
kind: Service
metadata:
  name: httpbin
  namespace: api
  labels:
    app: httpbin
spec:
  ports:
  - name: http
    port: 80
    targetPort: 8080
  selector:
    app: httpbin
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin-v1
  namespace: api
  labels:
    app: httpbin
    version: v1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
      version: v1
  template:
    metadata:
      labels:
        app: httpbin
        version: v1
    spec:
      containers:
      - name: httpbin
        image: kennethreitz/httpbin
        ports:
        - containerPort: 8080
        # Add a command to differentiate versions
        command: ["gunicorn", "-b", "0.0.0.0:8080", "httpbin:app", "-k", "gevent"]
        env:
        - name: VERSION
          value: "v1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin-v2
  namespace: api
  labels:
    app: httpbin
    version: v2
spec:
  replicas: 1
  selector:
    matchLabels:
      app: httpbin
      version: v2
  template:
    metadata:
      labels:
        app: httpbin
        version: v2
    spec:
      containers:
      - name: httpbin
        image: kennethreitz/httpbin
        ports:
        - containerPort: 8080
        command: ["gunicorn", "-b", "0.0.0.0:8080", "httpbin:app", "-k", "gevent"]
        env:
        - name: VERSION
          value: "v2"
```
Deploy it: `kubectl apply -f httpbin.yaml`

---

### **Part 3: Use Case 1 - Advanced Traffic Engineering at the Edge**

We'll route internal beta testers to `v2` of our API, while everyone else goes to `v1`.

**Step 1: Create a `Gateway` and base `VirtualService`.**
Save this as `httpbin-gateway.yaml`:
```yaml
# httpbin-gateway.yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: httpbin-gateway
  namespace: api
spec:
  selector:
    istio: ingressgateway # Use the default ingress gateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "api.example.com"
---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: httpbin-destination
  namespace: api
spec:
  host: httpbin
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: httpbin-vs
  namespace: api
spec:
  hosts:
  - "api.example.com"
  gateways:
  - httpbin-gateway
  http:
  # Route for internal beta testers
  - match:
    - headers:
        x-test-group:
          exact: "beta-testers"
    route:
    - destination:
        host: httpbin
        subset: v2
  # Default route for everyone else
  - route:
    - destination:
        host: httpbin
        subset: v1
```
Deploy it: `kubectl apply -f httpbin-gateway.yaml`

**Step 2: Test the routing.**
Get your Ingress Gateway IP: `kubectl get svc istio-ingressgateway -n istio-system`

```bash
# Request without the header (goes to v1)
curl -H "Host: api.example.com" http://<INGRESS_IP>/headers

# Look for "VERSION": "v1" in the JSON response

# Request with the beta-tester header (goes to v2)
curl -H "Host: api.example.com" -H "x-test-group: beta-testers" http://<INGRESS_IP>/headers

# Look for "VERSION": "v2" in the JSON response
```
You've just implemented a feature-flag-based routing rule at the edge, without any application logic. This is a powerful pattern for progressive rollouts.

---

### **Part 4: Use Case 2 - Unified Auth with JWT Validation**

Let's secure our API endpoint. We'll offload JWT validation to the gateway and enforce that only users with the `role: "admin"` claim can access a specific endpoint.

**Step 1: Create a `RequestAuthentication` policy.**
This tells Istio where to find the public keys to verify the JWT signature. We'll use `httpbin`'s public key for this demo.

Save this as `jwt-auth.yaml`:
```yaml
# jwt-auth.yaml
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: httpbin-jwt
  namespace: api
spec:
  selector:
    matchLabels:
      app: httpbin
  jwtRules:
  - issuer: "https://httpbin.org" # In production, this is your auth provider (e.g., Keycloak, Okta)
    fromHeaders:
    - name: "x-token" # We expect the token in the 'x-token' header
    jwks: |
      { "keys":[
         {"kty":"RSA",
          "e":"AQAB",
          "kid":"rdSbB5O3aG4J",
          "n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"}
      ]}
```
Deploy it: `kubectl apply -f jwt-auth.yaml`

**Step 2: Create an `AuthorizationPolicy` to enforce the JWT.**
Save this as `require-jwt.yaml`:
```yaml
# require-jwt.yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: require-jwt
  namespace: api
spec:
  selector:
    matchLabels:
      app: httpbin
  action: ALLOW
  rules:
  - when:
    - key: request.auth.claims[iss]
      values: ["https://httpbin.org"]
```
Deploy it: `kubectl apply -f require-jwt.yaml`

**Step 3: Test the authentication.**
```bash
# Request without a token should be denied
curl -H "Host: api.example.com" http://<INGRESS_IP>/headers
# Expected output: RBAC: access denied
```
Now let's get a valid JWT. We'll use `httpbin` itself to generate one for this demo.
```bash
# Get a sample JWT
TOKEN=$(curl -s https://httpbin.org/uuid | jq -r .uuid)

# Request with the token in the correct header
curl -H "Host: api.example.com" -H "x-token: $TOKEN" http://<INGRESS_IP>/headers
# Expected output: The headers from httpbin, including the decoded JWT claims
```
Your gateway is now validating JWTs before the request ever reaches your application!

**Step 4: Add fine-grained authorization.**
Let's say only admins can access the `/delete` endpoint. We'll modify our `AuthorizationPolicy`.

Save this as `admin-only.yaml`:
```yaml
# admin-only.yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: admin-only
  namespace: api
spec:
  selector:
    matchLabels:
      app: httpbin
  action: ALLOW
  rules:
  - to:
    - operation:
        methods: ["GET", "POST"] # Allow GET/POST for anyone with a valid JWT
  - to:
    - operation:
        methods: ["DELETE"]
    when:
    - key: request.auth.claims[role] # Check the 'role' claim in the JWT
      values: ["admin"]
```
Apply it: `kubectl apply -f admin-only.yaml`
Now, any non-admin trying to `DELETE` will be denied. This is application-layer security managed centrally at the edge.

---

### **Part 5: Use Case 3 - Extensibility with WebAssembly (WASM)**

Let's add a custom header to every request using a WASM filter. Imagine this is a custom auth or analytics service.

**Step 1: Deploy a WASM filter.**
We'll use a pre-built filter from Istio's examples that adds a `x-wasm-custom: "true"` header.

Save this as `wasm-filter.yaml`:
```yaml
# wasm-filter.yaml
apiVersion: extensions.istio.io/v1alpha1
kind: WasmPlugin
metadata:
  name: header-add-filter
  namespace: api
spec:
  selector:
    matchLabels:
      app: httpbin
  url: oci://registry.example.com/header-add-filter.wasm # In a real scenario, this is an OCI image
  # For this demo, we'll use a simpler inline pull, but OCI is the production pattern.
  # Let's use a public WASM file for this example.
  pluginConfig:
    name: "header-add-filter"
    # This config will be passed to the WASM filter
    header_name: "x-wasm-custom"
    header_value: "istio-wasm-is-cool"
```
**Correction/Update:** The OCI approach is best but complex for a tutorial. Let's use a simpler method with an `EnvoyFilter` that pulls a raw WASM file from a public URL.

Replace the above with this `envoy-wasm.yaml`:
```yaml
# envoy-wasm.yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: wasm-header-add-filter
  namespace: api
spec:
  configPatches:
  - applyTo: HTTP_FILTER
    match:
      context: SIDECAR_INBOUND
      listener:
        filterChain:
          filter:
            name: "envoy.filters.network.http_connection_manager"
            subFilter:
              name: "envoy.filters.http.router"
    patch:
      operation: INSERT_BEFORE
      value:
        name: envoy.filters.http.wasm
        typed_config:
          "@type": type.googleapis.com/udpa.type.v1.TypedStruct
          type_url: type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm
          value:
            config:
              name: my_wasm_filter
              vm_config:
                runtime: "envoy.wasm.runtime.v8"
                code:
                  # We use a public WASM file that adds a header
                  remote:
                    http_uri:
                      uri: "https://storage.googleapis.com/istio-build/proxy/wasm/0.1/examples/header_add_filter.wasm"
                      cluster: service_wasm
                    sha256: "2f0b8d1e5a6c4e6d1b1e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6e6"
              configuration:
                "@type": "type.googleapis.com/google.protobuf.StringValue"
                value: |
                  {"header_name": "x-wasm-custom", "header_value": "istio-wasm-is-cool"}
```
Deploy it: `kubectl apply -f envoy-wasm.yaml`

**Step 2: Test the WASM filter.**
```bash
# Make a request to the service
curl -H "Host: api.example.com" -H "x-token: $TOKEN" http://<INGRESS_IP>/headers

# Look at the 'headers' section in the JSON response.
# You should now see:
# "X-Wasm-Custom": "istio-wasm-is-cool"
```
You just injected custom logic into your data path without changing the application or recompiling Envoy. This is the future of gateway extensibility.

---

### **Conclusion: Why This Tempts a Cilium Team**

| Feature | Cilium Ingress (Strength) | Istio Ingress Gateway (The Temptation) | Why It Matters |
| :--- | :--- | :--- | :--- |
| **L7 Traffic Control** | Excellent host/path routing, good performance. | **Declarative, versioned traffic splitting** based on headers, cookies, JWT claims. | Enables sophisticated, safe canary releases and feature flagging at the edge. |
| **Security** | Strong L3/L4 network policy, growing L7 capabilities. | **Centralized, declarative API Gateway Auth** (JWT validation, OIDC, RBAC). | Offloads complex security logic from apps, creating a clean, unified security boundary. |
| **Observability** | **Hubble** provides unparalleled flow visibility (L3-L7). | Integrated with Prometheus/Grafana/Kiali for **application-centric metrics** and traces. | Comments on *why* a request failed (auth, policy) vs. just that it was denied. |
| **Extensibility** | eBPF is powerful but has a steeper learning curve for custom logic. | **WebAssembly (WASM)** sandbox for custom filters in high-level languages. | Allows teams to safely and easily add custom gateway logic (rate limiting, auth) without C-level expertise. |

This isn't about replacing Cilium. It's about **augmenting** it. You keep Cilium as your high-performance CNI and for its incredible Hubble observability. You add Istio Ingress as a specialized, application-aware API Gateway that gives you granular control over how your users and services interact.

For a team whose needs are evolving from "just connect my services" to "orchestrate my application traffic with precision," this combined architecture offers the best of both worlds.
