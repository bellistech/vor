# SNMP (Simple Network Management Protocol, RFC 1157 / 3410-3418)

> UDP-based polling and notification protocol for monitoring and managing network devices via OIDs, MIB trees, and (in v3) USM-authenticated, encrypted channels.

## Quick Reference

```bash
snmpwalk    -v2c -c public  192.0.2.1 system                  # walk a subtree
snmpget     -v2c -c public  192.0.2.1 sysDescr.0              # single OID
snmpbulkwalk -v2c -c public 192.0.2.1 ifTable                 # GETBULK walk
snmpset     -v2c -c private 192.0.2.1 sysContact.0 s "noc@x"  # write
snmptrap    -v2c -c public  10.0.0.5 '' coldStart             # send trap
```

```bash
# v3 authPriv (production)
snmpwalk -v3 -l authPriv \
  -u monUser -a SHA -A 'AuthPass!' \
  -x AES    -X 'PrivPass!' \
  192.0.2.1 ifHCInOctets
```

## SNMP in 60 Seconds

UDP/161 (agent), UDP/162 (trap receiver). The Manager (NMS) issues GETs;
the Agent on the managed device responds. SMI (Structure of Management
Information, ASN.1 subset) **describes** the data; the MIB
(Management Information Base) is the **tree** of objects identified by
OIDs (Object Identifiers, dotted-decimal). PDU types:

```
GET       — fetch one OID exactly
GETNEXT   — fetch lexicographic-next OID (used to walk)
GETBULK   — fetch many lex-next OIDs in one round-trip (v2c+)
SET       — write a writable OID
RESPONSE  — reply to GET/GETNEXT/GETBULK/SET
TRAP      — unsolicited notify, no ACK (v1 has its own format; v2c/v3 reuse PDU)
INFORM    — like TRAP but with RESPONSE-style ACK (v2c+)
REPORT    — engine-discovery / time-sync errors (v3 only)
```

The whole protocol is a **walk over a single global tree** of OIDs. Every
counter, every interface name, every BGP peer state lives somewhere on
that tree. Learn the tree, learn the protocol.

## Versions

```
SNMPv1   RFC 1157, 1988.   32-bit Counter32 only. Plain-text community.
                            No GETBULK. PDU types: Get, GetNext, Set, Response, Trap.
                            Errors: noSuchName, badValue, readOnly, genErr.
SNMPv2c  RFC 1901-1908.    Adds GETBULK, INFORM, Counter64 (high-capacity).
                            Still community-based authentication ("c" = community).
                            New error codes: noAccess, wrongType, wrongLength,
                            wrongValue, noCreation, inconsistentValue,
                            commitFailed, undoFailed, authorizationError.
SNMPv3   RFC 3411-3418.    USM (User-based Security): MD5/SHA HMAC + DES/3DES/AES.
                            VACM (View-based Access Control).
                            Security levels: noAuthNoPriv, authNoPriv, authPriv.
                            Engine ID and Engine Boots/Time replay protection.
SNMPv2u, v2*               Historical, deprecated. Ignore.
```

Wire compatibility: v2c speakers cannot decode v3 PDUs and vice versa.
Most agents listen for all three; managers MUST select one. Never deploy
v1 in 2026 — v1's only counter (`Counter32`) wraps at 4.29 GB which
overflows on a 1 Gbps link in 34 seconds.

## OIDs and the MIB Tree

Every object in SNMP is named by an OID — a dotted sequence of
non-negative integers — corresponding to a path in the global MIB tree:

```
                                 (root)
                                   |
       +---------------------------+---------------------------+
       0 (ccitt)                   1 (iso)                     2 (joint-iso-ccitt)
                                   |
                                   3 (org)
                                   |
                                   6 (dod)
                                   |
                                   1 (internet) ............... 1.3.6.1
                                   |
       +-------+-------+-----------+-------+-------+-----------+
       1       2       3           4       5       6
     directory  mgmt  experimental private  security  snmpV2
                |                  |
                1 (mib-2) 1.3.6.1.2.1   1 (enterprises) 1.3.6.1.4.1
                |                       |
        +-------+-------+               +- 9      Cisco
        1 system                        +- 2636   Juniper
        2 interfaces                    +- 30065  Arista
        3 at (deprecated)               +- 8072   net-snmp
        4 ip                            +- 8741   F5
        5 icmp                          +- 11     HP
        6 tcp                           +- 6876   VMware
        7 udp                           +- ...    everyone else
        ...
        25 host
        31 ifMIB (HC counters)
```

Anything under `1.3.6.1.4.1.X` is a **private enterprise MIB** for vendor
X (IANA-registered). Anything under `1.3.6.1.2.1` is **standard** (mib-2).

### 30+ Most-Queried OIDs

```
# system (1.3.6.1.2.1.1)
sysDescr.0           1.3.6.1.2.1.1.1.0      # device description, free-form string
sysObjectID.0        1.3.6.1.2.1.1.2.0      # vendor enterprise OID for sysObjectID
sysUpTime.0          1.3.6.1.2.1.1.3.0      # TimeTicks (1/100 sec) since agent (re)started
sysContact.0         1.3.6.1.2.1.1.4.0      # admin contact (writable)
sysName.0            1.3.6.1.2.1.1.5.0      # hostname (writable)
sysLocation.0        1.3.6.1.2.1.1.6.0      # physical location (writable)
sysServices.0        1.3.6.1.2.1.1.7.0      # bitmask of OSI layers offered

# IF-MIB ifTable (1.3.6.1.2.1.2.2.1.x), 32-bit
ifIndex              .1.3.6.1.2.1.2.2.1.1   # integer index
ifDescr              .1.3.6.1.2.1.2.2.1.2   # interface name (kernel-style)
ifType               .1.3.6.1.2.1.2.2.1.3   # 6=ether, 24=lo, 53=propVirtual, 131=tunnel
ifMtu                .1.3.6.1.2.1.2.2.1.4
ifSpeed              .1.3.6.1.2.1.2.2.1.5   # Gauge32 bps; saturates at 4.29 Gbps
ifPhysAddress        .1.3.6.1.2.1.2.2.1.6   # MAC
ifAdminStatus        .1.3.6.1.2.1.2.2.1.7   # 1=up,2=down,3=testing
ifOperStatus         .1.3.6.1.2.1.2.2.1.8   # 1=up,2=down,3=testing,4=unknown,5=dormant,6=notPresent,7=lowerLayerDown
ifLastChange         .1.3.6.1.2.1.2.2.1.9   # TimeTicks since last status change
ifInOctets           .1.3.6.1.2.1.2.2.1.10  # Counter32 — wraps fast on >100Mb
ifInUcastPkts        .1.3.6.1.2.1.2.2.1.11
ifInDiscards         .1.3.6.1.2.1.2.2.1.13
ifInErrors           .1.3.6.1.2.1.2.2.1.14
ifOutOctets          .1.3.6.1.2.1.2.2.1.16  # Counter32
ifOutDiscards        .1.3.6.1.2.1.2.2.1.19
ifOutErrors          .1.3.6.1.2.1.2.2.1.20

# IF-MIB ifXTable (1.3.6.1.2.1.31.1.1.1.x) — high-capacity / 64-bit
ifName               .1.3.6.1.2.1.31.1.1.1.1   # short name (Gi0/0/1, ge-0/0/0)
ifInMulticastPkts    .1.3.6.1.2.1.31.1.1.1.2
ifInBroadcastPkts    .1.3.6.1.2.1.31.1.1.1.3
ifOutMulticastPkts   .1.3.6.1.2.1.31.1.1.1.4
ifOutBroadcastPkts   .1.3.6.1.2.1.31.1.1.1.5
ifHCInOctets         .1.3.6.1.2.1.31.1.1.1.6   # Counter64 — use these on >=1 Gbps
ifHCInUcastPkts      .1.3.6.1.2.1.31.1.1.1.7
ifHCInMulticastPkts  .1.3.6.1.2.1.31.1.1.1.8
ifHCInBroadcastPkts  .1.3.6.1.2.1.31.1.1.1.9
ifHCOutOctets        .1.3.6.1.2.1.31.1.1.1.10  # Counter64
ifHCOutUcastPkts     .1.3.6.1.2.1.31.1.1.1.11
ifHCOutMulticastPkts .1.3.6.1.2.1.31.1.1.1.12
ifHCOutBroadcastPkts .1.3.6.1.2.1.31.1.1.1.13
ifLinkUpDownTrapEnable .1.3.6.1.2.1.31.1.1.1.14
ifHighSpeed          .1.3.6.1.2.1.31.1.1.1.15  # speed in Mbps (works above 4 Gbps)
ifPromiscuousMode    .1.3.6.1.2.1.31.1.1.1.16
ifConnectorPresent   .1.3.6.1.2.1.31.1.1.1.17
ifAlias              .1.3.6.1.2.1.31.1.1.1.18  # admin description (writable)
ifCounterDiscontinuityTime .1.3.6.1.2.1.31.1.1.1.19

# IP-MIB (1.3.6.1.2.1.4)
ipForwarding.0       .1.3.6.1.2.1.4.1.0     # 1=router, 2=host
ipDefaultTTL.0       .1.3.6.1.2.1.4.2.0
ipInReceives.0       .1.3.6.1.2.1.4.3.0
ipInDiscards.0       .1.3.6.1.2.1.4.8.0
ipForwDatagrams.0    .1.3.6.1.2.1.4.6.0

# TCP-MIB (1.3.6.1.2.1.6)
tcpRtoMin.0          .1.3.6.1.2.1.6.2.0
tcpMaxConn.0         .1.3.6.1.2.1.6.4.0
tcpActiveOpens.0     .1.3.6.1.2.1.6.5.0
tcpCurrEstab.0       .1.3.6.1.2.1.6.9.0     # current TCP ESTABLISHED count

# UDP-MIB (1.3.6.1.2.1.7)
udpInDatagrams.0     .1.3.6.1.2.1.7.1.0
udpNoPorts.0         .1.3.6.1.2.1.7.2.0
udpInErrors.0        .1.3.6.1.2.1.7.3.0
udpOutDatagrams.0    .1.3.6.1.2.1.7.4.0

# HOST-RESOURCES-MIB (1.3.6.1.2.1.25)
hrSystemUptime.0     .1.3.6.1.2.1.25.1.1.0
hrSystemNumUsers.0   .1.3.6.1.2.1.25.1.5.0
hrSystemProcesses.0  .1.3.6.1.2.1.25.1.6.0
hrMemorySize.0       .1.3.6.1.2.1.25.2.2.0    # KB
hrStorageDescr       .1.3.6.1.2.1.25.2.3.1.3
hrStorageAllocationUnits .1.3.6.1.2.1.25.2.3.1.4
hrStorageSize        .1.3.6.1.2.1.25.2.3.1.5
hrStorageUsed        .1.3.6.1.2.1.25.2.3.1.6
hrProcessorLoad      .1.3.6.1.2.1.25.3.3.1.2  # 0-100% per core
hrSWRunName          .1.3.6.1.2.1.25.4.2.1.2  # process basename
hrSWRunStatus        .1.3.6.1.2.1.25.4.2.1.7
hrSWRunPerfCPU       .1.3.6.1.2.1.25.5.1.1.1  # CPU centi-seconds since process start
hrSWRunPerfMem       .1.3.6.1.2.1.25.5.1.1.2  # KB

# BGP4-MIB (1.3.6.1.2.1.15)
bgpPeerState         .1.3.6.1.2.1.15.3.1.2    # 1=idle 2=connect 3=active 4=opensent 5=openconfirm 6=established
bgpPeerAdminStatus   .1.3.6.1.2.1.15.3.1.3
bgpPeerLocalAddr     .1.3.6.1.2.1.15.3.1.5
bgpPeerLocalAs       .1.3.6.1.2.1.15.3.1.7
bgpPeerRemoteAddr    .1.3.6.1.2.1.15.3.1.8
bgpPeerRemoteAs      .1.3.6.1.2.1.15.3.1.9

# OSPF-MIB (1.3.6.1.2.1.14)
ospfRouterId.0       .1.3.6.1.2.1.14.1.1.0
ospfNbrState         .1.3.6.1.2.1.14.10.1.6   # 8=Full

# ENTITY-MIB (1.3.6.1.2.1.47)
entPhysicalDescr     .1.3.6.1.2.1.47.1.1.1.1.2
entPhysicalClass     .1.3.6.1.2.1.47.1.1.1.1.5  # 3=chassis 6=power 7=fan 9=module 10=port
entPhysicalSerialNum .1.3.6.1.2.1.47.1.1.1.1.11

# SNMP framework (1.3.6.1.6.3)
snmpEngineID.0       .1.3.6.1.6.3.10.2.1.1.0
snmpEngineBoots.0    .1.3.6.1.6.3.10.2.1.2.0
snmpEngineTime.0     .1.3.6.1.6.3.10.2.1.3.0
snmpTrapOID.0        .1.3.6.1.6.3.1.1.4.1.0    # populated in v2c/v3 trap varbinds
sysUpTimeInstance    .1.3.6.1.2.1.1.3.0        # also in trap varbinds
```

