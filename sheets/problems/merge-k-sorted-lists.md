# Merge K Sorted Lists (Heap / Linked Lists)

Given k sorted linked lists, merge them into a single sorted linked list using a min-heap to repeatedly extract the smallest element in O(N log k) time.

## Problem

You are given an array of `k` linked lists, each sorted in ascending order.
Merge all the linked lists into one sorted linked list and return it.

**Constraints:**

- `k == len(lists)`
- `0 <= k <= 10^4`
- `0 <= lists[i].length <= 500`
- `-10^4 <= lists[i][j] <= 10^4`
- Each `lists[i]` is sorted in ascending order.
- The total number of nodes across all lists does not exceed `10^4`.

**Examples:**

```
Input:  [[1,4,5],[1,3,4],[2,6]]
Output: [1,1,2,3,4,4,5,6]

Input:  []
Output: []

Input:  [[]]
Output: []
```

## Hints

1. A brute-force approach collects all values and sorts them -- O(N log N). Can you do better?
2. Think about which data structure lets you efficiently find the minimum among k candidates.
3. You only need to compare the *heads* of the k lists at any given time. A min-heap of size k gives O(log k) extraction.
4. Use a dummy head node to simplify list construction; track a `current` tail pointer.
5. Each time you pop the minimum node from the heap, advance that list's pointer and push its next node (if any) back into the heap.
6. In Python, push `(value, index, node)` tuples to avoid comparing `ListNode` objects directly -- the index serves as a tiebreaker.

## Solution -- Go

