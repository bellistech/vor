# The Mathematics of Elasticsearch — Relevance Scoring and Distributed Search

> *Elasticsearch ranks documents using TF-IDF and BM25 scoring, distributes data across shards using hash-based routing, and coordinates searches with scatter-gather. The mathematics cover text relevance, shard sizing, merge policies, and aggregation accuracy.*

---

## 1. BM25 Relevance Scoring (Information Retrieval)

### The Problem

Elasticsearch uses BM25 (Best Match 25) to score documents against search queries. BM25 improves on TF-IDF by introducing term frequency saturation and document length normalization.

### The Formula

$$\text{BM25}(D, Q) = \sum_{i=1}^{n} \text{IDF}(q_i) \cdot \frac{f(q_i, D) \cdot (k_1 + 1)}{f(q_i, D) + k_1 \cdot \left(1 - b + b \cdot \frac{|D|}{\text{avgdl}}\right)}$$

Where:
- $f(q_i, D)$ = term frequency of $q_i$ in document $D$
- $|D|$ = document length (in tokens)
- $\text{avgdl}$ = average document length across the index
- $k_1 = 1.2$ (term frequency saturation, default)
- $b = 0.75$ (length normalization, default)

Inverse Document Frequency:

$$\text{IDF}(q_i) = \ln\left(1 + \frac{N - n(q_i) + 0.5}{n(q_i) + 0.5}\right)$$

Where $N$ = total documents, $n(q_i)$ = documents containing term $q_i$.

### Worked Example

Index: 10,000 documents, average length 200 tokens. Query: "wireless headphones". Document D has length 150, "wireless" appears 3 times, "headphones" appears 2 times. "wireless" in 500 docs, "headphones" in 200 docs.

$$\text{IDF}_{\text{wireless}} = \ln\left(1 + \frac{10000 - 500 + 0.5}{500 + 0.5}\right) = \ln(1 + 18.99) = 3.00$$

$$\text{IDF}_{\text{headphones}} = \ln\left(1 + \frac{10000 - 200 + 0.5}{200 + 0.5}\right) = \ln(1 + 48.88) = 3.91$$

Term score for "wireless":

$$S_w = 3.00 \times \frac{3 \times 2.2}{3 + 1.2 \times (1 - 0.75 + 0.75 \times \frac{150}{200})} = 3.00 \times \frac{6.6}{3 + 1.2 \times 0.8125}$$

$$= 3.00 \times \frac{6.6}{3.975} = 3.00 \times 1.661 = 4.98$$

---

## 2. Shard Routing (Consistent Hashing)

### The Problem

Documents are assigned to shards deterministically so that gets and puts target the correct shard without a lookup table.

### The Formula

$$\text{shard} = \text{hash}(\text{routing}) \bmod \text{number\_of\_primary\_shards}$$

Default routing = document `_id`. The hash function is MurmurHash3.

This is why shard count cannot change after index creation: it would change the modulo mapping.

### Shard Sizing

Recommended shard size: 10-50 GB. For a dataset of size $S$:

$$\text{shards} = \left\lceil \frac{S}{\text{target shard size}} \right\rceil$$

| Dataset | Target 30GB/shard | Shards Needed |
|:---:|:---:|:---:|
| 50 GB | 30 GB | 2 |
| 300 GB | 30 GB | 10 |
| 1 TB | 30 GB | 34 |
| 10 TB | 30 GB | 342 |

Total shards (with replicas):

$$\text{total shards} = P \times (1 + R)$$

Where $P$ = primaries, $R$ = replicas. 10 primaries with 1 replica = 20 total shards.

---

## 3. Scatter-Gather Search (Distributed Query)

### The Problem

A search query is broadcast to all relevant shards (scatter), each returns top-K results locally, and a coordinating node merges them (gather).

### The Formula

Query latency (assuming parallel shard execution):

$$L_{\text{query}} = L_{\text{coord}} + \max_{i=1}^{P}(L_{\text{shard}_i}) + L_{\text{merge}}$$

Merge cost for $P$ shards each returning $K$ results:

$$C_{\text{merge}} = O(P \cdot K \cdot \log(P \cdot K))$$

Network transfer:

$$B_{\text{scatter}} = P \times S_{\text{query}}$$

$$B_{\text{gather}} = P \times K \times S_{\text{result}}$$

### Worked Example

10 shards, top-100, each result ~500 bytes:

$$B_{\text{gather}} = 10 \times 100 \times 500 = 500 \text{ KB}$$

Merge: sort 1,000 items to find global top-100:

$$C_{\text{merge}} = O(1000 \times \log(1000)) \approx 10{,}000 \text{ comparisons}$$

