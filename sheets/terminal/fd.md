# fd (fast find alternative)

A simple, fast, and user-friendly alternative to the traditional find command that features sensible defaults, colorized output, regex patterns by default, respects .gitignore, parallel command execution, and a much more intuitive syntax for everyday file-finding tasks.

## Basic Usage

### Simple Patterns

```bash
# Find files matching a pattern (regex by default)
fd "pattern"

# Find files with exact name
fd -g "README.md"

# Find in a specific directory
fd "pattern" /path/to/dir

# Case insensitive (default for all-lowercase patterns)
fd "readme"                    # smart case: case-insensitive
fd "README"                    # smart case: case-sensitive (has uppercase)

# Force case sensitivity
fd -s "readme"                 # case sensitive
fd -i "README"                 # case insensitive
```

### Regex Patterns

```bash
# Default is regex
fd '\.go$'                     # files ending in .go
fd '^main\.'                   # files starting with "main."
fd '\d{4}-\d{2}-\d{2}'        # date-like filenames
fd 'test.*\.py$'               # Python test files

# Fixed string (glob mode)
fd -g "*.go"
fd -g "config.{json,yaml,toml}"
```

## File Type Filtering

### Type Flags

```bash
# Files only
fd -t f "pattern"

# Directories only
fd -t d "src"

# Symlinks only
fd -t l

# Executable files only
fd -t x

# Empty files and directories
fd -t e

# Socket files
fd -t s

# Combine types (OR logic)
fd -t f -t l "config"          # files and symlinks
```

### Extension Filtering

```bash
# Filter by extension
fd -e go                       # all .go files
fd -e rs                       # all .rs files
fd -e md                       # all .md files

# Multiple extensions
fd -e go -e rs                 # Go and Rust files

# Combine with pattern
fd -e go "test"                # Go files matching "test"
fd -e py "^test_"              # Python files starting with "test_"
```

## Exclusion and Filtering

### Excluding Patterns

```bash
# Exclude patterns
fd -E node_modules "pattern"
fd -E "*.min.js" "\.js$"
fd -E vendor -E .git "\.go$"

# Include hidden files (excluded by default)
fd -H "pattern"

# Include ignored files (from .gitignore)
fd -I "pattern"

# Include both hidden and ignored
fd -HI "pattern"

# No ignore (all files, including .git)
fd --no-ignore "pattern"
```

### Depth and Size

```bash
# Max depth
fd -d 2 "pattern"              # max 2 levels deep
fd --max-depth 3 "pattern"

# Min depth
fd --min-depth 2 "pattern"     # skip top-level matches

# Filter by size
fd -S +1m                      # files larger than 1 MB
fd -S -10k                     # files smaller than 10 KB
fd -S +100k -S -1m             # between 100 KB and 1 MB
```

### Time Filtering

```bash
# Modified within last 7 days
fd --changed-within 7d

# Modified before a specific date
fd --changed-before "2024-01-01"

# Modified within last 2 hours
fd --changed-within 2h

# Newer than a reference file
fd --newer "reference.txt"
```

## Command Execution

### Running Commands on Results

```bash
# Execute command for each result (parallel by default)
fd -e go --exec wc -l          # count lines in each Go file

# Placeholders:
#   {} = full path
#   {.} = path without extension
#   {/} = filename only
#   {//} = parent directory
#   {/.} = filename without extension

# Convert images
fd -e png --exec convert {} {.}.jpg

# Run tests for found files
fd -e go "test" --exec go test {}

# Change extension
fd -e .bak --exec mv {} {.}.old

# Execute with full path control
fd -e log --exec gzip {}
```

### Batch Execution

```bash
# Execute command once with all results as arguments
fd -e go --exec-batch wc -l    # single wc with all Go files

# Useful for commands that benefit from batch input
fd -e rs --exec-batch cargo fmt --
fd -e py --exec-batch black

# Batch delete
fd -e tmp --exec-batch rm

# Batch move
fd -e bak --exec-batch mv -t /backup/
```

## Output Formatting

### Custom Output

```bash
# Null-separated output (for xargs -0)
fd -0 "pattern" | xargs -0 rm

# Absolute paths
fd -a "pattern"

# List with details (like ls -l)
fd -l "pattern"

# Print file type indicator
fd "pattern" --list-details

# Color control
fd --color=never "pattern"     # for piping
fd --color=always "pattern"    # force colors in pipes
```

## Practical Examples

### Common Workflows

```bash
# Find and delete all .DS_Store files
fd -H -g ".DS_Store" --exec rm

# Find large log files
fd -e log -S +100m

# Find recently modified config files
fd -e yaml -e json -e toml --changed-within 1d

# Find broken symlinks
fd -t l -L --exec test ! -e {} \; -print

# Find duplicate filenames across directories
fd -t f | awk -F/ '{print $NF}' | sort | uniq -d

# Find Go files not in vendor
fd -e go -E vendor -E node_modules

# Find empty directories and remove them
fd -t e -t d --exec rmdir

# Count files by extension
fd -t f | sed 's/.*\.//' | sort | uniq -c | sort -rn
```

## Tips

- fd uses smart case by default: all-lowercase patterns are case-insensitive, patterns with uppercase are case-sensitive.
- By default fd respects `.gitignore`, `.ignore`, `.fdignore`, and hides hidden files -- use `-H` and `-I` to override.
- `--exec` runs commands in parallel across CPU cores; use `--exec-batch` for commands that handle multiple arguments at once.
- The placeholder `{.}` (path without extension) is extremely useful for file conversion tasks like `fd -e png --exec convert {} {.}.jpg`.
- Use `-0` with `xargs -0` for filenames containing spaces or special characters instead of piping directly.
- fd is 5-50x faster than `find` for typical searches because it skips gitignored directories and uses parallelism.
- Combine `-e` (extension) with a regex pattern to narrow results efficiently: `fd -e go "test"` is faster than `fd "test.*\.go$"`.
- Use `fd --changed-within 1d` to quickly find files you or your team modified today.
- The `-l` flag shows results with permissions, size, and timestamps like `ls -l` -- helpful for auditing.
- Global ignore patterns can be set in `~/.config/fd/ignore` to always exclude directories like `node_modules`.
- Pipe fd output into fzf for interactive selection: `fd -e go | fzf --preview 'bat {}'`.

## See Also

- find, fzf, ripgrep, bat, xargs, bash

## References

- [fd GitHub Repository](https://github.com/sharkdp/fd)
- [fd README — Usage](https://github.com/sharkdp/fd#how-to-use)
- [fd README — Command-line Options](https://github.com/sharkdp/fd#command-line-options)
- [Arch Wiki — fd](https://wiki.archlinux.org/title/Fd)
- [Ubuntu Manpage — fd-find](https://manpages.ubuntu.com/manpages/noble/man1/fdfind.1.html)
- [fd vs find Comparison](https://github.com/sharkdp/fd#benchmark)
