# RADIUS — Remote Authentication Dial-In User Service

The dominant AAA (Authentication, Authorization, Accounting) protocol for network access control: 802.1X, VPN, dial-up, switch/AP/firewall login. UDP-based, attribute-value-pair (AVP) framed, with per-NAS shared secrets. FreeRADIUS is the canonical open-source implementation; this sheet is the field manual for running it.

## Setup

```bash
# Debian / Ubuntu
sudo apt update
sudo apt install freeradius freeradius-utils freeradius-ldap freeradius-mysql freeradius-postgresql freeradius-krb5 freeradius-rest

# RHEL / CentOS / Rocky / Alma
sudo dnf install freeradius freeradius-utils freeradius-ldap freeradius-mysql freeradius-postgresql freeradius-krb5

# Arch
sudo pacman -S freeradius

# macOS (homebrew, for testing only)
brew install freeradius-server

# from source (FreeRADIUS 3.2.x)
git clone https://github.com/FreeRADIUS/freeradius-server.git
cd freeradius-server
./configure --prefix=/opt/freeradius --with-rlm-eap-tls
make && sudo make install

# verify
radiusd -v
# radiusd: FreeRADIUS Version 3.2.3, for host x86_64-pc-linux-gnu

# service
sudo systemctl enable --now freeradius      # Debian
sudo systemctl enable --now radiusd          # RHEL (binary is radiusd)

# foreground debug — the canonical mode while building config
sudo freeradius -X                           # Debian
sudo radiusd -X                              # RHEL
```

Alternatives in the RADIUS-server space:

```text
freeradius              — the open-source reference; written in C; modular; v3.x mainline, v4 in dev
radiusd                 — RHEL binary name for FreeRADIUS (same code, different package layout)
Microsoft NPS           — Network Policy Server; bundled with Windows Server; AD-tight; GUI-driven
Aruba ClearPass         — commercial; policy-rich; used in BYOD / NAC deployments
Cisco ISE               — commercial; identity policy + posture + profiling
Cisco ACS               — legacy; replaced by ISE
Radiator                — commercial Perl-based RADIUS; very flexible
JRadius / TinyRADIUS    — niche / embedded
radsecproxy             — TLS proxy in front of any RADIUS; for federation (eduroam)
```

The FreeRADIUS server is the dominant open-source RADIUS implementation. If you read "RADIUS server" in a doc and it doesn't say which, assume FreeRADIUS unless the org is Windows-shop (NPS) or Cisco (ISE).

## Protocol Overview

```text
RFC 2865    — RADIUS authentication (the canonical RFC; June 2000)
RFC 2866    — RADIUS accounting
RFC 2867    — Tunnel accounting attributes
RFC 2868    — Tunnel attributes
RFC 2869    — RADIUS extensions (Acct-Input-Gigawords, EAP-Message, Message-Authenticator)
RFC 3162    — IPv6 attributes (NAS-IPv6-Address, Framed-IPv6-Prefix)
RFC 3576    — first CoA / Disconnect (obsoleted)
RFC 5176    — CoA / Disconnect (current; replaces 3576)
RFC 5080    — common implementation issues / clarifications
RFC 5997    — Status-Server (keepalive)
RFC 6158    — design guidelines for new attributes
RFC 6613    — RADIUS over TCP (rare; debugging/test only)
RFC 6614    — RADIUS over TLS (RadSec); TCP/2083
RFC 6929    — extended attributes (Type values 241–246)
RFC 7585    — Dynamic Peer Discovery (NAPTR for eduroam)
RFC 8044    — modern attribute data types (datatypes for new AVPs)
RFC 8559    — Dynamic Authorization proxying
```

UDP ports:

```text
1812/udp    — Authentication (modern, IANA-assigned)
1813/udp    — Accounting (modern, IANA-assigned)
1645/udp    — Authentication (legacy; still used by some old gear)
1646/udp    — Accounting (legacy)
3799/udp    — CoA / Disconnect (RFC 5176)
2083/tcp    — RADIUS-over-TLS / RadSec (RFC 6614)
```

TCP variant: RFC 6613 specifies RADIUS over TCP, but it is a debugging/transitional spec; production deployments use UDP for plain RADIUS or TLS for secure transport. RadSec (RFC 6614) wraps RADIUS in TLS over TCP/2083 — used heavily in eduroam and any inter-org RADIUS federation where shared secrets are not acceptable.

Packet format:

```text
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|     Code      |  Identifier   |            Length             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                         Authenticator                         |
|                            (16 bytes)                         |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Attributes ...
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Codes:

```text
1   Access-Request
2   Access-Accept
3   Access-Reject
4   Accounting-Request
5   Accounting-Response
11  Access-Challenge
12  Status-Server (RFC 5997)
13  Status-Client
40  Disconnect-Request   (RFC 5176)
41  Disconnect-ACK
42  Disconnect-NAK
43  CoA-Request          (RFC 5176)
44  CoA-ACK
45  CoA-NAK
```

Attribute TLV format inside the packet:

```text
+-----+--------+------------------+
| Typ | Length | Value (Length-2) |
+-----+--------+------------------+
1 byte 1 byte    0–253 bytes
```

Length max 255 — values longer than 253 bytes are split across multiple attributes (EAP-Message especially).

## Architecture

```text
+------------+     EAPOL / PAP / CHAP    +-------------------+    Access-Request    +----------------+
| Supplicant | <-----------------------> | NAS               | -------------------> | RADIUS Server  |
| (laptop,   |                           | (Network Access   | <------------------- | (FreeRADIUS,   |
|  phone,    |                           |  Server: switch,  |   Access-Accept |    |  NPS, ISE)     |
|  AP client)|                           |  AP, VPN gateway, |   Access-Reject |    |                |
+------------+                           |  firewall, dial)  |   Access-Challenge   +----------------+
                                         +-------------------+                              |
                                                                                            v
                                                                           +-----------------------------+
                                                                           | Backends                    |
                                                                           |  - LDAP / Active Directory  |
                                                                           |  - SQL (MySQL, Postgres)    |
                                                                           |  - Local users file         |
                                                                           |  - Kerberos                 |
                                                                           |  - REST                     |
                                                                           +-----------------------------+
```

Vocabulary:

```text
Supplicant      — the device requesting access (laptop, phone, dial client, VPN client)
NAS             — Network Access Server; the device the supplicant connects to (switch, AP, VPN gateway)
RADIUS server   — the AAA server that validates and authorizes
Backend         — where the actual user database lives (LDAP/AD/SQL/file)
Shared secret   — symmetric key between NAS and server; used for authenticator + password obfuscation
Realm           — username suffix; @example.com routes to a different server
Proxy           — RADIUS server forwarding requests to another RADIUS server (federation)
```

The shared secret is per-NAS, never transmitted, and is the basis for both packet integrity (Authenticator field, Message-Authenticator AVP) and User-Password obfuscation. A stolen secret = total compromise of that NAS pairing.

## Authentication Flow

```text
NAS                                                      RADIUS server
 |                                                              |
 |-- Access-Request ------------------------------------------->|
 |   User-Name=alice                                            |
 |   User-Password=<obfuscated>                                 |
 |   NAS-IP-Address=10.0.0.1                                    |
 |   NAS-Port=42                                                |
 |   Calling-Station-Id=AA:BB:CC:DD:EE:FF                       |
 |   Message-Authenticator=<HMAC-MD5>                           |
 |                                                              |
 |                         <---- Access-Accept (if ok) ---------|
 |                                Reply-Message="Welcome"       |
 |                                Session-Timeout=3600          |
 |                                Tunnel-Private-Group-ID=20    |
 |                                                              |
 |                         <---- Access-Reject (if bad) --------|
 |                                Reply-Message="Bad password"  |
 |                                                              |
 |                         <---- Access-Challenge (EAP) --------|
 |                                State=<opaque>                |
 |                                EAP-Message=<bytes>           |
 |                                                              |
 |-- Access-Request (continuation) ---------------------------->|
 |   State=<opaque copy>                                        |
 |   EAP-Message=<bytes>                                        |
 |                                                              |
 |   ... (multiple round-trips for EAP) ...                     |
```

User-Password obfuscation (RFC 2865 §5.2):

```text
1.  pad password to a multiple of 16 bytes with NULs
2.  let p1, p2, ... be 16-byte chunks
3.  b1 = MD5(secret + Request-Authenticator)
    c1 = p1 XOR b1
4.  b2 = MD5(secret + c1)
    c2 = p2 XOR b2
5.  ... (chained)
6.  the User-Password attribute carries c1 || c2 || ...
```

This is obfuscation, not encryption — known plaintext + secret recovery is feasible. The Message-Authenticator AVP (HMAC-MD5 of the entire packet keyed with the shared secret) provides packet integrity and is REQUIRED for EAP, and post-BlastRADIUS (CVE-2024-3596) is REQUIRED for all packets in modern deployments.

CHAP-Password (Type 3): NAS sends a CHAP-Challenge (Type 60) and a CHAP-Password computed by the supplicant; server must have access to cleartext password to verify (which is why CHAP is incompatible with hashed-password backends).

## Accounting Flow

```text
NAS                                              RADIUS server
 |                                                       |
 |-- Accounting-Request (Acct-Status-Type=Start) ------->|
 |   Acct-Session-Id=ABC123                              |
 |   User-Name=alice                                     |
 |   NAS-IP-Address=10.0.0.1                             |
 |   Framed-IP-Address=10.20.30.40                       |
 |                                                       |
 |                  <---- Accounting-Response -----------|
 |                                                       |
 |   ... session in progress ...                         |
 |                                                       |
 |-- Accounting-Request (Acct-Status-Type=Interim) ----->|
 |   Acct-Session-Id=ABC123                              |
 |   Acct-Session-Time=300                               |
 |   Acct-Input-Octets=1048576                           |
 |   Acct-Output-Octets=2097152                          |
 |                                                       |
 |                  <---- Accounting-Response -----------|
 |                                                       |
 |-- Accounting-Request (Acct-Status-Type=Stop) -------->|
 |   Acct-Session-Id=ABC123                              |
 |   Acct-Session-Time=3600                              |
 |   Acct-Input-Octets=10485760                          |
 |   Acct-Output-Octets=20971520                         |
 |   Acct-Terminate-Cause=User-Request                   |
 |                                                       |
 |                  <---- Accounting-Response -----------|
```

Acct-Status-Type values:

```text
1  Start           — session begin
2  Stop            — session end
3  Interim-Update  — periodic stats during session (RFC 2869)
7  Accounting-On   — NAS booted (start of all sessions)
8  Accounting-Off  — NAS shutting down
9–14 reserved
15 Failed
```

Interim-Update interval is set by:

```text
Acct-Interim-Interval (85)   — server-pushed value in Access-Accept (seconds)
                               many NAS implementations also have a local config knob
                               typical values: 300, 600, 1800
