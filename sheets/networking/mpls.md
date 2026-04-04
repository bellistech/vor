# MPLS (Multiprotocol Label Switching)

Label-based forwarding mechanism that operates between L2 and L3, enabling traffic engineering, VPNs, and fast path switching across service provider and enterprise networks.

## Concepts

### Core Architecture

- **Label:** 20-bit identifier in the MPLS header (between L2 and L3 headers)
- **LSP (Label Switched Path):** Unidirectional path through the MPLS network
- **LER (Label Edge Router):** Ingress/egress router that pushes/pops labels
- **LSR (Label Switch Router):** Transit router that swaps labels
- **FEC (Forwarding Equivalence Class):** Group of packets forwarded the same way (e.g., same destination prefix)

### Label Operations

| Operation | Description                              |
|-----------|------------------------------------------|
| Push      | Add a label at ingress LER               |
| Swap      | Replace label at transit LSR             |
| Pop       | Remove label at egress LER               |
| PHP       | Penultimate Hop Popping: second-to-last router pops the label so egress does plain IP lookup |

### Label Stack

- Multiple labels can be stacked (used in L3VPN, TE tunnels)
- Bottom-of-stack bit (S=1) indicates the last label
- Outer label: transport across backbone
- Inner label: VPN or service identifier

## LDP (Label Distribution Protocol)

### Concepts

- Discovers neighbors via UDP 646 multicast (224.0.0.2)
- Establishes TCP 646 sessions for label exchange
- Maps FECs to labels (one label per IGP prefix)

### FRRouting LDP Configuration

```
mpls ldp
 # Router-id (typically loopback address)
 router-id 10.0.0.1
 # Ordered-control mode
 ordered-control
 #
 address-family ipv4
  # Announce this address to peers
  discovery transport-address 10.0.0.1
  #
  interface eth0
  interface eth1
  exit
 exit-address-family
```

### LDP Show Commands

```bash
# LDP neighbors and session state
vtysh -c "show mpls ldp neighbor"

# Label bindings (local and remote labels per FEC)
vtysh -c "show mpls ldp binding"

# LDP discovery (hello adjacencies)
vtysh -c "show mpls ldp discovery"

# LDP interface status
vtysh -c "show mpls ldp interface"
```

## RSVP-TE (Traffic Engineering)

### Concepts

- Builds explicit LSPs with bandwidth reservation
- Uses RSVP PATH and RESV messages to signal tunnels
- Supports CSPF (Constrained SPF) for path calculation
- Fast reroute (FRR) with facility backup or one-to-one backup

### Tunnel Configuration (Cisco IOS)

```
! MPLS TE requires OSPF or IS-IS with TE extensions
router ospf 1
 mpls traffic-eng router-id Loopback0
 mpls traffic-eng area 0

interface Tunnel0
 ip unnumbered Loopback0
 tunnel mode mpls traffic-eng
 tunnel destination 10.0.0.5
 tunnel mpls traffic-eng bandwidth 100000
 tunnel mpls traffic-eng path-option 1 explicit name PATH1
 tunnel mpls traffic-eng path-option 2 dynamic
 tunnel mpls traffic-eng fast-reroute

! Explicit path definition
ip explicit-path name PATH1 enable
 next-address 10.1.1.2
 next-address 10.2.2.2
```

## L3VPN (MPLS VPN)

### Concepts

- **VRF (Virtual Routing and Forwarding):** Isolated routing table per customer
- **RD (Route Distinguisher):** Makes customer prefixes unique in BGP (format: ASN:nn or IP:nn)
- **RT (Route Target):** Controls import/export of routes between VRFs via MP-BGP
- **PE (Provider Edge):** Router running VRFs and MP-BGP
- **CE (Customer Edge):** Customer router peering with PE

### VRF Configuration (FRRouting)

```
# Create VRF
vrf CUSTOMER-A
 # Apply VRF to an interface
 exit

interface eth2
 vrf CUSTOMER-A
 ip address 192.168.100.1/24

# BGP VPNv4 peering between PEs
router bgp 65000
 address-family vpnv4 unicast
  neighbor 10.0.0.2 activate
 exit-address-family

# Per-VRF BGP instance
router bgp 65000 vrf CUSTOMER-A
 address-family ipv4 unicast
  rd 65000:100
  rt vpn both 65000:100
  redistribute connected
  label vpn export auto
 exit-address-family
```

## L2VPN

### Pseudowire (Point-to-Point)

```
! Cisco IOS pseudowire config
interface GigabitEthernet0/1
 xconnect 10.0.0.5 100 encapsulation mpls
 ! 100 = VC ID, must match on both ends
```

### VPLS (Multipoint)

```
! Cisco IOS VPLS
l2vpn vfi context VPLS-A
 vpn id 100
 member 10.0.0.5 encapsulation mpls
 member 10.0.0.6 encapsulation mpls
```

## Linux MPLS

### Kernel Configuration

