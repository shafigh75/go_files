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



Below is a comprehensive tutorial covering several advanced fundamentals in Kubernetes that a DevOps engineer should master. In this guide you’ll find:

- **Fundamentals of Autoscaling**  
- **Network Policies (Ingress and Egress examples)**  
- **RBAC with a Real Example**  
- **Resource Management & Restricting Resources on Pods**  
- **Ingress Example**  
- **A List of Essential and Useful kubectl Commands**

Each section includes YAML manifests and command examples so you can try them out in your own cluster.

---

## 1. Autoscaling Fundamentals

Autoscaling in Kubernetes comes in several flavors:

### A. Horizontal Pod Autoscaler (HPA)

The Horizontal Pod Autoscaler automatically scales the number of Pod replicas in a workload (Deployment, ReplicaSet, or StatefulSet) based on observed metrics such as CPU utilization.

#### 1.1 Deployment Example

Below is an example Deployment that runs a simple container with a CPU stress tool:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: autoscaling-demo
  labels:
    app: autoscaling-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: autoscaling-demo
  template:
    metadata:
      labels:
        app: autoscaling-demo
    spec:
      containers:
        - name: autoscaling-demo-container
          image: vish/stress  # This image generates CPU load.
          args: ["-cpus", "1", "-timeout", "600s"]
          resources:
            requests:
              cpu: "100m"
            limits:
              cpu: "500m"
```

Apply this Deployment manifest with:

```bash
kubectl apply -f autoscaling-demo-deployment.yaml
```

#### 1.2 Horizontal Pod Autoscaler Example

Next, deploy an HPA that adjusts the number of replicas for the `autoscaling-demo` Deployment based on CPU utilization:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: autoscaling-demo-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: autoscaling-demo
  minReplicas: 1
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 50
```

Apply the HPA with:

```bash
kubectl apply -f hpa.yaml
```

As load increases, Kubernetes will dynamically adjust the number of replicas.

---

## 2. Network Policies (Ingress & Egress)

Network Policies allow you to control traffic to and from Pods.

### A. Ingress Network Policy

This policy allows only Pods with the label `access: allowed` to send traffic to Pods labeled `app: web` on TCP port 80:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-ingress-from-allowed
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  ingress:
    - from:
        - podSelector:
            matchLabels:
              access: allowed
      ports:
        - protocol: TCP
          port: 80
```

### B. Egress Network Policy

The following policy restricts Pods with the label `app: web` so they can only communicate externally on TCP port 80 and UDP port 53 (DNS):

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: restrict-egress
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  egress:
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - protocol: TCP
          port: 80
        - protocol: UDP
          port: 53
  policyTypes:
    - Egress
```

Apply the network policies with:

```bash
kubectl apply -f <network-policy-filename>.yaml
```

*Note:* Make sure your cluster uses a network provider (like Calico, Cilium, or Weave) that supports NetworkPolicy.

---

## 3. RBAC with a Real Example

Role-Based Access Control (RBAC) restricts what users or ServiceAccounts can do within a namespace or across the cluster.

### Example: Create a Role for Read-Access to Pods and Bind It to a User

#### 3.1 Role Definition

Create a Role that allows reading Pods in the `default` namespace:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default
  name: pod-reader
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "watch", "list"]
```

#### 3.2 RoleBinding

Bind the above role to a user (e.g., `jane-doe`):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: pod-reader-binding
  namespace: default
subjects:
  - kind: User
    name: jane-doe  # Replace with the actual user name
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

Apply both manifests with:

```bash
kubectl apply -f pod-reader-role.yaml
kubectl apply -f pod-reader-binding.yaml
```

Now, the user `jane-doe` will have only read access (get, watch, list) to Pods in the default namespace.

---

## 4. Resource Management: Requests & Limits on Pods

Resource management ensures that Pods have the resources they need while preventing overcommitment on cluster nodes.

### Example: Deployment with Resource Requests and Limits

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: resource-demo
  labels:
    app: resource-demo
spec:
  replicas: 2
  selector:
    matchLabels:
      app: resource-demo
  template:
    metadata:
      labels:
        app: resource-demo
    spec:
      containers:
        - name: resource-demo-container
          image: nginx:alpine
          resources:
            requests:
              memory: "64Mi"
              cpu: "250m"
            limits:
              memory: "128Mi"
              cpu: "500m"
```

