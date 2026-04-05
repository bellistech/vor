# Advanced IPv6 — Extension Header Chaining, NDP State Machines, Transition Mechanisms, and Security

> *IPv6 is not simply "longer addresses." Its redesigned header with extension header chaining, mandatory PMTUD, stateless autoconfiguration via NDP, and built-in multicast fundamentally change how packets traverse networks. The transition from IPv4 involves algorithmic address translation, stateless prefix mapping, and protocol encapsulation — each with distinct trade-offs in state, performance, and end-to-end transparency.*

---

## 1. IPv6 Header Format — Simplified for Speed

### Fixed Header Structure

The IPv6 base header is exactly 40 bytes — fixed size, no header checksum, no options field:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Version| Traffic Class |           Flow Label                  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Payload Length        |  Next Header  |   Hop Limit   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                                                               +
|                       Source Address                           |
+                        (128 bits)                              +
|                                                               |
+                                                               +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                                                               +
|                     Destination Address                        |
+                        (128 bits)                              +
|                                                               |
+                                                               +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Design Decisions vs IPv4

| IPv4 Header | IPv6 Header | Rationale |
|:---|:---|:---|
| Variable length (20-60 bytes) | Fixed 40 bytes | Hardware fast-path, no IHL calculation |
| Header checksum | Removed | L2 (Ethernet CRC) and L4 (TCP/UDP checksum) cover integrity |
| Fragment fields in base header | Separate Fragment Extension Header | Routers never fragment — only endpoints |
| Options in base header | Extension headers (chained) | Options rarely examined; keep the fast path clean |
| TTL (Time to Live) | Hop Limit | Name reflects actual behavior (hop count, not time) |
| Protocol field | Next Header | Dual purpose: identifies upper-layer OR next extension header |
| IHL (Header Length) | Removed | Fixed size makes it unnecessary |
| Total Length | Payload Length | Excludes the 40-byte base header itself |

### Why No Header Checksum

The removal of the header checksum eliminates per-hop recalculation. In IPv4, every router that decrements TTL must recompute the header checksum. With millions of packets per second on modern routers, this saves significant processing. The integrity argument:

- Layer 2 (Ethernet FCS, Wi-Fi CRC) catches bit errors on every hop
- Layer 4 (TCP checksum, UDP checksum — mandatory in IPv6) catches end-to-end corruption
- IPsec AH provides cryptographic integrity when needed

The gap: a bit flip in the Hop Limit or addresses between L2 hops would go undetected. In practice, L2 FCS catches this with overwhelming probability. The engineering trade-off favors forwarding speed.

---

## 2. Extension Header Chaining

### The Chain Model

Extension headers form a singly-linked list. Each header's Next Header (NH) field is a pointer to the type of the following header:

```
[IPv6 Base Header]  NH=0
        │
        ▼
[Hop-by-Hop Options]  NH=43
        │
        ▼
[Routing Header]  NH=44
        │
        ▼
[Fragment Header]  NH=6
        │
        ▼
[TCP Header + Data]
```

### Processing Rules

1. **Hop-by-Hop Options**: Must be examined by every node on the path. Must appear immediately after the IPv6 base header if present. This is the only extension header that intermediate routers process.

2. **All other extension headers**: Processed only by the node identified in the Destination Address field. Intermediate routers forward the packet without inspecting these headers.

3. **Order matters**: RFC 8200 specifies a recommended order. While not strictly enforced, violating the order causes interoperability problems with middleboxes and firewalls.

### The Middlebox Problem

Extension headers create a fundamental tension with stateless packet filtering:

A stateless firewall that needs to identify the upper-layer protocol (e.g., "permit TCP port 443") must walk the entire extension header chain to find the final Next Header value. This requires:

- Parsing variable-length headers
- Handling unknown extension header types
- Processing chains of arbitrary depth

In practice, many firewalls, load balancers, and IDS systems fail when they encounter extension headers. Studies have shown that packets with extension headers (especially Hop-by-Hop and Destination Options) have significantly higher drop rates across the internet. RFC 7045 addresses this, recommending that transit routers forward packets with unknown extension headers rather than dropping them, but compliance is inconsistent.

