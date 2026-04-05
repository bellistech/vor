# Site-to-Site VPN — IKE Protocol Internals, IPsec Architecture, and Cryptographic Analysis

> *Site-to-site VPNs establish encrypted tunnels between network devices using the Internet Key Exchange (IKE) protocol to negotiate Security Associations and IPsec to encrypt traffic. Understanding the IKEv1 vs IKEv2 exchange mechanics, Diffie-Hellman key derivation, ESP packet format, SA lifecycle management, and architectural tradeoffs between crypto maps and VTI is essential for designing, deploying, and troubleshooting enterprise VPN infrastructure.*

---

## 1. IKEv1 vs IKEv2 Exchange Comparison

### IKEv1 Exchanges

IKEv1 (RFC 2409) uses a two-phase approach with multiple exchange types:

```
Phase 1 — Main Mode (6 messages):

  Initiator                                     Responder
      |                                              |
      |--- SA proposal (encryption, hash, DH, auth)->|  Msg 1
      |<-- SA acceptance (selected proposal) --------|  Msg 2
      |                                              |
      |--- KE (DH public value) + Nonce ------------>|  Msg 3
      |<-- KE (DH public value) + Nonce -------------|  Msg 4
      |                                              |
      | [SKEYID derived — all subsequent msgs encrypted]
      |                                              |
      |--- ID + AUTH (hash of SKEYID_a) ------------>|  Msg 5 (encrypted)
      |<-- ID + AUTH (hash of SKEYID_a) -------------|  Msg 6 (encrypted)
      |                                              |
      | [IKE SA established — ISAKMP SA]

  Messages 1-4: cleartext (identity protected — not sent until encrypted)
  Messages 5-6: encrypted with derived SKEYID_e

Phase 1 — Aggressive Mode (3 messages):

  Initiator                                     Responder
      |                                              |
      |--- SA + KE + Nonce + ID ------------------->|  Msg 1 (cleartext)
      |<-- SA + KE + Nonce + ID + AUTH -------------|  Msg 2 (cleartext)
      |--- AUTH ---------------------------------------->|  Msg 3 (encrypted)
      |                                              |
      | [IKE SA established]

  All identity information sent in cleartext (Msgs 1-2)!
  Security implication: peer identity exposed to eavesdropper
  Use case: dynamic IP peers (identity cannot be determined from IP)

Phase 2 — Quick Mode (3 messages, all encrypted):

  Initiator                                     Responder
      |                                              |
      |--- HASH(1) + SA + Nonce + [KE] + ID_ci/cr ->|  Msg 1
      |<-- HASH(2) + SA + Nonce + [KE] + ID_ci/cr --|  Msg 2
      |--- HASH(3) ------------------------------------->|  Msg 3
      |                                              |
      | [IPsec SA pair established — one in each direction]

  [KE] = optional DH exchange for PFS
  ID_ci/cr = proxy identities (source/destination subnets)
```

### IKEv2 Exchanges

IKEv2 (RFC 7296) simplifies the protocol to four messages in two exchanges:

```
IKE_SA_INIT (2 messages, cleartext):

  Initiator                                     Responder
      |                                              |
      |--- SAi1 + KEi + Ni --------------------------->|  Msg 1
      |<-- SAr1 + KEr + Nr + [CERTREQ] --------------|  Msg 2
      |                                              |
      | [SKEYSEED derived — remaining messages encrypted]

  SAi1/SAr1: IKE SA crypto proposals
  KEi/KEr: DH public values (key exchange)
  Ni/Nr: nonces (freshness guarantee)

IKE_AUTH (2 messages, encrypted):

  Initiator                                     Responder
      |                                              |
      |--- IDi + [CERT] + AUTH + SAi2 + TSi + TSr -->|  Msg 3
      |<-- IDr + [CERT] + AUTH + SAr2 + TSr + TSi --|  Msg 4
      |                                              |
      | [IKE SA + first Child SA (IPsec SA) established]

  IDi/IDr: peer identities
  AUTH: authentication payload (PSK or RSA/ECDSA signature)
  SAi2/SAr2: Child SA (IPsec) crypto proposals
  TSi/TSr: traffic selectors (replaces proxy identities)

CREATE_CHILD_SA (for rekeying or additional Child SAs):

  Initiator                                     Responder
      |                                              |
      |--- SA + Ni + [KEi] + TSi + TSr ------------->|  Msg 1
      |<-- SA + Nr + [KEr] + TSi + TSr --------------|  Msg 2
      |                                              |
      | [New Child SA established or IKE SA rekeyed]
```

