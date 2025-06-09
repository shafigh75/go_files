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

Would you like me to provide:
- A Terraform setup for this?
- How to integrate with MetalLB for bare-metal load balancing?
- How to secure the ingress with authentication?

Let me know how you'd like to extend this setup!
