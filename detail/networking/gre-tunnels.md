# GRE Tunnels — Generic Routing Encapsulation Protocol Architecture

> *GRE is a stateless, point-to-point tunneling protocol that wraps arbitrary network-layer packets inside IP datagrams. Its design is governed by encapsulation overhead arithmetic, MTU path calculations, recursive routing graph cycles, and the multipoint extension model that enables DMVPN's spoke-to-spoke shortcuts. The protocol's simplicity — a 4-byte header with no built-in security — is both its strength and its limitation.*

---

## 1. Encapsulation Model — Protocol Layering

### The GRE Encapsulation Stack

GRE creates a tunnel by prepending a delivery header (outer IP) and a GRE header to the original (inner) packet:

```
[Outer IP Header][GRE Header][Inner Packet (original IP + payload)]
     20 bytes      4-16 bytes     variable
```

The outer IP header uses protocol number 47 to indicate GRE. The GRE header's Protocol Type field identifies the inner payload using standard EtherType values.

### Formal Encapsulation

Given an inner packet $P_{\text{inner}}$ of length $L$:

$$P_{\text{outer}} = H_{\text{IP}} \| H_{\text{GRE}} \| P_{\text{inner}}$$

Where $\|$ denotes concatenation. The total length:

$$L_{\text{outer}} = L_{\text{IP}} + L_{\text{GRE}} + L_{\text{inner}}$$

With $L_{\text{IP}} = 20$ bytes (no options) and $L_{\text{GRE}} = 4 + 4C + 4K + 4S$ bytes, where $C$, $K$, $S$ are the checksum, key, and sequence number flags (0 or 1).

---

## 2. GRE Header Bit Fields — Flag Semantics

### Base Header (4 bytes, always present)

| Bits | Field | Description |
|:---:|:---|:---|
| 0 | C (Checksum) | If 1, Checksum and Reserved1 fields are present (4 bytes) |
| 1 | Reserved | Must be 0 |
| 2 | K (Key) | If 1, Key field is present (4 bytes) |
| 3 | S (Sequence) | If 1, Sequence Number field is present (4 bytes) |
| 4-12 | Reserved0 | Must be 0 |
| 13-15 | Version | 0 for standard GRE, 1 for enhanced GRE (PPTP) |
| 16-31 | Protocol Type | EtherType of encapsulated payload |

### GRE Header Size Calculation

$$L_{\text{GRE}} = 4 + 4 \times (C + K + S)$$

| Configuration | C | K | S | GRE Header Size | Total Overhead (with IP) |
|:---|:---:|:---:|:---:|:---:|:---:|
| Minimal | 0 | 0 | 0 | 4 bytes | 24 bytes |
| Key only | 0 | 1 | 0 | 8 bytes | 28 bytes |
| Checksum only | 1 | 0 | 0 | 8 bytes | 28 bytes |
| Key + sequence | 0 | 1 | 1 | 12 bytes | 32 bytes |
| All options | 1 | 1 | 1 | 16 bytes | 36 bytes |

### Checksum Computation

When the C bit is set, the checksum covers the entire GRE header and payload:

$$\text{checksum} = \text{ones\_complement\_sum}(H_{\text{GRE}} \| P_{\text{inner}})$$

This is the standard Internet checksum (RFC 1071) — the same algorithm used for IP, TCP, and UDP checksums. The checksum field is set to zero during computation.

### Key Field Semantics

The 32-bit key field serves two purposes:

1. **Tunnel identification**: differentiates multiple tunnels between the same endpoint pair
2. **Weak authentication**: both sides must agree on the key value (but it is transmitted in cleartext)

The key is not a security mechanism — it provides no confidentiality or integrity. It is a demultiplexing aid.

### Sequence Number

The 32-bit sequence number enables in-order delivery detection. The sender increments the counter for each packet:

$$S_n = S_{n-1} + 1 \quad (\text{mod } 2^{32})$$

The receiver can detect reordering and loss but GRE provides no retransmission mechanism — it simply discards out-of-order packets when strict ordering is configured.

---

## 3. MTU Path Analysis — Overhead Arithmetic

### The MTU Problem

GRE encapsulation increases packet size. If the encapsulated packet exceeds the path MTU along the tunnel transport path, fragmentation occurs. This creates a cascading performance problem.

