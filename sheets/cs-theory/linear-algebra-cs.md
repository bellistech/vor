# Linear Algebra for Computer Science (Vectors, Matrices, and Applications)

A complete reference for vectors, matrices, linear transformations, decompositions, eigenvalues, SVD, PCA, PageRank, least squares, and numerical stability — the mathematical engine behind computer graphics, machine learning, and scientific computing.

## Vectors and Matrices

### Vectors

```
A vector v in R^n is an ordered n-tuple of real numbers:

  v = [v1, v2, ..., vn]^T

Operations:
  Addition:        u + v = [u1+v1, u2+v2, ..., un+vn]^T
  Scalar multiply: c * v = [c*v1, c*v2, ..., c*vn]^T
  Dot product:     u . v = u1*v1 + u2*v2 + ... + un*vn
  Norm (L2):       ||v|| = sqrt(v . v)
  Cross product:   u x v (only in R^3)

Properties:
  u . v = ||u|| * ||v|| * cos(theta)    angle between vectors
  u . v = 0  <==>  u perpendicular to v (orthogonal)
```

### Matrix Operations

```
Matrix A is an m x n array of real numbers (m rows, n columns)

Addition:       (A + B)_ij = a_ij + b_ij          (same dimensions)
Scalar mult:    (cA)_ij = c * a_ij
Multiplication: (AB)_ij = sum_k a_ik * b_kj       A is m x p, B is p x n => AB is m x n
                AB != BA in general                (not commutative)

Complexity of naive matrix multiply: O(n^3)
Strassen's algorithm: O(n^2.807)
Best known (2024): O(n^2.371552)
```

### Transpose

```
(A^T)_ij = a_ji                  flip rows and columns

Properties:
  (A^T)^T = A
  (A + B)^T = A^T + B^T
  (AB)^T = B^T * A^T              reverse order
  (cA)^T = c * A^T
```

### Inverse

```
A^(-1) exists iff det(A) != 0     (A is "nonsingular" or "invertible")

A * A^(-1) = A^(-1) * A = I       identity matrix

Properties:
  (A^(-1))^(-1) = A
  (AB)^(-1) = B^(-1) * A^(-1)     reverse order
  (A^T)^(-1) = (A^(-1))^T

2x2 inverse:
  A = [a b; c d]
  A^(-1) = (1/det(A)) * [d -b; -c a]
  det(A) = ad - bc
```

## Determinants

```
det(A) for n x n matrix:

2x2: det([a b; c d]) = ad - bc

3x3: cofactor expansion along first row
  det(A) = a11*C11 + a12*C12 + a13*C13
  where Cij = (-1)^(i+j) * det(Mij)   (Mij is minor matrix)

Properties:
  det(AB) = det(A) * det(B)
  det(A^T) = det(A)
  det(cA) = c^n * det(A)              for n x n matrix
  det(A^(-1)) = 1/det(A)
  Row swap: changes sign of determinant
  Row of zeros: det = 0
  Two identical rows: det = 0
  det(A) != 0  <==>  A is invertible
```

## Systems of Linear Equations

### Gaussian Elimination

```
Solve Ax = b by row reduction to echelon form:

  [A | b] --> row operations --> [U | c]

Row operations (preserve solution set):
  1. Swap two rows
  2. Multiply a row by nonzero scalar
  3. Add a multiple of one row to another

Back substitution from upper triangular U:
  x_n = c_n / u_nn
  x_i = (c_i - sum_{j>i} u_ij * x_j) / u_ii

Complexity: O(n^3) for n x n system
```

### LU Decomposition

```
Factor A = LU where L is lower triangular, U is upper triangular

  [a11 a12 a13]   [1   0  0] [u11 u12 u13]
  [a21 a22 a23] = [l21 1  0] [0   u22 u23]
  [a31 a32 a33]   [l31 l32 1] [0   0   u33]

Solving Ax = b:
  1. Factor A = LU                     O(n^3)
  2. Solve Ly = b  (forward sub)       O(n^2)
  3. Solve Ux = y  (back sub)          O(n^2)

With partial pivoting: PA = LU  (P is permutation matrix)
Advantage: factor once, solve for many right-hand sides b
```

## Vector Spaces

### Basis, Dimension, Rank

