# VRF (Virtual Routing and Forwarding)

Network virtualization at Layer 3 — isolated routing and forwarding tables on a single router, enabling multi-tenant environments, management plane separation, and MPLS L3VPN services.

## Concepts

### VRF Components

```
+------------------+      +------------------+
|   VRF "RED"      |      |   VRF "BLUE"     |
|  +-----------+   |      |  +-----------+   |
|  | RIB (RED) |   |      |  | RIB (BLUE)|   |
|  +-----------+   |      |  +-----------+   |
|  +-----------+   |      |  +-----------+   |
|  | FIB (RED) |   |      |  | FIB (BLUE)|   |
|  +-----------+   |      |  +-----------+   |
|  Interfaces:     |      |  Interfaces:     |
|   Gi0/1, Gi0/2   |      |   Gi0/3, Gi0/4   |
+------------------+      +------------------+

Global Routing Table (default VRF)
  +-----------+
  | RIB       |    Interfaces: Gi0/0, Lo0
  +-----------+
  +-----------+
  | FIB       |
  +-----------+

Each VRF has its own:
  - Routing table (RIB)
  - Forwarding table (FIB)
  - Interface membership
  - Routing protocol instances (optional)
  - ARP/ND cache (per-VRF)
```

### VRF-Lite vs VRF with MPLS

| Feature | VRF-Lite | VRF + MPLS L3VPN |
|:---|:---|:---|
| Label switching | No | Yes (VPN + transport labels) |
| MP-BGP required | No | Yes (VPNv4/VPNv6) |
| RD/RT required | No (IOS-XE) / Yes (IOS) | Yes |
| Scope | Single router or site | Multi-site across provider |
| Use case | Local segmentation, mgmt VRF | Enterprise WAN, service provider |
| Scalability | Per-device config | Centralized via BGP |

## VRF-Lite Configuration

### Cisco IOS (Legacy ip vrf)

```
! Define VRF (IPv4 only)
ip vrf MGMT
 rd 65000:999
 ! RD required even for VRF-Lite on classic IOS

! Assign interface to VRF
! WARNING: adding VRF to an interface REMOVES the existing IP address
interface GigabitEthernet0/1
 ip vrf forwarding MGMT
 ip address 10.99.1.1 255.255.255.0

! VRF-aware static route
ip route vrf MGMT 0.0.0.0 0.0.0.0 10.99.1.254

! VRF-aware ping and telnet
ping vrf MGMT 10.99.1.254
telnet 10.99.1.254 /vrf MGMT
```

### Cisco IOS-XE (VRF Definition — Dual-Stack)

```
! VRF definition supports both IPv4 and IPv6
vrf definition MGMT
 rd 65000:999
 !
 address-family ipv4
  route-target export 65000:999
  route-target import 65000:999
 exit-address-family
 !
 address-family ipv6
  route-target export 65000:999
  route-target import 65000:999
 exit-address-family

! Assign interface
interface GigabitEthernet1
 vrf forwarding MGMT
 ip address 10.99.1.1 255.255.255.0
 ipv6 address 2001:db8:99::1/64

! VRF-aware static routes
ip route vrf MGMT 0.0.0.0 0.0.0.0 10.99.1.254
ipv6 route vrf MGMT ::/0 2001:db8:99::fe

! VRF-Lite without RD (IOS-XE 16.x+ simplified)
vrf definition DATA
 !
 address-family ipv4
 exit-address-family
```

### Cisco NX-OS

```
! Create VRF
vrf context MGMT
 rd 65000:999
 address-family ipv4 unicast
  route-target import 65000:999
  route-target export 65000:999
 address-family ipv6 unicast
  route-target import 65000:999
  route-target export 65000:999

! Assign interface
interface Ethernet1/1
 vrf member MGMT
 ip address 10.99.1.1/24
 ipv6 address 2001:db8:99::1/64
 no shutdown

! VRF-aware routing
ip route 0.0.0.0/0 10.99.1.254 vrf MGMT

! NX-OS management VRF (built-in)
! mgmt0 interface is in VRF "management" by default
show vrf
show ip route vrf MGMT
```

