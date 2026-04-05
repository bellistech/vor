# Longest Increasing Subsequence (Dynamic Programming / Binary Search)

Find the length of the longest strictly increasing subsequence in an integer array.

## Problem

Given an integer array `nums`, return the length of the longest strictly increasing
subsequence (LIS).

A subsequence is derived from the array by deleting some or no elements without
changing the order of the remaining elements.

**Constraints:**

- `1 <= nums.length <= 2500`
- `-10^4 <= nums[i] <= 10^4`

**Examples:**

```
[10, 9, 2, 5, 3, 7, 101, 18] => 4
  Explanation: [2, 3, 7, 101] or [2, 3, 7, 18] or [2, 5, 7, 101] etc.

[0, 1, 0, 3, 2, 3] => 4
  Explanation: [0, 1, 2, 3]

[7, 7, 7, 7, 7, 7, 7] => 1
  Explanation: all elements are equal; strictly increasing requires length 1
```

## Hints

- **O(n^2) DP:** Define `dp[i]` as the length of the LIS ending at index `i`. For each `i`,
  check all `j < i` where `nums[j] < nums[i]` and take `dp[i] = max(dp[j] + 1)`.
- **O(n log n) Patience sort:** Maintain a `tails` array where `tails[k]` holds the
  smallest tail element of all increasing subsequences of length `k+1`. For each element,
  binary search for its position in `tails`. If it extends the longest subsequence, append;
  otherwise, replace the first element >= it.
- The `tails` array is always sorted, enabling binary search.

## Solution -- Go

```go
import (
	"sort"
)

// lengthOfLISDP is the O(n^2) DP approach.
func lengthOfLISDP(nums []int) int {
	n := len(nums)
	dp := make([]int, n)
	for i := range dp {
		dp[i] = 1
	}

	best := 1
	for i := 1; i < n; i++ {
		for j := 0; j < i; j++ {
			if nums[j] < nums[i] && dp[j]+1 > dp[i] {
				dp[i] = dp[j] + 1
			}
		}
		if dp[i] > best {
			best = dp[i]
		}
	}
	return best
}

// lengthOfLIS is the O(n log n) patience sort approach.
func lengthOfLIS(nums []int) int {
	// tails[i] = smallest tail element of all increasing subsequences of length i+1
	tails := make([]int, 0, len(nums))

	for _, num := range nums {
		// Binary search for leftmost position where tails[pos] >= num
		pos := sort.SearchInts(tails, num)

		if pos == len(tails) {
			// num is larger than all tails; extend
			tails = append(tails, num)
		} else {
			// Replace with smaller value
			tails[pos] = num
		}
	}

	return len(tails)
}
```

## Solution -- Python

```python
from typing import List
import bisect


class Solution:
    def length_of_lis_dp(self, nums: List[int]) -> int:
        """O(n^2) dynamic programming approach."""
        n = len(nums)
        dp = [1] * n  # Each element is a subsequence of length 1

        for i in range(1, n):
            for j in range(i):
                if nums[j] < nums[i]:
                    dp[i] = max(dp[i], dp[j] + 1)

        return max(dp)

    def length_of_lis(self, nums: List[int]) -> int:
        """O(n log n) patience sort with binary search."""
        # tails[i] = smallest tail of all increasing subsequences of length i+1
        tails: List[int] = []

        for num in nums:
            # Binary search for the leftmost position where tails[pos] >= num
            pos = bisect.bisect_left(tails, num)

            if pos == len(tails):
                # num is larger than all tails; extend the longest subsequence
                tails.append(num)
            else:
                # Replace tails[pos] with num (smaller tail is better)
                tails[pos] = num

        return len(tails)
```

## Solution -- Rust

```rust
struct Solution;

impl Solution {
    /// O(n^2) DP approach
    fn length_of_lis_dp(nums: &[i32]) -> usize {
        let n = nums.len();
        let mut dp = vec![1usize; n];
        let mut best = 1;

        for i in 1..n {
            for j in 0..i {
                if nums[j] < nums[i] && dp[j] + 1 > dp[i] {
                    dp[i] = dp[j] + 1;
                }
            }
            if dp[i] > best {
                best = dp[i];
            }
        }
        best
    }

    /// O(n log n) patience sort with binary search
    fn length_of_lis(nums: &[i32]) -> usize {
        // tails[i] = smallest tail of all increasing subsequences of length i+1
        let mut tails: Vec<i32> = Vec::new();

        for &num in nums {
            // Binary search for leftmost position where tails[pos] >= num
            let pos = tails.partition_point(|&x| x < num);

            if pos == tails.len() {
                tails.push(num);
            } else {
                tails[pos] = num;
            }
        }

        tails.len()
    }
}
```

## Solution -- TypeScript

