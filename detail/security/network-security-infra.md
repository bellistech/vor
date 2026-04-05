# Network Security Infrastructure — Defense-in-Depth Architecture and Theory

> *Network infrastructure devices — routers, switches, firewalls — are the foundation of every communication system. When an attacker compromises a core router, they do not merely access one system; they gain the ability to observe, modify, or disrupt every packet traversing that device. Infrastructure security therefore demands a layered defense model that addresses the three distinct processing domains of every network device: the data plane (forwarding engine), the control plane (routing protocol processor), and the management plane (administrative access). This document provides the theoretical foundation for infrastructure ACLs, unicast reverse path forwarding, routing protocol authentication, management plane hardening, and the mapping of these controls to industry frameworks such as the CIS Benchmarks and NIST 800-53.*

---

## 1. The Three-Plane Model and Attack Surface Analysis

### 1.1 Data Plane

The data plane (forwarding plane) handles the transit of user traffic through the device. On modern platforms, this is implemented entirely in hardware (ASICs, NPUs, TCAMs) and operates at line rate — typically terabits per second. The data plane performs destination lookup, packet rewrite (TTL decrement, MAC rewrite), QoS classification, ACL filtering, and output queuing.

**Attack surface:**

The data plane itself is relatively robust because ASIC processing is deterministic and does not involve general-purpose code execution. However, the data plane creates indirect attack vectors:

- **Spoofed source addresses:** Packets with forged source IPs can bypass ACLs, trigger false uRPF drops on return traffic, and enable reflection/amplification attacks. The data plane carries these packets at line rate if no ingress filtering is applied.

- **Packet-of-death vulnerabilities:** Certain malformed packets can cause ASIC exceptions that punt to the CPU, effectively converting a data plane attack into a control plane attack. Historical examples include crafted MPLS labels, IPv6 extension header chains, and GRE packets with invalid flags.

- **Bandwidth exhaustion:** Volumetric DDoS floods consume data plane capacity. While the ASIC handles the throughput, legitimate traffic is crowded out. ACLs and uRPF can filter some attack traffic in hardware.

- **Reconnaissance:** Attackers probe infrastructure addressing via traceroute (TTL expiry generates ICMP from the router CPU) and direct probes to router interfaces. iACLs deny these probes at the edge.

### 1.2 Control Plane

The control plane runs on the general-purpose CPU and manages the state machines that build the forwarding table. This includes routing protocol adjacencies (BGP, OSPF, IS-IS, EIGRP), label distribution (LDP, RSVP-TE), spanning tree (STP/RSTP/MST), and first-hop redundancy (VRRP, HSRP, GLBP).

**Attack surface:**

The control plane is the most critical and most vulnerable target:

- **Protocol manipulation:** An attacker who can inject forged routing protocol messages can redirect traffic through attacker-controlled paths, create routing loops, or withdraw routes causing black holes. Without authentication, routing protocols accept messages from any source that speaks the correct protocol.

- **Adjacency disruption:** TCP RST attacks against BGP sessions cause route withdrawals. OSPF hello spoofing can trigger adjacency flaps. IS-IS CSNP forgery can corrupt link-state databases. These attacks do not require compromising the router — only the ability to send crafted packets to the router's IP.

- **Resource exhaustion:** Flooding the CPU with exception packets (TTL expired, IP options, ARP glean) starves routing protocol processes. BGP keepalive timers expire, sessions tear down, and routes withdraw — even though the router hardware is functional.

- **BFD exploitation:** BFD operates on aggressive timers (often 50-300ms). If an attacker can delay or drop BFD packets for even a few hundred milliseconds, BFD declares the link down, triggering fast reroute. CoPP must prioritize BFD at high rates.

### 1.3 Management Plane

The management plane provides administrative access for configuration, monitoring, and firmware management. Protocols include SSH, SNMP, NETCONF, gRPC, NTP, RADIUS/TACACS+, syslog, and HTTPS.

**Attack surface:**

- **Credential attacks:** SSH brute force, SNMP community string guessing, TACACS+ dictionary attacks. Each failed attempt consumes CPU cycles.

- **Protocol vulnerabilities:** SNMP v1/v2c sends community strings in plaintext; NTP monlist provides amplification factors of 500x; HTTP admin interfaces may have XSS or CSRF vulnerabilities.

