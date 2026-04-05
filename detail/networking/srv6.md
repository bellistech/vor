# SRv6 Deep Dive — SID Encoding, Network Programming, and Overhead Analysis

> *SRv6 collapses the MPLS control and data planes into native IPv6, encoding forwarding instructions as IPv6 addresses. Every SID is simultaneously a routable locator and a behavioral instruction, enabling a network programming paradigm where the segment list is a program and each SID is a function call. The cost is overhead — 16 bytes per segment versus 4 bytes in SR-MPLS — which micro-SID compression addresses by packing multiple instructions into a single 128-bit container.*

---

## 1. SRv6 SID Structure and Encoding

### Anatomy of a 128-Bit SID

An SRv6 SID is a standard IPv6 address partitioned into three semantic fields. The partition sizes are not fixed by the protocol but are determined by the locator prefix length and the operator's design:

```
|<------- Locator (B bits) ------->|<-- Function (F bits) -->|<-- Args (A bits) -->|
|<------------------------------- 128 bits ---------------------------------------->|

B + F + A = 128
```

**Locator (B bits):** The routable prefix, advertised in the IGP (IS-IS SRv6 extensions or OSPFv3). When a packet arrives with a destination address matching a local locator, the node inspects the function field to determine the behavior. The locator serves dual purpose: it provides IPv6 reachability (any node can forward toward it via standard FIB lookup) and it identifies the owning node.

**Function (F bits):** Identifies the specific behavior to execute. Each function ID maps to a programmed action (END, END.DT4, END.X, etc.). The function is locally significant — the same function value can mean different things on different nodes.

**Arguments (A bits):** Optional parameters consumed by the function. Uses include:
- Flow identification for per-flow load balancing
- VRF identifier for dynamic VRF selection
- Interface index for specific interface steering
- Service instance for service function chaining

### Encoding Examples

| Locator | Function | Args | Full SID | Behavior |
|:---|:---|:---|:---|:---|
| fc00:0:1::/48 | 0x0001 | :: | fc00:0:1:1:: | END (transit) |
| fc00:0:1::/48 | 0xE001 | :: | fc00:0:1:E001:: | END.X (adj cross-connect) |
| fc00:0:1::/48 | 0xD100 | :: | fc00:0:1:D100:: | END.DT4 (VRF IPv4 lookup) |
| fc00:0:1::/48 | 0xD200 | :: | fc00:0:1:D200:: | END.DT6 (VRF IPv6 lookup) |
| fc00:0:1::/48 | 0xD300 | :: | fc00:0:1:D300:: | END.DT46 (dual-stack VRF) |
| fc00:0:1::/48 | 0xA100 | :: | fc00:0:1:A100:: | END.DX2 (L2 cross-connect) |

### Function Space Capacity

With a /48 locator, the function field occupies bits 48-63 (16 bits):

$$\text{Functions per node} = 2^{16} = 65{,}536$$

This provides ample space for:
- VRF SIDs: one per VRF per address family (END.DT4, END.DT6)
- Adjacency SIDs: one per interface per neighbor (END.X)
- Service SIDs: one per service function instance
- Flex-algo SIDs: one per algorithm per behavior

For very large nodes (data center leaf with thousands of VRFs), a /40 locator provides $2^{24} = 16{,}777{,}216$ function IDs.

---

## 2. Network Programming Paradigm

### SID as Function Call

SRv6 network programming (RFC 8986) models the segment list as a program executed by the network:

```
Program:     [SID_1, SID_2, ..., SID_n]
Execution:   Left to right (SID_n processed first, SID_1 last)
Instruction: Each SID = function(locator, behavior, arguments)
State:       Segments Left (SL) register = program counter
```

The analogy to function calls:

| Programming Concept | SRv6 Equivalent |
|:---|:---|
| Function name | SID (128-bit address) |
| Function body | Behavior code (END, END.DT4, etc.) |
| Arguments | SID argument field |
| Program counter | Segments Left (SL) |
| Instruction memory | Segment list in SRH |
| Return value | Modified packet (decapsulated, re-encapsulated, etc.) |

