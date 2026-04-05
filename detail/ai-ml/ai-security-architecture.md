# The Theory of AI Security — Attack Surface Analysis, Model Confidentiality, and Privacy-Preserving ML

> *AI security theory extends classical computer security into a domain where the attack surface includes the statistical properties of the model itself: adversarial examples exploit the geometry of decision boundaries, model extraction attacks leverage the information content of prediction APIs, and privacy-preserving techniques draw on cryptographic and information-theoretic foundations to enable computation over sensitive data without exposure. These theoretical frameworks provide the mathematical basis for building defensible AI systems.*

---

## 1. AI Attack Surface Analysis

### Formal Attack Surface Model

The attack surface of an AI system can be modeled as a tuple:

$$\mathcal{A} = (\mathcal{D}, \mathcal{T}, \mathcal{M}, \mathcal{I}, \mathcal{S})$$

where:
- $\mathcal{D}$: Data surface (training data, validation data, inference inputs)
- $\mathcal{T}$: Training surface (algorithms, hyperparameters, infrastructure)
- $\mathcal{M}$: Model surface (architecture, weights, gradients)
- $\mathcal{I}$: Inference surface (API endpoints, input/output processing)
- $\mathcal{S}$: Supply chain surface (dependencies, pre-trained components)

For each surface element $a \in \mathcal{A}$, we define:
- $\text{Exposure}(a)$: degree of attacker accessibility
- $\text{Sensitivity}(a)$: impact if compromised
- $\text{Controls}(a)$: set of security measures protecting $a$
- $\text{Residual Risk}(a) = \text{Exposure}(a) \times \text{Sensitivity}(a) \times (1 - \text{Effectiveness}(\text{Controls}(a)))$

### Threat Actor Capabilities

AI-specific threat actors are characterized by their knowledge level:

**White-box attacker:** Full access to model architecture, weights, training data, and gradients. This is the strongest adversary model and represents insider threats or post-extraction scenarios.

**Black-box attacker:** Access only to the model's input-output behavior via queries. Subdivided into:
- Score-based: receives full probability vector
- Label-only: receives only top-1 prediction
- Decision-based: receives only binary (accept/reject)

**Gray-box attacker:** Partial knowledge, such as knowing the architecture but not the weights, or having access to a subset of training data.

The security guarantee of a defense is only meaningful when stated relative to a specific attacker model. A defense proven secure against black-box attacks may fail trivially against white-box attackers.

## 2. Model Confidentiality

### Model Extraction Attacks — Theory

**Information-theoretic view:**
A model $f$ with $n$ parameters contains at most $n \cdot b$ bits of information (where $b$ is bits per parameter). An extraction attack attempts to recover this information through queries.

**Query complexity bounds:**

For a $d$-dimensional linear model: $\Theta(d)$ queries suffice for exact extraction. The attacker solves a linear system $Ax = b$ where rows are query inputs and $b$ contains corresponding outputs.

For ReLU networks with $L$ layers and $n_l$ neurons per layer: the model partitions input space into $O(\prod_l n_l)$ linear regions. Exact extraction requires identifying all region boundaries.

For a single hidden layer with $n$ ReLU neurons and $d$-dimensional input:
$$\text{Query complexity} = O(n \cdot d)$$

This follows from the fact that each neuron contributes a hyperplane in $\mathbb{R}^d$, and $d+1$ points on each hyperplane determine it.

**Practical extraction via knowledge distillation:**

The attacker trains a student model $f_S$ to mimic the target $f_T$:

$$f_S = \arg\min_{g} \mathbb{E}_{x \sim \mathcal{D}_q}[\text{KL}(f_T(x) \| g(x))]$$

The query distribution $\mathcal{D}_q$ significantly affects extraction quality:
- Natural data from the same domain: highest fidelity, requires domain knowledge
- Synthetic data (random noise): works surprisingly well for simple models
- Active learning queries: optimal information gain per query

**Fidelity metrics:**

$$\text{Agreement}(f_S, f_T) = P_{x \sim \mathcal{D}}[\arg\max f_S(x) = \arg\max f_T(x)]$$

$$\text{Fidelity}(f_S, f_T) = 1 - \mathbb{E}_{x \sim \mathcal{D}}[\text{KL}(f_T(x) \| f_S(x))]$$

### Extraction Defenses

