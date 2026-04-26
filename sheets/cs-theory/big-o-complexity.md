# Big-O & Complexity Analysis

Asymptotic analysis, growth rates, and the canonical complexity tables for every common data structure, algorithm, and operation.

## Setup

Asymptotic analysis is the language of "how does runtime scale with input size?" — the part of an algorithm's behavior that survives multiplication by constants and addition of lower-order terms. We strip away the implementation-specific noise (CPU model, language, compiler optimization level, cache topology) and ask how the work grows as the input grows without bound.

The variable `n` is the **input size**. What "size" means depends on the problem:

- For an array, `n` is the number of elements.
- For a graph, we use `V` (vertices) and `E` (edges) — sometimes `n` for `V`.
- For a string, `n` is the number of characters; `m` is often a pattern length.
- For matrices, `n` is the side length (so the matrix has `n²` entries).
- For numbers, `n` may mean the value itself OR the number of bits to represent it — this is the source of the famous "pseudo-polynomial" trap (e.g., subset-sum is "polynomial" in target value but exponential in bit-length).

```bash
# Always state the size variable explicitly
# "Sort an array of n integers" — n is array length
# "Factor an integer N" — N is the value, log N is the bit-length
# "BFS a graph G=(V,E)" — work is in terms of V + E
```

The phrase **"as n → ∞"** is core. We do not care what an algorithm does for `n = 5`. We care about the dominant growth term as `n` becomes arbitrarily large. An O(n²) algorithm beats an O(n) one for small inputs all the time — what asymptotic notation tells us is that there exists *some* threshold beyond which O(n) wins, and stays winning forever.

### Resource axes

Most analysis focuses on **running time** (work, operations), but other axes matter:

| Axis | What is being counted | Typical model |
|---|---|---|
| Running time | Elementary RAM operations (add, compare, load, store) | Word-RAM |
| Space complexity | Memory cells used (auxiliary or total) | Word-RAM |
| Network complexity | Messages or rounds in a distributed protocol | Message-passing |
| Query complexity | Oracle calls or comparisons (not other work) | Decision tree, comparison tree |
| Cache complexity | Cache misses (block transfers) | External-memory / cache-oblivious |
| Communication complexity | Bits exchanged between parties | Yao's model |
| Circuit complexity | Gate count or depth | Boolean circuits |

```bash
# Time is what most people mean by "complexity"
# Space matters when you're bounded by RAM or doing recursion
# Network matters in distributed systems and DB joins
# Query matters in lower-bound proofs (Ω(n log n) sort)
```

## Notation Family

Asymptotic notation is a family of comparisons between functions. Let `f, g : ℕ → ℝ⁺`.

### Big-O — upper bound

```
f(n) = O(g(n))   ⇔   ∃ c > 0, n₀ ≥ 0 :  0 ≤ f(n) ≤ c · g(n)   ∀ n ≥ n₀
```

"Eventually, f is dominated by a constant multiple of g."

### Big-Omega — lower bound

```
f(n) = Ω(g(n))   ⇔   ∃ c > 0, n₀ ≥ 0 :  0 ≤ c · g(n) ≤ f(n)   ∀ n ≥ n₀
```

"Eventually, f is at least a constant multiple of g."

### Big-Theta — tight bound

```
f(n) = Θ(g(n))   ⇔   f(n) = O(g(n))   AND   f(n) = Ω(g(n))
```

"f and g grow at the same rate up to constants."

### Little-o — strict upper bound

```
f(n) = o(g(n))   ⇔   ∀ c > 0,  ∃ n₀ ≥ 0 :  0 ≤ f(n) < c · g(n)   ∀ n ≥ n₀
                  ⇔   lim (n→∞) f(n)/g(n) = 0
```

"f grows *strictly* slower than g." Note the `< c·g(n)` rather than `≤ c·g(n)`, and that `c` is *any* positive constant — not just a particular one.

### Little-omega — strict lower bound

```
f(n) = ω(g(n))   ⇔   ∀ c > 0,  ∃ n₀ ≥ 0 :  f(n) > c · g(n)   ∀ n ≥ n₀
                  ⇔   lim (n→∞) f(n)/g(n) = ∞
```

"f grows strictly faster than g."

### Quick relationships

| Relation | Loose analogy |
|---|---|
| f = O(g) | f ≤ g |
| f = Ω(g) | f ≥ g |
| f = Θ(g) | f = g |
| f = o(g) | f < g |
| f = ω(g) | f > g |

```bash
# Examples
# 3n²        = O(n²)        Θ(n²)         o(n³)         ω(n)         Ω(n²)
# n          = O(n²)                       o(n²)                       Ω(log n)
# n log n    = O(n²)                       o(n²)         ω(n)
# 100n + 5n² = Θ(n²)                       o(n³)         ω(n)
# 2^n        = ω(n^k) for every k          (exponential beats every polynomial)
```

### The "abuse of notation" disclaimer

`f(n) = O(g(n))` is not equality. The right-hand side is a **set of functions**; the equals sign is shorthand for "is an element of":

```
O(g(n)) := { f : ℕ → ℝ⁺ | ∃ c, n₀ such that 0 ≤ f(n) ≤ c·g(n) ∀ n ≥ n₀ }

f(n) = O(g(n))   really means   f(n) ∈ O(g(n))
```

This is why these are *not* symmetric. `O(n) = O(n²)` is true (every linear function is also at most quadratic). `O(n²) = O(n)` is false. Reading the equals sign as English's "equals" leads to nonsense like `n = O(n²) = n³` ("therefore n = n³").

```bash
# Good practice: when in doubt, read the = as "is"
# "f(n) is O(n²)" rather than "f(n) equals O(n²)"
```

## Common Pitfalls

### Big-O is about growth rate, not performance

```bash
# An O(1) operation can be slower than an O(n²) operation for n = 10
# Hash table O(1) lookup may be 100ns; tiny array linear scan may be 5ns
# Always profile real workloads at realistic n
```

### Cache effects can dominate for small n

A linear scan of a `std::vector<int>` of 1000 elements in cache can outperform a hash-table lookup that misses cache. The RAM model assumes uniform memory access cost — false on every modern CPU.

### Constants matter

```bash
# 100n vs n² — for n = 50, 100n = 5000 < n² = 2500? No — 100·50 = 5000, 50² = 2500
# Crossover is where 100n = n², i.e. n = 100
# Below n = 100, the "asymptotically slower" n² is faster
# Below the crossover, the constant wins
```

```bash
# 1000n log n vs n² — crossover when 1000 log₂ n = n
# For n = 14000, n² ≈ 2·10⁸, 1000 n log n ≈ 1.4·10⁸ → log-linear wins above ~14k
```

### "n is large" is an assumption

If your `n` is bounded by a small constant (say, `n ≤ 50`), every algorithm is O(1) — the difference is the constant factor. Asymptotic analysis becomes useless.

### Amortized vs average vs worst case

These are *different things*, and conflating them is a top-tier mistake.

| Type | Meaning |
|---|---|
| **Worst case** | Maximum cost over all inputs of size n. Adversary picks the input. |
| **Best case** | Minimum cost over all inputs of size n. Rarely useful. |
| **Average case** | Expected cost over a *probability distribution* on inputs. Distribution must be specified. |
| **Amortized** | Average cost per operation over a *sequence* of operations, worst case over the sequence. No randomness. |

```bash
# Hash table insert
#   worst case:    O(n) — all keys collide
#   average case:  O(1) — under uniform-hashing assumption
#   amortized:     O(1) — when including occasional rehash
```

```bash
# Quicksort
#   worst case:    O(n²) — sorted input + first-element pivot
#   average case:  O(n log n) — over uniform random permutations
#   amortized:     not a meaningful concept for a one-shot algorithm
```

```bash
# Dynamic array push_back
#   worst case:    O(n) — the call that triggers reallocation
#   amortized:     O(1) — over a sequence of n pushes the total is O(n)
```

## Growth-Rate Ladder

Sorted from slowest-growing (best) to fastest-growing (worst).

| Class | Name | Example algorithms | Intractable beyond ~ |
|---|---|---|---|
| O(1) | Constant | Hash lookup, array index, stack push | n = anything |
| O(α(n)) | Inverse-Ackermann | Union-Find with path compression + union-by-rank | n = 10^80 (universe) |
| O(log log n) | Double-log | Van Emde Boas, interpolation search on uniform | n = 10^18 |
| O(log n) | Logarithmic | Binary search, balanced BST op | n = 10^18 |
| O(log² n) | Polylog | Some range queries, segment-tree LCA | 10^9 |
| O(√n) | Square-root | Mo's algorithm, square-root decomposition | 10^14 |
| O(n) | Linear | Single pass, hash build, BFS | 10^9 |
| O(n log n) | Linearithmic | Mergesort, FFT, comparison-sort lower bound | 10^7-10^8 |
| O(n √n) | n^1.5 | Some sqrt-decomp queries on length n | 10^6 |
| O(n²) | Quadratic | Bubble sort, naive matrix-vector, simple DP | 10^4 |
| O(n^2.373) | Sub-cubic matrix | Matrix multiplication (galactic) | 10^3 |
| O(n³) | Cubic | Naive matrix multiply, Floyd-Warshall | 500 |
| O(n^k) | Polynomial | k-clique check (k-fixed), DP with k indices | depends on k |
| O(2^n) | Exponential | Brute-force subset, naive Hamilton | n ≈ 25-30 |
| O(n!) | Factorial | Brute-force TSP, generate-all-permutations | n ≈ 11 |
| O(n^n) | Self-power | Naive enumeration with replacement | n ≈ 8 |

### Concrete operation counts

Rough estimate at "1 second of CPU = 10^8 ops". Anything taking > 10^9 ops is "minutes"; > 10^11 is "hours"; > 10^13 is "days+".

| n | log n | √n | n | n log n | n² | n³ | 2^n | n! |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 10 | 3 | 3 | 10 | 33 | 100 | 1k | 1k | 3.6M |
| 20 | 4 | 4 | 20 | 86 | 400 | 8k | 1M | 2.4·10^18 |
| 30 | 5 | 5 | 30 | 147 | 900 | 27k | 10^9 | huge |
| 50 | 6 | 7 | 50 | 282 | 2.5k | 125k | 10^15 | huge |
| 100 | 7 | 10 | 100 | 664 | 10k | 10^6 | 10^30 | huge |
| 1k | 10 | 32 | 1k | 10k | 10^6 | 10^9 | huge | huge |
| 10k | 13 | 100 | 10k | 130k | 10^8 | 10^12 | huge | huge |
| 100k | 17 | 316 | 100k | 1.7M | 10^10 | huge | huge | huge |
| 10^6 | 20 | 1k | 10^6 | 2·10^7 | 10^12 | huge | huge | huge |
| 10^7 | 23 | 3.2k | 10^7 | 2.3·10^8 | huge | huge | huge | huge |
| 10^9 | 30 | 31k | 10^9 | 3·10^10 | huge | huge | huge | huge |

```bash
# Practical 1-second budgets (rule of thumb)
#   O(log n)   — n = 10^18+ (essentially unbounded)
#   O(n)       — n ≤ 10^8
#   O(n log n) — n ≤ 5·10^7
#   O(n √n)    — n ≤ 10^5
#   O(n²)      — n ≤ 10^4
#   O(n³)      — n ≤ 500
#   O(2^n)     — n ≤ 25
#   O(n!)      — n ≤ 11
```

## Master Theorem

For divide-and-conquer recurrences:

```
T(n) = a · T(n / b) + f(n)        (a ≥ 1, b > 1)
```

- `a` = number of subproblems
- `n/b` = subproblem size
- `f(n)` = work done at this level (split + combine)

Define the **critical exponent**: `c* = log_b(a)`.

### The three cases

