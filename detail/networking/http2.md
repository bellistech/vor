# The Internals of HTTP/2 (Binary Framing, HPACK, Flow Control & Stream Mechanics)

> *HTTP/2 replaces HTTP/1.1's textual request-response pipeline with a binary-framed multiplexing layer. Understanding the frame format, HPACK compression algorithm, stream state machine, and flow control windowing reveals why HTTP/2 achieves its performance gains -- and where TCP-level head-of-line blocking limits them.*

---

## 1. Binary Framing Layer (The Wire Format)

### The Problem

HTTP/1.1 uses ASCII text delimited by CRLF, requiring complex parsing, ambiguous message boundaries, and making multiplexing impossible without multiple TCP connections.

### The Frame Header

Every HTTP/2 frame begins with a fixed 9-byte header:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  Length (24)                   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Type (8)    |   Flags (8)   |
+-+-+-----------+---------------+-------------------------------+
|R|                 Stream Identifier (31)                       |
+-+-------------------------------------------------------------+
|                         Payload (Length bytes)                  |
+---------------------------------------------------------------+
```

**Field breakdown:**

- **Length** (24 bits): payload size, not including the 9-byte header itself. Default maximum is 16,384 bytes ($2^{14}$); negotiable up to 16,777,215 bytes ($2^{24} - 1$) via `MAX_FRAME_SIZE`.

- **Type** (8 bits): one of 10 defined frame types (0x0 through 0x9, plus 0xa for CONTINUATION).

- **Flags** (8 bits): type-specific. Common flags:
  - `END_STREAM` (0x1) -- last frame for this stream
  - `END_HEADERS` (0x4) -- header block is complete
  - `PADDED` (0x8) -- frame includes padding
  - `PRIORITY` (0x20) -- includes priority fields (HEADERS only)

- **R** (1 bit): reserved, must be zero.

- **Stream Identifier** (31 bits): identifies the stream. Stream 0 is the connection control stream.

### Total overhead per frame

$$O_{frame} = 9 \text{ bytes (header)} + P_{pad} \text{ (optional padding)}$$

For a DATA frame carrying $N$ payload bytes, the framing overhead ratio is:

$$R_{overhead} = \frac{9}{N + 9}$$

At the default `MAX_FRAME_SIZE` of 16,384:

$$R_{overhead} = \frac{9}{16{,}393} \approx 0.055\%$$

Compare with HTTP/1.1 chunked encoding, where each chunk adds a hex length line plus two CRLFs -- typically 8-12 bytes for similar-sized chunks, but with variable-length parsing complexity.

---

## 2. HPACK Header Compression (RFC 7541)

### The Problem

HTTP headers are highly repetitive across requests. In HTTP/1.1, headers like `Host`, `User-Agent`, `Accept`, and `Cookie` are sent as uncompressed ASCII with every request. Studies showed headers average 800 bytes per request, with 90%+ redundancy across requests on the same connection.

### Why Not gzip?

The CRIME attack (2012) demonstrated that using general-purpose compression (gzip/DEFLATE) on HTTP headers leaks information through the compressed size. An attacker who can inject data into requests and observe response sizes can extract secrets (like session cookies) byte-by-byte. HPACK was designed specifically to resist this attack.

### The Three Tables

**Static Table (read-only, 61 entries):**

Pre-defined header name-value pairs shared by all HTTP/2 implementations. Selected entries:

```
Index  Header Name        Header Value
──────────────────────────────────────────
1      :authority         (empty)
2      :method            GET
3      :method            POST
4      :path              /
5      :path              /index.html
6      :scheme            http
7      :scheme            https
8      :status            200
...
61     www-authenticate   (empty)
```

**Dynamic Table (per-connection, FIFO eviction):**

A bounded buffer that grows as new headers are indexed. Maximum size controlled by `SETTINGS_HEADER_TABLE_SIZE` (default 4,096 bytes).

Each entry consumes:

$$S_{entry} = len(name) + len(value) + 32$$

The 32-byte overhead accounts for the entry's metadata in the table implementation. When adding an entry would exceed the table size, the oldest entries are evicted until there is room:

$$\text{While } S_{table} + S_{new} > S_{max}: \text{ evict oldest entry}$$

**Huffman Encoding Table (static, bit-level):**

A fixed Huffman code optimized for HTTP header values. Common ASCII characters get shorter codes:

```
Character  Huffman Code     Bits
─────────────────────────────────
'0'        00000            5
'1'        00001            5
'a'        00011            5
'e'        00100            5
'o'        00110            5
'/'        00111            5
'X'        11111100         8
'~'        1111111111100    13
```

Average compression ratio for typical header values: 70-80% of original size.

### Encoding Representations

HPACK defines four encoding forms, each identified by its high-order bits:

**1. Indexed Header Field (1 bit prefix: 1xxxxxxx):**

```
  0   1   2   3   4   5   6   7
