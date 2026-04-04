# Vector Databases (Embedding Storage, Indexing, and Retrieval)

A practical guide to vector databases covering ANN indexing algorithms (HNSW, IVF, PQ, ScaNN), database comparison (Pinecone, Weaviate, Milvus, Qdrant, Chroma, pgvector), distance metrics, hybrid search, filtering strategies, batch ingestion, and production deployment patterns.

## Vector Database Comparison
### Feature Matrix
```
| Database   | Type       | Language | ANN Index        | Filtering | Hybrid | Cloud     |
|-----------|------------|----------|------------------|-----------|--------|-----------|
| Pinecone   | Managed    | —        | Proprietary      | Yes       | Yes    | Yes       |
| Weaviate   | Self/Cloud | Go       | HNSW             | Yes       | Yes    | Yes       |
| Milvus     | Self/Cloud | Go/C++   | HNSW, IVF, PQ    | Yes       | Yes    | Zilliz    |
| Qdrant     | Self/Cloud | Rust     | HNSW             | Yes       | Yes    | Yes       |
| Chroma     | Embedded   | Python   | HNSW             | Yes       | No     | No        |
| pgvector   | Extension  | C        | HNSW, IVF        | Yes (SQL) | Yes    | Via PG    |
| FAISS      | Library    | C++/Py   | IVF, PQ, HNSW    | No        | No     | No        |
| Weaviate   | Self/Cloud | Go       | HNSW             | Yes       | Yes    | Yes       |
```

### When to Use What
```
| Scenario                        | Recommendation        | Reason                          |
|--------------------------------|-----------------------|---------------------------------|
| Prototype / small scale (<100K) | Chroma                | Zero config, in-process         |
| PostgreSQL already in stack     | pgvector              | No new infrastructure           |
| Production, managed service     | Pinecone or Qdrant    | Operational simplicity          |
| Large scale (>10M vectors)      | Milvus or Qdrant      | Distributed, horizontal scaling |
| Maximum performance control     | FAISS                 | Fine-grained index tuning       |
| Multi-modal (text + images)     | Weaviate              | Built-in vectorizer modules     |
```

## ANN Indexing Algorithms
### HNSW (Hierarchical Navigable Small World)
```
# HNSW builds a multi-layer graph where:
# - Layer 0: All vectors connected to ~M neighbors
# - Layer 1: Subset of vectors (skip connections)
# - Layer L: Very few vectors (entry points)
#
# Search: start at top layer, greedily traverse to nearest,
# drop to next layer, repeat until layer 0

# Key parameters:
# M             = max connections per node (default: 16)
# ef_construction = search width during build (default: 200)
# ef_search     = search width during query (default: 100)

# Tradeoffs:
# Higher M       -> better recall, more memory, slower build
# Higher ef      -> better recall, slower query
# Typical sweet spot: M=16, ef_construction=200, ef_search=100
```

### IVF (Inverted File Index)
```
# IVF partitions vectors into nlist clusters using k-means
# At query time, only nprobe nearest clusters are searched

# Key parameters:
# nlist  = number of clusters (sqrt(n) to 4*sqrt(n))
# nprobe = clusters to search (1 to nlist, tradeoff speed/recall)

# Example: 1M vectors
# nlist = 1024, nprobe = 32 -> search ~3% of data
# nlist = 4096, nprobe = 64 -> search ~1.5% of data
```

### PQ (Product Quantization)
```
# PQ compresses vectors by splitting into subvectors and
# quantizing each independently with k-means

# 1536-dim vector split into 192 subvectors of 8 dims each
# Each subvector quantized to 8-bit centroid ID
# Compressed size: 192 bytes (vs 6144 bytes for FP32)
# Compression ratio: ~32x

# Key parameters:
# M (subquantizers) = number of subvectors
# nbits            = bits per subquantizer (typically 8)
# Memory per vector: M * nbits / 8 bytes
```

