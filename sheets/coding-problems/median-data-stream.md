# Median from Data Stream (Two Heaps)

Maintain the running median of a stream in O(log n) per insert and O(1) per query using a max-heap for the lower half and a min-heap for the upper half.

## Problem

Design a data structure that supports the following operations on a stream of integers:

- `addNum(num)` — add an integer from the data stream.
- `findMedian()` — return the current median of all elements added so far.

The median is the middle value in an ordered list. If the count is even, the median is the average of the two middle values.

**Constraints:**

- `-10^5 <= num <= 10^5`
- There will be at least one element before `findMedian` is called.
- At most `5 * 10^4` total calls to `addNum` and `findMedian`.

**Example:**

```
MedianFinder mf = new MedianFinder()
mf.addNum(1)          // [1]
mf.addNum(2)          // [1, 2]
mf.findMedian() -> 1.5
mf.addNum(3)          // [1, 2, 3]
mf.findMedian() -> 2.0
```

## Hints

1. A naive sorted-list approach is O(n) per insert. A balanced BST is O(log n) per insert but awkward to implement.
2. Observe: the median is determined by only the **two middle elements**. Maintain the **lower half** and the **upper half** separately.
3. Use a **max-heap** for the lower half (top = largest of the smaller half) and a **min-heap** for the upper half (top = smallest of the larger half).
4. Maintain the invariant `|lower| == |upper|` or `|lower| == |upper| + 1`. Median is `lower.top()` if odd count, or `(lower.top() + upper.top()) / 2` if even.
5. On insert: push to the appropriate heap based on comparison with the lower heap's top, then rebalance by moving one element across if sizes differ by more than 1.

## Solution -- Go

