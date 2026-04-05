# The Mathematics of Linear Algebra for Computer Science (Decompositions, Algorithms, and Deep Results)

> *Linear algebra is the mathematics of data. Every pixel on your screen, every recommendation from a search engine, every prediction from a neural network passes through a matrix. Understanding the geometry behind these computations separates the practitioner who tunes hyperparameters from the one who invents the next algorithm.*

---

## 1. Singular Value Decomposition: Derivation and Geometry

### The Problem

Derive the singular value decomposition (SVD) of an arbitrary $m \times n$ real matrix, and explain its geometric meaning as a sequence of rotation, scaling, and rotation.

### The Formula

**Theorem (SVD).** For any $A \in \mathbb{R}^{m \times n}$ with $\text{rank}(A) = r$, there exist orthogonal matrices $U \in \mathbb{R}^{m \times m}$, $V \in \mathbb{R}^{n \times n}$ and a diagonal matrix $\Sigma \in \mathbb{R}^{m \times n}$ such that:

$$A = U \Sigma V^T$$

where $\Sigma = \text{diag}(\sigma_1, \sigma_2, \ldots, \sigma_r, 0, \ldots, 0)$ with $\sigma_1 \ge \sigma_2 \ge \cdots \ge \sigma_r > 0$.

**Derivation.** Consider the symmetric positive semi-definite matrix $A^T A \in \mathbb{R}^{n \times n}$. Since $A^T A$ is symmetric, the spectral theorem guarantees an orthonormal eigenbasis $\{v_1, \ldots, v_n\}$ with eigenvalues $\lambda_1 \ge \lambda_2 \ge \cdots \ge \lambda_n \ge 0$.

Define the singular values $\sigma_i = \sqrt{\lambda_i}$ for $i = 1, \ldots, r$ (the positive eigenvalues).

For $i = 1, \ldots, r$, define:

$$u_i = \frac{1}{\sigma_i} A v_i$$

We verify orthonormality:

$$u_i^T u_j = \frac{1}{\sigma_i \sigma_j} v_i^T A^T A v_j = \frac{1}{\sigma_i \sigma_j} v_i^T \lambda_j v_j = \frac{\lambda_j}{\sigma_i \sigma_j} \delta_{ij} = \delta_{ij}$$

Extend $\{u_1, \ldots, u_r\}$ to an orthonormal basis $\{u_1, \ldots, u_m\}$ of $\mathbb{R}^m$ (the additional vectors span the left null space of $A$).

Set $V = [v_1 | \cdots | v_n]$ and $U = [u_1 | \cdots | u_m]$. Then:

$$AV = [Av_1 | \cdots | Av_r | 0 | \cdots | 0] = [\sigma_1 u_1 | \cdots | \sigma_r u_r | 0 | \cdots | 0] = U\Sigma$$

Since $V$ is orthogonal, $A = U \Sigma V^T$. $\square$

**Geometric interpretation.** The SVD decomposes any linear map into three steps:

1. $V^T$: rotate the input space to align with the principal axes of $A$
2. $\Sigma$: scale along each axis by the singular values (and possibly change dimension)
3. $U$: rotate the output space to the final orientation

A unit sphere in $\mathbb{R}^n$ is mapped by $A$ to a hyperellipsoid in $\mathbb{R}^m$ whose semi-axes have lengths $\sigma_1, \ldots, \sigma_r$ in directions $u_1, \ldots, u_r$.

**Eckart-Young Theorem.** The best rank-$k$ approximation to $A$ in both the Frobenius norm and the spectral (2-) norm is:

$$A_k = \sum_{i=1}^{k} \sigma_i u_i v_i^T$$

$$\|A - A_k\|_F = \sqrt{\sigma_{k+1}^2 + \cdots + \sigma_r^2}, \quad \|A - A_k\|_2 = \sigma_{k+1}$$

### Worked Example

