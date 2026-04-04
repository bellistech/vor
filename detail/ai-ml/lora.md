# The Mathematics of LoRA -- Low-Rank Decomposition and Parameter Efficiency

> *LoRA exploits the observation that weight updates during fine-tuning occupy a low-dimensional subspace. By constraining updates to a rank-r factorization, LoRA achieves comparable performance to full fine-tuning while training less than 1% of the parameters, with theoretical guarantees from matrix approximation theory.*

---

## 1. Low-Rank Factorization (Linear Algebra)
### The Problem
Full fine-tuning updates a weight matrix $W \in \mathbb{R}^{d \times k}$ by adding $\Delta W \in \mathbb{R}^{d \times k}$, requiring $d \times k$ trainable parameters per layer. For LLaMA-7B with $d = k = 4096$, that is 16.7M parameters per projection matrix. LoRA hypothesizes that $\Delta W$ has low intrinsic rank.

### The Formula
LoRA constrains the update to a rank-$r$ factorization:

$$W' = W + \frac{\alpha}{r} BA$$

where:
- $B \in \mathbb{R}^{d \times r}$ (initialized to zeros)
- $A \in \mathbb{R}^{r \times k}$ (initialized from $\mathcal{N}(0, \sigma^2)$)
- $\alpha$ is the scaling hyperparameter
- $r \ll \min(d, k)$ is the rank

The number of trainable parameters per adapted layer:

$$|\theta_{\text{LoRA}}| = r(d + k)$$

### Parameter Efficiency Ratio

$$\eta = \frac{|\theta_{\text{LoRA}}|}{|\theta_{\text{full}}|} = \frac{r(d + k)}{dk}$$

For $d = k$:

$$\eta = \frac{2r}{d}$$

### Worked Example
For LLaMA-7B attention projections ($d = k = 4096$) with $r = 16$:

$$|\theta_{\text{LoRA}}| = 16 \times (4096 + 4096) = 131{,}072$$
$$|\theta_{\text{full}}| = 4096 \times 4096 = 16{,}777{,}216$$
$$\eta = \frac{131{,}072}{16{,}777{,}216} = 0.0078 = 0.78\%$$

For all 4 attention projections across 32 layers:

$$\text{Total LoRA params} = 4 \times 32 \times 131{,}072 = 16{,}777{,}216 \approx 16.8\text{M}$$
$$\text{Total model params} = 6{,}738{,}415{,}616 \approx 6.7\text{B}$$
$$\text{Overall } \eta = \frac{16.8\text{M}}{6.7\text{B}} = 0.25\%$$

## 2. SVD and the Eckart-Young Theorem (Matrix Theory)
### The Problem
LoRA's effectiveness rests on the assumption that $\Delta W$ can be well-approximated by a low-rank matrix. The Eckart-Young-Mirsky theorem provides the theoretical foundation.

### The Formula
Any matrix $M \in \mathbb{R}^{m \times n}$ has a singular value decomposition:

$$M = U \Sigma V^\top = \sum_{i=1}^{\min(m,n)} \sigma_i \mathbf{u}_i \mathbf{v}_i^\top$$

where $\sigma_1 \geq \sigma_2 \geq \cdots \geq 0$ are the singular values.

The best rank-$r$ approximation (Eckart-Young theorem):

$$M_r = \sum_{i=1}^{r} \sigma_i \mathbf{u}_i \mathbf{v}_i^\top = \arg\min_{\text{rank}(\hat{M}) \leq r} \|M - \hat{M}\|_F$$

The approximation error:

$$\|M - M_r\|_F = \sqrt{\sum_{i=r+1}^{\min(m,n)} \sigma_i^2}$$

### Relative Error Bound

$$\frac{\|M - M_r\|_F}{\|M\|_F} = \sqrt{\frac{\sum_{i=r+1}^{p} \sigma_i^2}{\sum_{i=1}^{p} \sigma_i^2}}$$

