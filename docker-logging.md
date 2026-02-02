
---

# Docker Storage & Logging Management Guide

## 1. The Problem Statement: "Disk Full."

A common issue in Docker environments is the root partition (`/`) filling up unexpectedly. Upon investigation using `du -sh`, users often find that `/var/lib/docker/overlay2` is consuming massive amounts of space.

In many cases, specifically with Nginx or high-traffic applications, the culprit is log accumulation inside the container's writable layer, which is not managed or rotated by default.

## 2. Core Concepts: How Docker Stores Data

To manage Docker effectively, you must understand how it handles the filesystem and logs.

### A. Overlay2: Diff vs. Merged

When you look inside an `overlay2` directory, you see several folders. The most important are:

- **`diff`**: This is the actual physical data. It contains every change made to the container (new files, modified logs, etc.) that isn't in the original image.
- **`merged`**: This is a virtual view. It combines the read-only image layers and the `diff` layer into one folder.

> **The Illusion of Size**:  
> If you run `du` on the parent folder, it might show 18GB used (9GB in `diff` and 9GB in `merged`). This is a lie. The data exists only once in `diff`. `merged` is just a mount point.  
> **Do not delete these manually**; you will corrupt your container.

### B. Internal Logs vs. Docker Logs (The Critical Difference)

This is where most beginners get confused. There are two ways logs are stored:

#### Internal Application Logs (The Danger Zone)

- **What they are**: Files written by the app to a path like `/var/log/nginx/access.log` inside the container.
- **Docker's Role**: Docker has zero control over these. It doesn't know they exist. They will grow until the disk is 100% full.
- **Management**: You must use `truncate` or internal log rotation.

#### Docker Logs (Standard Streams)

- **What they are**: Data the app sends to `stdout` (Standard Output) and `stderr` (Standard Error).
- **Docker's Role**: Docker captures this stream and saves it to a JSON file on the host machine.
- **Management**: You can limit these globally via `daemon.json` or in `docker-compose`.

## 3. Immediate Fix: Freeing Up Space

If your disk is at 100%, you need immediate relief without breaking the application.

### The Best Method: Truncation

Use the `truncate` command or shell redirection to empty the file **without deleting it**. Deleting (`rm`) a log file while the app is running won't free space because the process keeps the "file handle" open.

```bash
# Option 1: The Cleanest way (Sets size to 0 bytes)
truncate -s 0 /path/to/large/log/file.log

# Option 2: The Shortcut (Fastest in terminal)
> /path/to/large/log/file.log

# Note: Avoid 'echo " " > file' as it leaves a 1-byte newline character behind.
```

## 4. Long-Term Solutions & Configurations

### Strategy A: Redirecting to Stdout (The "Docker Way")

To allow Docker to manage your logs, you should tell Nginx to send logs to the output streams instead of a file.

**Nginx Config (`nginx.conf`):**

```nginx
http {
    # Define a custom format if needed
    log_format custom_json '{ "time": "$time_iso8601", "remote_addr": "$remote_addr", "status": "$status" }';

    # Redirect access and error logs to Docker's collector
    access_log /dev/stdout custom_json;
    error_log /dev/stderr warn;
}
```

### Strategy B: Setting Global Log Limits

Once logs are sent to `stdout`, you must tell Docker to rotate them so they don't grow forever.

**Docker Daemon Config (`/etc/docker/daemon.json`):**

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "500m",
    "max-file": "3"
  }
}
```

After applying, run:

```bash
systemctl restart docker
```

**Per-Service Config (`docker-compose.yml`):**

```yaml
services:
  nginx:
    image: nginx:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## 5. Moving Docker Storage to a Larger Partition

If your root (`/`) is small but you have a large data partition (e.g., `/mnt`), move the entire Docker directory there.

### Migration Steps:

1. **Stop Services**:
   ```bash
   systemctl stop docker.socket
   systemctl stop docker
   ```

2. **Edit Config**: Set `"data-root": "/mnt/docker"` in `/etc/docker/daemon.json`.

3. **Transfer Data**:
   ```bash
   rsync -aP /var/lib/docker/ /mnt/docker/
   ```
   The `-a` flag is vital to keep permissions.