### Protocol Comparison Summary

```
Feature              IKEv1                      IKEv2
─────────────────────────────────────────────────────────────────
Messages to SA       9 (main+quick)             4 (init+auth)
                     6 (aggressive+quick)
Aggressive mode      Yes (identity exposure)    No (not needed)
NAT-T                Extension (RFC 3947)       Built-in
EAP support          No                         Yes (IKE_AUTH)
Multiple Child SAs   Separate Quick Mode each   CREATE_CHILD_SA
Rekeying             Delete + new negotiation   In-place rekey
Cookie/anti-DoS      No                         Cookie challenge
Traffic selectors    Proxy IDs (ACL-based)      TSi/TSr payloads
Reliability          Vendor-specific retransmit Standard retransmit
Configuration        Mode-config (extension)    Built-in CP payload
MOBIKE               No                         Yes (RFC 4555)
Message format       Phase-specific             Uniform request/response
```

---

## 2. Diffie-Hellman Key Exchange Mathematics

### Classical DH (MODP Groups)

The Diffie-Hellman protocol allows two parties to derive a shared secret over an insecure channel without transmitting the secret itself:

```
Parameters (public, defined per group):
  p = large prime modulus
  g = generator of the multiplicative group Z*_p

  Group 14: p = 2048-bit prime, g = 2
  Group 15: p = 3072-bit prime, g = 2
  Group 16: p = 4096-bit prime, g = 2

Protocol:
  1. Initiator generates random private value: a ∈ [2, p-2]
     Computes public value: A = g^a mod p
     Sends A to responder

  2. Responder generates random private value: b ∈ [2, p-2]
     Computes public value: B = g^b mod p
     Sends B to initiator

  3. Shared secret computation:
     Initiator: S = B^a mod p = (g^b)^a mod p = g^(ab) mod p
     Responder: S = A^b mod p = (g^a)^b mod p = g^(ab) mod p
     Both derive the same S without transmitting a or b

  Security basis: Computational Diffie-Hellman (CDH) assumption
    Given (g, p, g^a mod p, g^b mod p), computing g^(ab) mod p
    is believed to be computationally infeasible for large p

  Attack complexity (best known):
    Number Field Sieve (NFS): sub-exponential in |p|
    Group 14 (2048-bit): ~2^112 operations (adequate through ~2030)
    Group 16 (4096-bit): ~2^152 operations
```

### Elliptic Curve DH (ECDH Groups)

```
ECC operates on points of an elliptic curve over a finite field:
  Curve equation: y^2 = x^3 + ax + b (mod p)

Parameters (public, defined per curve):
  (a, b, p) = curve parameters
  G = generator point (base point on the curve)
  n = order of G (number of points in subgroup)

  Group 19 (ECP-256 / NIST P-256):
    p = 2^256 - 2^224 + 2^192 + 2^96 - 1
    Key size: 256 bits (~128-bit security)

  Group 20 (ECP-384 / NIST P-384):
    p = 2^384 - 2^128 - 2^96 + 2^32 - 1
    Key size: 384 bits (~192-bit security)

  Group 21 (ECP-521 / NIST P-521):
    Key size: 521 bits (~256-bit security)

Protocol:
  1. Initiator: random scalar a, compute A = a * G (point multiplication)
  2. Responder: random scalar b, compute B = b * G
  3. Shared point: S = a * B = b * A = (ab) * G
     Shared secret = x-coordinate of S

  ECC advantage:
    256-bit ECC ≈ 3072-bit MODP security
    Much smaller key exchange payloads
    Faster computation (especially on constrained devices)
```

