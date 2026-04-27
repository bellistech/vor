# The Mathematics of LDAP — Protocol Internals, ASN.1, Tree Algorithms

> *LDAP is a BER-encoded ASN.1 protocol layered atop TCP, modelling the Directory Information Tree as a rooted ordered graph. Every search is a filter algebra evaluated against a candidate set, every bind is a SASL negotiation, every replication event is a CSN-ordered conflict-resolved write. Understanding LDAP requires reasoning simultaneously about ASN.1 length encodings, BTree fan-out, SCRAM HMAC chaining, and CSN lex-order causality.*

---

## 1. ASN.1 / BER Encoding (RFC 4511 §5.1)

### Wire Format Foundations

LDAP messages are encoded using a restricted subset of the Basic Encoding Rules (BER) of ASN.1 (ITU-T X.690). Every message is a **type-length-value** (TLV) triple. The protocol explicitly excludes Distinguished Encoding Rules (DER) requirements — LDAP servers MUST accept any valid BER encoding even though most implementations emit canonical-form output.

The TLV grammar:

```
encoding ::= identifier-octet+ length-octet+ contents-octet*
```

### Identifier Octet Layout

```
Bit:    8  7  6  5  4  3  2  1
       +--+--+--+--+--+--+--+--+
       |Class|P/C|     Tag      |
       +--+--+--+--+--+--+--+--+
```

| Field | Bits | Meaning |
|:---|:---:|:---|
| Class | 8-7 | 00=Universal, 01=Application, 10=Context-specific, 11=Private |
| P/C | 6 | 0=Primitive, 1=Constructed |
| Tag | 5-1 | Tag number (0-30); 31 = long-form follows |

For tag values $\geq 31$, bits 5-1 are all 1, and subsequent octets carry the tag in big-endian base-128, with bit 8 set on continuation octets and clear on the final octet.

### Length Encoding

Three forms exist: short, long-definite, indefinite.

**Short form** (length < 128):

```
+--+--+--+--+--+--+--+--+
| 0|       length        |
+--+--+--+--+--+--+--+--+
```

**Long-definite form** (length ≥ 128):

```
+--+--+--+--+--+--+--+--+
| 1|     N (1-126)       |   ← N = number of subsequent length octets
+--+--+--+--+--+--+--+--+
| length octet 1         |
+--+--+--+--+--+--+--+--+
| ...                    |
+--+--+--+--+--+--+--+--+
| length octet N         |
+--+--+--+--+--+--+--+--+
```

**Indefinite form** (constructed only): identifier `0x80`, terminated by end-of-contents `0x00 0x00`. RFC 4511 forbids this for LDAP.

### LDAPMessage ASN.1 Definition (RFC 4511 §4.1.1)

```asn1
LDAPMessage ::= SEQUENCE {
    messageID     MessageID,
    protocolOp    CHOICE {
        bindRequest        BindRequest,
        bindResponse       BindResponse,
        unbindRequest      UnbindRequest,
        searchRequest      SearchRequest,
        searchResEntry     SearchResultEntry,
        searchResDone      SearchResultDone,
        searchResRef       SearchResultReference,
        modifyRequest      ModifyRequest,
        modifyResponse     ModifyResponse,
        addRequest         AddRequest,
        addResponse        AddResponse,
        delRequest         DelRequest,
        delResponse        DelResponse,
        modDNRequest       ModifyDNRequest,
        modDNResponse      ModifyDNResponse,
        compareRequest     CompareRequest,
        compareResponse    CompareResponse,
        abandonRequest     AbandonRequest,
        extendedReq        ExtendedRequest,
        extendedResp       ExtendedResponse,
        intermediateResp   IntermediateResponse },
    controls      [0] Controls OPTIONAL }

MessageID ::= INTEGER (0 .. maxInt)
maxInt ::= INTEGER 2147483647
```

The `protocolOp` is a CHOICE — exactly one of the listed alternatives. Each alternative carries its own application-class tag (BindRequest = `[APPLICATION 0]`, BindResponse = `[APPLICATION 1]`, etc.). This eliminates the need for a discriminator field in the SEQUENCE.

### Application Tag Assignment

| Tag | Operation |
|:---:|:---|
| 0  | BindRequest |
| 1  | BindResponse |
| 2  | UnbindRequest |
| 3  | SearchRequest |
| 4  | SearchResultEntry |
| 5  | SearchResultDone |
| 6  | ModifyRequest |
| 7  | ModifyResponse |
| 8  | AddRequest |
| 9  | AddResponse |
| 10 | DelRequest |
| 11 | DelResponse |
| 12 | ModifyDNRequest |
| 13 | ModifyDNResponse |
| 14 | CompareRequest |
| 15 | CompareResponse |
| 16 | AbandonRequest |
| 19 | SearchResultReference |
| 23 | ExtendedRequest |
| 24 | ExtendedResponse |
| 25 | IntermediateResponse |

### Worked Example: Encoding `BindRequest(version=3, name="cn=admin", simple="secret")`

The ASN.1 definition:

```asn1
BindRequest ::= [APPLICATION 0] SEQUENCE {
    version       INTEGER (1 .. 127),
    name          LDAPDN,
    authentication AuthenticationChoice }

AuthenticationChoice ::= CHOICE {
    simple        [0] OCTET STRING,
    sasl          [3] SaslCredentials,
    ... }
```

Step-by-step encoding with messageID = 1:

**Inner BindRequest contents:**

1. `version = 3`:
   - Tag: `0x02` (Universal, primitive, INTEGER)
   - Length: `0x01`
   - Value: `0x03`
   - Octets: `02 01 03`

2. `name = "cn=admin"` (8 ASCII chars):
   - Tag: `0x04` (Universal, primitive, OCTET STRING)
   - Length: `0x08`
   - Value: `63 6E 3D 61 64 6D 69 6E`
   - Octets: `04 08 63 6E 3D 61 64 6D 69 6E`

3. `authentication = simple "secret"` (CHOICE selects `[0]`):
   - Tag: `0x80` (Context-specific class=10, primitive, tag 0 → `0b10000000`)
   - Length: `0x06`
   - Value: `73 65 63 72 65 74`
   - Octets: `80 06 73 65 63 72 65 74`

**BindRequest wrapper** (`[APPLICATION 0]`, constructed):
- Tag: `0x60` (Application class=01, constructed=1, tag 0 → `0b01100000`)
- Length: contents = 3 + 10 + 8 = 21 bytes → `0x15`
- Octets: `60 15 ...`

**LDAPMessage SEQUENCE:**
- messageID = 1: `02 01 01`
- protocolOp = above BindRequest: 23 bytes
- Total contents: 3 + 23 = 26 bytes
- Tag: `0x30` (Universal, constructed, SEQUENCE)
- Length: `0x1A` (26)

### Final Wire Bytes

```
30 1A                           ; SEQUENCE, len=26
   02 01 01                     ;   messageID = 1
   60 15                        ;   [APPLICATION 0] BindRequest, len=21
      02 01 03                  ;     version = 3
      04 08 63 6E 3D 61 64 6D 69 6E  ;     name = "cn=admin"
      80 06 73 65 63 72 65 74   ;     simple = "secret"
```

Hex stream: `30 1A 02 01 01 60 15 02 01 03 04 08 63 6E 3D 61 64 6D 69 6E 80 06 73 65 63 72 65 74`

Total: 28 bytes on the wire. The 6-byte password is transmitted in cleartext absent TLS.

### Length Encoding Edge Case

A SearchResultEntry returning 1024 attribute values might exceed 65535 bytes. Encoding length 100000:

- Long form, 3 length octets needed (100000 fits in 17 bits)
- First octet: `0x83` (`1` + `0000011`)
- Subsequent: `0x01 0x86 0xA0` (big-endian 100000)
- Full length prefix: `83 01 86 A0`

Maximum LDAP message size is implementation-defined; OpenLDAP defaults to `sockbuf_max_incoming = 262143` (2^18 − 1) for anonymous and `4194303` (2^22 − 1) for authenticated.

---

## 2. Search Filter Algebra (RFC 4515)

### Filter Grammar (RFC 4515 §3, ABNF)

```abnf
filter         = LPAREN filtercomp RPAREN
filtercomp     = and / or / not / item
and            = AMPERSAND filterlist
or             = VERTBAR  filterlist
not            = EXCLAMATION filter
filterlist     = 1*filter
item           = simple / present / substring / extensible
simple         = attr filtertype assertionvalue
filtertype     = equal / approx / greaterorequal / lessorequal
equal          = EQUALS
approx         = TILDE EQUALS
greaterorequal = RANGLE EQUALS
lessorequal    = LANGLE EQUALS
present        = attr EQUALS ASTERISK
substring      = attr EQUALS [initial] any [final]
initial        = assertionvalue
any            = ASTERISK *(assertionvalue ASTERISK)
final          = assertionvalue
extensible     = ( attr [dnattrs] [matchingrule] COLON EQUALS assertionvalue )
               / ( [dnattrs] matchingrule COLON EQUALS assertionvalue )
dnattrs        = COLON "dn"
matchingrule   = COLON oid
```

### Filter as Abstract Syntax Tree

A filter parses to a tree where internal nodes are AND/OR/NOT and leaves are atomic predicates (equal, present, substring, ge, le, approx, extensible). The root is always either an internal Boolean node or a single leaf.

Example: `(&(objectClass=person)(|(uid=alice)(uid=bob))(!(accountLocked=TRUE)))`

```
                    AND
            ┌────────┼────────┐
            │        │        │
     (=                  OR              NOT
      objectClass    ┌────┴────┐         │
      person)        =         =         =
                     uid       uid       accountLocked
                     alice     bob       TRUE
```

### Filter Evaluation Algorithm

```
EVAL(filter, entry):
  switch filter.kind:
    case AND:
      for child in filter.children:
        r = EVAL(child, entry)
        if r is FALSE:    return FALSE     // short-circuit
        if r is UNDEFINED: undef = true
      return UNDEFINED if undef else TRUE
    case OR:
      for child in filter.children:
        r = EVAL(child, entry)
        if r is TRUE:     return TRUE      // short-circuit
        if r is UNDEFINED: undef = true
      return UNDEFINED if undef else FALSE
    case NOT:
      r = EVAL(filter.child, entry)
      return TRUE if r=FALSE else FALSE if r=TRUE else UNDEFINED
    case EQUAL:
      values = entry[filter.attr]
      if values is None: return UNDEFINED        // attribute absent
      return TRUE if filter.value in values else FALSE
    case PRESENT:
      return TRUE if filter.attr in entry else FALSE
    case SUBSTRING:
      values = entry[filter.attr]
      if values is None: return UNDEFINED
      for v in values:
        if MATCH_SUBSTR(v, filter.initial, filter.any, filter.final):
          return TRUE
      return FALSE
    case GE / LE:
      values = entry[filter.attr]
      if values is None: return UNDEFINED
      return TRUE if any(v >= filter.value) for GE else any(v <= filter.value)
    case APPROX:
      values = entry[filter.attr]
      return TRUE if any soundex(v) == soundex(filter.value)
    case EXTENSIBLE:
      apply matching rule per RFC 4517
```

