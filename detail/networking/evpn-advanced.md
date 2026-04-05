# The Mathematics of Advanced EVPN — NLRI, DF Election, and Forwarding Analysis

> *EVPN replaces flood-and-learn with a BGP control plane for Layer 2/3 services. Understanding the NLRI encoding, DF election algorithms, MAC mobility mechanics, and the mathematical differences between symmetric and asymmetric IRB is fundamental to designing scalable EVPN fabrics.*

---

## 1. EVPN NLRI Format

### BGP EVPN Address Family

EVPN uses MP-BGP with:
- **AFI:** 25 (L2VPN)
- **SAFI:** 70 (EVPN)

### NLRI Encoding

Every EVPN NLRI begins with:

```
+--------+--------+
| Route  | Length  |
| Type   | (bytes)|
| (1 B)  | (1 B)  |
+--------+--------+
| Route Type Value |
| (variable)       |
+------------------+
```

The Route Type field determines the structure of the remaining value.

### Type 2 NLRI Deep Dive (MAC/IP)

The most common route type. Full encoding:

```
Bytes:  8       10       4       1      6      1     0/4/16   3     0/3
     +------+-------+------+------+-----+------+------+-----+------+
     |  RD  |  ESI  | ETag | MACl | MAC | IPl  |  IP  | L1  |  L2  |
     +------+-------+------+------+-----+------+------+-----+------+
```

- **RD (8 bytes):** Route Distinguisher — makes routes unique per PE
- **ESI (10 bytes):** Ethernet Segment Identifier — 0 for single-homed
- **Ethernet Tag (4 bytes):** VLAN ID or 0 for VLAN-based service
- **MAC Length (1 byte):** Always 48 (bits)
- **MAC Address (6 bytes):** The advertised MAC
- **IP Length (1 byte):** 0 (no IP), 32 (IPv4), or 128 (IPv6)
- **IP Address (0/4/16 bytes):** Bound IP address
- **Label 1 (3 bytes):** L2 VNI or MPLS label for bridging
- **Label 2 (0/3 bytes):** L3 VNI or MPLS label for routing (symmetric IRB)

**Total NLRI length:** 33 bytes (MAC only, no IP, one label) to 54 bytes (MAC + IPv6, two labels).

### NLRI Size Analysis

Per-route BGP UPDATE overhead:

| Component | Bytes |
|:---|:---:|
| BGP header | 19 |
| UPDATE fixed fields | 4 |
| Path attributes (AS-path, next-hop, extended communities) | ~60-80 |
| EVPN NLRI (Type 2, MAC+IPv4, 2 labels) | 44 |
| **Total per UPDATE** | **~130-150** |

For a fabric with $M$ MACs across $P$ PEs, each RR stores:

$$\text{RR memory} = M \times P_{advertising} \times S_{entry}$$

Where $S_{entry}$ is the per-path RIB entry size (~500-1000 bytes including path attributes).

Example: 100,000 MACs, 50 PEs, 750 bytes per entry:

$$\text{RR memory} = 100,000 \times 750 = 75 \text{ MB}$$

(Each MAC is advertised by one PE, so total is $M \times S_{entry}$, not $M \times P$.)

---

## 2. BGP Extended Communities for EVPN

### Route Target (Type 0x00/0x02, Sub-type 0x02)

Controls route distribution:

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| 0x00   | 0x02   |     ASN (2B)    |    Assigned Number (4B)           |
+--------+--------+--------+--------+--------+--------+--------+--------+
```

Or 4-byte ASN format:

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| 0x02   | 0x02   |       ASN (4B)            |  Assigned Number (2B)   |
+--------+--------+--------+--------+--------+--------+--------+--------+
```

### MAC Mobility (Type 0x06, Sub-type 0x00)

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| 0x06   | 0x00   | Flags  | Rsvd   |       Sequence Number (4B)        |
+--------+--------+--------+--------+--------+--------+--------+--------+

Flags: Bit 0 = Sticky (static MAC, cannot move)
```

### ESI Label (Type 0x06, Sub-type 0x01)

Used for split-horizon filtering:

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| 0x06   | 0x01   | Flags  | Rsvd   |        ESI Label (3B)             |
+--------+--------+--------+--------+--------+--------+--------+--------+

Flags: Bit 0 = Single-Active
```

