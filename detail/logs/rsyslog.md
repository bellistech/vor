# The Mathematics of rsyslog — Message Processing, Queue Theory & Forwarding Performance

> *rsyslog is a message routing engine with queuing theory at its core. Every message enters a pipeline of queues, filters, and actions — and the throughput, latency, and reliability are all governed by queue sizing, batch factors, and network costs.*

---

## 1. Message Processing Pipeline

### The Three-Stage Model

$$T_{message} = T_{input} + T_{rule\_processing} + T_{output}$$

Each stage has its own queue:

```
Input → [Main Queue] → Rule Engine → [Action Queues] → Output
```

### Throughput

$$throughput = \min(throughput_{input}, throughput_{rules}, throughput_{output})$$

The bottleneck determines system throughput. Typical capacities:

| Stage | Throughput | Bottleneck Factor |
|:---|:---:|:---|
| UDP input | 200K-500K msg/s | Kernel socket buffer |
| TCP input | 100K-300K msg/s | Connection handling |
| Rule engine | 500K-1M msg/s | Filter evaluation |
| File output | 200K-500K msg/s | Disk I/O |
| TCP forward | 50K-200K msg/s | Network + acknowledgment |
| Database output | 10K-50K msg/s | Query latency |

### Per-Message Processing Cost

$$T_{per\_message} = T_{parse} + T_{filter} + T_{template} + T_{output}$$

| Component | Cost |
|:---|:---:|
| RFC 3164/5424 parsing | 0.5-2 us |
| Filter evaluation (per rule) | 0.1-1 us |
| Template formatting | 0.5-2 us |
| File write (buffered) | 0.1-1 us |
| **Total (local file)** | **1-6 us** |

---

## 2. Queue Theory — Sizing and Behavior

### Queue Types

| Type | Persistence | Speed | Memory |
|:---|:---:|:---:|:---:|
| Direct (none) | No | Fastest | 0 |
| LinkedList | No | Fast | $N \times (msg\_size + 64)$ |
| FixedArray | No | Fastest | $max\_size \times ptr\_size$ |
| Disk | Yes | Slower | Disk space |
| Disk-Assisted | Both | Balanced | Both |

### Queue Sizing Formula

$$queue\_size \geq \frac{burst\_rate \times burst\_duration}{drain\_rate} + safety\_margin$$

Where:
- $burst\_rate$ = peak message rate
- $burst\_duration$ = how long the burst lasts
- $drain\_rate$ = output processing rate

**Example:** Burst of 100K msg/s for 60 seconds, output processes 50K msg/s:

$$queue\_size \geq \frac{100000 \times 60}{50000} = 120000 + safety$$

$$queue\_size = 200000 \text{ messages (with margin)}$$

### Memory Cost

$$memory_{queue} = queue\_size \times avg\_message\_size$$

Average syslog message: 200-500 bytes.

$$memory = 200000 \times 300 = 60 MB$$

### Disk-Assisted Queue

When memory queue reaches `highWatermark`:

$$spill\_to\_disk \iff queue\_depth > highWatermark$$

$$resume\_memory \iff queue\_depth < lowWatermark$$

Default: highWatermark = 80% of queue size, lowWatermark = 70%.

---

## 3. Facility and Severity — The Priority Formula

### The Priority Number

$$priority = facility \times 8 + severity$$

### Facilities (0-23)

| Code | Facility | Code | Facility |
|:---:|:---|:---:|:---|
| 0 | kern | 12 | ntp |
| 1 | user | 13 | audit |
| 2 | mail | 14 | alert |
| 3 | daemon | 15 | clock |
| 4 | auth | 16 | local0 |
| 5 | syslog | 17 | local1 |
| 6 | lpr | 18 | local2 |
| 7 | news | 19 | local3 |
| 8 | uucp | 20 | local4 |
| 9 | cron | 21 | local5 |
| 10 | authpriv | 22 | local6 |
| 11 | ftp | 23 | local7 |

### Severity Levels (0-7)

| Code | Severity | Keyword |
|:---:|:---|:---|
| 0 | Emergency | emerg |
| 1 | Alert | alert |
| 2 | Critical | crit |
| 3 | Error | err |
| 4 | Warning | warning |
| 5 | Notice | notice |
| 6 | Informational | info |
| 7 | Debug | debug |

### Decoding Priority

$$facility = \lfloor priority / 8 \rfloor$$
$$severity = priority \mod 8$$

**Example:** Priority 165:

$$facility = \lfloor 165/8 \rfloor = 20 = local4$$
$$severity = 165 \mod 8 = 5 = notice$$

---

## 4. Filtering Performance — Rule Engine Cost

### Filter Types

| Filter Type | Cost | Example |
|:---|:---:|:---|
| Facility/severity | $O(1)$ — bitmask | `kern.err` |
| Property comparison | $O(L)$ — string compare | `$msg contains "error"` |
| Regex | $O(L \times m)$ — NFA | `$msg regex "err[0-9]+"` |
| RainerScript expression | Variable | `if $severity <= 3 then ...` |

