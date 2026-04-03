# ip (iproute2) Deep Dive — Theory & Internals

> *The `ip` command is the modern interface to the Linux kernel's networking stack — it configures addresses, routes, neighbors, tunnels, and policy rules. Understanding its output means understanding the kernel's routing tables, ARP/NDP state machines, and the policy routing framework that enables multi-path and VRF.*

---

## 1. Routing Table — Longest Prefix Match

### The Algorithm

Linux maintains a routing table searched by longest prefix match (LPM):

$$\text{Route} = \arg\max_{r \in \text{FIB}} \text{prefix\_length}(r) \quad \text{s.t. } \text{dest} \in r.\text{network}$$

### FIB Data Structure

Linux uses LC-trie (Level-Compressed trie) for IPv4:

$$T_{lookup} = O(W) \quad \text{where } W = 32 \text{ (address width)}$$

In practice, with path compression: $O(\log N)$ where $N$ = number of routes.

### Route Table Scaling

| Routes | Memory | Lookup Time |
|:---:|:---:|:---:|
| 100 | ~50 KB | < 1 us |
| 10,000 | ~5 MB | < 1 us |
| 100,000 | ~50 MB | ~1 us |
| 1,000,000 (full BGP) | ~500 MB | ~1-2 us |

### Multiple Routing Tables

Linux supports 256 routing tables (0-255). `ip rule` directs traffic to specific tables:

$$\text{Table} = f(\text{source}, \text{fwmark}, \text{iif}, \text{tos}, \ldots)$$

Policy rules are evaluated by priority (lower number = higher priority):

| Priority | Rule | Table |
|:---:|:---|:---:|
| 0 | local | local (255) |
| 100 | from 10.0.0.0/8 | custom (100) |
| 200 | fwmark 0x1 | vpn (200) |
| 32766 | main | main (254) |
| 32767 | default | default (253) |

---

## 2. Neighbor Table — ARP/NDP State Machine

### ARP States

The `ip neigh` command shows the ARP/NDP cache state:

$$\text{State} \in \{\text{INCOMPLETE}, \text{REACHABLE}, \text{STALE}, \text{DELAY}, \text{PROBE}, \text{FAILED}\}$$

### State Transitions

```
          ARP request sent
NONE ──────────────────────> INCOMPLETE
                                │
                    ARP reply   │  timeout
                    received    │  (3 sec)
                       │        ↓
                       ↓      FAILED
                  REACHABLE
                       │
              timeout   │
           (30 sec)     │
                       ↓
                    STALE
                       │
                  traffic  │   no traffic
                  to host  │   (gc timeout)
                       ↓        ↓
                    DELAY    removed
                       │
                   1 sec │
                       ↓
                    PROBE ──(3 probes)──> FAILED
                       │
                   reply │
                       ↓
                  REACHABLE
```

### Timing Parameters

| Parameter | Default | Kernel Sysctl |
|:---|:---:|:---|
| Reachable time | 30 sec | `neigh/default/base_reachable_time_ms` |
| GC stale time | 60 sec | `neigh/default/gc_stale_time` |
| Retrans time | 1 sec | `neigh/default/retrans_time_ms` |
| Unicast probes | 3 | `neigh/default/ucast_solicit` |
| Mcast probes | 3 | `neigh/default/mcast_solicit` |
| Max entries | 1024 | `neigh/default/gc_thresh3` |

### Neighbor Table Sizing

$$N_{max} = \text{gc\_thresh3}$$

When the table exceeds `gc_thresh3`, garbage collection runs aggressively.

| Network | Active Hosts | gc_thresh3 Needed |
|:---|:---:|:---:|
| Small LAN | 50 | 1,024 (default) |
| Campus /16 | 5,000 | 8,192 |
| DC leaf switch | 50,000 | 65,536 |

---

## 3. Address Management — Multiple IPs per Interface

### Address Count

$$N_{addrs} = |\{a : a.\text{interface} = \text{dev}\}|$$

Linux has no practical limit on addresses per interface.

### Primary vs Secondary Addresses

The first address on a subnet is primary. Removing the primary removes all secondaries on that subnet (unless `promote_secondaries` is enabled).

$$\text{If primary removed: } \forall a \in \text{same subnet}: \text{delete}(a)$$

### Scope Hierarchy

