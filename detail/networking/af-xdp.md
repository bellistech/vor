# AF_XDP Internals -- Zero-Copy Packet Processing at Scale

> *The fastest packet is the one the kernel never touches.*

---

## 1. UMEM Layout and Frame Management

### The Architecture

UMEM (User Memory) is a contiguous memory region registered with the kernel via
`setsockopt(XDP_UMEM_REG)`. The kernel and userspace share this region directly,
eliminating copy overhead. The region is divided into fixed-size frames, each
capable of holding one packet.

```
UMEM Memory Region (e.g., 16 MB)
+----------+----------+----------+----------+-----+----------+
| Frame 0  | Frame 1  | Frame 2  | Frame 3  | ... | Frame N  |
| 4096 B   | 4096 B   | 4096 B   | 4096 B   |     | 4096 B   |
+----------+----------+----------+----------+-----+----------+
|<-- headroom -->|<-- packet data -->|<-- unused -->|

Frame address = frame_index * frame_size
Packet data   = frame_address + frame_headroom
```

### Frame Allocation Strategy

Userspace must manage a free-frame pool. The kernel does not track which frames
are available -- it only consumes addresses from the FILL ring and produces
addresses on the COMPLETION ring. A simple stack-based allocator works well:

```c
struct frame_allocator {
    uint64_t *stack;       // stack of free frame addresses
    uint32_t  top;         // index of next free slot
    uint32_t  capacity;    // total number of frames
};

static inline uint64_t frame_alloc(struct frame_allocator *fa) {
    if (fa->top == 0)
        return UINT64_MAX;  // out of frames
    return fa->stack[--fa->top];
}

static inline void frame_free(struct frame_allocator *fa, uint64_t addr) {
    fa->stack[fa->top++] = addr;
}

// Initialize: push all frame addresses onto the stack
void frame_allocator_init(struct frame_allocator *fa, uint32_t num_frames,
                          uint32_t frame_size) {
    fa->capacity = num_frames;
    fa->top      = num_frames;
    fa->stack    = malloc(num_frames * sizeof(uint64_t));
    for (uint32_t i = 0; i < num_frames; i++)
        fa->stack[i] = i * frame_size;
}
```

### Memory Considerations

For zero-copy mode, the kernel maps UMEM frames directly into DMA-accessible
memory. This imposes constraints:

- Frame size must be a power of two (typically 2048 or 4096).
- Hugepages (2 MB or 1 GB) improve TLB hit rates. With 4096 standard pages
  covering 16 MB of UMEM, TLB pressure is significant at high packet rates.
- The UMEM region must remain pinned in physical memory for the lifetime of the
  socket.
- Alignment to page boundaries avoids partial-page DMA mappings.

```c
// Allocate UMEM with hugepages for optimal DMA performance
void *umem_area = mmap(NULL, UMEM_SIZE,
                       PROT_READ | PROT_WRITE,
                       MAP_PRIVATE | MAP_ANONYMOUS | MAP_HUGETLB,
                       -1, 0);
if (umem_area == MAP_FAILED) {
    // Fallback to standard pages
    umem_area = mmap(NULL, UMEM_SIZE,
                     PROT_READ | PROT_WRITE,
                     MAP_PRIVATE | MAP_ANONYMOUS,
                     -1, 0);
}
```

## 2. Ring Buffer Producer/Consumer Protocol

### Ring Structure

Each ring is a single-producer, single-consumer (SPSC) structure backed by a
power-of-two-sized array. The producer and consumer maintain independent
monotonically increasing counters. The actual array index is derived via
bitmask:

```
Ring size: N (must be power of 2)
Mask:      N - 1

Producer counter: P (only producer writes, consumer reads)
Consumer counter: C (only consumer writes, producer reads)

Available for production: N - (P - C)
Available for consumption: P - C

Array index: counter & mask
```

### Memory Ordering

The rings use acquire/release semantics to ensure visibility across the
kernel-userspace boundary:

```c
// Producer side (e.g., userspace writing to FILL ring)
static inline void ring_submit(struct ring *r, uint32_t count) {
    // Ensure all descriptor writes are visible before advancing producer
    __atomic_store_n(r->producer, *r->producer + count, __ATOMIC_RELEASE);
}

// Consumer side (e.g., kernel reading from FILL ring)
static inline uint32_t ring_peek(struct ring *r, uint32_t *idx) {
    uint32_t entries = __atomic_load_n(r->producer, __ATOMIC_ACQUIRE)
                     - *r->cached_consumer;
    if (entries > 0)
        *idx = *r->cached_consumer;
    return entries;
}
```

