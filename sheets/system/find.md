# find (file search)

Search for files and directories in a directory hierarchy.

## By Name

### Name Matching

```bash
# Exact name
find /var/log -name "syslog"

# Wildcard (must quote to prevent shell expansion)
find /etc -name "*.conf"

# Case-insensitive
find /home -iname "readme*"

# Regex on full path
find /var -regex ".*\.log\.[0-9]+"
```

## By Type

### File Types

```bash
# Files only
find /etc -type f -name "*.conf"

# Directories only
find /home -type d -name ".ssh"

# Symlinks
find /usr -type l

# Empty files
find /tmp -type f -empty

# Empty directories
find /var -type d -empty
```

## By Size

### Size Filters

```bash
# Larger than 100MB
find / -type f -size +100M

# Smaller than 1KB
find /tmp -type f -size -1k

# Exactly 4096 bytes (one block)
find /tmp -type f -size 4096c

# Size units: c(bytes), k(KB), M(MB), G(GB)
```

## By Time

### Modification, Access, Change

```bash
# Modified in the last 24 hours
find /var/log -type f -mtime -1

# Modified more than 30 days ago
find /tmp -type f -mtime +30

# Modified in the last 60 minutes
find /var/log -type f -mmin -60

# Accessed in the last 7 days
find /home -type f -atime -7

# Changed (metadata) in last 2 days
find /etc -type f -ctime -2

# Newer than a reference file
find /var/log -type f -newer /var/log/syslog
```

## By Permissions

### Permission Filters

```bash
# World-writable files
find / -type f -perm -o=w

# SUID binaries
find / -type f -perm -4000

# SGID binaries
find / -type f -perm -2000

# Exact permissions
find /home -type f -perm 0644
```

## Executing Commands

### -exec and -delete

```bash
# Run a command on each result
find /tmp -type f -name "*.tmp" -exec rm {} \;

# Batch mode (faster, like xargs)
find /var/log -name "*.log" -exec gzip {} +

# Delete directly (faster than -exec rm)
find /tmp -type f -name "*.bak" -delete

# Print and delete
find /tmp -type f -mtime +30 -print -delete

# Prompt before each action
find /tmp -type f -name "*.tmp" -ok rm {} \;
```

### Pipe to xargs

```bash
# Handle filenames with spaces/special chars
find /home -type f -name "*.jpg" -print0 | xargs -0 ls -la

# Parallel execution
find /var/log -name "*.log" -print0 | xargs -0 -P 4 gzip

# Limit arguments per invocation
find /tmp -name "*.csv" -print0 | xargs -0 -n 10 wc -l
```

## Depth Control

### Max and Min Depth

```bash
# Only in current directory (no subdirectories)
find /etc -maxdepth 1 -type f -name "*.conf"

# At least 2 levels deep
find /home -mindepth 2 -name "*.ssh"

# Between 1 and 3 levels
find /var -mindepth 1 -maxdepth 3 -type f
```

## Combining Conditions

### AND, OR, NOT

```bash
# AND (implicit)
find /etc -type f -name "*.conf" -size +1k

# OR
find /tmp -name "*.log" -o -name "*.tmp"

# NOT
find /home -type f ! -name "*.txt"

# Grouping (must escape parens)
find /var \( -name "*.log" -o -name "*.gz" \) -mtime +30
```

## Exclude Directories

### Prune

```bash
# Skip .git directories
find /home/deploy/project -path '*/.git' -prune -o -type f -name "*.go" -print

# Skip multiple directories
find / -path /proc -prune -o -path /sys -prune -o -type f -name "*.conf" -print
```

## Tips

- Always quote glob patterns in `-name` to prevent the shell from expanding them before find sees them.
- `-delete` implies `-depth` (processes children before parents) and will not work with `-prune`.
- `-print0 | xargs -0` is the safe pattern for filenames containing spaces, newlines, or quotes.
- `-exec {} +` batches arguments like xargs and is significantly faster than `-exec {} \;` for many files.
- Put `-maxdepth` and `-mindepth` before other predicates -- find processes left-to-right.
- On macOS, `-regex` uses basic regex by default; use `-E` for extended regex.

## References

- [man find(1)](https://man7.org/linux/man-pages/man1/find.1.html)
- [man xargs(1)](https://man7.org/linux/man-pages/man1/xargs.1.html)
- [man locate(1)](https://man7.org/linux/man-pages/man1/locate.1.html)
- [GNU Findutils Manual](https://www.gnu.org/software/findutils/manual/html_mono/find.html)
- [GNU Findutils — Finding Files](https://www.gnu.org/software/findutils/manual/html_node/find_html/index.html)
- [Arch Wiki — Find](https://wiki.archlinux.org/title/Find)
- [Ubuntu Manpage — find](https://manpages.ubuntu.com/manpages/noble/man1/find.1.html)
- [Red Hat — Using find Command](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_file_systems/index)