```go
package main

import (
	"container/heap"
	"fmt"
)

// maxHeap: largest on top
type maxHeap []int

func (h maxHeap) Len() int            { return len(h) }
func (h maxHeap) Less(i, j int) bool  { return h[i] > h[j] }
func (h maxHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *maxHeap) Push(x interface{}) { *h = append(*h, x.(int)) }
func (h *maxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// minHeap: smallest on top
type minHeap []int

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i] < h[j] }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(int)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type MedianFinder struct {
	lower *maxHeap // lower half
	upper *minHeap // upper half
}

func NewMedianFinder() *MedianFinder {
	lo := &maxHeap{}
	hi := &minHeap{}
	heap.Init(lo)
	heap.Init(hi)
	return &MedianFinder{lower: lo, upper: hi}
}

func (m *MedianFinder) AddNum(num int) {
	if m.lower.Len() == 0 || num <= (*m.lower)[0] {
		heap.Push(m.lower, num)
	} else {
		heap.Push(m.upper, num)
	}
	// Rebalance: |lower| must be equal to |upper| or one larger
	if m.lower.Len() > m.upper.Len()+1 {
		heap.Push(m.upper, heap.Pop(m.lower))
	} else if m.upper.Len() > m.lower.Len() {
		heap.Push(m.lower, heap.Pop(m.upper))
	}
}

func (m *MedianFinder) FindMedian() float64 {
	if m.lower.Len() > m.upper.Len() {
		return float64((*m.lower)[0])
	}
	return (float64((*m.lower)[0]) + float64((*m.upper)[0])) / 2.0
}

func main() {
	mf := NewMedianFinder()
	mf.AddNum(1)
	mf.AddNum(2)
	if got := mf.FindMedian(); got != 1.5 {
		panic(fmt.Sprintf("Test 1 FAILED: got %v", got))
	}
	mf.AddNum(3)
	if got := mf.FindMedian(); got != 2.0 {
		panic(fmt.Sprintf("Test 2 FAILED: got %v", got))
	}

	// Monotonically increasing
	mf2 := NewMedianFinder()
	for i := 1; i <= 5; i++ {
		mf2.AddNum(i)
	}
	if got := mf2.FindMedian(); got != 3.0 {
		panic(fmt.Sprintf("Test 3 FAILED: got %v", got))
	}

	// Monotonically decreasing
	mf3 := NewMedianFinder()
	for i := 5; i >= 1; i-- {
		mf3.AddNum(i)
	}
	if got := mf3.FindMedian(); got != 3.0 {
		panic(fmt.Sprintf("Test 4 FAILED: got %v", got))
	}

	// Duplicates
	mf4 := NewMedianFinder()
	for _, v := range []int{2, 2, 2, 2, 2} {
		mf4.AddNum(v)
	}
	if got := mf4.FindMedian(); got != 2.0 {
		panic(fmt.Sprintf("Test 5 FAILED: got %v", got))
	}

	// Negatives
	mf5 := NewMedianFinder()
	for _, v := range []int{-5, -3, -1, 0, 1, 3, 5} {
		mf5.AddNum(v)
	}
	if got := mf5.FindMedian(); got != 0.0 {
		panic(fmt.Sprintf("Test 6 FAILED: got %v", got))
	}

	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
import heapq


class MedianFinder:
    def __init__(self):
        self.lower: list = []  # max-heap (store negated)
        self.upper: list = []  # min-heap

    def add_num(self, num: int) -> None:
        if not self.lower or num <= -self.lower[0]:
            heapq.heappush(self.lower, -num)
        else:
            heapq.heappush(self.upper, num)

        # Rebalance
        if len(self.lower) > len(self.upper) + 1:
            heapq.heappush(self.upper, -heapq.heappop(self.lower))
        elif len(self.upper) > len(self.lower):
            heapq.heappush(self.lower, -heapq.heappop(self.upper))

    def find_median(self) -> float:
        if len(self.lower) > len(self.upper):
            return float(-self.lower[0])
        return (-self.lower[0] + self.upper[0]) / 2.0


if __name__ == "__main__":
    mf = MedianFinder()
    mf.add_num(1)
    mf.add_num(2)
    assert mf.find_median() == 1.5, "Test 1"
    mf.add_num(3)
    assert mf.find_median() == 2.0, "Test 2"

    mf2 = MedianFinder()
    for i in range(1, 6):
        mf2.add_num(i)
    assert mf2.find_median() == 3.0, "Test 3"

    mf3 = MedianFinder()
    for i in range(5, 0, -1):
        mf3.add_num(i)
    assert mf3.find_median() == 3.0, "Test 4"

    mf4 = MedianFinder()
    for v in [2, 2, 2, 2, 2]:
        mf4.add_num(v)
    assert mf4.find_median() == 2.0, "Test 5"

    mf5 = MedianFinder()
    for v in [-5, -3, -1, 0, 1, 3, 5]:
        mf5.add_num(v)
    assert mf5.find_median() == 0.0, "Test 6"

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::BinaryHeap;
use std::cmp::Reverse;

struct MedianFinder {
    lower: BinaryHeap<i32>,             // max-heap
    upper: BinaryHeap<Reverse<i32>>,    // min-heap via Reverse wrapper
}

impl MedianFinder {
    fn new() -> Self {
        Self {
            lower: BinaryHeap::new(),
            upper: BinaryHeap::new(),
        }
    }

    fn add_num(&mut self, num: i32) {
        if self.lower.is_empty() || num <= *self.lower.peek().unwrap() {
            self.lower.push(num);
        } else {
            self.upper.push(Reverse(num));
        }

        if self.lower.len() > self.upper.len() + 1 {
            let top = self.lower.pop().unwrap();
            self.upper.push(Reverse(top));
        } else if self.upper.len() > self.lower.len() {
            let Reverse(top) = self.upper.pop().unwrap();
            self.lower.push(top);
        }
    }

    fn find_median(&self) -> f64 {
        if self.lower.len() > self.upper.len() {
            *self.lower.peek().unwrap() as f64
        } else {
            let lo = *self.lower.peek().unwrap() as f64;
            let Reverse(hi) = *self.upper.peek().unwrap();
            (lo + hi as f64) / 2.0
        }
    }
}

fn main() {
    let mut mf = MedianFinder::new();
    mf.add_num(1);
    mf.add_num(2);
    assert_eq!(mf.find_median(), 1.5);
    mf.add_num(3);
    assert_eq!(mf.find_median(), 2.0);

    let mut mf2 = MedianFinder::new();
    for i in 1..=5 {
        mf2.add_num(i);
    }
    assert_eq!(mf2.find_median(), 3.0);

    let mut mf3 = MedianFinder::new();
    for i in (1..=5).rev() {
        mf3.add_num(i);
    }
    assert_eq!(mf3.find_median(), 3.0);

    let mut mf4 = MedianFinder::new();
    for v in [2, 2, 2, 2, 2] {
        mf4.add_num(v);
    }
    assert_eq!(mf4.find_median(), 2.0);

    let mut mf5 = MedianFinder::new();
    for v in [-5, -3, -1, 0, 1, 3, 5] {
        mf5.add_num(v);
    }
    assert_eq!(mf5.find_median(), 0.0);

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
// Minimal binary-heap utility (generic)
class Heap<T> {
    private arr: T[] = [];
    constructor(private cmp: (a: T, b: T) => number) {}
    size(): number { return this.arr.length; }
    peek(): T { return this.arr[0]; }
    push(x: T): void {
        this.arr.push(x);
        this.siftUp(this.arr.length - 1);
    }
    pop(): T {
        const top = this.arr[0];
        const last = this.arr.pop()!;
        if (this.arr.length > 0) {
            this.arr[0] = last;
            this.siftDown(0);
        }
        return top;
    }
    private siftUp(i: number): void {
        while (i > 0) {
            const p = (i - 1) >> 1;
            if (this.cmp(this.arr[i], this.arr[p]) < 0) {
                [this.arr[i], this.arr[p]] = [this.arr[p], this.arr[i]];
                i = p;
            } else break;
        }
    }
    private siftDown(i: number): void {
        const n = this.arr.length;
        while (true) {
            let smallest = i;
            const l = 2 * i + 1, r = 2 * i + 2;
            if (l < n && this.cmp(this.arr[l], this.arr[smallest]) < 0) smallest = l;
            if (r < n && this.cmp(this.arr[r], this.arr[smallest]) < 0) smallest = r;
            if (smallest === i) break;
            [this.arr[i], this.arr[smallest]] = [this.arr[smallest], this.arr[i]];
            i = smallest;
        }
    }
}

class MedianFinder {
    private lower = new Heap<number>((a, b) => b - a); // max-heap
    private upper = new Heap<number>((a, b) => a - b); // min-heap

    addNum(num: number): void {
        if (this.lower.size() === 0 || num <= this.lower.peek()) {
            this.lower.push(num);
        } else {
            this.upper.push(num);
        }
        if (this.lower.size() > this.upper.size() + 1) {
            this.upper.push(this.lower.pop());
        } else if (this.upper.size() > this.lower.size()) {
            this.lower.push(this.upper.pop());
        }
    }

    findMedian(): number {
        if (this.lower.size() > this.upper.size()) {
            return this.lower.peek();
        }
        return (this.lower.peek() + this.upper.peek()) / 2;
    }
}

// Tests
const mf = new MedianFinder();
mf.addNum(1);
mf.addNum(2);
console.assert(mf.findMedian() === 1.5, "T1");
mf.addNum(3);
console.assert(mf.findMedian() === 2.0, "T2");

const mf2 = new MedianFinder();
for (let i = 1; i <= 5; i++) mf2.addNum(i);
console.assert(mf2.findMedian() === 3.0, "T3");

const mf3 = new MedianFinder();
for (let i = 5; i >= 1; i--) mf3.addNum(i);
console.assert(mf3.findMedian() === 3.0, "T4");

const mf5 = new MedianFinder();
for (const v of [-5, -3, -1, 0, 1, 3, 5]) mf5.addNum(v);
console.assert(mf5.findMedian() === 0.0, "T5");

console.log("All tests passed!");
```

