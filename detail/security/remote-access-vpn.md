# Remote Access VPN — SSL/TLS vs IPsec Architecture, AnyConnect Internals, and Zero-Trust Evolution

> *Remote access VPNs provide secure connectivity for individual users to enterprise networks. The two dominant approaches — SSL/TLS VPN and IPsec VPN — have fundamentally different architectures, protocol stacks, and deployment characteristics. Understanding the AnyConnect client architecture, DTLS optimization for real-time traffic, DAP evaluation logic, split tunnel security implications, always-on design, and the evolution toward ZTNA is essential for designing secure, scalable remote access infrastructure.*

---

## 1. SSL/TLS VPN vs IPsec VPN — Architectural Comparison

### Protocol Stack Comparison

```
SSL/TLS VPN (AnyConnect SSL):

  Application Layer
  ─────────────────
  AnyConnect Client ←→ VPN Head-end
       |                    |
  SSL/TLS (TCP 443)    or DTLS (UDP 443)
       |                    |
  TCP/UDP                TCP/UDP
       |                    |
  IP                     IP
       |                    |
  Physical               Physical

  Encapsulation:
    [IP][TCP/UDP 443][TLS Record][VPN Payload (inner IP packet)]

  Characteristics:
    - Operates at Layer 4-7 (TCP/UDP + TLS)
    - Uses standard HTTPS port (443) — firewall-friendly
    - TLS provides authentication, encryption, integrity
    - DTLS provides UDP-based tunnel for real-time traffic
    - Clientless mode possible (browser-based, no software)
    - Per-application tunneling supported


IPsec VPN (AnyConnect IKEv2):

  Application Layer
  ─────────────────
  AnyConnect Client ←→ VPN Head-end
       |                    |
  IKEv2 (UDP 500/4500)
       |                    |
  ESP (IP protocol 50)  or ESP-in-UDP (NAT-T, UDP 4500)
       |                    |
  IP                     IP
       |                    |
  Physical               Physical

  Encapsulation:
    [IP][ESP Header][Encrypted inner IP packet][ESP Trailer][ICV]
    Or with NAT-T:
    [IP][UDP 4500][ESP Header][Encrypted inner IP packet][ESP Trailer][ICV]

  Characteristics:
    - Operates at Layer 3 (IP + ESP)
    - IKEv2 for key exchange, ESP for data encryption
    - Full IP-level tunneling (all protocols, not just TCP/UDP)
    - Requires UDP 500 + 4500 (may be blocked on restrictive networks)
    - Always requires client software
    - Better performance for bulk encryption (fewer protocol layers)
```

### Detailed Comparison

```
Feature                SSL/TLS VPN              IPsec VPN
─────────────────────────────────────────────────────────────────────
Transport              TCP 443 / UDP 443 (DTLS) UDP 500/4500, ESP (50)
Firewall traversal     Excellent (HTTPS port)   Moderate (needs 500/4500)
NAT traversal          Native (TCP/UDP)         NAT-T required (UDP 4500)
Restrictive networks   Works through proxies    Often blocked
Client requirement     Optional (clientless)    Always required
Clientless mode        Yes (browser portal)     No
Per-app tunneling      Yes                      No (full IP tunnel)
Multicast support      Limited                  Yes
Performance overhead   Higher (TLS + TCP)       Lower (ESP direct)
Real-time traffic      DTLS mitigates TCP issue Good (UDP-native)
Authentication         Cert, SAML, EAP, MFA     Cert, PSK, EAP
Split tunneling        Yes                      Yes
Posture assessment     Yes (AnyConnect module)  Yes (AnyConnect module)
Platform support       Broad (+ browser)        Broad (client only)
```

### When to Use Which

