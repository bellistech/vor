# The Mathematics of xz — LZMA2 Compression Internals

> *xz uses LZMA2 (Lempel-Ziv-Markov chain Algorithm). The math covers dictionary sizing, match finder complexity, range coding efficiency, and the extreme memory trade-offs of high-ratio compression.*

---

## 1. LZMA2 Algorithm — Three-Stage Pipeline

### The Model

$$\text{LZMA2} = \text{Range Coding}(\text{Markov Model}(\text{LZ77 Output}))$$

**Stage 1 — LZ77:** Find repeated sequences using a large dictionary (up to 1.5 GiB).

**Stage 2 — Markov chain:** Context-dependent probability modeling for match/literal decisions.

**Stage 3 — Range coding:** Arithmetic coding with probabilities from the Markov model.

### vs DEFLATE (gzip)

| Feature | DEFLATE (gzip) | LZMA2 (xz) |
|:---|:---|:---|
| Dictionary | 32 KiB | Up to 1.5 GiB |
| Entropy coder | Huffman | Range coder |
| Context model | None | Markov chain |
| Compression ratio | 2-5x | 4-10x |
| Compression speed | 10-100 MB/s | 1-30 MB/s |

---

## 2. Dictionary Size — Memory Trade-off

### The Model

LZMA2's dictionary holds previously seen data for match finding. Larger dictionary = better compression but more memory.

### Memory Formula

$$\text{Compression Memory} = 3 \times \text{Dictionary Size} + 20 \text{ MiB (base)}$$

$$\text{Decompression Memory} = \text{Dictionary Size} + 1 \text{ MiB}$$

### Preset Levels

| Level | Dictionary | Compress RAM | Decompress RAM | Ratio (text) |
|:---:|:---:|:---:|:---:|:---:|
| -0 | 256 KiB | 3 MiB | 1 MiB | 3x |
| -1 | 1 MiB | 9 MiB | 2 MiB | 4x |
| -2 | 2 MiB | 17 MiB | 3 MiB | 5x |
| -3 | 4 MiB | 32 MiB | 5 MiB | 5.5x |
| -4 | 4 MiB | 48 MiB | 5 MiB | 5.8x |
| -5 | 8 MiB | 94 MiB | 9 MiB | 6x |
| -6 (default) | 8 MiB | 94 MiB | 9 MiB | 6.2x |
| -7 | 16 MiB | 186 MiB | 17 MiB | 6.5x |
| -8 | 32 MiB | 370 MiB | 33 MiB | 6.7x |
| -9 | 64 MiB | 674 MiB | 65 MiB | 6.9x |

### Extreme (-e) Mode

Adding `-e` enables additional match finders at higher CPU cost:

$$T_{-e} \approx 3-6 \times T_{normal} \quad (\text{for ~2-5\% better ratio})$$

---

## 3. Match Finder Algorithms

### The Model

LZMA2 uses several match finding algorithms, selected by preset:

| Algorithm | Complexity | Memory | Used In |
|:---|:---:|:---:|:---|
| Hash Chain (hc3, hc4) | O(n × chain_depth) | Low | Levels 1-3 |
| Binary Tree (bt2, bt3, bt4) | O(n × log n) | Higher | Levels 4-9 |

### Hash Chain

$$T_{search} = O(d) \quad \text{where } d = \text{chain depth (nice_len dependent)}$$

$$\text{Nice Match Length: } 8 \text{ (level 1)} \rightarrow 273 \text{ (level 9)}$$

Chain depth controls the trade-off: deeper = better matches = slower.

### Binary Tree

$$T_{search} = O(\log n) \text{ per position (amortized)}$$

$$\text{Total:} \quad O(N \times \log N) \text{ where } N = \text{input size}$$

The BT4 (binary tree with 4-byte hashing) gives the best compression.

---

## 4. Range Coding — Near-Optimal Entropy

### The Model

Range coding is a form of arithmetic coding that approaches the theoretical entropy limit.

### Entropy Bound

