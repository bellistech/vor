# JunOS L2VPN — Deep Dive Theory and Analysis

> In-depth exploration of JunOS L2VPN implementation: VPLS signaling comparison (LDP vs BGP), MAC table management internals, split-horizon enforcement, pseudowire mechanics, VPLS scaling analysis, and EVPN migration strategy. For JNCIE-SP level understanding.

## 1. JunOS L2VPN Implementation Architecture

### 1.1 L2VPN Service Model

JunOS implements two fundamental L2VPN service types:

**Point-to-Point (l2circuit / pseudowire):**
- Emulates a direct Ethernet wire between two CEs
- Uses targeted LDP for pseudowire signaling (RFC 4447)
- Virtual Circuit ID uniquely identifies the pseudowire between two PEs
- No MAC learning or flooding — transparent bit transport

**Multipoint (VPLS):**
- Emulates a virtual Ethernet switch across the MPLS network
- Full MAC learning, flooding, and forwarding
- Requires full mesh of pseudowires between all PEs (or H-VPLS for scaling)
- Signaled via LDP (RFC 4762) or BGP (RFC 4761)

### 1.2 Pseudowire Internals

A pseudowire consists of:

1. **PSN tunnel:** The MPLS LSP (LDP or RSVP-TE) providing transport between PEs
2. **PW label:** Inner label identifying the specific pseudowire within the PSN tunnel
3. **Control word (optional):** 4-byte field after the PW label for sequencing and padding

Label stack for L2VPN traffic:
```
[Transport label(s) | PW label | Control word (optional) | L2 frame]
```

The PW label is signaled via targeted LDP (FEC 128 or FEC 129) or BGP L2VPN NLRI.

### 1.3 FEC 128 vs FEC 129

| Aspect        | FEC 128 (PWid)              | FEC 129 (Generalized PWid)     |
|---------------|-----------------------------|---------------------------------|
| Identification| VC ID (32-bit)              | AGI + SAII + TAII               |
| Discovery     | Manual neighbor config      | BGP auto-discovery possible     |
| Scaling       | Simple, limited             | Better for large-scale          |
| JunOS config  | `virtual-circuit-id`        | `pseudowire-status-tlv`        |

## 2. VPLS Signaling Comparison

### 2.1 LDP-Signaled VPLS (RFC 4762 — Martini)

**Discovery:** Manual configuration of all PE neighbors. Each PE must be explicitly configured with every other PE in the VPLS instance.

**Signaling:**
1. PE establishes targeted LDP session to each remote PE
2. Each PE sends FEC 128 label mapping for the VPLS instance
3. Label mappings create a full mesh of pseudowires
4. Each pseudowire has independent PW labels in each direction

**Advantages:**
- Simpler to understand and configure for small deployments
- No BGP dependency
- Direct control over which PEs participate

**Disadvantages:**
- N*(N-1)/2 pseudowires in full mesh (O(n^2) scaling)
- Manual configuration of every neighbor on every PE
- No auto-discovery — adding a PE requires updating all existing PEs
- No built-in multihoming support

### 2.2 BGP-Signaled VPLS (RFC 4761 — Kompella)

**Discovery:** BGP auto-discovery using `route-distinguisher` and `vrf-target`. PEs discover each other through BGP VPN route exchange.

**Signaling:**
1. Each PE advertises a BGP L2VPN NLRI containing:
   - Route distinguisher
   - Site ID and label block (VE ID + VE block offset + VE block size)
   - Label base for pseudowire label computation
2. Remote PEs compute the PW label from the label block:
   ```
   PW label = label_base + (remote_VE_ID - VE_block_offset)
   ```
3. Full mesh of pseudowires is established automatically

**Advantages:**
- Auto-discovery via BGP (adding a PE requires only local config)
- Label block mechanism reduces signaling (one NLRI per PE, not per PW)
- Built-in multihoming with Designated Forwarder election
- Leverages existing BGP infrastructure and route reflectors

**Disadvantages:**
- Requires BGP with `family l2vpn signaling`
- Label block computation adds complexity
- Debugging requires understanding BGP L2VPN NLRI encoding

### 2.3 Label Computation in BGP VPLS

Given a PE with:
- Label base = 800000
- VE block offset = 1
- VE block size = 8