| Scope | Value | Meaning |
|:---|:---:|:---|
| host | 254 | Valid only on this host (loopback) |
| link | 253 | Valid only on this link |
| global | 0 | Valid everywhere (routable) |

---

## 4. ECMP Routing — Multi-Path Configuration

### Equal-Cost Routes

```
ip route add 10.0.0.0/24 nexthop via 192.168.1.1 weight 1 \
                          nexthop via 192.168.2.1 weight 1
```

### Weight Distribution

$$P(\text{nexthop}_i) = \frac{w_i}{\sum_j w_j}$$

| Nexthop | Weight | Traffic Share |
|:---|:---:|:---:|
| 192.168.1.1 | 1 | 50% |
| 192.168.2.1 | 1 | 50% |

Unequal weights (UCMP):

| Nexthop | Weight | Traffic Share |
|:---|:---:|:---:|
| 192.168.1.1 (10G) | 3 | 75% |
| 192.168.2.1 (1G) | 1 | 25% |

### Hash Algorithm

Linux uses L3/L4 hash by default:

$$\text{Path} = H(\text{src\_ip}, \text{dst\_ip}, \text{proto}, \text{src\_port}, \text{dst\_port}) \mod \sum w_i$$

---

## 5. Tunnel Configuration — Encapsulation Overhead

### `ip tunnel` / `ip link add type`

| Tunnel Type | Overhead | Effective MTU (1500) |
|:---|:---:|:---:|
| IPIP (IP-in-IP) | 20 B | 1,480 |
| GRE | 24 B | 1,476 |
| GRE + key | 28 B | 1,472 |
| SIT (6in4) | 20 B | 1,480 |
| VXLAN | 50 B | 1,450 |
| WireGuard | 60 B (IPv4) / 80 B (IPv6) | 1,440 / 1,420 |

### PMTU Discovery

$$MTU_{tunnel} = MTU_{path} - O_{encap}$$

When PMTUD is blocked (ICMP filtered), packets are silently dropped at $> MTU_{tunnel}$.

---

## 6. Network Namespace — Isolation Math

### `ip netns` Isolation

Each namespace has independent:
- Routing table(s)
- Interface list
- ARP/NDP table
- iptables/nftables rules
- Socket bindings

### Resource per Namespace

$$M_{namespace} \approx M_{routing} + M_{interfaces} + M_{conntrack}$$

| Component | Memory | At 100 Namespaces |
|:---|:---:|:---:|
| Routing (default) | ~100 KB | 10 MB |
| Conntrack | ~1 MB base | 100 MB |
| Interface state | ~50 KB | 5 MB |
| **Total** | ~1.15 MB | ~115 MB |

Container hosts with 1,000 network namespaces: ~1.15 GB just for network state.

### veth Pair Performance

Each veth pair adds:

$$T_{veth} \approx 2 \times T_{netstack} \approx 2 \text{ us latency}$$

Throughput: typically 90-95% of bare metal.

---

## 7. VRF — Virtual Routing and Forwarding

### Table Isolation

$$\text{FIB}_{VRF_i} \perp \text{FIB}_{VRF_j} \quad \text{(independent routing tables)}$$

VRFs allow overlapping IP addresses:

$$10.0.0.1 \in VRF_A \neq 10.0.0.1 \in VRF_B$$

### Scaling

| VRFs | Routes/VRF | Total Routes | Memory |
|:---:|:---:|:---:|:---:|
| 10 | 1,000 | 10,000 | ~5 MB |
| 100 | 1,000 | 100,000 | ~50 MB |
| 10 | 100,000 | 1,000,000 | ~500 MB |

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $\arg\max(\text{prefix\_len})$ | Maximum selection | Longest prefix match |
| $O(W)$ / $O(\log N)$ | Complexity | FIB lookup time |
| $w_i / \sum w_j$ | Ratio | ECMP traffic distribution |
| $MTU - O_{encap}$ | Subtraction | Tunnel effective MTU |
| $H(\text{5-tuple}) \mod \sum w$ | Hash/modular | Path selection |
| $N_{ns} \times M_{per\_ns}$ | Product | Namespace memory |

---

*The `ip` command replaced ifconfig, route, arp, and tunnel with a single tool that speaks directly to the kernel via netlink. Every container on every Linux host uses `ip` commands (or their API equivalent) to set up networking — it's the foundation layer that everything else builds on.*
