# The Mathematics of Kolmogorov Complexity -- Algorithmic Information, Randomness, and the Limits of Description

> *Kolmogorov complexity captures the intrinsic information content of individual objects, independent of any probability distribution -- revealing deep connections between computation, randomness, and the foundations of inductive inference.*

---

## 1. Formal Definition and the Invariance Theorem

### The Problem

Define an objective, distribution-free measure of the information content of a finite binary string, and show that this measure is robust -- essentially independent of the choice of computational model.

### The Formula

Fix a universal Turing machine $U$. The **plain Kolmogorov complexity** of a string $x \in \{0,1\}^*$ is:

$$K_U(x) = \min \{ |p| : U(p) = x \}$$

where $|p|$ denotes the length of the program $p$ in bits, and $U(p) = x$ means $U$ on input $p$ halts and outputs $x$.

**Invariance Theorem** (Kolmogorov 1965, Solomonoff 1964): For any two universal Turing machines $U_1$ and $U_2$, there exists a constant $c_{U_1,U_2}$ such that for all $x$:

$$|K_{U_1}(x) - K_{U_2}(x)| \leq c_{U_1,U_2}$$

### Proof

Since $U_1$ is universal, there exists a program $s_{21}$ such that for all inputs $p$, $U_1(s_{21} \cdot p) = U_2(p)$. Here $s_{21}$ is a fixed interpreter for $U_2$ written for $U_1$, and $s_{21} \cdot p$ denotes concatenation.

If $p^*$ is the shortest program for $x$ on $U_2$, then $s_{21} \cdot p^*$ is a valid program for $x$ on $U_1$, giving:

$$K_{U_1}(x) \leq |s_{21}| + |p^*| = K_{U_2}(x) + c$$

where $c = |s_{21}|$ depends only on $U_1$ and $U_2$. By symmetry (swapping roles), we get the bound in both directions.

### Significance

The constant $c$ is an artifact of the encoding, not of the object $x$. For asymptotic analysis (strings of growing length), the choice of UTM is irrelevant. We write $K(x)$ without subscript, understanding it is defined up to $O(1)$.

---

## 2. Incomputability of K(x)

### The Problem

Show that no algorithm can compute the Kolmogorov complexity function $K : \{0,1\}^* \to \mathbb{N}$.

### The Proof (Berry Paradox Formalization)

**Theorem**: $K$ is not a computable function.

**Proof** by contradiction. Suppose there exists a total computable function $f$ such that $f(x) = K(x)$ for all $x$.

Define the following program $P_n$:

```
enumerate all strings x in length-lexicographic order
for each x:
    if f(x) > n:
        output x and halt
```

This program outputs the first string $x$ with $K(x) > n$. The program $P_n$ has length:

$$|P_n| = O(\log n)$$

since $n$ is the only parameter and can be encoded in $\lceil \log_2 n \rceil + O(1)$ bits. But $P_n$ produces a string $x$ with $K(x) > n$, meaning:

$$K(x) \leq |P_n| = O(\log n)$$

For sufficiently large $n$, $O(\log n) < n$, giving $K(x) \leq O(\log n) < n < K(x)$. Contradiction. $\blacksquare$

### Upper Semicomputability

Although $K(x)$ is not computable, it is **upper semicomputable** (also called co-recursively enumerable): there exists a computable function $\phi(x, t)$ such that:

- $\phi(x, t) \geq \phi(x, t+1)$ for all $t$ (non-increasing in $t$)
- $\lim_{t \to \infty} \phi(x, t) = K(x)$

This is achieved by dovetailing: run all programs of length $1, 2, 3, \ldots$ in parallel, and whenever a program $p$ halts with output $x$, update the current best bound for $K(x)$ to $\min(\text{current}, |p|)$.

$K(x)$ is **not** lower semicomputable, which is precisely what prevents computability.

---

## 3. Incompressible Strings and the Counting Argument

### The Problem

