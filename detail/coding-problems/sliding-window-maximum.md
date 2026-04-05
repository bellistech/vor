# The Mathematics of Sliding Window Maximum -- Monotonic Structures and Amortized Analysis

> *A brute-force window scan costs O(nk). By exploiting a monotonic invariant on a double-ended queue, we collapse every window's maximum query into amortized O(1), yielding an O(n) algorithm -- a textbook case of trading structural insight for raw computation.*

---

## 1. The Sliding Window Domain

### The Problem

Given an array $A = [a_0, a_1, \ldots, a_{n-1}]$ and a positive integer $k \leq n$, compute the sequence:

$$M_i = \max(a_i, a_{i+1}, \ldots, a_{i+k-1}) \quad \text{for } i = 0, 1, \ldots, n - k$$

The output is an array of $n - k + 1$ values.

### The Brute-Force Baseline

The naive approach scans each window independently:

$$T_{\text{brute}} = \sum_{i=0}^{n-k} (k - 1) = (n - k + 1)(k - 1) \in O(nk)$$

For $k = \Theta(n)$, this degrades to $O(n^2)$.

### The Formula -- Deque Invariant

Let $D$ be a deque of indices. After processing element $a_i$, the invariant holds:

$$D = [d_0, d_1, \ldots, d_m] \quad \text{where} \quad a_{d_0} > a_{d_1} > \cdots > a_{d_m}$$

and every $d_j \in [i - k + 1, \, i]$.

The window maximum is always $a_{d_0}$, the value at the front of the deque.

### Worked Examples

**Example: $A = [1, 3, -1, -3, 5, 3, 6, 7]$, $k = 3$**

| Step $i$ | $a_i$ | Deque (indices) | Deque (values) | Output |
|----------|--------|-----------------|----------------|--------|
| 0        | 1      | [0]             | [1]            | --     |
| 1        | 3      | [1]             | [3]            | --     |
| 2        | -1     | [1, 2]          | [3, -1]        | 3      |
| 3        | -3     | [1, 2, 3]       | [3, -1, -3]    | 3      |
| 4        | 5      | [4]             | [5]            | 5      |
| 5        | 3      | [4, 5]          | [5, 3]         | 5      |
| 6        | 6      | [6]             | [6]            | 6      |
| 7        | 7      | [7]             | [7]            | 7      |

At step 1, pushing $a_1 = 3$ evicts index 0 from the back because $a_0 = 1 \leq 3$.
At step 3, index 1 is still within the window $[1, 3]$, so it remains at the front.
At step 4, pushing $a_4 = 5$ clears the entire deque since $5 > 3 > -1 > -3$.

---

## 2. Amortized Analysis -- Why O(n) Works

### The Potential Method

Define the potential function $\Phi = |D|$, the number of elements in the deque.

For each element $a_i$, the algorithm performs:

1. **Front eviction:** remove at most 1 element from the front ($O(1)$ worst case).
2. **Back eviction:** remove $r_i$ elements from the back where $a_{d_j} \leq a_i$.
3. **Push:** add index $i$ to the back ($O(1)$).

The actual cost of step $i$ is $c_i = 1 + r_i + 1 = r_i + 2$.

The change in potential is:

$$\Delta\Phi_i = 1 - r_i \quad \text{(one push, } r_i \text{ pops from back)}$$

The amortized cost is:

$$\hat{c}_i = c_i + \Delta\Phi_i = (r_i + 2) + (1 - r_i) = 3$$

Since $\hat{c}_i = O(1)$ for every step and $\Phi \geq 0$ always:

$$\sum_{i=0}^{n-1} c_i \leq \sum_{i=0}^{n-1} \hat{c}_i + \Phi_0 - \Phi_n \leq 3n + 0 = O(n)$$

### The Counting Argument (Simpler)

Each of the $n$ indices is pushed onto the deque exactly once and popped at most once.
Total pushes: $n$. Total pops: at most $n$. Total operations: at most $2n = O(n)$.

---

## 3. The Monotonic Deque as an Abstract Data Type

### Structural Properties

The monotonic deque maintains a **strictly decreasing** sequence of values. This yields three properties:

1. **Max at front:** $a_{d_0} = \max\{a_j : j \in D\}$.
2. **Bounded size:** $|D| \leq k$ since all indices lie within a window of size $k$.
3. **Temporal ordering:** $d_0 < d_1 < \cdots < d_m$, so the deque is also sorted by index.

### Relationship to Monotonic Stacks

A monotonic stack processes elements in one direction and answers "next greater element" queries. The monotonic deque extends this by adding front-removal, enabling it to handle the **expiration** of old elements as the window slides forward. The stack is a special case where $k = n$ (no expiration).

### Space Bound

The deque contains at most $k$ elements at any time, since all stored indices must lie within $[i - k + 1, i]$. Thus space is $O(k)$, not $O(n)$.

---

## Prerequisites

- **Deque (double-ended queue):** O(1) push/pop at both ends.
- **Amortized analysis:** potential method or aggregate counting argument.
- **Loop invariants:** reasoning about correctness by maintaining the monotonic property at each step.
- **Sliding window technique:** the two-pointer/fixed-width paradigm for subarray problems.

## Complexity

| Metric          | Value  | Justification                                                  |
|-----------------|--------|----------------------------------------------------------------|
| Time            | O(n)   | Each element is pushed and popped at most once (amortized O(1) per step) |
| Space           | O(k)   | Deque stores at most k indices within the active window        |
| Output size     | O(n-k+1) | One maximum per window position                             |
| Brute-force time | O(nk) | Comparison baseline; deque approach is a k-factor improvement  |
