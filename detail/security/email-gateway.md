# The Engineering of Email Security — Threat Landscape, Authentication Cryptography, and Content Analysis

> *Email remains the primary attack vector for phishing, malware delivery, and data exfiltration. A secure email gateway must layer reputation filtering, cryptographic authentication, content analysis, and encryption to defend the most exploited protocol in enterprise networking.*

---

## 1. Email Threat Landscape

### Attack Taxonomy

```
Email Threats
├── Spam (unsolicited bulk email)
│   ├── Commercial spam (advertising)
│   ├── Phishing (credential harvesting)
│   │   ├── Spear phishing (targeted)
│   │   ├── Whaling (executive targeting)
│   │   └── Clone phishing (replicate legitimate email)
│   └── Scam (419, lottery, romance)
├── Malware Delivery
│   ├── Attachment-based (macro, executable, archive)
│   ├── URL-based (drive-by download, exploit kit)
│   └── Fileless (PowerShell in macro, living-off-the-land)
├── Business Email Compromise (BEC)
│   ├── CEO fraud (impersonate executive)
│   ├── Invoice fraud (modify payment details)
│   └── Account takeover (compromised mailbox)
├── Data Exfiltration
│   ├── Intentional (insider threat)
│   └── Accidental (wrong recipient, over-sharing)
└── Infrastructure Abuse
    ├── Open relay exploitation
    ├── Backscatter (NDR to spoofed sender)
    └── Snowshoe spam (distributed low-volume)
```

### Threat Volume and Economics

$$P_{spam} = \frac{V_{spam}}{V_{total}} \approx 0.45 \text{ to } 0.85$$

Global email traffic is approximately 300+ billion messages per day, with spam accounting for 45-85% depending on measurement methodology. The economics favor attackers:

$$ROI_{phishing} = \frac{N_{recipients} \times P_{click} \times P_{credential} \times V_{account}}{C_{infrastructure}}$$

Where:
- $N_{recipients}$ = number of targets (millions at low cost)
- $P_{click}$ = probability of clicking link (~3-5% for untargeted, 30%+ for spear phishing)
- $P_{credential}$ = probability of entering credentials (~50% of clickers)
- $V_{account}$ = value of compromised account ($100-$10,000+)
- $C_{infrastructure}$ = cost of sending campaign ($10-$1,000)

Even with low click rates, the economics are overwhelmingly profitable for attackers.

---

## 2. SMTP Protocol Security

### SMTP Conversation Anatomy

```
Client                                 Server
  |--- TCP connect (port 25) ---------->|
  |<-- 220 mail.example.com ESMTP ------|
  |--- EHLO sender.example.com -------->|
  |<-- 250 (capabilities list) ---------|
  |                                     |
  |--- STARTTLS ----------------------->|  (upgrade to TLS)
  |<-- 220 Ready for TLS --------------|
  |=== TLS Handshake ==================|
  |                                     |
  |--- MAIL FROM:<sender@ext.com> ----->|  (envelope sender)
  |<-- 250 OK -------------------------|
  |--- RCPT TO:<user@example.com> ----->|  (envelope recipient)
  |<-- 250 OK -------------------------|
  |--- DATA --------------------------->|
  |<-- 354 Start mail input ------------|
  |--- From: "Display Name" <...> ---->|  (header From — can differ!)
  |--- To: user@example.com ---------->|  (header To)
  |--- Subject: Hello ----------------->|
  |--- (body) ------------------------->|
  |--- . ------------------------------>|  (end of data)
  |<-- 250 OK, queued ------------------|
  |--- QUIT --------------------------->|
  |<-- 221 Bye -------------------------|
```

### The Envelope vs Header Distinction

A critical security insight: the SMTP envelope (MAIL FROM, RCPT TO) and the message headers (From:, To:) are independent. This is by design (for mailing lists, forwarding, aliases) but creates the spoofing problem:

| Field | Controlled By | Verified By | Spoofable |
|-------|--------------|-------------|-----------|
| Envelope MAIL FROM | Sending MTA | SPF | Yes (without SPF) |
| Header From: | Message composer | DKIM + DMARC | Yes (without DMARC) |
| Header Reply-To: | Message composer | Nothing (display only) | Always |
| Display Name | Message composer | Nothing | Always |

### SMTP Security Limitations

SMTP (RFC 5321) was designed in 1982 with no authentication, encryption, or integrity:

