# Network Services — Theory and Protocol Internals

> *Network services are the invisible substrate that makes IP networks usable. DHCP turns raw connectivity into configured hosts. NTP turns independent oscillators into a coherent timebase. AAA turns open ports into controlled access points. SNMP turns opaque boxes into observable systems. Syslog turns ephemeral events into auditable history. Each protocol solves a coordination problem under constraints of latency, unreliability, and adversarial conditions.*

---

## 1. DHCP DORA Process and Relay Theory

### The DORA State Machine

DHCP address acquisition is a four-message exchange that implements a distributed resource allocation protocol. The client has no IP address and cannot be reached by unicast, so the protocol bootstraps communication through broadcast.

```
State Machine:

  INIT ──DHCPDISCOVER(broadcast)──> SELECTING
           │                              │
           │                    DHCPOFFER received
           │                              │
           v                              v
  SELECTING ──DHCPREQUEST(broadcast)──> REQUESTING
           │                              │
           │                    DHCPACK received
           │                              │
           v                              v
  REQUESTING ────────────────────────> BOUND
                                          │
                            T1 expires    │
                                          v
                                      RENEWING ──DHCPREQUEST(unicast)──> BOUND
                                          │                   (on ACK)
                            T2 expires    │
                                          v
                                      REBINDING ──DHCPREQUEST(bcast)──> BOUND
                                          │                   (on ACK)
                            Lease expires │
                                          v
                                        INIT (start over)
```

**Why broadcast for DHCPREQUEST?** The client broadcasts its REQUEST (even though it already knows the server) so that all servers that sent OFFERs can see which server was chosen. The non-selected servers withdraw their offers and return those addresses to their pools. This prevents address leakage in multi-server environments.

**Why broadcast for DHCPDISCOVER?** The client has no IP address (source 0.0.0.0) and does not know any server address. The only option is Layer 2 broadcast (destination 255.255.255.255), which is translated to Ethernet broadcast (FF:FF:FF:FF:FF:FF).

### Transaction ID and Race Conditions

Each DORA exchange is identified by a 32-bit transaction ID (xid) chosen by the client. The xid binds all four messages together:

```
xid = random_32bit()

DISCOVER.xid = xid
OFFER.xid    = xid     (server echoes it back)
REQUEST.xid  = xid
ACK.xid      = xid

The client ignores any message where response.xid != pending.xid.
```

In environments with multiple DHCP servers, the client may receive multiple OFFERs. RFC 2131 specifies that the client SHOULD select the first OFFER received, but implementations may use server preference or other criteria. The selected server is identified in the DHCPREQUEST via option 54 (Server Identifier).

### Relay Agent Theory

DHCP relies on Layer 2 broadcast, which routers do not forward. In routed networks, a relay agent bridges this gap by converting broadcasts to unicasts.

**The giaddr mechanism:**

```
1. Client broadcasts DHCPDISCOVER on its local subnet
2. Relay agent receives the broadcast on its local interface
3. Relay agent:
   a. Sets giaddr = IP address of the interface where the broadcast arrived
   b. Increments the hops field
   c. Forwards the packet as unicast to the configured DHCP server
4. Server receives the unicast packet and examines giaddr
5. Server uses giaddr to select the appropriate address pool:
   - If giaddr falls within subnet 10.1.1.0/24, allocate from that pool
   - If giaddr matches no configured subnet, silently drop
6. Server sends OFFER/ACK to the relay agent (unicast to giaddr)
7. Relay agent delivers the response to the client:
   - If broadcast flag is set: Layer 2 broadcast on the client's subnet
   - If broadcast flag is clear: unicast to the client's assigned address
     (using ARP or the chaddr field for Layer 2 addressing)
```

**Multi-hop relay:** When multiple relay agents exist in a path (relay chaining), only the first relay sets giaddr. Subsequent relays increment the hops field and may append Option 82 sub-options, but they do not overwrite giaddr. The server always uses the first relay's giaddr for pool selection.

**Option 82 (Relay Agent Information) internals:**

Option 82 carries metadata about the physical location of the client, inserted by the relay agent (or the switch in DHCP snooping scenarios).

```
Option 82 structure:
  Type: 82 (0x52)
  Length: total length of all sub-options
  Sub-options:
    Sub-option 1 — Circuit ID:
      Identifies the physical port and VLAN
      Format is vendor-specific, e.g., "Gi0/1:VLAN100"
      Used for: per-port IP assignment, location tracking

    Sub-option 2 — Remote ID:
      Identifies the relay device itself
      Typically the relay's MAC address or a configured string
      Used for: multi-switch environments, audit trails

    Sub-option 5 — Link Selection (RFC 3527):
      Override giaddr for pool selection
      Allows relay to use one IP for communication with the server
      but select a pool based on a different subnet

    Sub-option 9 — Vendor-Specific Information:
      Vendor-defined data carried through the relay
```

