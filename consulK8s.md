# learn Hashicorp consul + k8s
Great — below is a production-ready, step-by-step tutorial that walks you from the real-world why through installing Consul on Kubernetes, adding VM services, and integrating Consul into a cluster that already runs **Cilium CNI** and **Traefik**. I’ve included commands, commented config files, and concrete examples for each step. Treat this as a runbook you can follow on a fresh cluster or adapt for an existing one.

---

## Section 1: The "Why" — Real-World Use Case (ShopSphere)

### The scenario

ShopSphere is an e-commerce company with:

* New microservices running in Kubernetes: `product-catalog`, `checkout`, `user-profiles`.
* Legacy services (inventory, billing) on VMs in a separate network segment.
* Current discovery: K8s DNS inside the cluster and static IPs for VM services.

### Problems

* Two service networks (Kubernetes + VMs) are managed differently → operational complexity.
* No unified health checking or centralized catalog for cross-environment discovery.
* No easy way to do advanced L7 traffic management across both environments (e.g., route 5% of checkout traffic to a new implementation running in K8s while the rest hits the VM implementation).
* Feature flags configuration is scattered.

### How Consul helps

* **Unified service catalog**: Consul agents (K8s and VM) register services into the same catalog (same datacenter / logical Consul cluster). Pods and VMs can discover each other with a single API or DNS namespace.
* **Service mesh (Consul Connect)**: Deploy Envoy sidecars injected into K8s pods and run Consul sidecars on VMs. Consul Connect provides mTLS, intentions, and traffic splitting. Example: route 95% of `checkout` traffic to legacy VM implementation and 5% to the new K8s `checkout-v2`.
* **Key/Value store**: Use KV for feature flags or configuration that all services (VM or K8s) can read dynamically.
* **Health checks & observability**: Centralized health checks, UI and API for catalog inspection and mesh status.

---

## Section 2: Core Tutorial — Deploying Consul on Kubernetes (Helm)

### Prerequisites

* Kubernetes cluster (1.21+ recommended; adjust to your environment).
* `kubectl` configured to the cluster.
* `helm` (v3.x).
* `openssl` (optional, for generating certs if you want).
* Access to cluster nodes to join VMs to Consul (Section 3).

> Note: This guide uses HashiCorp's official Helm chart (`hashicorp/consul`).

---

### Add the HashiCorp Helm repo

```bash
# Add repo and update
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update
```

---

### values.yaml — production-ready but minimal

Below is a **commented** `values.yaml` for a high-availability, production-minded Consul on Kubernetes. Save it as `consul-values.yaml`.

```yaml
# consul-values.yaml
global:
  name: "consul"           # Release name / prefix for resources
  enabled: true

server:
  # Use 3 server replicas for RAFT quorum (HA)
  replicas: 3
  # Persist server data - important for production
  dataStorage:
    enabled: true
    storageClass: "standard"    # change to your cluster's StorageClass
    accessMode: ReadWriteOnce
    size: 10Gi

ui:
  enabled: true          # enable Consul Web UI

# Enable Consul Connect automatic sidecar injection
connectInject:
  enabled: true

# Consul Controller syncs k8s services to Consul and vice versa
controller:
  enabled: true

# Prepare ACLs for production (optional but recommended)
acl:
  enabled: true
  # bootstrapToken: <optionally provide existing bootstrap token>
  manageSystemACLs: true

# Configure telemetry and resource requests/limits
telemetry:
  enabled: true

# Resource requests/limits can be tuned per environment
serverResources:
  requests:
    cpu: "500m"
    memory: "1Gi"
  limits:
    cpu: "1"
    memory: "2Gi"

client:
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"

# RBAC & service accounts should be enabled automatically
rbac:
  create: true

# Optional: set pod disruption budgets for server pods
serverPDB:
  enabled: true
  minAvailable: 2

# (Optional) enable TLS encryption for gRPC, HTTP between agents if required
# datacenter: "dc1"  # optional custom datacenter name
```

**Key explanations**

* `server.replicas: 3` gives you RAFT quorum for HA (failures tolerated: 1).
* `connectInject.enabled: true` enables automatic sidecar injection for pods when annotated/labelled.
* `controller.enabled: true` deploys the Consul Kubernetes Controller which syncs k8s <-> Consul services.
* `acl.enabled: true` — for production, enable ACLs and manage tokens.

---

### Helm install

```bash
# Create namespace
kubectl create ns consul

# Install Consul with our values
helm install consul hashicorp/consul -n consul -f consul-values.yaml
```

---

### Verification

Check pods and state:

```bash
kubectl get pods -n consul -o wide
# Expect: consul-server-0/1/2, consul-client-... , consul-controller-..., consul-connect-injector-...

# For logs of a server:
kubectl logs -n consul statefulset/consul-server -c consul

# Port-forward to the UI (local access):
kubectl port-forward -n consul svc/consul-ui 8500:8500
# Then browse: http://localhost:8500
```

Check connectivity / health from inside the cluster:

```bash
# Exec into a consul client pod and query Consul members
POD=$(kubectl get pods -n consul -l "app=consul,component=client" -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n consul -it $POD -- consul members
```

---

## Section 3: Expansion — Integrating Virtual Machines into the Consul Cluster

You want VM services in the same Consul datacenter so they appear in the same catalog and can participate in Connect.

### 3.1 Install the Consul client on a VM

#### Ubuntu / Debian (systemd)

