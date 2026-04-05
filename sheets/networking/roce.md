# RoCE (RDMA over Converged Ethernet)

High-performance networking protocol that enables Remote Direct Memory Access over Ethernet, delivering zero-copy data transfer with kernel bypass and minimal CPU overhead for HPC, AI/ML training, and NVMe-oF storage workloads.

## Concepts

### RDMA Fundamentals

- **Zero-Copy:** Data moves directly between application memory buffers on different hosts without intermediate kernel copies
- **Kernel Bypass:** Applications access the NIC (RNIC) directly via user-space verbs, bypassing the OS network stack entirely
- **CPU Offload:** The NIC handles protocol processing, segmentation, and reassembly — freeing CPU cycles for application work
- **OS Bypass:** No context switches, no system calls on the data path — latency drops to single-digit microseconds

### RoCE Versions

| Feature | RoCE v1 | RoCE v2 |
|:---|:---|:---|
| Layer | L2 only | L3 routable (UDP/IP) |
| EtherType | 0x8915 | 0x0800 (IP) |
| Transport | InfiniBand over Ethernet | InfiniBand over UDP/IPv4 or IPv6 |
| UDP port | N/A | 4791 |
| Routing | Same L2 broadcast domain only | Routable across subnets |
| ECMP | Not supported | Supported via UDP source port hash |
| VXLAN/overlay | Not compatible | Compatible (with caveats) |
| Adoption | Legacy, rare | Industry standard |

### RoCE vs iWARP vs InfiniBand

| Property | RoCE v2 | iWARP | InfiniBand |
|:---|:---|:---|:---|
| Transport | UDP/IP | TCP/IP | Native IB |
| Lossless requirement | Yes (PFC/ECN) | No (TCP handles loss) | Yes (credit-based) |
| Latency | ~1-2 us | ~5-10 us | ~0.5-1 us |
| CPU overhead | Very low | Higher (TCP offload) | Very low |
| Fabric | Standard Ethernet switches | Standard Ethernet switches | IB switches |
| Cost | Low (Ethernet reuse) | Low (Ethernet reuse) | High (dedicated fabric) |
| Scalability | Good (with DCQCN) | Good (TCP CC) | Excellent |

## RDMA Architecture

### Queue Pairs (QP)

- Each RDMA connection uses a **Queue Pair**: one Send Queue (SQ) + one Receive Queue (RQ)
- QP types: RC (Reliable Connected), UC (Unreliable Connected), UD (Unreliable Datagram)
- RC is most common — provides reliable, in-order delivery (like TCP semantics)
- Each QP has a QP Number (QPN) — 24-bit identifier

```bash
# List queue pairs on a device
rdma res show qp dev mlx5_0

# Show QP details with state
rdma res show qp dev mlx5_0 -j | python3 -m json.tool
```

### Completion Queues (CQ)

- Every WQE (Work Queue Entry) posted to a QP generates a CQE (Completion Queue Entry) when done
- One CQ can be shared across multiple QPs
- CQ can trigger event notifications (interrupt-driven) or be polled (lower latency)

```bash
# Show completion queue resources
rdma res show cq dev mlx5_0

# Monitor CQ events
rdma res show cq dev mlx5_0 -j
```

### Memory Regions (MR)

- Application memory must be **registered** with the NIC before RDMA operations
- Registration pins pages in physical memory and creates a mapping in the NIC's translation table
- Each MR has an lkey (local access) and rkey (remote access for RDMA read/write)
- Registration is expensive — register once, reuse many times

```bash
# Show memory region usage
rdma res show mr dev mlx5_0

# Check memory registration statistics
cat /sys/class/infiniband/mlx5_0/hw_counters/np_cnp_sent
```

### Protection Domains (PD)

- PDs isolate resources — QPs, MRs, and address handles must belong to the same PD
- Prevents unauthorized access between different applications or connections
- Analogous to process isolation in the OS

### RDMA Verbs (Operations)

| Verb | Direction | Description | Remote MR Needed? |
|:---|:---|:---|:---:|
| Send | Initiator -> Target | Push data to remote receive buffer | No |
| Receive | Target <- Initiator | Pre-posted buffer for incoming send | No |
| RDMA Write | Initiator -> Target | Write directly to remote memory | Yes (rkey) |
| RDMA Read | Initiator <- Target | Read directly from remote memory | Yes (rkey) |
| Atomic CAS | Bidirectional | Compare-and-swap on remote memory | Yes (rkey) |
| Atomic FAA | Bidirectional | Fetch-and-add on remote memory | Yes (rkey) |

