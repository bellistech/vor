# DNS (Domain Name System)

The hierarchical, distributed naming system that maps human-friendly names to IP addresses, mail servers, service endpoints, and arbitrary metadata, plus the protocol, record types, zone-file syntax, DNSSEC chain validation, encrypted transports, and operational diagnostics that keep it running.

## Setup

DNS is the layer between user-visible names and routable addresses. Every hostname lookup, every TLS handshake, every email delivery, every Kubernetes service discovery, and every certificate issuance touches DNS. It is the canonical "this just works until it doesn't" system: when DNS is healthy, nothing notices it; when DNS is broken, everything else looks broken first. The protocol is RFC 1034 (concepts) and RFC 1035 (wire format), both from 1987, augmented by hundreds of subsequent RFCs.

```bash
# The protocol — UDP/TCP port 53; DoT 853; DoH 443; DoQ 853
# Wire format: 12-byte header + question(s) + answer(s) + authority(s) + additional(s)
# Hierarchical name space: "."(root) -> ".com"(TLD) -> "example.com"(domain) -> "www"(label)
# Names are read right-to-left: in "www.example.com." the trailing dot is the root.
```

```bash
# Implementations on Linux
named           # ISC BIND9 — reference authoritative+recursive server
unbound         # NLnet Labs — recursive-only, DNSSEC-validating
nsd             # NLnet Labs — minimal authoritative-only server
knot            # CZ.NIC Knot DNS — fast authoritative server
pdns_server     # PowerDNS Authoritative — database-backed
pdns_recursor   # PowerDNS Recursor — recursive-only
dnsmasq         # small DNS+DHCP forwarder — common on home routers
coredns         # Go-based plugin server — Kubernetes default
systemd-resolved  # Linux desktop default — local stub on 127.0.0.53
stubby          # local DoT proxy — listens on 127.0.0.1:53
```

```bash
# Implementations on macOS
mDNSResponder   # Apple's stub resolver and mDNS responder; not configurable directly
discoveryd      # legacy, removed in 10.10+
# Configuration via scutil/networksetup, not /etc/resolv.conf
scutil --dns                                  # show current resolver state
networksetup -getdnsservers Wi-Fi             # show DNS for an interface
```

```bash
# Implementations on Windows
Dnscache        # the local DNS Client service
ipconfig /all                                  # show resolver config per adapter
ipconfig /flushdns                             # clear local cache
ipconfig /displaydns                           # show cached entries
nslookup -type=A example.com 1.1.1.1           # query a specific server
```

```bash
# The resolver chain on Linux (typical glibc order, controlled by /etc/nsswitch.conf)
# 1. application calls getaddrinfo("example.com")
# 2. glibc reads /etc/nsswitch.conf  ->  hosts: files dns
# 3. "files"  -> /etc/hosts is consulted
# 4. "dns"    -> /etc/resolv.conf nameserver(s) are queried over UDP/53
# 5. result is returned (with optional caching layer like nscd / systemd-resolved)
getent hosts example.com                       # uses NSS (files+dns), respects nsswitch
host example.com                               # bypasses NSS, queries DNS directly
```

```bash
# Reality check — "DNS just works until it doesn't"
# Symptoms that look like network failures but are really DNS:
ping example.com                               # "Name or service not known"
curl https://api.example.com                   # "Could not resolve host"
ssh user@server                                # hangs for ~75s before failing
git push                                       # "Could not resolve hostname github.com"
# All of these mean: resolver chain failed, not that the network is down.
ping 8.8.8.8                                   # if this works, network is fine — it's DNS
```

## The Protocol

DNS runs primarily over UDP/53 because it is a request/response protocol where most messages fit in a single packet. TCP/53 is the fallback for large responses (>512 bytes without EDNS0, >4096 bytes typically with EDNS0) and the required transport for zone transfers (AXFR/IXFR). Modern variants encrypt the channel.

```bash
# UDP/53 — default transport
dig @1.1.1.1 example.com                       # standard UDP query
sudo tcpdump -ni any port 53                   # observe wire traffic

# TCP/53 fallback — automatic when UDP response is truncated (TC bit set)
dig @1.1.1.1 example.com +tcp                  # force TCP
dig @ns1.example.com example.com AXFR          # zone transfer ALWAYS uses TCP
dig @1.1.1.1 com DNSKEY                        # DNSKEY responses often need TCP
```

```bash
# The TC (truncated) flag — server signals "response too big for UDP, retry over TCP"
dig @1.1.1.1 example.com +bufsize=512          # force small UDP buffer to trigger TC
# Look for: "flags: qr aa tc rd ra" — the "tc" means truncated
```

```bash
# DNS-over-TLS (DoT) — port 853, RFC 7858
# TLS-encrypted, prevents passive snooping, validated cert
kdig @1.1.1.1 +tls example.com                 # knot-dnsutils kdig with TLS
kdig @1.1.1.1 +tls=1.1.1.1 -p 853 example.com  # explicit
# systemd-resolved config:
#   DNSOverTLS=yes
#   DNS=1.1.1.1#cloudflare-dns.com
```

```bash
# DNS-over-HTTPS (DoH) — port 443, RFC 8484
# Looks like ordinary HTTPS traffic, hard to block at network layer
curl -s -H 'accept: application/dns-message' \
  --data-binary @<(printf '\x00\x00\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x07example\x03com\x00\x00\x01\x00\x01') \
  https://cloudflare-dns.com/dns-query | xxd
# Easier: JSON variant
curl -s -H 'accept: application/dns-json' \
  'https://cloudflare-dns.com/dns-query?name=example.com&type=A' | jq .
```

```bash
# DNS-over-QUIC (DoQ) — port 853, RFC 9250
# Like DoT but over QUIC instead of TCP+TLS — lower latency, 0-RTT
kdig @1.1.1.1 +quic example.com                # if knot built with --enable-quic
```

```bash
# EDNS0 — RFC 6891, the OPT pseudo-RR that extends classic DNS
# Negotiates larger UDP responses (typically 4096), DNSSEC, client-subnet, cookies
dig +edns=0 example.com                        # send OPT pseudo-RR
dig +bufsize=4096 example.com                  # advertise 4096-byte UDP buffer
dig +dnssec example.com                        # DO bit (DNSSEC OK) requires EDNS0
# Without EDNS0, DNSSEC is impossible because RRSIG records overflow 512 bytes.
```

## Resource Record Types — Full Catalog

Every record set has a name (owner), class (almost always IN), TTL, type, and type-specific RDATA. The full list is registered with IANA; the catalog below covers everything you encounter in production.

### A — IPv4 Address (RFC 1035, type 1)

```bash
# Format: <name> <ttl> IN A <ipv4>
example.com.        3600  IN  A      93.184.216.34
www                 300   IN  A      10.0.0.5
api                 60    IN  A      203.0.113.10
mail                3600  IN  A      198.51.100.20
# Multiple A records at the same name = round-robin / multivalue load balancing
api                 60    IN  A      203.0.113.10
api                 60    IN  A      203.0.113.11
api                 60    IN  A      203.0.113.12
```

### AAAA — IPv6 Address (RFC 3596, type 28)

```bash
# Format: <name> <ttl> IN AAAA <ipv6>
example.com.        3600  IN  AAAA   2606:2800:220:1:248:1893:25c8:1946
www                 300   IN  AAAA   2001:db8::1
ipv6only            3600  IN  AAAA   2001:db8:cafe::beef
# Address compression rules (RFC 5952): one zero-run of "::" max, lowercase hex, no leading zeros
```

### CNAME — Canonical Name Alias (RFC 1035, type 5)

```bash
# Format: <name> <ttl> IN CNAME <target>
www                 3600  IN  CNAME  example.com.
shop                3600  IN  CNAME  shop.shopify.com.
docs                3600  IN  CNAME  example.github.io.
# CRITICAL: a CNAME owner cannot have ANY other record (no MX, no TXT, no A) — RFC 1034 §3.6.2
# CRITICAL: zone apex (@) cannot be CNAME because apex must have SOA + NS records.
#   Workarounds: ALIAS / ANAME (provider-specific, e.g. Route53 alias, Cloudflare CNAME flattening)
```

### MX — Mail Exchange (RFC 1035, type 15)

```bash
# Format: <name> <ttl> IN MX <preference> <mailserver>
example.com.        3600  IN  MX     10  mail.example.com.
example.com.        3600  IN  MX     20  mail2.example.com.
example.com.        3600  IN  MX     30  mail-backup.example.net.
# Preference: lower = more preferred. Senders try lowest first, fall back on failure.
# The MX target MUST be a hostname with A/AAAA — NEVER an IP address, NEVER a CNAME.
# "Null MX" disables incoming mail per RFC 7505:
example.com.        3600  IN  MX     0   .
```

### NS — Nameserver Delegation (RFC 1035, type 2)

```bash
# Format: <name> <ttl> IN NS <nameserver>
example.com.        86400 IN  NS     ns1.example.com.
example.com.        86400 IN  NS     ns2.example.com.
# In a parent zone, NS records DELEGATE the child to a different authority:
#   At .com authoritative: example.com. NS ns1.example.com.
#   At example.com authoritative: example.com. NS ns1.example.com.
# These must MATCH or you have lame delegation.
```

### SOA — Start of Authority (RFC 1035, type 6)

```bash
# Exactly one SOA per zone; all fields are mandatory.
# Format:
@  IN  SOA  <mname>  <rname>  ( <serial> <refresh> <retry> <expire> <minimum> )
@  IN  SOA  ns1.example.com. hostmaster.example.com. (
            2024010101    ; SERIAL  — incremented on every change (YYYYMMDDNN)
            7200          ; REFRESH — secondary checks primary every N seconds
            3600          ; RETRY   — wait this long after refresh fails
            1209600       ; EXPIRE  — secondary stops serving after this if primary unreachable
            3600 )        ; MINIMUM — negative-cache TTL (NXDOMAIN cache lifetime)
# RNAME is an email with first . treated as @  ->  hostmaster.example.com = hostmaster@example.com
```

### PTR — Reverse DNS (RFC 1035, type 12)

```bash
# Format: <reversed-ip>.in-addr.arpa.  IN  PTR  <hostname>
34.216.184.93.in-addr.arpa.   3600 IN PTR  example.com.
# IPv6 reverse uses ip6.arpa with reverse-nibble notation (32 hex digits separated by dots):
6.4.9.1.8.c.5.2.3.9.8.1.8.4.2.0.1.0.0.0.0.2.2.0.0.0.8.2.6.0.6.2.ip6.arpa. IN PTR example.com.
# In zone files, $ORIGIN simplifies things:
$ORIGIN 216.184.93.in-addr.arpa.
34  IN  PTR  example.com.
```

### TXT — Free-Form Text (RFC 1035, type 16)

```bash
# Format: <name> <ttl> IN TXT "<string>" ["<string>" ...]
# Each string is max 255 bytes; multiple strings concatenated by clients (DKIM splits long keys this way).
example.com.        3600  IN  TXT    "v=spf1 include:_spf.google.com -all"
_dmarc              3600  IN  TXT    "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"
selector1._domainkey 3600 IN  TXT    "v=DKIM1; k=rsa; p=MIGfMA0GCSq..." "...continued..."
google-site-verification 3600 IN TXT "google-site-verification=abc123xyz"
```

### SRV — Service Locator (RFC 2782, type 33)

