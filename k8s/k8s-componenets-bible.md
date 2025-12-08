# Part 3 — **Kubernetes Component Bible**

A practical, line-commented manifest collection + concise explanation for each core and advanced Kubernetes component.
Each manifest is **production-minded** (real fields, realistic defaults) with inline comments explaining what each field does and what alternatives you can use. Use these as templates: tweak image names, resource sizes, selectors, and storage class names to match your environment.

I did a focused sweep of up-to-date official docs for accuracy (Deployment, HPA, Gateway API, Ingress, kubeadm certs). Where helpful I cite them. ([Kubernetes][1])

---

> NOTE: each manifest includes explanatory comments that start with `#` (YAML comments). When you paste into a file, remove comments you don't want. I keep the most important/hidden flags explained.

---

# 1 — Pod (fundamental unit)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hello-pod              # Pod name
  labels:
    app: hello
spec:
  # restartPolicy: Always | OnFailure | Never
  restartPolicy: Always
  containers:
    - name: web
      image: nginx:1.25        # image tag should be immutable in prod (digest preferred)
      ports:
        - containerPort: 80
      env:
        - name: ENV
          value: production
      resources:
        requests:
          cpu: "100m"          # request: guarantee at least this; used by scheduler
          memory: "128Mi"
        limits:
          cpu: "500m"          # limit: hard cap; container cannot exceed this
          memory: "256Mi"
      livenessProbe:          # restarts containers that are unhealthy
        httpGet:
          path: /healthz
          port: 80
        initialDelaySeconds: 30
        periodSeconds: 10
      readinessProbe:         # controls whether pod receives traffic
        httpGet:
          path: /ready
          port: 80
        initialDelaySeconds: 5
        periodSeconds: 5
```

**Notes:** Pods are ephemeral: don't rely on pod names for persistence. Use Deployments/StatefulSets for managed lifecycles. Use image digests (`nginx@sha256:...`) in production to avoid implicit updates.

---

# 2 — ReplicaSet (low level; usually managed by Deployment)

```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: rs-hello
  labels:
    app: hello
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
        - name: web
          image: nginx:1.25
          ports:
            - containerPort: 80
```

**Notes:** ReplicaSet ensures `N` replica pods. You rarely create ReplicaSets directly—use Deployments instead so you get rolling updates, history, and rollbacks.

---

# 3 — Deployment (recommended for stateless apps)

Includes `strategy` section with RollingUpdate, and example of using annotation trick for canary/blue-green.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  labels:
    app: myapp
spec:
  replicas: 4                        # desired number of pods
  revisionHistoryLimit: 10           # how many old ReplicaSets to keep (for rollback)
  selector:
    matchLabels:
      app: myapp
  strategy:
    type: RollingUpdate              # RollingUpdate | Recreate
    rollingUpdate:
      maxSurge: 1                    # allow up to 1 extra pod during rollout (int or %)
      maxUnavailable: 1              # allow up to 1 pod unavailable during rollout
  template:
    metadata:
      labels:
        app: myapp
      annotations:
        # this annotation is a common trick to force a rolling update when you
        # update a ConfigMap or Secret by embedding a checksum (see examples).
        config-checksum: "sha256:...replace-with-calc"
    spec:
      containers:
        - name: app
          image: myregistry/myapp:1.2.3
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: "200m"
              memory: "256Mi"
            limits:
              cpu: "1000m"
              memory: "1Gi"
```

### Deployment strategies — quick guide

* **RollingUpdate (default):** Gradual replacement using `maxSurge` and `maxUnavailable`. Zero-downtime if readiness probes are configured correctly. (Use when new version is backwards compatible.) ([Kubernetes][1])
* **Recreate:** Kill all old pods then create new ones. Use when you must ensure only one version runs (legacy DB migration).
* **Canary (pattern, not native):** Deploy a separate Deployment for canary (e.g., `myapp-canary` with small `replicas=1`) and use traffic-splitting (Ingress/Gateway + weighted routing) to send a small percent to canary. Gateway API supports traffic splitting. ([Kubernetes Gateway API][2])
* **Blue-Green:** Maintain two deployments (blue/green) and swap Service selector or Gateway route to cutover. Blue-green is explicit and gives instant rollback.

