# CGNAT (Carrier-Grade NAT)

Large-scale network address translation deployed by service providers to extend IPv4 address lifetime by sharing public IPv4 addresses across multiple subscribers using port-based multiplexing.

## Concepts

### NAT Terminology

| Term | Meaning |
|------|---------|
| Inside local | Subscriber private address (e.g., 192.168.1.100) |
| Inside global | Post-NAT public address (shared, e.g., 203.0.113.1) |
| Outside local/global | Destination address (typically unchanged) |
| NAT44 | IPv4-to-IPv4 translation (also called LSN — Large Scale NAT) |
| NAT444 | Double NAT: CPE NAT + CGN NAT44 |
| DS-Lite | Dual-Stack Lite: IPv4-in-IPv6 tunnel to centralized CGNAT |
| NAT64 | IPv6-to-IPv4 translation (IPv6-only subscribers access IPv4 content) |
| 464XLAT | Client-side CLAT (NAT46) + provider-side PLAT (NAT64) |
| MAP-T | Mapping of Address and Port using Translation (stateless) |
| MAP-E | Mapping of Address and Port using Encapsulation (stateless) |

### CGN Placement in SP Network

```
Subscriber ──> BNG ──> Aggregation ──> CGN ──> Core/Peering
   [RFC 1918]           [100.64.0.0/10]         [Public IPv4]

# 100.64.0.0/10 (RFC 6598) is the shared address space
# Used between CPE and CGN to avoid conflicts with subscriber LANs
```

### Port Allocation Methods

| Method | Description | Logging Impact |
|--------|-------------|---------------|
| **Deterministic** | Fixed port range per subscriber, algorithmically derived | Minimal (mapping is computable) |
| **Dynamic port block** | Allocate blocks (e.g., 512 ports) on demand, extend as needed | Moderate (log block allocation/deallocation) |
| **Dynamic per-flow** | Allocate individual ports per connection | Extreme (log every connection) |

## NAT44 (Large Scale NAT)

### RFC 6888 Requirements

```
# REQ-1: Endpoint-Independent Mapping (EIM)
# Same internal IP:port always maps to same external IP:port
# Regardless of destination

# REQ-2: Endpoint-Independent Filtering (EIF) or
#         Address-Dependent Filtering (ADF)
# EIF: Accept incoming from any source to mapped port
# ADF: Accept only from previously contacted destinations

# REQ-3: Port parity preservation
# Odd internal port -> odd external port (RTP/RTCP)

# REQ-4: Hairpinning support
# Subscriber-to-subscriber traffic through CGN

# REQ-5: Fragment handling
# First fragment creates mapping; subsequent fragments forwarded

# REQ-7: At least 1000 ports per subscriber (recommended)
# REQ-8: Per-subscriber port/session limits must be configurable
# REQ-9: Application-aware ALG or port assignment for known protocols
```

### EIM/EIF/APF Behaviors

```
# Endpoint-Independent Mapping (EIM)
# Internal 10.0.0.1:5000 -> External 203.0.113.1:40000
# This mapping is used for ALL destinations
# Result: Better NAT traversal, P2P works, WebRTC works

# Endpoint-Independent Filtering (EIF)
# Any external host can send to 203.0.113.1:40000
# Most permissive, best for applications, worst for security

# Address-Dependent Filtering (ADF)
# Only hosts previously contacted by 10.0.0.1 can send inbound
# on that mapped port
# Better security, slight application impact

# Address and Port-Dependent Filtering (APDF)
# Only the exact IP:port previously contacted can send inbound
# Most restrictive, symmetric NAT behavior
# Breaks many P2P applications
```

### IOS-XR NAT44 Configuration

