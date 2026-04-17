# The Mathematics of Running Median -- Dual Heaps and Order Statistics on Streams

> *The median is the order statistic at rank $\lceil n/2 \rceil$. Computing it over a batch array with quickselect is $O(n)$ worst case. Computing it over a stream in amortised constant time requires a more structured invariant: two heaps partition the data so that the median is always at one of their peaks, maintained through $O(\log n)$ rebalancing on each insertion.*

---

## 1. The Two-Heap Invariant

### Partition Structure

After $n$ insertions, the data stream is conceptually sorted. We maintain two heaps:

- **Lower half** $L$: max-heap containing the $\lceil n/2 \rceil$ smallest elements. The maximum of $L$ (its root) is the largest element in the lower half.
- **Upper half** $U$: min-heap containing the $\lfloor n/2 \rfloor$ largest elements. The minimum of $U$ (its root) is the smallest element in the upper half.

Invariants (I1, I2, I3):

$$\text{I1:} \quad \forall x \in L, \forall y \in U : x \leq y$$
$$\text{I2:} \quad |L| = |U| \text{ or } |L| = |U| + 1$$
$$\text{I3:} \quad L, U \text{ satisfy the heap property}$$

### The Median Formula

$$\text{median}(S) = \begin{cases} \text{root}(L) & \text{if } |L| > |U| \text{ (odd count)} \\ \frac{\text{root}(L) + \text{root}(U)}{2} & \text{if } |L| = |U| \text{ (even count)} \end{cases}$$

By I1, root($L$) is the largest of the lower half and root($U$) is the smallest of the upper half — exactly the two middle elements when the total count is even.

---

## 2. Insertion Algorithm

### The Two-Step Rule

For incoming value $v$:

**Step 1 (tentative placement)**: Compare $v$ against root($L$). If $v \leq $ root($L$) or $L$ is empty, push to $L$; otherwise push to $U$.

**Step 2 (rebalance)**: Restore I2.

- If $|L| > |U| + 1$: pop root($L$), push into $U$.
- Else if $|U| > |L|$: pop root($U$), push into $L$.

### Invariant Preservation

**Claim**: Steps 1 + 2 preserve I1, I2, I3.

**Proof**:

I3 holds trivially — push and pop are heap operations that preserve the heap property.

I1 holds after Step 1 by the decision rule: $v$ is placed on the side it belongs to.

I2 may be violated by Step 1 but is restored by Step 2. The rebalance moves at most one element, and the element moved is the root (the extreme of its heap), preserving I1: moving root($L$) to $U$ places the largest of the lower half on the boundary — still $\leq$ all of $U$ by I1's prior state.

### Cost Analysis

- Step 1: one heap push = $O(\log n)$.
- Step 2: up to one pop + one push = $O(\log n)$.

Total per `addNum`: $O(\log n)$.

`findMedian`: $O(1)$ — peek at one or both heap roots.

---

## 3. Competitive Lower Bound

### Comparison Model Lower Bound

**Theorem**: Any comparison-based algorithm that maintains the running median under insertions requires $\Omega(\log n)$ amortised time per insertion.

**Proof sketch**: After $n$ insertions, the algorithm must be able to answer a sequence of median queries. A single median query reveals one element's rank. A standard information-theoretic argument shows that to distinguish among $n!$ possible orderings, the algorithm must perform $\Omega(\log(n!)) = \Omega(n \log n)$ total comparisons over $n$ insertions, giving $\Omega(\log n)$ amortised per insertion.

The two-heap solution achieves $O(\log n)$ exactly, matching the lower bound up to constants.

---

## 4. Alternative Data Structures

### Balanced BST / Order-Statistic Tree

A self-balancing BST augmented with subtree sizes supports:

- Insert: $O(\log n)$
- `select(k)`: find the $k$-th smallest, $O(\log n)$

Median is `select(⌈n/2⌉)`. Same asymptotic cost as two heaps but larger constants and more complex implementation.

### Skip List

Probabilistic balanced BST alternative with $O(\log n)$ expected insert and select. Suitable when deletion is also required (e.g., sliding-window median — LC 480).

### T-Digest (Approximate)

For high-volume streams where exactness is unnecessary:

- **T-digest** (Dunning, 2014): maintains centroids clustered around quantiles; merges use compression factor $\delta$.
- **Error bound**: approximate quantile $q$ has relative error $O(\sqrt{q(1-q)} / \delta)$. For median ($q = 0.5$), error is $O(1/(2\delta))$.
- **Space**: $O(\delta)$ regardless of stream size.
- **Time per insert**: amortised $O(\log \delta)$.

T-digest is the industry-standard choice for percentile monitoring in distributed systems (Prometheus histograms, Datadog metrics, Apache Druid).

### Reservoir Sampling

Sample $k$ elements uniformly at random from a stream of unknown length, compute median of the sample. Error decreases as $O(1/\sqrt{k})$ by the central limit theorem. Suitable for rough median estimation with bounded memory.

---

## 5. Sliding Window Extension (LC 480)

### The Deletion Challenge

For median over the last $k$ elements of a stream, arbitrary element deletion is required when the window advances. Standard binary heaps do not support $O(\log n)$ arbitrary deletion.

### Lazy Deletion Pattern

Augment the two-heap solution with a hash map `toDelete: value -> count`. When an element falls out of the window:

1. Increment `toDelete[old_value]`.
2. Track the size correction in a separate counter per side.
3. At the top of `findMedian`, repeatedly pop heap roots while `toDelete[root] > 0` and decrement.

Amortised $O(\log k)$ per window slide. Correctness hinges on only needing correct heap roots — stale elements deeper in the heap are harmless until they surface.

### Indexed Structure Alternative

A **multiset** (e.g., C++ `std::multiset`, Java `TreeMap`) supports $O(\log k)$ insert, delete, and access to the $i$-th element via iterator arithmetic.

Order-statistic trees (augmented red-black trees) achieve the same bounds explicitly.

---

## 6. Streaming Quantile Summaries

### The Greenwald-Khanna Algorithm

For $\epsilon$-approximate quantile queries over a stream of length $N$:

- Space: $O(\log(\epsilon N) / \epsilon)$
- Error bound: quantile $q$ is returned with rank in $[(q - \epsilon)N, (q + \epsilon)N]$

Used in production systems when space is the binding constraint (e.g., network telemetry summarisation).

### Count-Min Sketch

For frequency estimation (not directly median), $O(1/\epsilon)$ space with $\epsilon$ additive error. Combined with binary search, yields approximate median with extra complexity.

---

## 7. Numerical Considerations

### Integer Overflow

For 32-bit integers near $\pm 2^{31}$:

$$\frac{\text{root}(L) + \text{root}(U)}{2}$$

can overflow. Safe alternatives:

$$\text{median} = \text{root}(L) + \frac{\text{root}(U) - \text{root}(L)}{2}$$

The subtraction stays bounded when both roots are on the same side of zero. When they straddle zero, the result is trivially non-overflowing since one is $\leq 0$ and the other $\geq 0$.

### Floating-Point Median

Converting integer sums to float before division loses precision for 64-bit integers whose magnitude exceeds $2^{53}$ (the f64 mantissa limit). For such cases, use 128-bit integer arithmetic or symbolic rational representation.

---

## Prerequisites

- **Binary heaps**: structure, heap property, sift-up, sift-down, $O(\log n)$ push/pop
- **Order statistics**: $k$-th smallest/largest, quickselect, median-of-medians
- **Invariant-based reasoning**: loop and data-structure invariants, preservation under operations
- **Amortised analysis**: aggregate and potential methods for sequences of operations
- **Probabilistic data structures**: for streaming approximations (t-digest, reservoir sampling)

## Complexity

| Aspect | Bound | Notes |
|--------|-------|-------|
| `addNum` time | $O(\log n)$ | Up to 3 heap operations |
| `findMedian` time | $O(1)$ | Peek only |
| Space | $O(n)$ | All elements retained |
| Lower bound per insert | $\Omega(\log n)$ | Comparison-model information-theoretic |
| Skip list alternative | $O(\log n)$ expected | Supports deletion natively |
| T-digest approximate | $O(\log \delta)$ | Space $O(\delta)$, error $O(1/\delta)$ |
| Greenwald-Khanna | $O(\log(\epsilon N)/\epsilon)$ space | $\epsilon$-approximate rank |
| Sliding window median (LC 480) | $O(\log k)$ amortised | Lazy deletion or multiset |
| Reservoir sampling | $O(k)$ space, $O(1)$ per insert | Central limit error $O(1/\sqrt{k})$ |

---
