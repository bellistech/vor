# nslookup Deep Dive — Theory & Internals

> *nslookup is the simplest DNS query tool — a thin wrapper around DNS resolution with interactive and non-interactive modes. Understanding its output means understanding DNS query types, recursive vs authoritative responses, and the difference between canonical names and aliases.*

---

## 1. Resolution Path — Query Flow

### Non-Authoritative vs Authoritative

$$\text{Response type} = \begin{cases} \text{Authoritative} & \text{if server is NS for the zone} \\ \text{Non-authoritative} & \text{if server is a recursive resolver (cached)} \end{cases}$$

nslookup flags this in output:
- "Non-authoritative answer" = from cache (resolver)
- No flag = from authoritative nameserver directly

### Query to Specific Server

$$T_{response} = RTT_{to\_server} + T_{process}$$

When querying an authoritative server directly (e.g., `nslookup example.com ns1.example.com`), you bypass the resolver cache:

$$T_{authoritative} = RTT_{direct} + T_{zone\_lookup}$$

$$T_{recursive} = RTT_{resolver} + \max(0, T_{recursive\_chain})$$

---

## 2. Record Types — Query Taxonomy

### Common Queries

| Type | Query | Returns |
|:---|:---|:---|
| A | `nslookup example.com` | IPv4 address |
| AAAA | `nslookup -type=aaaa example.com` | IPv6 address |
| MX | `nslookup -type=mx example.com` | Mail exchangers + priorities |
| NS | `nslookup -type=ns example.com` | Nameservers |
| SOA | `nslookup -type=soa example.com` | Start of Authority |
| TXT | `nslookup -type=txt example.com` | Text records (SPF, DKIM) |
| CNAME | `nslookup -type=cname www.example.com` | Canonical name alias |
| PTR | `nslookup 93.184.216.34` | Reverse DNS |

### CNAME Chain Resolution

When a CNAME is encountered, nslookup follows the chain:

$$\text{www.example.com} \xrightarrow{CNAME} \text{cdn.example.net} \xrightarrow{A} \text{93.184.216.34}$$

Chain length adds latency:

$$T_{CNAME} = D_{chain} \times T_{query}$$

| Chain Depth | Additional Queries | Extra Latency (50 ms/query) |
|:---:|:---:|:---:|
| 1 | 1 | 50 ms |
| 2 | 2 | 100 ms |
| 3 | 3 | 150 ms |

Maximum chain depth: typically limited to 8-16 by resolvers to prevent infinite loops.

---

## 3. Reverse DNS — PTR Lookup Math

### The in-addr.arpa Construction

For IPv4 address $a.b.c.d$:

$$\text{PTR query} = d.c.b.a.\text{in-addr.arpa}$$

The octets are **reversed** because DNS reads right-to-left (most specific first).

### IPv6 Reverse DNS

For IPv6, each nibble (4 bits) becomes a label:

$$\text{2001:db8::1} \rightarrow \text{1.0.0.0...8.b.d.0.1.0.0.2.ip6.arpa}$$

Total labels: $32$ nibbles + `ip6.arpa` = 34 labels.

$$L_{PTR\_v6} = 32 \times 2 + 9 = 73 \text{ characters}$$

### Reverse DNS Completeness

Not all IPs have PTR records. Coverage rate:

$$C_{PTR} = \frac{N_{with\_PTR}}{N_{total\_IPs}}$$

Typical coverage: ~60-70% of publicly routable IPv4 addresses have PTR records.

---

## 4. Timeout and Retry Behavior

### Default Timing

$$T_{timeout} = T_{initial} \times 2^{n} \quad \text{(exponential backoff)}$$

nslookup defaults (platform-dependent):
- Initial timeout: 2-5 seconds
- Retries: 1-4 attempts
- Servers tried: all configured nameservers

### Total Worst-Case Wait

$$T_{worst} = N_{servers} \times R_{retries} \times T_{timeout}$$

| Servers | Retries | Timeout | Worst Case |
|:---:|:---:|:---:|:---:|
| 1 | 2 | 5 sec | 10 sec |
| 2 | 2 | 5 sec | 20 sec |
| 3 | 4 | 5 sec | 60 sec |

---

## 5. MX Priority — Weighted Mail Routing

### MX Record Selection

MX records include a preference value (lower = higher priority):

$$\text{Server selected} = \arg\min(\text{preference})$$

When multiple MX records share the same preference:

$$P(\text{server}_i) = \frac{1}{N_{same\_pref}}$$

### Failover Timing

$$T_{mail\_delivery} = T_{try\_primary} + \begin{cases} 0 & \text{if primary accepts} \\ T_{timeout} + T_{try\_secondary} & \text{if primary fails} \end{cases}$$

Typical MX configuration:

| Priority | Server | Role |
|:---:|:---|:---|
| 10 | mail1.example.com | Primary |
| 10 | mail2.example.com | Primary (load shared) |
| 20 | backup.example.com | Secondary |
| 30 | disaster.example.com | Tertiary |

---

## 6. SOA Record — Zone Timing Parameters

### SOA Fields

| Field | Example | Formula/Meaning |
|:---|:---:|:---|
| Serial | 2024010101 | $YYYYMMDDNN$ format |
| Refresh | 3600 | Seconds between slave checks |
| Retry | 900 | Seconds between failed retries |
| Expire | 604800 | Seconds until slave stops serving |
| Minimum TTL | 86400 | Negative cache TTL |

### Zone Transfer Timing

$$T_{stale} = T_{refresh} + T_{retry} \times R_{max} + T_{propagation}$$

If master fails, slaves serve stale data for:

$$T_{expire} = 604,800 \text{ sec} = 7 \text{ days (typical)}$$

---

## 7. nslookup vs dig vs host — Comparison

### Feature Matrix

| Feature | nslookup | dig | host |
|:---|:---:|:---:|:---:|
| Default output | Simplified | Full sections | Concise |
| DNSSEC display | No | Yes (+dnssec) | No |
| Batch mode | Interactive | File input | No |
| Timing info | No | Yes (Query time) | No |
| Trace mode | No | Yes (+trace) | No |
| JSON output | No | Yes (+json, newer) | No |

### When to Use Each

$$\text{Complexity needed} = \begin{cases} \text{Low (quick check)} & \rightarrow \text{host or nslookup} \\ \text{Medium (specific records)} & \rightarrow \text{nslookup -type=} \\ \text{High (debugging)} & \rightarrow \text{dig} \end{cases}$$

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $RTT + T_{process}$ | Summation | Query response time |
| $D_{chain} \times T_{query}$ | Product | CNAME chain latency |
| $d.c.b.a.\text{in-addr.arpa}$ | Reversal | PTR query construction |
| $N_{servers} \times R \times T_{timeout}$ | Product | Worst-case wait |
| $\arg\min(\text{preference})$ | Minimum selection | MX server choice |

## Prerequisites

- DNS record types, domain hierarchy, basic query resolution

---

*nslookup is the quick-and-dirty DNS tool that every sysadmin reaches for first. It won't show you DNSSEC chains or query timing, but for the 90% case — "does this name resolve, and to what?" — it's the fastest path from question to answer.*
