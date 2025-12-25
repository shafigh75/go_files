# 📚 Linux Network Troubleshooting Handbook
## A Comprehensive Guide to Diagnosing and Fixing Network Issues

---

## 🎯 **Introduction: The Network Troubleshooting Mindset**

When you see peaks in sockets, softnet packets, or other network metrics in Grafana, you're seeing symptoms. This handbook provides a systematic approach to diagnose the root cause, with special focus on kernel-level tuning and debugging.

---

## 🧠 **Part 1: Core Networking Concepts**

### **1.1 Connection States (TCP States)**
- **ESTABLISHED**: Fully open connection (healthy)
- **TIME_WAIT**: Connection closed, waiting to ensure all packets cleared (normal but can accumulate)
- **CLOSE_WAIT**: Remote end closed connection, local app hasn't closed it (⚠️ **BAD** - app bug)
- **SYN_SENT**: Outgoing connection attempt
- **SYN_RECV**: Incoming connection, waiting for completion (high count = SYN flood risk)
- **FIN_WAIT1/FIN_WAIT2**: Connection shutdown in progress

### **1.2 Key Kernel Concepts**
- **Softnet**: Kernel network processing queue. Drops occur when packets arrive faster than CPU can process them.
- **Conntrack**: Connection tracking table (used by NAT/firewalls). Exhaustion causes packet drops.
- **Socket Buffers**: Memory allocated for network packets (read/write buffers).
- **Backlog Queue**: Queue for incoming connection requests waiting to be accepted by applications.

### **1.3 Network Stack Flow**
```
Network Interface → Softnet Queue → TCP/IP Stack → Socket Buffers → Application
```
**Bottlenecks can occur at any stage!**

---

## 🔧 **Part 2: Essential Tools & Commands**

### **2.1 Connection Analysis**
```bash
# Connection state summary (your starting point)
netstat -ant | awk '{print $6}' | sort | uniq -c | sort -nr
ss -s  # More modern alternative to netstat -s

# Process-level connection analysis
ss -tulpn  # Show connections with process names
lsof -i -P -n | grep ESTABLISHED  # List open connections by process

# Remote IP analysis (detect floods/attacks)
ss -tn | awk 'NR>1 {print $5}' | cut -d: -f1 | sort | uniq -c | sort -nr | head -20
```

### **2.2 Kernel & Queue Monitoring**
```bash
# Softnet statistics (CRITICAL for packet drops)
cat /proc/net/softnet_stat  # 3 columns: packets processed, dropped, squeezed

# Interface statistics
ip -s link show  # RX/TX errors, drops, overruns
ethtool -S eth0 | grep -i drop  # Hardware-level drops

# CPU network interrupt usage
mpstat -P ALL 1 5  # Watch %soft (softirq) CPU usage
```

### **2.3 Connection Tracking**
```bash
# Conntrack table usage
cat /proc/sys/net/netfilter/nf_conntrack_count
cat /proc/sys/net/netfilter/nf_conntrack_max

# List conntrack entries (debugging)
conntrack -L | head -50
```

### **2.4 Real-time Monitoring**
```bash
# Bandwidth by connection
sudo iftop -i eth0

# Bandwidth by process
sudo nethogs eth0

# System activity reporting (historical analysis)
sar -n TCP,ETCP 1 10  # TCP statistics
sar -n DEV 1 10      # Network interface stats
```

### **2.5 System Logs**
```bash
# Kernel network messages
dmesg -T | grep -i 'drop\|overflow\|net\|socket'

# System logs around incident time
journalctl --since "2024-01-15 14:00:00" --until "2024-01-15 14:30:00" | grep -i network
```

---

## ⚙️ **Part 3: Kernel Parameters & sysctl Tuning (The Heart of This Handbook)**

### **3.1 TCP Connection Management**

#### **`net.ipv4.tcp_tw_reuse`**
- **What**: Allows reusing TIME_WAIT sockets for new outgoing connections
- **When needed**: High number of short-lived outgoing connections (web servers, APIs)
- **Default**: 0 (disabled)
- **Recommended**: 1 (enable)
- **Debug**: High TIME_WAIT count with many new connection attempts
- **Command**: `sysctl -w net.ipv4.tcp_tw_reuse=1`

