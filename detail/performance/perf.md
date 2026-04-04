# The Mathematics of perf — PMU Counters, Statistical Profiling & Cache Analysis

> *perf turns hardware Performance Monitoring Units into a statistical microscope. Every sample is a point estimate, every counter overflow a trigger, and the profile is a histogram governed by sampling theory and CPU microarchitecture.*

---

## 1. PMU Counter Overflow — Sampling Mechanism

### How perf Samples

perf programs a hardware counter to overflow after $N$ events, generating an interrupt that records the instruction pointer:

$$sample\_period = N \text{ events between samples}$$

$$sample\_rate = \frac{event\_rate}{sample\_period}$$

### Counter Overflow Math

The PMU counter counts from an initial value to overflow:

$$counter_{init} = 2^{width} - N$$

After $N$ events: counter wraps to 0, triggers NMI (Non-Maskable Interrupt).

On x86-64: counter width = 48 bits. Max period: $2^{48} - 1 \approx 2.8 \times 10^{14}$.

### Frequency Mode (-F)

`perf record -F 99` targets 99 samples/second:

$$sample\_period = \frac{event\_rate}{target\_frequency}$$

perf dynamically adjusts the period to maintain the target frequency as event rate changes.

**Example:** CPU at 3 GHz, profiling cycles at 99 Hz:

$$sample\_period = \frac{3 \times 10^9}{99} \approx 30,303,030 \text{ cycles between samples}$$

### Why 99 Hz (Not 100)?

Using 99 avoids **synchronization bias** with system timers that fire at 100 Hz (HZ=100). If perf and the timer fire at the same frequency, samples cluster at timer interrupt handlers.

$$P(bias) \propto \frac{1}{|f_{sample} - f_{timer}|} \text{ when frequencies are close}$$

---

## 2. Statistical Profiling — Confidence and Error

### Sampling Distribution

Each sample is a Bernoulli trial: did event $E$ occur at function $f$?

$$\hat{p}_f = \frac{samples\_in\_f}{total\_samples}$$

### Confidence Interval

$$CI_{95\%} = \hat{p}_f \pm 1.96 \times \sqrt{\frac{\hat{p}_f(1-\hat{p}_f)}{n}}$$

### Required Samples for Precision

To achieve margin of error $\epsilon$ at 95% confidence:

$$n \geq \frac{1.96^2 \times p(1-p)}{\epsilon^2}$$

| True % | $\epsilon = \pm 1\%$ | $\epsilon = \pm 5\%$ | $\epsilon = \pm 10\%$ |
|:---:|:---:|:---:|:---:|
| 50% | 9,604 | 384 | 96 |
| 10% | 3,457 | 138 | 35 |
| 1% | 381 | 15 | 4 |

**Key insight:** To detect a function consuming 1% of CPU time with $\pm 1\%$ precision, you need ~400 samples. At 99 Hz, that's ~4 seconds of profiling.

### Profile Accuracy Rule of Thumb

$$useful\_threshold \approx \frac{3}{\sqrt{n}}$$

With 10,000 samples: functions below $3/\sqrt{10000} = 3\%$ are noise.

---

## 3. Hardware Events — What the PMU Counts

### Event Types and Rates

| Event | Typical Rate (per second) | Meaning |
|:---|:---:|:---|
| cpu-cycles | $3 \times 10^9$ | Clock ticks |
| instructions | $1-5 \times 10^9$ | Instructions retired |
| cache-references | $10^7 - 10^9$ | L1/L2 cache accesses |
| cache-misses | $10^5 - 10^8$ | Last-level cache misses |
| branch-instructions | $10^8 - 10^9$ | Branch instructions |
| branch-misses | $10^6 - 10^8$ | Mispredicted branches |
| page-faults | $10^0 - 10^5$ | Virtual memory faults |

### Derived Metrics

$$IPC = \frac{instructions}{cycles} \text{ (Instructions Per Cycle)}$$

| IPC | Interpretation |
|:---:|:---|
| > 3.0 | Excellent (near superscalar limit) |
| 1.0-3.0 | Good |
| 0.5-1.0 | Memory-bound or branch-heavy |
| < 0.5 | Severely memory-bound |

### Cache Miss Ratio

$$miss\_ratio = \frac{cache\_misses}{cache\_references}$$

### Memory Wall — Cache Miss Cost

$$CPI = CPI_{base} + miss\_rate \times miss\_penalty$$

Where:
- $CPI_{base} \approx 0.25-1.0$ (no misses)
- $miss\_penalty = \frac{memory\_latency}{cycle\_time}$

At 3 GHz with 100 ns memory latency:

$$miss\_penalty = \frac{100ns}{0.333ns} = 300 \text{ cycles}$$

$$CPI = 0.5 + 0.05 \times 300 = 15.5$$

A 5% cache miss rate makes the CPU 31x slower than ideal.

---

## 4. perf stat — Counting Mode