## Chroma (Local/Embedded)
### Setup and Usage
```python
import chromadb

# Persistent client (data survives restarts)
client = chromadb.PersistentClient(path="./chroma_db")

# Create collection with custom distance metric
collection = client.get_or_create_collection(
    name="documents",
    metadata={
        "hnsw:space": "cosine",          # cosine, l2, or ip
        "hnsw:M": 16,
        "hnsw:construction_ef": 200,
        "hnsw:search_ef": 100,
    },
)

# Add documents (auto-embeds if embedding_function is set)
collection.add(
    ids=["doc1", "doc2", "doc3"],
    documents=["First document text", "Second doc", "Third doc"],
    metadatas=[
        {"source": "wiki", "year": 2024},
        {"source": "arxiv", "year": 2023},
        {"source": "wiki", "year": 2024},
    ],
)

# Query with metadata filtering
results = collection.query(
    query_texts=["search query"],
    n_results=5,
    where={"source": "wiki"},
    where_document={"$contains": "specific term"},
)

# Query with embeddings directly
results = collection.query(
    query_embeddings=[[0.1, 0.2, ...]],
    n_results=10,
    include=["documents", "metadatas", "distances"],
)
```

## Qdrant
### Setup and Usage
```python
from qdrant_client import QdrantClient
from qdrant_client.models import (
    Distance, VectorParams, PointStruct,
    Filter, FieldCondition, MatchValue, Range,
)

# Connect (local or cloud)
client = QdrantClient(url="http://localhost:6333")
# client = QdrantClient(url="https://xxx.cloud.qdrant.io", api_key="...")

# Create collection
client.create_collection(
    collection_name="documents",
    vectors_config=VectorParams(
        size=1536,
        distance=Distance.COSINE,
    ),
    hnsw_config={"m": 16, "ef_construct": 200},
)

# Upsert points
client.upsert(
    collection_name="documents",
    points=[
        PointStruct(id=1, vector=[0.1, 0.2, ...],
                    payload={"source": "wiki", "year": 2024}),
        PointStruct(id=2, vector=[0.3, 0.4, ...],
                    payload={"source": "arxiv", "year": 2023}),
    ],
)

# Search with filtering
results = client.search(
    collection_name="documents",
    query_vector=[0.15, 0.25, ...],
    limit=5,
    query_filter=Filter(
        must=[
            FieldCondition(key="source", match=MatchValue(value="wiki")),
            FieldCondition(key="year", range=Range(gte=2023)),
        ]
    ),
)
```

### Docker Deployment
```bash
docker run -p 6333:6333 -p 6334:6334 \
    -v $(pwd)/qdrant_storage:/qdrant/storage \
    qdrant/qdrant
```

## pgvector (PostgreSQL)
### Setup
```sql
-- Install extension
CREATE EXTENSION vector;

-- Create table
CREATE TABLE documents (
    id BIGSERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    embedding vector(1536),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert vectors
INSERT INTO documents (content, embedding, metadata)
VALUES (
    'Document text here',
    '[0.1, 0.2, ...]'::vector,
    '{"source": "wiki", "year": 2024}'::jsonb
);
```

### Indexing
```sql
-- HNSW index (recommended for most cases)
CREATE INDEX ON documents
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- IVF index (faster build, lower recall)
CREATE INDEX ON documents
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 1000);

-- Set search parameters
SET hnsw.ef_search = 100;
SET ivfflat.probes = 32;
```

### Queries
```sql
-- Cosine similarity search (<=>) or L2 distance (<->)
SELECT id, content,
       1 - (embedding <=> '[0.15, 0.25, ...]'::vector) AS similarity
FROM documents
WHERE metadata->>'source' = 'wiki'
ORDER BY embedding <=> '[0.15, 0.25, ...]'::vector
LIMIT 5;
```

## Pinecone (Managed)
```python
from pinecone import Pinecone, ServerlessSpec
pc = Pinecone(api_key="your-api-key")

pc.create_index(name="documents", dimension=1536, metric="cosine",
                spec=ServerlessSpec(cloud="aws", region="us-east-1"))
index = pc.Index("documents")

# Upsert and query
index.upsert(vectors=[
    {"id": "doc1", "values": [0.1, 0.2, ...],
     "metadata": {"source": "wiki", "year": 2024}},
])
results = index.query(vector=[0.15, 0.25, ...], top_k=5,
                       filter={"source": {"$eq": "wiki"}}, include_metadata=True)
```

