# The Mathematics of NAT — Port Exhaustion & Connection Table Sizing

> *NAT is a constrained mapping problem: 2^32 internal addresses multiplied by 2^16 ports must fit through a bottleneck of limited public IPs and 65,535 available ports. The math of exhaustion, table sizing, and traversal determines whether your network scales or collapses.*

---

## 1. Port Exhaustion Mathematics (Finite Resource Allocation)

### The Problem

A single NAT public IP has at most 65,535 TCP ports and 65,535 UDP ports. Each outbound connection consumes one (source IP, source port, dest IP, dest port, protocol) tuple. When do we run out?

### The Formula

Maximum concurrent connections per public IP:

$$C_{max} = (P_{high} - P_{low} + 1) \times D$$

Where:
- $P_{high} - P_{low} + 1$ = ephemeral port range size (typically 64,512 ports: 1024-65535)
- $D$ = number of unique destination (IP, port) pairs

For connections to a **single destination** (e.g., all users hitting one website):

$$C_{max}^{single} = P_{high} - P_{low} + 1 = 64,512$$

For connections to **many unique destinations**:

$$C_{max}^{multi} \leq 64,512 \times D_{unique}$$

### Worked Examples

| Scenario | Public IPs | Dest Diversity | Max Connections |
|:---|:---:|:---:|:---:|
| Office → single proxy | 1 | 1 | 64,512 |
| Office → diverse web | 1 | ~10,000 | ~64,512 (port limited) |
| Carrier-grade NAT (CGNAT) | 1 | diverse | 64,512 per IP |
| CGNAT with 16 IPs | 16 | diverse | 1,032,192 |
| Enterprise 2 IPs | 2 | diverse | 129,024 |

### Port Exhaustion Rate

If connections arrive at rate $\lambda$ and each lives for average duration $\bar{d}$:

$$E[\text{active}] = \lambda \times \bar{d}$$

Time to exhaust ports:

$$T_{exhaust} = \frac{C_{max}}{\lambda} \quad \text{if } \lambda \times \bar{d} > C_{max}$$

| Conn/sec $\lambda$ | Avg Duration $\bar{d}$ | Active | 64K Ports Sufficient? |
|:---:|:---:|:---:|:---:|
| 100 | 10s | 1,000 | Yes |
| 1,000 | 30s | 30,000 | Yes |
| 1,000 | 120s | 120,000 | No (need 2+ IPs) |
| 5,000 | 60s | 300,000 | No (need 5+ IPs) |
| 10,000 | 30s | 300,000 | No (need 5+ IPs) |

### Required Public IPs

$$N_{IPs} = \left\lceil \frac{\lambda \times \bar{d}}{C_{max}} \right\rceil$$

---

## 2. Conntrack Table Sizing (Memory Planning)

### The Problem

The Linux conntrack table tracks every NAT'd connection. Too small and connections drop; too large and you waste kernel memory. How to size it?

### The Formula

Each conntrack entry uses:

$$S_{entry} \approx 320 \text{ bytes (kernel 5.x+)}$$

Total memory:

$$M_{conntrack} = N_{max} \times S_{entry}$$

The hash table should have:

$$B_{buckets} = \frac{N_{max}}{4} \quad \text{(4 entries per bucket average)}$$

Each bucket is a pointer: 8 bytes on 64-bit systems:

$$M_{hash} = B_{buckets} \times 8$$

Total kernel memory:

$$M_{total} = M_{conntrack} + M_{hash}$$

### Worked Examples

| nf_conntrack_max | Buckets | Entry Memory | Hash Memory | Total |
|:---:|:---:|:---:|:---:|:---:|
| 65,536 | 16,384 | 20 MB | 128 KB | ~20 MB |
| 131,072 | 32,768 | 40 MB | 256 KB | ~40 MB |
| 262,144 | 65,536 | 80 MB | 512 KB | ~80 MB |
| 524,288 | 131,072 | 160 MB | 1 MB | ~161 MB |
| 1,048,576 | 262,144 | 320 MB | 2 MB | ~322 MB |
| 2,097,152 | 524,288 | 640 MB | 4 MB | ~644 MB |

### Sizing Formula for Production

$$N_{max} = \lambda_{peak} \times \bar{d}_{max} \times 1.5$$

The 1.5x safety factor accounts for connection bursts and stale entries not yet garbage-collected.

| Peak Rate | Max Duration | Recommended nf_conntrack_max | Memory |
|:---:|:---:|:---:|:---:|
| 500/s | 120s | 90,000 → 131,072 | 40 MB |
| 2,000/s | 120s | 360,000 → 524,288 | 160 MB |
| 10,000/s | 60s | 900,000 → 1,048,576 | 320 MB |

---

## 3. NAT Traversal — STUN/TURN Mathematics

### The Problem

NAT breaks peer-to-peer connectivity. STUN discovers the mapping; TURN relays when direct paths fail. What are the success rates and overhead costs?

### NAT Type Classification

| NAT Type | Mapping | Filtering | P2P Success |
|:---|:---|:---|:---:|
| Full Cone | Endpoint-Independent | None | ~100% |
| Restricted Cone | Endpoint-Independent | IP-restricted | ~90% |
| Port-Restricted Cone | Endpoint-Independent | IP+Port-restricted | ~80% |
| Symmetric | Endpoint-Dependent | IP+Port-restricted | ~10% |

