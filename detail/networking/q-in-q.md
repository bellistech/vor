# The Engineering of Q-in-Q — Frame Formats, VLAN Scaling, and L2 Tunneling

> *Q-in-Q (802.1ad) multiplies VLAN space by nesting tags, but the real engineering challenge is MTU management, TPID interoperability, and MAC table scaling across provider networks.*

---

## 1. 802.1ad Frame Format

### Detailed Frame Structure

```
Bytes:  6     6      4          4          2       46-1500   4
     +-----+-----+--------+--------+---------+---------+-----+
     | DA  | SA  | S-tag   | C-tag  | EthType | Payload | FCS |
     +-----+-----+--------+--------+---------+---------+-----+

S-tag (4 bytes):
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         TPID (0x88A8)         | PCP |D|       S-VID          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

C-tag (4 bytes):
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         TPID (0x8100)         | PCP |D|       C-VID          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Where:
  TPID = Tag Protocol Identifier (16 bits)
  PCP  = Priority Code Point (3 bits, 802.1p QoS)
  D    = Drop Eligible Indicator (1 bit)
  VID  = VLAN Identifier (12 bits, 0-4095)
```

### Key Observations

- The S-tag and C-tag are **identical in structure** (both 4 bytes with PCP, DEI, VID)
- The **only** difference is the TPID value: 0x88A8 for S-tag vs 0x8100 for C-tag
- A switch determines which is the outer tag by checking the TPID at the expected position
- This means the hardware must be TPID-aware; older 802.1Q-only hardware sees 0x88A8 as an unknown EtherType

### Triple Tagging (Theoretical)

Nothing in the frame format prevents additional tag stacking:

```
DA | SA | 0x88A8 | S-VID | 0x88A8 | S-VID2 | 0x8100 | C-VID | EthType | Payload | FCS
```

But this is rarely used in practice because:
- Each additional tag adds 4 bytes of overhead
- Hardware support is limited
- Two levels of stacking (Q-in-Q) provide 4094 x 4094 = 16.7M virtual service instances, which is sufficient for most deployments

---

## 2. TPID History

### Evolution of Tag Protocol Identifiers

| Year | TPID | Standard | Context |
|------|------|----------|---------|
| 1998 | 0x8100 | IEEE 802.1Q | Original VLAN tagging standard |
| ~2001 | 0x9100 | Non-standard | Early Q-in-Q implementations (Cisco, others) |
| ~2002 | 0x9200 | Non-standard | Alternative vendor implementations |
| 2005 | 0x88A8 | IEEE 802.1ad | Official provider bridge TPID |
| 2011 | 0x88A8 | IEEE 802.1Q (merged) | 802.1ad merged into 802.1Q-2011 |

### The 0x9100 Problem

Before 802.1ad standardization, vendors used 0x9100 as the outer TPID. This created interoperability issues:

- **Cisco IOS (older Catalyst):** Used 0x8100 for both inner and outer tags (no TPID distinction)
- **Some vendors:** Used 0x9100 as the outer TPID
- **802.1ad:** Standardized 0x88A8

In multi-vendor environments, both ends must agree on the outer TPID. Most modern platforms support configurable TPID values:

```
! IOS-XR: Set TPID
interface GigabitEthernet0/0/0/0
 dot1q tunneling ethertype 0x88a8

! JunOS
interfaces ge-0/0/0 {
    ether-options {
        ieee-802.3ad ae0;
    }
    vlan-tagging;
    stacked-vlan-tagging;
    native-vlan-id 1;
}
```

### Hardware TPID Matching

Modern ASICs support multiple TPID values simultaneously. The parser checks:

1. Bytes 12-13 after DA+SA: is this a known TPID?
2. If 0x88A8: parse as S-tag, then check next 2 bytes for C-tag (0x8100)
3. If 0x8100: parse as C-tag (single-tagged frame)
4. If neither: untagged frame

Some platforms allow configuring the parser to accept 0x9100 as a valid outer TPID for backward compatibility.

---

## 3. VLAN Space Multiplication

### VLAN Capacity

Single 802.1Q:
$$V_{single} = 4094 \text{ VLANs (VID 1-4094, 0 and 4095 reserved)}$$

Q-in-Q (802.1ad):
$$V_{QinQ} = V_{S-VLAN} \times V_{C-VLAN} = 4094 \times 4094 = 16,760,836 \text{ unique service identifiers}$$

### Service Scaling Example

A service provider with 1,000 enterprise customers, each needing up to 100 VLANs:

**Without Q-in-Q:**
$$V_{needed} = 1000 \times 100 = 100,000 > 4094$$
Impossible with single 802.1Q.

