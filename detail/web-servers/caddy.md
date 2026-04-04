# The Mathematics of Caddy — Automatic HTTPS Server Internals

> *Caddy is a modern web server with automatic HTTPS. The math covers ACME certificate lifecycle, HTTP/2 multiplexing, reverse proxy load balancing, and rate limiting internals.*

---

## 1. ACME Certificate Lifecycle — Automatic HTTPS

### The Model

Caddy automatically obtains and renews TLS certificates via ACME (Let's Encrypt). The renewal process follows a predictable timeline.

### Certificate Timeline

$$\text{Certificate Validity} = 90 \text{ days (Let's Encrypt default)}$$

$$\text{Renewal Window Start} = \text{Expiry} - 30 \text{ days (Caddy default)}$$

$$\text{Renewal Attempts} = \frac{\text{Renewal Window}}{\text{Retry Interval}}$$

### Renewal Timeline

| Day | Event |
|:---:|:---|
| 0 | Certificate issued |
| 60 | Renewal window opens (30 days before expiry) |
| 60-89 | Caddy attempts renewal (with backoff) |
| 90 | Certificate expires (if renewal fails) |

### ACME Challenge Types

| Challenge | Ports Required | Validation Time | Automation |
|:---|:---:|:---:|:---|
| HTTP-01 | 80 | 1-5 sec | Fully automatic |
| TLS-ALPN-01 | 443 | 1-5 sec | Fully automatic |
| DNS-01 | None | 30-120 sec | Needs DNS provider API |

### Certificate Storage Math

$$\text{Cert Storage} = \text{Domains} \times (\text{Cert Size} + \text{Key Size} + \text{Metadata})$$

$$\approx \text{Domains} \times (4 \text{ KiB} + 2 \text{ KiB} + 1 \text{ KiB}) = 7 \text{ KiB per domain}$$

| Domains | Storage |
|:---:|:---:|
| 10 | 70 KiB |
| 100 | 700 KiB |
| 1,000 | 6.8 MiB |
| 10,000 | 68 MiB |

### Let's Encrypt Rate Limits

| Limit | Value | Reset |
|:---|:---:|:---|
| Certificates per domain | 50/week | Rolling 7-day |
| New orders per account | 300/3 hours | Rolling 3-hour |
| Failed validations | 5/hour/hostname | Rolling 1-hour |
| Duplicate certificates | 5/week | Rolling 7-day |

---

## 2. HTTP/2 and HTTP/3 Multiplexing

### The Model

Caddy supports HTTP/2 (default) and HTTP/3 (QUIC). Multiplexing eliminates head-of-line blocking at the HTTP layer.

### HTTP/2 Stream Parallelism

$$\text{Concurrent Streams} = \min(\text{client max}, \text{server max}, 100 \text{ default})$$

$$\text{Throughput} = \frac{\text{Concurrent Streams} \times \text{Avg Response Size}}{T_{avg\_response}}$$

### HTTP/2 vs HTTP/1.1

$$\text{HTTP/1.1 Resources} = \frac{\text{Resources}}{6} \text{ rounds (6 connections max)}$$

$$\text{HTTP/2 Resources} = 1 \text{ round (all multiplexed)}$$

| Resources | HTTP/1.1 Rounds | HTTP/2 Rounds | Speedup |
|:---:|:---:|:---:|:---:|
| 6 | 1 | 1 | 1x |
| 12 | 2 | 1 | 2x |
| 30 | 5 | 1 | 5x |
| 60 | 10 | 1 | 10x |

### HTTP/3 (QUIC) Advantages

$$\text{Connection Setup} = \begin{cases} 3 \times \text{RTT} & \text{HTTP/1.1 + TLS 1.2} \\ 2 \times \text{RTT} & \text{HTTP/2 + TLS 1.3} \\ 1 \times \text{RTT} & \text{HTTP/3 QUIC (0-RTT resumption: 0)} \end{cases}$$

| Protocol | Handshake RTTs | At 50ms RTT | At 200ms RTT |
|:---|:---:|:---:|:---:|
| HTTP/1.1 + TLS 1.2 | 3 | 150 ms | 600 ms |
| HTTP/2 + TLS 1.3 | 2 | 100 ms | 400 ms |
| HTTP/3 (first) | 1 | 50 ms | 200 ms |
| HTTP/3 (resumed) | 0 | 0 ms | 0 ms |

---

## 3. Reverse Proxy Load Balancing

### Caddy's Load Balancing Policies

| Policy | Algorithm | Complexity |
|:---|:---|:---:|
| `random` | Uniform random | O(1) |
| `random_choose 2` | Power of two choices | O(1) |
| `round_robin` | Sequential | O(1) |
| `least_conn` | Min active connections | O(n) |
| `first` | First available | O(n) |
| `ip_hash` | Hash of client IP | O(1) |
| `uri_hash` | Hash of URI | O(1) |
| `header` | Hash of header value | O(1) |

### Power of Two Choices

The `random_choose 2` policy selects 2 random servers and picks the one with fewer connections:

$$E[\text{max load}] = O(\log \log n) \quad \text{vs } O(\log n) \text{ for pure random}$$

This dramatically reduces load imbalance with minimal overhead.

### Health Check Impact

$$\text{Effective Capacity} = \text{Healthy Servers} \times \text{Per-Server Capacity}$$

$$\text{Failover Load Increase} = \frac{\text{Total Load}}{\text{Healthy Servers}} - \frac{\text{Total Load}}{n}$$

| Servers | 1 Down | Load Increase per Server |
|:---:|:---:|:---:|
| 3 | 33% capacity lost | +50% each |
| 5 | 20% capacity lost | +25% each |
| 10 | 10% capacity lost | +11% each |
| 20 | 5% capacity lost | +5.3% each |

---

## 4. Rate Limiting

### Token Bucket Model

$$\text{Tokens} = \min(\text{burst}, \text{tokens} + \text{rate} \times \Delta t)$$

$$\text{Request Allowed} \iff \text{tokens} \geq 1$$

### Caddy Rate Limit Config

$$\text{Sustained Rate} = \text{rate (events/window)}$$

$$\text{Peak Rate} = \frac{\text{burst}}{\text{min interval between burst requests}}$$

$$\text{Recovery Time} = \frac{\text{burst}}{\text{rate}}$$

---

## 5. File Server — Static Content Performance

### Sendfile Optimization

$$T_{static} = T_{open} + T_{stat} + T_{sendfile}$$

$$T_{sendfile} = \frac{\text{File Size}}{\text{Network BW}} \quad (\text{zero-copy, no userspace buffering})$$

### ETag/If-None-Match

$$\text{304 Response} = \begin{cases} \text{Yes (0 bytes body)} & \text{if ETag matches} \\ \text{No (full response)} & \text{if ETag differs} \end{cases}$$

$$\text{Bandwidth Savings} = \frac{\text{Cache Hits}}{\text{Total Requests}} \times \text{Avg Response Size}$$

### Compression Trade-off

$$\text{Transfer Size} = \frac{\text{Original}}{\text{Compression Ratio}}$$

$$\text{Total Time} = T_{compress} + \frac{\text{Compressed Size}}{\text{BW}}$$

| File Size | Compress Time | Transfer (no compress, 10Mbps) | Transfer (gzip, 10Mbps) |
|:---:|:---:|:---:|:---:|
| 10 KiB | 0.1 ms | 8 ms | 0.1 + 1.6 ms = 1.7 ms |
| 100 KiB | 0.5 ms | 80 ms | 0.5 + 16 ms = 16.5 ms |
| 1 MiB | 5 ms | 800 ms | 5 + 160 ms = 165 ms |

---

## 6. Caddyfile vs JSON — Configuration Complexity

### Matcher Evaluation Order

Caddy evaluates matchers in a specific priority:

| Priority | Matcher Type | Example |
|:---:|:---|:---|
| 1 | Path | `/api/*` |
| 2 | Host | `example.com` |
| 3 | Method | `GET` |
| 4 | Header | `Content-Type: application/json` |
| 5 | Expression | `{remote.host} in 10.0.0.0/8` |

### Route Matching Complexity

$$T_{match} = O(R \times M)$$

Where $R$ = routes, $M$ = matchers per route.

For most configurations ($R < 100$, $M < 5$): effectively constant time.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| 90 - 30 = day 60 | Subtraction | Renewal window |
| $3 \times \text{RTT}$ vs $1 \times \text{RTT}$ | Multiplication | Protocol handshake |
| $O(\log \log n)$ | Double logarithmic | Power of two choices |
| $\frac{\text{burst}}{\text{rate}}$ | Rate equation | Rate limit recovery |
| $\frac{\text{Size}}{\text{BW}}$ | Rate equation | Transfer time |
| $\frac{\text{Hits}}{\text{Total}} \times \text{Size}$ | Product | Cache savings |

---

*Every Caddy request flows through these calculations — automatic HTTPS via ACME, HTTP/2 multiplexing, and smart load balancing, all configured in a Caddyfile that reads more like English than config syntax.*

## Prerequisites

- TLS/SSL certificate concepts (CA, CSR, chain of trust)
- HTTP/2 protocol basics (multiplexing, streams, HPACK)
- ACME protocol (Let's Encrypt challenge types)
- Reverse proxy concepts (upstream, health checks, load balancing)

## Complexity

- **Beginner:** Automatic HTTPS, static file serving, basic reverse proxy
- **Intermediate:** Handle/route directives, matchers, custom TLS, load balancing policies
- **Advanced:** ACME certificate lifecycle math, HTTP/2 stream multiplexing, rate limiter token bucket internals
