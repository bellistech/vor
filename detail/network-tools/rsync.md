# The Mathematics of rsync — Delta Transfer Algorithm Internals

> *rsync efficiently synchronizes files using a rolling checksum algorithm. The math covers the rsync delta algorithm, rolling hash computation, bandwidth savings, and transfer performance estimation.*

---

## 1. The rsync Algorithm — Rolling Checksum

### The Model

rsync's delta transfer uses a two-signature approach to find matching blocks between source and destination without transferring the entire file.

### Algorithm Steps

1. **Receiver:** Split destination file into blocks of size $S$, compute checksums
2. **Receiver:** Send checksums to sender (weak hash + strong hash per block)
3. **Sender:** Roll weak checksum across source file, compare with received checksums
4. **Sender:** Send only non-matching data + matching block references

### Rolling Checksum (Adler-32 variant)

$$a(k, l) = \sum_{i=k}^{l} b_i \mod M$$

$$c(k, l) = \sum_{i=k}^{l} (l - i + 1) \times b_i \mod M$$

$$\text{Rolling:} \quad a(k+1, l+1) = a(k, l) - b_k + b_{l+1}$$

$$\text{Rolling:} \quad c(k+1, l+1) = c(k, l) - (l - k + 1) \times b_k + a(k+1, l+1)$$

**Key property:** Rolling the hash forward by one byte is O(1), not O(block_size).

### Strong Checksum (MD5/xxHash)

$$\text{Collision probability} = \frac{1}{2^{128}} \text{ (MD5)} \quad | \quad \frac{1}{2^{128}} \text{ (xxHash128)}$$

---

## 2. Block Size Selection

### The Model

Block size determines the granularity of change detection.

### Default Block Size

$$S = \max(700, \sqrt{\text{file size}}) \quad (\text{rounded to power of 2 neighborhood})$$

### Block Size Trade-offs

$$\text{Checksum Data} = \frac{\text{File Size}}{S} \times (4 + 16) \text{ bytes}$$

Where 4 = weak hash, 16 = strong hash (MD5).

| File Size | Block Size | Blocks | Checksum Data |
|:---:|:---:|:---:|:---:|
| 1 MiB | 1 KiB | 1,024 | 20 KiB |
| 10 MiB | 3.2 KiB | 3,200 | 62.5 KiB |
| 100 MiB | 10 KiB | 10,240 | 200 KiB |
| 1 GiB | 32 KiB | 32,768 | 640 KiB |
| 10 GiB | 100 KiB | 102,400 | 2 MiB |

### Granularity vs Overhead

$$\text{Minimum Transfer} = \text{Changed Data} + \frac{\text{File Size}}{S} \times 20$$

Smaller blocks = better change detection = more checksum overhead.

---

## 3. Bandwidth Savings

### The Model

$$\text{Transfer Size} = \text{Checksum Data (receiver→sender)} + \text{Delta Data (sender→receiver)}$$

$$\text{Delta Data} = \text{Changed Blocks} \times S + \text{Unmatched Literal Data}$$

$$\text{Savings} = 1 - \frac{\text{Transfer Size}}{\text{File Size}}$$

### Worked Examples

| Scenario | File Size | Change | Transfer | Savings |
|:---|:---:|:---:|:---:|:---:|
| Config file edit (1 line) | 10 KiB | 100 bytes | ~300 bytes | 97% |
| Log rotation (append) | 100 MiB | 10 MiB append | ~10 MiB | 90% |
| Binary update (5% changed) | 500 MiB | 25 MiB | ~30 MiB | 94% |
| Database dump (80% same) | 10 GiB | 2 GiB | ~2.5 GiB | 75% |
| Completely new file | 1 GiB | 100% | 1 GiB + overhead | ~0% |

### Full Transfer vs Delta

$$\text{Speedup} = \frac{T_{full}}{T_{delta}} = \frac{\text{File Size} / \text{BW}}{\text{Delta Size} / \text{BW} + T_{checksum}}$$

---

## 4. Transfer Performance

### Performance Formula

$$T_{total} = T_{file\_list} + \sum_{i=1}^{n} (T_{checksum_i} + T_{delta_i} + T_{transfer_i})$$

### Pipeline Mode (--delay-updates, default)

rsync pipelines file operations:

$$T_{pipeline} = T_{file\_list} + \max(\sum T_{checksum}, \sum T_{delta}, \sum T_{transfer})$$

### Throughput Factors

| Factor | Impact | Mitigation |
|:---|:---|:---|
| Latency (RTT) | Each file needs round-trip | `--whole-file` for LAN |
| Small files | Per-file overhead dominates | Batch with tar |
| Checksum computation | CPU-bound | `--no-checksum` for initial |
| Compression | CPU for compress/decompress | `-z` for WAN, skip for LAN |

### Compression Savings

$$\text{Transfer with -z} = \frac{\text{Delta Data}}{\text{Compression Ratio}}$$

$$\text{Worth it if:} \quad \frac{T_{compress} + T_{transfer\_compressed}}{1} < T_{transfer\_raw}$$

$$\text{Break-even BW} = \frac{\text{Compression Ratio} - 1}{\text{Compression Ratio}} \times \text{CPU BW}$$

---

## 5. File List and Metadata

### File List Size

$$\text{File List} = n \times (\text{path length} + \text{metadata (32 bytes)})$$

| Files | Avg Path | File List Size | Transfer Time (1 Mbps) |
|:---:|:---:|:---:|:---:|
| 1,000 | 50 bytes | 80 KiB | 0.6s |
| 100,000 | 60 bytes | 8.8 MiB | 70s |
| 1,000,000 | 70 bytes | 97 MiB | 776s |

### --delete Performance

$$T_{delete} = \text{Missing Files} \times T_{unlink}$$

With `--delete-during` (default), deletion happens inline. With `--delete-before`, the full file list must be received first.

---

## 6. Incremental Recursion

### The Model

Since rsync 3.0, file lists are sent incrementally (not all at once).

$$\text{Memory (old)} = O(n) \quad (\text{entire file list in memory})$$

$$\text{Memory (incremental)} = O(\sqrt{n}) \quad (\text{directory at a time})$$

### Practical Impact

| Files | Old Memory | Incremental Memory |
|:---:|:---:|:---:|
| 10,000 | 1 MiB | 100 KiB |
| 1,000,000 | 100 MiB | 10 MiB |
| 10,000,000 | 1 GiB | 100 MiB |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $a(k+1) = a(k) - b_k + b_{l+1}$ | Rolling sum | Rolling checksum |
| $\frac{\text{Size}}{S} \times 20$ | Linear | Checksum overhead |
| $1 - \frac{\text{Delta}}{\text{Full}}$ | Ratio | Bandwidth savings |
| $\sqrt{\text{file size}}$ | Square root | Default block size |
| $\frac{\text{Delta}}{\text{Compression Ratio}}$ | Division | Compressed transfer |
| O(1) per byte rolled | Constant | Rolling hash step |

---

*Every `rsync -avz`, `rsync --progress`, and `rsync --dry-run` runs this algorithm — a 1996 invention (Andrew Tridgell's PhD thesis) that turned file synchronization from "copy everything" into "transfer only the differences" using rolling checksums.*
