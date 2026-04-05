# JunOS SRX Platform — Packet Processing Pipeline, Session Architecture, and Chassis Cluster Internals

> *The SRX is a session-oriented security platform where every packet is either creating a new session or matching an existing one. Understanding the flow-based processing pipeline, the security policy evaluation order, and the chassis cluster failover mechanics is foundational to JNCIE-SEC design and troubleshooting.*

---

## 1. SRX Packet Processing Pipeline

### Flow-Based Forwarding (Default Mode)

Flow-based forwarding is the SRX's primary mode. Every packet is evaluated against the session table. The pipeline has two paths:

**First Packet Path (slow path):**

```
Packet arrives on ingress interface
│
├─ 1. Screen checks (zone-level IDS)
│     └─ SYN flood, land attack, tear-drop, ping-of-death, etc.
│     └─ If screen matches → drop (before any session/policy processing)
│
├─ 2. Session lookup
│     └─ No existing session → enter first-packet processing
│
├─ 3. Static NAT check
│     └─ If match: translate destination
│
├─ 4. Destination NAT check
│     └─ If match: translate destination
│
├─ 5. Route lookup (FIB — uses translated destination)
│     └─ Determines egress interface → egress zone
│
├─ 6. Security policy lookup
│     ├─ Zone pair: ingress zone → egress zone
│     ├─ Source: original IP (pre-SNAT)
│     ├─ Destination: translated IP (post-DNAT/static)
│     ├─ Application: initial identification
│     ├─ If deny/reject → drop/reject, no session created
│     └─ If permit → continue
│
├─ 7. Application services (if policy enables them)
│     ├─ UTM (antivirus, web filter, anti-spam, content filter)
│     ├─ IDP (signature matching)
│     ├─ AppFW (application-level control)
│     └─ SSL proxy (decrypt for inspection)
│
├─ 8. Source NAT check
│     └─ If match: translate source
│
├─ 9. Session creation
│     ├─ Forward wing: original → translated
│     ├─ Reverse wing: translated → original
│     ├─ Policy reference, NAT bindings, timeouts
│     └─ Session installed in session table
│
└─ 10. Packet forwarded to egress interface
```

**Subsequent Packet Path (fast path):**

```
Packet arrives on ingress interface
│
├─ 1. Screen checks
│
├─ 2. Session lookup
│     └─ Match found → fast path
│
├─ 3. Apply cached translations (NAT)
│
├─ 4. Apply cached policy actions
│     └─ If application services enabled, may inspect payload
│
└─ 5. Forward to egress interface
```

The fast path bypasses route lookup, policy lookup, and NAT rule evaluation entirely. All information is cached in the session entry. This is why the SRX can process millions of packets per second for established sessions while the first-packet path is significantly slower.

### Packet-Based Forwarding

Packet-based mode disables all session tracking and security services. The SRX behaves as a stateless router:

```
Packet arrives → Route lookup → Forward
```

No sessions, no NAT, no security policies, no UTM/IDP. Each packet is independently routed. Use cases:

- MPLS transit segments where the SRX is a label-switching router
- IPv6 transit where flow processing is not needed
- Extreme performance requirements where stateless forwarding suffices

Packet mode can be enabled globally or per-interface. Per-interface packet mode requires a reboot.

---

## 2. Session Setup and Fast-Path Architecture

### Session Table Structure

Each session entry contains:

```
Session ID: unique 32-bit identifier
│
├─ Wing 1 (forward flow):
│   ├─ Source IP:port (original)
│   ├─ Destination IP:port (original)
│   ├─ Source IP:port (translated, if SNAT)
│   ├─ Destination IP:port (translated, if DNAT)
│   ├─ Ingress interface and zone
│   └─ Protocol, DSCP, flags
│
├─ Wing 2 (reverse flow):
│   ├─ Mirrors the translation of wing 1
│   └─ Egress interface and zone
│
├─ Policy reference (which policy permitted this session)
├─ NAT rule references
├─ Application identification state
├─ Application services state (UTM/IDP processing context)
├─ Timeout values (TCP/UDP/ICMP-specific)
├─ Byte and packet counters
└─ Session flags (syn-seen, established, closing, etc.)
```

### Session Matching

Session lookup uses a hash of the 5-tuple (src-ip, dst-ip, src-port, dst-port, protocol). Both directions are checked — a packet matching wing 1 is a forward packet; a packet matching wing 2 is a return packet.