### Behavior Taxonomy

Behaviors are classified by their action on the packet:

**Transit behaviors (no decapsulation):**
- END: FIB lookup on next segment (basic forwarding)
- END.X: Forward via specific adjacency (traffic engineering)
- END.T: Lookup in a specific IPv6 table (multi-topology)

**Decapsulation behaviors (terminate tunnel):**
- END.DT4: Decaps, IPv4 lookup in VRF (L3VPN egress)
- END.DT6: Decaps, IPv6 lookup in VRF
- END.DT46: Decaps, IPv4 or IPv6 lookup in VRF (dual-stack)
- END.DX4: Decaps, IPv4 cross-connect to specific next-hop
- END.DX6: Decaps, IPv6 cross-connect to specific next-hop
- END.DX2: Decaps, L2 cross-connect (VPWS)

**Binding behaviors (policy stitching):**
- END.B6.Encaps: Apply bound SRv6 policy (push new SRH)
- END.B6.Encaps.Red: Reduced encapsulation variant
- END.BM: Bind to SR-MPLS policy (SRv6-to-MPLS interworking)

### PSP, USP, and USD Flavors

Each endpoint behavior can have optional flavors that modify how the SRH is handled when the segment is the penultimate or ultimate segment:

**PSP (Penultimate Segment Pop):**
When SL reaches 1, the penultimate node:
1. Decrements SL to 0
2. Copies Segment List[0] to DA
3. Removes the SRH from the packet
4. Forwards a clean IPv6 packet (no SRH)

This is analogous to PHP in MPLS. The egress node receives a plain IPv6 packet and does not need to process an SRH.

**USP (Ultimate Segment Pop):**
The ultimate (last) node removes the SRH after processing. This is the default behavior when PSP is not used.

**USD (Ultimate Segment Decapsulation):**
The ultimate node decapsulates the outer IPv6 header entirely, exposing the inner packet. Used with END.DT4, END.DT6, and similar decapsulation behaviors.

These flavors are encoded in the locator advertisement and affect forwarding behavior at the penultimate and ultimate hops.

---

## 3. SRH Format Analysis (RFC 8754)

### Header Fields

The Segment Routing Header is an IPv6 Routing Extension Header with Routing Type 4:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Next Header   | Hdr Ext Len   | Routing Type=4| Segments Left |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Last Entry    |    Flags      |        Tag                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[0] (128 bits) — last segment          |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                         ...                                   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Segment List[n] (128 bits) — first segment         |
|                                                               |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|            Optional TLVs (variable length, padded)            |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**Field semantics:**

| Field | Size | Description |
|:---|:---|:---|
| Next Header | 8 bits | Type of the next header (e.g., 41 for IPv6-in-IPv6, 4 for IPv4-in-IPv6) |
| Hdr Ext Len | 8 bits | Length of SRH in 8-octet units, not including the first 8 octets |
| Routing Type | 8 bits | Always 4 for SRH |
| Segments Left | 8 bits | Index into segment list of the current active segment |
| Last Entry | 8 bits | Index of the last entry in the segment list (0-based) |
| Flags | 8 bits | Currently unused (must be 0) |
| Tag | 16 bits | Used for packet classification or marking (implementation-specific) |

### Segment List Ordering (Critical Detail)

The segment list is stored in **reverse order** relative to the forwarding path:

```
Desired path: A -> B -> C -> D (D is the final destination)

Segment list in SRH:
  Segment List[0] = SID_D     (last hop, final destination)
  Segment List[1] = SID_C
  Segment List[2] = SID_B     (first hop after headend)

SL initialized to: 2 (= Last Entry)

At headend:
  DA = Segment List[2] = SID_B  (first segment)
  SL = 2

At node B:
  Process SID_B -> decrement SL to 1
  DA = Segment List[1] = SID_C
  Forward toward SID_C

At node C:
  Process SID_C -> decrement SL to 0
  DA = Segment List[0] = SID_D
  Forward toward SID_D

At node D:
  Process SID_D (SL = 0, last segment)
  Execute terminal behavior (e.g., END.DT4 -> decapsulate)
```

