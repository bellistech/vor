# Algorithm Analysis (Asymptotic Bounds, Recurrences, and Proof Techniques)

A practitioner's reference for analyzing algorithm correctness and efficiency: asymptotic notation, solving recurrences, amortized analysis, and algorithm design paradigms with their analysis techniques.

## Asymptotic Notation

### Definitions

```
Big-O (upper bound):
  f(n) = O(g(n))  iff  exists c > 0, n0 > 0 such that
                        f(n) <= c * g(n)  for all n >= n0

Big-Omega (lower bound):
  f(n) = Omega(g(n))  iff  exists c > 0, n0 > 0 such that
                            f(n) >= c * g(n)  for all n >= n0

Big-Theta (tight bound):
  f(n) = Theta(g(n))  iff  f(n) = O(g(n))  AND  f(n) = Omega(g(n))

Little-o (strict upper bound):
  f(n) = o(g(n))  iff  for ALL c > 0, exists n0 such that
                        f(n) < c * g(n)  for all n >= n0
  Equivalently: lim_{n->inf} f(n)/g(n) = 0

Little-omega (strict lower bound):
  f(n) = omega(g(n))  iff  for ALL c > 0, exists n0 such that
                            f(n) > c * g(n)  for all n >= n0
  Equivalently: lim_{n->inf} f(n)/g(n) = infinity
```

### Common Growth Rates

```
O(1) < O(log n) < O(sqrt(n)) < O(n) < O(n log n) < O(n^2) < O(n^3) < O(2^n) < O(n!)

Concrete examples:
  O(1)        Hash table lookup (amortized)
  O(log n)    Binary search
  O(n)        Linear scan
  O(n log n)  Merge sort, optimal comparison sort
  O(n^2)      Insertion sort, naive matrix multiply
  O(n^3)      Standard matrix multiply
  O(2^n)      Subset enumeration
  O(n!)       Permutation enumeration
```

### Properties

```
Transitivity:  f = O(g) and g = O(h)  =>  f = O(h)
Reflexivity:   f = O(f), f = Theta(f)
Symmetry:      f = Theta(g)  iff  g = Theta(f)
Transpose:     f = O(g)  iff  g = Omega(f)
               f = o(g)  iff  g = omega(f)

Sum rule:      O(f + g) = O(max(f, g))
Product rule:  O(f * g) = O(f) * O(g)
```

## Recurrence Relations

### Master Theorem

```
For recurrences of the form: T(n) = a * T(n/b) + f(n)
where a >= 1, b > 1, f(n) asymptotically positive.

Compare f(n) with n^(log_b(a)):

Case 1: f(n) = O(n^(log_b(a) - epsilon))  for some epsilon > 0
         => T(n) = Theta(n^(log_b(a)))

Case 2: f(n) = Theta(n^(log_b(a)) * log^k(n))  for k >= 0
         => T(n) = Theta(n^(log_b(a)) * log^(k+1)(n))

Case 3: f(n) = Omega(n^(log_b(a) + epsilon))  for some epsilon > 0
         AND a * f(n/b) <= c * f(n)  for some c < 1  (regularity)
         => T(n) = Theta(f(n))
```

### Master Theorem Examples

| Recurrence | a | b | n^(log_b a) | Case | Result |
|---|---|---|---|---|---|
| T(n) = 2T(n/2) + n | 2 | 2 | n | 2 (k=0) | Theta(n log n) |
| T(n) = 4T(n/2) + n | 4 | 2 | n^2 | 1 | Theta(n^2) |
| T(n) = 2T(n/2) + n^2 | 2 | 2 | n | 3 | Theta(n^2) |
| T(n) = T(n/2) + 1 | 1 | 2 | 1 | 2 (k=0) | Theta(log n) |
| T(n) = 9T(n/3) + n | 9 | 3 | n^2 | 1 | Theta(n^2) |
| T(n) = 3T(n/4) + n log n | 3 | 4 | n^0.79 | 3 | Theta(n log n) |
| T(n) = 7T(n/2) + n^2 | 7 | 2 | n^2.81 | 1 | Theta(n^2.81) |