The DHCP server must echo Option 82 back in its response unchanged. The relay agent strips Option 82 before forwarding the response to the client. If the server modifies or drops Option 82, the relay agent may reject the response.

### DHCP Snooping — Layer 2 Security Theory

DHCP snooping operates at the switch level, classifying ports as trusted (uplinks to legitimate servers) or untrusted (access ports). It builds a binding table that maps MAC addresses to IP addresses, providing the foundation for DAI and IP Source Guard.

```
Snooping decision logic per DHCP message type:

Trusted port:
  All DHCP messages are forwarded (DISCOVER, OFFER, REQUEST, ACK, NAK)

Untrusted port:
  DHCPDISCOVER  → Forward (client is allowed to discover)
  DHCPREQUEST   → Forward (client is allowed to request)
  DHCPRELEASE   → Forward only if MAC+IP matches binding table
  DHCPDECLINE   → Forward

  DHCPOFFER     → DROP (only legitimate servers should offer)
  DHCPACK       → DROP (only legitimate servers should acknowledge)
  DHCPNAK       → DROP (only legitimate servers should NAK)

Binding table entry created on DHCPACK passing through trusted port:
  { MAC, IP, VLAN, Interface, Lease-Time, Expiry }
```

---

## 2. NTP Clock Synchronization Algorithms

### The NTP Timestamp and Offset Calculation

NTP measures the offset between a client clock and a server clock using four timestamps from a single request-response exchange:

```
Client                          Server
  |                               |
  |  t1 (originate timestamp)     |
  |  ──── NTP Request ──────────> |
  |                               | t2 (receive timestamp)
  |                               | t3 (transmit timestamp)
  |  <──── NTP Response ───────── |
  |  t4 (destination timestamp)   |
  |                               |

Clock offset (theta):
  theta = ((t2 - t1) + (t3 - t4)) / 2

Round-trip delay (delta):
  delta = (t4 - t1) - (t3 - t2)

Where:
  t1 = time client sent request (client's clock)
  t2 = time server received request (server's clock)
  t3 = time server sent response (server's clock)
  t4 = time client received response (client's clock)
```

**Derivation of the offset formula:** Let theta be the true clock offset (server_time - client_time) and d_up, d_down be the one-way network delays (unknown individually).

```
t2 = t1 + d_up + theta        (server receive = client send + delay + offset)
t4 = t3 + d_down - theta      (client receive = server send + delay - offset)

Adding both equations:
t2 + t4 = t1 + t3 + d_up + d_down
d_up + d_down = (t2 - t1) + (t4 - t3) = delta

Subtracting the second from the first:
t2 - t4 = t1 - t3 + d_up - d_down + 2*theta

If d_up = d_down (symmetric delay assumption):
2*theta = (t2 - t1) + (t3 - t4)
theta = ((t2 - t1) + (t3 - t4)) / 2
```

The critical assumption is symmetric network delay. When paths are asymmetric (different routes, different queue depths), the offset estimate contains an error of (d_up - d_down) / 2. This is the fundamental limitation of NTP and the reason PTP was developed.

### Marzullo's Algorithm

NTP uses Marzullo's algorithm (and its refinement by Mills) to select the best time estimate from multiple sources, each with an associated error interval.

**The problem:** Given n time sources, each reporting an offset theta_i with an error bound epsilon_i, find the smallest interval that contains the true time according to the majority of sources.

```
Each source i provides an interval [theta_i - epsilon_i, theta_i + epsilon_i]

Marzullo's algorithm:
1. Create a sorted list of all interval endpoints:
   For each source i, add:
     (theta_i - epsilon_i, +1)     ← interval start
     (theta_i + epsilon_i, -1)     ← interval end

2. Walk the sorted list, maintaining a running count:
   count = 0
   for each (value, type) in sorted order:
     count += type
     Track the maximum count reached

3. The optimal interval is where count >= ceil(n/2) + 1
   (majority agreement)

Example with 4 sources:
  Source A: [10ms, 30ms]
  Source B: [15ms, 25ms]
  Source C: [12ms, 28ms]
  Source D: [80ms, 120ms]    ← falseticker (outlier)

  Sorted endpoints: 10(+1), 12(+1), 15(+1), 25(-1), 28(-1), 30(-1), 80(+1), 120(-1)
  Counts:            1       2       3       2       1       0       1        0

  Maximum agreement = 3 sources in [15ms, 25ms]
  Source D is identified as a falseticker
  Best estimate: midpoint of [15ms, 25ms] = 20ms
```

NTP's implementation extends Marzullo's algorithm with additional heuristics: root distance (accumulated error from stratum 0), root dispersion (accumulated statistical error), and peer jitter.

### Clock Discipline Algorithm

NTP does not simply set the clock to the computed offset. Instead, it uses a hybrid phase-locked loop / frequency-locked loop (PLL/FLL) algorithm called the "clock discipline."

