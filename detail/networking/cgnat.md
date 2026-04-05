# The Mathematics of CGNAT — IPv4 Exhaustion and Address Sharing

> *Carrier-Grade NAT is applied mathematics: how do you divide 4 billion addresses among 8 billion people, 30 billion devices, and preserve the illusion that every endpoint has its own identity?*

---

## 1. IPv4 Exhaustion Math

### The Address Space

IPv4 addresses are 32 bits. The total address space:

$$N_{total} = 2^{32} = 4{,}294{,}967{,}296 \approx 4.3 \text{ billion}$$

But usable unicast addresses are far fewer:

| Block | Purpose | Size | Addresses |
|-------|---------|------|-----------|
| 0.0.0.0/8 | This network | /8 | 16,777,216 |
| 10.0.0.0/8 | RFC 1918 private | /8 | 16,777,216 |
| 100.64.0.0/10 | RFC 6598 shared | /10 | 4,194,304 |
| 127.0.0.0/8 | Loopback | /8 | 16,777,216 |
| 169.254.0.0/16 | Link-local | /16 | 65,536 |
| 172.16.0.0/12 | RFC 1918 private | /12 | 1,048,576 |
| 192.0.0.0/24 | IETF protocol | /24 | 256 |
| 192.0.2.0/24 | Documentation | /24 | 256 |
| 192.168.0.0/16 | RFC 1918 private | /16 | 65,536 |
| 198.18.0.0/15 | Benchmarking | /15 | 131,072 |
| 198.51.100.0/24 | Documentation | /24 | 256 |
| 203.0.113.0/24 | Documentation | /24 | 256 |
| 224.0.0.0/4 | Multicast | /4 | 268,435,456 |
| 240.0.0.0/4 | Reserved/Future | /4 | 268,435,456 |

$$N_{unusable} \approx 592{,}708{,}608$$

$$N_{usable} \approx 4{,}294{,}967{,}296 - 592{,}708{,}608 \approx 3{,}702{,}258{,}688 \approx 3.7 \text{ billion}$$

### The Sharing Ratio

With approximately 5 billion internet-connected devices (and growing), the sharing ratio is:

$$R_{sharing} = \frac{N_{devices}}{N_{available}} = \frac{5 \times 10^9}{3.7 \times 10^9} \approx 1.35$$

This means on average, every public IPv4 address must serve 1.35 devices. But the distribution is uneven: large cloud providers hold massive allocations while developing-region ISPs may have sharing ratios of 10:1 to 100:1.

### ISP-Level Exhaustion Example

A regional ISP with 500,000 subscribers and a /16 allocation:

$$N_{public} = 2^{32-16} = 65{,}536 \text{ addresses}$$

$$R_{sharing} = \frac{500{,}000}{65{,}536} \approx 7.6 \text{ subscribers per public IP}$$

Each subscriber needs some minimum number of ports. With 64,512 usable ports (1024-65535) per IP:

$$\text{Ports per subscriber} = \frac{64{,}512}{7.6} \approx 8{,}488 \text{ ports}$$

This is workable. But at 1,000,000 subscribers:

$$\text{Ports per subscriber} = \frac{64{,}512}{15.3} \approx 4{,}216 \text{ ports}$$

Still viable, but active gamers and P2P users routinely use 2,000+ simultaneous connections.

---

## 2. NAT444 vs DS-Lite Architecture

### NAT444 (Double NAT)

```
                     NAT #1                    NAT #2
CPE (192.168.1.x) ──────> BNG (100.64.x.x) ──────> CGN (203.0.113.x) ──> Internet
   RFC 1918            RFC 6598                    Public
```

**Translation chain:**

$$\text{Subscriber} \xrightarrow{NAT_{CPE}} \text{Shared Space} \xrightarrow{NAT_{CGN}} \text{Public Internet}$$

Problems with NAT444:
- Two layers of NAT state
- Double port consumption
- Increased latency (two translation lookups)
- Two points of failure
- Log correlation requires matching both NAT tables