1. **No sender authentication:** Any server can claim to send from any domain
2. **No encryption:** Messages travel in plaintext (STARTTLS is opportunistic)
3. **No integrity:** Messages can be modified in transit
4. **No non-repudiation:** No proof of who sent a message
5. **Store-and-forward:** Messages may traverse multiple MTAs, each a trust boundary

---

## 3. Reputation-Based Filtering

### Reputation Scoring Model

Email reputation systems assign scores to sending IP addresses based on historical behavior:

$$SBRS(IP) = f(V_{spam}, V_{legit}, V_{total}, T_{history}, B_{blocklist}, C_{complaint})$$

Where:
- $V_{spam}$ = volume of spam sent from this IP
- $V_{legit}$ = volume of legitimate mail
- $V_{total}$ = total mail volume
- $T_{history}$ = time period of observation
- $B_{blocklist}$ = presence on public blocklists (Spamhaus, Barracuda, etc.)
- $C_{complaint}$ = feedback loop complaints from recipients

The SenderBase Reputation Score (SBRS) maps to a scale of -10.0 to +10.0:

$$SBRS = 10 \times \left(\frac{V_{legit}}{V_{total}} - \frac{V_{spam}}{V_{total}}\right) \times W_{history} \times W_{blocklist}$$

### Reputation System Architecture

```
Sending IP                ESA                     Talos/SenderBase Cloud
    |                      |                              |
    |--- SMTP connect ---->|                              |
    |                      |--- DNS query: ------------>  |
    |                      |    ip.senderbase.org         |
    |                      |<-- SBRS: +4.2 --------------|
    |                      |                              |
    |                      | [Apply HAT policy based      |
    |                      |  on SBRS range]              |
    |                      |                              |
    |<-- 250 OK / 550 -----|                              |
```

### Connection-Level vs Message-Level Filtering

| Stage | Data Available | Actions | Resource Cost |
|-------|---------------|---------|---------------|
| Connection (HAT) | IP address, PTR record, EHLO | Accept/reject/throttle | Very low |
| Envelope (MAIL FROM/RCPT TO) | Sender domain, recipient | SPF check, recipient verification | Low |
| Header + Body | Full message content | Anti-spam, AV, content filters | High |
| Post-delivery | Retrospective verdicts | Recall, quarantine | Very high |

Blocking at the connection level is far more efficient: rejecting 50% of connections at the HAT eliminates 50% of processing load.

$$CPU_{saved} = P_{blocked\_at\_connection} \times C_{full\_scan}$$

For a gateway processing 1 million messages/day with 60% blocked at connection:
$$CPU_{saved} = 0.6 \times 1000000 \times C_{scan} = 600000 \text{ scans avoided}$$

---

## 4. Content Analysis Techniques

### Multi-Layer Analysis Pipeline

```
Message arrives → Layer 1: Reputation (IP/domain)
                → Layer 2: Header analysis (structure, encoding, routing)
                → Layer 3: Body analysis (text, URLs, patterns)
                → Layer 4: Attachment analysis (type, hash, sandbox)
                → Layer 5: Behavioral analysis (sending patterns)
                → Verdict: Clean / Spam / Malicious / Suspicious
```

### Heuristic Analysis

Heuristic engines apply hundreds of rules, each contributing a weighted score:

$$Score_{spam} = \sum_{i=1}^{N} W_i \times R_i$$

Where $R_i = 1$ if rule $i$ matches, 0 otherwise. The message is classified as spam if $Score_{spam} > Threshold$.

Example rules and weights:

| Rule | Description | Weight |
|------|-------------|--------|
| HTML_IMAGE_ONLY | Message is HTML with only images, no text | +3.2 |
| SUBJ_ALL_CAPS | Subject line entirely uppercase | +1.5 |
| MISSING_DATE | No Date header | +1.8 |
| FORGED_YAHOO_RCVD | Claims Yahoo origin but routing mismatch | +2.5 |
| RDNS_NONE | Sending IP has no reverse DNS | +1.3 |
| BAYES_99 | Bayesian classifier 99% confidence spam | +4.5 |
| URI_MALWARE | URL matches known malware domain | +5.0 |
| DKIM_VALID | Valid DKIM signature | -1.0 |
| SPF_PASS | SPF check passes | -0.5 |

Threshold calibration involves balancing false positives and false negatives:

