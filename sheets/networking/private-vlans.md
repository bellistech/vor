# Private VLANs (Layer 2 Isolation Within a VLAN)

Subdivide a single primary VLAN into isolated and community segments so hosts share a gateway but cannot freely communicate at Layer 2.

## PVLAN Architecture

### VLAN Types

| Type | Role | Behavior |
|-----------|-------------------------------|-----------------------------------------------|
| Primary | Parent VLAN | Carries traffic to/from promiscuous ports |
| Isolated | Secondary (one per primary) | Hosts cannot talk to each other at L2 |
| Community | Secondary (multiple allowed) | Hosts in same community talk; blocked to others|

### Port Types

| Port Type | Connects To | Sends To | Receives From |
|-------------|--------------------------|-------------------------------|-------------------------------|
| Promiscuous | Router, firewall, gateway| All secondary VLAN ports | All secondary VLAN ports |
| Isolated | End hosts needing full isolation | Promiscuous only | Promiscuous only |
| Community | End hosts sharing a group | Same community + promiscuous | Same community + promiscuous |

### Traffic Flow Diagram

```
                   +------------------+
                   |   Gateway / SVI  |
                   | (Promiscuous Port)|
                   +--------+---------+
                            |
              Primary VLAN 100 (trunk)
                            |
         +------------------+------------------+
         |                  |                  |
    +----+----+        +----+----+        +----+----+
    | Isolated |        |Community|        |Community|
    | VLAN 101 |        |VLAN 102 |        |VLAN 103 |
    +----+----+        +----+----+        +----+----+
         |                  |                  |
    +----+----+        +----+----+        +----+----+
    | Host A  |        | Host C  |        | Host E  |
    | Host B  |        | Host D  |        | Host F  |
    +---------+        +---------+        +---------+

    A <-/-> B           C <---> D          E <---> F
    A <---> GW          C <---> GW         E <---> GW
    A <-/-> C           C <-/-> E          (blocked across
    A <-/-> E           D <-/-> F           communities)
```

### Allowed Traffic Matrix

```
             Promisc   Isolated   Community-A   Community-B
Promisc        --        YES         YES           YES
Isolated      YES         NO          NO            NO
Community-A   YES         NO         YES            NO
Community-B   YES         NO          NO           YES
```

## PVLAN Associations

### Primary-to-Secondary Mapping

A primary VLAN can associate with:
- Exactly one isolated secondary VLAN
- One or more community secondary VLANs

```
Primary VLAN 100
  |-- Isolated VLAN 101   (only one allowed)
  |-- Community VLAN 102
  |-- Community VLAN 103
  +-- Community VLAN 104
```

### Association Command (IOS)

```
! Step 1: Create the VLANs
vlan 101
  private-vlan isolated
vlan 102
  private-vlan community
vlan 103
  private-vlan community
vlan 100
  private-vlan primary
  private-vlan association 101,102,103
```

### Association Command (NX-OS)

```
vlan 101
  private-vlan isolated
vlan 102
  private-vlan community
vlan 103
  private-vlan community
vlan 100
  private-vlan primary
  private-vlan association add 101,102,103
```

## Port Configuration

### Promiscuous Port (IOS)

```
interface GigabitEthernet0/1
  switchport mode private-vlan promiscuous
  switchport private-vlan mapping 100 101,102,103
  no shutdown
```

### Isolated Port (IOS)

```
interface GigabitEthernet0/2
  switchport mode private-vlan host
  switchport private-vlan host-association 100 101
  no shutdown
```

### Community Port (IOS)

```
interface GigabitEthernet0/3
  switchport mode private-vlan host
  switchport private-vlan host-association 100 102
  no shutdown
```

### Promiscuous Port (NX-OS)

```
interface Ethernet1/1
  switchport
  switchport mode private-vlan promiscuous
  switchport private-vlan mapping 100 101,102,103
  no shutdown
```