### Fragment Header Mechanics

IPv6 fragmentation differs fundamentally from IPv4:

- Only the source node may fragment
- The minimum MTU is 1280 bytes (vs 576 in IPv4)
- Path MTU Discovery is mandatory (ICMPv6 Packet Too Big)
- Fragment overlap is explicitly forbidden (RFC 5722)

The prohibition on fragment overlap was a security hardening measure. In IPv4, overlapping fragments were used in evasion attacks (Teardrop, Rose, fragmentation-based IDS evasion) where overlapping regions contained different data, confusing reassembly at the destination vs the IDS.

Fragment identification uses a 32-bit field, giving $2^{32} = 4,294,967,296$ unique fragment IDs. At high packet rates, ID reuse can cause misassembly:

$$\text{Reuse interval} = \frac{2^{32}}{\text{fragments/sec}}$$

At 1 million fragments/sec: reuse every ~4,295 seconds (~72 minutes). At 10 million/sec: reuse every ~429 seconds (~7 minutes). RFC 6864 (IPv4) addresses this concern; IPv6 mitigates it by limiting fragmentation to endpoints where fragment ID management is simpler.

---

## 3. NDP State Machine

### Neighbor Cache States

The NDP neighbor cache implements a state machine that tracks the reachability of on-link neighbors:

```
                    ┌─────────────────────────────────────┐
                    │                                     │
                    ▼                                     │
    ┌──────────┐  NS sent   ┌────────────┐              │
    │          │──────────→│            │   upper-layer  │
    │INCOMPLETE│            │ REACHABLE  │←──confirm─────┘
    │          │←──timeout──│            │
    └──────────┘  (no NA)   └────────────┘
         │                       │
         │ NA received           │ timeout (ReachableTime)
         │                       ▼
         │                  ┌──────────┐
         └────────────────→│          │
                            │  STALE   │
                            │          │
                            └──────────┘
                                 │
                                 │ packet sent to neighbor
                                 ▼
                            ┌──────────┐
                            │          │
                            │  DELAY   │──── DELAY_FIRST_PROBE_TIME (5s)
                            │          │
                            └──────────┘
                                 │
                                 │ timeout, no confirm
                                 ▼
                            ┌──────────┐
                            │          │──── send NS, wait RetransTimer
                            │  PROBE   │──── MAX_UNICAST_SOLICIT times
                            │          │
                            └──────────┘
                                 │
                                 │ all probes fail
                                 ▼
                            ┌──────────┐
                            │ FAILED   │
                            └──────────┘
```

### Timing Parameters

| Parameter | Default | Purpose |
|:---|:---:|:---|
| RetransTimer | 1,000 ms | Time between NS retransmissions |
| ReachableTime | 30,000 ms | Duration in REACHABLE state (base) |
| DELAY_FIRST_PROBE_TIME | 5,000 ms | Time in DELAY before moving to PROBE |
| MAX_UNICAST_SOLICIT | 3 | NS probes before declaring FAILED |
| MAX_MULTICAST_SOLICIT | 3 | DAD probes before declaring unique |

ReachableTime is randomized between 0.5x and 1.5x of the base value (from RA) to prevent synchronization of neighbor probes across hosts. The actual value used:

$$\text{ReachableTime}_{actual} \in [0.5 \times \text{Base}, 1.5 \times \text{Base}]$$

With default base of 30 seconds: actual range is 15-45 seconds.

### Upper-Layer Reachability Confirmation

A critical optimization: TCP acknowledgments serve as reachability confirmation. When TCP receives an ACK from a neighbor, it signals NDP that the neighbor is reachable, resetting the REACHABLE timer without sending any NS probes. This dramatically reduces NDP traffic on links with active TCP sessions.

Protocols that provide upper-layer confirmation:
- TCP (ACK received = neighbor reachable)
- SCTP (SACK/HEARTBEAT-ACK)

Protocols that do NOT provide confirmation:
- UDP (no acknowledgment mechanism)
- ICMPv6 echo-reply (implementation-dependent)

