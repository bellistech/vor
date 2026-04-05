# The Mathematics of Information Theory -- Entropy, Capacity, and the Limits of Communication

> *Shannon entropy quantifies the irreducible uncertainty in a random variable, establishing fundamental limits on data compression and reliable communication that no engineering can surpass.*

---

## 1. Entropy: Derivation and Properties

### The Problem

Derive Shannon entropy from first principles and establish its key properties.

### The Formula

We seek a function $H(p_1, p_2, \ldots, p_n)$ measuring the uncertainty of a discrete random variable $X$ with distribution $(p_1, \ldots, p_n)$. Shannon proved that the unique function satisfying three axioms (continuity, monotonicity in $n$ for uniform distributions, and recursion/grouping) is:

$$H(X) = -\sum_{i=1}^{n} p_i \log_2 p_i$$

with the convention $0 \log 0 = 0$ (by continuity).

### Derivation via the Grouping Axiom

Consider $n$ equally likely outcomes. The uncertainty should be $H = \log_2 n$. For non-uniform distributions, apply the grouping axiom: partition outcomes into groups and decompose total uncertainty.

For a binary variable with $P(X=1) = p$:

$$H(p) = -p \log_2 p - (1-p) \log_2 (1-p)$$

This is the binary entropy function $h(p)$, which peaks at $p = 1/2$ with $h(1/2) = 1$ bit.

### Key Properties

1. **Non-negativity:** $H(X) \geq 0$ with equality iff $X$ is deterministic.
2. **Maximum entropy:** $H(X) \leq \log_2 |\mathcal{X}|$ with equality iff $X$ is uniform.
3. **Concavity:** $H(\lambda p + (1-\lambda) q) \geq \lambda H(p) + (1-\lambda) H(q)$ for $\lambda \in [0,1]$.
4. **Chain rule:** $H(X_1, X_2, \ldots, X_n) = \sum_{i=1}^{n} H(X_i | X_1, \ldots, X_{i-1})$.
5. **Subadditivity:** $H(X_1, \ldots, X_n) \leq \sum_{i=1}^{n} H(X_i)$.

**Proof of concavity:** The function $f(p) = -p \log p$ is concave (since $f''(p) = -1/p < 0$ for $p > 0$). Entropy is a sum of concave functions, hence concave.

---

## 2. Source Coding Theorem (Shannon's First Theorem)

### The Problem

Prove that entropy is the fundamental limit of lossless compression.

### The Formula

**Theorem (Shannon, 1948).** For a discrete memoryless source (DMS) with entropy $H(X)$, the minimum achievable average code length $L^*$ satisfies:

$$H(X) \leq L^* < H(X) + 1$$

For block codes encoding $n$ source symbols jointly:

$$H(X) \leq \frac{L_n^*}{n} < H(X) + \frac{1}{n}$$

As $n \to \infty$, the average bits per symbol approaches $H(X)$.

### Proof Sketch

**Achievability (upper bound):** Construct a Shannon code with lengths $l_i = \lceil -\log_2 p_i \rceil$. Then:

$$L = \sum_i p_i l_i < \sum_i p_i (-\log_2 p_i + 1) = H(X) + 1$$

These lengths satisfy the Kraft inequality, so a prefix-free code exists.

**Converse (lower bound):** For any uniquely decodable code with lengths $l_1, \ldots, l_n$:

$$L - H(X) = \sum_i p_i \log_2 \frac{p_i}{2^{-l_i}} - \log_2 \left(\sum_i 2^{-l_i}\right) \cdot \sum_i p_i$$

By the Kraft inequality $\sum_i 2^{-l_i} \leq 1$, and by Gibbs' inequality (non-negativity of KL divergence):

$$L \geq H(X)$$

---

## 3. The Kraft Inequality and Optimal Codes

### The Problem

Characterize which sets of codeword lengths admit a prefix-free code.

### The Formula

