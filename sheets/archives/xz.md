# xz (LZMA2 compression)

High-ratio compression using the LZMA2 algorithm.

## Compress

### Basic Compression

```bash
# Compress (replaces original with .xz)
xz data.bin

# Keep original file
xz -k data.bin

# Compress multiple files
xz file1.txt file2.txt
```

### Compression Level

```bash
# Fast compression (less ratio)
xz -1 data.bin
xz --fast data.bin

# Best compression (very slow, high memory)
xz -9 data.bin
xz --best data.bin

# Extreme mode (even slower, marginally better)
xz -9e data.bin

# Default is -6
```

### Multi-threaded

```bash
# Use all available cores
xz -T0 data.bin

# Use specific number of threads
xz -T4 data.bin

# Threaded with compression level
xz -T0 -6 data.bin
```

## Decompress

### Uncompress Files

```bash
# Decompress (replaces .xz with original)
xz -d data.bin.xz

# Same as xz -d
unxz data.bin.xz

# Decompress keeping the .xz file
xz -dk data.bin.xz
```

## Stdout

### Pipe and Redirect

```bash
# Compress to stdout
xz -c data.bin > data.bin.xz

# Decompress to stdout
xz -dc data.bin.xz
xzcat data.bin.xz

# Read compressed files
xzcat data.bin.xz | head
xzgrep "pattern" data.txt.xz
xzless data.txt.xz
```

## List and Test

### Inspect and Verify

```bash
# Show compression info
xz -l data.bin.xz

# Verbose info (ratio, memory, flags)
xz -lv data.bin.xz

# Test integrity
xz -t data.bin.xz

# Test with verbose
xz -tv data.bin.xz
```

## Tips

- `xz` produces significantly smaller files than gzip (~30% smaller) but is 5-10x slower to compress. Decompression speed is comparable.
- `-T0` (multi-threaded) is essential for large files. Single-threaded xz on a multi-GB file takes a very long time.
- `-9` uses ~674MB of RAM per thread. On memory-constrained systems, use `-6` (default, ~94MB) or lower.
- `-9e` (extreme) is rarely worth it -- it adds substantial time for 1-3% size improvement.
- `pixz` and `pxz` are parallel xz alternatives if your version of xz does not support `-T`.
- `xz` is the standard for kernel tarballs and package distribution where compression ratio matters more than speed.
- For a balance of speed and ratio, consider `zstd` instead -- it compresses faster than gzip with ratios approaching xz.

## See Also

- tar
- gzip
- zip
- 7z

## References

- [XZ Utils Home Page](https://tukaani.org/xz/)
- [xz(1) Man Page](https://man7.org/linux/man-pages/man1/xz.1.html)
- [XZ Utils Documentation](https://tukaani.org/xz/format.html)
- [LZMA SDK](https://www.7-zip.org/sdk.html)
- [XZ Utils GitHub Repository](https://github.com/tukaani-project/xz)
- [pixz (Parallel Indexed XZ)](https://github.com/vasi/pixz)
