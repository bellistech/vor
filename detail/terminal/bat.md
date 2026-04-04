# The Mathematics of bat — Syntax Highlighting, Parsing Theory & Terminal Rendering

> *Syntax highlighting is a language recognition problem: parse a stream of bytes into tokens using a grammar automaton, map each token class to a color, and serialize the result as ANSI escape sequences -- all within the latency budget of interactive terminal display.*

---

## 1. Syntax Highlighting as Lexical Analysis (Automata Theory)

### The Tokenization Problem

Given source text $S = s_1 s_2 \ldots s_N$, syntax highlighting partitions $S$ into a sequence of tokens:

$$S = t_1 \cdot t_2 \cdot \ldots \cdot t_k$$

Where each token $t_i$ has a **scope** (type) drawn from a finite set:

$$scope \in \{keyword, string, comment, number, operator, identifier, \ldots\}$$

### TextMate Grammar Model

bat uses **TextMate grammars** (.tmLanguage) which define token rules as a set of regex patterns:

$$G = \{(pattern_i, scope_i) : i = 1 \ldots R\}$$

For each position in the text, the grammar tests patterns in priority order:

$$match = \arg\max_{priority} \{pattern_i : pattern_i \text{ matches at position } p\}$$

### Complexity of Tokenization

With $R$ rules, each a regex of average length $r$, applied to text of length $N$:

$$T_{tokenize} = O(N \times R \times r)$$

In practice, TextMate grammars use **begin/end** patterns that create a context stack:

$$T_{contextual} = O(N \times R_{active} \times r)$$

Where $R_{active} \ll R$ because only rules matching the current scope context are tested.

---

## 2. Context Stack and Scope Resolution (Pushdown Automaton)

### The Grammar Stack

TextMate grammars maintain a **context stack** -- a pushdown automaton where:

- `begin` patterns push a new scope onto the stack
- `end` patterns pop the scope
- Nested rules are only active within their parent scope

$$stack = [scope_1, scope_2, \ldots, scope_d]$$

The current scope is the concatenation: `scope_1.scope_2...scope_d`

### Example: Nested Highlighting

For Go source code:

```
func main() {                    // scope: source.go
    s := "hello \"world\""       // scope: source.go > string.quoted.double
    fmt.Println(s)               // scope: source.go
}
```

Stack depth during parsing:

| Position | Stack Depth | Active Rules |
|:---|:---:|:---:|
| `func` keyword | 1 | ~50 (top-level) |
| Inside `""` string | 2 | ~10 (string rules) |
| Escape `\"` | 3 | ~3 (escape rules) |
| After string close | 1 | ~50 (top-level) |

### Maximum Stack Depth

For well-formed source code, the maximum nesting depth:

$$d_{max} = O(\log N) \quad \text{typical}$$
$$d_{max} = O(N) \quad \text{worst case (pathological input)}$$

---

## 3. Theme Mapping (Color Theory & ANSI Encoding)

### Scope-to-Style Resolution

A theme maps scope selectors to visual styles:

$$theme: scope\_selector \to (fg\_color, bg\_color, style\_flags)$$

Scope matching uses **longest prefix match**:

| Scope Selector | Specificity | Matches |
|:---|:---:|:---|
| `comment` | 1 | All comments |
| `comment.line` | 2 | Line comments only |
| `comment.line.double-slash` | 3 | `//` comments only |

Resolution: $O(S \times d)$ where $S$ = number of theme rules and $d$ = scope depth.

### ANSI Color Encoding

Terminal colors are encoded as escape sequences:

| Mode | Colors | Escape Format | Bytes |
|:---|:---:|:---|:---:|
| 4-bit (basic) | 16 | `\033[31m` | 5 |
| 8-bit (256) | 256 | `\033[38;5;196m` | 10 |
| 24-bit (truecolor) | 16.7M | `\033[38;2;R;G;Bm` | 15-19 |

### Output Expansion Factor

For source text of length $N$ with $k$ tokens, the ANSI-colored output size:

$$N_{output} = N + k \times (C_{open} + C_{reset})$$

With truecolor: $C_{open} \approx 19$ bytes, $C_{reset} = 4$ bytes (`\033[0m`).

For typical code with ~1 token per 5 characters:

$$\frac{N_{output}}{N} = 1 + \frac{k \times 23}{N} \approx 1 + \frac{23}{5} \approx 5.6\times$$

Highlighted output is roughly 5-6x larger than plain text.

---

## 4. Line Range Selection (Efficient Partial Reading)

### Byte Offset Calculation

For `--line-range start:end`, bat must find byte offsets for line boundaries:

$$offset(line_n) = \sum_{i=1}^{n-1} |line_i| + 1$$

Without an index, finding line $n$ requires scanning: $O(offset(line_n))$.

### Newline Scanning with memchr

