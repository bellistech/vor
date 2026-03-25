# tar (tape archive)

Create, extract, and manipulate archive files.

## Create Archives

### Basic Creation

```bash
# Create a tar archive
tar cf archive.tar /path/to/directory/

# Create with verbose output
tar cvf archive.tar /path/to/directory/

# Create with gzip compression
tar czf archive.tar.gz /path/to/directory/

# Create with bzip2 compression
tar cjf archive.tar.bz2 /path/to/directory/

# Create with xz compression (best ratio)
tar cJf archive.tar.xz /path/to/directory/

# Create with zstd compression (fast + good ratio)
tar --zstd -cf archive.tar.zst /path/to/directory/
```

### Exclude Patterns

```bash
# Exclude a directory
tar czf project.tar.gz --exclude='.git' --exclude='node_modules' project/

# Exclude by pattern
tar czf backup.tar.gz --exclude='*.log' --exclude='*.tmp' /var/data/

# Exclude from a file
tar czf backup.tar.gz --exclude-from=.tarignore /var/data/
```

## Extract Archives

### Basic Extraction

```bash
# Extract (auto-detects compression)
tar xf archive.tar.gz

# Extract to a specific directory
tar xf archive.tar.gz -C /opt/

# Extract with verbose
tar xvf archive.tar.gz

# Extract specific files
tar xf archive.tar.gz path/to/file.txt

# Extract specific directory
tar xf archive.tar.gz project/src/
```

### Strip Leading Path Components

```bash
# Remove the first directory level
tar xf archive.tar.gz --strip-components=1 -C /opt/app/

# Useful when archive contains project-v1.2.3/ as top-level dir
```

## List Contents

### View Without Extracting

```bash
# List all files
tar tf archive.tar.gz

# List with details (permissions, size, date)
tar tvf archive.tar.gz

# List and grep
tar tf archive.tar.gz | grep config
```

## Preserve Permissions

### Ownership and Permissions

```bash
# Preserve permissions and ownership (default for root)
tar czf backup.tar.gz -p /etc/

# Extract preserving permissions
tar xpf backup.tar.gz

# Preserve numeric owner/group IDs
tar czf backup.tar.gz --numeric-owner /var/data/
tar xf backup.tar.gz --numeric-owner
```

## Incremental Backups

### Level-Based Backups

```bash
# Full backup (creates snapshot file)
tar czf full-backup.tar.gz -g /var/backup/snapshot.snar /var/data/

# Incremental backup (uses same snapshot)
tar czf incr-backup-1.tar.gz -g /var/backup/snapshot.snar /var/data/

# Restore: apply full then each incremental in order
tar xzf full-backup.tar.gz -g /dev/null -C /
tar xzf incr-backup-1.tar.gz -g /dev/null -C /
```

## Append and Update

### Modify Archives

```bash
# Append files to uncompressed archive
tar rf archive.tar newfile.txt

# Update only newer files in uncompressed archive
tar uf archive.tar /path/to/directory/

# Note: append and update do not work with compressed archives
```

## Piping

### Stream Archives

```bash
# Create and pipe to remote host
tar czf - /var/data/ | ssh remote_host "tar xzf - -C /opt/backup/"

# Pipe through pv for progress
tar cf - /var/data/ | pv | gzip > backup.tar.gz
```

## Tips

- `tar czf` = create + gzip + filename. The `f` must come last if you chain single-letter flags (it takes the filename as its argument).
- Modern tar auto-detects compression on extraction -- `tar xf` works for `.gz`, `.bz2`, `.xz`, and `.zst` without specifying the flag.
- `--strip-components=1` is essential when extracting release tarballs that wrap everything in a versioned directory.
- Always use `-C` for extraction destination rather than `cd`ing first -- it is safer in scripts.
- `tar` does not follow symlinks by default. Use `-h` to archive the targets instead.
- Compression comparison: `gzip` is fastest, `xz` has the best ratio, `zstd` is the best balance of speed and ratio.
- `-p` (preserve permissions) is on by default when running as root but not as a regular user.

## References

- [GNU Tar Manual](https://www.gnu.org/software/tar/manual/)
- [tar(1) Man Page](https://man7.org/linux/man-pages/man1/tar.1.html)
- [GNU Tar Operations](https://www.gnu.org/software/tar/manual/html_section/tar_toc.html)
- [pax (POSIX Archiver) Specification](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/pax.html)
- [bsdtar (libarchive)](https://www.libarchive.org/)
- [GNU Tar Incremental Backups](https://www.gnu.org/software/tar/manual/html_node/Incremental-Dumps.html)