```
Choose SSL/TLS VPN when:
  - Users are on restrictive networks (hotels, airports, coffee shops)
  - Firewall rules only allow port 443 outbound
  - Clientless access is needed for some users (BYOD, contractors)
  - SAML/SSO integration is required
  - Per-application tunneling is desired

Choose IPsec VPN when:
  - Maximum throughput is critical (bulk data transfer)
  - Multicast traffic must traverse the tunnel (IP multicast)
  - All IP protocols must be tunneled (not just TCP/UDP)
  - Users are on well-controlled networks (no port restrictions)
  - Lower latency required (fewer protocol layers)

Choose both (AnyConnect with fallback):
  - AnyConnect tries IKEv2 first, falls back to SSL/TLS
  - Best of both worlds: performance when possible, compatibility always
  - Configured via AnyConnect XML profile: <PrimaryProtocol>IPsec</PrimaryProtocol>
```

---

## 2. DTLS for Real-Time Traffic

### The TCP-over-TCP Problem

```
TCP as a VPN transport creates a well-known pathological condition
when carrying TCP application traffic:

  Outer TCP (TLS tunnel) + Inner TCP (application):

    Packet loss on the outer link triggers:
      1. Outer TCP retransmits the lost segment
      2. Inner TCP also detects loss (delayed ACK / timeout)
      3. Inner TCP retransmits independently
      4. Both retransmissions compete for bandwidth
      5. Outer TCP congestion window shrinks
      6. Inner TCP congestion window also shrinks
      7. Combined back-off is multiplicative — severe throughput collapse

    This is called "TCP meltdown" or "TCP-in-TCP" problem:
      - Observed at >1% packet loss
      - Throughput can drop to near-zero
      - Latency spikes to seconds
      - Real-time traffic (voice, video) becomes unusable

  UDP application traffic over TCP tunnel:
    - Less pathological (no inner TCP retransmission)
    - But still suffers from head-of-line blocking
    - A single lost outer TCP segment blocks all tunnel traffic
      until retransmitted, even if the inner UDP app doesn't care
```

### DTLS Architecture

```
DTLS (Datagram Transport Layer Security) solves TCP-in-TCP:

  Based on TLS 1.2 (DTLS 1.2 = RFC 6347) or TLS 1.3 (DTLS 1.3 = RFC 9147)
  Runs over UDP instead of TCP

  Properties:
    - No head-of-line blocking (each datagram independent)
    - No retransmission of tunnel layer (application decides)
    - Reordering handled by application, not tunnel
    - Same cryptographic strength as TLS
    - Same authentication (certificates, etc.)

  AnyConnect dual-channel architecture:
    1. TLS channel (TCP 443) — control plane + reliable data
       - Connection setup and teardown
       - Configuration exchange
       - Reliable application traffic (when DTLS unavailable)
    2. DTLS channel (UDP 443) — data plane
       - Bulk data transfer
       - Real-time traffic (voice, video)
       - Falls back to TLS if DTLS blocked

  Establishment:
    1. AnyConnect connects via TLS (TCP 443)
    2. Authenticates, receives session parameters
    3. AnyConnect opens DTLS channel (UDP 443) using session key
       from TLS handshake (avoids separate DTLS handshake)
    4. Data plane switches to DTLS
    5. TLS channel remains as fallback/control

  DTLS header overhead:
    DTLS 1.2 record header: 13 bytes
    DTLS epoch + sequence: 8 bytes
    AES-GCM nonce: 8 bytes
    AES-GCM tag: 16 bytes
    Total DTLS overhead: ~45 bytes per datagram
    vs TLS: ~29 bytes per record (but TCP adds 20+ bytes)
```

### DTLS vs TLS Performance

```
Performance comparison (1% packet loss, 100ms RTT):

  Metric              TLS (TCP 443)    DTLS (UDP 443)
  ────────────────────────────────────────────────────
  Throughput           ~60% of max      ~95% of max
  Latency (avg)        150-300ms        100-110ms
  Jitter               High (30-80ms)   Low (5-15ms)
  Voice MOS score      2.5-3.0          4.0-4.2
  Video quality        Degraded         Near-native
  Recovery from loss   Slow (TCP back-off) Instant (no back-off)

At 5% packet loss:
  TLS throughput drops to ~20% of max (TCP meltdown)
  DTLS throughput remains at ~80% of max

  Conclusion:
    DTLS is essential for acceptable remote access VPN performance
    TLS-only VPN is a fallback, not a primary data channel
    Always verify DTLS is established: show vpn-sessiondb detail anyconnect
```

