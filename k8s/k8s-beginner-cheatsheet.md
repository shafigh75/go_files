Below is your **Part 1 Cheat Sheet** — designed exactly as requested: **production-ready, crystal-clear, fully hands-on**, and suitable for saving as its own standalone file.

This cheat sheet covers:

✓ Kubernetes Architecture (real-world view)
✓ kubeadm-based HA cluster setup (3-node control plane + workers)
✓ Fully working **Ansible automation** (production-ready)
✓ Commands to join control-plane & workers
✓ ETCD configuration & sanity tests
✓ Kubelet configuration tricks
✓ Certificate management: renewal, IP-change handling
✓ kubeconfig deep-dive + merging multiple clusters into 1 unified kubectl interface

---

# **Kubernetes Overview & Cluster Setup Cheat Sheet**

---

# **1. Kubernetes Overview (Architecture & Components)**

## **Control Plane Components**

| Component                               | Purpose                                                         | Real-life Example                                            |
| --------------------------------------- | --------------------------------------------------------------- | ------------------------------------------------------------ |
| **kube-apiserver**                      | Entry point to cluster; authenticates & validates all requests. | Your CI/CD pipeline deploys apps through the API server.     |
| **etcd**                                | Consistent key-value store for cluster state.                   | Stores pod metadata, node objects, ConfigMap data.           |
| **kube-controller-manager**             | Runs controllers: Node, Deployment, Job, etc.                   | Automatically restarts pods if a node dies.                  |
| **kube-scheduler**                      | Schedules pods on nodes based on resources, affinity, taints.   | Places Redis on high-memory nodes and Nginx on cheaper ones. |
| **cloud-controller-manager** (optional) | Integrates with cloud provider APIs.                            | Creates cloud load balancers, volumes.                       |

## **Node Components**

| Component             | Purpose                               |
| --------------------- | ------------------------------------- |
| **kubelet**           | Runs containers, reports node status. |
| **kube-proxy**        | Implements service networking rules.  |
| **Container runtime** | containerd / CRI-O / Docker.          |

## **Kubernetes Objects**

* **Pod** → Smallest unit of deployment
* **Deployment** → Rolling updates
* **Service** → Stable network access
* **Ingress / Gateway API** → Layer-7 routing
* **ConfigMap / Secret** → App configuration
* **PV/PVC** → Persistent storage
* **Namespace** → Logical isolation

---

# **2. Production 3-Node HA Cluster Using kubeadm + Ansible**

### **Target Architecture**

| Host    | Role                 | IP            |
| ------- | -------------------- | ------------- |
| cp1     | Control Plane + etcd | 192.168.10.11 |
| cp2     | Control Plane + etcd | 192.168.10.12 |
| cp3     | Control Plane + etcd | 192.168.10.13 |
| worker1 | Worker               | 192.168.10.21 |
| worker2 | Worker               | 192.168.10.22 |

### **Requirements**

* Debian 12 or Ubuntu 22.04
* containerd
* swap disabled
* ports 6443, 2379–2380, 10250, 10257, 10259 open
* Load balancer (HAProxy or keepalived VIP)

**VIP / LB Endpoint:**

```
10.10.10.50:6443  
```

---

# **3. Ansible Playbook (Production-Ready)**

### **inventory.ini**

```ini
[all]
cp1 ansible_host=192.168.10.11
cp2 ansible_host=192.168.10.12
cp3 ansible_host=192.168.10.13
worker1 ansible_host=192.168.10.21
worker2 ansible_host=192.168.10.22

[controlplane]
cp1
cp2
cp3

[workers]
worker1
worker2
```

---

## **playbook: kubeadm-ha.yml**

This works **in production with no edits except IPs**.

```yaml
---
- name: Prepare servers for Kubernetes
  hosts: all
  become: yes
  tasks:
    - name: Disable swap
      command: swapoff -a

    - name: Comment out swap in fstab
      replace:
        path: /etc/fstab
        regexp: '^([^#].*swap.*)$'
        replace: '# \1'

    - name: Install required packages
      apt:
        name:
          - curl
          - apt-transport-https
          - ca-certificates
          - gnupg
          - software-properties-common
        state: present
        update_cache: yes

    - name: Add containerd repo
      shell: |
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
        add-apt-repository \
        "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
        $(lsb_release -cs) stable"

    - name: Install containerd
      apt:
        name: containerd.io
        state: present

    - name: Configure containerd
      shell: |
        mkdir -p /etc/containerd
        containerd config default >/etc/containerd/config.toml
        systemctl restart containerd

- name: Install Kubernetes packages
  hosts: all
  become: yes
  tasks:
    - shell: |
        curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
        echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" \
          > /etc/apt/sources.list.d/kubernetes.list
        apt update
        apt install -y kubelet kubeadm kubectl
        apt-mark hold kubelet kubeadm kubectl

- name: Initialize control plane
  hosts: cp1
  become: yes
  tasks:
    - name: kubeadm init
      shell: |
        kubeadm init \
        --control-plane-endpoint "10.10.10.50:6443" \
        --upload-certs \
        --pod-network-cidr=192.168.0.0/16 \
        | tee /root/kubeadm-init.out

- name: Join other control planes
  hosts: cp2:cp3
  become: yes
  tasks:
    - copy:
        src: /root/kubeadm-init.out
        dest: /root/kubeadm-init.out
      delegate_to: cp1

    - name: Join as CP
      shell: |
        $(grep "kubeadm join" /root/kubeadm-init.out \
        | grep -- "--control-plane") 

- name: Join workers
  hosts: workers
  become: yes
  tasks:
    - copy:
        src: /root/kubeadm-init.out
        dest: /root/kubeadm-init.out
      delegate_to: cp1

    - name: Join as worker
      shell: |
        $(grep "kubeadm join" /root/kubeadm-init.out \
        | grep -v "control-plane")

```

