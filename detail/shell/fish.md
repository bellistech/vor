# The Internals of Fish — Universal Variables, the Rust Rewrite, and Autosuggestions

> *Fish is the only mainstream interactive shell that deliberately broke from POSIX. It rewrote itself from C++ to Rust between 2023 and 2024, ships a unified `string` builtin that replaces sed/awk/tr, and synchronizes universal variables across every running session via a file watcher. Understanding these internals explains why fish startup is fast, why your bash scripts won't run unmodified, and why `abbr` is the canonical idiom rather than `alias`.*

---

## 1. Fish Architecture — Then and Now

### The Two Eras

Fish has had two distinct architectural epochs:

| Era | Years | Language | Build System | Notable |
|:---|:---|:---|:---|:---|
| Classic fish | 2005–2023 | C++ | autotools, then CMake | Single-threaded, hand-rolled parser |
| Rust fish | 2023–present | Rust (core) + C++ (legacy) | Cargo + CMake | Memory-safe parser, async-friendly |

The Rust port began in earnest in early 2023 (the `fish-shell/fish-shell` repo sprouted a `src/` directory of `.rs` files alongside the long-standing `.cpp` files). Fish 4.0 shipped in 2024 as the first release with a Rust core. Fish 4.x is now the actively-developed line; the C++ code that survives is reachable only through a shrinking FFI surface.

### The Core Pipeline

Every fish session executes the same pipeline for each command line:

```
input bytes  -> tokenizer  -> parser  -> AST  -> executor  -> jobs
                  ^             ^         ^         ^
                  |             |         |         |
              (UTF-8        (LL(1),   (typed     (forks,
              aware)        end-      tree of    pipes,
                            terminated) nodes)   redirections)
```

1. **Tokenizer**: scans the byte stream and emits tokens (`tok_string`, `tok_redirection`, `tok_pipe`, `tok_andand`, `tok_oror`, `tok_end`, etc.). Quoting and brace-expansion happen here; variable expansion is *deferred* to the executor so that quoting context is preserved.
2. **Parser**: builds an AST. Fish uses an LL(1) grammar that is unambiguous because every block ends with the literal keyword `end` — there is no `fi` / `done` / `esac` polymorphism.
3. **AST**: a typed tree of `ast::Node` values (or `ast_node_t` in the legacy C++). Each node is one of `ast::JobList`, `ast::Job`, `ast::Statement`, `ast::IfStatement`, etc.
4. **Executor**: walks the AST, performs variable expansion, command resolution (builtin > function > external), and then either calls a builtin function in-process or `fork()`s an external command. `exec` is a builtin that replaces the shell with the named process without forking.
5. **Jobs**: each pipeline is a *job*; each pipeline stage is a *process*. Jobs are tracked in a process group for terminal control and SIGTSTP/SIGCONT handling.

### "Everything Is a List"

Fish has one composite type: the list. There are no associative arrays, no maps, no objects. A scalar is just a list of length one.

```fish
set greeting hello       # list of length 1
set names alice bob carol # list of length 3
set -l empty             # list of length 0

count $greeting          # 1
count $names             # 3
count $empty             # 0
```

The `$names` expansion in command position becomes three positional arguments — this is fundamentally different from POSIX, where `$names` would be a single string requiring word splitting.

### The Rust Crate Layout (Fish 4.x)

```
fish-shell/
  src/
    builtins/        # builtin command implementations (one .rs per builtin)
    parser.rs        # tokenizer + AST builder
    ast.rs           # AST node types
    expand.rs        # variable, command, brace, glob expansion
    exec.rs          # process spawning, pipeline orchestration
    env/             # environment + universal variable subsystem
      universal.rs   # ~/.config/fish/fish_variables read/write/watch
      var.rs         # ENV_VAR struct (scope flags, exported, value)
    history.rs       # history file format (YAML-ish) + search
    highlight.rs     # syntax-highlighting state machine
    input.rs         # readline-equivalent: keybindings, completion driver
    reader.rs        # the interactive loop — prompt, edit, accept
    function.rs      # function autoloading + storage
    event.rs         # fish_postexec / on-variable / on-event dispatch
    string_match.rs  # the `string` builtin's PCRE2 wrapper
```

The legacy C++ files (`src/*.cpp`) shrink with each release; many were rewritten line-for-line into their `.rs` siblings.

---

## 2. The No-Word-Splitting Decision

### What Bash Does

In POSIX shells, *unquoted* variable expansion is split on `$IFS` (by default space, tab, newline) and the resulting fields are subject to globbing.

```bash
# bash
files="a.txt b.txt"
ls $files          # ls a.txt b.txt        (two arguments)
ls "$files"        # ls 'a.txt b.txt'      (one argument, file not found)
```

This is the source of an entire genre of bash bugs: anyone who has written `for f in $(ls)` has been bitten. The shellcheck linter exists in part to flag every unquoted expansion as suspicious.

### What Fish Does

Fish never splits on whitespace. A variable's value is whatever you stored, and a list-valued variable expands to multiple arguments only because each element of the list is a separate argument.

```fish
# fish
set files "a.txt" "b.txt"  # list of two strings
ls $files                  # ls a.txt b.txt  (two arguments — list expansion)

set name "John Doe"        # list of length 1
echo $name                 # John Doe        (single argument, NOT split)
```

There is no `IFS` variable in fish. None. You cannot configure it because the splitting it controls does not happen.

### The Implications

| Scenario | Bash | Fish |
|:---|:---|:---|
| `var=" a b "; echo $var` | Splits to `a` `b` | Echoes ` a b ` literally |
| `var="a*"; echo $var` | Globs and prints matches | Prints `a*` literally (no glob in expansion) |
| `for f in $(cmd)` | Splits cmd's stdout on IFS | Splits *only on newline* (less hostile) |
| `cmd $var` where var has spaces | Two args | One arg |

### Explicit String Splitting

When you actually want to split a string, you ask for it:

```fish
set csv "alice,bob,carol"
set parts (string split "," $csv)   # parts is a 3-element list
echo $parts[2]                       # bob

# split on whitespace (multiple separators not native — use regex)
set words (string split -r " +" "  hello   world  ")
# or, more idiomatically:
set words (string split " " "hello world" | string match -v "")
```

The contrast: in bash, splitting is the default and quoting is the explicit opt-out. In fish, splitting is the explicit opt-in. The result is that fish scripts have far fewer "what if my filename has a space in it" bugs.

---

## 3. Variable Scope Model

### The Four Scopes

Fish has four variable scopes, deliberately fewer than zsh's seven but with sharper semantics.

