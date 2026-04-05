# The Mathematics of Consecutive Sequences -- Set Theory and Amortized Analysis

> *A hash set transforms an unsorted array into a membership oracle, and the simple invariant "only start from sequence beginnings" converts a naive quadratic scan into a linear-time sweep where each element pays for itself at most twice.*

---

## 1. Hash Set as a Membership Oracle (Set Theory)

### The Problem

Why does inserting all elements into a hash set enable O(n) sequence detection, and what
algebraic properties of sets make the algorithm correct?

### The Formula

Let $S = \{x \mid x \in \text{nums}\}$ be the set of distinct elements. A hash set provides
an approximate characteristic function:

$$\chi_S(x) = \begin{cases} 1 & \text{if } x \in S \\ 0 & \text{if } x \notin S \end{cases}$$

evaluated in expected $O(1)$ time per query. The set operation eliminates duplicates:

$$|S| \le |\text{nums}| = n$$

A **consecutive sequence** starting at $a$ with length $k$ is the set $\{a, a+1, a+2, \ldots, a+k-1\} \subseteq S$. A starting point $a$ satisfies $a \in S \land (a - 1) \notin S$. The length is:

$$\ell(a) = \max\{k \in \mathbb{N} \mid \{a, a+1, \ldots, a+k-1\} \subseteq S\}$$

The answer is:

$$L^* = \max_{a \in S,\; a-1 \notin S} \ell(a)$$

### Worked Examples

$\text{nums} = [100, 4, 200, 1, 3, 2]$, so $S = \{1, 2, 3, 4, 100, 200\}$.

Sequence starts (elements where predecessor is absent):
- $1$: $0 \notin S$, so $1$ is a start. $\ell(1) = |\{1,2,3,4\} \cap S| = 4$.
- $100$: $99 \notin S$, so $100$ is a start. $\ell(100) = |\{100\} \cap S| = 1$.
- $200$: $199 \notin S$, so $200$ is a start. $\ell(200) = |\{200\} \cap S| = 1$.

Non-starts: $2$ ($1 \in S$), $3$ ($2 \in S$), $4$ ($3 \in S$).

$$L^* = \max(4, 1, 1) = 4$$

---

## 2. Amortized Analysis -- Why the Algorithm is O(n) (Complexity Theory)

### The Problem

The algorithm has a nested loop structure (outer loop over elements, inner while-loop
extending sequences). Prove the total work is $O(n)$ despite the nesting.

### The Formula

Define the following sets over $S$:

$$\text{Starts} = \{a \in S \mid a - 1 \notin S\}$$

These are the left endpoints of maximal consecutive runs. Each maximal run $R_i$ has length $\ell_i$, and the runs partition $S$:

$$S = R_1 \cup R_2 \cup \cdots \cup R_m, \quad R_i \cap R_j = \emptyset \text{ for } i \ne j$$

**Outer loop cost:** The outer loop iterates over all $|S|$ elements. For each element, it performs one hash lookup ($\chi_S(\text{num} - 1)$). Cost: $O(|S|)$.

**Inner loop cost:** The inner while-loop only executes when the current element is a sequence start. For start $a_i$ with run length $\ell_i$, the inner loop performs $\ell_i - 1$ iterations (each a hash lookup). Total inner loop work across all starts:

$$\sum_{i=1}^{m} (\ell_i - 1) = \sum_{i=1}^{m} \ell_i - m = |S| - m \le |S|$$

**Total work:**

$$T(n) = \underbrace{|S|}_{\text{outer loop}} + \underbrace{|S|}_{\text{inner loops}} + \underbrace{|S|}_{\text{set construction}} = O(|S|) = O(n)$$

Each element of $S$ is touched at most twice: once by the outer loop's predecessor check, and at most once by some inner loop extending through it. This is a **partition argument** -- the inner loop iterations across all starts are bounded by $|S|$ because the runs partition $S$.

### Worked Examples

$S = \{0, 1, 2, 3, 4, 5, 6, 7, 8\}$ (from input $[0,3,7,2,5,8,4,6,0,1]$):

- Starts: $\{0\}$ (only one maximal run).
- Inner loop iterations: $8$ (extending from $0$ to $8$).
- Outer loop iterations: $9$ (checking each element).
- Non-start skips: $8$ elements check predecessor, find it present, skip.
- Total hash lookups: $9$ (predecessor checks) $+ 9$ (extension checks including the failing one) $= 18 = 2|S|$.

This confirms $O(n)$ even when the entire array forms a single consecutive sequence.

---

## 3. Hash Table Expected Complexity and Worst-Case Bounds (Probability Theory)

### The Problem

The O(n) claim assumes O(1) hash lookups. Under what conditions does this hold, and what
is the worst-case behavior?

### The Formula

A hash table with $n$ elements and $m$ buckets has load factor $\alpha = n/m$. Under
**simple uniform hashing** (each key equally likely to hash to any bucket):

- Expected search time (successful): $\Theta(1 + \alpha/2)$
- Expected search time (unsuccessful): $\Theta(1 + \alpha)$

