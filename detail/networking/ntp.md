# The Mathematics of NTP — Clock Synchronization & Intersection Algorithms

> *Clock synchronization over a network is fundamentally an estimation problem: we can never know the true one-way delay, so NTP uses symmetric delay assumptions, statistical filtering, and Byzantine-tolerant intersection algorithms to bound the clock offset to within milliseconds of UTC.*

---

## 1. Clock Offset Estimation (Cristian's Algorithm)

### The Problem

A client sends a request to an NTP server and receives a reply. The network delays in each direction are unknown and potentially asymmetric. How can the client estimate its clock offset from the server?

### The Formula

Let the client send at local time $t_1$, server receive at $t_2$, server reply at $t_3$, client receive at $t_4$. The round-trip delay:

$$\delta = (t_4 - t_1) - (t_3 - t_2)$$

Assuming symmetric network delays ($d_1 \approx d_2 \approx \delta/2$), the estimated clock offset:

$$\theta = \frac{(t_2 - t_1) + (t_3 - t_4)}{2}$$

The maximum error bound (when delay is fully asymmetric):

$$|\theta_{\text{error}}| \leq \frac{\delta}{2}$$

### Worked Examples

**Example:** $t_1 = 100.000$, $t_2 = 100.012$, $t_3 = 100.013$, $t_4 = 100.025$:

$$\delta = (100.025 - 100.000) - (100.013 - 100.012) = 0.025 - 0.001 = 0.024 \text{ sec}$$

$$\theta = \frac{(100.012 - 100.000) + (100.013 - 100.025)}{2} = \frac{0.012 + (-0.012)}{2} = 0.000 \text{ sec}$$

Error bound: $|\theta_{\text{error}}| \leq 0.012$ sec. If the actual one-way delays are 18ms and 6ms (asymmetric):

$$\theta_{\text{true}} = 0.012 - 0.018 = -0.006 \text{ sec}$$

The estimate is 0.000, the truth is -0.006 — within the $\pm 0.012$ bound.

---

## 2. Marzullo's Intersection Algorithm (Byzantine Agreement)

### The Problem

NTP queries multiple servers, each returning an offset with an error bound. Some servers may be faulty ("falsetickers"). How does NTP find the largest interval consistent with a majority of sources?

### The Formula

Each source $i$ provides an interval $[\theta_i - \delta_i/2, \theta_i + \delta_i/2]$. Marzullo's algorithm finds the smallest interval that intersects with the intervals of at least $n - f$ sources, where $n$ is total sources and $f$ is the maximum tolerable falsetickers:

$$f < \frac{n}{2}$$

The algorithm: for each endpoint $e_k$ of all intervals, count how many intervals contain $e_k$. Find the narrowest range where the count $\geq n - f$.

For $n$ sources, the algorithm tolerates up to $f = \lfloor(n-1)/2\rfloor$ falsetickers.

### Worked Examples

**Example:** 5 NTP sources with offsets and error bounds (ms):

| Source | $\theta_i$ | $\delta_i/2$ | Interval |
|--------|-----------|-------------|----------|
| A | +2.0 | 5.0 | [-3.0, +7.0] |
| B | +3.5 | 4.0 | [-0.5, +7.5] |
| C | +50.0 | 3.0 | [+47.0, +53.0] |
| D | +2.8 | 6.0 | [-3.2, +8.8] |
| E | +1.5 | 5.5 | [-4.0, +7.0] |

With $n = 5$, $f = 2$, need agreement of $\geq 3$ sources. Source C is clearly a falseticker (offset 50ms). The intersection of A, B, D, E gives approximately $[-0.5, +7.0]$ ms, agreeing with 4 sources. NTP selects the system peer from these "truechimers."

---

## 3. Clock Drift and Frequency Discipline (Control Theory)

### The Problem

A computer's crystal oscillator drifts at rate $D$ ppm (parts per million). NTP must continuously adjust the clock frequency to compensate. How does the phase-locked loop (PLL) discipline the clock?

### The Formula

NTP uses a second-order PLL. The frequency adjustment at time $t$:

$$y(t) = y(t-1) + \frac{\theta(t)}{\tau^2}$$

The phase correction:

$$x(t) = x(t-1) + \frac{\theta(t)}{\tau} + y(t)$$

Where:
- $\theta(t)$ = measured offset at time $t$
- $\tau$ = poll interval (exponential: $2^n$ seconds, $n \in [4, 17]$, i.e., 16s to 36h)
- $y(t)$ = frequency estimate (drift rate)
- $x(t)$ = phase estimate

The Allan deviation measures oscillator stability:

$$\sigma_y(\tau) = \sqrt{\frac{1}{2(M-1)} \sum_{i=1}^{M-1} (\bar{y}_{i+1} - \bar{y}_i)^2}$$

### Worked Examples

**Example:** A clock drifts at 10 ppm. Over one day:

$$\text{drift} = 10 \times 10^{-6} \times 86400 = 0.864 \text{ sec/day}$$

With NTP polling every $\tau = 64$ seconds and measuring offset $\theta = 0.5$ ms:

$$\Delta y = \frac{0.0005}{64^2} = \frac{0.0005}{4096} = 1.22 \times 10^{-7} = 0.122 \text{ ppm adjustment}$$

Over 82 polls ($\approx 87$ min), NTP accumulates $82 \times 0.122 = 10$ ppm correction, fully compensating the drift. The drift value is stored in the drift file for fast convergence on restart.

---

## 4. Poll Interval Selection (Optimization)

### The Problem