### DS-Lite (Single NAT, IPv6 Transport)

```
                   IPv4-in-IPv6                NAT
CPE B4 (192.168.1.x) ════════════> AFTR (203.0.113.x) ──> Internet
      IPv6 transport              Decap + NAT44
```

**Translation chain:**

$$\text{Subscriber IPv4} \xrightarrow{encap} \text{IPv6 Tunnel} \xrightarrow{decap + NAT_{AFTR}} \text{Public Internet}$$

Advantages over NAT444:
- Only one NAT translation (at AFTR)
- IPv6 transport throughout the access network
- No RFC 6598 address consumption in the access network
- Simpler logging (single translation point)
- Subscriber identified by IPv6 source address (unique per subscriber)

### Comparison Table

| Factor | NAT444 | DS-Lite |
|--------|--------|---------|
| NAT layers | 2 | 1 |
| Access network protocol | IPv4 | IPv6 |
| Address consumption (access) | 100.64.0.0/10 pool | Zero IPv4 |
| MTU overhead | None | 40 bytes (IPv6 header) |
| CPE requirement | Standard NAT | B4 element (softwire) |
| Logging complexity | Two NAT tables | One NAT table |
| IPv6 readiness | No | Yes (access is IPv6-native) |
| Deployment complexity | Low (add CGN inline) | Medium (IPv6 access + AFTR) |

---

## 3. Port Allocation Algorithms

### Deterministic NAT

Each subscriber is assigned a fixed, algorithmically derived port range. No runtime state changes; the mapping is computable from configuration.

**Algorithm:**

Given:
- $P_{start} = 1024$ (first usable port)
- $P_{end} = 65535$ (last usable port)
- $B$ = block size (ports per subscriber)
- $S$ = subscriber index (0-based, derived from subscriber IP)

$$\text{Subscribers per IP} = K = \left\lfloor \frac{P_{end} - P_{start} + 1}{B} \right\rfloor$$

$$\text{External IP index} = \left\lfloor \frac{S}{K} \right\rfloor$$

$$\text{Port start} = P_{start} + (S \bmod K) \times B$$

$$\text{Port end} = \text{Port start} + B - 1$$

**Worked example:**

Configuration: Pool 203.0.113.0/24, block size $B = 1024$

$$K = \left\lfloor \frac{65535 - 1024 + 1}{1024} \right\rfloor = \left\lfloor \frac{64512}{1024} \right\rfloor = 63$$

For subscriber 100.64.0.50 (index $S = 50$):

$$\text{External IP index} = \left\lfloor \frac{50}{63} \right\rfloor = 0 \rightarrow 203.0.113.1$$

$$\text{Port start} = 1024 + (50 \bmod 63) \times 1024 = 1024 + 51200 = 52224$$

$$\text{Port end} = 52224 + 1023 = 53247$$

**Result:** 100.64.0.50 always maps to 203.0.113.1:52224-53247.

**Reverse lookup** (given public IP 203.0.113.1, port 52500, find subscriber):

$$S_{offset} = \left\lfloor \frac{52500 - 1024}{1024} \right\rfloor = \left\lfloor \frac{51476}{1024} \right\rfloor = 50$$

$$S = (\text{IP index} \times K) + S_{offset} = (0 \times 63) + 50 = 50 \rightarrow 100.64.0.50$$

### Dynamic Port Block Allocation

Subscribers receive port blocks on demand. When a block is exhausted, a new block is allocated.

**State machine:**

```
IDLE
  │── First connection ──> Allocate Block #1 (e.g., ports 1024-1535)
  │
ACTIVE (Block #1)
  │── Block #1 full ────> Allocate Block #2 (e.g., ports 5120-5631)
  │── All connections
  │   in Block #1 close -> Deallocate Block #1 (timer-based)
  │
ACTIVE (Block #2 only)
  │── Idle timeout ─────> Deallocate Block #2
  │
IDLE
```

**Block selection strategies:**

1. **Sequential:** Allocate next available block in order (simple, predictable)
2. **Random:** Randomly select from available blocks (security: harder to predict port range)
3. **Round-robin across IPs:** Distribute subscribers evenly across public IPs

