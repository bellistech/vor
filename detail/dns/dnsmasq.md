# The Mathematics of dnsmasq — Lightweight DNS/DHCP Internals

> *dnsmasq is a lightweight DNS forwarder, DHCP server, and TFTP server. The math covers DNS cache sizing, DHCP lease pool calculations, query forwarding performance, and blocklist filtering.*

---

## 1. DNS Cache — Memory and Performance

### The Model

dnsmasq caches DNS responses in memory. Cache size directly determines hit ratio and performance.

### Cache Size Configuration

$$\text{cache-size} = n \quad (\text{default: 150, max recommended: 10,000})$$

$$\text{Memory per Entry} \approx 100-300 \text{ bytes (name + rdata + TTL + overhead)}$$

$$\text{Total Cache Memory} = n \times \text{Avg Entry Size}$$

| cache-size | Memory | Suitable For |
|:---:|:---:|:---|
| 150 (default) | 30 KiB | Single user |
| 1,000 | 200 KiB | Small office |
| 5,000 | 1 MiB | Medium network |
| 10,000 | 2 MiB | Large network |
| 50,000 | 10 MiB | DNS sinkhole |

### Cache Hit Ratio Model

$$\text{Hit Ratio} = 1 - \frac{\text{Unique Domains Queried per TTL Window}}{\text{Cache Size}}$$

For a typical user querying ~500 unique domains per hour with avg TTL = 300s:

$$\text{Active Entries} \approx 500 \quad (\text{in any 5-minute window})$$

$$\text{Hit Ratio (cache=150)} \approx 1 - \frac{500}{150} = \text{negative (too small, many evictions)}$$

$$\text{Hit Ratio (cache=1000)} \approx 1 - \frac{500}{1000} = 50\%$$

$$\text{Hit Ratio (cache=5000)} \approx 1 - \frac{500}{5000} = 90\%$$

### Cache Effectiveness

$$\text{Effective Latency} = \text{Hit} \times T_{local} + (1 - \text{Hit}) \times T_{upstream}$$

| Hit Ratio | $T_{local}$ | $T_{upstream}$ | Effective |
|:---:|:---:|:---:|:---:|
| 50% | 0.1 ms | 30 ms | 15.1 ms |
| 80% | 0.1 ms | 30 ms | 6.1 ms |
| 90% | 0.1 ms | 30 ms | 3.1 ms |
| 95% | 0.1 ms | 30 ms | 1.6 ms |

---

## 2. DHCP Lease Pool — Address Management

### The Model

dnsmasq manages DHCP address pools with configurable ranges and lease durations.

### Pool Size

$$\text{Pool Size} = \text{End IP} - \text{Start IP} + 1$$

$$\text{Available} = \text{Pool Size} - \text{Active Leases} - \text{Static Reservations}$$

### Worked Example

```
dhcp-range=192.168.1.100,192.168.1.250,24h
```

$$\text{Pool Size} = 250 - 100 + 1 = 151 \text{ addresses}$$

### Lease Duration Trade-offs

$$\text{Address Turnover} = \frac{\text{Unique Clients per Day}}{\text{Pool Size}}$$

| Lease Duration | Address Reuse Rate | Best For |
|:---:|:---:|:---|
| 1 hour | High | Busy coffee shop, guest WiFi |
| 12 hours | Medium | Office network |
| 24 hours (default) | Low | Home network |
| 1 week | Very low | Server network |
| Infinite | None | Static-like |

### Pool Exhaustion

$$T_{exhaustion} = \frac{\text{Pool Size}}{\text{New Clients per Hour}} \times \text{Lease Duration (hours)}$$

$$\text{Actually:} \quad T_{exhaustion} = \frac{\text{Pool Size} \times \text{Lease Duration}}{\text{Client Arrival Rate} \times \text{Lease Duration}} = \frac{\text{Pool Size}}{\text{Arrival Rate}}$$

| Pool Size | New Clients/Hour | Time to Exhaust |
|:---:|:---:|:---:|
| 50 | 10 | 5 hours |
| 150 | 10 | 15 hours |
| 150 | 50 | 3 hours |
| 250 | 5 | 50 hours |

### Lease File Size

$$\text{Lease File Size} = \text{Active Leases} \times \text{Entry Size (avg 150 bytes)}$$

---

## 3. DNS Blocklist — Pi-hole Style Filtering

### The Model

dnsmasq can load blocklists (e.g., for ad blocking) as local DNS overrides.

### Blocklist Performance

$$\text{Lookup Time} = O(1) \quad (\text{hash table lookup per domain})$$

$$\text{Memory per Blocked Domain} \approx 50-100 \text{ bytes}$$

### Blocklist Scaling

