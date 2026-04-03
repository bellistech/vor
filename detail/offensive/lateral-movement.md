# The Mathematics of Lateral Movement — Network Traversal and Credential Propagation

> *Lateral movement is graph traversal through a network's authentication topology. Each compromised host reveals credentials that unlock adjacent hosts, creating an expanding frontier of access. The mathematics involve graph theory, credential reuse probability, and the exponential growth of the attack surface.*

---

## 1. Network as an Authentication Graph

### Graph Model

$$G_{auth} = (H, E) \quad \text{where } (h_i, h_j) \in E \iff \text{credentials from } h_i \text{ grant access to } h_j$$

### Graph Properties

| Property | Flat Network | Segmented | Zero Trust |
|:---|:---:|:---:|:---:|
| Average degree | 20-50 | 5-15 | 1-3 |
| Diameter | 2-3 | 4-6 | 10+ |
| Clustering coefficient | High (0.6-0.8) | Medium (0.3-0.5) | Low (<0.1) |
| Time to reach domain controller | Minutes | Hours | Days (if possible) |

### Credential Reuse Creates Dense Graphs

If $p$ is the probability that credentials from host $A$ work on host $B$:

$$E[|\text{neighbors}(h)|] = (N - 1) \times p$$

| Environment | $p$ (credential reuse) | Avg Neighbors (100 hosts) |
|:---|:---:|:---:|
| Poorly managed | 0.30 | 30 |
| Average enterprise | 0.15 | 15 |
| Well-segmented | 0.05 | 5 |
| Zero trust | 0.01 | 1 |

---

## 2. Attack Frontier Expansion

### BFS Model of Lateral Movement

Starting from initial compromise $h_0$, the attacker's frontier expands like BFS:

$$F(t+1) = F(t) \cup \{h : (h', h) \in E, h' \in F(t), h \notin F(t)\}$$

### Growth Rate

If each compromised host yields access to $d$ new hosts (average degree):

$$|F(t)| \approx \min(d^t, N)$$

| Time Steps | $d = 3$ | $d = 5$ | $d = 10$ |
|:---:|:---:|:---:|:---:|
| 1 | 3 | 5 | 10 |
| 2 | 9 | 25 | 100 |
| 3 | 27 | 125 | 1,000 |
| 4 | 81 | 625 | 10,000 |
| 5 | 243 | 3,125 | 100,000 |

In a flat network ($d = 10$), an attacker can reach 10,000 hosts in 4 hops — typically under 1 hour.

### Time per Hop

$$T_{hop} = T_{harvest} + T_{attempt} + T_{session}$$

| Component | Duration | Notes |
|:---|:---:|:---|
| Credential harvesting | 1-5 min | Mimikatz, token theft |
| Credential testing | 0.5-2 min | SMB/WinRM/SSH auth |
| Session establishment | 0.5-1 min | Shell/agent setup |
| **Total per hop** | **2-8 min** | |

### Total Compromise Time

$$T_{total} = \delta(h_0, h_{target}) \times T_{hop}$$

Where $\delta$ is the shortest path length in the credential graph.

For a domain controller at distance 3, $T_{hop} = 5$ min: $T = 15$ minutes.

---

## 3. Credential Types and Harvesting

### Windows Credential Hierarchy

| Credential | Where Found | Reuse Scope | Persistence |
|:---|:---|:---|:---:|
| NTLM hash | LSASS memory, SAM | Same password everywhere | Until password change |
| Kerberos TGT | LSASS memory | Domain-wide | 10 hours (default) |
| Kerberos service ticket | LSASS memory | Specific service | 10 hours |
| Cached domain creds | Registry (MSCACHE) | Offline auth | Until evicted |
| Cleartext password | LSASS (WDigest), logs | Universal | Until password change |
| DPAPI master key | User profile | Decrypt user secrets | Permanent |

### Credential Yield per Host

$$E[\text{credentials per host}] = n_{interactive} \times P(\text{cached})$$

| Host Type | Interactive Logons | Expected Credentials |
|:---|:---:|:---:|
| Workstation | 1-2 | 1-3 |
| Jump server | 5-20 | 5-20 |
| IT admin workstation | 3-10 | 5-15 (often privileged) |
| Domain controller | 10-50 | 10-50 (all admin) |
| File server | 2-5 | 2-5 |

### The Admin Hop Problem

IT administrators log into many systems, depositing credentials:

$$\text{Admin credential exposure} = |\text{systems admin logged into}|$$

A domain admin who logs into 20 systems creates 20 potential harvest points. Compromising ANY one yields domain admin credentials.

---

## 4. Pass-the-Hash Mathematics

### NTLM Authentication (No Password Needed)

$$\text{NTLM Response} = \text{HMAC-MD5}(\text{NT hash}, \text{challenge})$$

The NT hash is sufficient — no need for the plaintext password.

### Hash Equivalence

$$\text{NT hash}(P) = \text{MD4}(\text{UTF-16LE}(P))$$

Anyone possessing the hash can authenticate without knowing the password.

### Effective Password Reuse with PTH

If user $U$ uses the same password on $n$ systems:

$$\text{Systems accessible via PTH} = n$$

Enterprise average: a single harvested hash grants access to 3-5 additional systems.

---

## 5. Kerberos Attacks

### Kerberoasting — Service Ticket Cracking

Any domain user can request service tickets (TGS) for SPN-registered accounts:

$$\text{TGS} = \text{Encrypt}_{K_{service}}(\text{session key}, \text{metadata})$$

The TGS is encrypted with the service account's password hash — crackable offline.

### Offline Cracking Rate

| Hash Type | GPU Rate (RTX 4090) | 8-char Full ASCII Time |
|:---|:---:|:---:|
| Kerberos 5 TGS-REP (RC4) | $3 \times 10^9$/s | 37 minutes |
| Kerberos 5 TGS-REP (AES) | $2 \times 10^6$/s | 38 days |

RC4 (etype 23) is vastly easier to crack than AES (etype 17/18).

### Golden Ticket

With the KRBTGT hash, forge any TGT:

$$\text{Golden TGT} = \text{Encrypt}_{K_{krbtgt}}(\text{arbitrary PAC, arbitrary user, arbitrary groups})$$

Validity: 10 years (default TGT lifetime configurable by attacker).

### Silver Ticket

With a service account hash, forge TGS for that service:

$$\text{Silver TGS} = \text{Encrypt}_{K_{service}}(\text{arbitrary PAC})$$

Harder to detect than Golden Tickets (no TGT request to DC).

---

## 6. Detection Probability

### Per-Hop Detection

$$P(\text{detect hop}) = 1 - \prod_{i=1}^{n} (1 - P_i(\text{detect}))$$

| Detection Source | $P(\text{detect per hop})$ |
|:---|:---:|
| Windows Event Log (4624, 4625) | 0.20 |
| Network IDS (anomalous SMB) | 0.15 |
| EDR (credential dump detection) | 0.40 |
| Honey tokens/accounts | 0.10 |
| Behavioral analytics (UEBA) | 0.25 |

$$P(\text{detect per hop}) = 1 - (0.8)(0.85)(0.60)(0.90)(0.75) = 1 - 0.275 = 0.725$$

### Detection Over Multiple Hops

$$P(\text{detect across } k \text{ hops}) = 1 - (1 - P_{hop})^k$$

| Hops | $P(\text{detect})$ at 72.5%/hop |
|:---:|:---:|
| 1 | 72.5% |
| 2 | 92.4% |
| 3 | 97.9% |
| 4 | 99.4% |
| 5 | 99.8% |

More hops = higher cumulative detection probability. Defenders should maximize per-hop detection to make multi-hop movement nearly impossible to complete undetected.

---

## 7. Defense: Tiered Administration

### Tier Model

| Tier | Contains | Can Authenticate To |
|:---:|:---|:---|
| 0 | Domain controllers, PKI | Tier 0 only |
| 1 | Servers, applications | Tier 1 and below |
| 2 | Workstations, users | Tier 2 only |

### Cross-Tier Credential Exposure

Without tiering:

$$\text{Admin credential exposure} = N_{total}$$

With tiering:

$$\text{Tier 0 exposure} = N_{tier0} \ll N_{total}$$

| Tier Model | Domain Admin Exposure | Compromise Impact |
|:---|:---:|:---|
| No tiering | All hosts | One workstation = domain admin |
| 3-tier | DC only | Must compromise DC directly |
| PAW (Privileged Access Workstation) | Dedicated device | Physical access or PAW exploit |

### Reduction in Attack Paths

$$\frac{\text{Paths (no tiering)}}{\text{Paths (tiered)}} = \frac{N^2}{N_0^2 + N_1^2 + N_2^2}$$

For 1000 hosts (10 T0, 100 T1, 890 T2):

$$\text{Reduction} = \frac{10^6}{100 + 10{,}000 + 792{,}100} = \frac{10^6}{802{,}200} = 1.25\times$$

The path reduction alone is modest, but the credential isolation prevents T2 → T0 escalation entirely.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Auth graph $G = (H, E)$ | Graph theory | Network model |
| $d^t$ frontier growth | Exponential | Attack spread rate |
| $\delta(h_0, h_{target})$ | Shortest path | Movement planning |
| NTLM $=$ MD4(password) | Hash function | Pass-the-hash |
| $3 \times 10^9$/s (RC4) | Cracking rate | Kerberoasting |
| $1 - (1-P)^k$ | Cumulative detection | Multi-hop alerting |
| Tier isolation | Graph partitioning | Credential protection |

---

*Lateral movement is the core of every breach — initial compromise grants a foothold, but lateral movement turns a single compromised workstation into domain-wide access. The mathematics show that dense credential graphs make this trivial, while tiered administration and per-hop detection make it detectable and containable.*