```
The discipline operates in two modes:

1. Phase-Locked Loop (PLL) — for small, slow-changing offsets:
   - Adjusts clock phase (time) to track the reference
   - Uses a feedback loop with time constant tau (adaptive, 4 to 36 hours)
   - Offset correction: slew the clock at a controlled rate
   - Maximum slew rate: 500 ppm (0.5 ms/s)

2. Frequency-Locked Loop (FLL) — for large initial offsets or frequency drift:
   - Adjusts clock frequency to match the reference
   - Faster convergence for large offsets
   - Used during initial synchronization

Clock states:
  NSET  → Initial state, no time set
  FSET  → Frequency set from drift file, no time samples yet
  SPIK  → Large offset detected, may be a spike (wait for confirmation)
  FREQ  → Measuring frequency offset
  SYNC  → Locked, PLL/FLL actively disciplining

Step vs Slew decision:
  |offset| > 128 ms  → Step the clock (instant jump)
  |offset| <= 128 ms → Slew the clock (gradual adjustment)
  |offset| > 1000s   → Panic, refuse to set (unless -g flag)

The 128ms threshold (step threshold) is configurable:
  tinker step 0.1     # step if offset > 100ms
  tinker panic 0      # disable panic threshold (always correct)

Frequency drift is saved to the drift file (e.g., /var/lib/ntp/drift)
on shutdown and loaded on startup to reduce initial convergence time.
The drift value is in PPM (parts per million).
Typical crystal oscillator drift: 10-100 PPM.
```

### NTP Security: KoD and NTS

**Kiss-of-Death (KoD):** When a server is overloaded or the client is misbehaving, the server sends a KoD packet with stratum 0 and a four-character ASCII code in the reference ID field:

```
KoD codes:
  DENY  — Access denied by server ACL
  RSTR  — Access restricted (rate limited)
  RATE  — Client is polling too frequently
  INIT  — Association not yet initialized (kiss-of-death on first packet)

Client behavior on KoD:
  DENY/RSTR → Stop querying this server permanently
  RATE      → Increase poll interval exponentially
```

**Network Time Security (NTS, RFC 8915):** NTS adds cryptographic authentication to NTP without the key distribution problems of symmetric key authentication.

```
NTS protocol phases:

1. NTS-KE (Key Establishment) — TCP/TLS:
   - Client connects to NTS-KE server via TLS 1.3
   - Negotiates AEAD algorithm (default AES-SIV-CMAC-256)
   - Server provides: C2S key, S2C key, NTS cookies
   - Cookies are opaque to the client (encrypted by server)

2. NTP with NTS — UDP/123:
   - Client includes NTS extension fields in NTP packets:
     a. Unique Identifier (prevents replay)
     b. NTS Cookie (one per request, server decrypts to recover keys)
     c. NTS Authenticator (AEAD tag over the NTP packet)
   - Server validates the authenticator using keys from the cookie
   - Server includes new cookies in the response (cookie renewal)

Security properties:
  - No pre-shared keys required
  - Forward secrecy via TLS 1.3
  - Replay protection via unique identifier
  - No key management burden (cookies are self-contained)
  - Server is stateless (all state is in the cookie)
```

---

## 3. AAA Framework Architecture

### Method List Evaluation

AAA in Cisco IOS uses method lists to define the sequence of authentication, authorization, and accounting methods to try. Understanding the evaluation order is critical for both security and availability.

```
Method list evaluation rules:

aaa authentication login MY_LIST group tacacs+ group radius local

Evaluation proceeds left to right:
  1. Try TACACS+ server group
     - Server responds with ACCEPT → user authenticated, STOP
     - Server responds with REJECT → user denied, STOP
     - Server unreachable (timeout) → try next method
     - Server error → try next method

  2. Try RADIUS server group
     - Same logic: ACCEPT=stop, REJECT=stop, timeout=next

  3. Try local database
     - Username/password match → authenticated
     - No match → denied (final method, no fallback)

CRITICAL DISTINCTION:
  "REJECT" ≠ "ERROR"
  - REJECT (server says "wrong password") → STOP, deny access
  - ERROR (server unreachable)            → FALL THROUGH to next method

This means:
  - If TACACS+ is reachable and rejects → RADIUS and local are NOT tried
  - If TACACS+ is unreachable → RADIUS is tried
  - If both external servers are down → local database is tried
  - If all methods fail → access denied
```

**The "none" method:** Adding `none` as the last method grants access with no authentication if all servers are unreachable. This is a security risk but may be configured on console lines to prevent total lockout:

```
aaa authentication login CONSOLE_SAFE group tacacs+ local none

! "none" only triggers if TACACS+ is down AND local DB has no match
! This is a last-resort lockout prevention, not a security best practice
```

### Authorization Method Lists

Authorization determines what an authenticated user is allowed to do. TACACS+ provides granular command authorization; RADIUS does not.