### TCP State Machine

For TCP sessions, the SRX tracks connection state:

```
SYN received     → session created (initial timeout: 20s)
SYN-ACK seen     → half-open
ACK seen          → established (timeout: 1800s)
FIN/FIN-ACK      → closing (timeout: varies)
RST seen          → session torn down
Timeout           → session expired and removed
```

The `no-syn-check` option relaxes this — allows mid-stream sessions (critical for HA failover where the SYN was on the old primary).

### UDP and ICMP Sessions

UDP sessions are created on the first packet and age out based on inactivity (default 60 seconds). ICMP sessions match on type/code/ID and age out quickly (default 2 seconds).

---

## 3. Security Policy Evaluation Order

### Complete Evaluation Sequence

```
1. Intra-zone traffic check
│   └─ Default: deny (unlike some firewalls that default-allow intra-zone)
│
2. Zone-pair policies (from-zone X to-zone Y)
│   ├─ Evaluated top to bottom within the zone pair
│   ├─ First match wins
│   └─ If no match → fall through to global policies
│
3. Global policies
│   ├─ Evaluated top to bottom
│   ├─ First match wins
│   └─ If no match → implicit default policy
│
4. Default policy
│   └─ Configurable: permit-all or deny-all (factory default: deny-all)
```

### Policy Match Criteria

Each policy matches on:

```
Match fields:
├─ Source zone (implicit from "from-zone")
├─ Destination zone (implicit from "to-zone")
├─ Source address (address book entry or "any")
├─ Destination address (address book entry or "any")
├─ Application (predefined, custom, or "any")
├─ Source identity (user-based — JIMS, Active Directory)
├─ Dynamic application (AppID result — for AppFW)
└─ URL category (for URL-aware policies)
```

### Application Identification and Policy Re-evaluation

When AppID identifies the application (which may take several packets), the SRX re-evaluates the policy with the identified application. If the updated match changes the policy result:

- If the new policy denies traffic, the session is torn down
- If the new policy permits with different services, the services are updated

This re-evaluation is why you see initial packets "leak" through before AppFW blocks — the application is not yet identified.

---

## 4. Chassis Cluster Internals

### Link Types

A chassis cluster uses three link types:

```
Control Link (fxp1)
├─ Carries: heartbeats, configuration sync, RE-to-RE communication
├─ Bandwidth: low (1 Gbps typically sufficient)
├─ Failure: secondary node reboots (cannot risk split-brain)
└─ Dedicated interface, cannot be shared

Fabric Link (fab0/fab1)
├─ Carries: data traffic (transit packets when ingress/egress on different nodes)
├─ Bandwidth: must handle peak transit load
├─ Failure: no cross-node forwarding, RG failover may trigger
└─ Typically 10G or LAG for high-traffic clusters

Redundant Ethernet (reth)
├─ Virtual interface spanning both nodes
├─ Active member on primary node, standby on secondary
├─ Failover: secondary member becomes active, GARP sent
└─ Tied to a redundancy group (RG)
```

### Redundancy Groups (RG)

```
RG0 — Control plane (RE)
├─ Determines which node runs the primary RE
├─ Config sync: primary RE pushes config to secondary
├─ Only one RG0 — cannot split the control plane
└─ Failover: secondary RE takes over, possible service disruption

RG1+ — Data plane (traffic)
├─ Each RG owns a set of reth interfaces
├─ Can have different primary nodes (active/active forwarding)
├─ RG1 primary on node 0, RG2 primary on node 1 → active/active
└─ Failover: independent per RG based on priority + monitoring
```

### Failover Triggers

```
Automatic failover:
├─ Interface monitoring: interface down → weight subtracted from priority
│   └─ When priority drops to 0 or below threshold → failover
├─ IP monitoring: ping target unreachable → weight subtracted
│   └─ Detects upstream/downstream failures
├─ Control link failure: secondary reboots (hardcoded behavior)
├─ SPU failure: SPU-based RG failover
└─ Manual: request chassis cluster failover redundancy-group 1

Preemption:
├─ When enabled: original primary takes back RG when it recovers
├─ When disabled (default): secondary keeps the RG until manual failover
└─ Risk: preemption causes double-failover (disruption on failure + recovery)
```

### Session Synchronization

