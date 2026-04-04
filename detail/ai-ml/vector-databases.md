# The Mathematics of Vector Databases -- Graph Search, Quantization, and Recall-Latency Tradeoffs

> *Vector databases solve the approximate nearest neighbor (ANN) problem at scale: given a query point in high-dimensional space, find the k closest points from billions of candidates in milliseconds. The underlying algorithms balance construction cost, query latency, memory footprint, and recall through careful application of graph theory, quantization theory, and probabilistic data structures.*

---

## 1. HNSW Graph Construction (Graph Theory / Probability)
### The Problem
Exact nearest neighbor search in $d$ dimensions requires $O(nd)$ comparisons. For $n = 10^9$ vectors at $d = 1536$, this is $\sim 1.5 \times 10^{12}$ operations per query -- far too slow for real-time retrieval.

### The Formula
HNSW constructs a hierarchical graph with $L$ layers. The probability of a node appearing at layer $l$:

$$P(\text{layer} \geq l) = e^{-l / m_L}$$

where $m_L = 1 / \ln(M)$ and $M$ is the max connections parameter. The expected maximum layer:

$$L_{\max} = \lfloor \log_M n \rfloor = \frac{\ln n}{\ln M}$$

For $n = 10^6$, $M = 16$: $L_{\max} = \frac{13.8}{2.77} \approx 5$ layers.

### Construction Complexity
Inserting one node:
- Find nearest neighbors at each layer via greedy search: $O(M \cdot L_{\max} \cdot \log n)$
- Connect to $M$ neighbors at layer 0, fewer at higher layers
- Total build time for $n$ nodes:

$$T_{\text{build}} = O(n \cdot M \cdot \log^2 n)$$

### Search Complexity
Greedy search with beam width $ef$:

$$T_{\text{search}} = O(ef \cdot M \cdot \log n \cdot d)$$

The $d$ factor comes from distance computation for each candidate.

### Worked Example
For $n = 10^6$, $M = 16$, $ef_{\text{search}} = 100$, $d = 1536$:

$$T_{\text{search}} = 100 \times 16 \times \log_2(10^6) \times 1536$$

$$= 100 \times 16 \times 20 \times 1536 = 49{,}152{,}000 \text{ operations}$$

Compare to brute force: $10^6 \times 1536 = 1.536 \times 10^9$ operations.

Speedup: $\frac{1.536 \times 10^9}{4.9 \times 10^7} \approx 31\times$ with recall > 95%.

### Memory Usage
$$\text{Memory}_{\text{HNSW}} = n \cdot (d \cdot b_{\text{float}} + M \cdot b_{\text{link}})$$

For $n = 10^6$, $d = 1536$ (FP32), $M = 16$:

$$= 10^6 \times (1536 \times 4 + 16 \times 8) = 10^6 \times 6272 \approx 6 \text{ GB}$$

## 2. IVF Quantization and Search (Clustering Theory)
### The Problem
IVF (Inverted File) partitions the vector space into Voronoi cells using k-means, then searches only the cells nearest to the query.

### The Formula
K-means objective for $C$ clusters:

$$\min_{\mu_1, \ldots, \mu_C} \sum_{i=1}^{n} \min_{j=1}^{C} \|\mathbf{x}_i - \boldsymbol{\mu}_j\|^2$$

At query time, the search visits $nprobe$ clusters. The expected recall:

$$\text{Recall} \approx 1 - \left(1 - \frac{nprobe}{C}\right)^k$$

for retrieving $k$ nearest neighbors, assuming uniform distribution. In practice, recall is higher because nearby clusters are selected.

### Quantization Error
The average quantization error (residual) per vector:

$$\epsilon_{\text{IVF}} = \frac{1}{n}\sum_{i=1}^{n} \|\mathbf{x}_i - \boldsymbol{\mu}_{c(i)}\|^2$$

where $c(i)$ is the cluster assignment of $\mathbf{x}_i$. This error determines recall: lower error means the cluster centroid better represents its members.

