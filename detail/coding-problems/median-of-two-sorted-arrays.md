# The Mathematics of Median of Two Sorted Arrays -- Partition Search and Order Statistics

> *Merging two sorted arrays and taking the middle is the naive O(m + n) baseline. Binary-searching a balanced partition across the shorter array cuts this to O(log min(m, n)) — a textbook example of how order statistics on sorted inputs admit logarithmic-time solutions through constraint propagation on an invariant.*

---

## 1. The Partition Invariant

### The Problem

Given sorted arrays $A$ of length $m$ and $B$ of length $n$, the combined median partitions the merged array into two halves of nearly equal size. We seek indices $i \in [0, m]$ and $j \in [0, n]$ with:

$$i + j = \left\lceil \frac{m + n}{2} \right\rceil$$

such that the left partition $A[0..i-1] \cup B[0..j-1]$ contains the smallest $i + j$ elements of the merged array, and the right partition $A[i..m-1] \cup B[j..n-1]$ contains the remainder.

### The Four-Pointer Condition

Define:
- $L_A = A[i-1]$ with $A[-1] = -\infty$
- $R_A = A[i]$ with $A[m] = +\infty$
- $L_B = B[j-1]$ with $B[-1] = -\infty$
- $R_B = B[j]$ with $B[n] = +\infty$

The partition is correct if and only if:

$$L_A \leq R_B \quad \text{and} \quad L_B \leq R_A$$

This guarantees every element in the left partition is $\leq$ every element in the right partition, even across arrays.

### The Median Formula

Let $k = \lceil (m + n) / 2 \rceil$. For odd total length:

$$\text{median} = \max(L_A, L_B)$$

For even total length:

$$\text{median} = \frac{\max(L_A, L_B) + \min(R_A, R_B)}{2}$$

### Worked Example

$A = [1, 3, 8, 9, 15]$, $B = [7, 11, 18, 19, 21, 25]$. Combined length $m + n = 11$, so $k = 6$.

Trial $i = 2$, $j = 4$:
- $L_A = A[1] = 3$, $R_A = A[2] = 8$
- $L_B = B[3] = 19$, $R_B = B[4] = 21$
- Check: $L_A = 3 \leq R_B = 21$ ✓, but $L_B = 19 \not\leq R_A = 8$ ✗

Too few elements from $A$. Increase $i$. Trial $i = 3$, $j = 3$:
- $L_A = A[2] = 8$, $R_A = A[3] = 9$
- $L_B = B[2] = 18$, $R_B = B[3] = 19$
- Check: $8 \leq 19$ ✓, $18 \not\leq 9$ ✗

Still increase $i$. Trial $i = 4$, $j = 2$:
- $L_A = A[3] = 9$, $R_A = A[4] = 15$
- $L_B = B[1] = 11$, $R_B = B[2] = 18$
- Check: $9 \leq 18$ ✓, $11 \leq 15$ ✓

Valid. Odd total: median $= \max(9, 11) = 11$.

---

## 2. Correctness Proof

### Loop Invariant

After each binary-search iteration, the correct $i^*$ lies in $[lo, hi]$.

### Key Lemma

If $L_A > R_B$, then the correct $i^*$ is strictly less than current $i$.

**Proof sketch**: $L_A > R_B$ means $A[i-1] > B[j]$. In the merged sequence, $A[i-1]$ belongs to the right partition (not the left), so $i$ is too large. Any partition with $i' \geq i$ would place $A[i-1]$ in the left side and violate order; hence $i^* < i$. Binary search halves the candidate range.

### Termination

The search interval $[lo, hi]$ strictly shrinks each iteration. After $\lceil \log_2(m + 1) \rceil$ iterations, either the valid partition is found or the interval collapses. Since a valid partition always exists for sorted inputs, the loop terminates with the correct $i^*$.

### Why Binary-Search the Shorter Array

