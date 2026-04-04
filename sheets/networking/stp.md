# STP (Spanning Tree Protocol)

Layer 2 loop prevention protocol that builds a loop-free logical topology from a physical mesh by electing a root bridge, calculating shortest paths, and blocking redundant links while maintaining failover capability.

## STP Variants

```
Protocol   Standard      Convergence    Notes
──────────────────────────────────────────────────────────────────
STP        802.1D-1998   30-50 sec      Original, slow convergence
RSTP       802.1D-2004   1-6 sec        Rapid convergence, replaces STP
MSTP       802.1Q-2005   1-6 sec        Multiple spanning trees per VLAN group
PVST+      Cisco         30-50 sec      Per-VLAN STP (Cisco proprietary)
Rapid PVST+ Cisco        1-6 sec        Per-VLAN RSTP (Cisco proprietary)
```

## Root Bridge Election

```
# Every bridge has a Bridge ID:
#   Bridge ID = Priority (4 bits) + VLAN ID (12 bits) + MAC Address (48 bits)
#   Total: 8 bytes
#   Priority default: 32768 (0x8000)
#   Priority range: 0-61440 in multiples of 4096

# The bridge with the LOWEST Bridge ID becomes root
# Tie-breaker: lower MAC address wins

# Election process:
# 1. All bridges start claiming to be root
# 2. Each bridge sends BPDUs with its Bridge ID as root
# 3. On receiving a superior BPDU (lower root ID), bridge updates
# 4. Eventually all bridges agree on the same root
```

```bash
# Cisco — set bridge priority to become root
spanning-tree vlan 1 priority 4096          # lower = more likely root
spanning-tree vlan 1 root primary           # auto-sets priority 24576 or lower
spanning-tree vlan 1 root secondary         # auto-sets priority 28672

# Linux bridge — view/set STP
brctl showstp br0                           # show STP state
brctl setbridgeprio br0 4096               # set bridge priority
bridge link show                            # iproute2 bridge info

# Check current root bridge
show spanning-tree                          # Cisco
brctl showstp br0 | grep "root id"         # Linux
```

## Port Roles

```
Role          Description
──────────────────────────────────────────────────────────────────
Root Port     Best path to root bridge (one per non-root bridge)
Designated    Best port on a segment to reach root (one per segment)
Blocking      Redundant port, does not forward (STP)
Alternate     Backup root port (RSTP — provides fast failover)
Backup        Backup designated port (RSTP — rare, shared media)
```

### Port Role Selection

```
# Root Port selection (on each non-root bridge):
# 1. Lowest root path cost
# 2. Lowest sender Bridge ID
# 3. Lowest sender port priority
# 4. Lowest sender port number

# Designated Port selection (on each segment):
# 1. Bridge on segment with lowest root path cost
# 2. Lowest Bridge ID as tiebreaker

# All other ports become Blocking/Alternate
```

## Port States

### STP (802.1D) States

```
State        Forwards Data    Learns MACs    Duration
──────────────────────────────────────────────────────
Disabled     No               No             —
Blocking     No               No             20s (max age)
Listening    No               No             15s (forward delay)
Learning     No               Yes            15s (forward delay)
Forwarding   Yes              Yes            —

# Blocking → Listening → Learning → Forwarding
# Total convergence: 20 + 15 + 15 = 50 seconds (worst case)
```

### RSTP (802.1w) States

```
State        Forwards    Learns    STP Equivalent
──────────────────────────────────────────────────
Discarding   No          No        Disabled/Blocking/Listening
Learning     No          Yes       Learning
Forwarding   Yes         Yes       Forwarding

# RSTP achieves rapid convergence through:
# - Proposal/Agreement mechanism (sync between neighbors)
# - Edge ports (no BPDU expected → immediate forwarding)
# - Alternate ports (pre-computed backup root port)
```

## Path Cost

```
Bandwidth        STP Cost (802.1D)    RSTP Cost (802.1w)
─────────────────────────────────────────────────────────
10 Mbps          100                  2,000,000
100 Mbps         19                   200,000
1 Gbps           4                    20,000
10 Gbps          2                    2,000
40 Gbps          —                    500
100 Gbps         —                    200
```

```bash
# Cisco — set port cost
interface GigabitEthernet0/1
  spanning-tree cost 10                     # override auto cost
  spanning-tree port-priority 64            # port priority (0-240, step 16)

# Linux bridge
brctl setpathcost br0 eth0 4               # set port cost
brctl setportprio br0 eth0 64              # set port priority
# Or with iproute2
ip link set eth0 type bridge_slave cost 4
```

## BPDU (Bridge Protocol Data Unit)

```
# BPDU fields:
#   Protocol ID:        0x0000
#   Version:            0 (STP), 2 (RSTP), 3 (MSTP)
#   Type:               0x00 (Config), 0x80 (TCN), 0x02 (RSTP)
#   Flags:              TC, TCA, proposal, agreement, port role, etc.
#   Root Bridge ID:     8 bytes (priority + MAC)
#   Root Path Cost:     4 bytes
#   Sender Bridge ID:   8 bytes
#   Sender Port ID:     2 bytes (priority + port number)
#   Message Age:        2 bytes
#   Max Age:            2 bytes (default 20 seconds)
#   Hello Time:         2 bytes (default 2 seconds)
#   Forward Delay:      2 bytes (default 15 seconds)

# BPDUs are sent to destination MAC 01:80:C2:00:00:00
# BPDUs are NOT forwarded by switches — processed locally
```

