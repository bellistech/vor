# The Mathematics of SLAAC — Interface Identifier Generation, Address Entropy, and Privacy

> *SLAAC transforms a 48-bit MAC address into a 128-bit globally routable address through bit manipulation, hash functions, and cryptographic randomness. The math covers information theory (tracking entropy), birthday-problem collision rates, and the probabilistic guarantees of privacy extensions.*

---

## 1. EUI-64 — Deterministic Bit Manipulation (Binary Arithmetic)

### The Algorithm

Given a 48-bit MAC address $M$, the 64-bit Interface Identifier is:

$$IID_{EUI64} = (M[0:23] \oplus 0x020000) \| \text{0xFFFE} \| M[24:47]$$

Where $\oplus$ denotes XOR on bit 6 (the Universal/Local bit) of the first byte.

### Step-by-Step Binary

MAC: `00:1A:2B:3C:4D:5E` = `00000000:00011010:00101011:00111100:01001101:01011110`

```
Step 1: Split at 24-bit boundary
  OUI:       00000000:00011010:00101011
  Device ID: 00111100:01001101:01011110

Step 2: Insert 0xFFFE (16 bits)
  00000000:00011010:00101011:11111111:11111110:00111100:01001101:01011110

Step 3: Flip bit 6 of byte 0
  00000000 → 00000010 (bit 6: 0→1, meaning "locally administered")
  Result: 02:1A:2B:FF:FE:3C:4D:5E
```

### Information Content

EUI-64 preserves the full 48 bits of the MAC address:

$$H(IID_{EUI64}) = H(MAC) = 48 \text{ bits (at most)}$$

The remaining 16 bits are constant (`FF:FE`), so the effective entropy is:

$$H_{effective} = 48 \text{ bits} - H_{OUI} \approx 24 \text{ bits}$$

Since the OUI (first 24 bits) identifies the manufacturer (often guessable), only the device portion (last 24 bits) provides real entropy for distinguishing devices.

---

## 2. Stable Privacy Addresses — Hash-Based Generation (RFC 7217)

### The Algorithm

$$IID_{stable} = F(\text{Prefix}, \text{Interface}, \text{Network\_ID}, \text{DAD\_Counter}, \text{Secret\_Key})[0:63]$$

Where $F$ is a pseudorandom function (typically SHA-256 truncated to 64 bits).

### Entropy Analysis

The secret key provides the primary entropy source:

$$H(IID_{stable}) = \min(H(\text{Secret\_Key}), 64) \text{ bits}$$

With a 128-bit secret key: $H(IID_{stable}) = 64$ bits (maximum for a 64-bit IID).

### Unlinkability Property

For two different prefixes $P_1$ and $P_2$:

$$P(IID_{stable}(P_1) = IID_{stable}(P_2)) = \frac{1}{2^{64}}$$

The addresses generated for different networks are computationally independent. An observer seeing a device on network A cannot predict its address on network B (assuming the PRF is secure).

### Collision Probability

On a subnet with $n$ hosts using stable-privacy IIDs:

$$P(\text{collision}) \approx \frac{n^2}{2 \times 2^{64}} = \frac{n^2}{3.69 \times 10^{19}}$$

| Hosts ($n$) | $P(\text{collision})$ |
|:---:|:---:|
| 1,000 | $2.7 \times 10^{-14}$ |
| 1,000,000 | $2.7 \times 10^{-8}$ |
| $10^9$ | $2.7 \times 10^{-2}$ |

For any realistic subnet size, collisions are effectively impossible.

---

## 3. Temporary Addresses — Randomized Privacy (RFC 8981)

### Generation

$$IID_{temp} = \text{CSPRNG}(64 \text{ bits}) \quad \text{with bit 6 = 0}$$

The constraint on bit 6 (U/L bit = 0, meaning "universal") ensures temporary addresses are distinguishable from EUI-64 addresses (which have bit 6 = 1 for locally administered).

Effective entropy:

$$H(IID_{temp}) = 63 \text{ bits}$$

### Rotation Model

Temporary addresses follow a renewal process with period $T_{preferred}$:

$$N_{addresses}(t) = \left\lfloor \frac{t}{T_{preferred}} \right\rfloor + 1$$

With default $T_{preferred} = 86{,}400\text{s}$ (24 hours):

| Duration | Addresses Generated |
|:---:|:---:|
| 1 day | 2 |
| 1 week | 8 |
| 1 month | 31 |
| 1 year | 366 |

### Tracking Resistance

An observer who samples a host's address at random time $t$ can link two observations only if they fall within the same preferred lifetime window:

$$P(\text{linkable}) = \frac{T_{preferred}}{T_{observation\_span}}$$

For 24-hour rotation with a 30-day observation window:

$$P(\text{linkable}) = \frac{86{,}400}{2{,}592{,}000} = 0.033 = 3.3\%$$

Reducing $T_{preferred}$ to 3600s (1 hour):

$$P(\text{linkable}) = \frac{3{,}600}{2{,}592{,}000} = 0.0014 = 0.14\%$$

---

## 4. DAD Collision Analysis (Birthday Problem)

### Single Attempt