```bash
# Format: _<service>._<proto>.<name> <ttl> IN SRV <priority> <weight> <port> <target>
_sip._tcp.example.com.       3600 IN SRV 10  60  5060  sip1.example.com.
_sip._tcp.example.com.       3600 IN SRV 10  40  5060  sip2.example.com.
_sip._tcp.example.com.       3600 IN SRV 20  100 5060  sip-backup.example.com.
_xmpp-client._tcp            3600 IN SRV 5   0   5222  xmpp.example.com.
_minecraft._tcp.play         300  IN SRV 0   5   25565 mc1.example.com.
# priority: lower = preferred (like MX); weight: load distribution within same priority.
# target: hostname (with A/AAAA), NEVER an IP, NEVER a CNAME.
```

### CAA — Certificate Authority Authorization (RFC 8659, type 257)

```bash
# Format: <name> <ttl> IN CAA <flags> <tag> "<value>"
example.com.        3600  IN  CAA    0  issue       "letsencrypt.org"
example.com.        3600  IN  CAA    0  issuewild   ";"           ; disallow ANY wildcard certs
example.com.        3600  IN  CAA    0  iodef       "mailto:security@example.com"
example.com.        3600  IN  CAA    128 issue      "digicert.com" ; flag bit 128 = critical
# Tags: issue, issuewild, iodef, contactemail, contactphone
# Value ";" with no CA name = "no CA may issue this type"
# CAs MUST check CAA before issuance; absence = any CA may issue.
```

### DS — Delegation Signer (RFC 4034, type 43)

```bash
# Format: <name> <ttl> IN DS <key-tag> <algorithm> <digest-type> <digest>
example.com.        86400 IN  DS     12345  13  2  3a4f...d2c1
# Goes in the PARENT zone; pointers at child's KSK (key-signing key).
# Algorithm: 8=RSASHA256, 13=ECDSAP256SHA256, 14=ECDSAP384SHA384, 15=ED25519, 16=ED448
# Digest type: 1=SHA1 (deprecated), 2=SHA256, 4=SHA384
```

### DNSKEY — DNSSEC Public Key (RFC 4034, type 48)

```bash
# Format: <name> <ttl> IN DNSKEY <flags> <protocol> <algorithm> <public-key>
example.com.        86400 IN  DNSKEY 257  3  13  AwEAAa...b64key  ; KSK (flags=257)
example.com.        86400 IN  DNSKEY 256  3  13  AwEAAa...b64key  ; ZSK (flags=256)
# Flags:
#   bit 7 (256) = ZSK (Zone Signing Key) — signs everything else
#   bit 0 (257) = KSK (Key Signing Key)  — signs only DNSKEY RRset; DS in parent points here
# Protocol: always 3 (DNSSEC)
```

### RRSIG — DNSSEC Signature (RFC 4034, type 46)

```bash
# Auto-generated; one per signed RRset. Format:
# <name> <ttl> IN RRSIG <type-covered> <alg> <labels> <orig-ttl> <expiration> <inception> <key-tag> <signer> <signature>
example.com.  3600 IN RRSIG A 13 2 3600 20240501000000 20240401000000 12345 example.com. AB...sig
# Validity window: between inception and expiration (UTC Z notation).
# CRITICAL: clock skew breaks DNSSEC validation — keep NTP working on resolvers.
```

### NSEC / NSEC3 — Authenticated Denial of Existence (RFC 4034, RFC 5155)

```bash
# NSEC: linked list of next existing name; allows signed proof of non-existence.
example.com.   3600 IN NSEC api.example.com.  A NS SOA MX TXT RRSIG NSEC DNSKEY
# NSEC3: same idea but with hashed names (prevents zone walking).
# Format: <hashname>.<zone> NSEC3 <hash-alg> <flags> <iterations> <salt> <next-hash> <types>
1abc...xyz.example.com. 3600 IN NSEC3 1 0 10 ab12cd34 2def...uvw A RRSIG
# NSEC3PARAM at zone apex tells resolvers how to hash:
example.com. IN NSEC3PARAM 1 0 10 ab12cd34
```

### OPENPGPKEY — In-DNS PGP Key (RFC 7929, type 61)

```bash
# Owner name is SHA256(localpart) truncated to 28 bytes, hex-encoded, then ._openpgpkey.<domain>
# user@example.com  ->  hash + ._openpgpkey.example.com
abc123...def._openpgpkey.example.com. 3600 IN OPENPGPKEY <base64-public-key>
# Requires DNSSEC to be useful (otherwise an attacker can swap keys).
```

### SSHFP — SSH Host Key Fingerprint (RFC 4255, type 44)

```bash
# Format: <hostname> <ttl> IN SSHFP <algorithm> <fp-type> <fingerprint>
host.example.com.   3600  IN  SSHFP  1  1  abc123def456...   ; RSA + SHA1
host.example.com.   3600  IN  SSHFP  1  2  abc123def456...   ; RSA + SHA256
host.example.com.   3600  IN  SSHFP  4  2  abc123def456...   ; ED25519 + SHA256
# Algorithms: 1=RSA, 2=DSA, 3=ECDSA, 4=ED25519
# Fp types:  1=SHA1, 2=SHA256
# Generate from a host:
ssh-keygen -r host.example.com -f /etc/ssh/ssh_host_ed25519_key.pub
# Client must enable verification:
ssh -o VerifyHostKeyDNS=yes user@host.example.com
```

### TLSA — DANE TLS Authentication (RFC 6698, type 52)

```bash
# Format: _<port>._<proto>.<host> <ttl> IN TLSA <usage> <selector> <matching-type> <data>
_443._tcp.example.com.   3600  IN  TLSA  3  1  1   abc123def...sha256
# usage:          0=PKIX-TA (CA), 1=PKIX-EE (end-entity pinned)
#                 2=DANE-TA (private CA), 3=DANE-EE (end-entity, no PKIX)
# selector:       0=full cert, 1=SubjectPublicKeyInfo
# matching-type:  0=exact, 1=SHA256, 2=SHA512
# Generate from a cert:
openssl x509 -in cert.pem -pubkey -noout |
  openssl pkey -pubin -outform DER |
  openssl dgst -sha256 -binary | xxd -p -c 32
```

### LOC — Geographic Location (RFC 1876, type 29)

```bash
# Format: <name> <ttl> IN LOC <lat> <lon> <alt>m <size>m <hp>m <vp>m
example.com.   3600  IN  LOC  37 23 28.000 N 121 58 19.000 W 30m 100m 10m 10m
# Latitude/longitude in degrees minutes seconds, then altitude.
# Mostly historical; rarely used.
```

### SVCB / HTTPS — Service Binding (RFC 9460, types 64 and 65)

```bash
# Format: <name> <ttl> IN SVCB|HTTPS <priority> <target> <key=value ...>
# Priority 0 = AliasMode (like CNAME); >0 = ServiceMode (parameters apply).
example.com.   3600  IN  HTTPS  1  .   alpn="h2,h3" port=443 ipv4hint=93.184.216.34 ipv6hint=2606:2800:220:1::1
api.example.com. 3600 IN SVCB   1  svc.example.net.  alpn="h3" port=8443
# Common SvcParamKeys:
#   alpn=        application-layer protocols (h2, h3, dot, doq)
#   port=        non-default port hint
#   ipv4hint=    A-record hints (skip extra lookup)
#   ipv6hint=    AAAA-record hints
#   ech=         Encrypted ClientHello config (base64)
#   no-default-alpn  disables default ALPN
# Browsers use HTTPS RR to upgrade to HTTP/3 without TCP+ALPN negotiation.
```

### URI — Uniform Resource Identifier (RFC 7553, type 256)

```bash
# Format: <name> <ttl> IN URI <priority> <weight> "<uri>"
_ftp._tcp.example.com.   3600  IN  URI  10  1  "ftp://ftp.example.com/public"
_http._tcp.example.com.  3600  IN  URI  10  1  "https://example.com/"
```

### NAPTR — Naming Authority Pointer (RFC 3403, type 35)

```bash
# Used in ENUM (E.164 phone numbers) and S-NAPTR (service discovery).
# Format: <name> <ttl> IN NAPTR <order> <preference> "<flags>" "<service>" "<regex>" <replacement>
example.com.   3600 IN NAPTR 100 10 "u" "E2U+sip" "!^.*$!sip:info@example.com!" .
example.com.   3600 IN NAPTR 100 10 "S" "SIP+D2T" ""  _sip._tcp.example.com.
# flags: "u"=URI in regex, "s"=replacement is SRV, "a"=replacement is A/AAAA, "p"=protocol-specific
```

### SPF — Sender Policy Framework (deprecated as type 99, RFC 7208)

```bash
# DEPRECATED as a dedicated record type — SPF MUST be published as a TXT record.
# Bad (legacy SPF type 99 — do not use):
example.com.   3600  IN  SPF  "v=spf1 mx -all"
# Good (TXT record — the only correct way today):
example.com.   3600  IN  TXT  "v=spf1 mx ip4:198.51.100.0/24 include:_spf.google.com -all"
```

### Other Record Types You May Encounter

```bash
# HINFO        host info (CPU, OS) — historical, almost always omitted today
# RP           responsible person — administrative contact
# AFSDB        AFS database — legacy
# DNAME        DNS alias (whole subtree) — useful for renumbering, RFC 6672
# CDS / CDNSKEY  child-side DS / DNSKEY for parent automation, RFC 7344
# SMIMEA       S/MIME cert in DNS, RFC 8162
# CSYNC        child-to-parent sync, RFC 7477
# OPT (41)     EDNS0 pseudo-RR — never appears in zone files; on the wire only
# AXFR (252)   zone transfer query (not a stored record)
# IXFR (251)   incremental zone transfer query
# ANY (255)    "give me everything" query — most servers refuse for amplification reasons
```

## The Recursive vs Authoritative Distinction

Two roles, two completely different jobs. Understand which one you are talking to.

```bash
# RECURSIVE RESOLVER — the one your laptop talks to.
# Job: take the user's question and answer it, doing whatever lookups are needed.
# Examples: 8.8.8.8 (Google), 1.1.1.1 (Cloudflare), your ISP's resolver, 9.9.9.9 (Quad9)
# Caches everything for the TTL.

# AUTHORITATIVE NAMESERVER — the one that owns the zone.
# Job: answer questions about names within zones it is authoritative for.
# Examples: ns1.cloudflare.com for cloudflare.com, awsdns-NN.com for AWS-hosted zones
# Returns the "aa" (authoritative answer) flag bit.

# The canonical query:
# 1. Client (laptop) -> Recursive (1.1.1.1)  "What is www.example.com?"
# 2. Recursive -> Root (a.root-servers.net) "What is www.example.com?" -> referral to .com servers
# 3. Recursive -> .com TLD (a.gtld-servers.net) "What is www.example.com?" -> referral to example.com NS
# 4. Recursive -> Authoritative (ns1.example.com) "What is www.example.com?" -> answer
# 5. Recursive -> Client  "www.example.com is 93.184.216.34"
```

```bash
# Watch the canonical chain happen
dig +trace www.example.com
# You'll see four sections: roots -> .com NS -> example.com NS -> answer
```

```bash
# Tell which kind of server you are talking to
dig @8.8.8.8 example.com +norecurse
# If status NOERROR with full answer: cached recursive OR authoritative
# If status NOERROR with referral (NS in authority section, no answer): authoritative only
# If status REFUSED with RD bit but no RA: recursive that won't recurse for you (ACL)
dig @ns1.example.com example.com
# Look at flags: "qr aa rd" — "aa" = authoritative answer
```

```bash
# Your laptop NEVER talks to the root servers directly — that is the recursive's job.
# Your laptop talks to ONE server (the recursive in /etc/resolv.conf).
# It does not walk the delegation chain. It does not understand DNSSEC validation
# unless you run a local validating resolver.
```

