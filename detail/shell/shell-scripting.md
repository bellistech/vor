# The Internals of POSIX Shell Scripting — Execution Model, Word Splitting, and Portability

> *The POSIX shell is not a programming language; it is a job-control DSL with programming features bolted on after the fact. Every quirk — the unquoted-variable trap, the `set -e` ambiguity, the heredoc-in-pipe quirk, the `for f in $(ls)` disaster — derives from the shell's original purpose: a glue layer between processes. Understanding the four-phase command processing pipeline (tokenization → expansion → word-splitting → pathname-expansion) is the difference between a script that works and a script that works on every input.*

---

## 1. The POSIX Shell Execution Model

### 1.1 The Four-Phase Command Processing

The POSIX shell processes every command line through a strictly ordered pipeline. The order is normative — it is specified in IEEE Std 1003.1 (POSIX.1-2017), Section 2.6, and any deviation is a bug or a documented extension. Misunderstanding the order produces nearly every shell-scripting bug you have ever written.

```
Input: rm $files
       │
       ▼
Phase 1: Tokenization
       Split into tokens by metacharacters: <space>, <tab>, |, &, ;, (, ), <, >, newline
       │
       ▼
Phase 2: Expansion
       Expand:
         a. Brace expansion       (bashism — not POSIX)
         b. Tilde expansion       (~ → $HOME, ~user → user's home)
         c. Parameter expansion   ($var, ${var:-def}, ${var#prefix})
         d. Command substitution  ($(cmd), `cmd`)
         e. Arithmetic expansion  ($((expr)))
         f. Process substitution  (<(cmd), >(cmd))   (bashism — not POSIX)
       │
       ▼
Phase 3: Word splitting
       Split fields on $IFS (default: <space><tab><newline>)
       Skipped for: quoted strings ("...", '...'), $* in some contexts
       │
       ▼
Phase 4: Pathname expansion (globbing)
       Expand: *, ?, [...]
       Each resulting word is a glob pattern; no-match → literal (POSIX) or empty (with bash's nullglob)
       │
       ▼
Phase 5: Quote removal
       Strip quoting characters not produced by expansion
       │
       ▼
Execute
```

### 1.2 Why the Order Matters

Consider the canonical bug:

```bash
files="report 1.txt report 2.txt"
rm $files
```

The user *intends* to delete two files. What actually happens:

1. **Tokenization**: produces tokens `rm` and `$files`.
2. **Parameter expansion**: `$files` becomes the literal string `report 1.txt report 2.txt`.
3. **Word splitting**: with `IFS` at its default (`$' \t\n'`), the string splits into four words: `report`, `1.txt`, `report`, `2.txt`.
4. **Pathname expansion**: each word is treated as a glob; if `1.txt` exists it is kept, otherwise left literal.
5. **Execution**: `rm` receives four arguments and tries to delete four files.

Quoting changes this:

```bash
rm "$files"
```

`"$files"` is a quoted parameter expansion. Word splitting and pathname expansion are *both* skipped for quoted strings, so `rm` receives a single argument: `report 1.txt report 2.txt`. That's still wrong (the file does not exist as a single name), but it fails predictably instead of catastrophically.

### 1.3 Compatibility Across Shells

POSIX is the lowest common denominator. Real implementations diverge:

| Shell | Conformance | Notes |
|:------|:------------|:------|
| `dash` | Strict POSIX | Default `/bin/sh` on Debian/Ubuntu |
| `busybox sh` | Strict POSIX subset | Default on Alpine and most embedded Linux |
| `ash` (Almquist) | Strict POSIX | Ancestor of dash; FreeBSD `/bin/sh` |
| `ksh93` | POSIX + extensions | Real-world reference for POSIX semantics |
| `bash` | POSIX + many extensions | Default on most desktop Linux; macOS through 10.14 |
| `bash --posix` | POSIX mode | Disables most bashisms but not all |
| `zsh` | Mostly POSIX | Many incompatibilities by default |
| `zsh -o sh_emulation` (or invoked as `sh`) | POSIX | Approximates POSIX |
| `yash` | Strict POSIX with rigor | Yet Another Shell — useful as a portability oracle |

### 1.4 The Test Ladder

When writing portable scripts, you cannot trust your mental model — you must test. The standard ladder, from most strict to most permissive:

```bash
# Stage 1: dash — Debian's POSIX shell
dash ./script.sh

# Stage 2: BusyBox sh — embedded-Linux reality check
busybox sh ./script.sh

# Stage 3: bash --posix — bash with POSIX mode forced
bash --posix ./script.sh

# Stage 4: bash — the lingua franca of CI
bash ./script.sh

# Stage 5: zsh emulate sh — macOS users
zsh -c 'emulate sh; source ./script.sh'
```

If your script passes dash, it will pass nearly anywhere. If it fails dash but passes bash, you have unknowingly used a bashism.

---

## 2. Tokenization and Command Substitution

### 2.1 Lexer Rules

The shell tokenizer is recursive-descent and context-sensitive. The metacharacters that terminate a token are:

```
| & ; ( ) < > <space> <tab> <newline>
```