#### **`net.ipv4.tcp_fin_timeout`**
- **What**: How long to keep sockets in FIN_WAIT2 state before force-closing
- **When needed**: Many stuck FIN_WAIT2 connections
- **Default**: 60 seconds
- **Recommended**: 30 (balance between cleanup and safety)
- **Debug**: `netstat -ant | grep FIN_WAIT2` shows many stale connections
- **Command**: `sysctl -w net.ipv4.tcp_fin_timeout=30`

#### **`net.ipv4.tcp_max_syn_backlog`**
- **What**: Maximum number of SYN packets queued for processing
- **When needed**: SYN flood attacks or very high connection rates
- **Default**: 1024
- **Recommended**: 2048-4096 for high-traffic servers
- **Debug**: `ss -tun state SYN-RECV` shows many entries, or softnet drops during SYN floods
- **Command**: `sysctl -w net.ipv4.tcp_max_syn_backlog=2048`

### **3.2 Socket & Buffer Tuning**

#### **`net.core.somaxconn`**
- **What**: Maximum number of connections in the backlog queue for ANY port
- **When needed**: Application can't accept connections fast enough (listen queue overflow)
- **Default**: 128 (way too low for production)
- **Recommended**: 65535 (match with application settings)
- **Debug**: `ss -l` shows Recv-Q approaching Send-Q, or application logs show "connection refused"
- **Command**: `sysctl -w net.core.somaxconn=65535`

#### **`net.core.netdev_max_backlog`**
- **What**: Maximum packets queued when interface receives faster than kernel processes
- **When needed**: Softnet drops during traffic spikes
- **Default**: 1000
- **Recommended**: 20000-30000 for 10G+ interfaces
- **Debug**: `cat /proc/net/softnet_stat` shows non-zero second column (drops)
- **Command**: `sysctl -w net.core.netdev_max_backlog=30000`

#### **Socket Memory Buffers**
```bash
# Read buffer auto-tuning (min/default/max in pages)
net.core.rmem_default = 131072    # 128KB default
net.core.rmem_max = 16777216     # 16MB max
net.core.wmem_default = 131072   # 128KB default  
net.core.wmem_max = 16777216     # 16MB max

# TCP-specific memory (min/pressure/max in pages)
net.ipv4.tcp_rmem = 4096 87380 6291456    # 4KB-85KB-6MB
net.ipv4.tcp_wmem = 4096 65536 4194304    # 4KB-64KB-4MB
```
- **When needed**: High latency, packet loss, or throughput issues
- **Debug**: `netstat -s | grep retrans` shows high retransmits, or `ip -s link` shows overruns
- **Recommended**: Increase for high-bandwidth, high-latency networks

### **3.3 Connection Tracking (NAT/Firewall)**

#### **`net.netfilter.nf_conntrack_max`**
- **What**: Maximum number of tracked connections
- **When needed**: Conntrack table exhaustion (common with NAT, Docker, firewalls)
- **Default**: 65536 * (RAM in GB) / 16GB (varies by distro)
- **Recommended**: 262144 (256K) or higher for busy servers
- **Debug**: `dmesg | grep conntrack` shows "table full" messages, or `cat /proc/sys/net/netfilter/nf_conntrack_count` near max
- **Command**: `sysctl -w net.netfilter.nf_conntrack_max=262144`

#### **`net.netfilter.nf_conntrack_tcp_timeout_established`**
- **What**: How long to keep established connections in conntrack table
- **When needed**: Conntrack table fills with idle connections
- **Default**: 432000 seconds (5 days!)
- **Recommended**: 86400 (24 hours) or 43200 (12 hours)
- **Debug**: Conntrack count high but actual active connections low
- **Command**: `sysctl -w net.netfilter.nf_conntrack_tcp_timeout_established=86400`

### **3.4 SYN Flood Protection**

#### **`net.ipv4.tcp_syncookies`**
- **What**: Enable SYN cookies to protect against SYN floods
- **When needed**: Under SYN flood attack or high SYN_RECV count
- **Default**: 1 (enabled on most modern systems)
- **Recommended**: 1 (always enable)
- **Debug**: `ss -tun state SYN-RECV` shows many entries, or softnet drops during attacks
- **Command**: `sysctl -w net.ipv4.tcp_syncookies=1`