```
Authorization flow for command execution:

User types: "show running-config"
  │
  ├─ IOS checks: aaa authorization commands 15 default group tacacs+
  │
  ├─ IOS sends to TACACS+ server:
  │    {
  │      user: "admin",
  │      service: "shell",
  │      cmd: "show",
  │      cmd-arg: "running-config"
  │    }
  │
  ├─ TACACS+ server evaluates against its policy:
  │    permit user=admin cmd=show .*                    → PASS
  │    deny   user=netops cmd=configure .*              → FAIL
  │    permit user=netops cmd=show (interfaces|version) → PASS
  │
  └─ Server returns PASS/FAIL
     PASS → command executes
     FAIL → "Command authorization failed"

EXEC authorization (determines privilege level at login):
  aaa authorization exec default group tacacs+ local

  TACACS+ server returns AV pairs:
    priv-lvl=15        → user gets enable-level access immediately
    priv-lvl=1         → user gets user-level access
    acl=10             → apply access-list 10 to this session
    autocmd=show ver   → auto-execute command and disconnect
```

### Accounting Records

Accounting provides a complete audit trail of user activity. TACACS+ and RADIUS send accounting records to the server at different points in the session lifecycle.

```
Accounting record types:

START record — sent when:
  - User logs in (exec accounting)
  - Command is executed (command accounting)
  - Network connection starts (network accounting)

  Contains: username, timestamp, source IP, terminal line,
            authentication method, privilege level

STOP record — sent when:
  - User logs out
  - Command completes
  - Network connection ends

  Contains: everything from START plus: duration, bytes in/out,
            disconnect reason, commands executed

INTERIM record — periodic updates during long sessions:
  aaa accounting update periodic 15    ! update every 15 minutes

Example TACACS+ accounting record:
  Date/Time      User      Source       Action
  2026-04-05     admin     10.0.1.50   start shell
  2026-04-05     admin     10.0.1.50   cmd: show running-config
  2026-04-05     admin     10.0.1.50   cmd: configure terminal
  2026-04-05     admin     10.0.1.50   cmd: interface gi0/1
  2026-04-05     admin     10.0.1.50   cmd: shutdown
  2026-04-05     admin     10.0.1.50   stop shell elapsed=342s
```

### TACACS+ vs RADIUS Protocol Internals

```
TACACS+ packet structure:
  ┌──────────────────────────────────────┐
  │ Header (12 bytes, always cleartext)  │
  │   Major version: 0xC0               │
  │   Minor version: 0x00 or 0x01       │
  │   Type: AUTHEN(1), AUTHOR(2), ACCT(3)│
  │   Seq number (odd=client, even=server)│
  │   Flags: encrypted(0) or clear(1)    │
  │   Session ID (random, ties msgs)     │
  │   Length of body                      │
  ├──────────────────────────────────────┤
  │ Body (encrypted with shared key)     │
  │   Encryption: MD5 hash chain         │
  │   pseudo_pad = MD5(session_id +      │
  │     key + version + seq_no +         │
  │     prev_hash)                       │
  │   ciphertext = body XOR pseudo_pad   │
  └──────────────────────────────────────┘

  Key property: ENTIRE body is encrypted
  Separate TCP connection for each AAA function

RADIUS packet structure:
  ┌──────────────────────────────────────┐
  │ Code (1 byte): Access-Request(1),   │
  │   Accept(2), Reject(3), Challenge(11)│
  │ Identifier (1 byte): match req/resp │
  │ Length (2 bytes)                      │
  │ Authenticator (16 bytes):            │
  │   Request: random nonce              │
  │   Response: MD5(Code+ID+Length+      │
  │     RequestAuth+Attributes+Secret)   │
  ├──────────────────────────────────────┤
  │ Attributes (AVPs):                   │
  │   Type(1) + Length(1) + Value(var)   │
  │   User-Name: cleartext              │
  │   User-Password: encrypted with     │
  │     MD5(Secret + RequestAuth)        │
  │   All other attributes: CLEARTEXT   │
  └──────────────────────────────────────┘

  Key property: only User-Password is encrypted
  Single UDP transaction per auth attempt
  Auth+Authz combined in one exchange
```

---

## 4. SNMP Protocol Operations and MIB Tree Structure

### The MIB Tree

The Management Information Base is a hierarchical namespace of Object Identifiers (OIDs). Every managed object in every device maps to a unique position in this global tree.

```
MIB tree root:
  iso(1)
    └── org(3)
        └── dod(6)
            └── internet(1)
                ├── mgmt(2)
                │   └── mib-2(1)               ← standard MIBs
                │       ├── system(1)           ← sysDescr, sysUpTime, sysContact
                │       ├── interfaces(2)       ← ifTable, ifXTable
                │       ├── ip(4)               ← ipRouteTable, ipAddrTable
                │       ├── icmp(5)
                │       ├── tcp(6)
                │       ├── udp(7)
                │       └── snmp(11)
                ├── private(4)
                │   └── enterprises(1)         ← vendor-specific MIBs
                │       ├── cisco(9)
                │       ├── netSnmp(8072)
                │       ├── juniper(2636)
                │       └── ...
                └── security(5)

Full OID example:
  .1.3.6.1.2.1.1.1.0 = iso.org.dod.internet.mgmt.mib-2.system.sysDescr.0
```

