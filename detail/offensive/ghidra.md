# The Mathematics of Ghidra — Decompilation Theory and Binary Analysis

> *Ghidra's decompilation pipeline transforms raw machine code into structured pseudo-C through a series of mathematically grounded transformations: control flow recovery as graph theory, data flow analysis as fixed-point lattice computation, type inference as constraint satisfaction, and P-Code as an algebraic intermediate representation enabling cross-architecture reasoning.*

---

## 1. Control Flow Recovery (Graph Theory)

### Basic Block Partitioning

A basic block $B$ is a maximal sequence of instructions with:

$$\text{single entry at top}, \quad \text{single exit at bottom}$$

The Control Flow Graph (CFG) is a directed graph $G = (V, E)$ where:

$$V = \{B_1, B_2, \ldots, B_n\}, \quad E \subseteq V \times V$$

### Edge Classification

| Edge Type | Condition | Graph Property |
|:---|:---:|:---:|
| Fall-through | Sequential execution | $(B_i, B_{i+1})$ |
| Conditional | Branch taken/not-taken | Out-degree = 2 |
| Unconditional | Direct jump | Out-degree = 1 |
| Indirect | Computed target | Out-degree = $k$ (resolved) |
| Call | Function invocation | Inter-procedural |

### Dominator Trees

Node $d$ dominates node $n$ ($d \text{ dom } n$) if every path from entry to $n$ passes through $d$:

$$\text{Dom}(n) = \{n\} \cup \left(\bigcap_{p \in \text{pred}(n)} \text{Dom}(p)\right)$$

The immediate dominator $\text{idom}(n)$ is the closest strict dominator. The dominator tree is computed in $O(E \cdot \alpha(V))$ using Lengauer-Tarjan.

### Loop Detection

Natural loops are identified via back edges. Edge $(n, h)$ is a back edge if $h$ dominates $n$:

$$\text{Loop}(h, n) = \{h\} \cup \{m \mid m \text{ can reach } n \text{ without passing through } h\}$$

---

## 2. Data Flow Analysis (Lattice Theory)

### Reaching Definitions

A definition $d: x = \text{expr}$ at program point $p$ reaches point $q$ if there exists a path from $p$ to $q$ with no redefinition of $x$.

$$\text{OUT}(B) = \text{GEN}(B) \cup (\text{IN}(B) - \text{KILL}(B))$$

$$\text{IN}(B) = \bigcup_{p \in \text{pred}(B)} \text{OUT}(p)$$

### Fixed-Point Iteration

Data flow equations are solved by iterating until convergence on a lattice $(L, \sqsubseteq)$:

$$x^{(k+1)} = F(x^{(k)}), \quad \text{terminate when } x^{(k+1)} = x^{(k)}$$

For a lattice of height $h$ with $n$ nodes, worst-case iterations:

$$\text{Iterations} \leq h \times n$$

### SSA Form (Static Single Assignment)

Ghidra converts to SSA for analysis. Each variable is assigned exactly once:

$$x_1 = 5, \quad x_2 = x_1 + 3, \quad x_3 = \phi(x_1, x_2)$$

The $\phi$-function merges values at control flow joins. Number of $\phi$-functions bounded by:

$$|\phi| \leq |V| \times |\text{variables}|$$

---

## 3. P-Code Algebra (Intermediate Representation)

### P-Code as Abstract Machine

P-Code defines an abstract register-transfer machine with operations:

| P-Code Op | Semantics | Algebraic Form |
|:---|:---:|:---:|
| COPY | $v_{\text{out}} \leftarrow v_{\text{in}}$ | Identity |
| INT_ADD | $v_{\text{out}} \leftarrow v_1 + v_2$ | Group addition |
| INT_MULT | $v_{\text{out}} \leftarrow v_1 \times v_2$ | Ring multiplication |
| INT_AND | $v_{\text{out}} \leftarrow v_1 \wedge v_2$ | Boolean lattice meet |
| LOAD | $v_{\text{out}} \leftarrow \text{Mem}[v_{\text{addr}}]$ | Memory dereference |
| STORE | $\text{Mem}[v_{\text{addr}}] \leftarrow v_{\text{in}}$ | Memory update |
| CBRANCH | if $v_{\text{cond}}$ goto $v_{\text{target}}$ | Conditional transfer |