```
! Service CGN instance
service cgn CGN-INSTANCE
 service-type nat44 NAT44-INSTANCE
  !
  ! Inside VRF (subscriber-facing)
  inside-vrf default
   map address-pool NAT-POOL-1
   !
  !
  ! Outside VRF (internet-facing, can be same or different VRF)
  outside-vrf default
  !
  ! NAT pool definition
  address-pool NAT-POOL-1
   address-range 203.0.113.1 203.0.113.254
   !
  !
  ! Port block allocation
  portlimit 2048
  block-size 512
  !
  ! Session limits per subscriber
  session-limit per-subscriber 8000
  !
  ! ALG configuration
  alg ftp
  alg sip
  !
  ! Logging
  logging netflow version 9
   server 10.10.10.100 port 9995
   !
  !
 !
!

! Apply CGN to interfaces
interface TenGigE0/0/0/0
 description *** Inside (subscriber-facing) ***
 service cgn CGN-INSTANCE inside
 !

interface TenGigE0/0/0/1
 description *** Outside (internet-facing) ***
 service cgn CGN-INSTANCE outside
 !
```

### Deterministic NAT Configuration

```
! Deterministic NAT: predictable port mapping per subscriber
service cgn CGN-INSTANCE
 service-type nat44 NAT44-DET
  !
  inside-vrf default
   map address-pool NAT-POOL-DET
    deterministic
    !
   !
  !
  address-pool NAT-POOL-DET
   address-range 203.0.113.1 203.0.113.254
   !
  !
  ! Deterministic mapping: each subscriber gets a fixed port range
  ! Formula: subscriber_index * ports_per_sub = starting_port
  ! Subscriber 100.64.0.1 -> 203.0.113.1:1024-3071
  ! Subscriber 100.64.0.2 -> 203.0.113.1:3072-5119
  ! No per-flow logging required — mapping is computable
 !
!
```

## DS-Lite (RFC 6333)

### Architecture

```
              IPv6-only Network
CPE (B4) ═══════════════════════ BNG/CGN (AFTR)
  │                                    │
  │ IPv4-in-IPv6 softwire tunnel       │
  │ (IP protocol 4 inside IPv6)        │
  │                                    │
  [IPv4 LAN]                    [IPv4 Internet]

# B4 = Basic Bridging BroadBand element (CPE)
# AFTR = Address Family Transition Router (CGN)
```

### DS-Lite Configuration (AFTR Side)

```
! DS-Lite AFTR configuration (IOS-XR)
service cgn CGN-INSTANCE
 service-type ds-lite DSLITE-INSTANCE
  !
  ! B4 address (IPv6 tunnel endpoint on CPE)
  ! B4 elements connect via IPv6 softwire
  !
  address-pool DSLITE-POOL
   address-range 203.0.113.1 203.0.113.254
   !
  !
  ! Port and session limits (same as NAT44)
  portlimit 2048
  block-size 512
  session-limit per-subscriber 8000
  !
  ! Logging
  logging netflow version 9
   server 10.10.10.100 port 9995
   !
  !
 !
!

! Softwire tunnel interface
interface tunnel-ip 0
 ipv6 address 2001:db8::1/128
 tunnel mode ipv6
 service cgn CGN-INSTANCE inside
 !
```

## NAT64 / DNS64 (RFC 6146 / RFC 6147)

### Architecture

```
IPv6-only              NAT64              IPv4-only
Subscriber ──────────> Gateway ──────────> Content Server
2001:db8::1            Translates          93.184.216.34
                       IPv6 <-> IPv4

DNS64 synthesizes AAAA records for IPv4-only destinations:
  Query: example.com AAAA
  DNS64: Returns 64:ff9b::5db8:d822  (embeds 93.184.216.34)
  NAT64: Translates packets to/from 93.184.216.34
```

### NAT64 Well-Known Prefix

```
# 64:ff9b::/96 is the well-known prefix (RFC 6052)
# IPv4 address embedded in last 32 bits:
# 93.184.216.34 = 5db8:d822
# NAT64 address: 64:ff9b::5db8:d822

# Provider can also use a network-specific prefix (NSP)
# e.g., 2001:db8:64::/96
```

### DNS64 Configuration

