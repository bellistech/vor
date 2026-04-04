# The Mathematics of tar — Tape Archive Format Internals

> *tar is an archival format (no compression). The math covers block structure, padding overhead, header checksums, and the performance characteristics of sequential archive access.*

---

## 1. Block Structure — The 512-Byte Unit

### The Model

tar operates in 512-byte blocks. Every file header and data segment is padded to 512-byte boundaries.

### Padding Formula

$$\text{Padded Size} = \lceil \frac{\text{File Size}}{512} \rceil \times 512$$

$$\text{Padding Waste} = \text{Padded Size} - \text{File Size}$$

$$\text{Average Waste per File} = \frac{512}{2} = 256 \text{ bytes}$$

### Worked Examples

| File Size | Blocks | Padded Size | Waste |
|:---:|:---:|:---:|:---:|
| 1 byte | 1 | 512 bytes | 511 bytes (99.8%) |
| 100 bytes | 1 | 512 bytes | 412 bytes (80.5%) |
| 512 bytes | 1 | 512 bytes | 0 bytes (0%) |
| 1,000 bytes | 2 | 1,024 bytes | 24 bytes (2.3%) |
| 10,000 bytes | 20 | 10,240 bytes | 240 bytes (2.3%) |
| 1 MiB | 2,048 | 1 MiB | 0 bytes (0%) |

### Total Archive Size

$$\text{Archive Size} = \sum_{i=1}^{n} (512 + \text{Padded Data}_i) + 1024$$

Where:
- 512 = header block per file
- Padded Data = data blocks
- 1024 = two zero-filled end-of-archive blocks

---

## 2. Header Format — The 512-Byte Header

### UStar Header Layout

| Field | Offset | Size | Content |
|:---|:---:|:---:|:---|
| name | 0 | 100 | Filename |
| mode | 100 | 8 | File permissions (octal) |
| uid | 108 | 8 | Owner user ID (octal) |
| gid | 116 | 8 | Owner group ID (octal) |
| size | 124 | 12 | File size (octal) |
| mtime | 136 | 12 | Modification time (octal) |
| checksum | 148 | 8 | Header checksum |
| typeflag | 156 | 1 | File type |
| linkname | 157 | 100 | Link target |
| magic | 257 | 6 | "ustar\0" |
| version | 263 | 2 | "00" |
| uname | 265 | 32 | Owner username |
| gname | 297 | 32 | Owner group name |
| devmajor | 329 | 8 | Device major |
| devminor | 337 | 8 | Device minor |
| prefix | 345 | 155 | Path prefix |
| padding | 500 | 12 | Unused |

### Maximum File Size (UStar)

Size field is 11 octal digits:

$$\text{Max Size} = 8^{11} - 1 = 8,589,934,591 \text{ bytes} \approx 8 \text{ GiB}$$

### GNU tar Extensions

$$\text{Max Size (GNU)} = 2^{63} - 1 \text{ bytes} \approx 8 \text{ EiB (binary encoding)}$$

### Header Checksum

$$\text{Checksum} = \sum_{i=0}^{511} \text{header}[i] \quad (\text{treating checksum field as spaces (0x20)})$$

---

## 3. Archive Overhead Analysis

### Per-File Overhead

$$\text{Overhead per File} = 512 \text{ (header)} + \text{Padding (avg 256 bytes)}$$

$$\text{Avg Overhead} \approx 768 \text{ bytes per file}$$

### Overhead Ratio

$$\text{Overhead \%} = \frac{n \times 768}{n \times 768 + \sum \text{File Sizes}} \times 100$$

| Files | Avg File Size | Total Data | Archive Overhead | Overhead % |
|:---:|:---:|:---:|:---:|:---:|
| 100 | 1 MiB | 100 MiB | 75 KiB | 0.07% |
| 10,000 | 100 KiB | 977 MiB | 7.3 MiB | 0.75% |
| 100,000 | 4 KiB | 391 MiB | 73.2 MiB | 15.8% |
| 1,000,000 | 100 bytes | 95 MiB | 732 MiB | 88.5% |

**Key insight:** tar overhead is negligible for large files but dominant for millions of tiny files.

---

