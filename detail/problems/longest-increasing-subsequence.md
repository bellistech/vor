# The Mathematics of Longest Increasing Subsequence -- Order, Patience, and Dilworth

> *Finding the longest chain in a sequence connects dynamic programming, the theory of partial orders, and a surprising card game -- each revealing a different face of the same combinatorial structure.*

---

## 1. Optimal Substructure and the DP Recurrence (Dynamic Programming)

### The Problem

Define the LIS length precisely using a recursive formula and prove that the problem
exhibits optimal substructure.

### The Formula

Let $L(i)$ denote the length of the longest increasing subsequence ending at index $i$.

$$L(i) = 1 + \max_{j < i,\; a_j < a_i} L(j)$$

with $L(i) = 1$ when no valid $j$ exists. The answer is:

$$\text{LIS} = \max_{0 \le i < n} L(i)$$

**Optimal substructure:** If the LIS ending at position $i$ has length $k$, then removing
$a_i$ yields an increasing subsequence of length $k - 1$ ending at some earlier position.
This must itself be optimal (longest ending at that position), because any longer one
would contradict the optimality of $L(i)$.

### Worked Examples

Array: $[10, 9, 2, 5, 3, 7, 101, 18]$

| i | $a_i$ | $L(i)$ | Best predecessor |
|---|-------|--------|-----------------|
| 0 | 10 | 1 | -- |
| 1 | 9 | 1 | -- |
| 2 | 2 | 1 | -- |
| 3 | 5 | 2 | $a_2 = 2$ |
| 4 | 3 | 2 | $a_2 = 2$ |
| 5 | 7 | 3 | $a_3 = 5$ or $a_4 = 3$ |
| 6 | 101 | 4 | $a_5 = 7$ |
| 7 | 18 | 4 | $a_5 = 7$ |

Answer: $\max(L) = 4$. One LIS: $[2, 5, 7, 101]$.

---

## 2. The Patience Sorting Algorithm (Combinatorial Algorithms)

### The Problem

Why does maintaining a "tails" array with binary search yield the correct LIS length
in O(n log n)?

### The Formula

Define the tails array $T$ where $T[k]$ is the smallest tail element of all increasing
subsequences of length $k + 1$ found so far.

**Invariant:** $T$ is always strictly increasing: $T[0] < T[1] < \cdots < T[|T|-1]$.

For each new element $a_i$:
- If $a_i > T[\text{last}]$: append $a_i$ (extend longest subsequence)
- Otherwise: find the leftmost $k$ where $T[k] \ge a_i$ and set $T[k] = a_i$

$$k = \min\{j : T[j] \ge a_i\}$$

The binary search runs in $O(\log |T|)$ time per element.

**Correctness proof sketch:** The invariant guarantees that $|T|$ equals the LIS length.
Replacing $T[k]$ with a smaller value never invalidates existing subsequences but opens
the door for future extensions.

### Worked Examples

Array: $[10, 9, 2, 5, 3, 7, 101, 18]$

| Step | $a_i$ | Action | Tails $T$ |
|------|-------|--------|-----------|
| 1 | 10 | Append | [10] |
| 2 | 9 | Replace T[0] | [9] |
| 3 | 2 | Replace T[0] | [2] |
| 4 | 5 | Append | [2, 5] |
| 5 | 3 | Replace T[1] | [2, 3] |
| 6 | 7 | Append | [2, 3, 7] |
| 7 | 101 | Append | [2, 3, 7, 101] |
| 8 | 18 | Replace T[3] | [2, 3, 7, 18] |

Final: $|T| = 4$, so LIS length = 4.

Note: $T = [2, 3, 7, 18]$ happens to be a valid LIS here, but in general $T$ does not
represent an actual subsequence from the input.

---

## 3. Dilworth's Theorem and Chain Decomposition (Order Theory)

### The Problem

What is the dual relationship between increasing and decreasing subsequences?

### The Formula

**Dilworth's Theorem:** In any finite partially ordered set, the maximum length of an
antichain equals the minimum number of chains needed to cover the set.

Applied to sequences: define the partial order $(i, a_i) \le (j, a_j)$ iff $i < j$
and $a_i < a_j$. Then:

- A **chain** = an increasing subsequence
- An **antichain** = a set of pairwise incomparable elements

**Erdos-Szekeres Theorem:** Any sequence of more than $pq$ distinct elements contains
either an increasing subsequence of length $p + 1$ or a decreasing subsequence of
length $q + 1$.

$$n > pq \implies \text{LIS} > p \;\text{ or }\; \text{LDS} > q$$

### Worked Examples

For a sequence of length 5 with LIS = 2 (e.g., $[5, 4, 3, 2, 1]$), Dilworth's theorem
guarantees we need at least $\lceil 5/2 \rceil = 3$ decreasing subsequences to cover
all elements. In fact, LDS = 5 here.

For $[3, 1, 4, 1, 5, 9, 2, 6]$ (length 8):
- By Erdos-Szekeres with $p = q = 2$: since $8 > 4$, there must be an increasing
  subsequence of length 3 or a decreasing one of length 3. Indeed, $[1, 4, 5, 9]$
  is increasing with length 4.

---

## 4. Connection to Young Tableaux (Algebraic Combinatorics)

### The Problem

The patience sorting process has a deep connection to Robinson-Schensted correspondence
and Young tableaux.

### The Formula

The Robinson-Schensted (RS) correspondence is a bijection between permutations
$\sigma \in S_n$ and pairs of standard Young tableaux $(P, Q)$ of the same shape
$\lambda \vdash n$.

**Key result:** Under the RS correspondence, if $\sigma$ maps to shape $\lambda$,
then:

$$\text{LIS}(\sigma) = \lambda_1 \quad\text{(length of first row)}$$

$$\text{LDS}(\sigma) = \lambda_1' \quad\text{(length of first column)}$$

The patience sorting piles correspond exactly to the columns of the insertion
tableau $P$.

### Worked Examples

For the permutation $[2, 1, 3]$:
- RS insertion gives $P = \begin{bmatrix} 1 & 3 \\ 2 \end{bmatrix}$, shape $(2, 1)$
- $\lambda_1 = 2$, confirming LIS = 2 (e.g., $[1, 3]$ or $[2, 3]$)
- $\lambda_1' = 2$, confirming LDS = 2 (e.g., $[2, 1]$)

---

## 5. Expected LIS Length for Random Permutations (Probability Theory)

### The Problem

What is the expected length of the LIS for a uniformly random permutation of
$\{1, 2, \ldots, n\}$?

### The Formula

The Baik-Deift-Johansson theorem (1999) establishes:

$$\frac{L_n - 2\sqrt{n}}{n^{1/6}} \xrightarrow{d} F_2$$

where $L_n$ is the LIS length of a random permutation of length $n$, and $F_2$ is
the Tracy-Widom distribution (from random matrix theory).

For practical purposes:

$$E[L_n] \approx 2\sqrt{n} - 1.77n^{-1/6} + \cdots$$

### Worked Examples

- $n = 100$: $E[L_n] \approx 2\sqrt{100} = 20$
- $n = 10000$: $E[L_n] \approx 2\sqrt{10000} = 200$
- $n = 2500$ (our constraint): $E[L_n] \approx 2\sqrt{2500} = 100$

This means for a random array of 2500 elements, the LIS is typically around 100 --
much shorter than the array itself, but significantly longer than 1.

---

## Prerequisites

- Dynamic programming (optimal substructure, overlapping subproblems)
- Binary search on sorted arrays
- Partial orders, chains, and antichains
- Basic combinatorics and bijections

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Implement the O(n^2) DP solution. Understand why each element starts as a subsequence of length 1. Trace through the recurrence on small examples. |
| **Intermediate** | Implement the O(n log n) patience sort. Understand the tails array invariant and why binary search is correct. Reconstruct the actual subsequence using predecessor tracking. Prove the Erdos-Szekeres theorem. |
| **Advanced** | Study the Robinson-Schensted correspondence and its connection to patience sorting. Understand the Tracy-Widom distribution and the Baik-Deift-Johansson theorem. Extend to 2D problems (Russian Doll Envelopes). |
