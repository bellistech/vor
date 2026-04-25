# Bash (Bourne Again Shell)

GNU's Unix shell and command language — the de-facto interactive shell on Linux and the lingua franca of production scripts. POSIX-compatible mode plus a generous pile of extensions: arrays, associative arrays, `[[ ]]` tests, process substitution, `=~` regex, parameter expansion gymnastics, `mapfile`, `coproc`, `/dev/tcp`, and more.

## Setup

### Shebang and bash version

```bash
#!/usr/bin/env bash
# /usr/bin/env finds bash on $PATH — portable across distros and macOS Homebrew.
# Avoid #!/bin/bash on macOS: /bin/bash is bash 3.2 (frozen at GPLv2). Brew installs bash 5+ at /opt/homebrew/bin/bash.

# Print effective version:
echo "$BASH_VERSION"           # e.g. 5.2.21(1)-release
echo "${BASH_VERSINFO[0]}"     # major version only — 5

# Hard-fail on bash < 4 (no associative arrays, no ${var^^}, no mapfile, no globstar):
if (( BASH_VERSINFO[0] < 4 )); then
    echo "bash >= 4 required (got $BASH_VERSION)" >&2
    exit 1
fi
```

### Why bash 4 is the floor

```bash
# bash 3.2 (macOS /bin/bash) is missing:
#   - associative arrays         declare -A
#   - ${var^^} / ${var,,}        case conversion
#   - mapfile / readarray        slurp file into array
#   - globstar **                recursive glob
#   - coproc                     coprocesses
#   - <<<"$var" works, but here-string newline is a 5.x change
# If a script is meant to run on a fresh macOS, either: (a) require Homebrew bash, (b) use #!/usr/bin/env bash + version-check + abort, or (c) write POSIX sh.
```

### Find what bash is running

```bash
type bash                       # which bash binary the shell would exec
command -v bash                 # POSIX-clean version
which -a bash                   # every bash on $PATH (some systems have two)
ls -l /bin/bash /usr/local/bin/bash /opt/homebrew/bin/bash 2>/dev/null
```

### Interactive vs login vs script

```bash
# Login shell:        ssh login, console login — sources /etc/profile then ~/.bash_profile (or ~/.bash_login or ~/.profile)
# Interactive non-login (terminal tab): sources /etc/bash.bashrc then ~/.bashrc
# Non-interactive (script):             sources nothing — relies on shebang and explicit env

# Detect inside a script:
if [[ $- == *i* ]]; then echo "interactive"; fi
shopt -q login_shell && echo "login shell"
```

### Strict-mode preamble (every new script)

```bash
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Optional but recommended for scripts you debug:
# trap 'echo "ERR at line $LINENO: $BASH_COMMAND" >&2' ERR
```

## Strict Mode

### The four-line strict preamble

```bash
set -e               # exit immediately if any command exits non-zero (with caveats)
set -u               # treat unset variables as an error and exit
set -o pipefail      # a pipeline's exit status is the last non-zero command's, not the last command's
IFS=$'\n\t'          # field separator: newline + tab only, never space — kills word-splitting on spaces

# Equivalent one-liner:
set -euo pipefail
```

### Why each flag matters

```bash
# -e (errexit) — without it, scripts plough on after failures:
cp /etc/passwd /tmp/missing/dir/passwd   # fails silently without -e
echo "carrying on..."                    # would still print

# -u (nounset) — catches typos at runtime:
nmae="alice"        # typo
echo "$name"        # without -u: prints empty. with -u: error and exit.

# -o pipefail — without it, pipelines hide upstream failures:
false | true        # exit code without pipefail: 0 (only last command counts)
                    # with pipefail: 1
# Critical when piping into tee, jq, awk — upstream errors otherwise vanish.

# IFS=$'\n\t' — without it, "for x in $files" word-splits on space:
files="my file.txt other.txt"
for f in $files; do echo "$f"; done      # 3 iterations, not 2 — splits on space
```

### Where set -e quietly does NOT exit

```bash
# 1. Functions called in conditional context are not subject to errexit:
foo() { false; echo "still here"; }
foo || true                              # both lines run; exit not triggered

# 2. Commands in if/while/until conditions:
if some_command_that_fails; then ... fi  # failure is the condition, not an error

# 3. Left side of && or ||:
false && echo "nope"                     # OK — false is being tested

# 4. Commands whose return value is being inverted with !:
! false                                  # OK
```

### Disabling strict mode locally

```bash
# Temporarily allow a failing command:
set +e
risky_command
rc=$?
set -e
echo "risky returned $rc"

# Or use || true on the offender:
risky_command || true

# Or capture the exit code:
risky_command && rc=0 || rc=$?
```

### ERR trap for diagnostics

```bash
trap 'echo "FAIL line $LINENO: $BASH_COMMAND (exit $?)" >&2' ERR
# With set -e + ERR trap you get a stack trace one-liner on every fault.
```

## Variables

### Assignment (no spaces!)

```bash
name="world"           # CORRECT
name = "world"         # WRONG — bash reads "name" as a command, "=" and "world" as args
name= "world"          # WRONG — sets name to empty for one command, then runs "world"
name ="world"          # WRONG — same deal

# Multiple on one line:
foo=1 bar=2 baz=3

# Per-command env (only in scope for that one command):
LANG=C sort file.txt           # forces C locale for sort, doesn't change shell's LANG
```

### export — make it visible to children

```bash
PATH="$HOME/bin:$PATH"          # local to this shell only
export PATH                     # children (subprocesses) inherit it
export PATH="$HOME/bin:$PATH"   # combine — set + export in one line
declare -x VAR=value            # equivalent to export VAR=value

# Without export, child processes see the OLD value:
greeting="hi"
bash -c 'echo $greeting'        # prints empty — child can't see it
export greeting
bash -c 'echo $greeting'        # prints "hi"
```

### local — function scope only

```bash
greet() {
    local who="$1"              # not visible outside greet
    local greeting="hello"
    echo "$greeting, $who"
}
# After greet returns, $who and $greeting are unset.
# Without `local`, you'd pollute the global namespace.
```

### readonly — immutable

```bash
readonly DB_HOST="localhost"
DB_HOST="elsewhere"             # bash: DB_HOST: readonly variable
unset DB_HOST                   # bash: unset: DB_HOST: cannot unset: readonly variable
declare -r LIMIT=100            # equivalent

# readonly is process-local — exec'd children get the value but not the readonly bit.
```

### declare / typeset (synonyms)

```bash
declare -i count=0              # integer — assignments are evaluated as arithmetic
count="3 + 4"; echo "$count"    # prints 7

declare -a fruits               # explicit indexed array
declare -A config               # explicit associative array (bash 4+)
declare -l name="ALICE"         # always-lowercase
declare -u shout="alice"        # always-uppercase
declare -r FROZEN=1             # readonly
declare -x EXPORTED=1           # exported
declare -n ref=target           # nameref — pointer to another variable (bash 4.3+)

# Print every variable + its attributes:
declare -p

# Print just one:
declare -p PATH
```

### Special variables

```bash
$0          # script name (or shell name when interactive)
$1 $2       # positional args 1 and 2
${10}       # 10+ requires braces — $10 means $1 followed by literal 0
$#          # number of positional args
$@          # all args, "$@" preserves quoting (each arg stays a separate word)
$*          # all args, "$*" joins with first char of IFS
$?          # exit status of last foreground command
$$          # PID of current shell
$!          # PID of last backgrounded command
$_          # last argument of previous command
$-          # current option flags (eg. himBHs)
$LINENO     # current line in script
$BASH_SOURCE # path to current script (use ${BASH_SOURCE[0]} in functions)
$FUNCNAME   # current function name (array — caller chain)
$RANDOM     # 0..32767 random integer per read
$SECONDS    # seconds since shell start (or last assignment)
$PIPESTATUS # array of exit codes of last pipeline
$BASH_VERSION
$BASH_VERSINFO  # array: major, minor, patch, build, release, machine
$EUID $UID  # effective and real user id
$HOSTNAME $OSTYPE $MACHTYPE
```

### Unsetting variables

```bash
name="alice"
unset name             # variable is gone
unset -v name          # explicit: variable, not function
unset -f greet         # unset function

# Under set -u, referencing an unset variable errors. Two safe alternatives:
echo "${name:-}"       # empty if unset
echo "${name-}"        # empty if unset (but uses var even if explicitly empty — see :-

# Test for set vs unset:
[[ -v name ]] && echo "set"
[[ -z "${name:-}" ]] && echo "unset or empty"
```

## Quoting

### Single, double, ANSI-C, dollar-locale

```bash
echo 'hello $USER'              # single — literal, no expansion
echo "hello $USER"              # double — variable + command sub + arithmetic expand; backslash escapes some
echo $'line1\nline2'            # ANSI-C — \n \t \\ \xHH \uHHHH literal escapes
echo $"hello"                   # locale-translated string (i18n) — rarely used in practice
```

### When to single-quote

```bash
# Literal regex, awk programs, sed expressions, anything with $:
grep '^[A-Z][[:alnum:]]*$' file.txt
awk '{print $1}' file.txt
sed -E 's/foo\.bar/baz/g' file.txt

# Embedding literal $ to be expanded by another tool:
ssh host 'echo $HOME'           # $HOME expands on remote
ssh host "echo $HOME"           # $HOME expands locally — almost certainly wrong
```

### When to double-quote

```bash
# Anywhere a variable could contain spaces, glob chars, or empty:
cp "$src" "$dst"                # ALWAYS quote variables in paths
echo "user is: $user"           # safe interpolation
"${array[@]}"                   # safe array expansion
"$(command)"                    # safe command substitution
```

### The unquoted-variable disaster