#### **`net.ipv4.tcp_syn_retries`**
- **What**: Number of SYN retransmits before giving up
- **When needed**: Slow networks or intermittent connectivity
- **Default**: 6
- **Recommended**: 3-4 (faster failure detection)
- **Debug**: Long connection setup times, many SYN_SENT states
- **Command**: `sysctl -w net.ipv4.tcp_syn_retries=3`

### **3.5 File Descriptor Limits**

#### **`fs.file-max`**
- **What**: System-wide maximum number of file descriptors
- **When needed**: "Too many open files" errors, high socket usage
- **Default**: Varies by system RAM
- **Recommended**: 2097152 (2M) for busy servers
- **Debug**: `dmesg` shows "file-max limit reached", or applications crash with EMFILE
- **Command**: `sysctl -w fs.file-max=2097152`

#### **Per-process limits (ulimit)**
```bash
# Check current limits
ulimit -n

# Set in /etc/security/limits.conf
* soft nofile 65535
* hard nofile 65535
```

---

## 🕵️ **Part 4: Debugging Scenarios & Workflows**

### **Scenario 1: Softnet Packet Drops (Your Grafana Peak)**
**Symptoms**: High softnet drops in Grafana, slow network response
```bash
# 1. Check softnet statistics
watch -n 1 'cat /proc/net/softnet_stat'

# 2. Check CPU softirq usage
mpstat -P ALL 1 10

# 3. Check interface errors
ip -s link show

# 4. If CPU-bound: tune netdev_max_backlog
sysctl -w net.core.netdev_max_backlog=30000

# 5. If application can't keep up: tune somaxconn and app settings
sysctl -w net.core.somaxconn=65535
```

### **Scenario 2: Too Many TIME_WAIT Connections**
**Symptoms**: Thousands of TIME_WAIT states, new connections failing
```bash
# 1. Check TIME_WAIT count
netstat -ant | grep TIME_WAIT | wc -l

# 2. Enable reuse
sysctl -w net.ipv4.tcp_tw_reuse=1

# 3. Reduce timeout (carefully!)
sysctl -w net.ipv4.tcp_fin_timeout=30

# 4. Check if outgoing connections are the issue
ss -tun state TIME-WAIT | awk '{print $5}' | cut -d: -f1 | sort | uniq -c | sort -nr
```

### **Scenario 3: CLOSE_WAIT Leaks (Application Bug)**
**Symptoms**: Growing CLOSE_WAIT count, memory usage increasing
```bash
# 1. Identify problematic process
lsof -i | grep CLOSE_WAIT

# 2. Check application logs for errors
journalctl -u yourapp --since "1 hour ago"

# 3. Monitor CLOSE_WAIT growth
watch -n 5 'netstat -ant | grep CLOSE_WAIT | wc -l'

# 4. Restart the application (temporary fix)
systemctl restart yourapp

# 5. Permanent fix: debug application code for proper socket closure
```

### **Scenario 4: Conntrack Table Exhaustion**
**Symptoms**: New connections failing, "table full" in logs
```bash
# 1. Check current usage
cat /proc/sys/net/netfilter/nf_conntrack_count
cat /proc/sys/net/netfilter/nf_conntrack_max

# 2. Increase table size
sysctl -w net.netfilter.nf_conntrack_max=262144

# 3. Reduce timeout for established connections
sysctl -w net.netfilter.nf_conntrack_tcp_timeout_established=86400

# 4. Check for connection leaks
conntrack -L | head -50
```

---

## 🛡️ **Part 5: Prevention & Best Practices**

### **5.1 Essential sysctl.conf Settings**
```bash
# /etc/sysctl.conf - Production-ready defaults
# Apply with: sysctl -p

# TCP optimization
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_syn_backlog = 2048

# Socket queue sizes
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 30000

# Buffer sizes (for 1Gbps+ networks)
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# Connection tracking
net.netfilter.nf_conntrack_max = 262144
net.netfilter.nf_conntrack_tcp_timeout_established = 86400

# File descriptors
fs.file-max = 2097152
```

