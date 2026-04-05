# The Mathematics of Algorithm Analysis -- Recurrences, Amortization, and the Limits of Computation

> *Algorithm analysis transforms the art of programming into a science of provable guarantees. From solving recurrences to proving lower bounds, the techniques developed over the past sixty years give us the language to reason precisely about computational cost.*

---

## 1. The Master Theorem (Proof Sketch)

### The Problem

Prove the master theorem for recurrences of the form $T(n) = aT(n/b) + f(n)$, establishing the three cases that determine asymptotic behavior based on the relationship between $f(n)$ and $n^{\log_b a}$.

### The Formula

**Theorem.** Let $a \geq 1$, $b > 1$ be constants, and let $f(n)$ be a non-negative function. Define:

$$T(n) = aT(n/b) + f(n)$$

with $T(1) = \Theta(1)$. Then $T(n)$ is bounded asymptotically as follows:

**Case 1.** If $f(n) = O(n^{\log_b a - \epsilon})$ for some $\epsilon > 0$, then $T(n) = \Theta(n^{\log_b a})$.

**Case 2.** If $f(n) = \Theta(n^{\log_b a} \log^k n)$ for some $k \geq 0$, then $T(n) = \Theta(n^{\log_b a} \log^{k+1} n)$.

**Case 3.** If $f(n) = \Omega(n^{\log_b a + \epsilon})$ for some $\epsilon > 0$, and if $af(n/b) \leq cf(n)$ for some $c < 1$ and sufficiently large $n$ (regularity condition), then $T(n) = \Theta(f(n))$.

### Proof Sketch

Expand the recurrence by iterating:

$$T(n) = a^k T(n/b^k) + \sum_{j=0}^{k-1} a^j f(n/b^j)$$

The recursion bottoms out at $k = \log_b n$ levels, giving $a^{\log_b n} = n^{\log_b a}$ leaves each contributing $\Theta(1)$.

$$T(n) = \Theta(n^{\log_b a}) + \sum_{j=0}^{\log_b n - 1} a^j f(n/b^j)$$

The total cost is the sum of two terms: the leaf cost $\Theta(n^{\log_b a})$ and the internal node cost $\sum_{j=0}^{\log_b n - 1} a^j f(n/b^j)$.

**Case 1 analysis.** If $f(n) = O(n^{\log_b a - \epsilon})$, then:

$$a^j f(n/b^j) = a^j \cdot O\!\left(\left(\frac{n}{b^j}\right)^{\log_b a - \epsilon}\right) = O\!\left(n^{\log_b a - \epsilon} \cdot \left(\frac{a}{b^{\log_b a - \epsilon}}\right)^j\right)$$

Since $a / b^{\log_b a - \epsilon} = b^{\epsilon}$, each level grows by a factor of $b^{\epsilon} > 1$. The geometric series is dominated by its last term, but the leaf cost $\Theta(n^{\log_b a})$ dominates the sum. Therefore $T(n) = \Theta(n^{\log_b a})$.

**Case 2 analysis.** If $f(n) = \Theta(n^{\log_b a} \log^k n)$, then each level contributes $\Theta(n^{\log_b a} \log^k(n/b^j))$, and the sum of $\log_b n$ such terms yields $\Theta(n^{\log_b a} \log^{k+1} n)$.

**Case 3 analysis.** The regularity condition $af(n/b) \leq cf(n)$ ensures the series $\sum a^j f(n/b^j)$ is a decreasing geometric series. The root cost $f(n)$ dominates, giving $T(n) = \Theta(f(n))$.

### Gap Between Cases

The master theorem does not cover all functions $f(n)$. If $f(n) = \Theta(n^{\log_b a} / \log n)$, it falls between Cases 1 and 2. The Akra-Bazzi theorem (Section 2) handles such gaps.

---

## 2. The Akra-Bazzi Theorem

### The Problem

