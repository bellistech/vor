# The Internals of Regex — Engine Theory, NFA vs DFA, and Catastrophic Backtracking

> *Regular expressions are the user-facing skin of automata theory. Underneath the syntax sits a precise correspondence — regex ↔ NFA ↔ DFA — established by Kleene in 1956 and made constructive by Ken Thompson's 1968 paper. Every modern engine is a point on the spectrum between Thompson's linear-time NFA simulation and Perl's exponential-worst-case backtracker. Understanding which point your engine occupies is the difference between a 50-microsecond match and a thirty-second ReDoS outage.*

---

## 1. Regex Theory — Formal Languages

### 1.1 The Equivalence Triangle

The fundamental result of regular language theory is the equivalence of three formalisms:

```
       Regular Expression
              ▲
           │     │
           ▼     ▼
       NFA  ⇄  DFA
```

Kleene's theorem (1956) establishes that the languages accepted by finite automata are exactly the languages denoted by regular expressions. The constructive direction — from regex to a runnable matcher — is what every engine implements. The triangle has three legs:

1. **Regex → NFA** — Thompson's construction, O(M) time and states.
2. **NFA → DFA** — subset construction, O(2^M) worst-case states.
3. **DFA → regex** — state elimination via Brzozowski's algebraic method.

### 1.2 The Chomsky Hierarchy and Where Regex Lives

| Level | Grammar           | Automaton                  | Example language        | Regex matches?     |
|:------|:------------------|:---------------------------|:------------------------|:-------------------|
| 3     | Regular           | DFA / NFA                  | `(ab)*`                 | Yes — by definition |
| 2     | Context-free      | Pushdown automaton         | balanced parentheses    | No                 |
| 1     | Context-sensitive | Linear-bounded automaton   | `aⁿbⁿcⁿ`                | No (but backrefs do) |
| 0     | Unrestricted      | Turing machine             | the halting problem     | No                 |

Pure regular expressions (no backreferences, no recursion) sit firmly at Level 3. The moment you add `\1`-style backreferences, you've left the regular world — proven NP-complete by Aho (1990). Modern PCRE-style engines with recursion `(?R)` and subroutines `(?&name)` actually venture into context-free territory, which is part of why they cannot offer linear-time guarantees.

### 1.3 The Three Operators That Generate Everything

A regular expression over alphabet Σ is built from three operators:

| Operator       | Notation | Meaning                              |
|:---------------|:---------|:-------------------------------------|
| Concatenation  | `rs`     | r followed by s                      |
| Alternation    | `r\|s`   | r or s                               |
| Kleene star    | `r*`     | zero or more repetitions of r        |

Plus two constants: `ε` (empty string) and `∅` (empty language). Everything else — `+`, `?`, `[a-z]`, `\d`, `{m,n}` — is syntactic sugar:

```
r+   ≡  rr*
r?   ≡  r|ε
[abc] ≡  a|b|c
r{3} ≡  rrr
r{2,4} ≡ rr(r(r)?)?
.    ≡  finite alternation over the alphabet (ASCII or Unicode)
```

This minimal core is what Thompson's construction handles directly; the engine front-end desugars everything else before construction begins.

### 1.4 Cost of Each Construction

| Conversion          | Time    | Space (states)   | Notes                            |
|:--------------------|:--------|:-----------------|:---------------------------------|
| Regex → NFA         | O(M)    | O(M)             | Thompson's construction          |
| NFA simulation      | O(N·M)  | O(M)             | Active state set; N = input len  |
| NFA → DFA           | O(2^M)  | O(2^M) worst     | Subset construction              |
| DFA simulation      | O(N)    | O(2^M) states    | One transition per char          |
| DFA minimization    | O(K log K) | O(K) for K states | Hopcroft's algorithm           |

The deep insight: NFA simulation gives O(N·M) time using O(M) space. DFA gives O(N) time but pays in space. RE2 chooses the **lazy DFA**: build DFA states from NFA states on demand, cache them, evict under memory pressure. This is "on-the-fly subset construction" — you get DFA speed for the states you actually visit, with bounded memory.

---

## 2. Engine Implementations Compared

### 2.1 The Two Families

Every regex engine in widespread use falls into one of two families:

**Backtracking (recursive descent over the regex):**
- Perl, PCRE, PCRE2, Python `re`, Java `java.util.regex`, .NET `Regex` (default), Ruby Onigmo, JavaScript V8/SpiderMonkey/JavaScriptCore.
- Supports backreferences, lookaround, recursion, conditional groups, possessive quantifiers.
- Worst-case exponential time in input length.

**Automata-based (NFA/DFA simulation):**
- RE2 (Google), Go `regexp`, Rust `regex`, Hyperscan (Intel), GNU `grep -E` (with caching), .NET `RegexOptions.NonBacktracking` (since 2022).
- No backreferences, no lookaround in pure form.
- Linear-time guarantee.

### 2.2 Feature Matrix

| Feature                        | PCRE2 | Perl | Python re | Java | JS (V8) | .NET | RE2/Go | Rust regex | Hyperscan |
|:-------------------------------|:------|:-----|:----------|:-----|:--------|:-----|:-------|:-----------|:----------|
| Backreferences `\1`            | Yes   | Yes  | Yes       | Yes  | Yes     | Yes  | No     | No         | No        |
| Lookahead `(?=)`               | Yes   | Yes  | Yes       | Yes  | Yes     | Yes  | No     | No (ext)   | No        |
| Lookbehind fixed `(?<=)`       | Yes   | Yes  | Yes       | Yes  | Yes     | Yes  | No     | No (ext)   | No        |
| Variable-width lookbehind      | Yes   | Yes  | regex pkg | 9+   | Yes     | Yes  | No     | No         | No        |
| Atomic group `(?>)`            | Yes   | Yes  | 3.11+     | Yes  | No      | Yes  | No     | No         | No        |
| Possessive `*+ ++ ?+`          | Yes   | Yes  | 3.11+     | Yes  | No      | Yes  | No     | No         | No        |
| Recursion `(?R)`               | Yes   | Yes  | regex pkg | No   | No      | No   | No     | No         | No        |
| Named groups                   | Yes   | Yes  | Yes       | Yes  | Yes     | Yes  | Yes    | Yes        | No        |
| Unicode `\p{L}`                | Yes   | Yes  | Yes       | Yes  | u flag  | Yes  | Yes    | Yes        | Limited   |
| Linear-time guarantee          | No    | No   | No        | No   | No      | Opt  | Yes    | Yes        | Yes       |
| Multi-pattern parallel         | No    | No   | No        | No   | No      | No   | No     | RegexSet   | Yes       |

### 2.3 What Each Engine Actually Does Internally

**PCRE2** — compiles regex into an internal opcode list (a kind of bytecode), then executes via a tree-walking backtracking interpreter. Includes a JIT (`pcre2_jit_compile`) that emits native code via the Sljit framework. The execution model is "match the first opcode against the cursor; on success advance; on failure rewind via the saved-state stack."

**Python `re`** — written in C, compiles to a similar bytecode, executes via a C interpreter. The state machine is in `Modules/_sre/sre_lib.h`. Python 3.11 added atomic groups and possessive quantifiers; the third-party `regex` module (drop-in replacement) has variable-width lookbehind, fuzzy matching, and Unicode property tweaks.

**Java `Pattern`** — also a backtracking engine, compiled to a tree of `Node` objects in the `java.util.regex` package. Each node implements `match(Matcher, int, CharSequence)`. Java added variable-width lookbehind in JDK 9 and continues to extend feature support.

**JavaScript engines (V8, SpiderMonkey, JavaScriptCore)** — V8 uses the Irregexp engine, originally a backtracking interpreter, now also a JIT. JavaScriptCore has YARR (Yet Another Regex Runtime). All are backtracking. V8 added an experimental linear-time mode for a subset of features (`V8_FLAG_enable_experimental_regexp_engine`).

**.NET `Regex`** — three modes: interpreted, compiled (emits IL), and `RegexOptions.NonBacktracking` (added in .NET 7, 2022). NonBacktracking uses **Brzozowski derivatives** to build a symbolic DFA on the fly, providing linear-time matching. The compiled mode emits a method that does the matching directly via JIT.

**RE2** — written by Russ Cox (Google). Builds a Thompson NFA, then runs three simulators in priority order: a "one-pass" matcher for unambiguous regexes (linear, no backtracking), a "bit-state" backtracker for short inputs, and a lazy DFA for long inputs. The DFA cache is a hash table from NFA-state-set to DFA-state, with LRU eviction.

**Go `regexp`** — direct port of RE2's algorithm in Go. Same guarantees, same limitations.

**Rust `regex`** — written by Andrew Gallant (BurntSushi). Architecture: parser → HIR (high-level IR) → NFA → optional DFA. Selects between Pike VM, bounded backtracker, lazy DFA, and one-pass matcher per regex. Uses `aho-corasick` for literal optimizations and `memchr` for SIMD prefix scanning.

**Hyperscan** — Intel's library, compiles many regexes into a single multi-pattern matcher using SIMD instructions. Used in Suricata IDS to scan packets against thousands of signatures simultaneously. Architecture: NFA → "bounded repeats" → vectorized DFA → SIMD scanning.

### 2.4 Trade-off Matrix

| Property              | Backtracking                  | Automata                      |
|:----------------------|:------------------------------|:------------------------------|
| Time complexity       | O(2^N) worst, O(N) typical    | O(N) guaranteed               |
| Backreferences        | Native                        | Impossible (not regular)      |
| Lookaround            | Easy                          | Possible (intersection) but rare |
| Capturing groups      | Native                        | Requires extra bookkeeping    |
| Sub-match precision   | Leftmost first                | Leftmost-longest or first     |
| Untrusted input safe  | No (ReDoS)                    | Yes                           |
| Easy to extend        | Yes                           | Limited                       |
| Memory                | O(M) stack                    | O(M) NFA, up to O(2^M) DFA    |

