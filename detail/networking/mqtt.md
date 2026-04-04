# The Mathematics of MQTT — QoS Delivery Guarantees & Broker Scaling

> *MQTT's three QoS levels represent a spectrum from probabilistic delivery to deterministic exactly-once semantics, each with measurable costs in bandwidth, latency, and broker state — the mathematics of reliable messaging in unreliable networks.*

---

## 1. QoS 0 Delivery Probability (Bernoulli Trials)

### The Problem

With QoS 0, messages are sent once with no acknowledgment. If the link has packet loss rate $p$, what is the probability of a message arriving after $n$ independent transmissions?

### The Formula

Each QoS 0 publish is a Bernoulli trial with success probability $q = 1 - p$. For a single message:

$$P(\text{delivered}) = 1 - p$$

For $n$ independent messages, the number delivered follows $\text{Binomial}(n, q)$:

$$P(k \text{ delivered}) = \binom{n}{k} q^k (1-q)^{n-k}$$

Expected messages delivered:

$$E[k] = n \cdot q$$

Probability that at least one of $m$ repeated sends of the same message arrives:

$$P(\text{at least one}) = 1 - p^m$$

### Worked Examples

**Example 1:** IoT sensor publishing temperature every 5 seconds over a link with 2% packet loss. Over 1 hour ($n = 720$ messages):

$$E[\text{delivered}] = 720 \times 0.98 = 705.6$$

$$\sigma = \sqrt{720 \times 0.98 \times 0.02} = \sqrt{14.112} = 3.76$$

95% CI: $705.6 \pm 7.5$, so between 698 and 713 messages delivered.

**Example 2:** Critical alert sent 3 times at QoS 0 as a workaround, $p = 0.05$:

$$P(\text{received}) = 1 - 0.05^3 = 1 - 0.000125 = 0.999875$$

Triple-sending at QoS 0 approaches QoS 1 reliability but wastes bandwidth for 99.875% of messages.

---

## 2. QoS 1 Duplicate Rate (Retry Analysis)

### The Problem

QoS 1 retransmits until PUBACK is received. If the PUBACK itself is lost, the subscriber receives duplicates. What is the expected duplicate rate?

### The Formula

Let $p_f$ = forward loss (PUBLISH), $p_r$ = return loss (PUBACK). A message is delivered on attempt $k$ if the first $k-1$ PUBACKs were lost:

$$P(\text{delivered on attempt } k) = (1-p_f) \cdot p_r^{k-1} \cdot (1-p_r)$$

Expected number of times subscriber receives the message (given it arrives):

$$E[\text{copies}] = \frac{1}{1-p_r}$$

Duplicate rate (fraction of received messages that are duplicates):

$$D = 1 - (1-p_r) = p_r$$

### Worked Examples

**Example:** Link with $p_f = 0.02$, $p_r = 0.02$:

$$E[\text{copies}] = \frac{1}{0.98} = 1.0204$$

About 2% of messages are duplicated. Over 1M messages:

$$\text{duplicates} = 1000000 \times 0.0204 = 20400$$

For an application processing billing events, 20K duplicates would be catastrophic — use QoS 2 or application-level deduplication (idempotency keys).

---

## 3. QoS 2 Bandwidth Cost (Protocol Overhead)

### The Problem

QoS 2 requires a 4-packet handshake (PUBLISH, PUBREC, PUBREL, PUBCOMP). What is the total bandwidth overhead compared to QoS 0 for messages of payload size $L$?

### The Formula

Packet sizes (MQTT 3.1.1, minimal headers):

| Packet | Fixed Header | Variable Header | Payload | Total |
|--------|-------------|-----------------|---------|-------|
| PUBLISH (QoS 0) | 2 | 2 + topic_len | $L$ | $4 + t + L$ |
| PUBLISH (QoS 1) | 2 | 4 + topic_len | $L$ | $6 + t + L$ |
| PUBACK | 2 | 2 | 0 | 4 |
| PUBLISH (QoS 2) | 2 | 4 + topic_len | $L$ | $6 + t + L$ |
| PUBREC | 2 | 2 | 0 | 4 |
| PUBREL | 2 | 2 | 0 | 4 |
| PUBCOMP | 2 | 2 | 0 | 4 |

Total bytes per message delivery:

$$B_0 = 4 + t + L$$

$$B_1 = (6 + t + L) + 4 = 10 + t + L$$

$$B_2 = (6 + t + L) + 4 + 4 + 4 = 18 + t + L$$

Overhead ratio QoS 2 vs QoS 0:

$$R = \frac{B_2}{B_0} = \frac{18 + t + L}{4 + t + L}$$

### Worked Examples

**Example:** Topic length $t = 20$ bytes:

| Payload $L$ | $B_0$ | $B_1$ | $B_2$ | $R$ (QoS 2/QoS 0) |
|------------|-------|-------|-------|-------------------|
| 10 bytes | 34 | 40 | 48 | 1.41x |
| 100 bytes | 124 | 130 | 138 | 1.11x |
| 1000 bytes | 1024 | 1030 | 1038 | 1.01x |