### Instruction Lifting

Each native instruction maps to one or more P-Code operations. Lifting ratio:

$$R_{\text{lift}} = \frac{|P\text{-Code ops}|}{|N\text{ative instructions}|}$$

| Architecture | Typical Lifting Ratio | P-Code Ops per Instruction |
|:---|:---:|:---:|
| x86-64 | 3-5x | Complex CISC → multiple ops |
| ARM | 1.5-3x | Conditional execution expands |
| MIPS | 1.2-2x | RISC maps more directly |
| AVR | 1-1.5x | Simple 8-bit ops |

### Varnodes

P-Code operands are varnodes — (space, offset, size) tuples:

$$v = (\text{space}, \text{offset}, \text{size}) \in \mathcal{S} \times \mathbb{N} \times \mathbb{N}^+$$

Address spaces: `register`, `unique` (temporaries), `ram`, `const`, `stack`.

---

## 4. Type Inference (Constraint Satisfaction)

### Type Lattice

Ghidra's type system forms a lattice with $\top$ (unknown) and $\bot$ (conflict):

$$\bot \sqsubseteq \text{int8} \sqsubseteq \text{int16} \sqsubseteq \text{int32} \sqsubseteq \text{int64} \sqsubseteq \top$$

### Constraint Generation

Each P-Code operation generates type constraints:

$$\text{INT\_ADD}(v_1, v_2, v_3): \quad \tau(v_1) = \tau(v_2) = \tau(v_3) = \text{int}_n$$
$$\text{LOAD}(v_{\text{addr}}, v_{\text{out}}): \quad \tau(v_{\text{addr}}) = \text{ptr}(\tau(v_{\text{out}}))$$
$$\text{STORE}(v_{\text{addr}}, v_{\text{in}}): \quad \tau(v_{\text{addr}}) = \text{ptr}(\tau(v_{\text{in}}))$$

### Unification

Type constraints are solved via unification. Two types unify if:

$$\text{unify}(\tau_1, \tau_2) = \text{mgu}(\tau_1, \tau_2) \text{ (most general unifier)}$$

Complexity of Hindley-Milner unification: $O(n \cdot \alpha(n))$ for $n$ constraints, where $\alpha$ is the inverse Ackermann function.

### Struct Recovery

Field access patterns reveal struct layouts:

$$\text{LOAD}(\text{base} + 0) \Rightarrow \text{field}_0, \quad \text{LOAD}(\text{base} + 8) \Rightarrow \text{field}_1$$

Ghidra infers: `struct { type0 field_0; type1 field_1; }` with alignment:

$$\text{sizeof}(\text{struct}) = \max_i(\text{offset}_i + \text{sizeof}(\text{field}_i))$$

---

## 5. Decompilation Pipeline (Transformation Algebra)

### Pipeline Stages

$$\text{Binary} \xrightarrow{\text{Lift}} \text{P-Code} \xrightarrow{\text{SSA}} \text{SSA P-Code} \xrightarrow{\text{Simplify}} \text{Optimized} \xrightarrow{\text{Structure}} \text{AST} \xrightarrow{\text{Emit}} \text{C}$$

### Action Groups

Ghidra's decompiler applies transformation rules in groups:

| Action Group | Purpose | Transformations |
|:---|:---:|:---:|
| Heritage | SSA construction | $\phi$-insertion, renaming |
| Dead Code | Eliminate unused ops | Liveness, flag removal |
| Propagation | Constant/copy prop | Substitution, folding |
| Normalize | Canonical forms | Commutative reorder |
| Type Recovery | Infer types | Constraint solving |
| Restructure | Recover control flow | Interval analysis |

### Constant Propagation

If $v = c$ (constant) at every reaching definition, replace $v$ with $c$:

$$\text{val}(v) = \begin{cases} c & \text{if all reaching defs assign } c \\ \top & \text{if no reaching defs} \\ \bot & \text{if multiple different constants} \end{cases}$$

---

## 6. Control Flow Structuring (Interval Analysis)

### Structural Analysis