### Port Parity Preservation

RFC 6888 REQ-3 recommends preserving port parity (odd/even) for RTP/RTCP:

- RTP uses even port $P$
- RTCP uses $P + 1$ (odd)

In block allocation, this means blocks should be aligned on even boundaries, or paired ports must be allocated together.

---

## 4. Logging Volume Analysis

### The Logarithmic Cost of NAT Transparency

There is an inverse relationship between logging granularity and address sharing efficiency:

| Method | Log Events per Subscriber per Hour | Storage per 100K Subs per Day |
|--------|-----------------------------------|-------------------------------|
| No NAT (public IP per sub) | 0 | 0 |
| Deterministic NAT | 0 (computable) | ~0 (config backup only) |
| Dynamic port block | 2-10 (block alloc/dealloc) | 1-2 GB |
| Dynamic per-flow | 200-1000 (every connection) | 50-200 GB |

### Per-Flow Logging Cost Model

$$V_{daily} = N_{subs} \times C_{avg} \times R_{bytes} \times 24$$

Where:
- $N_{subs}$ = number of subscribers
- $C_{avg}$ = average connections per subscriber per hour
- $R_{bytes}$ = bytes per log record (typically 100-200 bytes for NetFlow v9/IPFIX)

**Realistic values:** $N_{subs} = 100{,}000$, $C_{avg} = 500$, $R_{bytes} = 150$:

$$V_{daily} = 100{,}000 \times 500 \times 150 \times 24 = 180{,}000{,}000{,}000 \text{ bytes} \approx 180 \text{ GB/day}$$

Over one year:

$$V_{yearly} = 180 \times 365 = 65{,}700 \text{ GB} \approx 64 \text{ TB/year}$$

At $0.023/GB for S3 Standard storage: $0.023 \times 65{,}700 = \$1{,}511$ per year just for storage, plus ingest, indexing, and query costs.

### Port Block Logging Cost Model

$$V_{daily} = N_{subs} \times E_{avg} \times R_{bytes} \times 24$$

Where $E_{avg}$ = block allocation events per subscriber per hour (typically 2-5):

$$V_{daily} = 100{,}000 \times 3 \times 100 \times 24 = 720{,}000{,}000 \text{ bytes} \approx 720 \text{ MB/day}$$

**Reduction factor:** $\frac{180 \text{ GB}}{0.72 \text{ GB}} = 250\times$ less logging volume.

### Retention Requirements

Many jurisdictions require NAT log retention for law enforcement:

| Jurisdiction | Typical Retention | Log Type Required |
|-------------|-------------------|-------------------|
| EU (Data Retention Directive) | 6-24 months (varies by country) | Source IP, port, timestamp |
| US (no federal mandate) | ISP discretion (6-18 months typical) | Sufficient for subpoena response |
| Australia | 2 years | Metadata retention mandatory |
| India | 1 year | Per TRAI/DoT regulations |

---

## 5. CGN Scaling Analysis

### Sessions Per Second (SPS)

The CGN must create and destroy NAT bindings at the rate subscribers generate new connections:

$$SPS = N_{subs} \times \frac{C_{peak}}{3600}$$

Where $C_{peak}$ = peak connections per subscriber per hour.

**Example:** 200K subscribers, peak 800 connections/hour:

$$SPS = 200{,}000 \times \frac{800}{3600} \approx 44{,}444 \text{ sessions/sec}$$

A typical hardware CGN (A10, F5, Cisco ASR9K service module) handles 1M-10M SPS. Software CGN on commodity hardware handles 100K-1M SPS.

### Concurrent Sessions

$$S_{concurrent} = N_{subs} \times S_{avg}$$

Where $S_{avg}$ = average concurrent sessions per subscriber (typically 50-200 for residential).

**Example:** 200K subscribers, 150 avg concurrent:

$$S_{concurrent} = 200{,}000 \times 150 = 30{,}000{,}000 = 30\text{M concurrent sessions}$$

