# The Mathematics of Reverse Engineering -- Binary Analysis and Complexity Theory

> *A compiled binary is a lossy compression of the programmer's intent; reverse engineering is the art of reconstructing meaning from the residual structure.*

---

## 1. Control Flow Graph Reconstruction (Graph Theory)

### The Problem

Disassemblers must reconstruct a program's control flow graph (CFG) from a flat byte stream. Indirect jumps (jump tables, function pointers, virtual dispatch) make complete CFG recovery undecidable in the general case. Understanding the theoretical limits helps analysts recognize when automated analysis will fail and manual intervention is needed.

### The Formula

A basic block is a maximal sequence of instructions with one entry point and one exit. The CFG is $G = (B, E)$ where B is the set of basic blocks and E the set of edges (branches). For a function with n basic blocks, the cyclomatic complexity is:

$$M = |E| - |B| + 2p$$

where p is the number of connected components (typically 1 for a single function). This measures the number of linearly independent paths:

$$\text{paths} = M$$

The number of possible execution paths through a function with d decision points (binary branches):

$$|\text{paths}| \leq 2^d$$

For indirect jumps with a jump table of size k, the potential edges increase by:

$$|E_{\text{indirect}}| = k - 1 \quad \text{(replacing 1 edge with k edges)}$$

The decidability boundary: determining the complete CFG is equivalent to the halting problem for arbitrary computed jump targets. For bounded programs (no recursion, finite loops), CFG recovery is decidable but potentially exponential.

### Worked Examples

**Example 1: Simple function complexity**

Function with 15 basic blocks, 20 edges, 1 component.
- Cyclomatic complexity: M = 20 - 15 + 2 = 7
- 7 independent paths to test/analyze
- With 5 binary decision points: upper bound 2^5 = 32 paths
- M = 7 << 32 because many branches are correlated

**Example 2: Obfuscated dispatcher**

Malware dispatcher with computed jump: `jmp [rax*8 + table]`. Table has 50 entries.
- Each entry adds an edge: 50 potential targets
- Cyclomatic complexity increases by 49
- Without table bounds analysis, disassembler may miss targets
- False positive rate: addresses in table that are not valid code = data misinterpreted as code

## 2. Entropy Analysis for Packing Detection (Information Theory)

### The Problem

Packed or encrypted binaries have near-uniform byte distributions in their code sections, while normal compiled code has characteristic statistical patterns. Shannon entropy provides a quantitative measure to distinguish packed from unpacked binaries and identify encrypted regions.

### The Formula

The Shannon entropy of a byte sequence $X = (x_1, x_2, \ldots, x_n)$:

$$H(X) = -\sum_{i=0}^{255} p_i \log_2 p_i$$

where $p_i = \frac{\text{count}(x_i = i)}{n}$ is the frequency of byte value i. The maximum entropy for bytes is:

$$H_{\max} = \log_2 256 = 8 \text{ bits}$$

Entropy ranges for different content types:

$$H = \begin{cases} 0 & \text{uniform data (all same byte)} \\ 3.5-5.5 & \text{typical compiled code (x86)} \\ 5.0-6.5 & \text{typical data sections} \\ 7.0-7.5 & \text{compressed data} \\ 7.9-8.0 & \text{encrypted/random data} \end{cases}$$

The Kullback-Leibler divergence from a reference distribution (e.g., typical x86 code) to the observed distribution:

$$D_{KL}(P \| Q) = \sum_{i=0}^{255} p_i \log_2 \frac{p_i}{q_i}$$

Low $D_{KL}$ indicates the section matches the reference; high $D_{KL}$ indicates anomalous content.

### Worked Examples

**Example 1: Section entropy comparison**

Binary with 4 sections:
- .text: H = 5.8, size = 45 KB (normal code)
- .rdata: H = 4.2, size = 12 KB (strings and constants)
- .data: H = 3.1, size = 8 KB (initialized globals, many zeros)
- .rsrc: H = 7.6, size = 200 KB (suspicious -- likely packed payload)

.rsrc entropy 7.6 >> 5.5 threshold: strong indicator of encrypted/compressed content.

**Example 2: Sliding window entropy**

Firmware blob (1 MB), 256-byte sliding window. Entropy profile:
- Offset 0x0000-0x1000: H = 4.5 (bootloader, ARM code)
- Offset 0x1000-0x8000: H = 7.8 (encrypted firmware body)
- Offset 0x8000-0x8100: H = 2.1 (padding/alignment)
- Offset 0x8100-0xA000: H = 5.2 (cleartext config data)

The transition from H = 4.5 to H = 7.8 at offset 0x1000 marks the encryption boundary. The key or decryption routine likely resides in the bootloader region (0x0000-0x1000).

## 3. Pattern Matching and Signature Generation (String Algorithms)

### The Problem

Identifying known functions, library code, and malware signatures in stripped binaries requires efficient pattern matching. FLIRT (Fast Library Identification and Recognition Technology) and YARA use pattern signatures with wildcards. The challenge is generating signatures that are specific enough to avoid false positives while being robust enough to survive compiler variations.

