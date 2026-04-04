# The Mathematics of Kerberos — Cryptographic Timestamps, Ticket Lifetimes, and Key Derivation

> *Kerberos V5 security rests on symmetric cryptography, strict time bounds, and key derivation functions. The protocol's resistance to replay attacks, offline brute-force, and credential theft can be quantified through the mathematics of clock skew tolerance, encryption strength, and ticket lifetime optimization.*

---

## 1. Clock Skew and Replay Window (Time-Based Security)

### The Problem

Kerberos authenticators include timestamps to prevent replay attacks. The KDC and services accept messages only if the timestamp falls within a skew tolerance $\delta$ (default 5 minutes). The replay window size determines both security and operational fragility.

### The Formula

An authenticator with timestamp $t_a$ is accepted by a verifier with clock $t_v$ if:

$$|t_v - t_a| \leq \delta$$

The replay window covers $2\delta$ seconds. During this window, the verifier must cache authenticators to detect replays. The cache size under load $\lambda$ (authentications/sec):

$$C_{replay} = 2\delta \cdot \lambda$$

Probability of clock skew exceeding $\delta$ given NTP jitter with standard deviation $\sigma$:

$$P_{reject} = 2 \cdot \Phi\left(-\frac{\delta}{\sigma}\right) = \text{erfc}\left(\frac{\delta}{\sigma\sqrt{2}}\right)$$

### Worked Examples

With $\delta = 300\text{s}$ (5 min), $\lambda = 1{,}000$ auth/sec:

| NTP Jitter ($\sigma$) | $P_{reject}$ | Cache Size |
|:---:|:---:|:---:|
| 10 ms | $\approx 0$ | 600,000 |
| 1 s | $\approx 0$ | 600,000 |
| 60 s | $3.1 \times 10^{-7}$ | 600,000 |
| 120 s | 0.012 | 600,000 |

Even with $\sigma = 60\text{s}$ jitter, the 5-minute window provides ample margin. But the replay cache holding 600K entries at high load shows why `GssapiNegotiateOnce` is important -- it limits cache growth.

---

## 2. Encryption Key Strength (Cryptanalysis)

### The Problem

Kerberos derives long-term keys from passwords using string-to-key functions. The effective security depends on the encryption type and the entropy of the input password.

### The Formula

For AES256-CTS-HMAC-SHA1-96 (the recommended enctype), the string-to-key function uses PBKDF2:

$$K = \text{PBKDF2-HMAC-SHA1}(\text{password}, \text{salt}, \text{iterations}, 256)$$

Brute-force cost to crack a password with entropy $H$ bits at rate $r$ guesses/sec:

$$T_{crack} = \frac{2^H}{r \cdot \text{iterations}}$$

For a dictionary of size $D$ with PBKDF2 iterations $i$:

$$T_{dict} = \frac{D \cdot i}{r}$$

### Worked Examples

With $r = 10^9$ hashes/sec (GPU cluster), PBKDF2 iterations $i = 4{,}096$ (MIT default):

| Password Type | Entropy ($H$) | Time to Crack |
|:---|:---:|:---:|
| 4-digit PIN | 13.3 bits | 0.04 ms |
| 8-char lowercase | 37.6 bits | 0.7 hours |
| 8-char mixed + symbols | 52.4 bits | 73 years |
| 12-char mixed + symbols | 78.7 bits | $1.6 \times 10^{10}$ years |
| Random 128-bit key | 128 bits | $1.4 \times 10^{25}$ years |

Service principals with `-randkey` get full 256-bit entropy, making brute-force computationally infeasible. User passwords are the weak link.

---

## 3. Ticket Lifetime Optimization (Risk Analysis)

### The Problem

Longer ticket lifetimes reduce KDC load but increase the window of exposure if a ticket is stolen. The optimal lifetime balances availability against security risk.

### The Formula

KDC authentication load with $U$ users, ticket lifetime $L$ (seconds), and work hours $W$:

$$\lambda_{KDC} = \frac{U}{L} \quad \text{(steady-state renewal rate)}$$

Risk exposure from a stolen TGT -- the attacker can impersonate the user for the remaining lifetime $R$:

$$E[R] = \frac{L}{2} \quad \text{(uniform distribution of theft time)}$$

Expected damage cost with per-second damage rate $d$:

$$E[\text{damage}] = P_{theft} \cdot d \cdot \frac{L}{2}$$

Optimal lifetime minimizing total cost (KDC cost $c_{kdc}$ per auth + expected damage):

$$L^* = \sqrt{\frac{2 \cdot U \cdot c_{kdc}}{P_{theft} \cdot d}}$$

### Worked Examples

With $U = 10{,}000$ users, $c_{kdc} = 0.001$ (normalized), $P_{theft} = 10^{-6}$/sec:

| Damage Rate ($d$) | Optimal $L^*$ | KDC Load | Expected Risk Window |
|:---:|:---:|:---:|:---:|
| 0.01 | 14.1 hours | 0.2/sec | 7 hours |
| 0.10 | 4.5 hours | 0.6/sec | 2.2 hours |
| 1.00 | 1.4 hours | 2.0/sec | 42 min |
| 10.0 | 27 min | 6.2/sec | 13 min |

