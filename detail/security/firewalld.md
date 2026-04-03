# The Mathematics of firewalld — Zone-Based Packet Filtering

> *firewalld implements zone-based network security through nftables/iptables. The mathematics involve set membership testing for zone classification, rule evaluation ordering for packet processing, and connection tracking state machines for stateful inspection.*

---

## 1. Zone Model — Set Partitioning

### Zone as Network Partition

firewalld assigns every network interface to exactly one zone, partitioning the network:

$$\text{Zones} = \{Z_1, Z_2, \ldots, Z_n\} \quad \text{where} \quad \bigcap_{i \neq j} Z_i \cap Z_j = \emptyset$$

Each zone defines an allowed service set:

$$\text{Zone}(Z) = (\text{interfaces}(Z), \text{services}(Z), \text{ports}(Z), \text{rich rules}(Z), \text{target}(Z))$$

### Default Zones

| Zone | Default Target | Typical Use | Trust Level |
|:---|:---:|:---|:---:|
| drop | DROP | Untrusted (no response) | 0 |
| block | REJECT | Untrusted (ICMP reject) | 1 |
| public | default (reject) | Internet-facing | 2 |
| external | default | NAT gateway | 3 |
| dmz | default | DMZ servers | 4 |
| work | default | Office network | 5 |
| home | default | Home network | 6 |
| internal | default | Internal LAN | 7 |
| trusted | ACCEPT | Full trust | 8 |

### Zone Selection Algorithm

For an incoming packet on interface $i$ from source $s$:

$$Z(p) = \begin{cases} Z_s & \text{if source } s \in \text{sources}(Z_s) \text{ for some } Z_s \\ Z_i & \text{if interface } i \in \text{interfaces}(Z_i) \\ Z_{default} & \text{otherwise} \end{cases}$$

Source-based zones take priority over interface-based zones.

---

## 2. Packet Processing Pipeline

### Rule Evaluation Order

firewalld generates nftables/iptables rules in a specific order:

| Priority | Rule Type | Action |
|:---:|:---|:---|
| 1 | Direct rules (legacy) | User-defined raw rules |
| 2 | Zone source matching | Assign zone by source |
| 3 | Zone interface matching | Assign zone by interface |
| 4 | Zone services/ports | Allow matching services |
| 5 | Zone rich rules | Complex allow/deny/log |
| 6 | Zone target | Default action for zone |

### Processing Cost

$$T_{packet} = T_{conntrack} + T_{zone\_match} + T_{rule\_eval}$$

With connection tracking (stateful):

$$T_{packet} = \begin{cases} T_{conntrack} & \text{established connection (fast path)} \\ T_{conntrack} + T_{full\_eval} & \text{new connection (slow path)} \end{cases}$$

| Path | Typical Latency | Percentage of Traffic |
|:---|:---:|:---:|
| Established (fast) | 0.5-2 $\mu$s | 95-99% |
| New connection (slow) | 5-50 $\mu$s | 1-5% |
| Invalid packet | 1-5 $\mu$s | <1% |

---

## 3. Connection Tracking — State Machine

### TCP State Tracking

The conntrack module tracks TCP connections through states:

```
NEW → SYN sent
ESTABLISHED → SYN+ACK received
RELATED → FTP data, ICMP error
INVALID → Malformed/unexpected
```

### Conntrack Table Sizing

$$\text{Max entries} = \text{nf\_conntrack\_max}$$

Default: 65,536 (may need tuning for busy servers).

Memory per entry: ~350 bytes (nf_conntrack struct).

$$\text{Memory} = \text{max entries} \times 350 \text{ bytes}$$

| Max Entries | Memory | Concurrent Connections |
|:---:|:---:|:---:|
| 65,536 | 23 MB | ~65K |
| 262,144 | 92 MB | ~262K |
| 1,048,576 | 367 MB | ~1M |

### Hash Table Lookup

Conntrack uses a hash table:

$$T_{lookup} = O(1) \text{ average, } O(n/\text{buckets}) \text{ worst case}$$

Default bucket count = max_entries / 4. With even distribution:

$$\text{Chain length} = \frac{\text{entries}}{\text{buckets}} = 4 \text{ (average)}$$

---

## 4. Service Definition — Port Sets

### Service as Port Set

$$\text{Service}(s) = \{(\text{protocol}, \text{port}) : \text{defined in } s.\text{xml}\}$$

| Service | Ports | Protocols |
|:---|:---|:---|
| ssh | 22 | TCP |
| http | 80 | TCP |
| https | 443 | TCP |
| dns | 53 | TCP, UDP |
| smtp | 25 | TCP |
| kerberos | 88, 749 | TCP, UDP |

### Zone Service Count

$$|\text{allowed connections}(Z)| = \sum_{s \in \text{services}(Z)} |s.\text{ports}|$$