Ghidra recovers high-level constructs (if/else, while, for, switch) from the CFG using interval analysis:

$$\text{Reduce}: G \rightarrow G' \text{ by collapsing recognized patterns}$$

### Pattern Recognition

| Pattern | CFG Shape | C Construct |
|:---|:---:|:---:|
| If-then | Diamond, one empty | `if (cond) { ... }` |
| If-then-else | Diamond, both filled | `if (cond) { A } else { B }` |
| While loop | Back edge to header | `while (cond) { ... }` |
| Do-while | Back edge from tail | `do { ... } while (cond)` |
| Switch | Multi-way branch | `switch (val) { case: ... }` |
| Break/Continue | Loop exit/back edge | `break; continue;` |

### Irreducible Control Flow

Some CFGs have no structured equivalent ($goto$ required):

$$\text{Reducible if every cycle has a single entry node (header)}$$

Percentage of irreducible CFGs in practice:

| Source | Reducible | Irreducible |
|:---|:---:|:---:|
| Compiled C/C++ | 99.5% | 0.5% |
| Hand-written assembly | 85% | 15% |
| Obfuscated malware | 60-80% | 20-40% |

---

## 7. Signature Matching (Function Identification)

### Function ID (FID) Database

FID computes hashes over function bodies for library identification:

$$h_{\text{full}} = \text{Hash}(\text{bytes}[f_{\text{start}} : f_{\text{end}}])$$
$$h_{\text{specific}} = \text{Hash}(\text{bytes} \setminus \text{relocations})$$

### Match Scoring

$$\text{Score}(f, \text{sig}) = \frac{|\text{matched bytes}|}{|\text{total bytes}|} \times w_{\text{hash}} + \frac{|\text{matched refs}|}{|\text{total refs}|} \times w_{\text{ref}}$$

| Match Quality | Score Range | Action |
|:---|:---:|:---:|
| Exact | 1.0 | Auto-apply name + types |
| High | 0.8 - 1.0 | Apply with review |
| Medium | 0.5 - 0.8 | Suggest, manual review |
| Low | < 0.5 | Ignore |

### Collision Probability

For $n$ functions and $b$-bit hash:

$$P(\text{collision}) \approx 1 - e^{-n^2 / 2^{b+1}}$$

With 64-bit hashes and 10,000 functions: $P \approx 2.7 \times 10^{-12}$.

---

## 8. Binary Diffing (Version Tracking Theory)

### Similarity Metrics

Function similarity between versions $f_A$ and $f_B$:

$$\text{Sim}_{\text{structural}} = \frac{2 \times |\text{matched blocks}|}{|V_A| + |V_B|}$$

$$\text{Sim}_{\text{instruction}} = 1 - \frac{\text{edit\_distance}(\text{ops}_A, \text{ops}_B)}{\max(|\text{ops}_A|, |\text{ops}_B|)}$$

### Correlator Cascade

Version Tracking applies correlators in priority order:

$$C_1 \rightarrow C_2 \rightarrow \cdots \rightarrow C_k$$

Each correlator $C_i$ produces match candidates, filtered by confidence threshold $\theta_i$:

$$M_i = \{(f_A, f_B) \mid \text{Sim}_{C_i}(f_A, f_B) \geq \theta_i\}$$

Remaining unmatched functions propagate to the next correlator.

---

*The power of Ghidra's analysis lies in composing these mathematical foundations: graph theory recovers structure, lattice theory computes properties, constraint satisfaction infers types, and algebraic transformations produce readable code. Understanding these foundations enables analysts to diagnose decompiler failures, write more effective scripts, and push the boundaries of automated binary analysis.*

## Prerequisites

- Graph theory fundamentals (directed graphs, dominators, strongly connected components)
- Basic lattice theory and fixed-point computation
- Familiarity with compiler intermediate representations and SSA form

## Complexity

- **Beginner:** Understanding CFG construction, basic block identification, and navigation of P-Code output
- **Intermediate:** Writing analysis scripts using data flow equations, applying type inference to recover struct layouts
- **Advanced:** Extending the decompiler with custom P-Code injection rules, building correlators for version tracking, and handling irreducible control flow in obfuscated binaries