**Table indexing in SNMP:** Tabular objects (like interface statistics) use row indices appended to the column OID.

```
ifTable (1.3.6.1.2.1.2.2):
  Each row is an interface, indexed by ifIndex

  ifDescr.1  = "GigabitEthernet0/0"     OID: .1.3.6.1.2.1.2.2.1.2.1
  ifDescr.2  = "GigabitEthernet0/1"     OID: .1.3.6.1.2.1.2.2.1.2.2

  ifOperStatus.1 = up(1)                OID: .1.3.6.1.2.1.2.2.1.8.1
  ifOperStatus.2 = down(2)              OID: .1.3.6.1.2.1.2.2.1.8.2

  ifInOctets.1 = 1234567890             OID: .1.3.6.1.2.1.2.2.1.10.1

Column OID structure:
  ifTable.ifEntry.ifDescr.ifIndex
  .1.3.6.1.2.1.2.2.1.2.<index>

For tables with compound indices (e.g., ipRouteTable), multiple index
values are concatenated:
  ipRouteNextHop.10.1.1.0 = 10.1.1.1
  OID: .1.3.6.1.2.1.4.21.1.7.10.1.1.0
  (index is the destination IP, encoded as four sub-OIDs)
```

### SNMP Protocol Operations

```
PDU types and their semantics:

GetRequest (0xA0):
  Manager → Agent: "Give me the value of this exact OID"
  Response: the value, or noSuchObject/noSuchInstance error
  Used for: reading individual values

GetNextRequest (0xA1):
  Manager → Agent: "Give me the value of the next OID after this one"
  Response: the next OID in lexicographic order and its value
  Used for: walking the MIB tree (repeated GetNext)
  Foundation of snmpwalk

GetBulkRequest (0xA5) — SNMPv2c/v3 only:
  Manager → Agent: "Give me the next N OIDs after this one"
  Parameters:
    non-repeaters: number of scalar OIDs to get (one value each)
    max-repetitions: number of table rows to return per OID
  Response: up to (non-repeaters + max-repetitions * remaining) varbinds
  Used for: efficient table retrieval (replaces repeated GetNext)

SetRequest (0xA3):
  Manager → Agent: "Set this OID to this value"
  Response: success or error (noAccess, wrongType, wrongValue, etc.)
  Used for: configuration changes, resetting counters
  Requires read-write community (v2c) or write access (v3 VACM)

Response (0xA2):
  Agent → Manager: response to any Get/Set request
  Contains: request-id (matches request), error-status, error-index, varbinds

Trap (0xA7) / SNMPv2-Trap (v2c):
  Agent → Manager: unsolicited notification
  Fire-and-forget (no acknowledgment)
  Standard traps: coldStart, warmStart, linkDown, linkUp, authenticationFailure

InformRequest (0xA6) — SNMPv2c/v3 only:
  Agent → Manager: acknowledged notification
  Manager sends Response to confirm receipt
  Agent retransmits if no response within timeout
  More reliable than traps but higher overhead
```

### SNMPv3 Security Model (USM and VACM)

```
User-based Security Model (USM):

Security levels:
  noAuthNoPriv  — username only (like community string but named)
  authNoPriv    — username + HMAC authentication (MD5, SHA-1, SHA-256, SHA-512)
  authPriv      — username + HMAC + encryption (DES, 3DES, AES-128/192/256)

Authentication process:
  1. Sender computes HMAC over the entire SNMP message using auth key
  2. HMAC is placed in msgAuthenticationParameters field (12 bytes, truncated)
  3. Receiver recomputes HMAC and compares
  4. Auth key derived from password via key localization:
     authKey = HMAC(password, engineID)
     (localization binds the key to a specific engine, preventing reuse)

Encryption process (AES-128):
  1. Derive privacy key from password (same localization as auth)
  2. Generate IV from engineBoots + engineTime + salt (64-bit counter)
  3. Encrypt scopedPDU using AES-CFB-128
  4. Encrypted data placed in msgData, salt in msgPrivacyParameters

Timeliness check (replay protection):
  - Each SNMP engine maintains engineBoots (reboot counter) and engineTime
  - Messages are accepted only if:
    |msg.engineTime - local.engineTime| < 150 seconds
    AND msg.engineBoots == local.engineBoots
  - Prevents replay of captured packets after 150-second window

View-based Access Control Model (VACM):

  Three tables control access:
  1. vacmSecurityToGroupTable: maps (securityModel, securityName) → groupName
  2. vacmAccessTable: maps (groupName, contextPrefix, securityModel,
     securityLevel) → {readView, writeView, notifyView}
  3. vacmViewTreeFamilyTable: defines which OID subtrees are in each view

  Example access chain:
    User "monitor" → securityModel SNMPv3 → group "readonly"
    Group "readonly" + authPriv → readView="allMIBs", writeView="none"
    View "allMIBs" → includes .1 (entire tree)
    View "none" → includes nothing (no write access)
```

