# VXLAN (Virtual Extensible LAN)

Overlay encapsulation protocol that extends L2 segments over an L3 underlay, enabling scalable multi-tenant network virtualization with up to 16 million logical networks.

## Concepts

### Core Components

- **VNI (VXLAN Network Identifier):** 24-bit ID (0-16777215), identifies the L2 segment (like a VLAN but vastly larger address space)
- **VTEP (VXLAN Tunnel Endpoint):** Encapsulates/decapsulates VXLAN traffic; can be a switch, hypervisor, or software bridge
- **Overlay:** The virtual L2 network carried inside VXLAN
- **Underlay:** The physical L3 network providing IP connectivity between VTEPs

### Encapsulation

- Original L2 frame is wrapped in: VXLAN header (8B) + UDP (8B) + outer IP (20B) + outer Ethernet (14B)
- Total overhead: 50 bytes (54 with outer VLAN tag)
- Destination UDP port: 4789 (IANA standard)
- Source UDP port: hash of inner frame headers (for ECMP distribution)

### Control Plane Options

| Method               | Description                                          |
|----------------------|------------------------------------------------------|
| Multicast            | VTEPs join a multicast group per VNI for BUM traffic |
| Unicast head-end rep | Ingress VTEP replicates BUM to a static list of peers|
| BGP EVPN             | Dynamic MAC/IP learning via BGP (preferred)          |

## Linux VXLAN Configuration

### Basic VXLAN Interface (Multicast)

```bash
# Create VXLAN interface with multicast-based learning
ip link add vxlan100 type vxlan \
  id 100 \                    # VNI
  group 239.1.1.1 \           # Multicast group for BUM traffic
  dev eth0 \                  # Underlay interface
  dstport 4789 \              # Standard VXLAN UDP port
  ttl 64

# Bring up and add to a bridge
ip link set vxlan100 up
ip link set vxlan100 master br0
```

### VXLAN with Unicast (Head-End Replication)

```bash
# Create VXLAN interface without multicast
ip link add vxlan100 type vxlan \
  id 100 \
  local 10.0.0.1 \            # Local VTEP IP
  dstport 4789 \
  nolearning                   # Disable data-plane learning (use FDB only)

ip link set vxlan100 up
ip link set vxlan100 master br0

# Manually populate FDB with remote VTEPs for BUM traffic
bridge fdb append 00:00:00:00:00:00 dev vxlan100 dst 10.0.0.2
bridge fdb append 00:00:00:00:00:00 dev vxlan100 dst 10.0.0.3

# Add known MAC-to-VTEP entries
bridge fdb add aa:bb:cc:dd:ee:01 dev vxlan100 dst 10.0.0.2
```

### Bridge Configuration

```bash
# Create a bridge for the VXLAN segment
ip link add br0 type bridge
ip link set br0 up

# Enable VLAN-aware bridge (optional, for multiple VNIs on one bridge)
ip link add br0 type bridge vlan_filtering 1

# Add local interfaces and VXLAN to the bridge
ip link set eth1 master br0
ip link set vxlan100 master br0

# Disable bridge learning on VXLAN port (when using EVPN)
bridge link set dev vxlan100 learning off
```

## BGP EVPN Integration

### Concepts

- EVPN uses MP-BGP to distribute MAC and IP reachability
- Replaces flood-and-learn with control-plane MAC/IP advertisement
- Supports ARP suppression to reduce BUM traffic

### EVPN Route Types

| Type | Name               | Purpose                                    |
|------|--------------------|--------------------------------------------|
| 2    | MAC/IP             | Advertise host MAC and optional IP          |
| 3    | Inclusive Multicast | VTEP discovery and BUM replication tree     |
| 5    | IP Prefix          | Advertise IP subnets (inter-VXLAN routing)  |

### FRRouting EVPN Configuration

```
# Enable BGP with EVPN address family
router bgp 65000
 bgp router-id 10.0.0.1
 neighbor 10.0.0.2 remote-as 65000
 neighbor 10.0.0.2 update-source lo
 #
 address-family l2vpn evpn
  neighbor 10.0.0.2 activate
  # Advertise all locally configured VNIs
  advertise-all-vni
  # Enable ARP/ND suppression
  arp-cache-size 1024
 exit-address-family

# Verify EVPN routes
# vtysh -c "show bgp l2vpn evpn"
# vtysh -c "show bgp l2vpn evpn route type macip"
# vtysh -c "show bgp l2vpn evpn route type multicast"
# vtysh -c "show evpn vni"
```

### Symmetric IRB (Inter-VXLAN Routing)

