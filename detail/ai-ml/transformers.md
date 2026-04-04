# The Mathematics of Transformers -- Attention, Gradient Flow, and Positional Encoding

> *The transformer architecture rests on three mathematical pillars: scaled dot-product attention that computes contextualized representations through weighted inner products, residual connections that preserve gradient magnitude across deep stacks, and positional encoding that breaks the permutation symmetry of attention. Understanding these foundations explains both the power and the limitations of modern LLMs.*

---

## 1. Scaled Dot-Product Attention (Linear Algebra)
### The Problem
Given a sequence of $n$ tokens, each represented as a $d$-dimensional vector, we need a mechanism that allows every token to selectively aggregate information from all other tokens based on learned relevance patterns.

### The Formula
$$\text{Attention}(Q, K, V) = \text{softmax}\!\left(\frac{QK^\top}{\sqrt{d_k}}\right)V$$

where $Q = XW_Q$, $K = XW_K$, $V = XW_V$ and $W_Q, W_K \in \mathbb{R}^{d \times d_k}$, $W_V \in \mathbb{R}^{d \times d_v}$.

The scaling factor $\sqrt{d_k}$ is critical. Without it, the dot products grow with $d_k$:

$$\mathbb{E}[\mathbf{q} \cdot \mathbf{k}] = 0, \quad \text{Var}[\mathbf{q} \cdot \mathbf{k}] = d_k$$

For large $d_k$, the softmax inputs have high variance, pushing the softmax into saturation regions where gradients vanish. Dividing by $\sqrt{d_k}$ normalizes the variance to 1.

### Multi-Head Attention
$$\text{MultiHead}(Q, K, V) = \text{Concat}(\text{head}_1, \ldots, \text{head}_h)W^O$$

$$\text{head}_i = \text{Attention}(QW^Q_i, KW^K_i, VW^V_i)$$

Parameter count per layer:

$$|\theta_{\text{MHA}}| = h(d \cdot d_k + d \cdot d_k + d \cdot d_v) + hd_v \cdot d = 4d^2 \quad \text{(when } d_k = d_v = d/h\text{)}$$

### Worked Example
For $n = 3$ tokens, $d_k = 2$:

$$Q = \begin{pmatrix} 1 & 0 \\ 0 & 1 \\ 1 & 1 \end{pmatrix}, \quad K = \begin{pmatrix} 1 & 1 \\ 0 & 1 \\ 1 & 0 \end{pmatrix}$$

$$\frac{QK^\top}{\sqrt{2}} = \frac{1}{\sqrt{2}}\begin{pmatrix} 1 & 0 & 1 \\ 1 & 1 & 0 \\ 2 & 1 & 1 \end{pmatrix} = \begin{pmatrix} 0.707 & 0 & 0.707 \\ 0.707 & 0.707 & 0 \\ 1.414 & 0.707 & 0.707 \end{pmatrix}$$

After softmax (row-wise):

$$A = \begin{pmatrix} 0.377 & 0.185 & 0.377 \\ 0.422 & 0.422 & 0.208 \\ 0.506 & 0.250 & 0.250 \end{pmatrix}$$

Token 3 attends most strongly to token 1 (score 0.506), as expected from their high dot product.

## 2. The Softmax Function (Probability Theory)
### The Formula
$$\text{softmax}(z_i) = \frac{e^{z_i}}{\sum_{j=1}^{n} e^{z_j}}, \quad \sum_i \text{softmax}(z_i) = 1$$

### The Jacobian
The Jacobian of softmax is needed for backpropagation:

$$\frac{\partial \text{softmax}(z_i)}{\partial z_j} = \text{softmax}(z_i)(\delta_{ij} - \text{softmax}(z_j))$$

$$J_{\text{softmax}} = \text{diag}(\mathbf{p}) - \mathbf{p}\mathbf{p}^\top$$

where $\mathbf{p} = \text{softmax}(\mathbf{z})$.

### Saturation and Gradient Vanishing
When one logit dominates ($z_i \gg z_j$ for all $j \neq i$):

$$\text{softmax}(z_i) \to 1, \quad \frac{\partial \text{softmax}(z_i)}{\partial z_j} \to 0$$

This is the "attention collapse" problem: when attention becomes too peaked, gradients vanish and the model cannot learn to redistribute attention.

## 3. Gradient Flow Through Residual Connections (Calculus)
### The Problem
Deep transformers (32-126 layers) would be impossible to train without residual connections. Understanding the gradient dynamics explains why.

### The Formula
With residual connection, layer $l$ computes:

$$\mathbf{x}_{l+1} = \mathbf{x}_l + F_l(\mathbf{x}_l)$$

where $F_l$ is the attention or FFN sub-layer.

The gradient through $L$ layers:

$$\frac{\partial \mathcal{L}}{\partial \mathbf{x}_0} = \frac{\partial \mathcal{L}}{\partial \mathbf{x}_L} \prod_{l=0}^{L-1} \left(I + \frac{\partial F_l}{\partial \mathbf{x}_l}\right)$$

Expanding this product:

$$\frac{\partial \mathcal{L}}{\partial \mathbf{x}_0} = \frac{\partial \mathcal{L}}{\partial \mathbf{x}_L}\left(I + \sum_{l} \frac{\partial F_l}{\partial \mathbf{x}_l} + \sum_{l<m} \frac{\partial F_l}{\partial \mathbf{x}_l}\frac{\partial F_m}{\partial \mathbf{x}_m} + \cdots\right)$$

The identity term $I$ ensures a direct gradient path from loss to input -- the "gradient highway."

### Without Residual Connections
$$\frac{\partial \mathcal{L}}{\partial \mathbf{x}_0} = \frac{\partial \mathcal{L}}{\partial \mathbf{x}_L} \prod_{l=0}^{L-1} \frac{\partial F_l}{\partial \mathbf{x}_l}$$

If $\left\|\frac{\partial F_l}{\partial \mathbf{x}_l}\right\| < 1$ for all layers (contraction), the gradient norm decays exponentially:

$$\left\|\frac{\partial \mathcal{L}}{\partial \mathbf{x}_0}\right\| \leq \left\|\frac{\partial \mathcal{L}}{\partial \mathbf{x}_L}\right\| \prod_{l} \left\|\frac{\partial F_l}{\partial \mathbf{x}_l}\right\| \to 0$$

For $L = 32$ layers with $\left\|\frac{\partial F_l}{\partial \mathbf{x}_l}\right\| = 0.9$:

$$0.9^{32} = 0.035 \quad \text{(96.5\% gradient loss)}$$

With residual connections, the gradient magnitude stays $\geq 1$ due to the identity term.

## 4. Pre-Norm vs Post-Norm Analysis (Optimization Theory)
### The Problem
The placement of layer normalization relative to the residual connection affects training stability.

### Post-Norm Gradient Scale
$$\mathbf{x}_{l+1} = \text{LN}(\mathbf{x}_l + F_l(\mathbf{x}_l))$$

The layer norm Jacobian:

$$\frac{\partial \text{LN}(\mathbf{y})}{\partial \mathbf{y}} = \frac{1}{\sigma}\left(I - \frac{1}{d}\mathbf{1}\mathbf{1}^\top - \frac{\hat{\mathbf{y}}\hat{\mathbf{y}}^\top}{d}\right) \odot \boldsymbol{\gamma}$$

where $\hat{\mathbf{y}}$ is the normalized vector. This Jacobian has spectral norm $\leq \|\boldsymbol{\gamma}\|_\infty / \sigma$. In Post-Norm, the gradient must pass through LN at every layer, potentially amplifying or dampening gradients unpredictably.

### Pre-Norm Gradient Scale
$$\mathbf{x}_{l+1} = \mathbf{x}_l + F_l(\text{LN}(\mathbf{x}_l))$$

$$\frac{\partial \mathbf{x}_{l+1}}{\partial \mathbf{x}_l} = I + \frac{\partial F_l}{\partial \text{LN}} \cdot \frac{\partial \text{LN}}{\partial \mathbf{x}_l}$$

The identity path is unobstructed by layer norm, giving more stable gradients. This is why Pre-Norm converges more reliably for deep models (LLaMA, GPT-2+).

## 5. Sinusoidal Positional Encoding (Fourier Analysis)
### The Formula
$$PE_{(pos, 2i)} = \sin\!\left(\frac{pos}{10000^{2i/d}}\right)$$
$$PE_{(pos, 2i+1)} = \cos\!\left(\frac{pos}{10000^{2i/d}}\right)$$

### Relative Position Property
The encoding at position $pos + k$ can be expressed as a linear transformation of the encoding at position $pos$:

$$\begin{pmatrix} PE_{(pos+k, 2i)} \\ PE_{(pos+k, 2i+1)} \end{pmatrix} = \begin{pmatrix} \cos(k\omega_i) & \sin(k\omega_i) \\ -\sin(k\omega_i) & \cos(k\omega_i) \end{pmatrix} \begin{pmatrix} PE_{(pos, 2i)} \\ PE_{(pos, 2i+1)} \end{pmatrix}$$

where $\omega_i = 10000^{-2i/d}$. This rotation matrix depends only on the offset $k$, not on the absolute position -- enabling the model to learn relative position patterns.

### Dot Product Decay
$$\langle PE_{pos}, PE_{pos+k} \rangle = \sum_{i=0}^{d/2-1} \cos(k\omega_i)$$

