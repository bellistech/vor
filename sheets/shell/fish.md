# Fish (Friendly Interactive Shell)

A user-friendly, opinionated shell with sane defaults: no word-splitting, autosuggestions, syntax highlighting, abbreviations, and a clean (non-POSIX) scripting language for fish 3.x (3.7+ recommended).

## Setup

```bash
# Install fish
brew install fish                       # macOS / Linuxbrew
sudo apt install fish                   # Debian / Ubuntu
sudo dnf install fish                   # Fedora / RHEL 9+
sudo pacman -S fish                     # Arch / Manjaro
sudo zypper install fish                # openSUSE
sudo apk add fish                       # Alpine
sudo xbps-install fish-shell            # Void

# Verify install
fish --version                          # fish, version 3.7.1
echo $FISH_VERSION                      # only set inside fish
fish_version                            # builtin command (3.6+)

# Make fish your default login shell
which fish                              # e.g. /opt/homebrew/bin/fish
echo (which fish) | sudo tee -a /etc/shells
chsh -s (which fish)                    # change login shell for current user
chsh -s /usr/local/bin/fish $USER

# Try fish without changing default
fish                                    # interactive subshell
exec fish                               # replace current shell

# Configuration paths (see "Config files")
echo $__fish_config_dir                 # ~/.config/fish
echo $__fish_user_data_dir              # ~/.local/share/fish
echo $__fish_sysconfdir                 # /etc/fish (system-wide)

# Reload config
exec fish                               # nuke and restart
source ~/.config/fish/config.fish       # re-source (most things)
```

## Why Fish

```bash
# Fish's design choices, summarized:

# 1. NO WORD-SPLITTING.  $var is one argument, even with spaces.
set greeting "hello world"
echo $greeting                          # one arg: "hello world"
# In bash: echo $greeting -> two args.  In fish: always one.

# 2. NON-POSIX SCRIPTING LANGUAGE.
# fish is NOT a POSIX shell.  Bash/POSIX scripts must be rewritten.
# Trade-off: cleaner syntax, but you can't paste bash one-liners.

# 3. ABBREVIATIONS expand on space, leaving the full command in history.
abbr -a gco 'git checkout'
# Type `gco main` + space -> shell rewrites to `git checkout main`.

# 4. AUTOSUGGESTIONS as you type (gray text from history & completions).
# Press Right-Arrow or Ctrl-F to accept.

# 5. SYNTAX HIGHLIGHTING built-in.
# Red = invalid command/path; blue/green = valid; underline = directory.

# 6. SANE DEFAULTS — no plugin needed for:
#  - History search (substring, type Up arrow)
#  - Tab completion that learns from man pages
#  - Multiline editing
#  - Per-directory history search

# 7. EVERY VARIABLE IS A LIST (1-indexed).
set fruits apple banana cherry
echo $fruits[2]                         # banana

# 8. LISTS DON'T WORD-SPLIT INTO ARGS UNEXPECTEDLY.
set files (ls)                          # exact filenames preserved
```

## Config files

```bash
# Per-user config (canonical paths, follow XDG)
~/.config/fish/config.fish              # always sourced (interactive AND scripts)
~/.config/fish/conf.d/*.fish            # snippets, sourced before config.fish, alphabetical
~/.config/fish/functions/*.fish         # autoloaded functions (file = function name)
~/.config/fish/completions/*.fish       # autoloaded completions
~/.config/fish/fish_variables           # universal variable database (managed; do not hand-edit normally)
~/.local/share/fish/fish_history        # interactive history (YAML-ish)

# System-wide config
/etc/fish/config.fish                   # system config (root-managed)
/etc/fish/conf.d/*.fish                 # system snippets
/etc/fish/functions/*.fish              # system functions
/etc/fish/completions/*.fish            # system completions

# Vendor directories (packaged-software drop-ins)
/usr/share/fish/vendor_conf.d/          # package conf.d snippets
/usr/share/fish/vendor_functions.d/     # package functions
/usr/share/fish/vendor_completions.d/   # package completions
# Use these instead of patching /etc/ when shipping a package.

# config.fish skeleton — guard interactive-only setup
if status is-interactive
    # interactive prompt setup, key bindings, abbr, etc.
    abbr -a g git
    set -g fish_greeting ""
end

# Always-run setup (universal vars, PATH for scripts) goes outside the guard.

# conf.d (preferred for modular config)
# ~/.config/fish/conf.d/10-path.fish
fish_add_path ~/.local/bin
# ~/.config/fish/conf.d/20-aliases.fish
abbr -a ll 'ls -la'
# Files run alphabetically -- prefix with numbers to control order.

# Autoload behavior:
# functions/foo.fish must define `function foo`.  fish loads it on first call.
# completions/foo.fish runs the first time you tab-complete `foo`.
```

## Variables

```bash
# Set a variable (default scope = function-local)
set name "Alice"

# Scope flags
set -l name "Alice"                     # local: this block/function only
set -g name "Alice"                     # global: this fish session
set -U name "Alice"                     # universal: persists across sessions
set -x name "Alice"                     # exported (env var, child processes see it)
set -Ux GOPATH $HOME/go                 # universal AND exported
set -gx LANG en_US.UTF-8                # global AND exported

# Erase
set -e name                             # erase variable
set -e --universal name                 # erase universal var
set -e PATH[3]                          # erase index 3 of list

# Append / prepend / index assignment
set -a fruits orange                    # append element to list
set -p fruits kiwi                      # prepend element to list
set fruits[2] grape                     # replace index 2

# Query
set -q name                             # exit 0 if set, 1 otherwise
set -q name; and echo "yes"
set -S name                             # show all scopes containing $name
set --names                             # list all variable names
set                                     # list all variables + values

# The 4 scopes (narrowest to widest)
# 1. local       — current block (if/for/function body)
# 2. function    — current function (default in functions)
# 3. global      — current shell session (default at top level)
# 4. universal   — all current and future sessions on this user

# Implicit scope rules:
#   - Inside a function with no flag => function scope
#   - At top level with no flag       => function scope (which == global there)
#   - -l forces local even inside if/while
#   - Existing variable: `set` modifies its existing scope unless overridden

# Read-only / electric vars (set automatically by fish):
echo $status                            # exit code of last foreground command
echo $pipestatus                        # list of exit codes for pipes
echo $argv                              # function/script arguments (list)
echo $PWD                               # current directory
echo $CMD_DURATION                      # ms last command took
echo $fish_pid                          # PID of this fish
echo $last_pid                          # PID of last backgrounded job
echo $version                           # fish version (also $FISH_VERSION)
```

## Universal Variables

```bash
# Universal variables persist across all current and future fish sessions
# for the current user.  Stored in ~/.config/fish/fish_variables.

# Set a universal var
set -U EDITOR nvim
set -U fish_greeting ""                 # disable greeting
set -U fish_color_command brblue        # change syntax-highlight color

# Exported universal var (child procs see it)
set -Ux GOPATH $HOME/go
set -Ux PYENV_ROOT $HOME/.pyenv

# Modify -- changes propagate to all running fish sessions IMMEDIATELY.
# Open two terminals, set -U FOO bar in one, echo $FOO in the other.

# List universal vars
set -U                                  # only universal scope (--universal)

# Erase universal var
set -eU GOPATH

# Migrate to/from non-universal:
set -e EDITOR                           # erase first
set -U EDITOR nvim                      # then re-set with desired scope
# Cannot have same name in two scopes simultaneously when re-setting.

# fish_variables file format (do not hand-edit while fish is running):
# SETUVAR --export GOPATH:/Users/me/go
# SETUVAR fish_color_command:brblue

# vs bash/zsh dotfiles:
#   bash: edit ~/.bashrc, source it, sometimes restart
#   zsh:  edit ~/.zshrc, source it
#   fish: just `set -U` once -- no file editing required

# WARNING: universal vars are PER-USER, PER-MACHINE.  Don't dotfile them via git.
# Use conf.d snippets that call `set -U` if needed cross-machine.

# fish_user_paths is the canonical universal PATH-extension list:
set -U fish_user_paths $HOME/.local/bin $HOME/bin /usr/local/sbin
# fish prepends fish_user_paths to PATH automatically.
```

## Quoting

```bash
# Single quotes — literal (NO interpolation, NO escapes except \\ and \')
echo 'hello $USER \n'                   # hello $USER \n  (literal)

# Double quotes — variable expansion, but NO command substitution syntax
echo "hello $USER"                      # hello alice
echo "today is "(date)                  # interpolate command sub via concatenation

# Escape sequences in DOUBLE quotes are limited:
# \"  literal double quote
# \\  backslash
# \$  literal dollar sign
# Other escapes pass through literally:
echo "tab\tnewline"                     # literal: tab\tnewline (NOT a tab!)

# Fish has NO ANSI-C $'...' quoting (which bash uses).
# Broken (bash idiom):
#   echo $'line1\nline2'                # fish: prints literally, no newline
# Fixed in fish (use printf or echo -e):
printf 'line1\nline2\n'
echo -e 'line1\nline2'

# Backslash escapes outside quotes
echo \$HOME                             # $HOME (literal)
echo \"hi\"                             # "hi"
echo a\ b                               # a b (single arg "a b")

# CRITICAL: fish DOES NOT WORD-SPLIT.
# In bash:
#   files="a b c"; ls $files            # treats $files as 3 args
# In fish:
set files "a b c"
ls $files                               # ONE arg, the literal string "a b c"
# To pass multiple args, store as a list:
set files a b c
ls $files                               # 3 args: a, b, c

# Quoted vs unquoted — almost always identical in fish:
set name "alice"
echo $name                              # alice
echo "$name"                            # alice  (no difference)
# Exception: empty list expansion.  Unquoted empty list disappears, quoted gives "".
set empty
count $empty                            # 0
count "$empty"                          # 1 (empty string)
```