| Zone Config | Services | Open Ports | nftables Rules |
|:---|:---:|:---:|:---:|
| Minimal (SSH only) | 1 | 1 | ~5 |
| Web server | 3 (ssh, http, https) | 3 | ~10 |
| Mail server | 5 | 8 | ~20 |
| Full service | 15 | 25+ | ~60 |

### Attack Surface by Zone

$$A(Z) = \frac{|\text{open ports}(Z)|}{65535} \times 100\%$$

| Open Ports | Attack Surface | Classification |
|:---:|:---:|:---|
| 1 | 0.0015% | Minimal |
| 5 | 0.0076% | Low |
| 20 | 0.031% | Medium |
| 100 | 0.153% | High |

---

## 5. Rich Rules — Complex Filtering

### Rich Rule Grammar

Rich rules support source/destination filtering with logging and rate limiting:

```
rule family="ipv4"
  source address="10.0.0.0/8"
  service name="ssh"
  log prefix="SSH-ACCESS" level="info"
  limit value="3/m"
  accept
```

### Rate Limiting Mathematics

The `limit` directive uses a token bucket:

$$\text{Allowed rate} = \frac{\text{value}}{\text{duration}}$$

| Limit | Rate | Tokens/Second |
|:---|:---:|:---:|
| 1/s | 1 per second | 1.0 |
| 3/m | 3 per minute | 0.05 |
| 10/h | 10 per hour | 0.0028 |
| 100/d | 100 per day | 0.0012 |

### Log Volume from Rich Rules

$$V_{log} = R_{matches} \times S_{entry} \times T$$

A rate-limited log at 3/minute:

$$V_{daily} = 3 \times 60 \times 24 \times 200\text{ bytes} = 864 \text{ KB/day}$$

Without rate limiting at 1000 matches/minute:

$$V_{daily} = 1000 \times 60 \times 24 \times 200 = 288 \text{ MB/day}$$

Rate limiting reduces log volume by **333x** in this example.

---

## 6. ICMP Type Filtering

### ICMP as Information Leak

Each allowed ICMP type reveals information:

| ICMP Type | Information Leaked | Risk |
|:---|:---|:---:|
| Echo Reply (0) | Host is alive | Low |
| Destination Unreachable (3) | Port/host status | Medium |
| Redirect (5) | Network topology | High |
| Time Exceeded (11) | Path (traceroute) | Medium |
| Timestamp Reply (14) | System clock | Low |
| Address Mask Reply (18) | Subnet mask | Medium |

### firewalld ICMP Policy

$$\text{Allowed ICMP} = \text{Zone ICMP set} \setminus \text{blocked ICMP types}$$

Recommended: allow types 0, 3, 11 (needed for path MTU discovery). Block all others.

---

## 7. Runtime vs Permanent — State Management

### Two-State Configuration

$$\text{State} = (\text{Runtime}, \text{Permanent})$$

| Operation | Runtime | Permanent | Survives Reload |
|:---|:---:|:---:|:---:|
| `--add-service` | Updated | Unchanged | No |
| `--add-service --permanent` | Unchanged | Updated | Yes (after reload) |
| `--add-service` + `--runtime-to-permanent` | Updated | Updated | Yes |

### Drift Detection

$$\text{Drift} = \text{Runtime config} \setminus \text{Permanent config}$$

If $|\text{Drift}| > 0$, a `firewall-cmd --reload` will lose runtime-only changes.

---

## 8. nftables Backend — Performance

### nftables vs iptables

| Feature | iptables | nftables |
|:---|:---|:---|
| Rule evaluation | Linear $O(n)$ | Sets + maps $O(1)$ |
| Atomic updates | No (sequential adds) | Yes (single transaction) |
| Rule count for 100 services | ~400 | ~50 |
| Reload time (1000 rules) | 2-5 seconds | 0.1-0.5 seconds |

### Set-Based Matching

nftables uses hash sets for port matching:

$$T_{match} = O(1) \text{ per packet (hash lookup)}$$

vs iptables linear chain:

$$T_{match} = O(n) \text{ where } n = \text{number of port rules}$$

For 100 services: nftables is ~100x faster per packet.

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Zone partitioning | Set partition ($\cap = \emptyset$) | Interface classification |
| Conntrack table | Hash table $O(1)$ | Stateful inspection |
| Rate limit (token bucket) | Rate function | Log and connection limiting |
| Attack surface $\%$ | Ratio | Port exposure |
| Rich rule evaluation | Sequential predicate | Complex filtering |
| nftables sets | Hash-based matching | High-performance lookup |

---

*firewalld is a zone-based abstraction over nftables/iptables — it translates human-readable service definitions into kernel-level packet filtering rules, evaluated millions of times per second at near-zero latency through connection tracking and hash-based set matching.*