### Juniper JunOS

```
# Create routing instance (VRF type)
set routing-instances MGMT instance-type vrf
set routing-instances MGMT interface ge-0/0/1.0
set routing-instances MGMT route-distinguisher 65000:999
set routing-instances MGMT vrf-target target:65000:999

# Static route within VRF
set routing-instances MGMT routing-options static route 0.0.0.0/0 next-hop 10.99.1.254

# VRF-Lite (virtual-router type — no RD/RT needed)
set routing-instances MGMT instance-type virtual-router
set routing-instances MGMT interface ge-0/0/1.0

# Verify
show route instance MGMT
show route table MGMT.inet.0
ping routing-instance MGMT 10.99.1.254
```

### Linux VRF (iproute2, kernel 4.3+)

```bash
# Create VRF device
sudo ip link add VRF-MGMT type vrf table 100
sudo ip link set VRF-MGMT up

# Assign interface to VRF
sudo ip link set eth1 master VRF-MGMT

# Add routes to VRF table
sudo ip route add default via 10.99.1.254 table 100
sudo ip route add 10.99.0.0/16 dev eth1 table 100

# VRF-aware operations
ip route show vrf VRF-MGMT
ip route show table 100
ping -I VRF-MGMT 10.99.1.254

# Bind a socket to VRF (SO_BINDTODEVICE or net.ipv4.tcp_l3mdev_accept)
sudo sysctl -w net.ipv4.tcp_l3mdev_accept=1     # accept connections on any VRF
sudo sysctl -w net.ipv4.udp_l3mdev_accept=1

# Run a command in VRF context
sudo ip vrf exec VRF-MGMT ping 10.99.1.254
sudo ip vrf exec VRF-MGMT ssh admin@10.99.1.254
sudo ip vrf exec VRF-MGMT curl http://10.99.1.1:8080

# Show all VRFs
ip vrf show
ip link show type vrf

# Persistent (systemd-networkd)
# /etc/systemd/network/10-vrf-mgmt.netdev
# [NetDev]
# Name=VRF-MGMT
# Kind=vrf
#
# [VRF]
# Table=100

# /etc/systemd/network/20-eth1.network
# [Match]
# Name=eth1
#
# [Network]
# VRF=VRF-MGMT
```

## VRF-Aware Services

### DHCP

```
! Cisco IOS — DHCP server in VRF
ip dhcp pool POOL-VRF-RED
 vrf RED
 network 192.168.10.0 255.255.255.0
 default-router 192.168.10.1
 dns-server 10.0.0.53

! DHCP relay across VRFs
interface GigabitEthernet0/1
 ip vrf forwarding RED
 ip address 192.168.10.1 255.255.255.0
 ip helper-address vrf SHARED 10.0.0.67
 ! Relay to DHCP server in VRF "SHARED"
```

### DNS, NTP, Syslog

```
! VRF-aware DNS
ip name-server vrf MGMT 10.99.1.53
ip domain lookup source-interface GigabitEthernet1 vrf MGMT

! VRF-aware NTP
ntp server vrf MGMT 10.99.1.123
ntp source GigabitEthernet1

! VRF-aware syslog
logging host 10.99.1.514 vrf MGMT
logging source-interface GigabitEthernet1 vrf MGMT

! VRF-aware SNMP
snmp-server host 10.99.1.162 vrf MGMT community PUBLIC
snmp-server trap-source GigabitEthernet1
```

### SSH, TACACS+, RADIUS

