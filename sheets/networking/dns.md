# DNS (Domain Name System Protocol and Administration)

A protocol-level and server-side DNS reference covering record types, zone files, DNSSEC, resolution mechanics, server configuration, and troubleshooting.

## Record Types

### Common Records

```bash
# A — IPv4 address mapping
# example.com.  IN  A  93.184.216.34

# AAAA — IPv6 address mapping
# example.com.  IN  AAAA  2606:2800:220:1:248:1893:25c8:1946

# CNAME — canonical name (alias); cannot coexist with other records at same name
# www.example.com.  IN  CNAME  example.com.

# MX — mail exchange; priority (lower = preferred) + mail server
# example.com.  IN  MX  10  mail.example.com.
# example.com.  IN  MX  20  mail2.example.com.

# NS — nameserver delegation
# example.com.  IN  NS  ns1.example.com.
# example.com.  IN  NS  ns2.example.com.

# PTR — reverse DNS (maps IP to name)
# 34.216.184.93.in-addr.arpa.  IN  PTR  example.com.

# TXT — arbitrary text (SPF, DKIM, domain verification, etc.)
# example.com.  IN  TXT  "v=spf1 mx ip4:93.184.216.0/24 -all"

# SOA — start of authority (one per zone, defines zone parameters)
# example.com.  IN  SOA  ns1.example.com. admin.example.com. (
#     2024010101  ; serial (YYYYMMDDNN convention)
#     3600        ; refresh (seconds)
#     900         ; retry (seconds)
#     604800      ; expire (seconds)
#     86400       ; negative caching TTL (seconds)
# )
```

### Service and Security Records

```bash
# SRV — service locator (proto, priority, weight, port, target)
# _sip._tcp.example.com.  IN  SRV  10 60 5060 sipserver.example.com.
# Format: _service._proto.name  TTL  IN  SRV  priority weight port target

# CAA — Certificate Authority Authorization (controls which CAs can issue certs)
# example.com.  IN  CAA  0 issue "letsencrypt.org"
# example.com.  IN  CAA  0 issuewild ";"           # disallow wildcard certs
# example.com.  IN  CAA  0 iodef "mailto:sec@example.com"

# TLSA — DANE TLS certificate association (binds cert to DNS)
# _443._tcp.example.com.  IN  TLSA  3 1 1 <sha256hash>
# Usage field: 0=CA, 1=EE pinned, 2=trust anchor, 3=EE direct

# NAPTR — Naming Authority Pointer (used in SIP, ENUM, S-NAPTR)
# example.com.  IN  NAPTR  100 10 "u" "E2U+sip" "!^.*$!sip:info@example.com!" .
```

### DNSSEC Records

```bash
# DNSKEY — public key for zone signing
# example.com.  IN  DNSKEY  257 3 13 <base64key>
# Flags: 256 = ZSK (Zone Signing Key), 257 = KSK (Key Signing Key)

# DS — Delegation Signer (hash of child KSK, placed in parent zone)
# example.com.  IN  DS  12345 13 2 <sha256hash>

# RRSIG — signature over a record set (auto-generated during signing)
# NSEC / NSEC3 — authenticated denial of existence
```

## Zone File Format

### Structure and Directives

```bash
# $ORIGIN sets the default domain suffix for unqualified names
$ORIGIN example.com.

# $TTL sets the default TTL for records that do not specify one
$TTL 3600

# $INCLUDE pulls in another zone file
$INCLUDE /etc/bind/zones/subzone.db

# @ is shorthand for $ORIGIN
@  IN  SOA  ns1.example.com. admin.example.com. (
    2024010101  ; serial — MUST increment on every change
    3600        ; refresh — how often secondaries check for updates
    900         ; retry — how often to retry after failed refresh
    604800      ; expire — when secondary stops answering if primary is down
    86400       ; minimum — negative caching TTL (NXDOMAIN cache time)
)
```

### Example Zone File

