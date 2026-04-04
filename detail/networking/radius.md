# The Mathematics of RADIUS — Authentication Timing, Shared Secret Security, and Accounting

> *RADIUS (Remote Authentication Dial-In User Service) governs network access through a request-response protocol with mathematically significant properties: MD5-based password hiding, retry timing that determines authentication latency, and accounting data that feeds capacity planning.*

---

## 1. Authentication Exchange — Timing Model

### The Request-Response Cycle

$$T_{auth} = T_{NAS \to Server} + T_{process} + T_{Server \to NAS}$$

For a single server, single attempt:

$$T_{auth} = RTT + T_{backend}$$

### Retry and Failover Timing

RADIUS uses UDP — no guaranteed delivery. Retransmission:

$$T_{retry}(n) = T_{initial} + n \times T_{interval}$$

Total time before declaring server dead:

$$T_{dead} = R \times T_{interval}$$

Where $R$ = max retries (typically 3).

### Multi-Server Failover

With $S$ servers and $R$ retries per server:

$$T_{worst} = S \times R \times T_{interval}$$

| Servers | Retries | Interval | Worst Case |
|:---:|:---:|:---:|:---:|
| 1 | 3 | 5 sec | 15 sec |
| 2 | 3 | 5 sec | 30 sec |
| 3 | 3 | 5 sec | 45 sec |
| 2 | 3 | 3 sec | 18 sec |

### Impact on User Experience

802.1X authentication must complete before network access. If the RADIUS server is slow:

$$T_{total} = T_{EAP} + T_{RADIUS} + T_{VLAN\_assignment}$$

For EAP-TLS with certificate validation:

| Component | Time |
|:---|:---:|
| EAP identity exchange | 100 ms |
| TLS handshake | 200 ms |
| RADIUS round trip | 50 ms |
| VLAN assignment | 10 ms |
| **Total** | **360 ms** |

---

## 2. Password Hiding — MD5 XOR Chain

### The Algorithm (RFC 2865)

The User-Password attribute is encrypted using MD5:

$$b_1 = p_1 \oplus MD5(S \| RA)$$
$$b_2 = p_2 \oplus MD5(S \| b_1)$$
$$b_n = p_n \oplus MD5(S \| b_{n-1})$$

Where:
- $p_i$ = 16-byte password blocks (padded with zeros)
- $S$ = shared secret
- $RA$ = Request Authenticator (16 random bytes)
- $\oplus$ = XOR

### Security Analysis

| Attack | Complexity | Mitigation |
|:---|:---:|:---|
| Brute-force shared secret | $O(2^{|S| \times 8})$ | Use long secrets (16+ chars) |
| MD5 collision | $O(2^{64})$ (birthday) | Use RADIUS over TLS (RadSec) |
| Offline dictionary | $O(|D|)$ dictionary size | Complex secrets, rate limiting |
| Replay attack | Trivial without RA | Request Authenticator provides nonce |

### Shared Secret Entropy

$$H = L \times \log_2(C)$$

Where $L$ = length, $C$ = character set size.

| Secret Length | Character Set | Entropy (bits) | Brute Force Time (10^9/sec) |
|:---:|:---:|:---:|:---:|
| 8 chars | lowercase (26) | 37.6 | 137 sec |
| 8 chars | mixed+digits (62) | 47.6 | 55 hours |
| 16 chars | mixed+digits (62) | 95.3 | $10^{12}$ years |
| 32 chars | ASCII printable (95) | 210 | Heat death of universe |

**Minimum recommendation:** 16 characters of mixed case + digits + symbols.

---

## 3. Response Authenticator — Integrity Check

### The Formula

$$\text{ResponseAuth} = MD5(\text{Code} \| \text{ID} \| \text{Length} \| \text{RequestAuth} \| \text{Attributes} \| \text{Secret})$$

This proves:
1. The response came from a server that knows the shared secret
2. The response hasn't been tampered with
3. The response matches the specific request (via RequestAuth binding)

### Verification Cost

One MD5 computation per response: ~0.001 ms on modern hardware.

$$\text{Verifications/sec} = \frac{1}{T_{MD5}} \approx 1,000,000 \text{/sec per core}$$

---

## 4. Accounting — Data Volume and Session Math

### Accounting Record Types