The `.0` suffix on a scalar means "the (only) instance". For columnar
objects (table cells) the suffix is the row index (often `ifIndex` for
interface tables, but can be a multi-element index like an IP+port).

## SMI (Structure of Management Information)

SMI is an ASN.1 subset that defines how MIB modules are written. SMIv1
(RFC 1155, deprecated) is similar to SMIv2 (RFC 2578-2580) but the
latter is what every modern device speaks.

### OBJECT-TYPE Macro

```asn.1
ifInOctets OBJECT-TYPE
    SYNTAX      Counter32
    MAX-ACCESS  read-only
    STATUS      current
    DESCRIPTION
        "The total number of octets received on the interface,
         including framing characters."
    ::= { ifEntry 10 }
```

Field meanings:

```
SYNTAX        ASN.1 base type or textual-convention
              INTEGER, INTEGER32, Unsigned32, Gauge32, Counter32, Counter64,
              TimeTicks, OCTET STRING, OBJECT IDENTIFIER, IpAddress, BITS,
              SEQUENCE, SEQUENCE OF Foo (table rows)
MAX-ACCESS    not-accessible | accessible-for-notify | read-only |
              read-write | read-create
STATUS        current | deprecated | obsolete
DESCRIPTION   human-readable
INDEX         (only on table entries) — list of objects forming the row index
::=           assigns this object's OID under its parent
```

### Tables — Entry-with-INDEX

```asn.1
ifTable OBJECT-TYPE
    SYNTAX      SEQUENCE OF IfEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    DESCRIPTION "..."
    ::= { interfaces 2 }

ifEntry OBJECT-TYPE
    SYNTAX      IfEntry
    MAX-ACCESS  not-accessible
    STATUS      current
    INDEX       { ifIndex }
    ::= { ifTable 1 }

IfEntry ::= SEQUENCE {
    ifIndex         InterfaceIndex,
    ifDescr         DisplayString,
    ifType          IANAifType,
    ifMtu           Integer32,
    ifSpeed         Gauge32,
    ifPhysAddress   PhysAddress,
    ifAdminStatus   INTEGER,
    ifOperStatus    INTEGER,
    ...
}
```

Walking `ifTable` lex-orders by column-then-row in v1 SMI but most
implementations return *row-major* in practice. Use `snmptable` for a
clean tabular render.

### Textual Conventions

`TEXTUAL-CONVENTION` defines a named subtype of a base SYNTAX with a
DISPLAY-HINT. Examples:

```
DisplayString          OCTET STRING (SIZE (0..255)), printable ASCII
PhysAddress            OCTET STRING — DISPLAY-HINT "1x:" → "00:11:22:..."
DateAndTime            OCTET STRING (SIZE (8|11)), encodes Y/M/D/H/M/S/dS/TZ
TruthValue             INTEGER { true(1), false(2) }
RowStatus              INTEGER { active(1), notInService(2), notReady(3),
                                 createAndGo(4), createAndWait(5), destroy(6) }
StorageType            INTEGER { other(1), volatile(2), nonVolatile(3),
                                 permanent(4), readOnly(5) }
TestAndIncr            Integer32 — write must equal read+1; used as soft-lock
TimeStamp              TimeTicks — sysUpTime when event occurred
```

### MIB Module Header

```asn.1
EXAMPLE-MIB DEFINITIONS ::= BEGIN
    IMPORTS
        OBJECT-TYPE, MODULE-IDENTITY, Counter64,
        enterprises FROM SNMPv2-SMI
        DisplayString FROM SNMPv2-TC;

    exampleMIB MODULE-IDENTITY
        LAST-UPDATED  "202604010000Z"
        ORGANIZATION  "Example, Inc."
        CONTACT-INFO  "noc@example.com"
        DESCRIPTION   "..."
        REVISION      "202604010000Z"
        DESCRIPTION   "Initial revision."
        ::= { enterprises 99999 }
END
```

## PDU Anatomy

### v2c Wire Format (BER-encoded ASN.1)

```
+--------------------------------------------------------------+
| SEQUENCE  (whole SNMP message)                               |
|   INTEGER       version           (0=v1, 1=v2c, 3=v3)        |
|   OCTET STRING  community         (e.g. "public")            |
|   PDU           [contextual tag]                             |
|     INTEGER       request-id                                 |
|     INTEGER       error-status    (or non-repeaters for BULK)|
|     INTEGER       error-index     (or max-repetitions)       |
|     SEQUENCE OF VarBind                                      |
|       SEQUENCE                                               |
|         OBJECT IDENTIFIER  oid                               |
|         <value>            value (or NULL on requests)       |
|       ...                                                    |
+--------------------------------------------------------------+
```

PDU type tags:

```
[0] GetRequest          0xA0
[1] GetNextRequest      0xA1
[2] GetResponse         0xA2  (v1 name; v2c+ rename to "Response")
[3] SetRequest          0xA3
[4] Trap-PDU (v1 only)  0xA4   — different layout (enterprise, agent-addr,
                                  generic-trap, specific-trap, time-stamp,
                                  varbinds)
[5] GetBulkRequest      0xA5  (v2c+)
[6] InformRequest       0xA6  (v2c+)
[7] SNMPv2-Trap         0xA7  (v2c+)
[8] Report              0xA8  (v3 only)
```

### Error Codes (error-status)

```
0  noError
1  tooBig             — response would exceed maxSize / MTU
2  noSuchName         — v1: OID not present (v2c+ uses noSuchObject in varbind)
3  badValue           — v1: SET type/length wrong (v2c+ has wrongType etc.)
4  readOnly           — v1: SET on RO object
5  genErr             — catch-all
6  noAccess           — v2c+: VACM denies
7  wrongType
8  wrongLength
9  wrongEncoding
10 wrongValue
11 noCreation         — RowStatus createAndGo on non-existent table refused
12 inconsistentValue
13 resourceUnavailable
14 commitFailed
15 undoFailed
16 authorizationError — v3: USM/VACM rejected
17 notWritable
18 inconsistentName
```

### v3 Message Format

```
+-----------------------------------------------------------+
| SEQUENCE                                                  |
|   INTEGER  msgVersion   (= 3)                             |
|   SEQUENCE msgGlobalData                                  |
|     INTEGER     msgID                                     |
|     INTEGER     msgMaxSize        (>= 484)                |
|     OCTET STR   msgFlags  (bit0=auth, bit1=priv, bit2=rep)|
|     INTEGER     msgSecurityModel  (3 = USM)               |
|   OCTET STR  msgSecurityParameters  (USMSecurityParameters|
|              BER-encoded inside)                          |
|       msgAuthoritativeEngineID                            |
|       msgAuthoritativeEngineBoots                         |
|       msgAuthoritativeEngineTime                          |
|       msgUserName                                         |
|       msgAuthenticationParameters  (12-byte HMAC truncate)|
|       msgPrivacyParameters         (DES IV / AES IV)     |
|   ScopedPDU (encrypted blob if priv else cleartext)       |
|     OCTET STR   contextEngineID                           |
|     OCTET STR   contextName                               |
|     PDU         (same as v2c PDU above)                   |
+-----------------------------------------------------------+
```

## Polling Patterns

```
GET       fetch single OID exactly. Errors if OID not present (v1) or returns
          noSuchObject/noSuchInstance varbind (v2c+).
GETNEXT   fetch the OID lexicographically *after* the one given. Used to
          enumerate (walk) — keep calling GETNEXT with the previous result's
          OID until response is endOfMibView or moves out of the requested
          subtree.
GETBULK   v2c+. Two parameters: non-repeaters (N) and max-repetitions (M).
          The first N varbinds are GETNEXT'd once; the remaining (varbinds-N)
          are GETNEXT'd up to M times. One round-trip returns up to
          (N + (V-N)*M) varbinds. Massive latency win.
WALK      not a PDU — a client pattern: repeated GETNEXT (or GETBULK) until
          you leave the subtree.
SET       write a writable OID. Atomic: all varbinds succeed or none do.
          For tables, use RowStatus createAndGo / destroy / active.
```

ASCII: GETNEXT walk vs GETBULK walk:

```
  GETNEXT walk over 1000 rows on a 50ms RTT link
  ┌──────────┐  GET  ┌──────────┐
  │ Manager  │──────▶│  Agent   │
  │          │◀──────│          │   1 round-trip per OID
  └──────────┘ Resp  └──────────┘
  Total: 1000 * 50ms = 50 s

  GETBULK with max-rep=25 on same 1000 rows
  ┌──────────┐ BULK  ┌──────────┐
  │ Manager  │──────▶│  Agent   │
  │          │◀──────│ 25 rows  │   40 round-trips
  └──────────┘ Resp  └──────────┘
  Total: 40 * 50ms = 2 s   (25× faster)
```

## Traps and Informs

### v1 Trap (legacy)

v1 traps have their own PDU shape: enterprise OID, agent-address,
generic-trap (0..6), specific-trap (vendor), timestamp, varbinds.

```
generic-trap codes
  0  coldStart
  1  warmStart
  2  linkDown
  3  linkUp
  4  authenticationFailure
  5  egpNeighborLoss
  6  enterpriseSpecific  (use specific-trap field)
```

### v2c/v3 Trap (modern)

v2c/v3 traps reuse the GetResponse-style PDU. The first two varbinds
are mandatory:

```
varbind[0] = sysUpTime.0   (TimeTicks)
varbind[1] = snmpTrapOID.0 (OID identifying the notification)
varbind[2..] = (notification-specific objects from the NOTIFICATION-TYPE)
```

Common standard trap OIDs (`1.3.6.1.6.3.1.1.5`):

```
.1  coldStart
.2  warmStart
.3  linkDown          (varbinds: ifIndex, ifAdminStatus, ifOperStatus)
.4  linkUp            (varbinds: ifIndex, ifAdminStatus, ifOperStatus)
.5  authenticationFailure
.6  egpNeighborLoss   (deprecated)
```

### TRAP vs INFORM

```
TRAP    — "fire and forget". UDP, no ACK. If the receiver is down or the
          packet is lost, the event vanishes.
INFORM  — TRAP plus expectation of a Response PDU from the receiver.
          Sender retransmits on timeout. Same wire shape as a request,
          opposite direction (agent → manager).
```

## SNMPv3 Security

### USM Levels

```
noAuthNoPriv    just a username — no integrity, no confidentiality.
                Use only on isolated mgmt VRF where v3 features (engineID
                discovery, contextName) are needed but the path is trusted.
authNoPriv      HMAC-MD5-96 or HMAC-SHA-96 (truncated to 12 octets)
                over the whole message. Tampering detected; no encryption.
                Algorithms: usmHMACMD5AuthProtocol, usmHMACSHAAuthProtocol,
                usmHMAC128SHA224, usmHMAC192SHA256, usmHMAC256SHA384,
                usmHMAC384SHA512 (RFC 7860).
authPriv        auth + encryption. Privacy protocols:
                  usmDESPrivProtocol     (CBC-DES, broken — avoid)
                  usm3DESEDEPrivProtocol (3DES-CBC)
                  usmAesCfb128Protocol   (AES-128-CFB, RFC 3826) — minimum bar
                  AES-192/256 via Cisco/Blumenthal extensions (not
                  fully standardised; check vendor)
```

### Engine ID and Time

Each authoritative SNMP engine has a unique `snmpEngineID` (5..32 octets,
typically formatted per RFC 3411 §5):

```
+----------+----------+--------------------------------------------+
| 4 bytes  | 1 byte   | up to 27 bytes                              |
| ent. OID | format   | engine-data (MAC, IPv4, IPv6, text, ...)    |
| (priv-   | (1..5)   |                                             |
|  enterprise number)                                                |
+----------+----------+--------------------------------------------+
```

Replay protection uses `snmpEngineBoots` (incremented per cold-start) and
`snmpEngineTime` (seconds since last boot). Receiver enforces a 150-second
window. Skew bigger than that → `notInTimeWindow` REPORT PDU; manager
re-syncs and retries.

### USM Authentication Flow

```
   Manager (non-auth engine)              Agent (authoritative engine)
   ----------------------------           -------------------------------
   1. send GetRequest with engineID=0,
      engineBoots=0, engineTime=0
              ──────────────────────────▶
                                         2. send REPORT containing
                                            usmStatsUnknownEngineIDs.0
                                            with REAL engineID/Boots/Time
              ◀──────────────────────────
   3. recompute auth/priv keys from
      (password, engineID) using
      Password-to-Key Algorithm
      (RFC 3414 §A.2)
   4. resend GetRequest with proper
      engineID, fresh msgID, HMAC,
      and (if priv) encrypted PDU
              ──────────────────────────▶
                                         5. verify HMAC, decrypt,
                                            check VACM, fetch values,
                                            encrypt response
              ◀──────────────────────────
```

### Password-to-Key (RFC 3414)

```
Ku = MD5(password repeated until 1 048 576 octets)        # MD5 variant
Kul = MD5(Ku || engineID || Ku)                           # localised key
```

Localised keys mean knowing one device's key does not let you talk to a
sibling device with the same passphrase — each agent has a different
engineID.

### VACM (View-based Access Control)

VACM tables (`1.3.6.1.6.3.16`):

```
vacmContextTable             — list of contextNames the agent supports
vacmSecurityToGroupTable     — securityModel + securityName → groupName
vacmAccessTable              — groupName + contextPrefix + securityModel +
                                securityLevel → readView, writeView,
                                notifyView
vacmViewTreeFamilyTable      — view definitions: subtree + mask + type
                                (included | excluded)
```

Pseudo-flow on every PDU:

```
   (securityModel, securityName) → groupName
   (groupName, contextName, securityLevel) → (readView, writeView, notifyView)
   for each varbind OID:
     check OID against the relevant view tree → permit / deny
```

## net-snmp Tools (Linux)

### Installation

```bash
# Debian / Ubuntu
apt install snmp snmpd snmp-mibs-downloader libsnmp-dev

# RHEL / Rocky / Alma
dnf install net-snmp net-snmp-utils net-snmp-libs

# Enable bundled MIBs (Debian disables them by default for licence reasons)
sed -i 's/^mibs :/# mibs :/' /etc/snmp/snmp.conf
download-mibs
```

### snmpwalk — Traverse a Subtree

```bash
snmpwalk -v2c -c public 192.0.2.1 system
```

Expected output:

```
SNMPv2-MIB::sysDescr.0 = STRING: Linux router 6.6.32 #1 SMP x86_64
SNMPv2-MIB::sysObjectID.0 = OID: NET-SNMP-MIB::netSnmpAgentOIDs.10
DISMAN-EVENT-MIB::sysUpTimeInstance = Timeticks: (1234567) 3:25:45.67
SNMPv2-MIB::sysContact.0 = STRING: noc@example.com
SNMPv2-MIB::sysName.0 = STRING: edge1
SNMPv2-MIB::sysLocation.0 = STRING: Rack 4, DC-East
SNMPv2-MIB::sysServices.0 = INTEGER: 72
SNMPv2-MIB::sysORLastChange.0 = Timeticks: (3) 0:00:00.03
```

Useful flags:

```
-v 1|2c|3            protocol version
-c COMMUNITY         community (v1/v2c)
-r RETRIES           default 5
-t TIMEOUT           default 1.0 s; bump for slow CPE
-Cc                  do NOT check that responses stay in subtree (rarely needed)
-Cr N                max-repetitions for GETBULK   (snmpbulkwalk)
-CB                  use only GETBULK (no GETNEXT fallback)
-On                  numeric OIDs (no MIB-name translation)
-Of                  full OID names (.iso.org.dod.internet...)
-Os                  short names (last node only)
-OU                  no units suffix
-OQ                  no type prefix
-Oq                  whitespace-only separator (oid value)
-Ov                  values only
-Oe                  enums as numbers, not labels
-OX                  use [index] for table indices
-Cf                  do not fix counter wraps in delta tools
-d                   dump every PDU to stderr (BER hex)
-Dall                debug *everything*
-Dusm,asn1           debug specific subsystems
```

### snmpget — Single OID

```bash
snmpget -v2c -c public 192.0.2.1 sysDescr.0 sysUpTime.0
```

```
SNMPv2-MIB::sysDescr.0 = STRING: Linux router 6.6.32 #1 SMP x86_64
DISMAN-EVENT-MIB::sysUpTimeInstance = Timeticks: (1234567) 3:25:45.67
```

### snmpgetnext

```bash
snmpgetnext -v2c -c public 192.0.2.1 ifDescr
# returns ifDescr.<smallest ifIndex>
```

### snmpbulkget / snmpbulkwalk

```bash
# 0 non-repeaters, 25 max-reps, walk ifTable
snmpbulkwalk -v2c -c public -Cn0 -Cr25 192.0.2.1 ifTable

# Compare: traditional walk
time snmpwalk     -v2c -c public 192.0.2.1 ifTable >/dev/null
# real    0m12.430s
time snmpbulkwalk -v2c -c public 192.0.2.1 ifTable >/dev/null
# real    0m0.620s
```

### snmptable — Render Tabular Data

```bash
snmptable -v2c -c public -Cb -Cw 120 192.0.2.1 ifTable
```

```
SNMP table: IF-MIB::ifTable

 ifIndex  ifDescr  ifType  ifMtu  ifSpeed     ifAdminStatus  ifOperStatus
 1        lo       softwareLoopback 65536  10000000  up      up
 2        eth0     ethernetCsmacd   1500   1000000000 up     up
 3        eth1     ethernetCsmacd   1500   1000000000 up     down
 4        wg0      tunnel           1420   0          up     up
```

### snmpset — Write

```bash
# Type letters: i=Integer, u=Unsigned, t=TimeTicks, a=IpAddress,
#               o=OID, s=String, x=hex, d=decimal-string, n=Null,
#               b=Bits, U=Counter64
snmpset -v2c -c private 192.0.2.1 sysContact.0 s "noc@example.com"
snmpset -v2c -c private 192.0.2.1 ifAdminStatus.3 i 2   # admin down ifIndex 3
snmpset -v2c -c private 192.0.2.1 ifAlias.3 s "downlink-rack-7"
```

Expected on success:

```
SNMPv2-MIB::sysContact.0 = STRING: noc@example.com
```

On failure:

```
Error in packet.
Reason: notWritable (That object does not support modification)
Failed object: SNMPv2-MIB::sysContact.0
```

### snmptrap / snmpinform

```bash
# v2c trap — args: receiver, uptime ('' = use sysUpTime.0), trap-OID, varbinds
snmptrap -v2c -c public 10.0.0.5 '' \
  IF-MIB::linkDown \
  ifIndex.3 i 3 \
  ifAdminStatus.3 i 2 \
  ifOperStatus.3 i 2

# v3 inform with authPriv
snmpinform -v3 -l authPriv -u trapUser \
  -a SHA -A 'AuthPass!' -x AES -X 'PrivPass!' \
  10.0.0.5 '' coldStart
```

### snmpdelta — Sample Counters at Intervals

```bash
snmpdelta -v2c -c public -Cs 5 -CT -Cl 192.0.2.1 \
  IF-MIB::ifHCInOctets.2 IF-MIB::ifHCOutOctets.2
```

```
2026-04-27 09:12:00 /5 sec: ifHCInOctets.2     /sec: 5840232
2026-04-27 09:12:05 /5 sec: ifHCInOctets.2     /sec: 5901288
2026-04-27 09:12:00 /5 sec: ifHCOutOctets.2    /sec: 4112004
2026-04-27 09:12:05 /5 sec: ifHCOutOctets.2    /sec: 4198120
```

`-Cs N` sample interval, `-CT` rate per second, `-Cl` log-style output.

### snmpdf — Disk Usage via HOST-RESOURCES-MIB

```bash
snmpdf -v2c -c public 192.0.2.1
```

```
Description     size (kB)   Used    Available  Used%
Memory Buffers  4078432     1238440 2839992    30%
Real Memory     8157024     5621400 2535624    68%
Swap Space      4194300     0       4194300    0%
/               104857600   42301456 62556144  40%
/home           524288000   311482008 212805992 59%
```

### snmpnetstat — Netstat-style via TCP/UDP/IP-MIB

```bash
snmpnetstat -v2c -c public -CP tcp 192.0.2.1
snmpnetstat -v2c -c public -Cr 192.0.2.1   # routing table
snmpnetstat -v2c -c public -Ci 192.0.2.1   # interface table
```

### snmpstatus — One-line health

