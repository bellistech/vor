# 802.1X — Port-Based Authentication Architecture and EAP Framework

> *IEEE 802.1X defines a port-based network access control mechanism that leverages the Extensible Authentication Protocol (EAP) to authenticate devices at the link layer before granting network access. The architecture separates the roles of supplicant, authenticator, and authentication server, creating a trust chain rooted in cryptographic identity verification. Understanding the protocol state machines, EAP method internals, and deployment topology is essential for building zero-trust wired and wireless networks.*

---

## 1. The 802.1X Architecture — Three-Party Trust Model

### Role Separation

The 802.1X model enforces network access through three distinct entities that never collapse into one:

| Role | Function | Examples |
|:---|:---|:---|
| Supplicant | Entity seeking access; runs EAP client software | wpa_supplicant, Windows native, macOS native, AnyConnect NAM |
| Authenticator | Enforcement point; relays EAP between supplicant and server | Cisco switch, Aruba AP, Juniper EX |
| Authentication Server | Makes the access decision; holds identity store | Cisco ISE, FreeRADIUS, Microsoft NPS, Aruba ClearPass |

The authenticator is intentionally stateless with respect to credentials. It never sees passwords or private keys. It functions as a relay and enforcement point, translating between EAPOL (Layer 2) on the supplicant side and RADIUS (Layer 3/UDP) on the server side.

### Protocol Stack

```
Layer 2 (EAPOL)              Layer 3 (RADIUS over UDP)

Supplicant <----> Authenticator <----> Authentication Server
   |                   |                       |
   | EAP over LAN      | EAP encapsulated      |
   | (EtherType 0x888E) | in RADIUS attributes  |
   |                   | (UDP 1812/1813)        |
   |                   |                       |
   | No IP address     | IP required           |
   | required           | (management VLAN)     |
```

### EAPOL Frame Format (IEEE 802.1X-2020)

```
+------------------+------------------+---------+----------+---------+
| Dst MAC (6)      | Src MAC (6)      | EType   | Version  | Type    |
| 01:80:C2:00:00:03| Supplicant MAC   | 0x888E  | 0x03     | 0x00    |
+------------------+------------------+---------+----------+---------+
| Length (2)        | Packet Body (variable)                          |
+-----------------+---------------------------------------------------+

EAPOL Types:
  0x00  EAP-Packet       — carries EAP frames
  0x01  EAPOL-Start       — supplicant initiates authentication
  0x02  EAPOL-Logoff      — supplicant terminates session
  0x03  EAPOL-Key         — 802.11 4-way handshake key exchange
  0x04  EAPOL-ASF-Alert   — Alerting Standard Forum (legacy)
  0x05  MKA               — MACsec Key Agreement (802.1X-2010+)
```

### Destination MAC Address

The multicast address `01:80:C2:00:00:03` is the PAE (Port Access Entity) group address. Bridges must not forward this address, ensuring EAPOL frames remain local to the link between supplicant and authenticator. This is critical: EAPOL is not routable and cannot traverse Layer 3 boundaries.

---

## 2. EAP Framework — Method Negotiation and Execution

### EAP Packet Structure (RFC 3748)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Code      |  Identifier   |            Length             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Type      |  Type-Data ...
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Code: 1=Request, 2=Response, 3=Success, 4=Failure
Type: 1=Identity, 2=Notification, 3=NAK, 4=MD5-Challenge,
      13=EAP-TLS, 21=EAP-TTLS, 25=PEAP, 43=EAP-FAST
```

### EAP State Machine (RFC 3748 Section 4)

The EAP conversation follows a strict lock-step request/response model. The server always initiates requests; the supplicant only sends responses.

```
State Diagram (Supplicant):

    IDLE
     |
     v
  RECEIVED_REQUEST
     |
     +-- Type matches? ---> PROCESS_METHOD ---> SEND_RESPONSE ---> IDLE
     |
     +-- Type unknown? ---> SEND_NAK (propose alternative) ---> IDLE

State Diagram (Authenticator — Pass-Through):

    IDLE
     |
     v
  EAPOL_RECEIVED
     |
     +-- EAPOL-Start ---> Send EAP-Request/Identity to supplicant
     |
     +-- EAP-Response ---> Encapsulate in RADIUS ---> Forward to server
     |
     v
  RADIUS_RECEIVED
     |
     +-- Access-Challenge ---> Extract EAP ---> Send to supplicant
     +-- Access-Accept    ---> Send EAP-Success ---> Authorize port
     +-- Access-Reject    ---> Send EAP-Failure ---> Keep unauthorized