---

## 4. SLAAC Algorithm — EUI-64 and Privacy Extensions

### EUI-64 Interface Identifier Generation

The original SLAAC algorithm (RFC 4862) derives the 64-bit Interface Identifier from the 48-bit MAC address:

```
MAC: aa:bb:cc:dd:ee:ff (48 bits)

Step 1: Split at the OUI boundary
  OUI:    aa:bb:cc (24 bits)
  Device: dd:ee:ff (24 bits)

Step 2: Insert FF:FE in the middle
  aa:bb:cc:ff:fe:dd:ee:ff (64 bits)

Step 3: Flip the Universal/Local bit (bit 7 of first byte)
  If bit 7 = 0 (universally administered) → set to 1
  If bit 7 = 1 (locally administered) → set to 0

  Binary of 'aa': 10101010
  Flip bit 7:     10101000 = 'a8'

Result IID: a8bb:ccff:fedd:eeff
Full SLAAC address: <prefix>:a8bb:ccff:fedd:eeff
```

### Why Flip the U/L Bit?

The decision to invert the meaning of the Universal/Local bit (RFC 4291, Section 2.5.1) is counterintuitive. In IEEE MAC semantics, U/L=0 means universally administered (burned-in). In IPv6 IID semantics, U/L=1 means globally unique.

The rationale: when manually configuring Interface IDs (e.g., ::1, ::2), the U/L bit is naturally 0. Flipping the semantics means manually configured addresses have U/L=0 (locally assigned), and EUI-64-derived addresses have U/L=1 (globally unique). This makes manual configuration simpler — you do not have to remember to set the bit.

### Privacy Extensions (RFC 8981)

EUI-64 addresses embed the MAC address, creating a persistent tracking identifier across networks and time. Privacy extensions solve this:

**Temporary Address Generation Algorithm:**

1. Start with a random 64-bit value (or derive from previous temporary IID + stored secret)
2. Concatenate with the interface identifier
3. Compute MD5 or SHA-1 hash
4. Take the leftmost 64 bits as the temporary IID
5. Clear bits 6 and 7 (set U/L=0, group/individual=0)
6. Verify the IID is not reserved (e.g., not all zeros, not FFFE pattern)
7. Combine with prefix to form temporary address
8. Run DAD

**Lifetime Management:**

| Parameter | Default | Purpose |
|:---|:---:|:---|
| TEMP_VALID_LIFETIME | 2 days | Total validity of temporary address |
| TEMP_PREFERRED_LIFETIME | 1 day | Duration address is preferred for new connections |
| REGEN_ADVANCE | 5 seconds | Generate new address this far before preferred expires |
| MAX_DESYNC_FACTOR | 10 minutes | Random offset to prevent regeneration synchronization |

The desynchronization factor is critical: without it, all hosts on a subnet would regenerate their temporary addresses simultaneously, causing a DAD storm. Each host picks a random $\delta \in [0, \text{MAX\_DESYNC\_FACTOR}]$ and subtracts it from TEMP_PREFERRED_LIFETIME.

### Stable Semantically-Opaque IIDs (RFC 7217)

A middle ground between EUI-64 (persistent, trackable) and privacy extensions (changing, breaks long-lived connections):

$$\text{IID} = F(\text{Prefix}, \text{Net\_Iface}, \text{Network\_ID}, \text{DAD\_Counter}, \text{Secret\_Key})$$

Where $F$ is a PRF (e.g., SHA-256 truncated to 64 bits). The IID is:
- **Stable** per prefix/interface combination (same address on same network)
- **Different** across networks (different prefix = different IID, no cross-network tracking)
- **Opaque** (no MAC address embedded, no sequential pattern)

This is the default in modern Linux (since ~kernel 4.1+) and Windows 10+.

---

## 5. DHCPv6 Message Flow

### Four-Message Exchange (SARR)

