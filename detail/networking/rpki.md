# The Mathematics of RPKI — Certificate Chains, ROA Validation Logic, and Deployment Game Theory

> *RPKI imposes a cryptographic trust hierarchy on the BGP routing system, transforming origin validation from a social trust problem into a mathematical verification. The analysis covers X.509 certificate chain validation, combinatorial prefix-to-ROA matching, and game-theoretic incentives driving global deployment.*

---

## 1. Certificate Chain Validation (Graph Theory / PKI Mathematics)

### The Trust DAG

RPKI forms a directed acyclic graph (DAG) rooted at five Trust Anchors (RIR root certificates):

$$\text{TA} \rightarrow \text{CA}_{LIR} \rightarrow \text{CA}_{ISP} \rightarrow \text{EE}_{ROA}$$

Each edge represents a certificate issuance. Validation requires traversing from the End Entity (EE) certificate back to a trusted root:

$$\text{Valid}(EE) = \exists \text{ path } (TA \rightarrow CA_1 \rightarrow \ldots \rightarrow CA_n \rightarrow EE)$$

Where every certificate in the chain:
1. Has a valid signature from its parent
2. Has not expired: $t_{now} \in [t_{notBefore}, t_{notAfter}]$
3. Contains IP/AS resources that are a subset of the parent's resources

### Resource Inheritance

If parent certificate $C_p$ holds resources $R_p$ and child $C_c$ holds $R_c$:

$$R_c \subseteq R_p$$

This is the **overclaiming** check. A child cannot certify resources the parent does not hold:

$$\text{Valid}(C_c) \Rightarrow \forall r \in R_c : r \in R_p$$

### Chain Depth and Validation Cost

With chain depth $d$ and $n$ certificates per level:

$$\text{Signature verifications} = d$$

$$\text{Resource containment checks} = d - 1$$

Typical RPKI chain depth: 3-4 (TA → RIR CA → ISP CA → EE). Validation is $O(d)$ per ROA, and $d$ is bounded by policy (rarely > 5).

---

## 2. ROA Validation — Prefix Matching Logic (Set Theory)

### The Validation Function

For a BGP route announcement $(p, l, as)$ where $p$ = prefix, $l$ = prefix length, $as$ = origin AS:

$$V(p, l, as) = \begin{cases}
\text{VALID} & \text{if } \exists \text{ ROA}: as = as_{ROA} \land p \supseteq p_{ROA} \land l \leq maxLen_{ROA} \\
\text{INVALID} & \text{if } \exists \text{ ROA covering } p \text{ but no matching ROA} \\
\text{NOT FOUND} & \text{if no ROA covers } p
\end{cases}$$

### Formal Definitions

A ROA $R = (as_R, p_R, l_R, maxLen_R)$ **covers** an announced prefix $(p, l)$ if:

$$p_R \text{ is a prefix of } p \quad \land \quad l_R \leq l$$

A ROA $R$ **matches** an announcement $(p, l, as)$ if:

$$R \text{ covers } (p, l) \quad \land \quad as = as_R \quad \land \quad l \leq maxLen_R$$

### Truth Table for a /24 Announcement

Given ROA: AS 64500, 203.0.113.0/24, maxLength /24:

| Announcement | Origin AS | Length | Covers? | Matches? | Result |
|:---|:---:|:---:|:---:|:---:|:---:|
| 203.0.113.0/24 | 64500 | 24 | Yes | Yes | VALID |
| 203.0.113.0/24 | 64501 | 24 | Yes | No (wrong AS) | INVALID |
| 203.0.113.0/25 | 64500 | 25 | Yes | No (25 > 24) | INVALID |
| 203.0.113.128/25 | 64500 | 25 | Yes | No (25 > 24) | INVALID |
| 198.51.100.0/24 | 64500 | 24 | No | N/A | NOT FOUND |

### maxLength Risk Analysis

A ROA with loose maxLength creates a larger "valid" space:

$$\text{Valid announcements} = \sum_{l=l_R}^{maxLen_R} 2^{l - l_R}$$

For prefix /24, maxLength /28:

$$\text{Valid announcements} = 2^0 + 2^1 + 2^2 + 2^3 + 2^4 = 1 + 2 + 4 + 8 + 16 = 31$$

For prefix /24, maxLength /24:

$$\text{Valid announcements} = 2^0 = 1$$

Loose maxLength increases the attack surface by $31\times$ in this example. RFC 9319 recommends setting maxLength equal to announced prefix length.

---

## 3. Deployment Game Theory — Incentive Analysis

### The Network Effect Model

RPKI ROV only protects against hijacks if the hijacked prefix has a ROA AND the receiving AS performs ROV. Model this as a two-player game:

Let $f_{ROA}$ = fraction of prefixes with ROAs, $f_{ROV}$ = fraction of ASes performing ROV.

$$P(\text{hijack blocked}) = f_{ROA} \times f_{ROV}$$

### Deployment Incentive Matrix

|  | ROA Created | No ROA |
|:---|:---:|:---:|
| **ROV Enabled** | Protected (mutual benefit) | No protection (origin) |
| **No ROV** | No protection (validator) | Status quo |

### Nash Equilibrium Analysis

The "free-rider" problem: ROV costs resources but only protects others' ROA-covered prefixes. Creating a ROA is cheap but only useful if others do ROV.

$$\text{Payoff}_{ROA}(f_{ROV}) = f_{ROV} \times V_{protection} - C_{ROA}$$

$$\text{Payoff}_{ROV}(f_{ROA}) = f_{ROA} \times V_{filtering} - C_{ROV}$$

The tipping point occurs when:

$$f_{ROA} > \frac{C_{ROV}}{V_{filtering}} \quad \text{and} \quad f_{ROV} > \frac{C_{ROA}}{V_{protection}}$$