Shorter poll intervals provide more frequent offset measurements but increase network load and are noisier. Longer intervals allow more drift between corrections. What is the optimal poll interval?

### The Formula

The error between corrections grows linearly with drift rate $D$ and poll interval $\tau$:

$$\epsilon_{\text{drift}} = D \cdot \tau$$

The measurement noise decreases with longer averaging:

$$\epsilon_{\text{noise}} = \frac{\sigma_n}{\sqrt{N}} \approx \frac{\sigma_n}{\sqrt{\tau / \tau_0}}$$

Total error (sum of drift and noise):

$$\epsilon(\tau) = D \cdot \tau + \frac{\sigma_n \cdot \sqrt{\tau_0}}{\sqrt{\tau}}$$

Minimize by taking the derivative and setting to zero:

$$\frac{d\epsilon}{d\tau} = D - \frac{\sigma_n \sqrt{\tau_0}}{2\tau^{3/2}} = 0$$

$$\tau_{\text{opt}} = \left(\frac{\sigma_n \sqrt{\tau_0}}{2D}\right)^{2/3}$$

### Worked Examples

**Example:** Drift $D = 5$ ppm $= 5 \times 10^{-6}$, measurement noise $\sigma_n = 2$ ms, base interval $\tau_0 = 1$ s:

$$\tau_{\text{opt}} = \left(\frac{0.002 \times 1}{2 \times 5 \times 10^{-6}}\right)^{2/3} = \left(200\right)^{2/3}$$

$$= e^{(2/3) \ln 200} = e^{(2/3)(5.298)} = e^{3.532} = 34.2 \text{ sec}$$

NTP rounds to the nearest power of 2: $\tau = 32$ seconds. For a stable oscillator with $D = 0.1$ ppm:

$$\tau_{\text{opt}} = \left(\frac{0.002}{2 \times 10^{-7}}\right)^{2/3} = (10000)^{2/3} = 464 \text{ sec}$$

NTP would select $\tau = 512$ seconds — explaining why well-synchronized hosts increase their poll interval.

---

## 5. Leap Second Smearing (Interpolation)

### The Problem

A positive leap second inserts 23:59:60 into the UTC timeline. Google's "leap smear" instead distributes this second over a window $W$. What is the smear function?

### The Formula

Google uses a cosine-based smear over 24 hours centered at the leap second ($t_L$):

$$\Delta(t) = \frac{1}{2}\left(1 - \cos\left(\pi \cdot \frac{t - t_L + W/2}{W}\right)\right)$$

For $t \in [t_L - W/2, t_L + W/2]$, and $\Delta(t) = 0$ before, $\Delta(t) = 1$ after.

The maximum rate of smearing (maximum frequency offset):

$$\left|\frac{d\Delta}{dt}\right|_{\max} = \frac{\pi}{2W}$$

### Worked Examples

**Example:** $W = 86400$ seconds (24 hours):

$$\left|\frac{d\Delta}{dt}\right|_{\max} = \frac{\pi}{2 \times 86400} = 1.818 \times 10^{-5} \approx 18.2 \text{ ppm}$$

At the midpoint (noon UTC on the leap second day), time is shifted by 0.5 seconds. The maximum frequency deviation is 18.2 ppm — well within the 500 ppm limit of the kernel's `adjtime()` system call.

At 6 hours before the leap second ($t = t_L - W/4$):

$$\Delta = \frac{1}{2}\left(1 - \cos\left(\pi \cdot \frac{W/4}{W}\right)\right) = \frac{1}{2}(1 - \cos(\pi/4)) = \frac{1}{2}(1 - 0.707) = 0.146 \text{ sec}$$

---

## 6. Stratum Decay and Network Depth (Graph Theory)

### The Problem

In a hierarchical NTP network, each hop from a stratum-0 source adds error. If the expected offset error at each stratum level is $\epsilon_s$, what is the total error at stratum $k$?

### The Formula

Errors accumulate through the stratum chain. If each level adds independent error with variance $\sigma_s^2$:

$$\sigma_k^2 = k \cdot \sigma_s^2$$

$$\sigma_k = \sigma_s \sqrt{k}$$

The maximum dispersion (NTP's root dispersion) accumulates as:

$$\Lambda_k = \sum_{i=1}^{k} \epsilon_i + \phi \cdot (t_{\text{now}} - t_{\text{last}})$$

Where $\phi$ is the maximum assumed drift rate (15 ppm per RFC 5905).

### Worked Examples

**Example:** Per-stratum error $\sigma_s = 0.5$ ms:

| Stratum | $\sigma_k$ | 95% CI |
|---------|-----------|--------|
| 1 | 0.50 ms | 1.0 ms |
| 2 | 0.71 ms | 1.4 ms |
| 3 | 0.87 ms | 1.7 ms |
| 4 | 1.00 ms | 2.0 ms |

Root dispersion at stratum 3, last update 64 sec ago:

$$\Lambda_3 = 3 \times 0.5 + 15 \times 10^{-6} \times 64 = 1.5 + 0.00096 \approx 1.501 \text{ ms}$$

This is why NTP limits stratum to 15: at stratum 15, $\sigma_{15} = 0.5\sqrt{15} = 1.94$ ms, and the accumulated dispersion makes time unreliable.

---

## Prerequisites

- Statistics (variance, standard deviation, Allan deviation)
- Control theory (phase-locked loop, feedback systems)
- Graph theory (hierarchical networks, tree depth)
- Optimization (minimizing error functions, derivatives)
- Trigonometry (cosine interpolation)
- Byzantine fault tolerance (agreement with faulty nodes)
