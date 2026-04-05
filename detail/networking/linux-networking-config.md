# Linux Networking Configuration — Architecture and Theory

> *The Linux networking stack is one of the most sophisticated network implementations in existence. From the interrupt-driven packet path through the kernel's protocol layers to userspace configuration daemons, understanding the architecture enables effective troubleshooting and performance tuning.*

---

## 1. Linux Network Stack Architecture

### Packet Receive Path

```
NIC hardware → IRQ → NAPI softirq
    ↓
netdev_budget loop (up to 300 packets)
    ↓
GRO (Generic Receive Offload) — coalesce segments
    ↓
netfilter PREROUTING
    ↓
Routing decision
    ↓
┌── Local delivery ──→ netfilter INPUT → Socket receive buffer → Application
└── Forward ──→ netfilter FORWARD → netfilter POSTROUTING → Transmit path
```

### Packet Transmit Path

```
Application → Socket send buffer
    ↓
TCP/UDP segmentation
    ↓
Routing lookup (FIB)
    ↓
netfilter OUTPUT → netfilter POSTROUTING
    ↓
Queueing discipline (qdisc)
    ↓
Driver transmit ring → NIC hardware
```

### Key Data Structures

| Structure | Purpose |
|:---|:---|
| `sk_buff` (skb) | The fundamental packet buffer — contains packet data, metadata, and pointers to protocol headers |
| `net_device` | Represents a network interface — holds stats, MTU, MAC, ops function pointers |
| `fib_table` | Forwarding Information Base — routing table in kernel |
| `sock` | Socket structure — binds transport protocol to userspace |
| `nf_hook_ops` | Netfilter hook registrations — where firewall rules attach |

### NAPI (New API) Polling

Traditional interrupt-per-packet is inefficient at high rates. NAPI switches to polling mode under load:

$$\text{if } interrupt\_rate > threshold \implies \text{switch to poll mode}$$

In poll mode, the kernel processes up to `netdev_budget` (default 300) packets per softirq cycle, dramatically reducing interrupt overhead:

$$CPU_{interrupt} \propto \frac{1}{batch\_size}$$

---

## 2. NetworkManager vs systemd-networkd

### Architectural Comparison

| Aspect | NetworkManager | systemd-networkd |
|:---|:---|:---|
| Design goal | Desktop/laptop flexibility | Server/container simplicity |
| Configuration | Connection profiles (keyfiles, ifcfg) | .network, .netdev, .link files |
| IPC | D-Bus | D-Bus (limited) |
| CLI | nmcli, nmtui | networkctl |
| DHCP client | Internal | systemd-networkd built-in |
| DNS | Delegates to systemd-resolved or manages directly | systemd-resolved |
| WiFi | Full support (wpa_supplicant) | Basic (requires wpa_supplicant) |
| VPN | Plugin architecture (OpenVPN, WireGuard, etc.) | WireGuard native only |
| Dispatchers | NetworkManager-dispatcher scripts | networkd-dispatcher |
| State | Tracks connection state, auto-reconnect | Declarative, stateless |

### When to Use Which

**NetworkManager** is the default on RHEL, Fedora, Ubuntu Desktop:
- Laptops/desktops with dynamic connectivity
- VPN, WiFi, mobile broadband requirements
- Complex connection profiles with priorities

**systemd-networkd** excels on:
- Servers with static configuration
- Containers (minimal footprint)
- Embedded systems
- Infrastructure where declarative config is preferred

### Connection Profile Lifecycle (NetworkManager)

```
Profile created (nmcli/keyfile/nmtui)
    ↓
Stored in /etc/NetworkManager/system-connections/
    ↓
NetworkManager loads profile
    ↓
Auto-activation check (autoconnect, priority)
    ↓
Device assignment (interface-name or match)
    ↓
IP configuration (DHCP or static)
    ↓
Dispatcher scripts fire (up/down events)
    ↓
DNS/routing applied
```

---

## 3. Bonding Driver Internals

### Kernel Module Architecture

The bonding driver (`bonding.ko`) creates a virtual interface that multiplexes traffic across multiple slave interfaces:

```
┌────────────────────────────┐
│     bond0 (master device)   │
│  ┌──────────────────────┐  │
│  │  Bonding module logic  │  │
│  │  - Mode handler        │  │
│  │  - Hash function       │  │
│  │  - MII monitor         │  │
│  │  - ARP monitor         │  │
│  └──────────┬─────────────┘  │
│      ┌──────┴──────┐        │
│    eth0          eth1        │
│  (slave 0)     (slave 1)    │
└────────────────────────────┘
```

### Mode Internals

**Mode 0 (balance-rr):** Transmits packets in sequential order through each slave. Simple but can cause out-of-order delivery (problematic for TCP). Requires switch configuration (port-channel/EtherChannel) for proper operation.

**Mode 1 (active-backup):** Only one slave is active at any time. ARP monitoring or MII monitoring detects failures. The `primary` option sets the preferred slave when available. No switch configuration needed.