---

## 3. AnyConnect Client Architecture

### Module Architecture

```
AnyConnect Secure Mobility Client is a modular platform:

  Core VPN Module (always present):
    - VPN agent daemon (vpnagentd / vpnagent.exe)
    - Manages tunnel lifecycle (connect, disconnect, reconnect)
    - Handles authentication (credentials, certificates, SAML)
    - Implements TLS + DTLS channels
    - IP stack integration (virtual adapter, routing, DNS)
    - Split tunnel enforcement
    - Always-on VPN logic

  Optional Modules (loaded as needed):

  ┌─────────────────────────────────────────────────────────────┐
  │ AnyConnect Core VPN                                         │
  ├──────────┬──────────┬──────────┬──────────┬────────────────┤
  │ Network  │ Posture  │ Web      │ DART     │ Umbrella       │
  │ Access   │ (ISE)    │ Security │ (Diag)   │ (DNS Security) │
  │ Manager  │          │ (SWG)    │          │                │
  │ (NAM)    │          │          │          │                │
  ├──────────┼──────────┼──────────┼──────────┼────────────────┤
  │ 802.1X   │ Endpoint │ Cloud    │ Log      │ DNS-layer      │
  │ wired/   │ compli-  │ web      │ collect  │ security +     │
  │ wireless │ ance     │ proxy    │ + bundle │ IP-layer       │
  │ suppl.   │ checks   │ enforce  │          │ enforcement    │
  └──────────┴──────────┴──────────┴──────────┴────────────────┘

  Module deployment:
    Modules specified in group-policy:
      anyconnect modules value dart,posture,websecurity,umbrella,nam
    Modules downloaded automatically when user connects
    Enabled/disabled via AnyConnect XML profile
```

### Virtual Adapter and Routing

```
AnyConnect creates a virtual network adapter on the endpoint:

  Windows:  "Cisco AnyConnect Secure Mobility Client Virtual Miniport Adapter"
  macOS:    utun0 (or utunN, kernel TUN device)
  Linux:    tun0 (via /dev/net/tun)

  Adapter behavior:
    - Receives IP address from VPN pool (or DHCP via VPN)
    - Default gateway manipulated based on tunnel mode:

  Full tunnel (tunnelall):
    - Default route (0.0.0.0/0) points to VPN adapter
    - All traffic forced through VPN
    - Original default route saved and restored on disconnect
    - DNS servers replaced with VPN-provided DNS

  Split tunnel (tunnelspecified):
    - Only specific routes added pointing to VPN adapter
    - Example: 10.0.0.0/8 → VPN adapter
    - Default route unchanged (internet direct)
    - DNS: split-DNS sends specific domain queries to VPN DNS

  Exclude specified (split-exclude):
    - Default route through VPN
    - Specific routes excluded (sent direct)
    - Example: 0.0.0.0/0 → VPN, but 10.100.0.0/16 → direct
    - Use case: VPN for everything except local LAN/printers

  Route management:
    AnyConnect modifies the OS routing table dynamically
    On disconnect: all VPN routes removed, original state restored
    On crash/kill: routes may persist (cleanup on next connect attempt)
```

### Reconnect and Roaming