Show that most strings are incompressible (have high Kolmogorov complexity), and characterize the properties of such strings.

### The Counting Argument

**Theorem**: For every $n$ and $c \geq 0$, the number of strings $x \in \{0,1\}^n$ with $K(x) < n - c$ is strictly less than $2^{n-c}$.

**Proof**: The number of binary programs of length strictly less than $n - c$ is:

$$\sum_{i=0}^{n-c-1} 2^i = 2^{n-c} - 1$$

Since the function $p \mapsto U(p)$ (restricted to halting programs) is a partial function, at most $2^{n-c} - 1$ strings of length $n$ can have a description of length $< n - c$. Therefore:

$$|\{x \in \{0,1\}^n : K(x) \geq n - c\}| \geq 2^n - 2^{n-c} + 1$$

The fraction of $c$-incompressible strings is at least $1 - 2^{-c}$. Setting $c = 0$: at least one string of every length is incompressible. Setting $c = 10$: more than $99.9\%$ of strings are $10$-incompressible. $\blacksquare$

### Properties of Incompressible Strings

A string $x$ of length $n$ with $K(x) \geq n - O(1)$ behaves like a "typical" outcome of $n$ fair coin flips. Specifically, if $K(x) \geq n$:

1. The number of $1$s in $x$ is $n/2 \pm O(\sqrt{n \log n})$.
2. The longest run of consecutive $0$s has length $\log_2 n \pm O(\log \log n)$.
3. Every substring of length $k$ appears with frequency approximately $2^{-k}$, for $k = O(\log n)$.
4. $x$ passes every polynomial-time computable statistical test.

These are consequences of the fact that any detectable deviation from randomness would yield a shorter description.

---

## 4. Chain Rule

### The Problem

Establish the analog of Shannon's chain rule $H(X,Y) = H(X) + H(Y|X)$ for Kolmogorov complexity.

### The Conditional Complexity

The **conditional Kolmogorov complexity** of $x$ given $y$ is:

$$K(x|y) = \min \{ |p| : U(p, y) = x \}$$

The program $p$ has access to $y$ as auxiliary input.

### The Chain Rule

**Theorem** (Kolmogorov, Zvonkin-Levin): For all strings $x, y$:

$$K(x, y) = K(x) + K(y | x, K(x)) + O(\log K(x,y))$$

The logarithmic term arises because the plain (non-prefix-free) encoding requires specifying the boundary between the description of $x$ and the description of $y|x$.

### Proof Sketch

**Upper bound**: Given a shortest program $p_x$ for $x$ (length $K(x)$) and a shortest program $p_{y|x}$ for $y$ given $x$ (length $K(y|x,K(x))$), we can reconstruct $(x,y)$ from the concatenation $p_x \cdot p_{y|x}$ together with $|p_x|$ (which takes $O(\log K(x))$ bits to encode since we need a self-delimiting encoding of the boundary). So:

$$K(x,y) \leq K(x) + K(y|x,K(x)) + O(\log K(x))$$

**Lower bound**: Given a shortest program for $(x,y)$, we can extract $x$ (yielding $K(x) \leq K(x,y) + O(1)$), and given $x$ and $K(x)$, we can extract $y$, yielding $K(y|x,K(x)) \leq K(x,y) - K(x) + O(\log K(x,y))$.

For **prefix-free complexity** $K_{\text{prefix}}$, the chain rule holds exactly up to $O(1)$:

$$K_{\text{prefix}}(x,y) = K_{\text{prefix}}(x) + K_{\text{prefix}}(y | x) + O(1)$$

This is one of the primary motivations for preferring prefix-free complexity in theoretical work.

---

## 5. Kolmogorov Complexity and Shannon Entropy

### The Problem

Relate the individual-object measure $K(x)$ to the distributional measure $H(X)$ from Shannon's information theory.

### The Relationship

Let $X$ be a random variable over $\{0,1\}^n$ with distribution $P$. Then:

$$\mathbb{E}[K(X)] = \sum_x P(x) \cdot K(x) \leq n \cdot H(X) + O(\log n)$$

More precisely, for an i.i.d. source with per-symbol entropy $H$:

$$\mathbb{E}[K(X_1 X_2 \cdots X_n)] = n H + O(\log n)$$

The $O(\log n)$ term accounts for encoding the length $n$.

**Individual strings**: For a specific outcome $x$ of $X_1 \cdots X_n$:

$$K(x) \approx -\log_2 P(x) + O(\log n)$$

with probability $1$ as $n \to \infty$ (by the individual randomness theorem).

### Key Distinctions

| Property | Shannon $H(X)$ | Kolmogorov $K(x)$ |
|---|---|---|
| Defined on | random variables | individual strings |
| Computable | yes (given distribution) | no |
| Requires | probability distribution | universal TM |
| Measures | expected surprise | descriptive complexity |

Shannon entropy is the expected value of Kolmogorov complexity (up to logarithmic terms) when the string is drawn from a known distribution.

---

## 6. Prefix-Free Complexity and Chaitin's Omega

### Prefix-Free Complexity

A **prefix-free Turing machine** $V$ is one whose domain is prefix-free: if $V(p)$ halts, then $V(q)$ does not halt for any proper extension $q$ of $p$. The prefix-free Kolmogorov complexity is:

$$K_{\text{prefix}}(x) = \min \{ |p| : V(p) = x \}$$

where $V$ is a universal prefix-free TM.

$K_{\text{prefix}}(x) \geq K(x)$ always, and $K_{\text{prefix}}(x) \leq K(x) + 2 \log_2 K(x) + O(1)$ (the overhead of self-delimiting the program length).

The key advantage: the set of valid programs forms a prefix-free set, so by the Kraft inequality:

$$\sum_{x} 2^{-K_{\text{prefix}}(x)} \leq 1$$

This sum defines the **universal semimeasure** $\mathbf{m}(x) = \sum_{p : V(p) = x} 2^{-|p|}$, which dominates every computable semimeasure up to a multiplicative constant.

### Chaitin's Omega

**Definition**: The halting probability of a universal prefix-free TM $V$:

$$\Omega_V = \sum_{p : V(p) \text{ halts}} 2^{-|p|}$$

By the Kraft inequality, $0 < \Omega < 1$, so $\Omega$ is a well-defined real number.

**Properties**:

1. $\Omega$ is **computably enumerable from below** (left-c.e.): we can compute a non-decreasing sequence of rationals converging to $\Omega$ by running programs and adding $2^{-|p|}$ whenever one halts.

2. $\Omega$ is **not computable**: if we knew $\Omega$ to $n$ bits of precision, we could decide the halting problem for all programs of length $\leq n$, because we would know the total contribution from all such programs and could determine which ones halt.

3. $\Omega$ is **Martin-Lof random**: $K_{\text{prefix}}(\Omega_1 \cdots \Omega_n) \geq n - O(1)$ for all $n$, where $\Omega_1 \Omega_2 \cdots$ is the binary expansion. This is the strongest possible form of randomness -- a single real number that concentrates the difficulty of the halting problem.

4. $\Omega$ is **Solovay complete** among left-c.e. reals: every left-c.e. real can be computed from $\Omega$ (they are equivalent in a strong reducibility sense).

---

## 7. Martin-Lof Randomness

### The Problem

Define what it means for an individual infinite binary sequence to be "random," without reference to a probability model.

### Constructive Null Sets

A **Martin-Lof test** is a uniformly recursively enumerable sequence of open sets $\{U_m\}_{m=1}^{\infty}$ with:

$$\mu(U_m) \leq 2^{-m}$$

where $\mu$ is the fair-coin (Lebesgue) measure on $\{0,1\}^{\omega}$.

An infinite sequence $\omega \in \{0,1\}^{\omega}$ **fails** the test if $\omega \in \bigcap_{m=1}^{\infty} U_m$.

