# The Mathematics of UFW — Uncomplicated Firewall

> *UFW is a frontend for iptables/nftables that simplifies rule management. Behind its simple interface, the mathematics of sequential rule evaluation, connection tracking, and rate limiting still apply — UFW just generates the underlying netfilter rules automatically.*

---

## 1. Rule Evaluation — Sequential First-Match

### Rule Chain

UFW generates iptables rules evaluated sequentially:

$$\text{Result}(p) = r_k.\text{action} \quad \text{where } k = \min\{i : r_i.\text{matches}(p)\}$$

The first matching rule determines the packet's fate.

### Rule Count and Performance

$$T_{eval} = O(k) \quad \text{where } k = \text{index of matching rule}$$

| Rules | Avg Eval (even distribution) | Worst Case |
|:---:|:---:|:---:|
| 10 | 5 comparisons | 10 |
| 50 | 25 comparisons | 50 |
| 100 | 50 comparisons | 100 |
| 500 | 250 comparisons | 500 |

Each comparison: ~0.1-0.5 $\mu$s. At 500 rules worst case: ~250 $\mu$s per packet.

### Default Policy

$$\text{default}(d) = \begin{cases} \text{deny (incoming)} & \text{standard — whitelist model} \\ \text{allow (outgoing)} & \text{standard — permissive egress} \\ \text{deny (routed)} & \text{standard — no forwarding} \end{cases}$$

The default policy is the implicit last rule: $r_{n+1} = \text{default action}$.

---

## 2. Connection Tracking States

### Stateful Rules

UFW leverages conntrack for stateful filtering:

$$\text{fast\_path}(p) = \begin{cases} \text{ACCEPT} & \text{if conntrack state} = \text{ESTABLISHED|RELATED} \\ \text{evaluate rules} & \text{if conntrack state} = \text{NEW} \end{cases}$$

This means ~95-99% of packets skip rule evaluation entirely.

### Effective Rule Evaluation

$$T_{avg} = P(\text{new}) \times T_{rules} + P(\text{established}) \times T_{conntrack}$$

Where $P(\text{new}) \approx 0.02$ and $P(\text{established}) \approx 0.98$:

$$T_{avg} = 0.02 \times 25\mu s + 0.98 \times 1\mu s = 1.48 \mu s$$

Even with 50 rules, the effective per-packet cost is ~1.5 $\mu$s.

---

## 3. Rate Limiting

### UFW Limit Rule

`ufw limit ssh` creates a rate-limiting rule:

$$\text{Rate} = 6 \text{ connections per 30 seconds per source IP}$$

$$R_{max} = \frac{6}{30} = 0.2 \text{ connections/second} = 12 \text{ connections/minute}$$

### Implementation: hashlimit

UFW's limit uses iptables `hashlimit`:

$$\text{Token bucket: } \frac{6 \text{ tokens}}{30s} \text{ with burst } = 6$$

| Attempt Pattern | Result |
|:---|:---|
| 6 rapid connections | All allowed (burst consumed) |
| 7th connection within 30s | Dropped |
| Steady 1 every 5s | All allowed (within rate) |
| Burst of 12 in 10s | 6 allowed, 6 dropped |

### Brute Force Impact

Without rate limit (attacker at 10/s):

$$\text{Attempts/hour} = 36{,}000$$

With rate limit (12/min):

$$\text{Attempts/hour} = 720$$

$$\text{Slowdown} = \frac{36{,}000}{720} = 50\times$$

---

## 4. Port and Protocol Mathematics

### TCP/UDP Port Space

$$|\text{Port space}| = 65535 \text{ per protocol} = 131{,}070 \text{ total (TCP + UDP)}$$

### Attack Surface Calculation

$$A = \frac{|\text{open ports}|}{|\text{total ports}|} \times 100\%$$

| Configuration | Open Ports | Attack Surface |
|:---|:---:|:---:|
| Default deny, SSH only | 1 | 0.0015% |
| Web server (22, 80, 443) | 3 | 0.0046% |
| Mail server (5 services) | 8 | 0.012% |
| Development machine | 20 | 0.031% |
| No firewall | 65,535 | 100% |

### Service Exposure Matrix

Each allowed rule creates an exposure:

$$E_{rule} = |\text{source IPs}| \times |\text{dest ports}|$$