- **Lateral movement:** Once an attacker gains management access to one device (via stolen credentials or exploited vulnerability), they can reconfigure routing, disable logging, install persistent backdoors, or pivot to other managed devices.

- **Supply chain:** Firmware images, configuration backups, and TFTP/SCP transfers can be intercepted on unencrypted management channels.

---

## 2. Unicast Reverse Path Forwarding (uRPF)

### 2.1 The Source Address Validation Problem

IP was designed without source address authentication. Any host can place any IP address in the source field of a packet, and routers will forward it based solely on the destination address. This enables:

- **Direct attacks with spoofed source:** The attacker sends packets with a spoofed source, making the victim respond to an innocent third party or making attack attribution impossible.

- **Reflection/amplification:** The attacker sends requests to amplifiers (DNS, NTP, memcached) with the victim's IP as source. Amplifiers respond to the victim with responses much larger than the queries.

- **TCP RST/SYN attacks:** Spoofed source addresses allow blind TCP injection attacks against established sessions (e.g., BGP).

### 2.2 uRPF Algorithm

uRPF validates the source address of incoming packets by checking whether the FIB contains a route back to the source address and, in strict mode, whether that route points to the interface the packet arrived on.

**Strict mode algorithm:**

```
For each incoming packet on interface I with source address S:
  1. Look up S in the FIB
  2. If no FIB entry exists for S → DROP (no route = bogon)
  3. If FIB entry exists, get the set of next-hop interfaces N = {n1, n2, ...}
  4. If I is in N → ACCEPT (packet arrived on expected interface)
  5. If I is NOT in N → DROP (asymmetric path or spoofed)
```

Strict mode requires symmetric routing. If the path from A to B traverses interface eth0, then the path from B to A must also traverse eth0. This holds true for single-homed customer connections but fails in networks with asymmetric routing (multiple upstreams, hot-potato routing).

**Loose mode algorithm:**

```
For each incoming packet on interface I with source address S:
  1. Look up S in the FIB
  2. If no FIB entry exists for S → DROP
  3. If FIB entry exists (via ANY interface) → ACCEPT
  4. Special: if only a default route matches, behavior depends on allow-default setting
```

Loose mode only validates that the source address is routable — it exists somewhere in the routing table. This catches bogon addresses (RFC 1918 from the Internet, unallocated space, martians) but does not catch spoofing of legitimate addresses.

**Feasible-path mode algorithm:**

```
For each incoming packet on interface I with source address S:
  1. Look up S in the FIB, including ALL feasible paths (ECMP, backup routes, IGP alternatives)
  2. If no FIB entry exists for S → DROP
  3. Get the set of ALL feasible next-hop interfaces F = {f1, f2, ...} (not just best path)
  4. If I is in F → ACCEPT
  5. If I is NOT in F → DROP
```

Feasible-path mode (RFC 3704, Section 2.4) extends strict mode by considering not just the best path but all alternative paths that the routing protocol knows about. This handles asymmetric routing in multi-homed environments where traffic might legitimately arrive on a backup path.

### 2.3 Mode Selection Decision Matrix

| Network Topology | Recommended Mode | Rationale |
|:---|:---|:---|
| Single-homed customer (stub) | Strict | Only one path in/out; symmetric by definition |
| Dual-homed customer (single ISP) | Feasible | Traffic may arrive on either link |
| Multi-homed customer (multi ISP) | Loose | Asymmetric routing expected |
| ISP peering interface | Loose | Hot-potato routing causes asymmetry |
| ISP core (P router) | None or Loose | Full table; all sources reachable everywhere |
| Data center leaf-spine (ECMP) | Feasible | ECMP provides multiple valid paths |
| IXP route server participant | None | Many peers, extreme asymmetry |

### 2.4 uRPF Interaction with Default Routes

A critical subtlety: if the FIB contains a default route (0.0.0.0/0), loose mode uRPF will match every source address against it and never drop anything. This makes loose mode useless on routers with a default route unless `allow-default` is explicitly disabled.

