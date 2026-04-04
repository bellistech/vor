# The Mathematics of Swap -- Page Reclaim Algorithms, Swappiness, and Compression Ratios

> *Swap transforms the binary constraint of finite RAM into a continuous spectrum of*
> *memory availability, trading latency for capacity. The kernel's page reclaim decisions*
> *are governed by LRU aging, watermark thresholds, and proportional scanning ratios.*

---

## 1. Swappiness and the Scan Balance (Proportional Control)

### The Problem

The `vm.swappiness` parameter controls the kernel's relative preference for reclaiming anonymous pages (swap out) versus file-backed pages (page cache eviction). How does this ratio work?

### The Formula

The kernel calculates scan ratios for anonymous and file LRU lists:

$$\text{anon\_prio} = \text{swappiness}$$

$$\text{file\_prio} = 200 - \text{swappiness}$$

Fraction of scanning effort directed at anonymous pages:

$$f_{\text{anon}} = \frac{\text{anon\_prio}}{\text{anon\_prio} + \text{file\_prio}} = \frac{\text{swappiness}}{200}$$

$$f_{\text{file}} = \frac{200 - \text{swappiness}}{200}$$

### Worked Examples

| swappiness | anon_prio | file_prio | f_anon | f_file | Behavior |
|-----------|-----------|-----------|--------|--------|----------|
| 0 | 0 | 200 | 0% | 100% | Never swap (unless OOM) |
| 1 | 1 | 199 | 0.5% | 99.5% | Minimal swapping |
| 10 | 10 | 190 | 5% | 95% | Light swap, prefer cache evict |
| 60 | 60 | 140 | 30% | 70% | Default, balanced |
| 100 | 100 | 100 | 50% | 50% | Equal preference |
| 200 | 200 | 0 | 100% | 0% | Maximum swap aggression |

At `swappiness=60`, the kernel scans approximately 30% anonymous pages and 70% file pages during reclaim. The actual pages reclaimed depend on their age in the LRU.

## 2. Page Reclaim Watermarks (Threshold System)

### The Problem

The kernel uses three memory watermarks to trigger different levels of page reclaim. How are these calculated and what happens at each threshold?

### The Formula

For each memory zone, the kernel calculates:

$$\text{min} = \text{min\_free\_kbytes} \times \frac{\text{zone\_size}}{\text{total\_memory}}$$

$$\text{low} = \text{min} \times \frac{125}{100} = 1.25 \times \text{min}$$

$$\text{high} = \text{min} \times \frac{150}{100} = 1.5 \times \text{min}$$

Reclaim behavior:

$$\text{action}(\text{free}) = \begin{cases}
\text{none} & \text{if free} > \text{high} \\
\text{kswapd wakes} & \text{if free} \leq \text{low} \\
\text{direct reclaim} & \text{if free} \leq \text{min} \\
\text{OOM kill} & \text{if free} = 0 \text{ and reclaim fails}
\end{cases}$$

### Worked Examples

System with 16 GB RAM, `min_free_kbytes = 67584` (66 MB):

| Watermark | Formula | Value | Free Memory |
|-----------|---------|-------|-------------|
| min | 66 MB | 66 MB | Below: direct reclaim (blocks allocations) |
| low | 66 x 1.25 | 82.5 MB | Below: kswapd background reclaim |
| high | 66 x 1.5 | 99 MB | Above: kswapd sleeps |

The gap between low and high (`16.5 MB`) is the kswapd operating range. It wakes at low and sleeps when free memory reaches high.

For a 64 GB server:

$$\text{min\_free\_kbytes} = \sqrt{64 \times 1024 \times 1024 \times 16} \approx 131{,}072 \text{ KB} = 128 \text{ MB}$$

| Watermark | Value |
|-----------|-------|
| min | 128 MB |
| low | 160 MB |
| high | 192 MB |

## 3. LRU Page Aging (List Rotation)

### The Problem

The kernel maintains Least Recently Used (LRU) lists to identify cold pages for reclaim. How does the aging algorithm decide which pages to evict?

### The Formula

Each page has an access bit. The kernel uses a two-list model:

- **Active list**: recently accessed pages
- **Inactive list**: candidates for reclaim

