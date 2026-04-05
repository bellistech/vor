# The Mathematics of Anagram Grouping -- Equivalence Classes and Hash Functions

> *Two words built from the same letters are not merely similar -- they are equivalent under permutation. Grouping anagrams is the act of discovering these equivalence classes, and the key insight is that every class needs a canonical representative.*

---

## 1. Permutations and Equivalence Relations (Algebra)

### The Problem

Let $\Sigma = \{a, b, \ldots, z\}$ be a finite alphabet of size 26. A string $w$ of length $k$ is an element of $\Sigma^k$. Two strings $u, v \in \Sigma^*$ are **anagrams** if and only if one can be obtained from the other by a permutation of character positions:

$$u \sim v \iff \exists \, \sigma \in S_k \text{ such that } v_i = u_{\sigma(i)} \; \forall \, i \in \{1, \ldots, k\}$$

where $S_k$ is the symmetric group on $k$ elements. This relation $\sim$ is reflexive, symmetric, and transitive -- it is an **equivalence relation**. The anagram groups we seek are precisely the equivalence classes $[w]_\sim$ under this relation.

### The Formula

The number of distinct anagrams of a string $w$ with character frequencies $f_1, f_2, \ldots, f_{26}$ (where $\sum f_i = k$) is given by the multinomial coefficient:

$$|[w]_\sim| = \frac{k!}{f_1! \cdot f_2! \cdots f_{26}!}$$

For the string "eat" with $k = 3$ and all frequencies equal to 1:

$$|[\text{eat}]_\sim| = \frac{3!}{1! \cdot 1! \cdot 1!} = 6$$

The six permutations are: eat, eta, aet, ate, tea, tae. Only those present in the input array appear in the output group.

### Worked Examples

**Example 1:** Given $\{\text{eat}, \text{tea}, \text{tan}, \text{ate}, \text{nat}, \text{bat}\}$, the equivalence classes are:

- $[\text{eat}]_\sim = \{\text{eat}, \text{tea}, \text{ate}\}$ (canonical form: "aet")
- $[\text{tan}]_\sim = \{\text{tan}, \text{nat}\}$ (canonical form: "ant")
- $[\text{bat}]_\sim = \{\text{bat}\}$ (canonical form: "abt")

**Example 2:** The empty string $\varepsilon$ has exactly one equivalence class $\{\ \varepsilon\ \}$ since $S_0$ is trivial.

---

## 2. Canonical Representatives and Hash Functions (Computation)

### The Problem

To partition $n$ strings into equivalence classes efficiently, we need a **canonical form** function $\phi : \Sigma^* \to K$ such that:

$$u \sim v \iff \phi(u) = \phi(v)$$

Any such function lets us use a hash map: insert each string into the bucket keyed by $\phi(w)$.

### The Formula

**Approach A -- Sorted canonical form.** Define $\phi_{\text{sort}}(w)$ as the string obtained by sorting the characters of $w$ in lexicographic order. Sorting takes $O(k \log k)$ per string using comparison sort.

**Approach B -- Frequency vector.** Define $\phi_{\text{freq}}(w) = (f_a, f_b, \ldots, f_z) \in \mathbb{N}^{26}$ where $f_c = |\{i : w_i = c\}|$. Computing the vector takes $O(k)$ per string. The frequency vector is a **complete invariant** of the equivalence class:

$$\phi_{\text{freq}}(u) = \phi_{\text{freq}}(v) \iff u \sim v$$

This holds because two strings are permutations of each other if and only if they share the same multiset of characters, and the frequency vector is the canonical encoding of that multiset.

### Worked Examples

For $w = \text{"eat"}$:

$$\phi_{\text{sort}}(\text{"eat"}) = \text{"aet"}$$

$$\phi_{\text{freq}}(\text{"eat"}) = (1, 0, 0, 0, 1, 0, \ldots, 0, 1, 0, \ldots, 0)$$

where position 0 (a) = 1, position 4 (e) = 1, position 19 (t) = 1, all others = 0.

For $w = \text{"tea"}$:

$$\phi_{\text{sort}}(\text{"tea"}) = \text{"aet"} = \phi_{\text{sort}}(\text{"eat"}) \quad \checkmark$$

$$\phi_{\text{freq}}(\text{"tea"}) = (1, 0, 0, 0, 1, 0, \ldots, 0, 1, 0, \ldots, 0) = \phi_{\text{freq}}(\text{"eat"}) \quad \checkmark$$

---

## 3. Hash Map Mechanics and Collision Analysis (Data Structures)

### The Problem

The algorithm inserts $n$ strings into a hash map $H : K \to \text{List}[\Sigma^*]$. The efficiency depends on the hash function over the key space $K$ and the expected number of collisions.

### The Formula

For the sorting approach, the key space is $K = \Sigma^*$ (sorted strings). The expected number of buckets (distinct keys) equals the number of equivalence classes $m$. If strings are distributed uniformly at random across $m$ classes, the expected bucket size is $n/m$.

The total work is:

$$T(n, k) = \sum_{i=1}^{n} C_{\phi}(k_i) + O(n)$$

where $C_\phi(k_i)$ is the cost of computing the canonical form for string $i$ of length $k_i$:

- Sorting: $C_{\text{sort}}(k) = O(k \log k)$, giving $T = O(n \cdot k \log k)$ when $k_i \le k$ for all $i$
- Counting: $C_{\text{freq}}(k) = O(k + |\Sigma|) = O(k + 26) = O(k)$, giving $T = O(n \cdot k)$

### Worked Examples

For input size $n = 10^4$ with max string length $k = 100$:

- Sorting: $10^4 \times 100 \times \log_2(100) \approx 10^4 \times 100 \times 6.6 \approx 6.6 \times 10^6$ operations
- Counting: $10^4 \times 100 = 10^6$ operations (plus $10^4 \times 26$ for key construction)

The counting approach yields roughly a $6\times$ constant-factor improvement at the upper constraint bound.

---

## 4. Prime Product Hashing -- An Arithmetic Alternative (Number Theory)

### The Problem

Both the sorting and counting approaches require either string comparison or vector comparison for hash-key equality. A third strategy assigns each letter a distinct prime number and uses the product as the key. Since prime factorization is unique (the Fundamental Theorem of Arithmetic), two strings produce the same product if and only if they are anagrams.

### The Formula

Assign primes $p_a = 2, p_b = 3, p_c = 5, p_d = 7, \ldots, p_z = 101$ (the first 26 primes). Define:

$$\phi_{\text{prime}}(w) = \prod_{i=1}^{k} p_{w_i}$$

By the Fundamental Theorem of Arithmetic, for any two strings $u, v$:

$$\phi_{\text{prime}}(u) = \phi_{\text{prime}}(v) \iff u \sim v$$

This is because the multiset of prime factors on each side is determined exactly by the character frequencies, and unique factorization guarantees no two distinct multisets yield the same product.

### Worked Examples

For $w = \text{"eat"}$:

$$\phi_{\text{prime}}(\text{"eat"}) = p_e \cdot p_a \cdot p_t = 11 \times 2 \times 71 = 1562$$

For $w = \text{"tea"}$:

$$\phi_{\text{prime}}(\text{"tea"}) = p_t \cdot p_e \cdot p_a = 71 \times 11 \times 2 = 1562 \quad \checkmark$$

For $w = \text{"tan"}$:

$$\phi_{\text{prime}}(\text{"tan"}) = p_t \cdot p_a \cdot p_n = 71 \times 2 \times 43 = 6106 \neq 1562 \quad \checkmark$$

**Caveat:** For long strings (e.g., $k = 100$), the product can exceed $101^{100} \approx 10^{200}$, which overflows fixed-width integers. This approach is elegant in theory but requires arbitrary-precision arithmetic in practice, making it slower than frequency counting despite its $O(k)$ per-string cost. It is primarily of theoretical interest.

---

## 5. Lower Bounds and Optimality (Complexity Theory)

### The Problem

Is $O(n \cdot k)$ optimal? Any algorithm must read every character of every string at least once to determine membership, so the input size alone gives a lower bound of $\Omega(n \cdot k)$.

### The Formula

The information-theoretic argument: the output is a partition of $n$ elements. The number of possible partitions (Bell number $B_n$) satisfies:

$$\log_2 B_n = \Theta(n \log n)$$

However, the bottleneck is not the partition structure but the character-level comparison. Since each of the $n \cdot k$ characters must be inspected, and the counting approach touches each character exactly once plus $O(n \cdot |\Sigma|)$ overhead for key construction:

$$T_{\text{optimal}} = \Theta(n \cdot k + n \cdot |\Sigma|) = \Theta(n \cdot k)$$

when $k \ge |\Sigma| = 26$. The frequency-vector approach is therefore **asymptotically optimal**.

### Worked Examples

For the constraint bounds $n = 10^4$, $k = 100$, $|\Sigma| = 26$:

- Lower bound: $\Omega(10^4 \times 100) = \Omega(10^6)$
- Counting approach: $O(10^4 \times 100 + 10^4 \times 26) = O(1.26 \times 10^6)$
- The constant overhead from the 26-slot key construction is negligible

The sorting approach at $O(n \cdot k \log k)$ is not optimal but remains practical for interview settings where $k \le 100$ and the logarithmic factor is small.

---

## Prerequisites

| Concept | Why It Matters |
|---------|---------------|
| Equivalence relations | Anagram grouping is partitioning by an equivalence relation on permutations |
| Symmetric group $S_k$ | Formalizes "rearrangement" -- two strings are anagrams iff one is a permutation of the other |
| Multinomial coefficients | Counts the size of each equivalence class (number of possible anagrams) |
| Hash maps | The data structure that maps canonical keys to groups in amortized $O(1)$ per lookup |
| Comparison sorting | The $O(k \log k)$ cost of the sorting-based canonical form |
| Counting sort / frequency vectors | The $O(k)$ alternative using character histograms as keys |

## Complexity

| Metric | Sorting Approach | Counting Approach |
|--------|-----------------|-------------------|
| Time | $O(n \cdot k \log k)$ | $O(n \cdot k)$ |
| Space | $O(n \cdot k)$ | $O(n \cdot k)$ |
| Key computation | $O(k \log k)$ per string | $O(k)$ per string |
| Hash map lookups | $O(n)$ amortized | $O(n)$ amortized |

Where $n$ is the number of strings and $k$ is the maximum string length. Space is $O(n \cdot k)$ in both cases because we store all input strings in the map regardless of key strategy.
