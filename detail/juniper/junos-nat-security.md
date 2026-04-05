# JunOS NAT Security — Policy Ordering, ALG Internals, HA Synchronization, and Evasion Techniques

> *NAT on SRX is not a simple address rewrite — it is deeply interleaved with the security policy engine, ALG deep packet inspection, and HA session synchronization. The ordering of DNAT before policy lookup and SNAT after creates a dual-phase translation model where misconfiguration leads to silent drops or unintended exposure. ALGs perform payload-level inspection that can be exploited or circumvented, and NAT state synchronization in HA introduces its own failure modes.*

---

## 1. NAT and Firewall Policy Ordering — The Dual-Phase Model

### SRX Packet Processing Pipeline

The SRX processes packets through a well-defined pipeline where NAT and security policy evaluation are interleaved:

```
Packet Ingress
  │
  ├─ 1. Ingress interface → determine ingress zone
  │
  ├─ 2. Screen processing (attack detection)
  │
  ├─ 3. Route lookup → determine egress interface and egress zone
  │
  ├─ 4. DESTINATION NAT evaluation
  │     └→ If DNAT rule matches: rewrite destination IP/port
  │        New destination determines actual egress zone (re-lookup if zone changes)
  │
  ├─ 5. Security policy lookup
  │     Match criteria: original-source + translated-destination + service
  │     └→ If deny: drop packet (no further processing)
  │     └→ If permit: continue
  │
  ├─ 6. SOURCE NAT evaluation
  │     └→ If SNAT rule matches: rewrite source IP/port
  │
  ├─ 7. ALG processing (if applicable, after session creation)
  │
  └─ 8. Forward packet
```

### Why DNAT Before Policy Matters

The decision to evaluate DNAT before the security policy is architecturally significant:

**Without pre-policy DNAT**: The security policy would need to reference the external (public) IP address as the destination. In environments with many DNAT mappings, this means security policies reference addresses that exist only in the NAT configuration, creating a semantic disconnect between "what the admin wants to protect" and "what address the policy references."

**With pre-policy DNAT**: The security policy references the actual internal server address. This is more intuitive — the admin writes a policy saying "allow HTTP to web-server-10.1.1.100" regardless of which external address maps to it. Multiple DNAT rules can map different external addresses to the same internal server, and a single policy covers all of them.

**The trap**: Administrators accustomed to Cisco ASA (where NAT and ACL are independent) often write SRX policies referencing the pre-NAT destination address. This results in no policy match, and the default deny drops the traffic silently.

### Policy Match Matrix

For a packet with original 5-tuple `{src=1.1.1.1, dst=203.0.113.10, proto=tcp, sport=50000, dport=80}` and DNAT rule translating `203.0.113.10:80 → 10.1.1.100:8080`:

| Field | Value Used in Policy Lookup |
|:---|:---|
| Source address | 1.1.1.1 (original, pre-SNAT) |
| Destination address | 10.1.1.100 (post-DNAT) |
| Source port | 50000 (original) |
| Destination port | 8080 (post-DNAT, if port translation occurred) |
| Application | HTTP (determined by AppID or port match) |
| Source zone | untrust (ingress interface zone) |
| Destination zone | trust (zone of 10.1.1.100's egress interface) |

If the DNAT changes the destination to an address in a different zone than the original destination, the zone pair for the policy lookup changes accordingly. This zone-shift behavior is unique to destination NAT and does not occur with source NAT.

---

## 2. ALG Deep Packet Inspection

### ALG Architecture

Application Layer Gateways operate within the SRX flow module, after session creation but integrated with NAT translation. The ALG framework:

```
Session Creation
  │
  ├─ Flow module identifies application (port-based or AppID)
  │
  ├─ ALG registered for this application?
  │   ├─ No → standard flow processing
  │   └─ Yes → ALG takes control of payload inspection
  │
  └─ ALG Processing:
      ├─ Parse application-layer payload
      ├─ Extract embedded IP addresses and ports
      ├─ Rewrite addresses/ports to match NAT translation
      ├─ Create "pinholes" (predicted sessions) for data channels
      └─ Update checksums
```

### SIP ALG Internals

SIP is the most complex ALG because SIP signaling contains multiple layers of embedded addressing:

**SIP Message Fields Requiring Rewrite:**

```
INVITE sip:user@10.1.1.100:5060 SIP/2.0          ← Request-URI
Via: SIP/2.0/UDP 10.1.1.50:5060                   ← Via header (response routing)
Contact: <sip:caller@10.1.1.50:5060>              ← Contact header (future requests)
Content-Type: application/sdp

v=0
o=- 12345 12345 IN IP4 10.1.1.50                  ← SDP origin
c=IN IP4 10.1.1.50                                ← SDP connection (media endpoint)
m=audio 20000 RTP/AVP 0                           ← SDP media (RTP port)
```

The ALG must rewrite **all** of these fields to use the NATted address. If any field is missed, the remote endpoint sends media or responses to the wrong address.

**SIP ALG Pinhole Creation:**

When the ALG parses the SDP body and sees `m=audio 20000 RTP/AVP 0`, it creates a predicted session (pinhole):

```
Pinhole: permit UDP from <remote-media-IP>:<any> to <NAT-addr>:<translated-20000>
         Timeout: 30 seconds (extended when RTP flows)

# Also creates RTCP pinhole: port = RTP port + 1 (20001)
```

The pinhole bypasses the normal security policy for the data channel — this is an inherent security risk because the ALG is effectively creating policy exceptions.

### FTP ALG Internals

FTP active mode requires the ALG to rewrite the PORT command:

```
Client (10.1.1.50, NATted to 203.0.113.50) sends:
  PORT 10,1,1,50,78,32     ← means: connect to 10.1.1.50:20000

ALG rewrites to:
  PORT 203,0,113,50,78,32  ← means: connect to 203.0.113.50:20000

ALG creates pinhole:
  permit TCP from <server>:<20> to 203.0.113.50:20000
```

FTP passive mode (PASV) requires rewriting the server's response:

```
Server (10.2.2.100, NATted to 198.51.100.100) responds:
  227 Entering Passive Mode (10,2,2,100,195,80)   ← 10.2.2.100:50000

ALG rewrites to:
  227 Entering Passive Mode (198,51,100,100,195,80) ← 198.51.100.100:50000
```

### ALG Security Implications

ALGs introduce several security concerns:

1. **Pinhole hijacking**: If an attacker can predict the pinhole parameters, they can inject traffic through the opened pinhole before the legitimate data channel is established.

2. **Payload manipulation**: If an attacker can inject crafted SIP or FTP commands, the ALG will dutifully create pinholes for attacker-controlled endpoints.

3. **Encrypted payloads**: ALGs cannot inspect encrypted payloads (SIP over TLS, FTPS implicit mode). The ALG silently fails, and the data channel is blocked by the security policy.

4. **Performance overhead**: ALGs perform per-packet payload inspection — significantly more expensive than standard flow processing. A SIP flood can overload the ALG processing path.

---

## 3. NAT State Synchronization in HA

### RTO Synchronization Protocol

In a chassis cluster, NAT state is synchronized via Real-Time Objects (RTOs) over the fabric link:

```
Active Node                          Standby Node
┌─────────────────┐                 ┌─────────────────┐
│ NAT Session     │  RTO sync       │ NAT Session     │
│ Table           │ ──────────────→ │ Table (replica) │
│                 │  (fabric link)  │                 │
│ Persistent NAT  │ ──────────────→ │ Persistent NAT  │
│ Table           │                 │ Table (replica) │
│                 │                 │                 │
│ ALG State       │ ──────────────→ │ ALG State       │
│ (pinholes)      │                 │ (pinholes)      │
└─────────────────┘                 └─────────────────┘
```

### What Gets Synchronized

| State | Synced? | Notes |
|:---|:---|:---|
| Source NAT session mapping | Yes | Full 5-tuple + translated tuple |
| Destination NAT session mapping | Yes | Full 5-tuple + translated tuple |
| Persistent NAT bindings | Yes | IP:port mappings survive failover |
| NAT pool allocation state | Yes | Port allocation counters |
| ALG pinholes | Yes | Predicted sessions for data channels |
| ALG parsing state | Partial | Mid-transaction state may be lost |
| NAT rule hit counters | No | Counters reset on new active node |
| Traceoptions state | No | Debug sessions not synced |

### Synchronization Timing

NAT session RTOs are synchronized when:
- A new session is created (SNAT/DNAT applied)
- A persistent NAT binding is created
- An ALG pinhole is opened
- A session is closed (binding released)

The synchronization is asynchronous — there is a small window (typically <100ms) where a failover could lose the most recently created sessions. For persistent NAT, the binding table is bulk-synchronized periodically in addition to incremental updates.

### Failure Modes

**Fabric link failure:**
```
If both control + data fabric links fail:
  → Split-brain condition
  → Both nodes become active
  → NAT sessions diverge
  → External hosts see duplicate translations (address conflicts)
  → Resolution: manual intervention, typically secondary node disabled

If only data fabric link fails:
  → Session sync degrades
  → New sessions still created on active node
  → Standby has stale session table
  → Failover results in existing sessions being dropped
```

**Pool exhaustion during failover:**

During failover, the new active node inherits the pool allocation state. However, if sessions were created in the synchronization gap, the port allocation bitmap may have small inconsistencies. The SRX resolves this by scanning the session table after becoming active and reconciling pool allocations.

---

## 4. NAT Security Implications and Evasion Techniques

### NAT as a Security Control — Limitations

NAT is often conflated with security, but it provides only:

1. **Address obscurity**: Internal topology is hidden from external observers
2. **Implicit ingress filtering**: Unsolicited inbound traffic has no NAT mapping and is dropped

NAT does **not** provide:
- Access control (that is the security policy's job)
- Payload inspection (that is IDP/ALG's job)
- Encryption or integrity (that is VPN's job)

### Evasion Techniques

**1. NAT slipstreaming (browser-based)**

An attacker tricks a browser behind NAT into making a request that causes the ALG to create a pinhole, allowing inbound access to an internal host:

```
Attack flow:
  1. Victim visits malicious page
  2. JavaScript crafts a SIP REGISTER or H.323 message embedded in HTTP POST
  3. If ALG inspects the payload (some ALGs inspect on non-standard ports),
     it creates a pinhole for the attacker's address
  4. Attacker connects through the pinhole to the victim

Mitigation:
  - Restrict ALG processing to standard ports only
  - Disable unnecessary ALGs
  - Use application-aware security policies
```

**2. IP fragment evasion**

NAT devices must reassemble fragments to translate embedded port numbers. If the NAT device does not fully reassemble before translating:

```
Attack flow:
  1. Attacker sends first fragment with IP header (src, dst) but no L4 header
  2. NAT translates the IP addresses
  3. Second fragment contains the L4 port numbers but may bypass port translation
  4. Destination reassembles a packet with untranslated port numbers

SRX mitigation:
  - SRX performs virtual reassembly before NAT processing
  - Fragments are buffered and reassembled in software
  - Reassembly timeout prevents fragment-based DoS
```

**3. ALG confusion attacks**

Sending malformed application-layer messages to confuse the ALG parser:

```
# Multi-line SIP headers to evade pattern matching
# Embedded CRLF in SIP body to inject fake SDP
# Oversized SIP messages exceeding ALG buffer

SRX mitigation:
  - ALG application screens (message-flood threshold, max message size)
  - Strict protocol compliance checking
  set security alg sip application-screen protect deny
```

---

## 5. Deterministic NAT for Logging Compliance

### The Compliance Problem

Regulations (PCI DSS, GDPR, law enforcement cooperation laws) require the ability to map a public IP:port observed externally back to a specific internal host at a given time. With dynamic PAT, this requires logging every NAT binding creation and deletion.

### Deterministic NAT Approach

Deterministic NAT pre-allocates port ranges to internal hosts, eliminating the need for per-session logging:

```
Algorithm:
  Given:
    - Internal subnet: 10.1.0.0/16 (65,536 hosts)
    - NAT pool: 203.0.113.0/24 (256 addresses)
    - Port range per host: (65536-1024) / (65536/256) = 252 ports per host

  Mapping:
    Internal host 10.1.0.1 → 203.0.113.0 ports 1024-1275
    Internal host 10.1.0.2 → 203.0.113.0 ports 1276-1527
    ...
    Internal host 10.1.1.1 → 203.0.113.1 ports 1024-1275
    ...

  To identify: "Who was 203.0.113.5:2048 at time T?"
    Address index: 5 → hosts 5*256 to 5*256+255 → 10.1.5.0 to 10.1.5.255
    Port offset: 2048-1024 = 1024 → 1024/252 = host index 4 → 10.1.5.4
```

### SRX Implementation

JunOS does not have a native "deterministic NAT" mode, but the behavior can be approximated:

```
# Use address-persistent to pin internal IPs to NAT pool addresses
set security nat source pool DET-POOL address 203.0.113.0/24
set security nat source pool DET-POOL address-persistent

# Combined with structured logging, this provides a mapping record
# address-persistent ensures same internal IP always maps to same external IP
# Port allocation is still dynamic within that IP, so per-session logging is still needed
# for full port-level tracing
```

For true deterministic NAT (port-level), carrier-grade NAT (CGNAT) solutions on MX series with deterministic NAT44 provide pre-computed port block allocation with no per-session logging requirement.

---

## 6. NAT Pool Exhaustion Analysis

### Port Allocation Mathematics

For source NAT with PAT:

```
Available ports per IP address:
  Total port range: 1024 to 65535 = 64,512 ports
  (Some implementations use 1024-65535, others 0-65535 minus well-known)

Maximum concurrent sessions per pool:
  S_max = N_addresses * P_ports_per_address
  S_max = N * 64512

Required pool size for target capacity:
  N = ceil(S_target / 64512)

Example: 500,000 concurrent sessions
  N = ceil(500000 / 64512) = ceil(7.75) = 8 IP addresses
```

### Exhaustion Detection

```
Pool utilization percentage:
  U = (active_sessions / S_max) * 100

Warning threshold (recommended): 80%
Critical threshold: 95%

At 100%: new sessions requiring this pool will fail
  - For source NAT: outbound connections silently fail
  - Existing sessions continue until they close
  - No ICMP or RST sent to the client — the session simply is not created
```

### Overflow Pool Strategy

```
Primary pool: 203.0.113.0/28 (16 addresses, ~1M sessions)
Overflow pool: interface (reth0.0 IP)

set security nat source pool PRIMARY address 203.0.113.0/32 to 203.0.113.15/32
set security nat source pool PRIMARY overflow-pool OVERFLOW
set security nat source pool OVERFLOW address 203.0.113.254/32

# When PRIMARY exhausts, new sessions use OVERFLOW
# Monitor: if overflow pool is being used, primary pool needs expansion
```

## Prerequisites

- Security zones and policies, IP routing, TCP/IP protocol suite, SRX platform architecture, chassis cluster fundamentals

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| NAT rule lookup | O(n) rules evaluated | O(rules) |
| Session table NAT lookup | O(1) hash | O(sessions) |
| ALG payload rewrite | O(payload_size) | O(1) per rewrite |
| Persistent NAT table lookup | O(1) hash | O(bindings) |
| RTO sync (per session) | O(1) per RTO | O(sessions) total |
| Pool port allocation | O(1) bitmap check | O(ports_per_IP) |

---

*NAT is a translation mechanism, not a security control. The security comes from the policy that permits or denies the translated flow. When NAT and policy are misaligned — referencing the wrong address in the wrong phase — the result is either silent drops that take hours to diagnose or unintended exposure that takes seconds to exploit. Master the dual-phase model, and NAT becomes predictable.*