### ES-Import Route Target (Type 0x06, Sub-type 0x02)

Allows efficient Type 4 route import based on ESI:

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| 0x06   | 0x02   |          ES-Import Value (6B, from ESI bytes 1-6)   |
+--------+--------+--------+--------+--------+--------+--------+--------+
```

### Router's MAC (Type 0x06, Sub-type 0x03)

Carries the router MAC for symmetric IRB:

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| 0x06   | 0x03   |              Router MAC Address (6B)                |
+--------+--------+--------+--------+--------+--------+--------+--------+
```

### Default Gateway (Type 0x06, Sub-type 0x0D)

Indicates the MAC is a default gateway (for anycast gateway):

```
+--------+--------+--------+--------+--------+--------+--------+--------+
| 0x06   | 0x0D   |              Reserved (6B, all zeros)               |
+--------+--------+--------+--------+--------+--------+--------+--------+
```

---

## 3. DF Election Algorithms

### The Problem

When multiple PEs are attached to the same Ethernet Segment, only one PE (the Designated Forwarder) should forward BUM traffic to the CE to prevent duplicates. DF election must be:
- Deterministic (all PEs agree)
- Even (load balanced across PEs)
- Fast (converges quickly on PE failure)

### Mod-Based Election (Default, RFC 7432)

The simplest algorithm. For VLAN $V$ with $N$ PEs ordered by IP address:

$$DF_{index} = V \bmod N$$

The PE at position $DF_{index}$ (0-indexed) in the sorted PE list is the DF for VLAN $V$.

**Example:** PEs with IPs [10.0.0.1, 10.0.0.2, 10.0.0.3], $N = 3$:

| VLAN | $V \bmod 3$ | DF PE |
|:---:|:---:|:---:|
| 1 | 1 | 10.0.0.2 |
| 2 | 2 | 10.0.0.3 |
| 3 | 0 | 10.0.0.1 |
| 4 | 1 | 10.0.0.2 |
| 5 | 2 | 10.0.0.3 |
| 6 | 0 | 10.0.0.1 |

**Properties:**
- Distribution: perfectly even when $V_{max}$ is a multiple of $N$
- On PE failure: all VLANs must be redistributed ($\frac{V}{N}$ VLANs move per failure)
- Computation: $O(1)$ per VLAN

**Weakness:** When a PE fails, $\frac{1}{N}$ of VLANs need DF reassignment across the remaining $N-1$ PEs. The redistribution is bursty — all affected VLANs switch simultaneously.

### HRW (Highest Random Weight) Election (RFC 8584)

Uses a hash function to distribute VLANs more evenly and with better failure properties:

$$DF = \arg\max_{i \in PEs} H(V, IP_i)$$

Where $H$ is a hash function (typically CRC32 or similar) applied to the VLAN ID concatenated with the PE IP.

**Properties:**
- Distribution: statistically even (depends on hash function quality)
- On PE failure: only VLANs where the failed PE had the highest hash move
- Minimal disruption: $\frac{V}{N}$ VLANs move on average, but they redistribute to random PEs (not all to one)
- Computation: $O(N)$ per VLAN (hash against all PEs)

**Advantage over mod-based:** When PE $P_k$ fails, only VLANs where $P_k$ was the winner move. The new winner among remaining PEs is independent for each VLAN, providing better distribution.

### Preference-Based Election (RFC 8584)

Explicit priority assignment:

$$DF = \text{PE with highest preference for VLAN } V$$

Ties broken by lowest PE IP. Allows administrative control over DF placement.

**Use case:** When specific PEs have better CE connectivity or capacity.

### Comparison

| Property | Mod-Based | HRW | Preference |
|:---|:---|:---|:---|
| Distribution | Perfect (mod) | Statistical (hash) | Administrative |
| Failure disruption | $\frac{V}{N}$ VLANs, all at once | $\frac{V}{N}$ VLANs, scattered | Depends on config |
| Computation | $O(1)$ | $O(N)$ per VLAN | $O(N)$ per VLAN |
| Predictability | High | Low (hash-dependent) | High |
| Configuration | None | None | Per-PE weights |
| RFC | 7432 | 8584 | 8584 |

---

## 4. MAC Mobility Sequence Numbers

### The Protocol

When a MAC $M$ moves from PE-A to PE-B:

1. PE-B learns MAC $M$ on a local port
2. PE-B checks if a Type 2 route for $M$ exists from another PE
3. If yes, PE-B increments the sequence number: $seq_{new} = seq_{old} + 1$
4. PE-B advertises Type 2 with MAC Mobility extended community, $seq_{new}$
5. All PEs receive the update; higher sequence wins → forwarding updated to PE-B
6. PE-A withdraws its Type 2 route for $M$ (implicit or explicit)

### Sequence Number Arithmetic

The sequence number is a 32-bit unsigned integer:

$$seq \in [0, 2^{32} - 1] = [0, 4,294,967,295]$$

**Rollover:** RFC 7432 uses modular arithmetic for comparison:

$$seq_a > seq_b \iff 0 < (seq_a - seq_b) < 2^{31}$$

This handles wraparound correctly (similar to TCP sequence number comparison).

### Duplicate Detection

To prevent infinite MAC flapping (loop detection):

$$\text{Duplicate if:} \quad moves > M_{max} \text{ within } T_{window}$$

Where:
- $M_{max}$ = maximum allowed moves (typically 5)
- $T_{window}$ = detection window (typically 180 seconds)

On duplicate detection:
1. MAC is frozen (not advertised) for $T_{freeze}$ seconds
2. Alert raised for operator investigation
3. After $T_{freeze}$, MAC is re-enabled

### Move Rate Analysis

For a host moving between two PEs at rate $R$ moves/second:

$$\text{BGP updates/sec} = R \times 2 \text{ (one advertisement + one implicit withdrawal)}$$

For a route reflector serving $P$ PEs:

$$\text{Updates reflected} = R \times 2 \times (P - 1)$$

Example: 10 MAC moves/sec, 50 PEs:

$$\text{RR update rate} = 10 \times 2 \times 49 = 980 \text{ updates/sec}$$

This is manageable, but a MAC flap storm (hundreds of moves/sec) can overwhelm the RR.

---

## 5. Split-Horizon with ESI Labels

### The Problem

In all-active multihoming, when PE-A receives a BUM frame from CE and floods it to remote PEs, PE-B (also connected to the same ES) must not forward it back to the CE. Without split-horizon, the CE receives a duplicate.

### The Solution: ESI Labels

1. When PE-A floods a BUM frame received from ES $E$, it pushes an ESI label in the MPLS stack
2. Remote PE-B receives the frame, sees the ESI label matches its local ES
3. PE-B drops the frame (does not forward to CE)

### Label Stack with ESI

```
+-------------------+-------------------+-------------------+-----------+
| Transport Label   | ESI Label         | EVPN Service Label| L2 Frame  |
| (to remote PE)    | (split-horizon)   | (EVI identifier)  |           |
+-------------------+-------------------+-------------------+-----------+
```

**ESI label allocation:**
- Each PE allocates an ESI label per Ethernet Segment
- Advertised via Type 1 (EAD per-ES) route with ESI Label extended community
- All PEs on the same ES know each other's ESI labels

### VXLAN Split-Horizon

In VXLAN fabrics, ESI labels are not available (VXLAN has no label stack). Split-horizon uses:
- Source VTEP IP filtering: remote PE checks if the source VTEP is a peer on the same ES
- Local bias: PE prefers locally learned MACs, does not forward BUM from remote PEs to local ES

---

## 6. BUM Handling: Ingress Replication vs Multicast

### Ingress Replication

Every PE replicates BUM frames to every other PE in the EVI:

$$\text{Replications per BUM frame} = N_{PEs} - 1$$

**Bandwidth overhead per PE for BUM traffic rate $B$:**

$$\text{BW}_{replication} = B \times (N_{PEs} - 1)$$

**Total network BUM bandwidth:**

$$\text{BW}_{total} = B \times N_{source\_PEs} \times (N_{PEs} - 1)$$

For 50 PEs, 1 Mbps BUM traffic per PE:

$$\text{BW per PE (egress)} = 1 \times 49 = 49 \text{ Mbps}$$

$$\text{Total} = 1 \times 50 \times 49 = 2,450 \text{ Mbps}$$

### Multicast Underlay

PEs join multicast groups for each EVI. BUM frames are sent once to the multicast tree:

$$\text{Replications at source PE} = 1 \text{ (to multicast group)}$$