```
# L3 VNI for inter-subnet routing
ip link add vxlan5000 type vxlan id 5000 local 10.0.0.1 dstport 4789 nolearning
ip link set vxlan5000 master br-l3vni
ip link set vxlan5000 up

# Associate L3 VNI with VRF
vrf TENANT-A
 vni 5000

router bgp 65000 vrf TENANT-A
 address-family l2vpn evpn
  advertise ipv4 unicast
  advertise ipv6 unicast
 exit-address-family
```

## Show and Inspection Commands

```bash
# Display VXLAN interface details (VNI, group, port, source)
ip -d link show vxlan100

# Show FDB entries (MAC-to-VTEP mappings)
bridge fdb show dev vxlan100

# Show bridge MAC table
bridge fdb show br br0

# FRRouting EVPN commands
vtysh -c "show evpn vni"
vtysh -c "show evpn vni 100"
vtysh -c "show evpn mac vni 100"
vtysh -c "show evpn arp-cache vni 100"
vtysh -c "show bgp l2vpn evpn summary"
vtysh -c "show bgp l2vpn evpn route type macip"
```

## Troubleshooting

### Verify Underlay Connectivity

```bash
# VXLAN uses UDP 4789; verify underlay IP reachability between VTEPs
ping 10.0.0.2

# Check for firewall rules blocking UDP 4789
iptables -L -n | grep 4789
```

### MTU Issues

```bash
# VXLAN adds 50 bytes of overhead
# Underlay MTU must be >= overlay MTU + 50
# If overlay uses 1500, underlay needs at least 1550 (use 9000 jumbo)
ip link show eth0 | grep mtu

# Set underlay MTU to accommodate VXLAN overhead
ip link set eth0 mtu 9000
```

### Capture VXLAN Traffic

```bash
# Capture encapsulated traffic on underlay
tcpdump -i eth0 -nn udp port 4789

# Decode inner VXLAN payload
tcpdump -i eth0 -nn udp port 4789 -e -v
```

### FDB Verification

```bash
# Missing FDB entries = traffic black-holed
# Verify BUM replication entries (dst 00:00:00:00:00:00)
bridge fdb show dev vxlan100 | grep "00:00:00:00:00:00"

# Verify known host entries
bridge fdb show dev vxlan100 | grep -v permanent
```

## Tips

- Always use jumbo frames (MTU 9000) on the underlay to avoid fragmentation; VXLAN + 1500 byte frames need at least 1550.
- BGP EVPN is the recommended control plane over multicast for production deployments; it scales better and provides ARP suppression.
- Set `nolearning` on VXLAN interfaces when using EVPN to avoid stale data-plane learned entries conflicting with control-plane entries.
- Use `arp-suppress` on bridge ports to reduce BUM traffic by responding to ARP from the VTEP's cache.
- Source UDP port entropy (hash of inner headers) is critical for ECMP load balancing across the underlay.
- In Cumulus Linux / NVIDIA SONiC, VXLAN and EVPN are first-class features with simplified configuration.
- Monitor `bridge fdb` regularly; stale entries cause intermittent connectivity issues that are hard to diagnose.
- For troubleshooting, `tcpdump` on UDP 4789 on the underlay interface is the fastest way to confirm encapsulation is working.
- VXLAN VNI and VLAN are independent namespaces; you map them together on the bridge. Keep a clear mapping document.
- When using symmetric IRB, the L3 VNI must be consistent across all VTEPs in the same VRF/tenant.

## References

- [RFC 7348 — VXLAN: A Framework for Overlaying Virtualized Layer 2 Networks over Layer 3 Networks](https://www.rfc-editor.org/rfc/rfc7348)
- [RFC 8365 — A Network Virtualization Overlay Solution Using EVPN](https://www.rfc-editor.org/rfc/rfc8365)
- [Linux Kernel VXLAN Documentation](https://www.kernel.org/doc/html/latest/networking/vxlan.html)
- [man ip-link — VXLAN Type](https://man7.org/linux/man-pages/man8/ip-link.8.html)
- [FRRouting EVPN/VXLAN Documentation](https://docs.frrouting.org/en/latest/evpn.html)
- [Cisco Nexus 9000 VXLAN Configuration](https://www.cisco.com/c/en/us/support/docs/switches/nexus-9000-series-switches/118978-config-vxlan-00.html)
- [Juniper VXLAN Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/evpn/topics/topic-map/sdn-vxlan.html)
- [Arista EOS VXLAN Configuration Guide](https://www.arista.com/en/um-eos/eos-vxlan)
- [Open vSwitch — VXLAN Tunnels](https://docs.openvswitch.org/en/latest/howto/vxlan/)
