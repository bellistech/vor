# The Mathematics of ECMP — Hash Distribution, Path Failure, and Load Balancing

> *Equal-Cost Multi-Path routing splits traffic across multiple paths of identical cost. The math involves hash function distribution, statistical load balancing quality, failure redistribution, and the polarization problem that occurs when multiple layers hash identically.*

---

## 1. Path Selection — Hash-Based Distribution

### The Problem

Given $K$ equal-cost paths and a stream of flows, assign each flow to a path deterministically (so packets within a flow stay in order).

### The Hash Function

$$\text{Path}(f) = H(f) \mod K$$

Where:
- $f$ = flow identifier (typically 5-tuple: src IP, dst IP, proto, src port, dst port)
- $H$ = hash function (CRC, XOR, Toeplitz, etc.)
- $K$ = number of equal-cost paths

### Distribution Quality

A perfect hash distributes $F$ flows across $K$ paths with:

$$E[\text{flows per path}] = \frac{F}{K}$$

$$\sigma = \sqrt{\frac{F(K-1)}{K^2}}$$

$$\text{Coefficient of variation} = \frac{\sigma}{\mu} = \sqrt{\frac{K-1}{F}}$$

### Worked Examples

| Flows ($F$) | Paths ($K$) | Expected/Path | Std Dev | CV (imbalance) |
|:---:|:---:|:---:|:---:|:---:|
| 100 | 2 | 50 | 5.0 | 10.0% |
| 100 | 4 | 25 | 4.3 | 17.3% |
| 1,000 | 4 | 250 | 13.7 | 5.5% |
| 10,000 | 4 | 2,500 | 43.3 | 1.7% |
| 10,000 | 8 | 1,250 | 33.1 | 2.6% |
| 100,000 | 16 | 6,250 | 60.6 | 1.0% |

**Key insight:** More flows → better balance. With $< 100$ flows, significant imbalance is expected.

---

## 2. Elephant Flow Problem — Bandwidth Skew

### The Problem

Flows are not equal in size. A few "elephant" flows carry most of the traffic, while many "mice" flows carry little.

### Zipf Distribution Model

Flow sizes often follow a Zipf-like distribution:

$$S_i = \frac{S_1}{i^\alpha}$$

Where $S_1$ = largest flow, $\alpha \approx 1$ (rank-size parameter).

### Bandwidth Imbalance

With $F$ flows on $K$ paths, if the top flow carries fraction $f_1$ of total bandwidth:

$$\text{Worst-case path load} \geq f_1 + \frac{1-f_1}{K}$$

| Top Flow Share | 4 Paths | 8 Paths | Ideal (1/K) |
|:---:|:---:|:---:|:---:|
| 10% | 32.5% | 21.3% | 25% / 12.5% |
| 25% | 43.75% | 35.7% | 25% / 12.5% |
| 50% | 62.5% | 56.3% | 25% / 12.5% |

A single flow consuming 50% of bandwidth makes ECMP nearly useless — one path gets 62.5% while others share the remaining 37.5%.

---

## 3. Path Failure — Redistribution Math

### The Problem

When one of $K$ paths fails, flows must be redistributed. How many flows are disrupted?

### With Modulo Hashing

$$\text{Disrupted flows} = F \times \frac{K-1}{K}$$

Because $H(f) \mod K \neq H(f) \mod (K-1)$ for most values.

| Original Paths | Remaining | Flows Disrupted | Disrupted % |
|:---:|:---:|:---:|:---:|
| 2 | 1 | $F/2$ | 50% |
| 4 | 3 | $3F/4$ | 75% |
| 8 | 7 | $7F/8$ | 87.5% |
| 16 | 15 | $15F/16$ | 93.75% |

### With Consistent Hashing

Consistent hashing (hash ring) minimizes disruption:

$$\text{Disrupted flows} = \frac{F}{K}$$

Only the flows on the failed path are redistributed.

| Original Paths | Remaining | Modulo Disruption | Consistent Disruption |
|:---:|:---:|:---:|:---:|
| 4 | 3 | 75% | 25% |
| 8 | 7 | 87.5% | 12.5% |
| 16 | 15 | 93.75% | 6.25% |

### Resilient Hashing

Modern implementations use resilient hash tables:

$$\text{Disrupted} = \frac{F}{K} \quad \text{(only affected bucket)}$$

Same as consistent hashing, but implemented as a flat lookup table with $B$ buckets ($B >> K$).

---

## 4. Hash Polarization — The Multi-Layer Problem

### The Problem

If two ECMP layers use the same hash function and fields:

$$H_1(f) \mod K_1 = H_2(f) \mod K_2$$

All flows that chose path $i$ at layer 1 will choose the same path at layer 2 — no additional load balancing occurs.

### Depolarization Techniques

| Technique | Method | Effectiveness |
|:---|:---|:---|
| Different hash seeds | $H(f, seed_1) \neq H(f, seed_2)$ | Good |
| Different hash fields | Layer 1: outer headers, Layer 2: inner | Good |
| Entropy labels (MPLS) | Add random label to hash input | Excellent |
| VXLAN source port | Inner flow hash → outer UDP src port | Excellent |

### Quantifying Polarization

With $K_1$ paths at layer 1 and $K_2$ at layer 2:

**Polarized:** Effective paths = $\max(K_1, K_2)$

**Depolarized:** Effective paths = $K_1 \times K_2$

| Layer 1 | Layer 2 | Polarized Paths | Depolarized Paths |
|:---:|:---:|:---:|:---:|
| 2 | 2 | 2 | 4 |
| 4 | 4 | 4 | 16 |
| 4 | 8 | 8 | 32 |
| 8 | 8 | 8 | 64 |

---

## 5. Weighted ECMP (UCMP)

### The Problem

Paths have equal routing cost but different capacities (e.g., 10G and 40G links).

### Weight Allocation

$$w_i = \frac{C_i}{\gcd(C_1, C_2, \ldots, C_K)}$$

| Path | Capacity | Weight | Traffic Share |
|:---|:---:|:---:|:---:|
| A | 10 Gbps | 1 | 20% |
| B | 40 Gbps | 4 | 80% |

### Implementation: Expanded Hash Table

Create a hash table with $\sum w_i$ entries:

$$B = \sum_{i=1}^{K} w_i$$

Path $i$ gets $w_i$ entries in the table.

| Weights | Table Size | Granularity |
|:---|:---:|:---:|
| 1:1:1:1 | 4 | 25% each |
| 1:4 | 5 | 20%/80% |
| 1:2:4 | 7 | 14%/29%/57% |
| 1:1:1:3 | 6 | 17%/17%/17%/50% |

---

## 6. ECMP in Clos Networks — Bisection Bandwidth

### Spine-Leaf Model

In a 2-tier Clos with $S$ spines and $L$ leaves:

$$K_{ECMP} = S \quad \text{(paths between any two leaves)}$$

### Bisection Bandwidth

$$BW_{bisection} = S \times L \times BW_{link}$$

### Oversubscription Ratio

$$R_{oversub} = \frac{L \times P_{south} \times BW_{south}}{L \times S \times BW_{north}} = \frac{P_{south} \times BW_{south}}{S \times BW_{north}}$$

For $P_{south} = 24$ (server-facing 10G ports) and $S = 4$ (40G uplinks):

$$R_{oversub} = \frac{24 \times 10}{4 \times 40} = \frac{240}{160} = 1.5:1$$

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $H(f) \mod K$ | Modular arithmetic | Path selection |
| $\sqrt{(K-1)/F}$ | Statistical (CV) | Balance quality |
| $F \times (K-1)/K$ | Fraction | Modulo rehash disruption |
| $F/K$ | Fraction | Consistent hash disruption |
| $K_1 \times K_2$ | Product | Depolarized effective paths |
| $C_i / \gcd(C)$ | GCD normalization | Weight calculation |
| $P_{south} \times BW_{south} / (S \times BW_{north})$ | Ratio | Oversubscription |

## Prerequisites

- hash functions, modular arithmetic, probability, load distribution

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Hash-based selection | O(1) | O(k) |
| Consistent hashing | O(log k) | O(k) |
| Path rebalance | O(k) | O(k) |

---

*ECMP is the mathematical backbone of every modern data center fabric — it's how a Clos network achieves non-blocking bandwidth without a single chassis switch. The quality of the hash function and the number of flows determine whether you get perfect load balancing or a hot spine.*