## Lossless Ethernet Requirements

### Why Lossless?

RoCE v2 uses UDP — no retransmission at the transport layer. Any packet loss causes the RDMA transport (RC) to trigger a Go-Back-N retransmission at the verbs layer, destroying performance. The network **must** be lossless.

### Priority Flow Control (PFC) — IEEE 802.1Qbb

- Per-priority PAUSE frames that stop traffic on a specific CoS/DSCP class without affecting other classes
- RoCE traffic is typically assigned to priority 3 or 4 (configurable)
- PFC prevents buffer overflow at each hop by signaling upstream to pause

```bash
# Enable PFC on priority 3 for RoCE (Mellanox/NVIDIA)
mlnx_qos -i eth0 --pfc 0,0,0,1,0,0,0,0

# Verify PFC configuration
mlnx_qos -i eth0

# Check PFC counters
ethtool -S eth0 | grep pfc
ethtool -S eth0 | grep pause

# On a switch (Cumulus/SONiC) — enable PFC on priority 3
# /etc/cumulus/datapath/traffic.conf:
# pfc.port.class_enable = 3
```

### ECN (Explicit Congestion Notification) — IEEE 802.1Qau

- Switches mark packets with ECN CE (Congestion Experienced) bits instead of dropping them
- The receiver generates a CNP (Congestion Notification Packet) back to the sender
- Sender reduces transmission rate based on CNP feedback

```bash
# Enable ECN on the NIC for RoCE traffic
sysctl -w net.ipv4.tcp_ecn=1

# Configure ECN marking threshold on switch (memory threshold in bytes)
# SONiC: /etc/sonic/config_db.json
# "WRED_PROFILE": {
#   "AZURE_LOSSLESS": {
#     "ecn": "ecn_all",
#     "green_min_threshold": "200000",
#     "green_max_threshold": "2000000"
#   }
# }
```

### DCQCN (Data Center QCN) — Congestion Control

- End-to-end congestion control algorithm designed specifically for RoCE v2
- Combines ECN feedback from the network with rate-based sender-side throttling
- Three components: (1) Switch marks ECN, (2) Receiver sends CNP, (3) Sender adjusts rate
- Sender uses additive-increase / multiplicative-decrease (AIMD) with timer-based recovery
- Alternative: HPCC (High Precision Congestion Control) uses INT (In-band Network Telemetry)

```bash
# DCQCN parameters on Mellanox ConnectX (via mlxreg)
# Congestion control mode
mlxreg -d /dev/mst/mt4123_pciconf0 --set "cc_algo=1"  # 1 = DCQCN

# Set initial rate (in Kbps)
mlxreg -d /dev/mst/mt4123_pciconf0 --set "init_rate=100000000"

# Configure rate reduction factor (alpha)
mlxreg -d /dev/mst/mt4123_pciconf0 --set "alpha=1024"
```

## NIC Configuration (Mellanox/NVIDIA ConnectX)

### Device Discovery

```bash
# List RDMA devices
ibv_devices
rdma link show

# Show device capabilities
ibv_devinfo -d mlx5_0

# Check firmware version
ibstat mlx5_0
ethtool -i eth0

# Show RDMA device details
rdma dev show mlx5_0
```

### RoCE Mode Configuration

```bash
# Set RoCE mode to v2 (routable)
cma_roce_mode -d mlx5_0 -p 1 2     # 2 = RoCE v2

# Or via sysfs
echo "RoCE v2" > /sys/class/infiniband/mlx5_0/ports/1/gid_attrs/types/0

# Verify GID table (shows RoCE version per entry)
cat /sys/class/infiniband/mlx5_0/ports/1/gids/0
cat /sys/class/infiniband/mlx5_0/ports/1/gid_attrs/types/0

# Show all GIDs
rdma link show mlx5_0/1
for i in $(seq 0 15); do
  echo "GID $i: $(cat /sys/class/infiniband/mlx5_0/ports/1/gids/$i) \
    type=$(cat /sys/class/infiniband/mlx5_0/ports/1/gid_attrs/types/$i)"
done
```

### DSCP/ToS Configuration

```bash
# Set DSCP value for RoCE traffic (26 = AF31, common for RoCE)
echo 104 > /sys/class/infiniband/mlx5_0/tc/1/traffic_class
# 104 = DSCP 26 << 2

# Configure trust mode to DSCP (instead of PCP/802.1p)
mlnx_qos -i eth0 --trust dscp

# Map DSCP to priority
mlnx_qos -i eth0 --dscp2prio set,26,3
```

