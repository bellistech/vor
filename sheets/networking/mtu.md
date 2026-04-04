# MTU (Maximum Transmission Unit & Path MTU Discovery)

The Maximum Transmission Unit defines the largest packet or frame size (in bytes) that a network interface can transmit without fragmentation. Path MTU Discovery (PMTUD) dynamically determines the smallest MTU along an end-to-end path using ICMP feedback, while Packetization Layer PMTUD (PLPMTUD) probes without relying on ICMP. Correct MTU configuration is critical for tunneled environments (GRE, VXLAN, Geneve, IPsec, WireGuard) where encapsulation overhead reduces the effective payload.

---

## MTU Layers and Terminology

```
Layer          Name              Typical Value    Includes
─────────────  ────────────────  ──────────────   ─────────────────────────
Layer 2        Ethernet MTU      1500 bytes       Ethernet payload only
               (frame payload)                    (excludes 14B header + 4B FCS)

Layer 3        IP MTU            1500 bytes       IP header + payload
               (link MTU)                         (same as Ethernet MTU)

Layer 4        TCP MSS           1460 bytes       TCP payload only
                                                  (IP MTU - 20B IP - 20B TCP)

Jumbo          Jumbo frame       9000 bytes       Ethernet payload
                                                  (common data center setting)

Minimum        IPv4 minimum MTU  68 bytes         Required by RFC 791
               IPv6 minimum MTU  1280 bytes       Required by RFC 8200

# Note: Ethernet frame on the wire = 14 (header) + payload + 4 (FCS) = up to 1518 bytes
# With 802.1Q VLAN tag: 1518 + 4 = 1522 bytes (baby giant)
# With QinQ (double tag): 1518 + 8 = 1526 bytes
```

## Path MTU Discovery (PMTUD)

```
Host A ──── Router 1 ──── Router 2 ──── Host B
MTU=1500    MTU=1500      MTU=1400      MTU=1500

1. Host A sends 1500-byte packet with DF (Don't Fragment) bit set
2. Router 2 cannot forward (1500 > 1400)
3. Router 2 sends ICMP back to Host A:
   IPv4: Type 3, Code 4 (Destination Unreachable, Fragmentation Needed)
         Contains: Next-Hop MTU = 1400
   IPv6: Type 2, Code 0 (Packet Too Big)
         Contains: MTU = 1400
4. Host A reduces packet size to 1400 and retransmits
5. Path MTU cached in routing table (ip route get shows PMTU)

# IPv4 PMTUD: RFC 1191 (depends on ICMP Type 3 Code 4)
# IPv6 PMTUD: RFC 8201 (depends on ICMPv6 Packet Too Big)
# IPv6 NEVER fragments at routers — only at the source
```

### PMTUD System Configuration

```bash
# Check current PMTUD behavior (Linux)
sysctl net.ipv4.ip_no_pmtu_disc
# 0 = PMTUD enabled (default, uses DF bit)
# 1 = PMTUD disabled (never set DF bit, allows fragmentation)
# 2 = PMTUD enabled but use interface MTU as initial value
# 3 = PMTUD disabled, fragment at PMTU if known

# Enable PMTUD
sysctl -w net.ipv4.ip_no_pmtu_disc=0

# View cached PMTU values
ip route get 10.0.0.2
# 10.0.0.2 via 192.168.1.1 dev eth0 ... mtu 1400

# Show all PMTU cache entries
ip route show cache

# Flush PMTU cache (force re-discovery)
ip route flush cache

# PMTU aging (seconds before cached PMTU expires)
sysctl net.ipv4.route.mtu_expires
# Default: 600 (10 minutes)

# Minimum PMTU the kernel will accept
sysctl net.ipv4.route.min_pmtu
# Default: 552 bytes
```

## PLPMTUD — Packetization Layer PMTUD (RFC 8899)