### The Four Rings in Detail

```
Ring          Direction            Purpose
----          ---------            -------
FILL          user -> kernel       "Here are empty frames you can write RX packets into"
RX            kernel -> user       "Here are frames now containing received packets"
TX            user -> kernel       "Here are frames containing packets to transmit"
COMPLETION    kernel -> user       "These TX frames are done; you can reuse them"

Data flow (receive path):
  user allocates frame -> FILL ring -> kernel DMA writes packet -> RX ring -> user processes

Data flow (transmit path):
  user writes packet -> TX ring -> kernel DMA reads packet -> COMPLETION ring -> user frees frame
```

### Descriptor Format

```c
// RX and TX ring descriptors (struct xdp_desc)
struct xdp_desc {
    __u64 addr;     // offset into UMEM (frame address + headroom)
    __u32 len;      // packet length in bytes
    __u32 options;  // flags (e.g., XDP_PKT_CONTD for multi-buffer)
};

// FILL and COMPLETION rings use bare uint64_t addresses
// FILL:       userspace writes frame base addresses
// COMPLETION: kernel returns frame base addresses after TX
```

## 3. XDP Program for AF_XDP Redirection

### Minimal Steering Program

The XDP program's sole job is to redirect packets to the correct AF_XDP socket
via the XSKMAP. The map is keyed by RX queue index, and userspace populates it
when binding sockets.

```c
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

struct {
    __uint(type, BPF_MAP_TYPE_XSKMAP);
    __uint(max_entries, 64);
    __type(key, int);
    __type(value, int);
} xsks_map SEC(".maps");

SEC("xdp")
int xdp_redirect_xsk(struct xdp_md *ctx) {
    int index = ctx->rx_queue_index;

    // Only redirect if a socket is bound to this queue
    if (bpf_map_lookup_elem(&xsks_map, &index))
        return bpf_redirect_map(&xsks_map, index, XDP_PASS);

    // No socket on this queue -- pass to normal stack
    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
```

### Selective Steering (Filter by Protocol)

A more practical program that only redirects specific traffic to AF_XDP,
allowing everything else to proceed through the normal kernel stack:

```c
SEC("xdp")
int xdp_filter_and_redirect(struct xdp_md *ctx) {
    void *data     = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    // Only redirect UDP traffic to AF_XDP
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;

    if (ip->protocol != IPPROTO_UDP)
        return XDP_PASS;

    int index = ctx->rx_queue_index;
    if (bpf_map_lookup_elem(&xsks_map, &index))
        return bpf_redirect_map(&xsks_map, index, XDP_PASS);

    return XDP_PASS;
}
```

### XSKMAP Population from Userspace

```c
// After creating the XSK socket, register it in the XSKMAP
int xsk_fd  = xsk_socket__fd(xsk);
int queue_id = 0;  // RX queue this socket is bound to

// Get map FD from the loaded BPF program
int map_fd = bpf_object__find_map_fd_by_name(bpf_obj, "xsks_map");

// Insert socket FD into map at the queue index
bpf_map_update_elem(map_fd, &queue_id, &xsk_fd, BPF_ANY);
```

## 4. Zero-Copy DMA Mapping

### How Zero-Copy Works

In copy mode, the kernel copies packet data from the driver's DMA buffer into
the UMEM frame. In zero-copy mode, the driver's DMA ring directly references
UMEM frames, eliminating the copy entirely.

```
Copy Mode:
  NIC -> DMA buffer (driver) -> memcpy -> UMEM frame -> userspace
  Cost: 1 copy per packet (~64 ns for 64-byte packet on modern CPU)

Zero-Copy Mode:
  NIC -> DMA directly into UMEM frame -> userspace
  Cost: 0 copies per packet

  The driver programs its DMA descriptors to point at UMEM frame addresses
  provided via the FILL ring. When a packet arrives, the NIC writes directly
  into the UMEM region that userspace will read.
```

### Driver Requirements

Zero-copy requires explicit driver support. The driver must implement the
`ndo_bpf` netdev operation with `XDP_SETUP_XSK_POOL`:

```
Drivers with zero-copy AF_XDP support:
  i40e      - Intel 40 GbE (XL710, X710)
  ice       - Intel 100 GbE (E810)
  ixgbe     - Intel 10 GbE (82599, X520)
  mlx5      - Mellanox ConnectX-4/5/6
  bnxt      - Broadcom NetXtreme
  stmmac    - Synopsys DWC Ethernet
```

