# Largest Rectangle in Histogram (Monotonic Stack)

Find the largest rectangular area in a histogram in O(n) time using a monotonic increasing stack that tracks candidate left boundaries.

## Problem

Given an array of non-negative integers `heights` representing a histogram where each bar has width 1, return the area of the largest rectangle that fits entirely within the histogram.

**Constraints:**

- `1 <= heights.length <= 10^5`
- `0 <= heights[i] <= 10^4`

**Example 1:**

```
Input:  heights = [2, 1, 5, 6, 2, 3]
Output: 10

Histogram:
          _ _
          | |
          | |_ _
       _ _| | |     maximum rectangle of height 5, width 2 = 10
       | | | |        (bars at indices 2 and 3)
   _   | | | |_ _
   |  _| | | | |
   | | | | | | |
    2 1 5 6 2 3
```

**Example 2:**

```
Input:  heights = [2, 4]
Output: 4
```

## Hints

1. For each bar, the largest rectangle containing it has height `heights[i]` and extends as far left/right as bars stay ≥ `heights[i]`.
2. Use a **monotonic increasing stack** of indices: while the incoming bar is shorter than the stack top, the top bar cannot extend rightward further — pop it and compute its rectangle.
3. When you pop bar `i` with the incoming index `r`, the right boundary is `r - 1` and the left boundary is `stack.top() + 1` (or `-1` if the stack is empty).
4. Width of the rectangle for popped bar at index `i` is `r - stack.top() - 1` after the pop, where `stack.top()` is the new top (the next smaller bar on the left).
5. Use a **sentinel bar of height 0** at the end to force all remaining bars to be popped and evaluated.

## Solution -- Go

```go
package main

import "fmt"

func largestRectangleArea(heights []int) int {
	stack := []int{} // indices of increasing bar heights
	maxArea := 0
	n := len(heights)

	for i := 0; i <= n; i++ {
		var h int
		if i == n {
			h = 0 // sentinel: flush stack
		} else {
			h = heights[i]
		}

		for len(stack) > 0 && heights[stack[len(stack)-1]] > h {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			height := heights[top]
			var width int
			if len(stack) == 0 {
				width = i
			} else {
				width = i - stack[len(stack)-1] - 1
			}

			if area := height * width; area > maxArea {
				maxArea = area
			}
		}
		stack = append(stack, i)
	}
	return maxArea
}

func main() {
	if got := largestRectangleArea([]int{2, 1, 5, 6, 2, 3}); got != 10 {
		panic(fmt.Sprintf("Test 1 FAILED: got %d", got))
	}
	if got := largestRectangleArea([]int{2, 4}); got != 4 {
		panic(fmt.Sprintf("Test 2 FAILED: got %d", got))
	}
	if got := largestRectangleArea([]int{0}); got != 0 {
		panic(fmt.Sprintf("Test 3 FAILED: got %d", got))
	}
	if got := largestRectangleArea([]int{1}); got != 1 {
		panic(fmt.Sprintf("Test 4 FAILED: got %d", got))
	}
	// Strictly increasing
	if got := largestRectangleArea([]int{1, 2, 3, 4, 5}); got != 9 {
		panic(fmt.Sprintf("Test 5 FAILED: got %d", got))
	}
	// Strictly decreasing
	if got := largestRectangleArea([]int{5, 4, 3, 2, 1}); got != 9 {
		panic(fmt.Sprintf("Test 6 FAILED: got %d", got))
	}
	// All equal
	if got := largestRectangleArea([]int{3, 3, 3, 3}); got != 12 {
		panic(fmt.Sprintf("Test 7 FAILED: got %d", got))
	}
	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from typing import List


def largest_rectangle_area(heights: List[int]) -> int:
    stack: List[int] = []  # indices of strictly increasing heights
    max_area = 0
    n = len(heights)

    for i in range(n + 1):
        h = 0 if i == n else heights[i]

        while stack and heights[stack[-1]] > h:
            top = stack.pop()
            height = heights[top]
            width = i if not stack else i - stack[-1] - 1
            max_area = max(max_area, height * width)

        stack.append(i)

    return max_area


if __name__ == "__main__":
    assert largest_rectangle_area([2, 1, 5, 6, 2, 3]) == 10, "Test 1"
    assert largest_rectangle_area([2, 4]) == 4, "Test 2"
    assert largest_rectangle_area([0]) == 0, "Test 3"
    assert largest_rectangle_area([1]) == 1, "Test 4"
    assert largest_rectangle_area([1, 2, 3, 4, 5]) == 9, "Test 5"
    assert largest_rectangle_area([5, 4, 3, 2, 1]) == 9, "Test 6"
    assert largest_rectangle_area([3, 3, 3, 3]) == 12, "Test 7"
    # Single tall bar flanked by zeros
    assert largest_rectangle_area([0, 5, 0]) == 5, "Test 8"
    # All zeros
    assert largest_rectangle_area([0, 0, 0]) == 0, "Test 9"
    print("All tests passed!")
```

## Solution -- Rust

