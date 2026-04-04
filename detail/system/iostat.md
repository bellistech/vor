# The Mathematics of iostat — Disk I/O Metrics, Queue Theory & Utilization

> *iostat applies queuing theory to block devices. Every metric — utilization, await, avgqu-sz — maps directly to a formula from operations research, and together they diagnose I/O bottlenecks with mathematical precision.*

---

## 1. Utilization — The %util Column

### Definition

$$\%util = \frac{time\_device\_busy}{observation\_interval} \times 100$$

The kernel tracks this via the `io_ticks` counter in `/proc/diskstats`.

### Interpretation for Single vs Multi-Queue

| Device Type | 100% util | Meaning |
|:---|:---:|:---|
| Single-queue HDD | Saturated | No more IOPS available |
| Multi-queue NVMe | **Not saturated** | Multiple queues serving in parallel |

For NVMe with 64 hardware queues, `%util = 100%` means at least one I/O was in flight at all times — it says nothing about saturation.

### True Saturation Detection

$$saturated = (\%util > 95\%) \land (await \gg svctm) \land (avgqu\text{-}sz > queue\_depth)$$

---

## 2. IOPS and Throughput

### IOPS Calculation

$$IOPS = \frac{r/s + w/s}{1} = reads\_per\_sec + writes\_per\_sec$$

### Throughput

$$throughput_{read} = rKB/s \times 1024 \text{ bytes/s}$$
$$throughput_{write} = wKB/s \times 1024 \text{ bytes/s}$$

### Average I/O Size

$$avg\_io\_size = \frac{throughput}{IOPS}$$

| avg_io_size | Typical Workload |
|:---:|:---|
| 4 KB | Random reads (database index) |
| 8-16 KB | Database data pages |
| 64-256 KB | Sequential file reads |
| 1+ MB | Large file streaming |

---

## 3. Queuing Theory — Little's Law in iostat

### Little's Law

$$L = \lambda \times W$$

Where:
- $L$ = `avgqu-sz` (average queue length, including in-service requests)
- $\lambda$ = `r/s + w/s` (arrival rate = IOPS)
- $W$ = `await` (average time in system = queue wait + service)

### Verification

If `IOPS = 5000`, `await = 4 ms`:

$$avgqu\text{-}sz = 5000 \times 0.004 = 20$$

**Check:** If iostat reports `avgqu-sz = 20`, the numbers are consistent. If not, sampling artifacts.

### Components of await

$$await = queue\_wait + service\_time$$
$$await \approx avgqu\text{-}sz \times svctm$$

When the queue is empty: $await \approx svctm$. As queue builds: await grows linearly.

---

## 4. Service Time and the M/D/1 Queue Model

### For HDD (Single Server Queue)

A single spinning disk approximates an **M/D/1 queue** (Poisson arrivals, deterministic service):

$$W_{M/D/1} = \frac{\rho}{2\mu(1-\rho)} + \frac{1}{\mu}$$

Where:
- $\rho = \lambda / \mu$ = utilization (0 to 1)
- $\lambda$ = arrival rate (IOPS)
- $\mu$ = max service rate (max IOPS)
- $1/\mu$ = average service time

### Worked Example

HDD with max 150 IOPS ($\mu = 150$), current load 120 IOPS ($\lambda = 120$):

$$\rho = 120/150 = 0.80$$

$$W = \frac{0.80}{2 \times 150 \times 0.20} + \frac{1}{150} = \frac{0.80}{60} + 0.00667 = 0.01333 + 0.00667 = 20ms$$

$$L = \lambda \times W = 120 \times 0.020 = 2.4 \text{ requests in system}$$

### The Utilization Curve (Hockey Stick)

As utilization approaches 100%, latency explodes:

| $\rho$ (%util) | Queue Wait (ms) | Total await (ms) | Queue Length |
|:---:|:---:|:---:|:---:|
| 10% | 0.37 | 7.04 | 0.11 |
| 50% | 3.33 | 10.00 | 1.50 |
| 80% | 13.33 | 20.00 | 2.40 |
| 90% | 30.00 | 36.67 | 5.50 |
| 95% | 63.33 | 70.00 | 10.50 |
| 99% | 330.0 | 336.7 | 50.5 |