For small messages (10 bytes), QoS 2 costs 41% more bandwidth. For typical IoT payloads (100+ bytes), the overhead is under 12%.

---

## 4. Broker Fan-Out and Topic Matching (Tree Traversal)

### The Problem

A broker with $T$ topic subscriptions must match each published message against potentially wildcard-containing subscriptions. What is the computational complexity of topic matching?

### The Formula

Topics form a trie (prefix tree) with branching factor $b$ and depth $d$ (number of `/`-separated levels). For a published topic with $d$ levels:

- Exact match: $O(d)$ — traverse trie directly
- Single-level wildcard (`+`): at each level, follow both the exact branch and the `+` branch: $O(2^d)$ worst case
- Multi-level wildcard (`#`): at each level, check for `#` terminator: $O(d)$ additional

Practical complexity with $S$ subscriptions:

$$O(d + W \cdot b)$$

Where $W$ is the number of wildcard subscriptions at matching depth levels.

Fan-out: if $M$ subscribers match a published message, the broker must copy/deliver $M$ times:

$$\text{Fan-out cost} = M \cdot (H_{\text{packet}} + L)$$

### Worked Examples

**Example:** Broker with 100K subscriptions, average topic depth $d = 4$, 500 wildcard subscriptions. Published message matches 50 subscribers with payload 200 bytes:

Topic match time: $O(4 + 500) \approx 504$ comparisons per publish.

Fan-out bandwidth: $50 \times (6 + 200) = 10300$ bytes per published message.

At 10K messages/sec inbound:

$$\text{outbound} = 10000 \times 10300 = 103 \text{ MB/sec} = 824 \text{ Mbps}$$

---

## 5. Session State and Memory Scaling (Combinatorics)

### The Problem

For persistent sessions (`clean_session=false`), the broker stores subscriptions and queued messages per client. With $C$ clients, $S$ subscriptions per client, and a maximum queue depth of $Q$ messages, what is the broker's memory requirement?

### The Formula

Per-client memory:

$$M_c = S \cdot M_{\text{sub}} + \min(Q, \bar{n}) \cdot \bar{L}$$

Where:
- $M_{\text{sub}} \approx 100$ bytes (topic filter + QoS + metadata)
- $\bar{n}$ = average queued messages for offline client
- $\bar{L}$ = average message size

Total broker memory:

$$M_{\text{total}} = C \cdot M_c + T_{\text{unique}} \cdot M_{\text{trie}}$$

Where $T_{\text{unique}}$ is the number of unique topic nodes in the trie and $M_{\text{trie}} \approx 200$ bytes per node.

### Worked Examples

**Example:** 50K IoT devices, 5 subscriptions each, offline 80% of the time, queue max 1000 messages at 100 bytes average:

Online clients ($C_{\text{on}} = 10000$):

$$M_{\text{on}} = 10000 \times (5 \times 100 + 0) = 5 \text{ MB}$$

Offline clients ($C_{\text{off}} = 40000$), average 200 queued messages:

$$M_{\text{off}} = 40000 \times (500 + 200 \times 100) = 40000 \times 20500 = 820 \text{ MB}$$

Total: 825 MB. If queue depth grows to max (1000):

$$M_{\text{off}} = 40000 \times (500 + 1000 \times 100) = 4.02 \text{ GB}$$

This is why `max_queued_messages` is critical for broker stability.

---

## 6. Retained Message Consistency (Convergence)

### The Problem

When a retained message is published to a topic, it replaces the previous value. With $n$ publishers racing to update the same retained topic, what is the probability of the final value being from a specific publisher?

### The Formula

If publishers send at Poisson rates $\lambda_1, \lambda_2, \ldots, \lambda_n$, the probability that the last message in any time window $T$ came from publisher $i$:

$$P(\text{last from } i) = \frac{\lambda_i}{\sum_{j=1}^{n} \lambda_j}$$

This follows from the competing Poisson processes property. The expected time between retained message updates:

$$E[\Delta t] = \frac{1}{\sum_{j=1}^{n} \lambda_j}$$

### Worked Examples

**Example:** Three sensors updating `room/temp`: sensor A at 0.2 msg/sec, B at 0.1 msg/sec, C at 0.05 msg/sec:

$$\lambda_{\text{total}} = 0.35 \text{ msg/sec}$$

$$P(\text{retained from A}) = \frac{0.2}{0.35} = 57.1\%$$

$$P(\text{retained from B}) = \frac{0.1}{0.35} = 28.6\%$$

$$P(\text{retained from C}) = \frac{0.05}{0.35} = 14.3\%$$

A new subscriber connecting at a random time has a 57.1% chance of seeing sensor A's value as the retained message. For consistent state, use a single publisher per retained topic or add timestamps to resolve ordering.

---

## Prerequisites

- Probability (Bernoulli trials, binomial distribution, Poisson processes)
- Queueing theory (arrival rates, service rates)
- Data structures (tries, tree traversal complexity)
- Information theory (overhead analysis, bandwidth calculation)
- Combinatorics (counting states, memory estimation)