```

### EAP Method Comparison — Security Properties

| Property | EAP-TLS | PEAP | EAP-TTLS | EAP-FAST |
|:---|:---|:---|:---|:---|
| RFC | 5216 | Draft | 5281 | 7170 |
| Server Certificate | Required | Required | Required | Optional (PAC) |
| Client Certificate | Required | Not required | Not required | Not required |
| Inner Method | None (mutual TLS) | MSCHAPv2 / EAP-GTC | PAP / CHAP / MSCHAPv2 | MSCHAPv2 / GTC / TLS |
| Forward Secrecy | Yes (with ECDHE) | Yes (TLS tunnel) | Yes (TLS tunnel) | Depends on PAC provisioning |
| Identity Protection | Yes (encrypted) | Yes (inner identity) | Yes (inner identity) | Yes (inside tunnel) |
| Dictionary Attack Resistant | Yes (no password) | Depends on inner | Depends on inner | Depends on inner |
| Requires PKI | Full (CA + client) | Partial (CA + server) | Partial (CA + server) | Minimal (PAC or cert) |
| Windows Native Support | Yes | Yes | Win 8+ | No (AnyConnect NAM) |
| macOS Native Support | Yes | Yes | Yes | No |

---

## 3. EAP-TLS — Mutual Certificate Authentication

### Protocol Flow

EAP-TLS (RFC 5216) establishes a full TLS handshake inside the EAP conversation. Both parties present X.509 certificates, providing mutual authentication without transmitting any passwords.

```
Supplicant                     RADIUS Server

EAP-Response/Identity -------->
                     <-------- EAP-Request (EAP-TLS Start)
EAP-Response (ClientHello) --->
                     <-------- EAP-Request (ServerHello,
                               ServerCertificate,
                               CertificateRequest,
                               ServerHelloDone)
EAP-Response (ClientCertificate,
  ClientKeyExchange,
  CertificateVerify,
  ChangeCipherSpec,
  Finished)          -------->
                     <-------- EAP-Request (ChangeCipherSpec,
                               Finished)
EAP-Response (empty) -------->
                     <-------- EAP-Success + Access-Accept
```

### Certificate Requirements

```
Server Certificate:
  - Subject: CN=radius.example.com
  - Extended Key Usage: id-kp-serverAuth (1.3.6.1.5.5.7.3.1)
  - SAN: DNS:radius.example.com
  - Signed by trusted CA (supplicant must trust this CA)

Client Certificate:
  - Subject: CN=device-hostname or CN=username
  - Extended Key Usage: id-kp-clientAuth (1.3.6.1.5.5.7.3.2)
  - SAN: optional (can contain UPN for Windows)
  - Signed by CA that RADIUS server trusts

Certificate Validation:
  1. Chain of trust (root CA -> intermediate -> leaf)
  2. Expiration check (notBefore/notAfter)
  3. CRL or OCSP revocation check
  4. Extended Key Usage verification
  5. Optional: Subject/SAN matching against identity store
```

### Security Analysis

EAP-TLS is the only EAP method immune to credential theft because no passwords are involved. The security relies entirely on private key protection:

- Private key never leaves the device (stored in TPM, smart card, or keychain)
- Compromising the RADIUS server does not yield client credentials
- Each device has a unique identity (certificate serial number)
- Revocation is immediate via CRL/OCSP (no password change propagation delay)

The primary cost is PKI infrastructure: certificate enrollment, renewal, and revocation at scale.

---

## 4. PEAP — Protected EAP

### Two-Phase Authentication

PEAP creates a TLS tunnel using only the server certificate, then runs an inner EAP method (typically MSCHAPv2) inside that tunnel.

```
Phase 1: TLS Tunnel Establishment
  - Server presents certificate (validates identity)
  - Client verifies server cert against trusted CA list
  - TLS session established (AES-256, etc.)
  - Outer identity: "anonymous@example.com" (privacy)

Phase 2: Inner Authentication (inside tunnel)
  PEAPv0: MS-CHAPv2
    - Server sends challenge
    - Client sends NT-Hash response
    - Mutual auth via authenticator response
    - Session key derived for MPPE

  PEAPv1: EAP-GTC
    - Generic Token Card
    - One-time passwords, tokens
    - Common with ISE + LDAP backends
