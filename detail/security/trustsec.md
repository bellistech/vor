# Cisco TrustSec — Identity-Based Segmentation Architecture and Protocol Internals

> *Cisco TrustSec replaces topology-dependent ACLs with identity-driven segmentation using 16-bit Security Group Tags (SGTs). By decoupling policy from IP addresses, TrustSec enables scalable micro-segmentation where adding endpoints does not require ACL updates. The architecture relies on three pillars: classification (SGT assignment at ingress), propagation (inline CMD tagging or SXP), and enforcement (SGACL at egress). Understanding the protocol mechanics, TCAM programming model, and migration strategies is essential for deploying TrustSec in campus and data center networks.*

---

## 1. TrustSec vs Traditional ACL — Scaling Analysis

### The ACL Explosion Problem

Traditional segmentation using IP-based ACLs suffers from combinatorial explosion. Given N security zones, the number of ACL rules required grows as O(N^2) for each enforcement point, and the rules must reference specific IP addresses or subnets:

```
Traditional ACL approach:
  - 10 security zones → 90 ACL rule sets (N * (N-1))
  - 50 security zones → 2,450 ACL rule sets
  - 100 security zones → 9,900 ACL rule sets

  Each rule set contains multiple ACEs (permit/deny lines)
  Total ACEs = rule_sets * avg_ACEs_per_set
  Example: 50 zones * 49 pairs * 20 ACEs avg = 49,000 ACEs per device

Problems:
  1. Every device needs the full ACL set (replicated everywhere)
  2. IP address changes require ACL updates across all devices
  3. TCAM is consumed per-ACE — physical hardware limit
  4. No mobility: VLAN change = ACL rewrite
  5. Audit trail: which ACL applies to which user?
```

### TrustSec Scaling Model

TrustSec separates classification from enforcement. The policy is defined as an SGT-to-SGT matrix, independent of IP addresses:

```
TrustSec approach:
  - 10 SGTs → 1 matrix (100 cells, most are "no policy" = permit/deny default)
  - 50 SGTs → 1 matrix (2,500 cells, sparse — typically 10-20% populated)
  - 100 SGTs → 1 matrix (10,000 cells, same sparsity)

  Each populated cell = 1 SGACL (a few ACEs)
  Total ACEs = populated_cells * avg_ACEs_per_SGACL
  Example: 50 SGTs * 10% populated * 250 cells * 5 ACEs = 1,250 ACEs

  Advantages:
    1. Policy defined once (ISE), pushed to enforcement points only
    2. IP changes do not affect policy (SGT follows identity)
    3. TCAM consumed only for unique SGACL content (shared across cells)
    4. Mobility: user moves, SGT follows via RADIUS or SXP
    5. Audit: SGT = role = clear semantic meaning
```

### TCAM Comparison

```
Traditional ACL TCAM usage:
  - Each ACE = 1 TCAM entry (source IP, dest IP, protocol, port)
  - Duplicate ACEs across interfaces/VLANs
  - TCAM capacity: 4K-32K entries (Catalyst 9K), 64K-128K (Nexus 9K)

TrustSec SGACL TCAM usage:
  - SGACL uses a separate TCAM region ("SGACL" or "role-based ACL")
  - Indexed by (source SGT, dest SGT) pair
  - ACL content is shared/deduplicated across matrix cells
  - Catalyst 9300: 2K SGACL entries (expandable via SDM template)
  - Nexus 9000: 4K SGACL entries
  - TCAM is consumed at egress enforcement point only (not every hop)
```

---

## 2. CMD (Cisco Meta Data) — Inline Tagging Frame Format

### Frame Structure

CMD inserts a 16-byte header between the source MAC address and the EtherType/VLAN tag of the original Ethernet frame. The CMD EtherType is 0x8909:

```
Standard Ethernet frame:
  [Dst MAC 6B][Src MAC 6B][EtherType 2B][Payload][FCS 4B]

CMD-tagged frame:
  [Dst MAC 6B][Src MAC 6B][CMD EType 0x8909 2B][CMD Header 16B][Original EtherType 2B][Payload][FCS 4B]

CMD Header breakdown (16 bytes):
  +-------+--------+-----------+-----------+----------+-----------+
  | Ver   | Length | CMD EType | SGP Type  | SGT Val  | Options   |
  | (1B)  | (1B)  | (2B)      | (2B)      | (2B)     | (8B)      |
  +-------+--------+-----------+-----------+----------+-----------+

  Version: 1 (current)
  Length: total CMD header length in 4-byte words
  CMD EtherType: 0x0001 (TrustSec)
  SGP Type: 0x0001 (Security Group Tag)
  SGT Value: 16-bit SGT (0x0000 - 0xFFFF)
  Options: reserved/padding
```

### DMAC and CRC Considerations

```
FCS (Frame Check Sequence) recalculation:
  - CMD insertion modifies the frame content between Src MAC and payload
  - FCS must be recalculated by the tagging device
  - FCS is recalculated again when CMD is stripped at the egress device
  - This is performed in hardware (ASIC) at line rate

  Impact on frame size:
    Standard Ethernet: 1518 bytes max (excluding preamble)
    CMD overhead: 18 bytes (EtherType 2B + CMD header 16B)
    CMD-tagged frame: 1536 bytes max
    Baby giant frame handling required on transit switches

    Solution: MTU adjustment on TrustSec trunk links
      system mtu 9198               ! or per-interface
      interface TenGigabitEthernet1/0/1
        mtu 9198                    ! accommodate CMD + VXLAN overhead
```

### Inline Tagging Behavior

```
Ingress classification:
  1. Device authenticates endpoint (802.1X, MAB, or static)
  2. SGT is assigned (from RADIUS, SXP, or manual config)
  3. IP-SGT binding is created in local binding table
  4. Outgoing frames on TrustSec-capable trunk links:
     → CMD header inserted with assigned SGT
     → FCS recalculated

Transit forwarding:
  1. CMD-tagged frame received on TrustSec trunk
  2. SGT extracted and associated with the flow
  3. Frame forwarded with CMD intact (SGT preserved)
  4. No IP-SGT binding table lookup needed (SGT is in-band)

Egress enforcement:
  1. CMD-tagged frame arrives at enforcement point
  2. Source SGT extracted from CMD header
  3. Destination SGT looked up from local IP-SGT binding table
  4. SGACL match: (src SGT, dst SGT) → permit/deny
  5. CMD stripped before delivery to endpoint
  6. FCS recalculated
```

---

## 3. SXP Protocol Internals

### Protocol Overview

SXP (SGT Exchange Protocol) is a control-plane protocol that propagates IP-to-SGT bindings between network devices. It operates over TCP port 64999 and uses MD5 authentication for connection integrity:

```
SXP characteristics:
  - Transport: TCP port 64999
  - Authentication: MD5 (optional but recommended)
  - Roles: Speaker (sends bindings) ↔ Listener (receives bindings)
  - SXPv1-v2: unidirectional (speaker-to-listener only)
  - SXPv3: loop detection via peer-sequence attribute
  - SXPv4: bidirectional mode (both speaker and listener)
  - Keepalive: configurable hold-time (default 120s)
  - Delete-hold-down timer: 120s (delay before deleting stale bindings)
```

### SXP Message Types

```
SXP Message Format:
  +---------+---------+----------+-----------+
  | Version | Type    | Length   | Payload   |
  | (4B)    | (4B)    | (4B)    | (var)     |
  +---------+---------+----------+-----------+

Message Types:
  OPEN       (1) — initiate connection, negotiate version
  OPEN_RESP  (2) — response to OPEN, version agreement
  UPDATE     (3) — add/delete IP-SGT bindings
  ERROR      (4) — report protocol errors
  PURGE_ALL  (5) — delete all bindings from a peer
  KEEPALIVE  (6) — connection liveness check

UPDATE message payload:
  +----------+----------+-------------+
  | Type     | Length   | Value       |
  +----------+----------+-------------+
  | ADD_IPv4  | 12      | prefix + SGT |
  | DEL_IPv4  | 12      | prefix + SGT |
  | ADD_IPv6  | 24      | prefix + SGT |
  | DEL_IPv6  | 24      | prefix + SGT |
  +----------+----------+-------------+

  Binding record:
    IP prefix length (1B) + IP address (4B or 16B) + SGT (2B) + padding
```