---

## 5. Syslog RFC 5424 Message Format

### Message Structure

RFC 5424 defines the modern syslog message format, replacing the ad-hoc BSD syslog format (RFC 3164). Every field has a defined syntax and semantics.

```
RFC 5424 ABNF:

SYSLOG-MSG = HEADER SP STRUCTURED-DATA [SP MSG]

HEADER = PRI VERSION SP TIMESTAMP SP HOSTNAME SP APP-NAME SP PROCID SP MSGID

PRI = "<" PRIVAL ">"
PRIVAL = 1*3DIGIT                  (0..191, = facility*8 + severity)
VERSION = NONZERO-DIGIT            (currently always "1")
TIMESTAMP = FULL-DATE "T" FULL-TIME    (RFC 3339 / ISO 8601)
           | NILVALUE ("-")
HOSTNAME = 1*255PRINTUSASCII | NILVALUE
APP-NAME = 1*48PRINTUSASCII | NILVALUE
PROCID = 1*128PRINTUSASCII | NILVALUE
MSGID = 1*32PRINTUSASCII | NILVALUE

STRUCTURED-DATA = NILVALUE | 1*SD-ELEMENT
SD-ELEMENT = "[" SD-ID *(SP SD-PARAM) "]"
SD-ID = SD-NAME                    (max 32 chars)
SD-PARAM = PARAM-NAME "=" %d34 PARAM-VALUE %d34

MSG = MSG-ANY | MSG-UTF8
MSG-UTF8 = BOM UTF-8-STRING        (BOM = 0xEF 0xBB 0xBF)
```

### Complete Message Dissection

```
Example message:
<165>1 2026-04-05T14:30:00.003Z router01.example.com ospfd 2853 ADJCHANGE [origin ip="10.1.1.1" software="FRRouting 9.1" swVersion="9.1"][timeQuality tzKnown="1" isSynced="1" syncAccuracy="11450"] OSPF neighbor 10.1.1.2 on eth0 state changed to Full

Field-by-field breakdown:
  <165>       PRI: facility=local4(20), severity=notice(5), 20*8+5=165
  1           VERSION: RFC 5424
  2026-04-05T14:30:00.003Z  TIMESTAMP: UTC, millisecond precision
  router01.example.com      HOSTNAME: FQDN of originating host
  ospfd       APP-NAME: the process that generated the message
  2853        PROCID: PID of the ospfd process
  ADJCHANGE   MSGID: message type identifier (application-defined)

  [origin ip="10.1.1.1" software="FRRouting 9.1" swVersion="9.1"]
              SD-ELEMENT with SD-ID "origin" (IANA-registered)
              Contains: source IP, software name, version

  [timeQuality tzKnown="1" isSynced="1" syncAccuracy="11450"]
              SD-ELEMENT with SD-ID "timeQuality" (IANA-registered)
              tzKnown=1: timezone is known
              isSynced=1: clock is synchronized
              syncAccuracy=11450: accuracy in microseconds

  OSPF neighbor 10.1.1.2 on eth0 state changed to Full
              MSG: free-form message text (UTF-8)
```

### IANA-Registered Structured Data IDs

```
SD-ID              Purpose
───────────────────────────────────────────────────────
origin             Identifies the originator
                   Params: ip, enterpriseId, software, swVersion

meta               Metadata about the message
                   Params: sequenceId, sysUpTime, language

timeQuality        Timestamp quality indicators
                   Params: tzKnown, isSynced, syncAccuracy

Custom SD-IDs use the format: name@enterpriseNumber
Example: myApp@32473 (32473 = your IANA Private Enterprise Number)
```

### Priority Calculation and Interpretation

```
PRIVAL = facility * 8 + severity

Encoding:
  Facility 0-23 (3 bits → 5 bits in PRI), Severity 0-7 (3 bits)
  Total PRI range: 0 to 191

Decoding:
  facility = PRIVAL / 8   (integer division)
  severity = PRIVAL % 8   (modulo)

Examples:
  <0>   = kern.emerg        (0*8 + 0)
  <34>  = auth.crit         (4*8 + 2)
  <165> = local4.notice     (20*8 + 5)
  <191> = local7.debug      (23*8 + 7)

Common network device mappings:
  Cisco IOS default: local7 (facility 23)
  Juniper default:   local0-local7 (configurable per process)
  Linux kernel:      kern (facility 0)
  sshd:              authpriv (facility 10)
  cron:              cron (facility 9)
```

### Transport Protocols