With $n$ hosts on a /64 using random IIDs, the probability a newly generated address collides with an existing one:

$$P(\text{single collision}) = 1 - \left(1 - \frac{1}{2^{64}}\right)^n \approx \frac{n}{2^{64}}$$

### Expected DAD Attempts Before Success

Modeled as a geometric random variable with success probability $p = 1 - n/2^{64}$:

$$E[\text{attempts}] = \frac{1}{p} = \frac{1}{1 - n/2^{64}} \approx 1 + \frac{n}{2^{64}}$$

For any realistic $n < 10^{12}$, this is indistinguishable from 1.

### DAD with Multiple Probes

With $k$ DAD probes and link error rate $\epsilon$, the probability of false negative (duplicate exists but all probes lost):

$$P(\text{false negative}) = \epsilon^k$$

The probability of false positive (no duplicate, but interference causes spurious NA):

$$P(\text{false positive}) \leq k \times P(\text{spurious NA per probe})$$

### DAD Latency

$$T_{DAD} = k \times T_{retrans}$$

With default $k=1$, $T_{retrans}=1\text{s}$: $T_{DAD} = 1\text{s}$.

Total SLAAC address acquisition time:

$$T_{total} = T_{RS} + T_{RA\_response} + T_{DAD} \approx 0 + 0.5 + 1.0 = 1.5\text{s}$$

---

## 5. Address Space Utilization (Combinatorics)

### Addresses Per Host

A single interface may have simultaneously:

| Address Type | Count | Source |
|:---|:---:|:---|
| Link-local | 1 | Always generated |
| SLAAC stable (per prefix) | 1 per prefix | One per RA prefix with A=1 |
| Temporary (per prefix) | 1-2 per prefix | Active + deprecated |
| DHCPv6 | 0-1 | If M flag set |

With $p$ prefixes advertised and privacy extensions:

$$N_{addresses} = 1 + p + 2p + \delta_{DHCPv6} = 1 + 3p + \delta_{DHCPv6}$$

For a typical dual-stack host with one prefix: $N = 1 + 3 + 0 = 4$ addresses.

### /64 Utilization

Even with maximum addresses per host:

$$\text{Utilization} = \frac{N_{hosts} \times N_{addr/host}}{2^{64}}$$

For 10,000 hosts with 5 addresses each:

$$\text{Utilization} = \frac{50{,}000}{1.8 \times 10^{19}} = 2.7 \times 10^{-15}$$

The /64 is effectively inexhaustible.

---

## 6. Prefix Lifetime Mathematics (Renewal Theory)

### Address Validity Window

Each address has a validity interval $[t_0, t_0 + T_{valid}]$ and a preference interval $[t_0, t_0 + T_{preferred}]$:

$$\text{deprecated}(t) = \begin{cases} \text{false} & \text{if } t \leq t_0 + T_{preferred} \\ \text{true} & \text{if } t_0 + T_{preferred} < t \leq t_0 + T_{valid} \\ \text{invalid} & \text{if } t > t_0 + T_{valid} \end{cases}$$

### RA Refresh — Renewal Process

Periodic RAs reset the lifetime countdown. If RA interval is $T_{RA}$ and valid lifetime is $T_{valid}$:

$$\text{Address stable iff } T_{RA} < T_{valid}$$

If the router fails, the address survives for at most $T_{valid}$ seconds after the last RA.

### Renumbering Transition

During renumbering, old prefix advertised with reduced lifetime $T'_{valid}$ and new prefix with full lifetime:

$$T_{transition} = T'_{valid}$$

Connections on old addresses drain during $[0, T'_{valid}]$ while new connections use new addresses immediately.

---

## 7. Summary of Formulas

| Formula | Domain | Application |
|:---|:---|:---|
| $M \oplus 0x020000 \| \text{FFFE} \| M'$ | Bit manipulation | EUI-64 generation |
| $F(\text{prefix}, \text{iface}, \text{secret})[0:63]$ | Hash function | Stable-privacy IID |
| $\text{CSPRNG}(64)$ | Cryptographic randomness | Temporary IID |
| $n^2 / (2 \times 2^{64})$ | Birthday problem | IID collision probability |
| $T_{preferred} / T_{span}$ | Probability | Tracking linkability |
| $\epsilon^k$ | Probability | DAD false negative rate |
| $k \times T_{retrans}$ | Linear | DAD completion time |

---

*SLAAC's elegance lies in eliminating centralized state: every host independently generates addresses with collision probability so low that the birthday paradox requires billions of hosts per subnet before it matters. The evolution from EUI-64 (zero privacy) through stable-privacy (per-network consistency) to temporary addresses (full unlinkability) traces a mathematical path from determinism through pseudorandomness to cryptographic entropy.*

## Prerequisites

- binary arithmetic and bitwise operations, hash functions and entropy, birthday problem and collision probability

## Complexity

- **Beginner:** Understand EUI-64 derivation from MAC address and why /64 prefix length is required for SLAAC.
- **Intermediate:** Analyze collision probability for random IIDs and evaluate privacy tradeoffs between address generation methods.
- **Advanced:** Model tracking resistance under adversarial observation and design renumbering strategies with bounded transition times.
