# The Mathematics of DynamoDB — Partitioning, Throughput, and Consistency

> *DynamoDB distributes data across partitions using consistent hashing. The mathematics cover partition key distribution, read/write capacity unit calculations, GSI fan-out costs, hot key probability, and the consistency model's impact on read throughput.*

---

## 1. Consistent Hashing (Partition Assignment)

### The Problem

DynamoDB assigns items to partitions by hashing the partition key. Uniform distribution is critical; skewed keys create hot partitions that throttle requests.

### The Formula

Hash function maps key to a position on a ring of size $2^{128}$:

$$\text{partition}(k) = \text{MD5}(k) \bmod P$$

Where $P$ is the number of partitions. The expected items per partition:

$$E[\text{items per partition}] = \frac{N}{P}$$

Standard deviation (assuming uniform hashing):

$$\sigma = \sqrt{\frac{N}{P} \times \left(1 - \frac{1}{P}\right)} \approx \sqrt{\frac{N}{P}}$$

### Worked Example

10 million items across 50 partitions:

$$E = \frac{10^7}{50} = 200{,}000 \text{ items/partition}$$

$$\sigma = \sqrt{\frac{10^7}{50}} = \sqrt{200{,}000} \approx 447$$

99.7% of partitions will have between $200{,}000 \pm 1{,}341$ items (3-sigma).

---

## 2. Capacity Unit Calculations (Throughput)

### The Problem

Provisioned mode requires estimating Read Capacity Units (RCU) and Write Capacity Units (WCU). Each unit allows a specific number of operations per second.

### The Formula

Read Capacity Units (strongly consistent):

$$\text{RCU} = \left\lceil \frac{S_{\text{item}}}{4 \text{ KB}} \right\rceil \times R$$

Eventually consistent reads cost half:

$$\text{RCU}_{\text{eventual}} = \frac{1}{2} \left\lceil \frac{S_{\text{item}}}{4 \text{ KB}} \right\rceil \times R$$

Transactional reads cost double:

$$\text{RCU}_{\text{transact}} = 2 \left\lceil \frac{S_{\text{item}}}{4 \text{ KB}} \right\rceil \times R$$

Write Capacity Units:

$$\text{WCU} = \left\lceil \frac{S_{\text{item}}}{1 \text{ KB}} \right\rceil \times W$$

### Worked Examples

Items of 3.5 KB, 1000 reads/sec (strongly consistent), 200 writes/sec:

$$\text{RCU} = \left\lceil \frac{3.5}{4} \right\rceil \times 1000 = 1 \times 1000 = 1{,}000$$

$$\text{WCU} = \left\lceil \frac{3.5}{1} \right\rceil \times 200 = 4 \times 200 = 800$$

Monthly cost (us-east-1 pricing):

$$C_{\text{read}} = 1{,}000 \times \$0.00013/\text{hr} \times 720 = \$93.60$$

$$C_{\text{write}} = 800 \times \$0.00065/\text{hr} \times 720 = \$374.40$$

---

## 3. Partition Throughput Limits

### The Problem

Each partition supports a fixed throughput ceiling. When a single partition receives disproportionate traffic, it becomes a hot partition.

### The Formula

Per-partition limits:

$$\text{RCU}_{\text{partition}} = 3{,}000 \text{ RCU}$$
$$\text{WCU}_{\text{partition}} = 1{,}000 \text{ WCU}$$
$$\text{Size}_{\text{partition}} = 10 \text{ GB}$$

Number of partitions needed:

$$P = \max\left(\left\lceil \frac{\text{RCU}_{\text{total}}}{3{,}000} \right\rceil, \left\lceil \frac{\text{WCU}_{\text{total}}}{1{,}000} \right\rceil, \left\lceil \frac{S_{\text{total}}}{10 \text{ GB}} \right\rceil\right)$$

### Hot Key Probability

If one key receives fraction $f$ of total traffic on a table with $P$ partitions:

$$\text{Throttled if } f \times \text{Total WCU} > 1{,}000$$

$$f_{\text{max}} = \frac{1{,}000}{\text{Total WCU}}$$

For a table provisioned at 5,000 WCU:

$$f_{\text{max}} = \frac{1{,}000}{5{,}000} = 0.20$$

Any key receiving more than 20% of write traffic will cause throttling.

---

## 4. GSI Fan-Out Cost (Write Amplification)

### The Problem

Every write to a base table that affects a GSI attribute propagates to the GSI. This creates write amplification proportional to the number of GSIs.