**Definition**: $\omega$ is **Martin-Lof random** if it passes every Martin-Lof test, i.e., $\omega \notin \bigcap_m U_m$ for every Martin-Lof test $\{U_m\}$.

### The Universal Test

**Theorem** (Martin-Lof 1966): There exists a **universal Martin-Lof test** $\{U_m^*\}$ such that $\omega$ is ML-random if and only if $\omega \notin \bigcap_m U_m^*$.

**Proof**: Enumerate all ML-tests as $\{U_m^{(1)}\}, \{U_m^{(2)}\}, \ldots$ (this is possible because the tests are uniformly r.e.). Define:

$$U_m^* = \bigcup_{i=1}^{\infty} U_{m+i}^{(i)}$$

Then $\mu(U_m^*) \leq \sum_{i=1}^{\infty} 2^{-(m+i)} = 2^{-m}$, so $\{U_m^*\}$ is a valid test, and any sequence failing some test $\{U_m^{(i)}\}$ also fails $\{U_m^*\}$.

### Schnorr-Levin Theorem

**Theorem**: $\omega$ is Martin-Lof random if and only if:

$$K_{\text{prefix}}(\omega_1 \omega_2 \cdots \omega_n) \geq n - O(1) \quad \text{for all } n$$

This is the fundamental bridge between Martin-Lof's measure-theoretic definition and Kolmogorov's complexity-theoretic definition.

### Consequences

ML-random sequences satisfy all effective versions of classical probability laws:

- **Strong law of large numbers**: $\lim_{n \to \infty} \frac{1}{n}\sum_{i=1}^n \omega_i = \frac{1}{2}$
- **Law of the iterated logarithm**: $\limsup_{n \to \infty} \frac{\sum_{i=1}^n (2\omega_i - 1)}{\sqrt{2n \ln \ln n}} = 1$
- **Normality**: $\omega$ is normal in base 2 (and in fact in all bases, by relativization)

---

## 8. Normalized Compression Distance (NCD)

### The Problem

Use Kolmogorov complexity to define a universal similarity metric between objects, then approximate it with real-world compressors.

### Information Distance

The **information distance** between strings $x$ and $y$ is:

$$E(x, y) = \max(K(x|y), K(y|x))$$

This is the length of the shortest program that computes $x$ from $y$ and $y$ from $x$. It is a metric (up to additive $O(\log)$ terms) and is **universal**: it minorizes every computable distance that is also an upper semicomputable distance.

### Normalized Information Distance (NID)

$$\text{NID}(x, y) = \frac{\max(K(x|y), K(y|x))}{\max(K(x), K(y))}$$

This normalizes to $[0, 1]$. NID is universal among normalized computable distances. It satisfies the metric axioms (up to negligible terms) and:

$$\text{NID}(x, y) = \frac{K(x,y) - \min(K(x), K(y))}{\max(K(x), K(y))} + O\left(\frac{\log n}{n}\right)$$

### Practical Approximation: NCD

Since $K$ is incomputable, replace it with a real compressor $C$:

$$\text{NCD}(x, y) = \frac{C(xy) - \min(C(x), C(y))}{\max(C(x), C(y))}$$

where $C(x)$ is the compressed size of $x$ and $C(xy)$ is the compressed size of the concatenation.

**Applications**: language classification, phylogenetic trees, music similarity, plagiarism detection, malware classification, clustering of genomic sequences.

---

## 9. Applications to Lower Bounds in Algorithms

### The Incompressibility Method

The incompressibility method proves lower bounds by:

1. Assume an algorithm uses fewer resources than claimed.
2. Show this would let us compress an incompressible string.
3. Contradiction.

### Example: Sorting Lower Bound

**Theorem**: Any comparison-based sorting algorithm requires $\Omega(n \log n)$ comparisons in the worst case.

**Proof**: Let $x$ be an incompressible string of length $n \log n$ bits, encoding a permutation $\pi$ of $\{1, \ldots, n\}$. There are $n!$ permutations, so $K(\pi) \geq \log_2(n!) = n \log n - O(n)$.

