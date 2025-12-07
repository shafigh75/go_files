Below is **Part 2 Cheat Sheet: KUBECTL NINJA** — a full-blown, production-grade, real-world, *DevOps-only* collection of commands, troubleshooting flows, patching tricks, recovery techniques, and dirty hacks that senior SRE/DevOps engineers use daily.

This is **not** a kubectl commands list.
This is **the cheat sheet senior engineers keep in their drawer**.

---

# **PART 2 — KUBECTL NINJA

The Ultimate DevOps Command-Line & Troubleshooting Cheat Sheet**

This cheat sheet covers:

✓ **kubectl mastery** (core + advanced + hidden tricks)
✓ **Force editing, patching, replacing objects**
✓ **Debug phases for broken nodes, pods, CNI, DNS, API server**
✓ **Direct node recovery using kubelet & static pod manipulation**
✓ **Force-rebuild control plane with static manifests**
✓ **Live debugging of workloads**
✓ **Cluster-wide search, diff, grep**
✓ **Fixing Terminating pods / stuck PVCs**
✓ **Mastering labels, selectors, field selectors**
✓ **Editing, restarting, reloading, scaling**
✓ **API server & etcd manual recovery using kubectl**
✓ **Advanced cleanup commands**

---

# 🔥 **SECTION A — kubectl Essentials (But Expert-Grade)**

## **A.1 Use kubectl like a CLI Ninja**

### **Auto-complete + aliases**

```
source <(kubectl completion bash)
alias k=kubectl
complete -F __start_kubectl k
```

### **Use short forms**

```
k get po
k get deploy
k get no
k get ns
```

### **Watch in real time**

```
k get pods -A -w
k get nodes -w
```

### **Wide mode (always use it)**

```
k get pods -o wide
```

---

## **A.2 Context/namespace switching**

```
k config get-contexts
k config use-context dev
k config set-context --current --namespace=backend
```

---

# 🔥 **SECTION B — Brutal Debugging & Inspection**

## **B.1 Describe EVERYTHING**

```
k describe pod mypod
k describe node worker1
k describe deploy mydeploy
k describe svc mysrv
```

---

## **B.2 Logs — expert mode**

### **Last 100 lines**

```
k logs mypod --tail 100
```

### **Follow logs**

```
k logs -f mypod
```

### **Logs for specific container**

```
k logs -f mypod -c sidecar
```

### **Previous crashed container**

```
k logs mypod -c app --previous
```

---

## **B.3 Exec inside pods**

```
k exec -it mypod -- bash
```

```
k exec -it mypod -- sh
```

---

## **B.4 Debug a pod using ephemeral container**

```
k debug pod/mypod -it --image=busybox --target=app
```

---

# 🔥 **SECTION C — Creation, Apply, Replace, Force**

## **C.1 Create and edit objects**

```
k create deploy nginx --image=nginx
k edit deploy nginx
```

---

## **C.2 Replace (idempotent + powerful)**

```
k replace -f deployment.yaml
```

---

## **C.3 Force replace (useful when fields are immutable)**

```
k replace --force -f deployment.yaml
```

### **OR delete immediately & reapply**

```
k delete -f deployment.yaml --force --grace-period=0
k apply -f deployment.yaml
```

---

## **C.4 Patch (the ultimate DevOps weapon)**

### **Add environment variable**

```
k patch deploy myapp \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"app","env":[{"name":"DEBUG","value":"true"}]}]}}}}'
```

### **Scale replicas via patch**

```
k patch deploy nginx -p '{"spec":{"replicas":5}}'
```

### **Add annotation to force rollout**

```
k patch deploy myapp -p \
'{"spec":{"template":{"metadata":{"annotations":{"roll":"'"$(date +%s)"'"}}}}}'
```

---

# 🔥 **SECTION D — Rollouts, Restarts, History**

## **D.1 Restart**

```
k rollout restart deploy/nginx
```

## **D.2 History**

```
k rollout history deploy nginx
k rollout undo deploy nginx
```

---

# 🔥 **SECTION E — Node-Level Operations (Advanced)**

## **E.1 Cordon & drain**

```
k cordon worker2
k drain worker2 --ignore-daemonsets --delete-emptydir-data
```

## **E.2 Uncordon**

```
k uncordon worker2
```

---

## **E.3 Check kubelet**

```
systemctl status kubelet
journalctl -u kubelet -f
```