### Host Port (NX-OS)

```
interface Ethernet1/2
  switchport
  switchport mode private-vlan host
  switchport private-vlan host-association 100 101
  no shutdown
```

## PVLAN with SVIs

### SVI Mapping (Inter-VLAN Routing)

Only the primary VLAN SVI is created. Secondary VLANs map into it:

```
interface Vlan100
  ip address 10.0.100.1 255.255.255.0
  private-vlan mapping 101,102,103
  no shutdown
```

All hosts (isolated + community) use 10.0.100.1 as their default gateway. The SVI acts as the promiscuous interface for routed traffic.

### IP Addressing

```
Primary VLAN 100:  10.0.100.0/24
  Gateway (SVI):   10.0.100.1
  Host A (iso):    10.0.100.10
  Host B (iso):    10.0.100.11
  Host C (comm):   10.0.100.20
  Host D (comm):   10.0.100.21
```

All hosts share the same subnet even though they are in different secondary VLANs.

## PVLAN Across Switches

### Trunk Configuration

PVLANs traverse trunks using standard 802.1Q tags. Both primary and secondary VLAN IDs must be allowed:

```
interface GigabitEthernet0/24
  switchport trunk encapsulation dot1q
  switchport mode trunk
  switchport trunk allowed vlan 100-103
  no shutdown
```

### PVLAN Edge (Protected Ports)

A simpler alternative when full PVLAN is overkill. Protected ports on the same switch cannot communicate with each other — traffic must go through a router.

```
interface GigabitEthernet0/5
  switchport protected
```

```
+----------+     +----------+
| Host A   |     | Host B   |
| (protected)    | (protected)
+-----+----+     +-----+----+
      |                |
  +---+----------------+---+
  |       Switch            |
  |   A <-/-> B (blocked)   |
  |   A  ---> Router (ok)   |
  +-------------------------+
```

Limitation: protected ports only apply within a single switch. Across trunks, the isolation is not enforced.

## PVLAN with DHCP

### DHCP Relay

Because isolated hosts cannot reach each other, a DHCP server connected to an isolated port cannot serve other isolated hosts. Place the DHCP server on the promiscuous port or use `ip helper-address` on the SVI:

```
interface Vlan100
  ip address 10.0.100.1 255.255.255.0
  ip helper-address 10.0.200.5
  private-vlan mapping 101,102,103
```

### ARP Behavior

The switch proxies ARP between secondary VLANs and the promiscuous port. Isolated hosts learn the gateway MAC via ARP through the promiscuous port. They never see each other's ARP replies.

```
Host A (isolated) --> ARP who-has GW? --> Promiscuous port --> GW replies
Host A (isolated) --> ARP who-has Host B? --> BLOCKED (both isolated)
```

## PVLAN Proxy Routing

When isolated hosts need to reach each other at Layer 3, the promiscuous port (SVI) routes between them. This is sometimes called "PVLAN proxy" or "hairpin routing":

```
Host A (10.0.100.10, isolated)
  --> packet to Host B (10.0.100.11, isolated)
  --> L2 blocked (both isolated)
  --> send to default GW 10.0.100.1
  --> SVI routes it back down to Host B
  --> Host B receives via promiscuous port

+--------+     +------+     +--------+
| Host A | --> | SVI  | --> | Host B |
| (.10)  |     |(.1)  |     | (.11)  |
+--------+     +------+     +--------+
   iso port    promisc port    iso port
```

Enable `ip local-proxy-arp` on the SVI so the router responds to ARP for hosts on the same subnet:

```
interface Vlan100
  ip address 10.0.100.1 255.255.255.0
  ip local-proxy-arp
  private-vlan mapping 101,102,103
```

## Troubleshooting

### Verification Commands

