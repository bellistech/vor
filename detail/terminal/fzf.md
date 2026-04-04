# The Mathematics of fzf — Fuzzy String Matching & Scoring Algorithms

> *Fuzzy finding is an information retrieval problem: given a query and a corpus of candidates, rank them by a scoring function that balances edit distance, substring alignment, and positional weighting in sublinear time.*

---

## 1. Fuzzy Matching Model (Approximate String Matching)

### Problem Definition

Given a query string $q = q_1 q_2 \ldots q_m$ and a candidate string $s = s_1 s_2 \ldots s_n$, fzf finds the **best subsequence alignment** -- positions $i_1 < i_2 < \ldots < i_m$ in $s$ such that $s_{i_k} = q_k$ for all $k$, maximizing a scoring function.

This is a variant of the **Longest Common Subsequence (LCS)** problem, but with a scoring twist.

### Subsequence vs Substring

A subsequence allows gaps; a substring does not:

| Concept | Query | Candidate | Match? |
|:---|:---|:---|:---:|
| Subsequence | `fzf` | `**f**u**z**zy-**f**inder` | Yes |
| Substring | `fzf` | `fuzzy-finder` | No |
| Substring | `fuz` | `fu**z**zy-finder` | Yes |

fzf matches subsequences but **rewards contiguity** -- consecutive matching characters score higher.

---

## 2. Scoring Function (Dynamic Programming)

### fzf's Smith-Waterman Variant

fzf uses a modified **Smith-Waterman** algorithm adapted for fuzzy finding. The DP recurrence:

$$S(i, j) = \max \begin{cases} S(i-1, j-1) + match\_score(i, j) & \text{if } q_i = s_j \\ S(i, j-1) & \text{skip candidate char} \\ 0 & \text{no match} \end{cases}$$

Where $match\_score$ incorporates bonuses:

$$match\_score(i, j) = base + bonus(j)$$

### Bonus Functions

| Bonus Type | Condition | Value | Rationale |
|:---|:---|:---:|:---|
| Consecutive | $i_{k} = i_{k-1} + 1$ | High | Contiguous runs are stronger matches |
| Word boundary | $s_{j-1} \in \{/, -, \_, .\}$ | High | Matching at word starts is intentional |
| Camel case | $s_j$ is uppercase, $s_{j-1}$ is lowercase | Medium | camelCase word boundaries |
| First character | $j = 0$ | High | Matching at string start |
| Gap penalty | Non-consecutive match | Negative | Penalize scattered matches |

### Gap Penalty Model

For a gap of length $g$ between consecutive matches:

$$penalty(g) = gap\_start + gap\_extension \times (g - 1)$$

This is an **affine gap penalty** -- the same model used in bioinformatics sequence alignment (BLAST, Smith-Waterman).

Total score:

$$Score = \sum_{k=1}^{m} match\_score(k) - \sum_{gaps} penalty(g_i)$$

---

## 3. Algorithmic Complexity (Search & Rank)

### Per-Candidate Matching

The DP table for matching query $q$ (length $m$) against candidate $s$ (length $n$):

$$T_{match} = O(m \times n)$$

$$S_{match} = O(m \times n)$$

In practice, fzf optimizes with early termination and bounded scoring windows.

### Full Pipeline

For $C$ total candidates:

$$T_{total} = T_{read} + T_{filter} + T_{score} + T_{sort}$$

| Phase | Complexity | Notes |
|:---|:---|:---|
| Read candidates | $O(C \times \bar{n})$ | Read all lines, $\bar{n}$ = avg length |
| Filter (subsequence check) | $O(C \times \bar{n})$ | Linear scan per candidate |
| Score (DP alignment) | $O(|matches| \times m \times \bar{n})$ | Only on filtered candidates |
| Sort (rank by score) | $O(|matches| \log |matches|)$ | Top-k extraction |

### Optimization: Early Filter

Before running expensive DP scoring, fzf does a quick **subsequence existence check**:

```
function has_subsequence(q, s):
    j = 0
    for each char c in s:
        if c == q[j]: j++
        if j == len(q): return true
    return false
```

This runs in $O(n)$ and eliminates non-matching candidates before the $O(m \times n)$ scoring phase.

---

## 4. Ranking and Tie-Breaking (Order Statistics)

### Multi-Criteria Sort

fzf ranks results by a tuple:

