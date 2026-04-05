# SD-Access -- Intent-Based Networking Architecture

> *SD-Access is Cisco's campus fabric implementation built on three protocol pillars: LISP separates endpoint identity (EID) from network location (RLOC) to create a scalable control plane, VXLAN encapsulates overlay traffic across an L3 underlay to eliminate spanning tree and HSRP dependencies, and Cisco TrustSec embeds Scalable Group Tags directly into VXLAN-GPO headers to enforce micro-segmentation policy without IP-based ACLs. DNA Center (now Catalyst Center) orchestrates all three planes through intent-based workflows, translating business policy into device-level configuration across hundreds of switches, access points, and wireless controllers.*

---

## 1. The Problem SD-Access Solves

Traditional campus networks suffer from fundamental architectural limitations that do not scale:

**Layer 2 sprawl.** VLANs span across access, distribution, and sometimes core layers. Spanning Tree Protocol (STP) blocks redundant paths to prevent loops, wasting bandwidth and creating complex failure domains. A single misconfigured trunk can cause a broadcast storm affecting thousands of endpoints.

**Static segmentation.** Security boundaries are defined by VLAN membership and IP-based ACLs. Moving a user between buildings means changing their VLAN, IP address, DHCP scope, and updating dozens of ACLs across multiple devices. There is no concept of identity-based policy — access is tied to network location, not user or device identity.

**Manual provisioning.** Each switch requires individual CLI configuration: VLANs, trunking, STP priorities, HSRP, DHCP relay, ACLs, QoS. A campus with 500 switches has 500 individual configurations to maintain, drift-check, and audit. Change windows are measured in weeks, not minutes.

**Limited mobility.** When a wireless client roams between access points connected to different switches, the network must re-learn the client's MAC, re-assign VLANs, and potentially change IP addresses. True seamless mobility requires stretching VLANs across the campus, which exacerbates the L2 sprawl problem.

SD-Access addresses all four problems simultaneously by introducing a fabric overlay that decouples identity from location, policy from topology, and provisioning from individual device configuration.

---

## 2. LISP Protocol Deep Dive

### 2.1 The Identity-Location Split

LISP (Locator/ID Separation Protocol), defined originally in RFC 6830 and revised in RFC 9300, introduces a fundamental separation:

- **EID (Endpoint Identifier):** The identity of an endpoint. In SD-Access, this is the host's MAC address, IPv4 address, or IPv6 address. EIDs are what applications and users care about — they should remain stable regardless of where the endpoint physically connects.

- **RLOC (Routing Locator):** The network location where an endpoint is currently attached. In SD-Access, this is the loopback IP address of the edge node (VTEP) where the host connects. RLOCs change when endpoints move; EIDs do not.

This separation means the underlay routing table only carries RLOC addresses (loopbacks of fabric nodes), not individual host routes. A fabric with 25,000 endpoints and 100 edge nodes has only approximately 100 routes in the underlay, not 25,000. The EID-to-RLOC mapping is maintained in a separate database — the LISP Map-Server.

### 2.2 Mapping System Architecture

The LISP mapping system in SD-Access consists of two logical functions, typically co-located on the same device (the control plane node):

**Map-Server (MS):**

The Map-Server accepts registrations from edge nodes and maintains the authoritative EID database. When an edge node learns a new endpoint (via ARP, DHCP snooping, or data-plane learning), it sends a Map-Register message to the MS containing:

```
Map-Register message fields:
  Nonce:          Random value for anti-replay
  Key-ID:         Authentication key identifier
  Auth-Data:      HMAC-SHA-256 of the message (mutual authentication)
  Record TTL:     How long the mapping is valid (typically 1440 minutes)
  EID-Prefix:     The endpoint's address (e.g., 10.1.1.100/32)
  RLOC:           The edge node's loopback (e.g., 192.168.1.1)
  Priority:       0-255 (lower = preferred; 255 = not used for forwarding)
  Weight:         Load balancing weight (0-100) among equal-priority RLOCs
  Instance-ID:    VN/VRF identifier (16-bit, maps to VXLAN VNI)
```

The MS stores this mapping and acknowledges with a Map-Notify message. Edge nodes refresh their registrations every 60 seconds. If a registration is not refreshed within the TTL, the MS removes the mapping.

**Map-Resolver (MR):**

The Map-Resolver handles lookup requests from edge nodes that need to forward traffic to an unknown destination. When an edge node (acting as ITR) receives traffic for an EID not in its local map-cache, it sends a Map-Request to the MR:

```
Map-Request resolution flow:

  1. ITR (Edge Node A) -> Map-Request -> MR
     "Where is EID 10.1.2.200?"

  2. MR looks up the EID in the MS database
     Found: 10.1.2.200 registered by ETR at RLOC 192.168.1.2

  3. MR forwards the Map-Request directly to the registering ETR (192.168.1.2)
     (This is the "proxy Map-Request" behavior in SD-Access)

  4. ETR (Edge Node B) -> Map-Reply -> ITR (Edge Node A)
     "10.1.2.200 is at RLOC 192.168.1.2, priority 1, weight 100"

  5. ITR caches the mapping (TTL = 60 min) and begins VXLAN encapsulation
```