---

## **E.4 Restart kubelet**

```
systemctl restart kubelet
```

---

# 🔥 **SECTION F — Static Pods (Where DevOps Magic Lives)**

On control plane nodes:

**Static pods live at:**

```
/etc/kubernetes/manifests/
```

These include:

* kube-apiserver.yaml
* kube-controller-manager.yaml
* kube-scheduler.yaml
* etcd.yaml

## **F.1 Force rebuild control plane pods**

Move the manifest temporarily:

```
mv /etc/kubernetes/manifests/kube-apiserver.yaml /root/
systemctl restart kubelet
```

➡ kubelet will **delete the apiserver** immediately.
Move it back:

```
mv /root/kube-apiserver.yaml /etc/kubernetes/manifests/
systemctl restart kubelet
```

➡ kubelet will **recreate the pod with fresh config/certs**.

This trick is used for:

✓ Renewed certs
✓ Changed SANs
✓ Correcting wrong command args
✓ Fixing broken images

---

# 🔥 **SECTION G — Certificates Ninja Mode**

## **G.1 View certificate expiration**

```
k get csr
kubeadm certs check-expiration
```

---

## **G.2 Renew ALL certs**

```
kubeadm certs renew all
systemctl restart kubelet
```

---

## **G.3 Regenerate apiserver cert with new IP or domain**

```
kubeadm init phase certs apiserver \
 --apiserver-advertise-address NEW-IP \
 --apiserver-cert-extra-sans NEW-IP,NEW-DOMAIN
```

Then **force rebuild**:

```
mv /etc/kubernetes/manifests/kube-apiserver.yaml /root/
mv /root/kube-apiserver.yaml /etc/kubernetes/manifests/
systemctl restart kubelet
```

---

# 🔥 **SECTION H — Stuck Pods, PVCs, Finalizers**

## **H.1 Force delete terminating pod**

```
k delete pod mypod --grace-period=0 --force
```

---

## **H.2 Remove finalizers**

Stuck PVC/namespace?

```
k get pvc mypvc -o yaml > pvc.yaml
```

Remove:

```
metadata:
  finalizers: []
```

Apply:

```
k apply -f pvc.yaml
```

---

## **H.3 Delete stuck namespace**

```
k get ns <name> -o json > ns.json
```

Remove finalizers.
Then:

```
k replace --raw "/api/v1/namespaces/<name>/finalize" -f ns.json
```

---

# 🔥 **SECTION I — Debugging Networking, DNS, CNI**

## **I.1 DNS test**

```
k run -it --rm debug --image=busybox -- nslookup kubernetes.default
```

---

## **I.2 Network test**

```
k run -it --rm ping --image=busybox -- ping 8.8.8.8
```

---

## **I.3 Check CNI**

```
ls /etc/cni/net.d/
```

```
k get pods -n kube-system | grep calico
k get pods -n kube-system | grep cilium
```

---

## **I.4 Node local routing**

```
ip route
iptables -L -v
```

---

# 🔥 **SECTION J — ETCD Debugging via kubectl**

## **Check etcd pod**

```
k get pods -n kube-system | grep etcd
```

---

## **Exec into etcd pod**

```
k exec -it etcd-cp1 -n kube-system -- sh
```

---

## **ETCD endpoint health**

```
ETCDCTL_API=3 etcdctl endpoint health
```

---

# 🔥 **SECTION K — Cluster-Wide Searches**

## **K.1 Search for all pods with label**

```
k get pods -A -l app=myapp
```

---

## **K.2 Field selectors**

```
k get pods --field-selector=status.phase=Pending
```

```
k get nodes --field-selector=spec.unschedulable=true
```

---

## **K.3 Grep pods by name**

```
k get pods -A | grep kafka
```

---

# 🔥 **SECTION L — YAML Ninja Mode**

## **L.1 Output YAML for any object**

```
k get deploy nginx -o yaml
```

---

## **L.2 Dry-run to generate YAML**

```
k create deploy myapp --image=nginx -o yaml --dry-run=client > myapp.yaml
```

---

# 🔥 **SECTION M — Exec on Nodes (DaemonSets for emergency)**

When cluster networking is broken and kubectl exec won’t work…

Create a diagnostic DaemonSet:

```
k apply -f https://k8s.io/examples/admin/daemonset.yaml
```

Exec into any node:

```
k exec -it ds/debug -- bash
```