| Case | Condition on f(n) | Result |
|---|---|---|
| 1 | f(n) = O(n^(c* − ε)) for some ε > 0 | T(n) = Θ(n^c*) |
| 2 | f(n) = Θ(n^c* · log^k n), k ≥ 0 | T(n) = Θ(n^c* · log^(k+1) n) |
| 3 | f(n) = Ω(n^(c* + ε)) for some ε > 0, AND a·f(n/b) ≤ c·f(n) for some c < 1 (regularity) | T(n) = Θ(f(n)) |

Case 1 — leaves dominate. Case 2 — work balanced; an extra log factor. Case 3 — root dominates.

### Worked examples

#### Merge sort

```
T(n) = 2 T(n/2) + Θ(n)
a = 2, b = 2, c* = log₂ 2 = 1
f(n) = Θ(n) = Θ(n^1) = Θ(n^c*)        → Case 2 with k = 0
T(n) = Θ(n · log n)
```

#### Binary search

```
T(n) = T(n/2) + Θ(1)
a = 1, b = 2, c* = log₂ 1 = 0
f(n) = Θ(1) = Θ(n^0) = Θ(n^c*)         → Case 2 with k = 0
T(n) = Θ(log n)
```

#### Karatsuba multiplication

```
T(n) = 3 T(n/2) + Θ(n)
a = 3, b = 2, c* = log₂ 3 ≈ 1.585
f(n) = Θ(n) = O(n^(c* − ε)) for ε ≈ 0.5  → Case 1
T(n) = Θ(n^log₂ 3) ≈ Θ(n^1.585)
```

#### Strassen matrix multiplication

```
T(n) = 7 T(n/2) + Θ(n²)
a = 7, b = 2, c* = log₂ 7 ≈ 2.807
f(n) = Θ(n²) = O(n^(c* − ε))            → Case 1
T(n) = Θ(n^log₂ 7) ≈ Θ(n^2.807)
```

#### Naive matrix multiplication (recursive 8-block)

```
T(n) = 8 T(n/2) + Θ(n²)
a = 8, b = 2, c* = log₂ 8 = 3
f(n) = Θ(n²) = O(n^(c* − ε))            → Case 1
T(n) = Θ(n^3)
```

#### Recurrence that does NOT fit (gap case)

```
T(n) = 2 T(n/2) + n log n
a = 2, b = 2, c* = 1, f(n) = n log n
n log n is between n^1 and n^(1+ε)      → no case applies
Use Akra-Bazzi or recursion tree:  T(n) = Θ(n · log² n)
```

```bash
# Akra-Bazzi handles non-equal-size subproblems
# T(n) = Σ aᵢ T(bᵢ n + h(n)) + g(n)
# Solution: T(n) = Θ(n^p · (1 + ∫ g(u) / u^(p+1) du))
# where Σ aᵢ bᵢ^p = 1
```

## Amortized Analysis

A single operation may be expensive, but a sequence of operations averages out. **Amortized** ≠ **average-case**: amortized is worst-case-over-sequence, no randomness involved.

### Aggregate method

Sum the total cost of `n` operations and divide by `n`.

```bash
# Dynamic array of capacity 1, doubling on overflow
# Sequence of n push_back operations
# Cost of i-th push: 1 (write) + (i if a doubling happens at step i, else 0)
# Doublings happen at i = 1, 2, 4, 8, ..., ≤ n
# Total copy cost = 1 + 2 + 4 + ... + n/2 + n ≤ 2n
# Total push cost ≤ n + 2n = 3n
# Amortized per push: O(3) = O(1)
```

### Accounting method

Charge each operation more than its actual cost; bank the surplus to pay for expensive future operations.

```bash
# Stack with multipop(k) — pop up to k elements
# Charge $2 per push: $1 for the push, $1 saved on the element
# Each element carries $1 with it
# When multipop pops the element, the $1 already-saved pays for it
# Amortized push: O(2) = O(1); amortized multipop: O(1)
```

### Potential method

Define a potential function `Φ : state → ℝ≥0` measuring "stored work."
Amortized cost = actual cost + ΔΦ.

```bash
# Dynamic array, Φ(state) = 2·size − capacity   (≥ 0 maintained)
# Push without resize:  actual = 1, Δsize = 1, Δcap = 0, ΔΦ = 2
#   amortized = 1 + 2 = 3 = O(1)
# Push with resize (size = cap before):  actual = size + 1
#   after: size' = size+1, cap' = 2·cap = 2·size
#   ΔΦ = (2(size+1) − 2 size) − (2 size − size) = 2 − size
#   amortized = (size + 1) + (2 − size) = 3 = O(1)
```

### Fibonacci heap

| Operation | Amortized |
|---|---|
| insert | O(1) |
| find-min | O(1) |
| union | O(1) |
| decrease-key | O(1) |
| delete-min | O(log n) |
| delete | O(log n) |

The amortized O(1) decrease-key is what makes Dijkstra `O(V log V + E)`.

### Splay tree

Self-adjusting BST. Amortized O(log n) per operation; some sequences (sequential access) achieve O(1) amortized via the **dynamic-finger** and **working-set** theorems.

## Space Complexity

Like time, space is measured asymptotically. Two flavours:

| Flavour | What is counted |
|---|---|
| **Total space** | Input + auxiliary + output |
| **Auxiliary space** | Everything beyond input/output |

A "linear-space" algorithm typically means O(n) auxiliary space.

### In-place algorithms

"In-place" usually means O(1) or O(log n) auxiliary space.

| Algorithm | Auxiliary |
|---|---|
| Heap sort | O(1) — true in-place |
| Quick sort | O(log n) — recursion stack |
| Merge sort (textbook) | O(n) — temp buffer |
| Merge sort (in-place variant) | O(1) — but slower constants |
| Selection / insertion / bubble sort | O(1) |
| Counting sort | O(k) — k = key range |

### Recursion stack

Recursion takes stack space proportional to call depth.

```bash
# Naive Fibonacci: T(n) = O(2^n) time, O(n) stack space
def fib(n):
    if n < 2: return n
    return fib(n-1) + fib(n-2)
```

```bash
# Tail-recursive sum: O(n) stack unless TCO
# Iterative version: O(1) stack
```

### Iterative vs recursive memory

| Recursive form | Stack frames |
|---|---|
| Quicksort partition recursion | O(log n) avg, O(n) worst |
| Mergesort | O(log n) |
| Naive recursion on linked list | O(n) — risk of stack overflow |
| In-order BST traversal (recursive) | O(h) where h = tree height |
| In-order BST traversal (Morris) | O(1) — uses tree pointers |

## Worst vs Average vs Best

### Quicksort: a textbook example

| Pivot strategy | Worst | Average | Best |
|---|---|---|---|
| First / last element | O(n²) | O(n log n) | O(n log n) |
| Random | O(n²) (with prob 1/n!) | O(n log n) expected | O(n log n) |
| Median-of-three | O(n²) | O(n log n) | O(n log n) |
| BFPRT median-of-medians | O(n log n) | O(n log n) | O(n log n) |

```bash
# The randomized version's expected O(n log n) is over the random choices,
# regardless of the input. The classical "average case" needs assumption
# of input distribution.
```

### Hash table

| Scheme | Worst | Average (uniform hashing) |
|---|---|---|
| Chaining | O(n) | O(1 + α) where α = n/m load factor |
| Open addressing (linear probe) | O(n) | O(1 / (1 − α)) for unsuccessful |
| Cuckoo hashing | O(n) for insert (rehash) | O(1) worst-case lookup |
| Perfect hashing (static) | O(1) — actual worst! | O(1) |

### Other examples

| Algorithm | Worst | Average | Best |
|---|---|---|---|
| Insertion sort | O(n²) | O(n²) | O(n) |
| Merge sort | O(n log n) | O(n log n) | O(n log n) |
| BST search | O(n) | O(log n) | O(1) |
| Bloom filter lookup | O(k) | O(k) | O(k) |
| Linear search | O(n) | O(n) | O(1) |

## Cache Complexity

The flat **RAM model** assumes every memory access is O(1) and identical. Real machines have a memory hierarchy: L1 (≈1 ns), L2 (≈3 ns), L3 (≈10 ns), DRAM (≈100 ns), SSD (≈100 µs), disk (≈10 ms). Adjacent levels differ by 1-2 orders of magnitude.

### External-memory model

Two parameters:

- `M` = words in cache
- `B` = words per cache line / disk block

Cost = number of block transfers between cache and "memory". Operations on data already in cache are free.

| Algorithm | RAM-model | Cache-model |
|---|---|---|
| Scan | Θ(n) | Θ(n / B) |
| Binary search (random access) | Θ(log n) | Θ(log(n / B)) |
| Sort (mergesort / btree-sort) | Θ(n log n) | Θ((n/B) · log_(M/B)(n/B)) |
| Permutation | Θ(n) | Θ(min(n, (n/B) log_(M/B)(n/B))) |
| Matrix multiply (n×n, naive) | Θ(n³) | Θ(n³ / B) |
| Matrix multiply (blocked, b = √M) | Θ(n³) | Θ(n³ / (B · √M)) |

### Cache-oblivious algorithms (CO)

Algorithms with optimal `(M, B)` performance *without* knowing M or B. Achieved by recursive divide-and-conquer that creates problems of all sizes.

```bash
# Cache-oblivious matrix multiply
#   recursively split A, B, C into 4 sub-blocks
#   base case fits in cache → automatic blocking at every cache level
#   Θ(n³ / (B · √M)) cache misses, optimal
```

| CO algorithm | Cache complexity |
|---|---|
| Funnelsort | Θ((n/B) · log_(M/B)(n/B)) |
| van Emde Boas tree layout | Θ(log_B n) |
| CO matrix transpose | Θ(n²/B) |

## Cellular Automata / Streaming

Sublinear models — when even reading the input is too expensive.

### Streaming model

Single pass, limited memory (often `O(polylog(n))` or `O(√n)`). No re-read.

```bash
# Standard problems
#   F₀ — number of distinct elements (cardinality)
#   F₁ — total count = n
#   F₂ — sum of squared frequencies (variance / second moment)
#   Fₖ — k-th frequency moment
#   Heavy hitters — elements with frequency > threshold
```

| Problem | Algorithm | Space |
|---|---|---|
| Distinct count (ε, δ) | HyperLogLog | O((1/ε²) log log n + log n) |
| Median (approx) | Munro-Paterson | O(p · n^(1/p)) for p passes |
| Heavy hitters | Misra-Gries | O(k) for top-k, ε = 1/k |
| Frequency estimate | Count-Min Sketch | O((1/ε) log(1/δ) · log n) |
| Second moment F₂ | AMS | O((1/ε²) log(1/δ) · log n) |

### Sketching

Linear data structures supporting `merge(s, s')`. Mergeability matters for distributed and streaming.

| Sketch | Estimates | Mergeable |
|---|---|---|
| HyperLogLog | Cardinality | Yes |
| Count-Min | Frequency (over-estimate) | Yes |
| Count-Sketch | Frequency (unbiased) | Yes |
| t-digest | Quantiles | Yes |
| KLL | Quantiles | Yes |

## Arrays

Fixed-size contiguous block; the canonical "raw" data structure.

| Operation | Complexity | Notes |
|---|---|---|
| access by index | O(1) | a[i] = base + i·sizeof |
| update by index | O(1) | |
| linear search | O(n) | unsorted |
| binary search | O(log n) | sorted |
| insert at end (fixed array) | impossible | full → must reallocate |
| insert at end (dynamic array) | O(1) amortized | doubling |
| insert at index k | O(n - k) | shift the tail |
| insert at start | O(n) | shift everything |
| delete at end | O(1) | |
| delete at index k | O(n - k) | shift the tail |
| delete at start | O(n) | shift everything |
| concatenate | O(n + m) | |
| slice (read-only view) | O(1) | language-dependent |
| slice (copy) | O(k) | k = slice length |
| reverse | O(n) | |
| min / max (unsorted) | O(n) | |
| sort | O(n log n) | comparison-based |

