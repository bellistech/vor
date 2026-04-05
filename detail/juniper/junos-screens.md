# JunOS Screens — Packet Pipeline Positioning, SYN Flood Mechanisms, and Tuning Methodology

> *Screens are the SRX's first line of defense — they execute in the packet processing pipeline before session lookup, security policy evaluation, and application-layer inspection. This early-stage positioning means screens can drop attack traffic before it consumes session table entries, policy evaluation cycles, or ALG processing resources. The most critical screen — SYN flood protection — uses a two-phase mechanism transitioning from passive monitoring to active SYN proxy or SYN cookie defense. Screen tuning requires understanding baseline traffic patterns, attack profiles, and the performance trade-offs of each screen type.*

---

## 1. Screen Processing in the SRX Packet Pipeline

### Pipeline Position

Screens operate at two points in the SRX packet processing pipeline:

```
Packet arrives on ingress interface
  │
  ├─ 1. Layer 2 processing (VLAN, MAC)
  │
  ├─ 2. SCREEN PROCESSING — PHASE 1 (stateless, per-packet)
  │     ├─ IP header validation (bad options, source route, fragment checks)
  │     ├─ TCP flag validation (SYN-FIN, NULL, FIN-no-ACK)
  │     ├─ ICMP validation (ping-of-death, fragment, large)
  │     ├─ Land attack detection (src == dst)
  │     ├─ Tear drop detection (overlapping fragments)
  │     └─ IP spoofing check
  │
  │     If screen drops packet → STOP (no further processing)
  │
  ├─ 3. Route lookup → determine egress interface and zone
  │
  ├─ 4. SCREEN PROCESSING — PHASE 2 (stateful, rate-based)
  │     ├─ SYN flood detection (per-source, per-destination rate tracking)
  │     ├─ ICMP flood detection (per-destination rate tracking)
  │     ├─ UDP flood detection (per-destination rate tracking)
  │     ├─ Port scan detection (connection pattern analysis)
  │     ├─ IP sweep detection (connection pattern analysis)
  │     └─ Session limit enforcement (per-source, per-destination)
  │
  │     If screen drops packet → STOP
  │
  ├─ 5. Destination NAT (if applicable)
  │
  ├─ 6. Session lookup
  │     ├─ Existing session → fast-path forwarding
  │     └─ New session → continue to policy
  │
  ├─ 7. Security policy evaluation
  │
  ├─ 8. Source NAT (if applicable)
  │
  └─ 9. Forward packet
```

### Why Position Matters

The critical insight is that screens protect the session table and policy engine:

```
Without screens:
  DDoS: 1,000,000 SYN packets/sec arrive
  → Each SYN creates a half-open session in the session table
  → Session table fills (e.g., 4M entries on SRX4100)
  → Legitimate new connections cannot be created
  → All services behind the SRX become unavailable
  → Security policy engine processes 1M policy lookups/sec (CPU overload)

With screens:
  DDoS: 1,000,000 SYN packets/sec arrive
  → Screen: SYN flood threshold exceeded
  → SYN proxy activated: SRX absorbs SYN flood
  → Only legitimate completed TCP handshakes create sessions
  → Session table remains healthy
  → Policy engine processes only legitimate traffic
  → Services remain available
```

### Stateless vs Stateful Screens

```
Stateless screens (Phase 1):
  - Examine individual packets in isolation
  - No per-flow state maintained
  - Negligible CPU/memory overhead
  - Examples: SYN-FIN, land, tear-drop, source-route, bad-option
  - False positive rate: near zero (these are always invalid packets)

Stateful screens (Phase 2):
  - Track rates and patterns across multiple packets
  - Maintain per-source and per-destination counters
  - Moderate CPU/memory overhead (counter tables)
  - Examples: SYN flood, ICMP flood, UDP flood, port scan, IP sweep
  - False positive rate: depends on threshold tuning
```

---

## 2. SYN Flood Protection Mechanisms

### Two-Phase Defense

SRX SYN flood protection operates in two phases:

```
Phase 1: Monitoring (below attack-threshold)
  │
  ├─ SRX counts SYN packets per destination per second
  ├─ If rate < alarm-threshold: normal operation, no action
  ├─ If rate >= alarm-threshold but < attack-threshold:
  │   └─ Generate alarm (syslog, SNMP trap)
  │      Traffic still forwarded normally
  │
  └─ If rate >= attack-threshold:
      └─ Transition to Phase 2: Active Protection

Phase 2: Active Protection (SYN proxy or SYN cookie)
  │
  ├─ SRX intercepts all SYN packets to the target destination
  ├─ SRX responds with SYN-ACK on behalf of the server
  ├─ Client must complete the 3-way handshake with SRX
  │   ├─ If ACK received (legitimate client):
  │   │   └─ SRX establishes connection to real server
  │   │      Proxies the session (client ↔ SRX ↔ server)
  │   └─ If no ACK (attack traffic):
  │       └─ Half-open session times out on SRX (not on server)
  │          Server never sees the SYN
  │
  └─ Returns to Phase 1 when rate drops below threshold
```

### SYN Proxy vs SYN Cookie

The SRX uses two mechanisms depending on the platform and configuration:

```
SYN Proxy:
  1. Client sends SYN to server (via SRX)
  2. SRX intercepts, sends SYN-ACK to client with SRX-generated ISN
  3. Client sends ACK (completes handshake with SRX)
  4. SRX sends SYN to real server
  5. Server sends SYN-ACK to SRX
  6. SRX sends ACK to server
  7. SRX proxies all subsequent packets between client and server

  Cost: SRX maintains state for every SYN-ACK sent (before ACK received)
  Limit: bounded by SRX memory for half-open proxy sessions

  ┌────────┐          ┌────────┐          ┌────────┐
  │ Client │          │  SRX   │          │ Server │
  └───┬────┘          └───┬────┘          └───┬────┘
      │     SYN           │                   │
      │──────────────────►│                   │
      │                   │ (intercept)       │
      │     SYN-ACK       │                   │
      │◄──────────────────│                   │
      │                   │                   │
      │     ACK           │                   │
      │──────────────────►│                   │
      │                   │ (client valid)    │
      │                   │     SYN           │
      │                   │──────────────────►│
      │                   │     SYN-ACK       │
      │                   │◄──────────────────│
      │                   │     ACK           │
      │                   │──────────────────►│
      │                   │                   │
      │◄── proxied data ──┼── proxied data ──►│


SYN Cookie (stateless SYN-ACK):
  1. Client sends SYN to server (via SRX)
  2. SRX sends SYN-ACK with cryptographically computed ISN (cookie)
     - ISN encodes: timestamp, MSS, source IP hash
     - SRX does NOT maintain state for this half-open connection
  3. If client sends ACK:
     - SRX validates the ACK number against the cookie formula
     - If valid: connection is legitimate, proceed to server
     - If invalid: drop (spoofed or replayed)

  Cost: NO per-connection state during the SYN flood
  Limit: bounded only by packet processing rate (very high)
  Drawback: some TCP options (window scaling, SACK) may be lost
            because they cannot be encoded in the cookie
```

### Threshold Tuning

```
Threshold relationships:

  alarm-threshold < attack-threshold

  alarm-threshold:
    "Something unusual is happening"
    Action: generate alarm, continue normal forwarding
    Recommendation: set to 2x normal peak SYN rate

  attack-threshold:
    "We are under attack, activate protection"
    Action: SYN proxy/cookie activated
    Recommendation: set to 5x normal peak SYN rate
    Too low: legitimate traffic triggers proxy (adds latency)
    Too high: real attacks are not mitigated in time

  source-threshold:
    Per-source SYN rate limit
    Action: drop SYNs from sources exceeding this rate
    Recommendation: 20-100 (a single host rarely needs more than 100 SYN/sec)
    Use case: block single-source SYN floods

  destination-threshold:
    Per-destination SYN rate limit
    Action: trigger protection for specific destination when exceeded
    Recommendation: server-specific (web server may handle 10,000 SYN/sec,
                    SMTP server may handle 100)

  timeout:
    Seconds to wait for TCP handshake completion
    Action: half-open sessions older than timeout are purged
    Recommendation: 20 seconds (default)
    Lower values: more aggressive cleanup, may drop slow clients
    Higher values: more memory consumed by half-open sessions
```