4. **Start Services**:
   ```bash
   systemctl start docker
   ```

5. **Verify**:
   ```bash
   docker info | grep "Root Dir"
   ```

## 6. Summary Checklist for Success

- [ ] Never delete files inside `overlay2` manually using `rm`.
- [ ] Always set `max-size` and `max-file` in your logging configuration.
- [ ] Prefer logging to `/dev/stdout` over writing to internal files.
- [ ] Monitor disk usage on both `/` and your data partitions.
- [ ] Use `rsync` if moving Docker data to preserve file ownership.

### Toolbox:

To see the unique space used by Docker:

```bash
docker system df
```

This command is "Docker-aware." It understands the OverlayFS structure and will correctly identify the shared size and the unique writable size of your containers.

---

## Better Approach

Below is the most practical, enterprise-grade, low-risk, and battle-tested approach for managing stdout/stderr logs of Docker containers, with proper rotation, retention, and forward compatibility. (But the final solution is given in the last section—so keep reading or jump to **Section 7** for the final decision.)

> I am deliberately giving you one clear recommendation (so far), not a menu of options.

### Executive Summary (Straightforward Recommendation)

**Use Docker with the `journald` logging driver + systemd-managed rotation + optional centralized shipping.**

This is the cleanest, safest, and most enterprise-friendly way to manage container stdout/stderr without touching application code, without log file corruption, and with predictable retention policies.

### Why `journald` Is the Enterprise Default

| Criterion               | `json-file`         | `journald`          |
|------------------------|---------------------|---------------------|
| Time-based rotation    | ❌ No               | ✅ Yes              |
| Size-based retention   | ⚠️ Limited          | ✅ Yes              |
| Crash-safe             | ❌ Risky            | ✅ Yes              |
| Signal-safe (no copytruncate) | ❌ No      | ✅ Yes              |
| Centralized logging ready | ❌ Weak          | ✅ Native           |
| Production support (RHEL, Ubuntu, SLES) | ⚠️ | ✅ Standard     |

> **Important**: Docker itself officially recommends not relying on `json-file` for production at scale.

### Final Architecture So Far (What You Will Actually Run)

```
Container stdout/stderr
↓
Docker logging driver (journald)
↓
systemd-journald
↓
• Daily rotation
• 14–30 days retention
• Optional forwarding to ELK / Loki / Splunk
```

### Concrete Implementation (Step-by-Step)

#### 1. Configure Docker to Use `journald`

`/etc/docker/daemon.json`:

```json
{
  "log-driver": "journald"
}
```

Restart Docker:

```bash
systemctl restart docker
```

From this point forward, every container stdout/stderr goes to journald.

#### 2. Configure Log Rotation & Retention (Time + Size)

`/etc/systemd/journald.conf`:

```ini
[Journal]
Storage=persistent
SystemMaxUse=5G
SystemKeepFree=1G
MaxRetentionSec=14day
Compress=yes
```

Apply changes:

```bash
systemctl restart systemd-journald
```

#### What This Guarantees

- Logs rotate automatically
- Logs expire after 14 days
- Disk usage is capped
- Compression is enabled
- No application restarts needed

#### 3. How You Access Container Logs (Operational Use)

- List containers:
  ```bash
  docker ps
  ```

- View logs of a container:
  ```bash
  journalctl CONTAINER_NAME=my-app
  ```

- Follow logs:
  ```bash
  journalctl -fu docker
  ```

- Filter by container ID:
  ```bash
  journalctl CONTAINER_ID=abc123
  ```

### Production Use Case Example

**Scenario**

- 40 Docker containers
- API + workers + cron jobs
- Traffic spikes
- Compliance requires 14-day retention
- Zero tolerance for log loss

**What Happens**

- App writes logs to stdout
- Docker streams logs to journald
- journald rotates logs daily
- Old logs removed after 14 days
- Disk never fills up
- Logs are queryable by container, service, unit, time

### Optional: Centralized Logging (Enterprise Add-on)

When ready, add one agent (no Docker change):

