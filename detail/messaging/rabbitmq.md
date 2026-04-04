# The Mathematics of RabbitMQ — AMQP Broker Internals

> *RabbitMQ implements AMQP with exchange-based routing. The math covers queue depth management, exchange routing complexity, memory watermarks, and cluster replication overhead.*

---

## 1. Exchange Routing — Binding Math

### Exchange Types and Routing Complexity

| Exchange Type | Routing Algorithm | Complexity |
|:---|:---|:---:|
| Direct | Exact routing key match | O(1) hash lookup |
| Fanout | Deliver to all bound queues | O(Q) where Q = bound queues |
| Topic | Pattern match (wildcards) | O(B) where B = bindings |
| Headers | Match header attributes | O(B × H) where H = headers |

### Topic Exchange Trie

Topic routing uses a trie structure with `.` as separator:

$$T_{topic\_route} = O(L \times B_{avg})$$

Where $L$ = routing key tokens, $B_{avg}$ = average bindings per node.

### Fan-out Calculations

$$\text{Messages Generated} = \text{Published} \times \text{Matching Bindings}$$

| Exchange | Bindings | 1,000 msg/s Published | Messages Routed |
|:---|:---:|:---:|:---:|
| Direct | 1 | 1,000 | 1,000 |
| Fanout (5 queues) | 5 | 1,000 | 5,000 |
| Fanout (50 queues) | 50 | 1,000 | 50,000 |
| Topic (avg 3 matches) | 3 | 1,000 | 3,000 |

---

## 2. Queue Depth and Flow Control

### The Model

Queue depth is determined by the difference between producer and consumer rates.

### Queue Growth Formula

$$\text{Queue Depth}(t) = \text{Depth}(0) + (\lambda - \mu) \times t$$

Where:
- $\lambda$ = arrival rate (messages/sec)
- $\mu$ = consumption rate (messages/sec)

If $\lambda > \mu$, queue grows unboundedly. This is the fundamental queuing theory result.

### Little's Law

$$L = \lambda \times W$$

Where:
- $L$ = average queue depth (messages)
- $\lambda$ = arrival rate
- $W$ = average time a message spends in the queue

### Worked Examples

| Arrival Rate | Consume Rate | Queue Growth | Time to 1M Messages |
|:---:|:---:|:---:|:---:|
| 1,000/s | 1,000/s | 0 (stable) | Never (stable) |
| 1,100/s | 1,000/s | 100/s | 2.8 hours |
| 2,000/s | 1,000/s | 1,000/s | 16.7 minutes |
| 10,000/s | 1,000/s | 9,000/s | 1.9 minutes |

### Queue Memory Formula

$$\text{Queue Memory} = \text{Depth} \times (\text{Msg Size} + \text{Msg Overhead})$$

Message overhead in RabbitMQ ~= 70 bytes per message (headers, properties, routing metadata).

| Depth | Msg Size | Overhead | Total Memory |
|:---:|:---:|:---:|:---:|
| 10,000 | 1 KiB | 70 bytes | 10.7 MiB |
| 100,000 | 1 KiB | 70 bytes | 106.5 MiB |
| 1,000,000 | 1 KiB | 70 bytes | 1.04 GiB |
| 1,000,000 | 10 KiB | 70 bytes | 9.83 GiB |

---

## 3. Memory Watermarks — Flow Control

### The Model

RabbitMQ uses memory high watermark to trigger flow control (producer throttling).

### Memory Watermark

$$\text{Watermark} = \text{vm\_memory\_high\_watermark} \times \text{Total RAM}$$

Default: `vm_memory_high_watermark = 0.4` (40% of RAM).

$$\text{Paging Threshold} = \text{Watermark} \times \text{vm\_memory\_high\_watermark\_paging\_ratio}$$

Default paging ratio = 0.5 (page to disk at 50% of watermark = 20% of RAM).

### Worked Example

*"32 GiB RAM server."*

$$\text{High Watermark} = 32 \times 0.4 = 12.8 \text{ GiB}$$

$$\text{Paging at} = 12.8 \times 0.5 = 6.4 \text{ GiB}$$

| RAM | High Watermark (40%) | Paging Starts | Flow Control |
|:---:|:---:|:---:|:---:|
| 8 GiB | 3.2 GiB | 1.6 GiB | at 3.2 GiB |
| 16 GiB | 6.4 GiB | 3.2 GiB | at 6.4 GiB |
| 32 GiB | 12.8 GiB | 6.4 GiB | at 12.8 GiB |
| 64 GiB | 25.6 GiB | 12.8 GiB | at 25.6 GiB |

### Disk Alarm