```bash
file="my report.pdf"
rm $file                        # actually runs: rm my report.pdf — TWO files deleted
rm "$file"                      # CORRECT — rm "my report.pdf"

# Globbing:
arg='*'
echo $arg                       # expands to every file in cwd
echo "$arg"                     # literal *

# Empty variable as positional:
empty=""
test $empty = "x"               # bash: test: =: unary operator expected (expanded to: test = x)
test "$empty" = "x"             # CORRECT — test "" = "x"

# Word splitting in for-loops:
files="a.txt b.txt c.txt"
for f in $files; do ...         # 3 iterations
files="a file.txt b.txt"
for f in $files; do ...         # 3 iterations again — wrong! splits on space
mapfile -t arr <<<"$files"      # FIX — read into array
```

### Mixing quotes for a literal '

```bash
echo 'it'\''s'                  # 'it' + escaped ' + 's'  → it's
echo "it's"                     # double quotes accept '
echo $'it\'s'                   # ANSI-C
```

## Parameter Expansion — Defaults and Errors

### Defaults: substitute when missing

```bash
echo "${name:-default}"        # use "default" if name is unset OR empty
echo "${name-default}"         # use "default" if name is unset (but NOT if empty)

# Difference matters:
foo=""
echo "${foo:-fallback}"        # → fallback (empty triggers it)
echo "${foo-fallback}"         # → empty   (set, even though empty, so no fallback)

unset bar
echo "${bar:-fallback}"        # → fallback
echo "${bar-fallback}"         # → fallback (also)
```

### Assign-default

```bash
echo "${name:=default}"        # if unset/empty: assign "default" to name AND substitute
echo "${name=default}"         # if unset only

# Useful at top of scripts:
: "${TIMEOUT:=30}"             # set TIMEOUT to 30 if not provided in env
: "${LOG_LEVEL:=INFO}"
# The leading : is the no-op command — we want the side effect, not output.
```

### Error-if-missing

```bash
echo "${name:?required: name}"  # if unset/empty: print error to stderr and exit script
echo "${name?required}"         # if unset only

# Top-of-script required-arg pattern:
: "${ENV_NAME:?must export ENV_NAME before running}"
# If ENV_NAME is unset: prints "ENV_NAME: must export ENV_NAME before running" and exits non-zero.
```

### Alt-value-if-set

```bash
echo "${name:+yes}"            # "yes" if name is set AND non-empty, else empty
echo "${name+yes}"             # "yes" if set (even if empty)

# Common: build an option string only when a variable is set:
extra=""
[[ -n "${VERBOSE:-}" ]] && extra="--verbose"
# OR the more compact:
extra="${VERBOSE:+--verbose}"
```

### The :- vs - mnemonic

```bash
# WITH colon:    treats unset and empty the same
# WITHOUT colon: treats them differently (only unset triggers the operation)
#
# 99% of the time you want the colon variant — empty is almost always "missing".
```

## Parameter Expansion — Substring and Length

### String length

```bash
s="hello, world"
echo "${#s}"                   # 12 — character count

# Array length is different:
arr=(a b c)
echo "${#arr[@]}"              # 3 — number of elements
echo "${#arr[0]}"              # 1 — length of element 0
```

### Substring extraction

```bash
s="hello, world"
echo "${s:7}"                  # "world"          — from offset 7 to end
echo "${s:7:3}"                # "wor"            — from offset 7, length 3
echo "${s:0:5}"                # "hello"          — first 5 chars
echo "${s: -5}"                # "world"          — last 5 (note SPACE before -5 to disambiguate from :-)
echo "${s:7:-1}"               # "worl"           — offset 7, stop 1 from the end (bash 4.2+)
echo "${s: -5:3}"              # "wor"            — start at -5, take 3
```

### Pitfall: negative offset needs space or parens

```bash
echo "${s:-5}"                 # WRONG — this is the default operator, prints "hello, world" if s is set
echo "${s: -5}"                # CORRECT — leading space makes -5 a number
echo "${s:(-5)}"               # also OK — parentheses disambiguate
```

### Array slicing

```bash
arr=(a b c d e f)
echo "${arr[@]:2}"             # c d e f       — from index 2
echo "${arr[@]:2:3}"           # c d e         — from index 2, 3 elements
echo "${arr[@]: -2}"           # e f           — last 2 (space before -)
echo "${@:2}"                  # all positional args from $2 onward
echo "${@:2:3}"                # 3 args starting at $2
```

## Parameter Expansion — Pattern

### Prefix/suffix removal

```bash
path="/home/user/docs/file.tar.gz"

echo "${path#*/}"              # home/user/docs/file.tar.gz   — shortest prefix match of */
echo "${path##*/}"             # file.tar.gz                  — longest prefix match → basename
echo "${path%.*}"              # /home/user/docs/file.tar      — shortest suffix match of .*
echo "${path%%.*}"             # /home/user/docs/file          — longest suffix match → drop all extensions
echo "${path%/*}"              # /home/user/docs               — strip last component → dirname

# Mnemonic: # is to the left of % on a US keyboard, mirroring left-strip vs right-strip.
# Single = shortest, double = longest.
```

### Replace pattern

```bash
s="banana"
echo "${s/a/X}"                # bXnana    — replace FIRST a
echo "${s//a/X}"               # bXnXnX    — replace ALL a
echo "${s/#b/X}"               # Xanana    — replace at START only
echo "${s/%a/X}"               # bananX    — replace at END only
echo "${s/a/}"                 # bnana     — delete first a (empty replacement)
echo "${s//[aeiou]/_}"         # b_n_n_    — pattern uses globs, not regex
```

### Pattern uses GLOB syntax, not regex

```bash
s="foo123bar"
echo "${s//[0-9]/}"            # foobar             — strip digits (glob char class)
echo "${s/foo*bar/X}"          # X                  — glob *
# For real regex, drop down to sed/awk or use [[ =~ ]] with capture groups:
[[ "$s" =~ ([a-z]+)([0-9]+)([a-z]+) ]] && echo "${BASH_REMATCH[2]}"  # 123
```

### Strip multiple extensions

```bash
f="archive.tar.gz"
echo "${f%.gz}"                # archive.tar
echo "${f%.tar.gz}"            # archive
echo "${f%%.*}"                # archive            — drop EVERYTHING after first dot
```

## Parameter Expansion — Case

### Upper, lower, toggle (bash 4+)

```bash
s="hello, World"
echo "${s^}"                   # Hello, World      — uppercase first char
echo "${s^^}"                  # HELLO, WORLD      — uppercase all
echo "${s,}"                   # hello, World      — lowercase first
echo "${s,,}"                  # hello, world      — lowercase all
echo "${s~}"                   # Hello, world      — toggle first char (bash 4+, deprecated)
echo "${s~~}"                  # HELLO, wORLD      — toggle all chars

# Pattern-restricted case change — change only matching characters:
echo "${s^^[aeiou]}"           # hEllO, WOrld     — uppercase vowels only
echo "${s,,[A-Z]}"             # hello, world      — lowercase upper-case letters
```

### Pre-bash-4 fallback

```bash
# bash 3.2 has no ${var^^}. Fall back to tr:
upper=$(echo "$s" | tr '[:lower:]' '[:upper:]')
lower=$(echo "$s" | tr '[:upper:]' '[:lower:]')

# Or use awk:
upper=$(awk -v s="$s" 'BEGIN{print toupper(s)}')
```

## Parameter Expansion — Indirect and Names

### Indirect expansion

```bash
# Get the value of a variable whose NAME is in another variable:
name="user"
user="alice"
echo "${!name}"                # alice — value of $user

# More common modern equivalent: nameref (bash 4.3+)
declare -n ref=user
echo "$ref"                    # alice — ref is a pointer
ref="bob"
echo "$user"                   # bob   — assignments through nameref affect target
```

### List variables/array keys by prefix

```bash
USER_NAME="alice"
USER_HOME="/home/alice"
USER_SHELL="/bin/zsh"

echo "${!USER_*}"              # USER_HOME USER_NAME USER_SHELL — names matching prefix
echo "${!USER_@}"              # same, but each name as separate quoted word with "${!USER_@}"

for v in "${!USER_@}"; do
    echo "$v = ${!v}"
done
```

### Array indices/keys

```bash
arr=(a b c)
echo "${!arr[@]}"              # 0 1 2

declare -A m=([host]=x [port]=8080)
echo "${!m[@]}"                # host port (order undefined)
```

## Numbers and Arithmetic

### `$(( ... ))` — arithmetic expansion (gives you the value)

```bash
echo $((3 + 5))                # 8
x=$((10 * 2))                  # x=20
echo $((2**10))                # 1024
echo $((10 / 3))               # 3 — integer division only
echo $((10 % 3))               # 1
echo $((0xff))                 # 255
echo $((0b1010))               # 10 (bash 4+)
echo $((010))                  # 8 — leading 0 is octal!  CAREFUL.
echo $((10#08))                # 8 — explicit base 10
echo $((16#ff))                # 255 — base 16

# Bitwise: & | ^ ~ << >>
echo $((0xff & 0x0f))          # 15
echo $((1 << 8))               # 256
```

### `(( ... ))` — arithmetic command (gives you a status)

```bash
((x++))                        # increment
((y = x * 2))                  # assignment
((x > 5)) && echo "big"        # use as condition

# Inside (( )) you do NOT need $ to deref:
i=0
while ((i < 10)); do
    ((i++))
done
echo $i                        # 10
```

### `let` — older synonym for (( ))

```bash
let x=5
let y=x*2
let z++
# Pretty much always prefer (( )) — same semantics, less quoting drama.
```

### `expr` — external POSIX tool, ancient

