# The Theory of Regular Expressions — Automata, Complexity, and Implementation

> *Regular expressions are formal specifications of regular languages. Thompson's construction converts a regex to an NFA in O(n). Subset construction converts an NFA to a DFA with worst-case 2^n states. Backtracking engines (PCRE, Python re) have exponential worst-case behavior. RE2/Go guarantee linear time by using automata directly.*

---

## 1. Formal Language Theory

### The Chomsky Hierarchy

| Level | Grammar | Automaton | Example |
|:------|:--------|:----------|:--------|
| 3 | Regular | Finite automaton (DFA/NFA) | `a*b+` |
| 2 | Context-free | Pushdown automaton | Balanced parentheses |
| 1 | Context-sensitive | Linear-bounded automaton | `a^n b^n c^n` |
| 0 | Unrestricted | Turing machine | Halting problem |

**True** regular expressions (without backreferences) recognize exactly the **regular languages** — Level 3.

### Regular Expression Operators

Three fundamental operators define all regular languages:

| Operator | Notation | Meaning |
|:---------|:---------|:--------|
| Concatenation | $rs$ | $r$ followed by $s$ |
| Alternation | $r \mid s$ | $r$ or $s$ |
| Kleene star | $r*$ | Zero or more repetitions of $r$ |

Everything else is **syntactic sugar**:
- `r+` = `rr*` (one or more)
- `r?` = `r|e` (zero or one, where $e$ = empty string)
- `[a-z]` = `a|b|c|...|z`
- `.` = any character (finite alternation)

---

## 2. Thompson's NFA Construction

### Algorithm

Convert regex to NFA in $O(n)$ time and $O(n)$ states (where $n$ = regex length):

**Base cases:**

```
Character 'a':     ──► (s) ─a─► ((f))

Empty string ε:    ──► (s) ─ε─► ((f))
```

**Inductive cases:**

**Concatenation** ($r \cdot s$): connect final state of $r$ to initial state of $s$:
```
──► [NFA for r] ─ε─► [NFA for s] ──►
```

**Alternation** ($r \mid s$): new start with $\varepsilon$-transitions to both:
```
         ┌─ε─► [NFA for r] ─ε─┐
──► (s) ─┤                     ├─► ((f))
         └─ε─► [NFA for s] ─ε─┘
```

**Kleene star** ($r*$):
```
         ┌───────────ε────────────┐
         │                        │
──► (s) ─┼─ε─► [NFA for r] ─ε─┬──┼─► ((f))
                    ▲          │
                    └────ε─────┘
```

### Worked Example: `a(b|c)*d`

Step 1: `a` → single-edge NFA
Step 2: `b|c` → alternation NFA (4 states)
Step 3: `(b|c)*` → Kleene star (6 states)
Step 4: `a(b|c)*d` → concatenation of three NFAs

Total: $O(n)$ states where $n = |\text{regex}|$.

---

## 3. Subset Construction — NFA to DFA

### Algorithm

Each DFA state corresponds to a **set of NFA states**:

1. Start: DFA start state = $\varepsilon\text{-closure}(\{q_0\})$
2. For each DFA state $S$ and input symbol $a$:
   - $S' = \varepsilon\text{-closure}(\bigcup_{q \in S} \delta(q, a))$
   - If $S'$ is new, add it as a DFA state
3. A DFA state is accepting if it contains any NFA accepting state

### Complexity

| Property | NFA | DFA |
|:---------|:----|:----|
| States | $O(n)$ | $O(2^n)$ worst case |
| Transitions per state | Multiple (nondeterministic) | Exactly one per symbol |
| Matching time | $O(n \times m)$ (simulate all states) | $O(m)$ (single state) |
| Space | $O(n)$ | $O(2^n)$ worst case |

Where $n$ = regex length, $m$ = input string length.

### Exponential Blowup Example

The regex `(a|b)*a(a|b){k}` (match strings where the k+1-th character from the end is `a`):
- NFA: $O(k)$ states
- DFA: $\Theta(2^k)$ states (must remember last $k$ characters)

---

## 4. Matching Strategies

### Strategy 1: Backtracking (PCRE, Python, Java, JavaScript)

Recursive descent over the regex, trying alternatives and backtracking on failure:

```
function match(regex, string, pos):
    if regex is empty: return pos == end
    if regex is "r|s": return match(r, string, pos) OR match(s, string, pos)
    if regex is "r*": try match(r, string, pos) then recurse, else skip
```

**Worst case:** Exponential time. The regex `(a+)+b` on input `"aaa...a"` (no `b`):

$$T(n) = O(2^n)$$

because the engine tries every possible way to partition the `a`s among the nested `+` operators.

### Catastrophic Backtracking — Worked Example

