Below is a **clean “field handbook” summary** of everything we discovered, plus a **minimal verification toolbox (one command per layer)** you can reuse in production incidents.

---

# 🧠 1. What actually happened (root cause chain)

You had a **resource ceiling problem, not a traffic problem**:

```text
Traffic spike (normal/moderate)
        ↓
Too many concurrent connections (not requests/sec)
        ↓
HAProxy maxconn too low → rejects connections
        ↓
FD limit too low (soft=1024 initially) → socket exhaustion
        ↓
TLS handshake failures + SYN_RECV buildup
        ↓
service instability / timeouts
```

---

# 📌 2. Key mental model (must remember)

| Layer         | Metric           | Meaning                    |
| ------------- | ---------------- | -------------------------- |
| Network       | pps (sar)        | packets/sec (NOT requests) |
| TCP           | connections (ss) | concurrent sockets         |
| App (HAProxy) | maxconn          | concurrency cap            |
| OS            | FD limit         | socket ceiling             |

👉 These are independent dimensions — not the same thing.

---

# 📊 3. Final state after fix

| Metric        | Value     | Meaning       |
| ------------- | --------- | ------------- |
| FD limit      | 200k      | healthy       |
| maxconn       | 20k       | safe baseline |
| traffic       | ~7.5k pps | normal        |
| estimated RPS | ~200–500  | moderate load |
| SYN_RECV      | ~0        | healthy       |
| drops/errors  | none      | stable        |

---

# 🧪 4. “Single command” production toolbox

Run this for a **full health snapshot**:

```bash
bash monitor.sh
```

Here is the **corrected production-grade version (handles HAProxy dual process + real signals):**

```bash
#!/bin/bash

echo "================ SYSTEM HEALTH REPORT ================"

PID=$(pgrep -xo haproxy)

echo ""
echo "[1] PROCESS CHECK"
ps -o pid,user,%cpu,%mem,cmd -p $PID

echo ""
echo "[2] FD LIMITS"
cat /proc/$PID/limits | grep "Max open files"

echo ""
echo "[3] FD USAGE"
ls /proc/$PID/fd | wc -l

echo ""
echo "[4] TCP CONNECTIONS (HAProxy only)"
ss -antp | grep haproxy | wc -l

echo ""
echo "[5] SYN_RECV STATE"
ss -ant state syn-recv sport = :443 | wc -l

echo ""
echo "[6] ESTABLISHED CONNECTIONS"
ss -ant state established sport = :443 | wc -l

echo ""
echo "[7] TIME_WAIT COUNT"
ss -ant state time-wait | wc -l

echo ""
echo "[8] INTERFACE TRAFFIC (1s sample)"
sar -n DEV 1 1 | grep Average -A 1

echo ""
echo "[9] KERNEL TCP HEALTH"
nstat -az | egrep "ListenOverflows|ListenDrops|TcpActiveOpens|TcpRetransSegs"

echo ""
echo "[10] HAProxy ERRORS (last 10 min)"
grep -E "SSL handshake|maxconn|FD limit" /var/log/haproxy.log | tail -n 20

echo "======================================================"
```

---

# 🚨 5. How to interpret this report quickly

### 🔴 Problem indicators

* SYN_RECV > 100–200 → SYN flood / backlog issue
* ListenDrops > 0 → kernel overload
* FD usage > 70% of limit → risk zone
* maxconn errors → HAProxy saturation
* SSL handshake failures spike → overload or timeout issue

---

### 🟢 Healthy indicators

* SYN_RECV ≈ 0
* TIME_WAIT stable
* FD usage < 20%
* low ListenDrops
* stable handshake logs

---

# 🧩 6. One-line “traffic reality check”

If you only run ONE command in an incident:

```bash
sar -n DEV 1 5
```

👉 This answers: “Is traffic actually high?”

Then correlate with:

```bash
ss -s
```

👉 This answers: “Are connections exploding?”

---

# 🧠 7. Final takeaway (what your incident taught)

You learned a production-critical truth:

> **Infrastructure failures are usually not caused by traffic volume — but by concurrency + resource ceilings.**

