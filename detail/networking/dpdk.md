# DPDK Internals -- Kernel Bypass Networking from First Principles

> *The fastest system call is the one you never make.*

---

## 1. EAL Initialization Sequence

### The Problem

Before any packet can be processed, DPDK must configure hugepages, detect
hardware topology, initialize memory zones, discover PCI devices, and set up
per-core data structures. What happens during `rte_eal_init()`, and why does
the order matter?

### The Sequence

The EAL initialization proceeds through a strict series of phases:

```
1. Parse command-line arguments (-l, -n, --socket-mem, -a, ...)
2. CPU detection and core mapping (lcore ID -> physical core)
3. Hugepage detection and memory reservation
4. Memory zone initialization (rte_memzone)
5. PCI bus scan and device enumeration
6. Service core initialization
7. Per-lcore thread creation (pthread_create per lcore)
8. PMD probe and device initialization
9. Timer subsystem setup
10. Interrupt thread launch
```

Each lcore gets its own thread, pinned to a specific CPU via
`pthread_setaffinity_np()`. This pinning is critical: DPDK assumes exclusive
core ownership, and any scheduler interference introduces latency jitter.

### Memory Channel Configuration

The `-n` flag specifies memory channels. Modern CPUs have 2-6 channels per
socket. DPDK interleaves hugepage allocations across channels to maximize
memory bandwidth:

$$BW_{total} = n_{channels} \times BW_{channel}$$

For DDR4-3200 with 4 channels:

$$BW_{total} = 4 \times 25.6 \text{ GB/s} = 102.4 \text{ GB/s}$$

Incorrect channel count causes suboptimal interleaving, reducing effective
bandwidth by up to 75% in the worst case (all allocations on a single channel).

---

## 2. Mbuf Structure and Pool Management

### The Problem

Every packet in DPDK is represented by an `rte_mbuf` structure. How is this
structure organized for cache efficiency, and how does the pool allocator
eliminate malloc overhead?

### Mbuf Layout

```
struct rte_mbuf (128 bytes, cache-line aligned):
+----------------------------------------------------------+
| Cacheline 0 (64 bytes):                                   |
|   buf_addr     - pointer to packet data buffer            |
|   buf_iova     - physical/IOVA address for DMA            |
|   data_off     - offset to start of packet data           |
|   refcnt       - reference count for cloning              |
|   nb_segs      - number of segments (scattered I/O)       |
|   port         - ingress port ID                          |
|   ol_flags     - offload flags (checksum, VLAN, TSO)      |
|   packet_type  - L2/L3/L4 packet type                     |
|   pkt_len      - total packet length across segments      |
|   data_len     - data length in this segment              |
+----------------------------------------------------------+
| Cacheline 1 (64 bytes):                                   |
|   hash         - RSS hash / flow director hash            |
|   vlan_tci     - VLAN tag                                 |
|   tx_offload   - TX offload parameters                    |
|   pool         - pointer to owning mempool                |
|   next         - next segment (chained mbufs)             |
|   timestamp    - packet timestamp                         |
|   dynfield[]   - dynamic fields for extensions            |
+----------------------------------------------------------+
| Headroom (128 bytes default):                             |
|   Space for prepending headers (encapsulation)            |
+----------------------------------------------------------+
| Packet data (up to buf_len - headroom):                   |
|   Actual packet bytes                                     |
+----------------------------------------------------------+
```

The first cacheline contains all fields needed for the fast path
(receive, classify, forward). Fields accessed only on transmit or in
slow paths are placed in the second cacheline.

### Pool Architecture

Mbuf pools (`rte_mempool`) use a two-level allocation scheme:

```
Global Ring (shared, lockless MPMC):
  [mbuf_ptr] [mbuf_ptr] [mbuf_ptr] ... (N entries)

Per-Core Cache (thread-local, no synchronization):
  Core 0 cache: [mbuf_ptr] [mbuf_ptr] ... (C entries)
  Core 1 cache: [mbuf_ptr] [mbuf_ptr] ... (C entries)
  ...
```