Pick backtracking when you need expressive features and control your input. Pick automata when input is hostile, throughput matters, or you're matching at scale.

---

## 3. Thompson's NFA Construction

### 3.1 The Recursive Algorithm

Ken Thompson's 1968 CACM paper "Regular Expression Search Algorithm" describes the construction. For each regex piece, produce an NFA fragment with exactly one entry and one exit state.

**Base case — single character `a`:**

```
   ─►( s )──a──►(( f ))
```

**Base case — empty string `ε`:**

```
   ─►( s )──ε──►(( f ))
```

**Concatenation `rs` — splice exit of r to entry of s:**

```
   ─►[ NFA(r) ]──ε──►[ NFA(s) ]──►
```

**Alternation `r|s` — new start with ε-fan-out, new end with ε-fan-in:**

```
                ┌──ε──►[ NFA(r) ]──ε──┐
   ─►( s' )─────┤                      ├─►(( f' ))
                └──ε──►[ NFA(s) ]──ε──┘
```

**Kleene star `r*` — bypass and loop edges:**

```
                  ┌────────ε─────────┐
                  │                  ▼
   ─►( s' )──ε──►[ NFA(r) ]──ε──►(( f' ))
                  ▲                  │
                  └────────ε─────────┘
```

**One-or-more `r+`** — same as above without the bypass ε-edge.

**Optional `r?`** — same as alternation with `r|ε`.

### 3.2 Properties of Thompson NFAs

| Property                          | Value                       |
|:----------------------------------|:----------------------------|
| States for length-M regex         | At most 2M                  |
| Transitions per state             | At most 2 (one ε, one char) |
| Has exactly one start state       | Yes                         |
| Has exactly one accept state      | Yes                         |
| ε-transitions allowed             | Yes (essential)             |

This regular structure makes simulation cheap: each input character advances every active state by at most one edge. With M states, the worst case is M active states, so each character costs O(M), giving O(N·M) total.

### 3.3 Worked Walkthrough: `a(b|c)*d`

Step 1 — `a`:
```
(0)──a──►(1)
```

Step 2 — `b`:
```
(2)──b──►(3)
```

Step 3 — `c`:
```
(4)──c──►(5)
```

Step 4 — `b|c` (alternation, new states 6 and 7):
```
       ┌─ε─►(2)──b──►(3)─ε─┐
(6)────┤                    ├─►(7)
       └─ε─►(4)──c──►(5)─ε─┘
```

Step 5 — `(b|c)*` (Kleene star, new states 8 and 9):
```
            ┌──────────ε───────────┐
            │                      ▼
(8)──ε─►(6)─...─(7)──ε─►(9)
            ▲                      │
            └──────ε───────────────┘
```

Step 6 — concatenate `a · (b|c)* · d`:
```
(0)──a──►(1)──ε──►(8)──...──►(9)──ε──►(10)──d──►(11)
```

Total: 12 states. Linear in regex length, as promised.

### 3.4 Reference Implementation in C-ish Pseudocode

```c
typedef struct State {
    int c;              /* character or epsilon */
    struct State *out;  /* primary transition */
    struct State *out1; /* alternation transition */
    int lastlist;       /* simulation bookkeeping */
} State;

typedef struct Frag {
    State *start;
    PtrList *out;       /* dangling outputs to be patched */
} Frag;

Frag literal(int c) {
    State *s = newstate(c, NULL, NULL);
    return (Frag){s, list1(&s->out)};
}

Frag concat(Frag a, Frag b) {
    patch(a.out, b.start);
    return (Frag){a.start, b.out};
}

Frag alt(Frag a, Frag b) {
    State *s = newstate(EPSILON, a.start, b.start);
    return (Frag){s, append(a.out, b.out)};
}

Frag star(Frag a) {
    State *s = newstate(EPSILON, a.start, NULL);
    patch(a.out, s);
    return (Frag){s, list1(&s->out1)};
}
```

This is essentially Russ Cox's famous one-page implementation from his 2007 article. It produces a digraph of `State` records — what Cox memorably calls "smashing the regex into a DAG."

---

## 4. Subset Construction — NFA to DFA

### 4.1 The Algorithm

A DFA simulates an NFA by tracking the **set of NFA states currently active**. Each unique set becomes a single DFA state. The construction:

1. Compute `ε-closure(start)` — the start state of the DFA is the set of all NFA states reachable from the NFA start by ε-edges.
2. For each unprocessed DFA state S and each input symbol c:
   - Compute `move(S, c)` — the set of NFA states reachable from any state in S by a c-edge.
   - Compute `ε-closure(move(S, c))` — the new DFA state.
   - If new, add to the worklist.
3. A DFA state is accepting if any of its NFA states is accepting.

### 4.2 ε-Closure Computation

```python
def epsilon_closure(states):
    stack = list(states)
    closure = set(states)
    while stack:
        s = stack.pop()
        for t in s.epsilon_transitions:
            if t not in closure:
                closure.add(t)
                stack.append(t)
    return frozenset(closure)
```

### 4.3 The Worst-Case State Explosion

Some regexes force the DFA to track exponentially many distinct subsets. Canonical example:

```
(a|b)*a(a|b){k}
```

This says "match any string whose (k+1)-th character from the end is `a`." The NFA has O(k) states. The DFA must remember the last k characters seen, requiring 2^k distinct states. Building it eagerly is catastrophic.

### 4.4 Lazy Subset Construction (RE2's Trick)

Instead of building the entire DFA upfront, build only the states you actually visit:

```
Cache: NFA-state-set → DFA-state
On each input character at runtime:
    1. Look up current DFA state.
    2. If transition for c is cached, follow it.
    3. Else compute move(S, c) → ε-closure → look up or create new DFA state.
    4. Cache the transition.
    5. If cache exceeds budget, evict (LRU or full reset).
```

For "normal" regexes the visited DFA is small. The bad cases gracefully degrade to NFA simulation when the cache is full. This is the algorithmic core of RE2's linear-time guarantee.

### 4.5 Hopcroft Minimization

Once you have a DFA, you can minimize it by partitioning states by behavioral equivalence. Hopcroft's algorithm runs in O(K log K) for K states. The minimal DFA is unique up to state renaming (Myhill-Nerode theorem).

| Phase           | Time            | Output                    |
|:----------------|:----------------|:--------------------------|
| Thompson        | O(M)            | NFA with ≤ 2M states      |
| Subset          | O(2^M · S · |Σ|) | DFA with ≤ 2^M states     |
| Hopcroft min    | O(K log K)      | Unique minimal DFA        |

Practical engines often skip minimization because the wins are small relative to the cost; RE2's lazy DFA avoids the issue entirely.

---

## 5. Catastrophic Backtracking

### 5.1 The Canonical Pathological Pattern

```
^(a+)+$
```

Apply to `"aaaaaaaaaaaaaaaaaaaaaaaaaaaab"` (28 a's followed by a b that won't match). The backtracker tries every possible way to partition the 28 a's between the inner `a+` matches.

For input of n a's followed by a non-match, the number of partitions is 2^(n-1). Each partition is a separate backtracking attempt. For n=30 that's about 500 million attempts; for n=40 it's a half-trillion — minutes of CPU time.

```
attempts(n a's):
  n=10  →     512
  n=20  →     524,288
  n=25  →     16,777,216
  n=30  →     536,870,912    (~5 seconds)
  n=35  →  17,179,869,184    (~3 minutes)
  n=40  → 549,755,813,888    (~hours)
```

### 5.2 Why It Happens

The engine sees `(a+)+` and asks: "How many a's should the outer group consume? How many should the inner group consume?" Then it tries every combination, abandoning each only when the final `$` anchor fails.

There's no algorithmic reason for this — the regex matches exactly the same language as `a+`. The engine just doesn't know that. A DFA-based engine would compile both to the same minimal automaton and run linearly.

### 5.3 Diagnostic Pattern Catalog

| Pattern shape                  | Why dangerous                                                  |
|:-------------------------------|:---------------------------------------------------------------|
| `(a+)+`, `(a*)*`               | Nested quantifiers — exponential partitions                    |
| `(a\|a)+`                      | Alternation overlap — duplicate paths                          |
| `(a\|aa)+`                     | Overlapping alternatives that consume same chars               |
| `(a\|ab)+b`                    | Alternation with shared prefix and trailing literal            |
| `(.+)+$`                       | Greedy "match anything" with backtrack on anchor               |
| `(.\|\\s)+`                    | Equivalent character classes in alternation                    |
| `^(\\w+\\s?)*$`                | Word + optional space repeated — each word can match many ways |
| `^([a-zA-Z]+)*$`               | Classic ReDoS in JS validators                                 |
| `^(\\d+)+$`                    | Same shape with digits                                         |

### 5.4 The ReDoS Attack Class

Regular Expression Denial of Service: a malicious actor crafts an input that triggers exponential backtracking, monopolizing a server thread. Real CVEs:

| CVE              | Project / Library            | Pattern issue                              |
|:-----------------|:-----------------------------|:-------------------------------------------|
| CVE-2018-13863   | ms (Node.js library)         | Time-string parser with nested quantifier  |
| CVE-2019-16769   | serialize-javascript         | Backtracking in escape regex               |
| CVE-2021-3807    | ansi-regex                   | ANSI escape sequence parser                |
| CVE-2022-25883   | semver                       | Semver range parsing                       |
| CVE-2017-16114   | marked (Markdown parser)     | Multiple linkable patterns                 |
| CVE-2020-7733    | Cloudflare WAF               | Regex in WAF rule (caused 2019 outage)     |

The infamous Cloudflare 2019 outage was a single misbehaving regex in a WAF rule causing 100% CPU on every edge node simultaneously — the result of a deployed-globally pattern hitting an attacker-style input combination.

### 5.5 Detection Techniques