Specifically:

* FD limit controls socket survival
* maxconn controls HAProxy concurrency
* SYN backlog controls connection intake
* traffic (pps/RPS) only triggers, not causes failure

---

### Another monitoring script:

```bash
#!/bin/bash

echo "================ SYSTEM HEALTH REPORT ================"

PID=$(pgrep -u haproxy -x haproxy | sort | head -n1)

echo ""
echo "[1] PROCESS CHECK"
ps -o pid,user,%cpu,%mem,cmd -p $PID

echo ""
echo "[2] FD LIMITS"
cat /proc/$PID/limits | grep "Max open files"

echo ""
echo "[3] FD USAGE"
ls /proc/$PID/fd | wc -l

echo ""
echo "[4] TCP CONNECTIONS (HAProxy only)"
ss -antp | grep haproxy | wc -l

echo ""
echo "[5] SYN_RECV STATE"
ss -ant state syn-recv sport = :443 | wc -l

echo ""
echo "[6] ESTABLISHED CONNECTIONS"
ss -ant state established sport = :443 | wc -l

echo ""
echo "[7] TIME_WAIT COUNT"
ss -ant state time-wait | wc -l

echo ""
echo "[8] INTERFACE TRAFFIC (1s sample)"
sar -n DEV 1 1 | grep Average -A 1

echo ""
echo "[9] KERNEL TCP HEALTH"
nstat -az | egrep "ListenOverflows|ListenDrops|TcpActiveOpens|TcpRetransSegs"

echo ""
echo "[10] HAProxy ERRORS (last 10 min)"
grep -E "SSL handshake|maxconn|FD limit" /var/log/haproxy.log | tail -n 20

echo "======================================================"
```

### And Another One:

```bash 
#!/bin/bash

PID=$(pgrep -u haproxy -x haproxy | sort | head -n1)

echo "======================================"
echo "        HAProxy HEALTH REPORT"
echo "======================================"
echo ""

########################################
# 1. FD USAGE
########################################
FD_USED=$(ls /proc/$PID/fd 2>/dev/null | wc -l)
FD_LIMIT=$(cat /proc/$PID/limits 2>/dev/null | grep "Max open files" | awk '{print $5}' | head -n1)

echo "[FD USAGE]"
echo "Used:   $FD_USED"
echo "Limit:  $FD_LIMIT"

if [[ "$FD_LIMIT" != "" && "$FD_LIMIT" -gt 0 ]]; then
  PCT=$(( FD_USED * 100 / FD_LIMIT ))
  echo "Ratio:  ${PCT}%"

  if [ $PCT -gt 80 ]; then
    echo "ALERT: FD usage HIGH"
  fi
fi

echo ""
########################################
# 2. SYN_RECV CHECK
########################################
SYN_RECV=$(ss -ant state syn-recv sport = :443 2>/dev/null | wc -l)

echo "[SYN_RECV QUEUE]"
echo "Count: $SYN_RECV"

if [ $SYN_RECV -gt 100 ]; then
  echo "ALERT: Possible SYN flood or backlog pressure"
fi

echo ""
########################################
# 3. TIME_WAIT CHECK
########################################
TIME_WAIT=$(ss -ant state time-wait 2>/dev/null | wc -l)

echo "[TIME_WAIT]"
echo "Count: $TIME_WAIT"

if [ $TIME_WAIT -gt 1000 ]; then
  echo "WARN: High connection churn"
fi

echo ""
########################################
# 4. SSL HANDSHAKE ERRORS
########################################
SSL_ERR=$(grep -c "SSL handshake failure" /var/log/haproxy.log 2>/dev/null)

echo "[SSL HANDSHAKE ERRORS - LAST 24H]"
echo "Count: $SSL_ERR"

if [ $SSL_ERR -gt 0 ]; then
  echo "WARN: handshake failures detected"
fi

echo ""
########################################
# 5. ESTABLISHED CONNECTIONS
########################################
ESTAB=$(ss -ant state established sport = :443 2>/dev/null | wc -l)

echo "[ESTABLISHED CONNECTIONS]"
echo "Count: $ESTAB"

echo ""
########################################
# 6. TOP PEER IPs
########################################
echo "[TOP PEER IPs]"

ss -ant sport = :443 2>/dev/null | \
awk 'NR>1 {print $5}' | cut -d: -f1 | \
sort | uniq -c | sort -nr | head -10

echo ""
########################################
# 7. SYN PACKET SIGNAL (approx)
########################################
echo "[SYN SIGNAL (estimation)]"
ss -ant state syn-recv sport = :443 2>/dev/null | wc -l

echo ""
echo "======================================"
echo "DONE"
echo "======================================"
```

