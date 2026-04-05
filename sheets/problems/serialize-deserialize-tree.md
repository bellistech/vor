# Serialize and Deserialize Binary Tree (Trees / DFS)

Design an algorithm to serialize a binary tree to a string and deserialize it back to the original tree structure.

## Problem

Given a binary tree, implement two functions:

1. **serialize(root)** -- Encode the tree into a single string.
2. **deserialize(data)** -- Decode the string back into the original tree.

The serialization format is not prescribed -- you may choose any scheme as long as
`deserialize(serialize(tree))` produces a structurally identical tree.

**Constraints:**

- Number of nodes: `[0, 10^4]`
- `-1000 <= Node.val <= 1000`

**Examples:**

```
Tree:       1
           / \
          2   3
             / \
            4   5

Serialized (preorder): "1,2,null,null,3,4,null,null,5,null,null"
```

- Serialize then deserialize the above tree returns an identical tree.
- `serialize(null)` returns `"null"`.
- `deserialize("null")` returns `null`.

## Hints

- Use **preorder DFS** traversal: record each node's value, and write `"null"` for
  every missing child. Separate tokens with commas.
- To deserialize, split the string by commas and consume tokens sequentially with a
  recursive function -- the preorder structure tells you exactly when to recurse left
  and right.
- An alternative BFS (level-order) approach also works: enqueue children (including
  nulls) and serialize level by level.

## Solution -- Go

```go
import (
	"strconv"
	"strings"
)

type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

// Serialize encodes a tree to a single string using preorder DFS.
func Serialize(root *TreeNode) string {
	var parts []string
	var dfs func(node *TreeNode)
	dfs = func(node *TreeNode) {
		if node == nil {
			parts = append(parts, "null")
			return
		}
		parts = append(parts, strconv.Itoa(node.Val))
		dfs(node.Left)
		dfs(node.Right)
	}
	dfs(root)
	return strings.Join(parts, ",")
}

// Deserialize decodes a string back to a tree.
func Deserialize(data string) *TreeNode {
	tokens := strings.Split(data, ",")
	idx := 0

	var dfs func() *TreeNode
	dfs = func() *TreeNode {
		if idx >= len(tokens) || tokens[idx] == "null" {
			idx++
			return nil
		}
		val, _ := strconv.Atoi(tokens[idx])
		idx++
		node := &TreeNode{Val: val}
		node.Left = dfs()
		node.Right = dfs()
		return node
	}

	return dfs()
}
```

## Solution -- Python

```python
from typing import Optional, List
from collections import deque


class TreeNode:
    def __init__(self, val: int = 0, left: Optional['TreeNode'] = None,
                 right: Optional['TreeNode'] = None):
        self.val = val
        self.left = left
        self.right = right


class Codec:
    """Preorder DFS approach."""

    def serialize(self, root: Optional[TreeNode]) -> str:
        """Encode a tree to a single string."""
        parts: List[str] = []

        def dfs(node: Optional[TreeNode]) -> None:
            if node is None:
                parts.append("null")
                return
            parts.append(str(node.val))
            dfs(node.left)
            dfs(node.right)

        dfs(root)
        return ",".join(parts)

    def deserialize(self, data: str) -> Optional[TreeNode]:
        """Decode a string back to a tree."""
        tokens = iter(data.split(","))

        def dfs() -> Optional[TreeNode]:
            val = next(tokens)
            if val == "null":
                return None
            node = TreeNode(int(val))
            node.left = dfs()
            node.right = dfs()
            return node

        return dfs()


class CodecBFS:
    """BFS (level-order) approach."""

    def serialize(self, root: Optional[TreeNode]) -> str:
        if root is None:
            return "null"
        parts: List[str] = []
        queue: deque = deque([root])
        while queue:
            node = queue.popleft()
            if node is None:
                parts.append("null")
            else:
                parts.append(str(node.val))
                queue.append(node.left)
                queue.append(node.right)
        return ",".join(parts)

    def deserialize(self, data: str) -> Optional[TreeNode]:
        tokens = data.split(",")
        if tokens[0] == "null":
            return None
        root = TreeNode(int(tokens[0]))
        queue: deque = deque([root])
        i = 1
        while queue and i < len(tokens):
            node = queue.popleft()
            if tokens[i] != "null":
                node.left = TreeNode(int(tokens[i]))
                queue.append(node.left)
            i += 1
            if i < len(tokens) and tokens[i] != "null":
                node.right = TreeNode(int(tokens[i]))
                queue.append(node.right)
            i += 1
        return root
```

