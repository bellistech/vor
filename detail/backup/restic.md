# The Mathematics of Restic — Deduplication and Content-Addressable Storage

> *Restic's design is grounded in content-defined chunking with Rabin fingerprints, cryptographic hashing for deduplication, and polynomial rolling hash functions that achieve expected O(1) chunk boundary detection across arbitrary byte streams.*

---

## 1. Content-Defined Chunking (Polynomial Hashing)

### Rabin Fingerprint Rolling Hash

Restic uses a Rabin fingerprint to find chunk boundaries. The polynomial rolling hash over a window of $w$ bytes:

$$H(B) = \left(\sum_{i=0}^{w-1} b_i \cdot p^{w-1-i}\right) \mod m$$

where $b_i$ are byte values, $p$ is an irreducible polynomial over $GF(2)$, and $m$ defines the modular field.

A chunk boundary occurs when:

$$H(B) \mod D = r$$

where $D$ is the target chunk divisor and $r$ is a fixed remainder.

### Chunk Size Distribution

With target average chunk size $\bar{C}$, divisor $D = \bar{C}$:

$$P(\text{boundary at byte } i) = \frac{1}{D}$$

$$E[\text{chunk size}] = D = \bar{C}$$

The chunk sizes follow a geometric distribution:

$$P(X = k) = \left(1 - \frac{1}{D}\right)^{k-1} \cdot \frac{1}{D}$$

$$\text{Var}(X) = \frac{1 - 1/D}{(1/D)^2} \approx D^2$$

### Chunk Size Bounds

Restic enforces minimum and maximum chunk sizes to prevent extreme fragmentation or oversized chunks:

| Parameter | Default | Purpose |
|:---|:---:|:---|
| Minimum chunk | 512 KiB | Prevent tiny chunks |
| Target average | 1 MiB | Balance dedup vs overhead |
| Maximum chunk | 8 MiB | Cap memory per chunk |

$$E[\text{chunks per file}] = \frac{\text{File Size}}{\bar{C}}$$

| File Size | Avg Chunks (1 MiB target) | Index Entries |
|:---:|:---:|:---:|
| 10 MiB | 10 | 10 |
| 1 GiB | 1,024 | 1,024 |
| 100 GiB | 102,400 | 102,400 |
| 1 TiB | 1,048,576 | 1,048,576 |

---

## 2. Deduplication Efficiency (Content-Addressable Storage)

### Hash-Based Deduplication

Each chunk is identified by its SHA-256 hash:

$$\text{ChunkID} = \text{SHA-256}(\text{chunk data})$$

Collision probability for $n$ chunks:

$$P(\text{collision}) \approx \frac{n^2}{2^{257}}$$

| Chunks Stored | Collision Probability |
|:---:|:---:|
| $10^6$ | $\approx 10^{-65}$ |
| $10^9$ | $\approx 10^{-59}$ |
| $10^{12}$ | $\approx 10^{-53}$ |
| $10^{15}$ | $\approx 10^{-47}$ |

### Dedup Ratio Model

For $N$ total backups with change rate $\delta$ per backup:

$$\text{Unique Data} = S_0 + (N-1) \cdot S_0 \cdot \delta$$

$$\text{Dedup Ratio} = \frac{N \cdot S_0}{S_0 + (N-1) \cdot S_0 \cdot \delta} = \frac{N}{1 + (N-1) \cdot \delta}$$

| Backups (N) | Change Rate ($\delta$) | Dedup Ratio | Storage vs Naive |
|:---:|:---:|:---:|:---:|
| 7 | 1% | 6.6x | 15.2% |
| 30 | 1% | 23.3x | 4.3% |
| 30 | 5% | 10.3x | 9.7% |
| 365 | 1% | 72.3x | 1.4% |

---

## 3. Encryption Model (AES-256 + Poly1305)

### Authenticated Encryption

Restic encrypts every pack file with AES-256-CTR and authenticates with Poly1305:

$$C = \text{AES-256-CTR}(K_{enc}, \text{nonce}, P)$$

$$\text{MAC} = \text{Poly1305}(K_{mac}, C)$$

### Key Derivation

The master key is derived from the user password using scrypt:

$$K_{master} = \text{scrypt}(password, salt, N=2^{15}, r=8, p=1, dkLen=64)$$

| scrypt Parameter | Value | Effect |
|:---|:---:|:---|
| N (CPU/memory cost) | $2^{15}$ | 32768 iterations |
| r (block size) | 8 | Memory per block: 1 KiB |
| p (parallelism) | 1 | Single-threaded derivation |
| Memory required | $128 \cdot N \cdot r$ | 32 MiB |
| dkLen | 64 bytes | 256-bit enc + 256-bit mac |

