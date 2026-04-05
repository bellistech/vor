# SPAN / ERSPAN (Port Mirroring and Remote Traffic Analysis)

Copy traffic from one or more source ports or VLANs to a destination port or remote IP for passive monitoring, enabling IDS/IPS inspection, forensic capture, lawful intercept, and network troubleshooting without inline disruption.

## SPAN Variants Overview

```
Variant     Transport         Encap       Scope            Standard
────────────────────────────────────────────────────────────────────────
Local SPAN  Same switch       None        Single chassis   Proprietary
RSPAN       RSPAN VLAN        802.1Q      L2 domain        Proprietary
ERSPAN      IP (GRE)          GRE         Any IP reach     RFC (draft)
```

```
Source Ports/VLANs ──┐
                     ├──► SPAN Session ──► Destination (port / RSPAN VLAN / GRE tunnel)
Filter (ACL/VLAN) ──┘

Local SPAN:   src port ──► dst port (same switch)
RSPAN:        src port ──► RSPAN VLAN ──► [trunk] ──► dst port (remote switch)
ERSPAN:       src port ──► GRE encap ──► IP network ──► GRE decap ──► dst port
```

## Local SPAN

### Cisco IOS — Basic Configuration

```bash
# Mirror Gi0/1 (both directions) to Gi0/24
monitor session 1 source interface GigabitEthernet0/1
monitor session 1 destination interface GigabitEthernet0/24

# Mirror only ingress traffic from Gi0/1
monitor session 1 source interface GigabitEthernet0/1 rx

# Mirror only egress traffic from Gi0/1
monitor session 1 source interface GigabitEthernet0/1 tx

# Mirror multiple source ports
monitor session 1 source interface GigabitEthernet0/1 - 4

# Mirror an entire VLAN (all ports in VLAN 10)
monitor session 1 source vlan 10

# Filter SPAN to only capture specific VLANs on a trunk source
monitor session 1 filter vlan 10 , 20 , 30

# Allow destination port to also send/receive traffic (rarely recommended)
monitor session 1 destination interface GigabitEthernet0/24 ingress vlan 10

# Remove a SPAN session
no monitor session 1
```

### Cisco IOS — Verification

```bash
# Show all SPAN sessions
show monitor session all

# Show specific session
show monitor session 1

# Show session detail (includes ACL filters)
show monitor session 1 detail

# Show platform SPAN resource usage
show platform software monitor session

# Example output:
# Session 1
# ---------
# Type                   : Local Session
# Source Ports            :
#     Both               : Gi0/1
# Destination Ports      : Gi0/24
#     Encapsulation      : Native
#     Ingress            : Disabled
# Filter VLANs           : 10,20,30
```

### Cisco NX-OS

```bash
# NX-OS uses the same monitor session syntax with minor differences
monitor session 1
  source interface ethernet 1/1 both
  destination interface ethernet 1/48
  no shut

# VLAN source
monitor session 2
  source vlan 100
  destination interface ethernet 1/48
  no shut

# Direction filtering on NX-OS
monitor session 3
  source interface ethernet 1/1 rx
  destination interface ethernet 1/48
  no shut

# ACL-based filtering (NX-OS)
monitor session 1
  source interface ethernet 1/1
  destination interface ethernet 1/48
  filter access-group SPAN-ACL
  no shut

# Verify
show monitor session all
show monitor session 1 detail
```

### Arista EOS

```bash
# Arista SPAN session
monitor session 1 source ethernet 1 both
monitor session 1 destination ethernet 48

# With truncation (capture headers only, reduce load)
monitor session 1 truncation size 128

# With CPU source (capture control plane traffic)
monitor session 1 source cpu rx

# Verify
show monitor session 1
```

## RSPAN (Remote SPAN)

```
┌──────────────────┐         Trunk (carries RSPAN VLAN)        ┌──────────────────┐
│  Source Switch    │ ─────────────────────────────────────────► │  Dest Switch     │
│                  │                                            │                  │
│  Gi0/1 (source)  │                                            │  Gi0/24 (to IDS) │
│  Session 1 ──►   │                                            │  ◄── Session 2   │
│  RSPAN VLAN 999  │                                            │  RSPAN VLAN 999  │
└──────────────────┘                                            └──────────────────┘
```

