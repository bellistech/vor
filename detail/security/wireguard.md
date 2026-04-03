# The Mathematics of WireGuard — Modern VPN Cryptography

> *WireGuard is a minimal VPN protocol using Noise_IK handshake framework, Curve25519 key exchange, ChaCha20-Poly1305 authenticated encryption, and BLAKE2s hashing. Its ~4,000 lines of kernel code implement a cryptographically sound tunnel with formally verified security properties.*

---

## 1. Cryptographic Primitives

### Fixed Cipher Suite

WireGuard uses exactly one cipher suite (no negotiation):

| Component | Algorithm | Security Level |
|:---|:---|:---:|
| Key exchange | Curve25519 ECDH | 128-bit |
| Symmetric encryption | ChaCha20-Poly1305 | 256-bit (128-bit auth) |
| Hashing | BLAKE2s | 128-bit |
| Key derivation | HKDF (BLAKE2s) | 128-bit |
| Cookie MAC | BLAKE2s-MAC-128 | 128-bit |

No cipher negotiation = no downgrade attacks.

### Curve25519 Key Generation

Private key: 256 bits of random data with clamping:

$$a[0] \mathbin{\&}= 248, \quad a[31] \mathbin{\&}= 127, \quad a[31] \mathbin{|}= 64$$

Public key: $A = a \cdot G$ where $G$ is the base point of Curve25519.

$$G = (9, \ldots) \quad \text{on } y^2 = x^3 + 486662x^2 + x \pmod{2^{255} - 19}$$

---

## 2. Noise_IK Handshake

### Handshake Protocol

