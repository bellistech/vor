# Graph Theory (Structures, Algorithms, and Applications)

A complete reference for graph types, representations, traversals, shortest paths, spanning trees, network flow, matching, coloring, and planarity — the combinatorial backbone of computer science.

## Graph Types

### Undirected Graph

```
G = (V, E) where E is a set of unordered pairs {u, v}

  A --- B       Vertices: {A, B, C, D}
  |     |       Edges: {{A,B}, {A,C}, {B,D}, {C,D}}
  C --- D

Degree of v = number of edges incident to v
Handshaking lemma: sum of all degrees = 2|E|
```

### Directed Graph (Digraph)

```
G = (V, E) where E is a set of ordered pairs (u, v)

  A --> B       Vertices: {A, B, C, D}
  |     |       Edges: {(A,B), (A,C), (B,D), (C,D)}
  v     v
  C --> D

In-degree(v)  = number of edges into v
Out-degree(v) = number of edges out of v
```

### Weighted Graph

```
G = (V, E, w) where w: E -> R assigns weights to edges

  A --3-- B
  |       |
  5       2
  |       |
  C --1-- D

Used for shortest paths, MST, network flow
```

### Bipartite Graph

```
G = (U, V, E) where every edge connects a vertex in U to one in V

  U: {1, 2, 3}     1 --- A
  V: {A, B, C}     2 --- B
                    3 --- C

Theorem: G is bipartite iff it contains no odd-length cycles
Test: BFS/DFS 2-coloring — O(V + E)
```

### Directed Acyclic Graph (DAG)

```
No directed cycles. Key properties:
  - Has at least one topological ordering
  - Topological sort via DFS or Kahn's algorithm — O(V + E)
  - Longest path computable in O(V + E) (unlike general graphs)

Applications: dependency resolution, scheduling, expression evaluation
```

### Tree

```
Connected acyclic undirected graph. Equivalent definitions:
  - Connected with |E| = |V| - 1
  - Acyclic with |E| = |V| - 1
  - Unique path between every pair of vertices

Rooted tree: one vertex designated as root, parent/child relationships
Spanning tree: subgraph that is a tree and includes all vertices
```

## Representations

### Adjacency Matrix

```
|V| x |V| matrix A where A[i][j] = 1 if edge (i,j) exists

For weighted graphs: A[i][j] = weight of edge (i,j), 0 or inf if absent

Space: O(V^2)
Edge lookup: O(1)
List neighbors: O(V)
Add edge: O(1)

Best for: dense graphs, matrix-based algorithms (Floyd-Warshall, spectral)
```

### Adjacency List

```
Array of |V| lists — each vertex stores its neighbors

For weighted graphs: store (neighbor, weight) pairs

Space: O(V + E)
Edge lookup: O(degree)
List neighbors: O(degree)
Add edge: O(1)

Best for: sparse graphs, traversals, most algorithms in practice
```

## Graph Traversals

### Breadth-First Search (BFS)

```
BFS(G, source):
  queue = [source]
  visited[source] = true
  dist[source] = 0
  while queue not empty:
    u = dequeue()
    for each neighbor v of u:
      if not visited[v]:
        visited[v] = true
        dist[v] = dist[u] + 1
        parent[v] = u
        enqueue(v)

Time: O(V + E)
Space: O(V)

Finds shortest paths in unweighted graphs
Produces BFS tree / shortest-path tree
Level-order traversal
```

### Depth-First Search (DFS)

```
DFS(G, source):
  stack = [source]           (or use recursion)
  while stack not empty:
    u = pop()
    if not visited[u]:
      visited[u] = true
      for each neighbor v of u:
        if not visited[v]:
          push(v)

Time: O(V + E)
Space: O(V)

DFS timestamps (discovery/finish) classify edges:
  Tree edge     — to unvisited vertex
  Back edge     — to ancestor (cycle detection)
  Forward edge  — to descendant (digraphs only)
  Cross edge    — everything else (digraphs only)
```

## Shortest Paths

### Dijkstra's Algorithm

```
Dijkstra(G, source):
  dist[source] = 0, dist[v] = inf for all other v
  priority_queue = [(0, source)]
  while pq not empty:
    (d, u) = extract_min()
    if d > dist[u]: continue
    for each neighbor v of u with weight w:
      if dist[u] + w < dist[v]:
        dist[v] = dist[u] + w
        parent[v] = u
        insert(pq, (dist[v], v))

Constraint: all edge weights must be non-negative
Time: O((V + E) log V) with binary heap
       O(V^2) with array (better for dense graphs)
       O(E + V log V) with Fibonacci heap
```

### Bellman-Ford Algorithm