### STUN Binding Discovery

STUN reveals the **server-reflexive** (mapped) address. For endpoint-independent mapping:

$$\text{Internal } (IP_i, P_i) \xrightarrow{\text{NAT}} \text{External } (IP_e, P_e)$$

The mapping $(IP_i, P_i) \rightarrow (IP_e, P_e)$ is reused for all destinations. Two peers behind endpoint-independent NATs can exchange reflexive addresses and connect directly.

### Symmetric NAT — Why It Breaks

For symmetric NAT, the mapping depends on the destination:

$$f(IP_i, P_i, IP_d, P_d) \rightarrow (IP_e, P_e)$$

The STUN server sees $(IP_e, P_e)$ mapped for its own address, but the peer will get a different mapping. Probability of guessing the peer's port:

$$P(guess) = \frac{1}{|P_{range}|} \approx \frac{1}{64512} \approx 0.00155\%$$

With $k$ simultaneous guesses (port prediction):

$$P(success) = 1 - \left(1 - \frac{1}{|P|}\right)^k \approx \frac{k}{|P|}$$

You need ~32,256 attempts for 50% success — impractical without predictable port allocation.

### TURN Relay Overhead

TURN relays all media through the server. Bandwidth overhead:

$$B_{TURN} = B_{media} \times 2 + H_{overhead}$$

Where the factor of 2 comes from traffic flowing client->TURN->peer and peer->TURN->client, and $H_{overhead}$ is TURN channel header (4 bytes per message).

For a VoIP call at 64 kbps:

$$B_{TURN} = 64 \times 2 = 128 \text{ kbps per call}$$

TURN server sizing for $N$ simultaneous calls:

$$B_{total} = N \times B_{TURN} + N \times H$$

| Simultaneous Calls | Bandwidth (64 kbps audio) | Bandwidth (2 Mbps video) |
|:---:|:---:|:---:|
| 100 | 12.8 Mbps | 400 Mbps |
| 1,000 | 128 Mbps | 4 Gbps |
| 10,000 | 1.28 Gbps | 40 Gbps |

---

## 4. CGNAT Address Sharing Ratio (Carrier-Grade NAT)

### The Problem

ISPs use CGNAT (RFC 6888) to share public IPv4 addresses among subscribers. What is the maximum sharing ratio?

### The Formula

RFC 6888 recommends a minimum of 1,000 ports per subscriber:

$$R_{sharing} = \frac{P_{total}}{P_{per\_sub}} = \frac{64,512}{1,000} \approx 64 \text{ subscribers per IP}$$

With logging requirements (RFC 7422), each connection generates a log entry:

$$L_{rate} = R_{sharing} \times \lambda_{per\_sub}$$

### Worked Examples

| Ports/Subscriber | Subscribers/IP | IPs for 100K Subscribers |
|:---:|:---:|:---:|
| 4,000 | 16 | 6,250 |
| 2,000 | 32 | 3,125 |
| 1,000 | 64 | 1,563 |
| 512 | 126 | 794 |
| 256 | 252 | 397 |

### Log Storage

Each NAT log entry (timestamp, internal IP:port, external IP:port, protocol) is ~100 bytes:

$$S_{logs} = R_{sharing} \times \lambda_{per\_sub} \times 100 \times T_{retention}$$

For 64 subs/IP, 100 connections/hour/sub, 6-month retention:

$$S = 64 \times 100 \times 100 \times (180 \times 24 \times 3600) = 64 \times 100 \times 100 \times 15,552,000 \approx 9.95 \text{ TB per IP}$$

This logging burden is a primary driver for IPv6 adoption.

---

## 5. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $C_{max} = P_{range} \times D$ | Multiplication | Port exhaustion |
| $N_{IPs} = \lceil \lambda \bar{d} / C_{max} \rceil$ | Ceiling division | IP planning |
| $N_{max} \times 320$ bytes | Linear scaling | Conntrack memory |
| $N_{max}/4$ buckets | Ratio | Hash table sizing |
| $1/\|P\|$ per guess | Uniform probability | Symmetric NAT traversal |
| $B \times 2$ per call | Doubling | TURN overhead |
| $P_{total}/P_{per\_sub}$ | Division | CGNAT sharing ratio |

## Prerequisites

- port numbering, connection tracking, hash tables, probability

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Conntrack lookup (5-tuple) | $O(1)$ hash | $O(N)$ entries |
| NAT translation (per packet) | $O(1)$ | $O(1)$ |
| Port allocation (new conn) | $O(1)$ bitmap | $O(P/8)$ |
| STUN binding request | $O(1)$ | $O(1)$ |
| Conntrack GC sweep | $O(N)$ | $O(1)$ |
| CGNAT log generation | $O(1)$ per conn | $O(N \times T)$ storage |

---

*NAT is a finite mapping from a large internal space to a small external space. The math is unforgiving: 65,535 ports per IP is a hard ceiling, conntrack tables eat 320 bytes per connection, and symmetric NAT breaks peer-to-peer by making port prediction a lottery with 64,512-to-1 odds.*
