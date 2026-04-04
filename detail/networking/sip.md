# The Mathematics of SIP — Transaction State Machines and Timing Analysis

> *SIP's reliability depends on precise timer interactions, exponential retransmission backoff, and state machine transitions. Understanding the mathematics behind these mechanisms reveals why calls fail and how to tune deployments for scale.*

---

## 1. Retransmission Timers (Exponential Backoff)

### Timer A — INVITE Retransmission over UDP

SIP over UDP is unreliable, so the UAC retransmits INVITE requests with exponential backoff. Timer A starts at T1 and doubles each retransmission:

$$T_A(n) = \min(T_1 \times 2^n, T_2)$$

Where:
- $T_1$ = RTT estimate (default 500 ms)
- $T_2$ = maximum retransmit interval (default 4 seconds)
- $n$ = retransmission attempt (0-indexed)

| Attempt ($n$) | Timer A Value | Cumulative Time |
|:---:|:---:|:---:|
| 0 | 500 ms | 500 ms |
| 1 | 1,000 ms | 1,500 ms |
| 2 | 2,000 ms | 3,500 ms |
| 3 | 4,000 ms | 7,500 ms |
| 4 | 4,000 ms (capped at T2) | 11,500 ms |
| 5 | 4,000 ms | 15,500 ms |
| 6 | 4,000 ms | 19,500 ms |

### Timer B — INVITE Transaction Timeout

Timer B defines the total time the client transaction waits for a response:

$$T_B = 64 \times T_1 = 64 \times 500\text{ ms} = 32\text{ s}$$

The value 64 comes from the maximum number of retransmissions in the geometric series. Total retransmission time before Timer B fires:

$$S = T_1 \sum_{n=0}^{k} 2^n = T_1(2^{k+1} - 1)$$

For 7 retransmissions capped at T2: $S \approx 31.5$ s, which fits within the 32-second Timer B window.

---

## 2. Transaction State Machine Complexity

### INVITE Client Transaction States

The INVITE client transaction (ICT) has 4 states: Calling, Proceeding, Completed, Terminated. Transition probabilities depend on network conditions.

$$\text{States} = \{S_C, S_P, S_{Comp}, S_T\}$$

The state transition matrix for a single INVITE under ideal conditions:

| From \ To | Calling | Proceeding | Completed | Terminated |
|:---|:---:|:---:|:---:|:---:|
| Calling | $p_r$ | $p_{1xx}$ | $p_{3-6xx}$ | $p_{timeout}$ |
| Proceeding | 0 | $p_r$ | $p_{2xx} + p_{3-6xx}$ | $p_{timeout}$ |
| Completed | 0 | 0 | $p_{ack}$ | $1 - p_{ack}$ |
| Terminated | 0 | 0 | 0 | 1 |

Where $p_r$ is the probability of remaining in the current state per timer tick.

### Timer D — Wait Time for Response Retransmissions

After entering Completed state, the client transaction waits Timer D before transitioning to Terminated:

$$T_D = \begin{cases} > 32\text{ s} & \text{for UDP} \\ 0\text{ s} & \text{for TCP/TLS} \end{cases}$$

This absorbs retransmitted responses that arrive after the ACK is sent.

---

## 3. Registration Interval Optimization

### Optimal Re-registration Timing

UAs must re-register before the binding expires. RFC 3261 recommends registering when the remaining time drops below a threshold:

$$T_{refresh} = T_{expires} - \delta$$

Where $\delta$ is a random backoff to prevent thundering herd:

$$\delta \in \left[\frac{T_{expires}}{2}, T_{expires} - T_{min}\right]$$

### Registration Storm Analysis

If $N$ UAs register with the same Expires value $E$, the registrar sees:

$$\text{Registrations/sec} = \frac{N}{E}$$

For $N = 50{,}000$ UAs with $E = 3600$ s:

$$\frac{50{,}000}{3600} \approx 13.9 \text{ reg/s (steady state)}$$

After a network outage of duration $D$, all UAs with $T_{remaining} < D$ re-register simultaneously:

$$N_{burst} = N \times \frac{\min(D, E)}{E}$$

For a 10-minute outage: $N_{burst} = 50{,}000 \times \frac{600}{3600} = 8{,}333$ simultaneous registrations.

---

## 4. Proxy Capacity Planning (Queueing Theory)

### Erlang B Model for SIP Trunks

The probability that all SIP trunks are busy (call blocking probability) follows the Erlang B formula:

$$B(N, A) = \frac{\frac{A^N}{N!}}{\sum_{k=0}^{N} \frac{A^k}{k!}}$$