| Type | Acct-Status-Type | When Sent |
|:---|:---:|:---|
| Start | 1 | Session begins |
| Stop | 2 | Session ends |
| Interim-Update | 3 | Periodic (every $T_{interim}$) |

### Record Volume

$$R_{total} = N_{sessions} \times (2 + \frac{T_{session}}{T_{interim}})$$

Where $T_{session}$ = average session duration, $T_{interim}$ = interim update interval.

| Users | Avg Session | Interim Interval | Records/Day |
|:---:|:---:|:---:|:---:|
| 1,000 | 8 hours | 5 min | $1000 \times (2 + 96) = 98,000$ |
| 10,000 | 1 hour | 10 min | $10000 \times (2 + 6) = 80,000$ |
| 100,000 | 30 min | None | $100000 \times 2 = 200,000$ |

### Storage Sizing

Each accounting record: ~500 bytes average.

$$\text{Storage/day} = R_{total} \times 500 \text{ bytes}$$

| Records/Day | Storage/Day | Storage/Year |
|:---:|:---:|:---:|
| 100,000 | 50 MB | 18 GB |
| 1,000,000 | 500 MB | 183 GB |
| 10,000,000 | 5 GB | 1.83 TB |

---

## 5. Server Capacity Planning

### Transactions per Second

$$TPS_{required} = \frac{N_{users} \times A_{per\_user}}{T_{peak}}$$

Where $A_{per\_user}$ = authentications per user per peak period.

### Worked Example: Enterprise WiFi

- 10,000 users
- 4 authentications per user per day (roaming, reconnects)
- Peak period: 2 hours (morning login surge)

$$TPS = \frac{10,000 \times 4}{2 \times 3600} = 5.6 \text{ TPS}$$

### Server Capacity

| Server Type | TPS Capacity | Users Supported (4 auth/day) |
|:---|:---:|:---:|
| Single-core (software) | 500 | 900,000 |
| Multi-core (4 cores) | 2,000 | 3,600,000 |
| Hardware appliance | 10,000 | 18,000,000 |

RADIUS is rarely the bottleneck — the backend (LDAP, SQL, certificate validation) typically limits throughput.

### Backend Latency Amplification

$$T_{auth} = T_{RADIUS} + T_{backend}$$

| Backend | Query Time | Auth TPS (serial) | Auth TPS (10 threads) |
|:---|:---:|:---:|:---:|
| Local file | 0.01 ms | 100,000 | 100,000 |
| LDAP (local) | 1 ms | 1,000 | 10,000 |
| LDAP (remote, 5ms RTT) | 10 ms | 100 | 1,000 |
| SQL database | 5 ms | 200 | 2,000 |

---

## 6. Proxy Chains — Latency Accumulation

### The Problem

RADIUS supports proxy chains: NAS → Proxy1 → Proxy2 → Home Server.

$$T_{total} = \sum_{i=1}^{P+1} RTT_i + T_{home}$$

Where $P$ = number of proxies.

| Hops | RTT per Hop | Home Processing | Total |
|:---:|:---:|:---:|:---:|
| 0 (direct) | 5 ms | 10 ms | 15 ms |
| 1 proxy | 5+5 ms | 10 ms | 20 ms |
| 2 proxies | 5+5+5 ms | 10 ms | 25 ms |
| 3 proxies | 5+5+5+5 ms | 10 ms | 30 ms |

### Realm Routing

Proxy routing is based on realm (e.g., `user@example.com`). Routing table:

$$\text{Realm} \rightarrow \text{Server pool}$$

No complex computation — just string matching or regex on the username.

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $S \times R \times T_{interval}$ | Product | Worst-case failover time |
| $p_n \oplus MD5(S \| b_{n-1})$ | XOR chain | Password hiding |
| $L \times \log_2(C)$ | Entropy | Shared secret strength |
| $N \times (2 + T/T_{int})$ | Rate calculation | Accounting record volume |
| $N \times A / T_{peak}$ | Rate | Required TPS |
| $\sum RTT_i + T_{home}$ | Summation | Proxy chain latency |

## Prerequisites

- shared secret hashing, modular arithmetic, protocol state machines

---

*RADIUS authenticates billions of network access attempts daily — from WiFi logins to VPN connections to ISP dial-in (hence the name). The protocol's math is simple by design, because authentication must be fast, reliable, and never the bottleneck between a user and the network.*