Strict mode with a default route pointing to the upstream interface will accept spoofed traffic arriving from upstream, since the default route's next-hop matches the upstream interface. This is why iACLs and anti-spoofing ACLs complement uRPF — they provide explicit deny rules that override the permissive default route match.

---

## 3. TCP-AO vs MD5: Routing Protocol Authentication Analysis

### 3.1 MD5 Authentication (RFC 2385)

BGP MD5 authentication (also used by OSPF and LDP) appends an MD5 hash to each TCP segment. The hash covers the TCP pseudo-header, the segment data, and a shared secret key.

**Strengths:**
- Simple to configure (single password per neighbor)
- Universally supported across all vendors and platforms
- Prevents trivial TCP RST injection and blind data injection

**Weaknesses:**
- **No key rollover:** The protocol supports exactly one key per session. Changing the key requires simultaneous configuration on both peers, which causes a brief authentication failure and potential session flap. In practice, operators never rotate keys.
- **Weak hash algorithm:** MD5 is cryptographically broken (collision attacks since 2004, practical preimage attacks feasible). While the specific MD5 usage in TCP (HMAC-style construction with the segment as input) is harder to exploit than generic MD5, the algorithm provides no security margin.
- **No key ID:** The authentication extension carries no identifier for which key was used. The receiver must try the single configured key. This makes automated key management impossible.
- **TCP option space:** The MD5 signature consumes 18 bytes of TCP option space, reducing room for other options like SACK and window scaling.

### 3.2 TCP Authentication Option (RFC 5925)

TCP-AO replaces MD5 with a modern framework designed for operational realities:

**Key chains with lifetimes:** TCP-AO references a key chain containing multiple keys (Master Key Tuples, or MKTs), each with independent send and accept lifetimes. This enables hitless key rollover:

```
Timeline for hitless key rotation:

Key 1 send:    |========================|
Key 1 accept:  |============================|      (accept window extends beyond send)
Key 2 send:            |========================|
Key 2 accept:      |============================|  (accept starts before send)

                   ^ overlap window: both keys valid
```

During the overlap window, the sender transmits with Key 2 while the receiver accepts either Key 1 or Key 2. This allows operators to configure the new key on each peer independently without time synchronization.

**Algorithm agility:** TCP-AO supports pluggable MAC algorithms. Current implementations support HMAC-SHA-1-96 and HMAC-SHA-256-128, with the ability to add new algorithms without protocol changes.

**Key Derivation Function (KDF):** TCP-AO derives per-connection traffic keys from the master key using a KDF that incorporates the connection's 4-tuple (source IP, dest IP, source port, dest port). This means even if two sessions share the same master key, their traffic keys differ, preventing replay across sessions.

**Key ID field:** Each segment carries a KeyID (which MKT was used for sending) and RNextKeyID (which MKT the sender expects to receive), enabling coordinated key negotiation between peers.

**Reduced option space:** TCP-AO uses 16 bytes minimum versus MD5's 18 bytes, and the MAC length is variable (shorter MACs trade security margin for option space).

### 3.3 Practical Migration Path

Migrating from MD5 to TCP-AO requires a session reset because the two mechanisms are mutually exclusive and there is no in-band negotiation. The recommended approach:

1. Schedule maintenance window
2. Configure TCP-AO key chain on both peers (but do not activate)
3. Simultaneously remove MD5 password and activate TCP-AO
4. BGP session will reset and re-establish with TCP-AO authentication
5. Verify with `show tcp ao statistics`

In environments where session resets are unacceptable, graceful restart (BGP GR) can maintain forwarding during the brief session disruption.

---

## 4. Defense-in-Depth: Layered Infrastructure Protection

### 4.1 The Kill Chain for Network Infrastructure

An attack against network infrastructure typically follows this progression:

```
Phase 1: Reconnaissance
  └─ Traceroute, SNMP scanning, banner grabbing, BGP looking glass
  └─ Identify router models, IOS versions, interface addresses

Phase 2: Exploitation of Access
  └─ SSH brute force, SNMP community guessing, default credentials
  └─ Exploit known CVE in management service (HTTP, SNMP, SSH)
  └─ Rogue routing protocol injection (unauthenticated BGP/OSPF)

Phase 3: Persistence
  └─ Create hidden admin account, modify startup-config
  └─ Install modified firmware (implant)
  └─ Add GRE tunnel for exfiltration

Phase 4: Impact
  └─ Traffic interception (mirror/SPAN to attacker)
  └─ Route manipulation (redirect traffic through attacker path)
  └─ Service disruption (clear bgp *, interface shutdown)
  └─ Credential harvesting (intercept RADIUS/TACACS+ from transiting traffic)
```