```bash
snmpstatus -v2c -c public 192.0.2.1
```

```
[UDP: [192.0.2.1]:161->[0.0.0.0]:0]=>[Linux router 6.6.32 #1 SMP x86_64]  Up: 3:25:45.67
Interfaces: 4, Recv/Trans packets: 102338472/89102374 | IP: 102338472, 0
```

### snmptranslate — OID/Name Conversion

```bash
snmptranslate -On  IF-MIB::ifInOctets       # → .1.3.6.1.2.1.2.2.1.10
snmptranslate -Of  .1.3.6.1.2.1.1.3.0       # → .iso.org.dod.internet.mgmt.mib-2.system.sysUpTime.0
snmptranslate -Td  IF-MIB::ifInOctets       # OBJECT-TYPE definition

# Tree under a subtree
snmptranslate -Tp -OS  IF-MIB::interfaces

# Find OID by name fragment
snmptranslate -To | grep -i ifhc
```

### snmpd — Run an Agent

```bash
snmpd -f -Lo                 # foreground, log to stdout
snmpd -f -Lf /var/log/snmpd.log
snmpd -DALL -f -Lo           # full debug

# Test config without restart
snmpd -t -C -c /etc/snmp/snmpd.conf
```

### snmptrapd — Receive Traps

```bash
snmptrapd -f -Lo -F "%y-%m-%d %T  %B [%a]  %V%n" -m ALL
```

```
2026-04-27 09:14:22  edge1.example.com [192.0.2.1]
DISMAN-EVENT-MIB::sysUpTimeInstance = Timeticks: (1234567) 3:25:45.67
SNMPv2-MIB::snmpTrapOID.0 = OID: IF-MIB::linkDown
IF-MIB::ifIndex.3 = INTEGER: 3
IF-MIB::ifAdminStatus.3 = INTEGER: down(2)
IF-MIB::ifOperStatus.3 = INTEGER: down(2)
```

### MIB Tools

```bash
mib2c   -c mib2c.scalar.conf  EXAMPLE-MIB::exampleObject  # generate net-snmp C scaffold
mib2c   -c mib2c.iterate.conf EXAMPLE-MIB::exampleTable   # for tables
smistrip example.txt > EXAMPLE-MIB.mib                    # extract module from RFC text
smilint -l 6 EXAMPLE-MIB.mib                              # validate (level 6 = pedantic)
smiquery oid IF-MIB::ifTable
smidiff EXAMPLE-MIB-1.mib EXAMPLE-MIB-2.mib               # diff two revisions
```

## /etc/snmp/snmpd.conf — Annotated

```conf
# /etc/snmp/snmpd.conf
###############################################################################
# AGENT ADDRESSES
###############################################################################
# Listen on UDP/161 IPv4+IPv6 on all interfaces. Add :PORT to override.
agentaddress udp:0.0.0.0:161,udp6:[::]:161

###############################################################################
# v1/v2c COMMUNITIES (legacy — avoid in production)
###############################################################################
# rocommunity NAME [SOURCE [OID [VIEW]]]
rocommunity public  10.0.0.0/8       -V systemview
rwcommunity private 10.0.0.5/32      -V allview
# IPv6 equivalents:
rocommunity6 public 2001:db8:1::/64

###############################################################################
# v3 USERS
###############################################################################
# Either (a) edit while snmpd is stopped:
createUser monitorUser SHA "AuthPass1234!" AES "PrivPass1234!"
# or (b) use net-snmp-config / net-snmp-create-v3-user (does the same thing)

# Grant 'monitorUser' read-only access to a view, with required priv:
rouser monitorUser priv systemview
rwuser adminUser   priv allview

###############################################################################
# VIEWS
###############################################################################
# view NAME { included | excluded } SUBTREE [MASK]
view systemview included .1.3.6.1.2.1.1
view systemview included .1.3.6.1.2.1.25.1
view ifview     included .1.3.6.1.2.1.2
view ifview     included .1.3.6.1.2.1.31
view allview    included .1
view allview    excluded .1.3.6.1.6.3.18    # hide notification config

###############################################################################
# v3 ENGINE ID — pin so keys survive reboots / IP changes
###############################################################################
engineID 0x80001f8880abcdef0123456789

###############################################################################
# SYSTEM INFO
###############################################################################
sysLocation    "Rack 4, DC-East"
sysContact     "noc@example.com"
sysServices    72
sysName        edge1.example.com

###############################################################################
# TRAP DESTINATIONS
###############################################################################
# v2c
trap2sink   10.0.0.5 public
informsink  10.0.0.5 public

# v3
trapsess -v3 -u trapUser -a SHA -A 'AuthPass1234!' \
              -x AES -X 'PrivPass1234!' -l authPriv 10.0.0.5

# Built-in trap toggles
authtrapenable 1
linkUpDownNotifications yes

###############################################################################
# DISMAN-EVENT (agent-side polling / threshold)
###############################################################################
monitor -r 60 -o ifDescr -o ifOperStatus "ifOper down" ifOperStatus != 1
defaultMonitors yes
linkUpDownNotifications yes

###############################################################################
# EXTEND — expose shell scripts as OIDs under NET-SNMP-EXTEND-MIB
###############################################################################
extend disk-free      /usr/local/bin/df-mb /
extend num-ssh-conns  /usr/bin/sh -c "ss -H -t state established '( dport = :22 )' | wc -l"

###############################################################################
# PASS / PASS-PERSIST — implement custom OIDs in any language
###############################################################################
pass        .1.3.6.1.4.1.99999.1 /usr/local/bin/my-mib-handler
pass_persist .1.3.6.1.4.1.99999.2 /usr/local/bin/my-persistent-handler.py
```

The legacy `com2sec / group / access` triplet still works:

```conf
#         secName    source            community
com2sec  notConfigUser  default        public

#       groupName       secModel  secName
group   notConfigGroup  v1        notConfigUser
group   notConfigGroup  v2c       notConfigUser

#       group           context sec.model sec.level prefix read     write notify
access  notConfigGroup  ""      any       noauth    exact  systemview none none
```

## /etc/snmp/snmptrapd.conf

```conf
authCommunity    log,execute,net public
authUser         log,execute,net trapUser   priv
disableAuthorization no

format2 %y-%m-%d %T %B %A %V

# Forward traps to a script
traphandle default /usr/local/bin/trap-to-syslog.sh

# Match a specific trap and run a different handler
traphandle IF-MIB::linkDown /usr/local/bin/linkdown-pager.sh

# Drop chatty traps you don't care about
ignoreAuthFailure yes
```

## Cisco IOS / IOS-XE SNMP

```
! v2c
snmp-server community RO_PUB    RO  10
snmp-server community RW_SECRET RW  11

! Restrict source IPs
ip access-list standard 10
 permit 10.0.0.0 0.255.255.255
ip access-list standard 11
 permit host 10.0.0.5

snmp-server location "Rack 4, DC-East"
snmp-server contact  "noc@example.com"
snmp-server chassis-id ABC1234XYZ
snmp-server ifindex persist                    ! keep ifIndex stable across reload

! Limit OIDs exposed
snmp-server view CUTDOWN iso included
snmp-server view CUTDOWN snmpUsmMIB excluded
snmp-server view CUTDOWN snmpVacmMIB excluded
snmp-server view CUTDOWN snmpCommunityMIB excluded

snmp-server group RO-GROUP v3 priv read CUTDOWN access 10
snmp-server user monUser RO-GROUP v3 auth sha 'AuthPass1234!' priv aes 128 'PrivPass1234!'

! Trap host(s)
snmp-server enable traps
snmp-server enable traps bgp
snmp-server enable traps ospf
snmp-server enable traps cpu threshold
snmp-server enable traps memory
snmp-server enable traps entity
snmp-server enable traps syslog
snmp-server host 10.0.0.5 version 2c TRAP_PUB
snmp-server host 10.0.0.5 version 3 priv monUser

! Engine ID
snmp-server engineID local 80000009030011223344AABB

! Source-interface for replies (helps ACLs at upstream firewalls)
snmp-server source-interface informs Loopback0
snmp-server trap-source Loopback0

! Tune (CPU protection)
snmp-server packetsize 1500
snmp-server queue-length 30
snmp-server tftp-server-list 10
snmp-server file-transfer access-group 10 protocol ftp
```

Verification:

```
edge1#show snmp
Chassis: ABC1234XYZ
Contact: noc@example.com
Location: Rack 4, DC-East
0 SNMP packets input
    0 Bad SNMP version errors
    0 Unknown community name
    0 Illegal operation for community name supplied
    0 Encoding errors
    0 Number of requested variables
    0 Number of altered variables
    0 Get-request PDUs
    0 Get-next PDUs
    0 Set-request PDUs
    0 Input queue packet drops (Maximum queue size 1000)

edge1#show snmp user
User name: monUser
Engine ID: 80000009030011223344AABB
storage-type: nonvolatile        active
Authentication Protocol: SHA
Privacy Protocol: AES128
Group-name: RO-GROUP

edge1#show snmp host
Notification host: 10.0.0.5  udp-port: 162   type: trap
user: monUser     security model: v3 priv

edge1#show snmp engineID
Local SNMP engineID:    80000009030011223344AABB
Remote Engine ID            IP-addr            Port
800000090300AABBCCDDEEFF    10.0.0.5           162
```

## Cisco NX-OS

```
snmp-server community RO_PUB group network-operator
snmp-server user monUser network-operator auth sha 'AuthPass1234!' priv aes-128 'PrivPass1234!'
snmp-server host 10.0.0.5 traps version 3 priv monUser
snmp-server enable traps
snmp-server enable traps bgp
snmp-server enable traps ospf
snmp-server enable traps entity
snmp-server source-interface trap mgmt0
```

## Juniper Junos

```
set snmp location "Rack 4, DC-East"
set snmp contact  "noc@example.com"
set snmp engine-id local 0011223344aabb

# v2c
set snmp community public authorization read-only
set snmp community public clients 10.0.0.0/8

# v3
set snmp v3 usm local-engine user monUser authentication-sha authentication-password 'AuthPass1234!'
set snmp v3 usm local-engine user monUser privacy-aes128       privacy-password       'PrivPass1234!'
set snmp v3 vacm security-to-group security-model usm security-name monUser group RO-GROUP
set snmp v3 vacm access group RO-GROUP default-context-prefix security-model usm security-level privacy read-view CUTDOWN
set snmp view CUTDOWN oid .1 include
set snmp view CUTDOWN oid jnxJsHostMIB exclude

# Trap targets and groups
set snmp trap-group NMS targets 10.0.0.5
set snmp trap-group NMS version v2
set snmp trap-options source-address lo0.0
set snmp v3 target-address NMS-V3 address 10.0.0.5 target-parameters NMS-V3-PARAMS
set snmp v3 target-parameters NMS-V3-PARAMS parameters message-processing-model v3
set snmp v3 target-parameters NMS-V3-PARAMS parameters security-model usm
set snmp v3 target-parameters NMS-V3-PARAMS parameters security-level privacy
set snmp v3 target-parameters NMS-V3-PARAMS parameters security-name monUser

# Per-event toggles
set snmp trap-group NMS categories link
set snmp trap-group NMS categories routing
set snmp trap-group NMS categories chassis
set snmp trap-group NMS categories authentication
set snmp trap-group NMS categories services
```

