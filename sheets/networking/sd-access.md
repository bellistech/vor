# SD-Access (Software-Defined Access with LISP and VXLAN)

Cisco intent-based networking architecture that uses LISP for the control plane, VXLAN for the data plane, and Cisco TrustSec (CTS/SGT) for the policy plane, all orchestrated through DNA Center (Catalyst Center) to automate campus fabric provisioning, segmentation, and assurance.

## Architecture Overview

### SD-Access Fabric Components

```
+------------------------------------------------------+
|                  DNA Center / Catalyst Center         |
|   (Design, Policy, Provision, Assurance workflows)   |
+------------------------------------------------------+
        |  REST API / PnP / NETCONF / YANG
        v
+------------------------------------------------------+
|                    Fabric Overlay                     |
|  Control Plane: LISP (EID-to-RLOC mapping)           |
|  Data Plane:    VXLAN (encapsulation + SGT tagging)   |
|  Policy Plane:  CTS / SGACLs (micro-segmentation)    |
+------------------------------------------------------+
        |
+------------------------------------------------------+
|                    Fabric Underlay                    |
|  IS-IS routed network (loopbacks, P2P links)         |
|  No STP, no HSRP — pure L3 routed underlay           |
+------------------------------------------------------+
```

### Three Planes

| Plane   | Protocol | Function                                              |
|---------|----------|-------------------------------------------------------|
| Control | LISP     | Host tracking, EID-to-RLOC mapping, mobility          |
| Data    | VXLAN    | Packet encapsulation, L2/L3 overlay, SGT propagation  |
| Policy  | CTS/SGT  | Scalable Group Tags, SGACLs, micro-segmentation       |

## Fabric Roles

### Node Roles

| Role                 | Function                                                  | Typical Platform         |
|----------------------|-----------------------------------------------------------|--------------------------|
| Control Plane Node   | Runs LISP Map-Server and Map-Resolver                     | Catalyst 9800, C9500     |
| Edge Node            | Connects endpoints, performs VXLAN encap/decap, SGT assign| Catalyst 9300/9400/9500  |
| Border Node          | Connects fabric to external networks (WAN, DC, internet)  | Catalyst 9500, ISR/ASR   |
| Intermediate Node    | Underlay routing only, forwards VXLAN-encapsulated frames | Any L3 switch/router     |
| Fabric WLC           | Integrates wireless into the fabric (CAPWAP to VXLAN)     | Catalyst 9800 WLC        |
| Extended Node        | L2 extension for IoT/legacy (connects to edge node trunk) | IE3x00, Digital Building |

### Control Plane Node

```
Functions:
  - LISP Map-Server (MS): Registers EID-to-RLOC mappings from edge nodes
  - LISP Map-Resolver (MR): Resolves EID lookups from querying edge nodes
  - Host database: Maintains MAC, IPv4, IPv6 -> RLOC mappings
  - Typically co-located MS/MR on same device
  - Redundancy: Deploy 2 control plane nodes per fabric site
```

### Edge Node

```
Functions:
  - LISP xTR (ETR + ITR): Registers local hosts, queries remote hosts
  - VXLAN Tunnel Endpoint (VTEP): Encapsulates/decapsulates overlay traffic
  - Anycast L3 gateway: Default gateway for all VLANs (distributed)
  - SGT assignment: Tags traffic based on ISE authorization
  - 802.1X / MAB / Web Auth: Authenticates endpoints via ISE
  - ARP suppression: Responds to ARP on behalf of known hosts (reduces broadcast)
```

### Border Node

```
Types of border nodes:
  Internal Border:  Connects to known networks (shared services, DC)
                    Advertises fabric EIDs into external routing (BGP/OSPF)
                    Imports external routes into fabric

  External Border:  Connects to unknown/untrusted networks (internet, WAN)
                    Default route injection into fabric
                    NAT/firewall integration point

  Anywhere Border:  Combines internal + external border (single device)
                    Common in smaller deployments

Border handoff options:
  - IP (L3) handoff:  Standard routing peering (BGP, OSPF, static)
  - SDA Transit:      LISP/VXLAN between fabric sites (preserves SGT)
  - IP Transit:       Standard IP routing between sites (loses SGT without inline tagging)
```

