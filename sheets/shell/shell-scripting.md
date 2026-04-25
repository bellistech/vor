# Shell Scripting (POSIX sh / portable)

> Writing portable shell scripts that run on dash, busybox sh, ksh, and bash --posix. The lowest-common-denominator that runs everywhere — no bashisms.

## Setup & Targets

POSIX shell is the standardized subset codified in IEEE Std 1003.1 (Issue 7, also called SUSv4 / "POSIX:2017"). The full spec lives at https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html. Real-world implementations:

```bash
dash               # Debian/Ubuntu /bin/sh — small, fast, strictly POSIX. The reference.
busybox sh         # Alpine /bin/sh — based on ash, slightly different from dash.
ksh / pdksh / mksh # Korn shell. POSIX superset.
bash               # GNU bash. POSIX superset; run with `bash --posix` to disable extensions.
zsh                # Z shell. POSIX-ish; run with `emulate sh` to limit to POSIX.
yash               # POSIX-strict, very pedantic. Good for portability checking.
ash                # FreeBSD /bin/sh. The dash ancestor.
```

```bash
#!/bin/sh                        # canonical portable shebang
#!/usr/bin/env sh                # if /bin/sh is not on PATH (rare; less standard)
```

```bash
# Verify what /bin/sh resolves to on a host:
ls -l /bin/sh                    # usually symlink to dash, bash, busybox, or ash
readlink -f /bin/sh

# Find the live shell from inside a script:
ps -p $$ -o comm=
```

```bash
# Smoke-check a script under each shell:
dash       ./script.sh
busybox sh ./script.sh
bash --posix ./script.sh
ksh        ./script.sh
yash       ./script.sh

# checkbashisms (Debian package devscripts) catches most bash-only constructs:
checkbashisms script.sh
```

The reason dash exists as `/bin/sh` on Debian/Ubuntu: faster boot. System init scripts run faster under dash than bash. Side effect: any "/bin/sh" script that uses bash features will silently break on Debian-family systems.

## Strict Mode

POSIX gives you `set -e` and `set -u`. POSIX gives you `set -o pipefail` only since SUSv4 2017 (Issue 7 TC2) — older dashes lack it. The "bash strict mode" `IFS=$'\n\t'` (`$'…'` ANSI-C quoting) is **bash-only**; the POSIX equivalent uses `printf` to inject literal characters.

```bash
#!/bin/sh
set -eu                      # POSIX-portable: errexit + nounset
set -o pipefail 2>/dev/null || true   # opt-in; ignore on shells that lack it

# POSIX way to set IFS to newline+tab (no $'\n\t'):
IFS=$(printf '\n\t')         # value is literal LF then TAB
```

`set -e` quirks (very real, very portable):

```bash
# DOES NOT trigger -e: failures inside if/while/until conditions
if grep nope file; then echo found; fi   # grep "fails" but script continues

# DOES NOT trigger -e: left side of && or ||
false && echo never                       # ok
false || echo recovered                   # ok

# In POSIX, only the LAST stage of a pipeline counts for $? (and -e):
false | true                              # exit status is 0 (true)
# enable pipefail (where supported) to fix.

# subshell exits don't always propagate predictably across shells; -e in subshells
# inherits but POSIX says behavior of inherited -e was historically variable.
```

The defensive style — explicit error checking — is more portable than relying on `-e`:

```bash
cmd || { echo "cmd failed" >&2; exit 1; }

# helper "die" function (covered later)
die() { printf '%s\n' "$*" >&2; exit 1; }
cmd || die "cmd failed"
```

## Variables

```bash
name=value                   # NO spaces around =
name = value                 # WRONG — runs `name` as a command with args = value
name= value                  # WRONG — runs `value` with name="" in its environment

# read-only
readonly PI=3.14159
readonly TAU                 # mark already-set var read-only
PI=3                         # error: read-only variable

# delete
unset name                   # remove variable
unset -f myfunc              # remove function

# export to environment (children see it)
export PATH="$HOME/bin:$PATH"
EDITOR=vi; export EDITOR     # split form (some old shells preferred this)

# one-shot env: only this command sees FOO
FOO=bar cmd                  # POSIX-portable scope-to-command idiom
```

POSIX guarantees `${VAR}` (braces) and `$VAR`. Always brace when concatenating with adjacent letters/digits/underscores:

```bash
file=log
echo "$file_$$"              # WRONG — looks up var "file_" (empty) then $$
echo "${file}_$$"            # right
```

`local` is **not** POSIX. Most real shells (dash, busybox ash, bash, ksh) support it inside functions, but yash and posh do not. If you need strict POSIX, simulate locals with subshell scope or naming discipline:

```bash
# Not strict POSIX, but works on dash/ash/bash/ksh:
myfunc() { local x=1; ...; }

# Strict POSIX alternative — run the function body in a subshell:
myfunc() ( x=1; do_stuff )    # parens, not braces, isolate state
```

## Quoting

Three quote levels in POSIX:

```bash
'literal'                    # single quotes — NOTHING is interpreted, not even \
"interpolated"               # double quotes — $var, $(cmd), `cmd`, \, \" expand
\$                           # backslash escapes the next character (outside single quotes)

# can't put a single quote inside single quotes; close, escape, reopen:
echo 'it'\''s broken'        # prints: it's broken
```

The unquoted-variable trap:

```bash
file="my file.txt"
rm $file                     # WRONG — runs: rm my file.txt  (two args)
rm "$file"                   # right

for f in $files; do ...      # WRONG — word-split + glob $files
# (there is no portable safe way to split a string into a list without IFS gymnastics)
```

When in doubt, quote. The exceptions where quoting is wrong:

```bash
# 1) on the right side of = in `case`, glob patterns must be UNQUOTED:
case "$x" in
    *.txt) ;;                # works — pattern is unquoted
    "*.txt") ;;              # WRONG — matches the literal three-char string *.txt
esac

# 2) you actually want word splitting (rare; use an array or "set --" instead).
```

## Parameter Expansion

POSIX guarantees a fixed set:

```bash
${var}            # value, or empty
${var:-default}   # value, or default if unset/empty (does not assign)
${var-default}    # value, or default if unset (empty stays empty)
${var:=default}   # value; assign default if unset/empty (returns the value)
${var=default}    # assign default if unset
${var:?error}     # value; print "error" to stderr and exit if unset/empty
${var?error}      # value; print "error" if unset
${var:+alt}       # alt if set and non-empty; else empty
${var+alt}        # alt if set; else empty

${#var}           # length in bytes (NOT chars in POSIX — implementation-defined for multibyte)

# Pattern stripping (uses GLOB patterns, not regex):
${var#pat}        # remove SHORTEST prefix matching pat
${var##pat}       # remove LONGEST prefix matching pat
${var%pat}        # remove SHORTEST suffix matching pat
${var%%pat}       # remove LONGEST suffix matching pat
```

Examples:

```bash
path=/usr/local/bin/foo.sh
echo "${path##*/}"           # foo.sh           (basename)
echo "${path%/*}"            # /usr/local/bin   (dirname)
echo "${path%.sh}.bak"       # /usr/local/bin/foo.bak

name=README.md
echo "${name%.*}"            # README
echo "${name##*.}"           # md
```

**Bashisms NOT in POSIX** — flag and avoid:

```bash
${var/pat/rep}         # bash only — substitution
${var//pat/rep}        # bash only — global substitution
${var^^}               # bash only — uppercase (or zsh ${(U)var})
${var,,}               # bash only — lowercase
${var:offset:len}      # bash/ksh — substring; POSIX has NO substring-by-index
${!var}                # bash — indirect expansion
${!prefix*}            # bash — variable-name listing
${var@Q}               # bash 4.4+ — quoted form
${var@A}               # bash 4.4+ — assignment form
```

POSIX substring/replace alternatives:

```bash
# replace first 'foo' with 'bar' — use sed:
new=$(printf '%s\n' "$var" | sed 's/foo/bar/')

# global replace — sed with g:
new=$(printf '%s\n' "$var" | sed 's/foo/bar/g')

# substring "first 5 chars":
prefix=$(printf '%s' "$var" | cut -c1-5)

# uppercase / lowercase:
upper=$(printf '%s' "$var" | tr '[:lower:]' '[:upper:]')
lower=$(printf '%s' "$var" | tr '[:upper:]' '[:lower:]')
```