```
Let A = [3 2; 2 3; 2 -2].

A^T A = [3 2 2]   [3 2]     [17  4]
        [2 3 -2] * [2 3]  =  [4  17]
                   [2 -2]

Eigenvalues of A^T A:
  det(A^T A - lambda*I) = (17-L)^2 - 16 = L^2 - 34L + 273 = (L-21)(L-13)
  lambda_1 = 21,  lambda_2 = 13
  sigma_1 = sqrt(21),  sigma_2 = sqrt(13)

Eigenvectors:
  lambda=21: (A^TA - 21I)v = 0 => [-4 4; 4 -4]v = 0 => v1 = [1/sqrt(2), 1/sqrt(2)]^T
  lambda=13: v2 = [1/sqrt(2), -1/sqrt(2)]^T

Left singular vectors:
  u1 = Av1/sigma1 = [5/(sqrt(2)*sqrt(21)), 5/(sqrt(2)*sqrt(21)), 0]^T
     = [5/sqrt(42), 5/sqrt(42), 0]^T
  u2 = Av2/sigma2 = [1/(sqrt(2)*sqrt(13)), -1/(sqrt(2)*sqrt(13)), 4/(sqrt(2)*sqrt(13))]^T
     = [1/sqrt(26), -1/sqrt(26), 4/sqrt(26)]^T

Verification: A = U * diag(sqrt(21), sqrt(13)) * V^T
```

---

## 2. PageRank as Power Iteration on a Stochastic Matrix

### The Problem

Show that PageRank is the stationary distribution of a Markov chain, and prove that power iteration converges to it.

### The Formula

Model the web as a directed graph $G = (V, E)$ with $N = |V|$ pages. Define the hyperlink matrix $H \in \mathbb{R}^{N \times N}$:

$$H_{ij} = \begin{cases} 1/L(j) & \text{if page } j \text{ links to page } i \\ 0 & \text{otherwise} \end{cases}$$

where $L(j)$ is the out-degree of page $j$. Dangling nodes (pages with no outgoing links) create zero columns, which break stochasticity. Replace each zero column with $\frac{1}{N} \mathbf{e}$, yielding the stochastic matrix $S$.

The Google matrix incorporates the damping factor $d \in (0,1)$:

$$G = dS + \frac{1-d}{N} \mathbf{e}\mathbf{e}^T$$

$G$ is column-stochastic (columns sum to 1), positive (all entries $> 0$), and therefore primitive.

**Perron-Frobenius Theorem.** A positive stochastic matrix $G$ has:
1. A unique eigenvalue $\lambda_1 = 1$ (the largest in magnitude)
2. All other eigenvalues satisfy $|\lambda_i| < 1$
3. The eigenvector $\pi$ for $\lambda_1 = 1$ has all positive entries

The PageRank vector $\pi$ is this unique Perron eigenvector, normalized so $\|\pi\|_1 = 1$.

**Power iteration.** Starting from any probability vector $\pi^{(0)}$ (typically $\pi^{(0)} = \frac{1}{N}\mathbf{e}$):

$$\pi^{(t+1)} = G \pi^{(t)}$$

**Convergence proof.** Expand $\pi^{(0)}$ in the eigenbasis of $G$:

$$\pi^{(0)} = c_1 \pi + c_2 v_2 + \cdots + c_N v_N$$

where $Gv_i = \lambda_i v_i$. Then:

$$\pi^{(t)} = G^t \pi^{(0)} = c_1 \pi + c_2 \lambda_2^t v_2 + \cdots + c_N \lambda_N^t v_N$$

Since $|\lambda_i| \le d < 1$ for $i \ge 2$ (the subdominant eigenvalues of $G$ are bounded by $d$), we have $\lambda_i^t \to 0$ as $t \to \infty$. Since $\pi^{(0)}$ is a probability vector and $\pi$ is the unique stationary distribution, $c_1 = 1$, so:

$$\pi^{(t)} \to \pi \quad \text{as } t \to \infty$$

The convergence rate is $O(d^t)$, so with $d = 0.85$, we reach precision $\epsilon$ in $O(\log(1/\epsilon) / \log(1/d))$ iterations.

For $\epsilon = 10^{-8}$: $t \approx \frac{8 \ln 10}{\ln(1/0.85)} \approx 113$ iterations.

### Worked Example