Solve recurrences with unequal subproblem sizes and non-integer coefficients that the master theorem cannot handle, such as $T(n) = T(n/3) + T(2n/3) + n$.

### The Formula

**Theorem (Akra-Bazzi 1998).** Consider the recurrence:

$$T(n) = \sum_{i=1}^{k} a_i T(b_i n + h_i(n)) + g(n)$$

where $a_i > 0$, $0 < b_i < 1$, $|h_i(n)| = O(n / \log^2 n)$, and $g(n)$ satisfies a polynomial growth condition. Let $p$ be the unique real number such that:

$$\sum_{i=1}^{k} a_i b_i^p = 1$$

Then:

$$T(n) = \Theta\!\left(n^p \left(1 + \int_1^n \frac{g(u)}{u^{p+1}} \, du\right)\right)$$

### Worked Example

Consider $T(n) = T(n/3) + T(2n/3) + n$.

Here $a_1 = 1, b_1 = 1/3, a_2 = 1, b_2 = 2/3$, and $g(n) = n$.

Find $p$ such that $(1/3)^p + (2/3)^p = 1$. Testing $p = 1$: $1/3 + 2/3 = 1$. So $p = 1$.

$$T(n) = \Theta\!\left(n \left(1 + \int_1^n \frac{u}{u^2} \, du\right)\right) = \Theta\!\left(n \left(1 + \int_1^n \frac{1}{u} \, du\right)\right) = \Theta(n(1 + \ln n)) = \Theta(n \log n)$$

This matches the expected result: an unbalanced binary recursion with linear work per level takes $\Theta(n \log n)$, just like merge sort.

### Why Akra-Bazzi Subsumes the Master Theorem

For $T(n) = aT(n/b) + n^c$: solve $a(1/b)^p = 1$, giving $p = \log_b a$. The integral then yields the master theorem's three cases depending on whether $c < p$, $c = p$, or $c > p$.

---

## 3. The Potential Method (Worked Example: Dynamic Array)

### The Problem

Prove that insertions into a dynamically resizing array (which doubles in capacity when full) have amortized $O(1)$ cost using the potential method.

### The Formula

Let $D_i$ denote the state of the array after the $i$-th operation. Define:

$$\Phi(D_i) = 2n_i - c_i$$

where $n_i$ is the number of elements stored and $c_i$ is the current capacity. We require $\Phi(D_0) = 0$ and $\Phi(D_i) \geq 0$ for all $i$.

Initially $n_0 = 0, c_0 = 0$, so $\Phi(D_0) = 0$. Since the array doubles when full ($n_i = c_i$ triggers resize to $2c_i$), we always have $n_i \geq c_i / 2$, hence $\Phi(D_i) = 2n_i - c_i \geq 0$.

The amortized cost of the $i$-th operation is:

$$\hat{c}_i = c_i^{\text{actual}} + \Phi(D_i) - \Phi(D_{i-1})$$

**Case 1: Insert without resize** ($n_{i-1} < c_{i-1}$).

Actual cost: $c_i^{\text{actual}} = 1$.

$$\Phi(D_i) - \Phi(D_{i-1}) = (2(n_{i-1}+1) - c_{i-1}) - (2n_{i-1} - c_{i-1}) = 2$$

$$\hat{c}_i = 1 + 2 = 3$$

**Case 2: Insert with resize** ($n_{i-1} = c_{i-1}$, capacity doubles to $2c_{i-1}$).

Actual cost: $c_i^{\text{actual}} = 1 + c_{i-1}$ (insert plus copy all elements).

$$\Phi(D_i) = 2(c_{i-1} + 1) - 2c_{i-1} = 2$$

$$\Phi(D_{i-1}) = 2c_{i-1} - c_{i-1} = c_{i-1}$$

$$\hat{c}_i = (1 + c_{i-1}) + (2 - c_{i-1}) = 3$$