$$\text{Range Coded Size} \geq H(\text{data}) = -\sum p_i \log_2(p_i) \text{ bits}$$

Range coding typically achieves within 0.01 bits/symbol of entropy (vs Huffman's 0.1-1 bit gap).

### Comparison with Huffman

| Encoding | Overhead vs Entropy | For 1 GiB File |
|:---|:---:|:---:|
| Huffman (DEFLATE) | +0.1-1.0 bits/symbol | +12-120 MiB |
| Range (LZMA) | +0.001-0.01 bits/symbol | +0.12-1.2 MiB |

### Why This Matters

For 1 GiB of English text (entropy ~4.5 bits/symbol):

$$\text{Huffman:} \frac{4.6}{8} = 57.5\% \text{ of original}$$

$$\text{Range:} \frac{4.501}{8} = 56.3\% \text{ of original}$$

The 1.2% difference seems small, but combined with the larger dictionary, LZMA achieves 30-60% better compression than DEFLATE.

---

## 5. Threaded Compression — Block Splitting

### The Model

xz supports multi-threaded compression by splitting input into blocks:

$$\text{Threads} = \min(\text{CPU Cores}, \lceil \frac{\text{Input Size}}{\text{Block Size}} \rceil)$$

### Block Size Impact

$$\text{Compression Ratio Loss} \approx \frac{\text{Dictionary Size}}{\text{Block Size}} \quad (\text{per block boundary, context lost})$$

| Block Size | Threads (1 GiB input) | Ratio Loss | Speed Gain |
|:---:|:---:|:---:|:---:|
| Unlimited (1 thread) | 1 | 0% | 1x |
| 64 MiB | 16 | ~0.5% | ~12x |
| 16 MiB | 64 | ~2% | ~40x |
| 4 MiB | 256 | ~5% | ~100x |

### Multi-threaded Memory

$$\text{Total Memory} = \text{Threads} \times \text{Per-Thread Memory}$$

| Threads | Level -6 | Total RAM |
|:---:|:---:|:---:|
| 1 | 94 MiB | 94 MiB |
| 4 | 94 MiB | 376 MiB |
| 8 | 94 MiB | 752 MiB |
| 16 | 94 MiB | 1.5 GiB |

---

## 6. Format Comparison — xz vs Alternatives

### Compression Ratio (1 GiB Linux kernel source)

| Format | Compressed Size | Ratio | Compress Time | Decompress Time |
|:---|:---:|:---:|:---:|:---:|
| gzip -6 | 180 MiB | 5.7x | 15s | 4s |
| bzip2 -9 | 140 MiB | 7.3x | 60s | 15s |
| xz -6 | 110 MiB | 9.3x | 90s | 5s |
| xz -9e | 100 MiB | 10.2x | 300s | 5s |
| zstd -19 | 115 MiB | 8.9x | 120s | 2s |

### When to Use xz

| Scenario | Best Choice | Reason |
|:---|:---|:---|
| Software distribution | xz | Best ratio, fast decompress |
| Log rotation | gzip or zstd | Fast compress matters |
| Backups | zstd | Balance of speed and ratio |
| Maximum compression | xz -9e | Smallest possible size |
| Real-time streaming | gzip -1 or zstd -1 | Speed over ratio |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $3 \times \text{Dict} + 20\text{M}$ | Linear | Compression memory |
| $\text{Dict} + 1\text{M}$ | Linear | Decompression memory |
| $-\sum p_i \log_2(p_i)$ | Entropy | Theoretical limit |
| $O(N \log N)$ | Linearithmic | BT4 match finding |
| $\text{Threads} \times \text{Per-Thread}$ | Linear scaling | Multi-thread memory |
| $\frac{\text{Dict}}{\text{Block}}$ | Ratio | Block boundary ratio loss |

---

*Every `xz -6`, `tar -J`, and Linux kernel `make xzImage` runs this algorithm — LZMA2 in a large dictionary with range coding, trading extreme CPU time during compression for near-optimal file sizes and fast decompression.*
