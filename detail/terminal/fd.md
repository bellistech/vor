# The Mathematics of fd — Directory Traversal Algorithms & Parallel File Search

> *Finding files is a graph traversal problem on a DAG of directories: the speed depends on how many nodes you prune before visiting them, how many you visit in parallel, and how cheaply you can test each one against a compiled pattern.*

---

## 1. Directory Tree as a Graph (Traversal Theory)

### The Filesystem DAG

A filesystem is a **directed acyclic graph** (DAG) with:

- Nodes: directories and files
- Edges: parent-child relationships (hard links can create DAG, not tree)
- Root: the search starting directory

For a tree with $N$ total entries and branching factor $b$ at depth $d$:

$$N = \sum_{i=0}^{d} b^i = \frac{b^{d+1} - 1}{b - 1}$$

### Traversal Strategies

| Strategy | Order | Memory | Use Case |
|:---|:---|:---|:---|
| BFS (breadth-first) | Level by level | $O(b^d)$ | Depth-limited searches |
| DFS (depth-first) | Branch by branch | $O(d)$ | General traversal |
| Parallel BFS | Level by level | $O(b^d)$ | fd default |
| Parallel DFS | Work-stealing | $O(P \times d)$ | Large trees |

fd uses a **parallel BFS/DFS hybrid** -- directories are dispatched to a thread pool, and each thread performs DFS within its subtree.

---

## 2. Pruning and Ignore Rules (Search Space Reduction)

### Gitignore as Tree Pruning

When fd encounters a `.gitignore` rule matching a directory, it **prunes the entire subtree**:

$$N_{visited} = N_{total} - \sum_{pruned} N_{subtree_i}$$

### Pruning Efficiency

For a project with ignore patterns covering fraction $p$ of the total tree:

$$speedup_{prune} = \frac{N_{total}}{N_{total}(1 - p)} = \frac{1}{1 - p}$$

| Project Type | Total Files | Ignored | $p$ | Speedup |
|:---|:---:|:---:|:---:|:---:|
| Go (vendor/) | 10,000 | 8,000 | 0.80 | 5x |
| Node.js (node_modules/) | 100,000 | 95,000 | 0.95 | 20x |
| Python (venv/) | 15,000 | 12,000 | 0.80 | 5x |
| C++ (build/) | 20,000 | 15,000 | 0.75 | 4x |

### Hidden File Exclusion

By default, fd skips entries starting with `.`:

$$N_{visible} = N_{total} \times (1 - f_{hidden})$$

Where $f_{hidden}$ is typically 5-15% of entries (`.git`, `.cache`, `.config`, etc.).

---

## 3. Pattern Matching Cost (Regex Compilation)

### Per-Entry Matching

For each non-pruned entry, fd tests the filename (not the full path, by default) against the compiled regex:

$$T_{match} = O(n \times m)$$

Where $n$ = filename length and $m$ = regex pattern length.

### Regex vs Glob

| Mode | Compilation | Match Cost | Example |
|:---|:---|:---|:---|
| Regex (default) | $O(m)$ NFA build | $O(n \times m)$ | `'test.*\.go$'` |
| Glob (`-g`) | $O(m)$ translation | $O(n)$ for simple | `'*.go'` |
| Fixed string | None | $O(n)$ substring | `'config'` |

For literal substrings, the match reduces to a substring search:

$$T_{literal} = O(n / m) \quad \text{(Boyer-Moore average)}$$

### Extension Matching (`-e`)

Extension filtering is a **constant-time hash lookup** after extracting the extension:

$$T_{extension} = O(1) \quad \text{after } O(n) \text{ extension extraction}$$

This is why `-e go` is faster than `'\.go$'` -- it avoids regex entirely.

---

## 4. Parallel Execution Model (Thread Pool)

### Work Distribution

fd uses a bounded thread pool with $P$ workers (default: number of CPU cores):

$$T_{parallel} = \frac{T_{sequential}}{P} + T_{sync}$$

### Directory-Level Parallelism

The work unit is a directory. When a worker discovers subdirectories, it pushes them to a shared queue:

$$T_{worker} = T_{readdir} + \sum_{entries} T_{filter} + T_{match}$$

### readdir Cost

Each `readdir()` syscall returns a batch of directory entries:

$$T_{readdir} = T_{syscall} + \frac{|entries|}{batch\_size} \times T_{iter}$$

On Linux with `getdents64`, a single syscall returns multiple entries (~32 KB buffer):

$$entries\_per\_call \approx \frac{32768}{avg\_dirent\_size} \approx 128$$

### Amdahl's Law for fd

