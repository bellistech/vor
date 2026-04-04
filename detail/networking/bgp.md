# The Mathematics of BGP — College Algebra Perspective

> *Border Gateway Protocol doesn't use a single formula like physics. It uses a decision algorithm (ordered tie-breaking rules) combined with discrete math formulas for scalability, convergence, and dampening.*

---

## 1. The Peering Formula (Combinatorics / Scalability)

### The Problem

In internal BGP (iBGP), every router must peer with every other router — a **full mesh**. This is a classic **combinations** problem from discrete math.

### The Formula

$$P = \frac{N(N-1)}{2}$$

Where:
- $P$ = number of peering sessions required
- $N$ = number of routers

This is identical to the **handshake problem** — "if everyone in a room shakes hands with everyone else, how many handshakes occur?" It's the binomial coefficient $\binom{N}{2}$.

### Worked Examples

| Routers ($N$) | Calculation | Peering Sessions ($P$) |
|:---:|:---|:---:|
| 5 | $\frac{5(4)}{2} = \frac{20}{2}$ | 10 |
| 10 | $\frac{10(9)}{2} = \frac{90}{2}$ | 45 |
| 25 | $\frac{25(24)}{2} = \frac{600}{2}$ | 300 |
| 50 | $\frac{50(49)}{2} = \frac{2450}{2}$ | 1,225 |
| 100 | $\frac{100(99)}{2} = \frac{9900}{2}$ | 4,950 |

### Growth Rate Analysis

This is a **quadratic function**. Expand it:

$$P = \frac{N^2 - N}{2} = \frac{1}{2}N^2 - \frac{1}{2}N$$

The dominant term is $\frac{1}{2}N^2$, so peering sessions grow as $O(N^2)$ — doubling routers roughly **quadruples** sessions.

**Proof by example:** 10 routers → 45 sessions. 20 routers → 190 sessions. $\frac{190}{45} \approx 4.2\times$

### The Inverse Problem

*"We can handle 500 peering sessions. How many routers can we support?"*

Set $P = 500$ and solve for $N$:

$$500 = \frac{N^2 - N}{2}$$

$$1000 = N^2 - N$$

$$N^2 - N - 1000 = 0$$

Apply the quadratic formula $N = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$ where $a=1, b=-1, c=-1000$:

$$N = \frac{1 \pm \sqrt{1 + 4000}}{2} = \frac{1 \pm \sqrt{4001}}{2} = \frac{1 \pm 63.25}{2}$$

Taking the positive root: $N = \frac{64.25}{2} = 32.13$

**Answer:** 32 routers (floor, since routers are discrete).
**Verify:** $\frac{32(31)}{2} = 496$ sessions (under 500)

### The Engineering Solution: Route Reflectors

A **Route Reflector** (RR) replaces full mesh with a hub-and-spoke model:

$$P_{RR} = N - 1$$

This is **linear** growth — massive improvement:

| Routers | Full Mesh $\frac{N(N-1)}{2}$ | Route Reflector $N-1$ | Savings |
|:---:|:---:|:---:|:---:|
| 10 | 45 | 9 | 80% |
| 50 | 1,225 | 49 | 96% |
| 100 | 4,950 | 99 | 98% |

With **redundant RRs** (typically 2): $P = 2(N-1) = 2N - 2$. Still linear.

---

## 2. The Convergence Formula

### The Problem

When a route fails, BGP routers must propagate the change. How long until the whole network agrees?

### The Formula

$$T_{convergence} \approx (\text{Max AS\_PATH} - \text{Min AS\_PATH}) \times \text{MRAI}$$

Where:
- $T_{convergence}$ = time to converge (seconds)
- Max/Min AS_PATH = longest/shortest path lengths (hop count) in the network
- MRAI = Minimum Route Advertisement Interval (default: **30 seconds** for eBGP)

### Worked Examples

**Example 1: Simple network**
- Shortest path: 2 AS hops
- Longest path: 5 AS hops
- MRAI: 30 seconds

$$T = (5 - 2) \times 30 = 90 \text{ seconds}$$

**Example 2: Large ISP transit**
- Shortest path: 1 AS hop (direct peer)
- Longest path: 8 AS hops (through multiple transit providers)
- MRAI: 30 seconds

$$T = (8 - 1) \times 30 = 210 \text{ seconds} = 3.5 \text{ minutes}$$

### Why This Matters

During convergence, packets can be **black-holed** (dropped) or **looped**. This formula explains why BGP is considered a "slow" protocol — it trades speed for stability.

### Path Exploration Worst Case

The theoretical worst case for convergence involves **path exploration**, where routers try every possible path before settling. For $N$ autonomous systems, the number of alternative paths can be:

$$\text{Updates} \leq N!$$

In practice this is bounded, but it's why BGP convergence is measured in **minutes**, not milliseconds (unlike OSPF/IS-IS which converge in sub-seconds).

---

## 3. The Dampening Formula (Exponential Decay)

### The Problem

A "flapping" route oscillates between up and down. BGP uses **route dampening** to suppress unstable routes using an exponential decay function.

### The Penalty Function

Each flap adds a penalty. The penalty decays over time:

$$\text{Penalty}(t) = P_0 \times 2^{-t / \tau}$$

Where:
- $P_0$ = penalty at $t = 0$ (initial penalty value)
- $t$ = elapsed time (minutes)
- $\tau$ = **half-life** (default: 15 minutes per RFC 2439)

This is the **exponential decay** function — same form as radioactive decay in physics.