# An Explanation of some important kernel flags: (numbers pertain to our own incident but the general description is a fine addon)

## Critical Drifts First

Three values stand out as genuinely problematic for your situation:

**1. `tcp_max_tw_buckets = 16384` — too small**
You had 378 TIME_WAIT sockets observed, which is fine *right now*, but with 120K active connections this bucket will fill under any churn spike and the kernel will start forcibly destroying TIME_WAIT sockets and logging warnings. At your scale this should be at least 1,000,000.

**2. `tcp_fin_timeout = 60` — too long**
60 seconds means a dead connection's FD is held for a full minute before cleanup. With 120K connections, even a 5% churn rate means thousands of FDs stuck in FIN_WAIT_2 for a minute each. At your FD pressure level (60% consumed), this is directly contributing to the problem. Should be 15.

**3. `tcp_keepalive_time = 7200` — dangerously long**
7200 seconds = 2 hours before the kernel sends the first keepalive probe on an idle connection. In a Flutter mobile app context, phones go to sleep, switch networks, and drop connections silently all the time. HAProxy and your Go backend will hold FDs open for up to 2 hours for connections that are already dead on the client side. Should be 60.

The rest are acceptable or minor. `somaxconn` and `tcp_max_syn_backlog` at 65535 are already good — your ListenOverflow problem was likely a previous state or a HAProxy `maxconn` ceiling, not the kernel queue size.

---

## Each Flag Explained

---

### `net.core.somaxconn = 65535` ✅
**What it is:** The maximum length of the pending connection queue for any listening socket — specifically connections that have completed the TCP 3-way handshake and are waiting for the application (HAProxy) to call `accept()`.

**What happens when it's too small:** The kernel drops incoming connections silently. This is exactly what caused your 28M `ListenOverflows`. The kernel fills this queue, then starts discarding new fully-established connections before HAProxy even sees them.

**Analogy:** A restaurant has a waiting area for seated-but-not-yet-served customers. `somaxconn` is how many chairs are in that waiting area. If it's full, new customers are turned away at the door even though tables exist.

**Your value:** 65535 is good. No change needed.

---

### `net.ipv4.tcp_max_syn_backlog = 65535` ✅
**What it is:** The queue for *half-open* connections — SYN received, SYN-ACK sent, but the final ACK from the client hasn't arrived yet (the TCP handshake is mid-flight).

**What happens when it's too small:** Under a SYN flood or simply high connection rate, the kernel drops incoming SYNs before the handshake completes. Clients see connection timeouts immediately.

**How it differs from somaxconn:** `somaxconn` is for completed handshakes waiting for `accept()`. `tcp_max_syn_backlog` is for handshakes still in progress. Two separate queues, two separate failure modes.

**Your SYN_RECV count was 1** — meaning this queue is essentially empty. Your value is fine.

---

### `net.core.netdev_max_backlog = 1000` ⚠️ minor
**What it is:** How many packets the kernel can queue on a NIC's receive ring when the CPU can't process them fast enough. This is per CPU core.

**What happens when it's too small:** At very high packet rates, the NIC fills this buffer and starts dropping packets at the driver level — before they even reach TCP. You'd see this in `ifconfig` as RX dropped or in `ethtool -S` as missed errors.

**Your traffic:** 9,878 rx packets/sec. At 1,000 per core this is probably fine unless you have a single-core bottleneck. Recommended 32,768 at your scale but not urgent.

---