| Phase | Parallelizable | Fraction |
|:---|:---:|:---|
| Walk root directory | No | ~2% |
| Subdirectory traversal | Yes | ~70% |
| Pattern matching | Yes | ~20% |
| Output formatting | Partially | ~8% |

With $f \approx 0.90$ and $P = 8$:

$$speedup = \frac{1}{0.10 + \frac{0.90}{8}} \approx 4.7\times$$

---

## 5. Command Execution (--exec) Cost Model

### Per-Result Execution

With `--exec`, fd runs a command for each match:

$$T_{exec} = N_{matches} \times (T_{fork} + T_{exec\_cmd} + T_{wait})$$

Where $T_{fork} \approx 100\mu s$ on Linux. fd parallelizes execution:

$$T_{parallel\_exec} = \frac{N_{matches} \times T_{exec\_cmd}}{P} + T_{fork\_overhead}$$

### Batch Execution (--exec-batch)

With `--exec-batch`, fd invokes the command once with all results:

$$T_{batch} = T_{fork} + T_{exec\_cmd}(N_{matches})$$

### When to Use Each

The crossover point where batch becomes faster:

$$N \times T_{fork} > T_{fork} + T_{marginal} \times N$$

$$N > \frac{T_{fork}}{T_{fork} - T_{marginal}}$$

For `wc -l`: $T_{fork} = 100\mu s$, $T_{marginal} = 1\mu s$ per extra file:

$$N > \frac{100}{99} \approx 2 \text{ files}$$

Batch is almost always faster for external commands, but `--exec` enables per-file parallelism which can win for CPU-heavy commands.

---

## 6. Comparison with GNU find (Algorithmic Differences)

### Feature Impact on Performance

| Optimization | find | fd | Impact |
|:---|:---:|:---:|:---|
| Parallelism | Single-threaded | Multi-threaded | $\approx P\times$ speedup |
| Ignore files | None | `.gitignore` aware | $\frac{1}{1-p}\times$ pruning |
| Hidden files | Included | Excluded by default | 5-15% reduction |
| Regex engine | POSIX (slow) | Rust regex (fast) | 2-5x match speed |
| Color output | None | Default | 0 (cosmetic) |

### Benchmark Model

$$\frac{T_{find}}{T_{fd}} = \frac{N_{total} \times T_{match\_posix}}{(N_{total} / prune) \times T_{match\_rust} / P}$$

For a Node.js project ($N = 100{,}000$, $prune = 20\times$, $P = 8$, $T_{rust}/T_{posix} = 0.3$):

$$\frac{T_{find}}{T_{fd}} = \frac{100{,}000}{5{,}000 \times 0.3 / 8} = \frac{100{,}000}{187.5} \approx 533\times$$

---

## 7. Smart Case and Unicode Folding

### Smart Case Decision Function

$$case\_mode = \begin{cases} \text{insensitive} & \text{if } \forall c \in pattern: c = lower(c) \\ \text{sensitive} & \text{otherwise} \end{cases}$$

This is an $O(m)$ check on the pattern string at compile time.

### Unicode Case Folding

For case-insensitive matching with Unicode:

$$|fold\_table| \approx 1{,}400 \text{ entries}$$

Each character lookup: $O(\log F)$ via binary search, or $O(1)$ with a precomputed table for ASCII.

For filenames (typically ASCII-heavy), the fast ASCII path dominates:

$$T_{case\_fold} = O(n) \quad \text{with branchless ASCII mask}$$

---

## 8. Summary of fd Performance Model

| Factor | Formula | Typical Impact |
|:---|:---|:---|
| Tree size | $N = \frac{b^{d+1} - 1}{b - 1}$ | Base workload |
| Pruning | $\frac{1}{1-p}$ | 4-20x reduction |
| Parallelism | $\approx P$ cores | 4-8x speedup |
| Regex vs extension | $O(n \times m)$ vs $O(1)$ | 2-5x for simple searches |
| Batch vs per-file exec | $1$ fork vs $N$ forks | $N\times$ for external commands |

## Prerequisites

- graph traversal, tree algorithms, parallel computing, filesystem internals, regular expressions, Amdahl's law

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Directory traversal (sequential) | $O(N)$ | $O(d)$ |
| Directory traversal (parallel) | $O(N/P)$ | $O(P \times d)$ |
| Regex match per entry | $O(n \times m)$ | $O(m)$ |
| Extension filter per entry | $O(n)$ | $O(1)$ |
| --exec (parallel) | $O(k \times T_{cmd} / P)$ | $O(P)$ |
| --exec-batch | $O(T_{cmd}(k))$ | $O(k)$ |

---

*fd turns file finding from a brute-force crawl into a pruned parallel search -- skip the ignored subtrees, distribute the work across cores, and test only the filename against a compiled automaton instead of the full path against a shell glob.*
