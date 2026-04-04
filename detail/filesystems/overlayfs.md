# The Mathematics of OverlayFS -- Layer Composition and Copy-on-Write Cost Models

> *A filesystem that lies beautifully: showing you one tree while hiding a forest of layers beneath.*

---

## 1. Layer Lookup Complexity (Path Resolution)

### The Problem

When a file is accessed through an overlay mount with $N$ lower layers, the
kernel must search layers top-down until the file is found. What is the expected
lookup time as a function of layer count and file distribution?

### The Formula

For a file access, the kernel checks layers in order: upper, lower_1, lower_2, ..., lower_N.
If the probability of a file existing in layer $i$ is $p_i$ (and layers are
checked sequentially), the expected number of layers checked:

$$E[L] = \sum_{i=1}^{N+1} i \cdot p_i \cdot \prod_{j=1}^{i-1}(1 - p_j)$$

For uniform distribution across layers ($p_i = 1/(N+1)$ for each):

$$E[L] = \frac{N+2}{2}$$

For a Docker-typical distribution where 80% of accesses hit the upper layer:

$$E[L] = 1 \cdot 0.8 + \sum_{i=2}^{N+1} i \cdot \frac{0.2}{N} \cdot \prod_{j=1}^{i-1}(1 - p_j)$$

$$E[L] \approx 0.8 + 0.2 \cdot \frac{N+2}{2} = 0.8 + 0.1(N+2)$$

### Worked Examples

**Example 1:** A Docker image with 12 layers (N=12). Uniform access pattern:

$$E[L] = \frac{12 + 2}{2} = 7 \text{ layers}$$

Each layer check costs approximately 1-2 microseconds (dentry lookup):

$$T_{lookup} = 7 \times 1.5 = 10.5 \text{ } \mu s$$

**Example 2:** Same image, but 80% of accesses are to recently modified files
(in upper layer):

$$E[L] \approx 0.8 + 0.1(12 + 2) = 0.8 + 1.4 = 2.2 \text{ layers}$$

$$T_{lookup} = 2.2 \times 1.5 = 3.3 \text{ } \mu s$$

This is why container workloads typically access upper-layer files and
performance remains acceptable even with many layers.

## 2. Copy-Up Cost (Write Amplification)

### The Problem

When a file in a lower layer is modified, OverlayFS performs a "copy-up":
the entire file is copied to the upper layer before the modification is applied.
What is the write amplification factor for different workload patterns?

### The Formula

Write amplification for a single copy-up of a file of size $S$ where only
$\delta$ bytes are modified:

$$W_{amp} = \frac{S + \delta}{\delta}$$

For a workload with $n$ files, each of size $S_i$, where $f$ fraction require
copy-up and $\delta_i$ bytes are modified per file:

$$W_{total} = \frac{\sum_{i=1}^{n \cdot f} S_i + \sum_{i=1}^{n} \delta_i}{\sum_{i=1}^{n} \delta_i}$$

With metacopy enabled, only metadata is copied initially:

$$W_{metacopy} = \frac{n \cdot f \cdot M + \sum_{i=1}^{n} \delta_i}{\sum_{i=1}^{n} \delta_i}$$

where $M$ is metadata size (typically 4 KB for the inode + xattrs).

### Worked Examples

**Example 1:** Appending 100 bytes to a 500 MB log file in the lower layer:

$$W_{amp} = \frac{500 \times 10^6 + 100}{100} = 5{,}000{,}001$$

This is catastrophic write amplification. OverlayFS copies 500 MB to write 100 bytes.

**Example 2:** With metacopy enabled for the same operation:

$$W_{metacopy} = \frac{4{,}096 + 100}{100} = 41.96$$

Still significant, but 120,000x better than without metacopy. The actual data
is copied lazily only when read through the upper layer.

## 3. Storage Deduplication (Layer Sharing)

### The Problem

Multiple containers sharing the same base image share lower layers. What is the
storage savings from layer sharing versus independent copies?

### The Formula

Storage with independent copies for $C$ containers, each with $N$ layers
of average size $S_L$ and unique upper layer of size $S_U$:

$$V_{independent} = C \times (N \times S_L + S_U)$$

Storage with OverlayFS layer sharing (shared lower layers):

$$V_{overlay} = N \times S_L + C \times S_U$$

Space savings:

$$\text{Savings} = 1 - \frac{V_{overlay}}{V_{independent}} = 1 - \frac{N \cdot S_L + C \cdot S_U}{C \cdot (N \cdot S_L + S_U)}$$

As $C \to \infty$:

$$\text{Savings} \to 1 - \frac{S_U}{N \cdot S_L + S_U}$$

### Worked Examples

**Example 1:** 50 containers sharing a 500 MB base image (5 layers x 100 MB).
Each container's upper layer is 20 MB:

$$V_{independent} = 50 \times (500 + 20) = 26{,}000 \text{ MB}$$

$$V_{overlay} = 500 + 50 \times 20 = 1{,}500 \text{ MB}$$

$$\text{Savings} = 1 - \frac{1{,}500}{26{,}000} = 94.2\%$$

**Example 2:** Only 3 containers with the same base:

$$V_{overlay} = 500 + 3 \times 20 = 560 \text{ MB}$$