For remote VE_ID = 3:
```
PW label = 800000 + (3 - 1) = 800002
```

This label is used by the remote PE when sending traffic TO this PE for VE_ID 3. Each PE independently computes the labels needed to reach every other PE.

## 3. MAC Table Management in JunOS

### 3.1 MAC Learning Process

JunOS VPLS MAC learning follows standard Ethernet switch behavior:

1. **Source MAC learning:** When a frame arrives on an interface (local or PW), the source MAC is recorded with the ingress interface/PW.
2. **Destination MAC lookup:** If the destination MAC is known, forward to the recorded interface/PW. If unknown, flood.
3. **MAC aging:** Entries not refreshed within the aging timer (default 300 seconds) are removed.
4. **MAC move:** If a MAC is learned on a different interface/PW than previously recorded, the entry is updated (MAC mobility).

### 3.2 MAC Table Storage

JunOS stores MAC entries in:
- **RE MAC table:** Software table used for management and monitoring (`show vpls mac-table`)
- **PFE MAC table:** Hardware table in the forwarding ASIC for line-rate lookups

The PFE table has a finite size determined by the hardware platform. When the table is full:
- New MACs cannot be learned
- Traffic to unknown destinations is flooded
- MAC limiting (`mac-limit`) prevents table exhaustion per interface

### 3.3 MAC Flushing

When a topology change occurs (PW failure, link down), stale MAC entries must be removed:

**LDP-based MAC flush:** The PE detecting the failure sends an LDP MAC Address Withdraw message to all peers. Peers flush MAC entries associated with the failed PE/PW.

**BGP-based MAC flush:** Uses BGP L2VPN NLRI withdrawal. When a PE withdraws its route, remote PEs flush associated MACs.

**Flush timing matters:** Faster MAC flush = faster convergence. Without flush, traffic continues to the failed path until MAC entries age out (potentially minutes).

### 3.4 Qualified and Unqualified Learning

**Unqualified learning (default):** One MAC table shared across all VLANs within the VPLS instance. A MAC address is unique regardless of VLAN.

**Qualified learning:** Separate MAC table per VLAN. The same MAC address can exist in different VLANs. Requires more table space but is needed for complex deployments with VLAN translation.

## 4. Split-Horizon in JunOS VPLS

### 4.1 The Problem: Loops in Full Mesh

VPLS creates a full mesh of pseudowires between PEs. Without split-horizon:
1. PE1 floods a BUM frame on all PWs (to PE2, PE3)
2. PE2 receives the frame, floods it on all PWs (back to PE1, and to PE3)
3. PE3 receives the frame twice (from PE1 and PE2)
4. Infinite loop ensues

### 4.2 Split-Horizon Rule

**Rule:** A frame received on a pseudowire is NEVER forwarded to another pseudowire in the same mesh group.

Implementation in JunOS:
- All pseudowires in a VPLS instance belong to the default mesh group
- Frames received from a PW are forwarded to local interfaces only (and vice versa)
- This eliminates PW-to-PW flooding loops without STP

### 4.3 Mesh Groups in H-VPLS

H-VPLS uses explicit mesh groups to control split-horizon behavior:

- **Core mesh group:** PWs between hub PEs. Split-horizon applies (no PW-to-PW forwarding).
- **Spoke mesh group:** PWs from spoke PEs. Different mesh group from core.
- Frames from a spoke PW CAN be forwarded to core PWs (different mesh group).
- Frames from a core PW CANNOT be forwarded to other core PWs (same mesh group).

This enables hub PEs to relay traffic between spoke PEs while preventing core loops.

### 4.4 Impact on BUM Traffic

BUM (Broadcast, Unknown unicast, Multicast) flooding with split-horizon:

```
CE1 --> PE1 --PW--> PE2 --> CE2
              |
              +--PW--> PE3 --> CE3

PE1 floods to all PWs and local interfaces.
PE2 receives via PW, forwards to local CE2 only (not to PE3 PW).
PE3 receives via PW, forwards to local CE3 only (not to PE2 PW).
```

Each PE floods BUM to all PWs and local interfaces. Each PE only forwards PW-received BUM to local interfaces. This ensures each CE receives exactly one copy.

## 5. VPLS Scaling Analysis

