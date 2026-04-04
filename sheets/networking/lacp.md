# LACP (Link Aggregation Control Protocol)

IEEE 802.3ad/802.1AX protocol that bundles multiple physical links into a single logical channel (LAG/bond), providing increased bandwidth and redundancy through negotiated aggregation with hash-based traffic distribution.

## Link Aggregation Concepts

```
# Physical topology:
# Switch A ═══ eth0 ═══ Switch B
#          ═══ eth1 ═══
#          ═══ eth2 ═══
#          ═══ eth3 ═══
#
# Logical topology:
# Switch A ═══ bond0 (4x1G = 4G aggregate) ═══ Switch B
#
# Key points:
# - All member links must be same speed and duplex
# - All member links must terminate on the same two devices
# - Traffic is distributed across links using a hash algorithm
# - Individual flows do NOT exceed single link speed
# - A single TCP connection uses ONE link (hash-pinned)
```

## LACP vs Static LAG

```
Feature         LACP (802.3ad)           Static (mode on)
──────────────────────────────────────────────────────────────────
Negotiation     Yes (LACPDU exchange)    None
Misconfiguration Protected               Not protected
Hot standby     Yes (>8 ports)           No
Link monitoring Yes (LACPDU timeout)     No (relies on physical)
Partner info    Yes (system ID, key)     No
Standard        IEEE 802.1AX             Vendor-specific
```

## LACPDU (LACP Data Unit)

```
# LACPDUs are sent to multicast MAC: 01:80:C2:00:00:02
# Ethertype: 0x8809 (Slow Protocols)
# Sent every 1 second (fast) or 30 seconds (slow)

# LACPDU fields:
Actor Information:
  System Priority:     16-bit (default 32768)
  System ID:           48-bit MAC address
  Key:                 16-bit (groups ports that can aggregate)
  Port Priority:       16-bit (default 32768)
  Port Number:         16-bit
  State:               8-bit flags (Activity, Timeout, Aggregation,
                       Synchronization, Collecting, Distributing,
                       Defaulted, Expired)

Partner Information:
  (Same fields as Actor, reflecting what we learned from partner)
```

## Linux Bonding Configuration

### Using ip/iproute2

```bash
# Load bonding module
modprobe bonding

# Create bond interface
ip link add bond0 type bond mode 802.3ad

# Set LACP parameters
ip link set bond0 type bond miimon 100          # link check interval (ms)
ip link set bond0 type bond lacp_rate fast       # LACPDU every 1s (vs 30s)
ip link set bond0 type bond xmit_hash_policy layer3+4  # hash policy

# Add member interfaces
ip link set eth0 down
ip link set eth1 down
ip link set eth0 master bond0
ip link set eth1 master bond0

# Bring everything up
ip link set bond0 up
ip link set eth0 up
ip link set eth1 up

# Assign IP
ip addr add 192.168.1.10/24 dev bond0
```

### Using netplan (Ubuntu)

```yaml
# /etc/netplan/01-bond.yaml
network:
  version: 2
  ethernets:
    eth0: {}
    eth1: {}
  bonds:
    bond0:
      interfaces:
        - eth0
        - eth1
      parameters:
        mode: 802.3ad
        lacp-rate: fast
        mii-monitor-interval: 100
        transmit-hash-policy: layer3+4
      addresses:
        - 192.168.1.10/24
      routes:
        - to: default
          via: 192.168.1.1
```

### Using nmcli (NetworkManager)

```bash
# Create bond
nmcli connection add type bond con-name bond0 ifname bond0 \
    bond.options "mode=802.3ad,miimon=100,lacp_rate=fast,xmit_hash_policy=layer3+4"

# Add members
nmcli connection add type ethernet con-name bond-eth0 ifname eth0 master bond0
nmcli connection add type ethernet con-name bond-eth1 ifname eth1 master bond0

# Set IP
nmcli connection modify bond0 ipv4.addresses 192.168.1.10/24
nmcli connection modify bond0 ipv4.method manual

# Activate
nmcli connection up bond0
```

### Using /etc/sysconfig (RHEL/CentOS)

```bash
# /etc/sysconfig/network-scripts/ifcfg-bond0
DEVICE=bond0
TYPE=Bond
BONDING_OPTS="mode=802.3ad miimon=100 lacp_rate=fast xmit_hash_policy=layer3+4"
IPADDR=192.168.1.10
NETMASK=255.255.255.0
ONBOOT=yes

# /etc/sysconfig/network-scripts/ifcfg-eth0
DEVICE=eth0
MASTER=bond0
SLAVE=yes
ONBOOT=yes

# /etc/sysconfig/network-scripts/ifcfg-eth1
DEVICE=eth1
MASTER=bond0
SLAVE=yes
ONBOOT=yes
```

## Bond Modes

```
Mode  Name              LACP?  Load Balance   Fault Tolerance
──────────────────────────────────────────────────────────────────
0     balance-rr        No     Round-robin    Yes (but out-of-order)
1     active-backup     No     None (1 active) Yes
2     balance-xor       No     Hash-based     Yes
3     broadcast         No     None (all)     Yes
4     802.3ad (LACP)    Yes    Hash-based     Yes
5     balance-tlb       No     Adaptive TX    Yes
6     balance-alb       No     Adaptive TX+RX Yes
```