| Tool           | How               |
|----------------|-------------------|
| ELK            | journalbeat       |
| Grafana Loki   | promtail          |
| Splunk         | splunk-forwarder  |
| Datadog        | systemd integration |

All read from journald, not from Docker files.

### What NOT to Do (Common Anti-Patterns)

- ❌ `json-file` + `logrotate + copytruncate` → race conditions, log loss, corruption
- ❌ Per-container log files → unmanageable at scale
- ❌ Application-level rotation → breaks container portability

---

## Final Verdict (Clear and Direct)

**Best enterprise solution:**

- stdout/stderr only
- Docker → journald
- systemd handles rotation
- Optional centralized shipping

This is stable, auditable, scalable, and future-proof.

---

# Best Scenario in 7 steps:

### 1️⃣ journald approach — what actually happens

**❓ Do we still need Docker `json-file`?**  
No.

Once you set:

```json
{
  "log-driver": "journald"
}
```

Docker does **not** create:
```
/var/lib/docker/containers/*/*.json
```
Those files no longer exist.

**❓ Is there any log file at all?**  
Not in the traditional sense.  
Logs are written only to the host’s systemd journal, stored in:
```
/var/log/journal/
```
They are binary journal files, not text files.

**❓ Does this depend on the container having journald/systemd?**  
Absolutely not. This is the key advantage.

- ✔ Containers:  
  - Can be Alpine  
  - Can be distroless  
  - Can have no systemd  
  - Can have no journald  

- ✔ Docker captures:  
  - stdout / stderr  

- ✔ Docker sends them to:  
  - host journald  

Your container image is completely irrelevant. This is **100% host-side logging**.

**❓ Does this rely on the container at all?**  
No.  
If the container crashes, logs are already persisted.

> **🔒 Important conclusion**  
> This is safer than `json-file` because:  
> - No file truncation  
> - No log corruption  
> - No dependency on container internals

---

### 2️⃣ Can we have different retention per app/container?

**❌ Direct answer**  
journald cannot do per-container retention policies.

Retention is global:
- `SystemMaxUse`
- `MaxRetentionSec`

**✅ Enterprise workaround (the correct one)**  
- Do global short retention on the host  
- and long retention in Elasticsearch  

**Example:**  
- journald: 3–7 days  
- Elasticsearch: 30 / 90 / 180 days  

> This is how RHEL / Ubuntu / large orgs do it.

**⚠️ What journald can do**  
You can filter and tag logs by:
- `CONTAINER_NAME`
- `CONTAINER_ID`
- `IMAGE_NAME`
- systemd unit

But retention is global, not per app.

---

### 3️⃣ Elasticsearch + Kibana with journald

> “Is it as easy as filebeat?”

Yes — but you use `journalbeat`, not filebeat.

**Correct Stack for This Architecture**

```
NestJS (stdout JSON logs)
↓
Docker (journald driver)
↓
systemd-journald (host)
↓
journalbeat
↓
Elasticsearch
↓
Kibana
```

---

### 4️⃣ Real-World Example (Concrete)

**Example App**

- NestJS API  
- Uses Winston / Watson logger  
- Logs JSON to stdout  

**Example log:**
```json
{
  "level": "info",
  "service": "orders-api",
  "requestId": "abc-123",
  "msg": "Order created",
  "userId": 42
}
```

#### Step 1 — Docker daemon

```json
{
  "log-driver": "journald"
}
```

Restart Docker.

#### Step 2 — journald retention (short)

```ini
[Journal]
SystemMaxUse=2G
MaxRetentionSec=7day
Compress=yes
```

#### Step 3 — Install Journalbeat

```bash
apt install journalbeat
# (or RPM equivalent)
```

#### Step 4 — journalbeat config (IMPORTANT PART)

`/etc/journalbeat/journalbeat.yml`:

```yaml
journalbeat.inputs:
- paths: []
  include_matches:
    - "_TRANSPORT=stdout"
    - "CONTAINER_NAME=orders-api"

processors:
  - decode_json_fields:
      fields: ["message"]
      target: ""
      overwrite_keys: true
  - add_fields:
      target: ""
      fields:
        environment: production
        app: orders-api

output.elasticsearch:
  hosts: ["http://10.10.10.20:9200"]
  index: "docker-orders-api-%{+yyyy.MM.dd}"

setup.kibana:
  host: "http://10.10.10.21:5601"
```

