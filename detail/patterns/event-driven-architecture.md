# The Mathematics of Event-Driven Systems — Ordering, Idempotency, and Windowing

> *Event-driven architectures must contend with the impossibility of global ordering, the necessity of idempotent operations, and the mathematics of time-windowed processing. These formal foundations explain why certain guarantees are achievable and others are not.*

---

## 1. Event Ordering Guarantees (Total vs Partial Order)

### The Problem

In a distributed event system, can we guarantee that all consumers see events in the same order? What are the costs and limitations?

### The Formula

A **total order** is a relation $\leq$ on a set $S$ that is reflexive, antisymmetric, transitive, and **total** (for all $a, b$: $a \leq b$ or $b \leq a$).

A **partial order** relaxes totality: some pairs of events are incomparable (concurrent).

**Partition-scoped total order** (Kafka model): Events within a single partition are totally ordered. Events across partitions have no ordering guarantee. For a topic with $P$ partitions:

- Within partition $p_i$: events $e_1 < e_2 < \ldots < e_n$ (total order by offset)
- Across partitions $p_i, p_j$: events are concurrent unless linked by causal dependency

**Trade-off**: A single partition provides total order but limits throughput to one consumer. $P$ partitions provide $P\times$ throughput but only per-partition ordering.

**Causal ordering** provides a middle ground: if event $A$ caused event $B$ (e.g., $B$ was produced after consuming $A$), then all consumers see $A$ before $B$. This requires tracking causal dependencies via vector clocks or similar mechanisms.

### Worked Examples

Order lifecycle events: OrderCreated, PaymentCharged, OrderShipped.

Using partition key = order ID: All events for order-123 go to the same partition, preserving the causal chain. Different orders (order-123, order-456) may be on different partitions — their relative order is undefined but irrelevant.

Pitfall: If OrderCreated goes to partition 0 and PaymentCharged goes to partition 1 (wrong partition key), a consumer might see PaymentCharged before OrderCreated. Use consistent partition keys to prevent this.

---

## 2. Lamport Timestamps for Causal Ordering (Event Systems)

### The Problem

How do we maintain causal ordering of events across services without a centralized sequencer?

### The Formula

Each service $S_i$ maintains a logical clock $L_i$. Event production rules:

1. Before producing event $e$: $L_i := L_i + 1$, attach $L_i$ as timestamp
2. When consuming event with timestamp $t$: $L_i := \max(L_i, t) + 1$

For two events $a, b$: if $L(a) < L(b)$, then either $a$ causally precedes $b$ or they are concurrent. If $a$ causally precedes $b$, then $L(a) < L(b)$ is guaranteed.

**Lamport total order**: Break ties using process ID: $(L(a), \text{pid}_a) < (L(b), \text{pid}_b)$ iff $L(a) < L(b)$ or ($L(a) = L(b)$ and $\text{pid}_a < \text{pid}_b$).

**Limitation**: Lamport clocks cannot determine if two events are concurrent. For that, use vector clocks: $V_a[i]$ = logical time of process $i$ as known by the producer of event $a$.

$$a \rightarrow b \iff V(a) < V(b) \quad \text{(componentwise less)}$$
$$a \parallel b \iff \neg(V(a) < V(b)) \land \neg(V(b) < V(a))$$

### Worked Examples

Three services: OrderService (OS), PaymentService (PS), ShippingService (SS).

1. OS produces OrderCreated, $L_{OS} = 1$, event timestamp $= 1$
2. PS consumes OrderCreated ($t=1$), sets $L_{PS} = \max(0, 1) + 1 = 2$
3. PS produces PaymentCharged, $L_{PS} = 3$, event timestamp $= 3$
4. SS consumes PaymentCharged ($t=3$), sets $L_{SS} = \max(0, 3) + 1 = 4$
5. SS produces OrderShipped, $L_{SS} = 5$, event timestamp $= 5$

Ordering: OrderCreated($1$) < PaymentCharged($3$) < OrderShipped($5$). Causal chain preserved. If InventoryService independently produces StockUpdated with $L_{IS} = 2$, this is concurrent with PaymentCharged despite $L = 2 < 3$; Lamport clocks cannot distinguish "before" from "concurrent" here.

---

## 3. Idempotency Mathematics (Function Composition)

### The Problem

When exactly-once delivery is impossible, we need idempotent processing. What makes a function idempotent, and how do we compose idempotent operations?

### The Formula

A function $f: S \rightarrow S$ is **idempotent** if:

$$f(f(x)) = f(x) \quad \forall x \in S$$

Equivalently, $f \circ f = f$. Applying $f$ any number of times yields the same result as applying it once.

**Composition of idempotent functions**: If $f$ and $g$ are both idempotent, $g \circ f$ is **not necessarily idempotent**.

Counterexample: Let $f(x) = 1$ and $g(x) = 2$. Both are idempotent ($f(f(x)) = f(1) = 1$, $g(g(x)) = g(2) = 2$). But $h = g \circ f$: $h(x) = 2$, $h(h(x)) = h(2) = g(f(2)) = g(1) = 2$. In this case $h$ happens to be idempotent. But consider $f(x) = x + 1 \bmod 3$ with $g(x) = \min(x, 1)$. $g$ is idempotent, $f$ is not. Composing non-idempotent $f$ with idempotent $g$ can yield non-idempotent results.

**Making operations idempotent**: Transform $f(x) = x + \delta$ (non-idempotent) into $f_k(x) = \text{if } k \notin \text{seen}: x + \delta, \text{mark } k$ (idempotent via deduplication with key $k$).

**Natural idempotency** of certain operations:
- SET x = 5 (idempotent: repeated application gives same result)
- DELETE WHERE id = 5 (idempotent: second delete is a no-op)
- INCREMENT x BY 1 (NOT idempotent: must use deduplication key)

