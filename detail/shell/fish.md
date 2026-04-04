# The Mathematics of fish -- Autosuggestion Algorithms and Shell Language Theory

> *fish implements a non-POSIX shell language with a context-free grammar, backed by a suggestion engine that combines prefix matching on command history with real-time syntax validation through finite automata over the file system and command namespace.*

---

## 1. Autosuggestion Engine (Prefix Matching + History Ranking)

### The Problem

As the user types characters $c_1, c_2, \ldots, c_k$, suggest the most likely completion from command history and file system paths in real-time (under 10ms latency).

### The Formula

Given prefix $p = c_1 c_2 \ldots c_k$ and history entries $H = \{h_1, h_2, \ldots, h_N\}$ ordered by recency, the suggestion is:

$$\text{suggest}(p) = \arg\max_{h_i \in H,\; p \sqsubseteq h_i} \text{score}(h_i)$$

where $p \sqsubseteq h$ means $p$ is a prefix of $h$, and:

$$\text{score}(h_i) = \alpha \cdot \text{recency}(h_i) + \beta \cdot \text{frequency}(h_i) + \gamma \cdot \text{success}(h_i)$$

- $\text{recency}(h_i) = N - i$ (position in history, most recent = highest)
- $\text{frequency}(h_i) = |\{j : h_j = h_i\}|$
- $\text{success}(h_i) = [exit\_code(h_i) = 0]$ (successful commands preferred)

### Worked Examples

History (most recent first): `git push`, `git pull`, `git push --force-with-lease`, `git status`

User types: `git pu`

Matches: `git push` (rank 1, recent), `git pull` (rank 2), `git push --force-with-lease` (rank 3)

Suggestion: `git push` (highest recency among matches).

---

## 2. Syntax Highlighting (Finite Automata)

### The Problem

Color each token in the command line in real-time as the user types: valid commands green, invalid commands red, strings yellow, operators cyan.

### The Formula

fish maintains a deterministic finite automaton (DFA) for token classification:

$$A = (Q, \Sigma, \delta, q_0, F)$$

States $Q$: `{command, argument, option, string, operator, redirect, error}`

Transition function for command validation:

$$\delta(q_{command}, t) = \begin{cases} q_{valid} & \text{if } t \in \text{PATH\_commands} \cup \text{builtins} \cup \text{functions} \\ q_{error} & \text{otherwise} \end{cases}$$

The PATH lookup is a hash table:

$$T_{lookup} = O(1) \text{ amortized}$$

For $|\text{PATH}|$ directories containing $C$ total commands:

$$T_{build\_cache} = O(|\text{PATH}| + C)$$

$$S_{cache} = O(C \times \bar{l})$$

where $\bar{l}$ is the average command name length.

---

## 3. Completion Engine (Trie-Based Matching)

### The Problem

Given a partial token, generate ranked completion candidates from commands, file paths, function arguments, and custom completion rules.

### The Formula

Completions are sourced from multiple providers. For partial input $p$:

$$C(p) = C_{cmd}(p) \cup C_{file}(p) \cup C_{custom}(p) \cup C_{history}(p)$$

Each candidate set uses prefix matching on a trie $T$:

$$C_{trie}(p) = \{w \in T : p \sqsubseteq w\}$$

Retrieval from a trie of $n$ entries with average length $l$:

$$T_{search} = O(|p| + |C_{trie}(p)|)$$

Ranking combines relevance and specificity:

$$\text{rank}(c) = w_1 \cdot \text{exact\_prefix}(c) + w_2 \cdot \text{description\_match}(c) + w_3 \cdot \text{frequency}(c)$$

---

## 4. Universal Variables (Distributed Shared State)

### The Problem

Share variable state across all running fish sessions without explicit synchronization commands.

### The Formula

Universal variables use a file-based consensus mechanism. For $n$ concurrent fish sessions $S_1, \ldots, S_n$:

$$\text{write}(var, val, S_i) \implies \forall j \neq i : \text{read}(var, S_j) = val \text{ after } \Delta t$$

The synchronization protocol:

1. Writer acquires file lock: $T_{lock} = O(1)$ amortized
2. Writer updates `~/.config/fish/fish_variables`
3. All sessions receive `SIGUSR1` signal (or poll on macOS)
4. Readers reload changed variables

Consistency model: **eventual consistency** with bounded delay:

$$\Delta t \leq T_{signal} + T_{reload} \approx O(\text{ms})$$

The variable file format is append-friendly:

$$S_{file} = O(V \times \bar{l})$$

where $V$ is the number of universal variables.

---

## 5. Event System (Reactive Programming)

### The Problem

Execute callback functions in response to signals, variable changes, and process lifecycle events.

### The Formula

fish implements an observer pattern. For event type $e$ and handler set $H_e$:

$$\text{dispatch}(e) = \forall h \in H_e : \text{execute}(h, \text{args}(e))$$

Event types and their trigger conditions:

$$\text{events} = \begin{cases} \text{on-variable}(v) & \text{when } v \text{ changes value} \\ \text{on-signal}(s) & \text{when signal } s \text{ received} \\ \text{on-event}(name) & \text{when named event emitted} \\ \text{on-process-exit}(pid) & \text{when process terminates} \\ \text{on-job-exit}(jid) & \text{when job completes} \end{cases}$$

Handler registration is $O(1)$, dispatch is $O(|H_e|)$.

---

## 6. Abbreviation Expansion (Pattern Matching)

### The Problem

Expand short tokens into full commands at the cursor position before execution, while preserving the ability to edit the expansion.

### The Formula

Given abbreviation map $M: \text{pattern} \to \text{expansion}$ and current token $t$ at position $pos$:

$$\text{expand}(t, pos) = \begin{cases} M(t) & \text{if } t \in \text{dom}(M) \land \text{position\_valid}(pos, M(t)) \\ t & \text{otherwise} \end{cases}$$

Position-dependent matching (fish 3.6+):

$$\text{position\_valid}(pos, rule) = \begin{cases} pos = 0 & \text{if rule.position = command} \\ \text{true} & \text{if rule.position = anywhere} \end{cases}$$

Regex abbreviations use NFA matching:

$$\text{expand}_{regex}(t) = \begin{cases} \text{apply\_function}(f, t) & \text{if } t \in L(r) \\ t & \text{otherwise} \end{cases}$$

Expansion is $O(|M| \times \bar{p})$ in the worst case where $\bar{p}$ is average pattern length, but typically $O(1)$ with hash lookup for exact patterns.

---

## 7. String Builtins (Regular Expression Engine)

### The Problem

Provide built-in string operations (match, replace, split, trim) that eliminate the need for external tools like `sed`, `awk`, and `tr`.

### The Formula

fish's `string match -r` uses PCRE2 (Perl-Compatible Regular Expressions). Regex compilation:

$$T_{compile} = O(|r|)$$

Matching with NFA simulation:

$$T_{match} = O(|s| \times |r|) \text{ worst case}$$

$$T_{match} = O(|s|) \text{ typical (no backtracking)}$$

Capture groups extract submatches:

$$\text{match}(r, s) = (m_0, m_1, \ldots, m_k)$$

where $m_0$ is the full match and $m_1, \ldots, m_k$ are groups.

String operations complexity:

| Operation | Time | Space |
|:---|:---|:---|
| `string match` (glob) | $O(n \times p)$ | $O(1)$ |
| `string match -r` | $O(n \times r)$ | $O(k)$ |
| `string replace` | $O(n)$ | $O(n)$ |
| `string split` | $O(n)$ | $O(n)$ |
| `string trim` | $O(n)$ | $O(1)$ |

---

## 8. Grammar and Parsing (Context-Free Language)

### The Problem

Parse fish's scripting language, which deliberately breaks from POSIX shell grammar for clarity.

### The Formula

fish uses a context-free grammar (CFG). Simplified production rules:

$$S \to \text{Statement}^*$$
$$\text{Statement} \to \text{Command} \mid \text{If} \mid \text{For} \mid \text{While} \mid \text{Switch} \mid \text{Function}$$
$$\text{If} \to \texttt{if}\; \text{Condition}\; \text{Statement}^* \;(\texttt{else if}\; \text{Condition}\; \text{Statement}^*)^* \;[\texttt{else}\; \text{Statement}^*]\; \texttt{end}$$
$$\text{For} \to \texttt{for}\; \text{var}\; \texttt{in}\; \text{args}\; \text{Statement}^*\; \texttt{end}$$

Key difference from POSIX: fish uses `end` instead of `fi`/`done`/`esac`, making the grammar LL(1) parseable:

$$\text{FIRST}(\text{If}) = \{\texttt{if}\}$$
$$\text{FIRST}(\text{For}) = \{\texttt{for}\}$$
$$\text{FOLLOW}(\text{Statement}) = \{\texttt{end}, \texttt{else}, \texttt{case}, \text{EOF}\}$$

Parsing complexity:

$$T_{parse} = O(n) \text{ (linear, LL(1) grammar)}$$

---

## Prerequisites

- formal language theory, finite automata, context-free grammars, trie data structures, regular expressions, observer pattern, eventual consistency

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Autosuggestion lookup | $O(N)$ worst, $O(1)$ typical | $O(N)$ history |
| Syntax validation | $O(1)$ per token | $O(C)$ command cache |
| Tab completion | $O(\|p\| + \|results\|)$ | $O(\text{trie size})$ |
| Abbreviation expansion | $O(1)$ hash lookup | $O(\|M\|)$ |
| String match (regex) | $O(n \times r)$ | $O(k)$ |
| Universal var sync | $O(V)$ | $O(V \times \bar{l})$ |

---

*fish's design philosophy -- discoverable defaults and a clean grammar -- is an exercise in reducing the Kolmogorov complexity of shell interaction: the simplest program that produces the user's intended behavior requires fewer bits of specification than in POSIX shells.*
