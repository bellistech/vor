# The Internals of Zsh — Extended Globbing, Completion System, and Differences from Bash

> *Zsh is a POSIX-compatible shell with extensions that address Bash's most common pain points: no word splitting on parameter expansion by default, extended globbing with qualifiers, a programmable completion system (compsys), and associative arrays as first-class citizens. It's the default shell on macOS since Catalina.*

---

## 1. Key Differences from Bash

### The Big Three

| Behavior | Bash | Zsh |
|:---------|:-----|:----|
| Word splitting on `$var` | Yes (unquoted) | **No** (by default) |
| Array indexing | 0-based | **1-based** |
| Glob no-match | Error (`failglob`) or literal | **Error** (by default) |

### Word Splitting — The Most Important Difference

```bash
# Bash:
files="a.txt b.txt"
for f in $files; do echo "$f"; done
# a.txt
# b.txt (split on spaces)

# Zsh:
files="a.txt b.txt"
for f in $files; do echo "$f"; done
# a.txt b.txt (single word — no splitting!)

# Zsh explicit splitting:
for f in ${=files}; do echo "$f"; done
# a.txt
# b.txt
```

In Zsh, `$var` behaves like Bash's `"$var"`. The `${=var}` operator explicitly requests word splitting.

### The `SH_WORD_SPLIT` Option

```zsh
setopt SH_WORD_SPLIT    # behave like Bash (split unquoted vars)
```

This is set automatically in `sh` and `ksh` emulation modes.

---

## 2. Extended Globbing

### Glob Qualifiers

Zsh's unique feature: **glob qualifiers** filter glob results by file attributes.

Syntax: `pattern(qualifiers)`

| Qualifier | Meaning | Example |
|:----------|:--------|:--------|
| `.` | Regular files | `*(.)` — all regular files |
| `/` | Directories | `*(/)` — all directories |
| `@` | Symlinks | `*(@)` — all symlinks |
| `*` | Executable | `*(*)` — all executables |
| `r`, `w`, `x` | Readable/writable/executable | `*(r)` — readable by owner |
| `R`, `W`, `X` | World-readable/writable/executable | `*(R)` — world-readable |
| `U` | Owned by current user | `*(U)` — my files |
| `L+n` | Size larger than n bytes | `*(L+1M)` — files > 1MB |
| `L-n` | Size smaller than n bytes | `*(L-1k)` — files < 1KB |
| `m+n` | Modified more than n days ago | `*(m+30)` — older than 30 days |
| `m-n` | Modified less than n days ago | `*(m-1)` — modified today |
| `om` | Sort by modification time | `*(om)` — newest first |
| `On` | Sort by name | `*(On)` — alphabetical |
| `[1,5]` | Select range | `*(om[1,5])` — 5 newest files |
| `N` | Null glob (no error if no match) | `*(N)` — empty if no match |
| `D` | Include dot files | `*(D)` — include hidden files |

### Worked Examples

```zsh
# Five largest files in current directory:
ls -la *(OL[1,5])

# All .log files modified in the last 24 hours:
ls *.log(m-1)

# All directories, sorted by name:
echo *(/:On)

# All empty files:
echo *(L0)

# All .py files owned by me, larger than 10KB:
echo **/*.py(U.L+10k)
```

### Recursive Globbing

```zsh
**/*.py          # all .py files recursively (same as Bash's globstar)
***/*.py         # same but follows symlinks
```

### Extended Glob Operators (`setopt EXTENDED_GLOB`)

| Pattern | Meaning | Bash Equivalent |
|:--------|:--------|:---------------|
| `^pattern` | NOT (negation) | `!(pattern)` with `extglob` |
| `~pattern` | NOT in filename position | — |
| `pattern1~pattern2` | Match 1 but not 2 | — |
| `#` | Zero or more (like regex `*`) | — |
| `##` | One or more (like regex `+`) | — |

---

## 3. The Completion System (compsys)

### Architecture

```
Key press (Tab)
    │
    ├── Determine completion context (command, argument, option, ...)
    │
    ├── Find matching completion function (_command)
    │
    ├── Generate completions (call completer functions)
    │
    ├── Filter and sort matches
    │
    └── Display (menu, list, or inline)
```

### Initialization

```zsh
autoload -Uz compinit && compinit
```

This loads the completion system and scans for completion definitions.

### Completion Styles

```zsh
# Case-insensitive completion:
zstyle ':completion:*' matcher-list 'm:{a-z}={A-Z}'

# Menu selection (arrow keys):
zstyle ':completion:*' menu select

# Group completions by type:
zstyle ':completion:*' group-name ''
zstyle ':completion:*:descriptions' format '%B%d%b'

# Fuzzy matching (allow 1 error):
zstyle ':completion:*' matcher-list '' \
    'm:{a-z}={A-Z}' \
    'r:|[._-]=* r:|=*' \
    'l:|=* r:|=*'
```

### `zstyle` Context String

```
:completion:function:completer:command:argument:tag
```

Each `:` segment narrows the scope. `*` matches anything.

### Writing Custom Completions

```zsh
#compdef mycommand

_mycommand() {
    _arguments \
        '-v[verbose output]' \
        '-o[output file]:filename:_files' \
        '--format[output format]:format:(json yaml toml)' \
        '*:input file:_files'
}

_mycommand "$@"
```

---

## 4. Arrays and Associative Arrays