+---+---+---+---+---+---+---+---+
| 1 |        Index (7+)         |
+---+---------------------------+
```

A complete match exists in the static or dynamic table. Cost: 1 byte for indices 1-126.

**2. Literal with Incremental Indexing (2 bit prefix: 01xxxxxx):**

```
  0   1   2   3   4   5   6   7
+---+---+---+---+---+---+---+---+
| 0 | 1 |      Index (6+)       |
+---+---+-----------------------+
|  H  |   Value Length (7+)     |
+---+---------------------------+
| Value String (Length octets)   |
+-------------------------------+
```

The header is added to the dynamic table. Future occurrences can use indexed representation.

**3. Literal without Indexing (4 bit prefix: 0000xxxx):**

Same structure but the entry is NOT added to the dynamic table. Useful for headers that vary per request (e.g., `Date`, `Content-Length`).

**4. Literal Never Indexed (4 bit prefix: 0001xxxx):**

Same structure but additionally signals that this value must NEVER be compressed, even on retransmission through intermediaries. Used for sensitive values like `Authorization` and `Cookie` to resist CRIME-like attacks.

### Compression Efficiency Over Time

Let $H_i$ denote the header block size (bytes) for request $i$ on a connection. For the first request, compression comes only from static table matches and Huffman encoding. For subsequent requests:

$$C_i = \frac{|encoded_i|}{|raw_i|}$$

Typical progression on a connection sending similar requests:

| Request | Raw Size | Encoded Size | $C_i$ |
|:-------:|:--------:|:------------:|:-----:|
| 1 | 800 B | 400 B | 50% |
| 2 | 800 B | 120 B | 15% |
| 3 | 820 B | 90 B | 11% |
| 10+ | 800 B | 40-60 B | 5-8% |

After the dynamic table is populated, only fields that changed between requests need encoding. On a typical browsing session, this achieves 85-95% compression of header data.

---

## 3. Stream State Machine

### States and Transitions

HTTP/2 defines seven stream states. Every stream begins in `idle` and ends in `closed`. The transitions are deterministic based on sent/received frames:

```
                              send PP /
                             recv PP
        +--------+          +--------+
        |        |          |        |
   +--->|  idle  |          |reserved|
   |    |        |          |(local/ |
   |    +--------+          |remote) |
   |      |    |            +--------+
   |      |    | send H /        |
   |      |    | recv H          | send H /
   |      |    |                 | recv H
   |      v    v                 v
   |    +--------+          +--------+
   |    |        |-- ES --->|  half  |
   |    |  open  |          | closed |
   |    |        |<-- ES ---|  (L/R) |
   |    +--------+          +--------+
   |      |    |                 |
   |      |    | recv ES /       | recv ES /
   |      |    | send ES         | send ES
   |      v    v                 v
   |    +--------+
   +----| closed |
        +--------+