### The Formula

A byte pattern with wildcards is a sequence over the alphabet $\Sigma = \{0x00, \ldots, 0xFF, ?\}$ where ? matches any byte. The specificity of a pattern of length L with w wildcard bytes:

$$\text{specificity} = \frac{L - w}{L}$$

The probability of a random match against uniformly distributed bytes:

$$P_{\text{false match}} = \left(\frac{1}{256}\right)^{L-w} = 256^{-(L-w)}$$

For a binary of size N bytes, the expected false positive count:

$$E[\text{FP}] = (N - L + 1) \cdot 256^{-(L-w)}$$

To achieve at most 1 expected false positive in a 10 MB binary:

$$256^{-(L-w)} \leq \frac{1}{10 \times 10^6}$$

$$(L-w) \geq \frac{\log(10^7)}{\log(256)} = \frac{7}{2.408} = 2.91$$

So at least 3 non-wildcard bytes needed for uniqueness (in practice, 16+ bytes for robustness).

### Worked Examples

**Example 1: YARA rule specificity**

YARA pattern: `{ 48 8B ?? ?? 48 89 ?? ?? FF 15 ?? ?? ?? ?? 85 C0 74 }` (17 bytes, 6 wildcards).

Specificity: (17-6)/17 = 64.7%.
P_false_match = 256^(-11) = 2^(-88) = 3.2 * 10^(-27).
In a 100 MB binary: E[FP] = 10^8 * 3.2 * 10^(-27) = 3.2 * 10^(-19) (effectively zero).

**Example 2: Minimal distinguishing pattern**

Two similar functions differing at 3 byte positions. Pattern must include at least one differing byte.
If functions share 95% of bytes (length 200, differ at 10 positions):
- Random 16-byte window hits a difference with probability: 1 - C(190,16)/C(200,16) = 1 - 0.434 = 0.566
- 56.6% chance a random window distinguishes them
- Three random windows: 1 - 0.434^3 = 0.918 (91.8% chance of distinction)

## 4. Symbolic Execution and Path Explosion (Complexity Theory)

### The Problem

Symbolic execution explores all feasible paths through a program by treating inputs as symbolic variables and collecting path constraints. While powerful for finding inputs that reach specific code (e.g., the "correct password" branch), the number of paths grows exponentially with branch points, limiting practical applicability.

### The Formula

For a program with d sequential binary decision points, the number of paths is:

$$|\text{paths}| = 2^d$$

With loops bounded by iteration count k, a loop with one branch inside contributes:

$$\text{paths}_{\text{loop}} = \sum_{i=0}^{k} 2^i = 2^{k+1} - 1$$

The total paths for a program with m loops of bound k and d non-loop branches:

$$|\text{paths}| = 2^d \cdot \prod_{j=1}^{m}(2^{k_j+1} - 1)$$

The constraint solving complexity at each path: for linear arithmetic constraints (conjunctions of $a_i x_i \leq b_i$), SMT solving is NP-complete. For bitvector arithmetic (actual machine integers), it is PSPACE-complete:

$$T_{\text{solve}} = O(2^{|vars|}) \quad \text{(worst case)}$$

State merging reduces paths by combining states with compatible constraints:

$$|\text{merged states}| \leq \min(|\text{paths}|, 2^{|vars|})$$

### Worked Examples

**Example 1: Password checker**

Sequential 8-character password check, each character compared independently (8 branches):
- Paths: 2^8 = 256
- Symbolic execution finds the correct path by solving 8 constraints
- Each constraint: `char[i] == expected[i]` (trivial for SMT)
- Total time: 256 * O(1) per constraint = O(256) (fast)

**Example 2: Hash verification**

Password hashed then compared: `if (sha256(input) == stored_hash)`.
- The hash function contains thousands of branches internally
- Symbolic execution through SHA-256: d approximately 20,000 branch points
- Paths: 2^20000 (intractable)
- Constraint: SHA-256 output equality is a system of bitvector equations
- SMT solver cannot invert SHA-256; symbolic execution fails here
- Solution: treat hash as black box, use concrete execution + fuzzing instead

**Example 3: Loop-dependent branch**

```
for (i = 0; i < n; i++) {
    if (buf[i] == key[i]) match++;
}
if (match == n) grant_access();
```

With n = 16: paths per iteration = 2, total = 2^16 = 65,536.
With merging (match counter as symbolic): states = 17 (0 to 16 matches).
Merging reduces 65,536 paths to 17 states (3,856x improvement).

## Prerequisites

- Graph theory (directed graphs, CFG, cyclomatic complexity, reachability)
- Information theory (Shannon entropy, KL divergence)
- String algorithms (pattern matching, Aho-Corasick, wildcards)
- Complexity theory (NP-completeness, decidability, halting problem)
- Computer architecture (x86-64 ISA, ARM, calling conventions)
- Abstract algebra (finite fields for cryptographic analysis)
- Satisfiability theory (SAT/SMT solving, constraint propagation)
- Compiler theory (SSA form, optimization passes, code generation patterns)