```
Three-page web:
  Page 1 -> Page 2, Page 3
  Page 2 -> Page 1
  Page 3 -> Page 1, Page 2

Hyperlink matrix H:
       from 1   from 2   from 3
to 1 [  0       1       1/2  ]
to 2 [  1/2     0       1/2  ]
to 3 [  1/2     0        0   ]

Google matrix G (d = 0.85):
  G = 0.85 * H + 0.05 * ones(3,3)

  G = [0.05   0.90   0.475]
      [0.475  0.05   0.475]
      [0.475  0.05   0.05 ]

Power iteration (starting at [1/3, 1/3, 1/3]^T):
  t=0:  [0.3333, 0.3333, 0.3333]
  t=1:  [0.4750, 0.3333, 0.1917]
  t=5:  [0.4509, 0.3338, 0.2153]
  t=20: [0.4494, 0.3343, 0.2163]  (converged)

Page 1 has the highest PageRank (most inbound links).
```

---

## 3. PCA: Derivation from Variance Maximization

### The Problem

Derive PCA by finding the direction that maximizes the variance of the projected data.

### The Formula

Let $X \in \mathbb{R}^{n \times d}$ be a centered data matrix ($n$ samples, $d$ features, each column has zero mean). The covariance matrix is:

$$C = \frac{1}{n-1} X^T X$$

We seek a unit vector $w \in \mathbb{R}^d$ ($\|w\| = 1$) that maximizes the variance of the projection $Xw$:

$$\text{Var}(Xw) = \frac{1}{n-1} \|Xw\|^2 = \frac{1}{n-1} w^T X^T X w = w^T C w$$

This is a constrained optimization problem:

$$\max_{w} \; w^T C w \quad \text{subject to} \quad w^T w = 1$$

Introducing a Lagrange multiplier $\lambda$:

$$\mathcal{L}(w, \lambda) = w^T C w - \lambda(w^T w - 1)$$

Setting $\nabla_w \mathcal{L} = 0$:

$$2Cw - 2\lambda w = 0 \implies Cw = \lambda w$$

Therefore $w$ must be an eigenvector of $C$, and the variance along $w$ is:

$$w^T C w = w^T \lambda w = \lambda$$

The variance is maximized by choosing $w$ as the eigenvector corresponding to the largest eigenvalue $\lambda_1$. This is the **first principal component**.

**Subsequent components.** The $k$-th principal component is the eigenvector corresponding to the $k$-th largest eigenvalue, subject to orthogonality with all previous components. This follows from the same optimization on the orthogonal complement.

**Total variance preserved by $k$ components:**

$$\frac{\sum_{i=1}^{k} \lambda_i}{\sum_{i=1}^{d} \lambda_i}$$

**Connection to SVD.** If $X = U\Sigma V^T$, then:

$$C = \frac{1}{n-1} X^T X = \frac{1}{n-1} V \Sigma^2 V^T$$

The principal components are the columns of $V$, and $\lambda_i = \sigma_i^2 / (n-1)$. In practice, PCA is computed via the SVD of $X$ rather than eigendecomposition of $C$, because it is more numerically stable and avoids forming $X^T X$ explicitly.

### Worked Example

```
Data (centered):
  X = [-1  -1]
      [-1   0]
      [ 0   1]
      [ 2   0]

Covariance:
  C = (1/3) * X^T X = (1/3) * [6  -1]  = [2     -1/3]
                               [-1  2]    [-1/3   2/3]

Eigenvalues: det(C - lambda*I) = 0
  (2-L)(2/3-L) - 1/9 = 0
  L^2 - 8L/3 + 11/9 = 0
  lambda_1 = (8 + sqrt(64-44))/6 = (8+sqrt(20))/6 ~ 2.079
  lambda_2 = (8 - sqrt(20))/6 ~ 0.588

Variance captured by PC1: 2.079 / 2.667 = 78.0%
Both components: 100%

PC1 eigenvector: direction of maximum variance
PC2 eigenvector: orthogonal, captures remaining variance
```

---

## 4. Least Squares: Deriving the Normal Equations

### The Problem

Derive the normal equations for the ordinary least squares problem from calculus, and give the geometric interpretation.

### The Formula

Given an overdetermined system $Ax = b$ where $A \in \mathbb{R}^{m \times n}$ ($m > n$) and $\text{rank}(A) = n$, we seek $\hat{x}$ minimizing the squared residual:

$$f(x) = \|Ax - b\|^2 = (Ax - b)^T(Ax - b) = x^T A^T A x - 2b^T A x + b^T b$$

**Calculus derivation.** Set the gradient to zero:

$$\nabla_x f = 2A^T A x - 2A^T b = 0$$

$$\boxed{A^T A \hat{x} = A^T b}$$