In both cases, $\hat{c}_i = 3 = O(1)$.

### Telescoping Argument

The total actual cost is bounded by:

$$\sum_{i=1}^{n} c_i^{\text{actual}} = \sum_{i=1}^{n} \hat{c}_i - \Phi(D_n) + \Phi(D_0) \leq \sum_{i=1}^{n} \hat{c}_i = 3n$$

since $\Phi(D_n) \geq 0$ and $\Phi(D_0) = 0$.

---

## 4. Matroid Theory and Greedy Correctness

### The Problem

Prove that the greedy algorithm finds a maximum-weight basis in any weighted matroid, providing a unifying framework for why greedy algorithms work on problems like minimum spanning trees.

### The Formula

A **matroid** is a pair $M = (E, \mathcal{I})$ where $E$ is a finite ground set and $\mathcal{I} \subseteq 2^E$ is a family of *independent sets* satisfying:

1. $\emptyset \in \mathcal{I}$
2. If $B \in \mathcal{I}$ and $A \subseteq B$, then $A \in \mathcal{I}$ (hereditary property)
3. If $A, B \in \mathcal{I}$ and $|A| < |B|$, then $\exists x \in B \setminus A$ such that $A \cup \{x\} \in \mathcal{I}$ (exchange property)

A **basis** is a maximal independent set. All bases of a matroid have the same cardinality (this follows from the exchange property).

**Weighted matroid problem.** Given weights $w : E \to \mathbb{R}^+$, find an independent set $S \in \mathcal{I}$ maximizing $w(S) = \sum_{e \in S} w(e)$.

**Greedy algorithm:**

```
Sort E by weight in decreasing order: e_1, e_2, ..., e_m
S = empty set
for i = 1 to m:
    if S union {e_i} is in I:
        S = S union {e_i}
return S
```

**Theorem (Rado 1957, Edmonds 1971).** The greedy algorithm produces a maximum-weight independent set in any weighted matroid.

### Proof by Exchange Argument

Let $G = \{g_1, g_2, \ldots, g_k\}$ be the greedy solution (elements in order of selection, so $w(g_1) \geq w(g_2) \geq \cdots$). Let $O = \{o_1, o_2, \ldots, o_k\}$ be any maximum-weight independent set (also sorted by weight). We show $w(g_i) \geq w(o_i)$ for all $i$.

Suppose for contradiction that $w(g_i) < w(o_i)$ for some minimal $i$. Consider $A = \{g_1, \ldots, g_{i-1}\}$ and $B = \{o_1, \ldots, o_i\}$. Both are independent, and $|A| < |B|$. By the exchange property, there exists $o_j \in B \setminus A$ such that $A \cup \{o_j\} \in \mathcal{I}$.

Since $o_j \in B$, we have $w(o_j) \geq w(o_i) > w(g_i)$. But the greedy algorithm at step $i$ chose $g_i$ -- the heaviest element that could extend $A$ to remain independent. Since $A \cup \{o_j\}$ is independent and $w(o_j) > w(g_i)$, greedy should have chosen $o_j$ (or something at least as heavy), contradiction.

Therefore $w(g_i) \geq w(o_i)$ for all $i$, and $w(G) = \sum w(g_i) \geq \sum w(o_i) = w(O)$.

### Application: Kruskal's Algorithm

The *graphic matroid* of a graph $G = (V, E)$ has ground set $E$ and independent sets = forests (acyclic edge subsets). This is indeed a matroid (the exchange property follows from the fact that a forest on $k$ edges has $|V| - k$ components). Kruskal's algorithm is exactly the greedy algorithm on this matroid with edge weights negated (minimizing weight = maximizing negative weight). Therefore Kruskal's algorithm correctly finds the minimum spanning tree.

---

## 5. Strassen's Algorithm Analysis

### The Problem

Analyze Strassen's algorithm for matrix multiplication and prove it achieves $O(n^{\log_2 7})$ arithmetic operations, improving on the naive $O(n^3)$.