### Memory Requirements

Each NAT session entry requires memory:

| Component | Size |
|-----------|------|
| 5-tuple (src IP, src port, dst IP, dst port, proto) | 13 bytes |
| Translated 5-tuple | 13 bytes |
| Timestamps (create, last-used) | 8 bytes |
| Flags, state, counters | 8 bytes |
| Hash table pointers | 16 bytes |
| **Total per session** | **~58 bytes** (rounded to 64 for alignment) |

$$M_{sessions} = S_{concurrent} \times 64 = 30{,}000{,}000 \times 64 = 1{,}920{,}000{,}000 \approx 1.9 \text{ GB}$$

With port block tracking overhead and indexing, practical memory is 2-3x the raw session state.

### Throughput Requirements

$$T_{bps} = N_{subs} \times BW_{avg}$$

**Example:** 200K subscribers, 2 Mbps average utilization:

$$T_{bps} = 200{,}000 \times 2{,}000{,}000 = 400 \text{ Gbps}$$

This exceeds single-box CGN capacity. Solutions:
- Multiple CGN appliances with subscriber-based hash distribution
- Deterministic NAT (stateless, line-rate forwarding on standard routers)
- MAP-T/MAP-E (stateless, distributed to CPE)

---

## 6. RFC 6888 Requirements Deep Dive

### Endpoint-Independent Mapping (EIM)

**Definition:** If an internal endpoint $I$ sends a packet to external endpoint $E_1$ and the CGN creates mapping $I \rightarrow M$, then subsequent packets from $I$ to any external endpoint $E_2$ must use the same mapping $M$.

$$\forall E_1, E_2: \text{map}(I, E_1) = \text{map}(I, E_2) = M$$

**Why it matters:** Without EIM, NAT traversal protocols (STUN, ICE) cannot discover the external mapping because the mapping changes with every new destination. This breaks WebRTC, VoIP, gaming, and P2P.

### Endpoint-Independent Filtering (EIF)

**Definition:** Once mapping $I \rightarrow M$ exists, any external endpoint can send packets to $M$ and they will be forwarded to $I$.

$$\exists \text{map}(I, M) \implies \forall E: \text{packet}(E \rightarrow M) \text{ forwarded to } I$$

**Security trade-off:** EIF is most permissive. Address-Dependent Filtering (ADF) restricts inbound to previously contacted IP addresses. Address and Port-Dependent Filtering (APDF) is most restrictive but breaks many applications.

### Comparison of Filtering Behaviors

```
Scenario: Internal host 10.0.0.1:5000 mapped to 203.0.113.1:40000
          Internal host has contacted 198.51.100.1:80

Inbound packet from 198.51.100.1:80 to 203.0.113.1:40000:
  EIF:  PASS    ADF:  PASS    APDF: PASS

Inbound packet from 198.51.100.1:443 to 203.0.113.1:40000:
  EIF:  PASS    ADF:  PASS    APDF: DROP (port mismatch)

Inbound packet from 198.51.100.2:80 to 203.0.113.1:40000:
  EIF:  PASS    ADF:  DROP    APDF: DROP (IP mismatch)

Inbound packet from 93.184.216.34:12345 to 203.0.113.1:40000:
  EIF:  PASS    ADF:  DROP    APDF: DROP (never contacted)
```

---

## 7. CGN Impact on Applications

### Application Compatibility Matrix

| Application | NAT44 Impact | Mitigation |
|-------------|-------------|------------|
| Web browsing | None | N/A |
| Email (IMAP/SMTP) | None | N/A |
| VoIP (SIP) | Moderate — SIP embeds IPs | ICE/STUN/TURN, disable SIP ALG |
| WebRTC | Low with EIM | STUN/TURN, EIM required |
| Online gaming | Low-Moderate | EIM + sufficient ports |
| P2P (BitTorrent) | High — needs inbound | Port forwarding impossible; UPnP breaks |
| FTP active mode | High — data channel uses PORT | FTP ALG or passive mode |
| IPsec VPN | High — ESP has no ports | NAT-T (UDP 4500 encapsulation) |
| GRE tunnels | Very high — no port multiplexing | Only one subscriber per public IP can use GRE |
| IPv6 transition (6to4) | Broken — requires public IPv4 | Use 6rd or native IPv6 instead |

