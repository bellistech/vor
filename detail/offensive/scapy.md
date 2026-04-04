# The Mathematics of Scapy -- Protocol Theory and Network Layer Calculus

> *Every network packet is a finite sequence of bits whose meaning is determined by a recursive grammar of protocol headers, and Scapy exposes this grammar as a composable algebra of layer objects.*

---

## 1. Protocol Layering and Header Composition (Category Theory)

### The Problem

Network protocols are organized as a stack of layers, each encapsulating the next. Scapy models this as a composition of typed objects where each layer interprets a prefix of the remaining byte stream. Understanding the formal structure reveals why some layer combinations are valid and others produce malformed packets.

### The Formula

A packet $P$ is a composition of layers $L_1 / L_2 / \ldots / L_n$ where each layer is a function mapping a byte sequence to a structured interpretation:

$$L_i : \mathbb{B}^* \to (\text{fields}_i, \mathbb{B}^*)$$

The composition forms a chain:

$$P = L_1 \circ L_2 \circ \ldots \circ L_n$$

Each layer $L_i$ has a fixed or computed header length $h_i$ and consumes $h_i$ bytes, passing the remainder to $L_{i+1}$:

$$|P| = \sum_{i=1}^{n} h_i + |\text{payload}|$$

The total overhead ratio:

$$\text{overhead} = \frac{\sum_{i=1}^{n} h_i}{|P|}$$

For a typical HTTPS request: Ethernet (14) + IP (20) + TCP (20) + TLS (5+) = 59 bytes minimum overhead.

| Stack | Header Bytes | Typical Payload | Overhead |
|:---|:---:|:---:|:---:|
| Ether/IP/TCP | 54 | 1,460 (MSS) | 3.6% |
| Ether/IP/UDP | 42 | 1,472 | 2.8% |
| Ether/IP/TCP/TLS | 59+ | 1,400 | 4.0% |
| Ether/802.1Q/IP/TCP | 58 | 1,456 | 3.8% |

## 2. Checksum Algorithms and Error Detection (Coding Theory)

### The Problem

IP, TCP, and UDP headers include checksums to detect transmission errors. Scapy must compute valid checksums when constructing packets and verify them when dissecting. The one's complement checksum provides lightweight error detection but has known blind spots.

### The Formula

The Internet checksum (RFC 1071) for a byte sequence divided into 16-bit words $w_1, w_2, \ldots, w_n$:

$$C = \overline{\bigoplus_{i=1}^{n} w_i}$$

where $\oplus$ is one's complement addition (add with end-around carry) and $\overline{x}$ is bitwise complement.

One's complement addition:

$$a \oplus b = \begin{cases} a + b & \text{if } a + b < 2^{16} \\ a + b - 2^{16} + 1 & \text{if } a + b \geq 2^{16} \end{cases}$$

The TCP pseudo-header included in the checksum:

$$\text{pseudo} = [\text{src\_ip}, \text{dst\_ip}, 0, \text{protocol}, \text{tcp\_length}]$$

Detection capability: the one's complement sum detects:
- All single-bit errors (Hamming distance $d = 2$)
- All burst errors of length $\leq 16$ bits

Undetected error probability for random errors:

$$P(\text{undetected}) = 2^{-16} = 1.5 \times 10^{-5}$$

This is why TCP relies on CRC-32 in the Ethernet layer and application-layer integrity checks for security.

### Worked Example

Computing checksum for IP header bytes `45 00 00 3c 1c 46 40 00 40 06 00 00 ac 10 0a 63 ac 10 0a 0c`:

Words: `0x4500, 0x003c, 0x1c46, 0x4000, 0x4006, 0x0000, 0xac10, 0x0a63, 0xac10, 0x0a0c`

Sum: `0x4500 + 0x003c + 0x1c46 + 0x4000 + 0x4006 + 0x0000 + 0xac10 + 0x0a63 + 0xac10 + 0x0a0c = 0x2_96ce`

Fold carry: `0x96ce + 0x2 = 0x96d0`

Complement: `~0x96d0 = 0x692f`

Checksum field: `0x692f`.

## 3. ARP Spoofing and Cache Poisoning (Trust Model Theory)

