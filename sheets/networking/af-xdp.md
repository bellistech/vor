# AF_XDP (Address Family XDP)

High-performance kernel bypass socket interface for userspace packet processing via XDP.

## Overview

AF_XDP provides a raw socket type (`AF_XDP` / `PF_XDP`) that delivers packets
directly from the NIC driver to userspace memory, bypassing the entire kernel
networking stack. An XDP program uses `XDP_REDIRECT` with an `XSKMAP` to steer
packets into AF_XDP sockets (XSKs). Shared memory (UMEM) and lock-free ring
buffers eliminate kernel-to-user copies.

## Core Concepts

```
XSK (XDP Socket)  — AF_XDP socket bound to a NIC queue
UMEM               — shared memory region between kernel and userspace
FILL ring          — userspace provides empty frames for the kernel to fill
COMPLETION ring    — kernel returns frames after TX completes
RX ring            — kernel delivers received packet descriptors
TX ring            — userspace submits packet descriptors for transmission
```

## UMEM Setup

```c
// Allocate UMEM — contiguous memory region divided into fixed-size frames
#define FRAME_SIZE  4096
#define NUM_FRAMES  4096
#define UMEM_SIZE   (FRAME_SIZE * NUM_FRAMES)

void *umem_area = mmap(NULL, UMEM_SIZE, PROT_READ | PROT_WRITE,
                       MAP_PRIVATE | MAP_ANONYMOUS | MAP_HUGETLBFS, -1, 0);

struct xsk_umem_config cfg = {
    .fill_size      = 2048,
    .comp_size      = 2048,
    .frame_size     = FRAME_SIZE,
    .frame_headroom = 0,
    .flags          = 0,
};

struct xsk_umem *umem;
struct xsk_ring_prod fill_ring;
struct xsk_ring_cons comp_ring;

xsk_umem__create(&umem, umem_area, UMEM_SIZE, &fill_ring, &comp_ring, &cfg);
```

## Ring Buffer Protocol

```
Producer                        Consumer
--------                        --------
1. Reserve N entries             1. Peek N entries
   xsk_ring_prod__reserve()        xsk_ring_cons__peek()
2. Write descriptors             2. Read descriptors
   *xsk_ring_prod__fill_addr()     *xsk_ring_cons__rx_desc()
3. Submit entries                3. Release entries
   xsk_ring_prod__submit()         xsk_ring_cons__release()

FILL ring:    userspace = producer,  kernel = consumer
RX ring:      kernel    = producer,  userspace = consumer
TX ring:      userspace = producer,  kernel = consumer
COMP ring:    kernel    = producer,  userspace = consumer
```

## XDP Program for AF_XDP Redirection

```c
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

struct {
    __uint(type, BPF_MAP_TYPE_XSKMAP);
    __uint(max_entries, 64);
    __type(key, int);
    __type(value, int);
} xsks_map SEC(".maps");

SEC("xdp")
int xdp_redirect_xsk(struct xdp_md *ctx) {
    int index = ctx->rx_queue_index;

    if (bpf_map_lookup_elem(&xsks_map, &index))
        return bpf_redirect_map(&xsks_map, index, XDP_PASS);

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
```

## Socket Creation with libbpf (xsk API)

```c
#include <xdp/xsk.h>

struct xsk_socket_config xsk_cfg = {
    .rx_size      = 2048,
    .tx_size      = 2048,
    .bind_flags   = XDP_COPY,          // or XDP_ZEROCOPY
    .xdp_flags    = XDP_FLAGS_DRV_MODE,
    .libbpf_flags = XSK_LIBBPF_FLAGS__INHIBIT_PROG_LOAD,
};

struct xsk_socket *xsk;
struct xsk_ring_cons rx_ring;
struct xsk_ring_prod tx_ring;

xsk_socket__create(&xsk, "eth0", 0 /* queue_id */, umem,
                   &rx_ring, &tx_ring, &xsk_cfg);
```

## Receive Path