**What this does**

- ✔ Reads only container stdout  
- ✔ Filters only your app  
- ✔ Parses JSON log body  
- ✔ Stores structured fields in ES  
- ✔ Uses daily indices  

#### Step 5 — Start journalbeat

```bash
systemctl enable journalbeat
systemctl start journalbeat
```

---

### 5️⃣ Compare with your old Filebeat idea

| Aspect                  | `json-file` + filebeat        | `journald` + journalbeat |
|------------------------|-------------------------------|---------------------------|
| File dependency        | Yes                           | No                        |
| Container crash safety | ❌                            | ✅                        |
| Log rotation           | Hacky                         | Native                    |
| JSON parsing           | Yes                           | Yes                       |
| Kubernetes ready       | ⚠️                            | ✅                        |
| Enterprise grade       | ⚠️                            | ✅                        |

---

### 6️⃣ Answering your concern directly

> “Is this as easy as filebeat?”

**Yes, operationally. And safer architecturally.**

You are simply moving:
- `filebeat` → `journalbeat`
- `json files` → `journald`

---

### Final Recommendation (YET!)

✔ **Use:**
- Docker → journald  
- journald → journalbeat  
- journalbeat → Elasticsearch  
- Kibana for visualization  

✔ **Keep:**
- Short retention on host  
- Long retention in ES  

✔ **Do NOT:**
- Rely on container logging files  
- Rotate Docker json files  
- Bind logging to base image internals  

---

## 7️⃣ Final Approach (I Kept the best for last 😎)

Below is a new, production-grade section you can append directly to your handbook. It corrects deprecations, uses the current Elastic best practice, and is written to be explicit, copy-paste ready, and battle-tested—no abstractions, no hand-waving.

I am also explicitly aligning this with NestJS + Winston JSON logs, which is our future need for JSON parsing and field manipulation, and our existing Elasticsearch + Kibana stack.

### 7.1 Final, Correct, and Supported Architecture

> **Yes — Journalbeat is deprecated.**  
> The correct and supported solution is **Filebeat with the `journald` input**.

**Final Data Flow**

```
NestJS (JSON logs → stdout)
↓
Docker (`journald` logging driver)
↓
systemd-journald (host-level storage, rotation, retention)
↓
Filebeat (journald input)
↓
Elasticsearch
↓
Kibana
```

This architecture is:
- Fully supported by Elastic  
- Container-runtime agnostic  
- Safe against log corruption  
- Proven in large-scale Linux environments  

---

### 7.2 Why This Is the Right Final Design

#### Why NOT Journalbeat
- Officially deprecated by Elastic  
- No future features or fixes  
- Replaced one-to-one by Filebeat journal input  

#### Why Filebeat + journald Is Superior
- One agent for files + journal  
- Same processors (JSON parsing, enrichment, dropping fields)  
- ILM, backpressure, retries, and buffering  
- Supported long-term by Elastic  

---

### 7.3 Hard Requirements (Non-Negotiable)

1. **Application**  
   - Must log JSON to stdout  
   - No file-based logs inside containers  
   - Your NestJS + Winston setup already satisfies this.

**Example log emitted by NestJS:**
```json
{
  "level": "info",
  "service": "orders-api",
  "requestId": "req-92f1",
  "userId": 42,
  "msg": "Order created"
}
```

2. **Docker: journald Logging Driver**

`/etc/docker/daemon.json`:

```json
{
  "log-driver": "journald"
}
```

Apply:
```bash
systemctl restart docker
```

> From this moment:  
> - No `json-file` logs exist  
> - No `/var/lib/docker/containers/*/*.json`  
> - All stdout/stderr → systemd journal  

---

### 7.4 journald: Host-Level Rotation & Retention

**Purpose**  
- Prevent disk exhaustion  
- Short-term retention only  
- Elasticsearch is the long-term store  

`/etc/systemd/journald.conf`:

```ini
[Journal]
Storage=persistent
SystemMaxUse=3G
SystemKeepFree=1G
MaxRetentionSec=7day
Compress=yes
```

Apply:
```bash
systemctl restart systemd-journald
```

**Result**  
- Automatic rotation  
- Hard disk usage cap  
- Time-based expiration  
- Zero interaction with containers  

---

### 7.5 Filebeat (Journal Input) — Production Configuration

**Install Filebeat**

```bash
apt install filebeat
# (or RPM equivalent)
```

---

### 7.6 Filebeat Configuration (Explicit & Real-World)

**Goal**  
- Read only Docker container logs  
- Filter only your NestJS app  
- Parse JSON logs  
- Add/remove fields  
- Prepare for Elasticsearch filtering and dashboards  

`/etc/filebeat/filebeat.yml`:

```yaml
filebeat.inputs:
- type: journald
  id: docker-journald
  seek: cursor
  include_matches:
    - "_TRANSPORT=stdout"
    - "CONTAINER_NAME=orders-api"

processors:
  # Parse JSON emitted by NestJS/Winston
  - decode_json_fields:
      fields: ["message"]
      target: ""
      overwrite_keys: true
      add_error_key: true

  # Add global metadata
  - add_fields:
      target: ""
      fields:
        environment: production
        platform: docker
        app: orders-api

  # Remove noisy or useless fields
  - drop_fields:
      fields:
        - "_cursor"
        - "host"
        - "agent"
        - "ecs"
        - "log"
        - "input"
        - "container.runtime"

output.elasticsearch:
  hosts: ["http://10.10.10.20:9200"]
  index: "docker-orders-api-%{+yyyy.MM.dd}"

setup.kibana:
  host: "http://10.10.10.21:5601"
```

---

### 7.7 Important Clarification: JSON Parsing (Critical Understanding)

**Why `message`?**  
In journald:
- Docker stores stdout as a string  
- That string is placed in the `message` field  

**Example before parsing:**
```json
{
  "message": "{ \"level\": \"info\", \"service\": \"orders-api\", \"msg\": \"Order created\" }"
}
```

**When Parsing Happens**  
- Parsing occurs inside Filebeat  
- Before `drop_fields`  
- Before sending to Elasticsearch  

**After `decode_json_fields`:**
```json
{
  "level": "info",
  "service": "orders-api",
  "msg": "Order created",
  "environment": "production",
  "app": "orders-api"
}
```

> This is why field-based filtering in Kibana works perfectly.

---

### 7.8 Filtering Multiple Apps (Real Production Pattern)

**Example:**

```yaml
include_matches:
  - "_TRANSPORT=stdout"
  - "CONTAINER_NAME=orders-api"
  - "CONTAINER_NAME=payments-api"
```

Or deploy multiple Filebeat inputs per app for stricter isolation.

---

### 7.9 Operational Usage

- **View Logs Locally**  
  ```bash
  journalctl CONTAINER_NAME=orders-api
  ```

- **Follow Logs**  
  ```bash
  journalctl -fu docker
  ```

- **Verify Filebeat**  
  ```bash
  filebeat test config
  filebeat test output
  systemctl status filebeat
  ```

---

### 7.10 Retention Strategy (Correct Enterprise Model)

| Layer         | Retention      |
|---------------|----------------|
| journald      | 3–7 days       |
| Elasticsearch | 30–180 days    |
| Cold / Snapshot | S3 / Object Storage |

> Per-container retention is not done at journald level.  
> It is done in Elasticsearch via ILM (this is intentional and correct).

---

### 7.11 What You Should NEVER Do

- ❌ Write logs to files inside containers  
- ❌ Use `logrotate` on Docker JSON logs  
- ❌ Use `copytruncate`  
- ❌ Depend on application-level rotation  
- ❌ Parse logs inside containers  

---

### 7.12 Final Verdict (This Is Your Handbook Answer)

**This is the final, supported, and production-grade solution:**

```
NestJS → JSON logs → stdout
Docker → `journald` logging driver
systemd → rotation + retention
Filebeat (journald input) → parsing + enrichment
Elasticsearch → long-term storage
Kibana → filtering & visualization
```