```bash
$ORIGIN example.com.
$TTL 3600

@       IN  SOA   ns1.example.com. hostmaster.example.com. (
                   2024032501  ; serial
                   3600        ; refresh
                   900         ; retry
                   604800      ; expire
                   86400 )     ; negative TTL

        IN  NS    ns1.example.com.
        IN  NS    ns2.example.com.

        IN  MX    10 mail.example.com.
        IN  MX    20 mail2.example.com.

        IN  A     93.184.216.34
        IN  AAAA  2606:2800:220:1:248:1893:25c8:1946
        IN  TXT   "v=spf1 mx -all"
        IN  CAA   0 issue "letsencrypt.org"

ns1     IN  A     93.184.216.10
ns2     IN  A     93.184.216.11
mail    IN  A     93.184.216.20
mail2   IN  A     93.184.216.21
www     IN  CNAME @

_sip._tcp  IN  SRV  10 60 5060 sip.example.com.
```

## DNSSEC

### Signing a Zone

```bash
# Generate Zone Signing Key (ZSK)
dnssec-keygen -a ECDSAP256SHA256 -n ZONE example.com
# Produces: Kexample.com.+013+NNNNN.key and .private

# Generate Key Signing Key (KSK)
dnssec-keygen -a ECDSAP256SHA256 -n ZONE -f KSK example.com

# Sign the zone file
dnssec-signzone -A -3 $(head -c 16 /dev/urandom | od -A n -t x1 | tr -d ' ') \
  -N INCREMENT -o example.com -t db.example.com

# Output: db.example.com.signed (use this in named.conf)
```

### DS Record and Key Rotation

```bash
# Generate DS record to submit to parent/registrar
dnssec-dsfromkey Kexample.com.+013+NNNNN.key
# Output: example.com. IN DS 12345 13 2 <hash>

# Key rotation workflow:
# 1. Generate new key pair
# 2. Add new DNSKEY to zone (pre-publish)
# 3. Wait for old TTL to expire
# 4. Sign zone with new key
# 5. Update DS at parent (for KSK rotation)
# 6. Remove old key after propagation

# Verify DNSSEC chain
dig +dnssec +cd example.com
dig DS example.com @parent-ns
delv example.com           # BIND's DNSSEC-aware lookup tool
```

### NSEC vs NSEC3

```bash
# NSEC — proves a name does not exist by listing the next existing name
# Problem: allows zone walking (enumerate all records)

# NSEC3 — hashed denial of existence (prevents casual zone walking)
# Uses salted hashes of names
# NSEC3PARAM record controls hash parameters

# Check NSEC3 parameters
dig NSEC3PARAM example.com
```

## DNS Resolution Flow

### Recursive vs Iterative

```bash
# Recursive: client asks resolver, resolver does all the work
# Client -> Recursive Resolver -> Root -> TLD -> Authoritative -> back to client

# Iterative: server returns a referral, client follows it
# Used between recursive resolvers and authoritative servers

# Trace the full resolution path
dig +trace example.com

# Check if a server does recursion
dig @8.8.8.8 example.com +norecurse
# If NOERROR with answer: authoritative or cached
# If NOERROR with referral: iterative only
```

### Caching and Negative Caching

```bash
# Positive cache: stores answers for TTL seconds
# Negative cache: stores NXDOMAIN for SOA minimum TTL

# Check the TTL of a cached response (decrements over time)
dig example.com | grep -E '^\S+\s+[0-9]+'
# The number after the name is the remaining TTL

# Force bypass cache (query authoritative directly)
dig @ns1.example.com example.com +norecurse

# Flush local DNS cache
sudo systemd-resolve --flush-caches     # systemd-resolved
sudo rndc flush                         # BIND
sudo unbound-control flush_zone example.com  # Unbound
```

## Split-Horizon DNS

```bash
# Different answers for internal vs external queries
# BIND view-based configuration:

# In named.conf:
# view "internal" {
#     match-clients { 10.0.0.0/8; 172.16.0.0/12; 192.168.0.0/16; };
#     zone "example.com" {
#         type master;
#         file "/etc/bind/zones/internal.example.com.db";
#     };
# };
#
# view "external" {
#     match-clients { any; };
#     zone "example.com" {
#         type master;
#         file "/etc/bind/zones/external.example.com.db";
#     };
# };

# Test split-horizon by querying from different source IPs
dig @dns-server example.com              # uses default source
dig @dns-server example.com -b 10.0.0.1  # specify source IP
```

