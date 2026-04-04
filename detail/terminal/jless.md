# The Mathematics of jless — Tree Structures, Navigation Complexity & JSON Parsing

> *A JSON document is a labeled ordered tree: navigating it is a graph traversal problem, searching it is pattern matching over a recursive data structure, and rendering it is a layout algorithm that maps tree depth to indentation within a viewport of fixed height.*

---

## 1. JSON as a Tree (Graph Theory)

### The Document Tree

A JSON document maps to a **labeled ordered tree** $T = (V, E, \lambda)$:

- **Nodes** $V$: each value (object, array, string, number, boolean, null)
- **Edges** $E$: parent-child relationships (object keys, array indices)
- **Labels** $\lambda$: key names for object children, indices for array children

For a JSON document with $n$ total values:

$$|V| = n, \quad |E| = n - 1$$

### Tree Metrics

| Metric | Symbol | Definition |
|:---|:---:|:---|
| Size | $n$ | Total number of nodes |
| Depth | $d$ | Maximum root-to-leaf path length |
| Branching factor | $b$ | Average children per internal node |
| Leaves | $L$ | Nodes with no children (primitives) |
| Internal nodes | $I$ | Objects and arrays |

The relationship: $n = I + L$ and $I = L - 1$ (for any tree):

$$n = 2L - 1$$

### Typical JSON Shapes

| Shape | Depth | Branching | Example |
|:---|:---:|:---:|:---|
| Wide and flat | 2-3 | 100+ | API list responses |
| Deep and narrow | 10+ | 2-3 | Nested config |
| Balanced | $\log_b n$ | $b$ | Structured data |
| Linear (array) | 2 | $n$ | Log entries |

---

## 2. Navigation Complexity (Cursor Movement)

### Movement Operations

Each navigation action in jless corresponds to a tree traversal operation:

| Key | Operation | Tree Traversal | Cost |
|:---|:---|:---|:---:|
| `j` | Next line | Next node in pre-order | $O(1)$ amortized |
| `k` | Previous line | Previous in pre-order | $O(1)$ amortized |
| `l` | Expand / enter child | First child | $O(1)$ |
| `h` | Collapse / go to parent | Parent pointer | $O(1)$ |
| `J` | Next sibling | Next sibling in parent | $O(1)$ |
| `K` | Previous sibling | Previous sibling | $O(1)$ |
| `g` | Go to root | Root pointer | $O(1)$ |
| `G` | Go to last | Rightmost leaf | $O(d)$ |

### Pre-order Traversal (j/k Movement)

`j` and `k` move through the document in **pre-order** (depth-first, parent before children):

$$visit\_order = [root, child_1, subtree(child_1), child_2, subtree(child_2), \ldots]$$

The total pre-order sequence has $n$ entries. With a doubly-linked flattened list, `j`/`k` are $O(1)$.

### Sibling Navigation (J/K)

Moving between siblings skips entire subtrees:

$$skip\_size(node) = |subtree(node)|$$

For a sibling in an array of $b$ elements, each with subtree size $s$:

$$T_{J} = O(1) \quad \text{(pointer to next sibling)}$$

But the visual effect is skipping $s$ lines -- this is the key UX advantage over `j`.

---

## 3. Collapse and Expand (Subtree Visibility)

### Collapse Model

When a node is collapsed, its entire subtree is hidden:

$$visible\_lines = n - \sum_{collapsed} (|subtree(c_i)| - 1)$$

Collapsing a node with subtree size $s$ reduces visible lines by $s - 1$.

### Depth-Based Collapse (1-9 Keys)

Pressing key $k$ collapses all nodes at depth $> k$:

$$visible(k) = |\{v \in V : depth(v) \leq k\}|$$

For a balanced tree with branching factor $b$:

$$visible(k) = \sum_{i=0}^{k} b^i = \frac{b^{k+1} - 1}{b - 1}$$

| Key | Depth | Visible (b=10) | Visible (b=3) |
|:---:|:---:|:---:|:---:|
| 1 | 1 | 11 | 4 |
| 2 | 2 | 111 | 13 |
| 3 | 3 | 1,111 | 40 |
| 4 | 4 | 11,111 | 121 |

Pressing `1` on an API response with 100 top-level keys shows exactly 101 lines -- instantly manageable.

---

## 4. Search Complexity (Pattern Matching Over Trees)

### Linear Search

jless searches both keys and values in pre-order:

$$T_{search} = O(n \times \bar{s})$$

Where $\bar{s}$ = average string length of keys and values.

### Search Space

The searchable content in a JSON document:

$$searchable = \sum_{v \in V} |key(v)| + |value(v)|$$

For a document with $n$ nodes, average key length $\bar{k}$, average value length $\bar{v}$:

$$|searchable| = n \times (\bar{k} + \bar{v})$$

### Incremental Search

As the user types each character of the search query $q$:

$$candidates(q_{1..i+1}) \subseteq candidates(q_{1..i})$$

This allows pruning: once a node fails to match the growing prefix, it stays excluded.

---

## 5. JSON Parsing (Computational Cost)

### Parsing Complexity

JSON parsing is $O(N)$ where $N$ = byte length of the input:

$$T_{parse} = O(N) \quad \text{(single pass, no backtracking)}$$

JSON is an **LL(1) grammar** -- each production can be determined by looking at one token ahead:

$$\text{value} \to \begin{cases} \text{string} & \text{if next = `"`} \\ \text{number} & \text{if next} \in \{0..9, -\} \\ \text{object} & \text{if next = `\{`} \\ \text{array} & \text{if next = `[`} \\ \text{true/false/null} & \text{if next} \in \{t, f, n\} \end{cases}$$

### Memory Model

jless builds an in-memory tree representation:

$$M_{tree} = n \times (S_{node} + \bar{k} + \bar{v})$$

Where $S_{node}$ = struct overhead per node (~64-128 bytes for type tag, children pointer, parent pointer, collapse state).

For a 10 MB JSON file with 100,000 nodes:

$$M \approx 100{,}000 \times (96 + 10 + 20) \approx 12.6 \text{ MB}$$

The in-memory representation is roughly 1.2x the file size for typical JSON.

### YAML Parsing Overhead

YAML parsing is significantly more complex than JSON:

$$T_{yaml} = O(N \times C_{yaml})$$

Where $C_{yaml} > C_{json}$ due to:

| Feature | JSON | YAML | Cost Multiplier |
|:---|:---:|:---:|:---:|
| Indentation tracking | No | Yes | ~2x |
| Anchor/alias resolution | No | Yes | ~1.5x |
| Multi-line strings | No | Yes | ~1.3x |
| Type inference | Explicit | Implicit | ~1.5x |

---

## 6. Viewport Rendering (Layout Algorithm)

### Line Layout

Each visible node maps to one or more display lines:

$$lines(v) = \begin{cases} 1 & \text{primitive value} \\ 1 & \text{collapsed container} \\ 2 + \sum_{c \in children} lines(c) & \text{expanded container (open + children + close)} \end{cases}$$

### Viewport Model

The terminal viewport shows $H$ lines. The rendering window:

$$window = [scroll\_offset, scroll\_offset + H)$$

Only nodes within the window need to be rendered:

$$T_{render} = O(H) \quad \text{per frame}$$

### Scroll Position Maintenance

When collapsing a node above the cursor, the scroll offset must adjust:

$$scroll' = scroll - (lines_{before} - 1)$$

Where $lines_{before}$ = number of lines the collapsed subtree occupied.

---

## 7. Path Representation (Tree Addressing)

### JSONPath Notation

The `p` command displays the path as a JSONPath expression:

$$path(v) = \prod_{i=0}^{d} accessor(v_i)$$

Where $accessor$ is either `.key` (object child) or `[index]` (array child).

Example: `.data.users[0].name`

### Path Length

$$|path(v)| = depth(v) \times \bar{a}$$

Where $\bar{a}$ = average accessor length. For depth 5 with average key length 8:

$$|path| \approx 5 \times 9 = 45 \text{ characters}$$

### Path as Address

Each path uniquely identifies a node. The total number of valid paths equals $n$:

$$|paths| = |V| = n$$

Path resolution (navigating from root to node via path) costs:

$$T_{resolve} = O(d \times \bar{b})$$

Where $\bar{b}$ = average branching factor (for object key lookup) or $O(d)$ for array index lookup.

---

## 8. Comparison: jless vs jq vs bat (Tool Tradeoffs)

### Computational Model Comparison

| Tool | Interaction | Parse Cost | Query Cost | Output |
|:---|:---|:---|:---|:---|
| jless | Interactive (TUI) | $O(N)$ once | $O(1)$ per move | Terminal |
| jq | Batch (pipeline) | $O(N)$ per query | $O(n)$ filter | stdout |
| bat -l json | View-only | $O(N)$ tokenize | None | Terminal |

### When Each Tool Wins

$$\text{jless wins when: } queries > 1 \text{ (interactive exploration)}$$
$$\text{jq wins when: } query\_count = 1 \text{ or scripted}$$
$$\text{bat wins when: } query\_count = 0 \text{ (just reading)}$$

### Memory Efficiency

| Tool | Memory | Model |
|:---|:---|:---|
| jless | $O(N)$ full tree | Parse once, navigate many |
| jq | $O(N)$ per query | Streaming (can be $O(1)$ for some filters) |
| bat | $O(H)$ viewport | Line-by-line tokenization |

---

## Prerequisites

- tree data structures, graph traversal, JSON specification, parsing theory, LL grammars, viewport rendering

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| JSON parse | $O(N)$ | $O(n)$ tree |
| YAML parse | $O(N \times C)$ | $O(n)$ tree |
| Navigation (j/k/h/l) | $O(1)$ | $O(1)$ |
| Go to end (G) | $O(d)$ | $O(1)$ |
| Search | $O(n \times \bar{s})$ | $O(1)$ |
| Collapse to depth $k$ | $O(n)$ | $O(n)$ flags |
| Viewport render | $O(H)$ | $O(H)$ |
| Path display | $O(d)$ | $O(d)$ |

---

*Navigating JSON is walking a tree with a viewport -- the mathematics of jless is about making tree traversal feel like scrolling text, where collapse operations prune subtrees to fit human attention spans and sibling jumps skip the structure you have already understood.*