### Worked Example
Suppose a weight update $\Delta W$ has singular values $\sigma = [10, 5, 2, 1, 0.5, 0.2, 0.1, \ldots]$.

Energy captured by rank-$r$ approximation:

$$\text{Energy}(r) = \frac{\sum_{i=1}^{r} \sigma_i^2}{\sum_i \sigma_i^2}$$

$$\text{Energy}(1) = \frac{100}{131.3} = 76.2\%$$
$$\text{Energy}(2) = \frac{125}{131.3} = 95.2\%$$
$$\text{Energy}(4) = \frac{130}{131.3} = 99.0\%$$
$$\text{Energy}(8) = \frac{131.05}{131.3} = 99.8\%$$

This shows why small $r$ (4-16) captures most of the fine-tuning signal: the singular values of $\Delta W$ decay rapidly.

## 3. The Scaling Factor (Optimization Theory)
### The Problem
The $\alpha / r$ scaling factor controls the magnitude of the LoRA update. Understanding its effect on gradient dynamics is essential for stable training.

### The Formula
The forward pass with LoRA:

$$h = Wx + \frac{\alpha}{r} BAx$$

The gradient with respect to $A$:

$$\frac{\partial \mathcal{L}}{\partial A} = \frac{\alpha}{r} B^\top \frac{\partial \mathcal{L}}{\partial h} x^\top$$

The gradient with respect to $B$:

$$\frac{\partial \mathcal{L}}{\partial B} = \frac{\alpha}{r} \frac{\partial \mathcal{L}}{\partial h} (Ax)^\top$$

### Effective Learning Rate
The effective learning rate for the LoRA update is:

$$\eta_{\text{eff}} = \eta \cdot \frac{\alpha}{r}$$

where $\eta$ is the optimizer learning rate. This explains why:
- Doubling $r$ while keeping $\alpha$ constant halves the update magnitude
- Setting $\alpha = 2r$ keeps $\eta_{\text{eff}} = 2\eta$ regardless of rank
- The common choice $\alpha = 2r$ with $\eta = 2 \times 10^{-4}$ gives stable training

### Initialization Analysis
With $B = 0$ and $A \sim \mathcal{N}(0, \sigma^2)$:

$$BA = 0 \quad \text{at initialization (no perturbation)}$$

After the first gradient step:

$$B_1 = -\eta \frac{\alpha}{r} \frac{\partial \mathcal{L}}{\partial h} (A_0 x)^\top$$

The initial update scale depends on $\|A_0\|_F \propto \sigma \sqrt{rk}$, which is why the Kaiming-like initialization $\sigma = 1/\sqrt{k}$ is commonly used for $A$.

## 4. Quantization Error in QLoRA (Numerical Analysis)
### The Problem
QLoRA combines 4-bit quantization of the base model with LoRA training. The quantization introduces error in the forward pass that the LoRA adapter must compensate for.

### The Formula
NormalFloat4 (NF4) quantization maps weights to one of $2^4 = 16$ values optimized for normally-distributed weights:

$$\hat{w} = Q_{\text{NF4}}(w) = \arg\min_{q \in \mathcal{Q}} |w - q|$$

where $\mathcal{Q}$ is the NF4 codebook, derived from the quantiles of $\mathcal{N}(0, 1)$.

The quantization error per element:

$$\epsilon = w - \hat{w}, \quad \mathbb{E}[\epsilon^2] \leq \frac{(\max \mathcal{Q} - \min \mathcal{Q})^2}{4 \cdot 2^{2b}}$$

### Double Quantization
QLoRA applies nested quantization: the quantization constants (scales) are themselves quantized to 8-bit:

$$\text{Memory per param} = 4 + \frac{32}{64} + \frac{8}{256} \approx 4.5 \text{ bits}$$

where:
- 4 bits for the weight
- 32-bit scale per block of 64 weights = 0.5 bits/weight
- 8-bit quantized scale per block of 256 = 0.03 bits/weight

