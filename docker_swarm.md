# Docker Swarm — Super-Complete Practical Tutorial

*Senior DevOps mentor → teaching a junior DevOps engineer.*
We'll treat this as a real hands-on course: realistic IPs, copy-paste commands, complete files, and real workflows. Work in a lab or isolated network (example IPs used: `192.168.1.10`, `192.168.1.11`, `192.168.1.12`, `192.168.1.20` for monitoring/logging). **Assume Ubuntu 22.04/24.04 hosts with Docker Engine installed.**

---

# 1. Containerization Basics (fast recap + practical)

## What you need locally on each host

```bash
# install docker (quick method)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# enable remote API on managers only if you need it (careful with security)
# example: edit /etc/docker/daemon.json to include {"hosts":["unix:///var/run/docker.sock","tcp://0.0.0.0:2375"]}
sudo systemctl restart docker
```

## Dockerfile example (real app)

`./app/Dockerfile`

```Dockerfile
# multi-stage build
FROM node:20-slim AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

FROM node:20-slim
WORKDIR /app
COPY --from=builder /app . 
EXPOSE 3000
CMD ["node", "server.js"]
```

`.dockerignore`

```
node_modules
.git
.env
```

Build & test locally:

```bash
docker build -t myregistry.local:5000/golang-app:1.0 ./app
docker run --rm -p 3000:3000 myregistry.local:5000/golang-app:1.0
# visit http://192.168.1.10:3000
```

---

# 2. Docker Swarm Architecture & Cluster Formation

## Nodes (lab topology)

* `192.168.1.10` — **manager1** (leader)
* `192.168.1.11` — **worker1**
* `192.168.1.12` — **worker2**
  *(Optional HA: add 2 more managers for odd number quorum)*

## Initialize Swarm (on `manager1`)

```bash
# init swarm and advertise manager IP
docker swarm init --advertise-addr 192.168.1.10
# get join tokens
docker swarm join-token worker   # prints worker token + join command
docker swarm join-token manager  # prints manager token + join command
```

## Join workers (on `worker1` & `worker2`)

Example printed join command:

```bash
docker swarm join --token SWMTKN-1-xxxx 192.168.1.10:2377
```

Verify cluster (on manager):

```bash
docker node ls
```

## Promote a worker to manager (if needed)

```bash
# on manager
docker node promote <NODE-ID>
```

**Notes on manager quorum:** keep an *odd* number of managers (1,3,5). Managers hold Raft data; too many managers reduces performance.

---

# 3. Services, Stacks, and Docker Compose in Swarm

## Service vs Container vs Stack

* `docker run` → single container
* `docker service create` → Swarm-managed service (replicated or global)
* `docker stack deploy -c docker-compose.yml mystack` → deploy a *stack* (multiple services) across the cluster using a Compose file (version `3.2+`).

## Example `docker-compose.yml` for Swarm stack

`docker-compose.yml`

```yaml
version: "3.9"

networks:
  app_net:
    driver: overlay
    attachable: true

services:
  web:
    image: myregistry.local:5000/golang-app:1.0
    networks:
      - app_net
    ports:
      - "80:3000"
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
      resources:
        limits:
          cpus: "0.50"
          memory: 512M
        reservations:
          cpus: "0.10"
          memory: 128M
    secrets:
      - db_pass
    configs:
      - source: nginx_conf
        target: /etc/nginx/nginx.conf

  redis:
    image: redis:6.2
    networks:
      - app_net
    deploy:
      replicas: 1

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD_FILE: /run/secrets/db_pass
      MYSQL_DATABASE: appdb
    volumes:
      - mysql_data:/var/lib/mysql
    networks:
      - app_net
    deploy:
      replicas: 1

volumes:
  mysql_data:

secrets:
  db_pass:
    external: false

configs:
  nginx_conf:
    file: ./nginx.conf
```

Deploy the stack:

```bash
docker stack deploy -c docker-compose.yml production
docker stack ls
docker stack services production
docker stack ps production
```

Scale `web`:

```bash
docker service scale production_web=6
```

Remove stack:

```bash
docker stack rm production
```

---

# 4. Joining Managers & Workers — Practical Flow

### Adding a new worker

1. On manager: `docker swarm join-token worker` → copy printed command.
2. On new node: run printed `docker swarm join --token ... 192.168.1.10:2377`.
3. Verify on manager: `docker node ls`.

### Adding a new manager

1. On manager: `docker swarm join-token manager` → copy printed command.
2. On new node: run printed `docker swarm join --token ... 192.168.1.10:2377`.
3. Verify: `docker node ls`.

### Demote / Remove nodes

```bash
docker node demote <NODE-ID>
docker node rm <NODE-ID>
```

