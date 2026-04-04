# The Mathematics of tc — Token Buckets, Queuing Theory & Delay Models

> *Traffic control is applied queueing theory: token buckets implement leaky-bucket rate limiting, HTB uses hierarchical borrowing with precise token accounting, and netem's delay distributions let you simulate any network path from fiber to satellite.*

---

## 1. Token Bucket Rate Limiting (TBF Model)

### The Problem

A Token Bucket Filter allows bursts up to a bucket size $B$ but enforces a long-term rate $R$. For a given traffic pattern, when does the bucket empty and packets start dropping?

### The Formula

The bucket holds $B$ tokens, refilled at rate $R$ tokens/sec. At time $t$, available tokens:

$$T(t) = \min\left(B, T(0) + R \cdot t - \sum_{i: t_i \leq t} s_i\right)$$

Where $s_i$ is the size of packet $i$ arriving at time $t_i$.

For a constant-rate source at rate $\lambda$:
- If $\lambda \leq R$: no drops (bucket never empties)
- If $\lambda > R$: bucket drains in time:

$$t_{\text{drain}} = \frac{B}{\lambda - R}$$

Maximum burst size before conformance:

$$S_{\text{burst}} = B + R \cdot t_{\text{burst}}$$

For an instantaneous burst ($t_{\text{burst}} \to 0$): $S_{\text{burst}} = B$.

### Worked Examples

**Example:** TBF with $R = 10$ Mbit/s, $B = 32$ KB = 262144 bits. Source sends at 100 Mbit/s burst:

$$t_{\text{drain}} = \frac{262144}{100 \times 10^6 - 10 \times 10^6} = \frac{262144}{90 \times 10^6} = 2.91 \text{ ms}$$

Burst data admitted: $B = 32$ KB. After 2.91 ms, traffic is rate-limited to 10 Mbit/s.

If the source stops for 5 ms after draining, tokens accumulated:

$$T_{\text{refill}} = 10 \times 10^6 \times 0.005 = 50000 \text{ bits} = 6.1 \text{ KB}$$

The bucket is 19% refilled, allowing a small burst before rate-limiting resumes.

---

## 2. HTB Hierarchical Borrowing (Graph Algorithm)

### The Problem

In HTB, child classes can borrow unused bandwidth from siblings via their parent. The borrowing follows a tree walk. For a tree with depth $d$ and $n$ classes, what is the bandwidth allocation when some classes are idle?

### The Formula

Each class $i$ has:
- $r_i$ = guaranteed rate
- $c_i$ = ceil (maximum with borrowing)
- $p_i$ = priority

The effective rate of class $i$ when a sibling $j$ is idle:

$$R_i^{\text{eff}} = r_i + \min\left(c_i - r_i, \frac{r_j \cdot r_i}{\sum_{k \in \text{active siblings}} r_k}\right)$$

In general, spare bandwidth $\Delta = r_j$ is distributed to active siblings proportional to their guaranteed rates:

$$R_i^{\text{eff}} = r_i + \Delta \cdot \frac{r_i}{\sum_{k \neq j} r_k}$$

Bounded by $c_i$:

$$R_i^{\text{eff}} = \min\left(c_i, r_i + \Delta \cdot \frac{r_i}{\sum_{k \neq j} r_k}\right)$$

### Worked Examples

**Example:** Parent with 100 Mbit, three children:

| Class | Rate | Ceil |
|-------|------|------|
| A | 50 Mbit | 100 Mbit |
| B | 30 Mbit | 80 Mbit |
| C | 20 Mbit | 50 Mbit |

If C is idle ($\Delta = 20$ Mbit):

$$R_A = \min\left(100, 50 + 20 \times \frac{50}{80}\right) = \min(100, 62.5) = 62.5 \text{ Mbit}$$

$$R_B = \min\left(80, 30 + 20 \times \frac{30}{80}\right) = \min(80, 37.5) = 37.5 \text{ Mbit}$$

Total: 62.5 + 37.5 = 100 Mbit. Bandwidth is fully utilized.

If both B and C are idle ($\Delta = 50$ Mbit):

$$R_A = \min(100, 50 + 50) = 100 \text{ Mbit}$$

Class A expands to use the full link.

---

## 3. CoDel Target Delay (Control Theory)

### The Problem

CoDel (Controlled Delay) drops packets when the minimum sojourn time in the queue exceeds a target for an interval. How does the dropping function adapt?

### The Formula

CoDel uses an inverse-square-root drop schedule. After entering the dropping state, the $n$-th drop occurs at:

$$t_n = t_{\text{first}} + \text{interval} \cdot \frac{1}{\sqrt{n}}$$

The inter-drop interval decreases:

$$\Delta t_n = t_{n+1} - t_n = \text{interval} \cdot \left(\frac{1}{\sqrt{n+1}} - \frac{1}{\sqrt{n}}\right)$$

For large $n$:

$$\Delta t_n \approx \text{interval} \cdot \frac{-1}{2n^{3/2}}$$

The dropping rate at step $n$:

$$\lambda_{\text{drop}}(n) = \frac{1}{\Delta t_n} \approx \frac{2n^{3/2}}{\text{interval}}$$

### Worked Examples

**Example:** Default interval = 100 ms. Drop schedule:

| Drop $n$ | Time (ms) | Inter-drop (ms) |
|----------|-----------|-----------------|
| 1 | 100.0 | — |
| 2 | 170.7 | 70.7 |
| 3 | 157.7 | 57.7 |
| 4 | 150.0 | 50.0 |
| 5 | 144.7 | 44.7 |
| 10 | 131.6 | 31.6 |
| 20 | 122.4 | 22.4 |

CoDel starts slow (one drop per 70ms) and accelerates. After 20 drops, it is dropping every 22ms — aggressive enough to signal congestion to TCP senders.

---

## 4. netem Delay Distribution (Statistical Modeling)

### The Problem

Network delay is not uniform. netem supports several distributions. How do they model real network behavior?

### The Formula

**Normal distribution:**

$$f(d) = \frac{1}{\sigma\sqrt{2\pi}} e^{-\frac{(d-\mu)^2}{2\sigma^2}}$$

**Pareto distribution** (heavy-tailed, models internet delay):

$$f(d) = \frac{\alpha \cdot d_{\min}^\alpha}{d^{\alpha+1}}, \quad d \geq d_{\min}$$

Mean: $E[d] = \frac{\alpha \cdot d_{\min}}{\alpha - 1}$ for $\alpha > 1$

**Pareto-normal** (netem's paretonormal): mixture model combining normal body with Pareto tail.

With correlation $\rho$, successive delays are:

$$d_n = \rho \cdot d_{n-1} + (1-\rho) \cdot X_n$$

Where $X_n$ is drawn from the base distribution.

### Worked Examples

**Example:** `netem delay 100ms 30ms distribution normal`:

$$\mu = 100 \text{ ms}, \quad \sigma = 30 \text{ ms}$$

$$P(d > 160) = P\left(Z > \frac{160-100}{30}\right) = P(Z > 2) = 0.0228$$

About 2.3% of packets experience > 160ms delay.

With `distribution pareto` ($\alpha = 3$, $d_{\min}$ scaled from jitter):

$$P(d > 2\mu) = \left(\frac{d_{\min}}{2\mu}\right)^3$$

The Pareto tail means rare but extreme delays — realistic for internet paths.

---

## 5. Fair Queuing Hash Collision (Birthday Problem)

### The Problem

fq_codel hashes flows into $N$ buckets. With $F$ active flows, what is the probability that two flows share a bucket, causing unfairness?

### The Formula

This is the birthday problem:

$$P(\text{collision}) = 1 - \prod_{i=0}^{F-1} \frac{N-i}{N} \approx 1 - e^{-\frac{F(F-1)}{2N}}$$

Expected flows before first collision:

$$E[F_{\text{first}}] \approx \sqrt{\frac{\pi N}{2}}$$

### Worked Examples

**Example:** fq_codel with $N = 1024$ flows (default):

$$E[F_{\text{first}}] \approx \sqrt{\frac{\pi \times 1024}{2}} = \sqrt{1608} \approx 40$$

With 40 flows, there is a ~50% chance of at least one collision. With 100 flows:

$$P(\text{collision}) \approx 1 - e^{-\frac{100 \times 99}{2048}} = 1 - e^{-4.83} = 0.992$$

99.2% chance of collision. This is why fq_codel uses 1024 buckets by default — enough for typical internet traffic but imperfect for high-flow-count servers.

---

## 6. Ingress Policing Loss Rate (Token Bucket)

### The Problem

Ingress policing uses a token bucket to drop excess traffic. For bursty traffic with a Pareto-distributed burst size, what is the expected drop rate?

### The Formula

With arrival rate $\lambda$, police rate $R$, and bucket $B$, the long-term drop rate:

$$P(\text{drop}) = \max\left(0, 1 - \frac{R}{\lambda}\right)$$

For bursty traffic with burst size $S \sim \text{Pareto}(\alpha, S_{\min})$, a burst exceeding the bucket:

$$P(S > B) = \left(\frac{S_{\min}}{B}\right)^\alpha$$

Fraction of burst data dropped when $S > B$:

$$f_{\text{drop}}(S) = \frac{S - B}{S}, \quad S > B$$

Expected drop fraction:

$$E[f_{\text{drop}}] = \int_B^\infty \frac{s - B}{s} \cdot \frac{\alpha S_{\min}^\alpha}{s^{\alpha+1}} ds$$

### Worked Examples

**Example:** Police at $R = 100$ Mbit, $B = 256$ KB. Bursts $\sim \text{Pareto}(\alpha=2, S_{\min}=10$ KB):

$$P(S > 256) = \left(\frac{10}{256}\right)^2 = 0.00153$$

Only 0.15% of bursts exceed the bucket. For those that do (e.g., $S = 512$ KB):

$$f_{\text{drop}} = \frac{512 - 256}{512} = 50\%$$

Half the burst data is dropped. Setting $B$ to at least the 99th percentile burst size minimizes legitimate drops.

---

## Prerequisites

- Queueing theory (token bucket, leaky bucket, FIFO, WFQ)
- Control theory (feedback loops, adaptive algorithms)
- Probability distributions (normal, Pareto, exponential)
- Graph algorithms (tree traversal, hierarchical scheduling)
- Statistics (birthday problem, correlation, heavy-tailed distributions)