```c
// 1. Pre-populate FILL ring with empty frame addresses
uint32_t idx;
xsk_ring_prod__reserve(&fill_ring, BATCH_SIZE, &idx);
for (int i = 0; i < BATCH_SIZE; i++)
    *xsk_ring_prod__fill_addr(&fill_ring, idx + i) = alloc_frame();
xsk_ring_prod__submit(&fill_ring, BATCH_SIZE);

// 2. Poll for received packets
unsigned int rcvd = xsk_ring_cons__peek(&rx_ring, BATCH_SIZE, &idx);
if (rcvd == 0) {
    // Optionally call recvmsg() or poll() to wake kernel
    recvmsg(xsk_socket__fd(xsk), &msg, MSG_DONTWAIT);
}

// 3. Process packets
for (int i = 0; i < rcvd; i++) {
    const struct xdp_desc *desc = xsk_ring_cons__rx_desc(&rx_ring, idx + i);
    uint8_t *pkt = xsk_umem__get_data(umem_area, desc->addr);
    // Process packet at pkt, length = desc->len
}
xsk_ring_cons__release(&rx_ring, rcvd);
```

## Transmit Path

```c
// 1. Write packet data into UMEM frame
uint64_t addr = alloc_frame();
uint8_t *pkt = xsk_umem__get_data(umem_area, addr);
memcpy(pkt, packet_data, packet_len);

// 2. Submit to TX ring
uint32_t idx;
xsk_ring_prod__reserve(&tx_ring, 1, &idx);
xsk_ring_prod__tx_desc(&tx_ring, idx)->addr = addr;
xsk_ring_prod__tx_desc(&tx_ring, idx)->len  = packet_len;
xsk_ring_prod__submit(&tx_ring, 1);

// 3. Kick the kernel to transmit
sendmsg(xsk_socket__fd(xsk), &msg, MSG_DONTWAIT);

// 4. Reclaim completed frames from COMPLETION ring
unsigned int completed = xsk_ring_cons__peek(&comp_ring, BATCH_SIZE, &idx);
for (int i = 0; i < completed; i++)
    free_frame(*xsk_ring_cons__comp_addr(&comp_ring, idx + i));
xsk_ring_cons__release(&comp_ring, completed);
```

## Zero-Copy vs Copy Mode

```
Mode         Flag          Requirements         Performance
----         ----          ------------         -----------
Copy         XDP_COPY      Any NIC, 4.18+       ~2-5 Mpps
Zero-copy    XDP_ZEROCOPY  Driver support, 5.4+  ~14-24 Mpps
```

```bash
# Check if driver supports zero-copy
ethtool -i eth0 | grep driver
# Drivers with zero-copy: i40e, ixgbe, mlx5, ice, bnxt

# Bind in zero-copy mode
./xdpsock -i eth0 -q 0 -z
```

## Busy-Poll Mode

```bash
# Enable busy-poll for lower latency (avoids interrupt-driven wakeup)
sysctl -w net.core.busy_poll=50
sysctl -w net.core.busy_read=50

# Per-socket via setsockopt
setsockopt(fd, SOL_SOCKET, SO_PREFER_BUSY_POLL, &one, sizeof(one));
setsockopt(fd, SOL_SOCKET, SO_BUSY_POLL, &timeout_us, sizeof(timeout_us));
setsockopt(fd, SOL_SOCKET, SO_BUSY_POLL_BUDGET, &budget, sizeof(budget));
```

## Multi-Queue Support

```bash
# Bind one XSK per hardware RX queue
# Each socket handles packets from its queue independently

# Show NIC queue count
ethtool -l eth0

# Set queue count to match worker threads
ethtool -L eth0 combined 4

# Use RSS (Receive Side Scaling) to distribute flows across queues
ethtool -X eth0 equal 4
```

```c
// Create one socket per queue
for (int q = 0; q < num_queues; q++) {
    xsk_socket__create(&xsk[q], "eth0", q, umem,
                       &rx_ring[q], &tx_ring[q], &xsk_cfg);
}
```

## xdpsock Sample Application

```bash
# Build from kernel source tree
cd linux/samples/bpf && make

# Receive mode
./xdpsock -i eth0 -r -q 0

# Transmit mode
./xdpsock -i eth0 -t -q 0

# Zero-copy mode
./xdpsock -i eth0 -r -z -q 0

# Bidirectional (RX + TX)
./xdpsock -i eth0 -l -q 0

# With poll() instead of busy-wait
./xdpsock -i eth0 -r -p

# Batch size
./xdpsock -i eth0 -r -b 64
```

