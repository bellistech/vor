# The Mathematics of Tailscale — WireGuard Cryptography and NAT Traversal

> *Tailscale builds a full mesh VPN on WireGuard primitives. The mathematics cover Curve25519 key exchange, ChaCha20-Poly1305 authenticated encryption, NAT traversal probability, DERP relay latency, and mesh scaling as the network grows.*

---

## 1. Curve25519 Key Exchange (Elliptic Curve Cryptography)

### The Problem

WireGuard (and thus Tailscale) uses Curve25519 for key agreement. Each peer generates a static keypair and an ephemeral keypair for every handshake.

### The Formula

Curve25519 operates on the Montgomery curve:

$$y^2 = x^3 + 486662x^2 + x \pmod{2^{255} - 19}$$

Key agreement via scalar multiplication:

$$\text{Shared} = a \cdot B = a \cdot (b \cdot G) = ab \cdot G$$

The security relies on the Elliptic Curve Discrete Logarithm Problem (ECDLP). Best attack (Pollard's rho):

$$T_{\text{attack}} = O\left(\sqrt{\frac{\pi \cdot n}{2}}\right) \approx 2^{126} \text{ operations for 252-bit group}$$

### Security Equivalence

| Curve | Group Order Bits | Security Bits | RSA Equivalent |
|:---:|:---:|:---:|:---:|
| Curve25519 | 252 | 126 | ~3072-bit RSA |
| P-256 | 256 | 128 | ~3072-bit RSA |
| P-384 | 384 | 192 | ~7680-bit RSA |

---

## 2. ChaCha20-Poly1305 (Authenticated Encryption)

### The Problem

WireGuard encrypts all tunnel traffic with ChaCha20-Poly1305 AEAD. This provides both confidentiality and integrity in a single pass.

### The Formula

ChaCha20 quarter-round function on state words $(a, b, c, d)$:

$$a \mathrel{+}= b; \quad d \mathrel{\oplus}= a; \quad d \lll 16$$
$$c \mathrel{+}= d; \quad b \mathrel{\oplus}= c; \quad b \lll 12$$
$$a \mathrel{+}= b; \quad d \mathrel{\oplus}= a; \quad d \lll 8$$
$$c \mathrel{+}= d; \quad b \mathrel{\oplus}= c; \quad b \lll 7$$

Full ChaCha20: 20 rounds (10 column rounds + 10 diagonal rounds), producing 64 bytes of keystream per block.

Poly1305 authentication tag:

$$\text{Tag} = \left(\sum_{i=1}^{n} (c_i + 2^{128}) \cdot r^{n-i+1}\right) \bmod (2^{130} - 5) + s$$

### Throughput

Without AES-NI (e.g., ARM devices), ChaCha20 outperforms AES:

| Cipher | x86-64 (Gbps) | ARM (Gbps) | Ratio on ARM |
|:---:|:---:|:---:|:---:|
| AES-256-GCM | 28.0 | 1.2 | 1.0x |
| ChaCha20-Poly1305 | 18.0 | 3.5 | 2.9x |

---

## 3. Mesh Scaling (Graph Theory)

### The Problem

Tailscale creates a full mesh where each node can communicate directly with every other. The number of potential tunnels grows quadratically.

### The Formula

For $n$ nodes, the maximum number of peer-to-peer tunnels:

$$E = \binom{n}{2} = \frac{n(n-1)}{2}$$

Each node maintains at most $n - 1$ peer entries:

$$\text{Memory per node} = (n - 1) \times S_{\text{peer state}}$$

WireGuard peer state is approximately 320 bytes per peer.

### Worked Examples

| Nodes | Potential Tunnels | Memory per Node | Total State |
|:---:|:---:|:---:|:---:|
| 10 | 45 | 2.8 KB | 28 KB |
| 100 | 4,950 | 31.6 KB | 3.1 MB |
| 1,000 | 499,500 | 319 KB | 312 MB |
| 10,000 | 49,995,000 | 3.1 MB | 31 GB |

In practice, Tailscale lazily establishes tunnels only when traffic flows, keeping active state much smaller.

---

## 4. NAT Traversal Probability (STUN/ICE)

### The Problem

Direct peer-to-peer connections require NAT traversal. Tailscale uses STUN-like probing to punch through NATs. Success depends on NAT type combinations.

### The Formula

NAT traversal success matrix (probability of direct connection):

| | Full Cone | Restricted | Port-Restricted | Symmetric |
|:---:|:---:|:---:|:---:|:---:|
| **Full Cone** | 1.0 | 1.0 | 1.0 | 1.0 |
| **Restricted** | 1.0 | 1.0 | 1.0 | ~0.5 |
| **Port-Restricted** | 1.0 | 1.0 | 1.0 | ~0.3 |
| **Symmetric** | 1.0 | ~0.5 | ~0.3 | ~0.05 |

Expected direct connection rate across a random network:

$$P_{\text{direct}} = \sum_{i,j} P(\text{type}_i) \cdot P(\text{type}_j) \cdot P_{\text{success}}(i,j)$$

### Worked Example

Assuming typical distribution: 30% full cone, 25% restricted, 30% port-restricted, 15% symmetric:

$$P_{\text{direct}} \approx 0.30 \times 1.0 + 0.25 \times 0.95 + 0.30 \times 0.88 + 0.15 \times 0.55 \approx 0.88$$

Approximately 88% of peer pairs can establish direct connections; the remaining 12% use DERP relays.

---

## 5. DERP Relay Latency (Queuing Theory)

### The Problem

When NAT traversal fails, traffic routes through DERP (Designated Encrypted Relay for Packets) servers. The additional hop adds latency.

### The Formula

DERP relay latency:

$$L_{\text{relay}} = L_{\text{A \to DERP}} + L_{\text{processing}} + L_{\text{DERP \to B}}$$

$$L_{\text{relay}} \approx 2 \times L_{\text{direct}} + L_{\text{processing}}$$

For the relay server under load, M/M/1 queue model:

$$L_{\text{queue}} = \frac{1}{\mu - \lambda}$$

Where $\mu$ is service rate and $\lambda$ is arrival rate. Utilization:

$$\rho = \frac{\lambda}{\mu}, \quad L_{\text{avg}} = \frac{\rho}{1 - \rho} \times \frac{1}{\mu}$$

### Worked Example

Direct RTT = 20 ms, DERP processing = 2 ms:

$$L_{\text{relay RTT}} = 2 \times 20 + 2 = 42 \text{ ms}$$

Overhead factor:

$$\text{Overhead} = \frac{42}{20} = 2.1\times$$

---

## 6. WireGuard Handshake (Noise Protocol)

### The Problem

WireGuard uses the Noise_IKpsk2 pattern for handshakes. The initiator and responder perform two Diffie-Hellman operations each.

### The Formula

Noise_IKpsk2 handshake computations:

$$\text{Initiator: } DH(E_i, S_r), \quad DH(S_i, S_r), \quad DH(E_i, E_r)$$
$$\text{Responder: } DH(S_r, E_i), \quad DH(S_r, S_i), \quad DH(E_r, E_i)$$

Total: 3 Curve25519 scalar multiplications per side.

Handshake cost:

$$C_{\text{handshake}} = 3 \times C_{\text{X25519}} + 2 \times C_{\text{HKDF}} + C_{\text{AEAD}}$$

$$C_{\text{handshake}} \approx 3 \times 50\mu s + 2 \times 1\mu s + 0.5\mu s = 152.5 \mu s$$

Handshake rate:

$$R_{\text{max}} = \frac{1}{152.5 \mu s} \approx 6,557 \text{ handshakes/s/core}$$

---

## 7. Key Rotation and Forward Secrecy

### The Problem

WireGuard rotates symmetric keys every 2 minutes or 2^64 messages (whichever comes first), providing forward secrecy.

### The Formula

Key derivation chain using HKDF:

$$K_{n+1} = \text{HKDF}(K_n, \text{DH}(E_i, E_r))$$

Probability of key compromise affecting past traffic:

$$P_{\text{past compromise}} = 0 \quad \text{(forward secrecy)}$$

Number of key rotations per day:

$$R_{\text{daily}} = \frac{24 \times 60}{2} = 720 \text{ rotations/day}$$

Storage for key history (not kept; immediately discarded):

$$S = 0 \text{ bytes (old keys are zeroed from memory)}$$

---

## Prerequisites

- elliptic-curve-cryptography, graph-theory, queuing-theory, networking-fundamentals