$$FPR = \frac{FP}{FP + TN}, \quad FNR = \frac{FN}{FN + TP}$$

A lower threshold catches more spam (lower FNR) but increases false positives (higher FPR). Most organizations prefer a small number of spam getting through over blocking legitimate mail.

### Bayesian Classification

Bayesian spam filtering calculates the probability that a message is spam given its word content:

$$P(spam | words) = \frac{P(words | spam) \times P(spam)}{P(words)}$$

Using the naive Bayes independence assumption:

$$P(spam | w_1, w_2, \ldots, w_n) = \frac{P(spam) \prod_{i=1}^{n} P(w_i | spam)}{P(spam) \prod_{i=1}^{n} P(w_i | spam) + P(ham) \prod_{i=1}^{n} P(w_i | ham)}$$

Training requires labeled corpora of spam and legitimate mail. Per-user Bayesian training improves accuracy because each user's mail profile differs.

### Machine Learning in Email Security

Modern email gateways apply ML models beyond naive Bayes:

| Technique | Application | Advantage |
|-----------|------------|-----------|
| Random Forest | URL classification | Handles mixed feature types |
| Neural Network (DNN) | Phishing detection | Learns complex patterns |
| NLP / Transformers | BEC detection | Understands language intent |
| Clustering | Campaign identification | Groups similar messages |
| Anomaly detection | Compromised account detection | Identifies behavioral changes |

BEC detection is particularly challenging because BEC messages contain no malware and no malicious URLs. Detection relies on:

$$P(BEC) = f(Header\_anomaly, Sender\_impersonation, Language\_urgency, Reply\_to\_mismatch)$$

---

## 5. DKIM Cryptographic Signing

### DKIM Signing Process

```
Sender MTA                                    DNS
    |                                          |
    | 1. Canonicalize headers + body           |
    | 2. Hash canonicalized content            |
    | 3. Sign hash with private key            |
    | 4. Add DKIM-Signature header             |
    |                                          |
    |--- Send message with DKIM-Signature ---->| Recipient MTA
                                               |
                                               | 1. Extract d= (domain) and s= (selector)
                                               | 2. Query DNS: s._domainkey.d TXT record
                                               |<--- Public key from DNS ---|
                                               | 3. Canonicalize headers + body
                                               | 4. Hash canonicalized content
                                               | 5. Verify signature with public key
                                               | 6. Pass or fail
```

### DKIM-Signature Header Anatomy

```
DKIM-Signature: v=1;
  a=rsa-sha256;              # signing algorithm
  c=relaxed/relaxed;         # canonicalization (header/body)
  d=example.com;             # signing domain
  s=sel1;                    # selector (DNS lookup key)
  t=1712300000;              # timestamp
  x=1712904800;              # expiration (optional)
  bh=abc123...=;             # body hash (base64)
  h=From:To:Subject:Date:    # signed headers
     MIME-Version;
  b=XYZ789...=               # signature (base64)
```

### Canonicalization

Canonicalization normalizes the message before hashing to tolerate benign modifications in transit:

| Mode | Header Treatment | Body Treatment |
|------|-----------------|----------------|
| simple | No modification (exact match) | Ignore trailing empty lines |
| relaxed | Lowercase header names, unfold, collapse whitespace | Collapse whitespace, ignore trailing empty lines |

$$C_{relaxed}(header) = lowercase(name) + ":" + collapse\_ws(value)$$

Most deployments use `relaxed/relaxed` because intermediate MTAs commonly modify whitespace and header casing.

### Key Size Considerations

$$Security_{DKIM} \propto KeySize_{bits}$$

| Key Size | Security | DNS TXT Size | Recommendation |
|----------|----------|-------------|----------------|
| 1024-bit RSA | Marginal (factorable with resources) | ~180 bytes | Minimum acceptable |
| 2048-bit RSA | Strong | ~400 bytes | Recommended |
| 4096-bit RSA | Very strong | ~800 bytes | May exceed DNS UDP limit (512 bytes) |
| Ed25519 | Strong (small key) | ~44 bytes | Emerging (RFC 8463) |

The 4096-bit key problem: DNS responses exceeding 512 bytes require TCP fallback or EDNS0, which some resolvers handle poorly. 2048-bit RSA is the practical sweet spot.

---

## 6. DMARC Alignment and Policy