## Special Parameters

```bash
$0     # script name (or shell name when interactive)
$1..$9 # positional args. Beyond 9, use ${10}, ${11}, ...
$#     # number of positional args
"$@"   # all args, each as a SEPARATE quoted word — almost always what you want
"$*"   # all args, joined into ONE word using first char of IFS (rarely what you want)
$?     # exit status of last command
$$     # PID of current shell
$!     # PID of last background command
$-     # current shell flags (e.g. himBH)
$_     # last argument of previous command (informal — POSIX in some contexts only)
```

The `"$@"` vs `$@` vs `"$*"` distinction:

```bash
set -- 'one two' three
for a in $@;   do echo "<$a>"; done   # 3 args: one / two / three  (BAD: split inside "one two")
for a in "$@"; do echo "<$a>"; done   # 2 args: one two / three     (correct)
for a in "$*"; do echo "<$a>"; done   # 1 arg : "one two three"     (joined)
```

`set --` rewrites positional parameters:

```bash
set -- a b c                 # $1=a $2=b $3=c $#=3
set -- "$@" extra            # append "extra"
shift                        # drop $1; everything moves down one
shift 2                      # drop $1 and $2
```

## Numbers and Arithmetic

POSIX has `$(( expression ))`. There is **no** standalone `(( ))` (that's bash/ksh) and **no** `let` (also bash/ksh).

```bash
n=$(( 1 + 2 ))               # 3
n=$(( n * 4 + 1 ))           # vars don't need $ inside (( ))
n=$(( 0xff ))                # 255 — hex literals OK
n=$(( 0755 ))                # 493 — octal (leading 0)
n=$(( 2#1010 ))              # bash extension — base#digits NOT POSIX

# operators: + - * / % (mod), ** (power — NOT POSIX! ksh/bash only)
# bitwise: & | ^ ~ << >>
# logical: ! && ||
# comparison: == != < <= > >= (return 0 / 1)
# ternary: a ? b : c
# assignment: = += -= *= /= %= &= |= ^= <<= >>=
```

Floating point is **not** in POSIX `$(( ))`. Use `bc` or `awk`:

```bash
result=$(printf '%s\n' "scale=4; 1/3" | bc)
result=$(awk 'BEGIN{printf "%.4f\n", 1/3}')
```

`expr` is the legacy fallback for shells where `$(( ))` is broken (none of the modern ones). Used historically:

```bash
n=`expr 1 + 2`               # spaces required around operators
n=`expr "$x" \* 2`           # * must be escaped (glob)
```

Avoid `expr` in new code. `$(( ))` is universal in real shells.

## String Operations — Pure POSIX

```bash
# length (bytes):
n=${#string}

# concatenation:
joined="$a$b"
joined="${a}_${b}"           # brace when adjacent to letters/digits

# substring — POSIX has NO direct ${var:offset:len}; use cut or printf:
sub=$(printf '%s' "$s" | cut -c2-5)        # chars 2 through 5
sub=$(printf '%s' "$s" | cut -c-5)         # first 5 chars
sub=$(printf '%s' "$s" | cut -c5-)         # from char 5 to end

# contains (no =~ in POSIX; use case):
case "$s" in
    *foo*) echo "has foo" ;;
    *)     echo "no foo"  ;;
esac

# starts with / ends with:
case "$s" in
    /*) echo "absolute path" ;;
    *.sh) echo "shell script" ;;
esac

# trim leading/trailing whitespace via sed:
trimmed=$(printf '%s' "$s" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')

# trim leading/trailing whitespace via parameter expansion (POSIX!):
trim() {
    s=$1
    while :; do case "$s" in
        " "*|"	"*) s=${s#?} ;;
        *) break ;;
    esac done
    while :; do case "$s" in
        *" "|*"	") s=${s%?} ;;
        *) break ;;
    esac done
    printf '%s' "$s"
}

# uppercase / lowercase (NO ${var^^} in POSIX):
upper=$(printf '%s' "$s" | tr 'a-z' 'A-Z')
lower=$(printf '%s' "$s" | tr 'A-Z' 'a-z')

# split on delimiter (no arrays in POSIX — use IFS + set --):
old_IFS=$IFS
IFS=:
set -- $PATH                 # $1, $2, ... are PATH components
IFS=$old_IFS
for dir; do echo "$dir"; done
```

## Conditional Tests

`test EXPR` and `[ EXPR ]` are **the same builtin**. The closing `]` is a literal argument and must be present.

File tests:

```bash
[ -e path ]    # exists (any type)
[ -f path ]    # regular file
[ -d path ]    # directory
[ -L path ]    # symbolic link (POSIX since 2001)
[ -h path ]    # symbolic link (legacy alias of -L)
[ -r path ]    # readable by current process
[ -w path ]    # writable
[ -x path ]    # executable
[ -s path ]    # exists and size > 0
[ -p path ]    # FIFO / named pipe
[ -S path ]    # socket
[ -b path ]    # block device
[ -c path ]    # character device
[ -t fd ]      # fd is a terminal (stdin = 0, stdout = 1, stderr = 2)

# comparing two paths:
[ a -nt b ]    # a newer than b (POSIX since 2001)
[ a -ot b ]    # a older than b
[ a -ef b ]    # same inode (hard-linked)
```

String tests:

```bash
[ -z "$s" ]    # empty
[ -n "$s" ]    # non-empty
[ "$a" = "$b" ]    # equal
[ "$a" != "$b" ]   # not equal
# == is bash; POSIX uses single = for string equality.
# < and > inside [ ] are NOT portable — use sort or expr.
```

Numeric tests (use these for numbers, NOT `=`):

```bash
[ "$a" -eq "$b" ]    # equal
[ "$a" -ne "$b" ]    # not equal
[ "$a" -lt "$b" ]    # less than
[ "$a" -le "$b" ]    # ≤
[ "$a" -gt "$b" ]    # greater than
[ "$a" -ge "$b" ]    # ≥
```

Boolean operators:

```bash
[ -f a ] && [ -r a ]            # AND (preferred — short-circuits, no quoting traps)
[ -f a ] || [ -f b ]            # OR

[ -f a -a -r a ]                # AND inside brackets — works but DEPRECATED in POSIX
[ -f a -o -f b ]                # OR — also deprecated; can be ambiguous

# Why deprecated: `[ "$a" = "$b" -a "$c" = "$d" ]` becomes ambiguous when
# any of a/b/c/d is a unary operator like -f or "(". Always use && / ||.

! [ -f file ]                   # negation
[ ! -f file ]                   # also fine
```

Always quote variables in `[ ]` — an empty variable becomes a missing argument:

```bash
x=
[ $x = "" ] && echo empty    # ERROR: [: =: unary operator expected
[ "$x" = "" ] && echo empty  # works
[ -z "$x" ] && echo empty    # cleaner
```

`[[ ]]` is **not POSIX** — bash/ksh/zsh only. It does NOT word-split or glob, supports `=~`, `<`, `>`, and pattern `==`. Convert away from it for portability:

```bash
[[ $x == foo* ]]             # bash
case "$x" in foo*) ;; esac   # POSIX equivalent
```

## Control Flow

```bash
if cmd; then
    statements
elif cmd2; then
    statements
else
    statements
fi

# the test command works because it returns 0 / 1 like any program:
if [ -f /etc/hosts ]; then ...; fi

# inline:
[ -f /etc/hosts ] && echo present || echo missing
# WARNING: this idiom is BUGGY if the `&&` branch fails — the `||` runs anyway.
# Use proper if/then if either side might fail.
```

`case` for pattern matching:

```bash
case "$value" in
    yes|y|Y)  do_yes ;;
    no|n|N)   do_no  ;;
    "")       do_default ;;
    *.txt)    handle_text "$value" ;;
    [0-9]*)   handle_numeric "$value" ;;
    *)        echo "unknown: $value" >&2 ;;
esac
```

The `;;` terminator is required. Bash adds `;&` (fall through to next without testing) and `;;&` (fall through and test) — both **non-POSIX**.

Patterns inside `case` must be **unquoted** to be patterns; quoting makes them literal:

```bash
case "$x" in
    *.txt) ;;                # glob match
    "*.txt") ;;              # literal three-char string match (probably a bug)
esac
```

## Loops

```bash
# C-style for is NOT POSIX. Use this instead:
i=1
while [ "$i" -le 10 ]; do
    echo "$i"
    i=$(( i + 1 ))
done

# for-each over a list:
for f in *.log; do
    echo "$f"
done

# for over arguments — `in` clause omitted means "in $@":
for arg do
    echo "<$arg>"
done

# while:
while read -r line; do
    process "$line"
done < input.txt

# until (loop while command FAILS):
until ping -c1 host >/dev/null 2>&1; do
    sleep 1
done

# break / continue:
for f in *; do
    [ -f "$f" ] || continue
    [ -L "$f" ] && break
    process "$f"
done

# break N / continue N — break out of N nested loops (POSIX):
for i in 1 2 3; do
    for j in a b c; do
        [ "$i$j" = "2b" ] && break 2
        echo "$i$j"
    done
done

# infinite loop:
while :; do ...; done        # `:` is the no-op builtin, faster than `true`
while true; do ...; done     # also fine, slightly slower
```

C-style `for ((i=0;i<10;i++))` is bash/ksh only. Brace ranges `{1..10}` are bash/zsh only. Portable counted loops use `seq` (not POSIX strictly, but ubiquitous) or arithmetic + `while`:

```bash
# WIDELY available, NOT in strict POSIX:
for i in $(seq 1 10); do echo "$i"; done

# strictly portable counted loop:
i=1
while [ "$i" -le 10 ]; do
    echo "$i"
    i=$(( i + 1 ))
done
```

## Functions

```bash
# POSIX function definition:
greet() {
    echo "hello, $1"
}

# `function greet { ... }` is ksh/bash, NOT POSIX.

greet world                  # call: arg passing identical to scripts
$#  $1  $2 ...               # inside the function refer to function args, not script args
$0                           # still the script name

# return value = exit code in 0..255 (NOT a string!):
is_root() {
    [ "$(id -u)" = "0" ]     # last command's exit status becomes the function's
}
if is_root; then echo "yes"; fi

return                       # use last command's status
return 0
return 42                    # propagate explicit code
```

To "return" a string, write to stdout and capture:

```bash
my_dir() { echo "/tmp/foo"; }
d=$(my_dir)
```

Local variables: see the Variables section. POSIX has no `local`, but every modern shell does.

Function arguments work just like script arguments:

```bash
process() {
    [ $# -ge 1 ] || { echo "process: need an arg" >&2; return 2; }
    for x do echo "got: $x"; done
}
process a 'b c' d
```

## Subshells vs Grouping

```bash
# subshell — runs in a child process; vars don't leak out:
( cd /tmp; ls )              # cwd outside is unchanged

# group — same shell; vars DO leak; needs trailing semicolon and spaces:
{ cd /tmp; ls; }             # cwd outside IS changed
                             # NOTE: { needs a space after it; } needs a ; before it
```

Common subshell uses:

```bash
# isolate `cd`:
( cd "$dir" && do_work )

# isolate IFS / set flags:
( IFS=:; set -f; echo $PATH ) # changes don't leak

# command substitution always creates a subshell:
out=$(cd /tmp; pwd)          # /tmp; outer cwd unchanged
```

Vars set in a subshell or in a command-substitution **do not** propagate out:

```bash
x=1
(x=2)
echo "$x"                    # 1

x=$(echo 2; x=99)            # x set to "2", inner x=99 is invisible
echo "$x"                    # 2

# similarly, a pipeline puts each side in a subshell on most POSIX shells:
echo a | read -r v           # v is set in the subshell, lost outside in dash/ash/bash<5.0
echo "$v"                    # empty in dash; bash 4.2+ with `lastpipe` differs
# fix: read from a here-doc / here-string / file:
read -r v <<EOF
a
EOF
```

## Pipes & Pipelines

```bash
cmd1 | cmd2 | cmd3           # stdout of N feeds stdin of N+1; runs concurrently

# In POSIX, $? is the exit status of the LAST stage:
false | true                 # $? = 0
true  | false                # $? = 1

# `pipefail` (where supported) makes $? = first non-zero, or 0 if all zero:
set -o pipefail              # POSIX:2017; bash, dash 0.5.10+, mksh, busybox ash
false | true                 # $? = 1 with pipefail set

# bash-only $PIPESTATUS array gives status of every stage; not POSIX:
true | false | true
echo "${PIPESTATUS[@]}"      # 0 1 0   (bash only)
```

A pipeline always creates subshells — see the read-loop gotcha above.

## Redirection

Order matters. Each redirection is processed left to right, so `2>&1 >file` and `>file 2>&1` differ:

```bash
cmd > file 2>&1              # both stdout and stderr to file (CORRECT)
cmd 2>&1 > file              # stderr → original stdout (terminal); stdout → file. WRONG.
```

```bash
cmd > file                   # stdout → file (truncate)
cmd >> file                  # stdout → file (append)
cmd < file                   # file → stdin
cmd 2> file                  # stderr → file
cmd 2>> file                 # stderr → file (append)
cmd > out 2> err             # stdout → out, stderr → err
cmd > /dev/null 2>&1         # silence both
cmd >file 2>&1               # both → file
cmd <> file                  # open file for read+write on stdin
cmd >&-                      # close stdout
cmd 2>&-                     # close stderr
cmd 0<&-                     # close stdin

# bashism alert:
cmd &> file                  # bash/zsh shorthand — NOT POSIX. Use: cmd > file 2>&1
cmd >& file                  # csh-style — NOT POSIX
cmd &>> file                 # bash 4+ — NOT POSIX
```

Here-documents:

```bash
cat <<EOF
$USER's home is $HOME
EOF                          # variables expanded; first column EOF must close

cat <<'EOF'
literal $USER and $HOME
EOF                          # quote any part of the delimiter to disable expansion

cat <<-EOF
	indented body — leading TABS are stripped (only tabs, not spaces!)
	another line
	EOF                  # the dash form lets you indent code; close with TABS too
```

```bash
# capture a here-doc to a variable:
msg=$(cat <<EOF
line 1
line 2
EOF
)

# pipe a here-doc into a command:
sort <<'EOF'
banana
apple
cherry
EOF
```

Here-strings `<<<` are **bash/ksh/zsh** — not POSIX:

```bash
grep foo <<<"$var"           # bashism
# POSIX:
printf '%s\n' "$var" | grep foo
```

File descriptor redirection patterns:

```bash
exec 3>logfile               # open fd 3 for write, attached to logfile
echo "hello" >&3             # write to fd 3
exec 3>&-                    # close fd 3

exec 4<infile                # open fd 4 for read
read -r line <&4
exec 4<&-

exec >logfile 2>&1           # redirect ALL further output of this script to logfile
```

## Process Substitution

`<(cmd)` and `>(cmd)` are **bash/ksh/zsh only** — not POSIX.

```bash
# bash/ksh/zsh:
diff <(sort a) <(sort b)

# POSIX equivalent — temp files:
tmpa=$(mktemp) tmpb=$(mktemp)
trap 'rm -f "$tmpa" "$tmpb"' EXIT
sort a > "$tmpa"
sort b > "$tmpb"
diff "$tmpa" "$tmpb"

# alternative: named pipes (FIFOs) for streaming:
fifo=$(mktemp -u)
mkfifo "$fifo"
trap 'rm -f "$fifo"' EXIT
producer > "$fifo" &
consumer < "$fifo"
wait
```

## Command Substitution

```bash
out=$(cmd)                   # POSIX, nestable, preferred
out=`cmd`                    # legacy backticks; harder to nest, hard to escape

# nesting:
result=$(echo $(date))       # easy
result=`echo \`date\``       # ugly with escaping

# trailing newlines are STRIPPED from $() output:
cwd=$(pwd)                   # newline removed automatically — what you want

# but only TRAILING — internal newlines survive:
text=$(printf 'a\nb\nc\n')
echo "$text"                 # a b c on three lines
```

## Pattern Matching with case

POSIX `case` patterns are **glob patterns** (not regex). Wildcards: `*`, `?`, `[abc]`, `[a-z]`, `[!abc]` (negation; `[^abc]` is NOT POSIX, though widely supported).

```bash
case "$s" in
    *)        ;;             # always matches — default
    ?)        ;;             # any single character
    [abc])    ;;             # one of a, b, c
    [!abc])   ;;             # NOT one of a, b, c (POSIX)
    [a-z]*)   ;;             # starts with lowercase
    *.txt|*.log) ;;          # OR with |
    "exact string") ;;       # quoting makes it literal