### The Formula

The naive algorithm for multiplying two $n \times n$ matrices requires $n^3$ multiplications and $n^3 - n^2$ additions, for $\Theta(n^3)$ total operations.

Strassen observed that two $2 \times 2$ matrices can be multiplied using only 7 multiplications (instead of 8) at the cost of more additions:

$$\begin{pmatrix} A & B \\ C & D \end{pmatrix} \begin{pmatrix} E & F \\ G & H \end{pmatrix} = \begin{pmatrix} P_5 + P_4 - P_2 + P_6 & P_1 + P_2 \\ P_3 + P_4 & P_1 + P_5 - P_3 - P_7 \end{pmatrix}$$

where the seven products are:

$$P_1 = A(F - H), \quad P_2 = (A + B)H, \quad P_3 = (C + D)E$$
$$P_4 = D(G - E), \quad P_5 = (A + D)(E + H)$$
$$P_6 = (B - D)(G + H), \quad P_7 = (A - C)(E + F)$$

Each $P_i$ involves one multiplication of $n/2 \times n/2$ matrices plus $O(n^2)$ additions.

### Recurrence and Solution

$$T(n) = 7T(n/2) + \Theta(n^2)$$

Applying the master theorem with $a = 7$, $b = 2$:

$$n^{\log_b a} = n^{\log_2 7} \approx n^{2.807}$$

Since $f(n) = \Theta(n^2) = O(n^{\log_2 7 - \epsilon})$ with $\epsilon \approx 0.807$, we are in Case 1:

$$T(n) = \Theta(n^{\log_2 7}) \approx \Theta(n^{2.807})$$

### Context and Subsequent Work

Strassen's 1969 result was groundbreaking: it disproved the conjecture that $\Omega(n^3)$ was optimal for matrix multiplication. The *exponent of matrix multiplication* $\omega$ is defined as the infimum over all $\alpha$ such that two $n \times n$ matrices can be multiplied in $O(n^{\alpha})$ operations.

| Year | Authors | Exponent |
|------|---------|----------|
| pre-1969 | Naive | 3.000 |
| 1969 | Strassen | 2.807 |
| 1978 | Pan | 2.796 |
| 1987 | Coppersmith-Winograd | 2.376 |
| 2012 | Williams | 2.3728 |
| 2024 | Duan-Wu-Zhou | 2.371339 |

The theoretical lower bound is $\omega \geq 2$ (every entry of the output must be written). Whether $\omega = 2$ remains a major open problem.

---

## 6. The Adversary Lower Bound for Comparison Sorting

### The Problem

Prove that any comparison-based sorting algorithm requires $\Omega(n \log n)$ comparisons in the worst case.

### The Formula

**Theorem.** Any deterministic comparison-based sorting algorithm requires at least $\lceil \log_2(n!) \rceil$ comparisons in the worst case.

**Decision tree model.** Any comparison sort on $n$ elements can be represented as a binary tree where:

- Each internal node represents a comparison $a_i \leq a_j$
- The left subtree corresponds to "yes," the right to "no"
- Each leaf is labeled with a permutation $\pi \in S_n$ representing the sorted order

Since the algorithm must correctly sort any permutation of $n$ distinct elements, the tree must have at least $n!$ leaves (one for each permutation). A binary tree with $L$ leaves has height at least $\lceil \log_2 L \rceil$.

$$\text{worst-case comparisons} \geq \lceil \log_2(n!) \rceil$$

### Stirling's Approximation

$$n! = \sqrt{2\pi n} \left(\frac{n}{e}\right)^n \left(1 + O\!\left(\frac{1}{n}\right)\right)$$

Therefore:

$$\log_2(n!) = n \log_2 n - n \log_2 e + \frac{1}{2}\log_2(2\pi n) + O\!\left(\frac{1}{n}\right) = n \log_2 n - \Theta(n)$$

