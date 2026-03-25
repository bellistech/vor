# rsync (Remote Sync)

Fast, versatile file copying — synchronizes files locally or over SSH with delta transfers.

## Local Sync

### Copy directories
```bash
rsync -av /src/dir/ /dst/dir/         # archive mode, verbose
rsync -av /src/dir /dst/              # copy dir itself (not just contents)
# Trailing slash on source matters:
#   /src/dir/  → copies contents of dir into /dst/dir/
#   /src/dir   → copies dir itself, creating /dst/dir/dir/
```

### Dry run
```bash
rsync -avn /src/ /dst/                # show what would be done
rsync -avn --delete /src/ /dst/       # show what would be deleted too
```

## Remote Sync (over SSH)

### Push to remote
```bash
rsync -avz /local/dir/ user@host:/remote/dir/
rsync -avz -e 'ssh -p 2222' /local/dir/ user@host:/remote/dir/  # custom SSH port
rsync -avz -e 'ssh -i ~/.ssh/deploy_key' /local/ user@host:/remote/
```

### Pull from remote
```bash
rsync -avz user@host:/remote/dir/ /local/dir/
```

## Archive Mode (-a)

### What -a includes
```bash
# -a is equivalent to -rlptgoD:
#   -r  recursive
#   -l  copy symlinks as symlinks
#   -p  preserve permissions
#   -t  preserve modification times
#   -g  preserve group
#   -o  preserve owner (requires root)
#   -D  preserve device and special files
```

## Compression and Progress

### Transfer options
```bash
rsync -avz /src/ /dst/                # compress during transfer
rsync -av --progress /src/ /dst/      # show per-file progress
rsync -av --info=progress2 /src/ /dst/  # overall progress (single line)
rsync -avP /src/ /dst/                # -P = --progress + --partial
```

## Exclude and Include

### Exclude patterns
```bash
rsync -av --exclude='*.log' /src/ /dst/
rsync -av --exclude='node_modules' --exclude='.git' /src/ /dst/
rsync -av --exclude-from=exclude-list.txt /src/ /dst/
```

### Include/exclude combos
```bash
# Only sync .go files
rsync -av --include='*/' --include='*.go' --exclude='*' /src/ /dst/

# Sync everything except build artifacts
rsync -av --exclude='build/' --exclude='dist/' --exclude='*.o' /src/ /dst/
```

## Delete Options

### Mirror source to destination
```bash
rsync -av --delete /src/ /dst/              # delete files in dst not in src
rsync -av --delete-after /src/ /dst/        # delete after transfer (safer)
rsync -av --delete-excluded /src/ /dst/     # also delete excluded files in dst
```

### Safe delete
```bash
rsync -av --delete --backup --backup-dir=/backups/$(date +%F) /src/ /dst/
```

## Partial and Resume

### Resume interrupted transfers
```bash
rsync -avP /src/large-file.iso user@host:/dst/   # --partial + --progress
rsync -av --partial /src/ /dst/                   # keep partial files
rsync -av --partial-dir=.rsync-partial /src/ /dst/  # store partials separately
```

## Bandwidth Limit

### Throttle transfer speed
```bash
rsync -avz --bwlimit=5000 /src/ user@host:/dst/   # limit to 5000 KB/s
rsync -avz --bwlimit=1m /src/ user@host:/dst/      # 1 MB/s (rsync 3.2.3+)
```

## Checksums

### Force checksum comparison
```bash
rsync -avc /src/ /dst/                # use checksum instead of mtime+size
# Slower but catches files with same size/mtime but different content
```

## Permissions and Ownership

### Handle permissions
```bash
rsync -av --chmod=Du=rwx,Dg=rx,Do=rx,Fu=rw,Fg=r,Fo=r /src/ /dst/  # set explicit perms
rsync -av --no-perms /src/ /dst/      # don't sync permissions
rsync -av --no-owner --no-group /src/ /dst/  # skip owner/group
rsync -av --chown=www-data:www-data /src/ /dst/  # set owner:group
```

## Common Patterns

### Deploy a website
```bash
rsync -avz --delete --exclude='.git' --exclude='*.env' \
  ./build/ user@web-server:/var/www/html/
```

### Backup with hardlinks (incremental)
```bash
rsync -av --delete --link-dest=/backups/latest /src/ /backups/$(date +%F)/
ln -sfn /backups/$(date +%F) /backups/latest
```

### Sync only newer files
```bash
rsync -av --update /src/ /dst/        # skip files that are newer on dst
```

### List files without copying
```bash
rsync -avn /src/ /dst/ | head -50     # preview what would sync
rsync --list-only user@host:/remote/  # list remote directory
```

## Tips

- Trailing slash on the source path is critical: `dir/` copies contents, `dir` copies the directory itself
- `-z` (compression) helps over slow networks but wastes CPU on fast LANs or when files are already compressed
- `--delete` removes files from destination that don't exist in source — always dry-run (`-n`) first
- `--info=progress2` gives a single overall progress line instead of per-file noise
- `--partial` or `-P` is essential for large files over unreliable connections
- rsync uses mtime + file size by default to detect changes; use `-c` for checksum if clocks are unreliable
- `--link-dest` creates space-efficient incremental backups using hardlinks to unchanged files
- rsync over SSH is encrypted; rsync daemon protocol (rsync://) is not
- `--dry-run` (`-n`) is your safety net — always use it with `--delete` the first time
- For very large file sets, `--info=progress2` is much more useful than `--progress`