A "word" is a maximal sequence of non-metacharacter characters, possibly including quoted segments. Quotes (`'`, `"`, `\`) suppress the lexer's special meaning of metacharacters within them, but the tokenizer still tracks the quote state.

```bash
echo 'hello | world'   # one argument: hello | world
echo "hello | world"   # one argument: hello | world
echo  hello | world    # pipeline: echo hello | world
```

### 2.2 Command Substitution: `$(...)` vs Backticks

POSIX provides two forms. They are *almost* equivalent.

```bash
# Modern form — recursive-descent parsing
result=$(grep "$pattern" "$file")

# Legacy form — backtick parsing
result=`grep "$pattern" "$file"`
```

The modern `$(...)` form parses recursively: the shell tokenizes the body using its full lexer, so quoting and nesting "just work". The backtick form uses a different, simpler parsing strategy that requires escaping the inner backticks and quotes:

```bash
# Modern: trivially nests
nested=$(echo $(echo $(date)))

# Legacy: ugly escaping
nested=`echo \`echo \\\`date\\\`\``
```

**Always prefer `$(...)`** unless you are targeting Bourne shells from the 1980s — modern POSIX (and every shell from the last 25 years) supports it.

### 2.3 Quote-State Tracking

The lexer maintains a quote-state machine. Within `"..."`, `$`, `` ` ``, and `\` retain their meaning; everything else is literal. Within `'...'`, *nothing* is special — not even `\`. This is why single-quoted strings cannot contain literal single quotes:

```bash
# Trying to embed a single quote in a single-quoted string:
echo 'It's broken'      # SYNTAX ERROR — the second ' closes the quote
echo 'It'\''s fixed'    # close, escape, reopen
echo "It's fine"        # use double quotes
echo "It"\''s mixed'    # arbitrary mixing
```

### 2.4 Heredoc Parsing

Heredocs are parsed in three modes:

```bash
# Mode 1: Unquoted delimiter — full expansion
cat <<EOF
User: $USER
Date: $(date)
EOF

# Mode 2: Quoted delimiter — NO expansion
cat <<'EOF'
The literal text $USER and $(date)
EOF

# Mode 3: Hyphen — strip leading TABS (not spaces) for indentation
cat <<-EOF
	indented but the tabs are stripped
	preserving the heredoc body
	EOF
```

The quoting can be on any character of the delimiter — `<<\EOF`, `<<"EOF"`, and `<<'EOF'` all suppress expansion.

### 2.5 The `$'C-style escape'` Bashism

Bash and zsh provide ANSI-C quoting:

```bash
# Bashism — NOT POSIX
newline=$'\n'
tab=$'\t'
escape=$'\e'
unicode=$'é'    # é
```

This is convenient but not portable. The POSIX equivalent uses `printf`:

```bash
# Portable
newline=$(printf '\n.')
newline=${newline%.}    # strip the trailing dot we added to preserve the newline
tab=$(printf '\t.')
tab=${tab%.}
```

The trailing-dot trick is necessary because command substitution strips trailing newlines.

---

## 3. Word Splitting — The Canonical Trap

### 3.1 IFS — The Internal Field Separator

The shell's word-splitting is governed by a single variable: `IFS`. By default, `IFS=$' \t\n'` — space, tab, newline. After parameter and command substitution, the resulting string is split into fields on any character in `IFS`, *unless the substitution was inside double quotes*.

```bash
files="a.txt b.txt c.txt"
rm $files       # splits to: rm a.txt b.txt c.txt   (3 args)
rm "$files"     # one arg:   rm "a.txt b.txt c.txt" (1 arg)
```

### 3.2 The Canonical Example

This is the bug Wooledge's BashFAQ describes as "the most common shell scripting mistake":

```bash
files="report 1.txt report 2.txt"
rm $files       # WRONG — deletes 4 files: report, 1.txt, report, 2.txt
```

Two filenames containing spaces become four separate arguments. The fix is to never store filenames in a string; use either an array (bashism) or careful per-file iteration with `find`.

### 3.3 IFS-Only Splitting Trick

You can override IFS to split on a non-default character, such as newline only:

```bash
# Read a list of file paths, one per line
old_ifs=$IFS
IFS=$'\n'
for file in $(cat list.txt); do
    process "$file"
done
IFS=$old_ifs
```

This is *acceptable* but still not robust — filenames containing newlines (yes, they exist on Unix) still break it. The truly safe form uses `read`:

```bash
while IFS= read -r file; do
    process "$file"
done < list.txt
```

### 3.4 Disabling Globbing with `set -f`

If you want word splitting but not pathname expansion:

```bash
set -f          # disable globbing (-o noglob)
patterns="*.txt *.md"
for p in $patterns; do
    echo "$p"   # prints "*.txt" and "*.md" as literals
done
set +f          # restore
```

### 3.5 Quoting and Word Splitting Interaction

The complete rule: **double-quoting suppresses both word splitting and pathname expansion**. Single-quoting suppresses everything including parameter expansion. Unquoted expansion triggers word splitting *and* pathname expansion in that order.

| Form | Word split? | Glob expand? | Param expand? |
|:-----|:-----------:|:------------:|:-------------:|
| `$var` | yes | yes | yes |
| `"$var"` | no | no | yes |
| `'$var'` | no | no | no (literal `$var`) |
| `\$var` | no | no | no (literal `$var`) |

The single most powerful rule of shell programming: **quote every variable expansion unless you have a specific reason not to.**

---

## 4. Pathname Expansion / Globbing

### 4.1 When It Happens

Pathname expansion runs *after* word splitting. Each word produced by splitting is treated independently as a glob pattern. If the pattern matches one or more files, it is replaced by the (sorted) list of matches. If it matches nothing, POSIX leaves the literal word unchanged.

```bash
ls *.txt
# If a.txt and b.txt exist:
#   tokenized: ls *.txt
#   expanded:  ls a.txt b.txt
# If no .txt files exist:
#   POSIX: ls *.txt   (literal)
#   bash with `shopt -s nullglob`: ls   (no args)
#   bash with `shopt -s failglob`: error and exit
```

### 4.2 Glob Operators

| Pattern | Matches |
|:--------|:--------|
| `*` | Any string (including empty), but not leading `.` |
| `?` | Any single character, but not leading `.` |
| `[abc]` | Any one of `a`, `b`, `c` |
| `[a-z]` | Any character in the range (locale-dependent — beware) |
| `[!abc]` | Any character not `a`, `b`, or `c` (POSIX) |
| `[^abc]` | Same as above (bash extension; not POSIX) |

### 4.3 The Dotglob Behavior

By default, leading-dot files (hidden files) are *not* matched by `*` or `?`. This is a deliberate convention so `rm *` does not delete `.bashrc`.

```bash
# Default behavior:
ls *           # excludes .bashrc, .config, etc.
ls .*          # includes . and .. — the parent-directory trap

# Bash extension:
shopt -s dotglob
ls *           # NOW includes hidden files (but NOT . or ..)
```

The `.*` glob trap is real:

```bash
# DO NOT do this:
rm -rf .*      # bash matches .. — recursive parent-directory deletion
```

### 4.4 Disabling Globbing

```bash
set -f          # POSIX: disable pathname expansion (also: set -o noglob)
echo *          # prints literal *
set +f          # re-enable
```

### 4.5 Unmatched-Glob Behavior — POSIX vs Bash

POSIX leaves unmatched globs as literals:

```bash
# In a directory with no .txt files:
for f in *.txt; do echo "$f"; done
# POSIX prints: *.txt
# This breaks naive loops because "*.txt" looks like a single file.
```

Bash provides `nullglob` and `failglob` as remedies:

```bash
shopt -s nullglob
for f in *.txt; do echo "$f"; done
# Loop body never runs if no matches.

shopt -s failglob
for f in *.txt; do echo "$f"; done
# Script exits with an error if no matches.
```

The POSIX-portable workaround is to test for existence:

```bash
for f in *.txt; do
    [ -e "$f" ] || continue   # skip the literal pattern if no matches
    process "$f"
done
```

---

## 5. Parameter Expansion — POSIX Subset

### 5.1 The Core POSIX Operators

POSIX defines a precise set of parameter-expansion operators. They are pure shell — no fork — and replace many calls to `sed`, `awk`, or `expr`.

| Form | Meaning |
|:-----|:--------|
| `${var}` | Value of `var` (braces disambiguate) |
| `${var:-default}` | If `var` is unset or empty, use `default` |
| `${var-default}` | If `var` is unset (but maybe empty), use `default` |
| `${var:=value}` | If `var` is unset or empty, assign `value` and use it |
| `${var=value}` | If `var` is unset, assign `value` and use it |
| `${var:?error}` | If `var` is unset or empty, write `error` to stderr and exit |
| `${var?error}` | If `var` is unset, write `error` and exit |
| `${var:+alt}` | If `var` is set and non-empty, use `alt`; else empty |
| `${var+alt}` | If `var` is set, use `alt`; else empty |
| `${#var}` | Length of `var` in bytes (POSIX) — *not* characters |
| `${var#prefix}` | Strip shortest prefix matching `prefix` |
| `${var##prefix}` | Strip longest prefix matching `prefix` |
| `${var%suffix}` | Strip shortest suffix matching `suffix` |
| `${var%%suffix}` | Strip longest suffix matching `suffix` |

### 5.2 Real Examples

```bash
# Default values
log_level=${LOG_LEVEL:-info}

# Mandatory variable
: "${DATABASE_URL:?must set DATABASE_URL}"

# Get the file extension (longest prefix removal up to last dot)
file="archive.tar.gz"
echo "${file##*.}"      # gz
echo "${file%.*}"       # archive.tar
echo "${file%%.*}"      # archive

# Get directory and basename without forking dirname/basename
path="/usr/local/bin/cs"
echo "${path%/*}"       # /usr/local/bin
echo "${path##*/}"      # cs

# Strip protocol from URL
url="https://example.com/path"
echo "${url#*://}"      # example.com/path
echo "${url%%://*}"     # https
```

### 5.3 Bash Extensions

Bash extends parameter expansion considerably:

| Form | POSIX? | Meaning |
|:-----|:------:|:--------|
| `${var/pat/rep}` | bashism | Replace first match of `pat` with `rep` |
| `${var//pat/rep}` | bashism | Replace all matches |
| `${var/#pat/rep}` | bashism | Replace if `pat` matches start |
| `${var/%pat/rep}` | bashism | Replace if `pat` matches end |
| `${!var}` | bashism | Indirect — expand the variable named by `$var` |
| `${var^}` | bashism | Capitalize first character |
| `${var^^}` | bashism | Uppercase all |
| `${var,}` | bashism | Lowercase first |
| `${var,,}` | bashism | Lowercase all |
| `${var:offset:length}` | bashism | Substring (zero-indexed) |

These are convenient but break in dash. The portable equivalents use `tr`, `sed`, or `awk`:

```bash
# Bash:
upper=${var^^}

# POSIX:
upper=$(printf '%s' "$var" | tr '[:lower:]' '[:upper:]')
```

Note: `[:lower:]`/`[:upper:]` in `tr` are POSIX character classes and work everywhere, but their *behavior* depends on the current locale.

---

## 6. Subshells and the Process Tree

### 6.1 The Subshell Operator

`(cmd)` runs `cmd` in a *subshell* — a forked copy of the shell process. Variables, current directory, and shell options assigned in the subshell do not affect the parent.

```bash
x=1
(x=2; echo "in subshell: $x")     # prints: in subshell: 2
echo "after subshell: $x"          # prints: after subshell: 1
```

This is fundamentally different from `{ cmd; }`, which is a *grouping* construct that runs in the current shell:

```bash
x=1
{ x=2; echo "in group: $x"; }      # prints: in group: 2
echo "after group: $x"              # prints: after group: 2
```

### 6.2 The Export Keyword

Subshells inherit *exported* variables. Non-exported variables are local to the parent shell only.

```bash
x=1
y=2
export y
sh -c 'echo "x=$x y=$y"'    # prints: x= y=2
```

A common pattern for invoking external commands with environment overrides:

```bash
DEBUG=1 LOG_LEVEL=trace ./my-script.sh
```

This sets `DEBUG` and `LOG_LEVEL` in the environment of the child process only.

### 6.3 The Pipeline Subshell Trap

This is the most insidious shell bug. Each component of a pipeline runs in its own subshell *in bash by default* (zsh has different defaults).

```bash
count=0
seq 1 5 | while read n; do
    count=$((count + 1))
done
echo "$count"    # prints: 0 — NOT 5
```

Because the `while` loop is the right side of a pipe, it ran in a subshell. The `count` variable was incremented in the subshell but the parent's `count` is untouched.

Workarounds:

```bash
# Workaround 1: shopt -s lastpipe (bash 4.2+)
shopt -s lastpipe
count=0
seq 1 5 | while read n; do count=$((count + 1)); done
echo "$count"    # 5

# Workaround 2: process substitution (bashism)
count=0
while read n; do count=$((count + 1)); done < <(seq 1 5)
echo "$count"    # 5

# Workaround 3: heredoc with command substitution
count=0
while read n; do count=$((count + 1)); done <<EOF
$(seq 1 5)
EOF
echo "$count"    # 5

# Workaround 4: portable alternative — capture and process
data=$(seq 1 5)
count=0
for n in $data; do count=$((count + 1)); done
echo "$count"    # 5
```

### 6.4 Process Substitution

Bash and zsh provide `<(cmd)` and `>(cmd)`:

```bash
diff <(sort file1.txt) <(sort file2.txt)
```

This expands to a path (typically `/dev/fd/63`) connected via a pipe to `cmd`. Not POSIX — dash will refuse it. The portable equivalent uses temporary files:

```bash
sort file1.txt > /tmp/a.$$
sort file2.txt > /tmp/b.$$
diff /tmp/a.$$ /tmp/b.$$
rm /tmp/a.$$ /tmp/b.$$
```

### 6.5 The `exec` Builtin

`exec` *replaces* the current shell process with the given command — no fork.

```bash
# Replace this shell with the editor; never returns
exec vim

# Re-exec the script with environment overrides
exec env CLEAN=1 PATH=/usr/local/bin:/usr/bin "$0" "$@"

# Redirect this shell's stdout for all subsequent commands
exec > /tmp/log.txt
echo "this goes to the log"

# Open a new file descriptor
exec 3< /etc/passwd
read -r line <&3
exec 3<&-    # close fd 3
```

`exec` is the foundation of `setsid`, `nohup`, `chroot`, and most "wrapper" scripts.

---

## 7. set -e — The Behavioral Surface

### 7.1 The Promise vs The Reality

`set -e` (also written `set -o errexit`) instructs the shell to exit immediately if any command fails. In practice, what counts as "fails" is surprisingly narrow, and the rules are inconsistent across shells.

The POSIX standard has been clarified over time — POSIX-2017 made some rules normative — but real shells implement subtle variations.

### 7.2 What Triggers Exit

```bash
set -e
false             # exits — non-zero exit status
ls /nonexistent   # exits — ls returns non-zero
```

### 7.3 What Does NOT Trigger Exit

```bash
set -e

# 1. Command in a conditional context
if false; then echo "yes"; fi    # does NOT exit — false is in if-condition
while false; do break; done       # does NOT exit
until false; do break; done       # does NOT exit

# 2. Command before && or ||
false && echo "yes"               # does NOT exit
false || echo "fallback"          # does NOT exit
true && false                     # does NOT exit on the false (it's part of &&)
                                  # but DOES exit if last in the chain returns non-zero

# 3. Negated command
! false                            # does NOT exit

# 4. Pipeline non-final stages (without pipefail)
false | true                       # does NOT exit — pipeline exit is true's
```

### 7.4 The Function-Call Surprise

This catches everyone. With `set -e`, a command that fails *inside a function called as part of an `&&` chain* does not trigger exit, even if `set -e` is in effect.

```bash
set -e
fail_inside() {
    false           # would normally exit
    echo "after"    # but reachable here
}
fail_inside && echo "outer ok"
# Function continues executing past `false` because the function call
# is part of an && chain.
```

This is the source of countless "but I have `set -e`!" debugging sessions. The behavior is documented in BashFAQ #105 (Wooledge) and is one reason experienced shell programmers consider `set -e` unreliable.

### 7.5 Explicit Error Handling

For mission-critical reliability, prefer explicit checks:

```bash
# Old-school but bulletproof
cmd || { echo "cmd failed: $?" >&2; exit 1; }

# Or with a helper
die() { echo "$@" >&2; exit 1; }
cmd || die "cmd failed"
```

This works in every shell ever written and is unambiguous about when the script exits.

### 7.6 The BashFAQ Reference

Wooledge's [BashFAQ #105](https://mywiki.wooledge.org/BashFAQ/105) is the definitive treatment. It catalogs every situation where `set -e` does or does not behave as expected. The summary: `set -e` is a useful safety net, but it is not a substitute for explicit error handling.

---

## 8. Pipelines and Exit Status

### 8.1 POSIX Pipeline Semantics

A pipeline `cmd1 | cmd2 | cmd3` runs all three commands in parallel, with `cmd1`'s stdout connected to `cmd2`'s stdin and `cmd2`'s stdout to `cmd3`'s stdin. The shell waits for *all* commands to finish, then reports the exit status of the *last* command:

```bash
false | true     # exit status: 0 (true's)
true | false     # exit status: 1 (false's)
```

This is a common source of bugs:

```bash
# Suppose grep prints to stdout, but we want to know if it found anything
grep "pattern" file.txt | wc -l
# Exit status is wc's (always 0); grep's is lost.
```

### 8.2 Bash's `pipefail` Option

Bash provides `set -o pipefail`:

```bash
set -o pipefail
false | true     # exit status: 1 (the first non-zero in the pipeline)
true | false     # exit status: 1
true | true      # exit status: 0
```

This is the canonical "did anything in this pipeline fail" mode. POSIX-2024 standardized `pipefail`, so this is now portable in modern shells.

### 8.3 The PIPESTATUS Array (Bashism)

Bash provides full per-stage exit status via `${PIPESTATUS[@]}`:

```bash
false | true | true
echo "exits: ${PIPESTATUS[@]}"    # exits: 1 0 0
```

zsh has `$pipestatus` (lowercase, naturally indexed differently). Not POSIX.

### 8.4 Did Anything Fail?

```bash
set -o pipefail    # bash, modern POSIX
if ! grep "pattern" file.txt | sort | uniq; then
    echo "pipeline failed somewhere" >&2
fi
```

Without `pipefail`, the portable check is uglier:

```bash
{ grep "pattern" file.txt; echo "grep_exit=$?" >> /tmp/exits; } | sort
```

### 8.5 SIGPIPE Handling

When the *consumer* of a pipe exits, the *producer* receives `SIGPIPE` on its next write. By default, this terminates the producer with exit status 141 (128 + 13, where 13 is SIGPIPE).

```bash
yes | head -1
# yes runs forever, prints "y\n" indefinitely
# head reads one line, exits
# yes's next write produces SIGPIPE
# yes exits with status 141 (or 128+13)
```

This is *correct behavior* and how unix pipelines self-throttle. But it can produce surprising exit statuses with `pipefail`:

```bash
set -o pipefail
yes | head -1
echo $?    # 141 — SIGPIPE
```

The remedy is to ignore SIGPIPE explicitly or to filter the exit status:

```bash
yes | head -1
status=${PIPESTATUS[0]}
[ "$status" -eq 0 ] || [ "$status" -eq 141 ]
```

---

## 9. Redirection Mechanics

### 9.1 The Redirection Operators

| Operator | Meaning | POSIX? |
|:---------|:--------|:------:|
| `> file` | Redirect stdout to `file` (truncate) | yes |
| `>> file` | Redirect stdout to `file` (append) | yes |
| `< file` | Redirect stdin from `file` | yes |
| `2> file` | Redirect stderr to `file` | yes |
| `2>&1` | Duplicate fd 2 to fd 1 (merge stderr to stdout) | yes |
| `>&2` | Duplicate fd 1 to fd 2 (write to stderr) | yes |
| `&> file` | Redirect stdout AND stderr to `file` | bashism |
| `>& file` | Same as `&>` | bashism (csh-derived) |
| `\|&` | Pipe both stdout and stderr | bashism |
| `<<EOF` | Heredoc | yes |
| `<<-EOF` | Heredoc with leading-tab stripping | yes |
| `<<<"str"` | Here-string | bashism |
| `<>` | Open for read+write | yes |
| `n<&m` | Duplicate fd m to n for reading | yes |
| `n>&m` | Duplicate fd m to n for writing | yes |
| `n<&-` | Close fd n | yes |

### 9.2 Order Matters: 2>&1 vs >file

The classic confusion:

```bash
# Wrong: stderr goes to old stdout (terminal), stdout goes to file
cmd 2>&1 > file.txt
# Result: file.txt has stdout; stderr still on terminal

# Right: stdout is redirected first, then stderr is duplicated to (now-redirected) stdout
cmd > file.txt 2>&1
# Result: file.txt has both stdout and stderr
```

The mental model: redirections are evaluated **left to right**, and `&n` captures the *current* destination of fd `n` at that moment. `2>&1` means "make fd 2 point wherever fd 1 currently points". If fd 1 still points to the terminal, that's where stderr goes.

### 9.3 Heredocs in Detail

```bash
# Standard heredoc
cat <<EOF
expanded: $USER
EOF

# Quote the delimiter to disable expansion
cat <<'EOF'
literal: $USER
EOF

# Hyphen strips leading TABS (not spaces!)
cat <<-EOF
	tab-indented body
	is stripped of its tabs
	EOF

# Heredoc as input to a command
sort <<EOF
banana
apple
cherry
EOF

# Multiple heredocs
cat <<HEAD - <<TAIL
header
HEAD
between
TAIL
```

The TAB-not-space behavior of `<<-` is a frequent gotcha. Many editors expand tabs to spaces by default, breaking the heredoc.

### 9.4 Here-Strings (Bashism)

```bash
# Bashism — feeds a string as stdin
read -r var <<<"hello world"
echo "$var"     # hello world

# Equivalent (sort of) using a heredoc
read -r var <<EOF
hello world
EOF

# POSIX equivalent — but creates a subshell
echo "hello world" | read -r var    # broken by pipeline subshell
```

### 9.5 /dev/null and Friends

```bash
cmd > /dev/null              # discard stdout
cmd 2> /dev/null             # discard stderr
cmd > /dev/null 2>&1         # discard both
cmd > /dev/stderr            # /dev/stderr is a Linux convenience symlink
cmd >&2                      # POSIX way to write to stderr
```

### 9.6 /dev/tcp/host/port (Bashism)

Bash has special-case handling for `/dev/tcp/host/port` and `/dev/udp/host/port`:

```bash
# Bashism: a TCP connection as a file
exec 3<>/dev/tcp/example.com/80
printf 'GET / HTTP/1.0\r\nHost: example.com\r\n\r\n' >&3
cat <&3
```

This is *not* a real file — it's parsed by bash and becomes a `socket(2)+connect(2)` syscall. Not POSIX. dash will try to open the literal path `/dev/tcp/...` and fail.

---

## 10. Job Control and Signals

### 10.1 Background Jobs

```bash
sleep 10 &
echo "started: $!"     # $! is the PID of the most recent background job
wait $!                # wait for it
echo "done"

# Multiple background jobs
sleep 5 & sleep 3 & sleep 7 &
wait                   # wait for all background jobs
```

### 10.2 The `trap` Builtin

`trap` registers a handler for a signal. The standard signals:

| Signal | Default | Common Use |
|:-------|:--------|:-----------|
| `EXIT` | (pseudo) | Cleanup on script exit |
| `INT` (2) | terminate | Ctrl-C |
| `TERM` (15) | terminate | Polite "please exit" |
| `HUP` (1) | terminate | Terminal hangup |
| `USR1`/`USR2` | terminate | User-defined signals |
| `CHLD` | (ignore) | Child process state changed |
| `PIPE` (13) | terminate | Wrote to a closed pipe |

```bash
# Cleanup on exit
trap 'rm -f "$tmpfile"' EXIT

# Handle Ctrl-C explicitly
trap 'echo "interrupted" >&2; exit 130' INT

# Reset to default
trap - INT

# Ignore a signal
trap '' INT
```

### 10.3 Asynchronous Trap Delivery

Traps are delivered asynchronously *between commands*. While the shell is waiting for a child via `wait`, a signal interrupts the wait, the trap runs, and `wait` returns with status 128+N.

```bash
trap 'echo "caught"' USR1
sleep 1000 &
pid=$!
kill -USR1 $$        # send USR1 to ourselves
wait $pid
```

### 10.4 Async-Signal-Safety

A subtle but critical reality: signal handlers in C must use only async-signal-safe functions. The shell's `trap` handlers are documented to defer execution until the current command finishes, *but* this guarantee is implementation-dependent. In practice:

- Inside a `trap` handler, prefer simple operations (assignments, `printf` to a file).
- Avoid operations that themselves might be interrupted (subshells, nested traps).
- For long-running scripts, set traps that just set a flag, then check the flag in your main loop.

```bash
caught=0
trap 'caught=1' INT

while :; do
    if [ "$caught" -eq 1 ]; then
        echo "exiting cleanly"
        break
    fi
    do_work
done
```

### 10.5 SIGCHLD and `wait -n`

Bash 4.3+ provides `wait -n` to wait for *any* one job to finish:

```bash
# Run 4 jobs, but only 2 in parallel
for i in 1 2 3 4 5 6 7 8; do
    work "$i" &
    [ "$(jobs -p | wc -l)" -ge 2 ] && wait -n
done
wait    # final cleanup
```

POSIX `wait` (without `-n`) waits for *all* jobs, which is much less useful for a worker pool.

---

## 11. Reading Files Safely

### 11.1 The Canonical Incantation

```bash
while IFS= read -r line; do
    process "$line"
done < file.txt
```

Every word of this is load-bearing:

| Element | Why |
|:--------|:----|
| `IFS=` | Prevents `read` from trimming leading/trailing IFS whitespace from the line |
| `-r` | Prevents `read` from interpreting `\` as an escape (so `\n` stays as 2 chars) |
| `"$line"` | Quotes the variable to prevent word splitting and globbing |
| `< file.txt` | Redirects file as stdin — no fork, no subshell |

Without `IFS=`, leading and trailing spaces are stripped. Without `-r`, lines ending in backslash are joined with the next line. Without quotes around `$line`, the value is split and globbed.

### 11.2 The No-Trailing-Newline Gotcha

POSIX defines a "text file" as having a final newline. Many editors enforce this; many real-world files do not. `read` returns non-zero (EOF) when it reads a partial line at end-of-file *that has no terminating newline* — and the partial line is still placed in the variable.

```bash
# Naive loop:
while IFS= read -r line; do
    process "$line"
done < file.txt
# If file.txt's last line has no newline, this DOES NOT process it.

# Correct loop:
while IFS= read -r line || [ -n "$line" ]; do
    process "$line"
done < file.txt
```

The `|| [ -n "$line" ]` clause says "if read failed but line is still non-empty, process it". This is the standard idiom for "read every line including a final unterminated one".

### 11.3 readarray / mapfile (Bashism)

Bash 4+ provides `mapfile` (also `readarray`):

```bash
mapfile -t lines < file.txt
echo "${#lines[@]}"      # line count
echo "${lines[0]}"        # first line
```

This is much faster than a `while read` loop because it's a single builtin call. Not POSIX; not in dash.

### 11.4 read with Multiple Variables

```bash
# Split on IFS
echo "alice 30 engineer" | { IFS=' ' read -r name age role; echo "$role"; }

# Split on : like /etc/passwd
while IFS=: read -r user pw uid gid name home shell; do
    echo "$user has uid $uid, shell $shell"
done < /etc/passwd
```

---

## 12. Iterating Files Safely

### 12.1 The Glob-for-Loop is Safe

This is the surprising-to-novices fact:

```bash
for f in *.txt; do
    process "$f"
done
```

This **works correctly** for filenames containing spaces, tabs, newlines, or any other character. Why? Because pathname expansion produces a list of arguments *before* word splitting. The shell knows `*.txt` is a glob, expands it to the actual filenames (each as a separate argument), and the `for` loop iterates over them as-is. Word splitting does not apply because the arguments are not strings being split — they are already separated arguments.

### 12.2 The `for f in $(ls)` Disaster

```bash
# THIS IS BROKEN
for f in $(ls); do
    process "$f"
done
```

Why this is broken:

1. `$(ls)` produces a single newline-separated string.
2. Word splitting kicks in (default IFS includes space + tab + newline).
3. Filenames with spaces are split into multiple arguments.
4. *Then* pathname expansion runs on each resulting word, so a file named `*` would expand to "all files".

There is no way to make this work safely. Use a glob or `find`.

### 12.3 find with -print0 and xargs -0

```bash
find . -name '*.txt' -print0 | xargs -0 process
```

`-print0` makes `find` separate filenames with null bytes (which cannot appear in filenames). `xargs -0` reads null-separated input. This is the canonical "process all files matching a find" pattern. Not POSIX strictly — `-print0` and `-0` are GNU extensions, but they are present on all modern systems.

### 12.4 The Strictly-Portable Form: -exec sh -c '...' sh {} +

For pure-POSIX scripts:

```bash
find . -name '*.txt' -exec sh -c '
    for f; do
        process "$f"
    done
' sh {} +
```

The `{} +` form passes as many filenames as possible per invocation (like xargs). The `'sh'` after the script is `$0` for the inline shell. The `for f; do` (without `in $@`) defaults to iterating positional parameters.

### 12.5 The Locale Problem

Filenames on Linux are arbitrary byte sequences (NUL is the only forbidden byte). Filenames are not necessarily valid UTF-8. The shell's behavior depends on `LC_CTYPE`:

```bash
# A file named with high-bit bytes
touch "$(printf 'caf\xe9.txt')"

# In LC_CTYPE=en_US.UTF-8, this file's name is invalid UTF-8
# Globs may or may not match it depending on the shell

# Force C locale for predictable byte-level handling
LC_ALL=C sh ./script.sh
```

For scripts that process file lists from untrusted sources, set `LC_ALL=C` at the top. This ensures byte-level matching, predictable sorting, and no UTF-8 surprises.

---

## 13. Functions and Variable Scoping

### 13.1 POSIX Function Definition

POSIX defines exactly one function syntax:

```bash
greet() {
    echo "Hello, $1"
}
greet world
```

Note: no `function` keyword (that's a bashism), no parameter names in the parens (always `()`).

Bash also accepts:

```bash
function greet { echo "Hello, $1"; }    # bashism
function greet() { echo "Hello, $1"; }  # bashism + POSIX combined (don't)
```

The portable form is `name() { ... }`. Use it.

### 13.2 The `local` Keyword

`local` is *not* POSIX. It is implemented by bash, dash, ksh, and zsh, but the precise semantics differ. Most working scripts can rely on it; a strict POSIX script cannot.

```bash
# Most shells
greet() {
    local name="$1"
    echo "Hello, $name"
}

# Pure POSIX — use a subshell to scope
greet() (
    name="$1"
    echo "Hello, $name"
)
```

The `( ... )` body makes the function run in a subshell, scoping all variable assignments. The cost: subshell setup overhead and the inability to set variables in the caller.

### 13.3 Dynamic Scoping

Bash uses *dynamic* scoping — a function sees the variables of its caller, not its lexical context.

```bash
inner() {
    echo "$x"    # sees outer's x
}

outer() {
    local x=42
    inner        # prints 42
}

x=1
outer            # prints 42, not 1
```

This is fundamentally different from most languages (which use lexical scoping). It can be useful — passing data through layers without explicit parameters — but it makes refactoring fragile.

### 13.4 Returning Values

POSIX functions return only an exit status (0-255). To return a string, you use a subshell with command substitution or assign to a global variable:

```bash
# Via stdout (preferred)
read_config() {
    cat /etc/myapp/config.txt
}
config=$(read_config)

# Via global
read_config_to_var() {
    config=$(cat /etc/myapp/config.txt)
}
read_config_to_var
echo "$config"

# Via named-ref (bashism — bash 4.3+)
read_config_into() {
    local -n out=$1
    out=$(cat /etc/myapp/config.txt)
}
read_config_into config
echo "$config"
```

---

## 14. Arithmetic — POSIX vs Bash

### 14.1 POSIX Arithmetic Expansion

POSIX provides one form: `$((expr))`.

```bash
x=5
y=10
sum=$((x + y))
product=$((x * y))
remainder=$((x % y))
power=$((2 ** 8))    # 256 — POSIX-2017+
```

Variables inside `$(())` do not need a `$` prefix; the expansion treats them as numeric expressions. Operators include `+ - * / % ** << >> & | ^ ~ ! && || == != < <= > >= ?:`.

### 14.2 Bash's `((expr))`

Bash provides `((expr))` as a *statement* (not an expansion):

```bash
((x++))
((y = x * 2))
if ((x > 5)); then echo big; fi
```

This sets variables and produces an exit status (0 if non-zero result, 1 if zero — *the opposite* of normal command exit codes). Not POSIX.

### 14.3 The Deprecated `$[expr]`

```bash
sum=$[x + y]    # works in bash, deprecated since 1995
```

Don't use. It's documented as obsolete.

### 14.4 Integer-Only — No Floats

POSIX shell arithmetic is integer-only. There is no float, no fixed-point. Division is integer division.

```bash
echo $((10 / 3))      # 3, not 3.33
echo $((10 / 3 * 3))  # 9 — order of operations matters because of integer truncation
echo $((10 * 3 / 3))  # 10 — multiply first
```

For floating-point math, fork an external tool:

```bash
# bc — arbitrary precision
result=$(echo "scale=4; 10 / 3" | bc)    # 3.3333

# awk
result=$(awk 'BEGIN { print 10 / 3 }')   # 3.33333

# python (when available)
result=$(python3 -c 'print(10 / 3)')     # 3.3333333333333335
```

### 14.5 The `expr` Legacy Command

Before `$(())` was widespread, scripts used the external `expr` command:

```bash
sum=$(expr 5 + 10)        # spaces required around operators
sum=$(expr 5 \* 10)        # * must be escaped (it's a glob)
```

`expr` is POSIX but slow (every call is a fork+exec). Modern scripts use `$(())` exclusively.

---

## 15. The Locale + IFS + LC_ALL Trinity

### 15.1 Why Locale Matters

Locale settings affect:

- **Sort order**: `LC_COLLATE` defines string comparison.
- **Character classes**: `LC_CTYPE` defines what `[:upper:]`, `[:digit:]` mean.
- **Date/time format**: `LC_TIME` defines `%c` etc.
- **Currency/number format**: `LC_MONETARY`, `LC_NUMERIC`.
- **Messages**: `LC_MESSAGES` defines language for `gettext`-aware programs.

`LC_ALL` overrides all of them. `LANG` is the fallback for unset `LC_*` variables.

### 15.2 The Sort Surprise

```bash
# In LC_COLLATE=en_US.UTF-8 (most desktop Linux defaults):
printf 'A\nB\na\nb\n' | sort
# Output:
# a
# A
# b
# B
# Case-INSENSITIVE collation; A and a interleaved.

# In LC_ALL=C:
LC_ALL=C sh -c 'printf "A\nB\na\nb\n" | sort'
# Output:
# A
# B
# a
# b
# ASCII byte order; uppercase before lowercase.
```

This frequently breaks shell scripts that use `sort | uniq`, `comm`, or string comparisons. A user reports "my dedup script doesn't work" and the cause is locale.

### 15.3 The Canonical Pattern

For any non-trivial shell script that does sorting, comparisons, or character-class manipulation:

```bash
#!/bin/sh
export LC_ALL=C    # predictable byte-wise behavior

# ... rest of script ...
```

This trades user-friendly localized output for *predictable, reproducible* behavior. For end-user scripts, you may want to leave the locale alone but explicitly invoke `LC_ALL=C cmd` for the parts that need byte-level semantics.

### 15.4 LANG vs LC_ALL

```bash
# Lookup precedence (highest to lowest):
#   LC_ALL  →  LC_<category>  →  LANG  →  default ("C")

LANG=en_US.UTF-8 LC_COLLATE=POSIX sort file
# Sort uses POSIX (byte) ordering; messages would be in en_US.

LC_ALL=C sort file
# Everything is C; LANG is ignored.
```

### 15.5 IFS Interaction

IFS is *also* affected by locale in obscure ways. Some shells normalize IFS=$' \t\n' to whatever the locale considers whitespace. In practice, set `IFS` explicitly when parsing:

```bash
IFS=$'\n'    # newline only
IFS=:        # colon (PATH-style)
IFS=         # nothing — disables splitting entirely
```

---

## 16. Performance and the Fork Cost

### 16.1 The Fork is Not Free

Every external command costs a fork+exec+wait cycle. On modern Linux, this is roughly 1ms per command. On older kernels, slower hardware, or systems with heavy ASLR, it can be 5-10ms.

```bash
# Slow — 1000 forks for 1000 lines
while read line; do
    echo "$line" | grep "pattern" | awk '{print $1}'
done < file.txt

# Fast — 1 fork
awk '/pattern/ { print $1 }' file.txt
```

### 16.2 Builtins vs Externals

Every shell has a list of builtin commands that run inside the shell process — no fork. The big ones:

| Builtin | External equivalent | Fork saved |
|:--------|:-------------------|:-----------|
| `printf` | `/usr/bin/printf` | yes |
| `read` | (no external) | always builtin |
| `[` / `test` | `/usr/bin/[` | yes |
| `[[` | (no external — bashism) | always builtin |
| `:` | `/usr/bin/true` | yes |
| `cd` | (no external — must be builtin) | always |
| `export` | (no external) | always |

Avoid: `cat`, `echo` (in the rare cases where `printf` isn't equivalent), `tr`, `wc`, `expr`, `seq`.

### 16.3 The Useless-Cat Anti-Pattern

```bash
# Wasteful
count=$(cat file.txt | wc -l)

# Equivalent — wc reads the file directly
count=$(wc -l < file.txt)

# Pure shell — counts lines without any external
count=0
while IFS= read -r _; do count=$((count + 1)); done < file.txt
```

### 16.4 Parameter Expansion vs Sed

```bash
# Replace .tar.gz with .tgz — naive
new=$(echo "$file" | sed 's/\.tar\.gz$/.tgz/')

# Pure shell — no fork
new=${file%.tar.gz}.tgz
```

```bash
# Length — naive
length=$(echo -n "$str" | wc -c)

# Pure shell
length=${#str}
```

```bash
# Substring extraction — naive
prefix=$(echo "$str" | cut -c1-10)

# Bash
prefix=${str:0:10}

# POSIX (without bashism)
prefix=$(printf '%.10s' "$str")
```

### 16.5 The case Statement vs grep

```bash
# Slow — fork for every iteration
for f in *.txt; do
    if echo "$f" | grep -q '^backup'; then
        echo "skipping $f"
    fi
done

# Fast — pure shell
for f in *.txt; do
    case "$f" in
        backup*) echo "skipping $f" ;;
    esac
done
```

`case` uses glob-pattern matching in-shell. No fork.

### 16.6 The Pipeline Cost

Each `|` is another fork. A pipeline of 5 commands is 5 forks. For a one-shot script this is fine; for a hot loop it's ruinous.

```bash
# 5 forks per line
ls -la | grep -v '^d' | awk '{print $9}' | sort | uniq

# Often replaceable with awk alone (1 fork total)
ls -la | awk '/^[^d]/ { count[$9]++ } END { for (f in count) print f }'
```

---

## 17. Portability Pitfalls

### 17.1 The Bashism Catalog

| Feature | POSIX? | Notes |
|:--------|:------:|:------|
| `[[ ]]` | no | use `[ ]` |
| `(( ))` | no | use `[ ]` with `-eq`, `-lt` etc. |
| Arrays `arr=(...)` | no | use whitespace-separated string + `set --` |
| `${arr[@]}` | no | no portable equivalent |
| `${var//pat/rep}` | no | use `sed` or `tr` |
| `${var^^}`, `${var,,}` | no | use `tr` |
| `${!var}` indirection | no | use `eval` (carefully) |
| `<<<` here-string | no | use heredoc |
| `<(cmd)` process sub | no | use temp file |
| `local` | no | use subshell function |
| `function` keyword | no | use `name() { ... }` |
| `&>` redirect both | no | use `> file 2>&1` |
| `$'...'` C-escape | no | use `printf` |
| `mapfile`/`readarray` | no | use `while read` loop |
| `coproc` | no | use named pipes |
| `read -a` | no | manual splitting |
| `read -N`, `-d` | no | not portable |
| `select` | yes (POSIX-2024) | recently standardized |
| `pipefail` | yes (POSIX-2024) | recently standardized |
| Brace expansion `{a,b,c}` | no | enumerate explicitly |

### 17.2 The checkbashisms Tool

Debian provides `checkbashisms` (in the `devscripts` package). It analyzes a script and reports bash-specific constructs:

```bash
checkbashisms ./my-script.sh
# possible bashism in ./my-script.sh line 5 (echo -e):
#     echo -e "...";
# possible bashism in ./my-script.sh line 12 (let ...):
#     let x=$x+1
```

It is heuristic and produces some false positives, but it is the standard "is this script POSIX-clean" tool.

### 17.3 The macOS /bin/sh Reality

macOS ships `/bin/sh` as bash invoked with `--posix` (or, on more recent versions, dash). The exact identity has changed:

| macOS version | /bin/sh | Notes |
|:--------------|:--------|:------|
| 10.14 and earlier | bash 3.2 in POSIX mode | "good enough" POSIX |
| 10.15-13 | dash | strict POSIX |
| 14+ | dash | strict POSIX |

For macOS, `bash` itself remains 3.2 (the last GPL-2 version) at `/bin/bash`. To get a modern bash, use Homebrew's `/opt/homebrew/bin/bash`. The shebang `#!/usr/bin/env bash` finds whichever is first in `PATH`.

### 17.4 The echo Disaster

`echo` is the most non-portable command in the shell. Behavior varies wildly:

```bash
echo -e '\n'    # bash: prints two newlines; dash: prints "-e\n"
echo -n 'foo'   # most shells: "foo"; some old systems: "-n foo"
echo '\n'       # bash with xpg_echo: newline; bash default: "\n"
```

The POSIX-recommended replacement is `printf`:

```bash
printf '%s\n' "first line"
printf '%s\n' "$line"          # safe even if $line starts with -
printf '%b\n' '\033[1mbold'    # interpret escapes
```

`printf` is in POSIX, behaves identically across shells, and handles `--` and leading-hyphen arguments cleanly.

### 17.5 Running with dash to Test

```bash
# Make a script POSIX-clean by running it through dash
dash -n ./script.sh    # parse-only; reports syntax errors
dash ./script.sh       # actual run

# Some scripts only fail at runtime in dash because of bashisms
# in code paths that didn't run during parsing
```

---

## 18. shellcheck — Static Analysis

### 18.1 The Tool

`shellcheck` is a lint for shell scripts. It catches the most common bugs (unquoted variables, useless cat, broken globbing) and many subtle ones (wrong quoting in conditionals, here-doc indentation issues).

```bash
shellcheck ./script.sh
# In ./script.sh line 5:
# rm $files
#    ^---^ SC2086: Double quote to prevent globbing and word splitting.
```

### 18.2 The Rule Catalogue

| Code | Rule |
|:-----|:-----|
| SC2086 | Double quote to prevent globbing and word splitting |
| SC2155 | Declare and assign separately to avoid masking return values |
| SC2046 | Quote `$(...)` to prevent word splitting |
| SC2002 | Useless `cat` — pass file directly to next command |
| SC2207 | Prefer `mapfile` over `arr=( $(cmd) )` for splitting |
| SC2034 | Variable appears unused — possible typo |
| SC2148 | Tips: missing shebang |
| SC1091 | Could not find sourced file — install or skip |
| SC2154 | Variable referenced but not assigned |
| SC2236 | Use `-n` instead of `! -z` |
| SC2181 | Check exit code directly with `if cmd` rather than `$?` |
| SC2230 | Avoid `which` — use `command -v` |

### 18.3 Inline Disable

Sometimes a warning is a false positive or the construct is intentional:

```bash
# shellcheck disable=SC2086
files=$(printf '%s\n' "$files")    # word-splitting is what we want here
```

### 18.4 .shellcheckrc

Repository-level configuration:

```text
# .shellcheckrc
shell=bash
disable=SC2034,SC2086    # accept these for the whole repo
external-sources=true     # follow `source` and `.` directives
```

### 18.5 Editor Integration

Almost every editor has a shellcheck plugin: vim (`ALE`, `vim-shellcheck`), Emacs (`flycheck`), VSCode (`bash-language-server`), Sublime, Atom, JetBrains. Run on save; fix on save. Treat shellcheck warnings as errors.

---

## 19. The Modern "Strict Mode"

### 19.1 The Pattern

Aaron Maxwell's "Bash Strict Mode" (2014) became the canonical opinionated preamble:

```bash
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'
```

Each piece:

| Setting | Effect |
|:--------|:-------|
| `set -e` | Exit on uncaught command failure |
| `set -u` | Exit if an unset variable is referenced |
| `set -o pipefail` | Pipeline exit = first non-zero stage |
| `IFS=$'\n\t'` | Word-split only on newline and tab; spaces in vars no longer split |

### 19.2 The Caveats

- `set -e` is not bulletproof (see Section 7).
- `set -u` makes `${var:-default}` mandatory for any variable that might be unset.
- `set -o pipefail` is bash-specific (and POSIX-2024).
- `IFS=$'\n\t'` uses a bashism (`$'...'`). For POSIX, set IFS via printf:

```bash
IFS="$(printf '\n\t')"
IFS="${IFS%?}"    # strip the trailing newline that command-sub doesn't strip on tab-only
```

### 19.3 The BASH_ENV Gotcha

When bash runs a non-interactive script, it sources `$BASH_ENV` first. If a malicious or buggy `BASH_ENV` is set in the environment, it can subvert the script before `set -e` takes effect.

```bash
# Defense: clear BASH_ENV at script start
unset BASH_ENV ENV
```

This is a low-priority hardening for trusted environments but worth knowing.

### 19.4 The Inevitable Workarounds

`set -u` is particularly aggressive. Common patterns to handle it:

```bash
# Default value
echo "${MY_VAR:-default}"

# Test for set
if [ -n "${MY_VAR:-}" ]; then
    echo "set: $MY_VAR"
fi

# Iterate possibly-empty positional parameters
set -u
for arg in "${@:-}"; do
    echo "$arg"
done
# Without ":-" this would fail with "$@" unset when zero args are passed.
```

### 19.5 Strict Mode Is Not Strict Enough

Real strict-mode scripts often add:

```bash
set -euo pipefail
shopt -s inherit_errexit    # bash 4.4+: inherit -e into subshells
shopt -s nullglob            # unmatched globs become empty (often safer)
shopt -s lastpipe            # last pipeline stage runs in current shell
trap 'echo "ERROR: line $LINENO" >&2' ERR
```

The result is bash-specific (none of `inherit_errexit`, `nullglob`, `lastpipe` are POSIX) but considerably more reliable.

---

## 20. The Bash-vs-POSIX Decision Tree

### 20.1 When to Write Portable POSIX

- The script will run on minimal environments: Alpine containers, BSD jails, BusyBox embedded Linux.
- The script is part of a system bootstrap (initramfs, container init, install scripts).
- The script must run on *any* Unix the user might have (distribution package install scripts).
- The script will be embedded as `/bin/sh` shebang in a Dockerfile based on Alpine.

### 20.2 When Bash-Only Is Fine

- The script is for a specific developer environment with bash installed.
- The script is part of a CI pipeline running on Ubuntu/Fedora/macOS-with-modern-bash.
- The complexity of POSIX-fying it would obscure the logic.
- You explicitly use `#!/usr/bin/env bash` and document the dependency.

### 20.3 When to Stop Using Shell Entirely

If your script:
- Has more than ~200 lines of business logic
- Manipulates JSON (more than `jq | head`)
- Needs structured concurrency
- Needs sophisticated error handling
- Will be modified by people who don't read shell

…then rewrite it in Python, Go, or Rust. Shell is for orchestration; not data processing. The point at which shell's quirkiness becomes a maintenance hazard varies by team, but ~200 lines is a reasonable rule of thumb.

### 20.4 Bash 4 Features and macOS

macOS ships bash 3.2 at `/bin/bash`. Linux ships bash 5.x. If you use bash 4+ features (`mapfile`, `${var,,}`, associative arrays), your script breaks on default macOS.

```bash
# Detect bash version
if [ "${BASH_VERSINFO[0]:-0}" -lt 4 ]; then
    echo "bash 4+ required. Install via Homebrew: brew install bash" >&2
    exit 1
fi
```

The shebang `#!/usr/bin/env bash` finds `/opt/homebrew/bin/bash` if Homebrew's bin is first in PATH. Document this requirement clearly.

---

## 21. Idioms at the Internals Depth

### 21.1 The Canonical Preamble

```bash
#!/usr/bin/env bash
#
# my-script.sh — short description
#
# Usage:
#   my-script.sh [OPTIONS] <args>
#
# Options:
#   -v, --verbose    enable verbose output
#   -h, --help       show this message

set -euo pipefail
IFS=$'\n\t'

# Resolve the script directory (handles symlinks)
SCRIPT_DIR=$(cd "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
SCRIPT_NAME=$(basename -- "${BASH_SOURCE[0]}")
```

`BASH_SOURCE[0]` is the path of the current script even when sourced. `dirname` and `basename` are POSIX. The `-- ` is to prevent option-parsing if the path starts with a dash (rare but possible).

### 21.2 The Trap-EXIT Cleanup

```bash
TMPDIR=$(mktemp -d)
trap 'rm -rf -- "$TMPDIR"' EXIT INT TERM

# ... use $TMPDIR ...

# Cleanup happens automatically on:
#   - normal exit (EXIT)
#   - Ctrl-C (INT, then EXIT)
#   - kill (TERM, then EXIT)
```

Multiple traps stack — register them in setup order:

```bash
trap 'echo "exiting"' EXIT
TMPDIR=$(mktemp -d)
trap 'rm -rf -- "$TMPDIR"; echo "exiting"' EXIT
```

The `--` after `rm -rf` is critical: if `$TMPDIR` is somehow empty, `rm -rf` would attempt to operate on the current directory.

### 21.3 Log Functions

The convention is logs to stderr, data to stdout. This way `script | jq` can pipe data without log lines polluting it.

```bash
log()  { printf '[%s] %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
warn() { printf '[%s] WARN: %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
err()  { printf '[%s] ERROR: %s\n' "$(date -u +%H:%M:%S)" "$*" >&2; }
die()  { err "$@"; exit 1; }

log "starting"
data=$(produce_data) || die "produce_data failed"
echo "$data"        # actual output to stdout
log "done"
```

### 21.4 Safe Argument Parsing

Pure POSIX argument parsing is a finite-state machine. The pattern:

```bash
verbose=0
output=""

while [ $# -gt 0 ]; do
    case "$1" in
        -v|--verbose)
            verbose=1
            shift
            ;;
        -o|--output)
            [ -n "${2:-}" ] || die "--output requires an argument"
            output=$2
            shift 2
            ;;
        --output=*)
            output=${1#*=}
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        -*)
            die "unknown option: $1"
            ;;
        *)
            break
            ;;
    esac
done
# Remaining positional args in "$@"
```

The `--` handling matches POSIX `getopts` and standard tools: anything after `--` is a positional argument even if it starts with `-`.

### 21.5 Atomic Write via Mv

```bash
write_atomic() {
    local target=$1
    local data=$2
    local tmp
    tmp=$(mktemp -t "$(basename -- "$target").XXXXXX")
    printf '%s\n' "$data" > "$tmp"
    chmod 644 "$tmp"
    mv -f -- "$tmp" "$target"
}
```

`mv` on the same filesystem is atomic — readers either see the old content or the new, never partial. Different filesystems require copy+rename, which is *not* atomic. A robust version checks `df` or uses `mktemp -d` next to the target.

### 21.6 The Lockfile Pattern

```bash
# Approach 1: flock (Linux util-linux)
LOCKFILE=/var/lock/my-script.lock
exec 9>"$LOCKFILE"
flock -n 9 || die "another instance is running"

# ... critical section ...

# Approach 2: mkdir (atomic everywhere)
LOCKDIR=/tmp/my-script.lock
if ! mkdir "$LOCKDIR" 2>/dev/null; then
    die "another instance is running"
fi
trap 'rmdir "$LOCKDIR"' EXIT

# ... critical section ...
```

`mkdir` is atomic (it either creates the directory or fails — no race window). `flock` is more robust but Linux-only.

### 21.7 Defensive PATH

A script invoked from a malicious environment may have a hostile PATH. Reset it explicitly for security-sensitive scripts:

```bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
export PATH
```

Or, even more paranoid — use full paths for every command:

```bash
/usr/bin/awk '...'
/bin/cp src dst
```

This is overkill for most scripts but appropriate for setuid wrappers and security-critical contexts.

---

## 22. Prerequisites

- Familiarity with the unix process model: fork, exec, wait, signals, file descriptors
- Comfort reading manpages and the POSIX.1 standard
- Knowledge of basic regular expressions (for grep/sed/awk)
- Understanding of file permissions, symlinks, and the filesystem hierarchy
- Some exposure to at least one programming language with explicit scope (helps contrast with shell's dynamic scoping)
- Ability to install and run multiple shells (dash, bash, zsh) for testing

---

## 23. Complexity

- **Tokenization**: O(n) in script length per command; one pass through input.
- **Word splitting**: O(m) where m is the length of the expanded value; one IFS scan.
- **Pathname expansion**: O(k * f) where k is glob-pattern complexity and f is filesystem entries. Naïve `*` is O(directory entries).
- **Parameter expansion**: pure-shell operations (`${var#prefix}` etc.) are O(n) string scans, no fork.
- **Command substitution**: O(fork+exec+wait+pipe-read), roughly 1ms baseline + child process time.
- **Pipeline**: O(n) processes, each fork+exec+wait, plus pipe IPC overhead.
- **Subshell**: O(1) fork, then O(child execution).
- **Builtins**: O(operation), no fork.
- **Script with N external command invocations**: O(N) forks, dominated by external execution. Optimization typically means reducing N (fewer pipes, fewer subshells, more in-shell parameter expansion).
- **Script with M lines of pure-shell logic**: O(M) interpreted, roughly microseconds per command.

For scripts with hot loops, the dominant cost is fork count, not algorithmic complexity. A script that is "O(n^2)" but does O(1) forks per iteration outperforms an "O(n log n)" script that forks every iteration once n exceeds ~1000.

---

## 24. See Also

- shell-scripting (sheet)
- bash
- zsh
- fish
- nushell
- polyglot

---

## 25. References

- [POSIX.1-2017 Shell Command Language](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html) — the authoritative specification of POSIX shell semantics
- [Wooledge BashGuide](https://mywiki.wooledge.org/BashGuide) — the canonical pedagogical reference for shell scripting practices
- [Wooledge BashFAQ](https://mywiki.wooledge.org/BashFAQ) — answers to the gnarliest shell scripting questions, including FAQ #105 on `set -e`
- [Bash Reference Manual (GNU)](https://www.gnu.org/software/bash/manual/bash.html) — the bash-specific extensions and behaviors
- [Bash Pitfalls (Wooledge)](https://mywiki.wooledge.org/BashPitfalls) — list of the most common shell scripting mistakes
- [shellcheck](https://www.shellcheck.net/) — online and offline static analysis for shell scripts
- [Aaron Maxwell, "Use the Unofficial Bash Strict Mode"](http://redsymbol.net/articles/unofficial-bash-strict-mode/) — the canonical "strict mode" article
- "The Linux Command Line" by William Shotts — accessible introduction to bash and unix utilities
- "Classic Shell Scripting" by Arnold Robbins and Nelson Beebe — deep dive on portable shell scripting and standard tools
- "Unix Power Tools" by Jerry Peek, Shelley Powers, Tim O'Reilly, and Mike Loukides — encyclopedia of unix idioms
- [dash(1) man page](http://manpages.debian.org/dash) — Debian's POSIX shell
- [busybox sh source](https://git.busybox.net/busybox/tree/shell) — minimal POSIX shell implementation
