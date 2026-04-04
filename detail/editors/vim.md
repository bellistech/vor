# The Theory of Vim — Command Grammar, Modal Editing, and Registers

> *Vim's key sequences form a context-free grammar: operator [count] motion/text-object. This composable grammar means N operators and M motions give N x M commands — not N x M bindings to memorize, but N + M primitives to combine. Key parsing uses a trie structure, macro execution treats register contents as instruction streams, and the undo tree (not stack) enables non-linear history navigation.*

---

## 1. Command Grammar — The Composable Language

### The Grammar

Normal mode commands follow a context-free grammar:

```
command   = [count] operator [count] motion
          | [count] operator [count] text-object
          | [count] operator operator         (linewise: dd, yy, cc, >>)
          | [count] simple-command            (x, r, J, ~, etc.)
```

### Operators (Verbs)

| Key | Operator | Action |
|:---:|:---------|:-------|
| `d` | Delete | Remove text, save to register |
| `c` | Change | Delete text, enter insert mode |
| `y` | Yank | Copy text to register |
| `>` | Indent | Shift right |
| `<` | Dedent | Shift left |
| `=` | Format | Auto-indent |
| `gq` | Format text | Reflow paragraph |
| `gu` | Lowercase | Convert to lowercase |
| `gU` | Uppercase | Convert to uppercase |
| `g~` | Toggle case | Swap case |
| `!` | Filter | Pipe through external command |

### Motions (Nouns — Movements)

| Key | Motion | Scope |
|:---:|:-------|:------|
| `w` | Next word start | Word |
| `e` | Next word end | Word |
| `b` | Previous word start | Word |
| `W/E/B` | Same but WORD (whitespace-delimited) | WORD |
| `0` | Line start | Line |
| `^` | First non-blank | Line |
| `$` | Line end | Line |
| `f{c}` | Forward to char | Inline |
| `t{c}` | Forward till (before) char | Inline |
| `}` | Next paragraph | Paragraph |
| `G` | End of file | File |
| `gg` | Start of file | File |
| `/{pattern}` | Next search match | File |

### The Combinatorial Explosion

With 11 operators and ~30 motions:

$$\text{Commands} = 11 \times 30 = 330 \text{ unique operations}$$

You memorize $11 + 30 = 41$ primitives, not 330 bindings.

### Text Objects (Noun Phrases)

Text objects select structured regions. Always used with operators:

| Key | Object | With `i` (inner) | With `a` (around) |
|:---:|:-------|:-----------------|:------------------|
| `w` | Word | Word only | Word + trailing space |
| `s` | Sentence | Sentence only | Sentence + trailing space |
| `p` | Paragraph | Paragraph only | Paragraph + blank line |
| `"` | Double-quoted | Inside quotes | Including quotes |
| `'` | Single-quoted | Inside quotes | Including quotes |
| `(` or `)` | Parentheses | Inside parens | Including parens |
| `{` or `}` | Braces | Inside braces | Including braces |
| `[` or `]` | Brackets | Inside brackets | Including brackets |
| `t` | Tag (HTML/XML) | Inside tags | Including tags |

### Worked Examples

| Keystrokes | Grammar Parse | Action |
|:-----------|:-------------|:-------|
| `d2w` | delete + 2 + word | Delete next 2 words |
| `ci"` | change + inner + double-quote | Change text inside quotes |
| `>ap` | indent + around + paragraph | Indent entire paragraph |
| `gUiw` | uppercase + inner + word | Uppercase current word |
| `3dd` | 3 + delete + delete(linewise) | Delete 3 lines |
| `y$` | yank + end-of-line | Copy to end of line |

---

## 2. Key Sequence Parsing — The Trie

### Trie-Based Dispatch

Vim's key parser maintains a **trie** (prefix tree) of all valid key sequences:

```
g ─── U ─── [motion]   → uppercase
  ─── u ─── [motion]   → lowercase
  ─── ~ ─── [motion]   → toggle case
  ─── g                → go to line 1

d ─── d                → delete line
  ─── w                → delete word
  ─── i ─── w          → delete inner word
      ─── "          → delete inner quotes
```

### Timeout for Ambiguous Prefixes

When a prefix could be a complete command or the start of a longer one, Vim waits for `timeoutlen` (default 1000ms):

- `d` could be the start of `dd`, `dw`, `diw`, etc.
- After timeout with no second key: treated as incomplete (no action)

---

## 3. Modal Editing — State Machine

### Modes as States

```
                        i, a, o, ...
              ┌────────────────────────────┐
              │                            ▼
          ┌────────┐                  ┌──────────┐
    ──►   │ Normal │                  │  Insert  │
          └───┬────┘                  └─────┬────┘
              │                            │
    v, V,     │    : (command-line)        │ Esc
    Ctrl-V    │                            │
              ▼                            │
         ┌──────────┐    ┌────────────┐    │
         │  Visual  │    │  Command   │◄───┘
         └─────┬────┘    └──────┬─────┘
               │                │
               └── Esc ────────►│ Enter (execute)
                                ▼
                           Back to Normal
```

### Mode Costs

| Transition | Cost | Why It Matters |
|:-----------|:-----|:---------------|
| Normal → Insert | 1 keystroke | Cheap |
| Insert → Normal | 1 keystroke (Esc) | Cheap |
| Normal → Visual | 1 keystroke | Cheap |
| Normal → Command | 1 keystroke (`:`) | Cheap |
| Any → Normal | Always Esc | Universal escape |

