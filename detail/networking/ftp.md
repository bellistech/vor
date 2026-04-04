# The Mathematics of FTP — Data Channel Dynamics & Transfer Optimization

> *FTP's dual-connection architecture creates a fascinating mathematical tension: the control channel is a finite state machine while the data channel is a bulk transfer pipe whose throughput depends on TCP window dynamics, NAT traversal probability, and port-range combinatorics.*

---

## 1. Active vs Passive Port Selection (Combinatorics)

### The Problem

In passive mode, the server selects a random port from a configured range for each data connection. If $N$ concurrent clients each need a data connection, what is the probability of a port collision when selecting from a range of $R$ ports?

### The Formula

This is the birthday problem. The probability that all $N$ clients get unique ports from a range of $R$:

$$P(\text{no collision}) = \prod_{i=0}^{N-1} \frac{R - i}{R} = \frac{R!}{R^N (R-N)!}$$

For large $R$ and moderate $N$, the approximation:

$$P(\text{collision}) \approx 1 - e^{-\frac{N(N-1)}{2R}}$$

The expected number of clients before the first collision:

$$E[N_{\text{collision}}] \approx \sqrt{\frac{\pi R}{2}}$$

### Worked Examples

**Example 1:** vsftpd configured with `pasv_min_port=40000`, `pasv_max_port=40100`, so $R = 101$ ports. With $N = 10$ concurrent connections:

$$P(\text{collision}) \approx 1 - e^{-\frac{10 \times 9}{2 \times 101}} = 1 - e^{-0.4455} = 1 - 0.6407 = 0.359$$

A 36% chance of collision with just 10 clients — dangerously narrow range.

**Example 2:** With $R = 10001$ (ports 40000-50000) and $N = 100$:

$$P(\text{collision}) \approx 1 - e^{-\frac{100 \times 99}{20002}} = 1 - e^{-0.4950} = 1 - 0.610 = 0.390$$

Expected clients before first collision:

$$E[N] \approx \sqrt{\frac{\pi \times 10001}{2}} \approx \sqrt{15709} \approx 125$$

---

## 2. FTP Transfer Throughput (TCP Dynamics)

### The Problem

FTP data connections are short-lived TCP connections. Each transfer starts with TCP slow start, which limits initial throughput. For a file of size $S$ bytes, what is the effective throughput considering slow start?

### The Formula

TCP slow start doubles the congestion window each RTT. After $k$ RTTs, the window is:

$$W_k = W_0 \cdot 2^k$$

Where $W_0$ is the initial window (typically 10 segments = 14,600 bytes). Total bytes sent during slow start after $k$ RTTs:

$$B_k = W_0 \cdot \sum_{i=0}^{k} 2^i = W_0 \cdot (2^{k+1} - 1)$$

The number of RTTs to exit slow start and reach window $W_{ss}$ (slow start threshold):

$$k_{ss} = \lceil \log_2(W_{ss} / W_0) \rceil$$

Effective throughput for a file of size $S$:

$$\text{Throughput}_{\text{eff}} = \frac{S}{k_{ss} \cdot \text{RTT} + \frac{S - B_{k_{ss}}}{\min(W_{ss}/\text{RTT}, B)}}$$

### Worked Examples

**Example:** Transfer a 1 MB file over a link with 50 ms RTT, $W_0 = 14600$ bytes, $W_{ss} = 256$ KB, bandwidth $B = 100$ Mbps:

$$k_{ss} = \lceil \log_2(262144 / 14600) \rceil = \lceil 4.17 \rceil = 5 \text{ RTTs}$$

$$B_5 = 14600 \times (2^6 - 1) = 14600 \times 63 = 919800 \text{ bytes} \approx 898 \text{ KB}$$

Remaining: $1048576 - 919800 = 128776$ bytes at full window rate:

$$\text{Time} = 5 \times 0.05 + \frac{128776}{262144/0.05} = 0.25 + 0.0246 = 0.2746 \text{ sec}$$

$$\text{Throughput}_{\text{eff}} = \frac{1048576 \times 8}{0.2746} = 30.5 \text{ Mbps}$$

Only 30.5 Mbps effective on a 100 Mbps link due to slow start — this is why `lftp` with parallel segments (`pget -n 5`) dramatically improves FTP speed.

---

## 3. Connection Tracking State Exhaustion (Counting)

### The Problem

Linux's `nf_conntrack` module tracks FTP control and data connections. Each FTP session creates at least 2 entries (control + data). With conntrack table size $C$, at what point does the system start dropping connections?

### The Formula

Each FTP session uses $2 + d$ conntrack entries where $d$ is the number of concurrent data transfers (for `mget`/`mput`). Total entries for $N$ clients:

$$E_{\text{total}} = N \cdot (2 + \bar{d})$$

Where $\bar{d}$ is the average concurrent data connections per client.

Probability of table exhaustion when entries follow a Poisson distribution with mean $\lambda = E_{\text{total}}$:

$$P(\text{overflow}) = P(X > C) = 1 - \sum_{k=0}^{C} \frac{\lambda^k e^{-\lambda}}{k!}$$

For large $\lambda$ and $C$, use the normal approximation:

$$P(\text{overflow}) \approx 1 - \Phi\left(\frac{C - \lambda}{\sqrt{\lambda}}\right)$$

### Worked Examples

**Example:** Default conntrack table $C = 65536$, each client has 1 data stream on average ($\bar{d} = 1$), so 3 entries per client:

Max safe clients: $N_{\max} = \lfloor 65536 / 3 \rfloor = 21845$

With $N = 20000$ clients ($\lambda = 60000$):

$$P(\text{overflow}) \approx 1 - \Phi\left(\frac{65536 - 60000}{\sqrt{60000}}\right) = 1 - \Phi(22.6) \approx 0$$

Safe. But at $N = 22000$ ($\lambda = 66000$):

$$P(\text{overflow}) \approx 1 - \Phi\left(\frac{65536 - 66000}{\sqrt{66000}}\right) = 1 - \Phi(-1.81) = \Phi(1.81) = 0.965$$

96.5% chance of overflow. Monitor with `cat /proc/sys/net/netfilter/nf_conntrack_count`.

---

## 4. Transfer Integrity and Bit Error Rates (Information Theory)

### The Problem

FTP has no built-in integrity verification (unlike SFTP which uses HMAC). Over a link with bit error rate $p_e$, what is the probability of undetected corruption in a file of $S$ bytes?

### The Formula

TCP uses a 16-bit checksum. The probability that a corrupted segment passes the checksum:

$$P(\text{undetected per segment}) \approx 2^{-16} = 1.53 \times 10^{-5}$$

For a file transferred in $n = \lceil S / \text{MSS} \rceil$ segments, each with error probability $p_s$:

$$p_s = 1 - (1 - p_e)^{8 \cdot \text{MSS}}$$

Probability of at least one undetected corruption:

$$P(\text{corrupt file}) = 1 - \left(1 - p_s \cdot 2^{-16}\right)^n$$

### Worked Examples

**Example:** 1 GB file, MSS = 1460 bytes, $p_e = 10^{-9}$ (typical fiber):

$$n = \frac{10^9}{1460} = 684932 \text{ segments}$$

$$p_s = 1 - (1 - 10^{-9})^{11680} \approx 1.168 \times 10^{-5}$$

$$P(\text{corrupt file}) = 1 - (1 - 1.168 \times 10^{-5} \times 1.53 \times 10^{-5})^{684932}$$

$$= 1 - (1 - 1.787 \times 10^{-10})^{684932} \approx 1 - e^{-1.224 \times 10^{-4}} \approx 1.224 \times 10^{-4}$$

About 1 in 8,000 transfers of 1 GB over fiber could have undetected corruption. This is why post-transfer checksum verification (MD5/SHA256) is essential for FTP.

---

## 5. Anonymous FTP Abuse — Connection Rate Modeling (Poisson Process)

### The Problem

An anonymous FTP server receives connection attempts from both legitimate users and scanners/bots. If legitimate connections arrive as a Poisson process with rate $\lambda_l$ and malicious connections at rate $\lambda_m$, what is the optimal rate limit?

### The Formula

Total arrival rate $\lambda = \lambda_l + \lambda_m$. With a rate limit of $r$ connections per second, the fraction of legitimate connections accepted:

$$f_l = \min\left(1, \frac{r}{\lambda}\right) \cdot \frac{\lambda_l}{\lambda}$$

Maximizing the ratio of legitimate to total accepted connections:

$$\text{maximize } \frac{f_l}{f_l + f_m} = \frac{\lambda_l}{\lambda_l + \lambda_m}$$

This ratio is independent of $r$ — rate limiting reduces volume but not the spam ratio. Connection-level rate limiting per IP is more effective:

$$r_{\text{per-ip}} = \frac{\lambda_l / N_l}{\alpha}$$

Where $N_l$ is the expected number of legitimate client IPs and $\alpha < 1$ is a safety factor.

### Worked Examples

**Example:** $\lambda_l = 5$/sec from 50 IPs, $\lambda_m = 200$/sec from 10000 IPs. Per-IP rates:

- Legitimate: $5/50 = 0.1$ conn/sec/IP
- Malicious: $200/10000 = 0.02$ conn/sec/IP

Setting per-IP limit at $r = 0.05$ conn/sec (1 per 20 sec):

- Legitimate IPs: 0.05/0.1 = 50% of connections accepted per IP, total $2.5$/sec
- Malicious IPs: 0.02 < 0.05, so all pass — but total is only $200$/sec

Better: `vsftpd` option `max_per_ip=3` limits concurrent connections, which is more effective than rate-based limiting.

---

## Prerequisites

- TCP/IP fundamentals (slow start, congestion window, MSS, RTT)
- Combinatorics (birthday problem, permutations)
- Probability (Poisson processes, normal approximation)
- Information theory (error detection, checksums)
- Network address translation (NAT traversal)