```

### PEAP Cryptobinding

Cryptobinding prevents man-in-the-middle attacks where an attacker terminates the outer TLS tunnel and relays the inner authentication to a different server. The Compound MAC (CMAC) ties the inner and outer TLS sessions together:

```
Cryptobinding TLV:
  - Outer TLS session key material
  - Inner EAP method session key
  - HMAC-SHA1-160 binding
  - Prevents tunnel teardown attacks

Without cryptobinding:
  Attacker <==TLS==> Supplicant
  Attacker ==relays inner auth==> Legitimate Server
  Result: Attacker gets access using relayed credentials

With cryptobinding:
  Inner key bound to outer TLS session
  Relay detected — authentication fails
```

---

## 5. RADIUS Interaction — The Glue Layer

### RADIUS Attributes for 802.1X

The authenticator and authentication server communicate using RADIUS (RFC 2865/2866). Key attributes for 802.1X:

| Attribute | Number | Direction | Purpose |
|:---|:---:|:---|:---|
| User-Name | 1 | Request | Identity from EAP-Response/Identity |
| NAS-IP-Address | 4 | Request | IP of the authenticator |
| NAS-Port | 5 | Request | Physical port number |
| Service-Type | 6 | Both | Framed (for 802.1X) = 2 |
| Framed-IP-Address | 8 | Accept | IP to assign (optional) |
| Filter-Id | 11 | Accept | ACL name to apply |
| NAS-Port-Type | 61 | Request | Ethernet (15) or Wireless-802.11 (19) |
| Tunnel-Type | 64 | Accept | VLAN (13) |
| Tunnel-Medium-Type | 65 | Accept | IEEE-802 (6) |
| Tunnel-Private-Group-Id | 81 | Accept | VLAN ID or name |
| EAP-Message | 79 | Both | Fragmented EAP payload |
| Message-Authenticator | 80 | Both | HMAC-MD5 of entire packet |
| NAS-Port-Id | 87 | Request | Interface name (e.g., GigabitEthernet1/0/1) |
| Calling-Station-Id | 31 | Request | Supplicant MAC address |
| Called-Station-Id | 30 | Request | Authenticator MAC (or SSID for wireless) |

### EAP Fragmentation in RADIUS

EAP messages (especially TLS certificate exchanges) can exceed the RADIUS maximum attribute size of 253 bytes. The EAP-Message attribute is repeated across multiple RADIUS attributes and reassembled:

```
RADIUS Access-Request:
  EAP-Message (79): bytes 0-252
  EAP-Message (79): bytes 253-505
  EAP-Message (79): bytes 506-758
  Message-Authenticator (80): HMAC-MD5 of full packet

Maximum RADIUS packet size: 4096 bytes
Maximum EAP payload per RADIUS packet: ~4000 bytes
Multiple RADIUS round-trips needed for large certificates
```

### Message-Authenticator (Attribute 80)

This attribute is mandatory for any RADIUS packet containing an EAP-Message attribute (RFC 3579). It is an HMAC-MD5 computed over the entire RADIUS packet using the shared secret:

```
Message-Authenticator = HMAC-MD5(shared_secret, RADIUS_packet)

Purpose:
  - Prevents attribute manipulation in transit
  - Authenticates the RADIUS packet source
  - Required by RFC 3579 for all EAP-bearing packets
  - Packets without it when required are silently dropped
```

---

## 6. Port States and the PAE State Machine

### Port Access Entity (PAE) States

The authenticator maintains a per-port state machine defined in IEEE 802.1X-2020 Clause 12:

```
                         +------------------+
                         |  INITIALIZE       |
                         | (port link up)    |
                         +--------+---------+
                                  |
                                  v
                         +------------------+
                         |  DISCONNECTED     |
                         | (waiting for      |
                         |  EAPOL-Start or   |
                         |  source MAC)      |
                         +--------+---------+
                                  |
                                  v
                         +------------------+
                         |  CONNECTING       |
                         | (send EAP-Req/ID) |
                         | (tx-period timer) |
                         +--------+---------+
                                  |
                          +-------+--------+
                          |                |
                    EAP-Response      Timeout (no response)
                          |                |
                          v                v
                   +-----------+    +-----------+
                   |AUTHENTICAT|    |GUEST VLAN |
                   |ING        |    |or HELD    |
                   |(EAP       |    |           |
                   | exchange) |    |           |
                   +-----+-----+    +-----------+
                         |
                   +-----+-----+
                   |           |
              Accept        Reject
                   |           |
                   v           v
            +----------+ +----------+
            |AUTHORIZED| |  HELD    |
            |(traffic  | |(quiet    |
            | flows)   | | period)  |
            +----------+ +----+-----+
                               |
                          quiet-period
                          expires
                               |
                               v
                        CONNECTING
                        (retry)
