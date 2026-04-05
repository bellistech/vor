# Trapping Rain Water (Arrays / Two Pointers)

Compute how much water can be trapped between bars of an elevation map using a two-pointer approach for O(n) time and O(1) space.

## Problem

Given `n` non-negative integers representing an elevation map where the width of each bar is 1,
compute how much water it can trap after raining.

**Constraints:**

- `1 <= height.length <= 2 * 10^4`
- `0 <= height[i] <= 10^5`

**Example 1:**

```
Input:  height = [0, 1, 0, 2, 1, 0, 1, 3, 2, 1, 2, 1]
Output: 6

Elevation map:

       #
   # ~ ~ # ~ #
 # ~ # # ~ # # # ~ #
 0 1 0 2 1 0 1 3 2 1 2 1

Water trapped = 6 units (shown as ~)
```

**Example 2:**

```
Input:  height = [4, 2, 0, 3, 2, 5]
Output: 9
```

**Example 3:**

```
Input:  height = [2, 0, 2]
Output: 2
```

## Hints

- Water at any position is determined by the minimum of the tallest bar to its left and the tallest bar to its right, minus the bar's own height.
- Use two pointers starting from both ends. Track `left_max` and `right_max` as they move inward.
- Always advance the pointer with the smaller max -- the water at that side is bounded by the smaller max regardless of what lies further inward.

## Solution -- Go

```go
package main

import "fmt"

func trap(height []int) int {
	left, right := 0, len(height)-1
	leftMax, rightMax := 0, 0
	water := 0

	for left < right {
		if height[left] < height[right] {
			if height[left] >= leftMax {
				leftMax = height[left]
			} else {
				water += leftMax - height[left]
			}
			left++
		} else {
			if height[right] >= rightMax {
				rightMax = height[right]
			} else {
				water += rightMax - height[right]
			}
			right--
		}
	}

	return water
}

func main() {
	tests := []struct {
		height   []int
		expected int
	}{
		{[]int{0, 1, 0, 2, 1, 0, 1, 3, 2, 1, 2, 1}, 6},
		{[]int{4, 2, 0, 3, 2, 5}, 9},
		{[]int{2, 0, 2}, 2},
		{[]int{3, 0, 0, 2, 0, 4}, 10},
		{[]int{0}, 0},
		{[]int{1, 2, 3, 4, 5}, 0},
	}

	for i, tc := range tests {
		got := trap(tc.height)
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
    def trap(self, height: List[int]) -> int:
        left, right = 0, len(height) - 1
        left_max, right_max = 0, 0
        water = 0

        while left < right:
            if height[left] < height[right]:
                if height[left] >= left_max:
                    left_max = height[left]
                else:
                    water += left_max - height[left]
                left += 1
            else:
                if height[right] >= right_max:
                    right_max = height[right]
                else:
                    water += right_max - height[right]
                right -= 1

        return water


if __name__ == "__main__":
    s = Solution()

    # Classic example
    assert s.trap([0, 1, 0, 2, 1, 0, 1, 3, 2, 1, 2, 1]) == 6

    # Valley between two tall bars
    assert s.trap([4, 2, 0, 3, 2, 5]) == 9

    # Simple gap
    assert s.trap([2, 0, 2]) == 2

    # Deep valley
    assert s.trap([3, 0, 0, 2, 0, 4]) == 10

    # Single bar => no water
    assert s.trap([0]) == 0

    # Ascending => no water
    assert s.trap([1, 2, 3, 4, 5]) == 0

    print("All tests passed!")
```

## Solution -- Rust

```rust
struct Solution;

impl Solution {
    fn trap(height: &[i32]) -> i32 {
        let mut left: usize = 0;
        let mut right: usize = height.len() - 1;
        let mut left_max: i32 = 0;
        let mut right_max: i32 = 0;
        let mut water: i32 = 0;

        while left < right {
            if height[left] < height[right] {
                if height[left] >= left_max {
                    left_max = height[left];
                } else {
                    water += left_max - height[left];
                }
                left += 1;
            } else {
                if height[right] >= right_max {
                    right_max = height[right];
                } else {
                    water += right_max - height[right];
                }
                right -= 1;
            }
        }

        water
    }
}

fn main() {
    assert_eq!(
        Solution::trap(&[0, 1, 0, 2, 1, 0, 1, 3, 2, 1, 2, 1]),
        6
    );
    assert_eq!(Solution::trap(&[4, 2, 0, 3, 2, 5]), 9);
    assert_eq!(Solution::trap(&[2, 0, 2]), 2);
    assert_eq!(Solution::trap(&[3, 0, 0, 2, 0, 4]), 10);
    assert_eq!(Solution::trap(&[0]), 0);
    assert_eq!(Solution::trap(&[1, 2, 3, 4, 5]), 0);
    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function trap(height: number[]): number {
    let left = 0;
    let right = height.length - 1;
    let leftMax = 0;
    let rightMax = 0;
    let water = 0;

    while (left < right) {
        if (height[left] < height[right]) {
            if (height[left] >= leftMax) {
                leftMax = height[left];
            } else {
                water += leftMax - height[left];
            }
            left++;
        } else {
            if (height[right] >= rightMax) {
                rightMax = height[right];
            } else {
                water += rightMax - height[right];
            }
            right--;
        }
    }

    return water;
}

// Tests
console.assert(trap([0, 1, 0, 2, 1, 0, 1, 3, 2, 1, 2, 1]) === 6, "Test 1 failed");
console.assert(trap([4, 2, 0, 3, 2, 5]) === 9, "Test 2 failed");
console.assert(trap([2, 0, 2]) === 2, "Test 3 failed");
console.assert(trap([3, 0, 0, 2, 0, 4]) === 10, "Test 4 failed");
console.assert(trap([0]) === 0, "Test 5 failed");
console.assert(trap([1, 2, 3, 4, 5]) === 0, "Test 6 failed");
console.log("All tests passed!");
```

## Complexity

| Metric | Value |
|--------|-------|
| Time   | O(n) -- single pass with two pointers meeting in the middle |
| Space  | O(1) -- only a fixed number of variables regardless of input size |

## Tips

- The two-pointer technique works because water at any position is bounded by the **smaller** of the two maxima. By advancing the pointer with the smaller max, we guarantee the water calculation at that position is correct without needing to see the rest of the array.
- An alternative O(n) time, O(n) space approach precomputes prefix-max and suffix-max arrays, then water[i] = min(prefix_max[i], suffix_max[i]) - height[i].
- A stack-based approach processes bars left to right, computing trapped water layer by layer when a taller bar is encountered. Also O(n) time, O(n) space.
- Edge cases: empty array, single element, monotonically increasing/decreasing arrays all trap zero water.

## See Also

- **Container With Most Water** -- related two-pointer problem on maximizing water area between two bars.
- **Largest Rectangle in Histogram** -- stack-based approach on elevation-like problems.
- **Sliding Window Maximum** -- monotonic deque for window queries over arrays.

## References

- [LeetCode 42 -- Trapping Rain Water](https://leetcode.com/problems/trapping-rain-water/)
- [NeetCode Explanation](https://neetcode.io/problems/trapping-rain-water)
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 9 -- Medians and Order Statistics