```bash
expr 3 + 5                     # 8 — but: requires SPACES around operators, * is glob, etc.
# Modern bash: never use expr. Use $(( )).
```

### Floating-point — bash has none

```bash
echo $((10 / 3))               # 3 — integer
# For real floats, shell out to bc, awk, or python:
echo "scale=4; 10/3" | bc       # 3.3333
awk 'BEGIN { printf "%.4f\n", 10/3 }'   # 3.3333
python3 -c 'print(10/3)'        # 3.3333333333333335
```

## Indexed Arrays

### Declare and assign

```bash
fruits=(apple banana cherry)            # literal
declare -a fruits                        # explicit declaration
fruits[0]="apple"; fruits[1]="banana"    # by index

# Sparse assignment is legal:
arr[0]="a"; arr[5]="f"
echo "${arr[@]}"                          # a f
echo "${!arr[@]}"                         # 0 5 — only the populated indices
echo "${#arr[@]}"                         # 2 — count of populated, NOT max index + 1

# From a command:
mapfile -t lines < file.txt              # one line per element, no trailing newline
readarray -t lines < file.txt            # synonym
files=( *.txt )                          # globbing
words=( $(cat list.txt) )                # word-split (only if you actually want it)
```

### Access

```bash
arr=(a b c d e)
echo "${arr[0]}"               # a — single element
echo "${arr[@]}"               # a b c d e — all elements as separate words
echo "${arr[*]}"               # a b c d e — all elements joined by IFS first char
echo "${#arr[@]}"              # 5 — count
echo "${!arr[@]}"              # 0 1 2 3 4 — indices

# Slicing:
echo "${arr[@]:1:3}"           # b c d — from index 1, 3 elements
echo "${arr[@]:2}"             # c d e — from index 2 to end
echo "${arr[@]: -2}"           # d e   — last 2 (space before -)
```

### "${arr[@]}" vs "${arr[*]}"

```bash
arr=("a b" "c d")

for x in "${arr[@]}"; do
    echo "<$x>"                # <a b>\n<c d>     — proper preservation
done

for x in "${arr[*]}"; do
    echo "<$x>"                # <a b c d>        — single element joined
done

# Rule: 99% of the time, you want "${arr[@]}".
```

### Append, modify, delete

```bash
arr=(a b c)
arr+=(d e)                     # append
arr[0]="A"                     # modify in place
unset 'arr[1]'                 # delete (leaves a hole)
echo "${arr[@]}"               # A c d e
echo "${!arr[@]}"              # 0 2 3 4 — index 1 is gone
arr=("${arr[@]}")              # reindex (compact)
echo "${!arr[@]}"              # 0 1 2 3
```

### Iteration patterns

```bash
arr=(one two three)

# Values:
for x in "${arr[@]}"; do echo "$x"; done

# Indices and values:
for i in "${!arr[@]}"; do echo "$i = ${arr[$i]}"; done

# C-style:
for ((i=0; i<${#arr[@]}; i++)); do echo "$i = ${arr[$i]}"; done
```

## Associative Arrays

### bash 4+ only

```bash
declare -A config              # MUST be declared before use; bash 3 does not support this

config[host]="localhost"
config[port]=5432
config[user]="postgres"

# Assignment-block:
declare -A m=(
    [host]="localhost"
    [port]=5432
    [user]="postgres"
)
```

### Access

```bash
echo "${config[host]}"                     # localhost
echo "${config[missing]:-not set}"         # default for missing keys
echo "${!config[@]}"                       # keys (order is undefined!)
echo "${config[@]}"                        # values
echo "${#config[@]}"                       # count

# Existence check (key exists, value may be empty):
[[ -v config[host] ]] && echo "host is set"

# Pre-bash-4.3 alternative — use indirect expansion:
key="host"
[[ -n "${config[$key]+_}" ]] && echo "set"
```

### Iteration

```bash
for k in "${!config[@]}"; do
    echo "$k = ${config[$k]}"
done

# To iterate in a stable order, sort the keys:
for k in $(printf '%s\n' "${!config[@]}" | sort); do
    echo "$k = ${config[$k]}"
done
```

### Delete

```bash
unset 'config[host]'                       # one entry — quote the brackets!
unset 'config'                              # the whole map
config=()                                   # reset to empty (declare -A still applies)
```

## Conditional Tests

### `[[ ]]` vs `[ ]` vs `test`

```bash
[[ "$a" == "$b" ]]              # bash builtin — preferred in bash scripts
[ "$a" = "$b" ]                 # POSIX builtin — works in /bin/sh
test "$a" = "$b"                # synonym for [
```

### What only `[[ ]]` gives you

```bash
[[ "$s" =~ ^[0-9]+$ ]]          # regex match — no equivalent in [
[[ "$a" == abc* ]]              # glob match — RHS is unquoted to be a pattern; quote it for literal
[[ "$a" < "$b" ]]               # string lex compare — in [ this would mean redirection
[[ -n "$a" && "$a" != "$b" ]]  # && and || inside test (in [ you must chain with -a -o which are deprecated)
[[ -z $unset ]]                 # NO word-splitting on unquoted vars inside [[ ]]
```

### File tests

```bash
[[ -e path ]]                  # exists (any type)
[[ -f path ]]                  # regular file
[[ -d path ]]                  # directory
[[ -L path ]]                  # symlink (does NOT follow)
[[ -h path ]]                  # synonym for -L
[[ -r path ]]                  # readable
[[ -w path ]]                  # writable
[[ -x path ]]                  # executable
[[ -s path ]]                  # exists and size > 0
[[ -z "$str" ]]                # zero-length string
[[ -n "$str" ]]                # non-zero-length
[[ a -nt b ]]                  # a newer than b (mtime)
[[ a -ot b ]]                  # a older than b
[[ a -ef b ]]                  # same file (same inode + device)
[[ -p path ]]                  # named pipe
[[ -S path ]]                  # socket
[[ -t 0 ]]                     # fd 0 is a terminal — useful for "am I in a pipe?"
```

### Numeric tests

```bash
[[ $a -eq $b ]]                # equal           (bash will work but (( )) is cleaner)
[[ $a -ne $b ]]                # not equal
[[ $a -lt $b ]]                # less than
[[ $a -le $b ]]                # less or equal
[[ $a -gt $b ]]                # greater than
[[ $a -ge $b ]]                # greater or equal

# More idiomatic — arithmetic context:
(( a == b ))
(( a < b ))
(( a >= 10 && a <= 20 ))
```

### String tests

```bash
[[ "$a" == "$b" ]]             # equal — BOTH = and == work in [[
[[ "$a" != "$b" ]]             # not equal
[[ "$s" < "$t" ]]              # lex less than
[[ "$s" > "$t" ]]              # lex greater than
[[ -z "$s" ]]                  # empty
[[ -n "$s" ]]                  # non-empty

# Glob match (RHS is a pattern):
[[ "$f" == *.txt ]]            # ends in .txt
[[ "$f" == *.[ch] ]]           # .c or .h

# Regex match — RHS is ERE:
[[ "$ip" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]
```

## Control Flow

### if / elif / else

```bash
if [[ -f config.yml ]]; then
    echo "yaml"
elif [[ -f config.json ]]; then
    echo "json"
else
    echo "no config"
    exit 1
fi

# One-liner short-circuits:
[[ -f file.txt ]] && process file.txt
command -v jq >/dev/null || { echo "jq required" >&2; exit 1; }
```

### case ... esac

```bash
case "$1" in
    start|run)         do_start ;;
    stop|kill)         do_stop ;;
    reload|HUP)        do_reload ;;
    [0-9]*)            echo "starts with digit" ;;
    *.tar.gz|*.tgz)    tar -xzf "$1" ;;
    -h|--help)         show_help ;;
    *)                 echo "unknown: $1" >&2; exit 64 ;;
esac
```

### Fall-through: ;; vs ;& vs ;;& (bash 4+)

```bash
case "$x" in
    a) echo "got a" ;;
    b) echo "got b" ;&        # ;& — fall through unconditionally to next clause's body
    c) echo "fell to c" ;;
    d) echo "got d" ;;&       # ;;& — re-test remaining patterns
    [d-f]) echo "in d-f" ;;
    *) echo "default" ;;
esac
```

## Loops

### for ... in list

```bash
# Static list:
for fruit in apple banana cherry; do
    echo "$fruit"
done

# Glob — quote the var, NOT the glob:
for f in *.log; do
    [[ -e "$f" ]] || continue       # if no matches, *.log expands to itself
    gzip "$f"
done

# Brace expansion:
for i in {1..10}; do echo "$i"; done
for ip in 10.0.0.{1..254}; do ping -c1 "$ip"; done

# Reading word-split output (rarely a good idea — prefer mapfile):
for f in $(ls); do echo "$f"; done   # breaks on filenames with spaces
mapfile -t files < <(ls -1)
for f in "${files[@]}"; do echo "$f"; done

# Sequence (use {x..y} when possible — no fork):
for i in $(seq 1 10); do echo "$i"; done   # forks seq
for ((i=1; i<=10; i++)); do echo "$i"; done # built-in
```

### C-style for

```bash
for ((i = 0; i < 10; i++)); do
    echo "$i"
done

for ((i = 1, j = 10; i <= j; i++, j--)); do
    echo "i=$i j=$j"
done
```

### while / until

```bash
i=0
while (( i < 5 )); do
    echo "$i"
    ((i++))
done

# until — loop while condition is FALSE:
until ping -c1 -W1 host >/dev/null 2>&1; do
    echo "waiting for host..."
    sleep 1
done

# Read file line by line — IFS= and -r are critical:
while IFS= read -r line; do
    echo ">>> $line"
done < input.txt
```

### break, continue