### RSPAN VLAN Creation (All Switches in Path)

```bash
# Create the RSPAN VLAN on every switch in the L2 path
vlan 999
  name RSPAN_VLAN
  remote-span

# RSPAN VLAN must be allowed on all trunks between source and destination switch
interface GigabitEthernet0/49
  switchport trunk allowed vlan add 999
```

### Source Switch Configuration

```bash
# Source switch: mirror traffic into RSPAN VLAN
monitor session 1 source interface GigabitEthernet0/1
monitor session 1 destination remote vlan 999

# With reflector port (some platforms require this)
# The reflector port is looped back and carries RSPAN traffic
monitor session 1 destination remote vlan 999 reflector-port GigabitEthernet0/47
```

### Destination Switch Configuration

```bash
# Destination switch: receive from RSPAN VLAN, send to local port
monitor session 2 source remote vlan 999
monitor session 2 destination interface GigabitEthernet0/24
```

### RSPAN Verification

```bash
show monitor session 1
show monitor session 2
show vlan id 999                  # verify remote-span attribute
show interfaces trunk             # verify RSPAN VLAN allowed on trunks
```

## ERSPAN (Encapsulated Remote SPAN)

### ERSPAN Header Formats

```
ERSPAN Type I (Original):
  No ERSPAN header — just GRE + mirrored frame
  GRE Protocol Type: 0x88BE
  Used on older Catalyst 6500

ERSPAN Type II (RFC draft):
  ┌─────────────────────────────────────────────────────────────────┐
  │ Outer IP Header │ GRE Header │ ERSPAN Header │ Mirrored Frame  │
  └─────────────────────────────────────────────────────────────────┘

  GRE Header (8 bytes):
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |0|0|0|1|0|00000|000000000|00000|    Protocol Type (0x88BE)    |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |                  Sequence Number                              |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

  ERSPAN Type II Header (8 bytes):
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  | Ver |    VLAN     |  COS  |En|T|        Session ID            |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |       Reserved          |             Index                   |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

  Ver:        0001 (Type II)
  VLAN:       Original VLAN of mirrored frame
  COS:        Class of Service of original frame
  En:         Encapsulation type (00=native, 01=ISL, 10=802.1Q)
  T:          Trunk bit (1 = source was trunk port)
  Session ID: 10 bits (0-1023), identifies the SPAN session
  Index:      20 bits, platform-specific port/module index

ERSPAN Type III (Timestamp support):
  Adds 12-byte platform-specific sub-header
  Includes 32-bit hardware timestamp (100 ns granularity)
  Supports optional marker packets for session keepalive
  D bit: direction (0=ingress, 1=egress)
  GRA field: timestamp granularity (00=100us, 01=100ns, 10=IEEE 1588)
  Protocol Type: 0x22EB
```

### Cisco IOS ERSPAN Source

```bash
# ERSPAN source session (traffic originates here)
monitor session 1 type erspan-source
  source interface GigabitEthernet0/1 both
  destination
    erspan-id 100
    ip address 10.0.0.100          # remote collector IP
    origin ip address 10.0.0.1     # this switch's source IP for GRE
  no shut

# With VLAN filter
monitor session 2 type erspan-source
  source interface GigabitEthernet0/1 - 4
  filter vlan 10 , 20
  destination
    erspan-id 200
    ip address 10.0.0.100
    origin ip address 10.0.0.1
  no shut
```

### Cisco IOS ERSPAN Destination

```bash
# ERSPAN destination session (traffic terminates here)
monitor session 3 type erspan-destination
  destination interface GigabitEthernet0/24
  source
    erspan-id 100
    ip address 10.0.0.100          # local IP matching source config
  no shut
```

### Cisco NX-OS ERSPAN

```bash
# NX-OS ERSPAN source
monitor session 1 type erspan-source
  source interface ethernet 1/1
  destination ip 10.0.0.100
  erspan-id 100
  vrf default
  ip ttl 64
  ip dscp 0
  origin ip 10.0.0.1
  no shut

# NX-OS ERSPAN destination
monitor session 2 type erspan-destination
  source ip 10.0.0.100
  destination interface ethernet 1/48
  erspan-id 100
  no shut
```