---

# 4 — StatefulSet (stateful apps: stable network ID + stable storage)

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: "postgres-headless"   # headless service for stable DNS
  replicas: 3
  podManagementPolicy: OrderedReady  # OrderedReady (default) or Parallel
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:15
          ports:
            - containerPort: 5432
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes: ["ReadWriteOnce"]
        storageClassName: fast-ssd
        resources:
          requests:
            storage: 50Gi
```

**Notes:** StatefulSet gives stable identity (`pod-0`, `pod-1`), persistent storage using `volumeClaimTemplates`. For DBs, use anti-affinity to spread pods across failure domains.

---

# 5 — DaemonSet (run copy on each node or subset)

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
spec:
  selector:
    matchLabels:
      app: node-exporter
  template:
    metadata:
      labels:
        app: node-exporter
    spec:
      hostNetwork: true                  # often used for node-level telemetry
      containers:
        - name: node-exporter
          image: prom/node-exporter:1.5
          resources:
            requests:
              cpu: "50m"
              memory: "64Mi"
      tolerations:
        - key: node-role.kubernetes.io/master
          operator: Exists
          effect: NoSchedule
```

**Notes:** Use DaemonSets for node-level agents (monitoring, logging). Use `nodeSelector`/`nodeAffinity` to limit to certain nodes.

---

# 6 — Job & CronJob (batch work)

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
spec:
  backoffLimit: 3                    # retries before marking failed
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - name: migrate
          image: myregistry/db-migrate:2
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: url
```

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-report
spec:
  schedule: "0 2 * * *"             # run daily at 02:00
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: report
              image: myregistry/report:latest
```

**Notes:** CronJob uses cron schedule format; use `successfulJobsHistoryLimit` and `failedJobsHistoryLimit` to control resource churn.

---

# 7 — Service (ClusterIP / NodePort / LoadBalancer / ExternalName)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp-svc
spec:
  selector:
    app: myapp
  type: ClusterIP                  # ClusterIP | NodePort | LoadBalancer | ExternalName
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
```

**Common variants:**

* `type: NodePort` — opens port on every node (30000-32767 default range). Good for debugging or bare-metal LB.
* `type: LoadBalancer` — Cloud providers will provision an external LB. For on-prem, use MetalLB.
* `ExternalName` — DNS alias to external service.

---

# 8 — Ingress (Ingress-NGINX example with annotation hints)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: myapp-ingress
  annotations:
    kubernetes.io/ingress.class: "nginx"              # choose a controller
    nginx.ingress.kubernetes.io/proxy-body-size: "8m"
    nginx.ingress.kubernetes.io/rewrite-target: /$1
spec:
  tls:
    - hosts:
        - example.com
      secretName: example-com-tls                      # TLS secret (k8s tls secret)
  rules:
    - host: example.com
      http:
        paths:
          - path: /?(.*)
            pathType: Prefix
            backend:
              service:
                name: myapp-svc
                port:
                  number: 80
```

**Notes:** Ingress is an API that controllers (Ingress-NGINX, Traefik, contour) implement. For advanced routing, consider Gateway API.

Official Ingress docs and controller-specific annotations are key. ([Kubernetes][3])

---

# 9 — Gateway API (modern replacement for Ingress) — Gateway + HTTPRoute example

```yaml
# Gateway resource — attaches to a controller (e.g., istio, contour, nginx-gateway)
apiVersion: gateway.networking.k8s.io/v1beta1
kind: Gateway
metadata:
  name: my-gateway
spec:
  gatewayClassName: nginx-gateway-class    # provided by controller
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      routes:
        namespaces:
          from: All                       # allow routes from any namespace
---
# HTTPRoute (routes to backend service)
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: myhttproute
spec:
  parentRefs:
    - name: my-gateway
  hostnames:
    - "example.com"
  rules:
    - matches:
        - path:
            type: Prefix
            value: "/"
      backendRefs:
        - name: myapp-svc
          port: 80
          weight: 100
```

