# Median of Two Sorted Arrays (Binary Search)

Find the median of two sorted arrays in O(log(min(m, n))) time by binary-searching a partition point that balances the combined lower and upper halves.

## Problem

Given two sorted arrays `nums1` of size `m` and `nums2` of size `n`, return the median of the two sorted arrays. The overall run-time complexity must be **O(log (m + n))** or better.

**Constraints:**

- `nums1.length == m`, `nums2.length == n`
- `0 <= m, n <= 1000`, `1 <= m + n <= 2000`
- `-10^6 <= nums1[i], nums2[i] <= 10^6`
- Both arrays are sorted in non-decreasing order.

**Example 1:**

```
Input:  nums1 = [1, 3], nums2 = [2]
Output: 2.00000
Explanation: merged = [1, 2, 3], median = 2
```

**Example 2:**

```
Input:  nums1 = [1, 2], nums2 = [3, 4]
Output: 2.50000
Explanation: merged = [1, 2, 3, 4], median = (2 + 3) / 2 = 2.5
```

**Example 3:**

```
Input:  nums1 = [], nums2 = [1]
Output: 1.00000
```

## Hints

1. Merging and taking the middle is **O(m + n)** — too slow. You must avoid looking at most elements.
2. Binary-search a **partition index** `i` in the shorter array such that `nums1[0..i-1]` plus `nums2[0..j-1]` forms the lower half of the combined array, where `j = (m + n + 1) / 2 - i`.
3. The correct partition satisfies `nums1[i-1] <= nums2[j]` and `nums2[j-1] <= nums1[i]` (the four-pointer invariant).
4. Always binary-search the **shorter** array so `i` stays in `[0, m]` and `j` stays non-negative.
5. Use sentinels `-∞` and `+∞` to handle the partition at either end without special-casing.

## Solution -- Go

