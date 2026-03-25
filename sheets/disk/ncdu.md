# ncdu (NCurses Disk Usage)

Interactive, fast disk usage analyzer with a terminal UI.

## Basic Usage

```bash
ncdu /                                   # scan root filesystem
ncdu /var                                # scan specific directory
ncdu .                                   # scan current directory
ncdu                                     # defaults to current directory
```

## Interactive Navigation

```bash
# Arrow keys or vim-style bindings
Up/k       # move up
Down/j     # move down
Enter/Right # enter directory
Left/<     # go to parent directory

# Actions
d          # delete selected file/directory (with confirmation)
n          # sort by name
s          # sort by size (default)
C          # sort by item count
M          # sort by mtime
g          # toggle between graph/percent/both
e          # show/hide hidden files
t          # toggle dirs before files
q          # quit
?          # help
```

## Scan Options

### Stay on One Filesystem

```bash
ncdu -x /                                # don't cross mount boundaries
```

### Exclude Patterns

```bash
ncdu --exclude "*.log" /var
ncdu --exclude "node_modules" /home
ncdu --exclude ".git" ~/projects

# Multiple exclusions
ncdu --exclude "*.log" --exclude "*.tmp" /var

# Exclude from file
ncdu --exclude-from exclude-list.txt /home
```

### Follow Symlinks

```bash
ncdu -L /var                             # follow symlinks (default: skip)
```

### Count Hard Links Only Once

```bash
ncdu --exclude-kernfs /                  # exclude /proc, /sys, etc.
```

## Export and Import

### Export Scan to File (Scan Once, Browse Later)

```bash
# Scan and save results
ncdu -o scan.json /                      # JSON output

# Scan on remote server, analyze locally
ssh server "ncdu -o- -x /" > server-scan.json
```

### Import Saved Scan

```bash
ncdu -f scan.json                        # load and browse
ncdu -f server-scan.json
```

### Scan Without UI (Background or Remote)

```bash
# Scan only, no UI -- useful for cron or scripts
ncdu -1xo scan.json /

# -1 = quiet mode (shows single-line progress)
# -x = stay on one filesystem
# -o = output to file
```

## Display Options

### Quiet Scan (Reduce Output)

```bash
ncdu -q /var                             # less frequent refresh during scan
ncdu -qq /var                            # no scan progress output
```

### Show Apparent Size (Not Disk Usage)

```bash
ncdu --apparent-size /var
```

### Confirm Before Delete

```bash
ncdu --confirm-quit /                    # ask before quitting too
```

## Color Modes

```bash
ncdu --color dark /                      # dark terminal (default)
ncdu --color off /                       # no colors
```

## Common Workflows

### Find What Fills the Disk

```bash
# Quick scan of root, staying on one filesystem
ncdu -x /

# Then navigate interactively:
# Press 's' to sort by size
# Enter the largest directory
# Keep drilling down until you find the culprit
```

### Analyze Remote Server

```bash
# Run scan on remote, browse locally
ssh server "ncdu -o- -x /" | ncdu -f-

# Or save and transfer
ssh server "ncdu -o- -x /" > server.json
ncdu -f server.json
```

### Cleanup Old Logs

```bash
ncdu /var/log
# Navigate to large files, press 'd' to delete
# ncdu asks for confirmation before each delete
```

## Tips

- `ncdu -x /` is the fastest way to find what is eating disk space; `-x` keeps it to one filesystem
- Export scans with `-o` for servers you cannot interactively SSH into; import with `-f` on your workstation
- The delete function (`d`) is permanent; ncdu confirms before deletion but there is no undo
- ncdu version 2 (Rust rewrite) is significantly faster on large filesystems and supports JSON export by default
- Scanning `/` without `-x` will include `/proc`, `/sys`, and other virtual filesystems that skew results
- On minimal servers without ncdu, fall back to `du -hx --max-depth=1 / | sort -rh | head`
- ncdu loads the full directory tree into memory; for extremely large filesystems (tens of millions of files), memory usage can be significant
- Press `g` to cycle through graph bar, percentage, and combined views