### Dynamic array (vector / ArrayList / list)

Automatic resizing: typically double when full.

```bash
# C++ std::vector, Java ArrayList, Python list, Go slice, Rust Vec
# Growth factor 2 (or 1.5) — gives amortized O(1) push_back
# Shrinking: usually NEVER shrinks automatically; or shrinks at 1/4 occupancy
```

```bash
# Why factor must be > 1 (and most use exactly 2 or 1.5):
#   factor = 2 → amortized 3·n total work; clean math
#   factor = 1.5 → reuses freed memory more easily (allocator-friendly)
#   factor < ~1.2 → amortized cost grows; gets non-constant
```

## Linked List Singly

Each node holds value + next pointer.

| Operation | Complexity (singly) | Complexity (doubly) |
|---|---|---|
| access by index | O(n) | O(n) |
| search | O(n) | O(n) |
| insert at head | O(1) | O(1) |
| insert at tail (with tail ptr) | O(1) | O(1) |
| insert at tail (no tail ptr) | O(n) | O(n) — but kept ptr |
| insert after node ref | O(1) | O(1) |
| insert before node ref | O(n) | O(1) |
| delete at head | O(1) | O(1) |
| delete at tail (no prev ptr) | O(n) | O(1) |
| delete by node ref | O(n) — find prev | O(1) |
| delete by value | O(n) | O(n) |
| reverse | O(n) | O(n) |
| concatenate (with tail ptr) | O(1) | O(1) |
| length (cached) | O(1) | O(1) |
| length (uncached) | O(n) | O(n) |

```bash
# Singly: cheap memory (one pointer per node), but no backward traversal
# Doubly: prev pointer doubles overhead but enables O(1) delete-by-ref
# Java LinkedList = doubly, Python deque = doubly with block links
# C++ std::forward_list = singly, std::list = doubly
```

```bash
# Cache behaviour: lists are TERRIBLE for cache
# Each .next is a pointer chase to a near-random heap address
# Vector beats list for sequential access in practice
# Linus rant: "if you think linked list is the answer, you don't understand the problem"
```

## Stack

LIFO. Two implementations: array-backed (resizable) or linked list.

| Operation | Array-backed | Linked-list |
|---|---|---|
| push | O(1) amortized | O(1) worst |
| pop | O(1) amortized | O(1) worst |
| peek | O(1) | O(1) |
| size | O(1) | O(1) |
| search (not standard) | O(n) | O(n) |

```bash
# Array-backed: tighter constants, cache-friendly, amortized only because of resize
# Linked-list: never resizes, true O(1) worst-case, but pointer-chasing overhead
# In practice: array-backed wins on every modern CPU for normal sizes
```

## Queue

FIFO. Implementations: circular array, linked list, two stacks.

| Operation | Circular array | Linked list | Two-stack queue |
|---|---|---|---|
| enqueue | O(1) amortized | O(1) | O(1) amortized |
| dequeue | O(1) | O(1) | O(1) amortized |
| peek (front) | O(1) | O(1) | O(1) amortized |
| size | O(1) | O(1) | O(1) |
| search | O(n) | O(n) | O(n) |

### Deque (double-ended queue)

| Operation | Complexity |
|---|---|
| push_front | O(1) |
| push_back | O(1) |
| pop_front | O(1) |
| pop_back | O(1) |
| peek_front | O(1) |
| peek_back | O(1) |
| index access | O(1) (block-based) or O(n) (LL) |

```bash
# Python collections.deque — block-doubly-linked, all ops O(1)
# C++ std::deque — array-of-arrays, O(1) ends, O(1) random access
# Go: no built-in; channel buffer or container/list
```

### Priority queue

Almost always backed by a binary heap.

| Operation | Binary heap | Fib heap | Pairing heap |
|---|---|---|---|
| insert | O(log n) | O(1) | O(1) |
| find-min | O(1) | O(1) | O(1) |
| extract-min | O(log n) | O(log n) amortized | O(log n) amortized |
| decrease-key | O(log n) | O(1) amortized | O(log n) amortized |
| merge | O(n) | O(1) | O(1) |

## Hash Table

Key → value via hash function.

| Operation | Average | Worst |
|---|---|---|
| insert | O(1) | O(n) |
| lookup | O(1) | O(n) |
| delete | O(1) | O(n) |
| iterate | O(n) | O(n) |
| resize (rehash) | O(n) | O(n) |

### Universal hashing

A family `H` of hash functions is **universal** if for any two keys `x ≠ y`:

```
Pr_{h ∈ H} [h(x) = h(y)] ≤ 1/m
```

Picking `h` from a universal family at table-creation time gives expected O(1) lookup *for any input* — defeating adversarial inputs.

### Perfect hashing

For a *static* set of `n` keys, two-level FKS scheme achieves O(n) total space and O(1) **worst-case** lookup. Cuckoo hashing also achieves O(1) worst-case lookup with O(n) space.

### Open addressing vs chaining

| Trait | Chaining | Open addressing |
|---|---|---|
| Resolution | Linked lists per bucket | In-table probing |
| Load factor α tolerance | up to ~1 (degrades smoothly) | < ~0.7 (degrades sharply) |
| Memory overhead | Pointer per node | None per entry |
| Cache friendly | Worse (pointer chasing) | Better (linear scan) |
| Deletion | Trivial | Tombstones or rehash |
| Linear probe expected | n/a | 1/(1-α) lookup, 1/(1-α)² insert |
| Implementations | Java HashMap, Go map | Rust HashMap, Python dict, Robin Hood, Cuckoo |

### Robin Hood hashing

Open-addressing variant. On insert, if the new probe sequence is longer than the existing entry's, swap them. Reduces variance in probe length; tighter clusters.

| Operation | Average | Worst (large α) |
|---|---|---|
| lookup | O(1) | O(log n) |
| insert | O(1) | O(log n) |

### Cuckoo hashing

