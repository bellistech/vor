# Binary Tree Level Order Traversal (Trees / BFS)

Given the root of a binary tree, return the level order traversal of its nodes' values (i.e., from left to right, level by level).

## Problem

Given the `root` of a binary tree, return its **level order traversal** as a nested list, where each inner list contains the values of nodes at that depth, from left to right.

**Constraints:**

- The number of nodes in the tree is in the range `[0, 2000]`.
- `-1000 <= Node.val <= 1000`

**Example:**

```
Input: root = [3,9,20,null,null,15,7]

        3
       / \
      9  20
         / \
        15   7

Output: [[3],[9,20],[15,7]]
```

**Example 2:**

```
Input: root = [1]
Output: [[1]]
```

**Example 3:**

```
Input: root = []
Output: []
```

## Hints

1. This is a classic **breadth-first search** (BFS) problem. BFS naturally visits nodes level by level.
2. Use a **queue** (FIFO). Start by enqueuing the root.
3. At each level, record the **current queue size** -- that tells you how many nodes belong to this level.
4. Dequeue exactly that many nodes, collect their values, and enqueue their children for the next level.
5. An empty tree (null root) should return an empty list.

## Solution -- Go

```go
package main

import "fmt"

type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

func levelOrder(root *TreeNode) [][]int {
	if root == nil {
		return nil
	}

	var result [][]int
	queue := []*TreeNode{root}

	for len(queue) > 0 {
		levelSize := len(queue)
		level := make([]int, 0, levelSize)

		for i := 0; i < levelSize; i++ {
			node := queue[0]
			queue = queue[1:]
			level = append(level, node.Val)

			if node.Left != nil {
				queue = append(queue, node.Left)
			}
			if node.Right != nil {
				queue = append(queue, node.Right)
			}
		}

		result = append(result, level)
	}

	return result
}

func main() {
	// Tree: [3,9,20,null,null,15,7]
	root := &TreeNode{
		Val:  3,
		Left: &TreeNode{Val: 9},
		Right: &TreeNode{
			Val:   20,
			Left:  &TreeNode{Val: 15},
			Right: &TreeNode{Val: 7},
		},
	}

	result := levelOrder(root)
	expected := [][]int{{3}, {9, 20}, {15, 7}}
	for i, level := range result {
		for j, v := range level {
			if v != expected[i][j] {
				panic(fmt.Sprintf("Test 1 FAILED at [%d][%d]: got %d, want %d", i, j, v, expected[i][j]))
			}
		}
	}

	// Single node
	single := &TreeNode{Val: 1}
	result2 := levelOrder(single)
	if len(result2) != 1 || result2[0][0] != 1 {
		panic("Test 2 FAILED: single node")
	}

	// Empty tree
	result3 := levelOrder(nil)
	if result3 != nil {
		panic("Test 3 FAILED: empty tree")
	}

	// Left-skewed tree: 1 -> 2 -> 3
	skewed := &TreeNode{Val: 1, Left: &TreeNode{Val: 2, Left: &TreeNode{Val: 3}}}
	result4 := levelOrder(skewed)
	if len(result4) != 3 || result4[0][0] != 1 || result4[1][0] != 2 || result4[2][0] != 3 {
		panic("Test 4 FAILED: skewed tree")
	}

	fmt.Println("All tests passed!")
}
```

## Solution -- Python

```python
from __future__ import annotations
from collections import deque


class TreeNode:
    def __init__(self, val: int = 0, left: TreeNode | None = None,
                 right: TreeNode | None = None):
        self.val = val
        self.left = left
        self.right = right


def level_order(root: TreeNode | None) -> list[list[int]]:
    if not root:
        return []

    result: list[list[int]] = []
    queue: deque[TreeNode] = deque([root])

    while queue:
        level_size = len(queue)
        level: list[int] = []

        for _ in range(level_size):
            node = queue.popleft()
            level.append(node.val)

            if node.left:
                queue.append(node.left)
            if node.right:
                queue.append(node.right)

        result.append(level)

    return result


if __name__ == "__main__":
    # Tree: [3,9,20,null,null,15,7]
    root = TreeNode(3, TreeNode(9), TreeNode(20, TreeNode(15), TreeNode(7)))
    assert level_order(root) == [[3], [9, 20], [15, 7]], "Test 1 failed"

    # Single node
    assert level_order(TreeNode(1)) == [[1]], "Test 2 failed"

    # Empty tree
    assert level_order(None) == [], "Test 3 failed"

    # Left-skewed tree
    skewed = TreeNode(1, TreeNode(2, TreeNode(3)))
    assert level_order(skewed) == [[1], [2], [3]], "Test 4 failed"

    # Complete binary tree
    complete = TreeNode(1, TreeNode(2, TreeNode(4), TreeNode(5)),
                        TreeNode(3, TreeNode(6), TreeNode(7)))
    assert level_order(complete) == [[1], [2, 3], [4, 5, 6, 7]], "Test 5 failed"

    print("All tests passed!")
```

## Solution -- Rust