```
Client                              Server
  │                                    │
  │──[Solicit]──→ ff02::1:2           │  (msg type 1)
  │  src: fe80::client                │  Client DUID, IA_NA (address request)
  │  dst: ff02::1:2 (all DHCP)       │  Optional: IA_PD (prefix delegation)
  │                                    │
  │           ←──[Advertise]───────────│  (msg type 2)
  │               Server DUID         │  Offered address/prefix
  │               Preference value    │  Server preference (0-255)
  │                                    │
  │──[Request]──→ Server              │  (msg type 3)
  │  Includes Server DUID             │  Selects specific server
  │  Requested IA_NA/IA_PD            │
  │                                    │
  │           ←──[Reply]───────────────│  (msg type 7)
  │               Assigned address    │  T1, T2 timers
  │               DNS servers         │  Domain search list
  │               Other options        │
  │                                    │
```

### Two-Message Exchange (Information-Request)

For stateless DHCPv6 (RA with M=0, O=1):

```
Client                              Server
  │                                    │
  │──[Information-Request]──→          │  (msg type 11)
  │  No IA_NA, no IA_PD              │  Only requesting config options
  │                                    │
  │           ←──[Reply]───────────────│  (msg type 7)
  │               DNS servers         │
  │               NTP servers          │
  │               Domain search list   │
  │                                    │
```

### Prefix Delegation Flow

The requesting router (CPE/branch router) acts as a DHCPv6 client requesting IA_PD:

```
RR (CPE)                     DR (ISP PE)
  │                              │
  │──[Solicit + IA_PD]──→       │  Request prefix delegation
  │                              │
  │     ←──[Advertise]──────────│  Offer: 2001:db8:c000::/36
  │         IA_PD prefix         │  Valid: 2592000s, Preferred: 604800s
  │                              │
  │──[Request + IA_PD]──→       │  Accept delegation
  │                              │
  │     ←──[Reply]──────────────│  Confirmed: 2001:db8:c000::/36
  │         IA_PD prefix         │  T1=302400s, T2=483840s
  │                              │
```

The RR then subnets the delegated prefix across its downstream interfaces. For a /48 delegation with /64 subnets: $2^{16} = 65,536$ available subnets. For a /56 delegation: $2^{8} = 256$ subnets.

### DUID Types

DHCPv6 identifies clients and servers by DUID (DHCP Unique Identifier), not by MAC address:

| Type | Name | Composition |
|:---|:---|:---|
| DUID-LLT (1) | Link-Layer + Time | Link-layer address + timestamp |
| DUID-EN (2) | Enterprise Number | IANA enterprise number + unique ID |
| DUID-LL (3) | Link-Layer | Link-layer address only |
| DUID-UUID (4) | UUID | RFC 6355, 128-bit UUID |

DUID-LLT is the most common. The timestamp component ensures uniqueness even if a NIC is replaced. DUIDs are per-device, not per-interface — a multi-homed host uses the same DUID on all interfaces.

---

## 6. IPv6 Multicast Addressing Structure

### Address Format

```
|   8    |  4  |  4  |                 112 bits                    |
+--------+-----+-----+---------------------------------------------+
|11111111|flags|scope|                 group ID                     |
+--------+-----+-----+---------------------------------------------+
  ff         T    S

Flags (4 bits): 0RPT
  R = Rendezvous Point address embedded (RFC 3956)
  P = Prefix-based (RFC 3306)
  T = Transient (0=well-known IANA, 1=transient/dynamic)
```

### Solicited-Node Multicast

Every IPv6 unicast and anycast address maps to a solicited-node multicast address:

$$\text{SN-Mcast} = \text{ff02::1:ff} \| \text{last 24 bits of unicast}$$

**Purpose**: NDP Neighbor Solicitation. Instead of broadcasting (as ARP does in IPv4), NS messages are sent to the solicited-node multicast group. Only nodes whose addresses share the same last 24 bits receive the NS. On a typical subnet, each solicited-node group has very few members (usually 1), making it effectively unicast.

**Efficiency analysis**: With $n$ nodes on a subnet, the probability that two nodes share the same solicited-node group is the birthday problem with $2^{24}$ bins:

$$P(\text{collision}) \approx 1 - e^{-n^2/(2 \times 2^{24})}$$