## Hash Policies

```bash
# xmit_hash_policy options (determines which link carries each flow)

layer2              # Hash: src MAC XOR dst MAC
                    # Simple but poor distribution with few MACs (e.g., router)

layer3+4            # Hash: src IP XOR dst IP XOR src port XOR dst port
                    # Best distribution for most workloads

layer2+3            # Hash: src MAC XOR dst MAC XOR src IP XOR dst IP
                    # Good balance for mixed L2/L3 traffic

encap2+3            # Hash on inner headers for tunneled traffic (VXLAN, GRE)
                    # Use when outer headers are all the same (tunnel endpoints)
encap3+4            # Hash on inner L3+L4 for tunneled traffic

# Check current hash policy
cat /proc/net/bonding/bond0 | grep "Transmit Hash"

# Change hash policy
ip link set bond0 type bond xmit_hash_policy layer3+4
```

## Switch-Side Configuration

```bash
# Cisco IOS
interface range GigabitEthernet0/1-4
  channel-group 1 mode active              # LACP active
  # mode active  = initiate LACP
  # mode passive = respond to LACP only
  # mode on      = static (no LACP)

interface Port-channel1
  switchport mode trunk
  switchport trunk allowed vlan 10,20,30

# Verify
show etherchannel summary
show etherchannel port-channel
show lacp neighbor
show lacp counters

# Arista EOS
interface Ethernet1-4
  channel-group 1 mode active

interface Port-Channel1
  switchport mode trunk

# Juniper JunOS
set interfaces ae0 aggregated-ether-options lacp active
set interfaces ge-0/0/0 ether-options 802.3ad ae0
set interfaces ge-0/0/1 ether-options 802.3ad ae0
```

## Monitoring

```bash
# Linux — bond status
cat /proc/net/bonding/bond0

# Key fields to check:
# Bonding Mode: IEEE 802.3ad Dynamic link aggregation
# MII Status: up
# LACP rate: fast
# Partner Mac Address: (should show switch MAC)
# Aggregator ID: (both ports should have same ID)

# Per-slave info
cat /sys/class/net/bond0/bonding/slaves         # list members
cat /sys/class/net/bond0/bonding/active_slave    # active slave (mode 1)
cat /sys/class/net/bond0/bonding/ad_partner_key  # partner LACP key

# Traffic distribution check
# Watch TX/RX counters per slave
watch -n 1 'cat /proc/net/bonding/bond0 | grep -A5 "Slave Interface"'

# Check LACP counters
ip -s link show bond0
ip -s link show eth0
ip -s link show eth1

# tcpdump for LACPDUs
tcpdump -i eth0 -nn ether proto 0x8809
```

## Tips

- LACP mode `active` on both ends is recommended. If both sides are `passive`, no LACPDUs are sent and the LAG never forms. At least one side must be active.
- The `layer3+4` hash policy gives the best distribution for general-purpose traffic because it includes source/destination ports, creating unique hashes for each TCP/UDP flow. Use `layer2` only if all traffic goes through a single router (same MAC pair).
- A single TCP connection can never exceed the speed of one member link. A 4x1G bond gives 4 Gbps aggregate but each flow maxes at 1 Gbps. Aggregation benefits workloads with many parallel flows.
- LACP fast rate (1-second LACPDUs) detects link failures in 3 seconds (3 missed LACPDUs). Slow rate (30-second) takes 90 seconds. Always use fast rate in production.
- If one side runs LACP and the other runs static (`mode on`), the LAG may appear to work but LACP cannot detect misconfigurations. Both sides should use LACP for safety.
- Hash distribution is inherently uneven with few flows. With only 2 active flows on a 4-link bundle, both flows may hash to the same link (25% chance of worst case). This is normal, not a bug.
- The `miimon` parameter (default 100ms) checks link state via MII. For faster detection, use `arp_interval` with `arp_ip_target` to verify end-to-end reachability, catching upstream failures that MII cannot detect.
- All member links must have the same LACP Key (same speed, duplex, and switch port config). If keys mismatch, LACP will not aggregate the links. Check `show lacp neighbor` to verify.
- On Linux, bonding mode 5 (`balance-tlb`) and 6 (`balance-alb`) do not require switch support. They adaptively distribute traffic using MAC address learning. Good for environments where switch LAG configuration is not possible.
- When using bonds with VLANs, create the VLAN interface on top of the bond, not on individual member interfaces: `ip link add link bond0 name bond0.100 type vlan id 100`.

## See Also

- ethernet, stp, vlan, ip

## References

- [IEEE 802.1AX — Link Aggregation](https://standards.ieee.org/standard/802_1AX-2020.html)
- [Linux Kernel — Bonding Documentation](https://www.kernel.org/doc/html/latest/networking/bonding.html)
- [Cisco — EtherChannel Configuration](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst9300/software/release/16-12/configuration_guide/lag/b_1612_lag_9300_cg.html)
- [Red Hat — Network Bonding](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_and_managing_networking/configuring-network-bonding_configuring-and-managing-networking)
