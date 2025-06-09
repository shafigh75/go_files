Absolutely! Here's a **step-by-step guide to installing Cilium as the CNI (Container Network Interface)** in your Kubernetes cluster and setting up **Cilium Ingress Controller** (using Cilium’s native ingress support) with a full working example.

---

# 🧩 Overview

This guide will:
- Set up a Kubernetes cluster using `kubeadm` (or assume you already have one).
- Install **Cilium as the CNI**.
- Enable and configure **Cilium Ingress Controller**.
- Deploy a sample app and expose it via Ingress.

> ✅ This works on cloud providers (AWS, GCP, Azure), bare metal, or local VMs.

---

## 🚀 Step 1: Prerequisites

### ✅ Hardware/Software Requirements
- One or more nodes (control-plane + workers)
- Docker or containerd installed
- `kubeadm`, `kubectl`, `kubelet` installed
- Root access or sudo privileges

### 🔒 Disable Swap
```bash
sudo swapoff -a
sudo sed -i '/swap/d' /etc/fstab
```

### 🔐 Enable Kernel Modules
```bash
cat <<EOF | sudo tee /etc/modules-load.d/cilium.conf
overlay
br_netfilter
EOF

sudo modprobe overlay
sudo modprobe br_netfilter

# Ensure sysctl params are set
cat <<EOF | sudo tee /etc/sysctl.d/kubernetes.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
EOF

sudo sysctl --system
```

---

## 🌐 Step 2: Initialize Kubernetes Cluster

If you're building a new cluster:

```bash
sudo kubeadm init --pod-network-cidr=10.0.0.0/16
```

> ⚠️ Do **not** install another CNI after this — we'll use Cilium.

Set up `kubectl` access:

```bash
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

---

## 🐚 Step 3: Install Cilium CLI

```bash
curl -L --remote-name-all https://github.com/cilium/cilium-cli/releases/latest/download/cilium-linux-amd64.tar.gz{,.sha256}
sha256sum --check cilium-linux-amd64.tar.gz.sha256
sudo tar xzvf cilium-linux-amd64.tar.gz -C /usr/local/bin
rm cilium-linux-amd64.tar.gz*
```

---

## 🧱 Step 4: Install Cilium as CNI

We’ll use `cilium install` command which handles everything:

```bash
cilium install \
  --version 1.15.7 \
  --config tunnel=disabled \
  --config nativeRoutingCIDR=10.0.0.0/16
```

> Replace `1.15.7` with the version you want (e.g., `1.16.0`)  
> `tunnel=disabled` enables **direct routing**, suitable for single subnet environments.

> If you're using multiple subnets or cloud provider, enable tunneling:
```bash
--config tunnel=vxlan
```

### 🔍 Check Installation Status
```bash
cilium status --wait
```

You should see:
```
Kubernetes:             OK
KubeProxyReplacement:   OK
CiliumHealth:           OK
Controller statuses:    8/8 healthy
...
```

All pods should be running:
```bash
kubectl get pods -n kube-system -l k8s-app=cilium
```

---

## 🌐 Step 5: Enable Cilium Ingress Controller

Cilium includes an **Envoy-based Ingress Controller** called `cilium-ingress`.

### 📦 Enable Ingress Controller

```bash
cilium hubble enable --ui
cilium ingress enable
```

> This installs:
- Envoy proxy pods
- `IngressClass` named `cilium`
- Required CRDs and services

Check if `IngressClass` is created:
```bash
kubectl get ingressclass
```

Output:
```
NAME      CONTROLLER
cilium    io.cilium/cilium-operator
```

---

## 🧪 Step 6: Deploy Sample App and Expose via Ingress

Let’s deploy a simple Nginx app and expose it through Cilium Ingress.

### 📄 File: `nginx-deployment.yaml`
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
```

### 📄 File: `nginx-service.yaml`
```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
```

### 📄 File: `nginx-ingress.yaml`
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nginx-ingress
  annotations:
    ingress.cilium.io/external-ips: "192.168.1.10" # Optional: specify IP
spec:
  ingressClassName: cilium
  rules:
  - http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx
            port:
              number: 80