### DMARC Evaluation Logic

```
DMARC Pass requires:
  (SPF Pass AND SPF Aligned) OR (DKIM Pass AND DKIM Aligned)

SPF Aligned: domain in MAIL FROM matches domain in header From:
  - strict: exact match
  - relaxed: organizational domain match (sub.example.com aligns with example.com)

DKIM Aligned: domain in DKIM d= matches domain in header From:
  - strict: exact match
  - relaxed: organizational domain match
```

$$DMARC_{result} = (SPF_{pass} \wedge SPF_{aligned}) \vee (DKIM_{pass} \wedge DKIM_{aligned})$$

### DMARC Policy Progression

Organizations should deploy DMARC incrementally:

```
Phase 1: Monitor (p=none)
  _dmarc.example.com IN TXT "v=DMARC1; p=none; rua=mailto:dmarc@example.com"
  → Receive aggregate reports, no enforcement
  → Duration: 2-4 weeks minimum

Phase 2: Quarantine subset (p=quarantine; pct=10)
  _dmarc.example.com IN TXT "v=DMARC1; p=quarantine; pct=10; rua=..."
  → 10% of failing messages quarantined
  → Increase pct gradually: 10 → 25 → 50 → 100

Phase 3: Quarantine all (p=quarantine; pct=100)
  → All failing messages quarantined
  → Monitor for false positives

Phase 4: Reject (p=reject)
  _dmarc.example.com IN TXT "v=DMARC1; p=reject; rua=...; ruf=..."
  → Failing messages rejected at SMTP
  → Maximum protection against domain spoofing
```

### DMARC Aggregate Reports

Aggregate reports (rua=) are XML documents sent by receiving MTAs that show:

$$Report = \{Source\_IP, Count, SPF\_result, DKIM\_result, DMARC\_disposition\}$$

Analyzing these reports reveals:
1. Legitimate sending sources not yet configured with SPF/DKIM
2. Unauthorized use of your domain (spoofing)
3. Misconfigured forwarding that breaks alignment
4. Volume of mail failing DMARC per source

### The Forwarding Problem

Email forwarding breaks SPF because the forwarding server's IP is not in the original domain's SPF record:

```
Original:  sender@example.com → user@company.com  (SPF pass)
Forwarded: sender@example.com → user@company.com → personal@gmail.com
           ↑ forwarding server IP not in example.com SPF → SPF fail
           ↑ but DKIM signature survives forwarding → DKIM pass
           ↑ DMARC: DKIM aligned pass → overall PASS
```

This is why DKIM is essential for DMARC: it survives forwarding while SPF does not. ARC (Authenticated Received Chain, RFC 8617) provides an additional mechanism for preserving authentication through forwarding chains.

---

## 7. SPF Mechanisms and Qualifiers

### SPF Record Syntax

```
v=spf1 [qualifier][mechanism] ... [qualifier]all

Mechanisms (checked left to right, first match wins):
  ip4:203.0.113.0/24       # match sending IP against CIDR
  ip6:2001:db8::/32         # match IPv6
  a                         # match A/AAAA records of domain
  mx                        # match MX records of domain
  include:_spf.google.com   # recursively check another SPF record
  exists:%{i}.spf.example.com  # DNS exists check (advanced)
  ptr                       # reverse DNS match (deprecated, slow)

Qualifiers:
  + (pass, default)     → SPF pass
  - (fail)              → SPF hard fail
  ~ (softfail)          → SPF soft fail (accept but mark)
  ? (neutral)           → no assertion
```

### SPF Lookup Limit

SPF has a hard limit of 10 DNS lookups (RFC 7208, Section 4.6.4):

$$Lookups_{total} = N_{include} + N_{a} + N_{mx} + N_{ptr} + N_{exists} + N_{redirect}$$

Each `include`, `a`, `mx`, `ptr`, `exists`, and `redirect` counts as one lookup. `ip4` and `ip6` do not count (no DNS query needed). Exceeding 10 lookups results in a permanent error (permerror).

Flattening strategies:
1. Replace `include` with resolved IP ranges (requires maintenance)
2. Use subdomains with separate SPF records
3. Use macro-based `exists` mechanisms
4. Consolidate sending infrastructure

### SPF Limitations

