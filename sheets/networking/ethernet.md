# Ethernet (IEEE 802.3)

Layer 2 LAN technology providing framed data transmission between directly connected nodes using MAC addresses, supporting speeds from 10 Mbps to 400 Gbps.

## Frame Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Preamble (7 bytes)                     |
+               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|               |      SFD      |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                  Destination MAC Address (6 bytes)             |
+                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                               |                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                    Source MAC Address (6 bytes)                |
+                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                               | EtherType / Length (2 bytes)  |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                     Payload (46-1500 bytes)                   +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    FCS / CRC-32 (4 bytes)                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **Preamble**: 7 bytes of 10101010 pattern for clock synchronization
- **SFD (Start Frame Delimiter)**: 1 byte (10101011), marks start of frame
- **Destination MAC**: 6 bytes, target hardware address
- **Source MAC**: 6 bytes, sender hardware address
- **EtherType/Length**: 2 bytes. Values >= 0x0600 = EtherType (protocol ID). Values <= 0x05DC = Length (IEEE 802.3)
- **Payload**: 46-1500 bytes (padded to 46 minimum). 9000 bytes with jumbo frames
- **FCS**: 4-byte CRC-32 checksum over destination MAC through payload

Minimum frame: 64 bytes (excluding preamble/SFD). Maximum: 1518 bytes (1522 with VLAN tag).

## EtherTypes

```
EtherType   Protocol
─────────────────────────────────────────
0x0800      IPv4
0x0806      ARP (Address Resolution Protocol)
0x8100      802.1Q VLAN-tagged frame
0x86DD      IPv6
0x8847      MPLS unicast
0x8848      MPLS multicast
0x8863      PPPoE Discovery
0x8864      PPPoE Session
0x88A8      802.1ad QinQ (provider bridging)
0x88CC      LLDP (Link Layer Discovery Protocol)
0x88E5      802.1AE MACsec
0x88F7      PTP (Precision Time Protocol, IEEE 1588)
0x8906      FCoE (Fibre Channel over Ethernet)
0x9000      Loopback (configuration testing)
```

## MAC Addresses

```
# Format: 6 bytes, written as hex pairs
AA:BB:CC:DD:EE:FF          # colon notation (Linux, macOS)
AA-BB-CC-DD-EE-FF          # hyphen notation (Windows)
AABB.CCDD.EEFF             # dot notation (Cisco)

# Structure
[OUI (3 bytes)][NIC-specific (3 bytes)]
# OUI = Organizationally Unique Identifier (assigned by IEEE)

# Bit flags in first byte
# Bit 0 (LSB): 0 = unicast, 1 = multicast
# Bit 1:       0 = globally unique (OUI), 1 = locally administered

# Special addresses
FF:FF:FF:FF:FF:FF          # Broadcast — delivered to all hosts on segment
01:00:5E:xx:xx:xx          # IPv4 multicast (lower 23 bits of multicast IP)
33:33:xx:xx:xx:xx          # IPv6 multicast (lower 32 bits of multicast IP)
01:80:C2:00:00:00          # STP bridge group address
01:80:C2:00:00:0E          # LLDP multicast
```

### OUI Lookup

```bash
# Look up manufacturer from MAC address
# First 3 bytes (OUI) identify the vendor

# Online: https://standards-oui.ieee.org/
# Local database (on many Linux distros)
grep -i "AA:BB:CC" /usr/share/ieee-data/oui.txt

# Common OUIs
# 00:50:56 — VMware
# 08:00:27 — VirtualBox
# 52:54:00 — QEMU/KVM
# 02:42:xx — Docker containers
# 00:15:5D — Hyper-V
```

## 802.1Q VLAN Tagging

```
# 802.1Q inserts a 4-byte tag after the source MAC address
# Tagged frame max size: 1522 bytes (standard 1518 + 4-byte tag)

VLAN Tag structure (4 bytes):
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     TPID (0x8100)            | PCP |D|        VID            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  16 bits                        3   1    12 bits

TPID  — Tag Protocol Identifier (0x8100 for 802.1Q)
PCP   — Priority Code Point (3 bits, 0-7, for QoS / CoS)
DEI   — Drop Eligible Indicator (1 bit, formerly CFI)
VID   — VLAN Identifier (12 bits, 0-4095)
        VID 0 = priority tag only (no VLAN)
        VID 1 = default VLAN
        VID 4095 = reserved
```

