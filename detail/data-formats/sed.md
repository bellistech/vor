# The Theory of sed — Stream Editor as Finite Transducer

> *sed (stream editor) is a non-interactive text transformation tool based on a simple virtual machine with two registers (pattern space and hold space), a program counter, and a set of 25 commands. It processes input line by line, applying a script of address-command pairs. Its computational model is a finite-state transducer augmented with two unbounded buffers.*

---

## 1. The sed Processing Model

### The Default Cycle

```
For each input line:
    1. Read line into pattern space (strip trailing newline)
    2. Execute all commands in the script (in order)
    3. Print pattern space (unless -n flag)
    4. Clear pattern space
    5. Repeat with next line
```

### Two Registers

| Register | Name | Purpose |
|:---------|:-----|:--------|
| Pattern space | PS | Current working buffer — where commands operate |
| Hold space | HS | Auxiliary buffer — persistent across cycles |

Both are string registers of unbounded size. The hold space starts as an empty line (`\n`).

### Formal Model

sed is a **deterministic finite transducer** augmented with two string registers:

$$\text{State} = (\text{PS}, \text{HS}, \text{PC}, \text{line\_number})$$

Each command transforms the state:

$$\text{cmd}: \text{State} \to \text{State} \times \text{Output}^*$$

---

## 2. Addressing — Selecting Lines

### Address Types

| Address | Matches | Example |
|:--------|:--------|:--------|
| Number | Line $n$ | `3` matches line 3 |
| `$` | Last line | `$` matches final line |
| `/regex/` | Lines matching regex | `/error/` matches lines containing "error" |
| `0` | Before first line | Used in `0,/pattern/` ranges |

### Address Ranges

```
addr1,addr2   # from addr1 to addr2 (inclusive)
```

Range state machine:
```
         addr1 matches
  [OFF] ──────────────► [ON: apply command]
                              │
                        addr2 matches
                              │
  [OFF] ◄────────────────────┘
```

### Negation

```
addr!command    # apply command to lines NOT matching address
```

### Step Addresses (GNU extension)

```
first~step    # line first, then every step-th line
0~2           # even lines (0, 2, 4, ...)
1~2           # odd lines (1, 3, 5, ...)
```

---

## 3. The Substitution Command

### Syntax

```
s/regex/replacement/flags
```

### The Replacement String

| Token | Meaning |
|:------|:--------|
| `&` | Entire matched text |
| `\1` - `\9` | Backreference to capture group |
| `\n` | Newline |
| `\\` | Literal backslash |

### Flags

| Flag | Meaning |
|:-----|:--------|
| `g` | Global — replace all matches (not just first) |
| `p` | Print pattern space if substitution made |
| `w file` | Write pattern space to file if substitution made |
| `n` | Replace n-th occurrence only |
| `I` | Case-insensitive (GNU) |

### Substitution Algorithm

Without `g` flag: find first match, replace, stop.
With `g` flag: find all non-overlapping matches left-to-right, replace each.

$$\text{gsub}(r, s, t) = t[0..m_1) + s + t[m_1..m_2) + s + \ldots$$

Where $m_i$ are non-overlapping match positions.

### Worked Examples

```bash
# Replace first occurrence:
echo "aaa" | sed 's/a/b/'         # baa

# Replace all:
echo "aaa" | sed 's/a/b/g'        # bbb

# Replace 2nd occurrence:
echo "aaa" | sed 's/a/b/2'        # aba

# Backreferences — swap two words:
echo "hello world" | sed 's/\(\w\+\) \(\w\+\)/\2 \1/'
# world hello

# Using & for matched text:
echo "hello" | sed 's/[a-z]*/(&)/'
# (hello)
```

---

## 4. Hold Space Commands — The Two-Register Machine

### Commands

| Command | Action | Notation |
|:--------|:-------|:---------|
| `h` | Copy pattern space to hold space | $\text{HS} \leftarrow \text{PS}$ |
| `H` | Append pattern space to hold space | $\text{HS} \leftarrow \text{HS} + \text{\textbackslash n} + \text{PS}$ |
| `g` | Copy hold space to pattern space | $\text{PS} \leftarrow \text{HS}$ |
| `G` | Append hold space to pattern space | $\text{PS} \leftarrow \text{PS} + \text{\textbackslash n} + \text{HS}$ |
| `x` | Exchange pattern and hold spaces | $\text{PS} \leftrightarrow \text{HS}$ |

### Worked Example: Reverse Line Order

```bash
sed -n '1!G; h; $p'
```

Step-by-step for input lines A, B, C:

| Line | Before | Command | PS After | HS After |
|:-----|:-------|:--------|:---------|:---------|
| A | PS=A, HS="" | `h` | A | A |
| B | PS=B, HS=A | `1!G` → `G` | B\nA | A |
| B | | `h` | B\nA | B\nA |
| C | PS=C, HS=B\nA | `1!G` → `G` | C\nB\nA | B\nA |
| C | | `h` | C\nB\nA | C\nB\nA |
| C | | `$p` | prints: C\nB\nA | |

Output: C, B, A (reversed).

---