```

### Timers

| Timer | Default | Purpose |
|:---|:---:|:---|
| tx-period | 30s | Interval between EAP-Request/Identity retransmissions |
| quiet-period | 60s | Wait after failed auth before allowing retry |
| supp-timeout | 30s | Wait for supplicant response to EAP request |
| server-timeout | 30s | Wait for RADIUS server response |
| reauth-period | 3600s | Time between periodic reauthentications |
| max-reauth-req | 2 | Maximum EAP-Request/Identity retransmissions |
| max-req | 2 | Maximum EAP requests for a given type |

### Timer Tuning for Deployment

```
Scenario: Fast MAB fallback for non-802.1X devices
  dot1x timeout tx-period 10
  dot1x max-reauth-req 2
  Total wait before MAB: 10s x 3 attempts = 30s (down from 90s)

Scenario: Slow WAN RADIUS (satellite office)
  dot1x timeout server-timeout 60
  radius-server timeout 60
  radius-server retransmit 3

Scenario: Frequent reauthentication (high-security zone)
  authentication timer reauthenticate 600
  # Or use RADIUS-supplied: Session-Timeout attribute
  authentication timer reauthenticate server
```

---

## 7. VLAN Assignment Architecture

### Dynamic VLAN Assignment Flow

```
1. Supplicant authenticates via EAP
2. RADIUS server evaluates policy (identity, group, posture)
3. Access-Accept includes VLAN attributes:
     Tunnel-Type (64) = VLAN
     Tunnel-Medium-Type (65) = IEEE-802
     Tunnel-Private-Group-Id (81) = "100" or "CORPORATE"
4. Authenticator moves port from default VLAN to assigned VLAN
5. Supplicant receives new DHCP lease in assigned VLAN
6. Full network access granted per VLAN policy
```

### VLAN Types in 802.1X Deployments

| VLAN Type | Trigger | Purpose | Typical ID |
|:---|:---|:---|:---:|
| Data VLAN | Successful auth | Normal access for authenticated devices | 100-199 |
| Voice VLAN | CDP/LLDP phone detection | IP phone traffic (QoS marked) | 200-299 |
| Guest VLAN | No 802.1X supplicant | Limited access for dumb devices | 999 |
| Auth-Fail VLAN | RADIUS Access-Reject | Remediation / limited access | 998 |
| Critical VLAN | RADIUS unreachable | Business continuity during outage | 997 |
| Quarantine VLAN | Posture non-compliant | Restricted access for remediation | 996 |

### VLAN Assignment Precedence

```
1. RADIUS-assigned VLAN (dynamic — highest priority)
2. Authentication-fail VLAN (if auth rejected)
3. Critical VLAN (if RADIUS unreachable)
4. Guest VLAN (if no supplicant detected)
5. Interface access VLAN (static default — lowest priority)

Critical requirement: The assigned VLAN must exist in the switch's
VLAN database. If the RADIUS-assigned VLAN does not exist on the
switch, authentication succeeds but the port remains in the
configured access VLAN. This is a common deployment error.
```

---

## 8. MAB — MAC Authentication Bypass

### How MAB Works

MAB is a fallback mechanism for devices that lack an 802.1X supplicant (printers, cameras, IoT sensors). The authenticator uses the device's MAC address as both the username and password in a RADIUS request.

```
1. Device connects (link up)
2. Authenticator sends EAP-Request/Identity (EAPOL)
3. No EAP-Response received (tx-period expires)
4. After max-reauth-req timeouts, authenticator falls back to MAB
5. Authenticator sends RADIUS Access-Request:
     User-Name = "aabbccddeeff"  (MAC address)
     User-Password = "aabbccddeeff"
     Service-Type = Call-Check (10)
     NAS-Port-Type = Ethernet (15)