### SXP Loop Prevention (SXPv3+)

```
Peer-sequence attribute:
  - Each SXP speaker appends its node-ID to the peer-sequence list
  - When a listener receives a binding, it checks:
    → If its own node-ID is already in the peer-sequence → LOOP → discard
    → Otherwise, append its node-ID and propagate

  Example:
    Switch-A (speaker) → Switch-B (listener/speaker) → Switch-C (listener)
    Binding: 10.1.1.5 → SGT 5
      At Switch-A: peer-seq = [A]
      At Switch-B: peer-seq = [A, B]
      At Switch-C: peer-seq = [A, B, C]
      If Switch-C also peers with Switch-A:
        Switch-C sends to Switch-A with peer-seq = [A, B, C]
        Switch-A sees itself in peer-seq → discard (loop prevented)
```

### SXP Scaling Considerations

```
SXP binding table capacity:
  - Catalyst 9300/9400: 16,000 IP-SGT bindings
  - Catalyst 9500: 32,000 IP-SGT bindings
  - Nexus 9000: 64,000 IP-SGT bindings
  - ISE (SXP aggregator): 500,000 IP-SGT bindings

SXP topology best practices:
  1. Use ISE as SXP aggregator (hub-and-spoke)
     - Access switches: SXP speakers → ISE
     - DC/enforcement: SXP listeners ← ISE
     - Reduces SXP mesh complexity from O(N^2) to O(N)

  2. Minimize SXP hops (each hop adds propagation delay)
     - Binding propagation: ~1 second per hop
     - Total convergence: hops * 1s + TCP setup time

  3. Prefer inline tagging where hardware supports it
     - SXP is a fallback for non-CMD-capable devices
     - Inline tagging provides real-time SGT (no binding table lookup)
```

---

## 4. SGACL TCAM Programming

### TCAM Region Allocation

```
SGACL uses a dedicated TCAM region separate from traditional ACLs:

Catalyst 9300 SDM templates:
  sdm prefer advanced       ! default — 2K SGACL entries
  sdm prefer custom         ! customizable TCAM allocation

  Custom TCAM allocation:
    sdm prefer custom sgacl 4096   ! increase SGACL entries

  Verify:
    show sdm prefer
    show platform hardware fed switch active fwd-asic resource tcam utilization

Nexus 9000 TCAM carving:
  hardware access-list tcam region arp-ether 0
  hardware access-list tcam region racl 0
  hardware access-list tcam region sgacl 2048
  copy running-config startup-config
  reload
```

### SGACL Lookup Pipeline

```
Packet processing pipeline with SGACL:

  1. Ingress port → Source SGT determination
     a. If CMD present → extract SGT from header
     b. If no CMD → lookup IP in IP-SGT binding table
     c. If no binding found → use port default SGT or SGT 0 (Unknown)

  2. Routing/switching decision (standard L2/L3 forwarding)

  3. Egress port → Destination SGT determination
     a. Lookup destination IP in local IP-SGT binding table
     b. If no binding → use SGT 0 (Unknown)

  4. SGACL lookup: TCAM match on (src_SGT, dst_SGT)
     a. If match → apply ACEs in the SGACL (permit/deny per flow)
     b. If no match → apply default policy (monitor/permit/deny)

  5. If permitted → forward (strip CMD if going to non-TrustSec port)
     If denied → drop + increment role-based counter

  Note: SGACL enforcement occurs at EGRESS, not ingress
  This means the enforcement device must know the destination SGT
```

### Monitor Mode

```
Monitor mode allows SGACL evaluation without dropping traffic:
  - All SGACL matches are logged but traffic is permitted
  - Used during TrustSec deployment for policy validation

  ! Enable monitor mode globally
  cts role-based monitor enable

  ! Enable monitor mode per-SGACL (ISE)
  ! In policy matrix: set contract to "Monitor" instead of "Enforce"

  ! Check monitor mode counters
  show cts role-based counters
    From  To   SW-Monitored  HW-Monitored  SW-Permit  HW-Permit
    5     100  1234          56789          0          0

  Migration path:
    1. Deploy SGT assignment (classification only)
    2. Enable SGACL in monitor mode (log matches, no drops)
    3. Analyze counters and adjust policy
    4. Switch to enforce mode per-cell in the matrix
```

