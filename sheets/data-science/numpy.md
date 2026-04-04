# NumPy (Numerical Python)

NumPy is the fundamental package for numerical computing in Python, providing the ndarray multidimensional array object, vectorized operations, broadcasting semantics, linear algebra routines, random number generation, and C-level performance for array computations that form the foundation of the entire scientific Python ecosystem.

## ndarray Creation
### Array Construction Functions
```python
import numpy as np

# From Python lists
a = np.array([1, 2, 3])                      # 1D
b = np.array([[1, 2], [3, 4]], dtype=np.float64)  # 2D with explicit type

# Zeros, ones, empty
np.zeros((3, 4))                               # 3x4 of zeros
np.ones((2, 3, 4))                             # 3D of ones
np.empty((5, 5))                               # uninitialized (fast)
np.full((3, 3), 7.0)                           # filled with 7.0
np.zeros_like(b)                               # same shape/dtype as b

# Ranges
np.arange(0, 10, 0.5)                         # [0, 0.5, 1.0, ..., 9.5]
np.linspace(0, 1, 100)                         # 100 points from 0 to 1
np.logspace(0, 3, 50)                          # 50 points from 10^0 to 10^3
np.geomspace(1, 1000, 4)                       # [1, 10, 100, 1000]

# Identity and diagonal
np.eye(4)                                      # 4x4 identity
np.diag([1, 2, 3])                             # diagonal matrix
np.diag(b)                                     # extract diagonal

# From functions
np.fromfunction(lambda i, j: i + j, (3, 3))   # f(row, col)
np.frombuffer(raw_bytes, dtype=np.float32)     # from buffer
np.loadtxt('data.csv', delimiter=',')          # from text file
```

## Broadcasting Rules
### Shape Compatibility
```python
# Broadcasting rules:
# 1. Arrays with fewer dims get leading 1s prepended
# 2. Dimensions of size 1 are stretched to match
# 3. Dimensions must be equal or one must be 1

# Examples of compatible shapes:
# (3, 4) + (4,)     -> (3, 4) + (1, 4)     -> (3, 4)
# (3, 1) + (1, 4)   -> (3, 4)
# (5, 3, 4) + (3, 1) -> (5, 3, 4) + (1, 3, 1) -> (5, 3, 4)

# Practical examples
A = np.arange(12).reshape(3, 4)    # shape (3, 4)
row_means = A.mean(axis=1, keepdims=True)  # shape (3, 1)
A_centered = A - row_means         # broadcasts (3, 4) - (3, 1)

# Column-wise standardization
col_mean = A.mean(axis=0)          # shape (4,)
col_std = A.std(axis=0)            # shape (4,)
A_standardized = (A - col_mean) / col_std  # broadcasts correctly

# Outer product via broadcasting
x = np.arange(5)[:, np.newaxis]    # shape (5, 1)
y = np.arange(3)[np.newaxis, :]    # shape (1, 3)
outer = x * y                      # shape (5, 3)

# Distance matrix
points = np.random.randn(100, 3)   # 100 points in 3D
diff = points[:, np.newaxis, :] - points[np.newaxis, :, :]  # (100, 100, 3)
distances = np.sqrt((diff ** 2).sum(axis=2))  # (100, 100)
```

## Vectorized Operations
### Universal Functions (ufuncs)
```python
# Arithmetic (element-wise)
np.add(a, b)           # a + b
np.multiply(a, b)      # a * b
np.power(a, 2)         # a ** 2
np.mod(a, 3)           # a % 3

# Comparison (returns boolean array)
np.greater(a, 5)       # a > 5
np.isnan(a)
np.isinf(a)
np.isfinite(a)

# Math functions
np.exp(a)
np.log(a)              # natural log
np.log2(a)
np.sin(a)
np.sqrt(a)
np.abs(a)
np.clip(a, 0, 1)       # clamp values

# Reduction operations
a.sum()                 # total sum
a.sum(axis=0)           # sum along axis 0
a.mean(axis=1)
a.std(ddof=1)           # sample std dev
a.min(), a.max()
a.argmin(), a.argmax()  # index of min/max
np.cumsum(a)            # cumulative sum
np.cumprod(a)           # cumulative product
np.percentile(a, [25, 50, 75])

# Boolean operations
np.any(a > 5)
np.all(a > 0)
np.count_nonzero(a > 5)
np.where(a > 0, a, 0)  # conditional selection
```