**Static analysis:**
- Tools: `safe-regex` (npm), `regexploit` (Python), `vuln-regex-detector`, ESLint plugin `eslint-plugin-redos`.
- Heuristic: look for nested quantifiers, alternation with overlap, unanchored repetition.

**Dynamic timing:**
- Time the regex against synthetic inputs of length N, 2N, 4N. If timing grows superlinearly, you have a problem.
- Set a hard timeout — e.g., Go's `regexp.SetMatchTimeout`-style controls (most languages don't ship this).

**Differential testing:**
- Run the same regex through PCRE and RE2. If RE2 takes 50µs and PCRE takes 30s, the pattern is ReDoS-prone.

### 5.6 Prevention Strategies

| Technique                       | Effect                                    | Engines               |
|:--------------------------------|:------------------------------------------|:----------------------|
| Possessive quantifiers `*+ ++`  | Kill backtracking in subexpr              | PCRE, Java, Ruby, .NET, Python 3.11+ |
| Atomic groups `(?>...)`         | Same, applied to a subexpression group    | PCRE, Java, Ruby, .NET, Python 3.11+ |
| Restructure to remove nesting   | `(a+)+` → `a+`                            | Universal             |
| Anchor the regex                | `^...$` reduces match-attempt positions   | Universal             |
| Use a linear-time engine        | RE2, Rust regex, .NET NonBacktracking     | Engine choice         |
| Set a timeout                   | Bound damage when ReDoS hits              | .NET, Java 14+, others |
| Length-limit the input          | Truncate inputs before regex              | Application code      |

---

## 6. Possessive Quantifiers and Atomic Groups

### 6.1 Atomic Groups `(?>...)`

An atomic group is "lock the match — no backtracking allowed back into me." Once the engine matches and exits an atomic group, the saved-state frames inside it are discarded. The group becomes algebraically a function: given the same starting position, it always produces the same outcome (success at length L, or failure).

Compare:

```
(?>a+)b   on  "aaab"   → matches (a+ takes "aaa", then b)
(?>a+)b   on  "aaa"    → fails (a+ takes "aaa", then b fails, no backtrack)
a+b       on  "aaab"   → matches (same)
a+b       on  "aaa"    → fails (same, but attempts every backoff)
```

The behavioral difference shows up in patterns where the inner `+` could give back characters to make the outer match work:

```
(?>a+)a   on  "aaa"    → fails (a+ ate everything, can't give back)
a+a       on  "aaa"    → matches (a+ gives back one char to leave "a" for the trailing a)
```

### 6.2 Possessive Quantifiers `*+ ++ ?+ {n,m}+`

Possessive quantifiers are syntactic sugar for atomic groups around a single quantifier:

```
a++   ≡  (?>a+)
a*+   ≡  (?>a*)
a?+   ≡  (?>a?)
```

They forbid the engine from giving back any matched characters.

### 6.3 Implementation: "Forget the Stack Frame"

In a backtracking engine, when you enter a group, the engine pushes a save-point. On failure later, it pops back to that save-point and tries a different branch. An atomic group simply discards its save-points on successful exit:

```
enter atomic group:
    sp_before = stack.size()
match the inner regex (possibly pushing save-points)
on success:
    stack.truncate(sp_before)   # discard everything pushed inside
on failure:
    backtrack normally past the group
```

The semantic effect: if anything later in the regex fails, the backtracker cannot try alternative inner-matches of the atomic group — it must backtrack past the entire group.

### 6.4 Fixing Catastrophic Backtracking

Original (catastrophic):
```
^(a+)+$
```

Fixed with atomic group:
```
^(?>a+)+$
```

But that's still pointless because `(?>a+)+` is no different from `a+`. The cleaner fix:
```
^a+$
```

A more realistic example — email validation gone wrong:
```
^([a-zA-Z0-9_.+-]+)+@([a-zA-Z0-9-]+\.)+[a-zA-Z0-9-.]+$
```

The `([a-zA-Z0-9_.+-]+)+` is a textbook ReDoS shape. Fix:
```
^[a-zA-Z0-9_.+-]++@(?>[a-zA-Z0-9-]+\.)+[a-zA-Z0-9-.]+$
```

Or restructure to remove the redundancy:
```
^[a-zA-Z0-9_.+-]+@(?:[a-zA-Z0-9-]+\.)+[a-zA-Z0-9-.]+$
```

### 6.5 Engine Support

| Engine             | `(?>...)` | `*+ ++ ?+` | Notes                           |
|:-------------------|:----------|:-----------|:--------------------------------|
| PCRE / PCRE2       | Yes       | Yes        | Foundational support            |
| Perl               | Yes       | Yes        | Inspiration for everyone        |
| Java               | Yes       | Yes        | Since 1.4                       |
| Ruby (Onigmo)      | Yes       | Yes        | Yes                             |
| Python re          | 3.11+     | 3.11+      | Added by Andre Roberge / others |
| Python `regex` pkg | Yes       | Yes        | Drop-in replacement, more feats |
| .NET Regex         | Yes       | Yes        | Both interpreted & NonBacktracking |
| JavaScript V8      | No        | No         | (Discussed, never landed)       |
| Go RE2             | No        | No         | Linear time, doesn't need them  |
| Rust regex         | No        | No         | Same — linear time              |

JavaScript is the major outlier. If you write JS regexes for browser use, restructure patterns or accept the risk.

---

## 7. Lookaround Implementation

### 7.1 The Four Variants

| Syntax     | Name                  | Width        | Consumes input?  |
|:-----------|:----------------------|:-------------|:-----------------|
| `(?=X)`    | Positive lookahead    | Zero         | No               |
| `(?!X)`    | Negative lookahead    | Zero         | No               |
| `(?<=X)`   | Positive lookbehind   | Zero         | No               |
| `(?<!X)`   | Negative lookbehind   | Zero         | No               |

Lookarounds are zero-width assertions: they check that the surrounding text matches X but don't consume any characters. This is sometimes called a "boundary condition."

### 7.2 Lookahead — Easy

Lookahead is "fork and check": at the current position, run the inner regex; if it matches (positive) or fails (negative), continue from the same position. Implementation cost is the cost of running the inner regex once per attempt. If the inner regex itself is dangerous, the lookahead will be too.

```
(?=\d{3})\d+   matches digit sequences of length ≥ 3
```

The engine sees `(?=\d{3})`, runs `\d{3}` from the current position, succeeds or fails, then continues with `\d+`.

### 7.3 Lookbehind — Hard

Lookbehind requires looking at characters **before** the current position. Two implementation strategies:

**Fixed-width lookbehind (classical):** require the inner regex to have a known constant length. Implement by jumping the cursor backward by that length and running the inner regex forward. This is what Java pre-JDK 9, Go's RE2 (it doesn't support lookbehind at all), and historical Python supported.

**Variable-width lookbehind (modern):** allow the inner regex to match any length. Implement by reverse-compiling the inner regex (Thompson construction on a reversed pattern produces an NFA over reversed strings) and matching backward. PCRE, .NET, Perl, JS (since 2018), Java 9+, and Python's `regex` package support this.

### 7.4 Engine Support Matrix

| Engine             | Positive `(?<=)` | Negative `(?<!)` | Variable-width lookbehind |
|:-------------------|:-----------------|:-----------------|:--------------------------|
| PCRE / PCRE2       | Yes              | Yes              | Yes                       |
| Perl               | Yes              | Yes              | Yes                       |
| Java               | Yes              | Yes              | Since JDK 9 (2017)        |
| .NET               | Yes              | Yes              | Yes                       |
| Ruby Onigmo        | Yes              | Yes              | Yes                       |
| Python re (stdlib) | Yes              | Yes              | No (fixed-width only)     |
| Python `regex` pkg | Yes              | Yes              | Yes                       |
| JavaScript         | Yes (2018)       | Yes (2018)       | Yes (V8 native)           |
| Go RE2             | No               | No               | —                         |
| Rust regex         | No               | No               | —                         |

### 7.5 The 2018 JavaScript Lookbehind Story

Lookbehind landed in V8 in 2017 and became part of ES2018. The implementation in Irregexp uses backwards execution: when the engine encounters `(?<=X)`, it runs a separate state machine that scans **right-to-left** from the current position. The inner regex is compiled in reverse order — last to first.

Combined with the `u` flag for full Unicode codepoint scanning and the `d` flag (ES2022) for match indices, modern JS regex is genuinely capable. The trade-off is complexity in the engine: V8's regex implementation is one of its most subtle subsystems.

### 7.6 Why Go Has None

Go's `regexp` package implements the RE2 algorithm. RE2 is a pure NFA/DFA simulator: its execution model is "advance through input, transitioning between states." There's no notion of running another regex from a saved cursor position, no backwards execution, no fork-and-check. Adding lookaround would require either bolting on a second engine for the assertion (defeating the linear-time guarantee) or solving an open research problem.

So Go opts to not support lookaround at all. If you need it, you write the equivalent code yourself, or you reach for a third-party PCRE binding (`go-pcre`).

---

## 8. Backreferences and Why They Break Linearity

### 8.1 The Syntax

```
(\w+) \1
```

Match a word followed by a space followed by the **same word** again. The `\1` is a backreference to the first capture group.

```
(\d+)-\1            same-number sequences like "42-42"
([abc])\1+          one of a/b/c followed by more of the same
^(.+)\1+$           strings made of repeated subunits
```

Numbered backreferences are `\1`, `\2`, etc. Named: `\k<name>` (Java/.NET) or `(?P=name)` (Python).

### 8.2 The Language Is No Longer Regular

Consider: `(.+)\1`. This matches strings of the form `xx` for any non-empty `x`. The set of all such strings is `{ ww : w ∈ Σ⁺ }`. This is the canonical example of a non-regular language. Proof sketch via the pumping lemma: any regular language has a pumping length p such that long strings can be split `xyz` with `|y| > 0`, `|xy| ≤ p`, and all `xyⁿz` in the language. For the doubled-string language, no such p exists.