If a manager is permanently gone, remove it and check quorum.

---

# 5. Deployment Strategies — Rolling Updates / Rollbacks / Blue-Green / Canary

## Rolling update (service)

```bash
docker service update \
  --image myregistry.local:5000/golang-app:2.0 \
  --update-parallelism 1 \
  --update-delay 10s \
  production_web
```

## Rollback

```bash
docker service rollback production_web
```

## Blue/Green (manual)

1. Deploy `myapp:green` as `production_web_green` (or `production_web:v1` label).
2. Deploy `myapp:blue` as `production_web_blue` with new image.
3. Switch proxy / VIP or update `nginx` config to route traffic to blue.
4. Remove green when validated.

## Canary (gradual)

1. Deploy a small subset of new version (e.g., 1 replica):

```bash
docker service create --name web_canary --replicas 1 --network app_net myregistry.local:5000/golang-app:canary
```

2. Use your load balancer or ingress to send a small percentage of traffic to canary.
3. If okay, increase replicas or switch main service to new image with rolling update.

**Pro tip:** use healthchecks in Compose to prevent routing to unhealthy containers:

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
  interval: 30s
  timeout: 3s
  retries: 3
```

---

# 6. Networking — Overlay, Ingress, and Service Discovery

## Create overlay network (encrypted)

```bash
docker network create -d overlay --opt encrypted app_net
```

## Ingress & Published ports

* `-p host_port:container_port` in service `deploy` will create routing mesh and publish via all nodes on that port; internal load balancing will route to tasks.

Example:

```bash
docker service create --name nginx --replicas 2 -p 80:80 nginx:alpine
```

## DNS & service discovery

From any container on same overlay network:

```bash
# example: `web` resolves to service VIP
curl http://web:3000/health
```

## Network security & segmentation

* Use multiple overlay networks to segment services (frontend vs backend).
* Use `endpoint_mode: dnsrr` for advanced service routing if you want DNS round-robin not VIP (useful for stateful/DB service discovery).

---

# 7. Storage Management

## Volumes in Swarm

Create named volume:

```bash
docker volume create mysql_data
```

Use in Compose (already shown). Volumes are managed by Docker local driver by default (on the node where the task runs). For multi-node persistence you need a shared filesystem.

## NFS backing (example)

On NFS server `192.168.1.50` with export `/srv/nfs/mysql`

* Create a volume with driver `local` and NFS options:

```bash
docker volume create \
  --name mysql_data \
  --opt type=nfs \
  --opt o=addr=192.168.1.50,rw \
  --opt device=:/srv/nfs/mysql
```

Compose driver options:

```yaml
volumes:
  mysql_data:
    driver: local
    driver_opts:
      type: "nfs"
      o: "addr=192.168.1.50,rw"
      device: ":/srv/nfs/mysql"
```

## Alternatives for production

* **GlusterFS**, **Ceph/RBD**, **Longhorn** (K8s), or enterprise storage drivers. Choose based on SLA and performance.

---

# 8. Monitoring & Logging (Production-ready stacks)

## Monitoring: Prometheus + Grafana + Node exporter (all as Swarm services)

Example `monitoring-stack.yml`

```yaml
version: "3.9"
services:
  prometheus:
    image: prom/prometheus:latest
    configs:
      - source: prometheus_yml
        target: /etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    volumes:
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana
    ports: ["3000:3000"]
    volumes:
      - grafana_data:/var/lib/grafana

  node-exporter:
    image: prom/node-exporter
    deploy:
      mode: global
    networks:
      - monitor_net

networks:
  monitor_net:
    driver: overlay

configs:
  prometheus_yml:
    file: ./prometheus.yml

volumes:
  prometheus_data:
  grafana_data:
```

Deploy:

```bash
docker stack deploy -c monitoring-stack.yml monitoring
```

## Logging: EFK (Elasticsearch, Fluentd, Kibana) or Loki + Promtail + Grafana

* Run Fluentd/Fluent Bit on each node (as global service) to forward logs to Elasticsearch or Loki.
* Example: Deploy Fluent Bit as global daemon to read container stdout and forward to ES `192.168.1.20:9200`.

---

# 9. Security — Image Scanning, Secrets, RBAC ideas

## Secrets (Swarm native)

Create secret:

```bash
echo "supersecret" | docker secret create db_pass -
```

Use in Compose (already shown). Secrets are stored encrypted at rest and delivered to containers only when used.

## Scan images with **Trivy** (supply commands and CI integration)

Trivy is an open-source scanner.

Install trivy locally (on dev/CI machine):

```bash
sudo apt-get install -y wget apt-transport-https gnupg lsb-release
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
echo deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main | sudo tee /etc/apt/sources.list.d/trivy.list
sudo apt update && sudo apt install -y trivy
```

Scan image example:

```bash
trivy image --severity HIGH,CRITICAL myregistry.local:5000/golang-app:1.0
```

Integrate in GitLab CI:

```yaml
stages:
  - build
  - scan
  - push
  - deploy