## LISP Protocol (Control Plane)

### Core Concepts

| Term         | Full Name                   | Description                                       |
|--------------|-----------------------------|---------------------------------------------------|
| EID          | Endpoint Identifier         | Host IP or MAC address (the identity)              |
| RLOC         | Routing Locator             | IP address of the VTEP/xTR (the location)          |
| Map-Server   | MS                          | Database that stores EID-to-RLOC registrations     |
| Map-Resolver | MR                          | Answers EID lookup queries from ITRs               |
| ITR          | Ingress Tunnel Router       | Encapsulates packets entering the overlay          |
| ETR          | Egress Tunnel Router        | Decapsulates packets leaving the overlay           |
| xTR          | Combined ITR + ETR          | Typical edge node role                             |
| PxTR         | Proxy xTR                   | Proxies LISP for non-LISP sites                    |
| PITR         | Proxy Ingress Tunnel Router | Accepts traffic from non-LISP networks into fabric |
| PETR         | Proxy Egress Tunnel Router  | Sends traffic from fabric to non-LISP networks     |
| SMR          | Solicit Map-Request         | Notification that a mapping has changed             |
| Instance ID  | IID                         | LISP namespace for VRF/VN isolation (maps to VNI)  |

### LISP Registration and Lookup Flow

```
Host A connects to Edge Node 1 (EN1):
  1. EN1 learns Host A's MAC + IP (via ARP, DHCP snooping, or data-plane learning)
  2. EN1 sends Map-Register to Control Plane Node (MS):
       EID: 10.1.1.100 -> RLOC: 192.168.1.1 (EN1 loopback), Instance-ID: 8192
  3. MS stores the mapping in its EID database

Host A sends traffic to Host B (on Edge Node 2):
  4. EN1 (ITR) sends Map-Request to MR: "Where is 10.1.2.200?"
  5. MR forwards to MS, which forwards to EN2 (the registering ETR)
  6. EN2 sends Map-Reply to EN1: "10.1.2.200 is at RLOC 192.168.1.2"
  7. EN1 caches the mapping and encapsulates traffic in VXLAN to EN2
  8. EN2 decapsulates and delivers to Host B
```

### LISP Map-Cache

```
Edge node map-cache (conceptual):
  EID-Prefix          RLOC             Instance-ID   TTL     State
  10.1.1.0/24         192.168.1.1      8192          1440    Registered (local)
  10.1.2.200/32       192.168.1.2      8192          60      Cached (remote)
  10.1.3.0/24         192.168.1.3      8192          60      Cached (remote)
  0.0.0.0/0           192.168.1.10     8192          1440    Default (border)

Map-cache timers:
  Registration TTL:  1440 min (local EIDs, refreshed every 60 sec)
  Remote TTL:        60 min (remote cached EIDs, refreshed on traffic)
  Negative TTL:      15 min (no mapping found — forward to border/PETR)
```

### Mobility with LISP

```
Host moves from EN1 to EN2:
  1. Host A appears on EN2 (new MAC learning event)
  2. EN2 sends Map-Register for Host A's EID (new RLOC = EN2)
  3. MS updates its database (RLOC changes from EN1 to EN2)
  4. MS sends SMR (Solicit Map-Request) to EN1
  5. EN1 receives SMR, sends new Map-Request for Host A
  6. EN1 receives Map-Reply with updated RLOC (now EN2)
  7. EN1 updates its map-cache — seamless, sub-second mobility
  8. No STP reconvergence, no ARP storms, no VLAN spanning
```

## VXLAN Data Plane

### VXLAN in SD-Access