Verification:

```
re0> show snmp statistics
SNMP statistics:
  Input:
    Packets: 12345, Bad versions: 0, Bad community names: 0,
    Bad community uses: 0, ASN parse errors: 0,
    Too bigs: 0, No such names: 0, Bad values: 0, Read onlys: 0, ...

re0> show snmp v3
Local engine ID: 0011223344aabb
Engine boots: 4
Engine time: 8129 seconds
Max message size: 65507 bytes

USM users:
   user "monUser",  group "RO-GROUP", auth SHA, priv AES128

Target addresses:
   NMS-V3   address 10.0.0.5/162   timeout 1500   retries 3
```

## Arista EOS

```
snmp-server location "Rack 4, DC-East"
snmp-server contact  "noc@example.com"
snmp-server community RO_PUB ro 10
snmp-server view CUTDOWN iso included
snmp-server view CUTDOWN snmpVacmMIB excluded
snmp-server group RO-GROUP v3 priv read CUTDOWN
snmp-server user monUser RO-GROUP v3 auth sha 'AuthPass1234!' priv aes 'PrivPass1234!'
snmp-server host 10.0.0.5 version 3 priv monUser
snmp-server enable traps
snmp-server vrf MGMT
```

Verification:

```
edge1#show snmp user
User name        : monUser
Authentication   : SHA
Privacy          : AES-128
Group            : RO-GROUP
EngineID         : 80000009030011223344AABB
remote           : false

edge1#show snmp counters
Packets received       : 12345
Packets sent           : 12340
Bad community names    : 0
Bad community use      : 0
Authentication failures: 0
ASN parse errors       : 0
Bad versions           : 0
```

## FRRouting / Quagga (AgentX subagent)

FRR exposes its own MIBs (BGP4-MIB, OSPF-MIB, ISIS-MIB) over AgentX
through a master snmpd:

```conf
# /etc/snmp/snmpd.conf
master agentx
agentXSocket  /var/agentx/master
agentXTimeout 60
```

```
# /etc/frr/daemons
bgpd_options="   -A 127.0.0.1 -M snmp"
ospfd_options="  -A 127.0.0.1 -M snmp"
isisd_options="  -A 127.0.0.1 -M snmp"
```

Then walk on the host:

```bash
snmpwalk -v2c -c public localhost 1.3.6.1.2.1.15.3   # bgpPeerEntry
```

## Common MIBs

```
SNMPv2-MIB              1.3.6.1.2.1.1     RFC 3418  system, snmp
IF-MIB                  1.3.6.1.2.1.2     RFC 2863  interfaces (32-bit)
IF-MIB ifXTable         1.3.6.1.2.1.31    RFC 2863  HC counters, ifName, ifAlias
IP-MIB                  1.3.6.1.2.1.4     RFC 4293  ip, ipv4 + ipv6 unified
TCP-MIB                 1.3.6.1.2.1.6     RFC 4022
UDP-MIB                 1.3.6.1.2.1.7     RFC 4113
HOST-RESOURCES-MIB      1.3.6.1.2.1.25    RFC 2790  CPU, memory, disk, processes
ENTITY-MIB              1.3.6.1.2.1.47    RFC 6933  hardware inventory
ENTITY-SENSOR-MIB       1.3.6.1.2.1.99    RFC 3433  temp, voltage, fan RPM
BRIDGE-MIB              1.3.6.1.2.1.17    RFC 4188  STP, L2 forwarding
Q-BRIDGE-MIB            1.3.6.1.2.1.17.7  RFC 4363  VLAN
LLDP-MIB                1.0.8802.1.1.2    IEEE 802.1AB
RMON-MIB                1.3.6.1.2.1.16    RFC 2819
RMON2-MIB               1.3.6.1.2.1.16.20 RFC 4502
DISMAN-EVENT-MIB        1.3.6.1.2.1.88    RFC 2981  agent-side threshold checks
DISMAN-PING-MIB         1.3.6.1.2.1.80    RFC 4560  remote ping
DISMAN-TRACEROUTE-MIB   1.3.6.1.2.1.81    RFC 4560
DISMAN-NSLOOKUP-MIB     1.3.6.1.2.1.82    RFC 4560
NOTIFICATION-LOG-MIB    1.3.6.1.2.1.92    RFC 3014
SNMP-FRAMEWORK-MIB      1.3.6.1.6.3.10    RFC 3411
SNMP-USER-BASED-SM-MIB  1.3.6.1.6.3.15    RFC 3414
SNMP-VIEW-BASED-ACM-MIB 1.3.6.1.6.3.16    RFC 3415
SNMP-COMMUNITY-MIB      1.3.6.1.6.3.18    RFC 3584
SNMPv2-TM               1.3.6.1.6.1.1     RFC 3417  transport mappings
NET-SNMP-EXTEND-MIB     1.3.6.1.4.1.8072.1.3.2  net-snmp 'extend' OIDs
NET-SNMP-AGENT-MIB      1.3.6.1.4.1.8072.1.5    net-snmp internals
HOST-RESOURCES-TYPES    1.3.6.1.2.1.25.7
BGP4-MIB                1.3.6.1.2.1.15    RFC 4273
OSPF-MIB                1.3.6.1.2.1.14    RFC 4750
OSPFv3-MIB              1.3.6.1.2.1.191   RFC 5643
ISIS-MIB                1.3.6.1.2.1.138   RFC 4444
MPLS-LSR-STD-MIB        1.3.6.1.2.1.166   RFC 3813
MPLS-TE-STD-MIB         1.3.6.1.2.1.10.166 RFC 3812
RADIUS-AUTH-CLIENT-MIB  1.3.6.1.2.1.67.1.2 RFC 4668

# Major vendor enterprise prefixes
Cisco          1.3.6.1.4.1.9
  CISCO-PROCESS-MIB     .9.9.109   CPU, busybusy
  CISCO-MEMORY-POOL-MIB .9.9.48    pool memory
  CISCO-ENHANCED-MEMPOOL-MIB .9.9.221  newer
  CISCO-ENVMON-MIB      .9.9.13    temp, fan, PSU
  CISCO-ENTITY-SENSOR-MIB .9.9.91  optical / DOM
  CISCO-LWAPP-AP-MIB    .9.9.513   wireless AP
  CISCO-FLASH-MIB       .9.9.10
  CISCO-CONFIG-COPY-MIB .9.9.96    "copy run start" via SNMP
Juniper        1.3.6.1.4.1.2636
  JUNIPER-MIB                      jnxBoxAnatomy (chassis)
  JUNIPER-OPERATING-MIB            CPU, memory, temperature
  JUNIPER-IF-MIB                   per-LU stats, queues
  JUNIPER-COS-MIB                  CoS queue depths
  JUNIPER-FIREWALL-MIB             firewall counters
Arista         1.3.6.1.4.1.30065
  ARISTA-EOS-MIB
  ARISTA-ENTITY-SENSOR-MIB
HP / Aruba     1.3.6.1.4.1.11   (HP),  1.3.6.1.4.1.14823 (Aruba)
F5             1.3.6.1.4.1.3375
Mikrotik       1.3.6.1.4.1.14988
VMware         1.3.6.1.4.1.6876
Linux netSnmp  1.3.6.1.4.1.8072
```

## Common Errors and Diagnostics

### Manager-Side

```
Timeout: No Response from <host>
    cause: agent down, ACL blocks 161, wrong community/version, firewall
    fix:   tcpdump -ni any 'udp port 161 and host 192.0.2.1'
           verify community via 'snmpget -v2c -c public host sysDescr.0'
           bump retries: -r 5 -t 3

snmpwalk: Unknown host (badhost)
    cause: DNS or hosts file
    fix:   getent hosts badhost; or use IP literal

snmpwalk: Unknown user name (USER NAME)
    cause: v3 user not in agent USM table or engineID mismatch after rekey
    fix:   re-run net-snmp-create-v3-user; check engineID matches

snmpwalk: Authentication failure (incorrect password, community or key)
    cause: wrong v3 auth password or wrong community
    fix:   verify '-A' / '-c'; remember v3 keys are localised to engineID

snmpwalk: Decryption error
    cause: wrong v3 priv password, or sender used unsupported priv algo
    fix:   match '-x' alg and '-X' password to agent config

snmpwalk: USM authentication failure: not in time window
    cause: clock skew between manager and agent > 150 s OR engineBoots stale
    fix:   chronyc sources;  on agent: 'show clock' / 'show snmp engineID'
           manager will auto-resync after one REPORT exchange — investigate
           if it persists

End of MIB                          (snmpwalk just exits — normal)
No Such Object available on this agent at this OID
    cause: OID not implemented (v2c+)
    fix:   snmpwalk a parent subtree to confirm; check vendor MIB version

No Such Instance currently exists at this OID
    cause: column exists, row index doesn't (e.g. ifIndex deleted)
    fix:   walk the table to see current indices

Reason: noSuchName (Bad object name)
    cause: v1 only — same root cause as noSuchObject

Reason: notWritable (That object does not support modification)
    cause: SET on read-only (or vendor refused even for read-write)
    fix:   check MAX-ACCESS in the MIB; check VACM writeView

Reason: wrongType / wrongLength / wrongValue / wrongEncoding
    cause: snmpset passed wrong syntax for that OID
    fix:   smiquery / snmptranslate -Td OID — read the SYNTAX

Reason: authorizationError (access denied to that object)
    cause: VACM excludes it from your view
    fix:   adjust 'view' / 'rouser' / 'access' on the agent

Reason: tooBig (Response message would have been too large)
    cause: huge GETBULK response did not fit within msgMaxSize
    fix:   lower max-repetitions: -Cr10
```

### Agent-Side / Daemon

```
snmpd: Error opening specified endpoint "udp:161"
    cause: another process already bound (often older snmpd, or agentx
           socket conflicting)
    fix:   ss -ulnp | grep 161 ; systemctl stop snmpd; recheck
           selinux: setenforce 0 (test) or proper policy

snmpd: error parsing config: line 24: bad VIEW name
    fix:   snmpd -t -C -c /etc/snmp/snmpd.conf  to validate without start

snmpd: Cannot find module (FOO-MIB): At line 12 in /etc/snmp/snmp.conf
    cause: MIB not in MIBDIRS path
    fix:   apt install snmp-mibs-downloader; download-mibs;
           or add MIBDIRS=+/path  in /etc/snmp/snmp.conf

snmpd: AgentX: Invalid socket path /var/agentx/master
    cause: SELinux / AppArmor or the dir not present
    fix:   mkdir -p /var/agentx; chmod 755 /var/agentx; chown root:snmp /var/agentx

snmpd: send response: Failure in sendto
    cause: source IP unreachable (asymmetric routing in mgmt VRF)
    fix:   set 'agentaddress udp:<MGMT_IP>:161' explicitly
```

### Wire-Level Diagnostics

```
# Capture both directions, decode SNMP
tcpdump -ni any -s 0 -vv 'udp and (port 161 or port 162)'

# Decode with tshark — shows OIDs, error-status, varbinds
tshark -ni any -O snmp -V -f 'udp port 161'

# Force IPv6
snmpwalk -v2c -c public udp6:[2001:db8::1]:161 system

# Use TCP transport (rare; needs agent support)
snmpwalk -v2c -c public tcp:192.0.2.1:1161 system

# Force a specific source address
snmpwalk -v2c -c public --clientaddr=10.0.0.99 192.0.2.1 system
```