```bash
for i in {1..10}; do
    (( i == 3 )) && continue        # skip iteration
    (( i == 7 )) && break           # exit loop
    echo "$i"
done

# Break N levels:
for i in 1 2 3; do
    for j in a b c; do
        [[ "$i$j" == "2b" ]] && break 2
    done
done
```

## Functions

### Definition

```bash
greet() {
    local name="${1:?name required}"
    local greeting="${2:-Hello}"
    echo "$greeting, $name!"
    return 0
}

# `function` keyword — bash extension, prefer the parens form:
function legacy_greet { echo "hi $1"; }    # works but non-portable
```

### Parameters

```bash
demo() {
    echo "name: $0"             # NOT the function name — it's the script name
    echo "function: $FUNCNAME"  # function name
    echo "first:    $1"
    echo "second:   $2"
    echo "tenth:    ${10}"      # MUST use braces past 9
    echo "count:    $#"
    echo "all:      $@"
    echo "all-1:    $*"
}
demo a b c d e f g h i j k
```

### `"$@"` vs `"$*"`

```bash
forward() {
    target_command "$@"             # CORRECT — preserves arg boundaries
    target_command "$*"             # WRONG — collapses everything into 1 arg
    target_command $@               # WRONG — word-splits each arg, glob-expanded
}
```

### Returning values

```bash
# return — sets $? (exit status), only 0..255
is_even() {
    (( $1 % 2 == 0 )) && return 0
    return 1
}
is_even 4 && echo "yes"

# Output via echo — capture with $( )
to_upper() {
    echo "${1^^}"
}
result=$(to_upper "hello")
echo "$result"                  # HELLO

# Output to a passed nameref (bash 4.3+) — avoids subshell + fork:
upper_into() {
    local -n out=$1
    out="${2^^}"
}
upper_into result "hello"
echo "$result"                  # HELLO
```

### Local variable safety

```bash
fn() {
    local i j tmp                # multiple in one line
    local count=0
    local arr=()                  # local array
    local -A map                  # local associative
    # ...
}
```

### Common pitfall: `local x=$(...)` masks errors

```bash
# BAD:
fn() {
    local x=$(false)            # status of `false` is masked — local always returns 0
    echo "x=$x rc=$?"            # rc=0 even though false failed
}

# GOOD: declare and assign separately
fn() {
    local x
    x=$(false) || return         # now $? is preserved on the assignment
    echo "$x"
}
```

## Subshells vs Grouping

### `( cmd )` — subshell

```bash
(cd /tmp && rm -rf cache)        # cd doesn't affect parent shell
(set -e; risky; risky2)          # errexit only inside the subshell

# Variables set in subshell don't leak out:
x=1
(x=2; echo "inside: $x")         # inside: 2
echo "outside: $x"               # outside: 1
```

### `{ cmd; }` — group, no subshell

```bash
# Note: space after { and ; before } are MANDATORY
{ echo a; echo b; } > out.txt    # redirect both lines together

# Variables persist:
x=1
{ x=2; echo "inside: $x"; }      # inside: 2
echo "outside: $x"               # outside: 2
```

### Subshell side-effects

```bash
# Ignored: cd, exec, set, ulimit, trap, function definitions
# Preserved: stdout/stderr (until the subshell ends)
# Cost: a fork. In hot loops avoid subshells where you can.

# BASH_SUBSHELL tracks depth:
echo "depth: $BASH_SUBSHELL"     # 0
( ( echo "depth: $BASH_SUBSHELL" ) )   # depth: 2
```

## Pipes and Pipelines

### Basics

```bash
cmd1 | cmd2                     # stdout of cmd1 → stdin of cmd2
cmd1 |& cmd2                    # stdout AND stderr of cmd1 → stdin of cmd2 (bash 4+)
                                # equivalent to: cmd1 2>&1 | cmd2
cmd1 | cmd2 | cmd3              # chain — exit status is cmd3's by default
```

### `set -o pipefail`

```bash
# Without pipefail, pipeline status is LAST command's status:
false | true; echo $?            # 0 — false's failure is invisible
set -o pipefail
false | true; echo $?            # 1 — last NON-ZERO wins

# Always combine with -e for fail-fast:
set -euo pipefail
```

### `PIPESTATUS` — exit code per stage

```bash
false | true | grep -q something
echo "${PIPESTATUS[@]}"          # 1 0 1 — exit code of each stage in order
echo "${PIPESTATUS[0]}"          # 1 — first stage

# PIPESTATUS is reset by every command. Save it immediately if you need it:
some | pipe | line
rcs=( "${PIPESTATUS[@]}" )
(( rcs[0] != 0 )) && echo "first stage failed"
```

### Subshell scope in pipelines

```bash
# Each side of a pipe runs in a SUBSHELL — variables set inside don't leak:
count=0
seq 5 | while read -r line; do ((count++)); done
echo "$count"                    # 0 — the count++ happened in the right-hand subshell

# Fix 1: process substitution
count=0
while read -r line; do ((count++)); done < <(seq 5)
echo "$count"                    # 5 — while-loop now in main shell

# Fix 2: shopt -s lastpipe (bash 4.2+, only in non-interactive mode)
shopt -s lastpipe
count=0
seq 5 | while read -r line; do ((count++)); done
echo "$count"                    # 5
```

## Redirection

### Stdout, stderr, append, combine

```bash
cmd > out.txt                    # stdout to file (truncate)
cmd >> out.txt                   # stdout to file (append)
cmd 2> err.txt                   # stderr to file
cmd 2>> err.txt                  # stderr append
cmd > out.txt 2> err.txt         # split
cmd > all.txt 2>&1               # both to same file — ORDER MATTERS
cmd 2>&1 > all.txt               # WRONG — stderr → original stdout, then stdout → file
cmd &> all.txt                   # bash shorthand: stdout + stderr to file (truncate)
cmd &>> all.txt                  # bash shorthand: stdout + stderr append
cmd > out.txt 2>&1               # CORRECT — redirect stdout first, then stderr → wherever stdout goes
```

### Discard output

```bash
cmd > /dev/null                 # silence stdout
cmd 2> /dev/null                # silence stderr
cmd > /dev/null 2>&1            # silence everything (POSIX)
cmd &> /dev/null                # silence everything (bash)
```

### Read from file

```bash
cmd < input.txt                 # stdin from file
cmd < <(other_cmd)              # stdin from process substitution
cmd <<< "$variable"             # stdin from here-string (newline appended)
```

### Open custom file descriptors

```bash
exec 3> log.txt                 # open fd 3 for write
echo "log entry" >&3
exec 3>&-                       # close fd 3

exec 4< input.txt
read -r line <&4
exec 4<&-

# Save and restore stdout:
exec 5>&1                       # save stdout to fd 5
exec > silence.log              # redirect stdout
echo "this goes to silence.log"
exec 1>&5 5>&-                  # restore, close 5
echo "back to terminal"
```

### bash special: /dev/tcp and /dev/udp

```bash
# Open a raw TCP connection — bash builtin, no nc required:
exec 3<> /dev/tcp/example.com/80
printf 'GET / HTTP/1.0\r\nHost: example.com\r\n\r\n' >&3
cat <&3
exec 3<&-

# Test if a port is open (returns 0 if connection succeeds):
timeout 1 bash -c '</dev/tcp/host/443' && echo open || echo closed
```

## Heredocs and Here-Strings

### `<<EOF` — expanded heredoc

```bash
name="alice"
cat <<EOF
Hello $name
Today is $(date +%F)
Math: $((2 + 2))
EOF
# Variables, command substitution, and arithmetic ARE expanded.
```

### `<<'EOF'` — literal heredoc

```bash
cat <<'EOF'
Hello $name
Today is $(date +%F)
EOF
# Output is literal. Use this for embedded scripts, AWK programs, regex, etc.
```

### `<<-EOF` — strip leading TABS only

```bash
if true; then
    cat <<-EOF
	indented with TABS
	will be stripped
	EOF
fi
# IMPORTANT: leading whitespace must be tabs, not spaces. Useful for indenting heredocs inside if/for.
```

### Here-string `<<<`

```bash
read -r first second <<< "alice bob"
echo "$first / $second"          # alice / bob

# Useful with grep/sed/awk to avoid echoing:
grep "error" <<< "$log"
[[ $(wc -w <<< "$s") -gt 5 ]] && echo "wordy"
```

### Heredoc as multi-line string variable

```bash
read -r -d '' usage <<'EOF'
Usage: prog [-h] [-v] FILE
  -h    show help
  -v    verbose
EOF
echo "$usage"
# read -d '' reads until EOF (null delimiter) — captures heredoc into a single string.
# `read` returns 1 because EOF terminates without delimiter — that's expected.
```

## Process Substitution

### `<(cmd)` — input from a command

```bash
diff <(sort file1) <(sort file2)
comm -12 <(sort a.txt) <(sort b.txt)        # lines in both
join -t, <(sort -t, -k1,1 a.csv) <(sort -t, -k1,1 b.csv)

# Reading into a while-loop without subshell scope:
while IFS= read -r line; do
    process "$line"
done < <(find . -name '*.go')
```

### `>(cmd)` — output to a command

```bash
# Tee to multiple processes:
some_long_cmd | tee >(gzip > out.log.gz) >(grep ERROR > errors.txt) > /dev/null

# Useful for tools that only write to a path argument:
mysqldump db | gzip > >(tee dump.sql.gz | sha256sum > dump.sha256)
```

### Named-pipe (FIFO) alternative

```bash
mkfifo mypipe
cmd1 > mypipe &
cmd2 < mypipe
rm mypipe
# Process substitution generates a /dev/fd/N path automatically — usually preferred.
```

## Command Substitution

### `$(cmd)` — modern, nestable