### The Formula

Total write cost with $g$ GSIs:

$$\text{WCU}_{\text{effective}} = \text{WCU}_{\text{base}} + \sum_{i=1}^{g} \text{WCU}_{\text{GSI}_i}$$

If all writes affect all GSIs:

$$\text{WCU}_{\text{total}} = \text{WCU}_{\text{base}} \times (1 + g)$$

### Worked Example

Base table: 500 WCU, 3 GSIs, 80% of writes affect GSI-1, 50% affect GSI-2, 100% affect GSI-3:

$$\text{WCU}_{\text{total}} = 500 + 500 \times 0.8 + 500 \times 0.5 + 500 \times 1.0$$

$$= 500 + 400 + 250 + 500 = 1{,}650 \text{ WCU}$$

Write amplification factor:

$$A = \frac{1{,}650}{500} = 3.3\times$$

---

## 5. Query Cost and Scan Efficiency

### The Problem

Query operations read all items matching a partition key, then apply filter expressions. Filter expressions reduce response size but not RCU consumed.

### The Formula

RCU consumed by a query returning $n$ items of size $s_i$ with filter keeping fraction $f$:

$$\text{RCU}_{\text{consumed}} = \left\lceil \frac{\sum_{i=1}^{n} s_i}{4 \text{ KB}} \right\rceil$$

$$\text{RCU}_{\text{wasted}} = \text{RCU}_{\text{consumed}} \times (1 - f)$$

Scan efficiency for a full table scan with parallel segments $p$:

$$T_{\text{scan}} = \frac{S_{\text{table}}}{p \times \text{throughput per segment}}$$

### Worked Example

Query reads 200 items (1 KB each), filter keeps 20 items:

$$\text{RCU consumed} = \left\lceil \frac{200 \times 1}{4} \right\rceil = 50 \text{ RCU}$$

$$\text{RCU wasted} = 50 \times 0.9 = 45 \text{ RCU (90% wasted)}$$

This is why key conditions are critical: they determine what is read, while filters only reduce what is returned.

---

## 6. TTL Deletion Rate (Background Process)

### The Problem

DynamoDB TTL deletes expired items in the background, not instantly. Understanding the deletion rate helps plan for queries that may return expired items.

### The Formula

Expected deletion lag:

$$L_{\text{TTL}} \sim \text{Uniform}(0, 48 \text{ hours})$$

Items visible after TTL expiry, given arrival rate $\lambda$ items/sec:

$$N_{\text{expired but visible}} = \lambda \times E[L_{\text{TTL}}] = \lambda \times 24 \times 3600$$

### Worked Example

1000 items expire per hour:

$$N = \frac{1000}{3600} \times 24 \times 3600 = 24{,}000 \text{ ghost items at any time}$$

Applications must filter on TTL attribute:

$$\text{FilterExpression: } \text{ExpiresAt} > \text{current\_epoch}$$

---

## 7. On-Demand vs Provisioned (Cost Crossover)

### The Problem

On-demand pricing charges per request. Provisioned charges per hour for reserved capacity. The crossover point determines which mode is cheaper.

### The Formula

On-demand cost:

$$C_{\text{od}} = R \times P_r + W \times P_w$$

Provisioned cost (per month):

$$C_{\text{prov}} = \text{RCU} \times 720 \times P_{\text{rcu/hr}} + \text{WCU} \times 720 \times P_{\text{wcu/hr}}$$

Crossover when $C_{\text{od}} = C_{\text{prov}}$:

$$R_{\text{crossover}} = \frac{\text{RCU} \times 720 \times P_{\text{rcu/hr}}}{P_r}$$

### Worked Example

At us-east-1 pricing ($1.25/M reads, $0.25/M writes on-demand vs $0.00013/RCU-hr, $0.00065/WCU-hr):

For reads at 100 RCU provisioned:

$$C_{\text{prov}} = 100 \times 720 \times 0.00013 = \$9.36/\text{month}$$

Equivalent on-demand reads:

$$R = \frac{9.36}{1.25 \times 10^{-6}} = 7.49 \times 10^6 = 7.49M \text{ reads/month}$$

$$= \frac{7.49 \times 10^6}{30 \times 24 \times 3600} \approx 2.89 \text{ reads/sec}$$

If steady-state traffic exceeds ~2.9 reads/sec per RCU, provisioned mode is cheaper.

---

## Prerequisites

- hash-functions, probability, queuing-theory, distributed-systems
