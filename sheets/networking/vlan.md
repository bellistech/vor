# VLAN (Virtual Local Area Network)

Layer 2 broadcast domain segmentation using IEEE 802.1Q tagging that logically partitions a physical switch into isolated networks, each with its own broadcast, multicast, and unknown unicast flooding scope.

## 802.1Q Tag Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          TPID (0x8100)        | PCP |D|       VLAN ID         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

TPID:    Tag Protocol Identifier (0x8100 = 802.1Q)
PCP:     Priority Code Point (3 bits, 0-7 for QoS / CoS)
DEI:     Drop Eligible Indicator (1 bit)
VLAN ID: 12 bits (0-4095)
         VID 0:    Priority-tagged frame (no VLAN assignment)
         VID 1:    Default VLAN
         VID 4095: Reserved
         Usable:   1-4094

# Tag adds 4 bytes to Ethernet frame
# Tagged frame: 14 (header) + 4 (tag) + payload + 4 (FCS) = max 1522 bytes
# Untagged frame: 14 (header) + payload + 4 (FCS) = max 1518 bytes
```

## VLAN Ranges

```
Range          VIDs        Description
──────────────────────────────────────────────────────────────────
Normal         1-1005      Stored in VLAN database (vlan.dat on Cisco)
Extended       1006-4094   Require VTP transparent mode or VTPv3
Reserved       0, 4095     Cannot be used for traffic
Default        1           Cannot be deleted, all ports start here
Native         varies      Untagged traffic on trunk (default: VLAN 1)
```

## Port Types

### Access Ports

```bash
# Carries traffic for a single VLAN
# Frames are untagged (host doesn't know about VLANs)

# Cisco
interface GigabitEthernet0/1
  switchport mode access
  switchport access vlan 10

# Arista
interface Ethernet1
  switchport mode access
  switchport access vlan 10

# Juniper
set interfaces ge-0/0/0 unit 0 family ethernet-switching vlan members vlan10
```

### Trunk Ports

```bash
# Carries traffic for multiple VLANs with 802.1Q tags
# Used for switch-to-switch, switch-to-router, switch-to-VM host

# Cisco
interface GigabitEthernet0/24
  switchport mode trunk
  switchport trunk encapsulation dot1q    # not needed on newer switches
  switchport trunk allowed vlan 10,20,30
  switchport trunk native vlan 99         # untagged VLAN

# Arista
interface Ethernet49
  switchport mode trunk
  switchport trunk allowed vlan 10,20,30
  switchport trunk native vlan 99

# Juniper
set interfaces ge-0/0/0 unit 0 family ethernet-switching interface-mode trunk
set interfaces ge-0/0/0 unit 0 family ethernet-switching vlan members [vlan10 vlan20 vlan30]
set interfaces ge-0/0/0 native-vlan-id 99
```

### Trunk Allowed VLANs

```bash
# Cisco — manage allowed VLANs on trunk
switchport trunk allowed vlan 10,20,30        # set exactly
switchport trunk allowed vlan add 40          # add VLAN
switchport trunk allowed vlan remove 20       # remove VLAN
switchport trunk allowed vlan all             # allow all
switchport trunk allowed vlan except 1        # all except VLAN 1
switchport trunk allowed vlan none            # block all (useless trunk)

# Always prune unnecessary VLANs from trunks
# Reduces broadcast domain scope and limits blast radius
```

## Native VLAN

```
# The native VLAN carries untagged frames on a trunk
# Default native VLAN is 1 (same as default access VLAN)
# SECURITY: Change native VLAN away from 1 to prevent VLAN hopping attacks

# VLAN hopping attack (double tagging):
# Attacker sends frame with two 802.1Q tags:
#   Outer tag: native VLAN (stripped by first switch)
#   Inner tag: target VLAN (forwarded by second switch)
# Only works when native VLAN matches attacker's access VLAN