For $n = 100$ nodes: $P \approx 0.03\%$. For $n = 1000$: $P \approx 2.9\%$. Collisions are rare, so the multicast group is almost always a single listener.

### Multicast Listener Discovery (MLD/MLDv2)

MLD is the IPv6 equivalent of IGMP:

| Protocol | IPv4 Equivalent | RFC | Purpose |
|:---|:---|:---|:---|
| MLDv1 | IGMPv2 | RFC 2710 | Basic join/leave |
| MLDv2 | IGMPv3 | RFC 3810 | Source-specific multicast (SSM) |

MLD messages are ICMPv6 (types 130-132 for MLDv1, type 143 for MLDv2) and are link-scoped (Hop Limit = 1, source = link-local). MLD snooping on switches is the IPv6 equivalent of IGMP snooping.

---

## 7. Transition Mechanism Comparison and Selection

### Decision Matrix

| Mechanism | Type | State | IPv4 Needed | Provider Support | Deployment Complexity | Status |
|:---|:---|:---|:---|:---|:---|:---|
| Dual-Stack | Native | N/A | Yes (both stacks) | Full | Low | Preferred |
| NAT64 + DNS64 | Translation | Stateful | IPv4 pool on NAT64 | Single-stack IPv6 | Medium | Recommended |
| 464XLAT | Translation | CLAT stateless, PLAT stateful | PLAT IPv4 pool | IPv6-only access | Medium | Widely deployed (mobile) |
| MAP-T | Translation | Stateless | Shared (port ranges) | IPv6-only infra | High | Growing |
| MAP-E | Encapsulation | Stateless | Shared (port ranges) | IPv6-only infra | High | Growing |
| DS-Lite | Encapsulation | Stateful (AFTR) | AFTR IPv4 pool | IPv6-only access | Medium | Deployed (ISPs) |
| 6rd | Encapsulation | Stateless | Yes (tunnel) | IPv4 infra | Medium | Niche |
| 6to4 | Encapsulation | Stateless | Yes (proto 41) | Anycast relay | Low | **Deprecated** |
| ISATAP | Encapsulation | Stateless | Yes (proto 41) | Intra-site only | Low | **Deprecated** |

### Selection Criteria

**Choose Dual-Stack when:**
- IPv4 addresses are available
- Transition is gradual (years-long)
- Application compatibility is paramount

**Choose NAT64/DNS64 when:**
- Deploying IPv6-only networks (data centers, mobile)
- IPv4 connectivity needed only for legacy destinations
- Willing to maintain NAT64 state and IPv4 address pool

**Choose 464XLAT when:**
- IPv6-only access network
- Must support IPv4-literal applications (legacy apps hardcoding IPv4)
- Mobile/wireless environment (Android, iOS native support)

**Choose MAP-T/MAP-E when:**
- ISP must share IPv4 addresses across subscribers
- Stateless operation required (no per-flow state at border)
- Deterministic port allocation needed for logging/compliance

### NAT64 Address Embedding

The NAT64 prefix embeds the IPv4 address at a position determined by the prefix length:

| Prefix Length | IPv4 Position (bits) | Format Example |
|:---|:---|:---|
| /96 | bits 96-127 | 64:ff9b::**c0a8:0101** |
| /64 | bits 64-95 | 2001:db8:1::**c0a8:01**01:... |
| /56 | bits 56-87 (skip bits 64-71) | 2001:db8:1:**c0**:00**a8:01**01:... |
| /48 | bits 48-79 (skip bits 64-71) | 2001:db8:**c0a8**:00:**0101**:... |
| /40 | bits 40-71 (skip bits 64-71) | 2001:db8:**c0**:00**a8:0101**:... |
| /32 | bits 32-63 (skip bits 64-71) | 2001:db8:**c0a8:0101**:00:... |

Bits 64-71 are always set to zero (the "u" bits) for prefixes shorter than /96. This avoids conflicting with the Interface ID's U/L bit.

---

## 8. NPTv6 Algorithmic Translation

### Translation Algorithm (RFC 6296)