**Query detection:**
Monitor for query patterns indicative of extraction:
- Unusually uniform input distribution (random querying)
- Systematic boundary probing (inputs close to decision boundaries)
- High query rate from single source
- Out-of-distribution queries (inputs unlike normal traffic)

**Defense effectiveness metric:**

$$\text{Defense Quality} = \frac{\text{Extraction Cost with Defense}}{\text{Extraction Cost without Defense}}$$

A defense that increases extraction cost by 100x while degrading legitimate accuracy by < 1% is considered effective.

**Watermarking for ownership verification:**
Embed a watermark $w$ in model $f$ such that:
1. $\text{Accuracy}(f_w) \approx \text{Accuracy}(f)$ (utility preserved)
2. $P[\text{Verify}(w, f_w) = \text{true}] > 1 - \delta$ (reliable detection)
3. $P[\text{Verify}(w, f') = \text{true}] < \epsilon$ for unrelated $f'$ (low false positive)
4. Watermark survives fine-tuning, pruning, and distillation (robustness)

The watermark verification uses a set of trigger inputs $\{(x_i^*, y_i^*)\}$ where the watermarked model produces specific outputs:

$$\text{Verify}(w, f) = \mathbb{1}\left[\frac{1}{|T|}\sum_{(x^*, y^*) \in T} \mathbb{1}[f(x^*) = y^*] > \tau\right]$$

## 3. Training Data Security

### Poisoning Detection Theory

**Statistical detection:**
Clean data follows distribution $P_{\text{clean}}$. Poisoned data is drawn from a mixture:

$$P_{\text{observed}} = (1 - \alpha) P_{\text{clean}} + \alpha P_{\text{poison}}$$

where $\alpha$ is the poisoning rate. Detection reduces to hypothesis testing:

$$H_0: \alpha = 0 \quad \text{vs} \quad H_1: \alpha > 0$$

Using the likelihood ratio test:

$$\Lambda = \frac{\sup_{\alpha > 0, P_{\text{poison}}} L(\alpha, P_{\text{poison}} | D)}{\sup L(0 | D)}$$

The detection power depends on:
- Poisoning rate $\alpha$ (lower → harder to detect)
- Distribution divergence $D_{\text{KL}}(P_{\text{poison}} \| P_{\text{clean}})$
- Sample size $n$

**Spectral Signatures (Tran et al.):**
Poisoned samples create a detectable spectral signature in the feature space. For a dataset with feature representations $\{h_i\}$:

1. Compute centered covariance: $\Sigma = \frac{1}{n}\sum_i (h_i - \bar{h})(h_i - \bar{h})^T$
2. Compute top singular vector $v_1$ of $\Sigma$
3. Compute outlier score: $s_i = (h_i^T v_1)^2$
4. Remove samples with highest $s_i$ scores

Theoretical justification: poisoned samples share a common perturbation direction, which aligns with a top singular vector of the feature covariance matrix.

**Activation Clustering:**
Using the penultimate layer activations $\{a_i\}$ for each class:

1. Apply dimensionality reduction (PCA, UMAP)
2. Cluster using DBSCAN or Gaussian mixture
3. Classes with more than one cluster may contain poisoned subsets
4. Smaller clusters are suspicious (poisoned data is typically minority)

### Backdoor Detection

**Neural Cleanse:**
For each target class $y_t$, find the minimal perturbation $\delta$ that causes any input to be classified as $y_t$:

$$\delta^* = \arg\min_\delta \|\delta\|_1 \quad \text{s.t.} \quad f(x + \delta) = y_t \quad \forall x$$

If the minimal perturbation for some class $y_t$ is anomalously small compared to other classes, a backdoor targeting $y_t$ likely exists.

**Anomaly Index:**

$$\text{AI}(y_t) = \frac{\|\delta_{y_t}^*\|_1 - \text{median}_y \|\delta_y^*\|_1}{\text{MAD}_y \|\delta_y^*\|_1}$$

where MAD is the median absolute deviation. Classes with $\text{AI} > 2$ are flagged as potential backdoor targets.

## 4. Inference Attacks

### Membership Inference — Theory

**Formal definition:**
Given model $f$ trained on dataset $D$, and a target sample $(x, y)$, determine:

$$M(f, x, y) \to \{\text{member}, \text{non-member}\}$$

**Overfitting connection:**
Membership inference exploits the generalization gap. Let $\ell_{\text{train}}$ and $\ell_{\text{test}}$ be training and test losses:

$$\text{MI Advantage} \leq O(\ell_{\text{test}} - \ell_{\text{train}})$$

A model with zero generalization gap is information-theoretically immune to membership inference.

**Shadow model approach (Shokri et al.):**

1. Train $k$ shadow models $\{f_1^s, \ldots, f_k^s\}$ on datasets sampled from similar distribution
2. For each shadow model, collect $(f_j^s(x), y, \text{member/non-member})$ tuples
3. Train attack model $A: (f(x), y) \to \{0, 1\}$
4. Apply attack model to target model outputs

**Metric-based attacks (simpler, often equally effective):**

$$M(f, x, y) = \begin{cases} \text{member} & \text{if } f(x)_y > \tau \\ \text{non-member} & \text{otherwise} \end{cases}$$

where $\tau$ is calibrated on a reference dataset. More sophisticated variants use:
- Modified prediction entropy: $H(f(x))$
- Per-class thresholds
- Calibrated confidence scores

### Model Inversion — Theory

**Optimization-based inversion:**

Given a model $f$ and a target class $y$, reconstruct a representative input:

$$x^* = \arg\max_x \log f(x)_y - \lambda_1 \|x\|_2^2 - \lambda_2 \text{TV}(x)$$

where $\text{TV}(x)$ is the total variation regularizer encouraging spatial smoothness:

$$\text{TV}(x) = \sum_{i,j} |x_{i+1,j} - x_{i,j}| + |x_{i,j+1} - x_{i,j}|$$

**GAN-enhanced inversion:**
Use a pre-trained generator $G$ to constrain the search to realistic images:

$$z^* = \arg\max_z \log f(G(z))_y$$

The reconstructed image $x^* = G(z^*)$ is more realistic and may more closely resemble actual training samples.

**Privacy risk metric:**
A model leaks information about individual $i$ if:

$$I(f; x_i | D_{-i}) > 0$$

where $I$ is mutual information, and $D_{-i}$ is the dataset without individual $i$. Differential privacy provides the standard defense, bounding this leakage to $\epsilon$ per individual.

## 5. LLM-Specific Threats

### Prompt Injection Taxonomy

**Direct Prompt Injection:**
User-supplied input contains instructions that override the system prompt. Formally:

$$f(\text{system\_prompt} \oplus \text{user\_input}) \neq f_{\text{intended}}(\text{user\_input})$$

where $\oplus$ denotes concatenation in the context window.

Categories of direct injection:
1. **Goal hijacking:** Redirect the model to a different task entirely
2. **Prompt leaking:** Extract the system prompt content
3. **Jailbreaking:** Bypass safety training to produce prohibited content
4. **Payload execution:** Cause the model to invoke tools/APIs with attacker-chosen parameters

**Indirect Prompt Injection:**
Malicious instructions embedded in data retrieved from external sources:

$$\text{data} = \text{retrieve}(\text{query})$$
$$f(\text{system\_prompt} \oplus \text{user\_input} \oplus \text{data}) \to \text{attacker\_goal}$$

Attack vectors for indirect injection:
- Poisoned web pages (retrieved via search/browsing)
- Malicious documents (processed via file upload)
- Compromised APIs (returning adversarial content)
- Manipulated database entries (surfaced via RAG)
- Social media content (ingested for analysis)

**Formal defense hierarchy:**
The instruction hierarchy principle orders context by privilege:

$$\text{Priority}(\text{system}) > \text{Priority}(\text{user}) > \text{Priority}(\text{tool\_output}) > \text{Priority}(\text{retrieved\_data})$$

Training models to respect this hierarchy reduces (but does not eliminate) injection risk.

### Jailbreaking Theory

**Taxonomy of jailbreaking techniques:**

1. **Persona-based:** Assign the model an alternative identity that does not have safety constraints
2. **Encoding-based:** Use base64, ROT13, pig latin, or custom encodings to bypass content filters
3. **Multi-turn escalation:** Gradually shift conversation toward prohibited territory
4. **Hypothetical framing:** "In a fictional world where..." or "For a novel I'm writing..."
5. **Instruction manipulation:** "Ignore previous instructions" or reformulate as a technical question
6. **Token-level attacks:** Adversarial suffixes found via gradient-based optimization (GCG attack)

**GCG (Greedy Coordinate Gradient) attack:**
Find an adversarial suffix $s$ such that:

$$\arg\max_s P(\text{harmful response} | \text{prompt} \oplus s)$$

Using a gradient-based search over discrete tokens:
1. Compute gradient of target loss w.r.t. token embeddings
2. For each position, find top-$k$ replacement tokens
3. Greedily substitute tokens to minimize loss
4. Iterate until model produces target (harmful) prefix

This produces seemingly random token sequences that reliably jailbreak models.

**Defense robustness analysis:**
A defense $D$ against jailbreaking is $(\epsilon, \delta)$-robust if:

$$P[\text{jailbreak succeeds} | D \text{ active}] \leq \delta \quad \forall \text{attacks with cost} \leq \epsilon$$

Currently, no defense achieves formal robustness guarantees against unbounded adversaries. Practical defenses aim for increasing the cost $\epsilon$ required for successful jailbreaking.

## 6. Differential Privacy Theory

### Formal Definitions

**$(\epsilon, \delta)$-Differential Privacy:**
A randomized mechanism $\mathcal{M}: \mathcal{D} \to \mathcal{R}$ satisfies $(\epsilon, \delta)$-DP if for all adjacent datasets $D, D'$ (differing in one record) and all measurable sets $S \subseteq \mathcal{R}$:

$$P[\mathcal{M}(D) \in S] \leq e^\epsilon \cdot P[\mathcal{M}(D') \in S] + \delta$$

- Pure DP: $\delta = 0$ (strongest guarantee)
- Approximate DP: $\delta > 0$ (allows small probability of catastrophic failure)
- Typically require $\delta < 1/n$ where $n$ is dataset size

**Renyi Differential Privacy (RDP):**
$\mathcal{M}$ satisfies $(\alpha, \epsilon)$-RDP if for all adjacent $D, D'$:

$$D_\alpha(\mathcal{M}(D) \| \mathcal{M}(D')) = \frac{1}{\alpha - 1} \log \mathbb{E}\left[\left(\frac{P[\mathcal{M}(D) = o]}{P[\mathcal{M}(D') = o]}\right)^\alpha\right] \leq \epsilon$$

RDP provides tighter composition bounds than basic $(\epsilon, \delta)$-DP, which is why it is preferred for accounting in DP-SGD.

**Conversion from RDP to $(\epsilon, \delta)$-DP:**

$$\epsilon_{\text{total}} = \min_\alpha \left[\epsilon_{\text{RDP}}(\alpha) + \frac{\log(1/\delta)}{\alpha - 1}\right]$$

### Mechanisms

**Laplace Mechanism:**
For a function $f: \mathcal{D} \to \mathbb{R}^d$ with global sensitivity $\Delta f = \max_{D \sim D'} \|f(D) - f(D')\|_1$:

$$\mathcal{M}(D) = f(D) + (Z_1, \ldots, Z_d) \quad \text{where } Z_i \sim \text{Lap}(\Delta f / \epsilon)$$

Satisfies $\epsilon$-DP (pure).

**Gaussian Mechanism:**
For $\ell_2$ sensitivity $\Delta_2 f$:

$$\mathcal{M}(D) = f(D) + \mathcal{N}(0, \sigma^2 I) \quad \text{where } \sigma = \frac{\Delta_2 f \sqrt{2\ln(1.25/\delta)}}{\epsilon}$$

Satisfies $(\epsilon, \delta)$-DP.

### Composition Theorems

**Basic composition:** If $\mathcal{M}_1$ is $(\epsilon_1, \delta_1)$-DP and $\mathcal{M}_2$ is $(\epsilon_2, \delta_2)$-DP, their composition is $(\epsilon_1 + \epsilon_2, \delta_1 + \delta_2)$-DP.

**Advanced composition (Dwork et al.):** $k$ applications of $(\epsilon, \delta)$-DP mechanisms yield:

$$(\epsilon', k\delta + \delta')\text{-DP where } \epsilon' = \sqrt{2k \ln(1/\delta')} \cdot \epsilon + k\epsilon(e^\epsilon - 1)$$

