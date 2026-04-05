# The Mathematics of Gaussian Elimination -- Numerical Linear Algebra and Pivoting Theory

> *Gaussian elimination reduces a system of linear equations to triangular form through a sequence of elementary row operations -- each operation preserving the solution set while driving the matrix toward a shape where back substitution trivially extracts the answer.*

---

## 1. Elementary Row Operations and Solution Invariance (Linear Algebra)

### The Problem

Prove that row swaps, row scaling, and row addition preserve the solution set of a
linear system.

### The Formula

A system $Ax = b$ is represented by an augmented matrix $[A|b]$. The three elementary
row operations are:

1. **Swap:** $R_i \leftrightarrow R_j$
2. **Scale:** $R_i \gets c \cdot R_i$ where $c \ne 0$
3. **Add:** $R_i \gets R_i + c \cdot R_j$

Each operation corresponds to left-multiplication by an invertible elementary matrix $E_k$.
The transformed system $E_k A x = E_k b$ has the same solution set because $E_k$ is
invertible: $x$ satisfies $Ax = b$ if and only if $x$ satisfies $E_k Ax = E_k b$.

After $m$ operations: $E_m \cdots E_1 A x = E_m \cdots E_1 b$, yielding upper triangular
form $Ux = c$ where $U = E_m \cdots E_1 A$.

### Worked Examples

$$\begin{bmatrix} 2 & 1 & -1 & | & 8 \\ -3 & -1 & 2 & | & -11 \\ -2 & 1 & 2 & | & -3 \end{bmatrix}$$

$R_2 \gets R_2 + \frac{3}{2}R_1$: $\begin{bmatrix} 2 & 1 & -1 & | & 8 \\ 0 & 1/2 & 1/2 & | & 1 \\ -2 & 1 & 2 & | & -3 \end{bmatrix}$

$R_3 \gets R_3 + R_1$: $\begin{bmatrix} 2 & 1 & -1 & | & 8 \\ 0 & 1/2 & 1/2 & | & 1 \\ 0 & 2 & 1 & | & 5 \end{bmatrix}$

$R_3 \gets R_3 - 4R_2$: $\begin{bmatrix} 2 & 1 & -1 & | & 8 \\ 0 & 1/2 & 1/2 & | & 1 \\ 0 & 0 & -1 & | & 1 \end{bmatrix}$

---

## 2. Back Substitution (Algorithm Design)

### The Problem

Given an upper triangular system $Ux = c$, solve for $x$ in O(n^2) time.

### The Formula

For an upper triangular matrix $U$ with $u_{ii} \ne 0$:

$$x_i = \frac{c_i - \sum_{j=i+1}^{n} u_{ij} x_j}{u_{ii}}, \quad i = n, n-1, \ldots, 1$$

Starting from the last row (one variable), each subsequent row uses previously computed
values. The sum involves at most $n - i$ terms.

**Total operations:** $\sum_{i=1}^{n}(n - i) = n(n-1)/2 = O(n^2)$.

### Worked Examples

From the triangular system above:

$$x_3 = \frac{1}{-1} = -1$$

$$x_2 = \frac{1 - (1/2)(-1)}{1/2} = \frac{1.5}{0.5} = 3$$

$$x_1 = \frac{8 - (1)(3) - (-1)(-1)}{2} = \frac{8 - 3 - 1}{2} = 2$$

Solution: $x = [2, 3, -1]$.

---

## 3. Partial Pivoting and Numerical Stability (Numerical Analysis)

### The Problem

Why does naive Gaussian elimination fail on certain well-conditioned systems, and how
does partial pivoting fix it?

### The Formula

Without pivoting, the growth factor $g$ measures how much intermediate values exceed the
original matrix entries:

$$g = \frac{\max_{i,j,k} |a_{ij}^{(k)}|}{\max_{i,j} |a_{ij}^{(0)}|}$$

Without pivoting, $g$ can be as large as $2^{n-1}$. With partial pivoting (swapping the
row with the largest absolute value in the pivot column):

$$g \le 2^{n-1} \text{ (theoretical worst case, but } g = O(n) \text{ in practice)}$$

The **backward error** bound with partial pivoting is:

$$\|A\hat{x} - b\| \le g \cdot n \cdot \epsilon \cdot \|A\| \cdot \|\hat{x}\|$$

where $\epsilon \approx 2.2 \times 10^{-16}$ (double precision machine epsilon).

### Worked Examples

