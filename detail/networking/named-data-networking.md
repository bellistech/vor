# Named Data Networking — Deep Dive

> *Math-heavy companion to `ramp-up/named-data-networking-eli5`. Every TLV byte, every PIT
> equation, every cache-replacement formula, every signature chain, written for the
> terminal-bound reader who needs the number, the encoding, and the worked example without
> opening a browser. Sources: NDN Packet Format v0.3, NFD Developer's Guide, RFC 8569
> (CCNx Semantics), RFC 8609 (CCNx Wire Format), RFC 7945 (ICN Research Challenges),
> Jacobson et al. "Networking Named Content" (CoNEXT 2009), and named-data.net technical
> reports TR-0001 through TR-0024.*

---

## Architecture and Data Plane

### Two-packet universe

NDN replaces TCP/IP's host-to-host conversation with a request/response pull where
**every byte on the wire is one of two packet types**:

```
+-----------------------------+         +------------------------------+
|        Interest             |  --->   |         Data                 |
|                             |         |                              |
|  Name (hierarchical)        |         |  Name (matches the Interest) |
|  CanBePrefix? MustBeFresh?  |         |  MetaInfo (FreshnessPeriod,  |
|  ForwardingHint, Nonce      |  <---   |    ContentType, FinalBlockId)|
|  InterestLifetime           |         |  Content (≤ 8 KiB typical)   |
|  HopLimit (optional)        |         |  Signature (every Data)      |
+-----------------------------+         +------------------------------+
```

There is no source address. There is no destination address. The **name in the Interest is
the destination**, and the name in the matching Data is the routing back-pointer. Routers
keep state for every in-flight Interest, and that state — not a header field — is what
returns the Data along the reverse path.

### Three-table forwarder

Every NDN router (the canonical implementation is **NFD**, the NDN Forwarding Daemon)
runs a three-table state machine for each arriving packet:

```
                +-----------------+
                |   Face (in)     |   "Face" = link abstraction (UDP/TCP/Eth/WebSocket)
                +--------+--------+
                         |
                         v
   Interest       +-------+-------+        Data           +-------+-------+
   arrives -----> | Content Store |  <-- arrives ------>  | Content Store |
                  |     (CS)      |                       |   (insert)    |
                  +-------+-------+                       +-------+-------+
                          | miss                                  |
                          v                                       v
                  +-------+-------+                       +-------+-------+
                  |  PIT lookup   |                       |  PIT lookup   |
                  +-------+-------+                       +-------+-------+
                          |                                       |
            (no match) -->| (existing entry: aggregate)           | (match: forward
                          v                                        |  to all in-faces)
                  +-------+-------+                                v
                  |  FIB lookup   |                       (drop entry, store in CS
                  +-------+-------+                        if FreshnessPeriod > 0)
                          |
                          v
                +---------+---------+
                | Forwarding strategy
                | best-route | multicast | asf | ncc | self-learning
                +---------+---------+
                          |
                          v
                +---------+---------+
                |   Face (out)      |
                +-------------------+
```

- **CS** (ContentStore): packet-level cache, indexed by full or prefix-of name.
- **PIT** (Pending Interest Table): every unsatisfied Interest, indexed by name, valued
  by the set of incoming faces and a nonce log for loop detection.
- **FIB** (Forwarding Information Base): name prefix → outgoing-face list (multi-path),
  populated by routing (NLSR) or static config (`nfdc route add`).

The three tables are the data plane. Routing protocols, application registration, signing
keys, and caching policy all sit above them.

### Face abstraction

A **face** is NDN's `socket`-like primitive — a bidirectional link to a peer or to a local
application. Examples:

```
face://udp4://192.0.2.10:6363       # UDP unicast
face://ether://[01:00:5e:01:01:01]/eth0   # Ethernet multicast
face://tcp6://[2001:db8::1]:6363
face://unix:///run/nfd.sock         # local app
face://ws://router.example/nfd
face://internal://                  # NFD's management
```

Each face has a unique 32-bit `FaceId`. The PIT stores `FaceId`s; the FIB stores
`(FaceId, cost)` tuples; the CS does not need faces (it keeps Data, not flow state).

---

## Name Encoding (TLV)

### TLV primer

NDN's wire format is **Type-Length-Value (TLV)** with variable-width type and length
fields, mirroring (and predating, in spirit) X.690 BER but tuned for sub-µs parsing.

```
+----------+----------+-----------+
|   TYPE   |  LENGTH  |   VALUE   |
+----------+----------+-----------+
   varint     varint     LENGTH
                         bytes
```

Variable-length integer rules (for both TYPE and LENGTH):

| First byte         | Total octets | Encoded range                     |
|:-------------------|:------------:|:----------------------------------|
| `0x00..0xFC`       | 1            | 0 .. 252                          |
| `0xFD` + 2 octets  | 3            | 253 .. 65 535                     |
| `0xFE` + 4 octets  | 5            | 65 536 .. 4 294 967 295           |
| `0xFF` + 8 octets  | 9            | 4 294 967 296 .. 2⁶⁴ − 1          |

So a TYPE under 253 fits in one byte; a 64-byte payload field's `LENGTH=64` fits in one
byte. Most names fit in `≤ 1 + 1` of overhead per component.

### Top-level types

```
INTEREST                = 0x05
DATA                    = 0x06
NAME                    = 0x07
GENERIC_NAME_COMPONENT  = 0x08
IMPLICIT_SHA256_DIGEST  = 0x01
PARAMETERS_SHA256_DIGEST= 0x02
KEYWORD_NAME_COMPONENT  = 0x20
SEGMENT_NAME_COMPONENT  = 0x32
BYTE_OFFSET             = 0x34
VERSION                 = 0x36
TIMESTAMP               = 0x38
SEQUENCE_NUM            = 0x3A
META_INFO               = 0x14
CONTENT                 = 0x15
SIGNATURE_INFO          = 0x16
SIGNATURE_VALUE         = 0x17
NONCE                   = 0x0A
INTEREST_LIFETIME       = 0x0C
HOP_LIMIT               = 0x22
APPLICATION_PARAMETERS  = 0x24
INTEREST_SIGNATURE_INFO = 0x2C
INTEREST_SIGNATURE_VALUE= 0x2E
CAN_BE_PREFIX           = 0x21
MUST_BE_FRESH           = 0x12
FORWARDING_HINT         = 0x1E
FRESHNESS_PERIOD        = 0x19
CONTENT_TYPE            = 0x18
FINAL_BLOCK_ID          = 0x1A
SIGNATURE_TYPE          = 0x1B
KEY_LOCATOR             = 0x1C
KEY_DIGEST              = 0x1D
```