```
Standard VXLAN header:
  +----+----+----+----+----+----+----+----+
  | Outer Eth | Outer IP | UDP  | VXLAN  | Inner Eth | Inner Payload |
  |  14 B     |  20 B    | 8 B  |  8 B   |  14 B     |  variable     |
  +----+----+----+----+----+----+----+----+
  Total overhead: 50 bytes (requires underlay MTU >= 9100)

SD-Access VXLAN-GPO extension (Group Policy Option):
  Standard VXLAN VNI (24 bits)  +  SGT in Group Policy ID field (16 bits)
  The VXLAN-GPO header carries the Scalable Group Tag inline
  No need for separate SGT tagging (SXP or inline 802.1AE)
```

### VNI (VXLAN Network Identifier) Mapping

```
VNI allocation in SD-Access:
  L2 VNI:  One per VLAN in the fabric (maps to VLAN broadcast domain)
           Range: 8188 - 16383 (default pool)
           Example: VLAN 10 -> L2 VNI 8192

  L3 VNI:  One per VN (Virtual Network / VRF)
           Range: 4097 - 8187 (default pool)
           Example: VN "Employee" -> L3 VNI 4097

  L2 VNI handles intra-subnet traffic (bridging within same VLAN)
  L3 VNI handles inter-subnet traffic (routing between VLANs in same VN)
```

### Anycast Gateway

```
Every edge node in the fabric:
  - Shares the same IP gateway address for each VLAN (e.g., 10.1.1.1/24)
  - Shares the same virtual MAC (e.g., 00:00:0C:9F:F0:01)
  - Host always has a local L3 gateway — no HSRP/VRRP needed
  - Inter-subnet traffic is routed locally at the first-hop edge node
  - Reduces east-west hair-pinning to zero
```

## Virtual Networks (Macro-Segmentation)

### VN Concepts

```
Virtual Network (VN):
  - Logical network partition mapped to a VRF on each fabric node
  - Each VN has its own L3 VNI, routing table, and LISP instance ID
  - Traffic between VNs is blocked by default (no route leaking)
  - Cross-VN traffic requires a firewall/fusion router at the border

Typical VN design:
  VN Name        Purpose                 L3 VNI   Instance-ID
  Employee       Corporate users         4097     4097
  Guest          Internet-only guests    4098     4098
  IoT            Sensors, cameras, BMS   4099     4099
  Voice          IP phones, UC           4100     4100
  Quarantine     Non-compliant devices   4101     4101
```

### Fusion Router (Cross-VN Routing)

```
Cross-VN traffic flow:
  VN "IoT" ---> Border Node ---> Fusion Router/Firewall ---> Border Node ---> VN "Employee"

Fusion router:
  - External router with interfaces in multiple VRFs
  - Route leaking controlled by firewall policy
  - All cross-VN traffic is inspected and filtered
  - Avoids direct VRF route leaking (which bypasses security)
```

## Scalable Group Tags (Micro-Segmentation)

### SGT Architecture

```
SGT lifecycle:
  1. Endpoint connects to edge node (wired or wireless)
  2. Edge node authenticates via ISE (802.1X, MAB, or Web Auth)
  3. ISE returns authorization: VLAN, VN, and SGT assignment
  4. Edge node tags all traffic from that endpoint with the assigned SGT
  5. SGT is carried in VXLAN-GPO header across the fabric
  6. Destination edge node enforces SGACL based on (Source SGT, Dest SGT)

SGT numbering (example):
  SGT     Name              Description
  2       Employees         Standard corporate users
  3       Developers        Engineering team
  4       Contractors       External contractors
  5       IoT_Sensors       Building sensors
  6       IoT_Cameras       Security cameras
  7       Servers           Data center servers
  8       PCI_Systems       Payment card systems
  9       Quarantine        Non-compliant endpoints
  65535   Unknown           Unauthenticated traffic
```

### SGACL Policy Matrix

```
                Dest SGT:
Src SGT:      Employees  Developers  Servers  PCI_Systems  IoT_Sensors
Employees     Permit     Permit      Permit   Deny         Deny
Developers    Permit     Permit      Permit   Deny         Permit
Contractors   Deny       Deny        Deny     Deny         Deny
IoT_Sensors   Deny       Deny        MQTT     Deny         MQTT
IoT_Cameras   Deny       Deny        HTTPS    Deny         Deny
Quarantine    Deny       Deny        Deny     Deny         Deny

SGACL example (Developers -> Servers):
  permit tcp dst eq 22           # SSH
  permit tcp dst eq 443          # HTTPS
  permit tcp dst eq 8080-8090    # Dev ports
  deny ip                        # Default deny
```