**RDP composition:** Simply additive in the RDP parameter: $k$ applications of $(\alpha, \epsilon)$-RDP yield $(\alpha, k\epsilon)$-RDP. This additive property makes RDP the preferred accounting method for iterative algorithms like DP-SGD.

### Privacy Budget

The total privacy expenditure across all computations on a dataset. For DP-SGD with $T$ training steps:

$$\epsilon_{\text{total}} \approx q \sqrt{T} \cdot \frac{\sqrt{2\ln(1/\delta)}}{\sigma}$$

where $q$ is the sampling probability (batch size / dataset size) and $\sigma$ is the noise multiplier.

Practical privacy budget allocation:
- $\epsilon < 1$: Strong privacy (significant utility loss)
- $1 \leq \epsilon \leq 10$: Moderate privacy (reasonable utility)
- $\epsilon > 10$: Weak privacy (limited protection against MI)
- $\delta < 1/n$: Standard requirement

## 7. Federated Learning Security

### Byzantine Attacks

In federated learning, malicious clients can send arbitrary gradient updates. A Byzantine attacker controls $f$ out of $n$ clients and can send any value in place of their true gradient.

**Byzantine-resilient aggregation rules:**

Coordinate-wise Median:
$$\text{Agg}(g_1, \ldots, g_n)_j = \text{median}(g_{1,j}, \ldots, g_{n,j})$$

