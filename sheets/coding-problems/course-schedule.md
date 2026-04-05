# Course Schedule (Graphs / Topological Sort)

Determine whether all courses can be completed given a set of prerequisite dependencies -- equivalently, detect if a directed graph contains a cycle.

## Problem

There are `numCourses` courses labeled `0` to `numCourses - 1`. You are given an array
`prerequisites` where `prerequisites[i] = [a, b]` means you must take course `b` before
course `a`.

Return `true` if you can finish all courses, `false` otherwise.

This is equivalent to asking: **does the directed dependency graph have a valid topological ordering?** A topological ordering exists if and only if the graph is a DAG (directed acyclic graph).

**Constraints:**

- `1 <= numCourses <= 2000`
- `0 <= prerequisites.length <= 5000`
- `prerequisites[i].length == 2`
- `0 <= a, b < numCourses`
- All prerequisite pairs are unique.

**Examples:**

```
numCourses=2, prerequisites=[[1,0]]       => true   (take 0 then 1)
numCourses=2, prerequisites=[[1,0],[0,1]] => false  (circular dependency)
numCourses=4, prerequisites=[[1,0],[2,0],[3,1],[3,2]] => true
numCourses=3, prerequisites=[[0,1],[1,2],[2,0]] => false (3-node cycle)
```

## Hints

- **Kahn's algorithm (BFS):** Track in-degrees. Start with nodes having in-degree 0.
  Process them, decrement neighbors' in-degrees, and enqueue newly zero-degree nodes.
  If all nodes are processed, no cycle exists.
- **DFS coloring:** Use three states -- WHITE (unvisited), GRAY (in current DFS path),
  BLACK (fully processed). Encountering a GRAY node means a back edge exists, indicating
  a cycle.
- Both approaches run in O(V + E) time and space.

## Solution -- Go

```go
import "fmt"

// canFinishBFS uses Kahn's algorithm (BFS topological sort).
func canFinishBFS(numCourses int, prerequisites [][]int) bool {
	adj := make([][]int, numCourses)
	inDegree := make([]int, numCourses)

	for _, p := range prerequisites {
		course, prereq := p[0], p[1]
		adj[prereq] = append(adj[prereq], course)
		inDegree[course]++
	}

	// Queue: nodes with in-degree 0
	queue := make([]int, 0)
	for i := 0; i < numCourses; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	processed := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		processed++
		for _, neighbor := range adj[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return processed == numCourses
}

// canFinishDFS uses DFS with three-color marking.
func canFinishDFS(numCourses int, prerequisites [][]int) bool {
	const (
		white = 0
		gray  = 1
		black = 2
	)

	adj := make([][]int, numCourses)
	for _, p := range prerequisites {
		adj[p[1]] = append(adj[p[1]], p[0])
	}

	color := make([]int, numCourses)

	var hasCycle func(node int) bool
	hasCycle = func(node int) bool {
		color[node] = gray
		for _, neighbor := range adj[node] {
			if color[neighbor] == gray {
				return true
			}
			if color[neighbor] == white && hasCycle(neighbor) {
				return true
			}
		}
		color[node] = black
		return false
	}

	for i := 0; i < numCourses; i++ {
		if color[i] == white && hasCycle(i) {
			return false
		}
	}
	return true
}
```

## Solution -- Python

```python
from typing import List
from collections import deque


class Solution:
    def can_finish_bfs(self, num_courses: int, prerequisites: List[List[int]]) -> bool:
        """Kahn's algorithm (BFS-based topological sort)."""
        # Build adjacency list and in-degree array
        adj: List[List[int]] = [[] for _ in range(num_courses)]
        in_degree = [0] * num_courses

        for course, prereq in prerequisites:
            adj[prereq].append(course)
            in_degree[course] += 1

        # Start with all nodes that have no prerequisites
        queue: deque = deque()
        for i in range(num_courses):
            if in_degree[i] == 0:
                queue.append(i)

        processed = 0
        while queue:
            node = queue.popleft()
            processed += 1
            for neighbor in adj[node]:
                in_degree[neighbor] -= 1
                if in_degree[neighbor] == 0:
                    queue.append(neighbor)

        # If all courses processed, no cycle
        return processed == num_courses

    def can_finish_dfs(self, num_courses: int, prerequisites: List[List[int]]) -> bool:
        """DFS with three-color marking to detect cycles."""
        WHITE, GRAY, BLACK = 0, 1, 2

        adj: List[List[int]] = [[] for _ in range(num_courses)]
        for course, prereq in prerequisites:
            adj[prereq].append(course)

        color = [WHITE] * num_courses

        def has_cycle(node: int) -> bool:
            color[node] = GRAY
            for neighbor in adj[node]:
                if color[neighbor] == GRAY:
                    return True  # Back edge = cycle
                if color[neighbor] == WHITE and has_cycle(neighbor):
                    return True
            color[node] = BLACK
            return False

        for i in range(num_courses):
            if color[i] == WHITE and has_cycle(i):
                return False

        return True
```

## Solution -- Rust

