# The Mathematics of Neo4j — Graph Algorithms and Traversal Complexity

> *Neo4j stores data as a property graph where nodes and relationships are first-class citizens. The mathematics cover graph traversal complexity, PageRank convergence, shortest path algorithms, community detection modularity, and the cost model for Cypher query planning.*

---

## 1. Graph Traversal Complexity (Adjacency)

### The Problem

Neo4j uses index-free adjacency: each node physically points to its neighbors. This makes traversal cost proportional to the local neighborhood size, not the total graph size.

### The Formula

Traversal cost for a single hop from node $v$:

$$C_{\text{hop}}(v) = O(\deg(v))$$

For a $k$-hop traversal from starting node $v$:

$$C_{\text{k-hop}} = O\left(\sum_{i=0}^{k-1} \bar{d}^i\right) = O\left(\frac{\bar{d}^k - 1}{\bar{d} - 1}\right)$$

Where $\bar{d}$ is the average degree. For large $\bar{d}$:

$$C_{\text{k-hop}} \approx O(\bar{d}^k)$$

### Worked Examples

| Graph Size | Avg Degree | 1-hop | 2-hop | 3-hop | 4-hop |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 1M nodes | 10 | 10 | 100 | 1,000 | 10,000 |
| 1M nodes | 50 | 50 | 2,500 | 125,000 | 6,250,000 |
| 10M nodes | 10 | 10 | 100 | 1,000 | 10,000 |

The key insight: traversal cost is independent of total graph size, only dependent on local connectivity. This is why graph databases excel at relationship-heavy queries.

---

## 2. PageRank (Eigenvector Centrality)

### The Problem

PageRank measures node importance based on the structure of incoming links. It models a random walk on the graph.

### The Formula

$$PR(v) = \frac{1 - d}{N} + d \sum_{u \in \text{in}(v)} \frac{PR(u)}{\deg_{\text{out}}(u)}$$

Where $d = 0.85$ is the damping factor and $N$ is total nodes.

In matrix form:

$$\vec{r} = (1 - d) \cdot \frac{\vec{1}}{N} + d \cdot M^T \vec{r}$$

Where $M_{ij} = \frac{1}{\deg_{\text{out}}(j)}$ if edge $j \to i$ exists.

### Convergence

PageRank converges at rate $d$ per iteration:

$$\|r^{(t)} - r^*\| \leq d^t \cdot \|r^{(0)} - r^*\|$$

Iterations for error $\epsilon$:

$$t = \left\lceil \frac{\ln(\epsilon)}{\ln(d)} \right\rceil = \left\lceil \frac{\ln(\epsilon)}{\ln(0.85)} \right\rceil$$

| Target Error | Iterations Needed |
|:---:|:---:|
| $10^{-2}$ | 29 |
| $10^{-4}$ | 57 |
| $10^{-6}$ | 85 |
| $10^{-8}$ | 113 |

### Computational Cost

Per iteration: $O(|E|)$ (process each edge once). Total:

$$C_{\text{PageRank}} = O(t \times |E|) \approx O(57 \times |E|) \text{ for } \epsilon = 10^{-4}$$

---

## 3. Shortest Path (Dijkstra and A*)

### The Problem

Finding the shortest weighted path between two nodes is a fundamental graph operation. Neo4j supports Dijkstra's algorithm and A* with heuristics.

### The Formula

Dijkstra's algorithm with binary heap:

$$C_{\text{Dijkstra}} = O((|V| + |E|) \log |V|)$$

With Fibonacci heap:

$$C_{\text{Dijkstra-Fib}} = O(|E| + |V| \log |V|)$$

A* with admissible heuristic $h$:

$$f(v) = g(v) + h(v)$$

Where $g(v)$ is the actual cost from source and $h(v)$ is the heuristic estimate to target.

### Bidirectional Dijkstra

Neo4j's GDS uses bidirectional search, exploring from both ends:

$$C_{\text{bidir}} \approx 2 \times O\left(\bar{d}^{k/2}\right) \ll O(\bar{d}^k)$$

### Worked Example

Graph with 1M nodes, avg degree 20, finding path of length 6:

$$C_{\text{forward}} = 20^6 = 64{,}000{,}000 \text{ nodes explored}$$

$$C_{\text{bidir}} = 2 \times 20^3 = 16{,}000 \text{ nodes explored}$$

Speedup: $4{,}000\times$.

---

## 4. Community Detection (Louvain Modularity)

### The Problem

