# The Internals of Bash — Expansion, Pipelines, and Process Model

> *Bash processes each command line through a precise 7-stage expansion pipeline: brace expansion, tilde expansion, parameter/variable expansion, command substitution, arithmetic expansion, word splitting, and filename expansion (globbing). Understanding this order explains every "quoting mystery" and word-splitting bug. Pipelines use kernel pipe buffers (default 64KB on Linux), and process substitution creates temporary file descriptors.*

---

## 1. The Expansion Pipeline

### Seven Stages in Exact Order

```
Raw command line
    │
    ├── 1. Brace expansion        {a,b,c} → a b c
    │
    ├── 2. Tilde expansion         ~ → /home/user
    │
    ├── 3. Parameter expansion     $var, ${var:-default}, ${var%pattern}
    │
    ├── 4. Command substitution    $(command) or `command`
    │
    ├── 5. Arithmetic expansion    $((expression))
    │
    ├── 6. Word splitting          IFS-based splitting of unquoted results
    │
    └── 7. Filename expansion      *, ?, [abc] (globbing)
    │
    Final: Quote removal (remove remaining quotes)
```

### Why Order Matters

```bash
files="*.txt"
echo $files      # Step 3: $files → "*.txt"
                 # Step 6: word splitting (no effect here)
                 # Step 7: glob expansion → "a.txt b.txt c.txt"

echo "$files"    # Step 3: $files → "*.txt"
                 # Steps 6,7: SKIPPED (double-quoted)
                 # Result: literal "*.txt"
```

### Stage Details

**Stage 1 — Brace Expansion** (before any variable expansion):
```bash
echo {1..5}        # 1 2 3 4 5
echo {a..z}        # a b c ... z
echo file{A,B,C}   # fileA fileB fileC
echo {1..10..2}    # 1 3 5 7 9 (step)
```

**Stage 2 — Tilde Expansion:**
```bash
~           → /home/user        (HOME)
~bob        → /home/bob         (bob's home)
~+          → $PWD
~-          → $OLDPWD
```

**Stage 3 — Parameter Expansion:**

| Syntax | Meaning |
|:-------|:--------|
| `${var}` | Value of var |
| `${var:-default}` | Default if unset/empty |
| `${var:=default}` | Assign default if unset/empty |
| `${var:+alternate}` | Alternate if set and non-empty |
| `${var:?error}` | Error if unset/empty |
| `${#var}` | String length |
| `${var%pattern}` | Remove shortest suffix match |
| `${var%%pattern}` | Remove longest suffix match |
| `${var#pattern}` | Remove shortest prefix match |
| `${var##pattern}` | Remove longest prefix match |
| `${var/old/new}` | Replace first occurrence |
| `${var//old/new}` | Replace all occurrences |
| `${var:offset:length}` | Substring |
| `${var^}` | Uppercase first char |
| `${var^^}` | Uppercase all |
| `${var,}` | Lowercase first char |
| `${var,,}` | Lowercase all |

**Stage 6 — Word Splitting:**

Unquoted results of expansions (stages 3-5) are split on characters in `$IFS` (default: space, tab, newline).

$$\text{split}(s, \text{IFS}) = \text{tokens where delimiters} \in \text{IFS}$$

**This is why you must quote variables:** `"$var"` prevents word splitting.

---

## 2. Quoting Rules

### Three Quoting Mechanisms

| Quoting | Expands Variables? | Expands Globs? | Word Splits? |
|:--------|:------------------:|:--------------:|:------------:|
| Unquoted | Yes | Yes | Yes |
| Double `"..."` | Yes | No | No |
| Single `'...'` | No | No | No |
| `$'...'` | No (but interprets escapes) | No | No |

### The Golden Rule

$$\text{Always double-quote variable expansions: } "\$var", "\$@", "\$(cmd)"$$

Exceptions: inside `[[ ]]` (no word splitting) and intentional glob expansion.

### `$@` vs `$*`

| Syntax | Unquoted | Quoted |
|:-------|:---------|:-------|
| `$*` | All args, split on IFS | `"$1 $2 $3"` (one word) |
| `$@` | All args, split on IFS | `"$1" "$2" "$3"` (separate words) |

`"$@"` is almost always what you want — it preserves argument boundaries.

---

## 3. Pipeline Implementation

### Kernel Pipe Buffer

A pipe `cmd1 | cmd2` creates a kernel pipe:

```
cmd1 stdout ──► [pipe buffer: 64KB] ──► cmd2 stdin
```

| Property | Linux | macOS |
|:---------|:------|:------|
| Default buffer size | 64 KB (16 pages) | 16 KB |
| Max buffer size | 1 MB (`fcntl F_SETPIPE_SZ`) | Fixed |
| Atomicity | Writes ≤ `PIPE_BUF` (4KB) are atomic | Same |

### Pipeline Execution

Each command in a pipeline runs in a **separate subprocess**:

```bash
cmd1 | cmd2 | cmd3
```

- Shell forks 3 child processes
- Creates 2 pipe pairs: `pipe1`, `pipe2`
- `cmd1`: stdout → pipe1 write end
- `cmd2`: stdin → pipe1 read end, stdout → pipe2 write end
- `cmd3`: stdin → pipe2 read end

### Exit Status

By default, the pipeline's exit status is the **last** command's exit status.

With `set -o pipefail`: exit status is the **rightmost failing** command's status.

```bash
false | true    # exit status: 0 (default), 1 (pipefail)
```

### Subshell Issue

In Bash (not zsh), the **last** command in a pipeline runs in a subshell:

```bash
echo "hello" | read var
echo "$var"    # empty! (read ran in subshell)

# Fix 1: lastpipe
shopt -s lastpipe
echo "hello" | read var    # works (Bash 4.2+)

# Fix 2: process substitution
read var < <(echo "hello")
```

---

## 4. Process Substitution

### Syntax

```bash
<(command)    # creates a readable file descriptor
>(command)    # creates a writable file descriptor
```

### Implementation

Process substitution creates a **named pipe** (FIFO) or `/dev/fd/N` entry:

```bash
diff <(sort file1) <(sort file2)

# Expands to something like:
diff /dev/fd/63 /dev/fd/62
# where fd 63 = pipe from "sort file1"
#       fd 62 = pipe from "sort file2"
```

### Cost

$$\text{Process substitution} = \text{fork}() + \text{pipe}() + \text{exec}()$$

Each `<(...)` creates a new subprocess and a pipe. Lighter than temporary files, heavier than simple pipes.

---

## 5. Subshell Forking

### What Creates a Subshell

| Construct | Subshell? | Why |
|:----------|:----------|:----|
| `(commands)` | Yes | Explicit subshell |
| `cmd1 \| cmd2` | Yes (each) | Pipeline components |
| `$(command)` | Yes | Command substitution |
| `<(command)` | Yes | Process substitution |
| `cmd &` | Yes | Background job |
| `{ commands; }` | No | Group in current shell |

### Fork Cost

$$\text{fork() on Linux} \approx 100-500 \mu s$$

Modern Linux uses **copy-on-write** (COW) — child shares parent's memory pages until one writes. The dominant cost is page table duplication.

### Avoiding Unnecessary Forks

```bash
# BAD — 3 forks:
result=$(echo "$var" | tr 'a-z' 'A-Z')

# GOOD — 0 forks (bash built-in):
result="${var^^}"
```

```bash
# BAD — fork per iteration:
cat file | while read line; do echo "$line"; done

# GOOD — no extra fork:
while read line; do echo "$line"; done < file
```

---

## 6. Arithmetic Evaluation

### `$((...))` — Arithmetic Context

Inside arithmetic context, all expansions use integer arithmetic (C-like):

| Operator | Meaning |
|:---------|:--------|
| `+`, `-`, `*`, `/`, `%` | Basic arithmetic |
| `**` | Exponentiation |
| `<<`, `>>` | Bit shift |
| `&`, `\|`, `^` | Bitwise AND, OR, XOR |
| `&&`, `\|\|` | Logical AND, OR |
| `? :` | Ternary |
| `++`, `--` | Increment/decrement |

Variables inside `$((...))` don't need `$`:

```bash
x=5
echo $((x * 3 + 1))    # 16 (no $ before x needed)
```

### Integer-Only

Bash arithmetic is **integer-only**. For floating point:

```bash
echo "3.14 * 2" | bc -l    # 6.28
printf "%.2f" "$(echo "3.14 * 2" | bc -l)"
```

---

## 7. Signal Handling and Traps

### The `trap` Command

```bash
trap 'cleanup_function' EXIT        # on script exit
trap 'echo "Ctrl-C caught"' INT     # on SIGINT
trap '' INT                          # ignore SIGINT
trap - INT                           # reset to default
```

### Common Signals

| Signal | Number | Default | Trappable? |
|:-------|:------:|:--------|:----------:|
| SIGINT | 2 | Terminate | Yes |
| SIGTERM | 15 | Terminate | Yes |
| SIGKILL | 9 | Terminate | **No** |
| SIGHUP | 1 | Terminate | Yes |
| SIGPIPE | 13 | Terminate | Yes |
| SIGSTOP | 19 | Stop | **No** |
| EXIT | — | — | Yes (Bash pseudo-signal) |
| ERR | — | — | Yes (on command failure) |
| DEBUG | — | — | Yes (before each command) |

---

## 8. Strict Mode

```bash
set -euo pipefail
```

| Flag | Effect |
|:-----|:-------|
| `-e` (`errexit`) | Exit on any command failure |
| `-u` (`nounset`) | Error on unset variable reference |
| `-o pipefail` | Pipeline fails if any component fails |

### Common `set -e` Gotchas

```bash
# This exits even though the failure is "handled":
count=$(grep -c "pattern" file)    # grep returns 1 if no match

# Fix:
count=$(grep -c "pattern" file || true)
```

---

## 9. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Expansion order | Brace → tilde → param → cmd → arith → split → glob |
| Word splitting | Unquoted expansions split on `$IFS` |
| Pipeline buffer | 64KB kernel pipe buffer (Linux) |
| Pipeline subshells | Each component forks |
| Process substitution | Named pipe + fork, `<(cmd)` / `>(cmd)` |
| Fork cost | ~100-500 us (COW pages) |
| Arithmetic | Integer-only, C-like operators |
| Quoting rule | Always double-quote: `"$var"`, `"$@"`, `"$(cmd)"` |

---

*Every Bash bug you've ever encountered — word splitting on filenames with spaces, lost variables in pipelines, unexpected glob expansion — is explained by the 7-stage expansion pipeline and the subshell forking rules. Learn the order of expansions, learn what creates a subshell, and quote everything. That's the entire discipline of reliable shell scripting.*

## Prerequisites

- Process model (fork, exec, wait, exit codes, signals)
- File descriptors and I/O redirection (stdin, stdout, stderr)
- Word splitting and globbing rules
- Environment variables and variable scoping (export, local, subshell)