```
AnyConnect handles network transitions gracefully:

  Session resumption:
    - TLS session tickets / IKEv2 session resumption
    - Avoids full re-authentication on brief disconnects
    - Configurable reconnect timeout (default 120 seconds)

  Network roaming:
    - Detects interface changes (Wi-Fi → Ethernet → cellular)
    - Automatically re-establishes tunnel on new interface
    - Session preserved (same IP address, no re-auth if within timeout)
    - DTLS channel re-established independently

  Suspend/resume (laptop sleep):
    - AutoReconnectBehavior: ReconnectAfterResume
    - On wake: AnyConnect attempts to resume session
    - If session expired on head-end: full re-authentication

  IKEv2 MOBIKE (RFC 4555):
    - Allows IP address change without IKE SA renegotiation
    - Initiator sends UPDATE_SA_ADDRESSES
    - Both peers update SA endpoints
    - No traffic interruption
```

---

## 4. DAP Evaluation Logic

### DAP Architecture

```
Dynamic Access Policies provide post-authentication policy decisions
based on real-time endpoint and user attributes:

  Evaluation flow:
    1. User authenticates successfully
    2. Group policy assigned (from tunnel-group or RADIUS)
    3. DAP engine evaluates ALL DAP records against session
    4. Records with ALL criteria matching are "selected"
    5. Selected DAP records are combined
    6. Combined DAP attributes override group policy
    7. Session established with final merged policy

  DAP record structure:
    ┌─────────────────────────────────────────────┐
    │ DAP Record: "CORP_COMPLIANT"                │
    │ Priority: 100                                │
    │                                              │
    │ Selection Criteria (AND logic):             │
    │   AAA: LDAP memberOf = "Corp-Employees"     │
    │   AND                                        │
    │   Endpoint: OS = Windows 10+                 │
    │   AND                                        │
    │   Endpoint: AnyConnect posture = Compliant   │
    │                                              │
    │ Action: Continue                             │
    │ Attributes:                                  │
    │   Network ACL: FULL_ACCESS_ACL               │
    │   Banner: "Welcome, compliant device"        │
    └─────────────────────────────────────────────┘
```

### Attribute Sources and Matching

```
DAP can match on multiple attribute sources:

  1. AAA Attributes (from RADIUS/LDAP response):
     - LDAP attributes: memberOf, department, title
     - RADIUS attributes: Class, Filter-Id, cisco-av-pair
     - RADIUS vendor-specific attributes
     Example: memberOf = CN=VPN-Users,OU=Groups,DC=corp,DC=com

  2. Endpoint Attributes (from AnyConnect):
     - Operating system (type + version)
     - AnyConnect version
     - Posture assessment status (compliant/non-compliant/unknown)
     - Anti-virus status (installed, definition age)
     - Anti-malware status
     - Firewall status (enabled/disabled)
     - Disk encryption status

  3. Connection Attributes:
     - Protocol (AnyConnect, clientless, IKEv2)
     - Tunnel group name
     - Client certificate attributes

  4. Custom Attributes:
     - Lua scripting for complex logic
     - LUA can access any attribute and apply custom evaluation
     Example: time-of-day restrictions, geolocation checks
```

### Multi-DAP Combination Logic

```
When multiple DAP records match, attributes are combined:

  ACL combination (most restrictive wins):
    DAP_A: network-acl = ACL_ALLOW_WEB (permit 80,443)
    DAP_B: network-acl = ACL_ALLOW_SSH (permit 22)
    Combined: both ACLs applied (intersection — must pass ALL ACLs)
    Result: traffic must be permitted by BOTH ACL_ALLOW_WEB AND ACL_ALLOW_SSH
    Effective: only traffic matching both ACLs is allowed

  Banner combination:
    All matching DAP banners displayed (concatenated)

  URL/web-type ACL combination:
    Union of all matching DAPs (additive)

  Numeric attributes (timeouts):
    Most restrictive value wins (shortest timeout)

  Action attributes:
    "terminate" in ANY matching DAP → session terminated
    "quarantine" overrides "continue"

  Priority:
    Higher priority number = higher precedence for conflicting attributes
    DAP with priority 100 overrides DAP with priority 50

  Default policy (DfltAccessPolicy):
    Applied only when NO DAP records match
    Typically configured as most restrictive (deny-all or quarantine)
    Best practice: always have at least one DAP match for legitimate users
```

