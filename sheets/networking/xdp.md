# XDP (eXpress Data Path)

Programmable, high-performance packet processing at the earliest point in the Linux networking stack.

## Overview

XDP hooks BPF programs directly to the network driver's receive path, enabling
line-rate packet processing without kernel bypass. Packets are handled before
`sk_buff` allocation, eliminating per-packet memory overhead.

## Return Actions

```
XDP_PASS    — continue normal stack processing
XDP_DROP    — drop the packet (fastest possible discard)
XDP_TX      — bounce the packet back out the same NIC
XDP_REDIRECT — forward to another NIC, CPU, or AF_XDP socket
XDP_ABORTED — drop + trace point for debugging
```

## Attachment Modes

```bash
# Native (driver) mode — best performance, requires driver support
ip link set dev eth0 xdpdrv obj prog.o sec xdp

# Generic (SKB) mode — works on any NIC, slower path
ip link set dev eth0 xdpgeneric obj prog.o sec xdp

# Offloaded mode — runs on NIC hardware (Netronome SmartNICs)
ip link set dev eth0 xdpoffload obj prog.o sec xdp
```

## Minimal XDP Program (C)

```c
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("xdp")
int xdp_drop_all(struct xdp_md *ctx) {
    return XDP_DROP;
}

char _license[] SEC("license") = "GPL";
```

## Compile and Load

```bash
# Compile with clang
clang -O2 -g -target bpf -c xdp_prog.c -o xdp_prog.o

# Load with ip
ip link set dev eth0 xdp obj xdp_prog.o sec xdp

# Detach
ip link set dev eth0 xdp off

# Verify attachment
ip link show dev eth0
```

## xdp-tools

```bash
# Install xdp-tools (Fedora/RHEL)
dnf install xdp-tools

# Load program
xdp-loader load -m native eth0 xdp_prog.o

# Show attached programs
xdp-loader status

# Unload
xdp-loader unload eth0 --all

# Run built-in drop counter
xdp-filter apply eth0 --mode native
xdp-filter port 80 --remove
```

## AF_XDP Sockets (Zero-Copy to Userspace)

```bash
# Create AF_XDP socket and bind
# Requires: XDP_REDIRECT + bpf_redirect_map with XSKMAP

# Example with xdpsock sample
cd linux/samples/bpf
make xdpsock
./xdpsock -i eth0 -r  # receive mode
./xdpsock -i eth0 -t  # transmit mode
./xdpsock -i eth0 -z  # zero-copy mode
```

```c
// Redirect to AF_XDP socket from XDP program
struct {
    __uint(type, BPF_MAP_TYPE_XSKMAP);
    __uint(max_entries, 64);
    __type(key, int);
    __type(value, int);
} xsks_map SEC(".maps");

SEC("xdp")
int xdp_sock_prog(struct xdp_md *ctx) {
    int index = ctx->rx_queue_index;
    if (bpf_map_lookup_elem(&xsks_map, &index))
        return bpf_redirect_map(&xsks_map, index, 0);
    return XDP_PASS;
}
```

## bpf_redirect_map

```c
// Redirect between interfaces using DEVMAP
struct {
    __uint(type, BPF_MAP_TYPE_DEVMAP);
    __uint(max_entries, 256);
    __type(key, int);
    __type(value, int);
} tx_port SEC(".maps");

SEC("xdp")
int xdp_redirect_prog(struct xdp_md *ctx) {
    return bpf_redirect_map(&tx_port, 0, 0);
}
```

```bash
# Populate DEVMAP from userspace
bpftool map update id <map_id> key 0 0 0 0 value <ifindex> 0 0 0
```

## DDoS Mitigation Example

```c
// Simple SYN flood filter
SEC("xdp")
int xdp_ddos(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_ABORTED;

    if (eth->h_proto != htons(ETH_P_IP))
        return XDP_PASS;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_ABORTED;

    // Rate-limit per source IP using BPF map
    __u32 src = ip->saddr;
    __u64 *count = bpf_map_lookup_elem(&rate_map, &src);
    if (count && *count > THRESHOLD)
        return XDP_DROP;

    return XDP_PASS;
}
```