esac

# fall through (NON-POSIX bash extensions):
;&     # fall to next pattern's body without testing — bash only
;;&    # fall to next pattern, test it — bash only
```

Extended globs `?(pat)`, `*(pat)`, `+(pat)`, `@(pat)`, `!(pat)` are **bash extglob / ksh** — not POSIX.

## Globbing

```bash
*           # any sequence (including empty), but does NOT match leading dot
?           # any one character
[abc]       # one of those chars
[a-z]       # range (locale-dependent! in C locale = a..z)
[!abc]      # not one of those (POSIX)
[^abc]      # not one of those (NOT POSIX, but works almost everywhere)

# leading-dot files: NOT matched by * unless you set the option (`shopt -s dotglob`
# in bash; `setopt globdots` in zsh; in dash this is impossible — match explicitly):
ls .*       # all dotfiles (and . and ..)
ls .[!.]*   # dotfiles that are NOT . or ..
```

Disable globbing with `set -f` (a.k.a. `set -o noglob`):

```bash
set -f
echo *.txt              # prints literal *.txt
set +f                  # re-enable
```

POSIX behavior when a glob matches **nothing** is implementation-defined but in practice: the pattern stays literal:

```bash
set -- /no/such/dir/*
echo "$1"               # /no/such/dir/*  (pattern unchanged)

# defensive — check the file exists:
for f in *.txt; do
    [ -e "$f" ] || continue
    process "$f"
done

# bash has nullglob/failglob via shopt; POSIX has neither.
```

## Reading Input

```bash
# read one line from stdin into one variable:
read -r line                 # -r prevents backslash interpretation — ALWAYS use -r

# read multiple words into separate variables (split on $IFS):
read -r first rest           # rest gets everything after the first IFS-separated word

# silent prompt (e.g. password) — POSIX has NO `read -s`. Use stty:
stty -echo
read -r password
stty echo
echo                         # newline since echo was off

# prompt — POSIX has NO `read -p` either. Print first:
printf 'name? '
read -r name
```

Read a file line-by-line — the canonical idiom:

```bash
while IFS= read -r line; do
    process "$line"
done < input.txt
```

Why `IFS=` and `-r`?

- `IFS=` (empty) prevents leading/trailing whitespace from being trimmed.
- `-r` prevents backslash from being treated as an escape.

Without these, ` foo \tbar ` becomes `foo bar` and you lose information.

```bash
# read from a command, NOT a file (subshell trap!):
echo a | while read -r v; do echo "$v"; done   # works
echo a | { read -r v; echo "$v"; }             # works
echo a | read -r v && echo "$v"                # FAILS in dash/ash/bash<5.0 — v is lost
                                                # because read runs in a subshell
# Workarounds: use $(cmd) / process substitution / here-doc instead.

# read until EOF — handle the case where the last line has no trailing newline:
while IFS= read -r line || [ -n "$line" ]; do
    process "$line"
done < file
```

## Functions vs Aliases

POSIX defines both, but **aliases are expanded only by interactive shells** by default. In a script, an alias defined and used in the same script may not expand at all (depends on shell and `expand_aliases`). Use functions in scripts. Always.

```bash
# DON'T do this in a script:
alias ll='ls -l'
ll                  # may or may not work depending on shell

# DO use a function:
ll() { ls -l "$@"; }
ll
```

## Argument Parsing — getopts

POSIX builtin for short options. No long options.

```bash
verbose=0
infile=
while getopts ':hvi:' opt; do
    case "$opt" in
        h) printf 'usage: %s [-v] [-i FILE]\n' "$0"; exit 0 ;;
        v) verbose=$(( verbose + 1 )) ;;
        i) infile=$OPTARG ;;
        :) printf 'option -%s requires an argument\n' "$OPTARG" >&2; exit 2 ;;
        \?) printf 'unknown option: -%s\n' "$OPTARG" >&2; exit 2 ;;
    esac
done
shift $(( OPTIND - 1 ))

# now $1, $2, ... are the non-option args
echo "remaining: $#"
for arg do echo "<$arg>"; done
```

Notes:
- The leading `:` in `':hvi:'` enables silent error mode — you handle `:` (missing arg) and `\?` (unknown opt) yourself instead of getopts printing a default message.
- An option followed by `:` takes an argument (e.g. `i:`).
- `OPTARG` holds the argument; `OPTIND` is the index of the next argument to process.
- `getopts` is **not** the external `getopt(1)` — that one varies wildly between BSD and GNU.

Limitation: no long options. For `--flag` style, parse manually.

## Argument Parsing — Manual

The canonical long-option loop:

```bash
verbose=0
infile=
output=

while [ $# -gt 0 ]; do
    case "$1" in
        -h|--help)
            printf 'usage: %s [--verbose] [--input FILE] [--output FILE]\n' "$0"
            exit 0
            ;;
        -v|--verbose)
            verbose=$(( verbose + 1 ))
            ;;
        -i|--input)
            [ $# -ge 2 ] || { echo "--input requires an argument" >&2; exit 2; }
            infile=$2
            shift
            ;;
        --input=*)
            infile=${1#--input=}
            ;;
        -o|--output)
            [ $# -ge 2 ] || { echo "--output requires an argument" >&2; exit 2; }
            output=$2
            shift
            ;;
        --output=*)
            output=${1#--output=}
            ;;
        --)                              # end-of-options marker
            shift
            break
            ;;
        -*)
            echo "unknown option: $1" >&2
            exit 2
            ;;
        *)
            break                        # first non-option — the rest is positional
            ;;
    esac
    shift
done

# $@ now holds positional args
```

The `--` convention lets the user end option parsing — useful when filenames start with `-`.

## Reading Lines from File

The right way:

```bash
while IFS= read -r line; do
    printf 'got: %s\n' "$line"
done < file.txt
```

Common variants:

```bash
# read from a pipe — beware the subshell trap:
some_cmd | while IFS= read -r line; do echo "$line"; done

# read CSV-ish — use IFS to split:
while IFS=, read -r name age city; do
    printf 'name=%s age=%s city=%s\n' "$name" "$age" "$city"
done < data.csv

# read with line counter:
n=0
while IFS= read -r line; do
    n=$(( n + 1 ))
    printf '%d: %s\n' "$n" "$line"
done < file.txt

# handle missing trailing newline on last line:
while IFS= read -r line || [ -n "$line" ]; do
    process "$line"
done < file.txt
```

## Iterating Files Safely

The classic mistake — iterating output of `ls`:

```bash
# BROKEN:
for f in $(ls *.txt); do            # word-splits on spaces, glob-expands AGAIN
    process "$f"
done
```

Right ways:

```bash
# 1) directly use the glob (handles spaces):
for f in *.txt; do
    [ -e "$f" ] || continue          # handle empty match
    process "$f"
done

# 2) find with -exec (POSIX):
find . -name '*.txt' -type f -exec process {} \;

# 3) find with -exec + (faster — batches like xargs):
find . -name '*.txt' -type f -exec process {} +

# 4) find -print | xargs (NOT safe for filenames with spaces/newlines/quotes):
find . -name '*.txt' -print | xargs process     # BROKEN on weird filenames

# 5) find -print0 | xargs -0 (GNU/BSD extension; not strict POSIX):
find . -name '*.txt' -print0 | xargs -0 process

# 6) find with a while loop (POSIX, handles ALL filenames):
find . -name '*.txt' -type f | while IFS= read -r f; do
    process "$f"                     # safe except for newline-in-filename
done

# 7) NUL-delimited safe loop (requires GNU find / BSD find):
find . -name '*.txt' -type f -print0 | (
    while IFS= read -r -d '' f; do   # bash -d '' is non-POSIX; in POSIX use xargs
        process "$f"
    done
)
```

Strictly POSIX-and-correct under any filename:

```bash
find . -name '*.txt' -type f -exec sh -c '
    for f do
        process "$f"
    done
' sh {} +
```

The `sh -c '...' sh {} +` trick: `find` passes filenames as args to a tiny inline shell, which iterates them safely.

## Error Handling — set -e

`set -e` (errexit) tells the shell to exit on any unchecked failure. Quirks:

```bash
set -e
false                       # script exits immediately

# but these DO NOT exit:
if false; then ...; fi      # condition context — exempt
while false; do ...; done   # condition context — exempt
false || true               # right side of || — exempt
false && cmd                # left side of && — counts as the condition
! true                      # negation — exits with success ($?=0)
false | true                # piped — only LAST stage matters in POSIX

# inside a function called as part of a condition, -e is DISABLED in old bash:
set -e
f() { false; echo "still here"; }
if f; then echo ok; fi      # in old bash: prints "still here" + "ok"
```

Older systems (Solaris /bin/sh, classic AIX, very old BSD) had buggy `set -e`. The defensive style is more portable:

```bash
cmd || exit 1
cmd || die "cmd failed"

# group multiple recoveries:
{ cmd1 && cmd2 && cmd3; } || die "pipeline failed"
```

The "official" middle ground used by many scripts:

```bash
#!/bin/sh
set -eu
: "${PIPEFAIL_OK=}"
( set -o pipefail ) 2>/dev/null && set -o pipefail
```

## Error Handling — trap

`trap` registers cleanup or signal handlers. Use POSIX signal names (no `SIG` prefix in the shell builtin), not numbers.

```bash
trap 'cleanup' EXIT                        # runs on any normal or `exit` exit
trap 'echo interrupted; exit 130' INT      # Ctrl-C
trap 'echo terminating; exit 143' TERM     # default `kill`
trap 'echo hangup; exit 129' HUP

# multiple signals share a handler:
trap 'cleanup; exit 1' INT TERM HUP

# clear a handler:
trap - INT

# ignore a signal entirely:
trap '' INT

# typical cleanup pattern:
tmpdir=$(mktemp -d) || exit 1
trap 'rm -rf "$tmpdir"' EXIT INT TERM HUP

# inspect current traps:
trap                                       # prints all set traps
```

POSIX standard signal names (always available): `HUP INT QUIT ILL TRAP ABRT FPE KILL USR1 SEGV USR2 PIPE ALRM TERM CHLD CONT STOP TSTP TTIN TTOU URG`. The pseudo-signal `EXIT` (or `0`) fires on shell exit.

`KILL` (9) and `STOP` (17/19) cannot be trapped.

```bash
# inside an EXIT trap, $? is the exit status that caused the exit:
on_exit() {
    rc=$?
    [ "$rc" -eq 0 ] || echo "failed with $rc" >&2
    rm -rf "$tmpdir"
    exit $rc
}
trap on_exit EXIT
```

`ERR` is **bash/ksh/zsh** — not POSIX.

## Logging Idiom

```bash
log()  { printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2; }
warn() { printf 'WARN: %s\n' "$*" >&2; }
err()  { printf 'ERR:  %s\n' "$*" >&2; }
die()  { printf 'FATAL: %s\n' "$*" >&2; exit 1; }
debug(){ [ "${DEBUG:-0}" = "1" ] && printf 'DBG: %s\n' "$*" >&2; }

# stdout = data, stderr = messages. Always.
result=$(do_work)            # captures only stdout
log "got: $result"

# tee log to a file AND keep terminal output:
exec > >(tee -a "$logfile") 2>&1     # bash only — process substitution
# POSIX way — start the script, log everything:
exec >> "$logfile" 2>&1              # all subsequent output goes to logfile
```

Saving original stdout/stderr to restore later:

```bash
exec 3>&1 4>&2               # save fds
exec >logfile 2>&1           # redirect everything
do_work
exec 1>&3 2>&4               # restore
exec 3>&- 4>&-               # close the saves
```

## Locking

POSIX has no `flock`. The portable atomic primitive is `mkdir`:

```bash
lockdir=/tmp/myscript.lock
if mkdir "$lockdir" 2>/dev/null; then
    trap 'rm -rf "$lockdir"' EXIT INT TERM HUP
    do_critical_section
else
    echo "already running (lock $lockdir)" >&2
    exit 1
fi
```

`mkdir` is atomic on POSIX filesystems: either it creates the directory or it fails. Don't use `[ -e file ] && touch file` — that's a TOCTOU race.

A PID-tracked lock for staleness detection:

```bash
lockdir=/tmp/myscript.lock
lockpid=$lockdir/pid

acquire_lock() {
    if mkdir "$lockdir" 2>/dev/null; then
        echo $$ > "$lockpid"
        trap 'rm -rf "$lockdir"' EXIT INT TERM HUP
        return 0
    fi
    # exists — check staleness
    if [ -f "$lockpid" ] && pid=$(cat "$lockpid") && kill -0 "$pid" 2>/dev/null; then
        return 1                 # holder alive
    fi
    rm -rf "$lockdir"            # stale; retry once
    acquire_lock
}

acquire_lock || die "could not acquire lock"
```

`flock(1)` (Linux util-linux) and `lockf(1)` (BSD) are non-POSIX but useful when you control the platform.

## Date/Time — Portable

```bash
date '+%Y-%m-%d'              # ISO date
date '+%Y-%m-%dT%H:%M:%S%z'   # ISO 8601-ish with timezone
date -u '+%Y-%m-%dT%H:%M:%SZ' # UTC, RFC 3339
date '+%s'                    # epoch seconds (POSIX, but newer systems only)

# parse a date — INCOMPATIBLE between GNU and BSD:
date -d '2024-01-15'                       # GNU only
date -j -f '%Y-%m-%d' '2024-01-15' '+%s'   # BSD/macOS
```

GNU/BSD differences:

```bash
# offset from now:
date -d '+1 day'                           # GNU
date -v +1d                                # BSD/macOS

# from epoch:
date -d "@$epoch"                          # GNU
date -r "$epoch"                           # BSD/macOS
```

For portable epoch arithmetic, use `awk`:

```bash
now=$(date '+%s')
hour_ago=$(awk -v n="$now" 'BEGIN{print n-3600}')
formatted=$(awk -v t="$hour_ago" 'BEGIN{print strftime("%Y-%m-%dT%H:%M:%SZ", t, 1)}')
# strftime in awk: GNU awk has it; mawk does NOT (use perl as another fallback).
```

For real cross-platform dates, `perl` (universally installed alongside POSIX systems) is the most portable parser:

```bash
ts=$(perl -MPOSIX=strftime -e 'print strftime("%Y-%m-%dT%H:%M:%SZ", gmtime(time-3600))')
```

## Subprocess

Sequencing operators — exit codes drive flow:

```bash
cmd1 ; cmd2          # always run cmd2 (regardless of cmd1's status)
cmd1 && cmd2         # run cmd2 only if cmd1 succeeded (exit 0)
cmd1 || cmd2         # run cmd2 only if cmd1 failed (exit non-zero)
cmd1 & cmd2          # run cmd1 in background, run cmd2 immediately
```

Backgrounding and waiting:

```bash
slow_thing &                 # run in background
pid=$!                       # save PID
do_other_work
wait "$pid"                  # wait for it; $? = its exit code
echo "child rc: $?"

# wait for ALL background children:
slow_thing &
slower_thing &
wait                         # waits for all; $? = 0 always (POSIX)

# kill it:
kill "$pid"                  # send TERM (15) — graceful
kill -INT "$pid"             # SIGINT (2) — Ctrl-C equivalent
kill -KILL "$pid"            # SIGKILL (9) — uncatchable
kill -0 "$pid"               # check if alive without signaling

# job control (works in interactive shells; spotty in scripts):
jobs                         # list jobs
fg %1                        # foreground job 1
bg %1                        # background it
```

`disown` is **bash-only**. The POSIX equivalent of "let it survive script exit" is `nohup`:

```bash
nohup long_running >out.log 2>&1 &
```

## Common External Tools

POSIX guarantees these utilities exist with documented behavior:

```
awk, sed, grep, find, xargs, cut, tr, sort, uniq, head, tail, wc,
paste, join, comm, expand, unexpand, fold, fmt, nl, od, dd, cat,
basename, dirname, mkdir, mkfifo, mktemp (POSIX:2008+), rm, rmdir,
ln, cp, mv, chmod, chown, touch, ls, stat (NO — not POSIX! ls -l only),
test, [, true, false, sleep, kill, ps, env, getopts, getconf,
printf, echo (avoid options — see below), expr, time, tee, tar,
gzip (NOT POSIX — pax is), pax, cpio, du, df, ulimit, who, id,
sh, command, type, hash, alias, unalias, jobs, fg, bg, wait,
trap, exec, exit, return, set, unset, readonly, export, shift,
read, eval, source (NOT POSIX! use `.`), break, continue
```

`stat` is **not POSIX**. Worse, GNU `stat` and BSD `stat` use opposite flag syntax. Portable file metadata uses `ls -l`, `find -printf` (GNU only), or `wc`:

```bash
# size of a file, portably:
size=$(wc -c < file)         # awkward but POSIX
size=$(ls -ln file | awk '{print $5}')

# mtime — really hard portably. Use find:
mtime_iso=$(find file -printf '%TY-%Tm-%Td' 2>/dev/null) # GNU only
# fallback: ls -l output parsing... (don't go there if you can avoid it)
```

`echo` flags are non-portable. POSIX says: no `-n`, no `-e`. Use `printf`:

```bash
echo -n hello       # might print "-n hello" on some shells (Solaris /bin/sh!)
printf 'hello'      # always works, no trailing newline
printf '%s\n' "$x"  # safe for any value of $x including starting with -
```

Other landmines:

```bash
sed -i              # IN-PLACE EDIT — incompatible:
sed -i 's/x/y/' f         # GNU
sed -i '' 's/x/y/' f      # BSD/macOS — empty backup-suffix is REQUIRED
sed -i.bak 's/x/y/' f     # works on both! always provide a suffix.

grep -P            # PCRE — GNU only, not POSIX (POSIX has -E for ERE, -F for fixed)
grep -E            # extended regex — POSIX
grep -F            # fixed strings — POSIX
grep -r            # recursive — POSIX:2017; older POSIX uses find + grep

awk gensub()       # GNU awk only
awk strftime()     # GNU awk only

xargs -P           # parallel — GNU/BSD; not POSIX
xargs -I {}        # POSIX (replace string)
xargs -0           # NUL-delimited — GNU/BSD; not POSIX
```

## sed for Scripting

```bash
sed 's/old/new/'             # first match per line
sed 's/old/new/g'            # all matches per line
sed 's|old|new|'             # alt delimiter (handy if pattern has /)
sed -n '5p'                  # print only line 5; -n suppresses default output
sed -n '5,10p'               # range
sed '5d'                     # delete line 5
sed '/pattern/d'             # delete lines matching pattern
sed -e 's/a/A/' -e 's/b/B/'  # multiple commands
sed -E 's/[0-9]+/N/'         # ERE — POSIX:2017+ (older systems used GNU's -r only)

# in-place — see warning above; provide a suffix to be portable:
sed -i.bak 's/old/new/' file

# print only N..M then quit (efficient on huge files):
sed -n '100,200p; 200q' big.log

# delete blank lines:
sed '/^$/d' file

# squeeze blank lines:
sed '/^$/N;/^\n$/D' file

# print between markers (inclusive):
sed -n '/BEGIN/,/END/p' file
```

POSIX `sed` does NOT support `\n` in the **replacement** part literally on every shell — use a real newline (escaped):

```bash
sed 's/sep/\
/g' file                     # works everywhere
sed 's/sep/\n/g' file        # GNU-portable; not strictly POSIX
```

## awk Quick Reference

```bash
awk '{print $1}'                          # first field per line
awk -F, '{print $2}'                      # comma-delimited
awk 'NR==5'                               # line 5
awk 'NR%2'                                # odd lines
awk 'END{print NR}'                       # line count (faster than wc -l on some)
awk '/pattern/'                           # like grep
awk '!seen[$0]++'                         # dedupe (preserves order)
awk '{a[$1]+=$2} END{for(k in a) print k,a[k]}'   # group sum
awk -v threshold=100 '$2 > threshold'

# BEGIN / END blocks:
awk 'BEGIN{FS=":"; OFS="|"} {print $1,$3}' /etc/passwd

# printf:
awk '{printf "%-20s %5d\n", $1, $2}'

# arithmetic on epoch (gawk has strftime; mawk doesn't):
awk 'BEGIN{print strftime("%F", systime()-86400)}'    # gawk: yesterday
```

See the dedicated `awk` sheet for depth.

## Common Errors and Fixes

```text
syntax error: unexpected end of file
```
Unbalanced `'`, `"`, `(`, `{`, `[`, here-doc that never closes. Re-read the file watching for nesting. `bash -n script` and `sh -n script` syntax-check without running.

```text
[: missing `]'
[: y: unary operator expected
```
Forgot to close `[ ... ]`, OR a variable was unquoted and expanded to nothing. Always quote: `[ "$x" = "y" ]`.

```text
[: ==: unary operator expected
```
You wrote `==` (bash) instead of `=` in `[ ]`. POSIX uses single `=` for string equality.

```text
syntax error: "(" unexpected
```
Often `[[ ... ]]` or `((...))` or `function name { ... }` — bashisms in `/bin/sh`. Convert to `[ ... ]`, `$(( ... ))`, `name() { ... }`.

```text
sh: 1: source: not found
```
`source` is bash; POSIX uses `.` (a single dot):

```bash
source other.sh              # bash
. other.sh                   # POSIX — note: needs space after dot
```

```text
command not found
```
PATH issue, typo, or you're in a stripped environment. `command -v cmd` checks if `cmd` exists portably.

```text
local: only valid in a function
```
`local` outside a function. Move it inside, or remove it (and rely on subshell scope) for strict POSIX.

```text
read: bad option(s)
```
`read -p`, `read -s`, `read -a`, `read -t` are bashisms. POSIX `read` only accepts `-r`. Use `printf` for prompts, `stty -echo` for silent reads.

The infamous silent `set -e exit`:

```bash
set -e
some_command          # fails — script just disappears with no message
# fix: explicit handling
some_command || die "some_command failed"
```

## POSIX Pitfalls

Bashisms accidentally introduced — broken / fixed pairs.

Arrays:

```bash
# bash:
arr=(a b c)
echo "${arr[1]}"
echo "${arr[@]}"

# POSIX has no arrays. Workarounds:
set -- a b c
echo "$2"            # b
for x do echo "$x"; done

# or use a delimited string + loop with IFS.
```

Case conversion:

```bash
echo "${name^^}"     # bash 4+ — fails on dash
echo "$name" | tr 'a-z' 'A-Z'    # POSIX
```

Substitution:

```bash
echo "${path//\//-}" # bash — replace / with -
echo "$path" | tr / -            # POSIX
echo "$path" | sed 's|/|-|g'     # also POSIX
```

Here-strings:

```bash
grep foo <<<"$x"     # bash/ksh/zsh
printf '%s\n' "$x" | grep foo    # POSIX
```

Test:

```bash
[[ $x == foo* ]]                 # bash
case "$x" in foo*) ;; esac       # POSIX

[[ $x =~ ^[0-9]+$ ]]             # bash
echo "$x" | grep -Eq '^[0-9]+$'  # POSIX
```

Compound:

```bash
(( i++ ))            # bash/ksh — arithmetic command
i=$(( i + 1 ))       # POSIX — assignment

(( a < b ))          # bash/ksh
[ "$a" -lt "$b" ]    # POSIX
```

Brace expansion:

```bash
echo {1..10}         # bash/zsh
seq 1 10             # widely available, NOT strict POSIX
i=1; while [ $i -le 10 ]; do echo $i; i=$((i+1)); done   # strict POSIX
```

Nesting `$()` works in POSIX. Backticks need painful escaping:

```bash
out=$(grep $(date +%Y) log)      # OK in POSIX
out=`grep \`date +%Y\` log`      # works but ugly
```

Some ancient `/bin/sh` (pre-1989 SVR2-derived) had bugs nesting `$()`. Modern POSIX shells handle it fine.

`declare`, `typeset`, `let`, `source`, `function NAME { ... }`, `&>`, `<<<`, `==` in `[`, `[[ ]]`, `(())`, `+=`, `${var:i:j}`, `${!var}`, `$RANDOM`, `$LINENO` (only `$LINENO` is POSIX in newer issues), `$SECONDS`, `$EPOCHSECONDS` — all bashisms.

## Word Splitting and Globbing

The single most important rule: **always quote variable expansions** unless you specifically want word splitting + globbing.

```bash
files="a.txt b.txt"
ls $files            # 2 files: a.txt, b.txt    (you USED splitting)
ls "$files"          # 1 arg: "a.txt b.txt" — looks for that literal name

# IFS controls split characters; default is " \t\n":
old_IFS=$IFS
IFS=:
set -- $PATH         # split PATH into positionals
IFS=$old_IFS
```

Globbing happens after splitting:

```bash
pattern="*.sh"
ls $pattern          # globs to files in cwd matching *.sh
ls "$pattern"        # tries to ls a literal "*.sh" file

# disable globbing for a critical block:
set -f
# ... handle data with literal * or ? in it ...
set +f
```

The "be paranoid" pattern combines `set -f` and explicit IFS:

```bash
old_IFS=$IFS
set -f                         # no globs
IFS=:
set -- $PATH
IFS=$old_IFS
set +f
for d do echo "$d"; done
```

## shellcheck

The single most valuable tool for shell scripting. Static analysis catches 95% of portability and correctness bugs:

```bash
shellcheck script.sh                     # default — assumes #!/bin/sh or shebang
shellcheck -s sh script.sh               # force POSIX sh mode
shellcheck -s bash script.sh             # force bash mode
shellcheck -s dash script.sh             # force dash (strictest POSIX)

# CI integration:
shellcheck -S error script.sh            # fail only on errors
shellcheck -e SC2086 script.sh           # ignore quoting warning (use sparingly)
```

Most-encountered codes:

```text
SC2086  Double-quote to prevent globbing and word splitting.    rm $f → rm "$f"
SC2046  Quote this to prevent word splitting.                   rm $(...)→ rm "$(...)"
SC2155  Declare and assign separately to avoid masking return values.
        local x=$(cmd)  →  local x; x=$(cmd) || return
SC2002  Useless cat. Use < or pipe directly.                    cat f|grep x → grep x f
SC2148  Tips depend on target shell — add a shebang.
SC2034  X appears unused. (review for typo)
SC2126  Consider using grep -c instead of grep|wc -l.
SC2053  Quote the right-hand side of = in [[ ]] to prevent glob matching.
SC2128  Expanding an array without an index only gives the first element.
SC2230  which is non-standard. Use builtin command -v instead.
SC2250  Prefer ${var} for clarity (style only).
SC3002  In POSIX sh, ${arr[@]} is undefined.    (running with -s sh)
SC3010  In POSIX sh, [[ ]] is undefined.
SC3045  In POSIX sh, export with assignment is undefined.       export X=1 → X=1; export X
```

`# shellcheck disable=SC2086` immediately above a line silences a single warning when you actually want splitting.

## Cross-Shell Testing

Run your script under multiple shells before declaring it portable:

```bash
# install on Debian/Ubuntu:
sudo apt install dash busybox bash ksh mksh yash

# install on macOS:
brew install dash bash ksh mksh

# run the script under each:
for sh in dash busybox bash mksh yash; do
    echo "=== $sh ==="
    $sh ./script.sh && echo OK || echo FAIL
done

# bash --posix forces POSIX-only mode (mostly):
bash --posix script.sh

# SHELLOPTS=posix or set -o posix inside bash also forces POSIX-ish.
```

The "lintian style" check Debian uses:

```bash
checkbashisms script.sh
posh script.sh             # `posh` is the Policy-compliant Ordinary SHell
```

`posh` and `yash --posix-strict` are the strictest reference shells — if your script runs under both, it's truly POSIX.

## Performance Patterns

Forking external processes is the dominant cost. Each `$(cmd)` is a fork+exec. Minimise where possible.

```bash
# fork-heavy:
n=$(echo "$s" | wc -c)

# fork-free:
n=${#s}

# fork-heavy:
base=$(basename "$path")

# fork-free:
base=${path##*/}