The Louvain algorithm detects communities by maximizing modularity, a measure of how dense connections are within communities compared to a random graph.

### The Formula

Modularity:

$$Q = \frac{1}{2m} \sum_{ij} \left[A_{ij} - \frac{k_i k_j}{2m}\right] \delta(c_i, c_j)$$

Where:
- $A_{ij}$ = adjacency matrix entry
- $k_i$ = degree of node $i$
- $m$ = total edges
- $c_i$ = community of node $i$
- $\delta$ = Kronecker delta (1 if same community)

Modularity gain from moving node $i$ to community $C$:

$$\Delta Q = \frac{k_{i,C}}{m} - \frac{\Sigma_C \cdot k_i}{2m^2}$$

Where $k_{i,C}$ = edges from $i$ to nodes in $C$, and $\Sigma_C$ = sum of degrees in $C$.

### Complexity

Louvain runs in near-linear time:

$$C_{\text{Louvain}} = O(|E| \cdot \log^2 |V|)$$

| Graph | Edges | Approx. Time (1 core) |
|:---:|:---:|:---:|
| 100K nodes | 1M | ~1 second |
| 1M nodes | 10M | ~15 seconds |
| 10M nodes | 100M | ~5 minutes |

---

## 5. Betweenness Centrality (Bridge Detection)

### The Problem

Betweenness centrality measures how often a node lies on shortest paths between other nodes. High betweenness indicates critical bridge nodes.

### The Formula

$$BC(v) = \sum_{s \neq v \neq t} \frac{\sigma_{st}(v)}{\sigma_{st}}$$

Where $\sigma_{st}$ = number of shortest paths from $s$ to $t$, and $\sigma_{st}(v)$ = those passing through $v$.

Normalized:

$$BC_{\text{norm}}(v) = \frac{BC(v)}{(N-1)(N-2)/2}$$

### Complexity

Brandes' algorithm:

$$C_{\text{Brandes}} = O(|V| \times |E|)$$

For unweighted graphs. This is expensive for large graphs; sampling-based approximation:

$$C_{\text{approx}} = O(k \times |E|), \quad k = O\left(\frac{\log N}{\epsilon^2}\right)$$

---

## 6. Cypher Query Planning (Cost Model)

### The Problem

Neo4j's query planner estimates execution cost using cardinality estimation and selects between index lookups, label scans, and expand operations.

### The Formula

Cardinality estimation for a pattern:

$$|\text{MATCH } (a:L_1)-[:R]->(b:L_2)| \approx |L_1| \times |L_2| \times \frac{|R|}{N^2}$$

More precisely, using selectivity:

$$\text{sel}(R, L_1, L_2) = \frac{|\{(a,b) : a \in L_1, (a,R,b), b \in L_2\}|}{|L_1| \times |L_2|}$$

Cost of index lookup:

$$C_{\text{index}} = O(\log |L| + k)$$

Where $k$ is the number of matching nodes.

Cost of label scan + filter:

$$C_{\text{scan}} = O(|L|)$$

The planner chooses index lookup when:

$$\text{selectivity} < \frac{\log |L|}{|L|}$$

---

## 7. Memory Estimation (Graph Projection)

### The Problem

GDS algorithms require projecting the graph into memory. Memory requirements depend on graph structure and algorithm needs.

### The Formula

Memory for projected graph:

$$M_{\text{graph}} = |V| \times (S_{\text{node}} + \bar{d} \times S_{\text{rel}})$$

Where $S_{\text{node}} \approx 56$ bytes (node ID + properties) and $S_{\text{rel}} \approx 24$ bytes (source + target + weight).

Algorithm-specific overhead:

$$M_{\text{PageRank}} = M_{\text{graph}} + |V| \times 16 \text{ (two double arrays)}$$

$$M_{\text{Louvain}} = M_{\text{graph}} + |V| \times 32 \text{ (community assignments + deltas)}$$

### Worked Example

Graph: 10M nodes, 100M relationships:

$$M_{\text{graph}} = 10^7 \times 56 + 10^8 \times 24 = 560 \text{ MB} + 2.4 \text{ GB} = 2.96 \text{ GB}$$

$$M_{\text{PageRank}} = 2.96 + 10^7 \times 16 / 10^9 = 2.96 + 0.16 = 3.12 \text{ GB}$$

Neo4j recommendation: allocate 2x estimated memory for safety headroom.

---

## Prerequisites

- graph-theory, linear-algebra, probability, algorithm-complexity