### DMA Address Translation

The kernel pins UMEM pages and computes DMA addresses for each frame. The
driver stores these in its descriptor ring:

```
Frame index      UMEM offset       Physical address    DMA address
-----------      -----------       ----------------    -----------
0                0x0000            0x1a000000          0x1a000000
1                0x1000            0x1a001000          0x1a001000
2                0x2000            0x1a002000          0x1a002000
...              ...               ...                 ...
```

When using IOMMU (e.g., Intel VT-d), the DMA address is an IOVA that the IOMMU
translates to the physical address. This adds a small latency (~10-20 ns) but
enables safe DMA without pinning physical pages.

## 5. Performance Comparison

### Throughput (64-byte packets, single core)

```
Framework        RX (Mpps)    TX (Mpps)    Latency (us)    Notes
---------        ---------    ---------    ------------    -----
Kernel stack     1.0          1.2          15-30           Full protocol processing
Raw socket       1.0          1.0          15-30           Kernel stack overhead
AF_XDP copy      2-5          2-5          5-10            Copy from DMA buffer
AF_XDP zero-copy 14-24        14-24        3-5             Driver-dependent
DPDK             30-40        30-40        1-2             Dedicated NIC, PMD
```

### Cost per Packet (Approximate CPU Cycles)

```
Operation                    Cycles     Notes
---------                    ------     -----
NIC DMA to memory            0          Hardware, async
XDP program execution        50-200     BPF JIT compiled
Ring descriptor update        10-20      Cache-line write
Syscall (sendmsg/recvmsg)   200-400     Only in wakeup mode
Busy-poll check              10-30      Avoids syscall
memcpy (64B, copy mode)      30-50      L1 cache hit
Context switch (IRQ)         1000-3000  Interrupt handler
```

### Scaling with Multiple Queues

Throughput scales linearly with the number of hardware RX queues, each served
by a dedicated AF_XDP socket and CPU core:

```
Queues    AF_XDP ZC (Mpps)    DPDK (Mpps)    Notes
------    ----------------    -----------    -----
1         14-24               30-40          Single core
2         28-48               60-80          RSS distribution
4         56-96               120-160        Typical server
8         100-180             200-300        High-end NIC
```

The AF_XDP throughput ceiling per queue depends on the driver and NIC. Intel
E810 (ice driver) achieves approximately 24 Mpps per queue in zero-copy mode.
Mellanox ConnectX-6 (mlx5) reaches similar numbers.

## 6. libbpf xsk API Walkthrough

### Complete Setup Sequence

The libbpf `xsk.h` API provides the canonical way to create and manage AF_XDP
sockets. The full initialization sequence:

```c
#include <xdp/xsk.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>

// Step 1: Allocate UMEM memory
#define NUM_FRAMES  4096
#define FRAME_SIZE  4096
size_t umem_size = NUM_FRAMES * FRAME_SIZE;

void *umem_area = mmap(NULL, umem_size, PROT_READ | PROT_WRITE,
                       MAP_PRIVATE | MAP_ANONYMOUS | MAP_HUGETLB, -1, 0);

// Step 2: Create UMEM object
struct xsk_umem_config umem_cfg = {
    .fill_size      = XSK_RING_PROD__DEFAULT_NUM_DESCS,  // 2048
    .comp_size      = XSK_RING_CONS__DEFAULT_NUM_DESCS,  // 2048
    .frame_size     = FRAME_SIZE,
    .frame_headroom = XDP_PACKET_HEADROOM,                // 256
    .flags          = 0,
};

struct xsk_umem *umem = NULL;
struct xsk_ring_prod fill_q;
struct xsk_ring_cons comp_q;

int ret = xsk_umem__create(&umem, umem_area, umem_size,
                           &fill_q, &comp_q, &umem_cfg);

// Step 3: Initialize frame allocator and populate FILL ring
struct frame_allocator fa;
frame_allocator_init(&fa, NUM_FRAMES, FRAME_SIZE);

uint32_t idx;
xsk_ring_prod__reserve(&fill_q, NUM_FRAMES / 2, &idx);
for (int i = 0; i < NUM_FRAMES / 2; i++)
    *xsk_ring_prod__fill_addr(&fill_q, idx++) = frame_alloc(&fa);
xsk_ring_prod__submit(&fill_q, NUM_FRAMES / 2);

// Step 4: Load XDP program
struct bpf_object *bpf_obj = bpf_object__open_file("xdp_redirect.o", NULL);
bpf_object__load(bpf_obj);
struct bpf_program *prog = bpf_object__find_program_by_name(bpf_obj,
                                                             "xdp_redirect_xsk");
int prog_fd = bpf_program__fd(prog);

// Step 5: Attach XDP program to interface
int ifindex = if_nametoindex("eth0");
bpf_xdp_attach(ifindex, prog_fd, XDP_FLAGS_DRV_MODE, NULL);

// Step 6: Create XSK socket
struct xsk_socket_config xsk_cfg = {
    .rx_size      = XSK_RING_CONS__DEFAULT_NUM_DESCS,
    .tx_size      = XSK_RING_PROD__DEFAULT_NUM_DESCS,
    .bind_flags   = XDP_ZEROCOPY,
    .xdp_flags    = XDP_FLAGS_DRV_MODE,
    .libbpf_flags = XSK_LIBBPF_FLAGS__INHIBIT_PROG_LOAD,
};

struct xsk_socket *xsk = NULL;
struct xsk_ring_cons rx_q;
struct xsk_ring_prod tx_q;

ret = xsk_socket__create(&xsk, "eth0", 0, umem, &rx_q, &tx_q, &xsk_cfg);

// Step 7: Register socket in XSKMAP
int map_fd = bpf_object__find_map_fd_by_name(bpf_obj, "xsks_map");
int xsk_fd = xsk_socket__fd(xsk);
int queue_id = 0;
bpf_map_update_elem(map_fd, &queue_id, &xsk_fd, BPF_ANY);
```

### Receive Loop

```c
void rx_loop(struct xsk_socket *xsk, struct xsk_ring_cons *rx_q,
             struct xsk_ring_prod *fill_q, struct frame_allocator *fa,
             void *umem_area) {
    struct pollfd fds = {
        .fd     = xsk_socket__fd(xsk),
        .events = POLLIN,
    };

    while (running) {
        uint32_t idx_rx = 0, idx_fill = 0;
        unsigned int rcvd;

        // Peek for received packets
        rcvd = xsk_ring_cons__peek(rx_q, BATCH_SIZE, &idx_rx);
        if (rcvd == 0) {
            poll(&fds, 1, 100);
            continue;
        }

        // Reserve space in FILL ring for replacement frames
        while (xsk_ring_prod__reserve(fill_q, rcvd, &idx_fill) < rcvd)
            poll(&fds, 1, 10);

        for (unsigned int i = 0; i < rcvd; i++) {
            const struct xdp_desc *desc =
                xsk_ring_cons__rx_desc(rx_q, idx_rx++);
            uint8_t *pkt = xsk_umem__get_data(umem_area, desc->addr);

            process_packet(pkt, desc->len);

            // Return consumed frame and provide a fresh one
            frame_free(fa, desc->addr);
            *xsk_ring_prod__fill_addr(fill_q, idx_fill++) = frame_alloc(fa);
        }

        xsk_ring_cons__release(rx_q, rcvd);
        xsk_ring_prod__submit(fill_q, rcvd);
    }
}
```

## 7. Batching Strategies for Throughput

### Why Batching Matters

At 14.88 Mpps (10 GbE line rate), each packet arrives every 67.2 ns. Processing
packets one at a time incurs per-packet overhead from ring operations and
potential syscalls. Batching amortizes this cost.

### Ring Operation Batching

Reserve and submit multiple ring entries in a single operation:

```c
#define BATCH_SIZE 64

// Bad: one-at-a-time (high overhead)
for (int i = 0; i < 64; i++) {
    xsk_ring_prod__reserve(&fill_q, 1, &idx);
    *xsk_ring_prod__fill_addr(&fill_q, idx) = frame_alloc(&fa);
    xsk_ring_prod__submit(&fill_q, 1);  // memory barrier per packet
}

// Good: batched (amortized overhead)
uint32_t reserved = xsk_ring_prod__reserve(&fill_q, BATCH_SIZE, &idx);
for (uint32_t i = 0; i < reserved; i++)
    *xsk_ring_prod__fill_addr(&fill_q, idx + i) = frame_alloc(&fa);
xsk_ring_prod__submit(&fill_q, reserved);  // single memory barrier
```

### Syscall Amortization