Searching $A$ with $m \leq n$ guarantees $i \in [0, m]$, so $j = k - i$ stays in $[k - m, k]$. With $k \leq (m + n + 1) / 2$, we have $j \leq (m + n + 1)/2$ and $j \geq k - m \geq (n - m - 1)/2 \geq 0$. If instead we searched the longer array, $j$ could go negative or exceed $n$, breaking sentinel logic.

---

## 3. The k-th Smallest Generalisation

### Formulation

Finding the $k$-th smallest element in $A \cup B$ (1-indexed) uses the same partition search with $i + j = k$. The answer is:

$$\text{kth} = \max(L_A, L_B)$$

This subsumes the median problem ($k = \lceil (m+n)/2 \rceil$) and also handles weighted medians, quartiles, and quantiles over pre-sorted streams.

### Recursive Elimination (Alternative)

An alternative $O(\log k)$ approach compares $A[k/2 - 1]$ with $B[k/2 - 1]$ and discards the smaller half:

$$\text{if } A[k/2 - 1] < B[k/2 - 1] : \text{discard } A[0 .. k/2 - 1], \text{ recurse with } k - k/2$$

Each recursive call halves $k$, giving $T(k) = T(k/2) + O(1) = O(\log k)$. This bound is $O(\log(m + n))$ for the median case — slightly weaker than $O(\log \min(m, n))$ but often easier to implement.

---

## 4. Lower Bound

### Information-Theoretic Argument

Any algorithm that determines the median must distinguish between the possible answers. On arrays of size $m$ and $n$, there are $m + n$ possible median values (considering only positions). A comparison-based algorithm requires $\Omega(\log(m + n))$ comparisons in the worst case, matching the upper bound.

**Theorem (Lower Bound)**: Computing the median of two sorted arrays of sizes $m$ and $n$ requires $\Omega(\log \min(m, n) + \log \log(m + n))$ comparisons in the comparison-based model.

The partition-search algorithm achieves $O(\log \min(m, n))$, within a $\log\log$ factor of optimal.

---

## 5. Numerical Stability

### Overflow Avoidance

In the even-length median formula:

$$\text{median} = \frac{\max(L_A, L_B) + \min(R_A, R_B)}{2}$$

For 32-bit signed integers near $\pm 2^{31}$, the sum overflows before division. Safe alternatives:

$$\text{median} = \frac{\max(L_A, L_B)}{2} + \frac{\min(R_A, R_B)}{2}$$

This halves each term first. For integer inputs, this may lose the fractional part; upgrade to `i64` or `f64` arithmetic.

### Sentinel Representation

Using `math.MinInt` and `math.MaxInt` as sentinels is portable. Alternatives:

- Conditional branching at boundaries (clean but verbose)
- `NaN` or `None` with explicit checks (type-system-friendly)
- `-∞` / `+∞` in floating-point (natural but loses integer precision)

The sentinel approach in the solutions above is the simplest and the least bug-prone.

---

## Prerequisites

- **Binary search**: loop invariant reasoning, termination, upper/lower bound search patterns
- **Sorted sequences**: merge semantics, order statistics, k-th element problems
- **Sentinel-based programming**: boundary elimination through $\pm \infty$ or equivalents
- **Amortised analysis**: understanding why naive merge is $O(m + n)$ vs sub-linear partition search

## Complexity

| Aspect | Bound | Notes |
|--------|-------|-------|
| Partition-search time | $O(\log \min(m, n))$ | Binary search on shorter array |
| Recursive elimination time | $O(\log(m + n))$ | Alternative algorithm |
| Naive merge time | $O(m + n)$ | Disqualified by problem constraints |
| Space | $O(1)$ | Constant extra memory |
| Lower bound (comparisons) | $\Omega(\log \min(m, n))$ | Information-theoretic |
| Comparisons in worst case | $\leq \lceil \log_2(m + 1) \rceil$ | Partition-search algorithm |

---
