# JunOS Firewall Filters — PFE Processing, Policer Algorithms, and uRPF Decision Trees

> *JunOS firewall filters are stateless packet classifiers executed in hardware on the Packet Forwarding Engine. Each filter is a sequential chain of terms compiled into ternary CAM (TCAM) entries, where term ordering directly impacts both correctness (first-match semantics) and performance (TCAM utilization). Policers implement token bucket rate limiters, and uRPF provides source-address validation against the forwarding table.*

---

## 1. Filter Processing Pipeline in the PFE

### Architecture: RE vs PFE

The Routing Engine (RE) is the control plane — it runs the Junos OS, routing protocols, and CLI. The Packet Forwarding Engine (PFE) is the data plane — it handles all packet forwarding in hardware (ASICs).

Firewall filters are **configured on the RE** but **compiled and installed on the PFE**. The RE's routing process daemon (rpd) compiles filter terms into microcode or TCAM entries that the PFE evaluates at line rate.

```
Control Plane (RE)                    Data Plane (PFE)
┌────────────────────┐               ┌─────────────────────────────┐
│  CLI / config      │  commit       │  Memory (TCAM)              │
│  rpd compiles      │ ──────────→   │  ┌───────────────────────┐  │
│  filter → microcode│               │  │ Term 1: match → action│  │
│                    │               │  │ Term 2: match → action│  │
│  routing table     │  FIB push     │  │ Term 3: match → action│  │
│  ──────────────────│ ──────────→   │  │ (implicit deny)       │  │
│                    │               │  └───────────────────────┘  │
└────────────────────┘               └─────────────────────────────┘
```

### Packet Processing Order

When a packet arrives on an interface:

```
1. Ingress interface
   └→ Input firewall filter (if configured)
      └→ Routing lookup (FIB)
         ├→ Transit packet:
         │  └→ Egress interface → Output firewall filter (if configured) → Wire
         └→ Exception packet (destined to device):
            └→ Loopback (lo0) input filter (if configured) → RE
```

Key distinction:
- **Transit traffic** hits ingress input filter + egress output filter (never lo0)
- **Exception traffic** hits ingress input filter + lo0 input filter (delivered to RE)
- **Locally-generated traffic** hits lo0 output filter (if any) + egress output filter

### TCAM Compilation