6. RADIUS server looks up MAC in endpoint database
7. Access-Accept with VLAN/ACL or Access-Reject
```

### MAC Format Variations

```
Format              Example              Common With
No separator        aabbccddeeff         Cisco default
Colon-separated     aa:bb:cc:dd:ee:ff    FreeRADIUS default
Hyphen-separated    aa-bb-cc-dd-ee-ff    Microsoft NPS
Dot-separated       aabb.ccdd.eeff       Cisco CLI display
Uppercase           AABBCCDDEEFF         Some RADIUS servers

IOS-XE format control:
  mab request format attribute 1 groupsize 2 separator : lowercase
  # Sends: aa:bb:cc:dd:ee:ff

Critical: The format configured on the switch MUST match what the
RADIUS server expects. A mismatch means MAB will always fail because
the username won't match any endpoint entry.
```

### MAB Security Considerations

MAB is inherently weaker than 802.1X because MAC addresses can be spoofed. Mitigations:

- Enable IP Source Guard (IPSG) to bind MAC to IP after auth
- Use DHCP snooping to prevent static IP assignment
- Enable Dynamic ARP Inspection (DAI) to prevent ARP spoofing
- Apply restrictive dACLs to MAB-authenticated ports
- Use RADIUS profiling (DHCP fingerprint, HTTP User-Agent) to validate device type
- Monitor for MAC flapping and duplicate MAC addresses
- Consider MACsec (802.1AE) for link-layer encryption where available

---

## 9. Deployment Phasing — Monitor, Low-Impact, Closed

### Phase 1: Monitor Mode

Monitor mode is the initial deployment phase where 802.1X is enabled but enforcement is disabled. All traffic is permitted regardless of authentication result.

```
Key Configuration:
  authentication open          ← permits all traffic pre/post-auth
  authentication port-control auto

Behavior:
  - 802.1X and MAB run normally
  - Authentication results are logged
  - Traffic flows whether auth succeeds, fails, or never starts
  - No impact to production

Purpose:
  - Discover which devices support 802.1X
  - Identify devices that need MAB entries
  - Verify RADIUS connectivity and policy
  - Build endpoint database in ISE
  - Typical duration: 2-4 weeks

Monitoring:
  show authentication sessions    ← verify devices authenticating
  show authentication sessions method dot1x  ← 802.1X-capable devices
  show authentication sessions method mab    ← MAB devices (need entries)
  show authentication sessions status unauthorized  ← failures to investigate
```

### Phase 2: Low-Impact Mode

Low-impact mode introduces partial enforcement via pre-authentication ACLs while still allowing basic network services.

```
Key Configuration:
  authentication open              ← still permits traffic
  ip access-group PRE-AUTH-ACL in  ← but filtered by ACL

Pre-Auth ACL (applied before authentication):
  permit udp any any eq bootps     ← DHCP
  permit udp any any eq bootpc
  permit udp any any eq domain     ← DNS
  permit icmp any any              ← ICMP for troubleshooting
  permit tcp any host 10.0.0.1 eq 80  ← captive portal
  deny   ip any any               ← block everything else

Post-Auth: RADIUS returns a downloadable ACL (dACL) or the pre-auth
ACL is replaced by a permit-all. Authenticated devices get full access;
unauthenticated devices get only DHCP, DNS, and portal.

Purpose:
  - Test enforcement with safety net
  - Users get basic connectivity even if auth fails
  - Identify policy gaps before full lockdown
  - Typical duration: 2-4 weeks
```

### Phase 3: Closed Mode

Closed mode is full enforcement. No traffic flows until authentication succeeds.

```
Key Configuration:
  authentication port-control auto   ← no "authentication open"
  # (absence of "authentication open" = closed mode)

Behavior:
  - Port blocks ALL traffic until Access-Accept
  - Only EAPOL frames permitted pre-authentication
  - Failed auth = no network access
  - Guest VLAN, auth-fail VLAN, critical VLAN provide safety nets

Prerequisites before enabling:
  □ All devices identified (802.1X, MAB, or exception)
  □ Guest VLAN configured for unknown devices
  □ Auth-fail VLAN configured for failed authentications
  □ Critical VLAN configured for RADIUS outages
  □ RADIUS server redundancy verified
  □ Help desk trained on 802.1X troubleshooting
  □ Exception process documented (for new devices, visitors)
```

---

## 10. Change of Authorization (CoA) — RFC 5176

### CoA Architecture

CoA inverts the traditional RADIUS model. Instead of the NAS initiating requests, the RADIUS server pushes changes to the NAS mid-session.

```
Traditional RADIUS:
  NAS ---Request---> RADIUS Server
  NAS <--Accept/Reject--- RADIUS Server