### 5.1 Full Mesh Problem

For N PEs in a VPLS instance:
- Number of pseudowires: `N * (N-1) / 2`
- Number of targeted LDP sessions (LDP VPLS): `N * (N-1) / 2`
- Per-PE state: `N - 1` pseudowires

| PEs (N) | Pseudowires | Per-PE State |
|---------|-------------|--------------|
| 5       | 10          | 4            |
| 10      | 45          | 9            |
| 50      | 1,225       | 49           |
| 100     | 4,950       | 99           |
| 500     | 124,750     | 499          |

### 5.2 H-VPLS Scaling Improvement

With H-VPLS (H hub PEs, S spoke PEs):
- Core full mesh: `H * (H-1) / 2`
- Spoke connections: `S` (one PW per spoke to one hub)
- Total: `H * (H-1) / 2 + S`

Example: 10 hub PEs, 90 spoke PEs:
- Full mesh: `10 * 9 / 2 = 45` core PWs + 90 spoke PWs = 135 total
- Compared to full mesh of 100 PEs: 4,950 PWs
- Reduction: 97% fewer pseudowires

### 5.3 BGP VPLS Scaling with Route Reflectors

BGP VPLS with RRs eliminates N^2 signaling:
- Each PE has one BGP session to the RR
- RR distributes L2VPN NLRI to all PEs
- Total BGP sessions: N (one per PE to RR)
- Data plane still requires full mesh of PWs, but signaling scales linearly

### 5.4 MAC Table Scaling

Key MAC table scaling concerns:
- Hardware ASIC MAC table limits (platform specific: 32K, 64K, 128K, 1M entries)
- Shared across all VPLS instances on the platform
- MAC explosion from uncontrolled BUM traffic
- Mitigation: MAC limiting per interface and per instance

## 6. Migration to EVPN

### 6.1 Why Migrate

EVPN addresses fundamental VPLS limitations:
- **Active-active multihoming:** VPLS supports only active-standby
- **MAC mobility:** EVPN has built-in MAC mobility procedures
- **ARP/ND suppression:** Reduces BUM flooding
- **Per-flow load balancing:** Aliasing via ESI labels
- **Control-plane MAC learning:** Reduces flooding
- **Integrated L2 and L3:** Type-5 routes for L3 with same instance

### 6.2 Migration Strategy on JunOS

**Phase 1: EVPN-VPLS interop**
- Deploy EVPN on new PEs while existing PEs run VPLS
- Use `virtual-switch` instance type that supports both protocols
- EVPN PEs and VPLS PEs coexist in the same broadcast domain

**Phase 2: Gradual PE migration**
- Convert PEs from VPLS to EVPN one at a time
- During transition, dual-stack PEs run both protocols
- MAC synchronization between EVPN and VPLS data planes

**Phase 3: Full EVPN**
- Remove VPLS configuration from all PEs
- Enable EVPN-only features (active-active, ARP suppression)
- Clean up legacy VPLS configuration

### 6.3 Key Differences in JunOS Configuration

| Aspect            | VPLS                           | EVPN                              |
|-------------------|--------------------------------|-----------------------------------|
| Instance type     | `vpls`                         | `evpn` or `virtual-switch`        |
| Signaling         | LDP or `family l2vpn`         | `family evpn signaling`           |
| MAC learning      | Data plane only                | Control plane (BGP) + data plane  |
| Multihoming       | BGP site-based (active-standby)| ESI-based (active-active)        |
| BUM handling      | Full flooding                  | Ingress replication or multicast  |

## See Also

- junos-mpls-advanced
- junos-l3vpn
- junos-evpn-vxlan

## References

- RFC 4761 — VPLS Using BGP for Auto-Discovery and Signaling
- RFC 4762 — VPLS Using LDP Signaling
- RFC 4447 — Pseudowire Setup and Maintenance Using LDP
- RFC 4448 — Encapsulation Methods for Transport of Ethernet over MPLS
- RFC 6074 — Provisioning, Auto-Discovery, and Signaling in L2VPNs
- RFC 7432 — BGP MPLS-Based Ethernet VPN
- Juniper TechLibrary: VPLS Technical Overview
- Juniper TechLibrary: EVPN-VPLS Interoperability Guide
