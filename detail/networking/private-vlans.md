# Private VLANs -- Intra-VLAN Layer 2 Isolation

> Private VLANs partition a single broadcast domain into smaller sub-domains at Layer 2,
> allowing hosts to share an IP subnet and default gateway while restricting direct
> communication between them. PVLANs solve the fundamental tension between IP address
> conservation and host isolation in shared network segments.

## 1. The Problem PVLANs Solve

Traditional VLANs provide broadcast domain isolation, but every VLAN requires its own IP
subnet. In environments with hundreds of tenants or services that all need access to the
same gateway, this approach wastes address space and complicates routing tables.

Consider a colocation provider with 200 customers on a single floor. Without PVLANs:

- 200 VLANs, each needing a /30 or /29 minimum
- 200 SVI interfaces on the router
- 200 OSPF/BGP neighbors or static routes
- Massive ACL complexity to control inter-VLAN traffic

With PVLANs:

- 1 primary VLAN, 1 IP subnet (e.g., /24)
- 1 SVI interface on the router
- Each customer port is isolated at L2
- Zero inter-customer traffic without any ACLs

The savings in address space, routing table entries, and administrative overhead are
substantial.

## 2. PVLAN Theory of Operation

### 2.1 VLAN Hierarchy

PVLANs introduce a parent-child relationship between VLANs:

```
Primary VLAN (parent)
  |
  +-- Isolated Secondary VLAN (at most one)
  |
  +-- Community Secondary VLAN A
  |
  +-- Community Secondary VLAN B
  |
  +-- Community Secondary VLAN C
  ...
```

The primary VLAN is the "real" VLAN from the perspective of the rest of the network.
It carries the subnet, the SVI, and the upstream trunk tags. Secondary VLANs exist only
within the PVLAN domain and are invisible outside of it.

### 2.2 Secondary VLAN Types

**Isolated VLAN:**
- Only one isolated VLAN can be associated with a given primary VLAN.
- Ports in the isolated VLAN cannot send frames to any other port in the isolated VLAN.
- They can only communicate with promiscuous ports.
- Internally, the switch drops any frame whose source and destination are both in the
  isolated VLAN, regardless of MAC address table entries.

**Community VLAN:**
- Multiple community VLANs can be associated with a single primary VLAN.
- Ports within the same community VLAN can communicate freely with each other.
- Ports in different community VLANs are blocked from each other.
- Community ports can communicate with promiscuous ports.

The switch enforces these rules by manipulating the VLAN tag on egress. A frame arriving
on an isolated port is tagged with the isolated VLAN ID internally. When the switch
determines the destination port is also isolated, it drops the frame. When the destination
is promiscuous, it retags the frame with the primary VLAN ID and forwards it.

### 2.3 Port Types in Detail

**Promiscuous Port:**
- Can communicate with every port in the PVLAN domain (isolated, community, and other
  promiscuous ports).
- Typically connected to the default gateway, firewall, load balancer, DHCP server,
  management station, or any shared service.
- Frames leaving a promiscuous port carry the primary VLAN ID.
- Frames arriving at a promiscuous port are forwarded to secondary VLANs based on the
  port's PVLAN mapping configuration.
- A promiscuous port must explicitly map all secondary VLANs it wants to reach.

**Isolated Host Port:**
- Assigned to the isolated secondary VLAN.
- Can only send frames to promiscuous ports.
- Cannot send frames to other isolated ports or community ports.
- Frames are tagged with the isolated VLAN ID internally.

**Community Host Port:**
- Assigned to one of the community secondary VLANs.
- Can send frames to other ports in the same community VLAN and to promiscuous ports.
- Cannot send frames to isolated ports or ports in a different community VLAN.

### 2.4 Frame Flow Internals