CoA (RFC 5176):
  RADIUS Server ---CoA-Request---> NAS (UDP 3799)
  RADIUS Server <--CoA-ACK/NAK--- NAS

Disconnect (RFC 5176):
  RADIUS Server ---Disconnect-Request---> NAS
  RADIUS Server <--Disconnect-ACK/NAK--- NAS
```

### CoA Message Types

| Message | Code | Direction | Purpose |
|:---|:---:|:---|:---|
| CoA-Request | 43 | Server -> NAS | Change session attributes |
| CoA-ACK | 44 | NAS -> Server | Change applied |
| CoA-NAK | 45 | NAS -> Server | Change failed (with error cause) |
| Disconnect-Request | 40 | Server -> NAS | Terminate session |
| Disconnect-ACK | 41 | NAS -> Server | Session terminated |
| Disconnect-NAK | 42 | NAS -> Server | Termination failed |

### CoA Use Cases

```
Posture Reassessment:
  1. Device authenticates — placed in quarantine VLAN
  2. Posture agent checks compliance (AV, patches, encryption)
  3. Agent reports to ISE: device compliant
  4. ISE sends CoA-Request: move to production VLAN, apply full dACL
  5. Switch reauthenticates, gets new authorization
  6. Device now has full access

Guest Self-Registration:
  1. Guest connects — MAB fails, WebAuth redirect
  2. Guest registers on portal, sponsor approves
  3. ISE sends CoA-Request: apply guest access policy
  4. Switch reauthenticates via RADIUS
  5. Guest gets internet-only access

Threat Response:
  1. IDS/IPS detects malicious traffic from endpoint
  2. Alert sent to ISE via pxGrid/syslog
  3. ISE sends Disconnect-Request to quarantine endpoint
  4. Switch terminates session, port moves to quarantine VLAN
  5. Incident response team investigates
```

### CoA Session Identification

The CoA-Request must identify which session to modify. Common session identification attributes:

```
Session Identification Methods:
  1. Acct-Session-Id (44)       ← unique per session (preferred)
  2. Calling-Station-Id (31)    ← MAC address of supplicant
  3. NAS-Port (5)               ← physical port number
  4. Audit-Session-Id (Cisco)   ← Cisco vendor-specific

IOS-XE requires at least one of these to match an active session.
If no match found, CoA-NAK returned with Error-Cause = 503 (Session
Context Not Found).
```

---

## 11. Wired vs. Wireless 802.1X

### Architectural Differences

| Aspect | Wired (802.3) | Wireless (802.11) |
|:---|:---|:---|
| Physical port | One port per device | Shared radio, virtual ports per client |
| EAPOL delivery | Direct Ethernet | Over-the-air (encrypted after 4-way handshake) |
| Authenticator | Switch | Wireless LAN Controller (WLC) or AP |
| Pre-auth state | Port blocked | Association allowed, traffic blocked |
| Key derivation | Optional (MACsec) | Required (PTK for unicast, GTK for broadcast) |
| Host modes | Single/Multi/Multi-Domain/Multi-Auth | Per-client (each STA authenticates) |
| VLAN assignment | Port VLAN change | Per-client VLAN (or tunnel mode) |
| Roaming | N/A | PMK caching, OKC, 802.11r (FT) |

### Wireless 802.1X Key Hierarchy

```
MSK (Master Session Key)
  ← derived from EAP method (EAP-TLS, PEAP, etc.)
  ← sent from RADIUS to WLC in MS-MPPE-Send-Key / MS-MPPE-Recv-Key

PMK (Pairwise Master Key)
  ← derived from MSK (first 256 bits)
  ← cached for fast reconnection

PTK (Pairwise Transient Key)
  ← derived from 4-way handshake:
     PTK = PRF(PMK, "Pairwise key expansion",
               Min(AA,SPA) || Max(AA,SPA) ||
               Min(ANonce,SNonce) || Max(ANonce,SNonce))

  PTK components:
    KCK (Key Confirmation Key)   — 128 bits — MIC in EAPOL-Key frames
    KEK (Key Encryption Key)     — 128 bits — encrypts GTK delivery
    TK  (Temporal Key)           — 128/256 bits — encrypts data frames

GTK (Group Temporal Key)
  ← encrypts broadcast/multicast traffic
  ← distributed from AP to clients via EAPOL-Key (encrypted with KEK)