### **5.2 Monitoring Setup**
```bash
# Install monitoring tools
sudo apt install sysstat net-tools iftop nethogs

# Enable sysstat data collection
sudo sed -i 's/ENABLED="false"/ENABLED="true"/' /etc/default/sysstat
sudo systemctl restart sysstat

# Add to crontab for regular monitoring
* * * * * sar -n TCP,ETCP 1 59 > /var/log/sar_tcp_$(date +\%H).log
```

### **5.3 Alerting Thresholds**
Set Grafana alerts for:
- Softnet drops > 0 for 5 minutes
- Conntrack usage > 80%
- CLOSE_WAIT count > 100
- TIME_WAIT count > 10000
- CPU softirq > 50%

---

## 📋 **Part 6: Quick Reference Cheat Sheet**

### **Immediate Diagnostics (Run First)**
```bash
ss -s                    # Connection summary
cat /proc/net/softnet_stat  # Softnet drops
ip -s link show          # Interface errors
dmesg -T | tail -50      # Kernel messages
mpstat -P ALL 1 5        # CPU softirq usage
```

### **Common sysctl Tuning by Scenario**
| Scenario | Key Parameters | Values |
|----------|----------------|---------|
| High Traffic Web Server | tcp_tw_reuse, somaxconn, netdev_max_backlog | 1, 65535, 30000 |
| SYN Flood Protection | tcp_syncookies, tcp_max_syn_backlog | 1, 2048 |
| NAT/Firewall Server | nf_conntrack_max, nf_conntrack_tcp_timeout_established | 262144, 86400 |
| High-Bandwidth Network | tcp_rmem, tcp_wmem, rmem_max, wmem_max | See section 3.2 |

### **Debugging Flowchart**
```
Grafana Peak Detected
    ↓
Check softnet drops (cat /proc/net/softnet_stat)
    ↓
Are drops > 0? → Yes → Check CPU softirq (mpstat)
    ↓ No               ↓ High? → Yes → Tune netdev_max_backlog
Check connection states (ss -s)  ↓ No
    ↓                           Check interface errors (ip -s)
ESTABLISHED high? → Yes → Check application throughput
    ↓ No
TIME_WAIT high? → Yes → Enable tcp_tw_reuse, reduce fin_timeout
    ↓ No
CLOSE_WAIT high? → Yes → Find leaking application
    ↓ No
Check conntrack usage
```

---

## 🎓 **Part 7: Advanced Topics**

### **7.1 Kernel Bypass (XDP/eBPF)**
For extreme performance:
- **XDP (eXpress Data Path)**: Process packets at driver level
- **eBPF**: Safe kernel programming for custom network logic
- **DPDK**: Full userspace networking (bypasses kernel entirely)

### **7.2 NUMA Awareness**
For multi-socket servers:
```bash
# Check NUMA topology
numactl --hardware

# Bind network IRQs to specific CPUs
irqbalance --oneshot

# Tune for local memory access
sysctl -w net.core.rmem_max=33554432  # 32MB for 10G+ networks
```

### **7.3 TCP BBR Congestion Control**
```bash
# Enable BBR (better throughput on high-BDP networks)
sysctl -w net.core.default_qdisc=fq
sysctl -w net.ipv4.tcp_congestion_control=bbr

# Verify
sysctl net.ipv4.tcp_congestion_control
```

---

## 🚀 **Final Tips**

1. **Always test changes**: `sysctl -w` for temporary testing, `/etc/sysctl.conf` for persistence
2. **Monitor before/after**: Capture metrics before tuning to measure impact
3. **Document everything**: Keep a changelog of what you tuned and why
4. **Start conservative**: Increase parameters gradually, not to maximum values
5. **Know your workload**: Web server tuning differs from database or real-time trading systems

**Remember**: The goal isn't to eliminate all TIME_WAIT or CLOSE_WAIT states, but to ensure they don't cause resource exhaustion or performance degradation. Every system is different - use this handbook as a guide, not a rigid rulebook.

Happy troubleshooting! 🛠️