Each term in a firewall filter is compiled into one or more TCAM entries. TCAM (Ternary Content-Addressable Memory) matches on three states per bit: 0, 1, or X (don't care).

A single term with multiple values in the same field expands into multiple TCAM entries:

```
term EXAMPLE {
    from {
        destination-port [22 80 443];    # 3 values
        protocol tcp;
    }
    then accept;
}
# Compiles to 3 TCAM entries (one per port value)
```

TCAM is a finite resource. On Memory-based platforms (MX, EX, QFX), typical TCAM capacity ranges from 1,500 to 256,000+ entries depending on the chipset. Exceeding TCAM capacity causes filter installation failure — the commit succeeds on the RE but the PFE rejects the filter.

Monitor TCAM usage:

```
show pfe statistics traffic             # PFE-level stats
show class-of-service interface         # CoS + filter counters
```

---

## 2. Term Ordering and Filter Optimization

### First-Match Semantics

JunOS evaluates terms sequentially from top to bottom. The **first** matching term determines the action. This has two implications:

1. **Correctness**: A broad "permit all" term placed before a specific "deny" term makes the deny unreachable
2. **Performance**: Frequently matched terms placed first reduce average TCAM lookups on software-based platforms

### Optimization Strategy

On hardware-based platforms (MX series with Memory ASICs), all terms are evaluated in parallel via TCAM — term order affects correctness but not per-packet latency. On software-based or resource-constrained platforms, sequential evaluation means order affects throughput.

**Recommended ordering for a Protect-RE filter:**

| Priority | Term | Rationale |
|:---:|:---|:---|
| 1 | Routing protocols (BGP, OSPF, IS-IS) | Must never be blocked — adjacency loss |
| 2 | Management (SSH, SNMP from trusted sources) | Operator access |
| 3 | Infrastructure (NTP, DNS, RADIUS, TACACS+) | Device services |
| 4 | Rate-limited ICMP | Reachability testing |
| 5 | Explicit deny-all with counter + log | Catch-all with visibility |

### TCAM Efficiency Tips

- Use prefix lists instead of inline addresses to share TCAM entries across terms
- Consolidate adjacent terms with identical actions
- Avoid overlapping match conditions across terms (wastes entries)
- Use `apply-groups` to template common filter patterns across interfaces

```
set policy-options prefix-list MGMT-SOURCES 10.1.0.0/24
set policy-options prefix-list MGMT-SOURCES 10.2.0.0/24

set firewall family inet filter PROTECT-RE term ALLOW-SSH from source-prefix-list MGMT-SOURCES
set firewall family inet filter PROTECT-RE term ALLOW-SSH from protocol tcp
set firewall family inet filter PROTECT-RE term ALLOW-SSH from destination-port ssh
set firewall family inet filter PROTECT-RE term ALLOW-SSH then accept
```

---

## 3. Policer Token Bucket Algorithm

### Two-Color Policer

JunOS uses a **single token bucket** for two-color policers. Packets are classified as either **in-profile** (green) or **out-of-profile** (red).

```
Parameters:
  B  = bandwidth-limit (tokens/second, in bits)
  S  = burst-size-limit (bucket capacity, in bytes)

Token bucket state:
  T(t) = tokens available at time t (in bytes)
  T(0) = S (bucket starts full)
```

**Token refill**: tokens accumulate at rate $B$ bits/sec = $B/8$ bytes/sec.

$$T(t) = \min\left(S, \; T(t_{prev}) + \frac{B}{8} \times \Delta t\right)$$

**Packet arrival**: a packet of size $P$ bytes arrives:

```
if T(t) >= P:
    T(t) = T(t) - P         # deduct tokens
    action = GREEN (accept)  # in-profile
else:
    T(t) unchanged           # no deduction
    action = RED (discard)   # out-of-profile
```

### Worked Example

Policer: bandwidth-limit 1m (1 Mbps), burst-size 62500 (62.5 KB = 500 Kbit).

```
Token refill rate: 1,000,000 / 8 = 125,000 bytes/sec
Bucket capacity:   62,500 bytes

Scenario: burst of 100 packets at 1500 bytes each (150 KB total)
  - Bucket starts full: 62,500 bytes
  - Packets 1-41:  62,500 / 1,500 = 41 packets accepted (drains bucket)
  - Packets 42-100: discarded (bucket empty, refill rate << arrival rate)
  - Time to refill: 62,500 / 125,000 = 0.5 seconds
```

### Burst Size Sizing

The burst-size should accommodate legitimate traffic bursts. Minimum viable formula:

$$S_{min} = \text{bandwidth-limit} \times \text{max-acceptable-burst-duration}$$

Rule of thumb: burst-size >= 10 * interface MTU. For a 1 Mbps policer on a GigE interface (1500 byte MTU):

$$S_{recommended} = 10 \times 1500 = 15{,}000 \text{ bytes minimum}$$

Junos enforces: burst-size must be between 1500 bytes and 100 GB.

### Three-Color Policer

Three-color policers (RFC 2697 srTCM, RFC 2698 trTCM) classify packets into three categories:

```
GREEN  → in-profile         (within committed rate)
YELLOW → excess             (within peak rate, above committed)
RED    → out-of-profile     (above peak rate)
```

**Single-Rate Three-Color Marker (srTCM)** — two token buckets:

```
Parameters:
  CIR = Committed Information Rate (tokens/sec)
  CBS = Committed Burst Size (bucket C capacity)
  EBS = Excess Burst Size (bucket E capacity)

Packet of size P arrives:
  if C(t) >= P:   GREEN   (deduct from C)
  elif E(t) >= P: YELLOW  (deduct from E)
  else:           RED     (no deduction)

Both buckets refill at CIR rate. C fills first, overflow fills E.
```

**Two-Rate Three-Color Marker (trTCM)** — two buckets, two rates:

```
Parameters:
  CIR = Committed Information Rate (refills bucket C)
  PIR = Peak Information Rate (refills bucket P, PIR >= CIR)
  CBS = Committed Burst Size
  PBS = Peak Burst Size

Packet of size P arrives:
  if P > PBS or P(t) < P:  RED
  elif P > CBS or C(t) < P: YELLOW
  else:                      GREEN (deduct from both C and P)
```

JunOS three-color policer configuration:

```
set firewall three-color-policer TC-POLICER two-rate-three-color
set firewall three-color-policer TC-POLICER two-rate-three-color committed-information-rate 10m
set firewall three-color-policer TC-POLICER two-rate-three-color committed-burst-size 100k
set firewall three-color-policer TC-POLICER two-rate-three-color peak-information-rate 20m
set firewall three-color-policer TC-POLICER two-rate-three-color peak-burst-size 200k
```

---

## 4. uRPF — Strict vs Loose Mode Decision Trees

### Strict Mode

```
Packet arrives on interface I with source address S

  ┌─ FIB lookup for S
  │
  ├─ No route found → DROP (spoofed or unknown source)
  │
  └─ Route found, next-hop interface = J
     │
     ├─ J == I → ACCEPT (source reachable via arrival interface)
     │
     └─ J != I → DROP (source reachable, but via different interface)
```

Strict mode enforces **symmetric routing**. If traffic from source S normally exits via ge-0/0/1, then packets claiming source S must arrive on ge-0/0/1. This breaks in:

- Asymmetric routing topologies
- Multihomed environments with multiple uplinks
- ECMP configurations (partially — see feasible-paths)

### Loose Mode

```
Packet arrives on interface I with source address S

  ┌─ FIB lookup for S
  │
  ├─ No route found → DROP (spoofed or unknown source)
  │
  └─ Route found (any interface) → ACCEPT
```

Loose mode only verifies that a route to the source exists — it does not check which interface. A **default route satisfies** loose mode for any source address, which significantly reduces its anti-spoofing effectiveness.

### Feasible-Paths Enhancement

Feasible-paths extends strict mode to accept packets if the source is reachable via **any equal-cost or backup path** through the arrival interface:

```
Packet arrives on interface I with source address S

  ┌─ FIB lookup for S → returns set of next-hop interfaces {J1, J2, ...Jn}
  │  (includes ECMP paths and feasible backup paths)
  │
  ├─ I ∈ {J1, J2, ...Jn} → ACCEPT
  │
  └─ I ∉ {J1, J2, ...Jn} → DROP
```

This solves the ECMP problem: if the router has two equal-cost paths to 10.0.0.0/8 via ge-0/0/0 and ge-0/0/1, strict mode would only accept on one interface, but feasible-paths accepts on both.

### Mode Selection Guide

| Scenario | Recommended Mode |
|:---|:---|
| Single-homed stub network | Strict |
| Dual-homed, symmetric routing | Strict |
| Dual-homed, asymmetric routing | Loose or feasible-paths |
| ECMP environment | Strict + feasible-paths |
| Internet edge with full table | Strict (no default route) |
| Internet edge with default route | Loose (default satisfies) — limited value |
| Internal core routers | Generally not needed |

---

## 5. Comparison with Cisco ACLs and Linux nftables

### Conceptual Mapping

| Feature | JunOS Firewall Filters | Cisco IOS ACLs | Linux nftables |
|:---|:---|:---|:---|
| Filter unit | Term | ACE (Access Control Entry) | Rule |
| Container | Filter (under family) | Named/numbered ACL | Chain (in table) |
| Evaluation | Sequential, first match | Sequential, first match | Sequential, first match |
| Default action | Implicit deny | Implicit deny | Chain policy (configurable) |
| Stateful? | No (stateless) | No (standard/extended) / Yes (reflexive/CBAC) | Yes (conntrack) |
| Non-terminating actions | count, log, policer, next term | log (then continues to next ACE) | counter, log, queue (with continue) |
| Rate limiting | Policer (inline or reference) | CAR / MQC policer (separate config) | limit expression / meter |
| Application point | Interface (input/output) | Interface (in/out) | Hook (input/forward/output) |

### Syntax Comparison

```
# JunOS: Allow SSH from 10.0.0.0/8
set firewall family inet filter ACL term ALLOW-SSH from source-address 10.0.0.0/8
set firewall family inet filter ACL term ALLOW-SSH from protocol tcp
set firewall family inet filter ACL term ALLOW-SSH from destination-port 22
set firewall family inet filter ACL term ALLOW-SSH then accept

# Cisco IOS: Allow SSH from 10.0.0.0/8
ip access-list extended ACL
 permit tcp 10.0.0.0 0.255.255.255 any eq 22

# Linux nftables: Allow SSH from 10.0.0.0/8
nft add rule inet filter input ip saddr 10.0.0.0/8 tcp dport 22 accept
```

### Key Differences

**JunOS vs Cisco:**
- JunOS uses **prefix notation** (10.0.0.0/8), Cisco uses **wildcard masks** (10.0.0.0 0.255.255.255)
- JunOS terms have names; Cisco ACEs are identified by sequence number
- JunOS `reject` sends ICMP unreachable; Cisco `deny` is silent (like JunOS `discard`)
- JunOS `next term` has no direct Cisco equivalent — in Cisco, `log` is the only non-terminating action
- JunOS supports `input-list` (multiple filters on one interface); Cisco does not natively chain ACLs

**JunOS vs Linux nftables:**
- nftables is **stateful** by default (conntrack integration); JunOS filters are stateless
- nftables supports **sets and maps** for efficient multi-value matching; JunOS uses prefix-lists (separate config)
- nftables chains have **configurable default policies**; JunOS always has implicit deny
- nftables supports **NAT, masquerade, and packet mangling** in the same framework; JunOS uses separate configuration hierarchies
- nftables rules are applied at the **host level**; JunOS filters are per-interface on a network device

---

## 6. Filter-Based Forwarding — Policy Routing

Filter-based forwarding (FBF) uses firewall filter actions to steer packets into alternate routing instances, overriding the default FIB lookup.

```
Normal path:   Packet → FIB lookup → forwarding
FBF path:      Packet → Filter match → routing-instance → separate FIB → forwarding
```

Use cases:
- Source-based routing (different customers via different uplinks)
- Application-based routing (web traffic via proxy, VoIP via low-latency path)
- Compliance routing (regulated traffic must traverse inspection device)

The `routing-instance` action in a filter term is a **terminating action** — evaluation stops and the packet is forwarded via the specified instance's routing table.

```
set routing-instances VR-WEB instance-type forwarding
set routing-instances VR-WEB routing-options static route 0.0.0.0/0 next-hop 10.0.1.1

set firewall family inet filter PBR term WEB-TRAFFIC from destination-port [80 443]
set firewall family inet filter PBR term WEB-TRAFFIC then routing-instance VR-WEB
set firewall family inet filter PBR term DEFAULT then accept

# RIB group required to import interface routes into forwarding instance
set routing-options interface-routes rib-group inet WEB-RIB
set routing-instances VR-WEB routing-options interface-routes rib-group inet WEB-RIB
```

---

## 7. Putting It All Together — Production Protect-RE Design

### Design Principles

1. **Whitelist model**: Explicit permit for required services, deny everything else
2. **Source restriction**: Management protocols only from trusted prefixes
3. **Rate limiting**: All ICMP and potentially abusive protocols get policers
4. **Logging**: Denied traffic counted and logged for security monitoring
5. **Protocol safety**: Routing protocols (BGP, OSPF, IS-IS, LDP) always permitted from valid peers

### Complete Production Filter

```
# Prefix lists
set policy-options prefix-list NMS-SERVERS 10.1.0.0/24
set policy-options prefix-list BGP-PEERS 10.0.0.1/32
set policy-options prefix-list BGP-PEERS 10.0.0.2/32
set policy-options prefix-list NTP-SERVERS 10.5.0.1/32
set policy-options prefix-list NTP-SERVERS 10.5.0.2/32

# Policers
set firewall policer ICMP-1M if-exceeding bandwidth-limit 1m
set firewall policer ICMP-1M if-exceeding burst-size-limit 15k
set firewall policer ICMP-1M then discard

# Filter
set firewall family inet filter PROTECT-RE term BGP from source-prefix-list BGP-PEERS
set firewall family inet filter PROTECT-RE term BGP from protocol tcp
set firewall family inet filter PROTECT-RE term BGP from destination-port bgp
set firewall family inet filter PROTECT-RE term BGP then count BGP-IN
set firewall family inet filter PROTECT-RE term BGP then accept

set firewall family inet filter PROTECT-RE term OSPF from protocol ospf
set firewall family inet filter PROTECT-RE term OSPF then count OSPF-IN
set firewall family inet filter PROTECT-RE term OSPF then accept

set firewall family inet filter PROTECT-RE term SSH from source-prefix-list NMS-SERVERS
set firewall family inet filter PROTECT-RE term SSH from protocol tcp
set firewall family inet filter PROTECT-RE term SSH from destination-port ssh
set firewall family inet filter PROTECT-RE term SSH then count SSH-IN
set firewall family inet filter PROTECT-RE term SSH then accept

set firewall family inet filter PROTECT-RE term SNMP from source-prefix-list NMS-SERVERS
set firewall family inet filter PROTECT-RE term SNMP from protocol udp
set firewall family inet filter PROTECT-RE term SNMP from destination-port snmp
set firewall family inet filter PROTECT-RE term SNMP then count SNMP-IN
set firewall family inet filter PROTECT-RE term SNMP then accept

set firewall family inet filter PROTECT-RE term NTP from source-prefix-list NTP-SERVERS
set firewall family inet filter PROTECT-RE term NTP from protocol udp
set firewall family inet filter PROTECT-RE term NTP from destination-port ntp
set firewall family inet filter PROTECT-RE term NTP then count NTP-IN
set firewall family inet filter PROTECT-RE term NTP then accept

set firewall family inet filter PROTECT-RE term ICMP from protocol icmp
set firewall family inet filter PROTECT-RE term ICMP from icmp-type [echo-request echo-reply unreachable time-exceeded]
set firewall family inet filter PROTECT-RE term ICMP then policer ICMP-1M
set firewall family inet filter PROTECT-RE term ICMP then count ICMP-IN
set firewall family inet filter PROTECT-RE term ICMP then accept

set firewall family inet filter PROTECT-RE term TRACEROUTE from protocol udp
set firewall family inet filter PROTECT-RE term TRACEROUTE from destination-port 33434-33534
set firewall family inet filter PROTECT-RE term TRACEROUTE then policer ICMP-1M
set firewall family inet filter PROTECT-RE term TRACEROUTE then count TRACE-IN
set firewall family inet filter PROTECT-RE term TRACEROUTE then accept

set firewall family inet filter PROTECT-RE term DENY-ALL then count DENIED
set firewall family inet filter PROTECT-RE term DENY-ALL then log
set firewall family inet filter PROTECT-RE term DENY-ALL then syslog
set firewall family inet filter PROTECT-RE term DENY-ALL then discard

# Apply
set interfaces lo0 unit 0 family inet filter input PROTECT-RE
```

## Prerequisites

- IP addressing, TCP/IP protocol suite, routing fundamentals, interface configuration, prefix notation

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| TCAM filter lookup (hardware) | O(1) | O(terms * expansions) |
| Sequential term evaluation (software) | O(n) | O(n) |
| Token bucket policer decision | O(1) | O(1) per policer |
| uRPF FIB lookup | O(1) avg (hash) | O(routes) |

---

*A firewall filter is the last line of defense between the network and the routing engine. The implicit deny is both your greatest safety net and your greatest risk — every service the RE needs must be explicitly permitted, or the device becomes unreachable. Design filters in lab, test with counters, deploy with a console cable connected.*