```bash
# PLPMTUD probes the path with increasing packet sizes
# without relying on ICMP feedback (which may be filtered)

# Works at Layer 4 (TCP, QUIC, SCTP) using probe packets
# that can be retransmitted if lost

# Algorithm phases:
# 1. BASE    — start at minimum safe MTU (1200 for IPv6, 68 for IPv4)
# 2. SEARCH  — binary search upward with probe packets
# 3. DONE    — optimal PMTU found, periodic re-probing
# 4. ERROR   — probe failed, reduce PMTU

# QUIC uses PLPMTUD by default (RFC 9000, Section 14)
# TCP can use PLPMTUD but most implementations still use classic PMTUD

# Linux TCP PLPMTUD support
sysctl net.ipv4.tcp_mtu_probing
# 0 = disabled (default)
# 1 = enabled only when ICMP blackhole detected
# 2 = always enabled

# Enable TCP MTU probing
sysctl -w net.ipv4.tcp_mtu_probing=1

# Base MSS for probing (starting point)
sysctl net.ipv4.tcp_base_mss
# Default: 1024

# Minimum MSS for probing (floor)
sysctl net.ipv4.tcp_min_snd_mss
# Default: 48

# Probe size (MSS value to try)
sysctl net.ipv4.tcp_probe_threshold
# Default: 8 (probe when MSS could increase by this many bytes)
```

## DF Bit and Fragmentation

```bash
# IPv4 DF (Don't Fragment) bit in IP header
# Located in the Flags field (byte offset 6, bit 1)
#
#  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
# |         Identification        |Flags|    Fragment Offset       |
# +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
#                                  ^^^
#                                  ||+-- MF (More Fragments)
#                                  |+--- DF (Don't Fragment)
#                                  +---- Reserved (must be 0)

# Test PMTUD with DF bit set (will get ICMP if MTU exceeded)
# Linux (-M do = set DF bit)
ping -M do -s 1472 10.0.0.2
# 1472 + 8 (ICMP) + 20 (IP) = 1500 bytes total

# macOS (-D = set DF bit)
ping -D -s 1472 10.0.0.2

# Windows (-f = set DF bit)
# ping -f -l 1472 10.0.0.2

# Find exact PMTU with binary search
ping -M do -s 1473 10.0.0.2    # too big (1501 total) → ICMP error
ping -M do -s 1472 10.0.0.2    # fits (1500 total) → success

# IPv6 has no DF bit — ALL IPv6 packets are implicitly DF
# Routers never fragment IPv6; only the source can fragment
# IPv6 fragmentation uses Extension Header (Next Header = 44)

# Check if a packet was fragmented
tcpdump -i eth0 -vv 'ip[6:2] & 0x3fff != 0'
# Captures packets with non-zero fragment offset or MF bit
```

## MSS Clamping (TCP)

```bash
# MSS (Maximum Segment Size) is negotiated in TCP SYN
# MSS = MTU - IP header (20) - TCP header (20) = MTU - 40
# With TCP options (timestamps): MSS = MTU - 40 - 12 = MTU - 52

# MSS clamping forces a lower MSS in TCP SYN packets
# Used when PMTUD is broken (ICMP blocked) to prevent blackholes

# iptables MSS clamping (match on SYN, rewrite MSS)
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN \
    -j TCPMSS --set-mss 1400

# Clamp MSS to PMTU automatically
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN \
    -j TCPMSS --clamp-mss-to-pmtu

# nftables equivalent
nft add rule inet mangle forward tcp flags syn / syn,rst \
    counter tcp option maxseg size set 1400

# nftables auto-clamp to PMTU
nft add rule inet mangle forward tcp flags syn / syn,rst \
    counter tcp option maxseg size set rt mtu

# PPPoE MSS clamp (common for DSL/FTTH)
# PPPoE adds 8 bytes → effective MTU = 1492
# MSS = 1492 - 40 = 1452
iptables -t mangle -A FORWARD -o ppp0 -p tcp --tcp-flags SYN,RST SYN \
    -j TCPMSS --set-mss 1452

# Verify MSS in captures
tcpdump -i eth0 -nn 'tcp[tcpflags] & (tcp-syn) != 0' -vv | grep -i mss
```

## Tunnel MTU Overhead Calculations