```
INGRESS (frame arrives)                 EGRESS (frame leaves)
+-----------------------+               +-----------------------+
| Source: Isolated port |               | Dest: Promiscuous     |
| Internal tag: Iso VLAN|---ALLOWED---->| Retag: Primary VLAN   |
+-----------------------+               +-----------------------+

+-----------------------+               +-----------------------+
| Source: Isolated port |               | Dest: Isolated port   |
| Internal tag: Iso VLAN|---DROPPED---->| (never reached)       |
+-----------------------+               +-----------------------+

+-----------------------+               +-----------------------+
| Source: Isolated port |               | Dest: Community port  |
| Internal tag: Iso VLAN|---DROPPED---->| (never reached)       |
+-----------------------+               +-----------------------+

+-----------------------+               +-----------------------+
| Source: Community-A   |               | Dest: Community-A     |
| Internal tag: Comm-A  |---ALLOWED---->| Tag: Comm-A VLAN      |
+-----------------------+               +-----------------------+

+-----------------------+               +-----------------------+
| Source: Community-A   |               | Dest: Community-B     |
| Internal tag: Comm-A  |---DROPPED---->| (never reached)       |
+-----------------------+               +-----------------------+

+-----------------------+               +-----------------------+
| Source: Community-A   |               | Dest: Promiscuous     |
| Internal tag: Comm-A  |---ALLOWED---->| Retag: Primary VLAN   |
+-----------------------+               +-----------------------+

+-----------------------+               +-----------------------+
| Source: Promiscuous   |               | Dest: Any mapped port |
| Internal tag: Primary |---ALLOWED---->| Retag: Secondary VLAN |
+-----------------------+               +-----------------------+
```

## 3. Complete Configuration Walkthrough

### 3.1 Topology

```
                        +-------------------+
                        |    Core Router    |
                        |  (L3 Gateway)     |
                        +--------+----------+
                                 |
                            Gi0/1 (promiscuous)
                                 |
                    +------------+------------+
                    |    Distribution Switch  |
                    |    (PVLAN-capable)       |
                    +--+-----+-----+-----+---+
                       |     |     |     |
                    Gi0/2 Gi0/3 Gi0/4 Gi0/5
                      |     |     |     |
                    HostA HostB HostC HostD
                    (iso) (iso) (com1) (com1)
                                 |     |
                              +--+-----+--+
                              | Community  |
                              | (can talk) |
                              +------------+

    Gi0/6 -- HostE (com2)
    Gi0/7 -- HostF (com2)

    Primary VLAN:   100
    Isolated VLAN:  199
    Community 1:    201
    Community 2:    202
```

### 3.2 IOS Configuration (Catalyst 3750/3850/9000)

```
! ============================================================
! Step 1: Define secondary VLANs FIRST (order matters)
! ============================================================
vlan 199
  name PVLAN-ISOLATED
  private-vlan isolated
!
vlan 201
  name PVLAN-COMMUNITY-WEB
  private-vlan community
!
vlan 202
  name PVLAN-COMMUNITY-DB
  private-vlan community
!
! ============================================================
! Step 2: Define primary VLAN and associate secondaries
! ============================================================
vlan 100
  name PVLAN-PRIMARY
  private-vlan primary
  private-vlan association 199,201,202
!
! ============================================================
! Step 3: Configure promiscuous port (gateway uplink)
! ============================================================
interface GigabitEthernet0/1
  description UPLINK-TO-ROUTER-PROMISCUOUS
  switchport mode private-vlan promiscuous
  switchport private-vlan mapping 100 199,201,202
  spanning-tree portfast
  no shutdown
!
! ============================================================
! Step 4: Configure isolated host ports
! ============================================================
interface GigabitEthernet0/2
  description HOST-A-ISOLATED
  switchport mode private-vlan host
  switchport private-vlan host-association 100 199
  spanning-tree portfast
  no shutdown
!
interface GigabitEthernet0/3
  description HOST-B-ISOLATED
  switchport mode private-vlan host
  switchport private-vlan host-association 100 199
  spanning-tree portfast
  no shutdown
!
! ============================================================
! Step 5: Configure community host ports
! ============================================================
interface GigabitEthernet0/4
  description HOST-C-COMMUNITY-WEB
  switchport mode private-vlan host
  switchport private-vlan host-association 100 201
  spanning-tree portfast
  no shutdown
!
interface GigabitEthernet0/5
  description HOST-D-COMMUNITY-WEB
  switchport mode private-vlan host
  switchport private-vlan host-association 100 201
  spanning-tree portfast
  no shutdown
!
interface GigabitEthernet0/6
  description HOST-E-COMMUNITY-DB
  switchport mode private-vlan host
  switchport private-vlan host-association 100 202
  spanning-tree portfast
  no shutdown
!
interface GigabitEthernet0/7
  description HOST-F-COMMUNITY-DB
  switchport mode private-vlan host
  switchport private-vlan host-association 100 202
  spanning-tree portfast
  no shutdown
!
! ============================================================
! Step 6: SVI for inter-VLAN routing and PVLAN proxy
! ============================================================
interface Vlan100
  ip address 10.0.100.1 255.255.255.0
  ip local-proxy-arp
  private-vlan mapping 199,201,202
  no shutdown
```