$$\text{Total replication} = N_{PEs} - 1 \text{ (handled by multicast tree in the underlay)}$$

**Bandwidth overhead per source PE:**

$$\text{BW}_{source} = B \times 1 = B$$

Network bandwidth is the same (every PE receives the frame), but replication is distributed across the underlay switches, not concentrated at the source PE.

### Break-Even Analysis

Ingress replication overhead grows linearly with PE count. The PE's replication capacity $C_{rep}$ determines the maximum fabric size:

$$N_{max} = \frac{C_{rep}}{B} + 1$$

For a PE with 10 Gbps replication budget and 100 Mbps BUM per EVI:

$$N_{max} = \frac{10,000}{100} + 1 = 101 \text{ PEs}$$

Beyond this, multicast underlay is necessary.

### Multicast Group Mapping

For $E$ EVIs mapped to $G$ multicast groups:

- **One-to-one:** $G = E$ (finest granularity, most state)
- **Many-to-one:** $G = 1$ (single group for all EVIs, maximum flooding)
- **Hashed:** $G = E \bmod K$ for some constant $K$ (balanced trade-off)

Multicast state per switch:

$$\text{State} = G \times S_{mcast\_entry}$$

---

## 7. Symmetric vs Asymmetric IRB — Forwarding Path Analysis

### Asymmetric IRB Forwarding

**Ingress PE processing:**

1. Receive frame on VLAN 100 (VNI 10100)
2. Perform MAC lookup: destination MAC is in VLAN 200
3. Route in the VRF: source subnet 10.100.0.0/24 → destination subnet 10.200.0.0/24
4. Determine egress PE from Type 2 route for destination MAC
5. Encapsulate with **destination L2 VNI (10200)** and send to egress PE

**Egress PE processing:**

1. Receive VXLAN packet with VNI 10200
2. Decapsulate → bridge domain for VLAN 200
3. MAC lookup → forward to local port (pure L2)

**Scaling constraint:** Every PE must have **every VNI** in the EVI configured, even if no local hosts exist in that VNI. For $V$ VLANs:

$$\text{VNI config per PE} = V$$

$$\text{Total VNI configs} = V \times P$$

With 1,000 VLANs and 50 PEs: 50,000 VNI configurations.

### Symmetric IRB Forwarding

**Ingress PE processing:**

1. Receive frame on VLAN 100 (VNI 10100)
2. Route in the VRF: determine egress PE
3. Encapsulate with **L3 VNI (50900, per-VRF)** and router MAC of egress PE as inner DMAC
4. Send to egress PE

**Egress PE processing:**

1. Receive VXLAN packet with L3 VNI 50900
2. Decapsulate → VRF lookup (IP routing)
3. Route to destination subnet (VLAN 200)
4. MAC lookup in VLAN 200 → forward to local port

**Scaling advantage:** PEs only configure VNIs for **locally-present VLANs** plus one L3 VNI per VRF:

$$\text{VNI config per PE} = V_{local} + N_{VRFs}$$

Where $V_{local} \ll V_{total}$. For 50 PEs where each PE has 20 local VLANs + 5 VRFs:

$$\text{Total VNI configs} = 50 \times (20 + 5) = 1,250$$

vs. 50,000 for asymmetric. **A 40x reduction.**

### Type 2 Route Label Analysis

**Asymmetric IRB Type 2 route:**
```
MAC: aa:bb:cc:dd:ee:ff
IP: 10.100.0.10
Label 1: 10100 (L2 VNI for VLAN 100)
Label 2: not present
```

Remote PE uses Label 1 to identify the bridge domain on the egress PE.

**Symmetric IRB Type 2 route:**
```
MAC: aa:bb:cc:dd:ee:ff
IP: 10.100.0.10
Label 1: 10100 (L2 VNI for intra-subnet forwarding)
Label 2: 50900 (L3 VNI for inter-subnet forwarding)
Router MAC: 00:11:22:33:44:55 (extended community)
```

Remote PE uses Label 2 (L3 VNI) for inter-subnet traffic and the Router MAC as inner DMAC. Label 1 is used for intra-subnet (same VLAN, different PE) traffic.

---

## 8. EVPN Convergence Analysis

### MAC Withdrawal (Mass Withdrawal via Type 1)

When an Ethernet Segment fails, the PE withdraws the per-ES EAD route (Type 1):