| Rule | Source IPs | Ports | Exposure |
|:---|:---:|:---:|:---:|
| `allow ssh` | $2^{32}$ | 1 | $4.3 \times 10^9$ |
| `allow from 10.0.0.0/8 to any port 22` | $2^{24}$ | 1 | $1.7 \times 10^7$ |
| `allow from 10.0.0.5 to any port 22` | 1 | 1 | 1 |

Restricting source IP reduces exposure by $256\times$ per subnet bit.

---

## 5. Application Profiles

### Profile Definition

UFW application profiles define service port sets:

$$\text{App}(a) = \{(\text{proto}_i, \text{port}_i) : i = 1, \ldots, k\}$$

| Application | Ports | Protocol |
|:---|:---|:---|
| OpenSSH | 22 | TCP |
| Apache Full | 80, 443 | TCP |
| Nginx Full | 80, 443 | TCP |
| Postfix | 25 | TCP |
| IMAP/POP3 Secure | 993, 995 | TCP |
| DNS | 53 | TCP/UDP |

### Rule Consolidation

Without profiles: $n$ individual port rules.
With profiles: 1 application rule.

$$\text{Rules}_{without} = \sum_{a \in \text{apps}} |a.\text{ports}|$$
$$\text{Rules}_{with} = |\text{apps}|$$

For a web + mail server: 8 port rules vs 2 application rules.

---

## 6. IPv6 Considerations

### Dual-Stack Rule Generation

UFW generates rules for both IPv4 and IPv6:

$$\text{Total rules} = 2 \times |\text{user rules}| + \text{framework rules}$$

| User Rules | IPv4 iptables | IPv6 ip6tables | Total |
|:---:|:---:|:---:|:---:|
| 5 | ~15 | ~15 | ~30 |
| 20 | ~50 | ~50 | ~100 |
| 50 | ~120 | ~120 | ~240 |

### IPv6 Address Space

$$|\text{IPv6}| = 2^{128} = 3.4 \times 10^{38}$$

Scanning an IPv6 /64 subnet at 1 billion probes/second:

$$T_{scan} = \frac{2^{64}}{10^9} = 1.8 \times 10^{10} \text{ seconds} = 585 \text{ years}$$

IPv6 makes network scanning impractical — but known addresses are still reachable.

---

## 7. Logging and Analysis

### Log Levels

| Level | Logged Events | Volume |
|:---|:---|:---|
| off | Nothing | 0 |
| low | Blocked packets (rate-limited) | Low |
| medium | Blocked + allowed (non-matching) | Medium |
| high | All blocked + all allowed | High |
| full | Everything including rate-limited | Very high |

### Log Volume Estimation

$$V_{log} = R_{events} \times S_{entry} \times T$$

| Level | Events/sec (typical server) | Daily Volume |
|:---|:---:|:---:|
| low | 1-10 | 20-200 MB |
| medium | 10-100 | 200 MB - 2 GB |
| high | 100-1000 | 2-20 GB |
| full | 1000+ | 20+ GB |

### Log Analysis: Top Blocked Sources

$$\text{Threat}(IP) = \frac{\text{blocks from } IP}{\text{total blocks}} \times 100\%$$

Typically follows a Pareto distribution: 20% of IPs cause 80% of blocked traffic.

---

## 8. UFW Rule Ordering Strategy

### Optimal Order

For minimum average evaluation cost:

1. Rate limits first (catch brute force before service check)
2. Deny rules for known-bad sources
3. Allow rules in frequency order (most-hit rules first)
4. Default deny (implicit last)

$$T_{avg} = \sum_{i=1}^{n} p_i \times i \times T_{compare}$$

Placing the most frequently matched rule first minimizes expected evaluation steps.

### Example

3 rules with match probabilities 0.7, 0.2, 0.1:

Optimal: $T_{avg} = 0.7(1) + 0.2(2) + 0.1(3) = 1.4$
Worst: $T_{avg} = 0.1(1) + 0.2(2) + 0.7(3) = 2.6$

Optimal ordering is 46% faster.

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| First-match $r_k$ | Sequential search | Rule evaluation |
| Conntrack fast path | Conditional probability | 95%+ packet bypass |
| Token bucket 6/30s | Rate function | Brute force limiting |
| $|\text{open}|/65535$ | Ratio | Attack surface |
| $2^{128}$ IPv6 | Exponential | Scan infeasibility |
| $\sum p_i \times i$ | Expected value | Rule ordering |

---

*UFW makes iptables accessible without sacrificing the underlying mathematics — every `ufw allow` command generates precisely structured netfilter rules that execute in the kernel at line rate, evaluated millions of times per second with sub-microsecond latency.*