$$\text{Savings} = 1 - \frac{560}{3 \times 520} = 64.1\%$$

Layer sharing becomes more valuable as container count increases.

## 4. Whiteout Density (Deletion Overhead)

### The Problem

Deleted files in lower layers are represented by whiteout entries in the upper
layer. As deletions accumulate, whiteout files consume inodes and directory
entries. What is the overhead?

### The Formula

Each whiteout consumes:
- One inode in the upper filesystem ($I_{size}$ bytes, typically 256 bytes on ext4)
- One directory entry ($D_{size}$ bytes, typically 8 + filename length bytes)

Total whiteout overhead for $W$ deletions:

$$V_{whiteout} = W \times (I_{size} + D_{size})$$

The whiteout ratio in a directory with $F$ original files and $W$ deleted:

$$R_{whiteout} = \frac{W}{F}$$

Directory readdir performance degrades as:

$$T_{readdir} = (F - W + W_{upper}) \times T_{entry}$$

where $W_{upper}$ includes whiteout entries that must be filtered.

### Worked Examples

**Example 1:** A container deletes 1,000 files from the base image. Average
filename length is 20 bytes:

$$V_{whiteout} = 1{,}000 \times (256 + 28) = 284{,}000 \text{ bytes} = 277 \text{ KB}$$

$$\text{Inodes consumed} = 1{,}000$$

On a filesystem with 1M inodes, this is 0.1% of inodes.

**Example 2:** An opaque directory replaces a lower directory containing 10,000
files. Without opaque xattr, we would need 10,000 whiteouts. With opaque:

$$V_{opaque} = 1 \times 30 \text{ bytes (xattr)} \ll 10{,}000 \times 284 = 2.84 \text{ MB}$$

Opaque directories reduce whiteout overhead by a factor of:

$$\frac{10{,}000 \times 284}{30} = 94{,}667\times$$

## 5. I/O Amplification (Metadata Operations)

### The Problem

OverlayFS metadata operations (stat, chmod, chown) may need to consult multiple
layers. What is the I/O amplification for metadata-heavy workloads?

### The Formula

For a `stat()` call, the kernel checks layers until the file is found. Each
layer check involves:
- dentry cache lookup: $T_d$ (typically ~100 ns if cached)
- Disk I/O if not cached: $T_{disk}$ (typically ~100 us for HDD, ~50 us for SSD)

Expected I/O operations per stat with cache miss probability $m$:

$$E[IO] = \sum_{i=1}^{N+1} (1 \cdot m + 0 \cdot (1-m)) \cdot P(\text{found at layer } i)$$

For a cold cache (all misses):

$$E[IO_{cold}] = E[L]$$

$$T_{stat} = E[L] \times T_{disk}$$

### Worked Examples

**Example 1:** 10-layer overlay on SSD, cold cache, uniform distribution:

$$E[L] = \frac{10 + 2}{2} = 6$$

$$T_{stat} = 6 \times 50 \text{ } \mu s = 300 \text{ } \mu s$$

Versus single filesystem: $T_{stat} = 50$ $\mu s$. The amplification factor is 6x.

**Example 2:** Same overlay, warm dentry cache (95% hit rate):

$$T_{stat} = 6 \times (0.05 \times 50{,}000 + 0.95 \times 100) \text{ ns}$$

$$T_{stat} = 6 \times (2{,}500 + 95) = 15{,}570 \text{ ns} = 15.6 \text{ } \mu s$$

Dentry cache effectiveness is critical for overlay performance.

## 6. Container Startup Impact (Layer Count vs Boot Time)

### The Problem

Container startup involves opening many files from the image layers. How does
layer count affect startup latency?

### The Formula

Startup opens $F$ files. Total lookup time:

$$T_{start} = F \times E[L] \times T_{layer\_check}$$

With $T_{layer\_check} = 1$ $\mu s$ (cached dentry lookup):

$$T_{start} = F \times \frac{N + 2}{2} \times 10^{-6} \text{ seconds}$$

Docker's 128-layer limit exists because:

$$T_{worst} = F \times \frac{128 + 2}{2} \times T_{check} = 65F \times T_{check}$$

### Worked Examples

**Example 1:** Application opens 500 files at startup, 8-layer image:

$$T_{start} = 500 \times \frac{10}{2} \times 1 = 2{,}500 \text{ } \mu s = 2.5 \text{ ms}$$

**Example 2:** Same application, 50-layer image (poorly optimized):

$$T_{start} = 500 \times \frac{52}{2} \times 1 = 13{,}000 \text{ } \mu s = 13 \text{ ms}$$

The 5.2x slowdown motivates Docker's multi-stage build pattern to minimize
layer count. Real startup is dominated by application init, but overlay lookup
adds measurable overhead for file-heavy boots.

## Prerequisites

- Linux VFS architecture (dentry cache, inode structures, mount namespaces)
- Filesystem internals (ext4/XFS inode layout, directory entry format)
- Copy-on-write semantics and write amplification
- Container image format (OCI layers, content-addressable storage)
- Combinatorics (expected value calculations, probability distributions)
- I/O performance modeling (latency vs throughput, cache hit ratios)
- Extended attributes (xattr) mechanics for whiteouts and opaque markers
