# The Mathematics of Proc-Sys -- Virtual Filesystem Mappings and Kernel Accounting Formulas

> *The /proc and /sys filesystems expose kernel data structures as file operations,*
> *creating a mathematical correspondence between in-memory objects and filesystem paths.*
> *Every value read from these files is computed from kernel counters, page tables, and scheduler state.*

---

## 1. Memory Accounting from /proc/meminfo (Additive Decomposition)

### The Problem

`/proc/meminfo` reports dozens of counters that overlap and nest. How do they relate mathematically?

### The Formula

Total memory decomposition:

$$\text{MemTotal} = \text{MemFree} + \text{Active} + \text{Inactive} + \text{Slab} + \text{PageTables} + \text{KernelStack} + \text{Other}$$

Available memory estimation (kernel function `si_mem_available()`):

$$\text{MemAvailable} = \text{MemFree} - \text{low\_watermark\_pages} + \text{pagecache\_reclaimable} + \text{slab\_reclaimable}$$

Where:

$$\text{pagecache\_reclaimable} = \text{Inactive(file)} + \text{Active(file)} - \min\left(\frac{\text{file\_pages}}{2}, \text{low\_watermark}\right)$$

$$\text{slab\_reclaimable} = \text{SReclaimable} - \min\left(\frac{\text{SReclaimable}}{2}, \text{low\_watermark}\right)$$

### Worked Examples

System with 16 GB RAM:

| Field | Value (KB) | Value (GB) | Meaning |
|-------|-----------|-----------|---------|
| MemTotal | 16,384,000 | 16.00 | Physical RAM |
| MemFree | 2,048,000 | 2.00 | Completely unused |
| Buffers | 512,000 | 0.50 | Block device cache |
| Cached | 5,120,000 | 5.00 | Page cache |
| Active | 6,144,000 | 6.00 | Recently accessed |
| Inactive | 4,096,000 | 4.00 | Candidates for reclaim |
| Slab | 768,000 | 0.75 | Kernel object cache |
| SReclaimable | 512,000 | 0.50 | Reclaimable slab |
| SUnreclaim | 256,000 | 0.25 | Non-reclaimable slab |

Available memory calculation (low_watermark = 66 MB):

$$\text{pagecache} = 5{,}120{,}000 - \min(2{,}560{,}000, \; 67{,}584) = 5{,}120{,}000 - 67{,}584 = 5{,}052{,}416 \text{ KB}$$

$$\text{slab} = 512{,}000 - \min(256{,}000, \; 67{,}584) = 512{,}000 - 67{,}584 = 444{,}416 \text{ KB}$$

$$\text{MemAvailable} = 2{,}048{,}000 - 67{,}584 + 5{,}052{,}416 + 444{,}416 = 7{,}477{,}248 \text{ KB} \approx 7.13 \text{ GB}$$

## 2. CPU Utilization from /proc/stat (Time Division)

### The Problem

`/proc/stat` reports cumulative CPU time in jiffies (typically 10ms ticks). How do we calculate current CPU utilization?

### The Formula

Read `/proc/stat` at times $t_1$ and $t_2$:

$$\text{total}_{t} = \text{user} + \text{nice} + \text{system} + \text{idle} + \text{iowait} + \text{irq} + \text{softirq} + \text{steal}$$

$$\Delta\text{total} = \text{total}_{t_2} - \text{total}_{t_1}$$

$$\text{CPU}\% = \frac{\Delta\text{total} - \Delta\text{idle} - \Delta\text{iowait}}{\Delta\text{total}} \times 100$$

Per-category utilization:

$$\text{user}\% = \frac{\Delta\text{user}}{\Delta\text{total}} \times 100$$

$$\text{iowait}\% = \frac{\Delta\text{iowait}}{\Delta\text{total}} \times 100$$

### Worked Examples

Two samples 1 second apart on 4 CPUs:

| Field | t1 | t2 | Delta |
|-------|-----|-----|-------|
| user | 100,000 | 100,300 | 300 |
| nice | 5,000 | 5,010 | 10 |
| system | 30,000 | 30,100 | 100 |
| idle | 250,000 | 250,150 | 150 |
| iowait | 10,000 | 10,040 | 40 |
| irq | 1,000 | 1,000 | 0 |
| softirq | 2,000 | 2,000 | 0 |
| steal | 0 | 0 | 0 |
| **total** | | | **600** |

$$\text{CPU}\% = \frac{600 - 150 - 40}{600} \times 100 = 68.3\%$$

$$\text{user}\% = \frac{300}{600} \times 100 = 50.0\%$$

$$\text{iowait}\% = \frac{40}{600} \times 100 = 6.7\%$$

Expected jiffies per second on 4 CPUs: $4 \times 100 = 400$ Hz. Measured 600 jiffies in 1 second suggests 6 logical CPUs (hyperthreading).

## 3. Process RSS vs PSS vs USS (Set Theory)

