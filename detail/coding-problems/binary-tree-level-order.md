# Breadth-First Tree Decomposition -- Level Order Traversal and Queue-Based Graph Search

> *Level order traversal decomposes a tree into horizontal slices, revealing structure that depth-first methods obscure. The algorithm is a direct application of breadth-first search constrained to a rooted tree, where the absence of cycles simplifies the visited-set machinery to nothing, and the queue becomes the sole organizing principle.*

---

## 1. The BFS Invariant on Trees

Given a rooted tree $T = (V, E)$ with root $r$ and $n = |V|$ nodes, define the **depth** of a node $v$ as the number of edges on the unique path from $r$ to $v$:

$$d(v) = \begin{cases} 0 & \text{if } v = r \\ d(\text{parent}(v)) + 1 & \text{otherwise} \end{cases}$$

The **level order traversal** produces a sequence of levels $L_0, L_1, \ldots, L_h$ where $h$ is the height of the tree and:

$$L_k = \{v \in V : d(v) = k\}$$

The BFS algorithm maintains a FIFO queue $Q$ with the following invariant: at the start of processing level $k$, the queue contains exactly the nodes of $L_k$, in left-to-right order.

**Proof by induction.** Base case: $Q = \{r\} = L_0$. Inductive step: assume $Q$ contains $L_k$ in order. Processing $L_k$ dequeues each node $v \in L_k$ left-to-right and enqueues its children left-to-right. Since children of nodes at depth $k$ have depth $k+1$, and children of a left-sibling precede children of a right-sibling, the queue now contains $L_{k+1}$ in left-to-right order. $\square$

### The Level-Size Technique

The key implementation detail is capturing $|L_k| = |Q|$ at the start of each level before any dequeue operations modify the queue. Let $s_k = |Q|$ at the start of level $k$. Then:

$$s_0 = 1, \quad s_{k+1} = \sum_{v \in L_k} |\text{children}(v)|$$

The algorithm dequeues exactly $s_k$ nodes for level $k$, collecting their values, and any nodes enqueued during this loop belong to level $k + 1$.

**Alternative approaches** include:
1. **Two-queue method:** alternate between two queues for current and next level. Uses $O(n)$ extra space from the second queue.
2. **Sentinel method:** enqueue a null marker after each level. Fragile and error-prone.
3. **DFS with depth tracking:** pass depth as a parameter during DFS, appending to `result[depth]`. Same time complexity but uses $O(h)$ stack space where $h$ is tree height (up to $O(n)$ for skewed trees).

The level-size technique is preferred: single queue, no sentinels, no recursion overhead.

## 2. Complexity Analysis

### Time Complexity

Each node is enqueued once and dequeued once. Each enqueue and dequeue is $O(1)$ for a proper queue (linked list or ring buffer). Total:

$$T(n) = \sum_{k=0}^{h} |L_k| \cdot O(1) = O(n)$$

For the DFS alternative, each node is visited once with $O(1)$ work per visit:

$$T_{\text{DFS}}(n) = O(n)$$

Both approaches are optimal since every node must be examined at least once.

### Space Complexity

The queue's maximum size equals the width of the tree -- the maximum number of nodes at any single level:

$$W(T) = \max_{0 \leq k \leq h} |L_k|$$

For a **complete binary tree** of height $h$ with $n = 2^{h+1} - 1$ nodes:

$$|L_k| = 2^k, \quad W(T) = 2^h = \frac{n+1}{2} = \Theta(n)$$

For a **skewed tree** (linear chain):

$$|L_k| = 1, \quad W(T) = 1 = O(1)$$

But the result list always requires $O(n)$ space regardless, so the overall space is $\Theta(n)$.

**DFS space comparison:** DFS uses $O(h)$ stack space where $h$ is tree height. For balanced trees $h = O(\log n)$, making DFS more space-efficient for the traversal itself. However, since the output is $O(n)$ in both cases, the practical difference is negligible.

### Width Bounds for Common Tree Shapes