Now you have **root shell on every worker node**.

---

# 🔥 **SECTION N — Fixing Common Control Plane Issues**

## **N.1 API server is down**

Check container logs:

```
crictl logs $(crictl ps | grep apiserver)
```

---

## **N.2 kubelet not pulling static pods**

```
journalctl -u kubelet -f
```

---

## **N.3 etcd corruption**

Restore snapshot:

```
ETCDCTL_API=3 etcdctl snapshot restore backup.db
```

---

# 🔥 **SECTION O — Best Commands for Daily Operations**

### **Create namespace**

```
k create ns prod
```

### **Label nodes**

```
k label node worker1 type=high-mem
```

### **Taint nodes**

```
k taint node worker1 key=value:NoSchedule
```

### **Remove taint**

```
k taint node worker1 key:NoSchedule-
```

### **Run temporary pod**

```
k run tmp --rm -it --image=busybox -- sh
```

### **Show all events**

```
k get events -A --sort-by=.metadata.creationTimestamp
```

### **Edit live config**

```
k edit cm/myconfig -n backend
```

---

# 🔥 Summary

This cheat sheet now covers **every production-grade kubectl action** that senior DevOps/SREs do:

✔ Patch, force-replace, rollout control
✔ Debug nodes, pods, DNS, CNI
✔ Manipulate static pods to rebuild control plane
✔ Certificate regeneration & IP migration
✔ Kill stuck pods/PVCs/namespaces
✔ Node draining & recovery
✔ Live debugging with ephemeral containers
✔ Merging contexts, namespaces, clusters
✔ ETCD diagnosis
✔ Every day-to-day kubectl command

---

# 🤷‍♂️ question:  does this cause restart?
```bash
k patch deploy nginx -p '{"spec":{"replicas":5}}'
```

What kubectl patch deploy nginx -p '{"spec":{"replicas":5}}' Does :

This command directly modifies the replicas field in the spec of your nginx Deployment object. 
It tells the Deployment controller: "I want the desired state of this application to be 5 running pods." 
The behavior then depends on whether you are scaling up or down. 

Scenario 1: Scaling Up (e.g., from 3 to 5 replicas) 
  
- The Deployment controller sees the desired number of replicas is now 5. 
- It checks the ReplicaSet it manages, which currently has 3 pods. 
- The ReplicaSet controller sees a discrepancy: it needs 2 more pods to reach the desired state. 
- It instructs Kubernetes to create 2 new pods based on the pod template. 
- The original 3 pods are completely untouched. They continue to run and serve traffic without any interruption. 

Result: You go from 3 running pods to 5 running pods. No restart occurs. 

---

Scenario 2: Scaling Down (e.g., from 5 to 3 replicas) 

- The Deployment controller sees the desired number of replicas is now 3. 
- It checks its ReplicaSet, which currently has 5 pods. 
- The ReplicaSet controller sees it has 2 too many pods. 
- It instructs Kubernetes to terminate 2 pods. Kubernetes will select which pods to terminate (often the newest ones first, but you can influence this with pod-deletion-cost). 
- The remaining 3 pods are completely untouched. 

Result: You go from 5 running pods to 3 running pods. No restart of the remaining pods occurs. 
What Does Cause a Restart (Rolling Update)? 

---

A "restart" in Kubernetes terms is usually a rolling update. This happens when you make a change to the Pod Template (spec.template) of the Deployment. 

When you change the pod template, Kubernetes doesn't just modify the existing pods. It can't. Pods are largely immutable once created. Instead, it does the following: 

- Creates a new ReplicaSet with the updated pod template. 
- Starts scaling up the new ReplicaSet (creating new pods). 
- Starts scaling down the old ReplicaSet (terminating old pods) in a controlled, gradual fashion to ensure there's no downtime. 

This is called a rolling update. 

Common changes that trigger a rolling update: 

- Changing the container image: `kubectl set image deployment/nginx nginx=nginx:1.23`
- Adding or updating an environment variable: `kubectl patch deploy nginx -p '{"spec":{"template":{"spec":{"containers":[{"name":"nginx","env":[{"name":"DEBUG","value":"true"}]}]}}}}'`
- Changing resource requests/limits.
- Explicitly triggering a restart: `kubectl rollout restart deployment/nginx` . This command works by adding a restart annotation to the pod template, which changes its hash and forces a rolling update.
     