### The Problem

Multiple processes share memory pages (shared libraries, copy-on-write). How do RSS, PSS, and USS differ?

### The Formula

For process $p$ with page set $P$:

$$\text{USS}(p) = |\{x \in P : \text{sharers}(x) = 1\}| \times \text{page\_size}$$

$$\text{RSS}(p) = |P| \times \text{page\_size}$$

$$\text{PSS}(p) = \sum_{x \in P} \frac{\text{page\_size}}{\text{sharers}(x)}$$

Properties:

$$\text{USS} \leq \text{PSS} \leq \text{RSS}$$

$$\sum_{p} \text{PSS}(p) = \text{total\_physical\_memory\_used}$$

$$\sum_{p} \text{RSS}(p) \geq \text{total\_physical\_memory\_used} \quad \text{(overcounts sharing)}$$

### Worked Examples

Three nginx workers sharing libc.so (2 MB), libssl.so (1 MB):

| Memory Region | Pages | Shared By | Per-process RSS | Per-process PSS |
|--------------|-------|-----------|----------------|-----------------|
| libc.so (text) | 512 | 3 workers | 2 MB | 0.67 MB |
| libssl.so (text) | 256 | 3 workers | 1 MB | 0.33 MB |
| Private heap | 1024 | 1 (unique) | 4 MB | 4 MB |
| Private stack | 512 | 1 (unique) | 2 MB | 2 MB |

| Metric | Worker 1 | Worker 2 | Worker 3 | Sum |
|--------|---------|---------|---------|-----|
| RSS | 9 MB | 9 MB | 9 MB | 27 MB |
| PSS | 7 MB | 7 MB | 7 MB | 21 MB |
| USS | 6 MB | 6 MB | 6 MB | 18 MB |
| **Actual memory** | | | | **21 MB** |

RSS overcounts by 6 MB. PSS sums to the true total.

## 4. Load Average Interpretation (Exponential Moving Average)

### The Problem

`/proc/loadavg` reports 1, 5, and 15-minute load averages. What do these numbers mean mathematically?

### The Formula

Load average is an exponential moving average of the run queue length plus uninterruptible sleep count:

$$L(t) = L(t-1) \times e^{-\Delta t / \tau} + n(t) \times (1 - e^{-\Delta t / \tau})$$

where:
- $n(t)$ = number of running + uninterruptible processes at time $t$
- $\tau$ = time constant (60s, 300s, or 900s for 1/5/15 min averages)
- $\Delta t$ = sampling interval (kernel uses 5 seconds)

Decay constants:

$$e^{-5/60} \approx 0.9200 \quad \text{(1-min)}$$

$$e^{-5/300} \approx 0.9835 \quad \text{(5-min)}$$

$$e^{-5/900} \approx 0.9945 \quad \text{(15-min)}$$

### Worked Examples

Interpreting load average on an 8-CPU system:

| Load Average | vs CPUs | Interpretation |
|-------------|---------|----------------|
| 0.50, 0.50, 0.50 | 6.25% | Very light load |
| 4.00, 4.00, 4.00 | 50% | Moderate, healthy |
| 8.00, 8.00, 8.00 | 100% | Fully utilized, no queue |
| 16.00, 8.00, 4.00 | 200%/100%/50% | Load spike, recovering |
| 4.00, 8.00, 16.00 | 50%/100%/200% | Load decreasing, was high |

Rule of thumb: load average should be below CPU count. Above = tasks are waiting.

Time for load average to reflect a change:

| Average | 63% settled | 95% settled | 99% settled |
|---------|------------|------------|------------|
| 1-min | 1 min | 3 min | 5 min |
| 5-min | 5 min | 15 min | 25 min |
| 15-min | 15 min | 45 min | 75 min |

## 5. Virtual Filesystem to Kernel Mapping (Function Mapping)

### The Problem

Each `/proc` and `/sys` file maps to a kernel function. How does the virtual filesystem translate file operations?

### The Formula

The mapping is a function from filesystem path to kernel data:

$$f: \text{path} \to \text{kernel\_function}(\text{arguments})$$

Read operation:

$$\text{read}(\text{/proc/meminfo}) \to \text{meminfo\_proc\_show}(\text{seq\_file})$$

$$\text{read}(\text{/proc/PID/status}) \to \text{proc\_pid\_status}(\text{task\_struct}, \text{seq\_file})$$

Write operation:

$$\text{write}(\text{/proc/sys/vm/swappiness}, v) \to \text{sysctl\_handler}(\text{vm.swappiness} = v)$$

### Worked Examples