```bash
# Download Consul - choose appropriate version for your environment
CONSUL_VERSION=1.14.0
curl -LO https://releases.hashicorp.com/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_linux_amd64.zip
unzip consul_${CONSUL_VERSION}_linux_amd64.zip
sudo mv consul /usr/local/bin/
sudo mkdir -p /etc/consul.d /opt/consul
sudo useradd --system --home /etc/consul.d --shell /bin/false consul
sudo chown -R consul:consul /etc/consul.d /opt/consul
```

#### CentOS / RHEL (similar steps — ensure `unzip` / `curl` installed)

---

### 3.2 consul.hcl for the VM agent

Create `/etc/consul.d/consul.hcl`. Replace `<CONSUL_SERVER_IP_1>` etc. with addresses you can reach (Kubernetes server nodes or load-balanced server VIP). Use whichever bootstrap/join method suits your environment.

**Option A — Static server IPs (simple private network):**

```hcl
# /etc/consul.d/consul.hcl
datacenter = "dc1"
data_dir = "/opt/consul"

# Run as a client-only agent
server = false
retry_join = [
  "10.0.0.10",   # consul-server-0 (private IP or LB)
  "10.0.0.11",   # consul-server-1
  "10.0.0.12"    # consul-server-2
]

# Enable connect (to allow sidecar/intentions)
connect {
  enabled = true
}

# Advertise an interface (optional if multiple nics)
# bind_addr = "172.16.1.10"
# advertise_addr = "172.16.1.10"

# Enable UI if desired (clients can expose UI on the node)
# ui_config { enabled = true }
```

**Option B — Cloud auto-join**
If your VMs and K8s are on a cloud provider, use cloud auto-join providers (aws, gcp) or use Consul tokens with cloud metadata. Implementation depends on cloud and is not shown here.

**Option C — Use Kubernetes API provider**
If your Kubernetes servers expose a discoverable list via cloud tags or DNS, you can configure `retry_join` with a provider or DNS. For on-prem, prefer static IPs or a load balancer in front of server RPC.

---

### 3.3 systemd unit for Consul agent

Create `/etc/systemd/system/consul.service`:

```ini
[Unit]
Description=Consul Agent
Requires=network-online.target
After=network-online.target

[Service]
User=consul
Group=consul
ExecStart=/usr/local/bin/consul agent -config-dir=/etc/consul.d
ExecReload=/bin/kill -HUP $MAINPID
KillMode=process
Restart=on-failure
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

Reload and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now consul
sudo journalctl -u consul -f
```

---

### 3.4 Register a VM service (NGINX example)

Option 1 — Static service definition file `/etc/consul.d/web.json`:

```json
{
  "service": {
    "name": "inventory",
    "tags": ["vm", "nginx", "external"],
    "port": 80,
    "check": {
      "http": "http://127.0.0.1:80/health",
      "interval": "10s"
    }
  }
}
```

Reload consul:

```bash
sudo systemctl reload consul
# or
consul reload
```

Option 2 — Register at runtime using the HTTP API:

```bash
curl --request PUT --data @web.json http://127.0.0.1:8500/v1/agent/service/register
```

---

### 3.5 Verification

* From the Consul UI (`http://localhost:8500` via port-forward) you should see `inventory` (or whatever name) in the Services list.
* From a Kubernetes pod, you can query the Consul API (if network allows) or use the consul controller's sync to see the VM service.

From inside a consul client pod:

```bash
kubectl exec -n consul -it $POD -- consul catalog services
kubectl exec -n consul -it $POD -- consul catalog nodes -service=inventory
```

---

## Section 4: Advanced Integration — Existing Cluster with Cilium + Traefik

This section covers how to safely add Consul to a cluster running **Cilium** (eBPF CNI) and **Traefik** as Ingress, ensuring no conflicts and enabling Traefik to discover services from the Consul catalog.

### Pre-amble & potential conflicts

* **CNI (Cilium)**: Cilium operates at L3/L4 using eBPF; Consul Connect operates at the application layer (Envoy L7). They are orthogonal: Cilium provides packet routing and network policies; Consul Connect manages L7 routing and security. Typical issues: Cilium policies that block traffic between the app container and its sidecar proxy will break Connect—so you must allow those flows.
* **Ingress (Traefik)**: Traefik normally discovers services via K8s Ingress or Service resources. Traefik also supports a `consulCatalog` provider to read services from Consul’s catalog (including VM services). This is powerful for exposing non-K8s services.

---

### 4.1 Consul Helm config for an existing cluster

Use a `consul-values-cilium-traefik.yaml` based on the earlier `values.yaml` with these additions:

```yaml
# consul-values-cilium-traefik.yaml
global:
  name: consul
  enabled: true

server:
  replicas: 3
  dataStorage:
    enabled: true
    size: 10Gi

ui:
  enabled: true

connectInject:
  enabled: true

controller:
  enabled: true
  # Controller will create ConsulService CRDs and sync K8s services into Consul

# (Optional) set hostNetworking for the client if needed for Traefik discovery on host
client:
  hostNetwork: false   # usually false is fine

# RBAC & service accounts
rbac:
  create: true
```

Install (if not already):

```bash
helm upgrade --install consul hashicorp/consul -n consul -f consul-values-cilium-traefik.yaml
```

**Notes**

* `controller.enabled: true` ensures Kubernetes services and endpoints are synchronized with the Consul catalog. This is key so Traefik can see both K8s and VM services in Consul.
* No special Cilium flags are required for Consul; you just need to be mindful of network policies.

---

### 4.2 Configure Traefik to use the Consul Catalog provider

