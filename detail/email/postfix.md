# The Mathematics of Postfix -- Queue Theory and Delivery Optimization

> *Every mail queue is a stochastic system: messages arrive as a Poisson process, servers drain them with exponential service times, and the administrator's job is to keep the queue length finite.*

---

## 1. Arrival and Service (Queueing Theory Foundations)

### The Problem

Postfix manages mail through multiple queues: incoming, active, deferred, hold, and corrupt. Messages arrive from clients (SMTP submissions), remote MTAs (inbound relay), and local programs (sendmail injection). The active queue has a finite concurrency limit. If arrival rate exceeds service rate, the deferred queue grows without bound and delivery latency explodes.

### The Formula

Model the Postfix active queue as an M/M/c queue (Markovian arrivals, Markovian service, c parallel workers):

$$\rho = \frac{\lambda}{c \cdot \mu}$$

where $\lambda$ is the message arrival rate (messages/second), $\mu$ is the per-worker service rate (deliveries/second per worker), and $c$ is the concurrency limit (`default_destination_concurrency_limit`). The system is stable only when $\rho < 1$.

The probability that an arriving message must wait (Erlang C formula):

$$C(c, \lambda/\mu) = \frac{\frac{(\lambda/\mu)^c}{c!} \cdot \frac{1}{1 - \rho}}{\sum_{k=0}^{c-1} \frac{(\lambda/\mu)^k}{k!} + \frac{(\lambda/\mu)^c}{c!} \cdot \frac{1}{1 - \rho}}$$

Mean time in system (waiting + service):

$$W = \frac{C(c, \lambda/\mu)}{c \cdot \mu - \lambda} + \frac{1}{\mu}$$

### Worked Examples

**Example 1: Sizing concurrency for a mailing list server.**

A list server sends 500 messages/minute ($\lambda = 8.33$ msg/s). Average delivery time to remote MTAs is 2 seconds ($\mu = 0.5$ msg/s per worker). Required concurrency for stability:

$$c > \frac{\lambda}{\mu} = \frac{8.33}{0.5} = 16.67$$

So at minimum $c = 17$ workers. For $c = 20$:

$$\rho = \frac{8.33}{20 \times 0.5} = 0.833$$

The system is stable with 17% headroom. Setting `default_destination_concurrency_limit = 20` handles this load.

**Example 2: Deferred queue growth estimation.**

If a destination MTA is down, messages for that domain defer. With 100 messages/hour arriving for the downed domain and the default retry schedule (300s, 300s, 300s, ..., up to `maximal_queue_lifetime = 5d`), the deferred queue accumulates:

$$Q_{deferred}(t) = \lambda \cdot t = 100 \cdot 120 = 12{,}000 \text{ messages after 5 days}$$

Each retry attempt consumes a worker slot, so deferred retries compete with fresh deliveries.

## 2. Backoff and Retry Scheduling (Exponential Backoff)

### The Problem

When delivery fails temporarily (4xx), Postfix reschedules the message with increasing delay. The parameters `minimal_backoff_time` and `maximal_backoff_time` control bounds. The actual delay doubles each attempt (exponential backoff).

### The Formula

Delay on attempt $n$:

$$d_n = \min\left(d_{max},\; d_{min} \cdot 2^{n-1}\right)$$

Total elapsed time before attempt $N$:

$$T_N = \sum_{n=1}^{N-1} d_n = d_{min} \cdot (2^{k} - 1) + d_{max} \cdot (N - 1 - k)$$

where $k = \lfloor \log_2(d_{max}/d_{min}) \rfloor$ is the attempt at which the cap is reached.

### Worked Examples

**Example: Default Postfix backoff.**

With $d_{min} = 300$s (5 min) and $d_{max} = 4000$s (~67 min):

| Attempt | Delay (s) | Cumulative (min) |
|---------|-----------|-------------------|
| 1       | 300       | 0                 |
| 2       | 600       | 5                 |
| 3       | 1200      | 15                |
| 4       | 2400      | 35                |
| 5       | 4000      | 75                |
| 6       | 4000      | 142               |

The cap kicks in at attempt 5. After 24 hours, approximately 19 attempts will have been made.

## 3. Rate Limiting (Token Bucket Model)

### The Problem

Postfix anvil implements per-client rate limiting. The parameters `smtpd_client_connection_rate_limit` and `smtpd_client_message_rate_limit` define maximum rates within `anvil_rate_time_unit`. This is equivalent to a token bucket.

### The Formula

A token bucket with rate $r$ tokens/second and burst capacity $b$:

$$\text{tokens}(t) = \min\left(b,\; \text{tokens}(t_0) + r \cdot (t - t_0)\right)$$

A connection is permitted if $\text{tokens}(t) \geq 1$, consuming one token. In Postfix terms:

$$r = \frac{\text{rate\_limit}}{\text{anvil\_rate\_time\_unit}}$$

With `smtpd_client_connection_rate_limit = 50` and `anvil_rate_time_unit = 60s`:

$$r = \frac{50}{60} \approx 0.833 \text{ connections/second}$$

### Worked Examples

**Example: Detecting a brute-force SMTP AUTH attack.**

An attacker connects 5 times/second. With limit 50 per 60s, the bucket drains in:

$$t_{drain} = \frac{50}{5} = 10 \text{ seconds}$$

After 10 seconds, all subsequent connections are rejected until the window slides. The attacker can make at most 50 attempts per minute regardless of connection speed.

## 4. DNS and MX Priority (Weighted Selection)