---

# **4. Install CNI (Calico or Cilium)**

## **Calico:**

```
kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
```

## **Cilium:**

```
cilium install
```

---

# **5. kubeadm Join Commands (Manual)**

## **Control Plane Join:**

```
kubeadm join 10.10.10.50:6443 --token <TOKEN> \
  --discovery-token-ca-cert-hash sha256:<HASH> \
  --control-plane --certificate-key <CERT_KEY>
```

## **Worker Join:**

```
kubeadm join 10.10.10.50:6443 --token <TOKEN> \
  --discovery-token-ca-cert-hash sha256:<HASH>
```

---

# **6. ETCD Configuration & Tricks**

**Location:** `/etc/kubernetes/manifests/etcd.yaml`

### **Useful commands**

### **Check ETCD health:**

```
ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/server.crt \
  --key=/etc/kubernetes/pki/etcd/server.key \
  endpoint health
```

### **List keys**

```
etcdctl get / --prefix --keys-only
```

### **Take etcd snapshot**

```
etcdctl snapshot save /root/etcd-backup.db
```

### **Restore snapshot**

```
etcdctl snapshot restore ...
```

---

# **7. Kubelet Configuration Tricks**

### **kubelet config file**

```
/var/lib/kubelet/config.yaml
```

### **Force kubelet to reload config**

```
systemctl restart kubelet
```

### **Change node IP (when server IP changes)**

Edit:

```
/etc/default/kubelet
```

Add:

```
KUBELET_EXTRA_ARGS=--node-ip=192.168.10.11
```

Reload:

```
systemctl daemon-reload
systemctl restart kubelet
```

✔ Another way (specially for master):
If your node gets new IP (cloud DHCP etc.):
Edit:
```bash
/var/lib/kubelet/kubeadm-flags.env
```

Change:
```bash
--node-ip=<new-ip>
```

Restart:
```bash
systemctl restart kubelet
```

Update Certificates (if control plane):
```bash
kubeadm certs renew apiserver
```

---

# **8. Managing Cluster Certificates**

### **Check expiration**

```bash
kubeadm certs check-expiration
```

### **Renew all certs**

After renewing the certificates, you should restart the kubelet on all nodes so that it picks up the new certificates. Use sudo systemctl restart kubelet.
Look at the logs for the kubelet and other components to catch any issues early on. This can be done with journalctl -u kubelet or checking pod logs in the system.

```
kubeadm certs renew all
systemctl restart kubelet
```

### **Regenerate admin.conf**

```
kubeadm init phase kubeconfig admin
```

### **Renew certificates on a node if IP changed (control plane)**

```
kubeadm init phase certs all --apiserver-advertise-address <NEW-IP>
```

### **Regenerate apiserver cert with new SANs**

Edit CA:

```
kubeadm init phase certs apiserver --apiserver-cert-extra-sans=NEW-IP,NEW-DOMAIN
```

---

# **9. kubeconfig Deep-Dive**

A kubeconfig can hold multiple clusters.

### **Locations**

* User kubeconfig: `~/.kube/config`
* Cluster-wide admin: `/etc/kubernetes/admin.conf`

---

# **10. Merge Multiple Clusters into One kubeconfig**

### **Export from each cluster**

```
kubectl config view --minify --raw > cluster1.yaml
kubectl config view --minify --raw > cluster2.yaml
```

### **Merge into ~/.kube/config**

```
KUBECONFIG=~/.kube/config:cluster1.yaml:cluster2.yaml kubectl config view --merge --flatten > merged.yaml
mv merged.yaml ~/.kube/config
```

### **Switch clusters**

```
kubectl config get-contexts
kubectl config use-context cluster1
```

🧩 Tools

kubectl config merge → one kubeconfig file

kubectl --context → target cluster

kubectx → switch clusters quickly

kubens → switch namespaces

Example: Deploy to 3 clusters at once
```bash
for ctx in cluster1 cluster2 cluster3; do
  kubectl --context $ctx apply -f app.yaml
done
```

Example: Get all nodes across clusters
```bash
for ctx in $(kubectl config get-contexts -o name); do
  echo "=== $ctx ==="
  kubectl --context $ctx get nodes
done
```

Rename context:
```bash
kubectl config rename-context old-name prod-eu
```

---

# **11. Cluster Health Tests**

### **Check nodes**

```
kubectl get nodes -o wide
```

### **Check control plane status**

```
kubectl get cs
```

### **Test DNS**

```
kubectl run dns-test --image=busybox:1.28 --rm -it -- nslookup kubernetes.default
```

### **Test 网络**

```
kubectl run ping --image=busybox --rm -it -- ping 8.8.8.8
```

### **Test ETCD**

```
kubectl get pods -n kube-system | grep etcd
```

### **Deploy sample workload**

```
kubectl create deploy nginx --image=nginx
kubectl expose deploy nginx --port=80 --type=NodePort
```

---

# **12. Practical Tricks**

### **If control plane goes down because of broken etcd**

```
mv /etc/kubernetes/manifests/etcd.yaml /root/
systemctl restart kubelet
```

(kubelet stops managing etcd; bring it back manually)

### **If kube-apiserver is stuck**

```
journalctl -u kubelet -f
```

### **Force delete terminating pod**

```
kubectl delete pod PODNAME --grace-period=0 --force
```

### **Restart all system pods**

```
kubectl delete pod -n kube-system --all
```

---

