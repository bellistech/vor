# The Mathematics of nginx — Event-Driven Server Internals

> *nginx uses an event-driven, non-blocking architecture. The math covers the epoll event loop, worker process scaling, upstream load distribution, connection limits, and buffer sizing.*

---

## 1. Event Loop — epoll vs select

### The Model

nginx uses OS-level event notification (epoll on Linux, kqueue on BSD) to handle thousands of connections per worker.

### Complexity Comparison

| Mechanism | Add/Remove | Wait for Events | Active Connections Check |
|:---|:---:|:---:|:---:|
| `select` | O(1) | O(n) | O(n) per call |
| `poll` | O(1) | O(n) | O(n) per call |
| `epoll` | O(1) | O(1) per event | O(k) where k = ready events |
| `kqueue` | O(1) | O(1) per event | O(k) where k = ready events |

### Why epoll Wins at Scale

For $n$ total connections and $k$ active connections:

$$T_{select} = O(n) \quad \text{(checks every fd)}$$

$$T_{epoll} = O(k) \quad \text{(only reports ready fds)}$$

| Total Connections | Active (1%) | select Cost | epoll Cost | Speedup |
|:---:|:---:|:---:|:---:|:---:|
| 1,000 | 10 | 1,000 ops | 10 ops | 100x |
| 10,000 | 100 | 10,000 ops | 100 ops | 100x |
| 100,000 | 1,000 | 100,000 ops | 1,000 ops | 100x |
| 1,000,000 | 10,000 | 1,000,000 ops | 10,000 ops | 100x |

---

## 2. Worker Process Scaling

### The Formula

$$\text{worker\_processes} = \text{CPU Cores}$$

$$\text{Total Max Connections} = \text{worker\_processes} \times \text{worker\_connections}$$

### Worked Examples

| CPU Cores | worker_connections | Max Simultaneous Connections |
|:---:|:---:|:---:|
| 1 | 1,024 | 1,024 |
| 4 | 1,024 | 4,096 |
| 8 | 4,096 | 32,768 |
| 16 | 8,192 | 131,072 |
| 32 | 16,384 | 524,288 |

### File Descriptor Requirement

Each connection uses a file descriptor. As a proxy, each client connection creates a backend connection:

$$\text{FDs per proxied connection} = 2 \quad (\text{client fd + upstream fd})$$

$$\text{Total FDs} = 2 \times \text{Connections} + \text{Log FDs} + \text{Listen Sockets}$$

$$\text{worker\_rlimit\_nofile} \geq 2 \times \text{worker\_connections}$$

---

## 3. Upstream Load Balancing — Weight Distribution

### Weighted Round-Robin

$$P(\text{server}_i) = \frac{w_i}{\sum_{j=1}^{n} w_j}$$

$$\text{Requests to server}_i \text{ per cycle} = w_i$$

### Worked Example

```nginx
upstream backend {
    server 10.0.0.1 weight=5;
    server 10.0.0.2 weight=3;
    server 10.0.0.3 weight=2;
}
```

$$\text{Total weight} = 5 + 3 + 2 = 10$$

| Server | Weight | Request Share | Per 1000 Requests |
|:---|:---:|:---:|:---:|
| 10.0.0.1 | 5 | 50% | 500 |
| 10.0.0.2 | 3 | 30% | 300 |
| 10.0.0.3 | 2 | 20% | 200 |

### Least Connections

$$\text{Select} = \arg\min_i \frac{\text{active\_connections}_i}{w_i}$$

### IP Hash (Session Affinity)

$$\text{Server} = \text{hash}(\text{client\_ip}) \mod n$$

Consistent hashing variant used to minimize redistribution when servers change:

$$\text{Disruption} = \frac{1}{n} \quad (\text{adding 1 server moves } \frac{1}{n} \text{ of requests})$$

---

## 4. Request Rate and Limiting

### rate= Configuration

$$\text{rate} = r \text{ requests/second}$$

$$\text{Burst tokens} = b$$

$$\text{Token refill rate} = r \text{ per second}$$

### Leaky Bucket Algorithm

$$\text{Allowed} = \begin{cases} \text{Yes} & \text{if tokens} > 0 \\ \text{Delayed (nodelay=off)} & \text{if burst available} \\ \text{Rejected (503)} & \text{if burst exceeded} \end{cases}$$