The three-valued logic (TRUE, FALSE, UNDEFINED) propagates per RFC 4511 §4.5.1.7. UNDEFINED arises when an attribute is absent or when a matching rule cannot be applied.

### Filter Selectivity Algebra

For independent leaf selectivities $s_i = \Pr[\text{entry matches leaf}_i]$:

$$s_{AND} = \prod_{i=1}^{k} s_i$$

$$s_{OR} = 1 - \prod_{i=1}^{k} (1 - s_i)$$

$$s_{NOT} = 1 - s$$

The independence assumption fails when filters reference correlated attributes (e.g., `(objectClass=person)` and `(givenName=*)` are highly correlated). Real selectivity statistics require histogram tracking per attribute.

### Substring Filter Cost

A substring filter `(cn=*smith*)` decomposes into three sub-patterns: `initial`, `any[]`, `final`. Index strategies:

| Pattern | Index Used | Cost |
|:---|:---|:---|
| `(cn=smith*)` | sub.initial BTree prefix scan | $O(\log N + R)$ |
| `(cn=*smith)` | sub.final BTree prefix scan on reversed string | $O(\log N + R)$ |
| `(cn=*smith*)` | sub.any inverted index (3-grams) | $O(K \cdot \log N)$ where $K$ = trigram count |
| `(cn=*sm*th*)` | trigram intersection | $O(K \cdot \log N + R)$ |

OpenLDAP `slapd-mdb` builds three sub-indexes: `subinitial`, `subany`, `subfinal`. The `subany` index uses fixed-length n-grams (default 3); patterns shorter than n disable the index.

### Worst-Case Filter Complexity

Filter $F$ over $N$ candidates with $L$ leaves and $D$ tree depth:

- Best (root AND with selective indexed first leaf): $O(\log N + R)$ where $R$ = result count
- Worst (no usable index, full subtree scan): $O(N \cdot L)$ — every entry tests every leaf

---

## 3. The DIT as a Tree

### Distinguished Names as Paths

A DN is an ordered sequence of RDNs (Relative Distinguished Names). RFC 4514 grammar:

```abnf
distinguishedName = [ relativeDistinguishedName *( COMMA relativeDistinguishedName ) ]
relativeDistinguishedName = attributeTypeAndValue *( PLUS attributeTypeAndValue )
attributeTypeAndValue     = attributeType EQUALS attributeValue
```

DN string `"uid=alice,ou=eng,dc=example,dc=com"` decomposes into 4 RDNs in left-to-right order, where the leftmost RDN is the entry's own RDN and successive RDNs walk towards the root. The root is the empty DN.

### Tree Operations

| Operation | Effect | Complexity |
|:---|:---|:---|
| Search BASE | Single entry lookup | $O(\log N)$ via DN index |
| Search ONE | List immediate children | $O(C)$ where $C$ = child count |
| Search SUB | Subtree walk | $O(K)$ where $K$ = subtree size |
| Add | Insert leaf entry | $O(\log N)$ |
| Delete | Remove leaf entry | $O(\log N)$ |
| ModifyDN (rename) | Atomic relocate, optionally with newSuperior | $O(\log N)$ if leaf, $O(K)$ if subtree |

ModifyDN with `newSuperior` and `deleteOldRDN=FALSE` requires an atomic re-parenting. OpenLDAP MDB performs this in a single transaction; consumers receive a delete + add via syncrepl unless the protocol carries the rename semantically.

### Tree Metrics

For a subtree rooted at entry $e$ with branching factor $b$ and depth $d$:

- Subtree size: $N_{sub} = \frac{b^{d+1} - 1}{b - 1}$
- Average path length to leaf: $d$
- Maximum DN component count: $d + r$ where $r$ = root depth from naming context

### ASCII DIT Layout

```
                       dc=com
                          │
                       dc=example
                  ┌───────┼────────┐
                ou=eng  ou=ops   ou=hr
                  │       │        │
               ┌──┴──┐  ┌─┴─┐    ┌─┴─┐
            uid=A uid=B uid=C uid=D uid=E uid=F

Depth d=4 from root
Branching at ou level: 3
Branching at uid level: 2
```

A SUB search at `dc=example` walks 9 entries. A SUB search at `ou=eng` walks 3 entries. The reduction is $9/3 = 3\times$ — exactly the inverse of the branching factor at the level skipped.

---

## 4. SASL Authentication Mechanisms

### Mechanism Inventory (RFC 4422)

| Mechanism | RFC | Plaintext Exposure | Mutual Auth | Channel Binding |
|:---|:---|:---:|:---:|:---:|
| ANONYMOUS | 4505 | n/a | No | No |
| PLAIN | 4616 | Yes | No | No |
| EXTERNAL | 4422 | No (cert-based) | Yes (TLS) | Implicit |
| LOGIN (deprecated) | — | Yes | No | No |
| CRAM-MD5 (deprecated) | 2195 | Server stores plaintext | No | No |
| DIGEST-MD5 (deprecated) | 2831 | No (challenge) | Yes | Yes |
| GSSAPI | 4752 | No (Kerberos) | Yes | Yes |
| SCRAM-SHA-1 | 5802 | No (HMAC) | Yes | Yes |
| SCRAM-SHA-256 | 7677 | No (HMAC) | Yes | Yes |
| SCRAM-SHA-512 | — | No (HMAC) | Yes | Yes |

### PLAIN Mechanism

Wire format (RFC 4616):

```
authzid UTF8NUL authcid UTF8NUL passwd
```

If `authzid` is empty, the authenticated identity (authcid) is also the authorization identity. PLAIN sends the password in cleartext; LDAP servers MUST reject PLAIN binds outside TLS unless explicitly enabled.

### EXTERNAL Mechanism

Authentication occurs at the transport layer (TLS client certificate or Unix socket peer credentials). The SASL exchange is a no-op:

```
C: bindRequest { sasl { mechanism: "EXTERNAL", credentials: <empty> } }
S: bindResponse { resultCode: success }
```

The server derives the authenticated DN from the certificate Subject (or via `olcSaslAuthzPolicy`) using a regex map (`olcAuthzRegexp`).

### GSSAPI / Kerberos v5

The full GSSAPI exchange involves:

1. Client obtains TGT via AS_REQ → AS_REP
2. Client requests service ticket (`ldap/server.example.com@REALM`) via TGS_REQ → TGS_REP
3. Client wraps service ticket in `AP_REQ` and sends as SASL initial response
4. Server unwraps with its keytab, validates authenticator, sends `AP_REP`
5. Optional integrity / confidentiality wrapping via `gss_wrap`

Kerberos clock skew tolerance defaults to 5 minutes. Authenticator replay protection uses a replay cache with 5-minute window.

### SCRAM Mathematical Model (RFC 5802)

SCRAM (Salted Challenge Response Authentication Mechanism) eliminates plaintext password exposure on both client and server.

#### Constants

- $H$: hash function (SHA-1 / SHA-256 / SHA-512)
- $\text{HMAC}(K, M)$: keyed hash
- $\text{Hi}(P, S, i) = U_1 \oplus U_2 \oplus \ldots \oplus U_i$ where $U_1 = \text{HMAC}(P, S \| 0x00000001)$ and $U_j = \text{HMAC}(P, U_{j-1})$ — this is PBKDF2

#### Key Derivation

$$\text{SaltedPassword} = \text{Hi}(\text{Normalize}(P), \text{salt}, i)$$

$$\text{ClientKey} = \text{HMAC}(\text{SaltedPassword}, \text{"Client Key"})$$

$$\text{StoredKey} = H(\text{ClientKey})$$

$$\text{ServerKey} = \text{HMAC}(\text{SaltedPassword}, \text{"Server Key"})$$

The server stores `(StoredKey, ServerKey, salt, iter)`. It never sees `SaltedPassword` or the password itself.

#### Auth Message Construction

$$\text{AuthMessage} = \text{client-first-bare} \| \text{,} \| \text{server-first} \| \text{,} \| \text{client-final-no-proof}$$

Where:

```
client-first-bare        = "n=user,r=<c-nonce>"
server-first             = "r=<c-nonce><s-nonce>,s=<base64-salt>,i=<iter>"
client-final-no-proof    = "c=<base64-cbind>,r=<c-nonce><s-nonce>"
```

#### Client Proof

$$\text{ClientSignature} = \text{HMAC}(\text{StoredKey}, \text{AuthMessage})$$

$$\text{ClientProof} = \text{ClientKey} \oplus \text{ClientSignature}$$

Server validation:

$$\text{ClientKey}' = \text{ClientProof} \oplus \text{ClientSignature}$$

