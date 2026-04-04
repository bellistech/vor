# The Mathematics of NumPy — Linear Algebra, Broadcasting, and Numerical Computation

> *NumPy's ndarray is a mathematical object that implements strided memory access over a contiguous buffer, enabling the entire toolkit of linear algebra, tensor operations, and numerical methods to execute at near-C speed from Python. The broadcasting rules formalize a system of implicit dimension expansion rooted in the algebra of array shapes, while the linear algebra routines implement the foundational decompositions of computational mathematics.*

---

## 1. Broadcasting Algebra (Shape Calculus)
### The Problem
When operating on arrays of different shapes, NumPy must determine if they are compatible and what the output shape will be. This follows a formal set of rules that can be expressed algebraically.

### The Formula
Given two arrays with shapes $A = (a_1, a_2, \ldots, a_n)$ and $B = (b_1, b_2, \ldots, b_m)$:

1. Pad the shorter shape with leading 1s: if $n < m$, $A' = (1, \ldots, 1, a_1, \ldots, a_n)$

2. For each dimension $i$, the output dimension is:

$$c_i = \begin{cases} a_i & \text{if } a_i = b_i \\ \max(a_i, b_i) & \text{if } a_i = 1 \text{ or } b_i = 1 \\ \text{error} & \text{otherwise} \end{cases}$$

The output shape $C = (c_1, c_2, \ldots, c_{\max(n,m)})$.

The total number of operations:

$$\text{ops} = \prod_{i=1}^{\max(n,m)} c_i$$

### Worked Examples
**Example 1**: Shapes $(3, 4)$ and $(4,)$

Pad: $(3, 4)$ and $(1, 4)$

$$c_1 = \max(3, 1) = 3, \quad c_2 = 4 = 4$$

Output shape: $(3, 4)$. Operations: $12$.

**Example 2**: Shapes $(5, 1, 3)$ and $(4, 1)$

Pad: $(5, 1, 3)$ and $(1, 4, 1)$

$$c_1 = \max(5, 1) = 5, \quad c_2 = \max(1, 4) = 4, \quad c_3 = \max(3, 1) = 3$$

Output shape: $(5, 4, 3)$. Operations: $60$.

**Example 3**: Shapes $(3, 4)$ and $(3,)$ -- INCOMPATIBLE

Pad: $(3, 4)$ and $(1, 3)$. Dimension 2: $4 \neq 3$ and neither is 1. Error.

## 2. Singular Value Decomposition (Matrix Analysis)
### The Problem
SVD decomposes any $m \times n$ matrix into three matrices, providing the foundation for dimensionality reduction (PCA), pseudoinverse computation, and low-rank approximation.

### The Formula
For matrix $A \in \mathbb{R}^{m \times n}$:

$$A = U \Sigma V^T$$

Where:
- $U \in \mathbb{R}^{m \times m}$ — orthonormal left singular vectors ($U^TU = I$)
- $\Sigma \in \mathbb{R}^{m \times n}$ — diagonal matrix of singular values $\sigma_1 \geq \sigma_2 \geq \cdots \geq 0$
- $V \in \mathbb{R}^{n \times n}$ — orthonormal right singular vectors ($V^TV = I$)

The best rank-$k$ approximation (Eckart-Young theorem):

$$A_k = \sum_{i=1}^{k} \sigma_i u_i v_i^T$$

Approximation error:

$$\|A - A_k\|_F = \sqrt{\sum_{i=k+1}^{\min(m,n)} \sigma_i^2}$$

### Worked Examples
**Example**: A 1000x500 matrix with singular values decaying as $\sigma_i = 100 \cdot e^{-0.05i}$.

$$\sigma_1 = 100, \quad \sigma_{10} = 100 \cdot e^{-0.5} \approx 60.7, \quad \sigma_{50} = 100 \cdot e^{-2.5} \approx 8.2$$

Total energy: $\|A\|_F^2 = \sum_{i=1}^{500} \sigma_i^2$

Energy captured by rank-50 approximation:

$$\frac{\sum_{i=1}^{50} \sigma_i^2}{\sum_{i=1}^{500} \sigma_i^2} = \frac{\sum_{i=1}^{50} 10000 \cdot e^{-0.1i}}{\sum_{i=1}^{500} 10000 \cdot e^{-0.1i}} \approx \frac{10000 \cdot \frac{1-e^{-5}}{1-e^{-0.1}}}{10000 \cdot \frac{1-e^{-50}}{1-e^{-0.1}}} = \frac{1-e^{-5}}{1-e^{-50}} \approx 0.9933$$

A rank-50 approximation captures 99.3% of the energy, reducing storage from 500,000 to 75,500 values.

## 3. Eigenvalue Decomposition (Spectral Theory)
### The Problem
Eigendecomposition reveals the fundamental modes of a linear transformation. It underlies PCA, graph Laplacians, Markov chains, and quantum mechanics simulations.

### The Formula
For a square matrix $A \in \mathbb{R}^{n \times n}$:

$$Av = \lambda v$$

$$A = Q \Lambda Q^{-1}$$

Where $\Lambda = \text{diag}(\lambda_1, \ldots, \lambda_n)$ and columns of $Q$ are eigenvectors.

For symmetric $A$ (real eigenvalues, orthogonal eigenvectors):

$$A = Q \Lambda Q^T, \quad Q^TQ = I$$

The characteristic polynomial:

$$\det(A - \lambda I) = 0$$

Spectral radius: $\rho(A) = \max_i |\lambda_i|$

