# The Mathematics of htop — Real-Time System Metrics & Visual Interpretation

> *htop is ps in motion — a real-time sampler that recomputes every metric each refresh cycle. Understanding the sampling math, the meter calculations, and the color coding transforms htop from eye candy into a diagnostic instrument.*

---

## 1. CPU Meter — Per-Core Utilization Breakdown

### CPU Time Decomposition

htop reads `/proc/stat` each refresh and computes deltas:

$$\Delta total = \Delta user + \Delta nice + \Delta system + \Delta idle + \Delta iowait + \Delta irq + \Delta softirq + \Delta steal$$

Each component as a percentage:

$$\%component = \frac{\Delta component}{\Delta total} \times 100$$

### Color Coding Formula

| Color | Component | Calculation |
|:---|:---|:---|
| Green | User (low priority normal) | $\Delta user / \Delta total$ |
| Blue | Low priority (nice) | $\Delta nice / \Delta total$ |
| Red | Kernel/System | $\Delta system / \Delta total$ |
| Cyan | IRQ time | $(\Delta irq + \Delta softirq) / \Delta total$ |
| Yellow/Orange | I/O wait | $\Delta iowait / \Delta total$ |
| Magenta | Steal | $\Delta steal / \Delta total$ |

### Bar Width Calculation

For a terminal $W$ columns wide, CPU bar width:

$$bar\_pixels = \lfloor (W - label\_width) \times \%util / 100 \rfloor$$

Each character in the bar represents:

$$per\_char = \frac{100\%}{W - label\_width}$$

For an 80-column terminal with 8-char label: each character $\approx 1.4\%$ CPU.

---

## 2. Memory and Swap Meters

### Memory Bar Composition

$$used = total - free - buffers - cache$$

$$\%used = \frac{used}{total} \times 100$$

$$\%buffers = \frac{buffers}{total} \times 100$$

$$\%cache = \frac{cache}{total} \times 100$$

| Color | Meaning | Concern Level |
|:---|:---|:---|
| Green | Used by processes | Normal |
| Blue | Buffers | Reclaimable |
| Yellow | Cache | Reclaimable |
| Red (if shown) | Swap used | Investigate if persistent |

### Available Memory (What Matters)

$$available = free + buffers + cache - min\_reclaim\_reserve$$

On a healthy system: $free$ is small, $available$ is large. htop 3.0+ shows this distinction.

### Swap Pressure

$$swap\_pressure = \frac{swap\_used}{swap\_total} \times 100$$

| swap_used % | Interpretation |
|:---:|:---|
| 0% | No memory pressure |
| 1-10% | Minor overflow, possibly transient |
| 10-50% | Significant pressure, monitor trends |
| > 50% | System under heavy memory pressure |

---

## 3. Refresh Rate and Sampling Theory

### Update Interval

htop defaults to **1.5 second** refresh (configurable 0.1s to 10s).

### Rate Accuracy vs Interval

CPU percentages are computed as:

$$\%CPU_{interval} = \frac{\Delta cputime}{\Delta walltime} \times 100$$

Shorter intervals → noisier measurements (higher variance):

$$\sigma(\%CPU) \propto \frac{1}{\sqrt{\Delta t}}$$

| Interval | Noise Level | Best For |
|:---:|:---|:---|
| 0.1s | Very noisy | Catching micro-bursts |
| 1.0s | Moderate | Interactive debugging |
| 1.5s | Low (default) | General monitoring |
| 5.0s | Very smooth | Long-term observation |

### Nyquist Constraint

To observe a phenomenon with period $T$:

$$refresh\_interval < \frac{T}{2}$$

A 2-second CPU spike requires refresh < 1 second to reliably capture.

---

## 4. Process Sorting — Comparison Functions

### Sort Key Mathematics

htop sorts by a single column using the standard comparison:

$$cmp(a, b) = key(a) - key(b)$$

### Sort Complexity

For $N$ processes:

$$T_{sort} = O(N \log N)$$

With typical 500 processes and 1.5s refresh: $500 \times \log_2(500) \times T_{compare} \approx 500 \times 9 \times 0.1\mu s = 0.45ms$ — negligible.

### Per-Interval CPU Sort vs Cumulative

| Sort | Formula | Shows |
|:---|:---|:---|
| %CPU (htop) | $\Delta cputime / \Delta walltime$ | Current activity |
| TIME+ (cumulative) | $\Sigma cputime$ since start | Total consumption |