## Complexity

| Operation      | Time       | Space |
|----------------|------------|-------|
| `addNum`       | O(log n)   | --    |
| `findMedian`   | O(1)       | --    |
| Overall space  | --         | O(n)  |

- Each `addNum` performs up to 3 heap operations, each O(log n).
- `findMedian` peeks at the top of one or both heaps — O(1).

## Tips

- **Size invariant**: enforce `|lower| ∈ {|upper|, |upper| + 1}`. Lower is allowed one extra when the count is odd so the median is simply its top.
- **Push-then-rebalance** is cleaner than conditional-push. Always push to the side decided by comparison, then rebalance in a single conditional block.
- **Python's `heapq` is min-only** — negate values for the max-heap. Don't forget to negate back when reading.
- **Rust's `BinaryHeap` is max-only** — wrap with `Reverse<T>` for a min-heap.
- For **sliding-window median** (LC 480), augment with a "lazy deletion" technique or use an indexed skip list / order-statistic tree. Two heaps alone cannot remove arbitrary elements cheaply.
- For **approximate medians** over high-volume streams, consider **T-digest** (Dunning) or **reservoir sampling** — the exact two-heap solution assumes all data fits in memory.
- **Overflow** on even median: `(lower.top() + upper.top()) / 2` can overflow int range near `MAX/2`. Use `lower.top() / 2 + upper.top() / 2 + adjustment` pattern or cast to `float64` / `f64` before adding.
- **Ties matter for `<=` vs `<`** in the push decision. Using `<=` keeps duplicates on the lower side, maintaining a total order; `<` works too but changes which heap new equal values land in.

## See Also

- [Sliding Window Maximum](sliding-window-maximum.md) -- deque-based running stat over a window.
- [Merge K Sorted Lists](merge-k-sorted-lists.md) -- another priority-queue classic.
- [Kth Largest Element in Stream](kth-largest-stream.md) -- single-heap variant for extreme-order statistics.

## References

- LeetCode 295: Find Median from Data Stream
- LeetCode 480: Sliding Window Median (windowed variant)
- Dunning, *Computing Extremely Accurate Quantiles Using t-Digests* (2014) — approximate streaming alternative
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 6 (Heapsort)