### Hdr Ext Len Calculation

$$\text{Hdr Ext Len} = \frac{\text{SRH size} - 8}{8} = \frac{8 + 16N + T - 8}{8} = 2N + \lceil T/8 \rceil$$

where $N$ is the number of segments and $T$ is the total TLV size (padded to 8-octet boundary).

For $N = 3$ segments, no TLVs:

$$\text{Hdr Ext Len} = 2 \times 3 = 6$$

$$\text{Actual SRH size} = (6 + 1) \times 8 = 56 \text{ bytes}$$

Verification: $8 + 16 \times 3 = 56$ bytes. Correct.

---

## 4. SRv6 Forwarding with Segments Left

### Complete Forwarding Walk-Through

Consider a 4-node path with SRv6 L3VPN:

```
CE1 --[IPv4]--> PE1 --[SRv6]--> P1 --[SRv6]--> P2 --[SRv6]--> PE2 --[IPv4]--> CE2

PE1 locator: fc00:0:1::/48
P1  locator: fc00:0:2::/48
P2  locator: fc00:0:3::/48
PE2 locator: fc00:0:4::/48

Segment list: [fc00:0:2:1::, fc00:0:3:1::, fc00:0:4:D100::]
              (P1 END,       P2 END,       PE2 END.DT4)
```

**Step 1: PE1 (Headend — H.Encaps)**

```
Input:  IPv4 packet from CE1 (src=10.1.1.10, dst=10.2.2.20)

Action: H.Encaps
  1. Push outer IPv6 header:
     Src = fc00:0:1::1 (PE1 loopback)
     Dst = fc00:0:3:1:: (= Segment List[2], first segment)
  2. Insert SRH:
     Segments Left = 2
     Last Entry = 2
     Segment List[0] = fc00:0:4:D100::  (PE2 END.DT4)
     Segment List[1] = fc00:0:3:1::     (P2 END)
     Segment List[2] = fc00:0:2:1::     (P1 END)
  3. Forward: FIB lookup on fc00:0:2:1:: -> toward P1

Output packet:
  [IPv6: src=PE1, dst=fc00:0:2:1::][SRH: SL=2][IPv4: 10.1.1.10->10.2.2.20]
```

**Step 2: P1 (Transit — END)**

```
Input:  IPv6 DA = fc00:0:2:1:: (matches local SID -> END behavior)

Action: END
  1. SL = SL - 1 = 1
  2. DA = Segment List[1] = fc00:0:3:1::
  3. FIB lookup on fc00:0:3:1:: -> toward P2

Output packet:
  [IPv6: src=PE1, dst=fc00:0:3:1::][SRH: SL=1][IPv4: 10.1.1.10->10.2.2.20]
```

**Step 3: P2 (Transit — END with PSP)**

```
Input:  IPv6 DA = fc00:0:3:1:: (matches local SID -> END with PSP)

Action: END (PSP flavor, since SL will become 0 after decrement)
  1. SL = SL - 1 = 0
  2. DA = Segment List[0] = fc00:0:4:D100::
  3. PSP: Remove the SRH (penultimate segment pop)
  4. FIB lookup on fc00:0:4:D100:: -> toward PE2

Output packet:
  [IPv6: src=PE1, dst=fc00:0:4:D100::][IPv4: 10.1.1.10->10.2.2.20]
  (No SRH — removed by PSP)
```

**Step 4: PE2 (Egress — END.DT4)**

```
Input:  IPv6 DA = fc00:0:4:D100:: (matches local SID -> END.DT4)

Action: END.DT4
  1. Remove outer IPv6 header
  2. Extract inner IPv4 packet
  3. IPv4 FIB lookup in VRF CUST-A for 10.2.2.20
  4. Forward to CE2

Output packet:
  [IPv4: src=10.1.1.10, dst=10.2.2.20]
```

### SL State Machine

$$SL_{initial} = N - 1$$