| Scope | Flag | Lifetime | Visibility |
|:---|:---|:---|:---|
| Local | `-l` / `--local` | Until end of nearest block (`begin`/`end`, function, loop) | Current scope only |
| Function | (default inside functions) | Until function returns | Within function and nested scopes |
| Global | `-g` / `--global` | Until shell exits | All scopes in current shell |
| Universal | `-U` / `--universal` | Persisted to disk; survives reboot | All current and future fish sessions |

Orthogonal to scope is the *export* flag (`-x` / `--export`), which makes the variable visible to child processes via the standard environment.

### The Flag Matrix

```fish
set -l x value         # local, not exported
set -g x value         # global, not exported
set -U x value         # universal, not exported (rare combination)
set -lx x value        # local AND exported (visible to children of this block)
set -gx x value        # global AND exported (the canonical "shell variable")
set -Ux x value        # universal AND exported (the canonical "user setting")
```

The `-U` and `-x` combination is what most users want for things like `EDITOR`: persist across sessions *and* make it visible to subprocesses.

```fish
set -Ux EDITOR nvim
set -Ux PAGER less
```

### Scope Resolution

When fish reads `$x`, it walks scopes from innermost to outermost:

1. Local (current block)
2. Each enclosing block, outward
3. Function scope (if inside a function)
4. Global
5. Universal

The first match wins. There is no shadowing surprise: `set` *creates* in the most local scope by default, but `set -g x value` from inside a function modifies (or creates) the global, never a local.

```fish
function demo
    set -l x inner
    set -g x outer
    echo $x        # "inner" — local takes precedence
end
demo
echo $x            # "outer" — global persists after function returns
```

### Querying Scope

```fish
set --show PATH    # prints scope, exported flag, and value(s) for PATH
set -q PATH        # exit 0 if PATH is set, nonzero otherwise
set -Sq PATH       # also true if PATH is just "set in any scope"
```

The output of `set --show PATH` is genuinely useful for debugging — it tells you *which scope* the value came from.

---

## 4. Universal Variable Persistence

### The File Format

Universal variables live in `~/.config/fish/fish_variables`. The file format is documented but obscure; here is a representative sample (fish 4.x):

```
# This file contains fish universal variable definitions.
# VERSION: 3
SETUVAR EDITOR:nvim
SETUVAR PAGER:less
SETUVAR fish_color_command:blue
SETUVAR fish_greeting:\x1d
SETUVAR --export fish_user_paths:/usr/local/bin\x1e/opt/homebrew/bin
```

Key encoding details:

- `\x1e` (record separator, ASCII 30) separates list elements
- `\x1d` (group separator, ASCII 29) is the empty-list sentinel
- `--export` precedes the variable name when the export flag is set
- Lines are sorted by name for stability (so version control sees minimal diffs)
- Backslash-escaped non-printable bytes preserve binary safety

### Live Updates Across Sessions

Fish watches the file for changes. The mechanism is OS-dependent:

| Platform | Mechanism |
|:---|:---|
| Linux | `inotify` on the file's parent directory |
| macOS | kqueue with `EVFILT_VNODE` |
| BSDs | kqueue (same as macOS) |
| Windows (WSL/Cygwin) | Polling fallback, ~1s interval |

When session A runs `set -U FOO bar`, fish A:

1. Acquires an advisory lock (`flock(LOCK_EX)`) on a sibling file `fishd.<hostname>.<uid>.lock`.
2. Parses the existing `fish_variables` file.
3. Mutates the in-memory representation.
4. Writes the new file atomically (write to `fish_variables.tmp`, then `rename(2)`).
5. Releases the lock.

Sessions B, C, D each receive a filesystem event. Their universal-variable subsystem re-reads the file and diffs the contents against their cached snapshot. For each variable that changed, they fire `on-variable` events. The result is that `set -U` in one terminal shows up in another terminal within a few hundred milliseconds.

### The Antipattern

Universal variables are seductive: "I'll use them as a global config store." Don't.

**Why it's bad:**