### DAP Design Patterns

```
Pattern 1: Posture-based tiered access
  DAP_COMPLIANT (priority 100):
    Criteria: posture = compliant
    Action: full-access ACL

  DAP_NONCOMPLIANT (priority 90):
    Criteria: posture = non-compliant
    Action: restricted ACL + remediation URL

  DAP_UNKNOWN (priority 80):
    Criteria: posture = unknown
    Action: quarantine ACL (only posture server + remediation)

Pattern 2: Role + device combined
  DAP_ADMIN_CORP (priority 100):
    Criteria: LDAP group = Admins AND device = managed
    Action: full admin access

  DAP_ADMIN_BYOD (priority 90):
    Criteria: LDAP group = Admins AND device = unmanaged
    Action: limited admin access (web-only, no SSH)

  DAP_USER_ANY (priority 50):
    Criteria: LDAP group = Users
    Action: standard user access

Pattern 3: Time-based restrictions (Lua)
  DAP_AFTERHOURS (priority 200):
    Criteria: Lua script (hour < 6 or hour > 22)
    Action: terminate (no VPN access outside business hours)
```

---

## 5. Split Tunnel Security Implications

### Split Tunnel Attack Surface

```
Split tunneling routes only specific traffic through VPN:
  Corporate traffic (10.0.0.0/8) → VPN tunnel
  Internet traffic → direct (bypasses VPN)

Security risks:

  1. Endpoint as bridge (dual-homed attack):
     Attacker on local network → endpoint → corporate network
     The VPN endpoint has routes to both networks simultaneously
     If endpoint firewall is weak → lateral movement into corporate

     Mitigation:
       - AnyConnect host-scan / posture (verify endpoint firewall)
       - Endpoint protection platform (EDR)
       - Host-based firewall rules (block local-to-corporate bridging)

  2. DNS leakage:
     Split-DNS not configured → corporate DNS queries go to ISP DNS
     ISP or attacker on local network sees internal hostnames
     Information disclosure: internal server names, service topology

     Mitigation:
       - Configure split-dns for corporate domains
       - Or use full tunnel for DNS (split-tunnel-all-dns enable)

  3. Man-in-the-middle on local network:
     Internet traffic is unencrypted (no VPN protection)
     Attacker on local Wi-Fi can intercept non-VPN traffic
     If user accesses corporate SaaS apps via direct internet →
     credentials exposed if not using HTTPS

     Mitigation:
       - HTTPS everywhere for corporate SaaS
       - Cisco Umbrella / DNS security module
       - Web security module in AnyConnect

  4. Malware exfiltration:
     Malware on endpoint sends stolen corporate data via direct internet
     Data bypasses corporate DLP/proxy (not going through VPN)

     Mitigation:
       - Full tunnel (all traffic through corporate inspection)
       - Or exclude-specified (VPN default, exclude only local subnet)
       - Endpoint DLP agent
       - AnyConnect Web Security module (cloud proxy)

  5. Local network enumeration:
     Endpoint can reach local network (printers, IoT, other devices)
     Compromised endpoint can pivot to local network attacks
     Corporate security team has no visibility

     Mitigation:
       - Always-on VPN with Closed failure policy
       - Local LAN access control in AnyConnect profile
       - Disable local LAN access: <LocalLanAccess>false</LocalLanAccess>
```

### Split Tunnel Modes Compared

```
Mode                    Security     User Experience    Bandwidth
────────────────────────────────────────────────────────────────────
Full tunnel             Highest      Worst (slow web)   Highest load
  (tunnelall)           All inspected Internet via corp  on head-end

Split include           Moderate     Good               Moderate
  (tunnelspecified)     Corp only    Direct internet    Corp traffic only

Split exclude           High         Good               Low load
  (excludespecified)    Default VPN  Local LAN direct   Most through VPN
                        Exclude local                    head-end

Per-app tunnel          Varies       Best               Lowest
  (app-based)           App-specific Other apps direct  Only target apps

Dynamic split tunnel    High         Good               Adaptive
  (AnyConnect 4.5+)     Domain-based Auto-adjusts       Based on destination
                        Include/exclude by FQDN
```

