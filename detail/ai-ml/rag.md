# The Mathematics of RAG -- Similarity, Ranking, and Embedding Geometry

> *Retrieval-Augmented Generation rests on the geometry of embedding spaces: queries and documents are mapped to high-dimensional vectors where proximity encodes semantic relevance. The mathematics of distance metrics, ranking functions, and information retrieval evaluation form the theoretical backbone of every RAG pipeline.*

---

## 1. Cosine Similarity (Linear Algebra / Geometry)
### The Problem
We need a metric that captures semantic similarity between a query embedding and a document embedding, invariant to the magnitude of the vectors (longer documents should not automatically score higher).

### The Formula
$$\text{cos}(\mathbf{q}, \mathbf{d}) = \frac{\mathbf{q} \cdot \mathbf{d}}{\|\mathbf{q}\| \, \|\mathbf{d}\|} = \frac{\sum_{i=1}^{n} q_i d_i}{\sqrt{\sum_{i=1}^{n} q_i^2} \cdot \sqrt{\sum_{i=1}^{n} d_i^2}}$$

For normalized vectors ($\|\mathbf{q}\| = \|\mathbf{d}\| = 1$):

$$\text{cos}(\mathbf{q}, \mathbf{d}) = \mathbf{q} \cdot \mathbf{d} = \sum_{i=1}^{n} q_i d_i$$

Cosine distance (used by vector databases):

$$d_{\text{cos}}(\mathbf{q}, \mathbf{d}) = 1 - \text{cos}(\mathbf{q}, \mathbf{d})$$

### Worked Example
Given 3-dimensional embeddings (simplified):

$$\mathbf{q} = [0.8, 0.3, 0.5], \quad \mathbf{d} = [0.7, 0.4, 0.6]$$

$$\text{cos}(\mathbf{q}, \mathbf{d}) = \frac{0.8(0.7) + 0.3(0.4) + 0.5(0.6)}{\sqrt{0.64 + 0.09 + 0.25} \cdot \sqrt{0.49 + 0.16 + 0.36}}$$

$$= \frac{0.56 + 0.12 + 0.30}{\sqrt{0.98} \cdot \sqrt{1.01}} = \frac{0.98}{0.9899 \times 1.0050} = \frac{0.98}{0.9949} \approx 0.985$$

High similarity -- these vectors point in nearly the same direction.

## 2. L2 (Euclidean) Distance (Metric Spaces)
### The Problem
An alternative distance metric that captures absolute difference in embedding space, used by some vector databases (FAISS default).

### The Formula
$$d_{L2}(\mathbf{q}, \mathbf{d}) = \|\mathbf{q} - \mathbf{d}\|_2 = \sqrt{\sum_{i=1}^{n}(q_i - d_i)^2}$$

Relationship to cosine similarity for normalized vectors:

$$d_{L2}^2 = 2(1 - \text{cos}(\mathbf{q}, \mathbf{d}))$$

This means for unit vectors, L2 distance and cosine distance are monotonically related -- they produce the same ranking.

### Inner Product vs Cosine vs L2
$$\text{Inner product: } \langle \mathbf{q}, \mathbf{d} \rangle = \|\mathbf{q}\| \cdot \|\mathbf{d}\| \cdot \cos\theta$$

For unnormalized vectors, inner product favors longer vectors (higher-magnitude embeddings). This is why normalization matters: without it, cosine similarity and inner product diverge.

### Worked Example
For the same 3-dimensional embeddings as above:

$$d_{L2} = \sqrt{(0.8-0.7)^2 + (0.3-0.4)^2 + (0.5-0.6)^2} = \sqrt{0.01 + 0.01 + 0.01} = \sqrt{0.03} \approx 0.173$$

Verification via the cosine-L2 relationship (for normalized versions):

$$\hat{\mathbf{q}} = \frac{\mathbf{q}}{\|\mathbf{q}\|} = \frac{[0.8, 0.3, 0.5]}{0.9899}, \quad \hat{\mathbf{d}} = \frac{\mathbf{d}}{\|\mathbf{d}\|} = \frac{[0.7, 0.4, 0.6]}{1.0050}$$

$$d_{L2}^2(\hat{\mathbf{q}}, \hat{\mathbf{d}}) = 2(1 - 0.985) = 0.030, \quad d_{L2} \approx 0.173$$

The values agree, confirming equivalence for normalized vectors.