```go
package main

import (
	"container/heap"
	"fmt"
)

type ListNode struct {
	Val  int
	Next *ListNode
}

// MinHeap of ListNode pointers
type NodeHeap []*ListNode

func (h NodeHeap) Len() int            { return len(h) }
func (h NodeHeap) Less(i, j int) bool  { return h[i].Val < h[j].Val }
func (h NodeHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *NodeHeap) Push(x interface{}) { *h = append(*h, x.(*ListNode)) }
func (h *NodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func mergeKLists(lists []*ListNode) *ListNode {
	h := &NodeHeap{}
	heap.Init(h)

	for _, node := range lists {
		if node != nil {
			heap.Push(h, node)
		}
	}

	dummy := &ListNode{}
	current := dummy

	for h.Len() > 0 {
		node := heap.Pop(h).(*ListNode)
		current.Next = node
		current = current.Next

		if node.Next != nil {
			heap.Push(h, node.Next)
		}
	}

	return dummy.Next
}

func buildList(vals []int) *ListNode {
	dummy := &ListNode{}
	curr := dummy
	for _, v := range vals {
		curr.Next = &ListNode{Val: v}
		curr = curr.Next
	}
	return dummy.Next
}

func listToSlice(head *ListNode) []int {
	var result []int
	for head != nil {
		result = append(result, head.Val)
		head = head.Next
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
	lists := []*ListNode{
		buildList([]int{1, 4, 5}),
		buildList([]int{1, 3, 4}),
		buildList([]int{2, 6}),
	}
	result := listToSlice(mergeKLists(lists))
	expected := []int{1, 1, 2, 3, 4, 4, 5, 6}
	if !sliceEqual(result, expected) {
		panic(fmt.Sprintf("Test 1 FAILED: got %v, want %v", result, expected))
	}

	if mergeKLists(nil) != nil {
		panic("Test 2 FAILED")
	}

	result = listToSlice(mergeKLists([]*ListNode{buildList([]int{1, 2, 3})}))
	if !sliceEqual(result, []int{1, 2, 3}) {
		panic(fmt.Sprintf("Test 3 FAILED: got %v", result))
	}

	lists = []*ListNode{buildList([]int{1, 3, 5}), buildList([]int{2, 4, 6})}
	result = listToSlice(mergeKLists(lists))
	if !sliceEqual(result, []int{1, 2, 3, 4, 5, 6}) {
		panic(fmt.Sprintf("Test 4 FAILED: got %v", result))
	}

	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
import heapq
from typing import List, Optional


class ListNode:
    def __init__(self, val: int = 0, next: Optional['ListNode'] = None):
        self.val = val
        self.next = next


class Solution:
    def merge_k_lists(self, lists: List[Optional[ListNode]]) -> Optional[ListNode]:
        # Min-heap: (value, list_index, node)
        # list_index is used as tiebreaker to avoid comparing ListNode objects
        heap: list = []

        for i, node in enumerate(lists):
            if node:
                heapq.heappush(heap, (node.val, i, node))

        dummy = ListNode(0)
        current = dummy

        while heap:
            val, idx, node = heapq.heappop(heap)
            current.next = node
            current = current.next

            if node.next:
                heapq.heappush(heap, (node.next.val, idx, node.next))

        return dummy.next


def build_list(vals: List[int]) -> Optional[ListNode]:
    dummy = ListNode(0)
    curr = dummy
    for v in vals:
        curr.next = ListNode(v)
        curr = curr.next
    return dummy.next


def list_to_array(head: Optional[ListNode]) -> List[int]:
    result = []
    while head:
        result.append(head.val)
        head = head.next
    return result


if __name__ == "__main__":
    s = Solution()

    lists = [build_list([1, 4, 5]), build_list([1, 3, 4]), build_list([2, 6])]
    result = list_to_array(s.merge_k_lists(lists))
    assert result == [1, 1, 2, 3, 4, 4, 5, 6], f"Test 1 failed: {result}"

    assert s.merge_k_lists([]) is None, "Test 2 failed"
    assert s.merge_k_lists([None]) is None, "Test 3 failed"

    result = list_to_array(s.merge_k_lists([build_list([1, 2, 3])]))
    assert result == [1, 2, 3], f"Test 4 failed: {result}"

    lists = [build_list([1, 3, 5]), build_list([2, 4, 6])]
    result = list_to_array(s.merge_k_lists(lists))
    assert result == [1, 2, 3, 4, 5, 6], f"Test 5 failed: {result}"

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::cmp::Reverse;
use std::collections::BinaryHeap;

#[derive(Debug, Clone)]
struct ListNode {
    val: i32,
    next: Option<Box<ListNode>>,
}

impl ListNode {
    fn new(val: i32) -> Self {
        ListNode { val, next: None }
    }
}

struct Solution;

impl Solution {
    fn merge_k_lists(lists: Vec<Option<Box<ListNode>>>) -> Option<Box<ListNode>> {
        // Min-heap: (value, list_index)
        // We also maintain a vector of current pointers
        let mut heads: Vec<Option<Box<ListNode>>> = lists;
        let mut heap: BinaryHeap<Reverse<(i32, usize)>> = BinaryHeap::new();

        // Initialize heap with head values
        for (i, head) in heads.iter().enumerate() {
            if let Some(node) = head {
                heap.push(Reverse((node.val, i)));
            }
        }

        let mut dummy = ListNode::new(0);
        let mut tail = &mut dummy;

        while let Some(Reverse((val, idx))) = heap.pop() {
            // Take the current head of list idx and advance it
            let mut node = heads[idx].take().unwrap();
            heads[idx] = node.next.take();

            // Push next element from same list if it exists
            if let Some(ref next_node) = heads[idx] {
                heap.push(Reverse((next_node.val, idx)));
            }

            // Append to result
            tail.next = Some(Box::new(ListNode::new(val)));
            tail = tail.next.as_mut().unwrap();
        }

        dummy.next
    }
}

fn build_list(vals: &[i32]) -> Option<Box<ListNode>> {
    let mut head: Option<Box<ListNode>> = None;
    for &v in vals.iter().rev() {
        let mut node = ListNode::new(v);
        node.next = head;
        head = Some(Box::new(node));
    }
    head
}

fn list_to_vec(mut head: Option<Box<ListNode>>) -> Vec<i32> {
    let mut result = Vec::new();
    while let Some(node) = head {
        result.push(node.val);
        head = node.next;
    }
    result
}

fn main() {
    let lists = vec![
        build_list(&[1, 4, 5]),
        build_list(&[1, 3, 4]),
        build_list(&[2, 6]),
    ];
    let result = list_to_vec(Solution::merge_k_lists(lists));
    assert_eq!(result, vec![1, 1, 2, 3, 4, 4, 5, 6]);

    let result = Solution::merge_k_lists(vec![]);
    assert!(result.is_none());

    let lists = vec![build_list(&[1, 2, 3])];
    let result = list_to_vec(Solution::merge_k_lists(lists));
    assert_eq!(result, vec![1, 2, 3]);

    let lists = vec![build_list(&[1, 3, 5]), build_list(&[2, 4, 6])];
    let result = list_to_vec(Solution::merge_k_lists(lists));
    assert_eq!(result, vec![1, 2, 3, 4, 5, 6]);

    let lists = vec![None, build_list(&[1, 2]), None];
    let result = list_to_vec(Solution::merge_k_lists(lists));
    assert_eq!(result, vec![1, 2]);

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
class ListNode {
    val: number;
    next: ListNode | null;
    constructor(val: number = 0, next: ListNode | null = null) {
        this.val = val;
        this.next = next;
    }
}

/** Simple min-heap for [value, listIndex] pairs */
class MinHeap {
    private data: [number, number][] = [];

    get size(): number {
        return this.data.length;
    }

    push(item: [number, number]): void {
        this.data.push(item);
        this.bubbleUp(this.data.length - 1);
    }

    pop(): [number, number] {
        const top = this.data[0];
        const last = this.data.pop()!;
        if (this.data.length > 0) {
            this.data[0] = last;
            this.sinkDown(0);
        }
        return top;
    }

    private bubbleUp(i: number): void {
        while (i > 0) {
            const parent = (i - 1) >> 1;
            if (this.data[parent][0] > this.data[i][0]) {
                [this.data[parent], this.data[i]] = [this.data[i], this.data[parent]];
                i = parent;
            } else {
                break;
            }
        }
    }

    private sinkDown(i: number): void {
        const n = this.data.length;
        while (true) {
            let smallest = i;
            const left = 2 * i + 1;
            const right = 2 * i + 2;
            if (left < n && this.data[left][0] < this.data[smallest][0]) smallest = left;
            if (right < n && this.data[right][0] < this.data[smallest][0]) smallest = right;
            if (smallest !== i) {
                [this.data[smallest], this.data[i]] = [this.data[i], this.data[smallest]];
                i = smallest;
            } else {
                break;
            }
        }
    }
}

function mergeKLists(lists: (ListNode | null)[]): ListNode | null {
    const heap = new MinHeap();
    const heads: (ListNode | null)[] = [...lists];

    // Initialize heap with head values
    for (let i = 0; i < heads.length; i++) {
        if (heads[i] !== null) {
            heap.push([heads[i]!.val, i]);
        }
    }

    const dummy = new ListNode(0);
    let current = dummy;

    while (heap.size > 0) {
        const [val, idx] = heap.pop();
        current.next = new ListNode(val);
        current = current.next;

        heads[idx] = heads[idx]!.next;
        if (heads[idx] !== null) {
            heap.push([heads[idx]!.val, idx]);
        }
    }

    return dummy.next;
}

// Helpers
function buildList(vals: number[]): ListNode | null {
    const dummy = new ListNode(0);
    let curr = dummy;
    for (const v of vals) {
        curr.next = new ListNode(v);
        curr = curr.next;
    }
    return dummy.next;
}

function listToArray(head: ListNode | null): number[] {
    const result: number[] = [];
    while (head) {
        result.push(head.val);
        head = head.next;
    }
    return result;
}

function arrEq(a: number[], b: number[]): boolean {
    return a.length === b.length && a.every((v, i) => v === b[i]);
}

// Tests
const lists1 = [buildList([1, 4, 5]), buildList([1, 3, 4]), buildList([2, 6])];
console.assert(
    arrEq(listToArray(mergeKLists(lists1)), [1, 1, 2, 3, 4, 4, 5, 6]),
    "Test 1 failed"
);

console.assert(mergeKLists([]) === null, "Test 2 failed");

console.assert(
    arrEq(listToArray(mergeKLists([buildList([1, 2, 3])])), [1, 2, 3]),
    "Test 3 failed"
);

const lists2 = [buildList([1, 3, 5]), buildList([2, 4, 6])];
console.assert(
    arrEq(listToArray(mergeKLists(lists2)), [1, 2, 3, 4, 5, 6]),
    "Test 4 failed"
);

console.log("All tests passed!");
```

