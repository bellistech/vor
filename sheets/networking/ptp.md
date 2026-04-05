# PTP (Precision Time Protocol)

IEEE 1588 Precision Time Protocol for sub-microsecond clock synchronization over a LAN, using hardware timestamping and a grandmaster clock hierarchy. Achieves <100 ns accuracy with hardware support, compared to NTP's millisecond-level accuracy.

## PTP vs NTP

```
Feature          PTP (IEEE 1588)              NTP
Accuracy         <1 us (software)             1-50 ms (internet)
                 <100 ns (HW timestamping)    <1 ms (LAN)
Transport        L2 (Ethernet) or UDP/319,320 UDP/123
Architecture     Grandmaster -> slaves        Client-server (stratum)
Timestamping     Hardware (PHC) or software   Software only
Standard         IEEE 1588-2019 (PTPv2.1)     RFC 5905 (NTPv4)
Linux daemon     ptp4l + phc2sys (linuxptp)   chrony / ntpd
Use cases        5G, finance, broadcast       General purpose
```

## Clock Types

```
Grandmaster Clock (GM)
     |         Top of the hierarchy; provides reference time
     |         Typically GPS/GNSS-disciplined oscillator
     v
Boundary Clock (BC)
     |         Has multiple PTP ports; acts as slave on upstream,
     |         master on downstream. Runs BMCA per port.
     v
Ordinary Clock (OC)
     |         Single PTP port; either master or slave
     |         End devices: servers, clients
     v
Transparent Clock (TC)
              Forwards PTP packets, updates correctionField
              with residence time. Does NOT participate in BMCA.
              Types: end-to-end (E2E) or peer-to-peer (P2P)
```

## PTP Message Types

```
Event Messages (timestamped):
  Sync          Master -> Slave    Carries master timestamp t1
  Delay_Req     Slave -> Master    Slave requests delay measurement (t3)
  Pdelay_Req    Port -> Peer       P2P delay measurement request
  Pdelay_Resp   Peer -> Port       P2P delay measurement response

General Messages (not timestamped):
  Follow_Up     Master -> Slave    Carries precise t1 (two-step mode)
  Delay_Resp    Master -> Slave    Carries t4 (master receive time of Delay_Req)
  Pdelay_Resp_Follow_Up            Precise Pdelay_Resp timestamp
  Announce      Master -> All      Carries clock quality for BMCA
  Signaling     Any -> Any         Negotiate parameters (unicast, intervals)
  Management    Any -> Any         Query/set clock properties
```

## Two-Step vs One-Step Operation

```
Two-Step (default):
  1. Master sends Sync with estimated timestamp
  2. Master sends Follow_Up with precise hardware timestamp (t1)
  Most common — simpler hardware, NIC timestamps after TX

One-Step:
  1. Master sends Sync with precise timestamp embedded
  No Follow_Up needed — hardware inserts timestamp on the wire
  Requires specialized NIC/switch silicon
  Lower message rate, lower latency

ptp4l config:
  twoStepFlag    1      # two-step (default)
  twoStepFlag    0      # one-step (requires HW support)
```

## Delay Measurement

```
End-to-End (E2E) — default:
  Uses Sync + Delay_Req/Delay_Resp
  Measures full path delay master <-> slave
  Works with any network (no switch PTP support needed)
  Scales poorly with many slaves (each sends Delay_Req to master)

  Offset  = ((t2 - t1) - (t4 - t3)) / 2
  Delay   = ((t2 - t1) + (t4 - t3)) / 2

  t1 = Sync departure (master)    t2 = Sync arrival (slave)
  t3 = Delay_Req departure (slave) t4 = Delay_Req arrival (master)

Peer-to-Peer (P2P):
  Uses Pdelay_Req/Pdelay_Resp/Pdelay_Resp_Follow_Up
  Measures link delay between adjacent nodes
  Requires ALL devices on path to support P2P
  Better scaling — each link measures independently
  Used with transparent clocks
```

## Best Master Clock Algorithm (BMCA)

```
BMCA selects the grandmaster and determines master/slave roles.
Each clock advertises its properties via Announce messages.
Selection priority (highest to lowest):

  1. priority1           (0-255, lower wins, default 128)
  2. clockClass          (6=primary ref, 7=holdover, 248=default)
  3. clockAccuracy       (0x20=25ns, 0x21=100ns, 0x22=250ns, ...)
  4. offsetScaledLogVariance  (Allan variance of clock)
  5. priority2           (tiebreaker, 0-255, lower wins)
  6. clockIdentity       (EUI-64 derived from MAC, final tiebreaker)

Announce timeout: 3 * announceInterval (default 3 * 2s = 6s)
On GM failure, BMCA re-elects within ~6 seconds
```

## PTP Profiles