### SGT Propagation Methods

| Method         | Where Used                     | How It Works                              |
|----------------|--------------------------------|-------------------------------------------|
| VXLAN-GPO      | Within fabric (overlay)        | SGT embedded in VXLAN header (inline)     |
| Inline (CTS)   | Between CTS-capable devices    | SGT in Ethernet CMD frame (802.1AE MACsec)|
| SXP            | To non-CTS devices (legacy)    | SGT-to-IP binding via TCP control plane   |
| ISE (pxGrid)   | To third-party devices         | SGT-to-IP published via pxGrid API        |

## DNA Center / Catalyst Center

### Workflow Overview

| Workflow    | Purpose                                                        |
|-------------|----------------------------------------------------------------|
| Design      | Define sites, buildings, floors, IP pools, network settings    |
| Policy      | Create VNs, define SGT groups, build SGACL matrix              |
| Provision   | Deploy devices, assign roles, push configs to fabric nodes     |
| Assurance   | Monitor health, AI/ML analytics, path trace, issue resolution  |

### Design Phase

```
Site hierarchy:
  Global
  └── Area: North America
      └── Building: HQ-Building-1
          └── Floor: Floor-1
              - IP address pools assigned
              - DHCP/DNS servers configured
              - Wireless SSIDs mapped
              - Network profiles applied

Network settings:
  - AAA server (ISE) integration
  - DNS/DHCP server assignments
  - NTP, SNMP, syslog settings
  - Image repository (SWIM — Software Image Management)
  - Credential profiles for device access
```

### Provision Phase

```
Plug and Play (PnP) device onboarding:
  1. New switch boots with no config
  2. Discovers DNA Center via:
     - DHCP option 43 (DNA Center IP)
     - DNS lookup (_pnp._tcp.<domain>)
     - Cisco cloud redirect (devicehelper.cisco.com)
  3. DNA Center pushes initial config (credentials, AAA, underlay routing)
  4. Admin assigns fabric role (edge, border, control plane)
  5. DNA Center generates and pushes full fabric config:
     - LISP control plane settings
     - VXLAN VTEP configuration
     - VN/VRF provisioning
     - SGT/SGACL policy
     - IS-IS underlay routing

No manual CLI required — entire fabric deployed via GUI/API
```

### Assurance Phase

```
Health dashboards:
  - Network health:  Device uptime, CPU, memory, link utilization
  - Client health:   Onboarding success rate, RSSI, throughput, latency
  - Application health: Response time, packet loss, jitter per app

AI/ML analytics:
  - Baselining: Learns normal behavior for each metric
  - Anomaly detection: Flags deviations from baseline
  - Issue correlation: Groups related symptoms into root cause
  - Guided remediation: Step-by-step fix recommendations

Path Trace:
  - Visual hop-by-hop path between any two endpoints
  - Shows VXLAN encap/decap points, SGT enforcement points
  - Identifies where traffic is dropped and why
```

## Host Onboarding

### Wired Host Onboarding

```
Wired endpoint connection flow:
  1. Host plugs into edge node switchport
  2. Edge node detects link-up, initiates 802.1X or MAB
  3. ISE authenticates endpoint (certificates, credentials, MAC)
  4. ISE returns authorization:
     - VLAN (mapped to L2 VNI)
     - Virtual Network (mapped to VRF/L3 VNI)
     - Scalable Group Tag (SGT)
     - dACL (optional downloadable ACL)
  5. Edge node places host in correct VLAN/VN
  6. Edge node assigns SGT to all traffic from this port
  7. Edge node registers host EID with control plane (LISP Map-Register)
  8. Host receives IP via DHCP (from fabric-integrated DHCP)

Port configuration modes:
  - Closed mode:   Only authenticated traffic allowed (strictest)
  - Low-impact mode: Pre-auth ACL allows DHCP, DNS, TFTP; rest denied until auth
  - Open mode:     All traffic allowed; SGT still assigned after auth (least disruptive)
```

