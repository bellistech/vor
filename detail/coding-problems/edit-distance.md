# The Mathematics of Edit Distance -- Metric Spaces and Sequence Alignment

> *The minimum number of atomic changes to transform one string into another defines a true metric on the space of all strings, connecting information theory, computational biology, and the algebra of string operations.*

---

## 1. The Wagner-Fischer Recurrence (Dynamic Programming)

### The Problem

Derive the recurrence relation for edit distance and prove its correctness via optimal
substructure.

### The Formula

Let $d(i, j)$ denote the edit distance between the first $i$ characters of string $s$
and the first $j$ characters of string $t$.

$$d(i, j) = \begin{cases} i & \text{if } j = 0 \\ j & \text{if } i = 0 \\ d(i-1, j-1) & \text{if } s_i = t_j \\ 1 + \min\begin{cases} d(i-1, j) & \text{(delete)} \\ d(i, j-1) & \text{(insert)} \\ d(i-1, j-1) & \text{(replace)} \end{cases} & \text{otherwise} \end{cases}$$

### Worked Examples

Computing $d(\text{"horse"}, \text{"ros"})$:

|   | "" | r | o | s |
|---|-----|---|---|---|
| "" | 0 | 1 | 2 | 3 |
| h | 1 | 1 | 2 | 3 |
| o | 2 | 2 | 1 | 2 |
| r | 3 | 2 | 2 | 2 |
| s | 4 | 3 | 3 | 2 |
| e | 5 | 4 | 4 | 3 |

Answer: $d(5, 3) = 3$. The three operations:
1. Replace 'h' with 'r': horse -> rorse
2. Delete second 'r': rorse -> rose
3. Delete 'e': rose -> ros

---

## 2. Edit Distance as a Metric (Metric Space Theory)

### The Problem

Prove that the Levenshtein distance satisfies the axioms of a metric space.

### The Formula

A metric $d$ on a set $\Sigma^*$ (all strings over alphabet $\Sigma$) must satisfy:

1. **Non-negativity:** $d(s, t) \ge 0$
2. **Identity:** $d(s, t) = 0 \iff s = t$
3. **Symmetry:** $d(s, t) = d(t, s)$
4. **Triangle inequality:** $d(s, u) \le d(s, t) + d(t, u)$

**Proof of symmetry:** Every insert in the forward direction corresponds to a delete
in the reverse, and vice versa. Replace is symmetric. Therefore the minimum cost
sequence from $s \to t$ can be reversed to give a sequence from $t \to s$ of equal cost.

**Proof of triangle inequality:** Given optimal edit sequences $s \to t$ (cost $a$) and
$t \to u$ (cost $b$), concatenating them gives a valid (not necessarily optimal) sequence
$s \to u$ of cost $a + b$. Since $d(s, u)$ is the minimum, $d(s, u) \le a + b$.

### Worked Examples

- $d(\text{"kitten"}, \text{"sitting"}) = 3$
- $d(\text{"sitting"}, \text{"kitten"}) = 3$ (symmetry)
- $d(\text{"kitten"}, \text{""}) = 6$, $d(\text{""}, \text{"sitting"}) = 7$
- Triangle: $d(\text{"kitten"}, \text{"sitting"}) = 3 \le 6 + 7 = 13$ (satisfied)

---

## 3. Space Optimization via Rolling Arrays (Algorithm Engineering)

### The Problem

Reduce the space complexity from $O(mn)$ to $O(\min(m, n))$ while preserving correctness.

### The Formula

At step $i$, cell $d(i, j)$ depends only on three values:
- $d(i-1, j)$ -- directly above (previous row, same column)
- $d(i, j-1)$ -- directly left (current row, previous column)
- $d(i-1, j-1)$ -- diagonal (previous row, previous column)

Therefore only two rows are needed: $\text{prev}[0..n]$ and $\text{curr}[0..n]$.

