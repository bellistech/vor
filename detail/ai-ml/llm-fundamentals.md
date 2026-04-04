# The Mathematics of Large Language Models -- Attention, Loss, and Scaling

> *Large language models are fundamentally next-token predictors trained via cross-entropy minimization over massive corpora. Their remarkable capabilities emerge from the interplay of quadratic-complexity attention, carefully tuned loss landscapes, and empirical scaling laws that predict performance as a function of compute, data, and parameters.*

---

## 1. Self-Attention Complexity (Linear Algebra)
### The Problem
Self-attention requires every token to attend to every other token in the sequence, creating a computational bottleneck that defines the practical limits of context windows.

### The Formula
$$\text{Attention}(Q, K, V) = \text{softmax}\!\left(\frac{QK^\top}{\sqrt{d_k}}\right)V$$

where $Q, K, V \in \mathbb{R}^{n \times d_k}$ and $n$ is the sequence length.

### Complexity Analysis
The matrix multiplication $QK^\top$ produces an $n \times n$ attention matrix:

$$\text{Time complexity: } O(n^2 \cdot d_k)$$
$$\text{Memory complexity: } O(n^2 + n \cdot d_k)$$

For multi-head attention with $h$ heads:

$$\text{MultiHead}(Q, K, V) = \text{Concat}(\text{head}_1, \ldots, \text{head}_h) W^O$$

$$\text{Total complexity: } O(n^2 \cdot d_{\text{model}})$$

### Worked Example
For LLaMA-2 7B with $n = 4096$, $d_{\text{model}} = 4096$, $h = 32$:

- Attention matrix size per head: $4096 \times 4096 = 16.7M$ entries
- Per layer (32 heads): $32 \times 16.7M = 536M$ FP16 multiplications
- Full model (32 layers): $32 \times 536M \approx 17.2B$ operations just for attention
- At 128K context: attention cost grows $\left(\frac{128K}{4K}\right)^2 = 1024\times$

## 2. Perplexity and Cross-Entropy Loss (Information Theory)
### The Problem
We need a principled way to measure how well a language model predicts text and to define the training objective that shapes the model's parameters.

### The Formula
Cross-entropy loss for next-token prediction:

$$\mathcal{L} = -\frac{1}{T}\sum_{t=1}^{T} \log P_\theta(x_t \mid x_{<t})$$

where $T$ is sequence length and $P_\theta$ is the model's predicted probability for the correct token.

Perplexity is the exponentiated cross-entropy:

$$\text{PPL} = \exp(\mathcal{L}) = \exp\!\left(-\frac{1}{T}\sum_{t=1}^{T} \log P_\theta(x_t \mid x_{<t})\right)$$

### Interpretation
$$\text{PPL} = k \implies \text{model is as uncertain as choosing uniformly among } k \text{ tokens}$$

- Perfect model: $\text{PPL} = 1$ (always assigns probability 1 to correct token)
- Random over 32K vocab: $\text{PPL} = 32{,}000$
- GPT-4 on typical English: $\text{PPL} \approx 5\text{--}10$

### Worked Example
Given a 3-token sequence where model assigns probabilities $[0.8, 0.3, 0.6]$ to correct tokens:

$$\mathcal{L} = -\frac{1}{3}\left(\ln 0.8 + \ln 0.3 + \ln 0.6\right)$$
$$= -\frac{1}{3}\left(-0.223 - 1.204 - 0.511\right) = \frac{1.938}{3} = 0.646$$
$$\text{PPL} = e^{0.646} \approx 1.91$$

## 3. Scaling Laws (Empirical Power Laws)
### The Problem
Training LLMs costs millions of dollars. We need to predict final model performance before committing resources, and to allocate compute optimally between model size and training data.

### The Kaplan Scaling Laws (OpenAI, 2020)
$$L(N) = \left(\frac{N_c}{N}\right)^{\alpha_N}, \quad \alpha_N \approx 0.076$$
$$L(D) = \left(\frac{D_c}{D}\right)^{\alpha_D}, \quad \alpha_D \approx 0.095$$
$$L(C) = \left(\frac{C_c}{C}\right)^{\alpha_C}, \quad \alpha_C \approx 0.050$$

where $N$ = parameters, $D$ = dataset tokens, $C$ = compute (FLOPs).

### The Chinchilla Scaling Law (DeepMind, 2022)
Optimal allocation: parameters and tokens should scale equally with compute budget.

$$N_{\text{opt}} \propto C^{0.5}, \quad D_{\text{opt}} \propto C^{0.5}$$

The "Chinchilla-optimal" ratio:

$$D_{\text{opt}} \approx 20 \cdot N$$

### Worked Example
For a 7B parameter model (Chinchilla-optimal):

$$D_{\text{opt}} = 20 \times 7\text{B} = 140\text{B tokens}$$
$$C \approx 6ND = 6 \times 7 \times 10^9 \times 140 \times 10^9 = 5.88 \times 10^{21} \text{ FLOPs}$$

On 8x A100 (80GB) at ~312 TFLOPS each (FP16):
$$\text{Time} = \frac{5.88 \times 10^{21}}{8 \times 312 \times 10^{12}} \approx 2.36 \times 10^6 \text{ seconds} \approx 27 \text{ days}$$

LLaMA-2 7B was trained on 2T tokens ($\approx 285 \times N$), significantly over the Chinchilla ratio, trading compute for better performance at inference time.