### Wireless Host Onboarding

```
Fabric-enabled wireless flow:
  1. Client associates to SSID on fabric-mode AP
  2. AP tunnels control traffic (802.1X) to Fabric WLC via CAPWAP
  3. WLC proxies authentication to ISE
  4. ISE returns VN, SGT, VLAN assignment
  5. WLC signals the fabric edge node (the AP's connected switch)
  6. AP switches to local-mode VXLAN: data traffic goes directly to edge node
     (NOT through WLC — distributed data plane)
  7. Edge node performs VXLAN encapsulation with SGT
  8. Edge node registers wireless client EID with control plane

Key difference from traditional wireless:
  Traditional:  All data traffic tunnels to WLC (centralized, bottleneck)
  Fabric:       Only control traffic goes to WLC; data stays local (distributed)
```

## Fabric-Enabled Wireless

### Architecture

```
             DNA Center
                |
          Fabric WLC (9800)
           /    |    \            Control plane (CAPWAP)
         AP1   AP2   AP3
          |     |     |           Data plane (VXLAN direct to edge)
         EN1   EN1   EN2         Edge Nodes (VTEPs)
          \     |     /
        Fabric Underlay (IS-IS)

AP modes in SD-Access:
  Local mode:    Standard operation, single-site deployment
  FlexConnect:   Used with extended nodes for remote offices
  Bridge mode:   Mesh AP (outdoor backhaul)

SSID-to-VN mapping:
  SSID              VN              SGT (default)
  Corp-WiFi         Employee        2 (Employees)
  Guest-WiFi        Guest           65535 (Unknown)
  IoT-WiFi          IoT             5 (IoT_Sensors)
```

## ISE Integration

### ISE Role in SD-Access

```
ISE provides:
  - Authentication: 802.1X (EAP-TLS, PEAP), MAB, Web Auth
  - Authorization:  VN assignment, SGT assignment, dACL
  - Profiling:      Endpoint type identification (OS, device type)
  - Posture:        Compliance checking (AV, patches, encryption)
  - pxGrid:         Real-time SGT-IP binding publication to DNA Center
  - Guest lifecycle: Sponsor portal, self-registration, time-limited access
  - BYOD:           Certificate provisioning, onboarding portal

ISE policy flow:
  Authentication Policy --> Who are you? (identity source: AD, LDAP, cert)
  Authorization Policy  --> What do you get? (VN, SGT, VLAN, dACL)
  Posture Policy        --> Are you compliant? (quarantine if not)
  Profiler Policy       --> What type of device? (auto-classify)
```

### pxGrid Integration

```
pxGrid (Platform Exchange Grid):
  - Pub/sub messaging bus for ISE context sharing
  - DNA Center subscribes to SGT-IP binding updates
  - Third-party integration: firewalls, SIEM, MDM
  - Real-time: binding updates published within seconds of auth

pxGrid topics:
  - SessionDirectory: Active sessions (IP, MAC, user, SGT)
  - TrustSecMetaData: SGT definitions and names
  - EndpointProfile: Device profiling data
  - MDMCompliance:   Mobile device management status
```

## Transit Types

### IP Transit (Traditional)

```
Fabric Site A                              Fabric Site B
+------------+    Standard IP routing    +------------+
| Border     |<--- BGP / OSPF / Static --->| Border     |
| Node A     |    (no VXLAN, no SGT)     | Node B     |
+------------+                           +------------+

Characteristics:
  - Simple: standard routing protocols between sites
  - SGT is LOST at the border (no VXLAN-GPO across WAN)
  - SXP or inline CTS can re-map SGT at remote border (extra config)
  - VN isolation requires separate VRFs on the WAN/transit network
  - Common for brownfield or when WAN devices are not fabric-aware
```

### SDA Transit (Fabric-to-Fabric)

