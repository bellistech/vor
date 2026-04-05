# The Mathematics of Consistent Hashing -- Ring Topology and Minimal Disruption

> *Consistent hashing maps both keys and nodes onto a circle, ensuring that when a node joins or leaves, only the keys in its arc migrate -- a property that transformed distributed caching from an O(n) reshuffling nightmare into an O(k/n) local adjustment.*

---

## 1. The Hash Ring as a Quotient Space (Topology)

### The Problem

Define the consistent hash ring mathematically and show how key assignment follows
from the ring structure.

### The Formula

Let $h: \text{Keys} \cup \text{Nodes} \to [0, 2^m)$ be a hash function mapping to
$m$-bit integers. Define the hash ring as the quotient space:

$$\mathcal{R} = \mathbb{Z}_{2^m} = [0, 2^m) \text{ with } 2^m \equiv 0$$

A key $k$ is assigned to the node $n$ such that $h(n)$ is the first node hash
encountered when walking clockwise from $h(k)$:

$$\text{node}(k) = \arg\min_{n \in N} \{(h(n) - h(k)) \mod 2^m\}$$

Equivalently, using a sorted array of node hashes $S = [s_0, s_1, \ldots, s_{|S|-1}]$:

$$\text{idx} = \min\{i \mid s_i \ge h(k)\}, \quad \text{with wrap: } \text{idx} = 0 \text{ if no such } i$$

### Worked Examples

Ring with $2^m = 16$, nodes at positions $\{2, 7, 12\}$:

| Key hash | Walk clockwise to | Assigned node |
|:---:|:---:|:---:|
| 0 | 2 | Node at 2 |
| 5 | 7 | Node at 7 |
| 10 | 12 | Node at 12 |
| 14 | 2 (wraps) | Node at 2 |

---

## 2. Minimal Disruption Property (Distributed Systems)

### The Problem

Prove that adding or removing a node remaps at most $O(k/n)$ keys on average.

### The Formula

With $n$ nodes and $k$ keys uniformly distributed on $\mathcal{R}$, each node owns an
arc of expected length $2^m / n$. When a new node $n_{+}$ is added at position $p$:

- Only keys in the arc between $n_{+}$'s predecessor and $p$ are remapped (they were
  assigned to $n_{+}$'s successor, now reassigned to $n_{+}$).
- Expected keys remapped: $k / n$ (the arc covers $1/n$ of the ring on average).

**Contrast with modular hashing** ($\text{node} = h(k) \mod n$): changing $n$ to $n+1$
remaps approximately $(n-1)/n \cdot k$ keys -- nearly all of them.

$$\text{Consistent: } \Delta_{\text{keys}} = O(k/n) \quad\text{vs.}\quad \text{Modular: } \Delta_{\text{keys}} = O(k)$$

### Worked Examples

100 nodes, 10,000 keys. Add 1 node:
- Consistent hashing: $\approx 10{,}000/100 = 100$ keys remap (1%).
- Modular hashing: $\approx 10{,}000 \cdot 99/100 = 9{,}900$ keys remap (99%).

---

## 3. Virtual Nodes and Load Variance (Probability Theory)

### The Problem

Without virtual nodes, the load variance across nodes is high. Quantify the improvement
from virtual nodes.

### The Formula

With $n$ real nodes and no virtual nodes, each node owns one arc. The arc lengths follow
a Dirichlet distribution. The expected load of each node is $1/n$, but the variance is:

$$\text{Var}(\text{load}_i) = \frac{n-1}{n^2(n+1)} \approx \frac{1}{n^2} \text{ for large } n$$

The coefficient of variation (standard deviation / mean):

$$\text{CV} = \sqrt{n - 1} \approx \sqrt{n}$$

With $R$ virtual nodes per real node (total $nR$ points on ring), treating each real
node as a collection of $R$ arcs:

$$\text{CV}_R \approx \sqrt{\frac{n}{R}}$$

Increasing $R$ by a factor of 4 halves the CV.

### Worked Examples

