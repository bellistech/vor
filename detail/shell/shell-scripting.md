# The Theory of Shell Scripting — POSIX, Portability, and Idioms

> *Shell scripting is programming in the POSIX shell language — a standardized subset that works across bash, zsh, dash, ash, and ksh. Understanding the difference between POSIX and bash-specific features, the process model (fork-exec), exit code conventions, and defensive scripting patterns is what separates reliable scripts from fragile ones.*

---

## 1. The Process Model — Fork and Exec

### Every External Command

```
Shell process (PID 1000)
    │
    ├── fork()  →  Child process (PID 1001, copy of shell)
    │                   │
    │                   └── exec("ls", "-la")  →  replaced by ls
    │
    └── wait(1001)  →  collects exit status
```

$$\text{External command cost} = \text{fork}() + \text{exec}() + \text{wait}()$$

### Built-in Commands Skip Fork

| Built-in | External | Why It Matters |
|:---------|:---------|:---------------|
| `cd` | — | Must change current process's directory |
| `echo` | `/bin/echo` | Built-in avoids fork overhead |
| `test` / `[` | `/bin/test` | Built-in avoids fork overhead |
| `read` | — | Must set variables in current shell |
| `export` | — | Must modify current environment |
| `eval` | — | Must execute in current shell |

### Subshell vs Current Shell

| Syntax | Shell | Variable Changes Persist? |
|:-------|:------|:-------------------------|
| `{ cmd; }` | Current | Yes |
| `( cmd )` | Subshell | No |
| `cmd1 \| cmd2` | Subshells (each) | No (Bash), last in current (Zsh) |
| `$(cmd)` | Subshell | No |

---

## 2. Exit Codes — The Return Type of Shell

### Convention

| Code | Meaning |
|:---:|:---------|
| 0 | Success |
| 1 | General error |
| 2 | Misuse of shell builtin |
| 126 | Command found but not executable |
| 127 | Command not found |
| 128+N | Killed by signal N |
| 130 | Ctrl-C (128 + 2 = SIGINT) |
| 137 | Killed (128 + 9 = SIGKILL) |
| 143 | Terminated (128 + 15 = SIGTERM) |

### Boolean Logic with Exit Codes

```bash
# AND: run cmd2 only if cmd1 succeeds
cmd1 && cmd2

# OR: run cmd2 only if cmd1 fails
cmd1 || cmd2

# NOT: invert exit code
! cmd1

# Compound:
cmd1 && cmd2 || cmd3    # WARNING: cmd3 runs if cmd2 fails too!
```

### Function Return Values

```bash
is_root() {
    [ "$(id -u)" -eq 0 ]    # return value = exit code of last command
}

if is_root; then
    echo "Running as root"
fi
```

---

## 3. POSIX Portability

### POSIX Shell vs Bash

| Feature | POSIX | Bash | Portable? |
|:--------|:-----:|:----:|:---------:|
| `[[ ]]` | No | Yes | No |
| `[  ]` / `test` | Yes | Yes | Yes |
| `$(cmd)` | Yes | Yes | Yes |
| `` `cmd` `` | Yes | Yes | Yes (but nesting is painful) |
| `$((expr))` | Yes | Yes | Yes |
| `${var/old/new}` | No | Yes | No |
| `${var,,}` | No | Yes | No |
| `local` keyword | Not guaranteed | Yes | Mostly portable |
| Arrays | No | Yes | No |
| `[[ =~ ]]` regex | No | Yes | No |
| `function f()` | No | Yes | No |
| `f()` (without `function`) | Yes | Yes | Yes |
| `set -o pipefail` | No (POSIX 2024: yes) | Yes | Becoming portable |

### Writing Portable Scripts

Shebang options:
```bash
#!/bin/sh           # POSIX shell (could be dash, ash, bash in POSIX mode)
#!/bin/bash          # Bash specifically
#!/usr/bin/env bash  # Bash via PATH (more portable across systems)
```

### The `[` vs `[[` Decision

| Feature | `[ ]` (POSIX) | `[[ ]]` (Bash/Zsh) |
|:--------|:-------------|:-------------------|
| Word splitting | Yes (must quote vars) | No |
| Glob expansion | Yes | No |
| Regex matching | No | `=~` |
| Pattern matching | No | `==` with globs |
| `&&` / `\|\|` inside | No (use `-a` / `-o`) | Yes |
| `<` / `>` | String comparison (may conflict with redirects) | String comparison (safe) |

---

## 4. Test Expressions

### File Tests

| Test | True If |
|:-----|:--------|
| `-e file` | File exists |
| `-f file` | Regular file |
| `-d file` | Directory |
| `-L file` | Symbolic link |
| `-r file` | Readable |
| `-w file` | Writable |
| `-x file` | Executable |
| `-s file` | Non-empty (size > 0) |
| `f1 -nt f2` | f1 newer than f2 |
| `f1 -ot f2` | f1 older than f2 |

### String Tests

