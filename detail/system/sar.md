# The Mathematics of sar — System Activity Reporting & Historical Analysis

> *sar is the flight recorder of system performance. It samples counters at fixed intervals, stores them in binary format, and reconstructs rates, averages, and trends — making it the only standard tool that answers "what happened yesterday at 3 AM?"*

---

## 1. Counter-to-Rate Conversion

### The Fundamental Calculation

sar reads cumulative counters and converts to rates:

$$rate = \frac{counter(t_2) - counter(t_1)}{t_2 - t_1}$$

This is a **finite difference approximation** of the derivative:

$$rate \approx \frac{d(counter)}{dt}$$

### Counter Wrap-Around

32-bit counters wrap at $2^{32} = 4,294,967,296$. If $counter(t_2) < counter(t_1)$:

$$\Delta = (2^{32} - counter(t_1)) + counter(t_2)$$

At 10 Gbps, a 32-bit byte counter wraps every:

$$T_{wrap} = \frac{2^{32}}{1.25 \times 10^9} = 3.4 \text{ seconds}$$

This is why 64-bit counters (used in modern `/proc/net/dev`) are essential.

---

## 2. CPU Metrics — Detailed Decomposition

### sar -u (CPU Utilization)

$$\%usr + \%nice + \%sys + \%iowait + \%steal + \%idle = 100\%$$

### Per-CPU Analysis (sar -P ALL)

For $n$ CPUs, total system utilization:

$$\%util_{system} = \frac{\sum_{i=0}^{n-1} (100 - \%idle_i)}{n}$$

### CPU Imbalance Detection

Standard deviation of per-CPU utilization:

$$\sigma_{cpu} = \sqrt{\frac{\sum_{i=0}^{n-1} (\%util_i - \overline{\%util})^2}{n}}$$

| $\sigma_{cpu}$ | Interpretation |
|:---:|:---|
| < 5% | Well-balanced |
| 5-15% | Minor imbalance |
| 15-30% | Check IRQ/process affinity |
| > 30% | Severe imbalance, single-threaded bottleneck |

---

## 3. Memory Metrics — sar -r

### Key Columns

$$\%memused = \frac{kbmemused}{kbmemtotal} \times 100$$

$$\%commit = \frac{kbcommit}{kbmemtotal + kbswptotal} \times 100$$

### Memory Commit Ratio

$$commit\_ratio = \frac{committed\_AS}{physical\_RAM + swap}$$

| Commit Ratio | Meaning |
|:---:|:---|
| < 50% | Conservative, plenty of headroom |
| 50-80% | Normal for busy servers |
| 80-100% | Tight, monitor closely |
| > 100% | Overcommitted (relies on kernel overcommit) |

### Linux Overcommit Model

With `vm.overcommit_memory = 0` (heuristic):

$$allowed = total\_RAM \times overcommit\_ratio / 100 + swap$$

Default `overcommit_ratio = 50`:

$$allowed = RAM \times 0.5 + swap$$

---

## 4. Disk I/O — sar -d

### IOPS and Throughput

$$tps = \frac{\Delta (reads + writes)}{\Delta t}$$

$$rd\_sec/s = \frac{\Delta sectors\_read}{\Delta t} \times 512 \text{ bytes}$$

### Average Request Size

$$avgrq\text{-}sz = \frac{rd\_sec/s + wr\_sec/s}{tps} \text{ (in sectors)}$$

$$avgrq\_bytes = avgrq\text{-}sz \times 512$$

### Service Time and Queue

