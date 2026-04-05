# Sliding Window Maximum (Arrays / Monotonic Deque)

Find the maximum value in every contiguous subarray of size k using a monotonic deque for O(n) performance.

## Problem

Given an integer array `nums` and a sliding window of size `k`, return an array of the maximum
values in each window position as the window moves from left to right across the array.

**Constraints:**

- `1 <= nums.length <= 10^5`
- `-10^4 <= nums[i] <= 10^4`
- `1 <= k <= nums.length`

**Example 1:**

```
Input:  nums = [1, 3, -1, -3, 5, 3, 6, 7], k = 3
Output: [3, 3, 5, 5, 6, 7]

Window positions:
  [1  3  -1] -3  5  3  6  7   -> max = 3
   1 [3  -1  -3] 5  3  6  7   -> max = 3
   1  3 [-1  -3  5] 3  6  7   -> max = 5
   1  3  -1 [-3  5  3] 6  7   -> max = 5
   1  3  -1  -3 [5  3  6] 7   -> max = 6
   1  3  -1  -3  5 [3  6  7]  -> max = 7
```

**Example 2:**

```
Input:  nums = [1], k = 1
Output: [1]
```

**Example 3:**

```
Input:  nums = [9, 8, 7, 6, 5], k = 3
Output: [9, 8, 7]
```

## Hints

- Maintain a deque (double-ended queue) that stores **indices**, not values, in decreasing order of their corresponding values.
- The front of the deque always holds the index of the current window's maximum. Pop from the front when it falls outside the window boundary.
- Before pushing a new index, pop all indices from the back whose values are less than or equal to the new element -- they can never be a future maximum.

## Solution -- Go