where $N$ is the number of segments. At each processing node:

$$SL_{next} = SL_{current} - 1$$
$$DA_{next} = \text{Segment List}[SL_{next}]$$

Processing terminates when the behavior is a decapsulation function (END.DT4, END.DT6, END.DX2, etc.) and $SL = 0$.

---

## 5. Micro-SID (uSID) Compression Theory

### The Overhead Problem

Standard SRv6 overhead compared to SR-MPLS:

| Segments | SR-MPLS Overhead | SRv6 Overhead | SRv6/SR-MPLS Ratio |
|:---:|:---:|:---:|:---:|
| 1 | 4 bytes | 24 bytes (8 + 16) | 6.0x |
| 2 | 8 bytes | 40 bytes (8 + 32) | 5.0x |
| 3 | 12 bytes | 56 bytes (8 + 48) | 4.7x |
| 5 | 20 bytes | 88 bytes (8 + 80) | 4.4x |
| 10 | 40 bytes | 168 bytes (8 + 160) | 4.2x |

For a 1500-byte MTU, 10 SRv6 segments consume 11.2% of the packet, versus 2.7% for SR-MPLS. This is a significant concern for latency-sensitive and bandwidth-constrained environments.

### uSID Encoding

Micro-SID addresses the overhead by packing multiple micro-instructions into a single 128-bit container:

```
Standard SRv6 SID:
  |<------- Locator (48 bits) ------->|<-- Function (16b) -->|<-- Args (64b) -->|
  1 SID = 1 instruction = 16 bytes

Micro-SID container (16-bit uSID variant):
  |<- Block (32b) ->|<- uSID1 (16b) ->|<- uSID2 ->|<- uSID3 ->|<- uSID4 ->|<- uSID5 ->|<- uSID6 ->|
  |<----------------------------------- 128 bits ------------------------------------------>|
  1 container = up to 6 instructions = 16 bytes

Micro-SID container (32-bit uSID variant):
  |<- Block (32b) ->|<-- uSID1 (32b) -->|<-- uSID2 -->|<-- uSID3 -->|
  |<--------------------------- 128 bits -------------------------------->|
  1 container = up to 3 instructions = 16 bytes
```

### uSID Compression Ratio

For $N$ micro-SIDs with 16-bit uSID encoding:

$$\text{Containers needed} = \left\lceil \frac{N}{6} \right\rceil$$

$$\text{uSID overhead} = 8 + 16 \times \left\lceil \frac{N}{6} \right\rceil \text{ bytes}$$

Comparison:

| Segments | SR-MPLS | Standard SRv6 | uSID (16-bit) | uSID Saving vs SRv6 |
|:---:|:---:|:---:|:---:|:---:|
| 1 | 4B | 24B | 24B (8+16) | 0% |
| 3 | 12B | 56B | 24B (8+16, 1 container) | 57% |
| 6 | 24B | 104B | 24B (8+16, 1 container) | 77% |
| 7 | 28B | 120B | 40B (8+32, 2 containers) | 67% |
| 12 | 48B | 200B | 40B (8+32, 2 containers) | 80% |

For 6 segments, uSID achieves parity with SR-MPLS (24B vs 24B). Beyond 6 segments, uSID is actually more efficient than SR-MPLS.

### uSID Forwarding Mechanics

At each uSID-aware node, the processing is a **shift operation**:

```
Initial container: [Block:uSID1:uSID2:uSID3:uSID4:uSID5:uSID6]
DA = Block:uSID1:... (first uSID identifies this node)

Processing at uSID1's node:
  1. Identify Block:uSID1 as a local micro-SID
  2. Execute the behavior associated with uSID1
  3. Shift left: remove uSID1, shift remaining uSIDs left, pad with zeros
  4. New DA = [Block:uSID2:uSID3:uSID4:uSID5:uSID6:0000]
  5. Forward based on new DA

When all uSIDs in a container are consumed (DA = Block:0000:0000:...):
  - Decrement SL
  - Move to the next container in the segment list
  - Copy Segment List[SL] to DA
```