## The Root Servers

Thirteen named root servers, A through M, operate the top of the DNS tree. Each is anycast across many physical locations worldwide for resilience and latency. The list is public and rarely changes.

```bash
# The 13 named roots (each is anycast — hundreds of physical instances globally)
a.root-servers.net   198.41.0.4         2001:503:ba3e::2:30  ; Verisign
b.root-servers.net   170.247.170.2      2801:1b8:10::b       ; USC ISI
c.root-servers.net   192.33.4.12        2001:500:2::c        ; Cogent
d.root-servers.net   199.7.91.13        2001:500:2d::d       ; University of Maryland
e.root-servers.net   192.203.230.10     2001:500:a8::e       ; NASA Ames
f.root-servers.net   192.5.5.241        2001:500:2f::f       ; ISC
g.root-servers.net   192.112.36.4       2001:500:12::d0d     ; DISA
h.root-servers.net   198.97.190.53      2001:500:1::53       ; ARL
i.root-servers.net   192.36.148.17      2001:7fe::53         ; Netnod
j.root-servers.net   192.58.128.30      2001:503:c27::2:30   ; Verisign
k.root-servers.net   193.0.14.129       2001:7fd::1          ; RIPE NCC
l.root-servers.net   199.7.83.42        2001:500:9f::42      ; ICANN
m.root-servers.net   202.12.27.33       2001:dc3::35         ; WIDE Project
```

```bash
# The IANA-managed root zone — the source of truth for TLD delegation
# Resolvers ship with a root.hints / named.cache file:
#   /etc/bind/db.root            (BIND)
#   /etc/unbound/root.hints      (Unbound, sometimes /var/lib/unbound/)
# Refresh occasionally:
curl -fsSL https://www.internic.net/domain/named.cache -o root.hints
unbound-anchor -a /var/lib/unbound/root.key   ; refresh DNSSEC root trust anchor
```

```bash
# Query a root directly
dig @a.root-servers.net . NS
dig @a.root-servers.net example.com  ; you'll get a referral, never an answer
```

## Glue Records and Delegation

When the authoritative NS for example.com is itself ns1.example.com, the parent zone (.com) cannot just say "ask ns1.example.com" without also saying where ns1.example.com is — otherwise the resolver would have to ask example.com to find example.com's nameserver. Glue solves this chicken-and-egg.

```bash
# In-bailiwick delegation needs glue:
# At the .com authoritative server:
example.com.        IN  NS    ns1.example.com.        ; delegation
example.com.        IN  NS    ns2.example.com.        ; delegation
ns1.example.com.    IN  A     198.51.100.10           ; GLUE — required
ns2.example.com.    IN  A     198.51.100.11           ; GLUE — required
ns1.example.com.    IN  AAAA  2001:db8::10            ; GLUE — required
```

```bash
# Out-of-bailiwick delegation needs NO glue (the NS is in another zone):
example.com.        IN  NS    ns1.dnsprovider.net.    ; resolver can look this up separately
# Your registrar interface usually warns you about glue when the NS is in-bailiwick.
```

```bash
# Check glue with dig
dig @a.gtld-servers.net example.com NS
# Look in the ADDITIONAL section for A/AAAA of the nameservers — that's the glue.
```

```bash
# Broken: in-bailiwick NS without glue
example.com.        IN  NS    ns1.example.com.
;; missing A record for ns1.example.com at parent
;; -> resolver gets stuck: needs example.com to find ns1.example.com to find example.com

# Fixed: provide glue at the registrar
# Most registrars expose a "Host Records" / "Glue Records" panel for in-bailiwick NS.
```

## Anycast

Anycast advertises the same IP from many physical locations via BGP. Routing protocols steer each client to the topologically nearest instance. The same `1.1.1.1` you query from London answers from a London POP; from Tokyo it answers from a Tokyo POP.

```bash
# Discover which POP is answering you for an anycast service
dig @1.1.1.1 -c CH -t TXT id.server          ; Cloudflare reveals the POP code
dig @8.8.8.8 -c CH -t TXT hostname.bind      ; Google's POP hostname
dig @9.9.9.9 -c CH -t TXT id.server          ; Quad9
```

```bash
# Compare with traceroute — first hop into the anycast network differs by region
traceroute -n 1.1.1.1
# The same IP, different path, different POP.
```

```bash
# Unicast contrast: one IP, one location, every client traverses the global internet to reach it.
# Trade-offs:
#   anycast  pro: low latency, DDoS resilience, automatic failover; con: harder to debug.
#   unicast  pro: predictable, easy to debug; con: single point of failure, latency varies.
```

## EDNS0 Extension

The classic 12-byte DNS header has no room for new flags or large payloads. EDNS0 (RFC 6891) introduced an OPT pseudo-RR carried in the additional section that negotiates extensions during the query. Without EDNS0, DNSSEC, DoH, large responses, and modern record types like SVCB don't work.

```bash
# Anatomy of the OPT pseudo-RR
# Owner: "."  Type: OPT (41)  Class: requested UDP buffer size (e.g. 4096)
# TTL field reused as: extended-rcode(8) | version(8) | DO(1) | Z(15)
# RDATA: option-code | option-length | option-data...

dig +bufsize=4096 +dnssec example.com         ; advertise 4096-byte buffer, set DO bit
dig +nocrypto +dnssec example.com             ; show OPT in human-readable form

# In dig output, look for:
;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags: do; udp: 4096
; COOKIE: ab12cd34...                         ; DNS cookies (RFC 7873)
```

```bash
# Common EDNS0 options
# DO bit          DNSSEC OK — client wants RRSIG/DNSKEY in response
# Client Subnet   ECS — resolver tells authoritative the client subnet (RFC 7871)
#                 Privacy-controversial; many resolvers strip or clamp this.
# Cookies         per-RFC-7873; helps mitigate spoofing
# Padding         pad query to a fixed block size for DoT/DoH privacy

# DO bit is required for DNSSEC — without it, RRSIG records are never returned
dig +noedns example.com                       ; query WITHOUT EDNS0 — no DNSSEC possible
```

```bash
# Broken: server doesn't support EDNS0 at all (rare today)
;; reply from unexpected source / FORMERR / connection reset
# Fixed: server upgrade or downgrade with +noedns flag (testing only)
```

## Caching and TTLs

Every record carries a TTL in seconds. Resolvers and clients cache the answer for that long. TTL design balances propagation speed against load on authoritative servers.

```bash
# TTL choice cheat sheet
# 60-300       low TTL — useful during migrations, GSLB, ephemeral records
# 3600         1 hour — common default for A/AAAA in stable zones
# 14400        4 hours — load-balanced services
# 86400        1 day — NS, SOA, CAA, infrequently changing records
# 604800       1 week — almost never appropriate for user-facing records
# 0            do-not-cache — useful for testing, harmful in production
```

```bash
# Watch a cached TTL count down
dig @1.1.1.1 example.com | grep -E '^example\.com'
# example.com.   3491   IN  A   93.184.216.34
# Run again 60s later: TTL 3431 — same cache, decrementing.
```

```bash
# Negative caching — NXDOMAIN/NODATA cached for SOA's MINIMUM field, capped by RFC 2308 at 86400s
dig @1.1.1.1 nonexistent.example.com         ; first time — query goes to authoritative
dig @1.1.1.1 nonexistent.example.com         ; second time — answered from negative cache
```

```bash
# Force-bypass cache: query authoritative directly
dig @ns1.example.com example.com +norecurse
dig @ns1.example.com example.com +retry=0    ; no UDP retry on timeout
```

```bash
# Flush local caches
sudo systemctl restart systemd-resolved      ; or: sudo resolvectl flush-caches
sudo rndc flush                              ; BIND
sudo rndc flushname example.com              ; BIND, single name
sudo unbound-control flush_zone example.com  ; Unbound
sudo killall -HUP mDNSResponder              ; macOS classic
sudo dscacheutil -flushcache                 ; macOS modern
ipconfig /flushdns                            ; Windows
```

```bash
# TTL-zero gotcha: not "instant", just "don't cache". Some resolvers ignore TTL=0
# and clamp to a minimum (e.g. 30s) to protect against bad operators.
```

## Zone File Syntax — BIND Format

BIND-style zone files are the canonical, human-readable format. RFC 1035 defines the master file format, and most authoritative servers (BIND, NSD, Knot) read it natively.

```bash
$TTL 3600                                       ; default TTL for records that omit it
$ORIGIN example.com.                            ; default suffix for unqualified names
$INCLUDE /etc/bind/zones/example.com.subzone.db ; pull in another file inline

@   IN  SOA  ns1.example.com. hostmaster.example.com. (
              2024010101    ; serial — YYYYMMDDNN convention; MUST increase
              7200          ; refresh
              3600          ; retry
              1209600       ; expire
              3600 )        ; minimum (negative-cache TTL)

@                IN  NS     ns1.example.com.
@                IN  NS     ns2.example.com.

@                IN  A      93.184.216.34
@                IN  AAAA   2606:2800:220:1::1
@                IN  MX     10  mail.example.com.
@                IN  MX     20  mail-backup.example.net.
@                IN  TXT    "v=spf1 mx -all"
@                IN  CAA    0   issue   "letsencrypt.org"

ns1              IN  A      198.51.100.10
ns2              IN  A      198.51.100.11
mail             IN  A      198.51.100.20
www              IN  CNAME  example.com.
api              IN  A      203.0.113.10
api              IN  A      203.0.113.11
api              IN  AAAA   2001:db8::10
*.dev            IN  A      203.0.113.99      ; wildcard

_dmarc           IN  TXT    "v=DMARC1; p=quarantine; rua=mailto:dmarc@example.com"
selector1._domainkey IN TXT "v=DKIM1; k=rsa; p=MIGfMA0..."
_sip._tcp        IN  SRV    10  60  5060  sip.example.com.
_443._tcp.www    IN  TLSA   3   1   1   abc...sha256
```

```bash
# Directives summary
$TTL <seconds>           ; default TTL going forward in the file
$ORIGIN <fqdn>           ; default suffix for unqualified names
$INCLUDE <file> [origin] ; inline-include another file (optional new origin)
$GENERATE ...            ; BIND-only: bulk-generate sequential records

# Special characters
@                        ; equals current $ORIGIN
.                        ; trailing dot = absolute name (no $ORIGIN appended)
;                        ; comment to end of line
( ... )                  ; line continuation across newlines
\<char>                  ; escape special characters in TXT or names
\NNN                     ; octal escape for byte values
"<string>"               ; quoted string in TXT (max 255 bytes per piece)
```

```bash
# Class field — almost always IN (Internet); HS, CH exist for legacy/diagnostics
@   IN  A   1.2.3.4
@   CH  TXT "version.bind"     ; CH (Chaos) class for server identity probes
```

```bash
# Validation tools
named-checkzone example.com /etc/bind/zones/example.com.db
named-checkconf /etc/bind/named.conf
nsd-checkzone example.com /etc/nsd/zones/example.com.zone
kzonecheck /etc/knot/zones/example.com.zone
```

## SOA Record — All Fields

Every zone has exactly one SOA at the apex. The SOA controls zone-transfer behavior between primary and secondary nameservers and the negative-cache TTL.

```bash
# Field by field
@  IN  SOA  ns1.example.com.   hostmaster.example.com.  (
            2024010101         ; SERIAL
            7200               ; REFRESH
            3600               ; RETRY
            1209600            ; EXPIRE
            3600 )             ; MINIMUM
```

