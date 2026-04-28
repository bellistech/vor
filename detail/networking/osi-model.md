# OSI Model — Deep Dive

> *Seven layers of abstraction, formalized in 1984 as ISO/IEC 7498-1, that turn "send a packet" into a stack of mathematically tractable boxes — each with its own framing, addressing, error model, and failure modes. The OSI model is not a protocol, it is a vocabulary; the math underneath each layer is what makes it operationally real.*

---

## OSI 7-layer model — origins and ITU-T X.200

### The standardization timeline

The Open Systems Interconnection model was developed jointly by the **International Organization for Standardization (ISO)** and the **International Telegraph and Telephone Consultative Committee (CCITT)**, which is now the **ITU Telecommunication Standardization Sector (ITU-T)**. The work was driven by the recognition in the late 1970s that proprietary networking stacks (IBM SNA, DECnet, Xerox XNS, Burroughs BNA) were balkanizing computer-to-computer communication.

| Year | Event |
|:---|:---|
| 1977 | ISO/TC 97/SC 16 formed; OSI work begins |
| 1978 | First draft reference model circulated |
| 1980 | Draft International Standard (DIS) 7498 |
| 1983 | ITU-T (then CCITT) approves Recommendation X.200 |
| 1984 | ISO publishes IS 7498:1984 — the canonical OSI reference model |
| 1989 | ISO 7498-1:1989 (first technical corrigendum) |
| 1994 | ISO/IEC 7498-1:1994 — current edition, Edition 2 |

The 1994 edition is the one cited operationally. It is a free download from ISO and ITU-T; the document itself is dense and precise but, importantly, **does not specify any wire format** — it specifies **abstract services and abstract protocols** that any concrete protocol can claim to implement.

### What was actually standardized

ISO/IEC 7498-1 standardizes:

1. **A 7-layer architectural model** — Physical, Data Link, Network, Transport, Session, Presentation, Application.
2. **Service Access Points (SAPs)** — abstract interfaces between adjacent layers.
3. **Protocol Data Units (PDUs)** — abstract message types per layer, with naming conventions:
   - Layer 1: bit / symbol
   - Layer 2: frame
   - Layer 3: packet (also "datagram" in some texts)
   - Layer 4: segment (TCP) / datagram (UDP)
   - Layer 5-7: message / data
4. **Service primitives** — `request`, `indication`, `response`, `confirm`.
5. **The encapsulation rule** — each layer prepends (and sometimes appends) a header to data passed down from above.

It does **not** standardize:
- Specific protocols (those are in companion ISO documents — e.g. ISO/IEC 8073 for OSI TP4, ISO/IEC 8473 for CLNP).
- Exact wire formats.
- Hardware specifics.

### OSI vs TCP/IP 4-layer model

The TCP/IP model (RFC 1122 host requirements) collapses OSI's seven layers into four. The mapping is well-defined for the lower layers and notoriously fuzzy for the upper three.

```
+-------------------+  +---------------------------------+
|  OSI 7 LAYER      |  |  TCP/IP 4 LAYER (RFC 1122)      |
+-------------------+  +---------------------------------+
|  7  Application   |  |                                 |
|  6  Presentation  |  |   4   Application               |
|  5  Session       |  |                                 |
+-------------------+  +---------------------------------+
|  4  Transport     |  |   3   Transport                 |
+-------------------+  +---------------------------------+
|  3  Network       |  |   2   Internet                  |
+-------------------+  +---------------------------------+
|  2  Data Link     |  |                                 |
|  1  Physical      |  |   1   Link / Network Access     |
+-------------------+  +---------------------------------+
```

Some texts use a 5-layer "hybrid" model (Tanenbaum, Kurose & Ross) that splits Link from Physical but keeps Application as one layer. RFC 1122 explicitly merges the upper three; RFC 1122 §1.1.3 notes the "session and presentation layers are not used in the Internet protocol suite."

| OSI Layer | OSI PDU | TCP/IP layer | Example concrete protocols |
|:---:|:---|:---|:---|
| 7 | APDU | Application | HTTP, SMTP, DNS, FTP, SSH, gRPC |
| 6 | PPDU | Application | TLS records, ASN.1 BER/DER, JSON, MIME |
| 5 | SPDU | Application | RPC session, NetBIOS, SOCKS, gQUIC streams |
| 4 | TPDU / segment | Transport | TCP, UDP, SCTP, QUIC, DCCP |
| 3 | Packet | Internet | IPv4, IPv6, ICMP, ICMPv6, IGMP, IPsec |
| 2 | Frame | Link | Ethernet, PPP, HDLC, 802.11, ARP |
| 1 | Bit / symbol | Link | 10BASE-T, 1000BASE-LX, DOCSIS, DSL |

---

## Layer 1 (Physical)

Layer 1 is where bits become physics. Voltage levels, light pulses, RF modulation, magnetic flux. Everything above Layer 1 assumes the physical link can carry symbols at some rate with some error probability.

### Encoding math — line codes

A **line code** maps logical bits to physical symbols. The fundamental design pressure: symbol streams must carry timing information (DC balance, transition density) so the receiver's clock can phase-lock to the sender's.

| Code | Bit/symbol ratio | Notes |
|:---|:---:|:---|
| NRZ (Non-Return-to-Zero) | 1:1 | Simplest. Long runs of 0 or 1 lose clock recovery. |
| NRZI (NRZ-Inverted) | 1:1 | Transition = 1, no transition = 0. Improves over NRZ. |
| Manchester (IEEE 802.3 10BASE-T) | 1:2 | Each bit is a transition; doubles bandwidth need. |
| Differential Manchester (Token Ring) | 1:2 | Like Manchester but only checks transition presence. |
| 4B/5B (FDDI, 100BASE-TX) | 4:5 | Maps every 4 bits to 5-bit symbol with ≥2 transitions. 80% efficient. |
| 8B/10B (1000BASE-X, PCIe Gen1/2) | 8:10 | DC balanced, error detection, comma symbols. 80% efficient. |
| 64B/66B (10GBASE-R, 25GBASE-R, PCIe Gen3+) | 64:66 | Scrambled framing. ~97% efficient. |
| PAM-4 (200/400G Ethernet) | 2 bits/symbol | 4 voltage levels, doubles bit rate at same baud. |

### Baud vs bit-rate

- **Baud (Bd)** = symbols per second.
- **Bit rate (bps)** = bits per second.
- Relationship: `bit_rate = baud × log2(M)` where `M` is the number of symbol levels.

Worked example — 100BASE-TX uses MLT-3 (3 levels) and 4B/5B encoding at 125 MBd:

```
useful_bits = 125 MBd × (4/5 from 4B5B) × log2(2 bits per MLT-3 transition?) 
            = 125 × 0.8 × 1 = 100 Mbps
```

Worked example — 10GBASE-R uses 64B/66B at 10.3125 GBd NRZ:

```
useful_bits = 10.3125 GBd × (64/66) = 10.0 Gbps
```

Worked example — 100GBASE-SR4 uses 4 fiber lanes, NRZ, 25.78125 GBd, 64B/66B:

```
per_lane = 25.78125 × 64/66 = 25 Gbps
total   = 4 × 25 = 100 Gbps
```

Worked example — 400GBASE-DR4 uses 4 lanes, PAM-4 (2 bits/symbol), 26.5625 GBd, 256B/257B:

```
per_lane = 26.5625 × 2 × (256/257) = ~52.94 Gbps  (advertised 53.125 Gbps with FEC overhead)
total   = 4 × 53.125 ≈ 212.5 Gbps signaling
```

### Attenuation and loss budget

Optical loss is logarithmic in dB:

```
loss_dB = 10 × log10(P_in / P_out)
P_out  = P_in × 10^(-loss_dB / 10)
```

A typical 1310 nm singlemode fiber loses **~0.35 dB/km**. A 1550 nm link loses **~0.22 dB/km**. SFP optic budgets quote **link budget** in dB:

```
budget_dB = launch_power_min - receiver_sensitivity
```

Example — 1000BASE-LX SFP:
- Launch min: -9.5 dBm
- Rx sensitivity: -19 dBm
- Budget: 9.5 dB
- Distance at 0.35 dB/km: 9.5 / 0.35 ≈ 27 km (matches 10 km spec with margin).

Copper attenuation is frequency-dependent (skin effect, ~√f). Cat 6A has ~32 dB at 500 MHz over 100 m. Once you exceed the budget, the receiver eye closes and bit error rate (BER) rockets.

### Shannon-Hartley capacity theorem

The fundamental upper bound on any analog channel:

```
C = B × log2(1 + SNR)
```

Where:
- `C` = channel capacity in bits per second
- `B` = bandwidth in Hz
- `SNR` = signal-to-noise ratio (linear, not dB)

To convert dB to linear:

```
SNR_linear = 10^(SNR_dB / 10)
```

Worked examples:

| Channel | B | SNR (dB) | SNR (linear) | C (Mbps) |
|:---|:---:|:---:|:---:|:---:|
| Voice POTS | 3.4 kHz | 36 | 4000 | 0.041 |
| ADSL2+ downstream | 2.2 MHz | 40 | 10000 | 29.2 |
| WiFi 5 (80 MHz) | 80 MHz | 40 | 10000 | 1063 |
| WiFi 6 (160 MHz, MIMO 8x8) | 160 MHz × 8 | 35 | 3162 | 14964 |
| 4G LTE (20 MHz, MIMO 4x4) | 80 MHz | 25 | 316 | 663 |

Real radios never reach Shannon — coding overhead (LDPC, polar codes), guard intervals, pilot subcarriers, and protocol headers all subtract. **80–90% of Shannon** is considered excellent.

### Bit error rate (BER) and frame loss

BER is the probability a single bit flips on the wire. Modern fiber: BER ≈ 10⁻¹². Copper Ethernet: 10⁻¹⁰ target. WiFi: 10⁻⁵ to 10⁻⁷ depending on conditions.

Probability of a frame of `N` bits being error-free:

```
P_clean = (1 - BER)^N ≈ 1 - N × BER   (for small BER × N)
```

Frame loss probability:

```
P_loss = 1 - P_clean ≈ N × BER
```

For a 12,000-bit Ethernet frame at BER = 10⁻¹²:

```
P_loss ≈ 12000 × 10^-12 = 1.2 × 10^-8
```

That is roughly 1 lost frame per 10⁸ frames. At 1.4 Mfps line rate (10 GbE), ~1 corrupted frame per 70 seconds — caught by FCS and retransmitted by Layer 4.

---

## Layer 2 (Data Link)

Layer 2 turns a bit pipe into a frame pipe. Frames are delimited, addressed, error-checked, and (sometimes) flow-controlled.

### Framing protocols

| Protocol | Frame format | Notes |
|:---|:---|:---|
| Ethernet II (DIX) | DA(6)+SA(6)+EtherType(2)+payload+FCS(4) | 64–1518 octet frame; jumbo to 9216. |
| IEEE 802.3 | DA(6)+SA(6)+Length(2)+LLC+payload+FCS(4) | EtherType ≥ 0x0600 distinguishes from 802.3. |
| HDLC | Flag(0x7E) + Address + Control + Info + FCS(2-4) + Flag | ISO/IEC 13239. Bit-stuffing for transparency. |
| PPP (RFC 1661) | Flag + Addr(0xFF) + Ctrl(0x03) + Protocol(2) + Info + FCS + Flag | HDLC-derived. Used for serial, PPPoE, dial-up. |
| Frame Relay | Flag + DLCI(2) + Info + FCS + Flag | DLCI = virtual circuit ID. |
| 802.11 WiFi | FC(2)+Dur(2)+Addr1-4(6 each)+Seq(2)+Body+FCS(4) | Up to 4 addresses (WDS), 802.11n adds HT control. |
| MPLS (RFC 3032) | 4-byte label stack between Layer 2 and Layer 3 | Label(20)+TC(3)+S(1)+TTL(8). |

Bit-stuffing (HDLC, USB) inserts a 0 after every 5 consecutive 1s in payload to avoid colliding with the 0x7E flag (`01111110`). Byte-stuffing (PPP async, SLIP) escapes the flag byte with an escape byte. Block coding (8B/10B, 64B/66B) handles framing at Layer 1 and obviates stuffing at Layer 2.

### CRC math (frame check sequence)

Ethernet uses **CRC-32 (IEEE 802.3)**, polynomial `0x04C11DB7` (reverse representation `0xEDB88320`). The CRC treats the frame as a polynomial over GF(2) and computes the remainder when divided by the generator polynomial. The 32-bit FCS is appended to the frame.

```
FCS = bit-string ÷ G(x)   (polynomial division mod 2)
```

Properties of CRC-32:
- Detects all 1, 2, and 3-bit errors.
- Detects all burst errors of length ≤ 32 bits.
- Detects all error patterns with an odd number of bits flipped.
- Probability of undetected random error: ≈ 2⁻³² ≈ 2.3 × 10⁻¹⁰.

For shorter frames or different protocols:

| CRC | Polynomial (Koopman notation) | Used by |
|:---|:---|:---|
| CRC-8-CCITT | 0xE7 (truncated 0x07) | ATM HEC, 1-Wire |
| CRC-16-CCITT | 0x8810 (truncated 0x1021) | HDLC, X.25, Bluetooth |
| CRC-16-IBM | 0xA001 | Modbus, USB |
| CRC-32 (Ethernet) | 0xEDB88320 (reversed) | Ethernet, PNG, ZIP, gzip |
| CRC-32C (Castagnoli) | 0x82F63B78 (reversed) | iSCSI, SCTP, EXT4 metadata, Btrfs |

### MAC addressing — 48-bit address space

The IEEE 802.3 MAC address is 48 bits, written as 6 hex octets separated by colons: `aa:bb:cc:dd:ee:ff`. The address space is `2^48 = 281,474,976,710,656` (~281 trillion).

The first 24 bits are the **OUI (Organizationally Unique Identifier)**, assigned by IEEE Registration Authority. The remaining 24 bits are vendor-assigned. The first byte's two least-significant bits encode flags:

```
| octet 0 |
+---------+
| 7-2  X  |   bits 7-2: rest of OUI
| 1   I/G |   1 = group/multicast, 0 = unicast
| 0   U/L |   1 = locally administered, 0 = globally unique
+---------+
```

Special MAC addresses:

| Address | Meaning |
|:---|:---|
| `ff:ff:ff:ff:ff:ff` | Broadcast |
| `01:00:5e:xx:xx:xx` | IPv4 multicast (low 23 bits map to IPv4 group) |
| `33:33:xx:xx:xx:xx` | IPv6 multicast (low 32 bits map to IPv6 group) |
| `01:80:c2:00:00:00` | STP BPDU |
| `01:80:c2:00:00:0e` | LLDP |
| `01:1b:19:00:00:00` | PTP |
| `02:xx:xx:xx:xx:xx` | Locally administered (U/L bit set) |

Birthday-paradox collision space: $\sqrt{2^{48}} \approx 16.7M$. With 16 million randomly-chosen MACs, you have a ~50% chance of one collision — which is why IEEE assigns OUIs hierarchically.

### Flow control

**XON/XOFF** (Layer 2 software flow control, RS-232) uses two ASCII codes: `0x11 (DC1, XON)` and `0x13 (DC3, XOFF)`. Receiver sends XOFF when its buffer fills, XON when drained. Brittle and obsolete except in legacy serial/console links.

**IEEE 802.3x PAUSE frames** are a hardware Layer 2 mechanism. A receiver sends a PAUSE frame to the special multicast `01:80:c2:00:00:01` with EtherType `0x8808`:

```
| Dst MAC (6) = 01:80:c2:00:00:01 |
| Src MAC (6)                     |
| EtherType (2) = 0x8808          |
| Opcode (2)    = 0x0001 (PAUSE)  |
| Quanta (2)    = N (pause time)  |
| Pad (42)                        |
| FCS (4)                         |
```

Pause time `N` is in units of 512 bit-times. At 1 Gbps, one quantum = 512 ns. At 10 Gbps, 51.2 ns. `N = 65535` → max pause ~33 ms at 1G, ~3.3 ms at 10G.

**Priority-based Flow Control (802.1Qbb)** extends PAUSE to per-traffic-class granularity, used in DCB (Data Center Bridging) for FCoE and lossless RoCE.

**Credit-based flow control** (Fibre Channel, PCIe) — receiver advertises buffer credits; sender consumes one credit per frame and stalls when credits hit zero. Loss-free by construction.

### Error detection vs error correction

Layer 2 typically only **detects** errors via FCS and either drops or retransmits the frame. Forward Error Correction (FEC) is selectively used at Layer 1/2 boundary on lossy media:

| FEC code | Overhead | Used by |
|:---|:---:|:---|
| Reed-Solomon RS(528, 514) (KR-FEC) | 2.7% | 100GBASE-CR4, 25GBASE-CR |
| Reed-Solomon RS(544, 514) (RS-FEC) | 5.8% | 100GBASE-KR4, 25GBASE-LR |
| LDPC | varies | DOCSIS 3.1, 5G NR, WiFi 6 |
| Polar codes | varies | 5G NR control channel |

