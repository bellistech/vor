# The Mathematics of SNMP — Protocol Internals, ASN.1 Encoding, and Polling Theory

> *SNMP (Simple Network Management Protocol) is an ASN.1/BER-encoded request-response protocol layered over UDP, organized around a globally-rooted MIB tree of OIDs. Every wire byte is a TLV triple, every authenticated message is an HMAC-truncated digest, every WALK is a deterministic lexicographic traversal, and every poll cycle is a queueing problem. The math governs whether a 50,000-device fleet polls cleanly in 60 seconds or buckles under retransmit storms.*

---

## 0. Scope and Layering

SNMP sits at the application layer on top of UDP/161 (agent) and UDP/162 (trap receiver). The protocol is defined across three orthogonal documents that must be read together:

| Layer | Defines | Primary RFCs |
|:---|:---|:---|
| Data definition (SMI) | How OIDs and types are described | RFC 1155 (SMIv1), RFC 2578-2580 (SMIv2) |
| Encoding | How PDUs become bytes on the wire | ITU-T X.680 (ASN.1), X.690 (BER) |
| Protocol operations | What messages exist and how they flow | RFC 1157 (v1), RFC 3416 (v2c/v3) |
| Architecture & security | Engines, USM, VACM | RFC 3411-3415, RFC 7860 |

The protocol stack at runtime:

```
+----------------------------------------+
|        Application (NMS / agent)       |
+----------------------------------------+
|  Dispatcher  | Message Processing      |
|  v1 / v2c / v3 subsystems              |
+----------------------------------------+
|  Security: Community (v1/v2c) | USM(v3)|
|  Access:   VACM (v3)                   |
+----------------------------------------+
|  ASN.1 / BER encoding (X.690)          |
+----------------------------------------+
|  UDP  (port 161 polling, 162 traps)    |
+----------------------------------------+
|  IPv4 / IPv6                           |
+----------------------------------------+
```

Three protocol versions remain in production use:

| Version | RFCs | Auth | Encryption | Notes |
|:---:|:---|:---|:---|:---|
| SNMPv1 | 1157 | community string (cleartext) | none | Counter32 only, GETNEXT only |
| SNMPv2c | 1901-1908 | community string (cleartext) | none | Counter64, GETBULK, INFORM |
| SNMPv3 | 3411-3418, 7860 | HMAC-MD5/SHA/SHA-2 | DES/AES-128/192/256 | USM + VACM, replay protection |

---

## 1. ASN.1 / BER Encoding (X.680, X.690)

Every SNMP PDU is serialized using the Basic Encoding Rules (BER) of ASN.1 — a Tag-Length-Value (TLV) format where each datum is three concatenated fields:

```
+--------+--------+----------+
|  Tag   | Length |  Value   |
| 1+ B   | 1+ B   | L bytes  |
+--------+--------+----------+
```

### 1.1 Tag Octet Encoding

The first octet (or first few octets, for high-numbered tags) encodes three independent properties:

```
 7 6 | 5 | 4 3 2 1 0
+----+---+---------+
|cls |P/C|tag-num  |
+----+---+---------+
```

- **Class** (bits 7-6):
  - `00` = Universal (built-in ASN.1 types)
  - `01` = Application (SNMP-specific: IpAddress, Counter32, etc.)
  - `10` = Context-specific (PDU types)
  - `11` = Private
- **P/C** (bit 5): 0 = Primitive (value is raw bytes), 1 = Constructed (value is more TLVs)
- **Tag number** (bits 4-0): If 0x1F, the tag is multi-byte; subsequent bytes use bit 7 as a continuation flag and bits 6-0 as base-128 digits.

### 1.2 Length Octet Encoding

BER offers two length forms:

- **Short form** (length < 128): one octet, bit 7 clear, bits 6-0 = length.
- **Long form** (length ≥ 128): first octet has bit 7 set; bits 6-0 = number of subsequent octets that hold the big-endian length value. (Indefinite form, 0x80, exists in BER but is forbidden in SNMP — a definite length must be sent.)

| Length | Encoding (hex) |
|:---:|:---|
| 5 | `05` |
| 127 | `7F` |
| 128 | `81 80` |
| 256 | `82 01 00` |
| 65535 | `82 FF FF` |
| 1,000,000 | `83 0F 42 40` |

### 1.3 Universal ASN.1 Types Used by SNMP

| Tag (decimal) | Hex | Type | Notes |
|:---:|:---:|:---|:---|
| 0x02 | 02 | INTEGER | Two's complement, big-endian, minimum bytes |
| 0x04 | 04 | OCTET STRING | Arbitrary bytes |
| 0x05 | 05 | NULL | Always length 0; used as placeholder in GET varbinds |
| 0x06 | 06 | OBJECT IDENTIFIER | Variable-length subidentifier encoding (see §1.5) |
| 0x30 | 30 | SEQUENCE | Constructed; ordered |
| 0x30 | 30 | SEQUENCE OF | Same tag as SEQUENCE |

### 1.4 SNMP Application Types (RFC 2578)

| Tag | Hex | Type | Range / Notes |
|:---:|:---:|:---|:---|
| `[APPLICATION 0]` | 40 | IpAddress | 4 octets, network byte order |
| `[APPLICATION 1]` | 41 | Counter32 | uint32, monotonic, wraps at 2^32 |
| `[APPLICATION 2]` | 42 | Gauge32 / Unsigned32 | uint32, may go up or down |
| `[APPLICATION 3]` | 43 | TimeTicks | uint32, hundredths of a second |
| `[APPLICATION 4]` | 44 | Opaque | OCTET STRING wrapping arbitrary BER |
| `[APPLICATION 5]` | 45 | NsapAddress | rare |
| `[APPLICATION 6]` | 46 | Counter64 | uint64, RFC 2578 |

### 1.5 OBJECT IDENTIFIER Encoding

The first two subidentifiers are merged: `40 * x + y`. Each subsequent subidentifier is encoded base-128 with bit 7 of every byte except the last set to 1 (the continuation flag).

Example: encode `1.3.6.1.2.1.1.3.0` (sysUpTime.0):

```
1.3        -> 40*1 + 3 = 43       = 0x2B
6          -> 6                   = 0x06
1          -> 1                   = 0x01
2          -> 2                   = 0x02
1          -> 1                   = 0x01
1          -> 1                   = 0x01
3          -> 3                   = 0x03
0          -> 0                   = 0x00

Encoded value bytes: 2B 06 01 02 01 01 03 00   (8 bytes)
With OID tag 06 + length 08:
06 08 2B 06 01 02 01 01 03 00
```