A process using 100% CPU for 1 second then stopping:
- %CPU sort: appears at top briefly, then drops
- TIME+ sort: gradually rises, never drops

---

## 5. Thread View — Expanded Process Model

### Thread Display

With thread view enabled (H key), htop shows:

$$displayed = \sum_{p \in processes} nlwp(p)$$

Where $nlwp$ = number of lightweight processes (threads) per process.

### Per-Thread CPU

Each thread has its own CPU time:

$$\%CPU_{thread} = \frac{\Delta cputime_{thread}}{\Delta walltime} \times 100$$

$$\sum_{threads \in process} \%CPU_{thread} = \%CPU_{process}$$

### Tree View CPU Aggregation

In tree view (F5), parent shows aggregate CPU:

$$\%CPU_{parent} = \%CPU_{self} + \sum_{c \in children} \%CPU_c$$

This is a **post-order traversal** aggregation of the process tree.

---

## 6. Load Average Display

### The Three Numbers

$$load_{1} = EMA(\text{runnable + uninterruptible}, \tau = 60s)$$
$$load_{5} = EMA(\text{runnable + uninterruptible}, \tau = 300s)$$
$$load_{15} = EMA(\text{runnable + uninterruptible}, \tau = 900s)$$

### Interpretation Rules

$$load\_per\_cpu = \frac{load}{n_{cpus}}$$

| load_per_cpu | State |
|:---:|:---|
| < 0.7 | Healthy |
| 0.7 - 1.0 | Moderate |
| 1.0 - 2.0 | Overloaded (some queuing) |
| > 2.0 | Severely overloaded |

### Trend Analysis

| Pattern | Meaning |
|:---|:---|
| $load_1 > load_5 > load_{15}$ | Load increasing |
| $load_1 < load_5 < load_{15}$ | Load decreasing |
| $load_1 \approx load_5 \approx load_{15}$ | Stable load |
| $load_1 \gg load_{15}$ | Recent spike |

---

## 7. Filter and Search Complexity

### Incremental Search (F3)

htop uses **linear scan** with substring match:

$$T_{search} = O(N \times L_{avg})$$

Where $N$ = process count, $L_{avg}$ = average command length.

### Filter (F4)

Filter applies a predicate to every process each refresh:

$$displayed = \{p : filter\_match(p.name)\}$$

$$T_{filter\_per\_refresh} = O(N \times L_{avg})$$

For 1000 processes at 1.5s refresh: negligible overhead.

### User Filter (u)

$$displayed = \{p : uid(p) = target\_uid\}$$

$O(N)$ integer comparison — the fastest filter.

---

## 8. Resource Usage of htop Itself

### CPU Cost

htop reads `/proc` every refresh interval:

$$reads\_per\_refresh = N_{processes} \times files\_per\_process + system\_files$$

Where $files\_per\_process \approx 3$ (`/proc/[pid]/stat`, `statm`, `cmdline`) and $system\_files \approx 5$ (`/proc/stat`, `meminfo`, `loadavg`, etc.).

$$T_{refresh} = (3N + 5) \times T_{procfs\_read}$$

For 500 processes: $T_{refresh} = 1505 \times 10\mu s \approx 15ms$

$$\%CPU_{htop} = \frac{T_{refresh}}{interval} = \frac{15ms}{1500ms} = 1\%$$

### Memory Cost

$$RSS_{htop} \approx base + N \times per\_process\_entry$$

Where $base \approx 5 MB$ and $per\_process \approx 2 KB$.

For 1000 processes: $RSS \approx 5 + 2 = 7 MB$.

---

## 9. Summary of htop Mathematics

| Metric | Formula | Domain |
|:---|:---|:---|
| CPU % | $\Delta component / \Delta total \times 100$ | Differential |
| Memory used | $total - free - buff - cache$ | Partition |
| Sort complexity | $O(N \log N)$ | Algorithm |
| Load per CPU | $load / n_{cpus}$ | Normalization |
| Search | $O(N \times L)$ | Linear scan |
| Self-cost | $(3N + 5) \times T_{read} / interval$ | Overhead |
| Sampling noise | $\sigma \propto 1/\sqrt{\Delta t}$ | Statistics |

---

*htop is a real-time operating system dashboard — every number recomputed each cycle, every bar redrawn, every sort re-executed. It's the kernel's vital signs rendered as a TUI, at the cost of a few milliseconds and a megabyte of RAM.*
