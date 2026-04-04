# RAG (Retrieval-Augmented Generation)

A practical guide to building RAG pipelines that ground LLM responses in external knowledge, covering embedding models, chunking strategies, vector stores, retrieval methods, reranking, evaluation metrics, and integration with LangChain and LlamaIndex.

## RAG Pipeline Overview
### Architecture
```
User Query
  -> Query Embedding (embed model)
  -> Vector Search (similarity retrieval)
  -> [Optional] Reranking (cross-encoder)
  -> Context Assembly (top-k chunks)
  -> LLM Generation (query + context -> answer)
  -> [Optional] Citation Extraction
```

### Minimal RAG with LangChain
```python
from langchain_community.document_loaders import PyPDFLoader
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain_openai import OpenAIEmbeddings, ChatOpenAI
from langchain_community.vectorstores import Chroma
from langchain.chains import RetrievalQA

# 1. Load documents
loader = PyPDFLoader("technical_doc.pdf")
docs = loader.load()

# 2. Chunk
splitter = RecursiveCharacterTextSplitter(
    chunk_size=1000,
    chunk_overlap=200,
    separators=["\n\n", "\n", ". ", " ", ""]
)
chunks = splitter.split_documents(docs)

# 3. Embed and store
embeddings = OpenAIEmbeddings(model="text-embedding-3-small")
vectorstore = Chroma.from_documents(chunks, embeddings,
                                     persist_directory="./chroma_db")

# 4. Retrieve and generate
llm = ChatOpenAI(model="gpt-4", temperature=0)
qa_chain = RetrievalQA.from_chain_type(
    llm=llm,
    retriever=vectorstore.as_retriever(search_kwargs={"k": 5}),
    return_source_documents=True,
)

result = qa_chain.invoke({"query": "What is the main finding?"})
print(result["result"])
```

## Embedding Models
### Model Comparison
```
| Model                        | Dims | Max Tokens | MTEB Score |
|------------------------------|------|------------|------------|
| text-embedding-3-small       | 1536 | 8191       | 62.3       |
| text-embedding-3-large       | 3072 | 8191       | 64.6       |
| embed-english-v3.0 (Cohere)  | 1024 | 512        | 64.5       |
| all-MiniLM-L6-v2             | 384  | 256        | 56.3       |
| bge-large-en-v1.5            | 1024 | 512        | 64.2       |
| e5-mistral-7b-instruct       | 4096 | 32768      | 66.6       |
| nomic-embed-text-v1.5        | 768  | 8192       | 62.3       |
```

### sentence-transformers
```python
from sentence_transformers import SentenceTransformer

model = SentenceTransformer("BAAI/bge-large-en-v1.5")

# Encode documents
doc_embeddings = model.encode(
    ["Document 1 text", "Document 2 text"],
    normalize_embeddings=True,
    show_progress_bar=True,
    batch_size=32,
)

# Encode query (BGE models need instruction prefix)
query_embedding = model.encode(
    ["Represent this sentence for searching: What is RAG?"],
    normalize_embeddings=True,
)
```

## Chunking Strategies
### Fixed-Size Chunking
```python
from langchain.text_splitter import CharacterTextSplitter

splitter = CharacterTextSplitter(
    chunk_size=500,
    chunk_overlap=50,
    separator="\n"
)
chunks = splitter.split_text(document_text)
```

### Recursive Character Splitting (Recommended)
```python
from langchain.text_splitter import RecursiveCharacterTextSplitter

splitter = RecursiveCharacterTextSplitter(
    chunk_size=1000,
    chunk_overlap=200,
    separators=["\n\n", "\n", ". ", " ", ""],
    length_function=len,
)
```

### Semantic Chunking
```python
from langchain_experimental.text_splitter import SemanticChunker
from langchain_openai import OpenAIEmbeddings

splitter = SemanticChunker(
    OpenAIEmbeddings(),
    breakpoint_threshold_type="percentile",
    breakpoint_threshold_amount=95,
)
chunks = splitter.split_text(document_text)
```

### Chunk Size Guidelines
```
| Content Type    | Chunk Size | Overlap | Rationale                   |
|----------------|------------|---------|------------------------------|
| General text    | 1000       | 200     | Good for most use cases      |
| Legal docs      | 1500       | 300     | Preserve clause context      |
| Code            | 500        | 100     | Function-level granularity   |
| Q&A pairs       | 200        | 0       | Each pair is self-contained  |
| Technical specs | 800        | 150     | Balance detail and context   |
```

## Vector Store Setup
### Chroma (Local)
```python
import chromadb

client = chromadb.PersistentClient(path="./chroma_db")
collection = client.get_or_create_collection(
    name="documents",
    metadata={"hnsw:space": "cosine"},
)

collection.add(
    ids=["doc1", "doc2"],
    documents=["First document", "Second document"],
    embeddings=[[0.1, 0.2, ...], [0.3, 0.4, ...]],
    metadatas=[{"source": "file1"}, {"source": "file2"}],
)

results = collection.query(
    query_embeddings=[[0.15, 0.25, ...]],
    n_results=5,
    where={"source": "file1"},
)
```