### Akra-Bazzi Theorem

```
Generalizes the master theorem. For recurrences of the form:

  T(n) = sum_{i=1}^{k} a_i * T(b_i * n) + g(n)

where a_i > 0, 0 < b_i < 1, and g(n) satisfies a polynomial growth condition.

Find p such that: sum_{i=1}^{k} a_i * b_i^p = 1

Then: T(n) = Theta(n^p * (1 + integral from 1 to n of g(u) / u^(p+1) du))

Handles: unequal subproblem sizes, non-integer divisions, floors/ceilings.
```

### Recursion Tree Method

```
Steps:
1. Draw the tree: each node = cost at that level of recursion
2. Sum costs at each level
3. Count the number of levels (typically log_b(n))
4. Sum across all levels

Example: T(n) = 3T(n/4) + cn^2

Level 0:  cn^2                                = cn^2
Level 1:  3 * c(n/4)^2                        = (3/16) * cn^2
Level 2:  9 * c(n/16)^2                       = (3/16)^2 * cn^2
  ...
Level i:  3^i * c(n/4^i)^2                    = (3/16)^i * cn^2

Depth: log_4(n) levels
Leaf cost: 3^(log_4 n) = n^(log_4 3) ~ n^0.79

Total = cn^2 * sum_{i=0}^{log_4 n} (3/16)^i
      = O(cn^2)  [geometric series with ratio < 1]
```

## Amortized Analysis

### Aggregate Method

```
Idea: Total cost of n operations / n = amortized cost per operation.

Example — Dynamic array (vector) with doubling:
  n insertions, array doubles at sizes 1, 2, 4, 8, ..., 2^k

  Total copy cost = 1 + 2 + 4 + ... + 2^k = 2^(k+1) - 1 < 2n
  Total insert cost = n (one per element)
  Total = n + 2n = 3n

  Amortized cost per insertion = 3n / n = O(1)
```

### Accounting Method

```
Idea: Charge each operation an "amortized cost." Overcharges build credit.
      Credit must never go negative.

Example — Dynamic array:
  Charge each insertion $3:
    $1 pays for the insertion itself
    $1 saved for copying this element later
    $1 saved for copying an element already in the array

  When doubling from size k to 2k:
    k elements need copying, each costs $1
    The k/2 elements inserted since last doubling each saved $2
    Total credit available: k/2 * $2 = $k >= cost of copying

  Credit >= 0 always.  Amortized cost per insert = $3 = O(1).
```

### Potential Method

```
Idea: Define a potential function Phi on the data structure state.
      Amortized cost = actual cost + Delta(Phi)

  Requirements: Phi(D0) = 0,  Phi(Di) >= 0 for all i.

Example — Dynamic array, Phi(D) = 2 * num_elements - capacity:
  Insert (no resize):
    actual cost = 1
    Delta(Phi) = 2 * 1 - 0 = 2
    amortized = 1 + 2 = 3

  Insert (with resize from k to 2k):
    actual cost = 1 + k  (insert + copy k elements)
    Phi_before = 2k - k = k
    Phi_after  = 2(k+1) - 2k = 2
    Delta(Phi) = 2 - k
    amortized = (1 + k) + (2 - k) = 3

  Amortized cost = O(1) per insertion.
```

## Divide and Conquer

### Key Algorithms

| Algorithm | Recurrence | Result | Naive |
|---|---|---|---|
| Merge sort | T(n) = 2T(n/2) + O(n) | O(n log n) | O(n^2) |
| Karatsuba multiplication | T(n) = 3T(n/2) + O(n) | O(n^1.585) | O(n^2) |
| Strassen matrix multiply | T(n) = 7T(n/2) + O(n^2) | O(n^2.807) | O(n^3) |
| Closest pair of points | T(n) = 2T(n/2) + O(n) | O(n log n) | O(n^2) |
| Select (median of medians) | T(n) = T(n/5) + T(7n/10) + O(n) | O(n) | O(n log n) |

### Pattern