# fork-heavy:
dir=$(dirname "$path")

# fork-free:
dir=${path%/*}

# fork-heavy:
upper=$(echo "$x" | tr a-z A-Z)

# in bash: upper=${x^^}    — bash only, no fork
# in POSIX, no fork-free way; tr is the cleanest external.

# replace fork-per-line with one awk:
# slow: while read; do echo "$line" | sed ... ; done < f
# fast: sed ... f                        # one fork
# fast: awk '...' f                       # one fork, more powerful
```

Avoid useless `cat`:

```bash
cat file | grep x            # extra fork
grep x file                  # cleaner
grep x < file                # if you really want stdin redirection
```

`exec` replaces the shell process — useful when the script is just a wrapper:

```bash
#!/bin/sh
# pre-flight checks...
exec real_program "$@"       # no extra shell process; real_program inherits PID
```

## Security

Quoting prevents most injection bugs. `eval` is almost always wrong.

```bash
# DANGEROUS:
eval "rm $user_input"        # user can supply '; rm -rf $HOME'

# safer alternatives:
case "$user_input" in
    [a-zA-Z0-9_-]*) rm "$user_input" ;;
    *) die "bad name" ;;
esac
```

`PATH`:

```bash
PATH=/usr/local/bin:/usr/bin:/bin   # set explicitly in scripts that run as root
export PATH
# or use absolute paths for security-critical commands.
```

Temp files:

```bash
# always use mktemp; check exit code:
tmp=$(mktemp -t myscript.XXXXXX) || die "mktemp failed"
trap 'rm -f "$tmp"' EXIT INT TERM HUP
echo "$secret" > "$tmp"

# directory:
tmpdir=$(mktemp -d -t myscript.XXXXXX) || die "mktemp -d failed"
trap 'rm -rf "$tmpdir"' EXIT INT TERM HUP
```

Permissions:

```bash
umask 077                    # files created mode 600, dirs 700 — for sensitive scripts
chmod 600 "$secret_file"
chmod 700 "$secret_dir"
```

`mktemp -d` without `-t` predicate or default template falls back differently across BSD/Linux. Always supply a template ending in 6+ `X`s.

When passing user data to commands, prefer `--`:

```bash
rm -- "$file"                # `--` ends options; protects against $file like "-rf"
grep -- "$pat" "$file"
```

## Idioms

The canonical preamble for a portable script:

```bash
#!/bin/sh
# myscript — short description
#
# Usage: myscript [-v] [-o OUTFILE] INPUT...

set -eu
[ "${TRACE:-0}" = "1" ] && set -x

PROGNAME=${0##*/}

