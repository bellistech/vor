# Email Security (SPF, DKIM, DMARC, ARC, BIMI, MTA-STS, DANE)

Unified reference for email authentication, policy enforcement, and transport security.

## SPF (Sender Policy Framework)

### Record Syntax

```bash
# Basic SPF record in DNS TXT
# v=spf1 <mechanisms> <qualifier>all
# Qualifiers: + (pass), - (fail), ~ (softfail), ? (neutral)

# Allow server IPs and includes
v=spf1 ip4:203.0.113.0/24 ip6:2001:db8::/32 include:_spf.google.com include:spf.protection.outlook.com mx a -all

# Mechanisms (evaluated left to right):
# ip4:CIDR      - match IPv4 range
# ip6:CIDR      - match IPv6 range
# include:domain - recurse into domain's SPF
# a              - match domain's A/AAAA records
# a:other.com    - match other domain's A records
# mx             - match domain's MX hosts
# mx:other.com   - match other domain's MX hosts
# exists:macro   - match if DNS lookup succeeds
# redirect=domain - use another domain's SPF entirely
```

### SPF Lookup Limit

```bash
# SPF has a 10 DNS lookup limit (RFC 7208 Section 4.6.4)
# include, a, mx, exists, redirect each cost 1 lookup
# ip4, ip6, all cost 0 lookups

# Check SPF lookup count
dig +short TXT example.com | grep spf
# Use online tools: https://www.kitterman.com/spf/validate.html

# Flatten SPF to reduce lookups
# Replace include: with resolved ip4:/ip6: entries
# Trade-off: must update when provider IPs change
```

### Testing SPF

```bash
# Query SPF record
dig +short TXT example.com | grep "v=spf1"

# Test with a specific IP
# Using pyspf
python3 -c "import spf; print(spf.check2(i='203.0.113.10', s='user@example.com', h='mail.example.com'))"

# Check via email headers
# Look for: Received-SPF: pass/fail/softfail/neutral
```

## DKIM (DomainKeys Identified Mail)

### Key Generation and Rotation

```bash
# Generate 2048-bit RSA key pair
opendkim-genkey -b 2048 -d example.com -s selector2024 -r

# Output: selector2024.private (signing key), selector2024.txt (DNS record)

# Install private key
cp selector2024.private /etc/opendkim/keys/example.com/
chown opendkim:opendkim /etc/opendkim/keys/example.com/selector2024.private
chmod 600 /etc/opendkim/keys/example.com/selector2024.private

# Add DNS TXT record from selector2024.txt:
# selector2024._domainkey.example.com. IN TXT "v=DKIM1; k=rsa; p=MIIBIjANBg..."

# Key rotation strategy:
# 1. Generate new key with new selector (e.g., selector2025)
# 2. Publish new DNS record, wait for propagation (48h)
# 3. Switch signing to new selector
# 4. Keep old DNS record for 7 days (in-flight verification)
# 5. Remove old DNS record
```

### OpenDKIM Configuration

```bash
# /etc/opendkim.conf
# Syslog                yes
# Domain                example.com
# Selector              selector2024
# KeyFile               /etc/opendkim/keys/example.com/selector2024.private
# Socket                inet:8891@localhost
# Canonicalization       relaxed/simple
# Mode                  sv   (sign and verify)
# SignatureAlgorithm     rsa-sha256
# InternalHosts          /etc/opendkim/TrustedHosts
# KeyTable               /etc/opendkim/KeyTable
# SigningTable           /etc/opendkim/SigningTable

# /etc/opendkim/KeyTable
# selector2024._domainkey.example.com example.com:selector2024:/etc/opendkim/keys/example.com/selector2024.private

# /etc/opendkim/SigningTable
# *@example.com    selector2024._domainkey.example.com

# Verify a DKIM signature
opendkim-testkey -d example.com -s selector2024 -vvv
```

## DMARC (Domain-based Message Authentication, Reporting, and Conformance)