### Effective Tunnel MTU

Given a path MTU of $M$ along the transport path:

$$MTU_{\text{tunnel}} = M - L_{\text{IP}} - L_{\text{GRE}}$$

For a standard 1500-byte Ethernet path with minimal GRE:

$$MTU_{\text{tunnel}} = 1500 - 20 - 4 = 1476 \text{ bytes}$$

### TCP MSS Adjustment

TCP Maximum Segment Size must account for both tunnel overhead and TCP/IP headers:

$$MSS = MTU_{\text{tunnel}} - L_{\text{IP\_inner}} - L_{\text{TCP}}$$
$$MSS = 1476 - 20 - 20 = 1436 \text{ bytes}$$

### Double Encapsulation (GRE over IPsec)

When GRE is protected by IPsec in tunnel mode:

$$L_{\text{total}} = L_{\text{outer\_IP}} + L_{\text{ESP}} + L_{\text{IV}} + L_{\text{GRE}} + L_{\text{inner}} + L_{\text{pad}} + L_{\text{ESP\_auth}}$$

For AES-256-CBC with SHA-256 HMAC:

| Component | Size (bytes) |
|:---|:---:|
| Outer IP header | 20 |
| ESP header | 8 |
| IV (AES-CBC) | 16 |
| GRE header | 4 |
| Inner IP header | 20 |
| Payload | variable |
| ESP padding | 1-16 (block alignment) |
| ESP pad length + next header | 2 |
| ESP auth (SHA-256 truncated) | 16 |
| **Minimum overhead** | **~86** |

Using IPsec transport mode instead of tunnel mode eliminates one IP header:

$$\text{Transport mode savings} = 20 \text{ bytes (one fewer IP header)}$$

$$MTU_{\text{GRE+IPsec\_transport}} \approx 1500 - 66 = 1434 \text{ bytes}$$

### Fragmentation Decision Tree

```
Inner packet arrives at tunnel ingress:
│
├─ L_inner ≤ MTU_tunnel?
│   ├─ Yes → Encapsulate, send
│   └─ No → Check DF bit
│       ├─ DF=0 → Pre-fragment inner packet, encapsulate each fragment
│       └─ DF=1 → Check PMTUD
│           ├─ PMTUD enabled → Send ICMP "frag needed" (type 3, code 4) to source
│           └─ PMTUD disabled → Fragment outer packet (post-fragmentation)
```

### Pre-fragmentation vs Post-fragmentation

**Pre-fragmentation** (preferred): fragment the inner packet before GRE encapsulation. Each fragment gets its own GRE + IP header. The receiving tunnel endpoint does not need to reassemble.

$$N_{\text{fragments}} = \lceil L_{\text{inner}} / MTU_{\text{tunnel}} \rceil$$

Each fragment is independently routable:

$$L_{\text{frag}_i} = \min(MTU_{\text{tunnel}}, L_{\text{remaining}}) + L_{\text{IP}} + L_{\text{GRE}} \leq M$$

**Post-fragmentation** (suboptimal): encapsulate the full inner packet, then fragment the outer packet. The receiving tunnel endpoint must reassemble the outer packet before de-encapsulation.

$$N_{\text{fragments}} = \lceil (L_{\text{inner}} + L_{\text{GRE}} + L_{\text{IP}}) / M \rceil$$

Post-fragmentation is worse because:
- Only the first fragment contains the GRE header
- Fragment loss requires retransmission of the entire original packet
- Reassembly consumes memory and CPU at the tunnel egress

---

## 4. Recursive Routing — Graph Cycle Detection

### The Recursive Routing Problem

A GRE tunnel creates a virtual link in the routing topology. If the routing protocol advertises the tunnel destination's reachability via the tunnel itself, a routing loop forms.

### Formal Model

Let $G = (V, E)$ be the routing graph where $V$ is the set of routers and $E$ is the set of links. A tunnel $t$ from router $A$ to router $B$ adds a virtual edge:

$$E' = E \cup \{(A, B)_t\}$$

Router $A$ needs to reach $B$'s physical address $IP_B$ to send encapsulated packets. If the best path to $IP_B$ traverses the tunnel edge $(A, B)_t$:

$$\text{next\_hop}(IP_B) = (A, B)_t \implies \text{encapsulate}(P, IP_B) \rightarrow \text{next\_hop}(IP_B) = (A, B)_t \implies \ldots$$

This is an infinite loop. The router detects it and brings the tunnel interface down.

### Route Recursion Depth

Most routing implementations limit recursion depth to prevent infinite loops:

$$\text{recursion\_depth} \leq D_{\text{max}} \quad (\text{typically } D_{\text{max}} = 3 \text{ or } 4)$$

When the limit is exceeded, the route is declared invalid and the tunnel interface transitions to down state.

### Prevention Strategies

The fundamental solution is ensuring the tunnel transport route (the route to the tunnel destination) does not traverse the tunnel itself. This requires maintaining two independent routing domains:

1. **Underlay routing**: carries tunnel endpoint reachability (static routes or separate IGP instance)
2. **Overlay routing**: runs over the tunnel, carries application traffic routes

The separation can be achieved through:

- Static routes with lower administrative distance
- VRF-based transport isolation (front-door VRF)
- Route filtering (distribute-list blocking tunnel destination from tunnel interface)
- Administrative distance manipulation

---

## 5. Multipoint GRE (mGRE) — NHRP Resolution Model

### Point-to-Point vs Multipoint

A standard GRE tunnel has a fixed destination — one tunnel interface maps to one remote endpoint. With $n$ remote sites, a hub needs $n$ tunnel interfaces:

$$\text{P2P tunnels required (full mesh)} = \frac{n \times (n-1)}{2}$$

mGRE removes the fixed destination. A single tunnel interface can communicate with multiple endpoints, with NHRP (Next Hop Resolution Protocol) providing dynamic destination resolution.

### NHRP Resolution Process

NHRP maps tunnel (overlay) addresses to transport (underlay) addresses. The process:

1. Spoke registers with hub (NHS): overlay IP $\rightarrow$ underlay IP mapping
2. Hub maintains NHRP cache of all spoke registrations
3. When spoke A wants to reach spoke B:
   a. Spoke A sends NHRP Resolution Request to hub
   b. Hub looks up spoke B's mapping and responds (or forwards request)
   c. Spoke A installs a shortcut route: spoke B's overlay IP $\rightarrow$ spoke B's underlay IP
   d. Subsequent traffic flows directly (spoke-to-spoke)

### DMVPN Phase Comparison

| Phase | Hub-to-Spoke | Spoke-to-Spoke | Routing |
|:---|:---|:---|:---|
| Phase 1 | Direct | Via hub | Next-hop = hub |
| Phase 2 | Direct | Direct (NHRP) | Next-hop = spoke |
| Phase 3 | Direct | Direct (NHRP redirect) | Next-hop = hub, shortcut installed |

### Phase 3 NHRP Redirect/Shortcut

In Phase 3, the hub does not change next-hop in routing updates (simplifying the routing design). Instead:

1. Hub receives traffic from spoke A destined for spoke B
2. Hub sends NHRP Redirect to spoke A: "reach spoke B directly at underlay IP"
3. Spoke A installs NHRP shortcut route overriding the routing table entry
4. Subsequent spoke-A-to-spoke-B traffic flows directly

The shortcut has a finite lifetime and is refreshed by traffic or NHRP registration:

$$T_{\text{shortcut}} = T_{\text{nhrp\_holdtime}} \quad (\text{default 7200 seconds})$$

### mGRE Scalability

A hub with $n$ spokes maintains:
- 1 tunnel interface (vs $n$ for P2P)
- $n$ NHRP cache entries
- $n$ routing adjacencies (if dynamic routing is used)

Memory per spoke:

$$M_{\text{per\_spoke}} = M_{\text{NHRP}} + M_{\text{routing}} + M_{\text{crypto}} \approx 1\text{-}5 \text{ KB}$$

For 1,000 spokes: approximately 1-5 MB of hub memory for tunnel state.

---

## 6. GRE TAP — Layer 2 Encapsulation

### L2 vs L3 GRE

Standard GRE (L3) encapsulates network-layer packets. GRETAP (L2 GRE) encapsulates entire Ethernet frames, including the MAC header:

```
L3 GRE: [Outer IP][GRE (proto=0x0800)][Inner IP][Payload]
L2 GRE: [Outer IP][GRE (proto=0x6558)][Inner Ethernet][Inner IP][Payload]
```

The additional Ethernet header adds 14 bytes of overhead:

$$L_{\text{GRETAP}} = L_{\text{IP}} + L_{\text{GRE}} + L_{\text{Ethernet}} + L_{\text{inner}}$$
$$\text{Overhead}_{\text{GRETAP}} = 20 + 4 + 14 = 38 \text{ bytes}$$
$$MTU_{\text{GRETAP}} = 1500 - 38 = 1462 \text{ bytes (inner IP MTU)}$$

### Use Cases for L2 GRE

1. **VLAN extension**: extend a broadcast domain across L3 boundaries
2. **VM migration**: maintain L2 adjacency for live migration
3. **Network monitoring**: mirror traffic (ERSPAN uses GRE with L2 payload)
4. **Legacy protocol support**: carry non-IP protocols (STP, ARP, IS-IS)

### ERSPAN (Encapsulated Remote SPAN)

ERSPAN uses GRE to transport mirrored traffic across routed networks:

```
[Outer IP][GRE][ERSPAN Header][Original Ethernet Frame]
```

ERSPAN adds its own 8-byte header inside GRE, bringing total overhead to:

$$\text{Overhead}_{\text{ERSPAN}} = 20 + 4 + 8 + 14 = 46 \text{ bytes}$$

ERSPAN Type II (used on most Cisco platforms) includes a 10-bit session ID, allowing multiple SPAN sessions to share a single GRE tunnel.

---

## 7. GRE Keepalive Mechanism

### Keepalive Packet Construction

GRE keepalives work differently from most keepalive protocols. Instead of a dedicated keepalive message, the router constructs a GRE packet addressed to itself:

1. Create an inner IP packet with destination = local tunnel IP
2. Encapsulate in GRE with tunnel destination = remote physical IP
3. Send through the tunnel
4. Remote end de-encapsulates the GRE header, sees the inner packet
5. Inner packet is routed — back through the tunnel to the originator
6. Originator receives its own keepalive — tunnel is confirmed bidirectional

### Keepalive Timer Model

Given keepalive interval $I$ seconds and retry count $R$:

$$T_{\text{down}} = I \times R$$

The tunnel transitions to down state after $R$ consecutive missed keepalives. Default values: $I = 10$, $R = 3$, so $T_{\text{down}} = 30$ seconds.

### Failure Detection Comparison

| Mechanism | Detection Time | Overhead | Notes |
|:---|:---:|:---|:---|
| GRE keepalive | $I \times R$ (30s default) | 1 packet per $I$ seconds | Both ends must support |
| BFD over GRE | 50ms-3s | 3-20 pps | Hardware-assisted on some platforms |
| Routing protocol hello | Protocol-dependent | Protocol-dependent | OSPF dead=40s, EIGRP hold=15s |
| IP SLA | Configurable | Configurable | Most flexible, highest overhead |

BFD (Bidirectional Forwarding Detection) provides sub-second failure detection and is preferred for fast convergence requirements.

---

## 8. Security Analysis

### GRE Security Properties

GRE provides **none** of the three fundamental security properties:

| Property | GRE Support | Implication |
|:---|:---:|:---|
| Confidentiality | No | Payload is visible to any observer on the path |
| Integrity | Partial (checksum) | Checksum detects corruption but not tampering |
| Authentication | No | Any host can inject GRE packets (key is cleartext) |

### Attack Vectors

1. **Packet injection**: attacker sends crafted GRE packets to tunnel endpoint; if the source IP matches the expected peer, the packet is accepted
2. **Tunnel hijacking**: attacker spoofs the tunnel source IP and injects traffic into the overlay network
3. **Reconnaissance**: GRE payload is unencrypted; passive observers see all tunneled traffic
4. **Denial of service**: flooding GRE endpoint with packets consumes de-encapsulation resources

### IPsec as the Security Layer

GRE relies entirely on IPsec for security:

$$\text{Secure GRE} = \text{GRE (encapsulation)} + \text{IPsec (security)}$$

IPsec transport mode is preferred over tunnel mode for GRE protection because GRE already provides the outer IP header — tunnel mode would add a redundant third IP header:

```
Tunnel mode:  [IPsec Outer IP][ESP][GRE Outer IP][GRE][Inner IP][Payload]  ← 3 IP headers
Transport mode: [GRE Outer IP][ESP][GRE][Inner IP][Payload]               ← 2 IP headers
```

Transport mode saves 20 bytes of overhead per packet.

---

## 9. Performance Characteristics

### Throughput Impact

GRE encapsulation reduces effective throughput due to overhead:

$$\text{Efficiency} = \frac{L_{\text{payload}}}{L_{\text{payload}} + L_{\text{overhead}}} \times 100\%$$

| Payload Size | GRE Overhead | Efficiency | With IPsec |
|:---:|:---:|:---:|:---:|
| 1460 bytes (typical TCP) | 24 bytes | 98.4% | ~94.3% |
| 500 bytes (medium) | 24 bytes | 95.4% | ~85.5% |
| 64 bytes (small/VoIP) | 24 bytes | 72.7% | ~42.7% |

Small packets suffer disproportionately from encapsulation overhead.

### CPU Cost

On platforms without hardware GRE offload, encapsulation and de-encapsulation consume CPU cycles:

$$C_{\text{GRE}} = C_{\text{header\_build}} + C_{\text{route\_lookup}} + C_{\text{checksum}} \quad (\text{if enabled})$$

Typical CPU cost per packet on modern hardware:

| Operation | Approximate Cost |
|:---|:---|
| GRE encapsulation | 100-500 ns |
| GRE de-encapsulation | 100-500 ns |
| IPsec encryption (AES-NI) | 200-1000 ns |
| Full GRE+IPsec | 500-2000 ns |

At 1 million packets/second, GRE+IPsec consumes approximately 0.5-2.0 CPU cores.

### Hardware Offload

Modern NICs and ASICs support GRE offload:

- **Encap/decap offload**: ASIC handles GRE header addition/removal
- **TSO with GRE**: TCP Segmentation Offload works through GRE encapsulation
- **RSS with GRE**: Receive Side Scaling hashes on inner packet headers
- **Checksum offload**: NIC computes GRE checksum in hardware

With full hardware offload, GRE throughput approaches line rate.

---

## 10. GRE vs Alternative Tunneling Protocols

### Protocol Comparison

| Feature | GRE | VXLAN | Geneve | IPsec (tunnel) | WireGuard |
|:---|:---:|:---:|:---:|:---:|:---:|
| Layer | L3 (L2 with TAP) | L2 | L2 | L3 | L3 |
| Transport | IP (proto 47) | UDP 4789 | UDP 6081 | IP (proto 50) | UDP 51820 |
| Overhead | 24 bytes | 50 bytes | 50+ bytes | ~60 bytes | 60 bytes |
| Encryption | No | No | No | Yes | Yes |
| Multipoint | mGRE | Inherent (multicast/unicast) | Inherent | No | Yes |
| NAT traversal | No | Yes (UDP) | Yes (UDP) | NAT-T (UDP 4500) | Yes (UDP) |
| Scalability | Moderate | High (24-bit VNI) | High (24-bit VNI) | Low | Moderate |
| Hardware offload | Widespread | Widespread | Growing | Specialized | Limited |

### When to Use GRE

- Point-to-point site connectivity with routing protocol support
- DMVPN hub-and-spoke topologies (mGRE + NHRP + IPsec)
- Carrying multicast or non-IP protocols across IP networks
- Network monitoring (ERSPAN)
- Simple tunneling where encryption is not required or is handled separately

---

## Prerequisites

- IP packet structure and header fields, IP protocol numbers, MTU and fragmentation, routing tables and longest prefix match, IPsec fundamentals (for GRE over IPsec), NHRP and DMVPN concepts (for mGRE)

---

*GRE's genius is its simplicity: a 4-byte header that turns any IP network into a virtual wire. It carries IPv4, IPv6, MPLS, Ethernet frames — anything with an EtherType. It asks nothing of the transport network except IP reachability between endpoints. This minimalism made it the foundation of DMVPN, the backbone of enterprise overlay networking for two decades. But simplicity has costs: no security, no flow entropy for ECMP, no built-in multitenancy. VXLAN and Geneve address these gaps, but GRE endures — because sometimes all you need is a tunnel.*
