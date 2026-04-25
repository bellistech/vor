# The Internals of Zsh — Globbing Engine, ZLE, Completion System, and Performance

> *Zsh is not just "Bash with extras". Underneath the familiar prompt sits a hand-written recursive-descent parser that lowers shell source into a compact internal "wordcode" representation, a multi-pass expansion pipeline whose ordering is loaded with subtle correctness traps, the most powerful globbing engine in any mainstream shell, the Zsh Line Editor (ZLE) — a programmable terminal-mode line editor with widgets, keymaps, and a multi-line buffer model — and a completion system (compsys) that compiles tag/style trees at runtime to drive context-aware tab completion. This deep dive examines the parts of zsh that the cheatsheet glosses over: how the parser hands work to the executor, the canonical glob qualifier evaluation order, every common parameter-expansion flag, the matchspec rules that power case-insensitive completion, the precmd/preexec hook chain, the loadable module system, the prompt engine cost model, and the plugin-manager turbo-mode tradeoffs. Code is real, options are spelled out, and zsh-version gates are noted where they matter.*

---

## 1. Zsh Architecture — Parser, Wordcode, and Executor

### 1.1 The Three-Stage Pipeline

Zsh source code travels through three distinct stages before a single syscall fires:

```
Source text                                    │
   │                                           │
   ▼                                           │  Lex/Parse:
[ Lexer (Src/lex.c)                       ]   │  ~10K LOC of C.
[   tokenization → Tok* token stream      ]   │  Hand-written —
[ Parser (Src/parse.c)                    ]   │  no yacc/bison.
[   recursive descent → wordcode (Wordcode) ] │
   │                                           │
   ▼                                           │
[ Wordcode buffer (compact bytecode-like)  ]  │  Cached on disk
[   shfunctab entries point at wordcode    ]  │  in zcompiled
[   for compiled functions (zcompile)      ]  │  files when used.
   │                                           │
   ▼                                           │
[ Executor (Src/exec.c)                   ]   │  Tree-walking
[   visits Wordcode nodes, performs        ]  │  interpreter over
[   expansions, forks, execs, redirs       ]  │  the wordcode AST.
   │                                           │
   ▼                                           │
[ Kernel: fork(2), execve(2), pipe(2),     ]  │
[   dup2(2), wait4(2), tcsetpgrp(2)        ]  │
```

