# ripgrep (fast regex search)

A line-oriented search tool that recursively searches the current directory for a regex pattern, automatically respecting .gitignore rules, skipping binary files and hidden directories, with multi-threaded performance that makes it dramatically faster than grep for codebase searches.

## Basic Usage

### Simple Search

```bash
# Search current directory recursively (default)
rg "pattern"

# Search specific files or directories
rg "TODO" src/
rg "func main" main.go

# Search stdin
cat /var/log/syslog | rg "error"

# Case insensitive
rg -i "error"

# Fixed string (no regex)
rg -F "user.name[0]"
```

### Regex Patterns

```bash
# Default regex engine (Rust regex, similar to PCRE without backreferences)
rg '\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}'    # IP addresses
rg 'func\s+\w+\('                             # Go function definitions
rg 'import\s*\{[^}]+\}'                       # JS/TS imports

# PCRE2 mode (backreferences, lookaheads)
rg -P '(\w+)\s+\1'                            # repeated words
rg -P '(?<=func\s)\w+'                        # word after "func" (lookbehind)

# Multiline matching
rg -U 'struct\s+\w+\s*\{[^}]*\}'             # Go struct definitions
```

## Output Control

### Context Lines

```bash
# Lines after match
rg -A 3 "error"

# Lines before match
rg -B 2 "error"

# Lines before and after
rg -C 5 "error"

# Show only matching part of line
rg -o '\d+\.\d+\.\d+'

# Count matches per file
rg -c "TODO"

# Count total matches
rg -c "TODO" | awk -F: '{sum+=$2} END {print sum}'
```

### File Listing Modes

```bash
# List only filenames with matches
rg -l "TODO"

# List filenames WITHOUT matches
rg --files-without-match "TODO"

# Show line numbers (default when terminal)
rg -n "pattern"

# No line numbers
rg --no-line-number "pattern"

# No filename headers
rg --no-filename "pattern"

# JSON output (for tooling)
rg --json "pattern"
```

## File Type Filters

### Built-in Type Definitions

```bash
# Search only Go files
rg -t go "func main"

# Search only Python files
rg -t py "import"

# Search only Markdown files
rg -t md "TODO"

# Multiple types
rg -t go -t rust "unsafe"

# Exclude a type (inverse)
rg -T js "import"              # search all EXCEPT JavaScript

# List all known types
rg --type-list

# Define custom type
rg --type-add 'web:*.{html,css,js}' -t web "class"
```

### Glob Patterns

```bash
# Include only specific patterns
rg -g '*.go' "pattern"
rg -g 'src/**/*.ts' "import"

# Exclude patterns (prefix with !)
rg -g '!*.min.js' "function"
rg -g '!vendor/**' "import"
rg -g '!{vendor,node_modules}/**' "TODO"

# Combine multiple globs
rg -g '*.go' -g '!*_test.go' "func"
```

## Directory and File Filtering

### Controlling What Gets Searched

```bash
# Search hidden files/directories too
rg --hidden "pattern"

# Don't respect .gitignore
rg --no-ignore "pattern"

# Don't respect any ignore files
rg --no-ignore-vcs --no-ignore-global "pattern"

# Follow symlinks
rg -L "pattern"

# Max depth
rg --max-depth 2 "pattern"

# Search specific file by name
rg --files | rg "config"

# List all files that would be searched (no pattern)
rg --files
rg --files -t go                # only Go files
```

## Replacement

### Search and Replace

```bash
# Preview replacement (does not modify files)
rg "old_func" --replace "new_func"

# With capture groups
rg '(\w+)_test\.go' --replace '${1}_spec.go'

# Replace in place (requires additional tool)
rg -l "old_func" | xargs sed -i 's/old_func/new_func/g'

# PCRE2 replacement with backreferences
rg -P '(\w+)\s+\1' --replace '$1'
```

## Configuration

### Config File

```bash
# ripgrep reads ~/.config/ripgrep/config or RIPGREP_CONFIG_PATH
# One flag per line, comments with #

# ~/.config/ripgrep/config
--smart-case
--hidden
--glob=!.git
--glob=!node_modules
--glob=!vendor
--max-columns=200
--max-columns-preview
--colors=match:fg:magenta
--colors=match:style:bold
```

### Smart Case

```bash
# Smart case: case-insensitive unless pattern has uppercase
rg -S "error"                  # matches "Error", "ERROR", "error"
rg -S "Error"                  # matches only "Error" (has uppercase)

# Force case sensitive
rg -s "error"
```

## Advanced Features

### Sorting and Statistics

```bash
# Sort results by file path
rg --sort path "TODO"

# Sort by modification time
rg --sort modified "TODO"

# Sort by number of matches
rg -c "TODO" | sort -t: -k2 -rn

# Show stats
rg --stats "TODO"
```

### Combining with Other Tools

```bash
# Feed into fzf for interactive selection
rg --color=always "pattern" | fzf --ansi

# Pipe to bat for highlighted output
rg -l "pattern" | xargs bat

# Use with xargs for batch operations
rg -l "deprecated_func" | xargs -I{} sed -i 's/deprecated_func/new_func/g' {}

# Count lines of code by type
rg --files -t go | xargs wc -l | tail -1
```

## Tips

- Smart case (`-S`) is the most useful default -- searches case-insensitively until you type an uppercase letter.
- ripgrep automatically respects `.gitignore`, `.ignore`, and `.rgignore` files, so you rarely need manual exclusions.
- Use `-t type` instead of glob patterns when possible -- it is faster and catches all relevant extensions.
- The `--json` flag outputs structured JSON per match, ideal for building editor integrations and tooling.
- Use `-U` (multiline) for patterns spanning multiple lines, like struct definitions or multi-line strings.
- `rg --files` lists all files that would be searched -- great for verifying your filters before a search.
- `RIPGREP_CONFIG_PATH` lets you point to a config file with persistent defaults like `--smart-case` and `--hidden`.
- The `-g '!pattern'` negated glob is the fastest way to exclude directories like `vendor/` or `node_modules/`.
- For PCRE2 features (lookaheads, backreferences), use `-P` but be aware it disables some optimizations.
- `rg -c "pattern" | sort -t: -k2 -rn` ranks files by match count -- useful for finding hotspots.
- Use `--max-columns=200 --max-columns-preview` to truncate long lines while showing a preview.

## See Also

- grep, fzf, fd, bat, find, sed

## References

- [ripgrep GitHub Repository](https://github.com/BurntSushi/ripgrep)
- [ripgrep User Guide](https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md)
- [ripgrep FAQ](https://github.com/BurntSushi/ripgrep/blob/master/FAQ.md)
- [ripgrep Regex Syntax](https://docs.rs/regex/latest/regex/#syntax)
- [ripgrep vs grep vs ag Benchmarks](https://blog.burntsushi.net/ripgrep/)
- [PCRE2 Documentation](https://www.pcre.org/current/doc/html/)
- [Arch Wiki — ripgrep](https://wiki.archlinux.org/title/Ripgrep)