## FAISS (Library)
```python
import faiss

d, n = 1536, 1_000_000

# HNSW index
index_hnsw = faiss.IndexHNSWFlat(d, 32)  # M=32
index_hnsw.hnsw.efConstruction = 200
index_hnsw.hnsw.efSearch = 100
index_hnsw.add(vectors)

# IVF + PQ (memory efficient for large scale)
quantizer = faiss.IndexFlatL2(d)
index_ivfpq = faiss.IndexIVFPQ(quantizer, d, nlist=1024, m=192, nbits=8)
index_ivfpq.train(vectors)
index_ivfpq.add(vectors)
index_ivfpq.nprobe = 32

D, I = index_hnsw.search(query_vectors, k=5)  # distances, indices
```

## Distance Metrics
### Comparison
```
| Metric           | Formula                  | Range     | Normalized? | When to Use        |
|-----------------|--------------------------|-----------|-------------|---------------------|
| Cosine           | 1 - cos(a,b)            | [0, 2]    | Yes         | Text embeddings     |
| L2 (Euclidean)   | ||a - b||_2              | [0, inf)  | No          | Image embeddings    |
| Inner Product    | -a . b                   | (-inf,inf)| No          | Normalized = cosine |
| Dot Product      | a . b                    | (-inf,inf)| No          | MaxSim, ColBERT     |
```

## Hybrid Search (Dense + Sparse)
### Reciprocal Rank Fusion
```python
def reciprocal_rank_fusion(results_lists, k=60):
    """Combine multiple ranked result lists using RRF."""
    scores = {}
    for results in results_lists:
        for rank, doc_id in enumerate(results):
            if doc_id not in scores:
                scores[doc_id] = 0
            scores[doc_id] += 1.0 / (k + rank + 1)

    return sorted(scores.items(), key=lambda x: x[1], reverse=True)

# Example: combine BM25 and vector search results
dense_results = vector_search(query, k=20)
sparse_results = bm25_search(query, k=20)
hybrid_results = reciprocal_rank_fusion([dense_results, sparse_results])
```

## Batch Ingestion
```python
# Chroma batch upsert
BATCH_SIZE = 1000
for i in range(0, len(documents), BATCH_SIZE):
    batch = documents[i:i+BATCH_SIZE]
    collection.upsert(
        ids=[d["id"] for d in batch],
        embeddings=[d["embedding"] for d in batch],
        metadatas=[d["metadata"] for d in batch],
    )
```

## Tips
- Start with Chroma for prototyping and pgvector if PostgreSQL is already in your stack -- avoid premature infrastructure
- HNSW is the default choice for most workloads: excellent recall/speed tradeoff with tunable parameters
- Set HNSW ef_search higher than your top-k (at least 2x) to get good recall -- ef_search=100 for top-10
- Always normalize embeddings before insertion if using cosine distance -- some databases do this automatically, some do not
- Pre-filter before vector search when possible: filtering after ANN search can return fewer results than expected
- Use IVF+PQ for datasets exceeding available RAM -- PQ compresses vectors 10-30x at modest recall cost
- Batch your inserts: vector databases are optimized for bulk upserts, not individual writes
- Monitor recall on a test set when tuning index parameters -- high throughput with low recall is useless
- Hybrid search (dense + sparse via RRF) typically outperforms either method alone by 5-15% on retrieval benchmarks
- For pgvector, always create the index after bulk loading -- index-then-insert is significantly slower
- Qdrant and Milvus support payload-based filtering that runs inside the HNSW traversal, not as a post-filter
- Dimension matters: 1536 (OpenAI) works well, but 384-768 (sentence-transformers) is often sufficient and 2-4x faster

## See Also
- rag, llm-fundamentals, transformers, prompt-engineering

## References
- [Pinecone Documentation](https://docs.pinecone.io/)
- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [Milvus Documentation](https://milvus.io/docs)
- [pgvector GitHub](https://github.com/pgvector/pgvector)
- [Chroma Documentation](https://docs.trychroma.com/)
- [FAISS Wiki](https://github.com/facebookresearch/faiss/wiki)
- [HNSW Paper (Malkov & Yashunin, 2018)](https://arxiv.org/abs/1603.09320)