log()  { printf '%s: %s\n' "$PROGNAME" "$*" >&2; }
warn() { printf '%s: warning: %s\n' "$PROGNAME" "$*" >&2; }
die()  { printf '%s: error: %s\n' "$PROGNAME" "$*" >&2; exit 1; }

usage() {
    cat <<EOF
Usage: $PROGNAME [OPTIONS] INPUT...
Options:
    -v          verbose
    -o FILE     output to FILE
    -h          this help
EOF
}

cleanup() {
    [ -n "${tmpdir:-}" ] && [ -d "$tmpdir" ] && rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM HUP

# arg parsing
verbose=0
outfile=
while getopts ':vo:h' opt; do
    case "$opt" in
        v) verbose=$(( verbose + 1 )) ;;
        o) outfile=$OPTARG ;;
        h) usage; exit 0 ;;
        :) die "option -$OPTARG requires an argument" ;;
        \?) die "unknown option: -$OPTARG" ;;
    esac
done
shift $(( OPTIND - 1 ))

[ $# -ge 1 ] || { usage >&2; exit 2; }

# main work
tmpdir=$(mktemp -d -t "${PROGNAME}.XXXXXX") || die "mktemp failed"

for input; do
    [ -r "$input" ] || die "cannot read: $input"
    [ "$verbose" -ge 1 ] && log "processing $input"
    # ...
done

[ -n "$outfile" ] && log "wrote $outfile"
```

The "wrapper that delegates to a binary" idiom:

```bash
#!/bin/sh
# thin wrapper that adds defaults then execs the real binary
exec /usr/libexec/myapp/bin --config /etc/myapp.conf --log "$LOG" "$@"
```

The "find-then-source" idiom (e.g. for plugins):

```bash
for f in "$plugindir"/*.sh; do
    [ -e "$f" ] || continue
    . "$f"                   # POSIX `.` not bash `source`
done
```

The "atomic write" pattern:

```bash
tmp=$(mktemp -t write.XXXXXX) || die "mktemp"
trap 'rm -f "$tmp"' EXIT
generate_content > "$tmp"
mv -f "$tmp" "$target"       # atomic rename within same filesystem
trap - EXIT
```

The "short-circuit defaults" idiom:

```bash
: "${EDITOR:=vi}"            # set EDITOR to vi if unset/empty
: "${LOG:=/var/log/x}"

# `:` is the no-op; `${EDITOR:=vi}` does the assignment as a side effect.
```

## Bashisms to Avoid

A concrete list, with the POSIX equivalent:

```text
arrays           arr=(a b c) / "${arr[@]}"        →  set -- a b c / "$@"  OR delimited string
[[ ... ]]                                           →  [ ... ]   (test)
&> file                                             →  > file 2>&1
&>> file                                            →  >> file 2>&1
<<< "string"                                        →  printf '%s\n' "string" |
+=                                                  →  var="${var}suffix"  /  var=$(( var + 1 ))
(( expr ))                                          →  : $(( expr ))   (use the value or discard it)
{1..10}                                             →  seq 1 10  /  while loop
$RANDOM                                             →  awk 'BEGIN{srand(); print int(rand()*32768)}'
$LINENO                                             →  POSIX:2017 — older shells lack it
$SECONDS                                            →  start=$(date +%s); now=$(date +%s); echo $((now-start))
$BASH_*                                             →  not portable; avoid
${var//x/y}                                         →  printf '%s' "$var" | sed 's/x/y/g'
${var^^} / ${var,,}                                 →  echo "$var" | tr a-z A-Z   /   tr A-Z a-z
${!var}                                             →  eval "v=\$$var"   (rarely necessary; usually a design smell)
${var:offset:len}                                   →  cut -c offset+1-...   /   awk substr()
source f                                            →  . f
function name { ... }                               →  name() { ... }
declare -X / typeset                                →  not POSIX (most shells have local)
local outside a function                            →  not allowed even where local exists
echo -e / echo -n                                   →  printf
read -p / -s / -t / -a / -d / -i / -n / -N         →  printf / stty / read in a loop
mapfile / readarray                                 →  while read; do ... done
shopt                                               →  use set -o where possible
caller / FUNCNAME                                   →  not portable
trap ... ERR                                        →  not POSIX; use explicit checks
trap ... DEBUG                                      →  not POSIX
PROMPT_COMMAND                                      →  not POSIX
PS3 / PS4 in scripts                                →  PS4 is POSIX since 2017
$'...'  ANSI-C quoting                              →  printf for special chars
```

A subtle one: assigning to a variable in front of a command scopes the var to that command, **including for builtins**:

```bash
LANG=C ls               # ls runs with LANG=C; outer LANG unchanged. POSIX. Universal.
IFS=: read -r a b c     # IFS only changes for this read.
```

This is the closest portable analogue of "local env var to one command".

## When NOT to Use POSIX sh

POSIX sh is excellent for: small wrappers, init scripts, install/configure scripts, CI glue, anything that has to run on a barebones box. It is bad for:

- **Real arrays / associative arrays.** Bash 4 has them, zsh has them, ksh has them. POSIX does not. If you need maps of maps, stop.
- **Unicode-aware string operations.** POSIX `${#var}` is bytes, not characters; `tr` works per byte. Reach for Python, Perl, or `awk` (gawk) with `LC_ALL=en_US.UTF-8`.
- **Complex data structures.** Same as above.
- **Floating point math.** `bc`, `awk`, or a real language.
- **Robust JSON.** `jq` is the answer; if you need to write JSON by hand in shell, you've lost.
- **Performance-sensitive loops on big data.** Forks dominate; rewrite the inner loop in awk/perl/python.
- **Scripts longer than ~200–300 lines.** Maintainability dies. Switch to Python or Go.

Rule of thumb: if you find yourself reaching for arrays, indirect references, or fancy string ops, escalate to bash (and document the requirement) or to a real language.

## Tips

- Add `set -eu` to every script. Add `pipefail` if you control the target. Add `set -x` (or guard it with a TRACE env var) for debugging.
- Quote every expansion. Period. Use `shellcheck` to catch the ones you forgot.
- Prefer `printf '%s\n'` over `echo` for unknown content.
- Use `$()` instead of backticks. Always.
- Use `$(( ))` for math. Don't fork `expr`.
- Inside `[ ]`, use `=` for strings and `-eq` for numbers. Quote both sides.
- Use `case` instead of chains of `if [ "$x" = "y" ]`. It's faster and reads better.
- For loops, iterate the glob directly: `for f in *.txt; do [ -e "$f" ] || continue; ...`. Never `for f in $(ls)`.
- For long options, hand-write the parser. Don't use external `getopt(1)` — its flags differ between BSD and GNU.
- `mktemp -d -t name.XXXXXX` for temp dirs; `trap 'rm -rf "$tmpdir"' EXIT` for cleanup.
- Use `command -v cmd >/dev/null 2>&1` instead of `which cmd` (which is non-POSIX and unreliable).
- Lock with `mkdir`, not `[ -e file ]` + `touch`. Atomicity matters.
- Save and restore `IFS` if you change it. The same for `set -f`.
- Read with `IFS= read -r`. Always both.
- Test under dash and busybox sh, not just bash. Most "POSIX" scripts have at least one bashism that only dash will reveal.
- Comment the WHY, not the WHAT. Future-you will thank you when revisiting a `case "$x" in *) ;; esac`.
- Keep functions tiny. The shell is not a great host for large programs.
- If a script is hard to write portably, the answer might be "rewrite in Python/Go", not "fight POSIX harder".

## See Also

- bash
- zsh
- fish
- nushell
- polyglot
- regex
- awk
- sql

## References

- POSIX shell language: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html
- POSIX utilities index: https://pubs.opengroup.org/onlinepubs/9699919799/idx/utilities.html
- Greg's wiki — Bash guide (applies to POSIX too): https://mywiki.wooledge.org/BashGuide
- Greg's wiki — Bash FAQ (most entries note POSIX vs bash): https://mywiki.wooledge.org/BashFAQ
- Greg's wiki — Bash pitfalls: https://mywiki.wooledge.org/BashPitfalls
- shellcheck — static analyzer: https://www.shellcheck.net/
- shellcheck wiki — code reference: https://www.shellcheck.net/wiki/
- checkbashisms (Debian devscripts): https://manpages.debian.org/checkbashisms
- dash man page: https://manpages.debian.org/dash
- POSIX Programmer's Manual (Linux): https://man7.org/linux/man-pages/man1/sh.1p.html
- POSIX `test` / `[`: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/test.html
- POSIX `getopts`: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/getopts.html
- POSIX `printf`: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/printf.html
- POSIX `awk`: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/awk.html
- POSIX `sed`: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/sed.html
- POSIX `find`: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/find.html
- POSIX `xargs`: https://pubs.opengroup.org/onlinepubs/9699919799/utilities/xargs.html
- The Open Group base spec home: https://pubs.opengroup.org/onlinepubs/9699919799/
- Rich's sh tricks: https://www.etalabs.net/sh_tricks.html
- Ubuntu DashAsBinSh policy: https://wiki.ubuntu.com/DashAsBinSh
- Autoconf portable shell guide: https://www.gnu.org/software/autoconf/manual/autoconf.html#Portable-Shell
- Bash reference manual (for contrast): https://www.gnu.org/software/bash/manual/bash.html