This setup is:
- Enterprise standard  
- Explicitly supported  
- Safe under load  
- Easy to operate  
- Future-proof  


# Bonus Section:

Below is a **self-contained, production-grade handbook section** you can append directly to your documentation.
It is written as a **long-term reference for an enterprise DevOps team**, with **explicit explanations**, **copy-paste-ready configs**, and **real operational examples**.

---

# **8. systemd-journald (Journal) — Enterprise Handbook**

## **8.1 What journald Is (And What It Is Not)**

### **What it is**

`systemd-journald` is the **central log collection daemon** for modern Linux systems running systemd.

It:

* Collects logs from **kernel**, **system services**, **Docker containers**, and **user processes**
* Stores logs in a **binary, indexed format**
* Provides **structured metadata**, not just raw text
* Handles **rotation, retention, compression, and integrity**

### **What it is NOT**

* Not a log shipper (it does not send logs to ELK by itself)
* Not a text log file system
* Not per-application configurable in isolation

Think of journald as:

> **A reliable, crash-safe, structured log database on the host**

---

## **8.2 How journald Works Internally**

### **Log Sources**

journald collects logs from:

* `stdout` / `stderr` of systemd services
* Kernel (`dmesg`)
* Docker containers (via Docker journald driver)
* Syslog socket (`/dev/log`)

### **Data Model**

Each log entry is a **record with fields**, not a line of text.

Example fields:

* `MESSAGE`
* `_PID`
* `_UID`
* `_SYSTEMD_UNIT`
* `_BOOT_ID`
* `CONTAINER_NAME`
* `PRIORITY`
* `_SOURCE_REALTIME_TIMESTAMP`

This is why filtering is extremely powerful.

---

## **8.3 Where journald Stores Logs**

### **Volatile Storage**

```text
/run/log/journal/
```

* Stored in memory
* Lost on reboot
* Used if persistent storage is disabled

### **Persistent Storage (Production Default)**

```text
/var/log/journal/
```

To enable persistent storage:

```bash
mkdir -p /var/log/journal
systemctl restart systemd-journald
```

**Enterprise rule**:
✅ Always use **persistent** storage.

---

## **8.4 journald Storage Format (Critical Understanding)**

* Logs are stored as **binary `.journal` files**
* Indexed by time, fields, and boot ID
* Cannot be edited manually
* Corruption-resistant
* Crash-safe (no partial writes)

This design:

* Eliminates `copytruncate` problems
* Prevents file descriptor leaks
* Guarantees consistency

---

## **8.5 Main Configuration File**

### **File Location**

```text
/etc/systemd/journald.conf
```

This file controls **global behavior**.

---

## **8.6 Production-Grade journald Configuration (Copy-Paste Ready)**

### **Recommended Enterprise Baseline**

```ini
[Journal]
Storage=persistent
Compress=yes
Seal=yes

SystemMaxUse=5G
SystemKeepFree=1G
SystemMaxFileSize=500M

MaxRetentionSec=7day

RateLimitIntervalSec=30s
RateLimitBurst=10000

SyncIntervalSec=5m
```

Apply:

```bash
systemctl restart systemd-journald
```

---

## **8.7 Configuration Options — Explained (What You Must Know)**

### **Storage**

```ini
Storage=persistent
```

* `persistent`: logs survive reboot
* `volatile`: logs lost on reboot
* `auto`: persistent if directory exists

**Production rule**: always `persistent`.

---

### **Compress**

```ini
Compress=yes
```

* Compresses old journal files
* Reduces disk usage dramatically
* Negligible CPU impact

**Always enable in production.**

---

### **Seal**

```ini
Seal=yes
```

* Cryptographically seals logs
* Detects tampering
* Required for compliance environments

---

### **SystemMaxUse**

```ini
SystemMaxUse=5G
```

* Absolute cap for total journal disk usage
* Journald will delete oldest entries automatically

**This is your hard safety net.**

---

### **SystemKeepFree**

```ini
SystemKeepFree=1G
```

* Ensures free disk space remains
* Prevents root filesystem exhaustion