```bash
# MNAME — primary master nameserver. Used by DNS NOTIFY (RFC 1996) and some tooling.
#         If you have a hidden master, this is where you put it.
# RNAME — admin email; first '.' replaced with '@'
#         hostmaster.example.com  ->  hostmaster@example.com
#         IETF convention: hostmaster@ for DNS issues; RFC 2142.
# SERIAL — 32-bit unsigned int; secondaries refetch zone when primary's serial > theirs.
#         Two common conventions:
#           YYYYMMDDNN  (2024010101 = first edit on 2024-01-01)
#           plain incrementing integer (1, 2, 3, ...)
# REFRESH — how often secondaries poll primary's SOA serial. Typical: 1-4 hours.
#         If primary supports NOTIFY, this is mostly a fallback.
# RETRY — wait period before retrying a failed REFRESH. Smaller than REFRESH.
# EXPIRE — secondary stops serving the zone after EXPIRE seconds without contact.
#         Common: 1-2 weeks. Too short = zone disappears during outages.
# MINIMUM — negative-caching TTL. RFC 2308 caps it at 24 hours regardless of value.
```

```bash
# Bumping the serial — the canonical ritual
# 1. Edit zone file
# 2. Update serial: 2024010101 -> 2024010102 (same day, second change)
# 3. Reload zone: rndc reload example.com
# 4. Watch secondaries pick up new serial:
#    dig SOA example.com @ns2.example.com +short
# 5. If serial doesn't roll forward, secondaries return stale data forever.
```

```bash
# Broken: forgot to bump serial
# Symptom: dig +short A example.com  returns OLD answer from secondary
# Diagnosis:
dig SOA example.com @ns1.example.com +short
dig SOA example.com @ns2.example.com +short    ; both should match newer value

# Fixed: increment serial, rndc reload, then send NOTIFY:
sudo rndc notify example.com
```

## Common Server Implementations

```bash
# BIND9 (named)
# Repo: gitlab.isc.org/isc-projects/bind9
# Role: authoritative + recursive (compile-time choice/runtime config)
# Config: /etc/named.conf or /etc/bind/named.conf
# Strengths: reference implementation, every feature, every bug
# Weaknesses: large attack surface, many features means many footguns
named -v
named -g -c /etc/named.conf                   ; foreground for debugging
rndc status
```

```bash
# PowerDNS Authoritative
# Role: authoritative; database backends (MySQL/Postgres/sqlite/LDAP/...)
# Config: /etc/powerdns/pdns.conf
# Strengths: API-driven, easy to script with, multi-backend
# Weaknesses: complex config, separate from PDNS Recursor
pdns_server --version
pdnsutil list-zone example.com
pdnsutil edit-zone example.com
pdns_control reload
```

```bash
# Knot DNS
# Role: authoritative-only; native DNSSEC signing
# Config: /etc/knot/knot.conf
# Strengths: very fast, modern code, online-signing support
knotc reload
knotc zone-status example.com
kdig @1.1.1.1 example.com                     ; knot-dnsutils
```

```bash
# NSD (Name Server Daemon)
# Role: authoritative-only; minimal feature set
# Config: /etc/nsd/nsd.conf
# Strengths: small, secure, fast; designed for root and TLD operators
# Weaknesses: no recursion, no DNSSEC online signing
nsd-control reload
nsd-checkzone example.com /etc/nsd/zones/example.com.zone
```

```bash
# dnsmasq
# Role: forwarder + DHCP + TFTP; perfect for home routers and labs
# Config: /etc/dnsmasq.conf
# Strengths: tiny, integrated DHCP, easy
# Weaknesses: not a real authoritative; no DNSSEC signing
dnsmasq --version
sudo systemctl restart dnsmasq
```

```bash
# CoreDNS
# Role: plugin-based; the default DNS in Kubernetes
# Config: Corefile
# Strengths: composable, Go, easy custom plugins
# Weaknesses: less battle-tested than BIND/NSD on the public internet
coredns -conf /etc/coredns/Corefile
kubectl -n kube-system get configmap coredns -o yaml
```

```bash
# Unbound
# Role: recursive-only validating resolver
# Config: /etc/unbound/unbound.conf
# Strengths: clean code, fast, DNSSEC default-on, modern
# Weaknesses: not authoritative
unbound-control reload
unbound-control flush_zone example.com
unbound-checkconf
```

```bash
# systemd-resolved
# Role: local stub resolver, caching, optional DoT
# Listens on 127.0.0.53:53; /etc/resolv.conf is usually a symlink to its stub file
resolvectl status
resolvectl query example.com
resolvectl statistics
```

## BIND Configuration

```bash
// /etc/named.conf — minimum viable authoritative server
options {
    directory "/var/named";
    listen-on    port 53 { any; };
    listen-on-v6 port 53 { any; };
    allow-query  { any; };
    recursion no;                              ; authoritative-only
    dnssec-validation auto;                    ; validate when recursing (no-op here)
    version "Tom Servo";                       ; hide real version from CH TXT version.bind
};

logging {
    channel default_log {
        file "/var/log/named/named.log" versions 3 size 10m;
        severity info;
        print-time yes;
        print-category yes;
    };
    category default { default_log; };
    category queries { default_log; };
};

zone "example.com" {
    type primary;                              ; (or "master"; "primary" preferred since 9.18)
    file "zones/example.com.db";
    allow-transfer { 198.51.100.11; };         ; only secondary may AXFR
    also-notify { 198.51.100.11; };            ; push NOTIFY to secondary on change
};

zone "." IN {
    type hint;
    file "named.ca";
};
```

```bash
// Recursive resolver pattern — DO NOT expose to internet
options {
    directory "/var/named";
    listen-on { 127.0.0.1; 10.0.0.1; };
    recursion yes;
    allow-recursion { 127.0.0.1; 10.0.0.0/8; }; ; never "any" on a recursive
    dnssec-validation auto;
    minimal-responses yes;                      ; smaller responses, less amplification
    rate-limit { responses-per-second 10; };    ; RRL for DDoS hardening
};
```

```bash
# rndc — runtime control
sudo rndc-confgen -a                          ; generate keys if not present
sudo rndc status
sudo rndc reload                              ; reload everything
sudo rndc reload example.com                  ; reload one zone
sudo rndc flush                               ; flush ALL caches
sudo rndc flushname example.com               ; flush single name
sudo rndc notify example.com                  ; send NOTIFY to secondaries
sudo rndc reconfig                            ; reread config without dropping caches
sudo rndc dumpdb -cache                       ; dump cache to named_dump.db
sudo rndc stats                               ; emit named.stats
sudo rndc trace 5                             ; raise debug level to 5
sudo rndc trace 0                             ; back to none
```

## Reverse DNS — PTR Records

Reverse DNS turns an IP back into a name. Every IP block has an authoritative reverse zone under `in-addr.arpa` (IPv4) or `ip6.arpa` (IPv6). The block owner runs that zone — for cloud IPs, that's your provider; for your own /24, you do.

```bash
# IPv4 reverse — reverse the octets, append in-addr.arpa
# Address: 93.184.216.34
# PTR owner: 34.216.184.93.in-addr.arpa.

$ORIGIN 216.184.93.in-addr.arpa.
@   IN  SOA  ns1.example.com. hostmaster.example.com. (
              2024010101 7200 3600 1209600 3600 )
@   IN  NS   ns1.example.com.
@   IN  NS   ns2.example.com.
34  IN  PTR  example.com.
35  IN  PTR  www.example.com.
```

```bash
# IPv6 reverse — every nibble (4 bits) reversed, joined with dots, appended with ip6.arpa
# Address: 2001:db8::1  (full form: 2001:0db8:0000:0000:0000:0000:0000:0001)
# PTR owner: 1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.

$ORIGIN 8.b.d.0.1.0.0.2.ip6.arpa.
1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0  IN  PTR  host.example.com.
```

```bash
# Forward-and-reverse match rule — mail server reputation requires it
# 1. Forward: mail.example.com  IN  A  198.51.100.20
# 2. Reverse: 20.100.51.198.in-addr.arpa  IN  PTR  mail.example.com.
# Spam filters (especially Microsoft's) reject mail when these don't match.

# Verify alignment
dig +short MX example.com
dig +short A mail.example.com
dig +short -x 198.51.100.20            ; -x flips IP for reverse query
```

```bash
# Cloud reality — for AWS/GCP/Azure IPs:
#   AWS: aws ec2 modify-instance-attribute --instance-id i-... --no-source-dest-check
#        Then create reverse via the EC2 console or via Route 53 elastic-IP attribute.
#   GCP: gcloud compute instances set-disk-auto-delete (PTR via reverse zone)
#   Azure: az network public-ip update --reverse-fqdn host.example.com.
# You cannot self-host PTRs for IPs you do not own.
```

## DNSSEC — The Chain of Trust

DNSSEC adds cryptographic signatures over RRsets so resolvers can detect tampering. Trust flows top-down: each parent zone signs a DS record committing to the child's KSK; the child signs everything else with its ZSK. The root zone's KSK is the universal trust anchor, distributed in software and signed annually by ICANN.

```bash
# The chain in pictures (top to bottom)
# .              KSK signs DNSKEY for .                     (trust anchor: root.key)
# .              ZSK signs DS for .com                       (delegation to .com)
# .com           KSK signs DNSKEY for .com
# .com           ZSK signs DS for example.com
# example.com    KSK signs DNSKEY for example.com
# example.com    ZSK signs A, AAAA, MX, ... for example.com  (the actual answers)
```

```bash
# Walk the chain manually
dig +dnssec . DNSKEY   @a.root-servers.net    ; root keys
dig +dnssec com DS     @a.root-servers.net    ; root signs .com DS
dig +dnssec com DNSKEY @a.gtld-servers.net    ; .com keys
dig +dnssec example.com DS     @a.gtld-servers.net   ; .com signs example.com DS
dig +dnssec example.com DNSKEY @ns1.example.com      ; child keys
dig +dnssec example.com A      @ns1.example.com      ; signed answer
```

```bash
# Validate end-to-end with delv (BIND's validating dig replacement)
delv +rtrace example.com                      ; show every signature checked
delv +vtrace example.com                      ; even more verbose
# Output:
;   ; fully validated
;   example.com.    3600 IN A  93.184.216.34
;   example.com.    3600 IN RRSIG A 13 2 3600 ...
```

```bash
# Inspect the AD bit (Authenticated Data) on resolver responses
dig +dnssec example.com | grep -E 'flags|status'
# ;; flags: qr rd ra ad; QUERY: 1, ANSWER: 1, ...    <-- "ad" = validated by resolver
# If you see "ad" in flags: resolver successfully validated the chain.
# If status is SERVFAIL on a DNSSEC-signed zone: validation failure (broken chain or bad sigs).
```

```bash
# The canonical "DNSSEC enabled but broken" symptoms
# 1. DS at parent doesn't match any current DNSKEY at child  ->  SERVFAIL everywhere
# 2. RRSIG expired (clock skew, rotation failure)            ->  SERVFAIL on signed RRsets
# 3. NSEC3 iteration count too high                          ->  insecure (RFC 9276)
# 4. Algorithm in DS not supported by parent                 ->  insecure delegation
# 5. Mixed algorithm sets without complete coverage          ->  validation fails
```

## DNSSEC — Zone Signing

```bash
# Generate a Key Signing Key (KSK) — flag 257
dnssec-keygen -a ECDSAP256SHA256 -n ZONE -f KSK example.com
# Generate a Zone Signing Key (ZSK) — flag 256
dnssec-keygen -a ECDSAP256SHA256 -n ZONE     example.com
# Output: Kexample.com.+013+12345.key  and  Kexample.com.+013+12345.private
```

