# The Mathematics of dmesg — Kernel Ring Buffer, Message Rates & Boot Diagnostics

> *dmesg reads the kernel's ring buffer — a fixed-size circular data structure where the oldest messages are overwritten by the newest. Understanding its capacity, overflow behavior, and timestamp mechanics is essential for kernel debugging.*

---

## 1. Ring Buffer Architecture

### Circular Buffer Model

The kernel log buffer is a **circular buffer** of fixed size:

$$buffer\_size = 2^{log\_buf\_len}$$

Default: $log\_buf\_len = 17 \implies 2^{17} = 131072 = 128 \text{ KB}$

Settable via boot parameter: `log_buf_len=N` (rounded up to power of 2).

### Capacity in Messages

Average kernel message: 80-120 bytes (including metadata).

$$max\_messages \approx \frac{buffer\_size}{avg\_message\_size}$$

| Buffer Size | ~Messages at 100 B/msg |
|:---:|:---:|
| 64 KB | 655 |
| 128 KB (default) | 1,310 |
| 512 KB | 5,242 |
| 1 MB | 10,485 |
| 16 MB | 167,772 |

### Overflow and Message Loss

When the buffer wraps, old messages are silently overwritten:

$$retention\_time = \frac{buffer\_size}{message\_rate \times avg\_size}$$

**Example:** 128 KB buffer, 50 messages/second at 100 bytes:

$$retention = \frac{131072}{50 \times 100} = 26.2 \text{ seconds}$$

After 26 seconds, the oldest messages are gone. This is why busy systems increase `log_buf_len`.

---

## 2. Timestamp Mechanics

### Monotonic Clock

dmesg timestamps use the kernel's **monotonic clock** (seconds since boot):

$$timestamp = \frac{clock\_monotonic\_ns}{10^9}$$

Format: `[seconds.microseconds]` — e.g., `[   12.345678]`

### Conversion to Wall Clock

$$wall\_time = boot\_time + timestamp_{monotonic}$$

Where $boot\_time$ can be found via:

$$boot\_time = time(now) - uptime$$

### Time Resolution

| Kernel Version | Resolution | Source |
|:---|:---:|:---|
| < 3.5 | 1 ms | jiffies (HZ=1000) |
| >= 3.5 | 1 us | `local_clock()` / TSC |
| With `printk_time=1` | 1 us | Always enabled since 2.6.11 |

### Boot Phase Timing

During boot, timestamps reveal initialization order and duration:

$$T_{phase} = timestamp_{end} - timestamp_{start}$$

$$T_{boot\_total} = \sum phases = timestamp_{last\_boot\_msg}$$

---

## 3. Log Levels — Severity as Filter

### Kernel Log Levels

$$level \in \{0, 1, 2, 3, 4, 5, 6, 7\}$$

| Level | Name | Macro | Console Default |
|:---:|:---|:---|:---:|
| 0 | KERN_EMERG | `pr_emerg()` | Always shown |
| 1 | KERN_ALERT | `pr_alert()` | Always shown |
| 2 | KERN_CRIT | `pr_crit()` | Always shown |
| 3 | KERN_ERR | `pr_err()` | Shown |
| 4 | KERN_WARNING | `pr_warn()` | Shown (default cutoff) |
| 5 | KERN_NOTICE | `pr_notice()` | Hidden |
| 6 | KERN_INFO | `pr_info()` | Hidden |
| 7 | KERN_DEBUG | `pr_debug()` | Hidden |

### Console Log Level

```
kernel.printk = current default minimum boot-time-default
```

Default: `4 4 1 7` — console shows levels 0-3.

### Filtering Efficiency

`dmesg -l err,crit,alert,emerg` (levels 0-3):

$$\%output = \frac{N_{0-3}}{N_{total}}$$

On a healthy system: $\%output \approx 1-5\%$.

---

## 4. Message Rate Analysis

### Rate Calculation

$$rate = \frac{N_{messages}}{T_{interval}}$$

### Rate Bursts During Boot

| Boot Phase | Typical Rate | Duration |
|:---|:---:|:---:|
| Early boot (hardware init) | 100-500 msg/s | 1-5 s |
| Driver loading | 200-1000 msg/s | 2-10 s |
| Filesystem mount | 10-50 msg/s | 1-3 s |
| Service startup | 5-20 msg/s | 5-30 s |
| Steady state | 0.1-5 msg/s | Continuous |

### Rate Limiting (printk_ratelimit)