**Notes:** Gateway API provides richer features: multiple listeners, TLS options, traffic splitting (weights), header manipulation, cross-namespace routing, and matches. Use a Gateway controller (Contour, Istio Gateway API, Nginx Gateway Controller) to act on these CRDs. ([Kubernetes Gateway API][2])

---

# 10 — ConfigMap & Secret (configuration & sensitive data)

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  LOG_LEVEL: info
  FEATURE_FLAG: "true"
---
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
stringData:
  url: postgres://user:password@postgres:5432/dbname
```

**Notes:** Secrets are base64-encoded in `kubectl get secret -o yaml`. For stronger security use an external secrets controller (ExternalSecrets, HashiCorp Vault, SealedSecrets). Do **not** store long-lived credentials in plain ConfigMaps.

---

# 11 — PersistentVolume, PersistentVolumeClaim & StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/no-provisioner   # "local" for local PVs, cloud provisioners exist (aws-ebs, gce-pd, csi drivers)
volumeBindingMode: WaitForFirstConsumer     # best for topology-aware volumes
reclaimPolicy: Delete
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-nfs-50Gi
spec:
  capacity:
    storage: 50Gi
  accessModes:
    - ReadWriteMany
  storageClassName: nfs
  nfs:
    server: 10.0.0.5
    path: /export/data
  persistentVolumeReclaimPolicy: Retain
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-data
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: fast-ssd
```

**Notes:** Prefer dynamic provisioning via `StorageClass` with a CSI driver. For local SSDs use `local` PVs and `volumeBindingMode: WaitForFirstConsumer` to schedule pod and PV to same node.

---

# 12 — Horizontal Pod Autoscaler (HPA) — v2 with custom metric example

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource            # built-in CPU autoscaling
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 60
    - type: Pods                # example custom metric by pods
      pods:
        metric:
          name: requests_per_second
        target:
          type: AverageValue
          averageValue: "100"
```

**Notes:** For external/custom metrics, configure Metrics Adapter (Prometheus Adapter) or external metrics API. HPA v2 supports multiple metrics and scaling policies. Official HPA docs give detailed guidance. ([Kubernetes][4])

---

# 13 — Vertical Pod Autoscaler (VPA) — example (recommends or updates)

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: myapp-vpa
spec:
  targetRef:
    apiVersion: "apps/v1"
    kind: "Deployment"
    name: "myapp"
  updatePolicy:
    updateMode: "Auto"        # Off | Initial | Auto (Auto will evict pods to apply new requests)
```

**Notes:** VPA and HPA can conflict. Use VPA for workloads that cannot scale horizontally or during sizing exercises (recommendation mode). For production, often use VPA in `Off`/`Initial` mode + HPA for horizontal scaling.

---

# 14 — PodDisruptionBudget (PDB) — protect during voluntary disruptions

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: myapp-pdb
spec:
  minAvailable: 2              # or use maxUnavailable
  selector:
    matchLabels:
      app: myapp
```

**Notes:** PDBs affect voluntary evictions (drains, upgrades). They don't prevent the system from killing pods in node failures (involuntary). Use `minAvailable` for strict availability, `maxUnavailable` for percentage-based rules. (Use `policy/v1` for modern clusters.) ([Kubernetes][5])

---

# 15 — ServiceAccount & RBAC (Role, RoleBinding, ClusterRole, ClusterRoleBinding)

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: app-sa
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: app-role
  namespace: default
rules:
  - apiGroups: [""]
    resources: ["pods", "services"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: app-rolebinding
  namespace: default
subjects:
  - kind: ServiceAccount
    name: app-sa
    namespace: default
roleRef:
  kind: Role
  name: app-role
  apiGroup: rbac.authorization.k8s.io
---
# ClusterRole / ClusterRoleBinding example (cluster-wide privileges)
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-reader
rules:
  - apiGroups: [""]
    resources: ["pods", "nodes"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cluster-reader-binding
subjects:
  - kind: User
    name: alice
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: cluster-reader
  apiGroup: rbac.authorization.k8s.io
```

**Notes:** Follow least privilege. Use `kubectl auth can-i` to validate permissions. Prefer scoped Roles over ClusterRoles.

---