### `net.core.rmem_max = 212992` ⚠️ low
**What it is:** The absolute maximum a socket's receive buffer can be set to. This is the ceiling — individual sockets won't exceed this regardless of what the application requests.

**What it affects:** How much unread incoming data the kernel can hold per socket before telling the sender to slow down (TCP flow control / window size). A small value means the kernel throttles the sender earlier than necessary, which hurts throughput on high-latency connections.

**Example:** A Flutter client on a mobile network with 80ms RTT trying to download a 1MB response. With 212KB buffer, the kernel's TCP window is constrained, so the Go backend has to stop sending and wait for ACKs more frequently. With 16MB buffer, it can pipeline much more data in flight.

**For your setup:** Flutter clients on mobile networks will benefit from larger buffers. Recommended 16777216 (16MB).

---

### `net.core.wmem_max = 212992` ⚠️ low
**What it is:** Same concept as `rmem_max` but for the send buffer. How much outgoing data the kernel can hold per socket waiting to be transmitted.

**What it affects:** HAProxy and your Go backend's ability to pipeline responses. Small send buffers mean the server fills the buffer, blocks, waits for ACKs, then continues — creating unnecessary latency cycles.

**Your value:** 212KB is the kernel default and is fine for LAN. For mobile clients with variable latency it's a bottleneck. Recommended 16777216.

---

### `net.ipv4.tcp_rmem = 4096 131072 30573920` ✅ mostly fine
**What it is:** Three values — minimum, default, and maximum receive buffer size per TCP socket. The kernel auto-tunes between min and max based on memory pressure.

```
4096      = minimum (4KB  — floor, used under memory pressure)
131072    = default  (128KB — starting size for new sockets)
30573920  = maximum  (29MB  — kernel can grow up to this)
```

**What it does:** The kernel automatically scales each socket's buffer based on observed RTT and bandwidth. A connection to a fast LAN client gets a smaller buffer; a slow mobile client gets a larger one. This is called TCP autotuning.

**Your value:** The max of 29MB is generous. Fine as-is, though the default of 128KB could be raised to 262144 if you notice slow starts on new connections.

---

### `net.ipv4.tcp_wmem = 4096 16384 4194304` ⚠️ low max
**What it is:** Same structure as `tcp_rmem` but for send buffers.

```
4096     = minimum  (4KB)
16384    = default  (16KB)
4194304  = maximum  (4MB)
```

**The problem:** Your max send buffer is only 4MB while your max receive buffer is 29MB. This asymmetry means you can receive data faster than you can send it. For a server that's primarily *sending* responses to Flutter clients, the send buffer ceiling matters more. Recommended max: 16777216 (16MB).

---

### `net.ipv4.tcp_max_tw_buckets = 16384` 🔴 critical
**What it is:** How many TIME_WAIT sockets the kernel will maintain simultaneously. TIME_WAIT is the state a connection enters after being closed — it persists for 2×MSL (typically 60 seconds) to catch any delayed packets.

**What happens when exceeded:** The kernel destroys the oldest TIME_WAIT socket to make room for the new one and logs: `TCP: time wait bucket table overflow`. That destroyed socket's port becomes immediately reusable, which can cause a new connection to accidentally receive a delayed packet from the old connection — data corruption.

**Your risk:** You have 120K active connections. Even at 1% close rate per second that's 1,200 connections/sec entering TIME_WAIT. Your bucket fills in about 13 seconds and starts overflowing. Set to 1,440,000.

---

### `net.ipv4.tcp_tw_reuse = 2` ✅
**What it is:** Controls whether the kernel can reuse a TIME_WAIT socket for a new *outgoing* connection.
- `0` = disabled
- `1` = enabled
- `2` = enabled only for loopback (safer default since Linux 4.6)

**What it does:** When HAProxy opens a new connection to your Go backend (which is local/loopback in many setups), it can reuse a port that's in TIME_WAIT instead of waiting 60 seconds for it to expire naturally. This expands the effective outbound port range.

**Your value:** 2 is the safe modern default. Fine as-is.

---

