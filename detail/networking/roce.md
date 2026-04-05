# RoCE — Remote Direct Memory Access over Converged Ethernet

> *RoCE eliminates the CPU from the data path by letting network adapters read and write application memory directly. The protocol layers InfiniBand transport semantics over standard Ethernet, trading the simplicity of "fire and forget" UDP for a zero-copy, kernel-bypass architecture that demands a lossless fabric — achieved through PFC, ECN, and end-to-end congestion control algorithms like DCQCN.*

---

## 1. RDMA Fundamentals — Why the CPU Is the Bottleneck

### The Traditional Network Stack Problem

In a conventional TCP/IP stack, sending 1 byte of application data triggers a cascade of CPU work:

1. **System call** — `send()` transitions from user space to kernel space (context switch: ~1-2 us)
2. **Socket buffer copy** — data is copied from user buffer to kernel socket buffer (memcpy)
3. **TCP processing** — segmentation, checksum, sequence numbers, window management
4. **IP processing** — routing lookup, header construction, fragmentation check
5. **Driver** — DMA descriptor setup, doorbell ring
6. **Receive side** — reverse of the above, plus interrupt handling and another memcpy to user space

**Total copies per message: 2-4.** Total context switches: 2-4. For a 64-byte message, the CPU spends more cycles on protocol processing than the NIC spends on the wire.

### The RDMA Model

RDMA collapses this entire stack:

| Step | Traditional | RDMA |
|:---|:---|:---|
| Data copies | 2-4 | 0 (zero-copy) |
| Context switches | 2-4 | 0 (kernel bypass) |
| Protocol processing | CPU | NIC hardware |
| Latency | 10-50 us | 1-2 us |
| CPU cycles per msg | ~10,000 | ~100 (doorbell only) |

### How Zero-Copy Works

1. Application registers a memory region with the NIC (one-time setup)
2. NIC's internal MMU maps virtual addresses to physical pages
3. For RDMA Write: sender posts a WQE with {remote_addr, rkey, local_addr, lkey, length}
4. Sender NIC DMAs data from local memory, constructs packets, transmits on wire
5. Receiver NIC DMAs data directly into the target application's memory buffer
6. No CPU involvement on either side after the initial WQE post

The receiver's CPU never touches the data. It learns the operation completed by polling or receiving a CQE (Completion Queue Entry).

---

## 2. The RDMA Programming Model — Verbs, Queues, and Memory

### The Verbs API

RDMA operations are called **verbs** — a term inherited from the InfiniBand specification. The libibverbs library exposes them to user space.

#### Two-Sided Operations (Channel Semantics)

**Send/Receive** works like a traditional message-passing model:

1. Receiver pre-posts a Receive WQE: "I have a buffer at address X, length Y, ready to accept data"
2. Sender posts a Send WQE: "Transmit data from my buffer at address A, length B"
3. Sender NIC transmits; receiver NIC DMAs into the pre-posted buffer
4. Both sides get a CQE when done

**Key constraint:** the receiver must post a receive buffer **before** the sender sends. If no receive buffer is posted, the NIC generates an RNR (Receiver Not Ready) NAK.

#### One-Sided Operations (Memory Semantics)

**RDMA Read/Write** operate on remote memory without any involvement from the remote CPU:

- **RDMA Write:** Sender specifies {remote_virtual_addr, rkey} and the NIC writes directly into remote memory. The remote CPU is not notified (unless the sender also sets an immediate data flag, which generates a CQE on the receiver).
- **RDMA Read:** Sender specifies {remote_virtual_addr, rkey} and the remote NIC DMAs the data back. The remote application does not know its memory was read.

This is the key innovation: the remote CPU does zero work for one-sided operations.

#### Atomic Operations

- **Compare-and-Swap (CAS):** Atomically compares a 64-bit value at a remote address with an expected value; if equal, replaces it with a new value. Returns the original value.
- **Fetch-and-Add (FAA):** Atomically adds a 64-bit value to a remote address. Returns the original value.

Atomics enable distributed locks, counters, and coordination without message passing.

### Queue Pair Architecture