H  = HEADERS    ES = END_STREAM    PP = PUSH_PROMISE
R  = RST_STREAM (transitions any state to closed)
```

**State definitions:**

- **idle**: Initial state. Can send/receive HEADERS to transition to open. Can receive PUSH_PROMISE to transition to reserved (remote).
- **reserved (local)**: Server has sent PUSH_PROMISE. Can only send HEADERS (to open) or RST_STREAM (to closed).
- **reserved (remote)**: Client has received PUSH_PROMISE. Can only receive HEADERS (to open) or send RST_STREAM (to closed).
- **open**: Both sides can send frames. Sending/receiving END_STREAM transitions to half-closed.
- **half-closed (local)**: This side has sent END_STREAM. Can still receive frames. Receiving END_STREAM transitions to closed.
- **half-closed (remote)**: Remote side has sent END_STREAM. Can still send frames. Sending END_STREAM transitions to closed.
- **closed**: Terminal state. Only PRIORITY and WINDOW_UPDATE may be received briefly (for race conditions).

### Stream ID Space

Stream identifiers are 31-bit unsigned integers. The assignment rules are:

$$\text{Client streams: } 1, 3, 5, \ldots, 2k - 1 \quad (k \geq 1)$$
$$\text{Server push streams: } 2, 4, 6, \ldots, 2k \quad (k \geq 1)$$
$$\text{Connection control: stream } 0$$

The maximum stream ID is $2^{31} - 1 = 2{,}147{,}483{,}647$. On a connection handling 1,000 requests/second (500 client + 500 push), the ID space lasts:

$$T_{exhaust} = \frac{2^{31} - 1}{1000} \approx 2{,}147{,}483 \text{ seconds} \approx 24.8 \text{ days}$$

When the stream ID space is exhausted, both sides must close the connection (GOAWAY) and establish a new one.

---

## 4. Flow Control (Windowing)

### Design Principles

HTTP/2 flow control is hop-by-hop (not end-to-end), credit-based, and operates at two levels:

1. **Connection-level**: a single window for all DATA frames on the connection (stream 0)
2. **Per-stream**: an independent window for each stream

Only DATA frames are subject to flow control. HEADERS, SETTINGS, PING, and other control frames are always allowed.

### Window Arithmetic

Let $W$ denote the flow control window (in bytes). Initially:

$$W_{initial} = \text{INITIAL\_WINDOW\_SIZE} = 65{,}535 \text{ bytes (default)}$$

When the sender transmits a DATA frame of $n$ bytes:

$$W_{new} = W_{old} - n$$

The sender MUST NOT send DATA if $W \leq 0$. The receiver grants additional credit by sending WINDOW_UPDATE:

$$W_{new} = W_{old} + \Delta$$

Where $\Delta$ is the increment in the WINDOW_UPDATE frame ($1 \leq \Delta \leq 2^{31} - 1$).

**Overflow protection:** the window MUST NOT exceed $2^{31} - 1$:

$$\text{If } W + \Delta > 2^{31} - 1 \text{, send FLOW\_CONTROL\_ERROR}$$

### Window Sizing Strategy

The optimal receive window depends on the bandwidth-delay product, just as with TCP:

$$W_{optimal} = BDP = B \times RTT$$

For a connection carrying $S$ concurrent streams each at bandwidth $b$:

$$W_{connection} \geq S \times b \times RTT$$
$$W_{stream} \geq b \times RTT$$

**Example:** 10 concurrent streams, each aiming for 10 MB/s throughput, with 50 ms RTT:

$$W_{stream} = 10 \times 10^6 \times 0.05 = 500{,}000 \text{ bytes} \approx 488 \text{ KB}$$
$$W_{connection} = 10 \times 500{,}000 = 5{,}000{,}000 \text{ bytes} \approx 4.8 \text{ MB}$$

The default 65,535 bytes would be undersized for any single stream on this link:

$$U = \frac{65{,}535}{500{,}000} = 13.1\%$$

Servers should tune `INITIAL_WINDOW_SIZE` to match expected conditions.

### Mid-Connection SETTINGS Change

When a SETTINGS frame changes `INITIAL_WINDOW_SIZE`, ALL existing streams have their windows adjusted:

$$W_{stream,new} = W_{stream,current} + (\text{INITIAL\_WINDOW\_SIZE}_{new} - \text{INITIAL\_WINDOW\_SIZE}_{old})$$

This can produce a negative window, in which case the sender must wait for WINDOW_UPDATE frames to restore positive credit before sending DATA.

---

## 5. Prioritization (RFC 7540 and RFC 9218)

### The RFC 7540 Dependency Tree (Deprecated)

The original prioritization model organized streams into a weighted dependency tree. Each stream had:

- **Parent stream**: the stream it depends on (default: stream 0, the root)
- **Weight**: an integer 1-256 representing relative priority among siblings
- **Exclusive flag**: if set, this stream becomes the sole child of the parent, and all former siblings become children of this stream

**Bandwidth allocation among siblings:**

Given $k$ sibling streams with weights $w_1, w_2, \ldots, w_k$, the bandwidth fraction allocated to stream $i$:

$$f_i = \frac{w_i}{\sum_{j=1}^{k} w_j}$$

**Example:** Three siblings with weights 64, 128, 64:

$$f_1 = \frac{64}{256} = 25\%, \quad f_2 = \frac{128}{256} = 50\%, \quad f_3 = \frac{64}{256} = 25\%$$

### Why It Failed

The dependency tree model was underspecified, allowing divergent implementations. Studies found that major browsers, CDNs, and servers all implemented different prioritization strategies, often conflicting. Some servers ignored priorities entirely. The result was unpredictable performance that varied by client-server combination.

### RFC 9218 Extensible Priorities (Replacement)

The replacement model uses two parameters conveyed in the `Priority` header field or `PRIORITY_UPDATE` frame:

- **Urgency** ($u$): integer 0-7, default 3. Lower = higher priority.
- **Incremental** ($i$): boolean. If true, responses are interleaved (e.g., progressive images). If false, responses are delivered sequentially within their urgency level.

```
Priority: u=0         # Critical (e.g., main HTML document)
Priority: u=2         # Important (e.g., CSS, key JS)
Priority: u=3         # Default (most resources)
Priority: u=3, i      # Default, incremental (images loading progressively)
Priority: u=7         # Background (analytics, prefetch)
```

Scheduling behavior for $n$ streams at the same urgency level $u$:

- If all non-incremental: serve them sequentially (one completes before the next begins)
- If all incremental: interleave frames round-robin
- Mixed: non-incremental streams are served first, then incrementals are interleaved

---

## 6. Server Push Mechanics

### The Protocol

Server push allows the server to send responses to requests the client has not yet made. The sequence:

1. Client sends a request on stream $S_c$ (odd-numbered).
2. Server sends `PUSH_PROMISE` on stream $S_c$, reserving a new even-numbered stream $S_p$.
3. The PUSH_PROMISE contains the complete request headers the client *would have* sent.
4. Server sends `HEADERS` + `DATA` frames on stream $S_p$ with the pushed response.
5. Client receives the pushed response and caches it.

### Cache Implications

A pushed response is stored in a "push cache" associated with the connection, not the browser's HTTP cache. The lifecycle:

1. Pushed response enters the connection's push cache.
2. If the client makes a matching request before the push cache entry expires, it is consumed (moved to HTTP cache).
3. Unconsumed push cache entries are discarded when the connection closes.
4. The browser's HTTP cache validators (`ETag`, `Last-Modified`) do not apply to the push cache -- the server must know what the client does and does not have cached.

### Why Push Failed in Practice

The fundamental problem: the server cannot know the client's cache state. If the client already has `/style.css` cached, pushing it wastes bandwidth. The `Cache-Digest` proposal (to let clients declare their cache contents) was never standardized.

**Wasted bandwidth calculation:** If a server pushes $P$ resources of average size $s$ bytes, and the client already has fraction $f$ cached:

$$B_{wasted} = f \times P \times s$$

With $f = 0.8$ (typical for returning visitors), $P = 5$, and $s = 50{,}000$ bytes:

$$B_{wasted} = 0.8 \times 5 \times 50{,}000 = 200{,}000 \text{ bytes} = 195 \text{ KB wasted}$$

The replacement, `103 Early Hints`, tells the browser what to preload without sending actual data, allowing the browser to check its own cache first.

---

## 7. Head-of-Line Blocking Analysis

### HTTP/1.1: Application-Layer HOL Blocking

HTTP/1.1 pipelining allows sending multiple requests without waiting for responses, but responses MUST be returned in order. If request $R_1$ takes $T_1$ seconds and $R_2$ takes $T_2$ seconds where $T_1 \gg T_2$:

$$T_{total,pipeline} = T_1 + T_2 \quad (\text{sequential, } R_2 \text{ waits for } R_1)$$

Browsers work around this by opening $C$ parallel connections (typically $C = 6$):

$$T_{total,parallel} \approx \frac{N \times \bar{T}}{C}$$

But each connection has its own TCP handshake, TLS negotiation, and slow start, adding latency.

### HTTP/2: TCP-Layer HOL Blocking

HTTP/2 solves application-layer HOL blocking but introduces TCP-layer HOL blocking. All streams share one TCP byte stream. If a TCP segment is lost, the kernel buffers all subsequent segments until the lost one is retransmitted:

$$T_{stall} = RTT + T_{retransmit}$$

During this stall, ALL streams are blocked, even those whose data was not in the lost segment. The probability that at least one stream is affected by a single packet loss event, given $S$ active streams each with fraction $f_i$ of the connection bandwidth:

$$P(\text{at least one stream blocked}) = 1 - \prod_{i=1}^{S} (1 - f_i)$$

For $S$ equal streams, $f_i = 1/S$:

$$P = 1 - \left(1 - \frac{1}{S}\right)^S \xrightarrow{S \to \infty} 1 - \frac{1}{e} \approx 0.632$$

In practice with 10 concurrent streams, the probability that a random packet loss blocks at least one stream is:

$$P = 1 - \left(\frac{9}{10}\right)^{10} \approx 0.651$$

But the impact is worse: ALL streams stall, not just the affected one. With HTTP/1.1's 6 connections, a loss on one connection only stalls the streams on that connection.

### HTTP/3 (QUIC): No HOL Blocking

QUIC provides independent byte streams within a single UDP connection. Packet loss on stream $i$ only stalls stream $i$; other streams continue unaffected. The stall impact is:

$$\text{Streams blocked per loss event: } 1 \text{ (QUIC)} \text{ vs } S \text{ (HTTP/2)}$$

### Performance Crossover

On low-loss networks (loss rate $p < 0.1\%$), HTTP/2's single connection outperforms HTTP/1.1's multiple connections due to:

- Shared congestion state
- Better bandwidth utilization
- No connection setup overhead

On high-loss networks ($p > 1\%$), the TCP HOL blocking penalty dominates. The crossover point depends on the number of concurrent streams and RTT, but empirically:

$$p_{crossover} \approx \frac{0.1}{S \times RTT / 100}$$

For 10 streams at 100 ms RTT: $p_{crossover} \approx 0.1\% = 10^{-3}$.

---

## 8. Performance Comparison with HTTP/1.1 Pipelining

### Connection Setup Cost

HTTP/1.1 with $C$ connections requires $C$ TCP handshakes and $C$ TLS handshakes:

$$T_{setup,1.1} = C \times (RTT_{TCP} + RTT_{TLS})$$

Where $RTT_{TLS}$ is 1-2 RTTs depending on TLS version (1 RTT for TLS 1.3, 2 for TLS 1.2).

HTTP/2 requires a single connection:

$$T_{setup,2} = RTT_{TCP} + RTT_{TLS} + RTT_{preface}$$

The preface exchange (client sends connection preface + SETTINGS; server sends SETTINGS + ACK) can overlap with the TLS handshake, so in practice:

$$T_{setup,2} \approx RTT_{TCP} + RTT_{TLS}$$

**Savings:** For 6 connections at 50 ms RTT with TLS 1.2 (2 RTTs):

$$T_{setup,1.1} = 6 \times (50 + 100) = 900 \text{ ms}$$
$$T_{setup,2} = 50 + 100 = 150 \text{ ms}$$
$$\text{Savings: } 750 \text{ ms}$$

### Slow Start Impact

Each new TCP connection begins with a small congestion window (typically 10 segments = 14,600 bytes). Reaching full throughput takes $k$ RTTs:

$$W_{k} = W_{init} \times 2^{k}$$

To reach a target window $W_{target}$:

$$k = \lceil \log_2(W_{target} / W_{init}) \rceil$$

For a 1 MB window: $k = \lceil \log_2(1{,}048{,}576 / 14{,}600) \rceil = \lceil 6.17 \rceil = 7$ RTTs.

HTTP/2's single connection reaches full speed once and stays there. HTTP/1.1's 6 connections each independently go through slow start when first used.

### Header Overhead

With HTTP/1.1, $N$ requests send $N$ full header sets. With HTTP/2 and HPACK:

$$B_{headers,1.1} = N \times H_{avg}$$
$$B_{headers,2} \approx H_{avg} + (N - 1) \times H_{delta}$$

Where $H_{avg} \approx 800$ bytes and $H_{delta} \approx 40-80$ bytes (only changed fields). For 100 requests:

$$B_{headers,1.1} = 100 \times 800 = 80{,}000 \text{ bytes}$$
$$B_{headers,2} \approx 800 + 99 \times 60 = 6{,}740 \text{ bytes}$$
$$\text{Compression ratio: } \frac{6{,}740}{80{,}000} = 8.4\%$$

---

## Tips

- HPACK's security model is as important as its compression. The "never indexed" representation exists specifically to prevent intermediary proxies from compressing sensitive headers, which would re-enable CRIME-like side-channel attacks.
- The 9-byte frame header is a fixed cost. For very small payloads (e.g., a 10-byte DATA frame), framing overhead is significant: $9 / 19 = 47\%$. Aggregate small writes before framing when possible.
- Flow control windows should match the bandwidth-delay product. The default 65,535 bytes is adequate for LAN traffic but catastrophically undersized for WAN. Most production servers set `INITIAL_WINDOW_SIZE` to 1-16 MB.
- The dynamic table size is a tradeoff: larger tables compress better but consume more memory per connection. At 10,000 concurrent connections with 4 KB tables each, the server uses 40 MB just for HPACK state.
- Stream ID exhaustion at $2^{31} - 1$ requires connection replacement. For very long-lived connections (gRPC streams), implement graceful GOAWAY and reconnection logic before hitting the limit.
- When analyzing HTTP/2 performance problems, start with flow control: a zero window on stream 0 (connection level) stalls all streams, while a zero window on a specific stream only stalls that stream. Use `nghttp -v` to observe WINDOW_UPDATE patterns.

## See Also

- http, quic, tls, tcp

## References

- [RFC 9113 -- HTTP/2](https://www.rfc-editor.org/rfc/rfc9113)
- [RFC 7541 -- HPACK: Header Compression for HTTP/2](https://www.rfc-editor.org/rfc/rfc7541)
- [RFC 9218 -- Extensible Prioritization Scheme for HTTP](https://www.rfc-editor.org/rfc/rfc9218)
- [RFC 8740 -- Using TLS 1.3 with HTTP/2](https://www.rfc-editor.org/rfc/rfc8740)
- [Varvello et al., "Is the Web HTTP/2 Yet?" (PAM 2016)](https://doi.org/10.1007/978-3-319-30505-9_17)
- [Marx et al., "Same Standards, Different Decisions: A Study of QUIC and HTTP/3 Implementation Diversity" (2020)](https://doi.org/10.1145/3419394.3423643)
