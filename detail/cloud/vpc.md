# The Mathematics of VPC — Network Segmentation and Address Space Design

> *A Virtual Private Cloud is an exercise in combinatorial network design: partitioning finite address spaces into subnets, computing routing decisions via longest-prefix match, and modeling security as layered packet filters with stateful and stateless evaluation.*

---

## 1. CIDR Address Space Partitioning (Combinatorics)

### The Address Space

An IPv4 CIDR block $a.b.c.d/n$ defines:

$$\text{Total addresses} = 2^{32-n}$$

$$\text{Usable hosts} = 2^{32-n} - k$$

Where $k$ accounts for reserved addresses (AWS reserves 5 per subnet).

### Subnet Subdivision

Given a VPC CIDR $/n$, splitting into subnets of size $/m$ where $m > n$:

$$\text{Number of subnets} = 2^{m-n}$$

$$\text{Hosts per subnet} = 2^{32-m} - 5 \quad \text{(AWS)}$$

### Worked Example: 10.0.0.0/16

| Subnet Size | Subnets Available | Hosts per Subnet | Total Usable Hosts |
|:---:|:---:|:---:|:---:|
| /20 | $2^{20-16} = 16$ | $2^{12} - 5 = 4,091$ | 65,456 |
| /24 | $2^{24-16} = 256$ | $2^{8} - 5 = 251$ | 64,256 |
| /26 | $2^{26-16} = 1,024$ | $2^{6} - 5 = 59$ | 60,416 |
| /28 | $2^{28-16} = 4,096$ | $2^{4} - 5 = 11$ | 45,056 |

### Address Efficiency

$$\eta = \frac{\sum_{i} \text{UsableHosts}(s_i)}{2^{32-n}} = 1 - \frac{k \times |\text{subnets}|}{2^{32-n}}$$

For a /16 VPC with 256 /24 subnets:

$$\eta = 1 - \frac{5 \times 256}{65536} = 1 - 0.0195 = 98.05\%$$

---

## 2. Routing Decision Theory (Longest Prefix Match)

### Route Selection Algorithm

Given a destination IP $d$ and route table entries $\{(C_i, T_i)\}$:

$$\text{NextHop}(d) = T_j \text{ where } j = \arg\max_{i: d \in C_i} |C_i.\text{prefix}|$$

This is the longest prefix match (LPM) algorithm.

### Route Priority

$$\text{Specificity}(C) = \text{prefix\_length}(C)$$

More specific routes always win:

$$10.0.1.0/24 \succ 10.0.0.0/16 \succ 0.0.0.0/0$$

### Route Table Evaluation Complexity

For $r$ routes in a table:

$$T_{lookup} = O(\log r) \quad \text{(trie-based)}$$

In practice, cloud route tables are small ($r < 100$), making lookup effectively $O(1)$.

### Route Propagation

With VPN/Direct Connect route propagation, the effective route table:

$$R_{eff} = R_{static} \cup R_{propagated}$$

Static routes take precedence over propagated routes for identical prefixes.

---

## 3. Security Group Evaluation (Stateful Packet Filter)

### Connection Tracking Model

A security group maintains a connection table $CT$:

$$CT = \{(src, dst, proto, sport, dport, state, ttl)\}$$

### Inbound Rule Evaluation

$$\text{Allow}_{in}(pkt) = \exists r \in R_{in}: \text{match}(pkt, r) \vee \exists c \in CT: \text{isReturn}(pkt, c)$$

The stateful property means return traffic is automatically allowed.

### Rule Matching Function

For a rule $r = (proto, port_{min}, port_{max}, cidr)$:

$$\text{match}(pkt, r) = (pkt.proto = r.proto) \wedge (pkt.dport \in [r.port_{min}, r.port_{max}]) \wedge (pkt.src \in r.cidr)$$

### Security Group Composition

When multiple SGs are attached to an ENI, rules are unioned:

$$R_{effective} = \bigcup_{sg \in \text{attached}} R(sg)$$

This is permissive composition — any matching rule in any SG allows the traffic.

---

## 4. Network ACL Evaluation (Stateless Ordered Filter)

### Rule Processing

NACLs evaluate rules in order by rule number:

$$\text{Decision}(pkt) = \text{action}(r_j) \text{ where } j = \min\{i : \text{match}(pkt, r_i)\}$$

Lower rule numbers are evaluated first. First match wins.

### Stateless Implication

Both directions must be explicitly allowed:

$$\text{Allowed}(flow) = \text{Allow}_{in}(pkt_{req}) \wedge \text{Allow}_{out}(pkt_{resp})$$

### Ephemeral Port Problem