### IKE Key Derivation

```
IKEv2 key derivation from DH shared secret:

  SKEYSEED = prf(Ni | Nr, g^ir)
    where g^ir = DH shared secret
    Ni, Nr = initiator and responder nonces

  Key material = prf+(SKEYSEED, Ni | Nr | SPIi | SPIr)
    Expanded into:
      SK_d  — used to derive Child SA keys
      SK_ai — IKE SA integrity key (initiator)
      SK_ar — IKE SA integrity key (responder)
      SK_ei — IKE SA encryption key (initiator)
      SK_er — IKE SA encryption key (responder)
      SK_pi — used in AUTH payload generation (initiator)
      SK_pr — used in AUTH payload generation (responder)

  Child SA key derivation:
    KEYMAT = prf+(SK_d, Ni | Nr [| g^ir_new if PFS])
    Split into: encryption key + integrity key for each direction

  prf+ is the PRF-based key expansion function:
    T1 = prf(K, S | 0x01)
    T2 = prf(K, T1 | S | 0x02)
    T3 = prf(K, T2 | S | 0x03)
    ...
    Output = T1 | T2 | T3 | ... (truncated to required length)
```

---

## 3. ESP Packet Format

### ESP Header and Trailer (RFC 4303)

```
ESP Tunnel Mode packet:

  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  | Outer IP Header (20B IPv4 / 40B IPv6)                         |
  | Protocol = 50 (ESP)                                           |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |                ESP Header                                      |
  |   Security Parameters Index (SPI) — 32 bits                   |
  |   Sequence Number — 32 bits                                    |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |                Initialization Vector (IV)                      |  ← depends on
  |                (AES-CBC: 16B, AES-GCM: 8B)                    |    cipher
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |                                                                |
  |                Encrypted Payload                               |  ← encrypted
  |   [Original IP Header (tunnel mode)]                          |    region
  |   [Original IP Payload]                                        |
  |   [ESP Padding (0-255 bytes)]                                 |
  |   [Pad Length (1 byte)]                                        |
  |   [Next Header (1 byte)]                                       |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |                Integrity Check Value (ICV)                     |  ← integrity
  |   (HMAC-SHA-256: 16B, AES-GCM: 16B)                          |    check
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

  Encryption coverage: IV + Payload + Padding + Pad Length + Next Header
  Integrity coverage:  ESP Header + IV + Encrypted Payload (not outer IP)

ESP Transport Mode:
  [Original IP Header][ESP Header][IV][Original Payload (encrypted)]
                                      [Padding][Pad Length][NH][ICV]

  Note: transport mode preserves the original IP header (not encrypted)
  Outer IP header = original IP header (modified: protocol = 50)
```

### ESP Processing Pipeline

```
ESP Encryption (outbound):

  1. Determine SA by routing/policy (SPD lookup)
  2. Construct ESP header: SPI (from SA), Sequence Number (increment)
  3. Pad payload to block boundary (AES = 16-byte blocks)
     Padding bytes: 1, 2, 3, 4, ... (sequential values)
     Append: Pad Length (1B) + Next Header (1B)
  4. Generate IV (random for CBC, counter for GCM)
  5. Encrypt: IV + padded payload
     AES-CBC: Ci = E(K, Pi XOR C(i-1)), C0 = IV
     AES-GCM: single-pass AEAD (encrypt + authenticate)
  6. Compute ICV over ESP header + IV + ciphertext
     HMAC-SHA-256: truncated to 128 bits (16 bytes)
     AES-GCM: GHASH tag (16 bytes)
  7. Prepend outer IP header (tunnel mode)
  8. Send packet

ESP Decryption (inbound):

  1. Match SPI to SA (SPI + destination IP = SA lookup key)
  2. Check sequence number against anti-replay window
  3. Verify ICV (integrity check)
     If ICV mismatch → drop (tampered or corrupted)
  4. Decrypt payload using key and IV from SA
  5. Remove padding (read Pad Length byte)
  6. Extract inner packet (tunnel mode) or original payload (transport)
  7. Forward to IP stack for routing/delivery
```