### Counter Multiplexing

x86-64 has limited PMU counters (typically 4-8 programmable):

$$counters_{available} \approx 4-8$$

When monitoring more events than counters, perf multiplexes:

$$time\_per\_event = \frac{total\_time}{n_{events}} \times \frac{counters_{available}}{1}$$

$$actual\_count \approx measured\_count \times \frac{total\_time}{enabled\_time}$$

### Multiplexing Error

$$error = \sqrt{\frac{1 - f}{f \times N}} \text{ where } f = \frac{enabled\_time}{total\_time}$$

With 12 events on 4 counters: $f = 4/12 = 0.33$:

$$error \propto \sqrt{\frac{0.67}{0.33}} = 1.4\times \text{ of non-multiplexed}$$

**Recommendation:** Measure critical events (cycles, instructions) in a separate run without multiplexing.

---

## 5. perf record — Profiling Overhead

### Sampling Overhead

$$overhead = sample\_rate \times T_{sample\_processing}$$

Where $T_{sample\_processing}$:

| Component | Cost |
|:---|:---:|
| NMI handler | 0.5-2 us |
| Stack unwinding | 1-10 us |
| Buffer write | 0.1-0.5 us |
| **Total per sample** | **2-13 us** |

**Example:** At 99 Hz: $overhead = 99 \times 5\mu s = 0.5ms/s = 0.05\%$

At 4000 Hz: $overhead = 4000 \times 5\mu s = 20ms/s = 2\%$

### Data File Size

$$file\_size = total\_samples \times bytes\_per\_sample$$

$$bytes\_per\_sample \approx 40 + stack\_depth \times 8 \text{ (with -g for call graph)}$$

| Duration | Rate | Samples | File Size (no callgraph) | File Size (-g, depth 20) |
|:---:|:---:|:---:|:---:|:---:|
| 10 s | 99 Hz | 990 | 40 KB | 200 KB |
| 60 s | 99 Hz | 5,940 | 240 KB | 1.2 MB |
| 60 s | 4000 Hz | 240,000 | 9.6 MB | 48 MB |
| 3600 s | 99 Hz | 356,400 | 14 MB | 70 MB |

---

## 6. Flame Graphs — Stack Aggregation

### Stack Trace Aggregation

perf script outputs raw stack traces. Flame graph generation:

$$frequency(stack) = \frac{count(stack)}{total\_samples}$$

### Width Calculation

In a flame graph, each frame's width:

$$width(frame) = \frac{samples\_containing\_frame}{total\_samples}$$

$$self\_time(func) = width(func\_at\_top\_of\_stack)$$

$$inclusive\_time(func) = width(func\_anywhere\_in\_stack)$$

### Interpretation Rules

$$hot\_function \iff self\_time(func) > threshold$$

Threshold depends on sample count: $threshold \approx 3/\sqrt{n}$

---

## 7. Branch Prediction Analysis

### Branch Miss Rate

$$miss\_rate = \frac{branch\_misses}{branch\_instructions}$$

### Cost of Branch Misprediction

$$penalty \approx 15-25 \text{ cycles (pipeline flush)}$$

$$CPI_{branches} = miss\_rate \times penalty$$

### Worked Example

Program with 20% branch instructions, 5% miss rate, 20-cycle penalty:

$$extra\_CPI = 0.20 \times 0.05 \times 20 = 0.20$$

On a baseline CPI of 0.5: $CPI_{total} = 0.5 + 0.2 = 0.7$ (40% slowdown from mispredictions).

### perf Detection

```bash
perf stat -e branches,branch-misses ./program
```

$$\%mispredicted = \frac{branch\text{-}misses}{branches} \times 100$$

| Miss Rate | Assessment |
|:---:|:---|
| < 1% | Well-predicted |
| 1-5% | Normal |
| 5-10% | Investigate hot branches |
| > 10% | Significant performance drag |

---

## 8. Summary of perf Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Sample period | $event\_rate / target\_freq$ | Rate conversion |
| Confidence interval | $p \pm 1.96\sqrt{p(1-p)/n}$ | Statistics |
| IPC | $instructions / cycles$ | Efficiency metric |
| Cache miss cost | $CPI_{base} + miss\_rate \times penalty$ | Performance model |
| Multiplexing error | $\sqrt{(1-f)/(f \times N)}$ | Sampling error |
| Profiling overhead | $sample\_rate \times T_{sample}$ | Cost model |
| File size | $samples \times (40 + depth \times 8)$ | Storage |
| Branch cost | $miss\_rate \times pipeline\_penalty$ | Microarchitecture |

## Prerequisites

- sampling theory, statistics (confidence intervals), CPU microarchitecture, hardware performance counters, cache hierarchy

---

*perf is a statistical microscope powered by hardware counters. Every sample is a random snapshot, and the profile emerges from thousands of snapshots — like a pointillist painting where each dot is placed by the CPU itself.*