# Prevention:
# 1. Set native VLAN to an unused VLAN (e.g., 999)
# 2. Tag native VLAN explicitly on all trunks
# 3. Never use VLAN 1 for user traffic
```

```bash
# Cisco — tag native VLAN (prevents double-tagging attack)
vlan dot1q tag native                        # global: tag native on all trunks

# Set native to unused VLAN
interface GigabitEthernet0/24
  switchport trunk native vlan 999
```

## Linux VLAN Configuration

```bash
# Create VLAN interface (iproute2)
ip link add link eth0 name eth0.10 type vlan id 10
ip link set eth0.10 up
ip addr add 192.168.10.1/24 dev eth0.10

# Remove VLAN interface
ip link delete eth0.10

# View VLAN interfaces
ip -d link show eth0.10
cat /proc/net/vlan/eth0.10

# Create multiple VLANs
for vid in 10 20 30; do
    ip link add link eth0 name eth0.$vid type vlan id $vid
    ip link set eth0.$vid up
done

# VLAN on bonded interface
ip link add link bond0 name bond0.10 type vlan id 10
ip link set bond0.10 up
ip addr add 192.168.10.1/24 dev bond0.10
```

### Netplan (Ubuntu)

```yaml
# /etc/netplan/01-vlans.yaml
network:
  version: 2
  ethernets:
    eth0: {}
  vlans:
    eth0.10:
      id: 10
      link: eth0
      addresses:
        - 192.168.10.1/24
    eth0.20:
      id: 20
      link: eth0
      addresses:
        - 192.168.20.1/24
```

### NetworkManager (nmcli)

```bash
# Create VLAN
nmcli connection add type vlan con-name vlan10 dev eth0 id 10 \
    ipv4.addresses 192.168.10.1/24 ipv4.method manual

# View VLANs
nmcli connection show | grep vlan

# Delete VLAN
nmcli connection delete vlan10
```

## Inter-VLAN Routing

### Router on a Stick

```bash
# Single router interface with sub-interfaces, one per VLAN

# Cisco router
interface GigabitEthernet0/0.10
  encapsulation dot1q 10
  ip address 192.168.10.1 255.255.255.0

interface GigabitEthernet0/0.20
  encapsulation dot1q 20
  ip address 192.168.20.1 255.255.255.0

# Linux router
ip link add link eth0 name eth0.10 type vlan id 10
ip link add link eth0 name eth0.20 type vlan id 20
ip addr add 192.168.10.1/24 dev eth0.10
ip addr add 192.168.20.1/24 dev eth0.20
ip link set eth0.10 up
ip link set eth0.20 up
sysctl -w net.ipv4.ip_forward=1
```

### Layer 3 Switch (SVI)

```bash
# Switch Virtual Interface — preferred for high-performance inter-VLAN routing
# Routing happens in hardware (ASIC), not CPU

# Cisco
ip routing

interface Vlan10
  ip address 192.168.10.1 255.255.255.0
  no shutdown

interface Vlan20
  ip address 192.168.20.1 255.255.255.0
  no shutdown

# Arista
ip routing

interface Vlan10
  ip address 192.168.10.1/24
```

## Linux Bridge with VLANs (VLAN-Aware Bridge)

```bash
# Modern Linux bridge with VLAN filtering

# Create bridge
ip link add br0 type bridge vlan_filtering 1
ip link set br0 up

# Add ports
ip link set eth0 master br0
ip link set eth1 master br0

# Configure port as access (VLAN 10)
bridge vlan add dev eth0 vid 10 pvid untagged
bridge vlan del dev eth0 vid 1                  # remove default VLAN 1

# Configure port as trunk (VLANs 10, 20, 30)
bridge vlan add dev eth1 vid 10
bridge vlan add dev eth1 vid 20
bridge vlan add dev eth1 vid 30

# Show VLAN configuration
bridge vlan show
bridge vlan show dev eth0

