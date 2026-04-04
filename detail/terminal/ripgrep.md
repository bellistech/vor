# The Mathematics of ripgrep — Finite Automata, Parallelism & Search Throughput

> *ripgrep is an exercise in applied automata theory and systems engineering: compile a regex to a minimal DFA, scatter file reads across cores, and race against disk bandwidth -- the goal is to make CPU faster than I/O so the disk is always the bottleneck.*

---

## 1. Regex Compilation Pipeline (Automata Construction)

### From Pattern to Automaton

ripgrep compiles a regex through a multi-stage pipeline:

$$\text{Pattern} \xrightarrow{\text{parse}} \text{AST} \xrightarrow{\text{Thompson}} \text{NFA} \xrightarrow{\text{powerset}} \text{DFA} \xrightarrow{\text{minimize}} \text{min-DFA}$$

### Thompson's Construction

For a regex of length $m$, Thompson's construction produces an NFA with:

$$|Q_{NFA}| \leq 2m \quad \text{states}$$
$$|\delta_{NFA}| \leq 4m \quad \text{transitions}$$

Construction time: $O(m)$.

### Powerset (Subset) Construction

Converting NFA to DFA via the powerset construction:

$$|Q_{DFA}| \leq 2^{|Q_{NFA}|}$$

Worst case is exponential, but in practice most states are unreachable. ripgrep uses **lazy DFA construction** -- states are built on demand and cached:

$$T_{lazy\_build} = O(|Q_{reached}|) \ll O(2^{|Q_{NFA}|})$$

The cache is bounded; if it fills, ripgrep falls back to NFA simulation.

---

## 2. The Literal Optimization (Inner Loop Acceleration)

### Extracting Literal Prefixes

Before running the full automaton, ripgrep extracts literal strings from the pattern:

| Pattern | Extracted Literals | Strategy |
|:---|:---|:---|
| `error` | `error` | Memchr + Boyer-Moore |
| `func\s+(\w+)` | `func` | Literal prefix scan |
| `(foo\|bar\|baz)` | `foo`, `bar`, `baz` | Aho-Corasick |
| `\d{3}-\d{4}` | (none) | Full DFA |

### Memchr and SIMD

For single-byte literal searches, ripgrep uses `memchr` which leverages SIMD instructions:

$$T_{memchr} = O\left(\frac{N}{W}\right)$$

Where $W$ is the SIMD width (typically 16-32 bytes with SSE/AVX). On a 4 GHz CPU with AVX2:

$$throughput \approx \frac{4 \times 10^9 \times 32}{1} = 128 \text{ GB/s}$$

This far exceeds memory bandwidth (~50 GB/s), so memchr is memory-bound in practice.

### Teddy Multi-Pattern SIMD

For small pattern sets (up to 8 literals), ripgrep uses the **Teddy** algorithm -- a SIMD-accelerated multi-pattern matcher:

$$T_{teddy} = O\left(\frac{N}{W}\right)$$

This matches multiple literals simultaneously by encoding byte fingerprints into SIMD registers.

---

## 3. Parallelism Model (Work Distribution)

### File-Level Parallelism

ripgrep parallelizes across files using a work-stealing thread pool:

$$T_{parallel} = \frac{T_{sequential}}{P} + T_{overhead}$$

Where $P$ = number of worker threads (defaults to number of logical cores).

### Amdahl's Law Applied

$$speedup = \frac{1}{(1 - f) + \frac{f}{P}}$$

Where $f$ = fraction of work that is parallelizable. For ripgrep:

| Component | Sequential? | Fraction |
|:---|:---:|:---|
| Directory traversal | Partially | ~5% |
| File reading (mmap) | Yes (OS) | ~30% |
| Regex matching | Fully parallel | ~60% |
| Output serialization | Sequential | ~5% |

With $f \approx 0.90$ and $P = 8$:

$$speedup = \frac{1}{0.10 + \frac{0.90}{8}} = \frac{1}{0.2125} \approx 4.7\times$$

### Lock-Free Output

ripgrep uses per-thread output buffers to avoid lock contention on stdout:

$$T_{output} = \max(T_{buffer\_fill}, T_{flush})$$

Results are printed per-file (not per-line), which amortizes synchronization overhead.

---

## 4. I/O Model (Memory Mapping vs Read)

### Memory-Mapped I/O

For large files, `mmap()` maps file pages directly into virtual memory:

$$T_{mmap} = T_{page\_fault} \times \lceil \frac{file\_size}{page\_size} \rceil$$

With 4 KB pages and readahead:

| Method | Syscalls | Copies | Kernel Overhead |
|:---|:---:|:---:|:---|
| `read()` loop | $\lceil N/buf \rceil$ | 1 (kernel to user) | Per-call |
| `mmap()` | 1 setup | 0 (zero-copy) | Page faults |

### When mmap Wins

