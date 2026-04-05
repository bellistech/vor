# Two Sum (Arrays / Hash Map)

Given an array of integers and a target sum, find the two indices whose values add up to the target using a hash map for O(n) lookup.

## Problem

Given an integer array `nums` and an integer `target`, return the indices of the two numbers
such that they add up to `target`. Each input has exactly one solution, and you may not use
the same element twice. You can return the answer in any order.

**Constraints:**

- `2 <= nums.length <= 10^4`
- `-10^9 <= nums[i] <= 10^9`
- `-10^9 <= target <= 10^9`
- Exactly one valid answer exists.

**Example 1:**

```
Input:  nums = [2, 7, 11, 15], target = 9
Output: [0, 1]

Explanation: nums[0] + nums[1] = 2 + 7 = 9
```

**Example 2:**

```
Input:  nums = [3, 2, 4], target = 6
Output: [1, 2]
```

**Example 3:**

```
Input:  nums = [3, 3], target = 6
Output: [0, 1]
```

## Hints

- For each element `nums[i]`, compute `complement = target - nums[i]` and check if the complement has been seen before.
- Use a hash map mapping each value to its index. This turns the O(n) inner search of the brute-force approach into O(1).
- Process elements left to right, checking the map *before* inserting the current element to avoid using the same index twice.

## Solution -- Go

```go
package main

import "fmt"

func twoSum(nums []int, target int) []int {
	seen := make(map[int]int) // value -> index

	for i, num := range nums {
		complement := target - num
		if j, ok := seen[complement]; ok {
			return []int{j, i}
		}
		seen[num] = i
	}

	return nil // no solution found
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
		target   int
		expected []int
	}{
		{[]int{2, 7, 11, 15}, 9, []int{0, 1}},
		{[]int{3, 2, 4}, 6, []int{1, 2}},
		{[]int{3, 3}, 6, []int{0, 1}},
		{[]int{1, 5, 3, 7}, 8, []int{1, 2}},
		{[]int{-1, -2, -3, -4, -5}, -8, []int{2, 4}},
		{[]int{0, 4, 3, 0}, 0, []int{0, 3}},
	}

	for i, tc := range tests {
		got := twoSum(tc.nums, tc.target)
		if !sliceEqual(got, tc.expected) {
			panic(fmt.Sprintf("Test %d FAILED: got %v, want %v", i, got, tc.expected))
		}
	}
	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from typing import List


class Solution:
    def two_sum(self, nums: List[int], target: int) -> List[int]:
        seen: dict[int, int] = {}  # value -> index

        for i, num in enumerate(nums):
            complement = target - num
            if complement in seen:
                return [seen[complement], i]
            seen[num] = i

        return []  # no solution found


if __name__ == "__main__":
    s = Solution()

    # Basic test
    assert s.two_sum([2, 7, 11, 15], 9) == [0, 1]

    # Non-adjacent pair
    assert s.two_sum([3, 2, 4], 6) == [1, 2]

    # Duplicate values
    assert s.two_sum([3, 3], 6) == [0, 1]

    # Mid-array pair
    assert s.two_sum([1, 5, 3, 7], 8) == [1, 2]

    # Negative numbers
    assert s.two_sum([-1, -2, -3, -4, -5], -8) == [2, 4]

    # Zeros
    assert s.two_sum([0, 4, 3, 0], 0) == [0, 3]

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::HashMap;

struct Solution;

impl Solution {
    fn two_sum(nums: &[i32], target: i32) -> Vec<usize> {
        let mut seen: HashMap<i32, usize> = HashMap::new();

        for (i, &num) in nums.iter().enumerate() {
            let complement = target - num;
            if let Some(&j) = seen.get(&complement) {
                return vec![j, i];
            }
            seen.insert(num, i);
        }

        vec![] // no solution found
    }
}

fn main() {
    assert_eq!(Solution::two_sum(&[2, 7, 11, 15], 9), vec![0, 1]);
    assert_eq!(Solution::two_sum(&[3, 2, 4], 6), vec![1, 2]);
    assert_eq!(Solution::two_sum(&[3, 3], 6), vec![0, 1]);
    assert_eq!(Solution::two_sum(&[1, 5, 3, 7], 8), vec![1, 2]);
    assert_eq!(Solution::two_sum(&[-1, -2, -3, -4, -5], -8), vec![2, 4]);
    assert_eq!(Solution::two_sum(&[0, 4, 3, 0], 0), vec![0, 3]);
    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function twoSum(nums: number[], target: number): number[] {
    const seen = new Map<number, number>(); // value -> index

    for (let i = 0; i < nums.length; i++) {
        const complement = target - nums[i];
        const j = seen.get(complement);
        if (j !== undefined) {
            return [j, i];
        }
        seen.set(nums[i], i);
    }

    return []; // no solution found
}

// Tests
function arraysEqual(a: number[], b: number[]): boolean {
    return a.length === b.length && a.every((v, i) => v === b[i]);
}

console.assert(arraysEqual(twoSum([2, 7, 11, 15], 9), [0, 1]), "Test 1 failed");
console.assert(arraysEqual(twoSum([3, 2, 4], 6), [1, 2]), "Test 2 failed");
console.assert(arraysEqual(twoSum([3, 3], 6), [0, 1]), "Test 3 failed");
console.assert(arraysEqual(twoSum([1, 5, 3, 7], 8), [1, 2]), "Test 4 failed");
console.assert(arraysEqual(twoSum([-1, -2, -3, -4, -5], -8), [2, 4]), "Test 5 failed");
console.assert(arraysEqual(twoSum([0, 4, 3, 0], 0), [0, 3]), "Test 6 failed");
console.log("All tests passed!");
```

## Complexity

| Metric | Value |
|--------|-------|
| Time   | O(n) -- single pass through the array with O(1) hash map lookups |
| Space  | O(n) -- the hash map stores at most n entries |

## Tips

- The brute-force approach uses two nested loops for O(n^2) time. The hash map eliminates the inner loop entirely.
- Check the map *before* inserting the current element. This naturally prevents using the same index twice and handles duplicate values correctly (e.g., `[3, 3]` with target `6`).
- If the problem asked for *values* instead of indices, you could sort the array and use two pointers for O(n log n) time and O(1) space.
- This pattern generalizes: Three Sum reduces to iterating one element and running Two Sum on the remainder.

## See Also

- **Three Sum** -- extends Two Sum with an outer loop and two-pointer inner search.
- **Two Sum II (Sorted Array)** -- two-pointer O(n) approach when input is pre-sorted.
- **Contains Duplicate** -- another hash-set membership check pattern.
- **Subarray Sum Equals K** -- prefix-sum + hash map variant of the same idea.

## References

- [LeetCode 1 -- Two Sum](https://leetcode.com/problems/two-sum/)
- [NeetCode Explanation](https://neetcode.io/problems/two-integer-sum)