```
! VRF-aware SSH server
ip ssh source-interface GigabitEthernet1 vrf MGMT

! VRF-aware TACACS+ (IOS-XE)
aaa group server tacacs+ TAC-SERVERS
 server-private 10.99.1.49 key 0 TACKEY
 ip vrf forwarding MGMT

! VRF-aware RADIUS
aaa group server radius RAD-SERVERS
 server-private 10.99.1.812 key 0 RADKEY
 ip vrf forwarding MGMT

! VRF-aware HTTP server
ip http server
ip http access-class 99
ip http client source-interface GigabitEthernet1 vrf MGMT
```

## VRF Route Leaking

### Static Route Leaking

```
! Leak route from VRF RED into global table
ip route 192.168.10.0 255.255.255.0 GigabitEthernet0/1 10.0.0.1 global
! "global" keyword means next-hop is in the global table

! Leak global route into VRF RED
ip route vrf RED 0.0.0.0 0.0.0.0 10.0.0.1 global

! Leak between VRFs (RED → BLUE)
ip route vrf BLUE 192.168.10.0 255.255.255.0 GigabitEthernet0/1 vrf RED 192.168.10.2

! IOS-XE: inter-VRF next-hop
ip route vrf BLUE 192.168.10.0 255.255.255.0 vrf RED 192.168.10.2
```

### BGP Route Leaking (Import/Export RT)

```
! VRF RED exports routes with RT 65000:100
vrf definition RED
 rd 65000:100
 address-family ipv4
  route-target export 65000:100
  route-target import 65000:100
  route-target import 65000:999    ! import shared services

! VRF SHARED-SERVICES exports routes to all VRFs
vrf definition SHARED-SERVICES
 rd 65000:999
 address-family ipv4
  route-target export 65000:999    ! exported to all who import 999
  route-target import 65000:100    ! import RED routes
  route-target import 65000:200    ! import BLUE routes

! BGP configuration for route leaking
router bgp 65000
 address-family ipv4 vrf RED
  redistribute connected
  redistribute static
 address-family ipv4 vrf SHARED-SERVICES
  redistribute connected
```

### Route-Map Controlled Leaking (Selective)

```
! Only leak specific prefixes between VRFs
ip prefix-list SHARED-PREFIXES permit 10.0.0.0/24
ip prefix-list SHARED-PREFIXES permit 10.0.1.0/24

route-map LEAK-TO-RED permit 10
 match ip address prefix-list SHARED-PREFIXES

vrf definition SHARED-SERVICES
 address-family ipv4
  export map LEAK-TO-RED
```

## MP-BGP VPNv4/VPNv6

### PE Router Configuration

```
! Full L3VPN PE configuration
router bgp 65000
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor 10.0.0.2 remote-as 65000
 neighbor 10.0.0.2 update-source Loopback0
 !
 address-family vpnv4 unicast
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 send-community both
 exit-address-family
 !
 address-family vpnv6 unicast
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 send-community both
 exit-address-family
 !
 address-family ipv4 vrf CUSTOMER-A
  redistribute connected
  neighbor 192.168.1.2 remote-as 65100
  neighbor 192.168.1.2 activate
 exit-address-family

! CE-PE routing (can be static, OSPF, EIGRP, or eBGP)
! OSPF as PE-CE protocol
router ospf 100 vrf CUSTOMER-A
 redistribute bgp 65000 subnets
 network 192.168.1.0 0.0.0.255 area 0
```

### Route Distinguisher vs Route Target

```
VPNv4 route construction:

  RD:PREFIX = 65000:100:192.168.1.0/24
  ├─ RD (65000:100) makes the prefix globally unique in BGP
  ├─ PREFIX (192.168.1.0/24) is the actual customer route
  └─ RT (65000:100) attached as extended community → controls import

Two customers with 10.0.0.0/8:
  Customer A: 65000:100:10.0.0.0/8  with RT 65000:100
  Customer B: 65000:200:10.0.0.0/8  with RT 65000:200
  → Different VPNv4 routes, imported into separate VRFs by RT matching
```

## VRF with MPLS L3VPN

