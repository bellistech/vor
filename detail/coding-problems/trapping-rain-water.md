# The Mathematics of Trapping Rain Water -- Two Pointers and Prefix Extrema

> *A brute-force per-bar scan costs O(n^2). By recognizing that trapped water at each position depends only on the minimum of two running maxima, we collapse the computation to a single O(n) pass with O(1) space -- a canonical example of the two-pointer narrowing technique.*

---

## 1. The Trapping Rain Water Domain

### The Problem

Given an array $H = [h_0, h_1, \ldots, h_{n-1}]$ of non-negative integers representing bar heights, compute the total volume of water trapped between bars after raining:

$$W = \sum_{i=0}^{n-1} \max\bigl(0, \, \min(L_i, R_i) - h_i\bigr)$$

where $L_i = \max(h_0, h_1, \ldots, h_i)$ and $R_i = \max(h_i, h_{i+1}, \ldots, h_{n-1})$ are the prefix and suffix maxima.

### The Brute-Force Baseline

For each bar $i$, scan left and right to find $L_i$ and $R_i$:

$$T_{\text{brute}} = \sum_{i=0}^{n-1} O(n) = O(n^2)$$

### The Prefix Array Approach

Precompute $L[i]$ and $R[i]$ in two passes:

$$L[0] = h_0, \quad L[i] = \max(L[i-1], h_i)$$
$$R[n-1] = h_{n-1}, \quad R[i] = \max(R[i+1], h_i)$$

Then $W = \sum_{i=0}^{n-1} \max(0, \min(L[i], R[i]) - h_i)$. Time: $O(n)$. Space: $O(n)$.

### The Two-Pointer Formula

The key insight: if $L_{\ell} < R_r$ for pointers $\ell$ and $r$, then the water at position $\ell$ is exactly $L_{\ell} - h_{\ell}$ (since the true right max is at least $R_r > L_{\ell}$, so $\min(L_{\ell}, R_{\ell}) = L_{\ell}$).

By symmetry, if $L_{\ell} \geq R_r$, the water at position $r$ is $R_r - h_r$.

### Worked Examples

**Example: $H = [0, 1, 0, 2, 1, 0, 1, 3, 2, 1, 2, 1]$**

| Step | $\ell$ | $r$ | $L_\ell$ | $R_r$ | Action | Water added |
|------|--------|-----|-----------|--------|--------|-------------|
| 1    | 0      | 11  | 0         | 1      | advance left  | 0  |
| 2    | 1      | 11  | 1         | 1      | advance right | 0  |
| 3    | 1      | 10  | 1         | 2      | advance left  | 0  |
| 4    | 2      | 10  | 1         | 2      | advance left  | 1  |
| 5    | 3      | 10  | 2         | 2      | advance right | 0  |
| 6    | 3      | 9   | 2         | 2      | advance right | 1  |
| 7    | 3      | 8   | 2         | 2      | advance right | 0  |
| 8    | 3      | 7   | 2         | 3      | advance left  | 0  |
| 9    | 4      | 7   | 2         | 3      | advance left  | 1  |
| 10   | 5      | 7   | 2         | 3      | advance left  | 2  |
| 11   | 6      | 7   | 2         | 3      | advance left  | 1  |

Total water: $0 + 0 + 0 + 1 + 0 + 1 + 0 + 0 + 1 + 2 + 1 = 6$ .

---

## 2. Correctness Proof -- Why Two Pointers Work

### Invariant

At each step, all positions outside $[\ell, r]$ have already been correctly processed.

### Key Lemma

If $h[\ell] < h[r]$, then the true right maximum for position $\ell$ satisfies $R_\ell \geq h[r] \geq h[\ell]$. Therefore:

$$\min(L_\ell, R_\ell) = L_\ell$$

and the water at $\ell$ is exactly $\max(0, L_\ell - h[\ell])$.

### Proof by Induction

**Base case:** $\ell = 0, r = n-1$. No positions have been processed; the invariant holds vacuously.

**Inductive step:** Assume the invariant holds for current $[\ell, r]$. Without loss of generality, suppose $h[\ell] < h[r]$ (the symmetric case is analogous).

- $L_\ell$ is correctly maintained as $\max(h[0], \ldots, h[\ell])$.
- $R_\ell \geq h[r] > h[\ell]$, so $\min(L_\ell, R_\ell) = L_\ell$.
- Water at $\ell$ is computed as $L_\ell - h[\ell]$ (or 0 if $h[\ell] = L_\ell$).
- Advancing $\ell$ to $\ell + 1$ maintains the invariant for $[\ell+1, r]$.

---

## 3. Alternative Approaches

### Stack-Based Approach

Process bars left to right. Maintain a stack of indices in decreasing height order. When a bar taller than the stack top is encountered, pop and compute the water trapped in the "valley" between the current bar, the popped bar, and the new stack top.

For each popped bar at index $\text{mid}$:

$$\text{bounded height} = \min(h[\text{current}], h[\text{stack top}]) - h[\text{mid}]$$
$$\text{width} = \text{current} - \text{stack top} - 1$$
$$\text{water} += \text{bounded height} \times \text{width}$$

Time: $O(n)$ (each bar pushed/popped at most once). Space: $O(n)$.

### Relationship to Histogram Problems

Trapping Rain Water is the "dual" of Largest Rectangle in Histogram. The histogram problem finds the largest rectangle bounded by bars; rain water fills the concavities between bars. Both use stack-based monotonic structures but with inverted logic.

---

## Prerequisites

- **Two-pointer technique:** narrowing a search space from both ends.
- **Prefix/suffix extrema:** running maximum or minimum over a sequence.
- **Monotonic stack:** maintaining sorted order for next-greater/next-smaller element queries.
- **Amortized analysis:** each element processed a constant number of times across all operations.

## Complexity

| Metric          | Two Pointers | Prefix Arrays | Stack-Based | Brute Force |
|-----------------|-------------|---------------|-------------|-------------|
| Time            | O(n)        | O(n)          | O(n)        | O(n^2)      |
| Space           | O(1)        | O(n)          | O(n)        | O(1)        |
| Passes          | 1           | 3             | 1           | n           |