### Key Thresholds

| Parameter | Default Value | Meaning |
|:---|:---:|:---|
| Penalty per flap | 1,000 | Added each time route toggles |
| Suppress threshold | 2,000 | Route is suppressed above this |
| Reuse threshold | 750 | Route is unsuppressed below this |
| Half-life ($\tau$) | 15 min | Time for penalty to halve |
| Max suppress time | 60 min | Ceiling regardless of penalty |

### Worked Example

A route flaps 3 times rapidly:

$$P_0 = 3 \times 1000 = 3000$$

**Q: When does penalty drop below the reuse threshold (750)?**

$$750 = 3000 \times 2^{-t/15}$$

Divide both sides by 3000:

$$\frac{750}{3000} = 2^{-t/15}$$

$$0.25 = 2^{-t/15}$$

Take $\log_2$ of both sides:

$$\log_2(0.25) = -\frac{t}{15}$$

$$-2 = -\frac{t}{15}$$

$$t = 30 \text{ minutes}$$

**The route is suppressed for 30 minutes after 3 flaps.**

### General Solution for Reuse Time

$$t_{reuse} = \tau \times \log_2\left(\frac{P_0}{\text{Reuse Threshold}}\right)$$

More flaps → higher $P_0$ → longer suppression, up to the max suppress time.

| Flaps | $P_0$ | $t_{reuse}$ | Actual (capped at 60 min) |
|:---:|:---:|:---:|:---:|
| 2 | 2,000 | $15 \times \log_2(2.67) = 21.1$ min | 21 min |
| 3 | 3,000 | $15 \times \log_2(4) = 30$ min | 30 min |
| 5 | 5,000 | $15 \times \log_2(6.67) = 40.7$ min | 41 min |
| 10 | 10,000 | $15 \times \log_2(13.33) = 55.5$ min | 56 min |
| 15 | 15,000 | $15 \times \log_2(20) = 64.9$ min | **60 min** (capped) |

---

## 4. Best Path Selection — Ordered Decision Algorithm

This isn't a formula — it's a **deterministic sequential algorithm**. BGP evaluates attributes in strict order, stopping at the first tiebreaker:

| Priority | Attribute | Rule | Algebra Analogy |
|:---:|:---|:---|:---|
| 1 | Weight | Highest wins | $\max(W)$ — Cisco proprietary |
| 2 | Local Preference | Highest wins | $\max(LP)$ |
| 3 | Locally Originated | Prefer self-originated | Boolean: originated = true |
| 4 | AS Path Length | Shortest wins | $\min(\|AS\_PATH\|)$ |
| 5 | Origin Type | IGP > EGP > Incomplete | Ordinal: $i < e < ?$ |
| 6 | MED | Lowest wins | $\min(MED)$ |
| 7 | eBGP over iBGP | External preferred | Boolean: external = true |
| 8 | IGP Metric | Lowest cost to next-hop | $\min(\text{IGP cost})$ |
| 9 | Router ID | Lowest wins | $\min(RID)$ |

### As a Piecewise Function

$$\text{Best}(R) = \begin{cases}
\arg\max_r W(r) & \text{if unique max Weight} \\
\arg\max_r LP(r) & \text{if Weight tied} \\
\arg\min_r |AS(r)| & \text{if LP tied} \\
\vdots & \\
\arg\min_r RID(r) & \text{if all else tied}
\end{cases}$$

---

## 5. CIDR Prefix Math (Subnetting with Exponents)

BGP routes are advertised as **CIDR prefixes**. The number of IP addresses in a prefix:

$$\text{Addresses} = 2^{(32 - \text{prefix length})}$$

This is an **exponential function** with base 2.

| Prefix | Calculation | Addresses |
|:---:|:---|:---:|
| /32 | $2^{0}$ | 1 |
| /24 | $2^{8}$ | 256 |
| /16 | $2^{16}$ | 65,536 |
| /8 | $2^{24}$ | 16,777,216 |

### Aggregation Savings

If you have 4 contiguous /24 prefixes, you can aggregate them into 1 prefix:

$$4 \times /24 = 1 \times /22$$

Why? $\log_2(4) = 2$, so $24 - 2 = 22$.

**General aggregation:** $2^k$ contiguous $/n$ prefixes aggregate to $1 \times /(n - k)$

The global BGP routing table currently holds **~1 million prefixes**. Every aggregation reduces table size, memory, and convergence time.

---

## 6. Summary of Functions by Type

| Formula | Math Type | College Algebra Topic |
|:---|:---|:---|
| $\frac{N(N-1)}{2}$ | Quadratic / Combinatorial | Polynomials, Combinations |
| $\Delta \times \text{MRAI}$ | Linear | Slope-intercept, Direct variation |
| $P_0 \times 2^{-t/\tau}$ | Exponential decay | Exponential functions, Logarithms |
| $2^{(32-n)}$ | Exponential | Exponents, Powers of 2 |
| Best Path | Piecewise / Algorithmic | Piecewise functions, Ordering |
| $N - 1$ (Route Reflector) | Linear | Linear vs. quadratic comparison |

## Prerequisites

- algebra, logarithms, exponential functions, combinatorics, quadratic equations

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Full mesh peering | O(n^2) | O(n^2) |
| Route reflector peering | O(n) | O(n) |
| Best path selection | O(k) per prefix | O(1) |
| Path exploration (worst) | O(n!) | O(n) |

---

*These aren't abstract exercises — every one of these calculations runs in production on the routers carrying the internet right now.*