```bash
today=$(date +%F)
files=$(find /var/log -name '*.gz')
echo "host: $(hostname -s)"

# Nest freely:
echo "outer: $(echo "inner: $(echo deep)")"
```

### `` `cmd` `` — legacy, avoid

```bash
today=`date +%F`                # works, but escapes are awful when nesting
echo "outer: \`echo inner\`"   # backslash hell
# Use `$(...)` everywhere except shells where it's truly unsupported (rare today).
```

### Trailing-newline stripping

```bash
echo "line1
line2
" > file.txt
content=$(cat file.txt)
echo "[$content]"                # [line1\nline2]  — trailing newlines stripped

# To preserve a trailing newline, append a sentinel:
content=$(cat file.txt; printf x)
content=${content%x}              # remove sentinel, keep all newlines including trailing
```

## Brace Expansion

### Sequences and lists

```bash
echo {1..5}                     # 1 2 3 4 5
echo {1..10..2}                 # 1 3 5 7 9      — step
echo {a..e}                     # a b c d e
echo {01..10}                   # 01 02 ... 10  — zero-padded width preserved
echo {Z..A}                     # Z Y X W ...   — descending

echo {a,b,c}                    # a b c
echo file{1,2,3}.txt            # file1.txt file2.txt file3.txt
echo /etc/{hosts,fstab,passwd}  # /etc/hosts /etc/fstab /etc/passwd

# Nested:
echo {a,b}{1,2}                  # a1 a2 b1 b2
echo {a,b}{1..3}                 # a1 a2 a3 b1 b2 b3
```

### Brace expansion happens FIRST

```bash
# Order: brace → tilde → parameter/command/arithmetic → word splitting → globbing
n=3
echo {1..$n}                    # {1..3} — VARIABLE not yet expanded when brace runs!
# Workaround:
eval echo "{1..$n}"             # 1 2 3 (mind the eval — only safe with trusted input)
seq 1 "$n"                      # forks but is variable-aware
for ((i=1; i<=n; i++)); do echo -n "$i "; done; echo
```

### Mass file ops

```bash
mkdir -p ~/projects/{frontend,backend,infra}/{src,test,docs}
mv file.{txt,bak}                # syntax error if both exist as args — wrong number
cp file.txt{,.bak}               # → cp file.txt file.txt.bak (cheap backup idiom)
```

## Globbing and extglob

### Basic globs

```bash
echo *.txt                      # all .txt in cwd (no match → literal *.txt unless nullglob set)
echo file?.txt                  # one-char wildcard — file1.txt, fileA.txt
echo file[123].txt              # any of 1, 2, 3
echo file[a-z].txt              # ranges
echo file[!0-9].txt             # negation: any char that is NOT a digit (! is bash; [^...] is for regex)
```

### shopt knobs

```bash
shopt -s nullglob               # *.nope expands to nothing instead of literal *.nope
shopt -s failglob               # *.nope errors out
shopt -s nocaseglob             # case-insensitive globs
shopt -s dotglob                # * matches dotfiles too
shopt -s globstar               # ** matches across directories
ls **/*.go                      # all .go files recursively (with globstar set)

# Inspect current state:
shopt | grep -E 'nullglob|failglob|dotglob|globstar'
```

### Extended globs (extglob)

```bash
shopt -s extglob

ls !(*.txt)                     # everything that does NOT end in .txt
ls *.+(c|h)                     # *.c or *.h — one or more
ls @(README|readme).?(txt|md)   # exact match: 0 or 1 of the suffix
ls *(0)                         # zero or more 0's
ls ?(file)                      # 0 or 1 'file'

# Patterns:
#   ?(p)   zero or one
#   *(p)   zero or more
#   +(p)   one or more
#   @(p)   exactly one
#   !(p)   NOT
```

## Tilde Expansion

```bash
echo ~                          # $HOME
echo ~root                      # root's home dir
echo ~+                         # $PWD
echo ~-                         # $OLDPWD
echo ~/projects                 # $HOME/projects

# Inside double quotes, ~ does NOT expand:
echo "~/projects"               # literal ~/projects
echo "$HOME/projects"           # use $HOME inside quotes
```

## Word Splitting

### What it is

```bash
# When the shell expands an unquoted variable or command substitution:
#   1. The result is split into words on $IFS (default: space, tab, newline)
#   2. Each word is then glob-expanded
# Quotes prevent BOTH steps.

IFS=$' \t\n'                    # default
files="a.txt b.txt"
ls $files                       # ls a.txt b.txt — two args
ls "$files"                     # ls "a.txt b.txt" — one arg (probably wrong)

f="my file.txt"
rm $f                           # rm my file.txt — TWO args, both wrong
rm "$f"                         # rm "my file.txt" — single arg
```

### Controlling splitting via IFS

```bash
# Split on comma:
csv="a,b,c"
IFS=, read -ra fields <<< "$csv"
echo "${fields[0]}"             # a
echo "${fields[1]}"             # b

# Always restore IFS or scope it:
old_ifs=$IFS
IFS=,
read -ra fields <<< "$csv"
IFS=$old_ifs

# OR keep the assignment local to the command:
IFS=, read -ra fields <<< "$csv"   # only this command sees IFS=,
```

### Empty IFS

```bash
# IFS= (empty) disables word splitting entirely:
IFS= read -r line < file        # entire line, including leading/trailing whitespace
mapfile -t arr < file           # works without IFS dance
```

## Job Control

### Background, foreground, kill

```bash
long_cmd &                      # run in background, print [job_id] PID
echo "started: $!"              # PID of last bg

jobs                             # list jobs in current shell
jobs -l                          # with PIDs
fg %1                            # bring job 1 to foreground
fg                               # foreground last job
bg %2                            # resume job 2 in background

kill %1                         # signal job 1
kill -9 %1                      # SIGKILL job 1
kill -INT $!                    # SIGINT to last bg PID

disown %1                       # detach job 1 from shell — survives shell exit
nohup long_cmd &                # alternate: ignore SIGHUP from start
```

### wait

```bash
long1 &
long2 &
long3 &
wait                             # wait for ALL background jobs
echo "all done"

# Wait for one specific job:
long1 & p1=$!
long2 & p2=$!
wait "$p1"                       # exits with that job's status

# Wait for the next to finish (bash 4.3+):
fast & slow &
wait -n                          # returns as soon as any one finishes
echo $?                          # exit code of whichever finished first
```

## Signals and Traps

### `trap` syntax

```bash
trap 'cmd' SIGNAL [SIGNAL ...]   # run cmd on each named signal
trap - SIGNAL                    # reset to default
trap '' SIGNAL                   # ignore signal (cannot be done for SIGKILL/SIGSTOP)
trap -p                          # list all current traps
```

### EXIT — your cleanup friend

```bash
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT     # always runs: normal exit, error exit, set -e, signals
# ...do work in $tmpdir...
```

### ERR — debug a script

```bash
trap 'echo "ERR: line $LINENO: $BASH_COMMAND (exit $?)" >&2' ERR
set -e
# Any failing command now prints a useful one-line trace.
```

### INT and TERM — graceful shutdown

```bash
cleanup() {
    echo "shutting down..."
    kill $(jobs -p) 2>/dev/null
    wait
    exit 130                     # 128 + SIGINT(2)
}
trap cleanup INT TERM
```

### Common signals

```bash
kill -l                          # list signal names
# 1  HUP   — hangup (terminal closed) — daemons reload config
# 2  INT   — interrupt (Ctrl-C)
# 3  QUIT  — quit (Ctrl-\), with core dump
# 9  KILL  — uncatchable, unblockable, untrappable
# 15 TERM  — graceful terminate (default for kill)
# 17 CHLD  — child stopped/exited (Linux x86_64)
# 18 CONT  — resume from stop
# 19 STOP  — uncatchable suspend
# 20 TSTP  — Ctrl-Z
```

## getopts

### Single-char only

```bash
verbose=0
output=""
while getopts ":vo:h" opt; do
    case "$opt" in
        v) verbose=1 ;;
        o) output="$OPTARG" ;;
        h) echo "Usage: $0 [-v] [-o file]"; exit 0 ;;
        :) echo "Option -$OPTARG requires an argument" >&2; exit 64 ;;
        \?) echo "Unknown option -$OPTARG" >&2; exit 64 ;;
    esac
done
shift $((OPTIND - 1))            # leave only positional args
echo "verbose=$verbose output=$output args=$*"
```

### Limitations

```bash
# - No long options (--verbose, --output=file). For those, write your own loop (see "Argument Parsing Best Practices").
# - The leading colon in ":vo:h" enables silent error mode — your case branches handle errors instead of getopts printing.
# - OPTARG holds the option argument when option requires one (o:).
# - OPTIND is the index of the next argument to be processed; reset it (OPTIND=1) if you call getopts again in the same shell.
```

## Reading Input

### `read` — line, fields, prompt, timeout, silent

```bash
read -r line                            # one line from stdin; -r preserves backslashes (ALWAYS use -r)
read -r -p "Name: " name                # with prompt
read -rs -p "Password: " pw; echo       # silent (no echo) — for passwords
read -r -t 5 ans || ans="default"       # 5-second timeout
read -r -n 1 ans                        # exactly 1 char (no Enter needed)
IFS=, read -ra fields                   # split on , into array fields
IFS=$'\t' read -ra cols                 # tab-separated
```

### Read a file line by line — the only safe pattern

```bash
while IFS= read -r line; do
    echo "[$line]"
done < input.txt
# - IFS= prevents trimming of leading/trailing whitespace
# - -r prevents \-escapes from being interpreted
# - omitting either is a bug
```

### Slurp file into array

```bash
mapfile -t lines < file.txt              # each line → element, no trailing newline
readarray -t lines < file.txt            # synonym
echo "${#lines[@]}"                       # line count
echo "${lines[0]}"                        # first line

# From a command:
mapfile -t pids < <(pgrep nginx)
```