bat uses `memchr` for newline scanning:

$$T_{line\_scan} = O\left(\frac{offset}{W}\right)$$

Where $W$ = SIMD width. For a 10 MB file starting at line 1000 with average 80-byte lines:

$$offset \approx 1000 \times 80 = 80\text{ KB}$$
$$T \approx \frac{80{,}000}{32} \approx 2{,}500 \text{ operations (negligible)}$$

---

## 5. Git Integration (Diff Computation)

### Change Detection Model

bat computes per-line change status by comparing the working copy against the git index:

$$status(line_i) \in \{unchanged, added, modified, deleted\}$$

This uses a variant of the **Myers diff algorithm**:

$$T_{diff} = O(N \times D)$$

Where $D$ = edit distance (number of changes). For small edits ($D \ll N$):

$$T_{diff} \approx O(N)$$

### Gutter Rendering Cost

The diff gutter adds constant overhead per line:

$$T_{gutter} = O(L) \quad \text{where } L = \text{number of displayed lines}$$

Each line gets one additional character (`+`, `~`, `-`, or space) plus ANSI coloring.

---

## 6. Paging Decision (TTY Detection)

### Auto-Paging Logic

bat's paging decision function:

$$page = \begin{cases} \text{always} & \text{if --paging=always} \\ \text{never} & \text{if --paging=never or not TTY} \\ \text{true} & \text{if } L_{output} > L_{terminal} \\ \text{false} & \text{otherwise} \end{cases}$$

### TTY Detection

```
is_tty = isatty(STDOUT_FILENO)
```

When output is piped, bat automatically:
1. Disables paging
2. Disables decorations (unless `--style` is set)
3. Keeps syntax highlighting (unless `--color=never`)

This is why `bat file.go | head` works correctly -- no pager interference.

---

## 7. Throughput and Rendering Pipeline

### End-to-End Pipeline

$$T_{total} = T_{read} + T_{tokenize} + T_{theme} + T_{render} + T_{page}$$

| Phase | Cost | Bottleneck |
|:---|:---|:---|
| File read | $O(N)$ | I/O |
| Tokenization | $O(N \times R_{active})$ | CPU (regex) |
| Theme resolution | $O(k \times S)$ | CPU (lookup) |
| ANSI rendering | $O(N_{output})$ | CPU (string build) |
| Terminal output | $O(N_{output})$ | Terminal bandwidth |

### Terminal Bandwidth Limit

Modern terminals process approximately:

$$BW_{terminal} \approx 10{-}100 \text{ MB/s of ANSI text}$$

With 5.6x expansion, bat's effective throughput:

$$throughput_{effective} \approx \frac{BW_{terminal}}{5.6} \approx 2{-}18 \text{ MB/s of source text}$$

For interactive use (files under 1 MB), this is imperceptible. For large files (>10 MB), the `--line-range` flag avoids processing the entire file.

---

## 8. Comparison: cat vs bat vs less

### Feature and Cost Comparison

| Tool | Tokenization | Coloring | Paging | Overhead |
|:---|:---:|:---:|:---:|:---|
| cat | None | None | None | $O(N)$ I/O only |
| bat | TextMate grammar | ANSI 24-bit | Auto (less) | $O(N \times R)$ |
| less | None | None | Built-in | $O(N)$ I/O + indexing |
| bat -pp | TextMate grammar | ANSI 24-bit | None | $O(N \times R)$ |

### Startup Latency

| Tool | Cold Start | Warm Start | Notes |
|:---|:---|:---|:---|
| cat | ~1 ms | ~1 ms | Minimal binary |
| bat | ~15 ms | ~5 ms | Grammar + theme load |
| less | ~3 ms | ~2 ms | Terminal setup |

bat's startup includes loading compiled syntax definitions and theme from a binary cache (`bat cache --build` pre-compiles them):

$$T_{startup} = T_{cache\_mmap} + T_{theme\_parse}$$

The cache (~2 MB) is memory-mapped, so warm starts avoid re-reading from disk.

---

## Prerequisites

- lexical analysis, pushdown automata, regular expressions, context-free grammars, ANSI escape codes, diff algorithms

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| File read | $O(N)$ | $O(N)$ |
| Tokenization | $O(N \times R_{active})$ | $O(d)$ stack |
| Theme resolution | $O(k \times S)$ | $O(S)$ |
| ANSI rendering | $O(N_{output})$ | $O(N_{output})$ |
| Git diff (Myers) | $O(N \times D)$ | $O(N)$ |
| Line range seek | $O(offset / W)$ | $O(1)$ |

---

*Syntax highlighting turns reading code into a recognition problem -- a pushdown automaton walks the grammar stack, a theme function maps scope to color, and ANSI escape codes deliver the result at 5x the byte cost but 10x the readability.*