```
! Show PVLAN configuration
show vlan private-vlan

! Show port PVLAN mode and associations
show interfaces switchport

! Show PVLAN type for a VLAN
show vlan private-vlan type

! Verify SVI mapping
show interfaces vlan 100 private-vlan mapping

! NX-OS specific
show vlan private-vlan
show interface ethernet 1/1 switchport
show vlan id 100 private-vlan
```

### Common Issues

| Symptom | Likely Cause | Fix |
|-------------------------------|-----------------------------------|---------------------------------------|
| Hosts in isolated can talk | Port not in host mode | Set `switchport mode private-vlan host` |
| All hosts blocked | Missing PVLAN association | Check `private-vlan association` on primary |
| Promiscuous port no traffic | Missing mapping | Add `private-vlan mapping` on port |
| SVI not routing PVLAN traffic| No `private-vlan mapping` on SVI | Map secondary VLANs to SVI |
| DHCP not working | Server on isolated port | Move server to promiscuous or use relay |
| Isolated hosts can't reach GW| Wrong host-association | Verify primary+secondary match |
| Trunk not carrying PVLANs | Secondary VLANs not in allowed list| Add all PVLAN IDs to trunk allowed VLANs |

### Debug Checklist

1. Confirm all VLANs exist: `show vlan brief`
2. Verify VLAN types: `show vlan private-vlan`
3. Check port mode: `show interfaces Gi0/1 switchport`
4. Confirm association: primary must list all secondaries
5. Confirm mapping: promiscuous port must map all secondaries
6. If using SVI: confirm `private-vlan mapping` on interface Vlan
7. If routing between isolated: enable `ip local-proxy-arp`

## Use Cases

### ISP Shared Segments
Customers on a shared L2 segment (e.g., colocation) get isolated ports. The ISP router is the promiscuous port. Customers share a subnet but cannot sniff each other's traffic.

### Hotel / Guest WiFi
Each room's AP port is isolated. All rooms share the same gateway and DHCP but cannot see each other. Prevents ARP spoofing and lateral movement.

### DMZ Server Isolation
Public-facing servers in a DMZ each get an isolated port. The firewall is promiscuous. Even if one server is compromised, it cannot directly attack the others at L2.

### Multi-tenant Data Centers
Tenant clusters use community VLANs (servers in a tenant can talk). Different tenants are isolated. The shared gateway provides upstream routing.

## Tips

- Only one isolated VLAN per primary VLAN is allowed, but many hosts can connect to that isolated VLAN
- Community VLANs allow intra-group communication; use them for server clusters that need L2 adjacency
- Always place DHCP servers and gateways on promiscuous ports
- Protected ports (PVLAN edge) do not persist across trunks; use full PVLANs for multi-switch deployments
- PVLAN configuration is cleared if the VLAN is deleted; be careful with `no vlan` commands
- PVLANs do not work in VTP transparent mode on some older IOS versions; check platform support
- On NX-OS, use `feature private-vlan` before any PVLAN config
- PVLAN does not protect against L3 attacks; hosts can still reach each other via routing unless ACLs are applied
- When using PVLAN proxy routing, always enable `ip local-proxy-arp`

## See Also

- vlans
- vlan-trunking
- 802.1q
- port-security
- dhcp-snooping
- arp-inspection
- acl

## References

- [RFC 5765 - Security Issues with Private VLANs](https://www.rfc-editor.org/rfc/rfc5765)
- [Cisco Private VLANs Configuration Guide (IOS)](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst3750x_3560x/software/release/15-2_4_e/configguide/b_1524e_3750x_3560x_cg/b_1524e_3750x_3560x_cg_chapter_01011.html)
- [Cisco NX-OS Private VLANs Configuration](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/vlan/configuration/guide/b-cisco-nexus-9000-nx-os-vlan-configuration-guide-93x/b-cisco-nexus-9000-nx-os-vlan-configuration-guide-93x_chapter_01000.html)
- [IEEE 802.1Q - Virtual Bridged Local Area Networks](https://standards.ieee.org/standard/802_1Q-2018.html)