These are the **normal equations**. Since $\text{rank}(A) = n$, the matrix $A^T A$ is invertible, and:

$$\hat{x} = (A^T A)^{-1} A^T b$$

The matrix $A^+ = (A^T A)^{-1} A^T$ is the **Moore-Penrose pseudoinverse** of $A$.

**Geometric derivation.** The residual $r = b - A\hat{x}$ must be orthogonal to the column space $\mathcal{C}(A)$:

$$A^T(b - A\hat{x}) = 0 \implies A^T A \hat{x} = A^T b$$

This says: $A\hat{x}$ is the orthogonal projection of $b$ onto $\mathcal{C}(A)$.

**Second-order condition.** The Hessian is $\nabla^2 f = 2A^T A$, which is positive definite when $A$ has full column rank, confirming this is a minimum.

### Worked Example

```
Fit y = c0 + c1*x to data: (1,2), (2,3), (3,6)

A = [1 1]    b = [2]
    [1 2]        [3]
    [1 3]        [6]

A^T A = [3  6]    A^T b = [11]
        [6 14]            [22]

Solve: [3 6; 6 14] * [c0; c1] = [11; 22]
  3c0 + 6c1 = 11
  6c0 + 14c1 = 22

  c1 = (22 - 2*11) / (14 - 2*6) = 0/2 = 0... let me redo:
  From row 1: c0 = (11 - 6c1)/3
  Row 2: 6*(11-6c1)/3 + 14c1 = 22 => 22 - 12c1 + 14c1 = 22 => 2c1 = 0...

  Actually: 6c0 + 14c1 = 22, and 2*(3c0 + 6c1) = 2*11 = 22
  So 6c0 + 12c1 = 22, subtract: 2c1 = 0 => c1 = 0?

  Recompute A^T b:
  A^T b = [1 1 1; 1 2 3] * [2;3;6] = [11; 20]

  3c0 + 6c1 = 11
  6c0 + 14c1 = 20

  Multiply first by 2: 6c0 + 12c1 = 22
  Subtract: 2c1 = -2 => c1 = -1?

  Recompute: A^T b = [1*2+1*3+1*6, 1*2+2*3+3*6] = [11, 26]

  3c0 + 6c1 = 11
  6c0 + 14c1 = 26

  Multiply first by 2: 6c0 + 12c1 = 22
  Subtract second: -2c1 = -4 => c1 = 2
  c0 = (11 - 12)/3 = -1/3

Best fit line: y = -1/3 + 2x
Residuals: r = [2-5/3, 3-11/3, 6-17/3] = [1/3, -2/3, 1/3]
Check: A^T r = [1 1 1; 1 2 3] * [1/3;-2/3;1/3] = [0, 0]  (orthogonal to columns)
```

---

## 5. Condition Number and Numerical Stability

### The Problem

Quantify how sensitive the solution of $Ax = b$ is to perturbations in $A$ and $b$, and establish practical guidelines for numerical computation.

### The Formula

Consider the perturbed system $(A + \delta A)(x + \delta x) = b + \delta b$. The **condition number** of $A$ in the 2-norm is:

$$\kappa(A) = \|A\|_2 \cdot \|A^{-1}\|_2 = \frac{\sigma_{\max}(A)}{\sigma_{\min}(A)}$$

**Perturbation bound (right-hand side only).** If $A(x + \delta x) = b + \delta b$:

$$\frac{\|\delta x\|}{\|x\|} \le \kappa(A) \frac{\|\delta b\|}{\|b\|}$$

**Perturbation bound (matrix only).** If $(A + \delta A)(x + \delta x) = b$:

$$\frac{\|\delta x\|}{\|x + \delta x\|} \le \kappa(A) \frac{\|\delta A\|}{\|A\|}$$

**Combined bound.** For perturbations in both $A$ and $b$:

$$\frac{\|\delta x\|}{\|x\|} \le \frac{\kappa(A)}{1 - \kappa(A) \frac{\|\delta A\|}{\|A\|}} \left( \frac{\|\delta A\|}{\|A\|} + \frac{\|\delta b\|}{\|b\|} \right)$$

**Floating-point arithmetic.** In IEEE 754 double precision, the unit roundoff is $u = 2^{-53} \approx 1.1 \times 10^{-16}$. Gaussian elimination with partial pivoting produces a computed solution $\hat{x}$ satisfying:

$$\frac{\|x - \hat{x}\|}{\|x\|} = O(\kappa(A) \cdot u)$$

Therefore, if $\kappa(A) \approx 10^k$, we lose roughly $k$ decimal digits of accuracy. A matrix with $\kappa(A) > 1/u \approx 10^{16}$ is effectively singular in double precision.

**Backward stability.** An algorithm is backward stable if the computed solution $\hat{x}$ is the exact solution of a nearby problem $(A + \Delta A)\hat{x} = b + \Delta b$ with $\|\Delta A\|/\|A\| = O(u)$ and $\|\Delta b\|/\|b\| = O(u)$. Gaussian elimination with partial pivoting, Householder QR, and the standard SVD algorithm are all backward stable.

### Worked Example

```
A = [1.0000  1.0001]     b = [2.0001]
    [1.0001  1.0002]         [2.0003]

Exact solution: x = [1, 1]^T

Singular values: sigma_1 ~ 2.0001, sigma_2 ~ 5e-5
Condition number: kappa(A) ~ 40000

Perturb b by delta_b = [0, 0.0001]:
  New solution: x' ~ [1 + 2, 1 - 2] = [3, -1]  (wildly different!)

Relative perturbation in b: ||delta_b||/||b|| ~ 0.00005
Relative change in x: ||delta_x||/||x|| ~ 2.0

Amplification factor: 2.0 / 0.00005 = 40000 = kappa(A)

Lesson: always check kappa(A) before trusting a computed solution.
```

---

## 6. Strassen's Matrix Multiplication

### The Problem

Multiply two $n \times n$ matrices faster than the naive $O(n^3)$ algorithm.

### The Formula

Strassen's insight (1969): multiply two $2 \times 2$ matrices using 7 multiplications instead of 8.

Partition $A$ and $B$ into $2 \times 2$ blocks:

$$A = \begin{bmatrix} A_{11} & A_{12} \\ A_{21} & A_{22} \end{bmatrix}, \quad B = \begin{bmatrix} B_{11} & B_{12} \\ B_{21} & B_{22} \end{bmatrix}$$

Define 7 intermediate products:

$$M_1 = (A_{11} + A_{22})(B_{11} + B_{22})$$
$$M_2 = (A_{21} + A_{22})B_{11}$$
$$M_3 = A_{11}(B_{12} - B_{22})$$
$$M_4 = A_{22}(B_{21} - B_{11})$$
$$M_5 = (A_{11} + A_{12})B_{22}$$
$$M_6 = (A_{21} - A_{11})(B_{11} + B_{12})$$
$$M_7 = (A_{12} - A_{22})(B_{21} + B_{22})$$

Then:

$$C_{11} = M_1 + M_4 - M_5 + M_7$$
$$C_{12} = M_3 + M_5$$
$$C_{21} = M_2 + M_4$$
$$C_{22} = M_1 - M_2 + M_3 + M_6$$

**Recurrence:** $T(n) = 7T(n/2) + O(n^2)$

By the Master theorem: $T(n) = O(n^{\log_2 7}) = O(n^{2.807})$.

**Practical considerations:**
- Strassen is faster for large $n$ (typically $n > 64$--$128$ in practice)
- Uses more additions: 18 additions vs. 4 additions for naive (but additions are cheaper than multiplications)
- Numerically less stable than naive multiplication
- Modern libraries (BLAS) use Strassen at the top levels, switching to highly optimized naive kernels for small blocks
- The theoretical lower bound for matrix multiplication is conjectured to be $O(n^2)$ (the exponent $\omega = 2$ conjecture), but no practical algorithm achieving $O(n^{2+\epsilon})$ for small $\epsilon$ is known

### Worked Example

```
A = [1 3; 7 5],  B = [6 8; 4 2]

M1 = (1+5)(6+2) = 6*8 = 48
M2 = (7+5)*6 = 72
M3 = 1*(8-2) = 6
M4 = 5*(4-6) = -10
M5 = (1+3)*2 = 8
M6 = (7-1)*(6+8) = 84
M7 = (3-5)*(4+2) = -12

C11 = 48 + (-10) - 8 + (-12) = 18
C12 = 6 + 8 = 14
C21 = 72 + (-10) = 62
C22 = 48 - 72 + 6 + 84 = 66

C = [18 14; 62 66]

Verify: [1 3; 7 5] * [6 8; 4 2] = [6+12 8+6; 42+20 56+10] = [18 14; 62 66]  (correct)
```