## Solution -- Rust

```rust
use std::cell::RefCell;
use std::rc::Rc;

type Tree = Option<Rc<RefCell<TreeNode>>>;

#[derive(Debug, PartialEq)]
struct TreeNode {
    val: i32,
    left: Tree,
    right: Tree,
}

impl TreeNode {
    fn new(val: i32) -> Tree {
        Some(Rc::new(RefCell::new(TreeNode {
            val,
            left: None,
            right: None,
        })))
    }

    fn with_children(val: i32, left: Tree, right: Tree) -> Tree {
        Some(Rc::new(RefCell::new(TreeNode { val, left, right })))
    }
}

struct Codec;

impl Codec {
    fn serialize(root: &Tree) -> String {
        let mut parts: Vec<String> = Vec::new();
        Self::serialize_dfs(root, &mut parts);
        parts.join(",")
    }

    fn serialize_dfs(node: &Tree, parts: &mut Vec<String>) {
        match node {
            None => parts.push("null".to_string()),
            Some(rc) => {
                let n = rc.borrow();
                parts.push(n.val.to_string());
                Self::serialize_dfs(&n.left, parts);
                Self::serialize_dfs(&n.right, parts);
            }
        }
    }

    fn deserialize(data: &str) -> Tree {
        let tokens: Vec<&str> = data.split(',').collect();
        let mut idx = 0;
        Self::deserialize_dfs(&tokens, &mut idx)
    }

    fn deserialize_dfs(tokens: &[&str], idx: &mut usize) -> Tree {
        if *idx >= tokens.len() || tokens[*idx] == "null" {
            *idx += 1;
            return None;
        }
        let val: i32 = tokens[*idx].parse().unwrap();
        *idx += 1;
        let left = Self::deserialize_dfs(tokens, idx);
        let right = Self::deserialize_dfs(tokens, idx);
        TreeNode::with_children(val, left, right)
    }
}
```

## Solution -- TypeScript

```typescript
class TreeNode {
    val: number;
    left: TreeNode | null;
    right: TreeNode | null;

    constructor(val: number = 0, left: TreeNode | null = null, right: TreeNode | null = null) {
        this.val = val;
        this.left = left;
        this.right = right;
    }
}

function serialize(root: TreeNode | null): string {
    const parts: string[] = [];

    function dfs(node: TreeNode | null): void {
        if (node === null) {
            parts.push("null");
            return;
        }
        parts.push(String(node.val));
        dfs(node.left);
        dfs(node.right);
    }

    dfs(root);
    return parts.join(",");
}

function deserialize(data: string): TreeNode | null {
    const tokens = data.split(",");
    let idx = 0;

    function dfs(): TreeNode | null {
        if (idx >= tokens.length || tokens[idx] === "null") {
            idx++;
            return null;
        }
        const val = parseInt(tokens[idx], 10);
        idx++;
        const node = new TreeNode(val);
        node.left = dfs();
        node.right = dfs();
        return node;
    }

    return dfs();
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(n) -- each node is visited exactly once during serialize and deserialize |
| Space | O(n) -- the serialized string stores n values plus n+1 null markers; recursion depth is O(h) where h is tree height |

## Tips

- **Preorder is the natural choice** because the root comes first, making reconstruction
  straightforward -- just consume tokens left-to-right.
- **Null markers are essential.** Without them you cannot distinguish between different
  tree shapes that share the same values (e.g., left-skewed vs. right-skewed).
- **Watch for negative numbers** when parsing -- splitting by comma handles this
  correctly, but splitting by individual characters would not.
- **BFS vs DFS tradeoff:** BFS serialization mirrors level-order and is more human-readable;
  DFS serialization is simpler to implement recursively.
- In Rust, trees require `Rc<RefCell<T>>` for shared ownership with interior mutability.
  The `Option` wrapper makes null representation natural.

## See Also

- binary-trees
- depth-first-search
- breadth-first-search
- recursion

## References

- [LeetCode 297 -- Serialize and Deserialize Binary Tree](https://leetcode.com/problems/serialize-and-deserialize-binary-tree/)
- [LeetCode 449 -- Serialize and Deserialize BST](https://leetcode.com/problems/serialize-and-deserialize-bst/)