```
Vector space V over R:
  Closed under addition and scalar multiplication
  Contains zero vector

Subspace: subset of V that is itself a vector space

Span({v1, ..., vk}) = all linear combinations c1*v1 + ... + ck*vk

Linear independence: c1*v1 + ... + ck*vk = 0 implies all ci = 0

Basis: linearly independent set that spans V
Dimension: number of vectors in any basis

Column space:  C(A) = span of columns of A
Row space:     R(A) = span of rows of A
Null space:    N(A) = {x : Ax = 0}

Rank-Nullity Theorem:
  rank(A) + nullity(A) = n       (n = number of columns)
  rank(A) = dim(C(A)) = dim(R(A))
  nullity(A) = dim(N(A))

rank(A) = rank(A^T)
rank(AB) <= min(rank(A), rank(B))
```

### Orthogonality

```
Orthogonal set: all pairs have dot product 0
Orthonormal set: orthogonal and all vectors have unit norm

Gram-Schmidt process: produce orthonormal basis from any basis
  u1 = v1 / ||v1||
  u2 = (v2 - (v2.u1)*u1) / ||v2 - (v2.u1)*u1||
  ...

Orthogonal matrix Q: Q^T * Q = I, so Q^(-1) = Q^T
  Preserves lengths and angles (rotation/reflection)

Orthogonal projection of v onto subspace W:
  proj_W(v) = sum_i (v . ui) * ui    where {ui} is orthonormal basis of W
```

## Linear Transformations

```
T: R^n -> R^m is linear if:
  T(u + v) = T(u) + T(v)
  T(cv) = c * T(v)

Every linear transformation has a matrix representation:
  T(x) = Ax    where A is m x n

Common transformations in R^2 and R^3:

Rotation by angle theta (2D):
  R = [cos(theta)  -sin(theta)]
      [sin(theta)   cos(theta)]

Scaling:
  S = [sx  0 ]
      [0   sy]

Reflection across x-axis:
  F = [1   0]
      [0  -1]

Shear:
  H = [1  k]
      [0  1]

3D rotation about z-axis:
  Rz = [cos(theta) -sin(theta) 0]
       [sin(theta)  cos(theta) 0]
       [0           0          1]

Homogeneous coordinates (translation as matrix multiply):
  [x']   [1 0 tx] [x]
  [y'] = [0 1 ty] [y]
  [1 ]   [0 0  1] [1]
```

## Eigenvalues and Eigenvectors

### Characteristic Polynomial

```
Av = lambda * v    (v != 0)

lambda = eigenvalue, v = eigenvector

Characteristic polynomial: det(A - lambda*I) = 0

2x2 example:
  A = [4 1; 2 3]
  det([4-L 1; 2 3-L]) = (4-L)(3-L) - 2 = L^2 - 7L + 10 = (L-5)(L-2) = 0
  Eigenvalues: lambda = 5, lambda = 2

Properties:
  sum of eigenvalues = trace(A) = sum of diagonal entries
  product of eigenvalues = det(A)
  Real symmetric matrix: all eigenvalues are real
  Eigenvalues of A^k = (eigenvalues of A)^k
```

### Diagonalization

```
A = P * D * P^(-1)

D = diagonal matrix of eigenvalues
P = matrix of eigenvectors as columns

A is diagonalizable iff it has n linearly independent eigenvectors
Real symmetric matrices are always diagonalizable (Spectral Theorem)

Benefit: A^k = P * D^k * P^(-1)   (fast matrix powers)
```

## Singular Value Decomposition (SVD)

```
Any m x n matrix A can be factored as:

  A = U * Sigma * V^T

  U:     m x m orthogonal (left singular vectors)
  Sigma: m x n diagonal (singular values sigma_1 >= sigma_2 >= ... >= 0)
  V:     n x n orthogonal (right singular vectors)

Singular values: sigma_i = sqrt(eigenvalue_i of A^T*A)

Truncated SVD (rank-k approximation):
  A_k = sum_{i=1}^{k} sigma_i * u_i * v_i^T

Eckart-Young Theorem: A_k is the best rank-k approximation
  minimizing ||A - B||_F over all rank-k matrices B

Applications: image compression, noise reduction, dimensionality reduction,
  pseudoinverse, least squares, latent semantic analysis
```

## Principal Component Analysis (PCA)

```
Given data matrix X (n samples x d features), centered (mean subtracted):

  1. Compute covariance matrix: C = (1/(n-1)) * X^T * X
  2. Find eigenvalues and eigenvectors of C (or use SVD of X)
  3. Sort eigenvectors by eigenvalue magnitude (descending)
  4. Project: Z = X * W_k    where W_k = top k eigenvectors

Variance captured by component i: lambda_i / sum(all lambdas)

Choosing k:
  - Keep enough components to capture 90-95% of total variance
  - Scree plot: look for "elbow"

Connection to SVD: if X = U * Sigma * V^T, then
  Principal components = columns of V
  Projected data = U * Sigma (or X * V)
```

