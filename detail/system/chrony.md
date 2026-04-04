# The Mathematics of Chrony — Clock Synchronization, Drift Correction & NTP Algorithms

> *Chrony is a statistical clock filter. It models your system clock as a linear function with drift and noise, estimates the true time using regression, and steers the clock with correction rates bounded by stability requirements.*

---

## 1. Clock Model — Drift and Offset

### The System Clock as a Linear Function

The system clock $C(t)$ relative to true time $t$:

$$C(t) = t + offset + drift \times (t - t_0) + noise(t)$$

Where:
- $offset$ = current time error (seconds)
- $drift$ = clock frequency error (ppm — parts per million)
- $t_0$ = reference time
- $noise(t)$ = random jitter

### Drift Units

$$1 \text{ ppm} = 1 \mu s / s = 86.4 ms / day$$

| Drift (ppm) | Error per Hour | Error per Day |
|:---:|:---:|:---:|
| 0.01 | 36 us | 0.86 ms |
| 0.1 | 360 us | 8.6 ms |
| 1.0 | 3.6 ms | 86.4 ms |
| 10.0 | 36 ms | 864 ms |
| 100.0 | 360 ms | 8.64 s |

Typical modern hardware: 1-50 ppm drift. Chrony corrects this to sub-millisecond accuracy.

### Maximum Unsynchronized Error

If NTP is lost for duration $T$:

$$max\_error = |drift| \times T$$

With 10 ppm drift, NTP down for 1 hour:

$$error = 10 \times 10^{-6} \times 3600 = 36 ms$$

---

## 2. NTP Measurement — Network Delay Estimation

### The Four Timestamps

Each NTP exchange produces 4 timestamps:

- $T_1$ = client send time
- $T_2$ = server receive time
- $T_3$ = server send time
- $T_4$ = client receive time

### Offset Calculation

$$offset = \frac{(T_2 - T_1) + (T_3 - T_4)}{2}$$

This formula assumes **symmetric network delay**: $d_{up} = d_{down}$.

### Round-Trip Delay

$$RTT = (T_4 - T_1) - (T_3 - T_2)$$

$$one\_way\_delay = \frac{RTT}{2}$$

### Asymmetry Error

If $d_{up} \neq d_{down}$:

$$error_{asymmetry} = \frac{d_{up} - d_{down}}{2}$$

| Network | Typical Asymmetry | Offset Error |
|:---|:---:|:---:|
| LAN (switched) | < 10 us | < 5 us |
| WAN (same continent) | 0.1-1 ms | 50-500 us |
| Satellite | 10-100 ms | 5-50 ms |
| Asymmetric DSL | 5-20 ms | 2.5-10 ms |

Satellite and DSL have inherently asymmetric paths — NTP accuracy is fundamentally limited.

---

## 3. Chrony's Clock Filter — Linear Regression

### Regression Model

Chrony uses **weighted linear regression** on recent offset measurements:

$$offset(t) = a + b \times (t - t_0) + \epsilon$$

Where:
- $a$ = estimated current offset
- $b$ = estimated drift rate (ppm)
- $\epsilon$ = residual (noise)

### Weighted Least Squares

$$\hat{b} = \frac{\sum w_i (t_i - \bar{t})(y_i - \bar{y})}{\sum w_i (t_i - \bar{t})^2}$$

$$\hat{a} = \bar{y} - \hat{b} \times \bar{t}$$

Weights $w_i$ decrease for older samples:

$$w_i = e^{-(t_{now} - t_i) / \tau}$$

Where $\tau$ is a time constant (chrony adapts this based on clock stability).

### Advantages Over ntpd

| Feature | ntpd | chronyd |
|:---|:---|:---|
| Filter | FLL/PLL (control loop) | Linear regression |
| Initial sync | Minutes | Seconds |
| Drift estimation | Slow convergence | Fast, statistical |
| Intermittent connectivity | Poor | Excellent |
| Sample selection | Clock filter, 8 samples | All samples, weighted |

---

## 4. Clock Correction — Slew vs Step

### Slew (Gradual Adjustment)

Chrony adjusts the clock frequency to gradually correct offset:

$$correction\_rate = \frac{offset}{correction\_time}$$

Linux `adjtime()` maximum slew: 500 ppm (0.5 ms/s).

$$T_{slew} = \frac{|offset|}{max\_slew\_rate} = \frac{|offset|}{500 \times 10^{-6}}$$

| Offset | Slew Duration |
|:---:|:---:|
| 1 ms | 2 seconds |
| 10 ms | 20 seconds |
| 100 ms | 200 seconds |
| 1 s | 2000 seconds (~33 min) |

### Step (Instant Jump)

Chrony steps the clock when offset exceeds a threshold (default: 1 second in first 3 corrections):

$$C_{new} = C_{old} + offset$$

**Risk:** A step can break:
- Timestamps in logs (non-monotonic)
- Certificate validity checks
- Database transaction ordering
- File modification times