## 3. Maximal Marginal Relevance (Information Retrieval)
### The Problem
Naive similarity search returns the top-k most similar documents, but these may be near-duplicates of each other, wasting context window space. MMR balances relevance against diversity.

### The Formula
$$\text{MMR} = \arg\max_{d_i \in R \setminus S} \left[\lambda \cdot \text{sim}(d_i, q) - (1 - \lambda) \cdot \max_{d_j \in S} \text{sim}(d_i, d_j)\right]$$

where:
- $R$ is the candidate set (e.g., top-20 from initial retrieval)
- $S$ is the set of already-selected documents
- $\lambda \in [0, 1]$ controls the relevance-diversity tradeoff
- $\lambda = 1$: pure relevance (same as similarity search)
- $\lambda = 0$: pure diversity (maximum dissimilarity from selected docs)

### Worked Example
Given $\lambda = 0.7$, query $q$, and candidates with scores:

| Document | sim(d, q) | max sim(d, S) | MMR Score |
|----------|-----------|---------------|-----------|
| $d_1$    | 0.95      | --            | Selected first |
| $d_2$    | 0.92      | 0.98 (to $d_1$) | $0.7(0.92) - 0.3(0.98) = 0.350$ |
| $d_3$    | 0.85      | 0.40 (to $d_1$) | $0.7(0.85) - 0.3(0.40) = 0.475$ |

$d_3$ wins despite lower relevance because it is more diverse from $d_1$.

## 4. NDCG (Normalized Discounted Cumulative Gain)
### The Problem
Evaluate the quality of a ranked retrieval list, accounting for both relevance grades and the position of relevant documents (higher is better for top positions).

### The Formula
$$\text{DCG}_k = \sum_{i=1}^{k} \frac{2^{rel_i} - 1}{\log_2(i + 1)}$$

$$\text{NDCG}_k = \frac{\text{DCG}_k}{\text{IDCG}_k}$$

where $\text{IDCG}_k$ is the DCG of the ideal ranking (sorting by relevance).

### Worked Example
Retrieved ranking with relevance grades $[3, 2, 0, 1, 3]$:

$$\text{DCG}_5 = \frac{2^3-1}{\log_2 2} + \frac{2^2-1}{\log_2 3} + \frac{2^0-1}{\log_2 4} + \frac{2^1-1}{\log_2 5} + \frac{2^3-1}{\log_2 6}$$

$$= \frac{7}{1} + \frac{3}{1.585} + \frac{0}{2} + \frac{1}{2.322} + \frac{7}{2.585}$$

$$= 7 + 1.893 + 0 + 0.431 + 2.708 = 12.031$$

Ideal ranking $[3, 3, 2, 1, 0]$:

$$\text{IDCG}_5 = \frac{7}{1} + \frac{7}{1.585} + \frac{3}{2} + \frac{1}{2.322} + \frac{0}{2.585} = 7 + 4.416 + 1.5 + 0.431 + 0 = 13.347$$

$$\text{NDCG}_5 = \frac{12.031}{13.347} \approx 0.901$$

## 5. Embedding Space Geometry (Representation Learning)
### The Problem
Understanding the structure of the embedding space helps explain why RAG works and when it fails.

### The Curse of Dimensionality
In high-dimensional spaces ($d \geq 100$), the ratio of maximum to minimum pairwise distances converges:

$$\lim_{d \to \infty} \frac{d_{\max} - d_{\min}}{d_{\min}} \to 0$$

This means all points become roughly equidistant. Trained embedding models counteract this by learning low-dimensional manifold structure within the high-dimensional space.

### Intrinsic Dimensionality
The effective dimensionality of an embedding space is often much lower than the nominal dimension:

$$d_{\text{intrinsic}} = \lim_{\epsilon \to 0} \frac{\log N(\epsilon)}{\log(1/\epsilon)}$$

where $N(\epsilon)$ is the number of $\epsilon$-balls needed to cover the data. For text embeddings with $d = 1536$, the intrinsic dimension is typically $d_{\text{intrinsic}} \approx 20\text{--}50$.

### Hubness Problem
In high dimensions, some points ("hubs") appear as nearest neighbors of many other points disproportionately:

$$N_k(\mathbf{x}) = |\{\mathbf{y} : \mathbf{x} \in \text{kNN}(\mathbf{y})\}|$$

Hub points have $N_k(\mathbf{x}) \gg k$. Centering the embedding matrix and normalizing mitigates hubness:

$$\hat{\mathbf{x}} = \frac{\mathbf{x} - \boldsymbol{\mu}}{\|\mathbf{x} - \boldsymbol{\mu}\|}$$