### Brute Force Resistance

$$T_{brute} = \frac{|\text{Password Space}|}{2} \times T_{scrypt}$$

| Password Type | Space Size | Time at 10 scrypt/sec |
|:---|:---:|:---:|
| 4-digit PIN | $10^4$ | 8 minutes |
| 8-char lowercase | $26^8 \approx 2 \times 10^{11}$ | 317 years |
| 12-char mixed | $72^{12} \approx 10^{22}$ | $10^{13}$ years |
| 20-char passphrase | $\gg 2^{128}$ | Infeasible |

---

## 4. Repository Pack File Structure (Bin Packing)

### Pack File Model

Restic groups chunks into pack files (default target: 4-16 MiB):

$$\text{Packs} = \left\lceil \frac{\text{Total Unique Chunks} \times \bar{C}}{\text{Pack Target Size}} \right\rceil$$

### Storage Overhead

Per-chunk metadata in the index:

$$\text{Index Entry} = 32\text{ (hash)} + 4\text{ (length)} + 4\text{ (pack offset)} + 32\text{ (pack ID)} = 72 \text{ bytes}$$

| Unique Data | Chunks (1 MiB avg) | Index Size |
|:---:|:---:|:---:|
| 100 GiB | 102,400 | 7.0 MiB |
| 1 TiB | 1,048,576 | 72 MiB |
| 10 TiB | 10,485,760 | 720 MiB |

### Prune Repacking Cost

When pruning, restic must repack packs containing both used and unused chunks:

$$\text{Repack Volume} = \text{Mixed Packs} \times \text{Avg Pack Size}$$

$$\text{Mixed Pack Fraction} \approx 1 - (1 - \delta_{forget})^{C_{per\_pack}}$$

where $\delta_{forget}$ is the fraction of chunks made obsolete and $C_{per\_pack}$ is chunks per pack.

---

## 5. Snapshot Tree Structure (Merkle DAG)

### Tree Representation

Restic stores the filesystem as a Merkle DAG where each directory is a tree node:

$$\text{TreeID} = \text{SHA-256}(\text{serialized tree node})$$

Total nodes for a filesystem with $d$ directories and $f$ files:

$$\text{Total Nodes} = d + f$$

$$\text{Tree Depth} = O(\log_{b}(f))$$

where $b$ is the average branching factor (files per directory).

### Diff Efficiency

Comparing two snapshots requires walking from root until tree hashes diverge:

$$\text{Nodes Compared} = O(\text{changed paths} \times \text{depth})$$

For a change affecting $k$ files in a tree of depth $h$:

$$\text{Best Case} = O(k \times h), \quad \text{Worst Case} = O(d + f)$$

---

## 6. Bandwidth and Transfer Optimization

### Incremental Backup Transfer

$$\text{Transfer} = \text{New Chunks} \times \bar{C} + \text{Metadata Overhead}$$

$$\text{New Chunks} \approx \text{Changed Bytes} / \bar{C} + \text{Boundary Shifts}$$

Boundary shift overhead from content-defined chunking with insertions of size $I$:

$$\text{Extra Chunks from Insert} \leq 2 \quad (\text{only chunks touching the edit boundary change})$$

| Operation | New Chunks | Transfer (1 MiB avg) |
|:---|:---:|:---:|
| Append 5 MiB to file | 5 | 5 MiB |
| Insert 1 byte mid-file | 2 | 2 MiB |
| Modify 100 bytes mid-file | 2 | 2 MiB |
| Delete 3 MiB mid-file | 2 | 2 MiB |

---

## Prerequisites

- Cryptographic hash functions (SHA-256, collision resistance)
- Rolling hash / Rabin fingerprints
- Content-addressable storage and Merkle trees
- Authenticated encryption (AES-CTR, Poly1305)
- Geometric probability distributions

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Backup (new data) | $O(n)$ where $n$ = bytes | $O(w)$ rolling hash window |
| Chunk boundary detection | $O(1)$ per byte (rolling) | $O(w)$ window buffer |
| Dedup lookup | $O(1)$ hash table | $O(c)$ where $c$ = unique chunks |
| Snapshot diff | $O(k \cdot h)$ | $O(h)$ tree depth |
| Restore | $O(n)$ | $O(1)$ streaming |
| Forget + Prune | $O(c)$ reindex | $O(c)$ index in memory |