```
Application
    |
    v
+-------------------+
|   Send Queue (SQ) | ----> Posts Send, RDMA Write, RDMA Read, Atomic WQEs
+-------------------+
|   Recv Queue (RQ) | ----> Posts Receive WQEs (buffers for incoming Sends)
+-------------------+
    |
    v
+-----------------------+
|  Completion Queue (CQ)| ----> CQEs report completion/error for each WQE
+-----------------------+
```

#### QP Types

| QP Type | Abbreviation | Reliable? | Connected? | Use Case |
|:---|:---:|:---:|:---:|:---|
| Reliable Connected | RC | Yes | Yes | Most RDMA apps, NVMe-oF, NCCL |
| Unreliable Connected | UC | No | Yes | Bulk transfer where loss is tolerable |
| Unreliable Datagram | UD | No | No | Service discovery, multicast |
| Extended Reliable Connected | XRC | Yes | Yes | Reduces QP count in multi-threaded apps |

**RC** dominates real-world usage. It provides:
- In-order delivery
- Retransmission on loss (Go-Back-N)
- Flow control via credits
- Support for all verb types including atomics

#### QP State Machine

Every QP transitions through a defined state machine:

```
RESET --> INIT --> RTR (Ready to Receive) --> RTS (Ready to Send) --> ERROR
                                                      |
                                                      v
                                                    SQD (Send Queue Drained)
```

| Transition | What Happens |
|:---|:---|
| RESET -> INIT | Assign PD, port, access flags |
| INIT -> RTR | Set remote QPN, GID, path MTU, PSN |
| RTR -> RTS | Set timeout, retry count, RNR retry, max outstanding |
| Any -> ERROR | Fatal error, QP must be destroyed and recreated |

The INIT->RTR->RTS transitions are where connection parameters (remote address, QP number, keys) are exchanged — typically via an out-of-band TCP connection managed by librdmacm.

### Memory Registration Deep Dive

Memory registration is the most misunderstood part of RDMA. Here is what happens internally:

1. **ibv_reg_mr()** is called with a virtual address range and access flags
2. Kernel pins the physical pages (prevents swapping) via `get_user_pages()`
3. Kernel programs the NIC's internal IOMMU/MTT (Memory Translation Table)
4. NIC can now translate {lkey/rkey, virtual_addr} -> physical_addr without any CPU involvement
5. Returns an `ibv_mr*` with lkey (for local access) and rkey (for remote access)

**Cost:** Registration of 1 GB takes ~1-5 ms (page pinning + MTT programming). This is why you register once and reuse.

**Access flags:**

| Flag | Meaning |
|:---|:---|
| IBV_ACCESS_LOCAL_WRITE | NIC can write to this region (required for receives) |
| IBV_ACCESS_REMOTE_WRITE | Remote nodes can RDMA Write to this region |
| IBV_ACCESS_REMOTE_READ | Remote nodes can RDMA Read from this region |
| IBV_ACCESS_REMOTE_ATOMIC | Remote nodes can perform atomics on this region |

**Security model:** rkey is a 32-bit value. Knowing the rkey and virtual address of a remote MR grants full access (within the specified flags). Applications must protect rkey distribution — it is essentially a capability token.

### Protection Domains

A Protection Domain groups related resources:

```
+--- Protection Domain (PD) ---+
|                               |
|   QP1   QP2   QP3            |   QPs in this PD can only access
|   MR1   MR2   MR3            |   MRs in the same PD
|   AH1   AH2                  |   Address Handles for UD QPs
|                               |
+-------------------------------+
```

Cross-PD access is impossible at the hardware level. This provides isolation between:
- Different applications on the same host
- Different connections within the same application
- Different tenants in a multi-tenant environment

---

## 3. RoCE v1 vs RoCE v2 — Layer 2 vs Layer 3

### RoCE v1 Encapsulation

```
+----------+----------+----------+----------+
| Ethernet | IB GRH   | IB BTH   | Payload  |
| 14 B     | 40 B     | 12 B     |          |
+----------+----------+----------+----------+
EtherType: 0x8915

GRH = Global Route Header (InfiniBand)
BTH = Base Transport Header (InfiniBand)
```