| Path | Kernel Function | Data Source |
|------|----------------|-------------|
| `/proc/meminfo` | `meminfo_proc_show()` | `si_meminfo()` struct |
| `/proc/PID/status` | `proc_pid_status()` | `task_struct` |
| `/proc/PID/maps` | `show_map_vma()` | `vm_area_struct` list |
| `/proc/stat` | `show_stat()` | per-CPU `kernel_stat` |
| `/proc/loadavg` | `loadavg_proc_show()` | `avenrun[]` array |
| `/sys/block/sda/stat` | `part_stat_show()` | `disk_stats` |
| `/sys/class/net/eth0/speed` | `netdev_show()` | `net_device` |

Each read returns a snapshot. There is no locking guarantee between reading multiple files.

## 6. TCP Buffer Auto-Tuning (Control Theory)

### The Problem

The kernel auto-tunes TCP buffer sizes based on congestion and RTT. How do the `/proc/sys/net/ipv4/tcp_rmem` and `tcp_wmem` parameters interact with auto-tuning?

### The Formula

Buffer size calculation:

$$\text{BDP} = \text{bandwidth} \times \text{RTT}$$

$$\text{optimal\_buffer} = 2 \times \text{BDP}$$

Auto-tuning range from `tcp_rmem` (min default max):

$$\text{buffer} \in [\text{min}, \; \text{max}]$$

$$\text{auto\_tuned\_buffer} = \min(\text{optimal\_buffer}, \; \text{max})$$

### Worked Examples

| Link | Bandwidth | RTT | BDP | Optimal Buffer | Default tcp_rmem max |
|------|-----------|-----|-----|----------------|---------------------|
| LAN | 1 Gbps | 0.5 ms | 62.5 KB | 125 KB | 6 MB (sufficient) |
| WAN | 100 Mbps | 50 ms | 625 KB | 1.25 MB | 6 MB (sufficient) |
| Intercontinental | 1 Gbps | 150 ms | 18.75 MB | 37.5 MB | 6 MB (too small!) |
| Satellite | 10 Mbps | 600 ms | 750 KB | 1.5 MB | 6 MB (sufficient) |

For high-BDP links, increase tcp_rmem max:

$$\text{tcp\_rmem max} \geq 2 \times \text{bandwidth} \times \text{RTT}$$

Total memory for $C$ connections at max buffer:

$$\text{mem}_{\text{tcp}} = C \times (\text{rmem} + \text{wmem}) \leq \text{tcp\_mem max} \times \text{page\_size}$$

## 7. Dirty Page Writeback Thresholds (Threshold Control)

### The Problem

The kernel uses `dirty_ratio` and `dirty_background_ratio` to control when page cache dirty pages are written to disk. How do these thresholds affect write performance?

### The Formula

$$\text{dirty\_threshold} = \text{MemTotal} \times \frac{\text{dirty\_ratio}}{100}$$

$$\text{dirty\_bg\_threshold} = \text{MemTotal} \times \frac{\text{dirty\_background\_ratio}}{100}$$

Write behavior:

$$\text{action}(\text{dirty\_pages}) = \begin{cases}
\text{no writeback} & \text{if dirty} < \text{bg\_threshold} \\
\text{background flush} & \text{if bg\_threshold} \leq \text{dirty} < \text{threshold} \\
\text{synchronous writeback} & \text{if dirty} \geq \text{threshold}
\end{cases}$$

### Worked Examples

System with 32 GB RAM, default ratios:

| Parameter | Ratio | Threshold | Description |
|-----------|-------|-----------|-------------|
| dirty_background_ratio | 10% | 3.2 GB | Flusher daemon wakes up |
| dirty_ratio | 20% | 6.4 GB | Writers blocked until flush |

Write burst analysis: writing at 2 GB/s to a disk that flushes at 500 MB/s:

| Time | Written | Flushed | Dirty Pages | State |
|------|---------|---------|------------|-------|
| 0s | 0 GB | 0 GB | 0 GB | Normal |
| 1s | 2 GB | 0.5 GB | 1.5 GB | Normal |
| 2s | 4 GB | 1.0 GB | 3.0 GB | Normal |
| 2.1s | 4.2 GB | 1.05 GB | 3.15 GB | BG flush starts |
| 3s | 6 GB | 1.5 GB | 4.5 GB | BG flushing |
| 4.3s | 8.6 GB | 2.15 GB | 6.45 GB | SYNC: writers stall |

For databases, lower these thresholds:
- `dirty_background_ratio = 5` and `dirty_ratio = 10` ensures more consistent write latency.

## Prerequisites

operating-systems, computer-architecture, networking, virtual-memory, control-theory

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Read /proc/meminfo | O(1) | O(1) output |
| Read /proc/PID/status | O(1) | O(1) |
| Read /proc/PID/maps | O(VMAs) | O(VMAs) output |
| Read /proc/PID/smaps | O(pages) per VMA | O(VMAs) output |
| Write /proc/sys parameter | O(1) | O(1) |
| Enumerate /proc (all PIDs) | O(processes) | O(1) per entry |
| Read /proc/stat | O(CPUs) | O(CPUs) output |
| sysctl -a (all tunables) | O(tunables) ~1000 | O(tunables) |