```
1. Divide the problem into a subproblems of size n/b
2. Conquer each subproblem recursively
3. Combine solutions in f(n) time

Key insight for Karatsuba:
  (a + b*x)(c + d*x) = ac + ((a+b)(c+d) - ac - bd)*x + bd*x^2
  3 multiplications instead of 4.

Key insight for Strassen:
  2x2 block matrix multiply with 7 multiplications instead of 8.
  Saves one multiplication per recursive level.
```

## Greedy Algorithms

### When Greedy Works

```
Two properties guarantee greedy correctness:

1. Greedy choice property:
   A globally optimal solution can be arrived at by making a
   locally optimal (greedy) choice at each step.

2. Optimal substructure:
   An optimal solution contains within it optimal solutions
   to subproblems.
```

### Matroid Theory (Brief)

```
A matroid M = (E, I) where:
  - E is a finite ground set
  - I is a family of "independent" subsets satisfying:
    (I1) Empty set is in I
    (I2) If A in I and B subset of A, then B in I  (hereditary)
    (I3) If A, B in I and |A| < |B|, then exists x in B\A
         such that A union {x} in I  (exchange property)

Theorem (Rado-Edmonds):
  The greedy algorithm finds a maximum-weight independent set
  in any weighted matroid.

Examples of matroids:
  - Graphic matroid (forests of a graph)
  - Linear matroid (linearly independent vectors)
  - Partition matroid
```

### Exchange Argument

```
Proof technique for greedy correctness:
1. Let G = greedy solution, O = optimal solution
2. Show you can "exchange" elements of O with elements of G
   without worsening the objective
3. Eventually transform O into G, proving G is optimal

Classic application: Activity selection, Huffman coding,
                     Kruskal's MST
```

## Dynamic Programming

### Requirements

```
1. Optimal substructure:
   An optimal solution to the problem contains optimal
   solutions to subproblems.

2. Overlapping subproblems:
   The recursive solution revisits the same subproblems
   many times.

Without (1): DP gives wrong answers.
Without (2): DP works but offers no speedup over divide-and-conquer.
```

### Memoization vs Tabulation

```
Top-down (memoization):
  - Recursive with a cache (hash map or array)
  - Only solves subproblems actually needed
  - Natural to write but may hit stack depth limits
  - Example: fib(n) with memo dictionary

Bottom-up (tabulation):
  - Iterative, fills table from base cases upward
  - Solves all subproblems in dependency order
  - Usually more space-efficient (can discard old rows)
  - No recursion overhead
  - Example: fib[0]=0, fib[1]=1, fib[i]=fib[i-1]+fib[i-2]

Time complexity: identical (both avoid recomputation).
Space: tabulation can often reduce from O(n^2) to O(n) or O(1).
```

### Classic DP Problems

| Problem | Subproblem | Time | Space |
|---|---|---|---|
| Fibonacci | F(i) = F(i-1) + F(i-2) | O(n) | O(1) |
| LCS | LCS(i,j) of prefixes | O(mn) | O(min(m,n)) |
| Edit distance | ED(i,j) of prefixes | O(mn) | O(min(m,n)) |
| Knapsack (0/1) | max value with capacity c, items 1..i | O(nW) | O(W) |
| Matrix chain | min multiplications for A_i..A_j | O(n^3) | O(n^2) |
| Shortest paths | dist(v, k hops) | O(VE) | O(V) |

## Randomized Algorithms

### Classification

```
Las Vegas algorithms:
  - Always produce correct output
  - Running time is a random variable
  - Example: randomized quicksort (expected O(n log n), always correct)

Monte Carlo algorithms:
  - Fixed running time
  - Output may be incorrect with bounded probability
  - One-sided error: only errs in one direction (RP, co-RP)
  - Two-sided error: may err in either direction (BPP)
  - Example: Miller-Rabin primality test (may say "prime" for composite)
```

### Expected Analysis

```
Randomized quicksort:
  E[comparisons] = sum_{i<j} Pr[z_i compared to z_j]
                 = sum_{i<j} 2/(j - i + 1)
                 = O(n * H_n) = O(n log n)

  where H_n = 1 + 1/2 + 1/3 + ... + 1/n (harmonic number)

Randomized selection (quickselect):
  E[T(n)] = O(n)  (geometric decrease in expected subproblem size)

Skip list:
  Expected search, insert, delete: O(log n)
  Expected space: O(n)
```