```bash
# Every tunnel adds encapsulation overhead, reducing inner MTU
# Overhead = outer headers added by the tunnel

# GRE (Generic Routing Encapsulation)
# Outer IP (20) + GRE header (4) = 24 bytes
# With GRE key: +4 = 28 bytes
# With GRE key + sequence: +8 = 32 bytes
# Inner MTU = 1500 - 24 = 1476

# GRE over IPv6
# Outer IPv6 (40) + GRE (4) = 44 bytes
# Inner MTU = 1500 - 44 = 1456

# VXLAN
# Outer Ethernet (14) + Outer IP (20) + Outer UDP (8) + VXLAN (8) = 50 bytes
# Inner MTU = 1500 - 50 = 1450

# Geneve (no options)
# Outer Ethernet (14) + Outer IP (20) + Outer UDP (8) + Geneve (8) = 50 bytes
# Inner MTU = 1500 - 50 = 1450
# With TLV options: add 4*N bytes per option

# IPsec (transport mode, AES-256-GCM)
# ESP header (8) + IV (8) + pad (0-15) + pad len (1) + next hdr (1) + ICV (16) = 34-49
# Typical overhead: ~50-73 bytes
# Inner MTU ≈ 1500 - 73 = 1427

# IPsec (tunnel mode, AES-256-GCM)
# Outer IP (20) + ESP (8) + IV (8) + inner IP (20 already counted) + pad + ICV
# Typical overhead: ~73-93 bytes
# Inner MTU ≈ 1500 - 73 = 1427 (varies with padding)

# WireGuard
# Outer IP (20) + Outer UDP (8) + WireGuard (32) = 60 bytes
# Inner MTU = 1500 - 60 = 1440
# With IPv6 outer: 40 + 8 + 32 = 80 bytes → inner = 1420

# MPLS (single label)
# MPLS label (4 bytes per label)
# Inner MTU = 1500 - 4 = 1496
# Double label (e.g., VPN): 1500 - 8 = 1492

# Summary table:
# Tunnel Type          Overhead (IPv4 outer)  Inner MTU (1500 base)
# ─────────────────    ─────────────────────  ────────────────────
# GRE (basic)          24 bytes               1476
# GRE + key + seq      32 bytes               1468
# VXLAN                50 bytes               1450
# Geneve (no opts)     50 bytes               1450
# Geneve + 16B opts    66 bytes               1434
# IPsec tunnel AES-GCM ~73 bytes              ~1427
# WireGuard            60 bytes               1440
# MPLS (1 label)        4 bytes               1496
# MPLS (2 labels)       8 bytes               1492
```

## MTU Black Holes

```bash
# MTU black hole: path where PMTUD fails because ICMP is filtered
# Symptoms:
#   - Small packets (pings, DNS) work fine
#   - Large transfers (HTTP, file copies) hang or timeout
#   - TCP SYN/SYN-ACK succeeds (small packets)
#   - Data transfer stalls when window fills with large segments

# Diagnosis steps:

# 1. Test with decreasing sizes and DF bit
for size in 1472 1400 1300 1200 1100 1000; do
    echo -n "Testing $size: "
    ping -M do -s $size -c 1 -W 2 10.0.0.2 2>&1 | tail -1
done

# 2. Use tracepath to find the MTU bottleneck
tracepath 10.0.0.2
# Reports asymmetric PMTU and bottleneck hop

# tracepath6 for IPv6
tracepath6 2001:db8::2

# 3. Check for ICMP filtering along the path
# If no "Frag needed" comes back, PMTUD is broken
tcpdump -i eth0 'icmp[0] = 3 and icmp[1] = 4'   # IPv4 Frag Needed
tcpdump -i eth0 'icmp6[0] = 2'                    # IPv6 Packet Too Big

# 4. Enable TCP MTU probing to work around blackholes
sysctl -w net.ipv4.tcp_mtu_probing=1

# 5. Apply MSS clamping as a last resort
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN \
    -j TCPMSS --clamp-mss-to-pmtu

# Common PMTUD blackhole locations:
# - Firewalls blocking ICMP "Frag Needed" / "Packet Too Big"
# - IPsec tunnels with incorrect MTU
# - Cloud provider VPC with hidden encapsulation
# - PPPoE links (MTU 1492) behind a 1500-byte path
```

