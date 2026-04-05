# JunOS NAT — Processing Order, Session Architecture, and Scaling Analysis

> *SRX NAT operates within the flow-based forwarding pipeline, applying translations at precise points relative to route lookup and security policy evaluation. Understanding the exact processing order, the interaction between NAT types, and the session-based translation model is essential for correct design and troubleshooting on the JNCIE-SEC.*

---

## 1. NAT Processing Order in the SRX Pipeline

### Complete Packet Flow with NAT

When a packet enters the SRX, it traverses a well-defined pipeline. NAT types are evaluated at specific stages:

```
Packet arrives on ingress interface
│
├─ 1. Screen checks (IDS/IPS screening)
│
├─ 2. Static NAT lookup
│     ├─ If match: translate destination (and reverse-translate source on return)
│     └─ If no match: continue
│
├─ 3. Destination NAT lookup
│     ├─ If match: translate destination IP (and optionally port)
│     └─ If no match: continue
│
├─ 4. Route lookup (uses TRANSLATED destination)
│     └─ Determines egress zone and interface
│
├─ 5. Security policy evaluation
│     ├─ Source zone: determined by ingress interface
│     ├─ Destination zone: determined by route lookup result
│     ├─ Source IP: ORIGINAL (pre-SNAT)
│     ├─ Destination IP: TRANSLATED (post-DNAT or post-static)
│     └─ Policy must match the above tuple
│
├─ 6. Application services (UTM, IDP, AppFW)
│
├─ 7. Source NAT lookup
│     ├─ If match: translate source IP (and optionally port)
│     └─ If no match: packet uses original source
│
└─ 8. Packet forwarded to egress interface
```

### Why This Order Matters

The processing order creates a critical asymmetry that is the source of most NAT configuration errors:

**Security policies see post-DNAT destinations but pre-SNAT sources.** This means:

- When writing a policy for inbound traffic to a server at 10.1.1.100 (NATed from 203.0.113.5), the policy destination-address must be `10.1.1.100`, not `203.0.113.5`
- When writing a policy for outbound traffic from 10.0.0.0/8 (SNATed to 203.0.113.10), the policy source-address must be `10.0.0.0/8`, not `203.0.113.10`

**Static NAT takes precedence over destination NAT.** If both a static rule and a destination rule match the same destination, the static rule wins because it is evaluated first. This is by design — static NAT provides bidirectional mapping, which is inherently more specific.

### Return Traffic (Reverse Translation)

NAT is session-based. When the SRX creates a session entry, it records both the forward and reverse translations:

```
Forward flow:
  Src: 10.1.1.5:45000 → Dst: 93.184.216.34:80
  NAT: Src → 203.0.113.10:12000 (source NAT applied)

Reverse flow (automatically created):
  Src: 93.184.216.34:80 → Dst: 203.0.113.10:12000
  NAT: Dst → 10.1.1.5:45000 (reverse source NAT)
```

Return traffic matches the reverse flow entry and is translated without re-evaluating NAT rules. This is true for all three NAT types.

---

## 2. Source vs Destination NAT Rule Evaluation

### Rule Set Selection

NAT rule sets are scoped by `from` and `to` context. Only rule sets matching the traffic's zone/interface/routing-instance pair are evaluated:

```
Source NAT rule sets:     from zone X → to zone Y
Destination NAT rule sets: from zone X → (to is implicit from route)
Static NAT rule sets:      from zone X → (bidirectional)
```

For source NAT, the `to` context is determined after the route lookup. For destination NAT and static NAT, only the `from` context matters because destination NAT occurs before route lookup.

### Rule Evaluation Within a Rule Set

Within a selected rule set, rules are evaluated sequentially. The first matching rule is applied. Match criteria include:

- Source address (source NAT rules)
- Destination address (destination/static NAT rules)
- Destination port (destination NAT rules)
- Application protocol (destination NAT rules, since Junos 15.1)

Once a rule matches, no further rules in that rule set (or subsequent rule sets) are evaluated. If no rule matches across all applicable rule sets, no translation occurs.

### Multiple Rule Sets

Multiple rule sets can exist for the same NAT type. They are evaluated in configuration order. The `insert` command controls ordering:

```
# Rule set evaluation order:
# 1. First rule set in config → all its rules (top to bottom)
# 2. Second rule set in config → all its rules (top to bottom)
# ... and so on until a match or exhaustion
```

This allows modular NAT design — separate rule sets for different zone pairs — while maintaining deterministic evaluation.

---

## 3. NAT and Security Policy Interaction

### The Policy Evaluation Window

Security policies operate in a "window" between destination NAT and source NAT:

```
                     ┌──────────────────────────────┐
  Static/Dest NAT   │     Security Policy Match     │   Source NAT
  ───────────────→   │  src: original                │  ───────────→
  (translates dst)   │  dst: translated              │  (translates src)
                     │  zone: ingress → egress       │
                     │  app: identified by ALG/DPI   │
                     └──────────────────────────────┘
```