NPTv6 performs a 1:1 stateless mapping between an internal prefix and an external prefix of the same length. The key insight is that IPv6 transport checksums (TCP, UDP, ICMPv6) include a pseudo-header containing source and destination addresses. Changing the prefix would invalidate the checksum — unless the translation is checksum-neutral.

**Checksum Neutrality:**

The algorithm adjusts the Interface Identifier portion of the address to compensate for the prefix change, ensuring that the one's complement sum of the entire address remains constant:

$$\text{sum}(\text{internal\_prefix}) + \text{sum}(\text{internal\_IID}) = \text{sum}(\text{external\_prefix}) + \text{sum}(\text{external\_IID})$$

Therefore:

$$\text{external\_IID} = \text{internal\_IID} - (\text{sum}(\text{external\_prefix}) - \text{sum}(\text{internal\_prefix}))$$

Where subtraction is one's complement (modular arithmetic with end-around carry).

**The adjustment is computed once** for a given prefix pair and applied as a constant offset to a single 16-bit word in the IID. This makes the per-packet operation trivial: add/subtract a precomputed constant from one 16-bit field.

### NPTv6 vs NAT66

NPTv6 is explicitly not NAT:

| Property | NPTv6 | NAT44/NAT66 |
|:---|:---|:---|
| State table | None (stateless) | Per-flow state |
| Port translation | None | Yes (NAPT) |
| Address mapping | 1:1 prefix | Many:1 or Many:Few |
| Bidirectional initiation | Yes | No (requires port forwarding) |
| Scalability | Unlimited (no state) | Limited by state table |
| Breaks | IPsec AH, address-embedding protocols | Same + port-dependent protocols |

### When to Use NPTv6

- **Multi-homing without PI (provider-independent) space**: Internal network uses ULA, NPTv6 translates to each ISP's PA (provider-assigned) prefix. If an ISP changes, only the NPTv6 mapping changes — internal addressing is stable.
- **Compliance with policy requiring stable internal addressing**: ULA provides predictable, non-changing internal addresses while GUA may be renumbered.
- **Not for address conservation**: IPv6 has no scarcity. NPTv6 exists for operational stability, not address sharing.

---

## 9. IPv6 Security Implications and Threat Model

### Attack Surface Differences from IPv4

| Threat | IPv4 | IPv6 |
|:---|:---|:---|
| Scanning | Feasible ($2^{8}$ = 256 hosts per /24) | Infeasible ($2^{64}$ hosts per /64) |
| ARP spoofing | ARP is unauthenticated | NDP is unauthenticated (but SEND exists) |
| Rogue DHCP | DHCP spoofing | Rogue RA (more damaging — controls default route AND addressing) |
| Fragmentation attacks | Common (overlapping fragments) | Mitigated (overlap forbidden, source-only frag) |
| Header manipulation | IP options rarely used | Extension headers create new attack surface |
| Reconnaissance | Port scanning + ping sweep | DNS enumeration, multicast ping, pattern prediction |

### Rogue RA Attack

A rogue RA is more dangerous than a rogue DHCP server because RA controls:
1. Default gateway (Router Lifetime)
2. On-link prefix determination (Prefix Information Option)
3. Address autoconfiguration (A flag, prefix)
4. MTU (MTU Option)
5. DNS (RDNSS Option, RFC 8106)

An attacker sending a crafted RA can:
- Redirect all traffic through their device (MITM via default gateway)
- Assign addresses from an attacker-controlled prefix
- Cause denial of service (zero Router Lifetime removes default route)
- Reduce MTU to force fragmentation (then exploit fragments)

**Mitigations**: RA Guard (RFC 6105), SEND (RFC 3971), RA rate limiting, monitoring for unexpected RAs.

### Scanning in /64 Space

Brute-force scanning a /64 subnet at 1 million packets/second:

$$\text{Time} = \frac{2^{64}}{10^6 \text{ pps}} = 1.8 \times 10^{13} \text{ seconds} \approx 584,942 \text{ years}$$