## Linux Traffic Mirroring (tc mirror)

### Mirror with tc (Traffic Control)

```bash
# Mirror all ingress traffic on eth0 to eth1
tc qdisc add dev eth0 ingress
tc filter add dev eth0 parent ffff: \
  protocol all u32 match u32 0 0 \
  action mirred egress mirror dev eth1

# Mirror all egress traffic on eth0 to eth1
tc qdisc add dev eth0 handle 1: root prio
tc filter add dev eth0 parent 1: \
  protocol all u32 match u32 0 0 \
  action mirred egress mirror dev eth1

# Mirror only TCP port 80 traffic
tc qdisc add dev eth0 ingress
tc filter add dev eth0 parent ffff: \
  protocol ip u32 \
  match ip protocol 6 0xff \
  match ip dport 80 0xffff \
  action mirred egress mirror dev eth1

# Mirror with ERSPAN encapsulation (Linux 4.18+)
# Create ERSPAN tunnel
ip link add erspan1 type erspan \
  local 10.0.0.1 remote 10.0.0.100 \
  seq key 100 erspan_ver 2 erspan_dir ingress erspan_hwid 0x4
ip link set erspan1 up

# Mirror traffic into ERSPAN tunnel
tc qdisc add dev eth0 ingress
tc filter add dev eth0 parent ffff: \
  protocol all u32 match u32 0 0 \
  action mirred egress mirror dev erspan1

# Remove all mirroring rules
tc qdisc del dev eth0 ingress
tc qdisc del dev eth0 root
```

### Linux ERSPAN Tunnel (Receive Side)

```bash
# Create ERSPAN receive tunnel
ip link add erspan1 type erspan \
  local 10.0.0.100 remote 10.0.0.1 \
  seq key 100 erspan_ver 2 erspan_dir ingress erspan_hwid 0x4
ip link set erspan1 up

# Capture on ERSPAN interface
tcpdump -i erspan1 -w /tmp/captured.pcap

# ERSPAN Type I tunnel (no ERSPAN header)
ip link add erspan_t1 type erspan \
  local 10.0.0.100 remote 10.0.0.1 \
  seq key 100 erspan_ver 1

# ERSPAN Type III tunnel (with timestamps)
ip link add erspan3 type erspan \
  local 10.0.0.100 remote 10.0.0.1 \
  seq key 100 erspan_ver 2
```

### Open vSwitch (OVS) Mirroring

```bash
# Create a mirror on OVS bridge br0
ovs-vsctl -- set Bridge br0 mirrors=@m \
  -- --id=@src get Port eth0 \
  -- --id=@dst get Port eth1 \
  -- --id=@m create Mirror name=span1 \
     select-src-port=@src select-dst-port=@src output-port=@dst

# Mirror all traffic on bridge to a GRE tunnel (ERSPAN-like)
ovs-vsctl add-port br0 gre1 \
  -- set interface gre1 type=gre options:remote_ip=10.0.0.100
ovs-vsctl -- set Bridge br0 mirrors=@m \
  -- --id=@src get Port eth0 \
  -- --id=@m create Mirror name=erspan1 \
     select-src-port=@src output-port=@gre1

# Mirror a specific VLAN
ovs-vsctl -- set Bridge br0 mirrors=@m \
  -- --id=@dst get Port eth1 \
  -- --id=@m create Mirror name=vlan_mirror \
     select-vlan=100 output-port=@dst

# Remove mirror
ovs-vsctl clear Bridge br0 mirrors

# Show mirrors
ovs-vsctl list Mirror
```

## Session Limits and Resource Constraints

```
Platform               Max SPAN Sessions    ERSPAN Sessions    Notes
──────────────────────────────────────────────────────────────────────────────
Catalyst 9300          8 total              4 (of the 8)       Shared pool
Catalyst 9500          8 total              4 (of the 8)       Shared pool
Catalyst 3850          4 total              Not supported      Local + RSPAN only
Nexus 9000             32 total             32                 Per-VDC limits
Nexus 7000             18 per VDC           18                 Per-VDC
Nexus 5000             4 total              2                  Limited ASIC
Catalyst 6500          64 total             Varies by sup      Per-supervisor
Arista 7050            4 total              4                  Shared
Juniper QFX5100        4 total              Not native         Use analyser
Linux (tc)             Unlimited*           Unlimited*         *Limited by CPU
OVS                    Per-bridge           Via GRE            Software
```