### Worked Example

```nginx
limit_req zone=api rate=10r/s burst=20 nodelay;
```

- Sustained: 10 req/s
- Burst: up to 30 req/s (10 sustained + 20 burst) momentarily
- Recovery: burst refills at 10 tokens/sec

$$T_{burst\_recovery} = \frac{b}{r} = \frac{20}{10} = 2 \text{ seconds}$$

---

## 5. Buffer Sizing — Memory Trade-offs

### Proxy Buffer Model

$$\text{Memory per Connection} = \text{proxy\_buffer\_size} + \text{proxy\_buffers (count × size)}$$

Default: `proxy_buffer_size 4k; proxy_buffers 8 4k;`

$$\text{Per Connection} = 4\text{K} + 8 \times 4\text{K} = 36\text{K}$$

### Total Buffer Memory

$$\text{Total Buffers} = \text{Active Connections} \times \text{Memory per Connection}$$

| Connections | Buffer/Conn | Total Buffer Memory |
|:---:|:---:|:---:|
| 1,000 | 36 KiB | 35 MiB |
| 10,000 | 36 KiB | 352 MiB |
| 100,000 | 36 KiB | 3.4 GiB |
| 10,000 | 128 KiB | 1.2 GiB |

### Proxy Buffering Off

$$\text{Memory (buffering off)} = \text{proxy\_buffer\_size only} = 4\text{K per connection}$$

Saves ~90% memory but increases upstream connection hold time.

### Client Body Buffer

$$\text{client\_body\_buffer\_size} = 8\text{K (default for 32-bit), 16K (64-bit)}$$

If request body > buffer, it's written to disk:

$$T_{disk\_body} = \frac{\text{Body Size}}{\text{Disk Write Speed}} + T_{seek}$$

---

## 6. Connection Timeouts — Resource Lifecycle

### Timeout Model

$$\text{Connection Lifetime} = T_{accept} + T_{read\_request} + T_{upstream} + T_{send\_response} + T_{keepalive}$$

| Directive | Default | Purpose |
|:---|:---:|:---|
| `client_header_timeout` | 60s | Wait for request headers |
| `client_body_timeout` | 60s | Wait for request body |
| `proxy_connect_timeout` | 60s | Connect to upstream |
| `proxy_read_timeout` | 60s | Wait for upstream response |
| `keepalive_timeout` | 75s | Keep-alive idle time |
| `send_timeout` | 60s | Send response to client |

### Keepalive Connection Savings

$$\text{Connections without keepalive} = \text{Requests/sec} \times T_{connection\_setup}$$

$$\text{Connections with keepalive} = \frac{\text{Requests/sec}}{\text{Requests per connection}}$$

| Requests/sec | Without Keepalive | With Keepalive (10 req/conn) | Savings |
|:---:|:---:|:---:|:---:|
| 1,000 | 1,000 conn/s | 100 conn/s | 90% |
| 10,000 | 10,000 conn/s | 1,000 conn/s | 90% |
| 100,000 | 100,000 conn/s | 10,000 conn/s | 90% |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| O(k) vs O(n) | Algorithmic | epoll vs select |
| $\text{Cores} \times \text{worker\_conn}$ | Product | Max connections |
| $\frac{w_i}{\sum w_j}$ | Weighted probability | Load distribution |
| $\frac{b}{r}$ | Rate equation | Burst recovery |
| $\text{Conns} \times \text{Buf/Conn}$ | Linear scaling | Buffer memory |
| $\frac{\text{RPS}}{\text{Req/Conn}}$ | Division | Keepalive savings |

---

*Every `nginx -s reload`, `stub_status`, and access log line reflects these event loop calculations — a server that handles 10,000+ concurrent connections per worker process through non-blocking I/O and efficient memory management.*

## Prerequisites

- TCP/IP networking (sockets, ports, connections)
- HTTP protocol basics (methods, headers, status codes)
- Linux process model (workers, file descriptors, signals)
- Event-driven I/O concepts (epoll, non-blocking sockets)

## Complexity

- **Beginner:** Static file serving, basic proxy_pass, server blocks
- **Intermediate:** Load balancing algorithms, SSL termination, rate limiting, caching
- **Advanced:** epoll event loop internals, worker_connections tuning, upstream connection pooling, buffer sizing math