FEC trades bandwidth for noise tolerance — a useful exchange when retransmission is expensive (satellite, DOCSIS, wireless).

---

## Layer 3 (Network)

Layer 3 introduces **end-to-end addressing** independent of the underlying physical topology, plus the routing math to decide where to send a packet.

### IP forwarding decision

For each ingress packet:
1. Decrement TTL (IPv4) / Hop Limit (IPv6). If 0 → drop, send ICMP Time Exceeded.
2. Lookup destination address in **FIB (Forwarding Information Base)** — longest-prefix match.
3. If no match → drop, send ICMP Destination Unreachable.
4. Check MTU against egress interface; fragment (IPv4) or send ICMP Packet-Too-Big (IPv6) if needed.
5. Recompute IPv4 header checksum (IPv6 has none).
6. Encapsulate in Layer 2 frame for next-hop and transmit.

### Longest-prefix match (LPM) complexity

A routing table entry is a `(prefix, mask, next-hop)` tuple. LPM finds the entry with the longest matching prefix. Naive linear scan is `O(N)`. Production routers use specialized data structures:

| Algorithm | Lookup | Update | Memory | Used in |
|:---|:---:|:---:|:---:|:---|
| Linear scan | O(N) | O(1) | O(N) | Toy / education |
| Binary trie (uncompressed) | O(W) | O(W) | O(NW) | Reference |
| Patricia trie (RFC 1812 nominal) | O(W) | O(W) | O(N) | Linux FIB up to 2.6 |
| LC-trie | O(log W) | O(W) | O(N) | Linux 2.6+ "trie" |
| DXR / SAIL | O(1) | O(N) | O(N) | Software fast path |
| TCAM (ternary CAM) | O(1) hardware | hardware | hardware | Cisco/Juniper hardware |
| Disjoint prefixes (Eatherton/Dittia) | O(1) | O(W) | O(N) | High-end ASIC |

`W` is the address width (32 for IPv4, 128 for IPv6), `N` is the number of prefixes. As of 2026 the IPv4 default-free zone has ~970,000 prefixes and IPv6 has ~210,000. Each is comfortably handled by modern silicon TCAMs (1M-entry IPv4, 512K IPv6 typical for merchant silicon like Tomahawk 5).

### Fragmentation

#### IPv4 fragmentation (RFC 791)

Routers may fragment a packet that exceeds an outgoing link's MTU when DF (Don't Fragment) is clear. Fields involved:

```
| Identification (16) | Flags (3): R DF MF | Fragment Offset (13) |
```

`Fragment Offset` is in **8-byte units**. Maximum offset = `8 × (2^13 - 1) = 65528` bytes, matching the 16-bit Total Length minus header.

Fragmenting a 4000-byte packet over a 1500-byte MTU link:

```
Original: 20 (IP header) + 3980 (data)
Frag 1:   20 (IP) + 1480 (data) = 1500B   offset=0,    MF=1
Frag 2:   20 (IP) + 1480 (data) = 1500B   offset=185,  MF=1   (185 × 8 = 1480)
Frag 3:   20 (IP) + 1020 (data) = 1040B   offset=370,  MF=0   (370 × 8 = 2960)
```

Reassembly is end-to-end; routers do not reassemble. RFC 5722 deprecates overlapping fragments because of the **teardrop attack**.

#### IPv6 fragmentation (RFC 8200)

Routers do **not** fragment IPv6 packets. Source must perform Path MTU Discovery (RFC 8201) and either segment the payload at Layer 4 or insert a Fragment Extension Header. The minimum IPv6 MTU is **1280 bytes** (originally 1500-IPSec/IPv6 difference; codified to give end systems a guaranteed path size).

### ICMP roles

ICMP is the diagnostic and signalling cousin of IP. Key types per RFC 792 (ICMPv4) / RFC 4443 (ICMPv6):

| Type (v4 / v6) | Name | Use |
|:---|:---|:---|
| 0 / 129 | Echo Reply | `ping` reply |
| 8 / 128 | Echo Request | `ping` |
| 3 / 1 | Destination Unreachable | "no route", "host unreachable" |
| 11 / 3 | Time Exceeded | `traceroute` |
| 5 / -- | Redirect | "use this gateway instead" (deprecated in practice) |
| -- / 2 | Packet Too Big | PMTUD signal in IPv6 |
| 13/14 / -- | Timestamp Req/Rep | Time sync (deprecated) |
| -- / 133-137 | NDP | Neighbor Discovery (replaces ARP, ICMP redirect) |

Filtering ICMP unconditionally breaks PMTUD and is a documented anti-pattern (RFC 4890 specifically lists which messages must not be dropped).

### RIB vs FIB

- **RIB (Routing Information Base)** — control-plane database. All learned routes from all protocols (BGP, OSPF, IS-IS, static, connected, RIP). May contain multiple paths per prefix. Lives in CPU memory.
- **FIB (Forwarding Information Base)** — data-plane structure. The **best** route per prefix, optimized for fast LPM. Lives in linecard hardware (TCAM) or fast-path memory.
- **RIB → FIB programming** — RIB selects best path per protocol-priority and AD (administrative distance), then installs into FIB. RIB is consistent across the box; FIB may be replicated per linecard.

```
+-------------+    +-------------+    +-------------+
| BGP / OSPF  |--->|     RIB     |--->|     FIB     |
| static etc. |    | (best path  |    | (LPM-       |
+-------------+    |  selection) |    |  optimized) |
                   +-------------+    +-------------+
                          |                  |
                       CPU mem            TCAM/silicon
```

A modern Cisco/Juniper edge router will hold ~1.2M IPv4 + 200K IPv6 RIB entries from full BGP tables, programming a similar number into FIB.

---

## Layer 4 (Transport)

Layer 4 turns the host-to-host packet stream into per-application byte streams (TCP) or message streams (UDP). The fundamental abstractions are **multiplexing via ports** and **end-to-end reliability** (TCP only).

### Multiplexing via ports

TCP and UDP both have 16-bit source and destination port fields:

```
| Source Port (16) | Dest Port (16) | ... |
```

Address space: `2^16 = 65536`, with port 0 reserved → 65535 usable ports per (host, transport) pair.

Port number ranges (IANA):

| Range | IANA Name | Use |
|:---|:---|:---|
| 0 | reserved | Cannot be used. |
| 1-1023 | Well-known / system | HTTP=80, SSH=22, DNS=53, etc. (root-only on Unix.) |
| 1024-49151 | Registered / user | App-specific (PostgreSQL=5432). |
| 49152-65535 | Dynamic / ephemeral | Transient client-side source ports. |

Linux ephemeral range default: `32768-60999` (`/proc/sys/net/ipv4/ip_local_port_range`). MacOS/BSD: `49152-65535`. Windows: `49152-65535` since Vista.

Connection identity tuple — the **5-tuple** uniquely identifies a flow:

```
(protocol, src IP, src port, dst IP, dst port)
```

A single server port can host millions of concurrent connections — what differs across them is the client side of the tuple. The C10K and C10M limits live here, gated by per-process FD limits and kernel socket lookup hashing.

### TCP vs UDP semantics

| Property | TCP | UDP |
|:---|:---|:---|
| Header size | 20 bytes (no options) | 8 bytes |
| Reliable delivery | Yes (ACK + retransmit) | No |
| Ordered delivery | Yes (sequence numbers) | No |
| Connection-oriented | Yes (3WHS, FIN/RST) | No |
| Flow control | Yes (window) | No |
| Congestion control | Yes (Reno/Cubic/BBR) | No (app-level if needed) |
| Multiplexing | 5-tuple | 5-tuple |
| Header overhead per byte (1 KB payload) | 1.95% | 0.78% |
| Useful when | Bulk transfer, web, email | Voice, video, DNS, VPN data plane |

TCP segment header (20 bytes mandatory + options up to 60):

```
| 0          | 1          | 2          | 3          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| Src Port (16)              | Dst Port (16)      |
| Sequence Number (32)                            |
| Acknowledgement Number (32)                     |
| DataOff(4)|Rsv(3)|NS|CWR|ECE|URG|ACK|PSH|RST|SYN|FIN| Window (16) |
| Checksum (16)              | Urgent Pointer (16)|
| Options (variable, up to 40 bytes)              |
| Data ...                                        |
```

UDP header (RFC 768):