```bash
# Sign the zone manually (BIND old-school workflow)
dnssec-signzone -A -3 $(head -c 16 /dev/urandom | xxd -p) \
                -N INCREMENT -o example.com -t example.com.zone
# -A      adds DNSKEYs from current keys directory
# -3 SALT enables NSEC3 with given salt
# -N INCREMENT bumps SOA serial automatically
# -o ORIGIN sets the zone origin
# Output: example.com.zone.signed
```

```bash
# Modern BIND inline-signing (no manual signzone runs)
zone "example.com" {
    type primary;
    file "zones/example.com.db";
    inline-signing yes;
    auto-dnssec maintain;                    ; BIND rotates ZSKs automatically
    key-directory "/var/named/keys/example.com";
};
```

```bash
# Knot DNS automatic signing
zone:
  - domain: example.com
    storage: /var/lib/knot/zones
    file: example.com.zone
    dnssec-signing: on
    dnssec-policy: default
```

```bash
# KSK rotation pain — DS at parent must update
# 1. Generate new KSK (mark inactive)
# 2. Publish new DNSKEY in zone (still using old KSK to sign DNSKEY RRset, plus the new one)
# 3. Wait at least 2x parent's DNSKEY TTL
# 4. Update DS at parent (registrar interface)
# 5. Wait at least 2x parent DS TTL
# 6. Roll signing to new KSK
# 7. Remove old KSK from DNSKEY RRset

# CDS / CDNSKEY automation (RFC 7344) lets the child publish its own DS update
# Parent automatically picks up CDS records from child if parent supports it (e.g. .ch, .se).
```

```bash
# ZSK rotation can be fully automatic — no parent involvement
# BIND auto-dnssec maintain handles double-signing windows automatically.
# Rotation cadence: ZSK monthly/quarterly, KSK yearly+.
```

## DNSSEC — Validation

```bash
# Inspect a zone's DNSSEC posture
dig +dnssec example.com SOA                   ; zone signed?
dig +dnssec example.com DNSKEY                ; KSK + ZSK present?
dig DS example.com @parent-ns                 ; DS at parent?
dig +cd +dnssec example.com                   ; CD bit = checking disabled (skip validation)

# Compare validating vs non-validating
dig +dnssec example.com                        ; AD bit set if valid
dig +cd +dnssec example.com                    ; AD never set when CD=1
```

```bash
# Common validation failures and what they mean
# SERVFAIL with "extended RCODE 6 (DNSKEY missing)"     ; child has DS at parent but no DNSKEY
# SERVFAIL with "extended RCODE 7 (RRSIGs missing)"     ; zone unsigned but DS present
# SERVFAIL with "extended RCODE 8 (RRSIG not signed)"   ; signature problem
# SERVFAIL with "extended RCODE 9 (no ZSK)"             ; missing key
# Use `+dnssec +nocrypto` and `delv` to walk the chain.

# Check extended DNS errors (RFC 8914) with dig
dig +dnssec example.com
;; OPT PSEUDOSECTION:
; EDE: 6 (DNSKEY Missing)
```

```bash
# Trust anchors — the root key
# /etc/unbound/root.key  or  /etc/bind/bind.keys  or  /var/lib/unbound/root.key
unbound-anchor -v -a /var/lib/unbound/root.key
# RFC 5011 lets resolvers automatically pick up new root KSKs as ICANN rotates them.
# Manual fetch (rare):
curl -fsSL https://data.iana.org/root-anchors/root-anchors.xml
```

```bash
# DNSSEC trust-anchor file format (BIND)
trust-anchors {
  "." initial-key 257 3 8 "AwEAAagAIK...base64...";
};
managed-keys {
  "." initial-key 257 3 8 "AwEAAagAIK...base64...";
};
```

## DoT / DoH / DoQ

Encrypted DNS variants protect query content (and patterns) from passive eavesdroppers and on-path tampering. They do NOT hide the destination IP of the resolver.

```bash
# DoT — DNS-over-TLS, RFC 7858, port 853
# stubby (Linux) — local DoT resolver listening on 127.0.0.1:53
# /etc/stubby/stubby.yml
upstream_recursive_servers:
  - address_data: 1.1.1.1
    tls_auth_name: "cloudflare-dns.com"
  - address_data: 9.9.9.9
    tls_auth_name: "dns.quad9.net"
sudo systemctl enable --now stubby
sudo dig @127.0.0.1 example.com
```

```bash
# systemd-resolved with DoT
# /etc/systemd/resolved.conf
[Resolve]
DNS=1.1.1.1#cloudflare-dns.com 9.9.9.9#dns.quad9.net
DNSOverTLS=yes
DNSSEC=allow-downgrade
Cache=yes
sudo systemctl restart systemd-resolved
resolvectl status                             ; verify DNSOverTLS=yes
```

```bash
# DoH — DNS-over-HTTPS, RFC 8484, port 443
# Browsers (Firefox, Chrome) ship with built-in DoH resolvers.
# Firefox: about:config -> network.trr.mode = 2 (DoH with system fallback)
# Chrome:  Settings -> Security -> Use secure DNS

# Local DoH proxy: cloudflared, dnsproxy, dnscrypt-proxy
cloudflared proxy-dns --address 127.0.0.1 --port 5053 --upstream https://1.1.1.1/dns-query
dig @127.0.0.1 -p 5053 example.com
```

```bash
# DoQ — DNS-over-QUIC, RFC 9250, port 853
# Same UDP port as DoT; multiplex over QUIC streams; 0-RTT for repeat queries.
# Knot dnsproxy with QUIC:
dnsproxy -u quic://1.1.1.1:853 -l 0.0.0.0 -p 5053
# Test with kdig:
kdig @1.1.1.1 +quic example.com
```

```bash
# Privacy / censorship trade-off
# DoT/DoQ — single port (853), easy to block at network layer
# DoH    — port 443 looks like HTTPS, hard to block without breaking the web
# Pick based on threat model.
```

## Dynamic DNS — RFC 2136

Dynamic DNS lets clients update zone records via authenticated UPDATE messages. Active Directory uses it; consumer dyn-DNS (no-ip, dyndns) is the public-internet flavor; ACME's DNS-01 challenge often uses it for cert issuance.

```bash
# nsupdate — the canonical client
# Generate a TSIG key
tsig-keygen -a hmac-sha256 update-key. > /etc/bind/keys/update-key.conf
# Or older syntax:
dnssec-keygen -a HMAC-SHA256 -b 256 -n HOST update-key

# In named.conf — accept updates with that key
key "update-key" {
    algorithm hmac-sha256;
    secret "base64secret==";
};
zone "example.com" {
    type primary;
    file "zones/example.com.db";
    update-policy { grant update-key. zonesub ANY; };
};

# Run nsupdate
nsupdate -k /etc/bind/keys/update-key.conf <<'EOF'
server ns1.example.com 53
zone example.com.
update delete dynamic.example.com. A
update add    dynamic.example.com. 60 A 198.51.100.99
send
EOF
```

```bash
# GSS-TSIG — Active Directory's flavor (Kerberos auth instead of pre-shared key)
# Used by Windows DHCP servers to register clients automatically:
# DHCP -> client gets address -> server registers A + PTR via GSS-TSIG -> AD-integrated DNS

# Test GSS-TSIG from Linux
kinit Administrator@EXAMPLE.COM
nsupdate -g
> server dc1.example.com
> update add host.example.com 300 A 10.0.0.50
> send
```

```bash
# Consumer dynamic DNS (no-ip.com, dyndns.org, duckdns.org, FreeDNS)
# Typically a simple HTTP API:
curl -s "https://www.duckdns.org/update?domains=mybox&token=TOKEN&ip="
# Or via ddclient daemon updating from cron.
```

## Service Discovery via DNS-SD + mDNS

Multicast DNS (RFC 6762) and DNS-SD (RFC 6763) let devices publish and discover services on the local link without any central server. The `.local` TLD is reserved; queries go to multicast 224.0.0.251 (IPv4) or ff02::fb (IPv6) on UDP port 5353.

```bash
# Browse mDNS services on macOS (Bonjour built-in)
dns-sd -B _http._tcp.                          ; list HTTP servers on the link
dns-sd -B _services._dns-sd._udp.              ; list ALL service types
dns-sd -L "MyHost" _ssh._tcp.                  ; resolve a specific instance
dns-sd -G v4v6 some-host.local                 ; resolve a name to A/AAAA

# Linux — Avahi
avahi-browse -a                                ; browse all services
avahi-browse -r _ssh._tcp                      ; resolve SSH instances
avahi-resolve --name some-host.local
avahi-publish-service "MyShare" _http._tcp 8080 path=/

# Query a .local name directly
ping some-host.local                           ; if mDNS responder is running
getent hosts some-host.local                   ; respects /etc/nsswitch.conf
```

```bash
# DNS-SD service-instance pattern
# <instance>._<service>._<proto>.<domain>
# Example: "Bob's Printer._ipp._tcp.local"
# PTR record:  _ipp._tcp.local.                 PTR  Bob's\032Printer._ipp._tcp.local.
# SRV record:  Bob's\032Printer._ipp._tcp.local. SRV 0 0 631 bobs-pc.local.
# TXT record:  Bob's\032Printer._ipp._tcp.local. TXT "txtvers=1" "rp=printers/bob"
```

```bash
# /etc/nsswitch.conf — common Linux entry that consults mDNS
hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4
# files            -> /etc/hosts
# mdns4_minimal    -> mDNS only for .local names
# [NOTFOUND=return] -> stop on NOTFOUND for .local
# dns              -> normal /etc/resolv.conf path
# mdns4            -> mDNS for everything else (rarely)
```

## SVCB and HTTPS Records

The new SVCB (RFC 9460) and its HTTPS-typed cousin are the modern replacement for HTTP `Alt-Svc`, with much stronger discovery semantics: ALPN advertisement, port hints, address hints, and Encrypted ClientHello (ECH) keys all carried in DNS, signed by DNSSEC.

```bash
# AliasMode (priority 0) — like CNAME but works at zone apex
example.com.   3600 IN HTTPS 0 svc.example.net.

# ServiceMode (priority > 0) — parameters apply
example.com.   3600 IN HTTPS 1 . alpn="h2,h3" port=443 ipv4hint=93.184.216.34 ipv6hint=2606:2800:220:1::1
api.example.com. 3600 IN SVCB 1 svc.example.net. alpn="h3" port=8443

# Encrypted ClientHello config in HTTPS RR
example.com.   60   IN HTTPS 1 . alpn="h3" ech="AEX+DQBBhwAg..."
```

```bash
# Why HTTPS RR matters for HTTP/3
# Without it: client must do TCP+TLS+ALPN to discover h3, then re-connect over QUIC.
# With it: client knows h3 endpoint up front; one round-trip saved on first request.
# Also reveals whether ECH is supported, lets browsers fail-closed if downgraded.

dig HTTPS example.com                          ; modern dig prints HTTPS RR
dig +dnssec HTTPS cloudflare.com               ; signed HTTPS RR with ECH
```

```bash
# HTTPS RR replaces:
# - HTTP Alt-Svc (Header-based, requires connection first)
# - SRV records for HTTP (which RFC 2782 explicitly excludes)
# - Multiple custom mechanisms (HSTS preload, etc.)
```

## Cache Poisoning and DNS Security Issues