| Test | True If |
|:-----|:--------|
| `-z "$s"` | Empty string |
| `-n "$s"` | Non-empty string |
| `"$a" = "$b"` | Equal |
| `"$a" != "$b"` | Not equal |

### Integer Tests

| Test | True If |
|:-----|:--------|
| `$a -eq $b` | Equal |
| `$a -ne $b` | Not equal |
| `$a -lt $b` | Less than |
| `$a -le $b` | Less or equal |
| `$a -gt $b` | Greater than |
| `$a -ge $b` | Greater or equal |

---

## 5. Here Documents and Here Strings

### Here Document

```bash
cat <<EOF
Hello, $USER
Today is $(date)
EOF
```

Variables and command substitutions are expanded. To suppress expansion:

```bash
cat <<'EOF'
Literal $USER and $(date)
EOF
```

### Here String (Bash/Zsh, not POSIX)

```bash
read var <<< "hello world"
```

Equivalent to `echo "hello world" | read var` but without a subshell.

### Indented Here Document

```bash
cat <<-EOF
	Tabs are stripped from the beginning
	of each line (tabs, not spaces)
	EOF
```

---

## 6. Common Patterns

### Safe Temporary Files

```bash
tmpfile=$(mktemp) || exit 1
trap 'rm -f "$tmpfile"' EXIT
```

### Checking Command Existence

```bash
# POSIX:
command -v git >/dev/null 2>&1 || { echo "git required"; exit 1; }

# Do NOT use 'which' — it's not POSIX and behaves differently across systems
```

### Reading Files Line by Line

```bash
while IFS= read -r line; do
    printf '%s\n' "$line"
done < file.txt
```

- `IFS=` prevents stripping leading/trailing whitespace
- `-r` prevents backslash interpretation
- `printf` instead of `echo` handles lines starting with `-`

### Processing Arguments

```bash
while [ $# -gt 0 ]; do
    case "$1" in
        -v|--verbose) verbose=1 ;;
        -o|--output)  output="$2"; shift ;;
        --)           shift; break ;;
        -*)           echo "Unknown option: $1" >&2; exit 1 ;;
        *)            break ;;
    esac
    shift
done
```

---

## 7. Defensive Scripting

### The Strict Mode Template

```bash
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# ... script body ...
```

| Setting | Protection |
|:--------|:-----------|
| `set -e` | Exit on error |
| `set -u` | Error on undefined variable |
| `set -o pipefail` | Pipeline fails on any component failure |
| `IFS=$'\n\t'` | Prevents space-based word splitting accidents |

### Quoting Rules (Recap)

$$\text{Rule: always double-quote unless you specifically need splitting or globbing}$$

```bash
# WRONG:
rm $file            # word splits, globs, breaks on spaces

# RIGHT:
rm "$file"          # safe

# WRONG:
for f in $(find . -name "*.txt"); do  # breaks on spaces in filenames

# RIGHT:
find . -name "*.txt" -print0 | while IFS= read -r -d '' f; do
    echo "$f"
done
```

---

## 8. Script Performance

### Command Cost Hierarchy

| Operation | Relative Cost | Example |
|:----------|:-------------|:--------|
| Variable assignment | 1x | `x=5` |
| Built-in command | ~10x | `echo "$x"` |
| Function call | ~10x | `myfunc` |
| Fork + built-in | ~1000x | `$(echo "$x")` |
| Fork + exec | ~5000x | `$(date)` |
| Fork + exec (complex) | ~10000x | `$(awk '{print $1}' file)` |

### The Loop Anti-Pattern

```bash
# TERRIBLE — forks per line:
while read line; do
    echo "$line" | grep "pattern" | awk '{print $1}'
done < file

# GOOD — single process:
awk '/pattern/ {print $1}' file
```

$$\text{Anti-pattern cost} = O(n \times C_{\text{fork}})$$
$$\text{Correct cost} = O(n \times C_{\text{line}}) + C_{\text{fork}}$$

---

## 9. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Process model | Fork-exec for externals, built-ins run in-shell |
| Exit codes | 0 = success, 1-125 = error, 126-127 = special, 128+N = signal |
| Portability | POSIX `sh` for scripts that must run everywhere |
| Quoting | Double-quote everything: `"$var"`, `"$@"` |
| Testing | `[ ]` for POSIX, `[[ ]]` for Bash/Zsh features |
| Performance | Minimize forks — use built-ins, avoid command substitution in loops |
| Strict mode | `set -euo pipefail` |
| File processing | Let awk/sed/grep process files, not shell loops |

---

*The fundamental truth of shell scripting: the shell is a job control language, not a programming language. It excels at orchestrating programs (pipes, redirects, process management) and fails at data processing (loops, arithmetic, string manipulation). Write your logic in awk/python/go and your orchestration in shell. The scripts that last are the ones that understand this boundary.*

## Prerequisites

- POSIX shell specification (portable constructs vs bash extensions)
- Process management (fork, exec, signals, exit codes)
- File I/O and redirection (pipes, here documents, process substitution)
- Error handling patterns (set -euo pipefail, trap, exit codes)
