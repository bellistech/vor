# Nushell (Structured Data Shell)

> A modern cross-platform shell where every command emits typed structured data — lists, records, and tables — instead of raw text streams, bringing database-style pipelines (where, select, sort-by, group-by, reduce) directly to the command line.

## Setup

Nushell is a single Rust binary with no required runtime. Install via your package manager of choice; `nu` runs side-by-side with bash/zsh without replacing them.

```bash
# macOS (Homebrew)
brew install nushell

# Cargo (any platform with Rust toolchain installed)
cargo install nu --locked
cargo install nu_plugin_polars nu_plugin_query nu_plugin_inc

# Windows (Scoop)
scoop install nu

# Windows (winget)
winget install nushell

# Debian / Ubuntu
sudo apt install nushell                      # if available in distro repo
# Or fetch the .deb release from github.com/nushell/nushell/releases

# Arch Linux
sudo pacman -S nushell

# Fedora
sudo dnf copr enable atim/nushell
sudo dnf install nushell

# NixOS
nix-env -iA nixpkgs.nushell

# Verify install
nu --version
# nushell 0.97.1

# Run nu without changing your default shell
nu                                            # start a nu subshell
exit                                          # back to bash/zsh
```

```bash
# Set nu as default login shell (must be in /etc/shells)
which nu                                      # find install path
echo "/opt/homebrew/bin/nu" | sudo tee -a /etc/shells
chsh -s /opt/homebrew/bin/nu

# Confirm
echo $env.SHELL                               # in nu
echo $SHELL                                   # in bash/zsh
```

```bash
# Try without committing — launch from your existing shell
nu -c 'ls | where size > 1mb | sort-by modified -r | first 5'

# Run a script file once
nu /path/to/script.nu

# Persistent ad-hoc usage in bash/zsh
alias nu-here='nu -c'                         # in your bash/zshrc
```

## Why Nushell

Nushell rebuilds the shell around the principle that pipelines should carry typed structured data — lists, records, tables — rather than untyped bytes. Every built-in returns a value with a knowable type that can be filtered, projected, sorted, and aggregated as if it were a SQL relation.

```bash
# In bash/zsh, you parse text every time
ls -l | awk '{print $9, $5}' | sort -k2 -n

# In nu, the shell already understands columns
ls | select name size | sort-by size

# Process listing as a real table you can query
ps | where cpu > 5 | sort-by cpu --reverse | first 10
```

```bash
# Read JSON without piping to jq
http get https://api.github.com/repos/nushell/nushell | get stargazers_count

# Pivot CSV with one command
open data.csv | group-by region | transpose region rows
```

Key wins:

- One binary on Linux, macOS, Windows, FreeBSD — same scripts work everywhere.
- No quoting hell: arguments are real values, not strings to be re-tokenized.
- `where`, `select`, `sort-by`, `group-by`, `each`, `reduce` instead of awk/sed/jq incantations.
- Plugin system in Rust, including `polars` for DataFrame-grade analytics.
- Strong types catch bugs at parse time, not 10 minutes into a long pipeline.

## Config Files