```go
package main

import (
	"fmt"
	"math"
)

func findMedianSortedArrays(nums1, nums2 []int) float64 {
	// Always binary-search the shorter array
	if len(nums1) > len(nums2) {
		nums1, nums2 = nums2, nums1
	}
	m, n := len(nums1), len(nums2)
	half := (m + n + 1) / 2

	lo, hi := 0, m
	for lo <= hi {
		i := (lo + hi) / 2
		j := half - i

		leftA := math.MinInt
		if i > 0 {
			leftA = nums1[i-1]
		}
		rightA := math.MaxInt
		if i < m {
			rightA = nums1[i]
		}
		leftB := math.MinInt
		if j > 0 {
			leftB = nums2[j-1]
		}
		rightB := math.MaxInt
		if j < n {
			rightB = nums2[j]
		}

		if leftA <= rightB && leftB <= rightA {
			if (m+n)%2 == 1 {
				return float64(max(leftA, leftB))
			}
			return float64(max(leftA, leftB)+min(rightA, rightB)) / 2.0
		} else if leftA > rightB {
			hi = i - 1
		} else {
			lo = i + 1
		}
	}
	panic("inputs are not sorted")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	if got := findMedianSortedArrays([]int{1, 3}, []int{2}); got != 2.0 {
		panic(fmt.Sprintf("Test 1 FAILED: got %v", got))
	}
	if got := findMedianSortedArrays([]int{1, 2}, []int{3, 4}); got != 2.5 {
		panic(fmt.Sprintf("Test 2 FAILED: got %v", got))
	}
	if got := findMedianSortedArrays([]int{}, []int{1}); got != 1.0 {
		panic(fmt.Sprintf("Test 3 FAILED: got %v", got))
	}
	if got := findMedianSortedArrays([]int{1, 2, 3, 4, 5}, []int{6, 7, 8, 9, 10}); got != 5.5 {
		panic(fmt.Sprintf("Test 4 FAILED: got %v", got))
	}
	// Disjoint
	if got := findMedianSortedArrays([]int{1, 2}, []int{-1, 3}); got != 1.5 {
		panic(fmt.Sprintf("Test 5 FAILED: got %v", got))
	}
	// Equal values
	if got := findMedianSortedArrays([]int{1, 1}, []int{1, 1}); got != 1.0 {
		panic(fmt.Sprintf("Test 6 FAILED: got %v", got))
	}
	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from typing import List
import math


def find_median_sorted_arrays(nums1: List[int], nums2: List[int]) -> float:
    # Binary-search the shorter array for smaller search space
    if len(nums1) > len(nums2):
        nums1, nums2 = nums2, nums1

    m, n = len(nums1), len(nums2)
    half = (m + n + 1) // 2
    lo, hi = 0, m

    while lo <= hi:
        i = (lo + hi) // 2
        j = half - i

        left_a = nums1[i - 1] if i > 0 else -math.inf
        right_a = nums1[i] if i < m else math.inf
        left_b = nums2[j - 1] if j > 0 else -math.inf
        right_b = nums2[j] if j < n else math.inf

        if left_a <= right_b and left_b <= right_a:
            if (m + n) % 2 == 1:
                return float(max(left_a, left_b))
            return (max(left_a, left_b) + min(right_a, right_b)) / 2.0
        elif left_a > right_b:
            hi = i - 1
        else:
            lo = i + 1

    raise ValueError("inputs not sorted")


if __name__ == "__main__":
    assert find_median_sorted_arrays([1, 3], [2]) == 2.0, "Test 1"
    assert find_median_sorted_arrays([1, 2], [3, 4]) == 2.5, "Test 2"
    assert find_median_sorted_arrays([], [1]) == 1.0, "Test 3"
    assert find_median_sorted_arrays([1, 2, 3, 4, 5], [6, 7, 8, 9, 10]) == 5.5, "Test 4"
    assert find_median_sorted_arrays([1, 2], [-1, 3]) == 1.5, "Test 5"
    assert find_median_sorted_arrays([1, 1], [1, 1]) == 1.0, "Test 6"
    # One empty
    assert find_median_sorted_arrays([1, 2, 3, 4], []) == 2.5, "Test 7"
    print("All tests passed!")
```

## Solution -- Rust