| Limitation | Description | Mitigation |
|-----------|-------------|-----------|
| Forwarding breaks SPF | Forwarding server IP not in SPF | Use DKIM + DMARC |
| 10-lookup limit | Complex sending infra exceeds limit | SPF flattening |
| Envelope-only | Checks MAIL FROM, not header From | DMARC adds alignment |
| No encryption | SPF provides authentication only | Use TLS |
| Spoofable without DMARC | SPF alone does not protect header From | Deploy DMARC |

---

## 8. Email Encryption Standards Comparison

### Encryption Models

| Standard | Encryption Scope | Key Management | Recipient Experience |
|----------|-----------------|---------------|---------------------|
| STARTTLS | Transport (MTA to MTA) | Server certificates | Transparent (no action) |
| S/MIME | End-to-end (message level) | X.509 certificates from CA | Must have certificate |
| PGP/GPG | End-to-end (message level) | Web-of-trust, public keys | Must have PGP key |
| Envelope (CRES) | Gateway-to-recipient portal | Gateway manages keys | Open in browser portal |
| MTA-STS | Transport (enforced TLS) | DNS + HTTPS policy | Transparent |

### Transport Encryption (STARTTLS)

```
MTA A                              MTA B
  |--- EHLO ----------------------->|
  |<-- 250-STARTTLS ----------------|
  |--- STARTTLS -------------------->|
  |<-- 220 Go ahead ----------------|
  |=== TLS Handshake ===============|
  |    (negotiate cipher suite)     |
  |    (verify certificate?)        |
  |=== Encrypted SMTP session ======|
```

STARTTLS is opportunistic by default: if the receiving MTA does not advertise STARTTLS or the handshake fails, the message is sent in plaintext. This creates a downgrade attack vector:

$$P_{encrypted} = P_{server\_supports\_TLS} \times P_{handshake\_success} \times (1 - P_{downgrade\_attack})$$

MTA-STS (RFC 8461) and DANE (RFC 7672) address this by allowing domains to publish policies requiring TLS.

### S/MIME vs PGP

| Feature | S/MIME | PGP |
|---------|--------|-----|
| Standard | RFC 8551 | RFC 4880 (OpenPGP) |
| Trust model | Hierarchical CA (X.509) | Web of trust |
| Key distribution | CA-issued certificates | Key servers, manual exchange |
| Email client support | Broad (Outlook, Apple Mail, Thunderbird) | Limited (plugins required) |
| Signing | Yes (non-repudiation) | Yes |
| Encryption | Yes (per-recipient public key) | Yes |
| Gateway integration | ESA supports gateway S/MIME | Less common |
| Enterprise management | CA infrastructure (PKI) | Complex (key management) |

### Encryption Decision Matrix

$$Security_{level} = f(Threat\_model, Compliance, User\_friction, Infrastructure)$$

| Requirement | Recommended | Why |
|-------------|-------------|-----|
| Basic transport security | STARTTLS + MTA-STS | Transparent, no user action |
| Regulatory compliance (HIPAA, PCI) | S/MIME or envelope encryption | Provable end-to-end encryption |
| External partner encryption | Envelope encryption (CRES) | No recipient infrastructure needed |
| Internal sensitive communication | S/MIME with PKI | Integrated with enterprise CA |
| Maximum security (nation-state threat) | PGP + STARTTLS + DANE | Layered, independent trust models |

---

## 9. DLP Pattern Matching

### Detection Techniques

| Technique | How It Works | Accuracy | False Positive Rate |
|-----------|-------------|----------|-------------------|
| Regex | Pattern match (SSN: \d{3}-\d{2}-\d{4}) | Medium | High (matches non-SSN numbers) |
| Smart Identifiers | Regex + checksum validation (Luhn for credit cards) | High | Low |
| Dictionaries | Match against word lists (medical terms, code names) | Medium | Medium |
| Exact data match (EDM) | Hash comparison against actual data set | Very High | Very Low |
| Fingerprinting | Document structure matching | High | Low |
| ML classifiers | Trained on labeled data | High | Medium |

### Smart Identifier Validation

Credit card detection with Luhn algorithm:

$$Luhn(n_1, n_2, \ldots, n_k) = \left(\sum_{i=1}^{k} d_i\right) \mod 10 = 0$$

Where $d_i$ is the digit value after doubling every second digit from the right and subtracting 9 if the doubled value exceeds 9.