### 3.3 NX-OS Configuration (Nexus 9000)

```
! ============================================================
! Step 0: Enable the feature
! ============================================================
feature private-vlan
!
! ============================================================
! Step 1: Define secondary VLANs
! ============================================================
vlan 199
  name PVLAN-ISOLATED
  private-vlan isolated
!
vlan 201
  name PVLAN-COMMUNITY-WEB
  private-vlan community
!
vlan 202
  name PVLAN-COMMUNITY-DB
  private-vlan community
!
! ============================================================
! Step 2: Define primary VLAN and associate
! ============================================================
vlan 100
  name PVLAN-PRIMARY
  private-vlan primary
  private-vlan association add 199
  private-vlan association add 201
  private-vlan association add 202
!
! ============================================================
! Step 3: Promiscuous port
! ============================================================
interface Ethernet1/1
  switchport
  switchport mode private-vlan promiscuous
  switchport private-vlan mapping 100 199,201,202
  no shutdown
!
! ============================================================
! Step 4: Isolated host ports
! ============================================================
interface Ethernet1/2
  switchport
  switchport mode private-vlan host
  switchport private-vlan host-association 100 199
  no shutdown
!
interface Ethernet1/3
  switchport
  switchport mode private-vlan host
  switchport private-vlan host-association 100 199
  no shutdown
!
! ============================================================
! Step 5: Community host ports
! ============================================================
interface Ethernet1/4
  switchport
  switchport mode private-vlan host
  switchport private-vlan host-association 100 201
  no shutdown
!
interface Ethernet1/5
  switchport
  switchport mode private-vlan host
  switchport private-vlan host-association 100 202
  no shutdown
!
! ============================================================
! Step 6: SVI with PVLAN mapping
! ============================================================
interface Vlan100
  ip address 10.0.100.1/24
  ip local-proxy-arp
  private-vlan mapping 199,201,202
  no shutdown
```

## 4. PVLAN Across Multiple Switches

### 4.1 Trunk Requirements

When PVLANs span multiple switches, all VLAN IDs (primary and all secondaries) must be
carried on the trunk. The receiving switch must have matching PVLAN configuration.

```
Switch-A                              Switch-B
+------------------+    802.1Q Trunk    +------------------+
|  PVLAN config    |<==================>|  PVLAN config    |
|  VLANs 100,199,  |  allowed: 100,    |  VLANs 100,199,  |
|  201,202         |  199,201,202       |  201,202         |
+------------------+                    +------------------+
```

**Trunk configuration on both switches:**

```
interface GigabitEthernet0/24
  switchport trunk encapsulation dot1q
  switchport mode trunk
  switchport trunk allowed vlan 100,199,201,202
  no shutdown
```

The secondary VLAN tags are preserved across the trunk. Switch-B uses the VLAN tag to
determine whether the incoming frame belongs to the isolated, community-A, or community-B
domain, and enforces the same forwarding rules locally.

### 4.2 PVLAN Edge vs Full PVLAN

PVLAN edge (protected ports) is a simplified mechanism available on most managed switches,
including those that do not support full PVLANs:

```
                 PVLAN Edge (Protected Ports)
+-----------------------------------------------------+
| - Single switch only (no trunk propagation)          |
| - Binary: protected or unprotected                   |
| - No community concept                              |
| - No secondary VLANs needed                         |
| - Configuration: one command per port                |
| - Protected <-/-> Protected (blocked)                |
| - Protected <---> Unprotected (allowed)              |
+-----------------------------------------------------+

                 Full PVLANs
+-----------------------------------------------------+
| - Multi-switch via trunks                            |
| - Three port types: promiscuous, isolated, community |
| - Community groups for intra-group communication     |
| - Requires primary + secondary VLANs                 |
| - Configuration: VLAN definitions + port mappings    |
| - Granular control over traffic flow                 |
+-----------------------------------------------------+
```

**When to use PVLAN edge:** Single-switch deployments with simple "no peer-to-peer"
requirements (e.g., a small office switch where each port should only reach the uplink).