---

### **SystemMaxFileSize**

```ini
SystemMaxFileSize=500M
```

* Max size of a single journal file
* Helps with filesystem performance

---

### **MaxRetentionSec**

```ini
MaxRetentionSec=7day
```

* Time-based retention
* Old logs deleted automatically

**Enterprise pattern**:

* Short retention locally
* Long retention in Elasticsearch

---

### **Rate Limiting (Critical for DoS Protection)**

```ini
RateLimitIntervalSec=30s
RateLimitBurst=10000
```

Meaning:

* Allow 10,000 messages per 30 seconds per service
* Prevents runaway logs from killing the system

Set higher for high-traffic APIs.

---

### **SyncIntervalSec**

```ini
SyncIntervalSec=5m
```

* How often logs are flushed to disk
* Lower = safer
* Higher = better performance

5 minutes is a good enterprise balance.

---

## **8.8 How journald Rotates Logs (Internals)**

journald rotates logs when **any** condition is met:

* Disk usage exceeds `SystemMaxUse`
* File exceeds `SystemMaxFileSize`
* Log entry is older than `MaxRetentionSec`

Rotation is:

* Automatic
* Safe
* No signals
* No truncation
* No data loss

---

## **8.9 journald vs logrotate (Why logrotate Is NOT Used)**

| Feature             | logrotate | journald |
| ------------------- | --------- | -------- |
| Binary safe         | ❌         | ✅        |
| Copytruncate needed | ❌         | ❌        |
| Crash safe          | ❌         | ✅        |
| Structured metadata | ❌         | ✅        |
| Container aware     | ❌         | ✅        |

**Never use logrotate on journald.**

---

## **8.10 journalctl — Core Operational Commands**

### **View All Logs**

```bash
journalctl
```

---

### **Follow Logs (Like tail -f)**

```bash
journalctl -f
```

---

### **Filter by Service**

```bash
journalctl -u nginx.service
```

---

### **Filter by Docker Container**

```bash
journalctl CONTAINER_NAME=orders-api
```

---

### **Filter by Time**

```bash
journalctl --since "2025-01-01 10:00:00"
journalctl --until "2025-01-01 12:00:00"
```

---

### **Filter by Priority**

```bash
journalctl -p err
```

Priority levels:

* `emerg`
* `alert`
* `crit`
* `err`
* `warning`
* `notice`
* `info`
* `debug`

---

### **Show Logs for Current Boot**

```bash
journalctl -b
```

Previous boot:

```bash
journalctl -b -1
```

---

### **Output as JSON (For Debugging Pipelines)**

```bash
journalctl -o json-pretty
```

---

### **Show Disk Usage**

```bash
journalctl --disk-usage
```

---

### **Vacuum Logs Manually**

```bash
journalctl --vacuum-time=3d
journalctl --vacuum-size=2G
```

(Usually not needed if config is correct.)

---

## **8.11 Real-World Example: Docker + NestJS Log Flow**

### **Application**

NestJS logs:

```json
{
  "level": "info",
  "service": "orders-api",
  "msg": "Order created",
  "orderId": 123
}
```

### **journald Stores**

* `MESSAGE` = JSON string
* `CONTAINER_NAME=orders-api`
* `_PID`, `_BOOT_ID`, timestamps

### **Query**

```bash
journalctl CONTAINER_NAME=orders-api -o json-pretty
```

You see:

* Exact log
* Container metadata
* Host metadata
* Boot/session metadata

---

## **8.12 Best Practices Summary (Enterprise Rules)**

✅ Persistent storage
✅ Hard disk usage cap
✅ Short local retention
✅ Compression enabled
✅ Rate limiting configured
✅ Centralized shipping (Filebeat)
❌ No application log files
❌ No logrotate
❌ No per-container retention on host

---

## **8.13 Final Positioning for DevOps Teams**

**journald is your:**

* Reliable ingestion layer
* Local safety net
* Structured log store
* First line of defense

**Elasticsearch is your:**

* Long-term retention
* Analytics engine
* Compliance archive

This separation of responsibility is **intentional**, **proven**, and **enterprise-standard**.

---

