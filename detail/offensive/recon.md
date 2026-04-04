# The Mathematics of Reconnaissance — Information Gathering and Attack Surface Mapping

> *Reconnaissance is the systematic enumeration of a target's digital footprint. The mathematics involve port scanning combinatorics, DNS enumeration probability, OSINT correlation, and the information-theoretic value of each discovered asset in reducing attack uncertainty.*

---

## 1. Port Scanning — Combinatorial Enumeration

### TCP Port Space

$$|\text{TCP ports}| = 65{,}535, \quad |\text{UDP ports}| = 65{,}535$$

### Scan Time Formula

$$T_{scan} = \frac{N_{hosts} \times N_{ports} \times T_{probe}}{P_{parallel}}$$

Where $T_{probe}$ is the time per probe and $P_{parallel}$ is the parallelism level.

### SYN Scan Throughput

| Tool | Max Rate | 65535 Ports on 1 Host | 1000 Hosts, Top 1000 Ports |
|:---|:---:|:---:|:---:|
| nmap (default) | 1,000 pps | 65 seconds | 1,000 seconds |
| nmap (aggressive) | 10,000 pps | 6.5 seconds | 100 seconds |
| masscan | 1,000,000 pps | 0.07 seconds | 1 second |
| zmap | 1,400,000 pps | 0.05 seconds | 0.7 seconds |

### Internet-Scale Scanning

Full IPv4 scan on a single port:

$$T = \frac{2^{32}}{R_{pps}} = \frac{4.3 \times 10^9}{10^6} = 4{,}295 \text{ seconds} = 72 \text{ minutes}$$

At 1M packets/second, the entire IPv4 internet can be scanned in ~1 hour.

### Service Detection

After port discovery, service fingerprinting:

$$T_{version} = N_{open\_ports} \times T_{fingerprint}$$

Where $T_{fingerprint} \approx 1-5$ seconds per port (probe + response analysis).

---

## 2. DNS Enumeration

### Subdomain Discovery Methods

| Method | Speed | Completeness | Stealth |
|:---|:---:|:---:|:---:|
| Zone transfer (AXFR) | Instant | 100% (if allowed) | Logged |
| Brute force | $|D| / R$ seconds | $|D| / |S_{total}|$ | Detectable |
| Certificate Transparency | Instant | 70-90% | Passive |
| Search engine dorking | Minutes | 20-40% | Passive |
| DNS reverse lookup | Slow | Variable | Active |

### Brute Force Mathematics

$$T_{brute} = \frac{|D|}{R_{queries/sec}}$$

Where $|D|$ is the dictionary size and $R$ is the query rate.

| Dictionary | Size | At 1000 qps | At 10000 qps |
|:---|:---:|:---:|:---:|
| Common subs (basic) | 5,000 | 5 sec | 0.5 sec |
| SecLists DNS | 110,000 | 110 sec | 11 sec |
| All 3-char combos | $36^3 = 46{,}656$ | 47 sec | 4.7 sec |
| All 4-char combos | $36^4 = 1{,}679{,}616$ | 28 min | 2.8 min |
| All 5-char combos | $36^5 = 60{,}466{,}176$ | 16.8 hours | 1.7 hours |

### Certificate Transparency Completeness

CT logs contain all certificates issued by participating CAs:

$$\text{Coverage} = \frac{|\text{CT-logged certs}|}{|\text{all certs}|} \approx 99\%$$

For a domain with $n$ subdomains, CT typically reveals:

$$E[\text{discovered}] = n \times 0.85 \quad \text{(85% coverage)}$$

CT is the most efficient passive subdomain enumeration — no interaction with the target.

---

## 3. OSINT — Open Source Intelligence

### Information Value

$$V(\text{info}) = P(\text{useful for attack}) \times I(\text{impact if exploited})$$

### OSINT Sources and Yield

| Source | Data Obtained | Typical Yield |
|:---|:---|:---:|
| LinkedIn | Employee names, roles, tech stack | 50-200 contacts |
| GitHub (org repos) | Code, secrets, internal URLs | 5-50 repos |
| Shodan/Censys | Open ports, services, banners | 10-100 hosts |
| Google dorking | Exposed files, admin panels | 5-50 findings |
| WHOIS/DNS | Registrant, name servers, IP ranges | 2-20 records |
| Job postings | Technology stack, architecture | 3-10 tech indicators |
| Pastebin/breach data | Leaked credentials | 0-1000 creds |

### Technology Stack Inference

Each piece of information reduces uncertainty about the target:

$$H(\text{target}) = H_0 - \sum_{i} I(x_i)$$

Where $I(x_i)$ is the information gained from observation $x_i$.

Example: "We use Kubernetes" → eliminates non-container architectures, narrows attack vectors.

---

## 4. Network Mapping — Topology Discovery

### Traceroute Mathematics

Each hop reveals one router:

$$\text{Hops} = \text{TTL}_{initial} - \text{TTL}_{remaining}$$

Typical internet path: 8-15 hops. Internal network: 2-5 hops.

### Network Range Estimation

Given discovered IPs $\{ip_1, ip_2, \ldots, ip_n\}$:

$$\text{Estimated range} = [\min(ip_i), \max(ip_i)] \text{ within the CIDR block}$$

WHOIS/BGP data confirms the actual allocation:

$$\text{Scan scope} = \text{WHOIS block size} = 2^{32 - \text{prefix}}$$

| CIDR | Hosts | Scan Time (1000 pps, top 100 ports) |
|:---:|:---:|:---:|
| /28 | 16 | 1.6 seconds |
| /24 | 256 | 26 seconds |
| /20 | 4,096 | 6.8 minutes |
| /16 | 65,536 | 1.8 hours |

---

## 5. Web Application Enumeration

### Directory/File Brute Force

$$T_{brute} = \frac{|W| \times |E|}{R_{req/sec}}$$

Where $|W|$ is wordlist size and $|E|$ is extension count.

| Wordlist | Words | Extensions | Candidates | Time (100 req/s) |
|:---|:---:|:---:|:---:|:---:|
| common.txt | 4,600 | 1 | 4,600 | 46 sec |
| directory-list-2.3-medium | 220,000 | 1 | 220,000 | 37 min |
| directory-list-2.3-medium | 220,000 | 5 (.php, .asp, .js, .html, .txt) | 1,100,000 | 3 hours |
| big.txt | 20,000 | 10 | 200,000 | 33 min |

### Status Code Analysis

| Code | Meaning | Action |
|:---:|:---|:---|
| 200 | Found | Analyze content |
| 301/302 | Redirect | Follow (may reveal internal paths) |
| 403 | Forbidden | Path exists (access control) |
| 404 | Not found | Skip |
| 500 | Server error | May indicate injection point |

### Crawl Coverage

$$\text{Coverage} = \frac{|\text{discovered pages}|}{|\text{total pages}|}$$

Spider/crawler limitations:
- JavaScript-rendered content: ~50-70% coverage without headless browser
- Authentication-required pages: 0% without credentials
- Sitemap + robots.txt: reveals paths the crawler would miss

---

## 6. Email Enumeration

### Email Pattern Discovery

Given confirmed emails, infer the pattern:

$$\text{Pattern} = f(\text{first name}, \text{last name})$$

| Pattern | Example | Prevalence |
|:---|:---|:---:|
| first.last | john.smith@corp.com | 40% |
| firstlast | johnsmith@corp.com | 15% |
| first_last | john_smith@corp.com | 10% |
| f.last | j.smith@corp.com | 15% |
| first.l | john.s@corp.com | 10% |
| first | john@corp.com | 10% |

### Email Verification

| Method | Reliability | Stealth |
|:---|:---:|:---:|
| SMTP VRFY | 60% (often disabled) | Logged |
| SMTP RCPT TO | 80% (accept/reject) | Logged |
| Office 365 enumeration | 95% (timing/response diff) | Semi-stealthy |
| Google Workspace enum | 90% (avatar API) | Passive |

### Harvest Size Estimation

$$E[\text{valid emails}] = N_{employees} \times P(\text{pattern match}) \times P(\text{data available})$$

For a 500-person company with LinkedIn profiles:

$$E[\text{emails}] = 500 \times 0.8 \times 0.6 = 240 \text{ verified emails}$$

---

## 7. Attack Surface Scoring

### Asset Risk Ranking

$$\text{Risk}(a) = \text{Exposure}(a) \times \text{Criticality}(a) \times \text{Vulnerability}(a)$$

| Factor | Low (0.2) | Medium (0.5) | High (1.0) |
|:---|:---|:---|:---|
| Exposure | Internal only | DMZ | Internet-facing |
| Criticality | Dev/test | Business app | Auth/payment/data |
| Vulnerability | Patched, hardened | Some findings | Unpatched, default creds |

### Prioritized Target List

| Asset | Exposure | Criticality | Vulnerability | Score |
|:---|:---:|:---:|:---:|:---:|
| VPN gateway | 1.0 | 1.0 | 0.5 | 0.50 |
| Web app (public) | 1.0 | 0.5 | 1.0 | 0.50 |
| Mail server | 1.0 | 0.5 | 0.5 | 0.25 |
| Dev server | 0.5 | 0.2 | 1.0 | 0.10 |
| Internal DB | 0.2 | 1.0 | 0.5 | 0.10 |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| $N_{hosts} \times N_{ports} / P$ | Throughput formula | Scan time |
| $36^k$ combos | Exponential | Subdomain brute force |
| CT coverage 85% | Statistical coverage | Passive enumeration |
| $2^{32-\text{prefix}}$ CIDR | Exponential | Network range |
| $|W| \times |E| / R$ | Linear | Directory brute force |
| Risk $= E \times C \times V$ | Product score | Target prioritization |

---

*Reconnaissance determines the outcome of every engagement — thorough enumeration reveals the one vulnerable service among thousands, the one leaked credential among millions of data points, and the one misconfiguration that grants initial access. The mathematics of combinatorial search and information theory guide this process from noise to signal.*

## Prerequisites

- TCP/IP port scanning mechanics (SYN, connect, UDP)
- DNS enumeration and zone transfer concepts
- OSINT correlation and information entropy

## Complexity

- **Beginner:** Nmap basic scans, whois, DNS lookups, Google dorking, Shodan searches
- **Intermediate:** Service fingerprinting, subdomain brute-forcing, certificate transparency, API enumeration, timing-based detection evasion
- **Advanced:** Port scan combinatorics, banner grab entropy analysis, OSINT graph correlation, scan rate optimization vs IDS detection thresholds