## Complexity

| Metric | Value | Explanation |
|--------|-------|-------------|
| **Time** | O(N log k) | Each of the N total nodes is pushed and popped from a heap of size at most k. Each heap operation is O(log k). |
| **Space** | O(k) | The heap holds at most k nodes (one per list). The output list reuses existing nodes (or allocates N new ones depending on the implementation). |
| **Best case** | O(N log k) | Same as average -- every node must be processed regardless. |
| **When k = 1** | O(N) | The heap has size 1 so push/pop are O(1); degenerates to a simple traversal. |
| **When k = N** | O(N log N) | Each list has one element; equivalent to heap-sorting all elements. |

Where **N** = total number of nodes across all k lists.

## Tips

- **Dummy head pattern**: Always use a dummy/sentinel node when building a linked list. It eliminates special-case handling for the first insertion and lets you return `dummy.next` at the end.
- **Tiebreaker in Python**: `heapq` compares tuples element-by-element. If two nodes share the same value, Python will try to compare `ListNode` objects (which is undefined). Insert a unique index as the second tuple element to break ties deterministically.
- **Go's `container/heap`**: You must implement all five methods of the `heap.Interface` (`Len`, `Less`, `Swap`, `Push`, `Pop`). The `Push`/`Pop` methods operate on the underlying slice, not the heap -- always call `heap.Push(h, x)` and `heap.Pop(h)`, never `h.Push(x)`.
- **Rust ownership**: The borrow checker makes in-place linked list manipulation tricky. The `take()` pattern (take ownership, replace with `None`) is idiomatic for moving nodes between lists.
- **Alternative approaches**: Divide-and-conquer merging (merge lists pairwise in rounds) also achieves O(N log k) and avoids the heap entirely, but the heap approach is simpler to implement and reason about.
- **Edge cases to test**: empty input (`[]`), list of empty lists (`[None, None]`), single list, two lists, lists of vastly different lengths.

## See Also

- **Merge Two Sorted Lists** -- the k=2 base case; simpler variant using two pointers.
- **Sort List** -- merge sort on a single linked list; uses the same merge primitive.
- **Kth Smallest Element in a Sorted Matrix** -- another heap-based k-way selection problem.
- **Find K Pairs with Smallest Sums** -- min-heap over candidate pairs from two sorted arrays.
- **Ugly Number II** -- min-heap to generate the next element from multiple sorted sequences.

## References

- LeetCode 23: Merge k Sorted Lists (Hard)
- Cormen, Leiserson, Rivest, Stein. *Introduction to Algorithms* (CLRS), Ch. 6 "Heapsort" and Ch. 2.3.1 "Merging".
- Knuth, Donald E. *The Art of Computer Programming*, Vol. 3, Section 5.4.1 "Multiway Merging and Replacement Selection".
- Go standard library: `container/heap` -- https://pkg.go.dev/container/heap
- Python standard library: `heapq` -- https://docs.python.org/3/library/heapq.html
- Rust standard library: `std::collections::BinaryHeap` -- https://doc.rust-lang.org/std/collections/struct.BinaryHeap.html
