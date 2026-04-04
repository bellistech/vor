# The Mathematics of Fail2ban — Rate-Based Intrusion Prevention

> *Fail2ban is a statistical anomaly detector: it monitors log files for authentication failure patterns and triggers firewall bans based on threshold-exceeding event rates. The mathematics involve counting processes, time windows, and the probability of banning legitimate users versus attackers.*

---

## 1. Ban Decision Function

### Core Algorithm

Fail2ban bans an IP when failure count exceeds a threshold within a time window:

$$\text{Ban}(IP) = \begin{cases} 1 & \text{if } \sum_{t-\text{findtime}}^{t} F(IP) \geq \text{maxretry} \\ 0 & \text{otherwise} \end{cases}$$

Where $F(IP)$ is the failure count for that IP in the window.

### Default Parameters

| Parameter | Default | Description |
|:---|:---:|:---|
| maxretry | 5 | Failures before ban |
| findtime | 600 s (10 min) | Sliding window |
| bantime | 600 s (10 min) | Ban duration |

### Effective Ban Rate

$$R_{ban} = \frac{\text{maxretry}}{\text{findtime}} = \frac{5}{600} = 0.0083 \text{ failures/sec}$$

Any IP exceeding 0.5 failures per minute gets banned.

---

## 2. Brute Force Attenuation

### Without Fail2ban

An attacker trying passwords at rate $r$ attempts/second:

$$\text{Attempts in time } T = r \times T$$

| Rate | 1 hour | 1 day | 1 week |
|:---:|:---:|:---:|:---:|
| 1/sec | 3,600 | 86,400 | 604,800 |
| 10/sec | 36,000 | 864,000 | 6,048,000 |
| 100/sec | 360,000 | 8,640,000 | 60,480,000 |

### With Fail2ban

After `maxretry` failures, the attacker is banned for `bantime`. Effective rate:

$$R_{effective} = \frac{\text{maxretry}}{\text{findtime} + \text{bantime}}$$

With defaults: $R = \frac{5}{600 + 600} = 0.00417$ attempts/second = **15 per hour**.

| bantime | Effective Rate | Attempts/Day | Slowdown Factor |
|:---:|:---:|:---:|:---:|
| 10 min | 15/hr | 360 | 240x |
| 1 hour | 1.4/hr | 33 | 2,618x |
| 24 hours | 0.06/hr | 1.4 | 61,714x |
| Permanent | maxretry total | 5 | $\infty$ |

### Time to Crack with Fail2ban

For a password with entropy $H$ bits:

$$T_{crack} = \frac{2^{H-1}}{R_{effective}}$$

| Password Entropy | Without F2B (10/s) | With F2B (15/hr) |
|:---:|:---:|:---:|
| 20 bits | 14.5 hours | 1,593 years |
| 30 bits | 621 days | 1.6M years |
| 40 bits | 1,742 years | 1.6B years |

---

## 3. False Positive Analysis

### Legitimate User Ban Probability

A legitimate user who mistype their password $k$ times:

$$P(\text{ban}) = P(k \geq \text{maxretry within findtime})$$

If a user makes $n$ login attempts per day with typo probability $p$:

$$P(\text{ban in one findtime window}) = \sum_{k=\text{maxretry}}^{n_{window}} \binom{n_{window}}{k} p^k (1-p)^{n_{window}-k}$$

### Worked Example

User logs in 5 times per 10-minute window, typo rate 10%, maxretry = 5:

$$P(\text{all 5 are typos}) = 0.1^5 = 10^{-5} = 0.001\%$$

$$P(\text{4 of 5 are typos}) = \binom{5}{4} \times 0.1^4 \times 0.9 = 0.00045 = 0.045\%$$

Total $P(\text{ban}) = P(k \geq 5) = 0.001\%$ — negligible for reasonable maxretry values.

### Optimal maxretry Selection

| maxretry | $P(\text{FP ban})$ (5 attempts, 10% typo) | Attacker Slowdown |
|:---:|:---:|:---:|
| 3 | 0.86% | 400x |
| 5 | 0.001% | 240x |
| 10 | $< 10^{-10}$ | 120x |
| 20 | $< 10^{-20}$ | 60x |

**Sweet spot:** maxretry = 5 gives negligible false positives with strong brute force protection.

---

## 4. Progressive Banning (ban time increment)

### Exponential Backoff

Fail2ban supports increasing ban times for repeat offenders:

$$\text{bantime}(n) = \text{bantime}_{base} \times \text{multiplier}^{n-1}$$

Where $n$ is the offense count.

