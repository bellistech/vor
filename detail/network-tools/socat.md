# The Mathematics of socat — Bidirectional Byte Stream Relay

> *socat establishes bidirectional data channels between two endpoints. The math covers relay throughput, buffer sizing, timeout management, and the address type combination space.*

---

## 1. Relay Model — Bidirectional Data Flow

### The Model

socat connects two address endpoints (A and B), relaying data bidirectionally:

$$A \xleftrightarrow{\text{socat}} B$$

### Throughput

$$\text{Throughput} = \min(\text{BW}_A, \text{BW}_B, \text{socat Processing})$$

socat processes data through a read-write loop:

$$T_{relay} = T_{read\_A} + T_{process} + T_{write\_B}$$

### Processing Overhead

$$\text{socat CPU overhead} \approx 5-15\% \quad (\text{userspace copy: read() + write()})$$

| Data Path | Without socat | With socat | Overhead |
|:---|:---:|:---:|:---:|
| TCP → TCP | Kernel forwarding | Read + Write | ~10% |
| UNIX socket → TCP | Direct | Read + Write | ~5% |
| PTY → TCP | Direct | Read + Write + PTY processing | ~15% |
| SSL → TCP | SSL library | Read + Decrypt + Write | ~20% |

---

## 2. Address Type Combinations

### The Model

socat supports ~20 address types. Any two can be combined.

### Combination Space

$$\text{Possible Combinations} = n^2 \quad (\text{any address type for each endpoint})$$

$$20^2 = 400 \text{ possible pairings}$$

### Common Address Types

| Address Type | Abbreviation | Examples |
|:---|:---|:---|
| TCP client | `TCP:host:port` | TCP connection |
| TCP listener | `TCP-LISTEN:port` | TCP server |
| UDP | `UDP:host:port` | UDP datagram |
| UNIX socket | `UNIX-CONNECT:path` | Unix domain socket |
| STDIN/STDOUT | `STDIO` | Terminal I/O |
| File | `FILE:path` | Regular file |
| PTY | `PTY` | Pseudo-terminal |
| Pipe | `PIPE:name` | Named pipe |
| SSL/TLS | `SSL:host:port` | Encrypted TCP |
| EXEC | `EXEC:command` | Subprocess |
| SYSTEM | `SYSTEM:command` | Shell command |

### Practical Combinations

| Use Case | socat Command Pattern | Throughput |
|:---|:---|:---:|
| TCP port forwarder | TCP-LISTEN ↔ TCP | Line rate |
| UNIX to TCP bridge | UNIX-LISTEN ↔ TCP | Line rate |
| SSL termination | SSL-LISTEN ↔ TCP | Cipher-limited |
| Serial to TCP | /dev/ttyUSB0 ↔ TCP | Baud-limited |
| File transfer | STDIO ↔ TCP | Network-limited |
| Virtual serial port | PTY ↔ PTY | Memory speed |

---

## 3. Buffer Sizing

### The Model

socat uses configurable buffers for read/write operations.

### Buffer Impact

$$\text{Reads per Transfer} = \lceil \frac{\text{Data Size}}{\text{Buffer Size}} \rceil$$

$$\text{Syscalls} = 2 \times \text{Reads} \quad (\text{one read + one write per buffer})$$

### Default Buffer

$$\text{Default Buffer} = 8192 \text{ bytes}$$

### Buffer Size Trade-offs

| Buffer Size | Syscalls per MiB | Throughput | Latency |
|:---:|:---:|:---:|:---:|
| 512 bytes | 2,048 | Low | Lowest |
| 4 KiB | 256 | Medium | Low |
| 8 KiB (default) | 128 | Good | Medium |
| 64 KiB | 16 | High | Higher |
| 1 MiB | 1 | Highest | Highest |

### Optimal Buffer for Streaming

$$\text{Optimal} = \min(\text{BDP}, \text{Available Memory})$$

For a 100 Mbps link with 10 ms RTT:

$$\text{BDP} = \frac{100 \times 10^6}{8} \times 0.010 = 125 \text{ KiB}$$

---

## 4. Timeout and Keepalive Math