### Firmware and Driver

```bash
# Update Mellanox firmware
mstflint -d /dev/mst/mt4123_pciconf0 -i fw-ConnectX6-rel.bin burn

# Check driver version
modinfo mlx5_core | grep version

# Reload driver
modprobe -r mlx5_ib mlx5_core && modprobe mlx5_core
```

## Linux RDMA Stack

### Core Packages

```bash
# Install rdma-core (provides libibverbs, librdmacm, ibverbs-utils)
# Debian/Ubuntu
apt install rdma-core libibverbs-dev librdmacm-dev ibverbs-utils perftest

# RHEL/CentOS/Fedora
dnf install rdma-core libibverbs-devel librdmacm-devel libibverbs-utils perftest

# Verify the stack is loaded
rdma link show
ls /dev/infiniband/
```

### Key Libraries

| Library | Purpose |
|:---|:---|
| libibverbs | Core RDMA verbs API (QP, CQ, MR management) |
| librdmacm | Connection Manager — handles QP setup, address resolution |
| libibumad | InfiniBand user MAD (management datagram) access |
| libmlx5 | Mellanox provider plugin for libibverbs |

### Kernel Modules

```bash
# Essential modules
modprobe ib_core          # Core InfiniBand/RDMA subsystem
modprobe ib_uverbs        # User-space verbs interface (/dev/infiniband/uverbsN)
modprobe rdma_ucm         # User-space connection manager
modprobe mlx5_core        # Mellanox ConnectX driver
modprobe mlx5_ib          # Mellanox RDMA/IB layer

# Verify modules are loaded
lsmod | grep -E "ib_|rdma|mlx5"

# Check uverbs devices
ls -la /dev/infiniband/
```

### rdma-core Configuration

```bash
# /etc/rdma/rdma.conf — load IB modules at boot
# iw_cxgb4 and iw_bnxt_re for iWARP NICs
# mlx5_ib for ConnectX NICs

# /etc/security/limits.conf — allow locked memory for MR registration
# @rdma  soft  memlock  unlimited
# @rdma  hard  memlock  unlimited

# Add user to rdma group
usermod -aG rdma $USER
```

## Performance Testing

### Bandwidth Tests

```bash
# Server (receiver)
ib_write_bw -d mlx5_0 -x 3 --report_gbits

# Client (sender)
ib_write_bw -d mlx5_0 -x 3 --report_gbits <server_ip>

# RDMA read bandwidth
ib_read_bw -d mlx5_0 -x 3 --report_gbits              # server
ib_read_bw -d mlx5_0 -x 3 --report_gbits <server_ip>   # client

# Send bandwidth
ib_send_bw -d mlx5_0 -x 3 --report_gbits               # server
ib_send_bw -d mlx5_0 -x 3 --report_gbits <server_ip>   # client
```

### Latency Tests

```bash
# RDMA write latency
ib_write_lat -d mlx5_0 -x 3                # server
ib_write_lat -d mlx5_0 -x 3 <server_ip>    # client

# RDMA read latency
ib_read_lat -d mlx5_0 -x 3                 # server
ib_read_lat -d mlx5_0 -x 3 <server_ip>     # client

# Send latency
ib_send_lat -d mlx5_0 -x 3                 # server
ib_send_lat -d mlx5_0 -x 3 <server_ip>     # client
```

### Expected Performance (ConnectX-6 Dx, 100 GbE)

| Test | Bandwidth | Latency |
|:---:|:---:|:---:|
| RDMA Write | ~97 Gbps | ~1.3 us |
| RDMA Read | ~97 Gbps | ~1.8 us |
| Send/Recv | ~95 Gbps | ~1.5 us |
| Atomic | N/A | ~2.5 us |

### qperf (Quick Performance)

```bash
# Server
qperf

# Client — test RoCE bandwidth and latency
qperf <server_ip> rc_bw rc_lat

# Detailed output
qperf <server_ip> -v rc_bw rc_lat ud_bw ud_lat
```

## Performance Tuning

### NIC Tuning

