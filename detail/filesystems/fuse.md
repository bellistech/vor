# The Mathematics of FUSE -- Latency Decomposition and Throughput Ceilings

> *The price of userspace freedom is paid in microseconds at every system call boundary.*

---

## 1. Request Latency Decomposition (The FUSE Round Trip)

### The Problem

Every FUSE operation crosses the kernel-userspace boundary twice. What is the
minimum latency for a single filesystem operation, and how do the components
stack up?

### The Formula

Total latency for a single FUSE operation:

$$T_{op} = T_{vfs} + T_{ctx\_switch} + T_{copy\_req} + T_{daemon} + T_{copy\_resp} + T_{ctx\_switch} + T_{vfs\_return}$$

Simplified:

$$T_{op} = 2T_{ctx} + 2T_{copy} + T_{vfs} + T_{daemon}$$

Where:
- $T_{ctx}$: context switch latency ($\sim$2-5 $\mu s$)
- $T_{copy}$: data copy between kernel and userspace ($S / BW_{mem}$)
- $T_{vfs}$: VFS overhead ($\sim$1 $\mu s$)
- $T_{daemon}$: userspace processing time (application-dependent)

For a native (in-kernel) filesystem:

$$T_{native} = T_{vfs} + T_{fs}$$

The FUSE overhead ratio:

$$\Omega = \frac{T_{op}}{T_{native}} = \frac{2T_{ctx} + 2T_{copy} + T_{vfs} + T_{daemon}}{T_{vfs} + T_{fs}}$$

### Worked Examples

**Example 1:** A `getattr` call (no data copy). $T_{ctx} = 3$ $\mu s$,
$T_{vfs} = 1$ $\mu s$, $T_{daemon} = 1$ $\mu s$:

$$T_{getattr} = 2(3) + 2(0) + 1 + 1 = 8 \text{ } \mu s$$

Native ext4 getattr: $T_{native} \approx 1$ $\mu s$

$$\Omega = \frac{8}{1} = 8\times \text{ overhead}$$

**Example 2:** A 4 KB `read` call. $BW_{mem} = 30$ GB/s:

$$T_{copy} = \frac{4096}{30 \times 10^9} = 0.14 \text{ } \mu s$$

$$T_{read} = 2(3) + 2(0.14) + 1 + 2 = 9.28 \text{ } \mu s$$

With splice (zero-copy), one copy is eliminated:

$$T_{read\_splice} = 2(3) + 0.14 + 1 + 2 = 9.14 \text{ } \mu s$$

For 4 KB reads, splice savings are negligible. For 1 MB reads:

$$T_{copy\_1MB} = \frac{10^6}{30 \times 10^9} = 33.3 \text{ } \mu s$$

$$\text{Splice savings} = 33.3 \text{ } \mu s \text{ per read}$$

## 2. Throughput Ceiling (Maximum IOPS)

### The Problem

Given the per-operation overhead of FUSE, what is the maximum achievable
IOPS for metadata operations and data operations?

### The Formula

Single-threaded maximum IOPS:

$$IOPS_{single} = \frac{1}{T_{op}}$$

Multi-threaded maximum IOPS with $W$ worker threads:

$$IOPS_{multi} = \frac{W}{T_{op}} \cdot \eta_{contention}$$

where $\eta_{contention}$ accounts for lock contention and is modeled by:

$$\eta_{contention} = \frac{1}{1 + \sigma(W - 1)}$$

$\sigma$ is the serialization fraction (typically 0.01-0.05 for well-designed FUSE daemons).

### Worked Examples

**Example 1:** Single-threaded metadata IOPS ($T_{op} = 8$ $\mu s$):

$$IOPS_{single} = \frac{1}{8 \times 10^{-6}} = 125{,}000 \text{ IOPS}$$

Native ext4: $IOPS_{native} = 1{,}000{,}000$ IOPS (cached metadata).

**Example 2:** Multi-threaded with 16 workers, $\sigma = 0.02$:

$$\eta = \frac{1}{1 + 0.02 \times 15} = \frac{1}{1.3} = 0.77$$

$$IOPS_{multi} = \frac{16}{8 \times 10^{-6}} \times 0.77 = 1{,}540{,}000 \text{ IOPS}$$

With sufficient threads, FUSE can exceed single-threaded native performance.