# Bridge self (for routing on the bridge)
bridge vlan add dev br0 vid 10 self
ip link add link br0 name br0.10 type vlan id 10
ip addr add 192.168.10.1/24 dev br0.10
```

## Monitoring & Troubleshooting

```bash
# Cisco
show vlan brief                              # all VLANs and port assignments
show vlan id 10                              # specific VLAN detail
show interfaces trunk                        # trunk port status and allowed VLANs
show interfaces switchport                   # per-port VLAN config
show mac address-table vlan 10              # MAC table for VLAN 10

# Linux
bridge vlan show                             # VLAN assignments per port
cat /proc/net/vlan/config                    # list all VLAN interfaces
ip -d link show type vlan                    # all VLAN interfaces with details

# Capture tagged traffic
tcpdump -i eth0 -e -n vlan                  # show VLAN tags
tcpdump -i eth0 -e -n vlan 10               # capture only VLAN 10
tcpdump -i eth0.10 -n                       # capture on VLAN sub-interface

# Check VLAN tagging issues
# If ping works on native VLAN but not on tagged VLAN:
# 1. Verify trunk allows that VLAN on both sides
# 2. Verify VLAN exists on both switches
# 3. Verify MTU (tagged frames are 4 bytes larger)
```

## Tips

- Always change the native VLAN away from VLAN 1 and assign it to an unused VLAN. VLAN 1 is the default on all ports and a common target for VLAN hopping attacks using double-tagged frames.
- Prune unnecessary VLANs from trunk links. A trunk carrying all 4094 VLANs floods broadcast, multicast, and unknown unicast traffic for every VLAN across every trunk, wasting bandwidth and CPU.
- VLAN-aware Linux bridges (`vlan_filtering 1`) are the modern approach. Legacy per-VLAN bridge configs (one bridge per VLAN) do not scale and waste memory with duplicate MAC tables.
- The 802.1Q tag adds 4 bytes to each frame. If an end host sends a 1500-byte frame and the switch tags it, the tagged frame is 1504 bytes. Trunks must support "baby giant" frames (1522 bytes) or you get silent drops.
- Inter-VLAN routing via SVIs (Layer 3 switch) is hardware-accelerated and handles millions of packets per second. Router-on-a-stick routes in CPU and is only suitable for small environments under 100 Mbps.
- VTP (VLAN Trunking Protocol) can delete all VLANs on every switch if a device with a higher revision number is plugged in. Use VTP transparent mode or VTPv3 with a primary server to prevent VLAN wipeouts.
- When using VLANs with VMware, Hyper-V, or KVM, the virtual switch must be configured for trunk mode (pass-through) and guest NICs must be VLAN-aware, or the hypervisor must tag/untag on behalf of guests.
- QinQ (802.1ad) stacks two VLAN tags, using TPID 0x88A8 for the outer (service) tag and 0x8100 for the inner (customer) tag. This gives service providers up to $4094 \times 4094 \approx 16.7$ million unique identifiers.
- On Cisco, `switchport trunk allowed vlan` is NOT additive. Running it twice overwrites the first command. Use `switchport trunk allowed vlan add` to add VLANs without removing existing ones.
- Voice VLANs (`switchport voice vlan 100`) allow IP phones and PCs on the same access port. The phone tags its traffic with the voice VLAN; the PC sends untagged traffic in the access VLAN.

## See Also

- ethernet, stp, lacp, ip, vxlan, iptables

## References

- [IEEE 802.1Q-2022 — VLANs and Bridges](https://standards.ieee.org/standard/802_1Q-2022.html)
- [IEEE 802.1ad — Provider Bridging (QinQ)](https://standards.ieee.org/standard/802_1ad-2005.html)
- [Cisco — VLAN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst9300/software/release/16-12/configuration_guide/vlan/b_1612_vlan_9300_cg.html)
- [Linux Kernel — VLAN Documentation](https://www.kernel.org/doc/html/latest/networking/vlan.html)
- [man ip-link (vlan)](https://man7.org/linux/man-pages/man8/ip-link.8.html)