```
BellmanFord(G, source):
  dist[source] = 0, dist[v] = inf for all other v
  repeat |V| - 1 times:
    for each edge (u, v, w) in E:
      if dist[u] + w < dist[v]:
        dist[v] = dist[u] + w
  # Negative cycle detection:
  for each edge (u, v, w) in E:
    if dist[u] + w < dist[v]:
      report NEGATIVE CYCLE

Time: O(VE)
Handles negative weights (but not negative cycles)
```

### Floyd-Warshall Algorithm

```
FloydWarshall(G):
  dist[i][j] = weight(i,j) if edge exists, inf otherwise
  dist[i][i] = 0
  for k = 1 to |V|:
    for i = 1 to |V|:
      for j = 1 to |V|:
        dist[i][j] = min(dist[i][j], dist[i][k] + dist[k][j])

Time: O(V^3)
Space: O(V^2)
All-pairs shortest paths
Detects negative cycles: dist[i][i] < 0 for some i
```

## Minimum Spanning Trees

### Kruskal's Algorithm

```
Kruskal(G):
  sort edges by weight ascending
  MST = empty set
  for each edge (u, v, w) in sorted order:
    if u and v are in different components:   (union-find)
      add (u, v, w) to MST
      union(u, v)
  return MST

Time: O(E log E) = O(E log V)
Greedy: always pick lightest edge that doesn't form a cycle
Uses union-find (disjoint set) data structure
```

### Prim's Algorithm

```
Prim(G, start):
  MST = empty set
  visited = {start}
  pq = all edges from start
  while |MST| < |V| - 1:
    (w, u, v) = extract_min(pq)
    if v not in visited:
      add (u, v, w) to MST
      visited.add(v)
      add all edges from v to pq

Time: O(E log V) with binary heap
       O(E + V log V) with Fibonacci heap
Greedy: grow MST one vertex at a time
```

### Cut Property

```
For any cut (S, V-S) of G, the minimum weight edge crossing
the cut is in every MST (assuming unique edge weights).

This is the correctness foundation for both Kruskal and Prim.
```

## Network Flow

### Ford-Fulkerson Method

```
FordFulkerson(G, s, t):
  initialize flow f = 0 on all edges
  while there exists an augmenting path p from s to t in residual graph:
    bottleneck = min capacity along p
    augment flow along p by bottleneck
    f += bottleneck
  return f

Residual graph G_f:
  For edge (u,v) with capacity c and flow f:
    Forward edge: capacity c - f
    Backward edge: capacity f

Edmonds-Karp: use BFS for augmenting paths — O(VE^2)
Dinic's algorithm: blocking flows — O(V^2 E)
```

### Max-Flow Min-Cut Theorem

```
In any flow network:
  max flow from s to t = min capacity of an s-t cut

An s-t cut (S, T) partitions vertices with s in S and t in T.
Capacity of cut = sum of capacities of edges from S to T.

Applications:
  - Bipartite matching (max matching = max flow)
  - Edge/vertex connectivity
  - Image segmentation
  - Network reliability
```

## Matching

### Bipartite Matching

```
A matching M in G is a set of edges with no shared endpoints.
Maximum matching: largest possible |M|.

Algorithms:
  Hungarian (Kuhn-Munkres) — O(V^3) for weighted assignment
  Hopcroft-Karp            — O(E sqrt(V)) for max cardinality
  Reduction to max flow    — O(VE)

Konig's theorem (bipartite only):
  max matching = min vertex cover
```

### Hall's Marriage Theorem

```
A bipartite graph G = (U, V, E) has a perfect matching
saturating every vertex in U if and only if:

  For all subsets S of U: |N(S)| >= |S|

  where N(S) = set of neighbors of S in V
```

## Graph Coloring

```
A proper k-coloring assigns one of k colors to each vertex
such that no two adjacent vertices share a color.

Chromatic number chi(G) = minimum k for a proper coloring

Key results:
  Bipartite graphs:  chi(G) = 2 (iff no odd cycles)
  Planar graphs:     chi(G) <= 4 (four color theorem, 1976)
  Any graph:         chi(G) <= Delta(G) + 1 (greedy bound)
  Brooks' theorem:   chi(G) <= Delta(G) unless G is complete or odd cycle

Computing chi(G) is NP-hard in general.
Greedy coloring: order vertices, assign smallest available color.

Applications: register allocation, scheduling, map coloring, frequency assignment
```

## Planarity

### Planar Graphs

```
A graph is planar if it can be drawn in the plane with no edge crossings.

Euler's formula: V - E + F = 2
  (V = vertices, E = edges, F = faces including unbounded face)

Corollaries:
  For simple connected planar graph with V >= 3:
    E <= 3V - 6
  If also triangle-free:
    E <= 2V - 4
```

### Kuratowski's Theorem

```
A graph is planar if and only if it contains no subdivision
of K_5 (complete graph on 5 vertices) or K_{3,3} (complete
bipartite graph on 3+3 vertices).

Equivalently (Wagner): no K_5 or K_{3,3} minor.

Testing planarity: O(V) algorithms exist (Hopcroft-Tarjan, Boyer-Myrvold)
```