## PageRank Algorithm

```
Web as directed graph: pages = nodes, hyperlinks = edges

PageRank models a random surfer:
  - With probability d, follow a random outgoing link
  - With probability (1-d), jump to a random page
  d = damping factor (typically 0.85)

Transition matrix M:
  M_ij = 1/L(j) if page j links to page i, else 0
  L(j) = number of outgoing links from j

PageRank vector r (stationary distribution):
  r = d * M * r + (1-d)/N * e     (e = vector of all ones)

Rewritten as eigenvector problem:
  r = A * r     where A = d*M + (1-d)/N * ee^T

Power iteration:
  r(0) = (1/N) * e                initial uniform distribution
  r(t+1) = A * r(t)               iterate until convergence
  Converges because A is stochastic with spectral gap

Convergence rate: O(d^t)  =>  typically 50-100 iterations suffice
```

## Least Squares

### Normal Equations

```
Overdetermined system Ax = b (m > n, no exact solution):

Minimize ||Ax - b||^2

Solution via normal equations:
  A^T * A * x = A^T * b
  x = (A^T * A)^(-1) * A^T * b    (if A^T*A is invertible)

A^+ = (A^T * A)^(-1) * A^T        pseudoinverse (Moore-Penrose)
```

### QR Factorization

```
A = QR    (Q orthogonal, R upper triangular)

Least squares via QR:
  Ax = b  =>  QRx = b  =>  Rx = Q^T * b    (solve by back sub)

More numerically stable than normal equations
Computed via Gram-Schmidt, Householder reflections, or Givens rotations
Householder is preferred in practice: O(2mn^2 - 2n^3/3)
```

## Matrix Decompositions Summary

```
Decomposition  | Form         | Conditions             | Cost     | Use Case
---------------|--------------|------------------------|----------|---------------------------
LU             | A = LU       | Square, nonsingular    | O(n^3)   | Solve linear systems
LU (pivoted)   | PA = LU      | Square                 | O(n^3)   | General linear systems
Cholesky       | A = LL^T     | Symmetric pos. def.    | O(n^3/3) | Fastest for SPD systems
QR             | A = QR       | Any m x n              | O(mn^2)  | Least squares
Eigendecomp    | A = PDP^(-1) | n indep. eigenvectors  | O(n^3)   | Matrix powers, stability
SVD            | A = USV^T    | Any m x n              | O(mn^2)  | Rank, approx, pseudoinverse
Schur          | A = QTQ^T    | Square                 | O(n^3)   | Eigenvalue computation
```

## Numerical Stability

```
Condition number: kappa(A) = ||A|| * ||A^(-1)||
  For 2-norm: kappa(A) = sigma_max / sigma_min

Interpretation:
  kappa(A) ~= 1:     well-conditioned (small errors stay small)
  kappa(A) >> 1:      ill-conditioned (small errors amplified)

Rule of thumb: lose log10(kappa(A)) digits of accuracy in solving Ax = b

Machine epsilon: eps ~= 2.2e-16 (64-bit double)
  Relative error in solution: ||delta_x||/||x|| <= kappa(A) * eps

Strategies for numerical stability:
  - Use pivoting in Gaussian elimination (partial pivoting standard)
  - Prefer QR over normal equations for least squares
  - Use Cholesky for symmetric positive definite systems
  - Iterative refinement: solve, compute residual, correct
  - Use condition number estimators before trusting results
```

## Sparse Matrices

```
Sparse: most entries are zero (typically nnz << n^2)

Storage formats:
  COO (coordinate): store (row, col, value) triples
  CSR (compressed sparse row): row_ptr, col_idx, values
  CSC (compressed sparse column): col_ptr, row_idx, values

COO: easy to construct, O(nnz) space
CSR: fast row access, efficient SpMV (sparse matrix-vector multiply)
CSC: fast column access, efficient for column operations

Sparse matrix-vector multiply: O(nnz) instead of O(n^2)

Common sparse solvers:
  Direct: sparse LU (SuperLU, UMFPACK)
  Iterative: conjugate gradient (SPD), GMRES (general)
  Preconditioners: incomplete LU/Cholesky, algebraic multigrid

Applications: FEM meshes, graph adjacency matrices, recommendation systems
```

## Applications in Computer Graphics

