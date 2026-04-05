# The Mathematics of Course Schedule -- Directed Graph Cycle Detection

> *When can a set of ordering constraints be simultaneously satisfied? The answer lies in the topology of directed graphs and the algebra of partial orders.*

---

## 1. Topological Ordering and Partial Orders (Order Theory)

### The Problem

Given a directed graph G = (V, E), a topological ordering is a linear sequence of all
vertices such that for every directed edge (u, v), vertex u appears before v. When does
such an ordering exist?

### The Formula

**Theorem:** A directed graph G has a topological ordering if and only if G is a DAG
(directed acyclic graph).

The number of distinct topological orderings of a DAG can vary enormously. For a DAG
with no edges (all vertices independent), there are $|V|!$ orderings. For a total order
(single chain), there is exactly 1.

The problem of counting topological orderings is #P-complete in general.

### Worked Examples

Graph with edges: $\{(0 \to 1), (0 \to 2), (1 \to 3), (2 \to 3)\}$

Valid topological orderings:
- $[0, 1, 2, 3]$
- $[0, 2, 1, 3]$

Both are valid because 0 precedes 1 and 2, and both 1 and 2 precede 3.

Graph with edges: $\{(0 \to 1), (1 \to 0)\}$

No topological ordering exists -- the cycle $0 \to 1 \to 0$ means we cannot place 0
before 1 and 1 before 0 simultaneously.

---

## 2. Kahn's Algorithm and In-Degree Analysis (Graph Theory)

### The Problem

Kahn's algorithm uses the invariant that a DAG must contain at least one vertex with
in-degree 0. Prove this, and show the algorithm's correctness.

### The Formula

**Lemma:** Every finite DAG has at least one vertex with in-degree 0.

*Proof by contradiction:* If every vertex has in-degree >= 1, then starting from any
vertex and following incoming edges backward, we must eventually revisit a vertex
(pigeonhole principle on |V| vertices), creating a cycle. Contradiction.

**Kahn's invariant:** At each step, remove a vertex with in-degree 0 and decrement the
in-degrees of its successors. After removing k vertices:

$$\text{processed} = k, \quad \sum_{v \in \text{remaining}} \text{in-degree}(v) = |E_{\text{remaining}}|$$

If the algorithm terminates with processed < |V|, the remaining subgraph has no vertex
with in-degree 0, meaning it contains a cycle.

### Worked Examples

Graph: 4 courses, edges $\{(0 \to 1), (0 \to 2), (1 \to 3), (2 \to 3)\}$

| Step | In-degrees | Queue | Processed |
|------|-----------|-------|-----------|
| Init | [0,1,1,2] | [0] | 0 |
| 1 | [-,0,0,2] | [1,2] | 1 |
| 2 | [-,-,0,1] | [2] | 2 |
| 3 | [-,-,-,0] | [3] | 3 |
| 4 | [-,-,-,-] | [] | 4 |

Processed = 4 = numCourses, so all courses can be completed.

---

## 3. DFS Coloring and Back Edge Classification (Graph Theory)

### The Problem

In a DFS traversal of a directed graph, edges are classified into four types. Only one
type indicates a cycle.

### The Formula

Edge classification in DFS:
- **Tree edge:** to an unvisited (WHITE) vertex -- part of the DFS tree
- **Back edge:** to an ancestor (GRAY) vertex -- **indicates a cycle**
- **Forward edge:** to a descendant (BLACK) vertex discovered via another path
- **Cross edge:** to a vertex (BLACK) in a different subtree

**Theorem:** A directed graph has a cycle if and only if a DFS traversal encounters
a back edge.

The three-color scheme encodes vertex state:
$$\text{color}(v) = \begin{cases} \text{WHITE} & v \text{ not yet discovered} \\ \text{GRAY} & v \text{ discovered, not yet finished (on stack)} \\ \text{BLACK} & v \text{ fully explored} \end{cases}$$

### Worked Examples

Graph: 3-node cycle $\{(0 \to 1), (1 \to 2), (2 \to 0)\}$

DFS from vertex 0:
1. Visit 0 (WHITE -> GRAY)
2. Visit 1 (WHITE -> GRAY)
3. Visit 2 (WHITE -> GRAY)
4. Edge 2 -> 0: vertex 0 is GRAY = **back edge detected, cycle exists**

The algorithm correctly returns false (cannot finish all courses).

---

## 4. Time Complexity via Aggregate Analysis (Amortized Analysis)

### The Problem

Both BFS and DFS claim O(V + E) time. Prove this rigorously using aggregate analysis.

### The Formula

**BFS (Kahn's):** Each vertex enters the queue at most once (when its in-degree hits 0).
Processing vertex v examines all outgoing edges $(v, w)$, decrementing in-degree[w].

$$T_{\text{BFS}} = \sum_{v \in V} (1 + \text{out-degree}(v)) = |V| + \sum_{v \in V} \text{out-degree}(v) = |V| + |E|$$

**DFS:** Each vertex transitions WHITE -> GRAY -> BLACK exactly once. The adjacency list
of vertex v is scanned once during the GRAY phase.

$$T_{\text{DFS}} = \sum_{v \in V} (O(1) + |\text{adj}(v)|) = O(|V| + |E|)$$

### Worked Examples

For the course schedule problem with V = 2000 and E = 5000:

$$T = O(2000 + 5000) = O(7000)$$

This is effectively instant on modern hardware. The problem is well within the
comfortable range for both approaches.

---

## 5. Relation to Constraint Satisfaction and 2-SAT (Complexity Theory)

### The Problem

Course prerequisites define a partial order. How does this relate to broader constraint
satisfaction problems?

### The Formula

The course schedule problem is a special case of constraint satisfaction where each
constraint is a simple binary ordering relation: $b \prec a$.

The satisfiability question "can all constraints be met simultaneously?" reduces to
acyclicity testing:

$$\text{satisfiable} \iff \nexists \text{ directed cycle in } G$$

More complex scheduling problems (with time windows, resource limits, or disjunctive
constraints) can be modeled as:
- **2-SAT** (polynomial time) for implication constraints
- **Integer Linear Programming** (NP-hard) for resource constraints
- **Constraint Programming** for general constraint networks

The simple prerequisite problem sits at the easiest end of this spectrum -- solvable
in linear time.

---

## Prerequisites

- Directed graphs (adjacency lists, in-degree, out-degree)
- BFS and DFS traversal
- Cycle detection in directed graphs
- Partial orders and total orders
- Basic proof techniques (contradiction, induction)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Understand that the problem is cycle detection. Implement Kahn's BFS approach: track in-degrees, process zero-degree nodes, check if all nodes are processed. |
| **Intermediate** | Implement DFS with three-color marking. Understand why two colors (visited/not visited) are insufficient for directed graphs. Prove correctness of both approaches. Solve Course Schedule II (return the ordering). |
| **Advanced** | Analyze strongly connected components (Tarjan's, Kosaraju's) for more general cycle structure. Extend to weighted dependency graphs, critical path analysis, and constraint satisfaction frameworks. Count topological orderings for specific graph families. |