RoCE v1 places InfiniBand headers directly inside an Ethernet frame. This means:
- **No IP header** — not routable beyond the L2 broadcast domain
- **No UDP header** — no ECMP hashing on source port
- Switches must understand EtherType 0x8915 or treat it as unknown protocol
- Functionally limited to a single VLAN or flat L2 segment

### RoCE v2 Encapsulation

```
+----------+----------+----------+----------+----------+
| Ethernet | IP       | UDP      | IB BTH   | Payload  |
| 14 B     | 20/40 B  | 8 B      | 12 B     |          |
+----------+----------+----------+----------+----------+
UDP Dst Port: 4791
UDP Src Port: hash(flow) — entropy for ECMP
```

RoCE v2 wraps the InfiniBand BTH inside a standard UDP/IP packet:
- **Routable** across subnets, through standard IP routers
- **ECMP-friendly** — UDP source port provides flow entropy
- **ECN-capable** — IP header carries ECN bits for congestion signaling
- **DSCP-capable** — IP header carries DSCP for QoS classification
- Switches treat it as normal UDP/IP traffic (no special protocol handling)

### Migration Considerations

| Aspect | RoCE v1 | RoCE v2 |
|:---|:---|:---|
| Network design | Flat L2, single VLAN | L3 leaf-spine fabric |
| Switch requirements | EtherType 0x8915 support | Standard L3 switch |
| ECMP | Not possible | Full support |
| Multi-site | Not possible | Routable (with tuning) |
| GID format | MAC-based | IP-based |
| PFC scope | L2 domain | Per-hop, end-to-end |

---

## 4. The Lossless Ethernet Problem

### Why RoCE Cannot Tolerate Loss

RDMA RC transport uses **Go-Back-N** retransmission. When a packet is lost:

1. Receiver detects a sequence gap
2. Receiver sends a NAK (negative acknowledgment)
3. Sender retransmits from the lost packet onward (not just the lost packet)
4. Retransmission timeout is typically 8-33 ms (NIC firmware timer)
5. During retransmission, the QP is stalled — no new operations complete

**Impact of a single lost packet:**

$$T_{penalty} = T_{detection} + T_{retransmit} + T_{reorder}$$

Typical values: $8\text{ ms} + 1\text{ ms} + 0.5\text{ ms} = 9.5\text{ ms}$

Compare with normal RDMA latency of 1-2 us. A single drop causes a **5,000x latency spike**. At scale, even 0.001% packet loss destroys tail latency.

### PFC — Priority-based Flow Control (IEEE 802.1Qbb)

PFC is a hop-by-hop flow control mechanism that operates per-priority:

```
                    PFC PAUSE (priority 3)
Switch A  <---------------------------  Switch B
   |                                       |
   | (stops sending priority 3 traffic)    | (buffer filling up)
   |                                       |
   +--- continues sending priority 0-2, 4-7 ---+
```

#### How PFC Works

1. Switch B's ingress buffer for priority 3 exceeds the XOFF threshold
2. Switch B sends a PFC PAUSE frame to Switch A for priority 3 only
3. Switch A stops transmitting priority 3 traffic (buffers it locally)
4. Switch B drains its buffer below the XON threshold
5. Switch B sends a PFC resume (or the pause timer expires)
6. Switch A resumes transmitting priority 3

#### PFC Thresholds

$$X_{OFF} = B_{total} - (R_{line} \times T_{propagation} \times N_{hops})$$

Where:
- $B_{total}$ = total buffer allocated to this priority
- $R_{line}$ = line rate in bytes/sec
- $T_{propagation}$ = round-trip propagation delay (cable + switch pipeline)
- $N_{hops}$ = number of hops for the PFC PAUSE to take effect

For 100 GbE with 300 ns cable delay and 1 us switch pipeline:

$$X_{OFF} = B_{total} - (12.5 \text{ GB/s} \times 2.6 \text{ us}) = B_{total} - 32,500 \text{ bytes}$$

The buffer must be large enough to absorb in-flight data while the PAUSE propagates.

