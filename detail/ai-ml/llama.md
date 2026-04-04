# The Mathematics of LLaMA -- RoPE Encoding and Grouped-Query Attention

> *LLaMA's architectural innovations center on two key mathematical contributions: Rotary Position Embeddings (RoPE) that encode relative position through complex-plane rotations, and Grouped-Query Attention (GQA) that reduces the KV cache bottleneck through key-value head sharing. These enable efficient long-context inference at scale.*

---

## 1. Rotary Position Embeddings (Complex Analysis / Linear Algebra)
### The Problem
Transformers need positional information because self-attention is permutation-invariant. Absolute positional embeddings (GPT-2 style) fail to generalize beyond training length. RoPE encodes relative position through rotation, enabling extrapolation and efficient computation.

### The Formula
For a query or key vector $\mathbf{x} \in \mathbb{R}^d$, RoPE applies a rotation based on position $m$:

$$f_{\text{RoPE}}(\mathbf{x}, m) = \begin{pmatrix} x_1 \\ x_2 \\ x_3 \\ x_4 \\ \vdots \\ x_{d-1} \\ x_d \end{pmatrix} \odot \begin{pmatrix} \cos m\theta_1 \\ \cos m\theta_1 \\ \cos m\theta_2 \\ \cos m\theta_2 \\ \vdots \\ \cos m\theta_{d/2} \\ \cos m\theta_{d/2} \end{pmatrix} + \begin{pmatrix} -x_2 \\ x_1 \\ -x_4 \\ x_3 \\ \vdots \\ -x_d \\ x_{d-1} \end{pmatrix} \odot \begin{pmatrix} \sin m\theta_1 \\ \sin m\theta_1 \\ \sin m\theta_2 \\ \sin m\theta_2 \\ \vdots \\ \sin m\theta_{d/2} \\ \sin m\theta_{d/2} \end{pmatrix}$$

where the frequency for dimension pair $i$ is:

$$\theta_i = \text{base}^{-2i/d}, \quad i = 0, 1, \ldots, d/2 - 1$$

LLaMA uses $\text{base} = 10000$ (same as original Transformer).

### Complex Number Formulation
Viewing consecutive pairs $(x_{2i}, x_{2i+1})$ as complex numbers $z_i = x_{2i} + j \cdot x_{2i+1}$:

$$f_{\text{RoPE}}(z_i, m) = z_i \cdot e^{jm\theta_i}$$

This is simply multiplication by a unit complex number -- a rotation in the 2D plane by angle $m\theta_i$.

### The Key Property: Relative Position
The attention score between positions $m$ and $n$ depends only on the relative distance:

$$\langle f_{\text{RoPE}}(\mathbf{q}, m), f_{\text{RoPE}}(\mathbf{k}, n) \rangle = \text{Re}\left[\sum_{i=0}^{d/2-1} q_i^* k_i \cdot e^{j(m-n)\theta_i}\right]$$

This means attention is a function of $(m - n)$, not of $m$ and $n$ independently.

### Worked Example
For $d = 4$, position $m = 3$, base $= 10000$:

$$\theta_0 = 10000^{0/4} = 1, \quad \theta_1 = 10000^{-2/4} = 0.01$$

$$m\theta_0 = 3.0 \text{ rad}, \quad m\theta_1 = 0.03 \text{ rad}$$

For $\mathbf{x} = [1.0, 0.5, 0.8, 0.3]$:

Pair 1 ($\theta_0$): rotate $(1.0, 0.5)$ by $3.0$ rad:
$$x_1' = 1.0 \cos(3.0) - 0.5 \sin(3.0) = 1.0(-0.990) - 0.5(0.141) = -1.060$$
$$x_2' = 0.5 \cos(3.0) + 1.0 \sin(3.0) = 0.5(-0.990) + 1.0(0.141) = -0.354$$

Pair 2 ($\theta_1$): rotate $(0.8, 0.3)$ by $0.03$ rad:
$$x_3' = 0.8 \cos(0.03) - 0.3 \sin(0.03) = 0.8(0.9996) - 0.3(0.030) = 0.791$$
$$x_4' = 0.3 \cos(0.03) + 0.8 \sin(0.03) = 0.3(0.9996) + 0.8(0.030) = 0.324$$

Note: low-frequency dimensions ($\theta_1$) rotate slowly -- they encode coarse/long-range position, while high-frequency dimensions ($\theta_0$) encode fine/local position.