Traefik supports a `consulCatalog` provider. The static configuration for Traefik needs to point at a Consul agent (address + token if ACLs enabled).

#### Traefik static configuration (example YAML for Traefik v2)

`traefik-config.yaml` (this is the Traefik static config — often provided as a ConfigMap or CLI args in the Traefik Helm Chart):

```yaml
# traefik-config.yaml
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  kubernetesIngress: {}   # existing provider
  consulCatalog:
    endpoint:
      # IP/host of a Consul agent accessible by Traefik
      address: "http://consul-client.consul.svc.cluster.local:8500"
      # If ACLs are enabled, provide token through environment variable or config
      # token: "<CONSUL_HTTP_TOKEN>"  # avoid hardcoding secrets; use secret + env
    watch: true
    prefix: "traefik"   # optional prefix to use for Traefik services
```

**Notes & deployment**

* If Traefik runs inside Kubernetes, use the service DNS `consul-client.consul.svc.cluster.local:8500` (the `consul` Helm chart usually creates a `consul-client` service).
* If Consul ACLs are enabled, supply a HTTP token to Traefik securely using Kubernetes secrets and set Traefik environment to read `CONSUL_HTTP_TOKEN`.
* Deploy Traefik with this static config (or use the Traefik Helm chart values to set `providers.consulCatalog`).

---

### 4.3 Labeling and tags for Traefik discovery

Traefik will discover services in Consul that have appropriate tags/metadata. For example, register a VM service with tags:

```json
{
  "service": {
    "name": "inventory",
    "tags": ["traefik.enable=true", "traefik.frontend.rule=Host:inventory.example.com"]
  }
}
```

Or when using the controller sync, you can annotate K8s Services so the controller exports tags:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: checkout
  annotations:
    consul.hashicorp.com/service-tags: "traefik.enable=true,traefik.frontend.rule=Host:checkout.example.com"
spec:
  ports:
    - name: http
      port: 80
  selector:
    app: checkout
```

Traefik will then create a route for `checkout.example.com` based on the Consul catalog entry (no Kubernetes Ingress required).

---

### 4.4 Cilium NetworkPolicy examples for Consul Connect

If your cluster uses Cilium default-deny policies or strict network policies, you must allow:

1. Application ⇄ Consul sidecar (Envoy) communication in the same pod.
2. Envoy outbound/inbound traffic to other service Envoys or Traefik.
3. Traefik → application (if Traefik routes directly to app or to the sidecar).

Below are example CiliumNetworkPolicy (CNP) manifests.

#### Example A — Allow traffic from Traefik pods to `checkout` service pods on port 80

```yaml
# allow-traefik-to-checkout.yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-traefik-to-checkout
  namespace: default
spec:
  endpointSelector:
    matchLabels:
      app: checkout        # target pods
  ingress:
  - fromEndpoints:
    - matchLabels:
        app: traefik      # allow only Traefik pods
    toPorts:
    - ports:
      - port: "80"
        protocol: TCP
```

#### Example B — Allow sidecar <-> app intra-pod traffic (allow by podSelector)

If you use automatic connect injection, sidecar and app share a pod. With Cilium, intra-pod traffic is allowed by default, but when using L3 filters, ensure you permit all localhost or pod-UID traffic. A safe approach is to allow traffic among pods with the `consul` connect label (or to allow access to named ports that Envoy uses).

A generic allow rule for pods with Connect sidecar:

```yaml
# allow-consul-sidecar.yaml
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: allow-consul-sidecar
  namespace: default
spec:
  endpointSelector:
    matchLabels:
      "consul.hashicorp.com/connect-inject": "true"  # all pods with consul injected
  ingress:
  - fromEndpoints:
    - matchLabels:
        "consul.hashicorp.com/connect-inject": "true"
  egress:
  - toEndpoints:
    - matchLabels:
        "consul.hashicorp.com/connect-inject": "true"