## Interface MTU Configuration

```bash
# Set interface MTU
ip link set eth0 mtu 9000

# View current MTU
ip link show eth0 | grep mtu

# View all interface MTUs
ip -br link show

# Set MTU permanently (systemd-networkd)
# /etc/systemd/network/10-eth0.network
# [Link]
# MTUBytes=9000

# Set MTU permanently (Netplan)
# /etc/netplan/01-config.yaml
# network:
#   ethernets:
#     eth0:
#       mtu: 9000

# Verify jumbo frames work end-to-end
ping -M do -s 8972 10.0.0.2
# 8972 + 20 (IP) + 8 (ICMP) = 9000 bytes

# Check NIC maximum supported MTU
ip -d link show eth0 | grep maxmtu
# If maxmtu not shown, check driver docs

# VLAN interface MTU
# Must be <= parent interface MTU (minus 4 bytes for VLAN tag if needed)
ip link set eth0 mtu 9000
ip link add link eth0 name eth0.100 type vlan id 100
ip link set eth0.100 mtu 9000

# Bridge MTU
# Set on bridge and all member interfaces
ip link set br0 mtu 9000
ip link set eth0 mtu 9000
ip link set eth1 mtu 9000

# Bond MTU
ip link set bond0 mtu 9000
```

## Jumbo Frames

```bash
# Jumbo frames = MTU > 1500 bytes (typically 9000)
# Supported by: most 1G/10G/25G/40G/100G Ethernet NICs
# NOT supported on: most consumer switches, some ISP uplinks

# Benefits:
# - Fewer packets for same data → less CPU interrupt overhead
# - Higher throughput for large transfers (fewer headers per byte)
# - Required for overlay networks to avoid inner fragmentation

# Risks:
# - ALL devices on the path must support the same jumbo MTU
# - One device with 1500 MTU silently drops jumbo frames if DF is set
# - Jumbo + PMTUD can create subtle black holes

# Test jumbo frame support end-to-end
# Set both sides to 9000 MTU first, then:
ping -M do -s 8972 10.0.0.2
# Success = jumbo frames work on this path

# Common jumbo MTU values:
# 9000  — most common data center setting
# 9216  — Cisco default jumbo (includes L2 headers)
# 9014  — some vendors
# 1500  — standard Ethernet (no jumbo)

# Baby giants (1501-1999 bytes):
# Needed for VLAN tags (1504), MPLS (1504-1508), Q-in-Q (1508)
# Most switches handle baby giants even without jumbo frame support

# Enable jumbo on a switch (example: Linux bridge)
ip link set eth0 mtu 9000
ip link set eth1 mtu 9000
ip link set br0 mtu 9000
```

## Kernel Sysctl Settings for MTU

```bash
# All MTU-related sysctls

# PMTUD control
sysctl net.ipv4.ip_no_pmtu_disc          # 0=enabled, 1=disabled
sysctl net.ipv4.route.mtu_expires        # PMTU cache timeout (600s)
sysctl net.ipv4.route.min_pmtu           # minimum PMTU accepted (552)

# TCP MTU probing (PLPMTUD)
sysctl net.ipv4.tcp_mtu_probing          # 0/1/2
sysctl net.ipv4.tcp_base_mss             # starting MSS for probing (1024)
sysctl net.ipv4.tcp_min_snd_mss          # minimum MSS floor (48)
sysctl net.ipv4.tcp_probe_threshold      # probe step size (8)

# IPv6 MTU
sysctl net.ipv6.conf.all.mtu             # IPv6 interface MTU
sysctl net.ipv6.route.mtu_expires        # IPv6 PMTU cache timeout

# IP fragmentation (when PMTUD is disabled or DF not set)
sysctl net.ipv4.ipfrag_high_thresh       # max memory for fragment reassembly
sysctl net.ipv4.ipfrag_low_thresh        # min memory for fragment reassembly
sysctl net.ipv4.ipfrag_time              # fragment reassembly timeout (30s)
sysctl net.ipv4.ipfrag_max_dist          # max reordering distance (64)

# IPv6 fragmentation
sysctl net.ipv6.ip6frag_high_thresh      # max memory for IPv6 fragments
sysctl net.ipv6.ip6frag_low_thresh       # min memory for IPv6 fragments
sysctl net.ipv6.ip6frag_time             # reassembly timeout (60s)

# Recommended production settings for tunnel-heavy environments:
sysctl -w net.ipv4.ip_no_pmtu_disc=0
sysctl -w net.ipv4.tcp_mtu_probing=1
sysctl -w net.ipv4.tcp_base_mss=1024
sysctl -w net.ipv4.route.mtu_expires=600
```