```

### Fast Roaming Mechanisms

```
PMK Caching (standard):
  - WLC caches PMK after initial authentication
  - Client reconnects: skip EAP, go directly to 4-way handshake
  - Limitation: only works with same WLC

OKC (Opportunistic Key Caching):
  - PMK shared across APs in same mobility group
  - PMKID = HMAC-SHA1-128(PMK, "PMK Name" || AA || SPA)
  - Client includes PMKID in (Re)Association Request
  - If AP has the PMK, skip EAP

802.11r (Fast BSS Transition):
  - Over-the-air and over-the-DS transitions
  - FT key hierarchy: R0KH -> R1KH -> PMK-R1
  - Authentication occurs with target AP before roaming
  - Reduces roam time to < 50ms (voice/video requirement)
```

---

## 12. ISE Policy Architecture

### Policy Sets

ISE evaluates policies in a hierarchical structure:

```
Policy Set (top level — matches on conditions like Wired_802.1X)
  |
  +-- Authentication Policy
  |     |
  |     +-- Rule 1: IF Wired_802.1X THEN use Active Directory
  |     +-- Rule 2: IF Wired_MAB THEN use Internal Endpoints
  |     +-- Rule 3: IF Wireless_802.1X THEN use Active Directory
  |
  +-- Authorization Policy
        |
        +-- Rule 1: IF AD:Group=DomainComputers AND Compliant
        |            THEN PermitAccess, VLAN=100, dACL=FULL
        |
        +-- Rule 2: IF AD:Group=BYOD AND Registered
        |            THEN PermitAccess, VLAN=200, dACL=LIMITED
        |
        +-- Rule 3: IF EndpointProfile=Cisco-IP-Phone
        |            THEN PermitAccess, VLAN=Voice, dACL=VOICE-ACL
        |
        +-- Rule 4: IF MAB AND EndpointProfile=Printer
        |            THEN PermitAccess, VLAN=IOT, dACL=PRINT-ONLY
        |
        +-- Rule 5: IF PostureStatus=NonCompliant
        |            THEN Quarantine, VLAN=996, dACL=REMEDIATION
        |
        +-- Default: DenyAccess
```

### Authorization Profiles

```
Authorization Profile: FULL-ACCESS
  VLAN: 100
  dACL: permit ip any any
  Reauthentication Timer: 28800 (8 hours)
  CoA Action: Reauthenticate

Authorization Profile: QUARANTINE
  VLAN: 996
  dACL:
    permit udp any any eq domain          ← DNS
    permit udp any any eq bootps          ← DHCP
    permit tcp any host 10.0.0.50 eq 8443 ← remediation portal
    deny ip any any
  Reauthentication Timer: 300 (5 minutes)
  CoA Action: Reauthenticate
  Web Redirection: Client Provisioning Portal

Authorization Profile: GUEST-REDIRECT
  VLAN: 999
  dACL:
    permit udp any any eq domain
    permit udp any any eq bootps
    deny ip any any
  Web Redirection (CWA): Guest Portal
  CoA Action: Terminate CoA
```

### Profiling

ISE profiling identifies endpoint types to apply appropriate policy even without 802.1X credentials:

```
Profiling Probes:
  RADIUS    — Calling-Station-Id (MAC OUI), NAS-Port-Type
  DHCP      — Options 12 (hostname), 55 (parameter list), 60 (vendor class)
  HTTP      — User-Agent string
  DNS       — Reverse lookup
  SNMP      — Device MIBs (CDP, LLDP, ARP table)
  NetFlow   — Traffic patterns
  NMAP      — Active scanning (OS fingerprint, open ports)

Profile Match:
  Certainty Factor = sum of matching attributes
  Each probe contributes a weighted score
  Minimum Certainty Factor required for profile assignment
  Example: Cisco-IP-Phone-7945 = OUI(10) + CDP(20) + DHCP(20) = 50
```

---

## 13. MACsec (802.1AE) Integration with 802.1X

### MACsec with 802.1X-2010

802.1X-2010 introduced MACsec Key Agreement (MKA) as an extension to 802.1X. After EAP authentication, the derived session keys bootstrap MACsec encryption at Layer 2.

```
Key Derivation for MACsec:

MSK (from EAP method)
  |
  v
CAK (Connectivity Association Key)
  ← derived from MSK
  ← shared between supplicant and authenticator
  |
  v