---

## 6. Always-On VPN Architecture

### Design Goals

```
Always-on VPN ensures the endpoint NEVER has unprotected network access:

  Goals:
    1. Automatic connection at boot/login (no user action)
    2. Automatic reconnection on network change
    3. No user ability to disconnect
    4. Optional: block all traffic if VPN is down (closed policy)

  Architecture:
    AnyConnect Service → runs as system service (not user process)
    Starts before user login → machine tunnel
    User authenticates → user tunnel (may replace or coexist)

  Connection failure policy:
    Open:   allow network access if VPN fails to connect
            User can work locally but has no VPN protection
            Logging of unprotected period

    Closed: block ALL network access if VPN fails
            No internet, no local network (except captive portal detection)
            Most secure but can lock out users if VPN is down
            Requires careful exception handling

  Captive portal detection:
    Problem: closed policy blocks all traffic, including captive portal
    Solution: AnyConnect detects captive portals (HTTP probe)
    Behavior: temporarily allows browser for portal authentication
    Then: immediately re-attempts VPN connection
    Profile setting: <CaptivePortalRemediationBrowserFailover>true</CaptivePortalRemediationBrowserFailover>
```

### Trusted Network Detection (TND)

```
TND automatically disables VPN when inside the corporate network:

  Detection methods:
    1. DNS-based: resolve specific internal domain → if successful, inside
       <TrustedDNSDomains>corp.internal.example.com</TrustedDNSDomains>
       <TrustedDNSServers>10.1.1.53,10.1.2.53</TrustedDNSServers>

    2. Certificate-based: specific HTTPS server responds with known cert
       <TrustedHttpsServerList>
         <TrustedHttpsServer>https://trust.internal.example.com</TrustedHttpsServer>
       </TrustedHttpsServerList>

  Logic:
    Network change detected →
      Probe trusted DNS/HTTPS →
        If reachable → inside corp → disconnect VPN (or don't connect)
        If not reachable → outside corp → connect VPN (always-on)

  Security consideration:
    DNS-based TND can be spoofed (attacker runs DNS server with matching zone)
    HTTPS-based TND is more secure (requires matching certificate)
    Both methods should be used together for defense in depth

  Machine tunnel + user tunnel coexistence:
    Machine tunnel: established at boot, uses certificate auth
    User tunnel: established at login, uses user credentials
    TND applies to user tunnel (machine tunnel may stay connected inside corp)
```

---

## 7. Zero-Trust Network Access (ZTNA) Evolution

### From VPN to ZTNA

```
Traditional VPN limitations:
  1. Network-level access: VPN grants access to entire network segments
     User on VPN can reach any host in the routed VPN pool
     Lateral movement possible once inside

  2. Castle-and-moat model: once authenticated, fully trusted
     No continuous verification during session
     Compromised endpoint maintains access until session expires

  3. Backhauling penalty: all traffic through central VPN head-end
     Cloud/SaaS traffic must traverse corporate network
     Increases latency and loads corporate infrastructure

  4. Complex policy: ACLs and group policies are coarse-grained
     Difficult to express per-application, per-context policies
     DAP helps but is limited to ASA/FTD platform

ZTNA principles (NIST SP 800-207):
  1. Never trust, always verify
  2. Least-privilege access (per-application, not per-network)
  3. Continuous verification (posture, behavior, context)
  4. Micro-segmentation (application-level isolation)
  5. Assume breach (limit blast radius)

ZTNA architecture:
  ┌──────────┐    ┌─────────────┐    ┌──────────────┐
  │ User +   │───>│ ZTNA Proxy/ │───>│ Application  │
  │ Device   │    │ Broker      │    │ (per-app     │
  │          │    │ (cloud edge)│    │  connector)  │
  └──────────┘    └─────────────┘    └──────────────┘
        │               │
        │ Identity +     │ Policy evaluation:
        │ Device posture │   Identity + device + context
        │ Context        │   → per-app access decision
        │               │
  ┌─────▼───────────────▼─────┐
  │ Identity Provider (IdP)    │
  │ + Device Trust Engine      │
  │ + Policy Engine            │
  └────────────────────────────┘

  Key differences from VPN:
    VPN:  authenticate → grant network access → user reaches apps
    ZTNA: authenticate → evaluate per-app policy → grant app access only
```