Suppose an algorithm sorts using $f(n)$ comparisons. Each comparison has a binary outcome, so the sequence of outcomes is a binary string $b$ of length $f(n)$. Given $b$ and the algorithm, we can reconstruct $\pi$. Therefore:

$$K(\pi) \leq f(n) + O(1)$$

Combined: $n \log n - O(n) \leq f(n) + O(1)$, so $f(n) = \Omega(n \log n)$. $\blacksquare$

### Example: Average-Case Complexity

The method also proves average-case lower bounds: since most strings are incompressible (by the counting argument), the lower bound holds for **most** inputs, not just worst-case constructions. This is stronger than adversarial arguments.

### Other Applications

- **Space lower bounds** for streaming algorithms
- **Communication complexity** lower bounds via Kolmogorov mutual information
- **Circuit complexity**: showing random functions require large circuits
- **Data structure lower bounds**: cell-probe model arguments

---

## 10. Solomonoff Induction and Algorithmic Probability

### The Problem

Use Kolmogorov complexity to define a universal prior for Bayesian prediction that makes no assumptions about the data-generating process.

### The Universal Semimeasure

The **algorithmic probability** (Solomonoff 1964) of a string $x$ is:

$$\mathbf{m}(x) = \sum_{p : V(p) = x} 2^{-|p|}$$

where $V$ is a universal prefix-free TM. This is the probability that a uniformly random infinite binary string, fed as a program to $V$, produces output beginning with $x$.

**Universality** (Solomonoff-Levin): For every computable probability measure $\mu$, there exists a constant $c_\mu$ such that:

$$\mathbf{m}(x) \geq c_\mu \cdot \mu(x) \quad \text{for all } x$$

### Connection to Complexity

$$-\log_2 \mathbf{m}(x) = K_{\text{prefix}}(x) + O(1)$$

for prefix-free complexity (the **coding theorem**). High algorithmic probability corresponds to low complexity.

### Prediction

Given observed data $x_1 \cdots x_n$, Solomonoff's predictor estimates:

$$P(x_{n+1} = 1 \mid x_1 \cdots x_n) = \frac{\mathbf{m}(x_1 \cdots x_n 1)}{\mathbf{m}(x_1 \cdots x_n)}$$

**Convergence theorem**: If the true data source is any computable measure $\mu$, then the total expected KL divergence between Solomonoff's predictions and $\mu$ is bounded:

$$\sum_{n=1}^{\infty} \mathbb{E}_\mu \left[ D_{KL}(\mu(\cdot | x_1 \cdots x_n) \| \mathbf{m}(\cdot | x_1 \cdots x_n)) \right] \leq K_{\text{prefix}}(\mu) \cdot \ln 2$$

The predictions converge rapidly to the truth, with total error bounded by the complexity of the true hypothesis.

---

## References

- Li, M. & Vitanyi, P. *An Introduction to Kolmogorov Complexity and Its Applications*, 4th ed., Springer, 2019.
- Kolmogorov, A.N. "Three approaches to the quantitative definition of information," *Problems of Information Transmission* 1(1):1-7, 1965.
- Solomonoff, R. "A formal theory of inductive inference," *Information and Control* 7(1-2):1-22, 224-254, 1964.
- Chaitin, G. "On the length of programs for computing finite binary sequences," *Journal of the ACM* 13(4):547-569, 1966.
- Martin-Lof, P. "The definition of random sequences," *Information and Control* 9(6):602-619, 1966.
- Downey, R. & Hirschfeldt, D. *Algorithmic Randomness and Complexity*, Springer, 2010.
- Grunwald, P. *The Minimum Description Length Principle*, MIT Press, 2007.
- Cilibrasi, R. & Vitanyi, P. "Clustering by compression," *IEEE Transactions on Information Theory* 51(4):1523-1545, 2005.