```

Accounting-Response contains no AVPs (just the code+id+length+authenticator) — its arrival is the only confirmation needed.

## CoA / Disconnect (RFC 5176)

CoA = Change-of-Authorization. The server proactively contacts the NAS to alter or terminate an in-progress session.

```text
RADIUS server                                              NAS
       |                                                    |
       |-- CoA-Request --------------------------------->   |
       |   User-Name=alice                                  |
       |   Acct-Session-Id=ABC123                           |
       |   Tunnel-Private-Group-ID=99   (move to VLAN 99)   |
       |                                                    |
       |   <-- CoA-ACK (success) ---------------------------|
       |   <-- CoA-NAK (failure) ---------------------------|
       |          Error-Cause=...                           |
       |                                                    |
       |-- Disconnect-Request --------------------------->  |
       |   User-Name=alice                                  |
       |   Acct-Session-Id=ABC123                           |
       |                                                    |
       |   <-- Disconnect-ACK (session killed) -------------|
       |   <-- Disconnect-NAK (failed) ---------------------|
       |          Error-Cause=...                           |
```

Sent to UDP/3799 on the NAS. Uses the shared secret for authentication. The NAS must identify the session — typical session-identification attributes:

```text
User-Name (1)                — together with Calling-Station-Id, often unique
Acct-Session-Id (44)         — the canonical session identifier
NAS-IP-Address (4)           — for proxied scenarios
Calling-Station-Id (31)      — the supplicant MAC, often used for MAB sessions
Framed-IP-Address (8)        — IP-based identification
```

Error-Cause (101) values for CoA-NAK / Disconnect-NAK:

```text
201  Residual-Session-Context-Removed
202  Invalid-EAP-Packet
401  Unsupported-Attribute
402  Missing-Attribute
403  NAS-Identification-Mismatch
404  Invalid-Request
405  Unsupported-Service
406  Unsupported-Extension
407  Invalid-Attribute-Value
501  Administratively-Prohibited
502  Request-Not-Routable
503  Session-Context-Not-Found
504  Session-Context-Not-Removable
505  Other-Proxy-Processing-Error
506  Resources-Unavailable
507  Request-Initiated
508  Multiple-Session-Selection-Unsupported
```

## AVP Catalog

The exhaustive RFC-defined attribute set (RFC 2865 / 2866 / 2869 / 3162 / 5176). Numbers in parentheses are the type values on the wire.

```text
1   User-Name                  string    — login (often realm-qualified: user@example.com)
2   User-Password              string    — obfuscated cleartext password
3   CHAP-Password              string    — CHAP response (1 byte CHAP-Id + 16 bytes hash)
4   NAS-IP-Address             ipaddr    — NAS sending the request (IPv4)
5   NAS-Port                   integer   — physical/virtual port number on NAS
6   Service-Type               integer   — what the user is asking for:
                                            1  Login
                                            2  Framed
                                            3  Callback-Login
                                            4  Callback-Framed
                                            5  Outbound
                                            6  Administrative
                                            7  NAS-Prompt
                                            8  Authenticate-Only
                                            9  Callback-NAS-Prompt
                                            10 Call-Check
                                            11 Callback-Administrative
7   Framed-Protocol            integer   — 1=PPP, 2=SLIP, 3=ARAP, 4=Gandalf, 5=Xylogics, 6=X.75
8   Framed-IP-Address          ipaddr    — assigned IP for framed user
9   Framed-IP-Netmask          ipaddr    — netmask for framed user
10  Framed-Routing             integer   — 0=None, 1=Send, 2=Listen, 3=Send+Listen
11  Filter-Id                  string    — name of ACL/filter to apply on NAS
12  Framed-MTU                 integer   — MTU for framed link (64–65535)
13  Framed-Compression         integer   — 0=None, 1=VJ-TCP/IP, 2=IPX, 3=Stac-LZS
14  Login-IP-Host              ipaddr    — host to connect to on Login service
15  Login-Service              integer   — Telnet=0, Rlogin=1, TCP-Clear=2, PortMaster=3, LAT=4
16  Login-TCP-Port             integer   — TCP port for Login
17                                       — (unused)
18  Reply-Message              string    — human-readable text shown to user
19  Callback-Number            string    — number for NAS to dial back
20  Callback-Id                string    — symbolic name for callback location
21                                       — (unused)
22  Framed-Route               string    — static route to install (network/mask gateway metric)
23  Framed-IPX-Network         integer   — IPX network number
24  State                      string    — opaque, returned by NAS in next Access-Request
                                            (mandatory when server sent Access-Challenge)
25  Class                      string    — opaque, returned by NAS in Accounting-Request
                                            (use to thread session context)
26  Vendor-Specific            string    — VSA wrapper; see Vendor-Specific Attributes
27  Session-Timeout            integer   — max seconds before session auto-terminates
28  Idle-Timeout               integer   — max seconds of idle before session terminates
29  Termination-Action         integer   — 0=Default, 1=RADIUS-Request (re-auth on timeout)
30  Called-Station-Id          string    — phone-number/MAC/SSID dialed/connected to
                                            (for Wi-Fi: AP MAC + ":" + SSID)
31  Calling-Station-Id         string    — supplicant phone/MAC (e.g., "AA-BB-CC-DD-EE-FF")
32  NAS-Identifier             string    — string identity of NAS (alternative to NAS-IP-Address)
33  Proxy-State                string    — opaque, used by proxies for state threading
34  Login-LAT-Service          string    — LAT service name
35  Login-LAT-Node             string    — LAT node name
36  Login-LAT-Group            string    — LAT group code
37  Framed-AppleTalk-Link      integer   — AppleTalk network for serial link
38  Framed-AppleTalk-Network   integer   — AppleTalk network for routing
39  Framed-AppleTalk-Zone      string    — AppleTalk zone name
40  Acct-Status-Type           integer   — 1=Start, 2=Stop, 3=Interim-Update,
                                            7=Accounting-On, 8=Accounting-Off, 15=Failed
41  Acct-Delay-Time            integer   — seconds NAS has been retrying this acct request
42  Acct-Input-Octets          integer   — bytes received from user (low 32 bits)
43  Acct-Output-Octets         integer   — bytes sent to user (low 32 bits)
44  Acct-Session-Id            string    — unique session identifier
45  Acct-Authentic             integer   — 1=RADIUS, 2=Local, 3=Remote, 4=Diameter
46  Acct-Session-Time          integer   — session duration in seconds
47  Acct-Input-Packets         integer   — packets received from user
48  Acct-Output-Packets        integer   — packets sent to user
49  Acct-Terminate-Cause       integer   — why session ended:
                                            1  User-Request
                                            2  Lost-Carrier
                                            3  Lost-Service
                                            4  Idle-Timeout
                                            5  Session-Timeout
                                            6  Admin-Reset
                                            7  Admin-Reboot
                                            8  Port-Error
                                            9  NAS-Error
                                            10 NAS-Request
                                            11 NAS-Reboot
                                            12 Port-Unneeded
                                            13 Port-Preempted
                                            14 Port-Suspended
                                            15 Service-Unavailable
                                            16 Callback
                                            17 User-Error
                                            18 Host-Request
50  Acct-Multi-Session-Id      string    — links multiple sessions of same user
51  Acct-Link-Count            integer   — number of links in multi-link session
52  Acct-Input-Gigawords       integer   — 2^32 byte multiples of input (high 32 bits)
53  Acct-Output-Gigawords      integer   — 2^32 byte multiples of output (high 32 bits)
54                                       — (reserved)
55  Event-Timestamp            integer   — Unix epoch when event occurred
60  CHAP-Challenge             string    — CHAP challenge if not in Authenticator
61  NAS-Port-Type              integer   — physical port type:
                                            0  Async
                                            1  Sync
                                            2  ISDN-Sync
                                            3  ISDN-Async-V120
                                            4  ISDN-Async-V110
                                            5  Virtual
                                            6  PIAFS
                                            7  HDLC-Clear-Channel
                                            8  X.25
                                            9  X.75
                                            10 G.3-Fax
                                            11 SDSL
                                            12 ADSL-CAP
                                            13 ADSL-DMT
                                            14 IDSL
                                            15 Ethernet
                                            16 xDSL
                                            17 Cable
                                            18 Wireless-Other
                                            19 Wireless-802.11
62  Port-Limit                 integer   — max ports for this user
63  Login-LAT-Port             string    — LAT port name
64  Tunnel-Type                integer   — 1=PPTP, 2=L2F, 3=L2TP, 4=ATMP, 5=VTP, 6=AH,
                                            13=VLAN (the dynamic-VLAN value)
65  Tunnel-Medium-Type         integer   — 1=IPv4, 2=IPv6, 6=802 (the dynamic-VLAN value)
66  Tunnel-Client-Endpoint     string    — IP/host of tunnel initiator
67  Tunnel-Server-Endpoint     string    — IP/host of tunnel terminator
68  Acct-Tunnel-Connection     string    — opaque tunnel ID for accounting
69  Tunnel-Password            string    — encrypted password for tunnel auth
70  ARAP-Password              string    — AppleTalk Remote Access Protocol password
71  ARAP-Features              string    — ARAP feature flags
72  ARAP-Zone-Access           integer   — 1=Default, 2=All-Zones-In-List, 4=All-Except
73  ARAP-Security              integer   — security module identifier
74  ARAP-Security-Data         string    — security module payload
75  Password-Retry             integer   — max ARAP password retries
76  Prompt                     integer   — 0=No-Echo, 1=Echo (for login prompts)
77  Connect-Info               string    — NAS-supplied connect string ("Modem v.90 / 53333 LAPM")
78  Configuration-Token        string    — opaque token for proxy chains
79  EAP-Message                string    — EAP packet bytes (max 253; concatenate multiples)
80  Message-Authenticator      string    — HMAC-MD5 of packet, keyed with shared secret
                                            MUST for EAP; STRONGLY RECOMMENDED post-BlastRADIUS
