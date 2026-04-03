# The Mathematics of NATS — High-Performance Messaging Internals

> *NATS is a lightweight, high-performance messaging system. The math covers subject routing, JetStream persistence, queue group distribution, and the cluster gossip protocol.*

---

## 1. Subject-Based Routing — Trie Lookup

### The Model

NATS routes messages by matching subjects (dot-separated tokens) against subscriptions. The server uses a **trie** data structure for efficient matching.

### Trie Lookup Complexity

$$T_{match} = O(L) \quad \text{where } L = \text{number of tokens in subject}$$

For subject `orders.us.west.created` ($L = 4$): 4 trie node traversals.

### Wildcard Matching

| Pattern | Matches | Traversal Cost |
|:---|:---|:---:|
| `orders.us.west.created` | Exact match only | O(L) |
| `orders.*.west.created` | `*` = single token wildcard | O(L) |
| `orders.>` | `>` = multi-token wildcard | O(L') where L' = prefix length |
| `orders.us.>` | All under `orders.us` | O(2) for prefix |

### Subscription Fan-out

$$\text{Messages Delivered} = \sum_{i=1}^{S} \text{match}(\text{subject}, \text{sub}_i)$$

Where $S$ = total subscriptions. With trie routing:

$$T_{route} = O(L) + O(M) \quad (\text{L for lookup, M for matched subscriber count})$$

### Worked Example

*"1 million subscriptions, message on `data.sensor.temp.building1`."*

- Trie lookup: O(4) = constant time, regardless of subscription count
- Delivery: O(M) where M = matching subscribers

| Total Subs | Matching Subs | Routing Time | vs Linear Scan |
|:---:|:---:|:---:|:---:|
| 1,000 | 5 | ~0.001 ms | 50x faster |
| 100,000 | 50 | ~0.005 ms | 1000x faster |
| 1,000,000 | 100 | ~0.01 ms | 5000x faster |

---

## 2. Queue Groups — Load Distribution

### The Model

Queue groups distribute messages across subscribers in a group. Each message goes to exactly one member.

### Distribution Formula

$$P(\text{subscriber } i \text{ receives}) = \frac{1}{|G|}$$

$$\text{Messages per subscriber} = \frac{\text{Total Messages}}{|G|}$$

Where $|G|$ = number of subscribers in the queue group.

### Load Imbalance

With random (pseudo-round-robin) distribution:

$$\text{Expected Imbalance} = O(\sqrt{\frac{n}{|G|}})$$

Where $n$ = total messages. For large $n$, distribution approaches uniform.

| Messages | Group Size | Expected per Sub | Std Dev |
|:---:|:---:|:---:|:---:|
| 10,000 | 5 | 2,000 | ~45 |
| 100,000 | 10 | 10,000 | ~100 |
| 1,000,000 | 20 | 50,000 | ~224 |

### Multiple Queue Groups

If a subject has $Q$ queue groups, each message is sent to one member of each group:

$$\text{Copies per Message} = Q + \text{Non-Queue Subscribers}$$

---

## 3. JetStream — Persistent Storage

### The Model

JetStream adds persistence to NATS via streams (append-only logs with configurable retention).

### Storage Sizing

$$\text{Stream Size} = \text{Message Rate} \times \text{Avg Message Size} \times \text{Retention Period}$$

### Replication Factor

$$\text{Total Storage} = \text{Stream Size} \times R \quad (\text{R = replication factor, 1 or 3})$$

### Worked Examples

| Msg Rate | Avg Size | Retention | R=1 | R=3 |
|:---:|:---:|:---:|:---:|:---:|
| 1,000/s | 1 KiB | 1 day | 82.4 GiB | 247.2 GiB |
| 10,000/s | 500 bytes | 7 days | 288.4 GiB | 865.1 GiB |
| 100,000/s | 200 bytes | 1 hour | 68.7 GiB | 206.0 GiB |
| 1,000/s | 10 KiB | 30 days | 2.4 TiB | 7.2 TiB |

### Consumer Acknowledgment

$$\text{Ack Wait Timeout} = \text{ack\_wait (default 30s)}$$

$$\text{Max Redeliveries} = \text{max\_deliver (default unlimited)}$$

$$\text{Effective Throughput} = \frac{\text{Messages per Ack Batch}}{\text{RTT} + T_{process}}$$

### Pull vs Push Consumers

| Mode | Throughput Model | Best For |
|:---|:---|:---|
| Push | $\text{Rate} = \min(\text{Server Rate}, \text{Client Rate})$ | Real-time, fast consumers |
| Pull | $\text{Rate} = \frac{\text{Batch Size}}{\text{RTT} + T_{process}}$ | Batch processing, backpressure |

---

## 4. Cluster Gossip Protocol

### The Model

NATS servers form a full-mesh cluster. Route information propagates via gossip.

### Full Mesh Connections

$$\text{Connections} = \frac{N(N-1)}{2}$$

| Servers | Connections | Gossip Messages per Update |
|:---:|:---:|:---:|
| 3 | 3 | 2 (broadcast to peers) |
| 5 | 10 | 4 |
| 7 | 21 | 6 |
| 10 | 45 | 9 |

### Subscription Propagation

When a client subscribes, the subscription propagates to all servers:

$$T_{propagation} = O(N) \times T_{route\_message}$$

$$\text{Route Memory} = \text{Unique Subscriptions} \times \text{Per-Sub Overhead (avg ~100 bytes)}$$

---

## 5. Message Throughput and Latency

### Wire Protocol Overhead

NATS protocol is text-based:

$$\text{Wire Size} = \text{PUB header} + \text{Subject Length} + \text{Payload Size} + \text{CRLF}$$

$$\text{PUB header} \approx 4 + |\text{subject}| + |\text{size digits}| + 4 \text{ bytes}$$

### Throughput Model

$$\text{Max Messages/sec} = \frac{\text{Network BW}}{\text{Avg Wire Size}}$$

| Network | Avg Msg (wire) | Max Msg/sec |
|:---:|:---:|:---:|
| 1 Gbps | 100 bytes | 1,250,000 |
| 1 Gbps | 1 KiB | 122,070 |
| 10 Gbps | 100 bytes | 12,500,000 |
| 10 Gbps | 1 KiB | 1,220,703 |

### Latency Breakdown

$$T_{e2e} = T_{pub\_encode} + T_{network} + T_{route} + T_{deliver}$$

| Component | Typical (LAN) | Typical (WAN) |
|:---|:---:|:---:|
| Encode | 0.001 ms | 0.001 ms |
| Network | 0.05-0.1 ms | 1-50 ms |
| Route (trie) | 0.001 ms | 0.001 ms |
| Deliver | 0.01 ms | 0.01 ms |
| **Total** | **0.06-0.11 ms** | **1-50 ms** |

---

## 6. Leaf Nodes and Super Clusters

### Leaf Node Model

Leaf nodes connect local NATS servers to a hub, reducing full-mesh overhead:

$$\text{Hub Connections} = L \quad (\text{one per leaf node})$$

$$\text{vs Full Mesh} = \frac{(L + H)(L + H - 1)}{2}$$

| Leaf Nodes | Hub Servers | Leaf Connections | Full Mesh Would Be |
|:---:|:---:|:---:|:---:|
| 10 | 3 | 10 | 78 |
| 50 | 3 | 50 | 1,378 |
| 200 | 5 | 200 | 20,910 |

### Subject Mapping/Import Cost

$$\text{Mapped Subject} = \text{Original Lookup} + \text{Mapping Lookup}$$

Adds O(1) per mapped subject to routing.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| O(L) trie lookup | Constant per token | Subject routing |
| $\frac{1}{|G|}$ | Uniform probability | Queue group distribution |
| $\text{Rate} \times \text{Size} \times T$ | Triple product | Stream storage |
| $\frac{N(N-1)}{2}$ | Quadratic | Cluster mesh connections |
| $\frac{\text{BW}}{\text{Msg Size}}$ | Rate equation | Max throughput |
| $L$ connections | Linear | Leaf node scaling |

---

*Every `nats sub`, `nats pub`, and `nats stream info` reflects these internals — a messaging system designed for simplicity where subject-based trie routing delivers microsecond latencies at millions of messages per second.*