The shift operation means transit nodes only need to match on `Block + first_uSID` (48 bits with 32-bit block + 16-bit uSID), which fits standard TCAM/LPM prefix matching.

### uN vs uA Behaviors

**uN (micro-Node):** The micro-SID equivalent of END. The node identifies itself, shifts the uSID, and forwards via FIB lookup to the next uSID's node.

**uA (micro-Adjacency):** The micro-SID equivalent of END.X. The node forwards to a specific adjacency rather than performing a FIB lookup. Used for micro-SID traffic engineering.

**uDT4, uDT6, uDX4, uDX6:** Decapsulation micro-behaviors equivalent to their full SRv6 counterparts.

---

## 6. SRv6 Security Considerations

### Threat Model

SRv6 inherits IPv6 security properties plus introduces SRH-specific threats:

| Threat | Description | Mitigation |
|:---|:---|:---|
| SRH injection | External host crafts packets with SRH to steer through the network | ACL at network edge: drop packets with Routing Type 4 from external sources |
| SID spoofing | Attacker crafts packets with DA matching a valid SID | Locators should use non-globally-routable prefixes (ULA fc00::/7 or provider PI) |
| SRH manipulation | Man-in-the-middle modifies segment list | HMAC TLV in SRH (RFC 8754 Section 2.1.2) provides integrity |
| Reconnaissance | Probing for valid SIDs in the network | Locator prefixes should not be advertised externally |
| Amplification | Using END.B6.Encaps to amplify by adding more segments | Rate-limit and restrict B6 SID to trusted sources |

### Infrastructure ACL Pattern

```
The critical security rule for SRv6:

  Permit SRH (Routing Type 4) ONLY from within the SRv6 domain.
  Drop SRH from external (untrusted) interfaces.

Implementation:
  - On all PE external interfaces: match and drop IPv6 packets with
    Routing Header Type 4 (SRH)
  - On all P-P and PE-P internal interfaces: permit SRH
  - Locator prefixes should not be in the global IPv6 DFZ BGP table

IOS-XR example:
  ipv6 access-list SRV6-EDGE-IN
   deny ipv6 any any routing-type 4
   permit ipv6 any any

  interface GigabitEthernet0/0/0/0   ! CE-facing
   ipv6 access-group SRV6-EDGE-IN ingress
```

### HMAC TLV for SRH Integrity

RFC 8754 defines an optional HMAC TLV that provides cryptographic integrity for the SRH:

```
HMAC TLV format:
  Type:     5
  Length:   38 (fixed)
  D-flag:   1 bit (reserved)
  HMAC Key ID: 32 bits (identifies the pre-shared key)
  HMAC:     256 bits (SHA-256 truncated)

The HMAC is computed over:
  - Source address
  - Last Entry
  - Flags
  - Segment List (all entries)
  - HMAC Key ID

This prevents segment list tampering by intermediate nodes or attackers.
```

The HMAC TLV adds 40 bytes to the SRH. It is rarely deployed in practice due to the overhead and the availability of simpler mitigations (infrastructure ACLs, non-routable locators).

---

## 7. SRv6 Overhead Analysis vs MPLS

### Per-Packet Overhead Comparison

For a typical L3VPN scenario with $N$ transit hops requiring TE:

**SR-MPLS:**

$$O_{SR-MPLS} = 4N \text{ bytes}$$

**SRv6 (full SIDs):**

$$O_{SRv6} = 40 + 8 + 16N = 48 + 16N \text{ bytes}$$

The 40 bytes is the outer IPv6 header. The 8 bytes is the SRH fixed header.

**SRv6 (with uSID, 16-bit):**

$$O_{uSID} = 40 + 8 + 16\left\lceil \frac{N}{6} \right\rceil = 48 + 16\left\lceil \frac{N}{6} \right\rceil \text{ bytes}$$

**SRv6 (no SRH, single segment):**

When only one segment is needed (e.g., direct L3VPN with no TE), many implementations omit the SRH entirely:

$$O_{SRv6\_single} = 40 \text{ bytes (outer IPv6 header only)}$$

