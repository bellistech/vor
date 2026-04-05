# Gaussian Elimination (Linear Algebra / Numerical Methods)

Solve a system of n linear equations in n unknowns using Gaussian elimination with partial pivoting, handling singular and inconsistent systems.

## Problem

Given an n x n coefficient matrix A and an n-element right-hand side vector b, solve
the system Ax = b for the unknown vector x.

**Constraints:**

- `1 <= n <= 1000`
- Coefficients are floating-point numbers.
- The system may be singular (no unique solution) or inconsistent (no solution).
- Use partial pivoting for numerical stability.

**Examples:**

```
System:
   2x +  y -  z =  8
  -3x -  y + 2z = -11
  -2x +  y + 2z = -3

A = [[ 2,  1, -1],
     [-3, -1,  2],
     [-2,  1,  2]]
b = [8, -11, -3]

Solution: x = [2, 3, -1]
```

## Hints

- **Forward elimination:** For each column k, find the row with the largest absolute
  value in column k (partial pivot), swap it to position k, then eliminate all entries
  below the pivot.
- **Back substitution:** After the matrix is in upper triangular form, solve from the
  last row upward.
- **Singular detection:** If a pivot is zero (or near zero after pivoting), the system
  is singular -- return an error or None.
- **Augmented matrix:** Append b as a column to A so row swaps apply to both.

## Solution -- Go

```go
import (
	"errors"
	"math"
)

func gaussianElimination(A [][]float64, b []float64) ([]float64, error) {
	n := len(b)
	if n == 0 {
		return nil, errors.New("empty system")
	}

	// Build augmented matrix [A|b]
	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i], A[i])
		aug[i][n] = b[i]
	}

	// Forward elimination with partial pivoting
	for k := 0; k < n; k++ {
		// Find pivot: row with largest |A[i][k]| for i >= k
		maxVal := math.Abs(aug[k][k])
		maxRow := k
		for i := k + 1; i < n; i++ {
			if v := math.Abs(aug[i][k]); v > maxVal {
				maxVal = v
				maxRow = i
			}
		}

		if maxVal < 1e-12 {
			return nil, errors.New("singular matrix")
		}

		// Swap rows
		aug[k], aug[maxRow] = aug[maxRow], aug[k]

		// Eliminate below
		for i := k + 1; i < n; i++ {
			factor := aug[i][k] / aug[k][k]
			for j := k; j <= n; j++ {
				aug[i][j] -= factor * aug[k][j]
			}
		}
	}

	// Back substitution
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = aug[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= aug[i][j] * x[j]
		}
		x[i] /= aug[i][i]
	}

	return x, nil
}
```

## Solution -- Python

```python
from typing import Optional, List


def gaussian_elimination(
    A: List[List[float]], b: List[float]
) -> Optional[List[float]]:
    """Solve Ax = b using Gaussian elimination with partial pivoting."""
    n = len(b)
    if n == 0:
        return None

    # Build augmented matrix [A|b]
    aug = [row[:] + [bi] for row, bi in zip(A, b)]

    # Forward elimination with partial pivoting
    for k in range(n):
        # Find pivot: row with largest |aug[i][k]| for i >= k
        max_row = k
        max_val = abs(aug[k][k])
        for i in range(k + 1, n):
            if abs(aug[i][k]) > max_val:
                max_val = abs(aug[i][k])
                max_row = i

        if max_val < 1e-12:
            return None  # Singular matrix

        # Swap rows
        aug[k], aug[max_row] = aug[max_row], aug[k]

        # Eliminate below
        for i in range(k + 1, n):
            factor = aug[i][k] / aug[k][k]
            for j in range(k, n + 1):
                aug[i][j] -= factor * aug[k][j]

    # Back substitution
    x = [0.0] * n
    for i in range(n - 1, -1, -1):
        x[i] = aug[i][n]
        for j in range(i + 1, n):
            x[i] -= aug[i][j] * x[j]
        x[i] /= aug[i][i]

    return x
```

## Solution -- Rust