High-value environments should use shorter ticket lifetimes. The standard 10-hour lifetime suits moderate-risk environments.

---

## 4. Cross-Realm Trust Path Length (Graph Theory)

### The Problem

In multi-realm deployments, a client in realm $A$ accessing a service in realm $C$ may need to traverse intermediate realms. Each hop adds latency and requires a cross-realm TGT. The trust topology determines path length and single points of failure.

### The Formula

For a hierarchical trust tree with depth $d$, the worst-case path between two realms at leaves:

$$\text{Hops}_{tree} = 2d$$

For a full mesh of $n$ realms:

$$\text{Hops}_{mesh} = 1 \quad \text{(direct trust)}$$

$$\text{Trust principals}_{mesh} = n(n-1) \quad \text{(bidirectional)}$$

Total authentication latency for $h$ hops with per-hop TGS-REQ/REP time $T_{tgs}$:

$$T_{total} = T_{as} + h \cdot T_{tgs} + T_{ap}$$

### Worked Examples

With $T_{as} = 15\text{ms}$, $T_{tgs} = 8\text{ms}$, $T_{ap} = 5\text{ms}$:

| Topology | Realms | Max Hops | Latency | Trust Principals |
|:---|:---:|:---:|:---:|:---:|
| Hub-and-spoke | 5 | 2 | 36 ms | 8 |
| Hierarchy (d=3) | 15 | 6 | 68 ms | 28 |
| Full mesh | 5 | 1 | 28 ms | 20 |
| Full mesh | 10 | 1 | 28 ms | 90 |

Hub-and-spoke offers a good balance -- low hop count with manageable trust principal count.

---

## 5. KDC Throughput and Sizing (Queuing Theory)

### The Problem

The KDC must handle AS-REQ and TGS-REQ traffic from all principals. Under-provisioning leads to authentication delays; over-provisioning wastes resources.

### The Formula

Modeling the KDC as an M/M/1 queue with arrival rate $\lambda$ and service rate $\mu$:

$$\text{Utilization: } \rho = \frac{\lambda}{\mu}$$

Average time in system (authentication latency):

$$W = \frac{1}{\mu - \lambda} = \frac{1}{\mu(1 - \rho)}$$

For $k$ KDC replicas (M/M/k queue), probability of queuing:

$$P_{queue} = \frac{(\lambda/\mu)^k}{k!(1-\rho)} \cdot P_0$$

where $P_0 = \left[\sum_{n=0}^{k-1} \frac{(\lambda/\mu)^n}{n!} + \frac{(\lambda/\mu)^k}{k!(1-\rho)}\right]^{-1}$

### Worked Examples

KDC service rate $\mu = 5{,}000$ req/sec per server:

| Users | $\lambda$ (req/sec) | 1 KDC ($\rho$) | Avg Latency | 2 KDCs ($\rho$) |
|:---:|:---:|:---:|:---:|:---:|
| 5,000 | 500 | 0.10 | 0.22 ms | 0.05 |
| 25,000 | 2,500 | 0.50 | 0.40 ms | 0.25 |
| 40,000 | 4,000 | 0.80 | 1.0 ms | 0.40 |
| 48,000 | 4,800 | 0.96 | 5.0 ms | 0.48 |

At $\rho > 0.8$, latency increases sharply. Two KDCs provide both redundancy and headroom.

---

## 6. Pre-Authentication and AS-REP Roasting (Attack Cost)

### The Problem

Without pre-authentication, any client can request an AS-REP containing data encrypted with the target principal's key, enabling offline brute-force. Pre-authentication requires the client to prove knowledge of the key before the KDC issues a ticket.

### The Formula

Without pre-auth (AS-REP roasting), attack cost:

$$C_{attack} = \frac{D}{r} \quad \text{(dictionary size / hash rate)}$$

With pre-auth (encrypted timestamp), the attacker must interact with the KDC for each guess:

$$C_{attack} = D \cdot T_{as} \quad \text{(network-bound, not CPU-bound)}$$

Slowdown factor from enabling pre-auth:

$$\text{Factor} = \frac{r \cdot T_{as}}{1} = r \cdot T_{as}$$

### Worked Examples

With $r = 10^9$ hashes/sec (offline), $T_{as} = 10\text{ms}$ (online), dictionary $D = 10^8$:

| Scenario | Time to Exhaust Dictionary |
|:---|:---:|
| No pre-auth (offline) | 0.1 seconds |
| Pre-auth (online) | 11.6 days |
| Pre-auth + lockout (3 attempts) | Infeasible |

Pre-authentication transforms the attack from a GPU-bound offline crack to a network-bound online attack, buying 10 million times more time.

---

## Prerequisites

- symmetric-cryptography, hash-functions, probability, queuing-theory, graph-theory, pbkdf2
