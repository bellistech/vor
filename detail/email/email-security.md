# The Mathematics of Email Security -- Cryptographic Authentication and Policy Verification

> *Email authentication is a chain of cryptographic proofs: SPF constrains the envelope, DKIM signs the content, DMARC aligns both to the visible sender, and ARC preserves the chain across forwarding hops.*

---

## 1. SPF Evaluation (Set Membership and DNS Recursion)

### The Problem

SPF verification asks: "Is the sending IP a member of the authorized set for this domain?" The authorized set is defined recursively through `include:` mechanisms, with a hard limit of 10 DNS lookups to prevent amplification attacks. Each `include:` expands into another SPF record that may itself contain includes, forming a tree.

### The Formula

Let $S(d)$ be the authorized IP set for domain $d$. Each SPF record defines:

$$S(d) = \bigcup_{m \in \text{mechanisms}(d)} \text{resolve}(m)$$

The DNS lookup cost of evaluating domain $d$:

$$L(d) = |\{m : m \in \{a, mx, include, exists, redirect\}\}| + \sum_{i \in \text{includes}(d)} L(i)$$

The constraint is:

$$L(d) \leq 10$$

For a domain with $n$ `include:` directives, each including $k$ sub-includes, the worst case:

$$L_{worst} = n + n \cdot k + n \cdot k^2 + \cdots$$

This geometric series shows why deeply nested includes rapidly exhaust the 10-lookup budget.

### Worked Examples

**Example: Calculating lookup cost for a typical enterprise SPF.**

```
v=spf1 include:_spf.google.com include:spf.protection.outlook.com include:sendgrid.net ip4:203.0.113.0/24 -all
```

Lookup tree:
- `include:_spf.google.com` (1) expands to 3 more includes = 4
- `include:spf.protection.outlook.com` (1) expands to 1 include = 2
- `include:sendgrid.net` (1) expands to 2 includes = 3
- `ip4:` = 0

Total: $L = 4 + 2 + 3 = 9$ lookups. One more provider include would exceed the limit.

**Flattening saves lookups** by resolving includes to explicit IP ranges:

$$L_{flat} = 0 \quad \text{(all ip4/ip6 mechanisms)}$$

Trade-off: flattened records break when providers change IPs.

## 2. DKIM Signatures (RSA Digital Signatures)

### The Problem

DKIM signs a canonicalized subset of email headers and body using RSA (or Ed25519). The verifier retrieves the public key from DNS and checks the signature. Security depends on key length: an $n$-bit RSA key requires approximately $e^{(64/9 \cdot n \cdot (\ln 2)^2)^{1/3}}$ operations to factor via the General Number Field Sieve.

### The Formula

RSA signature generation for message hash $H(m)$:

$$\sigma = H(m)^d \mod N$$

Verification:

$$H(m) \stackrel{?}{=} \sigma^e \mod N$$

where $(N, e)$ is the public key and $d$ is the private key. The security parameter is $|N|$ in bits.

GNFS complexity for factoring $N$:

$$T(N) = \exp\left(\left(\frac{64}{9}\right)^{1/3} (\ln N)^{1/3} (\ln \ln N)^{2/3}\right)$$

For key sizes:

$$T(1024\text{-bit}) \approx 2^{86} \text{ operations}$$
$$T(2048\text{-bit}) \approx 2^{117} \text{ operations}$$

### Worked Examples

**Example: DKIM canonicalization and hash.**

In `relaxed/simple` canonicalization:
- Headers: unfold lines, lowercase names, compress whitespace
- Body: no modification (simple)

The `bh=` tag is the base64 of SHA-256 over the canonicalized body:

$$bh = \text{base64}(\text{SHA-256}(\text{canon}_{simple}(\text{body})))$$

For body "Hello World\r\n":

$$\text{SHA-256}(\text{"Hello World\textbackslash r\textbackslash n"}) = \text{a591a6d40bf420...}$$

$$bh = \text{pZGm1Av0IEBKARczz7exkNYsZb8LzaMrV7J32a2fFG4=}$$

## 3. DMARC Alignment (Identifier Matching Logic)

### The Problem

DMARC requires that the domain in the `From:` header aligns with either the SPF-authenticated domain (envelope sender) or the DKIM-signing domain. Alignment can be strict (exact match) or relaxed (organizational domain match).

### The Formula

Let $d_{from}$ be the `From:` header domain, $d_{spf}$ the envelope sender domain, and $d_{dkim}$ the DKIM `d=` domain.

The organizational domain function $\text{org}(d)$ strips the leftmost label below the public suffix:

$$\text{org}(\text{mail.sub.example.com}) = \text{example.com}$$

DMARC passes if:

$$\text{DMARC} = \begin{cases} \text{pass} & \text{if } \text{SPF}_{align} \lor \text{DKIM}_{align} \\ \text{fail} & \text{otherwise} \end{cases}$$

where:

$$\text{SPF}_{align} = \begin{cases} d_{from} = d_{spf} & \text{(strict)} \\ \text{org}(d_{from}) = \text{org}(d_{spf}) & \text{(relaxed)} \end{cases}$$

$$\text{DKIM}_{align} = \begin{cases} d_{from} = d_{dkim} & \text{(strict)} \\ \text{org}(d_{from}) = \text{org}(d_{dkim}) & \text{(relaxed)} \end{cases}$$

### Worked Examples

**Example: Mailing list forwarding breaks alignment.**

Original: `From: user@example.com`, SPF passes for `example.com`, DKIM signed by `example.com`.

After list forwards: envelope sender becomes `list-bounce@lists.org`, so:
- $d_{spf} = \text{lists.org}$, $d_{from} = \text{example.com}$ -- SPF alignment fails (strict and relaxed)
- DKIM may survive if list doesn't modify signed headers -- DKIM alignment passes

If the list modifies the Subject (common), DKIM breaks too: DMARC fails entirely. This is why ARC exists.

## 4. ARC Chain Validation (Cryptographic Chain of Custody)

### The Problem

ARC creates a chain of cryptographic attestations as mail passes through intermediaries. Each hop $i$ adds three headers. The chain is valid only if every seal from $1$ to $n$ validates, and each hop's authentication results are signed.

### The Formula

ARC chain validity for $n$ hops:

$$\text{ARC}_{valid} = \bigwedge_{i=1}^{n} \text{verify}(\text{ARC-Seal}_i) \;\land\; \bigwedge_{i=1}^{n} \text{verify}(\text{ARC-Message-Signature}_i)$$

The chain validation status at hop $i$:

$$cv_i = \begin{cases} \text{none} & \text{if } i = 1 \\ \text{pass} & \text{if } \bigwedge_{j=1}^{i-1} \text{verify}(\text{Seal}_j) \\ \text{fail} & \text{otherwise} \end{cases}$$

If $cv_i = \text{fail}$ at any hop, the entire chain is untrusted and the receiving MTA falls back to standard DMARC evaluation.

### Worked Examples

**Example: Two-hop forwarding chain.**

1. Origin `example.com` sends to `list@forwarder.org`. Forwarder verifies DKIM (pass), SPF (pass). Adds ARC set $i=1$, $cv=\text{none}$.
2. Forwarder relays to `user@receiver.net`. Receiver verifies:
   - ARC-Seal $i=1$: signature valid over $\text{ARC-Authentication-Results}_1$ -- pass
   - ARC-Message-Signature $i=1$: valid over headers/body at hop 1 -- pass
   - $cv_2 = \text{pass}$ (all prior seals valid)
3. Receiver trusts the chain: original DKIM=pass, SPF=pass per ARC-AR $i=1$. DMARC override: pass.

## 5. DANE/TLSA Certificate Binding (Hash Pinning)

### The Problem

DANE binds a TLS certificate to a DNS name using DNSSEC-signed TLSA records. The TLSA record contains a hash of the certificate or public key. Verification checks that the presented certificate matches the pinned hash.

### The Formula

TLSA matching for selector 1 (public key), matching type 1 (SHA-256):

$$\text{match} = \left(\text{SHA-256}(\text{SubjectPublicKeyInfo}_{cert}) \stackrel{?}{=} \text{TLSA}_{data}\right)$$

The probability of a hash collision (birthday bound):

$$P(\text{collision}) \approx \frac{n^2}{2^{257}}$$

For SHA-256 with $n$ certificates observed globally ($n \approx 10^9$):

$$P \approx \frac{10^{18}}{2^{257}} \approx 10^{-59}$$

### Worked Examples

**Example: Generating and verifying a TLSA record.**

```
openssl x509 -in cert.pem -noout -pubkey | openssl pkey -pubin -outform DER | sha256sum
```

Output: `a]b4c5d6e7f8...` (64 hex chars = 256 bits)

DNS record: `_25._tcp.mail.example.com. IN TLSA 3 1 1 a1b4c5d6e7f8...`

Verification: Postfix connects to port 25, receives certificate, extracts public key, computes SHA-256, compares to TLSA record. Match means the certificate is trusted regardless of CA hierarchy.

## Prerequisites

- RSA cryptography (key generation, signing, verification)
- Hash functions (SHA-256 properties, collision resistance)
- DNS record types (TXT, TLSA, MX) and DNSSEC
- Set theory (union, membership testing)
- Boolean logic (conjunction, disjunction for alignment)
- Public Key Infrastructure (certificates, trust chains)