scan:
  stage: scan
  script:
    - docker build -t myregistry.local:5000/golang-app:$CI_COMMIT_SHA .
    - trivy image --exit-code 1 --severity HIGH,CRITICAL myregistry.local:5000/golang-app:$CI_COMMIT_SHA
  allow_failure: false
```

If Trivy finds HIGH/CRITICAL, pipeline fails. You can also use Trivy to produce SBOMs, JSON output for vulnerability tracking.

## Hardening

* Use non-root containers where possible (add `USER` in Dockerfile).
* Minimize image layers and tools.
* Use image signing and registry authentication (not covered in-depth here).
* For RBAC and team separation, Docker Swarm on OSS lacks fine RBAC — Docker Enterprise or external orchestration is required.

---

# 10. Scaling & Performance

## Horizontal scaling

```bash
docker service scale production_web=10
# or update
docker service update --replicas 10 production_web
```

## Resource limits & reservations (use in Compose `deploy.resources`)

Example shown earlier: CPU and memory limits.

## Node labels and affinities

Label node:

```bash
docker node update --label-add storage=ssd 192.168.1.11
```

Use constraint in Compose:

```yaml
deploy:
  placement:
    constraints:
      - node.labels.storage == ssd
```

## Performance tuning checklist

* Use appropriate MTU for overlay networks if your infrastructure uses jumbo frames.
* Monitor network throughput and storage IOPS.
* Co-locate stateful services with high I/O on fast storage.
* For high throughput, avoid network encryption on overlays if not needed (but be careful).

---

# 11. CI/CD Integration & Example Pipelines

## Example: GitLab CI to build, scan, push, and deploy (uses Docker CLI to manager)

`.gitlab-ci.yml`

```yaml
stages:
  - build
  - scan
  - push
  - deploy

build:
  image: docker:24
  services:
    - docker:dind
  script:
    - docker build -t registry.gitlab.com/myteam/myapp:$CI_COMMIT_SHA .
    - docker push registry.gitlab.com/myteam/myapp:$CI_COMMIT_SHA

scan:
  image: aquasec/trivy:latest
  script:
    - trivy image --exit-code 1 --severity HIGH,CRITICAL registry.gitlab.com/myteam/myapp:$CI_COMMIT_SHA

deploy:
  image: docker:24
  services:
    - docker:dind
  script:
    - docker login -u $DOCKER_USER -p $DOCKER_PASSWORD $CI_REGISTRY
    - docker stack deploy -c docker-compose.yml production
```

**Note:** In production, use secure methods to publish across private networks (e.g., run pipeline on a runner with access to the manager via SSH & run `docker stack deploy` there).

---

# 12. Troubleshooting & Common Issues

## Useful commands

```bash
# list services
docker service ls

# list tasks for a service
docker service ps production_web

# view logs (service-level)
docker service logs production_web

# view container logs (task-level)
docker logs <container-id>

# events (real-time)
docker events

# node info
docker node inspect <node-id> --pretty

# check swarm status / leader
docker node ls
```

## Common problems & fixes

* **Service stuck on `Pending`/`Rejected`** → Check placement constraints and node availability.
* **Container fails to start** → `docker service ps --no-trunc <service>` then inspect logs `docker logs <task-container-id>`.
* **Networking issues (no DNS)** → Verify overlay networks exist and that containers are attached. Check `docker network inspect`.
* **Storage/volume binding errors** → Verify volume driver and mount points exist on target nodes.
* **Token/Node join fail** → Network connectivity to port `2377` needs to be open (managers listen there).

---

# 13. Best Practices & IaC (Ansible / Terraform)

## IaC with Ansible — Example snippet to deploy a stack

`deploy-stack.yml` (Ansible)

```yaml
- hosts: managers
  become: true
  tasks:
    - name: Transfer docker-compose file
      copy:
        src: ./docker-compose.yml
        dest: /tmp/docker-compose.yml
    - name: Deploy stack
      shell: docker stack deploy -c /tmp/docker-compose.yml production
      args:
        warn: false