With dynamic resizing ($m = \Theta(n)$, so $\alpha = O(1)$), expected lookup is $O(1)$.

**Worst case** without randomization: all $n$ keys collide in one bucket, giving $O(n)$ per lookup and $O(n^2)$ total. This is mitigated by:

1. **Universal hashing:** Choose $h$ randomly from a family $\mathcal{H}$ such that $\Pr_{h \in \mathcal{H}}[h(x) = h(y)] \le 1/m$ for $x \ne y$. Expected chain length is $O(1 + \alpha)$.

2. **Perfect hashing (FKS):** Two-level scheme with $O(n)$ space and $O(1)$ worst-case lookup, but $O(n)$ expected construction time.

3. **Cuckoo hashing:** Two hash functions, $O(1)$ worst-case lookup, amortized $O(1)$ insertion.

For integer keys (as in this problem), standard hash functions perform well because
integer hash functions distribute uniformly. Adversarial inputs are not a practical concern.

### Worked Examples

Python `set` uses open addressing with perturbation-based probing. For the input
$[100, 4, 200, 1, 3, 2]$, the hash values of small integers in CPython are the integers
themselves. With table size $m = 8$ (next power of 2 above $6$):

$$h(1) = 1, \; h(2) = 2, \; h(3) = 3, \; h(4) = 4, \; h(100) = 100 \bmod 8 = 4$$

Here $4$ and $100$ collide at slot 4. The probing sequence resolves this in one extra step.
Total probes for 6 insertions: approximately 7 (one collision). Load factor $\alpha = 6/8 = 0.75$.

For Go's `map[int]bool`, the runtime uses a hash function seeded per map instance
(randomized at creation). With $n = 6$ elements, the initial bucket count is 8.
Each bucket holds up to 8 key-value pairs before overflow chaining. Expected collisions
for 6 elements in 8 buckets: $6 \times 5 / (2 \times 8) \approx 1.9$, consistent with
near-constant lookup.

---

## 4. Comparison-Based Lower Bound and Why Hashing is Necessary (Information Theory)

### The Problem

Can any algorithm solve Longest Consecutive Sequence in $O(n)$ time without hashing?
What is the fundamental lower bound for comparison-based approaches?

### The Formula

A comparison-based algorithm can only distinguish elements through pairwise comparisons
($<$, $=$, $>$). The decision tree for determining the longest consecutive run among $n$
elements must have at least $n!$ leaves (one per possible ordering), giving a lower bound:

$$\text{depth} \ge \log_2(n!) = \Omega(n \log n)$$

This means any comparison-based solution (including sorting) requires $\Omega(n \log n)$ time.
The hash set approach circumvents this bound by using the **integer structure** of the keys --
specifically, the ability to compute $x - 1$ and $x + 1$ and look them up in $O(1)$ expected
time. This is not a comparison operation; it is a **random access** operation on a hash table
indexed by value.

The distinction is fundamental: comparison-based algorithms treat elements as opaque objects
with only an ordering relation, while hash-based algorithms exploit the algebraic structure
of integers (successor, predecessor, equality via hashing).

### Worked Examples

**Sorting approach (comparison-based):** Sort the array, then scan for the longest run of
consecutive values. Sorting is $\Omega(n \log n)$; the scan is $O(n)$. Total: $O(n \log n)$.

For $n = 10^5$, the difference matters: $n \log_2 n \approx 1.7 \times 10^6$ comparisons
for sorting versus $2n = 2 \times 10^5$ hash lookups for the set approach -- roughly an
$8.5\times$ difference in operations.

**Radix sort (non-comparison):** If all values are bounded by $|v| \le V$, radix sort runs
in $O(n \cdot \log V / \log n)$. For $V = 10^9$ and $n = 10^5$, this is
$O(n \cdot 30/17) \approx O(1.76n)$, which is linear but with a larger constant than the
hash set approach. It also requires $O(n)$ auxiliary space, same as the hash set.

---

## Prerequisites

- Hash table fundamentals (chaining, open addressing, load factor)
- Amortized analysis (aggregate method, partition arguments)
- Set theory basics (membership, partitioning, characteristic functions)
- Big-O notation and asymptotic analysis
- Decision tree model and comparison-based lower bounds

- Hash table fundamentals (chaining, open addressing, load factor)
- Amortized analysis (aggregate method, partition arguments)
- Set theory basics (membership, partitioning, characteristic functions)
- Big-O notation and asymptotic analysis

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Implement the hash set solution. Verify O(n) behavior by counting hash lookups on small inputs. Understand why the "sequence start" check prevents redundant work. |
| **Intermediate** | Prove the O(n) bound using the partition argument. Analyze the expected number of hash collisions for uniform integer inputs. Compare with the sorting-based O(n log n) approach and identify when each is preferable. |
| **Advanced** | Analyze worst-case behavior under adversarial hash inputs. Study universal hashing families and their guarantees. Extend to the streaming variant (elements arrive one at a time) using Union-Find with path compression and union by rank, achieving O(n * alpha(n)) amortized time. Prove the lower bound: any comparison-based algorithm requires Omega(n log n), making hashing essential for O(n). |
