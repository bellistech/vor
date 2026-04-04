# The Mathematics of WebSocket — Frame Overhead & Connection Scaling

> *WebSocket's minimal 2-byte frame header makes it vastly more efficient than HTTP polling for real-time data, but the mathematics of masking, compression ratios, and connection scaling reveal the engineering trade-offs that determine whether a system handles thousands or millions of concurrent connections.*

---

## 1. Frame Overhead Analysis (Amortized Cost)

### The Problem

Each WebSocket frame carries a header of 2-14 bytes depending on payload size and masking. For a stream of messages with size distribution $S$, what is the average overhead ratio compared to raw TCP?

### The Formula

Frame header size as a function of payload length $L$ and direction:

$$H(L, \text{masked}) = \begin{cases} 2 + 4m & \text{if } L \leq 125 \\ 4 + 4m & \text{if } 126 \leq L \leq 65535 \\ 10 + 4m & \text{if } L > 65535 \end{cases}$$

Where $m = 1$ for client-to-server (masked), $m = 0$ for server-to-client.

Overhead ratio:

$$R(L) = \frac{H(L)}{L + H(L)}$$

For a message size distribution $f(L)$, expected overhead:

$$\bar{R} = E\left[\frac{H(L)}{L + H(L)}\right] = \int_0^\infty \frac{H(l)}{l + H(l)} f(l) \, dl$$

### Worked Examples

**Example 1:** Small JSON messages ($L = 50$ bytes), client-to-server (masked):

$$H = 2 + 4 = 6 \text{ bytes}$$

$$R = \frac{6}{56} = 10.7\%$$

Compare to HTTP polling (headers ~200-500 bytes per request):

$$R_{\text{HTTP}} = \frac{350}{400} = 87.5\%$$

WebSocket is 8x more efficient for small frequent messages.

**Example 2:** Large binary frame ($L = 100000$ bytes), server-to-client:

$$H = 10 + 0 = 10 \text{ bytes}$$

$$R = \frac{10}{100010} = 0.01\%$$

For large payloads, WebSocket overhead is negligible.

---

## 2. Masking and XOR Security (Cryptography)

### The Problem

Client-to-server frames must be masked with a 32-bit key to prevent cache poisoning attacks on intermediary proxies. The masking is a simple XOR operation. What are the statistical properties of the masking?

### The Formula

Each byte of payload $p_i$ is transformed:

$$m_i = p_i \oplus k_{i \bmod 4}$$

Where $k_0, k_1, k_2, k_3$ is the 4-byte masking key chosen uniformly at random. The entropy of the masking:

$$H(\text{mask}) = 32 \text{ bits}$$

Probability of an attacker guessing the mask:

$$P(\text{guess}) = \frac{1}{2^{32}} = 2.328 \times 10^{-10}$$

XOR properties (involution):

$$m_i \oplus k_{i \bmod 4} = p_i \oplus k_{i \bmod 4} \oplus k_{i \bmod 4} = p_i$$

### Worked Examples

**Example:** Mask key = `0x37 0xFA 0x21 0x3D`, payload = "Hi" (`0x48 0x69`):

$$m_0 = 0x48 \oplus 0x37 = 0x7F$$

$$m_1 = 0x69 \oplus 0xFA = 0x93$$

Unmasking:

$$p_0 = 0x7F \oplus 0x37 = 0x48 = \text{'H'}$$

$$p_1 = 0x93 \oplus 0xFA = 0x69 = \text{'i'}$$

The masking is not encryption — it prevents proxy cache poisoning by making the wire format unpredictable, not confidential.

---

## 3. Connection Scaling and Memory (Resource Modeling)

### The Problem

A WebSocket server must maintain state for each connection: TCP buffers, application state, and optionally compression context. What is the memory required for $N$ concurrent connections?

### The Formula

Per-connection memory:

$$M_{\text{conn}} = M_{\text{tcp}} + M_{\text{recv}} + M_{\text{send}} + M_{\text{app}} + M_{\text{deflate}}$$

Where:
- $M_{\text{tcp}} \approx 3.5$ KB (kernel socket buffers, minimal)
- $M_{\text{recv}}, M_{\text{send}}$ = read/write buffers (typically 4-16 KB each)
- $M_{\text{app}}$ = application state per connection
- $M_{\text{deflate}} = 2^w$ bytes per direction ($w$ = window bits, 0 if no compression)

Total server memory:

$$M_{\text{total}} = N \cdot M_{\text{conn}} + M_{\text{base}}$$

With context takeover disabled, $M_{\text{deflate}} = 0$ (allocated per message, freed after).

### Worked Examples

**Example 1:** 100K connections with compression (15-bit window), 8 KB buffers:

$$M_{\text{conn}} = 3.5 + 8 + 8 + 1 + 2 \times 32 = 84.5 \text{ KB}$$

$$M_{\text{total}} = 100000 \times 84.5 = 8.45 \text{ GB}$$

**Example 2:** 100K connections without compression, 4 KB buffers:

$$M_{\text{conn}} = 3.5 + 4 + 4 + 1 + 0 = 12.5 \text{ KB}$$

$$M_{\text{total}} = 100000 \times 12.5 = 1.25 \text{ GB}$$

Disabling compression context saves 6.4 GB for 100K connections — critical for high-connection-count servers.

---

## 4. permessage-deflate Compression Ratio (Information Theory)

### The Problem

The permessage-deflate extension uses LZ77 + Huffman coding. For messages with entropy $H$ bits per byte, what is the expected compression ratio?

### The Formula

