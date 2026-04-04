# The Mathematics of OpenVPN — Cryptographic Channels and Tunnel Performance

> *OpenVPN layers SSL/TLS over UDP or TCP to create encrypted tunnels. The mathematics span asymmetric key exchange complexity, symmetric cipher throughput, HMAC authentication overhead, and the bandwidth cost of encapsulation across tun and tap modes.*

---

## 1. RSA Key Exchange (Number Theory)

### The Problem

OpenVPN's PKI relies on RSA for certificate-based authentication. Security rests on the computational difficulty of factoring large semiprimes.

### The Formula

RSA key generation selects two large primes $p$ and $q$, computes $n = pq$, and derives the public exponent $e$ and private exponent $d$:

$$d \equiv e^{-1} \pmod{\lambda(n)}, \quad \lambda(n) = \text{lcm}(p-1, q-1)$$

Encryption and decryption:

$$c = m^e \bmod n, \quad m = c^d \bmod n$$

### Security Strength

The best-known factoring algorithm is the General Number Field Sieve (GNFS):

$$T_{\text{GNFS}}(n) = \exp\left(\left(\frac{64}{9}\right)^{1/3} (\ln n)^{1/3} (\ln \ln n)^{2/3}\right)$$

| RSA Key Size | Security Bits | GNFS Operations (approx.) |
|:---:|:---:|:---:|
| 2048 | 112 | $2^{112}$ |
| 3072 | 128 | $2^{128}$ |
| 4096 | 152 | $2^{152}$ |

---

## 2. Diffie-Hellman Key Agreement (Discrete Logarithm)

### The Problem

OpenVPN uses Diffie-Hellman (DH) parameters to establish forward-secret session keys. The `--dh` parameter file contains a large prime $p$ and generator $g$.

### The Formula

Each party selects a private value ($a$, $b$) and computes a public value:

$$A = g^a \bmod p, \quad B = g^b \bmod p$$

$$K_{\text{shared}} = B^a \bmod p = A^b \bmod p = g^{ab} \bmod p$$

For Elliptic Curve Diffie-Hellman (ECDH), the equivalent is scalar multiplication on a curve:

$$K = a \cdot B = b \cdot A = ab \cdot G$$

### DH Parameter Generation Time

Generating safe primes ($p = 2q + 1$ where $q$ is also prime):

$$E[\text{candidates}] \approx \frac{(\ln p)^2}{2}$$

| DH Size | Approx. Generation Time |
|:---:|:---:|
| 2048-bit | 5-30 seconds |
| 4096-bit | 1-10 minutes |
| 8192-bit | 30-120 minutes |

---

## 3. AES-GCM Throughput (Symmetric Cipher)

### The Problem

After key exchange, OpenVPN encrypts data with a symmetric cipher. AES-256-GCM is the modern default. Throughput depends on hardware AES-NI support and block processing.

### The Formula

AES operates on 128-bit blocks. GCM mode adds authentication with GHASH:

$$C_i = E_K(\text{counter}_i) \oplus P_i$$

$$\text{Tag} = \text{GHASH}_H(A, C) \oplus E_K(J_0)$$

Throughput with AES-NI:

$$\text{Throughput}_{\text{AES-NI}} \approx \frac{\text{clock speed} \times \text{blocks per cycle}}{\text{rounds}}$$

AES-256 uses 14 rounds. With AES-NI pipelining (~1 block per cycle):

$$\text{Throughput} \approx \frac{f_{\text{CPU}}}{14} \times 128 \text{ bits}$$

### Worked Example

A 3 GHz CPU with AES-NI:

$$\text{Throughput} = \frac{3 \times 10^9}{14} \times 128 = 27.4 \text{ Gbps (single core, theoretical)}$$

In practice, OpenVPN's userspace architecture limits throughput to ~500-800 Mbps on modern hardware due to context switching and TUN/TAP overhead.

---

## 4. HMAC Authentication Cost (tls-auth / tls-crypt)

### The Problem

`tls-auth` adds HMAC-SHA256 to the control channel to filter unauthenticated packets before TLS processing. `tls-crypt` additionally encrypts the control channel.

### The Formula

HMAC computation:

$$\text{HMAC}(K, m) = H\big((K' \oplus \text{opad}) \| H((K' \oplus \text{ipad}) \| m)\big)$$