---

## 7. Randomized Algorithms for SVD and the Johnson-Lindenstrauss Lemma

### The Problem

Compute approximate SVDs of very large matrices efficiently, and understand the theoretical foundation for random dimensionality reduction.

### The Formula

**Randomized SVD (Halko, Martinsson, Tropp, 2011).** To find a rank-$k$ approximation to $A \in \mathbb{R}^{m \times n}$:

1. Draw a random Gaussian matrix $\Omega \in \mathbb{R}^{n \times (k+p)}$ where $p$ is a small oversampling parameter (e.g., $p = 5$--$10$)
2. Form the sample matrix $Y = A\Omega \in \mathbb{R}^{m \times (k+p)}$
3. Compute a QR factorization $Y = QR$
4. Form $B = Q^T A \in \mathbb{R}^{(k+p) \times n}$
5. Compute the SVD of the small matrix $B = \hat{U}\Sigma V^T$
6. Set $U = Q\hat{U}$

The result $A \approx U\Sigma V^T$ is a near-optimal rank-$k$ approximation. The total cost is $O(mn(k+p))$ for forming $Y$, plus $O(m(k+p)^2)$ for QR, plus $O((k+p)^2 n)$ for the small SVD -- dramatically cheaper than the full SVD cost of $O(mn \min(m,n))$ when $k \ll \min(m,n)$.

**Error bound.** With probability at least $1 - 6p^{-p}$:

$$\|A - QQ^T A\|_2 \le \left(1 + \frac{4\sqrt{k+p}}{p-1} \cdot \sqrt{\min(m,n)}\right) \sigma_{k+1}$$

In practice, the error is typically within a small constant factor of $\sigma_{k+1}$.

**Johnson-Lindenstrauss Lemma (1984).** For any $\epsilon \in (0, 1)$ and any set $P$ of $n$ points in $\mathbb{R}^d$, there exists a map $f: \mathbb{R}^d \to \mathbb{R}^k$ with $k = O(\log n / \epsilon^2)$ such that for all $u, v \in P$:

$$(1 - \epsilon)\|u - v\|^2 \le \|f(u) - f(v)\|^2 \le (1 + \epsilon)\|u - v\|^2$$

**Construction.** The map $f(x) = \frac{1}{\sqrt{k}} Rx$ where $R \in \mathbb{R}^{k \times d}$ has i.i.d. entries from $\mathcal{N}(0, 1)$ satisfies the lemma with high probability.

**Proof sketch.** For a fixed pair $(u, v)$, the squared norm $\|f(u) - f(v)\|^2 = \frac{1}{k}\|R(u-v)\|^2$ is a sum of $k$ i.i.d. $\chi^2$ random variables, scaled by $\|u-v\|^2/k$. Concentration inequalities (sub-exponential tail bounds) show that this sum deviates from its mean $\|u-v\|^2$ by more than $\epsilon\|u-v\|^2$ with probability at most $2\exp(-c\epsilon^2 k)$. A union bound over all $\binom{n}{2}$ pairs gives the result when $k = O(\log n / \epsilon^2)$.

The JL lemma is the theoretical backbone of random projection methods in ML, approximate nearest neighbor search, and streaming algorithms.

### Worked Example

```
Scenario: 10 million documents, 100,000-dimensional TF-IDF vectors.
Goal: reduce to k dimensions while preserving pairwise distances within 10%.

n = 10^7, epsilon = 0.1
k = O(log(10^7) / 0.01) = O(16.1 / 0.01) ~ 1610 dimensions

Projection: multiply each 100,000-dim vector by a 1610 x 100,000 random matrix.

Storage: original = 10^7 * 10^5 = 10^12 entries (if dense)
         projected = 10^7 * 1610 ~ 1.6 * 10^10 entries

Speedup for pairwise distance: 100000/1610 ~ 62x faster
All pairwise distances preserved within +/- 10%.
```

---

## 8. Tensors and Multilinear Algebra

### The Problem

Extend matrix concepts to higher-order arrays (tensors), which arise in physics, data science, and deep learning.

### The Formula

A **tensor** of order $p$ (also called a $p$-way array or $p$-dimensional array) over $\mathbb{R}$ is an element of $\mathbb{R}^{n_1 \times n_2 \times \cdots \times n_p}$.

