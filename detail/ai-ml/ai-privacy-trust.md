# The Mathematics of AI Privacy and Trust — Differential Privacy, Federated Learning, and Explainability Theory

> *Privacy and trust in AI systems rest on rigorous mathematical foundations: differential privacy provides information-theoretic guarantees through calibrated noise mechanisms, federated learning algorithms bound information leakage through secure aggregation protocols, homomorphic encryption enables computation on ciphertexts via lattice-based cryptography, and explainability methods draw on game theory (Shapley values) and optimization theory to attribute model behavior to input features. These mathematical tools transform qualitative trust requirements into quantifiable, verifiable properties.*

---

## 1. Differential Privacy Mathematics

### The Privacy-Utility Fundamental Trade-off

For any mechanism $\mathcal{M}$ that satisfies $\epsilon$-differential privacy, the utility loss is bounded below:

$$\text{Error}(\mathcal{M}) \geq \Omega\left(\frac{\Delta f}{\epsilon}\right)$$

where $\Delta f$ is the sensitivity of the query function. This is a fundamental limit: stronger privacy (smaller $\epsilon$) necessarily means more noise and lower utility. No mechanism can circumvent this bound.

### Laplace Mechanism — Derivation

For a function $f: \mathcal{D} \to \mathbb{R}$ with global sensitivity $\Delta f$, the Laplace mechanism adds noise from the Laplace distribution:

$$\mathcal{M}(D) = f(D) + \text{Lap}\left(\frac{\Delta f}{\epsilon}\right)$$

**Proof that this satisfies $\epsilon$-DP:**

For adjacent datasets $D_1, D_2$:

$$\frac{P[\mathcal{M}(D_1) = z]}{P[\mathcal{M}(D_2) = z]} = \frac{\exp(-\epsilon|z - f(D_1)|/\Delta f)}{\exp(-\epsilon|z - f(D_2)|/\Delta f)}$$

$$= \exp\left(\frac{\epsilon(|z - f(D_2)| - |z - f(D_1)|)}{\Delta f}\right)$$

By the triangle inequality: $|z - f(D_2)| - |z - f(D_1)| \leq |f(D_1) - f(D_2)| \leq \Delta f$

Therefore: $\frac{P[\mathcal{M}(D_1) = z]}{P[\mathcal{M}(D_2) = z]} \leq e^\epsilon$

### Gaussian Mechanism — Derivation

The Gaussian mechanism uses $\ell_2$ sensitivity:

$$\Delta_2 f = \max_{D_1 \sim D_2} \|f(D_1) - f(D_2)\|_2$$

$$\mathcal{M}(D) = f(D) + \mathcal{N}\left(0, \sigma^2 I\right) \quad \text{where} \quad \sigma = \frac{\Delta_2 f \sqrt{2\ln(1.25/\delta)}}{\epsilon}$$

This satisfies $(\epsilon, \delta)$-DP. The Gaussian mechanism is preferred for high-dimensional queries because $\ell_2$ sensitivity scales as $\sqrt{d}$ while $\ell_1$ sensitivity scales as $d$ for $d$-dimensional outputs.

### Composition Theorems — Detailed

**Basic Sequential Composition:**
If $\mathcal{M}_1$ is $(\epsilon_1, \delta_1)$-DP and $\mathcal{M}_2$ is $(\epsilon_2, \delta_2)$-DP, and their outputs are computed on the same dataset, their joint release is $(\epsilon_1 + \epsilon_2, \delta_1 + \delta_2)$-DP.

This is tight in the worst case but often loose in practice.

**Advanced Composition (Dwork, Rothblum, Vadhan 2010):**
For $k$ mechanisms, each $(\epsilon_0, \delta_0)$-DP:

$$\text{Total: } (\epsilon_0\sqrt{2k\ln(1/\delta')} + k\epsilon_0(e^{\epsilon_0} - 1), \quad k\delta_0 + \delta')\text{-DP}$$

For small $\epsilon_0$: the total scales as $O(\epsilon_0\sqrt{k})$ rather than $O(\epsilon_0 k)$.

**Renyi Differential Privacy Composition:**
For $(\alpha, \epsilon)$-RDP:
- Composition: $k$ applications give $(\alpha, k\epsilon)$-RDP (additive)
- Subsampling amplification: if mechanism $\mathcal{M}$ satisfies $(\alpha, \epsilon)$-RDP and we subsample with probability $q$, the subsampled mechanism satisfies approximately $(\alpha, q^2 \epsilon)$-RDP for large $\alpha$

**Conversion from RDP to $(\epsilon, \delta)$-DP:**

$$\epsilon = \min_{\alpha > 1} \left[\epsilon_{\text{RDP}}(\alpha) + \frac{\ln((\alpha-1)/\alpha) - \ln\delta + \ln(\alpha)}{\alpha - 1}\right]$$

This optimization is performed numerically over $\alpha$ to find the tightest bound.

### Privacy Budget Accounting for DP-SGD

In DP-SGD, each training step applies the Gaussian mechanism with subsampling. For $T$ steps with batch sampling probability $q = B/N$ and noise multiplier $\sigma$:

Each step satisfies $(\alpha, \epsilon_{\text{step}}(\alpha))$-RDP where:

$$\epsilon_{\text{step}}(\alpha) \leq \frac{\alpha}{2\sigma^2} + O(q^2/\sigma^2)$$

(using the Poisson subsampling amplification bound)

After $T$ steps by composition: $(\alpha, T \cdot \epsilon_{\text{step}}(\alpha))$-RDP

Convert to $(\epsilon, \delta)$-DP:

$$\epsilon = T \cdot \epsilon_{\text{step}}(\alpha^*) + \frac{\ln(1/\delta)}{\alpha^* - 1}$$

where $\alpha^*$ minimizes this expression.

**Practical implications:**
- Doubling batch size $B$ (thus $q$) allows doubling $\sigma$ while maintaining the same per-step privacy cost
- Larger batches are strictly better for the privacy-utility trade-off
- The number of epochs is the primary driver of total privacy cost

## 2. Federated Learning Algorithms

### FedAvg — Convergence Analysis

**Setup:** $K$ clients, each with local dataset $D_k$ of size $n_k$. Total data $n = \sum_k n_k$. Global objective:

$$\min_w F(w) = \sum_{k=1}^{K} \frac{n_k}{n} F_k(w) \quad \text{where} \quad F_k(w) = \frac{1}{n_k} \sum_{i \in D_k} \ell(w; x_i, y_i)$$

**FedAvg Algorithm:**
1. Server initializes $w_0$
2. For each round $t = 0, 1, \ldots, T-1$:
   a. Server selects subset $S_t$ of $m$ clients
   b. Server broadcasts $w_t$ to selected clients
   c. Each client $k \in S_t$ runs $E$ epochs of SGD:
      $w_k^{(t+1)} = w_t - \eta \sum_{e=1}^{E} \nabla F_k(w_k^{(e)})$
   d. Server aggregates:
      $w_{t+1} = \sum_{k \in S_t} \frac{n_k}{\sum_{j \in S_t} n_j} w_k^{(t+1)}$

**Convergence bound (Li et al. 2020):**

For $\mu$-strongly convex, $L$-smooth objectives with bounded gradient dissimilarity $\Gamma$:

$$\mathbb{E}[F(w_T)] - F^* \leq \frac{L}{\mu} \exp\left(-\frac{\mu T}{L}\right) (F(w_0) - F^*) + \frac{\sigma^2}{\mu n} + \frac{E\Gamma^2}{\mu}$$

The third term $\frac{E\Gamma^2}{\mu}$ reflects the cost of non-IID data: more local epochs $E$ and higher data heterogeneity $\Gamma$ lead to larger error floor.

### FedSGD — Analysis

FedSGD is the special case of FedAvg with $E = 1$ (single gradient step per round):

$$w_{t+1} = w_t - \eta \sum_{k \in S_t} \frac{n_k}{n} \nabla F_k(w_t)$$

This is equivalent to centralized mini-batch SGD with the batch distributed across clients. Convergence is well-understood but communication cost is high (one round per gradient step).

**Communication efficiency comparison:**
- FedSGD: $T$ rounds for $T$ gradient steps
- FedAvg with $E$ local epochs: $T/E$ rounds for $T$ gradient steps (approximately)
- Trade-off: fewer rounds but potentially slower convergence due to client drift

### FL Privacy Guarantees

**User-level DP in FL:**
Each user's entire dataset is protected (not just individual records):

$$\text{User-level sensitivity} = \max_k \|\nabla F_k(w)\|_2$$

This is much larger than record-level sensitivity because one user may contribute thousands of records.

Clipping and noise:

$$\tilde{g}_k = \text{clip}(g_k, C) + \mathcal{N}(0, \sigma^2 C^2 I / m^2)$$

where $g_k = w_k^{(t+1)} - w_t$, $C$ is the clipping bound, and $m$ is the number of participating clients.

**Privacy amplification by subsampling in FL:**
If only $m$ out of $K$ clients participate in each round (Poisson sampling with probability $q = m/K$):

The per-round privacy cost is amplified: $\epsilon_{\text{round}} \approx q \cdot \epsilon_{\text{step}}$ for small $q$ and $\epsilon_{\text{step}}$.

## 3. Homomorphic Encryption Schemes

### BFV Scheme (Brakerski/Fan-Vercauteren)

**Key Generation:**
- Parameters: polynomial degree $n$, coefficient modulus $q$, plaintext modulus $t$
- Secret key: $s \leftarrow R_2$ (binary polynomial)
- Public key: $\text{pk} = ([-a \cdot s + e]_q, a)$ where $a \leftarrow R_q$, $e \leftarrow \chi$ (error distribution)

**Encryption:**
$$\text{ct} = ([pk_0 \cdot u + e_1 + \Delta \cdot m]_q, [pk_1 \cdot u + e_2]_q)$$
where $\Delta = \lfloor q/t \rfloor$ and $m$ is the plaintext polynomial.

**Homomorphic Operations:**
- Addition: $\text{ct}_{\text{add}} = \text{ct}_1 + \text{ct}_2$ (component-wise modular addition)
- Multiplication: $\text{ct}_{\text{mult}} = \text{ct}_1 \otimes \text{ct}_2$ (tensor product + relinearization)

**Noise growth:**
- Addition: noise grows linearly $e_{\text{add}} \approx e_1 + e_2$
- Multiplication: noise grows multiplicatively $e_{\text{mult}} \approx e_1 \cdot e_2$
- After $L$ levels of multiplication: $e \approx e_0^{2^L}$
- When noise exceeds $q/2t$, decryption fails

### CKKS Scheme (Cheon-Kim-Kim-Song)

CKKS is specifically designed for approximate arithmetic on real/complex numbers, making it the most suitable for ML.

**Encoding:** Map a vector of complex numbers to a polynomial:
$$\text{encode}: \mathbb{C}^{n/2} \to R_q$$

using the canonical embedding (inverse DFT on roots of the cyclotomic polynomial).

**Approximate arithmetic:**
$$\text{decrypt}(\text{ct}_1 + \text{ct}_2) \approx m_1 + m_2$$
$$\text{decrypt}(\text{ct}_1 \cdot \text{ct}_2) \approx m_1 \cdot m_2$$

with controlled approximation error that grows with computation depth.

**Rescaling:** After each multiplication, the ciphertext modulus is reduced to control noise:
$$\text{ct}' = \lfloor \text{ct} / p \rceil$$

This "uses up" one level of the modulus chain, limiting the total multiplicative depth.

**ML-specific operations in CKKS:**
- Matrix-vector multiplication: encode matrix rows in slots, use rotation + multiply-add
- Polynomial activation: approximate ReLU with polynomial $\sum_i a_i x^i$ (depth $= \lceil \log_2 d \rceil$ for degree $d$)
- Softmax: requires exponential approximation + inverse → high depth, major bottleneck

## 4. Synthetic Data Quality Metrics

### Statistical Fidelity Metrics

**Kolmogorov-Smirnov Distance:**
For continuous features, measure maximum CDF difference:

$$D_{\text{KS}} = \sup_x |F_{\text{real}}(x) - F_{\text{synth}}(x)|$$

Lower is better. Aggregate across features:
$$D_{\text{KS}}^{\text{avg}} = \frac{1}{d} \sum_{j=1}^{d} D_{\text{KS}}^{(j)}$$

**Jensen-Shannon Divergence:**
Symmetric, bounded divergence between distributions:

$$\text{JSD}(P \| Q) = \frac{1}{2} D_{\text{KL}}(P \| M) + \frac{1}{2} D_{\text{KL}}(Q \| M) \quad \text{where } M = \frac{P + Q}{2}$$

$\text{JSD} \in [0, \ln 2]$. Values < 0.1 indicate good fidelity.

**Correlation Preservation:**
Compare correlation matrices of real and synthetic data:

$$\Delta_{\text{corr}} = \|C_{\text{real}} - C_{\text{synth}}\|_F / \sqrt{d(d-1)/2}$$

where $\|\cdot\|_F$ is the Frobenius norm.

### ML Utility Metrics

**Train on Synthetic, Test on Real (TSTR):**
$$\text{TSTR} = \frac{\text{Accuracy}(\text{model trained on synthetic, tested on real})}{\text{Accuracy}(\text{model trained on real, tested on real})}$$

Values > 0.9 indicate high utility. This is the most practically meaningful metric.

**Train on Real, Test on Synthetic (TRTS):**
$$\text{TRTS} = \frac{\text{Accuracy}(\text{model trained on real, tested on synthetic})}{\text{Accuracy}(\text{model trained on real, tested on real})}$$

Measures whether the synthetic data covers the real data distribution.

### Privacy Metrics for Synthetic Data

**Distance to Closest Record (DCR):**
For each synthetic record $s$, compute distance to nearest real record:

$$\text{DCR}(s) = \min_{r \in D_{\text{real}}} d(s, r)$$

If $\text{DCR}(s) = 0$, the synthetic record is identical to a real record (privacy failure).

**Privacy loss metric:**
Compare DCR distribution of synthetic data to DCR distribution of a holdout set:

$$\text{Privacy Score} = P[\text{DCR}_{\text{synth}} > \text{DCR}_{\text{holdout}}]$$

Values near 0.5 indicate the synthetic data is no closer to training data than random holdout data (good privacy). Values near 0 indicate memorization.

## 5. Privacy-Utility Tradeoff Analysis

### Pareto Frontier

For a given task, the privacy-utility trade-off forms a Pareto frontier:

$$\mathcal{P} = \{(\epsilon, U(\epsilon)) : U(\epsilon) = \max_{\mathcal{M} \in \text{DP}(\epsilon)} \text{Utility}(\mathcal{M})\}$$

The frontier is monotonically increasing (more privacy budget → more utility) and concave (diminishing returns from additional privacy budget).

**Empirical estimation:**
Train models at multiple $\epsilon$ values and plot accuracy vs. $\epsilon$:

| $\epsilon$ | Accuracy (MNIST) | Accuracy (CIFAR-10) | Accuracy (IMDB) |
|------------|------------------|---------------------|-----------------|
| 0.5 | ~90% | ~45% | ~75% |
| 1.0 | ~95% | ~55% | ~80% |
| 3.0 | ~97% | ~65% | ~85% |
| 8.0 | ~98% | ~72% | ~88% |
| ∞ (no DP) | ~99% | ~93% | ~92% |

Key insight: the utility cost of privacy depends heavily on the task. Simple tasks (MNIST) tolerate strong privacy; complex tasks (CIFAR-10) suffer significantly.

### Optimal Privacy Budget Allocation

When answering multiple queries, allocate privacy budget to maximize total utility:

$$\max \sum_i U_i(\epsilon_i) \quad \text{s.t.} \quad \sum_i \epsilon_i \leq \epsilon_{\text{total}}$$

For queries with diminishing returns (concave $U_i$), the optimal allocation equalizes marginal utility:

$$\frac{\partial U_i}{\partial \epsilon_i} = \frac{\partial U_j}{\partial \epsilon_j} \quad \forall i, j$$

Allocate more budget to queries with higher marginal utility (e.g., more important analytics, more sensitive decisions).

## 6. Trust Calibration

### Formal Trust Model

Define calibrated trust as alignment between perceived and actual reliability:

$$\text{Calibration Error} = \mathbb{E}[|\text{Trust}(x) - P(\text{correct} | x)|]$$

A perfectly calibrated user's trust in the AI matches the AI's actual accuracy for each type of input.

**Expected Calibration Error (ECE):**

$$\text{ECE} = \sum_{b=1}^{B} \frac{|B_b|}{n} |\text{acc}(B_b) - \text{conf}(B_b)|$$

where inputs are binned by model confidence, $\text{acc}(B_b)$ is accuracy in bin $b$, and $\text{conf}(B_b)$ is average confidence in bin $b$.

### Trust-Performance Dynamics

Trust evolves based on observed AI performance:

$$T_{t+1} = \alpha \cdot T_t + (1 - \alpha) \cdot O_t$$

where $T_t$ is trust at time $t$, $O_t \in \{0, 1\}$ is the observed outcome (correct/incorrect), and $\alpha \in (0, 1)$ is the trust inertia (how slowly trust changes).

Properties:
- Trust recovers slowly after failures ($\alpha$ close to 1)
- Trust builds gradually with consistent performance
- A single dramatic failure can destroy years of built trust (asymmetric loss)
- Different users have different $\alpha$ values (trust disposition)

## 7. Explainability Fidelity

### LIME Fidelity Analysis

LIME's local approximation quality depends on:

1. **Kernel width $\sigma$:** Controls how local the explanation is
   - Too small: noisy explanations (few perturbed samples contribute)
   - Too large: explanation not local (averages over dissimilar regions)

2. **Number of perturbations $N$:** More samples → more stable explanations
   - Stability: $\text{Var}[\hat{\phi}_i] \propto 1/N$
   - Practical minimum: $N \geq 5000$ for tabular, $N \geq 1000$ for text

3. **Fidelity metric:** How well the local model approximates the true model

$$R^2_{\text{local}} = 1 - \frac{\sum_j \pi_j (f(x_j) - g(x_j))^2}{\sum_j \pi_j (f(x_j) - \bar{f})^2}$$

where $\pi_j$ are proximity weights, $f$ is the black-box model, and $g$ is the local linear model.

### SHAP — Theoretical Properties

SHAP values satisfy four axioms (Shapley axioms):

1. **Efficiency:** $\sum_i \phi_i = f(x) - E[f(X)]$ (attributions sum to prediction minus baseline)

2. **Symmetry:** If features $i$ and $j$ contribute equally in all coalitions, $\phi_i = \phi_j$

3. **Null player:** If feature $i$ contributes nothing in any coalition, $\phi_i = 0$

4. **Linearity:** For combined models $f = \alpha f_1 + \beta f_2$: $\phi_i^f = \alpha \phi_i^{f_1} + \beta \phi_i^{f_2}$

**Computational complexity:**
Computing exact Shapley values requires evaluating $f$ for all $2^d$ feature coalitions → $O(2^d)$ model evaluations.

KernelSHAP approximation: $O(d \log d)$ model evaluations using weighted least squares regression on coalition evaluations.

TreeSHAP exact computation: $O(TLD^2)$ where $T$ = number of trees, $L$ = maximum leaves, $D$ = maximum depth. Polynomial in model size, independent of feature count.

**Interaction SHAP:**
Shapley interaction values capture pairwise feature interactions:

$$\Phi_{ij} = \sum_{S \subseteq N \setminus \{i,j\}} \frac{|S|!(|N|-|S|-2)!}{2(|N|-1)!} \Delta_{ij}(S)$$

where $\Delta_{ij}(S) = f(S \cup \{i,j\}) - f(S \cup \{i\}) - f(S \cup \{j\}) + f(S)$.
