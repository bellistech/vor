# The Mathematics of delta -- Diff Algorithms and Syntax-Aware Rendering

> *delta transforms unified diff output into a visually rich display by combining the Myers diff algorithm for line-level changes with token-level edit distance for word highlighting, all rendered through a syntax-aware pipeline that preserves language structure.*

---

## 1. Line-Level Diffing (Shortest Edit Script)

### The Problem

Given two files as sequences of lines $A = a_1, a_2, \ldots, a_m$ and $B = b_1, b_2, \ldots, b_n$, find the minimum set of insertions and deletions to transform $A$ into $B$.

### The Formula

Git uses the Myers diff algorithm. Define an edit graph where moving right means delete from $A$, down means insert from $B$, and diagonal means match:

$$D(k) = \max\{x : (x, x-k) \text{ is reachable in } d \text{ steps}\}$$

where $k = x - y$ is the diagonal index and $d$ is the edit distance. The recurrence:

$$D_d(k) = \max \begin{cases} D_{d-1}(k-1) + 1 & \text{(insert)} \\ D_{d-1}(k+1) & \text{(delete)} \end{cases}$$

Then extend diagonally while $A[x+1] = B[y+1]$ (free matches).

### Worked Examples

$A$ = ["foo", "bar", "baz"], $B$ = ["foo", "qux", "baz"]

Edit graph diagonals: $k \in \{-1, 0, 1\}$

- $d=0$: start at $(0,0)$, diagonal to $(1,1)$ (match "foo"), stop.
- $d=1$: from $k=0$ try insert ($k=-1$) reaching $(1,2)$, or delete ($k=1$) reaching $(2,1)$.
- $d=2$: from $k=-1$ delete to $(2,2)$, diagonal to $(3,3)$ (match "baz"). Done.

Minimum edit distance $D = 2$ (delete "bar", insert "qux").

---

## 2. Word-Level Diff (Intra-Line Highlighting)

### The Problem

Given two similar lines $L_{old}$ and $L_{new}$, identify the specific tokens (words) that changed for emphasis highlighting.

### The Formula

delta tokenizes each line and applies edit distance at the token level. For tokens $T_{old} = t_1, \ldots, t_p$ and $T_{new} = t_1', \ldots, t_q'$:

$$d(T_{old}, T_{new}) = \min \text{ insertions + deletions + substitutions}$$

Using the standard DP:

$$D[i][j] = \min \begin{cases} D[i-1][j] + 1 \\ D[i][j-1] + 1 \\ D[i-1][j-1] + [t_i \neq t_j'] \end{cases}$$

Tokens not in the LCS are highlighted with emphasis styles.

### Worked Examples

$L_{old}$: `result = compute(x, y)` $\to$ tokens: [result, =, compute, (, x, ",", y, )]

$L_{new}$: `result = calculate(x, z)` $\to$ tokens: [result, =, calculate, (, x, ",", z, )]

Changed tokens: compute $\to$ calculate, y $\to$ z. These receive `minus-emph-style` and `plus-emph-style`.

---

## 3. Syntax Highlighting Pipeline (Finite Automata)

### The Problem

Apply language-aware syntax coloring to diff content while preserving diff-specific styling (plus/minus colors).

### The Formula

Syntax highlighting uses a stack-based finite automaton defined by TextMate grammars. For input string $s$ and grammar $G = (S, \Sigma, \delta, s_0, F)$:

$$\delta: S \times \Sigma^* \to S \times \text{Scope}$$

Each scope maps to a color via the theme:

$$\text{color}: \text{Scope} \to (fg, bg, \text{attrs})$$

Delta composites diff colors with syntax colors:

$$\text{final\_style}(c) = \begin{cases} \text{syntax}(c) \oplus \text{plus-bg} & \text{if added line} \\ \text{syntax}(c) \oplus \text{minus-bg} & \text{if removed line} \\ \text{syntax}(c) & \text{if context line} \end{cases}$$

where $\oplus$ means overlay foreground from syntax, background from diff.

---

## 4. Unified Diff Parsing (Grammar)

### The Problem

Parse the unified diff format to extract file headers, hunks, and line changes.

### The Formula

Unified diff follows a context-free grammar:

$$\text{Diff} \to \text{FilePair}^+$$
$$\text{FilePair} \to \text{Header} \;\text{Hunk}^+$$
$$\text{Header} \to \texttt{---}\;path_a \;\;\texttt{+++}\;path_b$$
$$\text{Hunk} \to \texttt{@@}\;range_a\;range_b\;\texttt{@@}\;\text{Line}^+$$
$$\text{Line} \to (\texttt{+} \mid \texttt{-} \mid \texttt{ })\;\text{content}$$

Range format: $-s_a,c_a\;+s_b,c_b$ where $s$ is start line and $c$ is count.

The invariant for each hunk:

$$|\{L : L \text{ is } \texttt{-} \text{ or context}\}| = c_a$$
$$|\{L : L \text{ is } \texttt{+} \text{ or context}\}| = c_b$$

---

## 5. Color-Moved Detection (Block Matching)

### The Problem

Detect blocks of code that were moved (not modified) between positions in the file to display them differently from actual additions/deletions.

### The Formula

Git's `diff.colorMoved` identifies moved blocks by hashing line content. For removed block $R = r_1, \ldots, r_k$ and added block $A = a_1, \ldots, a_k$:

$$\text{moved}(R, A) \iff h(r_i) = h(a_i) \;\forall\; i \in [1, k] \land k \geq \theta$$

where $h$ is a content hash and $\theta$ is the minimum block size (default 3 lines).

The hash function ignores whitespace changes when `colorMovedWS = ignore-space-change`:

$$h(l) = \text{hash}(\text{normalize\_ws}(l))$$

Delta maps git's moved-line ANSI markers to custom styles via `map-styles`.

---

## 6. Side-by-Side Layout (Column Allocation)

### The Problem

Render old and new file versions in adjacent columns, aligning matching lines and handling insertions/deletions.

### The Formula

Given terminal width $W$, line number widths $w_{ln}$, and separator width $w_{sep}$:

$$w_{content} = \frac{W - 2w_{ln} - w_{sep}}{2}$$

For each hunk, lines are paired:

$$\text{pair}(i) = \begin{cases} (old_i, new_j) & \text{if both present (change)} \\ (old_i, \emptyset) & \text{if deletion only} \\ (\emptyset, new_j) & \text{if insertion only} \end{cases}$$

Line wrapping for content exceeding $w_{content}$:

$$\text{wrapped\_lines}(l) = \left\lceil \frac{|l|}{w_{content}} \right\rceil$$

Total display lines per hunk:

$$L_{display} = \sum_{pairs} \max(\text{wrapped\_lines}(old_i), \text{wrapped\_lines}(new_j))$$

---

## 7. Complexity Analysis

### Per-Diff Processing

For files with $m$ and $n$ lines respectively, and average line length $\bar{l}$:

$$T_{line\_diff} = O((m + n) \times D)$$

where $D$ is the edit distance (Myers algorithm is $O((m+n) \times D)$).

$$T_{token\_diff} = O(|changed| \times \bar{l}^2)$$

$$T_{syntax} = O((m + n) \times \bar{l})$$

Total rendering time:

$$T_{total} = T_{line\_diff} + T_{token\_diff} + T_{syntax} + T_{render}$$

For typical diffs ($D \ll m + n$), the pipeline is nearly linear:

$$T_{total} \approx O((m + n) \times \bar{l})$$

### Memory

$$S = O((m + n) \times \bar{l})$$

The entire diff must be held in memory for syntax highlighting context.

---

## 8. Color Space and ANSI Encoding

### True Color vs 256-Color

Terminal color encoding uses different quantization levels:

$$\text{TrueColor}: c \in [0, 255]^3 \implies 16{,}777{,}216 \text{ colors}$$

$$\text{256-color}: c \in \{0, \ldots, 255\} \implies 216 \text{ color cube} + 24 \text{ grays}$$

The 216-color cube maps RGB:

$$\text{index} = 16 + 36r + 6g + b \quad \text{where } r, g, b \in \{0, \ldots, 5\}$$

Nearest color in the 256 palette:

$$c_{256} = \arg\min_{i \in [0,255]} \|RGB(i) - RGB_{target}\|_2$$

Delta defaults to true color (`true-color = always`) to avoid quantization artifacts in syntax themes.

---

## Prerequisites

- edit distance, dynamic programming, Myers diff algorithm, finite automata, context-free grammars, color theory, ANSI escape sequences

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Line diff (Myers) | $O((m+n) \times D)$ | $O(D^2)$ |
| Word diff (per line) | $O(p \times q)$ | $O(p \times q)$ |
| Syntax highlight | $O(n \times \bar{l})$ | $O(n)$ |
| Side-by-side layout | $O(n)$ | $O(n)$ |
| Color quantization | $O(1)$ per pixel | $O(1)$ |

---

*Diff rendering is a multi-layer transformation pipeline: structural diffing identifies changes, token-level alignment pinpoints modifications, syntax analysis preserves language semantics, and color compositing merges all three into a coherent visual output.*