## 4. Sequential Access — No Random Access

### The Model

tar has no index. Finding a file requires scanning from the beginning.

### Search Time

$$T_{find} = O(n) \quad \text{where } n = \text{number of files before target}$$

$$T_{find} = \frac{\text{Archive Bytes Before Target}}{\text{Read Speed}}$$

### Comparison with Indexed Formats

| Operation | tar | zip | Speedup |
|:---|:---:|:---:|:---:|
| List all files | O(archive size) | O(central directory) | 10-1000x |
| Extract one file | O(archive size) | O(1) seek + O(file size) | 10-1000x |
| Extract all files | O(archive size) | O(archive size) | ~1x |
| Create archive | O(total data) | O(total data) | ~1x |

### Streaming Advantage

tar's sequential format enables **streaming** (pipe-based processing):

$$\text{Memory Required} = \text{Buffer Size} \quad (\text{not archive size})$$

This is why `tar czf` pipes through gzip without needing the whole archive in memory.

---

## 5. Compression Pipeline — tar + compressor

### The Model

tar itself doesn't compress. It's piped through a compressor:

$$\text{tar.gz:} \quad \text{tar} \rightarrow \text{gzip}$$

$$\text{tar.xz:} \quad \text{tar} \rightarrow \text{xz}$$

$$\text{tar.zst:} \quad \text{tar} \rightarrow \text{zstd}$$

### Compression Comparison (1 GiB of source code)

| Format | Ratio | Compress Time | Decompress Time | Result Size |
|:---|:---:|:---:|:---:|:---:|
| .tar (none) | 1.0x | 3s | 3s | 1 GiB |
| .tar.gz (level 6) | 4.0x | 33s | 8s | 256 MiB |
| .tar.gz (level 9) | 4.2x | 100s | 8s | 244 MiB |
| .tar.bz2 | 5.0x | 120s | 30s | 205 MiB |
| .tar.xz (level 6) | 6.0x | 180s | 10s | 170 MiB |
| .tar.zst (level 3) | 4.5x | 10s | 5s | 228 MiB |
| .tar.zst (level 19) | 5.5x | 200s | 5s | 186 MiB |

### Compression throughput

$$T_{total} = \max(T_{tar\_read}, T_{compress})$$

If disk read is faster than compression: CPU-bound.
If compression is faster than disk: I/O-bound.

---

## 6. Record Size and Blocking Factor

### The Model

tar groups blocks into records for I/O efficiency:

$$\text{Record Size} = \text{Blocking Factor} \times 512$$

Default blocking factor = 20, so record size = 10,240 bytes.

### I/O Efficiency

$$\text{I/O Operations} = \frac{\text{Archive Size}}{\text{Record Size}}$$

| Blocking Factor | Record Size | I/Os for 1 GiB |
|:---:|:---:|:---:|
| 1 | 512 bytes | 2,097,152 |
| 20 (default) | 10 KiB | 104,858 |
| 128 | 64 KiB | 16,384 |
| 512 | 256 KiB | 4,096 |

Larger blocking factor = fewer I/O operations = faster on tape and HDD.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\lceil \frac{S}{512} \rceil \times 512$ | Ceiling alignment | Block padding |
| $n \times 768$ | Linear | Archive overhead |
| $O(n)$ | Linear scan | File lookup |
| $\frac{\text{Size}}{\text{Record}}$ | Division | I/O operations |
| $8^{11} - 1$ | Exponential | UStar max file size |
| $\sum \text{header}[i]$ | Summation | Header checksum |

---

*Every `tar czf`, `tar xf`, and `tar tf` operates on this 512-byte block format — a 1979 design for magnetic tape that became the universal archival format for Unix systems because its simplicity enables streaming.*

## Prerequisites

- File system concepts (inodes, permissions, ownership, symlinks)
- Binary data alignment (block sizes, padding)
- Octal number representation (tar headers use octal)

## Complexity

- **Beginner:** Create/extract archives, compression flags (-z, -j, -J)
- **Intermediate:** Incremental backups, exclude patterns, --strip-components, piping over SSH
- **Advanced:** 512-byte block structure internals, POSIX.1-2001 (pax) extended headers, sparse file handling
