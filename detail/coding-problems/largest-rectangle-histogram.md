# The Mathematics of Largest Rectangle in Histogram -- Monotonic Stacks and Amortized Linearity

> *Every bar in the histogram is the "tallest bar" of some maximal rectangle. Finding the horizontal extent of that rectangle -- the span where no bar is shorter -- reduces to computing the nearest smaller element on both sides. A monotonic stack computes both in a single linear pass, with each index pushed and popped exactly once: O(n) through amortized analysis of a superficially O(n^2) loop.*

---

## 1. The Rectangle Decomposition

### The Problem

Given heights $H = [h_0, h_1, \ldots, h_{n-1}]$, find:

$$\max_{0 \leq i < n} \, h_i \cdot (R_i - L_i - 1)$$

where $L_i$ is the largest index $j < i$ with $h_j < h_i$ (or $-1$ if none), and $R_i$ is the smallest index $j > i$ with $h_j < h_i$ (or $n$ if none).

The optimal rectangle's height equals some $h_i$, and its width equals the span of consecutive bars all $\geq h_i$. Every optimal rectangle has a "bottleneck bar" whose height determines the rectangle height.

### Worked Example

$H = [2, 1, 5, 6, 2, 3]$:

| $i$ | $h_i$ | $L_i$ | $R_i$ | Width | Area |
|-----|-------|-------|-------|-------|------|
| 0   | 2     | -1    | 1     | 1     | 2    |
| 1   | 1     | -1    | 6     | 6     | 6    |
| 2   | 5     | 1     | 4     | 2     | 10   |
| 3   | 6     | 2     | 4     | 1     | 6    |
| 4   | 2     | 1     | 6     | 4     | 8    |
| 5   | 3     | 4     | 6     | 1     | 3    |

Maximum: 10 at $i = 2$.

---

## 2. Monotonic Stacks and Nearest Smaller Elements

### The Nearest-Smaller-Element Problem

For each index $i$, find the nearest $j < i$ with $h_j < h_i$. The output is an array $L$ where $L[i] = j$ or $-1$.

### Algorithm

Maintain a stack of indices whose heights form a strictly increasing sequence from bottom to top. For each new index $i$:

1. Pop from the stack while the top's height is $\geq h_i$.
2. If the stack is empty, $L[i] = -1$. Otherwise, $L[i] = \text{stack.top()}$.
3. Push $i$.

### Invariant

At all times, the heights of stack elements form a strictly increasing sequence. When $h_i$ violates the invariant, we pop until restored; the popped indices have just found their "nearest smaller on the right" ($i$), which is exactly the information we need for the rectangle computation.

### Worked Example

$H = [2, 1, 5, 6, 2, 3]$, stack processing:

| Step | $i$ | $h_i$ | Pop indices | Stack after | $L[i]$ |
|------|-----|-------|-------------|-------------|--------|
| 1    | 0   | 2     | -           | [0]         | -1     |
| 2    | 1   | 1     | [0]         | [1]         | -1     |
| 3    | 2   | 5     | -           | [1, 2]      | 1      |
| 4    | 3   | 6     | -           | [1, 2, 3]   | 2      |
| 5    | 4   | 2     | [3], [2]    | [1, 4]      | 1      |
| 6    | 5   | 3     | -           | [1, 4, 5]   | 4      |

Symmetric right-to-left pass yields $R$. In the combined single-pass algorithm, we do both sides at once: when we pop index $k$ at step $i$, we record $R[k] = i$ and use $L[k]$ = new stack top.

---

## 3. Amortized Linear Time

### The Counting Argument

A naive look at the algorithm's two nested loops (`for i`, `while stack`) suggests $O(n^2)$. The correct bound is $O(n)$ via amortized analysis.

**Theorem**: The total number of stack operations across all $n$ outer iterations is at most $2n$.

**Proof (Aggregate Method)**:

Let $P$ = total pushes, $Q$ = total pops.

- $P \leq n$: each index is pushed exactly once (in the step $i$ where it is encountered).
- $Q \leq n$: once popped, an index never returns to the stack; hence total pops cannot exceed total pushes $\leq n$.

Total operations: $P + Q \leq 2n$. Each operation is $O(1)$. Total time: $O(n)$.

### Accounting Method Alternative

Charge 2 "credits" to each push. Use 1 credit at push time and save 1 credit with the index. When the index is later popped, use the saved credit to pay for the pop. Every operation is prepaid. Total credits allocated: $2n$. Total time: $O(n)$.