**With Q-in-Q:**
$$S\text{-}VLANs = 1000 \text{ (one per customer)}$$
$$C\text{-}VLANs = 100 \text{ (per customer, independent)}$$
$$Total = 1000 \times 100 = 100,000 \leq 16,760,836$$
Easily accommodated.

### VLAN Assignment Models

**Model 1: One S-VLAN per customer**
$$S\text{-}VLAN = Customer\_ID$$
- Simple, clean separation
- Maximum customers = 4094
- Each customer gets full 4094 C-VLAN space

**Model 2: One S-VLAN per service type**
$$S\text{-}VLAN = Service\_Type \text{ (e.g., 100=Internet, 200=VoIP, 300=IPTV)}$$
- Fewer S-VLANs needed
- C-VLANs identify individual customers within a service
- QoS/policy applied per S-VLAN (per service class)

**Model 3: Selective mapping (hybrid)**
$$S\text{-}VLAN = f(C\text{-}VLAN, Port)$$
- Most flexible, most complex
- Different C-VLANs from different ports can map to the same or different S-VLANs

---

## 4. Selective vs Port-Based Q-in-Q

### Port-Based

```
Customer traffic (any C-VLAN) --> [Push S-VLAN X] --> Provider
```

**Properties:**
- All traffic on the port receives the same S-tag
- Configuration: 1 line per customer port
- No visibility into customer VLAN structure
- Simplest deployment model

### Selective

```
Customer VLAN 10  --> [Push S-VLAN 100] --> Provider (service A)
Customer VLAN 20  --> [Push S-VLAN 200] --> Provider (service B)
Customer VLAN 30+ --> [Push S-VLAN 300] --> Provider (default)
```

**Properties:**
- Different C-VLANs map to different S-VLANs
- Enables per-VLAN service differentiation
- More configuration per port
- Requires C-VLAN awareness at the provider edge

### Decision Matrix

| Criterion | Port-Based | Selective |
|-----------|-----------|-----------|
| Configuration complexity | Low | Medium-High |
| VLAN visibility | None | Full |
| Per-service QoS | No (all same S-VLAN) | Yes (different S-VLANs per service) |
| Scalability (config lines) | $O(P)$ where P = ports | $O(P \times V)$ where V = mapped VLANs |
| Flexibility | Low | High |
| Use case | Simple L2 transport | Multi-service delivery |

---

## 5. L2PT PDU Tunneling

### The Problem

Layer 2 control protocols use well-known multicast MAC addresses:

| Protocol | Destination MAC | EtherType/LLC |
|----------|----------------|---------------|
| STP/RSTP | 01:80:C2:00:00:00 | LLC 0x4242 |
| LACP | 01:80:C2:00:00:02 | 0x8809 |
| 802.1X | 01:80:C2:00:00:03 | 0x888E |
| LLDP | 01:80:C2:00:00:0E | 0x88CC |
| CDP | 01:00:0C:CC:CC:CC | SNAP |

IEEE 802.1D mandates that bridges **must not forward** frames with DA in the range 01:80:C2:00:00:00 to 01:80:C2:00:00:0F. This means provider switches will consume or drop customer L2 protocol frames.

### L2PT Solution

L2PT rewrites the destination MAC to a tunneling MAC that the provider network forwards:

```
Customer side:                    Provider network:
DA: 01:80:C2:00:00:00 (STP)  --> DA: 01:00:0C:CD:CD:D0 (Cisco L2PT)
                                  Provider switches forward normally
                              --> DA: 01:80:C2:00:00:00 (restored at far end)
```

### L2PT Processing Pipeline

```
Customer     Provider Edge (Ingress)    Provider Core    Provider Edge (Egress)    Customer
 BPDU   -->  Rewrite DA to tunnel MAC  --> Forward  -->  Rewrite DA back to BPDU --> BPDU
```

### Standards-Based Alternative

IEEE 802.1ad defines **protocol frame forwarding** behavior:
- Frames with reserved MACs are classified as "customer bridge protocol data"
- Provider bridges are supposed to forward them transparently based on C-VLAN
- In practice, many implementations still require explicit L2PT configuration

### Security Consideration

L2PT must be rate-limited to prevent abuse:
- A malicious customer could flood BPDUs to overwhelm the provider control plane
- Typical rate limit: 100-2000 packets per second per port
- Exceeding the threshold should shut down the L2PT tunnel on that port (not the entire link)

---

## 6. Q-in-Q Scaling Analysis

### MAC Address Table Scaling

In Q-in-Q, the provider learns customer MAC addresses in the S-VLAN MAC table:

$$MAC_{provider} = \sum_{i=1}^{N_{customers}} MAC_{customer_i}$$

For 1,000 customers with 500 MACs each:
$$MAC_{provider} = 1000 \times 500 = 500,000 \text{ entries}$$

This is a significant scaling challenge. Provider switches need large MAC tables.

**Comparison with PBB (802.1ah):**
$$MAC_{PBB} = N_{provider\_nodes} \text{ (only backbone MACs)}$$

PBB encapsulates customer frames in a new Ethernet header with provider backbone MACs, hiding customer MACs from the provider core. This reduces MAC table size from hundreds of thousands to tens or hundreds.

### Broadcast Domain Scaling

Each S-VLAN is a broadcast domain. Customer broadcasts and unknown unicasts are flooded within the S-VLAN:

$$BUM_{S\text{-}VLAN} = \sum_{ports \in S\text{-}VLAN} BUM_{customer\_port}$$

For port-based Q-in-Q with one S-VLAN per customer, BUM traffic is limited to that customer's ports. For service-based S-VLAN assignment, many customers share a broadcast domain, increasing BUM traffic.

### Scaling Limits Summary

| Factor | Q-in-Q Limit | Mitigation |
|--------|-------------|------------|
| S-VLANs | 4094 | Use selective mapping to share S-VLANs |
| MAC table | Platform-dependent (32K-512K) | PBB, EVPN |
| BUM flooding | All ports in S-VLAN | Storm control, S-VLAN per customer |
| MTU | 4 extra bytes per tag | Jumbo frames |
| L2PT rate | Per-port configurable | Rate limiting |

---

## 7. Comparison with PBB, VPLS, and EVPN

### Architecture Comparison

| Feature | Q-in-Q | PBB (802.1ah) | VPLS | EVPN-VXLAN |
|---------|--------|---------------|------|------------|
| Encapsulation | S-tag + C-tag | B-DA + B-SA + B-tag + I-SID + C-tag | MPLS labels | VXLAN (UDP) + VNI |
| Service ID | S-VLAN (12-bit) | I-SID (24-bit) | VPN ID | VNI (24-bit) |
| Max services | 4,094 | 16,777,216 | Unlimited (MPLS labels) | 16,777,216 |
| MAC learning | Provider learns customer MACs | Provider core sees only B-MACs | PE learns customer MACs | PE learns via BGP + data plane |
| Control plane | None (data plane learning) | None (data plane learning) | LDP or BGP | BGP EVPN |
| BUM handling | Flood in S-VLAN | Flood in B-VLAN | Pseudowire replication | Ingress replication or multicast |
| Scalability | Low (MAC explosion) | Medium (B-MAC isolation) | Medium (PE state) | High (BGP control plane) |
| Complexity | Low | Medium | High | High |
| Transport | Ethernet | Ethernet | MPLS | IP/UDP |

### When to Use Each

**Q-in-Q:** Metro Ethernet access, simple service provider topologies, fewer than 4094 services, small MAC table environments.

**PBB:** Large metro networks needing MAC isolation, 802.1ah-capable hardware, scaling beyond 4094 services.

**VPLS:** MPLS-based provider networks, multi-point L2 services, existing MPLS infrastructure.

**EVPN-VXLAN:** Modern data center and WAN, need for control-plane MAC learning, multi-tenancy at scale, IP-based underlay.

### Migration Path

```
Q-in-Q (simple) --> PBB-EVPN (MAC isolation + control plane)
                 --> VPLS (MPLS transport)
                 --> EVPN-VXLAN (modern, scalable)
```

Most greenfield deployments today choose EVPN-VXLAN over Q-in-Q for its superior scalability and control-plane MAC learning. Q-in-Q remains relevant for simple metro access networks and as the CE-facing technology feeding into EVPN/VPLS PEs.

---

## See Also

- ethernet, vxlan, mpls, g8032-erp

## References

- [IEEE 802.1ad — Provider Bridges](https://standards.ieee.org/standard/802_1ad-2005.html)
- [IEEE 802.1Q — Bridges and Bridged Networks](https://standards.ieee.org/standard/802_1Q-2022.html)
- [IEEE 802.1ah — Provider Backbone Bridges](https://standards.ieee.org/standard/802_1ah-2008.html)
- [RFC 4761 — Virtual Private LAN Service (VPLS) Using BGP](https://www.rfc-editor.org/rfc/rfc4761)
- [RFC 7348 — VXLAN](https://www.rfc-editor.org/rfc/rfc7348)
- [RFC 7432 — BGP MPLS-Based Ethernet VPN](https://www.rfc-editor.org/rfc/rfc7432)
