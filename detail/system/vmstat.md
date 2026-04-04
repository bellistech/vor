# The Mathematics of vmstat — Virtual Memory Statistics & System Health Metrics

> *vmstat is a sampling instrument. Every column is a rate or a count, and understanding the relationships between them — memory pressure driving swap, CPU wait driving I/O — reveals the bottleneck equation of your system.*

---

## 1. Memory Columns — The Balance Equation

### The Fundamental Memory Identity

At any instant, physical RAM is partitioned:

$$RAM = free + buff + cache + used_{processes} + used_{kernel}$$

vmstat reports (in KB): `free`, `buff` (block device buffers), `cache` (page cache + reclaimable slab).

### Available vs Free

Free memory is misleading. **Available** memory (what can be allocated without swapping):

$$available \approx free + cache + buff - min\_watermark - unreclaimable$$

On a healthy system: $free$ can be near zero while $available$ is gigabytes — this is **normal**. The kernel uses free RAM as cache.

### Memory Pressure Detection

| Condition | vmstat Signature | Meaning |
|:---|:---|:---|
| Healthy | $free$ low, $cache$ high, $si=so=0$ | Cache is working |
| Moderate pressure | $cache$ shrinking, $si > 0$ | Reclaiming cache, some swap-in |
| Severe pressure | $so > 0$, $wa > 10\%$ | Actively swapping out |
| Thrashing | $si > 0 \land so > 0$, $wa > 50\%$ | Swapping in and out simultaneously |

### Thrashing Detection Formula

$$thrashing = (si + so > 0) \land (cache \text{ decreasing}) \land (wa > 20\%)$$

Quantitatively, if swap I/O exceeds useful work:

$$\frac{T_{swap\_IO}}{T_{total}} > 0.5 \implies thrashing$$

---

## 2. Swap Columns — si/so as Flow Rates

### Swap In/Out Rates

- `si` = pages swapped **in** from disk (KB/s)
- `so` = pages swapped **out** to disk (KB/s)

### Swap Throughput Impact

$$T_{swap\_delay} = \frac{pages\_swapped \times page\_size}{disk\_bandwidth}$$

| Storage | Bandwidth | 1 GB swap time | Latency per page (4 KB) |
|:---|:---:|:---:|:---:|
| HDD | 100 MB/s | 10 s | 40 us + 8 ms seek |
| SATA SSD | 500 MB/s | 2 s | 8 us |
| NVMe SSD | 3 GB/s | 0.33 s | 1.3 us |

### Working Set Size Estimation

If `so` is consistently > 0, the working set exceeds physical RAM:

$$working\_set \approx RAM + \int_0^T so(t)\ dt$$

Where the integral represents total data swapped out over observation period $T$.

---

## 3. CPU Columns — Utilization Decomposition

### CPU Time Categories

vmstat reports CPU time as percentages (summing to ~100%):

$$us + sy + id + wa + st = 100\%$$

| Column | Meaning | Counts As |
|:---|:---|:---|
| `us` | User space | Productive work |
| `sy` | Kernel/system | Overhead |
| `id` | Idle | Available capacity |
| `wa` | I/O wait | Blocked on disk/network |
| `st` | Steal time | Lost to hypervisor |

### Utilization Formula

$$utilization = us + sy = 100 - id - wa - st$$

$$effective\_capacity = 1 - \frac{wa + st}{100}$$

### CPU Saturation Detection

| Metric | Threshold | Meaning |
|:---|:---:|:---|
| $us + sy > 90\%$ | Saturated | CPU-bound |
| $wa > 20\%$ | I/O bottleneck | Disk/network-bound |
| $st > 5\%$ | Resource contention | Noisy neighbor / undersized VM |
| $r > 2 \times n_{cpu}$ | Overloaded | Run queue depth too high |

### Run Queue (r) — Little's Law Application

The `r` column shows runnable processes. By **Little's Law**:

$$\bar{L} = \lambda \times \bar{W}$$

Where $\bar{L}$ = average queue length (r), $\lambda$ = arrival rate, $\bar{W}$ = average wait time.

If $r$ is consistently $> n_{cpus}$, processes are waiting:

$$wait\_time \approx \frac{r - n_{cpus}}{n_{cpus}} \times avg\_timeslice$$

**Example:** 4 CPUs, `r` = 12, timeslice ~4 ms:

$$wait \approx \frac{12 - 4}{4} \times 4ms = 8ms \text{ average queue wait}$$

---

## 4. I/O Columns — bi/bo as Throughput

### Block I/O Rates

- `bi` = blocks read from disk (blocks/s, 1 block = 1 KB)
- `bo` = blocks written to disk (blocks/s)

### I/O Bandwidth

$$bandwidth_{read} = bi \times 1024 \text{ bytes/s}$$
$$bandwidth_{write} = bo \times 1024 \text{ bytes/s}$$
$$bandwidth_{total} = (bi + bo) \times 1024 \text{ bytes/s}$$

### Disk Utilization Estimation

If you know your disk's maximum throughput:

$$disk\_utilization \approx \frac{(bi + bo) \times 1024}{disk\_max\_throughput}$$

**Example:** `bi=50000`, `bo=20000`, NVMe max 3 GB/s:

$$utilization = \frac{70000 \times 1024}{3 \times 10^9} = \frac{71.7 MB/s}{3000 MB/s} = 2.4\%$$

### Correlation: bo and dirty pages

Write-back rate should match dirty page generation rate at steady state:

$$bo \approx \frac{dirty\_pages\_generated}{writeback\_interval}$$

If `bo` spikes when dirty pages hit `dirty_background_ratio`, that's the kernel flusher activating.

---

## 5. Sampling Theory — Interval Selection

### Nyquist-Shannon for System Monitoring

To capture a phenomenon with period $T_p$, sample at interval:

$$T_{sample} \leq \frac{T_p}{2}$$

| Phenomenon | Period | Min Sample Rate |
|:---|:---:|:---:|
| CPU bursts | 100 ms | 50 ms |
| I/O spikes | 1 s | 500 ms |
| Memory trends | 10 s | 5 s |
| Swap patterns | 30 s | 15 s |

### vmstat Interval Recommendations

```bash
vmstat 1 60    # 1-second samples for 60 seconds (fine-grained)
vmstat 5 720   # 5-second samples for 1 hour (monitoring)
vmstat 30      # 30-second continuous (long-term trend)
```

### Averaging Over Intervals

vmstat reports **rates** averaged over the interval:

$$reported\_rate = \frac{\Delta counter}{interval}$$

The first line of vmstat output is always the **average since boot** — discard it.

### Statistical Significance

For $n$ samples with mean $\bar{x}$ and standard deviation $s$:

$$CI_{95\%} = \bar{x} \pm 1.96 \times \frac{s}{\sqrt{n}}$$

To get CPU utilization within $\pm 2\%$ at 95% confidence with $s = 10\%$:

$$n \geq \left(\frac{1.96 \times 10}{2}\right)^2 = 96 \text{ samples}$$

At 1-second intervals: 96 seconds of data needed.

---

## 6. System Bottleneck Identification

### Decision Tree

$$bottleneck = \begin{cases}
\text{CPU-bound} & \text{if } us + sy > 90\% \land wa < 5\% \\
\text{I/O-bound} & \text{if } wa > 20\% \land bi + bo \text{ high} \\
\text{Memory-bound} & \text{if } so > 0 \land free \to 0 \\
\text{Balanced} & \text{otherwise}
\end{cases}$$

### Compound Bottleneck Example

vmstat output: `r=8`, `us=30`, `sy=10`, `wa=45`, `so=5000`:

1. $wa = 45\% > 20\%$ → I/O bound
2. $so = 5000$ KB/s → Swapping (memory pressure)
3. $r = 8$ on 4 CPUs → Processes queuing

**Diagnosis:** Memory pressure → swapping → I/O wait → CPU starvation. Root cause: insufficient RAM, not disk speed.

---

## 7. vmstat -s — Cumulative Counters

### Counter-Based Analysis

`vmstat -s` shows cumulative totals since boot. Rate calculation:

$$rate = \frac{counter_{t2} - counter_{t1}}{t2 - t1}$$

Key counters:
- `pages paged in/out` — total page I/O
- `pages swapped in/out` — swap-specific I/O
- `interrupts` — total hardware + software interrupts
- `context switches` — total context switches

### Context Switch Rate

$$cs\_rate = \frac{\Delta cs}{\Delta t}$$

| cs/sec | Interpretation |
|:---:|:---|
| < 5,000 | Light load |
| 5,000-50,000 | Normal server |
| 50,000-500,000 | High throughput |
| > 500,000 | Potential overhead |

Context switch overhead: $cs\_rate \times T_{switch} \approx cs\_rate \times 5\mu s$

At 100,000 cs/sec: $overhead = 500ms/s = 50\%$ of one CPU core.

---

## 8. Summary of vmstat Mathematics

| Metric | Formula | What It Reveals |
|:---|:---|:---|
| Memory identity | $RAM = free + buff + cache + used$ | Memory partitioning |
| Thrashing | $si > 0 \land so > 0 \land wa > 20\%$ | Working set > RAM |
| CPU utilization | $us + sy$ | Productive CPU usage |
| Queue wait | $(r - n_{cpu}) / n_{cpu} \times timeslice$ | Scheduling delay |
| I/O bandwidth | $(bi + bo) \times 1024$ | Disk throughput |
| Sampling | $T_{sample} \leq T_{phenomenon} / 2$ | Nyquist criterion |
| Confidence | $\bar{x} \pm 1.96 \times s / \sqrt{n}$ | Statistical validity |

## Prerequisites

- virtual memory, sampling theory, Nyquist criterion, statistics, memory partitioning

---

*vmstat is the vital signs monitor of your system. Like a doctor reading blood pressure and heart rate, you read memory pressure and CPU utilization — and the formulas tell you which organ is failing.*