### Hierarchical names

A name is a TLV of type `0x07` whose value is an ordered sequence of NameComponents. The
**delimiter is structural, not textual** — the conventional URI form (`/wikipedia/article/quic`)
is purely a human convenience. On the wire, slashes never appear.

```
URI form:  /wikipedia/article/quic
Wire form (TLV):

  07 0E                                   ; NAME, length 14
     08 09 wikipedia                      ; component "wikipedia"  (08, len 9, ASCII)
     08 07 article                        ; component "article"    (08, len 7)
     08 04 quic                           ; component "quic"       (08, len 4)

   total: 2 + (2+9) + (2+7) + (2+4) = 28 bytes? Let's count carefully:
   - NAME header: 07 0E -> 2 bytes
   - "wikipedia": 08 09 + 9 = 11 bytes
   - "article":   08 07 + 7 = 9 bytes
   - "quic":      08 04 + 4 = 6 bytes
   ---------------------------------
   total payload = 11 + 9 + 6 = 26
   But length 0E = 14 ?  That's wrong above; let's redo.

  Fixed:
  07 1A                                   ; NAME, length 26 (0x1A)
     08 09 'w' 'i' 'k' 'i' 'p' 'e' 'd' 'i' 'a'
     08 07 'a' 'r' 't' 'i' 'c' 'l' 'e'
     08 04 'q' 'u' 'i' 'c'

   total on wire = 2 + 26 = 28 bytes.
```

Order is canonical: components must appear in URI order, never sorted. Two names are equal
if and only if they are byte-equal.

### NameComponent types

| TYPE   | Hex  | Name                              | Semantics                                         |
|:-------|:-----|:----------------------------------|:--------------------------------------------------|
| 0x01   | 1    | ImplicitSha256DigestComponent     | SHA-256(packet) → exact-match hash, never forwarded by name match alone |
| 0x02   | 2    | ParametersSha256DigestComponent   | SHA-256(ApplicationParameters TLV)                |
| 0x08   | 8    | GenericNameComponent              | arbitrary octets (UTF-8 by convention, not rule)  |
| 0x20   | 32   | KeywordNameComponent              | typed marker: `<keyword>`, e.g. "metadata"        |
| 0x32   | 50   | SegmentNameComponent              | NonNegativeInteger segment number (chunked file)  |
| 0x34   | 52   | ByteOffsetNameComponent           | byte-offset variant for stream segmentation       |
| 0x36   | 54   | VersionNameComponent              | NonNegativeInteger version stamp (often Unix ms)  |
| 0x38   | 56   | TimestampNameComponent            | Unix microseconds                                 |
| 0x3A   | 58   | SequenceNumNameComponent          | per-stream sequence                               |

NonNegativeInteger encoding inside a typed component value is **big-endian, minimum
length** — `0x05` encodes as `01 05`, `0x0102` encodes as `02 01 02`, no leading zeros.

### ImplicitSha256DigestComponent in practice

When a consumer wants to demand a *specific* Data packet (not "any matching name") it
appends an ImplicitSha256DigestComponent with the SHA-256 of the entire Data packet:

```
/wikipedia/article/quic/<implicit-sha256:0x4b8c…ee01>
                       ^-- 0x01 0x20 + 32 bytes of digest
```

This pins the producer's signature *and* content. It is the strongest reference NDN
provides — equivalent to a Git object hash.

### ParametersSha256DigestComponent

For "signed Interests" or interests carrying parameters (`AppParameters` TLV), the
parameters' SHA-256 is embedded in the name as a digest component so the name itself
covers the parameter bytes. This is how NDN gets request-side immutability without an
extra header: changing parameters changes the name.

### Worked TLV: minimal Interest

```
05 18                              ; INTEREST, len 24
   07 14                           ; NAME, len 20
      08 06 'h' 'e' 'l' 'l' 'o'    ; len-mismatch demo? No: 'h'..'o' = 5 chars; 08 05 then.

  Corrected:
  05 17                            ; INTEREST, len 23
     07 13                         ; NAME, len 19
        08 05 h e l l o            ; "hello"             (07 bytes)
        08 0A "ndn-test-1"         ; len 10              (12 bytes)
     0A 04 11 22 33 44             ; NONCE, 4 random octets    (6 bytes)

   Total on wire: 2 + 23 = 25 bytes.
```

Every Interest **MUST** carry a 4-byte Nonce. NFD uses it for loop detection and for
deduplication when a multi-homed face receives the same Interest twice.

### Worked TLV: minimal Data

```
06 38                              ; DATA, len 56
   07 13                           ; NAME, len 19  (same /hello/ndn-test-1)
      08 05 h e l l o
      08 0A n d n - t e s t - 1
   14 03                           ; META_INFO, len 3
      19 01 78                     ;   FRESHNESS_PERIOD = 120 (0x78) ms
   15 0B                           ; CONTENT, len 11
      "hello world"
   16 03                           ; SIGNATURE_INFO
      1B 01 00                     ;   SIGNATURE_TYPE = 0 (DigestSha256)
   17 14                           ; SIGNATURE_VALUE, len 20
      <SHA-256(name||metainfo||content||sig_info), truncated for example>
```

The signature covers exactly: `Name || MetaInfo || Content || SignatureInfo`. Not the
top-level `06 38`. Not the `SignatureValue` itself. This is the canonical signed-buffer
range for every Data verifier.

---

## Forwarding Pipeline Math

### Per-Interest cost

Let:

- `c` = number of name components (depth of the name)
- `|CS|` = number of Data objects cached (≤ `MAX_CS_BYTES / avg_data_size`)
- `|PIT|` = number of pending Interests
- `|FIB|` = number of FIB prefixes
- `b_avg` = average length of a NameComponent (octets)