```

**Explanation**

* The first policy restricts ingress to `checkout` pods to only Traefik pods on port 80.
* The second policy allows traffic between pods that have Consul Connect injected (sidecar) — this ensures Envoy→Envoy flows aren't blocked.

> Tailor the label names to what your Consul injector actually annotates. If your injector uses another label (check `kubectl get pod -o yaml`), adapt the selectors accordingly.

---

### 4.5 Workflow example: Expose a VM service via Traefik (no K8s Ingress)

1. VM service `inventory` is registered in Consul with `traefik` tags (as shown in Section 3).
2. Traefik (running in Kubernetes) uses ConsulCatalog provider to discover `inventory`.
3. Traefik creates a dynamic route and exposes `inventory.example.com` to external clients.
4. Traefik sends traffic to the Consul catalog endpoint (which may resolve to the VM address via service entry), or Traefik routes through the Consul Connect proxy if you opt to use the mesh for L7 encryption/intentions.

This means **no Kubernetes Service/Ingress** is required for the VM service to be exposed via Traefik — only the Consul registration and the proper tags are required.

---

## Summary Architecture

* **Consul Servers** (StatefulSet, 3 replicas) run in Kubernetes as the control plane (RAFT).
* **Consul Clients** run as DaemonSet pods on Kubernetes nodes and as systemd Consul agents on external VMs — all join the same datacenter/catalog.
* **Consul Controller** syncs Kubernetes Services/Endpoints ⇄ Consul catalog.
* **Connect**: Sidecar Envoy proxies injected into K8s pods manage L7 traffic and mTLS. VMs run Consul proxy (Envoy) or use `connect-proxy` with Consul agent to participate in the mesh.
* **Traefik** discovers services via both Kubernetes provider (Ingress) and ConsulCatalog, enabling unified ingress for both K8s and VM services.
* **Cilium** supplies L3/L4 network connectivity and NetworkPolicies; policies must allow Envoy/app flows.

---

## Next Steps & Operational Tips

1. **Intentions**: Configure Consul Connect *intentions* to authorize traffic at the service level (e.g., allow `traefik` → `checkout`, disallow others).

   ```bash
   consul intention create -allow traefik checkout
   ```
2. **Traffic Splitting**: Use Consul’s traffic splitting to do canary rollouts across K8s and VM implementations:

   * Create multiple service entries (`checkout-v1`, `checkout-v2`) and define a service-resolver/service-split configuration to route X% to `checkout-v2`.
3. **ACLs/Service Tokens**: If you enabled ACLs, generate and distribute tokens for services (Traefik, Consul controllers, external agents). Use Kubernetes Secrets to keep them safe.
4. **Service Mesh Observability**: Integrate Consul telemetry with Prometheus & Grafana, and enable Envoy stats where needed.
5. **KV Feature Flags**: Use the KV store for feature flags (watch for changes in apps) or integrate with external feature-flag systems.
6. **Certificates and mTLS**: Consul automates mTLS between sidecars. If you need integrations (external proxies, LB), consider terminating TLS at a gateway or using Consul's CA.
7. **Backup & Recovery**: Backup Consul server data_dir or snapshots. For RAFT, keep consistent backups and follow HashiCorp’s snapshot guidance.

---

## Appendix — Useful Commands & Quick Reference

```bash
# Helm install (recap)
kubectl create ns consul
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update
helm install consul hashicorp/consul -n consul -f consul-values.yaml

# Check status
kubectl get pods -n consul
kubectl logs -n consul statefulset/consul-server -c consul

# Port-forward UI
kubectl -n consul port-forward svc/consul-ui 8500:8500

# From k8s pod: list services in catalog
kubectl exec -n consul -it <client-pod> -- consul catalog services
```

---


# part 2 : advanced topics

Nice — here’s **Part 2**: a deep, hands-on guide covering **ACLs**, the **Consul CLI**, how to use Consul inside Kubernetes manifests (including the Consul k8s CRDs), **ServiceResolver/ServiceSplitter (ServiceSplit)** usage (both HCL + k8s CRD approaches), **KV store** usage patterns (CLI, HTTP API, `consul-template`), how to register/define services (K8s and VM), how to expose Consul-backed services in **Traefik** routes, and how to wire **Consul DNS** inside the cluster (CoreDNS) and on external VMs.

I keep things concrete: commands, full config examples, comments, and operational notes. You can apply most of these examples directly in your ShopSphere environment — tune names/addresses/ports/policies to match your infra.

---

## Table of contents

* ACLs: bootstrap, policies, tokens, locking down services
* Consul CLI quick reference (useful commands + examples)
* Deep Consul usage inside Kubernetes manifests

  * Using annotations to register and tag services
  * Using Consul CRDs: ConsulService / ServiceResolver / ServiceSplitter
* ServiceSplit & ServiceResolver

  * HCL config entry examples (`consul config write`)
  * CRD examples (k8s-native)
  * A 95/5 canary example
* KV store usage: CLI, HTTP API, `consul-template` example
* Defining services

  * On VMs (static JSON/HCL)
  * In Kubernetes (annotations & ConsulService CRD)
* Traefik + Consul integration: how to use Consul-backed services in routes

  * Tag / metadata examples that create Traefik routers/services
  * Traefik static config snippet + secure token approach
* Consul DNS

  * Inside cluster: CoreDNS stub forward for `.consul` zone
  * On VMs: configure local agent or OS resolver to use Consul agent DNS
* Appendix: troubleshooting tips & concise quick-reference commands

---

## ACLs (Access Control Lists)

### Why ACLs

When Consul is used in production (Traefik reading catalog, VM agents joining, apps reading KV), enable ACLs to restrict who can read/write to the catalog, create configuration entries, or access the KV store.

> We assume `acl.enabled: true` in the Helm values. If you installed without ACLs, enabling later requires bootstrapping and token rollout.

### Bootstrap ACLs (server-side)

Run this from a shell with the Consul server CLI access (kubectl exec into a server pod or run from a machine with port-forward to a server).

```bash
# Port-forward to a Consul server or run in a kubectl pod
kubectl -n consul port-forward svc/consul-server 8500:8500 &   # optional for UI/API access
kubectl -n consul exec -it statefulset/consul-server -- consul acl bootstrap
```

The `bootstrap` command outputs a management token and some metadata. Save it somewhere secure. Example output (trimmed):

```text
AccessorID: 00000000-0000-0000-0000-000000000000
SecretID: s.xxxxxx-management-token-xxxx
TokenID:    s.xxxxxx-management-token-xxxx
```

Keep the `SecretID` (the bootstrap management token) safe: it can create policies, tokens, and is needed to operate.

### Create a policy and a token for Traefik

1. Create a policy HCL file `traefik-policy.hcl`:

```hcl
# traefik-policy.hcl
# Allow read access to services/catalog and read to config entries
# Adjust permissions according to least privilege model
node_prefix "" {
  policy = "read"
}

service_prefix "" {
  policy = "read"
}

# Allow Traefik to read the KV keys used for routing if you store config in KV
key_prefix "" {
  policy = "read"
}