#### PFC Deadlocks

PFC can cause **circular buffer dependency deadlocks**:

```
Switch A pauses Switch B (buffer full)
Switch B pauses Switch C (buffer full)
Switch C pauses Switch A (buffer full)
--> Deadlock: no switch can drain, traffic stops permanently
```

**Solutions:**
- **DSCP-based priority separation** — use different priorities for different traffic classes
- **Lossless traffic isolation** — only RoCE traffic uses the PFC-enabled priority
- **PFC watchdog** — detect and break deadlocks by dropping traffic after a timeout

### ECN — Explicit Congestion Notification (IEEE 802.1Qau)

ECN provides congestion feedback without dropping packets:

1. Sender marks packets as ECN-capable: IP header ECN field = `10` (ECT(0)) or `01` (ECT(1))
2. Switch detects congestion (queue depth exceeds threshold)
3. Switch marks the packet: ECN field = `11` (CE — Congestion Experienced)
4. Receiver detects CE marking and generates a CNP (Congestion Notification Packet) back to sender
5. Sender reduces its transmission rate

#### ECN Marking Thresholds (WRED)

Switches use Weighted Random Early Detection to decide when to mark:

| Queue Depth | Action |
|:---|:---|
| Below $K_{min}$ | No marking |
| Between $K_{min}$ and $K_{max}$ | Probabilistic marking (linear ramp) |
| Above $K_{max}$ | Mark all packets (100%) |

$$P_{mark} = \begin{cases} 0 & Q < K_{min} \\ \frac{Q - K_{min}}{K_{max} - K_{min}} \times P_{max} & K_{min} \leq Q \leq K_{max} \\ 1 & Q > K_{max} \end{cases}$$

### DCQCN — Data Center QCN (Congestion Control for RoCE)

DCQCN is the standard congestion control algorithm for RoCE v2, described in the SIGCOMM 2015 paper. It has three actors:

1. **Reaction Point (RP)** — the sender NIC, adjusts transmission rate
2. **Congestion Point (CP)** — the switch, marks packets with ECN
3. **Notification Point (NP)** — the receiver NIC, generates CNPs

#### Sender-Side Rate Control

The sender maintains a **current rate** $R_C$ and a **target rate** $R_T$:

**On receiving a CNP (rate decrease):**

$$R_T = R_C$$
$$R_C = R_C \times (1 - \alpha / 2)$$

Where $\alpha$ is the congestion severity factor, updated as:

$$\alpha = (1 - g) \times \alpha + g \times F$$

- $g$ = weight factor (typically 1/256)
- $F$ = 1 if CNP received in this interval, 0 otherwise

**On timer expiration (rate increase):**

$$R_T = R_T + R_{AI}$$
$$R_C = (R_C + R_T) / 2$$

Where $R_{AI}$ is the additive increase rate (typically 5-40 Mbps).

This produces the classic AIMD (Additive Increase, Multiplicative Decrease) behavior:
- **Fast decrease** when congestion is detected (multiplicative)
- **Slow increase** when congestion clears (additive)

#### DCQCN vs TCP Congestion Control

| Property | DCQCN | TCP CUBIC |
|:---|:---|:---|
| Signal | ECN (no loss) | Packet loss or ECN |
| Granularity | Per-QP rate | Per-flow window |
| Decrease factor | $\alpha/2$ (adaptive) | 0.3 (fixed) |
| Increase | Additive + averaging | Cubic function |
| Convergence | ~10-50 RTTs | ~100-1000 RTTs |
| Fairness | Good (per-flow) | Good (per-flow) |

---

## 5. iWARP — The TCP-Based Alternative

### Architecture

iWARP (Internet Wide Area RDMA Protocol) layers RDMA over TCP:

```
+----------+----------+----------+----------+----------+
| Ethernet | IP       | TCP      | MPA/DDP  | Payload  |
| 14 B     | 20 B     | 20 B     | 14 B     |          |
+----------+----------+----------+----------+----------+

MPA = Marker PDU Aligned framing
DDP = Direct Data Placement
```

### Why iWARP Exists