Page promotion (inactive to active):

$$\text{promote}(p) \iff \text{accessed}(p) = 1 \text{ while on inactive list}$$

Page demotion (active to inactive):

$$\text{demote}(p) \iff \text{accessed}(p) = 0 \text{ after scanning}$$

Scan pressure determines how many pages are examined:

$$\text{pages\_scanned} = \frac{\text{pages\_to\_reclaim}}{\text{scan\_ratio}}$$

### Worked Examples

Memory reclaim pass on a system with:

| List | Pages | Description |
|------|-------|-------------|
| Active anonymous | 500,000 | Heap/stack, recently accessed |
| Inactive anonymous | 200,000 | Heap/stack, not recently accessed |
| Active file | 300,000 | Page cache, recently accessed |
| Inactive file | 400,000 | Page cache, not recently accessed |

With `swappiness=60`, need to reclaim 10,000 pages:

$$\text{anon\_scan} = 10{,}000 \times \frac{60}{200} = 3{,}000 \text{ pages from inactive anon}$$

$$\text{file\_scan} = 10{,}000 \times \frac{140}{200} = 7{,}000 \text{ pages from inactive file}$$

Of the 3,000 anonymous pages scanned, those without the accessed bit set are swapped out. Of the 7,000 file pages scanned, clean pages are dropped, dirty pages are written back then dropped.

## 4. Zswap Compression Ratios (Information Theory)

### The Problem

Zswap compresses pages before writing to the swap device. The compression ratio determines how much physical swap is saved.

### The Formula

$$\text{compression\_ratio} = \frac{\text{original\_size}}{\text{compressed\_size}}$$

$$\text{effective\_swap} = \text{physical\_swap} \times \text{avg\_compression\_ratio}$$

$$\text{memory\_saved} = \text{stored\_pages} \times \text{page\_size} \times \left(1 - \frac{1}{\text{ratio}}\right)$$

### Worked Examples

Compression ratios by data type (4 KB pages):

| Data Type | Compressed Size | Ratio | Savings |
|----------|----------------|-------|---------|
| Zero-filled page | 16 bytes | 256:1 | 99.6% |
| Sparse data (mostly zeros) | 200 bytes | 20:1 | 95.1% |
| Text/strings | 1,200 bytes | 3.4:1 | 70.6% |
| Structured data (structs) | 1,800 bytes | 2.3:1 | 56.1% |
| Already compressed (JPEG) | 4,000 bytes | 1.0:1 | 0.0% |
| Random data | 4,100 bytes | 0.98:1 | -2.4% |

Zswap pool efficiency for a workload with mixed page types:

| Page Type | Count | Original | Compressed | Ratio |
|----------|-------|----------|------------|-------|
| Zero pages | 10,000 | 40 MB | 0.16 MB | 256:1 |
| Application heap | 50,000 | 200 MB | 90 MB | 2.2:1 |
| Text data | 20,000 | 80 MB | 24 MB | 3.3:1 |
| Random/encrypted | 5,000 | 20 MB | 20 MB | 1.0:1 |
| **Total** | **85,000** | **340 MB** | **134.16 MB** | **2.53:1** |

With a 4 GB swap partition and 2.53:1 average ratio, effective swap capacity:

$$\text{effective} = 4 \text{ GB} \times 2.53 = 10.12 \text{ GB}$$

## 5. Zram Efficiency (Memory vs Disk Tradeoff)

### The Problem

Zram uses RAM as a compressed swap device. When does the compression benefit outweigh the RAM cost?

### The Formula

Zram is beneficial when:

$$\text{compressed\_size} < \text{original\_size} - \text{metadata\_overhead}$$

Net memory gain:

$$\text{gain} = \text{original\_size} - \text{compressed\_size} - \text{overhead}$$

Where overhead is approximately 10% of compressed size for allocator metadata.

Breakeven compression ratio:

$$r_{\text{breakeven}} = \frac{1}{1 - \text{overhead\_fraction}} = \frac{1}{0.9} \approx 1.11$$

Pages with ratio below 1.11 should go to disk-backed swap, not zram.

### Worked Examples

System with 16 GB RAM, 8 GB zram configured:

| Scenario | Stored | Compressed | RAM Used | Net Gain |
|----------|--------|-----------|----------|----------|
| Best case (3:1) | 8 GB | 2.67 GB | 2.93 GB | 5.07 GB |
| Typical (2.5:1) | 8 GB | 3.20 GB | 3.52 GB | 4.48 GB |
| Poor (1.5:1) | 8 GB | 5.33 GB | 5.87 GB | 2.13 GB |
| Breakeven (1.1:1) | 8 GB | 7.27 GB | 8.00 GB | 0.00 GB |
| Worse than disk | 8 GB | 8.00 GB | 8.80 GB | -0.80 GB |

## 6. Swap Latency Impact (Performance Analysis)

### The Problem

When a page is swapped out and later accessed, the page fault incurs disk I/O latency. What is the performance impact?

### The Formula

Page fault latency:

$$t_{\text{swap\_fault}} = t_{\text{page\_fault}} + t_{\text{disk\_read}} + t_{\text{decompress}}$$

Application slowdown factor:

$$\text{slowdown} = 1 + \text{swap\_fault\_rate} \times \frac{t_{\text{swap\_fault}}}{t_{\text{instruction}}}$$

### Worked Examples

| Storage Type | Read Latency | Decompress (zswap) | Total Fault Time |
|-------------|-------------|-------------------|-----------------|
| NVMe SSD | 100 us | 5 us | 105 us |
| SATA SSD | 500 us | 5 us | 505 us |
| HDD (7200 RPM) | 8,000 us | 5 us | 8,005 us |
| Zram (in RAM) | 0 us | 5 us | 5 us |

Application making 1,000 memory accesses per second, with various swap fault rates:

| Fault Rate | NVMe Slowdown | HDD Slowdown | Zram Slowdown |
|-----------|--------------|-------------|--------------|
| 0.1% | 1.001x | 1.008x | 1.000x |
| 1% | 1.011x | 1.080x | 1.001x |
| 5% | 1.053x | 1.400x | 1.003x |
| 10% | 1.105x | 1.801x | 1.005x |
| 50% | 1.525x | 5.003x | 1.025x |

## 7. Swap Space Sizing (Capacity Planning)

### The Problem

How much swap space should be provisioned for a given workload?

### The Formula

Traditional sizing:

$$\text{swap} = \begin{cases}
2 \times \text{RAM} & \text{if RAM} \leq 2 \text{ GB} \\
\text{RAM} + 2 \text{ GB} & \text{if } 2 < \text{RAM} \leq 8 \text{ GB} \\
\text{RAM} \times 0.5 & \text{if } 8 < \text{RAM} \leq 64 \text{ GB} \\
4 \text{ GB (fixed)} & \text{if RAM} > 64 \text{ GB}
\end{cases}$$

Workload-aware sizing:

$$\text{swap} = \max(\text{peak\_memory} - \text{RAM}, 0) + \text{hibernate\_reserve}$$

### Worked Examples

| RAM | Traditional | Server (no hibernate) | Desktop (with hibernate) |
|-----|------------|----------------------|------------------------|
| 1 GB | 2 GB | 1 GB | 2 GB |
| 4 GB | 6 GB | 2 GB | 5 GB |
| 8 GB | 6 GB | 4 GB | 10 GB |
| 16 GB | 8 GB | 4 GB | 18 GB |
| 32 GB | 16 GB | 4 GB | 34 GB |
| 64 GB | 32 GB | 4 GB | 66 GB |
| 128 GB | 4 GB | 4 GB | Not practical |

For Kubernetes nodes: swap is typically disabled or set to 0.

## Prerequisites

virtual-memory, page-tables, lru-algorithms, compression-theory, storage-performance

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Page swap out (write to disk) | O(1) + disk I/O | O(1) per page |
| Page swap in (read from disk) | O(1) + disk I/O | O(1) per page |
| Zswap compress (per page) | O(page_size) | O(compressed_size) |
| Zswap decompress (per page) | O(compressed_size) | O(page_size) |
| LRU list scan | O(pages_scanned) | O(1) |
| Watermark check | O(1) per zone | O(1) |
| Swappiness ratio calculation | O(1) | O(1) |
| mkswap format | O(swap_size / block_size) | O(1) |
