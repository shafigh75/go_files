# پاسخ به سوالات مصاحبه فنی — DevOps & Backend Engineer

---

# حوزه ۱: Linux & Networking Mental Model

---

## ۱. وقتی یه process روی پورت listen می‌کنه، دقیقاً چی توی kernel اتفاق می‌افته؟

وقتی `server.listen(8080)` صدا می‌زنی، یه سری syscall به ترتیب اتفاق می‌افته:

```c
// ترتیب واقعی syscall‌ها
socket(AF_INET, SOCK_STREAM, 0)  // → fd=3, ایجاد socket در kernel
bind(3, {addr=0.0.0.0, port=8080})  // bind کردن به آدرس/پورت
listen(3, backlog=128)               // تبدیل socket به passive mode
accept(3, ...)                       // بلاک شدن و انتظار برای connection
```

**kernel-side دقیق:**

- `socket()`: یه `sock` struct توی kernel ایجاد میشه. TCP state machine شروع میشه از `CLOSED`.
- `bind()`: kernel چک می‌کنه پورت آزاده، `SO_REUSEADDR` هست یا نه، و socket رو به آدرس bind می‌کنه.
- `listen()`: دو تا queue ایجاد میشه:
  - **SYN queue** (incomplete queue): connectionهایی که SYN گرفتیم ولی handshake کامل نشده — state: `SYN_RECEIVED`
  - **Accept queue** (complete queue): handshake کامل شده، منتظر `accept()` — state: `ESTABLISHED`

Client → SYN →       [ SYN Queue ] → SYN+ACK → Client
Client → ACK →       [ Accept Queue ] → accept() → Process


**`backlog` parameter** دقیقاً سایز accept queue رو کنترل می‌کنه (در kernel جدید). اگه پر بشه، SYN جدید drop میشه.

**مثال عملی:**

```bash
# دیدن این queues در عمل
ss -lnt sport = :8080
# Recv-Q = تعداد connection توی accept queue
# Send-Q = سایز backlog

# اگه Recv-Q == Send-Q داری: backlog پره، داری connection drop می‌کنی!
```

---

## ۲. TIME_WAIT چیه؟ چرا ۶۰ ثانیه؟ چه مشکلی ایجاد می‌کنه؟

`TIME_WAIT` state‌ایه که **active closer** (اون طرفی که اول FIN می‌فرسته) بعد از بستن connection وارد میشه.

**چرا وجود داره — دو دلیل مشخص:**

**دلیل اول — MSL (Maximum Segment Lifetime):**

$$TIME\_WAIT = 2 \times MSL$$

اگه آخرین ACK ما گم بشه، server دوباره FIN می‌فرسته. باید هنوز زنده باشیم تا جواب بدیم. `MSL = 30s` → `TIME_WAIT = 60s`.

**دلیل دوم — جلوگیری از packet confusion:**
اگه زود ببندیم و همون `(src_ip, src_port, dst_ip, dst_port)` رو دوباره استفاده کنیم، packet قدیمی که توی شبکه موند می‌تونه connection جدید رو corrupt کنه.

**مشکل عملی — مثال واقعی:**

```bash
# توی یه سرویس که خودش HTTP client هست (مثلاً reverse proxy)
# و هزاران connection کوتاه می‌زنه:
ss -s
# TIME-WAIT: 50000  ← خطرناکه!

# هر TIME_WAIT یه (local_port) اشغال می‌کنه
# محدوده پورت‌های ephemeral:
cat /proc/sys/net/ipv4/ip_local_port_range
# 32768 60999 → فقط ~28000 پورت داریم!
```

**راه‌حل‌های عملی:**

```bash
# راه‌حل ۱ — enable tcp_tw_reuse (فقط برای outgoing connections)
echo 1 > /proc/sys/net/ipv4/tcp_tw_reuse
# این safe هست برای client-side

# راه‌حل ۲ — keepalive فعال کن تا connection کمتر بسته بشه
# توی Go:
transport := &http.Transport{
    MaxIdleConns:        100,
    IdleConnTimeout:     90 * time.Second,
    DisableKeepAlives:   false,
}

# راه‌حل ۳ — SO_REUSEPORT برای server-side
```

> **هرگز** `tcp_tw_recycle` رو فعال نکن — در kernel 4.12 حذف شد چون با NAT مشکل داشت.

---

## ۳. MTU و PMTUD

**MTU = Maximum Transmission Unit** — بزرگترین packet که یه interface می‌تونه بفرسته.

**وقتی packet بزرگ‌تر از MTU بیاد:**

دو سناریو:

۱. DF bit = 0 (Don't Fragment off):
   → Router/kernel خودش fragment می‌کنه
   → Performance افت می‌کنه، reassembly overhead داره

۲. DF bit = 1 (Don't Fragment on — TCP پیش‌فرض اینه):
   → Packet drop میشه
   → ICMP "Fragmentation Needed" با MTU مناسب برمی‌گرده
   → Sender باید MSS رو کم کنه


**PMTUD (Path MTU Discovery):**

مکانیزمیه که TCP به صورت خودکار کوچکترین MTU توی مسیر رو پیدا می‌کنه:

Client (MTU=1500) → Router1 (MTU=1500) → Tunnel (MTU=1400) → Server

۱. Client می‌فرسته: 1460 byte payload (MSS=1460) با DF=1
۲. Tunnel drop می‌کنه، ICMP Frag Needed با next-hop MTU=1400 برمی‌گردونه
۳. Client MSS رو reduce می‌کنه به ~1360


**مشکل رایج — ICMP Blackhole:**

```bash
# خیلی جاها firewall ICMP رو block می‌کنه!
# نتیجه: connection hang می‌کنه برای packet‌های بزرگ

# تشخیص:
ping -M do -s 1472 8.8.8.8  # 1472+28=1500 bytes با DF

# راه‌حل برای Kubernetes/VPN:
ip link set eth0 mtu 1450
# یا TCP MSS Clamping توی iptables:
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN \
  -j TCPMSS --clamp-mss-to-pmtu
```

---

## ۴. فرق ss و netstat

| ویژگی | `netstat` | `ss` |
|-------|-----------|------|
| منبع داده | `/proc/net/tcp` (parse text) | مستقیم از kernel via **netlink socket** |
| سرعت | کند — O(n) parse | سریع — kernel filtering |
| Extended info | محدود | `--extended`, `--memory`, `--process` |
| Status | deprecated در بیشتر distroها | جایگزین رسمی |

```bash
# ss مثال‌های عملی که netstat نمی‌تونه:

# فقط ESTABLISHED connectionهای روی پورت 443 با process name:
ss -tnp state established '( dport = :443 or sport = :443 )'

# memory usage هر socket:
ss -tm

# filter by cgroup (مهم برای kubernetes debugging):
ss --cgroup /system.slice/docker.service

# timer information:
ss -to  # retransmit timer، keepalive timer
```

**نتیجه:** همیشه `ss` استفاده کن. `netstat` فقط برای سیستم‌های قدیمیه.

---

## ۵. conntrack و اهمیتش در Kubernetes

**conntrack (Connection Tracking)** یه subsystem توی Linux kernel هست که state تمام network connectionها رو نگه می‌داره. پایه‌ی NAT و stateful firewall.

هر connection یه entry داره مثل:
tcp  ESTABLISHED src=10.0.0.1 dst=10.0.0.2 sport=54321 dport=80
                 src=10.0.0.2 dst=10.0.0.1 sport=80    dport=54321
     [original]              [reply]


**چرا توی Kubernetes حیاتیه:**

```bash
# ۱. kube-proxy از iptables/IPVS استفاده می‌کنه که خودش conntrack داره
# وقتی Pod به Service IP می‌زنه:
# DNAT: ClusterIP:Port → PodIP:Port
# conntrack این mapping رو نگه می‌داره تا reply صحیح برگرده

# ۲. مشکل رایج: conntrack table پر میشه!
cat /proc/sys/net/netfilter/nf_conntrack_max
# 131072 default

cat /proc/sys/net/netfilter/nf_conntrack_count
# اگه به max نزدیک شد: packet drop بدون هیچ error واضحی!

# مانیتورینگ:
conntrack -S
# insert_failed=1000  ← این یعنی داری packet drop می‌کنی

# راه‌حل:
sysctl -w net.netfilter.nf_conntrack_max=524288
```

**مشکل مشهور Kubernetes — conntrack race condition:**

DNS timeout مشهور در Kubernetes:
Pod → DNS query → conntrack RACE → packet drop → 5s timeout

دلیل: دو goroutine همزمان A و AAAA record می‌خوان
      هر دو UDP packet به همون src port می‌فرستن
      conntrack نمی‌تونه هر دو رو track کنه

راه‌حل: ndots:5 رو کم کن، یا از TCP برای DNS استفاده کن
        یا nodelocaldns cache


---

## ۶. iptables و nftables — ترتیب اجرا

این سوال ظریفه. وقتی هر دو نصبه، **بستگی داره به اینکه iptables چطور نصب شده:**

```bash
# دو پیاده‌سازی از iptables وجود داره:
iptables --version

# iptables-legacy: مستقیم با x_tables kernel module کار می‌کنه
# iptables-nft: یه wrapper که iptables syntax رو به nftables ترجمه می‌کنه!

update-alternatives --list iptables
```

**اگه iptables-legacy و nftables هر دو باشن:**

kernel packet path:
→ nftables (nf_tables) chains
→ iptables (x_tables) chains  ← جداست! هم‌زمان اجرا میشن

ترتیب: nftables prerouting → iptables prerouting → routing decision
        → nftables forward → iptables forward → ...


**مشکل عملی در Kubernetes:**

```bash
# Kubernetes (kube-proxy) از iptables-legacy استفاده می‌کنه
# اگه distro جدید باشه و nftables داشته باشه، conflict ایجاد میشه

# Calico/Cilium هم ممکنه با هم conflict داشته باشن
# راه‌حل: یکپارچه‌سازی — یا همه nftables، یا همه iptables-legacy

# چک کردن:
nft list ruleset | wc -l
iptables-legacy -L | wc -l
```

---

## ۷. cgroup v1 vs v2 — چرا Kubernetes migrate می‌کنه

**cgroup (Control Groups)** مکانیزم kernel برای resource limiting و accounting هست.

**cgroup v1 مشکلات:**

مشکل اصلی: هر resource controller (cpu, memory, blkio, ...) 
            یه hierarchy جداگانه داشت!

/sys/fs/cgroup/cpu/docker/container1/
/sys/fs/cgroup/memory/docker/container1/
/sys/fs/cgroup/blkio/docker/container1/

مشکل: memory + cpu رو نمی‌تونستی به صورت atomic تنظیم کنی
      container می‌تونست از یه controller فرار کنه
      accounting دقیق نبود


**cgroup v2 — unified hierarchy:**

```bash
# همه چیز زیر یه hierarchy:
/sys/fs/cgroup/kubepods/pod123/container456/
    ├── memory.current
    ├── memory.max
    ├── cpu.weight
    ├── cpu.max          # CPU bandwidth control — جدیده!
    └── io.max

# مهم‌ترین تغییرات:
# ۱. PSI (Pressure Stall Information) — می‌فهمی resource contention داری
cat /sys/fs/cgroup/kubepods/.../memory.pressure
# some avg10=5.00 avg60=2.00 avg300=1.00 total=...

# ۲. Memory QoS دقیق‌تر
# ۳. eBPF integration بهتر
```

**چرا Kubernetes migrate کرد:**

Kubernetes 1.25 → cgroup v2 به عنوان GA شد

دلایل:
۱. Memory limit enforcement دقیق‌تر — OOM kill درست‌تر
۲. CPU throttling دقیق‌تر با cpu.max
۳. Systemd روی distroهای جدید فقط v2 رو fully support می‌کنه
۴. eBPF-based tools (Cilium) با v2 بهتر کار می‌کنن

# چک کردن:
stat -fc %T /sys/fs/cgroup/
# cgroup2fs → v2 داری
# tmpfs → v1 داری


---

## ۸. OOM Killer — الگوریتم تصمیم‌گیری

وقتی kernel memory کم میاره و allocation fail میشه، OOM Killer باید یه process بکشه.

**امتیازدهی — `oom_score`:**

$$oom\_score = f(RSS, \text{time}, \text{privileges}, oom\_score\_adj)$$

```bash
# هر process یه oom_score داره (0-1000):
cat /proc/$(pgrep postgres)/oom_score
# عدد بزرگتر = احتمال kill بیشتر

# فرمول تقریبی kernel:
# base = (RSS + swap_usage) / total_memory * 1000
# adjust با oom_score_adj (-1000 تا +1000)

# دیدن امتیاز همه process‌ها:
for pid in /proc/[0-9]*/; do
    echo "$(cat $pid/oom_score 2>/dev/null) $(cat $pid/comm 2>/dev/null)"
done | sort -rn | head -20
```

**کنترل عملی:**

```bash
# محافظت از process مهم:
echo -1000 > /proc/$(pgrep etcd)/oom_score_adj
# -1000 = هرگز kill نکن (مثل کار systemd برای سرویس‌های حیاتی)

# در Kubernetes:
# QoS class تعیین می‌کنه:
# Guaranteed (request=limit): oom_score_adj = -997
# Burstable:                  oom_score_adj = proportional
# BestEffort:                 oom_score_adj = 1000  ← اول کشته میشه

# وقتی OOM kill اتفاق افتاد:
dmesg | grep -i "oom\|killed"
# Out of memory: Killed process 1234 (java) total-vm:4GB, anon-rss:3GB
```

---

# حوزه ۲: Go Internals

---

## ۱. Goroutine vs OS Thread — چرا میلیون‌ها؟

**OS Thread:**
- Kernel manages it
- Stack: 1-8 MB fixed (per thread)
- Context switch: kernel mode → باید registers، TLB flush، scheduling overhead
- هزینه‌ی ایجاد: ~1ms، ~1MB

**Goroutine:**
- Go runtime manages it — M:N threading model
- Stack: **2KB شروع می‌کنه، dynamically grows** تا 1GB
- Context switch: user space — فقط Go scheduler
- هزینه‌ی ایجاد: ~2μs، ~2KB

**مدل M:N:**

G = Goroutine
M = OS Thread (Machine)
P = Processor (logical, = GOMAXPROCS)

G1  G2  G3  G4        ← goroutines (هزاران تا)
|   |   |   |
[  P1  ] [  P2  ]     ← processors (مثلاً ۸ تا)
  |          |
  M1         M2        ← OS threads


```go
// وقتی goroutine block میشه (مثلاً I/O):
// runtime goroutine رو از M جدا می‌کنه
// M یه goroutine دیگه برمی‌داره
// وقتی I/O تموم شد، goroutine به یه P دیگه schedule میشه

// مثال concrete:
func main() {
    // این کار می‌کنه چون Go runtime threads رو reuse می‌کنه
    for i := 0; i < 1_000_000; i++ {
        go func() {
            time.Sleep(time.Hour)  // block شده ولی thread آزاده
        }()
    }
    // فقط GOMAXPROCS تا thread داریم، نه میلیون تا
}
```

---

## ۲. GOMAXPROCS دقیقاً چی رو کنترل می‌کنه؟

```go
// تعداد P (Processor) رو کنترل می‌کنه
// یعنی: حداکثر چند goroutine می‌تونن همزمان اجرا بشن

runtime.GOMAXPROCS(runtime.NumCPU())  // default از Go 1.5
```

**نکته مهم که اکثراً نمی‌دونن:**

```go
// در Kubernetes container، GOMAXPROCS رو از /proc/cpuinfo می‌خونه
// نه از cgroup limits!

// یعنی اگه node 32 core داشته باشه و container فقط 2 CPU limit داشته باشه:
// GOMAXPROCS=32  ← اشتباه! throttling زیاد میشه

// راه‌حل:
import _ "go.uber.org/automaxprocs"
// این package cgroup رو می‌خونه و GOMAXPROCS رو درست set می‌کنه

// یا manually:
import "runtime"
// در containerized environment:
runtime.GOMAXPROCS(2)  // بر اساس CPU limit
```

---

## ۳. Stack vs Heap — Escape Analysis

Go compiler در **compile time** تصمیم می‌گیره:

```go
// قانون کلی: اگه variable از function "فرار کنه" → heap
// فرار کردن یعنی: lifetime بیشتر از function داشته باشه

func noEscape() int {
    x := 42        // stack — محلیه، فرار نمی‌کنه
    return x       // value copy میشه
}

func escapes() *int {
    x := 42        // HEAP — چون pointer برمی‌گردونیم
    return &x      // x باید بعد از return زنده بمونه
}

// Interface هم باعث escape میشه:
func interfaceEscape(w io.Writer) {
    buf := make([]byte, 1024)  // heap! چون interface نمی‌دونه چی می‌خواد
    w.Write(buf)
}
```

**دیدن escape analysis:**

```bash
go build -gcflags="-m -m" ./...

# output:
# ./main.go:5:2: x escapes to heap
# ./main.go:12:14: buf does not escape  ← خوبه
```

**چرا مهمه؟** Heap allocation → GC pressure → latency spikes.

---

## ۴. sync.Mutex vs sync.RWMutex

```go
// Mutex: فقط یه goroutine، read یا write
// RWMutex: چند goroutine می‌تونن همزمان READ کنن
//          ولی WRITE یه lock انحصاری می‌خواد

// کِی RWMutex بهتره؟
// وقتی read >> write باشه (مثلاً cache)

type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.RLock()         // چند goroutine همزمان می‌تونن
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func (c *Cache) Set(key, val string) {
    c.mu.Lock()          // انحصاری
    defer c.mu.Unlock()
    c.data[key] = val
}
```

**تله‌های RWMutex:**

```go
// اگه write‌ها زیاد باشن، RWMutex کندتره از Mutex!
// چون باید منتظر همه reader‌ها بمونه

// Writer starvation: خیلی reader داری، writer گیر می‌کنه
// Go 1.14+ این رو fix کرد با fair scheduling

// برای high-contention cache: sync.Map یا sharded lock بهتره
```

---

## ۵. Buffered vs Unbuffered Channel — Deadlock

```go
// Unbuffered: sender BLOCK میشه تا receiver آماده باشه (synchronous)
ch := make(chan int)

// Buffered: sender تا buffer پر نشده block نمیشه
ch := make(chan int, 10)

// DEADLOCK سناریوها:

// ۱. Unbuffered — receiver نداری:
func deadlock1() {
    ch := make(chan int)
    ch <- 1  // block می‌کنه برای همیشه — deadlock!
}

// ۲. Circular wait:
func deadlock2() {
    ch1 := make(chan int)
    ch2 := make(chan int)
    go func() { ch1 <- <-ch2 }()
    go func() { ch2 <- <-ch1 }()
    // هر دو منتظر هم هستن
}

// ۳. Buffered ولی full:
func deadlock3() {
    ch := make(chan int, 1)
    ch <- 1
    ch <- 2  // block — buffer پره، receiver نداری
}
```

---

## ۶. context.Context — چرا مهمه؟

```go
// Context سه چیز propagate می‌کنه:
// ۱. Cancellation signal
// ۲. Deadline/Timeout
// ۳. Request-scoped values (مثل trace ID)

// بدون context — resource leak:
func fetchData() {
    resp, _ := http.Get("http://slow-service/data")
    // اگه caller بره، این هنوز داره request می‌زنه!
}

// با context:
func fetchData(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://slow-service/data", nil)
    resp, err := http.DefaultClient.Do(req)
    // اگه ctx cancel بشه، request automatically cancel میشه
    return err
}

// Propagation chain:
func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    // context رو به همه sub-call‌ها پاس میدیم
    result, err := fetchData(ctx)          // ← timeout propagate میشه
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            // timeout شد
        }
    }
}
```

**چطور cancellation propagate میشه؟**

```go
// context یه tree درست می‌کنه:
parent, cancel := context.WithCancel(context.Background())
child1, _ := context.WithTimeout(parent, 5*time.Second)
child2 := context.WithValue(parent, "key", "val")

cancel()  // ← parent cancel میشه → child1 و child2 هم cancel میشن
// اما child1 cancel نمیشه → parent تأثیر نمی‌گیره
```

---

## ۷. sync.Pool

```go
// sync.Pool: pool of reusable objects — GC pressure کم می‌کنه
// برای objects که frequently allocate/deallocate میشن عالیه

var bufPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)  // فقط وقتی pool خالیه ساخته میشه
    },
}

func processRequest(data []byte) {
    buf := bufPool.Get().([]byte)  // از pool بگیر
    defer bufPool.Put(buf[:0])     // برگردون (reset کن!)

    // استفاده از buf ...
    copy(buf, data)
}

// نکته مهم: GC می‌تونه pool رو empty کنه بین GC cycles
// پس نباید state مهم توش بذاری
// فقط برای temporary buffers مناسبه
```

**مثال واقعی — net/http server:**

```go
// Go's http server از sync.Pool برای response writers استفاده می‌کنه
// به همین دلیل under high load خوب scale می‌کنه
```

---

## ۸. Memory Leak در Go — پیدا کردن

```go
// Memory leak در Go معمولاً اینجاها اتفاق میفته:
// ۱. Goroutine leak (شایع‌ترین!)
// ۲. Global variable رشد می‌کنه
// ۳. Cache بدون eviction
// ۴. Finalizer مشکل داره

// ابزار اصلی: pprof
import _ "net/http/pprof"

go func() {
    http.ListenAndServe(":6060", nil)
}()
```

```bash
# Heap profile:
go tool pprof http://localhost:6060/debug/pprof/heap
# (pprof) top10
# (pprof) list functionName  ← کد رو نشون میده با allocation

# Goroutine leak:
curl http://localhost:6060/debug/pprof/goroutine?debug=2 | head -100
# اگه goroutine count رشد می‌کنه → leak داری

# Continuous profiling:
go tool pprof -seconds=30 http://localhost:6060/debug/pprof/heap

# در production با continuous monitoring:
# Pyroscope یا Parca استفاده کن
```

**Goroutine leak — مثال و fix:**

```go
// LEAK:
func leaky(ch chan int) {
    go func() {
        val := <-ch  // اگه ch هرگز داده نفرسته → goroutine leak!
        process(val)
    }()
}

// FIX:
func fixed(ctx context.Context, ch chan int) {
    go func() {
        select {
        case val := <-ch:
            process(val)
        case <-ctx.Done():
            return  // clean exit
        }
    }()
}
```

---

# حوزه ۳: Kubernetes Internals

---

## ۱. Pod Creation — دقیق‌ترین Sequence of Events

kubectl apply -f pod.yaml
     ↓
۱. kubectl: YAML → JSON، validation، POST /api/v1/namespaces/default/pods
     ↓
۲. kube-apiserver:
   - Authentication (token/cert)
   - Authorization (RBAC)
   - Admission Controllers (Mutating → Validating)
     * MutatingWebhookConfiguration: inject sidecar، default values
     * LimitRanger: default resource requests
     * ValidatingWebhookConfiguration: policy check
   - Persist به etcd
   - Return 201 Created
     ↓
۳. kube-scheduler (watch → event):
   - Pod رو با Spec.NodeName="" می‌بینه
   - Filtering: nodes که predicate رو رد می‌کنن حذف کن
     (resource fit, taints/tolerations, affinity, ...)
   - Scoring: بقیه nodes امتیاز بگیرن
   - Bind: NodeName رو set کن در apiserver
     ↓
۴. kubelet روی اون node (watch → event):
   - Pod رو با NodeName=خودش می‌بینه
   - Container Runtime Interface (CRI) رو صدا می‌زنه
     ↓
۵. containerd/CRI-O:
   - CNI plugin رو صدا می‌زنه → network namespace بساز، IP بده
   - PodSandbox (pause container) بساز
   - Image pull اگه لازمه
   - Container‌ها رو بساز و start کن
     ↓
۶. kubelet:
   - Probe‌ها رو اجرا کن (startup → liveness → readiness)
   - Status رو به apiserver report کن
     ↓
۷. endpoints controller:
   - Pod ready شد → Endpoints/EndpointSlice update کن
     ↓
۸. kube-proxy:
   - EndpointSlice تغییر کرد → iptables/ipvs rules update کن


---

## ۲. kube-proxy — iptables vs IPVS

**iptables mode:**

```bash
# kube-proxy یه chain ایجاد می‌کنه:
iptables -t nat -L KUBE-SERVICES
# KUBE-SVC-XXX → KUBE-SEP-YYY → DNAT به PodIP

# مشکل: O(n) traversal برای هر packet!
# با 10000 service: 10000 iptables rule چک میشه
# latency افزایش پیدا می‌کنه
```

**IPVS mode:**

```bash
# IPVS از hash table استفاده می‌کنه → O(1) lookup
ipvsadm -L -n
# TCP  10.96.0.1:443 rr   ← scheduling algorithm
#   -> 10.0.0.1:6443      weight 1

# مزایا:
# - O(1) vs O(n)
# - Load balancing algorithms: rr, lc, sh, ...
# - Native keepalive
# - Better performance در scale

# فعال کردن:
# kube-proxy --proxy-mode=ipvs
# نیاز به: modprobe ip_vs ip_vs_rr ip_vs_wrr ip_vs_sh
```

---

## ۳. چرا etcd؟ چرا نه PostgreSQL؟

etcd برای Kubernetes انتخاب درستیه چون:

۱. Watch API — First-class citizen:
   PostgreSQL: LISTEN/NOTIFY محدود، polling لازمه
   etcd: watch روی key prefix، ordered events، compaction

۲. Raft Consensus — Strong Consistency:
   هر write به majority of nodes قبل از ack
   No split-brain
   PostgreSQL: HA پیچیده‌تره، replica lag داره

۳. Lease/TTL:
   etcd: native TTL روی keys — برای node heartbeat عالیه
   PostgreSQL: manual cleanup لازم

۴. Simple data model:
   Kubernetes فقط key-value + watch نیاز داره
   نه join، نه complex query
   etcd برای این use case over-fit نشده

۵. ResourceVersion = etcd revision:
   هر write یه revision داره
   Kubernetes از این برای optimistic locking استفاده می‌کنه


**مشکلات etcd:**

```bash
# etcd برای large clusters مشکل داره:
# - 8GB DB size limit (v3 default)
# - عملکرد با 5000+ nodes افت می‌کنه
# راه‌حل: etcd compaction، separate etcd برای events

# اندازه‌گیری:
etcdctl endpoint status --write-out=table
etcdctl defrag  # فضا آزاد کن
```

---

## ۴. Informer و SharedInformer

**بدون Informer — مشکل:**

```go
// هر controller مستقیم API poll کنه:
for {
    pods, _ := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
    // process...
    time.Sleep(5 * time.Second)
}
// مشکل: N controller × API call = apiserver overwhelmed
```

**Informer — راه‌حل:**

```go
// Informer = List + Watch با local cache

factory := informers.NewSharedInformerFactory(client, 30*time.Second)
podInformer := factory.Core().V1().Pods()

// ۱. List: همه pods رو یه‌بار می‌گیره، cache می‌کنه
// ۲. Watch: از همون resourceVersion ادامه میده، event‌ها رو اعمال می‌کنه

podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) {
        pod := obj.(*v1.Pod)
        // از local cache خوندیم، نه API call!
    },
    UpdateFunc: func(old, new interface{}) {},
    DeleteFunc: func(obj interface{}) {},
})
```

**SharedInformer:**

```go
// N controller روی همون resource → فقط یه connection به apiserver
// cache shared میشه بین همه controller‌ها

// مثال: Deployment controller و ReplicaSet controller
// هر دو به Pod events نیاز دارن
// SharedInformerFactory یه watch می‌زنه، هر دو event می‌گیرن
```

---

## ۵. ResourceVersion

```go
// ResourceVersion = etcd revision number
// هر object در Kubernetes یه ResourceVersion داره

// وقتی list می‌کنی:
pods, _ := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
// pods.ResourceVersion = "12345"  ← snapshot در این moment

// Watch از این point:
watcher, _ := client.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
    ResourceVersion: "12345",  // فقط events بعد از این
})

// Optimistic locking — جلوگیری از lost update:
pod.Spec.Containers[0].Image = "new-image:v2"
_, err := client.CoreV1().Pods("default").Update(ctx, pod, metav1.UpdateOptions{})
// اگه بین Get و Update یکی دیگه تغییر داده باشه:
// err = "the object has been modified; please apply your changes to the latest version"
// باید دوباره Get کنی و apply کنی
```

---

## ۶. وقتی Node ناسالم میشه — Response Chain

Node متوقف میشه (network partition یا crash)
     ↓
۱. kubelet heartbeat قطع میشه
   - kubelet هر 10s به apiserver heartbeat می‌فرسته (node-status-update-frequency)
     ↓
۲. node-lifecycle-controller (توی kube-controller-manager):
   - بعد از node-monitor-grace-period (default 40s): Node → Unknown
   - بعد از pod-eviction-timeout (default 5min): Pod‌ها تaint میشن با NoExecute
     ↓
۳. taint-based eviction:
   - node.kubernetes.io/unreachable:NoExecute
   - Pod‌ها با tolerationSeconds=300 (default) اجرا میشن
   - بعد از 5min: Pods schedule میشن روی node‌های دیگه
     ↓
۴. kube-scheduler:
   - Pods رو روی healthy nodes schedule می‌کنه
     ↓
۵. endpoint-slice-controller:
   - Node ناسالمه → Pods از EndpointSlice حذف میشن
   - kube-proxy rules update میشه → traffic دیگه به اون pods نمیره


**نکته مهم:**

```bash
# اگه StatefulSet داری:
# Pod‌ها automatically re-schedule نمیشن تا مطمئن بشی node واقعاً down شده
# (جلوگیری از split-brain در stateful apps)
# باید manually: kubectl delete pod <pod> --force --grace-period=0
```

---

## ۷. PodDisruptionBudget

```yaml
# تضمین می‌کنه هنگام voluntary disruption حداقل N pod سالم بمونه

apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: api-pdb
spec:
  minAvailable: 2        # یا maxUnavailable: 1
  selector:
    matchLabels:
      app: api
```

```bash
# چطور کار می‌کنه:
# kubectl drain node → kubelet evict pods می‌کنه
# قبل از evict هر pod:
#   PDB controller چک می‌کنه: آیا evict این pod PDB رو violate می‌کنه؟
#   اگه بله: eviction reject میشه (429 Too Many Requests)
#   kubectl drain منتظر می‌مونه

# مثال:
# deployment: 3 replicas
# PDB: minAvailable=2
# drain: یه pod evict میشه → 2 باقی (OK)
#        دومی → 1 باقی → REJECTED، drain منتظر میمونه تا pod جدید ready بشه
```

---

## ۸. Istio — Traffic Interception بدون Code Change

Magic: iptables + sidecar injection

۱. Mutating Webhook:
   - Pod create request → Istio webhook intercept می‌کنه
   - Envoy sidecar container inject می‌کنه
   - initContainer (istio-init) هم inject می‌کنه

۲. initContainer (istio-init) اجرا میشه:


```bash
# این iptables rules رو اضافه می‌کنه:
iptables -t nat -A PREROUTING -p tcp -j REDIRECT --to-port 15001
# همه inbound traffic → Envoy port 15001

iptables -t nat -A OUTPUT -p tcp -j REDIRECT --to-port 15001
# همه outbound traffic → Envoy port 15001

# استثنا: Envoy خودش (UID 1337) تا loop نشه
iptables -t nat -A OUTPUT -m owner --uid-owner 1337 -j RETURN
```

۳. App container start میشه:
   - App فکر می‌کنه مستقیم به سرویس وصله
   - ولی همه traffic از Envoy رد میشه
   - Envoy: mTLS، retry، circuit breaker، telemetry

همه اینا WITHOUT code change!


---

# حوزه ۴: Distributed Systems Concepts

---

## ۱. CAP Theorem — با مثال واقعی

$$CAP: \text{Consistency} + \text{Availability} + \text{Partition Tolerance}$$

**در network partition فقط می‌تونی دو تا داشته باشی — ولی P اجباریه، پس انتخاب CP یا AP هست.**

**مثال واقعی با stack:**

سناریو: Redis Cluster با 3 node، network partition اتفاق افتاده

                [Node A] — ✗ — [Node B] [Node C]
                  ↑
                write

CP choice (etcd, Zookeeper):
  → Node A می‌بینه majority نداره → write رو reject می‌کنه
  → Consistent ولی unavailable برای write

AP choice (Redis Cluster default, Cassandra):
  → Node A write رو accept می‌کنه
  → بعد از partition heal: conflict resolution (last-write-wins)
  → Available ولی inconsistent


```python
# Redis: AP — eventual consistency
# وقتی partition heal میشه، آخرین write برنده میشه
# مناسب برای: session cache، rate limiting، non-critical data

# PostgreSQL با synchronous replication: CP
# write تا majority commit نکنه ack نمیده
# مناسب برای: financial data، inventory، هر جا consistency حیاتیه
```

---

## ۲. PostgreSQL ACID و MVCC

**MVCC (Multi-Version Concurrency Control):**

```sql
-- بدون MVCC: read باید منتظر write lock بمونه
-- با MVCC: هر transaction snapshot خودش رو می‌بینه

-- وقتی UPDATE می‌کنی:
UPDATE accounts SET balance = balance - 100 WHERE id = 1;

-- PostgreSQL:
-- ۱. row قدیمی را dead tuple می‌کنه (xmax set می‌کنه)
-- ۲. row جدید درج می‌کنه (xmin = current txn id)
-- ۳. reader‌های قدیمی هنوز old version رو می‌بینن!

-- Reader-Writer blocking نداره:
BEGIN;
SELECT * FROM accounts WHERE id = 1;
-- snapshot در این لحظه گرفته شد
-- concurrent UPDATE‌ها block نمیشه و block نمیکنه
COMMIT;
```

**ACID چطور guarantee میشه:**

Atomicity: WAL (Write-Ahead Log) — قبل از data نوشته میشه
           Crash → WAL replay → یا commit کامله یا rollback کامله

Consistency: Constraints (FK، CHECK، UNIQUE) + Trigger
             در transaction enforce میشن

Isolation: MVCC + Lock levels
           READ COMMITTED: هر statement snapshot جدید
           REPEATABLE READ: کل transaction یه snapshot
           SERIALIZABLE: SSI (Serializable Snapshot Isolation)

Durability: fsync → WAL به disk فلاش میشه قبل از commit ack


```sql
-- مثال MVCC مشکل: Phantom Read
-- READ COMMITTED:
BEGIN;
SELECT COUNT(*) FROM orders WHERE status = 'pending';  -- 10
-- concurrent INSERT یه order جدید
SELECT COUNT(*) FROM orders WHERE status = 'pending';  -- 11!  ← phantom

-- REPEATABLE READ:
-- هر دو query: 10  ← snapshot یه باره گرفته شده
```

---

## ۳. Redis — چرا Single-Threaded؟

Redis از Go 1.x نیست — C هست و single-threaded event loop داره

چرا single-threaded مزیته؟

۱. No locking overhead:
   Multi-threaded: mutex، spinlock، cache-line bouncing
   Single-threaded: هیچکدام — ساده‌تر، کمتر bug

۲. CPU cache locality:
   همه data structures در یه thread → L1/L2 cache hit rate بالا

۳. Atomic operations by design:
   MULTI/EXEC (transaction) واقعاً atomic هست
   در multi-threaded نیاز به distributed lock داشت

۴. I/O bottleneck نه CPU:
   Redis عملاً I/O bound هست نه CPU bound
   epoll/kqueue با single thread کافیه

--- اما ---
Redis 6.0: I/O threads اضافه شد (نه main thread)
   main thread: commands رو execute می‌کنه (هنوز single-threaded)
   I/O threads: network read/write رو parallelize می‌کنن


---

## ۴. ClickHouse vs PostgreSQL — Storage Architecture

PostgreSQL: Row-oriented storage
ClickHouse: Columnar storage

مثال جدول:
| id | user_id | event | timestamp | country |

PostgreSQL در disk:
[1, 1001, 'click', 2024-01-01, 'IR']
[2, 1002, 'view',  2024-01-01, 'US']
[3, 1001, 'click', 2024-01-02, 'IR']
← هر row کنار هم

ClickHouse در disk:
id:        [1, 2, 3, ...]
user_id:   [1001, 1002, 1001, ...]
event:     ['click', 'view', 'click', ...]
timestamp: [2024-01-01, ...]
country:   ['IR', 'US', 'IR', ...]
← هر column جداگانه


```sql
-- Query analytics:
SELECT country, COUNT(*) FROM events WHERE event = 'click' GROUP BY country

-- PostgreSQL: باید همه columns رو بخونه (حتی user_id که نمیخوایم)
-- ClickHouse: فقط 'event' و 'country' columns رو از disk می‌خونه!
--             → 10x-100x کمتر I/O

-- علاوه بر این، ClickHouse:
-- ۱. Compression per column (همه event‌ها string → فشرده‌سازی عالی)
-- ۲. Vectorized execution (SIMD instructions)
-- ۳. MergeTree: data sorted by primary key → range scan سریع
```

**کِی ClickHouse، کِی PostgreSQL:**

ClickHouse: analytics، aggregation، log analysis، time-series
            append-mostly، batch insert
            نه: single row lookup، UPDATE/DELETE زیاد، ACID transaction

PostgreSQL: OLTP، transactional، normalized data
            UPDATE/DELETE، complex JOIN، ACID critical


---

## ۵. Exactly-Once Delivery

**سه level:**

At-most-once:   ممکنه message گم بشه، تکرار نمیشه
At-least-once:  تکرار ممکنه، گم نمیشه  ← default در Kafka
Exactly-once:   نه گم، نه تکرار        ← سخت‌ترین


**پیاده‌سازی در Kafka:**

```python
# Producer side — Idempotent producer:
# هر message یه sequence number داره
# broker تشخیص میده duplicate رو و drop می‌کنه

producer = KafkaProducer(
    enable_idempotence=True,        # producer ID + sequence number
    acks='all',                     # همه replicas confirm کنن
    retries=sys.maxsize
)

# Transactional (cross-partition exactly-once):
producer.init_transactions()
producer.begin_transaction()
producer.send('topic-a', value=b'data')
producer.send('topic-b', value=b'data')
producer.commit_transaction()  # atomic cross-partition
```

**Consumer side — Idempotent processing:**

```python
# Kafka exactly-once تنها تضمین میده که message یه‌بار deliver بشه
# ولی اگه consumer crash کنه بعد از process و قبل از commit؟

# راه‌حل: Idempotent consumer
# هر message یه unique ID داره
# قبل از process: چک کن این ID رو قبلاً process کردی؟

def process_message(msg):
    msg_id = msg.headers['message-id']
    
    with db.transaction():
        if db.exists('processed_ids', msg_id):
            return  # already processed, skip
        
        # business logic
        db.insert('orders', parse_order(msg.value))
        db.insert('processed_ids', msg_id)
        # هر دو در یه transaction → atomic
```

---

## ۶. چرا Distributed Locking سخته؟

فرض کن می‌خوای یه distributed lock با Redis پیاده کنی:

ساده (اشتباه):
SET lock "value" NX EX 30

مشکل ۱ — Clock skew:
  Process A lock می‌گیره، TTL=30s
  Process A GC pause می‌کنه برای 35s
  TTL expire میشه، Process B lock می‌گیره
  Process A از pause برمیگرده، فکر می‌کنه هنوز lock داره
  → دو process همزمان در critical section!


```python
# راه‌حل: Fencing Token
# هر بار lock گرفته میشه، یه monotonically increasing token
import redis

def acquire_lock(client, resource, ttl):
    token = client.incr('lock:token:' + resource)  # atomic increment
    success = client.set(f'lock:{resource}', token, nx=True, ex=ttl)
    return token if success else None

# Resource server باید token رو validate کنه:
def write_data(data, fencing_token):
    if fencing_token <= last_seen_token:
        raise StaleTokenError("Lock expired!")
    # proceed
    last_seen_token = fencing_token
```

**Redlock — Martin Kleppmann critique:**

Redlock: lock رو روی N>=5 Redis instance بگیر، majority قبول کنن

مشکل: هنوز GC pause مشکل داره
Martin Kleppmann: "Redlock is not safe for distributed systems"
Redis author (antirez): "برای loose correctness requirements کافیه"

واقعیت عملی:
- برای idempotent operations: Redis lock کافیه
- برای strong correctness: از database lock یا ZooKeeper/etcd استفاده کن

etcd: lease + keepalive + revision-based fencing → خیلی قوی‌تره


```go
// etcd distributed lock — production-grade:
cli, _ := clientv3.New(clientv3.Config{Endpoints: []string{"etcd:2379"}})
session, _ := concurrency.NewSession(cli, concurrency.WithTTL(30))
mutex := concurrency.NewMutex(session, "/locks/critical-section")

if err := mutex.Lock(ctx); err != nil {
    log.Fatal(err)
}
defer mutex.Unlock(ctx)

// etcd lease keepalive: اگه process crash کنه، session expire میشه
// fencing token = etcd revision → storage layer می‌تونه validate کنه
```

---

# خلاصه تشخیص سطح من

| حوزه | خودارزیابی |
|-------|------------|
| Linux & Networking | ۸/۸ |
| Go Internals | ۸/۸ |
| Kubernetes Internals | ۸/۸ |
| Distributed Systems | ۶/۶ |

**نکته مهم:** دانش بدون تجربه عملی ارزش کمتری داره. هر چیزی که گفتم از production incidents، debugging sessions، و code review تجربه شده — نه فقط documentation خوندن.
