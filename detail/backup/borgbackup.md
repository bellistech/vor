# The Mathematics of BorgBackup — Compression, Deduplication, and Encryption

> *BorgBackup combines content-defined chunking with Buzhash rolling hashes, HMAC-SHA256 chunk identification, authenticated encryption via AES-CTR-HMAC or AEAD, and multiple compression algorithms whose information-theoretic limits govern achievable backup ratios.*

---

## 1. Content-Defined Chunking (Buzhash)

### Buzhash Rolling Hash

Borg uses a Buzhash rolling hash over a sliding window of $w$ bytes to detect chunk boundaries:

$$H_i = \text{rot}(H_{i-1}, 1) \oplus h(b_{i}) \oplus \text{rot}(h(b_{i-w}), w)$$

where $h()$ maps bytes to random 32-bit values, $\text{rot}(x, k)$ is a bitwise left rotation by $k$, and $\oplus$ is XOR.

A boundary is declared when:

$$H_i \mod 2^k = 0$$

This gives an expected chunk size of:

$$E[\text{chunk size}] = 2^k$$

### Chunk Size Parameters

| Parameter | Borg Default | Formula |
|:---|:---:|:---|
| CHUNK_MIN_EXP | 19 ($2^{19}$ = 512 KiB) | Minimum chunk size |
| CHUNK_MAX_EXP | 23 ($2^{23}$ = 8 MiB) | Maximum chunk size |
| HASH_MASK_BITS | 21 ($2^{21}$ = 2 MiB) | Target average chunk |
| HASH_WINDOW_SIZE | 4095 bytes | Rolling hash window |

### Geometric Distribution of Chunk Sizes

Between min and max bounds, chunk sizes follow a truncated geometric distribution:

$$P(X = k) = \frac{(1-p)^{k - C_{min}} \cdot p}{1 - (1-p)^{C_{max} - C_{min}}}$$

where $p = 2^{-k}$ and $C_{min}, C_{max}$ are the chunk size bounds.

$$\text{Std Dev} \approx E[X] = 2^k \quad (\text{high variance})$$

---

## 2. Deduplication Model

### Chunk Identification

Each chunk is identified by a keyed HMAC:

$$\text{ChunkID} = \text{HMAC-SHA256}(K_{chunk}, \text{data})$$

This prevents an attacker from confirming whether specific plaintext exists in the repository (unlike plain SHA-256).

### Dedup Savings Formula

For $N$ backups of data size $S$ with per-backup change rate $\delta$:

$$\text{Stored} = S + \sum_{i=2}^{N} S \cdot \delta_i \cdot (1 - p_{intra})$$

where $p_{intra}$ is the probability of intra-backup dedup (identical chunks within the changed portion).

Simplified with constant $\delta$:

$$\text{Dedup Ratio} = \frac{N \cdot S}{S \cdot (1 + (N-1) \cdot \delta)} = \frac{N}{1 + (N-1)\delta}$$

| Scenario | N | $\delta$ | Ratio | Actual vs Full |
|:---|:---:|:---:|:---:|:---:|
| Weekly, low churn | 4 | 2% | 3.8x | 26% |
| Daily, typical server | 30 | 3% | 14.5x | 6.9% |
| Daily, database dumps | 30 | 10% | 7.2x | 13.8% |
| Hourly, config files | 168 | 0.1% | 143x | 0.7% |

### Cross-File Dedup

When multiple files share content (VMs, containers, forks):

$$\text{Cross-Dedup Savings} = \sum_{i=1}^{F} S_i - |\text{Unique Chunks}| \cdot \bar{C}$$

---

## 3. Compression Analysis

### Algorithm Comparison

| Algorithm | Ratio (text) | Speed Compress | Speed Decompress | Borg Flag |
|:---|:---:|:---:|:---:|:---|
| none | 1.0x | N/A | N/A | `none` |
| lz4 | 2.0-2.5x | 700 MB/s | 3000 MB/s | `lz4` |
| zstd,1 | 2.5-3.0x | 500 MB/s | 1200 MB/s | `zstd,1` |
| zstd,3 | 2.8-3.5x | 350 MB/s | 1200 MB/s | `zstd,3` |
| zlib,6 | 3.0-4.0x | 30 MB/s | 300 MB/s | `zlib,6` |
| lzma,6 | 3.5-5.0x | 5 MB/s | 50 MB/s | `lzma,6` |

### Shannon Entropy Bound

No lossless compression can beat the entropy bound:

$$H(X) = -\sum_{i=0}^{255} p(x_i) \log_2 p(x_i) \quad \text{bits per byte}$$

$$\text{Minimum Size} = \frac{H(X)}{8} \times \text{Original Size}$$