## Variable Expansion

```bash
# Basic
set greeting hello
echo $greeting                          # hello

# Concatenation — adjacent tokens concatenate
echo $HOME/bin                          # /Users/alice/bin
echo "Hello, "$USER"!"                  # Hello, alice!
echo $a$b                               # cartesian product if both are lists!

# NO curly-brace expansion
# Broken (bash):
#   echo ${var}_suffix
# Fixed (fish):
echo {$var}_suffix                      # SAME thing? NO — {} is brace expansion
echo $var"_suffix"                      # OK
echo "$var"_suffix                      # OK

# fish DOES have brace expansion (bash-like, comma list):
echo {a,b,c}                            # a b c
echo file.{txt,md}                      # file.txt file.md
mkdir -p src/{lib,bin,test}             # 3 dirs
# Brace expansion happens BEFORE variable expansion.

# Multi-value vars expand to multiple args (no quoting needed)
set files a.txt b.txt c.txt
echo $files                             # a.txt b.txt c.txt
ls $files                               # ls receives 3 separate args

# Quoting "$var" does NOT prevent split-into-args; lists still expand:
echo "$files"                           # "a.txt" "b.txt" "c.txt" -- 3 separate args
# This is unlike bash where "$@" preserves args but "$*" merges.

# Cartesian product of lists
set a x y
set b 1 2
echo $a$b                               # x1 x2 y1 y2

# Length / count
echo (count $files)                     # 3

# Indexing during expansion
echo $files[1]                          # a.txt   (1-indexed!)
echo $files[-1]                         # c.txt   (last)
echo $files[2..3]                       # b.txt c.txt

# Slice in concatenation
echo prefix-$files[1]                   # prefix-a.txt

# Escape an unset var so it doesn't error
echo $does_not_exist                    # (empty, no error in expansion)
echo "$does_not_exist"                  # (empty)
```

## Lists

```bash
# Every fish variable is a list (sometimes called array).
# 1-indexed.  Negative indices count from the end.

# Create
set fruits apple banana cherry          # 3-element list
set empty                               # empty list (zero elements)
set one_element "just one"

# Length
count $fruits                           # 3

# Index (1-based)
echo $fruits[1]                         # apple
echo $fruits[3]                         # cherry
echo $fruits[-1]                        # cherry (last)
echo $fruits[-2]                        # banana (second-to-last)

# Out-of-range error:
echo $fruits[10]                        # error: array index out of bounds
# Test first: set -q fruits[10]; or (count $fruits) -ge 10

# Slices (range)
echo $fruits[1..2]                      # apple banana
echo $fruits[2..-1]                     # banana cherry
echo $fruits[-2..-1]                    # banana cherry

# Append / prepend (preferred over concat)
set -a fruits date                      # apple banana cherry date
set -p fruits apricot                   # apricot apple banana cherry date

# Replace element
set fruits[2] BANANA                    # apricot apple BANANA cherry date

# Erase element
set -e fruits[1]                        # apple BANANA cherry date

# Concatenate lists
set all $list_a $list_b

# Iterate
for f in $fruits
    echo "fruit: $f"
end

# Index iterate (parallel to indices)
for i in (seq (count $fruits))
    echo "$i: $fruits[$i]"
end

# Contains
contains apple $fruits                  # exit 0 if found
contains -i apple $fruits               # print index of first match (or 0)

# Reverse
for f in $fruits[-1..1]
    echo $f
end

# Sort (no builtin sort, pipe to coreutils)
printf '%s\n' $fruits | sort
```

## Arithmetic

```bash
# math is the canonical arithmetic builtin (operates on STRINGS, returns string)
math '2 + 3'                            # 5
math '10 / 3'                           # 3.3333333
math '10 // 3'                          # 3       (integer division)
math '10 % 3'                           # 1       (modulo)
math '2 ^ 10'                           # 1024    (power)
math '-5 + 3'                           # -2
math 'abs(-5)'                          # 5

# Use in assignments and substitution
set sum (math 1 + 2)
set width 80
echo (math "$width / 2")                # 40

# Precision / scale
math -s 2 '10 / 3'                      # 3.33    (2 decimals)
math -s 0 '10 / 3'                      # 3       (truncate to int)

# Built-in math functions
math 'sqrt(2)'                          # 1.4142135
math 'sin(pi/2)'                        # 1
math 'cos(0)'                           # 1
math 'tan(pi/4)'                        # 0.999...
math 'log(e)'                           # 1       (natural log)
math 'log10(1000)'                      # 3
math 'log2(1024)'                       # 10
math 'exp(1)'                           # 2.7182818
math 'ceil(3.2)'                        # 4
math 'floor(3.8)'                       # 3
math 'round(3.5)'                       # 4
math 'min(3,7,2)'                       # 2
math 'max(3,7,2)'                       # 7
math 'pow(2, 10)'                       # 1024

# Constants
math 'pi'                               # 3.1415927
math 'e'                                # 2.7182818

# Bitwise (fish 3.x: math handles via functions)
math '0xff & 0x0f'                      # 15
math '1 << 4'                           # 16
math '0xff ^ 0xaa'                      # XOR

# Hex / oct / bin literals
math '0xFF'                             # 255
math '0o17'                             # 15
math '0b101'                            # 5

# Comparison (returns 0/1)
math '5 > 3'                            # 1
math '5 == 5'                           # 1

# String-to-number coercion
math "$counter + 1"

# Errors:
math 'foo'                              # math: Error: ...  exit 1
# Trap with `or`: set v (math '1/0'); or set v 0
```

## Strings

```bash
# `string` is the everything-builtin for text processing.
# Subcommands: length sub upper lower escape unescape join split match
#              replace repeat trim pad collect split0 join0

# string length
string length 'hello'                   # 5
string length --visible '\033[31mred\033[0m'   # 3 (visible chars)

# string upper / lower / title-case (no title-case builtin; use awk)
string upper 'hello'                    # HELLO
string lower 'WORLD'                    # world

# string sub  -- substring (1-indexed)
string sub -s 2 'hello'                 # ello
string sub -s 2 -l 3 'hello'            # ell
string sub -s -3 'hello'                # llo  (last 3)
string sub -e 3 'hello'                 # hel  (until index 3)

# string match — glob OR regex
string match 'foo*' foobar              # foobar
string match -r '^[a-z]+' 'abc123'      # abc
string match -rq '^\d+$' "$x"; and echo numeric
string match -rg '(\d+)' 'port:8080'    # 8080  (capture group only, -g)
string match -e 'sub' 'subway'          # subway (-e prints whole, returns 0)
string match -v 'foo' bar               # bar (invert match)
string match -i 'FOO' foo               # foo (case-insensitive)

# string replace
string replace 'a' 'b' 'banana'         # bbnana (first match)
string replace -a 'a' 'b' 'banana'      # bbnbnb (all matches, -a)
string replace -r '(\w+)@(\w+)' '$2/$1' 'user@host'    # host/user
string replace --filter 'xx' 'YY' (echo line1; echo xxline2)
                                        # only outputs lines that matched (-f)

# string split / split0
string split ',' 'a,b,c'                # 3 lines: a / b / c
string split -m1 '=' 'k=v=more'         # k / v=more (max splits)
string split0                           # split on \0 bytes
printf 'a\0b\0c\0' | string split0      # a / b / c

# string join / join0
string join ',' a b c                   # a,b,c
string join \n a b c                    # multi-line
string join0 a b c | xargs -0 echo      # NUL-separated for xargs

# string trim
string trim '  hi  '                    # hi
string trim -l '  hi  '                 # 'hi  '   (left only)
string trim -r '  hi  '                 # '  hi'   (right only)
string trim -c '/' '/path/'             # path

# string repeat
string repeat -n 3 'ab'                 # ababab
string repeat -n 5 -m 3 'ab'            # aba   (max chars 3)

# string pad
string pad -w 10 'hi'                   #         hi   (right-align by default)
string pad -w 10 -r 'hi'                # hi          (left-align)
string pad -w 10 -c 0 '42'              # 0000000042

# string escape / unescape
string escape 'hello world'             # hello\ world (shell-safe)
string escape --style=script 'a$b'      # 'a$b'
string escape --style=url 'a b'         # a%20b
string escape --style=regex 'a.b'       # a\.b
string unescape 'hello\ world'          # hello world

# string collect — join stdin into a single string (preserve newlines)
set greeting (printf 'a\nb\nc\n' | string collect)
echo "$greeting"                        # a\nb\nc as single multiline string

# Common pipelines
ls | string match -r '\.md$'            # only .md files
echo "$line" | string trim | string lower
string split -n ' ' "a   b  c"          # -n drops empty results

# Performance: string is FAST.  It replaces sed/awk/tr/cut for most cases.
```