TCP handles loss recovery natively — no need for PFC or ECN. This makes iWARP deployable on any Ethernet network without lossless configuration.

### The Performance Trade-off

| Metric | RoCE v2 | iWARP |
|:---|:---|:---|
| Latency (64 B) | 1.3 us | 5-8 us |
| Bandwidth (100G) | 97 Gbps | 85-90 Gbps |
| CPU usage | Near zero | Higher (TCP offload) |
| Network config | Complex (PFC/ECN) | Simple (standard TCP) |
| Loss handling | Go-Back-N (catastrophic) | TCP retransmit (graceful) |
| Vendor support | Broad (NVIDIA, Broadcom) | Limited (Chelsio, Intel) |

### When to Choose iWARP

- Network cannot be configured lossless (shared fabric, WAN, legacy switches)
- Operational simplicity is more important than peak performance
- Small-scale deployments where 5 us vs 1.5 us latency does not matter

---

## 6. RoCE in Practice — Deployment Patterns

### Leaf-Spine Fabric for RoCE

```
       +--------+   +--------+   +--------+
       | Spine1 |   | Spine2 |   | Spine3 |
       +--------+   +--------+   +--------+
        /  |  \      /  |  \      /  |  \
       /   |   \    /   |   \    /   |   \
+------+ +------+ +------+ +------+ +------+
|Leaf1 | |Leaf2 | |Leaf3 | |Leaf4 | |Leaf5 |
+------+ +------+ +------+ +------+ +------+
  |  |     |  |     |  |     |  |     |  |
 GPU GPU  GPU GPU  GPU GPU  GPU GPU  GPU GPU
```

**Design rules:**
- PFC enabled on every link (leaf-to-host and leaf-to-spine)
- ECN enabled on all switches with consistent WRED thresholds
- DCQCN enabled on all NICs
- Single lossless priority (e.g., priority 3, DSCP 26/AF31)
- Jumbo frames end-to-end (MTU 9000 or 9216)
- ECMP across all spine links for load distribution

### QP Scaling

Each RDMA connection requires a QP on both endpoints. In a cluster with $N$ nodes where each node connects to $K$ other nodes:

$$\text{QPs per node} = K \times Q_{per\_connection}$$

Where $Q_{per\_connection}$ = number of QPs per connection (often 1 per CPU core or GPU).

| Cluster Size | Connections/Node | QPs/Connection | Total QPs/Node |
|:---:|:---:|:---:|:---:|
| 8 (small) | 7 | 8 | 56 |
| 64 (medium) | 63 | 8 | 504 |
| 512 (large) | 511 | 8 | 4,088 |
| 4,096 (hyperscale) | 4,095 | 8 | 32,760 |

Modern ConnectX NICs support millions of QPs, but each QP consumes NIC memory for its context. At hyperscale, **Shared Receive Queues (SRQ)** reduce memory usage by sharing receive buffers across many QPs.

### Multi-Path RoCE

For high availability and bandwidth aggregation:

| Approach | Mechanism | Complexity |
|:---|:---|:---:|
| LAG/Bond | NIC bonding (active-backup or LACP) | Low |
| Multi-port | Separate RDMA devices per port | Medium |
| Adaptive routing | Switch-level path selection | High |
| Multi-path RDMA | Application-level path management | High |

### RoCE and GPUDirect

GPUDirect RDMA enables a direct data path between the NIC and GPU memory, bypassing system memory entirely:

```
Traditional path:
  Remote NIC --> Local NIC --> System RAM --> GPU Memory
  (2 DMA transfers + CPU memcpy)

GPUDirect RDMA path:
  Remote NIC --> Local NIC --> GPU Memory
  (1 DMA transfer, zero CPU involvement)
```

**Requirements:**
- NIC and GPU on the same PCIe root complex (ideally same CPU socket)
- NVIDIA GPU with CUDA toolkit
- `nv_peer_mem` or `nvidia_peermem` kernel module
- NIC firmware with GPUDirect support (ConnectX-5+)

**Performance impact:**
- Eliminates one DMA transfer and one memcpy
- Reduces GPU-to-GPU latency by 30-50%
- Critical for NCCL AllReduce in distributed training