**When to use full PVLANs:** Multi-switch environments, when community grouping is
needed, or when SVI/routing integration is required.

## 5. PVLAN and DHCP

### 5.1 DHCP Server Placement

The DHCP server must be reachable from all hosts. Since isolated hosts can only talk to
promiscuous ports, the DHCP server has two placement options:

**Option A: DHCP server directly on a promiscuous port**

```
+------------+     +-----------+     +----------+
| DHCP Server|---->| Promisc   |     | Isolated |
| 10.0.100.5 |     | Port      |     | Host     |
+------------+     +-----------+     +----------+
                        |                 |
                   DHCP OFFER -------> ACCEPTED
```

**Option B: DHCP server on a remote subnet with ip helper-address**

```
interface Vlan100
  ip address 10.0.100.1 255.255.255.0
  ip helper-address 10.0.200.5
  private-vlan mapping 199,201,202
```

The SVI acts as the DHCP relay agent. DHCP broadcasts from isolated and community hosts
reach the SVI (promiscuous), which forwards them as unicast to the remote DHCP server.

### 5.2 DHCP Snooping with PVLANs

DHCP snooping must be configured on both the primary and all secondary VLANs:

```
ip dhcp snooping
ip dhcp snooping vlan 100,199,201,202
!
interface GigabitEthernet0/1
  ip dhcp snooping trust
```

The promiscuous port (uplink to DHCP server or relay) must be trusted. All host ports
remain untrusted by default.

## 6. ARP Handling in PVLAN Domains

### 6.1 Normal ARP Behavior (Without PVLANs)

In a standard VLAN, when Host A wants to reach Host B, it broadcasts an ARP request.
Host B responds with a unicast ARP reply. Both hosts learn each other's MAC addresses.

### 6.2 ARP in Isolated VLANs

Isolated hosts cannot receive broadcasts from other isolated hosts. The switch drops ARP
requests between isolated ports. This means:

- Host A (isolated) sends ARP request "who has 10.0.100.11?"
- The switch forwards this ARP to the promiscuous port only
- If Host B (10.0.100.11) is also isolated, it never sees the ARP request
- Without intervention, Host A can never resolve Host B's MAC address

This is the desired behavior for L2 isolation. But if L3 communication is needed (via the
router), we need `ip local-proxy-arp`.

### 6.3 Local Proxy ARP

When `ip local-proxy-arp` is enabled on the SVI:

```
Host A: ARP who-has 10.0.100.11?
  |
  v
Switch: Forward to promiscuous port (SVI Vlan100)
  |
  v
SVI: "I know 10.0.100.11 is on my subnet. I'll reply with MY MAC."
  |
  v
Host A: Learns that 10.0.100.11 is at MAC 0000.0c00.0100 (SVI MAC)
  |
  v
Host A: Sends IP packet to Host B, but dst MAC = SVI MAC
  |
  v
SVI: Routes the packet (even though it's same subnet) back to Host B
  |
  v
Host B: Receives the packet via the promiscuous port
```

This "hairpin" routing adds a small latency cost but maintains L2 isolation while
allowing L3 reachability.

### 6.4 Dynamic ARP Inspection (DAI) with PVLANs

DAI validates ARP packets against the DHCP snooping binding table. With PVLANs, configure
DAI on both primary and secondary VLANs:

```
ip arp inspection vlan 100,199,201,202
!
interface GigabitEthernet0/1
  ip arp inspection trust
```

## 7. Security Analysis

### 7.1 What PVLANs Protect Against

```
+---------------------------+-------------------------------------------+
| Threat                    | PVLAN Protection                          |
+---------------------------+-------------------------------------------+
| ARP spoofing between      | Isolated hosts cannot exchange ARP at     |
| isolated hosts            | L2; spoofed replies never arrive          |
+---------------------------+-------------------------------------------+
| MAC flooding between      | Flooded frames from isolated ports only   |
| isolated hosts            | reach promiscuous ports, not other hosts  |
+---------------------------+-------------------------------------------+
| Direct L2 attacks         | No Ethernet-level communication between   |
| (e.g., BPDU injection)   | isolated hosts                            |
+---------------------------+-------------------------------------------+
| Network reconnaissance    | Isolated hosts cannot see each other's    |
| (passive sniffing)        | traffic on the wire                       |
+---------------------------+-------------------------------------------+
| Lateral movement after    | Compromised host cannot pivot to other    |
| host compromise           | hosts at L2                               |
+---------------------------+-------------------------------------------+
```