81  Tunnel-Private-Group-ID    string    — VLAN tag for dynamic VLAN assignment ("20" or "vlan20")
82  Tunnel-Assignment-ID       string    — name of tunnel session
83  Tunnel-Preference          integer   — preference rank when multiple tunnels offered
84  ARAP-Challenge-Response    string    — ARAP challenge response
85  Acct-Interim-Interval      integer   — seconds between Interim-Update messages
86  Acct-Tunnel-Packets-Lost   integer   — tunnel-link packet loss counter
87  NAS-Port-Id                string    — string version of NAS-Port (e.g., "GigabitEthernet0/1")
88  Framed-Pool                string    — name of IP-pool to draw Framed-IP-Address from
89  CUI                        string    — Chargeable-User-Identity (RFC 4372); pseudonym
90  Tunnel-Client-Auth-ID      string    — tunnel-initiator auth ID
91  Tunnel-Server-Auth-ID      string    — tunnel-terminator auth ID
95  NAS-IPv6-Address           ipv6addr  — NAS IPv6 address
96  Framed-Interface-Id        ifid      — IPv6 interface ID for client (8 bytes)
97  Framed-IPv6-Prefix         ipv6prefix — IPv6 prefix to advertise to client
98  Login-IPv6-Host            ipv6addr  — IPv6 host for Login service
99  Framed-IPv6-Route          string    — static IPv6 route ("2001:db8::/32 via fe80::1")
100 Framed-IPv6-Pool           string    — name of IPv6-pool
101 Error-Cause                integer   — CoA/Disconnect failure reason (see CoA section)
```

Operator-related and EAP-related extensions worth knowing:

```text
102  EAP-Key-Name              string    — name for EAP-derived MSK; for key chaining
126  Operator-Name             string    — RFC 5580; "1example.com" prefix-coded namespace
127  Location-Information      string    — GEOPRIV-style location
128  Location-Data             string
129  Basic-Location-Policy-Rules
130  Extended-Location-Policy-Rules
131  Location-Capable          integer
132  Requested-Location-Info   integer
```

## Vendor-Specific Attributes (VSAs)

Encoded inside Type 26 (Vendor-Specific). The Value contains a Vendor-Id (4 bytes, IANA-assigned) followed by vendor-defined sub-attributes:

```text
+-----+--------+----------------------------------------+
| 26  | Length |  Vendor-Id (4 bytes) | vendor-attrs    |
+-----+--------+----------------------------------------+
```

Common vendor IDs and FreeRADIUS dictionary files:

```text
9       Cisco                  /usr/share/freeradius/dictionary.cisco
311     Microsoft              /usr/share/freeradius/dictionary.microsoft
2636    Juniper                /usr/share/freeradius/dictionary.juniper
14823   Aruba                  /usr/share/freeradius/dictionary.aruba
14988   Mikrotik               /usr/share/freeradius/dictionary.mikrotik
25053   Ruckus                 /usr/share/freeradius/dictionary.ruckus
4874    Redback                /usr/share/freeradius/dictionary.redback
529     Ascend / Lucent        /usr/share/freeradius/dictionary.ascend
1751    Fortinet               /usr/share/freeradius/dictionary.fortinet
12356   F5 Networks            /usr/share/freeradius/dictionary.f5
3076    Checkpoint             /usr/share/freeradius/dictionary.checkpoint
6527    Nokia (Alcatel-Lucent) /usr/share/freeradius/dictionary.nokia
2352    Extreme                /usr/share/freeradius/dictionary.extreme
5771    Riverbed               /usr/share/freeradius/dictionary.riverbed
```

Notable Cisco VSAs (Vendor-Id 9):

```text
Cisco-AVPair (1)              — string; the kitchen-sink vendor attribute
                                 examples:
                                  "shell:priv-lvl=15"
                                  "ip:inacl#100=permit ip any any"
                                  "subscriber:sa=internet(shape=10000000)"
Cisco-NAS-Port (2)             — string version of NAS-Port
Cisco-Disconnect-Cause (24)    — integer; granular disconnect codes
```

Microsoft VSAs (Vendor-Id 311) — the MSCHAPv2 / VPN bundle:

```text
MS-CHAP-Response (1)
MS-CHAP-Error (2)
MS-CHAP-CPW-1 (3)
MS-CHAP-CPW-2 (4)
MS-CHAP-LM-Enc-PW (5)
MS-CHAP-NT-Enc-PW (6)
MS-MPPE-Encryption-Policy (7)
MS-MPPE-Encryption-Type (8)
MS-MPPE-Send-Key (16)
MS-MPPE-Recv-Key (17)
MS-CHAP-Challenge (11)
MS-CHAP-MPPE-Keys (12)
MS-RAS-Vendor (9)
MS-CHAP-Domain (10)
MS-CHAP2-Response (25)
MS-CHAP2-Success (26)
MS-CHAP2-CPW (27)
MS-Primary-DNS-Server (28)
MS-Secondary-DNS-Server (29)
MS-Primary-NBNS-Server (30)
MS-Secondary-NBNS-Server (31)
```

Juniper VSAs (Vendor-Id 2636):

```text
Juniper-Local-User-Name (1)
Juniper-Allow-Commands (2)
Juniper-Deny-Commands (3)
Juniper-Allow-Configuration (4)
Juniper-Deny-Configuration (5)
Juniper-Interactive-Command (8)
Juniper-Configuration-Change (9)
Juniper-User-Permissions (10)
```

Aruba VSAs (Vendor-Id 14823):

```text
Aruba-User-Role (1)
Aruba-User-Vlan (2)
Aruba-Priv-Admin-User (3)
Aruba-Admin-Role (4)
Aruba-Essid-Name (5)
Aruba-Location-Id (6)
Aruba-Port-Id (7)
Aruba-Template-User (8)
Aruba-Named-User-Vlan (9)
Aruba-AP-Group (10)
Aruba-Device-Type (11)
Aruba-AS-Credential-Hash (12)
Aruba-WorkSpace-App-Name (13)
Aruba-Mdps-Device-Udid (14)
Aruba-Mdps-Device-Imei (15)
Aruba-Mdps-Device-Iccid (16)
Aruba-Mdps-Max-Devices (17)
Aruba-Mdps-Device-Name (18)
Aruba-Mdps-Device-Product (19)
Aruba-Mdps-Device-Version (20)
Aruba-Mdps-Device-Serial-No (21)
```

Mikrotik VSAs (Vendor-Id 14988):

```text
Mikrotik-Recv-Limit (1)
Mikrotik-Xmit-Limit (2)
Mikrotik-Group (3)
Mikrotik-Wireless-Forward (4)
Mikrotik-Wireless-Skip-Dot1x (5)
Mikrotik-Wireless-Enc-Algo (6)
Mikrotik-Wireless-Enc-Key (7)
Mikrotik-Rate-Limit (8)
Mikrotik-Realm (9)
Mikrotik-Host-IP (10)
Mikrotik-Mark-Id (11)
Mikrotik-Advertise-URL (12)
Mikrotik-Advertise-Interval (13)
Mikrotik-Recv-Limit-Gigawords (14)
Mikrotik-Xmit-Limit-Gigawords (15)
Mikrotik-Wireless-PSK (16)
```

Ruckus VSAs (Vendor-Id 25053):

```text
Ruckus-User-Groups (1)
Ruckus-SSID (2)
Ruckus-Sta-RSSI (3)
Ruckus-MAC-Address (4)
Ruckus-Tunnel-Type (5)
Ruckus-Wlan-Type (6)
Ruckus-Wlan-Service (7)
Ruckus-VLAN-ID (8)
Ruckus-Role (9)
```

In FreeRADIUS the dictionary files are loaded from `dictionary` (typically `raddb/dictionary` or `share/freeradius/dictionary`) which `$INCLUDE`s every vendor file. Custom vendors go in `raddb/dictionary` so they are not overwritten on package upgrade.

## FreeRADIUS Layout

Debian / Ubuntu:

```text
/etc/freeradius/3.0/
├── radiusd.conf            — main server config (top-level)
├── clients.conf            — list of NAS clients
├── proxy.conf              — proxy / realm config
├── users                   — legacy users file (deprecated; use SQL/LDAP)
├── dictionary              — local dictionary additions
├── policy.d/
│   ├── filter
│   ├── eap
│   ├── operator-name
│   ├── canonicalization
│   └── ...
├── mods-available/         — every available module's config
│   ├── pap
│   ├── chap
│   ├── mschap
│   ├── eap
│   ├── ldap
│   ├── sql
│   ├── files
│   ├── exec
│   ├── expr
│   ├── attr_filter
│   └── ...
├── mods-enabled/           — symlinks into mods-available; only these load
├── sites-available/        — every available virtual server
│   ├── default             — main auth/acct site
│   ├── inner-tunnel        — inner EAP method handler
│   ├── control-socket      — radmin socket
│   ├── status              — Status-Server site
│   ├── coa                 — CoA receiver
│   └── ...
├── sites-enabled/          — symlinks into sites-available
├── certs/                  — server CA + cert + key for EAP-TLS/TTLS/PEAP
│   ├── bootstrap           — script to mint a self-signed CA + server cert
│   ├── ca.pem
│   ├── server.pem
│   ├── server.key
│   └── dh
├── mods-config/
│   ├── files/              — for the files module (default $authorize, etc.)
│   ├── sql/
│   │   └── main/<dialect>/  — schema, queries
│   └── ...
└── trigger.conf
```

RHEL:

```text
/etc/raddb/                 — same structure as /etc/freeradius/3.0/
                              binary is /usr/sbin/radiusd
                              service is radiusd.service
                              user is radiusd
```

Macros and includes:

```text
$INCLUDE filename            — inline another file
${path:filename}             — substitute a defined path
unlang                       — the if/elsif/else mini-language inside virtual servers
```

## clients.conf

`clients.conf` lists every NAS allowed to talk to the server. A client unknown to the server has its packets silently dropped (or logged as `Ignoring request to ...`).

```ini
client switch1 {
    ipaddr           = 10.0.0.1
    secret           = SuperSecret123!
    nastype          = cisco
    shortname        = sw1
    require_message_authenticator = yes
    limit {
        max_connections   = 16
        lifetime          = 0
        idle_timeout      = 30
    }
}

client management-vlan {
    ipaddr           = 10.0.0.0/24
    secret           = NetworkAdminSecret
    nastype          = other
    shortname        = mgmt
}

client all-aps {
    ipv4addr         = 10.10.0.0/16
    secret           = WirelessSecret
    nastype          = aruba
    shortname        = aps
    require_message_authenticator = yes
}

client v6-nas {
    ipv6addr         = 2001:db8:1::/64
    secret           = V6Secret
    shortname        = v6nas
}

client by-fqdn {
    ipaddr           = vpn.example.com
    secret           = VpnSecret
    shortname        = vpn
}

# Dynamic clients: clients learned at runtime from a LDAP/SQL lookup.
client dynamic {
    ipaddr           = 10.20.0.0/16
    netmask          = 24
    dynamic_clients  = dynamic_client_server
    lifetime         = 3600
}
```

The `network/24` (or any prefix) form lets any IP in the block share one secret — useful for AP fleets and VPN concentrators with floating IPs. Per-NAS individual entries are still preferred where possible (per-device secrets, per-device shortnames in logs).

`require_message_authenticator = yes` makes Message-Authenticator AVP mandatory on every Access-Request from this client (set globally in modern installs to mitigate BlastRADIUS / CVE-2024-3596).

`nastype` hints the server about the vendor; the server uses this to apply vendor-appropriate post-auth attribute munging (e.g., Cisco-AVPair shell:priv-lvl).

Dynamic clients (advanced):

```ini
client dynamic_client_server {
    ipaddr            = 10.20.0.0/16
    secret            = template
    dynamic_clients   = dynamic_client_pool
    lifetime          = 3600
}

# In sites-enabled/dynamic_clients_pool:
server dynamic_client_server {
    listen { ... }
    client { ... lookup ... }
}
```

## users

The legacy `users` file (at `mods-config/files/authorize`). Entries use the format:

```text
USERNAME    CHECK-ATTRIBUTES, ...
            REPLY-ATTRIBUTES, ...
```

Indentation matters — reply attributes must be indented with whitespace. The trailing comma on a non-final reply line is mandatory.

```text
# Local check entry
alice    Cleartext-Password := "SecretAlice"
         Reply-Message      = "Welcome, Alice",
         Session-Timeout    = 3600,
         Tunnel-Type        = VLAN,
         Tunnel-Medium-Type = IEEE-802,
         Tunnel-Private-Group-ID = "20"

# MSCHAP-friendly entry (NT-Hash equivalent, for AD-style auth)
bob      NT-Password := 0xCBC501A4D2227783E2E1CE0BF0E0B2A1

# Group via Huntgroup
DEFAULT  Huntgroup-Name == "vpn-users"
         Service-Type        = Framed-User,
         Framed-Protocol     = PPP,
         Framed-IP-Netmask   = 255.255.255.255

# Match-all DEFAULT — fall through to LDAP for anything not matched above
DEFAULT  Auth-Type := LDAP
         Fall-Through = Yes
