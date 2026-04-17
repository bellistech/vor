# The Mathematics of Tries -- Prefix Trees, Radix Compression, and Amortised String Search

> *A trie is a DAG-shaped automaton for string membership: each path from root to terminal spells one key. Unlike hash tables that pay $O(L)$ per probe to compute a hash and then $O(L)$ per probe to compare, tries absorb hashing into the walk itself, giving $O(L)$ per operation with zero hash collisions. The trade-off is space: a naive trie allocates a child array per node, which radix compression and DAWG minimisation reduce to optimal.*

---

## 1. The Trie Structure

### Formal Definition

A **trie** over alphabet $\Sigma$ is a rooted tree $T = (V, E, r, F, \lambda)$ where:

- $V$ is a set of nodes
- $E \subseteq V \times \Sigma \times V$ is a set of labeled edges with unique labels per source node
- $r \in V$ is the root
- $F \subseteq V$ is the set of terminal (accepting) nodes
- $\lambda : E \to \Sigma$ is the edge-labeling function

The set of keys stored in $T$ is:

$$\text{Keys}(T) = \{s_1 s_2 \cdots s_k \in \Sigma^* : \exists v_0 \to v_1 \to \cdots \to v_k, \, v_0 = r, \, v_k \in F, \, \lambda(v_{i-1}, v_i) = s_i\}$$

Every string is represented by the unique path from root to its terminal node. Prefixes of stored keys correspond to initial subpaths.

### Path Uniqueness

**Invariant**: For any node $v$, all keys passing through $v$ share the prefix spelled from root to $v$. Hence, the subtree rooted at $v$ contains exactly the keys with that prefix.

---

## 2. Complexity Analysis

### Time per Operation

For a key of length $L$:

- **Insert**: walk $L$ edges, allocate up to $L$ nodes. $O(L)$ time, $O(L)$ worst-case space.
- **Search**: walk $L$ edges, check $\text{isEnd}$. $O(L)$ time.
- **StartsWith**: walk $L$ edges. $O(L)$ time.

Each edge step is $O(1)$ with array-based children; $O(\log \sigma)$ or $O(1)$ amortised with hash-based children.

### Space Complexity

Let $N$ be the number of keys and $L_{\max}$ the maximum key length. The naive upper bound is:

$$S_{\text{trie}} \leq N \cdot L_{\max} \cdot \sigma \cdot w$$

where $\sigma = |\Sigma|$ and $w$ is the pointer size. In practice, many nodes are shared among keys with common prefixes, so actual space depends on the **total number of distinct prefixes**:

$$S_{\text{actual}} = |\text{Prefixes}(K)| \cdot \sigma \cdot w$$

For English words with average shared prefixes, this is typically 0.3–0.5 of the naive bound.

### Comparison with Hash Tables

Hash table lookup: $O(L)$ for hashing + $O(L)$ for comparison on hit = $O(L)$ expected.

Trie lookup: $O(L)$ guaranteed, no collisions, with the bonus that unsuccessful lookups can terminate early at the first missing edge.

**Theorem**: For a set of keys $K$ with total length $T$, a trie uses $O(T)$ space (amortised across all keys sharing prefixes), while a hash table uses $O(T + N \log(1/\epsilon))$ where $\epsilon$ is the acceptable collision rate.

---

## 3. Radix / Patricia Tree Compression

### The Compression Rule

In a basic trie, any node with exactly one child and no $\text{isEnd}$ flag wastes space: its child could be merged with it by concatenating the edge labels.

A **radix tree** (or PATRICIA trie) stores variable-length edge labels, collapsing chains of single-child nodes into one compact edge.

Formally: an edge in a radix tree is labeled with a string $s \in \Sigma^+$, and the invariant is:

$$\forall v \in V : \text{deg}_{\text{out}}(v) \neq 1 \text{ or } v \in F$$

### Space Reduction

For $N$ keys over alphabet $\sigma$, a naive trie uses $O(T)$ nodes where $T$ is total length. A radix tree uses at most $2N - 1$ nodes regardless of $T$, since each key contributes at most one "branch" point and one terminal.

**Theorem**: For any set of keys, the radix tree has at most $2N - 1$ nodes.

**Proof sketch**: Each internal non-root node is a branch point (≥ 2 children). With $N$ leaves, a binary tree has $N - 1$ internal nodes. General $\sigma$-ary trees have fewer internal nodes, so the bound is $\leq 2N - 1$.

### IP Routing Application

Longest-prefix-match IP routing uses a binary radix tree on 32-bit (IPv4) or 128-bit (IPv6) addresses. Edge labels are bit strings; internal nodes represent subnet prefixes; terminal nodes store next-hop information.

Lookup time is $O(L)$ where $L$ is address length — $O(32)$ or $O(128)$ — effectively constant. Modern FIB implementations use compressed variants (LC-trie, Lulea algorithm, DIR-24-8) for hardware efficiency.

---

## 4. Suffix Automata and DAWGs

### Directed Acyclic Word Graph

A **DAWG** (Directed Acyclic Word Graph) is a minimal DFA recognising a finite set of strings. It extends the trie by merging nodes with identical suffix behaviour, producing a DAG instead of a tree.

### Minimisation

Two nodes $u, v$ are **equivalent** if they recognise the same set of suffix strings. A DAWG merges all equivalent nodes.

**Hopcroft's Algorithm** minimises a DFA in $O(n \sigma \log n)$ time. For lexicon DAWG construction, online algorithms (Daciuk et al., 2000) achieve $O(T)$ time for sorted input, where $T$ is total key length.