## 2. RoPE Context Extension (Frequency Analysis)
### The Problem
A model trained with max position $L$ fails when given positions $m > L$ because the rotation angles $m\theta_i$ exceed training distribution. Scaling methods modify the frequencies to accommodate longer contexts.

### Linear Scaling (Position Interpolation)
$$\theta_i' = \frac{\theta_i}{s}, \quad s = \frac{L'}{L}$$

This compresses all positions into the original range $[0, L]$:

$$m' = \frac{m}{s}, \quad m' \in [0, L] \text{ when } m \in [0, L']$$

### NTK-Aware Scaling
Rather than uniformly scaling all frequencies, NTK-aware scaling adjusts the base:

$$\text{base}' = \text{base} \cdot s^{d/(d-2)}$$

$$\theta_i' = (\text{base}')^{-2i/d} = \text{base}^{-2i/d} \cdot s^{-2i/(d-2)}$$

This preserves high frequencies (local position) while stretching low frequencies (global position).

### YaRN (Yet another RoPE extensioN)
YaRN combines NTK-aware scaling with an attention temperature correction:

$$\text{Attention}' = \text{softmax}\!\left(\frac{QK^\top}{\sqrt{d_k} \cdot t}\right)V$$

where $t = 0.1 \ln(s) + 1$ is a temperature factor that compensates for the entropy increase in attention distributions at longer contexts.

### Frequency Bands Analysis
For LLaMA-2 with $d = 128$ (per head), $L = 4096$:

$$\lambda_i = \frac{2\pi}{\theta_i} = 2\pi \cdot 10000^{2i/d}$$

- Shortest wavelength ($i = 0$): $\lambda_0 = 2\pi \approx 6.28$ positions
- Longest wavelength ($i = 63$): $\lambda_{63} = 2\pi \cdot 10000 \approx 62{,}832$ positions

At $4\times$ extension (16K context), only dimensions where $\lambda_i < 4096$ are "out of distribution." The higher-frequency dimensions (small $i$) need scaling while lower frequencies already have wavelengths exceeding the context.

## 3. Grouped-Query Attention (Linear Algebra / Complexity)
### The Problem
Standard Multi-Head Attention (MHA) requires separate Key and Value projections for each head. At inference time, these K and V matrices must be cached, creating a memory bottleneck proportional to number of heads.

### The Formula
In MHA, for $h$ heads each with dimension $d_k$:

$$\text{KV cache per token} = 2 \times h \times d_k \times \text{dtype\_bytes}$$

In GQA with $g$ KV groups (where $g$ divides $h$):

$$\text{KV cache per token} = 2 \times g \times d_k \times \text{dtype\_bytes}$$

The compression ratio:

$$\text{KV compression} = \frac{g}{h}$$

### GQA Variants
$$g = h: \quad \text{Multi-Head Attention (MHA) -- no sharing}$$
$$g = 1: \quad \text{Multi-Query Attention (MQA) -- all heads share one KV}$$
$$1 < g < h: \quad \text{Grouped-Query Attention (GQA) -- groups of heads share KV}$$

### Worked Example: LLaMA 3 8B
With $h = 32$ query heads and $g = 8$ KV groups, each group serves 4 query heads:

$$\text{Heads per KV group} = \frac{h}{g} = \frac{32}{8} = 4$$

KV cache memory per token (FP16):
$$\text{MHA: } 2 \times 32 \times 128 \times 2 = 16{,}384 \text{ bytes}$$
$$\text{GQA: } 2 \times 8 \times 128 \times 2 = 4{,}096 \text{ bytes}$$
$$\text{Reduction: } 4\times$$

For 128K context, 32 layers:
$$\text{MHA KV cache} = 128{,}000 \times 16{,}384 \times 32 = 64 \text{ GB}$$
$$\text{GQA KV cache} = 128{,}000 \times 4{,}096 \times 32 = 16 \text{ GB}$$

This $4\times$ reduction makes 128K context feasible on practical hardware.

### GQA Attention Computation
For query head $i$ in group $\lfloor ig/h \rfloor$:

$$\text{Attention}_i(Q_i, K_{\lfloor ig/h \rfloor}, V_{\lfloor ig/h \rfloor}) = \text{softmax}\!\left(\frac{Q_i K_{\lfloor ig/h \rfloor}^\top}{\sqrt{d_k}}\right)V_{\lfloor ig/h \rfloor}$$