## 3. Bandwidth vs Latency Tradeoff (Buffer Size Optimization)

### The Problem

FUSE read/write operations can use different buffer sizes. What is the optimal
buffer size that maximizes bandwidth while keeping latency acceptable?

### The Formula

Bandwidth as a function of buffer size $B$:

$$BW(B) = \frac{B}{T_{fixed} + T_{copy}(B)} = \frac{B}{T_{fixed} + \frac{B}{BW_{mem}}}$$

Taking the derivative and setting to zero to find optimal $B$:

$$\frac{dBW}{dB} = \frac{T_{fixed}}{\left(T_{fixed} + \frac{B}{BW_{mem}}\right)^2} = 0$$

This has no finite maximum; bandwidth increases monotonically with buffer
size. But latency also increases:

$$L(B) = T_{fixed} + \frac{B}{BW_{mem}}$$

The practical optimum balances bandwidth efficiency $\eta$ against latency:

$$\eta = \frac{BW(B)}{BW_{mem}} = \frac{B}{T_{fixed} \cdot BW_{mem} + B}$$

To achieve $\eta = 0.9$:

$$B_{0.9} = 9 \cdot T_{fixed} \cdot BW_{mem}$$

### Worked Examples

**Example 1:** $T_{fixed} = 8$ $\mu s$, $BW_{mem} = 30$ GB/s. Buffer for
90% bandwidth efficiency:

$$B_{0.9} = 9 \times 8 \times 10^{-6} \times 30 \times 10^9 = 2{,}160{,}000 \text{ bytes} \approx 2 \text{ MB}$$

Latency at this buffer size:

$$L = 8 + \frac{2{,}160{,}000}{30 \times 10^9} \times 10^6 = 8 + 72 = 80 \text{ } \mu s$$

**Example 2:** For 128 KB buffer (FUSE default max_read):

$$\eta = \frac{131{,}072}{8 \times 10^{-6} \times 30 \times 10^9 + 131{,}072} = \frac{131{,}072}{240{,}000 + 131{,}072} = 35.3\%$$

Only 35% bandwidth efficiency with the default buffer size.

## 4. Cache Effectiveness (Kernel Page Cache with FUSE)

### The Problem

FUSE can leverage the kernel page cache (`kernel_cache`, `auto_cache`). What is
the effective IOPS and bandwidth when cache hit ratio is considered?

### The Formula

Effective operation time with cache hit ratio $h$:

$$T_{eff} = h \cdot T_{cache} + (1 - h) \cdot T_{fuse}$$

Where $T_{cache} \approx 0.1$ $\mu s$ (page cache hit) and
$T_{fuse} \approx 8$ $\mu s$ (FUSE round trip).

Effective IOPS:

$$IOPS_{eff} = \frac{1}{T_{eff}} = \frac{1}{h \cdot T_{cache} + (1 - h) \cdot T_{fuse}}$$

Cache hit ratio for a working set $W$ and cache size $C$:

$$h \approx \min\left(1, \frac{C}{W}\right) \quad \text{(simplified, uniform access)}$$

For Zipfian access pattern (80/20 rule):

$$h_{zipf} \approx \left(\frac{C}{W}\right)^{1-\theta}$$

where $\theta \approx 0.7$ for typical file access patterns.

### Worked Examples

**Example 1:** Working set 10 GB, page cache 4 GB, uniform access:

$$h = \frac{4}{10} = 0.4$$

$$T_{eff} = 0.4 \times 0.1 + 0.6 \times 8 = 0.04 + 4.8 = 4.84 \text{ } \mu s$$

$$IOPS_{eff} = \frac{1}{4.84 \times 10^{-6}} = 206{,}600 \text{ IOPS}$$

**Example 2:** Same setup, Zipfian access ($\theta = 0.7$):

$$h_{zipf} = \left(\frac{4}{10}\right)^{0.3} = 0.4^{0.3} = 0.759$$

$$T_{eff} = 0.759 \times 0.1 + 0.241 \times 8 = 0.076 + 1.928 = 2.0 \text{ } \mu s$$

$$IOPS_{eff} = 500{,}000 \text{ IOPS}$$

Zipfian access patterns dramatically improve FUSE performance through caching.

## 5. Network-Backed FUSE (sshfs/s3fs Latency Model)

