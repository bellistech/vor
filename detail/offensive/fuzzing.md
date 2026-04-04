# The Mathematics of Fuzzing -- Coverage Theory and Mutation Probability

> *A fuzzer's effectiveness is bounded by the probability that random mutation discovers the precise byte sequences needed to traverse conditional branches, making coverage guidance the critical accelerant that transforms brute-force search into directed exploration.*

---

## 1. Coverage Models and Edge Instrumentation (Graph Theory)

### The Problem

Coverage-guided fuzzers need to measure how much of a program's control flow has been exercised. The choice of coverage granularity -- basic blocks, edges, or paths -- determines the fuzzer's sensitivity to new behavior and its resistance to path explosion.

### The Formula

A program's control flow graph $G = (V, E)$ has basic blocks $V$ and edges $E$ (branches between blocks). The coverage metrics form a hierarchy:

$$\text{block coverage} = \frac{|V_{\text{hit}}|}{|V|} \leq \text{edge coverage} = \frac{|E_{\text{hit}}|}{|E|} \leq \text{path coverage} = \frac{|\text{paths}_{\text{hit}}|}{|\text{paths}|}$$

AFL uses edge coverage via a hash of (source_block, dest_block) pairs stored in a shared bitmap of size $2^{16} = 65536$ entries. The edge hash:

$$h = (\text{prev\_loc} \gg 1) \oplus \text{cur\_loc}$$

where $\text{prev\_loc}$ and $\text{cur\_loc}$ are compile-time random IDs for each basic block. The right-shift prevents $h(A \to B) = h(B \to A)$.

Collision probability: with $n$ instrumented edges and bitmap size $B = 65536$:

$$P(\text{collision}) = 1 - \prod_{i=0}^{n-1}\frac{B - i}{B} \approx 1 - e^{-n^2/(2B)}$$

| Edges (n) | Collision Probability | Effective Coverage Loss |
|:---|:---:|:---:|
| 100 | 7.4% | Negligible |
| 500 | 84.2% | Moderate |
| 1,000 | 99.98% | Severe |
| 5,000 | ~100% | Bitmap saturated |

For large programs (n > 1000 edges), collisions cause the fuzzer to miss novel edges. Solutions include larger bitmaps ($2^{20}$) or context-sensitive coverage.

### Worked Example

Program with 3,200 basic blocks and 4,800 edges. Bitmap size $B = 65536$.

Expected collisions: $E[\text{collisions}] \approx \frac{n^2}{2B} = \frac{4800^2}{131072} = 175.8$

Approximately 176 edge pairs will share a bitmap slot, meaning the fuzzer cannot distinguish between them. This represents 352/4800 = 7.3% coverage blindness.

## 2. Mutation Strategy and Byte-Level Probability (Probability Theory)

### The Problem

A coverage-guided fuzzer must mutate an input of length $L$ bytes to satisfy a specific comparison in the target program. The probability of randomly guessing the correct mutation determines how long the fuzzer takes to pass each check.

### The Formula

For a comparison `if (input[i] == 0x42)` where byte $i$ has a random value, the probability of a single random byte flip hitting the correct value:

$$P(\text{hit}) = \frac{1}{256}$$

For a multi-byte magic comparison `if (*(uint32_t*)input == 0xDEADBEEF)`:

$$P(\text{hit}) = \frac{1}{256^4} = \frac{1}{4{,}294{,}967{,}296}$$

Expected mutations to satisfy the check:

$$E[\text{mutations}] = \frac{1}{P(\text{hit})} = 256^k$$

where $k$ is the number of bytes that must simultaneously be correct.

| Comparison Type | Bytes (k) | Expected Mutations | At 10k exec/s |
|:---|:---:|:---:|:---:|
| Single byte | 1 | 256 | 26 ms |
| 16-bit magic | 2 | 65,536 | 6.5 s |
| 32-bit magic | 4 | 4.3 billion | 5 days |
| 64-bit magic | 8 | $1.8 \times 10^{19}$ | 57 billion years |

This is why CMPLOG and input-to-state inference are essential. By logging comparison operands, the fuzzer can directly substitute the expected value:

$$P(\text{hit with CMPLOG}) \approx 1 \quad \text{(deterministic substitution)}$$

## 3. Corpus Distillation and Set Cover (Combinatorial Optimization)