Two startup files live in `~/.config/nushell/` (or `%APPDATA%\nushell\` on Windows). `env.nu` runs first to set up environment variables; `config.nu` runs second to configure the shell, prompt, and key bindings.

```bash
# Discover all the canonical paths via $nu
$nu | columns
# default-config-dir, config-path, env-path, history-path, loginshell-path,
# plugin-path, home-path, data-dir, cache-dir, temp-path, pid, ...

$nu.config-path                               # ~/.config/nushell/config.nu
$nu.env-path                                  # ~/.config/nushell/env.nu
$nu.history-path                              # ~/.config/nushell/history.txt|history.sqlite3
$nu.plugin-path                               # plugin registry file
```

```bash
# Edit configs from inside nu
config nu                                     # opens config.nu in $env.EDITOR
config env                                    # opens env.nu

# Reload the active session after editing
source $nu.config-path
source $nu.env-path
```

```bash
# Minimal env.nu skeleton
$env.EDITOR = "nvim"
$env.PATH = ($env.PATH | split row (char esep) | prepend ['/usr/local/bin' '~/.cargo/bin'])
$env.LANG = "en_US.UTF-8"

# Minimal config.nu skeleton
$env.config = {
    show_banner: false
    edit_mode: emacs
    table: { mode: rounded, index_mode: auto }
    completions: { algorithm: fuzzy, case_sensitive: false }
    history: { max_size: 100_000, file_format: sqlite, sync_on_enter: true }
    cursor_shape: { emacs: line, vi_insert: block, vi_normal: underscore }
}
```

## Hello World, Scripts

Scripts use the `.nu` extension and a shebang invoking `nu`. Pass arguments as ordinary positional/flag parameters declared with `def main`.

```bash
# hello.nu
#!/usr/bin/env nu
print "Hello, world!"
print $"Today is (date now | format date '%Y-%m-%d')"
```

```bash
chmod +x hello.nu
./hello.nu                                    # run via shebang
nu hello.nu                                   # explicit interpreter
nu -c 'print "inline expression"'             # one-shot
```

```bash
# script_with_args.nu
#!/usr/bin/env nu
def main [name: string, --shout (-s)] {
    let greeting = $"Hello, ($name)!"
    if $shout { print ($greeting | str upcase) } else { print $greeting }
}

# Usage
./script_with_args.nu World                   # Hello, World!
./script_with_args.nu World --shout           # HELLO, WORLD!
./script_with_args.nu World -s
```

## Data Types — Primitives

Nu has explicit primitives. `describe` reports the type of any expression.

```bash
42                | describe                  # int
3.14              | describe                  # float
"hello"           | describe                  # string
true              | describe                  # bool
(date now)        | describe                  # datetime
1day              | describe                  # duration
2gib              | describe                  # filesize
1..10             | describe                  # range<int>
null              | describe                  # nothing
0x[ff aa 01]      | describe                  # binary
```

```bash
# Filesize literals — KB/MB/GB use 1000, KiB/MiB/GiB use 1024
1kb           # 1,000 bytes
1kib          # 1,024 bytes
500mb + 200mb # 700 MB
2gib / 4      # 512 MiB

# Duration arithmetic
1day + 12hr                                   # 1day 12hr
(date now) + 1wk                              # next week
((date now) - 2024-01-01T00:00:00+00:00)      # duration since epoch

# Range
1..5         # 1, 2, 3, 4, 5  (inclusive)
1..<5        # 1, 2, 3, 4     (exclusive end)
1..          # open-ended (used with first / take)
```

## Data Types — Structured

```bash
# List — ordered sequence, any element types
[1 2 3]                                       # ints
['a' 'b' 'c']                                 # strings
[1 'two' 3.0 true]                            # heterogeneous

# Record — named fields with values
{name: "Alice", age: 30, role: "dev"}
{nested: {a: 1, b: 2}, items: [1 2 3]}

# Table — list of records sharing the same shape
[[name age]; ["Alice" 30] ["Bob" 25]]
[{name: "A", n: 1} {name: "B", n: 2}]         # equivalent

# Block — code reused as a closure value
let greet = { print "hi" }
do $greet

# Closure — block with parameters
let double = { |x| $x * 2 }
[1 2 3] | each $double                        # [2 4 6]

# Cell-path — typed accessor
let path = $.users.0.name                     # cell-path<users.0.name>
{users: [{name: "Alice"}]} | get $path        # "Alice"
```

## Variables

`let` binds an immutable name; `mut` declares a mutable binding; `const` produces a compile-time constant usable in `use` paths and module loading.

```bash
let count = 5                                 # immutable
let name = "Nu"

mut total = 0                                 # mutable
$total = $total + 1
for n in 1..10 { $total = $total + $n }

const VERSION = "1.0.0"                       # compile-time
const CONFIG_FILE = ($nu.home-path | path join ".myapp.toml")
use $CONFIG_FILE                              # const-only path

# Environment variables — must be on $env
$env.MY_VAR = "value"                         # global env
$env.PATH = ($env.PATH | append "/opt/bin")   # mutate PATH list

# Special variables
$nu                                           # paths, version, OS info
$env                                          # all env vars (record)
$in                                           # piped input inside closures
$it                                           # legacy; prefer $in or named param
```

## Strings

```bash
'literal'                                     # raw, no interpolation, no escapes
"double-quoted"                               # supports \n \t \" escapes
$"interpolated ($name) is ($age)"             # parens evaluated
`raw backtick string`                         # alternative literal

# Common operations via str command family
"Hello, World" | str upcase                   # "HELLO, WORLD"
"Hello, World" | str downcase                 # "hello, world"
"  spaces  "   | str trim                     # "spaces"
"foo,bar,baz"  | str replace "bar" "BAR"      # "foo,BAR,baz"
"foo,bar,baz"  | str replace --all "," ";"    # "foo;bar;baz"
"abcdef"       | str contains "cd"            # true
"abcdef"       | str starts-with "ab"         # true
"abcdef"       | str ends-with "ef"           # true
"abcdef"       | str length                   # 6
"abcdef"       | str substring 2..4           # "cd"

# Splitting
"a,b,c" | split row ","                       # ["a" "b" "c"]
"a:1 b:2 c:3" | split row " "                 # words → list
"a,b,c\n1,2,3" | split column ","             # table

# Parsing structured data from a string
"name=alice age=30" | parse "name={n} age={a}"
# ╭───┬───────┬────╮
# │ # │   n   │ a  │
# │ 0 │ alice │ 30 │
# ╰───┴───────┴────╯
```

## Numbers

Arithmetic operators behave normally; division of two ints returns a float, while `//` is integer division.

```bash
2 + 3                  # 5
10 / 3                 # 3.3333333333333335
10 // 3                # 3
10 mod 3               # 1
2 ** 8                 # 256
-5 | math abs          # 5

# Conversions
"42" | into int        # 42
"3.14" | into float    # 3.14
42 | into string       # "42"
0xff | into string     # "255"
255 | format number --pretty  # "255"

# math command — operates on lists
[1 2 3 4 5] | math sum         # 15
[1 2 3 4 5] | math avg         # 3
[1 2 3 4 5] | math min         # 1
[1 2 3 4 5] | math max         # 5
[1 2 3 4 5] | math median      # 3
[1 2 3 4 5] | math product     # 120
[1 2 3 4 5] | math stddev      # ≈1.41
[1 2 3 4 5] | math variance    # 2

# Trigonometry / power / sqrt
9    | math sqrt        # 3
0    | math sin
1    | math exp
100  | math log 10      # 2
```

## Lists

```bash
let xs = [10 20 30 40 50]

# Indexing
$xs.0                                         # 10
$xs.4                                         # 50
$xs | get 2                                   # 30
$xs | last                                    # 50
$xs | first                                   # 10
$xs | first 3                                 # [10 20 30]

# Length / slice
$xs | length                                  # 5
$xs | take 2                                  # [10 20]
$xs | skip 2                                  # [30 40 50]
$xs | slice 1..3                              # [20 30 40]

# Mutating (returns new list)
$xs | append 60                               # [10 20 30 40 50 60]
$xs | prepend 0                               # [0 10 20 30 40 50]
$xs | reverse                                 # [50 40 30 20 10]
$xs | sort                                    # [10 20 30 40 50]
$xs | sort --reverse                          # [50 40 30 20 10]

# Filter / map / reduce
$xs | where { |x| $x > 25 }                   # [30 40 50]
$xs | each { |x| $x * 2 }                     # [20 40 60 80 100]
$xs | reduce { |it, acc| $acc + $it }         # 150
$xs | reduce --fold 1 { |it, acc| $acc * $it }  # 12_000_000

# Flattening / dedup / chunking
[[1 2] [3 4]] | flatten                       # [1 2 3 4]
[1 1 2 3 3]   | uniq                          # [1 2 3]
[1 1 2 3 3]   | uniq --count
[1 2 3 4 5 6] | chunks 2                      # [[1 2] [3 4] [5 6]]
[1 2 3] | zip [a b c]                         # [[1 a] [2 b] [3 c]]

# Range → list
1..5    | each { |n| $n * $n }                # [1 4 9 16 25]
0..<10  | length                              # 10
```

## Records

```bash
let user = {name: "Alice", age: 30, role: "dev"}

# Access
$user.name                                    # "Alice"
$user | get age                               # 30

# Add / update
$user | upsert email "a@example.com"          # add field
$user | update age 31                         # change field
$user | reject role                           # drop field

# Merge / compare
{a: 1, b: 2} | merge {c: 3}                   # {a: 1, b: 2, c: 3}
$user | columns                               # ["name" "age" "role"]
$user | values                                # ["Alice" 30 "dev"]

# Iterate keys / values
$user | items { |k, v| $"($k) -> ($v)" }
```

## Tables

A table is a list of records that share the same column shape. Most built-in commands return a table — `ls`, `ps`, `sys`, `http get` (with JSON), and friends.

```bash
# Literal table
[[name age]; ["Alice" 30] ["Bob" 25] ["Carol" 35]]

# From data sources
open users.csv                                # auto-parses CSV
open users.json                               # auto-parses JSON
ls | select name size modified

# Access patterns
ls | get 0                                    # first row (record)
ls | get name                                 # name column (list)
ls | get name.0                               # first name (string)
(ls).name                                     # equivalent shorthand

# Common table commands
ls | where size > 1mb
ls | sort-by size --reverse
ls | select name size
ls | reject modified
ls | first 5
ls | last 5
ls | length
ls | reverse
```

## Pipelines

Pipelines are the heart of nu. Data flows from left to right with explicit, typed transformations.

```bash
# Filter → project → sort → limit
ls | where type == file | select name size | sort-by size -r | first 10

# Map with each
[1 2 3 4 5] | each { |n| $n * $n }            # [1 4 9 16 25]

# Filter with where
ls | where modified > (date now) - 1day

# Project columns
ps | select pid name cpu mem

# Sort
ls | sort-by modified -r
[[score]; [3] [1] [2]] | sort-by score        # ascending by default

# Group rows
ls | group-by type
# Returns a record: {file: <table>, dir: <table>}

# Insert / update / move columns
ls | insert kb { |row| $row.size / 1kb }
ls | update name { |row| $row.name | str upcase }
ls | move size --before name

# Reshape
[[a b]; [1 2] [3 4]] | transpose
# ╭──────┬─────────╮
# │ col1 │  col2   │
# │ a    │ b       │
# │ 1    │ 2       │
# │ 3    │ 4       │
# ╰──────┴─────────╯
```

## Conditionals

```bash
# if / else as expression — returns a value
let level = if $count > 10 { "many" } else if $count > 0 { "some" } else { "none" }

# match — structural pattern match
let kind = match $x {
    0 => "zero",
    1 => "one",
    2..10 => "small",
    _ => "big",
}

# match on records
match $user {
    {role: "admin"} => "elevated",
    {role: "guest"} => "anonymous",
    _ => "regular",
}

# Boolean short-circuit
$a > 0 and $b > 0
$x == null or $x == ""
not ($flag)
```

## Loops

In a pipeline, prefer `each`/`where`/`reduce` — they return values. Use `for` and `while` only for side effects that don't produce data.

```bash
# for — side effects, no return value
for n in 1..5 { print $"n = ($n)" }

# while
mut i = 0
while $i < 5 { print $i; $i = $i + 1 }

# loop with break
mut n = 0
loop {
    $n = $n + 1
    if $n >= 10 { break }
}

# In pipelines — each returns a list
1..5 | each { |n| $n * 2 }                    # [2 4 6 8 10]

# par-each — parallel map, order-preserving
ls **/*.log | par-each { |f| open $f.name | lines | length } | math sum
```

## Functions

`def` creates a custom command with explicit parameter types. Flags are declared with `--flag (-f)`. Use `def --env` for commands that mutate `$env`.

```bash
def greet [name: string, --loud (-l)] {
    let msg = $"Hello, ($name)!"
    if $loud { $msg | str upcase } else { $msg }
}

greet "World"                                 # "Hello, World!"
greet "World" --loud                          # "HELLO, WORLD!"
greet "World" -l

# Default parameter
def greet [name: string = "stranger"] { $"Hello, ($name)!" }
greet                                         # "Hello, stranger!"

# Return type
def add [a: int, b: int]: nothing -> int { $a + $b }

# Rest parameters
def sum-all [...nums: int] {
    $nums | math sum
}
sum-all 1 2 3 4 5                             # 15

# Mutating $env
def --env cd-up [] {
    cd ..
}

# Wrapped (passthrough) command
def --wrapped my-git [...args] {
    ^git ...$args
}
```

## Closures

A closure is a parameterized block. They are first-class values you can assign, pass, and invoke with `do`.

```bash
let double = { |x| $x * 2 }
do $double 21                                 # 42

# Closures capture enclosing scope by value
let factor = 3
let scale = { |x| $x * $factor }
[1 2 3] | each $scale                         # [3 6 9]

# Closures with multiple parameters
let combine = { |a, b| $"($a)-($b)" }
do $combine "left" "right"                    # "left-right"

# Used by built-ins
[1 2 3 4] | where { |n| $n mod 2 == 0 }       # [2 4]
[1 2 3 4] | reduce { |it, acc| $acc + $it }   # 10
```

## Control Flow Values

```bash
# Early return inside a function
def first-positive [xs: list<int>] {
    for x in $xs { if $x > 0 { return $x } }
    null
}

# try / catch
try {
    open missing.txt
} catch { |err|
    print $"failed: ($err.msg)"
    null
}

# do --ignore-errors — swallow external errors
do --ignore-errors { ^false }                 # exit code 1, no panic

# Raising errors
def safe-div [a: int, b: int] {
    if $b == 0 {
        error make { msg: "division by zero", label: { text: "denominator", span: (metadata $b).span } }
    }
    $a / $b
}
```

## Modules

Modules group related commands. Use `use` to import; `use mod *` brings all exports into scope. Set `NU_LIB_DIRS` to control the lookup path.

```bash
# mymod.nu — module file
export def hello [] { "Hello from module" }
export def add [a: int, b: int] { $a + $b }
export const VERSION = "1.0.0"

# Use the module
use mymod.nu                                  # loads as namespace mymod
mymod hello                                   # call namespaced
mymod add 2 3

# Bring everything into top-level scope
use mymod.nu *
hello
add 2 3

# Selective import
use mymod.nu [hello]

# Lookup path
$env.NU_LIB_DIRS = [
    ($nu.default-config-dir | path join "scripts")
    ($nu.home-path | path join ".nu" "lib")
]
```

## File Operations

`ls`, `cd`, `mkdir`, `mv`, `cp`, `rm` all return tables (or operate on paths). Filesystem listings are first-class data — pipe them through `where` and `sort-by` directly.

```bash
ls                                            # current directory as a table
ls *.md                                       # glob
ls **/*.rs                                    # recursive glob
ls -la                                        # long listing flags

# Pipe ls into anything
ls | where size > 1mb | sort-by modified -r
ls | where type == dir | length               # subdirectory count
ls **/*.{ts,tsx} | where size < 100kb

# Path operations
"/etc/hosts" | path basename                  # "hosts"
"/etc/hosts" | path dirname                   # "/etc"
"/etc/hosts" | path parse                     # record with parent/stem/extension

# Create / move / remove
mkdir new_dir
mkdir -p path/to/new_dir
mv old.txt new.txt
cp -r src dest
rm -r old_dir
touch newfile.txt

# Walk a directory recursively
ls **/* | where type == file | length
```

## Reading Files

`open` auto-detects the format from the extension and produces structured data. Use `--raw` to force raw text/binary.

```bash
open data.json                                # → record/list/table
open data.yaml                                # → record/list/table
open Cargo.toml                               # → record
open data.csv                                 # → table
open data.tsv                                 # → table
open data.xml                                 # → record (xml-shape)
open script.nu                                # → string (.nu unparsed)
open report.md                                # → string
open binary.bin --raw                         # → binary
open text.log --raw                           # → string (no parsing)

# Stream large files line-by-line
open access.log --raw | lines | first 10
open access.log --raw | lines | where { |l| $l =~ "ERROR" }

# Force parser regardless of extension
open data.txt | from json
open data.weird | from csv --separator '|'
```

## Writing Files

`save` writes a value back. Output is serialized to match the file extension, or use the explicit `to <fmt>` family.

```bash
{a: 1, b: 2} | save config.json               # auto JSON
{a: 1, b: 2} | save -f config.json            # force overwrite
ls | to csv | save files.csv

# Explicit conversions
{a: 1} | to json                              # '{"a":1}'
{a: 1} | to yaml
{a: 1} | to toml
ls     | to csv
ls     | to tsv
{a: 1} | to nuon                              # nu's native format

# Append (raw text)
"new line" | save --append --raw log.txt

# Save raw bytes
0x[de ad be ef] | save --raw blob.bin
```

## Subprocess

External commands behave like commands but their output is text by default. Use `^` to force the external version when a built-in shadows the name. `complete` captures stdout, stderr, and exit code as a record.

```bash
^ls                                           # force external ls (skip nu builtin)
^git status
^echo "hello"

# Pipe external output through nu parsers
^cat /etc/passwd | lines | first 5
^ip -j addr | from json                       # JSON-emitting tools shine

# Capture stdout/stderr/exit_code
let result = (do { ^cargo build } | complete)
$result.exit_code                             # 0 / 1 / etc.
$result.stdout
$result.stderr

# Run external with env tweaks
with-env {DEBUG: 1, RUST_LOG: "info"} { ^cargo run }

# Trim noisy output
^which python | str trim

# Pass nu data into stdin of an external
"hello" | ^cat
ls | to csv | ^column -s, -t
```

## Stdio / Args

```bash
# Reading user input
let name = (input "What is your name? ")
print $"Hello, ($name)"

# Read a password (no echo)
let pw = (input --suppress-output "Password: ")

# Print and stderr
print "stdout message"
print -e "stderr message"

# Script arguments — declare via main
def main [first: string, ...rest: string] {
    print $"first = ($first)"
    print $"rest  = ($rest)"
}
# nu script.nu hello world foo

# Exit codes
exit 0                                        # success
exit 2                                        # failure
```

## Environment Variables

Environment variables live on the `$env` record. `$env.PATH` is a real list of strings, not a colon-separated string — append/prepend with list operations. Permanent changes go in `env.nu`.

```bash
# Read
$env.HOME
$env.USER
$env | columns | sort

# Set (current scope only)
$env.MY_VAR = "value"

# Set inside a function — must use def --env
def --env enable-debug [] { $env.DEBUG = "1" }
enable-debug

# PATH manipulation
$env.PATH = ($env.PATH | append "/opt/bin")
$env.PATH = ($env.PATH | prepend "~/.local/bin")
$env.PATH = ($env.PATH | uniq)
$env.PATH = ($env.PATH | where $it != "/old/path")

# Run a single command with env overrides
with-env {RUST_LOG: "debug", PORT: "8080"} { ^myserver }

# Export to children — happens automatically; nu serializes $env.PATH back to OS string
^printenv PATH                                # see what children see

# Persistent — write to env.nu
config env                                    # opens env.nu in editor
```

## Date and Time

```bash
date now                                      # current datetime
date now | format date "%Y-%m-%d %H:%M:%S"
date now | format date "%FT%T%z"

# Parse a string into a datetime
"2026-04-25" | into datetime
"2026-04-25T12:00:00+00:00" | into datetime

# Date arithmetic with durations
(date now) + 1day
(date now) - 1wk
(date now) - 2024-01-01T00:00:00+00:00        # -> duration

# Components
date now | date to-record                     # {year, month, day, hour, minute, ...}
date now | date to-table                      # table form

# Common idioms
ls | where modified > (date now) - 7day
ls | sort-by modified --reverse | first 5

# Conversion
date now | format date "%s"                   # unix timestamp string
1714003200 * 1sec + 1970-01-01T00:00:00+00:00 # unix → datetime
```

## JSON / YAML / TOML / CSV / XML

```bash
# JSON
'{"a": 1, "b": [2, 3]}' | from json
{a: 1, b: [2, 3]} | to json
{a: 1} | to json --indent 2

# YAML
"a: 1\nb: [2, 3]" | from yaml
{a: 1} | to yaml

# TOML
open Cargo.toml | get package.name
{package: {name: "demo"}} | to toml

# CSV / TSV
"a,b\n1,2\n3,4" | from csv
"a\tb\n1\t2"   | from tsv
ls | to csv

# XML
open feed.xml | get content                   # nested record shape

# INI (via plugin or open --raw + custom parser)
open config.ini                               # if registered

# Auto-detection on open
open users.json                               # JSON
open users.yaml                               # YAML
open users.toml                               # TOML
open users.csv                                # CSV
open users.tsv                                # TSV
open users.xml                                # XML
```

## HTTP

`http` is a built-in client returning either parsed structured data or raw bytes. The `--full` flag returns a record with status/headers/body.

```bash
# GET — auto-parses JSON content-type
http get https://api.github.com/repos/nushell/nushell

# Specific field
http get https://api.github.com/repos/nushell/nushell | get stargazers_count

# Headers
http get https://api.example.com --headers [Authorization $"Bearer ($env.TOKEN)"]

# POST JSON
http post https://api.example.com/items {name: "demo", count: 1}

# Other verbs
http put    https://api.example.com/items/1 {name: "updated"}
http patch  https://api.example.com/items/1 {count: 2}
http delete https://api.example.com/items/1

# Capture full response
let resp = (http get https://example.com --full)
$resp.status                                  # 200
$resp.headers
$resp.body

# Download file
http get https://example.com/big.bin | save big.bin

# Form-encoded body
http post https://example.com/login {user: "alice", pass: "secret"} --content-type "application/x-www-form-urlencoded"
```

## SQL Queries

Two paths: the built-in `query db` for SQLite files, and the `polars` plugin for in-memory SQL over DataFrames.

```bash
# SQLite — read a database file
open users.db | query db "SELECT id, name FROM users WHERE active = 1"

# SQLite — write
open users.db | query db "INSERT INTO users(name) VALUES ('Eve')"

# Schema inspection
open users.db | schema

# Polars in-memory SQL
plugin add ~/.cargo/bin/nu_plugin_polars
plugin use polars

let df = (open data.csv | polars into-df)
$df | polars sql "SELECT region, AVG(amount) FROM df GROUP BY region"
```

## Plugins

Plugins extend nu with Rust-built commands. Install via cargo, register via `plugin add`, then activate with `plugin use`.

```bash
# Install a few popular plugins
cargo install nu_plugin_polars --locked
cargo install nu_plugin_query  --locked
cargo install nu_plugin_inc    --locked
cargo install nu_plugin_formats --locked

# Register and activate
plugin add ~/.cargo/bin/nu_plugin_polars
plugin add ~/.cargo/bin/nu_plugin_query
plugin add ~/.cargo/bin/nu_plugin_inc
plugin use polars
plugin use query
plugin use inc

# List / remove
plugin list
plugin rm polars

# Common community plugins
# nu_plugin_polars   — DataFrame engine
# nu_plugin_query    — XML/HTML/CSS-selector queries
# nu_plugin_inc      — increment a value (used in version bumps)
# nu_plugin_dbus     — Linux DBus client
# nu_plugin_skim     — fuzzy finder
```

## Polars Plugin

The `polars` plugin embeds the Rust polars DataFrame engine — fast columnar operations, lazy execution, SQL on tables.

```bash
plugin use polars

# Load CSV / Parquet
let df = (polars open data.csv)
let df = (polars open data.parquet)

# Convert nu table → DataFrame
let df = ([[a b]; [1 2] [3 4]] | polars into-df)

# Inspect
$df | polars schema
$df | polars first 5

# Filter / select / sort
$df | polars filter ((polars col amount) > 100)
$df | polars select a b
$df | polars sort-by [amount] --reverse

# Group-by + aggregations
$df
  | polars group-by region
  | polars agg [(polars col amount | polars sum) (polars col amount | polars mean)]

# SQL
$df | polars sql "SELECT region, SUM(amount) total FROM df GROUP BY region ORDER BY total DESC"

# Save back
$result | polars save out.parquet
```

## Working with Errors

Errors in nu are first-class structured values with a `msg`, optional labels (with file spans), and a help string.

```bash
# Throw
error make { msg: "bad input" }
error make {
    msg: "expected positive int",
    label: { text: "here", span: (metadata $val).span }
}

# Catch
try { open missing.txt } catch { |e| print $"failed: ($e.msg)" }

# Inspect a caught error
try { 1 / 0 } catch { |e|
    $e.msg
    $e.debug
    $e.raw
}

# Ignore failures from external commands
do --ignore-errors { ^false }                 # no panic
let result = (do --ignore-errors { ^cmd } | complete)
if $result.exit_code != 0 { print "failed but continuing" }

# Optional access on null with `?`
{a: 1}.b?                                     # null instead of error
$rec.maybe.deep?                              # null on missing path
```

## Strings — Advanced

```bash
# parse — extract fields with a format string
"name=alice age=30" | parse "name={n} age={a}"
ls | each { |row| $row.name | parse "{stem}.{ext}" } | flatten

# split row / split column / split chars
"a,b,c" | split row ","                       # list
"a,b\n1,2" | split column ","                 # table-of-strings
"abcdef" | split chars                        # ["a" "b" "c" "d" "e" "f"]

# str command family
"  hi  "    | str trim
"abc"       | str length
"abc"       | str index-of "b"               # 1
"abcabc"    | str replace --all "b" "B"      # "aBcaBc"
"AbCdEf"    | str downcase
"AbCdEf"    | str title-case
"foo bar"   | str pascal-case                # "FooBar"
"Foo Bar"   | str snake-case                 # "foo_bar"
"FooBar"    | str kebab-case                 # "foo-bar"

# Encoding
"hello"     | encode base64
"aGVsbG8="  | decode base64 | decode utf-8
"hi"        | encode hex                     # "6869"

# Regex match (use =~ for boolean test, str replace --regex for substitution)
"abc123" =~ '\d+'                             # true
"abc123" | str replace --regex '\d+' "###"   # "abc###"
"abc123 def456" | parse --regex '(?P<word>\w+?)(?P<num>\d+)'
```

## Math Command

```bash
[3 1 4 1 5 9 2 6] | math sum                  # 31
[3 1 4 1 5 9 2 6] | math avg                  # 3.875
[3 1 4 1 5 9 2 6] | math min                  # 1
[3 1 4 1 5 9 2 6] | math max                  # 9
[3 1 4 1 5 9 2 6] | math median               # 3.5
[3 1 4 1 5 9 2 6] | math mode                 # [1]
[3 1 4 1 5 9 2 6] | math product              # 6480
[3 1 4 1 5 9 2 6] | math stddev               # ≈2.66
[3 1 4 1 5 9 2 6] | math variance             # ≈7.11

# Operates on table columns when piped from a table
ls | get size | math sum                      # total bytes in cwd
ps | get cpu | math max
```

## Group-By and Aggregation

```bash
# Group rows by a column
ls | group-by type
# {file: <table>, dir: <table>, symlink: <table>}

# Group + count per category
ls | group-by type | transpose type rows | insert count { |r| $r.rows | length } | reject rows
# ╭───┬────────┬───────╮
# │ # │  type  │ count │
# │ 0 │ file   │ 14    │
# │ 1 │ dir    │ 3     │
# ╰───┴────────┴───────╯

# Reduce per group via reduce + group-by
$transactions
  | group-by region
  | transpose region txs
  | insert total { |r| $r.txs | get amount | math sum }
  | reject txs
  | sort-by total -r

# Multi-key group via composite
$logs | group-by { |r| $"($r.host)-($r.level)" }

# Pivot — transpose a record/table
[[a b]; [1 2] [3 4]] | transpose --header-row
```

## Common Builtins

These are the day-to-day verbs. All return structured data.

```bash
# Filesystem
cd, ls, pwd, mkdir, mv, cp, rm, touch, glob, watch

# Tables / lists / records
where, select, reject, sort-by, group-by, transpose, flatten, uniq, drop, take, first, last,
length, get, insert, update, upsert, move, rename, reverse, append, prepend, zip, chunks,
each, par-each, reduce, filter, find, skip, all, any, default

# Strings
str, parse, split row, split column, split chars, lines, char, decode, encode, format

# Numbers
math, into, format number

# Conversion
into int, into float, into string, into bool, into datetime, into duration, into filesize,
into binary, into record, into nothing
from json, from yaml, from toml, from csv, from tsv, from xml, from ini, from nuon
to json, to yaml, to toml, to csv, to tsv, to xml, to nuon, to text, to md

# I/O
open, save, http, watch, port, complete, do, with-env

# Control / misc
def, alias, use, source, exit, exec, help, history, which, debug, describe, metadata,
explore, view, table, print, input

# Date
date now, date format, date to-record, date to-table, format date

# DB
query db, schema, polars sql
```

## Custom Commands

Custom commands declare their parameters with types — nu auto-builds a `--help` page from the signature plus your `# comments`.

```bash
# Heavily annotated example
# Show the largest files in a directory.
def biggest [
    path: path = ".",                   # directory to scan
    --top (-t): int = 5,                # number of rows to return
    --kind: string = "file"             # "file" | "dir"
] {
    ls $path
      | where type == $kind
      | sort-by size -r
      | first $top
}

# Generated help
help biggest
biggest --help

# Wrapped passthrough — forward all flags to an external
def --wrapped my-grep [...rest] { ^grep -n ...$rest }

# Subcommands
def "tools list"   [] { ["foo" "bar" "baz"] }
def "tools install" [name: string] { print $"installing ($name)" }

tools list
tools install foo
```

## Migrating from Bash

| Bash idiom                     | Nu equivalent                                                                 |
| ------------------------------ | ----------------------------------------------------------------------------- |
| `ls -la`                       | `ls -la` (returns a table, not raw text)                                      |
| `grep "ERROR" log.txt`         | `open log.txt --raw | lines | where { $in =~ "ERROR" }`                       |
| `cat data.csv | awk -F,`       | `open data.csv | select 0`                                                    |
| `find . -name "*.rs"`          | `ls **/*.rs`                                                                  |
| `du -sh * | sort -h`           | `ls | select name size | sort-by size`                                        |
| `curl -s api | jq .field`      | `http get api | get field`                                                    |
| `ls | wc -l`                   | `ls | length`                                                                 |
| `for f in *.md; do ...; done`  | `ls *.md | each { |f| ... }`                                                  |
| `export VAR=val`               | `$env.VAR = "val"`                                                            |
| `if [ -f "$f" ]; then`         | `if ($f | path exists) {`                                                     |
| `command1 && command2`         | `command1; command2` (nu stops on error by default)                          |
| `$(cmd)` substitution          | `(cmd)` parens, no string boundary needed                                     |

```bash
# Bridge: call any traditional Unix tool, then parse its output
^uname -a              | str trim
^ip -j addr            | from json
^df -k                 | detect columns
^ps -ef                | detect columns
^docker ps --format json | from json --objects   # with formats plugin
```

## Common Errors

```bash
# Error: "command not found"
foo                                           # not a builtin, alias, or external
# Fix: confirm via `which foo` or use `^foo` if name clashes with a builtin
which foo
^bar                                          # force external resolution

# Error: "Type mismatch" / "column 'x' not found in record"
ls | get xyz
# Error: column not found
#   --> column is missing or misspelled
# Fix: inspect first
ls | columns
ls | first | describe

# Error: "External command 'X' did not return data"
# Happens when piping plain external output into a nu command expecting structure.
^echo "hi" | get 0
# Fix: parse first
^echo "hi" | lines | get 0

# Error: "Cannot store value of type ..."
$env.PATH = "/a:/b:/c"                        # PATH must be list, not string
# Fix:
$env.PATH = ("/a:/b:/c" | split row ":")

# Error: "Mismatched delimiter"
let x = [1, 2, 3                               # missing closing ']'
# Fix: balance brackets

# Error: "Variable not found"
let x = 5
print x                                        # missing $
# Fix: use $x for variable references
print $x

# Error: "expected closure but found block"
ls | each { print $in }                        # OK
ls | each $undefined                           # if $undefined isn't a closure → error
```

## Common Gotchas

Each pair shows the broken form and the fix.

```bash
# Spreading a list into command arguments
let files = ["a.txt" "b.txt" "c.txt"]

# Broken — passes the LIST as a single argument
^cat $files

# Fixed — spread with ...
^cat ...$files
```

```bash
# Environment variables are typed structured values, not raw strings
# Broken — concatenating PATH with ":"
$env.PATH = $"/usr/local/bin:($env.PATH)"     # PATH becomes a string!

# Fixed — keep it a list
$env.PATH = ($env.PATH | prepend "/usr/local/bin")
```

```bash
# `for` does NOT return values
# Broken — expecting a list back
let result = for n in [1 2 3] { $n * 2 }      # nothing returned
# Fixed — use each
let result = ([1 2 3] | each { |n| $n * 2 })  # [2 4 6]
```

```bash
# Piping into where/select needs a table or list, not a string
# Broken
"hello world" | where length > 3
# Fixed
"hello world" | split row " " | where { |w| ($w | str length) > 3 }
```

```bash
# Closures auto-return — no explicit return needed
# Broken — `return` inside a closure stops the closure too
[1 2 3] | each { |x| return ($x * 2) }
# Fixed
[1 2 3] | each { |x| $x * 2 }
```

```bash
# String interpolation MUST use parens, not bash-style braces
# Broken
$"Hello $name"                                # treats as literal
# Fixed
$"Hello ($name)"
```

```bash
# Capturing external command output
# Broken — assigns the live process to a variable
let log = ^cat large.log
# Fixed — collect output first
let log = (^cat large.log | collect)
```

```bash
# Modifying $env from a function requires --env
# Broken — env mutation lost on return
def setvar [] { $env.X = "y" }
setvar
$env.X                                         # not set!
# Fixed
def --env setvar [] { $env.X = "y" }
```

```bash
# Quoting an interpolation literally
# Broken
print '$"($name)"'                             # literal '$"($name)"'
# Fixed
print $"($name)"
```

```bash
# Mistaking `==` precedence with `and` chains
# Broken — pipe takes precedence weirdly across boundaries
ls | where size > 1mb and type == file        # works, but be explicit
# Fixed — group with parens
ls | where { |r| $r.size > 1mb and $r.type == "file" }
```

## Performance

- Nu compiles scripts to a typed bytecode-like IR before executing — re-running a function is cheap. Long pipelines are fully fused.
- Built-in commands operate on structured data directly without re-parsing — strongly prefer them over `awk`/`sed`/`jq` shell-outs.
- `par-each` distributes a closure across cores. Use it for CPU-bound work (parsing many files, hashing, image transforms).
- `collect` materializes a streaming pipeline; insert before reusing the same data multiple times.
- For analytics on tables larger than memory pressure allows, switch to the `polars` plugin — it uses lazy execution and SIMD.
- External commands fork a process per call. In hot loops, prefer a single shell-out followed by `lines`/`from json`/`from csv` over per-iteration shell-outs.
- Use `do --ignore-errors` only where errors are expected; otherwise let nu surface them so you don't burn time on a silent failure.

```bash
# Benchmark a snippet
timeit { ls **/*.rs | length }

# Cache an expensive pipeline
let cached = (do { http get $url } | collect)

# Parallel by default for IO-bound work
ls **/*.log | par-each { |f| open $f.name --raw | lines | length } | math sum
```

## Idioms

```bash
# Read JSON and grab a nested field
open package.json | get dependencies | columns

# Largest 5 files in cwd
ls | sort-by size --reverse | first 5 | select name size

# Recent edits
ls | where modified > (date now) - 1day | sort-by modified --reverse

# Convert YAML to JSON
open input.yaml | to json | save output.json

# Count files by extension
ls **/* | where type == file | each { |f| $f.name | path parse | get extension } | uniq --count | sort-by count -r

# Group-by SQL flavor
$txs | group-by region | transpose region rows | insert total { |r| $r.rows | get amount | math sum } | reject rows | sort-by total -r

# REST API → table → CSV
http get https://api.github.com/users/torvalds/repos
  | select name stargazers_count language
  | rename name stars language
  | sort-by stars --reverse
  | save torvalds-repos.csv

# Tail a log live
tail -f /var/log/syslog | each { |line| if ($line | str contains "ERROR") { $line } }
```

## Tips

- `describe` early and often — it tells you what type a stage of the pipeline produces.
- `^` forces an external command when nu has a builtin of the same name (`^ls`, `^cat`).
- `explore` opens a TUI viewer over any table — perfect for browsing API responses.
- `table --expand` shows nested cells without truncation.
- Save reusable functions to `~/.config/nushell/scripts/` and add it to `NU_LIB_DIRS`.
- Switch the prompt with `starship` (`use ~/.cache/starship/init.nu`).
- The `--full` flag on `http get` returns the full response (status, headers, body).
- For machine-readable output, prefer `to nuon` over `to json` — round-trips all nu types including filesize, duration, and datetime.
- `history | last 20` is a quick recall of recent commands; `history | where command =~ git`.
- `which cmd` shows whether a name resolves to an internal command, alias, custom, or external.
- `help <cmd>` for built-ins is exhaustive — types, examples, flags. Auto-built for your custom commands too.
- Use `?` for safe access on potentially missing record fields (`$rec.maybe?`).
- `tee` is built-in: split a stream and persist it (`http get url | tee { save raw.json } | get items`).
- For shell-outs, capture with `complete` so you have stdout/stderr/exit_code together.

## See Also

- bash, zsh, fish, shell-scripting, polyglot, regex, awk, sql, json

## References

- [Nushell Official Site](https://www.nushell.sh/)
- [Nushell Book](https://www.nushell.sh/book/)
- [Nushell Cookbook](https://www.nushell.sh/cookbook/)
- [Nushell Command Reference](https://www.nushell.sh/commands/)
- [Nushell GitHub Repository](https://github.com/nushell/nushell)
- [Awesome Nu](https://github.com/nushell/awesome-nu)
- [Nu Plugin Registry](https://www.nushell.sh/book/plugins.html)
- [Polars Plugin Docs](https://github.com/nushell/nushell/tree/main/crates/nu_plugin_polars)
- [Nu Migrating from Bash Guide](https://www.nushell.sh/book/coming_from_bash.html)
- [Custom Commands](https://www.nushell.sh/book/custom_commands.html)
- [Modules](https://www.nushell.sh/book/modules.html)
- [Standard Library](https://www.nushell.sh/book/standard_library.html)