```bash
# Enable MPLS in the kernel
sudo sysctl -w net.mpls.conf.eth0.input=1
sudo sysctl -w net.mpls.conf.lo.input=1
# Set maximum label table size
sudo sysctl -w net.mpls.platform_labels=100000

# Persistent settings in /etc/sysctl.d/mpls.conf
```

```
net.mpls.conf.eth0.input = 1
net.mpls.conf.lo.input = 1
net.mpls.platform_labels = 100000
```

### iproute2 MPLS Commands

```bash
# Add MPLS label route (swap label 100 to 200, forward via eth1)
ip -f mpls route add 100 as 200 via inet 10.1.1.2 dev eth1

# Push label (encapsulate IP traffic with MPLS label)
ip route add 10.10.0.0/16 encap mpls 100 via 10.1.1.2 dev eth1

# Pop label (decapsulate and do IP lookup)
ip -f mpls route add 100 via inet 10.1.1.2 dev eth1

# Label stack (push multiple labels)
ip route add 10.10.0.0/16 encap mpls 100/200 via 10.1.1.2 dev eth1

# Show MPLS routes
ip -f mpls route show

# Show MPLS stats
ip -s -f mpls route show
```

## MPLS Forwarding Table

```bash
# FRRouting: show MPLS forwarding table (LFIB)
vtysh -c "show mpls forwarding-table"

# Cisco IOS equivalents
# show mpls forwarding-table
# show mpls ldp bindings
# show mpls interfaces
# show mpls traffic-eng tunnels
```

## Troubleshooting

```bash
# Verify MPLS is enabled on interfaces
vtysh -c "show mpls ldp interface"

# Check LDP session is established
vtysh -c "show mpls ldp neighbor"

# Verify label bindings exist for destination prefixes
vtysh -c "show mpls ldp binding"

# Check the LFIB for correct label operations
vtysh -c "show mpls forwarding-table"

# Trace the label switched path
# Cisco: traceroute mpls ipv4 10.0.0.5/32
# Linux: use mtr or traceroute with MPLS extensions

# Verify VRF routes on PE
vtysh -c "show ip route vrf CUSTOMER-A"

# Check VPNv4 BGP table
vtysh -c "show bgp vpnv4 unicast all"

# Verify transport label (LDP) and VPN label (BGP) are both present
vtysh -c "show bgp vpnv4 unicast all 192.168.100.0/24"
```

## Tips

- LDP relies on the underlying IGP; if the IGP route disappears, the LDP label binding is withdrawn.
- Always use `router-id` and `transport-address` set to loopback for LDP stability.
- PHP (penultimate hop popping) is the default behavior; the egress router does a plain IP lookup, which is usually fine.
- MPLS MTU overhead is 4 bytes per label; adjust interface MTU accordingly (e.g., 1504 for single label, 1508 for two).
- In L3VPN, RD makes routes unique but RT controls route distribution; they serve different purposes.
- When troubleshooting L3VPN, verify the full chain: CE-PE adjacency, VRF route, RT import/export, VPNv4 BGP, LDP label.
- Linux kernel MPLS support is functional but limited compared to FRRouting or Cisco; use FRRouting for LDP signaling.
- RSVP-TE tunnels are powerful but operationally complex; consider Segment Routing (SR-MPLS) as a modern alternative.
- Label 0 is the explicit-null label (preserves QoS bits); label 3 is implicit-null (PHP).
- Always verify that `net.mpls.platform_labels` is set high enough on Linux; the default is often 0 (MPLS disabled).

## See Also

- bgp, vxlan, ipsec, ospf, is-is

## References

- [RFC 3031 — Multiprotocol Label Switching Architecture](https://www.rfc-editor.org/rfc/rfc3031)
- [RFC 3032 — MPLS Label Stack Encoding](https://www.rfc-editor.org/rfc/rfc3032)
- [RFC 5036 — LDP Specification](https://www.rfc-editor.org/rfc/rfc5036)
- [RFC 3209 — RSVP-TE: Extensions to RSVP for LSP Tunnels](https://www.rfc-editor.org/rfc/rfc3209)
- [RFC 4364 — BGP/MPLS IP Virtual Private Networks (L3VPN)](https://www.rfc-editor.org/rfc/rfc4364)
- [RFC 4761 — Virtual Private LAN Service (VPLS) Using BGP](https://www.rfc-editor.org/rfc/rfc4761)
- [FRRouting LDP Documentation](https://docs.frrouting.org/en/latest/ldpd.html)
- [Linux Kernel — MPLS Documentation](https://www.kernel.org/doc/html/latest/networking/mpls-sysctl.html)
- [Cisco MPLS Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/mp_basic/configuration/xe-16/mp-basic-xe-16-book.html)
- [Juniper MPLS Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/mpls/topics/topic-map/mpls-overview.html)
- [Nokia SR OS MPLS Documentation](https://documentation.nokia.com/sr/)