```
Fabric Site A                              Fabric Site B
+------------+    LISP + VXLAN          +------------+
| Border     |<--- VXLAN-GPO tunnel ----->| Border     |
| Node A     |    (SGT preserved)        | Node B     |
+------------+                           +------------+
       \                                       /
        ----> Transit Control Plane Node <----
              (shared MS/MR for both sites)

Characteristics:
  - Full fabric extension across WAN
  - SGT preserved end-to-end (VXLAN-GPO across transit)
  - VN isolation maintained natively (LISP instance IDs)
  - Requires VXLAN-capable WAN (or direct/dark fiber between sites)
  - Best for campus-to-campus within same metro
```

### SD-WAN Transit

```
Fabric Site A                              Fabric Site B
+------------+    SD-WAN overlay        +------------+
| Border     |<--- IPsec + VXLAN ------->| Border     |
| Node A     |    (via SD-WAN edges)     | Node B     |
+------------+                           +------------+

  SD-Access border peers with SD-WAN edge via VRF-aware handoff
  SD-WAN carries VRF traffic across WAN (maintains VN isolation)
  SGT propagation via SXP or inline CTS on SD-WAN edge
```

## Underlay Network

### IS-IS Design

```
Underlay requirements:
  - L3 routed: every link is a routed P2P (no L2 between switches)
  - IS-IS: single-area (Level-2), auto-provisioned by DNA Center
  - Loopbacks: /32 addresses on each node (used as RLOC/VTEP IP)
  - MTU: 9100+ on all fabric links (VXLAN overhead = 50 bytes)
  - No STP, no HSRP, no VLAN trunking on inter-switch links
  - BFD: enabled for sub-second link failure detection

Underlay IS-IS config (auto-generated by DNA Center):
  router isis
   net 49.0001.0101.0101.0001.00
   is-type level-2-only
   metric-style wide
   address-family ipv4 unicast
  interface Loopback0
   ip address 192.168.1.1 255.255.255.255
   ip router isis
  interface TenGigE1/0/1
   ip address 10.0.0.1 255.255.255.252
   ip router isis
   isis network point-to-point
```

## Migration Strategies

### Phased Migration Approach

```
Phase 1: Foundation
  - Deploy DNA Center, ISE, fabric WLC
  - Build underlay on new or existing L3 infrastructure
  - Designate control plane nodes and border nodes
  - Validate with a single-switch pilot (one edge node, one VN)

Phase 2: Edge Expansion
  - Convert access switches to fabric edge nodes (one IDF at a time)
  - Migrate users/VLANs to fabric (VN + SGT assignment)
  - Maintain legacy VLANs on non-fabric switches during transition
  - Border node provides L3 handoff to legacy network

Phase 3: Wireless Integration
  - Deploy fabric WLC (or convert existing WLC to fabric mode)
  - Migrate SSIDs to fabric-enabled mode
  - APs automatically join fabric when connected to edge nodes

Phase 4: Policy Enforcement
  - Define SGT groups based on existing ACL analysis
  - Build SGACL matrix in DNA Center (start with monitor mode)
  - Enable enforcement gradually (monitor -> enforce per policy)
  - Validate with path trace and assurance analytics

Phase 5: Multi-Site Extension
  - Connect sites via SDA transit or IP transit
  - Extend VNs and SGT policies across sites
  - Validate end-to-end segmentation and mobility
```

### Coexistence with Legacy Network

```
Brown-to-green migration:
  +-------------------+         +-------------------+
  | Legacy Network    |         | SD-Access Fabric   |
  | (STP, HSRP, ACLs)|<------->| (LISP, VXLAN, SGT) |
  +-------------------+  Border +-------------------+
                          Node
                       (L3 handoff)

Border node bridges the two worlds:
  - Fabric side: LISP/VXLAN overlay participant
  - Legacy side: Standard L3 routing (OSPF, BGP, static)
  - VLANs can be stretched temporarily via border (not recommended long-term)
  - SGT-to-ACL mapping at border for legacy enforcement
```

## Verification Commands