### Common Pitfall: Address Book Entries

Address book entries must reflect what the policy sees:

- For a destination-NATed server (public 203.0.113.5 → private 10.1.1.100):
  - The address book in the **destination zone** (e.g., trust) must contain `10.1.1.100`
  - NOT `203.0.113.5` — that is the pre-DNAT address the policy never sees

- For a source-NATed client (private 10.1.1.5 → public 203.0.113.10):
  - The address book in the **source zone** (e.g., trust) must contain `10.1.1.5`
  - NOT `203.0.113.10` — that is the post-SNAT address the policy never sees

### NAT with Global Policies

Global policies (which match without zone context) still see the same translated/untranslated combination. The zone-agnostic nature of global policies does not change when NAT translations are visible.

---

## 4. Persistent NAT and SIP

### The Problem

Standard source NAT with PAT creates ephemeral port mappings. When a SIP phone registers with an external SIP proxy, the proxy records the phone's public IP:port. If the NAT mapping changes (due to timeout or port reuse), the proxy can no longer reach the phone for incoming calls.

### Persistent NAT Solution

Persistent NAT binds an internal IP:port to a specific external IP:port for the duration of the inactivity timeout:

```
Internal 10.1.1.5:5060 ↔ External 203.0.113.10:40000
│
├─ This binding persists across sessions
├─ Inactivity timeout: configurable (default 60s for UDP)
├─ Max sessions: configurable per binding
└─ Access mode: controls who can use the binding
```

### Access Modes

Three access modes control which external hosts can send traffic to the persistent mapping:

1. **target-host** — Only the original destination can send return traffic. Tightest security, may break some SIP scenarios with media servers.

2. **target-host-port** — Only the original destination IP AND port. Even tighter, rarely used.

3. **any-remote-host** — Any external host can send traffic to the mapping. Required for full-cone NAT behavior needed by SIP, gaming, and P2P. Least restrictive.

### SIP ALG vs Persistent NAT

The SRX has a SIP ALG that rewrites SIP headers (Contact, Via, SDP c=/m= lines) with translated addresses. In many deployments, the SIP ALG and persistent NAT work together:

- SIP ALG: rewrites signaling so endpoints learn correct addresses
- Persistent NAT: ensures the mapping is stable for incoming media and re-INVITEs

In some cases, the SIP ALG is disabled (it can mangle non-standard SIP extensions), and persistent NAT with any-remote-host mode is used alone. The SIP endpoints must then use STUN/TURN/ICE to discover their translated addresses.

---

## 5. NAT64 Implementation

### Architecture

NAT64 translates between IPv6-only clients and IPv4-only servers. The SRX acts as the translation gateway at the IPv6/IPv4 boundary.

```
IPv6 Client              SRX (NAT64)                IPv4 Server
10.1.1.5 (v6)            ┌─────────────┐            93.184.216.34
      │                   │ Static NAT64│
      │  dst: 64:ff9b::  │ ──────────→ │  dst: 93.184.216.34
      │  5db8:d822        │ src NAT64   │  src: 203.0.113.200
      │ ─────────────→    │ pool        │  ─────────────────→
      │                   └─────────────┘
```

### Components

1. **DNS64** — Synthesizes AAAA records from A records by prepending the well-known prefix `64:ff9b::/96` (or a custom prefix). When the IPv6 client queries for `example.com`, DNS64 returns `64:ff9b::5db8:d822` (where `5db8:d822` = `93.184.216.34`).

2. **Static NAT64 rule** — Maps the `64:ff9b::/96` prefix to `0.0.0.0/0`, telling the SRX to extract the embedded IPv4 address from the lower 32 bits of the IPv6 destination.

3. **Source NAT pool** — Provides an IPv4 source address for the translated packets. Without this, the IPv4 server would see an IPv6 source it cannot route to.

### Stateful Translation

NAT64 on the SRX is stateful (RFC 6146). It maintains session state identical to regular NAT — the translation is bidirectional per session, and return traffic is reverse-translated.

### Limitations

- The SRX DNS ALG must be enabled for DNS64 functionality
- DNSSEC validation may fail because DNS64 modifies DNS responses
- Protocols with embedded IP addresses (FTP active mode, SIP without ALG) may break
- IPv4 literals in application-layer protocols (e.g., URLs with `http://1.2.3.4/`) cannot be translated

---

## 6. Session-Based NAT

### Session Creation with NAT

When the first packet of a flow arrives and a NAT rule matches, the SRX creates a session entry containing:

```
Session entry:
├─ Wing 1 (forward flow):
│   ├─ Original src IP:port
│   ├─ Original dst IP:port
│   ├─ Translated src IP:port (if source NAT)
│   └─ Translated dst IP:port (if destination/static NAT)
│
├─ Wing 2 (reverse flow):
│   ├─ Src = translated dst of forward
│   ├─ Dst = translated src of forward
│   └─ Reverse translations applied
│
├─ NAT rule references
├─ Policy reference
├─ Timeout values
└─ Session flags
```

### Fast-Path Processing

After session creation, subsequent packets in the same flow match the session table directly. No NAT rule lookup, no policy lookup — translations are applied from the cached session entry. This is the "fast path" and is the primary performance optimization in flow-based processing.

### Session Timeout and NAT

NAT sessions inherit timeouts from the security policy or protocol defaults:

- TCP established: 1800 seconds (30 minutes)
- TCP initial: 20 seconds
- UDP: 60 seconds
- ICMP: 2 seconds

When a session expires, the NAT translation is released. For persistent NAT, the binding persists beyond individual session timeouts, up to the configured `inactivity-timeout`.

---

## 7. NAT Scaling Considerations

### Session Table Capacity

NAT does not have a separate session table — it uses the same session table as security policies. Session limits vary by SRX model:

```
Platform             Max Sessions     NAT Pool Capacity
SRX300               64K             ~60K concurrent
SRX340               256K            ~250K concurrent
SRX345               375K            ~370K concurrent
SRX1500              2M              ~2M concurrent
SRX4100              4M              ~4M concurrent
SRX4200              8M              ~8M concurrent
SRX5400              10M+            ~10M concurrent
SRX5600              20M+            ~20M concurrent
SRX5800              40M+            ~40M concurrent
```

### Port Exhaustion

With PAT, each translated source IP provides approximately 63,000 usable ports (1024-65535). For high-scale deployments:

- **Multiple pool addresses** — Each additional IP adds ~63K ports
- **Port block allocation** — Allocate blocks (e.g., 256 ports per block), reducing per-session overhead
- **Deterministic NAT** — Statically partition the port space among internal hosts, eliminating per-session logging

### Pool Utilization Monitoring

```
show security nat source pool all
show security nat resource-usage source pool all
```

Key metrics:
- **Port utilization** — percentage of allocated ports in use
- **Address utilization** — how many pool IPs have active mappings
- **Peak concurrent** — high-water mark for capacity planning

### Deterministic NAT Scaling

Deterministic NAT divides the port range evenly across all possible internal hosts. The calculation:

```
Ports per host = (port range) / (number of internal hosts per external IP)

Example:
- Port range: 1024-65535 = 64,512 ports
- Pool: 1 IP, internal subnet: /24 (256 hosts)
- Ports per host: 64,512 / 256 = 252 ports per host
- Block size: 252 (rounded to configured block-size)
```

Advantage: no per-session logging required. Given the source IP and timestamp, the external IP:port mapping is deterministic — computable from configuration alone. This is critical for regulatory compliance in carrier-grade NAT (CGN) deployments.

### Performance Impact

NAT itself adds minimal per-packet overhead since translations are cached in the session entry (fast-path). However:

- **Session creation** — NAT rule lookup adds latency to the first packet
- **ALG processing** — DNS ALG, SIP ALG, FTP ALG inspect and modify payloads (slower)
- **Persistent NAT** — Maintaining binding table consumes memory proportional to concurrent bindings
- **Logging** — Per-session NAT logging can overwhelm syslog infrastructure at scale

---

## 8. Troubleshooting NAT Issues

### Common Failure Modes

1. **Missing proxy-ARP** — NAT pool IPs on the egress subnet get no ARP response. External hosts cannot reach the NAT IP. Fix: configure proxy-arp.

2. **Policy mismatch** — Policy references pre-DNAT destination or post-SNAT source. Fix: remember the policy evaluation window.

3. **Rule ordering** — A more general rule matches before a specific rule. Fix: use `insert ... before` to reorder.

4. **Pool exhaustion** — All ports in the pool are consumed. Symptoms: new connections fail silently. Fix: add pool addresses, enable PBA, or configure overflow pool.

5. **ALG interference** — SIP ALG mangles non-standard headers. Fix: disable the specific ALG if not needed.

6. **Route asymmetry** — Return traffic enters a different interface/zone, session not found. Fix: ensure symmetric routing or use chassis cluster with session failover.

### Debug Commands

```
# Trace NAT processing
set security flow traceoptions file nat-debug
set security flow traceoptions flag basic-datapath
set security flow traceoptions packet-filter TRACE1 source-address 10.1.1.5

# Monitor real-time sessions
monitor security flow session

# Check specific NAT rule hits
show security nat source rule RULE-NAME
```

The traceoptions output shows the exact sequence: static NAT check, destination NAT check, route lookup, policy lookup, source NAT check — allowing you to pinpoint where in the pipeline the failure occurs.