### Worked Examples
**Example**: Covariance matrix for PCA:

$$C = \begin{pmatrix} 4 & 2 \\ 2 & 3 \end{pmatrix}$$

Characteristic polynomial: $(4-\lambda)(3-\lambda) - 4 = \lambda^2 - 7\lambda + 8 = 0$

$$\lambda = \frac{7 \pm \sqrt{49 - 32}}{2} = \frac{7 \pm \sqrt{17}}{2}$$

$$\lambda_1 \approx 5.56, \quad \lambda_2 \approx 1.44$$

Variance explained by PC1: $\frac{5.56}{5.56 + 1.44} = \frac{5.56}{7.0} = 79.4\%$

## 4. Strided Memory Model (Computer Architecture)
### The Problem
NumPy's performance depends on how data is laid out in memory. The stride tuple defines the byte offset between consecutive elements along each dimension.

### The Formula
For an ndarray with shape $(d_1, d_2, \ldots, d_n)$ and itemsize $s$ bytes:

C-order (row-major) strides:

$$\text{stride}_k = s \cdot \prod_{j=k+1}^{n} d_j$$

F-order (column-major) strides:

$$\text{stride}_k = s \cdot \prod_{j=1}^{k-1} d_j$$

Memory offset for element at index $(i_1, i_2, \ldots, i_n)$:

$$\text{offset} = \text{base} + \sum_{k=1}^{n} i_k \cdot \text{stride}_k$$

Cache efficiency for traversal along axis $k$: proportional to $\frac{1}{\text{stride}_k}$.

### Worked Examples
**Example**: Array of shape $(3, 4, 5)$ with `float64` (8 bytes), C-order:

$$\text{stride}_3 = 8$$
$$\text{stride}_2 = 8 \times 5 = 40$$
$$\text{stride}_1 = 8 \times 5 \times 4 = 160$$

Strides: $(160, 40, 8)$.

Element at $(1, 2, 3)$: offset = $1 \times 160 + 2 \times 40 + 3 \times 8 = 264$ bytes.

Iterating along axis 2 (innermost): stride = 8 bytes = 1 float. Sequential memory access, cache-friendly.

Iterating along axis 0 (outermost): stride = 160 bytes = 20 floats. Strided access, potential cache misses.

## 5. Numerical Stability (Floating-Point Analysis)
### The Problem
Floating-point arithmetic introduces rounding errors that accumulate across operations. Understanding error bounds is critical for scientific computing.

### The Formula
Machine epsilon for IEEE 754 double precision:

$$\epsilon_{mach} = 2^{-52} \approx 2.22 \times 10^{-16}$$

For the sum of $n$ floating-point numbers, the relative error bound:

$$\left| \frac{\hat{S} - S}{S} \right| \leq (n-1) \epsilon_{mach} + O(\epsilon_{mach}^2)$$

Condition number of a matrix:

$$\kappa(A) = \|A\| \cdot \|A^{-1}\| = \frac{\sigma_{max}}{\sigma_{min}}$$

Relative error in solving $Ax = b$:

$$\frac{\|\hat{x} - x\|}{\|x\|} \leq \kappa(A) \cdot \frac{\|\delta b\|}{\|b\|}$$

### Worked Examples
**Example**: Matrix with $\sigma_{max} = 1000$, $\sigma_{min} = 0.001$:

$$\kappa(A) = \frac{1000}{0.001} = 10^6$$

A perturbation of $\|\delta b\|/\|b\| = 10^{-10}$ in the right-hand side causes:

$$\frac{\|\hat{x} - x\|}{\|x\|} \leq 10^6 \times 10^{-10} = 10^{-4}$$

We lose 4 digits of accuracy. With `float64` (16 digits), we retain about 12 significant digits.

For `float32` (7 digits), we retain only 3 digits — likely insufficient.

## 6. Pseudorandom Number Generation (Number Theory)
### The Problem
NumPy's `default_rng()` uses the PCG64 algorithm (Permuted Congruential Generator), which provides statistically excellent random numbers with a period of $2^{128}$.

### The Formula
PCG64 state transition (LCG step):

$$s_{n+1} = a \cdot s_n + c \pmod{2^{128}}$$

Where $a$ is the multiplier and $c$ is the increment.

Output permutation (XSL-RR):

$$\text{output} = \text{rotr}((s \oplus (s \gg 64)) \gg 58, s \gg 122)$$

For generating $N(\mu, \sigma^2)$ from $U(0,1)$ (Box-Muller transform):

$$Z_1 = \sqrt{-2 \ln U_1} \cos(2\pi U_2)$$
$$Z_2 = \sqrt{-2 \ln U_1} \sin(2\pi U_2)$$

Then $X = \mu + \sigma Z$.

### Worked Examples
**Example**: Generating 10,000 samples from $N(100, 15^2)$ and verifying:

Expected sample mean: $\mu = 100$
Expected standard error: $SE = \frac{15}{\sqrt{10000}} = 0.15$

95% confidence interval for sample mean: $100 \pm 1.96 \times 0.15 = [99.706, 100.294]$

Chi-squared test for normality with $k = 50$ bins:

$$\chi^2 = \sum_{i=1}^{50} \frac{(O_i - E_i)^2}{E_i}$$

With 49 degrees of freedom, critical value at $\alpha = 0.05$: $\chi^2_{0.05, 49} = 66.34$.

## Prerequisites
- linear-algebra, matrix-decomposition, floating-point-arithmetic, spectral-theory, computer-architecture, number-theory