### AES-GCM (Combined Mode — Preferred)

```
AES-GCM provides authenticated encryption in a single pass:

  Advantages over separate encrypt+HMAC:
    - Single pass: ~2x throughput vs AES-CBC + HMAC-SHA
    - Built-in authentication (no separate integrity algorithm)
    - Parallelizable encryption (CTR mode base)
    - Hardware acceleration (AES-NI + PCLMULQDQ)

  ESP with AES-GCM:
    - IV: 8 bytes (64-bit counter, prepended to 4-byte salt from SA)
    - No separate integrity algorithm needed
    - ICV: 16 bytes (128-bit GHASH tag)
    - Next Header in ESP trailer is authenticated but not encrypted
      (GCM AAD covers ESP header)

  IOS-XE configuration:
    crypto ipsec transform-set TSET_GCM esp-gcm 256
      mode tunnel

  Cipher suite selection hierarchy (strongest to weakest):
    1. AES-256-GCM          — AEAD, 256-bit, highest throughput
    2. AES-128-GCM          — AEAD, 128-bit, good performance
    3. AES-256-CBC + SHA-256 — separate encrypt+auth
    4. AES-128-CBC + SHA-256 — minimum recommended
    5. AES-128-CBC + SHA-1   — legacy (SHA-1 deprecated for new deploys)
    6. 3DES-CBC + SHA-1      — deprecated, do not use
```

---

## 4. IPsec SA Lifecycle

### SA States and Transitions

```
IPsec SA lifecycle:

  IDLE → NEGOTIATING → ESTABLISHED → REKEYING → DELETED
           │                            │
           └── FAILED ──────────────────┘

  Creation:
    1. Traffic matches SPD (Security Policy Database) entry
    2. IKE initiates Child SA negotiation
    3. Crypto parameters agreed, keys derived
    4. SA installed in SAD (Security Association Database)
    5. State: ESTABLISHED

  Maintenance:
    - SA has two lifetimes (whichever expires first triggers rekey):
      a. Time-based: default 3600s (1 hour) — configurable
      b. Volume-based: default 4,608,000 KB — configurable
    - Soft lifetime = hard lifetime - jitter (random 0-10%)
      At soft lifetime: initiate rekey
      At hard lifetime: delete SA if rekey failed

  Rekeying (IKEv2 CREATE_CHILD_SA):
    1. Soft lifetime reached → initiator sends CREATE_CHILD_SA
    2. New SA negotiated (new keys, new SPI)
    3. Old SA remains active until new SA is installed
    4. Traffic switches to new SA
    5. Old SA deleted after brief overlap (make-before-break)

  IKEv1 rekeying:
    - No in-place rekey — initiator deletes old SA, negotiates new
    - Brief traffic interruption during renegotiation
    - Lifetime jitter prevents both peers from initiating simultaneously

  Deletion:
    - Hard lifetime reached without successful rekey
    - DPD detects peer failure (3 missed probes)
    - Administrative clear (clear crypto sa)
    - Peer sends IKE DELETE notification
```

### SA Database Relationships

```
Security Policy Database (SPD):
  Defines what traffic should be protected and how
  Entries: (src_ip, dst_ip, protocol, ports) → action (protect/bypass/discard)
  In Cisco: crypto map ACL or VTI routing table

Security Association Database (SAD):
  Contains active SAs with keying material
  Indexed by: (SPI, destination IP, protocol [ESP/AH])
  Each entry:
    - SPI (32-bit, unique per direction)
    - Sequence number counter (64-bit with ESN)
    - Anti-replay window bitmap
    - Encryption key + algorithm
    - Integrity key + algorithm
    - SA lifetime counters (time + bytes)
    - Tunnel endpoints (outer src/dst IP)

  SAs are unidirectional — each tunnel has a pair:
    Outbound SA: local encrypts, remote decrypts
    Inbound SA:  remote encrypts, local decrypts
    Different SPIs for each direction
```