---

## 5. Policy Matrix Design Methodology

### Matrix Design Principles

```
Step 1: Define security groups (SGTs)
  - Map to organizational roles, not network topology
  - Start broad, refine later (5-15 SGTs initially)
  - Examples: Employees, Contractors, Guests, Servers, IoT, PCI, Voice

Step 2: Define default posture
  - Closed model: default deny, explicitly permit (most secure)
  - Open model: default permit, explicitly deny (easier migration)
  - Recommended: start open, move to closed after validation

Step 3: Populate matrix cells
  - Focus on high-value targets first (PCI, servers, management)
  - Leave unneeded cells empty (inherits default policy)
  - Use ISE's "catch-all" rule for unknown SGT pairs

Step 4: Iterate with monitor mode
  - Deploy policy in monitor mode
  - Collect traffic flow data (NetFlow, SGACL counters)
  - Identify legitimate flows that would be denied
  - Adjust SGACLs before switching to enforcement
```

### Matrix Sparsity and Optimization

```
A well-designed matrix is sparse (most cells are empty/default):

Example: 20 SGTs = 400 cells
  Populated cells: 40-80 (10-20%)
  Each cell: 1 SGACL with 3-10 ACEs
  Total unique SGACLs: 15-30 (many cells share the same SGACL)

SGACL reuse:
  - Define generic SGACLs: Permit_Web, Permit_DNS, Permit_DHCP, Deny_All
  - Apply the same SGACL to multiple matrix cells
  - Reduces TCAM consumption (SGACL content is deduplicated)

Naming convention:
  SGACL name format: <Action>_<Service>_<Modifier>
  Examples:
    Permit_Web           — permit 80, 443
    Permit_DNS_DHCP      — permit 53, 67, 68
    Deny_Direct_Server   — deny tcp any, permit icmp
    Permit_All           — permit ip
```

---

## 6. TrustSec in Campus vs Data Center

### Campus Deployment (SD-Access)

```
Campus characteristics:
  - Dynamic endpoints: laptops, phones, IoT, guests
  - SGT assignment: primarily 802.1X and MAB via ISE
  - Propagation: inline CMD on fabric (VXLAN-GPO in SD-Access)
  - Enforcement: fabric edge (access switches)
  - Scale: thousands of endpoints, 10-50 SGTs typical

  SD-Access flow:
    1. Endpoint connects to fabric edge switch
    2. 802.1X/MAB authentication via ISE
    3. ISE assigns SGT via RADIUS attribute
    4. Fabric edge encapsulates in VXLAN with GPO (SGT)
    5. Traffic traverses fabric underlay
    6. Egress fabric edge decapsulates, enforces SGACL
    7. Delivers to destination (CMD stripped)

  Key integration: Catalyst Center (DNA Center)
    - Automates fabric provisioning
    - Maps scalable groups → SGTs
    - Defines access contracts → SGACLs
    - Pushes policy to ISE → devices
```

### Data Center Deployment

```
Data center characteristics:
  - Static/semi-static workloads: servers, VMs, containers
  - SGT assignment: primarily static IP-SGT or VLAN-SGT
  - Propagation: inline CMD on Nexus fabric, or SXP to firewalls
  - Enforcement: Nexus ToR/leaf switches, ASA/FTD firewalls
  - Scale: tens of thousands of IPs, 20-100 SGTs

  DC segmentation model:
    Tier 1 — Macro-segmentation: VRFs (separate routing domains)
    Tier 2 — Micro-segmentation: SGTs within VRFs

  Nexus 9000 inline tagging:
    - Supported on 9200, 9300-EX/FX/GX, 9500
    - 40G/100G ports: line-rate CMD insert/strip
    - VXLAN-GPO: carries SGT across VXLAN fabric

  Firewall integration (ASA/FTD):
    - ASA 9.x supports SGT-based rules
    - Learns IP-SGT via SXP from Nexus switches
    - Firewall rules reference SGT names/numbers
    - Example: access-list SGACL permit tcp sgt 5 dgt 100 eq 443
```

---

## 7. Migration Strategies

### Phased Migration Approach