## 5. Branch Commands — Control Flow

### Labels and Branches

| Command | Action |
|:--------|:-------|
| `:label` | Define a label |
| `b label` | Branch (jump) to label |
| `b` | Branch to end of script (start next cycle) |
| `t label` | Branch to label if last `s///` succeeded |
| `T label` | Branch to label if last `s///` failed (GNU) |

### Loop Example: Remove Duplicate Spaces

```bash
sed ':loop; s/  / /g; t loop'
```

1. Label `:loop`
2. Replace double spaces with single
3. If substitution succeeded (`t`), jump back to `:loop`
4. Repeat until no more double spaces

### Turing Completeness

With branching and unbounded registers (pattern/hold space), sed is **Turing complete**. Any computable function can be expressed (though impractically).

---

## 6. Multi-Line Processing

### Commands

| Command | Action |
|:--------|:-------|
| `N` | Append next line to pattern space (with `\n`) |
| `P` | Print up to first `\n` in pattern space |
| `D` | Delete up to first `\n`, restart script with remainder |

### `N` — Building Multi-Line Patterns

```bash
# Join every pair of lines:
sed 'N; s/\n/ /'
```

Input: `A\nB\nC\nD` → `A B\nC D`

### `D` — Multi-Line Loop

`D` deletes the first line of a multi-line pattern space and restarts the script **without reading new input**. This creates a sliding window:

```bash
# Delete blank lines after section headers:
sed '/^#/{N; /\n$/D}'
```

---

## 7. Complete Command Reference

### Print and Delete

| Command | Action |
|:--------|:-------|
| `p` | Print pattern space |
| `P` | Print first line of pattern space |
| `d` | Delete pattern space, start next cycle |
| `D` | Delete first line of pattern space, restart script |

### Input/Output

| Command | Action |
|:--------|:-------|
| `n` | Print PS (unless `-n`), read next line into PS |
| `N` | Append next line to PS |
| `r file` | Append file contents after current line |
| `w file` | Write pattern space to file |

### Text Insertion

| Command | Action |
|:--------|:-------|
| `a\ text` | Append text after current line |
| `i\ text` | Insert text before current line |
| `c\ text` | Replace current line with text |

### Transformation

| Command | Action |
|:--------|:-------|
| `s/re/rep/` | Substitution |
| `y/src/dst/` | Transliterate (like `tr`) — character-by-character |
| `=` | Print current line number |
| `q` | Quit (print PS unless `-n`, then exit) |
| `Q` | Quit without printing (GNU) |

---

## 8. Regular Expression Dialect

### sed BRE (Basic Regular Expression)

| Pattern | Meaning | Note |
|:--------|:--------|:-----|
| `.` | Any character | |
| `*` | Zero or more | |
| `\+` | One or more | Escaped in BRE |
| `\?` | Zero or one | Escaped in BRE |
| `\(group\)` | Capture group | Escaped in BRE |
| `\|` | Alternation | Escaped in BRE (GNU) |
| `[abc]` | Character class | |
| `^` / `$` | Start/end anchor | |
| `\{n,m\}` | Repetition count | Escaped in BRE |

### ERE Mode (`-E` or `-r`)

With extended regex, grouping, alternation, and quantifiers are unescaped:

```bash
sed -E 's/(foo|bar)+/baz/g'    # ERE: no escaping
sed 's/\(foo\|bar\)\+/baz/g'   # BRE: backslash everything
```

---

## 9. Performance Characteristics

### Complexity

For input of $n$ lines, each of length $l$, and a script of $s$ commands:

$$\text{Time} = O(n \times s \times C_{\text{cmd}})$$

Where $C_{\text{cmd}}$ is the per-command cost:
- `s///`: $O(l \times |\text{regex}|)$ per line
- `y///`: $O(l)$ per line
- `h`, `g`, `x`: $O(l)$ (string copy)
- Branch: $O(1)$

### sed vs awk vs perl

| Tool | Best For |
|:-----|:---------|
| sed | Simple substitutions, line-oriented transforms |
| awk | Field-based processing, arithmetic, reporting |
| perl | Complex multi-line transforms, mixed text/data |

---

## 10. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Processing model | Line-by-line: read → execute script → print → repeat |
| Registers | Pattern space (working), Hold space (auxiliary) |
| Substitution | `s/regex/replacement/flags` — the core command |
| Control flow | `:label`, `b` (branch), `t` (conditional branch) |
| Multi-line | `N` (append next line), `P`/`D` (first-line ops) |
| Turing complete | Yes (with branches + unbounded registers) |
| Regex dialect | BRE default, ERE with `-E` flag |

---

*sed's power comes from its simplicity: two string buffers, a program counter, and 25 commands. It's the smallest useful text transformation language — smaller than awk, simpler than perl, faster than both for the problems it was designed to solve. The hold space is the key to its non-trivial capabilities: without it, sed could only do line-at-a-time substitution.*

## Prerequisites

- Regular expressions (BRE and ERE syntax)
- Stream processing and line-oriented text models
- Pattern space and hold space concepts
- Shell piping and stdin/stdout conventions