### 7.2 What PVLANs Do NOT Protect Against

```
+---------------------------+-------------------------------------------+
| Threat                    | Why PVLAN Fails                           |
+---------------------------+-------------------------------------------+
| L3 attacks via router     | If ip local-proxy-arp is enabled, hosts   |
|                           | CAN reach each other via the gateway      |
+---------------------------+-------------------------------------------+
| VLAN hopping via DTP      | PVLANs don't prevent trunk negotiation    |
|                           | attacks; disable DTP separately           |
+---------------------------+-------------------------------------------+
| Attacks on promiscuous    | If the gateway is compromised, all hosts  |
| port device               | are exposed                               |
+---------------------------+-------------------------------------------+
| Broadcast storms          | Broadcasts from promiscuous ports still   |
|                           | reach all mapped secondary ports          |
+---------------------------+-------------------------------------------+
| Management plane attacks  | PVLANs are data plane only; SNMP, SSH,    |
|                           | etc. are unaffected                       |
+---------------------------+-------------------------------------------+
```

### 7.3 Hardening Recommendations

1. Combine PVLANs with port security to limit MAC addresses per port.
2. Enable DHCP snooping and DAI on all PVLAN VLANs.
3. Apply ACLs on the SVI to restrict L3 traffic between isolated hosts when
   `ip local-proxy-arp` is enabled.
4. Disable DTP on all PVLAN ports: `switchport nonegotiate`.
5. Enable BPDU guard on all host ports.
6. Use IP Source Guard to prevent IP spoofing.

```
interface GigabitEthernet0/2
  switchport mode private-vlan host
  switchport private-vlan host-association 100 199
  switchport port-security
  switchport port-security maximum 1
  switchport port-security violation restrict
  switchport nonegotiate
  spanning-tree portfast
  spanning-tree bpduguard enable
  ip verify source
  no shutdown
```

## 8. PVLAN Use Case Deep Dives

### 8.1 ISP Colocation / Shared Ethernet Segment

**Scenario:** An ISP provides Ethernet access to 50 customers on a shared switch. Each
customer has one or two ports. All customers share a /24 subnet for simplicity.

**Design:**
- Primary VLAN 10 (10.0.10.0/24)
- Isolated VLAN 11 (all customer ports)
- Promiscuous port: ISP router (10.0.10.1)

All customers get addresses in 10.0.10.0/24. None can see each other's traffic. The ISP
manages one SVI, one subnet, one DHCP scope. If a customer needs multiple servers that
must communicate, create a community VLAN for that customer.

### 8.2 Hotel Guest Network

**Scenario:** 300-room hotel, each room has an Ethernet jack and a WiFi AP. Guests must
not be able to access other guests' devices.

**Design:**
- Primary VLAN 500 (172.16.0.0/23 for 500+ addresses)
- Isolated VLAN 501
- All room ports: isolated host
- Gateway + DHCP: promiscuous port on core switch
- Captive portal server: promiscuous port

Even if a guest runs Wireshark or ARP-spoofing tools, they cannot see or reach other
guests at L2.

### 8.3 DMZ Server Farm

**Scenario:** Six public-facing servers (web, mail, DNS, VPN, API, FTP) in a DMZ behind
a firewall. If one server is compromised, it should not be able to attack the others
directly.

**Design:**
- Primary VLAN 300 (192.168.100.0/24)
- Isolated VLAN 399 (all DMZ servers)
- Promiscuous port: inside interface of firewall
- SVI with ACLs to control which servers can talk to which

```
+----------+  +----------+  +----------+
| Web .10  |  | Mail .11 |  | DNS .12  |
| (iso)    |  | (iso)    |  | (iso)    |
+----+-----+  +----+-----+  +----+-----+
     |             |             |
     +-------------+-------------+
                   |
          PVLAN Switch (all isolated)
                   |
              Promiscuous port
                   |
            +------+------+
            |  Firewall   |
            | 192.168.100.1|
            +-------------+
```

If the web server is compromised, the attacker cannot ARP-scan or directly connect to
the mail or DNS server. All traffic must go through the firewall, where IPS and ACLs
can detect and block lateral movement.