## Linear Algebra
### numpy.linalg
```python
from numpy.linalg import (
    dot, inv, det, eig, svd, solve, norm, matrix_rank, qr, cholesky
)

A = np.array([[2, 1], [1, 3]])
b = np.array([5, 7])

# Matrix multiplication
C = A @ A                          # or np.matmul(A, A)
d = A @ b                          # matrix-vector product
s = np.dot(b, b)                   # dot product (scalar)

# Solving linear systems: Ax = b
x = solve(A, b)                    # x = A^{-1}b (never use inv directly)

# Decompositions
eigenvalues, eigenvectors = eig(A)
U, S, Vt = svd(A)                  # A = U @ diag(S) @ Vt
Q, R = qr(A)                       # A = QR
L = cholesky(A)                    # A = LL^T (A must be positive definite)

# Properties
det(A)                              # determinant
matrix_rank(A)                      # rank
norm(A, 'fro')                      # Frobenius norm
norm(b, 2)                          # L2 norm of vector
np.trace(A)                         # trace (sum of diagonal)

# Pseudoinverse (for non-square/singular matrices)
A_pinv = np.linalg.pinv(A)

# Least squares: min ||Ax - b||^2
x_ls, residuals, rank, sv = np.linalg.lstsq(A, b, rcond=None)
```

## Random Number Generation
### Generator API (NumPy 1.17+)
```python
from numpy.random import default_rng

rng = default_rng(seed=42)         # reproducible

# Uniform distributions
rng.random((3, 4))                  # [0, 1) uniform
rng.uniform(low=5, high=10, size=100)
rng.integers(0, 100, size=50)       # discrete uniform

# Normal distributions
rng.standard_normal((1000,))        # N(0, 1)
rng.normal(loc=100, scale=15, size=1000)  # N(100, 15^2)
rng.multivariate_normal(
    mean=[0, 0],
    cov=[[1, 0.5], [0.5, 1]],
    size=500
)

# Other distributions
rng.poisson(lam=5, size=1000)
rng.exponential(scale=2.0, size=1000)
rng.binomial(n=10, p=0.3, size=1000)
rng.beta(a=2, b=5, size=1000)
rng.gamma(shape=2, scale=1, size=1000)
rng.choice(a, size=10, replace=False)  # sampling without replacement

# Shuffling
rng.shuffle(a)                      # in-place
permuted = rng.permutation(a)       # returns copy
```

## Fancy Indexing and Advanced Selection
### Boolean and Integer Array Indexing
```python
a = np.arange(20).reshape(4, 5)

# Boolean indexing
mask = a > 10
a[mask]                             # 1D array of elements > 10
a[a % 2 == 0]                      # even elements

# Integer array indexing
rows = np.array([0, 2, 3])
cols = np.array([1, 3, 4])
a[rows, cols]                       # elements at (0,1), (2,3), (3,4)
a[np.ix_(rows, cols)]              # submatrix (3x3 block)

# Sorting
np.sort(a, axis=1)                  # sort each row
idx = np.argsort(a[:, 0])          # indices that would sort column 0
a[idx]                              # reorder rows by column 0

# Unique values
unique, counts = np.unique(a, return_counts=True)
unique, indices = np.unique(a, return_index=True)

# Structured indexing with np.take / np.put
np.take(a, [0, 2], axis=0)        # rows 0 and 2
np.put(a, [0, 1, 2], [99, 98, 97])  # flat index assignment
```