## ACL-Based SPAN Filtering

```bash
# Create ACL to match specific traffic
ip access-list extended SPAN-FILTER
  permit tcp any any eq 443
  permit tcp any any eq 80
  permit udp any any eq 53
  deny ip any any

# Apply ACL filter to SPAN session (Cisco IOS)
monitor session 1 source interface GigabitEthernet0/1
monitor session 1 destination interface GigabitEthernet0/24
monitor session 1 filter access-group SPAN-FILTER

# NX-OS: filter within session config
monitor session 1
  source interface ethernet 1/1
  destination interface ethernet 1/48
  filter access-group SPAN-FILTER
  no shut
```

## TAP vs SPAN Comparison

```
Feature            Network TAP                    SPAN / Mirror Port
──────────────────────────────────────────────────────────────────────────────
Inline?            Yes (sits in cable path)        No (copy from switch ASIC)
Packet loss        Zero (hardware copy)            Possible under oversubscription
Latency added      None (passive optical/copper)   None to source; mirror may lag
Full duplex        Separate TX/RX streams          Merged into one stream
Failure mode       Fail-open (traffic passes)      No effect (mirror just stops)
Visibility         Sees everything incl errors      May drop errored frames
Cost               $500-5000 per unit               Free (built into switch)
Deployment         Physical install required        Software config only
Scalability        One per link                     Limited by session count
Timestamping       Some TAPs add HW timestamps     ERSPAN Type III only
VLAN tags          Preserves all tags               May strip tags on access ports
Jumbo frames       Passes as-is                     Depends on dest port MTU
Oversubscription   N/A (1:1 copy)                   Dst port < sum of src ports
Best for           Critical links, compliance       Flexible, ad-hoc monitoring
```

## Packet Broker Integration

```
                    ┌─────────────────────────────────┐
 SPAN/TAP feeds ──► │        Packet Broker             │ ──► IDS/IPS
                    │  (Gigamon, Keysight, cPacket)    │ ──► Forensics
                    │                                  │ ──► APM
                    │  Functions:                      │ ──► DLP
                    │  - Aggregation (many-to-one)     │ ──► NetFlow/sFlow
                    │  - Filtering (by L2-L4 fields)   │
                    │  - Load balancing (1:N)          │
                    │  - Deduplication                 │
                    │  - Packet slicing / truncation   │
                    │  - SSL/TLS decryption            │
                    │  - Timestamping (ns precision)   │
                    │  - Header stripping (ERSPAN/GRE) │
                    │  - Tunnel termination            │
                    └─────────────────────────────────┘
```

## Troubleshooting

```bash
# SPAN destination port shows no traffic
# 1. Verify session is active
show monitor session 1
#    Status should be "Up" not "Admin Down"

# 2. Check source port is up and passing traffic
show interface GigabitEthernet0/1 | include packets
show interface GigabitEthernet0/1 | include line protocol

# 3. Verify destination port is not in a VLAN or STP blocking
show spanning-tree interface GigabitEthernet0/24
# SPAN destination port is removed from STP automatically

# 4. Check for oversubscription
# If source is 10G and destination is 1G, packets will be dropped
show interface GigabitEthernet0/24 | include output drops

# 5. On Linux, verify tc rules are active
tc filter show dev eth0 parent ffff:
tc -s filter show dev eth0 parent ffff:     # with statistics

# 6. Verify ERSPAN tunnel is up
show interface tunnel1
ping 10.0.0.100 source 10.0.0.1             # verify GRE reachability

# 7. RSPAN: verify RSPAN VLAN on all intermediate trunks
show vlan id 999
show interfaces trunk | include 999

# 8. Check platform-specific limits
show platform software monitor session
show hardware capacity monitor               # NX-OS
```

## Performance Impact