Pipeline costs per arriving Interest (NFD's actual data structures: NameTree + hash maps):

```
T_total = T_decode  + T_CS_lookup  + T_PIT_lookup  + T_FIB_lookup  + T_strategy
        ≈ O(c·b_avg) + O(c)         + O(c)          + O(c)          + O(1..k)

  where k = number of nexthops returned by FIB lookup
        all hashes are FNV-1a or CityHash on each component prefix, O(b_avg) each
```

Because every lookup is a longest-prefix match on names, the wall-clock cost is
**linear in name depth**, not in table size. NFD measures ~80 ns/component on a 2024
x86 server, so a 6-component Interest pipeline runs ~500 ns; NDN-DPDK on dedicated
cores hits ~30 ns/component for ~180 ns total.

### Cache-hit math

Define:

- `λ` = aggregate Interest arrival rate at a router (Interests/s)
- `h` = cache hit ratio (CS satisfaction probability)
- `μ_up` = upstream link rate (Interests/s the router actually emits)

Then:

```
μ_up = λ · (1 − h)         [each missed Interest produces one upstream Interest]

Bandwidth saved at upstream = λ · h · S_data
   where S_data = average Data packet size in bytes
```

For a Zipf popularity distribution with skew `α` over `M` distinct objects and a CS of
`C` slots, the **Che approximation** for hit ratio is:

```
h(C) ≈ 1 − exp( −λ · t_C / λ )      [for LRU under Zipf]

where t_C is the "characteristic time" satisfying:
   Σ_i (1 − exp(−λ_i · t_C)) = C
   λ_i = λ · i^(−α) / Σ_j j^(−α)
```

Closed-form for α > 1 and large M:

```
h ≈ 1 − (C / M)^(α − 1) / (α · ζ(α))
```

Practical implication: for video catalogues with α ≈ 0.8 and 100 k titles, a 1 %-of-catalogue
CS achieves ~25 % hit ratio at edge routers; at α ≈ 1.2, the same CS achieves > 50 %.

### Multi-router hierarchy

For a tree of `L` levels with cache hit ratio `h_l` at level `l` (consumer-side level 0,
producer-side level `L`):

```
  λ_l = λ_0 · ∏_{i=0..l−1} (1 − h_i)

  Aggregate origin offload = 1 − ∏_{l=0..L−1} (1 − h_l)
```

Three levels at h=0.4 each ⇒ origin sees `(0.6)³ = 0.216` of consumer traffic — 78.4 %
offload. A single CDN node with one CS slice at h=0.7 offloads only 70 %.

---

## PIT Mechanics

### Aggregation arithmetic

The PIT is what makes NDN multicast for free. When **N consumers** request the *same name*
within the InterestLifetime window:

```
N consumers send N Interests
   |
   v
Edge router receives N
   - 1st: PIT miss   -> create entry,  forward 1 Interest upstream
   - 2nd..Nth: PIT hit -> append in-face, do NOT forward upstream

  Upstream Interests sent  = 1
  Upstream Data received   = 1
  Downstream Data sent     = N (one per recorded in-face)
  Aggregation factor       = N : 1 = N
```

This is hop-by-hop; in a tree of fan-out `f` per level, the **upstream Interest count at
level `l`** is bounded by `f^(L−l)` consumers but only **`min(f^(L−l), 1)` upstream
Interests per name** if all levels can aggregate. Ideal case: `N` consumers issue
exactly **`L` upstream Interests** (one per level) regardless of `N`.

### Loop detection via Nonce

Every Interest carries a 4-byte random `Nonce`. The PIT entry stores the set of nonces
seen for the name. On Interest arrival:

```
if PIT[name].nonces.contains(nonce):
    drop("loop detected")
elif len(PIT[name].nonces) > MAX_NONCES_PER_PIT_ENTRY:
    apply LRU eviction inside the entry
else:
    PIT[name].nonces.insert(nonce)
    PIT[name].in_faces.insert(arriving_face)
```

A loop is fundamentally a *same Interest, same nonce, same name* re-arrival. Because
nonce is 32 bits, the probability of a spurious collision over a 1-second lifetime at
1 Mpps is `≈ 1 − exp(−10⁶² / 2⁶⁴) ≈ 5 × 10⁻⁸` — negligible.

### Lifetime-driven eviction

Each PIT entry has an absolute expiry `T_exp = T_arrival + InterestLifetime`. Default
`InterestLifetime = 4000 ms`; range 0 < L ≤ ~268 s (varint cap). Eviction is event-driven:

```python
# pseudocode of NFD's PIT scheduler
heap = MinHeap(key=lambda e: e.expiry)

def insert(entry): heap.push(entry)

def tick(now):
    while heap.peek() and heap.peek().expiry <= now:
        e = heap.pop()
        for f in e.in_faces:
            send_nack(f, name=e.name, reason="NoRoute" or "Timeout")
        remove_pit_entry(e)
```

### Memory bounds

NFD enforces both per-face and global caps:

```
PIT_capacity_global    = N_max_pending          (default 65536)
PIT_capacity_per_face  = N_max_per_face         (default 32 per face on multicast Eth)

if Σ PIT entries from face_i > PIT_capacity_per_face:
    drop new Interest with NACK reason "Congestion"
```

An attacker flooding distinct names tries to exhaust the PIT — this is the **Interest
Flooding Attack (IFA)**. Defenses:

```
Token-bucket per in-face: rate = R_i, burst = B_i
Drop if  arrival_rate(face_i) > R_i  AND  PIT_size(face_i) > θ_i

NDN Interest-flooding mitigation (Compagno et al., LCN 2013):
  satisfaction_ratio_i = data_returned_i / interests_sent_i
  if satisfaction_ratio_i < τ:  rate-limit face_i
```

Empirically, dropping at `τ = 0.1` recovers throughput within 3× InterestLifetime.

---

## Caching

### Replacement policies

CS replacement is policy-pluggable. Common choices and their cost-per-access (assuming
hash index for `Name → entry`):

| Policy   | Lookup | Insert | Eviction | Strength                         | Weakness                |
|:---------|:-------|:-------|:---------|:---------------------------------|:------------------------|
| LRU      | O(1)   | O(1)   | O(1)     | Fast, correct under recency      | Scan-resistant: no      |
| LFU      | O(1)*  | O(1)*  | O(log n) | Long-tail popularity             | Stale hot items linger  |
| 2Q       | O(1)   | O(1)   | O(1)     | Resists scan, two-tier admission | Tunable thresholds      |
| ARC      | O(1)   | O(1)   | O(1)     | Adaptive between recency/freq.   | Patent-encumbered       |
| LIRS     | O(1)   | O(1)   | O(1)     | Reuse-distance aware             | Complex bookkeeping     |
| Priority | O(log n) | O(log n) | O(log n) | NDN-specific: keep signed by   | Heap maintenance cost   |
|          |        |        |          | trust score                       |                         |

\* with a frequency map and bucket lists.

### Cache decision policies (where to cache)

Once a Data packet traverses a path, **which routers should cache it**? The answer
governs cooperative cache hit rates:

| Policy        | Rule                                                                 | Behaviour               |
|:--------------|:---------------------------------------------------------------------|:------------------------|
| LCE           | Leave Copy Everywhere — every router along path stores the Data      | High redundancy         |
| LCD           | Leave Copy Down — store only one hop down from where you got the hit | Pulls hot content edge  |
| MCD           | Move Copy Down — like LCD but evict from the upstream                | Even more aggressive    |
| ProbCache     | Cache at router `r` with probability `p_r = (depth−r)/depth · w_r`    | Diversifies caches      |
| Cache-Less    | Cache only if name appears in a "popular" set                         | Lowest memory           |
| Off-Path      | Cache only at producer-side aggregation points                        | Like a CDN              |

ProbCache analytic hit rate (Psaras et al., ICN 2012) under Zipf:

```
h_total ≈ 1 − ∏_{r=1..L} (1 − p_r · q_r)

where q_r is local hit-given-cached prob, ≈ Che approx for that router's CS.
```

NFD ships LCE as default with LRU; production deployments tune to ProbCache+2Q for
catalogue-style workloads.

### Freshness math

Each Data packet's MetaInfo can include `FreshnessPeriod` (ms). Combined with the
Interest selector `MustBeFresh`, the CS satisfaction rule is:

```
def cs_can_satisfy(interest, data):
    if not name_match(interest.name, data.name): return False
    if interest.must_be_fresh:
        age = now() - data.cached_at
        if age > data.freshness_period_ms: return False
    return True
```

Freshness only applies if the Interest sets the selector. By default Interests do *not*
set `MustBeFresh`, so a year-old cached Data with `FreshnessPeriod=0` satisfies new
Interests indefinitely — exactly what you want for immutable content like
`/wikipedia/article/quic/v=20251002083500`.

### Cache hit prediction model

Combining everything: for a hierarchy with caches `C_1..C_L`, Zipf popularity α, M
objects, request rate λ:

```
T_C(C_l) = solution of  Σ_i (1 − exp(−λ_l_i · T)) = C_l
h_l      = 1 − exp(−λ_l · T_C(C_l) / λ_l_total)
λ_{l+1}  = λ_l · (1 − h_l)
```

Iterate top-down. Closed-form for α=1 (true Zipf):

```
h_l ≈ ln(C_l / e) / ln(M)        [pure Zipf, large M]
```

So doubling C buys logarithmic hit-rate growth — there's no "knee" past ~0.1·M.

---

## Forwarding Strategies

NFD chooses an **outgoing face** from the FIB nexthop set via a per-prefix strategy. The
strategy interface is:

```cpp
// NFD strategy interface, simplified
class Strategy {
    virtual void afterReceiveInterest(face, interest, pitEntry) = 0;
    virtual void afterReceiveData(face, data, pitEntry) = 0;
    virtual void afterReceiveNack(face, nack, pitEntry) = 0;
    virtual void beforeExpirePendingInterest(pitEntry) = 0;
};
```

### best-route

Send to the FIB nexthop with the **lowest cost** that has not been used for this PIT
entry already. If all have been tried, drop. Costs come from NLSR.

```
nexthops = FIB[longest_match(name)]
unused = nexthops − pitEntry.tried_faces
if unused: send to argmin(cost ∈ unused)
else:      strategy returns NACK("NoRoute")
```

### multicast

Send to **all** nexthops in parallel. Simple, expensive on bandwidth, ideal for short-lived
discovery (e.g. NLSR Hello) or when the topology has many viable paths.

### asf (Adaptive SRTT-based Forwarding)

Maintain per-(prefix, face) round-trip estimate. On each Data return, update SRTT and
RTTVAR Jacobson/Karels-style:

```
sample = T_data_return − T_interest_send

SRTT  ← α · SRTT + (1 − α) · sample        [α = 0.875, i.e. shift-3 EWMA]
RTTVAR ← β · RTTVAR + (1 − β) · |SRTT − sample|   [β = 0.75]
RTO   = SRTT + 4 · RTTVAR

best = argmin_face SRTT[prefix, face]
probe_period = max(60s, 5 · SRTT[best])
```

ASF periodically *probes* alternative faces (at `probe_period`) to discover better paths
without committing all traffic. The probe budget is bounded by `probe_period^−1` per
prefix per face, so steady-state bandwidth waste is `≤ S_interest / probe_period` ≈
40 B / 60 s = 0.66 B/s per (prefix, face). Negligible.

### NCC (Named-Content Centric)

Earlier strategy from CCNx-1.0 era. Maintains per-face "predicted" arrival times. Mostly
of historical interest; ASF supersedes it on NFD ≥ 0.6.

### self-learning

Run multicast for the first Interest; Data return implicitly populates a routing entry
keyed on the in-face. After learning, subsequent Interests use best-route. Useful for
ad-hoc IoT meshes where there is no NLSR.

```
state[prefix].learned[in_face_of_returning_data] = cost_estimate
on next Interest for prefix:
    if state[prefix].learned: send best-route
    else: multicast
```

### Per-prefix selection

Operators bind strategies to prefixes:

```bash
nfdc strategy set /wikipedia                  /localhost/nfd/strategy/best-route
nfdc strategy set /campus/sensors/discovery   /localhost/nfd/strategy/multicast
nfdc strategy set /video/live                 /localhost/nfd/strategy/asf
nfdc strategy set /lan/zeroconf               /localhost/nfd/strategy/self-learning
```

Strategy is a longest-prefix property: more-specific prefixes override.

---

## Security Model

### Per-Data signature, mandatory

Unlike TLS which secures channels, NDN secures **objects**. Every Data packet carries a
signature; verification is independent of where the bytes came from. The implication is
profound: caches are safe by construction. A malicious cache can serve only correctly
signed bytes or get rejected.

The signed buffer is the byte range:

```
sig_buf = encode(Name) || encode(MetaInfo) || encode(Content) || encode(SignatureInfo)
```

The SignatureValue covers `sig_buf` — but not itself.

### SignatureType registry

| Type | Hex  | Algorithm                     | Public key needed? | Use                                |
|:-----|:-----|:------------------------------|:-------------------|:-----------------------------------|
| 0    | 0x00 | DigestSha256                  | no                 | integrity-only, in-network         |
| 1    | 0x01 | SignatureSha256WithRsa        | RSA-2048+          | producer authentication            |
| 3    | 0x03 | SignatureSha256WithEcdsa      | ECDSA P-256/P-384  | producer authentication, smaller   |
| 4    | 0x04 | SignatureHmacWithSha256       | shared HMAC key    | bilateral channel, e.g. NLSR Sync  |
| 5    | 0x05 | SignatureEd25519              | Ed25519 PK         | modern asymmetric                  |
| 6    | 0x06 | SignatureSha256WithEcdsaSha3  | reserved/experimental |                                  |

DigestSha256 has zero CPU cost on the verifier (just SHA-256(packet)) but provides no
authenticity. Used for ContentStore-internal integrity bookkeeping or for ephemeral
discovery packets where signing would dominate cost.

### KeyLocator

The `KeyLocator` TLV in `SignatureInfo` tells the verifier where to find the verification
key. It contains exactly one of:

```
KeyLocator = Name(/path/to/cert)            [most common]
           | KeyDigest(SHA-256 of public key bytes)
```

By embedding a *name*, KeyLocator turns the certificate into another NDN object — fetch it
with an Interest. The whole cert chain is content-routable.

### Trust schema math

A trust schema is a regex-like grammar over name components that says: "for any Data
matching pattern A, its signing cert's name must match pattern B". Example schema for a
campus IoT system:

```
rule "data signed by site cert":
    packet:   /campus/<dept>/<sensor>/<timestamp>
    signer:   /campus/<dept>/_KEY/<key-id>

rule "site cert signed by campus root":
    packet:   /campus/<dept>/_KEY/<key-id>
    signer:   /campus/_ROOT/_KEY/<key-id>

rule "campus root":
    packet:   /campus/_ROOT/_KEY/<key-id>
    signer:   self
```

Verification halts when:

1. signer == self at a known trust anchor (success), or
2. no matching rule (failure), or
3. cert unreachable (timeout, failure).

The trust schema is **expressed as NDN names**; the verifier evaluates the rules as a
regex match over name component arrays. Each rule is `O(c)` where `c` is component count.
Total verification cost is `O(c · D)` where `D` is depth of the cert chain (typically 3).

```python
def verify(packet, trust_schema, anchors):
    while not is_anchor(packet, anchors):
        rule = trust_schema.match(packet.name)
        if rule is None: return FAIL("no rule")
        if not rule.signer_pattern.matches(packet.signer_name): return FAIL("schema mismatch")
        cert = fetch(packet.signer_name)
        if not cert: return FAIL("cert unreachable")
        if not crypto_verify(packet.sig_buf, packet.sig_value, cert.pub_key):
            return FAIL("crypto mismatch")
        packet = cert            # walk one level toward the root
    return OK
```

---

## Routing — NLSR

**NLSR** (Named-data Link State Routing) is the canonical NDN routing protocol. It runs
*as an NDN application* — every NLSR message is an Interest or Data — yet it computes
prefix-to-nexthop mappings the FIB consumes.

### Data structures

```
LSA types:
   AdjacencyLSA   - which routers I am connected to, with link cost
   CoordinateLSA  - my hyperbolic coordinate (r, θ) for HR
   NameLSA        - which name prefixes I serve

LSDB = set of all LSAs received from all routers, keyed by (router_name, lsa_type, version)
```

### Cost computation

Each adjacency has an integer `cost` configured operator-side (default 10). Path cost is
the sum along the path. NLSR computes shortest-path with Dijkstra:

```
For each destination prefix p:
    nexthops = []
    for each path P from self to producer(p):
        cost_P = Σ link.cost for link in P
    nexthops = top-K paths by cost_P (default K=3, configurable)
    install in FIB:  /p -> [(face_to_first_hop_of_P, cost_P) for P in nexthops]
```

NLSR supports **multi-path** out of the box: best-route uses cost ordering; ASF lets
SRTT pick among them.

### Hyperbolic option

For ultra-large-scale topologies (10⁶ nodes), NLSR can use **hyperbolic routing**
(Greedy Forwarding in the hyperbolic plane). Each node has a coordinate `(r, θ)`; the
nexthop minimizes hyperbolic distance to destination:

```
d_hyp((r1,θ1), (r2,θ2)) = arccosh( cosh(r1)·cosh(r2) − sinh(r1)·sinh(r2)·cos(Δθ) )
```

Greedy forwarding is **scalable** (O(1) state per node, no LSDB) but **suboptimal** —
empirically reaches destination in 95-99 % of cases on Internet-scale topologies. NLSR
runs HR alongside link-state and falls back to LS on greedy failure.

### LSA flooding via Sync

Instead of flooding LSAs as in OSPF, NLSR uses an **NDN Sync** protocol (PSync or
ChronoSync) over the LSDB namespace. Routers subscribe to `/<network>/lsdb` and any
update produces a sync digest change which other routers Interest-fetch. Convergence
time:

```
t_converge ≈ d_max · (T_sync_interval + RTT_max)
```

with `T_sync_interval` typically 1 s and `RTT_max` ~50 ms in a campus network, ⇒
converges in seconds across ten-hop diameter.

### Adjacency formation timing

```
t=0     Hello Interest sent on each face: /<network>/<router>/HELLO
t≈0.1s  Neighbor responds with Hello Data including its own router-name + cost
t≈0.2s  Both sides emit AdjacencyLSA via Sync
t≈0.4s  Sync digest propagates one hop
t≈t_converge Routers run Dijkstra, install FIB
```

NFD then consumes NLSR's RIB (Routing Information Base) updates and writes the FIB.

---

## Sync Protocols

Sync (introduced as ChronoSync, refined as PSync, generalized as State-Vector Sync) is
NDN's dataset-synchronization primitive. It lets `N` peers reach eventual consistency
on a shared set of names without a central coordinator.

### ChronoSync (digest tree)

Each peer maintains a tree where leaves are `(participant, sequence_number)` pairs.
The root digest is SHA-256 over all leaves in canonical order.

```
Peer i sends sync Interest:  /<topic>/sync/<digest_state_i>
   if digest_state_j seen before by responder (peer k):
       k replies with Data containing the diff (set of new (participant, seq) leaves)
   else:
       k drops; Interest times out; peer i retries with subset digests
```

Cost per update:

```
Memory:  O(N) per peer  (one leaf per participant)
Network: O(1) round-trip in the common case (digest match)
         O(N) bytes when fetching the diff
Latency: ≤ 1 RTT for any peer to learn about update
```

### PSync (BF-based)

PSync uses an **Invertible Bloom Lookup Table** for the publisher set and a
**Bloom Filter** for the subscriber's interest set. The IBLT is the secret sauce:

```
IBLT structure:
  k hash functions, m cells
  each cell: count, key_xor, value_xor

To insert  (k, v):
  for h ∈ hash_funcs:
    cells[h(k)].count += 1
    cells[h(k)].key_xor ^= k
    cells[h(k)].value_xor ^= v

To compute set difference IBLT_A − IBLT_B:
  cells_diff[i].count    = A.cells[i].count - B.cells[i].count
  cells_diff[i].key_xor  = A.cells[i].key_xor XOR B.cells[i].key_xor
  cells_diff[i].value_xor= A.cells[i].value_xor XOR B.cells[i].value_xor

To "list" the diff:
  while exists cell with |count|==1:
    (k, v) = (key_xor, value_xor) of that cell
    if count == 1: A has it, B doesn't.
    if count ==−1: B has it, A doesn't.
    remove (k, v) from each h(k)
```

The IBLT decodes successfully iff the symmetric difference size `d ≤ m / 1.5` (Eppstein-Goodrich
bound for k=3 hashes). PSync sets `m = 80` cells → comfortable for `d ≤ 53` updates per
sync round. Beyond that, fall back to full-state transfer.

### Throughput formulas

For a topic with `λ` updates/s across `N` peers, sync interval `T`:

```
Per-peer outbound bandwidth  = λ · S_data + (1/T) · S_iblt
                              ≈ λ · S_data + 80 · 24 / T  bytes/s    (S_cell ≈ 24)
                              = λ · S_data + 1920 / T

Convergence time  ≤ T + RTT
Storage           = O(W · S_data)  where W is window of unexpired updates
```

For `λ = 10`, `S_data = 1 KiB`, `T = 1 s`: outbound ≈ 11.9 KiB/s — independent of peer
count `N`. PSync scales to thousands of peers per topic.

### State-Vector Sync (StateVectorSync, SVS)

Modern alternative: each peer broadcasts a *vector* `(node_id, latest_seq)*` with the
sync digest. Decoder is trivial (entry-wise max), at the cost of `O(N)` bytes per
sync packet. Better for high-churn participant sets where IBLT often falls back.

---

## Performance Model

### Latency vs IP/HTTP/HTTPS

Define request as fetching a single object. Measured RTT from `consumer → producer`
on a steady connection (no DNS):

| Stack          | Setup cost                  | Per-object cost      | Notes                              |
|:---------------|:----------------------------|:---------------------|:-----------------------------------|
| HTTP/1.1+TLS   | TCP 1 RTT + TLS 2 RTT       | 1 RTT                | 3 RTT first request                |
| HTTP/2         | TCP 1 RTT + TLS 2 RTT       | < 1 RTT (mux)        | 3 RTT first; near-0 after          |
| HTTP/3 (QUIC)  | 1 RTT (1-RTT) or 0 RTT (resumed) | < 1 RTT          | 1 RTT first; 0 RTT possible        |
| NDN            | **0 setup**                 | 1 RTT                | every Interest is independent      |
| NDN (cached)   | 0 setup                     | LAN RTT              | hit at first router                |

NDN's per-object 1 RTT is identical to HTTP/2 mux-on-warm-connection, **without** the TCP
or TLS setup amortization. With caching it falls below HTTP because intermediate routers
serve.

### Bandwidth — multicast for free

For `N` consumers fetching the same content from a single producer over a tree:

```
IP unicast:           bandwidth_origin = N · S_data
IP multicast:         bandwidth_origin = 1 · S_data (but rarely deployed E2E)
HTTP CDN:             bandwidth_origin = (N / fanout_per_pop) · S_data
NDN (PIT aggregation): bandwidth_origin = 1 · S_data   (universal, free, no config)
```

For `N=10000` consumers of a 4 MB asset:

```
IP unicast: 40 GB egress at origin
NDN PIT-aggregated: 4 MB
```

This is the killer-app argument for video/CDN workloads.

### In-network aggregation

In an idealized binary tree of depth `L` with `N = 2^L` consumers:

```
Upstream load at each level:
   level L (consumers)     : N Interests
   level L-1               : N/2
   level L-2               : N/4
   ...
   level 0  (origin)       : 1

Total Interest packets in-network: N + N/2 + ... + 1 = 2N − 1   (still O(N))

But upstream Data packets at each level:
   level L                 : N (delivered to consumers)
   level L-1               : N/2
   ...
   level 0  (origin sends) : 1

Origin egress:  S_data — independent of N.
Per-link maximum traffic at level l:  S_data (one Data per (sub-tree, name))
```

So origin sees **`O(1)`** load, the network's *internal* link maximum is also `O(1)` per
unique name — the per-link bandwidth doesn't grow with consumer count. Compare HTTP CDN
where origin still sees `O(N / popfanout)`.

### NDN-DPDK throughput

NDN-DPDK (NDN over Intel DPDK userspace network stack):

```
Single-core forwarding: 30 Mpps Interests   (~ 3 ns / packet on 100 GbE, name caching helps)
Multi-core (6 cores):   100 Gbps line-rate forwarding for 500-byte Data
PIT scaling:            ~10 M entries with 2 GiB allocation
```

Numbers from Shi et al., NDN-DPDK technical report, 2020-2024 measurements on Intel
Xeon 6248R @ 3 GHz with 100 GbE Mellanox ConnectX-5.

---

## Mobility

### Consumer mobility — "free"

A consumer that changes network attachment **just keeps issuing Interests**. The new
network's first router runs FIB lookup on the name; producer doesn't care where the
consumer is. No address rebinding, no NAT, no MIPv6.

```
Before move:
   consumer@AS1: Interest /netflix/show/X    →  AS1 router  →  producer
After move (different AS):
   consumer@AS2: Interest /netflix/show/X    →  AS2 router  →  producer
```

The *producer* sees no mobility event; the path differs but the name is the same.

### Producer mobility — hard

A *mobile producer* changes which router announces its name prefix. NLSR updates
take seconds; in-flight Interests miss.

Mitigations:

| Pattern                | Idea                                                                 | Cost           |
|:-----------------------|:---------------------------------------------------------------------|:---------------|
| Rendezvous-Point (RP)  | A static well-known prefix `/rp/...` aggregates updates              | One indirection |
| Tunneling (Kite)       | Producer sends signed "hint" Interests upstream, leaving breadcrumbs | Path memory    |
| MAP-Me                 | LSA flood on movement; routers update FIB locally                    | Faster than NLSR full |
| Anchorless mapping     | Hash-based DHT mapping name → current router                         | Lookup penalty |

Production deployments (e.g. NDN over LoRa for vehicular) typically use Kite or MAP-Me
because RP creates a single point of failure.

---

## CCNx vs NDN Differences

CCNx-1.0 (RFC 8569 / 8609) and NDN-0.3 (named-data.net spec) share genealogy but
diverged on wire format and semantics around 2014:

| Dimension          | NDN                                                    | CCNx-1.0                                  |
|:-------------------|:-------------------------------------------------------|:------------------------------------------|
| Wire format        | TLV with explicit `Name=0x07`                          | TLV with `T_NAME=0x0000`, fixed header    |
| Hop limit          | optional `HopLimit` TLV                                | mandatory `T_FIXED_HEADER` HopLimit field |
| Selectors          | richer (CanBePrefix, MustBeFresh)                      | none — exact name only                    |
| Name component types | 0x01..0xFF semantic registry                         | flat `T_NAME_SEGMENT=0x0001`              |
| Manifests          | informal (FLIC)                                         | first-class `T_MANIFEST` packet           |
| Forwarder          | NFD, ndn-cxx                                           | Metis (research forwarder)                |
| Sync               | ChronoSync, PSync, SVS                                 | not part of base spec                     |
| Routing            | NLSR                                                   | CCN-RouteProtocol (less developed)        |
| Trust              | trust schemas, name-based                              | also name-based but smaller deployment    |

For a router that must speak both, NFD has an experimental CCNx adaptation layer; in
practice deployments choose one and stick.

---

## Implementation Reference

### NFD (NDN Forwarding Daemon)

```
nfdc status report                   # full forwarder status
nfdc face list                       # list faces
nfdc face create remote udp4://1.2.3.4:6363
nfdc face destroy 256

nfdc route list                      # static FIB entries
nfdc route add /example udp4://1.2.3.4:6363 cost 10
nfdc route remove /example udp4://1.2.3.4:6363

nfdc cs config --capacity 65536 --enable-admit on --enable-serve on
nfdc cs erase /
nfdc strategy set /video /localhost/nfd/strategy/asf

nfdc rib list                        # routing protocol-managed entries
```

### ndn-cxx (C++ library)

```cpp
#include <ndn-cxx/face.hpp>
#include <ndn-cxx/security/key-chain.hpp>

ndn::Face face;
ndn::KeyChain keyChain;

// Producer
face.setInterestFilter("/example",
  [&](const auto& filter, const auto& interest) {
      auto data = std::make_shared<ndn::Data>(interest.getName());
      data->setFreshnessPeriod(10_s);
      data->setContent(reinterpret_cast<const uint8_t*>("hi"), 2);
      keyChain.sign(*data);              // selects best identity by trust schema
      face.put(*data);
  });

face.processEvents();

// Consumer
ndn::Interest interest("/example/foo");
interest.setMustBeFresh(true);
face.expressInterest(interest,
  [](const auto& i, const auto& d){ /* on data */ },
  [](const auto& i, const auto& nack){ /* on nack */ },
  [](const auto& i){ /* timeout */ });
face.processEvents();
```

### NDN-DPDK

```bash
ndndpdk-ctrl create-eth-port    --pci 0000:65:00.0
ndndpdk-ctrl create-eth-face    --port pci0 --remote 02:00:00:00:00:01
ndndpdk-ctrl insert-fib         --name /example --face 1
ndndpdk-ctrl set-cs-capacity    --capacity 1048576

ndndpdk-godemo pingclient --name /example/ping
```

### Throughput math

```
NDN-DPDK forwarding rate per core ≈ (CPU_GHz × IPC_eff) / cycles_per_packet
For Xeon 6248R: 3.0 GHz × 1.5 IPC / 200 cycles ≈ 22 Mpps/core
With 6 cores and pipeline parallelism, ≈ 100 Gbps for 500-B payloads.
```

---

## Worked Examples

### Example 1: Four consumers, PIT aggregation

Topology:

```
   C1 ----+
   C2 ----+--- R_edge --- R_core --- Producer
   C3 ----+
   C4 ----+
```

Timeline (all times in ms, RTT 50 ms edge↔core, 50 ms core↔producer):

```
t=0    C1 sends Interest /wikipedia/article/quic, nonce 0xAAAA
t=1    R_edge: PIT miss, create entry, in_face={C1}, fwd to R_core
t=2    C2 sends same name, nonce 0xBBBB
t=2    R_edge: PIT hit, append in_face C2, no upstream fwd
t=3    C3 sends, nonce 0xCCCC; PIT hit, in_face={C1,C2,C3}
t=3    C4 sends, nonce 0xDDDD; PIT hit, in_face={C1,C2,C3,C4}
t=51   R_core: PIT miss, create entry from R_edge, fwd to Producer
t=101  Producer signs and returns Data
t=151  R_core receives Data, satisfies its PIT entry, fwd to R_edge,
        store in CS if FreshnessPeriod>0, drop PIT entry
t=201  R_edge receives Data, fans out to all 4 in_faces, drops PIT,
        stores in CS
t=201..205  C1..C4 each receive Data

Counts:
  Interests at producer: 1
  Data at producer:      1
  Interests at R_edge inbound: 4
  Interests at R_edge outbound: 1
  Data at R_edge outbound: 4
  Aggregation factor: 4:1
```

Bandwidth saved at producer: `(4 − 1) · S_data = 3 · S_data`. For the same content
fetched a second time within the FreshnessPeriod, R_edge's CS hit serves all four
locally — producer sees zero load.

### Example 2: Cache-hit chain across three routers

Three-level tree, each with 10 % CS hit ratio independently (Zipf, M=100k, C=1k).

```
C → R1 → R2 → R3 → Producer

P(satisfy at R1) = 0.10
P(miss R1, satisfy R2) = 0.90 · 0.10 = 0.09
P(miss R1, miss R2, satisfy R3) = 0.90² · 0.10 = 0.081
P(reach producer) = 0.90³ = 0.729

Origin offload = 1 − 0.729 = 0.271 ≈ 27 %
Average upstream hops = 1·0.10 + 2·0.09 + 3·0.081 + 3·0.729 = 0.10 + 0.18 + 0.243 + 2.187 = 2.71
```

Doubling each level's cache to h=0.18:

```
P(reach producer) = 0.82³ ≈ 0.551
Origin offload = 1 − 0.551 = 0.449
Average hops = 1·0.18 + 2·0.1476 + 3·0.121 + 3·0.551 = 0.18 + 0.295 + 0.363 + 1.653 = 2.49
```

Linear cache scaling, sub-linear hop savings — matches Zipf log-law.

### Example 3: Signature verify chain

Goal: verify Data `/wikipedia/article/quic/v=20251002083500`.

```
1. Decode Data; extract:
     SignatureType  = 3 (EcdsaSha256)
     KeyLocator     = Name(/wikipedia/_KEY/k1)
     SignatureValue = 64 bytes of (r,s)

2. Compute sig_buf = Name || MetaInfo || Content || SignatureInfo
   Compute h = SHA-256(sig_buf)

3. Look up local trust store for Cert(/wikipedia/_KEY/k1).
   Not present? Express Interest /wikipedia/_KEY/k1, MustBeFresh=false.
   Receive Data containing X.509-style cert with EC public key Q1.

4. Verify ECDSA(Q1, h, SignatureValue) → ok.

5. Now verify Cert(/wikipedia/_KEY/k1) recursively:
     Its KeyLocator = Name(/_ROOT/_KEY/r0)
     Trust schema rule: site cert under /wikipedia/* signed by /_ROOT/_KEY/*  → match.
     Fetch /_ROOT/_KEY/r0 (or load from anchors list), verify, recurse.

6. Hit anchor /_ROOT/_KEY/r0 in trust-anchor store → terminate, success.
```

Total work for a 3-level chain: 3 Interest fetches (cached after first), 3 ECDSA
verifies (≈ 250 µs each on x86), one trust-schema regex pass (sub-µs).

### Example 4: NLSR adjacency formation timing

Two routers `A` and `B` connected by Ethernet face, NLSR Hello interval 60 s, Sync
interval 1 s.

```
t=0.000    A boots, NLSR starts
t=0.010    A sends Hello Interest /<network>/A/HELLO out FaceId=257 (Eth)
t=0.012    B receives Hello; matches name pattern; replies with Hello Data
            content = <B's router-name, link-cost=10>
t=0.014    A receives Hello Data; learns neighbor=B, cost=10
t=0.020    A publishes new AdjacencyLSA via Sync:
              /<network>/lsdb/A/ADJ/v=20251002083500/seg=0
t=1.020    Sync digest update propagates (1s sync interval)
t=1.025    B fetches A's new LSA via /<network>/lsdb/A/ADJ/...
t=1.040    B runs Dijkstra including A; installs route entries
t=1.050    B publishes its own AdjacencyLSA
t=2.050    A fetches B's LSA
t=2.070    A runs Dijkstra; installs FIB
t≈2.1      bidirectional reachability via NLSR-managed FIB

Total convergence: ~2.1 s for two-node
General: t_conv ≈ d_max · (T_sync_interval + processing) ≈ d_max · 1.05 s
```

For a 6-hop diameter network, expect ~6.3 s to full reconvergence after a single
adjacency change — comparable to OSPF SPF runs.

---

## When to Use NDN, When Not

### Good fits

| Workload                              | Why                                                              |
|:--------------------------------------|:-----------------------------------------------------------------|
| Video streaming (VoD + live)          | Caching, multicast aggregation, signature-once-serve-many        |
| Software distribution / package mirrors | Same as video — read-mostly, popularity-skewed                 |
| IoT mesh / vehicular ad-hoc           | Self-learning strategy, mobility, named data over LoRa/802.11p   |
| Large-file scientific data (NDN-CMS)  | Massive parallel fetch with PIT aggregation across HPC fabric    |
| Edge computing / function delivery    | Names address functions; signature gives integrity              |
| Disaster-tolerant overlays            | No address; works on partitions; cached state survives outage   |

### Poor fits

| Workload                                  | Why                                                            |
|:------------------------------------------|:---------------------------------------------------------------|
| Sub-millisecond control loops             | Per-Data signing dominates latency (200-500 µs ECDSA)         |
| Classical OLTP databases                  | Mutable per-row state; trust schema overhead per write         |
| Heavy 1-1 interactive (SSH, RDP)          | Caching doesn't help; per-packet signing overhead              |
| Anonymity-critical (Tor-like)             | Names leak intent; hop-by-hop visible plaintext naming         |
| High-write, low-read workloads            | Cache cannot amortize; producer-side load dominates            |

---

## See Also

- `networking/dns` — name resolution, hierarchy, caching analogues
- `networking/multicast-routing` — IP-layer multicast comparison; PIM-SSM, PIM-SM
- `networking/igmp` — host-side multicast joins; semantically closest to NDN Interest
- `cs-theory/distributed-systems` — sync, eventual consistency, CRDT-like merges
- `cs-theory/distributed-consensus` — NLSR converges without consensus; contrast
- `ramp-up/named-data-networking-eli5` — narrative companion

## References

- RFC 8569 — *Content-Centric Networking (CCNx) Semantics*, Mosko/Solis/Wood, 2019.
- RFC 8609 — *Content-Centric Networking (CCNx) Messages in TLV Format*, 2019.
- RFC 7945 — *Information-Centric Networking (ICN) Research Challenges*, Pentikousis et al., 2016.
- Jacobson, V. et al. — *Networking Named Content*, ACM CoNEXT 2009.
- Zhang, L. et al. — *Named Data Networking*, ACM SIGCOMM CCR, July 2014.
- NDN Project — *NDN Packet Format Specification 0.3*, named-data.net/doc/NDN-packet-spec/
- NDN TR-0001 — *Named Data Networking (NDN) Project*, NDN Project Team, October 2010.
- NDN TR-0021 — *NLSR: Named-data Link State Routing Protocol*, A.K.M. Hoque et al., 2013.
- NDN TR-0022 — *Trust Schema for NDN Applications*, Yu et al., 2015.
- NDN TR-0024 — *NDN-DPDK: NDN Forwarding at 100 Gbps*, Shi et al., 2020.
- Eppstein, D., Goodrich, M. — *Straggler Identification in Round-Trip Data Streams via
  Newton's Identities and Invertible Bloom Filters*, IEEE TKDE, 2011.
- Che, H. et al. — *Hierarchical Web Caching Systems: Modeling, Design and Experimental
  Results*, IEEE JSAC, 2002.
- Psaras, I. et al. — *Probabilistic In-Network Caching for Information-Centric Networks*,
  ACM SIGCOMM ICN Workshop, 2012.
- Compagno, A. et al. — *Poseidon: Mitigating Interest Flooding DDoS Attacks in NDN*,
  IEEE LCN 2013.
- Yi, C. et al. — *Adaptive Forwarding in Named Data Networking*, ACM SIGCOMM CCR, 2012.
- named-data.net — *NFD Developer's Guide*, current.
- named-data.net — *ndn-cxx: NDN C++ library with eXperimental eXtensions*, current.