This makes traditional IPv4-style scanning impractical. Attackers use alternative techniques:
- **DNS zone enumeration** (AXFR if allowed, or brute-force DNS names)
- **Multicast ping**: `ping6 ff02::1%eth0` reveals all link-local addresses on-link
- **Pattern-based guessing**: EUI-64 addresses (predictable from MAC OUI), low-byte addresses (::1, ::2), service-well-known addresses
- **Stable IID prediction**: If the IID generation algorithm is known, reduce the search space
- **Flow metadata**: NetFlow/sFlow records from prior traffic

### SEND (Secure Neighbor Discovery, RFC 3971)

SEND uses CGA (Cryptographically Generated Addresses) and RSA signatures to authenticate NDP messages:

- Router Authorization: RA messages carry certificates from a trust anchor
- CGA: Address owner proves ownership by demonstrating possession of the private key that generated the address
- Timestamp + Nonce: Prevents replay attacks

**Why SEND is rarely deployed:**
- Requires PKI infrastructure (certificate management)
- CGA address generation is computationally expensive
- Not supported by most commodity operating systems
- RA Guard provides 80% of the benefit with 10% of the complexity

---

## Prerequisites

- IPv6 fundamentals (addressing, prefix notation, basic NDP), TCP/IP protocol stack, binary and hexadecimal arithmetic, one's complement checksum, basic cryptography (hashing, public-key concepts), routing protocol concepts (OSPF, BGP, EIGRP)

---

## References

- [RFC 8200 — Internet Protocol, Version 6 (IPv6) Specification](https://www.rfc-editor.org/rfc/rfc8200)
- [RFC 4861 — Neighbor Discovery for IP version 6](https://www.rfc-editor.org/rfc/rfc4861)
- [RFC 4862 — IPv6 Stateless Address Autoconfiguration](https://www.rfc-editor.org/rfc/rfc4862)
- [RFC 8981 — Temporary Address Extensions for Stateless Autoconfiguration](https://www.rfc-editor.org/rfc/rfc8981)
- [RFC 7217 — A Method for Generating Semantically Opaque Interface Identifiers](https://www.rfc-editor.org/rfc/rfc7217)
- [RFC 8415 — Dynamic Host Configuration Protocol for IPv6 (DHCPv6)](https://www.rfc-editor.org/rfc/rfc8415)
- [RFC 6296 — IPv6-to-IPv6 Network Prefix Translation (NPTv6)](https://www.rfc-editor.org/rfc/rfc6296)
- [RFC 6146 — Stateful NAT64](https://www.rfc-editor.org/rfc/rfc6146)
- [RFC 6147 — DNS64](https://www.rfc-editor.org/rfc/rfc6147)
- [RFC 6877 — 464XLAT](https://www.rfc-editor.org/rfc/rfc6877)
- [RFC 7597 — Mapping of Address and Port with Encapsulation (MAP-E)](https://www.rfc-editor.org/rfc/rfc7597)
- [RFC 7599 — Mapping of Address and Port using Translation (MAP-T)](https://www.rfc-editor.org/rfc/rfc7599)
- [RFC 7045 — Transmission and Processing of IPv6 Extension Headers](https://www.rfc-editor.org/rfc/rfc7045)
- [RFC 5722 — Handling of Overlapping IPv6 Fragments](https://www.rfc-editor.org/rfc/rfc5722)
- [RFC 3971 — SEcure Neighbor Discovery (SEND)](https://www.rfc-editor.org/rfc/rfc3971)
- [RFC 6105 — IPv6 Router Advertisement Guard](https://www.rfc-editor.org/rfc/rfc6105)
- [RFC 7526 — Deprecating the Anycast Prefix for 6to4 Relay Routers](https://www.rfc-editor.org/rfc/rfc7526)
- [RFC 5969 — IPv6 Rapid Deployment on IPv4 Infrastructures (6rd)](https://www.rfc-editor.org/rfc/rfc5969)
- [RFC 3306 — Unicast-Prefix-based IPv6 Multicast Addresses](https://www.rfc-editor.org/rfc/rfc3306)
- [RFC 3810 — Multicast Listener Discovery Version 2 (MLDv2)](https://www.rfc-editor.org/rfc/rfc3810)