Allocation path:
1. Check per-core cache (no atomic ops, fastest)
2. If empty, bulk-refill from global ring (single CAS operation for N mbufs)
3. If ring empty, allocation fails (no fallback to malloc)

Deallocation reverses the process: return to per-core cache, flush to global
ring when cache exceeds threshold.

### Pool Sizing

The pool must be large enough to hold all in-flight packets across all queues,
plus the per-core caches:

$$N_{pool} = N_{ports} \times N_{queues} \times RX_{desc} + N_{cores} \times C_{cache} + margin$$

For 2 ports, 4 queues each, 1024 RX descriptors, 8 cores, 256 cache:

$$N_{pool} = 2 \times 4 \times 1024 + 8 \times 256 + 1024 = 11264$$

Round up to the next power of two minus one (required by the ring): 16383.

---

## 3. Ring Buffer Lock-Free Algorithm

### The Problem

DPDK's `rte_ring` is the fundamental inter-core communication primitive. How
does it achieve lock-free operation for both multi-producer and multi-consumer
scenarios?

### Ring Structure

The ring is a fixed-size circular buffer with head and tail pointers for both
producers and consumers:

```
struct rte_ring:
  prod.head  -- next slot a producer will claim
  prod.tail  -- last slot confirmed written (visible to consumers)
  cons.head  -- next slot a consumer will claim
  cons.tail  -- last slot confirmed read (visible to producers)
  ring[]     -- power-of-2 sized array of void pointers
  mask       -- (size - 1) for fast modulo
```

### Multi-Producer Enqueue (Lock-Free CAS)

```
Thread A                         Thread B
--------                         --------
1. old_head = prod.head           1. old_head = prod.head
2. new_head = old_head + 1        2. new_head = old_head + 1
3. CAS(prod.head, old_head,       3. CAS(prod.head, old_head,
       new_head)                        new_head)
   SUCCESS -> got slot old_head      FAIL -> retry from step 1
4. ring[old_head & mask] = obj       (now sees new old_head)
5. Wait until prod.tail == old_head
6. prod.tail = new_head           ... eventually gets a slot ...
```

The key insight is separating head (claim) from tail (commit). The CAS on
`prod.head` serializes slot claims. The spin-wait on `prod.tail` ensures
in-order visibility to consumers.

### Available Count

$$available = prod.tail - cons.head$$

$$free = size - (prod.head - cons.tail)$$

Both computed without locks using atomic loads.

### Performance Characteristics

Single-producer single-consumer (SPSC) avoids all atomic operations:

| Mode | Enqueue Cost | Dequeue Cost |
|------|-------------|-------------|
| SPSC | ~5 ns       | ~5 ns       |
| MPMC | ~15-30 ns   | ~15-30 ns   |

Burst operations amortize overhead:

$$T_{burst}(n) \approx T_{single} + (n-1) \times T_{copy}$$

For a burst of 32 objects:

$$T_{per\_obj} = \frac{T_{single} + 31 \times 2\text{ ns}}{32} \approx 2.4 \text{ ns}$$

---

## 4. RSS (Receive Side Scaling) Configuration

### The Problem

A single CPU core cannot process line-rate traffic on modern NICs. RSS
distributes incoming packets across multiple RX queues (and thus cores) using
a hash function. How is the hash computed, and how does the indirection table
map flows to queues?

### The Hash Function

RSS uses the Toeplitz hash over selected packet fields:

$$H = \bigoplus_{i=0}^{n-1} \left( K_{i} \cdot input\_bit_i \right)$$

Where K is a 40-byte (or 52-byte) secret key and the input is formed from
selected header fields (typically the 4-tuple: src_ip, dst_ip, src_port,
dst_port for TCP/UDP, or 2-tuple: src_ip, dst_ip for non-TCP/UDP).