### Worked Example
For $n = 10^6$ vectors, $C = 1024$ clusters, $nprobe = 32$:

- Vectors per cluster (average): $\frac{10^6}{1024} \approx 977$
- Vectors searched: $32 \times 977 = 31{,}264$ ($\sim 3.1\%$ of total)
- Cluster assignment: $O(C \cdot d) = O(1024 \times 1536)$ distance computations
- Within-cluster search: $O(nprobe \times n/C \times d) = O(32 \times 977 \times 1536)$
- Total: $\sim 49.6M$ vs brute-force $1.536 \times 10^9$: speedup $\approx 31\times$

## 3. Product Quantization (Information Theory / Compression)
### The Problem
Storing $n = 10^9$ vectors at $d = 1536$ dimensions in FP32 requires $\sim 5.7$ TB. Product quantization compresses each vector to a few dozen bytes while preserving distance computations.

### The Formula
PQ splits a $d$-dimensional vector into $m$ subvectors of dimension $d/m$:

$$\mathbf{x} = [\underbrace{x_1, \ldots, x_{d/m}}_{\text{subvector 1}}, \ldots, \underbrace{x_{d-d/m+1}, \ldots, x_d}_{\text{subvector } m}]$$

Each subvector is quantized to its nearest centroid from a codebook of $k = 2^b$ entries:

$$q_j(\mathbf{x}^{(j)}) = \arg\min_{c \in \mathcal{C}_j} \|\mathbf{x}^{(j)} - c\|^2$$

The compressed representation: $m$ centroid indices, each $b$ bits.

### Compression Ratio
$$\text{Original: } d \times 32 \text{ bits (FP32)}$$
$$\text{Compressed: } m \times b \text{ bits}$$
$$\text{Ratio: } \frac{32d}{mb}$$

For $d = 1536$, $m = 192$, $b = 8$:

$$\text{Ratio} = \frac{32 \times 1536}{192 \times 8} = \frac{49{,}152}{1{,}536} = 32\times$$

### Asymmetric Distance Computation (ADC)
Instead of compressing the query, compute exact distances between the query subvectors and codebook centroids:

$$d_{\text{ADC}}(\mathbf{q}, \mathbf{x}) = \sum_{j=1}^{m} \|\mathbf{q}^{(j)} - c_{q_j(\mathbf{x}^{(j)})}\|^2$$

Pre-compute a distance table: $m \times k$ distances, then lookup and sum.

$$T_{\text{precompute}} = O(m \cdot k \cdot d/m) = O(kd)$$
$$T_{\text{per\_vector}} = O(m) \quad \text{(just lookups and additions)}$$

### Quantization Error Bound
The expected distortion of PQ:

$$\mathbb{E}[\|\mathbf{x} - q(\mathbf{x})\|^2] = \sum_{j=1}^{m} \mathbb{E}[\|\mathbf{x}^{(j)} - q_j(\mathbf{x}^{(j)})\|^2]$$

By rate-distortion theory, the minimum achievable distortion with $b$ bits per subvector and Gaussian data:

$$D_{\min}(b) = \frac{d}{m} \cdot \sigma^2 \cdot 2^{-2b/(d/m)}$$

For $b = 8$, $d/m = 8$: $D_{\min} \approx 0.0039 \sigma^2$ per subvector.

## 4. Distance Metrics (Metric Spaces)
### Cosine Distance
$$d_{\cos}(\mathbf{a}, \mathbf{b}) = 1 - \frac{\mathbf{a} \cdot \mathbf{b}}{\|\mathbf{a}\| \|\mathbf{b}\|}$$

For normalized vectors: $d_{\cos} = 1 - \mathbf{a} \cdot \mathbf{b}$

Properties:
- Range: $[0, 2]$ (0 = identical direction, 2 = opposite)
- Not a true metric (violates triangle inequality for unnormalized vectors)
- Invariant to vector magnitude

### L2 Distance and its Relation to Cosine
For normalized vectors:

$$\|\mathbf{a} - \mathbf{b}\|_2^2 = 2(1 - \mathbf{a} \cdot \mathbf{b}) = 2 \cdot d_{\cos}(\mathbf{a}, \mathbf{b})$$

This means for normalized embeddings, L2 and cosine produce identical rankings.

### Inner Product (MaxSim)
$$\text{IP}(\mathbf{a}, \mathbf{b}) = \mathbf{a} \cdot \mathbf{b} = \sum_{i=1}^{d} a_i b_i$$

Not a distance (higher = more similar). Used in ColBERT-style late interaction:

$$\text{MaxSim}(Q, D) = \sum_{i=1}^{|Q|} \max_{j=1}^{|D|} Q_i \cdot D_j$$

## 5. Recall-Latency Tradeoffs (Algorithm Analysis)
### The Problem
Every ANN algorithm trades recall for speed. Understanding this tradeoff mathematically helps tune parameters optimally.

### HNSW: ef_search vs Recall
Empirically, HNSW recall follows a sigmoid-like curve:

$$\text{Recall}(ef) \approx 1 - \exp\!\left(-\frac{ef}{ef^*}\right)$$

where $ef^*$ is the characteristic search width (depends on $M$ and data distribution). Latency scales linearly:

$$\text{Latency}(ef) \propto ef \cdot M \cdot d$$

### IVF: nprobe vs Recall
$$\text{Recall}(nprobe) \approx 1 - \left(1 - \frac{nprobe}{C}\right)^{\gamma}$$

where $\gamma > k$ accounts for cluster overlap. Latency:

$$\text{Latency}(nprobe) = O\!\left(C \cdot d + nprobe \cdot \frac{n}{C} \cdot d\right)$$

The first term (cluster distance) is constant; the second scales with nprobe.

### Pareto Frontier
The recall-latency Pareto frontier for different algorithms:

```
Recall@10 | HNSW Latency | IVF Latency | IVFPQ Latency
0.90      | 0.5 ms       | 0.8 ms      | 0.3 ms
0.95      | 1.0 ms       | 2.0 ms      | 0.8 ms
0.99      | 3.0 ms       | 8.0 ms      | 5.0 ms
0.999     | 10 ms        | 30 ms       | 25 ms

(Approximate, for 1M vectors, d=1536, single core)
```

HNSW dominates at high recall; IVFPQ wins on memory-constrained deployments.

## 6. Dimensionality Reduction (Linear Algebra)
### The Problem
High-dimensional vectors are expensive to store and search. Reducing dimensionality can speed up search with acceptable recall loss.

### PCA Projection
$$\mathbf{x}_{\text{reduced}} = W_{\text{PCA}}^\top (\mathbf{x} - \boldsymbol{\mu})$$

where $W_{\text{PCA}} \in \mathbb{R}^{d \times d'}$ contains the top $d'$ eigenvectors of the covariance matrix.

The fraction of variance retained:

$$\text{Variance ratio} = \frac{\sum_{i=1}^{d'} \lambda_i}{\sum_{i=1}^{d} \lambda_i}$$

For typical text embeddings ($d = 1536$), reducing to $d' = 512$ retains 90-95% of variance while providing $3\times$ speedup in distance computation.

### Matryoshka Representation Learning
OpenAI's text-embedding-3 models support truncation: use only the first $d'$ dimensions.

$$\text{Effective embedding: } \mathbf{x}_{1:d'}$$

This works because Matryoshka training ensures the most important information is in the first dimensions. Recommended truncation points:

$$d' \in \{256, 512, 768, 1024\} \quad \text{for } d = 1536$$

## Prerequisites
- graph-theory (graph traversal, small-world networks, navigability)
- clustering (k-means, Voronoi partitions, centroid computation)
- information-theory (rate-distortion theory, codebook design)
- linear-algebra (inner products, norms, PCA, eigenvalue decomposition)
- probability (expected values, concentration inequalities)
- algorithm-analysis (time complexity, space complexity, amortized analysis)