```
Phase 1: Visibility (weeks 1-4)
  - Deploy ISE with RADIUS for 802.1X/MAB
  - Assign SGTs via authorization policy
  - NO enforcement — classification only
  - Verify SGT assignment: show cts role-based sgt-map all
  - Goal: every endpoint has an SGT

Phase 2: Monitor (weeks 5-8)
  - Define SGACLs on ISE (conservative policy)
  - Push to enforcement points in MONITOR mode
  - Analyze counters: show cts role-based counters
  - Identify false positives (legitimate traffic that would be denied)
  - Tune SGACLs

Phase 3: Limited enforcement (weeks 9-12)
  - Enable enforcement for low-risk cells first
    Example: Guest → Servers = Deny (high confidence, low impact)
  - Keep high-risk cells in monitor mode
  - Monitor for incidents / helpdesk tickets

Phase 4: Full enforcement (weeks 13-16)
  - Move remaining cells from monitor to enforce
  - Establish change management process for SGACL updates
  - Integrate with SIEM for SGT-based alerting

Phase 5: Optimization (ongoing)
  - Refine SGT granularity (split or merge groups)
  - Add new SGTs for emerging use cases (IoT, cloud)
  - Regular policy review (quarterly matrix audit)
```

### Coexistence with Traditional ACLs

```
During migration, TrustSec and traditional ACLs coexist:

Processing order on IOS-XE:
  1. Ingress port ACL (PACL)
  2. Ingress VACL (VLAN ACL)
  3. Ingress routed ACL (RACL)
  4. Routing decision
  5. Egress routed ACL (RACL)
  6. Egress VACL
  7. SGACL (role-based ACL)     ← applied last at egress
  8. Egress port ACL (PACL)

  Key insight: SGACL is evaluated AFTER traditional ACLs
  - If a traditional ACL denies traffic, SGACL is never checked
  - If a traditional ACL permits, SGACL can still deny
  - This allows gradual replacement: add SGACL, then remove traditional ACL

Migration strategy per-segment:
  1. Current: RACL permits/denies traffic between VLANs
  2. Add: SGACL in monitor mode (parallel enforcement)
  3. Verify: SGACL counters match RACL expectations
  4. Switch: SGACL to enforce mode
  5. Simplify: Replace RACL with broad permit (SGACL handles policy)
  6. Final: Remove RACL entirely (SGACL is sole enforcement)
```

### Common Migration Pitfalls

```
1. SGT 0 (Unknown) — endpoints without SGT assignment
   - All unclassified traffic gets SGT 0
   - Default policy for SGT 0 must be carefully designed
   - Too permissive: defeats segmentation purpose
   - Too restrictive: breaks unclassified legitimate traffic
   - Recommendation: start with permit, add monitoring, tighten gradually

2. IP-SGT binding staleness
   - SXP bindings persist after endpoint disconnects
   - Delete-hold-down timer (120s) delays cleanup
   - IP reuse by different endpoint → wrong SGT
   - Mitigation: use inline CMD where possible (real-time, no binding table)

3. Asymmetric path enforcement
   - SGACL enforced at egress only
   - If return traffic takes a different path, the return enforcement point
     must also have correct IP-SGT bindings
   - Mitigation: ensure all enforcement points have consistent binding tables

4. TCAM exhaustion
   - Monitor TCAM utilization before adding SGTs/SGACLs
   - Use SDM template to allocate more SGACL TCAM if needed
   - Deduplicate SGACLs (reuse same policy for multiple matrix cells)

5. SXP scaling in large networks
   - Full mesh SXP is O(N^2) connections
   - Use ISE as SXP hub (hub-and-spoke topology)
   - Consider inline tagging to eliminate SXP dependency
```

---

## See Also

- dot1x
- cisco-ise
- macsec
- acl
- zero-trust
- vxlan

## References

- Cisco TrustSec System Architecture (cisco.com)
- RFC 7343 — An Architecture for Secure Tag Exchange (SXP)
- Cisco Catalyst 9000 TrustSec Configuration Guide
- Cisco Nexus 9000 TrustSec Configuration Guide
- Cisco SD-Access Solution Design Guide
- Cisco ISE 3.x TrustSec Administration Guide
- IEEE 802.1AE (MACsec) — complementary L2 encryption