**Mode 4 (802.3ad/LACP):**

The driver participates in the LACP protocol state machine:

$$\text{Actor} \xrightleftharpoons{\text{LACPDU exchange}} \text{Partner}$$

States: Detached → Waiting → Attached → Collecting → Distributing

LACPDUs are exchanged every 1 second (fast) or 30 seconds (slow).

**Mode 5 (balance-tlb):** Adaptive transmit load balancing. Each slave has a `load` value:

$$load_{slave} = \frac{speed_{slave} - throughput_{slave}}{speed_{slave}}$$

New flows are assigned to the slave with the highest available capacity. No switch configuration needed.

**Mode 6 (balance-alb):** Extends mode 5 with receive load balancing by intercepting ARP replies and rewriting the source MAC to distribute incoming traffic across slaves.

### Link Monitoring

| Monitor | Method | Detect |
|:---|:---|:---|
| MII | Polls carrier state via MII registers | Physical link loss |
| ARP | Sends ARP requests to targets | L2/L3 connectivity |

$$\text{Failover time} \approx miimon \text{ (ms)} \times 2 + updelay$$

Default `miimon=100` means ~200ms failover detection.

---

## 4. LACP Implementation

### Protocol Details

LACP (Link Aggregation Control Protocol, IEEE 802.3ad/802.1AX) negotiates link aggregation between two systems:

| Field | Size | Purpose |
|:---|:---|:---|
| Actor System Priority | 2 bytes | Determines which end controls aggregation |
| Actor System ID | 6 bytes | MAC address of the actor |
| Actor Key | 2 bytes | Groups ports into aggregation groups |
| Actor Port Priority | 2 bytes | Determines port selection within group |
| Actor Port Number | 2 bytes | Unique port identifier |
| Actor State | 1 byte | LACP activity, timeout, aggregation, sync, collecting, distributing |

### Aggregation Selection

Ports are grouped into aggregation groups based on:

$$aggregation\_key = f(speed, duplex, port\_key)$$

Only ports with identical keys can aggregate. The **system with lower priority** (numerically) becomes the decision-maker.

### Hash-Based Distribution

Outbound traffic distribution uses a hash function to select the egress port:

$$slave\_index = hash(fields) \mod n_{slaves}$$

The hash fields depend on `xmit_hash_policy`:

| Policy | Hash Input | Distribution |
|:---|:---|:---|
| layer2 | src_mac XOR dst_mac | Per-MAC-pair |
| layer2+3 | src_mac XOR dst_mac XOR src_ip XOR dst_ip | Per-IP-pair |
| layer3+4 | src_ip XOR dst_ip XOR src_port XOR dst_port | Per-flow |

**layer3+4** provides the finest granularity since each TCP/UDP flow gets an independent hash.

---

## 5. Bridging and STP in Linux

### Bridge Architecture

A Linux bridge operates as a **software switch** in the kernel:

```
┌──────────────────────────────┐
│     Bridge (br0)              │
│  ┌────────────────────────┐  │
│  │  FDB (MAC table)       │  │
│  │  STP state machine     │  │
│  │  VLAN filtering        │  │
│  └────────────────────────┘  │
│      ↕        ↕        ↕     │
│    eth0     eth1     veth0   │
│  (port 1) (port 2) (port 3) │
└──────────────────────────────┘
```

### Forwarding Database (FDB)

The FDB maps MAC addresses to bridge ports:

$$FDB: MAC\_address \to (port, VLAN, age)$$

- **Learning:** When a frame arrives on a port, the source MAC is recorded with that port
- **Flooding:** Unknown destination MACs are flooded to all ports (except the source)
- **Aging:** Entries expire after `ageing_time` (default 300 seconds)

### STP (Spanning Tree Protocol)

Linux bridge supports STP (802.1D) and RSTP (802.1w) to prevent loops:

**Port states:** Disabled → Blocking → Listening → Learning → Forwarding

**Root bridge election:**

$$root = min(bridge\_priority : bridge\_MAC)$$

Default bridge priority: 32768 (0x8000). Lower values win.

**Path cost calculation:**

| Speed | STP Cost (802.1D) | RSTP Cost (802.1w) |
|:---|:---:|:---:|
| 10 Mbps | 100 | 2,000,000 |
| 100 Mbps | 19 | 200,000 |
| 1 Gbps | 4 | 20,000 |
| 10 Gbps | 2 | 2,000 |

**Convergence:** STP: 30-50 seconds (2x forward_delay). RSTP: < 1 second (proposal/agreement).

---

## 6. VRF Implementation in Kernel

### Design

VRF (Virtual Routing and Forwarding) in Linux creates isolated routing domains using the `l3mdev` (Layer 3 master device) framework:

```
┌────────────────────────────────────────┐
│             Kernel routing              │
│  ┌──────────┐  ┌──────────┐  ┌──────┐ │
│  │ Table 254 │  │ Table 10 │  │ T 20 │ │
│  │  (main)   │  │ (vrf-red)│  │(blue)│ │
│  └─────┬─────┘  └────┬─────┘  └──┬───┘ │
│        │              │           │      │
│      eth0          eth1        eth2      │
│   (default)     (vrf-red)   (vrf-blue)  │
└────────────────────────────────────────┘
```

### Routing Table Isolation

Each VRF is associated with a kernel routing table:

$$VRF_{device} \xleftrightarrow{1:1} routing\_table\_id$$

When a packet arrives on an interface enslaved to a VRF:
1. The `l3mdev` framework sets the routing table based on the VRF
2. Route lookups use only that VRF's table
3. The packet never "leaks" into other VRF tables

### Socket Binding

Applications can bind to a specific VRF:

```c
setsockopt(fd, SOL_SOCKET, SO_BINDTODEVICE, "vrf-red", strlen("vrf-red")+1);
```

Or use `ip vrf exec` which sets `SO_BINDTODEVICE` automatically for all sockets created by the process.

### VRF Leaking

Inter-VRF routing (route leaking) requires explicit routes pointing to the other VRF's table:

$$\text{vrf-red table:} \quad 10.0.0.0/8 \to \text{nexthop via vrf-blue}$$

This is done with `ip route add ... vrf vrf-red nexthop via ... dev eth2` where eth2 belongs to vrf-blue.

---

## 7. Network Namespace Isolation

### Kernel Implementation

Network namespaces provide **complete network stack isolation**. Each namespace has its own:

| Resource | Isolated |
|:---|:---|
| Interfaces | Own set of net_devices |
| Routing tables | Independent FIB |
| Firewall rules | Separate netfilter tables |
| Sockets | Independent socket hash tables |
| /proc/net | Namespace-specific |
| ARP table | Own neighbor cache |

### Namespace vs VRF

| Feature | Network Namespace | VRF |
|:---|:---|:---|
| Isolation level | Complete stack | Routing only |
| Interfaces | Moved between namespaces | Enslaved to VRF device |
| Firewall | Separate per namespace | Shared (same netfilter) |
| Sockets | Fully isolated | Bound via SO_BINDTODEVICE |
| Overhead | Higher (full stack copy) | Lower (routing table only) |
| Use case | Containers, strong isolation | Multi-tenant routing |

### Communication Patterns

```
Namespace-to-namespace: veth pairs
Namespace-to-host: veth pair + bridge/routing
Namespace-to-external: veth + NAT/routing
```

The `veth` (virtual ethernet) device is always created as a **pair**. Each end can be placed in a different namespace, creating a point-to-point link between namespaces.

---

## 8. Netfilter/nftables Architecture

### Netfilter Hook Points

Netfilter provides 5 hook points in the packet path:

```
Incoming → PREROUTING → Routing → INPUT → Local process
              ↓                              ↓
           FORWARD                        OUTPUT
              ↓                              ↓
           POSTROUTING ←←←←←←←←←←←← Routing
              ↓
           Outgoing
```

### nftables vs iptables

| Feature | iptables | nftables |
|:---|:---|:---|
| Kernel component | x_tables | nf_tables |
| Rule format | CLI flags | Rule language |
| Atomic updates | No (per-rule) | Yes (full transaction) |
| Protocol families | Separate (iptables, ip6tables, ebtables, arptables) | Unified (`inet`, `bridge`, `arp`) |
| Sets | ipset (separate) | Native sets and maps |
| Performance | Linear rule matching | Optimized (sets, maps, concatenations) |
| Tracing | TRACE target | `nft monitor trace` |

### nftables Processing Model

```
Packet enters hook point
    ↓
Table lookup (family match)
    ↓
Chain evaluation (base chain with priority)
    ↓
Rule matching (expressions evaluated left-to-right)
    ↓
Verdict: accept, drop, reject, continue, jump, goto
    ↓
If no match: chain policy (accept or drop)
```

### Chain Priority

Multiple chains can attach to the same hook point. Priority determines order:

$$evaluation\_order = sort(chains, by=priority, ascending)$$

Common priorities:

| Priority | Name | Use |
|:---:|:---|:---|
| -300 | raw | Connection tracking bypass |
| -200 | mangle | Packet modification |
| -150 | dstnat | Destination NAT |
| 0 | filter | Default filtering |
| 100 | security | SELinux/security |
| 150 | srcnat | Source NAT |

---

## References

- kernel.org: Documentation/networking/
- kernel.org: Documentation/networking/bonding.rst
- kernel.org: Documentation/networking/vrf.rst
- IEEE 802.1AX — Link Aggregation
- IEEE 802.1Q — Virtual Bridged Local Area Networks
- IEEE 802.1D — MAC Bridges
- nftables wiki (wiki.nftables.org)
- NetworkManager Reference Manual