### Indexed Arrays (1-Based!)

```zsh
arr=(one two three)
echo $arr[1]       # "one" (1-based, unlike Bash's 0-based)
echo $arr[-1]      # "three" (negative indexing)
echo $#arr         # 3 (length)
echo ${arr[2,3]}   # "two three" (slice)
```

### Associative Arrays

```zsh
typeset -A map
map=(key1 val1 key2 val2)
map[key3]=val3

echo $map[key1]           # val1
echo ${(k)map}            # key1 key2 key3 (keys)
echo ${(v)map}            # val1 val2 val3 (values)
echo ${(kv)map}           # key1 val1 key2 val2 (pairs)
echo ${#map}              # 3 (count)
```

### Array Flags (Parameter Expansion Flags)

| Flag | Meaning | Example |
|:-----|:--------|:--------|
| `(j:sep:)` | Join array with separator | `${(j:,:)arr}` → `"one,two,three"` |
| `(s:sep:)` | Split string into array | `${(s:,:)csv}` |
| `(u)` | Unique elements | `${(u)arr}` |
| `(o)` | Sort ascending | `${(o)arr}` |
| `(O)` | Sort descending | `${(O)arr}` |
| `(U)` | Uppercase | `${(U)var}` |
| `(L)` | Lowercase | `${(L)var}` |
| `(f)` | Split on newlines | `${(f)$(cat file)}` |
| `(F)` | Join with newlines | `${(F)arr}` |
| `(k)` | Keys of assoc array | `${(k)map}` |
| `(v)` | Values of assoc array | `${(v)map}` |

---

## 5. Prompt Customization

### Prompt Escape Sequences

| Escape | Meaning |
|:-------|:--------|
| `%n` | Username |
| `%m` | Hostname (short) |
| `%~` | Current directory (~ for home) |
| `%/` | Current directory (full) |
| `%F{color}` | Start foreground color |
| `%f` | Reset foreground |
| `%B` / `%b` | Bold on/off |
| `%?` | Exit code of last command |
| `%D{%H:%M}` | Date/time formatting |
| `%(?.✓.✗)` | Conditional on exit code |

### Left and Right Prompts

```zsh
PROMPT='%F{green}%n@%m%f:%F{blue}%~%f$ '
RPROMPT='%F{yellow}%D{%H:%M}%f'    # right-aligned prompt
```

### vcs_info Integration

```zsh
autoload -Uz vcs_info
precmd() { vcs_info }
zstyle ':vcs_info:git:*' formats '%b'    # branch name
PROMPT='%~ ${vcs_info_msg_0_} $ '
```

---

## 6. Zsh-Specific Features

### Anonymous Functions

```zsh
() {
    local temp="inside"
    echo $temp
}
# temp is scoped to the anonymous function
```

### Floating-Point Arithmetic

```zsh
zmodload zsh/mathfunc
echo $(( 3.14 * 2.0 ))         # 6.28 (Zsh supports floats!)
echo $(( sin(PI / 4) ))         # 0.70710678...
```

Bash only supports integer arithmetic. Zsh supports IEEE 754 doubles.

### Named Directories

```zsh
hash -d projects=~/Documents/projects
cd ~projects    # expands to ~/Documents/projects
```

### Precommand Modifiers

| Modifier | Effect |
|:---------|:-------|
| `noglob` | Disable globbing for this command |
| `nocorrect` | Disable spelling correction |
| `command` | Use external command (skip aliases/functions) |
| `builtin` | Use builtin (skip aliases/functions) |
| `exec` | Replace shell with command |

---

## 7. Hook Functions

Zsh provides hook points for executing code at specific moments:

| Hook | When It Runs |
|:-----|:-------------|
| `precmd` | Before each prompt display |
| `preexec` | Before each command execution |
| `chpwd` | After directory change |
| `periodic` | Every `$PERIOD` seconds |
| `zshaddhistory` | Before adding to history |
| `zshexit` | On shell exit |

### Hook Arrays (Multiple Functions)

```zsh
autoload -Uz add-zsh-hook

my_precmd() { echo "about to show prompt" }
add-zsh-hook precmd my_precmd
```

---

## 8. Summary of Key Differences

| Feature | Bash | Zsh |
|:--------|:-----|:----|
| Word splitting on `$var` | Yes | No (use `${=var}`) |
| Array indexing | 0-based | 1-based |
| Glob qualifiers | No | `*(.)`, `*(/)`, `*(L+1M)`, ... |
| Floating-point math | No | Yes |
| Right prompt | No | `RPROMPT` |
| Associative arrays | `declare -A` | `typeset -A` (richer API) |
| Parameter flags | No | `(j:,:)`, `(u)`, `(o)`, ... |
| Anonymous functions | No | `() { ... }` |
| Spelling correction | No | `setopt CORRECT` |
| Completion system | `bash-completion` | compsys (more powerful) |

---

*Zsh's killer feature is not any single thing — it's the cumulative effect of fixing Bash's paper cuts: no default word splitting, 1-based arrays (matching human counting), glob qualifiers for file filtering, floating-point math, and a completion system that understands command arguments. If Bash is "good enough for scripts," Zsh is "good enough to live in."*

## Prerequisites

- Bash fundamentals (parameter expansion, conditionals, loops)
- Glob patterns and extended globbing
- Completion system internals (zstyle, compsys)
- Shell option management (setopt, emulate)