## Lower Bounds

### Decision Tree Model

```
Any comparison-based sorting algorithm can be modeled as
a binary decision tree:
  - Internal nodes = comparisons (a_i < a_j?)
  - Leaves = output permutations
  - n! possible outputs => tree has >= n! leaves
  - Height >= log_2(n!) = Omega(n log n)  (Stirling's approximation)

Therefore: comparison-based sorting requires Omega(n log n) comparisons.
```

### Adversary Argument

```
Technique: Construct an adversary that forces any algorithm
to perform many operations, regardless of strategy.

Example — Finding the maximum:
  Adversary maintains a set of "potential winners."
  Each comparison eliminates at most one element.
  Must eliminate n-1 elements => at least n-1 comparisons.

Example — Merging two sorted lists of size n:
  Adversary argument shows 2n-1 comparisons necessary.
```

### Information-Theoretic Lower Bound

```
If there are N possible outputs, any algorithm that
distinguishes between them needs at least ceil(log_2(N))
binary decisions (comparisons, bits read, etc.).

Applications:
  Sorting:        N = n!,  bound = log(n!) = Omega(n log n)
  Searching:      N = n+1, bound = log(n+1) = Omega(log n)
  Selection:      More subtle — n - 1 for max, tighter for median
```

## Key Figures

| Name | Contribution | Year |
|---|---|---|
| Donald Knuth | "The Art of Computer Programming," O-notation popularization | 1968 |
| Robert Tarjan | Amortized analysis, splay trees, union-find analysis | 1985 |
| Thomas Cormen | "Introduction to Algorithms" (CLRS) | 1990 |
| Ronald Rivest | Co-author of CLRS, RSA algorithm analysis | 1990 |
| Anatolii Karatsuba | Fast integer multiplication (O(n^1.585)) | 1960 |
| Volker Strassen | Fast matrix multiplication (O(n^2.807)) | 1969 |
| Jon Bentley | Master theorem (with Haken, Saxe), programming pearls | 1980 |

## Tips

- Always specify the model of computation when claiming a lower bound (comparison-based, algebraic, etc.)
- The master theorem has gaps: if f(n) falls between cases, use Akra-Bazzi or the recursion tree
- Amortized O(1) does not mean every operation is O(1) -- individual operations can be expensive
- DP is not always better than greedy -- if the greedy choice property holds, greedy is simpler and faster
- For randomized algorithms, distinguish expected time (Las Vegas) from error probability (Monte Carlo)
- Lower bounds are only as strong as their model -- non-comparison sorts (radix, counting) beat n log n
- When analyzing divide-and-conquer, always check the regularity condition for Master Theorem Case 3

## See Also

- `detail/cs-theory/algorithm-analysis.md` -- master theorem proof, potential method examples, matroid theory, adversary arguments
- `sheets/cs-theory/complexity-theory.md` -- P, NP, NP-completeness, reductions
- `sheets/cs-theory/computability-theory.md` -- decidability, halting problem, Rice's theorem
- `sheets/cs-theory/information-theory.md` -- entropy, Kolmogorov complexity
- `sheets/algorithms/sorting.md` -- comparison sorts, lower bounds in practice

## References

- "Introduction to Algorithms" by Cormen, Leiserson, Rivest, and Stein (CLRS, 4th ed., 2022)
- "The Art of Computer Programming" by Donald Knuth (Vols. 1-4A)
- "Algorithm Design" by Kleinberg and Tardos (2005)
- Akra and Bazzi, "On the Solution of Linear Recurrence Equations" (Computational Optimization and Applications, 1998)
- Tarjan, "Amortized Computational Complexity" (SIAM J. Algebraic Discrete Methods, 1985)
- Karatsuba and Ofman, "Multiplication of Many-Digital Numbers by Automatic Computers" (1962)
- Strassen, "Gaussian Elimination is not Optimal" (Numerische Mathematik, 1969)