```rust
fn find_median_sorted_arrays(nums1: Vec<i32>, nums2: Vec<i32>) -> f64 {
    let (a, b) = if nums1.len() > nums2.len() {
        (nums2, nums1)
    } else {
        (nums1, nums2)
    };
    let m = a.len();
    let n = b.len();
    let half = (m + n + 1) / 2;
    let (mut lo, mut hi) = (0isize, m as isize);

    while lo <= hi {
        let i = ((lo + hi) / 2) as usize;
        let j = half - i;

        let left_a = if i > 0 { a[i - 1] } else { i32::MIN };
        let right_a = if i < m { a[i] } else { i32::MAX };
        let left_b = if j > 0 { b[j - 1] } else { i32::MIN };
        let right_b = if j < n { b[j] } else { i32::MAX };

        if left_a <= right_b && left_b <= right_a {
            if (m + n) % 2 == 1 {
                return left_a.max(left_b) as f64;
            }
            return (left_a.max(left_b) as f64 + right_a.min(right_b) as f64) / 2.0;
        } else if left_a > right_b {
            hi = i as isize - 1;
        } else {
            lo = i as isize + 1;
        }
    }
    panic!("inputs not sorted");
}

fn main() {
    assert_eq!(find_median_sorted_arrays(vec![1, 3], vec![2]), 2.0);
    assert_eq!(find_median_sorted_arrays(vec![1, 2], vec![3, 4]), 2.5);
    assert_eq!(find_median_sorted_arrays(vec![], vec![1]), 1.0);
    assert_eq!(
        find_median_sorted_arrays(vec![1, 2, 3, 4, 5], vec![6, 7, 8, 9, 10]),
        5.5
    );
    assert_eq!(find_median_sorted_arrays(vec![1, 2], vec![-1, 3]), 1.5);
    assert_eq!(find_median_sorted_arrays(vec![1, 1], vec![1, 1]), 1.0);
    assert_eq!(find_median_sorted_arrays(vec![1, 2, 3, 4], vec![]), 2.5);
    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function findMedianSortedArrays(nums1: number[], nums2: number[]): number {
    if (nums1.length > nums2.length) {
        [nums1, nums2] = [nums2, nums1];
    }
    const m = nums1.length;
    const n = nums2.length;
    const half = Math.floor((m + n + 1) / 2);
    let lo = 0, hi = m;

    while (lo <= hi) {
        const i = Math.floor((lo + hi) / 2);
        const j = half - i;

        const leftA = i > 0 ? nums1[i - 1] : -Infinity;
        const rightA = i < m ? nums1[i] : Infinity;
        const leftB = j > 0 ? nums2[j - 1] : -Infinity;
        const rightB = j < n ? nums2[j] : Infinity;

        if (leftA <= rightB && leftB <= rightA) {
            if ((m + n) % 2 === 1) {
                return Math.max(leftA, leftB);
            }
            return (Math.max(leftA, leftB) + Math.min(rightA, rightB)) / 2;
        } else if (leftA > rightB) {
            hi = i - 1;
        } else {
            lo = i + 1;
        }
    }
    throw new Error("inputs not sorted");
}

console.assert(findMedianSortedArrays([1, 3], [2]) === 2.0, "Test 1");
console.assert(findMedianSortedArrays([1, 2], [3, 4]) === 2.5, "Test 2");
console.assert(findMedianSortedArrays([], [1]) === 1.0, "Test 3");
console.assert(findMedianSortedArrays([1, 2, 3, 4, 5], [6, 7, 8, 9, 10]) === 5.5, "Test 4");
console.assert(findMedianSortedArrays([1, 2], [-1, 3]) === 1.5, "Test 5");
console.assert(findMedianSortedArrays([1, 1], [1, 1]) === 1.0, "Test 6");
console.assert(findMedianSortedArrays([1, 2, 3, 4], []) === 2.5, "Test 7");
console.log("All tests passed!");
```

## Complexity

| Aspect | Bound |
|--------|-------|
| Time   | O(log(min(m, n))) |
| Space  | O(1) |

- Binary search halves the partition range each iteration.
- Always search the shorter array to guarantee the log-min bound.
- Constant extra space — no merging or auxiliary array.

## Tips

- **Search the shorter array**, not the first argument. If you forget, your `j` index can go negative and crash the partition logic.
- **Use sentinels** (`-∞` / `+∞`) for partitions at the boundary. Conditional branching per boundary doubles the code and invites off-by-one bugs.
- The invariant `leftA <= rightB && leftB <= rightA` is the **four-pointer merge condition** — every sorted element to the left is ≤ every element to the right across both arrays.
- **Even vs odd total length**: for odd total, the median is the max of the two lefts (the "extra" element is on the left side by our `half = (m + n + 1) / 2` convention). For even total, average the two middle elements.
- The problem generalises to finding the **k-th smallest** element in two sorted arrays with the same technique — use `half = k` instead of `(m + n + 1) / 2`.
- **Why not merge-and-index?** Merging is O(m + n); the problem demands sub-linear. Interviewers specifically probe whether you recognise that most elements can be dismissed without inspection.

## See Also

- [Kth Smallest in Sorted Matrix](kth-smallest-sorted-matrix.md) -- generalisation to 2D sorted structure.
- [Binary Search](binary-search.md) -- the primitive operation.
- [Merge K Sorted Lists](merge-k-sorted-lists.md) -- related merge problem, but O(n log k) instead of O(log n).

## References

- LeetCode 4: Median of Two Sorted Arrays
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 9 (Selection)
- Knuth, *The Art of Computer Programming*, Vol. 3, Section 5.3.3 (Optimum merging)