### Port Exhaustion Detection

A subscriber hitting port limits exhibits these symptoms:
- DNS queries fail intermittently (new UDP ports cannot be allocated)
- Web pages partially load (some HTTP connections fail)
- Gaming disconnects under load
- Applications report "network unreachable" or connection timeout

Detection formula:

$$U_{port} = \frac{S_{active}}{B} \times 100\%$$

Where $S_{active}$ = active sessions, $B$ = allocated port block size. Alert at $U_{port} > 80\%$.

---

## 8. Transition Technology Comparison

### Decision Matrix

| Technology | NAT State | IPv6 Ready | Logging | CPE Changes | Scale Limit |
|-----------|-----------|------------|---------|-------------|-------------|
| NAT444 | Stateful (CGN + CPE) | No | Heavy | None | CGN capacity |
| DS-Lite | Stateful (AFTR) | Yes (access) | Moderate | B4 element | AFTR capacity |
| NAT64/DNS64 | Stateful (NAT64 GW) | Yes (end-to-end) | Moderate | None (DNS64) | NAT64 GW capacity |
| 464XLAT | Stateful (PLAT) | Yes (transport) | Moderate | CLAT (software) | PLAT capacity |
| MAP-T | **Stateless** | Yes | **None** | MAP CE | **Unlimited** |
| MAP-E | **Stateless** | Yes | **None** | MAP CE | **Unlimited** |
| Dual-stack | None | Yes | None | Dual-stack CPE | Address pool |
| 6rd | None (for IPv6) | Yes (tunneled) | None | 6rd CE | BR capacity |

### The Scaling Cliff

Stateful CGN has a fundamental scaling limit: the session table. As subscriber count grows linearly, session table size and logging volume grow linearly, but failure impact grows linearly too (losing a CGN with 30M sessions is catastrophic).

Stateless technologies (MAP-T, MAP-E) eliminate the session table entirely. The "CGN" becomes a simple border relay that translates or decapsulates based on deterministic rules. No session state, no logging, no single point of failure for subscriber sessions.

$$\text{Stateful CGN failure impact} = N_{sessions} \text{ (all drop)}$$

$$\text{MAP-T BR failure impact} = 0 \text{ sessions (stateless, any BR can serve any subscriber)}$$

This is why MAP-T/MAP-E are considered the end-state for IPv4-as-a-service over IPv6 networks.

---

## See Also

- bgp, mpls, ipv4, ipv6, bng, radius, dns, subnetting

## References

- [RFC 6888 — Requirements for Unicast UDP/TCP CGN](https://www.rfc-editor.org/rfc/rfc6888)
- [RFC 6598 — IANA-Reserved IPv4 Prefix for Shared Address Space](https://www.rfc-editor.org/rfc/rfc6598)
- [RFC 6333 — Dual-Stack Lite](https://www.rfc-editor.org/rfc/rfc6333)
- [RFC 6146 — Stateful NAT64](https://www.rfc-editor.org/rfc/rfc6146)
- [RFC 6147 — DNS64](https://www.rfc-editor.org/rfc/rfc6147)
- [RFC 6877 — 464XLAT](https://www.rfc-editor.org/rfc/rfc6877)
- [RFC 7597 — Mapping of Address and Port with Translation (MAP-T)](https://www.rfc-editor.org/rfc/rfc7597)
- [RFC 7599 — Mapping of Address and Port using Translation (MAP-T)](https://www.rfc-editor.org/rfc/rfc7599)
- [RFC 4787 — NAT Behavioral Requirements for Unicast UDP](https://www.rfc-editor.org/rfc/rfc4787)
- [RFC 6052 — IPv6 Addressing of IPv4/IPv6 Translators](https://www.rfc-editor.org/rfc/rfc6052)