### Port Types

```
# Access port — carries one VLAN, frames are untagged
# Trunk port  — carries multiple VLANs, frames are tagged
# Native VLAN — untagged traffic on a trunk is assigned to this VLAN
#               (default VLAN 1 on most switches — change it for security)

# QinQ (802.1ad) — double tagging for provider networks
# Outer tag: TPID 0x88A8 (service VLAN / S-VLAN)
# Inner tag: TPID 0x8100 (customer VLAN / C-VLAN)
```

## Speeds and Standards

```
Standard          Speed      Media / Cable               Max Distance
───────────────────────────────────────────────────────────────────────
10BASE-T          10 Mbps    Cat 3+ UTP, 2 pairs         100 m
100BASE-TX        100 Mbps   Cat 5+ UTP, 2 pairs         100 m
1000BASE-T        1 Gbps     Cat 5e+ UTP, 4 pairs        100 m
1000BASE-SX       1 Gbps     MMF (850nm)                 550 m
1000BASE-LX       1 Gbps     SMF (1310nm)                5 km
10GBASE-T         10 Gbps    Cat 6a+ UTP, 4 pairs        100 m
10GBASE-SR        10 Gbps    MMF (850nm)                 300 m
10GBASE-LR        10 Gbps    SMF (1310nm)                10 km
25GBASE-CR        25 Gbps    Twinax DAC                  5 m
25GBASE-SR        25 Gbps    MMF (850nm)                 100 m
40GBASE-CR4       40 Gbps    Twinax DAC (4 lanes)        7 m
40GBASE-SR4       40 Gbps    MMF (4x 850nm)              150 m
100GBASE-CR4      100 Gbps   Twinax DAC (4 lanes)        5 m
100GBASE-SR4      100 Gbps   MMF (4x 850nm)              100 m
100GBASE-LR4      100 Gbps   SMF (4x WDM)               10 km
400GBASE-SR8      400 Gbps   MMF (8x 850nm)              100 m
400GBASE-DR4      400 Gbps   SMF (4x 1310nm)             500 m
```

### Autonegotiation

```bash
# Autoneg lets both ends agree on speed, duplex, and flow control
# Both sides advertise capabilities; highest common mode is selected

# Check autoneg status
ethtool eth0 | grep -i auto

# Force speed/duplex (disable autoneg)
ethtool -s eth0 speed 1000 duplex full autoneg off

# Re-enable autoneg
ethtool -s eth0 autoneg on
```

## Jumbo Frames

```bash
# Jumbo frames: MTU > 1500 bytes (typically 9000)
# Reduces per-packet overhead for bulk transfers
# ALL devices on the L2 path must support the same MTU

# Set MTU
ip link set dev eth0 mtu 9000

# Verify MTU
ip link show eth0 | grep mtu

# Test end-to-end jumbo frame support
ping -M do -s 8972 192.168.1.1     # 8972 + 20 IP + 8 ICMP = 9000

# Persistent (varies by distro)
# /etc/network/interfaces:  mtu 9000
# /etc/sysconfig/network-scripts/ifcfg-eth0:  MTU=9000
# netplan: ethernets: eth0: mtu: 9000
```

## Link Aggregation (802.3ad / LACP)

```bash
# Bond multiple physical links into one logical link
# Increases bandwidth and provides redundancy

# Bonding modes
# mode 0 (balance-rr)    — Round-robin, requires switch support
# mode 1 (active-backup) — Only one active link, failover
# mode 2 (balance-xor)   — XOR hash on MAC addresses
# mode 3 (broadcast)     — Send on all interfaces
# mode 4 (802.3ad/LACP)  — Dynamic aggregation with LACP, requires switch support
# mode 5 (balance-tlb)   — Adaptive transmit load balancing
# mode 6 (balance-alb)   — Adaptive load balancing (tx + rx)

# Create bond with LACP
ip link add bond0 type bond mode 802.3ad
ip link set eth0 master bond0
ip link set eth1 master bond0
ip link set bond0 up
ip addr add 192.168.1.10/24 dev bond0

# Check bond status
cat /proc/net/bonding/bond0

# LACP hash policy (determines how traffic is distributed)
# layer2        — src/dst MAC (default)
# layer3+4      — src/dst IP + port (best for IP traffic)
# layer2+3      — src/dst MAC + IP
echo layer3+4 > /sys/class/net/bond0/bonding/xmit_hash_policy
```

