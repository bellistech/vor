# Nushell (Nu)

> A modern shell that treats input and output as structured data — tables, records, and lists flow through pipelines instead of raw text, bringing database-style operations (where, select, sort-by, group-by, reduce) directly to the command line.

## Data Types

### Primitives

```nu
# Integers and floats
42
3.14

# Strings (single or double quotes)
"hello world"
'literal string, no interpolation'

# String interpolation
let name = "Nu"
$"Welcome to ($name)shell"

# Booleans
true
false

# Durations and file sizes
3hr + 15min
2gb + 500mb

# Dates
2026-04-03T12:00:00+00:00
(date now)
```

### Tables

```nu
# Literal table
[[name, age, role]; [Alice, 30, "dev"], [Bob, 25, "ops"], [Carol, 35, "dev"]]

# From command output (already structured)
ls | where size > 1kb

# Access column
(ls).name

# Access specific row
(ls | get 2)
```

### Records

```nu
# Record literal
{name: "Alice", age: 30, role: "dev"}

# Access field
{name: "Alice", age: 30}.name

# Merge records
{a: 1} | merge {b: 2}

# Update field
{name: "Alice", age: 30} | upsert age 31
```

### Lists

```nu
# List literal
[1 2 3 4 5]

# Nested list
[[1 2] [3 4] [5 6]]

# Range
1..10
1..5 | each { |n| $n * 2 }
```

## Pipeline Operators

### Filtering with where

```nu
# Filter rows by condition
ls | where size > 10kb
ps | where cpu > 5
sys host | get name

# Multiple conditions
ls | where size > 1kb and type == "file"
ls | where name =~ '\.rs$'

# Null checks
[{a: 1} {b: 2}] | where a != null
```

### Selecting and Renaming Columns

```nu
# Select specific columns
ls | select name size
ps | select pid name cpu

# Reject columns
ls | reject modified

# Rename columns
ls | rename filename filesize --column [name size]
```

### Sorting

```nu
# Sort ascending
ls | sort-by size
ls | sort-by modified --reverse

# Sort by multiple columns
[[name score grade]; [A 90 "A"] [B 85 "B"]] | sort-by score --reverse

# Reverse
ls | sort-by size | reverse
```

### Grouping and Aggregation

```nu
# Group by column
ls | group-by type

# Group and count
ps | group-by name | transpose name procs | insert count { |r| $r.procs | length }

# Reduce (fold)
[1 2 3 4 5] | reduce --fold 0 { |it, acc| $acc + $it }

# Math operations
[1 2 3 4 5] | math sum
[1 2 3 4 5] | math avg
ls | get size | math sum
```

### Transformations

```nu
# Each (map)
[1 2 3] | each { |n| $n * 2 }

# Par-each (parallel map)
ls | par-each { |f| open $f.name | lines | length }

# Flatten nested tables
[[1 2] [3 4]] | flatten

# Uniq and dedup
[1 2 2 3 3 3] | uniq
[1 2 2 3 3 3] | uniq --count
```

## Custom Commands

### Defining Commands

```nu
# Basic command
def greet [name: string] {
    $"Hello, ($name)!"
}

# With default parameter
def greet [name: string = "world"] {
    $"Hello, ($name)!"
}

# With return type
def add [a: int, b: int] -> int {
    $a + $b
}

# With flags
def search [pattern: string, --case-insensitive(-i)] {
    if $case_insensitive {
        grep -i $pattern
    } else {
        grep $pattern
    }
}
```

### Type System

```nu
# Type annotations
let x: int = 42
let items: list<int> = [1 2 3]
let config: record<host: string, port: int> = {host: "localhost", port: 8080}

# Type checking
42 | describe       # int
"hello" | describe  # string
```

## Modules and Overlays

### Creating Modules

```nu
# mymodule.nu
export def hello [] { "Hello from module" }
export def add [a, b] { $a + $b }
export const VERSION = "1.0.0"
```

### Using Modules

```nu
# Import module
use mymodule.nu

# Use module commands
mymodule hello
mymodule add 2 3

# Import specific items
use mymodule.nu [hello add]
hello

# Overlay
overlay use mymodule.nu
overlay hide mymodule
overlay list
```

## Configuration

### config.nu and env.nu

```nu
# ~/.config/nushell/env.nu — environment setup (loaded first)
$env.PATH = ($env.PATH | split row (char esep) | prepend '/usr/local/bin')
$env.EDITOR = "nvim"

# ~/.config/nushell/config.nu — shell configuration
$env.config = {
    show_banner: false
    edit_mode: vi
    table: {
        mode: rounded
        index_mode: auto
    }
    completions: {
        algorithm: fuzzy
        case_sensitive: false
    }
    history: {
        max_size: 100_000
        file_format: sqlite
    }
}
```

## Plugins

### Plugin Management

```nu
# Register a plugin
plugin add ~/.cargo/bin/nu_plugin_formats
plugin add ~/.cargo/bin/nu_plugin_query

# List registered plugins
plugin list

# Use plugin command (after registration)
"<html><body>Hello</body></html>" | query web "body"

# Remove plugin
plugin rm nu_plugin_formats
```

## Working with External Commands

### Running External Commands

```nu
# Run external command (output as raw string)
^git status

# Capture as structured data
^git log --oneline -10 | lines | split column " " hash message

# Pipe structured data to external
ls | to csv | save files.csv

# Complete — capture stdout, stderr, exit_code
do { ^cargo build } | complete
```

### Data Format Conversion

```nu
# JSON
open data.json
'{"a": 1}' | from json
{a: 1, b: 2} | to json

# CSV / TSV
open data.csv
ls | to csv
"a,b\n1,2" | from csv

# YAML / TOML
open config.yaml
open Cargo.toml
{a: 1} | to yaml

# Dataframes (with polars plugin)
open large_file.csv --raw | polars into-df
```

## Tips

- Use `describe` on any value to see its type — essential for debugging pipelines
- Prefix external commands with `^` when a Nu built-in has the same name (e.g., `^ls` vs `ls`)
- Use `explore` to interactively browse any structured data in a TUI viewer
- Pipe through `table --expand` to see deeply nested structures without truncation
- Use `to nuon` and `from nuon` for Nu's native serialization format — faster than JSON for Nu data
- Store reusable functions in `~/.config/nushell/config.nu` or separate module files
- Use `par-each` instead of `each` for CPU-bound parallel transformations
- Use `complete` to capture stdout, stderr, and exit_code from external commands
- History search works best with `file_format: sqlite` in config for faster fuzzy recall
- Errors are structured values — use `try { } catch { |e| $e }` for typed error handling
- Use `input list` for interactive selection menus in scripts

## See Also

- bash, zsh, fish, shell-scripting

## References

- [Nushell Official Documentation](https://www.nushell.sh/book/)
- [Nushell GitHub Repository](https://github.com/nushell/nushell)
- [Nushell Cookbook](https://www.nushell.sh/cookbook/)
- [Nushell Plugin Registry](https://www.nushell.sh/book/plugins.html)
- [Nushell Command Reference](https://www.nushell.sh/commands/)