### Worked Examples

Payment processing: `ChargeCard(amount=100, idempotencyKey="pay-abc")`.

First call: charges $100, stores `{key: "pay-abc", result: "success", txId: "tx-123"}`.
Retry (same key): looks up `pay-abc`, returns stored result without charging again.
Different key: `ChargeCard(amount=100, idempotencyKey="pay-def")` — new charge.

The deduplication store transforms a non-idempotent side effect (charging money) into an idempotent operation by memoizing results keyed by the idempotency key.

---

## 4. Stream Processing Windowing (Tumbling, Sliding, Session)

### The Problem

How do we aggregate events over time when events arrive out of order and processing is distributed?

### The Formula

**Tumbling window** of size $w$: Non-overlapping, fixed-size windows. Event with timestamp $t$ belongs to window $\lfloor t/w \rfloor$.

$$\text{Window}(t, w) = \left[\lfloor t/w \rfloor \cdot w, \quad \lfloor t/w \rfloor \cdot w + w\right)$$

**Sliding window** of size $w$ with slide $s$: Overlapping windows. Event at time $t$ belongs to multiple windows:

$$\text{Windows}(t, w, s) = \left\{ \left[k \cdot s, \quad k \cdot s + w\right) \mid k \cdot s \leq t < k \cdot s + w \right\}$$

Number of windows containing any event: $\lceil w/s \rceil$.

**Session window** with gap $g$: Dynamic windows based on activity. A session ends when no events arrive for duration $g$. If events arrive at times $t_1 < t_2 < \ldots < t_n$:

$$\text{New session at } t_i \iff t_i - t_{i-1} > g$$

**Watermark**: A monotonically increasing timestamp $W(t)$ asserting "no events with timestamp $< W(t)$ will arrive after processing time $t$." Windows can close when $W(t) \geq$ window end time.

**Allowed lateness** $\ell$: Events arriving after watermark but within $\ell$ are still processed (update previously emitted result). Events arriving after $W(t) + \ell$ are dropped.

### Worked Examples

Tumbling window, $w = 60s$: Counting page views per minute.

- Event at $t = 45s$ goes to window $[0, 60)$
- Event at $t = 61s$ goes to window $[60, 120)$
- Event at $t = 59s$ arriving at processing time $t_p = 65s$: watermark determines if it is late

Sliding window, $w = 300s, s = 60s$: "Page views in the last 5 minutes, updated every minute."

- Event at $t = 150s$ belongs to windows: $[0,300), [60,360), [120,420)$ — three windows
- Each window overlaps, providing a "rolling average" effect
- Memory: must maintain $\lceil 300/60 \rceil = 5$ concurrent windows

Session window, $g = 30$ minutes: User browsing sessions.

- Events at $t = 0, 5, 12, 28, 70$ (minutes): Session 1 = $[0, 28]$ (gaps $\leq 30$), Session 2 = $[70, 70]$ (gap of 42 > 30)

---

## 5. Exactly-Once Delivery Impossibility (Two Generals)

### The Problem

Can a sender guarantee that a receiver processes a message exactly once, even with unreliable networks?

### The Formula

**Two Generals' Problem**: Two generals must agree on attack time by sending messengers through enemy territory. Each messenger may be captured (message lost).

General A sends "attack at dawn." General B receives it and sends ACK. But B does not know if ACK was received. B sends another ACK. But A does not know if that ACK's ACK was received. This recursion is infinite.

**Theorem**: No finite protocol can guarantee agreement over an unreliable channel.

**Practical exactly-once**: Achieved by combining at-least-once delivery with idempotent processing:

$$\text{exactly-once} = \text{at-least-once delivery} + \text{idempotent consumer}$$

At-least-once: sender retries until ACK received (may deliver duplicates).
Idempotent consumer: processing a message multiple times produces the same result as processing it once.

**Kafka's exactly-once semantics**: Uses a combination of:
1. Idempotent producer (sequence numbers per partition)
2. Transactional writes (atomic multi-partition writes)
3. Consumer read-committed isolation

This achieves exactly-once within the Kafka ecosystem, but the guarantee ends at the Kafka boundary. External side effects (sending email, charging credit card) require application-level idempotency.

### Worked Examples

Without idempotency: Producer sends "credit $100 to account A." Network hiccup causes retry. Consumer processes twice: account A gets $200.

With idempotency: Producer sends "credit $100, txId=tx-789." Consumer checks: `tx-789` already processed? If yes, skip. If no, credit $100 and record `tx-789`. Retry is harmless.

Kafka transactional producer: Begins transaction, sends to partitions P1 and P2, commits. If commit fails, transaction is aborted and nothing is visible to consumers. If commit succeeds, both messages are atomically visible.

---

## Prerequisites

- Partial orders and lattice theory basics
- Logical clocks (Lamport, vector)
- Basic probability for out-of-order event analysis
- Understanding of distributed messaging systems

## Complexity

| Concept | Space | Time | Key Constraint |
|---|---|---|---|
| Lamport clock | $O(1)$ per event | $O(1)$ per event | Cannot detect concurrency |
| Vector clock | $O(n)$ per event | $O(n)$ merge | $n$ = number of producers |
| Tumbling window | $O(1)$ windows active | $O(1)$ per event | Fixed, non-overlapping |
| Sliding window | $O(w/s)$ windows active | $O(w/s)$ per event | Overlapping |
| Session window | $O(\text{active sessions})$ | $O(1)$ per event | Dynamic, gap-based |
| Dedup store | $O(k)$ for $k$ keys | $O(1)$ lookup | TTL-bounded keys |
