# The Mathematics of Word Ladder -- Implicit Graphs and Bidirectional Search

> *An explicit graph of word-to-word edit neighbours has $O(N^2)$ edges in the worst case. An implicit graph -- where neighbours are generated on demand -- costs nothing to construct and $O(L \cdot 26)$ to query. Bidirectional BFS collapses an $O(b^d)$ search into $O(2 b^{d/2})$ by expanding from both ends, exponentially reducing the frontier size for the same shortest path length.*

---

## 1. Graph Formulation

### The Word Graph

Let $D$ be a dictionary of $N$ words, each of length $L$ over an alphabet $\Sigma$ of size $\sigma$ (typically $\sigma = 26$). Define the undirected graph $G = (V, E)$ where:

$$V = D \cup \{w_s\}, \quad E = \{(u, v) : u, v \in V, \, \text{ham}(u, v) = 1\}$$

Here $\text{ham}(u, v)$ is the Hamming distance, the number of positions at which the two words differ.

The problem reduces to computing $d_G(w_s, w_t)$, the shortest path length in $G$, plus 1 (to count the starting word).

### Edge Count

In the worst case, each word has up to $L \cdot (\sigma - 1)$ neighbours in $V$, but only those present in $D$ count. The total edge count:

$$|E| \leq \frac{N \cdot L \cdot (\sigma - 1)}{2}$$

For $N = 5000$, $L = 10$, $\sigma = 26$: up to 625,000 edges. Pre-building this graph is $O(N \cdot L \cdot \sigma)$ time and $O(|E|)$ space.

### Implicit vs Explicit Representation

Explicit graph: pre-compute all edges, store adjacency lists. Pros: neighbours are a constant-time lookup. Cons: high memory; wasteful if only one query issued.

Implicit graph: generate neighbours on demand. Pros: no pre-computation; $O(1)$ extra space. Cons: $O(L \cdot \sigma)$ per neighbour query.

For single-query shortest path, implicit wins. For many queries over a fixed dictionary, explicit wins via amortisation.

---

## 2. BFS Correctness and Complexity

### The BFS Invariant

Breadth-first search from $w_s$ maintains:

$$d_k = \{v \in V : d_G(w_s, v) = k\}$$

where $d_k$ is the set of nodes at depth $k$. All nodes in $d_k$ are discovered before any node in $d_{k+1}$. Since $G$ is unweighted, BFS computes $d_G$ exactly.

### Time Complexity (Single BFS)

Let $b$ = branching factor (expected neighbours per node present in $D$), $d$ = shortest path length.

$$T_{\text{BFS}} = O(b^d \cdot L \cdot \sigma)$$

Each visited node spawns $L \cdot \sigma$ candidate strings. If $b \approx L \cdot \sigma$ (dense dictionary) the exponent dominates; otherwise the cost is roughly proportional to the number of visited dictionary words, bounded above by $N$.

Pessimistic bound: $T_{\text{BFS}} \leq O(N \cdot L \cdot \sigma)$, since each word is visited at most once and generates $L \cdot \sigma$ candidates.

---

## 3. Bidirectional Search

### The Exponential Reduction

Searching simultaneously from both $w_s$ and $w_t$ with BFS, the two frontiers meet at depth $d/2$:

$$T_{\text{bi}} = O(2 b^{d/2} \cdot L \cdot \sigma) = O(b^{d/2} \cdot L \cdot \sigma)$$

For $b = 10$, $d = 10$: single BFS visits $10^{10}$ nodes (infeasible), bidirectional visits $2 \cdot 10^5$ (trivial).

### Frontier Balancing

Expanding the smaller frontier at each step minimises the total work. If $|F_1| < |F_2|$, expanding $F_1$ costs $|F_1| \cdot L \cdot \sigma$; expanding $F_2$ would cost more. Choosing wrong doubles per-step cost but does not change asymptotic complexity.

### Termination Condition

Let $V_1$ = visited from source side, $V_2$ = visited from target side. At the moment of generating a new candidate $v$:

$$v \in V_2 \implies d_G(w_s, w_t) = d_1(v) + d_2(v)$$

where $d_1, d_2$ are the respective depths from each side. The BFS layering guarantees this is the minimum such sum.

### Correctness Proof

**Theorem**: Bidirectional BFS on an unweighted graph returns the shortest path length.

**Proof sketch**: Let $w_s = u_0, u_1, \ldots, u_d = w_t$ be the shortest path. The forward BFS discovers all nodes at distance $k$ before those at $k+1$; similarly backward. The path's midpoint is visited by one direction at depth $\lfloor d/2 \rfloor$ and by the other at depth $\lceil d/2 \rceil$. Hence the algorithm terminates at step $\lceil d/2 \rceil$ with the correct distance.