## DNS over HTTPS / TLS

### DoH (DNS over HTTPS)

```bash
# Encrypts DNS queries inside HTTPS (port 443)
# Prevents ISP/middlebox snooping on DNS queries

# Test DoH with curl
curl -s -H 'accept: application/dns-json' \
  'https://dns.google/resolve?name=example.com&type=A' | jq .

# Configure systemd-resolved for DoT (DoH not natively supported yet)
# /etc/systemd/resolved.conf:
# [Resolve]
# DNS=1.1.1.1#cloudflare-dns.com
# DNSOverTLS=yes
```

### DoT (DNS over TLS)

```bash
# Encrypts DNS queries over TLS (port 853)
# Test with kdig (from knot-dnsutils)
kdig @1.1.1.1 +tls example.com

# Unbound as a DoT forwarder:
# forward-zone:
#     name: "."
#     forward-tls-upstream: yes
#     forward-addr: 1.1.1.1@853#cloudflare-dns.com
#     forward-addr: 8.8.8.8@853#dns.google
```

## Common Server Configurations

### BIND (named)

```bash
# Check BIND configuration syntax
named-checkconf /etc/bind/named.conf
named-checkzone example.com /etc/bind/zones/db.example.com

# Reload zone after changes
sudo rndc reload example.com

# View BIND cache statistics
sudo rndc stats
cat /var/named/data/named_stats.txt

# Basic zone definition in named.conf
# zone "example.com" {
#     type master;
#     file "/etc/bind/zones/db.example.com";
#     allow-transfer { 10.0.0.2; };      # secondary NS IP
#     also-notify { 10.0.0.2; };
# };
```

### Unbound

```bash
# Check Unbound config
unbound-checkconf

# Reload Unbound
sudo unbound-control reload

# Dump cache
sudo unbound-control dump_cache > cache.txt

# Load cache
sudo unbound-control load_cache < cache.txt

# Basic unbound.conf for a caching resolver:
# server:
#     interface: 0.0.0.0
#     access-control: 10.0.0.0/8 allow
#     do-ip6: yes
#     prefetch: yes
#     num-threads: 4
#     cache-max-ttl: 86400
#     cache-min-ttl: 300
```

## Troubleshooting

### Diagnostic Commands

```bash
# Full resolution trace (follow the delegation chain)
dig +trace example.com

# Query a specific nameserver
dig @ns1.example.com example.com

# Check SOA serial (verify zone update propagation)
dig SOA example.com +short
# Compare serial across nameservers
dig @ns1.example.com SOA example.com +short
dig @ns2.example.com SOA example.com +short

# Check all records at a name
dig ANY example.com    # note: many servers block ANY queries now

# Test zone transfer (if allowed)
dig AXFR example.com @ns1.example.com
```

### Response Codes

```bash
# NOERROR  — query successful (may have 0 answers for empty non-terminal)
# NXDOMAIN — name does not exist in the zone
# SERVFAIL — server failed to process query (often DNSSEC validation failure)
# REFUSED  — server refuses to answer (recursion not allowed, ACL, etc.)
# FORMERR  — malformed query

# Check response status
dig example.com | grep "status:"
# ;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 12345

# SERVFAIL debugging — try bypassing DNSSEC validation
dig +cd example.com    # cd = checking disabled
# If +cd works but normal query fails: DNSSEC problem
```

### TTL Debugging

```bash
# See actual TTL values (counts down from original)
dig example.com

# See original TTL (query authoritative directly)
dig @ns1.example.com example.com

# Common TTL issues:
# - Changed record but old answer persists: wait for old TTL to expire
# - Before a migration, lower TTL to 300 (5 min) at least 48h in advance
# - After migration, raise TTL back to 3600+ for performance
```

## DNS Security

### Cache Poisoning

```bash
# Attacker sends forged responses to poison resolver cache
# Mitigations:
# - Source port randomization (enabled by default on modern resolvers)
# - DNSSEC validation
# - DNS cookies (RFC 7873)

# Check if your resolver randomizes source ports
dig +short porttest.dns-oarc.net TXT
```