```bash
# Enable adaptive interrupt coalescing
ethtool -C eth0 adaptive-rx on adaptive-tx on

# Set ring buffer sizes
ethtool -G eth0 rx 8192 tx 8192

# Enable hardware GRO/LRO
ethtool -K eth0 gro on lro on

# Pin interrupts to specific CPUs (NUMA-aware)
# Show IRQ affinities for mlx5
cat /proc/interrupts | grep mlx5
# Set affinity (example: IRQ 72 to CPU 0)
echo 1 > /proc/irq/72/smp_affinity
```

### System Tuning

```bash
# Increase locked memory limit (required for MR registration)
ulimit -l unlimited

# Or persistently in /etc/security/limits.conf:
# * soft memlock unlimited
# * hard memlock unlimited

# Huge pages for large memory regions
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# NUMA-aware allocation — pin process to NUMA node of the NIC
numactl --cpunodebind=0 --membind=0 ./my_rdma_app

# Check NIC NUMA node
cat /sys/class/net/eth0/device/numa_node

# Disable irqbalance for latency-sensitive workloads
systemctl stop irqbalance
```

### Switch Tuning

```bash
# Lossless buffer allocation (SONiC example)
# Allocate dedicated buffer pool for priority 3
# /etc/sonic/config_db.json:
# "BUFFER_POOL": {
#   "ingress_lossless_pool": {
#     "size": "12766208",
#     "type": "ingress",
#     "mode": "dynamic"
#   }
# }

# Enable PFC watchdog (detect PFC storms)
# pfcwd start --action drop --detection-time 200 --restoration-time 400 Ethernet0

# Verify no PFC storms
show pfcwd stats
```

## RoCE with VXLAN/Overlay Networks

### Challenges

- VXLAN adds 50 bytes of encapsulation overhead, reducing effective MTU for RDMA
- Inner UDP source port entropy may conflict with RoCE's UDP port 4791
- PFC/ECN must be configured on both underlay and overlay segments
- Some NICs support hardware VXLAN offload with RDMA (ConnectX-6 Dx+)

### Configuration

```bash
# Set MTU to accommodate VXLAN overhead + RoCE headers
ip link set eth0 mtu 9000      # underlay
ip link set vxlan100 mtu 8950  # overlay (9000 - 50 VXLAN overhead)

# RoCE over VXLAN requires NIC support for inner header parsing
# Verify NIC supports VXLAN offload
ethtool -k eth0 | grep vxlan

# NVIDIA ConnectX-6 Dx: enable RoCE over overlay
# mlxconfig -d /dev/mst/mt4123_pciconf0 set ROCE_OVER_OVERLAY=1
```

## Use Cases

### HPC (High-Performance Computing)

- MPI (Message Passing Interface) over RDMA for inter-node communication
- Reduces MPI latency from ~50 us (TCP) to ~1-2 us (RoCE)
- Libraries: OpenMPI, MPICH, Intel MPI all support RoCE natively

### AI/ML Training

- GPU-to-GPU communication via GPUDirect RDMA (bypasses CPU entirely)
- NCCL (NVIDIA Collective Communications Library) uses RoCE for AllReduce, AllGather
- Critical for distributed training with large models (LLMs, vision transformers)

```bash
# Enable GPUDirect RDMA
modprobe nv_peer_mem

# Verify GPUDirect
cat /sys/kernel/mm/memory_peers/nv_mem/version

# NCCL environment variables for RoCE
export NCCL_IB_DISABLE=0
export NCCL_NET_GDR_LEVEL=5
export NCCL_IB_GID_INDEX=3
```

### Storage — NVMe over Fabrics (NVMe-oF)

- NVMe commands sent over RoCE to remote NVMe SSDs
- Sub-10-microsecond remote storage access — near-local performance
- Linux kernel has native nvme-rdma transport

```bash
# Discover NVMe-oF targets over RoCE
nvme discover -t rdma -a 192.168.1.100 -s 4420

# Connect to remote NVMe namespace
nvme connect -t rdma -n nqn.2024-01.com.example:nvme -a 192.168.1.100 -s 4420

# Show connected NVMe-oF controllers
nvme list-subsys

# Verify RDMA transport
cat /sys/class/nvme/nvme1/transport
```

## Show and Inspection Commands

