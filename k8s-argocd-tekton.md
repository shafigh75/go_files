# Kubernetes CI/CD with Tekton and ArgoCD: A Production-Ready Handbook

## Table of Contents

| Section | Description |
|---------|-------------|
| [1. Introduction](#1-introduction) | Overview of the CI/CD pipeline architecture |
| [2. Prerequisites](#2-prerequisites) | Required tools and setup |
| [3. Tekton Overview](#3-tekton-overview) | Understanding Tekton components and flow |
| [4. Go Application](#4-go-application) | Sample Go application for our pipeline |
| [5. GitLab Setup](#5-gitlab-setup) | Configuring GitLab repository and registry |
| [6. Tekton CI Pipeline](#6-tekton-ci-pipeline) | Building the CI pipeline with Tekton |
| [7. Kustomize Configuration](#7-kustomize-configuration) | Using Kustomize for configuration management |
| [8. ArgoCD Setup](#8-argocd-setup) | Installing and configuring ArgoCD |
| [9. End-to-End Workflow](#9-end-to-end-workflow) | Understanding the complete workflow |
| [10. Production Considerations](#10-production-considerations) | Security, scalability, and monitoring |
| [11. Troubleshooting](#11-troubleshooting) | Common issues and solutions |

## 1. Introduction

This handbook provides a comprehensive guide to implementing a secure, cloud-native CI/CD pipeline using Tekton for Continuous Integration and ArgoCD for Continuous Deployment on Kubernetes. We'll walk through a real-world example using a Go application, GitLab as our source code repository and container registry, and Kustomize for configuration management.

Our pipeline follows GitOps principles:
1. Automatically trigger when code is pushed to GitLab
2. Build, test, and scan the Go application
3. Build and push a Docker image to GitLab registry
4. Create a merge request to update manifests with the new image digest
5. ArgoCD automatically deploys after merge approval

## 2. Prerequisites

Before starting, ensure you have:

- A Kubernetes cluster (v1.20+)
- `kubectl` configured to access your cluster
- Docker installed and configured
- Go installed (v1.19+)
- GitLab account with repository access
- Helm 3 installed
- Tekton CLI (`tkn`) installed

## 3. Tekton Overview

Tekton is a cloud-native solution for building CI/CD systems. It's built on Kubernetes Custom Resource Definitions (CRDs) and provides:

- **Tasks**: Reusable steps that perform specific actions
- **Pipelines**: Sequences of tasks that form a CI/CD workflow
- **TaskRuns and PipelineRuns**: Executions of tasks and pipelines
- **Triggers**: Event-based triggers to start pipelines
- **Workspaces**: Volumes for passing data between tasks

### Tekton Component Flow

1. **EventListener**: Receives HTTP events (e.g., webhooks) with TLS termination
2. **Trigger**: Processes events and creates resources
3. **TriggerBinding**: Extracts parameters from event payloads
4. **TriggerTemplate**: Defines resources to create (e.g., PipelineRuns)
5. **Pipeline**: Defines the workflow with tasks
6. **Task**: Defines individual steps in the workflow
7. **Workspace**: Shared storage between tasks
8. **Results**: Outputs from tasks that can be used by subsequent tasks

## 4. Go Application

Let's create a secure Go web application that we'll use throughout this guide.

### Directory Structure

```
go-app/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   └── api/
│       └── handlers.go
├── pkg/
│   └── version/
│       └── version.go
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

### Application Code

**go.mod**
```go
module gitlab.com/yourusername/go-app

go 1.19

require (
    github.com/gorilla/mux v1.8.0
)
```

**pkg/version/version.go**
```go
package version

var (
    // Version is the application version
    Version = "1.0.0"
    // GitCommit is the git commit hash
    GitCommit = "unknown"
    // BuildDate is the build date
    BuildDate = "unknown"
)
```

**internal/api/handlers.go**
```go
package api

import (
    "encoding/json"
    "net/http"
    "time"

    "gitlab.com/yourusername/go-app/pkg/version"
)

// HealthResponse represents the health check response
type HealthResponse struct {
    Status     string `json:"status"`
    Version    string `json:"version"`
    Timestamp  string `json:"timestamp"`
    Uptime     string `json:"uptime"`
}

// VersionResponse represents the version response
type VersionResponse struct {
    Version    string `json:"version"`
    GitCommit  string `json:"git_commit"`
    BuildDate  string `json:"build_date"`
    GoVersion  string `json:"go_version"`
}

var startTime = time.Now()

// HealthHandler handles health check requests
func HealthHandler(w http.ResponseWriter, r *http.Request) {
    uptime := time.Since(startTime).String()
    
    response := HealthResponse{
        Status:    "ok",
        Version:   version.Version,
        Timestamp: time.Now().Format(time.RFC3339),
        Uptime:    uptime,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}

// VersionHandler handles version requests
func VersionHandler(w http.ResponseWriter, r *http.Request) {
    response := VersionResponse{
        Version:   version.Version,
        GitCommit: version.GitCommit,
        BuildDate: version.BuildDate,
        GoVersion: "1.19",
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(response)
}
```

**cmd/server/main.go**
```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    "gitlab.com/yourusername/go-app/internal/api"
)

func main() {
    // Create router
    r := mux.NewRouter()
    
    // API routes
    r.HandleFunc("/health", api.HealthHandler).Methods("GET")
    r.HandleFunc("/version", api.VersionHandler).Methods("GET")
    
    // Health check endpoint for Kubernetes
    r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    }).Methods("GET")
    
    // Start server
    srv := &http.Server{
        Handler:      r,
        Addr:         ":8080",
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }
    
    // Graceful shutdown
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()
    
    log.Println("Server started on :8080")
    
    // Wait for interrupt signal to gracefully shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down server...")
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    
    // Shutdown server
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }
    
    log.Println("Server exited properly")
}
```

**Dockerfile** (Fixed security issues)
```dockerfile
# Build stage
FROM golang:1.19-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with version information
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG VERSION=1.0.0

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "\
    -X 'gitlab.com/yourusername/go-app/pkg/version.Version=${VERSION}' \
    -X 'gitlab.com/yourusername/go-app/pkg/version.GitCommit=${GIT_COMMIT}' \
    -X 'gitlab.com/yourusername/go-app/pkg/version.BuildDate=${BUILD_DATE}'" \
    -a -installsuffix cgo -o main ./cmd/server

# Final stage with non-root user
FROM alpine:3.18

# Create non-root user
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup && \
    apk --no-cache add ca-certificates tzdata && \
    rm -rf /var/cache/apk/*

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Set permissions
RUN chown -R appuser:appgroup /app && \
    chmod -R 755 /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./main"]
```

## 5. GitLab Setup

### Creating the Repository

1. Create a new repository in GitLab named `go-app`
2. Clone the repository locally and push our Go application code
3. Enable the container registry in your GitLab project settings

### GitLab Personal Access Token

Create a personal access token in GitLab with the following scopes:
- `api` (for merge requests)
- `read_repository`
- `write_repository`
- `read_registry`
- `write_registry`

Store this token securely - we'll use it for Tekton authentication.

## 6. Tekton CI Pipeline

### 6.1 Installing Tekton

```bash
# Create dedicated namespace for Tekton
kubectl create namespace tekton-pipelines

# Install Tekton Pipelines
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.47.0/release.yaml

# Install Tekton Triggers
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/v0.22.0/release.yaml

# Install Tekton CLI
# Linux
curl -LO https://github.com/tektoncd/cli/releases/download/v0.29.0/tkn_0.29.0_Linux_x86_64.tar.gz
tar xvzf tkn_0.29.0_Linux_x86_64.tar.gz -C /tmp/
sudo mv /tmp/tkn /usr/local/bin/

# macOS
brew install tektoncd-cli
```

### 6.2 Creating Service Account for Tekton (Fixed RBAC)

```yaml
# tekton-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tekton-pipeline
  namespace: tekton-pipelines
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: tekton-pipeline-role
  namespace: tekton-pipelines
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "secrets", "configmaps", "persistentvolumeclaims", "events"]
  verbs: ["get", "list", "create", "update", "delete", "watch"]
- apiGroups: ["tekton.dev"]
  resources: ["tasks", "pipelines", "taskruns", "pipelineruns", "pipelineresources"]
  verbs: ["get", "list", "create", "update", "delete", "watch"]
- apiGroups: ["triggers.tekton.dev"]
  resources: ["eventlisteners", "triggers", "triggerbindings", "triggertemplates"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["policy"]
  resources: ["podsecuritypolicies"]
  resourceNames: ["tekton-restricted"]
  verbs: ["use"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: tekton-pipeline-binding
  namespace: tekton-pipelines
subjects:
- kind: ServiceAccount
  name: tekton-pipeline
  namespace: tekton-pipelines
roleRef:
  kind: Role
  name: tekton-pipeline-role
  apiGroup: rbac.authorization.k8s.io
```

```bash
kubectl apply -f tekton-rbac.yaml
```

### 6.3 Creating Secrets for Credentials

```bash
# Create namespace for our application
kubectl create namespace go-app-ci

# Git credentials (for cloning and MR creation)
kubectl create secret generic gitlab-credentials \
  --from-literal=username=<your-gitlab-username> \
  --from-literal=password=<your-gitlab-personal-access-token> \
  --from-literal=url=gitlab.com \
  -n go-app-ci

# Docker registry credentials
kubectl create secret docker-registry gitlab-registry-secret \
  --docker-server=registry.gitlab.com \
  --docker-username=<your-gitlab-username> \
  --docker-password=<your-gitlab-personal-access-token> \
  -n go-app-ci
```

### 6.4 Tekton Triggers (Fixed Security)

**triggers/gitlab-webhook-secret.yaml**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gitlab-webhook-secret
  namespace: go-app-ci
type: Opaque
stringData:
  webhook-secret: $(openssl rand -hex 20)
```

**triggers/trigger-template.yaml**
```yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerTemplate
metadata:
  name: go-app-pipeline-template
  namespace: go-app-ci
spec:
  params:
  - name: git-revision
    description: The git revision
    default: main
  - name: git-repository-url
    description: The git repository url
  - name: git-repository-name
    description: The git repository name (for image paths)
  - name: git-commit-sha
    description: The git commit SHA
  - name: git-commit-message
    description: The git commit message
  - name: git-namespace
    description: The GitLab namespace/group
  - name: manifests-repo-url
    description: URL of the manifests repository
  
  resourcetemplates:
  - apiVersion: tekton.dev/v1beta1
    kind: PipelineRun
    metadata:
      generateName: go-app-pipeline-run-
      namespace: go-app-ci
    spec:
      serviceAccountName: tekton-pipeline
      pipelineRef:
        name: go-app-ci-pipeline
      
      params:
      - name: git-revision
        value: $(tt.params.git-revision)
      - name: git-repository-url
        value: $(tt.params.git-repository-url)
      - name: git-commit-sha
        value: $(tt.params.git-commit-sha)
      - name: git-commit-message
        value: $(tt.params.git-commit-message)
      - name: image-registry
        value: registry.gitlab.com
      - name: image-repository
        value: $(tt.params.git-namespace)/$(tt.params.git-repository-name)
      - name: image-tag
        value: $(tt.params.git-commit-sha)
      - name: manifests-repo-url
        value: $(tt.params.manifests-repo-url)
      
      workspaces:
      - name: shared-workspace
        volumeClaimTemplate:
          spec:
            accessModes: [ReadWriteOnce]
            resources:
              requests:
                storage: 2Gi
      - name: git-credentials
        secret:
          secretName: gitlab-credentials
      - name: docker-config
        secret:
          secretName: gitlab-registry-secret
```

**triggers/trigger-binding.yaml**
```yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerBinding
metadata:
  name: go-app-pipeline-binding
  namespace: go-app-ci
spec:
  params:
  - name: git-revision
    value: $(body.checkout_sha)
  - name: git-repository-url
    value: $(body.project.git_http_url)
  - name: git-commit-sha
    value: $(body.checkout_sha)
  - name: git-commit-message
    value: $(body.commits[0].message)
  - name: git-repository-name
    value: $(body.project.name)
  - name: git-namespace
    value: $(body.project.namespace)
  - name: manifests-repo-url
    value: https://gitlab.com/$(body.project.namespace)/go-app-manifests.git
```

**triggers/trigger.yaml**
```yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: Trigger
metadata:
  name: go-app-trigger
  namespace: go-app-ci
spec:
  serviceAccountName: tekton-pipeline
  bindings:
  - ref: go-app-pipeline-binding
  template:
    ref: go-app-pipeline-template
```

**triggers/event-listener.yaml**
```yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: EventListener
metadata:
  name: go-app-listener
  namespace: go-app-ci
spec:
  serviceAccountName: tekton-pipeline
  replicas: 2
  triggers:
  - triggerRef: go-app-trigger
    interceptors:
    - gitlab:
        secretRef:
          secretName: gitlab-webhook-secret
          secretKey: webhook-secret
        eventTypes: ["Push Hook"]
  resources:
    kubernetesResource:
      spec:
        template:
          spec:
            containers:
            - resources:
                requests:
                  memory: "64Mi"
                  cpu: "100m"
                limits:
                  memory: "128Mi"
                  cpu: "500m"
```

### 6.5 Tekton Tasks (Fixed Security and Added Testing)

**tasks/fetch-source.yaml**
```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: fetch-source
  namespace: go-app-ci
spec:
  description: Fetch source code from Git repository with proper credentials
  workspaces:
  - name: shared-workspace
  - name: git-credentials
  params:
  - name: url
    type: string
  - name: revision
    type: string
    default: main
  steps:
  - name: clone-repository
    image: alpine/git:v2.38.1
    script: |
      # Setup git credentials
      mkdir -p /root/.git
      cat /workspace/git-credentials/username | tr -d '\n' > /tmp/username
      cat /workspace/git-credentials/password | tr -d '\n' > /tmp/password
      cat /workspace/git-credentials/url | tr -d '\n' > /tmp/url
      
      git config --global credential.helper store
      echo "https://$(cat /tmp/username):$(cat /tmp/password)@$(cat /tmp/url)" > /root/.git-credentials
      chmod 600 /root/.git-credentials
      
      # Clone repository
      cd /workspace/shared-workspace
      git clone --depth=1 --single-branch --branch $(params.revision) $(params.url) .
      git config --global --add safe.directory /workspace/shared-workspace
```

**tasks/test-go.yaml**
```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: test-go
  namespace: go-app-ci
spec:
  description: Run Go tests and security scans
  workspaces:
  - name: shared-workspace
  steps:
  - name: run-tests
    image: golang:1.19-alpine
    script: |
      cd /workspace/shared-workspace
      go test -v -cover ./...
      
      # Static analysis
      go vet ./...
      
      # Security scan (optional - add if needed)
      # go install github.com/securego/gosec/v2/cmd/gosec@latest
      # gosec -fmt=sarif -out=results.sarif ./...

  - name: scan-dependencies
    image: aquasec/trivy:0.45.1
    script: |
      cd /workspace/shared-workspace
      trivy fs --security-checks vuln --exit-code 1 --severity CRITICAL ./ 
```

**tasks/build-go.yaml**
```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: build-go
  namespace: go-app-ci
spec:
  description: Build Go application with version information
  workspaces:
  - name: shared-workspace
  params:
  - name: git-commit
    type: string
  - name: build-date
    type: string
    default: $(date +%Y-%m-%dT%H:%M:%SZ)
  - name: version
    type: string
    default: 1.0.0
  steps:
  - name: build
    image: golang:1.19-alpine
    env:
    - name: CGO_ENABLED
      value: "0"
    - name: GOOS
      value: linux
    script: |
      cd /workspace/shared-workspace
      
      # Build with version information
      go build -ldflags "\
        -X 'gitlab.com/yourusername/go-app/pkg/version.Version=$(params.version)' \
        -X 'gitlab.com/yourusername/go-app/pkg/version.GitCommit=$(params.git-commit)' \
        -X 'gitlab.com/yourusername/go-app/pkg/version.BuildDate=$(params.build-date)'" \
        -o bin/main ./cmd/server
      
      # Verify binary
      file bin/main
      chmod +x bin/main
```

**tasks/build-and-push-image.yaml**
```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: build-and-push-image
  namespace: go-app-ci
spec:
  description: Build and push Docker image to registry using Kaniko
  workspaces:
  - name: shared-workspace
  - name: docker-config
  params:
  - name: image-url
    type: string
  - name: context
    type: string
    default: .
  - name: dockerfile
    type: string
    default: ./Dockerfile
  results:
  - name: image-digest
    description: Digest of the built image
  steps:
  - name: build-and-push
    image: gcr.io/kaniko-project/executor:v1.23.2
    env:
    - name: DOCKER_CONFIG
      value: /tekton/home/.docker
    script: |
      # Setup Docker config directory
      mkdir -p /tekton/home/.docker
      cp /workspace/docker-config/.dockerconfigjson /tekton/home/.docker/config.json
      
      # Build and push image
      /kaniko/executor \
        --dockerfile=$(params.dockerfile) \
        --context=/workspace/shared-workspace/$(params.context) \
        --destination=$(params.image-url) \
        --snapshotMode=redo \
        --use-new-run \
        --digest-file=/workspace/image-digest.txt
      
      # Output digest
      cat /workspace/image-digest.txt | tee $(results.image-digest.path)
```

**tasks/create-merge-request.yaml**
```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: create-merge-request
  namespace: go-app-ci
spec:
  description: Create merge request to update manifests repository
  workspaces:
  - name: git-credentials
  params:
  - name: git-namespace
    type: string
  - name: project-name
    type: string
  - name: manifests-repo-url
    type: string
  - name: image-registry
    type: string
  - name: image-repository
    type: string
  - name: image-tag
    type: string
  - name: image-digest
    type: string
  - name: git-commit-sha
    type: string
  - name: git-commit-message
    type: string
  steps:
  - name: create-mr
    image: alpine/gitlab-cli:latest
    env:
    - name: GITLAB_HOST
      value: gitlab.com
    - name: GITLAB_TOKEN
      valueFrom:
        secretKeyRef:
          name: gitlab-credentials
          key: password
    script: |
      # Clone manifests repository
      git clone $(params.manifests-repo-url) manifests
      cd manifests
      
      # Configure git
      git config --global user.email "tekton-ci@gitlab.com"
      git config --global user.name "Tekton CI"
      
      # Create branch for this update
      BRANCH_NAME="update-image-$(params.git-commit-sha)"
      git checkout -b $BRANCH_NAME
      
      # Update image in overlays
      find overlays -name kustomization.yaml -exec sed -i "s|image: $(params.image-registry)/$(params.image-repository):.*|image: $(params.image-registry)/$(params.image-repository)@$(params.image-digest)|g" {} \;
      
      # Add gitlab-ci.yml if not exists (for ArgoCD webhook)
      if [ ! -f .gitlab-ci.yml ]; then
        cat > .gitlab-ci.yml << EOF
        argocd-sync:
          image: alpine:3.18
          before_script:
            - apk add --no-cache curl
          script:
            - |
              curl -XPOST -H "Content-Type: application/json" \
                -H "Authorization: Bearer $ARGOCD_AUTH_TOKEN" \
                "$ARGOCD_WEBHOOK_URL?path=overlays/production" || true
          only:
            - main
        EOF
      fi
      
      # Commit changes
      git add .
      git commit -m "Update image to $(params.image-tag)
      
      Automated update from Tekton CI pipeline
      Commit SHA: $(params.git-commit-sha)
      Image digest: $(params.image-digest)"
      
      # Push branch
      git push --set-upstream origin $BRANCH_NAME
      
      # Create merge request
      gitlab project merge-request create \
        --project $(params.git-namespace)/go-app-manifests \
        --source-branch $BRANCH_NAME \
        --target-branch main \
        --title "Update application image to $(params.image-tag)" \
        --description "Automated update from Tekton CI pipeline for commit $(params.git-commit-sha)
        
        **Image Details:**
        - Repository: $(params.image-registry)/$(params.image-repository)
        - Tag: $(params.image-tag)
        - Digest: $(params.image-digest)
        
        **Source Commit:**
        - SHA: $(params.git-commit-sha)
        - Message: $(params.git-commit-message)"
```

### 6.6 Tekton Pipeline (Fixed Flow)

**pipelines/go-app-ci-pipeline.yaml**
```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: go-app-ci-pipeline
  namespace: go-app-ci
spec:
  description: CI pipeline for Go application with testing and image building
  params:
  - name: git-revision
    type: string
    default: main
  - name: git-repository-url
    type: string
  - name: git-commit-sha
    type: string
  - name: git-commit-message
    type: string
  - name: git-namespace
    type: string
  - name: image-registry
    type: string
    default: registry.gitlab.com
  - name: image-repository
    type: string
  - name: image-tag
    type: string
  - name: manifests-repo-url
    type: string
  
  workspaces:
  - name: shared-workspace
  - name: git-credentials
  - name: docker-config
  
  tasks:
  - name: fetch-source
    taskRef:
      name: fetch-source
    params:
    - name: url
      value: $(params.git-repository-url)
    - name: revision
      value: $(params.git-revision)
    workspaces:
    - name: shared-workspace
      workspace: shared-workspace
    - name: git-credentials
      workspace: git-credentials

  - name: test-go
    taskRef:
      name: test-go
    runAfter: [fetch-source]
    workspaces:
    - name: shared-workspace
      workspace: shared-workspace

  - name: build-go
    taskRef:
      name: build-go
    runAfter: [test-go]
    params:
    - name: git-commit
      value: $(params.git-commit-sha)
    - name: build-date
      value: $(date +%Y-%m-%dT%H:%M:%SZ)
    workspaces:
    - name: shared-workspace
      workspace: shared-workspace

  - name: build-and-push-image
    taskRef:
      name: build-and-push-image
    runAfter: [build-go]
    params:
    - name: image-url
      value: $(params.image-registry)/$(params.image-repository):$(params.image-tag)
    workspaces:
    - name: shared-workspace
      workspace: shared-workspace
    - name: docker-config
      workspace: docker-config

  - name: create-merge-request
    taskRef:
      name: create-merge-request
    runAfter: [build-and-push-image]
    params:
    - name: git-namespace
      value: $(params.git-namespace)
    - name: project-name
      value: "go-app"
    - name: manifests-repo-url
      value: $(params.manifests-repo-url)
    - name: image-registry
      value: $(params.image-registry)
    - name: image-repository
      value: $(params.image-repository)
    - name: image-tag
      value: $(params.image-tag)
    - name: image-digest
      value: $(tasks.build-and-push-image.results.image-digest)
    - name: git-commit-sha
      value: $(params.git-commit-sha)
    - name: git-commit-message
      value: $(params.git-commit-message)
    workspaces:
    - name: git-credentials
      workspace: git-credentials
```

### 6.7 Deploying the Tekton Pipeline

```bash
# Create the namespace
kubectl create namespace go-app-ci

# Apply RBAC
kubectl apply -f tekton-rbac.yaml

# Create secrets
kubectl create secret generic gitlab-credentials \
  --from-literal=username=<your-gitlab-username> \
  --from-literal=password=<your-gitlab-personal-access-token> \
  --from-literal=url=gitlab.com \
  -n go-app-ci

kubectl create secret docker-registry gitlab-registry-secret \
  --docker-server=registry.gitlab.com \
  --docker-username=<your-gitlab-username> \
  --docker-password=<your-gitlab-personal-access-token> \
  -n go-app-ci

# Create webhook secret
kubectl create secret generic gitlab-webhook-secret \
  --from-literal=webhook-secret=$(openssl rand -hex 20) \
  -n go-app-ci

# Create tasks
kubectl apply -f tasks/

# Create pipeline
kubectl apply -f pipelines/

# Create triggers
kubectl apply -f triggers/

# Expose the EventListener with TLS (using ingress)
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tekton-webhook
  namespace: go-app-ci
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
spec:
  tls:
  - hosts:
    - tekton-webhook.yourdomain.com
    secretName: tekton-webhook-tls
  rules:
  - host: tekton-webhook.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: el-go-app-listener
            port:
              number: 8080
EOF
```

### 6.8 GitLab Webhook Configuration

1. Get the webhook URL:
   ```
   https://tekton-webhook.yourdomain.com
   ```

2. In GitLab, go to Settings > Webhooks and add a new webhook:
   - URL: `https://tekton-webhook.yourdomain.com`
   - Secret Token: The same token used in the `gitlab-webhook-secret`
   - Trigger events: Push events only
   - Enable SSL verification

## 7. Kustomize Configuration

### Directory Structure

```
manifests/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── kustomization.yaml
└── overlays/
    ├── production/
    │   ├── kustomization.yaml
    │   └── resources.yaml
    └── staging/
        ├── kustomization.yaml
        └── resources.yaml
```

### Base Manifests

**manifests/base/deployment.yaml**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-app
  labels:
    app: go-app
    app.kubernetes.io/name: go-app
    app.kubernetes.io/component: backend
    app.kubernetes.io/part-of: go-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: go-app
  template:
    metadata:
      labels:
        app: go-app
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 10001
        runAsGroup: 10001
        fsGroup: 10001
      containers:
      - name: go-app
        image: registry.gitlab.com/yourusername/go-app@sha256:placeholder
        ports:
        - containerPort: 8080
          name: http
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 30
          periodSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          failureThreshold: 3
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: tmp-volume
          mountPath: /tmp
      volumes:
      - name: tmp-volume
        emptyDir: {}
```

**manifests/base/service.yaml**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: go-app-service
  labels:
    app: go-app
    app.kubernetes.io/name: go-app
    app.kubernetes.io/component: backend
    app.kubernetes.io/part-of: go-app
spec:
  selector:
    app: go-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: http
  type: ClusterIP
```

**manifests/base/kustomization.yaml**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- deployment.yaml
- service.yaml

commonLabels:
  app.kubernetes.io/name: go-app
  app.kubernetes.io/component: backend
  app.kubernetes.io/part-of: go-app

images:
- name: registry.gitlab.com/yourusername/go-app
  newName: registry.gitlab.com/yourusername/go-app
  newTag: placeholder
```

### Production Overlay

**manifests/overlays/production/kustomization.yaml**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
- ../../base

namespace: production

patchesStrategicMerge:
- resources.yaml

replicas:
- name: go-app
  count: 3

resources:
- resources.yaml

commonAnnotations:
  argocd.argoproj.io/sync-wave: "1"
```

**manifests/overlays/production/resources.yaml**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-app
spec:
  template:
    spec:
      containers:
      - name: go-app
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "500m"
```

### Staging Overlay

**manifests/overlays/staging/kustomization.yaml**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

bases:
- ../../base

namespace: staging

patchesStrategicMerge:
- resources.yaml

replicas:
- name: go-app
  count: 1

resources:
- resources.yaml

commonAnnotations:
  argocd.argoproj.io/sync-wave: "1"
```

**manifests/overlays/staging/resources.yaml**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-app
spec:
  template:
    spec:
      containers:
      - name: go-app
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "200m"
```

### Setting up the Manifests Repository

1. Create a new repository in GitLab named `go-app-manifests`
2. Clone the repository locally and add the manifests structure above
3. Create a `.gitlab-ci.yml` file for ArgoCD webhook notifications:
```yaml
argocd-sync:
  image: alpine:3.18
  before_script:
    - apk add --no-cache curl
  script:
    - |
      curl -XPOST -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ARGOCD_AUTH_TOKEN" \
        "$ARGOCD_WEBHOOK_URL?path=overlays/production" || true
  only:
    - main
```
4. Push the manifests to the repository

## 8. ArgoCD Setup

### 8.1 Installing ArgoCD

```bash
# Create namespace
kubectl create namespace argocd

# Install ArgoCD
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.8.0/manifests/install.yaml

# Create namespaces for applications
kubectl create namespace staging
kubectl create namespace production

# Set up TLS ingress for ArgoCD
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-server
  namespace: argocd
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - argocd.yourdomain.com
    secretName: argocd-server-tls
  rules:
  - host: argocd.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              name: http
EOF
```

### 8.2 Get Initial Password

```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

### 8.3 ArgoCD Projects and Applications

**argocd/projects/go-app-project.yaml**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: go-app
  namespace: argocd
spec:
  description: Go Application Project
  sourceRepos:
  - https://gitlab.com/yourusername/go-app-manifests.git
  destinations:
  - server: https://kubernetes.default.svc
    namespace: staging
  - server: https://kubernetes.default.svc
    namespace: production
  clusterResourceWhitelist:
  - group: '*'
    kind: Namespace
  namespaceResourceWhitelist:
  - group: '*'
    kind: '*'
  roles:
  - name: ci-role
    description: Role for CI pipeline to trigger syncs
    policies:
    - p, proj:go-app:ci-role, applications, sync, go-app/*, allow
    - p, proj:go-app:ci-role, applications, get, go-app/*, allow
    groups:
    - system:serviceaccounts:go-app-ci
```

**argocd/applications/go-app-staging.yaml**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: go-app-staging
  namespace: argocd
  finalizers:
  - resources-finalizer.argocd.argoproj.io
spec:
  project: go-app
  source:
    repoURL: https://gitlab.com/yourusername/go-app-manifests.git
    targetRevision: HEAD
    path: overlays/staging
    directory:
      recurse: true
  destination:
    server: https://kubernetes.default.svc
    namespace: staging
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
    - PrunePropagationPolicy=foreground
    - PruneLast=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
  ignoreDifferences:
  - group: apps
    kind: Deployment
    jsonPointers:
    - /spec/replicas
```

**argocd/applications/go-app-production.yaml**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: go-app-production
  namespace: argocd
  finalizers:
  - resources-finalizer.argocd.argoproj.io
spec:
  project: go-app
  source:
    repoURL: https://gitlab.com/yourusername/go-app-manifests.git
    targetRevision: HEAD
    path: overlays/production
    directory:
      recurse: true
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
    - PrunePropagationPolicy=foreground
    - PruneLast=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
  ignoreDifferences:
  - group: apps
    kind: Deployment
    jsonPointers:
    - /spec/replicas
```

### 8.4 Deploying ArgoCD Applications

```bash
# Apply project and applications
kubectl apply -f argocd/projects/
kubectl apply -f argocd/applications/

# Create service account token for GitLab CI webhook
kubectl create serviceaccount argocd-webhook -n argocd
kubectl create rolebinding argocd-webhook-binding -n argocd \
  --role=argocd-webhook \
  --serviceaccount=argocd:argocd-webhook

# Get token for GitLab CI
kubectl -n argocd create token argocd-webhook --duration=87600h > argocd-webhook-token.txt

# Set up GitLab CI variables
# AROCD_WEBHOOK_URL = https://argocd.yourdomain.com/api/webhook
# AROCD_AUTH_TOKEN = <contents of argocd-webhook-token.txt>
```

## 9. End-to-End Workflow

Let's understand the complete workflow from code change to deployment:

1. **Developer pushes code to GitLab**
   - GitLab sends a webhook to Tekton EventListener (with TLS and payload verification)
   - Tekton Trigger extracts parameters from the webhook payload
   - TriggerTemplate creates a PipelineRun with the extracted parameters

2. **Tekton CI Pipeline Executes**
   - **Fetch Source Task**: Clones the repository and checks out the specific commit
   - **Test Go Task**: Runs unit tests, static analysis, and security scans
   - **Build Go Task**: Compiles the Go application with version information
   - **Build and Push Image Task**: Builds a Docker image and pushes it to GitLab registry with digest
   - **Create Merge Request Task**: Creates a merge request to update the manifests repository with the new image digest

3. **Merge Request Approval**
   - Developer or approver reviews the merge request
   - Merge request is approved and merged to main branch
   - GitLab CI triggers ArgoCD webhook to sync the application

4. **ArgoCD Deployment**
   - ArgoCD detects changes in the manifests repository
   - ArgoCD syncs the new manifests to the cluster (staging first, then production)
   - The new image is deployed with health checks and rollbacks if needed

5. **Verification**
   - ArgoCD monitors the health of the deployed application
   - If there are issues, ArgoCD can automatically rollback to a previous version
   - Metrics and logs are collected for monitoring

## 10. Production Considerations

### Security

- **Secrets Management**: Use Kubernetes secrets with proper RBAC. Consider HashiCorp Vault for production secrets.
- **Network Policies**: Implement network policies to restrict traffic between components:
  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: NetworkPolicy
  metadata:
    name: tekton-restrict
    namespace: go-app-ci
  spec:
    podSelector: {}
    ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: kube-system
    egress:
    - to:
      - namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: argocd
  ```
- **Image Scanning**: Integrate Trivy or Clair for container image scanning in the pipeline.
- **Pod Security**: Use Pod Security Standards with restricted profiles.

### Scalability

- **Resource Limits**: Set appropriate resource requests and limits for all components.
- **Pipeline Optimization**: Use parallel execution where possible and cache dependencies.
- **Storage**: Use fast storage classes for PVCs and implement cleanup policies.

### Monitoring and Observability

- **ArgoCD Metrics**: Enable Prometheus metrics for ArgoCD:
  ```yaml
  apiVersion: argoproj.io/v1alpha1
  kind: Application
  metadata:
    annotations:
      prometheus.io/scrape: "true"
      prometheus.io/port: "8082"
  ```
- **Tekton Monitoring**: Install Tekton dashboard and monitoring:
  ```bash
  kubectl apply -f https://raw.githubusercontent.com/tektoncd/dashboard/main/config/release/monitoring.yaml
  ```
- **Logging**: Configure centralized logging with Loki or ELK stack.
- **Alerting**: Set up alerts for pipeline failures and deployment issues.

### High Availability

- **ArgoCD HA**: Deploy ArgoCD in HA mode:
  ```bash
  kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.8.0/manifests/ha/install.yaml
  ```
- **Tekton HA**: Configure multiple replicas for EventListeners and controllers.
- **Database Backup**: Regularly backup ArgoCD's database using Velero or native backup tools.

## 11. Troubleshooting

### Common Tekton Issues

1. **Task Fails Due to Missing Permissions**
   - Check RBAC permissions for the service account
   - Verify that the service account has access to required resources
   - Use `kubectl describe rolebinding tekton-pipeline-binding -n go-app-ci` to verify bindings

2. **Workspace Issues**
   - Check PVC status: `kubectl get pvc -n go-app-ci`
   - Verify storage class availability
   - Check for sufficient disk space

3. **Git Authentication Failures**
   - Verify secret contents: `kubectl get secret gitlab-credentials -n go-app-ci -o yaml`
   - Check token permissions in GitLab
   - Ensure the token has not expired

### Common ArgoCD Issues

1. **Sync Failures**
   - Check application status: `argocd app get go-app-production`
   - View sync history: `argocd app history go-app-production`
   - Check logs: `kubectl logs -l app.kubernetes.io/name=argocd-server -n argocd`

2. **Resource Conflicts**
   - Use `argocd app get go-app-production --show-operation`
   - Check for resource ownership conflicts
   - Verify namespace exists and has proper permissions

3. **Health Check Failures**
   - Check application logs: `kubectl logs -l app=go-app -n production`
   - Verify health check endpoints are accessible
   - Check resource utilization and limits

## Cleanup Strategy

To prevent resource accumulation, implement automatic cleanup:

```yaml
# cleanup-pipeline.yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: cleanup-resources
  namespace: go-app-ci
spec:
  tasks:
  - name: cleanup-pvcs
    taskRef:
      name: kubectl-task
    params:
    - name: command
      value: "delete pvc --field-selector=status.phase=Released --all"
```

Schedule this pipeline using Kubernetes CronJobs:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: tekton-cleanup
  namespace: go-app-ci
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: tekton-pipeline
          containers:
          - name: cleanup
            image: bitnami/kubectl
            command: ["kubectl", "delete", "pvc", "--field-selector=status.phase=Released", "--all", "-n", "go-app-ci"]
          restartPolicy: OnFailure
```

## Conclusion

This comprehensive guide provides a production-ready CI/CD pipeline using Tekton and ArgoCD for a Go application. It addresses critical security concerns, follows GitOps principles, and includes proper testing and monitoring.

The pipeline is designed to be:
- **Secure**: Least-privilege RBAC, non-root containers, TLS everywhere
- **Reliable**: Automated testing, image scanning, and rollback capabilities
- **Auditable**: All changes go through merge requests with human approval
- **Maintainable**: Clear separation of concerns and documented workflows

By following this guide, you'll have a robust, cloud-native CI/CD pipeline that automatically builds, tests, and deploys your applications whenever changes are pushed to your GitLab repository, while maintaining security and compliance standards.