```

Apply all manifests:

```bash
kubectl apply -f nginx-deployment.yaml
kubectl apply -f nginx-service.yaml
kubectl apply -f nginx-ingress.yaml
```

---

## 🧭 Step 7: Access Your Ingress

### Get Ingress IP
```bash
kubectl get ingress
```

Output:
```
NAME            CLASS   HOSTS   ADDRESS         PORTS   AGE
nginx-ingress   cilium  *       192.168.1.10    80      2m
```

Now visit: `http://192.168.1.10` in browser or test with:

```bash
curl http://192.168.1.10
```

You should see the default Nginx welcome page.

---

## 🛡️ Step 8: Optional – TLS Support with Cert-Manager

To add HTTPS:

1. Install cert-manager:
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

2. Create `ClusterIssuer` and add TLS stanza to your Ingress:
```yaml
tls:
- hosts:
  - myapp.example.com
  secretName: tls-secret
```

See [cert-manager docs](https://cert-manager.io/docs/) for Let's Encrypt integration.

---

## 🧰 Additional Useful Commands

### View Envoy proxies
```bash
kubectl get pods -n kube-system -l k8s-app=cilium-envoy
```

### View Hubble UI (optional but helpful)
```bash
cilium hubble enable --ui
kubectl port-forward svc/hubble-ui -n kube-system 8081
```

Open [http://localhost:8081](http://localhost:8081)

---

## 📝 Summary Checklist

| Task | Status |
|------|--------|
| ✅ Kubernetes cluster initialized | ✔️ |
| ✅ Cilium CLI installed | ✔️ |
| ✅ Cilium installed as CNI | ✔️ |
| ✅ Cilium Ingress enabled | ✔️ |
| ✅ Sample app deployed and exposed via Ingress | ✔️ |
| ✅ Ingress accessible via IP | ✔️ |

---

## 📚 References

- [Cilium Docs](https://docs.cilium.io/)
- [Cilium Ingress Docs](https://docs.cilium.io/en/stable/network/servicemesh/ingress-controller/)
- [Cilium CLI GitHub](https://github.com/cilium/cilium-cli)

---



Great question! Let's dive into **how Cilium Ingress works**, what gets exposed, and how traffic flows from the outside world to your Kubernetes services.

---

## 🧠 What Gets Exposed as Cilium Ingress?

### ✅ Short Answer:

Cilium Ingress exposes a **Kubernetes Ingress resource** using an **Envoy-based proxy** that runs as DaemonSet pods on your nodes. By default, it binds to port `80` and `443` on each node's IP address.

So yes — **the IPs of the nodes running the Cilium Ingress pods are what get exposed** to external traffic.

---

## 🌐 How Does Cilium Ingress Handle Incoming Traffic?

Here’s a breakdown of the flow:

```
[Client] → [NodeIP:80/443] → [Cilium Ingress (Envoy)] → [Service → Pod]
```

### 🔍 Step-by-step Flow

1. **Client sends request to NodeIP:80**
   - Example: `curl http://<node-ip>/myapp`

2. **Request hits Cilium-managed iptables or eBPF rules**
   - These redirect traffic to the local **Envoy proxy** pod (part of Cilium Ingress)

3. **Envoy Proxy (Ingress Controller)**
   - Evaluates Host/path rules defined in the `Ingress` resource
   - Routes traffic to the appropriate backend Kubernetes Service

4. **Traffic is load balanced across Pods**
   - Envoy proxies the request to the corresponding service/pod based on routing rules

---

## 📦 Components Involved

| Component | Description |
|----------|-------------|
| `IngressClass` | Defines which controller handles the Ingress (`cilium`) |
| `Ingress` | Standard Kubernetes Ingress object with host/path rules |
| `Envoy` | Runs as a DaemonSet pod on nodes where ingress traffic is expected |
| `CiliumEnvoyConfig` | Optional CRD to customize Envoy config |
| `Cilium Ingress Backend` | Internal mapping between Ingress and Services |

---

## 🧱 Where Is Ingress Exposed?

By default, **Cilium Ingress listens on all node IPs**, on ports `80` and `443`.

This means:
- Any node in your cluster can receive traffic for any Ingress rule.
- You can point your DNS or LoadBalancer to **any worker node IP**.

> ⚠️ This assumes your nodes have public IPs or are behind a shared firewall/load balancer.

---

## 🛠 Optional: Use Specific IPs or Hosts

You can restrict which IP(s) the Ingress binds to by using annotations:

```yaml
annotations:
  ingress.cilium.io/external-ips: "192.168.1.10"
```

Or specify multiple IPs:

```yaml
ingress.cilium.io/external-ips: "192.168.1.10,192.168.1.11"
```

This tells Cilium to bind only to those IPs instead of all interfaces.

---

## 🔄 How Does Cilium Handle TLS?

Cilium supports TLS termination via standard Kubernetes Ingress fields:

```yaml
spec:
  tls:
  - hosts:
    - example.com
    secretName: tls-secret
```

Just provide a valid TLS secret with certificate and key, and Cilium will terminate TLS at the Ingress level.

You can also use tools like **cert-manager** to automate certificate provisioning.

---

## 🧪 How to Test It

After deploying an Ingress:

```bash
kubectl get ingress
```

Output:
```
NAME            CLASS   HOSTS   ADDRESS         PORTS   AGE
nginx-ingress   cilium  *       192.168.1.10    80      5m
```

Now test:

```bash
curl -v http://192.168.1.10
```

If you're not on the same network, ensure the node is reachable and ports `80`/`443` are open.

---

## 🧩 Advanced Setup: Using MetalLB or Cloud LB

To get a **shared IP address** for Ingress (like a traditional LoadBalancer), you can combine Cilium Ingress with:

### 1. **MetalLB** (for bare-metal):
- Assigns a virtual IP (VIP) that routes to Ingress nodes
- Works seamlessly with Cilium

### 2. **Cloud Provider LB** (AWS/GCP/Azure):
- If you're on cloud provider, consider using Cilium with native LB integration or use a Service of type `LoadBalancer` to front Ingress

Example:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: cilium-ingress
  namespace: kube-system
spec:
  type: LoadBalancer
  selector:
    k8s-app: cilium-envoy
  ports:
    - name: http
      port: 80
      targetPort: 80
    - name: https
      port: 443
      targetPort: 443
```

This gives you a single public IP for your Ingress.

---

## 📊 Summary Table

| Feature | Cilium Ingress Behavior |
|--------|-------------------------|
| Exposes | Node IPs (by default) |
| Ports | `80`, `443` |
| Ingress Controller Type | Envoy-based |
| TLS Support | Yes, via standard Ingress spec |
| Host/path routing | Yes |
| Custom IPs | Supported via annotation |
| Scalability | Scales per-node (DaemonSet) |
| Integration | Works with cert-manager, MetalLB, etc. |

---

## ✅ Final Notes

- Cilium Ingress is lightweight and tightly integrated with eBPF and Cilium’s dataplane.
- No need for extra controllers like NGINX Ingress; everything runs inside Cilium.
- Ideal for environments where simplicity, performance, and observability are key.

---




Absolutely! Let's bring everything together and walk through a **production-grade setup** using:

- **Cilium as CNI**
- **Cilium Ingress Controller**
- **MetalLB for Load Balancing**
- Exposing your Kubernetes services via a **dedicated external IP range**

---

# 🧱 Production-Grade Setup: Cilium + MetalLB + Ingress

We'll assume:
- You have a basic cluster with Cilium installed.
- You're on **bare metal** or private cloud (no native LB like AWS ELB).
- You want to expose HTTP/HTTPS traffic via a shared VIP (virtual IP).

---

## 🔧 Step 1: Install MetalLB

### 📦 Add the MetalLB Helm Repository

```bash
helm repo add metallb https://metallb.github.io/metallb
helm repo update
```

### 📄 Create MetalLB ConfigMap (Layer2 Mode)

Create a file called `metallb-config.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - 192.168.1.200-192.168.1.250
```

This config:
- Uses Layer2 mode (simplest for most networks)
- Allocates IPs from `192.168.1.200` to `192.168.1.250` for LoadBalancer services

> ⚠️ Replace this IP range with one that matches your network.

### 🚀 Deploy MetalLB Using Helm

```bash
kubectl create namespace metallb-system

helm install metallb metallb/metallb --namespace metallb-system
kubectl apply -f metallb-config.yaml
```

---

## 🔌 Step 2: Configure Cilium Ingress Service to Use MetalLB

By default, Cilium’s Envoy pods run as a DaemonSet but don't get an external IP unless we tell Kubernetes to assign one.

We will now patch the Cilium Ingress Service to be of type `LoadBalancer`.

### 📝 Get the Current Ingress Service

```bash
kubectl get svc -n kube-system cilium-ingress -o yaml > cilium-ingress-svc.yaml
```

Edit the file and change `.spec.type` to `LoadBalancer`:

```yaml
spec:
  type: LoadBalancer
  selector:
    k8s-app: cilium-envoy
  ports:
    - name: http
      port: 80
      targetPort: 80
    - name: https
      port: 443
      targetPort: 443
```

Apply the updated service:

```bash
kubectl apply -f cilium-ingress-svc.yaml
```

Now check the service:

```bash
kubectl get svc -n kube-system cilium-ingress
```

Output:
```
NAME              TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)                      AGE
cilium-ingress    LoadBalancer   10.96.120.10   192.168.1.200    80:30432/TCP,443:30433/TCP   5m
```

🎉 You now have a single **external IP** (`192.168.1.200`) assigned by MetalLB that routes to your Cilium Ingress.

---

## 🧪 Step 3: Test Your Ingress

Let’s deploy a simple app and expose it via Ingress:

### 📁 File: `nginx-deploy.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
```

### 📁 File: `nginx-service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  selector:
    app: nginx
  ports:
    - port: 80
      targetPort: 80
```

### 📁 File: `nginx-ingress.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nginx-ingress
spec:
  ingressClassName: cilium
  rules:
  - http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx
            port:
              number: 80
```

Apply them:

```bash
kubectl apply -f nginx-deploy.yaml
kubectl apply -f nginx-service.yaml
kubectl apply -f nginx-ingress.yaml
```

Now test:

```bash
curl http://192.168.1.200
```

You should see the Nginx welcome page.

---

## 🔐 Optional: TLS Ingress with Cert-Manager

If you'd like HTTPS support:

### 1. Install Cert-Manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

### 2. Create ClusterIssuer (e.g., SelfSigned or Let's Encrypt)

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-cluster-issuer
spec:
  selfSigned: {}
```

Apply:
```bash
kubectl apply -f issuer.yaml
```

### 3. Update Ingress with TLS

```yaml
spec:
  tls:
  - hosts:
    - example.com
    secretName: tls-secret
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx
            port:
              number: 80
```

Apply and test:

```bash
curl -k https://example.com
```

(Use `-k` if using self-signed cert.)

---

## 🧰 Useful Commands

### View External IPs Assigned
```bash
kubectl get svc -n kube-system cilium-ingress
```

### Check Ingress Pod IPs
```bash
kubectl get pods -n kube-system -l k8s-app=cilium-envoy -o wide
```

### View MetalLB Speaker Logs
```bash
kubectl logs -n metallb-system -l app=metallb,component=speaker
```

---

## 📊 Summary Table

| Component | Role |
|----------|------|
| Cilium | CNI, eBPF dataplane, Ingress controller |
| Envoy | Runs as Ingress proxy |
| MetalLB | Provides LoadBalancer IPs for ingress |
| IngressClass | Routes traffic via Cilium |
| Service (LoadBalancer) | Gets public IP from MetalLB pool |

---

## ✅ Final Checklist

| Task | Status |
|------|--------|
| ✅ Cilium Installed | ✔️ |
| ✅ MetalLB Installed & Configured | ✔️ |
| ✅ Cilium Ingress Enabled | ✔️ |
| ✅ Ingress Service Type Changed to LoadBalancer | ✔️ |
| ✅ Sample App Deployed and Exposed | ✔️ |
| ✅ TLS Support (Optional) | ✔️ |
| ✅ DNS Points to External IP | ✔️ |

---

## 📚 References

- [Cilium Ingress Docs](https://docs.cilium.io/en/stable/network/servicemesh/ingress-controller/)
- [MetalLB Docs](https://metallb.universe.tf/)
- [Cert-Manager Docs](https://cert-manager.io/docs/)

---