```
Default Profile (IEEE 1588):
  delay_mechanism    E2E
  network_transport  UDPv4
  announceInterval   1 (2 sec)
  syncInterval       0 (1 sec)
  domain             0

Telecom Profile (ITU-T G.8275.1 / G.8275.2):
  G.8275.1: Full timing support (L2, all nodes PTP-aware)
  G.8275.2: Partial timing support (L3/UDP, some non-PTP nodes)
  Used in 4G/5G mobile backhaul
  logSyncInterval    -4 (16 per second)
  domain             24-43

Power Profile (IEEE C37.238):
  For power grid substations (IEC 61850)
  Uses P2P delay mechanism
  1 PPS output for protection relay synchronization
  domain             0
  VLAN tagging required

gPTP — IEEE 802.1AS (Automotive/AV):
  Generalized PTP for time-sensitive networking (TSN)
  P2P delay only, 802.3 (Ethernet) transport
  Used in automotive (AUTOSAR), pro audio/video
  logSyncInterval    -3 (8 per second)
  Tighter BMCA ("best time-aware transmitter")
```

## PTP Domains

```
# Domain separates independent PTP instances on the same network
# Clocks in different domains ignore each other
# Valid range: 0-127

ptp4l config:
  domainNumber    0      # default domain
  domainNumber    24     # telecom profile

# Use cases for multiple domains:
# - Separate PTP instances for different applications
# - Redundant grandmasters in different domains
# - Testing new PTP config without disrupting production
```

## Hardware Timestamping

```bash
# Check NIC hardware timestamping support
ethtool -T eth0
# Look for:
#   hardware-transmit, hardware-receive
#   hardware-raw-clock
#   PTP Hardware Clock: 0    (PHC index)

# List PTP hardware clocks
ls /dev/ptp*
# /dev/ptp0  /dev/ptp1

# Read PHC time
phc_ctl /dev/ptp0 get

# Compare PHC to system clock
phc_ctl /dev/ptp0 cmp

# NICs with good PTP support:
# Intel i210, i225, X710, E810 (best — sub-10ns)
# Mellanox ConnectX-4/5/6
# Broadcom BCM57xxx
# Solarflare XtremeScale
```

## linuxptp — ptp4l (PTP Daemon)

```bash
# Install
sudo apt install linuxptp          # Debian/Ubuntu
sudo dnf install linuxptp          # Fedora/RHEL

# Run as slave with hardware timestamping
sudo ptp4l -i eth0 -m -s
# -i eth0   interface
# -m        print to stdout
# -s        slave-only mode

# Run with config file
sudo ptp4l -f /etc/ptp4l.conf

# Example /etc/ptp4l.conf
[global]
twoStepFlag             1
slaveOnly               1
priority1               128
priority2               128
domainNumber            0
clockClass              248
clockAccuracy           0xFE
offsetScaledLogVariance 0xFFFF
free_running            0
freq_est_interval       1
dscp_event              46
dscp_general            34
network_transport       UDPv4
delay_mechanism         E2E
time_stamping           hardware
tx_timestamp_timeout    10
summary_interval        0

[eth0]

# Check synchronization status
ptp4l output fields:
  master offset   -3 s2 freq  +1234 path delay     567
  # offset: ns from master (target: <100ns with HW)
  # s2: servo state (s0=unlocked, s1=step, s2=locked)
  # freq: frequency adjustment in ppb
  # path delay: measured delay in ns
```

## linuxptp — phc2sys (PHC to System Clock)

```bash
# Sync system clock to PTP hardware clock
sudo phc2sys -s eth0 -w -m
# -s eth0   source (PHC on eth0)
# -w        wait for ptp4l to lock first
# -m        print to stdout

# Sync system clock to PHC with specific offset
sudo phc2sys -s /dev/ptp0 -c CLOCK_REALTIME -O 0 -m

# Sync CLOCK_REALTIME from ptp4l (automatic PHC)
sudo phc2sys -a -r -m
# -a        auto mode (reads ptp4l via SHM)
# -r        sync CLOCK_REALTIME

# Check system vs PHC offset
phc2sys output:
  phc2sys[1234]: CLOCK_REALTIME phc offset    -12 s2 freq   +567
  # offset in ns, s2 = locked
```

## PTP over Ethernet (L2) vs UDP (L4)

```bash
# L2 transport — raw Ethernet (EtherType 0x88F7)
# Lower latency, no IP stack overhead
# Requires L2-aware switches
# Cannot cross routers
ptp4l -f /etc/ptp4l.conf
# Config: network_transport L2

# L4 transport — UDP (ports 319 event, 320 general)
# Works across routed networks (with boundary clocks)
# UDPv4: multicast 224.0.1.129 (E2E) / 224.0.0.107 (P2P)
# UDPv6: multicast ff0e::181 (E2E) / ff02::6b (P2P)
ptp4l -f /etc/ptp4l.conf
# Config: network_transport UDPv4   or   UDPv6

# Unicast negotiation (telecom profile)
# Config:
#   unicast_master_table
#   unicast_req_duration 300
```

## gPTP (IEEE 802.1AS) for Automotive and AV