---

## Tips

- Always test end-to-end MTU with `ping -M do -s <size>` before deploying tunnels or jumbo frames. A single link with a lower MTU will silently drop oversized packets if ICMP is filtered.
- MSS clamping (`iptables -j TCPMSS --clamp-mss-to-pmtu`) is the most reliable fix for PMTUD black holes. It works at the TCP layer and does not depend on ICMP being forwarded correctly.
- Enable `tcp_mtu_probing=1` on all Linux servers. It automatically detects and works around PMTUD black holes by probing with different segment sizes when retransmissions are detected.
- When running overlay networks (VXLAN, Geneve, WireGuard), set the underlay MTU to at least 1600 (ideally 9000) to avoid fragmenting inner 1500-byte packets. The overlay interface MTU should be set to `underlay_mtu - overhead`.
- IPv6 never fragments at intermediate routers. If a packet is too large, the router sends ICMPv6 Packet Too Big and drops the packet. This makes PMTUD even more critical for IPv6 than IPv4.
- Tunnel MTU overhead stacks. GRE inside IPsec adds both overheads: outer IP (20) + ESP (~50) + GRE (24) + inner packet. Calculate the total overhead for your specific tunnel stack, not just individual layers.
- PMTU cache entries expire (default 600 seconds). After expiry, the host re-probes with full-size packets, potentially hitting the MTU bottleneck again. Lower `mtu_expires` in high-churn environments.
- Jumbo frames must be consistent across the entire L2 domain. One port at 1500 MTU on a 9000 MTU fabric will cause intermittent drops for large frames — the hardest kind of network issue to debug.
- Watch for baby giants (1501--1522 bytes) caused by VLAN tags or MPLS labels. Most modern switches handle these, but older hardware may drop them. Check switch port error counters for "giants" or "oversize".
- The 1280-byte minimum MTU for IPv6 (vs 68 bytes for IPv4) means IPv6 networks have a much higher guaranteed baseline. PLPMTUD (RFC 8899) can safely start probing at 1280 for any IPv6 path.

---

## See Also

- tcp, ipv6, icmp, vxlan, geneve, ipsec

## References

- [RFC 1191 — Path MTU Discovery (IPv4)](https://www.rfc-editor.org/rfc/rfc1191)
- [RFC 8201 — Path MTU Discovery for IPv6](https://www.rfc-editor.org/rfc/rfc8201)
- [RFC 8899 — PLPMTUD (Packetization Layer Path MTU Discovery)](https://www.rfc-editor.org/rfc/rfc8899)
- [RFC 791 — Internet Protocol (IPv4, minimum MTU)](https://www.rfc-editor.org/rfc/rfc791)
- [RFC 8200 — Internet Protocol Version 6 (IPv6)](https://www.rfc-editor.org/rfc/rfc8200)
- [RFC 4821 — PLPMTUD (original, superseded by RFC 8899)](https://www.rfc-editor.org/rfc/rfc4821)
- [Linux Kernel — IP Sysctl](https://www.kernel.org/doc/html/latest/networking/ip-sysctl.html)
- [man tracepath](https://linux.die.net/man/8/tracepath)