```bash
# List all RDMA devices and ports
rdma link show
ibv_devices
ibv_devinfo

# Show device statistics
rdma statistic show dev mlx5_0

# Show resource utilization (QPs, CQs, MRs)
rdma res show dev mlx5_0
rdma res show qp dev mlx5_0
rdma res show cq dev mlx5_0
rdma res show mr dev mlx5_0

# Show GID table (RoCE addressing)
rdma link show mlx5_0/1

# Network interface statistics relevant to RoCE
ethtool -S eth0 | grep -E "rx_vport_rdma|tx_vport_rdma"
ethtool -S eth0 | grep -E "pfc|pause|ecn|cnp"

# Check for RoCE errors
ethtool -S eth0 | grep -E "out_of_sequence|packet_seq_err|implied_nak"
```

## Troubleshooting

### Connectivity Issues

```bash
# Verify RDMA device is up and has a valid GID
rdma link show
ibv_devinfo -d mlx5_0

# Test basic RDMA connectivity
rping -s -v                     # server
rping -c -a <server_ip> -v      # client

# Check routing for RoCE v2 (UDP 4791)
ip route get <remote_ip>
```

### Performance Issues

```bash
# Check PFC counters — high pause counts indicate congestion
ethtool -S eth0 | grep pfc

# Check for CNP (Congestion Notification Packets)
ethtool -S eth0 | grep cnp

# Verify NUMA alignment
cat /sys/class/net/eth0/device/numa_node
numactl --hardware

# Check for retransmissions (should be 0 on a healthy lossless fabric)
ethtool -S eth0 | grep -E "retry|rnr|timeout"
```

### PFC Storm Detection

```bash
# A PFC storm occurs when a device continuously sends PFC pause frames
# This pauses traffic on an entire priority class across the fabric

# Monitor PFC pause frame counts
watch -n 1 'ethtool -S eth0 | grep pfc'

# Enable PFC watchdog on SONiC switches
# pfcwd start --action drop Ethernet0-48

# Drop counters if watchdog is active
show pfcwd stats
```

## Tips

- Always use RoCE v2 in new deployments — RoCE v1 is L2-only and cannot be routed across subnets.
- Configure PFC, ECN, and DCQCN together as a system — PFC alone causes head-of-line blocking and potential deadlocks without congestion control.
- Pin RDMA applications to the NUMA node of the NIC; cross-NUMA memory access adds 50-100 ns of latency per operation.
- Register memory regions once and reuse them; MR registration involves kernel calls, page pinning, and NIC MMU programming — it is the most expensive RDMA setup operation.
- Use jumbo frames (MTU 9000) on the entire fabric; RoCE performance degrades significantly with 1500-byte MTUs due to per-packet NIC processing overhead.
- For AI/ML workloads, enable GPUDirect RDMA and set NCCL environment variables to prefer IB/RoCE over TCP sockets.
- Monitor `ethtool -S` counters for `cnp_sent`, `cnp_received`, `pfc_pause`, and `out_of_sequence` — these are the four key health indicators.
- Set `memlock` to unlimited for any process using RDMA; the default 64 KB limit will cause MR registration failures.
- Use `perftest` tools (ib_write_bw, ib_read_lat) to baseline performance before deploying applications — they isolate fabric issues from application issues.
- When running RoCE over VXLAN, ensure the NIC supports overlay offload and that both underlay and overlay MTUs account for double encapsulation overhead.

## See Also

- ethernet, vxlan, tcp, udp, ecmp, bgp

## References

- [RoCE v2 Specification — InfiniBand Trade Association](https://www.infinibandta.org/ibta-specification/)
- [RFC 7306 — Remote Direct Memory Access (RDMA) Protocol Extensions](https://www.rfc-editor.org/rfc/rfc7306)
- [IEEE 802.1Qbb — Priority-based Flow Control](https://standards.ieee.org/ieee/802.1Qbb/4677/)
- [IEEE 802.1Qau — Congestion Notification](https://standards.ieee.org/ieee/802.1Qau/4048/)
- [NVIDIA MLNX_OFED Documentation](https://docs.nvidia.com/networking/display/mlnxofedv24010331/)
- [rdma-core — Linux RDMA User-Space Libraries](https://github.com/linux-rdma/rdma-core)
- [Linux Kernel RDMA Subsystem](https://www.kernel.org/doc/html/latest/infiniband/index.html)
- [NVMe over Fabrics Specification](https://nvmexpress.org/specifications/)
- [DCQCN: Data Center Quantized Congestion Notification (SIGCOMM 2015)](https://dl.acm.org/doi/10.1145/2785956.2787484)
- [NVIDIA GPUDirect RDMA Documentation](https://docs.nvidia.com/cuda/gpudirect-rdma/)
- [perftest — RDMA Performance Tests](https://github.com/linux-rdma/perftest)