---

## 5. Crypto Map vs VTI Architecture

### Crypto Map Architecture

```
Crypto map model (legacy):

  Packet flow:
    1. Outbound packet arrives at egress interface
    2. Crypto map ACL evaluated against packet headers
    3. If match → encrypt with specified SA parameters
    4. If no match → send in cleartext
    5. Encrypted packet sent out physical interface

  Characteristics:
    - Policy-based VPN (not route-based)
    - "Interesting traffic" ACL defines what to encrypt
    - No tunnel interface — crypto applied on physical interface
    - Cannot run routing protocols through the VPN natively
    - Each ACL entry = separate IPsec SA pair
    - Multiple crypto map entries per interface (sequenced)

  Limitations:
    - Complex ACL management (must be mirrored on both peers)
    - No per-tunnel QoS, NetFlow, or interface-level features
    - Multicast/dynamic routing requires GRE (adds complexity)
    - Failover requires multiple crypto map entries + tracking
    - Split tunneling requires explicit ACL entries
    - Asymmetric routing can bypass crypto map evaluation
```

### VTI Architecture

```
VTI model (modern):

  Packet flow:
    1. Routing table directs packet to Tunnel interface
    2. Tunnel interface applies IPsec encapsulation
    3. Encapsulated packet routed via physical interface
    4. No ACL evaluation — all traffic through tunnel is encrypted

  Characteristics:
    - Route-based VPN (routable tunnel interface)
    - Routing protocols (OSPF, BGP, EIGRP) run natively
    - Per-tunnel interface features: QoS, ACL, NetFlow, PBR
    - Multicast supported without GRE (IPsec tunnel mode)
    - Clean failover with routing protocol convergence
    - Simpler configuration (no mirror ACLs)

  Static VTI (sVTI):
    - Point-to-point, fixed peer
    - One Tunnel interface per peer
    - Suitable for: fixed site-to-site VPNs with few peers

  Dynamic VTI (dVTI):
    - Hub creates Virtual-Access interfaces on demand
    - Cloned from Virtual-Template
    - Suitable for: hub-and-spoke with many dynamic spokes
    - Scales better than sVTI for large spoke counts

  VTI vs crypto map decision:
    ┌─────────────────────────────┬──────────┬────────┐
    │ Requirement                 │ Crypto Map│ VTI   │
    ├─────────────────────────────┼──────────┼────────┤
    │ Routing protocols over VPN  │ No*      │ Yes    │
    │ Per-tunnel QoS              │ No       │ Yes    │
    │ Per-tunnel ACL              │ No       │ Yes    │
    │ Multicast through VPN      │ No*      │ Yes    │
    │ Simple failover             │ No       │ Yes    │
    │ Dynamic spoke creation      │ No       │ Yes    │
    │ Legacy IOS support          │ Yes      │ Varies │
    │ Multiple SA per peer        │ Yes      │ No**   │
    └─────────────────────────────┴──────────┴────────┘
    * Possible with GRE overlay
    ** VTI uses a single SA pair per tunnel (all traffic encrypted)
```

---

## 6. NAT Traversal (NAT-T) Encapsulation

### NAT-T Detection and Encapsulation

```
NAT-T detection (IKE_SA_INIT):
  Both peers include NAT_DETECTION_SOURCE_IP and
  NAT_DETECTION_DESTINATION_IP notification payloads:

    NAT_DETECTION_*_IP = SHA-1(SPIi | SPIr | IP | port)

  Each peer computes the expected hash for its own and peer's IP:port
  If received hash ≠ computed hash → NAT is present on that side

  After NAT detection:
    IKE: switch from UDP 500 to UDP 4500 (all subsequent IKE traffic)
    ESP: encapsulate in UDP 4500 (Non-ESP Marker = 0x00000000)

NAT-T encapsulated ESP packet:

  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  | Outer IP Header                                    |
  | Src: NAT-translated IP   Dst: peer public IP      |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  | UDP Header (src: random high, dst: 4500)           |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  | Non-ESP Marker: 0x00000000 (only for IKE over 4500)|
  | or ESP Header (SPI ≠ 0 distinguishes from IKE)     |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  | ESP Payload (encrypted original packet)            |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

  Note: NAT devices track UDP flows by src:port ↔ dst:port
  Keepalive packets (1 byte 0xFF payload) sent every 20-30s
  to prevent NAT mapping timeout
```

