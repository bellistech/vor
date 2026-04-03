# The Mathematics of HAProxy — Load Balancer Internals

> *HAProxy is a high-performance TCP/HTTP load balancer. The math covers load balancing algorithms, health check timing, connection queuing (Little's Law), stick table sizing, and maxconn capacity planning.*

---

## 1. Load Balancing Algorithms

### Round-Robin

$$\text{Server}_i = i \mod n \quad (\text{request number } i, n \text{ servers})$$

$$\text{Requests per Server} = \frac{\text{Total Requests}}{n}$$

### Weighted Round-Robin

$$P(\text{server}_i) = \frac{w_i}{\sum_{j=1}^{n} w_j}$$

### Least Connections

$$\text{Select} = \arg\min_i \text{active\_connections}_i$$

With weights:

$$\text{Select} = \arg\min_i \frac{\text{active\_connections}_i}{w_i}$$

### Source Hash (Session Persistence)

$$\text{Server} = \text{hash}(\text{src\_ip}) \mod \text{total\_weight}$$

### URI Hash

$$\text{Server} = \text{hash}(\text{URI}) \mod \text{total\_weight}$$

### Algorithm Comparison

| Algorithm | Balanced | Sticky | Stateful | Best For |
|:---|:---:|:---:|:---:|:---|
| roundrobin | Equal | No | No | Stateless APIs |
| leastconn | By load | No | Yes | Long connections (DB, WS) |
| source | No | Yes | Yes | Session affinity |
| uri | No | Yes | Yes | Cache distribution |
| random | Statistical | No | No | Large server pools |
| first | No | No | No | Server consolidation |

---

## 2. Health Check Timing

### The Model

HAProxy's health checks use three parameters: `inter` (check interval), `fall` (failures to mark down), `rise` (successes to mark up).

### Detection Time

$$T_{failure\_detection} = \text{fall} \times \text{inter} + T_{timeout}$$

$$T_{recovery\_detection} = \text{rise} \times \text{inter}$$

### Worked Examples

```
server web1 10.0.0.1:80 check inter 2000 fall 3 rise 2
```

$$T_{failure} = 3 \times 2\text{s} + 2\text{s (timeout)} = 8\text{s}$$

$$T_{recovery} = 2 \times 2\text{s} = 4\text{s}$$

| inter | fall | rise | Failure Detection | Recovery Detection |
|:---:|:---:|:---:|:---:|:---:|
| 1s | 2 | 1 | 3s | 1s |
| 2s | 3 | 2 | 8s | 4s |
| 5s | 3 | 2 | 17s | 10s |
| 10s | 5 | 3 | 60s | 30s |

### Health Check Bandwidth

$$\text{Check Traffic} = \text{Servers} \times \frac{1}{\text{inter}} \times \text{Check Packet Size}$$

| Servers | inter | Checks/sec | Bandwidth (1K pkt) |
|:---:|:---:|:---:|:---:|
| 10 | 2s | 5 | 5 KiB/s |
| 100 | 2s | 50 | 50 KiB/s |
| 1,000 | 5s | 200 | 200 KiB/s |

---

## 3. Connection Queuing — Little's Law

### The Model

When all servers reach `maxconn`, new connections queue. Little's Law governs queue behavior.

### Little's Law

$$L = \lambda \times W$$

Where:
- $L$ = average queue depth
- $\lambda$ = arrival rate (connections/sec)
- $W$ = average wait time in queue

### Server Capacity

$$\text{Server Throughput} = \frac{\text{maxconn}}{T_{avg\_response}}$$

$$\text{Backend Capacity} = \sum_{i=1}^{n} \frac{\text{maxconn}_i}{T_{avg}}$$

### Worked Example

*"3 servers, maxconn=100 each, avg response 50ms. Arrival rate 5000 req/s."*

$$\text{Capacity} = 3 \times \frac{100}{0.050} = 6,000 \text{ req/s}$$

$$\text{Utilization} = \frac{5,000}{6,000} = 83.3\%$$

At 83% utilization (M/M/c queue model), average wait time:

$$W \approx \frac{T_{service}}{c(1 - \rho)} \times P_{queue}$$

Where $\rho = 0.833$, $c = 300$ (total maxconn), $P_{queue}$ = probability of queuing.

### Queue Timeout

```
timeout queue 10s
```

$$\text{Requests Dropped} = \lambda \times P(W > 10\text{s})$$

| Utilization | Avg Wait | P(Wait > 10s) |
|:---:|:---:|:---:|
| 50% | ~0 ms | ~0% |
| 80% | 5 ms | ~0% |
| 90% | 50 ms | ~0% |
| 95% | 500 ms | ~0.1% |
| 99% | 5,000 ms | ~10% |

---

## 4. Maxconn Capacity Planning

### Global maxconn

$$\text{Memory} = \text{maxconn} \times \text{Per-Connection Memory}$$

Per-connection memory depends on buffers:

$$\text{Per-Connection} = 2 \times \text{tune.bufsize (16 KiB default)} + \text{Session Overhead (~2 KiB)}$$

$$\text{Per-Connection} \approx 34 \text{ KiB}$$

### Worked Examples

| maxconn | Memory | Concurrent Users |
|:---:|:---:|:---:|
| 1,000 | 33 MiB | 1,000 |
| 10,000 | 332 MiB | 10,000 |
| 100,000 | 3.2 GiB | 100,000 |
| 1,000,000 | 32.4 GiB | 1,000,000 |

### File Descriptor Limits

$$\text{FDs Required} = 2 \times \text{maxconn} + \text{Listeners} + \text{Log FDs} + \text{Health Checks}$$

---

## 5. Stick Tables — Session Persistence

### The Model

Stick tables store session data (source IP, cookie, etc.) in memory for persistence decisions.

### Memory Formula

$$\text{Stick Table Memory} = \text{Entries} \times (\text{Key Size} + \text{Data Size} + \text{Overhead})$$

### Entry Sizes

| Key Type | Key Size | Data (counters) | Overhead | Per Entry |
|:---|:---:|:---:|:---:|:---:|
| ip (IPv4) | 4 bytes | 0 | ~50 bytes | ~54 bytes |
| ip (IPv6) | 16 bytes | 0 | ~50 bytes | ~66 bytes |
| string (32) | 32 bytes | 0 | ~50 bytes | ~82 bytes |
| ip + counters | 4 bytes | 40 bytes | ~50 bytes | ~94 bytes |

### Worked Examples

| Entries | Per Entry | Total Memory |
|:---:|:---:|:---:|
| 10,000 | 54 bytes | 527 KiB |
| 100,000 | 54 bytes | 5.1 MiB |
| 1,000,000 | 94 bytes | 89.6 MiB |
| 10,000,000 | 94 bytes | 896 MiB |

### Stick Table Expiry

$$\text{Active Entries} = \min(\lambda \times \text{TTL}, \text{Table Size Limit})$$

Where $\lambda$ = unique clients/sec, TTL = entry expire time.

---

## 6. SSL/TLS Termination Cost

### Handshake CPU Cost

$$\text{TLS Handshakes/sec} = \frac{\text{CPU Cores} \times \text{Handshakes per Core}}{\text{Connection Reuse Ratio}}$$

| Key Type | Handshakes/Core/sec | 8-Core Server |
|:---|:---:|:---:|
| RSA 2048 | ~1,500 | 12,000 |
| RSA 4096 | ~300 | 2,400 |
| ECDSA P-256 | ~10,000 | 80,000 |
| ECDSA P-384 | ~3,000 | 24,000 |

### Session Resumption

$$\text{Full Handshakes} = \text{Total Connections} \times (1 - \text{Session Reuse Rate})$$

| Reuse Rate | Full Handshakes (10K conn/s) | CPU Savings |
|:---:|:---:|:---:|
| 0% | 10,000/s | None |
| 50% | 5,000/s | 50% |
| 80% | 2,000/s | 80% |
| 95% | 500/s | 95% |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $i \mod n$ | Modular arithmetic | Round-robin selection |
| $\frac{w_i}{\sum w_j}$ | Weighted probability | Load distribution |
| $L = \lambda W$ | Little's Law | Queue depth |
| $\text{fall} \times \text{inter}$ | Linear | Failure detection time |
| $\text{maxconn} \times 34\text{K}$ | Linear scaling | Memory sizing |
| $\frac{\text{maxconn}}{T_{response}}$ | Rate equation | Server throughput |

---

*Every `show stat`, `show info`, and `show servers state` on the HAProxy stats socket reflects these calculations — a load balancer where Little's Law and connection queuing determine whether your backend drowns or thrives.*