## Structured Arrays and Memory Layout
### Custom dtypes and Memory Order
```python
# Structured arrays (record arrays)
dt = np.dtype([
    ('name', 'U20'),
    ('age', 'i4'),
    ('salary', 'f8')
])
employees = np.array([
    ('Alice', 30, 75000.0),
    ('Bob', 25, 65000.0)
], dtype=dt)
employees['name']                   # array(['Alice', 'Bob'])
employees[employees['age'] > 27]   # filter by field

# Memory layout
a_c = np.array([[1, 2], [3, 4]], order='C')   # row-major (default)
a_f = np.array([[1, 2], [3, 4]], order='F')   # column-major (Fortran)

# Check contiguity
a_c.flags['C_CONTIGUOUS']          # True
a_f.flags['F_CONTIGUOUS']          # True

# Views vs copies
b = a[::2]                          # view (shared memory)
c = a[::2].copy()                   # independent copy
np.shares_memory(a, b)             # True

# Strides (bytes between elements)
a = np.zeros((3, 4), dtype=np.float64)
a.strides                           # (32, 8) — 4 floats per row, 8 bytes each

# Reshaping without copy (only if contiguous)
a.reshape(4, 3)                     # new view if possible
a.ravel()                           # 1D view
a.flatten()                         # 1D copy (always)
```

## Performance Patterns
### Vectorization Best Practices
```python
# SLOW: Python loop
result = np.empty(len(a))
for i in range(len(a)):
    result[i] = a[i] ** 2 + 2 * a[i] + 1

# FAST: Vectorized
result = a ** 2 + 2 * a + 1

# Conditional assignment
# SLOW
for i in range(len(a)):
    if a[i] > 0:
        result[i] = np.log(a[i])
    else:
        result[i] = 0

# FAST
result = np.where(a > 0, np.log(np.abs(a)), 0)

# Batch operations
# SLOW: repeated concatenation
arrays = [np.random.randn(100) for _ in range(1000)]
result = arrays[0]
for arr in arrays[1:]:
    result = np.concatenate([result, arr])

# FAST: single concatenation
result = np.concatenate(arrays)

# Memory-efficient operations with out parameter
np.multiply(a, b, out=result)      # write directly to pre-allocated array
```

## Tips
- Always use the new `default_rng()` Generator API instead of `np.random.seed()` for better statistical properties and thread safety
- Prefer `np.linalg.solve(A, b)` over `np.linalg.inv(A) @ b` for solving linear systems; it is faster and numerically more stable
- Use `keepdims=True` in reductions to maintain broadcasting compatibility when combining results back with the original array
- Understand the difference between views and copies; slicing creates views (shared memory) while fancy indexing creates copies
- Use `np.einsum()` for complex tensor contractions; it is often faster and more readable than chains of reshape/transpose/matmul
- Choose C-order (row-major) for row-wise operations and F-order (column-major) for column-wise operations to maximize cache locality
- Pre-allocate output arrays with `np.empty()` and use the `out=` parameter of ufuncs to avoid temporary array allocations
- Use `np.float32` instead of `np.float64` when precision allows; it halves memory usage and can double throughput on modern CPUs
- Avoid boolean indexing in tight loops; build the mask once and apply it, or use `np.where` for branchless computation
- Profile with `%timeit` before optimizing; NumPy's internal optimizations sometimes make seemingly slow patterns fast

## See Also
- pandas, scikit-learn, scipy, tensorflow, pytorch

## References
- [NumPy Official Documentation](https://numpy.org/doc/stable/)
- [NumPy User Guide](https://numpy.org/doc/stable/user/index.html)
- [NumPy for MATLAB Users](https://numpy.org/doc/stable/user/numpy-for-matlab-users.html)
- [From Python to NumPy (Nicolas Rougier)](https://www.labri.fr/perso/nrougier/from-python-to-numpy/)