## 4. Emergent Abilities (Phase Transitions)
### The Problem
Certain capabilities (chain-of-thought reasoning, multi-step arithmetic, translation) appear suddenly as models scale, showing sharp phase transitions rather than gradual improvement.

### The Scaling Threshold Model
Performance on an emergent task can be modeled as:

$$P(\text{success}) = \begin{cases} \epsilon & \text{if } N < N^* \\ \sigma\!\left(\beta \cdot \log\frac{N}{N^*}\right) & \text{if } N \geq N^* \end{cases}$$

where $\sigma$ is the sigmoid function and $N^*$ is the critical parameter count.

### Observed Thresholds
```
| Capability              | Approx. Threshold | Evidence           |
|------------------------|-------------------|--------------------|
| Few-shot arithmetic     | ~10B parameters   | GPT-3 / PaLM      |
| Chain-of-thought        | ~60B parameters   | PaLM / Flan        |
| Multi-step reasoning    | ~100B parameters  | GPT-4 / Claude     |
| Code generation         | ~10B parameters   | Codex / StarCoder  |
```

### Alternative View (Schaeffer et al., 2023)
Emergent abilities may be artifacts of the metric:

$$\text{Exact-match metric: } \mathbb{1}[P(x) > \tau] \quad \text{shows sharp transition}$$
$$\text{Log-likelihood metric: } \log P(x) \quad \text{shows smooth scaling}$$

The choice of discontinuous evaluation metrics can create the appearance of phase transitions from underlying smooth improvement.

## 5. Temperature and Sampling (Probability Theory)
### The Formula
Temperature-scaled softmax over logits $z_i$:

$$P(x_i) = \frac{\exp(z_i / \tau)}{\sum_j \exp(z_j / \tau)}$$

- $\tau \to 0$: argmax (greedy), distribution collapses to one-hot
- $\tau = 1$: standard softmax
- $\tau \to \infty$: uniform distribution

### Top-p (Nucleus) Sampling
Select the smallest set $V_p$ such that:

$$\sum_{x_i \in V_p} P(x_i) \geq p$$

Then renormalize probabilities over $V_p$.

### Entropy of Sampling Distribution
$$H = -\sum_i P(x_i) \log P(x_i)$$

Higher temperature increases entropy (more randomness). For vocabulary $|V|$:

$$0 \leq H \leq \log |V|$$

## 6. Quantization Error (Numerical Analysis)
### The Formula
For weight matrix $W$ quantized to $\hat{W}$ with $b$ bits:

$$\|W - \hat{W}\|_F \leq \frac{\max(|W|) - \min(|W|)}{2^b - 1} \cdot \sqrt{mn}$$

where $m \times n$ is the matrix shape.

### GPTQ Objective
GPTQ minimizes layer-wise quantization error using second-order information:

$$\hat{W} = \arg\min_{\hat{W}} \|WX - \hat{W}X\|_2^2$$

where $X$ is a calibration dataset. The Hessian $H = 2X X^\top$ guides which weights to quantize first (those with smallest $H_{ii}^{-1}$).

## 7. Inference Arithmetic (Systems Performance)
### The Problem
Practitioners need to estimate inference costs, latency, and memory requirements before deploying a model.

### Memory Estimation
Total inference memory for a model with $N$ parameters, context length $n$, and batch size $b$:

$$M_{\text{total}} = M_{\text{weights}} + M_{\text{KV cache}} + M_{\text{activations}}$$

$$M_{\text{weights}} = N \times \frac{\text{bits}}{8}$$

$$M_{\text{KV cache}} = 2 \times L \times d \times n \times b \times \frac{\text{bits}}{8}$$

where $L$ is the number of layers and $d$ is the hidden dimension.

### Worked Example
For LLaMA-2 7B in FP16, context 4096, batch 1:

$$M_{\text{weights}} = 6.7 \times 10^9 \times 2 = 13.4 \text{ GB}$$
$$M_{\text{KV cache}} = 2 \times 32 \times 4096 \times 4096 \times 1 \times 2 = 2.1 \text{ GB}$$
$$M_{\text{total}} \approx 15.5 \text{ GB (plus ~1 GB for activations and overhead)}$$

In 4-bit quantization:
$$M_{\text{weights}} = 6.7 \times 10^9 \times 0.5 = 3.35 \text{ GB}$$
$$M_{\text{total}} \approx 5.5 \text{ GB (KV cache remains in FP16)}$$

### Tokens Per Second
The theoretical maximum throughput during generation (memory-bound regime):

$$\text{tokens/sec} = \frac{\text{Memory bandwidth (GB/s)}}{M_{\text{weights}} / b}$$

For an A100 (2 TB/s bandwidth), 7B FP16 model, batch 1:

$$\text{tokens/sec} = \frac{2000}{13.4} \approx 149 \text{ tokens/sec}$$

With batching ($b = 32$): the weight transfer is amortized, so throughput scales nearly linearly until KV cache saturates memory.

### FLOPs Per Token
During generation, each token requires approximately:

$$\text{FLOPs/token} \approx 2N \quad \text{(for a model with } N \text{ parameters)}$$

For LLaMA-2 7B: $\approx 13.4 \times 10^9$ FLOPs per generated token.

## Prerequisites
- linear-algebra (matrix multiplication, norms, eigenvalues)
- probability (softmax, conditional probability, Bayes' theorem)
- information-theory (entropy, cross-entropy, KL divergence)
- calculus (gradients, chain rule, optimization)
- numerical-methods (floating point representation, quantization)