```go
package main

import "fmt"

func maxSlidingWindow(nums []int, k int) []int {
	// dq stores indices in decreasing order of nums values
	dq := make([]int, 0, k)
	result := make([]int, 0, len(nums)-k+1)

	for i := 0; i < len(nums); i++ {
		// Remove indices outside the window
		for len(dq) > 0 && dq[0] < i-k+1 {
			dq = dq[1:]
		}

		// Remove indices whose values are <= current value
		for len(dq) > 0 && nums[dq[len(dq)-1]] <= nums[i] {
			dq = dq[:len(dq)-1]
		}

		dq = append(dq, i)

		// Record max once we have a full window
		if i >= k-1 {
			result = append(result, nums[dq[0]])
		}
	}

	return result
}

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func main() {
	tests := []struct {
		nums     []int
		k        int
		expected []int
	}{
		{[]int{1, 3, -1, -3, 5, 3, 6, 7}, 3, []int{3, 3, 5, 5, 6, 7}},
		{[]int{1}, 1, []int{1}},
		{[]int{4, 2, 7, 3}, 4, []int{7}},
		{[]int{5, 3, 8, 1}, 1, []int{5, 3, 8, 1}},
		{[]int{9, 8, 7, 6, 5}, 3, []int{9, 8, 7}},
		{[]int{3, 3, 3, 3}, 2, []int{3, 3, 3}},
	}

	for i, tc := range tests {
		got := maxSlidingWindow(tc.nums, tc.k)
		if !sliceEqual(got, tc.expected) {
			panic(fmt.Sprintf("Test %d FAILED: got %v, want %v", i, got, tc.expected))
		}
	}
	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from collections import deque
from typing import List


class Solution:
    def max_sliding_window(self, nums: List[int], k: int) -> List[int]:
        # dq stores indices; nums[dq[0]] is always the window max
        dq: deque = deque()
        result: List[int] = []

        for i, val in enumerate(nums):
            # Remove indices that have fallen out of the window
            while dq and dq[0] < i - k + 1:
                dq.popleft()

            # Maintain decreasing order: remove smaller elements from back
            while dq and nums[dq[-1]] <= val:
                dq.pop()

            dq.append(i)

            # Once we have a full window, record the max
            if i >= k - 1:
                result.append(nums[dq[0]])

        return result


if __name__ == "__main__":
    s = Solution()

    # Basic test
    assert s.max_sliding_window([1, 3, -1, -3, 5, 3, 6, 7], 3) == [3, 3, 5, 5, 6, 7]

    # Single element
    assert s.max_sliding_window([1], 1) == [1]

    # Window equals array length
    assert s.max_sliding_window([4, 2, 7, 3], 4) == [7]

    # Window of size 1 => original array
    assert s.max_sliding_window([5, 3, 8, 1], 1) == [5, 3, 8, 1]

    # Decreasing array
    assert s.max_sliding_window([9, 8, 7, 6, 5], 3) == [9, 8, 7]

    # All same values
    assert s.max_sliding_window([3, 3, 3, 3], 2) == [3, 3, 3]

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::VecDeque;

struct Solution;

impl Solution {
    fn max_sliding_window(nums: &[i32], k: usize) -> Vec<i32> {
        let mut dq: VecDeque<usize> = VecDeque::new();
        let mut result = Vec::with_capacity(nums.len() - k + 1);

        for i in 0..nums.len() {
            // Remove indices outside the window
            while let Some(&front) = dq.front() {
                if front + k <= i {
                    dq.pop_front();
                } else {
                    break;
                }
            }

            // Remove indices whose values are <= current value
            while let Some(&back) = dq.back() {
                if nums[back] <= nums[i] {
                    dq.pop_back();
                } else {
                    break;
                }
            }

            dq.push_back(i);

            // Record max once we have a full window
            if i >= k - 1 {
                result.push(nums[dq[0]]);
            }
        }

        result
    }
}

fn main() {
    assert_eq!(
        Solution::max_sliding_window(&[1, 3, -1, -3, 5, 3, 6, 7], 3),
        vec![3, 3, 5, 5, 6, 7]
    );
    assert_eq!(Solution::max_sliding_window(&[1], 1), vec![1]);
    assert_eq!(Solution::max_sliding_window(&[4, 2, 7, 3], 4), vec![7]);
    assert_eq!(
        Solution::max_sliding_window(&[5, 3, 8, 1], 1),
        vec![5, 3, 8, 1]
    );
    assert_eq!(
        Solution::max_sliding_window(&[9, 8, 7, 6, 5], 3),
        vec![9, 8, 7]
    );
    assert_eq!(
        Solution::max_sliding_window(&[3, 3, 3, 3], 2),
        vec![3, 3, 3]
    );
    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function maxSlidingWindow(nums: number[], k: number): number[] {
    // Using an array as a deque (indices stored in decreasing value order)
    const dq: number[] = [];
    const result: number[] = [];

    for (let i = 0; i < nums.length; i++) {
        // Remove indices outside the window
        while (dq.length > 0 && dq[0] < i - k + 1) {
            dq.shift();
        }

        // Remove indices whose values are <= current value
        while (dq.length > 0 && nums[dq[dq.length - 1]] <= nums[i]) {
            dq.pop();
        }

        dq.push(i);

        // Record max once we have a full window
        if (i >= k - 1) {
            result.push(nums[dq[0]]);
        }
    }

    return result;
}

// Tests
function arraysEqual(a: number[], b: number[]): boolean {
    return a.length === b.length && a.every((v, i) => v === b[i]);
}

console.assert(
    arraysEqual(maxSlidingWindow([1, 3, -1, -3, 5, 3, 6, 7], 3), [3, 3, 5, 5, 6, 7]),
    "Test 1 failed"
);
console.assert(arraysEqual(maxSlidingWindow([1], 1), [1]), "Test 2 failed");
console.assert(arraysEqual(maxSlidingWindow([4, 2, 7, 3], 4), [7]), "Test 3 failed");
console.assert(
    arraysEqual(maxSlidingWindow([5, 3, 8, 1], 1), [5, 3, 8, 1]),
    "Test 4 failed"
);
console.assert(
    arraysEqual(maxSlidingWindow([9, 8, 7, 6, 5], 3), [9, 8, 7]),
    "Test 5 failed"
);
console.assert(
    arraysEqual(maxSlidingWindow([3, 3, 3, 3], 2), [3, 3, 3]),
    "Test 6 failed"
);
console.log("All tests passed!");
```

## Complexity

| Metric | Value |
|--------|-------|
| Time   | O(n) -- each element is pushed and popped from the deque at most once |
| Space  | O(k) -- the deque holds at most k indices at any time |

## Tips

- The deque invariant is **monotonically decreasing** values from front to back. This guarantees the front is always the maximum for the current window.
- Every element enters and leaves the deque exactly once, giving the amortized O(n) bound despite the nested loops.
- In Go and TypeScript, a slice/array simulates the deque; in Python and Rust, use the standard library `deque` / `VecDeque` for O(1) front removal.
- Edge case: when `k == 1`, the output is the input array itself; when `k == n`, the output is a single-element array containing the global maximum.

## See Also

- **Monotonic Stack** -- same structural idea applied to next-greater-element and histogram problems.
- **Minimum Window Substring** -- another sliding window classic, but with hash-map tracking instead of a deque.
- **Longest Substring Without Repeating Characters** -- variable-width sliding window variant.
- **Range Maximum Query (Sparse Table)** -- O(1) per query with O(n log n) preprocessing, useful for static arrays.

## References

- [LeetCode 239 -- Sliding Window Maximum](https://leetcode.com/problems/sliding-window-maximum/)
- [NeetCode Explanation](https://neetcode.io/problems/sliding-window-maximum)
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 10 -- Stacks and Queues