This indirect resolution through the ETR (rather than the MR directly answering) ensures that the most current mapping is always returned, because the ETR holds the authoritative registration.

### 2.3 LISP Instance IDs and Multi-Tenancy

LISP Instance IDs provide namespace isolation for multi-tenancy. Each Virtual Network (VN) in SD-Access maps to a unique Instance ID, which in turn maps to a VXLAN VNI:

```
Mapping chain:
  VN "Employee"  ->  LISP Instance-ID 4097  ->  L3 VNI 4097  ->  VRF EMPLOYEE
  VN "Guest"     ->  LISP Instance-ID 4098  ->  L3 VNI 4098  ->  VRF GUEST
  VN "IoT"       ->  LISP Instance-ID 4099  ->  L3 VNI 4099  ->  VRF IOT

Each instance ID maintains a completely separate EID database:
  Instance 4097:  10.1.1.100 -> RLOC 192.168.1.1
  Instance 4098:  10.1.1.100 -> RLOC 192.168.1.3  (same EID, different VN = OK)
```

This allows overlapping IP address spaces across VNs without conflict, because EID lookups are always scoped to a specific Instance ID.

### 2.4 Proxy Tunnel Routers (PxTR)

When fabric endpoints need to communicate with non-LISP networks (the internet, a legacy campus, a data center), Proxy Tunnel Routers bridge the gap:

**PITR (Proxy Ingress Tunnel Router):**
- Accepts traffic from non-LISP sources destined for fabric EIDs
- Performs Map-Request to find the EID's RLOC
- Encapsulates in VXLAN and forwards into the fabric
- Typically runs on the border node

**PETR (Proxy Egress Tunnel Router):**
- Accepts VXLAN-encapsulated traffic from fabric edge nodes destined for non-LISP prefixes
- Decapsulates and forwards using standard IP routing
- Used when the edge node has no direct route to the destination
- Border node advertises a "negative map-reply" or default mapping pointing to itself as PETR

```
Non-LISP to Fabric (PITR):
  External host -> Border (PITR) -> LISP lookup -> VXLAN to Edge -> Host

Fabric to Non-LISP (PETR):
  Host -> Edge -> No LISP mapping found -> VXLAN to Border (PETR) -> IP routing -> External
```

### 2.5 Solicit Map-Request (SMR) and Mobility

When a mapping changes (endpoint moves, or is deregistered), the Map-Server must notify all nodes that have cached the old mapping. LISP uses the Solicit Map-Request (SMR) mechanism:

```
Mobility event — Host A moves from EN1 to EN2:

  Time T0:  Host A registered at EN1 (RLOC = 192.168.1.1)
            EN3 has cached mapping: Host A -> 192.168.1.1

  Time T1:  Host A disconnects from EN1, connects to EN2
            EN2 detects Host A (link-up, ARP, DHCP)
            EN2 sends Map-Register: Host A -> RLOC 192.168.1.2

  Time T2:  MS updates its database
            MS sends SMR to EN1 (previous registrant)
            EN1 removes Host A from its local registration

  Time T3:  Any node with a stale cache (e.g., EN3) sends traffic to old RLOC (EN1)
            EN1 does not have Host A -> drops or sends Map-Request
            EN3's cache eventually times out (TTL-based)
            OR: MS can proactively send SMR to nodes with known cached entries

Convergence time: < 1 second for the MS update
                  Up to 60 seconds for cache refresh (worst case)
                  Sub-second if SMR is sent proactively to cached nodes
```

---

## 3. VXLAN Data Plane in SD-Access

### 3.1 Standard VXLAN vs VXLAN-GPO

Standard VXLAN (RFC 7348) provides a 24-bit VNI field for network segmentation, supporting up to 16 million logical networks. SD-Access extends VXLAN with the Group Policy Option (GPO) to carry Scalable Group Tags inline:

```
Standard VXLAN header (8 bytes):
  Bits:  0                   1                   2                   3
         0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
        |R|R|R|R|I|R|R|R|            Reserved                           |
        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
        |                VXLAN Network Identifier (VNI) |   Reserved    |
        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

VXLAN-GPO header (SD-Access extension):
        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
        |R|R|R|R|I|R|R|G|        Group Policy ID (SGT)                  |
        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
        |                VXLAN Network Identifier (VNI) |   Reserved    |
        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

  G bit = 1:  Group Policy ID field contains a valid SGT
  SGT:        16-bit Scalable Group Tag (0-65535)
  VNI:        24-bit VXLAN Network Identifier
```

The VXLAN-GPO extension is critical because it allows SGT to travel with every data packet across the fabric without requiring a separate control-plane protocol (like SXP) to distribute SGT-to-IP bindings.

### 3.2 Encapsulation Stack

A complete SD-Access data frame traversing the fabric looks like this:

```
+------------------+--------------------------------------------------+
| Layer            | Content                                          |
+------------------+--------------------------------------------------+
| Outer Ethernet   | Src MAC: EN1 uplink                              |
| (14 bytes)       | Dst MAC: Next-hop router MAC                     |
|                  | EtherType: 0x0800 (IPv4)                         |
+------------------+--------------------------------------------------+
| Outer IP         | Src IP: EN1 loopback (RLOC) 192.168.1.1          |
| (20 bytes)       | Dst IP: EN2 loopback (RLOC) 192.168.1.2          |
|                  | Protocol: UDP (17)                                |
+------------------+--------------------------------------------------+
| Outer UDP        | Src Port: Entropy (hash of inner headers)        |
| (8 bytes)        | Dst Port: 4789 (VXLAN well-known)                |
+------------------+--------------------------------------------------+
| VXLAN-GPO        | VNI: 8192 (L2 VNI for this VLAN)                 |
| (8 bytes)        | SGT: 3 (Developers group)                        |
+------------------+--------------------------------------------------+
| Inner Ethernet   | Src MAC: Host A                                  |
| (14 bytes)       | Dst MAC: Host B (or anycast gateway MAC)         |
+------------------+--------------------------------------------------+
| Inner IP         | Src IP: 10.1.1.100 (Host A EID)                  |
| (20+ bytes)      | Dst IP: 10.1.2.200 (Host B EID)                  |
+------------------+--------------------------------------------------+
| Inner Payload    | Application data (TCP/UDP + payload)             |
+------------------+--------------------------------------------------+

Total overhead: 14 + 20 + 8 + 8 + 14 = 64 bytes
Original frame: assumed 14-byte Ethernet header already present
Net additional overhead: 50 bytes
```

### 3.3 L2 VNI vs L3 VNI Forwarding

SD-Access uses two types of VNI for different forwarding decisions:

**L2 VNI (intra-subnet, bridging):**

When Host A (10.1.1.100) communicates with Host B (10.1.1.200) in the same VLAN/subnet but on different edge nodes:

```
1. Host A sends frame to Host B (same subnet, knows MAC via ARP)
2. EN1 looks up Host B's MAC in LISP map-cache (L2 instance)
3. EN1 encapsulates in VXLAN with L2 VNI (e.g., 8192)
4. EN2 receives, decapsulates, delivers to Host B on local VLAN
5. This is pure L2 bridging over the VXLAN overlay
```

**L3 VNI (inter-subnet, routing):**

When Host A (10.1.1.100) communicates with Host C (10.1.2.100) in a different subnet but same VN:

```
1. Host A sends frame to its default gateway (anycast gateway on EN1)
2. EN1 routes the packet: looks up 10.1.2.100 in LISP map-cache (L3 instance)
3. EN1 encapsulates in VXLAN with L3 VNI (e.g., 4097 for the VN)
4. Outer IP: EN1 RLOC -> EN2 RLOC
5. Inner Ethernet: Src = anycast MAC, Dst = Host C MAC
6. EN2 receives, decapsulates, delivers to Host C on local VLAN
7. This is distributed L3 routing — the first-hop edge node routes directly
```

The L3 VNI approach means inter-subnet traffic never needs to traverse a centralized router or firewall (unless crossing VN boundaries). Every edge node is a first-hop router for all subnets in the VN, eliminating the traditional traffic trombone through distribution/core layer gateways.

### 3.4 ARP Suppression

In traditional networks, ARP broadcasts flood across all ports in a VLAN. In SD-Access, edge nodes suppress unnecessary ARP traffic:

```
ARP suppression flow:
  1. Host A sends ARP request: "Who has 10.1.1.200?"
  2. EN1 intercepts the ARP broadcast (does not flood to fabric)
  3. EN1 checks LISP map-cache for 10.1.1.200
  4. If found: EN1 generates ARP reply on behalf of Host B
     (using Host B's MAC from the LISP database)
  5. Host A receives ARP reply without any broadcast crossing the fabric
  6. If NOT found: EN1 sends LISP Map-Request, then replies when resolved

Impact:
  - Broadcast domain is limited to a single edge node
  - No BUM (Broadcast, Unknown unicast, Multicast) flooding across VXLAN
  - Scales to thousands of endpoints without broadcast storms
  - Reduces underlay bandwidth consumption significantly
```

### 3.5 Multi-Destination Traffic (BUM Handling)

For traffic that must reach multiple destinations (broadcast, unknown unicast, multicast), SD-Access uses head-end replication at the edge node:

```
Head-end replication:
  1. Host sends a broadcast frame (e.g., ARP for unknown host, DHCP discover)
  2. Edge node cannot suppress (target unknown)
  3. Edge node replicates the frame and sends VXLAN-encapsulated copy
     to every other edge node in the same L2 VNI
  4. Uses a fabric-wide multicast group OR unicast replication list
  5. Each receiving edge node decapsulates and delivers locally

Replication modes:
  - Ingress replication (unicast): Edge sends N copies to N peers
    Simpler, no multicast needed in underlay, but O(N) copies
  - Underlay multicast: Edge sends 1 copy to multicast group
    More efficient for large fabrics, requires PIM in underlay
    DNA Center auto-configures underlay multicast when selected
```

---

## 4. Scalable Group Tags (SGT) and Policy Enforcement

### 4.1 SGT Assignment Mechanisms

SGTs can be assigned through multiple methods, prioritized as follows:

```
Assignment priority (highest to lowest):
  1. VLAN-SGT mapping (static, on the edge node)
  2. Subnet-SGT mapping (static, for known IP ranges)
  3. Port-SGT mapping (static, for dedicated device ports)
  4. ISE dynamic assignment (802.1X/MAB authorization result)
  5. IP-SGT binding via SXP (learned from upstream devices)
  6. Default SGT (catch-all for unclassified traffic)

ISE dynamic assignment (most common in SD-Access):
  ISE Authorization Profile:
    Access Type:     ACCESS_ACCEPT
    VLAN:            Dynamic (fabric-assigned)
    Virtual Network: Employee
    Scalable Group:  Developers (SGT=3)
    dACL:            (optional per-session ACL)
```

### 4.2 SGACL Enforcement Model

SGACL enforcement in SD-Access follows a destination-based model:

```
Enforcement decision point:

  Source (EN1)                          Destination (EN2)
  +----------+     VXLAN-GPO           +----------+
  | Assign   | ----SGT=3----------->  | Enforce  |
  | SGT=3    |     VNI=4097           | SGACL    |
  +----------+                         +----------+
  Developers                           Servers (SGT=7)

  EN2 checks policy matrix:
    Source SGT = 3 (Developers)
    Destination SGT = 7 (Servers)
    SGACL = permit tcp 22,443,8080-8090; deny ip

  If packet matches "permit" -> forward to server
  If packet matches "deny"   -> drop + log
```

The destination-based enforcement model means the egress edge node makes the policy decision. This is efficient because:

- The edge node knows the destination SGT (it assigned it to the locally connected endpoint)
- The source SGT arrives in the VXLAN-GPO header (no additional lookup needed)
- SGACLs are downloaded to each edge node from ISE via CoA (Change of Authorization)

### 4.3 Policy Matrix Design Principles

Designing an effective SGACL matrix requires careful planning:

```
Design approach:
  1. Start with a default-deny posture (all SGT-to-SGT pairs = deny)
  2. Identify required communication flows:
     - Business application dependencies
     - Infrastructure services (DHCP, DNS, NTP, AD)
     - Management traffic (SSH, SNMP, syslog)
  3. Create SGACLs that permit only required protocols
  4. Group endpoints by role and trust level, not department:
     - "Developers" and "Employees" share some access but differ in dev tool access
     - "IoT_Sensors" and "IoT_Cameras" may need different server access
  5. Use a "Shared_Services" SGT for infrastructure (DNS, DHCP, AD)
     that most groups can reach
  6. Test in monitor mode before enforcement

Common mistakes:
  - Too many SGTs (>50): Creates an unmanageable NxN matrix
  - Too few SGTs (<5):   Provides insufficient granularity
  - SGTs based on org chart: Organizational changes break policy
  - Permitting ICMP everywhere: Creates a covert channel
  - Forgetting infrastructure flows: Breaks DHCP, DNS, AD auth
```

### 4.4 SGT Scalability Considerations

```
SGT matrix size:
  N SGTs -> N x N possible policy cells
  20 SGTs -> 400 cells (manageable)
  50 SGTs -> 2,500 cells (complex but feasible)
  100 SGTs -> 10,000 cells (extremely difficult to manage)

TCAM impact:
  Each unique SGACL contract consumes TCAM entries on the edge node
  Catalyst 9300: ~3,000 SGACL TCAM entries
  Catalyst 9500: ~8,000 SGACL TCAM entries
  Shared contracts reduce consumption (same ACL for multiple SGT pairs)

Best practice: Keep SGT count under 50 for most deployments
               Use shared SGACL contracts where possible
               Monitor TCAM utilization: show platform hardware tcam utilization
```

---

## 5. DNA Center (Catalyst Center) Internals

### 5.1 Architecture

DNA Center runs as a cluster of 1 or 3 physical or virtual appliances:

```
DNA Center cluster architecture:
  +--------------------------------------------------+
  | Kubernetes (container orchestration)              |
  |  +------------+  +------------+  +------------+  |
  |  | APIC-EM    |  | Assurance  |  | Identity   |  |
  |  | (NBI/SBI)  |  | (Telemetry)|  | (ISE/pxGrid)|  |
  |  +------------+  +------------+  +------------+  |
  |  +------------+  +------------+  +------------+  |
  |  | PnP Server |  | SWIM       |  | IPAM       |  |
  |  | (ZTP)      |  | (Images)   |  | (IP pools) |  |
  |  +------------+  +------------+  +------------+  |
  |  +---------------------------------------------+ |
  |  | PostgreSQL  | Elasticsearch | Kafka | Redis  | |
  |  +---------------------------------------------+ |
  +--------------------------------------------------+

Appliance requirements (physical):
  CPU:     56+ cores (Xeon Gold)
  RAM:     256 GB
  Storage: 2.4 TB SSD (RAID)
  Network: 4x 10G interfaces (cluster, management, inband, services)

3-node cluster:
  - Active-active-active for API and services
  - Data replicated across all 3 nodes
  - Survives single-node failure without downtime
  - Required for production deployments (single-node = lab only)
```

### 5.2 Southbound Interfaces

DNA Center communicates with network devices through multiple protocols:

```
Protocol usage:
  NETCONF/YANG:  Primary config push mechanism (structured, transactional)
  CLI (SSH):     Fallback for features not yet modeled in YANG
  SNMP:          Device discovery, basic monitoring
  SWIM:          Software image distribution (SCP/SFTP-based)
  PnP:           Zero-touch provisioning for new devices
  Streaming telemetry: gRPC dial-in/dial-out for real-time metrics
  Syslog:        Event collection for Assurance analytics
  NetFlow/IPFIX: Traffic flow analysis
```

### 5.3 Intent-Based API (Northbound)

DNA Center exposes a comprehensive REST API for automation:

```
API categories:
  /dna/intent/api/v1/site              Site hierarchy management
  /dna/intent/api/v1/network-device    Device inventory and management
  /dna/intent/api/v1/client-health     Client health metrics
  /dna/intent/api/v1/topology          Network topology visualization
  /dna/intent/api/v1/template          Configuration templates
  /dna/intent/api/v1/sda/              SD-Access fabric operations

Example: Add a device to the fabric as an edge node
  POST /dna/intent/api/v1/business/sda/edge-device
  Body:
    {
      "deviceManagementIpAddress": "10.0.0.50",
      "siteNameHierarchy": "Global/NA/HQ/Floor1"
    }

Example: Create a Virtual Network
  POST /dna/intent/api/v1/business/sda/virtual-network
  Body:
    {
      "virtualNetworkName": "IoT_Network",
      "isGuestVirtualNetwork": false
    }

Authentication: Token-based (POST /dna/system/api/v1/auth/token)
Rate limiting: 5 requests/second per client (default)
Async operations: Long-running tasks return a taskId for polling
```

---

## 6. Fabric-Enabled Wireless Deep Dive

### 6.1 Control Plane vs Data Plane Split

Traditional wireless architectures centralize both control and data at the WLC. SD-Access fabric wireless splits these planes:

```
Traditional (centralized):
  Client -> AP -> CAPWAP tunnel -> WLC -> Switch -> Network
  ALL traffic traverses the WLC (bottleneck at scale)
  WLC bandwidth = total fabric wireless throughput

Fabric-enabled wireless:
  Control plane:
    Client auth -> AP -> CAPWAP -> WLC -> ISE
    Roaming events, client state, RF management via CAPWAP

  Data plane:
    Client data -> AP -> VXLAN -> Edge Node (local switch) -> Fabric
    Data traffic goes directly to the edge node
    WLC is NOT in the data path

  Result:
    WLC handles control only (lightweight, scales to 6000+ APs)
    Data throughput limited by edge node capacity, not WLC
    SGT assigned at the edge node (same as wired clients)
    Consistent policy for wired and wireless endpoints
```

### 6.2 AP-to-Edge VXLAN Tunnel

In fabric mode, the AP establishes a VXLAN tunnel directly to the edge node it is physically connected to:

```
AP VXLAN behavior:
  1. AP boots, discovers WLC via CAPWAP (traditional discovery)
  2. WLC assigns AP to a fabric site and configures fabric mode
  3. AP learns its connected edge node IP (via WLC or DHCP option)
  4. AP establishes VXLAN tunnel to edge node's VTEP IP
  5. Client associates to SSID, WLC handles 802.1X via CAPWAP
  6. ISE returns VN + SGT for the client
  7. WLC signals edge node: "Client X is on VN Y with SGT Z"
  8. AP encapsulates client data frames in VXLAN to edge node
  9. Edge node decapsulates, applies SGT, re-encapsulates in fabric VXLAN
  10. Client is now part of the fabric — same as a wired host

AP VXLAN tunnel details:
  Source IP:    AP management IP
  Destination:  Edge node loopback (VTEP)
  VNI:         L2 VNI for the client's VLAN
  UDP port:    4789
  SGT:         NOT carried in AP-to-edge VXLAN (edge node adds it)
```

### 6.3 Roaming in Fabric Wireless

```
Intra-site roaming (same fabric site, different AP/edge node):
  1. Client moves from AP1 (on EN1) to AP2 (on EN2)
  2. AP2 sends reassociation to WLC via CAPWAP
  3. WLC updates client state (new AP, potentially new edge node)
  4. WLC signals EN2: "Client X is now here, VN Y, SGT Z"
  5. EN2 sends LISP Map-Register (updates client EID -> new RLOC)
  6. Control plane node (MS) updates mapping, sends SMR to EN1
  7. Seamless roaming — no IP change, no re-authentication (PMK caching)

Inter-site roaming (different fabric sites):
  - Requires L2 extension or subnet stretching between sites (not recommended)
  - OR: Client gets new IP via DHCP at new site (L3 roaming)
  - L3 roaming is cleaner but requires application tolerance for IP change
  - Mobile devices handle L3 roaming well (MPTCP, QUIC)
```

---

## 7. ISE as the Policy Engine

### 7.1 Authentication Methods in SD-Access