## Sample Workflows

### List Interfaces with Names and Operational State

```bash
paste \
  <(snmpwalk -v2c -c public -Oqv 192.0.2.1 IF-MIB::ifName) \
  <(snmpwalk -v2c -c public -Oqv 192.0.2.1 IF-MIB::ifAlias) \
  <(snmpwalk -v2c -c public -Oqv 192.0.2.1 IF-MIB::ifOperStatus) | column -t
```

```
Gi0/0/1   uplink-core1     up
Gi0/0/2   uplink-core2     up
Gi0/0/3   downlink-rack-7  down
Gi0/0/4                    up
Lo0       loopback         up
```

### Map ifIndex → ifName (stable lookups)

```bash
snmpwalk -v2c -c public -Os 192.0.2.1 IF-MIB::ifName | \
  awk -F'[ .=]' '/ifName/ {print $2"\t"$NF}' | sort -n
```

### 64-bit Throughput in Mbps Over a 60s Window

```bash
HOST=192.0.2.1; CMTY=public; IDX=2

read IN1 OUT1 < <(snmpget -v2c -c $CMTY -Oqv $HOST \
   IF-MIB::ifHCInOctets.$IDX IF-MIB::ifHCOutOctets.$IDX)
sleep 60
read IN2 OUT2 < <(snmpget -v2c -c $CMTY -Oqv $HOST \
   IF-MIB::ifHCInOctets.$IDX IF-MIB::ifHCOutOctets.$IDX)

bc -l <<EOF
in_mbps  = ((${IN2}  - ${IN1}) * 8 / 60) / 1000000
out_mbps = ((${OUT2} - ${OUT1}) * 8 / 60) / 1000000
print "in: ", in_mbps, " Mbps  out: ", out_mbps, " Mbps\n"
EOF
```

### CPU Load on Every Core

```bash
snmpwalk -v2c -c public -Oqv 192.0.2.1 HOST-RESOURCES-MIB::hrProcessorLoad
```

```
12
8
4
99
3
```

### BGP Peer Health

```bash
snmpwalk -v2c -c public 192.0.2.1 BGP4-MIB::bgpPeerState | \
  awk -F'[ .=]' '/bgpPeerState/ {ip=$2"."$3"."$4"."$5; print ip"\t"$NF}'
```

```
192.0.2.10      6     # established
192.0.2.11      6
198.51.100.4    3     # active (down)
```

State legend: 1=idle 2=connect 3=active 4=opensent 5=openconfirm 6=established.

### Inventory Chassis via ENTITY-MIB

```bash
snmpwalk -v2c -c public -Os 192.0.2.1 ENTITY-MIB::entPhysicalDescr | \
  paste - <(snmpwalk -v2c -c public -Os -Oqv 192.0.2.1 ENTITY-MIB::entPhysicalSerialNum)
```

### Cisco Optical DOM (Tx/Rx Power)

```bash
# CISCO-ENTITY-SENSOR-MIB::entSensorValue, sensorType=14 (dBm)
snmpwalk -v2c -c public 192.0.2.1 1.3.6.1.4.1.9.9.91.1.1.1.1.4 | \
  awk -F'[ .=]' '/entSensorValue/ {print $2,$NF/100" dBm"}'
```

### Trigger a Cisco config-copy (running → startup) over SNMP

```bash
H=192.0.2.1; C=private; ROW=$$
# 1. createAndWait row
snmpset -v2c -c $C $H ccCopyProtocol.$ROW       i 4   \
                         ccCopySourceFileType.$ROW i 4 \
                         ccCopyDestFileType.$ROW   i 3 \
                         ccCopyEntryRowStatus.$ROW i 5
# 2. activate
snmpset -v2c -c $C $H ccCopyEntryRowStatus.$ROW i 1
# 3. poll completion
snmpwalk -v2c -c $C $H ccCopyState.$ROW
# 4. clean up
snmpset -v2c -c $C $H ccCopyEntryRowStatus.$ROW i 6
```

### Walk a Whole Fleet in Parallel

```bash
parallel -j 32 --tag '
  snmpget -v2c -c public -t2 -r1 {} sysName.0 sysDescr.0 sysUpTime.0 \
    || echo "DOWN: {}"
' :::: hosts.txt
```

```
edge01.example.com    SNMPv2-MIB::sysName.0 = STRING: edge01
edge01.example.com    SNMPv2-MIB::sysDescr.0 = STRING: Cisco IOS XE 17.9.4
edge02.example.com    SNMPv2-MIB::sysName.0 = STRING: edge02
core03.example.com    DOWN: core03.example.com
```

### snmpd-side Threshold Trap on CPU

```conf
# Send a trap when 1-min CPU > 90% for 60s
monitor -r 30 -e cpuHigh -o hrProcessorLoad.196608 \
   "cpuHigh: cpu busy" hrProcessorLoad.196608 > 90

# Define the notification (or rely on DISMAN-EVENT-MIB::mteTriggerFired)
```

## Polling Strategies

### Why GETBULK Beats Repeated GETNEXT

Latency dominates. Each GETNEXT is 1 PDU each way (`2 * RTT` worst case
per object — usually `RTT + agent-CPU`). GETBULK packs N objects into
one response. On a 50ms RTT link with 1000 objects:

```
GETNEXT walk:   1000 RTTs ≈ 50 s
GETBULK 25:       40 RTTs ≈ 2  s
GETBULK 50:       20 RTTs ≈ 1  s
GETBULK 100:      10 RTTs ≈ 0.5 s   (often hits tooBig — back off)
```

Pick max-repetitions so that response stays under 1500 bytes (single
fragment) on lossy paths, or 9000 on jumbo intra-DC.

### Timeouts and Retries

```
snmp[walk|get|...] flags
   -t TIMEOUT   per-attempt timeout (seconds; default 1)
   -r RETRIES   default 5 — total budget = (RETRIES+1) * TIMEOUT
```

Typical settings:

```
LAN-local fast box:    -t 0.5  -r 1     (avoid pile-on)
Slow CPE / overloaded: -t 5    -r 2
WAN / satellite:       -t 10   -r 3
```

### Agent CPU Protection

SNMP runs on the **control plane** of routers and switches. A naive
walk of large tables (mac-address-table on a 384-port leaf, BGP RIB on a
full-table edge) will spike CPU and may starve routing protocols
(OSPF/BGP keepalive loss → flap).

Mitigations:

```
Cisco IOS:  snmp-server packetsize 1500
            snmp-server queue-length 30
            snmp-server file-transfer access-group <ACL>
            ip access-list extended SNMP-ACL
              permit udp host 10.0.0.5 any eq snmp
            cpu threshold type total rising 80 interval 5

Junos:      set snmp interface mgmt-only        (lock to mgmt iface)
            set system processes routing snmp-management-fast-poll-interval 60

snmpd:      sysOREnable disable                 (skip slow OBJECTS-Registered table)
            agentXSocket tcp:localhost:705      (relieves single-thread snmpd)
            cpu_limit                            (RHEL systemd unit override)
```

Stagger fleets — never poll N devices simultaneously. Spread by
`hash(hostname) % interval`.

### Counter Wrap

Counter32 wraps every:

```
   At 100 Mbps:   2^32 / (100e6 / 8)         = 343 s (~6 min)
   At 1 Gbps:     2^32 / (1e9   / 8)         = 34 s
   At 10 Gbps:    2^32 / (10e9  / 8)         = 3.4 s
   At 100 Gbps:   2^32 / (100e9 / 8)         = 343 ms  (uselessly fast)
```

Counter64:

```
   At 100 Gbps:   2^64 / (100e9 / 8)         = 46.7 years
```

Use `ifHCInOctets` / `ifHCOutOctets` (`ifXTable`) on anything ≥1 Gbps.
Polling Counter32 at 5-min intervals on a 1 Gbps link is **broken** —
you cannot tell wraps from real data.

Delta logic must handle wrap:

```python
def delta_counter(prev, curr, bits=64):
    mod = 1 << bits
    return (curr - prev) % mod    # works as long as no DOUBLE wrap occurred
```

For Counter32 detect a discontinuity by sampling
`ifCounterDiscontinuityTime` — if it changed since last poll, drop the
delta (counters were reset, e.g. SNMP module reload).

## Modern Alternatives

```
gNMI / gRPC streaming     OpenConfig + YANG schemas, push from device,
                           sub-second cadence, structured (protobuf).
                           See network-os/cisco-ios, monitoring/model-driven-telemetry.

NETCONF / RESTCONF        Configuration-oriented (YANG). Some operational
                           data; less common for high-frequency telemetry.

IPFIX / NetFlow / sFlow   Per-flow data (not per-counter).
                           sFlow is sampled and fast; NetFlow/IPFIX is
                           cache-based.

OpenTelemetry             For applications. Some infra exporters via
                           Prometheus adapter.

Prometheus snmp_exporter  Bridges SNMP into Prometheus pull model. Generated
                           per-platform from "generator.yml" against MIB files.
                           One prometheus job per device class.

Telegraf snmp input       SNMP → Influx / Elastic / Kafka. Supports v1/v2c/v3
                           with translation table.
```

When SNMP still wins:

```
Universal vendor / firmware support — every box that has a console has SNMP.
Trap-only event sources (UPS, environmentals, legacy IDS/IPS).
Power, fan, optical, transceiver inventory pre-OpenConfig.
Black-box appliances where you cannot enable streaming telemetry.
Tiny CPE where gRPC + TLS + protobuf is too heavy.
```

When to prefer streaming:

```
Sub-second cadence
Strict typing / schema discovery
Bulk-data scale (full Internet RIB, 100k flows)
TLS-encrypted, mutually authenticated transport out-of-the-box
```

## Tools and Ecosystem

```
LibreNMS, Observium, Cacti, Zabbix, Icinga, OpenNMS — open-source NMS
SolarWinds NPM, PRTG, ScienceLogic SL1, Auvik           — commercial
Prometheus snmp_exporter   github.com/prometheus/snmp_exporter
   - generator.yml → snmp.yml mapping ; runs as sidecar
Telegraf snmp + snmp_trap inputs (InfluxData)
Grafana with snmp_exporter or influxdb datasource
NAV (UNINETT), Cricket, MRTG (RRDtool ancestor of all the above)
StackStorm / Salt Beacons for trap-driven automation
```

## Worked Examples

### Example 1 — Bandwidth Graphing One Interface in Prometheus

`generator.yml`:

```yaml
modules:
  if_mib_basic:
    walk:
      - 1.3.6.1.2.1.31.1.1.1.1     # ifName
      - 1.3.6.1.2.1.31.1.1.1.6     # ifHCInOctets
      - 1.3.6.1.2.1.31.1.1.1.10    # ifHCOutOctets
      - 1.3.6.1.2.1.2.2.1.8        # ifOperStatus
    version: 2
    auth:
      community: public
    lookups:
      - source_indexes: [ifIndex]
        lookup: 1.3.6.1.2.1.31.1.1.1.1
        drop_source_indexes: false
```

Generate:

```bash
generator generate -m /usr/share/snmp/mibs -m mibs/ -g generator.yml -o snmp.yml
```

Prometheus job:

```yaml
- job_name: snmp-edges
  static_configs:
    - targets: [192.0.2.1, 192.0.2.2, 192.0.2.3]
  metrics_path: /snmp
  params:
    module: [if_mib_basic]
  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_target
    - source_labels: [__param_target]
      target_label: instance
    - target_label: __address__
      replacement: snmp-exporter:9116
```

PromQL:

```
sum by (instance, ifName) (
  rate(ifHCInOctets[5m]) * 8 / 1e6
)
```

### Example 2 — Alerting on Link-Down Trap with snmptrapd → Alertmanager

`/etc/snmp/snmptrapd.conf`:

```
authCommunity log,execute,net public
format2 %v
traphandle IF-MIB::linkDown /usr/local/bin/linkdown-alert.sh
```

`/usr/local/bin/linkdown-alert.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
read HOSTLINE
read OIDLINE
while read VARLINE; do
  case "$VARLINE" in
    *ifIndex*)       IDX=${VARLINE##* } ;;
    *ifAdminStatus*) AS=${VARLINE##* } ;;
    *ifOperStatus*)  OS=${VARLINE##* } ;;
  esac
done

curl -sS -X POST http://alertmanager:9093/api/v2/alerts \
  -H 'Content-Type: application/json' \
  -d "$(cat <<JSON
[{
  "labels": {
    "alertname": "linkDown",
    "host":      "${HOSTLINE}",
    "ifIndex":   "${IDX}",
    "severity":  "critical"
  },
  "annotations": {
    "summary": "linkDown on ${HOSTLINE} ifIndex ${IDX}",
    "ifAdminStatus": "${AS}",
    "ifOperStatus":  "${OS}"
  }
}]
JSON
)"
```

Run snmptrapd:

```
snmptrapd -f -Lo -m ALL -F '%y-%m-%d %T %B'
```

Test fire:

```bash
snmptrap -v2c -c public localhost '' IF-MIB::linkDown \
  ifIndex.3 i 3 ifAdminStatus.3 i 2 ifOperStatus.3 i 2
```

Expected log:

```
2026-04-27 09:31:20 localhost
linkDown forwarded to alertmanager (host=localhost ifIndex=3)
```

### Example 3 — Polling All-CPU on a Fleet (Cisco)

```bash
#!/usr/bin/env bash
# fleet-cpu.sh — emit hostname, 5-sec, 1-min, 5-min CPU
HOSTS=hosts.txt; CMTY=public

# 5-sec, 1-min, 5-min averages on Cisco
# CISCO-PROCESS-MIB::cpmCPUTotal5secRev .1.3.6.1.4.1.9.9.109.1.1.1.1.6
# CISCO-PROCESS-MIB::cpmCPUTotal1minRev .1.3.6.1.4.1.9.9.109.1.1.1.1.7
# CISCO-PROCESS-MIB::cpmCPUTotal5minRev .1.3.6.1.4.1.9.9.109.1.1.1.1.8
parallel -j 64 --tag '
  read S5 < <(snmpget -v2c -c '"$CMTY"' -Oqv -t2 -r1 {} \
                .1.3.6.1.4.1.9.9.109.1.1.1.1.6.1)
  read M1 < <(snmpget -v2c -c '"$CMTY"' -Oqv -t2 -r1 {} \
                .1.3.6.1.4.1.9.9.109.1.1.1.1.7.1)
  read M5 < <(snmpget -v2c -c '"$CMTY"' -Oqv -t2 -r1 {} \
                .1.3.6.1.4.1.9.9.109.1.1.1.1.8.1)
  printf "%-25s %3s %3s %3s\n" {} $S5 $M1 $M5
' :::: $HOSTS | sort -k4 -n
```

Output:

```
edge17.example.com           5  10  12
edge04.example.com           7  11  14
edge02.example.com           9  13  15
core03.example.com          78  82  85
```

### Example 4 — Comparing snmp_exporter vs Telegraf

Both bridge SNMP into a TSDB. Pick:

```
snmp_exporter             Telegraf snmp input
-----------------------   ----------------------------------------
Pull (Prometheus scrape)  Push (Telegraf → InfluxDB / Kafka / ...)
generator.yml -> snmp.yml inline TOML config
HC counter rate via       Telegraf does delta() on its end with its
PromQL rate()             own CounterCache
Tag rewrites via          Tag from index lookups, snmp.translate
relabel_configs
Memory: ~80 MB / 1k tgts  Memory: depends on inputs.snmp count + interval
Multitenant?  one job per Multitenant via [[inputs.snmp]] blocks
device class
```

`telegraf.conf` (excerpt):

```toml
[[inputs.snmp]]
  agents = ["udp://192.0.2.1:161", "udp://192.0.2.2:161"]
  version = 2
  community = "public"
  timeout   = "5s"
  retries   = 2
  agent_host_tag = "host"

  [[inputs.snmp.field]]
    name = "sysUpTime"
    oid  = "DISMAN-EVENT-MIB::sysUpTimeInstance"

  [[inputs.snmp.table]]
    name = "interface"
    inherit_tags = ["host"]
    [[inputs.snmp.table.field]]
      name = "ifName"
      oid  = "IF-MIB::ifName"
      is_tag = true
    [[inputs.snmp.table.field]]
      name = "ifHCInOctets"
      oid  = "IF-MIB::ifHCInOctets"
    [[inputs.snmp.table.field]]
      name = "ifHCOutOctets"
      oid  = "IF-MIB::ifHCOutOctets"
```

### Example 5 — Diagnose "Timeout" End-to-End

```bash
# 1. Reachable?
ping -c1 -W1 192.0.2.1                       # ICMP path
nc -zuv 192.0.2.1 161                        # UDP/161 (best-effort)

# 2. Wire capture — does the Get even reach the device? Reply lost?
sudo tcpdump -ni any -vv 'host 192.0.2.1 and udp port 161'

# 3. Auth issue?
snmpwalk -v2c -c public 192.0.2.1 sysDescr.0 -d
# -d dumps PDUs to stderr.

# 4. ACL / view limits?
# Compare a known-good sysDescr against a deeper OID — a working
# system query but a failing ifTable means VACM / view exclusion.

# 5. Agent CPU pegged? (control plane DoS)
ssh net-admin@192.0.2.1 'show processes cpu sorted | exclude 0.00'

# 6. Fragmentation? (response over PMTU on path)
snmpwalk -v2c -c public 192.0.2.1 ifTable -Cr5     # smaller bulk
```

## ASCII: Polling vs Streaming

```
   SNMP Polling                         Streaming Telemetry (gNMI)
   ────────────                         ──────────────────────────
   Manager                              Manager
     │  every 60s                         │
     │ GET ifHCInOctets                   │ Subscribe (sample, 5s)
     ├────────────────▶ Agent             │
     │                                    ├────────────────▶ Device
     │ Response (1 OID)                   │
     ◀────────────────                    │ stream: t=0,5,10,... (forever)
     │                                    ◀────────────────
     │ Repeat for next OID                │
     │ Repeat for next device             │ Push includes JSON/protobuf path:
     │ Aggregate after walk               │   /interfaces/interface[name=eth0]
                                          │       /state/counters/in-octets
   Coupling: tight (request-response)    Coupling: loose, schema-typed
   Latency:  poll-interval-bound         Latency:  near-real-time
   Encoding: BER ASN.1                   Encoding: protobuf / JSON
   Auth:     USM / community             Auth:     mTLS / OpenConfig RBAC
```

## ASCII: USM Security Negotiation

```
 ┌─────────────┐                                 ┌─────────────┐
 │  Manager    │                                 │   Agent     │
 │ (non-auth)  │                                 │ (authoritative)│
 └──────┬──────┘                                 └──────┬──────┘
        │  GET sysDescr.0                                │
        │  msgAuthEngineID  = 0x00                       │
        │  msgAuthBoots     = 0                          │
        │  msgAuthTime      = 0                          │
        │  user= "monUser"  level=authPriv               │
        ├───────────────────────────────────────────────▶│
        │                                                │
        │  REPORT  usmStatsUnknownEngineIDs.0 += 1       │
        │  msgAuthEngineID  = 80001f8880abcdef           │
        │  msgAuthBoots     = 7                          │
        │  msgAuthTime      = 12345                      │
        │◀───────────────────────────────────────────────│
        │                                                │
        │ derive Kul = MD5(Ku || engineID || Ku)         │
        │ derive priv key similarly                      │
        │                                                │
        │  GET sysDescr.0                                │
        │  msgAuthEngineID  = 80001f8880abcdef           │
        │  msgAuthBoots     = 7                          │
        │  msgAuthTime      = 12345                      │
        │  HMAC-SHA-96 truncated to 12 bytes             │
        │  encrypted ScopedPDU (AES-128-CFB)             │
        ├───────────────────────────────────────────────▶│
        │                                                │
        │  RESPONSE sysDescr.0 = "Linux ..."             │
        │  same engineID/Boots/Time + HMAC + cipher      │
        │◀───────────────────────────────────────────────│
```

## ASCII: MIB Tree at a Glance

```
.iso.org.dod.internet (1.3.6.1)
├── .mgmt.mib-2 (.2.1)
│   ├── system           (.1)
│   ├── interfaces       (.2)
│   ├── ip               (.4)
│   ├── icmp             (.5)
│   ├── tcp              (.6)
│   ├── udp              (.7)
│   ├── snmp             (.11)
│   ├── bgp              (.15)
│   ├── ospf             (.14)
│   ├── ifMIB            (.31)
│   ├── ipv6 / inet      (.55, in IP-MIB)
│   ├── host             (.25)
│   └── entityMIB        (.47)
├── .private.enterprises (.4.1)
│   ├── 9     cisco
│   ├── 2636  juniper
│   ├── 30065 arista
│   ├── 14988 mikrotik
│   └── 8072  net-snmp
└── .snmpV2 (.6)
    ├── snmpModules (.3)
    │   ├── snmpFrameworkMIB    (.10)
    │   ├── snmpMPDMIB          (.11)
    │   ├── snmpUsmMIB          (.15)
    │   ├── snmpVacmMIB         (.16)
    │   └── snmpCommunityMIB    (.18)
    └── snmpProxys (.5)
```

## Capacity and Sizing

```
Per-device cost (snmpd)         ~5 MB RSS baseline + ~1 MB per 10k OIDs cached
NMS poller per 1k devices       ~2 polls/s/device * 60s burst → ~120k pps PEAK
                                  spread randomly, batch via GETBULK
GETBULK max-rep                 25-50 typical; raise carefully on jumbo intra-DC
PDU size                        agent default 1472 (UDP/IPv4 1500 - 28); some
                                  vendors expose maxMsgSize via 'snmp-server packetsize'
Trap rate (storm protection)    install rate-limiting on snmptrapd via fail2ban
                                  pattern, or use trap dampening on the device
                                  (Cisco: 'snmp-server trap-rate-limit')
```

## Tips

- v3 with authPriv is the only acceptable production posture. v2c
  communities cross every boundary in cleartext.
- Pin `snmpEngineID` in the agent config so localised keys survive
  reboots and IP renumbering.
