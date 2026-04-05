# The Mathematics of Code Coverage — Metrics, Complexity, and Theoretical Limits

> *Coverage numbers are among the most cited yet most misunderstood metrics in software engineering. This explores the mathematical relationships between coverage types, cyclomatic complexity and its connection to path explosion, MC/DC as the avionics standard, and the theoretical limits of what coverage can and cannot tell you.*

---

## 1. Coverage Metric Mathematics (Set Theory)

### The Problem

Different coverage metrics measure different things. What are the precise mathematical definitions, and what are the subset relationships between them?

### The Formula

Let $P$ be a program with:
- $S = \{s_1, \ldots, s_n\}$: set of statements
- $B = \{b_1, \ldots, b_m\}$: set of branch edges (each decision has true/false edges)
- $C = \{c_1, \ldots, c_k\}$: set of conditions (atomic boolean expressions)
- $\Pi = \{\pi_1, \ldots, \pi_p\}$: set of execution paths

For a test suite $T$, define the **executed set** $E_T(X) \subseteq X$ as the elements actually exercised.

**Statement coverage**: $SC = \frac{|E_T(S)|}{|S|}$

**Branch coverage**: $BC = \frac{|E_T(B)|}{|B|}$

**Condition coverage**: $CC = \frac{|E_T(C \times \{T,F\})|}{2|C|}$

**Path coverage**: $PC = \frac{|E_T(\Pi)|}{|\Pi|}$

**Subsumption hierarchy** (proven):

$$PC \implies MC/DC \implies BC \implies SC$$

Meaning: 100% path coverage guarantees 100% branch coverage, which guarantees 100% statement coverage. The reverse does not hold.

**Counterexample** (SC does not imply BC):

```go
if a || b {
    x = 1  // statement S1
}
```

Test: $a = true, b = true$. Statement coverage of S1: 100%. But branch `(a||b) = false` is never taken. Branch coverage: 50%.

### Worked Examples

**Example**: Function with 3 sequential if-statements, each with an else branch.

- Statements: $|S| = 6$ (3 true branches + 3 false branches)
- Branches: $|B| = 6$ (true/false for each if)
- Paths: $|\Pi| = 2^3 = 8$ (each if is independent)

Minimum tests for:
- 100% statement coverage: 2 tests (all-true, all-false)
- 100% branch coverage: 2 tests (same)
- 100% path coverage: 8 tests (all combinations)

## 2. Cyclomatic Complexity vs Path Count (Graph Theory)

### The Problem

McCabe's cyclomatic complexity $V(G)$ is often equated with the number of test cases needed. What is the actual relationship between complexity and the number of paths?

### The Formula

For a control flow graph $G = (V, E)$ with $V$ nodes, $E$ edges, and $P$ connected components:

$$V(G) = E - V + 2P$$

For a single function ($P = 1$):

$$V(G) = E - V + 2$$

Equivalent formulation: $V(G) = $ number of decision points + 1.

**Relationship to paths**:

$$V(G) \leq |\Pi| \leq 2^{V(G) - 1}$$

The lower bound is the number of linearly independent paths (basis paths). The upper bound is the maximum for fully independent decisions.

**With loops**, path count becomes infinite (each loop iteration is a different path). Practical analysis bounds loop iterations:

$$|\Pi_{bounded}| = \prod_{i=1}^{d} (k_i + 1) \cdot 2^{c}$$

where $d$ = number of loops, $k_i$ = max iterations of loop $i$, $c$ = number of non-loop conditionals.

### Worked Examples

**Example**: Function with cyclomatic complexity $V(G) = 5$.

- Minimum basis paths: 5
- Maximum independent paths: $2^{5-1} = 16$
- With one loop (max 3 iterations) and 3 conditionals: $|\Pi| = (3+1) \times 2^3 = 32$

Branch coverage requires $\geq 2 \times 4 = 8$ test cases (2 per decision for 4 decisions). But path coverage requires up to 32.

**Complexity distribution** in real codebases (empirical):

| $V(G)$ range | Risk level | Typical coverage achievable |
|-------------|-----------|---------------------------|
| 1-10 | Low | 95-100% path coverage feasible |
| 11-20 | Moderate | Branch coverage practical, path coverage hard |
| 21-50 | High | Focus on branch + MC/DC |
| 50+ | Very high | Refactor before testing |

## 3. MC/DC — Modified Condition/Decision Coverage (Boolean Analysis)

### The Problem

MC/DC is mandated by DO-178C Level A (flight-critical avionics software). It requires that each condition independently affects the decision outcome. What is the minimum test set?

### The Formula

For a decision $D$ with $n$ conditions $c_1, c_2, \ldots, c_n$:

**MC/DC requirement**: for each condition $c_i$, there must exist two test cases that:
1. Differ only in the value of $c_i$ (all other conditions fixed)
2. The decision $D$ evaluates to different outcomes

**Minimum test cases**: $n + 1$ (for decisions where conditions are not masked).

**For coupled conditions** (where changing one condition forces another to change), the minimum may be higher, up to $2n$.

**Unique-cause MC/DC**: strictly one condition differs between paired test cases.