### makestep Directive

`makestep 1.0 3` means: step if offset > 1 second, only during first 3 updates.

After initial sync, only slew corrections (monotonic clock preserved).

---

## 5. Source Selection — Falseticker Detection

### Intersection Algorithm

With multiple NTP sources, chrony must identify **falsetickers** (incorrect clocks):

Given $n$ sources with offset estimates $(\theta_i \pm \delta_i)$, find the largest subset where intervals overlap:

$$consensus = \{i : [\theta_i - \delta_i, \theta_i + \delta_i] \cap I_{majority} \neq \emptyset\}$$

### Byzantine Fault Tolerance

NTP can tolerate $f$ faulty sources out of $n$ total:

$$f < \frac{n}{2}$$

Minimum sources for reliability:
- 1 source: no fault detection
- 2 sources: can detect disagreement, can't determine which is wrong
- 3 sources: can tolerate 1 falseticker
- 4 sources: can tolerate 1 falseticker with confidence

**Recommendation:** 4+ sources for production systems.

### Source Quality Metrics

| Metric | chronyc Field | Meaning |
|:---|:---|:---|
| Stratum | St | Distance from reference clock |
| Polling interval | Poll | $2^{poll}$ seconds between queries |
| Reach | Reach | Octal bitmask of last 8 responses |
| Offset | Offset | Current estimated offset |
| Jitter | Jitter | Root-mean-square of offset residuals |

### Polling Interval Adaptation

$$poll\_interval = 2^{poll} \text{ seconds}$$

| Poll Value | Interval | Use Case |
|:---:|:---:|:---|
| 6 | 64 s | Initial synchronization |
| 8 | 256 s | LAN sources |
| 10 | 1024 s | Stable WAN sources |
| 12 | 4096 s | Very stable sources |

Chrony increases poll when clock is stable, decreases when drift changes.

---

## 6. Frequency Tracking — The driftfile

### Drift File Format

Chrony saves the estimated drift to `/var/lib/chrony/drift`:

$$drift\_value \text{ (ppm, double precision)}$$

### Cold Start vs Warm Start

| Start Type | Initial Offset Error | Time to Sub-ms |
|:---|:---:|:---:|
| Cold (no driftfile) | Up to seconds | 30-60 seconds |
| Warm (with driftfile) | < 10 ms | 5-15 seconds |

### Drift Stability

$$drift\_change = |drift(t) - drift(t - \Delta t)|$$

| Drift Change | Cause |
|:---:|:---|
| < 0.01 ppm/day | Normal crystal aging |
| 0.01-0.1 ppm/day | Temperature variation |
| 0.1-1 ppm/day | Significant temperature swing |
| > 1 ppm/day | Hardware problem or VM migration |

Temperature coefficient of quartz: $\approx 0.04 \text{ ppm}/^{\circ}C$ near the turnover point.

---

## 7. Accuracy Budget — Error Breakdown

### Total Error

$$error_{total} = \sqrt{error_{network}^2 + error_{oscillator}^2 + error_{asymmetry}^2 + error_{resolution}^2}$$

### Error Sources

| Source | Magnitude | Reducible? |
|:---|:---:|:---|
| Network jitter | 0.1-10 ms | Yes (more samples) |
| Network asymmetry | 0-50 ms | Partially (PTP, GPS) |
| Oscillator noise | 0.01-1 us | No (hardware limit) |
| Timestamping resolution | 1 us (software) | Yes (HW timestamping) |
| Interrupt latency | 1-100 us | Yes (kernel tuning) |

### Achievable Accuracy

| Setup | Typical Accuracy |
|:---|:---:|
| Internet NTP (3+ sources) | 1-10 ms |
| LAN NTP (dedicated server) | 100 us - 1 ms |
| PTP (hardware timestamping) | 10-100 ns |
| GPS/PPS (direct reference) | 1-10 us |

---

## 8. Summary of Chrony Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Clock model | $C(t) = t + offset + drift \times \Delta t$ | Linear model |
| NTP offset | $(T_2 - T_1 + T_3 - T_4) / 2$ | Mean of asymmetry |
| Slew duration | $\|offset\| / max\_slew\_rate$ | Linear correction |
| Drift (ppm) | $1\ ppm = 86.4\ ms/day$ | Unit conversion |
| Fault tolerance | $f < n/2$ sources | Byzantine consensus |
| Poll interval | $2^{poll}$ seconds | Exponential |
| Error budget | $\sqrt{\sum e_i^2}$ | Quadrature sum |

## Prerequisites

- linear regression, Kalman filtering, NTP protocol, clock drift, statistical estimation, network latency

---

*Chrony is a Kalman filter for your system clock — continuously estimating offset and drift from noisy network measurements, and steering the clock frequency to converge on true time. Its regression-based approach is why it syncs in seconds where ntpd takes minutes.*