---

## 4. Adjacency via Wildcard Buckets

### Bucket Construction

For each word $w \in D$, generate $L$ patterns by replacing each character with a wildcard `*`. For `hot` ($L=3$): `*ot`, `h*t`, `ho*`.

Build a map $M : \text{pattern} \to \{\text{words matching}\}$. A word $w$'s neighbours are:

$$N(w) = \bigcup_{p \in \text{patterns}(w)} M[p] \setminus \{w\}$$

### Complexity

Construction: $O(N \cdot L)$ patterns, each $O(L)$ to generate. Total $O(N \cdot L^2)$.

Neighbour query: $O(L)$ pattern lookups, each returning up to $\sigma - 1$ words (on average). Total $O(L \cdot \bar{b})$ where $\bar{b}$ is the average words-per-pattern bucket size.

### Trade-off

For a dictionary re-queried $Q$ times:
- Implicit (no buckets): $Q \cdot O(N \cdot L \cdot \sigma)$
- Bucket-based: $O(N \cdot L^2) + Q \cdot O(d \cdot L \cdot \bar{b})$

When $Q$ is large or $\sigma \gg L$, buckets win significantly.

---

## 5. Generalisations

### Word Ladder II — All Shortest Paths (LC 126)

Track predecessor sets during BFS: for each discovered node $v$, record all nodes $u$ that reached $v$ at depth $d(v) - 1$. After BFS completes, perform DFS from $w_t$ back through predecessors to enumerate all shortest paths.

Complexity: BFS is $O(V + E)$ unchanged. Path enumeration is $O(P \cdot d)$ where $P$ is the number of shortest paths — can be exponential in the worst case.

### Levenshtein Distance BFS

Replace Hamming-1 neighbours with Levenshtein-1 neighbours (insertions, deletions, substitutions). The graph has more edges per node but the BFS framework is unchanged.

Neighbour generation: for each position, generate substitutions ($L \cdot \sigma$), deletions ($L$), and insertions ($(L+1) \cdot \sigma$). Total $O(L \cdot \sigma)$ candidates.

### Genome Sequence Alignment

DNA sequences with $\sigma = 4$ (A, C, G, T) and length $L \sim 10^3$ to $10^6$. The same implicit-graph BFS identifies nearest neighbours by single base changes, but the scale demands specialised data structures: suffix arrays, FM-index, or Bloom filters for dictionary membership.

---

## 6. Lower Bounds and Impossibility

### Comparison Model

Any algorithm that determines whether $w_s$ and $w_t$ are connected must examine enough dictionary words to rule out disconnection. In the worst case this is $\Omega(N)$.

For shortest path length, if the answer is $d$, the algorithm must examine at least one node at each of depths $1, 2, \ldots, d$ (else it cannot certify depth $d$). Hence $\Omega(d)$ visits.

Bidirectional BFS achieves $O(b^{d/2})$, exponentially better than single BFS but bounded below by $\Omega(d + \sqrt{|E|})$ in terms of graph size. For dense dictionaries this matches up to polynomial factors.

### Inapproximability

Approximating the shortest word ladder length within a factor $c < 2$ is not easier than exact computation — the graph structure forces exact BFS or worse.

---

## Prerequisites

- **Graph theory**: unweighted graphs, shortest paths, BFS traversal, predecessor trees
- **Hash sets and maps**: O(1) membership testing for dictionary lookups
- **Bidirectional search**: Pohl's algorithm (1971) and its complexity analysis
- **Levenshtein distance**: for the generalisation to insertions/deletions
- **String manipulation**: in-place character mutation for neighbour generation

## Complexity

| Aspect | Bound | Notes |
|--------|-------|-------|
| Single BFS time | $O(N \cdot L \cdot \sigma)$ | Visits each word once, $L \cdot \sigma$ candidates each |
| Bidirectional BFS time | $O(b^{d/2} \cdot L \cdot \sigma)$ | Exponential reduction vs single BFS |
| Wildcard bucket construction | $O(N \cdot L^2)$ | One-time preprocessing |
| Wildcard query time | $O(L \cdot \bar{b})$ | Per node |
| Space (implicit BFS) | $O(N \cdot L)$ | Visited set + frontier |
| Space (wildcard buckets) | $O(N \cdot L^2)$ | Bucket storage |
| Lower bound | $\Omega(d)$ | Must reach depth $d$ to certify |
| Word Ladder II enumeration | $O(P \cdot d)$ | $P$ = number of shortest paths, exponential worst case |

---
