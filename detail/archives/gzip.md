# The Mathematics of gzip — DEFLATE Compression Internals

> *gzip uses the DEFLATE algorithm (LZ77 + Huffman coding). The math covers sliding window mechanics, Huffman tree construction, compression ratio vs level trade-offs, and throughput estimation.*

---

## 1. DEFLATE Algorithm — Two-Stage Compression

### The Model

DEFLATE combines two compression techniques in sequence:

$$\text{DEFLATE} = \text{Huffman Encoding}(\text{LZ77 Output})$$

**Stage 1 — LZ77:** Replace repeated byte sequences with (distance, length) back-references.

**Stage 2 — Huffman:** Encode the LZ77 output with variable-length bit codes based on frequency.

### Overall Compression

$$\text{Compression Ratio} = \frac{\text{Original Size}}{\text{Compressed Size}}$$

$$\text{Space Savings} = 1 - \frac{\text{Compressed}}{\text{Original}}$$

---

## 2. LZ77 Sliding Window

### The Model

LZ77 maintains a sliding window of previously seen data. When a match is found, it emits a (distance, length) pair instead of raw bytes.

### Window Parameters

$$\text{Window Size} = 32 \text{ KiB} = 32,768 \text{ bytes (gzip/DEFLATE)}$$

$$\text{Max Match Distance} = 32,768 \text{ bytes back}$$

$$\text{Max Match Length} = 258 \text{ bytes}$$

$$\text{Min Match Length} = 3 \text{ bytes}$$

### Match Encoding Savings

$$\text{Savings per Match} = \text{Match Length} - \text{Back-Reference Size}$$

A back-reference costs ~3 bytes (distance + length encoding):

$$\text{Net Savings} = \text{Length} - 3 \text{ bytes}$$

| Match Length | Raw Bytes | Back-Reference | Savings |
|:---:|:---:|:---:|:---:|
| 3 | 3 bytes | 3 bytes | 0 (break-even) |
| 4 | 4 bytes | 3 bytes | 1 byte |
| 10 | 10 bytes | 3 bytes | 7 bytes |
| 50 | 50 bytes | 3 bytes | 47 bytes |
| 258 (max) | 258 bytes | 3 bytes | 255 bytes |

### Hash Chain Lookup

gzip uses a hash table with chaining to find matches:

$$\text{Hash} = \text{hash}(\text{next 3 bytes}) \mod \text{Hash Size}$$

$$\text{Hash Size} = 32,768 \text{ entries (matching window size)}$$

---

## 3. Huffman Coding — Entropy Encoding

### The Model

Huffman coding assigns shorter bit codes to more frequent symbols.

### Shannon Entropy (Theoretical Minimum)

$$H = -\sum_{i=1}^{n} p_i \log_2(p_i) \quad \text{bits per symbol}$$

$$\text{Theoretical Minimum Size} = \frac{H \times \text{Original Size}}{8} \text{ bytes}$$

### Worked Example

*"File with 4 symbols: A (50%), B (25%), C (15%), D (10%)."*

$$H = -(0.5 \log_2 0.5 + 0.25 \log_2 0.25 + 0.15 \log_2 0.15 + 0.10 \log_2 0.10)$$

$$= -(−0.5 − 0.5 − 0.41 − 0.33) = 1.74 \text{ bits/symbol}$$

Fixed encoding: 2 bits/symbol. Huffman encoding: ~1.75 bits/symbol.

$$\text{Savings} = 1 - \frac{1.75}{2.0} = 12.5\%$$

### Huffman Tree for Example

| Symbol | Frequency | Code | Bits |
|:---:|:---:|:---:|:---:|
| A | 50% | 0 | 1 bit |
| B | 25% | 10 | 2 bits |
| C | 15% | 110 | 3 bits |
| D | 10% | 111 | 3 bits |

$$\text{Avg bits} = 0.5(1) + 0.25(2) + 0.15(3) + 0.10(3) = 1.75$$

---

## 4. Compression Level Trade-offs

### The Model

gzip levels 1-9 control the effort spent finding LZ77 matches. Higher levels search longer chains.

### Level Parameters