$$\text{curr}[j] = \begin{cases} \text{prev}[j-1] & \text{if } s_i = t_j \\ 1 + \min(\text{prev}[j],\; \text{curr}[j-1],\; \text{prev}[j-1]) & \text{otherwise} \end{cases}$$

After processing row $i$, swap: $\text{prev} \leftarrow \text{curr}$.

### Worked Examples

For $d(\text{"abc"}, \text{"abc"})$:
- Init: prev = [0, 1, 2, 3]
- Row 1 ('a'): curr = [1, 0, 1, 2] (diagonal match at j=1)
- Row 2 ('b'): curr = [2, 1, 0, 1] (diagonal match at j=2)
- Row 3 ('c'): curr = [3, 2, 1, 0] (diagonal match at j=3)
- Answer: prev[3] = 0

---

## 4. Connection to Longest Common Subsequence (String Algorithms)

### The Problem

Edit distance and LCS (longest common subsequence) are intimately related. Express one
in terms of the other.

### The Formula

For strings $s$ of length $m$ and $t$ of length $n$, with only insertions and deletions
allowed (no replacements), the edit distance equals:

$$d_{\text{indel}}(s, t) = m + n - 2 \cdot \text{LCS}(s, t)$$

For the full Levenshtein distance (with replacements), the relationship is:

$$d(s, t) \le m + n - 2 \cdot \text{LCS}(s, t)$$

with equality when every non-matching character is handled by insert/delete rather than
replacement.

### Worked Examples

$s = \text{"horse"}$, $t = \text{"ros"}$:
- LCS = "os" (length 2), or "rs" (length 2)
- $d_{\text{indel}} = 5 + 3 - 2(2) = 4$
- Actual $d = 3$ (replace is cheaper than delete + insert)

The difference $4 - 3 = 1$ represents one replacement used instead of a delete + insert pair.

---

## 5. Applications in Computational Biology (Bioinformatics)

### The Problem

Edit distance is the foundation of biological sequence alignment. How does the
Needleman-Wunsch algorithm generalize Wagner-Fischer?

### The Formula

Needleman-Wunsch uses a scoring matrix $\sigma(a, b)$ for substitution costs and a
gap penalty $g$ for insertions/deletions:

$$F(i, j) = \max\begin{cases} F(i-1, j-1) + \sigma(s_i, t_j) \\ F(i-1, j) + g \\ F(i, j-1) + g \end{cases}$$

When $\sigma(a, b) = 0$ if $a = b$ and $-1$ otherwise, and $g = -1$, the negated
alignment score equals the Levenshtein distance:

$$d(s, t) = -F(m, n)$$

Affine gap penalties ($g_{\text{open}} + k \cdot g_{\text{extend}}$) model biological
reality better but increase algorithmic complexity.

### Worked Examples

DNA alignment of AGCT and ACT:
- Match score = +1, mismatch = -1, gap = -2

| | "" | A | C | T |
|---|---|---|---|---|
| "" | 0 | -2 | -4 | -6 |
| A | -2 | 1 | -1 | -3 |
| G | -4 | -1 | 0 | -2 |
| C | -6 | -3 | 0 | -1 |
| T | -8 | -5 | -2 | 1 |

Optimal alignment: A-GCT / A-_CT (gap in position 2), score = 1.

---

## Prerequisites

- 2D dynamic programming tables
- String operations (insert, delete, replace)
- Metric space axioms
- Longest common subsequence (LCS)
- Basic linear algebra (for the matrix interpretation)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Build and fill the full DP table. Trace the solution path. Understand the three operations and their positions in the table (left, above, diagonal). |
| **Intermediate** | Implement space-optimized version. Prove edit distance is a metric. Relate to LCS. Extend to weighted edit distances. Backtrack to find the actual edit sequence. |
| **Advanced** | Study Needleman-Wunsch and Smith-Waterman for biological sequence alignment. Analyze Ukkonen's optimization for bounded edit distance. Explore edit distance automata for approximate string matching. |