## Spanning Tree Protocol

```
# STP (802.1D) prevents L2 loops by blocking redundant paths
# RSTP (802.1w) — Rapid STP, converges in seconds vs 30-50s for STP
# MSTP (802.1s) — Multiple STP, separate trees per VLAN group

# Port states (STP):
# Disabled    — Administratively down
# Blocking    — Receives BPDUs only, no data forwarding
# Listening   — Processing BPDUs, not learning MACs
# Learning    — Learning MACs, not forwarding data
# Forwarding  — Fully operational

# Port states (RSTP):
# Discarding  — Combines blocking + listening
# Learning    — Learning MACs
# Forwarding  — Fully operational

# Port roles:
# Root port       — Best path to root bridge (one per non-root switch)
# Designated port — Best port on a segment toward root (forwards traffic)
# Alternate port  — Backup path to root (RSTP fast failover)
# Backup port     — Redundant path on same segment
```

## Linux Configuration

### Interface Management

```bash
# List interfaces
ip link show                       # all interfaces
ip -s link show eth0               # with statistics

# Bring interface up/down
ip link set eth0 up
ip link set eth0 down

# Set MAC address
ip link set eth0 address aa:bb:cc:dd:ee:ff

# Create VLAN interface
ip link add link eth0 name eth0.100 type vlan id 100
ip addr add 192.168.100.1/24 dev eth0.100
ip link set eth0.100 up

# Delete VLAN interface
ip link del eth0.100
```

### Bridge (Software Switch)

```bash
# Create bridge
ip link add br0 type bridge
ip link set br0 up

# Add interfaces to bridge
ip link set eth0 master br0
ip link set eth1 master br0

# Assign IP to bridge
ip addr add 192.168.1.1/24 dev br0

# Enable STP on bridge
ip link set br0 type bridge stp_state 1

# Show bridge info
bridge link show
bridge vlan show

# VLAN filtering on bridge
ip link set br0 type bridge vlan_filtering 1
bridge vlan add vid 100 dev eth0            # allow VLAN 100 on eth0
bridge vlan add vid 100 dev eth0 pvid untagged  # set as native/access VLAN
```

### ethtool

```bash
# Interface info
ethtool eth0                       # speed, duplex, autoneg, link status

# Driver/firmware info
ethtool -i eth0                    # driver name, version, firmware

# Interface statistics
ethtool -S eth0                    # NIC-level counters (rx/tx errors, drops, etc.)

# Ring buffer sizes
ethtool -g eth0                    # show current and max ring buffer sizes
ethtool -G eth0 rx 4096 tx 4096   # set ring buffer sizes

# Offload settings
ethtool -k eth0                    # show offload features
ethtool -K eth0 tso on gso on gro on  # enable offloads

# Test link
ethtool -t eth0 online             # run NIC self-test
```

## MAC Table / FDB

```bash
# Linux bridge forwarding database
bridge fdb show                    # all entries
bridge fdb show br br0             # entries for bridge br0
bridge fdb add aa:bb:cc:dd:ee:ff dev eth0 master static  # static entry
bridge fdb del aa:bb:cc:dd:ee:ff dev eth0 master          # delete entry

# On managed switches (Cisco IOS example)
# show mac address-table
# show mac address-table dynamic
# show mac address-table address AABB.CCDD.EEFF
# clear mac address-table dynamic
```

## Monitoring

```bash
# Interface statistics
ip -s link show eth0               # packets, bytes, errors, drops

# Detailed NIC stats (errors, collisions, CRC)
ethtool -S eth0 | grep -E 'error|drop|crc|collision|fcs'

# Watch for link state changes
ip monitor link                    # real-time link up/down events
dmesg | grep -i 'link'            # kernel log for link changes

# Packet capture at L2
tcpdump -i eth0 -e                 # show Ethernet headers (MAC addresses)
tcpdump -i eth0 ether proto 0x0806 # ARP frames only
tcpdump -i eth0 ether host aa:bb:cc:dd:ee:ff  # filter by MAC

# LLDP neighbors
lldpctl                            # show LLDP neighbor info (lldpd)
```