```rust
use std::collections::VecDeque;

#[derive(Debug)]
struct TreeNode {
    val: i32,
    left: Option<Box<TreeNode>>,
    right: Option<Box<TreeNode>>,
}

impl TreeNode {
    fn new(val: i32) -> Self {
        TreeNode { val, left: None, right: None }
    }

    fn with_children(val: i32, left: Option<Box<TreeNode>>,
                     right: Option<Box<TreeNode>>) -> Self {
        TreeNode { val, left, right }
    }
}

fn level_order(root: Option<&TreeNode>) -> Vec<Vec<i32>> {
    let Some(root) = root else {
        return vec![];
    };

    let mut result = Vec::new();
    let mut queue = VecDeque::new();
    queue.push_back(root);

    while !queue.is_empty() {
        let level_size = queue.len();
        let mut level = Vec::with_capacity(level_size);

        for _ in 0..level_size {
            let node = queue.pop_front().unwrap();
            level.push(node.val);

            if let Some(ref left) = node.left {
                queue.push_back(left);
            }
            if let Some(ref right) = node.right {
                queue.push_back(right);
            }
        }

        result.push(level);
    }

    result
}

fn main() {
    // Tree: [3,9,20,null,null,15,7]
    let root = TreeNode::with_children(
        3,
        Some(Box::new(TreeNode::new(9))),
        Some(Box::new(TreeNode::with_children(
            20,
            Some(Box::new(TreeNode::new(15))),
            Some(Box::new(TreeNode::new(7))),
        ))),
    );
    assert_eq!(level_order(Some(&root)), vec![vec![3], vec![9, 20], vec![15, 7]]);

    // Single node
    let single = TreeNode::new(1);
    assert_eq!(level_order(Some(&single)), vec![vec![1]]);

    // Empty tree
    assert_eq!(level_order(None), Vec::<Vec<i32>>::new());

    // Left-skewed tree
    let skewed = TreeNode::with_children(
        1,
        Some(Box::new(TreeNode::with_children(
            2,
            Some(Box::new(TreeNode::new(3))),
            None,
        ))),
        None,
    );
    assert_eq!(level_order(Some(&skewed)), vec![vec![1], vec![2], vec![3]]);

    println!("All tests passed!");
}
```

## Solution -- TypeScript

```typescript
class TreeNode {
    val: number;
    left: TreeNode | null;
    right: TreeNode | null;

    constructor(val: number = 0, left: TreeNode | null = null,
                right: TreeNode | null = null) {
        this.val = val;
        this.left = left;
        this.right = right;
    }
}

function levelOrder(root: TreeNode | null): number[][] {
    if (!root) return [];

    const result: number[][] = [];
    const queue: TreeNode[] = [root];

    while (queue.length > 0) {
        const levelSize = queue.length;
        const level: number[] = [];

        for (let i = 0; i < levelSize; i++) {
            const node = queue.shift()!;
            level.push(node.val);

            if (node.left) queue.push(node.left);
            if (node.right) queue.push(node.right);
        }

        result.push(level);
    }

    return result;
}

// Tests
const root = new TreeNode(3,
    new TreeNode(9),
    new TreeNode(20, new TreeNode(15), new TreeNode(7))
);
const result = levelOrder(root);
console.assert(JSON.stringify(result) === "[[3],[9,20],[15,7]]", "Test 1 failed");

// Single node
const single = new TreeNode(1);
console.assert(JSON.stringify(levelOrder(single)) === "[[1]]", "Test 2 failed");

// Empty tree
console.assert(JSON.stringify(levelOrder(null)) === "[]", "Test 3 failed");

// Left-skewed
const skewed = new TreeNode(1, new TreeNode(2, new TreeNode(3)));
console.assert(JSON.stringify(levelOrder(skewed)) === "[[1],[2],[3]]", "Test 4 failed");

// Complete binary tree
const complete = new TreeNode(1,
    new TreeNode(2, new TreeNode(4), new TreeNode(5)),
    new TreeNode(3, new TreeNode(6), new TreeNode(7))
);
console.assert(
    JSON.stringify(levelOrder(complete)) === "[[1],[2,3],[4,5,6,7]]",
    "Test 5 failed"
);

console.log("All tests passed!");
```

## Complexity

| Aspect | Bound |
|--------|-------|
| Time   | O(n)  |
| Space  | O(n)  |

- **Time** is O(n) because every node is enqueued and dequeued exactly once.
- **Space** is O(n) for the queue and the result list. The queue holds at most one full level; in a complete binary tree the last level has ~n/2 nodes.

## Tips

- **BFS vs DFS:** Level order traversal is the canonical BFS problem on trees. DFS (with a depth parameter) can also solve it, but BFS with a queue is the natural and expected approach.
- **Track level size, not delimiters.** Recording `len(queue)` at the start of each level iteration is cleaner than using sentinel values or separate "next level" queues.
- **Edge case: empty tree.** Always check for a null root before enqueuing. Forgetting this is the most common bug.
- **Queue implementation matters in interviews.** In Go, slicing `queue[1:]` works but reallocates. In Python, use `collections.deque` for O(1) popleft. In TypeScript, `Array.shift()` is O(n); mention this trade-off.

## See Also

- [Binary Tree Zigzag Level Order](binary-tree-zigzag-level-order.md) -- alternate left-right ordering per level.
- [Binary Tree Right Side View](binary-tree-right-side-view.md) -- BFS keeping only the last node per level.
- [Maximum Depth of Binary Tree](maximum-depth-of-binary-tree.md) -- simpler tree traversal (DFS or BFS).

## References

- LeetCode 102: Binary Tree Level Order Traversal
- Cormen et al., *Introduction to Algorithms* (CLRS), Chapter 22 (Breadth-First Search)
- Knuth, *The Art of Computer Programming*, Volume 1, Section 2.3 (Trees)