```
802.1X (certificate or credential-based):
  - EAP-TLS: Mutual certificate authentication (most secure)
    Client presents certificate, ISE validates against CA
    ISE presents certificate, client validates
    No password exposure, machine + user auth possible
  - PEAP-MSCHAPv2: Password-based (most common for managed devices)
    ISE presents certificate (server auth)
    Client sends username/password (encrypted tunnel)
  - EAP-FAST: Cisco proprietary, similar to PEAP but with PAC
    Used for legacy devices that cannot do TLS

MAB (MAC Authentication Bypass):
  - For devices that cannot do 802.1X (printers, cameras, IoT)
  - Edge node sends MAC address as the username/password to ISE
  - ISE checks MAC against endpoint database (static or profiled)
  - Less secure: MAC addresses can be spoofed
  - Mitigated by profiling (ISE checks DHCP fingerprint, CDP/LLDP data)

Web Authentication:
  - For guest users and BYOD
  - Edge node redirects HTTP to ISE guest portal
  - User enters credentials or sponsor approval
  - ISE pushes CoA (Change of Authorization) with VN/SGT after auth
```

### 7.2 Profiling and Posture

```
ISE Profiler:
  Data sources:
    - DHCP:    Option 55 (Parameter Request List), hostname, vendor class
    - CDP/LLDP: Platform string, capabilities, model
    - HTTP:    User-Agent header (OS type, version)
    - RADIUS:  Calling-Station-ID attributes
    - SNMP:    Device MIB queries (CDP neighbors, MAC table)
    - NetFlow: Traffic patterns (ports, protocols)
    - pxGrid:  MDM integration (Intune, Jamf, Workspace ONE)

  Profiling output:
    Device type:     "Apple-iPhone-15"
    Certainty:       90 (out of 100)
    Policy action:   Assign SGT "Mobile_Devices" (SGT=12)

ISE Posture:
  Checks performed on managed endpoints:
    - Antivirus installed and updated (definition age < 7 days)
    - OS patches current (critical patches within 30 days)
    - Firewall enabled
    - Disk encryption active (BitLocker, FileVault)
    - USB storage disabled (DLP compliance)

  Posture outcomes:
    Compliant:     Assigned to production VN + SGT
    Non-compliant: Assigned to Quarantine VN + restricted SGT
    Unknown:       Assigned to limited-access VN (posture agent not installed)
```

### 7.3 pxGrid Context Sharing

```
pxGrid architecture:
  +----------+          +-----------+          +-----------+
  | ISE      |--pxGrid->| DNA Center|--pxGrid->| Firewall  |
  | (pub)    |          | (sub)     |          | (sub)     |
  +----------+          +-----------+          +-----------+

  ISE publishes:
    - Session data: IP, MAC, username, SGT, posture state
    - SGT definitions: SGT number -> name mapping
    - Endpoint profiles: device type classification

  DNA Center subscribes:
    - Updates fabric policy with real-time SGT bindings
    - Feeds Assurance analytics with user/device context

  Third-party subscribers:
    - Palo Alto: Uses SGT for firewall policy (Dynamic Address Groups)
    - Splunk: Enriches security events with user/device context
    - ServiceNow: Automates incident creation based on posture changes
    - CrowdStrike: Correlates endpoint threat data with network SGT

  Protocol: WebSocket (STOMP over WSS)
  Authentication: Certificate-based mutual TLS
  Scalability: 50+ subscribers per ISE deployment
```

---

## 8. Transit Options In Depth

### 8.1 SDA Transit (LISP/VXLAN Between Sites)

SDA transit extends the fabric overlay between sites, preserving full LISP/VXLAN/SGT semantics:

```
Site A                    Transit                    Site B
+--------+         +------------------+         +--------+
| EN-A   |--VXLAN--| Border-A |--VXLAN--| Border-B |--VXLAN--| EN-B   |
+--------+         +----------+         +----------+         +--------+
                         |                   |
                    Transit Control Plane Node
                    (shared MS/MR for both sites)

Requirements:
  - VXLAN-capable WAN (dark fiber, MPLS with large MTU, or SD-WAN)
  - Shared or federated control plane nodes
  - Consistent VNI allocation across sites
  - MTU >= 9100 on all transit links

Benefits:
  - SGT preserved end-to-end (VXLAN-GPO across transit)
  - Seamless mobility between sites (LISP handles EID re-registration)
  - Unified policy enforcement (same SGACL matrix across sites)
  - No SGT-to-ACL translation needed at borders

Limitations:
  - WAN must support jumbo frames (VXLAN overhead)
  - Higher WAN bandwidth consumption (VXLAN overhead on every packet)
  - More complex troubleshooting (VXLAN encapsulation across WAN)
  - Not suitable for internet-based WAN (MTU constraints)
```

### 8.2 IP Transit (Standard Routing Between Sites)