### 4.2 Control Mapping to Kill Chain

Each infrastructure security control disrupts one or more phases:

| Control | Phase Disrupted | Mechanism |
|:---|:---|:---|
| iACLs | 1, 2 | Block probes and exploit traffic from reaching infrastructure addresses |
| uRPF | 2 | Prevent spoofed source addresses used in blind injection attacks |
| CoPP | 2 | Rate-limit exploit attempts, prevent CPU exhaustion |
| MPP | 2 | Restrict management protocol access to designated interfaces |
| Routing auth (MD5/TCP-AO) | 2 | Prevent forged routing protocol injection |
| GTSM (TTL security) | 2 | Ensure eBGP peers are directly connected (TTL=255) |
| SNMP v3 authPriv | 1, 2 | Encrypted + authenticated management queries |
| VTY ACLs | 2 | Restrict SSH access to management subnet |
| AAA + TACACS+ | 2, 3 | Centralized auth, per-command authorization, audit trail |
| Logging + syslog TLS | 3, 4 | Detect unauthorized changes, tamper-resistant log export |
| Secure boot | 3 | Verify firmware integrity, prevent implant persistence |
| NTP auth | 2 | Prevent time manipulation (affects certificate validation, logs) |

### 4.3 Minimum Viable Security Posture

Not all environments can deploy every control simultaneously. The following priority order maximizes security impact per unit of effort:

**Priority 1 (deploy immediately):**
- SSH v2 only, disable Telnet
- VTY ACLs restricting to management subnet
- Remove SNMP v1/v2c community strings
- Enable secret (type 8 or 9, scrypt-based)
- Disable unused services (HTTP, finger, CDP on external interfaces)
- CoPP with default strict profile

**Priority 2 (deploy within 30 days):**
- iACLs on all external-facing interfaces
- Routing protocol authentication (MD5 minimum)
- uRPF on customer-facing interfaces
- NTP authentication
- Centralized syslog with timestamps
- AAA via TACACS+ with command accounting

**Priority 3 (deploy within 90 days):**
- SNMP v3 authPriv migration
- TCP-AO replacing MD5 (where supported)
- GTSM for all eBGP sessions
- BGP max-prefix limits
- RPKI/ROV for prefix validation
- Secure boot verification
- Management-plane protection (MPP)

---

## 5. CIS Benchmark Mapping

### 5.1 CIS Cisco IOS Benchmark Controls

The CIS Cisco IOS/IOS-XE Benchmark provides specific technical controls that map directly to the infrastructure security practices covered in this document:

| CIS Control ID | Control Title | Infrastructure Security Mapping |
|:---|:---|:---|
| 1.1.1 | Enable AAA | TACACS+ for authentication, authorization, accounting |
| 1.1.2 | Enable secret | Type 8/9 scrypt-hashed enable secret |
| 1.2.1 | SSH for remote management | SSH v2 only, disable Telnet |
| 1.2.2 | SSH timeout | ip ssh time-out 60 |
| 1.2.3 | SSH retries | ip ssh authentication-retries 3 |
| 1.3.1 | VTY ACL | access-class on line vty |
| 1.4.1 | Exec timeout | exec-timeout 15 0 on VTY |
| 2.1.1 | CDP disabled | no cdp run / no cdp enable on external interfaces |
| 2.2.1 | No IP source-route | no ip source-route |
| 2.2.2 | No IP directed-broadcast | no ip directed-broadcast |
| 2.2.3 | No IP proxy-arp | no ip proxy-arp on untrusted |
| 2.3.1 | SNMP community removed | no snmp-server community |
| 2.3.2 | SNMP v3 | snmp-server group v3 priv |
| 3.1.1 | Timestamps | service timestamps log datetime msec |
| 3.1.2 | Logging host | logging host with TLS |
| 3.2.1 | NTP authentication | ntp authenticate + trusted-key |
| 4.1.1 | Routing authentication | OSPF/BGP/EIGRP auth |
| 5.1.1 | iACL | ACL on external interfaces |
| 5.2.1 | uRPF | ip verify unicast source reachable-via |