```rust
fn gaussian_elimination(a: &[Vec<f64>], b: &[f64]) -> Option<Vec<f64>> {
    let n = b.len();
    if n == 0 {
        return None;
    }

    // Build augmented matrix [A|b]
    let mut aug: Vec<Vec<f64>> = Vec::with_capacity(n);
    for i in 0..n {
        let mut row = a[i].clone();
        row.push(b[i]);
        aug.push(row);
    }

    // Forward elimination with partial pivoting
    for k in 0..n {
        // Find pivot
        let mut max_row = k;
        let mut max_val = aug[k][k].abs();
        for i in (k + 1)..n {
            let v = aug[i][k].abs();
            if v > max_val {
                max_val = v;
                max_row = i;
            }
        }

        if max_val < 1e-12 {
            return None; // Singular matrix
        }

        // Swap rows
        aug.swap(k, max_row);

        // Eliminate below
        for i in (k + 1)..n {
            let factor = aug[i][k] / aug[k][k];
            for j in k..=n {
                aug[i][j] -= factor * aug[k][j];
            }
        }
    }

    // Back substitution
    let mut x = vec![0.0; n];
    for i in (0..n).rev() {
        x[i] = aug[i][n];
        for j in (i + 1)..n {
            x[i] -= aug[i][j] * x[j];
        }
        x[i] /= aug[i][i];
    }

    Some(x)
}
```

## Solution -- TypeScript

```typescript
function gaussianElimination(
    A: number[][],
    b: number[]
): number[] | null {
    const n = b.length;
    if (n === 0) return null;

    // Build augmented matrix [A|b]
    const aug: number[][] = A.map((row, i) => [...row, b[i]]);

    // Forward elimination with partial pivoting
    for (let k = 0; k < n; k++) {
        // Find pivot: row with largest |aug[i][k]| for i >= k
        let maxRow = k;
        let maxVal = Math.abs(aug[k][k]);
        for (let i = k + 1; i < n; i++) {
            const v = Math.abs(aug[i][k]);
            if (v > maxVal) {
                maxVal = v;
                maxRow = i;
            }
        }

        if (maxVal < 1e-12) {
            return null; // Singular matrix
        }

        // Swap rows
        [aug[k], aug[maxRow]] = [aug[maxRow], aug[k]];

        // Eliminate below
        for (let i = k + 1; i < n; i++) {
            const factor = aug[i][k] / aug[k][k];
            for (let j = k; j <= n; j++) {
                aug[i][j] -= factor * aug[k][j];
            }
        }
    }

    // Back substitution
    const x = new Array(n).fill(0);
    for (let i = n - 1; i >= 0; i--) {
        x[i] = aug[i][n];
        for (let j = i + 1; j < n; j++) {
            x[i] -= aug[i][j] * x[j];
        }
        x[i] /= aug[i][i];
    }

    return x;
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(n^3) -- three nested loops over n |
| Space | O(n^2) -- augmented matrix |

## Tips

- **Partial pivoting is essential.** Without it, small pivots amplify rounding errors
  catastrophically. Always swap in the row with the largest absolute value in the pivot column.
- **Full pivoting** (searching rows *and* columns) gives better numerical stability but
  is rarely worth the overhead. Partial pivoting is sufficient for most practical problems.
- **Singularity threshold:** Use a tolerance like `1e-12` rather than exact zero. Floating-point
  arithmetic makes exact zero comparisons unreliable.
- **In-place vs. augmented:** The augmented matrix `[A|b]` approach is clearest for
  implementation. Production solvers (LAPACK) work in-place with separate L and U factors.
- **LU decomposition** is the factored form of Gaussian elimination: `A = PLU` where P is
  the permutation matrix from pivoting, L is lower triangular, U is upper triangular. Once
  factored, solving for multiple right-hand sides is O(n^2) each.
- **Do not use for large sparse systems.** For sparse matrices, use iterative methods
  (conjugate gradient, GMRES) or sparse direct solvers.

## See Also

- matrix-operations
- lu-decomposition
- numerical-methods
- linear-algebra

## References

- [Gaussian Elimination (Wikipedia)](https://en.wikipedia.org/wiki/Gaussian_elimination)
- [Numerical Recipes, Chapter 2 -- Solution of Linear Algebraic Equations](https://numerical.recipes/)
- [LAPACK Users' Guide](https://www.netlib.org/lapack/lug/)
