# The Mathematics of sysctl — Kernel Tuning Parameters & Their Formulas

> *sysctl exposes the kernel's tuning knobs. Behind every parameter is a formula — buffer sizes, hash table dimensions, timer intervals — all computable, all interrelated.*

---

## 1. Network Buffer Sizing — TCP Memory

### TCP Buffer Auto-Tuning

The kernel auto-tunes TCP buffers within bounds set by sysctl:

```
net.ipv4.tcp_rmem = min default max
net.ipv4.tcp_wmem = min default max
```

### Bandwidth-Delay Product (BDP)

The optimal buffer size for maximum throughput:

$$BDP = bandwidth \times RTT$$

Where bandwidth is in bytes/sec and RTT is in seconds.

| Link | Bandwidth | RTT | BDP |
|:---|:---:|:---:|:---:|
| LAN (1 Gbps) | 125 MB/s | 0.5 ms | 62.5 KB |
| Metro (10 Gbps) | 1.25 GB/s | 5 ms | 6.25 MB |
| Cross-country (1 Gbps) | 125 MB/s | 60 ms | 7.5 MB |
| Intercontinental (1 Gbps) | 125 MB/s | 200 ms | 25 MB |

### Setting tcp_rmem/tcp_wmem

$$max\_buffer \geq BDP \times 2 \text{ (for safety margin)}$$

**Example:** Cross-country 10 Gbps link, 40 ms RTT:

$$BDP = 1.25 \times 10^9 \times 0.04 = 50 \text{ MB}$$

$$tcp\_rmem\_max = 100 \text{ MB} = 104857600$$

### Total TCP Memory

```
net.ipv4.tcp_mem = low pressure high  (in pages)
```

$$tcp\_mem\_high = \frac{total\_RAM \times fraction}{page\_size}$$

Typical fraction: 25-50% of RAM for TCP buffers. For 64 GB RAM:

$$tcp\_mem\_high = \frac{64 \times 10^9 \times 0.25}{4096} = 3,906,250 \text{ pages}$$

---

## 2. Connection Tracking Table — Hash Sizing

### nf_conntrack Hash Table

$$nf\_conntrack\_max = nf\_conntrack\_buckets \times 8$$

Each bucket is a linked list with average length 8 (load factor).

### Memory Cost

Each conntrack entry: ~320 bytes on 64-bit systems.

$$memory = nf\_conntrack\_max \times 320 \text{ bytes}$$

| Max Connections | Memory | Suitable For |
|:---:|:---:|:---|
| 65,536 | 20 MB | Small server |
| 262,144 | 80 MB | Medium server |
| 1,048,576 | 320 MB | Load balancer |
| 4,194,304 | 1.28 GB | Large NAT gateway |

### Lookup Complexity

With hash table and chaining:

$$T_{lookup} = O(1 + \alpha) \text{ where } \alpha = \frac{n}{buckets} = load\_factor$$

At default load factor 8: each lookup checks ~8 entries. To reduce to ~4:

$$buckets = \frac{nf\_conntrack\_max}{4}$$

---

## 3. Virtual Memory — Swappiness & Dirty Ratios

### vm.swappiness

Controls the kernel's preference for swapping anonymous pages vs dropping page cache:

$$\text{swap tendency} = mapped\_ratio + swappiness$$
$$\text{cache tendency} = (100 - mapped\_ratio) + (100 - swappiness)$$

The kernel swaps when $swap\_tendency > cache\_tendency$:

$$mapped\_ratio + swappiness > 200 - mapped\_ratio - swappiness$$

$$mapped\_ratio > 100 - swappiness$$

| swappiness | Meaning | Swap when mapped_ratio > |
|:---:|:---|:---:|
| 0 | Avoid swap (except OOM) | 100% (never, effectively) |
| 10 | Minimal swap (recommended for SSDs) | 90% |
| 60 | Default | 40% |
| 100 | Aggressive swap | 0% (always consider) |

### vm.dirty_ratio and vm.dirty_background_ratio

$$dirty\_bytes_{max} = total\_RAM \times \frac{dirty\_ratio}{100}$$

$$dirty\_bytes_{background} = total\_RAM \times \frac{dirty\_background\_ratio}{100}$$

| Threshold | Default | Trigger |
|:---|:---:|:---|
| dirty_background_ratio | 10% | Background writeback starts |
| dirty_ratio | 20% | Synchronous writeback (process blocks) |

**Example:** 32 GB RAM:

$$background = 32 \times 0.10 = 3.2 \text{ GB of dirty pages triggers background flush}$$
$$sync = 32 \times 0.20 = 6.4 \text{ GB of dirty pages blocks writers}$$

### Dirty Page Writeback Rate