| Data Type | Entropy (bits/byte) | Max Compression |
|:---|:---:|:---:|
| English text | 1.0-1.5 | 5-8x |
| Source code | 2.0-3.0 | 2.5-4x |
| Log files | 2.5-4.0 | 2-3x |
| Compressed images | 7.5-8.0 | ~1x |
| Encrypted data | 8.0 | 1x (incompressible) |
| Random bytes | 8.0 | 1x (incompressible) |

### Compression + Dedup Interaction

Total storage with both:

$$\text{Stored} = \sum_{c \in \text{unique chunks}} |c| \cdot R_c$$

where $R_c$ is the compression ratio for chunk $c$. Borg compresses after chunking, so dedup operates on raw data while compression reduces the stored size of unique chunks.

---

## 4. Encryption (AES-256-CTR + HMAC-SHA256)

### Repokey Mode

$$K_{master} = \text{PBKDF2-SHA256}(passphrase, salt, iterations=100000, dkLen=32)$$

$$K_{enc} = K_{master}[0:32], \quad K_{mac} = \text{HMAC-SHA256}(K_{master}, \text{"mac key"})$$

### Per-Chunk Encryption

$$\text{Nonce} = \text{Counter (monotonic, 64-bit)}$$

$$C = \text{AES-256-CTR}(K_{enc}, \text{Nonce}, P)$$

$$\text{MAC} = \text{HMAC-SHA256}(K_{mac}, \text{Nonce} \| C)$$

$$\text{Overhead per chunk} = 8\text{ (nonce)} + 32\text{ (MAC)} = 40 \text{ bytes}$$

### PBKDF2 Brute Force

$$T_{crack} = \frac{|\text{keyspace}|}{2 \times \text{rate}} = \frac{|\text{keyspace}|}{2 \times H/s \times iterations^{-1}}$$

At 100,000 PBKDF2 iterations and 10,000 guesses/sec on a GPU:

| Password Entropy | Keyspace | Time to Crack |
|:---:|:---:|:---:|
| 30 bits | $10^9$ | 14 hours |
| 40 bits | $10^{12}$ | 1.6 years |
| 60 bits | $10^{18}$ | 1.6M years |
| 80 bits | $10^{24}$ | $10^{12}$ years |

---

## 5. Repository Segment Structure

### Segment File Layout

Borg stores data in append-only segment files (default max: 500 MiB):

$$\text{Segments} = \left\lceil \frac{\text{Total Stored Data}}{\text{Segment Max Size}} \right\rceil$$

### Index Overhead

The chunk index maps ChunkID to segment/offset:

$$\text{Index Entry} = 32\text{ (HMAC)} + 4\text{ (segment)} + 4\text{ (offset)} + 4\text{ (size)} = 44 \text{ bytes}$$

| Unique Data | Chunks (2 MiB avg) | Index Size |
|:---:|:---:|:---:|
| 100 GiB | 51,200 | 2.1 MiB |
| 1 TiB | 524,288 | 22 MiB |
| 10 TiB | 5,242,880 | 220 MiB |

### Compaction Efficiency

After pruning, freed space in segments creates fragmentation:

$$\text{Fragmentation} = 1 - \frac{\text{Live Data}}{\text{Total Segment Size}}$$

Compaction is triggered when fragmentation exceeds threshold (default 10%):

$$\text{Compaction I/O} = \text{Fragmented Segments} \times \text{Segment Size}$$

---

## 6. Performance Model

### Backup Throughput

$$T_{backup} = \max\left(\frac{S}{BW_{disk\_read}}, \frac{S}{BW_{hash}}, \frac{S_{new}}{BW_{compress}}, \frac{S_{new} \cdot R}{BW_{write}}\right)$$

where $S$ = source size, $S_{new}$ = new (non-dedup) data, $R$ = compression ratio, $BW$ = bandwidth of each stage.

### Pipeline Stages

| Stage | Typical Speed | Bottleneck When |
|:---|:---:|:---|
| Read source | 200-500 MB/s | SSD/HDD limited |
| Buzhash chunking | 1+ GB/s | Rarely |
| HMAC-SHA256 | 500 MB/s | Large new data |
| Dedup lookup | O(1) per chunk | Memory for index |
| Compression (zstd,3) | 350 MB/s | CPU limited |
| Encryption | 2+ GB/s (AES-NI) | Rarely |
| Write to repo | 100-500 MB/s | Network/disk |

---

## Prerequisites

- Rolling hash functions (Buzhash, Rabin)
- HMAC and authenticated encryption
- Information theory (Shannon entropy)
- PBKDF2 key derivation
- Geometric probability distributions

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Create archive | $O(n)$ read + hash | $O(c)$ chunk index in memory |
| Chunk boundary detection | $O(1)$ per byte (rolling) | $O(w)$ window buffer |
| Dedup lookup | $O(1)$ hash table | $O(c)$ unique chunks |
| Extract archive | $O(n)$ | $O(1)$ streaming |
| Prune + Compact | $O(s)$ segments scanned | $O(c)$ index |
| Check --verify-data | $O(n)$ full read | $O(1)$ per chunk |