**Masking MC/DC**: allows multiple condition changes if the extras are "masked" (their values don't affect the outcome).

### Worked Examples

**Example**: Decision $D = (a \wedge b) \vee c$ with 3 conditions.

Truth table with MC/DC pairs:

| Test | a | b | c | D | Demonstrates |
|------|---|---|---|---|-------------|
| T1 | T | T | F | T | baseline for a, b |
| T2 | F | T | F | F | a independently affects D (vs T1) |
| T3 | T | F | F | F | b independently affects D (vs T1) |
| T4 | F | F | T | T | c independently affects D |
| T5 | F | F | F | F | pair for c (vs T4) |

5 test cases for 3 conditions. But we can optimize:

| Test | a | b | c | D | Pairs |
|------|---|---|---|---|-------|
| T1 | T | T | F | T | a: (T1,T2), b: (T1,T3) |
| T2 | F | T | F | F | |
| T3 | T | F | F | F | |
| T4 | F | F | T | T | c: (T4,T5) |
| T5 | F | F | F | F | |

Minimum: 4 test cases (T1, T2, T3, T4 — with T5 = T2 or T3 serving as c's pair).

Actually: $n + 1 = 4$ tests suffice for unique-cause MC/DC.

## 4. Theoretical Coverage Limits (Undecidability)

### The Problem

Can we ever achieve 100% path coverage? Are there theoretical limits to what coverage can guarantee?

### The Formula

**Dead code detection** is undecidable in general (reduces to the halting problem). A statement $s$ is dead if no input reaches it:

$$\text{dead}(s) \iff \forall i \in I : s \notin \text{trace}(P, i)$$

This is equivalent to asking "does there exist an input that reaches $s$?" — which is undecidable for Turing-complete languages.

**Practical implication**: static analysis can identify *some* dead code but cannot identify *all* dead code. Coverage tools can identify code that *wasn't* reached but cannot determine if it *could* be reached.

**Coverage vs correctness** — the fundamental limitation:

$$\text{100% coverage} \not\Rightarrow \text{correctness}$$

Proof by counterexample:

```go
func Add(a, b int) int {
    return a * b  // BUG: should be a + b
}
```

Test: `assert Add(2, 2) == 4` — achieves 100% statement, branch, and path coverage. But the function is incorrect for all inputs except $(0,0), (2,2)$.

**Bernstein's coverage model**: the probability that a test suite with coverage $c$ catches a randomly placed fault:

$$P(\text{detect fault}) \approx c^{1/k}$$

where $k$ is the "defect coupling" factor (typically 1.5-3). For $c = 0.80$ and $k = 2$:

$$P \approx 0.80^{0.5} = 0.894$$

89.4% detection probability — significantly less than the 80% coverage might suggest.

### Worked Examples

**Example**: A test suite achieves 90% statement coverage.

With $k = 2$ (moderate coupling):
$$P(\text{detect}) = 0.90^{0.5} = 0.949$$

With $k = 3$ (low coupling — faults in rarely-executed code):
$$P(\text{detect}) = 0.90^{0.333} = 0.966$$

The uncovered 10% may contain faults that are disproportionately dangerous (error paths, edge cases, security checks).

## 5. Coverage Composition (Multi-Package Analysis)

### The Problem

When testing a system of packages $P_1, P_2, \ldots, P_n$, how does per-package coverage relate to system-level coverage?

### The Formula

**Weighted system coverage**:

$$C_{system} = \frac{\sum_{i=1}^{n} w_i \cdot C_i}{\sum_{i=1}^{n} w_i}$$

where $w_i = |S_i|$ (number of statements in package $i$) and $C_i$ is the coverage of package $i$.

**Cross-package coverage** (with `-coverpkg=./...`):

$$C_{cross}(P_j | T_i) = \text{coverage of } P_j \text{ when running tests of } P_i$$

The total coverage of $P_j$ across all test suites:

$$C_{total}(P_j) = 1 - \prod_{i=1}^{n}\left(1 - C_{cross}(P_j | T_i)\right)$$

This uses the independence assumption — each test suite covers different parts.

### Worked Examples

**Example**: 3 packages with statement counts and per-package coverage:

| Package | Statements | Own test coverage | Cross-package coverage from others |
|---------|-----------|-------------------|-----------------------------------|
| handler | 200 | 85% | 10% from store tests |
| store | 150 | 90% | 5% from handler tests |
| auth | 100 | 70% | 15% from handler tests |

System coverage (own tests only):
$$C_{system} = \frac{200 \times 0.85 + 150 \times 0.90 + 100 \times 0.70}{450} = \frac{170 + 135 + 70}{450} = 83.3\%$$

Auth total coverage with cross-package:
$$C_{total}(\text{auth}) = 1 - (1-0.70)(1-0.15) = 1 - 0.30 \times 0.85 = 74.5\%$$

## Prerequisites

- Set theory (subsets, intersections, cardinality)
- Graph theory (control flow graphs, cyclomatic complexity)
- Boolean algebra (conditions, decisions, masking)
- Basic probability (independence, product rule)

## Complexity

| Analysis | Time Complexity | Space Complexity |
|----------|----------------|-----------------|
| Statement coverage collection | $O(|S|)$ per test | $O(|S|)$ bitmap |
| Branch coverage collection | $O(|B|)$ per test | $O(|B|)$ bitmap |
| Path enumeration | $O(2^{V(G)})$ worst case | $O(|\Pi|)$ |
| MC/DC test generation | $O(n \cdot 2^n)$ worst case | $O(2^n)$ |
| Coverage profile merging | $O(F \cdot S)$ per profile | $O(F \cdot S)$ |

Where: $|S|$ = statements, $|B|$ = branches, $V(G)$ = cyclomatic complexity, $n$ = conditions per decision, $F$ = files, $S$ = statements per file.