---

## 7. NVMe over Fabrics with RoCE

### Architecture

NVMe-oF RDMA transport maps NVMe commands to RDMA operations:

| NVMe Operation | RDMA Mapping |
|:---|:---|
| Command submission | RDMA Send (command capsule) |
| Command response | RDMA Send (response capsule) |
| Data transfer (read) | RDMA Read (target reads from initiator MR) |
| Data transfer (write) | RDMA Write (target writes to initiator MR) |

### Performance Characteristics

| Metric | Local NVMe | NVMe-oF (TCP) | NVMe-oF (RDMA/RoCE) |
|:---|:---:|:---:|:---:|
| Read latency (4K) | 10 us | 50-80 us | 12-15 us |
| Write latency (4K) | 8 us | 40-60 us | 10-12 us |
| IOPS (4K rand read) | 1,000K | 400K | 800K |
| Bandwidth (seq read) | 7 GB/s | 5 GB/s | 6.5 GB/s |
| CPU overhead | Baseline | High | Low |

NVMe-oF over RoCE achieves **near-local** storage performance — the network adds only 2-5 us of latency.

### Scaling Considerations

Each NVMe-oF connection uses RDMA QPs. A storage cluster with $S$ storage nodes, $C$ compute nodes, and $N$ namespaces per storage node:

$$\text{QPs per compute node} = S \times N \times Q_{per\_ns}$$

With 16 storage nodes, 4 namespaces each, and 2 QPs per namespace: $16 \times 4 \times 2 = 128$ QPs per compute node.

---

## 8. Monitoring and Observability

### Key Counters

| Counter | Meaning | Healthy Value |
|:---|:---|:---|
| `rx_vport_rdma_unicast_packets` | RDMA packets received | Increasing steadily |
| `tx_vport_rdma_unicast_packets` | RDMA packets sent | Increasing steadily |
| `np_cnp_sent` | CNPs sent (as notification point) | Low, stable |
| `rp_cnp_handled` | CNPs handled (as reaction point) | Low, stable |
| `out_of_sequence` | Packets received out of order | 0 |
| `packet_seq_err` | Sequence errors (lost packets) | 0 |
| `implied_nak_seq_err` | Implied NAKs (retransmissions) | 0 |
| `local_ack_timeout_err` | QP timeouts | 0 |
| `rnr_nak_retry_err` | Receiver not ready errors | 0 |
| `rx_pfc_pause_duration` | Time paused by PFC (us) | Low |

### Health Check Script

```bash
#!/bin/bash
DEV=${1:-mlx5_0}
ETH=$(rdma link show $DEV/1 | awk -F'netdev ' '{print $2}' | awk '{print $1}')

echo "=== RDMA Device: $DEV (netdev: $ETH) ==="

# Check for errors (all should be 0)
echo "--- Error Counters ---"
for counter in out_of_sequence packet_seq_err implied_nak_seq_err \
               local_ack_timeout_err rnr_nak_retry_err; do
    val=$(cat /sys/class/infiniband/$DEV/hw_counters/$counter 2>/dev/null || echo "N/A")
    echo "  $counter: $val"
done

# Check congestion signals
echo "--- Congestion Counters ---"
for counter in np_cnp_sent rp_cnp_handled; do
    val=$(cat /sys/class/infiniband/$DEV/hw_counters/$counter 2>/dev/null || echo "N/A")
    echo "  $counter: $val"
done

# Check PFC
echo "--- PFC Counters ---"
ethtool -S $ETH 2>/dev/null | grep -E "pfc|pause"
```

---

## 9. Common Failure Modes