| Blocked Domains | Memory | Load Time | Query Impact |
|:---:|:---:|:---:|:---:|
| 10,000 | 800 KiB | <1s | None |
| 100,000 | 8 MiB | 2-5s | None |
| 500,000 | 40 MiB | 10-20s | Negligible |
| 1,000,000 | 80 MiB | 20-40s | Negligible |
| 5,000,000 | 400 MiB | 60-120s | Minimal |

### Blocking Effectiveness

$$\text{Blocked Queries \%} = \frac{\text{Queries Matching Blocklist}}{\text{Total Queries}} \times 100$$

Typical home network: 15-40% of DNS queries are ads/tracking.

$$\text{Bandwidth Saved} \approx \text{Blocked \%} \times \text{Avg Ad Resource Size} \times \text{Queries/Day}$$

---

## 4. Query Forwarding — Upstream Performance

### The Model

dnsmasq forwards cache misses to upstream DNS servers.

### Forwarding Throughput

$$\text{Forwarding QPS} = \frac{1}{T_{upstream\_RTT}}$$

With parallel forwarding to multiple upstreams:

$$\text{QPS} = \frac{\text{Max Outstanding Queries}}{T_{avg\_upstream}}$$

### Upstream Selection

```
server=8.8.8.8
server=1.1.1.1
server=9.9.9.9
```

dnsmasq forwards to the fastest responding upstream:

$$\text{Selected} = \arg\min_i T_{response_i}$$

### Upstream Failure Detection

$$\text{Server marked down after} = \text{N consecutive timeouts}$$

$$T_{timeout} = 2 \text{ seconds (default)}$$

---

## 5. TFTP Server — Boot Performance

### The Model

dnsmasq includes a TFTP server for PXE network booting.

### Transfer Time

$$T_{tftp} = \frac{\text{File Size}}{\text{Block Size} \times \frac{1}{\text{RTT}}}$$

Default TFTP block size = 512 bytes. With block size extension (RFC 2348): up to 65464 bytes.

| File Size | Block=512 | Block=1468 (MTU) | Block=65464 |
|:---:|:---:|:---:|:---:|
| 1 MiB | 2,048 packets | 697 packets | 16 packets |
| 50 MiB | 102,400 packets | 34,854 packets | 781 packets |

$$T_{1MiB, 512} = 2048 \times 1\text{ms RTT} = 2\text{s}$$

$$T_{1MiB, 65464} = 16 \times 1\text{ms RTT} = 0.016\text{s}$$

---

## 6. Resource Usage — Minimal Footprint

### Process Memory

$$\text{Base Memory} \approx 1-3 \text{ MiB}$$

$$\text{Total Memory} = \text{Base} + \text{Cache} + \text{DHCP Leases} + \text{Blocklist}$$

| Configuration | Memory |
|:---|:---:|
| Default (cache=150) | 2 MiB |
| Large cache (5000) + DHCP (200 leases) | 5 MiB |
| Pi-hole (500K blocklist, 10K cache) | 50 MiB |
| Maxed out (1M blocklist, 10K cache, 500 leases) | 100 MiB |

### vs BIND

| Feature | dnsmasq | BIND |
|:---|:---:|:---:|
| Memory (basic) | 2 MiB | 50 MiB |
| QPS (cached) | 10K-50K | 100K-1M |
| Startup time | <1s | 5-30s |
| Zone hosting | No (forwarding only) | Yes (authoritative) |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $n \times 200$ bytes | Linear | Cache memory |
| $1 - \frac{\text{Unique}}{\text{Cache Size}}$ | Ratio | Hit ratio |
| $\text{End} - \text{Start} + 1$ | Subtraction | DHCP pool size |
| $\frac{\text{Pool}}{\text{Arrival Rate}}$ | Rate equation | Pool exhaustion time |
| O(1) hash lookup | Constant | Blocklist query |
| $\text{Hit} \times T_l + (1-\text{Hit}) \times T_u$ | Weighted average | Effective latency |

---

*Every `dnsmasq --test`, `kill -HUP`, and DNS query to port 53 runs through this lightweight forwarder — a single binary that replaces separate DNS, DHCP, and TFTP servers with <3 MiB of RAM.*

## Prerequisites

- DNS forwarding vs authoritative resolution
- DHCP lease lifecycle (DORA: Discover, Offer, Request, Acknowledge)
- Hash table caching and LRU eviction

## Complexity

- **Beginner:** DNS forwarding setup, DHCP range configuration, static leases
- **Intermediate:** Blocklist filtering, conditional forwarding, PXE/TFTP boot, upstream server failover
- **Advanced:** Cache sizing formulas, DHCP pool exhaustion math, query rate limiting, memory footprint analysis