Sessions are synchronized from the primary node to the secondary over the fabric link (or control link as backup). The sync covers:

- Session table entries (both wings)
- NAT translations
- IPsec SA state (for VPN failover)
- Application services state (limited — UTM/IDP state is not synced)

After failover:

- TCP sessions continue if `no-syn-check` and `no-sequence-check` are enabled
- UDP sessions continue transparently
- IPsec tunnels re-negotiate or use synced SA state
- UTM/IDP inspections restart from scratch (no stateful sync)
- Gratuitous ARP (GARP) updates upstream switches about new MAC-to-IP mapping

---

## 5. SRX Scaling

### Session Table Scaling

```
Platform          Max Sessions    CPS (connections/sec)    FW Throughput
SRX300            64K             5K                       1 Gbps
SRX320            64K             5K                       1 Gbps
SRX340            256K            12K                      3 Gbps
SRX345            375K            15K                      5 Gbps
SRX380            380K            28K                      20 Gbps
SRX1500           2M              50K                      9 Gbps
SRX4100           4M              200K                     40 Gbps
SRX4200           8M              300K                     80 Gbps
SRX4600           10M             400K                     120 Gbps
SRX5400           10M+            350K                     100 Gbps
SRX5600           20M+            500K                     220 Gbps
SRX5800           40M+            600K                     350 Gbps
```

### Policy Scaling

- Maximum policies: varies by model (4K–100K+)
- Policy lookup is optimized with hash tables, not linear scan
- Large policy sets: use address sets and application sets to reduce rule count
- Policy compilation happens on commit — large policy sets increase commit time

### Throughput Considerations

Throughput depends on packet size and enabled features:

```
Feature               Throughput Impact
Firewall only         Baseline (IMIX)
+ NAT                 ~90% of baseline
+ IPsec               ~30-60% of baseline (depends on cipher)
+ UTM (AV stream)     ~20-40% of baseline
+ IDP                 ~30-50% of baseline
+ SSL proxy           ~10-20% of baseline
All features          ~5-15% of baseline
```

---

## 6. SRX vs Cisco FTD Comparison

```
Feature                   SRX Series              Cisco FTD (Firepower)
─────────────────────────────────────────────────────────────────────────
OS                        JunOS (FreeBSD-based)    FTD (Linux-based, Snort)
CLI                       Hierarchical (set/show)  FMC GUI + FlexConfig CLI
Management                CLI / J-Web / Junos      FMC (Firepower Management
                          Space / Security Dir.    Center) — GUI-primary
Config model              Candidate + commit       Deploy from FMC to device
Rollback                  rollback 0-49 (instant)  Limited (FMC snapshots)
HA                        Chassis cluster          Failover pair (A/S, A/A)
                          (active/active per RG)   (limited A/A support)
Session sync              Full (fabric link)       Full (failover link)
VPN                       Route-based (st0) or     Route-based (VTI) or
                          policy-based             policy-based
NAT                       Rule-set based           NAT rule/policy
                          (from/to zone context)   (auto-NAT + manual-NAT)
IPS engine                IDP (Juniper sigs)       Snort (Snort rules + VDB)
Application ID            AppID/AppSecure          OpenAppID / Snort AppID
URL filtering             Enhanced WF (Websense)   URL filtering (Talos)
Packet mode               Yes (per-interface)      Not available
Automation                NETCONF/YANG, PyEZ,      REST API, FMC API,
                          REST API                 Ansible modules
Multi-tenancy             Logical systems, LSYS    Multi-domain in FMC
Troubleshooting           Extensive CLI traceopts   Mostly via FMC/CLI debug
                          (per-packet trace)       (less granular)
```

Key architectural differences:

1. **Commit model** — JunOS uses candidate configuration with atomic commit and rollback. FTD deploys from centralized FMC with no instant rollback. This makes JunOS significantly easier to troubleshoot and recover from misconfigurations.

2. **HA architecture** — SRX chassis cluster supports per-RG active/active (different RGs primary on different nodes). FTD failover is simpler but less flexible.

3. **Packet mode** — SRX can selectively bypass flow processing per-interface for pure routing segments. FTD has no equivalent — all traffic goes through the inspection engine.

4. **Management model** — SRX is CLI-first with optional GUI. FTD is GUI-first (FMC) with limited direct CLI access. For large-scale automation, both support APIs, but JunOS NETCONF/YANG is more mature.