This sum decreases with $|k|$ but does not monotonically decay (it oscillates), which limits the encoding's ability to represent distance directly. This is one reason learned and rotary encodings outperform sinusoidal in practice.

## 6. Computational Complexity (Algorithm Analysis)
### Per-Layer Cost
$$\text{Self-Attention: } O(n^2 d + nd^2) \quad \text{(attention matrix + projections)}$$
$$\text{Feed-Forward: } O(nd \cdot d_{ff}) = O(4nd^2) \quad \text{(standard } d_{ff} = 4d\text{)}$$
$$\text{Total per layer: } O(n^2 d + nd^2)$$

For $n \ll d$: FFN dominates ($O(nd^2)$)
For $n \gg d$: attention dominates ($O(n^2 d)$)

The crossover point where attention cost equals FFN cost:

$$n^2 d = nd^2 \implies n = d$$

For $d = 4096$ (LLaMA-7B), attention dominates when context length $> 4096$ tokens.

### Flash Attention Complexity
Standard: $O(n^2)$ memory for the attention matrix.
Flash Attention: $O(n)$ memory via block-wise computation (tiling), same $O(n^2)$ compute but far fewer HBM reads.

$$\text{IO complexity: } O\!\left(\frac{n^2 d^2}{M}\right) \quad \text{(Flash)} \quad \text{vs} \quad O(n^2 + nd) \quad \text{(Standard)}$$

where $M$ is SRAM size. Flash Attention is IO-aware: it reduces slow HBM access at the cost of recomputation.

## 7. Feed-Forward Network Approximation (Function Analysis)
### The Problem
The FFN sub-layer in each transformer block acts as a key-value memory that stores factual knowledge. Understanding its approximation capacity explains why wider FFNs improve factual recall.

### The Formula
The standard FFN computes:

$$\text{FFN}(\mathbf{x}) = W_2 \cdot \text{GELU}(W_1 \mathbf{x} + \mathbf{b}_1) + \mathbf{b}_2$$

where $W_1 \in \mathbb{R}^{d_{ff} \times d}$ and $W_2 \in \mathbb{R}^{d \times d_{ff}}$.

Each row of $W_1$ defines a "key" pattern $\mathbf{k}_i$, and the corresponding column of $W_2$ stores a "value" $\mathbf{v}_i$:

$$\text{FFN}(\mathbf{x}) \approx \sum_{i=1}^{d_{ff}} \text{GELU}(\mathbf{k}_i^\top \mathbf{x} + b_i) \cdot \mathbf{v}_i$$

This is a sum of gated memory lookups. The capacity (number of stored facts) scales with $d_{ff}$, which is why $d_{ff} = 4d$ is standard and why LLaMA uses $d_{ff} = 8d/3 \times 3 = 8d$ total parameters via SwiGLU.

### Universal Approximation
With GELU activation and $d_{ff} \to \infty$, the FFN can approximate any continuous function on compact subsets of $\mathbb{R}^d$. The approximation error for a Lipschitz function $f$ with constant $L$:

$$\|f - \text{FFN}\|_\infty \leq \frac{Ld}{\sqrt{d_{ff}}}$$

This provides a theoretical justification for scaling FFN width: doubling $d_{ff}$ reduces approximation error by $\sqrt{2}$.

## 8. Attention Head Specialization (Representation Theory)
### The Problem
Different attention heads learn to capture different linguistic relationships. Understanding head specialization explains redundancy, pruning, and multi-head design.

### The Formula
The attention pattern of head $i$ at layer $l$ can be characterized by its entropy:

$$H_i^{(l)} = -\sum_{j=1}^{n} A_{ij}^{(l)} \log A_{ij}^{(l)}$$

Low entropy heads ($H \approx 0$): attend to a single position (often syntactic: previous token, delimiter, first token).

High entropy heads ($H \approx \log n$): distributed attention (semantic aggregation over the full context).

### Head Importance Score
The importance of head $i$ at layer $l$ can be measured by the expected sensitivity of the loss:

$$I_i^{(l)} = \mathbb{E}_x\left[\left|\frac{\partial \mathcal{L}}{\partial \text{head}_i^{(l)}}\right| \cdot \left|\text{head}_i^{(l)}\right|\right]$$

Empirically, head importance follows a heavy-tailed distribution: a small fraction of heads are critical (removing them causes large loss increases), while many can be pruned with minimal impact. This underlies structured pruning approaches that remove 30-50% of heads with <1% perplexity degradation.

## Prerequisites
- linear-algebra (matrix multiplication, inner products, projections, norms)
- calculus (chain rule, Jacobians, gradient computation)
- probability (softmax, probability distributions, entropy)
- fourier-analysis (sinusoidal functions, frequency decomposition)
- complexity-theory (big-O notation, time/space tradeoffs)
- optimization (gradient descent, learning rate, convergence)
