# Longest Consecutive Sequence (Arrays / Hash Sets)

Find the length of the longest consecutive elements sequence in an unsorted array in O(n) time using a hash set to identify and extend sequence starting points.

## Problem

Given an unsorted array of integers `nums`, return the length of the longest consecutive
elements sequence. The algorithm must run in O(n) time.

**Constraints:**

- `0 <= len(nums) <= 10^5`
- `-10^9 <= nums[i] <= 10^9`

**Examples:**

```
[100, 4, 200, 1, 3, 2]         => 4  (sequence [1, 2, 3, 4])
[0, 3, 7, 2, 5, 8, 4, 6, 0, 1] => 9  (sequence [0, 1, 2, 3, 4, 5, 6, 7, 8])
[]                              => 0  (empty array)
[1]                             => 1  (single element)
[1, 2, 3, 4, 5]                => 5  (already consecutive)
[1, 3, 5, 7]                   => 1  (no consecutive pair)
[1, 1, 1, 1]                   => 1  (duplicates collapse in set)
```

## Related Problems

| Problem | Key Difference | Approach |
|---------|---------------|----------|
| **Longest Consecutive Sequence** | **Unsorted, O(n) required** | **Hash set + sequence start detection** |
| Missing Number | 1 missing from [0..n] | XOR or arithmetic sum |
| Longest Increasing Subsequence | Non-contiguous, values increase | DP or patience sorting O(n log n) |
| Contains Duplicate II | Nearby duplicates | Sliding window + hash set |
| Array Nesting | Cycle length in index-following | Visited set, no sorting |

## Walkthrough

The key insight is that we only need to count from **sequence starting points** -- numbers
whose predecessor (`num - 1`) is not in the set. This avoids redundant work and ensures
each element is visited at most twice total.

Consider `nums = [100, 4, 200, 1, 3, 2]`:

**Step 1 -- Build hash set:** `{100, 4, 200, 1, 3, 2}`. This gives O(1) lookups.

**Step 2 -- Find sequence starts:** For each number, check if `num - 1` is in the set.
- `100`: is `99` in the set? No. Start of a sequence.
- `4`: is `3` in the set? Yes. Not a start -- skip.
- `200`: is `199` in the set? No. Start of a sequence.
- `1`: is `0` in the set? No. Start of a sequence.
- `3`: is `2` in the set? Yes. Not a start -- skip.
- `2`: is `1` in the set? Yes. Not a start -- skip.

**Step 3 -- Extend from starts:**
- From `100`: check `101`? No. Streak = 1.
- From `200`: check `201`? No. Streak = 1.
- From `1`: check `2`? Yes. Check `3`? Yes. Check `4`? Yes. Check `5`? No. Streak = 4.

**Result:** `max(1, 1, 4) = 4`.

## Hints

- **Core insight:** Put all numbers in a hash set for O(1) lookups. Then iterate over the
  set and only start counting from numbers that are the **beginning** of a sequence (i.e.,
  `num - 1` is not in the set).
- **Why this is O(n):** Each number is visited at most twice -- once during the outer
  iteration, and at most once as part of extending a sequence from its starting point.
  The "skip non-starts" check ensures we never re-traverse a sequence from the middle.
- **Duplicates:** The hash set naturally deduplicates. `[1, 1, 1, 1]` becomes `{1}`,
  giving a streak of 1.
- **Negative numbers:** The algorithm works identically for negative numbers. A sequence
  like `[-3, -2, -1, 0]` has start `-3` (since `-4` is absent) and streak 4.
- **Empty array:** Return 0. The set is empty, the loop body never executes.
- **Why not sort?** Sorting gives O(n log n). The hash set approach achieves O(n) by
  trading time for space -- O(n) extra memory for the set.
- **Alternative -- Union-Find:** A disjoint set union (DSU) approach can also solve this
  in O(n * alpha(n)) which is effectively O(n), but the hash set approach is simpler
  and has better constant factors.

## Solution -- Go