```bash
# DNS TXT record at _dmarc.example.com
# v=DMARC1; p=reject; sp=quarantine; rua=mailto:dmarc-agg@example.com; ruf=mailto:dmarc-forensic@example.com; adkim=s; aspf=s; pct=100; fo=1

# Tag breakdown:
# p=       policy: none | quarantine | reject
# sp=      subdomain policy (inherits p if absent)
# rua=     aggregate report URI (daily XML reports)
# ruf=     forensic/failure report URI (per-message)
# adkim=   DKIM alignment: s (strict) | r (relaxed, default)
# aspf=    SPF alignment: s (strict) | r (relaxed, default)
# pct=     percentage of messages to apply policy (1-100)
# fo=      failure reporting: 0 (both fail) | 1 (either fail) | d (DKIM) | s (SPF)
# ri=      reporting interval in seconds (default 86400)

# Recommended rollout:
# Phase 1: p=none (monitor only, collect reports)
# Phase 2: p=quarantine; pct=10 (test with small percentage)
# Phase 3: p=quarantine; pct=100
# Phase 4: p=reject; pct=100

# Query DMARC record
dig +short TXT _dmarc.example.com
```

### Aggregate Report Analysis

```bash
# Reports arrive as gzipped XML (RFC 7489 Appendix C)
# Extract and parse
gunzip report.xml.gz
xmllint --format report.xml

# Key fields in aggregate reports:
# <policy_published> - your published DMARC policy
# <record><row><source_ip> - sending IP
# <record><row><count> - number of messages
# <record><row><policy_evaluated><dkim> - pass/fail
# <record><row><policy_evaluated><spf> - pass/fail
# <record><row><policy_evaluated><disposition> - none/quarantine/reject

# Parse with open-source tools
# parsedmarc: pip install parsedmarc
parsedmarc -i report.xml.gz -o parsed_output/
parsedmarc --elasticsearch https://es.example.com:9200 report.xml.gz
```

## ARC (Authenticated Received Chain)

```bash
# ARC preserves authentication results across forwarding hops
# Three headers added by each intermediary:
# ARC-Authentication-Results: i=1; mx.forwarder.com; dkim=pass; spf=fail; dmarc=fail
# ARC-Message-Signature: i=1; a=rsa-sha256; d=forwarder.com; s=arc2024; ...
# ARC-Seal: i=1; a=rsa-sha256; d=forwarder.com; s=arc2024; cv=none; ...

# cv= (chain validation): none (first), pass, fail
# i=  (instance number): increments at each hop

# OpenARC configuration (milter)
# /etc/openarc.conf
# Socket          inet:8894@localhost
# Domain          forwarder.com
# Selector        arc2024
# KeyFile         /etc/openarc/keys/arc2024.private
# AuthservID      mx.forwarder.com
# Mode            sv

# Postfix integration
# smtpd_milters = inet:localhost:8891, inet:localhost:8893, inet:localhost:8894
```

## BIMI (Brand Indicators for Message Identification)

```bash
# DNS TXT record at default._bimi.example.com
# v=BIMI1; l=https://example.com/brand/logo.svg; a=https://example.com/brand/vmc.pem

# l= SVG logo URL (Tiny PS profile, square, under 32KB)
# a= VMC (Verified Mark Certificate) from DigiCert/Entrust

# Requirements:
# - DMARC p=quarantine or p=reject (p=none disqualifies)
# - Valid SVG Tiny PS logo
# - VMC certificate (for Gmail/Yahoo badge display)

# Validate BIMI record
dig +short TXT default._bimi.example.com
```

## MTA-STS (Mail Transfer Agent Strict Transport Security)

```bash
# 1. Publish DNS TXT record
# _mta-sts.example.com. IN TXT "v=STSv1; id=20240101"

# 2. Host policy at https://mta-sts.example.com/.well-known/mta-sts.txt
# version: STSv1
# mode: enforce
# mx: mail.example.com
# mx: backup.example.com
# max_age: 604800

# mode: testing | enforce | none
# max_age: seconds to cache (604800 = 1 week)

# 3. Optional: TLSRPT for reporting
# _smtp._tls.example.com. IN TXT "v=TLSRPTv1; rua=mailto:tls-reports@example.com"
```