| Order | Object | Example |
|-------|--------|---------|
| 0 | Scalar | Temperature at a point |
| 1 | Vector | RGB color: $\mathbb{R}^3$ |
| 2 | Matrix | Grayscale image: $\mathbb{R}^{m \times n}$ |
| 3 | 3-tensor | Color image: $\mathbb{R}^{m \times n \times 3}$ |
| 4 | 4-tensor | Batch of color images: $\mathbb{R}^{b \times m \times n \times 3}$ |

**Tensor decompositions** generalize matrix decompositions:

**CP decomposition (CANDECOMP/PARAFAC).** A rank-$R$ CP decomposition of a 3-tensor $\mathcal{X} \in \mathbb{R}^{I \times J \times K}$ is:

$$\mathcal{X} \approx \sum_{r=1}^{R} a_r \circ b_r \circ c_r$$

where $\circ$ denotes the outer product, $a_r \in \mathbb{R}^I$, $b_r \in \mathbb{R}^J$, $c_r \in \mathbb{R}^K$.

**Tucker decomposition.** A Tucker decomposition is:

$$\mathcal{X} \approx \mathcal{G} \times_1 A \times_2 B \times_3 C$$

where $\mathcal{G} \in \mathbb{R}^{R_1 \times R_2 \times R_3}$ is the core tensor and $A \in \mathbb{R}^{I \times R_1}$, $B \in \mathbb{R}^{J \times R_2}$, $C \in \mathbb{R}^{K \times R_3}$ are factor matrices. The $\times_n$ denotes the $n$-mode product.

**Key differences from matrices:**
- Tensor rank is NP-hard to compute (unlike matrix rank)
- Best rank-$R$ approximation may not exist (unlike Eckart-Young for matrices)
- No unique "SVD" for tensors of order $\ge 3$, but HOSVD (Higher-Order SVD) provides an orthogonal Tucker decomposition

**Applications in deep learning:**
- Weight tensors in convolutional layers: $\mathbb{R}^{C_{\text{out}} \times C_{\text{in}} \times k_h \times k_w}$
- Tensor decomposition for model compression: replace a large weight tensor with low-rank factors
- Attention mechanisms in transformers involve tensor contractions
- Einstein summation notation (`einsum`) is the lingua franca for tensor operations in modern ML frameworks

### Worked Example

```
3-tensor X of shape 3 x 3 x 2 (e.g., 3x3 color image with 2 channels):

X[:,:,0] = [1 2 3]    X[:,:,1] = [2 4 6]
            [4 5 6]               [8 10 12]
            [7 8 9]               [14 16 18]

Observation: X[:,:,1] = 2 * X[:,:,0]

This tensor has CP rank 1 if X[:,:,0] itself has rank 1.
X[:,:,0] does not have rank 1 (it's rank 2), so the full tensor has CP rank 2:

X = sigma1 * (a1 o b1 o [1,2]) + sigma2 * (a2 o b2 o [1,2])

where a1, b1, a2, b2 come from the rank-2 decomposition of X[:,:,0].
The [1,2] factor captures the channel relationship.
```

---

## Prerequisites

- Calculus (partial derivatives, gradients, Lagrange multipliers)
- Basic linear algebra (matrix operations, determinants, systems of equations)
- Proof techniques (induction, contradiction)
- Probability basics (for randomized algorithms and PCA)
- Algorithm analysis and asymptotic notation

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Perform matrix operations (multiply, transpose, invert 2x2). Solve small systems by Gaussian elimination. Compute eigenvalues of 2x2 matrices. Understand geometric meaning of linear transformations. State the rank-nullity theorem. |
| **Intermediate** | Derive the normal equations for least squares. Compute SVD of small matrices by hand. Explain PCA as variance maximization. Implement power iteration for PageRank. Analyze condition number and its effect on solution accuracy. Apply Gram-Schmidt orthogonalization. |
| **Advanced** | Prove convergence of power iteration via spectral analysis. Derive error bounds for randomized SVD. Analyze Strassen's algorithm via the Master theorem. Connect the Johnson-Lindenstrauss lemma to concentration inequalities. Understand tensor rank complexity and decomposition uniqueness conditions. Relate the Perron-Frobenius theorem to Markov chain convergence. |