Compare to MPLS L3VPN (2-label stack): $O_{MPLS\_VPN} = 8$ bytes.

### Throughput Impact

For 64-byte packets (worst case, common in VoIP):

| Encapsulation | Overhead | Effective Payload | Overhead % |
|:---|:---:|:---:|:---:|
| SR-MPLS (3 labels) | 12B | 52B/64B = 81% | 19% |
| SRv6 (3 segments) | 104B | 64B/168B = 38% | 62% |
| SRv6 uSID (3 segments) | 64B | 64B/128B = 50% | 50% |
| SRv6 (1 segment, no SRH) | 40B | 64B/104B = 62% | 38% |

For 1500-byte packets (typical):

| Encapsulation | Overhead | Effective Payload | Overhead % |
|:---|:---:|:---:|:---:|
| SR-MPLS (3 labels) | 12B | 99% | 0.8% |
| SRv6 (3 segments) | 104B | 94% | 6.5% |
| SRv6 uSID (3 segments) | 64B | 96% | 4.1% |
| SRv6 (1 segment, no SRH) | 40B | 97% | 2.6% |

### MTU Implications

With a physical MTU of 1500 bytes and SRv6 encapsulation, the effective inner MTU is:

$$MTU_{inner} = MTU_{phys} - O_{SRv6}$$

For 5 segments:

$$MTU_{inner} = 1500 - (40 + 8 + 80) = 1372 \text{ bytes}$$

This is below the typical internet MTU assumption (1400-1500 bytes), which forces either:
1. Jumbo frames (9000+ MTU) on all provider links
2. Path MTU Discovery (PMTUD) to negotiate smaller packet sizes
3. Fragmentation at the headend (expensive)

For uSID with 5 segments (fits in 1 container):

$$MTU_{inner} = 1500 - (40 + 8 + 16) = 1436 \text{ bytes}$$

Significantly better, but still below 1500. Jumbo frames remain the recommendation.

---

## 8. SRv6 Interworking

### SRv6 to SR-MPLS Gateway

When an SRv6 domain connects to an SR-MPLS domain, a gateway node performs translation:

**SRv6-to-MPLS (END.BM behavior):**

```
SRv6 domain        Gateway          SR-MPLS domain
  PE1 ----[SRH]----> GW ---[MPLS]----> PE2

Gateway processing (END.BM):
  1. Receive SRv6 packet with DA matching END.BM SID
  2. Pop SRv6 encapsulation (outer IPv6 + SRH)
  3. Push MPLS label stack corresponding to the bound SR-MPLS policy
  4. Forward as MPLS packet

The END.BM SID binds to a specific SR-MPLS policy:
  SID fc00:0:10:BM01:: -> SR-MPLS policy {16002, 16005}
```

**MPLS-to-SRv6 (H.Encaps at gateway):**

```
SR-MPLS domain      Gateway          SRv6 domain
  PE1 ---[MPLS]----> GW ---[SRH]----> PE2

Gateway processing:
  1. Receive MPLS packet, pop labels
  2. Apply H.Encaps: push outer IPv6 header + SRH
  3. SRH contains the SRv6 segment list for the SRv6 domain
  4. Forward as IPv6/SRv6 packet
```

### SRv6 and IPv4/IPv6 Coexistence

SRv6 requires IPv6 forwarding on all provider links. For networks with mixed IPv4/IPv6 infrastructure:

**Option 1: IPv6 overlay on IPv4 underlay**
- Tunnels (GRE, VXLAN) carry IPv6 across IPv4 segments
- Adds encapsulation overhead on top of SRv6 overhead
- Not recommended (double encapsulation)

**Option 2: Dual-stack provider core**
- All links carry both IPv4 and IPv6
- SR-MPLS operates on IPv4, SRv6 operates on IPv6
- Gradual migration: start SRv6 for new services, keep SR-MPLS for existing

**Option 3: IPv6-only provider core**
- Full migration to IPv6 on all provider links
- IPv4 services (customer IPv4 traffic) carried via END.DT4/END.DX4
- Cleanest architecture but requires complete IPv6 deployment