**Kraft Inequality.** A prefix-free binary code with codeword lengths $l_1, l_2, \ldots, l_m$ exists if and only if:

$$\sum_{i=1}^{m} 2^{-l_i} \leq 1$$

**McMillan's extension:** The same inequality is necessary and sufficient for any uniquely decodable code (not just prefix-free).

### Proof of Necessity

Model codewords as leaves in a complete binary tree of depth $l_{\max}$. Each codeword of length $l_i$ "claims" $2^{l_{\max} - l_i}$ leaves of the full tree. Since no two prefix-free codewords share a descendant leaf:

$$\sum_{i=1}^{m} 2^{l_{\max} - l_i} \leq 2^{l_{\max}}$$

Dividing both sides by $2^{l_{\max}}$ yields the Kraft inequality.

### Optimal Code Construction

The optimal codeword lengths satisfy $l_i^* = -\log_2 p_i$ (which may not be integer). Huffman coding finds the optimal integer lengths. For symbol probabilities that are powers of $1/2$, Huffman coding achieves entropy exactly.

---

## 4. Channel Capacity and the BSC

### The Problem

Calculate the capacity of a binary symmetric channel and interpret the noisy channel coding theorem.

### The Formula

A binary symmetric channel (BSC) flips each bit independently with probability $p$:

```
Input       Output
  0 ---1-p--- 0
    \       /
     p     p
    /       \
  1 ---1-p--- 1
```

The capacity is:

$$C_{\text{BSC}} = \max_{P(X)} I(X; Y) = 1 - H(p)$$

where $H(p) = -p \log_2 p - (1-p) \log_2(1-p)$ is the binary entropy function.

### Derivation

$$I(X; Y) = H(Y) - H(Y|X)$$

Since the channel is memoryless, $H(Y|X) = H(p)$ regardless of the input distribution. To maximize $I(X; Y)$, we maximize $H(Y)$. Since $Y$ is binary, $H(Y) \leq 1$ with equality when $P(Y=0) = P(Y=1) = 1/2$, which occurs when the input is uniform. Therefore:

$$C = 1 - H(p)$$

| Crossover $p$ | Capacity $C$ (bits) | Interpretation |
|---|---|---|
| 0.0 | 1.000 | Perfect channel |
| 0.01 | 0.919 | Near-perfect |
| 0.1 | 0.531 | Noisy but usable |
| 0.25 | 0.189 | Very noisy |
| 0.5 | 0.000 | Completely random (useless) |

---

## 5. Asymptotic Equipartition Property (AEP)

### The Problem

Establish the information-theoretic analog of the law of large numbers.

### The Formula

**AEP (Shannon-McMillan-Breiman Theorem).** For a sequence $X_1, X_2, \ldots, X_n$ of i.i.d. random variables drawn from $P(X)$:

$$-\frac{1}{n} \log_2 P(X_1, X_2, \ldots, X_n) \xrightarrow{p} H(X) \quad \text{as } n \to \infty$$

### Consequences: The Typical Set

The typical set $A_\epsilon^{(n)}$ consists of sequences $(x_1, \ldots, x_n)$ satisfying:

$$2^{-n(H(X) + \epsilon)} \leq P(x_1, \ldots, x_n) \leq 2^{-n(H(X) - \epsilon)}$$

Properties of the typical set:

1. $P(A_\epsilon^{(n)}) > 1 - \epsilon$ for sufficiently large $n$.
2. $|A_\epsilon^{(n)}| \leq 2^{n(H(X) + \epsilon)}$.
3. $|A_\epsilon^{(n)}| \geq (1 - \epsilon) \cdot 2^{n(H(X) - \epsilon)}$.

**Interpretation:** Among the $|\mathcal{X}|^n$ possible sequences, only about $2^{nH(X)}$ are "typical" -- and they carry almost all the probability. This is why compression to $H(X)$ bits per symbol is possible: we only need to index the typical sequences.

---

## 6. Rate-Distortion Theory

### The Problem