```
Scenario                          Impact on Source Port    Impact on Switch
──────────────────────────────────────────────────────────────────────────────
Local SPAN (ASIC-based)           None                    Minimal (ASIC copy)
Local SPAN (CPU-based, old HW)    None                    5-15% CPU increase
RSPAN                             None                    Trunk bandwidth used
ERSPAN                            None                    CPU for GRE encap
Multiple sessions (same source)   None                    Multiplied ASIC load
VLAN-based SPAN (large VLAN)      None                    High replication load
Linux tc mirror                   None on source          CPU proportional to pps
OVS mirror                        None on source          CPU proportional to pps

# Key rules:
# - SPAN never drops or delays source traffic (best-effort copy)
# - Destination port bandwidth must match or exceed source
# - ERSPAN adds ~50 bytes overhead per packet (IP+GRE+ERSPAN headers)
# - ERSPAN uses switch CPU for encapsulation (not all packets may be mirrored)
# - On high-throughput links, use TAPs instead of SPAN for guaranteed capture
```

## Tips

- SPAN destination ports drop all normal traffic. Any device previously connected to a SPAN destination port loses connectivity. Always use a dedicated, unused port and label it clearly to prevent accidental use.
- When mirroring a trunk port, the destination receives tagged frames by default. If your analyzer does not support 802.1Q tags, use `encapsulation replicate` on the destination to strip tags, or configure VLAN filtering to mirror only the VLAN of interest.
- ERSPAN adds approximately 50 bytes of overhead per packet (20-byte IP header + 8-byte GRE header + 8-byte ERSPAN header + optional). Ensure the path MTU between source and destination supports the larger frames to avoid fragmentation, which severely degrades performance.
- On oversubscribed SPAN sessions where the aggregate source bandwidth exceeds the destination port speed, the switch silently drops mirrored packets. There is no alert or log. Always check destination port output drop counters.
- Linux `tc` mirroring happens in software on the CPU. At high packet rates (above 1 Mpps), expect significant CPU load. For sustained high-throughput mirroring on Linux, use hardware offload with ERSPAN-capable NICs (Mellanox ConnectX, Intel E810) or dedicated TAPs.
- ERSPAN Type III includes nanosecond-granularity hardware timestamps and a direction bit. If your capture analysis requires precise timing or differentiating ingress from egress, always prefer Type III over Type II where hardware supports it.
- Do not mirror traffic to a port connected to a regular network device. The destination port transmits mirrored frames, and any connected host will see alien traffic. This can cause IP conflicts, MAC table pollution, and security exposure.
- RSPAN traffic traverses the production network over trunks. On congested trunks, RSPAN frames are subject to the same QoS policies and may be dropped during congestion. Use ERSPAN with a dedicated management VRF when monitoring across congested links.
- ACL-based SPAN filtering reduces the volume of mirrored traffic and conserves destination port bandwidth. Always filter when you know the traffic of interest (e.g., only HTTP/HTTPS for web application troubleshooting).
- SPAN sessions persist across switch reboots (stored in running-config). After troubleshooting is complete, always remove SPAN sessions. Forgotten sessions waste switch resources and can create security blind spots if the destination port is later repurposed.

## See Also

- vlan, tc, tcpdump, tshark, iptables, nftables, vxlan, private-vlans, cos-qos

## References

- [IETF — ERSPAN Type III Draft (draft-foschiano-erspan)](https://datatracker.ietf.org/doc/html/draft-foschiano-erspan)
- [Cisco — Configuring SPAN and RSPAN (Catalyst 9300)](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst9300/software/release/17-6/configuration_guide/nmgmt/b_176_nmgmt_9300_cg/configuring_span_and_rspan.html)
- [Cisco — ERSPAN Configuration Guide (NX-OS)](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/interfaces/configuration/guide/b-cisco-nexus-9000-nx-os-interfaces-configuration-guide-93x/b-cisco-nexus-9000-nx-os-interfaces-configuration-guide-93x_chapter_011001.html)
- [Linux Kernel — ERSPAN Tunnel Documentation](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [man tc-mirred (Linux)](https://man7.org/linux/man-pages/man8/tc-mirred.8.html)
- [Open vSwitch — Port Mirroring](https://docs.openvswitch.org/en/latest/faq/configuration/)
- [Gigamon — Visibility Fabric Architecture](https://www.gigamon.com/products/visibility-fabric.html)