So a regex with backreferences denotes a language that **cannot** be recognized by any finite automaton, NFA or DFA. The subset construction breaks at the conceptual level — you can't track "is this position the same character as position k?" in a finite-state machine because k can be arbitrarily large.

### 8.3 NP-Completeness

Aho's 1990 paper "Algorithms for Finding Patterns in Strings" proved that membership in a regex-with-backreferences language is NP-complete. The proof reduces 3-SAT to backreference matching: encode variable assignments as backreferences forcing characters to agree.

This means there's no known polynomial algorithm. Backtracking engines handle backreferences naturally because they already have the full match state available — they just compare the current input to the previously captured substring.

### 8.4 Why RE2 Refuses

RE2's design contract is: linear time, bounded memory, untrusted input safe. Backreferences fundamentally violate this. Russ Cox's choice: don't implement them at all. If you need backreferences, RE2 is the wrong tool — use PCRE on trusted input, or use a parser.

### 8.5 Alternatives to Backreferences

If you find yourself reaching for `\1`, consider:

| Goal                                  | Alternative                              |
|:--------------------------------------|:-----------------------------------------|
| Match doubled words                   | Two-pass: regex finds words, code compares |
| Validate paired delimiters            | Use a parser (PEG, parser combinator)    |
| Match palindromes                     | Not possible in regex (context-free)     |
| Detect duplicate letters              | Iterate character-by-character           |
| Configuration "key = key" tautology   | Parse, then check semantically           |

The general rule: as soon as you cross from "find pattern in text" to "validate structural relationships in text," reach for a parser.

---

## 9. Unicode Handling

### 9.1 The "What Does \w Match?" Question

Historically `\w` matched `[A-Za-z0-9_]`. Unicode-aware engines now commonly match any Unicode "word character" — letters in any script, digits, marks, joiners.

| Engine             | `\w` default                  | Unicode opt-in       |
|:-------------------|:------------------------------|:---------------------|
| PCRE2              | ASCII (`[A-Za-z0-9_]`)        | `(*UCP)` or PCRE2_UCP flag |
| Java               | Unicode if UNICODE_CHARACTER_CLASS | `Pattern.UNICODE_CHARACTER_CLASS` |
| Python re          | Unicode by default in 3.x     | `re.A` to force ASCII |
| JavaScript         | ASCII                         | `u` flag enables Unicode |
| .NET               | Unicode                       | `RegexOptions.ECMAScript` for ASCII |
| Go RE2             | ASCII (`[A-Za-z0-9_]`)        | `(?U)` for UCP — actually `\p{L}` etc. |
| Rust regex         | ASCII                         | `(?u)` to enable Unicode |
| Perl               | Unicode if `use utf8`         | `/a` for ASCII       |

### 9.2 Unicode Property Classes

The Unicode property catalog `\p{...}` and its negation `\P{...}` give precise control:

| Property         | Meaning                                    |
|:-----------------|:-------------------------------------------|
| `\p{L}`          | Letter (any case, any script)              |
| `\p{Lu}`         | Uppercase letter                           |
| `\p{Ll}`         | Lowercase letter                           |
| `\p{Lt}`         | Titlecase letter                           |
| `\p{N}`          | Number (any kind)                          |
| `\p{Nd}`         | Decimal number                             |
| `\p{P}`          | Punctuation                                |
| `\p{Pc}`         | Connector punctuation (e.g., `_`)          |
| `\p{S}`          | Symbol                                     |
| `\p{Z}`          | Separator (whitespace-ish)                 |
| `\p{Zs}`         | Space separator                            |
| `\p{Cc}`         | Control character                          |
| `\p{Script=Latin}` | Latin script                             |
| `\p{Script=Han}` | Chinese characters                         |
| `\p{Script=Cyrillic}` | Cyrillic                              |
| `\p{Block=...}`  | Unicode block (e.g., `Arrows`, `Emoticons`) |

These come from the Unicode Character Database (UCD) and are updated annually. Engine-specific support varies — PCRE2 has the most complete catalog, Go has fewer scripts/blocks, JavaScript requires the `u` flag.

### 9.3 UTF-8 Byte-Level vs Codepoint-Level

Two semantic models:

**Byte-level (default in many engines):**
- `.` matches one byte. `[ -ÿ]` is a byte range.
- Non-ASCII codepoints become 2-4 bytes; matching them requires multi-byte literals.
- Faster, but breaks on surrogate pairs and combining characters.

**Codepoint-level (with Unicode flag):**
- `.` matches one codepoint. `\u{1F600}` matches U+1F600 directly.
- Engine decodes UTF-8 (or UTF-16 in Java/JS) on the fly.
- Slightly slower but semantically correct.

JavaScript's `u` flag promotes codepoint-level scanning. Without it, `/^.$/` doesn't match `"😀"` because the emoji is a UTF-16 surrogate pair (two `😀` 16-bit units), and `.` matches one unit.

### 9.4 Case-Folding Under Unicode

Case-insensitive matching `(?i)` is straightforward in ASCII (just OR `0x20`). Unicode is harder:

- `İ` (Turkish capital I with dot, U+0130) lowercases to `i` plus combining dot (`i̇`) — a one-to-two mapping.
- `ß` (German sharp s) uppercases to `SS` — also one-to-two.
- `ς` and `σ` (Greek final and non-final lowercase sigma) are both lowercase forms of `Σ` — context-dependent.

Most engines use **simple case-folding** (one-to-one) and ignore the special cases. Some (ICU-based) do **full case-folding**. The specification is in Unicode Standard Annex #29.

### 9.5 Grapheme Clusters

A "user-perceived character" might be multiple codepoints — e.g., `é` can be U+00E9 (single code point) or `e` + U+0301 (combining acute). An emoji like 👨‍👩‍👧 is several emoji glued by ZWJs.