### Why ESP Cannot Traverse NAT Without NAT-T

```
ESP (IP protocol 50) has no port numbers:
  - NAT rewrites IP addresses + TCP/UDP ports for multiplexing
  - ESP has no ports — NAT cannot distinguish multiple ESP flows
    from the same internal IP (or to the same external IP)
  - NAT checksum recalculation breaks ESP integrity check
  - AH explicitly protects the IP header — any NAT modification
    invalidates the AH ICV

NAT-T solution:
  - Wraps ESP in UDP (which has ports for NAT multiplexing)
  - UDP checksum set to 0 (or recalculated by NAT)
  - ESP SPI provides flow discrimination within the UDP wrapper
  - Multiple IPsec tunnels behind same NAT: different SPIs on UDP 4500
```

---

## 7. Anti-Replay Window

### Sliding Window Mechanism

```
Anti-replay prevents an attacker from capturing and retransmitting
valid ESP packets to disrupt communication or replay transactions:

  Sender:
    - 64-bit sequence number counter per SA (starts at 1)
    - Incremented for each packet sent
    - Lower 32 bits in ESP header, upper 32 bits implicit (ESN)
    - Counter wraps: SA must be rekeyed before overflow

  Receiver:
    - Maintains sliding window of W bits (default W=64)
    - Window tracks which sequence numbers have been received
    - Right edge = highest sequence number received

    Window state: [N-W+1 ... N]  (N = highest received)
    For incoming packet with sequence S:
      If S > N: advance window, accept, mark S as received
      If N-W < S ≤ N: check if S already marked → accept or reject
      If S ≤ N-W: reject (too old, outside window)

  Example (W=8, N=20):
    Window covers: [13, 14, 15, 16, 17, 18, 19, 20]
    Bitmap:        [1,  0,  1,  1,  0,  1,  1,  1]
                   received  missed    received

    Packet S=21: accept, advance window to [14..21]
    Packet S=15: already received → reject (replay)
    Packet S=12: outside window → reject (too old)
    Packet S=17: not yet received → accept, mark

  Window sizing:
    W=64  — default, suitable for most links
    W=512 — recommended for high-speed links (10G+) or
             links with reordering (parallel crypto engines)
    W=1024 — maximum, for very high-speed or out-of-order delivery

    Larger window = more memory per SA but fewer false positives
    on reordered packets
```

### Extended Sequence Numbers (ESN)

```
ESN (RFC 4304):
  - Standard ESP: 32-bit sequence number → wraps at 2^32 (~4 billion)
  - At 10 Gbps with 1000-byte packets: wraps in ~57 minutes
  - Before ESN: SA must rekey before wrap (frequent rekeying at high speed)

  ESN extends to 64 bits:
  - Upper 32 bits NOT transmitted in ESP header (saves bandwidth)
  - Receiver tracks upper bits locally
  - Wrap at 2^64 packets — effectively never wraps
  - Anti-replay window uses full 64-bit sequence number

  IOS-XE: ESN enabled by default with IKEv2
```

---

## 8. PFS Security Analysis

### Why PFS Matters