### 5.2 NIST 800-53 Control Family Mapping

| NIST Control | Description | Infrastructure Implementation |
|:---|:---|:---|
| AC-3 | Access Enforcement | VTY ACLs, MPP, SNMP v3 ACLs |
| AC-4 | Information Flow Enforcement | iACLs, uRPF, anti-spoofing |
| AC-17 | Remote Access | SSH v2, VTY restrictions, MPP |
| AU-2 | Audit Events | Syslog, AAA accounting, archive logging |
| AU-3 | Content of Audit Records | Timestamps with msec, origin-id |
| AU-6 | Audit Review | Centralized syslog, SIEM integration |
| CM-7 | Least Functionality | Disable unused services, shut unused ports |
| IA-2 | Identification and Authentication | TACACS+, local enable secret |
| IA-5 | Authenticator Management | Key chains, TCP-AO key rotation |
| SC-5 | Denial of Service Protection | CoPP, iACLs, storm control |
| SC-7 | Boundary Protection | iACLs, uRPF, edge ACLs |
| SC-8 | Transmission Confidentiality | SNMP v3 AES, syslog TLS, SSH |
| SI-4 | System Monitoring | SNMP traps, syslog, NetFlow |

---

## 6. Operational Considerations

### 6.1 Change Management for Infrastructure Security

Infrastructure security changes carry inherent risk because a misconfiguration can cause self-lockout (VTY ACL blocks the operator) or routing disruption (authentication mismatch tears down adjacencies).

**Safe deployment sequence for routing authentication:**

1. Verify current neighbor state: `show ip bgp summary`, `show ip ospf neighbor`
2. Configure authentication on the local device (key chain first, then activate on interface/neighbor)
3. Monitor for adjacency loss for 60 seconds
4. If adjacency drops, immediately remove authentication (rollback)
5. Configure authentication on the remote device within the hold/dead timer window
6. Verify adjacency re-establishes with authentication active on both ends

**Safe deployment sequence for iACLs:**

1. Build the ACL with explicit permits for all known-good traffic
2. Add `log` keyword to the final `deny` entries
3. Apply the ACL in monitor/count mode if the platform supports it (NX-OS `statistics per-entry`)
4. Monitor logged denies for 24-72 hours
5. Remove `log` from deny entries (logging under load causes CPU impact)
6. Move from permissive (permit ip any any at end) to restrictive (remove final permit)

### 6.2 Monitoring Infrastructure Security Controls

Deployed controls require continuous monitoring to detect both attacks and misconfigurations:

**CoPP exceed counters:** A sudden increase in dropped packets on the BGP or OSPF class indicates either an attack or a legitimate traffic spike (convergence event, new peers). Correlate CoPP drops with routing table changes and BGP session states.

**uRPF drop counters:** Non-zero uRPF drops can indicate spoofed traffic (good — the control is working) or a routing asymmetry that developed after uRPF was deployed (bad — legitimate traffic is being dropped). Investigate every uRPF drop spike.

**Authentication failure logs:** Repeated OSPF/BGP authentication failures indicate either a key mismatch (operational issue) or an active injection attempt. Correlate with source addresses.

**VTY connection logs:** Failed SSH attempts should be logged and correlated. A spike from a single source suggests brute force; distributed failures suggest credential stuffing with a leaked password list.

### 6.3 Automation and Compliance Scanning

Infrastructure security controls can be validated automatically using configuration compliance tools:

- **Batfish/Pybatfish:** Offline configuration analysis that can verify ACL correctness, routing authentication presence, and service configuration without connecting to live devices.
- **Cisco Prime/DNA Center:** Template compliance checking against CIS benchmark configurations.
- **NAPALM/Nornir:** Python frameworks for automated configuration auditing across multi-vendor networks.
- **Custom scripts:** Parse `show running-config` output for required security statements (snmp v3, no snmp community, ip ssh version 2, etc.).

A compliance scan should run at minimum weekly and after every change window, with results fed into the organization's GRC (Governance, Risk, Compliance) platform.