### pgvector (PostgreSQL)
```sql
-- Enable extension
CREATE EXTENSION vector;

-- Create table with vector column
CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    content TEXT,
    embedding vector(1536),
    metadata JSONB
);

-- Create HNSW index
CREATE INDEX ON documents
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- Similarity search
SELECT content, 1 - (embedding <=> query_embedding) AS similarity
FROM documents
ORDER BY embedding <=> query_embedding
LIMIT 5;
```

## Retrieval Methods
### Similarity Search vs MMR
```python
# Basic similarity search
retriever = vectorstore.as_retriever(
    search_type="similarity",
    search_kwargs={"k": 5},
)

# MMR (Maximal Marginal Relevance) - reduces redundancy
retriever = vectorstore.as_retriever(
    search_type="mmr",
    search_kwargs={"k": 5, "fetch_k": 20, "lambda_mult": 0.7},
)

# Similarity with score threshold
retriever = vectorstore.as_retriever(
    search_type="similarity_score_threshold",
    search_kwargs={"score_threshold": 0.7, "k": 10},
)
```

### Hybrid Search (Dense + Sparse)
```python
from langchain.retrievers import EnsembleRetriever
from langchain_community.retrievers import BM25Retriever

# Sparse retriever (BM25)
bm25 = BM25Retriever.from_documents(documents, k=5)

# Dense retriever (vector)
dense = vectorstore.as_retriever(search_kwargs={"k": 5})

# Ensemble with reciprocal rank fusion
hybrid = EnsembleRetriever(
    retrievers=[bm25, dense],
    weights=[0.4, 0.6],
)
```

## Reranking
### Cross-Encoder Reranking
```python
from langchain.retrievers import ContextualCompressionRetriever
from langchain.retrievers.document_compressors import CrossEncoderReranker
from langchain_community.cross_encoders import HuggingFaceCrossEncoder

# Load cross-encoder
cross_encoder = HuggingFaceCrossEncoder(model_name="cross-encoder/ms-marco-MiniLM-L-6-v2")
compressor = CrossEncoderReranker(model=cross_encoder, top_n=3)

# Wrap retriever with reranker
reranking_retriever = ContextualCompressionRetriever(
    base_compressor=compressor,
    base_retriever=vectorstore.as_retriever(search_kwargs={"k": 20}),
)
```

### Cohere Rerank
```python
from langchain_cohere import CohereRerank

compressor = CohereRerank(
    model="rerank-english-v3.0",
    top_n=5,
)
```

## Evaluation
### RAGAS Framework
```python
from ragas import evaluate
from ragas.metrics import (
    faithfulness,
    answer_relevancy,
    context_precision,
    context_recall,
)
from datasets import Dataset

eval_dataset = Dataset.from_dict({
    "question": ["What is RAG?"],
    "answer": ["RAG combines retrieval with generation..."],
    "contexts": [["RAG is a technique that..."]],
    "ground_truth": ["RAG stands for Retrieval-Augmented Generation..."],
})

results = evaluate(
    eval_dataset,
    metrics=[faithfulness, answer_relevancy,
             context_precision, context_recall],
)
print(results)
# {'faithfulness': 0.92, 'answer_relevancy': 0.88,
#  'context_precision': 0.85, 'context_recall': 0.90}
```

## LlamaIndex Integration
```python
from llama_index.core import VectorStoreIndex, SimpleDirectoryReader
from llama_index.core import Settings
from llama_index.llms.openai import OpenAI
from llama_index.embeddings.openai import OpenAIEmbedding

Settings.llm = OpenAI(model="gpt-4", temperature=0)
Settings.embed_model = OpenAIEmbedding(model="text-embedding-3-small")

documents = SimpleDirectoryReader("./data").load_data()
index = VectorStoreIndex.from_documents(documents)

query_engine = index.as_query_engine(similarity_top_k=5)
response = query_engine.query("What are the key findings?")
print(response)
```

## Tips
- Start with chunk_size=1000 and chunk_overlap=200 as a baseline, then tune based on retrieval quality
- Always normalize embeddings when using cosine similarity -- unnormalized vectors give wrong rankings
- Use hybrid search (BM25 + dense) for domains with specific terminology (legal, medical, technical)
- Reranking with a cross-encoder on top-20 results typically improves precision by 10-20%
- Add metadata filtering (date, source, category) to reduce the search space before vector similarity
- Evaluate with RAGAS before deploying -- faithfulness below 0.8 means the LLM is hallucinating over your docs
- For large corpora (>100K chunks), use HNSW indexing -- brute-force search becomes impractical
- Embed queries and documents with the same model -- mixing embedding models produces meaningless similarities
- Use text-embedding-3-small for cost efficiency; upgrade to large only if retrieval quality demands it
- Cache embeddings aggressively -- re-embedding unchanged documents wastes money and time
- Test your chunking strategy with real queries: if retrieved chunks lack the answer, no amount of LLM skill helps

## See Also
- vector-databases, prompt-engineering, llm-fundamentals, raft, transformers

## References
- [LangChain RAG Tutorial](https://python.langchain.com/docs/tutorials/rag/)
- [LlamaIndex Documentation](https://docs.llamaindex.ai/en/stable/)
- [RAGAS Evaluation Framework](https://docs.ragas.io/en/latest/)
- [Chunking Strategies for LLM Applications](https://www.pinecone.io/learn/chunking-strategies/)
- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings)
- [Cohere Rerank Documentation](https://docs.cohere.com/docs/reranking)