## Globbing

```bash
# Standard wildcards
*                                       # any chars except / and leading .
?                                       # single char
[abc]                                   # one of a/b/c
[a-z]                                   # range
**                                      # recursive ANY (including subdirs)

# Examples
ls *.md                                 # files ending .md (current dir)
ls **.md                                # ALL .md files recursively (incl. cur dir)
ls **/*.md                              # all .md in subdirs only
ls test_[0-9].txt                       # test_0.txt..test_9.txt

# fish ERRORS by default if no match:
ls *.zzz                                # error: no matches for: '*.zzz'

# Equivalent of bash `nullglob` — silence the error
ls *.zzz 2>/dev/null; or true
# Or use a wrapper that survives no-match:
function safe_glob; for f in $argv; test -e $f; and echo $f; end; end

# Shell flag to control no-match behavior
set -g fish_no_match_glob 1             # NOT a real var; not supported in fish
# Instead, use a helper function or check before iterating:
set files *.txt
if test -z "$files[1]"
    echo no files
end

# Hidden files
ls .*                                   # only dotfiles
ls {.,}*                                # both hidden and visible

# Case-insensitive globbing (no flag) — emulate via find
find . -iname '*.md'

# Behavior differences vs bash:
# - bash: `for f in *.zzz` silently iterates 0 times (default)
# - fish: same `for` loop ERRORS unless prefixed: `for f in *.zzz 2>/dev/null`
#   OR check: set files *.zzz 2>/dev/null; for f in $files; ...; end
```

## Conditionals

```bash
# if / else if / else / end
if test -f config.fish
    echo "exists"
else if test -d /etc/fish
    echo "dir exists"
else
    echo "neither"
end

# Use any command's exit code
if grep -q TODO file.txt
    echo "found TODO"
end

# Negation with `not`
if not test -f missing
    echo "absent"
end

# Combinators: and / or (keywords, NOT &&/||)
if test -f a.txt; and test -f b.txt
    echo "both exist"
end

# Group with begin..end
if begin; test -f a; or test -f b; end; and test -f c
    echo "(a or b) AND c"
end

# string match in conditionals (often cleaner than test)
if string match -q '*.md' $file
    echo "markdown"
end

if string match -rq '^v\d+\.' $version
    echo "starts with v<digit>."
end

# switch / case
switch $argv[1]
    case start
        echo starting
    case stop
        echo stopping
    case 'reload*'                      # glob pattern
        echo reloading
    case '*'
        echo "unknown: $argv[1]"
end

# case patterns are GLOBS by default (not regex):
switch $file
    case '*.md'
        echo markdown
    case '*.txt' '*.log'                # multiple patterns
        echo text
end

# Ternary-ish
set msg (test -f file; and echo found; or echo missing)

# Status code chains
mkdir -p out; and cd out; and echo done
mkdir bad 2>/dev/null; or echo "could not mkdir"
```

## test command