Apply the manifest with:

```bash
kubectl apply -f resource-demo.yaml
```

Each container is guaranteed at least 250 millicores of CPU and 64MB memory and will not exceed 500 millicores of CPU and 128MB memory.

---

## 5. Ingress Example

An Ingress resource routes external traffic to Services within the cluster. You must have an Ingress Controller (for example, the NGINX Ingress Controller) deployed.

### Example: Ingress Manifest Routing Traffic for Two Hosts

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
    - host: www.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web-service
                port:
                  number: 80
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-service
                port:
                  number: 80
```

Apply this Ingress manifest with:

```bash
kubectl apply -f web-ingress.yaml
```

---

## 6. Essential and Useful kubectl Commands

As a DevOps engineer, mastering `kubectl` is crucial. Below is a list of important commands to help you manage, inspect, and troubleshoot your Kubernetes environments.

### A. Basic Commands

```bash
# Display cluster version info:
kubectl version --short

# Get cluster nodes:
kubectl get nodes

# Get all resources in the default namespace:
kubectl get all

# Watch ongoing changes (e.g., Pods):
kubectl get pods --watch

# Describe a specific Pod:
kubectl describe pod <pod-name>
```

### B. Creating and Deleting Resources

```bash
# Create a resource from a YAML file:
kubectl apply -f <resource.yaml>

# Delete a resource defined in a file:
kubectl delete -f <resource.yaml>

# Force delete (if necessary):
kubectl delete pod <pod-name> --grace-period=0 --force
```

### C. Managing Deployments and Scaling

```bash
# Check rollout status of a Deployment:
kubectl rollout status deployment/<deployment-name>

# Rollback a Deployment:
kubectl rollout undo deployment/<deployment-name>

# Scale a Deployment:
kubectl scale deployment/<deployment-name> --replicas=3
```

### D. Logging and Debugging

```bash
# Fetch logs from a specific Pod:
kubectl logs <pod-name>

# Stream real-time logs from a Pod:
kubectl logs -f <pod-name>

# Open a shell inside a Pod:
kubectl exec -it <pod-name> -- /bin/sh
```

### E. Resource Inspection and Editing

```bash
# Get a detailed view of a resource in YAML format:
kubectl get pod <pod-name> -o yaml

# Edit a resource directly:
kubectl edit deployment <deployment-name>
```

### F. Namespace Management

```bash
# List all namespaces:
kubectl get namespaces

# Set a default namespace for your kubectl session:
kubectl config set-context --current --namespace=<namespace-name>
```

### G. Advanced Queries and Label Filtering

```bash
# Get resources with a specific label:
kubectl get pods -l app=web

# Use jsonpath to extract a specific field (e.g., pod IP):
kubectl get pod <pod-name> -o jsonpath="{.status.podIP}"
```

### H. Working with Custom Resources and Events

```bash
# List events (useful for troubleshooting):
kubectl get events

# List custom resource definitions (CRDs):
kubectl get crd
```

### I. Config Viewing and Contexts

```bash
# View current context:
kubectl config current-context

# List all contexts:
kubectl config get-contexts

# Switch to a different context:
kubectl config use-context <context-name>
```

---

## Conclusion

This tutorial provided hands-on examples for:

- **Autoscaling:** Deploying a sample workload and setting up a Horizontal Pod Autoscaler.
- **Network Policies:** Implementing both ingress and egress rules.
- **RBAC:** Creating a Role and a RoleBinding with a real example.
- **Resource Management:** Defining resource requests and limits for Pods.
- **Ingress:** Exposing services through an Ingress resource.
- **kubectl Commands:** A comprehensive list of commands to operate, inspect, and troubleshoot your Kubernetes environment.

By practicing these configurations in your lab environment (using Minikube, KIND, or a managed Kubernetes cluster), you'll build the foundation to become a true kubectl guru and a well-rounded Kubernetes DevOps engineer. Happy scaling, securing, and managing your clusters!