## AF_XDP vs DPDK vs Raw Sockets

```
Feature              AF_XDP               DPDK                 Raw sockets
-------              ------               ----                 -----------
Kernel bypass        Partial (XDP path)   Full (UIO/VFIO)     None
NIC dedication       No (shared)          Yes (exclusive)      No (shared)
Kernel integration   Yes (XDP/eBPF)       No (PMD drivers)     Yes (full stack)
Throughput (64B)     14-24 Mpps           ~40 Mpps             ~1 Mpps
Latency              ~3-5 us              ~1-2 us              ~15-30 us
Hugepages            Optional             Required             N/A
Root required        CAP_NET_ADMIN+BPF    Yes (or VFIO)        CAP_NET_RAW
Kernel visibility    Full (ethtool, tc)   None                 Full
Live migration       Supported            Complex              Supported
Container support    Good (netns)         Limited              Good
```

## Kernel Version Requirements

```
Feature                          Minimum Kernel
-------                          --------------
AF_XDP socket support            4.18
XDP_REDIRECT to XSKMAP           4.18
Copy mode                        4.18
UMEM shared between sockets      5.1
Zero-copy mode                   5.4
Busy-poll support                5.11
Multi-buffer (jumbo frames)      6.3
XDP metadata to AF_XDP           6.3
```

## Use Cases

- **High-performance packet capture** -- selective capture at line rate without dropping
- **Custom protocol stacks** -- userspace TCP/UDP for specialized workloads
- **Load balancers** -- L4 load balancing with kernel-integrated health checks
- **Network monitoring** -- tap and mirror with flow-aware sampling
- **IDS/IPS** -- inline inspection without kernel module overhead
- **DNS servers** -- ultra-low-latency authoritative resolvers

## Tips

- Pre-populate the FILL ring before entering the receive loop; the kernel cannot deliver packets without empty frames
- Use batch operations (reserve/submit multiple descriptors) to amortize syscall overhead
- Align UMEM frame size to page boundaries (4096) for zero-copy compatibility
- Pin the XDP program and XSK to the same CPU core for cache locality
- Use `SO_PREFER_BUSY_POLL` for latency-sensitive workloads; use interrupt mode for power efficiency
- UMEM can be shared across multiple sockets on different queues to reduce memory usage
- Set `XDP_FLAGS_DRV_MODE` explicitly to fail fast if native mode is unsupported
- Monitor dropped frames via `bpftool map dump` on the XSKMAP and ethtool stats
- Use `XSK_LIBBPF_FLAGS__INHIBIT_PROG_LOAD` when loading your own XDP program separately
- Frame headroom (`frame_headroom` in UMEM config) reserves space for prepending headers
- Compile XDP programs with BTF (`-g`) for CO-RE portability across kernel versions
- For maximum throughput, use one socket per queue and one thread per socket

## See Also

- xdp (XDP framework that AF_XDP builds on)
- dpdk (full kernel bypass alternative with higher throughput ceiling)
- ebpf (the BPF subsystem underlying XDP and AF_XDP)

## References

- [AF_XDP Documentation (kernel.org)](https://www.kernel.org/doc/html/latest/networking/af_xdp.html)
- [AF_XDP Tutorial (xdp-project)](https://github.com/xdp-project/xdp-tutorial/tree/master/advanced03-AF_XDP)
- [libbpf xsk.h API](https://github.com/libbpf/libbpf/blob/master/src/xsk.h)
- [libxdp Repository](https://github.com/xdp-project/xdp-tools/tree/master/lib/libxdp)
- [XDP Paper (ACM CoNEXT 2018)](https://dl.acm.org/doi/10.1145/3281411.3281443)
- [AF_XDP Kernel Selftests](https://github.com/torvalds/linux/tree/master/tools/testing/selftests/bpf)
- [Intel Ice Driver AF_XDP Support](https://www.kernel.org/doc/html/latest/networking/device_drivers/ethernet/intel/ice.html)
