# DPDK (Data Plane Development Kit)

High-performance, kernel-bypass packet processing framework for fast data plane applications.

## Overview

DPDK moves network I/O from the kernel into userspace, eliminating system call
overhead, context switches, and interrupt processing. Applications poll NICs
directly through poll-mode drivers (PMDs), achieving tens of millions of packets
per second on commodity hardware.

## Architecture

```
+--------------------------------------------------+
|  Application (l2fwd, l3fwd, OVS-DPDK, VPP, ...)  |
+--------------------------------------------------+
|  DPDK Libraries                                    |
|  rte_ethdev | rte_mbuf | rte_ring | rte_hash | .. |
+--------------------------------------------------+
|  EAL (Environment Abstraction Layer)               |
|  hugepages | CPU affinity | PCI | memory mgmt      |
+--------------------------------------------------+
|  Poll-Mode Drivers (PMDs)                          |
|  ixgbe | i40e | mlx5 | virtio | af_xdp | ...      |
+--------------------------------------------------+
|  Hardware (NICs bound to vfio-pci / uio)           |
+--------------------------------------------------+
```

## NIC Binding

```bash
# Show NIC status (bound vs unbound)
dpdk-devbind.py --status

# Bind NIC to DPDK-compatible driver
modprobe vfio-pci
dpdk-devbind.py --bind=vfio-pci 0000:03:00.0

# Bind using UIO (older, less secure)
modprobe uio_pci_generic
dpdk-devbind.py --bind=uio_pci_generic 0000:03:00.0

# Unbind and return to kernel driver
dpdk-devbind.py --bind=ixgbe 0000:03:00.0

# Show detailed device info
dpdk-devbind.py --status-dev net
```

## Hugepage Setup

```bash
# Reserve 1024 x 2MB hugepages
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# Reserve 4 x 1GB hugepages (better for DPDK)
echo 4 > /sys/kernel/mm/hugepages/hugepages-1048576kB/nr_hugepages

# Mount hugepages filesystem
mkdir -p /dev/hugepages
mount -t hugetlbfs nodev /dev/hugepages

# Persistent via /etc/fstab
# nodev /dev/hugepages hugetlbfs defaults 0 0

# NUMA-aware allocation
echo 2 > /sys/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages
echo 2 > /sys/devices/system/node/node1/hugepages/hugepages-1048576kB/nr_hugepages

# Check allocation
cat /proc/meminfo | grep -i huge
```

## EAL Initialization

```bash
# Common EAL parameters
./my_app \
  -l 0-3 \               # logical cores to use
  -n 4 \                  # memory channels
  --socket-mem 1024,1024 \ # MB per NUMA socket
  --huge-dir /dev/hugepages \
  --file-prefix myapp \   # allow multiple DPDK processes
  -- \
  <app-specific args>

# Run on specific cores with core mask
./my_app -c 0xf -n 4 -- -p 0x3

# Use specific PCI devices only
./my_app -l 0-3 -n 4 -a 0000:03:00.0 -a 0000:03:00.1
```

```c
/* EAL init in code */
#include <rte_eal.h>

int main(int argc, char *argv[]) {
    int ret = rte_eal_init(argc, argv);
    if (ret < 0)
        rte_exit(EXIT_FAILURE, "EAL init failed\n");

    argc -= ret;
    argv += ret;
    /* application logic follows */
}
```

## Mbuf Pool and Packet I/O