Without pivoting on:
$$\begin{bmatrix} 10^{-20} & 1 \\ 1 & 1 \end{bmatrix} \begin{bmatrix} x_1 \\ x_2 \end{bmatrix} = \begin{bmatrix} 1 \\ 2 \end{bmatrix}$$

Factor: $f = 1/10^{-20} = 10^{20}$. After elimination: $x_2 \approx 1$, but $x_1 = (1 - 1)/10^{-20} = 0$ (wrong -- should be $\approx 1$).

With partial pivoting (swap rows first):
$$\begin{bmatrix} 1 & 1 \\ 10^{-20} & 1 \end{bmatrix} \Rightarrow x_2 \approx 1, \quad x_1 = 2 - 1 = 1 \quad\text{(correct)}$$

---

## 4. LU Decomposition as Factored Elimination (Matrix Theory)

### The Problem

Show that Gaussian elimination implicitly computes a factorization $PA = LU$.

### The Formula

The sequence of elementary matrices from elimination gives:

$$E_m \cdots E_1 P A = U$$

where $P$ is the permutation matrix from row swaps. Therefore:

$$PA = (E_m \cdots E_1)^{-1} U = LU$$

$L$ is unit lower triangular (1s on diagonal), with entries $l_{ij} = $ the multiplier
used to eliminate $a_{ij}$ during step $j$:

$$l_{ij} = \frac{a_{ij}^{(j)}}{a_{jj}^{(j)}}, \quad i > j$$

**Benefit:** Once $PA = LU$ is computed (O(n^3)), solving for any new right-hand side
$b'$ requires only two O(n^2) triangular solves: $Ly = Pb'$ then $Ux = y$.

### Worked Examples

For the 3x3 example (no pivoting needed on the first column since 2 is already largest after
the pivot swap with row 2):

$$L = \begin{bmatrix} 1 & 0 & 0 \\ -3/2 & 1 & 0 \\ -1 & 4 & 1 \end{bmatrix}, \quad U = \begin{bmatrix} 2 & 1 & -1 \\ 0 & 1/2 & 1/2 \\ 0 & 0 & -1 \end{bmatrix}$$

$LU$ recovers the original (permuted) $A$.

---

## 5. Singularity and the Determinant (Abstract Algebra)

### The Problem

How does Gaussian elimination detect singular systems, and what is the connection to
the determinant?

### The Formula

After reduction to upper triangular form $U$:

$$\det(A) = (-1)^s \prod_{i=1}^{n} u_{ii}$$

where $s$ is the number of row swaps performed. The system is singular when
$\det(A) = 0$, which happens when any diagonal entry $u_{ii} = 0$.

In floating-point arithmetic, we use a threshold $\tau$ (e.g., $10^{-12}$):

$$|u_{kk}| < \tau \implies \text{system is (numerically) singular}$$

**Singular systems fall into two categories:**

1. **Inconsistent** (no solution): the augmented column has a nonzero entry in a row
   where all coefficient entries are zero.
2. **Underdetermined** (infinitely many solutions): the augmented column also has zero
   in that row. Free variables exist.

### Worked Examples

$$\begin{bmatrix} 1 & 2 \\ 2 & 4 \end{bmatrix} x = \begin{bmatrix} 3 \\ 6 \end{bmatrix}$$

After $R_2 \gets R_2 - 2R_1$: $\begin{bmatrix} 1 & 2 & | & 3 \\ 0 & 0 & | & 0 \end{bmatrix}$

Pivot $u_{22} = 0$. Augmented entry is also 0: underdetermined. Infinite solutions:
$x_1 = 3 - 2t$, $x_2 = t$.

Changing $b = [3, 7]$: after elimination $\begin{bmatrix} 1 & 2 & | & 3 \\ 0 & 0 & | & 1 \end{bmatrix}$.
Row 2 says $0 = 1$: inconsistent. No solution.

---

## Prerequisites

- Matrix multiplication and notation
- Elementary row operations
- Upper/lower triangular matrices
- Floating-point arithmetic basics

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Implement naive Gaussian elimination without pivoting. Solve 2x2 and 3x3 systems by hand. Understand back substitution. |
| **Intermediate** | Add partial pivoting. Detect singular systems. Understand the LU factorization interpretation. Implement in multiple languages with proper floating-point tolerance. |
| **Advanced** | Analyze growth factors and backward error bounds. Implement full pivoting. Study condition numbers and their effect on solution accuracy. Compare with iterative refinement, QR factorization, and SVD for ill-conditioned systems. |