$$\text{Accept iff } H(\text{ClientKey}') = \text{StoredKey}$$

#### Server Signature

$$\text{ServerSignature} = \text{HMAC}(\text{ServerKey}, \text{AuthMessage})$$

The client validates `ServerSignature` to authenticate the server (mutual auth).

#### Exchange Diagram

```
Client                                         Server
  │                                              │
  │── n,,n=user,r=<c-nonce> ───────────────────▶│
  │   (client-first)                             │
  │                                              │
  │◀── r=<c-nonce><s-nonce>,s=<salt>,i=<iter>──│
  │   (server-first)                             │
  │                                              │
  │── c=<cbind>,r=<combined-nonce>,            │
  │   p=base64(ClientProof) ───────────────────▶│
  │   (client-final)                             │
  │                                              │
  │◀── v=base64(ServerSignature) ───────────────│
  │   (server-final)                             │
  │                                              │
```

#### Why SCRAM Is Safer Than DIGEST-MD5

| Property | DIGEST-MD5 | SCRAM-SHA-256 |
|:---|:---:|:---:|
| Server stores plaintext | Often yes (for HA1 derivation) | No |
| Hash function | MD5 (broken) | SHA-256 |
| Salt | No | Yes |
| Iteration count | No | Yes (default 4096+) |
| Channel binding | Optional | Required option |
| Mutual auth | Yes | Yes |
| Server-side compromise reveals password | Yes (HA1 reversible) | No (StoredKey is one-way) |

Server compromise of SCRAM data exposes:

1. `StoredKey` — usable to impersonate the user IF the attacker also has `AuthMessage` and can compute `HMAC(StoredKey, AuthMessage)` plus XOR with a `ClientKey` they cannot derive without the original password. Effectively useless without password recovery.
2. `ServerKey` — usable only to forge `server-final` messages to clients (impersonate server).

PBKDF2 iteration count $i$ defends against offline brute force. Time per guess: $i \cdot t_{\text{HMAC}}$. For $i = 100000$ and $t_{\text{HMAC}} = 1\,\mu s$: 100 ms/guess → $10^7$ years for an 8-char password against a single GPU.

### Channel Binding (RFC 5929)

`tls-server-end-point`: hash of the server's TLS certificate.
`tls-unique`: TLS finished message bytes.

Binding the SASL exchange to the TLS channel prevents man-in-the-middle attacks where an attacker terminates TLS and forwards SASL.

---

## 5. Replication Algorithms

### OpenLDAP syncrepl (RFC 4533)

#### Cookie-Based Incremental Sync

The consumer maintains a sync cookie containing:

1. `csn=<set-of-CSNs>` — context CSN tracking
2. `rid=<replica-id>` — consumer identifier
3. `sid=<server-id>` — last seen supplier

Cookie wire format:

```
rid=001,csn=20240315120000.000000Z#000000#000#000000
```

#### Modes

**refreshOnly**: poll-based.

```
Consumer                                   Supplier
   │                                          │
   │── SearchRequest + SyncRequestControl ───▶│
   │   (mode=refreshOnly, cookie=<old>)       │
   │                                          │
   │◀── SearchResultEntry × N ────────────────│
   │   (each with SyncStateControl)           │
   │                                          │
   │◀── SearchResultDone + SyncDoneControl ───│
   │   (cookie=<new>)                         │
```

**refreshAndPersist**: persistent search.

```
Consumer                                   Supplier
   │── SearchRequest + SyncRequestControl ───▶│
   │   (mode=refreshAndPersist, cookie=...)   │
   │                                          │
   │◀── SearchResultEntry × N (refresh phase) │
   │                                          │
   │◀── SyncInfo (newcookie) ─────────────────│
   │   (transition to persist phase)          │
   │                                          │
   │◀── SearchResultEntry (live updates) ─────│
   │◀── SearchResultEntry (live updates) ─────│
   │       ⋮                                  │
```

#### CSN — Change Sequence Number

Format: `<timestamp>.<microseconds>Z#<count>#<sid>#<modcount>`

Example: `20240315120030.123456Z#000001#001#000002`

| Field | Width | Meaning |
|:---|:---:|:---|
| timestamp | 14 | YYYYMMDDhhmmss in UTC |
| microseconds | 6 | sub-second tiebreaker |
| count | 6 | sequence within timestamp |
| sid | 3 | Server ID (000-fff hex, 12 bits) |
| modcount | 6 | per-modify-op counter for multi-attribute changes |

Lexicographic ordering of CSN strings = causal ordering of writes (within the precision of the timestamp). The `sid` field disambiguates concurrent writes at different replicas.

#### Conflict Resolution: Last-Write-Wins

For two writes $W_1, W_2$ to the same `(DN, attribute)` tuple:

- If $\text{CSN}(W_1) < \text{CSN}(W_2)$: $W_2$ wins
- If $\text{CSN}(W_1) > \text{CSN}(W_2)$: $W_1$ wins
- If $\text{CSN}(W_1) = \text{CSN}(W_2)$: impossible (sid disambiguates)

Per-attribute granularity means concurrent edits to different attributes of the same entry merge naturally.

#### syncrepl Pseudocode (Consumer Side)

```
LOOP:
  cookie = LOAD_COOKIE()
  CONNECT(supplier_url)
  BIND(mech)
  SEARCH(base, scope, filter, attrs,
         controls=[SyncRequestControl(mode=refreshAndPersist, cookie=cookie)])

  PHASE = REFRESH
  WHILE response = NEXT():
    if response is SearchResultEntry:
      ssc = response.controls[SyncStateControl]
      switch ssc.state:
        case ADD:    LOCAL_ADD(response.entry)
        case MODIFY: LOCAL_MODIFY(response.entry)
        case DELETE: LOCAL_DELETE(response.dn)
        case PRESENT: mark_present(response.dn)
    elif response is SyncInfoMessage:
      if response.has(refreshDelete) or response.has(refreshPresent):
        sweep_phase()
      if response.newcookie:
        cookie = response.newcookie
        SAVE_COOKIE(cookie)
      if PHASE == REFRESH and response.refreshDone:
        PHASE = PERSIST
    elif response is SearchResultDone:
      cookie = response.controls[SyncDoneControl].cookie
      SAVE_COOKIE(cookie)
      break
  RECONNECT_DELAY()
GOTO LOOP
```

### Active Directory MMR (Multi-Master Replication)

#### USN — Update Sequence Number

Each DC maintains:

- `highestCommittedUsn`: monotonic 64-bit counter of local writes
- `up-to-dateness vector`: map of `(invocationID, USN)` per known DC

Vector clock semantics: replica $A$ knows it has applied all $B$'s writes up to $\text{USN}_B = u$.

#### Stamp Format

Every attribute carries `(version, originating_time, originating_dsa, originating_usn)`:

| Field | Bits | Purpose |
|:---|:---:|:---|
| version | 32 | per-attribute write counter |
| originating_time | 64 | UTC seconds since 1601-01-01 |
| originating_dsa | 128 | invocationID GUID |
| originating_usn | 64 | USN at originating DC |

#### Conflict Resolution Order

Per RFC-style conflict ordering used by AD:

1. Higher `version` wins
2. If tied, more recent `originating_time` wins
3. If tied, higher `originating_dsa` GUID (lexicographic) wins

#### Tombstone Retention

Deleted entries become tombstones with `isDeleted=TRUE`. Default retention 60 days (Win2003) / 180 days (Win2003 SP1+) / configurable. Replicas pruning a tombstone before all peers have replicated the deletion creates the **lingering object** problem: deleted entries reappear. The "strict replication consistency" feature blocks this by failing replication if originator's USN is below tombstone lifetime threshold.

#### Replication Topology

KCC (Knowledge Consistency Checker) computes a 2-connected directed graph among DCs within a site, with inter-site links following site-link cost. Default intra-site replication interval: 15 seconds for urgent updates; default inter-site: 180 minutes.

### Multi-Master Conflict Scenarios

| Scenario | Resolution |
|:---|:---|
| Concurrent write to same attribute | Highest version → latest timestamp → highest GUID |
| Concurrent rename to same RDN | Loser gets renamed to `CNF:<guid>` (conflict-suffixed) |
| Concurrent delete + modify | Delete wins; modify discarded |
| Schema update conflicts | Replicated as serialized add/delete sequence |
| Cross-replica DN collision after rename | Conflict resolution per attribute, may manifest as duplicate object stub |

---

## 6. Index Theory

### OpenLDAP Index Types

| Type | Index Form | Query Pattern | Cost |
|:---|:---|:---|:---|
| `eq` | BTree on attr-value | `(attr=value)` | $O(\log N + R)$ |
| `pres` | BTree on attr-name | `(attr=*)` | $O(\log N + R)$ |
| `sub.initial` | BTree on prefixes | `(attr=prefix*)` | $O(\log N + R)$ |
| `sub.any` | BTree on n-grams | `(attr=*infix*)` | $O(K \cdot \log N)$ |
| `sub.final` | BTree on suffixes | `(attr=*suffix)` | $O(\log N + R)$ |
| `approx` | BTree on soundex | `(attr~=value)` | $O(\log N + R)$ |
| `inherit` | Class hierarchy | objectClass walk | $O(C)$ |

### MDB Backend Memory Model

The `mdb` backend uses LMDB (Lightning Memory-Mapped Database):

- B+tree page size: 4096 bytes
- Branching factor: $\approx \frac{4096 - 16}{8 + \text{key\_size}}$
- For 8-byte ID and 32-byte key: ~80 children per page
- Tree depth for $N = 10^7$ entries: $\lceil \log_{80}(10^7) \rceil = 4$

### Index Size Calculation

Per-entry index storage estimates:

| Index | Bytes/entry | 10M entries |
|:---|:---:|:---:|
| eq | 24 (ID + key + ptr) | 240 MB |
| pres | 8 (ID only) | 80 MB |
| sub.initial | 32 × avg-prefix-count | 1.5 GB |
| sub.any | 64 × avg-trigram-count | 3.0 GB |
| sub.final | 32 | 320 MB |

Five typical indexes on `cn`: `eq, pres, sub` consume ~5 GB for 10M entries. The directory itself at ~2 KB/entry is 20 GB. Indexes are 25% of total — a real workload constraint.

### Equality Index Build Algorithm

```
BUILD_EQ_INDEX(attr, entries):
  index = empty BTree
  for entry in entries:
    if attr in entry:
      for value in entry[attr]:
        normalized = NORMALIZE(value, attr.equality_rule)
        index.INSERT(normalized, entry.id)
  index.FLUSH()
```

Normalization includes case folding (`caseIgnoreMatch`), whitespace collapse, Unicode canonicalization (NFKC). A naive cost analysis ignoring normalization:

$$T_{build} = N \cdot \bar{V} \cdot O(\log N)$$

where $\bar{V}$ = average value count per entry. For $N = 10^6$, $\bar{V} = 2$, BTree insert at 1µs: $T_{build} \approx 40$ s.

### Substring Index Trigram Construction

For attribute value `"alice"`:

1. Pad: `"##alice$$"` (start markers + end marker)
2. Generate 3-grams: `"##a"`, `"#al"`, `"ali"`, `"lic"`, `"ice"`, `"ce$"`, `"e$$"`
3. Insert each 3-gram → entry-id pair into `sub.any` index

Query `(cn=*lic*)`:

1. Tokenize query into 3-grams: `"lic"`
2. Lookup `"lic"` in `sub.any` → candidate set $C$
3. For each $e \in C$, fetch attribute and verify literal match (false positives possible)

False-positive rate depends on n-gram collision probability, which scales with $|alphabet|^{-n}$. For ASCII letters and trigrams: $26^{-3} \approx 5.7 \times 10^{-5}$ — but practical text is far from uniform, yielding much higher collision rates.

### Index Selectivity Statistics

The query optimizer needs cardinality estimates per indexed attribute. OpenLDAP does not maintain detailed histograms (unlike PostgreSQL) — it relies on:

1. `idl_cache_max_size`: candidate ID list cache
2. `mdb stats`: per-index entry count
3. `slapindex -v`: rebuild from scratch with statistics dump

For accurate cost-based optimization, sample-based selectivity estimation:

$$\hat{s}(\text{predicate}) = \frac{|\text{matches in sample}|}{|\text{sample}|}$$

Sample size $n$ for confidence $1-\alpha$ and error $\epsilon$:

$$n \geq \frac{z_{1-\alpha/2}^2 \cdot \hat{s}(1-\hat{s})}{\epsilon^2}$$

For $\alpha = 0.05$, $\epsilon = 0.01$, $\hat{s} = 0.5$: $n \approx 9604$.

---

## 7. Search Algorithm Decomposition

### SearchRequest ASN.1 (RFC 4511 §4.5.1)

```asn1
SearchRequest ::= [APPLICATION 3] SEQUENCE {
    baseObject     LDAPDN,
    scope          ENUMERATED {
        baseObject   (0),
        singleLevel  (1),
        wholeSubtree (2),
        ... },
    derefAliases   ENUMERATED {
        neverDerefAliases  (0),
        derefInSearching   (1),
        derefFindingBaseObj (2),
        derefAlways        (3) },
    sizeLimit      INTEGER (0 .. maxInt),
    timeLimit      INTEGER (0 .. maxInt),
    typesOnly      BOOLEAN,
    filter         Filter,
    attributes     AttributeSelection }
```

### Search Algorithm

```
SEARCH(req):
  # Step 1: resolve baseObject
  base_entry = LOOKUP_DN(req.baseObject)
  if base_entry is None:
    return error(noSuchObject)

  # Step 2: collect candidates per scope
  switch req.scope:
    case BASE:
      candidates = [base_entry]
    case ONE:
      candidates = CHILDREN(base_entry)
    case SUB:
      candidates = SUBTREE(base_entry)

  # Step 3: filter optimizer — try indexes first
  optimized = OPTIMIZE_FILTER(req.filter, indexes)
  if optimized.usable_index:
    candidates = candidates ∩ INDEX_LOOKUP(optimized)

  # Step 4: linear filter pass
  results = []
  for entry in candidates:
    if EVAL(req.filter, entry) is TRUE:
      if ACL_CHECK(req.bound_dn, entry, "read"):
        results.append(entry)
        if len(results) > req.sizeLimit:
          return error(sizeLimitExceeded)
      if elapsed() > req.timeLimit:
        return error(timeLimitExceeded)

  # Step 5: apply controls
  if req.has(SortControl):
    results = SORT(results, req.SortControl.keys)
  if req.has(PagedResultsControl):
    page, cookie = PAGE(results, req.PagedResultsControl.size, req.PagedResultsControl.cookie)
    results = page
    response_controls = [PagedResultsControl(cookie=cookie)]

  # Step 6: project attributes
  for entry in results:
    SEND(SearchResultEntry(entry.dn, project(entry, req.attributes, req.typesOnly)))

  SEND(SearchResultDone(success, response_controls))
```

### Complexity by Scope

| Scope | Indexed Filter | Unindexed Filter |
|:---|:---|:---|
| BASE | $O(\log N)$ | $O(\log N + F)$ |
| ONE | $O(C \cdot F)$ | $O(C \cdot F)$ |
| SUB | $O(\log N + R \cdot F)$ | $O(K \cdot F)$ |

Where:

- $N$ = total entries
- $C$ = direct child count
- $K$ = subtree size = $\sum b^i$
- $R$ = result count after index intersection
- $F$ = filter evaluation cost per entry

### Aliases (derefAliases)

`alias` objectClass entries point to other DNs via `aliasedObjectName`. Dereferencing during search creates cycle risk. RFC 4511 §4.1.10 mandates loop detection — implementations cap dereference depth (OpenLDAP: 16 levels).

Alias cost: each dereference is a fresh DN lookup. Worst case: $O(D \log N)$ for $D$-deep chain.

### typesOnly

When `typesOnly=TRUE`, server returns attribute names without values. Wire reduction: $O(R \cdot \bar{A} \cdot \bar{V})$ → $O(R \cdot \bar{A})$ where $\bar{A}$ = avg attributes/entry, $\bar{V}$ = avg values/attribute. Useful for "does this entry have attribute X" probes.

---

## 8. Paged Results (RFC 2696)

### Control Definition

```asn1
realSearchControlValue ::= SEQUENCE {
    size            INTEGER (0..maxInt),  -- requested page size; cookie length on response
    cookie          OCTET STRING }        -- opaque server cursor
```

### Control OID

`1.2.840.113556.1.4.319` (registered originally by Microsoft for AD).

### Cursor Semantics

The cookie is opaque to the client. Server-side, it typically encodes:

- Position in the result stream (e.g., last DN seen, or a B-tree iterator state)
- Search invariants (filter hash, base DN, scope) for validation
- Timestamp for expiry checks

### Page Iteration

```
Client                                Server
  │                                     │
  │── SearchRequest +                   │
  │   PagedResultsControl(size=100,     │
  │                        cookie="") ─▶│
  │                                     │
  │◀── 100 SearchResultEntry ───────────│
  │                                     │
  │◀── SearchResultDone +               │
  │    PagedResultsControl(             │
  │      cookie="<opaque-bytes>") ──────│
  │                                     │
  │── SearchRequest (same) +            │
  │   PagedResultsControl(size=100,     │
  │                        cookie=<>) ─▶│
  │                                     │
  │◀── 100 SearchResultEntry ───────────│
  │                                     │
  │◀── SearchResultDone +               │
  │    PagedResultsControl(             │
  │      cookie="") ────────────────────│
  │   (empty cookie = end of results)   │
```

### Math

- Total pages: $\lceil R / S \rceil$ where $R$ = total result count, $S$ = page size
- Average client-side memory: $O(S)$ regardless of $R$
- Server-side cursor memory: $O(P)$ where $P$ = active sessions × per-session state

Cursor expiry typically 60-300 seconds idle. After expiry, returning a stale cookie gets `unwillingToPerform` (53) or `unavailableCriticalExtension` (12).

### AD Page Size Limit

Active Directory enforces `MaxPageSize = 1000` by default. To retrieve >1000 entries, paged results is mandatory. The DC defaults are configurable via `ntdsutil` → "LDAP policies".

---

## 9. LDAP Performance Models

### Bind Latency Decomposition

| Phase | Cost (typical) | Notes |
|:---|:---:|:---|
| TCP SYN/SYN-ACK/ACK | 1 RTT | ~0.1-50 ms |
| TLS handshake (TLS 1.2 ECDHE-RSA) | 2 RTT + crypto | ~10-100 ms |
| TLS handshake (TLS 1.3) | 1 RTT + crypto | ~5-50 ms |
| BindRequest/BindResponse | 1 RTT | ~0.1-50 ms |
| SCRAM 3-way (SCRAM-SHA-256) | 2 RTT | ~0.2-100 ms |
| GSSAPI (Kerberos AS_REQ + TGS_REQ + AP_REQ) | 4-6 RTTs | ~10-200 ms |

End-to-end simple bind over TLS 1.3: 3 RTTs + crypto cost.

### Search Cost Decomposition

$$T_{search} = T_{net,req} + T_{base\_lookup} + T_{filter\_eval} + T_{result\_serial} + T_{net,resp}$$

For result $R$ entries with average $\bar{A}$ attributes and $\bar{V}$ values:

$$T_{result\_serial} = R \cdot (\text{ASN.1 overhead} + \bar{A} \cdot (\text{tag} + \text{name length}) + \bar{A} \cdot \bar{V} \cdot \text{value length})$$

ASN.1 overhead per SearchResultEntry: 4-8 bytes. Per attribute: 4-12 bytes. Per value: 2-4 bytes.

### Throughput Math

| Configuration | Ops/sec |
|:---|:---:|
| Single connection, no TLS | 1000-5000 (LAN) |
| Single connection, TLS | 500-3000 (LAN) |
| Connection pool 50, no TLS | 30000-100000 |
| Connection pool 50, TLS | 20000-80000 |
| AD GC over WAN (30 ms RTT) | 30 (single conn) |
| AD GC + pool 50 over WAN | 1000-1500 |

Network bound at $\approx \text{pool} / \text{RTT}$ for query-per-RTT.

### AD GC vs DC

Global Catalog (GC) holds a partial replica of every domain in the forest, indexed on a fixed attribute set (gc-replication-attribute-set). DC holds full schema for one domain.

- Forest-wide search of `(userPrincipalName=alice@example.com)`: GC required, single query
- Without GC: query each DC in each domain — N domains × M DCs

Port assignment:

- DC LDAP: 389 / 636
- GC LDAP: 3268 / 3269

### Connection Pool Sizing

Little's Law:

$$L = \lambda W$$

Where:

- $L$ = average connections in use (pool occupancy)
- $\lambda$ = arrival rate (queries/sec)
- $W$ = average response time

Pool size $L \geq \lambda W$ to avoid queue formation. Add safety factor of 1.5-2x for variance.

For $\lambda = 5000$ qps and $W = 5$ ms: $L \geq 25$, recommended pool = 50.

---

## 10. Schema Validation Math

### Schema Object Class Hierarchy

```
top (ABSTRACT)
 ├── person (STRUCTURAL)
 │    ├── organizationalPerson (STRUCTURAL)
 │    │    └── inetOrgPerson (STRUCTURAL)
 │    └── residentialPerson (STRUCTURAL)
 ├── groupOfNames (STRUCTURAL)
 ├── groupOfUniqueNames (STRUCTURAL)
 ├── organizationalUnit (STRUCTURAL)
 ├── posixAccount (AUXILIARY)
 ├── shadowAccount (AUXILIARY)
 └── extensibleObject (AUXILIARY)
```

### Class Kinds

| Kind | Primary Purpose | Multiplicity per Entry |
|:---|:---|:---:|
| ABSTRACT | Provides shared MUST/MAY without instantiation | 0 (cannot be primary) |
| STRUCTURAL | Defines the entry's primary identity | exactly 1 chain |
| AUXILIARY | Adds attributes orthogonally | 0 or more |

An entry's `objectClass` set: exactly one structural chain (each parent up to `top`) + any number of auxiliary classes.

### Required Attributes

For an entry with object classes $\{O_1, O_2, \ldots, O_k\}$:

$$\text{must}(\text{entry}) = \bigcup_{i=1}^{k} \text{must}(O_i)$$

$$\text{may}(\text{entry}) = \bigcup_{i=1}^{k} \text{may}(O_i) \setminus \text{must}(\text{entry})$$

$$\text{allowed}(\text{entry}) = \text{must}(\text{entry}) \cup \text{may}(\text{entry})$$

### Validation Algorithm

```
VALIDATE(entry):
  classes = entry.objectClass
  must_set = empty
  allowed_set = empty

  # Walk class hierarchy
  for cls in classes:
    chain = ANCESTOR_CHAIN(cls)
    for ancestor in chain:
      must_set ∪= ancestor.must
      allowed_set ∪= ancestor.must ∪ ancestor.may

  # Verify required attributes present
  for attr in must_set:
    if attr not in entry.attributes:
      return error(objectClassViolation, attr)

  # Verify no extra attributes
  for attr in entry.attributes:
    if attr not in allowed_set and "extensibleObject" not in classes:
      return error(objectClassViolation, attr)

  # Verify exactly one structural chain
  structural_classes = [c for c in classes if c.kind == STRUCTURAL]
  if not exactly_one_leaf_chain(structural_classes):
    return error(objectClassViolation)

  # Verify syntax per attribute
  for attr, values in entry.attributes:
    schema = LOOKUP_ATTR_SYNTAX(attr)
    for v in values:
      if not schema.matches(v):
        return error(invalidAttributeSyntax, attr)

  return success
```

Time: $O(C + A + \sum_i |V_i| \cdot t_{\text{syntax}})$ where $C$ = class count, $A$ = attribute count, $|V_i|$ = values per attribute.

### Attribute Syntax Definitions (RFC 4517)

| Syntax OID | Name | Validation |
|:---|:---|:---|
| 1.3.6.1.4.1.1466.115.121.1.15 | DirectoryString | UTF-8 |
| 1.3.6.1.4.1.1466.115.121.1.27 | INTEGER | Numeric |
| 1.3.6.1.4.1.1466.115.121.1.36 | NumericString | `[0-9 ]+` |
| 1.3.6.1.4.1.1466.115.121.1.38 | OID | dotted-decimal |
| 1.3.6.1.4.1.1466.115.121.1.40 | OctetString | binary |
| 1.3.6.1.4.1.1466.115.121.1.50 | TelephoneNumber | E.123 lenient |
| 1.3.6.1.4.1.1466.115.121.1.55 | UTC Time | YYMMDDhhmmss[Z\|±hhmm] |
| 1.3.6.1.4.1.1466.115.121.1.24 | Generalized Time | YYYYMMDDhhmmss[.fff][Z\|±hhmm] |

---

## 11. ACL / ACI Evaluation

### OpenLDAP olcAccess Rules

Wire syntax (LDIF):

```
olcAccess: to dn.subtree="ou=eng,dc=example,dc=com" attrs=userPassword
  by self =w
  by anonymous auth
  by * none

olcAccess: to dn.subtree="ou=eng,dc=example,dc=com" attrs=cn,sn,givenName
  by * read

olcAccess: to *
  by self read
  by * none
```

Rules are evaluated in order; the first rule whose `to` clause matches the request decides access. Within the matching rule, `by` clauses are evaluated in order; the first matching `by` clause's permission is granted.

### Permission Levels

| Level | Includes |
|:---:|:---|
| none | (no access) |
| disclose | error code visibility |
| auth | bind/compare for authentication |
| compare | compare operation |
| search | filter component evaluation |
| read | full read access |
| write | modify, add, delete |
| manage | rename DN, schema mods |

Permissions are cumulative — `write` includes everything below.

### Decision Algorithm

```
ACCESS_CHECK(bound_dn, target_dn, attribute, operation):
  for rule in olcAccess (in order):
    if rule.to.matches(target_dn, attribute):
      for by in rule.by_clauses (in order):
        if by.matches(bound_dn):
          required_level = LEVEL_FOR(operation)
          return by.level >= required_level
  return DENY  # default
```

Worst case: $O(R \cdot G)$ where $R$ = total rules, $G$ = group memberships expanded for `by group=` clauses.

### 389-DS / Active Directory ACI

ACIs attach as multi-valued attributes on entries (`aci` in 389-DS, `nTSecurityDescriptor` in AD):

```
aci: (target="ldap:///ou=eng,dc=example,dc=com")
     (targetattr="cn || sn || mail")
     (version 3.0; acl "Allow read"; allow (read,search,compare)
      userdn="ldap:///cn=engineers,ou=groups,dc=example,dc=com";)
```

Inheritance: child entries inherit applicable ACIs from ancestors in the DIT. Conflict: deny rules override allow rules within the same scope.

### AD nTSecurityDescriptor

Binary SDDL-encoded. Components:

- `Owner SID` (8-32 bytes)
- `Group SID` (optional)
- `DACL` (Discretionary Access Control List): array of ACEs
- `SACL` (System Access Control List): audit ACEs

Each ACE:

```
+--+------+-------+--------+-----------+
|Type|Flags|Size   |Mask    |SID       |
+--+------+-------+--------+-----------+
| 1 |  1  |   2   |   4    | variable  |
+--+------+-------+--------+-----------+
```

Mask is 32-bit access bitmap: ADS_RIGHT_DS_READ_PROP (0x10), ADS_RIGHT_DS_WRITE_PROP (0x20), ADS_RIGHT_DS_CREATE_CHILD (0x1), etc.

---

## 12. Active Directory Schema Specifics

### Critical Attributes

| Attribute | Syntax | Constraint | Purpose |
|:---|:---|:---|:---|
| `sAMAccountName` | DirectoryString | ≤20 chars, NetBIOS-safe | Legacy login |
| `userPrincipalName` | DirectoryString | RFC 822 format | Modern login (UPN) |
| `objectGUID` | OctetString | 128-bit immutable | Replica-stable identity |
| `objectSid` | OctetString | Variable-length SID | Authorization identity |
| `userAccountControl` | INTEGER | 32-bit flags | Account state |
| `pwdLastSet` | Generalized Time | UTC | Password age tracking |
| `accountExpires` | INTEGER | FILETIME or 0/maxValue | Expiry |
| `lastLogonTimestamp` | INTEGER | FILETIME, replicated | Last interactive auth |
| `memberOf` | DN | back-link | Group membership reverse |

### userAccountControl Flag Bits

| Bit | Hex | Symbol |
|:---:|:---:|:---|
| 0 | 0x0001 | SCRIPT |
| 1 | 0x0002 | ACCOUNTDISABLE |
| 3 | 0x0008 | HOMEDIR_REQUIRED |
| 4 | 0x0010 | LOCKOUT |
| 5 | 0x0020 | PASSWD_NOTREQD |
| 6 | 0x0040 | PASSWD_CANT_CHANGE |
| 7 | 0x0080 | ENCRYPTED_TEXT_PWD_ALLOWED |
| 8 | 0x0100 | TEMP_DUPLICATE_ACCOUNT |
| 9 | 0x0200 | NORMAL_ACCOUNT |
| 11 | 0x0800 | INTERDOMAIN_TRUST_ACCOUNT |
| 12 | 0x1000 | WORKSTATION_TRUST_ACCOUNT |
| 13 | 0x2000 | SERVER_TRUST_ACCOUNT |
| 16 | 0x10000 | DONT_EXPIRE_PASSWORD |
| 17 | 0x20000 | MNS_LOGON_ACCOUNT |
| 18 | 0x40000 | SMARTCARD_REQUIRED |
| 19 | 0x80000 | TRUSTED_FOR_DELEGATION |
| 20 | 0x100000 | NOT_DELEGATED |
| 21 | 0x200000 | USE_DES_KEY_ONLY |
| 22 | 0x400000 | DONT_REQ_PREAUTH |
| 23 | 0x800000 | PASSWORD_EXPIRED |
| 24 | 0x1000000 | TRUSTED_TO_AUTH_FOR_DELEGATION |
| 25 | 0x2000000 | NO_AUTH_DATA_REQUIRED |
| 26 | 0x4000000 | PARTIAL_SECRETS_ACCOUNT |

Common queries:

- `(&(objectClass=user)(userAccountControl:1.2.840.113556.1.4.803:=2))` — disabled accounts (LDAP_MATCHING_RULE_BIT_AND)
- `(&(objectClass=user)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))` — enabled accounts

The OID `1.2.840.113556.1.4.803` is `LDAP_MATCHING_RULE_BIT_AND`; `1.2.840.113556.1.4.804` is `LDAP_MATCHING_RULE_BIT_OR`.

### SID Structure

```
+--+--+--+--+--+--+--+--+--+--+--+--+
|Rev|SubAuthCount|        Authority |
+--+--+--+--+--+--+--+--+--+--+--+--+
|         SubAuthority[0]            |
|         SubAuthority[1]            |
|              ⋮                      |
|         SubAuthority[N-1]          |
+----+----+----+----+----+----+----+
```

| Field | Size | Purpose |
|:---|:---:|:---|
| Revision | 1 byte | Always 1 |
| SubAuthorityCount | 1 byte | N (max 15) |
| IdentifierAuthority | 6 bytes | Top-level domain (e.g., 5 = NT_AUTHORITY) |
| SubAuthority[i] | 4 bytes each | Domain RID + per-object RID |

Example: `S-1-5-21-1234567890-1234567890-1234567890-500` = built-in Administrator.

### FILETIME

64-bit count of 100-nanosecond intervals since 1601-01-01T00:00:00Z UTC:

$$\text{FILETIME} = (\text{unix\_seconds} + 11644473600) \times 10^7 + \text{nanos}/100$$

`accountExpires=0` and `accountExpires=0x7FFFFFFFFFFFFFFF` both mean "never expires".

### Why Two Usernames

| Login style | Field | Format |
|:---|:---|:---|
| Pre-Win2000 (NetBIOS) | `sAMAccountName` | DOMAIN\user |
| Win2000+ / federation | `userPrincipalName` | user@upnsuffix |

UPN suffix can differ from FQDN — admins configure alternative UPN suffixes for SSO/SAML/OIDC integration. SAMAccountName remains for legacy NTLM clients.

---

## 13. Modern Federation: SAML, OIDC, SCIM

### SAML 2.0 with LDAP Backend

SP-IdP flow with LDAP as identity store:

```
User    Browser     SP            IdP             LDAP
 │        │         │              │                │
 │ access │         │              │                │
 │───────▶│         │              │                │
 │        │ GET resource           │                │
 │        │────────▶│              │                │
 │        │  302    │              │                │
 │        │◀────────│              │                │
 │        │ AuthnRequest           │                │
 │        │───────────────────────▶│                │
 │        │ login form             │                │
 │        │◀───────────────────────│                │
 │ creds  │                        │                │
 │───────▶│  POST creds            │                │
 │        │───────────────────────▶│                │
 │        │                        │ search/bind    │
 │        │                        │───────────────▶│
 │        │                        │  result        │
 │        │                        │◀───────────────│
 │        │ SAMLResponse(assertion)│                │
 │        │◀───────────────────────│                │
 │        │ POST /acs              │                │
 │        │────────▶│              │                │
 │        │ resource│              │                │
 │        │◀────────│              │                │
```

The IdP performs an LDAP simple bind (or SASL) to validate the password, then issues a SAML assertion containing attributes pulled from LDAP via search.

### OIDC with LDAP

OIDC adds JWT issuance atop SAML semantics:

1. Authorization endpoint → user authenticates against LDAP
2. Token endpoint issues `id_token` (signed JWT) with claims sourced from LDAP attributes
3. Userinfo endpoint returns more LDAP attributes on demand

JWT payload claims commonly mapped from LDAP:

| OIDC Claim | LDAP Attribute |
|:---|:---|
| `sub` | `objectGUID` / `uid` / `entryUUID` |
| `name` | `cn` |
| `given_name` | `givenName` |
| `family_name` | `sn` |
| `email` | `mail` |
| `email_verified` | (computed/static) |
| `groups` | `memberOf` (DN-to-name mapped) |

### SCIM 2.0 (RFC 7642-7644)

REST-based provisioning protocol. Resources:

| Resource | Endpoint | Maps to LDAP |
|:---|:---|:---|
| User | `/Users` | `inetOrgPerson` |
| Group | `/Groups` | `groupOfNames` |
| Schema | `/Schemas` | objectClass+attributeType |
| ResourceType | `/ResourceTypes` | per object class |

Core User schema:

```json
{
  "id": "<server-assigned-uuid>",
  "externalId": "<client-assigned-id>",
  "userName": "alice",
  "name": {
    "formatted": "Alice Smith",
    "familyName": "Smith",
    "givenName": "Alice"
  },
  "emails": [{ "value": "alice@example.com", "primary": true }],
  "active": true,
  "groups": [{ "value": "<group-id>", "display": "Engineers" }]
}
```

### LDAP-to-SCIM Cardinality Mapping

| LDAP Cardinality | SCIM Cardinality | Note |
|:---|:---|:---|
| Single-valued attribute | scalar | direct |
| Multi-valued attribute (e.g., `mail`) | array of complex | wrap each value in `{value, type, primary}` |
| Multi-valued (e.g., `memberOf`) | array of references | each element is a User/Group $ref |
| Operational attribute | not exposed | filter out |
| Binary attribute | base64 string | per RFC 7643 |

### Migration Math

For $N$ users with average $\bar{V}$ multi-valued mail attributes:

- LDAP entries: $N$
- SCIM JSON document size per user: $O(\bar{V} + |\text{groups}| + |\text{phone}|)$
- Bulk migration time: $N \cdot T_{\text{ldap\_search}} + N \cdot T_{\text{scim\_post}}$

Typical: 50-200 ms per user via SCIM endpoint, dominated by HTTP+TLS overhead, not LDAP search.

---

## 14. LDAP Injection

### Vulnerability Class

Like SQL injection, but for filter strings constructed via string concatenation:

```python
# VULNERABLE
filter = f"(uid={user_input})"
ldap.search_s(base, SUB, filter)
```

If `user_input = "alice)(uid=*"`:

```
filter = "(uid=alice)(uid=*)"
```

LDAP parsers vary in their handling. Some treat the second `(uid=*)` as garbage and ignore it; others (especially older OpenLDAP) parse it as an additional filter list element under an implicit AND, broadening the search.

### Bypass Patterns

| Input | Resulting filter | Effect |
|:---|:---|:---|
| `*` | `(uid=*)` | match all (presence filter) |
| `*)(objectClass=*` | `(uid=*)(objectClass=*))` | bypass auth check |
| `a)(\|(password=*` | `(uid=a)(\|(password=*))` | extract password presence |
| `\\` | `(uid=\\)` | escape error |

### Mitigation: RFC 4515 §3 Escape Rules

The following characters MUST be escaped in assertion values:

| Char | Escape |
|:---:|:---|
| `*` | `\2a` |
| `(` | `\28` |
| `)` | `\29` |
| `\` | `\5c` |
| `\0` | `\00` |
| `/` | `\2f` |

Escape function:

```python
def escape_filter_chars(s):
    out = []
    for c in s:
        if c in '\x00*\\()/':
            out.append('\\%02x' % ord(c))
        elif ord(c) >= 0x80:  # also escape non-ASCII for safety
            for b in c.encode('utf-8'):
                out.append('\\%02x' % b)
        else:
            out.append(c)
    return ''.join(out)
```

### DN Escaping (RFC 4514 §2.4)

DN escaping rules differ from filter escaping. Required escapes in DN attribute values:

| Position | Chars |
|:---|:---|
| Leading | `#`, ` ` (space) |
| Trailing | ` ` (space) |
| Anywhere | `"`, `+`, `,`, `;`, `<`, `>`, `\`, `\0` |

### Library Functions

- Python `ldap3.utils.conv.escape_filter_chars`
- Python `ldap3.utils.dn.escape_rdn`
- Java `javax.naming.ldap.LdapName` (parses, doesn't accept raw strings)
- .NET `System.DirectoryServices.Protocols.LdapConnection` (parameterized via `SearchRequest` object)
- Go `github.com/go-ldap/ldap/v3` `ldap.EscapeFilter`

---

## 15. TLS Considerations

### LDAPS vs StartTLS

| Property | LDAPS | StartTLS |
|:---|:---|:---|
| Port | 636 | 389 |
| TLS at connection start | Yes | No (upgrade) |
| Standardization | Convention only | RFC 4511 §4.14 |
| Compatibility | All TLS-aware clients | Modern clients |
| Failure mode | Connection refused if no TLS | Plaintext if upgrade fails |

### StartTLS Exchange

```
Client                                Server
  │                                     │
  │── ExtendedRequest                   │
  │   (oid=1.3.6.1.4.1.1466.20037) ───▶│
  │                                     │
  │◀── ExtendedResponse                 │
  │   (resultCode=success, oid=...) ────│
  │                                     │
  │═══════ TLS handshake ═══════════════│
  │                                     │
  │── BindRequest (now encrypted) ─────▶│
  │                                     │
  │◀── BindResponse ────────────────────│
```

### StartTLS Downgrade Attack

A MITM attacker can strip the StartTLS exchange:

```
Client                  MITM                    Server
  │── StartTLS ────────▶│                         │
  │                     │── StartTLS ───────────▶│
  │                     │◀── success ────────────│
  │◀── unwillingToPerform ─                       │
  │                     │                         │
  │── BindRequest (cleartext) ──── intercept ───▶│
```

Mitigation: enforce TLS at the client by refusing to bind without TLS. Server-side: configure `security ssf=128` (OpenLDAP) or "Require LDAPS" (AD).

### TLS Cipher Suite Selection

Modern recommendations (2024):

| Cipher Suite | Status | Notes |
|:---|:---|:---|
| TLS_AES_256_GCM_SHA384 | TLS 1.3, preferred | AEAD |
| TLS_CHACHA20_POLY1305_SHA256 | TLS 1.3, mobile | AEAD |
| TLS_AES_128_GCM_SHA256 | TLS 1.3 | AEAD |
| TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 | TLS 1.2 | OK |
| TLS_RSA_WITH_AES_256_GCM_SHA384 | TLS 1.2, weak | No PFS |
| TLS_RSA_WITH_3DES_EDE_CBC_SHA | Disable | SWEET32 |
| TLS_RSA_WITH_RC4_128_SHA | Disable | RC4 broken |

### Certificate Validation in LDAP Clients

Clients MUST validate:

1. Server certificate signature chains to a trusted root
2. Certificate `notBefore ≤ now ≤ notAfter`
3. Certificate `subjectAltName.dNSName` matches LDAP server hostname (RFC 6125)
4. Revocation status via OCSP or CRL

OpenLDAP client: `TLS_REQCERT demand`, `TLS_CACERT /etc/ssl/cacert.pem`.

### Bind Strength

OpenLDAP `security ssf=N` requires N-bit minimum cipher strength:

- `ssf=0`: any (insecure)
- `ssf=56`: 56-bit (DES, weak)
- `ssf=128`: AES-128 minimum
- `ssf=256`: AES-256 minimum

Per-rule SSF: ACL clauses can require minimum SSF for the operation:

```
olcAccess: to dn.subtree="dc=example,dc=com" attrs=userPassword
  by ssf=128 self =w
  by * none
```

---

## 16. Failure Modes

### DNS / SRV Resolution

LDAP clients use SRV records for DC discovery:

```
_ldap._tcp.example.com.        SRV  0 100 389  dc1.example.com.
_ldap._tcp.example.com.        SRV  0 100 389  dc2.example.com.
_ldap._tcp.dc._msdcs.example.com. SRV 0 100 389 dc1.example.com.
```

Failure: SRV record absent → fallback to A record query for hostname → connection failure if DNS misconfigured. Symptoms: `gss_init_sec_context: Server not found in Kerberos database` (for AD); `Cannot contact LDAP server` (OpenLDAP).

### Clock Skew

Kerberos rejects authenticators with `Clock skew too great` (KRB_AP_ERR_SKEW) when client/server clocks differ by more than `clockskew = 300s` (default). SCRAM nonce validity depends on server-side replay cache window (typically 5 minutes).

NTP synchronization is mandatory. AD domains use the PDC emulator FSMO role as authoritative time source.

### Index Corruption

OpenLDAP MDB index files: `<attr>.bdb` per attribute per index type. Corruption causes:

- Stale ID lists in candidate set → false positives in result
- Missing entries from index → false negatives (silent data loss)
- Bypass to unindexed scan → latency spikes from $O(\log N)$ to $O(N)$

Detection: `slapindex -v` rebuild compares to expected. Recovery: `slapindex -q -v -b <suffix>` rebuilds all indexes.

### Replication Lag

```
   Supplier writes        Consumer A applies     Consumer B applies
       │                         │                         │
       │ t0                      │ t0+5s                   │ t0+30s
       │                         │                         │
       │                         │                         │
   ────┴─────────────────────────┴─────────────────────────┴──── time
```

Read at consumer A at $t_0+10\text{s}$ sees the change; read at consumer B at $t_0+10\text{s}$ does not. Application-level workaround: read-after-write consistency by directing read to the same replica as write, or by pinning sessions.

Lag detection metrics (OpenLDAP):

- `cn=Monitor,cn=Connections`: per-replica syncrepl status
- `contextCSN` attribute on suffix entry: latest CSN seen
- Difference between supplier and consumer `contextCSN` = lag

### Split-Brain

Network partition between replicas creates divergent state. Both sides accept writes; on heal, syncrepl conflict resolution applies. With per-attribute LWW, edits to disjoint attributes merge cleanly. Edits to the same attribute lose data on the loser side.

Total data loss bound: per (DN, attribute) pair, 1 write per side max → for $W$ writes during partition, expected loss $\leq W/2$.

### ACL Masking

A common misconfiguration: ACL denies `read` but permits `auth`. Affected operations:

- Search returning the entry: `noSuchObject` (32) if disclose denied
- Bind: succeeds despite invisibility
- Compare: succeeds for selected attributes

The error code `32` masks the existence of the entry — debugging this requires elevated credentials to introspect the ACL chain.

### Operation-Level Timeouts

| Layer | Timeout | Default |
|:---|:---|:---:|
| TCP connect | OS-level | 75-180 s |
| TLS handshake | Library | 10-30 s |
| Bind | LDAP client | 30 s |
| Search timeLimit | LDAP request | 0 (server policy) |
| Search sizeLimit | LDAP request | 0 (server policy) |
| Idle TCP | OS keepalive | 7200 s |
| Server `idletimeout` | OpenLDAP slapd | 0 (disabled) |

OpenLDAP server-side limits:

```
limits dn.exact="cn=admin,dc=example,dc=com" size=unlimited time=unlimited
limits anonymous size=10 time=5
```

### Resource Exhaustion

| Resource | Effect of exhaustion |
|:---|:---|
| File descriptors | New connections refused |
| RAM (entry cache) | LRU eviction → cache miss → disk reads |
| RAM (idl_cache) | Search planner falls back to full scan |
| Disk space | Writes fail with `unwillingToPerform` |
| Threads | Operation queueing, latency spike |

OpenLDAP `slapd` connection limits: `tool-threads N`, `concurrency M`. Default: $\text{cores} \cdot 2 + 1$ threads.

---

## 17. Deep Dive: ASN.1 SearchResultEntry Encoding

### Definition

```asn1
SearchResultEntry ::= [APPLICATION 4] SEQUENCE {
    objectName     LDAPDN,
    attributes     PartialAttributeList }

PartialAttributeList ::= SEQUENCE OF PartialAttribute

PartialAttribute ::= SEQUENCE {
    type    AttributeDescription,
    vals    SET OF AttributeValue }
```

### Worked Example

Entry:

```
dn: uid=alice,ou=eng,dc=example,dc=com
objectClass: inetOrgPerson
objectClass: posixAccount
cn: Alice
uidNumber: 1001
```

Encoding (assuming messageID = 2):

```
30 ?? ?? ??                           ; LDAPMessage SEQUENCE
   02 01 02                           ;   messageID=2
   64 ?? ?? ??                        ;   [APPLICATION 4] SearchResultEntry
      04 26                           ;     objectName OCTET STRING (38 bytes)
        "uid=alice,ou=eng,dc=example,dc=com"
      30 ?? ?? ??                     ;     attributes SEQUENCE
        30 ?? ?? ??                   ;       PartialAttribute (objectClass)
          04 0B                       ;         type "objectClass"
          31 ??                       ;         SET OF
            04 0F  "inetOrgPerson"   ;           value 1
            04 0C  "posixAccount"    ;           value 2
        30 ??                         ;       PartialAttribute (cn)
          04 02 "cn"
          31 09
            04 05 "Alice"
        30 ??                         ;       PartialAttribute (uidNumber)
          04 09 "uidNumber"
          31 06
            04 03 "1001"
```

### Multi-Value Encoding

`SET OF` (BER tag 0x31) preserves no order. Implementations typically emit values in storage order.

### Attribute Description Format (RFC 4512)

```
AttributeDescription = AttributeType *( ";" option )
```

Examples:

- `cn` — bare type
- `cn;lang-en` — language tag option
- `userCertificate;binary` — binary encoding option

The `;binary` option (RFC 4522) forces transfer encoding for syntaxes that would otherwise be string-encoded — important for `userCertificate;binary` to receive raw DER instead of an attempted text encoding.

---

## 18. Connection State Machine

### LDAP Session FSM

```
                        ┌──────────────┐
                        │   CLOSED     │
                        └──────┬───────┘
                               │ TCP connect
                               ▼
                        ┌──────────────┐
                        │  CONNECTED   │
                        └──────┬───────┘
                               │ optional StartTLS
                               ▼
                        ┌──────────────┐
                        │ TLS_PENDING  │
                        └──────┬───────┘
                               │ TLS established
                               ▼
                        ┌──────────────┐
                        │   ANON_BOUND │
                        └──────┬───────┘
                               │ BindRequest
                               ▼
                        ┌──────────────┐
                        │ BIND_PENDING │
                        └──────┬───────┘
                               │ BindResponse(success)
                               ▼
                        ┌──────────────┐
                ┌──────▶│  AUTHENTICATED│◀──────┐
                │       └──────┬────────┘       │
                │              │                │
                │ submit op    │ UnbindRequest  │ rebind
                ▼              ▼                │
        ┌──────────────┐  ┌──────────────┐     │
        │ OP_PENDING   │  │   CLOSING    │─────┘
        └──────┬───────┘  └──────────────┘
               │ response
               └────────▶ AUTHENTICATED
```

### Concurrent Operations

LDAP allows multiple concurrent operations on a single connection (multiplexed by `messageID`). The server tracks outstanding operations per connection up to a configured limit.

OpenLDAP slapd:
- `concurrency`: max simultaneous operations across all connections
- `connection_max_pending`: per-connection outstanding ops limit (default 100)

### Abandon Operation

```asn1
AbandonRequest ::= [APPLICATION 16] MessageID
```

No response. Client signals "abandon the operation with this messageID". Server SHOULD stop work; client MUST NOT use that messageID again.

Race: server may have already sent the result before processing the abandon. Client should discard any results arriving for an abandoned messageID.

---

## 19. Subschema Subentry

### Schema Discovery

The root DSE (`""`) advertises `subschemaSubentry` pointing to the subschema entry. Typically `cn=Subschema` or `cn=schema,cn=config`.

Query:

```
ldapsearch -x -H ldap://server -b "" -s base "(objectclass=*)" subschemaSubentry
```

Then read the subschema entry:

```
ldapsearch -x -H ldap://server -b "cn=Subschema" -s base "(objectclass=*)" \
  attributeTypes objectClasses ldapSyntaxes matchingRules dITContentRules
```

### attributeTypes Definition Format (RFC 4512)

```
( 2.5.4.3
  NAME 'cn'
  DESC 'commonName(X.520)'
  SUP name )
```

| Field | Purpose |
|:---|:---|
| OID | unique identifier |
| NAME | one or more textual names |
| DESC | human description |
| SUP | parent attribute type |
| EQUALITY | matching rule for `=` |
| ORDERING | matching rule for `<=`, `>=` |
| SUBSTR | matching rule for substring |
| SYNTAX | OID of value syntax |
| SINGLE-VALUE | scalar vs multi-valued |
| NO-USER-MODIFICATION | operational only |
| USAGE | userApplications / directoryOperation / distributedOperation / dSAOperation |

### objectClasses Definition Format

```
( 2.5.6.6
  NAME 'person'
  DESC 'RFC2256: a person'
  SUP top
  STRUCTURAL
  MUST ( sn $ cn )
  MAY  ( userPassword $ telephoneNumber $ seeAlso $ description ) )
```

### Schema Modification

OpenLDAP supports runtime schema modification via `cn=schema,cn=config` only since OpenLDAP 2.4 (slapd-config / OLC). Active Directory supports schema mods only through the schema master FSMO and requires `Schema Update Allowed` flag plus reboot for some changes.

---

## 20. Performance Tuning Cookbook

### OpenLDAP MDB Tuning

| Parameter | Default | Tuning |
|:---|:---|:---|
| `olcDbMaxSize` | 10 GB | Set to expected DB size × 2 |
| `olcDbReaders` | 126 | match concurrent search clients |
| `olcDbCheckpoint` | 0 | enable for crash safety |
| `olcDbEnvFlags` | (none) | set `MDB_NOMETASYNC,MDB_NOSYNC` for write-heavy workloads (sacrifices durability) |

### Index Selection Heuristics

For attribute $A$ with cardinality $|A|$ values across $N$ entries:

- Index if $|A| / N < 0.1$ (selective)
- Skip index if $|A| / N > 0.5$ (low selectivity, scan competitive)
- For substring search frequency $f_s > 0.1$: add `sub` index

### Query-Level Optimizations

```
# BEFORE (full scan likely)
filter: (|(givenName=*alice*)(sn=*alice*)(mail=*alice*))

# AFTER (specific attribute)
filter: (mail=alice@example.com)
```

```
# BEFORE (subtree at root with low-selectivity filter)
base: dc=example,dc=com
scope: sub
filter: (objectClass=person)

# AFTER (narrow base)
base: ou=eng,dc=example,dc=com
scope: sub
filter: (objectClass=person)
```

### Connection Pool Configuration

| Library | Pool option |
|:---|:---|
| OpenLDAP `libldap` | per-thread, no pooling — application-level pool |
| Python `python-ldap` | none built-in |
| Python `ldap3` | `ServerPool`, `Connection(client_strategy=POOLED)` |
| Java `UnboundID` | `LDAPConnectionPool` |
| Go `go-ldap` | application-level |

Recommended: pool sized for $\lambda \cdot W$ + 50% headroom. Health-check interval: 30-60s. Max-idle: 10 minutes (close before server `idletimeout`).

### Cache Hierarchy

```
   Application
        │
        ▼
   App Memcached / Redis        (TTL: 60-3600s, app-controlled)
        │ miss
        ▼
   SSSD / nss_ldap cache         (TTL: per cache_*_timeout)
        │ miss
        ▼
   Connection pool
        │
        ▼
   LDAP server
        │
        ▼
   slapd entry cache             (LRU, size = olcDbCachesize)
        │ miss
        ▼
   slapd idl_cache               (filter result cache)
        │ miss
        ▼
   MDB pages (mmap-ed)           (OS page cache)
        │ miss
        ▼
   Disk
```

Each layer multiplies effective query rate by its hit ratio. Cumulative hit rate of 99% reduces server load by 100×.

---

## 21. Operational Pseudocode: syncrepl Conflict Resolution

```
APPLY_REMOTE_UPDATE(supplier_csn, dn, attribute, new_value):
  local_csn = LOAD_ATTR_CSN(dn, attribute)

  if local_csn is None:
    # never seen, accept unconditionally
    STORE(dn, attribute, new_value, supplier_csn)
    return ACCEPTED

  if supplier_csn > local_csn:
    # remote write is more recent
    STORE(dn, attribute, new_value, supplier_csn)
    return ACCEPTED

  if supplier_csn < local_csn:
    # local write is more recent — discard remote
    return REJECTED_STALE

  if supplier_csn == local_csn:
    # impossible: CSN includes sid for uniqueness
    PANIC("CSN collision detected")

DETECT_CONFLICT(local_writes, remote_writes):
  conflicts = []
  for (dn, attr) in keys(local_writes) ∩ keys(remote_writes):
    lcsn = local_writes[(dn, attr)].csn
    rcsn = remote_writes[(dn, attr)].csn
    if lcsn != rcsn:
      conflicts.append((dn, attr, lcsn, rcsn))
  return conflicts

RECONCILE_AFTER_PARTITION(local_replica, remote_replica):
  for entry in remote_replica.changes_since(local_replica.contextCSN):
    for (attr, value, csn) in entry.modifications:
      APPLY_REMOTE_UPDATE(csn, entry.dn, attr, value)
  local_replica.contextCSN = max(local_replica.contextCSN, remote_replica.contextCSN)
```

---

## 22. Mathematical Properties of CSN Ordering

### Total vs Partial Order

CSN ordering is a **total order** (every pair is comparable) but only a **causal partial order** with respect to write events (not every CSN-greater event is causally after a CSN-lesser event — concurrent writes at different replicas have unrelated CSNs).

Formally, with $W_1 \to W_2$ denoting "$W_1$ happens-before $W_2$":

- $\text{CSN}(W_1) < \text{CSN}(W_2) \implies \neg(W_2 \to W_1)$ (sound)
- $W_1 \to W_2 \implies \text{CSN}(W_1) < \text{CSN}(W_2)$ (consistent)
- $\text{CSN}(W_1) < \text{CSN}(W_2) \not\implies W_1 \to W_2$ (incomplete: concurrent writes)

The `sid` component is necessary because timestamp + count is not unique across replicas with synchronized clocks. With $S$ replicas writing at rate $\lambda$ ops/sec, and timestamp resolution $\delta$, expected collisions per second:

$$E[\text{collisions}] = \binom{S\lambda}{2} \delta^2 / 2$$

For $S = 5$, $\lambda = 1000$, $\delta = 10^{-6}$ s: $E \approx 12.5$ collisions/sec at the timestamp+count level. The `sid` field disambiguates all of these.

### Vector Clock Equivalence

CSN-with-sid is operationally equivalent to a vector clock indexed by replica:

$$\text{VC}[i] = \max\{\text{CSN}.\text{count} \mid \text{CSN}.\text{sid} = i\}$$

The contextCSN attribute tracks the max-per-sid effectively — OpenLDAP stores it as a multi-valued attribute, one CSN per known supplier.

---

## 23. Cost of Group Membership Resolution

### Group Models

| Model | Storage | Membership query | Reverse lookup |
|:---|:---|:---|:---|
| `groupOfNames` (RFC 4519) | `member` attr on group | LDAP search for `(member=<dn>)` | scan all groups |
| `groupOfUniqueNames` | `uniqueMember` attr on group | LDAP search for `(uniqueMember=<dn>)` | scan all groups |
| `posixGroup` (RFC 2307) | `memberUid` attr (uid string) | LDAP search for `(memberUid=<uid>)` | scan all groups |
| AD `group` | `member` (DN) and `memberOf` (back-link, computed) | direct attr read | direct attr read |
| OpenLDAP memberOf overlay | back-link maintained automatically | direct attr read on user | search group's `member` |

### Nested Group Expansion

Group $G$ has members $M_1, \ldots, M_k$ where some $M_i$ are themselves groups. Effective member set:

$$\text{members}(G) = \text{users}(G.\text{member}) \cup \bigcup_{H \in \text{groups}(G.\text{member})} \text{members}(H)$$

Time to expand: depth-first traversal $O(|V| + |E|)$ where $V$ = visited groups, $E$ = total memberships.

### AD LDAP_MATCHING_RULE_IN_CHAIN

OID `1.2.840.113556.1.4.1941`:

```
(memberOf:1.2.840.113556.1.4.1941:=CN=Admins,OU=Groups,DC=example,DC=com)
```

Server-side recursive expansion. Cost: $O(D \cdot \bar{B})$ where $D$ = group nesting depth, $\bar{B}$ = average branching factor. Cycles detected and short-circuited.

### Token Bloat

Kerberos token includes all group SIDs. If a user is in $K$ groups (transitive), token size grows linearly. Default `MaxTokenSize = 12000` bytes; ~500 groups maximum before authentication fails.

---

## 24. RFC Cross-Reference Table

| RFC | Year | Title |
|:---|:---:|:---|
| 4510 | 2006 | LDAP Technical Specification Road Map |
| 4511 | 2006 | LDAP Protocol Operations |
| 4512 | 2006 | LDAP Directory Information Models |
| 4513 | 2006 | LDAP Authentication Methods and Security Mechanisms |
| 4514 | 2006 | String Representation of DNs |
| 4515 | 2006 | String Representation of Search Filters |
| 4516 | 2006 | LDAP URLs |
| 4517 | 2006 | Syntaxes and Matching Rules |
| 4518 | 2006 | Internationalized String Preparation |
| 4519 | 2006 | Schema for User Applications |
| 4520 | 2006 | IANA Considerations for LDAP |
| 4521 | 2006 | Considerations for LDAP Extensions |
| 4522 | 2006 | Binary Encoding Option |
| 4523 | 2006 | X.509 Certificate Schema |
| 4524 | 2006 | COSINE LDAP Schema |
| 4525 | 2006 | Modify-Increment Extension |
| 4526 | 2006 | Absolute True/False Filters |
| 4527 | 2006 | Read Entry Controls |
| 4528 | 2006 | Assertion Control |
| 4529 | 2006 | Requesting Attributes by Object Class |
| 4530 | 2006 | entryUUID Operational Attribute |
| 4531 | 2006 | Turn Operation |
| 4532 | 2006 | Who Am I? Extension |
| 4533 | 2006 | Content Synchronization Operation (syncrepl) |
| 4422 | 2006 | SASL Framework |
| 4505 | 2006 | SASL ANONYMOUS |
| 4616 | 2006 | SASL PLAIN |
| 4752 | 2006 | SASL GSSAPI |
| 5802 | 2010 | SASL SCRAM |
| 5803 | 2010 | SCRAM Stored Password Format |
| 5929 | 2010 | TLS Channel Bindings |
| 7677 | 2015 | SCRAM-SHA-256 |
| 2696 | 1999 | Paged Results Control |
| 2891 | 2000 | Server Side Sorting Control |
| 3045 | 2001 | Vendor Information |
| 3296 | 2002 | Named Subordinate References |
| 3672 | 2003 | Subentries |
| 3673 | 2003 | Manage DSA IT Control |
| 3866 | 2004 | Language Tag and Range |
| 3909 | 2004 | Cancel Operation |
| 3928 | 2004 | LCUP Client Update |
| 7642 | 2015 | SCIM Definitions, Overview, Concepts, Requirements |
| 7643 | 2015 | SCIM Core Schema |
| 7644 | 2015 | SCIM Protocol |

ITU-T standards referenced:

- X.500 (1988) — The directory: Overview of concepts, models, services
- X.501 — Models
- X.520 — Selected attribute types
- X.521 — Selected object classes
- X.690 — ASN.1 BER/DER/CER encoding rules

---

## 25. Performance Numbers (Production)

### Benchmarks (OpenLDAP 2.6, MDB backend, 8-core x86_64, NVMe)

| Operation | Cold cache | Warm cache |
|:---|:---:|:---:|
| Bind (simple, no TLS) | 0.4 ms | 0.4 ms |
| Bind (simple, TLS 1.3) | 4 ms | 4 ms |
| Bind (SCRAM-SHA-256) | 6 ms | 6 ms |
| Search BASE indexed | 2 ms | 0.2 ms |
| Search SUB indexed (R=100) | 8 ms | 1.2 ms |
| Search SUB unindexed (N=100k) | 850 ms | 90 ms |
| Add | 3 ms | 3 ms |
| Modify (1 attr) | 2 ms | 1.5 ms |
| Delete (leaf) | 2 ms | 2 ms |
| ModifyDN | 5 ms | 4 ms |

### Throughput

| Workload | Single conn | Pool=50 |
|:---:|:---:|:---:|
| 100% reads (indexed) | 2500 qps | 90000 qps |
| 100% reads (unindexed) | 12 qps | 580 qps |
| 80% read / 20% write | 1800 qps | 60000 qps |
| 100% writes | 800 qps | 18000 qps |

### Memory Footprint

| Entries | Indexes | RSS |
|:---:|:---:|:---:|
| 10k | eq+pres on 10 attrs | 80 MB |
| 100k | eq+pres+sub on 10 attrs | 600 MB |
| 1M | eq+pres+sub on 10 attrs | 5.5 GB |
| 10M | eq+pres+sub on 10 attrs | 48 GB |

### Active Directory at Scale

| Forest size | Replication time (full) | GC search latency |
|:---:|:---:|:---:|
| 100k users, 1 DC | n/a | 2 ms |
| 1M users, 5 DCs (LAN) | 30 minutes | 5 ms |
| 10M users, 50 DCs (WAN) | 4-12 hours | 30-100 ms |

---

## 26. ASCII Diagrams Reference

### Full DIT with all elements

```
                          ROOT (empty DN)
                             │
                ┌────────────┼────────────┐
            dc=com       dc=org        dc=net
                │            │            │
        dc=example    dc=example2     ...
                │
        ┌───────┼───────┬────────┐
      ou=people ou=groups ou=systems ou=services
        │            │
   ┌────┼────┐  ┌────┼────┐
 uid=A uid=B  cn=eng  cn=ops
 (inetOrgPerson)  (groupOfNames)
```

### syncrepl persist phase data flow

```
Supplier                          Consumer
  │                                  │
  │ accept LDAP write                │
  │ assign CSN                       │
  │ store entry                      │
  │ append to changelog              │
  │                                  │
  │ ─── SearchResultEntry ─────────▶│
  │     +SyncStateControl            │
  │     state=add/modify/delete      │
  │     entryUUID                    │
  │     cookie                       │
  │                                  │ ack via TCP
  │                                  │ store entry
  │                                  │ update local contextCSN
  │                                  │
  │ ─── SyncInfoMessage ────────────▶│
  │     newcookie                    │
  │                                  │ persist cookie
```

### Paged results cursor

```
Search invocation 1:
  Server creates cursor S1 = { filter_hash, base, scope, position=0 }
  Returns entries [0..99], cookie = encrypt(S1.id, S1.position=100)

Search invocation 2 with cookie:
  Server decrypts cookie → looks up cursor S1
  Resumes at position=100, returns entries [100..199]
  Updates cursor: position=200
  Returns cookie = encrypt(S1.id, 200)

Search invocation N (last):
  Returns remaining entries, cookie = "" (empty)
  Server frees cursor S1
```

### SCRAM channel binding

```
       TLS 1.3 channel
   Client ◀═════════════▶ Server
          │ tls-unique = H(handshake transcript)
          │
          ▼
   SCRAM exchange
   Client sends: ... cb=tls-unique:<base64> ...
   Server verifies cb matches its own tls-unique
          │
          ▼
   If MITM has different TLS handshake → cb mismatch → AUTH FAIL
```

---

## 27. Glossary

| Term | Definition |
|:---|:---|
| ABNF | Augmented Backus-Naur Form (RFC 5234) |
| ACE | Access Control Entry |
| ACI | Access Control Instruction |
| ASN.1 | Abstract Syntax Notation One |
| BER | Basic Encoding Rules |
| BTree / B+tree | Balanced search tree |
| CSN | Change Sequence Number |
| DAP | Directory Access Protocol (X.500) |
| DC | Domain Controller |
| DER | Distinguished Encoding Rules |
| DIT | Directory Information Tree |
| DN | Distinguished Name |
| DSE | DSA-Specific Entry |
| FSMO | Flexible Single Master Operations |
| GC | Global Catalog |
| HKDF | HMAC-based Key Derivation Function |
| IDL | ID List (matched entry IDs) |
| KCC | Knowledge Consistency Checker |
| LDAP | Lightweight Directory Access Protocol |
| LDIF | LDAP Data Interchange Format |
| LMDB | Lightning Memory-Mapped Database |
| MMR | Multi-Master Replication |
| OID | Object Identifier |
| OLC | OpenLDAP Online Configuration (cn=config) |
| PBKDF2 | Password-Based Key Derivation Function 2 |
| RDN | Relative Distinguished Name |
| RFC | Request for Comments |
| SACL | System Access Control List |
| SASL | Simple Authentication and Security Layer |
| SCRAM | Salted Challenge Response Authentication Mechanism |
| SID | Security Identifier (Windows) |
| SSF | Security Strength Factor (OpenLDAP) |
| TLV | Type-Length-Value |
| TGT | Ticket-Granting Ticket (Kerberos) |
| UPN | User Principal Name |
| USN | Update Sequence Number (AD) |
| UUID | Universally Unique Identifier |
| VLV | Virtual List View |

---

## 28. References (Full)

### IETF RFCs

- RFC 4510-4533 — Complete LDAPv3 specification suite
- RFC 4422 — SASL framework
- RFC 4505, 4616, 4752 — SASL mechanisms (ANONYMOUS, PLAIN, GSSAPI)
- RFC 5802, 5803, 7677 — SCRAM family
- RFC 5869 — HKDF
- RFC 5929 — TLS Channel Bindings
- RFC 6125 — Server identity verification within PKIX
- RFC 2696 — Paged Results Control
- RFC 2891 — Server-Side Sorting Control
- RFC 3672 — Subentries
- RFC 3909 — Cancel Operation
- RFC 7642-7644 — SCIM 2.0
- RFC 8446 — TLS 1.3
- RFC 5234 — ABNF

### ITU-T

- ITU-T X.500 — The directory: Overview
- ITU-T X.501 — Models
- ITU-T X.520 — Selected attribute types
- ITU-T X.521 — Selected object classes
- ITU-T X.680 — ASN.1 specification
- ITU-T X.690 — ASN.1 BER/DER/CER

### Books

- Howes, T., Smith, M., Good, G. — "Understanding and Deploying LDAP Directory Services" (2nd ed., 2003)
- Butcher, M. — "Mastering OpenLDAP" (2007)
- Carter, G. — "LDAP System Administration" (2003)
- Voglmaier, R. — "The ABCs of LDAP" (2003)
- Donley, C. — "LDAP Programming, Management and Integration" (2003)

### Implementation References

- OpenLDAP Software 2.6 Administrator's Guide (https://www.openldap.org/doc/admin26/)
- 389 Directory Server Documentation
- Microsoft Active Directory Technical Reference (https://learn.microsoft.com/en-us/windows/win32/ad/)
- ApacheDS Documentation (Apache Foundation)

### Foundational Theory

- Comer, D. — "The Ubiquitous B-Tree" ACM Computing Surveys 11(2), 1979
- Lamport, L. — "Time, Clocks, and the Ordering of Events in a Distributed System" CACM 21(7), 1978
- Krawczyk, H., Bellare, M., Canetti, R. — RFC 2104 (HMAC), 1997
- Kaliski, B. — RFC 2898 (PKCS #5: PBKDF2), 2000

---

## Prerequisites

- tree-structures, boolean-algebra, probability, big-o-notation, btree-indexes, networking-fundamentals, asn1-encoding, sasl-framework, hmac-and-pbkdf2, distributed-systems, vector-clocks, transport-layer-security