3 nodes, 3000 keys:

| Replicas $R$ | CV | Typical range per node |
|:---:|:---:|:---:|
| 1 | $\sqrt{3} \approx 1.73$ | 0--3000 |
| 10 | $\sqrt{0.3} \approx 0.55$ | 550--1450 |
| 100 | $\sqrt{0.03} \approx 0.17$ | 830--1170 |
| 150 | $\sqrt{0.02} \approx 0.14$ | 860--1140 |

At $R = 150$, each node gets within ~14% of the ideal 1000 keys.

---

## 4. Binary Search on the Ring (Algorithm Design)

### The Problem

How does the ring lookup achieve O(log n) time?

### The Formula

The sorted array $S$ of $|S| = nR$ hash values supports binary search. For a key hash
$h$, find the smallest index $i$ such that $S[i] \ge h$:

$$i = \text{bisect\_left}(S, h)$$

If $i = |S|$, wrap to $i = 0$. The node is $\text{ring}[S[i]]$.

**Complexity:** Binary search is $O(\log(nR))$. With $R = 150$ and $n = 100$, this is
$\log_2(15{,}000) \approx 14$ comparisons.

**BTreeMap alternative (Rust):** A B-tree map with `range(h..)` provides the same
$O(\log(nR))$ lookup without maintaining a separate sorted array. The `range` iterator
finds the first key $\ge h$ in logarithmic time.

### Worked Examples

Sorted keys: $[5, 12, 28, 45, 67, 89]$. Looking up key hash $h = 30$:

- Binary search: $\text{bisect\_left}([5, 12, 28, 45, 67, 89], 30) = 3$
- $S[3] = 45 \ge 30$. Node: `ring[45]`.

Looking up $h = 95$:
- $\text{bisect\_left} = 6 = |S|$. Wrap to 0.
- Node: `ring[5]`.

---

## 5. Hash Function Quality and Collisions (Information Theory)

### The Problem

What properties must the hash function satisfy for consistent hashing to work correctly?

### The Formula

The hash function $h$ must provide:

1. **Determinism:** Same input always produces same output.
2. **Uniformity:** Outputs are approximately uniformly distributed over $[0, 2^m)$.
3. **Avalanche:** Small input changes produce large output changes.

**Collision probability** (birthday paradox): With $nR$ virtual nodes, the probability
of at least one hash collision is approximately:

$$P(\text{collision}) \approx 1 - e^{-\frac{(nR)^2}{2 \cdot 2^m}}$$

For 64-bit hashes ($m = 64$) with $nR = 15{,}000$:

$$P \approx 1 - e^{-\frac{2.25 \times 10^8}{3.69 \times 10^{19}}} \approx 6.1 \times 10^{-12}$$

Negligible. For 32-bit hashes with the same parameters:

$$P \approx 1 - e^{-\frac{2.25 \times 10^8}{8.59 \times 10^9}} \approx 2.6\%$$

Use at least 64-bit hashes in production.

### Worked Examples

MD5 (128-bit) truncated to 64 bits: collision probability for 10,000 virtual nodes
is $\approx 2.7 \times 10^{-12}$. Practically zero.

FNV-1a (32-bit) for 10,000 virtual nodes: $\approx 1.2\%$ collision probability.
Acceptable for testing, not for production.

---

## Prerequisites

- Hash functions and their properties
- Binary search on sorted arrays
- Modular arithmetic
- Basic probability (birthday paradox)
- Distributed systems concepts (partitioning, replication)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Implement a basic hash ring with no virtual nodes. Test that the same key always maps to the same node. Verify wrap-around behavior. |
| **Intermediate** | Add virtual nodes and measure distribution evenness. Implement add/remove with minimal key remapping. Use binary search for O(log n) lookup. Add thread safety with read-write locks. |
| **Advanced** | Analyze load variance as a function of replica count. Implement weighted nodes for heterogeneous clusters. Compare with rendezvous hashing and jump hash. Study Dynamo and Cassandra's use of consistent hashing with virtual nodes and replication. |