### End-to-End Packet Walk

```
CE-A (192.168.1.2)                                    CE-B (192.168.2.2)
   │                                                      │
   │ IP: 192.168.1.2 → 192.168.2.2                       │
   ▼                                                      ▲
PE-1 (ingress)                                        PE-2 (egress)
   │ VRF lookup → VPN label 500                           │
   │ BGP NH → PE-2 (10.0.0.2) → transport label 300      │
   │ Push: [300][500][IP]                                  │
   ▼                                                      │
P-1 (core)                                                │
   │ Swap transport: 300 → 301                            │
   ▼                                                      │
P-2 (penultimate)                                         │
   │ PHP: pop transport label 301                         │
   │ Forward: [500][IP] to PE-2                           │
   ▼                                                      │
PE-2 receives [500][IP]                                   │
   │ Pop VPN label 500 → identifies VRF CUSTOMER-A        │
   │ FIB lookup in VRF → forward to CE-B                  │
   └──────────────────────────────────────────────────────┘
```

### Verification Commands

```
! Show VRF configuration and interfaces
show vrf
show vrf detail RED
show ip vrf interfaces

! Show VRF routing table
show ip route vrf RED
show ipv6 route vrf RED

! Show VRF BGP table
show bgp vpnv4 unicast all
show bgp vpnv4 unicast vrf RED
show bgp vpnv4 unicast rd 65000:100

! Show VPN labels
show bgp vpnv4 unicast vrf RED labels
show mpls forwarding-table vrf RED

! Show VRF CEF/FIB
show ip cef vrf RED
show ip cef vrf RED 192.168.2.0/24 detail

! NX-OS
show vrf
show ip route vrf RED
show bgp l2vpn evpn

! JunOS
show route instance
show route table RED.inet.0
show route table bgp.l3vpn.0
show mpls lsp

! Linux
ip vrf show
ip route show vrf VRF-MGMT
ip route show table 100
```

## Management Plane Separation

### Dedicated Management VRF

```
! Best practice: separate management traffic from data plane
vrf definition Mgmt-vrf
 !
 address-family ipv4
 exit-address-family

! Management interface in VRF
interface GigabitEthernet0/0
 vrf forwarding Mgmt-vrf
 ip address 10.99.1.1 255.255.255.0

! All management services use Mgmt-vrf
ip route vrf Mgmt-vrf 0.0.0.0 0.0.0.0 10.99.1.254
ip ssh source-interface GigabitEthernet0/0 vrf Mgmt-vrf
logging host 10.99.1.514 vrf Mgmt-vrf
ntp server vrf Mgmt-vrf 10.99.1.123
snmp-server host 10.99.1.162 vrf Mgmt-vrf community PUBLIC

! ACL restricting management access to Mgmt-vrf only
ip access-list extended MGMT-ACCESS
 permit tcp 10.99.0.0 0.0.255.255 any eq 22
 deny ip any any log
!
line vty 0 15
 access-class MGMT-ACCESS in vrf-also
 transport input ssh
```

### NX-OS Default Management VRF

```
! NX-OS has a built-in "management" VRF
! mgmt0 interface is automatically in this VRF
show vrf management

! All management commands default to management VRF
! To reach data-plane addresses from management:
copy running-config tftp://10.1.1.1/config vrf management
ping 10.99.1.1 vrf management
ssh admin@10.99.1.1 vrf management
```

## Inter-VRF Routing

### Shared Services Design

```
Topology: Multiple tenant VRFs need access to shared services (DNS, NTP, AAA)

    VRF-RED ──────┐
                  │     ┌──────────────────────┐
    VRF-BLUE ─────┼────→│  VRF-SHARED-SERVICES │──→ DNS, NTP, AAA
                  │     └──────────────────────┘
    VRF-GREEN ────┘

Implementation via RT import/export:
  VRF-RED:    import 65000:100, import 65000:999
  VRF-BLUE:   import 65000:200, import 65000:999
  VRF-GREEN:  import 65000:300, import 65000:999
  VRF-SHARED: export 65000:999, import 65000:100, 200, 300

Tenants can reach shared services but NOT each other
(RED does not import 200 or 300)
```