### Cisco ZTNA / Secure Access

```
Cisco's evolution from VPN to ZTNA:

  Stage 1: VPN + TrustSec (current)
    AnyConnect VPN → network access
    TrustSec SGT → macro/micro-segmentation
    ISE posture → endpoint compliance
    Limitation: network-level access, not app-level

  Stage 2: Duo + AnyConnect (hybrid)
    Duo MFA → strong authentication
    Duo Device Trust → endpoint verification
    AnyConnect + Duo → VPN with per-app MFA
    Limitation: still network-level tunnel

  Stage 3: Cisco Secure Access (ZTNA)
    Cloud-delivered security service
    Per-application access (no full network tunnel)
    Continuous trust evaluation
    Identity-aware proxy (replaces VPN for SaaS/web apps)
    Clientless for web apps, agent-based for non-web
    Integration: Duo identity + Umbrella DNS + Secure Web Gateway

  Coexistence:
    VPN is not going away — needed for:
      - Legacy applications (non-HTTP)
      - Full network access for IT/admin
      - Environments without ZTNA infrastructure
    ZTNA applies to:
      - Web and SaaS applications
      - Developer tools (SSH via reverse proxy)
      - Modern microservices with API gateways

  Migration approach:
    1. Deploy VPN with posture + MFA (baseline security)
    2. Add per-app policies (DAP + group policies)
    3. Deploy ZTNA for web/SaaS apps (reduce VPN scope)
    4. Move non-web apps to ZTNA connectors
    5. VPN becomes fallback for edge cases only
```

### ZTNA vs VPN Decision Matrix

```
Use Case                          VPN    ZTNA    Notes
──────────────────────────────────────────────────────────────
Web app access                    OK     Better  ZTNA: per-app, no tunnel
SaaS access (O365, Salesforce)    Poor   Better  VPN backhauling is wasteful
SSH/RDP to servers                OK     OK      ZTNA via TCP proxy/connector
Legacy thick-client app           Better Poor    May need full network access
VoIP/video (SIP/RTP)             Better Poor    Needs full IP-level tunnel
Network admin (CLI to switches)   Better OK      Full network access needed
IoT device management             Better Poor    Non-standard protocols
Contractor access (BYOD)          OK     Better  ZTNA: no network exposure
High-security environments        OK     Better  ZTNA: smaller blast radius
Branch office connectivity        Better N/A     Site-to-site VPN, not RA
Offline/air-gapped networks       Better N/A     ZTNA requires cloud broker
```

---

## See Also

- site-to-site-vpn
- ipsec
- tls
- cisco-ftd
- cisco-ise
- dot1x
- zero-trust
- oauth

## References

- Cisco AnyConnect Secure Mobility Client Administrator Guide
- Cisco ASA VPN Configuration Guide (9.x)
- RFC 6347 — Datagram Transport Layer Security Version 1.2 (DTLS)
- RFC 9147 — The Datagram Transport Layer Security Protocol Version 1.3
- RFC 4555 — IKEv2 Mobility and Multihoming (MOBIKE)
- NIST SP 800-207 — Zero Trust Architecture
- Cisco Secure Access (ZTNA) Architecture Guide
- Cisco Duo Security — Zero Trust for the Workforce
- Cisco ISE Posture Services Design Guide
