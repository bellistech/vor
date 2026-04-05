# The Mathematics of Word Break -- String Segmentation and Finite Automata

> *Deciding whether a string can be decomposed into a concatenation of words from a finite dictionary connects dynamic programming, formal language theory, and the algebraic structure of free monoids.*

---

## 1. The DP Recurrence (Dynamic Programming)

### The Problem

Derive the recurrence for deciding if a string $s$ of length $n$ can be segmented into
words from a dictionary $D$.

### The Formula

Let $dp(i)$ be true if the prefix $s[0..i)$ can be segmented into dictionary words.

$$dp(i) = \begin{cases} \text{true} & \text{if } i = 0 \\ \bigvee_{j=0}^{i-1} \left[ dp(j) \wedge s[j..i) \in D \right] & \text{otherwise} \end{cases}$$

The answer is $dp(n)$.

### Worked Examples

$s = \text{"leetcode"}$, $D = \{\text{"leet"}, \text{"code"}\}$:

| $i$ | Prefix | Matching $j$ | $dp(i)$ |
|-----|--------|--------------|---------|
| 0 | "" | (base) | true |
| 1 | "l" | none | false |
| 2 | "le" | none | false |
| 3 | "lee" | none | false |
| 4 | "leet" | $j=0$: dp(0)=T, "leet" $\in D$ | true |
| 5 | "leetc" | none | false |
| 6 | "leetco" | none | false |
| 7 | "leetcod" | none | false |
| 8 | "leetcode" | $j=4$: dp(4)=T, "code" $\in D$ | true |

---

## 2. Connection to Formal Languages (Automata Theory)

### The Problem

Express the word break problem in terms of regular languages and the Kleene star.

### The Formula

Let $L = \{w_1, w_2, \ldots, w_k\}$ be the dictionary (a finite set of strings). The
language of all segmentable strings is the Kleene star:

$$L^* = \{\varepsilon\} \cup L \cup L^2 \cup L^3 \cup \cdots$$

where $L^n = \{w_{i_1} w_{i_2} \cdots w_{i_n} \mid w_{i_j} \in L\}$.

The word break problem asks: $s \in L^*$?

Since $L$ is finite, $L^*$ is a regular language. An NFA can be constructed with states
$\{0, 1, \ldots, n\}$ (positions in $s$) where there is a transition from state $j$ to
state $i$ if $s[j..i) \in D$. The DP solution is exactly the subset construction
(determinization) of this NFA applied to the specific input $s$.

### Worked Examples

$D = \{\text{"ab"}, \text{"abc"}, \text{"c"}\}$, $s = \text{"abc"}$:

NFA transitions from position 0:
- "ab" $\in D$: $0 \to 2$
- "abc" $\in D$: $0 \to 3$

From position 2:
- "c" $\in D$: $2 \to 3$

Reachable states from 0: $\{0\} \to \{2, 3\}$. State 3 (= $n$) is reachable, so $s \in L^*$.

Two valid segmentations: "ab"+"c" and "abc".

---

## 3. Complexity Analysis (Algorithm Analysis)

### The Problem

Analyze the time and space complexity of the DP solution with and without optimization.

### The Formula

**Basic DP:** For each position $i \in [1, n]$, check all $j \in [0, i)$. Each check
involves a substring comparison of length up to $n$.

$$T(n) = \sum_{i=1}^{n} \sum_{j=0}^{i-1} O(i - j) = O\left(\sum_{i=1}^{n} \frac{i(i+1)}{2}\right) = O(n^3)$$

With hash set lookups (O(m) for a string of length m, amortized O(1) with rolling hash):

$$T(n) = O(n^2 \cdot m_{\text{avg}})$$

**Optimized DP:** Let $L_{\max} = \max_{w \in D} |w|$. Only check $j \in [\max(0, i - L_{\max}), i)$:

$$T(n) = O(n \cdot L_{\max} \cdot m_{\text{avg}})$$

Space: $O(n)$ for the DP array, $O(\sum |w_i|)$ for the hash set.

### Worked Examples

$n = 300$ (max constraint), $|D| = 1000$, $L_{\max} = 20$:
- Basic: $O(300^2 \cdot 20) = O(1{,}800{,}000)$ -- fast
- Optimized: $O(300 \cdot 20 \cdot 20) = O(120{,}000)$ -- very fast

---

## 4. Trie-Based Solution (Data Structures)

### The Problem

Optimize dictionary lookups using a trie to avoid redundant character comparisons.

### The Formula

Build a trie from the dictionary. For each position $j$ where $dp(j)$ is true, walk
the trie character by character from $s[j]$ onward. Every time a word-end node is
reached at position $i$, set $dp(i) = \text{true}$.

This eliminates redundant prefix comparisons: if "cat" and "cats" are both in $D$,
the trie shares the prefix "cat" and only diverges at the fourth character.

Total work per starting position $j$: $O(L_{\max})$ character comparisons.

$$T(n) = O(n \cdot L_{\max})$$

### Worked Examples

$D = \{\text{"cat"}, \text{"cats"}, \text{"and"}, \text{"sand"}\}$:

```
root
├── c → a → t (end) → s (end)
├── a → n → d (end)
└── s → a → n → d (end)
```

For $s = \text{"catsand"}$, starting at position 0:
- Walk: c-a-t (end! dp[3]=true), c-a-t-s (end! dp[4]=true), c-a-t-s-a (no child) -- stop.

---

## 5. BFS/Graph Interpretation (Graph Theory)

### The Problem

Model the word break problem as a shortest-path reachability problem on a DAG.

### The Formula

Construct a directed graph $G = (V, E)$ where:
- $V = \{0, 1, \ldots, n\}$ (positions in the string)
- $(j, i) \in E$ iff $s[j..i) \in D$

The word break problem asks: is there a path from vertex 0 to vertex $n$?

Since all edges go from smaller to larger indices, $G$ is a DAG. BFS or DFS from
vertex 0 decides reachability in $O(|E|)$ time. The number of edges is at most
$n \cdot L_{\max}$ (for each vertex, at most $L_{\max}$ outgoing edges to check).

The DP solution is equivalent to topological-order BFS on this DAG.

### Worked Examples

$s = \text{"applepenapple"}$, $D = \{\text{"apple"}, \text{"pen"}\}$:

Edges:
- $(0, 5)$: "apple"
- $(5, 8)$: "pen"
- $(8, 13)$: "apple"

Path: $0 \to 5 \to 8 \to 13$ -- vertex 13 ($= n$) is reachable.

---

## Prerequisites

- 1D dynamic programming
- Hash sets and string hashing
- Formal languages and automata (Kleene star, NFA)
- Tries (prefix trees)
- Graph reachability (BFS/DFS on DAGs)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Implement the basic DP solution. Trace through examples. Understand why greedy fails. Use a hash set for O(1) word lookups. |
| **Intermediate** | Optimize inner loop with max word length bound. Implement trie-based solution. Understand the BFS/graph interpretation. Extend to Word Break II (find all segmentations). |
| **Advanced** | Analyze connection to formal languages and Kleene star. Study Aho-Corasick automaton for multi-pattern matching. Explore the free monoid structure. Investigate approximate word break (with edit distance tolerance). |