### QLoRA Forward Pass
$$h = \hat{W}x + \frac{\alpha}{r}BAx$$

The LoRA component $BA$ operates in full precision (BF16), compensating for quantization error:

$$h = (W + \underbrace{(\hat{W} - W)}_{\text{quant error}} + \underbrace{\frac{\alpha}{r}BA}_{\text{LoRA update}})x$$

The LoRA adapter implicitly learns to offset quantization artifacts while adapting to the target task.

## 5. Intrinsic Dimensionality (Manifold Learning)
### The Problem
Aghajanyan et al. (2021) showed that pre-trained models have low intrinsic dimensionality for fine-tuning: the parameter space needed for adaptation is far smaller than the total parameter count.

### The Formula
Define a random projection $P \in \mathbb{R}^{D \times d}$ where $D$ is the full parameter count and $d$ is the intrinsic dimension. Fine-tuning in the projected space:

$$\theta = \theta_0 + P\phi, \quad \phi \in \mathbb{R}^d$$

The intrinsic dimensionality $d_{90}$ is the smallest $d$ such that:

$$\frac{\mathcal{L}(\theta_0 + P\phi^*) - \mathcal{L}(\theta^*_{\text{full}})}{\mathcal{L}(\theta_0) - \mathcal{L}(\theta^*_{\text{full}})} \leq 0.10$$

### Empirical Results
For a RoBERTa-base model:

$$D = 125{,}000{,}000 \quad \text{(125M full parameters)}$$
$$d_{90} \approx 200 \quad \text{(intrinsic dimensionality)}$$

This means $99.9998\%$ of the parameter space is redundant for fine-tuning. LoRA with $r = 8$ on attention projections provides $\sim$800K parameters, which is $4000\times$ the intrinsic dimension -- comfortably overcomplete.

The Johnson-Lindenstrauss lemma guarantees that random projections preserve pairwise distances in $\mathbb{R}^d$ with:

$$d \geq \frac{8 \ln n}{\epsilon^2}$$

dimensions, where $n$ is the number of points and $\epsilon$ is the distortion tolerance.

## 6. Adapter Merging (Linear Algebra)
### The Problem
At inference time, LoRA adapters can be merged into the base weights, eliminating runtime overhead. Understanding the merging operation ensures correctness.

### The Formula
The merged weight matrix:

$$W_{\text{merged}} = W + \frac{\alpha}{r}BA$$

This is a rank-$r$ perturbation. The merged matrix has full rank (generically):

$$\text{rank}(W_{\text{merged}}) = \min(d, k) \quad \text{(almost surely)}$$

### Multiple Adapter Merging
For two LoRA adapters with different tasks:

$$W_{\text{merged}} = W + \frac{\alpha_1}{r_1}B_1 A_1 + \frac{\alpha_2}{r_2}B_2 A_2$$

The merged update has rank at most $r_1 + r_2$. If the two adapters were trained on very different tasks, their subspaces may be nearly orthogonal:

$$\|B_1 A_1 - B_2 A_2\|_F \approx \sqrt{\|B_1 A_1\|_F^2 + \|B_2 A_2\|_F^2}$$

This near-orthogonality is why task arithmetic (adding/subtracting adapters) works in practice: different tasks occupy different subspaces of the weight update manifold.

### Weight Interpolation (Model Soups)
Linear interpolation between base and adapted weights:

$$W(\lambda) = W + \lambda \cdot \frac{\alpha}{r}BA, \quad \lambda \in [0, 1]$$

At $\lambda = 0$: base model. At $\lambda = 1$: fully adapted. Intermediate $\lambda$ values can improve out-of-distribution robustness by controlling adaptation strength.

## Prerequisites
- linear-algebra (matrix factorization, SVD, rank, norms, eigenvalues)
- optimization (gradient descent, learning rate schedules, convergence)
- numerical-analysis (quantization error, floating-point arithmetic)
- probability (normal distribution, random projections)
- matrix-theory (Eckart-Young theorem, low-rank approximation)