```bash
# gPTP is the PTP profile for Time-Sensitive Networking (TSN)
# Used in automotive (AUTOSAR Ethernet), pro audio (AES67),
# industrial automation, and audio/video bridging (AVB)

# Run ptp4l in 802.1AS mode
sudo ptp4l -i eth0 -f /etc/gPTP.conf -m

# Example /etc/gPTP.conf
[global]
gmCapable               1
priority1                248
priority2                248
logAnnounceInterval      0
logSyncInterval          -3
logMinPdelayReqInterval  0
announceReceiptTimeout   3
syncReceiptTimeout       3
neighborPropDelayThresh  800
min_neighbor_prop_delay  -20000000
assume_two_step          1
path_trace_enabled       1
follow_up_info           1
transportSpecific        0x1
network_transport        L2
delay_mechanism          P2P

[eth0]
```

## Use Cases

```
5G Mobile Networks:
  Phase sync required for TDD (Time Division Duplex)
  ITU-T G.8275.1/G.8275.2 profiles
  <1.5 us accuracy at cell site (3GPP requirement)

High-Frequency Trading (HFT):
  MiFID II requires <100 us timestamp accuracy
  Exchange feeds timestamped with PTP (e.g., CME, NYSE)
  Custom PTP NICs (Solarflare, Xilinx)

Broadcasting (SMPTE ST 2110):
  IP-based broadcast replaces SDI
  PTP syncs audio, video, and ancillary data streams
  SMPTE ST 2059 profile

Power Grid (IEEE C37.238):
  Synchrophasor measurement (PMUs)
  IEC 61850 substation automation
  1 PPS output for protective relays

Industrial Automation (TSN):
  IEEE 802.1AS (gPTP) for factory floor
  EtherCAT, PROFINET IRT, OPC UA
  Motion control: <1 us cycle accuracy
```

## Troubleshooting

```bash
# Verify PTP traffic is reaching the interface
tcpdump -i eth0 -nn 'ether proto 0x88f7'       # L2
tcpdump -i eth0 -nn 'udp port 319 or udp port 320'  # L4

# Check ptp4l servo state
journalctl -u ptp4l | grep "master offset"
# s0 = FREERUN (not locked)
# s1 = STEP (initial step correction)
# s2 = LOCKED (normal operation)

# Check if PHC is being adjusted
phc_ctl /dev/ptp0 get           # read PHC time
cat /sys/class/ptp/ptp0/clock_name

# Check NIC driver PTP support
ethtool -i eth0 | grep driver
# Drivers with good support: igb, igc, i40e, ice, mlx5_core

# Firewall rules for PTP over UDP
iptables -A INPUT -p udp --dport 319 -j ACCEPT   # event
iptables -A INPUT -p udp --dport 320 -j ACCEPT   # general

# Common issues:
# - "no timestamp" errors: NIC/driver lacks HW timestamping
# - Large offset oscillation: switch adding variable delay (need BC/TC)
# - Slave never locks: wrong domain, VLAN mismatch, or firewall
# - freq adjustment at limit: oscillator quality too poor for PTP
```

## Tips

- Always use hardware timestamping when available; software timestamping adds 10-100 us of jitter and defeats PTP's purpose
- Run phc2sys alongside ptp4l to keep the system clock in sync with the PHC; applications reading CLOCK_REALTIME need both
- Use boundary clocks at network aggregation points; transparent clocks are simpler but require every switch in the path to support them
- Set priority1 explicitly on your intended grandmaster to avoid unexpected BMCA elections after network changes
- For telecom deployments, use G.8275.1 (full on-path support) when possible; G.8275.2 (partial) tolerates non-PTP hops but with degraded accuracy
- Monitor master offset continuously; a locked slave (s2 state) should show offset consistently under 100 ns with hardware timestamping
- Separate PTP traffic into its own VLAN or use DSCP marking (EF/46 for event messages) to prevent queuing delays
- On Intel NICs (i210, E810), verify firmware supports PTP; older firmware revisions may lack timestamping or have known bugs
- Use domain separation when running multiple PTP instances; clocks in different domains do not interfere with each other
- For gPTP (802.1AS), every device on the path must support P2P delay measurement; a single non-gPTP switch breaks the chain
- Consider PTP-aware switches with hardware timestamping (boundary or transparent clock); commodity switches add microseconds of jitter
- Test failover by shutting down the grandmaster; BMCA should re-elect within the announce timeout (default ~6 seconds)

## See Also

- ntp, ethernet, ethtool, tc, vlan

## References

- [IEEE 1588-2019 — PTPv2.1 Standard](https://standards.ieee.org/standard/1588-2019.html)
- [linuxptp Project](https://linuxptp.sourceforge.net/)
- [linuxptp Man Pages — ptp4l(8), phc2sys(8)](https://linuxptp.sourceforge.net/documentation.html)
- [ITU-T G.8275.1 — Telecom Full Timing Support](https://www.itu.int/rec/T-REC-G.8275.1)
- [ITU-T G.8275.2 — Telecom Partial Timing Support](https://www.itu.int/rec/T-REC-G.8275.2)
- [IEEE 802.1AS — gPTP for TSN](https://standards.ieee.org/standard/802_1AS-2020.html)
- [SMPTE ST 2059 — PTP for Broadcast](https://www.smpte.org/standards)
- [Red Hat — Configuring PTP with ptp4l](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/8/html/configuring_basic_system_settings/assembly_configuring-ptp-using-ptp4l_configuring-basic-system-settings)