### The Problem

When Postfix delivers to a domain, it queries MX records and connects to the lowest-priority (highest-preference) server. If multiple MX records share the same priority, Postfix distributes load among them.

### The Formula

For $n$ MX records at the same priority, each with equal weight, the probability of selecting server $i$:

$$P(i) = \frac{1}{n}$$

For domains with multiple priority tiers, Postfix attempts tier $k$ only after all servers in tiers $1, \ldots, k-1$ have failed. Expected delivery time with per-tier failure probability $p_k$ and connection timeout $\tau$:

$$E[T] = \sum_{k=1}^{K} \left(\prod_{j=1}^{k-1} p_j\right) \cdot \left(\sum_{j=1}^{k-1} n_j \cdot \tau + \frac{1}{\mu_k}\right)$$

### Worked Examples

**Example: Three-tier MX with failover.**

| Priority | Servers | Failure Prob | Timeout |
|----------|---------|-------------|---------|
| 10       | 2       | 0.01        | 30s     |
| 20       | 1       | 0.05        | 30s     |
| 30       | 1       | 0.10        | 30s     |

Expected delivery time:

$$E[T] \approx (1 - 0.01) \cdot 2\text{s} + 0.01 \cdot (1 - 0.05) \cdot (60\text{s} + 2\text{s}) + 0.01 \cdot 0.05 \cdot (90\text{s} + 2\text{s})$$

$$E[T] \approx 1.98 + 0.589 + 0.046 = 2.62\text{s}$$

The multi-tier design keeps expected latency low despite having fallback servers.

## 5. Connection Caching (Amortized Overhead)

### The Problem

Establishing a TLS connection involves a TCP handshake (1 RTT), TLS handshake (2 RTT for TLS 1.2, 1 RTT for TLS 1.3), and SMTP EHLO exchange (1 RTT). Postfix connection caching (`smtp_connection_cache_on_demand`) amortizes this overhead across multiple messages.

### The Formula

Without caching, per-message overhead:

$$O_{uncached} = \text{RTT}_{tcp} + \text{RTT}_{tls} + \text{RTT}_{ehlo} + T_{data}$$

With caching and $m$ messages per cached connection:

$$O_{cached} = \frac{\text{RTT}_{tcp} + \text{RTT}_{tls} + \text{RTT}_{ehlo}}{m} + T_{data}$$

Throughput improvement factor:

$$\text{Speedup} = \frac{O_{uncached}}{O_{cached}} = \frac{H + T_{data}}{\frac{H}{m} + T_{data}}$$

where $H = \text{RTT}_{tcp} + \text{RTT}_{tls} + \text{RTT}_{ehlo}$.

### Worked Examples

**Example: Transatlantic delivery.**

RTT to remote server: 120ms. TLS 1.2 handshake: 240ms (2 RTT). EHLO: 120ms. Data transfer: 50ms. So $H = 480$ms.

Without caching: $480 + 50 = 530$ms per message.

With caching and $m = 10$ messages per connection: $48 + 50 = 98$ms per message.

$$\text{Speedup} = \frac{530}{98} \approx 5.4\times$$

Connection caching delivers a 5.4x throughput improvement for high-latency destinations.

**Example 2: TLS 1.3 vs TLS 1.2 overhead.**

TLS 1.3 reduces the handshake to 1 RTT (from 2 in TLS 1.2). For the same transatlantic link:

$H_{1.3} = 120 + 120 + 120 = 360$ms vs $H_{1.2} = 120 + 240 + 120 = 480$ms.

$$\text{Improvement} = \frac{480 - 360}{480} = 25\%$$

Per-message savings with caching ($m = 10$): $(480 - 360)/10 = 12$ms -- marginal when amortized, but the first message in each connection saves 120ms.

## 6. Message Size Distribution (Heavy-Tailed Traffic)

### The Problem

Email message sizes follow a heavy-tailed distribution. Most messages are small (< 50 KB), but a few are very large (10+ MB attachments). Queue management and rate limiting must account for variance in processing time, since large messages consume disproportionate I/O and bandwidth.

### The Formula

Email sizes are often modeled as log-normal with parameters $\mu_L$ and $\sigma_L$:

$$f(x) = \frac{1}{x \sigma_L \sqrt{2\pi}} \exp\left(-\frac{(\ln x - \mu_L)^2}{2\sigma_L^2}\right)$$

The mean and variance of message size $X$:

$$E[X] = e^{\mu_L + \sigma_L^2/2}, \quad \text{Var}(X) = (e^{\sigma_L^2} - 1) \cdot e^{2\mu_L + \sigma_L^2}$$

### Worked Examples

**Example: Estimating queue bandwidth requirements.**

From server logs: median message size = 15 KB ($\mu_L = \ln(15360) = 9.64$), with $\sigma_L = 2.0$.

$$E[X] = e^{9.64 + 2.0} = e^{11.64} = 113{,}000 \text{ bytes} \approx 110 \text{ KB}$$

The mean (110 KB) is 7.3x the median (15 KB) due to the heavy tail. Queue bandwidth planning must use the mean, not the median, or large messages will cause unexpected congestion.

## Prerequisites

- Queueing theory fundamentals (M/M/1, M/M/c models, Erlang formulas)
- Exponential backoff and retry strategies
- Token bucket rate limiting model
- DNS MX record resolution and priority semantics
- TCP/TLS handshake mechanics and round-trip time analysis
- Log-normal distributions and heavy-tailed traffic modeling
- Basic probability (conditional probability, expected value)