### Rule Evaluation Cost

Rules are evaluated **in order** (first match or all matches depending on config):

$$T_{rules} = \sum_{r=1}^{R} T_{filter}(r) \times P(\text{reaches rule } r)$$

With `stop` directive (formerly `~`):

$$P(\text{reaches rule } r) = \prod_{i=1}^{r-1} (1 - P(match_i \land stop_i))$$

### Optimization: Place Cheap, Selective Rules First

$$T_{optimized} = T_{cheap} + P(pass\_cheap) \times T_{expensive}$$

**Example:** 80% of messages are `info` severity. First rule `if $severity > 6 then stop`:

$$T = T_{severity\_check} + 0.2 \times T_{remaining\_rules}$$

---

## 5. Network Forwarding — TCP vs UDP

### UDP Forwarding

$$T_{send} = T_{serialize} + T_{network}$$

$$throughput_{UDP} = \frac{MTU - headers}{T_{send}}$$

| Parameter | Value |
|:---|:---:|
| Max message size (UDP) | 65,507 bytes (theoretical) |
| Practical max (RFC 5426) | 2048 bytes |
| Typical syslog message | 200-500 bytes |

$$messages\_per\_second_{UDP} \approx \frac{bandwidth}{avg\_message\_size}$$

On 1 Gbps: $\frac{125 \times 10^6}{300} \approx 416,000 \text{ msg/s}$

### UDP Reliability

$$P(message\_lost) = P(kernel\_buffer\_full) + P(network\_loss)$$

Kernel buffer size: `net.core.rmem_max`. At high rates:

$$P(loss) \approx 1 - \frac{drain\_rate}{arrival\_rate} \text{ when } arrival > drain$$

### TCP Forwarding (RELP)

TCP ensures delivery but adds overhead:

$$T_{TCP\_msg} = T_{serialize} + T_{send} + T_{ACK\_wait}$$

$$throughput_{TCP} = \frac{BDP}{RTT}$$

On a 10 ms RTT link with 64 KB window:

$$throughput = \frac{65536}{0.01} = 6.5 MB/s \approx 21000 \text{ msg/s at 300 bytes}$$

RELP adds application-level acknowledgment:

$$T_{RELP} = T_{TCP} + T_{app\_ACK} \approx T_{TCP} + 0.1ms$$

---

## 6. Batching — Output Efficiency

### Batch Write Model

rsyslog batches messages for output:

$$T_{batch\_write} = T_{setup} + N_{batch} \times T_{per\_msg}$$

$$throughput = \frac{N_{batch}}{T_{batch\_write}}$$

### File Output Batching

| Batch Size | Writes/s | Throughput | fsync Impact |
|:---:|:---:|:---:|:---|
| 1 | 100K | 100K msg/s | fsync per message: 50K/s max |
| 10 | 50K | 500K msg/s | fsync per batch: 500K/s |
| 100 | 10K | 1M msg/s | Rarely the bottleneck |

### asyncWritingAllowed

$$throughput_{async} = \frac{throughput_{sync}}{P(IO\_wait)}$$

With async writing, the action queue buffers while I/O completes:

$$effective = throughput_{CPU} \text{ (I/O hidden behind queue)}$$

---

## 7. Template Processing — String Cost

### Template Evaluation

$$T_{template} = \sum_{property} T_{extract}(property) + T_{format}(property)$$

### Property Types

| Property | Extract Cost | Example |
|:---|:---:|:---|
| msg | $O(1)$ — pointer | `%msg%` |
| hostname | $O(1)$ — cached | `%hostname%` |
| timestamp | $O(1)$ — cached | `%timestamp%` |
| programname | $O(L)$ — parse | `%programname%` |
| Custom property | $O(L)$ — regex | `%$.myvar%` |

### Output String Size

$$output\_size = \sum_{field} |formatted(field)| + |separators|$$

**Example:** JSON template with 8 fields, average 50 chars each:

$$output\_size = 8 \times 50 + overhead = 500 \text{ bytes}$$

$$daily\_storage = messages\_per\_day \times output\_size$$

At 100K messages/day: $100000 \times 500 = 50 MB/day$.

---

## 8. Summary of rsyslog Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Pipeline throughput | $\min(input, rules, output)$ | Bottleneck |
| Queue sizing | $burst \times duration / drain$ | Capacity |
| Priority encoding | $facility \times 8 + severity$ | Integer packing |
| Rule evaluation | $\sum T_{filter} \times P(reached)$ | Sequential cost |
| UDP throughput | $bandwidth / msg\_size$ | Network bound |
| TCP throughput | $BDP / RTT$ | Window limited |
| Batch efficiency | $N_{batch} / T_{batch}$ | Amortization |
| Daily storage | $messages/day \times output\_size$ | Capacity planning |

---

*rsyslog is a message router with queuing semantics. Every message follows a path through filters and actions, and the throughput is determined by the narrowest pipe in the pipeline. Size your queues for bursts, batch your writes for throughput, and filter early to reduce downstream load.*
