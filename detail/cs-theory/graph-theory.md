# The Mathematics of Graph Theory (Structures, Algorithms, and Deep Results)

> *A graph is one of the most versatile abstractions in mathematics -- deceptively simple in definition yet rich enough to encode problems from network routing to quantum computing, from social dynamics to the four-color conjecture that resisted proof for over a century.*

---

## 1. Formal Definitions

### The Problem

Establish the precise mathematical objects underlying graph theory, providing a rigorous foundation for all subsequent results.

### The Formula

A **graph** is an ordered pair $G = (V, E)$ where $V$ is a finite set of **vertices** and $E \subseteq \binom{V}{2}$ is a set of **edges** (unordered pairs of distinct vertices).

A **directed graph** (digraph) is $G = (V, E)$ where $E \subseteq V \times V$ is a set of **arcs** (ordered pairs).

A **weighted graph** is a triple $G = (V, E, w)$ where $w: E \to \mathbb{R}$ is a **weight function**.

Key derived concepts:

- The **degree** of $v \in V$ is $\deg(v) = |\{e \in E : v \in e\}|$
- The **neighborhood** of $v$ is $N(v) = \{u \in V : \{u, v\} \in E\}$
- A **walk** is a sequence $v_0, e_1, v_1, e_2, \ldots, e_k, v_k$ where each $e_i = \{v_{i-1}, v_i\} \in E$
- A **path** is a walk with no repeated vertices
- A **cycle** is a walk with $v_0 = v_k$ and no other repeated vertices
- $G$ is **connected** if there exists a path between every pair of vertices
- A **subgraph** $H = (V', E')$ of $G$ satisfies $V' \subseteq V$ and $E' \subseteq E \cap \binom{V'}{2}$

**Handshaking Lemma.** For any graph $G = (V, E)$:

$$\sum_{v \in V} \deg(v) = 2|E|$$

**Proof.** Each edge $\{u, v\}$ contributes exactly 1 to $\deg(u)$ and 1 to $\deg(v)$, so each edge contributes exactly 2 to the total degree sum. $\square$

### Worked Example

Consider $G$ with $V = \{a, b, c, d\}$ and $E = \{\{a,b\}, \{a,c\}, \{b,c\}, \{b,d\}, \{c,d\}\}$.

```
Degrees: deg(a) = 2, deg(b) = 3, deg(c) = 3, deg(d) = 2
Sum of degrees = 2 + 3 + 3 + 2 = 10 = 2 * 5 = 2|E|  (verified)

Adjacency matrix:
       a  b  c  d
  a  [ 0  1  1  0 ]
  b  [ 1  0  1  1 ]
  c  [ 1  1  0  1 ]
  d  [ 0  1  1  0 ]

Adjacency list:
  a -> [b, c]
  b -> [a, c, d]
  c -> [a, b, d]
  d -> [b, c]
```

---

## 2. Correctness of Dijkstra's Algorithm

### The Problem

Prove that Dijkstra's algorithm correctly computes single-source shortest paths in a graph with non-negative edge weights.

### The Formula

**Theorem.** Let $G = (V, E, w)$ be a weighted graph with $w(e) \ge 0$ for all $e \in E$. Dijkstra's algorithm computes $\text{dist}[v] = \delta(s, v)$ for every $v \in V$, where $\delta(s, v)$ is the true shortest-path distance from source $s$.

**Proof.** We prove the following loop invariant by induction on the number of vertices added to the settled set $S$:

**Invariant:** When vertex $u$ is extracted from the priority queue (added to $S$), $\text{dist}[u] = \delta(s, u)$.

**Base case.** $s$ is extracted with $\text{dist}[s] = 0 = \delta(s, s)$.

**Inductive step.** Suppose the invariant holds for all previously settled vertices. Let $u$ be the next vertex extracted with $\text{dist}[u] = d$. Assume for contradiction that $\delta(s, u) < d$.

Consider a true shortest path $P$ from $s$ to $u$. Let $(x, y)$ be the first edge on $P$ where $x \in S$ and $y \notin S$. Then:

$$\delta(s, u) = \delta(s, x) + w(x, y) + \delta(y, u)$$

By the inductive hypothesis, $\text{dist}[x] = \delta(s, x)$. When $x$ was settled, edge $(x, y)$ was relaxed, so:

$$\text{dist}[y] \le \text{dist}[x] + w(x, y) = \delta(s, x) + w(x, y)$$

Since $w \ge 0$, $\delta(y, u) \ge 0$, so:

$$\text{dist}[y] \le \delta(s, x) + w(x, y) \le \delta(s, x) + w(x, y) + \delta(y, u) = \delta(s, u) < d = \text{dist}[u]$$

But $u$ was extracted before $y$, meaning $\text{dist}[u] \le \text{dist}[y]$, which contradicts $\text{dist}[y] < \text{dist}[u]$. Therefore $\text{dist}[u] = \delta(s, u)$. $\square$

**Note.** The proof fails if any edge weight is negative, because $\delta(y, u) \ge 0$ no longer holds.

### Worked Example

```
Graph with source A:
  A --1-- B --3-- D
  |               |
  4               1
  |               |
  C ------2------ D

Step  Extract  dist[A]  dist[B]  dist[C]  dist[D]  S
  0   A        0        1        4        inf      {A}
  1   B        0        1        4        4        {A,B}
  2   C        0        1        4        4        {A,B,C}
  3   D        0        1        4        4        {A,B,C,D}

Shortest paths: A->A=0, A->B=1, A->C=4, A->D=4
```

---

## 3. The Max-Flow Min-Cut Theorem

### The Problem

Prove the fundamental duality between maximum flow and minimum cut in a network, establishing one of the most powerful results in combinatorial optimization.

### The Formula

A **flow network** is a directed graph $G = (V, E)$ with capacity function $c: E \to \mathbb{R}_{\ge 0}$, a source $s \in V$, and a sink $t \in V$.

A **flow** is a function $f: E \to \mathbb{R}_{\ge 0}$ satisfying:
1. **Capacity constraint:** $0 \le f(u, v) \le c(u, v)$ for all $(u, v) \in E$
2. **Conservation:** $\sum_{(u,v) \in E} f(u, v) = \sum_{(v, w) \in E} f(v, w)$ for all $v \in V \setminus \{s, t\}$

The **value** of flow $f$ is $|f| = \sum_{(s,v) \in E} f(s, v) - \sum_{(v,s) \in E} f(v, s)$.

An **$s$-$t$ cut** $(S, T)$ is a partition of $V$ with $s \in S$ and $t \in T$. Its **capacity** is:

$$c(S, T) = \sum_{\substack{u \in S, v \in T \\ (u,v) \in E}} c(u, v)$$

**Max-Flow Min-Cut Theorem (Ford and Fulkerson, 1956).** The following are equivalent:

1. $f$ is a maximum flow
2. The residual graph $G_f$ contains no augmenting path from $s$ to $t$
3. $|f| = c(S, T)$ for some $s$-$t$ cut $(S, T)$

**Proof ($1 \Rightarrow 2$).** If $G_f$ contains an augmenting path, we can increase $|f|$ by the bottleneck capacity along that path, contradicting maximality.

**Proof ($2 \Rightarrow 3$).** Suppose no augmenting path exists. Define $S = \{v \in V : v \text{ is reachable from } s \text{ in } G_f\}$ and $T = V \setminus S$. Then $s \in S$ and $t \in T$ (since $t$ is unreachable).

For any edge $(u, v)$ with $u \in S, v \in T$: since $v$ is not reachable in $G_f$, there is no residual capacity from $u$ to $v$, so $f(u, v) = c(u, v)$.

For any edge $(v, u)$ with $v \in T, u \in S$: if $f(v, u) > 0$, there would be a residual backward edge from $u$ to $v$, making $v$ reachable, contradicting $v \in T$. So $f(v, u) = 0$.

Therefore:

$$|f| = \sum_{\substack{u \in S, v \in T}} f(u,v) - \sum_{\substack{v \in T, u \in S}} f(v,u) = \sum_{\substack{u \in S, v \in T}} c(u,v) - 0 = c(S, T)$$

**Proof ($3 \Rightarrow 1$).** For any flow $f'$ and any $s$-$t$ cut $(S, T)$: $|f'| \le c(S, T)$ (weak duality, since flow across any cut is at most the cut capacity). If $|f| = c(S, T)$, then $f$ achieves this upper bound and must be maximum. $\square$

### Worked Example

```
Network:
  s --10--> A --8--> t
  s --5---> B --7--> t
  A --3---> B

Max flow = 15 (10 through s->A->t, 5 through s->B->t)
Min cut: S = {s}, T = {A, B, t}, capacity = 10 + 5 = 15
```

---

## 4. Konig's Theorem

### The Problem

Prove the duality between maximum matchings and minimum vertex covers in bipartite graphs.

### The Formula

A **matching** $M$ in $G$ is a set of pairwise non-adjacent edges. A **vertex cover** $C$ is a set of vertices such that every edge has at least one endpoint in $C$.

**Konig's Theorem (1931).** In any bipartite graph:

$$\nu(G) = \tau(G)$$

where $\nu(G) = $ size of maximum matching and $\tau(G) = $ size of minimum vertex cover.

**Proof.** The inequality $\nu(G) \le \tau(G)$ is immediate: any vertex cover must include at least one endpoint of each matched edge, so $|C| \ge |M|$ for any cover $C$ and matching $M$.

For $\nu(G) \ge \tau(G)$: model the bipartite graph $G = (U, V, E)$ as a flow network. Add source $s$ with edges of capacity 1 to all $u \in U$. Add edges of capacity $\infty$ from $u$ to $v$ for each $(u,v) \in E$. Add edges of capacity 1 from all $v \in V$ to sink $t$.

The max flow equals the max matching (integral flow on bipartite network). By max-flow min-cut, the min cut has finite capacity equal to the max flow. The min cut selects vertices from $U$ and $V$ that form a vertex cover of size equal to the max flow. $\square$

**Note.** Konig's theorem fails for non-bipartite graphs. Example: triangle $K_3$ has $\nu = 1$ but $\tau = 2$.

---

## 5. The Five-Color Theorem

### The Problem

Prove that every planar graph can be properly colored with at most five colors, a stepping stone toward the famous four-color theorem.

### The Formula

**Euler's Formula for Planar Graphs.** For any connected planar graph with $V$ vertices, $E$ edges, and $F$ faces (including the unbounded face):

$$V - E + F = 2$$

**Corollary.** For a simple connected planar graph with $V \ge 3$: $E \le 3V - 6$.

**Proof of corollary.** Each face is bounded by at least 3 edges, and each edge borders at most 2 faces: $2E \ge 3F$, so $F \le \frac{2E}{3}$. Substituting into Euler's formula: $V - E + \frac{2E}{3} \ge 2$, giving $E \le 3V - 6$. $\square$

**Corollary.** Every simple planar graph has a vertex of degree at most 5.

**Proof.** If every vertex had degree $\ge 6$, then $2E = \sum \deg(v) \ge 6V$, so $E \ge 3V$, contradicting $E \le 3V - 6$. $\square$

**Five-Color Theorem (Heawood, 1890).** Every planar graph is 5-colorable.

**Proof** (by induction on $|V|$). Base: $|V| \le 5$ is trivially 5-colorable.

Inductive step: Let $G$ be a planar graph with $|V| > 5$. By the corollary above, $G$ contains a vertex $v$ with $\deg(v) \le 5$.

**Case 1:** $\deg(v) \le 4$. Remove $v$, inductively 5-color $G - v$, then color $v$ with any color not used by its at most 4 neighbors.

**Case 2:** $\deg(v) = 5$ and all 5 neighbors use distinct colors. Let the neighbors be $v_1, \ldots, v_5$ colored $c_1, \ldots, c_5$ clockwise in the planar embedding.

Consider the subgraph $H_{1,3}$ induced by vertices colored $c_1$ or $c_3$. If $v_1$ and $v_3$ are in different components of $H_{1,3}$, swap $c_1 \leftrightarrow c_3$ in the component containing $v_1$. Now $v_1$ has color $c_3$ and $v_3$ still has $c_3$, freeing $c_1$ for $v$.

If $v_1$ and $v_3$ are in the same component, there is a path from $v_1$ to $v_3$ using only colors $c_1$ and $c_3$. This path, together with $v$, forms a closed curve separating $v_2$ from $v_4$ in the plane. Therefore $v_2$ and $v_4$ are in different components of $H_{2,4}$ (subgraph of colors $c_2, c_4$). Swap $c_2 \leftrightarrow c_4$ in the component containing $v_2$, freeing $c_2$ for $v$. $\square$

---

## 6. Euler's Formula for Planar Graphs

### The Problem

Prove the fundamental relationship $V - E + F = 2$ for connected planar graphs and derive its consequences.

### The Formula

**Euler's Formula.** For any connected planar graph $G$ with $V$ vertices, $E$ edges, and $F$ faces:

$$V - E + F = 2$$

**Proof** (by induction on $E$).

**Base case.** If $E = 0$, then $G$ is a single vertex: $V = 1$, $F = 1$ (unbounded face), and $1 - 0 + 1 = 2$.

**Inductive step.** Assume the formula holds for all connected planar graphs with fewer than $E$ edges.

**Case 1:** $G$ is a tree. Then $E = V - 1$ and $F = 1$ (only unbounded face), so $V - (V-1) + 1 = 2$.

**Case 2:** $G$ contains a cycle. Let $e$ be an edge on some cycle. Removing $e$ gives a connected planar graph $G' = G - e$ with $V' = V$, $E' = E - 1$, and $F' = F - 1$ (removing $e$ merges two faces). By induction:

$$V - (E - 1) + (F - 1) = 2 \implies V - E + F = 2 \quad \square$$

**Consequences:**

1. **Non-planarity of $K_5$:** $V = 5$, $E = 10$. But $E \le 3V - 6 = 9$. Contradiction.
2. **Non-planarity of $K_{3,3}$:** $V = 6$, $E = 9$. Triangle-free, so $E \le 2V - 4 = 8$. Contradiction.
3. **Genus formula:** For a graph embedded on a surface of genus $g$: $V - E + F = 2 - 2g$.

---

## 7. Ramsey Theory Basics

### The Problem

Establish that sufficiently large structures inevitably contain highly organized substructures, formalizing the principle that "complete disorder is impossible."

### The Formula

**Ramsey's Theorem (1930).** For any positive integers $r$ and $s$, there exists a minimum integer $R(r, s)$ such that any 2-coloring (red/blue) of the edges of $K_n$ with $n \ge R(r, s)$ contains either a red $K_r$ or a blue $K_s$.

**Theorem.** $R(r, s)$ exists for all $r, s \ge 1$, and:

$$R(r, s) \le \binom{r + s - 2}{r - 1}$$

**Proof** (by induction on $r + s$).

Base cases: $R(r, 1) = R(1, s) = 1$ (trivially).

Inductive step: Let $n = R(r-1, s) + R(r, s-1)$, and 2-color $K_n$. Pick any vertex $v$. Partition the remaining $n - 1$ vertices into $A$ (red neighbors of $v$) and $B$ (blue neighbors of $v$).

Since $|A| + |B| = n - 1 = R(r-1, s) + R(r, s-1) - 1$, either $|A| \ge R(r-1, s)$ or $|B| \ge R(r, s-1)$.

- If $|A| \ge R(r-1, s)$: Among $A$, there is a red $K_{r-1}$ (add $v$ for red $K_r$) or a blue $K_s$.
- If $|B| \ge R(r, s-1)$: Among $B$, there is a red $K_r$ or a blue $K_{s-1}$ (add $v$ for blue $K_s$).

Therefore $R(r, s) \le R(r-1, s) + R(r, s-1) \le \binom{r+s-2}{r-1}$. $\square$

**Known values:**

```
R(3,3) = 6    (party problem: among 6 people, 3 mutual friends or 3 mutual strangers)
R(3,4) = 9
R(3,5) = 14
R(4,4) = 18
R(4,5) = 25
R(5,5) is unknown — bounded between 43 and 46
```

---

## 8. Spectral Graph Theory (Laplacian)

### The Problem

Introduce the connection between graph structure and the eigenvalues of associated matrices, revealing algebraic invariants that encode combinatorial properties.

### The Formula

The **adjacency matrix** $A$ of $G$ is the $|V| \times |V|$ matrix with $A_{ij} = 1$ if $\{i, j\} \in E$ and 0 otherwise.

The **degree matrix** $D$ is the diagonal matrix with $D_{ii} = \deg(i)$.

The **Laplacian matrix** is $L = D - A$.

**Properties of $L$:**

1. $L$ is symmetric and positive semidefinite
2. Row sums are zero, so $L \mathbf{1} = \mathbf{0}$ (the all-ones vector is an eigenvector with eigenvalue 0)
3. For any vector $\mathbf{x} \in \mathbb{R}^{|V|}$:

$$\mathbf{x}^T L \mathbf{x} = \sum_{\{i,j\} \in E} (x_i - x_j)^2$$

Let the eigenvalues of $L$ be $0 = \lambda_1 \le \lambda_2 \le \cdots \le \lambda_n$.

**Key results:**

- **Connectivity:** The multiplicity of eigenvalue 0 equals the number of connected components. In particular, $G$ is connected if and only if $\lambda_2 > 0$.
- **Algebraic connectivity (Fiedler, 1973):** $\lambda_2$ is called the algebraic connectivity. Larger $\lambda_2$ means the graph is "more connected."
- **Cheeger inequality:** For the edge expansion $h(G)$:

$$\frac{\lambda_2}{2} \le h(G) \le \sqrt{2 \lambda_2}$$

- **Matrix-Tree Theorem (Kirchhoff, 1847):** The number of spanning trees of $G$ is:

$$t(G) = \frac{1}{n} \lambda_2 \lambda_3 \cdots \lambda_n = \frac{1}{n} \prod_{i=2}^{n} \lambda_i$$

Equivalently, $t(G)$ equals any cofactor of $L$.

### Worked Example

```
Graph: path P_3 on vertices {1, 2, 3} with edges {1,2} and {2,3}

A = [ 0 1 0 ]    D = [ 1 0 0 ]    L = [ 1 -1  0 ]
    [ 1 0 1 ]        [ 0 2 0 ]        [-1  2 -1 ]
    [ 0 1 0 ]        [ 0 0 1 ]        [ 0 -1  1 ]

Eigenvalues of L: 0, 1, 3
  lambda_1 = 0  (always)
  lambda_2 = 1  (algebraic connectivity > 0, so connected)
  lambda_3 = 3

Number of spanning trees = (1/3) * 1 * 3 = 1  (P_3 is itself a tree)
```

---

## 9. Graph Isomorphism Complexity

### The Problem

Determine the computational complexity of deciding whether two graphs are structurally identical, one of the most tantalizing open problems in complexity theory.

### The Formula

Two graphs $G_1 = (V_1, E_1)$ and $G_2 = (V_2, E_2)$ are **isomorphic** ($G_1 \cong G_2$) if there exists a bijection $\phi: V_1 \to V_2$ such that:

$$\{u, v\} \in E_1 \iff \{\phi(u), \phi(v)\} \in E_2$$

The **Graph Isomorphism Problem (GI):** Given $G_1, G_2$, decide whether $G_1 \cong G_2$.

**Complexity status:**

- GI $\in$ NP (a permutation $\phi$ serves as a polynomial-time verifiable certificate)
- GI is not known to be in P
- GI is not known to be NP-complete
- GI $\in$ co-AM, which means if GI is NP-complete then the polynomial hierarchy collapses to $\Sigma_2^P$ (considered unlikely)

**Babai's breakthrough (2015):** GI can be solved in **quasipolynomial time**:

$$T(n) = 2^{O((\log n)^c)} \text{ for a constant } c$$

This places GI strictly between P and NP under standard assumptions, making it one of the very few natural problems with this intermediate status (alongside factoring).

**Tractable special cases:**

| Graph class | Algorithm | Time |
|---|---|---|
| Trees | AHU algorithm | $O(n \log n)$ |
| Planar graphs | Hopcroft-Wong | $O(n \log n)$ |
| Bounded degree | Luks | $O(n^{O(1)})$ |
| Bounded treewidth | Bodlaender | $O(n^{O(1)})$ |

**Graph invariants** (necessary but not sufficient for isomorphism):
- Degree sequence
- Number of vertices, edges, triangles
- Spectrum of adjacency/Laplacian matrix
- Characteristic polynomial

Non-isomorphic graphs sharing all common invariants are called **cospectral mates** and demonstrate that no polynomial set of simple invariants suffices to solve GI.

---

## Prerequisites

- Set theory, functions, and relations (bijections, equivalence relations)
- Basic linear algebra (matrices, eigenvalues, eigenvectors)
- Proof techniques (induction, contradiction, pigeonhole principle)
- Algorithm analysis and asymptotic notation
- Discrete probability basics (for randomized graph algorithms)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Define graphs formally. Implement BFS and DFS. Compute shortest paths with Dijkstra on small examples. Identify bipartite graphs. Understand adjacency matrix vs. adjacency list tradeoffs. |
| **Intermediate** | Prove correctness of Dijkstra's algorithm. Apply max-flow min-cut to model optimization problems. Implement Tarjan's SCC algorithm. Prove Euler's formula and its corollaries. Reduce matching to network flow. |
| **Advanced** | Derive spectral properties from the Laplacian. Apply Ramsey theory bounds. Understand the complexity landscape of graph isomorphism and Babai's quasipolynomial result. Prove Konig's theorem via LP duality. Connect Cheeger inequality to expander graphs and random walks. Analyze graph algorithms on restricted graph classes (planar, bounded treewidth). |