```

Run:

```bash
ansible-playbook -i hosts deploy-stack.yml
```

## Terraform (high level)

* Use cloud provider Terraform modules to provision VMs.
* Use `remote-exec` to `docker swarm join`.
* Use Ansible provisioner or Terraform `local-exec` to deploy stacks.

**Tip:** Keep your Compose files in Git, and use CI to deploy them to manager(s). Use Ansible to manage node packages, network mounts (NFS), and host configuration.

---

# 14. Advanced Topics

## Healthchecks + graceful shutdown

Set `stop_grace_period` in Compose:

```yaml
deploy:
  update_config:
    order: start-first
  stop_grace_period: 1m30s
```

## Backup & Restore volumes

* For NFS: snapshot `/srv/nfs/mysql` or use database dumps (best).
* For local volumes: `docker run --rm -v mysql_data:/data busybox tar czf /backup/mysql_data.tgz /data` (but local volumes are node-specific; prefer shared storage for backups).

## Upgrading Docker & Rolling upgrades of managers

* Upgrade managers one-by-one, demote if necessary, then re-promote.
* Always check raft status and maintain quorum.

---

# 15. Security deeper — image scanning, runtime scanning, secrets

## Trivy (deeper)

* **Local scans:** `trivy image --format json -o report.json myimage:tag`
* **SBOM:** `trivy image --format cyclonedx --output sbom.cdx.json myimage:tag`
* **CI usage:** fail builds on severity threshold or produce reports for security team.

## Runtime scanning (optional)

* Use Falco (Sysdig) as a daemon to detect anomalous behaviors (container privilege escalation, shell access, etc.).

## Secrets lifecycle

* Create secrets via CLI or Compose (external secrets recommended).
* Rotate secrets by creating a new secret and updating services to use it, then remove old secret when no task uses it.

---

# 16. Full Hands-On Example (end-to-end)

### Setup:

1. Create 3 Ubuntu VMs: `192.168.1.10` (manager), `192.168.1.11` (worker1), `192.168.1.12` (worker2). Install Docker on all.
2. On manager:

   ```bash
   docker swarm init --advertise-addr 192.168.1.10
   ```
3. On worker nodes: join with the printed token.
4. Create `docker-compose.yml` (from earlier) on manager.
5. Deploy:

   ```bash
   docker stack deploy -c docker-compose.yml production
   ```
6. Monitor:

   ```bash
   docker service ls
   docker service ps production_web
   docker stack services production
   docker service logs production_web
   ```

### Add monitoring:

* Deploy monitoring stack as `monitoring` (Prometheus + Grafana) — use provided `monitoring-stack.yml`.

### CI/CD:

* Add GitLab CI `deploy` job to push new image and run `docker stack deploy` on manager via secure runner.

### Scan images before deploy:

* Add Trivy job in CI for `HIGH` or `CRITICAL` vulnerabilities.

---

# 17. Learning checklist & practice exercises

1. Build and push a simple app image to local registry.
2. Init Swarm, join two workers.
3. Deploy the `docker-compose.yml` stack and scale `web` to 5 replicas.
4. Simulate a rolling update with a new image and rollback.
5. Configure an overlay network and verify DNS between containers.
6. Create an NFS volume and attach MySQL service to it.
7. Add Prometheus/Grafana monitoring and view metrics.
8. Run Trivy on your built image and remediate any HIGH findings.
9. Write an Ansible playbook that deploys the stack and can be used by CI runner.
10. Simulate a manager failure and recover (remove node, rejoin).

---

# 18. Quick Reference Commands

```bash
docker swarm init --advertise-addr 192.168.1.10
docker swarm join --token SWMTKN-1-... 192.168.1.10:2377
docker stack deploy -c docker-compose.yml production
docker service ls
docker service ps <service>
docker service logs <service>
docker node ls
docker node update --label-add storage=ssd <node>
docker secret create db_pass ./db_pass.txt
docker service update --image myimage:v2 --update-parallelism 1 --update-delay 10s production_web
trivy image --severity HIGH,CRITICAL myregistry:tag
```

---

# 19. Final Advice from a Mentor

* Use **compose v3+** and `docker stack` to run apps in Swarm; keep local developer `docker-compose` compatible.
* Keep infra immutable: use images and configuration as code.
* Integrate **Trivy** early in CI to detect vulnerabilities before deployment.
* Use **secrets** and private registries, and secure manager endpoints (do not expose Docker remote API to the public).
* Prefer orchestrating `docker stack deploy` from a trusted CI runner with least privileges.
* If you need enterprise-level RBAC and advanced security, consider Kubernetes (but Swarm is simpler and great for smaller fleets and rapid prototypes).

---

If you want, next I can:

* Provide a **full git repo** layout (Dockerfile, compose, nginx.conf, prometheus.yml, Ansible playbook).
* Generate a **step-by-step lab script** to create the three VMs in VirtualBox or cloud and run everything end-to-end.
  Tell me which you want and I’ll produce the complete repo & deploy script right now.