### Confirm/prompt idiom

```bash
read -r -p "Continue? [y/N] " ans
case "${ans,,}" in
    y|yes) echo "ok" ;;
    *)     echo "abort"; exit 0 ;;
esac
```

## Strings

### Concatenation by adjacency

```bash
a="hello"
b="world"
c="$a $b"                       # "hello world"
c=$a$b                          # "helloworld"
c+=" extra"                     # append (in-place)
```

### Length and slicing

```bash
s="hello, world"
echo "${#s}"                    # 12
echo "${s:0:5}"                 # "hello"
echo "${s: -5}"                 # "world" (note space)
```

### Splitting

```bash
csv="a,b,c"
IFS=, read -ra parts <<< "$csv"
for p in "${parts[@]}"; do echo "<$p>"; done

# By any delimiter, multiple-char — bash can't directly. Use awk:
echo "abXXcdXXef" | awk -F"XX" '{for(i=1;i<=NF;i++) print $i}'

# Or replace delimiter with a single char first:
s="abXXcdXXef"
IFS=$'\n' read -d '' -ra parts <<< "${s//XX/$'\n'}"
```

### Trimming whitespace (extglob)

```bash
shopt -s extglob
s="   hello, world   "
echo "[${s##+([[:space:]])}]"    # left-trim:   [hello, world   ]
echo "[${s%%+([[:space:]])}]"    # right-trim:  [   hello, world]
trimmed="${s##+([[:space:]])}"
trimmed="${trimmed%%+([[:space:]])}"
echo "[$trimmed]"                  # [hello, world]
```

## printf

### Common format specifiers

```bash
printf '%s\n' "$x"               # string
printf '%d\n' 42                 # integer
printf '%05d\n' 42                # 00042 — zero-padded
printf '%-10s|%s\n' a b           # left-aligned in 10-char field
printf '%x\n' 255                 # ff (hex)
printf '%o\n' 8                   # 10 (octal)
printf '%e\n' 1234567.89          # 1.234568e+06
printf '%.2f\n' 3.14159           # 3.14
printf '%b\n' 'a\tb\nc'           # interprets backslash escapes (\t \n \\ \xHH)
printf '%q\n' "it's a 'value'"    # shell-quoted — paste-safe
```

### Format reuse for arrays

```bash
arr=(one two three)
printf '%s\n' "${arr[@]}"        # one\ntwo\nthree — format reapplied per arg
printf '%-10s = %s\n' name alice age 30 city sf
```

### printf vs echo

```bash
echo -n "no newline"             # -n suppresses newline — but NOT POSIX, fails on some shells
echo -e "esc\nape"               # -e enables escapes — also non-portable
# printf is portable AND predictable:
printf 'no newline'
printf 'esc\nape\n'
```

## echo vs printf

```bash
# echo is a builtin in bash but its behavior varies across shells:
#   - bash echo recognizes -n and -e (with extra escapes)
#   - dash echo does NOT recognize -e — prints literally
#   - some echo implementations treat -e as data
# Conclusion: in scripts, use printf. echo is fine for interactive use.

printf '%s\n' "$msg"             # always works
printf '%s' "$msg"                # no newline
```

## File Operations

### Test before action

```bash
[[ -f "$file" ]] || { echo "missing: $file" >&2; exit 1; }
[[ -d "$dir"  ]] || mkdir -p "$dir"
[[ -r "$file" ]] || { echo "cannot read: $file" >&2; exit 1; }
[[ -w "$dir"  ]] || { echo "cannot write: $dir" >&2; exit 1; }
```

### Common operations

```bash
touch "$file"                   # create empty / update mtime
mkdir -p "$dir"                 # create incl. parents, no error if exists
mv "$src" "$dst"
cp -r "$src" "$dst"             # recursive
rm -rf "$dir"                   # NEVER with unquoted vars or wildcards from user input
ln -s "$target" "$linkname"     # symlink
ln "$src" "$hardlink"           # hard link
chmod 644 "$file"
chmod +x "$script"
chown user:group "$file"
stat "$file"
file "$file"                    # detect file type
```

### Quoting paths — every. single. time.

```bash
# THIS DELETES THE WRONG THING:
dir="my photos"
rm -rf $dir/*                   # → rm -rf my photos/* — also tries to glob "my" and "photos/*"

# Correct:
rm -rf -- "$dir"/*              # -- ends options (defends against names like -i)
```

## Date Arithmetic

### Epoch

```bash
date +%s                        # epoch seconds (since 1970-01-01 UTC)
date -u +%FT%TZ                 # ISO 8601 UTC: 2024-05-21T14:32:01Z
date +"%Y-%m-%d %H:%M:%S"       # human-readable
```

### GNU date (Linux)

```bash
date -d "+1 day"                 # tomorrow same time
date -d "+1 hour"                # an hour from now
date -d "2024-05-01"             # parse a date
date -d "@1700000000"            # interpret epoch
date -d "yesterday" +%F          # 2024-05-20
date -d "next monday" +%F
EPOCH=1700000000 date -d "@$EPOCH" +"%F %T %Z"
```

### macOS / BSD date (different flag!)

```bash
# GNU:    date -d "+1 day"
# BSD:    date -v +1d
date -v +1d                     # tomorrow
date -v +1m                     # one month
date -v -1d                     # yesterday
date -j -f "%Y-%m-%d" "2024-05-01" +%s    # parse to epoch
date -r 1700000000              # epoch → human (no -d)

# Cross-platform: use python or perl, or detect:
if date --version >/dev/null 2>&1; then     # GNU
    yesterday=$(date -d "yesterday" +%F)
else                                          # BSD
    yesterday=$(date -v -1d +%F)
fi
```

## Regex

### `=~` inside `[[ ]]`

```bash
ip="10.0.0.1"
if [[ "$ip" =~ ^([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})$ ]]; then
    echo "valid: ${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${BASH_REMATCH[3]}.${BASH_REMATCH[4]}"
fi
# BASH_REMATCH[0] = full match
# BASH_REMATCH[1..N] = capture groups
```

### Important: do NOT quote the regex

```bash
[[ "$s" =~ ^[0-9]+$ ]]           # CORRECT — RHS is a regex pattern
[[ "$s" =~ "^[0-9]+$" ]]         # WRONG — quotes turn it into a literal string match
# If you need to embed a literal in a regex, store it in a var:
re='^[0-9]+$'
[[ "$s" =~ $re ]]                # var on RHS is NOT word-quoted as literal
```

### POSIX character classes (ERE)

```bash
[[ "$s" =~ ^[[:alpha:]]+$ ]]
[[ "$s" =~ ^[[:digit:]]+$ ]]
[[ "$s" =~ ^[[:alnum:]_]+$ ]]
[[ "$s" =~ [[:space:]] ]]
```

### When to drop down to sed/awk/grep -E

```bash
# Bash regex is ERE only — no backreferences (\1) inside the pattern. Use:
echo "$s" | sed -E 's/(.+)@(.+)/\2 -> \1/'
echo "$s" | grep -E '^([0-9]+)\.([0-9]+)$'
echo "$s" | awk 'match($0,/[a-z]+/){print substr($0,RSTART,RLENGTH)}'
```

## Locking

### `flock` — file-based mutex

```bash
lock=/var/lock/myjob.lock
(
    flock -n 9 || { echo "already running"; exit 1; }
    # critical section ...
    sleep 60
) 9> "$lock"
# fd 9 stays open inside the subshell; flock holds it until shell exits.
# -n = non-blocking (fail immediately if held)
# -w 30 = wait up to 30s
# -x = exclusive (default)
# -s = shared
```

### Single-instance script idiom

```bash
LOCK=/var/lock/$(basename "$0").lock
exec 9> "$LOCK"
flock -n 9 || { echo "another instance running" >&2; exit 1; }
# rest of the script runs with the lock held until exit
```

## Sourcing vs Executing

### `.` and `source`

```bash
source ./lib.sh                  # bash builtin
. ./lib.sh                       # POSIX synonym
# Both: read the file in the CURRENT shell. Functions, variables, and aliases persist.
# Use for: config files, function libraries, dotfiles.

bash ./script.sh                 # NEW shell — variables don't leak back
./script.sh                      # exec via shebang — also new process
```

### Self-locating scripts

```bash
# Get the directory of the running script (works whether invoked or sourced):
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
# - ${BASH_SOURCE[0]} works in both sourced libs (where $0 is the parent shell) and scripts
# - cd into the dir, then pwd, gets a canonical absolute path
# - On macOS, no realpath needed for this idiom

source "$SCRIPT_DIR/lib.sh"
```

## Argument Parsing Best Practices

### The "manual long-flag parser" idiom

```bash
verbose=0
output=""
files=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        -v|--verbose)        verbose=1; shift ;;
        -o|--output)         output="$2"; shift 2 ;;
        --output=*)          output="${1#*=}"; shift ;;
        -h|--help)           show_help; exit 0 ;;
        --)                  shift; files+=("$@"); break ;;        # stop processing
        -*)                  echo "Unknown option: $1" >&2; exit 64 ;;
        *)                   files+=("$1"); shift ;;
    esac
done

# After the loop, "${files[@]}" holds positional args.
echo "verbose=$verbose output=$output files=${files[*]}"
```

### Why not getopt(1)?

```bash
# - GNU getopt supports long options but is non-portable (BSD getopt is different).
# - Output requires `eval` which is dangerous with untrusted args.
# - The manual loop is safer, more transparent, and more flexible.
```

## Logging Idiom

### Stderr for messages, stdout for data