| Tree Shape | Height $h$ | Max Width $W$ | Total Nodes $n$ |
|------------|-----------|---------------|-----------------|
| Complete binary | $\lfloor \log_2 n \rfloor$ | $\lceil n/2 \rceil$ | $2^{h+1}-1$ |
| Perfect binary | $\log_2(n+1) - 1$ | $(n+1)/2$ | $2^{h+1}-1$ |
| Full binary (worst) | $n/2$ | $\lceil n/2 \rceil$ | odd |
| Skewed (linear) | $n-1$ | $1$ | $n$ |
| $k$-ary complete | $\lfloor \log_k n \rfloor$ | $\Theta(n \cdot \frac{k-1}{k})$ | $(k^{h+1}-1)/(k-1)$ |

## 3. Relationship to General BFS

Level order traversal on a tree is BFS with the visited set removed. On a general graph $G = (V, E)$, BFS requires marking nodes as visited to avoid revisiting them via cycles or multiple paths. On a tree, the absence of cycles guarantees each node is reached exactly once -- through its unique parent.

The BFS shortest-path property still holds trivially: the depth $d(v)$ equals the shortest path distance from the root, which is also the unique path distance since trees have exactly one path between any two nodes.

### BFS on Graphs -- The Visited Set Cost

For a general graph with $|V| = n$ vertices and $|E| = m$ edges:

$$T_{\text{BFS}} = O(n + m), \quad S_{\text{BFS}} = O(n) \text{ (visited set + queue)}$$

For a tree, $m = n - 1$, so $T = O(n + (n-1)) = O(n)$, consistent with our tree-specific analysis.

## 4. Variants and Extensions

### Zigzag Level Order (LeetCode 103)

Alternate the direction of collection at each level: left-to-right for even levels, right-to-left for odd levels. The BFS structure is identical; only the level collection step changes (reverse odd-indexed levels, or use a deque and alternate append direction).

### Right Side View (LeetCode 199)

Return only the last node at each level (the rightmost visible node from the right side). Modify BFS to keep only the last value from each level, or equivalently, only record `queue[level_size - 1]`.

### Bottom-Up Level Order (LeetCode 107)

Return levels in reverse order (deepest level first). Run standard BFS, then reverse the result list. Alternatively, insert each level at the front of the result list.

### Average of Levels (LeetCode 637)

Compute the mean value at each level. Replace the level collection with a running sum, then divide by level size.

All variants share the same $O(n)$ time and $O(n)$ space bounds.

## 5. Queue Implementation Considerations

The choice of queue data structure affects constant factors:

**Array-based (Go slice, TypeScript Array):** `shift()` / `queue[1:]` are $O(k)$ where $k$ is the current queue length, making the naive implementation $O(n^2)$ in the worst case. Mitigations:
- Go: use an index variable instead of reslicing, or use `container/list`
- TypeScript: use a proper queue implementation or accept the $O(n^2)$ for interview purposes

**Deque (Python `collections.deque`, Rust `VecDeque`):** $O(1)$ amortized push/pop from both ends via a ring buffer. This is the correct choice.

**Linked list queue:** $O(1)$ worst-case enqueue/dequeue, but poor cache locality and allocation overhead.

For interview purposes, the $O(n)$ vs $O(n^2)$ distinction from queue choice is worth mentioning but rarely disqualifying. The algorithmic insight (BFS + level-size tracking) is what matters.

---

## Prerequisites

- **Trees:** rooted tree structure, depth, height, parent-child relationships, tree traversal orderings (preorder, inorder, postorder, level order).
- **BFS:** breadth-first search on graphs, FIFO queue invariant, shortest-path property, visited-set elimination for acyclic structures.
- **Queues:** FIFO semantics, amortized $O(1)$ operations, deque (double-ended queue) for efficient front removal.
- **Induction:** proof technique for establishing loop invariants and queue contents at each BFS step.

## Complexity

| Aspect | Bound | Notes |
|--------|-------|-------|
| Time | $O(n)$ | Each node enqueued and dequeued once |
| Space (queue) | $O(W)$ | $W$ = max tree width; $\Theta(n)$ worst case (complete tree) |
| Space (output) | $\Theta(n)$ | All node values stored in result |
| Space (total) | $\Theta(n)$ | Queue + output |
| DFS alternative time | $O(n)$ | Same asymptotic bound |
| DFS alternative space | $O(h)$ + $O(n)$ | $h$ = height (stack) + output; $h = O(\log n)$ balanced, $O(n)$ skewed |
| Queue operations (deque) | $O(1)$ amortized | Per enqueue/dequeue |
| Queue operations (array shift) | $O(n)$ per shift | Degrades total to $O(n^2)$; avoid in production |