```
Without PFS:
  Child SA keys derived from: SK_d + Ni + Nr (from IKE SA)
  If IKE SA long-term key is compromised (e.g., PSK stolen):
    → Attacker can derive SK_d
    → All past and future Child SA keys can be computed
    → All recorded encrypted traffic can be decrypted

  Timeline of attack:
    1. Attacker records all encrypted VPN traffic (passive)
    2. Years later, attacker obtains PSK or private key
    3. Attacker replays IKE negotiation, derives SKEYSEED
    4. From SKEYSEED → SK_d → all Child SA keys
    5. All historical traffic decrypted

With PFS:
  Each Child SA includes a fresh DH exchange:
    KEYMAT = prf+(SK_d, g^ir_new | Ni | Nr)
  The new DH shared secret (g^ir_new) is ephemeral
  Compromising SK_d is insufficient — attacker also needs
  the ephemeral DH private values (a, b) for each Child SA

  Security guarantee:
    - Each Child SA has independent keying material
    - Compromise of one Child SA does not affect others
    - Compromise of IKE SA keys does not expose Child SA data
    - Forward secrecy: past sessions remain secure even if
      long-term credentials are later compromised

PFS cost:
  - Additional DH computation per rekey (CPU overhead)
  - One extra round-trip in CREATE_CHILD_SA (bandwidth)
  - Group 14 (MODP-2048): ~10ms per DH on modern hardware
  - Group 19 (ECP-256): ~1ms per DH (much faster)
  - Recommendation: always enable PFS with ECP groups
```

---

## 9. Cipher Suite Selection

### Algorithm Recommendations (2024+)

```
Tier 1 — Recommended (deploy for new VPNs):
  IKE SA:    AES-256-GCM + SHA-384 + ECP-384 (group 20)
  Child SA:  AES-256-GCM + PFS group 20
  Auth:      ECDSA P-384 certificates or PSK (256-bit min)

Tier 2 — Acceptable (existing deployments):
  IKE SA:    AES-256-CBC + SHA-256 + MODP-2048 (group 14)
  Child SA:  AES-256-CBC + SHA-256 + PFS group 14
  Auth:      RSA-2048 certificates or PSK

Tier 3 — Legacy (plan migration):
  IKE SA:    AES-128-CBC + SHA-1 + MODP-1536 (group 5)
  Child SA:  AES-128-CBC + SHA-1 + PFS group 5
  Note:      SHA-1 deprecated for digital signatures, acceptable for HMAC
             Group 5 considered weak (1536-bit MODP)

Tier 4 — Deprecated (do not use):
  3DES, DES, MD5, DH groups 1/2, NULL encryption
  These provide insufficient security margin

Security equivalence:
  ┌──────────────┬───────────────┬──────────────┬──────────┐
  │ Symmetric    │ DH/ECDH      │ RSA/DSA      │ Security │
  ├──────────────┼───────────────┼──────────────┼──────────┤
  │ AES-128      │ MODP-3072(15)│ RSA-3072     │ 128-bit  │
  │              │ ECP-256 (19) │ ECDSA P-256  │          │
  ├──────────────┼───────────────┼──────────────┼──────────┤
  │ AES-192      │ MODP-7680    │ RSA-7680     │ 192-bit  │
  │              │ ECP-384 (20) │ ECDSA P-384  │          │
  ├──────────────┼───────────────┼──────────────┼──────────┤
  │ AES-256      │ MODP-15360   │ RSA-15360    │ 256-bit  │
  │              │ ECP-521 (21) │ ECDSA P-521  │          │
  └──────────────┴───────────────┴──────────────┴──────────┘

  Note: match security levels across all algorithms in a suite
  Mismatched levels waste resources (strongest component limited by weakest)
```

---

## See Also

- ipsec
- remote-access-vpn
- cryptography
- pki
- tls
- cisco-ftd

## References

- RFC 7296 — Internet Key Exchange Protocol Version 2 (IKEv2)
- RFC 2409 — The Internet Key Exchange (IKEv1)
- RFC 4303 — IP Encapsulating Security Payload (ESP)
- RFC 4302 — IP Authentication Header (AH)
- RFC 3948 — UDP Encapsulation of IPsec ESP Packets (NAT-T)
- RFC 4304 — Extended Sequence Number (ESN) Addendum
- RFC 3526 — MODP Diffie-Hellman Groups
- RFC 5903 — ECP Groups for IKE and IKEv2
- RFC 6379 — Suite B Cryptographic Suites for IPsec
- NIST SP 800-77 Rev. 1 — Guide to IPsec VPNs