Regex: `(a+)+b`
Input: `"aaaa"` (4 a's, no b)

The engine tries all partitions of 4 a's:
- (aaaa) — fail at b
- (aaa)(a) — fail at b
- (aa)(aa) — fail at b
- (aa)(a)(a) — fail at b
- (a)(aaa) — fail at b
- (a)(aa)(a) — fail at b
- (a)(a)(aa) — fail at b
- (a)(a)(a)(a) — fail at b

That's $2^{n-1}$ partitions. For 30 a's: over 500 million attempts.

### Strategy 2: Automata Simulation (RE2, Go regexp, Rust regex)

Simulate the NFA directly, tracking all active states simultaneously:

$$\text{Time} = O(n \times m)$$

Where $n$ = regex size (number of NFA states), $m$ = input length.

**Guarantee:** Linear in input length for fixed regex. No catastrophic backtracking.

**Tradeoff:** Cannot support backreferences (`\1`), lookahead, or lookbehind — these go beyond regular languages.

---

## 5. DFA State Caching (Lazy DFA)

### The Hybrid Approach

RE2 uses a **lazy DFA**: build DFA states on demand, cache them, evict under memory pressure.

```
NFA states: {1, 3, 5}  ──cache lookup──►  DFA state 7 (cached)
                                           → transition on 'a' → DFA state 12
```

| Strategy | Build Time | Match Time | Memory |
|:---------|:-----------|:-----------|:-------|
| Full DFA | $O(2^n)$ up front | $O(m)$ | $O(2^n)$ |
| Lazy DFA | On-demand | $O(m)$ amortized | Bounded cache |
| NFA simulation | None | $O(nm)$ | $O(n)$ |

---

## 6. Extended Features Beyond Regular Languages

### Backreferences

`(a+)\1` matches `aa`, `aaaa`, `aaaaaa` — strings of $2k$ a's. This is **not a regular language** (it's context-sensitive). Matching with backreferences is **NP-complete**.

### Lookahead and Lookbehind

| Syntax | Name | Consumes Input? |
|:-------|:-----|:----------------|
| `(?=X)` | Positive lookahead | No |
| `(?!X)` | Negative lookahead | No |
| `(?<=X)` | Positive lookbehind | No |
| `(?<!X)` | Negative lookbehind | No |

Lookaheads can be implemented with automata (intersection of regular languages is regular). Lookbehinds require reverse matching.

### Atomic Groups and Possessive Quantifiers

`(?>a+)b` — once the group matches, it **never backtracks**. Prevents catastrophic backtracking at the cost of changing match semantics.

---

## 7. Regex Optimization Techniques

### Common Optimizations in Engines

| Optimization | Description | Speedup |
|:-------------|:------------|:--------|
| Literal prefix extraction | `/^foo\d+/` → search for "foo" first | Orders of magnitude |
| Boyer-Moore for literals | Skip characters based on pattern | Sublinear average |
| One-pass matching | Single-pass when no ambiguity | $O(m)$ guarantee |
| DFA minimization | Merge equivalent DFA states | Reduce state count |
| Character class compilation | `[a-z]` → bitmap lookup | $O(1)$ per character |

### Anchoring

| Anchor | Meaning | Optimization |
|:-------|:--------|:-------------|
| `^` | Start of string/line | Only try matching at position 0 |
| `$` | End of string/line | Can start search from end |
| `\b` | Word boundary | Skip non-boundary positions |

---

## 8. Complexity Summary

| Operation | Backtracking Engine | Automata Engine |
|:----------|:-------------------|:----------------|
| Compile | $O(n)$ | $O(n)$ NFA, $O(2^n)$ DFA |
| Match (best) | $O(m)$ | $O(m)$ |
| Match (worst) | $O(2^m)$ | $O(nm)$ NFA, $O(m)$ DFA |
| Backreferences | Supported (NP-complete) | Not supported |
| Lookahead | Supported | Limited |
| Memory | $O(n)$ stack depth | $O(n)$ NFA states or $O(2^n)$ DFA |

---

## 9. Summary of Key Theorems

| Theorem | Statement |
|:--------|:----------|
| Kleene's theorem | Regular expressions and finite automata define the same languages |
| Thompson's construction | Regex → NFA in $O(n)$ time and states |
| Subset construction | NFA → DFA, worst case $2^n$ states |
| Myhill-Nerode | Minimum DFA states = number of distinguishable string classes |
| Pumping lemma | If $L$ is regular, long strings can be "pumped" (repeated) |
| NP-completeness | Regex matching with backreferences is NP-complete |

---

*The gap between "regex" in theory (regular languages, linear time) and "regex" in practice (backreferences, exponential backtracking) is vast. Every catastrophic backtracking CVE, every ReDoS attack, exploits this gap. Know which engine you're using and whether it guarantees linear time.*

## Prerequisites

- Formal language theory (regular languages, context-free languages)
- Finite automata (DFA, NFA, NFA-to-DFA conversion)
- Backtracking algorithms and their failure modes
- Character encoding (ASCII, Unicode character classes)

## Complexity

| Operation | Engine | Time Complexity | Notes |
|---|---|---|---|
| Match (no backrefs) | DFA (RE2, Go) | O(n) | Linear time guaranteed |
| Match (no backrefs) | NFA (PCRE) | O(n) typical, O(2^n) worst | Backtracking can be exponential |
| Match (with backrefs) | PCRE/Perl | O(2^n) worst | Backreferences make the problem NP-hard |
| NFA construction | Thompson | O(m) | m = pattern length |
| DFA construction | Subset construction | O(2^m) worst | Exponential state explosion possible |