| Failure | Symptom | Root Cause | Fix |
|:---|:---|:---|:---|
| MR registration fails | `errno 12 (ENOMEM)` | memlock ulimit too low | Set `ulimit -l unlimited` |
| QP moves to ERROR state | Connection drops | Retransmission limit exceeded | Fix PFC/ECN, check cables |
| RNR NAK errors | Slow throughput | Receiver not posting buffers fast enough | Increase RQ depth, tune app |
| PFC storm | All traffic on a priority halts | Faulty NIC or misconfigured PFC | Enable PFC watchdog |
| High CNP rate | Throughput oscillation | Aggressive ECN thresholds | Tune WRED $K_{min}$/$K_{max}$ |
| DCQCN too aggressive | Low throughput | Alpha converges too fast | Reduce $g$ weight factor |
| DCQCN too slow | Buffer overflow | Alpha converges too slowly | Increase $g$, lower $K_{min}$ |
| Cross-NUMA latency | 50-100 ns penalty per op | NIC and app on different NUMA nodes | Pin to NIC's NUMA node |
| GID table empty | Cannot connect | Missing IP on RDMA interface | Assign IP, check `rdma link` |

---

## 10. The Future — Beyond Traditional RoCE

### Emerging Technologies

- **Adaptive Routing:** Switches dynamically reroute packets around congested links (replaces static ECMP)
- **In-Network Computing:** SmartNICs/DPUs perform RDMA operations + application logic (NVIDIA BlueField)
- **SHARP (Scalable Hierarchical Aggregation and Reduction Protocol):** Offloads collective operations (AllReduce) to the switch network
- **Ultra Ethernet Consortium (UEC):** Industry effort to build a next-generation lossless Ethernet transport that replaces RoCE's InfiniBand heritage with a native Ethernet RDMA protocol — packet spraying, multi-path, and built-in congestion control

### Scale Challenges

At 400 GbE and beyond:
- PFC becomes increasingly problematic (tighter timing, larger buffers needed)
- DCQCN convergence speed must improve (faster links = faster congestion buildup)
- QP scaling becomes a bottleneck (millions of connections in hyperscale AI clusters)
- The industry is moving toward **connectionless RDMA** and **packet spraying** to eliminate QP scaling limits

---

## Prerequisites

- ethernet, tcp, udp, vxlan, ecmp, bgp, subnetting

---

## References

- [InfiniBand Trade Association — RoCE v2 Specification](https://www.infinibandta.org/ibta-specification/)
- [RFC 7306 — Remote Direct Memory Access (RDMA) Protocol Extensions](https://www.rfc-editor.org/rfc/rfc7306)
- [RFC 5040 — Direct Data Placement over Reliable Transports](https://www.rfc-editor.org/rfc/rfc5040)
- [RFC 5041 — Direct Data Placement Protocol (DDP) / Remote Direct Memory Access Protocol (RDMAP) Security](https://www.rfc-editor.org/rfc/rfc5041)
- [IEEE 802.1Qbb — Priority-based Flow Control](https://standards.ieee.org/ieee/802.1Qbb/4677/)
- [IEEE 802.1Qau — Congestion Notification](https://standards.ieee.org/ieee/802.1Qau/4048/)
- [Zhu et al. — "Congestion Control for Large-Scale RDMA Deployments" (SIGCOMM 2015) — DCQCN Paper](https://dl.acm.org/doi/10.1145/2785956.2787484)
- [NVIDIA MLNX_OFED User Manual](https://docs.nvidia.com/networking/display/mlnxofedv24010331/)
- [rdma-core — GitHub Repository](https://github.com/linux-rdma/rdma-core)
- [Linux Kernel Documentation — InfiniBand/RDMA](https://www.kernel.org/doc/html/latest/infiniband/index.html)
- [NVMe over Fabrics Specification](https://nvmexpress.org/specifications/)
- [NVIDIA GPUDirect RDMA](https://docs.nvidia.com/cuda/gpudirect-rdma/)
- [Ultra Ethernet Consortium](https://ultraethernet.org/)

---

*RoCE turned commodity Ethernet into an RDMA fabric by borrowing InfiniBand's transport semantics and wrapping them in UDP/IP. The price is a lossless network requirement that adds operational complexity — PFC for hop-by-hop flow control, ECN for congestion signaling, DCQCN for end-to-end rate adaptation. When configured correctly, the reward is sub-2-microsecond latency and near-wire-speed throughput with zero CPU overhead — the foundation of modern AI training clusters and disaggregated storage.*
