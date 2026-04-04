# The Mathematics of IPsec — Diffie-Hellman, SA Lifecycles, and Encryption Overhead

> *IPsec is a cryptographic framework governed by modular exponentiation (key exchange), symmetric cipher throughput (encryption overhead), and state machine transitions (IKE negotiation). The math determines both security strength and performance cost.*

---

## 1. Diffie-Hellman Key Exchange — Modular Exponentiation

### The Problem

Two parties must agree on a shared secret over an insecure channel. This is the discrete logarithm problem.

### The Algorithm

Given a prime $p$ and generator $g$:

1. Alice chooses random $a$, computes $A = g^a \mod p$, sends $A$
2. Bob chooses random $b$, computes $B = g^b \mod p$, sends $B$
3. Alice computes: $S = B^a \mod p = g^{ab} \mod p$
4. Bob computes: $S = A^b \mod p = g^{ab} \mod p$

Both arrive at the same shared secret $S$.

### Security: The Discrete Logarithm Problem

An attacker sees $g$, $p$, $A = g^a \mod p$ and must find $a$. The best known algorithm (General Number Field Sieve) has complexity:

$$O\left(\exp\left(c \cdot (\ln p)^{1/3} \cdot (\ln \ln p)^{2/3}\right)\right)$$

### DH Group Strengths

| DH Group | Type | Key Size (bits) | Equivalent Symmetric Security | Operations to Break |
|:---:|:---|:---:|:---:|:---:|
| 2 | MODP | 1,024 | ~80-bit | $\sim 2^{80}$ |
| 14 | MODP | 2,048 | ~112-bit | $\sim 2^{112}$ |
| 19 | ECP | 256 | ~128-bit | $\sim 2^{128}$ |
| 20 | ECP | 384 | ~192-bit | $\sim 2^{192}$ |
| 21 | ECP | 521 | ~256-bit | $\sim 2^{256}$ |

### Computation Cost

Modular exponentiation with $n$-bit numbers:

$$T_{DH} = O(n^2 \times \log n) \quad \text{(with fast multiplication)}$$

| DH Group | Key Size | Relative Cost | Typical Time |
|:---:|:---:|:---:|:---:|
| 14 (MODP 2048) | 2,048-bit | 1x (baseline) | ~5 ms |
| 19 (ECP 256) | 256-bit | 0.1x | ~0.5 ms |
| 20 (ECP 384) | 384-bit | 0.3x | ~1.5 ms |
| 21 (ECP 521) | 521-bit | 0.8x | ~4 ms |

ECP (Elliptic Curve) groups provide equivalent security at ~10x lower cost than MODP groups.

---

## 2. IKE State Machine — Phase Timing

### IKEv2 Exchange Sequence

$$T_{IKEv2} = 2 \times RTT + T_{DH} + T_{auth}$$

| Exchange | Messages | Purpose | Time Component |
|:---:|:---:|:---|:---|
| IKE_SA_INIT | 2 (1 RTT) | DH exchange, negotiate crypto | $RTT + T_{DH}$ |
| IKE_AUTH | 2 (1 RTT) | Authentication, establish Child SA | $RTT + T_{auth}$ |

### Worked Example

- RTT = 50 ms, DH Group 19 (ECP-256) = 0.5 ms, RSA-2048 auth = 2 ms:

$$T_{total} = 2(50) + 0.5 + 2 = 102.5 \text{ ms}$$

### IKEv1 Comparison (Main Mode)

$$T_{IKEv1} = 3 \times RTT + T_{DH} + T_{auth} = 152.5 \text{ ms}$$

IKEv2 saves one round trip compared to IKEv1 Main Mode (6 messages → 4 messages).

### Tunnel Setup Rate

$$\text{Tunnels/sec} = \frac{1}{T_{IKE}} \times N_{cores}$$

| Gateway | Cores | DH Group | Tunnels/sec |
|:---|:---:|:---:|:---:|
| Low-end | 2 | 14 (MODP 2048) | ~400 |
| Mid-range | 8 | 19 (ECP 256) | ~16,000 |
| High-end + HSM | 16 | 19 (ECP 256) | ~80,000 |

---

## 3. SA Lifetime and Rekeying Math

### The Problem

Security Associations (SAs) must be rekeyed before the underlying crypto becomes weak. Two limits apply:

$$T_{rekey} = \min(T_{lifetime}, V_{lifetime} / R)$$

Where:
- $T_{lifetime}$ = time-based lifetime (seconds)
- $V_{lifetime}$ = volume-based lifetime (bytes)
- $R$ = data rate (bytes/sec)

### Common SA Lifetimes

| SA Type | Time Lifetime | Volume Lifetime | Purpose |
|:---|:---:|:---:|:---|
| IKE SA | 86,400 sec (24h) | N/A | Control channel |
| Child SA (ESP) | 3,600 sec (1h) | 4 GB | Data encryption |
| Child SA (high-sec) | 900 sec (15min) | 1 GB | Sensitive data |

### Rekeying Frequency

At 1 Gbps sustained throughput:

$$T_{volume} = \frac{4 \times 10^9 \text{ bytes}}{125 \times 10^6 \text{ bytes/sec}} = 32 \text{ sec}$$

The volume limit triggers rekeying every 32 seconds — long before the 3,600-second time limit.