### Timeout Parameters

| Option | Default | Effect |
|:---|:---:|:---|
| `-T timeout` | - | Inactivity timeout (total) |
| `-t timeout` | 0.5s | Shutdown timeout after EOF |
| `connect-timeout` | OS default | TCP connection timeout |
| `keepalive` | Off | TCP keepalive |

### Inactivity Detection

$$\text{Connection Idle} = T_{last\_data} > T_{timeout}$$

### Keepalive Math

$$T_{keepalive\_detect} = \text{keepidle} + \text{keepintvl} \times \text{keepcnt}$$

Default Linux: $7200 + 75 \times 9 = 7875 \text{ sec} \approx 2.2 \text{ hours}$.

Tuned for socat relay:

$$T_{detect} = 60 + 10 \times 3 = 90 \text{ sec}$$

---

## 5. Fork Model — Concurrent Connections

### The Model

`socat TCP-LISTEN:port,fork` forks a child process per connection.

### Resource Usage

$$\text{Memory} = \text{Connections} \times \text{Process Size (socat ~2 MiB)}$$

$$\text{Max Connections} = \min(\text{Max Processes}, \frac{\text{Available Memory}}{\text{Process Size}}, \text{FD Limit})$$

| Connections | Memory (fork) | CPU Overhead |
|:---:|:---:|:---:|
| 10 | 20 MiB | Negligible |
| 100 | 200 MiB | Low |
| 1,000 | 2 GiB | Medium |
| 10,000 | 20 GiB | High (fork overhead) |

### Fork vs Event-Driven

$$\text{Fork (socat):} \quad O(n) \text{ processes, } O(1) \text{ per connection}$$

$$\text{Event-driven (nginx):} \quad O(1) \text{ processes, } O(n) \text{ connections per process}$$

For >1,000 concurrent connections, socat's fork model becomes impractical.

---

## 6. SSL/TLS Bridging

### The Model

socat can terminate or originate TLS connections.

### TLS Overhead

$$T_{tls\_setup} = T_{tcp} + T_{handshake}$$

$$\text{Data Overhead} = \frac{\text{TLS Record Header (5 bytes)} + \text{MAC (16-32 bytes)}}{\text{Payload}} \times 100\%$$

For 8 KiB payload:

$$\text{Overhead} = \frac{5 + 32}{8192} = 0.45\%$$

### TLS Bridge Throughput

$$\text{Throughput} = \min(\text{Cipher Speed}, \text{Network BW}, \text{socat Buffer Rate})$$

| Operation | Throughput |
|:---|:---:|
| TCP → TCP (plain) | ~10 Gbps |
| TCP → SSL (encrypt) | ~2-5 Gbps |
| SSL → TCP (decrypt) | ~2-5 Gbps |
| SSL → SSL (re-encrypt) | ~1-3 Gbps |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\min(\text{BW}_A, \text{BW}_B)$ | Min function | Relay throughput |
| $n^2$ | Quadratic | Address combinations |
| $\lceil \frac{\text{Data}}{\text{Buffer}} \rceil \times 2$ | Ceiling + multiplication | Syscall count |
| $\text{Conns} \times 2\text{MiB}$ | Linear | Fork memory |
| $\frac{\text{Header}}{\text{Payload}}$ | Ratio | TLS overhead |
| $\text{idle} + \text{intvl} \times \text{cnt}$ | Linear | Keepalive detection |

---

*Every `socat TCP-LISTEN:8080,fork TCP:backend:80` creates a userspace relay — connecting any two address types with a simple read-write loop that trades the overhead of process-per-connection for the flexibility of 400+ address combinations.*

## Prerequisites

- Socket programming (TCP, UDP, Unix domain sockets)
- File descriptor and bidirectional I/O concepts
- Process forking model for concurrent connections

## Complexity

- **Beginner:** TCP relay, port forwarding, stdin/stdout piping, simple listeners
- **Intermediate:** TLS wrapping, Unix socket bridging, UDP relaying, timeout and retry options
- **Advanced:** Address type combinatorics (400+ pairs), fork-per-connection overhead, buffer sizing, throughput analysis for userspace relay