---

## 3. Reconnaissance Detection Algorithms

### Port Scan Detection

The SRX detects port scans by tracking the interval between connection attempts from a single source to different ports on a single destination:

```
Algorithm:
  Maintain per-source tracking table:
    source_ip → {last_port_time, ports_scanned, destination_ip}

  On each new SYN from source S to destination D port P:
    interval = current_time - last_port_time[S][D]
    if interval < threshold (microseconds):
      scan_count[S][D] += 1
      if scan_count[S][D] > alert_count:
        ACTION: drop packet, generate alarm
        BLOCK: all subsequent packets from S for block_timeout
    last_port_time[S][D] = current_time

  Detection sensitivity:
    threshold = 5000 µs (5 ms): detects aggressive scans (nmap -T4, masscan)
    threshold = 10000 µs (10 ms): detects moderate scans (nmap -T3)
    threshold = 50000 µs (50 ms): detects slow scans (nmap -T2)
    threshold = 1000000 µs (1 sec): detects very slow scans (high false positive risk)
```

### IP Sweep Detection

```
Algorithm:
  Maintain per-source tracking table:
    source_ip → {last_host_time, hosts_swept, port}

  On each ICMP echo-request (or TCP SYN to same port) from source S:
    if destination != last_destination[S]:
      interval = current_time - last_host_time[S]
      if interval < threshold:
        sweep_count[S] += 1
        if sweep_count[S] > alert_count:
          ACTION: drop, generate alarm
      last_host_time[S] = current_time
      last_destination[S] = current_destination

  Detection sensitivity:
    Same threshold logic as port scan
    Low threshold: detects fast sweeps (nmap -sn -T4)
    High threshold: detects slow sweeps (more false positives from legitimate monitoring)
```

### Evasion of Reconnaissance Detection

```
Attacker evasion techniques:

1. Slow scanning (below threshold):
   nmap -T1 or custom timing: 1 port per 15-60 seconds
   Detection: requires very high threshold → many false positives
   Mitigation: combine with IDP signatures for nmap fingerprinting

2. Distributed scanning (many sources):
   Multiple compromised hosts each scan a few ports
   No single source exceeds threshold
   Mitigation: per-destination session limits + SIEM correlation

3. Fragmented scanning:
   SYN in fragments to evade TCP flag analysis
   Mitigation: syn-frag screen drops fragmented SYNs

4. Idle scanning (using zombie host):
   Attacker uses third-party host's predictable IPID to scan
   Source IP is the zombie, not the attacker
   Mitigation: ip-spoofing screen + uRPF
```

---

## 4. Screen vs IDP Comparison

### Functional Overlap and Differences

| Feature | Screens | IDP (Intrusion Detection/Prevention) |
|:---|:---|:---|
| Processing position | Before session lookup | After session creation |
| Inspection depth | L3/L4 headers only | L3-L7 (including payload) |
| State | Minimal (rate counters) | Full session tracking |
| Signature-based | No | Yes (thousands of signatures) |
| Anomaly-based | Basic (rate thresholds) | Advanced (protocol anomaly) |
| Performance impact | Minimal | Moderate to high |
| DDoS protection | Yes (SYN flood, rate limits) | Limited (resource-intensive) |
| Exploit detection | No | Yes (buffer overflow, SQLi, XSS) |
| Custom signatures | No | Yes |
| False positive rate | Very low (stateless checks) | Moderate (requires tuning) |
| Evasion resistance | Low (simple pattern checks) | High (reassembly, normalization) |

### Complementary Deployment