Some engines provide `\X` to match a grapheme cluster (extended grapheme cluster per UAX #29):

| Engine      | `\X` support     |
|:------------|:-----------------|
| PCRE2       | Yes              |
| Perl        | Yes              |
| Ruby Onigmo | Yes              |
| Java        | Pattern.compile with UNICODE_CHARACTER_CLASS — but no \X; use `\b{g}` boundaries |
| .NET        | No `\X`, but StringInfo.GetTextElementEnumerator |
| Python re   | No (use `regex` package) |
| Go RE2      | No               |
| Rust regex  | No (use `unicode-segmentation` crate) |

### 9.6 Normalization

Unicode allows multiple representations of the same logical text. To compare correctly, normalize first:

| Form  | Description                                      |
|:------|:-------------------------------------------------|
| NFC   | Composed (single codepoint preferred)            |
| NFD   | Decomposed (base + combining)                    |
| NFKC  | Compatibility composed                           |
| NFKD  | Compatibility decomposed                         |

Best practice: normalize input to NFC before regex matching. Most regex engines do not normalize automatically.

### 9.7 Unicode Technical Standard #18

UTS #18 specifies three levels of regex Unicode support:

- **Level 1**: Basic Unicode support — codepoint ranges, properties, simple case folding.
- **Level 2**: Extended — full case folding, grapheme clusters, extended word boundaries.
- **Level 3**: Tailored — locale-specific collation, line breaking.

Most modern engines reach Level 1. PCRE2 and ICU-based engines reach Level 2. Level 3 is essentially CLDR territory and beyond regex.

---

## 10. Engine-Specific Quirks

### 10.1 JavaScript

```javascript
// Literal regex
const re = /\b(\w+)\b/gi;
const re2 = new RegExp('\\b(\\w+)\\b', 'gi');

// Flags: g, i, m, s, u, y, d
// g — global (find all)
// i — case-insensitive
// m — multiline (^$ match line bounds)
// s — dotAll (. matches \n)
// u — Unicode (codepoint scanning)
// y — sticky (anchor to lastIndex)
// d — indices (provide d.indices on match)

// u flag enables \u{HHHHH} escapes for codepoints > 0xFFFF
const emoji = /\u{1F600}/u;        // matches "😀"
const noU = /\u{1F600}/;            // SyntaxError (without u, only \uHHHH)

// Sticky y flag for tokenizer-style matching
const tok = /\w+/y;
tok.lastIndex = 5;
tok.exec(string);                   // anchored at position 5 only

// d flag for capture indices (ES2022)
const m = /(\w+) (\w+)/d.exec("Hello World");
m.indices[1];                       // [0, 5]  — start/end of group 1
m.indices[2];                       // [6, 11] — start/end of group 2
```

Quirks: literal-syntax regexes are pre-parsed at script parse time; `new RegExp(...)` constructs at runtime. `RegExp.prototype.test` advances `lastIndex` only with the `g` or `y` flag. Lookbehind shipped in 2018 but is unsupported in old browsers (ES2018+ required).

### 10.2 Python `re`

```python
import re

# Compile once, reuse
pat = re.compile(r"(\d{4})-(\d{2})-(\d{2})", re.MULTILINE)
pat.match(s)        # anchored at start of string
pat.search(s)       # find anywhere
pat.fullmatch(s)    # require entire string match (3.4+)
pat.findall(s)      # all non-overlapping matches
pat.finditer(s)     # iterator of Match objects
pat.sub(repl, s)    # replace
pat.split(s)        # split by matches

# Flags: I, M, S, X, A, U
re.I  # IGNORECASE
re.M  # MULTILINE
re.S  # DOTALL — . matches \n
re.X  # VERBOSE — whitespace and #-comments ignored
re.A  # ASCII — \w \W \b etc. ASCII-only
re.U  # UNICODE (default in Python 3)

# Named groups (?P<name>...) — note Python's P prefix
m = re.match(r"(?P<year>\d{4})-(?P<month>\d{2})", "2024-12")
m.group('year')     # "2024"
m['month']           # "12"  (subscript syntax, 3.6+)

# Raw strings essential — \b means backspace in normal strings
r"\b\w+\b"           # word boundary
"\b\w+\b"            # backspace-w-plus-backspace — broken
```

The third-party `regex` package is a nearly-drop-in replacement with significant extras: variable-width lookbehind, fuzzy matching `(?:pattern){e<=2}` (allow up to 2 errors), splitter improvements, possessive quantifiers, atomic groups (these landed in stdlib `re` only in 3.11). Worth installing for any non-trivial regex work in Python.

### 10.3 Go `regexp`

```go
import "regexp"

// Compile (panic on bad pattern)
re := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)

// Compile + check
re, err := regexp.Compile(`(\d{4})-(\d{2})-(\d{2})`)

// Methods follow a consistent naming scheme:
// Find, FindAll, FindIndex, FindAllIndex
// FindString, FindStringIndex, FindStringSubmatch
// FindStringSubmatchIndex, ReplaceAllString
// ReplaceAllStringFunc, Split, MatchString

re.MatchString("2024-12-25")              // true/false
re.FindString("...2024-12-25...")         // "2024-12-25"
re.FindStringSubmatch("...2024-12-25...") // ["2024-12-25" "2024" "12" "25"]
re.ReplaceAllString(s, "$1/$2/$3")        // backref by group number
re.ReplaceAllString(s, "${year}/${month}/${day}")

// Named groups
re := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
match := re.FindStringSubmatch("2024-12-25")
result := make(map[string]string)
for i, name := range re.SubexpNames() {
    if i != 0 && name != "" {
        result[name] = match[i]
    }
}
```

Go's regexp follows the RE2 syntax document (`pkg.go.dev/regexp/syntax`). No backreferences, no lookaround, no recursion. Linear time guaranteed. For regex-heavy workloads, Go's stdlib is one of the fastest options in the language ecosystem.

### 10.4 Rust `regex`

```rust
use regex::Regex;

let re = Regex::new(r"(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})").unwrap();

if let Some(caps) = re.captures("2024-12-25") {
    println!("{}", &caps["year"]);
    println!("{}", &caps[1]);
}

// Iterate all matches
for m in re.find_iter(text) {
    println!("{}: {}", m.start(), m.as_str());
}

// RegexSet for multi-pattern matching
use regex::RegexSet;
let set = RegexSet::new(&[
    r"\bfoo\b",
    r"\bbar\b",
    r"\bbaz\b",
]).unwrap();
let matches: Vec<_> = set.matches(text).into_iter().collect();
```

The Rust `regex` crate's architecture deserves study. It selects between four matchers per pattern:

| Matcher          | When used                                        |
|:-----------------|:-------------------------------------------------|
| Pike VM          | Regexes that need full features (capture, etc.)  |
| Bounded backtracker | Small input, small NFA                        |
| Lazy DFA         | Long inputs, common case                         |
| One-pass NFA     | Regexes with no ambiguity                        |

It also has a `regex-lite` companion crate for smaller binaries (no SIMD, no Unicode tables, smaller feature set). For full performance you want `regex` with default features.

### 10.5 Java `java.util.regex`

```java
import java.util.regex.*;

Pattern p = Pattern.compile("(\\d{4})-(\\d{2})-(\\d{2})", Pattern.CASE_INSENSITIVE);
Matcher m = p.matcher("2024-12-25");
if (m.matches()) {                  // anchored, full string
    String year = m.group(1);
}
if (m.find()) {                     // find anywhere
    String year = m.group(1);
}

// Replace
String out = m.replaceAll("$1/$2/$3");

// Named groups (Java 7+)
Pattern p = Pattern.compile("(?<year>\\d{4})-(?<month>\\d{2})");
Matcher m = p.matcher(s);
if (m.find()) m.group("year");

// Flags as bit-OR'd ints
Pattern.CASE_INSENSITIVE  // (?i)
Pattern.MULTILINE         // (?m)
Pattern.DOTALL            // (?s)
Pattern.UNICODE_CASE      // (?u)
Pattern.UNICODE_CHARACTER_CLASS  // (?U) — full Unicode \w etc.
Pattern.COMMENTS          // (?x) — verbose mode
Pattern.UNIX_LINES        // (?d)
```

`Matcher` is stateful — it tracks current position, last match, etc. This is by design but confuses people who expect functional purity.

Java's regex is backtracking, with all the usual ReDoS risks. Java 14+ added `Pattern.CANON_EQ` for canonical-equivalence matching. JDK 9 added variable-width lookbehind.

### 10.6 PCRE2 — The Kitchen Sink

PCRE2 (the C library, not the Perl-Compatible Regular Expressions header file) is the most feature-complete library. The full feature surface:

```
(?>...)             atomic group
(?:...)             non-capturing group
(?<name>...)        named capture
(?#comment)         inline comment
(?(condition)yes|no) conditional
(?(R)...)           recursion check
(?R)                full pattern recursion
(?&name)            subroutine call
(?(DEFINE)...)      define-only group (no match)
\K                  reset match start
(*PRUNE)            backtrack control: fail at this point
(*FAIL)             backtrack control: force failure
(*COMMIT)           backtrack control: no further match attempts
(*ACCEPT)           backtrack control: succeed immediately
(*MARK:name)        named mark for diagnostics
(*UTF)              UTF mode (UTF-8 by default in PCRE2)
(*UCP)              full Unicode character properties
(*CR)(*LF)(*CRLF)   newline conventions
(*LIMIT_MATCH=n)    runtime resource limits
(*LIMIT_RECURSION=n)
```

Backtracking control verbs are powerful and dangerous. `(*PRUNE)` says "if the current match fails, don't backtrack past this point." `(*COMMIT)` is stronger: "if the current match fails, give up entirely." Use them to fix specific catastrophic backtracking spots without restructuring the whole pattern.

```
^(?:(?>[a-z]+)(*COMMIT)\d+)*$
```

The `(*COMMIT)` ensures that once we commit to consuming letters at a given position, we won't backtrack and try a different split.

### 10.7 .NET `Regex` — and the NonBacktracking Mode

.NET's regex has three execution modes:

```csharp
using System.Text.RegularExpressions;

// 1. Interpreted — default
var r1 = new Regex(@"(\d{4})-(\d{2})");

// 2. Compiled — emit IL via Reflection.Emit, then JIT
var r2 = new Regex(@"(\d{4})-(\d{2})", RegexOptions.Compiled);

// 3. NonBacktracking (since .NET 7)
var r3 = new Regex(@"(\d{4})-(\d{2})", RegexOptions.NonBacktracking);
```

The NonBacktracking mode uses **Brzozowski derivatives** — an algebraic approach where the derivative `D_a(R)` of a regex `R` with respect to character `a` is itself a regex describing what `R` matches after consuming an `a`. Iteratively computing derivatives gives a DFA on the fly.

The advantage: you can support a subset of features (anchors, character classes, alternation, repetition, capture, backreferences in some cases) within a derivative-based DFA without losing linear-time guarantees. It's the only mainstream linear-time engine that supports captures cleanly — a notable engineering achievement.

Trade-off: NonBacktracking does not currently support backreferences or unrestricted lookaround. So you choose: features and ReDoS risk (default), or safety with reduced features (NonBacktracking).

### 10.8 Ruby Onigmo

Ruby uses Onigmo (fork of Oniguruma). It's a feature-rich backtracking engine roughly equivalent to PCRE in capability. Ruby integrates regex into the language at the syntax level:

```ruby
# Literal regex
re = /(\d{4})-(\d{2})/
"2024-12" =~ re
$1   # "2024"
$2   # "12"

# Match operator
if (m = re.match("2024-12-25"))
    m[1]
    m[:year]  # if named (?<year>\d{4})
end

# String#scan
"foo bar baz".scan(/\w+/)   # ["foo", "bar", "baz"]
```

---

## 11. Performance Characteristics

### 11.1 The Cardinal Rules

1. **Anchor when possible.** `^...` reduces match attempts from O(N) starting positions to 1.
2. **Prefer specific character classes over `.`.** `\d+` is far more optimizable than `.+` because the engine can fail fast on non-digits.
3. **Avoid alternation when a character class works.** `(a|b|c)` → `[abc]`. Character classes compile to bitmap lookups; alternation becomes branches.
4. **Factor common prefixes out of alternation.** `(abc|abd|abe)` → `ab(c|d|e)` → `ab[cde]`.
5. **Avoid catastrophic patterns.** No nested unanchored quantifiers.
6. **Compile once, reuse.** Regex compilation is non-trivial (parsing, NFA construction, optimization). Cache compiled regexes outside hot loops.
7. **Use literal search for literal strings.** `grep -F` (fixed) is much faster than `grep -E '^foo$'`.

### 11.2 The Literal-String-as-Regex Tax

If you write `grep "needle" file`, GNU grep checks "is this a literal pattern?" If yes, it runs Boyer-Moore. If no, it constructs an NFA. The check is simple — the moment any metacharacter appears, you're paying NFA-simulation costs.

| Search method                      | Throughput (typical)       |
|:-----------------------------------|:---------------------------|
| `grep -F` (fixed-string, B-M)      | 1-10 GB/s                  |
| `grep` (BRE, mostly literal)       | 0.5-1 GB/s                 |
| `grep -E` (ERE, simple regex)      | 0.2-1 GB/s                 |
| `grep -P` (PCRE)                   | 50-500 MB/s                |
| `ripgrep` (Rust regex)             | 1-10 GB/s (SIMD)           |

`ripgrep` is dramatically faster than GNU grep for many workloads because Rust's `regex` crate uses SIMD (`memchr`) for literal scanning, has better DFA scheduling, and skips the locale processing that bloats GNU grep.

### 11.3 Compile Costs

```python
# Bad — compiles on every iteration
for line in lines:
    if re.search(r"^ERROR:", line):
        ...

# Good — compile once
err_re = re.compile(r"^ERROR:")
for line in lines:
    if err_re.search(line):
        ...

# Python re does cache compiled patterns (re._cache), but cache is small
# and not contractually guaranteed. Explicit compile is always safer.
```

```go
// Same in Go
errRe := regexp.MustCompile(`^ERROR:`)
for scanner.Scan() {
    if errRe.MatchString(scanner.Text()) { ... }
}
```

### 11.4 Multi-Pattern Optimization

If you have many patterns to match, don't run each separately. Use:

- **Aho-Corasick** for many literal needles. Builds a trie with failure links, runs in O(N + sum of needles) per text.
- **Hyperscan** for many regex patterns simultaneously.
- **Rust regex `RegexSet`** for moderate numbers of regexes.

Building a single combined regex with alternation `(re1|re2|re3|...)` works for small numbers of patterns but degrades quickly past a few hundred alternatives.

### 11.5 Profiling Regex Performance

When a regex is slow, hypothesize and test:

1. Instrument with timing: log regex execution time and input length, plot.
2. If timing is superlinear in length, suspect catastrophic backtracking.
3. Run the regex through a static-analysis tool (`safe-regex`, `regexploit`).
4. Test on RE2 vs PCRE — divergence indicates the pattern is exploiting backtracking.
5. Look at the regex compile cost separately from match cost.

---

## 12. POSIX BRE vs ERE vs PCRE

### 12.1 The Three Standards

POSIX defines two regex flavors, plus PCRE is a third widely-implemented standard.

| Feature                | BRE (Basic)          | ERE (Extended)       | PCRE                  |
|:-----------------------|:---------------------|:---------------------|:----------------------|
| `()` for grouping      | `\(...\)`            | `(...)`              | `(...)`               |
| `{m,n}` repetition     | `\{m,n\}`            | `{m,n}`              | `{m,n}`               |
| `+` (one or more)      | Not supported (BRE)  | `+`                  | `+`                   |
| `?` (zero or one)      | Not supported (BRE)  | `?`                  | `?`                   |
| `|` (alternation)      | Not supported        | `|`                  | `|`                   |
| `\1` backreferences    | Yes                  | No (POSIX standard)  | Yes                   |
| Lookaround             | No                   | No                   | Yes                   |
| Non-greedy             | No                   | No                   | Yes (`*?`, `+?`)      |
| Character classes      | `[...]`              | `[...]`              | `[...]` + `\d \w` etc. |
| Anchors                | `^` `$`              | `^` `$`              | `^` `$` `\b` `\A` etc. |

### 12.2 Tools and Their Defaults

| Tool         | Default flavor   | Other modes                          |
|:-------------|:-----------------|:-------------------------------------|
| `grep`       | BRE              | `-E` (ERE), `-P` (PCRE), `-F` (fixed) |
| `egrep`      | ERE (alias for `grep -E`) |                            |
| `fgrep`      | Fixed (alias for `grep -F`) |                          |
| `sed`        | BRE              | `-E` or `-r` (ERE)                   |
| `awk`        | ERE              | (no BRE)                             |
| `vim`        | Vim regex (own)  | `\v`, `\V`, `\m`, `\M` modes         |
| `pcregrep`   | PCRE             |                                      |
| `ripgrep` (`rg`) | Rust regex   | `-P` for PCRE2 fallback              |

### 12.3 Practical Differences

```bash
# Capturing groups — BRE needs escaping
grep    '\(foo\)bar'    file       # BRE: () are literal unless escaped
grep -E '(foo)bar'      file       # ERE: () are metacharacters
grep -P '(foo)bar'      file       # PCRE: same as ERE plus more features

# Repetition operators
grep    'foo\{2,4\}'    file       # BRE: \{ \}
grep -E 'foo{2,4}'      file       # ERE: { }

# + and ? don't exist in BRE
grep    'foo+'          file       # BRE: matches "foo+" literally!
grep -E 'foo+'          file       # ERE: matches "foo", "foo", etc.

# Backreferences
grep    '\(foo\)\1'     file       # BRE: yes
grep -E '(foo)\1'       file       # ERE: undefined behavior (impl-specific)
                                   # GNU grep -E supports it; POSIX says no
```

`sed -E` (or BSD `sed -r`) switches to ERE; the unflagged `sed` uses BRE. This is the source of countless portability headaches between GNU and BSD systems.

### 12.4 PCRE-Specific Features

```
\d \D \w \W \s \S        digit / word / whitespace classes
\b \B \A \Z \z           word boundary, string anchors
*? +? ?? {n,m}?          non-greedy quantifiers
(?:...)                  non-capturing group
(?=...) (?!...)          lookahead
(?<=...) (?<!...)        lookbehind
(?<name>...)             named capture
\K                       reset match start
(?>...)                  atomic group
(*PRUNE) (*COMMIT) etc.  backtrack control
```

If you need any of these, you need PCRE or a similar engine — POSIX won't deliver.

---

## 13. Vim Regex

### 13.1 The Magic Modes

Vim has four magic levels controlling escaping:

| Mode             | Prefix | Special chars          | Effect                          |
|:-----------------|:-------|:-----------------------|:--------------------------------|
| Very magic       | `\v`   | All punctuation special | Closest to PCRE/ERE           |
| Magic (default)  | `\m`   | Most are special        | Vim default                    |
| Nomagic          | `\M`   | Few are special         | Just `*` and `^` `$`           |
| Very nomagic     | `\V`   | Only `\` is special     | Almost all literal             |

```vim
" Default 'magic' mode — () must be \(...\) for grouping
:%s/\(\d\+\)-\(\d\+\)/\2-\1/g

" Very-magic — escaping not needed
:%s/\v(\d+)-(\d+)/\2-\1/g

" Very-nomagic — almost everything literal
:/\Va.b.c            " Find literal "a.b.c"
```

The `\v` mode is what most people want because it matches PCRE intuition. Habitually prefix patterns with `\v`.

### 13.2 Vim-Specific Features

```vim
" \zs and \ze — match start/end markers (subset of \K behavior)
:/foo\zsbar         " Match "foo" but only highlight starting from "bar"
:/foo\zebar         " Match "foo" only when followed by "bar", up to before bar

" \= in replacement — Vim expression evaluation
:%s/\d\+/\=submatch(0)*2/g          " Double every number
:%s/\v(\d+)/\=printf("%04d", str2nr(submatch(1)))/g

" Word boundaries
\<      word start
\>      word end

" Position assertions
\%^     beginning of file (super-anchor)
\%$     end of file
\%V     visual area
\%#     cursor position

" Named character classes
\a \A   alpha / non-alpha
\d \D   digit / non-digit
\w \W   word char / non-word
\s \S   whitespace / non-whitespace
\l \L   lowercase / non-lowercase
\u \U   uppercase / non-uppercase
\x \X   hex / non-hex
\o \O   octal / non-octal
\p \P   printable / non-printable
```

### 13.3 Substitution Idioms

```vim
" Case-preserving replace
:%s/\v(<\w)|(\w)/\u\1\l\2/g          " Title-case every word

" Multi-line — Vim defaults to single-line; use \n in pattern
:%s/foo\nbar/baz/g                    " Match "foo<newline>bar"

" Confirm each replacement
:%s/foo/bar/gc

" Use any delimiter to avoid escape hell
:%s#http://[^/]\+#https://example.com#g

" Smart-case (matches Vim's smartcase setting interactively)
:set smartcase
" Then /foo matches case-insensitively, /Foo matches case-sensitively
```

---

## 14. RE2 Algorithm

### 14.1 Russ Cox's Articles

In 2007 Russ Cox published a series of articles at swtch.com/~rsc/regexp/ that resurrected interest in NFA-based matching:

1. **"Regular Expression Matching Can Be Simple And Fast (but is slow in Java, Perl, PHP, Python, Ruby, ...)"** — the famous comparison showing exponential-time PCRE-style engines on `(a?){n}a^n` vs linear-time Thompson simulation.
2. **"Regular Expression Matching: the Virtual Machine Approach"** — explains compiling regexes to a small bytecode VM with three execution strategies (recursive, backtracking, Thompson's simulation, Pike's NFA).
3. **"Regular Expression Matching in the Wild"** — describes RE2's algorithm: lazy DFA built from NFA, with NFA simulation as fallback.
4. **"Regular Expression Matching with a Trigram Index"** — describes Google Code Search's substring index for narrowing regex search across petabytes of code.

These articles, plus Pike's earlier work (sam editor, structural regex), are the canonical resources.

### 14.2 The Algorithm

RE2's architecture:

```
Regex source
    ↓
  Parser (regexp/syntax/parse.go)
    ↓
  Simplification (constant folding, etc.)
    ↓
  Prog — bytecode for NFA
    ↓
  Selector chooses one of:
        OnePass  (linear, no ambiguity)
        BitState (small input, small NFA: bounded backtracking with bit-set memoization)
        Lazy DFA (long input, common case)
        NFA      (full Pike VM fallback)
    ↓
  Match
```

The lazy DFA's structure:

```
type DFA struct {
    states     map[NFAStateSet]*State   // set-of-NFA-states → DFA state
    cacheBudget int64                   // max bytes
    ...
}
```

When matching reaches an unknown DFA state, compute the ε-closure of the NFA states under that input character, hash the resulting set, look up or insert. If the cache exceeds budget, evict (RE2 typically wipes and rebuilds).

### 14.3 Why It's Linear

The DFA at any input character does a single state transition: O(1). The construction of that state happens at most once per state (cached). The number of states is bounded by 2^|NFA|, but in practice tiny. So matching an input of length N takes O(N) time once the DFA is warm, with O(N + state_construction) total.

The pessimistic worst case (cache thrashing) falls back to Pike VM simulation: O(N × M), still polynomial.

### 14.4 Limitations as Design Choices

RE2's "no backreferences, no lookaround" limitations are deliberate:
- Backreferences make the language non-regular; no NFA-equivalent exists.
- Lookaround can be implemented but adds enormous complexity for niche use; Cox punts.
- Variable-width captures within DFA states require NFA-level fallback.

The trade-off: for **untrusted input** (web servers, indexers, code search) RE2's safety wins. For **trusted patterns** with rich features (Perl scripts, log scrapers), PCRE wins.

### 14.5 Where RE2 Is Used

- Google Code Search (until shutdown), now Sourcegraph derivatives.
- Google Sheets `REGEXMATCH`, `REGEXEXTRACT`.
- Cloud Logging filter expressions.
- Go's standard library `regexp` package — direct port.
- gRPC generated code path matching.
- Many WAF / IDS rule engines that scan untrusted traffic.

---

## 15. Hyperscan

### 15.1 The Multi-Pattern Problem

Snort and Suricata IDS engines need to scan packets against thousands of signatures. Running each signature as a separate regex is too slow. Hyperscan's premise: compile all the regexes into a single combined matcher that finds **any** matching signature in a single pass.

### 15.2 Architecture

Hyperscan compiles regexes into one of several internal engines:

| Engine     | Use case                              |
|:-----------|:--------------------------------------|
| Literal matching (FDR / Teddy) | Many literal needles, SIMD-accelerated |
| NFA        | Small regexes with bounded states     |
| DFA        | Larger combined automata              |
| Bounded repeats | `\d{1,1024}` style patterns       |
| Big DFA    | Truly large combined patterns         |

The compiler picks per-pattern, then stitches them into a single scanning pipeline.

### 15.3 SIMD

Hyperscan exploits Intel's vector instructions (SSE, AVX, AVX2, AVX-512) for parallel byte comparisons. Instead of "compare one input byte to one DFA transition," it compares 16/32/64 bytes simultaneously:

```c
// Pseudocode for SIMD-accelerated byte set matching
__m256i input = _mm256_loadu_si256(text + pos);
__m256i mask  = _mm256_cmpestrm(target_chars, input);
uint32_t match_bits = _mm256_movemask_epi8(mask);
if (match_bits) {
    // bit position tells which character offset matched
    process_match(pos + __builtin_ctz(match_bits));
}
```

### 15.4 Use Cases

- **Suricata IDS** — rule scanning at line-rate (10-100 Gbps). Without Hyperscan, Suricata couldn't keep up.
- **Cloudflare** — uses Hyperscan in WAF for rule pre-filtering before falling back to per-pattern engines.
- **Trustwave / Imperva** — content filtering.
- **Logging/SIEM** — searching streams of log data for signatures.

### 15.5 Limitations

- Limited regex feature set (no backrefs, no lookaround, restricted captures).
- API is C, requires careful integration.
- Compilation is expensive (seconds to minutes for thousands of patterns); designed for "compile once, scan forever" workloads.
- Originally x86-only; ARM port (vectorscan) is community-maintained.

---

## 16. Anti-Patterns and Better Tools

### 16.1 Don't Parse HTML/XML with Regex

The famous Stack Overflow answer (Bobince, 2009) explains why. HTML is not regular — it's not even reliably context-free in practice (entities, comments, CDATA, scripts, broken markup). Regex matches simple cases, then fails subtly on:

```html
<a title="<>" href="...">
<a href = "..." class='foo'>
<a><a></a></a>          <!-- nested -->
<!-- <a href="..."> -->  <!-- in comment -->
<![CDATA[<a>]]>          <!-- in CDATA -->
```

Use a real parser: BeautifulSoup, lxml, JSoup, html.parser, libxml2.

### 16.2 Don't Parse JSON with Regex

JSON has nesting, escape sequences, Unicode, numbers with optional exponents. Regex can't reliably handle the recursion. Use:

- `jq` for command-line JSON queries.
- Language-native JSON parsers (every language has one).
- `gron` for converting JSON to grep-friendly flat form.

### 16.3 Don't Parse CSV with Regex

The "CSV problem" looks simple — split on `,` — until you encounter:

```csv
"Smith, John",42,"He said ""hi"""
```

Quoted commas, escaped quotes, multi-line fields, varying delimiters. Use a real CSV library (Python `csv`, Go `encoding/csv`, Rust `csv` crate).

### 16.4 Don't Parse Email Addresses with Regex

The official RFC 5322 email regex is over 6000 characters. The "good enough" `\S+@\S+` accepts garbage; tighter patterns reject valid addresses (e.g., quoted local parts, internationalized domain names). Use a library; better, just send a verification email.

### 16.5 Don't Parse URLs / IP Addresses / Phone Numbers / ... with Regex

Same pattern: the format is too complex, the edge cases too many. Use language-standard parsing libraries:

| Format         | Better tool                                  |
|:---------------|:---------------------------------------------|
| URLs           | `urllib.parse`, `net/url`, `URL` (WHATWG)    |
| IP addresses   | `ipaddress`, `net.ParseIP`, `IPAddress` (.NET) |
| Phone numbers  | Google's libphonenumber                      |
| Dates / times  | `dateutil`, `time.Parse`, ISO 8601 libraries |
| Markdown       | A Markdown parser (commonmark, marked)       |
| Source code    | A real parser / parser-combinator / PEG      |

### 16.6 When Regex IS the Right Tool

Genuinely good fits for regex:
- Single-line text validation: postal codes, simple identifiers, formatted numbers.
- Log line scraping where the format is fixed and simple.
- Search-and-replace with simple structural matching.
- Tokenization for programming languages — but only the lexer; the parser is separate.
- Quick exploratory data work (`grep | sed | awk` pipelines).

### 16.7 Parser Alternatives

When regex isn't enough:

| Approach           | Tools / Libraries                              |
|:-------------------|:-----------------------------------------------|
| Parser combinators | Haskell `parsec`, Python `parsy`, Rust `nom`, Go `goyacc`, JS `parsimmon` |
| PEG                | Go `pigeon`, Python `pyparsing`, Rust `pest`, JS `peg.js` |
| LALR / LR          | `yacc`, `bison`, `antlr`                       |
| Recursive descent  | Hand-written, often clearer for small grammars |
| Treesitter         | Incremental parser framework (used by Neovim, GitHub) |

---

## 17. Idioms at the Internals Depth

### 17.1 The Universal Field-Extractor

```python
import re

LOG_RE = re.compile(r"""
    ^
    (?P<ts>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?)
    \s+
    (?P<level>DEBUG|INFO|WARN|ERROR|FATAL)
    \s+
    \[(?P<thread>[^\]]+)\]
    \s+
    (?P<logger>\S+)
    \s+-\s+
    (?P<msg>.*)
    $
""", re.VERBOSE)

m = LOG_RE.match(line)
if m:
    record = m.groupdict()
```

Key tricks:
- `re.VERBOSE` allows whitespace and comments inside the pattern.
- Named groups make extraction self-documenting.
- Anchors `^...$` ensure the whole line conforms.
- Optional fragments `(?:\.\d+)?` handle format variations.

### 17.2 The Validate-Format Pattern

```python
ipv4_octet = r"(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)"
ipv4_re = re.compile(rf"^{ipv4_octet}(?:\.{ipv4_octet}){{3}}$")
```

Build complex patterns by string-concatenating named pieces. Each piece is independently testable. Use raw `f`-strings (or template strings in your language) to splice in subpatterns.

### 17.3 The Replace-with-Callback Pattern

```python
# Python
def slugify(m):
    return m.group(0).lower().replace(" ", "-")

re.sub(r"[A-Z][A-Za-z\s]+", slugify, text)
```

```javascript
// JavaScript
text.replace(/(\w+)\s+(\w+)/g, (match, first, last) => `${last}, ${first}`);
```

```go
// Go
re.ReplaceAllStringFunc(text, func(s string) string {
    return strings.ToUpper(s)
})
```

When the replacement depends on the match, use a function form. It's more maintainable than embedded backreferences.

### 17.4 The "Find Balanced" Workaround

Real regex (without recursion) cannot find balanced parentheses. PCRE2 has recursive subroutines:

```
\((?:[^()]|(?R))*\)        full PCRE2 — recursive match
```

Without recursion, fake it for limited depth:

```
\((?:[^()]|\((?:[^()]|\([^()]*\))*\))*\)    nested up to depth 3
```

Or just use a parser.

### 17.5 Aggressive Anchoring

Every regex used for **validation** should be anchored end-to-end:

```python
# Bad — matches "abc123" inside "junkabc123junk"
re.search(r"[a-z]+\d+", "junkabc123junk")    # Match found!

# Good — anchored
re.fullmatch(r"[a-z]+\d+", "junkabc123junk")  # None
re.fullmatch(r"[a-z]+\d+", "abc123")          # Match
```

For Java: `Matcher.matches()` requires full match; `Matcher.find()` does not. Use the right method.

### 17.6 The Regex-DSL Trick

Build complex regexes from named pieces; never write giant unfactored patterns:

```python
WHITESPACE   = r"[ \t]"
NEWLINE      = r"\r?\n"
WORD         = r"\w+"
QUOTED       = r'"(?:[^"\\]|\\.)*"'
COMMENT      = r"#[^\n]*"

DEFINITION = rf"""
    ^
    (?P<key>{WORD})
    {WHITESPACE}*=
    {WHITESPACE}*
    (?P<value>{QUOTED}|{WORD})
    {WHITESPACE}*
    (?:{COMMENT})?
    $
"""

definition_re = re.compile(DEFINITION, re.VERBOSE | re.MULTILINE)
```

Each named piece is independently testable. The full regex reads like a grammar.

### 17.7 The "Compile Hot Patterns at Module Load" Idiom

```python
# At module top level — compiled once when module is imported
_DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
_EMAIL_RE = re.compile(r"^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$")
_UUID_RE  = re.compile(r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")

def is_date(s):  return bool(_DATE_RE.match(s))
def is_email(s): return bool(_EMAIL_RE.match(s))
def is_uuid(s):  return bool(_UUID_RE.match(s))
```

```go
// Go — package-level vars compiled at init time
var (
    dateRE  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
    emailRE = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
    uuidRE  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)
```

### 17.8 The "Test Your Regex Like Code" Idiom

```python
import unittest, re

class TestEmailRegex(unittest.TestCase):
    EMAIL_RE = re.compile(r"^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$")

    def test_valid(self):
        for addr in ["a@b.co", "user.name+tag@example.com", "x@y.museum"]:
            self.assertIsNotNone(self.EMAIL_RE.match(addr), addr)

    def test_invalid(self):
        for addr in ["", "@b.co", "a@", "a@b", "a b@c.com"]:
            self.assertIsNone(self.EMAIL_RE.match(addr), addr)
```

Never deploy a regex that hasn't been tested against the corpus you'll actually face.

### 17.9 The "Always Profile on Real Data" Idiom

A regex that runs in 10 microseconds against your test inputs may run in 30 seconds against an attacker's input. Profile on:
- Empty strings.
- Maximum-length strings (200KB? 2MB? whatever your limit is).
- Strings full of the trigger character (e.g., `"aaaa..."` for `(a+)+` patterns).
- Strings with no match at all.
- Strings with the match very late in the input.

If timing varies wildly, you have a problem.

---

## 18. Prerequisites

- Formal language theory (regular, context-free, context-sensitive languages).
- Finite automata (DFA, NFA, ε-NFA), subset construction, NFA simulation.
- Recursive descent parsing and backtracking algorithms.
- Big-O analysis and worst-case reasoning.
- Unicode encoding (UTF-8, UTF-16, codepoints, grapheme clusters).
- Basic compiler theory (lexing, parsing, IR).
- Algorithms for string matching: Boyer-Moore, Knuth-Morris-Pratt, Aho-Corasick.
- Engine concepts: bytecode, interpretation, JIT compilation.

---

## 19. Complexity

| Operation                          | Engine                  | Time complexity            | Notes                                         |
|:-----------------------------------|:------------------------|:---------------------------|:----------------------------------------------|
| Regex parse                        | All                     | O(M)                       | M = regex length                              |
| Thompson NFA construction          | All NFA-aware           | O(M)                       | At most 2M states                             |
| Subset construction (eager)        | RE2 (rare), .NET NB     | O(2^M)                     | Exponential states, exponential time          |
| Lazy subset construction           | RE2, Rust DFA           | O(N + states_visited)      | Amortized linear in input                     |
| DFA simulation                     | DFA-based               | O(N)                       | N = input length                              |
| NFA simulation (Pike VM)           | RE2 fallback, Rust      | O(N · M)                   | Track all active NFA states                   |
| Backtracking match (best)          | PCRE, Perl, Python      | O(N)                       | Disambiguates immediately                     |
| Backtracking match (worst)         | PCRE, Perl, Python      | O(2^N) or worse            | Catastrophic backtracking                     |
| Backreferences match               | Backtracking only       | NP-complete                | Aho 1990                                      |
| Multi-pattern (Hyperscan)          | Hyperscan               | O(N) for any K patterns    | SIMD-accelerated                              |
| Multi-literal (Aho-Corasick)       | Aho-Corasick            | O(N + sum-of-needles)      | Output linear in matches                      |
| DFA minimization (Hopcroft)        | Optimization step       | O(K log K)                 | K = pre-min states                            |
| Brzozowski derivatives             | .NET NonBacktracking    | O(N) amortized             | Symbolic DFA on the fly                       |

---

## 20. See Also

- regex (the practical sheet — quick reference for syntax and common patterns)
- polyglot (regex syntax differences across languages, side-by-side)
- awk (regex-driven record processing in shell)
- sed (BRE/ERE for stream editing)
- python (Python `re` module idioms and `regex` 3rd-party package)
- javascript (JS regex literal syntax, flags, modern features)
- go (Go `regexp` package, RE2 semantics)
- rust (Rust `regex` crate, performance characteristics)

---

## 21. References

### Primary Sources
- Russ Cox, **"Regular Expression Matching Can Be Simple And Fast"** (2007), https://swtch.com/~rsc/regexp/regexp1.html — the canonical "linear-time NFA simulation" article that motivated RE2. Includes the famous Perl-vs-Thompson timing chart on `(a?){n}a^n`.
- Russ Cox, **"Regular Expression Matching: the Virtual Machine Approach"** (2009), https://swtch.com/~rsc/regexp/regexp2.html — bytecode VM and four execution strategies.
- Russ Cox, **"Regular Expression Matching in the Wild"** (2010), https://swtch.com/~rsc/regexp/regexp3.html — RE2's actual algorithm: lazy DFA + cache + NFA fallback.
- Russ Cox, **"Regular Expression Matching with a Trigram Index"** (2012), https://swtch.com/~rsc/regexp/regexp4.html — Code Search architecture.
- Ken Thompson, **"Regular Expression Search Algorithm"**, *Communications of the ACM* 11(6), 1968 — the original Thompson NFA construction paper.
- Stephen Kleene, **"Representation of Events in Nerve Nets and Finite Automata"** (1956) — the foundational paper establishing regex ↔ FA equivalence.
- Janusz Brzozowski, **"Derivatives of Regular Expressions"**, *Journal of the ACM* 11(4), 1964 — the algebraic alternative to subset construction; basis for .NET NonBacktracking.
- Alfred V. Aho, **"Algorithms for Finding Patterns in Strings"**, in *Handbook of Theoretical Computer Science Volume A* (1990) — proves NP-completeness of regex with backreferences.
- Rob Pike, **"The Implementation of newsqueak"** (1991), and Sam editor source — Pike VM origin.

### Books
- Aho, Sethi, Ullman, **"Compilers: Principles, Techniques, and Tools"** (the Dragon Book), Chapter 3 covers Thompson construction, subset construction, and DFA minimization in textbook detail.
- Aho, Lam, Sethi, Ullman, **"Compilers: Principles, Techniques, and Tools"**, 2nd ed. (the Purple Dragon Book) — same chapter, expanded.
- Hopcroft, Motwani, Ullman, **"Introduction to Automata Theory, Languages, and Computation"** — the formal theory in full rigor.
- Jeffrey E.F. Friedl, **"Mastering Regular Expressions"**, 3rd ed., O'Reilly, 2006 — the practitioner's bible. Covers Perl, .NET, Java, PHP, Python, Ruby, JavaScript engine semantics.
- Stuart Russell and Peter Norvig, **"Artificial Intelligence: A Modern Approach"** — section on search and backtracking provides the algorithmic background.

### Documentation and Tools
- **regex101.com** — interactive regex tester with PCRE, ECMAScript, Python, Go, Java flavors. Shows step-by-step execution and timing.
- **regular-expressions.info** — Jan Goyvaerts's canonical online reference. Engine-by-engine feature tables, deep semantic notes.
- **PCRE2 documentation**, https://pcre.org/current/doc/ — full pattern syntax, API reference, JIT details.
- **RE2 syntax**, https://github.com/google/re2/wiki/Syntax — exact syntax accepted by RE2 / Go regexp.
- **Rust regex docs**, https://docs.rs/regex/ — including the `Performance` and `Untrusted input` sections.
- **Java Pattern Javadoc**, https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/util/regex/Pattern.html — full feature reference for `java.util.regex`.
- **Python re documentation**, https://docs.python.org/3/library/re.html, and `regex` package, https://pypi.org/project/regex/.
- **MDN RegExp reference**, https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/RegExp — JavaScript regex including `u`, `y`, `d` flags.
- **.NET Regex documentation**, https://learn.microsoft.com/en-us/dotnet/standard/base-types/regular-expressions — including the NonBacktracking mode design notes.
- **Hyperscan**, https://intel.github.io/hyperscan/dev-reference/ — Intel's high-performance multi-pattern engine.
- **Vectorscan** (ARM port of Hyperscan), https://github.com/VectorCamp/vectorscan.

### Standards
- **Unicode Technical Standard #18: Unicode Regular Expressions**, https://unicode.org/reports/tr18/ — three-level conformance specification for regex Unicode support.
- **Unicode Standard Annex #29: Unicode Text Segmentation**, https://unicode.org/reports/tr29/ — grapheme cluster, word, and sentence boundary specification.
- **POSIX.2** (IEEE Std 1003.2) — defines BRE and ERE.
- **ECMA-262** (ECMAScript Language Specification), regex section — JavaScript regex semantics.

### ReDoS and Security
- **OWASP Regular Expression Denial of Service**, https://owasp.org/www-community/attacks/Regular_expression_Denial_of_Service_-_ReDoS — attack class overview, pattern catalog.
- **Davis et al., "The Impact of Regular Expression Denial of Service (ReDoS) in Practice"** (ICSE 2018) — empirical study showing ReDoS prevalence in npm and PyPI.
- **safe-regex** (npm), https://github.com/davisjam/safe-regex — static analyzer for ReDoS-prone patterns.
- **regexploit**, https://github.com/doyensec/regexploit — Python ReDoS detector.

### Articles Worth Reading
- Cloudflare, **"Details of the Cloudflare outage on July 2, 2019"**, https://blog.cloudflare.com/details-of-the-cloudflare-outage-on-july-2-2019/ — the postmortem of a global ReDoS-induced outage from a single regex change.
- Andrew Gallant, **"ripgrep is faster than {grep, ag, git grep, ucg, pt, sift}"**, https://blog.burntsushi.net/ripgrep/ — performance deep-dive on the Rust regex crate.
- Olin Shivers, **"A Universal Scripting Framework"**, regex-aware scripting; broader context.
- Ville Laurikari, **"NFAs with Tagged Transitions"** (2000) — captures-with-DFA approach used in TRE library.

*The gap between regex theory (regular languages, linear time) and regex practice (backreferences, exponential backtracking) is vast. Every catastrophic-backtracking CVE, every ReDoS attack, every Cloudflare-style outage exploits this gap. Know which engine you're using and whether it guarantees linear time.*