# 16 — TLS: Kubernetes TLS Secret (used by Ingress/Gateway)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-com-tls
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-cert>
  tls.key: <base64-encoded-key>
```

**Notes:** For automation, use cert-manager (ACME) to generate TLS secrets automatically.

---

# 17 — Certificate management (kubeadm-managed notes)

* `kubeadm certs check-expiration` — list local PKI certificate expirations.
* `kubeadm certs renew all` — renew kubeadm-managed certs; you must `systemctl restart kubelet` after renewing kubelet client certs. ([Kubernetes][6])

**Tip:** When adding new API SANs (IP/DNS) to apiserver certs, use:

```bash
kubeadm init phase certs apiserver --apiserver-cert-extra-sans "new.example.com,10.0.0.1"
# then restart kubelet / rotate manifests (move static pod trick) to pick up new certs
```

---

# 18 — Advanced & Misc components

### RuntimeClass

```yaml
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: kata
handler: kata-runtime
```

**Use:** Switch container runtime (gVisor, Kata) per Pod for isolation.

### NetworkPolicy (deny-by-default example)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-namespace
spec:
  podSelector:
    matchLabels:
      app: myapp
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              project: frontend
```

**Use:** Implement least-privilege network access.

---

# 19 — Comments about common options & trade-offs

* **Resource Requests/Limits:** Always set requests and limits in production. Requests drive scheduling; limits prevent noisy neighbors.
* **Probes:** Readiness probe controls service traffic; liveness probe controls restarts. Misconfigured probes cause rollout issues.
* **Affinity & Anti-affinity:** Use `preferredDuringSchedulingIgnoredDuringExecution` for best-effort spreading and `requiredDuringSchedulingIgnoredDuringExecution` for strict placement.
* **PodSecurityPolicy / Pod Security Admission:** Use PSA (builtin) or OPA/Gatekeeper for enforcements (no privileged containers, disallow hostPath).
* **Storage:** Use CSI drivers for dynamic provisioning (cloud: aws-ebs/gce-pd; on-prem: Rook/Ceph, OpenEBS, Longhorn). For shared file systems choose NFS/CephFS (RWO vs RWX matters).
* **Ingress vs Gateway API:** Ingress is stable and widely supported; Gateway API is newer and more expressive (traffic split, multi-listener, cross-namespace). For advanced traffic management prefer Gateway API. ([Kubernetes][7])

---

# 20 — How to annotate & comment your manifests for teams (best practice)

* Add `app.kubernetes.io/*` labels (convention) for tooling compatibility.
* Add `maintainer` and `git-commit` annotations to trace which repo/commit produced the manifest.
* Use Kustomize or Helm for overlays and templating; store env-specific patch files in Git (GitOps).

---

# 21 — Quick manifest snippets for “everyday” tasks

**Force rollout by annotation (triggered by CI after image push):**

```yaml
metadata:
  annotations:
    ci-image-sha: "sha256:abc123"   # change this to trigger rollout
```

**Deployment canary pattern (traffic split handled by Gateway/Ingress):**

* `myapp` (replicas: 4) — production
* `myapp-canary` (replicas: 1) — new version
* Use Gateway/Ingress weighted rules (e.g., weight: 90 -> myapp, weight: 10 -> myapp-canary)

---

## Sources / further reading

* Kubernetes workloads & Deployments docs. ([Kubernetes][1])
* HPA & autoscaling (v2) docs and walkthrough. ([Kubernetes][4])
* Gateway API guides & HTTPRoute. ([Kubernetes Gateway API][2])
* Ingress concept & ingress-nginx config. ([Kubernetes][3])
* kubeadm certificate management reference. ([Kubernetes][8])

---


[1]: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/?utm_source=chatgpt.com "Deployments"
[2]: https://gateway-api.sigs.k8s.io/guides/?utm_source=chatgpt.com "Getting started - Kubernetes Gateway API"
[3]: https://kubernetes.io/docs/concepts/services-networking/ingress/?utm_source=chatgpt.com "Ingress"
[4]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/?utm_source=chatgpt.com "Horizontal Pod Autoscaling"
[5]: https://kubernetes.io/docs/tasks/run-application/configure-pdb/?utm_source=chatgpt.com "Specifying a Disruption Budget for your Application"
[6]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-certs/?utm_source=chatgpt.com "kubeadm certs"
[7]: https://kubernetes.io/docs/concepts/services-networking/gateway/?utm_source=chatgpt.com "Gateway API"
[8]: https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-certs/?utm_source=chatgpt.com "Certificate Management with kubeadm"