```typescript
/** O(n^2) DP approach */
function lengthOfLISDP(nums: number[]): number {
    const n = nums.length;
    const dp = new Array(n).fill(1);
    let best = 1;

    for (let i = 1; i < n; i++) {
        for (let j = 0; j < i; j++) {
            if (nums[j] < nums[i]) {
                dp[i] = Math.max(dp[i], dp[j] + 1);
            }
        }
        best = Math.max(best, dp[i]);
    }
    return best;
}

/** O(n log n) patience sort with binary search */
function lengthOfLIS(nums: number[]): number {
    // tails[i] = smallest tail of all increasing subsequences of length i+1
    const tails: number[] = [];

    for (const num of nums) {
        // Binary search for leftmost position where tails[pos] >= num
        let lo = 0;
        let hi = tails.length;
        while (lo < hi) {
            const mid = (lo + hi) >> 1;
            if (tails[mid] < num) {
                lo = mid + 1;
            } else {
                hi = mid;
            }
        }

        if (lo === tails.length) {
            tails.push(num);
        } else {
            tails[lo] = num;
        }
    }

    return tails.length;
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time (DP) | O(n^2) -- nested loop over all pairs |
| Time (Patience) | O(n log n) -- binary search for each of n elements |
| Space (DP) | O(n) -- the dp array |
| Space (Patience) | O(n) -- the tails array (at most n elements) |

## Walkthrough

### Tracing the O(n^2) DP on [10, 9, 2, 5, 3, 7, 101, 18]

```
Index:  0   1   2   3   4   5    6    7
Value: 10   9   2   5   3   7  101   18
dp:     1   1   1   2   2   3    4    4
```

- `dp[0] = 1`: base case, [10]
- `dp[1] = 1`: no j < 1 with nums[j] < 9 (10 >= 9)
- `dp[2] = 1`: no j < 2 with nums[j] < 2
- `dp[3] = 2`: nums[2]=2 < 5, so dp[3] = dp[2]+1 = 2. Subsequence: [2, 5]
- `dp[4] = 2`: nums[2]=2 < 3, so dp[4] = dp[2]+1 = 2. Subsequence: [2, 3]
- `dp[5] = 3`: best from dp[3]+1=3 or dp[4]+1=3. Subsequence: [2, 5, 7] or [2, 3, 7]
- `dp[6] = 4`: dp[5]+1 = 4. Subsequence: [2, 3, 7, 101]
- `dp[7] = 4`: dp[5]+1 = 4. Subsequence: [2, 3, 7, 18]

### Tracing the O(n log n) patience sort on [10, 9, 2, 5, 3, 7, 101, 18]

```
Step 1: num=10,  tails=[]      -> append      -> [10]
Step 2: num=9,   tails=[10]    -> replace [0] -> [9]
Step 3: num=2,   tails=[9]     -> replace [0] -> [2]
Step 4: num=5,   tails=[2]     -> append      -> [2, 5]
Step 5: num=3,   tails=[2, 5]  -> replace [1] -> [2, 3]
Step 6: num=7,   tails=[2, 3]  -> append      -> [2, 3, 7]
Step 7: num=101, tails=[2,3,7] -> append      -> [2, 3, 7, 101]
Step 8: num=18,  tails=[2,3,7,101] -> replace [3] -> [2, 3, 7, 18]
```

Final `len(tails) = 4`. The tails array `[2, 3, 7, 18]` happens to be a valid LIS here,
but this is coincidental -- in general tails does not represent an actual subsequence.

## Tips

- **The tails array does NOT store the actual LIS.** It stores the smallest possible tail
  for each subsequence length. To reconstruct the actual subsequence, track predecessor
  indices alongside the tails array.
- **Strictly increasing means `<`, not `<=`.** Use `bisect_left` (not `bisect_right`) to
  correctly handle duplicates -- replacing an equal element maintains the strictly
  increasing invariant.
- **The patience sort name** comes from the card game Patience (Solitaire). Each pile's top
  card corresponds to an element in the tails array.
- **For the O(n^2) approach**, initializing all `dp[i] = 1` is essential -- every element
  is itself a subsequence of length 1.
- **Go's `sort.SearchInts`** is equivalent to `bisect_left` in Python -- it finds the
  leftmost position where the target could be inserted while maintaining sorted order.
- **Rust's `partition_point`** with `|&x| x < num` gives the leftmost index where
  `tails[pos] >= num`, which is the equivalent of `bisect_left`.
- **To reconstruct the actual LIS**, maintain a parallel array `parent[i]` recording which
  predecessor index was used for each element. Then trace backwards from the index with
  the maximum dp value.
- **Non-strictly increasing (<=):** For the variant allowing equal consecutive elements,
  use `bisect_right` instead of `bisect_left` in the patience approach.

## See Also

- dynamic-programming
- binary-search
- greedy-algorithms
- patience-sort

## References

- [LeetCode 300 -- Longest Increasing Subsequence](https://leetcode.com/problems/longest-increasing-subsequence/)
- [Patience Sorting (Wikipedia)](https://en.wikipedia.org/wiki/Patience_sorting)
- [LeetCode 354 -- Russian Doll Envelopes (2D LIS extension)](https://leetcode.com/problems/russian-doll-envelopes/)