```rust
fn largest_rectangle_area(heights: Vec<i32>) -> i32 {
    let mut stack: Vec<usize> = Vec::new();
    let mut max_area: i32 = 0;
    let n = heights.len();

    for i in 0..=n {
        let h = if i == n { 0 } else { heights[i] };

        while let Some(&top) = stack.last() {
            if heights[top] <= h {
                break;
            }
            stack.pop();
            let height = heights[top];
            let width = match stack.last() {
                None => i as i32,
                Some(&left) => (i - left - 1) as i32,
            };
            max_area = max_area.max(height * width);
        }
        stack.push(i);
    }
    max_area
}

fn main() {
    assert_eq!(largest_rectangle_area(vec![2, 1, 5, 6, 2, 3]), 10);
    assert_eq!(largest_rectangle_area(vec![2, 4]), 4);
    assert_eq!(largest_rectangle_area(vec![0]), 0);
    assert_eq!(largest_rectangle_area(vec![1]), 1);
    assert_eq!(largest_rectangle_area(vec![1, 2, 3, 4, 5]), 9);
    assert_eq!(largest_rectangle_area(vec![5, 4, 3, 2, 1]), 9);
    assert_eq!(largest_rectangle_area(vec![3, 3, 3, 3]), 12);
    assert_eq!(largest_rectangle_area(vec![0, 5, 0]), 5);
    assert_eq!(largest_rectangle_area(vec![0, 0, 0]), 0);
    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
function largestRectangleArea(heights: number[]): number {
    const stack: number[] = [];
    let maxArea = 0;
    const n = heights.length;

    for (let i = 0; i <= n; i++) {
        const h = i === n ? 0 : heights[i];

        while (stack.length > 0 && heights[stack[stack.length - 1]] > h) {
            const top = stack.pop()!;
            const height = heights[top];
            const width = stack.length === 0 ? i : i - stack[stack.length - 1] - 1;
            maxArea = Math.max(maxArea, height * width);
        }
        stack.push(i);
    }
    return maxArea;
}

console.assert(largestRectangleArea([2, 1, 5, 6, 2, 3]) === 10, "Test 1");
console.assert(largestRectangleArea([2, 4]) === 4, "Test 2");
console.assert(largestRectangleArea([0]) === 0, "Test 3");
console.assert(largestRectangleArea([1]) === 1, "Test 4");
console.assert(largestRectangleArea([1, 2, 3, 4, 5]) === 9, "Test 5");
console.assert(largestRectangleArea([5, 4, 3, 2, 1]) === 9, "Test 6");
console.assert(largestRectangleArea([3, 3, 3, 3]) === 12, "Test 7");
console.assert(largestRectangleArea([0, 5, 0]) === 5, "Test 8");
console.assert(largestRectangleArea([0, 0, 0]) === 0, "Test 9");
console.log("All tests passed!");
```

## Complexity

| Aspect | Bound |
|--------|-------|
| Time   | O(n) |
| Space  | O(n) |

- Each index is **pushed and popped exactly once**, giving linear total work even though inner `while` loops appear nested.
- Stack depth is bounded by the number of bars — O(n) worst case (strictly increasing heights).

## Tips

- The **sentinel zero bar** at the end is the cleanest trick. Without it, you need a second loop to drain the remaining monotonic stack, doubling the code surface.
- **Store indices, not heights** on the stack. You need the index to compute the width; the height you re-look-up via `heights[top]`.
- **Width computation** after pop: if the stack is empty, width = current index `i` (rectangle extends to the start). Otherwise, width = `i - stack.top() - 1` (bounded by the new stack top on the left and `i - 1` on the right).
- This problem powers the **Maximal Rectangle in Binary Matrix** (LC 85): treat each row as a histogram where height accumulates consecutive 1s, and apply this algorithm per row for an overall O(m·n) solution.
- Related **monotonic stack** problems: Next Greater Element (LC 496), Daily Temperatures (LC 739), Sliding Window Maximum (deque variant), Trapping Rain Water (also solvable with stack). The pattern is: maintain a stack of indices where `heights[i]` is monotone, pop on violation, do bounded work per pop.
- The divide-and-conquer O(n log n) solution exists but is strictly inferior: it recurses on the minimum bar, splits, and combines. Monotonic stack is asymptotically and practically faster.

## See Also

- [Maximal Rectangle](maximal-rectangle.md) -- apply this algorithm row-by-row on a binary matrix.
- [Sliding Window Maximum](sliding-window-maximum.md) -- another monotonic deque technique.
- [Trapping Rain Water](trapping-rain-water.md) -- monotonic stack solves this too, though two-pointers is the O(1) space winner.
- [Daily Temperatures](daily-temperatures.md) -- the cleanest introductory monotonic stack problem.

## References

- LeetCode 84: Largest Rectangle in Histogram
- LeetCode 85: Maximal Rectangle (two-dimensional generalisation)
- Sedgewick & Wayne, *Algorithms*, 4th ed., Section 1.3 (Stacks and queues)