## Performance Benchmarks

```bash
# Measure XDP drop rate with pktgen
modprobe pktgen
echo "add_device eth0" > /proc/net/pktgen/kpktgend_0
# Configure and run...

# Typical throughput (single core, 64-byte packets):
# XDP_DROP native:     ~24 Mpps (million packets/sec)
# XDP_DROP generic:    ~5  Mpps
# XDP_TX native:       ~18 Mpps
# XDP_REDIRECT:        ~14 Mpps
# iptables DROP:       ~3  Mpps
# Normal stack:        ~1  Mpps

# Monitor with bpftool
bpftool prog show
bpftool map dump id <map_id>

# Trace XDP events
perf stat -e xdp:* -a sleep 1
```

## Debugging

```bash
# Trace XDP_ABORTED events
perf record -e xdp:xdp_exception -a
perf script

# View BPF verifier output
ip link set dev eth0 xdp obj prog.o verb

# Use bpf_printk for printf-style debugging
# Output goes to: /sys/kernel/debug/tracing/trace_pipe
cat /sys/kernel/debug/tracing/trace_pipe
```

## Multi-Program Chaining (libxdp)

```bash
# libxdp supports chaining multiple XDP programs via freplace
xdp-loader load -m native eth0 prog1.o prog2.o

# Programs run in sequence; first non-PASS verdict wins
# Dispatcher manages the chain automatically
```

## Kernel and Driver Requirements

```bash
# Check kernel XDP support
grep -i xdp /boot/config-$(uname -r)

# Drivers with native XDP support (partial list):
# i40e, ixgbe, mlx5, nfp, bnxt, virtio_net, veth, bond

# Check if driver supports XDP
ethtool -i eth0 | grep driver
```

## Tips

- Always validate packet bounds before accessing headers; the BPF verifier rejects unsafe access
- Use native mode in production; generic mode is for development and testing only
- XDP programs have a 4096-instruction limit per program (raised to 1M in 5.2+ with bounded loops)
- Tail calls (`bpf_tail_call`) let you chain logic beyond the instruction limit
- AF_XDP zero-copy requires driver support; check with `ethtool -i`
- Use `BPF_MAP_TYPE_PERCPU_ARRAY` for counters to avoid lock contention
- `XDP_TX` requires the NIC to have a TX queue available on the same CPU
- Pin BPF maps to `/sys/fs/bpf/` for sharing between programs and userspace
- Compile with `-O2` always; unoptimized BPF code will fail the verifier
- Test with `xdpgeneric` first, then switch to `xdpdrv` once logic is verified
- Use `bpf_xdp_adjust_head` to add/remove headers for encap/decap
- BTF (BPF Type Format) enables CO-RE for portable programs across kernel versions

## See Also

- tc/BPF (traffic control eBPF for egress processing)
- AF_XDP (kernel bypass sockets for userspace networking)
- DPDK (Data Plane Development Kit, full kernel bypass alternative)
- eBPF (the broader BPF ecosystem: tracing, security, networking)
- Cilium (Kubernetes CNI built on XDP and eBPF)

## References

- [XDP Tutorial (xdp-project)](https://github.com/xdp-project/xdp-tutorial)
- [BPF and XDP Reference Guide (Cilium)](https://docs.cilium.io/en/latest/bpf/)
- [AF_XDP Documentation (kernel.org)](https://www.kernel.org/doc/html/latest/networking/af_xdp.html)
- [xdp-tools Repository](https://github.com/xdp-project/xdp-tools)
- [XDP Paper (ACM CoNEXT 2018)](https://dl.acm.org/doi/10.1145/3281411.3281443)
- [Linux BPF Documentation](https://www.kernel.org/doc/html/latest/bpf/index.html)