```
! BIND9 DNS64 example
options {
    dns64 64:ff9b::/96 {
        clients { any; };
        mapped { !rfc1918; any; };
        exclude { 64:ff9b::/96; };
    };
};

# When a AAAA query returns NXDOMAIN or no AAAA record,
# DNS64 synthesizes a AAAA from the A record response:
# A: 93.184.216.34 -> AAAA: 64:ff9b::5db8:d822
```

## 464XLAT (RFC 6877)

### Architecture

```
IPv4 App ──> CLAT ──> IPv6 Network ──> PLAT ──> IPv4 Internet
             (NAT46)                   (NAT64)

# CLAT (Customer-side translator): In CPE or host
#   Translates IPv4 to IPv6 using provider NAT64 prefix
# PLAT (Provider-side translator): Standard NAT64 gateway
#   Translates IPv6 back to IPv4

# Solves: IPv4-only apps that don't work with DNS64/NAT64
#   (apps using IPv4 literals, not DNS)
# Used by: Android, iOS, Windows (mobile networks)
```

## MAP-T / MAP-E (RFC 7597 / RFC 7598)

### MAP-T (Translation)

```
# Stateless IPv4/IPv6 translation
# Each subscriber gets a deterministic share of an IPv4 address
# Defined by: IPv4 prefix + port range, derived from IPv6 prefix

# MAP rule defines the mapping:
# IPv6 prefix: 2001:db8::/32
# IPv4 prefix: 203.0.113.0/24
# EA-bits length: 16 (embedded address bits)
# PSID offset: 6 (port set identifier)

# Subscriber 2001:db8:0001::/48 maps to:
#   IPv4: 203.0.113.1, ports 1024-2047
# No per-flow state on the MAP-T border relay
# Massively scalable — no session table
```

### MAP-E (Encapsulation)

```
# Same mapping algorithm as MAP-T
# But uses IPv4-in-IPv6 encapsulation instead of translation
# Subscriber encapsulates IPv4 in IPv6
# MAP-E BR (Border Relay) decapsulates

# Trade-off vs MAP-T:
# MAP-E: preserves IPv4 header, better ALG compatibility
# MAP-T: no encapsulation overhead, better MTU
```

## Port Block Allocation

### Block Size Calculation

```
# Available ports per public IPv4 address:
# Range 1024-65535 = 64,512 usable ports (ports 0-1023 reserved)

# Subscribers per IPv4 address (given port block size):
# subscribers = floor(64512 / block_size)

# Block size 512:  floor(64512 / 512)  = 126 subscribers per IP
# Block size 1024: floor(64512 / 1024) = 63  subscribers per IP
# Block size 2048: floor(64512 / 2048) = 31  subscribers per IP

# Total subscribers for a /24 pool (254 usable IPs):
# 512 block:  254 * 126 = 32,004 subscribers
# 1024 block: 254 * 63  = 16,002 subscribers
# 2048 block: 254 * 31  = 7,874  subscribers
```

### Deterministic Port Mapping Formula

```
# For deterministic NAT, the mapping is computed:
# external_ip_index = subscriber_index / subscribers_per_ip
# port_start = 1024 + (subscriber_index % subscribers_per_ip) * block_size
# port_end   = port_start + block_size - 1

# Example: block_size=1024, subscriber 100.64.0.50
# subscriber_index = 50
# external_ip_index = 50 / 63 = 0 -> 203.0.113.1
# port_start = 1024 + (50 % 63) * 1024 = 1024 + 51200 = 52224
# port_end   = 52224 + 1023 = 53247
# Mapping: 100.64.0.50 -> 203.0.113.1:52224-53247
```

## Logging Requirements

### What Must Be Logged