```c
#include <rte_mbuf.h>
#include <rte_ethdev.h>

/* Create mbuf memory pool */
struct rte_mempool *mbuf_pool = rte_pktmbuf_pool_create(
    "MBUF_POOL",         /* name */
    8192,                /* number of mbufs */
    256,                 /* cache size per core */
    0,                   /* private data size */
    RTE_MBUF_DEFAULT_BUF_SIZE,  /* data room size */
    rte_socket_id()      /* NUMA socket */
);

/* Receive burst of packets */
#define BURST_SIZE 32
struct rte_mbuf *rx_pkts[BURST_SIZE];
uint16_t nb_rx = rte_eth_rx_burst(port_id, queue_id, rx_pkts, BURST_SIZE);

/* Process and transmit */
for (int i = 0; i < nb_rx; i++) {
    /* access packet data */
    uint8_t *pkt = rte_pktmbuf_mtod(rx_pkts[i], uint8_t *);
    uint16_t len = rte_pktmbuf_pkt_len(rx_pkts[i]);
    /* ... modify packet ... */
}

uint16_t nb_tx = rte_eth_tx_burst(port_id, queue_id, rx_pkts, nb_rx);

/* Free unsent packets */
for (int i = nb_tx; i < nb_rx; i++)
    rte_pktmbuf_free(rx_pkts[i]);
```

## Port Configuration

```c
/* Configure Ethernet port */
struct rte_eth_conf port_conf = {
    .rxmode = {
        .mq_mode = RTE_ETH_MQ_RX_RSS,   /* enable RSS */
        .offloads = RTE_ETH_RX_OFFLOAD_CHECKSUM,
    },
    .rx_adv_conf = {
        .rss_conf = {
            .rss_key = NULL,  /* use default key */
            .rss_hf = RTE_ETH_RSS_IP | RTE_ETH_RSS_TCP | RTE_ETH_RSS_UDP,
        },
    },
    .txmode = {
        .mq_mode = RTE_ETH_MQ_TX_NONE,
        .offloads = RTE_ETH_TX_OFFLOAD_MBUF_FAST_FREE,
    },
};

rte_eth_dev_configure(port_id, nb_rx_queues, nb_tx_queues, &port_conf);

/* Setup RX queues */
for (int q = 0; q < nb_rx_queues; q++)
    rte_eth_rx_queue_setup(port_id, q, 1024, rte_eth_dev_socket_id(port_id),
                           NULL, mbuf_pool);

/* Setup TX queues */
for (int q = 0; q < nb_tx_queues; q++)
    rte_eth_tx_queue_setup(port_id, q, 1024, rte_eth_dev_socket_id(port_id),
                           NULL);

/* Start the port */
rte_eth_dev_start(port_id);
rte_eth_promiscuous_enable(port_id);
```

## Ring Buffers

```c
#include <rte_ring.h>

/* Create lockless multi-producer multi-consumer ring */
struct rte_ring *ring = rte_ring_create("MY_RING", 1024,
                                         rte_socket_id(),
                                         RING_F_MP_HTS_ENQ | RING_F_MC_HTS_DEQ);

/* Single-producer single-consumer (fastest) */
struct rte_ring *sp_ring = rte_ring_create("SP_RING", 1024,
                                            rte_socket_id(),
                                            RING_F_SP_ENQ | RING_F_SC_DEQ);

/* Enqueue / dequeue */
void *obj;
rte_ring_enqueue(ring, obj);
rte_ring_dequeue(ring, &obj);

/* Burst operations */
rte_ring_enqueue_burst(ring, objs, n, NULL);
rte_ring_dequeue_burst(ring, objs, n, NULL);
```

## Hash, LPM, and ACL Libraries

```c
/* Exact-match hash table */
#include <rte_hash.h>
struct rte_hash_parameters params = {
    .name = "flow_table",
    .entries = 65536,
    .key_len = sizeof(struct flow_key),
    .hash_func = rte_hash_crc,
    .socket_id = rte_socket_id(),
};
struct rte_hash *ht = rte_hash_create(&params);
rte_hash_add_key_data(ht, &key, data);

/* Longest Prefix Match (IPv4 routing) */
#include <rte_lpm.h>
struct rte_lpm *lpm = rte_lpm_create("route_table", rte_socket_id(), &config);
rte_lpm_add(lpm, ipv4_addr, prefix_len, next_hop);
rte_lpm_lookup(lpm, dst_ip, &next_hop);

/* ACL classification */
#include <rte_acl.h>
/* Define field defs, create context, add rules, build, classify */
```