The design goal: **Normal mode is the resting state**. You visit other modes briefly and return.

---

## 4. Registers — Named Storage

### Register Types

| Register | Syntax | Purpose |
|:---------|:-------|:--------|
| `"a`-`"z` | Named | Explicit storage (26 registers) |
| `"A`-`"Z` | Named append | Append to corresponding lowercase |
| `""` | Unnamed | Last delete/yank (default) |
| `"0` | Yank | Last yank only |
| `"1`-`"9` | Numbered | Last 9 deletes (stack, `"1` = most recent) |
| `"-` | Small delete | Deletes < 1 line |
| `"*` | System clipboard | Selection (X11) or clipboard (macOS) |
| `"+` | System clipboard | Clipboard (X11) |
| `"/` | Search | Last search pattern |
| `":` | Command | Last command-line command |
| `".` | Insert | Last inserted text |
| `"_` | Black hole | Discard (no side effects) |
| `"=` | Expression | Evaluate expression |

### Macro Execution = Register Playback

A macro recorded with `q{reg}` stores keystrokes in register `{reg}`. Playback with `@{reg}` **feeds the register contents back through the key parser** as if typed.

This means:
- `"ayy` (yank line to register a) + `@a` = execute that line's text as vim commands
- You can edit macros by pasting the register, modifying, and yanking back

### The Dot Command — Single-Entry Redo

`.` repeats the last **change** (insert, delete, replace, etc.). It replays the exact operator-motion/text-object combination.

The dot command is separate from the undo/redo system — it tracks:
- The operator
- The motion or text-object
- Any text typed in insert mode

---

## 5. The Undo Tree

### Tree, Not Stack

Most editors maintain an undo **stack** — redo is lost when you undo and then make a change. Vim maintains an undo **tree** — every state is preserved.

```
         state 1
          /    \
      state 2   state 2' (after undo + new edit)
        |
      state 3

Linear undo: state 1 → 2 → 3
Undo:        state 3 → 2 → 1
New edit:    state 1 → 2' (state 2, 3 lost in linear model)

Vim tree: ALL states preserved
  g- → move to earlier state in chronological time
  g+ → move to later state in chronological time
  :earlier 5m → state from 5 minutes ago
```

### Undo Granularity

Each "change" in normal mode is one undo unit. In insert mode, the entire insertion (from entering insert mode to pressing Esc) is one undo unit.

Break undo units in insert mode with `Ctrl-G u`:

```
iHello<C-G u> World<Esc>
# Now 'u' undoes " World" only, not "Hello World"
```

---

## 6. Search and the Pattern Register

### Regular Expression Dialect

Vim uses its own regex dialect with two modes:

| Setting | Escaping | Similar To |
|:--------|:---------|:-----------|
| `\v` (very magic) | Only alphanumeric/underscore are literal | PCRE (modern) |
| `\m` (magic, default) | Some special chars need `\` | POSIX BRE/ERE hybrid |
| `\M` (nomagic) | Most chars are literal | Old grep |
| `\V` (very nomagic) | Only `\` is special | Literal |

### Search Offsets

```
/pattern/e          " cursor at end of match
/pattern/+2         " 2 lines below match
/pattern/b+3        " 3 characters from beginning of match
```

---

## 7. Command-Line Mode — Ex Commands

### The Ex Heritage

Vim's `:` commands descend from **ex**, the line editor (1976). The addressing scheme:

```
:[range]command[!] [args]
```

### Range Notation

| Range | Meaning |
|:------|:--------|
| `.` | Current line |
| `$` | Last line |
| `%` | Entire file (`1,$`) |
| `'a,'b` | From mark `a` to mark `b` |
| `/pat1/,/pat2/` | From next match of pat1 to next match of pat2 |
| `+n` / `-n` | Relative offset |

### Power Commands

| Command | Action |
|:--------|:-------|
| `:g/pattern/cmd` | Execute `cmd` on every matching line |
| `:v/pattern/cmd` | Execute `cmd` on every NON-matching line |
| `:s/pat/rep/g` | Substitute (on range) |
| `:norm @a` | Execute macro `a` on range |
| `:%!sort` | Filter entire file through `sort` |

The `:g` (global) command is essentially `grep` + `ed command` — it was the origin of the name `grep` (g/re/p = global regex print).

---

## 8. Summary of Key Concepts

| Concept | Formalization | Key Detail |
|:--------|:-------------|:-----------|
| Command grammar | CFG: `[count] operator [count] motion` | $N + M$ primitives → $N \times M$ commands |
| Key parsing | Trie with timeout | Prefix disambiguation |
| Modes | Finite state machine | Normal is resting state |
| Registers | 48+ named storage cells | Macros = register playback |
| Undo | Tree (not stack) | Chronological navigation with `g-`/`g+` |
| Dot command | Last-change replay | Most powerful single key |
| `:g` | Global command | Origin of `grep` |

---

*Vim's design is a language, not a list of shortcuts. The operator-motion grammar is its central innovation — it means that learning one new operator or one new motion multiplies your capabilities, not adds to them. This is why experienced Vim users are faster: they're not remembering 500 bindings, they're composing 50 primitives.*

## Prerequisites

- Modal editing concepts (normal, insert, visual, command-line modes)
- Operator-motion grammar (verbs, nouns, modifiers)
- Regular expressions (Vim regex dialect)
- Terminal emulator basics and key sequence handling