$$\text{Updates} = 1 \text{ (single withdrawal)}$$

All remote PEs invalidate all MACs associated with that ESI:

$$\text{MACs invalidated} = M_{ES} \text{ (all MACs on that ES)}$$

**Without Type 1 (pure Type 2 withdrawal):**

$$\text{Updates} = M_{ES} \text{ (one per MAC)}$$

For an ES with 10,000 MACs:
- Type 1 mass withdrawal: 1 BGP UPDATE, convergence in ~50ms
- Type 2 individual withdrawal: 10,000 BGP UPDATEs, convergence in seconds

**Convergence time with Type 1:**

$$T_{converge} = T_{detect} + T_{BGP\_withdraw} + T_{FIB\_update}$$

Where:
- $T_{detect}$ = BFD (150ms) or interface down (~10ms)
- $T_{BGP\_withdraw}$ = single UPDATE propagation (~10-50ms via RR)
- $T_{FIB\_update}$ = bulk MAC invalidation (~10-50ms in hardware)

**Total: < 250ms** for mass withdrawal convergence.

### Aliasing Convergence

When a new PE joins an ES (Type 1 per-EVI advertisement):

1. Remote PEs add the new PE as an additional next-hop for MACs on that ES
2. Traffic is load-balanced across all active PEs on the ES

$$\text{Load per PE} = \frac{1}{N_{PEs\_on\_ES}}$$

Convergence: time for BGP advertisement to propagate + FIB programming.

---

## 9. EVPN vs VPLS — Quantitative Comparison

### State Comparison for N PEs, M MACs, V VLANs

| Metric | VPLS (LDP) | VPLS (BGP) | EVPN |
|:---|:---:|:---:|:---:|
| Signaling sessions | $\frac{N(N-1)}{2}$ | $N$ (with RR) | $N$ (with RR) |
| PW/label state | $N-1$ per PE | $N-1$ per PE | 1 per MAC (Type 2) |
| MAC learning | Data plane | Data plane | Control plane (BGP) |
| MAC table sync | None | None | Full (via Type 2) |
| BUM optimization | None | None | ARP suppression |
| Multihoming | None (or MC-LAG) | None (or MC-LAG) | Native (ESI) |
| MAC mobility | None | None | Sequence numbers |
| IP prefix routing | N/A | N/A | Type 5 |

### Convergence Comparison

| Scenario | VPLS | EVPN |
|:---|:---|:---|
| PE failure | MAC relearning via flooding (seconds) | Type 1 mass withdrawal (< 250ms) |
| MAC move | Flood + relearn (seconds) | Type 2 update (< 1 second) |
| New PE joins | Manual PW config + MAC learning | BGP auto-discovery + aliasing |
| Link failure | PW down + MAC flush | ES down + mass withdrawal |

### Operational Complexity

| Operation | VPLS | EVPN |
|:---|:---|:---|
| Add new PE | Configure PW on every existing PE | Configure RT on new PE only |
| Add new VLAN | Touch all PEs (VPLS instance) | Touch only PEs with local hosts |
| Troubleshoot MAC | Data-plane packet capture | `show bgp l2vpn evpn route-type 2` |
| Multihoming | External MC-LAG (vendor-specific) | Standard ESI (interoperable) |

---

## References

- [RFC 7432 — BGP MPLS-Based Ethernet VPN](https://www.rfc-editor.org/rfc/rfc7432)
- [RFC 8584 — Framework for EVPN Designated Forwarder Election](https://www.rfc-editor.org/rfc/rfc8584)
- [RFC 7209 — Requirements for Ethernet VPN](https://www.rfc-editor.org/rfc/rfc7209)
- [RFC 8365 — Network Virtualization Overlay Using EVPN](https://www.rfc-editor.org/rfc/rfc8365)
- [RFC 9135 — Integrated Routing and Bridging in EVPN](https://www.rfc-editor.org/rfc/rfc9135)
- [RFC 9136 — IP Prefix Advertisement in EVPN](https://www.rfc-editor.org/rfc/rfc9136)
- [RFC 4761 — VPLS Using BGP (for comparison)](https://www.rfc-editor.org/rfc/rfc4761)
- [RFC 4762 — VPLS Using LDP (for comparison)](https://www.rfc-editor.org/rfc/rfc4762)