The parser does not construct a conventional pointer-based AST. Instead it writes into a compact array of `wordcode` 32-bit cells (see `Src/zsh.h`'s `Wordcode` typedef). Each cell encodes either an opcode (`WC_*` constants — `WC_LIST`, `WC_SUBLIST`, `WC_PIPE`, `WC_REDIR`, `WC_ASSIGN`, `WC_SIMPLE`, `WC_FUNCDEF`, `WC_IF`, `WC_FOR`, `WC_WHILE`, `WC_CASE`, …) or a payload (string-table offset, length, control flag). Strings are interned in a separate string table.

### 1.2 Why Wordcode

Wordcode exists for three reasons:

1. **Compactness.** A function body is one contiguous array. Function dispatch is `Eprog *prog = shfunctab[name]; execwordcode(prog->prog, prog->len);` — pointer chase, no tree walk over scattered nodes.
2. **Caching.** `zcompile foo.zwc foo.zsh` writes the wordcode to disk. `autoload`-able functions fall back to `.zwc` files automatically, skipping re-parsing.
3. **Re-entrancy.** Executing a function does not require reparsing its source. Functions parsed once at `autoload -Uz funcname` are cached forever (or until `unfunction`).

### 1.3 The Visit Pattern

The executor in `Src/exec.c` walks wordcode through a giant `switch` dispatch:

```c
/* simplified from Src/exec.c — execpline, execlist, execcmd */
static int
execlist(Estate state, int dont_change_job, int exiting)
{
    while (state->pc < state->end) {
        wordcode code = *state->pc;
        switch (wc_code(code)) {
        case WC_LIST:     execlist(...);   break;
        case WC_SUBLIST:  execsublist(...);break;
        case WC_PIPE:     execpline(...);  break;
        case WC_SIMPLE:   execcmd(...);    break;
        case WC_IF:       execif(...);     break;
        case WC_FOR:      execfor(...);    break;
        /* ... ~20 more opcodes ... */
        }
    }
    return lastval;
}
```

Each opcode handler may recurse into the executor for child blocks. The state struct `Estate` is the program counter (`pc`) plus the wordcode buffer end pointer.

### 1.4 Inspecting Wordcode

You cannot dump zsh wordcode the way you dump CPython bytecode with `dis`, but you can probe the parser:

```zsh
# show the parsed function body, normalized
which -x4 my_function
typeset -f my_function

# compile a script to a wordcode file
zcompile myscript.zsh           # produces myscript.zwc

# autoload prefers .zwc over .zsh if both exist and zwc is newer
fpath+=(~/zsh-funcs)
autoload -Uz my_helper          # picks up .zwc transparently
```

### 1.5 Trace Without Binary

The closest user-visible analogue to wordcode disassembly is `set -x` with a richer `PS4`:

```zsh
PS4='+%N:%i> '   # function name : line number
set -x
my_function
set +x
```

Output reveals which wordcode-level construct fired (a `for` opcode versus a series of `simple` calls).

---

## 2. Globbing Engine — Internals

### 2.1 The Multi-Pass Expansion Pipeline

When zsh sees a word on a command line, it runs **a fixed sequence of passes** in this order. Skipping or reordering a stage is impossible from user code — the order is hard-coded in `Src/subst.c`'s `prefork`/`postfork` machinery.

```
1.  History expansion           (!! !N !$ !* !^)        — done in lex stage
2.  Alias expansion             (alias names)            — done in lex stage
3.  Brace expansion             {a,b,c}  {1..10}  {01..99}
4.  Tilde expansion             ~  ~user  ~+  ~-  ~~
5.  Parameter expansion         $var  ${var}  ${var:-x}  with all flags
6.  Command substitution        $(cmd)  `cmd`
7.  Arithmetic expansion        $((expr))
8.  Process substitution        <(cmd)  >(cmd)            — needs MULTIOS or /dev/fd
9.  Word splitting              only if SH_WORD_SPLIT or ${=var} or (s::) flag
10. Filename generation         globbing — *, ?, [...], extended glob (#) (~) (^)
11. Quote removal               strip surviving \ ' "
```

Two consequences are non-obvious:

- **Globbing happens last**. A glob never sees the literal text the user typed — it sees the result of every prior pass. `echo $pattern/*` first substitutes `$pattern`, then globs.
- **Word splitting happens before globbing**, but in zsh splitting is OFF by default. So `echo $files` glob-matches each element of an array but does not split a string into words.

### 2.2 Why Order Matters — A Worked Example

```zsh
setopt EXTENDED_GLOB
suffix='log'
files=( ~/var/*.${suffix} )
```

Pipeline:

1. Brace: no braces, skip.
2. Tilde: `~/var` → `/home/me/var`.
3. Parameter: `${suffix}` → `log`. Word now `/home/me/var/*.log`.
4. Command sub / arith / process sub: skip.
5. Splitting: skip (default off).
6. Globbing: `*.log` matches all log files. Array filled.

Now flip step 3 vs 6 in your head — if globbing happened first, `*.${suffix}` would match nothing (no file is literally named `*.${suffix}`).

### 2.3 The Canonical Extended-Glob Power Patterns

```zsh
setopt EXTENDED_GLOB

# ^ — negation
ls ^*.log               # all files NOT ending in .log

# ~ — except (set difference)
ls *.txt~secret.txt     # all .txt EXCEPT secret.txt

# # — Kleene-like repetition (PATTERN# = zero or more, PATTERN## = one or more)
ls *(.x)#.bak           # any number of (.x) groups, then .bak

# (foo|bar) — alternation, requires KSH_GLOB or extended glob
ls (foo|bar)*.txt

# **/ — recursive descent (no setopt needed)
ls **/*.go              # all .go files at any depth
ls -d **/test/          # all directories named test, any depth

# ***/ — like **/ but follows symlinks
ls ***/log

# Pattern grouping with backreferences (zsh ≥ 4.3)
[[ $str = (#b)(*).log ]] && echo "stem=$match[1]"
```

### 2.4 Nested Glob Qualifier Evaluation Order

A glob qualifier is a parenthesized expression appended to a pattern: `*.txt(.om[1,5])`. Inside the qualifier, multiple selectors and modifiers chain left to right, and the order **matters**.

```zsh
# Top 5 most recently modified .log files, on / volume only
ls -l /var/log/*.log(.om[1,5])
#                |  |    |
#                |  |    +--> [1,5]    range: keep elements 1..5 after sort
#                |  +-------> om       order by modification time, newest first
#                +----------> .        filter: regular files only

# Reading order, left to right:
#   1. Filter to type:                  .   (regular files)
#   2. Sort:                            om  (newest first)
#   3. Slice:                           [1,5]
```

The internal pipeline is:

```
match candidates  →  type/permission filters (., /, @, =, *, %)  →
modifiers (e.g. (e:expr:))  →  sort qualifiers (oN, om, oc, oa, oL)  →
range slice [a,b]  →  final list
```

### 2.5 Proof-of-Power Patterns

```zsh
# All .pdf files modified in the last 24 hours, sorted newest first
ls /docs/**/*.pdf(.mh-24om)

# Files larger than 100 MB, owned by current user
ls **/*(.LM+100u${UID})

# Files matching glob AND a custom predicate
ls *(.e:'[[ $REPLY -nt /tmp/marker ]]':)
#         REPLY is the current candidate filename inside the predicate.

# All directories that contain a .git subdir (mark-of-repo)
print -l **/(*/.git(N/))

# Only files whose name has > 3 dots in it
ls *(#qe:'[[ ${REPLY//[^.]/} = ...* ]]':)
```

The `(#q...)` form attaches a glob qualifier to a parameter expansion result, not to a literal pattern — useful for `print -l ${files:#*tmp*}(#q.om[1])`.

---

## 3. Glob Qualifiers Catalog

Glob qualifiers chain inside `(...)` after a glob and act as selectors, sorters, modifiers, and slicers.

| Qualifier | Effect |
|:----------|:-------|
| `.` | Regular files only |
| `/` | Directories only |
| `@` | Symbolic links |
| `=` | Sockets |
| `*` | Executable plain files |
| `%` | Special files (any of `b`, `c`, `p`, `s`, …) |
| `%b`, `%c`, `%p` | Block / character / pipe device specifically |
| `r`, `w`, `x` | Owner readable / writable / executable |
| `R`, `W`, `X` | World readable / writable / executable |
| `A`, `I`, `E` | Group read / write / execute |
| `s`, `S`, `t` | Setuid / setgid / sticky bit |
| `u${UID}` | Owned by user (id or name) |
| `g${GID}` | Owned by group |
| `f<perm>` | Permission match: `f-W` = not world-writable, `f+x` = at least exec |
| `L+1m` | Size > 1 MiB (`Lk`/`Lm`/`Lg` units; `+`/`-` for greater/less) |
| `m-1`, `m+7` | Modified in the last 1 day, more than 7 days ago |
| `mh-24` | Modified in the last 24 hours (`mh`=hours, `mM`=months, `mw`=weeks) |
| `a-1`, `a+30` | Accessed within the last day, more than 30 days ago |
| `c-1` | inode-changed (ctime) within last day |
| `D` | Include dotfiles in this match (overrides default hide) |
| `N` | Null glob — silently expand to nothing if no match |
| `Y10` | Stop after the first 10 matches (early termination — fast) |
| `oN` | Order: by name (default — same as no qualifier) |
| `on` | Order: by name reversed |
| `om` | Order: by mtime, newest first |
| `Om` | Order: by mtime, oldest first |
| `oc` | Order: by ctime, newest first |
| `oa` | Order: by atime, newest first |
| `oL` | Order: by file size, largest first |
| `od` | Order: by directory depth |
| `o+func` | Order: by user-defined sort function |
| `[a,b]` | Range: keep matches `a` through `b` (1-indexed) |
| `[1]` | First match only |
| `[-1]` | Last match only |
| `e:'expr':` | Eval: keep candidate iff `expr` is true (`$REPLY` = candidate) |
| `+func` | Like `e:` but call a function |
| `:modifier:` | Apply a `:h`/`:t`/`:r`/`:e`/`:s/old/new/` modifier to each result |

### 3.1 Worked-Out Catalog Examples

```zsh
# (.r-W) — regular files that ARE owner-readable AND NOT world-writable
ls -l /etc/*(.r-W)

# (om) — sort by mtime newest first
ls -l ~/Downloads/*(om)

# (oc) — sort by ctime newest first (inode change, not file change)
ls -l /etc/*(oc)

# (oN) — explicit name sort (default; useful for clarity inside generated patterns)
ls -l /var/log/*.log(oN)

# (./) — error if no match? No — ./ means files OR directories
# To require both filter "regular file" + "directory" you can't, but:
ls -l /etc/*(.)         # only regular files
ls -d /etc/*(/)         # only directories

# (D) — include dotfiles (default zsh hides them)
ls -d ~/(*D)

# (N) — null glob, no error if nothing matches
local matches=( ~/.foo/*(N) )
(( ${#matches} == 0 )) && echo "no foo"

# (a-1) — accessed within the last day
ls -l ~/Documents/**/*(a-1)

# (m-1) — modified within the last day
ls -l ~/work/**/*(m-1)

# (L+1m) — size larger than 1 MiB
du -h **/*(.L+1m)

# Already shown but: (om) — order by modification time, newest first
ls -l /var/log/*.log(om[1,10])    # top 10 newest

# (Y10) — early termination after 10 matches (huge speedup on large dirs)
ls /very/big/dir/**/*(.Y10)
```

### 3.2 Combining Qualifiers Safely

Qualifiers chain with no separator. The implicit grouping is: filters → ordering → range → modifiers.

```zsh
# Top 5 newest log files larger than 10 KiB, owned by me, mark them with :t (tail = basename)
print -l /var/log/**/*(.L+10ku${UID}om[1,5]:t)
```

If you need an OR between filters, use the comma-glued multi-qualifier form:

```zsh
# Files that are regular OR symlinks
ls *(.,@)
```

---

## 4. ZLE — Zsh Line Editor Internals

### 4.1 The Widget Model

Everything that can edit the command line is a **widget** — a named, callable piece of editor logic. Built-in widgets implement movement, history, deletion, completion. User widgets are zsh functions registered via `zle -N`.

```
keymap (default: emacs or viins)        keystroke "^A"
   │                                       │
   ▼                                       ▼
key-binding table:  bindkey '^A' beginning-of-line
                                        │
                                        ▼
                          widget "beginning-of-line"
                                        │
                                        ▼
              [ built-in C function in Src/Zle/zle_*.c ]
                  OR
              [ user shell function registered via zle -N ]
```

### 4.2 User-Defined Widgets

```zsh
# A widget that inserts the current git branch into the buffer
my-insert-branch() {
    local branch
    branch=$(git symbolic-ref --short HEAD 2>/dev/null) || return 0
    LBUFFER+="$branch"        # LBUFFER = chars left of cursor; RBUFFER = right of cursor
}
zle -N my-insert-branch
bindkey '^G^B' my-insert-branch
```

Inside a widget:

| Variable | Meaning |
|:---------|:--------|
| `BUFFER` | The full edit buffer (read/write) |
| `LBUFFER` | Left of cursor |
| `RBUFFER` | Right of cursor |
| `CURSOR` | Cursor position (0-indexed) |
| `MARK` | Mark position |
| `NUMERIC` | Numeric prefix argument from `^U` |
| `KEYS` | The keystroke sequence that invoked this widget |
| `WIDGET` | The widget's own name |
| `LASTWIDGET` | Previous widget run |
| `PREDISPLAY`/`POSTDISPLAY` | Read-only text shown before/after the buffer |
| `REGION_ACTIVE` | 1 if a region is active |

### 4.3 Pre-Prompt Phase vs Editing Phase

ZLE has two distinct life-cycle moments:

```
[ shell main loop ]
   │
   ▼
[ precmd hooks fire ]                  ← chpwd, precmd, vcs_info, prompt computation
   │
   ▼
[ PROMPT printed to terminal ]
   │
   ▼
[ ZLE entered ]                         ← zle-line-init widget fires
   │
   ▼  user edits...
   │  zle-line-pre-redraw fires before each redraw
   │  zle-keymap-select fires on keymap change (vi-mode cursor!)
   ▼
[ user hits Enter → zle-line-finish fires ]
   │
   ▼
[ preexec hooks fire ]                 ← right before command runs
   │
   ▼
[ command runs ]
```

The **special widget names** that fire automatically:

| Widget | Fires When |
|:-------|:-----------|
| `zle-line-init` | About to start editing |
| `zle-line-finish` | About to leave editor |
| `zle-line-pre-redraw` | Before each redraw |
| `zle-keymap-select` | Keymap changed (vicmd↔viins) |
| `zle-isearch-update` / `zle-isearch-exit` | i-search transitions |

```zsh
# vi-mode cursor: block in vicmd, beam in viins
function zle-keymap-select {
    case $KEYMAP in
        vicmd)      print -n -- "\e[2 q" ;;       # block
        viins|main) print -n -- "\e[6 q" ;;       # beam
    esac
}
zle -N zle-keymap-select
```

### 4.4 The Keymap Stack

A keymap is a name → widget table. Multiple keymaps coexist:

| Keymap | Use |
|:-------|:----|
| `main` | The active keymap (alias to either `emacs` or `viins`) |
| `emacs` | Emacs bindings |
| `viins` | Vi insert mode |
| `vicmd` | Vi command (normal) mode |
| `visual` | Vi visual mode (zsh ≥ 5.0.8) |
| `viopp` | Vi operator-pending mode |
| `isearch` | Active during incremental search |
| `command` | `execute-named-cmd` overlay |

```zsh
bindkey -l                       # list all keymaps
bindkey -M viins '^E' end-of-line
bindkey -M vicmd 'j' down-history
bindkey -A emacs main            # alias main → emacs (sets default mode)

# Define your own keymap and link a chord prefix
bindkey -N mygit
bindkey -M mygit 's' git-status-widget
bindkey -M mygit 'c' git-commit-widget
bindkey '^Gg' mygit              # ^Gg s, ^Gg c …
```

### 4.5 The Multi-Line Buffer Model

ZLE stores the buffer as a single string with embedded `\n` separators. Movement widgets compute (logical-line, column) on demand from `CURSOR`. The `PREDISPLAY`/`POSTDISPLAY` strings are concatenated for display only — they do not become part of `BUFFER`.

```zsh
# Highlight syntax errors with PREDISPLAY (zsh-syntax-highlighting works this way)
zle-line-pre-redraw() {
    region_highlight=( "0 ${#BUFFER} fg=red,bold" )    # highlight whole buffer red
}
zle -N zle-line-pre-redraw
```

`region_highlight` is an array of `start end style` triples. Each redraw zsh re-applies them.

---

## 5. Parameter Expansion Flag Machinery

Zsh's `${(flags)var}` flag set rivals a small DSL. Flags chain left to right and apply in a specific order.

### 5.1 Flag Reference

| Flag | Effect |
|:-----|:-------|
| `(P)` | Treat `var` as a name reference; expand the parameter whose name is the value |
| `(e)` | Re-evaluate result as a parameter expansion (one extra round of `$` substitution) |
| `(q)`, `(qq)`, `(qqq)`, `(qqqq)`, `(q-)` | Quote: `q`=backslash, `qq`=single, `qqq`=double, `qqqq`=`$''`, `q-`=minimal POSIX |
| `(Q)` | Strip one level of quoting |
| `(k)` | For an associative array, expand keys instead of values |
| `(v)` | With `(k)`, also include values; or invert other selectors |
| `(i)` | Case-insensitive ordering for `o` flag |
| `(o)` | Sort ascending |
| `(O)` | Sort descending |
| `(n)` | Numeric sort |
| `(u)` | Uniqueify after sort |
| `(z)` | Split using shell parser rules (handles quotes — Z-shell-style word split) |
| `(L)` | Lowercase |
| `(U)` | Uppercase |
| `(C)` | Capitalize first letter of each word |
| `(s:.:)` | Split at literal `.` |
| `(j:,:)` | Join with `,` |
| `(f)` | Split at newlines (shorthand for `(ps:\n:)`) |
| `(F)` | Join with newlines |
| `(@)` | Quote-protect array elements |
| `(t)` | Type info: returns words like `array readonly`, `integer`, `scalar`, … |
| `(p)` | Recognize escape sequences inside the *separator* of `s::` / `j::` |
| `(#)` | Replace each element with its length |
| `(%)` | Apply prompt-style escapes (like in `PROMPT`) |
| `(g::)` | Process backslash escapes (zsh ≥ 5.3) |
| `(M)` | With `${var:#pat}`, *match* instead of *exclude* |
| `(B)`, `(E)`, `(N)` | With `(M)`, return Begin / End / Length of match |

### 5.2 Order of Operations Inside `${(flags)var}`

The flag list is parsed in this canonical order:

```
1. Subscripting flags         (P)
2. Type flags                 (t)
3. Splitting and joining      (s::), (j::), (f), (F), (z)
4. Sorting and uniqueing      (o), (O), (n), (i), (u)
5. Case conversion            (L), (U), (C)
6. Quoting                    (q)/(Q), (g::), (e), (%)
7. Length / replacement       (#)
```

If you write `${(US)var}` zsh applies `S` (substring search direction) before `U` (upper). Reading left-to-right is *not* the execution order — the order is fixed by the flag's category.

### 5.3 Canonical "Split, Sort, Unique" Idiom

```zsh
str='banana,apple,cherry,apple'

# Split on comma → sort ascending → unique
print -l ${(uos:,:)str}
# apple
# banana
# cherry

# Split on newline (what `(f)` is for) → reverse sort
ls /etc | print -l ${(Of)$(cat -)}
```

### 5.4 Word Splitting via `(z)`

`(z)` invokes the shell's own parser to split, honoring quotes and escapes:

```zsh
cmdline='echo "hello world" $HOME'
words=( ${(z)cmdline} )
print -l "${words[@]}"
# echo
# "hello world"
# $HOME
```

`(z)` is the right tool for emulating shell word splitting from inside a zsh script (e.g. when reading historical commands).

### 5.5 Modifier Syntax (Suffix Modifiers)

After a parameter, after a glob, after the elements of an array — you can apply csh-style modifiers:

| Modifier | Effect |
|:---------|:-------|
| `:h` | Head: like `dirname` |
| `:t` | Tail: like `basename` |
| `:r` | Remove file extension |
| `:e` | Just the file extension |
| `:l` | Lowercase |
| `:u` | Uppercase |
| `:s/old/new/` | sed-style substitution (first match only) |
| `:gs/old/new/` | sed-style substitution, global |
| `:q` / `:Q` | Quote / dequote |
| `:A` | Absolute path (resolves `..`, `.`, but **not** symlinks) |
| `:P` | Absolute path, also resolves symlinks (zsh ≥ 5.3) |

```zsh
file='/etc/foo/bar.tar.gz'
print ${file:h}             # /etc/foo
print ${file:t}             # bar.tar.gz
print ${file:r}             # /etc/foo/bar.tar
print ${file:r:r}           # /etc/foo/bar
print ${file:e}             # gz
print ${file:t:r}           # bar.tar
print ${file:gs|/etc|/usr|} # /usr/foo/bar.tar.gz

paths=(/etc /var/log /tmp/foo)
print -l ${paths:t}         # etc, log, foo

# History modifier (works on $!! style references too)
echo /etc/passwd
echo !$:h                   # /etc
```

---

## 6. Completion System Internals

### 6.1 The Compsys Bootstrap

```zsh
autoload -Uz compinit
compinit
```

What happens:

1. `compinit` checks `$fpath` for files starting with `_`.
2. For each file `_foo`, it parses the first line for `#compdef foo` directives and registers the function in the `_comps` associative array: `_comps[git]=_git`.
3. It builds the `compcontext` machinery and `_main_complete` driver.
4. It writes `$ZDOTDIR/.zcompdump` with the cached map (timestamp-keyed).

`compinit -i` skips insecure-directory checks (faster but unsafe). `compinit -C` skips the security check entirely and always uses the cache.

### 6.2 The Tag and Style System

When TAB is pressed, zsh figures out the current `compcontext`:

```
context = :completion:<function>:<command>:<argument>:<tag>
```

For example, completing the second positional argument to `git checkout`:

```
:completion:complete:git:argument-2:branches
```

Styles attach to context patterns:

```zsh
# Group results by tag
zstyle ':completion:*' group-name ''

# Show menu after first tab
zstyle ':completion:*' menu select=2

# Case-insensitive AND prefix-anchored matching
zstyle ':completion:*' matcher-list \
    'm:{a-zA-Z}={A-Za-z}' \
    'r:|[._-]=* r:|=*' \
    'l:|=* r:|=*'

# Use ls-like coloring for filename completion
zstyle ':completion:*:default' list-colors ${(s.:.)LS_COLORS}

# Cache slow completions
zstyle ':completion:*' use-cache yes
zstyle ':completion:*' cache-path "$HOME/.cache/zsh/compcache"
```

Every `_` function consults `zstyle` to learn what the user wants for its tag.

### 6.3 How `_files` / `_command` / `_arguments` Compose

```
_arguments       — handles flags, positional args; calls back to _files / _hosts / etc.
   │
   ├─→ _files   — generic file completion with type filters (-/, -. , -g)
   ├─→ _command — completes any command name with proper PATH search
   ├─→ _hosts   — pulls from ~/.ssh/known_hosts, /etc/hosts
   └─→ _users   — pulls from /etc/passwd or NSS
```

A typical handwritten `_foo`:

```zsh
#compdef foo

_foo() {
    _arguments -s -S \
        '(-h --help)'{-h,--help}'[show help]' \
        '-v[verbose]' \
        '--config=[config file]:config:_files -g "*.conf"' \
        '1: :_foo_subcommands' \
        '*::: :->args'

    case $state in
      args)
        case $words[1] in
          add)    _arguments '*:filename:_files' ;;
          remove) _arguments '*:installed:_foo_installed_packages' ;;
        esac
      ;;
    esac
}

_foo_subcommands() {
    local -a subs=(add:'add a thing' remove:'remove a thing' list:'list things')
    _describe -t commands 'foo subcommand' subs
}
```

### 6.4 Matchspec Rules (`matcher-list`)

The matchspec mini-language tells zsh how to map typed text to candidate text:

| Spec | Meaning |
|:-----|:--------|
| `m:<from>=<to>` | Match: any `from` char on input matches a `to` char on the candidate (both directions if symmetric) |
| `M:<from>=<to>` | Match-from-anywhere variant |
| `l:<lanchor>\|<remain>=<from>` | Left-anchor at `lanchor`, then match `from` to remain |
| `r:<remain>=<from>\|<ranchor>` | Right-anchor at `ranchor` |
| `b:<from>=<to>` | Match initial substring (begin) |
| `e:<from>=<to>` | Match final substring (end) |
| `B:<from>=<to>` | Match the whole word at the beginning |
| `E:<from>=<to>` | Match the whole word at the end |

The most-used three-tier setup:

```zsh
zstyle ':completion:*' matcher-list \
    'm:{a-zA-Z}={A-Za-z}' \              # case-insensitive
    'r:|[._-]=* r:|=*' \                 # break on _ - . — fzm-style
    'l:|=* r:|=*'                        # progressive partial-word
```

Tier 1 fires first; if no candidates result, zsh falls back to tier 2; then tier 3. Each tier widens the match.

### 6.5 The `:completion:*` Style Tree — Common Recipes

```zsh
# Show descriptions
zstyle ':completion:*:descriptions' format '%F{yellow}%d%f'

# Group same-tag results
zstyle ':completion:*' group-name ''

# Color the completion menu like ls
zstyle ':completion:*:default' list-colors ${(s.:.)LS_COLORS}

# Squeeze multiple slashes
zstyle ':completion:*' squeeze-slashes true

# Always use menu (interactive selection)
zstyle ':completion:*' menu select

# Approximate matching: allow 1 typo per ~7 chars
zstyle ':completion:*' completer _expand _complete _approximate
zstyle ':completion:*:approximate:*' max-errors 'reply=( $((($#PREFIX+$#SUFFIX)/7)) numeric )'

# Hide private and double-underscore zsh internals
zstyle ':completion:*:functions' ignored-patterns '_*'
```

---

## 7. Hooks and Trap Mechanism

### 7.1 The Hook Functions

Zsh's hooks fire at specific lifecycle points. Each hook is implemented by a **function array**: `precmd_functions`, `preexec_functions`, `chpwd_functions`, `zshaddhistory_functions`, `zshexit_functions`, `periodic_functions`.

| Hook | Fires |
|:-----|:------|
| `precmd` | Just before the prompt is drawn (every prompt) |
| `preexec` | After Enter, just before the command runs |
| `chpwd` | After `cd` changes `$PWD` |
| `zshaddhistory` | When a new line is about to enter history (return non-zero to suppress) |
| `periodic` | Every `$PERIOD` seconds, before next prompt |
| `zshexit` | On shell exit |

The recommended way to register is via `add-zsh-hook`:

```zsh
autoload -Uz add-zsh-hook

my_precmd() { [[ -d .git ]] && vcs_info }
add-zsh-hook precmd my_precmd

my_preexec() { _start=$EPOCHREALTIME }
add-zsh-hook preexec my_preexec

my_chpwd() { ls --color }
add-zsh-hook chpwd my_chpwd
```

`add-zsh-hook` deduplicates and supports `add-zsh-hook -d` to remove.

### 7.2 Difference From Bash's `PROMPT_COMMAND`

| | Bash | Zsh |
|:--|:-----|:----|
| Pre-prompt hook | `PROMPT_COMMAND` (string, eval'd) | `precmd_functions` (array of names) |
| Post-cd hook | manual via `cd ()` override | `chpwd_functions` (array) |
| Pre-exec hook | DEBUG trap (with `extdebug`) | `preexec_functions` (array) |
| Multiple hooks | last writer wins, unless you concatenate | array — additive by default |
| Hook receives the command? | DEBUG trap: `BASH_COMMAND` | preexec: `$1`=raw, `$2`=expanded, `$3`=full |

### 7.3 Trap Mechanism

`trap` covers signals plus three pseudo-signals: `EXIT`, `DEBUG`, `ZERR`.

```zsh
# Cleanup on exit
trap 'rm -rf $tmpdir' EXIT

# Run on every command (like bash DEBUG)
trap 'echo "about to run: $ZSH_DEBUG_CMD"' DEBUG

# Run after every command that returns non-zero
trap 'echo "command failed, status=$?"' ZERR

# Standard signals
trap 'echo got SIGINT' INT
trap '' QUIT          # ignore SIGQUIT
trap - INT            # restore default
```

`TRAPSIG_function` form (preferred for non-EXIT traps because it preserves scope):

```zsh
TRAPINT() { print "interrupted"; return 130 }
TRAPUSR1() { print "got USR1" }
```

The function form is local-friendly and stack-safe.

---

## 8. Job Control Internals

### 8.1 Process Groups and the Foreground TTY

A **process group** (pgrp) is a kernel-tracked set of processes that share a pgid. Job control means:

- Each pipeline launched from the shell becomes its own pgrp via `setpgid()` immediately after fork.
- Exactly one pgrp owns the controlling terminal at a time. The shell calls `tcsetpgrp(0, pgid)` to grant ownership.
- Signals from the keyboard (`Ctrl-C` → `SIGINT`, `Ctrl-Z` → `SIGTSTP`) go to the **foreground** pgrp.

```
[ shell process pgid=100 owns tty ]
   │
   │ launches pipeline:  cmd1 | cmd2
   ▼
[ fork → pgid=200, exec cmd1 ]
[ fork → pgid=200, exec cmd2 ]    (same pgid as cmd1)
   │
   │ shell calls tcsetpgrp(0, 200)
   ▼
[ pgrp 200 now owns tty;        ]
[ Ctrl-C in tty → SIGINT to 200 ]
   │
   │ pipeline exits
   ▼
[ shell calls tcsetpgrp(0, 100) ]
[ shell back in foreground       ]
```

### 8.2 The Jobs Table

Internally zsh keeps a `Job` array (`Src/jobs.c`):

```c
struct job {
    pid_t gleader;          /* group leader pid       */
    pid_t other_pids[];     /* members of pgrp        */
    int   stat;             /* STAT_DONE | STAT_STOPPED | STAT_INUSE | ... */
    char *pwd;              /* directory job was started in */
    Process procs;          /* linked list of struct process */
};
```

`jobs` walks this table and prints a human view. `%1`, `%2`, … are job numbers indexing it. `%cmd` matches by command-name prefix; `%?str` matches anywhere in the command line.

### 8.3 Job-Control Signals

| Signal | Sender | Effect |
|:-------|:-------|:-------|
| `SIGCHLD` | kernel → parent | child exited / stopped / continued |
| `SIGTSTP` | tty → fg pgrp | suspend (Ctrl-Z) |
| `SIGSTOP` | anything | unconditional suspend (uncatchable) |
| `SIGCONT` | shell | resume a stopped pgrp |
| `SIGTTIN` | kernel | bg pgrp tried to read tty |
| `SIGTTOU` | kernel | bg pgrp tried to write tty (when `stty tostop`) |

Zsh installs a `SIGCHLD` handler that calls `wait4()` on every change and updates the jobs table. The state transition table:

```
Running    --(SIGTSTP)--> Stopped     (printed: [1]+ Stopped)
Stopped    --(SIGCONT)--> Running     (printed: [1]+ Running)
Running    --(exit)----->  Done        (printed: [1]+ Done)
Stopped    --(SIGKILL)--> Done w/sig   (printed: [1]+ Killed)
```

### 8.4 The Canonical "Wait for Any" via `wait -n`

```zsh
# Launch N jobs, then collect them as they finish (any order)
for url in $urls; do
    fetch "$url" &
done

# wait -n unblocks when any one child exits
while (( ${#${(k)jobstates}} )); do
    wait -n
    print "one job done — $? — $#${(k)jobstates} remaining"
done
```

`wait -n` (zsh ≥ 5.1, also bash ≥ 4.3) returns when any (still-running) child exits and sets `$?` to its exit status. Without `-n`, `wait` (no args) waits for all of them.

### 8.5 `disown` and `nohup`

```zsh
sleep 9999 &
disown %1               # remove from jobs table; survives shell exit
nohup long_job &        # disowns AND ignores SIGHUP
```

`disown -h %1` keeps the job in the table but marks it to ignore `SIGHUP` only on shell exit.

---

## 9. History Mechanics

### 9.1 The HISTFILE Write Pipeline

```
[ user hits Enter ]
   │
   ▼
[ zshaddhistory hooks fire — can suppress ]
   │
   ▼
[ entry appended to in-memory $HISTORY array ]
   │
   ▼  (depends on options)
   │
   ┌─→ INC_APPEND_HISTORY ON  → write THIS line to $HISTFILE now
   │
   └─→ INC_APPEND_HISTORY OFF → write deferred until shell exit (default)
                                  ... unless SHARE_HISTORY is on:
   ┌─→ SHARE_HISTORY ON       → write now, AND re-read $HISTFILE before
   │                              every prompt (so other sessions are visible)
```

### 9.2 `SHARE_HISTORY` vs `INC_APPEND_HISTORY`

| Option | Write Now | Re-read Before Prompt | Use Case |
|:-------|:---:|:---:|:--------|
| (none — default) | No | No | Lone interactive session |
| `INC_APPEND_HISTORY` | Yes | No | Append immediately, don't share |
| `SHARE_HISTORY` | Yes | Yes | Multiple terminals stay in sync |
| `INC_APPEND_HISTORY_TIME` (zsh ≥ 5.1) | Yes (with time) | No | Like INC_APPEND but writes only after command finishes |

```zsh
setopt SHARE_HISTORY            # all terminals share history live
setopt EXTENDED_HISTORY         # write timestamp + duration
setopt HIST_IGNORE_ALL_DUPS     # remove old dup before adding new
setopt HIST_IGNORE_SPACE        # ignore lines starting with space
setopt HIST_REDUCE_BLANKS       # collapse runs of blanks
setopt HIST_VERIFY              # !! expansion shows result before run
HISTFILE=$HOME/.zsh_history
HISTSIZE=200000                 # in-memory
SAVEHIST=200000                 # on-disk
```

### 9.3 EXTENDED_HISTORY Format

With `setopt EXTENDED_HISTORY`, lines look like:

```
: 1714150800:0;ls -la
: 1714150803:1;rm /tmp/foo
^ ^          ^ ^
| |          | command (may be multi-line via \n escapes)
| |          duration in seconds
| start time (epoch)
literal `: ` marker (so non-zsh shells can ignore the line)
```

### 9.4 History Substitution

| Token | Meaning |
|:------|:--------|
| `!!` | The previous command |
| `!N` | Command number N |
| `!-N` | Nth previous command |
| `!str` | Most recent command starting with `str` |
| `!?str` | Most recent command containing `str` |
| `^a^b^` | Replace `a` with `b` in previous command, run it |
| `!$` | Last word of previous command |
| `!^` | First argument of previous command |
| `!*` | All arguments of previous command |
| `!:N` | Nth word (0-indexed; `!:0` = command name) |
| `!:N-M` | Words N through M |
| `!!:s/a/b/` | Substitute a → b (first match) |
| `!!:gs/a/b/` | Substitute globally |
| `!!:&` | Repeat last `s/// ` substitution |

```zsh
ls /etc/passwd
cat !$                    # cat /etc/passwd

mkdir /tmp/proj
cd !$                     # cd /tmp/proj

cp file1 file2
^cp^mv                    # mv file1 file2
```

### 9.5 The `fc` Command — In-Editor History Editing

```zsh
fc                        # edit last command in $EDITOR, run on save
fc -l                     # list last 16
fc -l 100 110             # list 100–110
fc -e vim 100 105         # edit lines 100–105 in vim, run all
fc -s old=new             # repeat last command with substitution
fc -p                     # push current history (private session)
```

`fc` writes the selected lines to a temp file, opens `$EDITOR`, and on close re-reads and submits. Empty buffer = abort.

---

## 10. Modules and Loadable Builtins

### 10.1 The Module System

Zsh ships as a small core plus loadable modules. `zmodload` ties them in at runtime.

```zsh
zmodload                # list loaded modules
zmodload -L             # list with full paths
zmodload zsh/datetime   # load by name
zmodload -u zsh/regex   # unload
zmodload -F zsh/parameter +p:userdirs   # load only one feature
```

### 10.2 Useful Modules

| Module | Provides |
|:-------|:---------|
| `zsh/datetime` | `strftime` builtin, `$EPOCHSECONDS`, `$EPOCHREALTIME` |
| `zsh/mathfunc` | `sqrt`, `log`, `exp`, `sin`, `rand48`, … math functions in `(( ))` |
| `zsh/regex` | POSIX `=~` regex match, `$MATCH`, `$BEGIN`, `$END` |
| `zsh/system` | `syserror`, `sysopen`, `sysread`, `sysseek`, `syswrite`, `zsystem` |
| `zsh/zselect` | `zselect` builtin — non-blocking I/O multiplexing |
| `zsh/zutil` | `zstyle`, `zparseopts`, `zformat`, `zregexparse` |
| `zsh/parameter` | Special params: `$jobstates`, `$funcstack`, `$nameddirs`, `$dirstack` |
| `zsh/computil` | Internal completion helpers (used by `_arguments`, `_describe`) |
| `zsh/complist` | Menu selection, list colors |
| `zsh/zftp` | Built-in FTP client |
| `zsh/zpty` | Pseudo-terminal management |
| `zsh/net/tcp` | TCP socket primitives |
| `zsh/sched` | `sched` builtin — schedule events |
| `zsh/zle` | The line editor itself (autoloaded) |
| `zsh/curses` | `zcurses` builtin — full curses bindings |

### 10.3 `zsh/datetime` — Microsecond Timing

```zsh
zmodload zsh/datetime

print $EPOCHSECONDS            # 1714150800
print $EPOCHREALTIME           # 1714150800.123456 (microsecond)

# strftime — format epoch to string, OR parse string to epoch
strftime '%Y-%m-%d %H:%M:%S' $EPOCHSECONDS
strftime -r -s parsed '%Y-%m-%d' '2026-04-25'
print $parsed

# Microsecond benchmarking
local t0=$EPOCHREALTIME
slow_thing
local t1=$EPOCHREALTIME
printf 'slow_thing took %.3f ms\n' $(( (t1 - t0) * 1000 ))
```

### 10.4 `zsh/mathfunc` — Math Functions in `(( ))`

```zsh
zmodload zsh/mathfunc

print $(( sqrt(2) ))           # 1.4142135623730951
print $(( log(10) ))           # 2.302585092994046
print $(( sin(3.14159/2) ))    # 0.99999...

# rand48: RNG suitable for non-cryptographic use, in [0,1)
print $(( rand48() ))

# zsh/random for /dev/urandom-backed integers
zmodload -F zsh/system b:zsystem
RANDOM=$EPOCHREALTIME           # seed from time
print $(( RANDOM % 100 ))
```

### 10.5 `zsh/regex` — POSIX Regex

```zsh
zmodload zsh/regex

if [[ "hello world 42" =~ '([a-z]+) ([a-z]+) ([0-9]+)' ]]; then
    print "match[0]=$MATCH"          # whole match
    print "match[1]=$match[1]"       # first group
    print "begin=$BEGIN end=$END"
fi
```

### 10.6 `zsh/zutil` — `zparseopts`

`zparseopts` is the right way to parse args inside a zsh function (don't shell out to `getopt`):

```zsh
zmodload zsh/zutil

myfunc() {
    local -a verbose force config
    zparseopts -D -E -F -- \
        v=verbose -verbose=verbose \
        f=force   -force=force \
        c:=config -config:=config

    print "verbose: ${#verbose}"
    print "config: $config[2]"
    print "remaining: $@"
}

myfunc -v --config=/etc/foo.conf positional1 positional2
```

`-D` = remove parsed options from `$@`; `-E` = stop at first non-option; `-F` = error on unknown.

---

## 11. Performance — `zsh -xv` and `zprof`

### 11.1 `zsh -xv` Trace

```zsh
zsh -xv -i -c exit 2>&1 | head -200      # trace startup, show first 200 lines
zsh -xv -i -c exit 2> /tmp/zsh.trace
```

`-x` = expand+trace each command before executing. `-v` = print each line of input as it is read. Together they show what `.zshrc` is loading, in order.

`PS4` controls the trace prefix. The most informative form:

```zsh
PS4=$'%D{%H:%M:%S.%.} %N:%i> '
zsh -xv -i -c exit 2> /tmp/zsh.trace
```

Now each line carries timestamp, file, and line number.

### 11.2 `zprof` — Function-Level Profiler

```zsh
# at the very top of .zshrc
zmodload zsh/zprof

# ...rest of .zshrc...

# at the bottom of .zshrc (or run interactively after sourcing)
zprof
```

Output:

```
num  calls                time                       self            name
-----------------------------------------------------------------------------------
 1)    1          85.34   85.34   42.5%      85.34   85.34   42.5%  compinit
 2)    1          45.11   45.11   22.4%      45.11   45.11   22.4%  nvm.sh
 3)    1          22.40   22.40   11.1%      22.40   22.40   11.1%  pyenv-init
 4)    7          18.10    2.59    9.0%      18.10    2.59    9.0%  __git_ps1
 ...
```

The `time` column is total ms (including children). `self` is exclusive (function body, not children). Look for big `self` numbers — those are the functions to optimize.

### 11.3 Bottleneck-Finding Approach

```zsh
# 1. Baseline: cold-start
time zsh -i -c exit
# real 0.420s   ← terrible

# 2. Bisect with -d (skip global zshenv) and -f (no rcs)
time zsh -fi -c exit         # ~0.01s — confirms it's our config
time zsh -di -c exit         # skip /etc files

# 3. Profile
zsh -i -c 'zmodload zsh/zprof; source ~/.zshrc; zprof' | head -30

# 4. Trace a specific section
PS4=$'+ %D{%S.%.} %N:%i> '
TIMEFMT=$'\nuser %U\nsys %S\nreal %E'
{ time my_slow_section } 2>&1 | tail -5
```

### 11.4 Common Startup Costs (and Fixes)

| Cost | Typical | Fix |
|:-----|:-------:|:----|
| `compinit` | 50–150 ms | Rebuild `.zcompdump` only once a day; use `compinit -C` on cached path |
| `nvm.sh` | 100–500 ms | Lazy-load on first `node` / `npm` call |
| `pyenv init` | 50–200 ms | Cache output: `eval "$(pyenv init -)"` once, persist |
| `direnv hook` | 5–10 ms | Negligible, leave |
| `git_prompt` | 10–50 ms / prompt | Async via `vcs_info` + zsh-async |
| `oh-my-zsh` core | 100–300 ms | Replace with curated `.zshrc` |

The 100ms-startup target is achievable on a modern laptop with disciplined `.zshrc`.

### 11.5 Lazy-Loading Pattern

```zsh
# Stub: defer nvm until first use
nvm() {
    unset -f nvm
    export NVM_DIR="$HOME/.nvm"
    source "$NVM_DIR/nvm.sh"
    nvm "$@"
}

# Same trick for pyenv
pyenv() {
    unset -f pyenv
    eval "$(command pyenv init -)"
    pyenv "$@"
}
```

First `nvm install 20` pays the load cost; every subsequent shell start is free.

### 11.6 The `compinit` Cache Trick

```zsh
autoload -Uz compinit
# Rebuild zcompdump only if older than 24h
if [[ -n ~/.zcompdump(#qN.mh+24) ]]; then
    compinit
else
    compinit -C        # skip security check, use cache
fi
```

The glob `(#qN.mh+24)` is a glob qualifier: null-glob (`N`), regular file (`.`), modified more than 24 hours ago (`mh+24`). If it matches (= file exists and is old), rebuild. Otherwise use the cache.

---

## 12. Differences From Bash at the Internals Level

### 12.1 No Word-Splitting by Default

```bash
# Bash:
files="a.txt b.txt"
for f in $files; do print $f; done   # 2 iterations
```

```zsh
# Zsh:
files="a.txt b.txt"
for f in $files; do print $f; done   # 1 iteration
for f in ${=files}; do print $f; done # 2 iterations — explicit split
```

### 12.2 `${~var}` — Explicit Glob Expansion

```zsh
pat='*.log'
print $pat       # literal '*.log'
print ${~pat}    # expanded — globs the cwd
```

By default, parameter expansion does *not* re-trigger globbing. `${~var}` opts in.

### 12.3 `${(z)var}` — Z-Shell Word Split

Already covered in §5.4. The point is: bash has no equivalent — `read -ra arr <<<"$cmd"` does not handle quotes correctly. `(z)` does.

### 12.4 `FUNCTION_ARGZERO`

```zsh
# In a function called `foo` invoked as `foo bar baz`:
print $0       # "foo" if FUNCTION_ARGZERO (default), else "$0" of script

unsetopt FUNCTION_ARGZERO
print $0       # script name
```

This option drives whether `$0` inside a function refers to the function name or the script.

### 12.5 `&&` / `||` in Conditionals

Zsh accepts both styles; arithmetic-style `((...))` works in both:

```zsh
# C-style
if (( x > 5 && y < 10 )); then ...; fi

# String-style with [[ ]]
if [[ -f $f && -r $f ]]; then ...; fi

# In zsh, [[ ]] supports more operators than bash:
[[ $a -ot $b ]]                  # a older than b
[[ $a -nt $b ]]                  # a newer than b
[[ $f = (#i)*.LOG ]]             # case-insensitive glob match (extended glob)
[[ $s -pcre-match $regex ]]      # PCRE if zsh/pcre loaded
```

### 12.6 `$((...))` vs `((...))` — Two Arithmetic Contexts

```zsh
# $((expr)) — substitutes the value
print $(( 2 + 3 ))               # prints 5

# ((expr)) — evaluates, returns shell-style status (0=true=non-zero, 1=false=zero)
((x = 5))
((x > 0)) && print 'positive'
```

In `((...))` you don't need `$` on variable names (and you should not use `$` — assignments are direct).

### 12.7 Arrays — 1-Indexed

```zsh
arr=(a b c d)
print $arr[1]       # a   ← 1-indexed!
print $arr[-1]      # d
print $arr[2,3]     # b c
print ${#arr}       # 4

# Bash equivalent:  echo ${arr[0]} (0-indexed)
```

`KSH_ARRAYS` option flips this: `setopt KSH_ARRAYS` makes `$arr[0]` the first element (for compatibility scripts).

### 12.8 Associative Arrays

```zsh
typeset -A colors
colors=( red 1  green 2  blue 3 )
colors[purple]=4

print $colors[red]              # 1
print ${(k)colors}              # red green blue purple
print ${(v)colors}              # 1 2 3 4
print ${(kv)colors}             # red 1 green 2 blue 3 purple 4

for k v in ${(kv)colors}; do print "$k=$v"; done
```

### 12.9 `print` vs `echo`

```zsh
print -l a b c            # one per line
print -r -- "$str"        # raw, no interpretation, no leading dash issues
print -P '%F{red}hi%f'    # apply prompt-style escapes
print -n -- "no newline"  # like echo -n
```

`print` is built-in and predictable; `echo`'s handling of `-n` and `-e` varies even between zsh and bash. Zsh-portable scripts use `print -r -- "$x"`.

---

## 13. The Prompt Engine

### 13.1 PROMPT, RPROMPT, PROMPT2

```
PROMPT       primary prompt (left)
RPROMPT      right prompt — auto-hides when typing
PROMPT2      continuation prompt (after backslash, unclosed quote, …)
PROMPT3      select-loop prompt
PROMPT4      xtrace prefix (PS4)
```

### 13.2 Prompt Escape Sequences

| Escape | Meaning |
|:-------|:--------|
| `%n` | Username |
| `%m` | Short hostname |
| `%M` | FQDN hostname |
| `%~` | `$PWD` with `~` substitution |
| `%/` | `$PWD` literal |
| `%C` | Trailing component of `$PWD` |
| `%c`, `%.<n>c` | Trailing `<n>` components |
| `%T`, `%t` | 24h / 12h time |
| `%D`, `%D{fmt}` | Date / strftime-format |
| `%?` | Last command's exit status |
| `%!` | History event number |
| `%#` | `%` if non-root, `#` if root |
| `%(?.X.Y)` | Conditional: X if `$? == 0`, else Y |
| `%(!.X.Y)` | Conditional: root vs not |

### 13.3 Color and Style

| Escape | Effect |
|:-------|:-------|
| `%F{color}` … `%f` | Foreground color region |
| `%K{color}` … `%k` | Background color region |
| `%B` … `%b` | Bold |
| `%U` … `%u` | Underline |
| `%S` … `%s` | Standout (reverse video) |
| `%{...%}` | Literal terminal escape — *not counted in width* |

Colors accept names (`red`, `green`, `blue`, `cyan`, `magenta`, `yellow`, `black`, `white`) or 256-color codes (`%F{208}`).

```zsh
# Two-line prompt with color and exit-code marker
PROMPT='%F{cyan}%n@%m%f %F{yellow}%~%f %(?.%F{green}.%F{red})%#%f '
RPROMPT='%F{240}%D{%H:%M:%S}%f'
```

### 13.4 Width-Counting Rules

Zsh counts visible width to know how much to truncate (`%<...<` and `%>...>`). Anything inside `%{...%}` is treated as zero-width — so terminal escapes you embed manually must be wrapped:

```zsh
# WRONG — zsh counts the escape codes as visible width
PROMPT=$'\e[31m%n\e[0m> '

# RIGHT
PROMPT='%{$\e[31m%}%n%{$\e[0m%}> '

# RIGHTEST — use the prompt-system %F/%f form
PROMPT='%F{red}%n%f> '
```

### 13.5 Truncation

```zsh
# Truncate $PWD to 30 chars from the left, with leading "..." marker
PROMPT='%30<...<%~%<<> '
```

Form: `%<len><truncstring><field>%<<` where `<` truncates left side, `>` truncates right side.

### 13.6 `prompt_subst` — Variables in Prompts

By default, `$variable` inside `PROMPT` is **not** expanded. Enable substitution:

```zsh
setopt PROMPT_SUBST
PROMPT='${branch}> '

precmd() {
    branch=$(git symbolic-ref --short HEAD 2>/dev/null) || branch=''
}
```

Without `PROMPT_SUBST`, the literal `${branch}` would appear.

### 13.7 `vcs_info` — VCS in Prompt

```zsh
autoload -Uz vcs_info
zstyle ':vcs_info:*' enable git hg
zstyle ':vcs_info:*' formats '%F{green}(%b)%f'
zstyle ':vcs_info:*' actionformats '%F{red}(%b|%a)%f'

precmd() { vcs_info }
setopt PROMPT_SUBST
PROMPT='%n@%m %~ ${vcs_info_msg_0_}%# '
```

`vcs_info` shells out to `git` (or `hg`) once per prompt — that cost is real on huge repos. Async patterns:

```zsh
# Render prompt without VCS first; fill in async
async_init() {
    typeset -g vcs_info_msg_0_=''
    autoload -Uz async && async
    async_start_worker prompt_worker -n
    async_register_callback prompt_worker prompt_callback
}
prompt_callback() {
    vcs_info_msg_0_=$3
    zle reset-prompt
}
precmd() {
    async_job prompt_worker compute_branch
}
```

### 13.8 The Cost Model

Every prompt evaluation runs:

1. `precmd_functions` (each function in array)
2. Variable expansion in `PROMPT` (with `PROMPT_SUBST`)
3. Terminal write via `tputs`-like sequences

Anything in `precmd` is on the **hot path**. A 200ms `git status` in `precmd` makes every prompt 200ms slow.

Rule of thumb: keep `precmd` total under 5ms. Move slower work to async workers and trigger `zle reset-prompt` on completion.

---

## 14. Plugin Manager Internals

### 14.1 The Lazy-Deferred-vs-Eager-Load Tradeoff

Three loading strategies:

| Strategy | Startup Cost | First-Use Cost | Memory | Problem |
|:---------|:-------------|:---------------|:-------|:--------|
| Eager (oh-my-zsh) | High — everything loads before prompt | Zero | High | 200-500ms startup |
| Deferred (zsh-defer / antidote) | Low | None visible | Medium | Background load steals CPU |
| Turbo (zinit ice "wait") | Lowest | None (loaded by fixed delay) | Medium | Race conditions if dependent plugin needed first |
| Lazy (manual function stub) | Zero | Real cost on first call | Lowest | Stub has to mimic real interface |

### 14.2 Zinit Turbo Mode

Zinit's "turbo" works by registering plugins to a priority queue, then loading them via `zsh-defer` or its built-in `zinit-load-async` after the prompt is ready.

```zsh
source ~/.local/share/zinit/zinit.git/zinit.zsh

# Eagerly load (highest priority) — needed for prompt itself
zinit ice depth=1
zinit light romkatv/powerlevel10k

# Turbo: load 0s after prompt is ready, in lucid (no install messages) mode
zinit wait lucid for \
    zsh-users/zsh-syntax-highlighting \
    zsh-users/zsh-autosuggestions \
    zsh-users/zsh-completions

# Turbo with delay
zinit wait'1' lucid for \
    Aloxaf/fzf-tab

# Turbo with trigger (load on first invocation of subcommand)
zinit ice wait lucid trigger-load'!gh'
zinit light cli/cli
```

The `ice` modifiers customize how a single `zinit light/load` call behaves: `wait`, `trigger-load`, `as`, `from`, `pick`, `mv`, `atclone`, `atpull`, `atinit`, `atload`, `make`, `nocompletions`, …

### 14.3 Antidote and `zsh-defer`

Antidote uses `zsh-defer` (a separate widget-based deferred-loader):

```zsh
# .zsh_plugins.txt
zsh-users/zsh-syntax-highlighting kind:defer
zsh-users/zsh-autosuggestions kind:defer
zsh-users/zsh-history-substring-search kind:defer

# .zshrc
source ~/.antidote/antidote.zsh
antidote load
```

`zsh-defer` schedules `source` calls to run **after** the first prompt is shown via the `precmd` hook. Visible startup time = ~time to read `.zshrc` + parse plugins file.

### 14.4 Cache Invalidation Problem

Both zinit and antidote build large `zcompdump`-like caches: the merged plugin source, the merged `fpath`, the prebuilt completion table. When a plugin is updated, the cache is stale.

Mitigations:

```zsh
# Antidote: rebuild the static plugin file
antidote bundle <.zsh_plugins.txt >.zsh_plugins.zsh
# .zshrc just sources the prebuilt file — fastest possible
source ~/.zsh_plugins.zsh

# Zinit: cdupdate triggers atpull hooks
zinit update --all
```

The general rule: rebuild caches whenever you `git pull` plugins; otherwise expect mysterious "this widget doesn't exist" errors.

### 14.5 Other Players

- **zplug** — older, slower, dependency tracking via Bundler-style file.
- **zgenom** — fork of zgen with auto-update; very fast cold start.
- **sheldon** — Rust-implemented; TOML config; static plugin file.
- **pure** Zsh — for the minimalist: just `source` files in order, no manager.

---

## 15. Idioms at the Internals Depth

### 15.1 The Canonical Fast `.zshrc`

```zsh
# ~/.zshrc — target: <100ms cold start

# 1. Profile only when explicitly asked
[[ -n $ZSH_PROFILE ]] && zmodload zsh/zprof

# 2. Path setup before anything else
typeset -gU path PATH                    # unique entries
path=(
    $HOME/bin
    $HOME/.local/bin
    /usr/local/bin
    /usr/bin
    /bin
)

# 3. Options
setopt AUTO_CD AUTO_PUSHD PUSHD_IGNORE_DUPS PUSHD_SILENT
setopt EXTENDED_GLOB GLOB_DOTS NUMERIC_GLOB_SORT
setopt INTERACTIVE_COMMENTS
setopt SHARE_HISTORY EXTENDED_HISTORY HIST_IGNORE_ALL_DUPS HIST_IGNORE_SPACE
setopt PROMPT_SUBST
setopt NO_BEEP
HISTFILE=$HOME/.zsh_history
HISTSIZE=200000
SAVEHIST=200000

# 4. Compinit, with cache trick
autoload -Uz compinit
if [[ -n $HOME/.zcompdump(#qN.mh+24) ]]; then
    compinit
    touch $HOME/.zcompdump
else
    compinit -C
fi

# 5. Hooks via add-zsh-hook (not direct overrides)
autoload -Uz add-zsh-hook

# 6. Plugin manager — turbo mode for everything non-prompt
source $HOME/.local/share/zinit/zinit.git/zinit.zsh
zinit ice depth=1
zinit light romkatv/powerlevel10k
zinit wait lucid for \
    zsh-users/zsh-syntax-highlighting \
    zsh-users/zsh-autosuggestions \
    zsh-users/zsh-completions

# 7. Lazy loaders
nvm()   { unset -f nvm;   source $HOME/.nvm/nvm.sh; nvm "$@" }
pyenv() { unset -f pyenv; eval "$(command pyenv init -)"; pyenv "$@" }

# 8. Aliases (cheap, eager)
alias ll='ls -lah'
alias ..='cd ..'
alias gs='git status'

# 9. Local overrides
[[ -f $HOME/.zshrc.local ]] && source $HOME/.zshrc.local

# 10. Profile output (if requested)
[[ -n $ZSH_PROFILE ]] && zprof
```

### 15.2 Right-Prompt with Color

```zsh
# A right prompt that's only visible while idle
RPROMPT='%(?..%F{red}(%?%)%f )%F{240}%D{%H:%M:%S}%f'
#         ^^^                  ^^^
#         non-zero exit code   timestamp
#         shown in red         in dim gray
```

### 15.3 Mode-Aware Vi Prompt

```zsh
# Show vi mode in the prompt
function zle-keymap-select zle-line-init {
    case $KEYMAP in
        vicmd)      VI_MODE='%F{red}[N]%f' ;;
        viins|main) VI_MODE='%F{green}[I]%f' ;;
    esac
    zle reset-prompt
}
zle -N zle-line-init
zle -N zle-keymap-select

PROMPT='${VI_MODE} %F{cyan}%~%f %# '
bindkey -v
```

### 15.4 `cmdcache` and `zcompcache`

```zsh
# Use a cache directory for all completion caches
zstyle ':completion:*' use-cache true
zstyle ':completion:*' cache-path "$XDG_CACHE_HOME/zsh"

# rehash after install — but lazily
zstyle ':completion:*' rehash true
```

### 15.5 `select` Loops With Custom Prompt

```zsh
PS3='choose: '
select fruit in apple banana cherry; do
    [[ -n $fruit ]] && { print "you chose $fruit"; break; }
done
```

`PS3` is the select-loop prompt — independent of `PROMPT`.

### 15.6 The Fast Glob Print

```zsh
# Print the 5 newest entries in the current directory
print -l *(om[1,5])

# Print all empty files
print -l **/*(.L0)

# Print all files modified more recently than my marker
touch /tmp/marker
sleep 1
echo content > newfile.txt
print -l **/*(.e:'[[ $REPLY -nt /tmp/marker ]]':)
```

### 15.7 `zparseopts` for a Real Function

```zsh
# Greppable, parameterized function
zmodload zsh/zutil

mygrep() {
    local -a opt_color opt_recursive opt_after
    zparseopts -D -E -F -- \
        c=opt_color   -color=opt_color \
        r=opt_recursive -recursive=opt_recursive \
        A:=opt_after  -after:=opt_after \
      || { print 'usage: mygrep [-c] [-r] [-A N] PATTERN [FILE...]' >&2; return 2 }

    local pattern=$1; shift
    grep ${opt_color:+--color=auto} \
         ${opt_recursive:+-r}        \
         ${opt_after:+-A $opt_after[2]} \
         -- "$pattern" "$@"
}
```

### 15.8 Async Pattern with `zsh-async`

```zsh
source ~/.zsh-async/async.zsh

async_init
async_start_worker my_worker -u
async_register_callback my_worker my_callback

async_job my_worker my_long_running_job arg1 arg2

my_callback() {
    local job_name=$1
    local return_code=$2
    local stdout=$3
    local stderr=$5
    print "job $job_name done: $stdout"
    zle reset-prompt
}
```

---

## 16. Prerequisites

- POSIX shell familiarity — variables, redirection, pipelines, exit codes.
- Bash baseline — `if`/`for`/`while`, `[[ ]]`, `$()`, parameter expansion, arrays. Crucial for understanding the *delta* zsh gives you.
- Process model — `fork(2)`, `exec(2)`, process groups, signals (`SIGINT`/`SIGTSTP`/`SIGCHLD`/`SIGCONT`), the controlling tty.
- Glob basics — `*`, `?`, `[abc]`. Extended glob is the natural extension.
- Terminal escape sequences — `\e[...m` for SGR (color/bold), `\e[...h/l` for modes, `\e]...\a` for OSC. The prompt and ZLE both speak these.
- Regex (basic + extended). PCRE is optional but `=~` requires `zsh/regex` semantics.
- C-level systems intuition — helps for §1 (parser → wordcode), §8 (pgrp/tty), §10 (loadable modules).
- Familiarity with at least one tab-completion-driven shell — fish or bash-completion works.

---

## 17. Complexity

| Area | Operation | Complexity | Notes |
|:-----|:----------|:----------:|:------|
| Parsing | source → wordcode | O(n) | n = source bytes; one linear pass |
| Function dispatch | call into shfunc | O(1) amortized | Hash lookup in `shfunctab` |
| Glob match | pattern vs string | O(p · s) generic, O(p+s) for simple | Backtracking on `(...)` and `*` |
| Glob qualifier sort `om` | n files | O(n log n) | One stat per file + qsort |
| Glob with `[a,b]` slice | n candidates | O(n) candidates, O(b) keepers | Full match still required |
| Compinit cold | k completion files | O(k) | k ≈ 600 on macOS Catalina |
| Compinit cached (`-C`) | k entries | O(k) load only | No disk parse |
| Tab completion | one tab | O(c · m) | c = candidates, m = matchspec rules tried |
| ZLE redraw | redraw event | O(b) | b = buffer length, plus syntax-highlight pass |
| Job table query (`jobs`) | n jobs | O(n) | n typically < 10 |
| `wait -n` | one of n children | O(1) syscall | Kernel does the wait |
| History append | one entry | O(1) memory, O(L) write | L = entry length |
| History dedup (`HIST_IGNORE_ALL_DUPS`) | new entry | O(H) | H = current history size |
| Prompt eval | one prompt | O(p) | p = total work in `precmd` + format |
| Plugin load (eager) | k plugins | O(Σ source bytes) | linear in plugin total source |
| Plugin load (turbo) | k plugins | O(1) startup, O(Σ) deferred | Deferred to after prompt |

---

## 18. See Also

- `zsh` — the practical cheatsheet for daily zsh use, options, idioms
- `bash` — the lingua-franca POSIX-extended shell zsh extends; word-splitting and array semantics differ
- `fish` — opinionated alternative shell with web-based config; no script-portability with bash, but excellent UX out of the box
- `nushell` — structured-data shell where pipelines carry typed records, not byte streams
- `polyglot` — small portable prompt theme that works in zsh, bash, ksh, and pdksh

---

## 19. References

- [Zsh Manual](https://zsh.sourceforge.io/Doc/) — the canonical reference, all options/builtins/expansions
- [Z-Shell FAQ](https://zsh.sourceforge.io/FAQ/) — covers the "why does it do that" questions
- [zsh.org](https://www.zsh.org/) — project home, release notes, mailing-list archives
- [zsh GitHub mirror](https://github.com/zsh-users/zsh) — `Src/parse.c`, `Src/exec.c`, `Src/glob.c`, `Src/Zle/*.c`
- [zsh-users mailing list archives](https://www.zsh.org/mla/) — searchable history of design discussions and patches
- [From Bash to Z Shell](https://www.bash2zsh.com/) — Kiddle, Peek, Stephenson; the standard book; deep coverage of compsys and ZLE
- [zsh-completions](https://github.com/zsh-users/zsh-completions) — community completion functions; canonical examples of `_arguments`/`_files`/`_describe` usage
- [zinit](https://github.com/zdharma-continuum/zinit) — the turbo-mode plugin manager; source is a useful read for understanding async loading
- [antidote](https://github.com/mattmc3/antidote) — minimal plugin manager built on `zsh-defer`
- [zsh-syntax-highlighting](https://github.com/zsh-users/zsh-syntax-highlighting) — canonical example of using `region_highlight` and `zle-line-pre-redraw`
- [zsh-autosuggestions](https://github.com/zsh-users/zsh-autosuggestions) — canonical example of async ZLE widgets and history-driven prediction
- `man zshall` — single-page concatenation of all zsh manual sections; useful with `less` + search
- `info zsh` — Texinfo-form manual on systems with it
- `Etc/completion-style-guide` in zsh source — completion authoring conventions
- RFC-style internal doc: `Src/glob.c` comment header — actual algorithm description for filename generation