```
| Src Port (16) | Dst Port (16) | Length (16) | Checksum (16) |
| Data ...                                                    |
```

UDP checksum is optional in IPv4, mandatory in IPv6 (RFC 6935 allows zero-checksum for tunneling).

### TCP connection state machine

```
                  +----------+
                  |  CLOSED  |
                  +----+-----+
                       |
           passive open|     |active open
                       v     v
                +-----------+      +-----------+
                |  LISTEN   |<---->|   SYN-SENT|
                +-----+-----+      +-----+-----+
              SYN/SYN+ACK|              |SYN+ACK/ACK
                       v|              v
                  +-----------+    +-----------+
                  |SYN-RECVD  |    |ESTABLISHED|
                  +-----+-----+    +-----+-----+
                        |                |
                        +----------------+
                                |
             +------------------+--------------------+
             |          (close path)                |
        FIN/ACK|                              |ACK
             v                                v
       +-----------+                    +-----------+
       |FIN-WAIT-1 |                    |CLOSE-WAIT |
       +-----+-----+                    +-----+-----+
       ACK|         |FIN/ACK                |FIN/ACK
          v         v                       v
     +---------+  +---------+         +-----------+
     |FIN-WAIT2|  |CLOSING  |         |LAST-ACK   |
     +----+----+  +----+----+         +-----+-----+
          |          |ACK                   |ACK
          |FIN/ACK   v                      v
          +------+TIME-WAIT+----------+CLOSED
                 (2MSL hold-down)
```

`TIME_WAIT` lasts `2 × MSL` (Maximum Segment Lifetime). Per RFC 793 MSL = 2 minutes → `TIME_WAIT = 240 s`. Linux default is 60 s; tuned via `tcp_fin_timeout`.

### Sliding window math

The sender's send window:

```
SND.UNA <= seq < SND.UNA + SND.WND
```

Effective throughput limit:

```
throughput <= min(cwnd, rwnd) / RTT
```

Where `cwnd` = congestion window (sender-side), `rwnd` = receiver advertised window.

Bandwidth-delay product (BDP):

```
BDP = bandwidth × RTT   [bits or bytes]
```

For full link utilization, need `min(cwnd, rwnd) >= BDP`. With 16-bit window field max 65,535 bytes, RFC 7323 Window Scale (shift count 0-14) gives effective 1 GB.

For deeper TCP math (sequence wrap, AIMD, BBR), see `detail/networking/tcp.md`.

---

## Layer 5 (Session)

Layer 5 manages long-lived **dialogs** between communicating peers — establishing them, suspending and restarting them, marking checkpoints, and tearing them down.

### Session establishment, dialog control

Pure-OSI session services (ISO 8326) include:

| Service | Description |
|:---|:---|
| Session connection establishment | Negotiate session parameters (token, sync, activity). |
| Token management | "Major synchronization" token for half-duplex turn-taking. |
| Activity management | Bracket related work into atomic activities. |
| Synchronization (minor / major) | Mark recovery points within a dialog. |
| Resynchronization | Restart from a known sync point after failure. |
| Exception reporting | Out-of-band error signalling within the session. |

In practice these primitives almost never appear on the modern internet — the closest analogues are TLS sessions (Layer 6/7), gRPC bidirectional streams, MQTT persistent sessions, and SSH multiplexing.

### Checkpoint / restart

The OSI ideal: every long file transfer marks **synchronization points** so that a failure restarts from the last checkpoint, not from byte zero. Implementations:

- **HTTP Range requests (RFC 7233)** — `Range: bytes=N-` resumes a download from offset N. The client tracks the checkpoint, not the protocol.
- **rsync block algorithm** — file checkpoints are implicit per-block.
- **ZeroMQ / NATS JetStream** — message acknowledgement implies durable checkpointing.
- **TCP itself does not checkpoint** — a torn-down TCP connection loses unacked data. Layer 7 must reconstruct.

### Why Layer 5 was absorbed into Layer 7

Three forces collapsed Layer 5 in TCP/IP:

1. **TCP already provides a connection** — the `(5-tuple, ESTABLISHED)` fact is a session in everything but name. Adding a separate session layer felt redundant for early Internet apps.
2. **Stateless application protocols won** — HTTP/0.9 was deliberately stateless. Where state was needed (cookies, JWT, OAuth tokens), it was layered on top of HTTP.
3. **Session primitives are application-specific** — what "checkpoint" means for FTP differs from what it means for X.400 mail or SQL transactions. A generic layer added complexity without clear benefit.

Today's de facto Layer 5 stack:
- **TLS sessions and session resumption (RFC 8446 §2.2 PSK)** — abbreviated handshake from session ticket.
- **HTTP/2 streams** — multiplexed concurrent requests within one TCP connection.
- **HTTP/3 / QUIC streams** — same idea over UDP, connection migration on IP change.
- **gRPC bidirectional streaming** — long-lived RPC as a session.
- **WebSocket** — upgrade HTTP to a persistent bidirectional channel.
- **SSH channels** — multiplexed shells/file transfers within one SSH session.
- **MQTT clean-session = false** — broker remembers subscriptions across reconnects.

---

## Layer 6 (Presentation)

Layer 6 is the encoding/encryption boundary. Anything that converts **abstract data → bytes** lives here: character sets, structured-data encodings, compression, and encryption primitives.

### ASCII / Unicode / UTF-8

| Encoding | Code points | Bytes per character | Note |
|:---|:---:|:---:|:---|
| ASCII (ANSI X3.4) | 0-127 | 1 | 7-bit; foundational. |
| Latin-1 (ISO 8859-1) | 0-255 | 1 | Western Europe. |
| Unicode (ISO/IEC 10646) | 0-10FFFF (1.1M) | abstract | Just code points. |
| UTF-8 | 0-10FFFF | 1-4 | ASCII-compatible, byte-stream-safe. |
| UTF-16 | 0-10FFFF | 2 or 4 | BMP=2, supplementary=surrogate pair. |
| UTF-32 / UCS-4 | 0-10FFFF | 4 | Fixed width; rarely on the wire. |

UTF-8 byte length per code point:

```
0x00000-0x0007F  -> 1 byte    (0xxxxxxx)
0x00080-0x007FF  -> 2 bytes   (110xxxxx 10xxxxxx)
0x00800-0x0FFFF  -> 3 bytes   (1110xxxx 10xxxxxx 10xxxxxx)
0x10000-0x10FFFF -> 4 bytes   (11110xxx 10xxxxxx 10xxxxxx 10xxxxxx)
```

A 1024-character ASCII string is 1024 bytes UTF-8. The same string in CJK ideographs is ~3072 bytes UTF-8 vs 2048 bytes UTF-16. Encoding choice affects bandwidth.

### ASN.1 BER / DER / PER

**ASN.1 (Abstract Syntax Notation One)** is ISO/IEC 8824 — the canonical Layer 6 schema language. It defines abstract types; encoding rules turn them into bytes.

| Encoding rule | Property | Used by |
|:---|:---|:---|
| BER (Basic Encoding Rules) | Tag-Length-Value, multiple valid encodings per value | LDAP, SNMP |
| DER (Distinguished Encoding Rules) | BER subset, unique encoding per value | X.509 certs, CMS, PKCS |
| CER (Canonical Encoding Rules) | DER cousin; rarely used | (legacy) |
| PER (Packed Encoding Rules) | Bit-packed, very compact | UMTS/LTE/5G NAS, S1AP |
| OER (Octet Encoding Rules) | Byte-aligned, faster | Aviation (ATN, ADS-B) |
| XER (XML Encoding Rules) | XML representation | Niche |
| JER (JSON Encoding Rules) | JSON representation | Modern integrations |

A simple TLV in DER:

```
INTEGER 1234

bytes: 02 02 04 D2
        ^  ^  ^^^^^
        |  |  value 1234 = 0x04D2
        |  length = 2 octets
        tag = INTEGER (2)
```

### Modern Layer 6 equivalents

| Format | Schema language | Wire shape | Notes |
|:---|:---|:---|:---|
| JSON (RFC 8259) | JSON Schema, OpenAPI | UTF-8 text | Human-readable, self-delimiting. |
| MessagePack | none / .msgpack | Binary tag-length-value | JSON-compatible, smaller. |
| CBOR (RFC 8949) | CDDL | Binary major-type/value | IoT / COSE / CWT. |
| Protocol Buffers v3 | `.proto` | Binary tag-wire-type | gRPC payloads, Bigtable. |
| Cap'n Proto | `.capnp` | Wire-aligned, zero-copy | Sandstorm, ML pipelines. |
| FlatBuffers | `.fbs` | Wire-aligned, zero-copy | Game telemetry, Android IPC. |
| Avro | `.avsc` | Binary, schema-shipped | Kafka, Hadoop. |
| Apache Thrift | `.thrift` | Binary or compact | Facebook/Twitter legacy. |