```rust
use std::collections::VecDeque;

struct Solution;

impl Solution {
    /// Kahn's algorithm (BFS topological sort)
    fn can_finish_bfs(num_courses: usize, prerequisites: &[[usize; 2]]) -> bool {
        let mut adj: Vec<Vec<usize>> = vec![vec![]; num_courses];
        let mut in_degree = vec![0usize; num_courses];

        for p in prerequisites {
            let (course, prereq) = (p[0], p[1]);
            adj[prereq].push(course);
            in_degree[course] += 1;
        }

        let mut queue: VecDeque<usize> = VecDeque::new();
        for i in 0..num_courses {
            if in_degree[i] == 0 {
                queue.push_back(i);
            }
        }

        let mut processed = 0;
        while let Some(node) = queue.pop_front() {
            processed += 1;
            for &neighbor in &adj[node] {
                in_degree[neighbor] -= 1;
                if in_degree[neighbor] == 0 {
                    queue.push_back(neighbor);
                }
            }
        }

        processed == num_courses
    }

    /// DFS with three-color cycle detection
    fn can_finish_dfs(num_courses: usize, prerequisites: &[[usize; 2]]) -> bool {
        #[derive(Clone, Copy, PartialEq)]
        enum Color { White, Gray, Black }

        let mut adj: Vec<Vec<usize>> = vec![vec![]; num_courses];
        for p in prerequisites {
            adj[p[1]].push(p[0]);
        }

        let mut color = vec![Color::White; num_courses];

        fn has_cycle(node: usize, adj: &[Vec<usize>], color: &mut [Color]) -> bool {
            color[node] = Color::Gray;
            for &neighbor in &adj[node] {
                if color[neighbor] == Color::Gray {
                    return true;
                }
                if color[neighbor] == Color::White && has_cycle(neighbor, adj, color) {
                    return true;
                }
            }
            color[node] = Color::Black;
            false
        }

        for i in 0..num_courses {
            if color[i] == Color::White && has_cycle(i, &adj, &mut color) {
                return false;
            }
        }
        true
    }
}
```

## Solution -- TypeScript

```typescript
/** Kahn's algorithm (BFS topological sort) */
function canFinishBFS(numCourses: number, prerequisites: number[][]): boolean {
    const adj: number[][] = Array.from({ length: numCourses }, () => []);
    const inDegree = new Array(numCourses).fill(0);

    for (const [course, prereq] of prerequisites) {
        adj[prereq].push(course);
        inDegree[course]++;
    }

    const queue: number[] = [];
    for (let i = 0; i < numCourses; i++) {
        if (inDegree[i] === 0) queue.push(i);
    }

    let processed = 0;
    while (queue.length > 0) {
        const node = queue.shift()!;
        processed++;
        for (const neighbor of adj[node]) {
            inDegree[neighbor]--;
            if (inDegree[neighbor] === 0) queue.push(neighbor);
        }
    }

    return processed === numCourses;
}

/** DFS with three-color cycle detection */
function canFinishDFS(numCourses: number, prerequisites: number[][]): boolean {
    const WHITE = 0, GRAY = 1, BLACK = 2;

    const adj: number[][] = Array.from({ length: numCourses }, () => []);
    for (const [course, prereq] of prerequisites) {
        adj[prereq].push(course);
    }

    const color = new Array(numCourses).fill(WHITE);

    function hasCycle(node: number): boolean {
        color[node] = GRAY;
        for (const neighbor of adj[node]) {
            if (color[neighbor] === GRAY) return true;
            if (color[neighbor] === WHITE && hasCycle(neighbor)) return true;
        }
        color[node] = BLACK;
        return false;
    }

    for (let i = 0; i < numCourses; i++) {
        if (color[i] === WHITE && hasCycle(i)) return false;
    }
    return true;
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(V + E) -- each vertex and edge is visited at most once in both BFS and DFS |
| Space | O(V + E) -- adjacency list storage plus the in-degree array or color array |

## Tips

- **Kahn's algorithm is often preferred in interviews** because it naturally produces the
  topological ordering as a side effect (the processing order).
- **The DFS three-color scheme** is more general -- it works for detecting cycles in any
  directed graph, not just dependency graphs. GRAY = "currently on the recursion stack."
- **Common mistake:** Using a simple `visited` boolean instead of three colors in DFS.
  A two-state DFS cannot distinguish between a back edge (cycle) and a cross edge
  (already fully explored from another path).
- **Edge direction matters.** The prerequisite `[a, b]` means `b -> a` in the graph
  (b must come before a). Getting this backwards is a frequent bug.
- For the related problem "Course Schedule II" (return the ordering), simply record the
  processing order in Kahn's or the reverse post-order in DFS.

## See Also

- graphs
- topological-sort
- depth-first-search
- breadth-first-search
- cycle-detection

## References

- [LeetCode 207 -- Course Schedule](https://leetcode.com/problems/course-schedule/)
- [LeetCode 210 -- Course Schedule II](https://leetcode.com/problems/course-schedule-ii/)
- [Kahn's Algorithm (Wikipedia)](https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm)