This is the **hockey stick curve** — latency doubles between 80% and 90% utilization, then explodes.

---

## 5. NVMe Multi-Queue Model

### Parallel Queue Model

NVMe devices have $Q$ hardware queues, each capable of $d$ outstanding commands:

$$max\_concurrent = Q \times d$$

Typical NVMe: $Q = 64$, $d = 64$: $max\_concurrent = 4096$.

### Effective Utilization

For multi-queue devices:

$$\rho_{effective} = \frac{avgqu\text{-}sz}{max\_concurrent}$$

**Example:** `avgqu-sz = 32`, max concurrent = 4096:

$$\rho_{eff} = 32/4096 = 0.78\% \text{ (not even close to saturated)}$$

Even at `%util = 100%`, the device may be lightly loaded.

### NVMe Throughput Model

$$max\_throughput = \min(IOPS_{max} \times io\_size,\ bandwidth_{max})$$

A typical NVMe: 500K IOPS at 4KB = 2 GB/s, but bandwidth cap at 3.5 GB/s.

$$crossover\_size = \frac{bandwidth_{max}}{IOPS_{max}} = \frac{3.5 \times 10^9}{500000} = 7 KB$$

Below 7 KB: IOPS-limited. Above 7 KB: bandwidth-limited.

---

## 6. Read/Write Mix Impact

### Weighted Average Service Time

$$svctm_{mixed} = f_r \times svctm_r + f_w \times svctm_w$$

Where $f_r + f_w = 1$ are the read and write fractions.

Writes are typically slower due to:
- Write amplification on SSDs: $WA = \frac{physical\_writes}{logical\_writes}$
- Journal writes on HDDs (write twice for journaling filesystems)

### Write Amplification Impact

$$effective\_write\_IOPS = \frac{raw\_IOPS}{WA}$$

| SSD State | Write Amplification | Effective Write IOPS (at 100K raw) |
|:---|:---:|:---:|
| Fresh/TRIM'd | 1.0-1.1x | 90,909-100,000 |
| 50% full | 1.5-2.0x | 50,000-66,667 |
| 90% full | 3.0-5.0x | 20,000-33,333 |
| 100% full | 5.0-10.0x | 10,000-20,000 |

---

## 7. Interval Selection and Rate Calculation

### iostat Counter Mechanics

iostat reads counters from `/proc/diskstats` and computes rates:

$$rate = \frac{counter(t_2) - counter(t_1)}{t_2 - t_1}$$

The first report is **cumulative since boot** — always discard it.

### Choosing the Interval

$$interval \leq \frac{1}{2 \times f_{phenomenon}}$$

For detecting I/O bursts (sub-second): use `iostat -x 1`.
For trending (minute-scale): use `iostat -x 60`.

### Combining with Other Tools

| iostat metric | Cross-reference | Tool |
|:---|:---|:---|
| High %util | Which process? | `iotop`, `pidstat -d` |
| High await | Which syscall? | `strace -e trace=read,write` |
| High avgqu-sz | Filesystem issue? | `df -i`, `filefrag` |
| Low throughput | Configuration? | `hdparm -t`, `fio` |

---

## 8. Summary of iostat Mathematics

| Metric | Formula | Theory |
|:---|:---|:---|
| Utilization | $busy\_time / interval$ | Time fraction |
| IOPS | $reads/s + writes/s$ | Throughput |
| Little's Law | $L = \lambda \times W$ | Queuing theory |
| M/D/1 wait | $\rho / (2\mu(1-\rho)) + 1/\mu$ | Single-server queue |
| NVMe utilization | $avgqu\text{-}sz / max\_concurrent$ | Multi-server queue |
| Hockey stick | $W \to \infty$ as $\rho \to 1$ | Queuing theory |
| Write amplification | $physical / logical$ | Storage physics |

## Prerequisites

- queuing theory, Little's Law, utilization metrics, block device I/O, probability distributions

---

*iostat is queuing theory applied to hardware. The hockey-stick curve — latency exploding as utilization approaches 100% — is not a bug; it's a mathematical certainty. Keep your disks below 70% utilization, or learn to love the curve.*