### Space Advantage

DAWGs exploit shared *suffixes* as well as prefixes. For example, {dog, dogs, log, logs, frog, frogs} has shared suffix "s" across multiple words. A trie has 6 terminal nodes and ~15 internal; a DAWG collapses the shared "s" edges, reducing total node count by ~30%.

For English dictionaries (~400K words), DAWGs achieve 2–5× space reduction over tries.

---

## 5. Augmented Tries for Autocomplete

### Top-K Suggestion Problem

Given a prefix $p$, return the $k$ most popular keys starting with $p$.

**Solution 1 (DFS with priority queue)**: walk to the prefix node, DFS the subtree, collect all terminal nodes with their popularity scores, sort by score. Time: $O(|\text{subtree}| \log k)$ worst case.

**Solution 2 (precomputed top-K)**: augment each node with a min-heap of its $k$ most popular descendant keys. Update heaps on insert/update. Query: $O(L + k)$ — walk to prefix, read heap.

### Weighted Sum Problem

For frequency or probability trees, augment each node with the sum of its subtree's weights. Enables $O(L)$ query of "probability mass of all keys with this prefix" — fundamental operation in compression (arithmetic coding) and language modeling.

---

## 6. Information-Theoretic Lower Bounds

### Space Lower Bound

For a set of $N$ distinct keys drawn from $\Sigma^L$, the information-theoretic lower bound on storage is:

$$S_{\min} = \lceil \log_2 \binom{\sigma^L}{N} \rceil \approx N \log_2 \frac{\sigma^L}{N}$$

For $N = 10^6$, $\sigma = 26$, $L = 10$: $S_{\min} \approx 10^6 \log_2 (26^{10}/10^6) \approx 3.6 \times 10^7$ bits = 4.5 MB.

Naive trie uses ~$N \sigma L w \approx 2 \times 10^9$ bytes (2 GB). Even with shared prefixes (~0.3× naive), still 600 MB. DAWG compression brings this to ~10–50 MB, within an order of magnitude of the information-theoretic lower bound.

### Query Time Lower Bound

Any data structure supporting key membership queries over $\Sigma^L$ must make $\Omega(L)$ comparisons in the worst case (to distinguish a key from all keys differing in the last position). The trie achieves this bound exactly.

---

## 7. Concurrent and Persistent Tries

### Hash-Array-Mapped Tries (HAMT)

HAMTs (Bagwell, 2001) use a bitmap at each node to compactly represent sparse 32-ary children. Used in persistent immutable maps in Clojure, Scala, and Elm.

Properties:

- Immutability: updates produce a new root sharing most of the old structure
- $O(\log_{32} N) \approx O(1)$ for most workloads
- Structural sharing: $N$ versions of the map share most nodes

### Lock-Free Tries

For concurrent insert/lookup, lock-free tries use CAS (compare-and-swap) on edge slots. Reads are wait-free; writes retry on contention.

Practical examples: `java.util.concurrent.ConcurrentHashMap` (hash-table-based, not trie), `ctrie` in Scala, Rust's `im::HashMap` (HAMT-based).

---

## 8. Practical Considerations

### Memory Allocation

In production tries, per-node allocation via standard heap allocators causes significant pointer-chasing and cache misses. Remedies:

1. **Arena allocation**: pre-allocate a contiguous pool of nodes; children store indices instead of pointers. Improves cache locality by 2–10×.
2. **Compressed node types**: use 4-byte indices instead of 8-byte pointers; use variable-size children (bitmap + compact array) for sparse nodes.
3. **Double-array trie** (Aoe, 1989): encode the trie in two integer arrays BASE and CHECK, achieving $O(1)$ transitions with $O(N)$ total space.

### When NOT to Use a Trie

- **Small number of very long keys**: hash table is simpler, same asymptotic cost.
- **Sparse alphabet with unpredictable characters**: trie pointer overhead dominates; rolling hash or BWT may be better.
- **Range queries on numeric keys**: B-tree or order-statistic tree is more flexible.
- **Approximate string matching**: trie alone doesn't help; combine with BK-tree or Levenshtein automaton.

---

## Prerequisites

- **Tree structures**: rooted trees, depth-first/breadth-first traversal, node vs edge labelling
- **Finite automata theory**: DFAs, regular languages, minimisation
- **Amortised analysis**: space amortised across shared prefixes
- **Pointer vs index representation**: trade-offs between heap pointers and contiguous arrays
- **Information theory**: entropy lower bounds for data structures

## Complexity

| Aspect | Bound | Notes |
|--------|-------|-------|
| `insert` time | $O(L)$ | $L$ = key length |
| `search` time | $O(L)$ | -- |
| `startsWith` time | $O(L)$ | -- |
| Space (naive) | $O(N \cdot L \cdot \sigma)$ | $\sigma$ = alphabet size |
| Space (amortised, shared prefixes) | $O(T)$ | $T$ = total key length |
| Radix tree node count | $\leq 2N - 1$ | Regardless of $T$ |
| DAWG space savings | 2–5× vs trie | For natural-language lexicons |
| Query lower bound | $\Omega(L)$ | Information-theoretic |
| IP routing lookup | $O(\log |\text{addr}|)$ effective | LC-trie, DIR-24-8 etc. |
| HAMT query | $O(\log_{32} N) \approx O(1)$ | Persistent immutable variant |
| Top-K autocomplete | $O(L + k)$ | With precomputed heaps |

---