# Allow reading the config entries (service-resolver, splitter) used by Traefik
config_entry_prefix "" {
  policy = "read"
}

# Optionally allow metrics read
operator "" {
  policy = "read"
}
```

2. Create the policy and token:

```bash
# Using management token exported in CONSUL_HTTP_TOKEN env
export CONSUL_HTTP_ADDR=http://127.0.0.1:8500
export CONSUL_HTTP_TOKEN=<bootstrap-secret-id>

# create policy
consul acl policy create -name "traefik-policy" -rules @traefik-policy.hcl

# create token bound to the policy
consul acl token create -description "Traefik token" -policy-name "traefik-policy"
# output contains SecretID for Traefik to use
```

Store the created token in a Kubernetes Secret (for Traefik) or a secure vault.

### Create a more restrictive token for VM agents

VM Consul clients generally need to register services (agent-local), perform heartbeats, and join. For an agent, you might rely on bootstrap or use ACL tokens with the `agent` role. HashiCorp recommends using `node` and `service` policies for dynamic agents. Example token creation (operator-level tokens are sensitive — prefer scoped tokens):

```bash
consul acl token create -description "vm-agent-token" -policy-name "default"
```

(Adjust the policy to limit as needed.)

### Use tokens in the Helm chart (controller & consul client)

* For the Consul Kubernetes controller and Traefik you will mount the token via env var `CONSUL_HTTP_TOKEN` from a Kubernetes Secret (avoid hardcoding). Example in a K8s Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: consul-traefik-token
  namespace: kube-system   # or where Traefik lives
type: Opaque
stringData:
  CONSUL_HTTP_TOKEN: "s.xxxxxxx-your-token"
```

Then in Traefik Deployment add env:

```yaml
env:
- name: CONSUL_HTTP_ADDR
  value: "http://consul-client.consul.svc.cluster.local:8500"
- name: CONSUL_HTTP_TOKEN
  valueFrom:
    secretKeyRef:
      name: consul-traefik-token
      key: CONSUL_HTTP_TOKEN
```

---

## Consul CLI — quick reference and examples

Helpful `consul` commands you’ll use regularly (run in a pod that has the consul binary or on any machine with bin and network access to Consul HTTP API).

### Common cluster & agent info

```bash
# list members
consul members

# check agent status
consul info

# check cluster leader and peers
consul operator raft list-peers
```

### Catalog / services

```bash
# list all services
consul catalog services

# list nodes providing a service
consul catalog nodes -service=checkout

# query health (checks) for a service
consul health checks -service checkout
consul health check -node <node-name>
```

### KV store

```bash
# put a kv
consul kv put config/feature/checkout-enabled true

# get a kv (raw)
consul kv get -raw config/feature/checkout-enabled

# list keys under prefix
consul kv get -recurse config/feature

# delete a key or prefix
consul kv delete -prefix config/feature
```

### ACLs

```bash
# bootstrap (one-time, returns management token)
consul acl bootstrap

# list policies
consul acl policy list

# create policy (HCL file)
consul acl policy create -name mypolicy -rules @mypolicy.hcl

# create token
consul acl token create -description "mytoken" -policy-name "mypolicy"

# lookup token info
consul acl token read -accessor <accessor-id>
```

### Config entries (resolver / splitter / router / defaults)

You write config entries via `consul config write` which accepts HCL or JSON file.

```bash
# apply a config entry from HCL
consul config write service-resolver-checkout.hcl

# list config entries
consul config entries

# delete config entry
consul config remove -kind=service-splitter -name=checkout
```

---

## Deep Consul usage inside Kubernetes manifests

Two main ways to influence what Consul knows about your Kubernetes services:

1. **Annotations** on `Service` objects (or Pods) so the Consul controller will export K8s services into Consul with tags and metadata.
2. **Consul CRDs** (installed by the Consul controller) such as `ConsulService`, `ServiceResolver`, `ServiceSplitter/ServiceSplit`, `ServiceDefaults`, etc. CRDs allow k8s-native declarative management of Consul resources.

### Example: Annotate a Kubernetes Service to send tags into Consul

```yaml
apiVersion: v1
kind: Service
metadata:
  name: checkout
  namespace: default
  annotations:
    # export as a Consul service with tags to be read by Traefik
    consul.hashicorp.com/service-name: "checkout"            # optional override
    consul.hashicorp.com/service-tags: "traefik.enable=true,traefik.http.routers.checkout.rule=Host(`checkout.example.com`),traefik.http.services.checkout.loadbalancer.server.port=8080"
spec:
  selector:
    app: checkout
  ports:
    - name: http
      port: 80
      targetPort: 8080
```

**Notes**