## Common Issues

### Duplex Mismatch

```bash
# Symptoms: slow transfers, late collisions, CRC errors, incrementing error counters
# Cause: one side autoneg, other side forced — autoneg falls back to half duplex

# Check both ends
ethtool eth0 | grep -E 'Speed|Duplex|Auto'

# Fix: either both autoneg or both forced to same speed/duplex
ethtool -s eth0 speed 1000 duplex full autoneg on
```

### MTU Mismatch

```bash
# Symptoms: small packets work, large fail; TCP works (MSS clamping), pings > MTU fail
# Cause: jumbo frames enabled on some but not all devices in the L2 path

# Verify MTU across the path
ip link show eth0 | grep mtu       # check each hop

# Test
ping -M do -s 1472 192.168.1.1    # test standard MTU
ping -M do -s 8972 192.168.1.1    # test jumbo MTU

# Fix: ensure consistent MTU on ALL devices in the broadcast domain
```

### STP Loops

```bash
# Symptoms: broadcast storm, network saturated, MAC table instability
# Cause: STP disabled or misconfigured, redundant paths without loop prevention

# Enable STP
ip link set br0 type bridge stp_state 1

# On managed switches: enable BPDU guard on access ports
# Drops port if BPDU received (prevents rogue switches)
```

### MAC Flapping

```bash
# Symptoms: switch logs show same MAC learned on different ports rapidly
# Cause: L2 loop, VM migration, misconfigured bonding, or MITM

# Check on Linux bridge
bridge fdb show | grep aa:bb:cc    # look for same MAC on multiple ports

# On managed switches: "MAC flapping" log messages
# Fix: check for loops, verify STP, check bonding config on servers
```

## Tips

- Always use autonegotiation unless you have a specific reason not to. Forcing speed on one side while the other autonegotiates is the number one cause of duplex mismatch, which degrades performance silently.
- When deploying jumbo frames, test the entire L2 path first. A single device with 1500 MTU will silently drop jumbo-sized frames or fragment at L3, causing hard-to-diagnose performance issues.
- VLAN 1 is the default native VLAN on most switches and carries control plane traffic (STP BPDUs, CDP, VTP). Move user traffic off VLAN 1 and change the native VLAN on trunks to reduce attack surface.
- LACP (mode 4) is the only bonding mode that requires switch configuration and provides true aggregation with failover. Mode 1 (active-backup) is the safest choice when you cannot configure the switch.
- For LACP with `layer3+4` hash policy, a single TCP flow will still use only one link. Aggregation improves throughput only when there are multiple concurrent flows.
- MAC address randomization (common on mobile devices) can cause issues with DHCP leases, MAC-based access control, and network monitoring. Plan for this in your network design.
- CRC/FCS errors incrementing on `ethtool -S` indicate physical layer problems: bad cable, failing transceiver, or EMI. Replace the cable first, then the transceiver.
- The minimum Ethernet frame is 64 bytes. Frames smaller than this (runts) indicate a physical or driver issue. The NIC pads payloads shorter than 46 bytes automatically.

## References

- [IEEE 802.3 Ethernet Standard](https://standards.ieee.org/ieee/802.3/)
- [IEEE 802.1Q — VLAN Tagging](https://standards.ieee.org/ieee/802.1Q/)
- [Linux Kernel — Bonding Driver Documentation](https://www.kernel.org/doc/html/latest/networking/bonding.html)
- [Linux Kernel — VLAN Documentation](https://www.kernel.org/doc/html/latest/networking/vlan.html)
- [Linux Kernel — Bridge Documentation](https://www.kernel.org/doc/html/latest/networking/bridge.html)
- [man ip-link](https://man7.org/linux/man-pages/man8/ip-link.8.html)
- [Open vSwitch Documentation](https://docs.openvswitch.org/en/latest/)
- [Cisco Catalyst Switch Configuration Guides](https://www.cisco.com/c/en/us/support/switches/catalyst-9300-series-switches/products-installation-and-configuration-guides-list.html)
- [Juniper Ethernet Switching Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/l2-switching/topics/topic-map/ethernet-switching-overview.html)