---

## 9. SRv6 for Service Chaining

### Service Function Chaining with SRv6

SRv6 naturally supports service chaining by encoding service functions as SIDs in the segment list:

```
Traffic path: Client -> Firewall -> IDS -> Load Balancer -> Server

Segment list:
  [SID_FW, SID_IDS, SID_LB, SID_Server]

Each service function is a SID endpoint:
  SID_FW:     fc00:0:10:SF01::   (END.DX2 to firewall VM interface)
  SID_IDS:    fc00:0:20:SF02::   (END.DX2 to IDS appliance)
  SID_LB:     fc00:0:30:SF03::   (END.DX4 to load balancer)
  SID_Server: fc00:0:40:D100::   (END.DT4 to server VRF)

The packet visits each service in order, steered by the SRH segment list.
No NSH (Network Service Header) needed — SRv6 SIDs encode the chain.
```

### Advantages Over NSH-Based Chaining

| Aspect | NSH (RFC 8300) | SRv6 Service Chaining |
|:---|:---|:---|
| Header | NSH (8-byte base + metadata) | SRH (reuses IPv6 extension header) |
| Metadata | NSH MD-type 1 (16B) or MD-type 2 (variable) | SRH TLVs or SID arguments |
| Network integration | Requires NSH-aware fabric | Native IPv6 forwarding |
| Service proxy | Needed at each SF if not NSH-aware | SIDs routed natively; SF sees standard packets |
| Path steering | Service Function Forwarder (SFF) | Standard IPv6 FIB + SRH processing |
| Topology independence | Requires SFC overlay | Inherent in IPv6 routing |

### Service-Aware SID Encoding

The SID argument field can carry service metadata:

```
SID: fc00:0:10:SF01:VLAN100::

Locator:   fc00:0:10::/48     (service node)
Function:  0xSF01             (service function 1)
Argument:  VLAN100            (service VLAN tag or tenant ID)

The service node extracts the argument field and uses it to:
  - Select the correct service VM/container
  - Apply tenant-specific policy
  - Tag traffic for the service function
```

---

## 10. Summary of Key Formulas

| Formula | Description |
|:---|:---|
| $O_{SRv6} = 48 + 16N$ | Full SRv6 overhead for N segments (bytes) |
| $O_{uSID} = 48 + 16\lceil N/6 \rceil$ | uSID overhead for N micro-segments (bytes) |
| $O_{SR-MPLS} = 4N$ | SR-MPLS overhead for N segments (bytes) |
| $\text{Functions/node} = 2^{128-B}$ | SID function space with B-bit locator |
| $SL_{init} = N - 1$ | Initial Segments Left value for N segments |
| $DA_{next} = \text{SegList}[SL - 1]$ | Next destination address after processing |
| $\text{Hdr Ext Len} = 2N + \lceil T/8 \rceil$ | SRH header extension length field |
| $MTU_{inner} = MTU_{phys} - O_{SRv6}$ | Effective inner MTU with SRv6 encap |

## Prerequisites

- ipv6 (IPv6 addressing, extension headers, NDP), segment-routing (SR architecture, prefix SID, adjacency SID, flex-algo), mpls (label switching, label stack, PHP), bgp (VPNv4/v6, EVPN address families), is-is (IGP fundamentals, TLV extensions)

---

*SRv6 transforms IPv6 from a transport protocol into a programmable network substrate. Each SID is simultaneously a routable address and a behavioral instruction, eliminating the separation between addressing and forwarding that has defined networking since the OSI model. The overhead cost — inherent in the 128-bit SID width — is the price of this programmability. Micro-SID compression makes the trade-off viable by approaching MPLS-level efficiency while retaining the full network programming model. As hardware catches up (native SRH processing in merchant silicon) and IPv6 deployment reaches critical mass, SRv6 is positioned to subsume MPLS as the unified transport for VPN, TE, and service chaining — not because it is more efficient, but because it is more expressive.*