---

## 4. Correctness Proof

### Loop Invariant

After processing index $i$:
1. The stack contains indices in increasing order of both position and height.
2. For every index popped so far, we have correctly computed its maximum rectangle.

### The Pop Rule

When popping index $k$ at step $i$:
- Right boundary of $k$'s rectangle: $R_k = i$ (the first index to the right with smaller or equal height)
- Left boundary: $L_k$ = stack top after popping $k$, or $-1$ if empty
- Rectangle width: $i - L_k - 1$
- Rectangle area: $h_k \cdot (i - L_k - 1)$

### The Sentinel

Appending a sentinel bar of height 0 at index $n$ guarantees every real bar gets popped, since 0 is strictly less than any non-negative real bar height. Without the sentinel, bars left in the stack at the end of the main loop need a separate drain pass.

### Handling Equal Heights

The pop condition is `heights[top] > h` (strict), not `>=`. On equal heights, we do not pop. This is correct: if bars at indices $i < j$ have equal height $h$, the rectangle containing $i$ extends at least to $j$, so we should not compute $i$'s rectangle yet. The later pop at some $k > j$ with smaller height will correctly use $L[j] = $ index of last smaller bar before $i$ — giving the same $L$ value for $i$ and $j$.

Alternative: use `>=` and compensate. Equivalent results, slightly cleaner code if you track the leftmost equal-height index carefully.

---

## 5. Generalisations

### Maximal Rectangle in Binary Matrix (LC 85)

For an $m \times n$ binary matrix, define $H_r[c]$ = number of consecutive 1s ending at row $r$, column $c$. Apply the histogram algorithm to each row:

$$\text{max-area} = \max_{r} \, \text{LargestRect}(H_r)$$

Total time: $O(m \cdot n)$. Space: $O(n)$ for the running histogram.

### Maximum Subarray Sum (non-histogram but related)

Kadane's algorithm and the monotonic-stack approach both rely on identifying a bottleneck. Kadane identifies prefix sum minima; monotonic stack identifies height bottlenecks. Both are $O(n)$ by amortisation.

### Stock Span Problem

Given stock prices, compute the number of consecutive prior days with price $\leq$ today's. Same monotonic-stack machinery, different interpretation. The stack stores indices of decreasing prices; pop when a new higher price arrives.

---

## 6. Lower Bound

### Comparison Model

Any algorithm that computes the histogram maximum rectangle must, at minimum, examine every bar at least once (else the adversary adjusts an unexamined bar arbitrarily tall). Hence $\Omega(n)$.

The monotonic stack achieves $O(n)$, matching the lower bound. No asymptotically faster algorithm exists in the comparison model.

### Divide-and-Conquer ($O(n \log n)$)

An alternative: find the minimum bar (breaks the histogram into two), recurse on left and right, and consider the rectangle spanning the whole range at height equal to the minimum.

Recurrence $T(n) = T(a) + T(n - a - 1) + O(n)$ gives $O(n \log n)$ for balanced splits (min near the middle), but degenerates to $O(n^2)$ on a sorted array.

With a sparse table for range-minimum queries, splits become $O(1)$ and total time reaches $O(n \log n)$. Still strictly worse than the monotonic stack's $O(n)$.

---

## Prerequisites

- **Stacks**: LIFO semantics, amortized analysis of push/pop sequences
- **Amortized analysis**: aggregate method, accounting method, potential method
- **Invariants**: loop invariants for stack-based algorithms, monotonicity preservation
- **Nearest smaller element**: the primitive behind many monotonic stack problems
- **Range minimum queries**: for the $O(n \log n)$ divide-and-conquer alternative

## Complexity

| Aspect | Bound | Notes |
|--------|-------|-------|
| Monotonic stack time | $O(n)$ | Amortized: each index pushed/popped once |
| Monotonic stack space | $O(n)$ | Stack depth bounded by $n$ |
| Divide-and-conquer time | $O(n \log n)$ | With sparse table for RMQ |
| Divide-and-conquer space | $O(n \log n)$ | Sparse table size |
| Naive time | $O(n^2)$ | Per-bar left/right expansion |
| Lower bound (comparisons) | $\Omega(n)$ | Must examine every bar |
| Binary matrix extension time | $O(m \cdot n)$ | Histogram per row |
| Binary matrix extension space | $O(n)$ | Running histogram only |

---