The `sendmsg()` / `recvmsg()` syscalls wake the kernel to process rings. In
non-busy-poll mode, minimizing syscall frequency is critical:

```c
// Amortize wakeup: only call sendmsg after submitting a full batch
uint32_t pending_tx = 0;

void tx_enqueue(struct xsk_ring_prod *tx_q, uint64_t addr, uint32_t len) {
    uint32_t idx;
    xsk_ring_prod__reserve(tx_q, 1, &idx);
    struct xdp_desc *desc = xsk_ring_prod__tx_desc(tx_q, idx);
    desc->addr = addr;
    desc->len  = len;
    xsk_ring_prod__submit(tx_q, 1);
    pending_tx++;

    if (pending_tx >= BATCH_SIZE) {
        sendmsg(xsk_socket__fd(xsk), &msg, MSG_DONTWAIT);
        pending_tx = 0;
    }
}
```

### Prefetching

For packet processing workloads, prefetch the next packet while processing the
current one:

```c
for (unsigned int i = 0; i < rcvd; i++) {
    const struct xdp_desc *desc = xsk_ring_cons__rx_desc(rx_q, idx + i);
    uint8_t *pkt = xsk_umem__get_data(umem_area, desc->addr);

    // Prefetch next packet into L1 cache
    if (i + 1 < rcvd) {
        const struct xdp_desc *next = xsk_ring_cons__rx_desc(rx_q, idx + i + 1);
        __builtin_prefetch(xsk_umem__get_data(umem_area, next->addr), 0, 3);
    }

    process_packet(pkt, desc->len);
}
```

### Batch Size Selection

```
Batch size    Throughput impact    Latency impact    Notes
----------    ----------------    --------------    -----
1             Baseline            Lowest            Per-packet syscall/barrier
16            ~2x improvement     Minor increase    Good for low-latency
32            ~3x improvement     Moderate           Sweet spot for most workloads
64            ~3.5x improvement   Higher            Optimal for throughput
128+          Diminishing returns  High              Ring contention risk
```

The optimal batch size depends on the workload. For pure forwarding (minimal
per-packet processing), 32-64 maximizes throughput. For compute-heavy processing
(e.g., DPI, regex matching), smaller batches (16) keep latency bounded.

### Combined RX/TX Batching

For forwarding applications, combine receive and transmit batches:

```c
while (running) {
    // Batch receive
    uint32_t idx_rx;
    unsigned int rcvd = xsk_ring_cons__peek(&rx_q, BATCH_SIZE, &idx_rx);

    // Batch transmit -- rewrite headers in-place and forward
    uint32_t idx_tx;
    if (rcvd > 0 && xsk_ring_prod__reserve(&tx_q, rcvd, &idx_tx) == rcvd) {
        for (unsigned int i = 0; i < rcvd; i++) {
            const struct xdp_desc *rx_desc =
                xsk_ring_cons__rx_desc(&rx_q, idx_rx + i);

            // Modify packet in-place (same UMEM frame)
            uint8_t *pkt = xsk_umem__get_data(umem_area, rx_desc->addr);
            rewrite_headers(pkt, rx_desc->len);

            // Submit on TX ring with same address
            struct xdp_desc *tx_desc =
                xsk_ring_prod__tx_desc(&tx_q, idx_tx + i);
            tx_desc->addr = rx_desc->addr;
            tx_desc->len  = rx_desc->len;
        }

        xsk_ring_cons__release(&rx_q, rcvd);
        xsk_ring_prod__submit(&tx_q, rcvd);
        sendmsg(xsk_socket__fd(xsk), &msg, MSG_DONTWAIT);
    }

    // Reclaim completed TX frames
    uint32_t idx_comp;
    unsigned int completed = xsk_ring_cons__peek(&comp_q, BATCH_SIZE, &idx_comp);
    if (completed > 0) {
        for (unsigned int i = 0; i < completed; i++)
            frame_free(&fa, *xsk_ring_cons__comp_addr(&comp_q, idx_comp + i));
        xsk_ring_cons__release(&comp_q, completed);
    }

    // Replenish FILL ring
    uint32_t idx_fill;
    if (xsk_ring_prod__reserve(&fill_q, completed, &idx_fill) == completed) {
        for (unsigned int i = 0; i < completed; i++)
            *xsk_ring_prod__fill_addr(&fill_q, idx_fill + i) = frame_alloc(&fa);
        xsk_ring_prod__submit(&fill_q, completed);
    }
}
```