## DANE/TLSA (DNS-based Authentication of Named Entities)

```bash
# TLSA record binds a TLS certificate to a DNS name via DNSSEC
# _25._tcp.mail.example.com. IN TLSA 3 1 1 <sha256hash>

# Fields: usage selector matching-type
# Usage: 0=CA, 1=EE pinning, 2=trust anchor, 3=domain-issued (most common)
# Selector: 0=full cert, 1=public key only
# Matching: 0=exact, 1=SHA-256, 2=SHA-512

# Generate TLSA hash from certificate
openssl x509 -in cert.pem -noout -pubkey | \
  openssl pkey -pubin -outform DER | \
  openssl dgst -sha256 -binary | \
  xxd -p -c 64

# Verify DANE with ldns
ldns-dane verify mail.example.com 25

# Requires DNSSEC-signed zone
# Postfix DANE support:
# smtp_tls_security_level = dane
# smtp_dns_support_level = dnssec
```

## Comprehensive Verification

```bash
# Test all records for a domain
dig +short TXT example.com | grep spf
dig +short TXT selector2024._domainkey.example.com
dig +short TXT _dmarc.example.com
dig +short TXT default._bimi.example.com
dig +short TXT _mta-sts.example.com
dig +short TXT _smtp._tls.example.com
dig +short TLSA _25._tcp.mail.example.com

# Send test email and inspect headers for:
# Authentication-Results: dkim=pass; spf=pass; dmarc=pass
# ARC-Authentication-Results (if forwarded)
# Received-SPF: pass

# Online tools:
# https://mxtoolbox.com/SuperTool.aspx
# https://mail-tester.com/
# https://dmarcian.com/dmarc-inspector/
```

## Tips

- Start DMARC at `p=none` to collect reports before enforcing -- jumping to `p=reject` can block legitimate mail.
- Keep SPF records under 10 DNS lookups; flatten includes for high-volume senders.
- Use 2048-bit RSA keys for DKIM; 1024-bit is considered weak and may be rejected.
- Rotate DKIM selectors annually; overlap old and new DNS records for 7 days during rotation.
- Set `adkim=s` (strict alignment) to prevent subdomain spoofing via DKIM.
- ARC is essential for mailing lists and forwarding services that break SPF/DKIM alignment.
- BIMI requires `p=quarantine` or `p=reject` in DMARC -- `p=none` disqualifies your logo.
- MTA-STS starts in `mode: testing` with TLSRPT reporting before switching to `mode: enforce`.
- DANE/TLSA requires DNSSEC on your zone; without it, TLSA records are ignored.
- Monitor DMARC aggregate reports weekly to catch shadow IT sending mail as your domain.
- Use `fo=1` in DMARC to get failure reports when either SPF or DKIM fails, not just both.
- Keep `rua=` and `ruf=` pointing to monitored mailboxes; stale addresses mean silent policy failures.

## See Also

- postfix (MTA configuration, milter integration)
- dovecot (IMAP server receiving authenticated mail)
- wireshark (SMTP/TLS packet inspection)

## References

- [RFC 7208 - SPF](https://datatracker.ietf.org/doc/html/rfc7208)
- [RFC 6376 - DKIM](https://datatracker.ietf.org/doc/html/rfc6376)
- [RFC 7489 - DMARC](https://datatracker.ietf.org/doc/html/rfc7489)
- [RFC 8617 - ARC](https://datatracker.ietf.org/doc/html/rfc8617)
- [RFC 8461 - MTA-STS](https://datatracker.ietf.org/doc/html/rfc8461)
- [RFC 7671 - DANE/TLSA](https://datatracker.ietf.org/doc/html/rfc7671)
- [BIMI Working Group](https://bimigroup.org/implementation-guide/)
- [MXToolbox](https://mxtoolbox.com/SuperTool.aspx)