WireGuard uses the Noise_IK pattern (initiator knows responder's static key):

```
Initiator (I)                    Responder (R)
  I.static (S_I)                   R.static (S_R)
  I.ephemeral (E_I)                R.ephemeral (E_R)

  msg1: E_I, AEAD(DH(E_I, S_R), S_I), AEAD(..., timestamp)  -->
                                <-- msg2: E_R, AEAD(DH(E_R, S_I), empty)
```

### DH Operations per Handshake

| Message | DH Computations | Keys Mixed |
|:---|:---:|:---|
| msg1 (initiator) | 2 | $\text{DH}(E_I, S_R)$, encrypt $S_I$ |
| msg1 (responder) | 2 | Verify $E_I$, decrypt $S_I$ |
| msg2 (responder) | 2 | $\text{DH}(E_R, E_I)$, $\text{DH}(E_R, S_I)$ |
| msg2 (initiator) | 2 | Verify $E_R$ |
| **Total** | **4 per side** | 4 DH operations each |

### Key Derivation Chain

The handshake maintains a chaining key $C$ and hash $H$:

$$C_{i+1}, K_i = \text{HKDF}(C_i, \text{DH result}_i)$$

After the handshake, two transport keys are derived:

$$T_{send}, T_{recv} = \text{HKDF}(C_{final}, \text{empty})$$

### Handshake Performance

| Operation | Time | Notes |
|:---|:---:|:---|
| 1 Curve25519 DH | ~0.1 ms | Fixed-base: ~0.05 ms |
| 4 DH operations | ~0.4 ms | Per side |
| ChaCha20-Poly1305 AEAD | <0.01 ms | Small handshake payloads |
| BLAKE2s hash | <0.01 ms | |
| **Total handshake** | **~0.5 ms** | Both sides |

---

## 3. Transport Data Encryption

### Packet Encryption

Each data packet:

$$C = \text{ChaCha20-Poly1305}(T_{send}, \text{counter}, \text{plaintext})$$

| Field | Size | Purpose |
|:---|:---:|:---|
| Type | 1 byte | Message type (4 = transport) |
| Reserved | 3 bytes | Zero |
| Receiver index | 4 bytes | Session identifier |
| Counter | 8 bytes | Nonce (monotonic) |
| Encrypted payload | variable | ChaCha20-Poly1305 |
| Auth tag | 16 bytes | Poly1305 MAC |

### Overhead per Packet

$$\text{Overhead} = 16\text{ (header)} + 16\text{ (auth tag)} + 20\text{ (IP)} + 8\text{ (UDP)} = 60 \text{ bytes}$$

### MTU Calculation

$$\text{WireGuard MTU} = \text{Interface MTU} - 60 \text{ (v4)} \text{ or } - 80 \text{ (v6)}$$

| Interface MTU | WireGuard MTU (IPv4) | Overhead % |
|:---:|:---:|:---:|
| 1500 | 1440 | 4.0% |
| 9000 (jumbo) | 8940 | 0.7% |
| 1280 (IPv6 min) | 1220 | 4.7% |

---

## 4. Anti-Replay — Sliding Window

### Counter-Based Replay Protection

WireGuard uses a sliding bitmap window to track received counters:

$$\text{Window} = [N_{max} - W + 1, \ldots, N_{max}]$$

Where $W$ = window size (default: 2048) and $N_{max}$ = highest counter seen.

$$\text{Accept}(n) = \begin{cases} \text{true} & \text{if } n > N_{max} \text{ (advance window)} \\ \text{true} & \text{if } n \geq N_{max} - W + 1 \text{ AND bit}(n) = 0 \text{ (not yet seen)} \\ \text{false} & \text{otherwise (replay or too old)} \end{cases}$$

### Window Memory

$$\text{Memory} = \frac{W}{8} = \frac{2048}{8} = 256 \text{ bytes per session}$$

### Maximum Reordering Tolerance

Packets can arrive up to $W = 2048$ out of order without being rejected:

$$\text{Max reorder} = 2048 \text{ packets}$$

At 1500 bytes/packet and 1 Gbps: $\frac{2048 \times 1500 \times 8}{10^9} = 24.6 \text{ ms}$ of reordering tolerance.

---

## 5. Key Rotation — Rekeying

### Transport Key Lifetime

WireGuard rekeys every 2 minutes OR every $2^{64} - 1$ messages (whichever comes first):

$$T_{rekey} = \min(120 \text{ seconds}, 2^{64} - 1 \text{ messages})$$

### Nonce Exhaustion

With 8-byte (64-bit) counter:

$$N_{max} = 2^{64} - 1 = 1.8 \times 10^{19}$$

At 1 million packets/second:

$$T_{exhaust} = \frac{2^{64}}{10^6} = 1.8 \times 10^{13} \text{ seconds} = 585{,}000 \text{ years}$$

The 2-minute rekey happens long before nonce exhaustion.

### Forward Secrecy

Each rekey generates new ephemeral keys:

$$E_{new}, S_{new} \leftarrow \text{fresh DH keypair}$$

Compromising long-term keys does not reveal past session keys (ephemeral DH provides forward secrecy).

---

## 6. Cookie Mechanism — DoS Protection

### Under-Load Response

When under load, WireGuard responds with a cookie instead of completing the handshake:

$$\text{Cookie} = \text{BLAKE2s-MAC-128}(K_{cookie}, \text{source IP + port})$$

Where:

$$K_{cookie} = \text{BLAKE2s}(\text{random secret} \| \lfloor t / 120 \rfloor)$$

The cookie secret rotates every 2 minutes.

### DoS Mitigation

| State | Server Action | CPU Cost |
|:---|:---|:---:|
| Normal load | Full handshake | 0.5 ms |
| Under load | Return cookie challenge | 0.01 ms |
| With valid cookie | Complete handshake | 0.5 ms |

The cookie mechanism forces attackers to:
1. Have a real source IP (not spoofed)
2. Complete a round-trip (cookie exchange)
3. Pay full handshake CPU cost per connection

### Rate Limiting Effect

Without cookies: attacker can force $\frac{1}{0.5 \text{ms}} = 2000$ handshakes/second per core.

With cookies: attacker must first receive and respond to cookie, adding RTT latency:

$$R_{attack} = \frac{1}{\text{RTT} + 0.5\text{ms}}$$

At 100ms RTT: $R = \frac{1}{0.1005} = 9.95$ handshakes/second — a 200x reduction.

---

## 7. Allowed IPs — Cryptokey Routing

### Routing Table

WireGuard's "cryptokey routing" maps IP ranges to peer public keys:

$$\text{route}(\text{dest IP}) = \text{peer whose AllowedIPs contains dest IP}$$

This is a longest-prefix match in a trie/radix tree:

$$T_{lookup} = O(\text{prefix length}) = O(32) \text{ for IPv4, } O(128) \text{ for IPv6}$$

### Configuration Example

| Peer | Public Key | AllowedIPs | Effect |
|:---|:---|:---|:---|
| Peer A | `xTIB...` | 10.0.0.2/32 | Single host |
| Peer B | `jNGL...` | 10.0.1.0/24 | Subnet |
| Peer C | `aF1t...` | 0.0.0.0/0 | Full tunnel (all traffic) |

### Ingress Filtering

Packets from a peer are only accepted if the source IP is in that peer's AllowedIPs:

$$\text{Accept}(p, \text{peer}) \iff p.\text{src} \in \text{AllowedIPs}(\text{peer})$$

This prevents IP spoofing within the VPN — a peer authenticated as key $K$ can only send from their assigned IPs.

---

## 8. Performance Comparison

### Throughput

| VPN | Cipher | Throughput | Latency Added |
|:---|:---|:---:|:---:|
| WireGuard | ChaCha20-Poly1305 | 3-5 Gbps | 0.1-0.3 ms |
| OpenVPN (UDP) | AES-256-GCM | 0.5-1 Gbps | 1-5 ms |
| IPsec (IKEv2) | AES-256-GCM | 2-4 Gbps | 0.5-2 ms |

### Code Size and Attack Surface

$$\text{Attack surface} \propto \text{lines of code}$$

| VPN | Lines of Code | CVEs (2018-2024) |
|:---|:---:|:---:|
| WireGuard | ~4,000 | 2 |
| OpenVPN | ~100,000 | 15+ |
| strongSwan (IPsec) | ~400,000 | 20+ |

Ratio: WireGuard has ~25x less code than OpenVPN, correlating with fewer vulnerabilities.

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Curve25519 $a \cdot G$ | Elliptic curve scalar multiply | Key generation/exchange |
| Noise_IK (4 DH ops) | Protocol state machine | Handshake |
| ChaCha20-Poly1305 | AEAD stream cipher | Data encryption |
| Sliding window bitmap | Bitfield | Anti-replay |
| Cookie BLAKE2s-MAC | Keyed hash | DoS protection |
| Longest-prefix match | Trie lookup | Cryptokey routing |
| 2-minute rekey | Time-bounded keys | Forward secrecy |

---

*WireGuard proves that cryptographic protocols can be both simple and secure — its ~4,000 lines implement a formally analyzed protocol with no cipher negotiation, no legacy compatibility baggage, and near-line-rate performance.*