SAK (Secure Association Key)
  ← derived by Key Server (highest priority MKA participant)
  ← distributed via MKPDU (MKA Protocol Data Unit)
  ← rotated periodically
  |
  v
MACsec Encryption (AES-GCM-128 or AES-GCM-256)
  ← encrypts all frames on the link
  ← provides confidentiality + integrity + replay protection
```

### IOS-XE MACsec Configuration

```
! MACsec with 802.1X (uplink to distribution switch)
key chain MACSEC-KEYCHAIN macsec
  key 01
    cryptographic-algorithm aes-256-cmac
    key-string 7 <encrypted-key>
    lifetime local 00:00:00 Jan 1 2025 infinite

interface TenGigabitEthernet1/0/1
  macsec
  mka policy MKA-POLICY
  mka pre-shared-key key-chain MACSEC-KEYCHAIN

! MACsec policy
mka policy MKA-POLICY
  macsec-cipher-suite gcm-aes-256
  confidentiality-offset 0
  key-server priority 0
  sak-rekey-interval 3600
```

---

## 14. Scalability and High Availability

### RADIUS Server Redundancy

```
Redundancy Patterns:

Active/Standby:
  radius server ISE-PRIMARY
    address ipv4 10.1.1.100
  radius server ISE-SECONDARY
    address ipv4 10.1.1.101

  aaa group server radius ISE
    server name ISE-PRIMARY
    server name ISE-SECONDARY
    deadtime 15           ← mark dead for 15 minutes after failure

Load Balancing:
  aaa group server radius ISE
    server name ISE-PRIMARY
    server name ISE-SECONDARY
    load-balance method least-outstanding
    # Distributes requests to server with fewest pending requests

Automate-Tester (probe dead server for recovery):
  radius server ISE-PRIMARY
    automate-tester username probe-user probe-on
    # Sends periodic test auth to detect server recovery
    # Faster than waiting for deadtime to expire
```

### Scale Considerations

```
Per-Switch Limits:
  - Maximum concurrent auth sessions: platform-dependent
    Catalyst 9300: 16,384 sessions
    Catalyst 3850: 8,192 sessions
  - RADIUS timeout x retransmit x sessions = peak RADIUS load
  - Enable RADIUS accounting for session tracking

ISE Scale (per node):
  - Policy Service Node (PSN): ~20,000 concurrent sessions
  - 5 PSN cluster: ~100,000 concurrent sessions
  - RADIUS auth rate: ~1,500 authentications/second per PSN
  - Profiling endpoint database: 2,000,000 endpoints per deployment

Network Impact:
  - EAPOL is local to the link (not routed)
  - RADIUS traffic traverses management network
  - CoA traffic: relatively low volume, high importance
  - Plan RADIUS source-interface for reachability
```

---

## Prerequisites

- RADIUS (Remote Authentication Dial-In User Service) — shared secret, attributes, accounting
- TLS — certificate validation, cipher negotiation, key derivation
- PKI — certificate authority, enrollment, revocation (especially for EAP-TLS)
- Ethernet (IEEE 802.3) — frame format, VLAN tagging (802.1Q)
- VLAN fundamentals — trunking, access ports, inter-VLAN routing
- AAA concepts — authentication, authorization, accounting model
- Active Directory / LDAP — identity stores for user/machine authentication

## References

- IEEE 802.1X-2020 — Port-Based Network Access Control
- IEEE 802.1AE-2018 — MACsec: MAC Security
- RFC 3748 — Extensible Authentication Protocol (EAP)
- RFC 3579 — RADIUS Support for EAP
- RFC 5216 — EAP-TLS Authentication Protocol
- RFC 5281 — Extensible Authentication Protocol Tunneled TLS (EAP-TTLS)
- RFC 7170 — Tunnel Extensible Authentication Protocol (TEAP / EAP-FAST successor)
- RFC 2865 — Remote Authentication Dial-In User Service (RADIUS)
- RFC 2866 — RADIUS Accounting
- RFC 5176 — Dynamic Authorization Extensions to RADIUS (CoA)
- RFC 4017 — EAP Method Requirements for Wireless LANs
- Cisco Identity-Based Networking Services Configuration Guide (IOS-XE 17.x)
- Cisco ISE 3.x Administrator Guide
- Juniper 802.1X Authentication Configuration Guide
- Aruba ClearPass Policy Manager Documentation
- Microsoft NPS with 802.1X — https://learn.microsoft.com/en-us/windows-server/networking/technologies/nps/nps-top