* The controller will pick up the annotation and create a Consul service entry with those tags.
* Tags are comma-separated; Traefik will read these tags and interpret them as dynamic config (assuming Traefik's Consul provider supports tag-to-label mapping).

### Example: ConsulService CRD — register an external service from within K8s

If you want to register an external VM service via a k8s manifest (handy for GitOps), use the `Registration` CRD or `ConsulService` CRD depending on your controller version. Example `Registration` (the schema can differ between controller versions — check yours):

```yaml
apiVersion: consul.hashicorp.com/v1alpha1
kind: Registration
metadata:
  name: inventory-external
  namespace: consul
spec:
  name: inventory
  address: 10.0.10.20        # VM IP
  port: 80
  tags:
    - "traefik.enable=true"
    - "traefik.http.routers.inventory.rule=Host(`inventory.example.com`)"
  # health check
  checks:
  - name: http
    http: "http://10.0.10.20:80/health"
    interval: "10s"
```

This creates a Consul catalog entry for the VM service using k8s manifests — great for GitOps workflows.

---

## ServiceResolver / ServiceSplit (ServiceSplitter) — traffic management & canaries

Two interchangeable approaches:

* Use **Consul config entries** (HCL/JSON + `consul config write`) — server-side objects.
* Or use **Consul CRDs** (k8s) — Kubernetes-native.

**Important:** The Kind names vary slightly in naming. The current Consul config Kind for split is `service-splitter` (a.k.a service-splitter) — below I show HCL examples from Consul docs plus k8s CRD examples.

---

### 1) HCL approach — write config entries

#### ServiceResolver (optional subsets or failover)

`service-resolver` controls subset selection, failover, and redirect behavior.

`service-resolver-checkout.hcl`:

```hcl
# service-resolver-checkout.hcl
Kind = "service-resolver"
Name = "checkout"

# Example: define default subset behavior; you can map metadata into subsets if needed.
# Optionally configure failover or subset selection here.
```

Apply:

```bash
consul config write service-resolver-checkout.hcl
```

#### ServiceSplitter — 95/5 canary example (two subsets v1 & v2)

Two ways to express splits: split among subsets of the same service or among different service names (service subsets). Below is a splitter that routes 95% to `checkout-v1` subset and 5% to `checkout-v2`.

`service-splitter-checkout.hcl`:

```hcl
# service-splitter-checkout.hcl
Kind = "service-splitter"
Name = "checkout"
Splits = [
  {
    Weight        = 95
    ServiceSubset = "v1"
  },
  {
    Weight        = 5
    ServiceSubset = "v2"
  },
]
```

Apply:

```bash
consul config write service-splitter-checkout.hcl
```

**How it works**

* Your services must expose metadata or tags that make them be recognized as subset `v1` or `v2` — usually service metadata key `version` or `subset`.
* The sidecar/Envoy uses the splitter to decide which upstream subset to route to.

---

### 2) Kubernetes CRD approach

If you prefer Kubernetes-native YAML, install `consul-k8s` (controller) and use CRDs. The API group is usually `consul.hashicorp.com/v1alpha1`. Example ServiceSplit CRD (namespaces & exact schema depend on your controller version; adapt if your cluster uses slightly different apiVersion):

```yaml
apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceSplit
metadata:
  name: checkout-split
  namespace: default
spec:
  service: checkout               # the logical service name
  splits:
    - weight: 95
      subset: v1
    - weight: 5
      subset: v2
```

Example ServiceResolver CRD:

```yaml
apiVersion: consul.hashicorp.com/v1alpha1
kind: ServiceResolver
metadata:
  name: checkout-resolver
  namespace: default
spec:
  service: checkout
  # optional: default subset, redirects, failover rules
  # defaultSubset: v1   # (depends on your CRD schema)
```

Apply with:

```bash
kubectl apply -f service-split.yaml
kubectl apply -f service-resolver.yaml
```

**Note:** The precise CRD fields and names are tied to the `consul-k8s` version installed. Use `kubectl get crd | grep consul` and `kubectl explain serviceresolver.consul.hashicorp.com` to inspect the exact schema in your cluster.

---

### Example: full canary flow (practical steps)

1. Deploy `checkout` v1 pods with service tag `subset=v1` (annotation or service metadata).
2. Deploy `checkout` v2 pods with `subset=v2` (new deployment).
3. Create a `ServiceResolver` (if you need advanced subset mapping).
4. Create `ServiceSplit` (95/5) config entry via `consul config write` or k8s CRD.
5. Ensure Envoy sidecars are injected: `connectInject.enabled=true` or annotate pods with `consul.hashicorp.com/connect-inject: "true"`.
6. Test: send traffic to `checkout` and observe ~5% routed to v2 using logs or metrics.

---

## KV store — patterns & examples

### Basic CLI usage

```bash
# put a key
consul kv put shop/feature/checkout_v2 true

# get key (raw)
consul kv get -raw shop/feature/checkout_v2

# get with metadata and flags
consul kv get -recurse shop/feature

# delete
consul kv delete shop/feature/checkout_v2
```

### HTTP API (for programs)

* Put:

```bash
curl --request PUT --data 'true' http://127.0.0.1:8500/v1/kv/shop/feature/checkout_v2
```

* Get raw:

```bash
curl http://127.0.0.1:8500/v1/kv/shop/feature/checkout_v2?raw
```

### Using KV from apps

Options:

1. **Direct HTTP calls** — application hits Consul HTTP API. (Simple, but requires app to handle retries, token, etc.)
2. **consul-template** — render config files or environment files dynamically when KV changes, optionally trigger reloads.
3. **Envoy/Connect dynamic configuration** — some config can be staged via config entries and the service mesh.
4. **Sidecar or libraries** — use consul client libraries (Go, Python, etc.)

#### Example: consul-template to generate a feature flag file

`/etc/consul-templates/feature.tmpl`:

```
# Generated file with feature flags
checkout_v2_enabled = {{ key "shop/feature/checkout_v2" }}
```

`/etc/consul-templates/consul-template.hcl`:

```hcl
# consul-template configuration to render the file and restart app on change
template {
  source = "/etc/consul-templates/feature.tmpl"
  destination = "/etc/myapp/feature.conf"
  command = "systemctl restart myapp"   # restart app whenever file changes
  perms = "0644"
}
```

Run consul-template:

```bash
consul-template -config=/etc/consul-templates/consul-template.hcl
```

This is a robust pattern to keep apps reacting to KV changes without embedding Consul logic into the application.

---

## Defining services

### On VMs (static JSON / HCL service definition)

`/etc/consul.d/inventory.json`:

```json
{
  "service": {
    "name": "inventory",
    "port": 80,
    "tags": ["vm", "inventory", "traefik.enable=true", "traefik.http.routers.inventory.rule=Host(`inventory.example.com`)"],
    "check": {
      "http": "http://127.0.0.1:80/health",
      "interval": "10s"
    }
  }
}
```

Reload agent:

```bash
sudo consul reload
# or systemctl restart consul
```

### In Kubernetes (annotation -> controller sync) — example service + pod

Pod/Deployment (checkout v2) with connect annotation for sidecar and version label:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: checkout-v2
  labels:
    app: checkout
    version: v2
spec:
  replicas: 2
  selector:
    matchLabels:
      app: checkout
      version: v2
  template:
    metadata:
      labels:
        app: checkout
        version: v2
      annotations:
        consul.hashicorp.com/connect-inject: "true"    # auto inject Envoy sidecar
    spec:
      containers:
      - name: checkout
        image: ghcr.io/yourorg/checkout:v2
        ports:
        - containerPort: 8080
```

K8s Service with tags for the Consul controller:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: checkout
  annotations:
    consul.hashicorp.com/service-tags: "version=v2,traefik.http.routers.checkout.rule=Host(`checkout.example.com`)"
spec:
  selector:
    app: checkout
  ports:
    - port: 80
      targetPort: 8080
```

**Controller behavior**: the Consul controller watches Services and Endpoints and creates corresponding Consul services with the provided tags.

---

## Traefik + Consul — how to do ingress routing using Consul catalog

### Traefik static configuration snippet (Traefik v2)

Add `consulCatalog` provider in Traefik’s static configuration (via ConfigMap or helm values). Traefik must have network access to Consul agent and appropriate ACL token if ACLs enabled.

```yaml
# traefik-static-config.yaml (fragment)
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  kubernetesIngress: {} # keep existing k8s provider
  consulCatalog:
    endpoint:
      address: "http://consul-client.consul.svc.cluster.local:8500"
      # If ACLs are enabled, Traefik picks up CONSUL_HTTP_TOKEN from env var or consul config
    watch: true
    prefix: "traefik"
```

### Tagging a Consul service such that Traefik creates router + service

Two key tag styles that Traefik can support (depending on its Consul provider implementation):

**Example tags to create a router and service**:

* Router rule (traefik.http.routers.<name>.rule)
* Service port (traefik.http.services.<name>.loadbalancer.server.port)

When you register a service in Consul (VM JSON or K8s annotations), include tags such as:

```text
traefik.http.routers.inventory.rule=Host(`inventory.example.com`)
traefik.http.services.inventory.loadbalancer.server.port=80
traefik.http.routers.inventory.tls=true
```

**VM JSON example** (excerpt of `inventory.json` tags array):

```json
"tags": [
  "traefik.http.routers.inventory.rule=Host(`inventory.example.com`)",
  "traefik.http.services.inventory.loadbalancer.server.port=80",
  "traefik.http.routers.inventory.tls=true"
]
```

**K8s Service annotation example**:

```yaml
metadata:
  annotations:
    consul.hashicorp.com/service-tags: "traefik.http.routers.checkout.rule=Host(`checkout.example.com`),traefik.http.services.checkout.loadbalancer.server.port=80"
```

Once Traefik reads those tags from Consul, it constructs routers and services dynamically — meaning **VM services show up in Traefik without a Kubernetes Ingress**.

### Security: give Traefik a read-only token and store it in a Secret

1. Create a minimal read-only token as in the ACL section.
2. Create a k8s Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: consul-traefik-token
  namespace: kube-system
type: Opaque
stringData:
  CONSUL_HTTP_TOKEN: "s.xxxxxxxxx-traefik"
```

3. Inject into Traefik Deployment env:

```yaml
env:
- name: CONSUL_HTTP_TOKEN
  valueFrom:
    secretKeyRef:
      name: consul-traefik-token
      key: CONSUL_HTTP_TOKEN
```

---

## Consul DNS — resolving `.consul` names inside k8s and on VMs

Consul exposes DNS on port **8600** by default. The DNS zone commonly used is `.consul`. Use it for `service-name.service.consul` or similar.

### Inside Kubernetes: CoreDNS stub/forward to Consul DNS

Edit the CoreDNS ConfigMap (`kube-system` namespace) to forward `.consul` queries to the Consul client service. Typical steps:

1. Get the cluster IP or DNS name of the consul client DNS service. The Consul Helm chart usually creates a headless `consul-client` service; DNS port is 8600. Using the service name works.

2. Patch CoreDNS (`ConfigMap` named `coredns`):

```bash
kubectl -n kube-system edit configmap coredns
```

Add a `consul` section. Example Corefile snippet to append:

```
    # Add Consul DNS forwarding zone
    consul.local:53 {
        errors
        cache 30
        forward . consul-client.consul.svc.cluster.local:8600
    }
```

A more common method is to add an entry for the `.consul` domain:

```
consul:53 {
    errors
    cache 30
    forward . consul-client.consul.svc.cluster.local:8600
}
```

**Example full Corefile excerpt** (location depends on your CoreDNS distribution):

```text
.:53 {
    errors
    health
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
       pods verified
       fallthrough in-addr.arpa ip6.arpa
    }
    prometheus :9153
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}