Protocol Buffers v3 wire format is varint-tagged:

```
| field_tag (varint) | wire_type (3 bits) | value (varint or length-delimited) |
```

Wire types:

| Type | Meaning |
|:---:|:---|
| 0 | Varint (int32, int64, uint32, uint64, bool, enum) |
| 1 | 64-bit fixed (fixed64, sfixed64, double) |
| 2 | Length-delimited (string, bytes, embedded, packed) |
| 5 | 32-bit fixed (fixed32, sfixed32, float) |

### TLS 1.3 record protocol (RFC 8446)

TLS lives at Layer 6 — it converts plaintext application bytes to authenticated, encrypted record bytes.

```
struct {
    ContentType opaque_type;        // 1 byte; always 0x17 (application_data) post-handshake
    ProtocolVersion legacy_version; // 2 bytes; always 0x0303 for TLS 1.3
    uint16 length;                  // 2 bytes; up to 2^14 + 256 = 16640 bytes
    opaque encrypted_record[length];// AEAD ciphertext + auth tag
} TLSCiphertext;
```

Record fragmentation: max 2^14 (16384) plaintext bytes per record. AEAD overhead (AES-GCM, ChaCha20-Poly1305) is **16 bytes** per record (the auth tag). For large transfers, that overhead amortizes to **0.097%**. For small writes (one byte per record) overhead dominates — use buffered writes.

---

## Layer 7 (Application)

Layer 7 is where humans and programs see the network. The protocol catalog at this layer is enormous; the common ones cluster into request/response, publish/subscribe, streaming, and remote-execution patterns.

### Protocol catalog

| Protocol | RFC / Spec | Transport | Default port | Pattern |
|:---|:---|:---|:---:|:---|
| HTTP/1.1 | RFC 9110/9112 | TCP | 80 / 443 (TLS) | Request/response, text framing |
| HTTP/2 | RFC 9113 | TCP | 443 | Multiplexed binary frames over TLS |
| HTTP/3 | RFC 9114 | UDP/QUIC | 443 | Multiplexed over QUIC |
| WebSocket | RFC 6455 | TCP (HTTP upgrade) | 80/443 | Bidirectional message frames |
| gRPC | grpc.io spec | HTTP/2 | varies | RPC over HTTP/2 + Protobuf |
| DNS | RFC 1034/1035, 9156 | UDP/TCP/QUIC | 53 / 853 (DoT) / 443 (DoH) | Request/response |
| SMTP | RFC 5321 | TCP | 25 / 465 / 587 | Mail submission/relay |
| IMAP4 | RFC 9051 | TCP | 143 / 993 | Mail retrieval, server-side state |
| POP3 | RFC 1939 | TCP | 110 / 995 | Mail download |
| FTP | RFC 959 | TCP (separate ctrl/data) | 21 / 20 | File transfer |
| SFTP | draft-ietf-secsh-filexfer | TCP/SSH | 22 | File transfer over SSH |
| SSH | RFC 4251-4254 | TCP | 22 | Shell, port forward, file copy |
| LDAP | RFC 4511 | TCP | 389 / 636 | Directory queries (BER-encoded) |
| MQTT | OASIS MQTT 5 | TCP/WS | 1883 / 8883 | Pub/sub, IoT |
| AMQP | OASIS AMQP 1.0 | TCP | 5672 / 5671 | Brokered messaging |
| NTP | RFC 5905 | UDP | 123 | Time synchronization |
| SNMPv3 | RFC 3414 | UDP | 161 / 162 | Network management |
| SIP | RFC 3261 | UDP/TCP | 5060 / 5061 | VoIP signaling |
| RTP / RTCP | RFC 3550 | UDP | dynamic | Real-time media |

### Application-layer addressing

URIs (RFC 3986) are the application-layer "address" most apps use:

```
scheme://userinfo@host:port/path?query#fragment
```

Relative URIs, content negotiation (Accept, Accept-Encoding), and RESTful resource naming are all Layer 7 concepts that have no analog at Layers 1-6.

### Common Layer 7 framing pitfalls

- **Head-of-line blocking** in HTTP/1.1 keep-alive: one slow response stalls all subsequent on the same connection. HTTP/2 fixes within a connection; HTTP/3+QUIC fixes across all streams.
- **Slowloris attack** — adversary opens many connections and sends headers byte-by-byte. Mitigated with read timeouts and connection limits.
- **Request smuggling (HTTP/1.1 CL.TE / TE.CL)** — front-end and back-end disagree on framing. Mitigated by rejecting ambiguous Transfer-Encoding/Content-Length combinations.
- **WebSocket fragment confusion** — large frames split across TCP segments are easy to mishandle in custom parsers; use a vetted library.

---

## OSI vs TCP/IP — exact mapping table

The mapping below is the de facto consensus from RFC 1122, ITU-T X.200 references, and modern architecture texts (Tanenbaum, Kurose & Ross, Stevens). The TCP/IP "Application" layer is intentionally broad because the original ARPANET design folded session/presentation responsibilities into apps.

```
+--------+------------------+-------------------+----------------------------------------+
|  OSI # |   OSI Name       |  TCP/IP Layer     |  Concrete protocol examples            |
+--------+------------------+-------------------+----------------------------------------+
|   7    |  Application     |  Application      |  HTTP, DNS, SMTP, SSH, FTP, gRPC, MQTT |
|   6    |  Presentation    |  Application      |  TLS records, ASN.1, JSON, Protobuf    |
|   5    |  Session         |  Application      |  TLS sessions, gRPC streams, RPC       |
|   4    |  Transport       |  Transport        |  TCP, UDP, SCTP, QUIC, DCCP            |
|   3    |  Network         |  Internet         |  IPv4, IPv6, ICMP, IGMP, IPsec ESP/AH  |
|   2    |  Data Link       |  Link / Network   |  Ethernet, PPP, HDLC, 802.11, ARP, MPLS|
|        |                  |  Access           |                                        |
|   1    |  Physical        |  Link / Network   |  10BASE-T, 1000BASE-LX, 802.11ax PHY   |
|        |                  |  Access           |                                        |
+--------+------------------+-------------------+----------------------------------------+
```

A few protocols defy clean Layer mapping:

- **MPLS** — sits between L2 and L3, often called "Layer 2.5".
- **ARP** (RFC 826) — uses an Ethernet frame but resolves an L3 address; commonly called L2.5.
- **GRE** (RFC 2784) — encapsulates almost any protocol in IP; technically L3, often described as a tunnel layer.
- **IPsec ESP** (RFC 4303) — encrypts a Layer 3 packet; lives in L3.
- **TLS** — encrypts above TCP; lives in L6, but DTLS atop UDP is often deployed as if it were L4.5.
- **QUIC** (RFC 9000) — combines L4 reliability, L5 connection state, and L6 TLS in one protocol over UDP.

Modern stacks blur layers; the model survives because **the failure modes still cluster by layer**, which is what makes it operationally useful.

---

## Encapsulation overhead math

Every byte of payload pays a tax to each layer's header. The classic "TCP-over-IPv4-over-Ethernet II" stack:

```
+---------------+---------------+---------------+---------------+
|  Eth Hdr (14) |  IPv4 Hdr (20)|  TCP Hdr (20) |  Payload      |  +  FCS (4)
+---------------+---------------+---------------+---------------+
```

Per-frame fixed overhead = 14 + 20 + 20 + 4 = **58 bytes** (counting FCS) or **54 bytes** (excluding FCS, common quote).

For a maximum-size 1500-byte Ethernet payload (MTU):

```
TCP_payload_max = 1500 - 20 (IP) - 20 (TCP) = 1460 bytes  (the MSS)
Eth_frame_size  = 14 + 20 + 20 + 1460 + 4 = 1518 bytes on the wire (without preamble)
With preamble + IFG:
  preamble (7) + SFD (1) + frame (1518) + IFG (12) = 1538 bytes wire time per packet.
Useful_efficiency = 1460 / 1538 = 94.93%
```

For an IPv6 + TCP frame (no IP options):

```
IPv6 header = 40 bytes (vs IPv4 20)
MSS = 1500 - 40 - 20 = 1440 bytes
Efficiency = 1440 / 1538 = 93.63%
```

For TLS 1.3 atop TCP atop IPv4 atop Ethernet, with one TLS record per segment:

```
TLS overhead = 5 (header) + 16 (AEAD tag) = 21 bytes
Effective MSS = 1460 - 21 = 1439 bytes
Efficiency vs raw payload = 1439 / 1538 = 93.56%
```

### Jumbo frame impact

Standard Ethernet allows 1500-byte payloads. Jumbo frames extend to typically **9000 bytes** (sometimes 9216 to accommodate VLAN/LLC):

```
TCP_MSS_jumbo = 9000 - 20 - 20 = 8960 bytes
Per-frame fixed overhead = 58 bytes (constant)
Efficiency = 8960 / (8960 + 58 + 12 preamble/IFG) = 8960 / 9030 = 99.22%
```

That is **+4.3%** more efficient than standard MTU for bulk transfer. Jumbo frames also reduce CPU per-frame interrupts by ~6× — worth it on storage and HPC fabrics.

### Tunneling overhead

| Tunnel | Overhead per packet |
|:---|:---:|
| GRE (RFC 2784) over IPv4 | 24 bytes (20 outer IP + 4 GRE) |
| IPv4-in-IPv4 (RFC 2003) | 20 bytes |
| 6in4 (RFC 4213) | 20 bytes |
| MPLS (per label) | 4 bytes per label; usually 2 labels = 8 bytes |
| VXLAN (RFC 7348) over IPv4 | 50 bytes (20 IP + 8 UDP + 8 VXLAN + 14 inner Eth) |
| Geneve (RFC 8926) over IPv4 | 50+ bytes (8 Geneve fixed + variable TLVs) |
| IPsec ESP transport (AES-GCM) | 18-25 bytes (8 SPI/seq + 16 ICV + 1-9 padding) |
| IPsec ESP tunnel (AES-GCM, IPv4) | 38-45 bytes (above + 20 outer IP) |
| WireGuard (RFC draft) | 32 bytes (4 type + 4 idx + 8 ctr + 16 Poly1305) + 20 outer IP + 8 UDP = 60 bytes |
| L2TPv3 over IP (RFC 3931) | 12 bytes (no UDP) |
| GTP-U (3GPP TS 29.281) over UDP/IPv4 | 36 bytes (20 IP + 8 UDP + 8 GTP) |

Stacking tunnels compounds. A VXLAN-in-IPsec deployment can lose **70+ bytes** per packet, dropping effective MSS to **~1400 bytes** and hammering OSPF/BGP keepalives that assume 1480-byte MSS.

---

## MTU math

### Path MTU Discovery (RFC 1191 / RFC 8201)

PMTUD finds the smallest MTU on a path so the source can presize segments and avoid fragmentation.

#### IPv4 PMTUD

1. Source sends DF=1 packets at the local interface MTU.
2. Router with smaller egress MTU returns ICMP Type 3, Code 4 ("Fragmentation Needed and DF set") with `MTU of next-hop` field.
3. Source caches the new MTU per destination, retries.

```
Initial MTU        Local interface = 1500
ICMP Frag-Needed   "next-hop = 1480"
Source caches      1480 for that destination
```

If ICMP is filtered (a common firewall mistake), PMTUD black-hole results — packets vanish. Symptom: short flows succeed (handshake fits in MSS), bulk transfer hangs.

#### IPv6 PMTUD

Identical mechanism but uses **ICMPv6 Type 2 (Packet Too Big)**. RFC 4890 mandates this be permitted through firewalls; filtering is malpractice.

#### Packetization Layer PMTUD (RFC 4821)

PLPMTUD avoids ICMP entirely. The Layer 4 itself probes — TCP sends progressively larger segments and watches for loss. Slower convergence but resilient to ICMP filtering. Linux enables it (`net.ipv4.tcp_mtu_probing=1`).

### Fragmentation cost

Fragmented packets are catastrophic for performance:

- **Reassembly is end-host work** — adds CPU and buffer pressure.
- **Loss of one fragment loses the whole datagram** — IP has no per-fragment retransmit.
- **Stateful firewalls and NATs may drop fragments** — first frag has L4 ports, subsequent do not.
- **DNSSEC and EDNS0 over UDP** depend on PMTUD or UDP-fragmentation tolerance — broken paths cause random DNS resolution failures (motivated DoH/DoT/DoQ).

Probability a `k`-fragment datagram arrives intact when per-frag loss is `p`:

```
P_arrive = (1 - p)^k
```

For `p = 0.01`, `k = 4`: `P_arrive = 0.96`. For `p = 0.05`, `k = 4`: `P_arrive = 0.81`. Compare to the unfragmented case at `p = 0.01`: `P_arrive = 0.99`.

### Common MTU values

| MTU | Where |
|:---:|:---|
| 65,535 | Theoretical IPv4 max (16-bit Total Length) |
| 65,527 | IPv6 max single packet (Total Length 16 bits + 40 IPv6 header — 0 = jumbogram with HBH ext = 4 GB max via Jumbo Payload option, RFC 2675) |
| 16,110 | InfiniBand IPoIB |
| 9216 | Jumbo (some Cisco/Arista) |
| 9000 | Jumbo (de facto default) |
| 4470 | FDDI / older HiPPI |
| 4500 | MPLS over Ethernet inner default |
| 2304 | 802.11 (with overhead) |
| 1500 | Standard Ethernet IEEE 802.3 |
| 1492 | PPPoE (1500 - 8) |
| 1480 | GRE over IPv4 (1500 - 20) |
| 1452 | DSL with PPPoA |
| 1280 | IPv6 minimum (RFC 8200 §5) |
| 576  | IPv4 minimum reassembly (RFC 791 §3.2) |
| 296  | SLIP minimum |
| 68   | IPv4 minimum link MTU (RFC 791 §3.2) |
| 40   | IPv6 minimum header itself |

The **1280** floor for IPv6 is critical: every link must transport at least a 1280-byte packet end-to-end. This guarantees DNS/DHCPv6/NDP work without fragmentation.

---

## Why OSI matters in 2026

The OSI model is criticized as "academic" — TCP/IP won, no real protocols implement OSI session/presentation. Yet the model survives in 2026 because it provides three durable benefits.

### A debugging mental model

When a service is "down," the layered model is the fastest triage tool ever invented. From the bottom up:

```
L1: Is the cable plugged in? Link LED?       -> 'ip link show', 'ethtool eth0'
L2: ARP/MAC resolved?                          -> 'ip neigh show', 'arp -an'
L3: Routing reaches the destination?           -> 'ip route get 1.1.1.1', 'mtr', 'traceroute'
L4: TCP/UDP port open? Three-way handshake?    -> 'nc -vz host port', 'ss -tnp', 'tcpdump -ni any'
L5/6: TLS handshake completes?                 -> 'openssl s_client -connect host:443'
L7: Application response valid?                 -> 'curl -v https://host/path'
```

Asking "what's the lowest layer that's still working?" converges on the failure point in minutes.

### Vendor "feature at layer X" claims

Every networking vendor pitches "Layer 4 load balancing" vs "Layer 7 load balancing", "Layer 2 segmentation" vs "Layer 3 micro-segmentation", "Layer 7 firewall", etc. The OSI vocabulary is the contract:

| Claim | What it means in practice |
|:---|:---|
| "Layer 2 switch" | Forwards by MAC address, ignores L3+. |
| "Layer 3 switch" | Routes by IP, line-rate hardware FIB lookup. |
| "Layer 4 load balancer" | Forwards by 5-tuple, no payload inspection (e.g. AWS NLB, IPVS). |
| "Layer 7 load balancer" | Inspects HTTP path/headers/cookies (e.g. nginx, HAProxy, Envoy). |
| "Layer 7 firewall" | Inspects application payload (snort, suricata, NGFW). |
| "Layer 2 VPN (L2VPN)" | Bridges Ethernet segments across an IP cloud (VXLAN, EVPN). |
| "Layer 3 VPN (L3VPN)" | Tunnels IP between sites (MPLS L3VPN, IPsec). |
| "Layer 4-7 services" | LB + firewall + WAF + DLP — a deliberately fuzzy term. |

Without the vocabulary, you can't read the data sheet.

### ISO 27001 / SOC2 controls map to layers

Modern security frameworks tag controls by layer because **mitigation effectiveness is layer-specific**:

| Control family | OSI Layer | Examples |
|:---|:---:|:---|
| Physical security | 1 | Locked racks, FIPS-140 tamper evidence, BadUSB defenses. |
| Link-layer access control | 2 | 802.1X port auth, MACsec encryption, ARP inspection. |
| Network segmentation | 3 | VLANs, ACLs, microsegmentation policies. |
| Transport security | 4 | TLS termination policy, DDoS rate-limiters. |
| Session continuity | 5/6 | TLS resumption tokens, session hijacking prevention. |
| Data-at-encoding | 6 | Field-level encryption, Protobuf schema validation. |
| Application controls | 7 | WAF, OWASP Top 10 mitigations, RBAC. |

ISO 27001 Annex A (114 controls in 2022 revision), CIS Benchmarks, and NIST 800-53 all map roughly to the OSI layers. Compliance auditors literally ask "show me your Layer 7 protections."

---

## Layer-X attacks

A layer-by-layer rogue's gallery of common attacks and mitigations.

### Layer 1 attacks

| Attack | Description | Mitigation |
|:---|:---|:---|
| RF jamming | Flood spectrum to drown legitimate signals. | Spread spectrum, frequency hopping, regulatory enforcement. |
| Cable tapping | Optical tap, vampire tap on copper. | Tamper-evident conduit, MACsec encryption, fiber bend-loss detectors. |
| TEMPEST emissions | Side-channel from EM leakage. | Shielded rooms, NSA TEMPEST-rated equipment. |
| Power grid attack | Crash the switch by killing PoE upstream. | UPS, diverse power feeds, redundant uplinks. |
| Optical fiber bending | Inject/leak signals via micro-bend. | OTDR monitoring, sealed cable plant. |

### Layer 2 attacks

| Attack | Description | Mitigation |
|:---|:---|:---|
| ARP poisoning (RFC 826 abuse) | Inject false `IP→MAC` mappings. | Dynamic ARP Inspection (DAI), 802.1X, static ARP for critical hosts. |
| MAC flooding (CAM overflow) | Flood switch CAM table → switch becomes hub. | Port-security, sticky MAC, limit MACs per port. |
| VLAN hopping (double-tag) | Stack two 802.1Q tags to traverse VLANs. | Disable DTP; explicit native VLAN; trunk allowlists. |
| STP root takeover | Send superior BPDU to become root bridge. | BPDU Guard on access ports, Root Guard on uplinks. |
| 802.11 deauth flood | Spoof deauth frames to kick clients. | 802.11w PMF (protected management frames). |
| Rogue DHCP | Hand out malicious gateway/DNS. | DHCP Snooping, DHCPv6 RA Guard. |
| LLDP/CDP spoofing | Pretend to be a known network device. | Disable on edge ports; AAA tied to hardware ID. |
| MAC spoofing | Forge a peer's MAC to bypass filtering. | 802.1X with EAP-TLS; MACsec. |

### Layer 3 attacks

| Attack | Description | Mitigation |
|:---|:---|:---|
| IP spoofing | Forge source IP. | uRPF (unicast Reverse Path Forwarding), BCP 38 ingress filtering. |
| ICMP smurf attack | Spoofed ICMP echoes amplified via broadcast. | Disable directed broadcasts (RFC 2644). |
| Ping of Death | Oversize fragmented ICMP. | Modern stacks immune; sanity-check reassembled length. |
| Teardrop | Overlapping IP fragments. | RFC 5722 — drop all overlapping fragments. |
| ICMP redirect injection | Force traffic through attacker. | Disable ICMP redirects (`net.ipv4.conf.all.accept_redirects=0`). |
| BGP hijacking | Announce more-specific prefix. | RPKI, BGPsec, IRR validation. |
| Route leaks | Accidentally re-advertise customer routes to peers. | RFC 8212 default-deny, RPKI ROAs, communities. |
| OSPF/IS-IS spoofing | Inject false LSAs. | Authentication (HMAC-SHA), TTL=255 GTSM. |

### Layer 4 attacks

| Attack | Description | Mitigation |
|:---|:---|:---|
| TCP SYN flood | Exhaust server connection queue. | SYN cookies (RFC 4987), backlog tuning, hardware mitigations. |
| TCP RST injection | Forge RST mid-flow. | TCP-AO (RFC 5925), unpredictable seq numbers, TCP MD5 (legacy). |
| Connection exhaustion | Open many half-open connections. | Per-source rate limiting, idle timeouts. |
| UDP amplification (DNS, NTP, memcached) | Spoof source, get amplified response. | BCP 38, NTP `monlist` disabled, Memcached UDP off, DNS rate-limit. |
| TCP slow start abuse | Fool BBR/Cubic with crafted ACKs. | ABC, careful ACK validation. |
| Sequence prediction | Old Mitnick-era attack. | Modern PRNG-based ISN (RFC 6528). |
| TIME_WAIT exhaustion | Burn ephemeral ports on busy LB. | Increase port range, `tcp_tw_reuse`, connection reuse. |

### Layer 5/6 attacks

| Attack | Description | Mitigation |
|:---|:---|:---|
| TLS downgrade (POODLE, Logjam) | Force weaker cipher/protocol. | Disable SSLv3/TLS1.0/1.1, prefer TLS1.3, HSTS. |
| BEAST / CRIME / BREACH | Compression/CBC side channels. | TLS 1.3 forbids RC4, CBC-mode patches, `Vary: Accept-Encoding`. |
| Heartbleed (CVE-2014-0160) | OpenSSL heartbeat bug leaks memory. | Patch OpenSSL ≥ 1.0.1g. |
| Renegotiation MITM | Inject before TLS renegotiation. | Disable client-initiated renegotiation; secure renegotiation extension. |
| Padding oracle (Lucky-13) | Exploit AES-CBC error timing. | TLS 1.3 only AEAD, Lucky-13 patches. |
| Session hijacking | Steal cookie/session-id. | HttpOnly + Secure + SameSite cookies, short TTL, rebind on privilege escalation. |
| ASN.1 parser confusion (Frankencerts, ROBOT) | Permissive certificate parser. | Strict DER, fuzzed parsers, X.509-validators enforce DER, not BER. |

### Layer 7 attacks

| Attack | Description | Mitigation |
|:---|:---|:---|
| SQL injection | `' OR '1'='1` etc. | Parameterized queries, stored procs, input validation, WAF. |
| Cross-site scripting (XSS) | Inject `<script>` into another user's view. | Output encoding, CSP headers, framework auto-escape. |
| Cross-site request forgery (CSRF) | Trick logged-in user's browser. | SameSite cookies, anti-CSRF tokens, same-origin checks. |
| Server-side request forgery (SSRF) | Make server fetch attacker URL. | Egress allowlists, IMDSv2 (AWS), URL parser hardening. |
| HTTP request smuggling | CL/TE header confusion. | RFC 9110 strict parsing, reject ambiguous frames. |
| Path traversal | `../../etc/passwd`. | Canonicalize and validate paths, chroot/sandbox. |
| OAuth flow abuse | Stolen authorization code, token theft. | PKCE, short-lived tokens, redirect-URI exact match. |
| Slowloris | Drip-feed HTTP headers. | Header timeout, per-IP connection cap. |
| GraphQL introspection abuse | Enumerate schema for attack surface. | Disable introspection in prod, query depth limit, complexity analysis. |
| Open redirect | `?next=https://evil` parameter. | Allowlist redirect targets, server-relative redirects only. |
| Cache poisoning | Crafted Vary/Host header gets bad response cached. | Defense-in-depth, sanitize cache keys. |

---

## Worked examples

### Example 1 — `curl https://example.com` packet trace through all 7 layers

Assume:
- Local interface MTU 1500
- IPv4 dual-stack, one TCP connection
- TLS 1.3 with X25519 + AES-128-GCM
- HTTP/1.1 GET /

Layer-by-layer byte accounting for the **first request packet** (after handshake):

```
+=========================================================================+
|  L7  HTTP request (text)                                                |
|       "GET / HTTP/1.1\r\n"                                              |
|       "Host: example.com\r\n"                                           |
|       "User-Agent: curl/8.5.0\r\n"                                      |
|       "Accept: */*\r\n"                                                 |
|       "\r\n"                                                            |
|       => 78 bytes plaintext                                              |
+=========================================================================+
|  L6  TLS 1.3 record encryption                                          |
|       record_header (5) + plaintext (78) + AEAD_tag (16) = 99 bytes     |
|       Ciphertext: 99 bytes                                               |
+=========================================================================+
|  L5  No explicit Layer 5 — TLS session_id is implicit                   |
+=========================================================================+
|  L4  TCP segment                                                        |
|       TCP header (20) + payload (99) = 119 bytes                        |
+=========================================================================+
|  L3  IPv4 packet                                                        |
|       IPv4 header (20) + segment (119) = 139 bytes                      |
+=========================================================================+
|  L2  Ethernet II frame                                                  |
|       MAC dst (6) + MAC src (6) + EtherType (2) +                       |
|       payload (139) + FCS (4) = 157 bytes                               |
+=========================================================================+
|  L1  10GBASE-T physical                                                 |
|       Preamble (7) + SFD (1) + frame (157) +                            |
|       IFG (12) = 177 bytes wire time                                    |
|       64B/66B encoding overhead × 1.03125 = 182 raw line bytes          |
+=========================================================================+
```