```
# For law enforcement / abuse handling, ISP must be able to
# answer: "Who was using public IP X, port Y, at time T?"

# Minimum log fields:
# - Timestamp (UTC, millisecond precision)
# - Internal IP address
# - Internal port
# - External IP address (post-NAT)
# - External port (post-NAT)
# - Protocol (TCP/UDP/ICMP)
# - Destination IP (optional but useful)

# Deterministic NAT advantage: no per-flow logging needed
# The mapping formula answers the question directly

# Dynamic NAT: must log every port block allocation/deallocation
# Per-flow dynamic: must log every connection (massive volume)
```

### Logging Volume Estimation

```
# Per-flow logging (worst case):
# Average subscriber: 500 connections/hour
# 100K subscribers: 50M log entries/hour
# At ~150 bytes/entry: 7.5 GB/hour = 180 GB/day

# Port block logging (dynamic blocks):
# Average subscriber: 2-5 block events/hour
# 100K subscribers: 500K entries/hour
# At ~100 bytes/entry: 50 MB/hour = 1.2 GB/day

# Deterministic NAT:
# Zero runtime logging required
# Mapping is computable from configuration
# ~0 GB/day (only need config backup)
```

### NetFlow/IPFIX Logging Configuration

```
! IOS-XR CGN NetFlow logging
service cgn CGN-INSTANCE
 service-type nat44 NAT44-INSTANCE
  logging netflow version 9
   server 10.10.10.100 port 9995
    refresh-rate 600
    timeout-rate 30
    !
   !
  !
 !
!

! NetFlow export for port block allocations
flow exporter CGN-EXPORT
 destination 10.10.10.100
 transport udp 9995
 source Loopback0
 !
```

## ALG Considerations

```
# Application Layer Gateways inspect and modify payload
# Required when application embeds IP:port in data (not headers)

# Common ALGs:
# FTP:  Rewrites PORT/PASV commands
# SIP:  Rewrites Via, Contact, SDP c=/m= lines
# H.323: Rewrites Setup/Connect messages
# RTSP: Rewrites Transport header

# ALG problems at scale:
# - CPU intensive (deep packet inspection)
# - Encrypted traffic (TLS) cannot be ALG'd
# - SIP ALG is notoriously buggy — often better to disable
# - HTTP ALG: not needed (HTTP does not embed addresses)

# Best practice: Enable FTP ALG, disable SIP ALG,
# rely on ICE/STUN/TURN for VoIP and video
```

## A10 Thunder CGN

```
# A10 Thunder is a popular hardware CGN appliance

# NAT pool configuration
ip nat pool CGN-POOL 203.0.113.1 203.0.113.254 netmask /24
  port-batch-size 512
  simultaneous-batch-allocation 4
  !

# Inside source configuration
ip nat inside source list SUBSCRIBERS pool CGN-POOL

# Deterministic NAT
ip nat deterministic inside 100.64.0.0/10 outside CGN-POOL

# Logging to LSN log server
logging lsn enable
logging lsn destination 10.10.10.100 port 514
logging lsn log port-allocation
logging lsn log port-deallocation

# Session limits
ip nat translation max-entries per-ip 8000
ip nat translation tcp-timeout 300
ip nat translation udp-timeout 30
```

## Show Commands

```bash
# IOS-XR CGN show commands
show service cgn CGN-INSTANCE nat44 statistics
show service cgn CGN-INSTANCE nat44 session
show service cgn CGN-INSTANCE nat44 session inside-address 100.64.1.10

# Translation table lookup
show service cgn CGN-INSTANCE nat44 mapping inside-address 100.64.1.10

# Pool utilization
show service cgn CGN-INSTANCE nat44 pool address-pool NAT-POOL-1

# Port block allocations
show service cgn CGN-INSTANCE nat44 port-block inside-address 100.64.1.10

# Session counts and limits
show service cgn CGN-INSTANCE nat44 counters

# DS-Lite specific
show service cgn CGN-INSTANCE ds-lite statistics
show service cgn CGN-INSTANCE ds-lite session
```

## Troubleshooting

### Subscriber Cannot Reach Internet