Characterize the fundamental limits of lossy compression.

### The Formula

For a source $X$ and distortion measure $d(x, \hat{x})$, the rate-distortion function is:

$$R(D) = \min_{P(\hat{x}|x): \mathbb{E}[d(X, \hat{X})] \leq D} I(X; \hat{X})$$

**For a Bernoulli($p$) source with Hamming distortion:**

$$R(D) = \begin{cases} H(p) - H(D) & \text{if } 0 \leq D \leq \min(p, 1-p) \\ 0 & \text{if } D \geq \min(p, 1-p) \end{cases}$$

**For a Gaussian source with squared-error distortion ($X \sim \mathcal{N}(0, \sigma^2)$):**

$$R(D) = \begin{cases} \frac{1}{2} \log_2 \frac{\sigma^2}{D} & \text{if } 0 < D \leq \sigma^2 \\ 0 & \text{if } D > \sigma^2 \end{cases}$$

**Interpretation:** $R(D)$ gives the minimum number of bits per source symbol needed to reconstruct the source within average distortion $D$. Below this rate, distortion $D$ is unachievable; above it, codes exist that achieve it.

---

## 7. Relationship to Thermodynamics (Landauer's Principle)

### The Problem

Connect information-theoretic entropy to physical entropy and the thermodynamics of computation.

### The Formula

**Landauer's Principle (1961).** Erasing one bit of information in a computational device dissipates at least:

$$E_{\min} = k_B T \ln 2$$

of heat, where $k_B = 1.38 \times 10^{-23}$ J/K is Boltzmann's constant and $T$ is the temperature in Kelvin.

At room temperature ($T = 300$ K):

$$E_{\min} \approx 2.87 \times 10^{-21} \text{ J} \approx 0.018 \text{ eV per bit erased}$$

### Connection to Shannon Entropy

The Boltzmann entropy of a system with $W$ microstates is:

$$S = k_B \ln W$$

For a system encoding $n$ bits: $W = 2^n$, so $S = n k_B \ln 2$.

Shannon entropy and thermodynamic entropy are related by:

$$S_{\text{thermo}} = k_B \ln 2 \cdot H_{\text{Shannon}}$$

**Maxwell's Demon resolution:** Szilard (1929) and Landauer (1961) showed that the demon must erase its memory to complete the cycle, and this erasure dissipates at least $k_B T \ln 2$ per bit, saving the second law of thermodynamics.

---

## 8. Applications to Cryptography

### The Problem

Apply information-theoretic tools to quantify the security of cryptographic systems.

### The Formula

**Perfect Secrecy (Shannon, 1949).** A cipher achieves perfect secrecy iff:

$$I(M; C) = 0$$

where $M$ is the plaintext and $C$ is the ciphertext. Equivalently, $H(M|C) = H(M)$: observing the ciphertext reveals nothing about the message.

**Shannon's bound:** For perfect secrecy:

$$H(K) \geq H(M)$$

The key must be at least as long as the message. The one-time pad achieves this bound with equality.

**Unicity distance:** The minimum ciphertext length $n_0$ at which a cipher can be broken (the key is uniquely determined):

$$n_0 = \frac{H(K)}{D}$$

where $D = \log_2 |\mathcal{M}| - H(M)$ is the redundancy of the plaintext language per symbol. For English ($D \approx 1.5$ bits/char) with a 128-bit key: $n_0 \approx 85$ characters.

---

## 9. Applications to Compression

### The Problem

Connect entropy to practical compression algorithms and their theoretical limits.

### Lempel-Ziv and Dictionary Methods

The LZ78/LZW family achieves the entropy rate for stationary ergodic sources asymptotically:

$$\lim_{n \to \infty} \frac{L_{\text{LZ}}(X_1^n)}{n} = H_\infty(X) \quad \text{a.s.}$$

where $H_\infty(X)$ is the entropy rate $\lim_{n \to \infty} H(X_1, \ldots, X_n)/n$.

### Practical Compression Performance