Two hash functions `h₁, h₂`. Each key lives at exactly one of two positions. Insert: if both are taken, kick one out and re-place it (cuckoo'd). Worst case: a cycle → rehash.

| Operation | Worst-case |
|---|---|
| lookup | O(1) — at most 2 cells |
| delete | O(1) |
| insert | O(1) amortized; occasional rebuild |

## Binary Search Tree (Unbalanced)

| Operation | Average | Worst |
|---|---|---|
| search | O(log n) | O(n) |
| insert | O(log n) | O(n) |
| delete | O(log n) | O(n) |
| min / max | O(log n) | O(n) |
| in-order traversal | O(n) | O(n) |
| predecessor / successor | O(log n) | O(n) |

```bash
# Worst case: insert sorted keys 1, 2, 3, ..., n
# The "tree" degenerates into a linked list of length n
# All operations become O(n)
# Solution: balanced BSTs (AVL, Red-Black, Treap, Splay)
```

## Balanced BST: AVL

Strict balance: at every node, `|height(L) − height(R)| ≤ 1`.

| Operation | Worst-case |
|---|---|
| search | O(log n) |
| insert | O(log n) — at most 2 rotations |
| delete | O(log n) — up to log n rotations |
| min / max | O(log n) |
| range query | O(log n + k) |

Height bound: `h ≤ 1.44 · log₂(n + 2)` — tighter than Red-Black.

```bash
# AVL vs Red-Black tradeoff
#   AVL: stricter balance → faster lookups (shorter trees)
#   AVL: more rotations on insert/delete (especially delete)
#   Red-Black: looser balance → slightly taller (h ≤ 2 log n)
#   Red-Black: at most 2 rotations on insert, 3 on delete
#   AVL wins read-heavy workloads; RB wins mixed/write-heavy
```

## Balanced BST: Red-Black

Coloring invariant: every path from a node to its descendant leaves contains the same number of black nodes (the **black-height**).

Height bound: `h ≤ 2 · log₂(n + 1)`.

| Operation | Worst-case |
|---|---|
| search | O(log n) |
| insert | O(log n) — ≤ 2 rotations + recolors |
| delete | O(log n) — ≤ 3 rotations + recolors |
| min / max | O(log n) |
| predecessor / successor | O(log n) |

### Real-world defaults

| Library | Implementation |
|---|---|
| Java TreeMap, TreeSet | Red-Black |
| C++ std::map, std::set, std::multimap | Red-Black (typically) |
| Linux kernel CFS scheduler | Red-Black |
| Linux EXT3 HTree directories | Variant |
| Python sortedcontainers | Skip list (Python pure) |
| Go: no built-in | (third-party RB trees exist) |

## B-Tree / B+Tree

Generalized BST: each node holds up to `B - 1` keys and `B` children. Designed for disk/SSD where each node read is one I/O.

| Operation | Complexity (in disk reads) | RAM model |
|---|---|---|
| search | O(log_B n) | O(log n) |
| insert | O(log_B n) | O(log n) |
| delete | O(log_B n) | O(log n) |
| range query (k results) | O(log_B n + k/B) | O(log n + k) |

### B+Tree

All keys in leaves; internal nodes hold only routing keys. Leaves linked left-to-right for fast range scans.

```bash
# Database indexes are almost universally B+Trees
# PostgreSQL btree indexes: B+Tree
# MySQL InnoDB primary index: B+Tree (data clustered)
# SQL Server, Oracle, MariaDB: B+Tree
# SQLite: B-Tree (with B+ leaves for tables)
```

```bash
# Why B and not 2?
#   Each node fits a page (4 KB or 8 KB)
#   With ~100-byte entries, B ~ 40-80
#   Tree of height 4 with B = 100 holds 100^4 = 10^8 keys
#   Lookup = 4 disk reads
#   A binary tree on 10^8 keys = 27 levels = 27 disk reads
```

## Heap (Binary)

Array-backed complete binary tree. Min-heap: parent ≤ children.

For 0-indexed array:
- `parent(i) = (i - 1) / 2`
- `left(i) = 2i + 1`
- `right(i) = 2i + 2`

| Operation | Complexity |
|---|---|
| find-min | O(1) |
| insert (push) | O(log n) — bubble up |
| extract-min (pop) | O(log n) — sift down |
| decrease-key (with index) | O(log n) |
| build-heap (heapify all) | O(n) — not O(n log n)! |
| heap-sort | O(n log n) |
| meld / merge two heaps | O(n) |

### Why build-heap is O(n)

```
∑_{h=0}^{log n} (n / 2^(h+1)) · h  =  n · ∑ h / 2^(h+1)  ≤  n · 2  =  O(n)
```

The sum `∑ h/2^h` converges to a constant. Most nodes are near the bottom and have small height to sift down.

## Heap (Fibonacci)

Theoretically optimal for Dijkstra and Prim.

| Operation | Amortized |
|---|---|
| insert | O(1) |
| find-min | O(1) |
| union (meld) | O(1) |
| decrease-key | O(1) |
| delete-min | O(log n) |
| delete | O(log n) |

### Why rare in practice

```bash
# Constants are HUGE — pointer-heavy structure with marked/parent/child/sibling
# Cache-unfriendly
# Pairing heap or simple binary heap usually faster real-world
# Used in:
#   - Theoretical analysis (proves Dijkstra O(V log V + E))
#   - Network simulators with millions of decrease-key ops
#   - Some Prim's MST implementations on dense graphs
```

## Heap (Pairing)

Self-adjusting, simpler than Fibonacci, often faster in practice.

| Operation | Amortized |
|---|---|
| insert | O(1) |
| find-min | O(1) |
| union | O(1) |
| decrease-key | O(log n) (open: conjectured O(1)) |
| delete-min | O(log n) |

```bash
# C++ Boost: pairing_heap
# Implementation is ~30 lines vs 100+ for Fibonacci
# Real-world: usually beats Fibonacci even on Dijkstra
```

## Trie (Prefix Tree)

Tree where each edge is labeled with a character; path from root spells a string.

Let `m` = key length, `Σ` = alphabet size, `n` = total keys.

| Operation | Complexity |
|---|---|
| insert | O(m) |
| lookup | O(m) |
| delete | O(m) |
| prefix search (count keys with prefix) | O(m) |
| prefix iteration | O(m + k) where k = output |
| starts-with check | O(m) |
| longest-common-prefix | O(min m) |

### Trie vs hash table

| Trait | Trie | Hash table |
|---|---|---|
| Lookup | O(m) | O(m) (m = hash time) |
| Prefix search | O(m + k) | O(n + k) — must scan all |
| Sorted iteration | O(n · avg-m) — yes, sorted | impossible without separate sort |
| Memory | O(n · m · Σ) worst | O(n · m) |
| Cache | Bad — pointer chains | Better |

### Compressed trie (radix tree / Patricia)

Merge chains of single-child nodes into one edge labelled with a substring. Saves memory; same asymptotic ops.

```bash
# Radix tree applications
#   Linux kernel: page cache (struct page lookup by offset)
#   Routing tables (longest prefix match for IP routing)
#   Erlang term storage (ETS)
#   Java's java.util.IpAddressTrie (Guava)
```

## Suffix Tree / Suffix Array

For string `S` of length `n`.

### Suffix tree

Compressed trie of all `n` suffixes of `S`.

| Operation | Complexity |
|---|---|
| construction (Ukkonen) | O(n) |
| pattern search (length m) | O(m) |
| count pattern occurrences | O(m + occ) |
| longest repeated substring | O(n) |
| longest common substring (2 strings) | O(n + m) |
| space | O(n) — but big constant |

### Suffix array

Sorted array of starting positions of suffixes.

| Operation | Complexity |
|---|---|
| construction (DC3 / SA-IS) | O(n) |
| construction (sort-based) | O(n log² n) or O(n log n) |
| pattern search (with LCP) | O(m + log n) |
| pattern search (binary search only) | O(m log n) |
| space | O(n) — small constant |

### LCP array

`LCP[i]` = longest common prefix of `suffix[SA[i-1]]` and `suffix[SA[i]]`. Built in O(n) from suffix array (Kasai's algorithm).

```bash
# Practical guidance
#   Suffix array + LCP: tighter memory, hot in competitive programming
#   Suffix tree: needed for some problems (longest substring of k-distinct...)
#   Suffix automaton: O(n) build, O(m) match — alternative to suffix tree
```

## Segment Tree

Balanced binary tree over an array, each node stores aggregate of a range.

| Operation | Complexity |
|---|---|
| build | O(n) |
| point update | O(log n) |
| range update (lazy propagation) | O(log n) |
| range query (sum, min, max, gcd, ...) | O(log n) |
| range update + range query | O(log n) with lazy prop |
| space | O(4n) ≈ O(n) |

### Variants

| Variant | Use |
|---|---|
| Persistent segment tree | Versioned queries: O(log n) per version |
| 2D segment tree | Range queries on a grid: O(log² n) |
| Segment tree beats | Chthulu-tier hard optimizations |
| Merge-sort tree | Range k-th smallest: O(log² n) |
| Wavelet tree | Range k-th, range count: O(log σ) |

## Fenwick Tree (Binary Indexed Tree)

Implicit tree using bit tricks. Supports prefix-sum and point-update.

| Operation | Complexity |
|---|---|
| build | O(n) (or O(n log n) naive) |
| point update | O(log n) |
| prefix sum (1..k) | O(log n) |
| range sum (l..r) | O(log n) — two prefix sums |
| range update + point query (with diff array) | O(log n) |
| range update + range query | O(log n) — two BITs |
| space | O(n) — exact, no overhead |

```bash
# Typical iteration: i += i & -i  (move to parent)
#                    i -= i & -i  (move to next range to sum)
# Constants: BIT is faster than segment tree for plain sum+update
# But: less flexible — needs invertible op (sum yes, max no)
```

## Disjoint-Set / Union-Find

For maintaining partitions of `n` elements.

| Optimisation | Op cost (over m ops on n elements) |
|---|---|
| Naive (no opt) | O(m · n) |
| Union by rank only | O(m · log n) |
| Path compression only | O(m · log n) |
| Union by rank + path compression | O(m · α(n)) |

`α(n)` is the inverse Ackermann function — `< 5` for any `n ≤ 2^65536`. Effectively O(1).

| Operation | Complexity |
|---|---|
| make-set | O(1) |
| find | O(α(n)) amortized |
| union | O(α(n)) amortized |

```bash
# Applications
#   Kruskal's MST
#   Connected components in dynamic graphs
#   Image segmentation (union neighboring same-color pixels)
#   Equivalence relations
#   Network connectivity in offline queries
```

## Skip List

Probabilistic structure: levels, each successive level skips more nodes.

| Operation | Average | Worst |
|---|---|---|
| search | O(log n) | O(n) |
| insert | O(log n) | O(n) |
| delete | O(log n) | O(n) |
| range query | O(log n + k) | O(n) |

```bash
# Worst case is O(n) but happens with vanishingly small probability
# At level L, expected number of nodes = n / 2^L
# Expected max level = log₂ n
# Expected operations: O(log n)
```

### Concurrent skip list

Easier to make lock-free than balanced trees. Java's `ConcurrentSkipListMap` is a high-throughput sorted map.

| Library | Use of skip list |
|---|---|
| Redis sorted sets (ZSET) | yes |
| LevelDB / RocksDB MemTable | yes |
| Java ConcurrentSkipListMap | yes |
| MongoDB WiredTiger | one of the cache structures |

## Bloom Filter

Probabilistic set membership: never false negative, possible false positive.

Parameters: `n` items, `m` bits, `k` hash functions.

| Operation | Complexity |
|---|---|
| insert | O(k) |
| query | O(k) |
| union (same m, k) | O(m) bitwise OR |
| delete | not supported (use Counting Bloom) |

### False-positive rate

```
p ≈ (1 − e^(−kn/m))^k
```

### Optimal `k`

```
k* = (m/n) · ln 2 ≈ 0.693 · m/n
```

### Optimal `m` for target `p`

```
m = − n · ln p / (ln 2)²  ≈  −1.44 · n · log₂ p
```

Common sizing: `m ≈ 9.6n` per 1% FPR; `m ≈ 14.4n` per 0.1% FPR.

| Target FPR | Bits per element |
|---|---|
| 50% | 1 |
| 10% | 4.8 |
| 1% | 9.6 |
| 0.1% | 14.4 |
| 0.01% | 19.2 |

```bash
# Variants
#   Counting Bloom — supports delete; m·counter-bits
#   Cuckoo Filter — supports delete; tighter; SIMD-friendly
#   Quotient Filter — cache-friendly; resizable
#   Scalable Bloom Filter — grows with insertions
```

## Count-Min Sketch

Probabilistic frequency estimate with over-estimate guarantee.

Parameters: depth `d`, width `w`. Memory: `d · w` counters.

| Operation | Complexity |
|---|---|
| insert | O(d) |
| query | O(d) |
| merge | O(d · w) |

### Error guarantee

```
Pr[ estimate(x) > true(x) + ε · N ]  ≤  δ
where  w = ⌈ e / ε ⌉  and  d = ⌈ ln(1/δ) ⌉
N = total number of insertions
```

| Use case | What it estimates |
|---|---|
| Heavy hitters | Top-k elements with frequency > threshold |
| Network flows | Per-flow byte counts in routers |
| Stream join | Approximate join cardinality |
| Rate limiting | Per-key requests in window |

## HyperLogLog

Cardinality estimation in `O(1)` space (after fixed bucket count `m`).

| Operation | Complexity |
|---|---|
| add | O(1) |
| estimate | O(m) |
| merge | O(m) |

Standard error: `1.04 / √m`. With `m = 2^14 = 16384` buckets and ~1.5 KB memory, error ≈ 0.81%.

```
Algorithm idea
  hash element h(x) — uniform random bits
  use first b bits of h(x) to pick bucket (m = 2^b)
  remaining bits: count leading zeros, store max in bucket
  cardinality estimate ≈ α_m · m² / Σ 2^(−bucket[i])
  α_m is a bias-correction constant
```

```bash
# Used by
#   Redis PFCOUNT / PFADD / PFMERGE
#   Google: BigQuery COUNT(DISTINCT) approximate
#   PostgreSQL: hll extension; Citus for distributed counts
#   Druid: cardinality aggregation
```

## Sorting — Selection

Pick the smallest, swap to front; repeat.

| Aspect | Value |
|---|---|
| Best | Θ(n²) |
| Average | Θ(n²) |
| Worst | Θ(n²) |
| Space | O(1) auxiliary, in-place |
| Stable | No |
| Comparisons | n(n-1)/2 always |
| Swaps | n - 1 |

```python
def selection_sort(a):
    for i in range(len(a)):
        m = i
        for j in range(i + 1, len(a)):
            if a[j] < a[m]:
                m = j
        a[i], a[m] = a[m], a[i]
```

```bash
# Where it wins: minimal swaps when writes are very expensive (flash, EEPROM)
# Otherwise: never the right choice
```

## Sorting — Insertion

Like sorting cards in your hand.

| Aspect | Value |
|---|---|
| Best | Θ(n) — already sorted |
| Average | Θ(n²) |
| Worst | Θ(n²) — reverse sorted |
| Space | O(1) auxiliary, in-place |
| Stable | Yes |
| Adaptive | Yes — runs in O(n + d) where d = inversions |

```python
def insertion_sort(a):
    for i in range(1, len(a)):
        x = a[i]
        j = i - 1
        while j >= 0 and a[j] > x:
            a[j + 1] = a[j]
            j -= 1
        a[j + 1] = x
```

```bash
# Where it wins: small n (≤ 32) or nearly-sorted input
# Used as base case in Timsort, introsort, std::sort
# Cache-friendly; few comparisons on small inputs
```

## Sorting — Bubble

Repeatedly compare adjacent pairs and swap.

| Aspect | Value |
|---|---|
| Best | Θ(n) — with early exit |
| Average | Θ(n²) |
| Worst | Θ(n²) |
| Space | O(1), in-place |
| Stable | Yes |

```bash
# Almost never the right choice; pedagogical only
# Even insertion sort is uniformly better
# Notable only as the answer to "what's the slowest natural sort"
```

## Sorting — Merge

Divide-and-conquer split-and-merge.

| Aspect | Value |
|---|---|
| Best | Θ(n log n) |
| Average | Θ(n log n) |
| Worst | Θ(n log n) |
| Space | O(n) auxiliary |
| Stable | Yes |
| External (disk-friendly) | Yes |
| Parallelisable | Yes — independent halves |

```python
def merge_sort(a):
    if len(a) <= 1: return a
    mid = len(a) // 2
    L = merge_sort(a[:mid])
    R = merge_sort(a[mid:])
    return merge(L, R)
```

```bash
# Where it wins:
#   - Linked lists (no random access) — O(n log n), O(1) extra space
#   - External sorting (multi-pass over disk)
#   - Stable sort needed and you can afford O(n) memory
#   - Parallel sorting
# Java Arrays.sort for objects, Python sorted (Timsort base case is merge)
```

## Sorting — Quick

Choose pivot, partition, recurse.

| Aspect | Value |
|---|---|
| Best | Θ(n log n) |
| Average | Θ(n log n) |
| Worst | Θ(n²) |
| Space | O(log n) avg, O(n) worst (recursion) |
| Stable | No (typical impl) |
| In-place | Yes |
| Cache-friendly | Yes — sequential partition |

### Pivot strategies

| Strategy | Worst-case input |
|---|---|
| First or last element | Already sorted |
| Random | Adversarial (probability 1/n!) |
| Median-of-three (first/middle/last) | Killer inputs exist (Musser 1997) |
| Median-of-medians (BFPRT) | Always Θ(n log n), but slow constants |
| Introselect (random until depth) | Always O(n log n), good constants |

```bash
# Modern std::sort (introsort)
#   1. Quicksort with median-of-three
#   2. After 2·log₂ n recursion levels, switch to heapsort
#   3. Below threshold (16-32), switch to insertion sort
# Result: O(n log n) worst, real-world the fastest
```

## Sorting — Heap

Build heap, extract-min n times.

| Aspect | Value |
|---|---|
| Best | Θ(n log n) |
| Average | Θ(n log n) |
| Worst | Θ(n log n) |
| Space | O(1), in-place |
| Stable | No |
| Cache-friendly | No — strided access pattern |

```bash
# Where it wins:
#   - O(n log n) worst case + in-place (only such sort!)
#   - As fallback in introsort
#   - Embedded systems with strict memory bounds
# Rarely the fastest in benchmarks vs quicksort
```

## Sorting — Counting

For integer keys in `[0, k)`.

| Aspect | Value |
|---|---|
| Time | Θ(n + k) |
| Space | Θ(n + k) |
| Stable | Yes (proper impl) |
| Comparison-based | No |
| Constraint | Keys are bounded integers |

```python
def counting_sort(a, k):
    count = [0] * k
    for x in a: count[x] += 1
    out, j = [], 0
    for v in range(k):
        for _ in range(count[v]):
            out.append(v); j += 1
    return out
```

```bash
# Where it wins:
#   - Histogram-based work
#   - Subroutine for radix sort
#   - When k = O(n)
# When it loses:
#   - k >> n (sparse keys)
```

## Sorting — Radix

Sort by digit, LSD or MSD.

| Variant | Time | Space | Stable |
|---|---|---|---|
| LSD | O(d · (n + b)) | O(n + b) | Yes |
| MSD | O(d · (n + b)) | O(n + b · d) | Yes (proper impl) |

`d` = number of digits, `b` = base (radix).

```bash
# 32-bit integers, base 256: d = 4 passes over n
# 64-bit integers, base 256: d = 8 passes
# String sort: d = max string length
# In practice radix sort beats quicksort for huge integer arrays
```

## Sorting — Bucket

Distribute into k buckets, sort each, concatenate.

| Aspect | Value |
|---|---|
| Best (uniform) | Θ(n + k) |
| Average | Θ(n + k) |
| Worst | Θ(n²) — all in one bucket |
| Space | Θ(n + k) |

```bash
# Best when input is uniformly distributed over a known range
# Worst when distribution is concentrated
# Used in some database query engines for sorting floats with known range
```

## Sorting — Introsort

Hybrid: quicksort + heapsort fallback + insertion-sort base.

| Aspect | Value |
|---|---|
| Worst | Θ(n log n) |
| Average | Θ(n log n) |
| Space | O(log n) |
| Stable | No |
| In-place | Yes |

```bash
# Used by:
#   - C++ std::sort (since C++03)
#   - .NET Array.Sort
#   - Microsoft STL, libstdc++, libc++
# The de-facto best general-purpose comparison sort
```

## Sorting — Timsort

Hybrid: identifies natural runs, merges them.

| Aspect | Value |
|---|---|
| Best | Θ(n) — single sorted run |
| Average | Θ(n log n) |
| Worst | Θ(n log n) |
| Space | O(n) |
| Stable | Yes |
| Adaptive | Yes — exploits pre-existing order |

```bash
# Algorithm sketch
#   1. Scan left-to-right, finding "runs" — already sorted subsequences
#   2. Reverse descending runs into ascending
#   3. Pad short runs to a minimum size with binary insertion sort
#   4. Push runs onto a stack; merge by maintaining invariants
# Used by:
#   - Python list.sort, sorted (since 2002)
#   - Java Arrays.sort for objects (since Java 7)
#   - Android, V8, Rust slice::sort, Swift
```

## Lower Bound for Comparison Sort

```
Any comparison-based sort needs Ω(n log n) comparisons in the worst case.
```

### Decision tree argument

```
- A comparison sort is a binary decision tree (compare a[i], a[j]; branch)
- There are n! possible permutations
- Each must lead to a distinct leaf
- A binary tree of height h has at most 2^h leaves
- So 2^h ≥ n!  ⇒  h ≥ log₂(n!) ≈ n log₂ n − n / ln 2
- Worst-case path length is the worst-case comparison count
- Therefore Ω(n log n)
```

### Beating the bound — when?

Linear-time sorts exist when input has *structure*:

| Sort | When applicable |
|---|---|
| Counting sort | Integer keys in O(n) range |
| Radix sort | Fixed-width keys (digits, ints, strings) |
| Bucket sort | Uniformly distributed keys |
| Pigeonhole sort | Small key range relative to n |

These are *non-comparison* sorts; the lower bound doesn't apply.

## Graph Traversal

Standard model: graph `G = (V, E)`, `n = |V|`, `m = |E|`.

### BFS / DFS

| Operation | Adj list | Adj matrix |
|---|---|---|
| BFS / DFS | O(V + E) | O(V²) |
| Detect connected components | O(V + E) | O(V²) |
| Detect cycle (undirected) | O(V + E) | O(V²) |
| Detect cycle (directed) | O(V + E) | O(V²) |
| Bipartiteness | O(V + E) | O(V²) |
| 2-coloring | O(V + E) | O(V²) |

```bash
# BFS uses a queue → finds shortest paths (in unweighted graphs)
# DFS uses a stack → can be recursive; finds back-edges, articulation points
# Time = O(visit each vertex once + traverse each edge once) = O(V + E)
```

### Adjacency list vs matrix

| Trait | List | Matrix |
|---|---|---|
| Space | O(V + E) | O(V²) |
| Add edge | O(1) | O(1) |
| Remove edge | O(degree) | O(1) |
| Edge query (u, v)? | O(degree) | O(1) |
| Iterate neighbours of v | O(degree(v)) | O(V) |
| Iterate all edges | O(V + E) | O(V²) |
| Best for | Sparse graphs (E = O(V)) | Dense graphs (E = Θ(V²)) |

## Shortest Path — Dijkstra

Single-source shortest path on **non-negative** edge weights.

| Heap | Complexity |
|---|---|
| Naive (array, find-min) | O(V²) |
| Binary heap | O((V + E) log V) |
| Pairing heap | O((V + E) log V) practical |
| Fibonacci heap | O(V log V + E) |
| Dial's bucket queue (bounded weights) | O(V + E + max-weight) |

```bash
# Why non-negative?
#   Dijkstra commits to a vertex's distance when extracted from PQ
#   A negative edge could later "improve" that committed distance
#   Bellman-Ford handles it by relaxing for V-1 rounds
```

## Shortest Path — Bellman-Ford

Handles negative weights; detects negative cycles.

| Aspect | Complexity |
|---|---|
| Time | O(V · E) |
| Space | O(V) |
| Detects negative cycle | Yes — extra V-th relaxation iteration |
| Distributed variant | Distance-vector routing (RIP) |

```bash
# Algorithm
#   for V-1 iterations:
#     for each edge (u, v) with weight w:
#       dist[v] = min(dist[v], dist[u] + w)
#   if any edge can still be relaxed → negative cycle reachable
```

## Shortest Path — Floyd-Warshall

All-pairs shortest path.

| Aspect | Complexity |
|---|---|
| Time | O(V³) |
| Space | O(V²) |
| Negative edges | OK (no negative cycle) |
| Negative cycle detection | dist[i][i] < 0 |
| Transitive closure | Variant: O(V³) Boolean ops |

```bash
# Triple loop
#   for k in V:
#     for i in V:
#       for j in V:
#         dist[i][j] = min(dist[i][j], dist[i][k] + dist[k][j])
# Pulse: with bitset and adjacency matrix, transitive closure
# can be done in O(V³ / 64) on a 64-bit machine.
```

## Shortest Path — A*

Heuristic search; generalises Dijkstra.

| Aspect | Complexity |
|---|---|
| Time | O(b^d) worst (b = branching, d = depth) |
| Time (admissible h) | ≤ O(E + V log V) like Dijkstra |
| Time (consistent h) | ≤ O(E + V log V); never re-expands |
| Space | O(V) for closed/open sets |

### Heuristic properties

| Property | Definition |
|---|---|
| Admissible | h(n) ≤ h*(n) — never overestimates |
| Consistent | h(u) ≤ w(u,v) + h(v) for every edge — triangle inequality |
| Consistent ⇒ admissible | Yes |

```bash
# h(n) = 0 → A* = Dijkstra
# h(n) = h*(n) → A* expands only the optimal path
# Manhattan distance on grid (no obstacles) is consistent for 4-connectivity
# Euclidean distance is consistent for any planar graph with Euclidean weights
```

## MST — Kruskal

| Aspect | Complexity |
|---|---|
| Total time | O(E log E) = O(E log V) |
| Sort step | O(E log E) |
| Union-find ops | O(E · α(V)) |
| Space | O(V + E) |

```bash
# Algorithm
#   sort edges by weight
#   for each edge (u, v) in sorted order:
#     if find(u) != find(v):
#       add edge to MST; union(u, v)
# Best for sparse graphs and offline (all edges available)
```

## MST — Prim

| Heap | Complexity |
|---|---|
| Naive (array) | O(V²) |
| Binary heap | O((V + E) log V) |
| Fibonacci heap | O(E + V log V) |

```bash
# Algorithm
#   start at any vertex; mark visited
#   repeatedly: take cheapest edge to an unvisited vertex
#   = Dijkstra-like, but priority is edge weight not path distance
# Best for dense graphs (V² weight matrix); Fibonacci heap optimal asymptotic
```

## Topological Sort

For a DAG.

| Algorithm | Complexity |
|---|---|
| Kahn's algorithm (BFS, in-degree queue) | O(V + E) |
| DFS-based (post-order) | O(V + E) |

```bash
# Kahn's: queue of vertices with in-degree 0; pop, append to order, decrement out-neighbours
# DFS: post-order traversal gives reverse-topological; reverse for topological
# Detect non-DAG: Kahn's leaves nodes; DFS finds back-edge
```

```bash
# Applications
#   Build systems (make, ninja)
#   Task scheduling with dependencies
#   Course prerequisite checking
#   Single-source shortest path on DAGs (linear time, even with negative edges)
```

## Strongly Connected Components

A maximal set of vertices where every vertex reaches every other.

| Algorithm | Complexity | Notes |
|---|---|---|
| Tarjan | O(V + E) | Single DFS pass; uses low-link |
| Kosaraju | O(V + E) | Two DFS passes; reverse graph |
| Path-based (Gabow) | O(V + E) | Two stacks |

```bash
# Tarjan: most popular; tracks discovery time and low-link
# Kosaraju: easiest to explain; (1) DFS to get finish order; (2) DFS on transpose in reverse-finish order
# Output: V partitioned into SCCs; condensation graph is a DAG
```

```bash
# Applications
#   2-SAT solver (build implication graph; SCC each clause)
#   Web graph clustering
#   Circuit dependency analysis
#   Component graph for algorithm post-processing
```

## Max Flow

| Algorithm | Complexity | Notes |
|---|---|---|
| Ford-Fulkerson (DFS) | O(E · max_flow) | Doesn't terminate on irrational weights |
| Edmonds-Karp (BFS) | O(V · E²) | Always terminates; integer or rational |
| Dinic | O(V² · E) | O(E · √V) on unit-capacity bipartite |
| Push-relabel (FIFO) | O(V³) | |
| Push-relabel (highest-label) | O(V² · √E) | Best deterministic |
| Orlin (2013) | O(V · E) | Strongly poly-time |

```bash
# Min-cut max-flow theorem: max flow = min cut
# Capacity scaling: O(E² log U) where U = max capacity
# Network simplex: O(V·E·U·C) but fast in practice
```

## Bipartite Matching

| Algorithm | Complexity |
|---|---|
| Naive augmenting path | O(V · E) |
| Hopcroft-Karp | O(E · √V) |
| Hungarian algorithm (assignment) | O(V³) |
| Min-cost max-flow | O(V² · E) or better |

### König's theorem

In a bipartite graph: `max matching = min vertex cover`.

This implies a polynomial reduction:
- Max bipartite matching ↔ min vertex cover in bipartite ↔ max independent set in bipartite

(In general graphs these are NP-hard.)

## Network Flow Applications

### Bipartite matching

Source → left vertices (cap 1) → right vertices (cap 1) → sink.
Max flow = max matching.

### Vertex cover (bipartite)

Min vertex cover = max matching by König's. Solve as bipartite matching.

### Project selection / closure

Variables (projects) with profits/costs and dependencies. Build flow network: source → profitable, costs → sink, dependencies → infinite edges. Min cut = projects to skip.

### Image segmentation

Pixels = nodes; foreground/background priors = source/sink edges; smoothness = inter-pixel edges. Min cut = optimal segmentation. Boykov-Kolmogorov algorithm specialized for this.

```bash
# Other reductions to max flow
#   Edge-disjoint paths
#   Vertex-disjoint paths (split each vertex)
#   Multi-source / multi-sink (super-source)
#   Lower-bound capacities (constructive)
#   Maximum-density subgraph (parametric)
```

## String Matching

Match pattern `P` (length `m`) in text `T` (length `n`).

| Algorithm | Preprocess | Search | Total | Notes |
|---|---|---|---|---|
| Naive | 0 | O(n · m) | O(n · m) | Slides one character |
| KMP | O(m) | O(n) | O(n + m) | Failure function |
| Boyer-Moore | O(m + Σ) | O(n / m) best, O(n · m) worst | sublinear | Bad-char + good-suffix heuristics |
| Boyer-Moore-Horspool | O(m + Σ) | O(n) avg, O(n · m) worst | simpler BM | Common in libc, grep |
| Rabin-Karp | O(m) | O(n + m) avg, O(n · m) worst | rolling hash | Multi-pattern friendly |
| Z-algorithm | O(m) | O(n) | O(n + m) | Z-array |
| Aho-Corasick | O(Σ |Pᵢ|) | O(n + occ) | finite automaton | Multi-pattern |
| Suffix tree (built once) | O(n) | O(m + occ) per pattern | many patterns | Large constant |
| FM-index | O(n log n) | O(m) | compressed | bowtie / bwa use this |

```bash
# When to use which
#   - One pattern, online: Boyer-Moore-Horspool (or memmem in libc)
#   - One pattern, theoretical guarantee: KMP
#   - Many patterns: Aho-Corasick
#   - Many queries on same text: build suffix tree / array once
#   - Approximate match: bitap (Wu-Manber) O(n · ⌈m/w⌉)
```

## Number-Theoretic

| Operation | Complexity |
|---|---|
| Euclidean GCD | O(log min(a, b)) |
| Extended Euclidean (Bezout) | O(log min(a, b)) |
| Modular exponentiation a^b mod n | O(log b) multiplications |
| Sieve of Eratosthenes | O(n log log n) |
| Linear sieve | O(n) |
| Sieve of Atkin | O(n / log log n) |
| Miller-Rabin (k rounds) | O(k · log³ n) |
| AKS deterministic | O(log^7.5 n) |
| Pollard's rho factoring | O(n^(1/4)) expected |
| Trial division up to √n | O(√n / log n) primes |
| Discrete log (baby-step giant-step) | O(√n) |
| Discrete log (Pollard rho) | O(√n) expected |

### Euclidean algorithm intuition

```
gcd(a, b) = gcd(b, a mod b),  gcd(a, 0) = a
```

The number of steps is `O(log min(a, b))` because each two steps reduce by ≥ a factor of 2 (Lamé's theorem; Fibonacci numbers are the worst case).

### Sieve of Eratosthenes

```python
def sieve(n):
    is_prime = [True] * (n + 1)
    is_prime[0] = is_prime[1] = False
    for i in range(2, int(n**0.5) + 1):
        if is_prime[i]:
            for j in range(i*i, n + 1, i):
                is_prime[j] = False
    return [i for i, p in enumerate(is_prime) if p]
```

```bash
# O(n log log n) — Mertens' theorem on prime harmonic
# Linear sieve: O(n) by ensuring each composite is marked exactly once
#   keep "smallest prime factor" array; iterate primes in order
```

## Linear Algebra

| Operation | Naive | Best known |
|---|---|---|
| Matrix-vector multiply (n×n by n×1) | O(n²) | O(n²) optimal |
| Matrix multiply (n×n) | O(n³) | O(n^2.371552) (Williams 2024) |
| Matrix inversion | O(n³) | O(n^ω) — same as multiply |
| Determinant | O(n³) | O(n^ω) |
| LU decomposition | O(n³) | O(n^ω) |
| QR decomposition | O(n³) | O(n^ω) |
| SVD | O(n³) | O(n^ω) |
| Eigenvalues | O(n³) | O(n^ω) (numerically) |
| Solve Ax = b (Gaussian) | O(n³) | O(n^ω) |
| Sparse Ax = b (k nonzeros) | O(k · iter) | depends on conditioning |

### Sub-cubic matrix multiplication timeline

| Year | Algorithm | Exponent |
|---|---|---|
| 1969 | Strassen | 2.807 |
| 1978 | Pan | 2.795 |
| 1979 | Bini | 2.7799 |
| 1981 | Schönhage | 2.522 |
| 1986 | Strassen (laser) | 2.479 |
| 1990 | Coppersmith-Winograd | 2.376 |
| 2010 | Stothers | 2.374 |
| 2014 | Le Gall | 2.3729 |
| 2020 | Alman-Williams | 2.37286 |
| 2024 | Williams-Xu-Xu-Zhou | 2.371552 |

```bash
# Most "subcubic" algorithms are GALACTIC — break-even at n > 10^9
# Strassen is practical from n ~ 100; LAPACK uses it inside
# Real-world libraries: BLAS, MKL, cuBLAS — all use Strassen-derived
# Only naive O(n³) used for n < 64 (cache + SIMD beats theoretical asymptotic)
```

## FFT

Fast Fourier Transform: discrete Fourier transform in O(n log n).

| Operation | Complexity |
|---|---|
| FFT (n a power of 2) | O(n log n) |
| Inverse FFT | O(n log n) |
| Polynomial multiplication (degree n) | O(n log n) |
| Integer multiplication (n digits) | O(n log n · log log n) — Schönhage-Strassen |
| Integer multiplication (Harvey-Hoeven 2019) | O(n log n) |
| Convolution of length n | O(n log n) |
| Number-theoretic transform (NTT) | O(n log n) |

### The Cooley-Tukey recurrence

```
T(n) = 2 T(n/2) + O(n)   →   T(n) = O(n log n)
```

```bash
# Applications
#   Polynomial multiplication
#   Big integer multiplication (GMP, OpenSSL bignum)
#   Signal processing (audio, DSP, wireless)
#   Image processing (frequency-domain filtering)
#   Convolutional neural networks (FFT-based conv)
#   Pattern matching with wildcards (Clifford O(n log n))
```

## NP / NP-Hard / NP-Complete

### Definitions

- **P** — problems solvable in polynomial time on a deterministic Turing machine.
- **NP** — problems where a "yes" answer can be *verified* in polynomial time given a certificate.
- **NP-hard** — at least as hard as every problem in NP. (Not necessarily in NP.)
- **NP-complete** — both NP-hard and in NP.

```
P ⊆ NP   (everything you can solve, you can verify)
P = NP?  Open. Most believe P ≠ NP.
```

### Reduction

`A ≤_p B` means: a polynomial-time algorithm for B yields a polynomial-time algorithm for A.

```bash
# To prove a new problem X is NP-hard:
#   1. Pick a known NP-hard problem K (3SAT, Vertex Cover, ...)
#   2. Show K ≤_p X by giving a polynomial-time reduction
#   3. If X ∈ NP, then X is NP-complete
```

### Cook-Levin theorem

```
SAT is NP-complete.
```

Every problem in NP reduces to SAT in polynomial time. SAT was the first NP-complete problem.

### Classic NP-complete problems

| Problem | Decision version |
|---|---|
| SAT | Is this Boolean formula satisfiable? |
| 3-SAT | Same, restricted to 3-CNF clauses |
| Vertex Cover | Is there a vertex cover of size ≤ k? |
| Independent Set | Is there an independent set of size ≥ k? |
| Clique | Does the graph contain a k-clique? |
| Hamiltonian Cycle | Is there a cycle visiting every vertex? |
| Hamiltonian Path | Path visiting every vertex once? |
| TSP (decision) | Tour of length ≤ k? |
| Knapsack (decision) | Pack value ≥ V into capacity W? |
| Subset Sum | Does any subset sum to T? |
| Graph 3-Coloring | Can the graph be properly 3-colored? |
| Bin Packing | Pack items into k bins of capacity C? |
| Job Scheduling | Can jobs finish by deadline? |
| Set Cover | Cover universe with k sets? |
| Steiner Tree | Connect terminals with tree of weight ≤ k? |
| Integer Programming | Does the IP have a feasible solution? |

```bash
# If ANY NP-complete problem is in P, then P = NP (and ALL of NP is in P)
# This is why NP-completeness is used as practical evidence of intractability
# Karp's 1972 paper showed 21 problems are NP-complete; today thousands
```

### Other complexity classes worth knowing

| Class | Definition |
|---|---|
| co-NP | "no" answer verifiable in polynomial time |
| NP ∩ co-NP | Both yes and no certifiable; e.g. integer factoring is here |
| PSPACE | Solvable in polynomial space (allowing exponential time) |
| EXPTIME | Solvable in 2^poly(n) time |
| EXPSPACE | Solvable in 2^poly(n) space |
| BPP | Bounded-error randomised polynomial-time |
| RP | One-sided random poly-time |
| ZPP | Las Vegas — expected poly-time |
| #P | Counting versions ("how many satisfying assignments") |

```
P ⊆ NP ⊆ PSPACE ⊆ EXPTIME ⊆ EXPSPACE
P ⊆ BPP ⊆ PSPACE
NP ⊆ PH (polynomial hierarchy) ⊆ PSPACE
```

## Approximation Algorithms

For NP-hard optimisation, often we relax exactness for tractable approximation.

### Definitions

- **α-approximation** for minimisation: returns solution of cost ≤ α · OPT.
- **α-approximation** for maximisation: returns solution of value ≥ (1/α) · OPT.
- **PTAS** (Polynomial-Time Approximation Scheme) — for any ε > 0, has a (1 + ε)-approx in poly(n) time. May be exponential in 1/ε.
- **FPTAS** (Fully Polynomial-TAS) — same, but poly in both n and 1/ε.

| Problem | Best known approximation |
|---|---|
| Vertex Cover | 2 (trivial; no better unless UGC) |
| TSP (general) | unbounded (no const-factor unless P = NP) |
| TSP (metric) | 1.5 (Christofides 1976) |
| TSP (Euclidean) | PTAS (Arora 1996) |
| Set Cover | ln n (greedy; tight unless P = NP) |
| Knapsack | FPTAS — (1 + ε) in O(n²/ε) |
| Bin Packing | Asymptotic FPTAS (Karmarkar-Karp 1982) |
| Steiner Tree | 1.39 (Byrka et al 2010) |
| Max Cut | 0.878 (Goemans-Williamson SDP) |
| Max-3-SAT | 7/8 (Karloff-Zwick; tight by Håstad) |
| Facility Location | 1.488 (Li 2011) |

```bash
# Christofides for metric TSP
#   1. Compute MST T
#   2. Find min-weight perfect matching M on odd-degree vertices of T
#   3. Eulerian tour of T ∪ M
#   4. Shortcut to Hamiltonian cycle
# Cost ≤ 1.5 · OPT
```

### Inapproximability

| Problem | Lower bound |
|---|---|
| Set Cover | ≥ ln n unless P = NP (Feige 1998) |
| Max-3-SAT | ≥ 7/8 unless P = NP (Håstad 2001) |
| Clique | ≥ n^(1-ε) unless P = NP |
| TSP general | No const-factor unless P = NP |
| Vertex Cover | ≥ 1.36 unless P = NP; ≥ 2 under UGC |

## Randomized Algorithms

### Las Vegas vs Monte Carlo

| Type | Output | Time |
|---|---|---|
| Las Vegas | Always correct | Random — expected polynomial |
| Monte Carlo | Possibly wrong (bounded-error) | Always polynomial |

### Examples

| Algorithm | Type | Complexity |
|---|---|---|
| Randomised quicksort | Las Vegas | O(n log n) expected |
| Randomised quickselect | Las Vegas | O(n) expected |
| Miller-Rabin primality | Monte Carlo | O(k · log³ n), error ≤ 4^(-k) |
| Solovay-Strassen | Monte Carlo | similar |
| AKS | Deterministic poly-time | O(log^7.5 n) — practically slower |
| Karger's min cut | Monte Carlo | O(n²) per run, O(n²) trials |
| Karger-Stein min cut | Monte Carlo | O(n² log³ n) total |
| Reservoir sampling | Las Vegas | O(n) |
| Bloom filter | Monte Carlo (no false negatives, possible false positive) | O(k) |
| Treap | Las Vegas | O(log n) expected |

### Karger's min cut

```bash
# Algorithm
#   while > 2 vertices: pick random edge, contract its endpoints
#   the remaining edges are a candidate min cut
# Probability of finding true min cut in one run: ≥ 2 / (n(n-1))
# Run O(n² log n) times: success probability → 1 - 1/n
# Total: O(n²·m·log n) for the boost, or O(n² · n²) with edge list
```

### Randomised quicksort analysis

```
Indicator: X_ij = 1 if i-th and j-th smallest are compared
Pr[X_ij = 1] = 2 / (j - i + 1)
E[total comparisons] = Σ_i Σ_{j>i} 2 / (j - i + 1) = O(n log n)
```

## Online Algorithms

Algorithm receives input piece-by-piece and must commit irrevocably without seeing the future.

### Competitive ratio

```
ALG is c-competitive  ⇔  cost(ALG, σ) ≤ c · cost(OPT, σ) + α
                                                            ↑ constant additive
for every input sequence σ. OPT is the offline optimum.
```

### Examples

| Problem | Best deterministic | Notes |
|---|---|---|
| Ski rental | 2-competitive | Buy after b days where b = price; pay ≤ 2·OPT |
| Paging (LRU) | k-competitive | k = cache size; tight |
| Paging (any deterministic) | ≥ k | matching lower bound |
| Paging (randomised, marking) | O(log k)-competitive | better than deterministic |
| List update (move-to-front) | 2-competitive | |
| k-server (line) | k-competitive | DC algorithm |
| k-server (general) | unknown — conjecture k-competitive | |
| Bin packing (online) | 1.5-competitive | first-fit-decreasing offline |

```bash
# Ski rental
#   You don't know how many days you'll ski
#   Each day rent costs $1, buy costs $b
#   ALG: rent for b days, then buy
#   If you ski < b days: cost = days, OPT = days; ratio 1
#   If you ski ≥ b days: cost = 2b, OPT = b; ratio 2
#   So ALG is 2-competitive (deterministic optimal)
#   Randomised: e/(e-1) ≈ 1.58 competitive
```

## Common Errors

### Confusing average and worst

```bash
# WRONG: "Hash table is O(1)" — only on average / expected
# RIGHT: O(1) average; O(n) worst (collision chain attack)
```

### Ignoring constants when n is small

```bash
# WRONG: pick mergesort over insertion sort always
# RIGHT: insertion sort beats mergesort for n < ~20 due to constants
#        (this is why std::sort uses insertion sort below threshold)
```

### Hash table assumed always O(1)

```bash
# WRONG: "I'll use a hashmap, it's O(1)"
# RIGHT: O(1) requires:
#   - Good hash function (uniform)
#   - Load factor under control (resize on threshold)
#   - No adversarial inputs (use SipHash for user-controlled keys)
# Otherwise: HashDoS attack → O(n) per lookup
```

### Binary search on unsorted

```bash
# WRONG: binary search assumes sorted; fails silently
# RIGHT: sort first (Θ(n log n)), THEN binary search (O(log n))
#        if you only do one query, just linear search (O(n))
```

### Recursion stack vs auxiliary space

```bash
# WRONG: "merge sort is O(1) space because it's in-place merge"
# RIGHT: recursion stack is O(log n); merge buffer is O(n) — total O(n)
#        "in-place" usually allows O(log n) for stack
```

### Big-O ≠ Big-Theta

```bash
# WRONG: "linear search is O(n)" suggesting tightness
# RIGHT: linear search is Θ(n) — tight bound
#        for an upper-bound-only claim, O(n²) is also "correct"
```

### Confusing input-size encoding

```bash
# WRONG: "factoring is polynomial in N (the integer)"
# RIGHT: input length is log N (bits); factoring is sub-exponential in log N
#        — pseudo-polynomial-time algorithms run in poly(N) but exp(log N)
```

### Master theorem misapplication

```bash
# WRONG: T(n) = 2T(n/2) + n log n → "Case 2"
# RIGHT: f(n) = n log n is not Θ(n^c* log^k n) for k = 0 — it's k = 1
#        Case 2 with k = 1 → T(n) = Θ(n log² n)
```

### Confusing P, NP, NP-complete

```bash
# WRONG: "this is in NP, so it's intractable"
# RIGHT: NP includes P (every poly-time problem can be verified in poly time)
#        Sorting is in NP. Sorting is also in P. Sorting is easy.
#        "In NP" alone says nothing about hardness.
```

## Common Gotchas

```bash
# BROKEN: sum of 1+1/2+1/3+...+1/n is O(1)
# (the harmonic series — assume it converges like a geometric series)
```

```bash
# FIXED:  H_n ≈ ln n = O(log n)
# Diverges, but slowly. The constant in O(log n) is small (1/ln 2 ≈ 1.44).
# Used in analysis of: quicksort recurrence, coupon collector, online learning regret.
```

```bash
# BROKEN: "Build heap in O(n log n)" — naive analysis: n inserts, each O(log n)
```

```bash
# FIXED:  Build heap in O(n) when done bottom-up.
# Σ (h · n / 2^(h+1)) = O(n) because Σ h/2^h converges.
# Top-down naive: O(n log n). Bottom-up Floyd's heapify: O(n).
```

```bash
# BROKEN: "Insertion sort is O(n²)" — therefore worse than mergesort always
```

```bash
# FIXED:  Insertion sort is O(n + d) where d = number of inversions.
# On already-sorted input: O(n). Beats mergesort when d ≤ n.
# This is why Timsort exploits "natural runs" — turns mergesort
# into insertion sort on each run.
```

```bash
# BROKEN: "Quicksort worst case O(n²) means it's slower than mergesort in practice"
```

```bash
# FIXED:  Average and best are O(n log n) with smaller constants.
# Cache-friendly partition; in-place. Beats mergesort on RAM workloads.
# Std::sort (introsort) makes worst case O(n log n) by switching to heapsort.
```

```bash
# BROKEN: "Hashmap put/get is O(1)" — lookups always constant
```

```bash
# FIXED:  Average O(1) under uniform hashing assumption with α bounded.
# Worst case O(n) — chain becomes a list.
# Universal hashing: pick h randomly from a family → defeats adversary.
# HashDoS: 2003 attacks on PHP/Perl/Python → introduced random hash seeds.
# 2011: hash flooding paper → SipHash; Python 3.4 enabled by default.
```

```bash
# BROKEN: "Dijkstra always works for shortest path" — no caveat
```

```bash
# FIXED:  Dijkstra requires non-negative edge weights.
# A negative edge can violate the "extracted vertex is finalised" invariant.
# For negative edges: Bellman-Ford O(V·E), Johnson's O(V·E + V² log V).
# For DAG with negative edges: topo-sort + relax in O(V + E) — easiest case.
```

```bash
# BROKEN: "BFS finds shortest path in any graph" — generic claim
```

```bash
# FIXED:  BFS finds shortest path in UNWEIGHTED graphs (or unit weights).
# For weighted with non-negative: Dijkstra.
# For 0/1 weights: 0-1 BFS with deque (O(V + E)).
# For arbitrary weights: Bellman-Ford or Johnson's.
```

```bash
# BROKEN: "Sorting is Θ(n log n)" — apparently universal lower bound
```

```bash
# FIXED:  Comparison-based sorting is Ω(n log n).
# Counting/radix/bucket sorts are O(n) or O(nd) — non-comparison.
# Use them when keys have structure (bounded integers, fixed-width strings).
# Radix sort 32-bit ints in 4 passes of 256 buckets — O(4n) ≈ O(n).
```

```bash
# BROKEN: "Bloom filter is O(1) per operation" — therefore equivalent to hash set
```

```bash
# FIXED:  Bloom filter is O(k) where k = number of hash functions.
# Pros: 1.44 · log₂(1/p) bits per element vs 64+ for a hashset entry.
# Cons: false positives at rate p; cannot delete (Counting Bloom can).
# Use when: massive n, can tolerate occasional false positive,
# memory > correctness (e.g., "URL probably crawled?" before fetch).
```

```bash
# BROKEN: "Trie is faster than hash for any string operation" — therefore always trie
```

```bash
# FIXED:  Trie is O(m) where m = key length (vs hash also O(m) for hashing).
# Hash often has better constants on point queries.
# Trie wins for prefix queries (autocomplete, longest-prefix-match in routing).
# Memory: trie is O(n·m·Σ) worst — can be huge. Compressed trie / radix tree fixes.
```

```bash
# BROKEN: "Recursion is always O(stack depth) memory"
```

```bash
# FIXED:  Tail-call optimisation (TCO) reduces tail recursion to O(1) stack.
# Languages with TCO: Scheme, ML, Lisp, Lua (manual), Scala (annotated).
# Languages without: Python (deliberately), Java, Go (mostly no), C/C++ (compiler).
# In Python, deep recursion → RecursionError; convert to iteration.
```

## Quick Reference Table

Operations across data structures (average/expected unless noted; n = size).

| Structure | access[i] | search | insert | delete | min/max | range |
|---|---|---|---|---|---|---|
| Array (static) | O(1) | O(n) / O(log n) sorted | impossible | impossible | O(n) / O(1) sorted | O(k) |
| Dynamic array | O(1) | O(n) / O(log n) sorted | O(1) end / O(n) mid | O(1) end / O(n) mid | O(n) / O(1) sorted | O(k) |
| Linked list (singly) | O(n) | O(n) | O(1) head / O(n) mid | O(1) head / O(n) ref | O(n) | O(n) |
| Linked list (doubly) | O(n) | O(n) | O(1) any with ref | O(1) with ref | O(n) | O(n) |
| Stack | O(n) (top only) | O(n) | O(1) push | O(1) pop | n/a | O(n) |
| Queue | O(n) (head only) | O(n) | O(1) enq | O(1) deq | n/a | O(n) |
| Deque | O(1) (block) | O(n) | O(1) ends | O(1) ends | n/a | O(n) |
| Hash table | n/a | O(1) avg, O(n) worst | O(1) avg | O(1) avg | O(n) | O(n) |
| BST (unbalanced) | n/a | O(log n) avg, O(n) worst | same | same | O(log n) avg | O(log n + k) avg |
| AVL / Red-Black | n/a | O(log n) | O(log n) | O(log n) | O(log n) | O(log n + k) |
| B-tree | n/a | O(log_B n) | O(log_B n) | O(log_B n) | O(log_B n) | O(log_B n + k/B) |
| B+tree | n/a | O(log_B n) | O(log_B n) | O(log_B n) | O(log_B n) | O(log_B n + k/B) |
| Binary heap | n/a | O(n) | O(log n) | O(log n) min | O(1) min | O(n) |
| Fibonacci heap | n/a | O(n) | O(1) am | O(log n) am | O(1) min | O(n) |
| Trie | n/a | O(m) | O(m) | O(m) | depth-traverse | O(m + k) prefix |
| Segment tree | O(log n) point | O(log n) | O(log n) update | n/a | O(log n) | O(log n) |
| Fenwick tree (BIT) | n/a | n/a (no search) | O(log n) | n/a | n/a | O(log n) prefix |
| Skip list | O(n) | O(log n) avg | O(log n) avg | O(log n) avg | O(log n) | O(log n + k) |
| Disjoint set | n/a | O(α) find | O(1) make | n/a | n/a | n/a |
| Bloom filter | n/a | O(k) (false-pos) | O(k) | impossible | n/a | n/a |
| Count-min sketch | n/a | O(d) (over-est) | O(d) | n/a | n/a | n/a |
| HyperLogLog | n/a | n/a | O(1) | n/a | n/a | n/a (cardinality only) |

### Sort comparison table

| Sort | Best | Average | Worst | Space | Stable | In-place | Adaptive |
|---|---|---|---|---|---|---|---|
| Bubble | Θ(n) | Θ(n²) | Θ(n²) | O(1) | Yes | Yes | Yes |
| Selection | Θ(n²) | Θ(n²) | Θ(n²) | O(1) | No | Yes | No |
| Insertion | Θ(n) | Θ(n²) | Θ(n²) | O(1) | Yes | Yes | Yes |
| Shell | Θ(n log n) | Θ(n^1.5) | Θ(n^2) | O(1) | No | Yes | Yes |
| Merge | Θ(n log n) | Θ(n log n) | Θ(n log n) | O(n) | Yes | No | No |
| Quick | Θ(n log n) | Θ(n log n) | Θ(n²) | O(log n) | No | Yes | No |
| Heap | Θ(n log n) | Θ(n log n) | Θ(n log n) | O(1) | No | Yes | No |
| Intro | Θ(n log n) | Θ(n log n) | Θ(n log n) | O(log n) | No | Yes | No |
| Tim | Θ(n) | Θ(n log n) | Θ(n log n) | O(n) | Yes | No | Yes |
| Counting | Θ(n + k) | Θ(n + k) | Θ(n + k) | O(n + k) | Yes | No | No |
| Radix (LSD) | Θ(d(n+b)) | Θ(d(n+b)) | Θ(d(n+b)) | O(n + b) | Yes | No | No |
| Bucket | Θ(n + k) | Θ(n + k) | Θ(n²) | O(n + k) | Yes | No | No |

### Graph algorithm complexity table

| Algorithm | Time | Space | Notes |
|---|---|---|---|
| BFS | O(V + E) | O(V) | Unweighted shortest path |
| DFS | O(V + E) | O(V) | Recursion stack |
| Dijkstra (binary heap) | O((V+E) log V) | O(V) | Non-negative weights |
| Dijkstra (Fibonacci) | O(V log V + E) | O(V) | Optimal |
| Bellman-Ford | O(V · E) | O(V) | Negative weights, cycle detection |
| Floyd-Warshall | O(V³) | O(V²) | All-pairs |
| Johnson's | O(V·E + V² log V) | O(V²) | All-pairs, sparse, with negative |
| A* | O(b^d) worst | O(b^d) | Heuristic, admissible |
| Kruskal MST | O(E log E) | O(V + E) | Sparse-friendly |
| Prim MST (binary) | O((V+E) log V) | O(V + E) | |
| Prim MST (Fibonacci) | O(E + V log V) | O(V + E) | Dense graphs |
| Topological sort | O(V + E) | O(V) | DAG |
| Tarjan SCC | O(V + E) | O(V) | Single DFS |
| Kosaraju SCC | O(V + E) | O(V) | Two DFS, transpose |
| Edmonds-Karp max flow | O(V · E²) | O(V + E) | |
| Dinic max flow | O(V² · E) | O(V + E) | O(E√V) on bipartite |
| Hopcroft-Karp matching | O(E · √V) | O(V + E) | Bipartite |
| Hungarian assignment | O(V³) | O(V²) | |

### String algorithm complexity table

| Algorithm | Preprocess | Search | Notes |
|---|---|---|---|
| Naive | 0 | O(n · m) | |
| KMP | O(m) | O(n + m) | Failure function |
| Z-algorithm | O(m) | O(n + m) | |
| Boyer-Moore | O(m + Σ) | O(n / m) best | Sublinear average |
| Rabin-Karp | O(m) | O(n + m) avg | Multi-pattern via hash family |
| Aho-Corasick | O(Σ |Pᵢ|) | O(n + occ) | Multi-pattern |
| Suffix tree (build) | O(n) | O(m) per query | Big constant |
| Suffix array (build) | O(n) (SA-IS) | O(m + log n) | Smaller memory |
| FM-index | O(n log n) | O(m) | Compressed; bowtie/bwa |
| Edit distance (DP) | n/a | O(n · m) | Levenshtein |
| Hunt-Szymanski LCS | n/a | O((n + r) log n) | r = match pairs |

## Idioms

```bash
# Bottleneck is sort?
#   → Use the language's default (Timsort, introsort) — already optimal.
#   → If integer keys with bounded range: radix sort beats it.
```

```bash
# Set membership?
#   → Hash set for exact, in-memory.
#   → Bloom filter when memory > correctness (URL crawl, cache check).
#   → Cuckoo filter if you need delete + Bloom-like FPR.
```

```bash
# Shortest path with non-negative weights?
#   → Dijkstra with binary heap (or Fibonacci on dense graphs).
```

```bash
# Shortest path with possible negative weights?
#   → Bellman-Ford (or SPFA queue variant).
#   → If negative cycle is possible, Bellman-Ford detects.
```

```bash
# Shortest path on a DAG (any weights)?
#   → Topo-sort + linear relaxation. O(V + E). Beats Dijkstra and Bellman-Ford.
```

```bash
# All-pairs shortest path?
#   → Floyd-Warshall O(V³) for dense; Johnson's O(V·E + V² log V) for sparse.
```

```bash
# Sliding-window aggregate (sum/min/max/...)?
#   → Monotonic deque for min/max in O(1) amortized.
#   → Prefix sums for sum (O(1) per query).
```

```bash
# Range sum + point update?
#   → Fenwick tree (BIT) — smaller constant than segment tree.
```

```bash
# Range update + range query?
#   → Segment tree with lazy propagation.
```

```bash
# k-th smallest in array?
#   → quickselect O(n) average; nth_element in C++; introselect.
#   → If many queries on same array: sort once, O(log n) lookup.
```

```bash
# Top-k from a stream?
#   → Min-heap of size k: insert if > min, then extract-min.
#   → Total O(n log k).
```

```bash
# Distinct count from a stream?
#   → HyperLogLog. ~1 KB for 1% error.
#   → Linear counting for exact small cardinalities (< few million).
```

```bash
# Heavy hitters from a stream?
#   → Misra-Gries for ε-approximate top-k.
#   → Count-min sketch for arbitrary frequency queries.
```

```bash
# Substring search, single pattern?
#   → libc strstr / memmem (Two-Way / Boyer-Moore variant).
#   → KMP for guaranteed worst case.
```

```bash
# Substring search, many patterns?
#   → Aho-Corasick. Build automaton once, match in O(n + occ).
```

```bash
# Need a sorted associative container?
#   → Java TreeMap, C++ std::map, Python sortedcontainers.SortedDict.
#   → Skip list or red-black tree underneath.
```

```bash
# Need a hash map with predictable performance under attack?
#   → SipHash-keyed, or randomized seed at process startup.
#   → Python and Ruby do this by default.
```

```bash
# Disk-resident sorted index?
#   → B+tree. Universal database default. SQLite, Postgres, MySQL InnoDB.
```

```bash
# In-memory write-heavy index?
#   → LSM-tree (RocksDB) — turns random writes into sequential.
#   → Skip-list MemTable + sorted SSTables on disk.
```

```bash
# Geometric closest-point queries?
#   → k-d tree O(log n) average, O(n) worst.
#   → Ball tree if metric not Euclidean.
#   → Approximate: locality-sensitive hashing.
```

```bash
# Nearest-neighbour search in high dimension?
#   → "Curse of dimensionality" — exact methods degrade to O(n).
#   → ANN: HNSW, IVFFLAT, ScaNN, FAISS — sub-linear with controlled recall.
```

```bash
# Union-find / connectivity queries on a static graph?
#   → Disjoint set with path compression + union by rank — α(n) ≈ O(1).
#   → Tarjan's offline LCA reduces to disjoint set.
```

## See Also

- algorithm-analysis
- complexity-theory
- distributed-systems
- graph-theory
- hash-function-theory

## References

- Cormen, Leiserson, Rivest, Stein. *Introduction to Algorithms* (CLRS). 4th ed., MIT Press, 2022. The canonical reference; chapters 3-4 (asymptotic, recurrences), 17 (amortized), 25 (all-pairs SP), 26 (max flow), 34-35 (NP, approx).
- Sedgewick, Wayne. *Algorithms*. 4th ed., Addison-Wesley, 2011. More implementation-oriented; companion site at algs4.cs.princeton.edu has Java code and analysis.
- Knuth. *The Art of Computer Programming* (TAOCP). Vols 1-4A. The deepest treatment of algorithm analysis; vol 3 is the sorting & searching reference.
- Papadimitriou, Vazirani, Dasgupta. *Algorithms*. McGraw-Hill, 2008. Free PDF online; concise alternative to CLRS.
- Tarjan. *Data Structures and Network Algorithms*. SIAM, 1983. Original amortized analysis treatment.
- Frigo, Leiserson, Prokop, Ramachandran. "Cache-Oblivious Algorithms" (FOCS 1999).
- Cormode, Muthukrishnan. "An Improved Data Stream Summary: The Count-Min Sketch and its Applications" (LATIN 2004).
- Flajolet, Fusy, Gandouet, Meunier. "HyperLogLog: the analysis of a near-optimal cardinality estimation algorithm" (AofA 2007).
- competitive-programming.io / cp-algorithms.com — hands-on reference for graph and string algorithms with idiomatic implementations.
- Garey, Johnson. *Computers and Intractability: A Guide to the Theory of NP-Completeness*. W.H. Freeman, 1979. The reference for NP-completeness reductions.
- Vazirani. *Approximation Algorithms*. Springer, 2003. Free PDF; covers PTAS, FPTAS, LP-rounding, primal-dual.
- Mitzenmacher, Upfal. *Probability and Computing*. 2nd ed., Cambridge, 2017. Randomized algorithms and probabilistic analysis.
- Borodin, El-Yaniv. *Online Computation and Competitive Analysis*. Cambridge, 1998.
- Williams, Xu, Xu, Zhou. "New bounds for matrix multiplication: from alpha to omega" (SODA 2024). Current world-record matrix-multiply exponent.
- Harvey, van der Hoeven. "Integer multiplication in time O(n log n)" (Annals of Math, 2021). Settles the long-conjectured optimal integer-multiply.
- Williams. *On the Difference Between ACC and AC* + various circuit-complexity surveys. The frontier of complexity-theoretic lower bounds.