$$(score, -length, -index)$$

Sorted lexicographically: highest score first, then shortest candidate, then most recent (for history).

### Score Distribution

For a query of length $m$ on random strings of length $n$ with alphabet size $\sigma$:

$$P(\text{subsequence exists}) = 1 - \prod_{k=0}^{m-1} \left(1 - \frac{1}{\sigma}\right)^{n-k}$$

For $m = 3$, $n = 20$, $\sigma = 36$ (alphanumeric):

$$P \approx 1 - \left(\frac{35}{36}\right)^{54} \approx 1 - 0.214 \approx 0.786$$

About 78.6% of random 20-char strings contain any 3-char query as a subsequence -- this is why scoring quality matters more than mere matching.

---

## 5. Interactive Search — Incremental Computation

### Query Extension

When the user types an additional character, fzf can **incrementally update** rather than recompute from scratch:

$$q' = q + c$$

Only candidates that matched $q$ need to be checked for $q'$:

$$candidates(q') \subseteq candidates(q)$$

This gives an effective speedup:

$$T_{incremental} = O(|matches(q)| \times m' \times \bar{n})$$

### Query Deletion

When a character is deleted, fzf must widen the candidate set -- it falls back to the last cached state for the shorter prefix.

---

## 6. Comparison with Edit Distance Approaches

### Levenshtein Distance

Classical fuzzy matching uses Levenshtein edit distance:

$$d(a, b) = \min \text{ insertions + deletions + substitutions to transform } a \to b$$

DP complexity: $O(m \times n)$

### Why fzf Does NOT Use Edit Distance

| Property | Edit Distance | fzf Subsequence |
|:---|:---|:---|
| Model | Transform $q$ into $s$ | Find $q$ within $s$ |
| Query length | Must be similar to candidate | Can be much shorter |
| Gap handling | Counts as edits | Affine penalty (tolerant) |
| Typing model | Correcting typos | Abbreviating names |
| Use case | Spell checking | File path selection |

fzf's model matches human behavior: users type abbreviated prefixes (`mgo` for `main.go`), not misspelled full names.

---

## 7. Parallelism and Throughput

### Multi-threaded Architecture

fzf uses a **producer-consumer** model:

$$throughput = \min(T_{read}, C_{threads} \times T_{match\_per\_thread})$$

| Component | Threading | Notes |
|:---|:---|:---|
| Input reader | Single thread | Sequential I/O |
| Matcher/scorer | Multi-threaded | Work-stealing pool |
| Renderer | Single thread | Terminal output |

### Throughput Benchmarks (Approximate)

For 1 million file paths (~50 bytes average):

$$T_{read} \approx \frac{50 \times 10^6}{disk\_bw} \approx 25\text{ms (NVMe)}$$

$$T_{filter} \approx \frac{10^6 \times 50}{10^9} \approx 50\text{ms (single core)}$$

$$T_{total} \approx 50\text{ms on 4 cores} \approx 15\text{ms}$$

Interactive response well within the 100ms perceptual threshold.

---

## 8. Information-Theoretic View

### Query Entropy

Each character typed narrows the candidate set. The information gained per keystroke:

$$I(c) = \log_2 \frac{|candidates(q)|}{|candidates(q + c)|}$$

On average, with alphabet size $\sigma = 36$:

$$I_{avg} \approx \log_2 \sigma \approx 5.2 \text{ bits per character}$$

After $m$ characters:

$$|candidates| \approx C \times \sigma^{-m}$$

For $C = 100{,}000$ files and $m = 3$: $\approx 100{,}000 / 36^3 \approx 2$ candidates remain. Three keystrokes typically suffice to uniquely identify a file.

---

## Prerequisites

- dynamic programming, subsequence algorithms, string matching, affine gap penalties, information theory, edit distance

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Subsequence check | $O(n)$ | $O(1)$ |
| DP scoring (per candidate) | $O(m \times n)$ | $O(m \times n)$ |
| Full filter pass | $O(C \times \bar{n})$ | $O(C)$ |
| Sort results | $O(k \log k)$ | $O(k)$ |
| Incremental update | $O(k' \times m \times \bar{n})$ | $O(k')$ |

---

*Fuzzy finding is approximate string matching tuned for human intent -- the scoring function encodes the assumption that you are abbreviating, not misspelling, and rewards the alignment patterns that real users produce.*