Useful payload ratio: `78 / 182 = 42.9%`. The connection setup (TLS handshake) costs another ~5 KB before this first request. For a long-lived keep-alive HTTP/1.1 connection or HTTP/2 multiplexed stream, fixed overhead amortizes — at MTU-sized payload (1460 bytes TCP), ratio climbs to **94%+**.

### Example 2 — Wireshark interpretation per layer

A typical Wireshark capture shows the OSI model literally as a tree:

```
Frame 1: 157 bytes on wire (1256 bits), 157 bytes captured (1256 bits)
   [Layer 1 metadata: arrival timestamp, interface]
Ethernet II, Src: aa:bb:cc:11:22:33, Dst: dd:ee:ff:44:55:66
   [Layer 2: MAC addresses, EtherType=0x0800]
Internet Protocol Version 4, Src: 192.0.2.10, Dst: 198.51.100.1
   [Layer 3: TTL, identification, fragment offset, header checksum]
Transmission Control Protocol, Src Port: 53124, Dst Port: 443, Len: 99
   [Layer 4: seq, ack, flags, window, checksum]
Transport Layer Security
   [Layer 5/6: TLSv1.3 Record Layer: Application Data Protocol: http-over-tls]
Hypertext Transfer Protocol
   [Layer 7: GET / HTTP/1.1, headers]
```

The protocol expansion mirrors the OSI stack. Filters use this taxonomy:

```bash
# L2 filter
tcpdump -ni eth0 'ether host aa:bb:cc:11:22:33'
# L3 filter
tcpdump -ni eth0 'host 192.0.2.10 and not host 192.0.2.11'
# L4 filter
tcpdump -ni eth0 'tcp port 443'
# L7 filter (in Wireshark display filter)
http.request.method == "GET" and http.host == "example.com"
```

### Example 3 — total wire bytes for a 1 KB HTTP GET

Assume 1024-byte response body, HTTP/1.1, plain HTTP (no TLS), Ethernet II, IPv4.

```
Request:
  HTTP req     ~ 80 bytes
  TCP hdr      = 20 bytes
  IP hdr       = 20 bytes
  Eth + FCS    = 18 bytes
  Preamble+IFG = 20 bytes
  Total req     = 158 bytes wire

Response:
  HTTP headers ~ 200 bytes
  Body         = 1024 bytes
  Total HTTP   = 1224 bytes plaintext
  TCP segments needed at MSS=1460 = 1 segment (fits)
  TCP+IP+Eth+FCS = 58 bytes
  Preamble+IFG = 20 bytes
  Total resp   = 1224 + 58 + 20 = 1302 bytes wire

Plus 3-way handshake (3 × 78 wire bytes) = 234 bytes
Plus FIN exchange (4 × 78 wire bytes) = 312 bytes
Plus ACKs for the response (typically 1) = 78 bytes

Total wire = 158 + 1302 + 234 + 312 + 78 = 2084 bytes
Useful HTTP body = 1024 bytes
Efficiency = 1024 / 2084 = 49.1%
```

For a 100 KB body the efficiency rises to **~95%** because the per-connection overhead amortizes.

### Example 4 — SDN vs traditional networking layering

Traditional networking does control + data plane on every router:

```
Router 1 ----------- Router 2 ----------- Router 3
[OSPF, BGP CPU]      [OSPF, BGP CPU]      [OSPF, BGP CPU]
[FIB on linecard]    [FIB on linecard]    [FIB on linecard]
```

SDN separates them:

```
                  +------------------+
                  | SDN Controller   |  (centralized control plane,
                  | (OpenFlow,       |   knows full network graph)
                  | NETCONF, P4Runtime)|
                  +------------------+
                       |        |
                       v        v
        +------------+    +------------+    +------------+
        | OF switch1 |    | OF switch2 |    | OF switch3 |
        | match+act  |    | match+act  |    | match+act  |
        +------------+    +------------+    +------------+
```

The OSI model still applies, but the **control plane** becomes a Layer 7 application that programs the **data plane** at L2-L4. P4 (Programming Protocol-independent Packet Processors) lets the operator define **custom Layer 1-4 parsing** in a high-level language; the chip is reprogrammed accordingly.

This blurs the line between "network protocol" and "application" — the OSI reference still grounds the conversation. A P4-programmed switch is a Layer 2/3/4 device whose behavior is described by a Layer 7 program.

---

## See Also

- `networking/tcp` — transport-layer reference
- `networking/ip` — IPv4 and IPv6 packet structure
- `networking/ethernet` — Layer 2 framing
- `networking/dns` — Layer 7 application protocol example
- `ramp-up/osi-model-eli5` — narrative companion to this page
- `ramp-up/tcp-eli5` — friendly TCP introduction

---

## References

- ITU-T Recommendation X.200 (07/1994) — Information technology — Open Systems Interconnection — Basic Reference Model: The basic model.
- ISO/IEC 7498-1:1994 — Information technology — Open Systems Interconnection — Basic Reference Model: The Basic Model.
- ISO/IEC 7498-2:1989 — Security Architecture (Layer-mapped security services).
- RFC 1122 — Requirements for Internet Hosts — Communication Layers (Braden, 1989).
- RFC 1123 — Requirements for Internet Hosts — Application and Support (Braden, 1989).
- RFC 791 — Internet Protocol (Postel, 1981).
- RFC 793 / 9293 — Transmission Control Protocol (Postel, 1981 / Eddy, 2022).
- RFC 768 — User Datagram Protocol (Postel, 1980).
- RFC 826 — Address Resolution Protocol (Plummer, 1982).
- RFC 1191 — Path MTU Discovery (Mogul, Deering, 1990).
- RFC 8201 — Path MTU Discovery for IPv6 (McCann, Deering, Mogul, Hinden, 2017).
- RFC 4821 — Packetization Layer Path MTU Discovery (Mathis, Heffner, 2007).
- RFC 8200 — Internet Protocol, Version 6 (IPv6) Specification (Deering, Hinden, 2017).
- RFC 4443 — ICMPv6 (Conta, Deering, Gupta, 2006).
- RFC 4890 — Recommendations for Filtering ICMPv6 Messages in Firewalls (Davies, Mohacsi, 2007).
- RFC 7323 — TCP Extensions for High Performance (Borman, Braden, Jacobson, Scheffenegger, 2014).
- RFC 8446 — The Transport Layer Security (TLS) Protocol Version 1.3 (Rescorla, 2018).
- RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport (Iyengar, Thomson, 2021).
- RFC 9110/9111/9112/9113/9114 — HTTP Semantics, Caching, HTTP/1.1, HTTP/2, HTTP/3.
- RFC 1034 / 1035 — Domain Names — Concepts and Facilities / Implementation (Mockapetris, 1987).
- IEEE 802.3-2022 — Ethernet (frame format, autonegotiation, FEC).
- IEEE 802.1Q-2022 — Bridges and Bridged Networks (VLAN tagging).
- IEEE 802.1X-2020 — Port-Based Network Access Control.
- IEEE 802.11-2020 — Wireless LAN MAC and PHY.
- ITU-T G.957 / G.652 — Optical fiber characteristics.
- Tanenbaum, A.S. and Wetherall, D.J. — *Computer Networks*, 6th ed., Pearson (2021).
- Kurose, J. and Ross, K. — *Computer Networking: A Top-Down Approach*, 8th ed., Pearson (2021).
- Stevens, W.R. — *TCP/IP Illustrated, Vol. 1*, 2nd ed., Addison-Wesley (2011).
- Peterson, L. and Davie, B. — *Computer Networks: A Systems Approach*, 6th ed., Morgan Kaufmann (2021).
- Day, J. — *Patterns in Network Architecture*, Prentice Hall (2008) — for a critical re-evaluation of OSI.
- Zimmermann, H. — "OSI Reference Model — The ISO Model of Architecture for Open Systems Interconnection," IEEE Trans. Commun. (1980).
- Saltzer, J., Reed, D., Clark, D. — "End-to-End Arguments in System Design," ACM TOCS (1984).