```

Operators:

```text
=        — set if not already set (assignment in reply)
:=       — set, replacing any existing
==       — exact-match comparison (in check items)
!=       — not equal
<        — less than
>        — greater than
<=       — at most
>=       — at least
=~       — regex match
!~       — regex no-match
=*       — exists
!*       — does not exist
+=       — append (multi-valued)
```

`Fall-Through = Yes` lets the next stanza also match — without it the first match wins and processing stops.

The legacy `users` file is fine for tiny local lists; for anything serious move to SQL or LDAP.

## mods

Modules under `mods-available/`, enabled via symlink into `mods-enabled/`. Each module has its own config block.

```text
pap              — Password Authentication Protocol; cleartext compare
chap             — CHAP; needs cleartext to verify
mschap           — MS-CHAP and MS-CHAPv2; integrates with AD via ntlm_auth
eap              — EAP framing + sub-method dispatch (md5, tls, ttls, peap, mschapv2, fast, sim, aka)
ldap             — LDAP / AD lookups; password or NT-hash retrieval
sql              — SQL backend (radcheck/radreply/radusergroup/radacct)
files            — the legacy users file
exec             — call external program (nss_wrapper, custom AAA)
expr             — expression evaluator (math, string)
attr_filter      — strip attributes (e.g., post-proxy)
redundant        — try modules in order; first to succeed wins
load-balance     — round-robin between modules
unix             — /etc/passwd / /etc/shadow lookup (rare today)
realm            — strip / route by realm
preprocess       — clean up attributes (Calling-Station-Id format normalization)
sradutmp         — write utmp-style records
detail           — write detail accounting log file
linelog          — line-by-line custom logging
sometimes        — chance-of-success debugging tool
always           — always returns ok/fail/etc. (for unang fall-throughs)
date             — Unix-time conversion
counter          — count events (e.g., daily login limit)
soh              — Statement-of-Health (NAP)
otp              — one-time password (HOTP/TOTP via rlm_otp)
yubikey          — YubiKey OTP / U2F
totp             — Time-based OTP
rest             — REST-API call
python / perl    — embedded interpreter for custom logic
sigtran          — SS7 SIGTRAN for EAP-AKA (telco)
```

A module file looks like:

```ini
# mods-available/pap
pap {
    auth_type = PAP
    normalise = yes
}
```

```ini
# mods-available/eap (excerpt)
eap {
    default_eap_type = peap
    timer_expire     = 60
    ignore_unknown_eap_types = no
    cisco_accounting_username_bug = no
    max_sessions = 4096

    md5 { }

    tls-config tls-common {
        private_key_password = whatever
        private_key_file     = /etc/freeradius/3.0/certs/server.key
        certificate_file     = /etc/freeradius/3.0/certs/server.pem
        ca_file              = /etc/freeradius/3.0/certs/ca.pem
        dh_file              = /etc/freeradius/3.0/certs/dh
        ca_path              = /etc/freeradius/3.0/certs
        cipher_list          = "HIGH:!SSLv2:!SSLv3:!TLSv1:!TLSv1.1"
        tls_min_version      = "1.2"
        tls_max_version      = "1.3"
        ecdh_curve           = "prime256v1"
        cache {
            enable      = yes
            lifetime    = 24
            max_entries = 255
        }
        verify {
            tmpdir       = /tmp/radiusd
            client       = "/usr/bin/openssl verify -CAfile /etc/freeradius/3.0/certs/ca.pem"
        }
    }

    tls {
        tls = tls-common
    }

    ttls {
        tls = tls-common
        default_eap_type = mschapv2
        copy_request_to_tunnel = no
        use_tunneled_reply     = no
        virtual_server         = "inner-tunnel"
    }

    peap {
        tls                    = tls-common
        default_eap_type       = mschapv2
        copy_request_to_tunnel = no
        use_tunneled_reply     = no
        virtual_server         = "inner-tunnel"
        require_client_cert    = no
    }

    mschapv2 {
        send_error = no
    }
}
```

```ini
# mods-available/ldap (excerpt)
ldap {
    server      = "ldap.example.com"
    identity    = "cn=radius,ou=service,dc=example,dc=com"
    password    = ServiceUserPassword
    base_dn     = "ou=people,dc=example,dc=com"
    filter      = "(uid=%{User-Name})"
    sasl { }

    user {
        base_dn    = "ou=people,dc=example,dc=com"
        filter     = "(uid=%{User-Name})"
        scope      = "sub"
    }

    group {
        base_dn        = "ou=people,dc=example,dc=com"
        filter         = "(objectClass=posixGroup)"
        membership_attribute = "memberOf"
    }

    options {
        chase_referrals = yes
        rebind          = yes
        net_timeout     = 1
        timeout         = 4
        timelimit       = 3
        idle            = 60
        probes          = 3
        interval        = 3
    }

    tls {
        ca_file  = /etc/ssl/certs/ca-bundle.pem
        require_cert = "demand"
    }

    pool {
        start = 5
        min   = 4
        max   = 10
        spare = 3
        uses  = 0
        lifetime = 0
        idle_timeout = 60
    }
}
```

```ini
# mods-available/sql
sql {
    dialect      = "mysql"
    driver       = "rlm_sql_mysql"

    server       = "127.0.0.1"
    port         = 3306
    login        = "radius"
    password     = "radpass"
    radius_db    = "radius"

    acct_table1  = "radacct"
    acct_table2  = "radacct"
    postauth_table = "radpostauth"
    authcheck_table = "radcheck"
    authreply_table = "radreply"
    groupcheck_table = "radgroupcheck"
    groupreply_table = "radgroupreply"
    usergroup_table  = "radusergroup"

    delete_stale_sessions = yes
    pool {
        start = 5
        min   = 4
        max   = 10
        spare = 3
        uses  = 0
        lifetime = 0
        idle_timeout = 60
    }

    read_clients = yes
    client_table = "nas"
}
```

## sites-enabled

A virtual server is the policy plumbing for one listener (or the inner method of EAP). Each has named sections invoked at the right moment:

```text
listen { }            — bind to ip:port (auth or acct or coa)
authorize { }         — Access-Request lookup; selects Auth-Type
authenticate { }      — actually validate credentials per Auth-Type
post-auth { }         — runs after Accept (and Reject in the Post-Auth-Type Reject sub-section)
preacct { }           — pre-accounting munging
accounting { }        — accounting handlers (sql, detail, linelog)
session { }           — session DB (simultaneous-use enforcement)
pre-proxy { }         — before forwarding to upstream
post-proxy { }        — after upstream answers
```

Default site:

```ini
server default {
    listen {
        type = auth
        ipaddr = *
        port = 0
        limit {
            max_connections = 16
            lifetime        = 0
            idle_timeout    = 30
        }
    }

    listen {
        type = acct
        ipaddr = *
        port = 0
    }

    authorize {
        filter_username
        preprocess
        chap
        mschap
        suffix
        eap {
            ok = return
        }
        files
        sql
        ldap
        expiration
        logintime
        pap
    }

    authenticate {
        Auth-Type PAP {
            pap
        }
        Auth-Type CHAP {
            chap
        }
        Auth-Type MS-CHAP {
            mschap
        }
        digest
        eap
    }

    preacct {
        preprocess
        acct_unique
        suffix
        files
    }

    accounting {
        detail
        unix
        sql
        attr_filter.accounting_response
    }

    session { }

    post-auth {
        update {
            reply: += session-state:
        }
        sql
        exec
        remove_reply_message_if_eap

        Post-Auth-Type REJECT {
            sql
            attr_filter.access_reject
            eap
            remove_reply_message_if_eap
        }
    }

    pre-proxy { }
    post-proxy { eap }
}
```

Inner-tunnel (the inside of TTLS / PEAP):

```ini
server inner-tunnel {
    listen {
        ipaddr = 127.0.0.1
        port   = 18120
        type   = auth
    }

    authorize {
        filter_username
        chap
        mschap
        suffix
        update control { Proxy-To-Realm := LOCAL }
        eap {
            ok = return
        }
        files
        sql
        ldap
        expiration
        logintime
        pap
    }

    authenticate {
        Auth-Type PAP { pap }
        Auth-Type CHAP { chap }
        Auth-Type MS-CHAP { mschap }
        eap
    }

    session { }

    post-auth {
        Post-Auth-Type REJECT {
            attr_filter.access_reject
        }
    }
}
```

CoA listener:

```ini
server coa {
    listen {
        type   = coa
        ipaddr = *
        port   = 3799
    }

    recv-coa {
        update control { Acct-Status-Type := CoA }
        sql
    }

    send-coa { }
}
```

## EAP Methods

EAP = Extensible Authentication Protocol (RFC 3748). RADIUS carries EAP packets inside EAP-Message (79); the outer dialog is multi-round Access-Challenge / Access-Request.

```text
Method            RFC          Cert        Mutual   Identity-priv   Notes
EAP-MD5           3748         no          no       no              deprecated, no MITM resist
EAP-OTP           3748         no          no       no              one-time password
EAP-GTC           3748         no          no       no              generic-token-card
EAP-TLS           5216         both sides  yes      no              modern; cert-based, smartcards
EAP-TTLS          5281         server only yes      yes (tunnel)    inner: PAP, CHAP, MSCHAPv2
EAP-PEAP          drafts       server only yes      yes (tunnel)    Microsoft; inner: MSCHAPv2
EAP-FAST          4851         no/PAC      yes      yes (tunnel)    Cisco; PAC instead of cert
EAP-MSCHAPv2      2759         no          yes      no              inner method, AD-friendly
EAP-SIM           4186         no          yes      yes             GSM SIM, telco roaming
EAP-AKA           4187         no          yes      yes             UMTS USIM
EAP-AKA-prime     5448         no          yes      yes             AKA prime; key-derivation
EAP-PWD           5931         no          yes      no              password-based, no cert
EAP-EKE           6124         no          yes      no              encrypted-key-exchange
EAP-NOOB          9140         no          yes      no              IoT bootstrap
TEAP              7170         server      yes      yes             tunnel-EAP; FAST successor
```

Notes:

```text
EAP-MD5
    only proves the supplicant knows the password; server not authenticated; trivially attacked.
    fine for testing, never for production.