This gives the tight bound:

$$\text{worst-case comparisons} \geq n \log_2 n - 1.443n + O(\log n)$$

### Adversary Argument Version

An adversary maintains a partial order consistent with all comparisons answered so far. When the algorithm queries $a_i \leq a_j$:

1. The adversary counts how many linear extensions are consistent with "$a_i \leq a_j$" ($L_{\text{yes}}$) and with "$a_i > a_j$" ($L_{\text{no}}$).
2. The adversary answers whichever option leaves more consistent linear extensions.

After each comparison, at least half the remaining linear extensions survive. Starting from $n!$ linear extensions, after $k$ comparisons at least $n! / 2^k$ remain. The algorithm can only terminate when exactly one linear extension remains:

$$\frac{n!}{2^k} \leq 1 \implies k \geq \log_2(n!)$$

### Optimality of Merge Sort

Merge sort uses at most $n \lceil \log_2 n \rceil - 2^{\lceil \log_2 n \rceil} + 1$ comparisons. The Ford-Johnson merge-insertion sort achieves the information-theoretic lower bound $\lceil \log_2(n!) \rceil$ for small $n$ and is conjectured to be optimal.

---

## 7. Smoothed Analysis

### The Problem

Explain why some algorithms with poor worst-case complexity perform well in practice. Smoothed analysis, introduced by Spielman and Teng (2001), provides a framework that interpolates between worst-case and average-case analysis.

### The Formula

Let $A$ be an algorithm, and let $\sigma > 0$ be a perturbation parameter. The **smoothed complexity** of $A$ is:

$$C_{\sigma}^{\text{smooth}}(n) = \max_{\text{input } x} \; \mathbb{E}_{\text{perturbation } \tilde{x}}[T(A, \tilde{x})]$$

where $\tilde{x}$ is obtained by adding a Gaussian perturbation of magnitude $\sigma$ to each coordinate of $x$.

An algorithm has **polynomial smoothed complexity** if $C_{\sigma}^{\text{smooth}}(n)$ is polynomial in $n$ and $1/\sigma$.

### The Simplex Method

The motivating application was the simplex method for linear programming:

- **Worst case:** Klee and Minty (1972) showed that the simplex method can take exponential time ($2^n$ pivots on specific polytopes).
- **Practice:** The simplex method is extremely fast on real-world instances.

**Theorem (Spielman-Teng 2004).** The shadow-vertex simplex method has smoothed polynomial complexity. Specifically, for an LP $\max\{c^T x : Ax \leq b\}$ where $A$ is an $m \times n$ matrix with entries perturbed by Gaussian noise of variance $\sigma^2$:

$$\mathbb{E}[\text{number of pivots}] = O\!\left(\frac{n^{13/4} m^{1/4}}{\sigma}\right)$$

Later improvements reduced this to $O(n^{2.5} \sqrt{\log n} / \sigma)$ and eventually polynomial bounds with better constants.

### Significance

Smoothed analysis resolves the paradox for several algorithms:

| Algorithm | Worst Case | Smoothed Complexity |
|-----------|-----------|---------------------|
| Simplex method | Exponential | Polynomial |
| $k$-means (Lloyd's) | Exponential iterations | Polynomial |
| Perceptron | Exponential | Polynomial |
| ICP (iterative closest point) | Slow convergence | Polynomial |

The key insight is that pathological inputs occupy a measure-zero set in input space. Any slight perturbation (modeling noise in real data, floating-point imprecision, or measurement error) pushes the input away from these pathological configurations.

---

## 8. Competitive Analysis for Online Algorithms

### The Problem

Analyze algorithms that must make irrevocable decisions without knowledge of future input. Competitive analysis compares an online algorithm's cost to the optimal offline algorithm's cost.

### The Formula

An online algorithm $\text{ALG}$ is **$c$-competitive** if there exists a constant $b$ such that for all input sequences $\sigma$:

$$\text{ALG}(\sigma) \leq c \cdot \text{OPT}(\sigma) + b$$

where $\text{OPT}(\sigma)$ is the cost of the optimal offline algorithm that knows the entire input in advance. The **competitive ratio** is the infimum over all $c$ for which $\text{ALG}$ is $c$-competitive.

### Worked Example: Ski Rental Problem

You can rent skis for \$1/day or buy them for \$B. You do not know how many days you will ski.

**Break-even algorithm:** Rent for $B-1$ days, then buy on day $B$.

- If you ski $d < B$ days: pay $d$ (rent only). OPT pays $d$. Ratio = 1.
- If you ski $d \geq B$ days: pay $(B-1) + B = 2B - 1$ (rent then buy). OPT pays $B$ (buy immediately). Ratio $= (2B-1)/B < 2$.

The break-even algorithm is $2$-competitive. No deterministic algorithm can achieve a ratio better than $2$.

**Randomized:** A randomized algorithm can achieve competitive ratio $e/(e-1) \approx 1.58$ against an oblivious adversary.

### Paging (Online Caching)

Given a cache of size $k$ and a sequence of page requests:

| Algorithm | Competitive Ratio | Type |
|-----------|------------------|------|
| LRU (Least Recently Used) | $k$ | Deterministic |
| FIFO (First In First Out) | $k$ | Deterministic |
| FWF (Flush When Full) | $k$ | Deterministic |
| Marking algorithm | $2H_k \approx 2\ln k$ | Randomized |
| Lower bound (deterministic) | $k$ | Proved optimal |
| Lower bound (randomized) | $H_k \approx \ln k$ | Proved optimal |

**Theorem (Sleator-Tarjan 1985).** No deterministic online paging algorithm has a competitive ratio better than $k$.

**Proof sketch.** The adversary maintains a set of $k+1$ pages. When the algorithm has pages $S$ in cache (with $|S| = k$), the adversary requests the unique page $p \notin S$. This forces a page fault on every request. Meanwhile, the optimal offline algorithm (using Belady's furthest-in-the-future rule) faults at most once every $k$ requests, since after each fault it can serve the next $k-1$ requests from cache. The ratio approaches $k$.

### The Potential Function Approach

Competitive analysis often uses potential functions analogous to amortized analysis. Define $\Phi$ as a function of the online and offline states:

$$\text{ALG}_i + \Delta\Phi_i \leq c \cdot \text{OPT}_i$$

where $\text{ALG}_i$ and $\text{OPT}_i$ are the costs incurred at step $i$. Summing over all steps and using $\Phi \geq 0$ at the end yields the competitive ratio.

For LRU paging, a standard potential function is:

$$\Phi = k \cdot |\{p \in \text{LRU cache but not in OPT cache}\}|$$

which yields the tight competitive ratio of $k$.

---

## Prerequisites

- Discrete mathematics: summations, logarithms, induction
- Probability theory: expectation, linearity of expectation, indicator random variables
- Basic data structures: arrays, trees, heaps, hash tables
- Familiarity with common algorithms: sorting, graph search, divide-and-conquer
- Linear algebra basics (for Strassen analysis)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Apply Big-O, Big-Omega, Big-Theta to simple functions. Use the master theorem on standard recurrences. Explain why merge sort is O(n log n). Distinguish memoization from tabulation. |
| **Intermediate** | Perform amortized analysis using all three methods. Apply Akra-Bazzi to non-standard recurrences. Prove greedy correctness via exchange argument. Derive the comparison sorting lower bound. Analyze randomized algorithms using indicator random variables. |
| **Advanced** | Prove the master theorem from the recursion tree. Identify matroid structure in optimization problems. Apply smoothed analysis to explain practical performance. Design and analyze competitive online algorithms. Construct adversary arguments for novel lower bounds. |