## Flow Classification (rte_flow)

```c
#include <rte_flow.h>

/* Steer TCP port 80 traffic to queue 1 */
struct rte_flow_attr attr = { .ingress = 1 };

struct rte_flow_item_eth eth_spec = { /* ... */ };
struct rte_flow_item_ipv4 ipv4_spec = { /* ... */ };
struct rte_flow_item_tcp tcp_spec = { .hdr.dst_port = rte_cpu_to_be_16(80) };
struct rte_flow_item_tcp tcp_mask = { .hdr.dst_port = 0xFFFF };

struct rte_flow_item pattern[] = {
    { .type = RTE_FLOW_ITEM_TYPE_ETH },
    { .type = RTE_FLOW_ITEM_TYPE_IPV4 },
    { .type = RTE_FLOW_ITEM_TYPE_TCP, .spec = &tcp_spec, .mask = &tcp_mask },
    { .type = RTE_FLOW_ITEM_TYPE_END },
};

struct rte_flow_action_queue queue = { .index = 1 };
struct rte_flow_action actions[] = {
    { .type = RTE_FLOW_ACTION_TYPE_QUEUE, .conf = &queue },
    { .type = RTE_FLOW_ACTION_TYPE_END },
};

struct rte_flow_error error;
struct rte_flow *flow = rte_flow_create(port_id, &attr, pattern, actions, &error);
```

## testpmd

```bash
# Launch testpmd (interactive packet forwarder)
dpdk-testpmd -l 0-3 -n 4 -- -i --portmask=0x3

# Inside testpmd shell
testpmd> show port info all
testpmd> set fwd macswap        # swap src/dst MAC
testpmd> set fwd io             # raw I/O forwarding
testpmd> set burst 64
testpmd> start                  # begin forwarding
testpmd> show port stats all
testpmd> stop
testpmd> quit

# Non-interactive with stats
dpdk-testpmd -l 0-3 -n 4 -- --portmask=0x3 --forward-mode=macswap \
  --stats-period=1
```

## Example Applications

```bash
# l2fwd -- Layer 2 forwarding
dpdk-l2fwd -l 0-3 -n 4 -- -p 0x3 -T 1

# l3fwd -- Layer 3 forwarding (LPM or hash-based)
dpdk-l3fwd -l 0-3 -n 4 -- -p 0x3 --config="(0,0,1),(1,0,2)" \
  --parse-ptype --lookup lpm

# skeleton -- minimal DPDK app template
# Found in examples/skeleton/ in DPDK source
```

## Multi-Core Patterns

```
Run-to-Completion (each core handles full pipeline):
  Core 0: RX -> Parse -> Process -> TX
  Core 1: RX -> Parse -> Process -> TX
  Core 2: RX -> Parse -> Process -> TX

Pipeline (stages split across cores):
  Core 0: RX -> ring0
  Core 1: ring0 -> Parse -> Process -> ring1
  Core 2: ring1 -> TX

# Run-to-completion: simpler, better cache locality
# Pipeline: better for unequal stage costs, more flexible
```

## OVS-DPDK

```bash
# Install OVS with DPDK support
ovs-vsctl set Open_vSwitch . other_config:dpdk-init=true
ovs-vsctl set Open_vSwitch . other_config:dpdk-socket-mem="1024,1024"
ovs-vsctl set Open_vSwitch . other_config:dpdk-lcore-mask=0x3

# Create DPDK bridge
ovs-vsctl add-br br0 -- set bridge br0 datapath_type=netdev
ovs-vsctl add-port br0 dpdk-p0 -- set Interface dpdk-p0 type=dpdk \
  options:dpdk-devargs=0000:03:00.0
ovs-vsctl add-port br0 dpdk-p1 -- set Interface dpdk-p1 type=dpdk \
  options:dpdk-devargs=0000:03:00.1

# Add vhost-user port for VM
ovs-vsctl add-port br0 vhost-user0 -- set Interface vhost-user0 \
  type=dpdkvhostuserclient \
  options:vhost-server-path="/tmp/sock0"
```