The hash produces a 32-bit value. The lower bits index into the indirection
table (RETA), which maps to queue IDs:

$$queue = RETA[H \& (RETA\_size - 1)]$$

### Indirection Table

A typical RETA has 128 or 512 entries:

```
RETA[0]   = queue 0
RETA[1]   = queue 1
RETA[2]   = queue 2
RETA[3]   = queue 3
RETA[4]   = queue 0    (wraps)
...
RETA[127] = queue 3
```

Uniform distribution:

$$P(queue_j) = \frac{|\{i : RETA[i] = j\}|}{|RETA|}$$

For 4 queues and 128 entries, each queue receives approximately 25% of flows.

### Symmetric RSS

Standard Toeplitz can hash src/dst to different queues for the two directions
of a flow. Symmetric RSS uses a key that satisfies:

$$H(src, dst) = H(dst, src)$$

This ensures both directions of a TCP connection land on the same queue,
which is critical for stateful processing.

---

## 5. Flow Director and rte_flow API

### The Problem

RSS distributes flows based on a hash, but cannot steer specific flows to
specific queues, apply per-flow actions (mark, count, drop), or offload
classification to hardware. The rte_flow API provides fine-grained flow
control.

### rte_flow Architecture

```
Flow Rule = Attributes + Pattern + Actions

Attributes:
  - ingress/egress/transfer
  - priority level
  - group ID

Pattern (match criteria):
  ETH -> IPv4 -> TCP -> END
  Each item has spec (values) and mask (which bits matter)

Actions (what to do):
  QUEUE(n)         -- steer to specific queue
  RSS(queues[])    -- distribute across queue subset
  DROP             -- discard the packet
  MARK(id)         -- tag packet with ID (available in mbuf)
  COUNT            -- increment hardware counter
  JUMP(group)      -- jump to another group
  MODIFY_FIELD     -- rewrite header fields
  ENCAP/DECAP      -- tunnel operations
```

### Pattern Matching Precedence

Hardware evaluates rules by priority (lower number = higher priority). When
multiple rules match, the highest priority rule wins:

$$action = rule_i \text{ where } i = \arg\min_{j} \{priority_j : packet \in pattern_j\}$$

### Offload Levels

```
Level              | Where         | Performance
--------------------|---------------|------------------
Full HW offload    | NIC hardware  | Wire speed
Partial offload    | NIC + PMD     | Near wire speed
Software fallback  | PMD in CPU    | CPU-limited
```

Query `rte_flow_validate()` before `rte_flow_create()` to check whether the
NIC can offload a given rule.

---

## 6. Memory Management (IOVA Modes)

### The Problem

NICs perform DMA directly to/from packet buffers. The NIC needs physical
addresses, but userspace applications work with virtual addresses. How does
DPDK resolve this, and what are the tradeoffs between IOVA modes?

### IOVA-as-PA (Physical Address)

```
Application VA  -->  DPDK memzone  -->  Physical Address (PA)
NIC DMA uses PA directly

Pros: Works with all hardware, including those without IOMMU
Cons: Requires /dev/mem or pagemap access (root), security concern
```

Physical addresses are obtained via `/proc/self/pagemap`:

$$PA = PFN \times PAGE\_SIZE + offset$$

### IOVA-as-VA (Virtual Address)

```
Application VA  -->  IOMMU  -->  Physical Address
NIC DMA uses VA, IOMMU translates

Pros: No root for PA lookup, better security (IOMMU isolation)
Cons: Requires IOMMU (Intel VT-d / AMD-Vi), VFIO driver
```

VFIO configures the IOMMU mapping so that IOVA equals the userspace VA.
The NIC sees the same addresses as the application, simplifying buffer
management.

### Memory Segments

DPDK tracks memory in segments backed by hugepages:

```
Heap
  Segment 0: 1GB hugepage @ VA 0x100000000, IOVA 0x100000000
  Segment 1: 1GB hugepage @ VA 0x140000000, IOVA 0x140000000
  ...

Each segment:
  - Contiguous in both VA and IOVA
  - Registered with IOMMU (if VFIO)
  - Contains memzones, mempools, rings
```

---

## 7. Multi-Process Support

### The Problem

Some architectures require multiple processes to share DPDK resources (e.g.,
a control plane process and a data plane process, or multiple independent
workers). How does DPDK support multi-process memory sharing?

### Architecture

```
Primary Process:
  - Initializes EAL, creates all shared memory structures
  - Owns hugepage mappings and PCI device configuration
  - Creates mempools, rings, hash tables in shared memory

Secondary Process(es):
  - Attaches to existing shared memory (same hugepage mappings)
  - Must use same --file-prefix as primary
  - Can access all shared rte_mempool, rte_ring, rte_hash objects
  - Cannot create new shared memory or configure devices
```

### Launching

```bash
# Primary
./my_app -l 0-3 -n 4 --proc-type=primary

# Secondary (must match --file-prefix and memory config)
./my_app -l 4-7 -n 4 --proc-type=secondary
```

### Shared Object Lookup

Secondary processes find shared objects by name:

```c
/* Primary creates */
struct rte_ring *ring = rte_ring_create("shared_ring", 1024,
                                         rte_socket_id(), 0);

/* Secondary looks up */
struct rte_ring *ring = rte_ring_lookup("shared_ring");
```

All named DPDK objects (mempools, rings, hash tables, memzones) are stored in
a shared configuration structure mapped at the same virtual address in all
processes.

---

## 8. Performance Tuning

### Cache Alignment

All performance-critical structures in DPDK are aligned to cache line
boundaries (64 bytes on x86):

```c
struct my_data {
    uint64_t counter;
    uint64_t timestamp;
    /* ... */
} __rte_cache_aligned;
```

False sharing occurs when two cores write to different fields in the same
cache line. The cache coherency protocol (MESI/MOESI) forces the line to
bounce between cores:

$$T_{false\_sharing} = N_{cores} \times T_{cache\_invalidation} \approx N \times 40\text{--}100 \text{ ns}$$

Using `__rte_cache_aligned` on per-core structures eliminates this.

### Prefetching

DPDK uses explicit prefetch instructions to hide memory latency:

```c
for (int i = 0; i < nb_rx; i++) {
    /* Prefetch next packet while processing current */
    if (i + 1 < nb_rx)
        rte_prefetch0(rte_pktmbuf_mtod(pkts[i + 1], void *));

    process_packet(pkts[i]);
}
```

L1 cache miss penalty on modern x86:

$$T_{L1\_miss} \approx 4\text{--}5 \text{ ns (L2 hit)}$$
$$T_{L2\_miss} \approx 12\text{--}15 \text{ ns (L3 hit)}$$
$$T_{L3\_miss} \approx 40\text{--}80 \text{ ns (DRAM)}$$

Prefetching converts L3 misses into L1 hits when the prefetch distance
matches the processing time per packet.

### Batch Processing

Processing packets in bursts amortizes per-batch overhead:

$$T_{total}(n) = T_{setup} + n \times T_{per\_pkt}$$

$$T_{avg}(n) = \frac{T_{setup}}{n} + T_{per\_pkt}$$

As $n$ increases, the setup cost becomes negligible. Typical burst sizes:

| Burst Size | Overhead per Packet | Notes                    |
|-----------|--------------------|-----------------------------|
| 1         | T_setup            | Maximum overhead              |
| 8         | T_setup / 8        | Minimum practical burst       |
| 32        | T_setup / 32       | Default DPDK burst size       |
| 64        | T_setup / 64       | Diminishing returns beyond    |

The optimal burst size balances latency (smaller = lower) against throughput
(larger = higher). For most applications, 32 provides a good tradeoff.