```
RFC 5426 — Syslog over UDP:
  Port 514 (default)
  Unreliable: no delivery confirmation, no congestion control
  One message per UDP datagram
  Maximum message size: limited by UDP/IP (65535 - headers)
  Practical limit: often 1024 or 2048 bytes

RFC 5425 — Syslog over TLS:
  Port 6514 (default)
  Reliable: TCP + TLS 1.2/1.3
  Message framing: octet counting (length prefix)
    MSG-LEN SP SYSLOG-MSG
    Example: 175 <165>1 2026-04-05T...
  Mutual TLS authentication supported
  Certificate validation prevents log injection

RFC 5426 message loss scenarios:
  - Network congestion → UDP drops (no backpressure)
  - Receiver buffer overflow → kernel drops
  - Firewall state table full → dropped silently
  - Source throttling (logging rate-limit) → intentional drop

Mitigation: use TCP/TLS (RFC 5425) or RELP (Reliable Event Logging Protocol)
  RELP adds application-level acknowledgment on top of TCP
  rsyslog native: imrelp/omrelp modules
```

---

## 6. PTP / IEEE 1588 Clock Recovery Theory

### The Precision Time Protocol Problem

NTP achieves millisecond accuracy over the internet and microseconds on LANs. Many applications (telecom, financial trading, industrial control, broadcast media) require sub-microsecond or even nanosecond accuracy. PTP achieves this through hardware timestamping and a different synchronization model.

### PTP Message Exchange and Offset Calculation

```
PTP uses a two-step or one-step synchronization exchange:

Two-step synchronization:

Master                                    Slave
  |                                         |
  |  ── Sync (t1 approximate) ──────────>   |
  |  ── Follow_Up (t1 precise) ─────────>   | t2 = receive timestamp
  |                                         |
  |  <── Delay_Req ─────────────────────    | t3 = send timestamp
  |  ── Delay_Resp (t4) ────────────────>   |
  |                                         |

One-step: t1 is embedded directly in the Sync message
  (requires hardware that can modify the packet on-the-fly)

Timestamps:
  t1 = master sends Sync (precise, from hardware)
  t2 = slave receives Sync (precise, from hardware)
  t3 = slave sends Delay_Req (precise, from hardware)
  t4 = master receives Delay_Req (precise, from hardware)

Offset and delay calculation (identical to NTP):
  offset = ((t2 - t1) - (t4 - t3)) / 2
  delay  = ((t2 - t1) + (t4 - t3)) / 2

Same symmetric delay assumption as NTP, but hardware timestamping
removes the kernel/software jitter that dominates NTP error.
```

### Best Master Clock Algorithm (BMCA)

The BMCA determines which clock in a PTP domain should be the grandmaster. It runs on every PTP port and evaluates Announce messages from all candidates.

```
BMCA comparison order (IEEE 1588 Section 9.3.4):

1. Priority1 (0-255, lower is better, default 128)
   → Allows administrative override of all other criteria
   → A clock with priority1=0 always wins

2. Clock class (higher class = better quality)
   Class 6:  primary reference (GPS, atomic)
   Class 7:  primary reference, holdover
   Class 52: degraded, was class 7
   Class 187: application-specific
   Class 248: default (free-running oscillator)
   Class 255: slave-only (never becomes master)

3. Clock accuracy (enumeration, lower = more accurate)
   0x20: 25 ns
   0x21: 100 ns
   0x22: 250 ns
   0x23: 1 us
   0x24: 2.5 us
   0x31: >10 s (unknown)

4. Offset scaled log variance
   → Statistical measure of clock stability
   → Lower = more stable oscillator

5. Priority2 (0-255, lower is better, default 128)
   → Tiebreaker between otherwise equal clocks

6. Clock identity (EUI-64, derived from MAC address)
   → Final tiebreaker (deterministic, prevents oscillation)

The BMCA runs every Announce interval (default 2 seconds).
If a better clock appears, the grandmaster changes within
3 * announceInterval (default 6 seconds).
```

### Transparent Clocks and Correction Fields

PTP introduces transparent clocks to solve a problem NTP cannot: switch queuing delay.

```
The problem:
  When a PTP message passes through a non-PTP-aware switch, it sits in
  output queues for a variable time (microseconds to milliseconds).
  This queuing delay is indistinguishable from propagation delay,
  and it varies per packet, destroying accuracy.

Transparent Clock solution:
  A PTP-aware switch timestamps the message at ingress and egress,
  computes the residence time, and adds it to the correctionField.

  correctionField += (t_egress - t_ingress)

  The slave subtracts the accumulated correctionField from its
  delay calculation:

  corrected_delay = raw_delay - correctionField

  This removes all queuing delay from the measurement, regardless
  of how many switches are in the path.

Two types:
  End-to-End Transparent Clock (E2E TC):
    - Corrects Sync and Delay_Req messages
    - Compatible with end-to-end delay measurement

  Peer-to-Peer Transparent Clock (P2P TC):
    - Uses Pdelay messages to measure each link independently
    - Corrects Sync messages using accumulated peer delay
    - Faster convergence, works with redundant topologies

Boundary Clock (BC) vs Transparent Clock (TC):
  BC: terminates PTP on each port, re-syncs as a new clock
      → Each hop adds its own clock error
      → N hops = N * single-hop error (error accumulates)

  TC: passes PTP through with residence time correction
      → Queuing delay is corrected, not accumulated
      → N hops ≈ single-hop error (error does not accumulate)
```