| Data Rate | Volume Limit (4 GB) | Time Limit (1h) | Effective Rekey |
|:---:|:---:|:---:|:---:|
| 10 Mbps | 3,200 sec | 3,600 sec | 3,200 sec (volume) |
| 100 Mbps | 320 sec | 3,600 sec | 320 sec (volume) |
| 1 Gbps | 32 sec | 3,600 sec | 32 sec (volume) |
| 10 Gbps | 3.2 sec | 3,600 sec | 3.2 sec (volume) |

At 10 Gbps, a VPN gateway is rekeying every 3.2 seconds per tunnel — DH computation becomes a bottleneck.

### Rekey Jitter

To prevent all SAs from rekeying simultaneously (thundering herd), implementations add random jitter:

$$T_{actual} = T_{rekey} \times (1 - R_{jitter})$$

Where $R_{jitter}$ is random in $[0, 0.1]$ (10% window). For a 3,600-second lifetime, rekeying happens between 3,240 and 3,600 seconds.

---

## 4. Encryption Overhead — Throughput Math

### ESP Packet Overhead

$$O_{ESP} = \text{SPI}(4) + \text{Seq}(4) + \text{IV}(8\text{-}16) + \text{Pad}(0\text{-}15) + \text{PadLen}(1) + \text{NH}(1) + \text{ICV}(12\text{-}16)$$

### Per-Cipher Overhead

| Cipher | IV | ICV | Total Overhead | Throughput Impact at 1500B MTU |
|:---|:---:|:---:|:---:|:---:|
| AES-128-CBC + HMAC-SHA256 | 16 B | 16 B | 42-57 B | 2.8-3.8% |
| AES-256-GCM | 8 B | 16 B | 34-49 B | 2.3-3.3% |
| ChaCha20-Poly1305 | 8 B | 16 B | 34-49 B | 2.3-3.3% |

### Tunnel Mode Additional Overhead

Tunnel mode adds an outer IP header (20 bytes IPv4, 40 bytes IPv6):

$$O_{tunnel} = O_{ESP} + O_{outer\_header}$$

**Effective MTU in tunnel mode:**

$$MTU_{effective} = MTU_{link} - O_{tunnel}$$

| Mode | Link MTU | Overhead | Effective MTU | Payload Efficiency |
|:---|:---:|:---:|:---:|:---:|
| Transport (AES-GCM) | 1,500 | 34 B | 1,466 B | 97.7% |
| Tunnel IPv4 (AES-GCM) | 1,500 | 54 B | 1,446 B | 96.4% |
| Tunnel IPv6 (AES-GCM) | 1,500 | 74 B | 1,426 B | 95.1% |

### Crypto Throughput (AES-NI)

Modern CPUs with AES-NI achieve:

$$T_{AES-GCM} \approx 5\text{-}10 \text{ Gbps per core}$$

| Cores | AES-GCM Throughput | AES-CBC Throughput |
|:---:|:---:|:---:|
| 1 | 8 Gbps | 2 Gbps |
| 4 | 32 Gbps | 8 Gbps |
| 8 | 64 Gbps | 16 Gbps |

AES-GCM is 4x faster than AES-CBC because GCM parallelizes (CTR mode) while CBC is sequential.

---

## 5. Perfect Forward Secrecy — The Cost of Safety

### The Math

Without PFS, compromising the IKE SA key reveals all past and future Child SA keys (they're derived from the same keying material).

With PFS, each Child SA runs a fresh DH exchange:

$$K_{child} = PRF(DH_{new}, N_i, N_r)$$

**Cost of PFS:**

$$T_{rekey\_PFS} = T_{rekey\_base} + T_{DH}$$

At 32-second rekeying (1 Gbps, 4 GB volume limit) with ECP-256:

$$\text{DH ops/sec} = \frac{1}{32} \times N_{tunnels} = 0.03 \times N_{tunnels}$$

| Concurrent Tunnels | DH ops/sec (PFS) | DH ops/sec (no PFS) |
|:---:|:---:|:---:|
| 100 | 3 | 0 |
| 1,000 | 31 | 0 |
| 10,000 | 312 | 0 |
| 100,000 | 3,125 | 0 |

At 100K tunnels with PFS, the gateway needs ~3,125 DH operations/second — feasible with ECP-256 (~0.5 ms each = ~1.6 cores) but challenging with MODP-2048 (~5 ms each = ~15.6 cores).

---

## 6. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $g^a \mod p$ | Modular exponentiation | DH key exchange |
| $2 \times RTT + T_{DH} + T_{auth}$ | Summation | IKEv2 setup time |
| $\min(T_{life}, V_{life}/R)$ | Minimum function | Effective rekey interval |
| $T \times (1 - R_{jitter})$ | Random scaling | Rekey jitter |
| $MTU - O_{tunnel}$ | Subtraction | Effective MTU |
| $N_{tunnels} / T_{rekey}$ | Rate calculation | DH operations/sec |
| $\exp(c \cdot (\ln p)^{1/3} \cdot (\ln\ln p)^{2/3})$ | Sub-exponential | DLP difficulty |

## Prerequisites

- modular arithmetic, exponential functions, finite fields, cryptography fundamentals

---

*Every VPN tunnel on the internet starts with modular exponentiation — the same math that Diffie and Hellman published in 1976. The numbers got bigger, the curves got elliptic, but the core idea remains: compute forward is easy, compute backward is infeasible.*