```
Model-View-Projection pipeline:

  v_screen = P * V * M * v_object

  M = model matrix (object -> world)
  V = view matrix (world -> camera)
  P = projection matrix (camera -> clip space)

Perspective projection (4x4 homogeneous):
  [2n/(r-l)   0       (r+l)/(r-l)   0        ]
  [0          2n/(t-b) (t+b)/(t-b)  0        ]
  [0          0       -(f+n)/(f-n)  -2fn/(f-n)]
  [0          0       -1             0        ]

Quaternion rotation (avoids gimbal lock):
  q = w + xi + yj + zk,   ||q|| = 1
  Rotation matrix from quaternion is 3x3 (convert as needed)

Skinning/animation: blend matrices per vertex
  v' = sum_i w_i * M_i * v    (weighted sum of bone transforms)
```

## Applications in Machine Learning

```
Gradient descent (matrix form):
  theta = theta - alpha * X^T * (X*theta - y)    (linear regression)

Covariance matrix:
  C = (1/(n-1)) * X^T * X     (centered data)
  Symmetric positive semi-definite

Neural network forward pass:
  z = W * x + b               (linear layer)
  a = sigma(z)                (activation)
  Weight matrix W: n_out x n_in

Backpropagation uses chain rule on Jacobian matrices:
  dL/dW = dL/da * da/dz * dz/dW

Dimensionality reduction:
  PCA: project onto top-k eigenvectors of covariance matrix
  Random projection: Johnson-Lindenstrauss lemma
    For eps in (0,1), project R^d -> R^k with k = O(log(n)/eps^2)
    Preserves pairwise distances within (1 +/- eps) factor
```

## Key Figures

```
Carl Friedrich Gauss (1777-1855)
  - Gaussian elimination (method of least squares)
  - Fundamental contributions to number theory, statistics, astronomy
  - The "Prince of Mathematicians"

Gilbert Strang (b. 1934)
  - Pioneered teaching of applied linear algebra
  - "Introduction to Linear Algebra" — standard CS/engineering textbook
  - MIT OpenCourseWare linear algebra lectures (millions of views)

Larry Page (b. 1973) and Sergey Brin (b. 1973)
  - PageRank algorithm (eigenvector of web link matrix)
  - Founders of Google
  - Transformed information retrieval via linear algebra

Gene Golub (1932-2007)
  - SVD algorithms and matrix computations
  - Co-author of "Matrix Computations" (the bible of numerical LA)
  - Founding editor of SIAM Journal on Scientific Computing

Charles Van Loan (b. 1946)
  - Co-author of "Matrix Computations" with Golub
  - Contributions to matrix exponential, Kronecker products
  - Structured matrix algorithms
```

## Tips

- Understand the four fundamental subspaces: C(A), N(A), C(A^T), N(A^T)
- For any system Ax = b: check rank(A) vs rank([A|b]) to determine solvability
- SVD is the Swiss Army knife: use it for rank, pseudoinverse, low-rank approximation, and condition number
- Prefer QR over normal equations for least squares (better numerical stability)
- Cholesky is 2x faster than LU for symmetric positive definite matrices
- Sparse matrix storage is essential when n > 10^4 and density < 1%
- Eigenvalues of symmetric matrices are always real — use this as a sanity check
- In ML: covariance matrices are always positive semi-definite by construction

## See Also

- `detail/cs-theory/linear-algebra-cs.md` — SVD derivation, PageRank power iteration proof, PCA from variance maximization, condition number analysis, Strassen's algorithm
- `sheets/cs-theory/graph-theory.md` — spectral graph theory uses eigenvalues of graph Laplacian
- `sheets/cs-theory/complexity-theory.md` — complexity of matrix multiplication, P vs NP
- `sheets/cs-theory/information-theory.md` — entropy and dimensionality reduction connections
- `sheets/cs-theory/number-theory-crypto.md` — modular arithmetic and matrices over finite fields

## References

- Strang, Gilbert. "Introduction to Linear Algebra" (Wellesley-Cambridge Press, 6th edition, 2023)
- Golub, Gene H. and Van Loan, Charles F. "Matrix Computations" (Johns Hopkins University Press, 4th edition, 2013)
- Trefethen, Lloyd N. and Bau, David. "Numerical Linear Algebra" (SIAM, 1997)
- Axler, Sheldon. "Linear Algebra Done Right" (Springer, 4th edition, 2024)
- Page, Lawrence et al. "The PageRank Citation Ranking: Bringing Order to the Web" (Stanford InfoLab, 1999)