Example: 4532015112830366
1. Regex matches: 16 digits in credit card format
2. Luhn checksum validates: legitimate card number format
3. BIN prefix (4532) identifies: Visa
4. Confidence: High (not a random number)

### DLP Severity Scoring

$$Severity = \sum_{i=1}^{N} W_i \times Count_i \times Confidence_i$$

Where:
- $W_i$ = weight of classifier $i$ (SSN=10, credit card=8, keyword=2)
- $Count_i$ = number of matches for classifier $i$
- $Confidence_i$ = confidence level (0-1) based on validation

A message with 5 SSNs (validated) and 2 credit cards (Luhn-verified):
$$Severity = 10 \times 5 \times 1.0 + 8 \times 2 \times 1.0 = 66 \text{ (Critical)}$$

---

## 10. False Positive Management

### The False Positive Cost

$$Cost_{FP} = N_{FP} \times (T_{investigation} \times C_{analyst} + V_{delayed\_business})$$

For an organization processing 100,000 messages/day with 0.01% false positive rate:
$$N_{FP} = 100000 \times 0.0001 = 10 \text{ false positives/day}$$
$$Cost_{FP} = 10 \times (15\text{min} \times \$50/\text{hr} + \$100) = \$225/\text{day}$$

### Strategies for Reducing False Positives

| Strategy | Implementation | Impact |
|----------|---------------|--------|
| Per-user Bayesian training | Allow users to report FP/FN, retrain model | Reduces FP by 20-40% |
| Allowlist trusted senders | HAT sender groups with ACCEPT | Eliminates FP for known senders |
| Tune spam thresholds | Raise threshold for executive mail policies | Reduces FP for critical users |
| DKIM/DMARC passing | Reduce spam score for authenticated mail | Fewer FP from legitimate bulk senders |
| End-user quarantine | Let users release their own false positives | Reduces admin burden |
| Quarantine digest | Daily notification of quarantined messages | Users self-service FP release |
| Safe/blocked sender lists | Per-user or per-domain lists | Fine-grained control |

### Feedback Loop Architecture

```
User reports FP         Admin reviews           System learns
       |                      |                      |
       |--- Release from ---->|                      |
       |    quarantine        |                      |
       |                      |--- Adjust rules ---->|
       |                      |    (allowlist,        |
       |                      |     tune threshold)   |
       |                      |                      |
       |                      |<-- Updated model ----|
       |                      |                      |
       | [Future similar      |                      |
       |  messages pass]      |                      |
```

$$Accuracy_{t+1} = Accuracy_t + \alpha \times (Feedback_{FP} + Feedback_{FN})$$

Where $\alpha$ is the learning rate. Continuous feedback improves accuracy over time, but care must be taken to avoid adversarial training (attackers deliberately training the filter to accept malicious patterns).

---

## See Also

- tls, pki, dns, cryptography, cisco-ise

## References

- [Cisco Secure Email Administrator Guide](https://www.cisco.com/c/en/us/td/docs/security/esa/esa15-0/user_guide/b_ESA_Admin_Guide_15-0.html)
- [RFC 5321 — Simple Mail Transfer Protocol](https://www.rfc-editor.org/rfc/rfc5321)
- [RFC 7208 — Sender Policy Framework (SPF)](https://www.rfc-editor.org/rfc/rfc7208)
- [RFC 6376 — DomainKeys Identified Mail (DKIM) Signatures](https://www.rfc-editor.org/rfc/rfc6376)
- [RFC 7489 — Domain-based Message Authentication (DMARC)](https://www.rfc-editor.org/rfc/rfc7489)
- [RFC 8551 — S/MIME 4.0 Message Specification](https://www.rfc-editor.org/rfc/rfc8551)
- [RFC 4880 — OpenPGP Message Format](https://www.rfc-editor.org/rfc/rfc4880)
- [RFC 8461 — SMTP MTA Strict Transport Security (MTA-STS)](https://www.rfc-editor.org/rfc/rfc8461)
- [RFC 7672 — SMTP Security via Opportunistic DANE TLS](https://www.rfc-editor.org/rfc/rfc7672)
- [RFC 8617 — Authenticated Received Chain (ARC)](https://www.rfc-editor.org/rfc/rfc8617)
- [RFC 8463 — DKIM Ed25519-SHA256 Algorithm](https://www.rfc-editor.org/rfc/rfc8463)
- [Cisco Talos Intelligence Group](https://talosintelligence.com/)
