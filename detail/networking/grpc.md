# The Mathematics of gRPC — Protobuf Encoding, HTTP/2 Framing & Flow Control

> *gRPC's performance advantage is mathematical: varint encoding compresses integers to 1-5 bytes instead of fixed 4-8, HTTP/2 multiplexing eliminates connection overhead, and flow control windows balance throughput against memory. Every byte on the wire is accounted for.*

---

## 1. Protobuf Varint Encoding (Variable-Length Integer Encoding)

### The Problem

Protocol Buffers encode integers using variable-length encoding (varints). How many bytes does each value require, and how much space does this save compared to fixed-width encoding?

### The Formula

A varint uses 7 bits of data per byte, with the MSB as a continuation bit:

$$\text{bytes}(n) = \left\lceil \frac{\lfloor \log_2(n) \rfloor + 1}{7} \right\rceil$$

For $n = 0$: 1 byte (special case).

### Worked Examples

| Value $n$ | Binary | Varint Bytes | Fixed32 Bytes | Savings |
|:---:|:---|:---:|:---:|:---:|
| 0 | 0 | 1 | 4 | 75% |
| 1 | 1 | 1 | 4 | 75% |
| 127 | 1111111 | 1 | 4 | 75% |
| 128 | 10000000 | 2 | 4 | 50% |
| 16,383 | 14 bits | 2 | 4 | 50% |
| 16,384 | 15 bits | 3 | 4 | 25% |
| 2,097,151 | 21 bits | 3 | 4 | 25% |
| 2,097,152 | 22 bits | 4 | 4 | 0% |
| 268,435,455 | 28 bits | 4 | 4 | 0% |
| 268,435,456 | 29 bits | 5 | 4 | -25% |
| $2^{32} - 1$ | 32 bits | 5 | 4 | -25% |
| $2^{63} - 1$ | 63 bits | 9 | 8 | -12.5% |
| $2^{64} - 1$ | 64 bits | 10 | 8 | -25% |

Key insight: varints save space for values < 2,097,152 (~2M). For larger values, fixed encoding is more efficient. Most real-world IDs, counts, and enum values are well within the efficient range.

### ZigZag Encoding (Signed Integers)

Protobuf's `sint32`/`sint64` use ZigZag encoding to handle negative numbers efficiently:

$$\text{zigzag}(n) = \begin{cases} 2n & \text{if } n \geq 0 \\ -2n - 1 & \text{if } n < 0 \end{cases}$$

| Signed Value | ZigZag Value | Varint Bytes |
|:---:|:---:|:---:|
| 0 | 0 | 1 |
| -1 | 1 | 1 |
| 1 | 2 | 1 |
| -2 | 3 | 1 |
| 63 | 126 | 1 |
| -64 | 127 | 1 |
| 64 | 128 | 2 |

Without ZigZag, `-1` as an `int32` would be encoded as `0xFFFFFFFF` (5 varint bytes). With ZigZag, it is 1 (1 byte).

---

## 2. Protobuf Wire Format (Tag-Length-Value)

### The Problem

How much overhead does Protobuf's wire format add per field, and how does this compare to JSON?

### The Formula

Each field is encoded as:

$$\text{field bytes} = \text{tag} + \text{length (if applicable)} + \text{value}$$

Tag encoding:

$$\text{tag} = (\text{field\_number} << 3) | \text{wire\_type}$$

Wire types:

| Wire Type | Meaning | Used For |
|:---:|:---|:---|
| 0 | Varint | int32, int64, uint32, uint64, sint32, sint64, bool, enum |
| 1 | 64-bit | fixed64, sfixed64, double |
| 2 | Length-delimited | string, bytes, embedded messages, packed repeated |
| 5 | 32-bit | fixed32, sfixed32, float |

### Size Comparison: Protobuf vs JSON

For a user message: `{id: 12345, name: "Alice", email: "alice@example.com", age: 30}`

**JSON:**
```
{"id":12345,"name":"Alice","email":"alice@example.com","age":30}
```
Size: 63 bytes

**Protobuf:**
```
Field 1 (id=12345):    tag(1) + varint(3) = 4 bytes
Field 2 (name):        tag(1) + len(1) + "Alice"(5) = 7 bytes
Field 3 (email):       tag(1) + len(1) + "alice@example.com"(17) = 19 bytes
Field 4 (age=30):      tag(1) + varint(1) = 2 bytes
Total: 32 bytes
```

$$\text{Compression ratio} = 1 - \frac{32}{63} = 49.2\%$$

### Scaling with Repeated Fields

For $N$ repeated User messages:

| $N$ | JSON Size | Protobuf Size | Ratio |
|:---:|:---:|:---:|:---:|
| 1 | 63 B | 32 B | 50.8% |
| 10 | 632 B | 322 B | 50.9% |
| 100 | 6,302 B | 3,202 B | 50.8% |
| 1,000 | 63,002 B | 32,002 B | 50.8% |

For integer-heavy messages (metrics, IDs), Protobuf achieves 70-80% compression. For string-heavy messages, the advantage narrows to 40-50%.

---

## 3. HTTP/2 Frame Overhead (Per-Message Cost)

### The Problem

gRPC sends each message as one or more HTTP/2 DATA frames preceded by a 5-byte gRPC length-prefixed message header. What is the total overhead per RPC?

### The Formula

Per gRPC message:

$$S_{total} = 5_{gRPC\_header} + S_{protobuf} + 9_{HTTP2\_frame\_header}$$

The 5-byte gRPC header: 1 byte compression flag + 4 bytes message length.

For a complete unary RPC (request + response):

$$S_{RPC} = S_{request\_headers} + S_{request\_data} + S_{response\_headers} + S_{response\_data} + S_{trailers}$$

### Worked Examples

Request headers (HPACK compressed, warm connection): ~20-50 bytes
Response headers: ~20-30 bytes
Trailers (grpc-status, grpc-message): ~15-25 bytes

| Message Size | gRPC + HTTP/2 Overhead | Total Wire Size | Overhead % |
|:---:|:---:|:---:|:---:|
| 10 B | ~120 B | ~130 B | 92.3% |
| 100 B | ~120 B | ~220 B | 54.5% |
| 1 KB | ~120 B | ~1,144 B | 10.5% |
| 10 KB | ~120 B | ~10,360 B | 1.2% |
| 100 KB | ~120 B | ~102,520 B | 0.12% |
| 1 MB | ~120 B | ~1,048,696 B | 0.01% |

### Comparison with REST/JSON over HTTP/1.1

| Protocol | Headers (typical) | Payload Encoding | Connection Setup |
|:---|:---:|:---|:---:|
| REST/JSON HTTP/1.1 | 400-800 B | JSON (text) | Per request (no keep-alive) or shared |
| REST/JSON HTTP/2 | 20-50 B (HPACK) | JSON (text) | Shared (multiplexed) |
| gRPC HTTP/2 | 20-50 B (HPACK) | Protobuf (binary) | Shared (multiplexed) |

For small messages (<100 bytes), the protocol overhead dominates. For large messages, the Protobuf encoding advantage (50% smaller) is the primary win.

---

## 4. Deadline Propagation (Time Budget Arithmetic)

### The Problem

gRPC propagates deadlines through service call chains. How much time does each service have, and when should it give up?

### The Formula

If the client sets a deadline at absolute time $D$, and service $i$ receives the call at time $t_i$:

$$\text{remaining}_i = D - t_i$$

The service must complete its work AND any downstream calls within $\text{remaining}_i$.

For a call chain $A \to B \to C$:

$$\text{remaining}_C = D - t_A - \delta_{A \to B} - \delta_{process_B} - \delta_{B \to C}$$

Where $\delta$ values are network and processing latencies.

### Worked Example

Client deadline: 5 seconds total.

| Service | Arrival Time | Remaining | Network to Next | Processing | Available for Downstream |
|:---|:---:|:---:|:---:|:---:|:---:|
| A (gateway) | 0 ms | 5000 ms | 10 ms | 50 ms | 4940 ms |
| B (logic) | 60 ms | 4940 ms | 5 ms | 200 ms | 4735 ms |
| C (database) | 265 ms | 4735 ms | — | 100 ms | — |

### Deadline Budget Splitting

If a service fans out to $k$ parallel downstream calls:

$$\text{remaining per call} = \text{remaining} - \text{own processing}$$

All parallel calls share the same deadline (they execute concurrently).

If calls are sequential:

$$\text{budget per call} = \frac{\text{remaining} - \text{own processing}}{k}$$

With 3 sequential downstream calls and 4 seconds remaining:

$$\text{per call} = \frac{4000 - 200}{3} = 1267 \text{ ms each}$$

---

## 5. HTTP/2 Flow Control Window (Backpressure Math)

### The Problem

HTTP/2 flow control limits how much data a sender can transmit before receiving a WINDOW_UPDATE. How does this affect gRPC streaming throughput?

### The Formula

Effective throughput with flow control:

$$\Theta = \frac{W}{RTT}$$

Where $W$ is the flow control window size (default 65,535 bytes for both connection and stream level).

### Worked Examples