- They're stored in your home directory but are *machine-specific* — syncing dotfiles across machines via git won't carry them. (You'd have to commit `fish_variables`, which is generally a bad idea because it intermixes user preferences and computed values.)
- They have no schema, so a typo creates a new variable rather than failing.
- Removing a feature means scattering `set -Ue OLD_VAR` cleanups everywhere.

**Use instead:**

- For settings that should survive: `set -Ux EDITOR nvim` is fine — it's explicitly an environment variable.
- For colors and theme: the `fish_color_*` family is *meant* to be universal.
- For everything else: write them as `set -g` in `~/.config/fish/config.fish` so the source of truth is your version-controlled config, not a stateful sidecar file.

### Erasing

```fish
set -Ue MY_OLD_VAR    # erase from universal scope (the only sane way to clean up)
set -e PATH           # erase from default scope (whichever is innermost)
```

---

## 5. The `string` Builtin — Internals

### The Design Philosophy

In bash you compose: `echo "$var" | sed 's/foo/bar/'`. In fish you call `string`:

```fish
string replace foo bar $var
```

The fork+exec to spawn `sed` costs roughly 1–5ms; an in-process `string replace` costs microseconds. For interactive shells where the prompt may run dozens of `string` calls, this matters.

### The Subcommands

| Subcommand | Purpose | Example |
|:---|:---|:---|
| `string match` | Glob or regex match (filter) | `string match "*.md" *` |
| `string match -r` | PCRE2 regex with capture groups | `string match -r '(\d+)' "v42"` |
| `string replace` | Literal substring replacement | `string replace foo bar $s` |
| `string replace -r` | PCRE2 substitution | `string replace -r '\d+' N $s` |
| `string split` | Split on a separator | `string split , a,b,c` |
| `string split0` | Split on NUL bytes | `find ... -print0 \| string split0` |
| `string split -m N` | Limit to N splits | `string split -m 1 = "k=v=more"` |
| `string sub` | Substring by start/length | `string sub -s 2 -l 3 abcdef` |
| `string trim` | Strip whitespace (or chars) | `string trim "  hi  "` |
| `string trim -l` / `-r` | Trim only left or right | `string trim -l "  hi  "` |
| `string repeat -n 3` | Repeat | `string repeat -n 5 "ab"` |
| `string escape` | Quote for fish reuse | `string escape "$weird"` |
| `string unescape` | Inverse of escape | `string unescape "\\\$"` |
| `string upper/lower/title` | Case conversion | `string upper "hello"` |
| `string length` | Byte or char count | `string length --visible $s` |
| `string pad` | Pad to width | `string pad -w 10 hi` |
| `string collect` | Force a single arg | `set out (cmd \| string collect)` |
| `string join` / `string join0` | Inverse of split | `string join , $list` |

### Regex Capture Groups

```fish
set version "fish version 3.7.1"
if string match -rq '(?<major>\d+)\.(?<minor>\d+)\.(?<patch>\d+)' -- $version
    echo "major=$major minor=$minor patch=$patch"
end
```

Named groups (`?<name>`) become local variables in the calling scope. Unnamed capture groups are emitted as a list.

### Escape Modes

The `string escape` builtin has four escape styles, each round-trips through `string unescape`:

| Mode | Flag | Output style |
|:---|:---|:---|
| Script | (default) | Backslash escapes for fish |
| URL | `--style=url` | Percent-encoding (`%20`) |
| Var | `--style=var` | Variable-name-safe (alphanumeric + `_`) |
| Regex | `--style=regex` | Escapes PCRE metacharacters |

```fish
string escape --style=url "hello world & friends"   # hello%20world%20%26%20friends
string escape --style=regex "1.2.3"                 # 1\.2\.3
```

### Why It Matters

A real `git status` parser:

```fish
function changed_files
    git status --porcelain | string split0 -m 0 \n | while read line
        set status (string sub -s 1 -l 2 -- $line)
        set path (string sub -s 4 -- $line)
        echo "$status -> $path"
    end
end
```

In bash this would be a tangle of `read -r`, IFS manipulation, and `awk`. In fish it's three `string` calls.

---

## 6. Function Definitions and Loaded-on-Demand

### The Autoload Mechanism

When you type `myfunc` at the prompt, fish resolves the name in this order:

1. **Builtin?** (`set`, `cd`, `function`, `string`, `count`, ...)
2. **Function?** Scan `$fish_function_path`, which by default is:
   - `~/.config/fish/functions/`
   - `/usr/share/fish/functions/`
   - `/usr/share/fish/vendor_functions.d/`
   - Plus anything added by plugins or via `set -Ua fish_function_path /custom`
3. **External command?** Walk `$PATH`.

If a function is *autoloaded*, the file `<name>.fish` is sourced *the first time the function is called*. This is the canonical way to lazy-load expensive functions: put them in `~/.config/fish/functions/<name>.fish` and they cost nothing at startup.

```fish
# ~/.config/fish/functions/deploy.fish
function deploy --description "deploy to production"
    # ... 200 lines of logic ...
end
```

This file is *not* sourced when fish starts. It's parsed lazily. Your `config.fish` should never contain the body of a function — only invocations or definitions that *must* run at startup (like setting `$PATH`).

### Function Header Flags

The full grammar of `function`:

```fish
function NAME [OPTIONS...]; BODY; end
```

The most useful options, with their canonical use cases:

| Flag | Purpose |
|:---|:---|
| `--description "..."` / `-d` | Shown in `functions --details` and tab completion |
| `--argument-names a b c` / `-a` | Bind `$argv[1..3]` to named locals |
| `--on-event NAME` / `-e` | Fire on a fish event (`fish_prompt`, custom emits) |
| `--on-variable NAME` / `-v` | Fire when variable changes |
| `--on-job-exit JOBSPEC` / `-j` | Fire when a backgrounded job exits |
| `--on-process-exit PID` / `-p` | Fire when a specific PID exits |
| `--on-signal SIG` / `-s` | Fire on signal (e.g., `SIGUSR1`, `INT`, `WINCH`) |
| `--no-scope-shadowing` / `-S` | See/modify caller's local variables |
| `--inherit-variable NAME` / `-V` | Capture caller's value at definition time |
| `--wraps CMD` / `-w` | Inherit `CMD`'s tab completions |

### Argument Binding Pattern

```fish
function greet --argument-names name greeting
    echo "$greeting, $name!"
end

greet alice hello   # hello, alice!
greet bob           # , bob! ($greeting is empty list)
```

`$argv` is always available as the full list; `--argument-names` is sugar.

### The "Function as Alias" Pattern

Fish has an `alias` builtin, but it is implemented as a wrapper around `function`. The canonical pattern:

```fish
function gst --wraps='git status'
    git status $argv
end
funcsave gst
```

`funcsave` writes the function to `~/.config/fish/functions/gst.fish`. The next time you type `gst`, fish autoloads it. `--wraps='git status'` inherits git's completions, so tab-completion still works.

### Closures via `--inherit-variable`

Fish does not have first-class lexical closures, but you can capture values:

```fish
function make_greeter --argument-names greeting
    function _greeter --inherit-variable greeting
        echo "$greeting, $argv"
    end
    functions -c _greeter "greeter_$greeting"
end

make_greeter hi
greeter_hi alice    # hi, alice
```

`functions -c` copies a function under a new name. `--inherit-variable` captures the value at definition time (not by reference).

---

## 7. The Event System

### Built-in Events

Fish emits a number of events automatically. Listeners are registered with `function --on-event NAME`.

| Event | When |
|:---|:---|
| `fish_prompt` | Before `fish_prompt` runs (i.e., before drawing the prompt) |
| `fish_preexec` | Just before executing a command |
| `fish_postexec` | Just after a command completes |
| `fish_command_not_found` | When fish can't resolve a command |
| `fish_exit` | When the shell is exiting |
| `fish_focus_in` | Terminal focus gained (requires terminal support) |
| `fish_focus_out` | Terminal focus lost |
| `fish_winch` | Terminal window resized (SIGWINCH) |
| `fish_cancel` | User pressed Ctrl-C at the prompt |
| `fish_posterror` | A syntax error was reported |

### Variable-Change Events

```fish
function _redraw_prompt_on_pwd --on-variable PWD
    # fires every time PWD changes (i.e., on cd)
    commandline -f repaint
end
```

Variable events are the foundation of dynamic prompts: when `$status` (set automatically after every command) changes, your prompt re-renders.

### Custom Events

```fish
function on_deploy --on-event deploy_complete
    notify-send "Deploy finished at $(date)"
end

# Elsewhere:
emit deploy_complete production
```

`emit` takes the event name and any number of arguments, which become `$argv` inside the listener.

### Multi-Listener Semantics

Multiple functions can listen to the same event. Fish runs them in *registration order* (the order in which their `--on-event` declarations were parsed). All listeners run; there is no short-circuit and no return-value chain.

```fish
function logger --on-event fish_postexec
    echo "[$(date)] ran: $argv" >> ~/.cmd.log
end

function flasher --on-event fish_postexec
    test $status -ne 0 && set_color red && echo "FAIL" && set_color normal
end
```

Both run after every command. There is no way to declare priority — registration order is the contract.

### Process and Job Exit Events

```fish
sleep 60 &
function _on_sleep --on-process-exit %last
    echo "background sleep finished"
end
```

`%last` is shorthand for the most recent backgrounded PID. For a specific job: `--on-job-exit %1`.

### The Cost Model

Each event dispatch is O(N) in the number of registered listeners. For high-frequency events like `fish_prompt`, this matters: a prompt with 30 listeners adds 30 function-call overheads to every keystroke that triggers a redraw. Profile with `fish --profile` if your prompt feels sluggish.

---

## 8. Abbreviations vs Aliases

### What `abbr` Does

`abbr` is fish's killer feature for command-line ergonomics. An abbreviation is a token that expands *as you type space or enter*, leaving the full command in your shell history.

```fish
abbr -a gco git checkout
abbr -a gp 'git push origin (git rev-parse --abbrev-ref HEAD)'

# At the prompt, type:
gco<space>
# fish replaces the line with:
git checkout |
```

The abbreviation has been *expanded into the editable buffer*. You can edit it before pressing enter, and the history records `git checkout main`, not `gco main`. Six months later you can read your history and understand what you did.

### What `alias` Does

`alias` in fish is a *function-generator*. It creates a function with the name of the alias and the body of the expansion.

```fish
alias gco='git checkout'
# Equivalent to:
function gco
    git checkout $argv
end
```

The function shadows the original. Your history records `gco main`, which is opaque a year later.

### The Comparison Table

| Property | `abbr` | `alias` (in fish) | bash/zsh `alias` |
|:---|:---|:---|:---|
| Expands in buffer | Yes | No (transparent) | No (transparent) |
| History records full form | Yes | No | No |
| Editable before submit | Yes | No | No |
| Visible to scripts | No | Yes (it's a function) | Yes |
| Persistence | `--save` writes to universal | `funcsave` writes to file | `~/.bashrc` |
| Cost | Negligible (lookup at space) | Function-call overhead | String substitution |
| Position-aware (3.6+) | Yes (`--position command`) | No | No |

### Position-Aware Abbreviations (Fish 3.6+)

```fish
# Only expand at command position (start of pipeline)
abbr -a --position command g git

# Expand anywhere (the default)
abbr -a --position anywhere ngrep 'grep -n'
```

Without position-awareness, `git push g foo` would expand the second `g` mid-command. With `--position command`, only the leading token expands.

### Regex Abbreviations (Fish 3.6+)

```fish
abbr -a --regex 'v\d+\.\d+\.\d+' --function expand_version
function expand_version
    string replace -r '^v' '' -- $argv
end
```

Type `v3.7.1<space>` and fish strips the leading `v`. Useful for tag/version idioms.

### The `--save` and `--erase` Flags

```fish
abbr -a --save gco git checkout    # persists across sessions
abbr --erase gco                    # removes
abbr --query gco                    # exit 0 if defined
abbr -s                             # show all (script-friendly format)
```

Saved abbreviations land in `~/.config/fish/fish_variables` as universal variables (one per abbreviation, prefixed `_fish_abbr_`). This means they sync across all your fish sessions immediately.

---

## 9. Pipes and Process Model

### The Fork-Per-Stage Model

A fish pipeline `cmd1 | cmd2 | cmd3` results in:

1. fish `pipe(2)` creates two pipe pairs.
2. fish `fork(2)` three times.
3. Each child closes the unused pipe ends and `dup2`s its stdin/stdout to the right pipe.
4. Each child `execve(2)`s its command.
5. fish `waitpid(2)` for each child.

For a pipeline of *N* stages, fish forks *N* processes. Builtins are special — they can run in-process if they're at the *start* of the pipeline; otherwise they fork too (because they need their own stdin/stdout).

### The `$status` Array

In bash, `$?` is the exit code of the last command. To get the codes of all stages, you read `${PIPESTATUS[@]}`, which is a separate variable that you have to remember.

In fish, `$status` is a list:

```fish
false | true | grep "x"
echo $status              # 0          (last stage)
echo $pipestatus          # 1 0 1      (all stages — fish-specific)
```

`$pipestatus` is the array; `$status` is the last element of `$pipestatus`. There is no separate "PIPESTATUS" — it's all just `$pipestatus`.

### Redirection Operators

| Operator | Bash equivalent | Meaning |
|:---|:---|:---|
| `>` | `>` | stdout to file (truncate) |
| `>>` | `>>` | stdout to file (append) |
| `2>` | `2>` | stderr to file |
| `&>` | `&>` | both stdout and stderr to file |
| `\|` | `\|` | stdout to pipe |
| `2>\|` | `2>&1 \|` | stderr to pipe |
| `&\|` | `\|&` (zsh) / `2>&1 \|` (bash) | both to pipe |
| `<` | `<` | stdin from file |
| `<&-` | `<&-` | close stdin |

The fish-isms:

```fish
# both stdout and stderr to file
make &> build.log

# both stdout and stderr through a pipe
make &| less

# stderr only through a pipe (stdout untouched)
make 2>| grep ERROR
```

### Process Substitution

Fish does not have bash's `<(cmd)` syntax. The idiomatic replacement is the `psub` function:

```fish
# bash: diff <(cmd1) <(cmd2)
diff (cmd1 | psub) (cmd2 | psub)
```

`psub` writes its stdin to a temp file (or named pipe on Linux) and prints the path. The temp file is cleaned up after the surrounding command exits.

### Backgrounding and Job Control

```fish
sleep 60 &              # background
jobs                    # list jobs
fg %1                   # foreground job 1
disown %1               # detach (won't be killed on shell exit)
wait                    # block until all background jobs finish
```

Fish job IDs (`%1`, `%2`, ...) are stable for the life of the job. `%last` always refers to the most recently backgrounded job.

---

## 10. The `argparse` Builtin

### The Canonical Pattern

Every well-written fish function that takes flags follows this template:

```fish
function deploy --description "deploy to env"
    argparse 'h/help' 'v/verbose' 'e/env=' 't/timeout=!_validate_int' -- $argv
    or return

    if set -q _flag_help
        echo "usage: deploy [-v] -e ENV [-t SEC]"
        return 0
    end

    set -l env $_flag_env
    set -l timeout (set -q _flag_timeout; and echo $_flag_timeout; or echo 30)

    test -n "$env"; or echo "missing -e"; or return 1

    # ... use $argv (positional args remaining after flags) ...
end
```

### The Spec Grammar

Each spec string has the form `SHORT/LONG[=|=?|=+][!VALIDATOR]`:

| Spec | Meaning |
|:---|:---|
| `h/help` | `-h` and `--help`, no argument |
| `e/env=` | `-e VAL` / `--env=VAL`, exactly one argument |
| `f/file=?` | optional argument |
| `i/include=+` | one or more occurrences accumulate into a list |
| `t/timeout=!_validate_int` | argument required, validated by function |

A bare `h/help` becomes `$_flag_h` and `$_flag_help` set to one or more `--help` (counted occurrences). `e/env=` becomes `$_flag_e` and `$_flag_env` set to the argument value.

### The `--` Convention

The `-- $argv` is the boundary between argparse's own options and the user's argv. After argparse runs:

- Recognized flags become `$_flag_*` variables in the *function's* local scope (because argparse uses `--no-scope-shadowing` semantics on its caller).
- Unrecognized positionals overwrite `$argv` to contain only the positional remainder.

```fish
function example
    argparse 'v/verbose' 'o/output=' -- $argv
    or return

    echo "verbose: $_flag_verbose"
    echo "output:  $_flag_output"
    echo "argv:    $argv"
end

example -v -o /tmp/out.log file1 file2
# verbose: -v
# output:  /tmp/out.log
# argv:    file1 file2
```

### Validators

```fish
function _validate_int
    string match -qr '^\d+$' -- $_flag_value
    or echo "$_flag_name must be an integer" >&2
end

function example
    argparse 't/timeout=!_validate_int' -- $argv
    or return
    # ...
end

example --timeout abc
# error: --timeout must be an integer
```

The validator function reads `$_flag_value` (the proposed value) and `$_flag_name` (the option being validated) from the caller's scope. Returning nonzero rejects the value.

### Comparison with Bash `getopts`

```bash
# bash
while getopts ":hve:t:" opt; do
    case $opt in
        h) help=1 ;;
        v) verbose=1 ;;
        e) env="$OPTARG" ;;
        t) timeout="$OPTARG" ;;
        \?) echo "invalid option" ;;
    esac
done
shift $((OPTIND-1))
```

Bash's `getopts` doesn't support long options (you'd need GNU `getopt`, which is a separate fork-exec). It doesn't validate types. It doesn't generate help text. It's a 1980s-era POSIX leftover.

Fish's `argparse` is the modern alternative: long options, accumulators, validators, scope-aware. For any fish function with more than two flags, use `argparse`.

---

## 11. Autosuggestions and History Search

### What You See

As you type, fish shows a grey "ghost" continuation of the current line — the suggestion. Right-arrow accepts it; Ctrl-F also accepts; Alt-rightarrow accepts only the next word.

```
$ git push origin <ghost: main --force-with-lease>
```

### Where the Suggestion Comes From

Fish maintains three sources, queried in order:

1. **History match (prefix)**. The most recent line in `~/.local/share/fish/fish_history` whose prefix matches the current input.
2. **History match (subsequence)**. If no prefix match, fall back to subsequence ranking (Levenshtein-ish, weighted by recency).
3. **Completion match**. If history has no candidate, ask the active completion script (`complete -c git ...` definitions). The first completion is shown as the suggestion.

### The History File Format

Fish history is a YAML-ish line-oriented format:

```yaml
- cmd: git status
  when: 1714060000
- cmd: git push origin main
  when: 1714060042
  paths:
    - origin
    - main
```

Each entry has `cmd` (the command), `when` (Unix timestamp), and an optional `paths` array (used for path-aware history search). Files are appended; deduplication happens at read time, with the most recent occurrence winning.

### Ctrl-R: Interactive History Search

`Ctrl-R` invokes the *interactive history pager*. As you type, fish filters the history list using a fuzzy (subsequence) match. Up/down arrows navigate; enter accepts.

The fuzzy ranking weights:

- **Length match**: shorter matches rank higher.
- **Word boundary match**: `gss` matching `g[it] s[tatus] -[s]` ranks high because each character matches a word start.
- **Recency**: ties broken by most-recent-first.

### History Configuration

```fish
set -U fish_history default                  # which history "session" to use
set -U HISTSIZE 50000                        # max entries kept
set -U fish_history "${USER}_${HOSTNAME}"    # per-host history (avoid syncing)
```

The `fish_history` variable selects the history *file*; setting it to a different name lets you have project-specific or host-specific histories. `private` is a special value that disables history persistence entirely for the session.

### Disabling History for a Single Command

Prefix a command with a space to skip history:

```fish
 secret_command --token=abc123     # leading space — not in history
```

This requires `set -gx fish_skip_history_with_leading_space 1`, which is the default in fish 3.4+.

---

## 12. Color and Theming

### The `set_color` Builtin

```fish
set_color red                       # foreground red
set_color -b blue                   # background blue
set_color --bold red                # bold red
set_color --italics yellow
set_color --underline cyan
set_color --reverse
set_color FF6600                    # 24-bit (truecolor) hex
set_color normal                    # reset
```

In a prompt:

```fish
function fish_prompt
    set_color cyan
    echo -n (whoami)
    set_color normal
    echo -n "@"
    set_color magenta
    echo -n (hostname -s)
    set_color normal
    echo -n " "
    set_color blue
    echo -n (prompt_pwd)
    set_color normal
    echo -n " > "
end
```

### Syntax-Highlighting Variables

Fish exposes its syntax highlighting as a family of universal variables. Setting them changes the editor in real time across all sessions.

| Variable | What It Colors |
|:---|:---|
| `fish_color_command` | Recognized commands |
| `fish_color_param` | Parameters/arguments |
| `fish_color_quote` | Quoted strings |
| `fish_color_redirection` | `>`, `<`, `\|`, `&>` |
| `fish_color_end` | `end`, `;`, `&` |
| `fish_color_error` | Syntax errors / unknown commands |
| `fish_color_comment` | Comments |
| `fish_color_match` | Matching brace under cursor |
| `fish_color_search_match` | Highlighted history-search match |
| `fish_color_operator` | `&&`, `\|\|`, etc. |
| `fish_color_escape` | `\n`, `\xNN` in strings |
| `fish_color_autosuggestion` | The grey ghost text |

```fish
set -U fish_color_command blue --bold
set -U fish_color_autosuggestion 555 --italics    # dim grey, italic
set -U fish_color_param FF6600                    # truecolor
```

### The Pager Color Family

```fish
set -U fish_pager_color_prefix cyan --bold      # the matched prefix in completions
set -U fish_pager_color_completion white
set -U fish_pager_color_description yellow
set -U fish_pager_color_progress black --background=cyan
set -U fish_pager_color_selected_background --background=blue
```

These control the popup completion menu (the table you see after pressing Tab on an ambiguous prefix).

### `fish_config theme` (Fish 3.4+)

Fish ships with a builtin theme browser:

```fish
fish_config theme list
fish_config theme show
fish_config theme save "Solarized Dark"
fish_config theme dump > my_theme.theme
```

`fish_config` opens a web UI in your browser for visual editing. The dumped theme file is a text format that can be checked into your dotfiles.

---

## 13. Plugin System and `fisher`

### The Layout Convention

A fish plugin is a directory or repo with up to four conventional subdirectories:

| Subdir | Purpose | Loaded When |
|:---|:---|:---|
| `functions/` | Function autoload sources | On first call to `<funcname>` |
| `completions/` | Tab completion specs | On first tab-complete of `<cmdname>` |
| `conf.d/` | Setup snippets | At shell startup, alphabetical order |
| `themes/` | Color theme files (3.4+) | On `fish_config theme` invocation |

`conf.d/` files are sourced eagerly at startup, so they should be tiny — just `set` calls and `bind` registrations. Heavy work belongs in `functions/`.

### `fisher` — the Canonical Package Manager

```fish
# Install fisher itself
curl -sL https://raw.githubusercontent.com/jorgebucaran/fisher/main/functions/fisher.fish | source && fisher install jorgebucaran/fisher

# Install a plugin
fisher install IlanCosman/tide@v6
fisher install jorgebucaran/nvm.fish
fisher install PatrickF1/fzf.fish

# List installed
fisher list

# Update all
fisher update

# Remove
fisher remove jorgebucaran/nvm.fish
```

`fisher` reads `~/.config/fish/fish_plugins` (a plain text file, one plugin per line) and synchronizes the installed set. Committing this file to your dotfiles repo means `fisher update` on a new machine reproduces your exact plugin set.

### `oh-my-fish` (OMF)

`oh-my-fish` is the older alternative. It installs to `~/.local/share/omf` and uses `package.json`-style metadata. It is still maintained but `fisher` is the de-facto standard in the fish 3.x and 4.x era — fisher is smaller, faster, and uses the conventional subdirectory layout without metadata files.

### The `fish_plugins` File

```
# ~/.config/fish/fish_plugins
jorgebucaran/fisher
IlanCosman/tide@v6
jorgebucaran/nvm.fish
PatrickF1/fzf.fish
```

This is the source of truth. `fisher update` reads this file, fetches the listed plugins (cloning git refs), and prunes anything not listed.

### Writing a Plugin

```fish
# my-plugin/functions/hello.fish
function hello --description "say hello"
    echo "hello, $argv"
end

# my-plugin/completions/hello.fish
complete -c hello -d "say hello"

# my-plugin/conf.d/hello.fish
# only setup at startup — keep tiny
set -gx HELLO_GREETING "hi"
```

Push to GitHub. Users install with `fisher install user/my-plugin`.

---

## 14. Performance — Startup and Profile

### The Startup Sequence

When fish starts, it does (roughly):

1. Read built-in init from `/usr/share/fish/config.fish` and friends.
2. Read `~/.config/fish/conf.d/*.fish` (alphabetical).
3. Read `~/.config/fish/config.fish`.
4. Load universal variables from `~/.config/fish/fish_variables`.
5. Set up signal handlers and the terminal state machine.
6. Render the first prompt.

The cumulative wall-clock budget for "responsive" feels under 100ms. Anything more and you'll feel a lag opening a new terminal.

### Profiling

```fish
fish --profile-startup ~/fish-startup.log -i -c exit
# generates a tab-separated log:
# time(usec)   sum(usec)   command
```

Each line is one statement; `time` is its own cost; `sum` is its cost plus the cost of everything it called. Look for the lines with the highest `sum`.

```fish
fish --profile ~/fish-runtime.log -i
# profiles the interactive session — every command, every keystroke event, every prompt render
```

Fish 4.x adds `--profile-startup-out` (alias) and refines the format slightly.

### Common Slow-Start Sources

| Source | Typical cost | Mitigation |
|:---|:---|:---|
| Universal-variable file load | 1–5ms | Don't accumulate thousands |
| `pyenv init`, `rbenv init`, `nvm.sh` | 50–500ms | Use lazy-loading shims |
| `direnv hook fish` | 5–10ms | Acceptable; can defer |
| `starship init fish` | 10–30ms | Acceptable; single fork |
| Auto-completion man-page parse | First-tab only | One-time, cached to `~/.local/share/fish/generated_completions` |
| Conditional `command -q foo` checks | 1ms each | Cache results in universal vars |

### Minimizing `config.fish`

The fast `config.fish`:

```fish
# only runs interactively
status is-interactive; or exit

# universal-vars handle PATH, EDITOR, PAGER — don't re-set
# universal-vars handle colors — don't re-set
# universal-vars handle abbreviations — don't re-set

# only thing here: tooling that *must* run on every shell
if command -q starship
    starship init fish | source
end
```

Everything else moves to `~/.config/fish/conf.d/*.fish` (where it's organized) or `~/.config/fish/functions/*.fish` (where it's lazy).

### The Universal-Variable File Cost

Each `set -U` write incurs:

- Acquire flock: ~0.1ms.
- Read existing file: ~0.5ms.
- Diff + serialize: ~0.5ms per variable.
- Atomic rename: ~0.2ms.
- Notify all sessions: 0–5ms (depends on watcher and number of sessions).

So 100 universal variables in a single batch: ~5ms. 1000 universal variables: maybe 30ms. The file load on shell startup is roughly linear in the number of variables.

---

## 15. Migration from Bash

### The Hard Truth

Fish is **not POSIX**. None of these work in fish:

```fish
# These all fail in fish:
export FOO=bar              # 'export' is a function in fish, but the syntax is `set -x FOO bar`
[ -f file ]                  # works (test alias) but quotes/branching differ
foo=$(cmd)                   # not assignment syntax — use `set foo (cmd)`
if [ "$x" = "y" ]; then      # 'then' is bash-only — fish has no 'then'
$(cmd)                       # subshell — fish uses (cmd) without dollar
$@ "$@"                      # all-args — fish has $argv
```

### The `bass` Tool

`bass` is a fish plugin that lets you source bash scripts in a bash subshell and import the resulting environment changes:

```fish
fisher install edc/bass

# Run a bash-only setup script:
bass source ~/.nvm/nvm.sh
bass export FOO=bar

# Now $FOO is set in fish, even though the syntax was bash.
```

Internally, `bass` does:

1. `bash -c "source $1; env"` to capture the resulting environment.
2. Diffs it against the current environment.
3. Translates each diff to `set -gx VAR value`.

This lets you keep using bash-only tooling (`nvm`, `rvm`, `pyenv`'s init script, vendor SDK setup scripts) inside fish.

### The Right Mental Model

```
Use bash for: scripts that run on lots of machines, scripts that other people maintain,
              scripts that need to be POSIX-compatible, CI/CD pipelines.

Use fish for: your interactive shell, your personal aliases (abbr), your prompt,
              your local productivity scripts.
```

The shebang `#!/usr/bin/env fish` is fine for your personal scripts, but anything you commit to a multi-developer repo should be `#!/bin/bash` or `#!/bin/sh`.

### Translation Cheat-Sheet

| Bash | Fish |
|:---|:---|
| `export FOO=bar` | `set -gx FOO bar` |
| `unset FOO` | `set -e FOO` |
| `FOO=$(cmd)` | `set FOO (cmd)` |
| `if [ -f /etc/passwd ]; then ...; fi` | `if test -f /etc/passwd; ...; end` |
| `for i in 1 2 3; do echo $i; done` | `for i in 1 2 3; echo $i; end` |
| `function foo { echo "$1"; }` | `function foo; echo $argv[1]; end` |
| `case "$x" in foo) ...;; esac` | `switch $x; case foo; ...; end` |
| `cmd1 && cmd2 \|\| cmd3` | `cmd1; and cmd2; or cmd3` (or `&&`/`\|\|` in 3.0+) |
| `echo "${var:-default}"` | `echo (set -q var; and echo $var; or echo default)` |
| `${#var}` | `string length $var` |
| `${var/foo/bar}` | `string replace foo bar $var` |
| `$@` | `$argv` |
| `$1` | `$argv[1]` |
| `>&2` | `>&2` (same) |
| `2>&1` | `2>&1` (same) |
| `$?` | `$status` |
| `$$` | `$fish_pid` |
| `$!` | `$last_pid` |
| `${PIPESTATUS[*]}` | `$pipestatus` |

---

## 16. The Rust Rewrite

### The Backstory

Fish was written in C++ from 2005 to 2023. The rewrite to Rust began in late 2022 as a discussion thread (https://github.com/fish-shell/fish-shell/issues/8420) and accelerated through 2023. The first PRs ported leaf utilities (string manipulation, the wcs<->utf8 conversion layer); by mid-2023 the parser was being ported; by late 2023 the executor was the active battleground. Fish 4.0 shipped in 2024 as the first release where the *core* of fish — the parser, executor, env subsystem — runs in Rust.

### Why Rust

The stated motivations from the maintainers:

1. **Memory safety**. Fish's C++ had a long history of obscure bugs in completion handling, terminal escape sequence parsing, and signal handling — places where pointer lifetime is subtle. Rust's borrow checker eliminates an entire class of these.
2. **Better concurrency primitives**. Fish was always single-threaded for the executor, but file watching, completions, and history search benefit from `Arc<Mutex<...>>` and channels in ways that C++ shared-mutable-state made painful.
3. **Build simplicity**. Cargo + rustup is more reliable across platforms than the old autotools/cmake mix. Cross-compilation in Cargo is one flag.
4. **Developer-pool growth**. The set of developers who can read Rust is now larger than the set who can read fish-flavored C++.

### What Stayed C++

A small core stayed in C++ for fish 4.0:

- Some terminal handling glue.
- Legacy builtin implementations that hadn't been ported yet.
- A few platform-specific bits (notably some macOS-specific kqueue code).

The plan is to drive these to zero across the 4.x series. Fish 4.x can compile with both Cargo and CMake (Cargo is preferred); the legacy code is wrapped in `extern "C"` and called from Rust via `bindgen`-generated bindings.

### What Changed for Plugin Authors

**Almost nothing.** This is a deliberate design goal. Fish scripts are interpreted; the language they target is unchanged. The Rust internals are an *implementation detail*. A fish 3.x plugin runs unchanged on fish 4.x, with one caveat: behaviors that depended on undocumented C++ implementation details (e.g., specific evaluation order in some pathological tokenizer cases) may have shifted slightly.

If you wrote your plugin to the documented language, you're fine. If you wrote it to "what fish 3.6 happened to do," you may see edge-case differences.

### What Changed for fish Hackers

If you contribute to fish itself, the codebase is now Rust. Build with:

```bash
cargo build --release
# or, the legacy path that still works in 4.x:
cmake -B build && cmake --build build
```

The crate split lives at the top of `Cargo.toml` and mostly mirrors the file structure described in section 1.

### Performance Claims

The maintainers' published benchmarks for the rewrite:

| Operation | C++ fish 3.7 | Rust fish 4.0 | Delta |
|:---|:---|:---|:---|
| Cold startup | ~50ms | ~35ms | -30% |
| Parse 10K lines of script | ~80ms | ~60ms | -25% |
| `string match -r` over 100K lines | ~120ms | ~90ms | -25% |
| Universal-var file load (100 vars) | ~3ms | ~2ms | -33% |
| Memory at idle | ~12MB | ~9MB | -25% |

These are rough numbers from the public benchmark harness; your mileage will vary by platform and workload.

---

## 17. Idioms at the Internals Depth

### The Fast `config.fish`

```fish
# ~/.config/fish/config.fish

# Bail out fast for non-interactive shells (scripts use #!/usr/bin/env fish
# but don't need any of this).
status is-interactive; or exit

# PATH is a universal variable already; only adjust if necessary.
# Don't repeatedly prepend — that grows the universal var on every shell.
contains /usr/local/bin $fish_user_paths; or set -Ua fish_user_paths /usr/local/bin

# Defer expensive init. starship is fast enough; nvm is not.
command -q starship; and starship init fish | source

# nvm is lazy: only init when 'nvm' or 'node' is invoked.
function nvm
    functions -e nvm
    bass source ~/.nvm/nvm.sh
    nvm $argv
end

function node
    functions -e node
    bass source ~/.nvm/nvm.sh
    node $argv
end
```

The pattern: a function that **deletes itself** on first invocation, runs the expensive init, and re-invokes the now-real command.

### Deferred Initialization Pattern

```fish
function _lazy_init_kubectl
    functions -e _lazy_init_kubectl
    kubectl completion fish | source
end

function kubectl
    _lazy_init_kubectl
    command kubectl $argv
end
```

`command kubectl` bypasses fish's function lookup (which would re-enter `kubectl` infinitely). After the first call, the wrapper has been replaced by the real completion-generated wrapper from `kubectl completion fish`.

### Abbr Over Alias for Visible Commands

The rule of thumb:

- **`abbr`**: anything you'd want to read in your shell history. Git aliases, kubectl shortcuts, docker shortcuts.
- **`alias`**: things that wrap a command transparently and never need to be edited. `alias ls='ls -F --color=auto'` for example.
- **`function`**: anything more than one line, anything with conditional logic.

### Universal-Var-for-Stable-Settings Pattern

Settings that should follow you across sessions and survive logout/login:

```fish
set -Ux EDITOR nvim                          # editor
set -Ux PAGER 'less -R'                      # pager
set -Ux MANPAGER 'sh -c "col -bx | bat -l man -p"'   # man pager
set -Ux fish_greeting ""                     # disable greeting

# Visual settings
set -U fish_color_autosuggestion 555 --italics
set -U fish_pager_color_selected_background --background=blue

# Tool integration
set -Ux FZF_DEFAULT_COMMAND 'fd --type f'
```

Once set, you never re-set them; they live in `fish_variables` and persist.

### The Conditional Keybinding Pattern

```fish
# Only bind if the tool exists
function fish_user_key_bindings
    if command -q fzf
        fzf_key_bindings
    end
    bind \cr _atuin_search   # if atuin is installed
    bind \e\[A history-prefix-search-backward
    bind \e\[B history-prefix-search-forward
end
```

Fish calls `fish_user_key_bindings` once at shell startup; that's where you put `bind` calls.

### Detecting Fish Version

```fish
# Run different code paths based on fish version
if test (string split . -- $version | head -n 1) -ge 4
    # fish 4.x: use new feature
    set --function FOO bar
else
    # fish 3.x: fallback
    set -l FOO bar
end
```

`$version` is a universal variable set by fish at startup containing the running version string.

### Function Composition

```fish
# Compose two filters:
function notes
    cd ~/notes
    and ls *.md
    and grep -l TODO *.md
end

# This is fish's "and-chain": each command runs only if previous succeeded.
# Equivalent to bash:  cd ~/notes && ls *.md && grep -l TODO *.md
```

### The "Stash and Restore" Idiom

```fish
function with_dir --argument-names dir
    set -l prev (pwd)
    cd $dir
    or return
    $argv[2..-1]
    set -l ret $status
    cd $prev
    return $ret
end

with_dir /tmp git status
```

`$argv[2..-1]` is a list slice: elements 2 through end. Used as a command, fish executes it.

---

## 18. Prerequisites

- Familiarity with at least one POSIX shell (bash or zsh) for contrast
- Working knowledge of process model fundamentals: `fork(2)`, `exec(2)`, `waitpid(2)`, file descriptors, pipes, and signals
- Comfort with regex (PCRE2 dialect, used by `string match -r`)
- Basic understanding of UTF-8 (fish is fully UTF-8-aware; bytes vs chars matter for `string sub` and `string length --visible`)
- Willingness to leave POSIX habits at the door — fish's grammar is intentionally divergent
- Conceptual familiarity with event-driven programming for the `--on-event` / `--on-variable` model
- For the Rust internals: passing knowledge of Rust ownership and `Arc<Mutex<...>>` is helpful but not required

---

## 19. Complexity

| Operation | Time | Space | Notes |
|:---|:---|:---|:---|
| Tokenize line | $O(n)$ | $O(n)$ | Linear in line length |
| Parse to AST | $O(n)$ | $O(n)$ | LL(1) grammar, no backtracking |
| Variable scope lookup | $O(d)$ | – | $d$ = scope depth, typically ≤ 10 |
| Universal-var load | $O(V)$ | $O(V \bar{l})$ | $V$ vars, $\bar{l}$ avg length |
| Universal-var write | $O(V)$ | $O(V \bar{l})$ | Atomic rename guarantees consistency |
| `string match -r` | $O(n m)$ worst, $O(n)$ typical | $O(k)$ for $k$ groups | PCRE2 |
| `string replace` | $O(n)$ | $O(n)$ | |
| Function autoload | $O(f)$ first call, $O(1)$ after | – | $f$ = function file size |
| Event dispatch | $O(L)$ | – | $L$ = registered listeners |
| Abbreviation lookup | $O(1)$ amortized | $O(A)$ | Hash table on token |
| History search (prefix) | $O(N)$ | $O(N)$ | $N$ = history entries; cached |
| History search (fuzzy) | $O(N \bar{l})$ | $O(N)$ | Subsequence ranking |
| Tab completion | $O(\|p\| + \|results\|)$ | $O(T)$ trie | $p$ = prefix |
| Job spawn (pipeline of $S$ stages) | $S$ × `fork` + `exec` | $O(S)$ | One process per stage |
| `fork(2)` + `execve(2)` cost | ~1–5ms (Linux), 2–10ms (macOS) | – | OS-dependent |
| Cold start | ~30–50ms | ~10MB | Down ~30% in fish 4.x vs 3.7 |
| First prompt render | ~10–30ms | – | Dominated by `fish_prompt` complexity |

---

## 20. See Also

- fish (the practical sheet — `cs shell/fish`)
- bash (POSIX baseline, scripting target)
- zsh (the older interactive-shell-with-features, contrast point)
- nushell (the structured-data alternative)
- polyglot (the cross-language idioms reference)

---

## 21. References

- Official documentation: https://fishshell.com/docs/current/
- Design philosophy: https://fishshell.com/docs/current/design.html
- Source repository: https://github.com/fish-shell/fish-shell
- Tutorial: https://fishshell.com/docs/current/tutorial.html
- Language reference: https://fishshell.com/docs/current/language.html
- Interactive use: https://fishshell.com/docs/current/interactive.html
- The Rust port discussion: https://github.com/fish-shell/fish-shell/issues/8420
- Fish 4.0 release notes: https://github.com/fish-shell/fish-shell/blob/master/CHANGELOG.rst
- `fisher` package manager: https://github.com/jorgebucaran/fisher
- `bass` (run bash inside fish): https://github.com/edc/bass
- `tide` prompt: https://github.com/IlanCosman/tide
- `fzf.fish` integration: https://github.com/PatrickF1/fzf.fish
- The `string` builtin reference: https://fishshell.com/docs/current/cmds/string.html
- The `argparse` builtin reference: https://fishshell.com/docs/current/cmds/argparse.html
- Universal variables explainer: https://fishshell.com/docs/current/language.html#shell-variables
- Event handling: https://fishshell.com/docs/current/language.html#event-handlers
- Abbreviations: https://fishshell.com/docs/current/cmds/abbr.html
- The deprecated/Rust-port FAQ: https://github.com/fish-shell/fish-shell/wiki/Rust-port-progress
- The `awesome-fish` curated list: https://github.com/jorgebucaran/awesome.fish