### The Problem

A fuzzing corpus grows over time as new coverage-triggering inputs are discovered. Many inputs trigger overlapping coverage. Corpus minimization finds the smallest subset that preserves all observed coverage, which is the minimum set cover problem.

### The Formula

Let $U$ be the set of all covered edges and $S = \{s_1, s_2, \ldots, s_n\}$ be the corpus where each $s_i$ covers edge set $C_i \subseteq U$. The minimum set cover:

$$\min |T| \quad \text{s.t.} \quad T \subseteq S, \quad \bigcup_{s_i \in T} C_i = U$$

This is NP-hard, so afl-cmin uses a greedy approximation:

1. Pick the input covering the most uncovered edges
2. Mark those edges as covered
3. Repeat until all edges are covered

The greedy algorithm achieves approximation ratio:

$$|T_{\text{greedy}}| \leq |T_{\text{opt}}| \cdot H(|U|)$$

where $H(n) = \ln n + 1$ is the harmonic number. For 10,000 edges:

$$|T_{\text{greedy}}| \leq |T_{\text{opt}}| \cdot 10.2$$

In practice, fuzzing corpora compress dramatically. A corpus of 50,000 inputs covering 8,000 edges typically minimizes to 500-2,000 inputs.

### Worked Example

Corpus: 12,000 inputs, 5,400 unique edges covered.

Greedy pass 1: input covering 1,200 edges selected. Remaining: 4,200 uncovered.
Greedy pass 2: input covering 800 new edges. Remaining: 3,400.
After 15 passes: 4,800 edges covered (89%) with 15 inputs.
After 200 passes: 5,400 edges covered (100%) with 200 inputs.

Compression ratio: 12,000 / 200 = 60x reduction.

## 4. Energy Scheduling and Markov Chain Models (Stochastic Processes)

### The Problem

A fuzzer must decide how many mutations (energy) to allocate to each seed in the corpus. Seeds closer to uncovered code should receive more energy. Power schedules model this allocation as a Markov chain where the fuzzer's exploration of the input space converges to a stationary distribution over program states.

### The Formula

AFL's power schedule assigns energy $e(s)$ to seed $s$ based on its properties:

$$e(s) = \alpha \cdot \frac{\text{bitmap\_size}(s)}{\overline{\text{bitmap\_size}}} \cdot \frac{\overline{\text{exec\_time}}}{\text{exec\_time}(s)} \cdot 2^{\min(\text{depth}(s), 16)}$$

where $\alpha$ is a base energy, bitmap_size is the number of edges triggered, and depth is the mutation chain length from the initial seed.

The "fast" schedule (Bohme et al., CCS 2017) models the fuzzer as a Markov chain over program states. The stationary distribution concentrates on low-frequency paths:

$$e_{\text{fast}}(s) \propto \frac{1}{f(s)^p}$$

where $f(s)$ is the number of times seed $s$ has been selected and $p$ controls the exploration-exploitation tradeoff.

The expected time to discover a path $\pi$ with branching probability $p_\pi$:

$$E[T_\pi] = \frac{1}{p_\pi \cdot r}$$

where $r$ is the fuzzer's execution rate (execs/sec).

## 5. Crash Deduplication and Stack Hashing (Hash Functions)

### The Problem

A single root-cause bug may produce thousands of crash inputs with varying surface symptoms. Deduplication groups crashes by likely root cause to focus triage effort. The standard approach hashes the crash stack trace, but the choice of hash granularity affects both false merging (different bugs grouped) and false splitting (same bug split).

### The Formula

Given a crash with stack frames $[f_1, f_2, \ldots, f_d]$ at depth $d$, a stack hash truncated to depth $k$:

$$h_k = \text{hash}(f_1, f_2, \ldots, f_{\min(k,d)})$$

With $n$ crashes and $b$ true unique bugs, the deduplication quality metrics:

$$\text{Precision} = \frac{|\text{true unique bugs found}|}{|\text{unique hashes}|}$$

$$\text{Recall} = \frac{|\text{true unique bugs found}|}{b}$$

Empirical results across fuzzing benchmarks:

| Hash Depth (k) | Unique Hashes | Precision | Recall |
|:---|:---:|:---:|:---:|
| 1 (crash function only) | Low | High (over-merged) | Low |
| 3 (top 3 frames) | Medium | Balanced | Medium |
| 5 (top 5 frames) | Medium-High | Good | Good |
| Full stack | High | Low (over-split) | High |