## Virtio and Vhost (VM Networking)

```bash
# Launch QEMU with vhost-user backend (connected to OVS-DPDK)
qemu-system-x86_64 \
  -chardev socket,id=char0,path=/tmp/sock0,server=on \
  -netdev vhost-user,id=net0,chardev=char0,vhostforce=on \
  -device virtio-net-pci,netdev=net0

# Inside guest: use DPDK virtio PMD for best performance
dpdk-testpmd -l 0-1 -n 4 -- -i
```

## Performance: DPDK vs Kernel

```
Metric             | Kernel Stack | DPDK
--------------------|-------------|------
64-byte pps (10G)  | ~1 Mpps     | ~14.8 Mpps
Latency (avg)      | 15-30 us    | 2-5 us
CPU per packet     | ~1000 cyc   | ~80-200 cyc
System calls/pkt   | 2+          | 0
Context switches   | yes         | no (busy-poll)
Interrupts         | yes         | no (poll-mode)
```

## DPDK vs XDP vs AF_XDP

```
Feature          | DPDK         | XDP            | AF_XDP
-----------------|--------------|----------------|--------
Runs in          | userspace    | kernel (driver)| userspace
Kernel bypass    | full         | no             | partial
NIC binding      | required     | no             | no
Hugepages        | required     | no             | no
Driver support   | PMD needed   | driver hook    | XDP + XSKMAP
Programming      | C (DPDK API) | C (BPF)        | C (socket API)
Max pps (10G)    | ~14.8 Mpps   | ~24 Mpps       | ~10 Mpps
Packet modify    | full         | limited        | full
Kernel features  | none         | all            | most
```

## Tips

- Always pin DPDK threads to dedicated cores with isolcpus to prevent scheduler interference
- Use 1GB hugepages over 2MB for fewer TLB misses and better performance
- Match mbuf pool size to expected burst sizes; undersized pools cause rx drops
- Allocate memory on the same NUMA node as the NIC to avoid cross-node penalties
- Use rte_eth_rx_burst/tx_burst with batch sizes of 32 or 64 for amortized overhead
- Prefer VFIO over UIO for production -- VFIO provides IOMMU isolation and is more secure
- Enable hardware offloads (checksum, TSO, RSS) to free CPU cycles
- Monitor stats via rte_eth_stats_get() and rte_eth_xstats_get() rather than ethtool
- Use rte_prefetch0() on the next mbuf in a burst loop to hide memory latency
- For containers, use AF_XDP PMD to avoid NIC unbinding from the kernel
- Set RTE_MBUF_F_INDIRECT on cloned mbufs to avoid double-free
- Compile with -march=native to enable platform-specific SIMD optimizations

## See Also

- XDP (in-kernel high-performance packet processing, no kernel bypass)
- AF_XDP (hybrid approach: XDP redirect to userspace sockets)

## References

- [DPDK Official Documentation](https://doc.dpdk.org/guides/)
- [DPDK API Reference](https://doc.dpdk.org/api/)
- [DPDK Getting Started Guide](https://doc.dpdk.org/guides/linux_gsg/)
- [DPDK Sample Applications](https://doc.dpdk.org/guides/sample_app_ug/)
- [OVS-DPDK Documentation](https://docs.openvswitch.org/en/latest/intro/install/dpdk/)
- [dpdk-devbind.py Reference](https://doc.dpdk.org/guides/tools/devbind.html)
- [DPDK Programmer's Guide](https://doc.dpdk.org/guides/prog_guide/)
- [Understanding DPDK (Red Hat)](https://www.redhat.com/en/blog/understanding-dpdk)