---

# continue

It covers **node & pod affinity / anti-affinity**, **taints & tolerations**, **topologySpreadConstraints**, and **probes** (liveness/readiness/startup) with realistic examples, line-comments, and best-practice guidance.

Use these manifests directly (adjust labels, node names, and values for your environment). I kept comments in YAML so you can paste into files and keep the explanations inline.

---

# Affinity, Anti-Affinity, Taints/Tolerations & Probes — Practical Reference

---

## 1) Node Affinity (schedule pods onto nodes with matching labels)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: node-affinity-example
  labels:
    app: affinity-app
spec:
  containers:
    - name: app
      image: nginx:1.25
      ports: [{ containerPort: 80 }]
  affinity:
    nodeAffinity:
      # requiredDuringSchedulingIgnoredDuringExecution means the scheduler
      # will only place the pod on nodes matching these rules.
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
              - key: node-role.kubernetes.io/worker
                operator: In
                values:
                  - ""
              - key: topology.kubernetes.io/zone
                operator: In
                values:
                  - eu-west-1a
                  - eu-west-1b
      # preferredDuringSchedulingIgnoredDuringExecution gives a weighted preference
      # (scheduler will try but not fail scheduling if unavailable)
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          preference:
            matchExpressions:
              - key: hardware-type
                operator: In
                values:
                  - ssd-optimized
```

**When to use nodeAffinity**

* Place workloads on nodes with GPUs, SSDs, or special NICs.
* Schedule zone-aware workloads to ensure locality (lower latency).
* Use required for hard constraints (DB must be on SSD nodes). Use preferred for soft preferences (better performance but not mandatory).

---

## 2) Pod Affinity (co-locate pods — useful for locality / caching)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-affinity-example
  labels:
    app: frontend
spec:
  containers:
    - name: app
      image: busybox
      command: ["sh","-c","sleep 3600"]
  affinity:
    podAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            topologyKey: "kubernetes.io/hostname"   # prefer same node
            labelSelector:
              matchLabels:
                app: cache                         # prefer nodes that run 'cache' pods
```

**Use cases**

* Co-locate frontend with cache pods to reduce latency.
* Use *preferred* for performance gains; avoid required if it might block scheduling.

---

## 3) Pod Anti-Affinity (spread replicas across failure domains)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: anti-affinity-deploy
spec:
  replicas: 3
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchLabels:
                  app: backend
              topologyKey: "topology.kubernetes.io/zone"   # strictly prevent two 'backend' pods on same zone
```

**Notes**

* `requiredDuringSchedulingIgnoredDuringExecution` is strict — scheduler will not place a pod where it would violate this rule.
* For less strict behavior, use `preferredDuringSchedulingIgnoredDuringExecution` with weights.
* Use anti-affinity to ensure HA across zones/nodes for stateful services.

---

## 4) TopologySpreadConstraints (balanced spreading across nodes/zones)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spread-deploy
spec:
  replicas: 6
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      topologySpreadConstraints:
        - maxSkew: 1                        # at most 1 difference between the most/least populated topology domain
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: DoNotSchedule  # or ScheduleAnyway
          labelSelector:
            matchLabels:
              app: web
```

**Why use topologySpreadConstraints**

* Newer, more explicit way to evenly distribute pods across failure domains (zones, nodes).
* Prefer over complex podAntiAffinity for large clusters because it’s more scalable.

---

## 5) Taints & Tolerations (protect nodes or dedicate them)

**Taint a node (CLI):**

```bash
# Mark a node so only pods with matching tolerations can be scheduled
kubectl taint nodes node01 dedicated=ml:NoSchedule
```

**Remove a taint:**

```bash
kubectl taint nodes node01 dedicated:NoSchedule-
```