| Algorithm | Approach | English text (bits/char) | Entropy limit |
|---|---|---|---|
| ASCII (uncompressed) | Fixed-length | 8.0 | -- |
| Huffman | Symbol frequencies | ~4.5 | $H(X)$ |
| Arithmetic | Interval subdivision | ~4.2 | $H(X)$ |
| LZ77/gzip | Dictionary + Huffman | ~2.5 | $H_\infty(X)$ |
| PPM | Context modeling | ~2.0 | $H_\infty(X)$ |
| Neural (NNCP) | Neural prediction | ~1.2 | $H_\infty(X)$ |
| English entropy rate | Theoretical limit | ~1.0-1.5 | -- |

---

## 10. Worked Example: Complete Information-Theoretic Analysis

### Setup

Consider a joint distribution over $X \in \{0, 1\}$ and $Y \in \{0, 1\}$:

| | $Y=0$ | $Y=1$ |
|---|---|---|
| $X=0$ | 3/8 | 1/8 |
| $X=1$ | 1/8 | 3/8 |

### Marginals

$P(X=0) = 1/2$, $P(X=1) = 1/2$, $P(Y=0) = 1/2$, $P(Y=1) = 1/2$.

### Entropy Calculations

$H(X) = -2 \cdot \frac{1}{2} \log_2 \frac{1}{2} = 1$ bit

$H(Y) = 1$ bit

$H(X,Y) = -2 \cdot \frac{3}{8} \log_2 \frac{3}{8} - 2 \cdot \frac{1}{8} \log_2 \frac{1}{8} = \frac{3}{4} \cdot 1.415 + \frac{1}{4} \cdot 3 = 1.811$ bits

$H(Y|X) = H(X,Y) - H(X) = 1.811 - 1 = 0.811$ bits

$I(X;Y) = H(X) + H(Y) - H(X,Y) = 1 + 1 - 1.811 = 0.189$ bits

### Verification

$H(Y|X) = H(3/8, 1/8 \mid X=0) \cdot P(X=0) + H(1/8, 3/8 \mid X=1) \cdot P(X=1)$
$= 2 \cdot \frac{1}{2} \cdot h(3/4) = h(3/4) = 0.811$ bits. Consistent.

---

## Tips

- The AEP is the information-theoretic workhorse: it underlies proofs of both source coding and channel coding theorems. Master it first.
- When computing channel capacity, always check whether the optimal input distribution is uniform. For symmetric channels, it usually is.
- Rate-distortion theory is the lossy analog of source coding. Think of $R(D)$ as the "entropy remaining after you tolerate distortion $D$."
- Landauer's principle sets the ultimate physical limit on computation energy. Current transistors dissipate roughly $10^4$ times more than this limit.
- For cryptographic applications, information-theoretic security (perfect secrecy) is strictly stronger than computational security. The one-time pad is the only practical cipher achieving it.
- The KL divergence $D_{KL}(P \| Q)$ appears everywhere: maximum likelihood estimation minimizes $D_{KL}(\hat{P}_{\text{data}} \| P_\theta)$, variational inference minimizes $D_{KL}(Q \| P)$, and hypothesis testing uses it for error exponents.

## See Also

- coding-theory
- compression
- probability
- cryptography
- signal-processing
- complexity-theory

## References

- Shannon, C. E. "A Mathematical Theory of Communication" (1948), Bell System Technical Journal
- Shannon, C. E. "Communication Theory of Secrecy Systems" (1949), Bell System Technical Journal
- Cover, T. M. & Thomas, J. A. "Elements of Information Theory" (2nd ed., Wiley, 2006)
- MacKay, D. J. C. "Information Theory, Inference, and Learning Algorithms" (Cambridge, 2003)
- Landauer, R. "Irreversibility and Heat Generation in the Computing Process" (1961), IBM J. Res. Dev.
- Gallager, R. G. "Information Theory and Reliable Communication" (Wiley, 1968)