EAP-MSCHAPv2 (alone)
    must run inside a TLS tunnel (PEAP/TTLS) — bare MSCHAPv2 is broken (Moxie's chapcrack 2012).

EAP-TLS
    the gold standard; both ends present X.509 certs; mutual auth; no password to steal.
    requires PKI (CA + per-user certs).

EAP-TTLS
    server presents cert; supplicant validates server; tunnel established.
    inner method is anything (PAP, CHAP, MSCHAPv2); inner identity not exposed in outer.
    favored where AD lives behind LDAP or token-auth is used.

EAP-PEAP
    MS variant of TTLS; server cert outside, MSCHAPv2 inside; AD-tight.
    PEAPv0 (MSCHAPv2 inner) is what most "PEAP" deployments mean.

EAP-FAST
    Cisco; uses Protected Access Credential (PAC) instead of (or in addition to) cert.
    PAC provisioning is the operational pain point.

EAP-SIM / AKA / AKA'
    SIM/USIM card holds the secret; used in carrier Wi-Fi calling and eduroam-like roaming.

EAP-PWD
    Augmented PAKE; resistant to dictionary attacks; no cert needed.
    quietly underused; nice option for password-only environments.
```

## EAP-TLS Setup

```ini
# mods-enabled/eap (relevant subset)
eap {
    default_eap_type = tls
    tls-config tls-common {
        private_key_file = /etc/freeradius/3.0/certs/server.key
        certificate_file = /etc/freeradius/3.0/certs/server.pem
        ca_file          = /etc/freeradius/3.0/certs/ca.pem
        dh_file          = /etc/freeradius/3.0/certs/dh
        tls_min_version  = "1.2"
        verify {
            tmpdir = /tmp/radiusd
            client = "/usr/bin/openssl verify -CAfile /etc/freeradius/3.0/certs/ca.pem"
        }
    }
    tls {
        tls = tls-common
    }
}
```

```bash
# Mint a CA + server cert with the bundled bootstrap
cd /etc/freeradius/3.0/certs
sudo make destroycerts
sudo ./bootstrap

# Issue a client cert for a user
cd /etc/freeradius/3.0/certs
sudo openssl req -newkey rsa:2048 -keyout alice.key -out alice.csr -subj "/CN=alice@example.com"
sudo openssl ca -config ca.cnf -in alice.csr -out alice.pem
sudo openssl pkcs12 -export -in alice.pem -inkey alice.key -certfile ca.pem -out alice.p12

# Distribute alice.p12 to the supplicant; it loads CA + client-cert + key.
```

Inside `authorize { }` of the `default` server, a TLS-Client-Cert-CN check pins users to specific certs:

```ini
authorize {
    if (TLS-Client-Cert-CN) {
        update control {
            Auth-Type := EAP
        }
    }
}

post-auth {
    if (EAP-Type == TLS) {
        if (TLS-Client-Cert-CN != User-Name) {
            reject
        }
    }
}
```

CN/SAN check policy: enforce that the cert subject matches the User-Name (or that the supplicant is in an allow-list group).

## EAP-PEAP Setup

```ini
# mods-enabled/eap
eap {
    default_eap_type = peap
    tls-config tls-common {
        private_key_file = /etc/freeradius/3.0/certs/server.key
        certificate_file = /etc/freeradius/3.0/certs/server.pem
        ca_file          = /etc/freeradius/3.0/certs/ca.pem
        dh_file          = /etc/freeradius/3.0/certs/dh
        tls_min_version  = "1.2"
    }
    peap {
        tls                = tls-common
        default_eap_type   = mschapv2
        virtual_server     = "inner-tunnel"
        require_client_cert = no
    }
    mschapv2 { }
}
```

Inner-tunnel uses MSCHAPv2 — typically forwarded to AD via `ntlm_auth`:

```ini
# mods-available/mschap
mschap {
    use_mppe   = yes
    require_encryption    = yes
    require_strong        = yes
    with_ntdomain_hack    = yes
    ntlm_auth             = "/usr/bin/ntlm_auth --request-nt-key --allow-mschapv2 --username=USER --domain=EXAMPLE --challenge=CHAL --nt-response=RESP"
}
```

## 802.1X

802.1X is the link-layer access-control protocol that wraps EAP over LAN (EAPOL) between supplicant and authenticator, then re-encapsulates the same EAP packets inside RADIUS between authenticator and AAA server.

```text
Supplicant                   Authenticator                  RADIUS server
(laptop)                     (switch / AP)                  (FreeRADIUS)
    |                              |                              |
    |--- EAPOL-Start ------------->|                              |
    |                              |                              |
    |<-- EAP-Request/Identity -----|                              |
    |                              |                              |
    |--- EAP-Response/Identity --->|                              |
    |                              |--- RADIUS Access-Request --->|
    |                              |    (EAP-Message=Identity)    |
    |                              |                              |
    |                              |<-- RADIUS Access-Challenge --|
    |                              |    (EAP-Message=...)         |
    |<-- EAP-Request --------------|                              |
    |                              |                              |
    | ... TLS handshake / inner method exchange ...               |
    |                              |                              |
    |<-- EAP-Success --------------|<-- RADIUS Access-Accept -----|
    |                              |    (Tunnel-Private-Group-ID, |
    |                              |     Filter-Id, etc.)         |
```

EAPOL frames (Ethertype 0x888E):

```text
EAPOL-Start
EAPOL-EAP        — wraps the EAP packet
EAPOL-Logoff
EAPOL-Key        — key material (4-way handshake on Wi-Fi)
EAPOL-Encapsulated-ASF-Alert
```

The authenticator never inspects EAP — it is a relay. All policy lives on the RADIUS server.

## Dynamic VLAN Assignment

The standard tuple in Access-Accept (RFC 3580):

```text
Tunnel-Type              = VLAN          (13)
Tunnel-Medium-Type       = IEEE-802      (6)
Tunnel-Private-Group-ID  = "20"          — the VLAN ID
```

In FreeRADIUS unlang:

```ini
post-auth {
    if (User-Name =~ /^contractor-/) {
        update reply {
            Tunnel-Type             := VLAN
            Tunnel-Medium-Type      := IEEE-802
            Tunnel-Private-Group-ID := "99"
        }
    } elsif (LDAP-Group == "engineering") {
        update reply {
            Tunnel-Type             := VLAN
            Tunnel-Medium-Type      := IEEE-802
            Tunnel-Private-Group-ID := "20"
        }
    } else {
        update reply {
            Tunnel-Type             := VLAN
            Tunnel-Medium-Type      := IEEE-802
            Tunnel-Private-Group-ID := "10"
        }
    }
}
```

VLAN ID can be sent as a numeric string ("20") or with a "VLAN" prefix on some Aruba/Cisco gear (e.g., "vlan20") — verify on a test port. RFC 3580 says numeric.

## CoA / Dynamic Re-auth

To kick a compromised endpoint or move a session between VLANs:

```bash
# Disconnect a session
echo 'User-Name = "alice"
Acct-Session-Id = "0000ABC123"' | \
    radclient -x switch1.example.com:3799 disconnect SuperSecret123!

# Move a user to quarantine VLAN
echo 'User-Name = "alice"
Acct-Session-Id = "0000ABC123"
Tunnel-Type = VLAN
Tunnel-Medium-Type = IEEE-802
Tunnel-Private-Group-ID = "999"' | \
    radclient -x switch1.example.com:3799 coa SuperSecret123!
```

Use cases: NAC posture failure, blocklisted MAC, schedule-based VLAN changes (guest to after-hours quarantine), bandwidth re-shape mid-session.

## radclient

```bash
# Basic Access-Request
echo "User-Name = alice
User-Password = secret
NAS-IP-Address = 127.0.0.1
NAS-Port = 0" | radclient -x localhost:1812 auth testing123

# Accounting Start
echo "User-Name = alice
Acct-Status-Type = Start
Acct-Session-Id = ABC123
NAS-IP-Address = 127.0.0.1" | radclient -x localhost:1813 acct testing123

# Accounting Stop
echo "User-Name = alice
Acct-Status-Type = Stop
Acct-Session-Id = ABC123
Acct-Session-Time = 600
Acct-Input-Octets = 1024
Acct-Output-Octets = 2048
Acct-Terminate-Cause = User-Request" | radclient -x localhost:1813 acct testing123

# Status-Server (keepalive)
echo "Message-Authenticator = 0x00" | radclient -x localhost:1812 status testing123

# Disconnect / CoA
echo "User-Name = alice
Acct-Session-Id = ABC123" | radclient -x switch:3799 disconnect testing123
echo "User-Name = alice
Acct-Session-Id = ABC123
Tunnel-Type = VLAN
Tunnel-Medium-Type = IEEE-802
Tunnel-Private-Group-ID = 99" | radclient -x switch:3799 coa testing123

# Batch (one packet per blank-line-separated stanza in file)
radclient -f requests.txt -x localhost:1812 auth testing123
```

Useful flags:

```text
-x        verbose; show packet contents
-t SEC    timeout per request (default 3)
-r N      retries (default 3)
-c N      send N copies (load test)
-p N      parallel sends
-i ID     starting packet identifier
-P proto  IPv4/IPv6
-d DIR    raddb dir for dictionary
-D DIR    dictionary dir
-s        only print summary stats
-S FILE   read shared secret from file
-f FILE   read attribute stanzas from file
-n N      suppress some output
-q        quiet
-v        version
```

## radtest

A wrapper around `radclient` for the common PAP test:

```bash
# radtest USERNAME PASSWORD SERVER NAS-PORT SECRET [PROTO]
radtest alice secret 127.0.0.1 1812 testing123
radtest alice secret 127.0.0.1:1812 0 testing123 pap
radtest -t mschap alice secret 127.0.0.1:1812 0 testing123
radtest -t chap  alice secret 127.0.0.1:1812 0 testing123
radtest -t pap   alice secret 127.0.0.1:1812 0 testing123
```

`-t` selects the auth flavor: `pap`, `chap`, `mschap`, `eap-md5` (the EAP types that don't require a TLS handshake).

## radmin

`radmin` connects to the running daemon over its control socket — handy for live introspection without restarting.

```bash
sudo radmin -e "stats client"
sudo radmin -e "stats home_server"
sudo radmin -e "stats memory"
sudo radmin -e "stats queue"
sudo radmin -e "debug level 4"
sudo radmin -e "debug file /tmp/radius-debug.log"
sudo radmin -e "set module config modname key value"
sudo radmin -e "show modules"
sudo radmin -e "show xlat"
sudo radmin -e "hup"
```

The control socket lives at the path defined in `sites-enabled/control-socket` (typically `/var/run/freeradius/freeradius.sock`).

## radius-debug

`radiusd -X` is the canonical foreground debug mode. It runs the server in a single thread, logs every packet, every module call, every attribute mutation, every unlang decision, and the post-auth result.

```bash
sudo systemctl stop freeradius
sudo freeradius -X 2>&1 | tee /tmp/radius-debug.log
# now generate the failing request from the supplicant or with radclient
# read the log line-by-line; the failure is always there
```

Reading `-X` output:

```text
(N)  rad_recv: Access-Request packet from host X port Y, id=Z, length=L
(N)    User-Name = "..."                  — incoming attribute trace
(N)  authorize {                          — entering authorize section
(N)    [pap] = noop                       — module returned noop
(N)    [files] users: Matched entry alice
(N)  } # authorize = ok
(N)  Found Auth-Type = PAP
(N)  authenticate {
(N)    [pap] login attempt with password "secret"
(N)    [pap] = ok
(N)  } # authenticate = ok
(N)  post-auth { ... }
(N)  Sent Access-Accept Id Z from X port Y to W port V length 0
```

If a module logs `noop` it didn't run; `reject` it failed; `ok` it succeeded; `fail` it errored. The summary line at the bottom of the request tells you which post-auth fired.

## clients.conf Recipes

```ini
# Single switch
client core-sw1 {
    ipaddr = 10.10.0.10
    secret = SuperSecret
    nastype = cisco
    require_message_authenticator = yes
}

# AP fleet by /16
client ap-fleet {
    ipaddr = 10.20.0.0/16
    secret = APSecret
    nastype = aruba
}

# IPv6
client v6-vpn {
    ipv6addr = 2001:db8:42::/48
    secret = V6VpnSecret
}

# Hostname-resolved (resolved at startup; no DNS-update)
client vpn-cluster {
    ipaddr = vpn.example.com
    secret = HostNameSecret
}

# Per-virtual-server
client guest-portal {
    ipaddr = 10.50.0.0/24
    secret = GuestSecret
    virtual_server = "guest-server"
}

# Dynamic-client template (paired with sites-enabled/dynamic-clients)
client dynamic-pool {
    ipaddr = 10.30.0.0/16
    secret = template
    dynamic_clients = dyn_lookup_server
    lifetime = 3600
}

# CoA-only client (different IP from auth)
client coa-source {
    ipaddr = 10.40.0.5
    secret = CoASecret
    coa_server = coa-pool
}
```

## SQL Backend

Supported dialects: `mysql`, `postgresql`, `sqlite`, `oracle`, `mssql`. Schema files live at `mods-config/sql/main/<dialect>/schema.sql`.

```bash
# core tables (excerpt of the canonical schema)
CREATE TABLE radcheck (
    id          int(11) unsigned NOT NULL auto_increment,
    username    varchar(64) NOT NULL default '',
    attribute   varchar(64) NOT NULL default '',
    op          char(2) NOT NULL DEFAULT '==',
    value       varchar(253) NOT NULL default '',
    PRIMARY KEY (id),
    KEY username (username(32))
);

CREATE TABLE radreply (
    id          int(11) unsigned NOT NULL auto_increment,
    username    varchar(64) NOT NULL default '',
    attribute   varchar(64) NOT NULL default '',
    op          char(2) NOT NULL DEFAULT '=',
    value       varchar(253) NOT NULL default '',
    PRIMARY KEY (id),
    KEY username (username(32))
);

CREATE TABLE radusergroup (
    username    varchar(64) NOT NULL default '',
    groupname   varchar(64) NOT NULL default '',
    priority    int(11) NOT NULL default '1',
    KEY username (username(32))
);

CREATE TABLE radgroupcheck (
    id          int(11) unsigned NOT NULL auto_increment,
    groupname   varchar(64) NOT NULL default '',
    attribute   varchar(64) NOT NULL default '',
    op          char(2) NOT NULL DEFAULT '==',
    value       varchar(253) NOT NULL default '',
    PRIMARY KEY (id),
    KEY groupname (groupname(32))
);

CREATE TABLE radgroupreply (
    id          int(11) unsigned NOT NULL auto_increment,
    groupname   varchar(64) NOT NULL default '',
    attribute   varchar(64) NOT NULL default '',
    op          char(2) NOT NULL DEFAULT '=',
    value       varchar(253) NOT NULL default '',
    PRIMARY KEY (id),
    KEY groupname (groupname(32))
);

CREATE TABLE radacct (
    radacctid           bigint(21) NOT NULL auto_increment,
    acctsessionid       varchar(64) NOT NULL default '',
    acctuniqueid        varchar(32) NOT NULL default '',
    username            varchar(64) NOT NULL default '',
    realm               varchar(64) default '',
    nasipaddress        varchar(15) NOT NULL default '',
    nasportid           varchar(32) default NULL,
    nasporttype         varchar(32) default NULL,
    acctstarttime       datetime NULL default NULL,
    acctupdatetime      datetime NULL default NULL,
    acctstoptime        datetime NULL default NULL,
    acctinterval        int(12) default NULL,
    acctsessiontime     int(12) unsigned default NULL,
    acctauthentic       varchar(32) default NULL,
    connectinfo_start   varchar(50) default NULL,
    connectinfo_stop    varchar(50) default NULL,
    acctinputoctets     bigint(20) default NULL,
    acctoutputoctets    bigint(20) default NULL,
    calledstationid     varchar(50) NOT NULL default '',
    callingstationid    varchar(50) NOT NULL default '',
    acctterminatecause  varchar(32) NOT NULL default '',
    servicetype         varchar(32) default NULL,
    framedprotocol      varchar(32) default NULL,
    framedipaddress     varchar(15) NOT NULL default '',
    framedipv6address   varchar(45) NOT NULL default '',
    framedipv6prefix    varchar(45) NOT NULL default '',
    framedinterfaceid   varchar(44) NOT NULL default '',
    delegatedipv6prefix varchar(45) NOT NULL default '',
    PRIMARY KEY (radacctid),
    UNIQUE KEY acctuniqueid (acctuniqueid),
    KEY username (username),
    KEY framedipaddress (framedipaddress),
    KEY acctsessionid (acctsessionid),
    KEY acctsessiontime (acctsessiontime),
    KEY acctstarttime (acctstarttime),
    KEY acctstoptime (acctstoptime),
    KEY nasipaddress (nasipaddress)
);

CREATE TABLE radpostauth (
    id          int(11) NOT NULL auto_increment,
    username    varchar(64) NOT NULL default '',
    pass        varchar(64) NOT NULL default '',
    reply       varchar(32) NOT NULL default '',
    authdate    timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE nas (
    id          int(10) NOT NULL auto_increment,
    nasname     varchar(128) NOT NULL,
    shortname   varchar(32),
    type        varchar(30) DEFAULT 'other',
    ports       int(5),
    secret      varchar(60) DEFAULT 'secret' NOT NULL,
    server      varchar(64),
    community   varchar(50),
    description varchar(200) DEFAULT 'RADIUS Client',
    PRIMARY KEY (id),
    KEY nasname (nasname)
);
```

```bash
# load schema
mysql -u root -p radius < /etc/freeradius/3.0/mods-config/sql/main/mysql/schema.sql

# add a user
mysql -u root -p radius -e "INSERT INTO radcheck (username, attribute, op, value) VALUES ('alice', 'Cleartext-Password', ':=', 'secret');"

# add VLAN reply
mysql -u root -p radius -e "INSERT INTO radreply (username, attribute, op, value) VALUES ('alice', 'Tunnel-Private-Group-ID', '=', '20');"
```

```bash
sudo apt install freeradius-mysql           # Debian
sudo dnf install freeradius-mysql           # RHEL
sudo ln -s /etc/freeradius/3.0/mods-available/sql /etc/freeradius/3.0/mods-enabled/sql
```

The `nas` table replaces `clients.conf` when `read_clients = yes` in the sql module — handy for runtime NAS provisioning.

## LDAP Backend

```ini
ldap {
    server     = "ldap.example.com"
    port       = 636
    identity   = "cn=radius,ou=service,dc=example,dc=com"
    password   = "BindPass"
    base_dn    = "ou=people,dc=example,dc=com"
    user {
        base_dn = "ou=people,dc=example,dc=com"
        filter  = "(uid=USER)"
    }
    group {
        base_dn              = "ou=groups,dc=example,dc=com"
        filter               = "(objectClass=posixGroup)"
        membership_attribute = "memberOf"
    }
    update {
        control:Password-With-Header   = userPassword
        control:NT-Password            = sambaNTPassword
        reply:Tunnel-Private-Group-ID  = employeeVLAN
    }
    tls {
        ca_file       = /etc/ssl/certs/ca-certificates.crt
        require_cert  = "demand"
    }
}
```

Wire-up in `authorize { }`:

```ini
authorize {
    ldap
    if (control:Auth-Type == LDAP) {
        update control { Auth-Type := PAP }
    }
    pap
}
```

LDAP returns `userPassword` for PAP/CHAP-able users. For MS-CHAP you need `sambaNTPassword` (or AD's NT-hash via ntlm_auth).

## AD Integration

Two production patterns:

```text
1. ntlm_auth + winbind        — classic; works for MSCHAPv2
2. realmd + sssd              — modern; integrates with AD via Kerberos
```

Pattern 1 — Samba/winbind:

```bash
sudo apt install samba winbind krb5-user libpam-winbind libnss-winbind
sudo realm join --user=Administrator EXAMPLE.COM
# OR classic: net ads join -U Administrator
sudo systemctl enable --now winbind

# verify
wbinfo -u                         # list AD users
wbinfo -t                         # test trust
ntlm_auth --request-nt-key --domain=EXAMPLE --username=alice --password=secret
```

```ini
# mods-available/mschap
mschap {
    ntlm_auth = "/usr/bin/ntlm_auth --request-nt-key --allow-mschapv2 --username=USER --domain=EXAMPLE --challenge=CHAL --nt-response=RESP"
}
```

Add `freerad` to the winbindd_priv group so it can read the privileged pipe:

```bash
sudo usermod -aG winbindd_priv freerad
sudo systemctl restart freeradius
```

This is the canonical FreeRADIUS-on-Linux acting as 802.1X server for Windows clients via PEAP-MSCHAPv2.

## Operator Names

`Operator-Name` (RFC 5580) is a string with a one-character namespace prefix:

```text
1   IETF Tag (defaults to FQDN)            "1example.com"
2   Realm-style                            "2example.com"
3   E.212 (mobile country/network code)    "3310410"
4   ICCID                                  "489881..."
0   None                                   "0"
```

In a federation (eduroam), the inner-most NAS attaches its Operator-Name so the home server knows which visited network the user roamed from. Useful for accounting splits and roaming-policy decisions.

## RadSec (RFC 6614)

RadSec wraps RADIUS in TLS over TCP/2083. It replaces the per-NAS shared secret with TLS mutual auth and replaces UDP with reliable delivery. Eduroam uses RadSec end-to-end between federation members.

```ini
# sites-enabled/tls
listen {
    type     = auth+acct
    ipaddr   = *
    port     = 2083
    proto    = tcp
    clients  = radsec
    tls {
        certificate_file = /etc/freeradius/3.0/certs/radsec-server.pem
        private_key_file = /etc/freeradius/3.0/certs/radsec-server.key
        ca_file          = /etc/freeradius/3.0/certs/radsec-ca.pem
        require_client_cert = yes
    }
}

# clients.conf
client radsec {
    ipaddr = 0.0.0.0/0
    proto  = tls
}
```

`radsecproxy`:

```bash
sudo apt install radsecproxy
```

```ini
# /etc/radsecproxy.conf
ListenUDP             *:1812
ListenTCP             *:2083
LogLevel              3
LoopPrevention        on

tls default {
    CACertificateFile     /etc/radsecproxy/ca.pem
    CertificateFile       /etc/radsecproxy/cert.pem
    CertificateKeyFile    /etc/radsecproxy/key.pem
}

server eduroam-de {
    type   = TLS
    host   = etlr1.eduroam.de:2083
    secret = radsec
    tls    = default
}

realm @example.de {
    server = eduroam-de
}

client local-nas {
    type   = UDP
    host   = 10.0.0.0/8
    secret = SharedSecretWithLocalAPs
}
```

## CoA + Disconnect Setup

Server side — bind a CoA listener and define which NAS may reach it:

```ini
# clients.conf — also defines who can SEND CoA into us
client coa-source {
    ipaddr = 10.0.0.50
    secret = CoASecret
    coa_server = "coa-source"
}

# proxy.conf — define the NAS we will SEND CoA TO
home_server coa-target-1 {
    type    = coa
    ipaddr  = 10.20.0.10
    port    = 3799
    secret  = CoASecret
    response_window = 5
}

home_server_pool coa-pool {
    type   = fail-over
    home_server = coa-target-1
}

# clients.conf
client dynamic-coa {
    ipaddr      = 10.20.0.0/16
    secret      = CoASecret
    coa_server  = coa-pool
}
```

Triggering CoA from policy:

```ini
post-auth {
    if (control:Tmp-String-0) {
        update coa {
            User-Name        := User-Name
            Acct-Session-Id  := Acct-Session-Id
            Tunnel-Type             := VLAN
            Tunnel-Medium-Type      := IEEE-802
            Tunnel-Private-Group-ID := "999"
        }
    }
}
```

## Logging

```ini
# mods-available/detail
detail {
    filename = /var/log/freeradius/radacct/detail
    permissions = 0640
    header = "%t"
}

# mods-available/linelog
linelog {
    filename = /var/log/freeradius/linelog
    format   = "Authentication: result=PACKET-TYPE"
    reference = "messages.default"
    messages {
        Access-Accept = "OK   USER@NAS"
        Access-Reject = "FAIL USER@NAS: REASON"
        default       = "OTHR USER@NAS"
    }
}
```

Important log files (Debian path):

```text
/var/log/freeradius/radius.log         — main daemon log
/var/log/freeradius/radacct/<NAS>/...  — detail accounting per-NAS, per-day
/var/log/freeradius/auth.log           — when 'auth = yes' under log{}
/var/log/freeradius/auth_badpass.log   — failed auths with attempted password
/var/log/freeradius/auth_goodpass.log  — succeeded auths with password (DANGEROUS, debug only)
```

```ini
# radiusd.conf top-level
log {
    destination = files
    file        = /var/log/freeradius/radius.log
    syslog_facility = daemon
    stripped_names  = no
    auth        = yes
    auth_badpass = yes
    auth_goodpass = no
    msg_badpass  = "Auth failed for USER from NAS-IP"
    msg_denied   = "You are already logged in"
}
```

## Common Errors

```text
"Access-Reject"
    Generic auth fail. Look at the preceding lines in -X for the actual reason —
    "rlm_pap: ERROR: No password configured for the user", "MSCHAPv2: failed",
    "Login incorrect".

"rlm_pap: ERROR: No password configured for the user"
    The user matched but no Cleartext-Password / Password-With-Header / NT-Password
    attribute came back from the backend. Common with LDAP returning a hashed pwd
    that PAP can't reverse, or when sql returned the wrong column.

"Failed to authenticate the user"
    Generic. The Auth-Type module returned reject; previous lines say which.

"ERROR: Discarding duplicate request from client X port Y - ID: Z"
    The server already answered packet ID Z and is refusing to reprocess.
    Caused by the NAS retransmitting before the answer arrived.
    Fix: increase NAS retransmit timeout, or fix the slow backend.

"WARNING: Auth-Type already set. Not setting to PAP"
    Earlier module already chose Auth-Type. Usually means EAP set it and a later
    module tried again — almost always harmless.

"rlm_eap: SSL_read failed"
    TLS handshake broken. Cert path issues, expired cert, supplicant rejecting CA,
    cipher mismatch, TLS version mismatch.

"Failing the request because cert chain not validated"
    Client cert presented but server can't verify against ca_file. Wrong CA, or
    intermediate not bundled, or cert/CA mismatch.

"EAP/peap: Initiate"
    Normal: server starting a PEAP exchange. Not an error.

"ERROR: shared secret is incorrect"
    Authenticator field validation failed. NAS and server have different secrets.
    Symmetric: the NAS will see the same on its side.

"Discarding response with bad authenticator"
    Reply from a home server doesn't validate. Wrong proxy secret, or someone is
    spoofing.

"ERROR: Unknown vendor"
    Got a Vendor-Specific attribute with a Vendor-Id we don't have a dictionary
    for. Drop in dictionary.<vendor> or add to local dictionary.

"ERROR: No Auth-Type found"
    authorize{} did not select an Auth-Type. Either the user wasn't matched, or
    the password attribute wasn't found, or the EAP module didn't fire because
    EAP-Message was missing.

"WARNING: Skipping client - duplicate request"
    Same as Discarding-duplicate; informational.

"Login OK: [USER] (from client X port N cli MAC)"
    Standard success line. 'cli' is the Calling-Station-Id (MAC address).

"Login incorrect: [USER] (from client X port N cli MAC)"
    Standard failure line.

"Suspicious value Acct-Input-Octets"
    Acct-Input-Octets non-zero on a Start packet — NAS bug or rolled-over counter.

"WARNING: pool: ... is full, allowing extra connections"
    SQL/LDAP pool exhausted; raise pool {max=...}.

"WARNING: Could not find matching client"
    Packet source IP not in clients.conf. Add a matching client { } block.

"Ignoring request to auth address * port 1812 from unknown client X port Y"
    Same as above; some FreeRADIUS versions phrase it this way.

"rlm_ldap: bind as cn=radius,... failed: Invalid credentials"
    The service-bind credentials are wrong. Re-check the identity and password.

"rlm_ldap: User-Password attribute not found in config item list"
    LDAP returned no userPassword for the user. The bind user may lack ACL to
    read it (typical AD problem — the radius bind user must be in
    "Read NTLM-Password" or similar ACL).

"rlm_mschap: FAILED: MS-CHAP2-Response is incorrect"
    MSCHAPv2 hash mismatch. Wrong NT-hash in backend, or domain qualification
    mismatch (with_ntdomain_hack toggling).

"WARNING: Outer and inner identities are not the same user"
    Anonymous outer ID was used but inner identity is something else. This is
    actually NORMAL for privacy-preserving PEAP/TTLS. Ensure copy_request_to_tunnel
    and use_tunneled_reply are set correctly.

"ERROR: Failed reading users file"
    Path/permission problem on mods-config/files/authorize.

"radiusd: Couldn't find configuration for ..."
    Module declared but not configured. Check mods-enabled/<modname>.

"WARNING: Module rejects request"
    A module returned reject — see Module-Failure-Message in the same log.
```

## Common Gotchas

```text
1. NAS not in clients.conf -> silent reject
   broken: no answer ever reaches the NAS; -X says "Ignoring request to ... from
            unknown client".
   fixed:  add a client {} block matching the NAS IP, restart freeradius.

2. Shared secret mismatch
   broken: NAS sees "Discarding response with bad authenticator"; server log
            "Received packet with invalid Message-Authenticator".
   fixed:  align secret on both sides; on Cisco use 'key 0 PLAIN' to ensure
            you're typing it cleartext, not a Type-7 ciphertext.

3. Message-Authenticator missing on EAP packets (BlastRADIUS / CVE-2024-3596)
   broken: post-CVE FreeRADIUS rejects EAP requests without Message-Authenticator.
            The supplicant fails 802.1X authentication with no detail at the
            switch. Server log shows "ERROR: Received packet without Message-
            Authenticator from client X".
   fixed:  set 'require_message_authenticator = yes' globally; upgrade NAS
            firmware that adds Message-Authenticator (Cisco IOS, ArubaOS, etc.).

4. radiusd ran once as root, files now un-writable as freerad
   broken: "Permission denied opening /var/log/freeradius/radacct/.../detail"
   fixed:  chown -R freerad:freerad /var/log/freeradius /etc/freeradius/3.0
            and start the service via systemd, never directly as root.

5. Cleartext-Password vs SSHA-Password vs NT-Password confusion
   broken: PAP says "No password configured for the user" even though LDAP has
            a userPassword.
   fixed:  if the password is hashed (SSHA, MD5-Crypt, ...) use
            Password-With-Header (the rlm_pap module strips the {SSHA} prefix
            and matches). For MSCHAP, you need NT-Password (NT-Hash). For raw
            cleartext, Cleartext-Password.

6. PEAP outer identity vs inner identity mismatch
   broken: outer User-Name is "anonymous@example.com", inner is "alice".
            With copy_request_to_tunnel=no the inner-tunnel doesn't see
            User-Name=alice and rejects.
   fixed:  in eap.peap set virtual_server="inner-tunnel" and ensure that
            inner-tunnel's authorize{} processes User-Name from the inner EAP-
            Identity. Avoid copy_request_to_tunnel except where strictly needed.

7. LDAP bind user lacks ACL on userPassword (or unicodePwd in AD)
   broken: lookup succeeds, attributes missing, "No password configured".
   fixed:  grant the radius service account read on userPassword (or use
            ntlm_auth for AD, which doesn't read NT-hash directly).

8. SQL connection pool exhausted under load
   broken: requests stall, log: "WARNING: Threads in use ... no connections
            available". Auth latency spikes; some packets time out.
   fixed:  raise pool { max = 30; spare = 5 } in mods-enabled/sql; tune DB
            max_connections; add a HA replica.

9. clientacme fails LDAP TLS due to incomplete cert chain
   broken: "rlm_ldap: TLS: peer cert untrusted or revoked".
   fixed:  bundle intermediates into ca_file, or set ca_path to a directory of
            hashed CAs (c_rehash); never set require_cert=never in production.

10. eap module loaded but no inner-tunnel virtual-server
    broken: PEAP/TTLS requests stop after the TLS handshake; "ERROR: Could not
             find virtual server inner-tunnel".
    fixed:  ln -s /etc/freeradius/3.0/sites-available/inner-tunnel
              /etc/freeradius/3.0/sites-enabled/inner-tunnel
            then restart.

11. FreeRADIUS 3.0 vs 3.2 config schema differences
    broken: copying a 3.0 config to a 3.2 install errors with "Failed to parse
             X" or unknown sections.
    fixed:  read /etc/freeradius/3.0/UPGRADE-3.0-3.2.txt; key changes:
             - 'use_tunneled_reply' moved
             - 'session-state' replaces 'inner-tunnel: control:' for some attrs
             - 'cipher_list' default tightened
             - Operator-Name decoding stricter

12. Port 1812 default but NAS sends to 1645 (or vice versa)
    broken: server listening only on 1812, switch configured for 1645 — packets
             never arrive.
    fixed:  add another listen{port=1645} or change the switch:
             cisco: 'radius server X / address ipv4 X auth-port 1812 acct-port 1813'

13. NAS-Identifier vs NAS-IP-Address proxy confusion
    broken: in a proxy chain, the home server sees the proxy's IP and rejects.
    fixed:  use NAS-Identifier for identity; let the proxy preserve the original
             via Operator-Name or a custom Class attribute.

14. Detail accounting file ownership wrong after rotation
    broken: logrotate creates the file as root:root; freerad can't write.
    fixed:  put 'create 0640 freerad freerad' in the logrotate stanza.

15. UTF-8 usernames truncated at 64 octets in radacct
    broken: long realm-qualified usernames truncated, breaking joins.
    fixed:  ALTER TABLE radacct MODIFY username VARCHAR(253), and the same on
             radcheck/radreply.

16. Multiple home_servers in a fail-over pool, but the dead one keeps getting
    tried because Status-Server isn't enabled.
    broken: every Nth request slow.
    fixed:  enable status_check = status-server in the home_server stanza, and
             ensure the upstream RADIUS supports Status-Server (RFC 5997).

17. Calling-Station-Id format differences (AA:BB:CC:DD:EE:FF vs AA-BB-...)
    broken: SQL exact-match policy doesn't fire because of dash-vs-colon.
    fixed:  use rewrite policies in policy.d/canonicalization to normalize MAC
             format upon receipt.
```

## Diagnostic Tools

```bash
# Foreground full debug (canonical)
sudo freeradius -X

# Capture RADIUS on the wire
sudo tcpdump -i any -nn -vvv -s 0 'port 1812 or port 1813 or port 3799'

# Save for Wireshark; Wireshark has a built-in RADIUS dissector
sudo tcpdump -i any -w /tmp/radius.pcap 'port 1812 or port 1813 or port 3799'
wireshark /tmp/radius.pcap

# Live-trace a single user without restarting the daemon
sudo raddebug -t 60 -u alice
# attaches to the control socket; prints all log lines that mention alice for 60s

# raddrelay — replay accounting packets from a detail file to a different server
raddrelay -f /var/log/freeradius/radacct/10.0.0.1/detail-20260425 \
          -d /etc/freeradius/3.0/dictionary \
          -s testing123 backup-radius:1813

# Validate config
sudo freeradius -C
# parses everything but doesn't start; exits non-zero on error

# Show effective config including INCLUDE expansions
sudo freeradius -X -d /etc/freeradius/3.0 -n radiusd

# Watch detail accounting in realtime
sudo tail -F /var/log/freeradius/radacct/10.0.0.1/detail-20260425
```

## radsecproxy

`radsecproxy` is the Swiss-army TLS-proxy for RADIUS. Use it to:

```text
- terminate RadSec for organizations whose backend AAA is plain UDP
- federate via TLS to upstream confederations (eduroam, OpenRoaming)
- decouple shared-secret universe from organizational boundaries
- add LoopPrevention and proxy-state across multi-hop chains
```

```ini
# /etc/radsecproxy.conf

ListenUDP        *:1812
ListenTCP        *:2083
LogLevel         3
LogDestination   file:///var/log/radsecproxy.log
LoopPrevention   on

tls default {
    CACertificateFile  /etc/radsecproxy/eduroam-ca.pem
    CertificateFile    /etc/radsecproxy/our-cert.pem
    CertificateKeyFile /etc/radsecproxy/our-key.pem
    PolicyOID          1.3.6.1.4.1.25178.3.1.1
}

server etlr1 {
    type   = TLS
    host   = etlr1.eduroam.org:2083
    secret = radsec
    tls    = default
    StatusServer = on
}

server etlr2 {
    type   = TLS
    host   = etlr2.eduroam.org:2083
    secret = radsec
    tls    = default
    StatusServer = on
}

realm DEFAULT {
    server = etlr1
    server = etlr2
}

realm @example.org {
    replymessage = "Welcome home"
    server       = local-radius
}

server local-radius {
    type        = UDP
    host        = 127.0.0.1:1812
    secret      = sharedlocal
    StatusServer = on
}

client local-aps {
    type   = UDP
    host   = 10.0.0.0/8
    secret = APsecret
}
```

The eduroam reference deployment uses radsecproxy as the visited- and home-network entry points; institutional FreeRADIUS sits behind it.

## Cisco AAA Integration

```bash
! IOS / IOS-XE
aaa new-model

radius server PRIMARY
 address ipv4 10.0.0.5 auth-port 1812 acct-port 1813
 key 0 SuperSecret123!
 timeout 5
 retransmit 3
 automate-tester username probe-user probe-on
!
radius server SECONDARY
 address ipv4 10.0.0.6 auth-port 1812 acct-port 1813
 key 0 SuperSecret123!
!
aaa group server radius RADIUS-GRP
 server name PRIMARY
 server name SECONDARY
 ip radius source-interface Loopback0
!
aaa authentication login default group RADIUS-GRP local
aaa authentication dot1x default group RADIUS-GRP
aaa authorization exec default group RADIUS-GRP if-authenticated
aaa authorization network default group RADIUS-GRP
aaa accounting exec default start-stop group RADIUS-GRP
aaa accounting dot1x default start-stop group RADIUS-GRP
aaa accounting update newinfo periodic 5
!
! 802.1X global
dot1x system-auth-control
!
! Per-port
interface GigabitEthernet1/0/1
 switchport mode access
 authentication host-mode multi-domain
 authentication port-control auto
 mab
 dot1x pae authenticator
 dot1x timeout tx-period 5
!
! CoA listener
aaa server radius dynamic-author
 client 10.0.0.5 server-key SuperSecret123!
 port 3799
!
! Show / debug
show aaa servers
show authentication sessions
show dot1x interface Gi1/0/1 details
debug radius
debug radius authentication
debug aaa authentication
debug dot1x all
```

`key 7` denotes Cisco's Type-7 reversible encryption — useful only for hiding the password from over-the-shoulder eyes; never for security. Use `key 0 <plain>` to type cleartext, or use the device's secure key store.

## Juniper RADIUS

```bash
# JunOS
set system radius-server 10.0.0.5 secret "SuperSecret123!"
set system radius-server 10.0.0.5 source-address 10.0.0.1
set system radius-server 10.0.0.5 retry 3
set system radius-server 10.0.0.5 timeout 5
set system radius-server 10.0.0.5 port 1812
set system radius-server 10.0.0.5 accounting-port 1813
set system radius-server 10.0.0.6 secret "SuperSecret123!"
set system authentication-order [ radius password ]
set system accounting destination radius server 10.0.0.5 secret "SuperSecret123!"
set system accounting events login
set system accounting events change-log
set system accounting events interactive-commands

# 802.1X
set protocols dot1x authenticator authentication-profile-name dot1x-profile
set access profile dot1x-profile authentication-order radius
set access profile dot1x-profile radius authentication-server 10.0.0.5
set access profile dot1x-profile radius accounting-server 10.0.0.5
set protocols dot1x authenticator interface ge-0/0/1.0 supplicant single

# verify
show system radius
show network-access aaa statistics
show network-access aaa subscribers
show dot1x interface ge-0/0/1
```

## Idioms

```text
"Always use Message-Authenticator on every EAP request"
   It is required by the EAP RFC and post-BlastRADIUS by every modern server.
   Set 'require_message_authenticator = yes' globally.

"Default to RadSec for inter-organization RADIUS"
   Shared secrets across org boundaries are a perpetual operational burden and
   a CVE waiting to happen. Use TLS mutual auth instead.

"Always have a local fallback"
   Even with RADIUS in 'aaa authentication login default group RADIUS-GRP local'
   put 'local' last so a console login still works when AAA is down.

"Use Cleartext-Password only when no MSCHAPv2 is needed"
   PAP/CHAP with cleartext is fine for VPN with strong transport security; for
   802.1X over PEAP-MSCHAPv2 you need NT-Password (NT-hash) instead.

"PEAP for AD-integrated 802.1X, EAP-TLS for cert-based"
   PEAP is the path of least resistance with AD passwords; EAP-TLS is the gold
   standard when you can mint per-user certs.

"Tune the SQL pool for peak"
   At 802.1X reauth-storm time (school day start, conference event start) the
   SQL pool must absorb a flood. Plan for 5x average concurrency.

"Log every Access-Reject"
   The detail+linelog combo gives you a forensic trail without the Access-
   Accept volume.

"Test with radclient before involving the NAS"
   Most config errors show up in radclient -x faster than in a switch debug.

"Every NAS gets its own secret"
   Per-NAS shared secrets limit blast radius if one is leaked.

"Keep a 'denyall' default"
   In policy, the default after all matching rules should be reject; never
   accept-by-default.

"Prefer NAS-Identifier over NAS-IP-Address for identity"
   IPs change behind NAT or in a HA pair; an explicit NAS-Identifier doesn't.

"Keep Acct-Interim-Interval reasonable"
   30s is too aggressive (RADIUS overload); 1800s is too coarse (poor billing).
   300-600s is the sweet spot.

"Stash session context in Class"
   Class (25) round-trips through accounting unchanged. Stuff a session-id or
   correlation token there at Access-Accept time and read it back from
   Acct-Status-Type=Stop for clean joins.
```

## See Also

- kerberos
- tacacs
- ssh
- tls
- openssl
- ldap
- polyglot

## References

- RFC 2865 — Remote Authentication Dial In User Service (RADIUS) — https://datatracker.ietf.org/doc/html/rfc2865
- RFC 2866 — RADIUS Accounting — https://datatracker.ietf.org/doc/html/rfc2866
- RFC 2867 — RADIUS Accounting Modifications for Tunnel Protocol Support — https://datatracker.ietf.org/doc/html/rfc2867
- RFC 2868 — RADIUS Attributes for Tunnel Protocol Support — https://datatracker.ietf.org/doc/html/rfc2868
- RFC 2869 — RADIUS Extensions — https://datatracker.ietf.org/doc/html/rfc2869
- RFC 3162 — RADIUS and IPv6 — https://datatracker.ietf.org/doc/html/rfc3162
- RFC 3580 — IEEE 802.1X RADIUS Usage Guidelines — https://datatracker.ietf.org/doc/html/rfc3580
- RFC 3748 — Extensible Authentication Protocol (EAP) — https://datatracker.ietf.org/doc/html/rfc3748
- RFC 4186 — EAP-SIM — https://datatracker.ietf.org/doc/html/rfc4186
- RFC 4187 — EAP-AKA — https://datatracker.ietf.org/doc/html/rfc4187
- RFC 4372 — Chargeable User Identity (CUI) — https://datatracker.ietf.org/doc/html/rfc4372
- RFC 5080 — Common RADIUS Implementation Issues — https://datatracker.ietf.org/doc/html/rfc5080
- RFC 5176 — Dynamic Authorization Extensions to RADIUS (CoA / Disconnect) — https://datatracker.ietf.org/doc/html/rfc5176
- RFC 5216 — EAP-TLS Authentication Protocol — https://datatracker.ietf.org/doc/html/rfc5216
- RFC 5281 — EAP-TTLS — https://datatracker.ietf.org/doc/html/rfc5281
- RFC 5448 — EAP-AKA' — https://datatracker.ietf.org/doc/html/rfc5448
- RFC 5580 — Carrying Location Objects in RADIUS — https://datatracker.ietf.org/doc/html/rfc5580
- RFC 5931 — EAP-PWD — https://datatracker.ietf.org/doc/html/rfc5931
- RFC 5997 — Status-Server Packets in RADIUS — https://datatracker.ietf.org/doc/html/rfc5997
- RFC 6158 — RADIUS Design Guidelines — https://datatracker.ietf.org/doc/html/rfc6158
- RFC 6613 — RADIUS over TCP — https://datatracker.ietf.org/doc/html/rfc6613
- RFC 6614 — RADIUS over TLS (RadSec) — https://datatracker.ietf.org/doc/html/rfc6614
- RFC 6929 — RADIUS Protocol Extensions — https://datatracker.ietf.org/doc/html/rfc6929
- RFC 7170 — TEAP — https://datatracker.ietf.org/doc/html/rfc7170
- RFC 7585 — Dynamic Peer Discovery for RADSEC — https://datatracker.ietf.org/doc/html/rfc7585
- RFC 8044 — Data Types in RADIUS — https://datatracker.ietf.org/doc/html/rfc8044
- RFC 8559 — Dynamic Authorization Proxying in RADIUS — https://datatracker.ietf.org/doc/html/rfc8559
- FreeRADIUS documentation — https://freeradius.org/documentation/
- FreeRADIUS GitHub — https://github.com/FreeRADIUS/freeradius-server
- eduroam — https://eduroam.org/
- BlastRADIUS / CVE-2024-3596 — https://www.blastradius.fail/
- radsecproxy — https://radsecproxy.github.io/
- IANA RADIUS attribute registry — https://www.iana.org/assignments/radius-types/radius-types.xhtml