For TCP responses, the client uses an ephemeral port:

$$port_{ephemeral} \in [1024, 65535]$$

NACLs must allow this entire range for return traffic.

### NACL vs Security Group Comparison

| Property | Security Group | NACL |
|:---|:---|:---|
| State | Stateful | Stateless |
| Default | Deny all in, allow all out | Allow all |
| Rule eval | All rules (union) | Ordered (first match) |
| Scope | ENI | Subnet |
| Return traffic | Automatic | Explicit rule needed |
| Complexity | $O(|R|)$ per packet | $O(|R|)$ worst case |

---

## 5. NAT Gateway Throughput (Queueing Theory)

### Port Allocation

A NAT gateway maps private addresses to a single public IP:

$$\text{Max connections} = 65535 - 1024 = 64511 \text{ per destination}$$

For $d$ unique destinations:

$$\text{Max total connections} = 64511 \times d$$

### Throughput Model

NAT gateway capacity:

$$BW_{nat} = \min(BW_{instance}, BW_{nat\_limit})$$

AWS NAT gateway: 45 Gbps burst, sustained based on connection count.

### Cost Model

$$C_{nat} = C_{hourly} \times H + C_{per\_GB} \times D_{processed}$$

For $H = 730$ hours/month and $D = 1$ TB:

$$C_{nat} = 0.045 \times 730 + 0.045 \times 1024 = \$32.85 + \$46.08 = \$78.93$$

### VPC Endpoint Savings

Gateway endpoints (S3, DynamoDB) bypass NAT:

$$\text{Savings} = D_{s3} \times C_{per\_GB} = D_{s3} \times \$0.045/\text{GB}$$

---

## 6. VPC Peering and Transit Gateway (Graph Theory)

### Peering Topology

VPC peering creates an undirected graph $G = (V, E)$ where $|V|$ = number of VPCs.

Full mesh peering:

$$|E_{mesh}| = \binom{|V|}{2} = \frac{|V|(|V|-1)}{2}$$

| VPCs | Full Mesh Connections | Transit Gateway Attachments |
|:---:|:---:|:---:|
| 3 | 3 | 3 |
| 5 | 10 | 5 |
| 10 | 45 | 10 |
| 50 | 1,225 | 50 |

### Non-Transitivity Constraint

VPC peering is non-transitive:

$$\text{Peer}(A, B) \wedge \text{Peer}(B, C) \not\Rightarrow \text{Connectivity}(A, C)$$

Transit Gateway solves this:

$$\forall v_i, v_j \in V: \text{Connected}(v_i, v_j) \iff \text{Attached}(v_i, TGW) \wedge \text{Attached}(v_j, TGW)$$

### CIDR Overlap Detection

Peering requires non-overlapping CIDRs:

$$\forall (v_i, v_j) \in E: \text{CIDR}(v_i) \cap \text{CIDR}(v_j) = \emptyset$$

Two CIDRs overlap iff:

$$\text{Overlap}(C_1, C_2) = (\text{base}(C_1) < \text{end}(C_2)) \wedge (\text{base}(C_2) < \text{end}(C_1))$$

---

## 7. Flow Log Analysis (Statistical Sampling)

### Flow Record Format

Each record captures a 5-tuple aggregation window:

$$R = (src, dst, srcPort, dstPort, proto, packets, bytes, action, start, end)$$

### Traffic Volume Estimation

Total bytes over time window $[t_1, t_2]$:

$$B_{total} = \sum_{r: r.start \geq t_1 \wedge r.end \leq t_2} r.bytes$$

### Anomaly Detection

Using standard deviation over time windows:

$$\text{Anomaly}(w) = |B(w) - \mu| > k\sigma$$

Where $\mu$ and $\sigma$ are computed over the trailing $N$ windows, and $k$ is typically 2 or 3.

---

*VPC design is constrained optimization: maximize usable hosts within address space limits, minimize attack surface through layered security rules, and balance cost against throughput at NAT boundaries. Every routing decision, security evaluation, and peering connection follows deterministic mathematical rules.*

## Prerequisites

- CIDR notation and IPv4 subnetting
- Set theory and boolean logic
- Graph theory (undirected graphs, connectivity)
- Basic queueing theory for throughput analysis

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| CIDR containment check | $O(1)$ | $O(1)$ |
| Longest prefix match | $O(\log r)$ | $O(r)$ |
| Security group evaluation | $O(|R|)$ | $O(|CT|)$ |
| NACL evaluation | $O(|R|)$ worst | $O(1)$ |
| Peering overlap detection | $O(|V|^2)$ | $O(|V|)$ |
| Flow log aggregation | $O(n)$ records | $O(n)$ |