- `snmp-server ifindex persist` (Cisco) and `set chassis ifd-index-table-size`
  (Junos) keep `ifIndex` stable across reload — your monitoring keeps
  working without re-indexing.
- Always poll `ifHC*` (Counter64) on links ≥1 Gbps. 32-bit counters wrap
  too fast to compute deltas reliably.
- Test SET on a lab device first; many writable OIDs immediately commit
  configuration (BGP shut, interface admin-down).
- Lock the agent down with VACM views — many MIBs leak passwords or
  topology far beyond what your NMS needs (e.g. running config, ACL
  bodies, BGP communities).
- Source-interface and ACLs: bind SNMP to the management VRF / loopback
  and ACL inbound to your NMS only. Many fabrics treat SNMP in the
  default VRF as part of the data path — disastrous under load.
- For agent-side polling/threshold use DISMAN-EVENT-MIB rather than NMS
  polling at high frequency. The agent is closer to the data.
- A SNMP "walk" with no response for 30+ seconds is almost always either
  a missing return-path firewall pinhole or a broken VACM rule, not the
  agent being slow.
- If snmpwalk hangs at the same OID across reboots, it's a vendor MIB
  bug. Skip past with `--maxRepeaters=10 -Cr10` and report to vendor.

## Relationship to Other Protocols

```
ICMP / ping             liveness only — pair with SNMP for context
NetFlow / sFlow / IPFIX flow sampling — pairs with SNMP counters for capacity
Syslog                  free-text events — pairs with SNMP traps (events ≥ thresholds)
TACACS+ / RADIUS        AAA — does NOT cover SNMP USM (separate user DB)
NTP                     critical for v3 time-window check (150s tolerance)
SSH / NETCONF / gNMI    config-side; SNMP is the read-side until streaming wins
LLDP                    L2 discovery — feeds LLDP-MIB used by SNMP topology mappers
```

## Performance Numbers (Typical)

```
snmpd net-snmp 5.9 on a 4-vCPU VM      ~30 000 OID req/s (sustained)
                                        ~80 000 OID req/s (burst, GETBULK)
Cisco IOS-XE control-plane SNMP        ~200-1 500 GET/s before SNMPd CPU saturates
Cisco NX-OS                            ~2 000-5 000 GET/s sustained
Juniper Junos (mib2d)                  ~3 000 GET/s sustained
Arista EOS                             ~5 000 GET/s sustained
```

These numbers vary wildly by chassis and software version; always
measure before tuning poller cadence.

## Compatibility and Pitfalls

- `community` strings sent in cleartext means anyone on path can replay
  GET-requests forever. Treat v2c communities as read-only on a
  *trusted* mgmt VRF only.
- DES-CBC privacy is broken; refuse to enable in any audited
  environment. Use AES-128 minimum, AES-256 where supported (RFC 7860
  HMAC-SHA-2 + Cisco/Blumenthal AES-256).
- HMAC-MD5 should be avoided (collision history); use HMAC-SHA-256 or
  better.
- v3 requires NTP. Without sync the manager will hammer the agent with
  REPORT-induced retries and burn CPU on every authoritative engine.
- AgentX subagents (FRR, SNMPTT) communicate over a Unix socket
  (`/var/agentx/master`) by default. Permissions: snmpd runs as
  `snmp:snmp`; the socket dir must be group-writable by `snmp` for
  rootless subagents.
- Some vendors (older Mikrotik, Ubiquiti EdgeRouter) implement SNMP
  GETBULK incorrectly — always test bulkwalk against bulkwalk-with-N=0.

## Migration Strategy: SNMP → Streaming

```
Phase 0   Inventory: every device, version, MIBs in use, community/v3 user
Phase 1   Add v3 auth+priv on every device while keeping v2c read-only enabled
Phase 2   Stand up streaming pipeline (gNMI / OpenConfig / model-driven-telemetry)
          alongside SNMP polling; reconcile metrics via Prometheus rules
Phase 3   Cut alerts from SNMP traps to streaming events one event class at a time
Phase 4   Decommission v2c communities; restrict v3 to write-only / break-glass
Phase 5   Decommission snmp_exporter / Telegraf SNMP for streamed signal
```

## See Also

- `monitoring/prometheus`
- `monitoring/grafana`
- `monitoring/netflow-ipfix`
- `monitoring/model-driven-telemetry`
- `network-os/cisco-ios`
- `network-os/junos`
- `networking/netconf`
- `networking/dns`
- `networking/bgp`
- `networking/ospf`

## References

- [RFC 1157 — SNMPv1](https://www.rfc-editor.org/rfc/rfc1157)
- [RFC 1901 — Introduction to Community-based SNMPv2](https://www.rfc-editor.org/rfc/rfc1901)
- [RFC 1905 — Protocol Operations for SNMPv2](https://www.rfc-editor.org/rfc/rfc1905)
- [RFC 1908 — Coexistence between v1 and v2](https://www.rfc-editor.org/rfc/rfc1908)
- [RFC 2578 — SMIv2 (Structure of Management Information)](https://www.rfc-editor.org/rfc/rfc2578)
- [RFC 2579 — Textual Conventions for SMIv2](https://www.rfc-editor.org/rfc/rfc2579)
- [RFC 2580 — Conformance Statements for SMIv2](https://www.rfc-editor.org/rfc/rfc2580)
- [RFC 2790 — HOST-RESOURCES-MIB](https://www.rfc-editor.org/rfc/rfc2790)
- [RFC 2863 — IF-MIB (Interfaces Group MIB)](https://www.rfc-editor.org/rfc/rfc2863)
- [RFC 2981 — DISMAN-EVENT-MIB](https://www.rfc-editor.org/rfc/rfc2981)
- [RFC 3411 — SNMP Architecture](https://www.rfc-editor.org/rfc/rfc3411)
- [RFC 3412 — Message Processing and Dispatching](https://www.rfc-editor.org/rfc/rfc3412)
- [RFC 3413 — SNMP Applications](https://www.rfc-editor.org/rfc/rfc3413)
- [RFC 3414 — User-based Security Model (USM)](https://www.rfc-editor.org/rfc/rfc3414)
- [RFC 3415 — View-based Access Control Model (VACM)](https://www.rfc-editor.org/rfc/rfc3415)
- [RFC 3416 — Protocol Operations for SNMPv2](https://www.rfc-editor.org/rfc/rfc3416)
- [RFC 3417 — Transport Mappings for SNMPv2](https://www.rfc-editor.org/rfc/rfc3417)
- [RFC 3418 — MIB for SNMPv2](https://www.rfc-editor.org/rfc/rfc3418)
- [RFC 3584 — Coexistence between v1, v2c, v3](https://www.rfc-editor.org/rfc/rfc3584)
- [RFC 3826 — AES Cipher in SNMP USM](https://www.rfc-editor.org/rfc/rfc3826)
- [RFC 4022 — TCP-MIB](https://www.rfc-editor.org/rfc/rfc4022)
- [RFC 4113 — UDP-MIB](https://www.rfc-editor.org/rfc/rfc4113)
- [RFC 4188 — BRIDGE-MIB](https://www.rfc-editor.org/rfc/rfc4188)
- [RFC 4273 — BGP4-MIB](https://www.rfc-editor.org/rfc/rfc4273)
- [RFC 4293 — IP-MIB](https://www.rfc-editor.org/rfc/rfc4293)
- [RFC 4363 — Q-BRIDGE-MIB](https://www.rfc-editor.org/rfc/rfc4363)
- [RFC 4444 — ISIS-MIB](https://www.rfc-editor.org/rfc/rfc4444)
- [RFC 4502 — RMON2-MIB](https://www.rfc-editor.org/rfc/rfc4502)
- [RFC 4560 — DISMAN-PING/TRACE/NSLOOKUP-MIB](https://www.rfc-editor.org/rfc/rfc4560)
- [RFC 4668 — RADIUS-AUTH-CLIENT-MIB](https://www.rfc-editor.org/rfc/rfc4668)
- [RFC 4750 — OSPFv2 MIB](https://www.rfc-editor.org/rfc/rfc4750)
- [RFC 4789 — SNMP over IEEE 802 Networks](https://www.rfc-editor.org/rfc/rfc4789)
- [RFC 5343 — SNMP Context EngineID Discovery](https://www.rfc-editor.org/rfc/rfc5343)
- [RFC 5590 — Transport Subsystem](https://www.rfc-editor.org/rfc/rfc5590)
- [RFC 5591 — Transport Security Model](https://www.rfc-editor.org/rfc/rfc5591)
- [RFC 5592 — SSH Transport Model](https://www.rfc-editor.org/rfc/rfc5592)
- [RFC 5643 — OSPFv3 MIB](https://www.rfc-editor.org/rfc/rfc5643)
- [RFC 5953 — TLS Transport Model](https://www.rfc-editor.org/rfc/rfc5953)
- [RFC 6353 — TLS Transport Model](https://www.rfc-editor.org/rfc/rfc6353)
- [RFC 6933 — ENTITY-MIB v4](https://www.rfc-editor.org/rfc/rfc6933)
- [RFC 7860 — HMAC-SHA-2 Authentication for USM](https://www.rfc-editor.org/rfc/rfc7860)
- [Net-SNMP Project](http://www.net-snmp.org/)
- [Net-SNMP — snmpwalk Man Page](http://www.net-snmp.org/docs/man/snmpwalk.html)
- [Net-SNMP — snmpd.conf Man Page](http://www.net-snmp.org/docs/man/snmpd.conf.html)
- [Net-SNMP — snmptrapd.conf Man Page](http://www.net-snmp.org/docs/man/snmptrapd.conf.html)
- [Prometheus snmp_exporter](https://github.com/prometheus/snmp_exporter)
- [Prometheus snmp_exporter Generator](https://github.com/prometheus/snmp_exporter/tree/main/generator)
- [InfluxData Telegraf SNMP Input Plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/snmp)
- [LibreNMS](https://www.librenms.org/)
- [Observium](https://www.observium.org/)
- [Zabbix SNMP Documentation](https://www.zabbix.com/documentation/current/en/manual/config/items/itemtypes/snmp)
- [IANA — SNMP Number Spaces](https://www.iana.org/assignments/smi-numbers/smi-numbers.xhtml)
- [IANA — Private Enterprise Numbers (1.3.6.1.4.1.X)](https://www.iana.org/assignments/enterprise-numbers/enterprise-numbers)
- [Cisco SNMP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/snmp/configuration/xe-16/snmp-xe-16-book.html)
- [Cisco SNMP Object Navigator](https://snmp.cloudapps.cisco.com/Support/SNMP/do/BrowseOID.do)
- [Juniper SNMP Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/network-mgmt/topics/topic-map/snmp-overview.html)
- [Arista EOS SNMP Configuration](https://www.arista.com/en/um-eos/eos-section-43-1-snmp-introduction)
- ["Essential SNMP" — Mauro & Schmidt, O'Reilly, 2005](https://www.oreilly.com/library/view/essential-snmp-2nd/0596008406/)
- man pages: snmpwalk(1), snmpget(1), snmpset(1), snmptrap(1), snmpd(8), snmptrapd(8), snmpd.conf(5), snmptrapd.conf(5), snmp.conf(5), mib2c(1), smistrip(1), smilint(1)