### 8.4 Multi-Tenant Cloud Infrastructure

**Scenario:** A private cloud with 20 tenants. Each tenant has 2-10 VMs. VMs within a
tenant need L2 adjacency (e.g., for clustering). VMs across tenants must be isolated.

**Design:**
- Primary VLAN 800 (10.10.0.0/20 for large address space)
- Community VLAN 810: Tenant Alpha (VMs can talk)
- Community VLAN 811: Tenant Beta
- Community VLAN 812: Tenant Gamma
- ...up to Community VLAN 829
- Isolated VLAN 899: Shared single-VM tenants
- Promiscuous port: shared gateway/load balancer

## 9. Troubleshooting Guide

### 9.1 Diagnostic Command Reference

**Show PVLAN VLAN information:**
```
Switch# show vlan private-vlan

Primary  Secondary  Type              Ports
-------  ---------  ----------------  ----------------------------------------
100      199        isolated          Gi0/2, Gi0/3
100      201        community         Gi0/4, Gi0/5
100      202        community         Gi0/6, Gi0/7
```

**Show switchport details:**
```
Switch# show interfaces GigabitEthernet0/2 switchport
Name: Gi0/2
Switchport: Enabled
Administrative Mode: private-vlan host
Operational Mode: private-vlan host
Administrative Trunking Encapsulation: negotiate
Negotiation of Trunking: Off
Access Mode VLAN: 1 (default)
Trunking Native Mode VLAN: 1 (default)
Administrative Native VLAN tagging: enabled
Voice VLAN: none
Administrative private-vlan host-association: 100 (PVLAN-PRIMARY) 199 (PVLAN-ISOLATED)
Administrative private-vlan mapping: none
Administrative private-vlan trunk native VLAN: none
Administrative private-vlan trunk Native VLAN tagging: enabled
Administrative private-vlan trunk encapsulation: dot1q
Administrative private-vlan trunk normal VLANs: none
Administrative private-vlan trunk associations: none
Administrative private-vlan trunk mappings: none
Operational private-vlan: 100 (PVLAN-PRIMARY) 199 (PVLAN-ISOLATED)
```

**Show SVI PVLAN mapping:**
```
Switch# show interfaces Vlan100 private-vlan mapping
Interface       Secondary VLAN    Type
-----------     ----------------  -----------------
Vlan100         199               isolated
Vlan100         201               community
Vlan100         202               community
```

### 9.2 Systematic Troubleshooting Procedure

```
START
  |
  v
[1] Do VLANs exist?
    show vlan brief
    --> If missing, create them
  |
  v
[2] Are VLAN types correct?
    show vlan private-vlan
    --> Verify: primary, isolated, community
  |
  v
[3] Is association configured?
    show vlan private-vlan
    --> Primary must list all secondaries
  |
  v
[4] Are ports in correct mode?
    show interfaces Gi0/X switchport
    --> Check "Administrative Mode: private-vlan host/promiscuous"
    --> Check "Operational Mode" matches
  |
  v
[5] Is host-association correct?
    show interfaces Gi0/X switchport
    --> Verify primary + secondary pair
  |
  v
[6] Is promiscuous mapping complete?
    show interfaces Gi0/1 switchport
    --> "private-vlan mapping" must list all secondaries
  |
  v
[7] Is SVI mapping configured? (if routing)
    show interfaces Vlan100 private-vlan mapping
    --> Must list all secondaries
  |
  v
[8] Is ip local-proxy-arp enabled? (if isolated hosts need L3)
    show running-config interface Vlan100
    --> Look for "ip local-proxy-arp"
  |
  v
[9] Are trunks carrying all VLANs? (multi-switch)
    show interfaces trunk
    --> All primary + secondary VLANs in allowed list
  |
  v
[10] STP issues?
     show spanning-tree vlan 100,199,201,202
     --> All VLANs should be forwarding on expected ports
  |
  v
RESOLVED
```

### 9.3 Common Failure Modes

**Symptom: Isolated hosts can communicate with each other.**

Root causes:
1. Ports are not actually in `private-vlan host` mode (they may be in access mode on
   the primary VLAN, which has no isolation).
2. The host-association points to a community VLAN instead of the isolated VLAN.
3. VTP is overwriting local PVLAN config (VTP must be transparent or off for PVLANs
   on older IOS).