### `net.ipv4.tcp_fin_timeout = 60` 🔴 critical for your FD situation
**What it is:** How long the kernel keeps a socket in FIN_WAIT_2 state. FIN_WAIT_2 happens when your side has closed the connection but the remote side hasn't sent its FIN yet — the connection is half-closed.

**Why it matters for FDs:** A socket in FIN_WAIT_2 still holds an FD. At 120K connections, if even 5% are in slow-close states, that's 6,000 FDs stuck for 60 seconds each. With your FD ceiling pressure, these dead FDs are directly competing with real connections.

**Example:** A Flutter user switches from WiFi to 4G mid-session. The TCP connection goes dead — no RST, no FIN, just silence. HAProxy starts the close, sends FIN, gets nothing back. That socket sits in FIN_WAIT_2 for 60 seconds consuming an FD. With 16 set instead, it's cleaned up in 15 seconds.

**Change to:** 15

---

### `net.ipv4.ip_local_port_range = 32768 60999` ⚠️ worth expanding
**What it is:** The range of ephemeral ports the kernel can use for *outbound* connections. Every connection HAProxy opens toward your Go backend uses one port from this range.

**The math:** 60999 - 32768 = 28,231 available ports. If HAProxy has 60K connections to the backend, and each needs a unique source port, you'd exhaust this range. In practice `tw_reuse` and connection pooling help, but the range is still tight.

**Change to:** `1024 65535` — giving you 64,511 ports. Avoid going below 1024 (reserved for privileged services).

---

### `net.ipv4.tcp_retries2 = 15` ⚠️ too patient
**What it is:** How many times the kernel retransmits a packet on an established connection before giving up and killing the socket. Each retry doubles the timeout (exponential backoff), so 15 retries can mean waiting up to **~30 minutes** before a dead connection is declared dead.

**Why this hurts you:** A dead Flutter client (phone off, network gone) leaves a connection in your Go backend and HAProxy holding FDs. With 15 retries, those zombie connections survive for potentially 30 minutes. With 8 retries, ~8-10 minutes. Combined with the keepalive fix, dead connections get cleaned up much faster.

---

### `net.ipv4.tcp_syn_retries = 6` ⚠️ slightly high
**What it is:** How many times the kernel retries a SYN packet when making an outgoing connection (HAProxy → backend). Retry 1 waits 1s, retry 2 waits 2s, retry 3 waits 4s... exponential. At 6 retries that's up to 63 seconds before declaring a backend unreachable.

**For your setup:** Your Go backend is local. If it's not responding in 3 retries (~7 seconds), it's down. HAProxy's own `timeout connect` should catch this first, but kernel-level 6 retries is still wasteful. Change to 3.

---

### `net.ipv4.tcp_keepalive_time = 7200` 🔴 critical for Flutter clients
**What it is:** How long a TCP connection must be *idle* before the kernel sends the first keepalive probe. 7200 = 2 hours.

**The Flutter problem:** Mobile apps go to background, phones sleep, networks switch. The TCP connection dies silently on the client side but HAProxy and your Go backend have no idea — they hold the FD open for 2 hours waiting for activity that will never come. Those are zombie FDs directly eating into your 200K limit.

**Change to:** 60 seconds. After 60 seconds of silence, start probing.

---

### `net.ipv4.tcp_keepalive_intvl = 75` ⚠️
**What it is:** Once keepalive probing starts (after `keepalive_time`), how often to send each probe. 75 seconds between probes.

**Combined with `keepalive_time = 7200`:** First probe at 2 hours, then every 75 seconds. With 9 probes (`keepalive_probes`) that's 2 hours + 11 minutes before a dead connection is cleaned up.

**Change to:** 10 seconds between probes.

---

### `net.ipv4.tcp_keepalive_probes = 9` ⚠️
**What it is:** How many unanswered keepalive probes before declaring the connection dead and closing it.

**Current behavior:** After 2 hours idle, send 9 probes 75 seconds apart = 11 more minutes. Total: **2 hours 11 minutes** to detect a dead Flutter connection.

**Proposed behavior:** After 60s idle, send 6 probes 10 seconds apart = 1 more minute. Total: **~70 seconds** to detect a dead connection and free the FD.

**Change to:** 6

---