| Offense | multiplier=2 | multiplier=4 |
|:---:|:---:|:---:|
| 1st | 10 min | 10 min |
| 2nd | 20 min | 40 min |
| 3rd | 40 min | 160 min (2.7 hr) |
| 4th | 80 min | 640 min (10.7 hr) |
| 5th | 160 min (2.7 hr) | 2560 min (42.7 hr) |
| 10th | 85.3 hr | 29,127 hr (3.3 yr) |

### Effective Attempts with Exponential Backoff

Total attempts after $n$ ban cycles:

$$A(n) = n \times \text{maxretry}$$

Total time:

$$T(n) = n \times \text{findtime} + \text{bantime}_{base} \times \frac{\text{multiplier}^n - 1}{\text{multiplier} - 1}$$

Effective rate drops rapidly:

$$R(n) = \frac{n \times \text{maxretry}}{T(n)} \xrightarrow{n \to \infty} 0$$

---

## 5. Multi-Jail Architecture

### Jail Configuration

Each service runs an independent jail:

| Jail | Log File | Typical Failures/Day | Filter Regex Patterns |
|:---|:---|:---:|:---:|
| sshd | `/var/log/auth.log` | 100-10,000 | 5-8 |
| apache-auth | access.log | 50-500 | 3-5 |
| postfix | mail.log | 500-5,000 | 4-6 |
| dovecot | mail.log | 100-1,000 | 3-4 |

### Total Ban Count

$$B_{total} = \sum_{j \in \text{jails}} B_j$$

Where $B_j$ is the number of active bans in jail $j$.

### Firewall Rule Count

Each ban creates a firewall rule:

$$\text{iptables rules} = \sum_{j} |B_j|$$

Performance degrades with rule count:

| Active Bans | iptables Lookup | ipset Lookup |
|:---:|:---:|:---:|
| 100 | ~10 $\mu$s | ~1 $\mu$s |
| 1,000 | ~100 $\mu$s | ~1 $\mu$s |
| 10,000 | ~1 ms | ~1 $\mu$s |
| 100,000 | ~10 ms | ~1 $\mu$s |

**ipset** uses hash tables: $O(1)$ lookup regardless of ban count.

---

## 6. Log Parsing Performance

### Regex Matching Cost

$$T_{parse} = n_{lines} \times n_{patterns} \times T_{regex}$$

Where $T_{regex}$ is the time per regex match (~1-10 $\mu$s).

| Log Lines/sec | Patterns | Parse Time/sec | CPU Usage |
|:---:|:---:|:---:|:---:|
| 10 | 5 | 0.25 ms | 0.025% |
| 100 | 5 | 2.5 ms | 0.25% |
| 1,000 | 10 | 50 ms | 5% |
| 10,000 | 10 | 500 ms | 50% |

At 10,000 log lines/second, fail2ban consumes significant CPU — consider rate limiting the log source or using compiled regex.

### Polling vs Inotify

| Method | Latency | CPU at Idle | CPU Under Load |
|:---|:---:|:---:|:---:|
| Polling (1s interval) | 0-1 second | Constant | Constant + parse |
| Inotify | < 10 ms | Near zero | Parse only |

Inotify is preferred: event-driven, lower latency, lower idle CPU.

---

## 7. Distributed Attack Resistance

### Single-IP Limitation

Fail2ban tracks per-IP. A distributed attack from $n$ IPs:

$$\text{Total rate} = n \times R_{effective} = n \times \frac{\text{maxretry}}{\text{findtime} + \text{bantime}}$$

| Attacker IPs | Attempts/Hour (default F2B) |
|:---:|:---:|
| 1 | 15 |
| 10 | 150 |
| 100 | 1,500 |
| 1,000 | 15,000 |
| 10,000 | 150,000 |

Against botnets with 10,000+ IPs, fail2ban alone is insufficient. Complementary defenses:
- Geographic IP blocking: reduces $n$ by blocking entire regions
- CAPTCHAs: adds per-attempt cost
- Account lockout: limits per-account regardless of source IP

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Threshold count $\geq$ maxretry | Counting process | Ban decision |
| $\frac{\text{maxretry}}{\text{findtime} + \text{bantime}}$ | Rate calculation | Effective attack speed |
| Binomial $P(k \geq m)$ | Probability distribution | False positive rate |
| $\text{base} \times m^{n-1}$ | Exponential growth | Progressive banning |
| ipset $O(1)$ vs iptables $O(n)$ | Algorithmic complexity | Firewall performance |
| $n \times R_{effective}$ | Linear scaling | Distributed attack rate |

## Prerequisites

- threshold arithmetic, exponential backoff, regular expressions, rate limiting

---

*Fail2ban is a simple but effective rate limiter — it cannot stop a determined attacker with a botnet, but it transforms brute force from a minutes-to-hours problem into a years-to-never problem for single-source attacks, all through basic threshold arithmetic.*