Multiple query heads attend to the same KV projections but with different query projections, allowing diverse attention patterns while sharing the memory-intensive K and V states.

## 4. RMSNorm (Normalization Theory)
### The Problem
LLaMA replaces LayerNorm with RMSNorm for computational efficiency, removing the mean-centering step.

### The Formula
$$\text{RMSNorm}(\mathbf{x}) = \frac{\mathbf{x}}{\text{RMS}(\mathbf{x})} \odot \boldsymbol{\gamma}$$

where:

$$\text{RMS}(\mathbf{x}) = \sqrt{\frac{1}{d}\sum_{i=1}^{d} x_i^2 + \epsilon}$$

Compared to LayerNorm:

$$\text{LayerNorm}(\mathbf{x}) = \frac{\mathbf{x} - \mu}{\sigma} \odot \boldsymbol{\gamma} + \boldsymbol{\beta}$$

RMSNorm saves: one mean computation, one subtraction, and the bias parameter $\boldsymbol{\beta}$.

## 5. SwiGLU Activation (Gated Linear Units)
### The Formula
LLaMA uses SwiGLU in the feed-forward network:

$$\text{FFN}(\mathbf{x}) = (\text{Swish}(W_{\text{gate}} \mathbf{x}) \odot W_{\text{up}} \mathbf{x}) W_{\text{down}}$$

where $\text{Swish}(x) = x \cdot \sigma(x)$ and $\sigma$ is the sigmoid function.

The gate dimension is $\frac{8}{3}d$ (rounded to nearest multiple of 256), giving 3 weight matrices instead of 2 but with smaller intermediate dimension:

$$\text{Standard FFN params: } 2 \times d \times 4d = 8d^2$$
$$\text{SwiGLU FFN params: } 3 \times d \times \frac{8d}{3} = 8d^2$$

Same parameter count, but SwiGLU empirically achieves better loss at the same compute budget.

## 6. Training Compute Estimation (Systems Performance)
### The Problem
Understanding the compute requirements for LLaMA models helps practitioners estimate training costs and hardware needs.

### The Formula
Total training FLOPs for a model with $N$ parameters trained on $D$ tokens:

$$C \approx 6ND$$

The factor of 6 comes from: 2 FLOPs per parameter per forward pass (multiply-accumulate) times 3 (forward + backward + gradient accumulation).

### Worked Examples
LLaMA 2 7B trained on 2T tokens:

$$C = 6 \times 6.7 \times 10^9 \times 2 \times 10^{12} = 8.04 \times 10^{22} \text{ FLOPs}$$

On a cluster of 2048 A100 GPUs at 312 TFLOPS (BF16) each with 40% MFU (Model FLOPS Utilization):

$$\text{Effective throughput} = 2048 \times 312 \times 0.4 = 255{,}590 \text{ TFLOPS}$$

$$\text{Training time} = \frac{8.04 \times 10^{22}}{2.56 \times 10^{17}} \approx 314{,}000 \text{ seconds} \approx 3.6 \text{ days}$$

LLaMA 3.1 405B trained on 15.6T tokens:

$$C = 6 \times 405 \times 10^9 \times 15.6 \times 10^{12} = 3.79 \times 10^{25} \text{ FLOPs}$$

On 16,384 H100 GPUs at 990 TFLOPS with 40% MFU:

$$\text{Training time} = \frac{3.79 \times 10^{25}}{16384 \times 990 \times 10^{12} \times 0.4} \approx 5.84 \times 10^6 \text{ s} \approx 67.6 \text{ days}$$

### Cost Estimation
At $\sim$\$2/GPU-hour for H100 instances:

$$\text{Cost}_{405B} = 16384 \times 67.6 \times 24 \times \$2 \approx \$53.2\text{M}$$

This explains why only a few organizations can train frontier models, and why efficient fine-tuning (LoRA/QLoRA) is essential for adaptation.

## Prerequisites
- complex-analysis (Euler's formula, unit circle, rotation in complex plane)
- linear-algebra (dot product, matrix multiplication, orthogonal transformations)
- signal-processing (frequency analysis, wavelengths, Nyquist theorem)
- calculus (sigmoid function, chain rule for SwiGLU gradients)
- information-theory (attention entropy, softmax temperature)