**Symptom: No host can reach the gateway.**

Root causes:
1. Promiscuous port mapping is missing or incomplete.
2. SVI does not have `private-vlan mapping` for the secondary VLANs.
3. The promiscuous port is down or in err-disabled state.
4. STP is blocking the promiscuous port.

**Symptom: Community hosts cannot talk to each other.**

Root causes:
1. They are accidentally in different community VLANs.
2. They are in the isolated VLAN, not a community VLAN.
3. STP topology change has put one port in blocking state.

**Symptom: PVLAN works on one switch but not across the trunk.**

Root causes:
1. Secondary VLAN IDs not in the trunk's allowed list.
2. Remote switch does not have matching PVLAN configuration.
3. VTP pruning is removing secondary VLANs from the trunk.
4. Native VLAN mismatch causing tag stripping.

## 10. Platform-Specific Notes

### 10.1 VTP Interaction

On IOS switches using VTP:
- VTP version 1 and 2: PVLANs require **VTP transparent mode**. VTP server mode will
  not propagate PVLAN associations.
- VTP version 3: Supports PVLAN propagation in server and client modes.

```
! Required for VTPv1/v2
vtp mode transparent
```

### 10.2 STP Considerations

Each secondary VLAN runs its own STP instance (in PVST+ mode) or shares instances
(in MST mode). Ensure STP root and port states are consistent across all PVLAN VLANs.

In MST, map all PVLAN VLANs (primary and secondaries) to the same MST instance:

```
spanning-tree mode mst
spanning-tree mst configuration
  instance 1 vlan 100,199,201,202
```

### 10.3 Platform Support Matrix

```
+---------------------------+--------+--------+--------+
| Feature                   | Cat3750| Cat9000| Nexus9K|
+---------------------------+--------+--------+--------+
| Isolated VLAN             |  Yes   |  Yes   |  Yes   |
| Community VLAN            |  Yes   |  Yes   |  Yes   |
| Promiscuous trunk         |  Yes   |  Yes   |  Yes   |
| PVLAN over vPC            |  N/A   |  N/A   |  Yes   |
| SVI PVLAN mapping         |  Yes   |  Yes   |  Yes   |
| PVLAN with DHCP snooping  |  Yes   |  Yes   |  Yes   |
| PVLAN with DAI            |  Yes   |  Yes   |  Yes   |
| VTP v3 PVLAN propagation  |  Yes   |  Yes   |  N/A   |
+---------------------------+--------+--------+--------+
```

### 10.4 NX-OS vPC with PVLANs

On Nexus switches using vPC (virtual Port Channel), PVLANs require consistent
configuration on both vPC peers:

```
! Both vPC peer switches must have identical:
! - VLAN definitions and types
! - PVLAN associations
! - SVI configuration
! - PVLAN port mappings on vPC member ports

! Verify consistency:
show vpc consistency-parameters pvlan
```

## Prerequisites

- Switch must support Private VLANs (enterprise-class L2/L3 switch)
- VTP transparent mode (VTPv1/v2) or VTP version 3
- Understanding of standard VLANs, trunking (802.1Q), and STP
- Familiarity with SVI (Switch Virtual Interface) configuration for L3 routing
- For NX-OS: `feature private-vlan` must be enabled
- For multi-switch: trunk links must carry all PVLAN VLAN IDs
- Understanding of ARP, DHCP relay, and proxy ARP concepts

## References

- [RFC 5765 - Security Issues with Private VLANs](https://www.rfc-editor.org/rfc/rfc5765)
- [Cisco IOS Private VLANs Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst3750x_3560x/software/release/15-2_4_e/configguide/b_1524e_3750x_3560x_cg/b_1524e_3750x_3560x_cg_chapter_01011.html)
- [Cisco NX-OS Private VLANs Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/vlan/configuration/guide/b-cisco-nexus-9000-nx-os-vlan-configuration-guide-93x/b-cisco-nexus-9000-nx-os-vlan-configuration-guide-93x_chapter_01000.html)
- [IEEE 802.1Q-2018 - Bridges and Bridged Networks](https://standards.ieee.org/standard/802_1Q-2018.html)
- [Cisco Private VLAN Catalyst Switching Design Guide](https://www.cisco.com/c/en/us/support/docs/lan-switching/private-vlans-pvlans/40781-194.html)
