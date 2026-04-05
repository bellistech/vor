# The Mathematics of Two Sum -- Hash-Based Complement Search and Amortized Hashing

> *A brute-force pair search costs O(n^2). By reframing the problem as a complement lookup and leveraging hash map amortized O(1) access, we reduce the entire search to a single linear pass -- the canonical example of trading space for time.*

---

## 1. The Two Sum Domain

### The Problem

Given an array $A = [a_0, a_1, \ldots, a_{n-1}]$ and a target value $t$, find indices $i \neq j$ such that:

$$a_i + a_j = t$$

Exactly one solution exists. Return $[i, j]$ with $i < j$.

### The Brute-Force Baseline

The naive approach tests all pairs:

$$T_{\text{brute}} = \sum_{i=0}^{n-2} \sum_{j=i+1}^{n-1} 1 = \binom{n}{2} = \frac{n(n-1)}{2} \in O(n^2)$$

For $n = 10^4$, this is roughly $5 \times 10^7$ comparisons -- feasible but wasteful.

### The Formula -- Complement Rewriting

The key insight is algebraic: for each element $a_i$, the required partner satisfies:

$$a_j = t - a_i$$

Define the **complement** $c_i = t - a_i$. The problem reduces to: for each $i$, does $c_i$ exist in the set of previously seen values?

This transforms a **search problem** into a **membership problem**, which hash tables solve in amortized $O(1)$.

### Worked Examples

**Example: $A = [2, 7, 11, 15]$, $t = 9$**

| Step $i$ | $a_i$ | Complement $c_i = 9 - a_i$ | Hash Map State | Found? |
|----------|--------|-----------------------------|----------------|--------|
| 0        | 2      | 7                           | {}             | No     |
| 1        | 7      | 2                           | {2: 0}         | Yes, at index 0 |

Result: $[0, 1]$.

At step 0, the complement 7 is not in the (empty) map, so we insert $\{2 \to 0\}$.
At step 1, the complement 2 is found at index 0, so we return immediately.

**Example: $A = [3, 2, 4]$, $t = 6$**

| Step $i$ | $a_i$ | Complement $c_i$ | Hash Map State | Found? |
|----------|--------|-------------------|----------------|--------|
| 0        | 3      | 3                 | {}             | No     |
| 1        | 2      | 4                 | {3: 0}         | No     |
| 2        | 4      | 2                 | {3: 0, 2: 1}   | Yes, at index 1 |

Result: $[1, 2]$.

Note at step 0: the complement of 3 is 3 itself, but the map is empty so there is no false self-match. This illustrates why we check *before* inserting.

---

## 2. Hash Map Analysis -- Why O(n) Works

### Expected-Case Analysis

A hash map with a good hash function and load factor $\alpha < 1$ provides:

- **Insert:** $O(1)$ expected (amortized $O(1)$ with dynamic resizing)
- **Lookup:** $O(1)$ expected

Over $n$ iterations, each performing one lookup and at most one insert:

$$T_{\text{expected}} = \sum_{i=0}^{n-1} O(1) = O(n)$$

### Worst-Case Considerations

In the worst case, all keys collide into the same bucket, degrading lookup to $O(n)$ and total time to $O(n^2)$. However:

1. **Randomized hashing** (e.g., SipHash in Rust, randomized seed in Go) provides $O(1)$ expected time per operation regardless of input distribution.
2. **Open addressing with Robin Hood hashing** bounds the maximum probe length to $O(\log n)$ with high probability.

For practical purposes, the algorithm is $O(n)$ expected, $O(n \log n)$ with high probability under randomized hashing.

### Space Analysis

The hash map stores at most $n$ key-value pairs. Each pair is $(a_i, i)$, requiring $O(1)$ space. Total:

$$S = O(n)$$

With typical hash map overhead (load factor, metadata), the constant factor is roughly $2$-$4\times$ the raw data size.

---

## 3. The Complement Pattern as a General Technique

### Generalization to k-Sum

The Two Sum complement pattern extends hierarchically:

- **Two Sum:** For each $a_i$, look up $t - a_i$. Time: $O(n)$.
- **Three Sum:** For each $a_i$, solve Two Sum on the remainder with target $t - a_i$. Time: $O(n^2)$.
- **k-Sum:** Recursively reduce to $(k-1)$-Sum. Time: $O(n^{k-1})$.

The general recurrence is:

$$T(n, k) = n \cdot T(n, k-1), \quad T(n, 2) = O(n) \implies T(n, k) = O(n^{k-1})$$

### Alternative: Sort + Two Pointers

If we sort the array first ($O(n \log n)$), two pointers from both ends find the pair in $O(n)$:

$$T_{\text{sort}} = O(n \log n) + O(n) = O(n \log n)$$

This uses $O(1)$ extra space (or $O(n)$ if we need original indices), trading time for space compared to the hash map approach.

### Comparison of Approaches

| Approach | Time | Space | Returns Indices? |
|----------|------|-------|------------------|
| Brute force | $O(n^2)$ | $O(1)$ | Yes |
| Hash map | $O(n)$ expected | $O(n)$ | Yes |
| Sort + two pointers | $O(n \log n)$ | $O(n)$ | Yes (with index tracking) |
| Sort + binary search | $O(n \log n)$ | $O(n)$ | Yes (with index tracking) |

---

## 4. Correctness Proof

### Invariant

After processing elements $a_0, a_1, \ldots, a_{i-1}$, the hash map $H$ satisfies:

$$H = \{a_j \to j \mid 0 \leq j < i\}$$

### Proof of Correctness

Let $(p, q)$ with $p < q$ be the unique solution, so $a_p + a_q = t$.

- When $i = q$, we compute $c_q = t - a_q = a_p$.
- Since $p < q$, element $a_p$ was inserted at step $p$, so $a_p \in H$ with $H[a_p] = p$.
- The lookup succeeds and returns $[p, q]$.

### No False Positives

We check the map *before* inserting $a_i$. Therefore, $a_i$ itself is never in the map during its own lookup, preventing $i = j$ false matches.

For duplicate values (e.g., $A = [3, 3]$, $t = 6$): at step 1, $c_1 = 3$ is found in the map from step 0's insertion. The map stores the *first* occurrence's index, which is different from the current index, so the result $[0, 1]$ is correct.

---

## Prerequisites

- **Hash tables:** amortized O(1) insert and lookup under uniform hashing assumption.
- **Complement rewriting:** algebraic reformulation of $a + b = t$ as $b = t - a$.
- **Amortized analysis:** understanding expected vs. worst-case bounds for hash operations.
- **Space-time tradeoff:** the foundational concept that additional memory can eliminate redundant computation.

## Complexity

| Metric          | Value  | Justification                                                  |
|-----------------|--------|----------------------------------------------------------------|
| Time            | O(n)   | Single pass with O(1) expected hash map operations per element |
| Space           | O(n)   | Hash map stores at most n key-value pairs                      |
| Brute-force time | O(n^2) | Comparison baseline; hash map is an n-factor improvement       |
| Sort approach   | O(n log n) | Alternative when space is constrained                      |