| Window Size $W$ | RTT | Max Throughput |
|:---:|:---:|:---:|
| 64 KB | 1 ms | 512 Mbps |
| 64 KB | 10 ms | 51.2 Mbps |
| 64 KB | 50 ms | 10.2 Mbps |
| 64 KB | 100 ms | 5.1 Mbps |
| 1 MB | 10 ms | 819 Mbps |
| 1 MB | 50 ms | 163.8 Mbps |
| 1 MB | 100 ms | 81.9 Mbps |
| 16 MB | 100 ms | 1.31 Gbps |

### Two Levels of Flow Control

HTTP/2 has both connection-level and stream-level windows:

$$\Theta_{stream} = \min\left(\frac{W_{stream}}{RTT}, \frac{W_{connection}}{RTT \times N_{streams}}\right)$$

Where $N_{streams}$ is the number of active streams sharing the connection.

With default windows (64 KB each), 10 concurrent streams, 50 ms RTT:

$$\Theta_{stream} = \min\left(\frac{65535}{0.05}, \frac{65535}{0.05 \times 10}\right) = \min(10.2 \text{ Mbps}, 1.02 \text{ Mbps})$$

Each stream is limited to 1.02 Mbps. To fix this, increase the connection window:

$$W_{connection} = N_{streams} \times W_{stream}$$

gRPC typically increases the initial connection window via SETTINGS frame.

### Window Update Frequency

WINDOW_UPDATE frames should be sent when the window is approximately half consumed:

$$\text{update interval} = \frac{W / 2}{\Theta_{target}}$$

For 64 KB window at 100 Mbps target:

$$\text{interval} = \frac{32768}{12500000} = 2.6 \text{ ms}$$

This means ~385 WINDOW_UPDATE frames per second per stream, which is non-trivial overhead for many concurrent streams.

---

## 6. Message Size Economics (Chunking vs Unary)

### The Problem

gRPC has a default max message size of 4 MB. When should you use streaming instead of large unary messages?

### The Formula

Memory consumption for unary:

$$M_{unary} = S_{message} \times 2 \quad \text{(serialized buffer + deserialized object)}$$

For server handling $C$ concurrent requests:

$$M_{total} = C \times S_{message} \times 2$$

With streaming, each chunk is processed independently:

$$M_{streaming} = C \times S_{chunk} \times 2$$

### Worked Examples

| Message Size | Concurrent Requests | Unary Memory | Streaming (64 KB chunks) |
|:---:|:---:|:---:|:---:|
| 1 MB | 100 | 200 MB | 12.5 MB |
| 10 MB | 100 | 2 GB | 12.5 MB |
| 100 MB | 10 | 2 GB | 1.25 MB |
| 1 MB | 1000 | 2 GB | 125 MB |

### Time to First Byte (TTFB)

Unary: client waits for entire response to be serialized and sent.

$$TTFB_{unary} = T_{serialize} + T_{transfer}$$

Streaming: client receives first chunk immediately.

$$TTFB_{streaming} = T_{serialize\_chunk} + T_{transfer\_chunk}$$

For a 10 MB response at 100 Mbps:

$$TTFB_{unary} = 10 + 800 = 810 \text{ ms}$$
$$TTFB_{streaming} = 0.1 + 5.1 = 5.2 \text{ ms}$$

Streaming delivers first data 155x faster.

---

## 7. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\lceil (\lfloor \log_2 n \rfloor + 1) / 7 \rceil$ | Ceiling log | Varint size |
| $(field\_num << 3) \| wire\_type$ | Bit manipulation | Tag encoding |
| $1 - S_{pb}/S_{json}$ | Ratio | Encoding efficiency |
| $D - t_i$ | Subtraction | Deadline remaining |
| $W / RTT$ | Division | Flow control throughput |
| $W_{conn} / (RTT \times N)$ | Division | Per-stream throughput |
| $C \times S \times 2$ | Product | Memory consumption |

## Prerequisites

- variable-length encoding, binary protocols, flow control, HTTP/2 framing

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Varint encode/decode | $O(1)$ (max 10 bytes) | $O(1)$ |
| Protobuf serialize | $O(N)$ fields | $O(S)$ message size |
| Protobuf deserialize | $O(N)$ fields | $O(S)$ message size |
| HPACK header encode | $O(H)$ header bytes | $O(T)$ dynamic table |
| Flow control check | $O(1)$ window compare | $O(1)$ per stream |
| Stream multiplexing | $O(1)$ stream ID lookup | $O(N)$ streams |
| Deadline propagation | $O(1)$ per hop | $O(1)$ |

---

*gRPC's performance story is told in bytes: varint encoding saves 50-75% on small integers, Protobuf halves payload size vs JSON, and HPACK compresses headers by 90%+ on warm connections. But the real win is architectural — HTTP/2 multiplexing replaces N TCP connections with N streams on one connection, eliminating the setup cost that dominates latency for small RPCs.*