```bash
# Rule: scripts intended for piping should ONLY emit data on stdout.
# Logs, progress, errors → stderr.

log()  { printf '[%s] %s\n' "$(date +%FT%T)" "$*" >&2; }
warn() { printf '[%s] WARN: %s\n' "$(date +%FT%T)" "$*" >&2; }
err()  { printf '[%s] ERR:  %s\n' "$(date +%FT%T)" "$*" >&2; }
die()  { err "$@"; exit 1; }

log "starting work"
warn "config missing — using defaults"
err "could not connect"
die "fatal: $msg"
```

### Levels via env var

```bash
LOG_LEVEL=${LOG_LEVEL:-info}
declare -A LEVELS=( [debug]=0 [info]=1 [warn]=2 [error]=3 )

log() {
    local level=$1; shift
    (( LEVELS[$level] >= LEVELS[$LOG_LEVEL] )) || return 0
    printf '[%s] %5s %s\n' "$(date +%FT%T)" "${level^^}" "$*" >&2
}

log debug "loaded config"
log info  "ready"
log warn  "deprecated flag"
log error "exiting"
```

### Optional color when stderr is a TTY

```bash
if [[ -t 2 ]]; then
    RED=$'\e[31m' YEL=$'\e[33m' GRN=$'\e[32m' RST=$'\e[0m'
else
    RED='' YEL='' GRN='' RST=''
fi
err()  { printf '%sERR%s  %s\n' "$RED" "$RST" "$*" >&2; }
warn() { printf '%sWARN%s %s\n' "$YEL" "$RST" "$*" >&2; }
ok()   { printf '%sOK%s   %s\n' "$GRN" "$RST" "$*" >&2; }
```

## Defensive Patterns

### EXIT trap for cleanup

```bash
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
# Whatever happens — error, signal, normal — $tmpdir is cleaned up.
```

### `mktemp` for safe temp files

```bash
tmpfile=$(mktemp)                       # /tmp/tmp.AbCdEf
tmpfile=$(mktemp /tmp/myapp.XXXXXX)     # custom prefix
tmpdir=$(mktemp -d)                     # directory
trap 'rm -rf "$tmpfile" "$tmpdir"' EXIT

# NEVER use predictable names:  tmp=/tmp/myapp.$$        # symlink-attack target
```

### Fail-fast preamble

```bash
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Ensure required commands exist:
for cmd in jq curl awk; do
    command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 1; }
done

# Ensure required env:
: "${API_TOKEN:?API_TOKEN must be set}"
: "${API_URL:=https://api.example.com}"
```

### Explicit return codes

```bash
# Bash exit codes (sysexits.h conventions):
# 0   success
# 1   general error
# 2   misuse of shell builtin
# 64  EX_USAGE     — wrong CLI usage
# 65  EX_DATAERR   — bad input data
# 69  EX_UNAVAILABLE — service unavailable
# 70  EX_SOFTWARE  — internal error
# 77  EX_NOPERM    — permission denied
# 130 SIGINT       — Ctrl-C
# 137 SIGKILL      — killed (128 + 9)
# 143 SIGTERM      — terminated (128 + 15)
```

### Shift after consuming arg

```bash
while [[ $# -gt 0 ]]; do
    case "$1" in
        -o) output="$2"; shift 2 ;;     # consume both -o and its arg
        -v) verbose=1; shift ;;          # consume only -v
        *)  break ;;                     # rest are positional
    esac
done
# After the loop, "$@" is just positional args.
```

## Common Gotchas

### Unquoted vars

```bash
# THE most common bash bug:
file="my report.pdf"
[ -f $file ] && rm $file        # both are wrong: word-splitting
[[ -f "$file" ]] && rm "$file"  # correct
```

### `[` vs `[[`

```bash
# [[ ]] is bash, smarter, no word-splitting, supports =~ &&  ||
# [ ] is the test command — POSIX-portable, but trips on empty/whitespace vars
[ "$x" = "$y" ]                  # POSIX safe — note quotes
[[ $x == $y ]]                   # bash, no quotes needed
```

### `set -e` and command substitution

```bash
set -e
x=$(false; true)                 # x is set, $? is 0 — false's failure was in a subshell
echo "ok"                         # still runs
# If you need fail-fast inside a subshell, repeat the trap:
x=$(set -e; might_fail; echo "$something")
```

### `set -e` and pipelines (without pipefail)

```bash
set -e
false | true                     # exit code 0 — script continues
set -o pipefail
false | true                     # exit 1 — script aborts (with -e)
```

### `set -e` and functions called in conditions

```bash
set -e
fn() { false; echo "still here"; }
fn || true                       # both lines of fn run; -e is suppressed in conditional context
fn                                # script exits at the false
```

### Subshell scope: `var=$(cmd)` vs `cmd | read var`

```bash
# var=$(cmd) — runs cmd in subshell but var IS set in parent (the assignment is the parent's)
total=$(wc -l < big.txt)         # OK — parent has total

# cmd | read var — read runs in a subshell, parent never sees it (without lastpipe)
echo "hello" | read var
echo "var=$var"                  # empty (in default mode)

# Fix:
read var <<< "$(echo "hello")"
read -r var < <(echo "hello")
```

### Here-string adds a trailing newline

```bash
s="hello"
wc -c <<< "$s"                   # 6 — that's 5 + the newline
printf '%s' "$s" | wc -c         # 5
```

### IFS leakage

```bash
# Setting IFS for one command is safe:
IFS=, read -ra parts <<< "$csv"

# But this leaks for the rest of the shell:
IFS=,
read -ra parts <<< "$csv"
# Now everywhere else IFS=, until reset.

# Save/restore or scope tightly:
oldifs=$IFS; IFS=,; ... ; IFS=$oldifs
```

### Reading until newline misses last line without newline

```bash
# If the file does NOT end with a newline, the last line is silently dropped:
while IFS= read -r line; do echo "$line"; done < file
# Fix:
while IFS= read -r line || [[ -n "$line" ]]; do echo "$line"; done < file
```

### `BASH_SUBSHELL` to detect subshell

```bash
# 0 in main shell, 1+ in subshells:
echo $BASH_SUBSHELL              # 0
( echo $BASH_SUBSHELL )           # 1
( ( echo $BASH_SUBSHELL ) )       # 2
```

## Performance

### Avoid forks — prefer builtins

```bash
# Slow (forks):
upper=$(echo "$s" | tr a-z A-Z)
len=$(echo "$s" | wc -c)
basename "$path"
dirname "$path"

# Fast (builtin / parameter expansion):
upper="${s^^}"
len="${#s}"
base="${path##*/}"
dir="${path%/*}"
```

### `printf` over `echo` loop for arrays

```bash
arr=(a b c d)
# Slow:
for x in "${arr[@]}"; do echo "$x"; done
# Fast:
printf '%s\n' "${arr[@]}"
```

### Here-strings over echo|

```bash
# Slow (forks /bin/echo, then a pipe):
grep "needle" <<<"$haystack"

# vs. echo | grep — also slow due to pipe + 2 procs
echo "$haystack" | grep "needle"

# <<< is fastest if you only need stdin once.
```

### Built-in arithmetic over `expr`

```bash
n=$((n + 1))                     # builtin — no fork
n=$(expr "$n" + 1)               # forks expr — slow in loops
```

### Read whole file into memory once

```bash
# Slow (open/close per line):
while IFS= read -r line; do ...; done < big.txt

# Fast (when you can fit it):
mapfile -t lines < big.txt
for line in "${lines[@]}"; do ...; done
```

## Portability

### POSIX vs bash

```bash
# Bash extensions that DON'T work in /bin/sh:
#   [[ ]]               — use [ ]
#   (( ))               — use $(( )) inside [ ]
#   arrays              — POSIX has only $1..$N
#   ${var^^} ${var,,}   — use tr
#   <<< here-string     — use printf '%s\n' "$x" |
#   $'...' ANSI-C       — use printf '\n' or actual escapes
#   ==                  — use =
#   |&                  — use 2>&1 |
#   &>                  — use > x 2>&1
#   <(cmd) >(cmd)       — use named pipes (mkfifo)
#   shopt               — set -o where applicable
#   declare/local       — POSIX has no `local`
#   coproc              — no equivalent

# If you need /bin/sh portability: shebang is #!/bin/sh and run shellcheck -s sh.
```

### `#!/bin/sh` vs `#!/usr/bin/env bash`

```bash
#!/bin/sh                  # POSIX portability — many systems link sh → dash, ash, busybox
#!/bin/bash                # bash specifically, but path may differ (macOS Homebrew /opt/homebrew/bin/bash)
#!/usr/bin/env bash        # bash from $PATH — portable, recommended for bash scripts
```

### GNU vs BSD coreutils

```bash
# Each of these has subtle differences:
date -d "+1 day"            (GNU)    vs   date -v +1d         (BSD)
sed -i 's/x/y/' f           (GNU)    vs   sed -i '' 's/x/y/'  (BSD)
readlink -f f               (GNU)    vs   not in BSD (use realpath or python)
stat -c '%s' f              (GNU)    vs   stat -f '%z' f      (BSD)
xargs -d '\n'               (GNU)    vs   not in BSD (use -0 + tr)
ls --color                  (GNU)    vs   ls -G               (BSD)
grep -P                     (GNU)    vs   not in BSD (use perl/ag)
tac                          (GNU)    vs   tail -r            (BSD)

# Detect:
if sed --version >/dev/null 2>&1; then SED_INPLACE=( -i );      else SED_INPLACE=( -i '' ); fi
sed "${SED_INPLACE[@]}" 's/x/y/' file
```

## shellcheck

### Run it, every time

```bash
shellcheck script.sh
shellcheck -s bash script.sh         # force bash dialect
shellcheck -s sh   script.sh         # POSIX dialect
shellcheck -x script.sh              # follow `source`d files
```