```bash
# Verify CGN instance is active
show service cgn CGN-INSTANCE summary

# Check if subscriber has port block allocated
show service cgn CGN-INSTANCE nat44 port-block inside-address 100.64.1.10

# If no port block: check pool exhaustion
show service cgn CGN-INSTANCE nat44 pool address-pool NAT-POOL-1 | include utiliz

# Check session limit not exceeded
show service cgn CGN-INSTANCE nat44 counters | include "session limit"

# Verify routing: subscriber traffic reaching CGN inside interface?
show interface TenGigE0/0/0/0 | include "packets input"
```

### Application Breakage

```bash
# Check if ALG is enabled for the application
show running-config service cgn CGN-INSTANCE | include alg

# For VoIP/SIP: try disabling SIP ALG (often helps)
# For gaming: ensure EIM behavior, check port allocation

# For P2P: Check filtering mode (EIF needed for inbound)
# Verify hairpinning works (subscriber-to-subscriber)

# Port exhaustion per subscriber (too many connections):
show service cgn CGN-INSTANCE nat44 session inside-address 100.64.1.10 | count
# If near session limit, increase portlimit or block-size
```

### Logging Not Working

```bash
# Verify NetFlow export
show service cgn CGN-INSTANCE nat44 logging statistics

# Check collector reachability
ping 10.10.10.100 source Loopback0

# Verify NetFlow flow records arriving at collector
# On collector: check for template refresh (first records define fields)
```

## Tips

- Use RFC 6598 shared address space (100.64.0.0/10) between CPE and CGN, never RFC 1918, to avoid address conflicts with subscriber LANs.
- Deterministic NAT eliminates logging costs entirely; strongly prefer it unless subscriber counts exceed what deterministic can handle per public IP.
- Size port blocks based on heaviest users: 1024-2048 ports covers 95%+ of residential subscribers; gamers and P2P users need more.
- Always configure per-subscriber session limits to prevent a single subscriber from exhausting NAT state.
- Monitor pool utilization and set alerts at 80%; when a pool is full, new subscribers get no service with no obvious error message.
- Keep TCP translation timeout at 300 seconds and UDP at 30 seconds to match RFC 4787 recommendations and prevent state exhaustion.
- SIP ALG causes more problems than it solves in CGN; disable it and rely on ICE/STUN/TURN for VoIP NAT traversal.
- For IPv6 transition, prefer DS-Lite or 464XLAT over NAT444; both reduce the number of NAT layers from two to one.
- MAP-T/MAP-E are ideal for large-scale deployments where stateless operation eliminates the CGN as a single point of failure.
- Logging storage grows linearly with subscriber count; budget 1-2 GB/day per 100K subscribers with port-block logging.

## See Also

- ipv4, ipv6, bng, radius, subnetting, dns

## References

- [RFC 6888 — Requirements for Unicast UDP/TCP CGN](https://www.rfc-editor.org/rfc/rfc6888)
- [RFC 6598 — IANA-Reserved IPv4 Prefix for Shared Address Space (100.64.0.0/10)](https://www.rfc-editor.org/rfc/rfc6598)
- [RFC 6333 — Dual-Stack Lite Broadband Deployments Following IPv4 Exhaustion (DS-Lite)](https://www.rfc-editor.org/rfc/rfc6333)
- [RFC 6146 — Stateful NAT64](https://www.rfc-editor.org/rfc/rfc6146)
- [RFC 6147 — DNS64](https://www.rfc-editor.org/rfc/rfc6147)
- [RFC 6877 — 464XLAT](https://www.rfc-editor.org/rfc/rfc6877)
- [RFC 7597 — MAP-T](https://www.rfc-editor.org/rfc/rfc7597)
- [RFC 7598 — MAP-E](https://www.rfc-editor.org/rfc/rfc7598)
- [RFC 4787 — NAT Behavioral Requirements for Unicast UDP](https://www.rfc-editor.org/rfc/rfc4787)
- [Cisco IOS-XR CGN Configuration Guide](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/cgn/configuration/guide/b-cgn-cg-asr9000.html)