$$\text{Disk Alarm at} = \max(\text{disk\_free\_limit}, 50 \text{ MiB})$$

Default: `disk_free_limit = {mem_relative, 1.0}` = same as RAM.

---

## 4. Prefetch and Consumer Throughput

### The Model

`prefetch_count` controls how many unacknowledged messages a consumer can hold.

### Optimal Prefetch

$$\text{Optimal Prefetch} = \frac{\text{RTT}}{\text{Processing Time per Message}} + 1$$

### Throughput vs Prefetch

$$\text{Throughput (prefetch=1)} = \frac{1}{\text{RTT} + T_{process}}$$

$$\text{Throughput (prefetch=N)} = \min\left(\frac{N}{T_{process} \times N + \text{RTT}}, \frac{1}{T_{process}}\right)$$

| RTT | $T_{process}$ | Prefetch=1 | Prefetch=10 | Prefetch=100 |
|:---:|:---:|:---:|:---:|:---:|
| 1 ms | 1 ms | 500/s | 909/s | 990/s |
| 1 ms | 10 ms | 91/s | 99/s | 100/s |
| 10 ms | 1 ms | 91/s | 500/s | 909/s |
| 10 ms | 10 ms | 50/s | 91/s | 99/s |

**Key insight:** Prefetch matters most when RTT >> processing time.

---

## 5. Cluster Replication — Quorum Queues

### The Model

Quorum queues use Raft consensus with a configurable replication factor.

### Write Latency

$$T_{write} = \max(T_{leader\_write}, T_{slowest\_majority\_follower})$$

$$\text{Majority} = \lfloor \frac{N}{2} \rfloor + 1$$

| Cluster Size | Majority | Fault Tolerance | Write Latency |
|:---:|:---:|:---:|:---|
| 1 | 1 | 0 nodes | Fastest |
| 3 | 2 | 1 node | Median of 3 |
| 5 | 3 | 2 nodes | Median of 5 |
| 7 | 4 | 3 nodes | Median of 7 |

### Replication Bandwidth

$$\text{Replication BW} = \text{Message Rate} \times \text{Avg Size} \times (N - 1)$$

| Msg Rate | Avg Size | 3-Node | 5-Node |
|:---:|:---:|:---:|:---:|
| 1,000/s | 1 KiB | 2 MB/s | 4 MB/s |
| 10,000/s | 1 KiB | 20 MB/s | 40 MB/s |
| 10,000/s | 10 KiB | 200 MB/s | 400 MB/s |

### Quorum Queue Memory

$$\text{Memory} = \text{In-memory Messages} \times (\text{Size} + 200 \text{ bytes Raft overhead})$$

---

## 6. Message TTL and Dead Letter Math

### TTL Expiry

$$\text{Expired Messages} = \lambda \times \max(0, 1 - \frac{\mu}{\lambda}) \times P(\text{age} > \text{TTL})$$

For a stable queue where $\mu \geq \lambda$:

$$\text{Max Message Age} = \frac{\text{Queue Depth}}{\mu}$$

$$\text{TTL Effective if} \quad \text{TTL} < \frac{\text{Depth}}{\mu}$$

### Dead Letter Queue Sizing

$$\text{DLQ Rate} = \text{Rejection Rate} + \text{TTL Expiry Rate} + \text{Queue Overflow Rate}$$

$$\text{DLQ Storage} = \text{DLQ Rate} \times \text{Avg Size} \times \text{DLQ Retention}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $(\lambda - \mu) \times t$ | Linear growth | Queue depth |
| $L = \lambda W$ | Little's Law | Queue sizing |
| $\text{RAM} \times 0.4$ | Percentage | Memory watermark |
| $\frac{\text{RTT}}{T_{process}} + 1$ | Ratio | Optimal prefetch |
| $\lfloor N/2 \rfloor + 1$ | Floor | Quorum majority |
| $\text{Rate} \times \text{Size} \times (N-1)$ | Product | Replication BW |

---

*Every `rabbitmqctl list_queues`, `rabbitmq-diagnostics memory_breakdown`, and management UI metric reflects these internals — an AMQP broker where queue depth management and memory watermarks are the difference between stable messaging and cascading backpressure.*

## Prerequisites

- AMQP protocol concepts (exchanges, queues, bindings, routing keys)
- Erlang/OTP process model basics
- Memory management and flow control
- Queueing theory fundamentals (arrival rate, service rate)

## Complexity

- **Beginner:** Queue declaration, basic publish/consume
- **Intermediate:** Exchange routing, memory watermark tuning, prefetch optimization
- **Advanced:** Queue depth backpressure modeling, quorum queue Raft overhead, lazy queue disk I/O estimation
