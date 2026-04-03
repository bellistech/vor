# The Mathematics of lsof — File Descriptors, Resource Limits & Leak Detection

> *lsof maps the kernel's file descriptor tables into human-readable form. Every open file, socket, pipe, and device is a numbered entry in a per-process table, and lsof reveals the size, shape, and health of that table.*

---

## 1. File Descriptor Architecture

### The Three-Level Table

Linux uses a three-tier structure for open files:

```
Process → fd table (per-process) → file description (system-wide) → inode
```

$$fd\_count_{process} = |\{fd : fd \in fd\_table(pid)\}|$$

$$file\_descriptions_{system} = \sum_{p} fd\_count_p - shared\_fds$$

### File Descriptor Number Space

File descriptors are non-negative integers: $fd \in \{0, 1, 2, ..., fd\_max - 1\}$.

Reserved: 0 (stdin), 1 (stdout), 2 (stderr).

$$available\_fds = fd\_limit - 3 - reserved\_by\_runtime$$

### Per-Process fd Limit

$$nofile = \min(soft\_limit, hard\_limit, fs.nr\_open)$$

| Level | Default | Maximum | Set By |
|:---|:---:|:---:|:---|
| Soft limit | 1024 | hard limit | ulimit -n |
| Hard limit | 1048576 | nr_open | ulimit -Hn |
| fs.nr_open | 1048576 | 2^31-1 | sysctl |
| fs.file-max | ~RAM/10 | ~2^63 | sysctl (system-wide) |

---

## 2. System-Wide File Descriptor Accounting

### Total Open Files

$$total\_fds = \sum_{p \in processes} fd\_count(p)$$

Read from `/proc/sys/fs/file-nr`: `allocated  0  max`

### Memory Cost per File Descriptor

| Structure | Size (bytes) | Notes |
|:---|:---:|:---|
| struct file | 256 | Open file description |
| struct fdtable entry | 8 | Pointer in fd array |
| struct dentry | 192 | Cached directory entry |
| struct inode | 600 | Cached inode (shared) |
| **Per unique fd** | **~460** | Excluding shared inodes |

$$memory_{all\_fds} = total\_fds \times 460 \text{ bytes}$$

1 million open fds: $\approx 440 \text{ MB}$ kernel memory.

### File Descriptor Exhaustion

When $total\_fds \to fs.file\text{-}max$:

$$remaining = fs.file\text{-}max - total\_fds$$

At $remaining = 0$: all `open()`, `socket()`, `accept()` calls fail with `EMFILE` or `ENFILE`.

---

## 3. Socket Counting — Network Connection Analysis

### TCP Connection States

lsof shows socket states. Connection distribution:

$$N_{tcp} = N_{ESTABLISHED} + N_{TIME\_WAIT} + N_{CLOSE\_WAIT} + N_{LISTEN} + ...$$

### TIME_WAIT Accumulation

$$N_{TIME\_WAIT} = connection\_close\_rate \times 2 \times MSL$$

Where $MSL = 60s$ (Maximum Segment Lifetime, Linux default):

$$N_{TIME\_WAIT} = close\_rate \times 120s$$

| Close Rate | TIME_WAIT sockets |
|:---:|:---:|
| 10/s | 1,200 |
| 100/s | 12,000 |
| 1000/s | 120,000 |
| 5000/s | 600,000 |

Each TIME_WAIT socket holds $\approx 300$ bytes of kernel memory + 1 file descriptor.

### CLOSE_WAIT Detection (Leak Indicator)

CLOSE_WAIT means the remote end closed but the local application hasn't:

$$leak\_rate = \frac{d(N_{CLOSE\_WAIT})}{dt}$$

If $leak\_rate > 0$ for sustained periods → application is leaking connections.

---

## 4. File Descriptor Leak Detection

### Leak Rate Calculation

Sample fd count over time:

$$fd\_count(t_1), fd\_count(t_2), ..., fd\_count(t_n)$$

$$leak\_rate = \frac{fd\_count(t_n) - fd\_count(t_1)}{t_n - t_1}$$

### Time to Exhaustion

$$T_{exhaustion} = \frac{fd\_limit - fd\_count_{current}}{leak\_rate}$$