## Euler and Hamilton Paths

### Eulerian Paths and Circuits

```
Euler path:    visits every EDGE exactly once
Euler circuit: Euler path that starts and ends at same vertex

Existence (undirected):
  Euler circuit: every vertex has even degree + connected
  Euler path:    exactly 0 or 2 vertices have odd degree + connected

Existence (directed):
  Euler circuit: in-degree = out-degree for all vertices + strongly connected
  Euler path:    at most one vertex with out - in = 1,
                 at most one with in - out = 1, rest equal

Hierholzer's algorithm: O(E)
```

### Hamiltonian Paths and Circuits

```
Hamilton path:    visits every VERTEX exactly once
Hamilton circuit: Hamilton path that returns to start

No simple necessary and sufficient condition exists.
Decision problem is NP-complete.

Sufficient conditions:
  Dirac's theorem:  if deg(v) >= V/2 for all v, then Hamiltonian circuit exists
  Ore's theorem:    if deg(u) + deg(v) >= V for all non-adjacent u,v, then Hamiltonian
```

## Strongly Connected Components

### Tarjan's Algorithm

```
Tarjan(G):
  index = 0, stack = []
  for each vertex v:
    if v.index undefined:
      strongconnect(v)

  strongconnect(v):
    v.index = v.lowlink = index++
    push v onto stack, v.onStack = true
    for each edge (v, w):
      if w.index undefined:
        strongconnect(w)
        v.lowlink = min(v.lowlink, w.lowlink)
      elif w.onStack:
        v.lowlink = min(v.lowlink, w.index)
    if v.lowlink == v.index:
      pop SCC from stack until v is popped

Time: O(V + E), single DFS pass
```

### Kosaraju's Algorithm

```
Kosaraju(G):
  1. DFS on G, push vertices onto stack in finish order
  2. Compute transpose G^T (reverse all edges)
  3. Pop vertices from stack, run DFS on G^T
     Each DFS tree in step 3 is one SCC

Time: O(V + E), two DFS passes
Simpler to implement than Tarjan, same complexity
```

## Key Figures

```
Leonhard Euler (1707-1783)
  - Founded graph theory with the Konigsberg bridge problem (1736)
  - Euler's formula for planar graphs: V - E + F = 2
  - Eulerian paths and circuits

Edsger Dijkstra (1930-2002)
  - Dijkstra's shortest path algorithm (1956)
  - Pioneered structured programming and semaphore concept

Joseph Kruskal (1928-2010)
  - Kruskal's minimum spanning tree algorithm (1956)
  - Kruskal-Katona theorem in combinatorics

Robert Tarjan (1948-)
  - Tarjan's SCC algorithm, offline LCA algorithm
  - Co-inventor of splay trees, Fibonacci heaps
  - Turing Award 1986 (with Hopcroft) for data structures and graph algorithms

Lester Ford Jr. (1927-2017)
  - Bellman-Ford shortest path algorithm (with Bellman)
  - Ford-Fulkerson max flow method (with Fulkerson)

Delbert Ray Fulkerson (1924-1976)
  - Ford-Fulkerson max flow method
  - Max-flow min-cut theorem (with Ford)
  - Contributions to combinatorial optimization and linear programming
```

## Tips

- BFS for unweighted shortest paths, Dijkstra for non-negative weights, Bellman-Ford when negative weights are possible
- Always check for negative cycles when using Bellman-Ford
- Kruskal is often simpler to implement; Prim is better for dense graphs
- DFS edge classification is the key to cycle detection, topological sort, and SCC
- Bipartiteness check: run BFS/DFS with 2-coloring — if you find a conflict, the graph has an odd cycle
- For flow problems, always think in terms of the residual graph
- Planarity testing is linear time, but the algorithms are complex — in practice, use a library

## See Also

- `detail/cs-theory/graph-theory.md` — formal definitions, Dijkstra correctness proof, max-flow min-cut theorem proof, spectral graph theory
- `sheets/cs-theory/complexity-classes.md` — P, NP, NP-completeness (many graph problems are NP-hard)
- `sheets/cs-theory/automata-theory.md` — finite automata as labeled directed graphs
- `sheets/algorithms/sorting.md` — topological sort, comparison-based lower bounds

## References

- Cormen, Leiserson, Rivest, Stein. "Introduction to Algorithms" (MIT Press, 4th edition, 2022)
- Bondy and Murty. "Graph Theory" (Springer, Graduate Texts in Mathematics, 2008)
- Diestel, Reinhard. "Graph Theory" (Springer, 5th edition, 2017)
- West, Douglas. "Introduction to Graph Theory" (Pearson, 2nd edition, 2001)
- Schrijver, Alexander. "Combinatorial Optimization: Polyhedra and Efficiency" (Springer, 2003)