Tolerates up to $f < n/2$ Byzantine clients. Convergence rate: $O(d/n + df/n^2)$ where $d$ is dimension.

Krum:
Select the gradient whose sum of distances to its $n - f - 1$ nearest neighbors is minimal:

$$g^* = \arg\min_{g_i} \sum_{j \in \text{nn}(i, n-f-1)} \|g_i - g_j\|^2$$

Tolerates $f < (n-3)/2$ Byzantine clients.

Trimmed Mean:
For each coordinate, remove the $\beta$ fraction of largest and smallest values, then average the rest:

$$\text{Agg}(g_1, \ldots, g_n)_j = \frac{1}{n - 2\lfloor\beta n\rfloor} \sum_{i=\lfloor\beta n\rfloor+1}^{n - \lfloor\beta n\rfloor} g_{(i),j}$$

where $g_{(i),j}$ is the $i$-th order statistic. Tolerates $f < \beta n$.

### Model Poisoning in FL

**Targeted model poisoning (Bhagoji et al.):**
A malicious client crafts updates that introduce a backdoor while being stealthy:

$$g_{\text{malicious}} = \lambda \cdot g_{\text{backdoor}} + (1 - \lambda) \cdot g_{\text{honest}}$$

The blending parameter $\lambda$ trades off attack effectiveness vs. detectability.

