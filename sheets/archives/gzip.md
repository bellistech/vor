# gzip (GNU zip compression)

Compress and decompress files using the gzip algorithm.

## Compress

### Basic Compression

```bash
# Compress a file (replaces original with .gz)
gzip access.log

# Compress keeping the original
gzip -k access.log

# Compress multiple files (each gets its own .gz)
gzip file1.txt file2.txt file3.txt

# Compress all files in current directory
gzip *
```

### Compression Level

```bash
# Fastest compression (least ratio)
gzip -1 access.log
gzip --fast access.log

# Best compression (slowest)
gzip -9 access.log
gzip --best access.log

# Default is -6
gzip -6 access.log
```

## Decompress

### Uncompress Files

```bash
# Decompress (replaces .gz with original)
gzip -d access.log.gz

# Same as gzip -d
gunzip access.log.gz

# Decompress keeping the .gz file
gzip -dk access.log.gz
gunzip -k access.log.gz
```

## Stdout

### Pipe and Redirect

```bash
# Compress to stdout (original unchanged)
gzip -c access.log > access.log.gz

# Decompress to stdout
gzip -dc access.log.gz
gunzip -c access.log.gz
zcat access.log.gz

# Read compressed file without decompressing
zcat access.log.gz | grep "error"
zgrep "error" access.log.gz
zless access.log.gz
```

## Test and List

### Verify and Inspect

```bash
# Test integrity
gzip -t access.log.gz

# Test with verbose (show details)
gzip -tv access.log.gz

# List compression info
gzip -l access.log.gz

# Verbose list (shows ratio, method)
gzip -lv access.log.gz
```

## Recursive

### Compress Directory Contents

```bash
# Compress all files in a directory recursively
gzip -r /var/log/old/

# Decompress all .gz files recursively
gzip -dr /var/log/old/
```

## Tips

- `gzip` replaces the original file by default. Use `-k` to keep it or `-c` to write to stdout.
- `zcat`, `zgrep`, `zless`, and `zdiff` let you work with gzip'd files without decompressing them first.
- `pigz` is a parallel gzip implementation that uses multiple cores -- drop-in replacement: `pigz -p 8 access.log`.
- Compression levels 1-6 are fast with diminishing returns. 7-9 are significantly slower for marginal improvement.
- `gzip` only compresses single files. To compress a directory, use `tar czf` to bundle and compress in one step.
- `.gz` files include a CRC32 checksum. Use `gzip -t` to verify integrity after transfer.

## References

- [GNU Gzip Manual](https://www.gnu.org/software/gzip/manual/)
- [gzip(1) Man Page](https://man7.org/linux/man-pages/man1/gzip.1.html)
- [zcat(1) Man Page](https://man7.org/linux/man-pages/man1/zcat.1.html)
- [zgrep(1) Man Page](https://man7.org/linux/man-pages/man1/zgrep.1.html)
- [pigz (Parallel gzip)](https://zlib.net/pigz/)
- [RFC 1952 -- GZIP File Format Specification](https://datatracker.ietf.org/doc/html/rfc1952)
- [zlib Library](https://zlib.net/)