Where:
- $N$ = number of SIP trunks (concurrent call capacity)
- $A$ = offered traffic in Erlangs ($A = \lambda \times h$)
- $\lambda$ = call arrival rate (calls/hour)
- $h$ = average holding time (hours)

| Trunks ($N$) | Erlangs ($A$) | Blocking $B(N,A)$ |
|:---:|:---:|:---:|
| 10 | 5.0 | 1.84% |
| 15 | 10.0 | 1.85% |
| 20 | 13.0 | 1.31% |
| 30 | 21.0 | 1.16% |
| 50 | 37.9 | 1.00% |

### Transactions Per Second (TPS)

A SIP proxy processes each call as multiple transactions. For a basic call:

$$\text{Transactions/call} = T_{INVITE} + T_{ACK} + T_{BYE} = 3$$

With registration and subscription overhead:

$$\text{Total TPS} = \frac{N_{calls} \times 3}{T_{avg}} + \frac{N_{UAs}}{T_{reg}} + \frac{N_{subs}}{T_{sub}}$$

---

## 5. Digest Authentication Hash Computation

### RFC 2617 Response Calculation

The digest response is computed via MD5 (or SHA-256 in RFC 7616):

$$HA_1 = H(\text{username} : \text{realm} : \text{password})$$

$$HA_2 = H(\text{method} : \text{URI})$$

$$\text{response} = H(HA_1 : \text{nonce} : nc : cnonce : qop : HA_2)$$

Where $H$ is the hash function. The nonce provides replay protection, and the nc (nonce count) prevents nonce reuse.

### Computational Cost per Authentication

Each authentication requires 3 hash operations. At scale:

$$\text{Hashes/sec} = 3 \times \text{TPS}_{auth}$$

For 1,000 authentications/sec: 3,000 MD5 operations/sec — trivial for modern CPUs but relevant when considering SHA-256 upgrade impact.

---

## 6. Session Timer Mathematics (RFC 4028)

### Keep-alive Interval

Session timers prevent orphaned calls when BYE is lost. The refresh interval:

$$T_{refresh} = \frac{SE}{2}$$

Where $SE$ is the Session-Expires value. Both parties must send re-INVITE or UPDATE before expiry.

### Minimum Session Interval

$$SE_{min} \leq SE \leq SE_{max}$$

$$SE_{min} = 90 \text{ s (RFC minimum)}$$

If the refresher fails to refresh, the session terminates after:

$$T_{terminate} = SE + \text{grace period}$$

---

## 7. Codec Bandwidth Calculations

### Total Bandwidth per Call

$$BW_{total} = (BW_{codec} + BW_{headers}) \times \frac{1000}{T_{ptime}}$$

Where:
- $BW_{codec}$ = codec bitrate per sample
- $BW_{headers}$ = IP(20) + UDP(8) + RTP(12) = 40 bytes
- $T_{ptime}$ = packetization interval in ms

| Codec | Bitrate | ptime | Payload | Headers | Total/pkt | Bandwidth |
|:---|:---:|:---:|:---:|:---:|:---:|:---:|
| G.711 | 64 kbps | 20 ms | 160 B | 40 B | 200 B | 80 kbps |
| G.711 | 64 kbps | 30 ms | 240 B | 40 B | 280 B | 74.7 kbps |
| G.729 | 8 kbps | 20 ms | 20 B | 40 B | 60 B | 24 kbps |
| G.722 | 64 kbps | 20 ms | 160 B | 40 B | 200 B | 80 kbps |
| Opus | 32 kbps | 20 ms | 80 B | 40 B | 120 B | 48 kbps |

### Capacity Planning

Maximum concurrent calls on a link:

$$N_{calls} = \frac{BW_{link}}{2 \times BW_{call}}$$

Factor of 2 accounts for bidirectional media. A 10 Mbps link with G.711:

$$N_{calls} = \frac{10{,}000{,}000}{2 \times 80{,}000} = 62 \text{ calls}$$

---

*SIP's apparent simplicity hides a deeply stateful protocol where millisecond-level timer precision, exponential backoff curves, and queueing dynamics determine whether your phone system handles 50 users or 50,000.*

## Prerequisites

- exponential functions, queueing theory basics, hash functions and cryptographic primitives

## Complexity

- **Beginner:** Timer calculations and retransmission backoff series
- **Intermediate:** Erlang B blocking probability and capacity planning
- **Advanced:** State machine analysis and registration storm modeling under failure conditions