**Example:** Process has 500 fds, limit 1024, gaining 2 fds/minute:

$$T_{exhaustion} = \frac{1024 - 500}{2} = 262 \text{ minutes} \approx 4.4 \text{ hours}$$

### Leak Classification

| Pattern | lsof Signature | Diagnosis |
|:---|:---|:---|
| Growing REG files | `fd count ↑`, all type REG | Files opened but not closed |
| Growing sockets | `fd count ↑`, all type sock | Connection leak |
| Growing pipes | `fd count ↑`, all type FIFO | Pipe leak (fork without close) |
| Stable but high | `fd count` constant, near limit | Not a leak, needs higher limit |

---

## 5. Performance of lsof Itself

### Scanning Cost

lsof reads `/proc/[pid]/fd/` for every process:

$$T_{lsof} = N_{processes} \times (T_{readdir\_fd} + fd\_count \times T_{readlink})$$

| Component | Cost |
|:---|:---:|
| Open `/proc/[pid]/fd/` | 10-50 us |
| `readlink()` per fd | 1-5 us |
| `stat()` for type info | 5-10 us |

**Example:** 500 processes, average 50 fds each:

$$T = 500 \times (25\mu s + 50 \times 3\mu s) = 500 \times 175\mu s = 87.5ms$$

### Filtering Optimization

| Filter | Speedup | Mechanism |
|:---|:---|:---|
| `-p PID` | $\frac{N_{proc}}{1}$ | Single process only |
| `-i :PORT` | $\approx 10\times$ | Skip non-socket fds |
| `-u USER` | $\frac{N_{proc}}{N_{user\_proc}}$ | Skip other users |
| `+D /path` | Variable | Only fds under path |
| No flags | 1x (baseline) | Full system scan |

---

## 6. Deleted File Space Recovery

### The "Deleted but Open" Problem

When a file is deleted but still held open:

$$disk\_used = \sum_{f \in deleted\_open} size(f)$$

This space is **invisible** to `du` and `df` disagrees with `du`:

$$df\_used - du\_used = \sum deleted\_open\_files$$

### Finding and Recovering

```bash
lsof +L1  # Show files with link count < 1 (deleted)
```

Space freed only when the last fd is closed:

$$space\_freed(f) = \begin{cases} size(f) & \text{if } refcount(f) = 0 \\ 0 & \text{if } refcount(f) > 0 \end{cases}$$

### Worked Example

A log file deleted while the application still writes to it:

$$growth\_rate = 10 \text{ MB/min}$$

$$current\_size = 50 \text{ GB}$$

$$T_{disk\_full} = \frac{free\_space}{growth\_rate}$$

Fix: Restart the process (closes fd) → 50 GB freed instantly.

---

## 7. Repeat Mode (-r) — Continuous Monitoring

### Sampling Mathematics

`lsof -r INTERVAL` repeats the scan:

$$samples = \frac{duration}{interval}$$

$$overhead = \frac{T_{scan}}{interval} \times 100\%$$

**Example:** `lsof -r 5` on a system with 87ms scan time:

$$overhead = \frac{87ms}{5000ms} = 1.7\%$$

### Trend Detection

With repeat mode, compute:

$$\Delta fd = fd\_count(t) - fd\_count(t - interval)$$

$$trend = \frac{\sum_{i=1}^{n} \Delta fd_i}{n}$$

If $trend > 0$ consistently → leak confirmed.

---

## 8. Summary of lsof Mathematics

| Concept | Formula | Domain |
|:---|:---|:---|
| fd cost | $total\_fds \times 460$ bytes | Memory |
| TIME_WAIT | $close\_rate \times 2 \times MSL$ | Networking |
| Leak rate | $\Delta fd / \Delta t$ | Monitoring |
| Time to exhaustion | $(limit - current) / leak\_rate$ | Capacity |
| Scan cost | $N_{proc} \times (T_{dir} + fds \times T_{link})$ | Performance |
| Deleted space | $df_{used} - du_{used}$ | Disk accounting |
| Filtering speedup | $N_{total} / N_{filtered}$ | Optimization |

---

*lsof shows you what the kernel is holding onto — every file, every socket, every pipe. When your system says "too many open files," lsof tells you who opened them and why they're still there.*