### Amplification Attacks

```bash
# DNS used as DDoS amplifier (small query -> large response)
# Mitigations:
# - Rate limiting (RRL) in BIND:
#   rate-limit { responses-per-second 10; };
# - Disable open recursion
#   allow-recursion { localhost; 10.0.0.0/8; };
# - Block ANY queries from external sources
```

### DANE / TLSA

```bash
# DANE uses TLSA records to pin TLS certificates to DNS (requires DNSSEC)
# Verify TLSA record
dig TLSA _443._tcp.example.com

# Generate a TLSA record from a certificate
openssl x509 -in cert.pem -outform DER | \
  openssl dgst -sha256 -binary | \
  xxd -p -c 32
# Use output as: _443._tcp.example.com. IN TLSA 3 1 1 <hash>
```

## systemd-resolved

```bash
# Check current DNS configuration
resolvectl status

# Query through systemd-resolved
resolvectl query example.com

# Flush DNS cache
resolvectl flush-caches

# Show cache statistics
resolvectl statistics

# Configuration file
# /etc/systemd/resolved.conf
# [Resolve]
# DNS=1.1.1.1 8.8.8.8
# FallbackDNS=9.9.9.9
# Domains=~.                  # route all queries through this resolver
# DNSSEC=allow-downgrade
# DNSOverTLS=opportunistic
# Cache=yes
# CacheFromLocalhost=no

# Restart resolved after config changes
sudo systemctl restart systemd-resolved

# Symlink resolv.conf for compatibility
sudo ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
```

## Tips

- Always increment the SOA serial number when editing zone files. Use the YYYYMMDDNN convention and never let the serial go backward.
- CNAME records cannot coexist with any other record type at the same name. This means the zone apex (example.com) usually cannot be a CNAME (use ALIAS/ANAME if your DNS provider supports it).
- Before a DNS migration or IP change, lower the TTL to 300 seconds at least 48 hours in advance, then raise it back after the change is confirmed.
- `dig +trace` is the single most useful DNS debugging command. It shows the full delegation chain and reveals where resolution breaks.
- If `dig` works but applications fail, check `/etc/nsswitch.conf` (hosts line) and `/etc/resolv.conf` for local resolver issues.
- SERVFAIL often means a DNSSEC validation failure. Test with `dig +cd` to confirm; if that works, the zone has a DNSSEC configuration problem.
- When setting up a new zone, always test with `named-checkzone` or `nsd-checkzone` before loading it into production.
- For internal/private DNS, Unbound (caching resolver) + NSD (authoritative) is a lightweight alternative to BIND for split roles.
- CAA records are checked by CAs before issuing certificates. Set them to prevent unauthorized certificate issuance.
- Keep zone transfer (AXFR) restricted to known secondary nameservers. Open zone transfers leak your entire DNS inventory.

## References

- [RFC 1034 — Domain Names: Concepts and Facilities](https://www.rfc-editor.org/rfc/rfc1034)
- [RFC 1035 — Domain Names: Implementation and Specification](https://www.rfc-editor.org/rfc/rfc1035)
- [RFC 4033 — DNS Security Introduction and Requirements (DNSSEC)](https://www.rfc-editor.org/rfc/rfc4033)
- [RFC 7858 — DNS over Transport Layer Security (DoT)](https://www.rfc-editor.org/rfc/rfc7858)
- [RFC 8484 — DNS Queries over HTTPS (DoH)](https://www.rfc-editor.org/rfc/rfc8484)
- [RFC 9460 — Service Binding and Parameter Specification via the DNS (SVCB and HTTPS RRs)](https://www.rfc-editor.org/rfc/rfc9460)
- [IANA DNS Parameters — Resource Record Types](https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml)
- [IANA Root Zone Database](https://www.iana.org/domains/root/db)
- [Cloudflare — What Is DNS?](https://www.cloudflare.com/learning/dns/what-is-dns/)
- [ISC BIND 9 Documentation](https://bind9.readthedocs.io/en/latest/)
- [Unbound DNS Resolver Documentation](https://unbound.docs.nlnetlabs.nl/en/latest/)
- [PowerDNS Documentation](https://doc.powerdns.com/)