```
Site A                    WAN                       Site B
+--------+         +----------+    BGP/OSPF   +----------+         +--------+
| EN-A   |--VXLAN--| Border-A |---IP routing---| Border-B |--VXLAN--| EN-B   |
+--------+         +----------+               +----------+         +--------+

Border handoff:
  - Border-A decapsulates VXLAN, routes as standard IP to Border-B
  - VRF-to-VRF peering preserves VN isolation (one BGP session per VRF)
  - SGT is stripped at Border-A (VXLAN header removed)

SGT re-mapping options across IP transit:
  1. SXP (SGT Exchange Protocol):
     Border-A sends SGT-IP bindings to Border-B via TCP session
     Border-B re-tags incoming traffic with correct SGT
     Limitation: SXP is a control-plane protocol — delay between binding update
     and enforcement; not real-time

  2. Inline CTS tagging (802.1AE MACsec):
     Requires CTS-capable WAN devices at both ends
     SGT carried in Ethernet CMD field (MACsec encrypted)
     Real-time, no control-plane delay
     Limitation: requires MACsec-capable WAN interfaces

  3. No SGT (accept the gap):
     Traffic between sites is not micro-segmented
     VN isolation (VRF) still maintained via VRF-aware BGP
     Macro-segmentation preserved, micro-segmentation lost
     Acceptable if inter-site traffic passes through a firewall
```

### 8.3 SD-WAN Integration

```
SD-Access + SD-WAN integration:
  Site A Fabric                SD-WAN                    Site B Fabric
  +--------+    +----------+  +--------+  +----------+  +--------+
  | EN-A   |--->| Border-A |--->| vEdge-A|--->| vEdge-B|--->| Border-B |--->| EN-B   |
  +--------+    +----------+  +--------+  +----------+  +----------+    +--------+

  VRF handoff:
    - SD-Access border exports VRF routes to SD-WAN edge
    - SD-WAN carries VRF traffic across the WAN (OMP per-VRF routing)
    - SD-WAN edge at remote site imports routes into remote SD-Access border
    - VN isolation preserved through VRF-to-VPN-to-VRF mapping

  SGT across SD-WAN:
    - Cisco SD-WAN (Viptela) supports inline CTS tagging
    - SGT embedded in SD-WAN IPsec tunnel headers
    - End-to-end SGT preservation: SD-Access -> SD-WAN -> SD-Access
    - Requires matching CTS configuration on SD-WAN edges
```

---

## 9. Migration Strategies and Coexistence

### 9.1 Migration Assessment

Before migrating to SD-Access, assess the current environment:

```
Assessment checklist:
  Infrastructure:
    [ ] All access switches Catalyst 9000 series (or compatible)
    [ ] All inter-switch links capable of jumbo MTU (9100+)
    [ ] No dependency on STP features (root guard, loop guard, BPDU filter)
    [ ] Existing WLC is Catalyst 9800 (or can be upgraded/replaced)
    [ ] ISE deployed with device administration and endpoint auth

  Network design:
    [ ] Current VLAN-to-subnet mapping documented
    [ ] IP address pools inventoried (DHCP scopes, static assignments)
    [ ] ACLs documented and mapped to communication requirements
    [ ] Wireless SSIDs and security settings documented
    [ ] WAN/DC connectivity and routing documented

  Endpoint inventory:
    [ ] Percentage of 802.1X-capable endpoints known
    [ ] IoT devices cataloged (printers, cameras, sensors, BMS)
    [ ] BYOD policy defined (allowed devices, onboarding method)
    [ ] Static IP devices identified (servers, printers, medical devices)

  Organizational:
    [ ] Network team trained on SD-Access (DNACIE certification path)
    [ ] Change management process supports phased rollout
    [ ] Rollback plan documented for each migration phase
    [ ] Monitoring and alerting updated for fabric-specific metrics
```

### 9.2 Brownfield Migration Patterns

```
Pattern 1: Parallel fabric (recommended for large campuses)
  - Build new fabric underlay alongside existing network
  - Migrate one IDF/closet at a time
  - Border node connects fabric to legacy network via L3 handoff
  - Users continue working during migration (IP changes via DHCP renewal)
  - Legacy VLANs decommissioned after all endpoints migrate

Pattern 2: In-place conversion (smaller sites, less disruption tolerance)
  - Convert existing switches to fabric role one at a time
  - Requires switches that support both legacy and fabric mode
  - More complex: must handle mixed-mode operation
  - Higher risk: failure affects production traffic

Pattern 3: Greenfield island (new buildings/floors)
  - Deploy fabric from scratch in new construction
  - Connect to existing network via border node
  - No migration needed for new space
  - Gradually expand fabric as existing hardware is refreshed

Migration timeline (typical):
  Phase 1 (Foundation):     4-8 weeks (DNA Center, ISE, underlay, pilot)
  Phase 2 (Edge expansion): 2-4 weeks per building (depends on size)
  Phase 3 (Wireless):       2-4 weeks (WLC upgrade, SSID migration)
  Phase 4 (Policy):         4-8 weeks (SGT design, monitor, enforce)
  Phase 5 (Multi-site):     4-8 weeks per additional site
  Total:                    4-12 months for a large campus (1000+ endpoints)
```

### 9.3 Rollback Planning

```
Rollback triggers:
  - Fabric control plane node failure with no redundancy
  - ISE outage causing all endpoints to lose authentication
  - DNA Center failure preventing policy updates
  - Application performance degradation traced to VXLAN overhead
  - Endpoint compatibility issues (devices failing 802.1X)

Rollback strategy per phase:
  Underlay: Keep legacy routing config in a backup; restore via console
  Edge node: Revert to access-layer VLAN config (saved in DNA Center)
  Border: Disconnect fabric, restore legacy distribution routing
  Wireless: Revert WLC to non-fabric mode, re-associate APs
  Policy: Disable SGACL enforcement (single DNA Center action)

Key rollback principle:
  Never decommission legacy infrastructure until the fabric
  has been stable for at least 30 days with full traffic load.
```