```bash
# `test` (and `[`) are POSIX-style file/string/numeric tests.
# Synonym: [ EXPR ]    (the closing ] is required)

# File tests
test -e file                            # exists (any type)
test -f file                            # regular file
test -d dir                             # directory
test -L link                            # symlink
test -r file                            # readable
test -w file                            # writable
test -x file                            # executable
test -s file                            # exists AND non-empty
test -b file                            # block device
test -c file                            # character device
test -p file                            # named pipe (FIFO)
test -S file                            # socket
test -k file                            # sticky bit
test -u file                            # setuid
test -g file                            # setgid
test -O file                            # owned by user
test -G file                            # owned by user's group

# File comparison
test FILE1 -nt FILE2                    # FILE1 newer than FILE2
test FILE1 -ot FILE2                    # older than
test FILE1 -ef FILE2                    # same inode (hardlink)

# String tests
test -z "$s"                            # empty (zero length)
test -n "$s"                            # non-empty
test "$a" = "$b"                        # equal
test "$a" != "$b"                       # not equal
# NOTE: test uses single =, NOT == (some shells accept ==).
# fish-3.x's test ALSO accepts == as alias for =, but stick to =.

# Numeric tests
test 5 -eq 5                            # equal
test 5 -ne 3                            # not equal
test 5 -lt 7                            # less than
test 5 -le 5                            # less or equal
test 7 -gt 5                            # greater
test 7 -ge 7                            # greater or equal

# Combinators
test -f a.txt -a -f b.txt               # AND  (-a)
test -f a.txt -o -f b.txt               # OR   (-o)
test ! -f a.txt                         # NOT
# PREFERRED: use fish's `and` / `or` keywords instead, clearer:
test -f a.txt; and test -f b.txt

# Grouping
test \( -f a -o -f b \) -a -f c

# Bracket form
[ -f file.txt ]; and echo yes           # space inside brackets is required!
[ "$x" = "$y" ]                         # equal

# Common pitfalls
# 1. Empty variable explosion (fish DOES NOT have this — no word-split!)
#    bash:  test $maybe_empty = "x"   # syntax error if empty
#    fish:  test "$maybe_empty" = "x" # works, $maybe_empty is one arg either way
# 2. POSIX test does NOT support ==.  Use =.
# 3. -a/-o are deprecated in POSIX; prefer `; and` / `; or`.
# 4. Use `string match` for glob/regex matching (test does not glob).
```

## Loops

```bash
# for loop — iterate over a list
for f in *.md
    echo $f
end

# Numeric range with seq
for i in (seq 1 10)
    echo $i
end

for i in (seq 0 2 20)                   # start 0, step 2, end 20
    echo $i
end

for i in (seq 10 -1 1)                  # countdown
    echo $i
end

# Iterate over function args
function greet
    for name in $argv
        echo "hi $name"
    end
end

# while loop
set i 0
while test $i -lt 10
    echo $i
    set i (math $i + 1)
end

# Read lines from a file
while read -l line
    echo "got: $line"
end < /etc/hosts

# Infinite loop with break
while true
    read -P 'q to quit: ' answer
    if test "$answer" = q
        break
    end
end

# break / continue
for n in (seq 1 100)
    if test $n -eq 50
        break
    end
    if test (math $n % 2) -eq 0
        continue
    end
    echo $n
end

# Loop with else (no equivalent — emulate)
set found 0
for f in *.md
    set found 1
end
if test $found -eq 0
    echo "no markdown files"
end

# Iterate over a command's output (line-by-line)
for line in (cat /etc/hosts)
    echo "$line"
end
# WARNING: (cmd) splits on NEWLINES.  See "Process Substitution" below.
```

## Functions

```bash
# Define a function
function greet
    echo "hello, $argv[1]"
end

# Description (shows in `functions`, `funced`, completions)
function greet --description 'say hello'
    echo "hello, $argv[1]"
end

# Arguments via $argv (always a list)
function add
    set sum 0
    for n in $argv
        set sum (math $sum + $n)
    end
    echo $sum
end
add 1 2 3 4                             # 10

# Useful argv shortcuts
function demo
    echo "name:  $argv[1]"
    echo "rest:  $argv[2..-1]"
    echo "last:  $argv[-1]"
    echo "count: "(count $argv)
end

# Local variables (default in functions, but `-l` is explicit)
function f
    set -l tmp /tmp/work
    # tmp is visible only inside f
end

# Return values — print to stdout, AND/OR return exit code
function is_dir
    test -d $argv[1]
    return $status                      # explicit exit code
end

# Wrap a command (preserve completions)
function ls --wraps ls --description 'colorized ls'
    command ls -GH $argv
end
# `command` bypasses functions (avoids recursion)

# Save a function permanently
funcsave greet                          # writes ~/.config/fish/functions/greet.fish
funced greet                            # edit in $EDITOR, then funcsave

# List / show / erase
functions                               # all functions
functions greet                         # show definition
functions -e greet                      # erase
functions --details greet               # show file path

# Autoload: drop greet.fish in ~/.config/fish/functions/
# File MUST be named after the function (greet.fish defines `function greet`).
# Autoloaded on first invocation.

# Anonymous-ish (one-liner) using `;`
function hi; echo hi; end

# Nested functions are NOT supported.  Define at top-level or in autoload files.
```

## Function Decorators

```bash
# Functions can react to events via flags on the `function` definition.

# --on-event NAME — run when `emit NAME` fires
function announce --on-event my_event
    echo "got my_event with args $argv"
end
emit my_event hello world               # triggers announce

# Built-in events:
function on_prompt --on-event fish_prompt
    # runs before each prompt
end
function on_pre  --on-event fish_preexec
    # runs after Enter, before command runs
end
function on_post --on-event fish_postexec
    # runs after command finishes, before next prompt
end
function on_exit --on-event fish_exit
    echo bye
end
function on_cnf  --on-event fish_command_not_found
    echo "404: $argv[1]"
end

# --on-variable NAME — run when variable changes
function on_pwd --on-variable PWD
    echo "now in $PWD"
end

# --on-job-exit JOBSPEC — run when a job exits
sleep 60 &
function on_done --on-job-exit %last
    echo "sleep finished"
end

# --on-process-exit PID — run when a specific PID exits
function bye --on-process-exit $fish_pid
    echo "shell exiting"
end
# (More common: %self, %last, or numeric pid)

# --on-signal SIGNAL — run on signal (use SIGINT, SIGTERM, etc.)
function on_int --on-signal SIGINT
    echo "got SIGINT"
end

# --no-scope-shadowing — function shares parent's local scope (rare)
function inner --no-scope-shadowing
    set foo 42                          # modifies caller's $foo
end

# Inspect handlers
functions --handlers                    # list all registered handlers
functions --handlers-type variable      # filter by type
```

## Events

```bash
# Standard events fish emits:
fish_prompt                             # before drawing prompt
fish_preexec                            # after Enter, before command exec
fish_postexec                           # after command finishes
fish_exit                               # shell exiting
fish_cancel                             # Ctrl-C cancelled commandline
fish_command_not_found                  # cmd not found (replaces command_not_found_handler)
fish_posterror                          # syntax error in commandline

# Custom events
emit my_event arg1 arg2

function listener --on-event my_event
    echo "received: $argv"
end

# fish_command_not_found example — auto-suggest a package:
function fish_command_not_found
    set -l cmd $argv[1]
    echo "fish: '$cmd' not found.  Did you mean to install it?"
end

# fish_preexec is great for measuring time, logging:
function _log --on-event fish_preexec
    echo (date +%s)" $argv" >> ~/.cmd.log
end

# Events fire ONLY in the parent shell (not in subshells like (cmd)).
```

## Aliases

```bash
# `alias` in fish creates a FUNCTION (not a textual alias like bash).
alias g 'git'
alias gst 'git status'
alias rgrep 'rg --hidden --no-ignore'

# Persist (auto-save to functions dir)
alias --save g 'git'                    # writes ~/.config/fish/functions/g.fish

# Show / list / erase
alias                                   # all aliases (lists functions that look like aliases)
alias g                                 # show specific
functions g                             # show generated function body
functions -e g                          # erase

# Limitation: aliases run as functions, so:
# - You CAN pass arguments: alias g 'git'; g status -> git status
# - You CANNOT do positional rewrites without a real function:
#     bash:  alias foo='cmd $1'
#     fish:  must define a real function with $argv

# vs Abbreviations (NEXT section):
# alias  : invisible expansion at runtime, history shows "g"
# abbr   : expand-on-space, history shows "git" (the full command)
```

## Abbreviations

```bash
# abbr expands when you press SPACE or ENTER, leaving the FULL command
# in the buffer and history.  Best for muscle-memory training.

abbr -a g git                           # `g` + space -> `git `
abbr -a gst 'git status'
abbr -a gco 'git checkout'
abbr -a gcm 'git commit -m'
abbr -a gp  'git push'
abbr -a k   kubectl
abbr -a dc  'docker compose'

# Abbreviations are stored UNIVERSALLY by default — they persist across sessions
# without any save command.  No funcsave needed.

# List / show / erase
abbr                                    # all
abbr --show                             # with expansions
abbr --list                             # names only
abbr -e g                               # erase

# Position-dependent abbreviations (3.6+)
abbr -a --position command l 'ls -la'   # only at start of command
abbr -a --position anywhere ! 'sudo'    # anywhere on commandline

# Regex abbreviations (3.6+)
abbr -a --regex '^!!\$' --function last_history
function last_history
    echo $history[1]
end

# Function-driven expansion (3.6+) — dynamic
abbr -a gco --function gco_branches --regex 'gco'
function gco_branches
    set -l b (git branch --sort=-committerdate | string trim | head -1)
    echo "git checkout $b"
end

# Trigger explicitly (when typing a script)
echo (abbr --query g)                   # check exists

# Move from alias to abbr — generally preferred:
# alias g 'git'   # invisible
# abbr -a g git   # visible, history-friendly
```

## Conditional Execution

```bash
# fish uses `and` / `or` keywords (after `;` or newline).
# These replace bash's && and ||.

# bash:                fish:
# cmd1 && cmd2         cmd1; and cmd2
# cmd1 || cmd2         cmd1; or cmd2
# cmd1 && cmd2 || cmd3 cmd1; and cmd2; or cmd3

# Practical examples
mkdir build; and cd build               # only cd if mkdir succeeded
test -f .env; or echo "no .env"
git pull; and make; and ./run

# Beware the trailing-or trap (bash users)
# bash:  cmd1 && cmd2 || cmd3   # cmd3 runs if cmd1 OR cmd2 fails
# fish:  same logic in fish     # cmd3 runs if `cmd1; and cmd2` is false
test -f f; and echo yes; or echo no

# `not` inverts exit status
not test -f file; and echo "missing"

# Group conditions
if begin; test -f a; or test -f b; end
    echo "a or b exists"
end

# Pipefail-equivalent: $pipestatus is a list of exit codes
false | true
echo $status                            # 0 (last cmd in pipe)
echo $pipestatus                        # 1 0 (each cmd's exit)
```

## Pipes & Redirection

```bash
# Pipe stdout
cmd1 | cmd2

# Background
cmd &

# Stderr in fish — old syntax was `^` (DEPRECATED, removed-ish).
# Modern fish (3.x) uses bash-like 2> / &> / 2>&1.

# Redirect stdout to file
cmd > out.txt                           # overwrite
cmd >> out.txt                          # append

# Redirect stderr
cmd 2> err.txt
cmd 2>> err.txt

# Combine stdout+stderr
cmd > all.txt 2>&1
cmd &> all.txt                          # bash-extension shorthand (fish supports)

# Redirect stdin
cmd < in.txt

# Discard
cmd > /dev/null
cmd 2> /dev/null
cmd &> /dev/null
cmd > /dev/null 2>&1

# Send to /dev/stderr / /dev/stdout
echo "warn" > /dev/stderr
echo "msg"  > /dev/stdout

# Pipe stderr (use 2>| or redirect-then-pipe)
cmd 2>| grep error                      # pipes stderr only
cmd |& grep .                           # pipes BOTH (fish 3.x)
cmd 2>&1 | grep err                     # classic combined pipe

# Heredoc (none in fish — use printf or read)
# bash:
#   cat <<EOF
#   hello
#   EOF
# fish:
printf '%s\n' 'line1' 'line2' 'line3' > out
echo 'multi
line
string' > out

# Here-string (none in fish; use echo|cmd or printf|cmd)
# bash:  cmd <<< "hello"
# fish:  echo hello | cmd

# Open extra FDs (advanced)
exec 3< file.txt
read -l line <&3

# Tee
cmd | tee out.txt
cmd | tee -a out.txt                    # append
cmd 2>&1 | tee both.txt
```

## Process Substitution

```bash
# fish has NO <(cmd) bash-style process substitution.
# Use `psub` to create a temp file holding the command's output.

# diff two command outputs
diff (sort a.txt | psub) (sort b.txt | psub)

# Treat a command's output as a file argument
vimdiff (cat /etc/hosts | psub) (cat /etc/services | psub)

# psub options
cmd | psub                              # default: temp file, deleted on exit
cmd | psub -F                           # use a fifo (still file-like)
cmd | psub -s .json                     # suffix the temp file (some tools care)

# Equivalent of >(cmd)?  Limited.  Use a fifo:
mkfifo /tmp/myfifo
cmd_producer > /tmp/myfifo &
cmd_consumer < /tmp/myfifo

# Or use `tee` with a sub-shell:
cmd | tee (other_cmd | psub) > /dev/null

# Common pattern: sort + uniq diff
diff (cat a | sort -u | psub) (cat b | sort -u | psub)
```

## Subprocess

```bash
# Command substitution — capture output of a command into a variable / argument.
# fish syntax: (cmd)   NOT $(cmd) and NOT `cmd` (backticks REMOVED in 3.x).

set today (date +%Y-%m-%d)
echo "today is "(date)
ls (which python)                       # pass output as argument

# Nesting
echo (string upper (whoami))

# Capture stderr too
set out (cmd 2>&1)

# Lines split into list elements (preserves whitespace inside lines)
set lines (cat /etc/hosts)              # one element per line
count $lines

# To preserve newlines in one string, use string collect
set whole (cat file.txt | string collect)
echo "$whole"

# To NOT split on newlines (e.g., a binary blob), use string collect
set blob (cmd | string collect)

# Direct redirection writes to file as usual
cmd > file.txt

# Run cmd silently, only check exit status
if cmd >/dev/null 2>&1
    echo ok
end

# `eval` runs a string as fish code
set code 'echo hi'
eval $code

# Run cmd in a subshell context (rare; use parens)
echo before
fish -c 'cd /tmp; pwd'                  # spawn child fish
echo after                              # cwd unchanged in parent
```

## Reading Input

```bash
# Basic prompt
read line                               # waits for stdin, sets $line
echo "got: $line"

# With prompt text
read -P 'name: ' name
read --prompt-str 'name: ' name         # same thing

# Custom prompt function
function _myprompt; echo -n '> '; end
read --prompt _myprompt name

# Silent (passwords)
read -s --prompt-str 'password: ' pw

# Local scope (good in functions/scripts)
read -l -P 'enter: ' answer

# Read with timeout (seconds, accepts decimals)
read -t 5 -P 'quick: ' x; or echo "timed out"

# Read N characters (no Enter needed, -n)
read -n 1 -P 'press a key: ' k

# Read into a list (split on $IFS, defaults to whitespace)
read -a values
# user types: a b c    -> $values is list of 3 elements

# Read with explicit delimiter
read -d : a b c < /etc/passwd           # split first line on `:` into a,b,c

# Read from a file (loop)
while read -l line
    echo $line
end < /etc/hosts

# Read full file into a single var preserving newlines
set contents (cat file.txt | string collect)

# Confirmation helper
read -P 'continue? [y/N] ' confirm
if test "$confirm" = y
    echo proceeding
end
```

## Job Control

```bash
# Background a command
sleep 30 &
sleep 60 &

# List jobs (with PIDs and status)
jobs
jobs -p                                 # PIDs only
jobs -l                                 # long format

# Bring to foreground
fg                                      # most recent
fg %1                                   # by job number
fg %sleep                               # by name prefix

# Send to background (after Ctrl-Z)
bg                                      # most recent stopped
bg %2

# Suspend foreground job
# Press Ctrl-Z

# Kill jobs
kill %1                                 # job 1
kill -9 %2                              # SIGKILL job 2
kill -TERM %sleep
kill $last_pid                          # last backgrounded process

# Disown — detach so job survives shell exit
disown                                  # last job
disown %1
disown -a                               # all

# Wait for backgrounded jobs
sleep 5 &
sleep 10 &
wait                                    # all jobs
wait $last_pid                          # specific PID
wait %1                                 # specific job
wait -n                                 # any one (3.5+)

# Run job in shell session group (avoid signal forwarding)
nohup long_running > out.log 2>&1 &
disown

# Capture exit code of a backgrounded job
sleep 1 &
set bg_pid $last_pid
wait $bg_pid
echo $status

# Job-related variables
echo $last_pid                          # PID of last & job
echo $fish_pid                          # PID of this fish
```

## Argument Parsing — argparse builtin

```bash
# argparse is the canonical robust CLI argument parser for fish functions.

function deploy
    # Spec: name short_long  (multiple specs separated by space)
    # Trailing punctuation:
    #   no  punct = boolean flag
    #   =       = required value
    #   =?      = optional value
    #   =+      = repeatable (collects list)
    argparse 'h/help' 'v/verbose' 'i/input=' 'e/env=?' 'x/exclude=+' -- $argv
    or return                           # bail on parse error (non-zero exit)

    if set -q _flag_help
        echo "usage: deploy [-v] [-i INPUT] [-e [ENV]] [-x EXC]... target"
        return 0
    end

    set -q _flag_verbose; and echo "verbose mode"
    set -q _flag_input;   and echo "input: $_flag_input"
    set -q _flag_env;     and echo "env: $_flag_env"
    set -q _flag_exclude; and echo "exclude: $_flag_exclude"

    set -l targets $argv
    echo "targets: $targets"
end

deploy -v -i config.yaml -x foo -x bar prod-1 prod-2
# verbose mode
# input: config.yaml
# exclude: foo bar
# targets: prod-1 prod-2

# Spec format details
# 'h/help'           short -h, long --help, no value
# 'i/input='         short -i, long --input, REQUIRED value
# 'env=?'            long --env only, OPTIONAL value
# 'x/exclude=+'      repeatable, becomes list in $_flag_exclude
# '#-val'            no short, no long, custom — see docs

# Validation: argparse exits non-zero on bad args, prints helpful error to stderr.

# Stop parsing at --
function f
    argparse 'v/verbose' -- $argv
    or return
    # everything after -- is in $argv
end

# Pass through unknown flags
function wrap
    argparse --ignore-unknown 'v/verbose' -- $argv
    or return
    command real_cmd $argv              # unknown flags preserved
end

# Validate value of a flag
function f
    argparse 'l/level=' -- $argv
    or return
    if set -q _flag_level
        contains -- $_flag_level debug info warn error
        or begin; echo bad level; return 2; end
    end
end

# Min/max positional args
argparse --min-args=1 --max-args=3 -- $argv

# Function name in errors
argparse --name=mytool -- $argv
```

## Color and Prompt

```bash
# set_color — emit terminal color escapes
set_color red                           # foreground red
set_color -b yellow                     # background yellow
set_color --bold blue                   # bold + blue
set_color --italics                     # italic (terminal-dependent)
set_color --underline                   # underline
set_color --dim                         # dim
set_color --reverse                     # reverse video
set_color normal                        # reset

# Named colors: red green yellow blue magenta cyan white black
# Bright variants: brred brblue brblack ...
# Hex / rgb (24-bit, terminal-dependent)
set_color FF8800
set_color -b 222244

# Use in prompt
function fish_prompt
    set_color cyan
    echo -n (whoami)
    set_color normal
    echo -n '@'
    set_color brblack
    echo -n (prompt_hostname)
    set_color normal
    echo -n ':'
    set_color brblue
    echo -n (prompt_pwd)
    set_color normal
    echo -n '> '
end

# fish_right_prompt — right-aligned prompt segment
function fish_right_prompt
    set_color brblack
    date +%H:%M
    set_color normal
end

# fish_mode_prompt — vi-mode indicator (when vi keybindings active)
function fish_mode_prompt
    switch $fish_bind_mode
        case default; echo -n '[N] '
        case insert;  echo -n '[I] '
        case visual;  echo -n '[V] '
    end
end

# Disable greeting
set -U fish_greeting ""

# Greeting variable
set -U fish_greeting "welcome to fish"

# Color theme variables — see ALL with: set | grep ^fish_color
echo $fish_color_command                # default command color
echo $fish_color_param                  # parameter color
echo $fish_color_quote                  # quoted strings
echo $fish_color_redirection            # > | <
echo $fish_color_error                  # syntax error
echo $fish_color_autosuggestion         # gray suggestion text
echo $fish_color_search_match           # history search highlight

# Set colors universally
set -U fish_color_command brblue
set -U fish_color_error brred
set -U fish_color_autosuggestion '777'  # gray hex

# Pager colors (tab-completion menu)
set -U fish_pager_color_prefix      brwhite --bold
set -U fish_pager_color_completion  brblack
set -U fish_pager_color_description yellow
set -U fish_pager_color_progress    cyan

# Theme picker
fish_config theme show                  # list themes
fish_config theme choose 'Coolbeans'    # apply
fish_config                             # browser-based UI
```

## Universal Settings

```bash
# Common universal settings to set once, forever.

# Greeting
set -U fish_greeting ""

# PATH extension (preferred way; deduplicated and persistent)
set -U fish_user_paths $HOME/.local/bin $HOME/bin /usr/local/sbin
fish_add_path $HOME/.cargo/bin          # appends if missing
fish_add_path -a /opt/bin               # append (default is prepend)
fish_add_path -p $HOME/.dev             # prepend explicitly
fish_add_path --move $HOME/.bin         # move to front if already present

# History size (default 256k)
set -U fish_history_max 100000

# Editor
set -Ux EDITOR nvim
set -Ux VISUAL nvim
set -Ux PAGER less

# Less options
set -Ux LESS '-R -F -X --mouse'

# Locale
set -Ux LANG en_US.UTF-8
set -Ux LC_ALL en_US.UTF-8

# Color theme (built-in themes in fish_config)
set -U fish_color_command brblue --bold
set -U fish_color_error brred
set -U fish_color_autosuggestion 777

# Vi keybindings as default
fish_vi_key_bindings
# To revert:
fish_default_key_bindings

# Third-party prompts that integrate well:
#   tide      — `fisher install IlanCosman/tide@v6` (async, fast, like p10k)
#   starship  — language-agnostic; install via cargo, then in config.fish:
#               starship init fish | source
#   bobthefish — `fisher install oh-my-fish/theme-bobthefish`
#   pure      — `fisher install rafaelrinaldi/pure`
```

## History

```bash
# History file: ~/.local/share/fish/fish_history (YAML-ish)

# Up arrow / Ctrl-P  — substring search backward
# Down arrow / Ctrl-N — substring search forward
# Type a prefix then Up — only history lines matching that prefix

# Search in pager
history search 'git push'               # all matches
history search --contains 'docker'
history search --prefix 'cd '
history search --exact 'ls'

# Show all
history                                 # most recent first
history --reverse                       # oldest first
history --max=20

# Delete an entry
history delete 'rm -rf /'               # interactive prompt
history delete --exact --case-sensitive 'rm -rf /'

# Clear history
history clear                           # confirm
history clear-session                   # only this session's entries

# Merge from other sessions (multi-shell sync)
history merge

# Export
history > ~/cmd-archive.txt

# fzf integration (best-in-class)
fisher install PatrickF1/fzf.fish
# Then: Ctrl-R for history search, Ctrl-Alt-F for files, Ctrl-Alt-L for git log

# Skip a command from history (lead with a space — like bash HISTCONTROL=ignorespace)
# Note: fish does NOT have ignorespace by default.  Workaround:
function _no_history --on-event fish_preexec
    if string match -q ' *' -- "$argv"
        builtin history delete --exact -- "$argv"
    end
end
```

## Autocomplete & Completions

```bash
# fish auto-generates completions from man pages.  Regenerate:
fish_update_completions

# Custom completions live in ~/.config/fish/completions/<cmd>.fish

# complete syntax
complete -c CMD [OPTIONS]

# Common option flags
# -c CMD    command to complete
# -s X      short option -X
# -l XYZ    long option --XYZ
# -o XYZ    old-style -XYZ (one dash, multi-char)
# -d "DESC" description shown in pager
# -a "..."  list of arguments to suggest
# -r        requires an argument
# -f        DON'T suggest filenames
# -F        DO suggest filenames
# -x        equivalent to -r -f
# -n COND   only suggest when COND succeeds
# -k        keep suggestion list order (no sort)

# Examples
complete -c mytool -s h -l help -d 'show help'
complete -c mytool -s o -l output -r -F -d 'output file'
complete -c mytool -l verbose -d 'verbose output'
complete -c mytool -a 'start stop reload' -f

# Subcommands
complete -c mytool -n __fish_use_subcommand -a start -d 'start service'
complete -c mytool -n __fish_use_subcommand -a stop  -d 'stop service'

# Conditional flag (only for `start`)
complete -c mytool -n '__fish_seen_subcommand_from start' \
    -s d -l daemon -d 'run as daemon'

# Dynamic — call a command for suggestions
complete -c kubectl -n __fish_use_subcommand \
    -a "(kubectl api-resources --no-headers -o name)"

complete -c git -n '__fish_git_using_command checkout' \
    -a '(git branch | string trim)'

# Filter helpers fish provides
__fish_use_subcommand                   # true if no subcommand chosen yet
__fish_seen_subcommand_from <names>     # true if a subcommand is in argv
__fish_complete_directories             # directories
__fish_complete_path                    # full path
__fish_complete_command                 # commands in $PATH
__fish_complete_pids                    # running PIDs
__fish_complete_user_at_hosts           # SSH-style user@host
__fish_complete_suffix .png .jpg        # files with given suffix
__fish_complete_groups                  # group names
__fish_complete_users                   # user names

# Test a completion in-place
complete -c mytool                      # show all rules
complete -c mytool -e                   # erase all
complete --do-complete='git checko'     # show what fish would suggest

# Common workflow: write your completions, drop in completions/, done.
# Reloads automatically when file changes.
```

## Tab and Suggestions

```bash
# fish's killer interactive features:

# Autosuggestions  (gray text appearing as you type, drawn from history + completions)
# - Right Arrow  / Ctrl-F        : accept entire suggestion
# - Alt-Right    / Alt-F         : accept ONE word
# - Esc / Ctrl-G                 : dismiss suggestion

# Tab completion
# - Tab                          : complete; cycle through choices
# - Shift-Tab                    : reverse cycle
# - Tab + Tab                    : show pager menu
# - In pager: Tab/Shift-Tab navigate, Enter selects, Esc cancels

# History expansion / search
# - Up Arrow                     : prefix search backward
# - Down Arrow                   : prefix search forward
# - Alt-Up / Alt-Down            : token-based history (next arg)

# Multi-line editing
# - Alt-Enter                    : insert literal newline (don't run)
# - Ctrl-X Ctrl-E (or Alt-E)     : edit current line in $EDITOR
# - Alt-V                        : same as Alt-E

# Smart cursor movement
# - Alt-B / Alt-F                : back/forward one word
# - Ctrl-A / Ctrl-E              : start/end of line
# - Alt-W                        : "what is this?" — show docs for the cmd
# - Alt-L                        : list directory contents

# Quick history / sudo
# - Alt-S                        : prepend `sudo ` to commandline
# - !!                           : NOT supported by default; abbr it:
#                                    abbr -a !! --regex '^!!\$' --function lhist

# Token deletion / replace
# - Alt-D                        : delete next word
# - Ctrl-W                       : delete previous word
# - Ctrl-U                       : delete to start of line
# - Ctrl-K                       : delete to end of line

# Help in real time
# - Press F1 (or Alt-H) on a command -> opens man page

# Show what abbr would expand to BEFORE running
# (just type abbr + space; fish replaces it inline)

# Interactive search
# - Ctrl-R (with fzf.fish plugin) : fuzzy history
# - Ctrl-Alt-F                    : fuzzy file finder (fzf.fish)
# - Ctrl-Alt-L                    : fuzzy git log (fzf.fish)
```

## Plugins / Managers

```bash
# Plugin managers (pick one):

# === fisher (most popular, simplest) ===
curl -sL https://raw.githubusercontent.com/jorgebucaran/fisher/main/functions/fisher.fish | source
fisher install jorgebucaran/fisher

fisher install PatrickF1/fzf.fish       # fzf integration
fisher install IlanCosman/tide@v6       # async prompt (fastest)
fisher install jethrokuan/z             # z directory jumping
fisher install meaningful-ooo/sponge    # remove failed commands from history
fisher install jorgebucaran/autopair.fish # auto-close brackets/quotes
fisher install franciscolourenco/done   # desktop notify long jobs
fisher install gazorby/fish-abbreviation-tips

fisher list                             # what's installed
fisher update                           # all
fisher update <plugin>                  # one
fisher remove <plugin>

# === oh-my-fish (omf) — older, more themes ===
curl https://raw.githubusercontent.com/oh-my-fish/oh-my-fish/master/bin/install | fish
omf install bobthefish                  # popular theme
omf install agnoster
omf list
omf update
omf remove bobthefish

# === fundle — git-tracked, no-curl install ===
curl -sfL https://git.io/fundle-install | source

# Recommended prompt third-parties
# tide     — `fisher install IlanCosman/tide@v6`
# starship — universal prompt; in config.fish: `starship init fish | source`
# pure     — minimalist; `fisher install rafaelrinaldi/pure`
# bobthefish — git-aware; `omf install bobthefish` or fisher equivalent

# Common plugins to consider:
# autopair          — auto-close () [] {} '' ""
# z                 — `z partial` jumps to most-frecent matching dir
# fzf.fish          — Ctrl-R fuzzy history etc.
# nvm.fish          — Node version manager fish-native
# pyenv             — drop the bash hook into config.fish
# kubectl-completions — usually shipped by kubectl itself

# Audit a plugin BEFORE install — read its source.
# Universal var: plugins often install completions and conf.d files.
```

## Bind keys

```bash
# bind — bind a key sequence to a command.

# Inspect current bindings
bind                                    # all bindings (default mode)
bind --all                              # include all modes
bind --key-names                        # list special key names
bind \cr                                # show what Ctrl-R does

# Common keybind syntax
# \c<x>     Control-x
# \e<x>     Alt-x (escape sequence)
# \cm       Enter (Ctrl-M)
# \ck       Ctrl-K
# \e\\      Alt-Backslash
# \r        carriage return
# Use bind --key-names for canonical names like 'up', 'down', 'home', 'end', 'f1'..'f12'.

# Examples
bind \cr 'history-search-backward'      # Ctrl-R
bind \cs 'commandline -i sudo\ '        # Ctrl-S inserts "sudo "
bind \el 'clear; commandline -f repaint' # Alt-L clears screen
bind \e. 'commandline -i (history -1 | string split " " | tail -1)'

# Insert text
bind \cv 'commandline -i (xclip -o)'    # Ctrl-V paste

# Vi mode
fish_vi_key_bindings                    # switch
fish_default_key_bindings               # back to defaults
fish_hybrid_key_bindings                # vi + emacs in different modes

# In vi mode, bindings are per-mode:
bind -M default <key> <cmd>
bind -M insert  <key> <cmd>
bind -M visual  <key> <cmd>

# Special editor functions you can bind to (commandline -f X)
# accept-autosuggestion
# backward-char / forward-char
# backward-word / forward-word
# backward-kill-word / forward-kill-word
# beginning-of-line / end-of-line
# kill-line / kill-whole-line
# yank
# expand-abbr
# repaint
# history-search-backward / history-search-forward
# undo / redo
# delete-or-exit
# (See `bind --function-names` for the full list.)

# Persist bindings — put them in fish_user_key_bindings function:
function fish_user_key_bindings
    bind \cr history-search-backward
    bind \e. token-history-token
end
funcsave fish_user_key_bindings
```

## Math and Data Manipulation

```bash
# math (see Arithmetic) — arithmetic
# string (see Strings) — text processing
# count — list length
count $list                             # number of elements
count *.md                              # number of matching files
count (string split , 'a,b,c')          # 3

# contains — list membership
contains apple $fruits                  # exit 0 if member
contains -i apple $fruits               # print 1-based index, or 0
contains -- -h $argv                    # use -- to handle leading dashes

# type — what kind of name is this?
type ls                                 # ls is /bin/ls
type cd                                 # cd is a builtin
type myfunc                             # myfunc is a function with definition
type -t ls                              # -t prints just the type
type -p ls                              # path only (like which)
type -q ls                              # quiet — exit code only

# command — bypass functions/aliases, run external command
command ls                              # /bin/ls regardless of alias
command -v git                          # path to git (POSIX-ish)
command -s git                          # source path
command -q git                          # exit 0 if exists

# builtin — bypass functions, run shell builtin
builtin cd /tmp                         # run real cd, even if `cd` is a function

# functions / functions -e — manage functions
functions --names                       # list function names
functions --all                         # include hidden underscore funcs

# set utilities
set -q VAR                              # exists?
set -e VAR                              # erase
set -l name val                         # local
set -U name val                         # universal
set -S name                             # show all scopes for name

# random — pseudo-random integer
random                                  # 0 .. 32767
random 1 100                            # 1..100 inclusive
random 1 2 100                          # 1, 3, 5, ... 99 (start, step, end)
random choice apple banana cherry       # pick one element
random seed 42                          # seed PRNG for reproducibility

# date arithmetic — use `date` and `math`
set now (date +%s)
set later (math $now + 3600)            # one hour from now
date -r $later                          # human-readable (BSD date)
```

## Common Builtins

```bash
# Quick reference of fish builtins (run `builtin -n` to list all):

# Output
echo "text"                             # print + newline (no escapes by default)
echo -n "no newline"                    # -n suppress newline
echo -e "with\nescapes"                 # -e enable escapes
printf '%s\n' a b c                     # printf supports format strings + escapes
printf '%-10s %d\n' name 42             # column align

# Variables
set NAME val                            # set variable
set -e NAME                             # erase
set -l NAME val                         # local
set -g NAME val                         # global
set -U NAME val                         # universal
set -x NAME val                         # exported
set -q NAME                             # query

# Math / strings (already covered)
math '1 + 2'
string upper hi

# Lists
count $list
contains apple $fruits

# Control
if; else if; else; end
switch; case; case '*'; end
for; in; end
while; end
break / continue
return [N]
exit [N]

# Functions
function foo; ...; end
end                                     # closes blocks

# Sourcing / eval
source ~/.config/fish/config.fish       # `.`  is also `source`
eval (cmd)                              # run output as fish code
exec fish                               # replace shell

# Status / signals
status is-interactive                   # are we interactive?
status is-login                         # login shell?
status is-block                         # in a block?
status current-function                 # current function name
status current-line-number              # script line
status filename                         # script name
status fish-path                        # path to fish

# Type / introspection
type ls
command -v git
builtin cd
functions
abbr
alias
bind

# Job control
jobs / fg / bg / wait / disown / kill

# I/O / FS (these are external on most systems but fish uses them constantly)
read / printf / echo

# Random / time
random
time cmd                                # measure command time
```

## Common Errors

```bash
# 1. "fish: Unknown command: foo"
#    -> command not in $PATH or function not defined.
#    Fix: install it, or check `type foo`, or update fish_user_paths.

# 2. "Missing end to balance this <function|if|while|for|switch|begin>"
#    -> you forgot `end`.
function broken
    echo hi
# fish: Missing end to balance ...
# Fix:
function fixed
    echo hi
end

# 3. "fish: Variables cannot be bracketed."
#    -> you wrote ${var} (curly braces around a var name).
#    Broken:
#       echo ${name}_suffix             # error
#    Fixed:
echo {$name}_suffix                     # uses brace-list  (1 element if scalar)
echo "$name"_suffix                     # concatenation
echo $name"_suffix"                     # also fine

# 4. "Variables cannot be a single ',' or '_' character"
#    -> happens when you accidentally write `set , value`
#       or use illegal var name.  Fix: use [a-zA-Z_][a-zA-Z0-9_]* names.

# 5. "Expected an argument like 'NAME=VALUE' for ..."
#    -> wrong `set` syntax.  fish uses spaces, NOT `=`:
#    Broken:
#       NAME=value cmd                  # works (one-shot env)
#       set NAME=value                  # ERROR
#    Fixed:
#       set NAME value
#       set -x NAME value cmd           # exported

# 6. "fish: Unsupported use of '='. In fish, please use 'set NAME VALUE'."
#    -> assigning with =.  See above.

# 7. "fish: $? is not the exit status. ..."
#    -> use $status, not $?
echo $status

# 8. "No matches for wildcard '...'"
#    -> default behavior on no glob match.
#    Workaround:
set files *.zzz 2>/dev/null
or set files

# 9. "fish: command substitutions not allowed"
#    -> using $(cmd) in some context where fish uses (cmd).
#    Fix: $(cmd) is a fish 3.4+ alias for (cmd) — older versions reject it.

# 10. "fish: Unknown option '-z'"
#     -> `test -z $maybe_empty` without quoting may pass zero args.
#     Wait — fish has no word-split, so this rarely happens.
#     If it does: `test -z "$x"` (always quote in test for clarity).

# 11. "fish: Cannot find function 'foo'"
#     -> autoload file misnamed.  Must be ~/.config/fish/functions/foo.fish

# 12. "Could not change to ..."   (cd error)
#     -> dir doesn't exist / no perms.  Check.

# 13. "Universal variable file is corrupt"
#     -> ~/.config/fish/fish_variables damaged.
#     Fix: back up, delete, restart fish.  Universals re-created.

# 14. "fish: Failed to read history file"
#     -> ~/.local/share/fish/fish_history corrupt.
#     Fix: history clear; or rm + restart.

# 15. "the function 'X' calls itself"
#     -> infinite recursion.  Wrap a command properly:
#     Broken:
#        function ls; ls -la $argv; end           # recurses!
#     Fixed:
function ls; command ls -la $argv; end           # `command` skips this function
```

## Common Gotchas

```bash
# 1. NO $((...)) arithmetic
# Broken:
#   echo $((2 + 3))
# Fixed:
echo (math '2 + 3')

# 2. NO $? for exit status
# Broken:
#   echo $?
# Fixed:
echo $status

# 3. NO <(cmd) process substitution
# Broken:
#   diff <(cmd1) <(cmd2)
# Fixed:
diff (cmd1 | psub) (cmd2 | psub)

# 4. NO [[ ... ]] double brackets
# Broken:
#   [[ "$x" == "y" ]] && echo yes
# Fixed:
test "$x" = y; and echo yes
string match -q y "$x"; and echo yes

# 5. NO POSIX `if [ ... ]` style preferred over `test`
# Both work, but `test` is fish-idiomatic.
# Works:
if [ -f file ]; echo y; end             # OK but space inside [ ] required
# Preferred:
if test -f file; echo y; end

# 6. NO $(...) command substitution syntax (fish 3.3-)
# fish 3.4+ accepts $(cmd) as alias for (cmd).
# Older fish:
# Broken: $(date)
# Fixed:
echo (date)

# 7. NO 0-indexed lists
# Broken:
#   echo $list[0]                       # error: array index out of bounds
# Fixed:
echo $list[1]                           # first element

# 8. Quoting "$var" doesn't change anything (no word-split anyway)
set msg "hello world"
ls $msg                                 # 1 arg
ls "$msg"                               # 1 arg — same behavior

# 9. List vars expand to multiple args even when "quoted"
set files a.txt b.txt
ls $files                               # 2 args
ls "$files"                             # STILL 2 args (fish does NOT join lists in quotes!)
# To force-join into 1 arg, use string join:
ls (string join ' ' $files)             # 1 arg "a.txt b.txt"

# 10. (cmd) splits on NEWLINES (each line = list element)
set files (ls)
# If filenames contain newlines this breaks.  Use string collect or NUL-split:
set files (find . -print0 | string split0)

# 11. && and || are NOT supported as statement separators
# Broken:
#   cmd1 && cmd2
# Fixed:
cmd1; and cmd2
cmd1; or cmd2

# 12. NO `export` keyword
# Broken:
#   export EDITOR=nvim
# Fixed:
set -x EDITOR nvim
set -Ux EDITOR nvim                     # universal + exported

# 13. NO `local` keyword
# Broken:
#   local x=1
# Fixed:
set -l x 1

# 14. NO `if-else-fi` — use `end`
# Broken (bash):
#   if [ x ]; then echo y; fi
# Fixed (fish):
if test x; echo y; end

# 15. for (( i=0; i<10; i++ )) — NO C-style for
# Fixed:
for i in (seq 0 9); echo $i; end

# 16. NO `function foo() {}` JS/C-style
# Fixed:
function foo; ...; end

# 17. NO heredoc <<EOF
# Fixed:
echo 'multi
line
string'
printf '%s\n' line1 line2

# 18. NO here-string <<<
# Fixed:
echo "input" | cmd

# 19. NO `set -e` (errexit) shell option
# fish doesn't have shopt-style global error-exit.
# Use `or return` or `; or exit` after risky commands:
risky_cmd; or return 1
```

## Migrating from Bash/Zsh

```bash
# fish is NOT a POSIX shell.  Bash scripts WILL NOT run as fish scripts.

# Strategies:

# 1. Keep bash for scripts; use fish interactively.
#    .bashrc unchanged, login shell = fish, /bin/bash for `#!/usr/bin/env bash` scripts.
#    This is the most common setup.

# 2. Source bash profile via `bass` (a fisher plugin)
fisher install edc/bass
bass source ~/.bashrc                   # runs in bash, exports vars to fish
bass source /opt/something/setup.sh

# 3. Convert scripts to fish.
# Common rewrites:

# bash                              fish
# --------------------------------  --------------------------------
# export X=y                        set -x X y
# X=y cmd                           env X=y cmd  (or X=y cmd works too in fish)
# arr=(a b c)                       set arr a b c
# arr[0]                            $arr[1]                    (1-indexed!)
# ${arr[@]}                         $arr
# ${#arr[@]}                        (count $arr)
# $((x + 1))                        (math $x + 1)
# $?                                $status
# $$                                $fish_pid
# $!                                $last_pid
# `cmd`                             (cmd)
# $(cmd)                            (cmd) or $(cmd) in 3.4+
# <(cmd)                            (cmd | psub)
# >(cmd)                            mkfifo workaround
# &&                                ; and
# ||                                ; or
# read -p "?" var                   read -P "?" var
# read -s var                       read -s var (same)
# [ -f f ] && ...                   test -f f; and ...
# [[ "$a" == "$b" ]]                test "$a" = "$b"
# [[ "$a" =~ regex ]]               string match -rq 'regex' "$a"
# echo -e "a\nb"                    printf 'a\nb\n'
# x="abc"; ${x%c}                   string sub -e -2 -- $x
# x="abc"; ${x:0:2}                 string sub -l 2 -- $x
# IFS=, read -ra arr <<< "a,b,c"    set arr (string split , a,b,c)

# 4. Run a single bash one-liner from fish:
bash -c 'for i in {1..5}; do echo $i; done'

# 5. Use $POSH-/zsh-style auto-detection — NO, fish is its own thing.

# Keep your interactive aliases / functions in fish; keep your DEPLOY scripts in bash.

# Don't try to make fish run #!/bin/sh scripts.  Just don't.
```

## Performance

```bash
# fish is fast (Rust rewrite in 4.0; C++ in 3.x), but config can slow startup.

# Measure startup time
time fish -i -c exit                    # ~30-100ms is typical
fish --profile-startup ~/fish-startup.log -i -c exit
sort -k2 -n ~/fish-startup.log | tail -20   # slowest steps

# Profile a function or script
fish --profile=/tmp/prof -c 'source ~/myscript.fish'
sort -k2 -n /tmp/prof | tail -30

# Common slow culprits
# 1. Heavy work in config.fish that runs every shell.
#    Fix: guard with `status is-interactive` or move to conf.d snippets that
#    only do imports.
# 2. Conda / pyenv / nvm hooks that call subprocesses on every prompt.
#    Fix: lazy-load — load on first use, not on shell start.

# Lazy-load pattern
function pyenv
    functions -e pyenv                  # erase this stub
    source (pyenv init - | psub)        # load real init
    pyenv $argv                         # delegate
end

# Cache expensive prompt data
function fish_prompt
    set -l cached_branch (cat /tmp/git-branch-cache 2>/dev/null)
    # ... use cache, refresh in background
end

# Avoid `command -v X` in hot paths — it forks.  Cache:
if not set -q __have_eza
    set -gx __have_eza 0
    command -q eza; and set -gx __have_eza 1
end
test $__have_eza -eq 1; and alias ls eza

# Batch universal variable updates
# Each `set -U` triggers IPC to other fish sessions.  Group them.

# Use abbr instead of functions when no logic is needed (faster, no fork).

# Don't source giant scripts in fish_prompt.  Render directly.

# tide is async; great default fast prompt.
# starship is fast (Rust) but adds one fork per prompt.
```

## Idioms

```bash
# 1. Universal variables for stable settings
set -Ux EDITOR nvim
set -U fish_greeting ""
fish_add_path $HOME/.local/bin

# 2. abbr for visible muscle-memory training
abbr -a g git
abbr -a k kubectl
abbr -a tf terraform

# 3. functions over aliases when there's logic
function mkcd
    mkdir -p $argv[1]; and cd $argv[1]
end
funcsave mkcd

# 4. argparse for proper CLI
function deploy
    argparse 'h/help' 'v/verbose' 'e/env=' -- $argv; or return
    # ...
end

# 5. status checks
if command -q docker
    abbr -a d docker
end

# 6. status is-interactive guard
if status is-interactive
    abbr -a g git
end

# 7. string command for everything text-y
set domain (string match -rg '@(.+)' alice@example.com)
set words  (string split ' ' "$line")
echo "$line" | string match -rq '^ERROR'

# 8. Use `command` to escape function recursion
function ls; command ls --color=auto $argv; end

# 9. Use `or return` for graceful early-exit in functions
function deploy
    require_tool docker; or return
    docker compose up; or return
end

# 10. `or echo "default"` for fallbacks
set name (whoami); or set name unknown

# 11. Load expensive tools lazily (see Performance)

# 12. One function per file in functions/ for autoload + zero-cost

# 13. Group config into conf.d/ files numbered by load order:
#     ~/.config/fish/conf.d/00-path.fish
#     ~/.config/fish/conf.d/10-editor.fish
#     ~/.config/fish/conf.d/20-abbr.fish

# 14. Guard universals so they only set once (idempotent config):
set -q EDITOR; or set -Ux EDITOR nvim

# 15. Use process substitution via psub for tools needing files:
diff (cmd1 | psub) (cmd2 | psub)

# 16. Validate args early:
function f
    test (count $argv) -ge 1; or begin
        echo "usage: f NAME" >&2; return 2
    end
end

# 17. Prefer `string match -q` over grep -q for single string checks:
string match -q '*.md' $f; and echo markdown
# (no fork compared to `echo $f | grep -q '\.md$'`)

# 18. Stream-friendly: pipe `string` ops, don't loop unless needed.

# 19. Color output with set_color, reset with set_color normal.

# 20. Document functions with --description, fish surfaces it in funced/help.
```

## Tips

```bash
# - `Alt-E` or `Alt-V` opens the current commandline in $EDITOR -- great for long pipelines.
# - `Alt-S` prepends `sudo` (no plugin needed).
# - Press F1 over a command to open its man page inline (`Alt-H` also works).
# - `funced foo` opens function `foo` in $EDITOR; save with `funcsave foo`.
# - `fish_config` opens browser-based theme/prompt picker.
# - `string match -rq` (regex, quiet) is the fast in-memory regex test — beats `grep -q`.
# - `set -S name` shows ALL scopes that have `name` set; great for shadowing debugging.
# - `status is-interactive` and `status is-login` let you guard interactive-only work.
# - Use `command -q tool` (fast, no fork) over `which tool 2>/dev/null` (forks).
# - `fish_add_path` deduplicates and persists; use it instead of editing PATH manually.
# - `set -U fish_history_max 100000` keeps a deep history.
# - `history merge` pulls in commands from other open fish sessions.
# - Color tweaks live in universals: `set -U fish_color_command brblue`.
# - `fish_update_completions` regenerates completions from man pages whenever you install new tools.
# - `abbr --query NAME` checks if an abbreviation exists.
# - `bass source script.sh` runs a bash script and inherits its exported vars.
# - Use `psub` whenever a tool needs a file path but you have a pipeline.
# - Prefer abbr over alias for git/kubectl/etc. -- expanded form ends up in history.
# - Vi mode: `fish_vi_key_bindings`, revert: `fish_default_key_bindings`.
# - `random choice item1 item2 item3` picks one element at random.
# - `printf '%-10s %d\n' name 42` works the same as bash printf.
# - `fish --no-config` starts fish skipping config.fish (debugging).
# - `fish -d 5 -i` runs interactively with verbose debug output level 5.

```

## See Also

- bash, zsh, nushell, shell-scripting, polyglot, regex, awk

## References

- [fish shell Documentation](https://fishshell.com/docs/current/)
- [fish Tutorial](https://fishshell.com/docs/current/tutorial.html)
- [fish Commands Index](https://fishshell.com/docs/current/cmds/)
- [fish-shell on GitHub](https://github.com/fish-shell/fish-shell)
- [awesome-fish (Curated Plugins)](https://github.com/jorgebucaran/awesome-fish)
- [fish FAQ](https://fishshell.com/docs/current/faq.html)
- [fish for bash users](https://fishshell.com/docs/current/fish_for_bash_users.html)
- [fisher Plugin Manager](https://github.com/jorgebucaran/fisher)
- [tide Prompt](https://github.com/IlanCosman/tide)
- [Starship Prompt](https://starship.rs/)