**Pod toleration manifest — allow scheduling on tainted node**

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: toleration-example
  labels:
    app: ml-job
spec:
  tolerations:
    # match key and operator
    - key: "dedicated"
      operator: "Equal"
      value: "ml"
      effect: "NoSchedule"      # tolerates NoSchedule taint
    # generic tolerate any NoSchedule taint
    - operator: "Exists"
      effect: "NoSchedule"
  containers:
    - name: trainer
      image: myregistry/trainer:1.0
```

**Taint types and what they mean**

* `NoSchedule` — kube-scheduler won't place non-tolerant pods on node.
* `PreferNoSchedule` — scheduler avoids but may place if necessary.
* `NoExecute` — evicts existing pods that do not tolerate the taint (can include `tolerationSeconds` to delay eviction).

**NoExecute example with timeout**

```yaml
tolerations:
  - key: "maintenance"
    operator: "Equal"
    value: "true"
    effect: "NoExecute"
    tolerationSeconds: 600   # tolerate eviction for 10 min, then get evicted
```

**Common use-cases**

* **Dedicated nodes**: GPU nodes, high-memory nodes — taint them and add tolerations to pods that need them.
* **Critical system isolation**: taint master/control-plane nodes to avoid scheduling app pods.
* **Maintenance windows**: add NoExecute with tolerationSeconds to give time to gracefully terminate jobs.

---

## 6) Combining nodeAffinity + tolerations (dedicated GPU pods)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-job
spec:
  containers:
    - name: trainer
      image: myorg/gpu-trainer:2.0
      resources:
        limits:
          nvidia.com/gpu: 1           # require GPU via device plugin
  nodeSelector:                    # quick shorthand (strict)
    gpu: "true"
  tolerations:
    - key: "dedicated"
      operator: "Equal"
      value: "gpu"
      effect: "NoSchedule"
```

**Notes**

* `nodeSelector` is simpler but less flexible than `nodeAffinity`.
* Prefer `nodeAffinity` for complex rules (zones, multiple keys, weighted preferences).

---

## 7) Liveness, Readiness, Startup Probes — Methods & Best Practices

### Probe types supported

* **HTTP GET** — controller performs HTTP request to container endpoint.
* **TCP Socket** — attempts to open TCP connection.
* **Exec** — runs a command inside the container; success if exit code 0.

### Generic probe fields

* `initialDelaySeconds` — seconds to wait before first probe
* `periodSeconds` — how often to probe
* `timeoutSeconds` — how long to wait for probe response
* `successThreshold` — consecutive successes for success (useful for startup)
* `failureThreshold` — consecutive failures before action

---

### Example: all three probes combined

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: probes-example
  labels:
    app: probes
spec:
  containers:
    - name: app
      image: myregistry/app:1.0
      ports:
        - containerPort: 8080
      # Startup probe detects long startup, prevents liveness from killing too early
      startupProbe:
        httpGet:
          path: /health/startup
          port: 8080
        failureThreshold: 60          # allow up to 60 failures at periodSeconds interval
        periodSeconds: 5
      # Liveness probe ensures container is alive (restart if unhealthy)
      livenessProbe:
        httpGet:
          path: /health/live
          port: 8080
        initialDelaySeconds: 30
        periodSeconds: 10
        timeoutSeconds: 3
        failureThreshold: 3
      # Readiness probe controls traffic (Service only routes when ready)
      readinessProbe:
        tcpSocket:
          port: 8080
        initialDelaySeconds: 5
        periodSeconds: 5
        timeoutSeconds: 1
```

**Practical guidance**

* **Startup probe**: use when app has heavy initialization (migrations, caches). Without it, liveness probe may kill container before it's ready. Set `startupProbe` and keep `livenessProbe` conservative.
* **Liveness probe**: avoid an aggressive `failureThreshold=1`. Typical values: `initialDelaySeconds: 30-60`, `periodSeconds: 10`, `failureThreshold: 3`.
* **Readiness probe**: should be strict — if pod is not ready do not receive traffic. Use TCP/HTTP checks that reflect app’s readiness to serve requests.
* **Timeouts**: set `timeoutSeconds` slightly larger than expected response time but small enough to detect failure.
* **Exec probes**: use only if necessary and container runtimes allow; exec probes incur overhead.
* **Probe endpoints**: instrument application to provide cheap, reliable health endpoints — avoid heavy operations in health checks (e.g., full DB migrations in readiness).
* **Resource-aware**: ensure probes are not starved by CPU throttling. Set requests to cover probe overhead.

---

## 8) Probe examples for common apps

### HTTP service (fast)

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 80
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 2
```