$$\text{mmap advantage} = \frac{syscall\_count \times syscall\_cost}{page\_fault\_cost \times page\_count}$$

For sequential reads with OS readahead, mmap wins when files are large enough to amortize the setup cost (~64 KB threshold).

ripgrep uses `mmap` for files above a threshold and `read()` for small files.

---

## 5. .gitignore Matching (Glob Automata)

### Pattern Compilation

Each `.gitignore` pattern is compiled to a glob matcher. For $G$ patterns:

$$T_{ignore\_check} = O(G \times \bar{p})$$

Where $\bar{p}$ = average pattern length. ripgrep compiles all patterns into a single **gitignore automaton**:

$$T_{compiled} = O(\bar{p})$$

### Impact on Search Space

Given a repository with $F$ total files and $I$ ignored files:

$$speedup_{ignore} = \frac{F}{F - I}$$

Typical ratios in a Node.js project:

| Directory | Files | Ignored? |
|:---|:---:|:---:|
| `src/` | 200 | No |
| `node_modules/` | 50,000 | Yes |
| `.git/` | 5,000 | Yes |

$$speedup = \frac{55{,}200}{200} = 276\times$$

This is the single largest performance advantage over grep.

---

## 6. Complexity Analysis (End-to-End)

### Per-File Cost

$$T_{file} = T_{open} + T_{read} + T_{match} + T_{output}$$

$$T_{match} = \begin{cases} O(N / W) & \text{literal (SIMD)} \\ O(N) & \text{DFA} \\ O(N \times m) & \text{NFA fallback} \end{cases}$$

### Total Search Cost

$$T_{total} = T_{walk} + \frac{1}{P} \sum_{f \in files} T_{file}(f)$$

$$T_{walk} = O(F \times G) \quad \text{directory walk with ignore checking}$$

### Throughput Model

$$throughput = \min\left(\frac{P \times regex\_speed}{1}, disk\_bandwidth\right)$$

| Scenario | CPU Throughput | Disk Bandwidth | Bottleneck |
|:---|:---|:---|:---|
| Literal search, NVMe | 50 GB/s (SIMD) | 3.5 GB/s | Disk |
| Complex regex, NVMe | 0.5 GB/s | 3.5 GB/s | CPU |
| Literal search, HDD | 50 GB/s | 0.2 GB/s | Disk |
| Complex regex, HDD | 0.5 GB/s | 0.2 GB/s | Disk |

---

## 7. Comparison with grep and ag (The Silver Searcher)

### Algorithmic Differences

| Feature | grep | ag | ripgrep |
|:---|:---|:---|:---|
| Regex engine | POSIX NFA | PCRE | Rust regex (DFA/NFA hybrid) |
| Literal optimization | Sometimes | No | Always (Teddy/memchr) |
| Threading | Single | Multi | Multi (work-stealing) |
| Ignore files | Manual | `.gitignore` | `.gitignore` + `.ignore` + `.rgignore` |
| Memory mapping | No | Yes | Conditional |
| Unicode | Full | Full | Configurable |

### Benchmark Model

For a codebase with $F$ files, $N$ total bytes, pattern with literal prefix:

$$T_{grep} \approx \frac{N}{BM\_speed} + F \times T_{open}$$

$$T_{rg} \approx \frac{N_{filtered}}{P \times SIMD\_speed} + \frac{F_{filtered}}{P} \times T_{open}$$

Where $N_{filtered} \ll N$ and $F_{filtered} \ll F$ due to gitignore.

---

## 8. Unicode and Encoding Costs

### UTF-8 Validation

ripgrep operates on raw bytes by default but validates UTF-8 when needed:

$$T_{utf8} = O(N) \quad \text{with SIMD validation}$$

### Case Folding Complexity

Case-insensitive matching with Unicode requires mapping each character through a folding table:

$$|\text{Unicode case folds}| \approx 1{,}400 \text{ mapping pairs}$$

Simple ASCII folding: $O(1)$ per byte (mask bit 5). Full Unicode folding: $O(\log F)$ per codepoint via binary search on the fold table, where $F$ = number of fold entries.

---

## Prerequisites

- automata theory, finite state machines, regular expressions, SIMD instructions, memory-mapped I/O, parallel algorithms, Amdahl's law

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Regex compile (NFA) | $O(m)$ | $O(m)$ |
| Lazy DFA build | $O(\|reached\|)$ | Bounded cache |
| Literal search (SIMD) | $O(N/W)$ | $O(1)$ |
| DFA match | $O(N)$ | $O(1)$ |
| NFA match | $O(N \times m)$ | $O(m)$ |
| Parallel search | $O(N / P)$ | $O(P)$ |
| Directory walk | $O(F \times G)$ | $O(F)$ |

---

*ripgrep is what happens when you take the theoretical minimum for regex matching, add every systems-level optimization the hardware offers, and then skip 99% of the files before you even start -- the fastest search is the one you never have to do.*