| Level | Hash Chain Length | Lazy Match | Speed | Ratio |
|:---:|:---:|:---:|:---:|:---:|
| 1 (fastest) | 4 | No | ~100 MB/s | ~2.0x |
| 2 | 5 | No | ~90 MB/s | ~2.1x |
| 3 | 6 | No | ~80 MB/s | ~2.2x |
| 4 | 8 | Yes | ~50 MB/s | ~2.5x |
| 5 | 16 | Yes | ~40 MB/s | ~2.7x |
| 6 (default) | 32 | Yes | ~30 MB/s | ~3.0x |
| 7 | 64 | Yes | ~20 MB/s | ~3.1x |
| 8 | 128 | Yes | ~15 MB/s | ~3.2x |
| 9 (best) | 4096 | Yes | ~10 MB/s | ~3.3x |

### Diminishing Returns

$$\frac{\Delta \text{Ratio}}{\Delta \text{Speed}} \text{ decreases rapidly after level 6}$$

| Level Jump | Ratio Improvement | Speed Cost |
|:---|:---:|:---:|
| 1 -> 6 | +50% (2.0x -> 3.0x) | 70% slower |
| 6 -> 9 | +10% (3.0x -> 3.3x) | 67% slower |

**Level 6 is the sweet spot** — 90% of the compression at 3x the speed of level 9.

---

## 5. Data Type Compression Ratios

### Typical Ratios by Content

| Data Type | Typical Ratio | Savings | Reason |
|:---|:---:|:---:|:---|
| English text | 3-4x | 67-75% | Repetitive patterns, skewed frequency |
| HTML/XML | 5-10x | 80-90% | Highly repetitive tags |
| JSON | 4-8x | 75-88% | Repetitive keys, whitespace |
| CSV | 3-6x | 67-83% | Repetitive delimiters, patterns |
| Log files | 5-15x | 80-93% | Repetitive formats |
| Source code | 3-5x | 67-80% | Keywords, indentation |
| Binary executables | 1.5-2.5x | 33-60% | Some patterns, mostly random |
| Already compressed (jpg, mp4) | 1.0-1.02x | 0-2% | No redundancy left |
| Random data | 1.0x (may grow) | 0% | Maximum entropy |

### gzip Header Overhead

$$\text{gzip Header} = 10 \text{ bytes (minimum)} + \text{Optional: filename, comment, extra fields}$$

$$\text{gzip Trailer} = 8 \text{ bytes (CRC32 + original size)}$$

For very small files:

$$\text{Overhead Ratio} = \frac{18}{\text{Original Size}}$$

| Original | Compressed | With gzip Overhead | Effective |
|:---:|:---:|:---:|:---:|
| 10 bytes | 15 bytes | 33 bytes | 3.3x bigger |
| 100 bytes | 70 bytes | 88 bytes | 1.14x smaller |
| 1 KiB | 400 bytes | 418 bytes | 2.5x smaller |
| 10 KiB | 3 KiB | 3,018 bytes | 3.4x smaller |

**Do not gzip files < ~150 bytes — the header overhead can make them larger.**

---

## 6. Decompression Performance

### The Model

Decompression is much faster than compression because it doesn't need to search for matches.

$$\text{Decompress Speed} \approx 3-5\times \text{Compress Speed (same level)}$$

| Level | Compress | Decompress | Ratio |
|:---:|:---:|:---:|:---:|
| 1 | 100 MB/s | 400 MB/s | 4x faster |
| 6 | 30 MB/s | 400 MB/s | 13x faster |
| 9 | 10 MB/s | 400 MB/s | 40x faster |

**Decompression speed is essentially constant** regardless of compression level — it only reads the stream.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $-\sum p_i \log_2(p_i)$ | Entropy (information theory) | Theoretical compression limit |
| $\text{Length} - 3$ | Subtraction | Match savings |
| $1 - \frac{\text{Compressed}}{\text{Original}}$ | Ratio | Space savings |
| $\frac{\Delta\text{Ratio}}{\Delta\text{Speed}}$ | Marginal analysis | Level selection |
| $\frac{18}{\text{Size}}$ | Reciprocal | Header overhead |
| 32 KiB window | Constant | LZ77 match distance |

---

*Every `gzip -6`, `zlib.compress()`, and HTTP `Content-Encoding: gzip` header runs this algorithm — LZ77 + Huffman coding in a 32 KiB sliding window, the same algorithm that compresses most of the internet's traffic.*

## Prerequisites

- Lempel-Ziv (LZ77) sliding window concept
- Huffman coding (variable-length prefix codes)
- CRC32 checksums for integrity verification

## Complexity

- **Beginner:** Compress/decompress single files, compression levels 1-9
- **Intermediate:** Piping via zcat/zgrep, parallel gzip (pigz), HTTP content encoding
- **Advanced:** DEFLATE algorithm internals, LZ77 match finding, Huffman tree construction, zlib window sizing