Cost per packet:

$$C_{\text{tls-auth}} = 2 \times C_{\text{SHA-256}}(\text{header} + \text{payload})$$

$$C_{\text{tls-crypt}} = C_{\text{AES-256-CTR}}(\text{control packet}) + C_{\text{HMAC-SHA-256}}$$

### Per-Packet Overhead

| Mode | Additional Bytes | CPU Cost per Packet |
|:---:|:---:|:---:|
| No auth | 0 | 0 |
| tls-auth | 20 (HMAC) | ~0.2 us |
| tls-crypt | 48 (IV + HMAC) | ~0.5 us |

---

## 5. Encapsulation Overhead (MTU Mathematics)

### The Problem

VPN tunneling wraps each packet in additional headers. The effective payload shrinks with each layer of encapsulation.

### The Formula

For a UDP tun-mode tunnel:

$$\text{Overhead} = H_{\text{outer IP}} + H_{\text{UDP}} + H_{\text{OpenVPN}} + H_{\text{cipher}} + H_{\text{HMAC}}$$

$$\text{Overhead} = 20 + 8 + 2 + 16_{\text{(IV)}} + 20_{\text{(HMAC)}} = 66 \text{ bytes}$$

Effective MTU:

$$\text{MTU}_{\text{effective}} = \text{MTU}_{\text{link}} - \text{Overhead}$$

$$\text{MTU}_{\text{effective}} = 1500 - 66 = 1434 \text{ bytes}$$

### Overhead Percentage

$$\text{Overhead \%} = \frac{\text{Overhead}}{\text{MTU}_{\text{link}}} \times 100 = \frac{66}{1500} \times 100 = 4.4\%$$

For small packets (e.g., VoIP at 160 bytes payload):

$$\text{Overhead \%}_{\text{VoIP}} = \frac{66}{160 + 66} \times 100 = 29.2\%$$

### TCP-over-TCP Problem

When using `proto tcp`, TCP retransmission at both inner and outer layers causes exponential backoff interaction:

$$\text{Effective throughput} \propto \frac{1}{(\text{RTT}_{\text{outer}} + \text{RTT}_{\text{inner}})^2}$$

This is why UDP is strongly preferred for OpenVPN transport.

---

## 6. TLS Handshake Cost (Session Establishment)

### The Problem

Each new client connection requires a full TLS handshake. The computational cost scales with key size and cipher suite.

### The Formula

RSA handshake operations:

$$C_{\text{handshake}} = C_{\text{RSA-sign}} + C_{\text{RSA-verify}} + C_{\text{DH}} + C_{\text{PRF}}$$

With RSA-2048:

$$C_{\text{RSA-sign}} \approx 1.5 \text{ ms}, \quad C_{\text{RSA-verify}} \approx 0.05 \text{ ms}$$

### Connection Rate

Maximum new connections per second (single core):

$$R_{\text{max}} = \frac{1}{C_{\text{handshake}}} \approx \frac{1}{2 \text{ ms}} = 500 \text{ conn/s}$$

For ECDHE-RSA (P-256):

$$C_{\text{ECDHE}} \approx 0.3 \text{ ms}, \quad R_{\text{max}} \approx 1500 \text{ conn/s}$$

---

## 7. Bandwidth Estimation (Capacity Planning)

### The Problem

Estimating required server bandwidth for N concurrent clients with varying traffic profiles.

### The Formula

$$B_{\text{server}} = \sum_{i=1}^{N} \left( b_i \times (1 + o) \right)$$

Where $b_i$ is client $i$'s throughput and $o$ is the encapsulation overhead ratio:

$$o = \frac{H_{\text{tunnel}}}{\text{MTU}_{\text{effective}}}$$

### Worked Example

100 clients, each averaging 10 Mbps, with 4.4% overhead:

$$B_{\text{server}} = 100 \times 10 \times 1.044 = 1044 \text{ Mbps} \approx 1.04 \text{ Gbps}$$

Memory per client (approximate):

$$M_{\text{total}} = N \times (M_{\text{TLS session}} + M_{\text{buffers}}) = 100 \times (64 \text{ KB} + 128 \text{ KB}) = 18.75 \text{ MB}$$

---

## Prerequisites

- number-theory, cryptography, networking-fundamentals, tcp-ip