AFL uses edge coverage bitmap differences for deduplication: two crashes are considered unique if they trigger different edge coverage profiles. This is orthogonal to stack hashing.

## 6. Havoc Mutation Operators (Random Process Theory)

### The Problem

The "havoc" stage of AFL applies a random sequence of mutation operators to an input. The number and type of operators applied per mutation round follows a geometric distribution, and the combined effect determines the probability of escaping local coverage plateaus.

### The Formula

In each havoc iteration, AFL applies $m$ stacked mutations where $m$ is drawn from:

$$P(m = k) = \frac{1}{2^k} \quad \text{(geometric with } p = 0.5\text{)}$$

$$E[m] = 2, \quad \text{Var}(m) = 2$$

Each mutation is selected uniformly from a set of operators $\{o_1, \ldots, o_r\}$ (bit flip, byte flip, arithmetic, interesting values, block delete, block insert, overwrite, etc.). With $r = 16$ operators:

$$P(\text{specific operator sequence of length } k) = \frac{1}{r^k} = \frac{1}{16^k}$$

The probability that a havoc round produces a specific $k$-byte change:

$$P(\text{specific change}) = P(m \geq k) \cdot \frac{1}{r^k} \cdot \frac{1}{\binom{L}{k}}$$

where $L$ is the input length. For a 1,000-byte input needing a 3-byte specific change:

$$P = \frac{1}{2^3} \cdot \frac{1}{16^3} \cdot \frac{1}{\binom{1000}{3}} = \frac{1}{8} \cdot \frac{1}{4096} \cdot \frac{1}{1.66 \times 10^8} \approx 1.8 \times 10^{-13}$$

At 10,000 executions per second, this requires approximately 1.7 years -- explaining why structure-aware fuzzing dramatically outperforms blind havoc for complex formats.

## 7. Grammar-Based Fuzzing and Production Probabilities (Formal Language Theory)

### The Problem

For highly structured inputs (programming languages, protocol messages, serialization formats), random byte-level mutation almost never produces syntactically valid inputs. Grammar-based fuzzing generates inputs from a formal grammar, but must balance coverage of grammar productions against input length explosion.

### The Formula

Given a context-free grammar $G = (N, \Sigma, P, S)$ with nonterminals $N$, terminals $\Sigma$, productions $P$, and start symbol $S$, a probabilistic grammar assigns weight $w(p)$ to each production $p$. The probability of generating a specific derivation tree $T$:

$$P(T) = \prod_{p \in T} w(p)$$

The expected derivation depth with uniform production weights and average branching factor $b$:

$$E[\text{depth}] = \frac{\ln |\Sigma_{\text{target}}|}{\ln b}$$

The expected input length grows exponentially with depth for recursive grammars:

$$E[|x|] = O(b^d)$$

To control length, use depth-bounded generation with probability decay:

$$w_d(p) = w(p) \cdot \gamma^{\text{depth}} \quad \text{where } \gamma < 1$$

This biases toward terminal productions at greater depths, keeping inputs finite.

---

*The fundamental insight of coverage-guided fuzzing is that it transforms an intractable search problem -- finding one crashing input among $256^L$ possibilities -- into a tractable one by decomposing the search into incremental coverage gains, where each new edge discovered reduces the remaining search space and guides mutation toward unexplored program states.*

## Prerequisites

- Probability theory (geometric distributions, expected values, birthday paradox)
- Graph theory (control flow graphs, edge coverage, reachability)
- Combinatorial optimization (set cover, greedy approximation algorithms)
- Information theory (entropy, compression, Kolmogorov complexity)
- Formal language theory (context-free grammars, parsing, derivation trees)
- Stochastic processes (Markov chains, stationary distributions)

## Complexity

- **Beginner:** Understanding coverage metrics (block vs. edge), running AFL++ with default settings, interpreting crash outputs
- **Intermediate:** Custom harness development, corpus engineering, power schedule selection, CMPLOG integration, crash triage automation
- **Advanced:** Grammar-based fuzzer construction, symbolic execution hybrid approaches, kernel fuzzing with syzkaller, custom mutation strategies for protocol-specific targets