### The Problem

Network-backed FUSE filesystems add network round-trip latency to every
uncached operation. How does network latency dominate the performance model?

### The Formula

Total operation time for network-backed FUSE:

$$T_{net} = T_{fuse} + T_{rtt} + T_{transfer}$$

$$T_{net} = (2T_{ctx} + T_{vfs} + T_{daemon}) + RTT + \frac{S}{BW_{net}}$$

The network amplification factor:

$$A_{net} = \frac{T_{net}}{T_{fuse}} = 1 + \frac{RTT + \frac{S}{BW_{net}}}{T_{fuse}}$$

For metadata operations ($S = 0$):

$$A_{meta} = 1 + \frac{RTT}{T_{fuse}}$$

### Worked Examples

**Example 1:** sshfs with 10 ms RTT (cross-continent). Metadata operation:

$$T_{net} = 8 + 10{,}000 + 0 = 10{,}008 \text{ } \mu s \approx 10 \text{ ms}$$

$$A_{meta} = 1 + \frac{10{,}000}{8} = 1{,}251\times$$

$$IOPS_{meta} = \frac{1}{10 \times 10^{-3}} = 100 \text{ IOPS}$$

An `ls` of a directory with 1,000 entries (each needing getattr):

$$T_{ls} = 1{,}000 \times 10 \text{ ms} = 10 \text{ seconds}$$

This explains why sshfs feels slow for directory listings without caching.

**Example 2:** s3fs reading a 10 MB file with 50 ms RTT (to S3), 100 Mbps:

$$T_{transfer} = \frac{10 \times 10^6}{100 \times 10^6 / 8} = 800 \text{ ms}$$

$$T_{net} = 0.008 + 50 + 800 = 850 \text{ ms}$$

With `vfs-cache-mode full` (rclone), subsequent reads hit local cache:

$$T_{cached} = T_{fuse} = 8 \text{ } \mu s$$

$$\text{Speedup} = \frac{850{,}000}{8} = 106{,}250\times$$

## 6. Writeback Cache Impact (Write Coalescing)

### The Problem

FUSE writeback cache batches multiple small writes into larger ones. What is
the throughput improvement from write coalescing?

### The Formula

Without writeback cache, $n$ small writes of size $s$ each:

$$T_{no\_wb} = n \times (T_{fuse} + \frac{s}{BW_{mem}})$$

With writeback cache, writes are coalesced into $\lceil ns / B \rceil$ large
writes of buffer size $B$:

$$T_{wb} = \left\lceil \frac{ns}{B} \right\rceil \times (T_{fuse} + \frac{B}{BW_{mem}})$$

Speedup:

$$S_{wb} = \frac{T_{no\_wb}}{T_{wb}} \approx \frac{n \cdot T_{fuse}}{\frac{ns}{B} \cdot T_{fuse}} = \frac{B}{s}$$

For small writes, the speedup approaches the buffer-to-write ratio.

### Worked Examples

**Example 1:** 10,000 writes of 100 bytes each, $B = 128$ KB:

$$T_{no\_wb} = 10{,}000 \times 8 = 80{,}000 \text{ } \mu s = 80 \text{ ms}$$

$$\left\lceil \frac{10{,}000 \times 100}{131{,}072} \right\rceil = 8 \text{ coalesced writes}$$

$$T_{wb} = 8 \times (8 + 4.4) = 99.2 \text{ } \mu s \approx 0.1 \text{ ms}$$

$$S_{wb} = \frac{80}{0.1} = 800\times$$

**Example 2:** 100 writes of 64 KB each (already large):

$$\left\lceil \frac{100 \times 65{,}536}{131{,}072} \right\rceil = 50 \text{ writes}$$

$$S_{wb} = \frac{100}{50} = 2\times$$

Writeback cache provides dramatic improvement for small writes but diminishing
returns as write size approaches the buffer size.

## Prerequisites

- Linux VFS layer (superblock, inode, dentry, file operations)
- System call mechanics and context switch costs
- Memory bandwidth and DMA fundamentals
- Page cache architecture and eviction policies (LRU)
- Queueing theory for multi-threaded FUSE daemon modeling
- Network latency concepts (RTT, bandwidth-delay product)
- Zipf's Law and access pattern distributions