### Top rules to know by SC number

```bash
# SC2086  — Double-quote to prevent globbing and word splitting.
#           Fix: rm "$file" instead of rm $file.

# SC2155  — Declare and assign separately to avoid masking return values.
#           Fix: local x; x=$(cmd) instead of local x=$(cmd).

# SC2016  — Expressions don't expand in single quotes — use double quotes.
#           Often a false positive (you ARE intending literal). Add # shellcheck disable=SC2016 if so.

# SC2034  — Foo appears unused. Verify it or export it.
# SC2046  — Quote this to prevent word splitting.   x=$(cmd) is fine; for x in $(cmd) is the problem.
# SC2059  — Don't use variables in the printf format string.
#           Fix: printf '%s\n' "$msg" instead of printf "$msg".
# SC2128  — Expanding an array without an index gives only the first element.   Use "${arr[@]}".
# SC2148  — Tips depend on shell type — add a shebang.
# SC2164  — Use cd ... || exit  in case cd fails.
# SC2181  — Check exit code directly with `if cmd; then` or `cmd || ...`.
# SC2206  — Quote to prevent word splitting (when assigning array=( $list )).
# SC2207  — Prefer mapfile or read -a to split command output.
# SC2235  — Use {} braces to group commands instead of (parentheses) when no subshell needed.
```

### Disable a rule for one line

```bash
# shellcheck disable=SC2086
echo $intentionally_unquoted
```

## Security

### Always quote, never trust input

```bash
# DON'T: glob expansion of user input
filename="$1"
rm $filename                     # if $1 is "*" you delete cwd

# DON'T: variable injection into eval
cmd="rm $user_input"
eval "$cmd"                      # any shell metacharacters in $user_input → arbitrary code

# DON'T: command substitution with untrusted input
result=$(grep "$search" file)    # if $search is "x; rm -rf /" — well, less bad here, but be careful
```

### `eval` is almost always wrong

```bash
# eval is acceptable in two scenarios:
#   1. You GENERATED the string yourself (e.g., expanding a brace pattern with a variable).
#   2. You shell-quoted every variable with printf %q.
# Otherwise, find a different approach (arrays, namerefs, declare).

# Safe-ish if you must:
arg=$(printf '%q' "$user_input")
eval "echo $arg"
```

### IFS attacks

```bash
# An attacker who controls IFS can break a script:
#   IFS=. /usr/bin/expensive_cmd     ← if your script trusts IFS, weird splits happen

# Defense: set IFS yourself, or unset it:
unset IFS
IFS=$' \t\n'

# Better: use arrays and "${arr[@]}", which doesn't depend on IFS for splitting.
```

### PATH attacks

```bash
# A relative path or a poisoned $PATH can run an attacker's binary:
PATH="$HOME/.local/bin:$PATH"    # if attacker writes "ls" there, your "ls" calls theirs

# Defense in scripts that run with elevated privs:
PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PATH
hash -r                          # forget any cached lookups

# Or call by absolute path:
/usr/bin/ls
```

### `mktemp -d` and predictable names

```bash
# DON'T:
tmp=/tmp/myscript.$$             # PID is guessable; race exists
echo "data" > "$tmp"

# DO:
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT
echo "data" > "$tmpdir/data"
```

### `umask` and sensitive files

```bash
umask 077                        # new files: rw------- ; new dirs: rwx------
secrets=$(mktemp)
echo "$API_TOKEN" > "$secrets"   # only owner can read
chmod 600 "$secrets"             # belt-and-braces
```

### Avoid `trap '' SIGNAL` permanently

```bash
# Ignoring SIGINT/SIGTERM blocks legitimate shutdown:
trap '' INT TERM                 # don't do this without care — operator can't Ctrl-C
# Better: trap and clean up gracefully.
```

## Idioms

### The bash strict-mode preamble

```bash
#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

trap 'echo "ERR line $LINENO: $BASH_COMMAND" >&2' ERR

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PROG=$(basename "$0")

log()  { printf '[%s] %s\n' "$(date +%FT%T)" "$*" >&2; }
err()  { printf '[%s] ERR: %s\n' "$(date +%FT%T)" "$*" >&2; }
die()  { err "$@"; exit 1; }

main() {
    # parse args, do work
    :
}

main "$@"
```

### The trap-EXIT cleanup

```bash
tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

# Also useful: capture exit code in trap to do conditional cleanup
on_exit() {
    local rc=$?
    if (( rc != 0 )); then
        echo "FAILED (rc=$rc)" >&2
        kept_logs="$tmpdir.kept"
        cp -r "$tmpdir" "$kept_logs"
    fi
    rm -rf "$tmpdir"
    exit "$rc"
}
trap on_exit EXIT
```

### The "usage" heredoc

```bash
usage() {
    cat <<'EOF'
Usage: prog [OPTIONS] FILE...

Options:
  -v, --verbose          Verbose output
  -o, --output FILE      Write result to FILE (default: stdout)
  -h, --help             Show this message

Examples:
  prog -v -o out.txt in.txt
  prog --output=out.txt in.txt
EOF
}
```

### Ensure single instance

```bash
LOCK=/var/lock/$(basename "$0").lock
exec 9>"$LOCK"
flock -n 9 || die "another instance is already running"
```

### Ensure required commands

```bash
need() { command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"; }
need jq
need curl
need awk
```

### Confirm-before-destruct

```bash
confirm() {
    local prompt="${1:-Are you sure? [y/N] }"
    local ans
    read -r -p "$prompt" ans
    [[ "${ans,,}" == y* ]]
}

confirm "Delete $dir? [y/N] " || exit 0
rm -rf -- "$dir"
```

## Tips

- Always quote `"$variables"` — word splitting and globbing on unquoted expansions is the #1 source of shipped bash bugs.
- `set -euo pipefail` + `IFS=$'\n\t'` belongs at the top of every script.
- Use `[[ ]]` over `[ ]` in bash — handles spaces, supports `=~` and `&&`/`||`.
- Use `(( ))` for arithmetic — cleaner than `-eq -lt -gt` chains.
- Always `local` your variables in functions — they pollute the global namespace otherwise.
- Always use `read -r` — without it, backslashes get interpreted.
- Always quote `"$@"` when forwarding args — `$@` and `$*` differ subtly.
- Prefer `$(cmd)` over backticks — nests cleanly, no escaping pain.
- Use `mktemp -d` for temp dirs and `trap 'rm -rf "$tmp"' EXIT` for cleanup.
- `mapfile -t lines < file` to slurp a file into an array (bash 4+).
- `${var:-}` is the safe way to deref a possibly-unset var under `set -u`.
- Prefer parameter expansion (`${var//x/y}`, `${var^^}`, `${path##*/}`) over forks to `tr`/`sed`/`basename`.
- Process substitution `<(cmd)` lets while-loops keep their variables — pipes don't.
- `printf` is portable and predictable; `echo -e`/`echo -n` are not.
- Use `flock` for single-instance scripts — `[[ -f lockfile ]]` is racy.
- Test scripts with `bash -n script.sh` (parse-check) and `shellcheck script.sh` before committing.
- Add `set -x` (or run `bash -x script.sh`) to see every command as it runs.
- macOS `/bin/bash` is 3.2 — assume bash 4+ is at Homebrew path or require it explicitly.
- `command -v` is the portable way to test if a command exists — not `which`.
- Heredocs with `<<'EOF'` (quoted delimiter) suppress all expansion — perfect for embedded awk/sed/python.
- Globs are NOT regex. `*.txt`, `[abc]`, `?` — different language, different rules.

## See Also

- shell-scripting
- shellcheck
- zsh
- fish
- nushell
- tmux
- screen
- linux-automation-scripting
- awk
- sed
- grep
- regex
- find
- xargs
- jq
- tar
- gzip
- rsync
- ssh
- ssh-tunneling
- vim
- emacs
- make
- cron
- systemd
- systemd-timers
- signals

## References

- [Bash Reference Manual](https://www.gnu.org/software/bash/manual/) -- complete GNU Bash reference
- [man bash](https://man7.org/linux/man-pages/man1/bash.1.html) -- bash man page
- [BashGuide (Wooledge)](https://mywiki.wooledge.org/BashGuide) -- comprehensive beginner-to-advanced guide
- [Bash FAQ (Wooledge)](https://mywiki.wooledge.org/BashFAQ) -- answers to frequently asked questions
- [Bash Pitfalls (Wooledge)](https://mywiki.wooledge.org/BashPitfalls) -- common mistakes and how to avoid them
- [Greg's Wiki](https://mywiki.wooledge.org/) -- the canonical bash hub (BashGuide, BashFAQ, BashPitfalls live here)
- [Bash Hackers Wiki (archived)](https://web.archive.org/web/2023*/https://wiki.bash-hackers.org/) -- in-depth articles and scripting patterns
- [POSIX Shell Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html) -- portable shell behavior
- [ShellCheck](https://www.shellcheck.net/) -- online shell script linter and analyzer
- [ShellCheck wiki (per-rule details)](https://www.shellcheck.net/wiki/) -- one page per SC#### with examples
- [GNU Readline Library](https://tiswww.case.edu/php/chet/readline/rltop.html) -- line editing and key bindings used by Bash
- [Bash Changes (NEWS)](https://tiswww.case.edu/php/chet/bash/NEWS) -- changelog for every Bash release
- [Advanced Bash-Scripting Guide](https://tldp.org/LDP/abs/html/) -- TLDP, dated but exhaustive
- [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) -- conventions for production-quality scripts
- [Defensive BASH programming](https://kfirlavi.herokuapp.com/blog/2012/11/14/defensive-bash-programming) -- patterns for robust scripts