**Scaling attack:**
Amplify the malicious update to overcome averaging:

$$g_{\text{malicious}} = \frac{n}{f} \cdot g_{\text{backdoor}}$$

where $n$ is the total number of clients and $f$ is the number of compromised clients. After aggregation, the backdoor update has the same magnitude as if all clients contributed it.

## 8. Secure Multi-Party Computation for ML

### Secret Sharing for ML

**Shamir's Secret Sharing:**
Split a secret $s$ into $n$ shares such that any $t$ shares can reconstruct $s$, but $t-1$ shares reveal nothing.

Encode $s$ as constant term of a random polynomial of degree $t-1$:

$$p(x) = s + a_1 x + a_2 x^2 + \cdots + a_{t-1} x^{t-1}$$

Share $i$ is the point $(i, p(i))$. Reconstruction via Lagrange interpolation:

$$s = p(0) = \sum_{i \in S} p(i) \prod_{j \in S, j \neq i} \frac{j}{j - i}$$

**Additive Secret Sharing for ML:**
Split each value $x$ into $n$ shares: $x = x_1 + x_2 + \cdots + x_n \pmod{p}$

Addition is local: $(x + y)_i = x_i + y_i$
Multiplication requires communication (Beaver triples):
$xy = (x_1 + x_2)(y_1 + y_2) = x_1 y_1 + x_1 y_2 + x_2 y_1 + x_2 y_2$

**ML operations on shares:**
Linear operations (matrix multiply, add, ReLU approximation) are relatively efficient. Non-linear operations (softmax, division, comparison) require garbled circuits or additional rounds of communication.

**Overhead analysis:**
For a neural network with $L$ layers and $W$ total parameters:
- Communication: $O(W \cdot L)$ field elements per inference
- Rounds: $O(L)$ (one round per non-linear layer)
- Computation: $O(W)$ per party (same as plaintext, plus overhead for modular arithmetic)
- Total slowdown: typically $1,000 - 10,000\times$ compared to plaintext inference

This makes SMPC practical for small models or low-throughput applications, but impractical for large-scale LLM inference with current technology.