## 6. Retrieval Evaluation Metrics (Statistics)
### Precision and Recall at k
$$\text{Precision}@k = \frac{|\text{relevant} \cap \text{retrieved}_k|}{k}$$

$$\text{Recall}@k = \frac{|\text{relevant} \cap \text{retrieved}_k|}{|\text{relevant}|}$$

### Mean Average Precision
$$\text{AP} = \frac{1}{|\text{relevant}|}\sum_{k=1}^{n} \text{Precision}@k \cdot \mathbb{1}[\text{doc}_k \text{ is relevant}]$$

$$\text{MAP} = \frac{1}{|Q|}\sum_{q \in Q} \text{AP}(q)$$

### Reciprocal Rank
$$\text{RR} = \frac{1}{\text{rank of first relevant document}}$$

$$\text{MRR} = \frac{1}{|Q|}\sum_{q \in Q} \text{RR}(q)$$

### Worked Example
Given a query with 3 relevant documents in a corpus, and the retrieval system returns 5 documents with relevance labels $[1, 0, 1, 0, 1]$:

$$\text{Precision}@1 = 1/1 = 1.0, \quad \text{Precision}@3 = 2/3 = 0.667, \quad \text{Precision}@5 = 3/5 = 0.6$$

$$\text{Recall}@1 = 1/3 = 0.333, \quad \text{Recall}@3 = 2/3 = 0.667, \quad \text{Recall}@5 = 3/3 = 1.0$$

$$\text{AP} = \frac{1}{3}\left(\frac{1}{1} + \frac{2}{3} + \frac{3}{5}\right) = \frac{1}{3}(1.0 + 0.667 + 0.6) = 0.756$$

$$\text{RR} = \frac{1}{1} = 1.0 \quad \text{(first relevant document at position 1)}$$

## 7. Faithfulness and Relevance Scoring (Information Theory)
### The Problem
RAG evaluation requires measuring both whether the generated answer is faithful to the retrieved context (faithfulness) and whether the answer addresses the user's question (relevance).

### Faithfulness as Entailment
Faithfulness can be decomposed into atomic claims. For each claim $c_i$ in the answer, check if it is entailed by the context $\mathcal{C}$:

$$\text{Faithfulness} = \frac{1}{|C|}\sum_{i=1}^{|C|} \mathbb{1}[c_i \text{ is entailed by } \mathcal{C}]$$

### Answer Relevance via Embedding Similarity
Generate $n$ questions from the answer and measure their similarity to the original question:

$$\text{Relevance} = \frac{1}{n}\sum_{i=1}^{n} \text{cos}(\text{emb}(q), \text{emb}(q_i^{\text{gen}}))$$

This captures whether the answer contains information that would lead someone to ask the original question.

### Context Precision (Ranking Quality)
$$\text{Context Precision}@k = \frac{1}{\min(k, |\text{relevant}|)}\sum_{i=1}^{k} \text{Precision}@i \cdot \mathbb{1}[\text{chunk}_i \text{ is relevant}]$$

This measures whether relevant chunks appear at the top of the retrieval results, which matters because LLMs attend more to early context.

## 8. Chunk Overlap and Information Loss (Signal Processing)
### The Problem
When splitting documents into chunks, information at chunk boundaries can be lost. Overlap mitigates this but increases storage cost.

### The Formula
For a document of length $L$, chunk size $c$, and overlap $o$:

$$n_{\text{chunks}} = \left\lceil\frac{L - o}{c - o}\right\rceil$$

$$\text{Storage overhead} = \frac{n_{\text{chunks}} \times c}{L} = \frac{c}{c - o}$$

For $c = 1000$, $o = 200$: overhead $= 1000/800 = 1.25\times$ (25% more embeddings to store).

The probability that a relevant passage spanning $s$ tokens is fully contained in at least one chunk:

$$P(\text{captured}) = 1 - \max\left(0, \frac{s - c}{s - o}\right)$$

For $s = 300$, $c = 1000$: $P = 1.0$ (fully captured). For $s = 1200$, $c = 1000$, $o = 200$: the passage spans chunks but the overlap ensures no gap exceeds $c - o = 800$ tokens.

## Prerequisites
- linear-algebra (dot product, norms, matrix operations)
- probability (conditional probability, Bayes' theorem)
- information-theory (entropy, mutual information)
- metric-spaces (distance functions, triangle inequality)
- statistics (precision, recall, F1, hypothesis testing)