consul:53 {
    errors
    cache 30
    forward . consul-client.consul.svc.cluster.local:8600
}
```

After update, restart CoreDNS (if not automatically reloaded). The effect: pods can `nslookup web.service.consul` and CoreDNS will forward to Consul.

**Test inside a pod**:

```bash
kubectl run -it --rm --restart=Never busybox --image=busybox -- nslookup inventory.service.consul
```

### On VMs: use a local Consul agent as DNS resolver or configure OS resolver

Two common options:

1. **Run Consul agent locally** (recommended): The VM’s local Consul agent listens on `127.0.0.1:8600` for DNS. Configure OS resolver (or `dnsmasq`) to forward `.consul` queries to `127.0.0.1#8600`. This avoids changing global resolv.conf.

2. **System-level direct resolver** (if no local agent): Add Consul client node (on a reachable IP) as a nameserver for domain `.consul` using `dnsmasq` or `systemd-resolved` split DNS.

Example `/etc/resolv.conf` (not recommended to directly edit on systems using network manager):

```text
# Not recommended to replace primary nameserver; better use dnsmasq
nameserver 127.0.0.1
search localdomain
```

**dnsmasq example**:

Add to `/etc/dnsmasq.d/10-consul`:

```
# forward .consul to local consul agent
server=/consul/127.0.0.1#8600
```