The theoretical lower bound on compressed size (Shannon's source coding theorem):

$$C_{\min} = \frac{H(X)}{8} \cdot L$$

Where $H(X)$ is the entropy in bits per byte and $L$ is the original message length. Compression ratio:

$$r = \frac{C_{\text{actual}}}{L}$$

For DEFLATE with window size $W = 2^w$ bytes, the effective compression approaches the entropy bound when:

$$L \gg W \quad \text{and} \quad \text{source is stationary}$$

With context takeover, successive messages share the dictionary:

$$r_{\text{ctx}} \leq r_{\text{no-ctx}}$$

The improvement from context takeover for messages of size $L$ with inter-message redundancy $\rho$:

$$r_{\text{ctx}} \approx r_{\text{no-ctx}} \cdot (1 - \rho)$$

### Worked Examples

**Example:** JSON messages averaging 200 bytes, entropy $H = 4.5$ bits/byte:

$$r_{\text{theory}} = \frac{4.5}{8} = 0.5625$$

Practical DEFLATE typically achieves $r \approx 0.4\text{--}0.6$ for JSON. For 200-byte messages:

$$C_{\text{no-ctx}} \approx 200 \times 0.55 = 110 \text{ bytes}$$

With context takeover and $\rho = 0.3$ inter-message redundancy (repeated keys):

$$C_{\text{ctx}} \approx 110 \times 0.7 = 77 \text{ bytes}$$

Bandwidth savings over 1M messages/sec:

$$\Delta B = 10^6 \times (200 - 77) = 123 \text{ MB/sec} = 984 \text{ Mbps}$$

---

## 5. Heartbeat Interval and Failure Detection (Timeout Theory)

### The Problem

Ping/pong heartbeats detect dead connections. If the network has jitter with standard deviation $\sigma_j$ and the heartbeat interval is $T$, what timeout $T_o$ minimizes false positives while detecting failures quickly?

### The Formula

Round-trip time follows $\text{RTT} \sim N(\mu_r, \sigma_j^2)$. A pong is "late" if:

$$\text{RTT} > T_o$$

False positive rate (healthy connection declared dead):

$$P(\text{false positive}) = 1 - \Phi\left(\frac{T_o - \mu_r}{\sigma_j}\right)$$

Setting $T_o = \mu_r + k\sigma_j$ for $k$ standard deviations:

| $k$ | $P(\text{false positive})$ |
|-----|---------------------------|
| 2 | 2.28% |
| 3 | 0.13% |
| 4 | 0.003% |

Mean time to detect a true failure:

$$E[T_{\text{detect}}] = T + \frac{T_o}{2}$$

### Worked Examples

**Example:** $\mu_r = 50$ ms, $\sigma_j = 20$ ms, heartbeat $T = 30$ sec:

For $k = 3$: $T_o = 50 + 60 = 110$ ms. False positive rate per ping: 0.13%.

Over 24 hours ($24 \times 3600 / 30 = 2880$ pings), expected false disconnections:

$$E[\text{false disc}] = 2880 \times 0.0013 = 3.7$$

Too high. Use $k = 4$: $T_o = 130$ ms, $P = 0.003\%$:

$$E[\text{false disc}] = 2880 \times 0.00003 = 0.09$$

Less than 1 false disconnection per day. Detection time:

$$E[T_{\text{detect}}] = 30 + 0.065 = 30.065 \text{ sec}$$

---

## 6. Polling vs WebSocket: Break-Even Analysis (Cost Optimization)

### The Problem

When does WebSocket become more efficient than HTTP polling? If messages arrive at rate $\lambda$ per second and the HTTP polling interval is $T_p$, at what $\lambda$ does WebSocket win on bandwidth?

### The Formula

HTTP polling bandwidth (regardless of message arrival):

$$B_{\text{poll}} = \frac{H_{\text{req}} + H_{\text{resp}}}{T_p} + \lambda \cdot (H_{\text{resp}} + L)$$

WebSocket bandwidth:

$$B_{\text{ws}} = \lambda \cdot (H_{\text{ws}} + L) + \frac{H_{\text{ping}}}{T_{\text{hb}}}$$

Break-even when $B_{\text{poll}} = B_{\text{ws}}$:

$$\lambda_{\text{break}} = \frac{(H_{\text{req}} + H_{\text{resp}}) / T_p}{H_{\text{resp}} - H_{\text{ws}} + H_{\text{ping}} / (T_{\text{hb}} \cdot \lambda)}$$

### Worked Examples

**Example:** $H_{\text{req}} = 300$ bytes, $H_{\text{resp}} = 200$ bytes, $H_{\text{ws}} = 6$ bytes, $T_p = 1$ sec, $T_{\text{hb}} = 30$ sec, $L = 100$ bytes:

$$B_{\text{poll}} = 500 + \lambda \times 300 \text{ bytes/sec}$$

$$B_{\text{ws}} = \lambda \times 106 + 2 \text{ bytes/sec}$$

$$500 + 300\lambda = 106\lambda + 2$$

$$194\lambda = -498 \implies \lambda \approx 0$$

WebSocket wins at any message rate when HTTP polls every second. At $T_p = 60$ sec:

$$8.33 + 300\lambda = 106\lambda + 2$$

$$194\lambda = -6.33 \implies \lambda \approx 0$$

WebSocket is almost always more bandwidth-efficient — the crossover is essentially at zero messages.

## Prerequisites

- Information theory (Shannon entropy, source coding theorem)
- Probability (normal distribution, false positive rates)
- Cryptography basics (XOR properties, cache poisoning)
- Resource modeling (memory estimation, connection state)
- Amortized analysis (overhead per operation)