$$await = avgqu\text{-}sz / tps \times 1000 \text{ (ms, via Little's Law)}$$

$$svctm \approx \%util / tps \times 10 \text{ (ms, deprecated)}$$

---

## 5. Network — sar -n DEV

### Interface Throughput

$$bandwidth_{rx} = rxkB/s \times 8 / 1000 \text{ (Mbps)}$$

$$bandwidth_{tx} = txkB/s \times 8 / 1000 \text{ (Mbps)}$$

### Interface Utilization

$$\%util_{rx} = \frac{bandwidth_{rx}}{link\_speed} \times 100$$

**Example:** 10 Gbps link, `rxkB/s = 500000` (500 MB/s = 4 Gbps):

$$\%util_{rx} = \frac{4000}{10000} = 40\%$$

### Packet Rate Analysis

$$pps_{total} = rxpck/s + txpck/s$$

Each packet has overhead:

$$overhead\_per\_packet \approx 20B_{IP} + 20B_{TCP} + 14B_{Ethernet} + 4B_{FCS} = 58 \text{ bytes}$$

For small packets (e.g., 100 bytes payload):

$$\%overhead = \frac{58}{100 + 58} = 36.7\%$$

### Error Rate

$$error\_rate = \frac{rxerr/s + txerr/s}{rxpck/s + txpck/s}$$

| Error Rate | Severity |
|:---:|:---|
| 0% | Normal |
| < 0.01% | Acceptable |
| 0.01-0.1% | Investigate |
| > 0.1% | Hardware/driver issue |

---

## 6. Context Switches and Interrupts — sar -w

### Context Switch Rate

$$cswch/s = \frac{\Delta context\_switches}{\Delta t}$$

### Context Switch Classification

$$cswch_{total} = cswch_{voluntary} + cswch_{involuntary}$$

- **Voluntary:** Process yields (I/O wait, sleep, mutex)
- **Involuntary:** Preempted by scheduler (timeslice expired)

$$\frac{involuntary}{total} \text{ high} \implies \text{CPU contention}$$

### Interrupt Rate

$$intr/s = \frac{\Delta interrupts}{\Delta t}$$

At 10 Gbps with 64KB MTU:

$$pps = \frac{10 \times 10^9}{64 \times 1024 \times 8} = 19,073 \text{ packets/s per queue}$$

With 8 RX queues: $\approx 153K$ interrupts/second from NIC alone.

---

## 7. Data Collection Architecture

### sa File Format

sar stores binary data in `/var/log/sa/saDD` (DD = day of month):

$$file\_size \approx n_{intervals} \times n_{metrics} \times 8 \text{ bytes}$$

With 10-minute intervals (144/day) and ~200 metrics:

$$daily\_size = 144 \times 200 \times 8 = 230 KB$$

Monthly retention: $230KB \times 31 = 7.1 MB$

### Temporal Resolution

$$intervals\_per\_day = \frac{86400}{collection\_interval}$$

| Interval | Samples/Day | Storage/Day |
|:---:|:---:|:---:|
| 1 min | 1440 | 2.3 MB |
| 5 min | 288 | 460 KB |
| 10 min (default) | 144 | 230 KB |
| 60 min | 24 | 38 KB |

### Statistical Significance

For a metric with variance $\sigma^2$, the confidence interval over $n$ samples:

$$CI_{95\%} = \bar{x} \pm 1.96 \times \frac{\sigma}{\sqrt{n}}$$

At 10-minute intervals, 1 day = 144 samples:

$$CI_{95\%} = \bar{x} \pm 0.163 \sigma$$

---

## 8. Time-Based Analysis Patterns

### Day-Over-Day Comparison

$$\Delta metric = metric_{today}(t) - metric_{yesterday}(t)$$

$$\%change = \frac{\Delta metric}{metric_{yesterday}(t)} \times 100$$

### Peak Detection

$$peak = \max_{t \in [start, end]} metric(t)$$

$$peak\_ratio = \frac{peak}{mean}$$

| Peak Ratio | Pattern |
|:---:|:---|
| 1.0-1.5 | Steady load |
| 1.5-3.0 | Moderate spikes |
| 3.0-10.0 | Bursty workload |
| > 10.0 | Extreme bursts, capacity risk |

### Moving Average (Trend Extraction)

$$MA_k(t) = \frac{1}{k} \sum_{i=0}^{k-1} metric(t - i \times interval)$$

A 6-sample moving average at 10-minute intervals smooths over 1 hour, revealing hourly trends.

---

## 9. Summary of sar Mathematics

| Domain | Formula | Type |
|:---|:---|:---|
| Rate calculation | $\Delta counter / \Delta t$ | Finite difference |
| CPU imbalance | $\sigma(\%util_i)$ | Standard deviation |
| Commit ratio | $committed / (RAM + swap)$ | Ratio |
| Network utilization | $throughput / link\_speed$ | Percentage |
| Error rate | $errors / packets$ | Ratio |
| Storage sizing | $intervals \times metrics \times 8$ | Capacity |
| Confidence interval | $\bar{x} \pm 1.96\sigma/\sqrt{n}$ | Statistics |
| Peak ratio | $peak / mean$ | Burstiness measure |

---

*sar is the only tool that remembers yesterday. While vmstat and iostat show you the present, sar shows you the history — and history, with the right formulas, predicts the future.*