```
Defense in depth — screens and IDP work together:

  Screen layer (first):
    ├─ Drop invalid packets (SYN-FIN, land, tear-drop)
    ├─ Rate-limit flood traffic (SYN flood, ICMP flood)
    ├─ Block reconnaissance (port scan, IP sweep)
    └─ Enforce session limits

  IDP layer (second, on permitted traffic only):
    ├─ Detect application-layer exploits (SQLi, XSS, buffer overflow)
    ├─ Protocol anomaly detection (malformed HTTP, DNS tunneling)
    ├─ Known vulnerability signatures (CVE-based rules)
    └─ Custom signatures for organization-specific threats

  Benefit of layering:
    Screens reduce the volume of traffic IDP must inspect
    IDP processing is expensive — 10x more CPU per packet than screens
    Without screens: DDoS can overwhelm IDP engine
    With screens: DDoS is absorbed, IDP inspects only legitimate traffic
```

---

## 5. Screen Tuning Methodology

### Baseline Establishment

```
Step 1: Deploy screens in monitor-only mode (alarm without drop)
  - Set all thresholds very high (unlikely to trigger)
  - Monitor screen statistics for 1-2 weeks
  - Collect baseline data:

    show security screen statistics zone untrust

    Baseline metrics to record:
    - Peak ICMP packets/sec (normal)
    - Peak SYN packets/sec (normal and during busy periods)
    - Peak UDP packets/sec (normal)
    - Average sessions per source IP
    - Average sessions per destination IP
    - Port scan detection events (false positive rate)

Step 2: Calculate initial thresholds
  alarm_threshold = baseline_peak * 2
  attack_threshold = baseline_peak * 5
  source_threshold = max_per_source_baseline * 3
  destination_threshold = max_per_destination_baseline * 2

Step 3: Deploy with calculated thresholds
  - Monitor for false positives (legitimate traffic dropped)
  - Monitor for false negatives (attacks not detected)
  - Adjust thresholds based on operational experience

Step 4: Iterative refinement
  - Review screen statistics weekly
  - Adjust thresholds based on:
    - New services (web server launch may increase legitimate SYN rate)
    - New threats (DDoS campaign may require lower thresholds)
    - Business changes (seasonal traffic patterns)
```

### Per-Service Tuning

```
Different services need different thresholds:

Web server (high traffic):
  destination-threshold: 10000-50000 SYN/sec
  session-limit destination: 50000-100000

SMTP server (moderate traffic):
  destination-threshold: 100-500 SYN/sec
  session-limit destination: 5000-10000

DNS server (burst traffic):
  UDP flood threshold: 10000-50000 packets/sec
  session-limit destination: 100000

Internal file server (low traffic):
  destination-threshold: 50-100 SYN/sec
  session-limit destination: 1000

Approach: use multiple zones with different screen profiles
  - DMZ zone: tuned for web/mail server traffic patterns
  - Server zone: tuned for internal server patterns
  - User zone: tuned for end-user workstation patterns
```

### Monitoring and Alerting

```
Key screen metrics to monitor:

  show security screen statistics zone untrust

  Critical counters:
  - TCP SYN flood: high count = active SYN flood or misconfigured threshold
  - Source session limit: identifies specific abusive sources
  - Destination session limit: identifies targeted services
  - Port scan: indicates reconnaissance activity
  - ICMP flood: common DDoS vector

  SNMP traps for screen events:
  set security screen ids-option PERIMETER-SCREEN alarm-without-drop
  # During tuning: alarm but don't drop (monitor false positives)

  After tuning:
  # Remove alarm-without-drop (or don't set it) — screens will drop
  set security screen ids-option PERIMETER-SCREEN tcp syn-flood alarm-threshold 1000
  # Alarm at 1000 SYN/sec, drop at attack-threshold
```

---

## 6. Screen Performance Impact

### Processing Cost