---

## 10. Troubleshooting Framework

### 10.1 Systematic Approach

```
Layer-by-layer troubleshooting:

  1. Underlay (IS-IS, MTU, BFD):
     show isis neighbors
     show isis database detail
     show ip route isis
     ping <remote-loopback> size 9000 df-bit   # Verify MTU

  2. LISP control plane (EID registration, map-cache):
     show lisp instance-id <iid> ipv4 server     # On control plane node
     show lisp instance-id <iid> ipv4 map-cache   # On edge node
     show lisp session                             # MS/MR connectivity

  3. VXLAN data plane (VTEP peers, VNI state):
     show nve peers
     show nve vni
     show vxlan vtep
     show nve interface nve1 detail

  4. CTS/SGT (tag assignment, SGACL enforcement):
     show cts role-based sgt-map all
     show cts role-based permissions
     show cts role-based counters
     show cts environment-data

  5. Authentication (ISE, 802.1X, MAB):
     show authentication sessions interface <intf>
     show authentication registrations
     test aaa group <method> <user> <pass> legacy   # Test RADIUS

  6. Wireless fabric:
     show wireless fabric summary
     show wireless fabric vnid
     show ap name <ap> config general
```

### 10.2 Common Failure Modes

```
Failure: Endpoint cannot authenticate
  Symptoms: Port in unauthorized state, no IP address
  Check: show authentication sessions interface GigE1/0/1
  Common causes:
    - ISE unreachable (check RADIUS connectivity)
    - Certificate mismatch (EAP-TLS: check CA chain)
    - MAB: MAC not in ISE endpoint database
    - Port misconfiguration (wrong auth method order)

Failure: Endpoint authenticated but no fabric connectivity
  Symptoms: Has IP, SGT assigned, but cannot reach remote hosts
  Check: show lisp instance-id <iid> ipv4 map-cache
  Common causes:
    - LISP registration failed (check MS/MR connectivity)
    - VXLAN tunnel not established (check NVE peers, MTU)
    - VNI mismatch between sites (check DNA Center VN config)
    - Underlay routing broken (check IS-IS adjacency)

Failure: Cross-VN traffic blocked
  Symptoms: Can reach same-VN hosts but not other VN hosts
  Expected: Cross-VN blocked by design unless fusion router configured
  Check: Verify fusion router/firewall has correct VRF interfaces
  Common causes:
    - Fusion router not configured for new VN
    - Firewall policy blocking inter-VN traffic
    - Missing route leaking between VRFs on fusion router

Failure: SGT policy not enforcing
  Symptoms: Traffic that should be denied is permitted
  Check: show cts role-based counters
  Common causes:
    - SGACL in monitor mode (not enforce)
    - Source SGT not assigned (check ISE authorization)
    - SGACL not downloaded to edge node (check CTS environment)
    - Contract mismatch (wrong permit/deny for SGT pair)
```

---

## Prerequisites

- VXLAN fundamentals (VNI, VTEP, encapsulation overhead)
- VLAN and inter-VLAN routing concepts
- 802.1X authentication and RADIUS protocol
- Basic understanding of overlay/underlay network architecture
- IS-IS or OSPF routing protocol fundamentals
- Familiarity with Cisco IOS-XE CLI and configuration model

## References

- [RFC 6830 — The Locator/ID Separation Protocol (LISP)](https://www.rfc-editor.org/rfc/rfc6830)
- [RFC 9300 — The Locator/ID Separation Protocol (LISP) — Revised](https://www.rfc-editor.org/rfc/rfc9300)
- [RFC 9301 — LISP Control Plane](https://www.rfc-editor.org/rfc/rfc9301)
- [RFC 7348 — Virtual eXtensible Local Area Network (VXLAN)](https://www.rfc-editor.org/rfc/rfc7348)
- [RFC 8365 — A Network Virtualization Overlay Solution Using EVPN](https://www.rfc-editor.org/rfc/rfc8365)
- [Cisco SD-Access Solution Design Guide (CVD)](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-sda-design-guide.html)
- [Cisco SD-Access Segmentation Design Guide](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-sda-segmentation-design-guide.html)
- [Cisco DNA Center User Guide](https://www.cisco.com/c/en/us/td/docs/cloud-systems-management/network-automation-and-management/dna-center/dna-center-user-guide.html)
- [Cisco TrustSec (CTS) Architecture and Configuration](https://www.cisco.com/c/en/us/td/docs/switches/lan/trustsec/configuration/guide/trustsec.html)
- [Cisco ISE Administrator Guide](https://www.cisco.com/c/en/us/td/docs/security/ise/admin-guide.html)
- [Cisco pxGrid Documentation](https://developer.cisco.com/docs/pxgrid/)
- [LISP IETF Working Group](https://datatracker.ietf.org/wg/lisp/documents/)
