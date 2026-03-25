# grep (pattern search)

Search file contents for lines matching a pattern.

## Basic Usage

### Simple Search

```bash
# Search a file
grep "error" /var/log/syslog

# Search multiple files
grep "error" /var/log/*.log

# Search stdin
cat /etc/passwd | grep deploy
```

## Regex Modes

### Basic, Extended, PCRE

```bash
# Basic regex (BRE) — default, must escape +, ?, |, ()
grep 'error\|warning' /var/log/syslog

# Extended regex (ERE) — no escaping needed for +, ?, |, ()
grep -E 'error|warning' /var/log/syslog

# Perl-compatible regex (PCRE) — lookaheads, \d, \w, etc.
grep -P '\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}' access.log

# Fixed string (no regex, literal match)
grep -F 'user.name[0]' config.json
```

## Recursive Search

### Search Directories

```bash
# Recursive
grep -r "TODO" /home/deploy/project/

# Recursive with line numbers
grep -rn "TODO" /home/deploy/project/

# Follow symlinks
grep -rL "pattern" /etc/
```

## Include and Exclude

### Filter Files

```bash
# Only search .go files
grep -rn "func main" --include="*.go" /home/deploy/project/

# Multiple includes
grep -rn "import" --include="*.py" --include="*.pyi" .

# Exclude a pattern
grep -rn "TODO" --exclude="*.min.js" .

# Exclude directories
grep -rn "TODO" --exclude-dir=vendor --exclude-dir=node_modules .

# Exclude binary files
grep -rI "pattern" .
```

## Output Control

### Context Lines

```bash
# 3 lines after match
grep -A 3 "error" /var/log/syslog

# 2 lines before match
grep -B 2 "error" /var/log/syslog

# 2 lines before and after
grep -C 2 "error" /var/log/syslog
```

### Match Formatting

```bash
# Line numbers
grep -n "error" /var/log/syslog

# Only filenames with matches
grep -rl "TODO" /home/deploy/project/

# Only filenames without matches
grep -rL "TODO" /home/deploy/project/

# Count matches per file
grep -rc "error" /var/log/

# Only the matching part (not the whole line)
grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+' access.log

# Color highlight
grep --color=auto "error" /var/log/syslog
```

## Invert Match

### Lines That Do NOT Match

```bash
# Exclude lines matching pattern
grep -v "^#" /etc/ssh/sshd_config

# Chain: remove comments and blank lines
grep -v "^#" /etc/ssh/sshd_config | grep -v "^$"
```

## Case Sensitivity

### Ignore Case

```bash
grep -i "error" /var/log/syslog

# Case-insensitive recursive search
grep -ri "exception" /var/log/
```

## Word and Line Matching

### Whole Words and Lines

```bash
# Match whole word only (not "errors" or "myerror")
grep -w "error" /var/log/syslog

# Match entire line
grep -x "exactly this line" file.txt
```

## Common Patterns

### Practical Examples

```bash
# Find non-comment, non-blank config lines
grep -v -E '^\s*(#|$)' /etc/ssh/sshd_config

# Extract email addresses
grep -oP '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}' file.txt

# Find processes (exclude the grep itself)
ps aux | grep '[n]ginx'

# Count occurrences across files
grep -rc "error" /var/log/ | grep -v ':0$'
```

## Tips

- `grep -P` (PCRE) is not available on macOS default grep -- install GNU grep via `brew install grep` (as `ggrep`).
- `grep '[n]ginx'` is a trick to avoid matching the grep process itself in `ps | grep` output.
- `-I` (capital I) skips binary files -- essential for recursive searches to avoid garbled output.
- `--color=auto` only colorizes when output is a terminal; safe to put in an alias.
- `grep -r` follows symlinks on some versions; use `grep -r --no-dereference` to avoid loops.
- For large codebases, `ripgrep` (`rg`) is dramatically faster than `grep -r` and respects `.gitignore`.

## References

- [man grep(1)](https://man7.org/linux/man-pages/man1/grep.1.html)
- [man regex(7)](https://man7.org/linux/man-pages/man7/regex.7.html)
- [GNU Grep Manual](https://www.gnu.org/software/grep/manual/grep.html)
- [GNU Grep — Regular Expressions](https://www.gnu.org/software/grep/manual/grep.html#Regular-Expressions)
- [GNU Grep — Performance](https://www.gnu.org/software/grep/manual/grep.html#Performance)
- [Arch Wiki — grep](https://wiki.archlinux.org/title/grep)
- [Ubuntu Manpage — grep](https://manpages.ubuntu.com/manpages/noble/man1/grep.1.html)
- [ripgrep (rg) — Faster Alternative](https://github.com/BurntSushi/ripgrep)