```
Screen type cost analysis (relative CPU overhead per packet):

  Stateless screens (constant cost, negligible impact):
    SYN-FIN check:       ~1 CPU cycle (bit mask comparison)
    Land attack check:   ~2 CPU cycles (address comparison)
    Source route check:   ~1 CPU cycle (option field check)
    Bad option check:     ~3 CPU cycles (option parsing)
    Tear drop check:     ~5 CPU cycles (fragment offset comparison)
    Total stateless:     ~12 CPU cycles per packet

  Rate-based screens (requires counter table lookup):
    SYN flood tracking:  ~20 CPU cycles (hash table lookup + counter update)
    ICMP flood tracking: ~15 CPU cycles (counter lookup + threshold check)
    UDP flood tracking:  ~15 CPU cycles (counter lookup + threshold check)
    Session limit check: ~10 CPU cycles (counter lookup)

  Pattern-based screens (requires state tracking):
    Port scan detection: ~50 CPU cycles (state table lookup + interval calculation)
    IP sweep detection:  ~50 CPU cycles (state table lookup + interval calculation)

  For comparison:
    Session table lookup: ~100-200 CPU cycles
    Security policy lookup: ~200-500 CPU cycles
    IDP signature matching: ~1000-5000 CPU cycles per packet
    SSL decryption: ~10000-50000 CPU cycles per packet

Conclusion: screens add < 5% CPU overhead relative to the full pipeline.
           The protection they provide far outweighs the cost.
```

### Memory Requirements

```
Screen state memory consumption:

  Rate counters:
    Per-destination flood counter: ~32 bytes
    Per-source flood counter: ~32 bytes
    Maximum tracked entries: platform-dependent (typically 100K-1M)

  Reconnaissance detection:
    Per-source scan state: ~64 bytes
    Per-source sweep state: ~64 bytes
    Maximum tracked sources: platform-dependent (typically 50K-500K)

  SYN proxy state (during active protection):
    Per half-open connection: ~128 bytes
    Maximum half-open: platform-dependent
      SRX300: 10,000
      SRX1500: 100,000
      SRX4100: 500,000
      SRX4600: 1,000,000
      SRX5800: 5,000,000

  SYN cookie: NO per-connection state (stateless by design)
    Only CPU cost (cookie computation per SYN-ACK)
    Preferred for extreme SYN flood volumes
```

### Platform Capacity Under Attack

```
SRX screen processing capacity (approximate):

  Platform    | Max SYN flood (with proxy) | Max SYN flood (with cookie)
  SRX300      | 10,000 SYN/sec             | 50,000 SYN/sec
  SRX345      | 20,000 SYN/sec             | 100,000 SYN/sec
  SRX1500     | 100,000 SYN/sec            | 500,000 SYN/sec
  SRX4100     | 500,000 SYN/sec            | 2,000,000 SYN/sec
  SRX4600     | 1,000,000 SYN/sec          | 5,000,000 SYN/sec
  SRX5400     | 2,000,000 SYN/sec          | 10,000,000 SYN/sec
  SRX5800     | 5,000,000 SYN/sec          | 20,000,000 SYN/sec

  Beyond these rates: upstream mitigation required (ISP, CDN, scrubbing center)
  SRX is a firewall, not a DDoS scrubber — screens provide first-line defense,
  not volumetric DDoS absorption.
```

## Prerequisites

- TCP/IP protocol suite, TCP three-way handshake, IP fragmentation, ICMP message types, SRX security zones, security policies

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Stateless screen check (per packet) | O(1) | O(1) |
| Rate counter lookup | O(1) hash | O(tracked_entries) |
| SYN proxy session create | O(1) | O(half_open_sessions) |
| SYN cookie generate | O(1) | O(1) — stateless |
| Port scan state lookup | O(1) hash | O(tracked_sources) |
| Session limit check | O(1) hash | O(tracked_entries) |

---

*Screens are the bouncer at the door — they check IDs before anyone gets inside. The beauty of screens is their simplicity: a SYN-FIN packet is never valid, a land attack is never legitimate, and a source-routed packet from the Internet has no business existing. These checks cost almost nothing and prevent attacks that have been known for decades but are still attempted daily. The hard part is tuning the rate-based screens — too aggressive and you block your own users, too permissive and you let the flood in. Baseline first, tune second, monitor always.*