```bash
# LISP control plane status
show lisp site summary
show lisp instance-id <iid> ipv4 server
show lisp instance-id <iid> ipv4 map-cache

# VXLAN status
show nve peers
show nve vni
show vxlan vtep

# Fabric edge verification
show cts role-based sgt-map all
show cts role-based permissions
show cts role-based counters

# Underlay verification
show isis neighbors
show isis database detail
show ip route isis

# Wireless fabric verification
show wireless fabric summary
show ap summary
show wireless client summary

# DNA Center API (REST)
GET https://<dnac>/dna/intent/api/v1/network-device
GET https://<dnac>/dna/intent/api/v1/topology/site-topology
GET https://<dnac>/dna/intent/api/v1/client-health
```

## Scaling Guidelines

| Parameter                          | Recommended Maximum       |
|------------------------------------|---------------------------|
| Fabric sites per DNA Center        | 100                       |
| Edge nodes per fabric site         | 100                       |
| Endpoints per fabric site          | 25,000                    |
| VNs per fabric site                | 32                        |
| SGTs per deployment                | 65,535 (16-bit field)     |
| SSIDs per fabric site              | 16                        |
| APs per fabric WLC (9800)          | 6,000                     |
| Control plane nodes per site       | 2 (redundancy pair)       |
| Border nodes per site              | 2 (active/active)         |

## Tips

- Always deploy two control plane nodes per site for redundancy; a single MS/MR failure takes down all new host registrations and lookups.
- Set underlay MTU to at least 9100 on all fabric links; VXLAN adds 50 bytes and fragmentation kills fabric performance silently.
- Start SGT policy enforcement in monitor mode; use the SGACL counters and DNA Center policy analysis to validate before switching to enforce.
- Use low-impact mode for wired port authentication during migration; it allows DHCP and DNS before authentication completes, reducing helpdesk calls.
- Design VNs around security boundaries, not organizational structure; too many VNs increase complexity without improving security.
- Keep the underlay simple: IS-IS with point-to-point links, no redistribution, no route manipulation; DNA Center expects to own the underlay.
- For multi-site deployments, prefer SDA transit if you need end-to-end SGT preservation; IP transit loses SGT context at the border.
- Integrate ISE posture assessment early; quarantine VN catches non-compliant devices before they reach production resources.
- Run fabric WLC in HA SSO pair; wireless control plane failure affects all fabric APs even though data plane is distributed.
- Use DNA Center Assurance path trace before and after changes to verify traffic flows; it shows VXLAN encap points and SGT enforcement.
- Do not span VLANs across the fabric border into legacy networks long-term; stretched VLANs defeat the purpose of L3 underlay and reintroduce STP.
- Leverage pxGrid to feed SGT context to perimeter firewalls; this extends micro-segmentation beyond the fabric boundary.

## See Also

- vxlan, vlan, is-is, segment-routing, cisco-aci, sd-wan, radius, tacacs, private-vlans

## References

- [Cisco SD-Access Design Guide (CVD)](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-sda-design-guide.html)
- [LISP RFC 6830 — Locator/ID Separation Protocol](https://www.rfc-editor.org/rfc/rfc6830)
- [LISP RFC 9300 — LISP Revised](https://www.rfc-editor.org/rfc/rfc9300)
- [VXLAN RFC 7348 — Virtual eXtensible Local Area Network](https://www.rfc-editor.org/rfc/rfc7348)
- [VXLAN-GPO (Group Policy Option) — draft-smith-vxlan-group-policy](https://datatracker.ietf.org/doc/html/draft-smith-vxlan-group-policy)
- [Cisco TrustSec (CTS) Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/lan/trustsec/configuration/guide/trustsec.html)
- [Cisco DNA Center User Guide](https://www.cisco.com/c/en/us/td/docs/cloud-systems-management/network-automation-and-management/dna-center/dna-center-user-guide.html)
- [Cisco ISE Admin Guide](https://www.cisco.com/c/en/us/td/docs/security/ise/admin-guide.html)
- [Cisco SD-Access Segmentation Design Guide](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-sda-segmentation-design-guide.html)