## PortFast and BPDU Guard

```bash
# PortFast — skip Listening/Learning, go straight to Forwarding
# ONLY for edge ports (connected to end hosts, not switches)

# Cisco — per interface
interface GigabitEthernet0/1
  spanning-tree portfast
  spanning-tree bpduguard enable

# Cisco — globally (all access ports)
spanning-tree portfast default
spanning-tree portfast bpduguard default

# BPDU Guard — shut down port if BPDU is received
# Prevents loops when someone plugs a switch into an edge port
# Port goes to err-disabled state

# Recover from err-disabled
errdisable recovery cause bpduguard
errdisable recovery interval 300            # re-enable after 300s

# BPDU Filter — stop sending/receiving BPDUs (DANGEROUS)
# Effectively disables STP on the port — can cause loops
spanning-tree bpdufilter enable             # per-port only, never global
```

## Root Guard and Loop Guard

```bash
# Root Guard — prevent a port from becoming root port
# Use on ports where you KNOW the root should NOT be
interface GigabitEthernet0/1
  spanning-tree guard root

# If a superior BPDU arrives, port goes to "root-inconsistent" state
# Port recovers automatically when superior BPDUs stop

# Loop Guard — prevent alternate/root ports from transitioning
# to designated/forwarding if BPDUs stop (unidirectional link failure)
interface GigabitEthernet0/1
  spanning-tree guard loop

# Global loop guard
spanning-tree loopguard default
```

## MSTP (Multiple Spanning Tree Protocol)

```bash
# MSTP maps VLANs to instances — fewer STP calculations than per-VLAN STP
# All switches in a region must agree on: name, revision, VLAN-to-instance mapping

# Cisco MSTP config
spanning-tree mode mst

spanning-tree mst configuration
  name CAMPUS
  revision 1
  instance 1 vlan 1-100
  instance 2 vlan 101-200
  instance 3 vlan 201-4094

# Set root for an instance
spanning-tree mst 1 root primary
spanning-tree mst 2 root primary

# Instance 0 is the IST (Internal Spanning Tree) — carries all unmapped VLANs
```

## Monitoring

```bash
# Cisco
show spanning-tree
show spanning-tree summary
show spanning-tree vlan 10
show spanning-tree interface Gi0/1
show spanning-tree root                     # show root for all VLANs
show spanning-tree blockedports
show spanning-tree inconsistentports

# Linux bridge
brctl showstp br0
bridge -d link show
cat /sys/class/net/br0/bridge/stp_state     # 0=disabled, 1=enabled

# Capture BPDUs
tcpdump -i eth0 -n ether dst 01:80:c2:00:00:00
```

## Tips

- Always plan which switch should be root bridge. If you do not set priorities, the switch with the lowest MAC address wins, which is usually the oldest switch in the network — the one most likely to fail.
- PortFast should be enabled on EVERY access port connected to end hosts. Without it, PCs wait 30-50 seconds for the port to reach forwarding state, causing DHCP timeouts, PXE boot failures, and user complaints.
- BPDU Guard and PortFast are a pair. Never enable PortFast without BPDU Guard. If someone plugs a switch into a PortFast port without BPDU Guard, you get a loop with no STP protection.
- Root Guard protects your root bridge election. Place it on all ports where the root should never be learned — typically on distribution-to-access links. The access layer should never dictate the root.
- STP convergence of 30-50 seconds is caused by the forward delay timer (15s listening + 15s learning). RSTP eliminates this with the proposal/agreement mechanism and converges in 1-6 seconds for most topologies.
- A unidirectional link failure (fiber TX works, RX broken) can cause a loop because the blocking port stops receiving BPDUs and transitions to forwarding. Loop Guard detects this by keeping the port in an inconsistent state.
- MSTP reduces CPU overhead by mapping multiple VLANs to a single spanning tree instance. With 200 VLANs, PVST+ runs 200 separate STP instances. MSTP might use 3-4 instances for the same topology.
- The maximum network diameter for STP is 7 hops (switches) from the root. Beyond this, Message Age exceeds Max Age (20 seconds) and BPDUs are discarded, breaking STP. RSTP relaxes this with hop-count-based aging.
- TCN (Topology Change Notification) BPDUs cause MAC address table flushing on all switches. Frequent topology changes (flapping links, DHCP clients) cause excessive flooding. Use `spanning-tree portfast` on access ports to suppress TCN.
- STP does not provide load balancing. In a dual-uplink topology, one link is always blocked. Use LACP (link aggregation) for per-flow load sharing, or MSTP/PVST+ with different root bridges per VLAN group for basic traffic engineering.

## See Also

- ethernet, vlan, lacp, tcpdump

## References

- [IEEE 802.1D-2004 — Rapid Spanning Tree Protocol](https://standards.ieee.org/standard/802_1D-2004.html)
- [IEEE 802.1Q-2022 — MSTP and VLANs](https://standards.ieee.org/standard/802_1Q-2022.html)
- [Cisco — Understanding STP](https://www.cisco.com/c/en/us/support/docs/lan-switching/spanning-tree-protocol/5234-5.html)
- [RFC 7727 — Spanning Tree Protocol (STP) Application of YANG](https://www.rfc-editor.org/rfc/rfc7727)
- [man brctl](https://linux.die.net/man/8/brctl)