### The Problem

ARP operates without authentication; any host can claim to own any IP address. ARP cache poisoning exploits this trust model to redirect traffic. The attack's success depends on timing, cache expiration, and the race condition between legitimate and spoofed replies.

### The Formula

An ARP cache entry maps $(IP, MAC)$ with a timeout $\tau$ (typically 60-300 seconds). A poisoning attack must send gratuitous ARP replies at rate $r_a$ faster than the victim refreshes:

$$r_a > \frac{1}{\tau}$$

The probability of successful poisoning with competing legitimate replies at rate $r_l$:

$$P(\text{poison}) = \frac{r_a}{r_a + r_l}$$

For continuous poisoning at rate $r_a$ = 0.5 replies/sec against legitimate refresh $r_l$ = 1/300 replies/sec:

$$P(\text{poison}) = \frac{0.5}{0.5 + 0.0033} = 0.9934$$

The window of vulnerability after cache expiration:

$$t_{\text{vulnerable}} = \tau - t_{\text{last\_legitimate}}$$

With static ARP entries or Dynamic ARP Inspection (DAI), the attack success drops to:

$$P(\text{poison with DAI}) \approx 0$$

### Worked Example

Network with 50 hosts, ARP cache timeout 120 seconds. Attacker sends spoofed ARP at 1 reply per 2 seconds.

- Target cache refreshes legitimately every 120 seconds
- Attacker sends 60 spoofed entries per cache lifetime
- Each spoofed reply has 60/61 = 98.4% chance of being the last one cached
- For MITM, both target and gateway must be poisoned
- Joint probability: $0.984^2 = 0.968$ (96.8% effective MITM)

## 4. Port Scanning Theory and Response Analysis (Decision Theory)

### The Problem

Port scanning determines which services are running on a remote host by sending probe packets and interpreting responses. Different scan types (SYN, FIN, NULL, XMAS) exploit different aspects of the TCP specification to infer port state while minimizing detection.

### The Formula

The TCP specification (RFC 793) defines response behavior that creates a decision function for each scan type:

$$\text{state}(p) = \begin{cases} \text{open} & \text{if response matches open criteria} \\ \text{closed} & \text{if response matches closed criteria} \\ \text{filtered} & \text{if no response (timeout)} \end{cases}$$

For SYN scan:

$$\text{state}_{\text{SYN}}(p) = \begin{cases} \text{open} & \text{if SYN-ACK received} \\ \text{closed} & \text{if RST received} \\ \text{filtered} & \text{if no response or ICMP unreachable} \end{cases}$$

For FIN/NULL/XMAS scans (exploit RFC 793 Section 3.9):

$$\text{state}_{\text{stealth}}(p) = \begin{cases} \text{open|filtered} & \text{if no response} \\ \text{closed} & \text{if RST received} \end{cases}$$

The scan completion time for $n$ ports with timeout $t$ and parallelism $k$:

$$T_{\text{scan}} = \frac{n}{k} \cdot \max(t_{\text{response}}, t)$$

For 65,535 ports with 1-second timeout and parallelism 100:

$$T_{\text{scan}} = \frac{65535}{100} \cdot 1 = 655 \text{ seconds} \approx 11 \text{ minutes}$$

| Scan Type | Packets Sent | Detectable | Open Signal | Closed Signal |
|:---|:---:|:---:|:---:|:---:|
| SYN | 1 per port | Moderate | SYN-ACK | RST |
| Connect | 3+ per port | High | Connection | RST |
| FIN | 1 per port | Low | Silence | RST |
| NULL | 1 per port | Low | Silence | RST |
| XMAS | 1 per port | Low | Silence | RST |
| UDP | 1 per port | Low | App response | ICMP unreach |

## 5. IP Fragmentation and Evasion (Reassembly Theory)

### The Problem

IP fragmentation splits large packets into smaller fragments that are reassembled at the destination. IDS/IPS systems must reassemble fragments before inspection, creating opportunities for evasion through overlapping fragments, tiny fragments, or out-of-order delivery.

### The Formula

An IP datagram of size $S$ with MTU $M$ produces $n$ fragments:

$$n = \left\lceil \frac{S - 20_{\text{IP header}}}{M - 20} \right\rceil$$

Each fragment carries:
- Fragment offset: $\text{offset}_i = i \cdot \lfloor(M - 20) / 8\rfloor \cdot 8$
- More Fragments flag: $MF = (i < n - 1)$
- Fragment size: $\min(M - 20, S - 20 - \text{offset}_i)$

Overlapping fragment evasion: when two fragments overlap at the same offset, the reassembly policy determines which data is used:

$$\text{policy} = \begin{cases} \text{first} & \text{BSD, Linux (use first fragment's data)} \\ \text{last} & \text{Windows (use last fragment's data)} \end{cases}$$

The attacker crafts fragments where the IDS sees one payload (benign) and the target sees another (malicious):

$$\text{IDS reassembly} \neq \text{target reassembly}$$

The number of possible reassembly orderings for $n$ fragments:

$$|\text{orderings}| = n!$$

For 10 fragments: $10! = 3{,}628{,}800$ possible arrival orders, making exhaustive IDS analysis impractical.

### Worked Example

1500-byte MTU, 4000-byte payload:

Fragment count: $\lceil 4000 / 1480 \rceil = 3$

- Fragment 1: offset=0, size=1480, MF=1
- Fragment 2: offset=1480, size=1480, MF=1
- Fragment 3: offset=2960, size=1040, MF=0

Evasion with overlap: insert a 4th fragment at offset=20 with 8 bytes of benign data. If the IDS uses first-wins policy and the target uses last-wins, the IDS sees benign data at bytes 20-27 while the target sees malicious data.

## 6. Traceroute Mechanics and Path Inference (Graph Reconstruction)

### The Problem

Traceroute discovers the network path by exploiting the IP Time-to-Live (TTL) field. Each router decrements TTL and sends an ICMP Time Exceeded when TTL reaches zero. The sequence of responding routers reveals the forwarding path, but load balancing, asymmetric routing, and rate limiting complicate interpretation.

### The Formula

A traceroute with maximum TTL $T$ sends probes and observes:

$$\text{path} = [r_1, r_2, \ldots, r_k] \quad \text{where } k \leq T$$

Router $r_i$ responds to probe with $TTL = i$. The round-trip time to hop $i$:

$$RTT_i = t_{\text{response}_i} - t_{\text{probe}_i}$$

With ECMP (Equal-Cost Multi-Path) load balancing, the path is not unique. Paris traceroute fixes the flow identifier to ensure consistent routing:

$$\text{flow\_id} = \text{hash}(\text{src\_ip}, \text{dst\_ip}, \text{src\_port}, \text{dst\_port}, \text{protocol})$$

The number of potential paths through a network with $b$ load-balanced hops, each with $w$ equal-cost next-hops:

$$|\text{paths}| = w^b$$

For 3 load-balanced routers with 4 paths each: $4^3 = 64$ potential paths. Standard traceroute may show a different path per probe, creating "diamond" artifacts in the output.

---

*Scapy's power lies in its refusal to abstract away the byte-level reality of network protocols. By exposing every header field as a mutable Python attribute, it transforms the network stack from an opaque system call interface into a transparent algebraic structure where packets are first-class objects that can be composed, decomposed, mutated, and analyzed with the full expressiveness of a programming language.*

## Prerequisites

- Network protocols (TCP/IP stack, Ethernet, ARP, DNS, ICMP)
- Coding theory (checksums, CRC, error detection bounds)
- Graph theory (network topology, path inference, routing)
- Probability theory (detection rates, race conditions, timing)
- Formal language theory (protocol grammars, BPF filter syntax)
- Number theory (one's complement arithmetic, modular operations)

## Complexity

- **Beginner:** Crafting and sending ICMP/TCP packets, reading pcap files, basic ARP scanning, DNS queries
- **Intermediate:** TCP SYN scanning, ARP cache poisoning, DNS spoofing, custom protocol layers, traceroute implementation
- **Advanced:** IDS evasion via fragmentation, protocol fuzzing with Scapy, covert channel construction, stateful protocol interaction, timing-based fingerprinting