### PTP Profiles and Domain Separation

```
PTP profiles define parameter sets for specific industries:

Profile              Domain   Sync Interval   Delay Mechanism
Default (1588)       0        1s (2^0)        E2E or P2P
Telecom (G.8275.1)   24       1/16s (2^-4)    E2E
Telecom (G.8275.2)   44       1/16s (2^-4)    E2E (unicast)
gPTP (802.1AS)       0        1/8s (2^-3)     P2P only
Power (C37.238)      0        1s (2^0)        P2P
AES67 (audio)        0        1/8s (2^-3)     varies

Domain numbers (0-127) provide isolation:
  - Clocks in domain 0 ignore messages from domain 24
  - Allows multiple PTP systems on the same network
  - Each domain has its own grandmaster election

Transport options:
  IEEE 802.3 (Layer 2): EtherType 0x88F7, multicast
    Sync/Announce: 01:1B:19:00:00:00
    Delay_Req:     01:80:C2:00:00:0E
    Pdelay:        01:80:C2:00:00:0E

  UDP/IPv4 (Layer 3): port 319 (event), port 320 (general)
    Multicast: 224.0.1.129 (default domain)
    Unicast: configurable per profile

  UDP/IPv6 (Layer 3): same ports
    Multicast: FF0x::181
```

### Error Budget Analysis

```
PTP accuracy depends on the weakest link in the timestamping chain.
Typical error contributions:

Source                        Error Contribution
──────────────────────────────────────────────────
Oscillator stability          1-10 ns (TCXO/OCXO)
Hardware timestamp resolution 4-8 ns (typical PHY)
PHY asymmetry (TX vs RX)     0-50 ns (correctable via calibration)
Cable asymmetry               0.1 ns/m temperature coefficient
Switch residence time jitter  1-10 ns (good TC), 1-100 us (non-PTP switch)
Asymmetric network path       unbounded (PTP cannot detect this)

Total error (well-engineered PTP network):
  Single hop:  < 100 ns
  5-hop campus: < 500 ns with BCs, < 200 ns with TCs
  WAN (unicast PTP): 1-10 us (asymmetry dominates)

For comparison:
  NTP over internet:     1-50 ms
  NTP on LAN:           0.1-1 ms
  chrony on LAN:        10-100 us
  PTP with SW stamps:   1-100 us
  PTP with HW stamps:   10-100 ns
  GPS/PPS direct:       10-50 ns
```

---

## Prerequisites

- dhcp, ntp, ptp, snmp, tacacs, radius, syslog, subnetting, tcp, udp

## References

- [RFC 2131 — Dynamic Host Configuration Protocol](https://www.rfc-editor.org/rfc/rfc2131)
- [RFC 3046 — DHCP Relay Agent Information Option](https://www.rfc-editor.org/rfc/rfc3046)
- [RFC 3527 — Link Selection Sub-option for Relay Agent Information](https://www.rfc-editor.org/rfc/rfc3527)
- [RFC 5905 — NTPv4: Protocol and Algorithms Specification](https://www.rfc-editor.org/rfc/rfc5905)
- [RFC 8915 — Network Time Security for NTP](https://www.rfc-editor.org/rfc/rfc8915)
- [Mills, D. "Computer Network Time Synchronization" (2006)](https://www.eecis.udel.edu/~mills/book.html)
- [Marzullo, K. "Maintaining the Time in a Distributed System" (1984)](https://dl.acm.org/doi/10.5555/899882)
- [IEEE 1588-2019 — Precision Time Protocol](https://standards.ieee.org/standard/1588-2019.html)
- [ITU-T G.8275.1 — PTP Telecom Profile](https://www.itu.int/rec/T-REC-G.8275.1)
- [IEEE 802.1AS-2020 — Timing and Synchronization (gPTP)](https://standards.ieee.org/standard/802_1AS-2020.html)
- [RFC 2865 — RADIUS](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 8907 — TACACS+ Protocol](https://www.rfc-editor.org/rfc/rfc8907)
- [RFC 3411 — SNMP Architecture](https://www.rfc-editor.org/rfc/rfc3411)
- [RFC 3414 — USM for SNMPv3](https://www.rfc-editor.org/rfc/rfc3414)
- [RFC 3415 — VACM for SNMP](https://www.rfc-editor.org/rfc/rfc3415)
- [RFC 5424 — The Syslog Protocol](https://www.rfc-editor.org/rfc/rfc5424)
- [RFC 5425 — TLS Transport Mapping for Syslog](https://www.rfc-editor.org/rfc/rfc5425)