```go
package main

import "fmt"

func longestConsecutive(nums []int) int {
	numSet := make(map[int]bool, len(nums))
	for _, n := range nums {
		numSet[n] = true
	}

	longest := 0

	for num := range numSet {
		// Only start from sequence beginnings
		if !numSet[num-1] {
			current := num
			streak := 1

			for numSet[current+1] {
				current++
				streak++
			}

			if streak > longest {
				longest = streak
			}
		}
	}

	return longest
}

func main() {
	tests := []struct {
		nums     []int
		expected int
	}{
		{[]int{100, 4, 200, 1, 3, 2}, 4},
		{[]int{0, 3, 7, 2, 5, 8, 4, 6, 0, 1}, 9},
		{[]int{}, 0},
		{[]int{1}, 1},
		{[]int{1, 2, 3, 4, 5}, 5},
		{[]int{1, 3, 5, 7}, 1},
		{[]int{1, 1, 1, 1}, 1},
	}

	for i, tc := range tests {
		got := longestConsecutive(tc.nums)
		if got != tc.expected {
			panic(fmt.Sprintf("Test %d FAILED: got %d, want %d", i, got, tc.expected))
		}
	}
	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from typing import List


class Solution:
    def longest_consecutive(self, nums: List[int]) -> int:
        num_set = set(nums)
        longest = 0

        for num in num_set:
            # Only start counting from sequence beginnings
            if num - 1 not in num_set:
                current = num
                streak = 1

                while current + 1 in num_set:
                    current += 1
                    streak += 1

                longest = max(longest, streak)

        return longest


if __name__ == "__main__":
    s = Solution()

    assert s.longest_consecutive([100, 4, 200, 1, 3, 2]) == 4
    assert s.longest_consecutive([0, 3, 7, 2, 5, 8, 4, 6, 0, 1]) == 9
    assert s.longest_consecutive([]) == 0
    assert s.longest_consecutive([1]) == 1
    assert s.longest_consecutive([1, 2, 3, 4, 5]) == 5
    assert s.longest_consecutive([5, 4, 3, 2, 1]) == 5
    assert s.longest_consecutive([1, 3, 5, 7]) == 1
    assert s.longest_consecutive([1, 1, 1, 1]) == 1

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::HashSet;

struct Solution;

impl Solution {
    fn longest_consecutive(nums: &[i32]) -> i32 {
        let num_set: HashSet<i32> = nums.iter().copied().collect();
        let mut longest = 0;

        for &num in &num_set {
            // Only start from sequence beginnings
            if !num_set.contains(&(num - 1)) {
                let mut current = num;
                let mut streak = 1;

                while num_set.contains(&(current + 1)) {
                    current += 1;
                    streak += 1;
                }

                if streak > longest {
                    longest = streak;
                }
            }
        }

        longest
    }
}

fn main() {
    assert_eq!(Solution::longest_consecutive(&[100, 4, 200, 1, 3, 2]), 4);
    assert_eq!(
        Solution::longest_consecutive(&[0, 3, 7, 2, 5, 8, 4, 6, 0, 1]),
        9
    );
    assert_eq!(Solution::longest_consecutive(&[]), 0);
    assert_eq!(Solution::longest_consecutive(&[1]), 1);
    assert_eq!(Solution::longest_consecutive(&[1, 2, 3, 4, 5]), 5);
    assert_eq!(Solution::longest_consecutive(&[1, 3, 5, 7]), 1);
    assert_eq!(Solution::longest_consecutive(&[1, 1, 1, 1]), 1);

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function longestConsecutive(nums: number[]): number {
    const numSet = new Set(nums);
    let longest = 0;

    for (const num of numSet) {
        // Only start from sequence beginnings
        if (!numSet.has(num - 1)) {
            let current = num;
            let streak = 1;

            while (numSet.has(current + 1)) {
                current++;
                streak++;
            }

            longest = Math.max(longest, streak);
        }
    }

    return longest;
}

// Tests
console.assert(longestConsecutive([100, 4, 200, 1, 3, 2]) === 4, "Test 1 failed");
console.assert(longestConsecutive([0, 3, 7, 2, 5, 8, 4, 6, 0, 1]) === 9, "Test 2 failed");
console.assert(longestConsecutive([]) === 0, "Test 3 failed");
console.assert(longestConsecutive([1]) === 1, "Test 4 failed");
console.assert(longestConsecutive([1, 2, 3, 4, 5]) === 5, "Test 5 failed");
console.assert(longestConsecutive([1, 3, 5, 7]) === 1, "Test 6 failed");
console.assert(longestConsecutive([1, 1, 1, 1]) === 1, "Test 7 failed");
console.log("All tests passed!");
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(n) -- each element is visited at most twice (once in outer loop, once in inner extension) |
| Space | O(n) -- hash set stores all unique elements |
| Hash operations | O(n) -- n insertions + at most 2n lookups (one predecessor check + one successor check per element) |

## Tips

- **Why the "sequence start" check is critical:** Without the `num - 1 not in set` guard,
  you would start counting from every element, leading to O(n^2) worst case. For example,
  `[1, 2, 3, ..., n]` would trigger n sequences of lengths n, n-1, ..., 1. The guard
  ensures only one count per sequence.
- **Go map iteration order:** In Go, `map` iteration order is randomized. This does not
  affect correctness -- the algorithm finds the same longest streak regardless of iteration
  order. It does mean the sequence start that is processed first varies between runs.
- **Hash set vs. sorting trade-off:** Sorting gives O(n log n) time with O(1) extra space
  (or O(n) for merge sort). The hash set gives O(n) time but O(n) space. For very large
  arrays where memory is tight, sorting may be preferable despite the slower asymptotic time.
- **Python's `set` performance:** Python's `set` uses open addressing with random probing.
  Average-case lookup is O(1), but worst-case is O(n) with pathological hash collisions.
  For integer keys this is extremely unlikely.
- **Rust's `HashSet` note:** Rust's standard `HashSet` uses SipHash by default (DoS-resistant
  but slightly slower). For performance-critical code, consider `FxHashSet` from the
  `rustc-hash` crate, which uses a faster non-cryptographic hash.
- **Integer overflow in Rust:** The `num - 1` and `current + 1` operations could overflow
  for `i32::MIN` or `i32::MAX`. In practice the constraint `-10^9 <= nums[i] <= 10^9` is
  well within `i32` range, so `num - 1` and `current + 1` never overflow. For a fully
  defensive implementation, use `checked_sub` and `checked_add`.
- **TypeScript `Set` behavior:** JavaScript's `Set` preserves insertion order, so iteration
  order is deterministic (unlike Go maps). This does not affect correctness but can affect
  cache behavior in benchmarks.
- **Union-Find alternative:** Each number can be a node; when inserting `num`, union with
  `num - 1` and `num + 1` if they exist. The largest component size is the answer. This
  is O(n * alpha(n)) and more complex to implement, but useful when elements arrive
  as a stream.
- **Testing strategy:** Always test empty arrays, single elements, all-same elements
  (duplicates), fully consecutive ranges, and arrays with no consecutive pairs. These
  cover the main edge cases.
- **Real-world usage:** This pattern appears in time-series gap detection (finding the
  longest uninterrupted run of timestamps), genome sequencing (longest contiguous read
  alignment), and network analysis (longest chain of sequential packet IDs).

## See Also

- hash-set
- arrays
- union-find
- longest-increasing-subsequence
- contains-duplicate

## References

- [LeetCode 128 -- Longest Consecutive Sequence](https://leetcode.com/problems/longest-consecutive-sequence/)
- [Hash Table (Wikipedia)](https://en.wikipedia.org/wiki/Hash_table)
- [Disjoint-Set / Union-Find (Wikipedia)](https://en.wikipedia.org/wiki/Disjoint-set_data_structure)
- [Python set implementation (CPython)](https://github.com/python/cpython/blob/main/Objects/setobject.c)
- [Rust HashSet documentation](https://doc.rust-lang.org/std/collections/struct.HashSet.html)