```bash
# Kaminsky attack (2008) — predict the resolver's outbound query and race it with forged responses
# Original cause: low entropy in query ID (16 bits) + fixed outbound source port
# Mitigation: source-port randomization (16 more bits of entropy) — total ~32 bits
# Mitigation: 0x20 query-name randomization — mix uppercase/lowercase in query names
# Mitigation: DNSSEC validation — cryptographic signatures defeat blind spoofing entirely

# Check your resolver's source-port randomness
dig +short porttest.dns-oarc.net TXT
# "204.176.110.10 is GOOD: 26 queries in 19.6 seconds from 26 ports with std dev 21527"
# GOOD = high port entropy; POOR = predictable

# Check 0x20 mixed-case enforcement
dig EXample.coM @1.1.1.1                       ; resolver should preserve case in echo
```

```bash
# DNS rebinding — attack via short-TTL records flipping between public and RFC1918
# Attacker controls evil.com, returns A=1.2.3.4 (TTL=1) then A=192.168.1.1 on second query
# Browser (same-origin policy) thinks it's still talking to evil.com
# Mitigations:
#   - Resolvers / browsers refuse RFC1918 answers from public domains (dnsmasq stop-dns-rebind)
#   - Application: Validate Host header and SSRF protection
unbound conf:
  private-address: 10.0.0.0/8
  private-address: 172.16.0.0/12
  private-address: 192.168.0.0/16
  private-domain: "trusted-internal.example."  ; allow-list for legitimate internal zones
```

```bash
# Amplification attacks — small UDP query, large UDP response, spoofed source IP
# Mitigations:
#   - Disable open recursion (allow-recursion { localnets; }; in BIND)
#   - Response Rate Limiting (RRL) on authoritative
#   - Refuse ANY queries (use minimal-responses or refuse-any in newer BIND)
#   - BCP38 ingress filtering at network layer (defeats spoofing)

# Check if a server is an open resolver
dig @<ip> isc.org +short
# If you get an answer and you're not on its allow-list, it's open. Tell the operator.
```

```bash
# DNS exfiltration — encode data in subdomain queries to attacker-controlled domain
# Mitigations:
#   - Egress DNS filtering (block direct outbound 53 from clients)
#   - Force all DNS through inspecting recursive
#   - Look for high-entropy / long subdomains in DNS logs (Pi-hole, NXLog, Splunk)
```

## Common DNS Records for Email

The email-authentication trio: SPF says who can send; DKIM proves no tampering; DMARC tells receivers what to do when SPF/DKIM fail. All three live in DNS as TXT records.

```bash
# MX record (mandatory for receiving mail)
example.com.       3600  IN  MX  10  mail.example.com.
example.com.       3600  IN  MX  20  mail-backup.example.net.
mail.example.com.  3600  IN  A   198.51.100.20

# SPF — Sender Policy Framework, RFC 7208
example.com.       3600  IN  TXT "v=spf1 mx include:_spf.google.com ip4:198.51.100.0/24 ~all"
# Mechanisms:
#   v=spf1            version
#   mx                allow MX hosts
#   include:DOMAIN    include another domain's SPF
#   ip4:CIDR          allow IPv4 range
#   ip6:CIDR          allow IPv6 range
#   a                 allow A records of the domain
#   exists:DOMAIN     allow if DOMAIN resolves
# Qualifiers: + (pass, default), - (fail), ~ (softfail), ? (neutral)
# Suffix:    ~all = soft-fail unmatched; -all = hard-fail
```

```bash
# DKIM — DomainKeys Identified Mail, RFC 6376
selector1._domainkey.example.com. 3600 IN TXT "v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb3DQ..."
# Selector chosen by sender (e.g. selector1, google, mailchimp); allows multiple keys for rotation.
# Each outgoing message has a DKIM-Signature header naming the selector + domain.

# Generate a DKIM key (OpenDKIM)
opendkim-genkey -b 2048 -d example.com -s selector1 -D /etc/opendkim/keys/example.com/
# Output: selector1.private (key) + selector1.txt (DNS record to publish)
```

```bash
# DMARC — Domain-based Message Authentication, Reporting & Conformance, RFC 7489
_dmarc.example.com. 3600 IN TXT "v=DMARC1; p=quarantine; rua=mailto:dmarc-reports@example.com; ruf=mailto:dmarc-forensic@example.com; pct=100; sp=reject; adkim=s; aspf=s"
# Tags:
#   v=DMARC1                  required version
#   p=                        policy: none / quarantine / reject
#   sp=                       subdomain policy (defaults to p)
#   pct=                      percentage of mail to apply policy to (rollout knob)
#   rua=                      aggregate report destination (daily XML)
#   ruf=                      forensic report destination (per-failure)
#   adkim=                    DKIM alignment: r=relaxed, s=strict
#   aspf=                     SPF alignment: r=relaxed, s=strict
```

```bash
# MTA-STS — RFC 8461, opportunistic TLS for inbound mail
_mta-sts.example.com.            3600 IN TXT "v=STSv1; id=20240101000000Z"
mta-sts.example.com.             3600 IN A   198.51.100.30
# HTTPS file at https://mta-sts.example.com/.well-known/mta-sts.txt
# Contents (plain text):
#   version: STSv1
#   mode: enforce
#   mx: mail.example.com
#   max_age: 604800
```

```bash
# TLS-RPT — RFC 8460, reporting destination for MTA-STS / DANE failures
_smtp._tls.example.com. 3600 IN TXT "v=TLSRPTv1; rua=mailto:tls-reports@example.com"
```

```bash
# BIMI — Brand Indicators for Message Identification (logo in inbox UI)
default._bimi.example.com. 3600 IN TXT "v=BIMI1; l=https://example.com/logo.svg; a=https://example.com/vmc.pem"
# Requires DMARC p=quarantine or reject + valid VMC certificate.
```

## resolv.conf and the Resolver

```bash
# /etc/resolv.conf — classic Unix resolver config
# Format:
nameserver 1.1.1.1
nameserver 9.9.9.9
search corp.example.com example.com
domain   corp.example.com                     ; deprecated alternative to search
options timeout:2 attempts:2 rotate edns0 single-request

# nameserver       — up to MAXNS=3; queried in order, fall through on timeout
# search           — list of suffixes appended to short names
# options:
#   timeout:N      seconds to wait per server (default 5)
#   attempts:N     queries per server (default 2)
#   rotate         round-robin between nameservers (vs always first)
#   edns0          enable EDNS0 (default on modern glibc)
#   single-request send A and AAAA serially (older sockets)
#   single-request-reopen open new socket between A and AAAA (workaround for broken NAT)
#   ndots:N        if a name has fewer than N dots, search list is tried first (default 1)
#   no-tld-query   refuse queries that look like single-label TLDs
```

```bash
# /etc/nsswitch.conf — the order in which name services are consulted
# Typical:
hosts: files dns
# More elaborate (modern systemd + mdns):
hosts: files mdns4_minimal [NOTFOUND=return] resolve [!UNAVAIL=return] dns mdns4
# files                /etc/hosts
# mdns4_minimal        only .local names via mDNS
# resolve              systemd-resolved (over DBus or stub on 127.0.0.53)
# dns                  classic resolv.conf path
# myhostname           always provides own hostname (no DNS)
```

```bash
# Test order with getent (uses NSS) and host (bypasses NSS)
getent hosts example.com                       ; respects nsswitch
host example.com                               ; uses BIND resolver lib, not NSS
ping example.com                               ; usually uses NSS getaddrinfo
```

```bash
# systemd-resolved generates /etc/resolv.conf; never edit it directly there.
ls -l /etc/resolv.conf
# /etc/resolv.conf -> /run/systemd/resolve/stub-resolv.conf
# Real config:
sudoedit /etc/systemd/resolved.conf
sudo systemctl restart systemd-resolved
```

## /etc/hosts

The local override file. Read before DNS by default. Same format on every Unix and Windows.

```bash
# Format:
# <ip>  <canonical-name>  [aliases...]
127.0.0.1   localhost
127.0.1.1   myhostname
::1         localhost ip6-localhost ip6-loopback
ff02::1     ip6-allnodes
ff02::2     ip6-allrouters

# Custom entries
10.0.0.50   db.dev   db
10.0.0.51   cache.dev
93.184.216.34 example.com www.example.com
```

```bash
# Common patterns
# 1. Force dev hostname during local development
127.0.0.1   api.local.example.com

# 2. Block ad/tracker domains
0.0.0.0     ads.example.com
0.0.0.0     tracker.example.com

# 3. Test a new server before DNS update propagates
# Edit /etc/hosts: 198.51.100.99  www.example.com
# Then: curl https://www.example.com — hits the new IP
# Remove after verification.

# 4. Pin a name during DNS migrations
198.51.100.99  api.example.com
```

```bash
# Locations
# Linux/macOS:  /etc/hosts
# Windows:      C:\Windows\System32\drivers\etc\hosts (admin write)
# Format identical across platforms.
```

```bash
# Cache invalidation after editing /etc/hosts
sudo systemd-resolve --flush-caches            ; systemd-resolved
sudo killall -HUP mDNSResponder                ; macOS classic
sudo dscacheutil -flushcache                   ; macOS modern
ipconfig /flushdns                              ; Windows
# Most apps re-read /etc/hosts on each lookup, but long-running daemons may cache.
```

## systemd-resolved

```bash
# Service status and per-link config
resolvectl status                              ; full report — global + per-interface
resolvectl status eth0                         ; one interface
resolvectl statistics                          ; cache hits, query counts
resolvectl flush-caches                        ; nuke positive + negative caches

# Query through resolved
resolvectl query example.com                   ; A + AAAA + cache info
resolvectl query --type=MX example.com         ; specific type
resolvectl query --type=TXT _dmarc.example.com

# Set DNS for a link (useful on a per-VPN basis)
sudo resolvectl dns wg0 10.0.0.1
sudo resolvectl domain wg0 ~corp.example.com   ; route corp.example.com through wg0's DNS
sudo resolvectl revert wg0                      ; reset link to default
```

```bash
# /etc/systemd/resolved.conf — global config
[Resolve]
DNS=1.1.1.1#cloudflare-dns.com 9.9.9.9#dns.quad9.net
FallbackDNS=8.8.8.8 1.0.0.1
Domains=~.                                     ; route all queries through these (override per-link)
DNSSEC=allow-downgrade                         ; allow non-DNSSEC zones, validate signed ones
DNSOverTLS=opportunistic                       ; or "yes" for strict
Cache=yes
CacheFromLocalhost=no
ReadEtcHosts=yes
LLMNR=no                                       ; disable Microsoft LLMNR if you don't need it
MulticastDNS=yes                               ; mDNS resolution for .local
```

```bash
# /etc/resolv.conf and resolved
ls -l /etc/resolv.conf
# Common targets:
#   /run/systemd/resolve/stub-resolv.conf   ; uses 127.0.0.53 stub (default, recommended)
#   /run/systemd/resolve/resolv.conf        ; uses upstream directly (no caching)
#   /etc/resolvconf/resolv.conf             ; dynamic generation by resolvconf package

# If apps misbehave with the stub, point them directly:
sudo ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf
```

```bash
# Per-VPN split DNS pattern
# 1. Bring up wg0
# 2. resolvectl dns wg0 10.0.0.1
# 3. resolvectl domain wg0 ~corp.example.com
# Now: corp.example.com queries -> 10.0.0.1 via wg0; everything else -> normal DNS.
```

## Common Diagnostic Workflows

