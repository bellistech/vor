# du (Disk Usage)

Estimate file and directory space usage.

## Basic Usage

```bash
du                                       # current directory, all subdirs
du /var/log                              # specific directory
du file.tar.gz                           # specific file
```

## Human-Readable Output

```bash
du -h /var/log                           # KiB, MiB, GiB
du -h --si /var/log                      # KB, MB, GB (powers of 1000)
```

## Summary (Total Only)

```bash
du -sh /var/log                          # total size of directory
du -sh /home/*                           # size of each item in /home
du -sh /var/log /var/cache /tmp          # multiple directories
```

## Depth Control

```bash
du -h --max-depth=1 /var                 # one level deep
du -h --max-depth=2 /home               # two levels deep
du -h -d 1 /var                          # short form (GNU and BSD)
```

## Sort by Size

```bash
# Largest directories first
du -sh /var/* | sort -rh

# Top 10 largest directories
du -h --max-depth=1 /var | sort -rh | head -10

# Find the largest directories anywhere
du -h --max-depth=3 / 2>/dev/null | sort -rh | head -20
```

## Exclude Patterns

```bash
# Exclude by pattern
du -sh --exclude="*.log" /var
du -sh --exclude="node_modules" /home/alice/projects

# Multiple exclusions
du -sh --exclude="*.log" --exclude="*.tmp" /var

# Exclude from file
du -sh --exclude-from=exclude-list.txt /var
```

## Specific File Types

```bash
# Size of all .log files
du -ch /var/log/*.log | tail -1          # -c adds grand total

# Size of all files matching a pattern (with find)
find /home -name "*.jpg" -exec du -ch {} + | tail -1
```

## Count Options

```bash
# Include files (not just directories)
du -ah /var/log                          # all files and dirs

# Grand total
du -ch /var/log /var/cache               # -c adds total line

# Apparent size (file size, not disk blocks)
du -sh --apparent-size /var/log

# Block size
du -B 1M /var/log                        # display in MiB
du -B 1G /var/log                        # display in GiB
```

## Cross-Filesystem Boundaries

```bash
# Stay on one filesystem (don't follow mount points)
du -shx /                                # only root filesystem

# This is very useful for finding what fills up /
du -hx --max-depth=1 / | sort -rh | head -15
```

## Dereference Symlinks

```bash
du -shL /var/log                         # follow symlinks
```

## Common Workflows

### Find What Fills a Disk

```bash
# Start broad, then drill down
du -hx --max-depth=1 / | sort -rh | head
du -hx --max-depth=1 /var | sort -rh | head
du -hx --max-depth=1 /var/log | sort -rh | head
```

### Compare Directory Sizes

```bash
du -sh /home/alice /home/bob /home/carol | sort -rh
```

### Size of Current Directory Only (No Subdirs)

```bash
du -sh .
```

## Tips

- `du -shx /` is the go-to command when a filesystem is full; `-x` prevents crossing into other mounts
- `sort -rh` sorts human-readable sizes correctly (GNU sort only); on older systems, use `sort -rn` with `-B 1M`
- `du` counts allocated disk blocks, not file size; sparse files and filesystem overhead cause discrepancies
- `--apparent-size` shows the file size as `ls -l` would; useful for estimating transfer sizes
- `du` traverses directories recursively by default, which can be slow on large filesystems; use `--max-depth` to limit
- Redirect stderr with `2>/dev/null` to suppress "Permission denied" noise when scanning system directories
- On macOS/BSD, use `-d` instead of `--max-depth` and note that `--exclude` is a GNU extension
- For interactive exploration, `ncdu` is significantly faster and more user-friendly than repeated `du` commands

## References

- [du(1) Man Page](https://man7.org/linux/man-pages/man1/du.1.html)
- [GNU Coreutils — du](https://www.gnu.org/software/coreutils/manual/html_node/du-invocation.html)
- [stat(2) Man Page](https://man7.org/linux/man-pages/man2/stat.2.html)
- [Arch Wiki — Disk Usage](https://wiki.archlinux.org/title/List_of_applications#Disk_usage_display)
- [Ubuntu Manpage — du](https://manpages.ubuntu.com/manpages/noble/man1/du.1.html)