The kernel rate-limits repeated messages:

$$allowed = ratelimit\_burst \text{ messages per } ratelimit\_jiffies / HZ \text{ seconds}$$

Default: 10 messages per 5 seconds.

$$suppressed = max(0, actual\_rate - \frac{burst}{interval}) \times interval$$

### Message Flood Impact

At extreme rates ($> 10000$ msg/s), printk becomes a bottleneck:

$$T_{printk} \approx 1-10\mu s \text{ per message (with console)}$$

$$\%CPU_{printk} = rate \times T_{printk}$$

At 10,000 msg/s: $\%CPU = 10000 \times 5\mu s = 50ms/s = 5\%$ of one core.

With serial console: $T_{printk} \approx 100\mu s$ (baud rate limited) → 100% of one core at 10K msg/s.

---

## 5. Hardware Error Detection

### Memory Errors (EDAC)

dmesg reports memory errors via the EDAC subsystem:

$$error\_rate = \frac{correctable\_errors}{time}$$

| Rate | Severity | Action |
|:---:|:---|:---|
| 0 errors/day | Normal | None |
| 1-10 CE/day | Warning | Monitor trend |
| > 10 CE/day | Critical | Plan DIMM replacement |
| Any UE | Fatal | System may crash |

### Predicting DIMM Failure

Studies show correctable error rate predicts uncorrectable errors:

$$P(UE | CE > threshold) \approx 20-70\%$$

If a DIMM produces $> 10$ correctable errors in 24 hours, failure probability increases dramatically.

### Disk Error Pattern

dmesg I/O errors follow patterns:

$$pattern = \begin{cases}
\text{Single sector} & \text{if } errors \text{ at same LBA} \\
\text{Growing region} & \text{if } \Delta LBA \text{ increases over time} \\
\text{Random scatter} & \text{if } LBA \text{ distribution uniform}
\end{cases}$$

Growing region → surface degradation → imminent failure.

---

## 6. OOM Killer Messages

### OOM Log Analysis

When the OOM killer activates, dmesg contains:

$$score(p) = \frac{RSS_p + swap_p}{total\_RAM + total\_swap} \times 1000 + oom\_score\_adj_p$$

The selected victim: $victim = \arg\max_p score(p)$

### Memory State at OOM

dmesg reports zone watermarks at OOM time:

```
Node 0 Normal: free:12345kB min:67890kB low:84862kB high:101835kB
```

$$OOM\_triggered \iff free_{all\_zones} < min_{all\_zones}$$

### OOM Frequency

$$MTBO = \frac{uptime}{OOM\_count} \text{ (Mean Time Between OOMs)}$$

| MTBO | Assessment |
|:---:|:---|
| > 30 days | Occasional, acceptable |
| 1-30 days | Frequent, needs tuning |
| < 1 day | Critical, needs more RAM |

---

## 7. Facility-Based Analysis

### dmesg vs journalctl for Kernel Messages

| Feature | dmesg | journalctl -k |
|:---|:---|:---|
| Source | Ring buffer (RAM) | Journal (disk) |
| Persistence | Lost on reboot | Survives reboot |
| Size limit | $2^{log\_buf\_len}$ | Journal size limit |
| Speed | $O(N)$ read | $O(\log N)$ seek |
| Boot history | Current boot only | All stored boots |

### When to Use Each

$$\text{Use dmesg when: } \begin{cases} \text{Current boot only needed} \\ \text{Journal not available (early boot, rescue)} \\ \text{Fastest access to kernel messages} \end{cases}$$

---

## 8. Summary of dmesg Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Buffer capacity | $2^{log\_buf\_len} / avg\_msg\_size$ | Power of 2 |
| Retention time | $buffer\_size / (rate \times avg\_size)$ | Duration |
| Timestamp | $boot\_time + monotonic\_ns / 10^9$ | Clock conversion |
| Rate limiting | $burst / interval$ | Throttling |
| printk CPU cost | $rate \times T_{printk}$ | Overhead |
| OOM score | $(RSS + swap) / (RAM + swap) \times 1000 + adj$ | Scoring |
| DIMM failure | $P(UE \mid CE > threshold)$ | Conditional probability |

## Prerequisites

- ring buffers, kernel architecture, log levels, monotonic clocks, rate limiting

---

*dmesg is the kernel talking to you. Its ring buffer is finite and its messages are ephemeral — but in the seconds after a crash, a hang, or a hardware failure, those messages are the most valuable data on the system.*