### Database-backed service (needs DB & cache)

```yaml
startupProbe:
  exec:
    command: ["sh", "-c", "pg_isready -U postgres -h $DB_HOST"]
  failureThreshold: 60
  periodSeconds: 10

livenessProbe:
  exec:
    command: ["sh", "-c", "curl -sSf http://localhost:8080/health/live || exit 1"]
  initialDelaySeconds: 60
  periodSeconds: 15
  failureThreshold: 4

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 2
```

---

## 9) Best Practices & Hard Lessons

* **Readiness vs Liveness**: readiness gates traffic; liveness restarts containers. Never implement readiness as a no-op — it's the correct place to verify dependencies (DB, caches). Liveness should detect unrecoverable hangs, not slow startup.
* **Startup probe**: Mandatory for apps with long bootstraps. It prevents premature restarts and reduces churn during deployments.
* **Avoid heavy checks**: health endpoints must be lightweight (e.g., check DB connection pool, not run heavy queries).
* **Graceful shutdown**: set `terminationGracePeriodSeconds` (default 30s) and `preStop` hooks to allow graceful close. Example:

  ```yaml
  lifecycle:
    preStop:
      exec:
        command: ["/bin/sh", "-c", "sleep 10; /usr/bin/graceful-shutdown"]
  ```
* **Resource requests**: always set `requests` so probes and containers get CPU when under pressure. If CPU is throttled, probes may show false negatives.
* **Probe thresholds**: maintain conservative `failureThreshold` to avoid noisy restarts during short network blips. Prefer to tune `periodSeconds` and `failureThreshold` together.
* **Use readiness gating for in-place updates**: for zero-downtime, ensure new pods pass readiness before old pods are removed.
* **Monitoring**: collect probe events (kube_events) and track restarts — frequent restarts indicate misconfiguration or resource pressure.
* **Test locally**: simulate slow responses and ensure `startupProbe` + `livenessProbe` combination behaves as expected.

---

## 10) CLI Quick Reference (useful commands)

```bash
# label and taint nodes
kubectl label node node01 gpu=true
kubectl taint nodes node01 dedicated=gpu:NoSchedule

# show nodes and taints
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.taints}{"\n"}{end}'

# check pod affinity decisions (events)
kubectl describe pod <podname> -n <ns> | grep -i affinity -A5

# view topologySpreadConstraints reasons in events
kubectl describe pod <pod> -n <ns> | grep -i "topology spread" -A4

# force test scheduling: try to schedule a pod with nodeAffinity but without toleration
kubectl apply -f node-affinity-example.yaml
kubectl get pod -w

# check probe failures / restart counts
kubectl get pods -o wide
kubectl describe pod <pod> -n <ns>    # shows events and probe failures
kubectl logs <pod> -c <container> --previous
```

---

## 11) Real-world patterns & recipes

* **Dedicated GPUs**: Taint GPU nodes `kubectl taint nodes node-gpu dedicated=gpu:NoSchedule`; label `gpu=true`; use nodeAffinity + tolerations in pod spec. Use device plugin for `nvidia.com/gpu` limits.
* **Critical system pods on control-plane**: Taint masters `node-role.kubernetes.io/controlplane=true:NoSchedule`; add toleration to kube-system critical pods only.
* **Spread DB replicas**: Use `topologySpreadConstraints` with `zone` and `maxSkew:1` to minimize risk of multiple replicas per zone.
* **Canary with affinity**: Run canary on same node as traffic generators for low-latency testing using `podAffinity` or direct nodeAffinity to test in production-like locality.
* **Maintenance drain strategy**: Add `PreferNoSchedule` taint during maintenance window to slowly migrate non-critical pods; later add `NoSchedule` to force eviction.

---