Restart dnsmasq: `sudo systemctl restart dnsmasq`.

**Examples caution:** Avoid overriding system DNS for all queries — only forward the `.consul` domain to Consul.

---

## Traefik + Ingress + Consul: use-cases & examples

You asked: *"in our traefik setup which we have as ingress controller, how can we use consul inside ingress routes?"* — two models:

1. **Traefik discovers services via ConsulCatalog**: you tag Consul services (VM or k8s-exported) with Traefik-style tags. Traefik builds routers and services from these tags (no k8s Ingress needed for VM services).
2. **Classic K8s Ingress**: Keep using k8s Ingress for K8s services. Consul Controller continues to sync services to Consul for mesh use and for cross-environment discovery, but Traefik uses k8s provider for K8s services.

If you want a **single place to define routes** and support **VM services** on the same Traefik, prefer the ConsulCatalog provider and define routes as tags on Consul services, examples shown earlier.

### Example: Expose a VM `inventory` service via Traefik using Consul tags

1. VM registers service `inventory` in Consul with tags:

```json
"tags": [
  "traefik.http.routers.inventory.rule=Host(`inventory.shop.example`)",
  "traefik.http.services.inventory.loadbalancer.server.port=80",
  "traefik.http.routers.inventory.tls=true"
]
```

2. Traefik Consul provider picks this up and creates a router `inventory` and binds a service to the discovered backend servers.

3. Traefik will perform load balancing and TLS termination (if configured) and route to the VM.

**Note:** For HTTP(S) ingress with path-based rules, use tags like `traefik.http.routers.inventory.rule=Host(`inventory.shop.example`) && PathPrefix(`/api`)` etc.

---

## Additional helpful patterns

### Using Consul Service Intentions (authorize service-to-service)

Create intentions to allow/deny calls among services:

```bash
# deny by default, then allow specific
consul intention create -deny -service "checkout" -source "unknown"
# allow traefik -> checkout
consul intention create -allow -service "checkout" -source "traefik"
```

When using ACLs, create tokens for the services and annotate or mount them into pods.

### Using Consul as the single service catalog in a multi-datacenter setup

* Use WAN federation when you need multi-dc.
* Use `replication` and data replication for KV and config entries where necessary.

---

## Troubleshooting & operational tips

* If sidecar traffic is blocked: check Cilium policies, allow pod-internal flows and Envoy <-> peer flows.
* If Traefik sees no services in Consul: check ACL token, check Consul service tags, check network reachability from Traefik pod to `consul-client` service on port 8500.
* `consul connect env <service>` and `consul connect proxy` can be useful to locally debug proxying.
* Use `consul monitor` to watch events and health checks in real-time.
* Watch Consul Server logs for RAFT election messages if cluster instability occurs.

---

## Appendix — Checklist before applying this in production

* [ ] Run Consul servers with persistent volumes, PDBs, and correct resource requests.
* [ ] Enable ACLs and rotate tokens using automation (don't keep management token in plaintext).
* [ ] Use TLS and proper certificate management if exposing Consul APIs publicly.
* [ ] Ship Traefik a least-privilege read-only token scoped to what it needs in Consul (catalog + config entries).
* [ ] Confirm CoreDNS forwarding `.consul` properly and that there are no circular DNS loops.
* [ ] Add monitoring (Prometheus + Grafana) for Consul metrics, Envoy metrics, Traefik metrics.
* [ ] Automate Consul config entries as code (GitOps) — either `consul config write` via CI pipelines or CRDs in k8s.

---

## Quick reference commands

```bash
# ACL bootstrap
consul acl bootstrap

# Create a policy & token
consul acl policy create -name traefik-policy -rules @traefik-policy.hcl
consul acl token create -description "Traefik token" -policy-name traefik-policy

# Write a service-splitter
consul config write service-splitter-checkout.hcl

# Put/get kv
consul kv put shop/feature/checkout_v2 true
consul kv get -raw shop/feature/checkout_v2

# List services
consul catalog services

# Create a ServiceSplit CRD (k8s)
kubectl apply -f service-split.yaml

# Check Intentions
consul intention list
```

---