```bash
# The five questions to ask, in order
# 1. Is this name resolving AT ALL?
dig +short example.com                         ; any answer?
getent hosts example.com                       ; does NSS see it?

# 2. WHICH server are we querying?
cat /etc/resolv.conf                           ; classic
resolvectl status                              ; systemd-resolved
scutil --dns                                   ; macOS

# 3. What RR types come back?
dig example.com A
dig example.com AAAA
dig example.com MX
dig example.com NS
dig example.com SOA

# 4. Is DNSSEC validating?
dig +dnssec example.com | grep flags           ; look for "ad"
dig +cd example.com                            ; bypass validation

# 5. Is the AUTHORITATIVE server returning what we expect?
dig +short NS example.com                      ; find the auth servers
dig @ns1.example.com example.com SOA           ; query each one
dig @ns2.example.com example.com SOA           ; compare serials
```

```bash
# Compare resolvers when results differ between users
dig @1.1.1.1   example.com +short
dig @8.8.8.8   example.com +short
dig @9.9.9.9   example.com +short
dig @ns1.example.com example.com +short
# If they disagree: stale cache on one OR split-horizon misconfig OR DNSSEC failure on validators.
```

```bash
# Compare zone serial across all authoritative NS
for ns in $(dig +short NS example.com); do
  echo -n "$ns: "
  dig @"$ns" SOA example.com +short
done
# Mismatch = secondary not refreshing.
```

```bash
# Trace the delegation chain
dig +trace example.com
dig +trace +nodnssec example.com               ; without DNSSEC checking
dig +trace +additional example.com             ; show glue
```

```bash
# Test from outside your network
# - https://dnsviz.net          DNSSEC visualization
# - https://intodns.com         general health check
# - https://www.dnssy.org       DNSSEC signing checker
# - https://mxtoolbox.com       mail-related records
# - https://dnschecker.org      propagation across global resolvers
```

## Common Errors and Fixes

```bash
# NXDOMAIN — "Non-Existent Domain"
# Meaning: the authoritative says this name does not exist
# Causes:
#   - typo in the query
#   - record was deleted but cache hasn't expired
#   - wildcard didn't match
# Fix: dig +trace to find the authoritative; verify zone file; flush caches.
dig nonexistent.example.com
;; status: NXDOMAIN
```

```bash
# SERVFAIL — "Server failure"
# Meaning: resolver couldn't produce an answer
# Causes:
#   - DNSSEC validation failure (most common today)
#   - upstream authoritative unreachable
#   - recursion disabled where expected
#   - lame delegation
# Fix:
#   1. dig +cd example.com  (checking-disabled bypasses DNSSEC)
#      If +cd works but normal fails: DNSSEC chain broken.
#   2. Query authoritative directly: dig @ns1.example.com example.com
#   3. Test multiple resolvers: dig @1.1.1.1 ... ; dig @8.8.8.8 ...
dig brokendnssec.test
;; status: SERVFAIL
```

```bash
# REFUSED — "Server refuses to answer"
# Meaning: server doesn't serve this zone OR refuses to recurse for you
# Causes:
#   - querying an authoritative-only server about a different zone
#   - querying a recursive that has ACLs blocking your source IP
# Fix: use the right server, or get added to the ACL.
dig @1.1.1.1 unrelated.zone.example
;; status: REFUSED
```

```bash
# FORMERR — "Format error"
# Meaning: query was malformed (wrong opcode, bad EDNS, bogus question)
# Causes:
#   - middlebox tampering on the path
#   - server doesn't understand EDNS option you sent
#   - corrupt UDP packet
# Fix: try +noedns (testing only); inspect with tcpdump.
```

```bash
# NOTIMP — "Not implemented"
# Meaning: server doesn't implement this opcode
# Common case: nsupdate to a server that doesn't support DNS UPDATE
# Fix: target a server that does, or add allow-update to its config.
```

```bash
# YXDOMAIN — "Name exists when it shouldn't" (rare; mainly DNS UPDATE pre-conditions)
# NXRRSET — "RRset doesn't exist when it should" (DNS UPDATE pre-conditions)
# YXRRSET — "RRset exists when it shouldn't" (DNS UPDATE pre-conditions)
# NOTAUTH — "Not authoritative for zone"
# NOTZONE — "Name not contained in zone"
```

```bash
# "Name or service not known" (getaddrinfo error)
# Almost always means: resolver returned no result OR was unreachable.
# Fix:
#   1. Check /etc/resolv.conf  -> are nameservers correct?
#   2. dig +short example.com  -> does a manual query work?
#   3. resolvectl status       -> what is systemd-resolved doing?
#   4. ping nameserver_ip      -> is the resolver reachable?
```

```bash
# "Temporary failure in name resolution"
# Meaning: resolver returned SERVFAIL or timed out.
# Fix: same as above, plus check upstream resolver health.
```

```bash
# "Lame delegation"
# Meaning: parent's NS points at a server that doesn't think it's authoritative.
# Detection:
dig +short NS example.com @parent-tld
dig SOA example.com @<each-NS> +short
# If any NS returns REFUSED or non-AA answers -> lame.
# Fix: at the registrar, remove the bad NS; OR fix the server's zone config to add the zone.
```

```bash
# "Broken trust chain"
# Meaning: DNSSEC validation failed.
# Common subcauses:
#   - DS at parent doesn't match any current DNSKEY at child
#   - RRSIGs expired (clock skew, missed key roll)
#   - Algorithm in DS unknown to validators
# Detection:
dig +dnssec example.com                        ; SERVFAIL with EDE 6/7/8/9
delv example.com                               ; tells you the exact reason
dnsviz lookup example.com                      ; web tool for visualization
# Fix: regenerate keys, resign zone, re-publish DS at registrar; wait for TTLs to expire.
```

```bash
# "Connection refused" / "no servers could be reached"
# Meaning: resolver port (53/853/443) blocked or service down.
# Fix: check firewalls; try a different transport (DoH/DoT); ping the IP first.
```

## Common Misconfigurations

```bash
# BAD: CNAME at zone apex (RFC 1034 violation)
# example.com.   IN CNAME shop.shopify.com.
# This breaks SOA, NS, MX, TXT — none can coexist with CNAME.
# FIX (provider-specific):
#   - AWS Route 53: ALIAS record (not a real DNS record, returned as A/AAAA)
#   - Cloudflare:   CNAME flattening (synthesized A response)
#   - Azure DNS:    Alias record set
#   - PowerDNS:     ALIAS record (resolved at query time)
# OR:
#   - Use plain A/AAAA records pointing at static IPs
```

```bash
# BAD: forgot to bump SOA serial
# Symptom: secondaries serve stale data forever.
# Fix:
#   1. Edit zone, increment serial.
#   2. rndc reload example.com (or service-specific reload)
#   3. Verify: dig SOA @ns2.example.com +short  -> serial matches
#   4. If not: rndc notify example.com OR check secondary's allow-notify
```

```bash
# BAD: TTL too low for stable records (e.g. TTL=1)
# Symptom: query rate spikes; load on authoritative; user-visible latency.
# Fix: set 300-3600 for normal records; reserve <60s for active failover only.
```

```bash
# BAD: TTL too high during planned migration (e.g. 86400)
# Symptom: cannot shift traffic for a full day after change.
# Fix: lower TTL to 300 at least 48h BEFORE migration; raise back after change settles.
@   IN  A   93.184.216.34
;; later, two days before migration:
$TTL 300
@   IN  A   93.184.216.34
;; on migration day, change A to new IP. Wait 5 min. Confirm. Raise TTL back.
```

```bash
# BAD: split-horizon misconfig (internal vs external answers diverge or break)
# Symptom: clients on VPN see one set of answers; off-VPN clients see another;
#          Some clients see neither.
# Fix:
#   - Enumerate every record that differs internal vs external; document.
#   - Use BIND views + match-clients ACLs OR separate authoritative servers entirely.
#   - Test from inside AND outside the perimeter after each change.
```

```bash
# BAD: open recursive resolver on the public internet
# Symptom: your resolver is used in DDoS amplification attacks; ISP complains.
# Fix:
#   options { recursion yes; allow-recursion { 127.0.0.1; 10.0.0.0/8; }; };
#   And: rate-limit { responses-per-second 5; window 5; };
#   And: prevent ANY queries: refuse-any in newer BIND; minimal-responses yes.
```

```bash
# BAD: wildcard accidentally matching everything
# example.com:
# *           IN A 93.184.216.34
# www         IN A 93.184.216.34
# Then query for typo.example.com -> matches wildcard, returns the IP.
# Fix: use specific records; OR scope wildcard to a subdomain (*.dev IN A).
```

```bash
# BAD: missing A/AAAA for MX target
# example.com.   IN MX 10 mail.example.com.
# ;; (no A or AAAA for mail.example.com.)
# Symptom: mail bounces with "no MX found" or "host not found" at receivers.
# Fix:
mail.example.com. IN A    198.51.100.20
mail.example.com. IN AAAA 2001:db8::20
```

```bash
# BAD: CNAME used as MX target (RFC 5321 §5.1 forbids this)
# example.com.    IN MX 10 mail.example.com.
# mail.example.com. IN CNAME mailhost.provider.net.
# Fix: use A/AAAA at mail.example.com directly, OR set MX to mailhost.provider.net.
```

```bash
# BAD: stale DS at parent after key rotation
# Symptom: SERVFAIL for entire zone after KSK rolled.
# Fix: update DS at registrar to match new KSK; wait 2x parent's DS TTL.
```

```bash
# BAD: zone transfer (AXFR) world-readable
dig AXFR example.com @ns1.example.com           ; succeeds — zone leaked
# Fix:
zone "example.com" {
    type primary;
    allow-transfer { key transfer-key; 198.51.100.11; };
};
```

## Performance and Hardening

```bash
# BIND hardening boilerplate
options {
    directory "/var/named";
    listen-on    { any; };
    listen-on-v6 { any; };

    // recursive only for trusted networks
    recursion yes;
    allow-recursion { 127.0.0.1; 10.0.0.0/8; };

    // DNSSEC by default
    dnssec-validation auto;

    // Rate-limit responses to mitigate amplification (BIND 9.10+)
    rate-limit {
        responses-per-second 10;
        window 5;
        slip 2;
        log-only no;
    };

    // Minimal responses reduce response size and amplification factor
    minimal-responses yes;

    // Hide version
    version "Tom Servo";

    // EDNS0 buffer cap
    max-udp-size 1232;                          ; matches DNS Flag Day 2020 recommendation

    // Cache size
    max-cache-size 1024m;
};
```

```bash
# Unbound performance config
server:
    interface: 127.0.0.1
    interface: 10.0.0.1
    access-control: 10.0.0.0/8 allow
    do-ip6: yes
    prefetch: yes                               ; refresh records before they expire
    prefetch-key: yes                           ; refresh DNSKEYs proactively
    cache-min-ttl: 0
    cache-max-ttl: 86400
    msg-cache-size: 256m
    rrset-cache-size: 256m
    num-threads: 4
    serve-expired: yes                          ; stale-while-revalidate (RFC 8767)
    serve-expired-ttl: 3600
    qname-minimisation: yes                     ; RFC 9156 — leak less to authoritatives
    harden-dnssec-stripped: yes
    harden-below-nxdomain: yes
    harden-glue: yes
    harden-large-queries: yes
```

```bash
# DNS Flag Day 2020 — UDP buffer cap of 1232 bytes
# Reason: avoid IP fragmentation, which is unreliable across modern networks.
# Action: set max-udp-size 1232 in BIND, edns-buffer-size: 1232 in Unbound.
```

```bash
# Monitor key health metrics
rndc stats                                      ; -> /var/named/data/named.stats
unbound-control stats                           ; queries, cache, latency
# Watch for:
#   - spike in NXDOMAIN     (could indicate DGA / random subdomain attacks)
#   - cache hit rate drop   (resolver overloaded or cache evicting)
#   - SERVFAIL rate         (upstream / DNSSEC issue)
#   - response time p99     (latency to authoritative)
```