---

## 4. Lucene Segment Merging (Log-Structured Merge)

### The Problem

Elasticsearch (via Lucene) writes immutable segments that are periodically merged. The merge policy affects both write performance and search speed.

### The Formula

Tiered merge policy merges segments when $n$ segments of similar size exist:

$$\text{Merge triggered when: } |\{s : s_{\text{size}} \in [\text{floor}, \text{ceil}]\}| \geq \text{max\_merge\_at\_once}$$

Write amplification from merging (similar to LSM trees):

$$W_{\text{amplification}} = O\left(\frac{\text{max\_merge\_at\_once}}{\text{segments\_per\_tier}} \times \log_{\text{merge factor}}(N)\right)$$

Default: max_merge_at_once = 10, segments_per_tier = 10:

$$W \approx 1.0 \times \log_{10}(N)$$

### Segment Count Impact on Search

Search must check every segment. Query time scales linearly with segment count:

$$L_{\text{search}} \propto \sum_{i=1}^{k} \log(n_i) + k \times C_{\text{open}}$$

This is why force-merging to 1 segment improves search performance on read-only indices.

---

## 5. Aggregation Accuracy (Approximate Counting)

### The Problem

Terms aggregations return approximate counts. The `size` parameter and shard-level truncation can cause inaccuracy.

### The Formula

Each shard returns its local top-$K$ terms. Error occurs when a globally frequent term is not in a shard's local top-$K$:

$$\text{Error bound} = \sum_{i=K+1}^{N_{\text{unique}}} f_i$$

Where $f_i$ is the frequency of the $i$-th most frequent term on that shard.

For uniformly distributed terms across $P$ shards:

$$P(\text{term in global top-K but missing from shard}) = \left(1 - \frac{K}{N_{\text{unique per shard}}}\right)^{P}$$

### Shard Size Heuristic

To achieve 99% accuracy, request `shard_size`:

$$\text{shard\_size} = K \times 1.5 + 10$$

Default: `shard_size = size * 1.5 + 10`.

---

## 6. Index Lifecycle Cost (Storage Tiers)

### The Problem

ILM moves indices through hot/warm/cold/frozen tiers. Each tier has different cost and performance characteristics.

### The Formula

Total storage cost for a time-series index with daily volume $V$, retention $D$ days:

$$C_{\text{total}} = \sum_{t \in \text{tiers}} V \times D_t \times (1 + R_t) \times C_t \times \frac{1}{\text{compression}_t}$$

Where $D_t$ is days in tier $t$, $R_t$ is replica count, $C_t$ is cost per GB.

### Worked Example

10 GB/day, 1 year retention:

| Tier | Days | Replicas | Compression | Cost/GB/mo | Monthly Cost |
|:---:|:---:|:---:|:---:|:---:|:---:|
| Hot | 7 | 1 | 1x | $0.10 | $14.00 |
| Warm | 23 | 1 | 1.5x | $0.05 | $15.33 |
| Cold | 60 | 0 | 2x | $0.02 | $6.00 |
| Frozen | 275 | 0 | 3x | $0.005 | $4.58 |

$$C_{\text{total}} = 14.00 + 15.33 + 6.00 + 4.58 = \$39.91/\text{month}$$

Without ILM (all hot, 1 replica):

$$C_{\text{no ILM}} = 10 \times 365 \times 2 \times 0.10 = \$730/\text{month}$$

Savings: 94.5%.

---

## 7. Inverted Index Space (Posting Lists)

### The Problem

Elasticsearch stores an inverted index mapping terms to document IDs. The space complexity depends on vocabulary size and posting list lengths.

### The Formula

Inverted index size:

$$S_{\text{index}} = |V| \times S_{\text{term}} + \sum_{t \in V} |\text{postings}(t)| \times S_{\text{posting}}$$

Where $|V|$ is vocabulary size. With variable-byte encoding, posting entry ~2-4 bytes.

Heap's Law for vocabulary growth:

$$|V| = K \cdot N^{\beta}, \quad K \approx 30\text{-}100, \quad \beta \approx 0.4\text{-}0.6$$

### Worked Example

Corpus: 1M documents, 10B tokens, $\beta = 0.5$, $K = 50$:

$$|V| = 50 \times (10^{10})^{0.5} = 50 \times 100{,}000 = 5{,}000{,}000 \text{ unique terms}$$

$$S_{\text{index}} \approx 5 \times 10^6 \times 20 + 10^{10} \times 3 \approx 100 \text{ MB} + 30 \text{ GB} = 30.1 \text{ GB}$$

---

## Prerequisites

- information-retrieval, probability, distributed-systems, data-structures