### Physical Loopback (Cable) Inter-VRF

```
! Legacy approach: connect two interfaces with a cable
! or use sub-interfaces on same physical port
interface GigabitEthernet0/1.10
 encapsulation dot1Q 10
 vrf forwarding RED
 ip address 172.16.0.1 255.255.255.252

interface GigabitEthernet0/1.20
 encapsulation dot1Q 20
 vrf forwarding BLUE
 ip address 172.16.0.2 255.255.255.252

! Route between VRFs via sub-interfaces
ip route vrf RED 192.168.20.0 255.255.255.0 172.16.0.2
ip route vrf BLUE 192.168.10.0 255.255.255.0 172.16.0.1
```

## Tips

- Adding a VRF to an interface removes its existing IP address. Always re-apply the address after the `vrf forwarding` command.
- Use a dedicated management VRF to isolate SSH, SNMP, syslog, and NTP traffic from the data plane. This is a security best practice and simplifies ACL management.
- Route Distinguisher must be unique per VRF per PE router. A common convention is `<router-id>:<vrf-id>` or `<ASN>:<customer-id>`.
- Route Target controls policy (who sees whose routes). RD controls uniqueness (preventing prefix collisions in BGP). Do not confuse them.
- When troubleshooting VRF connectivity, always specify the VRF in diagnostic commands: `ping vrf RED`, `show ip route vrf RED`, `traceroute vrf RED`.
- On Linux, `ip vrf exec <vrf> <command>` runs any command within a VRF context — useful for curl, ping, ssh, and even starting services bound to a VRF.
- VRF route leaking between tenants creates a security boundary violation. Only leak specific prefixes (shared services) and control with route-maps.
- On NX-OS, the `management` VRF exists by default for mgmt0. Do not delete or repurpose it.
- For VRF-Lite without MPLS, IOS-XE `vrf definition` does not require RD/RT. IOS classic `ip vrf` always requires RD.
- In MP-BGP L3VPN, the VPN label identifies the destination VRF on the egress PE. The transport label gets the packet to the egress PE. Both are required.

## See Also

- mpls-vpn, mpls, bgp, ospf, eigrp, network-namespaces, vlan, private-vlans, pbr, segment-routing

## References

- [RFC 4364 — BGP/MPLS IP Virtual Private Networks (L3VPN)](https://www.rfc-editor.org/rfc/rfc4364)
- [RFC 4659 — BGP-MPLS IP VPN Extension for IPv6 VPN](https://www.rfc-editor.org/rfc/rfc4659)
- [RFC 4382 — MPLS/BGP Layer 3 VPN Management Information Base](https://www.rfc-editor.org/rfc/rfc4382)
- [RFC 4760 — Multiprotocol Extensions for BGP-4](https://www.rfc-editor.org/rfc/rfc4760)
- [RFC 4684 — Constrained Route Distribution for BGP/MPLS IP VPN](https://www.rfc-editor.org/rfc/rfc4684)
- [Cisco — VRF-Lite Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_pi/configuration/xe-16/iri-xe-16-book/iri-vrf-lite.html)
- [Cisco — MPLS L3VPN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/mp_l3_vpns/configuration/xe-16/mp-l3-vpns-xe-16-book.html)
- [Juniper — Routing Instances Overview](https://www.juniper.net/documentation/us/en/software/junos/routing-instances/topics/concept/routing-instances-overview.html)
- [Linux Kernel — VRF Documentation](https://www.kernel.org/doc/html/latest/networking/vrf.html)
- [FRRouting — VRF Support](https://docs.frrouting.org/en/latest/vrf.html)