### SIMD Optimization

DPDK uses SIMD intrinsics for bulk operations (4/8/16 packets at once):

```c
/* Example: bulk free using SSE/AVX */
/* rte_mempool_put_bulk() uses vector stores internally */
/* Packet classification can use _mm256_cmpeq_epi32 for parallel matching */
```

Throughput gain from SIMD:

$$speedup = \frac{W_{SIMD}}{W_{scalar}} \leq \frac{register\_width}{element\_width}$$

For AVX2 (256-bit) processing 32-bit fields: up to 8x theoretical speedup.

---

## 9. DPDK vs Kernel Stack -- Latency and Throughput Benchmarks

### Throughput Comparison

For 64-byte packets on 10 GbE (14.88 Mpps line rate):

| Stack          | Mpps  | Line Rate | CPU Cycles/Pkt |
|----------------|-------|-----------|----------------|
| Kernel (raw)   | 1.0   | 6.7%      | ~1000-1500     |
| Kernel (NAPI)  | 2.5   | 16.8%     | ~600           |
| iptables       | 3.0   | 20.2%     | ~500           |
| XDP native     | 24.0  | 161%*     | ~50-100        |
| DPDK           | 14.8  | 100%      | ~80-200        |
| DPDK (2x10G)   | 29.6  | 100%      | ~80-200        |

(*XDP_DROP exceeds 10G line rate because it operates on a fast path before
the NIC fills the ring; the Mpps number reflects raw processing capacity
measured with pktgen.)

### Latency Comparison

Median and tail latency for UDP echo (64-byte payload):

| Stack        | p50 Latency | p99 Latency | p99.9 Latency |
|-------------|-------------|-------------|---------------|
| Kernel       | 20 us       | 150 us      | 500 us        |
| Kernel (busy)| 12 us       | 50 us       | 200 us        |
| XDP          | 5 us        | 15 us       | 40 us         |
| DPDK         | 3 us        | 8 us        | 15 us         |

### Why DPDK Wins on Latency

The kernel networking stack processes a packet through multiple layers, each
adding latency and jitter:

$$L_{kernel} = L_{interrupt} + L_{softirq} + L_{skb\_alloc} + L_{netfilter} + L_{routing} + L_{socket} + L_{copy\_to\_user}$$

DPDK eliminates all of these:

$$L_{dpdk} = L_{poll} + L_{process}$$

Where $L_{poll}$ is the time to check the NIC descriptor ring (a few memory
reads, no system calls, no interrupts) and $L_{process}$ is application logic.

The tail latency improvement is even more significant because DPDK eliminates
sources of jitter: interrupt coalescing timers, scheduler preemption, softirq
deferral, and memory allocation delays.

### Cost Tradeoff

DPDK achieves lower latency and higher throughput at the cost of dedicating
CPU cores to busy-polling:

$$\text{CPU cost} = N_{cores} \times 100\% \text{ utilization (always polling)}$$

This is efficient when traffic rates justify the dedicated cores. For low
traffic volumes, interrupt-driven kernel networking or XDP may be more
resource-efficient.

---

## References

- [DPDK Programmer's Guide](https://doc.dpdk.org/guides/prog_guide/)
- [DPDK API Reference](https://doc.dpdk.org/api/)
- [Intel DPDK Performance Report](https://fast.dpdk.org/doc/perf/)
- [Understanding the Linux Networking Stack](https://blog.packagecloud.io/monitoring-tuning-linux-networking-stack-receiving-data/)
- [Comparing DPDK, XDP, and AF_XDP](https://www.redhat.com/en/blog/using-dpdk-xdp-and-af-xdp)
- [Lock-Free Ring Buffer Design (DPDK)](https://doc.dpdk.org/guides/prog_guide/ring_lib.html)
- [IOVA Modes in DPDK](https://doc.dpdk.org/guides/prog_guide/env_abstraction_layer.html)