```bash
# Block known-bad domains (Pi-hole / RPZ pattern)
# RPZ — Response Policy Zone, RFC-style mechanism in BIND
# named.conf:
options {
    response-policy { zone "rpz.example.com"; };
};
zone "rpz.example.com" {
    type primary;
    file "rpz.example.com.db";
};
# rpz.example.com.db:
$TTL 60
@           IN SOA  ns1 hostmaster ( 2024010101 7200 3600 1209600 60 )
@           IN NS   ns1
ads.bad.example.   CNAME .                     ; NXDOMAIN response
malware.bad.       CNAME *.                    ; NODATA
phish.bad.         A     0.0.0.0               ; sinkhole
```

## Idioms

```bash
# Zone file structure with @ at apex
$TTL 3600
$ORIGIN example.com.
@   IN  SOA   ns1.example.com. hostmaster.example.com. (2024010101 7200 3600 1209600 3600)
@   IN  NS    ns1.example.com.
@   IN  NS    ns2.example.com.
@   IN  A     93.184.216.34
@   IN  MX    10  mail.example.com.
www IN  CNAME @
```

```bash
# SERIAL=YYYYMMDDNN
# Edit on 2024-03-25, second change of the day:
;; serial: 2024032502
# Common gotcha: never let serial decrease (use date 99 trick if you must reset).
# Practical reset: 2099010100 (jump to far future) so you can resume normal numbering after.
```

```bash
# Pre-migration low-TTL ritual
# 1. 72h before: edit zone, $TTL 3600 -> $TTL 300; bump serial.
# 2. 72h-1h before: monitor; verify TTL=300 propagation across resolvers.
# 3. Migration window: change A from old IP to new IP; bump serial.
# 4. Verify everywhere: dig +short A example.com @1.1.1.1 ; @8.8.8.8 ; @9.9.9.9
# 5. 24-48h after stable: raise $TTL back to 3600; bump serial.
```

```bash
# SPF flatten-and-publish pattern
# Problem: SPF record exceeds 10 DNS lookup limit (RFC 7208 §4.6.4)
# Solution: replace include: directives with literal ip4:/ip6: of resolved IPs.
# Tooling: scripts that resolve all the includes recursively and rewrite the TXT record.
# Example output:
example.com. IN TXT "v=spf1 ip4:35.190.247.0/24 ip4:64.18.0.0/20 ip4:64.233.160.0/19 ... -all"
```

```bash
# CAA-record for cert-issuance restriction
example.com. IN CAA 0 issue "letsencrypt.org"
example.com. IN CAA 0 issue "amazon.com"
example.com. IN CAA 0 issuewild ";"
example.com. IN CAA 0 iodef "mailto:security@example.com"
example.com. IN CAA 128 issue "letsencrypt.org" ; flag bit 128 = critical (must understand)
```

```bash
# Dual-stack rollout pattern
;; Phase 1 — add AAAA alongside A
example.com. IN A    93.184.216.34
example.com. IN AAAA 2606:2800:220:1::1

;; Phase 2 — verify happy-eyeballs (RFC 8305) on real clients
;; Phase 3 — deprecate legacy A endpoints if needed
```

```bash
# Multi-region failover via low-TTL CNAME chain
api.example.com.            IN CNAME api-gslb.example.com.
api-gslb.example.com.       IN A     203.0.113.10  ; 60s TTL
;; A health checker rewrites api-gslb's A to a backup IP if primary fails.
```

```bash
# Tunneled-name pattern for cert validation (ACME DNS-01)
_acme-challenge.example.com. IN TXT "abc123challengeresponse"
;; Lifetime: minutes; remove after validation completes.
```

## Tips

- Always increment SOA serial; it is the only signal that propagates updates to secondaries. Forgetting is the single most common DNS bug.
- A CNAME owner cannot have any other record. The zone apex (example.com) cannot be a CNAME because it must have SOA and NS. Use ALIAS/ANAME or A records.
- `dig +trace` is the single most valuable debug command. It shows the entire delegation chain and reveals the broken hop.
- If `dig` works but `getaddrinfo`-based apps fail, the problem is NSS / `/etc/nsswitch.conf` / `/etc/resolv.conf`, not DNS itself.
- SERVFAIL on a DNSSEC-signed zone almost always means validation failure. `dig +cd` (checking-disabled) confirms; if `+cd` works, the chain is broken.
- Lower TTLs to 300s at least 48 hours before any planned IP change. Raise them back after the change is verified.
- Never allow recursion from the public internet. Open recursive resolvers are conscripted into amplification DDoS within minutes.
- Restrict AXFR to your known secondaries. Open AXFR leaks every internal hostname you have.
- Reverse DNS (PTR) for mail servers must match the forward A record, or major receivers will reject your mail outright.
- Inline DNSSEC signing (BIND 9.16+, Knot, PowerDNS) automates the key rotation pain. Use it; do not hand-sign in production.
- Monitor SOA serial across all authoritative NS in CI. Drift is the early warning of a broken zone-transfer pipeline.
- For private DNS, a small Unbound (recursive) plus NSD or Knot (authoritative) is dramatically simpler than BIND and cuts the attack surface by an order of magnitude.
- CAA records prevent any CA you didn't list from issuing certs for your domain. Set them. Audit them in CI.
- Set `version "redacted"` in BIND options to hide the real version from `version.bind` Chaos-class TXT probes.
- Use `dnsviz.net` and `intodns.com` for offline / external visibility into your zone health.
- The DNS Flag Day 2020 cap of 1232 bytes for UDP responses is a good default; larger and you risk fragmentation across the internet.
- Distinguish between recursive and authoritative when reading any DNS error. The same SERVFAIL means very different things on each side.
- The trailing dot matters. `example.com` (no dot) inside a zone gets `$ORIGIN` appended; `example.com.` (trailing dot) is absolute.
- TXT-record string limit is 255 bytes per quoted segment. Long DKIM keys MUST be split across multiple quoted segments.

## See Also

- dig, tls, openssl, polyglot, bash

## References

- [RFC 1034 — Domain Names: Concepts and Facilities](https://www.rfc-editor.org/rfc/rfc1034)
- [RFC 1035 — Domain Names: Implementation and Specification](https://www.rfc-editor.org/rfc/rfc1035)
- [RFC 1995 — Incremental Zone Transfer in DNS (IXFR)](https://www.rfc-editor.org/rfc/rfc1995)
- [RFC 1996 — A Mechanism for Prompt Notification of Zone Changes (NOTIFY)](https://www.rfc-editor.org/rfc/rfc1996)
- [RFC 2136 — Dynamic Updates in the DNS (DNS UPDATE)](https://www.rfc-editor.org/rfc/rfc2136)
- [RFC 2181 — Clarifications to the DNS Specification](https://www.rfc-editor.org/rfc/rfc2181)
- [RFC 2308 — Negative Caching of DNS Queries](https://www.rfc-editor.org/rfc/rfc2308)
- [RFC 2671 / 6891 — EDNS0](https://www.rfc-editor.org/rfc/rfc6891)
- [RFC 2782 — DNS RR for Specifying Service Location (SRV)](https://www.rfc-editor.org/rfc/rfc2782)
- [RFC 2845 / 8945 — TSIG](https://www.rfc-editor.org/rfc/rfc8945)
- [RFC 3596 — DNS Extensions to Support IPv6](https://www.rfc-editor.org/rfc/rfc3596)
- [RFC 4033 — DNS Security Introduction and Requirements](https://www.rfc-editor.org/rfc/rfc4033)
- [RFC 4034 — Resource Records for the DNS Security Extensions](https://www.rfc-editor.org/rfc/rfc4034)
- [RFC 4035 — Protocol Modifications for the DNS Security Extensions](https://www.rfc-editor.org/rfc/rfc4035)
- [RFC 4255 — SSHFP](https://www.rfc-editor.org/rfc/rfc4255)
- [RFC 5155 — DNSSEC Hashed Authenticated Denial of Existence (NSEC3)](https://www.rfc-editor.org/rfc/rfc5155)
- [RFC 5321 — Simple Mail Transfer Protocol](https://www.rfc-editor.org/rfc/rfc5321)
- [RFC 6376 — DomainKeys Identified Mail Signatures (DKIM)](https://www.rfc-editor.org/rfc/rfc6376)
- [RFC 6698 — DNS-Based Authentication of Named Entities (DANE TLSA)](https://www.rfc-editor.org/rfc/rfc6698)
- [RFC 6762 — Multicast DNS](https://www.rfc-editor.org/rfc/rfc6762)
- [RFC 6763 — DNS-Based Service Discovery](https://www.rfc-editor.org/rfc/rfc6763)
- [RFC 7208 — Sender Policy Framework (SPF)](https://www.rfc-editor.org/rfc/rfc7208)
- [RFC 7344 / 8078 — Automating DNSSEC Delegation Trust Maintenance (CDS/CDNSKEY)](https://www.rfc-editor.org/rfc/rfc8078)
- [RFC 7489 — Domain-based Message Authentication (DMARC)](https://www.rfc-editor.org/rfc/rfc7489)
- [RFC 7505 — A "Null MX" No Service Resource Record](https://www.rfc-editor.org/rfc/rfc7505)
- [RFC 7858 — DNS over Transport Layer Security (DoT)](https://www.rfc-editor.org/rfc/rfc7858)
- [RFC 7873 — Domain Name System (DNS) Cookies](https://www.rfc-editor.org/rfc/rfc7873)
- [RFC 8484 — DNS Queries over HTTPS (DoH)](https://www.rfc-editor.org/rfc/rfc8484)
- [RFC 8914 — Extended DNS Errors](https://www.rfc-editor.org/rfc/rfc8914)
- [RFC 9156 — DNS Query Name Minimisation to Improve Privacy](https://www.rfc-editor.org/rfc/rfc9156)
- [RFC 9230 / 9250 — DNS over QUIC (DoQ)](https://www.rfc-editor.org/rfc/rfc9250)
- [RFC 9460 — Service Binding and Parameter Specification via the DNS (SVCB and HTTPS RRs)](https://www.rfc-editor.org/rfc/rfc9460)
- [IANA DNS Parameters — Resource Record Types](https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml)
- [IANA Root Zone Database](https://www.iana.org/domains/root/db)
- [DNS Flag Day 2020 — UDP buffer cap](https://www.dnsflagday.net/2020/)
- [ISC BIND 9 Documentation](https://bind9.readthedocs.io/en/latest/)
- [NLnet Labs Unbound Documentation](https://unbound.docs.nlnetlabs.nl/en/latest/)
- [NLnet Labs NSD Documentation](https://nsd.docs.nlnetlabs.nl/en/latest/)
- [PowerDNS Authoritative + Recursor Documentation](https://doc.powerdns.com/)
- [Knot DNS Documentation](https://www.knot-dns.cz/documentation/)
- [CoreDNS Documentation](https://coredns.io/manual/toc/)
- [systemd-resolved man page](https://www.freedesktop.org/software/systemd/man/latest/systemd-resolved.service.html)
- ["DNS and BIND" by Cricket Liu and Paul Albitz (5th edition, O'Reilly)](https://www.oreilly.com/library/view/dns-and-bind/0596100574/)
- [Cloudflare Learning Center — DNS](https://www.cloudflare.com/learning/dns/what-is-dns/)
- [DNSViz — DNSSEC Visualization](https://dnsviz.net/)
- [intoDNS — DNS Health Check](https://intodns.com/)
