# Kubernetes brief overview 

Below is a list of some of the most important Kubernetes component kinds that every DevOps engineer should be familiar with:

1. **Pod**  
   - The smallest deployable unit representing one or more containers running together.

2. **Service**  
   - An abstraction that defines a logical set of Pods and a policy by which to access them, typically via a stable IP address or DNS name.

3. **Deployment**  
   - Manages the desired state for Pods and ReplicaSets. It provides declarative updates for Pods and ensures the correct number are running.

4. **ReplicaSet**  
   - Ensures that a specified number of pod replicas are running at any given time. Deployments typically manage ReplicaSets.

5. **StatefulSet**  
   - Designed for stateful applications. It manages the deployment and scaling of a set of Pods, providing guarantees about the ordering and uniqueness of these Pods.

6. **DaemonSet**  
   - Ensures that a copy of a Pod runs on all (or some) Nodes. Useful for running cluster-wide services like logs or monitoring agents.

7. **Job**  
   - Creates one or more Pods that run to completion. It is used for short-lived or batch tasks.

8. **CronJob**  
   - Similar to Job but schedules Pods to run at specified times or intervals (like cron in Linux).

9. **ConfigMap**  
   - Stores non-confidential configuration data in key-value pairs. It can be used to configure applications.

10. **Secret**  
    - Similar to ConfigMaps but used for storing sensitive data, such as passwords, tokens, or keys. Data is encoded in base64.

11. **Ingress**  
    - Manages external access to services in a cluster, providing features like load balancing, SSL termination, and name-based virtual hosting.

12. **PersistentVolume (PV) and PersistentVolumeClaim (PVC)**  
    - PVs represent a piece of storage in the cluster that has been provisioned by an administrator.  
    - PVCs are requests for storage by a user. They bind to PVs to dynamically provision storage for pods.

13. **Namespace**  
    - Provides a mechanism for isolating groups of resources within a single cluster, useful for multi-tenant environments or separating development and production workloads.

14. **Node**  
    - Represents a worker machine in Kubernetes (a VM or physical server). Nodes host Pods and are managed by the control plane.

15. **ServiceAccount**  
    - Provides an identity for processes that run in a Pod. It is used for access control and interacting with the Kubernetes API.



Below are simple, complete YAML manifest examples for many of the common Kubernetes component kinds a DevOps engineer should know. In a real cluster, you often don’t need to create some of these (for example, Nodes are managed by Kubernetes automatically), but these examples show the basic structure and fields for each kind.

──────────────────────────────
1. Pod  
──────────────────────────────

This manifest defines a basic Pod running a single container:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
  labels:
    app: myapp
spec:
  containers:
    - name: mycontainer
      image: nginx:alpine
      ports:
        - containerPort: 80
```

──────────────────────────────
2. Service  
──────────────────────────────

This manifest creates a ClusterIP Service that fronts Pods with the label `app: myapp`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: example-service
spec:
  selector:
    app: myapp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
```

──────────────────────────────
3. Deployment  
──────────────────────────────

A Deployment managing 3 replicas of a Pod running Nginx:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
        - name: nginx-container
          image: nginx:alpine
          ports:
            - containerPort: 80
```

──────────────────────────────
4. ReplicaSet  
──────────────────────────────

A ReplicaSet ensuring 2 replicas of a basic Pod with a label:

```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: example-replicaset
spec:
  replicas: 2
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
        - name: nginx-container
          image: nginx:alpine
          ports:
            - containerPort: 80
```

──────────────────────────────
5. StatefulSet  
──────────────────────────────

A StatefulSet for a stateful application such as a simple database or even an nginx pod with stable network identity:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: example-statefulset
spec:
  serviceName: "nginx"
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
          image: nginx:alpine
          ports:
            - containerPort: 80
          volumeMounts:
            - name: www
              mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
    - metadata:
        name: www
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 1Gi
```

──────────────────────────────
6. DaemonSet  
──────────────────────────────

A DaemonSet to run a Pod on all (or selected) nodes (for example, a log collector):

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: example-daemonset
spec:
  selector:
    matchLabels:
      name: fluentd
  template:
    metadata:
      labels:
        name: fluentd
    spec:
      containers:
        - name: fluentd
          image: fluent/fluentd:v1.11-debian-1
          resources:
            limits:
              memory: 200Mi
              cpu: 100m
```

──────────────────────────────
7. Job  
──────────────────────────────

A Job that runs a one-time task (e.g., printing a message):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: example-job
spec:
  template:
    metadata:
      name: example-job
    spec:
      containers:
        - name: hello
          image: busybox
          command: ["echo", "Hello, Kubernetes!"]
      restartPolicy: Never
  backoffLimit: 3
```

──────────────────────────────
8. CronJob  
──────────────────────────────

A CronJob that runs a Job on a schedule (every minute in this example):

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: example-cronjob
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: hello
              image: busybox
              command: ["echo", "Hello from CronJob!"]
          restartPolicy: OnFailure
```

──────────────────────────────
9. ConfigMap  
──────────────────────────────

A ConfigMap storing non-sensitive configuration data:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example-configmap
data:
  app.properties: |
    setting1=value1
    setting2=value2
```

──────────────────────────────
10. Secret  
──────────────────────────────

A Secret storing sensitive information (values encoded in base64). For simplicity, here are pre-encoded values for `username` (admin) and `password` (s3cr3t):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-secret
type: Opaque
data:
  username: YWRtaW4=         # base64 for 'admin'
  password: czNjcjN0         # base64 for 's3cr3t'
```

──────────────────────────────
11. Ingress  
──────────────────────────────

An Ingress resource to expose a Service externally. (Note that an Ingress Controller must be installed in your cluster.)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
    - host: example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: example-service
                port:
                  number: 80
```

──────────────────────────────
12. PersistentVolume (PV) & PersistentVolumeClaim (PVC)  
──────────────────────────────

First, a PersistentVolume (usually created by an administrator):

PersistentVolume (PV):

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: example-pv
spec:
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /mnt/data    # For local testing only; in production consider networked storage.
```

PersistentVolumeClaim (PVC):

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: example-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

──────────────────────────────
13. Namespace  
──────────────────────────────

A Namespace to isolate resources:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: example-namespace
```

──────────────────────────────
14. Node  
──────────────────────────────

Nodes are usually provisioned by the cluster; however, here’s an example of a minimal Node manifest. (Note: Manual Node creation is uncommon; the K8s control plane automatically manages Nodes.)

```yaml
apiVersion: v1
kind: Node
metadata:
  name: example-node
spec:
  taints:
    - key: "node-role.kubernetes.io/master"
      effect: "NoSchedule"
```

──────────────────────────────
15. ServiceAccount  
──────────────────────────────

A ServiceAccount provides an identity for Pods running in a Namespace:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: example-serviceaccount
  namespace: default
```

---