With $C_{ROA} \approx 0$ (free via RIR portals) and $C_{ROV}$ modest (validator + config), the equilibrium favors universal adoption once $f_{ROA}$ exceeds $\sim 30\%$ (crossed in 2022).

### Current State (2024)

$$f_{ROA} \approx 0.52, \quad f_{ROV} \approx 0.40$$

$$P(\text{hijack blocked}) \approx 0.52 \times 0.40 = 0.208$$

Approximately 21% of potential hijacks are now blocked by RPKI. As both factors increase, protection improves quadratically in the product.

---

## 4. VRP Count and Memory — Scaling Analysis

### Current Scale

The global RPKI system produces approximately 400,000+ Validated ROA Payloads (VRPs) as of 2024.

### Memory Requirements

Each VRP stored on a router:

| Field | Size |
|:---|:---:|
| Prefix (IPv4 or IPv6) | 16 bytes |
| Max length | 1 byte |
| Origin AS | 4 bytes |
| Flags / metadata | 3 bytes |
| **Total** | **~24 bytes** |

$$M = N_{VRP} \times 24 \text{ bytes}$$

| VRP Count | Memory |
|:---:|:---:|
| 400,000 | 9.6 MB |
| 1,000,000 | 24 MB |
| 5,000,000 | 120 MB |

### Lookup Complexity

VRP lookup during BGP origin validation uses a prefix trie:

$$\text{Lookup time} = O(W)$$

Where $W$ = address width (32 for IPv4, 128 for IPv6). Independent of VRP count.

### RTR Update Overhead

Incremental RTR updates transmit only deltas. For $\Delta$ changes per refresh cycle:

$$\text{Bandwidth} = \Delta \times 24 \text{ bytes per cycle}$$

Typical $\Delta \approx 100\text{-}500$ per hour, so bandwidth is negligible (~12 KB/cycle).

---

## 5. Cryptographic Overhead — Signature Verification (Computational Complexity)

### Per-ROA Validation Cost

Each ROA requires:
1. EE certificate signature verification: 1 RSA or ECDSA verify
2. CA chain verification: $d-1$ signature verifies
3. CRL/manifest checks: 1-2 additional verifies per CA level

$$\text{Total verifications per ROA} = 2d - 1$$

### Bulk Validation Timing

With RSA-2048 signatures (most common in RPKI):

$$T_{verify} \approx 0.1 \text{ ms per signature (modern hardware)}$$

For 400,000 ROAs with average chain depth 3:

$$T_{total} = 400{,}000 \times (2 \times 3 - 1) \times 0.1\text{ ms} = 200{,}000\text{ ms} = 200\text{ s}$$

In practice, CA certificates are cached and verified once, reducing this to:

$$T_{effective} \approx 400{,}000 \times 0.1\text{ ms} = 40\text{ s}$$

Validators typically complete a full cycle in 2-5 minutes including download time.

---

## 6. ROA Coverage — Combinatorial Analysis

### The Coverage Gap

A prefix with no ROA is in NOT FOUND state. The fraction of BGP table covered by ROAs:

$$\text{Coverage} = \frac{|\{r \in BGP : \exists \text{ ROA covering } r\}|}{|BGP|}$$

### Partial Deployment Risk

If an organization creates ROAs for only some of its prefixes:

$$\text{Unprotected fraction} = 1 - \frac{n_{ROA}}{n_{total}}$$

More dangerously, inconsistent ROA creation can cause operational issues:

$$P(\text{self-inflicted INVALID}) = \frac{n_{announced\_without\_ROA}}{n_{total}} \times f_{ROV}$$

If you announce 10 prefixes but only create ROAs for 8, and a covering ROA from a parent allocation exists, the 2 uncovered prefixes may become INVALID (not NOT FOUND).

### Covering ROA Problem

A ROA for /22 with maxLength /22 makes any announcement of /23 or /24 from that block INVALID, even if announced by the correct AS:

$$\text{ROA: AS64500, 203.0.112.0/22, maxLen=/22}$$
$$\text{Announce: AS64500, 203.0.113.0/24} \rightarrow \textbf{INVALID} \text{ (24 > 22)}$$

This is the most common operational RPKI mistake. Always create ROAs matching your exact announcements.

---

## 7. Summary of Formulas

| Formula | Domain | Application |
|:---|:---|:---|
| $R_c \subseteq R_p$ | Set theory | Certificate resource inheritance |
| $\sum_{l=l_R}^{maxLen} 2^{l-l_R}$ | Combinatorics | Valid announcement space |
| $f_{ROA} \times f_{ROV}$ | Probability | Hijack protection rate |
| $N \times 24$ bytes | Linear | VRP memory on router |
| $O(W)$ trie lookup | Algorithmic | Origin validation speed |
| $2d - 1$ verifications | Linear | Crypto cost per ROA |

---

*RPKI transforms BGP origin validation from "trust everyone" to "verify cryptographically." The mathematics show that even partial deployment provides meaningful protection, and the game-theoretic incentives now favor adoption: with ROA creation free and ROV costs modest, the Nash equilibrium has shifted from "nobody deploys" to "everybody deploys." The remaining challenge is closing the NOT FOUND gap — the 48% of routes without ROAs that remain vulnerable.*

## Prerequisites

- public key cryptography and X.509 certificates, set theory and prefix matching, basic game theory

## Complexity

- **Beginner:** Understand ROAs, the three validation states (Valid/Invalid/Not Found), and why RPKI matters for BGP security.
- **Intermediate:** Analyze maxLength risk, configure validators and RTR protocol, and deploy ROV policies on routers.
- **Advanced:** Model deployment incentives, evaluate covering ROA edge cases, and design RPKI monitoring for ISP-scale networks.
