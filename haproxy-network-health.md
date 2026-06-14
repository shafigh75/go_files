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