For a subidentifier like `8072` (commonly seen at `1.3.6.1.4.1.8072` — net-snmp's PEN):

```
8072 = 0x1F88 = 0b 0011_1111_0000_1000
Group into 7-bit chunks (high to low): 0b0111111  0b0001000
                                     = 63        = 8
With continuation:                     0xBF       0x08
```

### 1.6 Worked Example — Encoding `sysUpTime.0 = 12345`

The full varbind `OID = sysUpTime.0`, value `TimeTicks 12345`:

```
INTEGER step:
  TimeTicks tag = [APPLICATION 3] = 0x43
  Value 12345 = 0x3039 → 2 bytes
  TLV: 43 02 30 39

OID step (from §1.5):
  06 08 2B 06 01 02 01 01 03 00

VarBind = SEQUENCE { name OID, value ANY }:
  Inner = (OID TLV) || (value TLV)
        = 06 08 2B 06 01 02 01 01 03 00 || 43 02 30 39   = 14 bytes
  SEQUENCE:
    30 0E 06 08 2B 06 01 02 01 01 03 00 43 02 30 39
```

Total VarBind = 16 bytes. A request that asks for sysUpTime by name (value-side `NULL` 0x05 0x00) would be only 12 bytes for that single VarBind.

### 1.7 INTEGER Minimum-Length Rule

INTEGER values use the *minimum* number of octets in two's complement. Rules:

- The most significant bit of the first content octet is the sign bit.
- Leading 0x00 octets are forbidden if the next byte's high bit is 0.
- Leading 0xFF octets are forbidden if the next byte's high bit is 1.

Examples:

| Value | Encoded | Why |
|:---:|:---|:---|
| 0 | `02 01 00` | one byte, value 0 |
| 127 | `02 01 7F` | sign bit clear |
| 128 | `02 02 00 80` | leading 0x00 needed so 0x80 is not interpreted as -128 |
| -1 | `02 01 FF` | all-ones |
| -128 | `02 01 80` | one-byte two's complement |
| -129 | `02 02 FF 7F` | needs leading 0xFF |

Counter32, Gauge32, TimeTicks use the *unsigned* INTEGER convention but follow the same minimum-length rule with an implicit non-negative range — a leading 0x00 *is* required when the high bit would otherwise be set.

---

## 2. PDU Structure Mathematics

### 2.1 SNMPv1 / v2c Message

```
Message ::= SEQUENCE {
  version       INTEGER,           -- 0=v1, 1=v2c, 3=v3
  community     OCTET STRING,
  data          ANY                -- the PDU
}

GetRequest-PDU      ::= [0]  IMPLICIT PDU
GetNextRequest-PDU  ::= [1]  IMPLICIT PDU
Response-PDU        ::= [2]  IMPLICIT PDU
SetRequest-PDU      ::= [3]  IMPLICIT PDU
GetBulkRequest-PDU  ::= [5]  IMPLICIT BulkPDU
InformRequest-PDU   ::= [6]  IMPLICIT PDU
SNMPv2-Trap-PDU     ::= [7]  IMPLICIT PDU
Report-PDU          ::= [8]  IMPLICIT PDU

PDU ::= SEQUENCE {
  request-id        INTEGER (-2147483648..2147483647),
  error-status      INTEGER {
                      noError(0), tooBig(1), noSuchName(2),
                      badValue(3), readOnly(4), genErr(5),
                      noAccess(6), wrongType(7), wrongLength(8),
                      wrongEncoding(9), wrongValue(10), noCreation(11),
                      inconsistentValue(12), resourceUnavailable(13),
                      commitFailed(14), undoFailed(15), authorizationError(16),
                      notWritable(17), inconsistentName(18) },
  error-index       INTEGER (0..max-bindings),
  variable-bindings VarBindList
}

BulkPDU ::= SEQUENCE {
  request-id        INTEGER,
  non-repeaters     INTEGER (0..max-bindings),
  max-repetitions   INTEGER (0..max-bindings),
  variable-bindings VarBindList
}

VarBindList ::= SEQUENCE OF VarBind
VarBind ::= SEQUENCE { name OBJECT IDENTIFIER, value ANY }
```

### 2.2 PDU Size Budget

Default UDP datagram limit is 65,507 bytes (65535 - 8 UDP - 20 IPv4). RFC 3417 mandates an agent MUST accept at least 484 bytes; modern implementations support 65,535 bytes. However, IP fragmentation in the WAN is the practical limit — most operators size for path MTU minus headers:

```
PDU_max ≤ MTU_path - L2_overhead - 20 (IPv4) - 8 (UDP) - msg_overhead

Standard Ethernet:    1500 - 14 - 20 - 8  = 1458 bytes
Jumbo:                9000 - 14 - 20 - 8  = 8958 bytes
Internet path-safe:   1280 - 20 - 8       = 1252 bytes (IPv6 minimum)
```

Every layer of nesting consumes a few bytes for tag + length:

| Field | Cost (bytes) |
|:---|:---:|
| Outer SEQUENCE tag/length | 4 |
| Version INTEGER | 3 |
| Community OCTET STRING | 2 + len(community) |
| PDU tag/length | 4 |
| request-id INTEGER | 6 |
| error-status INTEGER | 3 |
| error-index INTEGER | 3 |
| VarBindList SEQUENCE | 4 |
| Per VarBind | 2 + 2 + len(OID) + len(value-TLV) |
| **Fixed overhead (typical)** | ~30 + len(community) |

### 2.3 The "tooBig" Error

When an agent's response *would* exceed its `snmpMaxMessageSize` (or the configured PDU buffer), it MUST return a Response PDU with `error-status = tooBig(1)` and an empty VarBindList (RFC 3416 §4.2.5). Manager behaviour:

1. Reduce `max-repetitions` (for GETBULK) or split the OID list (for GET).
2. Retry.

The condition occurs when the response budget is bounded by:

```
S_response = 30 + len(community) + Σ (2 + len(OID_i) + len(value_TLV_i))
```

If `S_response > min(local_buf, remote_buf, MTU_safe)`, tooBig fires. Production implementations track per-agent `snmpMaxResponseSize` and pre-trim.

---

## 3. SNMPv3 USM Cryptographic Analysis (RFC 3414, RFC 7860)

USM (User-based Security Model) provides three security levels:

| Level | Auth | Priv | Keyword |
|:---|:---:|:---:|:---|
| noAuthNoPriv | none | none | `noAuthNoPriv` |
| authNoPriv | HMAC | none | `authNoPriv` |
| authPriv | HMAC | symmetric | `authPriv` |

### 3.1 Authentication Algorithms

| Algorithm | RFC | Hash output | HMAC truncation | Localized key length |
|:---|:---:|:---:|:---:|:---:|
| HMAC-MD5-96 | 3414 | 128 bits | 96 bits (12 B) | 16 bytes |
| HMAC-SHA-96 | 3414 | 160 bits | 96 bits (12 B) | 20 bytes |
| HMAC-SHA-224-128 | 7860 | 224 bits | 128 bits (16 B) | 28 bytes |
| HMAC-SHA-256-192 | 7860 | 256 bits | 192 bits (24 B) | 32 bytes |
| HMAC-SHA-384-256 | 7860 | 384 bits | 256 bits (32 B) | 48 bytes |
| HMAC-SHA-512-384 | 7860 | 512 bits | 384 bits (48 B) | 64 bytes |

Note: MD5 has been cryptanalytically broken for collision resistance, but HMAC-MD5 has not been broken in 2026; nonetheless, modern deployments choose SHA-256 or higher.

### 3.2 Privacy Algorithms

| Algorithm | RFC | Mode | Block | Key | Notes |
|:---|:---:|:---|:---:|:---:|:---|
| DES-CBC | 3414 | CBC | 64 bits | 56 bits | Deprecated; broken |
| 3DES-CBC | drafts | CBC-EDE | 64 bits | 168 bits | Legacy only |
| AES-128-CFB | 3826 | CFB | 128 bits | 128 bits | Mainstream |
| AES-192-CFB | 7860 | CFB | 128 bits | 192 bits | Optional |
| AES-256-CFB | 7860 | CFB | 128 bits | 256 bits | High-security deployments |

### 3.3 Password-to-Key Algorithm (RFC 3414 §2.6)

The "1MB key derivation" expands a passphrase into a master key by hashing exactly 1,048,576 bytes (2^20) of repeated passphrase. Pseudocode:

```
function passwordToMasterKey(password, hash):
    buf := bytes()
    while len(buf) < 1_048_576:
        buf := buf || password
    truncate buf to 1_048_576 bytes
    return hash(buf)
```

This is by design "deliberately expensive": at MD5 ~700 MB/s, 1 MB = ~1.5 ms; at SHA-512 ~400 MB/s, 1 MB = ~2.5 ms. Multiplied by every SNMPv3 user × engine relearning, it discourages brute-forcing and makes each key materialization a first-class lifecycle event.

### 3.4 Engine ID Localization

Once the master key `Ku` is derived, it is localized to a specific authoritative engine ID `E`:

```
K_localized = hash( Ku || E || Ku )
```

Properties:

- A user has the same master password but a *different* localized key on every engine.
- Compromise of one engine's key store does not yield the user's password (preimage resistance) and does not yield other engines' keys (engine ID is part of the input).
- Replay across engines is impossible because messages are authenticated under the destination engine's localized key.

```
       password (utf8)
            │
            ▼
   ┌──────────────────┐
   │ pad to 1 MiB     │  buf = repeat(pw) trunc 2^20
   └────────┬─────────┘
            ▼
   ┌──────────────────┐
   │ H(buf) = Ku      │  master key
   └────────┬─────────┘
            ▼
   for each engine E:
   ┌──────────────────┐
   │ H(Ku || E || Ku) │  = K_localized(E,user)
   └──────────────────┘
```

---

## 4. The Authentication Algorithm Step-by-Step

For an outgoing `authPriv` message:

1. Build the unencrypted `scopedPDU = SEQUENCE { contextEngineID, contextName, data }`.
2. Encrypt `scopedPDU` using AES-128-CFB with key `K_priv_loc` and a freshly generated 16-byte IV (engineBoots ‖ engineTime ‖ 64-bit salt). The salt is sent as `msgPrivacyParameters`.
3. Build the full message structure with `msgAuthenticationParameters` set to 12 zero bytes.
4. Compute `tag = HMAC(K_auth_loc, wholeMsgWithZeroedAuth)`; truncate `tag` to 96 (or 128/192/256/384) bits.
5. Replace the zero placeholder with the truncated tag.
6. Transmit.

Receiver runs steps in reverse: zero the auth field, recompute HMAC, constant-time compare; on success, decrypt scopedPDU.

### 4.1 Byte-Flow Diagram

```
+---------------------------------- wholeMsg ----------------------------------+
| SEQUENCE                                                                     |
|   version=3                                                                  |
|   msgGlobalData = SEQUENCE {                                                 |
|       msgID, msgMaxSize, msgFlags=0x07 (auth+priv+reportable), msgSecModel=3 |
|   }                                                                          |
|   msgSecurityParameters = OCTET STRING wrapping SEQUENCE {                   |
|       msgAuthoritativeEngineID,                                              |
|       msgAuthoritativeEngineBoots,                                           |
|       msgAuthoritativeEngineTime,                                            |
|       msgUserName,                                                           |
|       msgAuthenticationParameters  ← 12 zero bytes during HMAC               |
|       msgPrivacyParameters         ← 8-byte salt for CFB IV                  |
|   }                                                                          |
|   msgData = ENCRYPTED scopedPDU                                              |
+------------------------------------------------------------------------------+
                       │
                       ▼ HMAC over the whole serialized
                          message with msgAuth = 0×00..00
                       │
                       ▼ truncate to {96|128|192|256|384} bits
                       │
                       ▼ overwrite msgAuthenticationParameters
```

### 4.2 AES-CFB IV Construction (RFC 3826)

```
IV = msgAuthoritativeEngineBoots (4 B BE)
   ‖ msgAuthoritativeEngineTime  (4 B BE)
   ‖ salt                        (8 B random per packet)
```

The salt is transmitted in `msgPrivacyParameters`. Crucially, the salt MUST be unique per engineBoots/engineTime tuple — collisions break confidentiality of CFB. Implementations use a 64-bit counter that is checkpointed across reboots.

---

## 5. Anti-Replay Protection

SNMPv3 binds every authenticated PDU to the destination engine's monotonic time:

| Field | Source | Meaning |
|:---|:---|:---|
| `msgAuthoritativeEngineID` | The agent | Globally unique engine ID |
| `msgAuthoritativeEngineBoots` | The agent | Increments every reboot, persistent |
| `msgAuthoritativeEngineTime` | The agent | Seconds since last boot |

A receiver accepts a PDU iff:

```
msg.engineID == local.engineID AND
msg.engineBoots == local.engineBoots AND
|msg.engineTime - local.engineTime| ≤ 150 s
```

If `msg.engineBoots > local.engineBoots`, the receiver may update its cached value (for managers tracking many agents). If outside the 150-second window, the agent returns a `usmStatsNotInTimeWindows` Report PDU and discards. The 150-second window means clock drift between manager and agent must stay below 2.5 minutes; in practice, NTP slew with a 0.5-ppm drift only adds ~43 ms/day, so a freshly synced fleet is fine but a long-isolated device will eventually fail.

Manager-driven discovery (RFC 3414 §4) bootstraps clocks: send a Report request with engineBoots/engineTime = 0 → agent responds with its real values → manager updates cache → subsequent requests pass the time check.

---

## 6. VACM (View-Based Access Control Model, RFC 3415)

VACM converts an authenticated message into an access decision. The decision is a four-step lookup:

```
Step 1: (securityModel, securityName) → groupName       [ vacmSecurityToGroupTable ]
Step 2: (groupName, contextName,
         securityModel, securityLevel) → viewName       [ vacmAccessTable ]
Step 3: For each varbind, lookup viewName + OID         [ vacmViewTreeFamilyTable ]
Step 4: Permitted iff longest-match entry has
        type = included
```

### 6.1 Decision Algorithm

```
function vacmCheck(securityModel, securityName, securityLevel,
                   contextName, viewType, oid):
    g := lookupGroup(securityModel, securityName)
    if g == nil: return notInView
    a := lookupAccess(g, contextName, securityModel, securityLevel, viewType)
    if a == nil: return notInView
    v := a.viewName[viewType]   # read | write | notify
    if v == "" : return notInView
    return matchesView(v, oid)

function matchesView(v, oid):
    bestLen := -1
    bestType := excluded
    for entry in viewTreeFamily where entry.viewName == v:
        if subtreePrefixMatches(entry.subtree, entry.mask, oid):
            if length(entry.subtree) > bestLen:
                bestLen := length(entry.subtree)
                bestType := entry.type    # included | excluded
    return bestType == included
```

### 6.2 Worked Truth Table

Configuration:

```
SECURITY-TO-GROUP:
  (USM, "alice")  → group "ops"
  (USM, "bob")    → group "ro"

ACCESS:
  (group=ops, ctx="", USM, authPriv) → readView=all,  writeView=all
  (group=ro,  ctx="", USM, authPriv) → readView=mib2, writeView=""

VIEW TREE:
  view "all"  : 1.3.6.1                    included, mask=ff
  view "mib2" : 1.3.6.1.2.1                included, mask=ff
  view "mib2" : 1.3.6.1.2.1.4.21           excluded, mask=ff   -- IP route table
```

| User | OP | OID | Decision |
|:---|:---|:---|:---|
| alice (authPriv) | GET | 1.3.6.1.4.1.9.2.1.58 | included by `all` → permitted |
| alice (authPriv) | SET | 1.3.6.1.2.1.1.5.0 (sysName) | writeView=all → permitted |
| bob (authPriv) | GET | 1.3.6.1.2.1.1.3.0 (sysUpTime) | included by `mib2` → permitted |
| bob (authPriv) | GET | 1.3.6.1.2.1.4.21.1 | longest-match excluded → denied |
| bob (authPriv) | SET | any | writeView empty → denied |
| bob (authNoPriv) | GET | any | level mismatch → notInView |

### 6.3 View Mask Semantics

Each subtree carries a bitmask `mask` aligned to subidentifier positions; mask bit `i` = 1 means subidentifier `i` of the test OID must equal the configured value, mask bit 0 = wildcard. Mask `ff..` (all 1s) means strict prefix match; sparse masks let one entry match families like `1.3.6.1.2.1.2.2.1.*.{interface}`.

---

## 7. Polling Theory and Math

### 7.1 Per-Operation Costs

| Operation | Wire cost | Round trips |
|:---|:---|:---:|
| GET (single OID) | 2 packets | 1 RTT |
| GET (N OIDs in one PDU) | 2 packets | 1 RTT (if fits) |
| GETNEXT WALK of N OIDs | 2N packets | N RTT |
| GETBULK WALK of N OIDs (max-rep = M) | 2 ⌈N/M⌉ packets | ⌈N/M⌉ RTT |

### 7.2 Bandwidth of a Periodic Pollset

For a fleet of `D` devices, polling `O` objects each every `T` seconds, with a per-OID encoded size of `b` bytes (typical 50 bytes BER + 50 bytes IP/UDP overhead amortized):

```
BW_total = D × O × b × 8 / T   bits/sec
```

Worked numbers:

| D | O | T | b | BW_total |
|---:|---:|---:|---:|---:|
| 100 | 100 | 60 s | 100 B | 13.3 kbps |
| 1,000 | 100 | 60 s | 100 B | 133 kbps |
| 1,000 | 200 | 60 s | 100 B | 267 kbps |
| 10,000 | 100 | 60 s | 100 B | 1.33 Mbps |
| 50,000 | 200 | 60 s | 100 B | 13.3 Mbps |

### 7.3 Optimal `max-repetitions` for GETBULK

Two pressures:

1. **Latency**: ⌈N/M⌉ RTTs grows for small `M`.
2. **Wasted work**: GETBULK returns variables strictly past the requested OID, but it returns `M` of them per varbind in the request — exceeding the subtree boundary wastes both agent CPU and bandwidth.

If a target subtree contains exactly `S` objects and we ask for `M`:

```
RTTs    = ⌈S / M⌉
Excess  = M - (S mod M)        (if S mod M != 0)
PDU_size ≈ overhead + M × b
```

For PDU_size constrained by MTU `U`:

```
M_max = (U - overhead) / b
M_opt = min( M_max, ⌈S / RTT_budget⌉ )
```

Practical guidance:

- 10-50 for densely-populated tables (interfaces, BGP peers).
- Keep `max-repetitions × b ≤ 1400` to fit standard MTU.
- Set `non-repeaters = 0` unless mixing scalars and tables.

### 7.4 Polling Jitter and Concurrency

A poller with `C` concurrent workers, mean per-poll latency `L`, can sustain:

```
QPS_poller = C / L
```

To poll D devices each requiring `K` GETBULK calls, every `T` seconds:

```
Required QPS = D × K / T
C_min = ⌈ Required QPS × L ⌉
```

Example: D = 5,000, K = 4 GETBULKs/device, T = 60 s, L = 50 ms.

```
Required QPS = 5,000 × 4 / 60 = 333.3
C_min        = ⌈333.3 × 0.050⌉ = 17 workers
```

To absorb tail latency (P99 ≈ 4× P50) and scrub jitter, multiply by 2-4.

### 7.5 Latency Distribution

Per-poll latency decomposes:

```
L = T_serialize + T_wire + T_agent + T_wire + T_parse
```

Typical values on a LAN:

```
T_serialize ≈ 10-100 µs   (poller side)
T_wire      ≈ 0.1-2 ms    (depends on path)
T_agent     ≈ 1-100 ms    (varies by vendor and load)
T_parse     ≈ 10-100 µs
```

Across WAN paths, `T_wire` dominates: 20-80 ms RTT for cross-continental, plus 2-5% packet loss → expect 1-2 retransmits at the application layer.

---

## 8. PDU Size Limits

| Boundary | Bytes | Cause |
|:---|---:|:---|
| Theoretical UDP/IPv4 | 65,507 | UDP length field (16 bits) - 8 (UDP) - 20 (IPv4) |
| Theoretical UDP/IPv6 (jumbogram disabled) | 65,527 | minus IPv6 header |
| Modern net-snmp default `agentXSocket` | 65,535 | configurable |
| Junos/IOS XR default | 1472 | MTU - 28 |
| Practical Internet path | 1280-1452 | smallest common path MTU |

When response > path MTU, the IP layer fragments. Fragmentation problems in SNMP:

- **MTU black holes** in firewalls dropping fragments silently.
- **Path MTU discovery failure** under DF=1 + ICMP Frag Needed blocked.
- **Reassembly DoS surface** at the agent.

Mitigation: shrink GETBULK `max-repetitions` so `PDU ≤ 1400 B` is the rule of thumb. Production NMS platforms expose `--max-pdu-size` and probe the right value via OctetString round-trips.

---

## 9. Lexicographic Ordering of OIDs

OIDs are arrays of unsigned subidentifiers. Lexicographic comparison `a <_lex b`:

```
function lex_cmp(a, b):
    n = min(len(a), len(b))
    for i in 0..n-1:
        if a[i] < b[i]: return -1
        if a[i] > b[i]: return  1
    if len(a) < len(b): return -1
    if len(a) > len(b): return  1
    return 0
```

A WALK is the iteration:

```
function walk(start_oid):
    current = start_oid
    while true:
        next = GETNEXT(current)
        if next == endOfMibView: break
        if not isPrefix(start_oid, next): break
        emit next
        current = next.oid
```

A bulk walk substitutes:

```
function bulk_walk(start_oid, max_rep):
    current = start_oid
    while true:
        results = GETBULK(non_repeaters=0, max_rep=max_rep, current)
        for r in results:
            if r.oid > end_of_subtree(start_oid): return
            emit r
        if last(results).oid > end_of_subtree(start_oid): return
        current = last(results).oid
```

### 9.1 Trie Representation of the MIB Tree

```
                  1
                  └─ 3
                     └─ 6
                        └─ 1
                           ├─ 2 (mib-2)
                           │   └─ 1 (system) → sysDescr.0, sysObjectID.0, ...
                           ├─ 4 (private)
                           │   └─ 1 (enterprises)
                           │       ├─ 9   (Cisco)
                           │       ├─ 2636 (Juniper)
                           │       └─ 8072 (net-snmp)
                           └─ 6 (snmpV2)
                               └─ 3 (snmpModules)
```

Walking `1.3.6.1.2.1` traverses MIB-2 in the order: `1.3.6.1.2.1.1.1.0`, `1.3.6.1.2.1.1.2.0`, …, `1.3.6.1.2.1.1.9.x`, `1.3.6.1.2.1.2.1.0` (ifNumber), `1.3.6.1.2.1.2.2.1.1.1` (ifIndex.1), and so on.

Agents implement the OID space as a balanced search tree (red-black, B-tree, or trie); GETNEXT is `O(log N)` for `N` registered OIDs. Net-snmp uses a sorted linked list of "subtrees" with binary-search at the top, then linear scan inside a subtree.

---

## 10. SMIv2 Type System (RFC 2578)

### 10.1 Base Types

| ASN.1 base | SMI use | Range |
|:---|:---|:---|
| INTEGER | INTEGER, Integer32 | -2^31 .. 2^31-1 |
| INTEGER (sub) | enumerations | named values |
| OCTET STRING | OCTET STRING | 0..65535 octets |
| OBJECT IDENTIFIER | OBJECT IDENTIFIER | global tree |
| NULL | NULL | placeholder only |
| IpAddress | IpAddress | 4-octet IPv4 |
| Counter32 | Counter32 | 0..2^32-1, monotonic |
| Gauge32 / Unsigned32 | Gauge32 | 0..2^32-1 |
| TimeTicks | TimeTicks | 0..2^32-1, in 1/100 s |
| Opaque | Opaque | wraps arbitrary BER |
| Counter64 | Counter64 | 0..2^64-1 |
| BITS | BITS | named bit positions |

### 10.2 Common Textual Conventions

| TC | Defined in | Underlying | Meaning |
|:---|:---|:---|:---|
| TruthValue | RFC 1903 / 2579 | INTEGER | true(1), false(2) |
| RowStatus | RFC 1903 / 2579 | INTEGER | active(1), notInService(2), notReady(3), createAndGo(4), createAndWait(5), destroy(6) |
| StorageType | RFC 1903 / 2579 | INTEGER | other(1), volatile(2), nonVolatile(3), permanent(4), readOnly(5) |
| MacAddress | RFC 2578 | OCTET STRING (SIZE(6)) | 6-byte MAC |
| PhysAddress | RFC 2578 | OCTET STRING | generic L2 |
| DateAndTime | RFC 2579 | OCTET STRING (SIZE(8|11)) | year/month/.../tz |
| DisplayString | RFC 2579 | OCTET STRING (SIZE(0..255)) | NVT ASCII |
| InetAddress | RFC 4001 | OCTET STRING | family-tagged address |

### 10.3 Conformance Hierarchy

SMIv2 organizes definitions into:

- **MODULE-IDENTITY** — a single `1` per MIB file.
- **OBJECT-TYPE** — leaf objects with `MAX-ACCESS`, `STATUS`, `DESCRIPTION`.
- **OBJECT-IDENTITY** — non-leaf nodes, tags.
- **MODULE-COMPLIANCE** — required/optional groups for compliance levels.
- **OBJECT-GROUP** / **NOTIFICATION-GROUP** — aggregate compliance units.
- **AGENT-CAPABILITIES** — vendor-claimed support set.

A vendor's MIB module imports from RFC modules (`SNMPv2-SMI`, `SNMPv2-TC`, `IF-MIB`, …) and registers under their PEN (`1.3.6.1.4.1.<PEN>`).

---

## 11. SNMP Trap Format Math

### 11.1 v1 Trap PDU (legacy)

```
Trap-PDU ::= [4] IMPLICIT SEQUENCE {
  enterprise        OBJECT IDENTIFIER,
  agent-addr        IpAddress,
  generic-trap      INTEGER {
                       coldStart(0), warmStart(1), linkDown(2), linkUp(3),
                       authenticationFailure(4), egpNeighborLoss(5),
                       enterpriseSpecific(6) },
  specific-trap     INTEGER,
  time-stamp        TimeTicks,
  variable-bindings VarBindList
}
```

### 11.2 v2c / v3 SNMPv2-Trap PDU

A Trap is just a PDU with tag `[7]`. The first two varbinds are mandatory and ordered:

```
varbind[0] = sysUpTime.0       (1.3.6.1.2.1.1.3.0)        TimeTicks
varbind[1] = snmpTrapOID.0     (1.3.6.1.6.3.1.1.4.1.0)    OID = the trap OID
varbind[2..] = trap-specific data
```

Why the ordering matters:

- Trap correlation — an NMS keys events by `snmpTrapOID.0`, which is fixed-position.
- `sysUpTime.0` lets the receiver detect a boot since the last trap (`uptime_now < uptime_prev`) and de-duplicate "ColdStart spam" after a flap.

### 11.3 Standard Trap OIDs (RFC 3418)

| OID | Name |
|:---|:---|
| `1.3.6.1.6.3.1.1.5.1` | coldStart |
| `1.3.6.1.6.3.1.1.5.2` | warmStart |
| `1.3.6.1.6.3.1.1.5.3` | linkDown |
| `1.3.6.1.6.3.1.1.5.4` | linkUp |
| `1.3.6.1.6.3.1.1.5.5` | authenticationFailure |

Mapping v1 generic trap N → v2c OID: `1.3.6.1.6.3.1.1.5.(N+1)` (RFC 3584).

### 11.4 INFORM vs Trap

- **Trap**: fire-and-forget UDP. Loss = silent.
- **INFORM**: receiver acknowledges with a Response PDU. Sender retries with timeout/backoff.

INFORM cost:

```
overhead_inform = 2 packets/event + retry_factor
overhead_trap   = 1 packet/event
```

Use INFORM for high-importance, low-rate signals (chassis failure); use Trap for high-rate, individually-disposable signals (link flaps on a 4096-port spine).

---

## 12. Modern SNMP Performance

Production benchmarks (2024-2025):

| Platform | Sustained OIDs/s | Peak OIDs/s | Notes |
|:---|---:|---:|:---|
| Cisco IOS XE 17.x | 800-1,500 | 3,000 | CPU-bound on RP |
| Cisco IOS XR 7.x | 1,000-2,500 | 5,000 | concurrent workers per LC |
| Juniper Junos 22.x | 2,000-5,000 | 10,000 | mib2d scales horizontally |
| Arista EOS 4.x | 4,000-8,000 | 20,000 | snmpd on x86 control plane |
| Linux net-snmp 5.9 | 10,000-50,000 | 100,000 | CPU-bound on core polled |
| FRR + custom AgentX | 5,000-15,000 | 30,000 | sub-agent isolation |

Key bottlenecks:

1. **MIB walk dispatch on agent** — net-snmp's subtree linked-list scan dominates above 10k OIDs.
2. **TLS-style USM crypto** — SHA-256 + AES-128 adds 30-80 µs per PDU on a Cortex-A55 RP.
3. **Routing protocol locks** — BGP MIB walks require holding the BGP RIB read lock; long walks starve update workers (the canonical "BGP slows when monitoring spikes" failure).

Why streaming telemetry (gNMI dial-out, OpenConfig) is 10-100× faster:

- One TCP+gRPC connection persists for hours; no per-poll setup cost.
- Protobuf encoding is ~5× smaller than BER (see §14).
- Agent pushes only on change ("on-change") or sub-second cadence ("sample").
- No GETNEXT/GETBULK round trips.

### 12.1 Polling Intervals in Production

| Class | Interval | Examples |
|:---|---:|:---|
| Routine health | 60 s | sysUpTime, ifOperStatus, cpu%, memory% |
| Less critical | 300 s | storage usage, fan/PSU/temp environmentals |
| High-frequency | 5-15 s | BGP peer state on edge, ECMP balance, queue drops |
| Event-driven | trap-only | linkUp/Down, BFD up/down, ENVMON alarms |

A 60-second cadence on 50-100k counters is the boundary at which most teams add a streaming-telemetry sidecar.

---

## 13. SNMP-to-Prometheus Bridge Architecture

Prometheus' `snmp_exporter` translates SNMP responses into Prometheus samples. Pipeline:

```
+----------+   YAML   +---------------+  HTTP /metrics  +-----------+
| MIB Mods | -------> | snmp_exporter | <-------------- |Prometheus |
+----------+ via      +---------------+      scrape     +-----------+
             generator        │
                              │ SNMP UDP
                              ▼
                         +----------+
                         |  agent   |
                         +----------+
```

### 13.1 Generator Pre-compute

The generator (`generator.yaml` → `snmp.yml`) parses MIB modules at *build time* with libsmi/go-smi and emits:

- Numeric OIDs (avoiding runtime SMI parsing).
- Index decoding rules (e.g., split `ifEntry`'s IMPLIED INTEGER index from a sysName INDEX).
- Type and conversion hints (Counter32 → Prometheus counter; Gauge → gauge; TimeTicks → conversion to seconds).

This eliminates ~50 ms of per-scrape SMI work per module and yields deterministic OID set selection.

### 13.2 Latency Budget

```
T_scrape  = T_walk_all_modules + T_decode + T_format_metrics
T_budget  = scrape_interval - safety_margin

If T_walk > T_budget:   Prometheus marks the target down.
```

Concrete: 30-second Prometheus scrape interval, 5-second safety margin → walk + render must finish in 25 seconds. For 5,000 OIDs at agent peak 1,500 OIDs/s, the walk alone is 3.3 s — fine. At 50,000 OIDs, you would spread modules across multiple targets or move to a streaming source.

---

## 14. Comparison: SNMP vs gNMI vs NETCONF

| Property | SNMP | NETCONF | gNMI |
|:---|:---|:---|:---|
| Transport | UDP | TCP + SSH (port 830) | TCP + gRPC + TLS (port 9339) |
| Encoding | BER (binary) | XML (text) | Protobuf (binary) |
| Direction | Manager pulls; agent traps | Manager pulls; subscriptions optional | Pull + Push, Subscribe |
| Reliability | UDP loss possible | TCP reliable | TCP reliable |
| Schema | SMIv2 (.mib) | YANG | YANG |
| Streaming telemetry | weak (traps only) | RFC 5277 notifications | first-class subscribe |
| Typical bandwidth/OID | 50 B | 200-500 B (XML verbose) | 10 B (protobuf compact) |

Bandwidth efficiency (uncompressed, on-wire):

```
SNMP:   varbind ≈ 4 (SEQ tag/len) + len(OID) + 4 + len(value) ≈ 50 B
gNMI:   path proto + value proto                              ≈ 10 B
ratio:  5×

After gzip/snappy, gNMI compresses better because the Protobuf field tags repeat,
giving a further 2-3× saving.
```

Latency to first byte:

```
SNMP poll:           1 RTT, no setup
NETCONF pull:        SSH + RPC = 2-3 RTT setup amortized; then 1 RTT/op
gNMI subscribe:      1 RTT setup, then push (no RTT on update)
```

---

## 15. Failure Mode Analysis

### 15.1 UDP Loss in WAN

Loss probability `p` per packet. For a poll cycle of `K` PDUs:

```
P(success in single try) = (1 - p)^K
```

At `p = 0.02, K = 4`: `(0.98)^4 = 0.922` — 7.8% of polls will retransmit at least one PDU. With a 3-attempt retry budget and 5 s timeout, a long fiber cut producing `p ≈ 0.5` triggers a synchronized retry storm exactly when bandwidth is scarce.

### 15.2 Agent CPU Exhaustion → Cascade

Agent overload feedback loop:

```
   high poll rate ─┬─> agent CPU saturated
                   │           │
                   │           ▼
                   │      slow responses
                   │           │
                   │           ▼
                   ▲    poller times out
                   │           │
                   │           ▼
                   └──── poller retries (more load)
```

Mitigation:

- Per-engine concurrency caps in agent (`snmpd.conf agentSizeLimit`, vendor-specific QoS).
- Manager-side circuit breaker: if `failure_rate > 30%` over 1 minute, double the polling interval temporarily.

### 15.3 Trap Loss

Trap is UDP. Receiver ingestion above queue depth = silent drop. Solutions:

- INFORM with retry/ack on critical events.
- Dual receivers with anycast; idempotent event id.
- Rate-limit at the agent (`snmp-server enable traps mac-notification` rate limits, etc.) — accept degraded resolution rather than burst-driven loss.

### 15.4 Engine ID Collisions

Default engine IDs are derived from MAC/IP; cloned VMs can produce identical engine IDs, breaking time-window security and replay protection. Best practice: include a 16-byte random suffix at first boot, persisted to non-volatile storage.

### 15.5 Clock Drift

`notInTimeWindows` errors = SNMPv3 fails closed. After a long power-off or NTP outage, a manager's cached `engineBoots/engineTime` no longer matches; a discovery exchange re-syncs. Operational impact: a fresh outage produces a `usmStatsNotInTimeWindows` Report, not a Response — interpret correctly to avoid masking auth failures.

---

## 16. Best-Practice Math

### 16.1 Concurrency Sizing

```
total_polls/sec   = D × O / (T × poll_efficiency)
worker_count      = ⌈ polls/sec × mean_latency_seconds ⌉
recommended       = worker_count × 2   (P99 headroom)
```

`poll_efficiency` accounts for TCP-style backoff, GETBULK overhead, and inter-batch idle: 0.6 - 0.8 in practice.

### 16.2 Optimal max-repetitions

A heuristic that hits MTU and target subtree size:

```
M_opt = floor( min( (MTU - overhead) / b , target_walk_size ) )
```

With MTU = 1500, overhead = 100 B, b = 50 B, target = 256:

```
M_opt = floor( min( 1400/50 , 256 ) ) = min(28, 256) = 28
```

### 16.3 Trap Receiver Sizing

```
Sustained trap rate = peak_event_rate × fan_out_per_event
Receiver QPS        = parallel_workers / mean_processing_seconds
Queue size          ≥ peak_event_rate × storm_duration_seconds
```

Example: 200-rack DC, 1 outage event = 200 traps fan-out × 5 traps/device = 1,000 trap burst over ~3 s. Single threaded receiver at 10 ms/trap = 100 QPS. Queue must hold 1,000 traps; 32-worker pool sustains 3,200 QPS.

---

## 17. Tools for Wire-Level Analysis

| Tool | Purpose | Idiom |
|:---|:---|:---|
| `wireshark` | GUI BER decoder | display filter `snmp` |
| `tshark` | CLI Wireshark | `tshark -i any -d udp.port==161,snmp -Y snmp` |
| `tcpdump` | raw capture | `tcpdump -i any -n -w snmp.pcap "udp port 161 or udp port 162"` |
| `snmpwalk` | active probe | `snmpwalk -v2c -c public host system` |
| `snmpbulkwalk` | GETBULK probe | `snmpbulkwalk -v2c -c public -Cr50 host ifTable` |
| `snmpget` | single OID | `snmpget -v3 -l authPriv -u u -a SHA -A pw -x AES -X pw host sysUpTime.0` |
| `snmptrap` | send a trap | `snmptrap -v2c -c public host '' enterprises.foo` |
| `snmpdf`, `snmpnetstat`, `snmpstatus` | net-snmp utilities | high-level views |
| `mib2c` | code generation | `mib2c -c mib2c.scalar.conf MY-MIB::myObj` |
| `asn1c` | offline ASN.1 compiler | round-trip BER samples |
| `smilint`, `smistrip` | MIB validation | catches MIB authoring errors |

`tshark` decoding example:

```
$ tshark -O snmp -V -r snmp.pcap | head -40
Frame 1: 92 bytes on wire ...
Internet Protocol Version 4, Src: 10.0.0.1, Dst: 10.0.0.2
User Datagram Protocol, Src Port: 50001, Dst Port: 161
Simple Network Management Protocol
    version: v2c (1)
    community: public
    data: get-request (0)
        get-request
            request-id: 1234
            error-status: noError (0)
            error-index: 0
            variable-bindings: 1 item
                1.3.6.1.2.1.1.3.0: Value (Null)
                    Object Name: 1.3.6.1.2.1.1.3.0 (iso.3.6.1.2.1.1.3.0)
                    Value (Null)
```

Useful one-liners:

```
# Hex dump matching SNMP traffic for offline ASN.1 work
tcpdump -i any -nn -X -s0 'udp port 161' | head -50

# Full OID-to-name in walks
snmpwalk -v2c -c public -ObentU host 1.3.6.1.2.1.1

# Time a bulk walk to estimate sustained OIDs/s
time snmpbulkwalk -v2c -c public -Cr50 host 1.3.6.1.2.1.2.2 | wc -l
```

---

## 18. Counter Math — Rate Derivation (carry-over)

### 18.1 Rate Formula

SNMP counters (ifInOctets, ifOutOctets) are cumulative. To compute a rate:

```
Rate = (C_t2 - C_t1) / (t2 - t1)
```

### 18.2 Counter32 Wrap Time

```
T_wrap32 = 2^32 / R   (seconds)
```

| Speed (b/s) | Bytes/s | T_wrap32 |
|---:|---:|---:|
| 10 Mb/s | 1.25 × 10^6 | 57.3 min |
| 100 Mb/s | 1.25 × 10^7 | 5.73 min |
| 1 Gb/s | 1.25 × 10^8 | 34.4 s |
| 10 Gb/s | 1.25 × 10^9 | 3.43 s |
| 25 Gb/s | 3.125 × 10^9 | 1.37 s |
| 100 Gb/s | 1.25 × 10^10 | 0.34 s |

Above ~1 Gb/s, polling every 60 s misses wraps repeatedly.

### 18.3 Counter64 Wrap Time

```
T_wrap64 = 2^64 / R
```

At 10 Gb/s: `2^64 / 1.25e9 ≈ 1.47 × 10^10 s ≈ 467 years`. Counter64 (HC counters) is mandatory for ≥ 1 Gb/s interfaces.

### 18.4 Wrap Detection

```
if C_t2 < C_t1:
    delta_C = (2^B - C_t1) + C_t2
else:
    delta_C = C_t2 - C_t1
```

Ambiguity: if more than one wrap occurred between samples, the math undercounts. The only safe rule is to **poll faster than `T_wrap`**, or use Counter64.

---

## 19. Trap Storm — Queueing Analysis (carry-over)

### 19.1 Storm Rate Model

```
R_storm = N_devices × T_per_device / Δt
```

| Devices | Traps each | Δt | R_storm |
|---:|---:|---:|---:|
| 100 | 5 | 10 s | 50/s |
| 500 | 10 | 5 s | 1,000/s |
| 1,000 | 20 | 2 s | 10,000/s |

### 19.2 Receiver Capacity

```
C_recv = N_threads / T_process
```

10 threads × 1 ms each = 10,000 traps/s sustained.

### 19.3 Time to Queue Overflow

```
T_overflow = Q_max / (R_storm - C_recv)         (when R_storm > C_recv)
```

Q_max = 10,000 trap slots, R_storm = 20,000/s, C_recv = 10,000/s → overflow in 1 s. After overflow, UDP traps are dropped silently. INFORM is the structural fix because the agent will retry.

---

## 20. SNMPv3 Discovery Exchange

A manager that has never spoken to an engine must discover the engine's `engineID`, `engineBoots`, and `engineTime`:

```
Manager                                    Agent (engine E)

    +---- Get Request, msgFlags=reportable ---->
    |    msgAuthEngineID = 0x00 (empty)
    |    securityLevel = noAuthNoPriv
    |
    |<--- Report PDU --------------------------+
         msgAuthEngineID = E
         msgAuthEngineBoots = b
         msgAuthEngineTime = t
         varbind: usmStatsUnknownEngineIDs

    Manager caches (E, b, t, t_local), then proceeds with authPriv.
```

If the manager's cached `(b, t)` is stale, the agent answers with `usmStatsNotInTimeWindows` and the manager re-discovers. Two RTTs added to startup; zero RTTs at steady state.

---

## 21. AgentX (RFC 2741) — Subagent Architecture

Large monolithic agents (net-snmp on a router with 100+ MIBs) split functionality into:

- **Master agent** — owns UDP/161, performs ASN.1, USM, VACM, dispatch.
- **Subagents** — loadable modules (UNIX domain socket or TCP) registering OID subtrees.

Communication uses AgentX PDUs (a separate ASN.1 schema). Operational implications:

- Subagent crash → master returns `genErr` for that subtree, not a global outage.
- Per-subagent timeouts (`agentXTimeout`) bound master-side latency.
- Watchdog re-registers after subagent restart.

```
+------------------+
|  Master agent    |
|  (port 161)      |
+--------┬---------+
         │ AgentX
   ┌─────┼─────┬─────┐
   ▼     ▼     ▼     ▼
ifMib  bgpMib qosMib chassisMib   (subagents)
```

---

## 22. Bulk Counter Math at Scale

Consider a 100-port leaf with HC counters polled every 30 s:

```
OIDs/poll = 100 × {ifHCInOctets, ifHCOutOctets, ifHCInUcastPkts,
                   ifHCOutUcastPkts, ifInDiscards, ifOutDiscards}
          = 600 OIDs

GETBULK with M=30  →  ⌈600/30⌉ = 20 PDUs round trips per scrape
                      20 × 1 ms agent + 20 × 0.5 ms wire = ~30 ms total
```

Across 1,000 leaves: 1,000 × 600 = 600,000 OIDs every 30 s → 20,000 OIDs/s sustained — a small Prometheus + snmp_exporter footprint, comfortably below most agents' peak.

---

## 23. ASN.1 Encoding — Larger Worked Example

Encode a complete `GetRequest-PDU` for `sysUpTime.0` with community `public`:

```
Build inner VarBind first:
  OID name:  06 08 2B 06 01 02 01 01 03 00          (10 bytes)
  Value:     05 00                                  (NULL placeholder, 2 bytes)
  VarBind = 30 0C 06 08 2B 06 01 02 01 01 03 00 05 00     (14 bytes)

VarBindList SEQUENCE OF VarBind = 30 0E (VarBind)         (16 bytes)

PDU body:
  request-id 0x12345678  = 02 04 12 34 56 78        (6 bytes)
  error-status 0          = 02 01 00                 (3 bytes)
  error-index 0           = 02 01 00                 (3 bytes)
  varbinds                = 30 10 30 0E ...          (18 bytes)

GetRequest-PDU [0] IMPLICIT SEQUENCE:
  Tag = A0 (context-specific 0, constructed)
  Inner = (request-id) || (err-status) || (err-idx) || (varbinds)
        = 6 + 3 + 3 + 18 = 30 bytes
  PDU = A0 1E 02 04 12 34 56 78 02 01 00 02 01 00 30 10 30 0E ...

Message:
  version v2c = 02 01 01                            (3 bytes)
  community "public" = 04 06 70 75 62 6C 69 63      (8 bytes)
  data = PDU (32 bytes)

Outer SEQUENCE = 30 LEN ... where LEN = 3 + 8 + 32 = 43 = 0x2B
   Total: 30 2B 02 01 01 04 06 70 75 62 6C 69 63 A0 1E ...

Total bytes on the wire: 45 (1 tag + 1 length + 43 content).
Plus IP (20) + UDP (8) = 73-byte UDP datagram.
```

For a Counter32 response with value `12345`:

```
Replace value placeholder with Counter32 = [APPLICATION 1] = 0x41
  41 02 30 39          (4 bytes)
VarBind: 30 0E 06 08 2B 06 01 02 01 01 03 00 41 02 30 39   (16 bytes)
```

The response is 47 bytes total — barely larger than the request. This per-OID economy is what makes SNMP feasible at scale on UDP, and what gNMI/Protobuf improves on with another factor of five.

---

## 24. Vendor-Specific PDU Limits and Tunables

| Vendor | Default PDU buffer | Tuning knob |
|:---|---:|:---|
| net-snmp 5.9 | 65,535 | `agentXSocket` / `pduBuffer` |
| Cisco IOS XE | 1472 | `snmp-server packetsize <bytes>` |
| Cisco IOS XR | 1500 | `snmp-server max-pdu-size` |
| Juniper Junos | 1500 | `snmp pdu-size <bytes>` |
| Arista EOS | 65,507 | `snmp-server max-engine-size` |
| HPE / Aruba | 1500 | `snmp-server max-message-size` |

Bumping past 1500 only helps on jumbo-MTU paths or on local Unix sockets to subagents.

---

## 25. Telemetry Migration Sketch

A practical migration from SNMP to gNMI streaming for a fleet of 10,000 routers:

```
Phase 0:   Inventory MIBs in active use; eliminate dead OIDs.
Phase 1:   Stand up gnmi_collector with same sample paths (e.g., openconfig-interfaces).
Phase 2:   Run dual-write: snmp_exporter for legacy dashboards, gNMI for low-latency alerting.
Phase 3:   Cut alerting to gNMI; keep SNMP polls at 5 min for compliance and traps.
Phase 4:   Decommission SNMP polling fleet; keep snmptrapd for legacy events that lack a YANG model.
```

Math: gNMI dial-in/dial-out reduces aggregated bandwidth from ~13 Mbps (50k devices × 200 OIDs / 60 s × 100 B) to ~3 Mbps (Protobuf + dedup), and tail latency from up to 60 s (next poll) to under 1 s (subscribe push).

---

## 25.1 Coexistence Math (RFC 3584)

Operators run heterogeneous fleets where v1, v2c, and v3 traffic coexists. RFC 3584 defines bidirectional translations a proxy must implement.

### v1 → v2c translation rules

- v1 generic-trap N (0..6) → v2c snmpTrapOID `1.3.6.1.6.3.1.1.5.(N+1)` if N < 6, else combine `enterprise.0.specific-trap`.
- v1 enterprise OID copies into v2c `varbind[snmpTrapEnterprise]`.
- v1 agent-addr → v2c `varbind[snmpTrapAddress]` (`1.3.6.1.6.3.18.1.3.0`).
- v1 community appears unchanged on the v2c side.

### v2c → v1 translation rules

- v2c `Counter64` MUST be dropped (v1 has no Counter64). Some proxies fall back to "wrap to Counter32" — only safe below 4 Gb of cumulative count.
- v2c `error-status` codes 6-18 collapse to v1 `genErr(5)` (with translation table for SDK debugging).
- v2c GETBULK becomes a series of v1 GETNEXT calls, multiplying RTT by `M`.

### Translation cost

```
T_proxy = T_decode_v1 + T_walk_translation + T_encode_v2 + 2 * T_wire
```

A proxy's CPU per translated PDU is ~5-10 µs on x86; the latency hit is dominated by wire round trips when GETBULK degrades into N GETNEXT.

---

## 25.2 SNMP over IPv6

Transport mapping for IPv6 is RFC 3417 / RFC 3411. UDP/161 and UDP/162 unchanged. PDU layout unchanged. Differences:

- `snmpTargetAddrTAddress` is 18 octets: 16 bytes IPv6 + 2 bytes port (big-endian).
- IPv6 minimum MTU is 1280 — `max-repetitions` should be sized to a 1252-byte ceiling (1280 - 20 IPv6 - 8 UDP).
- `IpAddress` SNMP type is **still 4 octets** — for IPv6 you must use `InetAddress` from RFC 4001.

Encoding `InetAddressIPv6` for `2001:db8::1` is an OCTET STRING of 16 bytes:

```
04 10 20 01 0D B8 00 00 00 00 00 00 00 00 00 00 00 01
```

The associated `InetAddressType` enum (1=ipv4, 2=ipv6, 3=ipv4z, 4=ipv6z, 16=dns) is sent as a separate INTEGER varbind to disambiguate.

---

## 25.3 Notification De-duplication Math

A receiver sees the same trap from redundant agents (HSRP/VRRP active-active, NSO, dual supervisors). De-duplication key:

```
hash = H( engineID ‖ snmpTrapOID.0 ‖ canonical(varbinds) ‖ floor(sysUpTime.0 / window) )
```

`window` rounds sysUpTime to (e.g.) 5 seconds, making bursts within the window collapse to a single event. False-positive rate (two distinct events colliding):

```
P(collision) ≈ N_events / 2^hash_bits
```

For 1 million events/day and 64-bit hash: `P ≈ 5.4e-14` per pair, negligible.

---

## 25.4 Bulk-Update Window Math

When a manager periodically writes configuration via SET (e.g., snmp-server location strings, ACL pushes), there is a serialization concern: SET PDUs are atomic per PDU but not across PDUs. To push N changes atomically across a fleet:

```
T_window = N_devices × T_per_set / parallelism
```

If the change is non-idempotent (sequence-dependent), the manager must:

1. Acquire a soft lock via `snmpSetSerialNo.0` (RFC 1907) — read it, increment, write it back; collisions retry.
2. Apply the SET.
3. Optionally read back to confirm.

This costs 3 RTTs per device but makes concurrent writers safe.

---

## 25.5 Persistence and Engine Boots

`engineBoots` MUST be persisted across reboots and incremented exactly once per boot. Failure modes:

- **Lost engineBoots** (e.g., dataless container restart) → managers reject with `usmStatsNotInTimeWindows` until re-discovery.
- **Repeated engineBoots** (e.g., snapshotted VM rolled back) → replay window opens; an attacker who captured a previous PDU could replay it for the duration of the time-window.

Recommendation: store `engineBoots` in a write-once-per-boot file with `fsync` before any authPriv traffic; ship snapshots without the file so clones bootstrap fresh.

---

## 25.6 MIB Module Authoring Math

Every OBJECT-TYPE in a MIB consumes a numeric subidentifier. A vendor under PEN `1.3.6.1.4.1.X` typically allocates:

```
.X.1   products          (chassis identity OIDs)
.X.2   features          (feature MIBs)
.X.3   protocols         (protocol-specific objects)
.X.4   notifications     (trap definitions)
.X.5   experimental      (pre-RFC modules)
.X.6   example/test
```

A table OID layout:

```
fooTable        1.3.6.1.4.1.X.2.1            (table)
  fooEntry      1.3.6.1.4.1.X.2.1.1          (row)
    fooIndex    1.3.6.1.4.1.X.2.1.1.1        (column 1, the INDEX)
    fooName     1.3.6.1.4.1.X.2.1.1.2        (column 2)
    fooStatus   1.3.6.1.4.1.X.2.1.1.3        (column 3, RowStatus)
```

A row instance OID is `<column>.<index>`; for `fooName.42`: `1.3.6.1.4.1.X.2.1.1.2.42`. Conversion to Counter64 indexed by sysName means the index encodes a string of length L: `L. char1. char2. ... charL` with each char as its decimal subidentifier.

---

## 25.7 Cardinality and Memory Math at the Agent

For an agent storing `N` registered OIDs each with `m` bytes of metadata + `v` bytes of value:

```
Memory_node ≈ N × (m + v + tree_pointers)
```

net-snmp's per-OID overhead is ~200 bytes plus value; 50,000 OIDs ≈ 10 MB resident. On routers with 50,000-port chassis, ifTable alone is 50,000 rows × 22 columns × 50 B ≈ 55 MB — non-trivial on small RPs. Solution: paginate the table behind AgentX subagents that lazily materialize rows.

---

## 25.8 Wireshark Decode Cheat Code

```
Display filter examples:
  snmp                                 -- any SNMP
  snmp.community == "public"           -- legacy traffic only
  snmp.version == 3                    -- v3 only
  snmp.msgUserName == "monitor"        -- per-user filter
  snmp.errorStatus != 0                -- failed responses
  snmp.NoSuchObject || snmp.NoSuchInstance || snmp.endOfMibView
                                       -- WALK termination markers
  snmp.variable_bindings.name contains "1.3.6.1.4.1.9"
                                       -- Cisco enterprise
```

Decryption: in Wireshark go to *Edit → Preferences → Protocols → SNMP → Users Table*, enter `engine_id`, `username`, auth/priv credentials. Wireshark recomputes K_localized and decrypts in place.

---

## 25.9 Bench Numbers — Per-Operation Wire Sizes

Empirical measurements from net-snmp 5.9 against an Arista 7050 (loopback, no loss):

| Operation | Request bytes | Response bytes | Notes |
|:---|---:|---:|:---|
| GET sysUpTime.0 | 45 | 47 | minimal exchange |
| GETNEXT in ifTable | 47 | 88 | one ifEntry column |
| GETBULK M=10, ifTable | 49 | 720 | 10 columns returned |
| GETBULK M=50, ifTable | 49 | 3,200 | spills to two IP fragments at MTU 1500 |
| Trap (linkDown) | 110 | 0 | unacknowledged |
| INFORM (linkDown) | 110 | 110 | acknowledged with Response |
| Authenticated GET (SHA-256) | 125 | 127 | +HMAC + USM headers |
| authPriv GET (SHA-256/AES-128) | 141 | 143 | +AES padding rounded to 16 B |

Per-operation crypto cost (Cortex-A55 1.4 GHz):

| Step | Cycles | Time |
|:---|---:|---:|
| BER decode of 100-byte message | ~3,000 | 2.1 µs |
| HMAC-SHA-256 over 100 bytes | ~6,500 | 4.6 µs |
| AES-128-CFB decrypt 100 bytes | ~2,400 | 1.7 µs |
| VACM lookup (in-cache) | ~1,800 | 1.3 µs |
| BER encode response | ~3,500 | 2.5 µs |
| **Total per authPriv GET** | **~17,200** | **~12 µs** |

So a 1.4 GHz embedded RP can sustain ~80,000 authPriv GETs/s in pure crypto, before any MIB-handler dispatch — proving the bottleneck is rarely the crypto itself but the agent's tree traversal and serialization of vendor data.

---

## 26. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| `tag_byte = (class<<6) | (P/C<<5) | tag_num` | Bit packing | BER encoding |
| `len_short / len_long_form` | Variable-length integer | BER lengths |
| `K_loc = H(Ku ‖ E ‖ Ku)` | Hash composition | USM key localization |
| `tag = trunc_n(HMAC(K_auth, msg))` | HMAC + truncation | USM auth |
| `IV = boots ‖ time ‖ salt` | Concatenation | AES-CFB IV (RFC 3826) |
| `|Δt| ≤ 150 s` | Inequality | USM time-window check |
| `BW = D · O · b · 8 / T` | Linear model | Polling bandwidth |
| `RTTs = ⌈N / M⌉` | Ceiling | GETBULK round trips |
| `C_min = ⌈polls/s · L⌉` | Rate × latency | Worker pool sizing |
| `T_wrap = 2^B / R` | Division | Counter wrap time |
| `T_overflow = Q / (R - C)` | Queue model | Trap storm survival |
| `M_opt = ⌊(MTU - overhead) / b⌋` | MTU-bounded | GETBULK tuning |

---

## 27. References

- RFC 1155 — Structure and Identification of Management Information (SMIv1)
- RFC 1156 — MIB-I
- RFC 1157 — Simple Network Management Protocol (SNMPv1)
- RFC 1901-1908 — SNMPv2c
- RFC 2578-2580 — SMIv2 (SMI, Textual Conventions, Conformance)
- RFC 2741 — AgentX
- RFC 2790 — Host Resources MIB
- RFC 3410 — Introduction and Applicability Statement for Internet Standard SNMP Framework
- RFC 3411 — Architecture for SNMP Frameworks
- RFC 3412 — Message Processing and Dispatching
- RFC 3413 — SNMP Applications
- RFC 3414 — User-based Security Model (USM)
- RFC 3415 — View-based Access Control Model (VACM)
- RFC 3416 — Protocol Operations
- RFC 3417 — Transport Mappings
- RFC 3418 — MIB for SNMP itself
- RFC 3584 — Coexistence between SNMP versions
- RFC 3826 — AES Cipher Algorithm in USM
- RFC 4001 — Textual Conventions for Internet Network Addresses
- RFC 4133 — Entity MIB v3
- RFC 5277 — NETCONF Event Notifications (for comparison)
- RFC 6353 — TLS Transport Model for SNMP (rare deployment)
- RFC 7860 — HMAC-SHA-2 Authentication Protocols in USM
- ITU-T X.680 — Abstract Syntax Notation One (ASN.1)
- ITU-T X.690 — ASN.1 Encoding Rules: BER, CER, DER
- "Essential SNMP" — Mauro & Schmidt, O'Reilly (2nd ed.)
- "Understanding SNMP MIBs" — Perkins & McGinnis, Prentice Hall

## Prerequisites

- counter arithmetic, polling intervals, integer overflow
- HMAC and hash-function fundamentals
- ASN.1 / BER (see ITU-T X.680/X.690)
- UDP fragmentation behaviour and path MTU

---

*SNMP has monitored billions of network interfaces for three decades. The protocol's math — BER lengths, USM key derivation, polling-cycle bandwidth, GETBULK ceiling division, counter wraps, queueing during trap storms — determines whether your monitoring system catches a network outage in five seconds or becomes the post-incident root cause.*