$$T_{flush} = \frac{dirty\_bytes}{disk\_write\_speed}$$

With 3.2 GB dirty and 500 MB/s SSD: $T_{flush} = 6.4s$. If dirty data accumulates faster than disk can flush, writers eventually block.

---

## 4. File Descriptor Limits

### System-Wide Limit

```
fs.file-max = max_open_files
```

Default formula (kernel auto-calculates):

$$file\_max = \frac{total\_RAM\_pages}{10}$$

For 64 GB RAM: $\frac{64 \times 10^9 / 4096}{10} = 1,562,500$ files.

### Per-File Memory Cost

Each open file descriptor consumes kernel memory:

| Structure | Size | Notes |
|:---|:---:|:---|
| struct file | ~256 bytes | File object |
| struct dentry | ~192 bytes | Directory entry cache |
| struct inode | ~600 bytes | Inode cache (shared) |
| **Total per fd** | **~450 bytes** | Excluding inode sharing |

$$memory_{fds} = N_{fds} \times 450 \text{ bytes}$$

1 million open files: $\approx 430 \text{ MB}$ of kernel memory.

---

## 5. Network Queue Sizing

### net.core.somaxconn

Maximum listen() backlog:

$$somaxconn = \min(application\_backlog, net.core.somaxconn)$$

Default: 4096 (was 128 before kernel 5.4).

### Sizing for Connection Rate

$$required\_backlog \geq \lambda \times T_{accept}$$

Where $\lambda$ = connection arrival rate, $T_{accept}$ = time between accept() calls.

**Example:** 10,000 connections/second, accept loop takes 100 us each:

$$backlog \geq 10000 \times 0.0001 = 1 \text{ (tiny — accept is fast)}$$

But during bursts or application stalls:

$$backlog \geq \lambda_{peak} \times T_{stall} = 10000 \times 0.5s = 5000$$

### net.core.netdev_max_backlog

Packets queued per CPU before kernel processing. Default: 1000.

$$queue\_time = \frac{netdev\_max\_backlog}{packet\_rate\_per\_cpu}$$

At 1 million packets/sec/CPU: $queue\_time = 1000 / 10^6 = 1ms$ of buffering.

---

## 6. Kernel Semaphore and IPC Limits

### kernel.sem

Four values: `SEMMSL SEMMNS SEMOPM SEMMNI`

| Parameter | Meaning | Default | Formula |
|:---|:---|:---:|:---|
| SEMMSL | Max semaphores per set | 32000 | Per-set limit |
| SEMMNS | Max semaphores system-wide | 1,024,000 | $SEMMNI \times SEMMSL$ |
| SEMOPM | Max ops per semop() call | 500 | Per-call limit |
| SEMMNI | Max semaphore sets | 32000 | System-wide limit |

### Shared Memory (kernel.shmmax)

$$shmmax = max\_shared\_memory\_segment\_bytes$$

For databases: typically set to $\frac{RAM}{2}$ or $RAM - 2GB$.

$$shmall = \frac{shmmax}{page\_size} \text{ (in pages)}$$

---

## 7. ARP and Neighbor Cache

### net.ipv4.neigh.default.gc_thresh{1,2,3}

Controls the ARP cache garbage collection thresholds:

| Parameter | Default | Meaning |
|:---|:---:|:---|
| gc_thresh1 | 128 | Below this: no GC |
| gc_thresh2 | 512 | Above this: GC after 5 seconds |
| gc_thresh3 | 1024 | Hard maximum: immediate GC |

### Sizing for Large Networks

In a /16 network (65,536 hosts), if the server communicates with many hosts:

$$gc\_thresh3 \geq active\_neighbors \times 1.5$$

Each ARP entry: ~256 bytes. At 10,000 entries: $\approx 2.5 \text{ MB}$.

---

## 8. Summary of sysctl Mathematics

| Parameter | Formula | Domain |
|:---|:---|:---|
| TCP buffer max | $BDP \times 2$ | Network throughput |
| Conntrack memory | $max \times 320$ bytes | Firewall/NAT |
| Dirty ratio threshold | $RAM \times ratio / 100$ | I/O writeback |
| File max | $RAM\_pages / 10$ | Resource limits |
| Listen backlog | $\lambda_{peak} \times T_{stall}$ | Connection queuing |
| Swap tendency | $mapped\_ratio + swappiness$ | Memory management |
| ARP cache | $active\_neighbors \times 1.5$ | Network neighbor table |

## Prerequisites

- kernel internals, TCP/IP networking, virtual memory, bandwidth-delay product, connection tracking

---

*sysctl is the control plane for kernel behavior. Every parameter has a formula behind it, and the right value depends on your workload's mathematics — not on blog posts that say "just set it to X."*
