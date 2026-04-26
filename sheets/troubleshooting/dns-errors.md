# DNS Errors

Field manual for diagnosing DNS failures: every response code, resolver-side failure mode, dig-trace anomaly, and "DNS just feels broken" diagnostic, with verbatim error text, root cause, and the fix.

## Setup

DNS is a hierarchical, distributed key/value store keyed by name and type. A query carries a QNAME, QTYPE, and QCLASS; a response carries an RCODE plus zero or more records in the ANSWER, AUTHORITY, and ADDITIONAL sections.

```text
Query:    QNAME=www.example.com  QTYPE=A  QCLASS=IN
Response: RCODE=NOERROR
          ANSWER:     www.example.com. 300 IN A 93.184.216.34
          AUTHORITY:  example.com. 86400 IN NS a.iana-servers.net.
          ADDITIONAL: a.iana-servers.net. 86400 IN A 199.43.135.53
```

Iterative vs recursive:

- **Stub resolver** (the libc on your laptop, or Go's net.Resolver) sends a single recursive query (RD=1) to a recursive resolver and expects a final answer.
- **Recursive resolver** (8.8.8.8, 1.1.1.1, your ISP's, systemd-resolved) does the iterative legwork: queries roots, follows NS referrals down to the authoritative server, returns the answer to the stub.
- **Authoritative server** (the origin of truth for a zone) answers only for zones it is configured to serve, with the AA bit set, and refers (or refuses) for everything else.

Resolution path for `www.example.com`:

```text
stub  ──RD=1──>  recursive resolver
                 ├─> root (.) ────────> referral to com NS
                 ├─> com NS ──────────> referral to example.com NS
                 ├─> example.com NS ──> ANSWER (AA=1)
stub  <─────── final answer ─── recursive resolver
```

Cache layers (in order; any one can lie):

1. Application cache (Java's `networkaddress.cache.ttl`, Node's none-by-default, Go's none).
2. Browser cache (Chrome ~60s, Firefox configurable; visible at `chrome://net-internals/#dns`).
3. OS stub cache (systemd-resolved, mDNSResponder, nscd, dnsmasq).
4. Recursive resolver cache (the one that does the real work; respects TTL).
5. Negative cache (NXDOMAIN/NODATA caches honour SOA MINIMUM, capped at 1h per RFC 2308).

Wire facts:

- UDP port 53 (default), TCP port 53 (truncation fallback or large RRsets).
- DoT TCP/853, DoH TCP/443, DoQ UDP/853.
- Default UDP message size 512 bytes; EDNS0 raises this (commonly to 1232 or 4096).
- Transaction ID is 16 bits → birthday-bound and the basis for cache poisoning concerns (mitigated by source-port randomisation and DNS Cookies).

```bash
# A query for the bare essentials
dig +noall +answer www.example.com A

# What the stub thinks (man 5 resolv.conf)
cat /etc/resolv.conf

# What the system resolver returns (NSS, not DNS — uses /etc/hosts + DNS)
getent hosts www.example.com

# The full picture from one query
dig www.example.com
```

## RCODE Catalog

The 4-bit RCODE field in the DNS header (extended to 12 bits via EDNS) tells you why a query did or didn't succeed. Memorise the common ones (0/2/3/5) and recognise the rare ones.

### 0 — NOERROR

Query was processed successfully. The ANSWER section may be empty (NODATA — see next section) or populated.

```text
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 47213
;; ANSWER SECTION:
www.example.com.  300 IN A  93.184.216.34
```

If status is NOERROR but ANSWER is empty, the name exists but no record of the requested QTYPE exists. This is *not* the same as NXDOMAIN.

### 1 — FORMERR (Format Error)

The server could not parse the request. Almost always a client/library bug, a corrupted UDP datagram, or a mismatch in EDNS options.

```text
;; ->>HEADER<<- opcode: QUERY, status: FORMERR, id: 12345
```

Cause: malformed query (bad name compression, illegal label length >63, missing question section, bogus EDNS option).

Fix: pcap the query (`tcpdump -i any -w /tmp/q.pcap port 53`) and decode in Wireshark; identify the malformed field; patch the client.

### 2 — SERVFAIL (Server Failure)

The server failed to complete the query. Catch-all for "I tried but couldn't." The most operationally annoying RCODE because it can mean a dozen different things.

```text
;; ->>HEADER<<- opcode: QUERY, status: SERVFAIL, id: 47213
;; QUESTION SECTION:
;example.broken.       IN A
```

Common causes (in rough order of frequency):

1. **DNSSEC validation failed** at the recursive resolver. Re-query with `+cd` (checking-disabled) — if it then succeeds, DNSSEC is the culprit.
2. **Lame delegation**: parent has NS records pointing to a server that doesn't host the zone.
3. **Upstream timeout**: recursive resolver couldn't reach any authoritative.
4. **Broken zone**: SOA serial regressed, missing required records, expired secondary.
5. **EDNS confusion**: middlebox dropped EDNS responses; resolver didn't fall back.

Diagnostic:

```bash
dig +cd  example.com           # if this works, DNSSEC failed
dig +trace example.com         # see where the chain breaks
dig @<auth-ns> example.com     # query auth directly to check zone
```

### 3 — NXDOMAIN (Non-Existent Domain)

The QNAME does not exist anywhere under the authoritative zone. Cached per RFC 2308 negative caching.

```text
;; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN, id: 47213
;; AUTHORITY SECTION:
example.com.  3600 IN SOA  ns.icann.org. noc.dns.icann.org. 2024010101 7200 3600 1209600 3600
```

The SOA in the AUTHORITY section sets the negative-cache TTL (the smaller of SOA TTL and SOA MINIMUM, capped at 1h).

Causes: typo, deleted record, name in a different zone than expected, wildcard not matching, QNAME minimization quirks.

### 4 — NOTIMP (Not Implemented)

The server doesn't implement the requested query type. Rare in modern infrastructure; you might see it from very old or stripped-down servers when asking for IXFR, AXFR, or SVCB on legacy authoritatives.

```text
;; ->>HEADER<<- opcode: QUERY, status: NOTIMP, id: 12345
```

Fix: query a different server, or use a supported QTYPE.

### 5 — REFUSED

The server refuses to answer for policy reasons. The number-one example: querying a public IP that doesn't recurse for you.

```text
;; ->>HEADER<<- opcode: QUERY, status: REFUSED, id: 12345
```

Causes:

- Authoritative-only server, you asked for a zone it doesn't host.
- Recursive resolver with ACLs (most modern public resolvers refuse open-recursion to non-customers).
- Zone transfer (AXFR) refused because you're not on the allow-xfer list.
- Rate-limit response slipped in as REFUSED.

Fix: query the right server (`dig @<your-resolver>`), or get added to the allow list.

### 6 — YXDOMAIN

Used by RFC 2136 dynamic updates: "name should not exist, but it does." You won't see this from a normal A/AAAA lookup.

### 7 — YXRRSET

RFC 2136 dynamic update: "RRset that should not exist does."

### 8 — NXRRSET

RFC 2136 dynamic update: "RRset that should exist does not."

### 9 — NOTAUTH

Server is not authoritative for the zone in the query (or, in TSIG context, not authorised to perform the request).

```text
;; ->>HEADER<<- opcode: QUERY, status: NOTAUTH, id: 12345
```

Fix: query the actual authoritative server.

### 10 — NOTZONE

Update name (in RFC 2136 update) is not within the zone specified by the update.

### 16 — BADVERS / BADSIG

Same numeric value, different contexts:

- **BADVERS** (extended via EDNS OPT): EDNS version unsupported; client used a higher version than the server can handle.
- **BADSIG** (TSIG): TSIG/SIG(0) signature did not validate.

```text
;; ->>HEADER<<- opcode: QUERY, status: BADVERS, id: 12345
;; OPT PSEUDOSECTION:
; EDNS: version: 1, flags:; udp: 4096
```

Fix: drop EDNS version (most clients use 0); fix shared key for TSIG.

### 17 — BADKEY

TSIG: key not recognised by the server.

### 18 — BADTIME

TSIG: timestamp on signed message is outside the fudge window. Almost always clock skew between client and server.

```bash
ntpq -p                     # check NTP sync
chronyc tracking            # alternative
```

### 19 — BADMODE

TKEY (RFC 2930): mode not supported.

### 20 — BADNAME

TKEY: duplicate key name.

### 21 — BADALG

TKEY: algorithm not supported.

### 22 — BADTRUNC

TSIG (RFC 4635): bad truncation of MAC.

## NXDOMAIN vs NOERROR/No-Data

The single most-misunderstood DNS distinction. Internalise this and a quarter of "DNS is broken" tickets evaporate.

**NXDOMAIN** — the QNAME does not exist in the zone. *No record of any type exists at this name.*

```text
;; status: NXDOMAIN
;; QUESTION:  doesnotexist.example.com. IN A
;; ANSWER:    (empty)
;; AUTHORITY: SOA for example.com (caps the negative TTL)
```

**NOERROR with empty ANSWER (NODATA)** — the QNAME exists, but no record of the requested QTYPE exists. Other QTYPEs at this name might.

```text
;; status: NOERROR
;; QUESTION:  www.example.com. IN AAAA
;; ANSWER:    (empty)
;; AUTHORITY: SOA for example.com
```

The classic case: an A-only host being queried for AAAA returns NOERROR + empty answer, **not** NXDOMAIN.

Application impact:

- `getaddrinfo(host, NULL, AI_ADDRCONFIG, ...)` will only ask for AAAA if the host has at least one configured non-loopback IPv6 address. If both A and AAAA are queried, NOERROR-NODATA on AAAA is silently fine.
- Glibc and musl differ in subtle ways around `EAI_NONAME` vs `EAI_AGAIN` when only one of A/AAAA is NXDOMAIN and the other is NOERROR-NODATA.
- Java's `InetAddress.getByName()` collapses both to `UnknownHostException` — you can't tell the difference at the Java layer.
- Go's `net.LookupHost` returns `*net.DNSError` with `IsNotFound=true` for NXDOMAIN; for NOERROR-NODATA you get an empty slice and no error.

```bash
# Demonstrate the difference
dig +noall +comments doesnotexist.example.com A
dig +noall +comments www.example.com AAAA   # AAAA on an A-only host
```

UDP/512 limit (RFC 1035 baseline, without EDNS):

- DNS responses larger than 512 bytes have the TC (truncation) bit set.
- Compliant clients retry over TCP/53 transparently.
- With EDNS0, the requestor advertises a larger max payload (e.g. 1232 bytes — the IPv6-friendly value, smaller than 1500-MTU minus headers).

```text
;; flags: qr aa rd ra; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; flags TC=1   ← truncation
```

QNAME minimization (RFC 7816):

- A modern recursive resolver does NOT send the full QNAME to every server in the chain. It sends only the labels needed at each level.
- Querying `www.example.com` from a resolver that does QNAME-min:
  - Asks roots for `com` (QTYPE=NS).
  - Asks com TLD for `example.com` (QTYPE=NS).
  - Asks `example.com` auth for `www.example.com` (QTYPE=A).
- Side effects: some misconfigured authoritative servers return REFUSED or NXDOMAIN to the partial-QNAME NS queries, which can cause SERVFAIL at the resolver. RFC 9156 relaxes this by falling back to non-minimised queries.

## SERVFAIL Diagnostic

SERVFAIL is the resolver saying "I tried." Walk this ladder when you see it:

```bash
# 1. Is it DNSSEC?
dig +cd <name>          # checking disabled — if this succeeds, DNSSEC is broken

# 2. Where in the chain does it break?
dig +trace <name>       # iterative trace from the roots

# 3. Does the auth server have the zone?
dig @<auth-ns> <name>   # if this returns REFUSED/NOTAUTH → lame delegation

# 4. Is the auth server reachable at all?
dig @<auth-ns> <name> +tcp +timeout=10

# 5. Are EDNS options getting munged?
dig +noedns <name>      # disable EDNS — if this works, middlebox is the problem

# 6. Resolver-side log
journalctl -u systemd-resolved -e | grep -i fail
# or, with BIND, /var/log/named/queries.log
```

Lame delegation:

```text
Parent (com TLD) says:    example.com NS ns1.example.com.  ns2.example.com.
Child (ns1.example.com):  REFUSED — I don't have this zone
```

Fix: either configure the named auth server to actually serve the zone, or update the parent NS records.

DNSSEC validation failure:

```text
;; ->>HEADER<<- status: SERVFAIL
;; flags: qr rd ra; ad bit NOT set
$ dig +cd example.com   ← succeeds → DNSSEC is the culprit
```

Diagnose with:

```bash
dig +dnssec example.com DNSKEY
dig +dnssec example.com DS @<parent-ns>
dnsviz print example.com           # visualise the chain
delv +rtrace example.com           # BIND validating tool
```

Upstream timeout:

```text
;; ->>HEADER<<- status: SERVFAIL
;; Query time: 5000 msec    ← maxed out the resolver's per-NS timeout
```

Causes: firewall dropping UDP, IPv6-only path with broken IPv4 fallback, all auth NS unreachable.

Broken zone:

- SOA serial went backwards (or didn't increment) → secondaries refuse to update; primary may serve old data; resolvers may SERVFAIL on inconsistent answers.
- Missing required records (SOA, NS).
- Expired secondary: SOA expire elapsed, secondary stops answering.

```bash
# Check zone health
named-checkzone example.com /etc/bind/zones/example.com.zone
dig SOA example.com @<each-NS>     # serials should match
```

## REFUSED Diagnostic

REFUSED means policy, not failure. The server *can* answer, it *won't*.

Open-resolver hardening:

```text
$ dig @203.0.113.5 example.com
;; status: REFUSED
```

`203.0.113.5` is authoritative-only or restricts recursion to its own customers. Use a real recursor.

AXFR refused (zone transfer):

```text
$ dig @ns1.example.com example.com AXFR
; Transfer failed.
;; status: REFUSED
```

Fix: `allow-transfer { trusted_ips; };` on the auth server, or use IXFR with TSIG.

Authoritative-not-for-this-zone:

```text
$ dig @ns.someotherzone.com example.com
;; status: REFUSED
```

The server hosts other zones, not this one. Find the right NS via `dig +trace` or `dig NS example.com`.

Reflection-attack hardening:

- Public resolvers that do recurse (8.8.8.8, 1.1.1.1) accept queries from anywhere, but rate-limit and validate.
- Most ISP and corporate resolvers refuse external IPs entirely.
- The "open resolver" project tracks misconfigured servers — if your resolver is listed, you're a DDoS amplifier.

## Truncated Responses (TC bit)

```text
;; flags: qr aa rd ra; QUERY: 1, ANSWER: 0, AUTHORITY: 0, ADDITIONAL: 0
;; ANSWER: TC=1
```

The TC bit means "the answer didn't fit in this UDP datagram; retry over TCP."

Default UDP limits:

- Without EDNS: 512 bytes (RFC 1035).
- With EDNS0: whatever the requestor advertised (commonly 1232, 4096, 65535).

```bash
dig www.example.com TXT          # often hits TC=1 due to long SPF/DKIM
dig +tcp www.example.com TXT     # force TCP from the start
dig +bufsize=4096 www.example.com TXT   # raise EDNS buffer

# Verify TCP path actually works
dig +tcp +tries=1 +timeout=5 @8.8.8.8 example.com
```

Common breakages:

1. Stateful firewall/NAT only permits UDP/53 outbound, drops TCP/53 → resolver retries forever, eventually SERVFAILs.
2. Old middleboxes with 512-byte UDP DNS assumption strip EDNS OPT records → resolver sees no EDNS → falls back to 512 → larger zones break.
3. Path MTU black hole on the UDP-large-response path → fragments dropped → resolver retries TCP → see (1).

Fix: open TCP/53 outbound; modernise the middlebox; let EDNS0 cookies negotiate.

## Negative Caching

RFC 2308: NXDOMAIN and NODATA responses are cached. The TTL of the cached negative answer is `min(SOA_TTL, SOA_MINIMUM, max_ncache_ttl)` — capped at 1 hour by most resolvers regardless of what the SOA says.

```text
example.com. 86400 IN SOA ns. hostmaster. 2024010101 7200 3600 1209600 3600
                                                                      ^^^^
                                                              MINIMUM/neg TTL
```

SOA fields, in order:

1. **MNAME** — primary nameserver.
2. **RNAME** — admin email (with `.` instead of `@`).
3. **SERIAL** — incrementing integer; secondaries compare to decide refresh.
4. **REFRESH** — secondary checks every N seconds.
5. **RETRY** — if refresh fails, retry every N seconds.
6. **EXPIRE** — secondary stops serving if it can't reach primary for N seconds.
7. **MINIMUM** — negative-cache TTL (legacy: also default TTL pre-RFC 2308).

Common bug:

```text
example.com. 86400 IN SOA ns. host. 1 3600 600 86400 604800
                                                       ^^^^^^
                                                       7 days!
```

A 7-day negative-cache TTL means an NXDOMAIN propagates for a week. When you finally add the missing record, recursors keep returning NXDOMAIN until 604800s elapse or operators flush.

Fix: set MINIMUM to 300–3600 seconds.

```bash
# Force flush a misbehaving cache
sudo resolvectl flush-caches                       # systemd-resolved
sudo dscacheutil -flushcache                       # macOS
sudo killall -HUP mDNSResponder                    # macOS
ipconfig /flushdns                                 # Windows
sudo rndc flush                                    # BIND
sudo unbound-control flush_zone example.com        # Unbound
```

Negative cache and DNSSEC: NSEC/NSEC3 records prove non-existence; if validation passes the AD bit is set and the negative answer is "authentic."

## dig +trace Walkthrough

`+trace` performs the iterative resolution that a recursive resolver would do. Use it to find the exact step where delegation breaks.

```bash
dig +trace +nodnssec www.example.com
```

Output, step by step:

```text
;; QUESTION SECTION:
;www.example.com.   IN A

; <<>> DiG 9.18.x <<>> +trace +nodnssec www.example.com
;; global options: +cmd

# Step 1: query a root nameserver (one of a..m.root-servers.net)
.        518400 IN NS  a.root-servers.net.
.        518400 IN NS  b.root-servers.net.
...
;; Received 239 bytes from 192.168.1.1#53(192.168.1.1) in 4 ms

# Step 2: a root referred us to com TLD nameservers
com.     172800 IN NS  a.gtld-servers.net.
com.     172800 IN NS  b.gtld-servers.net.
...
;; Received 1182 bytes from 198.41.0.4#53(a.root-servers.net) in 19 ms

# Step 3: com TLD referred us to example.com authoritative
example.com.  172800 IN NS  a.iana-servers.net.
example.com.  172800 IN NS  b.iana-servers.net.
;; Received 730 bytes from 192.5.6.30#53(a.gtld-servers.net) in 53 ms

# Step 4: authoritative answers
www.example.com.  300 IN A  93.184.216.34
;; Received 49 bytes from 199.43.135.53#53(a.iana-servers.net) in 14 ms
```

What goes wrong, and at which step:

- **No referral from root** → roots don't know about your TLD (very rare; the TLD doesn't exist).
- **Referral from TLD points to NS that don't answer** → lame delegation, missing glue, or auth NS down. The trace will time out at this step.
- **NXDOMAIN from intermediate** → some authoritative misconfigured to NXDOMAIN partial QNAMEs (breaks QNAME minimization).
- **REFUSED from authoritative** → auth server doesn't host the zone (lame).
- **SERVFAIL at the end** → DNSSEC validation tripped (rerun with `+cd`) or the auth returned bogus data.

```bash
# Trace with DNSSEC validation
dig +trace +dnssec www.example.com

# Disable validation entirely
dig +trace +cd www.example.com

# Trace asking only one root
dig @a.root-servers.net +norecurse www.example.com
```

Glue records: when an NS record points to a name *inside the zone it serves* (in-bailiwick), the parent must include the A/AAAA of that NS in the ADDITIONAL section. Without glue, you can't reach the NS to ask it about its own zone (chicken-and-egg).

```text
example.com. NS ns1.example.com.       ← in-bailiwick; needs glue
ns1.example.com. A 192.0.2.1            ← the glue record
```

Forgotten glue manifests as `dig +trace` hanging at the parent step, never reaching the auth.

## dig Output Anatomy

Memorise this template; dig output is the single most useful diagnostic in DNS.

```text
; <<>> DiG 9.18.16-1+deb12u1-Debian <<>> www.example.com
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 47213
;; flags: qr rd ra ad; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
; COOKIE: a3b1...

;; QUESTION SECTION:
;www.example.com.       IN  A

;; ANSWER SECTION:
www.example.com.  300   IN  A   93.184.216.34

;; Query time: 14 msec
;; SERVER: 127.0.0.53#53(127.0.0.53) (UDP)
;; WHEN: Mon Apr 21 10:14:32 UTC 2025
;; MSG SIZE  rcvd: 60
```

Line-by-line:

- **`; <<>> DiG 9.X.X <<>> ...`** — version + arguments. Different dig versions have different defaults (e.g. `+yaml`, `+ednsopt`).
- **`;; global options: +cmd`** — what dig recorded as your invocation.
- **`;; Got answer:`** — preamble for the response section.
- **`;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 47213`** — the header. `opcode`: QUERY/IQUERY/STATUS/NOTIFY/UPDATE. `status`: the RCODE name (NOERROR/SERVFAIL/NXDOMAIN/REFUSED/...). `id`: 16-bit transaction ID (a hint at randomness).
- **`;; flags:`** — single-letter flags in the response:
  - `qr` — this is a response (always set in dig output).
  - `aa` — authoritative answer (set by authoritative server).
  - `tc` — truncated; retry over TCP.
  - `rd` — recursion desired (echoed from the query).
  - `ra` — recursion available (set by the resolver if it recurses).
  - `ad` — authentic data (DNSSEC validated successfully).
  - `cd` — checking disabled (you asked the resolver to skip DNSSEC).
- **`QUERY: N, ANSWER: N, AUTHORITY: N, ADDITIONAL: N`** — counts in each section.
- **`;; OPT PSEUDOSECTION:`** — EDNS metadata. `udp: 1232` is the advertised buffer size. `COOKIE` is RFC 7873.
- **`;; QUESTION SECTION:`** — what was asked. Note the leading `;` (it's a comment because the question section is informational on a response).
- **`;; ANSWER SECTION:`** — the answer records. Format: `<name>. <ttl> <class> <type> <rdata>`.
- **`;; AUTHORITY SECTION:`** — NS records (delegation/referral) or SOA (negative answers).
- **`;; ADDITIONAL SECTION:`** — glue, EDNS, OPT records, sometimes A/AAAA for the AUTHORITY section's NS.
- **`;; Query time: 14 msec`** — RTT to the resolver (not end-to-end DNS resolution time).
- **`;; SERVER: 127.0.0.53#53(127.0.0.53) (UDP)`** — which resolver answered, with port and transport.
- **`;; WHEN:`** — local timestamp.
- **`;; MSG SIZE rcvd: 60`** — bytes in the response. Watch this near 512 (UDP without EDNS) and 1232 (typical EDNS budget).

Reading specific records:

```text
www.example.com.  300  IN  A     93.184.216.34
^^^^^^^^^^^^^^^^   ^^^  ^^  ^^^   ^^^^^^^^^^^^^
NAME (FQDN, dot)   TTL  CLASS    RDATA
                           TYPE
```

Multi-line records (long TXT, DNSSEC RRSIGs):

```text
example.com. 300 IN TXT "v=spf1 include:_spf.example.com ~all"
example.com. 300 IN TXT (
    "v=DKIM1; k=rsa; "
    "p=MIGfMA0GCSqGS..."
    " ...QIDAQAB"
)
```

## dig Common Flags Quick Reference

```bash
# Output shaping
dig +short  example.com           # only the rdata
dig +noall  +answer  example.com  # only the ANSWER section
dig +noall  +authority example.com
dig +nocmd  +noquestion example.com
dig +stats  example.com           # explicitly include stats footer
dig +yaml   example.com           # YAML output

# Resolution control
dig +recurse   example.com        # set RD bit (default)
dig +norecurse example.com        # clear RD bit (asks for what's cached only)
dig +trace     example.com        # iterative trace from root
dig +tries=1   example.com        # don't retry on UDP timeout
dig +retry=2   example.com        # number of UDP retries
dig +timeout=2 example.com        # per-try timeout in seconds
dig +ndots=1   example.com        # ndots override (resolv.conf)

# DNSSEC
dig +dnssec    example.com        # request RRSIG/NSEC
dig +cd        example.com        # checking-disabled (don't validate)
dig +nocd      example.com        # default — validate

# Transport
dig +tcp       example.com        # force TCP from the start
dig +vc        example.com        # synonym (virtual circuit)
dig +tls       example.com        # DoT (knot's kdig only)
dig +https     example.com        # DoH (knot's kdig only)
dig +bufsize=4096  example.com    # advertise larger EDNS buffer
dig +noedns    example.com        # disable EDNS entirely

# Targeting
dig @8.8.8.8       example.com
dig @1.1.1.1 -p 53 example.com
dig -t MX          example.com    # specific RRTYPE
dig -t TXT _dmarc.example.com
dig -x 8.8.8.8                    # reverse PTR (auto-formats in-addr.arpa)
dig -x 2001:db8::1                # reverse v6 PTR (ip6.arpa)
dig -4             example.com    # use IPv4 only
dig -6             example.com    # use IPv6 only

# Class
dig -c IN  example.com            # default
dig -c CH txt version.bind        # CHAOS class for server version banners

# Multiple in one run
dig example.com A example.com AAAA example.com MX
```

Useful combos:

```bash
# What everyone really wants
dig +short example.com

# Trace + just the relevant lines
dig +trace +noall +answer example.com

# Check a single auth without recursion
dig @ns1.example.com +norecurse example.com SOA

# Reverse lookup
dig +short -x 93.184.216.34

# DNSSEC chain dump
dig +dnssec example.com DNSKEY
dig +dnssec example.com DS @<parent-ns>
```

## DNSSEC Failures

DNSSEC adds RRSIG signatures over RRsets, NSEC/NSEC3 records for proof-of-non-existence, and a chain of trust from the root KSK down via DS/DNSKEY records.

States you'll see:

- **`ad` flag set** → resolver validated successfully.
- **`ad` flag absent** → not validated. Could be: zone unsigned, resolver doesn't validate, validation failed.
- **`status: SERVFAIL`** → validation failed; resolver refuses to return data.
- **`status: NOERROR` with `+cd`** but **`SERVFAIL` without `+cd`** → DNSSEC is broken.

Validation states:

- **SECURE** — chain validated.
- **INSECURE** — zone has provable opt-out (no DS at parent, NSEC proves it).
- **BOGUS** — chain present but fails (bad signature, expired RRSIG, missing DS).
- **INDETERMINATE** — no trust anchor for this part of the tree (extremely rare; root anchor is universal).

Common DNSSEC failures:

1. **RRSIG expiration**:

```text
example.com. 300 IN RRSIG A 8 2 300 20240101000000 20231201000000 12345 example.com. <sig>
                                    ^^^^^^^^^^^^^^^ inception      ^^^^^^^^^^^^^^^ expiration
```

If `now > expiration`, validation fails. Common when zone signing is forgotten or the auto-resigner cron broke. Fix: re-sign the zone.

2. **KSK rotation without DS update**:
   - Operator generated new KSK, signed with new RRSIG, but didn't push the new DS to the parent.
   - Resolver follows old DS → expects old KSK → new signatures fail.
   - Fix: push DS update to the registrar/parent and wait for parent TTL to elapse.

3. **Time skew on the validator**:
   - Validator clock is wrong → RRSIG looks expired or not-yet-valid.
   - Fix: sync NTP. `chronyc tracking` or `timedatectl status`.

4. **Algorithm mismatch / unsupported**:
   - Old resolver doesn't understand newer alg (e.g. ECDSAP384SHA384 = 14, ED25519 = 15, ED448 = 16).
   - Resolver treats as INSECURE if no compatible alg.

5. **Chain broken at a parent**:
   - DS at parent doesn't match any DNSKEY at child.
   - Diagnose with `dnsviz print example.com`.

Diagnostic commands:

```bash
# Show signed records
dig +dnssec example.com A
dig +dnssec example.com DNSKEY
dig +dnssec example.com DS @<parent-ns>

# BIND's chain-trace (replaces deprecated +sigchase)
delv example.com
delv +rtrace example.com

# Visualisation
dnsviz print example.com
dnsviz query example.com

# Test against a validating resolver
dig @1.1.1.1 example.com         # Cloudflare validates
# vs
dig @<isp-resolver> example.com  # may or may not validate
```

The deprecated `+sigchase` (with `+trusted-key=`) used a key file you supplied; modern setups use `delv` which reads from `/etc/bind/bind.keys` or compiled-in trust anchors.

## Resolver-Side Errors

The resolver-side is everything that happens before a query leaves your laptop. Bugs here masquerade as DNS issues but never touch a real DNS server.

`/etc/resolv.conf` misconfiguration:

```text
# Pointing nowhere reachable
nameserver 192.168.99.99       # not on your network
nameserver 10.0.0.1            # behind a VPN that's down

# Order matters
nameserver 192.168.1.1         # tried first
nameserver 8.8.8.8             # only used if first times out
```

`/etc/hosts` overrides: NSS consults `/etc/hosts` before DNS by default. A stale entry lies forever.

```text
# /etc/hosts
127.0.0.1   localhost
93.184.216.34   example.com    # this overrides DNS!
```

Diagnose: `getent hosts example.com` (uses NSS, sees /etc/hosts) vs `dig example.com` (always DNS).

NSS order (`/etc/nsswitch.conf`):

```text
hosts:  files mdns4_minimal [NOTFOUND=return] dns mymachines
```

- `files` = /etc/hosts
- `dns` = DNS via resolv.conf
- `mdns` / `mdns4` = multicast DNS (Bonjour/Avahi)
- `mymachines` = systemd-machined registered machines
- `[NOTFOUND=return]` = if mdns4_minimal returns NOTFOUND, stop here; don't fall through

Common pitfall: mDNS short-circuits a `.local` query before DNS even sees it. If you have a `.local` zone in real DNS, the resolution fails because mDNS handles those names exclusively.

systemd-resolved cache:

```bash
resolvectl status                    # per-link configuration
resolvectl query example.com         # one-shot query via resolved
resolvectl flush-caches              # clear cache
resolvectl statistics                # cache hit rate
resolvectl monitor                   # live query stream

# Configuration in /etc/systemd/resolved.conf
# DNSStubListener=yes  → listens on 127.0.0.53
# DNSSEC=allow-downgrade
# DNSOverTLS=opportunistic
```

macOS:

```bash
# Flush all DNS caches
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder

# Inspect resolution config
scutil --dns                         # all configured resolvers per scope

# Test query via system resolver
dscacheutil -q host -a name example.com
```

Windows:

```text
ipconfig /flushdns
ipconfig /displaydns
nslookup example.com
nslookup example.com 8.8.8.8
```

The "ping works but X doesn't" diagnostic: some applications bypass NSS (and therefore /etc/hosts and resolv.conf) by using their own resolver (Java, Go's pure-Go resolver under certain build tags, Chrome's Async DNS).

```bash
# What does the libc resolver see?
getent hosts example.com

# What does dig see (via your default resolver)?
dig +short example.com

# What does dig see going to a public resolver?
dig +short @1.1.1.1 example.com

# What does ping see (libc-resolver)?
ping -c1 example.com

# What does Java see? (run a tiny test)
java -e 'java.net.InetAddress.getByName("example.com").getHostAddress()'

# What does Go see?
GODEBUG=netdns=go go run -ldflags '-X main.host=example.com' lookup.go
GODEBUG=netdns=cgo go run lookup.go
```

If `getent` and `dig` agree but the application disagrees, the application has its own resolver — find it.

## CNAME Issues

A CNAME maps a name to another name. Resolution chases the chain until it hits an A/AAAA (or another non-CNAME terminal type).

```text
www.example.com.  300 IN CNAME  app.example.com.
app.example.com.  300 IN CNAME  app-prod.us-east-1.elb.amazonaws.com.
app-prod.us-east-1.elb.amazonaws.com. 60 IN A 198.51.100.10
```

Failure modes:

1. **CNAME loop**:

```text
a.example.com.  IN CNAME  b.example.com.
b.example.com.  IN CNAME  a.example.com.
```

A modern resolver detects loops and returns SERVFAIL. Older resolvers chased indefinitely.

2. **CNAME at apex (illegal per RFC 1034 §3.6.2)**:

```text
example.com.  IN CNAME  www.example.com.   ← ILLEGAL
example.com.  IN SOA   ...
example.com.  IN NS    ns1.example.com.
example.com.  IN MX    10 mail.example.com.
```

A CNAME forbids any other RR at the same name, but a zone apex *requires* SOA and NS. Modern providers (Cloudflare, Route53) offer ALIAS / ANAME / CNAME-flattening — they evaluate the target at query time and synthesise an A record for the apex.

```text
# Route53 ALIAS (synthetic — not real DNS)
example.com.  300 IN A  93.184.216.34   ← synthesised at query time

# Cloudflare CNAME-flattening
example.com.  300 IN A  104.16.x.x      ← Cloudflare resolves CNAME, returns A
```

3. **CNAME plus other RRs at the same name** (illegal):

```text
www.example.com.  IN CNAME  app.example.com.
www.example.com.  IN A      192.0.2.1     ← ILLEGAL
```

named-checkzone catches this; some DNS UIs let you save it anyway and then secondaries refuse to load.

4. **Long CNAME chains**: each step is its own DNS query. If any link is slow, latency adds up. Chains of 10+ hops happen with SaaS providers.

## AAAA-vs-A Blackholes

The IPv6 happy-path failure that haunts users with bad IPv6:

```text
1. App calls getaddrinfo("example.com", ...).
2. AAAA returns an IPv6 address.
3. App tries connect() to the v6 address.
4. There's no working IPv6 path → connect blocks until kernel timeout (~75s).
5. App finally tries the A record → works.
6. User waited 75 seconds for a "DNS" problem that's actually a connectivity problem.
```

Diagnostic:

```bash
dig AAAA example.com
ping6 -c2 example.com
mtr -6 example.com
mtr -4 example.com
```

If `dig AAAA` returns an address but `ping6` times out, you have a v6 blackhole.

Mitigations:

- **Happy Eyeballs (RFC 8305)**: client races v6 and v4 connect attempts in parallel, with v6 a small head start. Used by Chrome, Firefox, getaddrinfo on modern Linux/macOS. Caps the wait to ~250ms.
- **Disable IPv6 on the host**: removes AAAA from getaddrinfo's response (with AI_ADDRCONFIG).

```bash
# Linux: disable v6 globally
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1

# Linux: disable v6 on one interface
sudo sysctl -w net.ipv6.conf.eth0.disable_ipv6=1
```

- **Configure the captive portal/network properly** so v6 actually works.

## /etc/resolv.conf Semantics

```text
# Comments start with # or ;
search corp.example.com example.com
domain corp.example.com         # legacy single-domain (search wins if both present)

nameserver 192.0.2.1            # primary
nameserver 192.0.2.2            # tried only if primary times out

options ndots:2 timeout:1 attempts:2 rotate edns0 trust-ad use-vc single-request
```

Key options:

- **`search`**: when a query has fewer than `ndots` dots, append each search-list entry in order until one resolves.
- **`ndots:N`**: minimum number of dots in a name before it's treated as fully qualified. Default 1 on Linux. Famous Kubernetes pain: pods get `ndots:5`, so `google.com` (one dot) is tried with five suffixes first.
- **`timeout:N`**: per-server timeout (seconds).
- **`attempts:N`**: total tries per server.
- **`rotate`**: round-robin across nameservers instead of always trying the first.
- **`edns0`**: enable EDNS0 in libc resolver (older glibc).
- **`trust-ad`**: pass through the AD bit from upstream (otherwise libc clears it).
- **`use-vc`**: always use TCP.
- **`single-request`** / **`single-request-reopen`**: serialise A/AAAA queries instead of in parallel; works around buggy NATs that confuse simultaneous A+AAAA on the same source port.

`ndots` Kubernetes example:

```text
# Inside a pod
search default.svc.cluster.local svc.cluster.local cluster.local
options ndots:5

$ nslookup google.com
# tries google.com.default.svc.cluster.local
# tries google.com.svc.cluster.local
# tries google.com.cluster.local
# finally tries google.com.
```

Each non-final query is an NXDOMAIN round-trip. Fix: `dnsConfig.options: [{name: ndots, value: "1"}]` in the Pod spec, or always FQDN your names with a trailing dot.

systemd-resolved makes `/etc/resolv.conf` a symlink:

```text
/etc/resolv.conf -> /run/systemd/resolve/stub-resolv.conf
# stub-resolv.conf points at 127.0.0.53 (the resolved stub)
```

You can opt out:

```bash
sudo rm /etc/resolv.conf
sudo ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf
# Now /etc/resolv.conf reflects per-link DNS, not the stub.
```

NetworkManager / dhclient management: anything writing `/etc/resolv.conf` based on DHCP. To pin it: `chattr +i /etc/resolv.conf` (immutable) — but this fights with the DHCP client.

## systemd-resolved

Modern Linux systems (Ubuntu, Fedora, Debian-with-systemd) often run systemd-resolved as the local stub.

```bash
# Status overview
resolvectl status

# Per-interface DNS configuration
resolvectl status eth0

# Diagnostic query (uses the same path as applications)
resolvectl query example.com
resolvectl query --type=AAAA example.com
resolvectl query --cache=no example.com   # bypass cache

# Cache management
resolvectl flush-caches
resolvectl statistics
resolvectl reset-server-features          # clear EDNS feature-detection state

# Live query monitoring
resolvectl monitor
```

Configuration in `/etc/systemd/resolved.conf`:

```text
[Resolve]
DNS=1.1.1.1#cloudflare-dns.com 8.8.8.8#dns.google
FallbackDNS=9.9.9.9
Domains=~.
DNSSEC=allow-downgrade
DNSOverTLS=opportunistic
DNSStubListener=yes
Cache=yes
ReadEtcHosts=yes
```

- **DNSSEC=yes** — strict; SERVFAIL on bogus.
- **DNSSEC=allow-downgrade** — try DNSSEC, fall back if upstream doesn't support it.
- **DNSSEC=no** — never validate.
- **DNSOverTLS=yes** — strict; refuse plain DNS.
- **DNSOverTLS=opportunistic** — try DoT, fall back to plain.
- **DNSStubListener=yes** — listen on 127.0.0.53 (default; what /etc/resolv.conf points at).

Per-link DNS via systemd-networkd:

```text
# /etc/systemd/network/10-eth0.network
[Network]
DHCP=yes
DNS=10.0.0.53
Domains=corp.example.com
```

Or via networkctl:

```bash
sudo resolvectl dns eth0 1.1.1.1 1.0.0.1
sudo resolvectl domain eth0 example.com
```

Per-link DNS lets you have one resolver for VPN traffic and another for everything else (split-horizon DNS that works).

## Common DNS Records and Their Failure Modes

### A — IPv4 address

```text
www.example.com. 300 IN A 93.184.216.34
```

Failure modes: typo in IP, stale record after server moved, multiple A records for round-robin without health checks (clients keep trying dead IPs).

### AAAA — IPv6 address

```text
www.example.com. 300 IN AAAA 2606:2800:220:1:248:1893:25c8:1946
```

Failure modes: present but no working v6 path (see AAAA blackholes); IPv6-only host with v4-only client.

### CNAME — Canonical name (alias)

```text
www.example.com. 300 IN CNAME app.example.com.
```

Failure modes: at apex (illegal); coexists with other RRs (illegal); loop; chain too long.

### MX — Mail exchanger

```text
example.com. 3600 IN MX 10 mx1.example.com.
example.com. 3600 IN MX 20 mx2.example.com.
```

Lower priority = preferred. Failure modes: target is a CNAME (RFC 5321 forbids); target has no A/AAAA; missing trailing dot makes target relative.

### NS — Nameserver delegation

```text
example.com. 86400 IN NS ns1.example.com.
example.com. 86400 IN NS ns2.example.com.
```

Failure modes: lame delegation (NS doesn't host zone); missing glue for in-bailiwick NS; only one NS (RFC 2182 says at least 2, ideally on different prefixes).

### SOA — Start of Authority

```text
example.com. 3600 IN SOA ns1.example.com. admin.example.com. (
    2024010101  ; serial
    7200        ; refresh (2h)
    3600        ; retry (1h)
    1209600     ; expire (14d)
    3600        ; minimum (negative TTL)
)
```

Failure modes: serial regression (secondaries refuse to update); RNAME contains literal `@` instead of `.`; missing trailing dot; too-large minimum TTL bakes NXDOMAINs into caches.

### TXT — Text records

```text
example.com. 300 IN TXT "v=spf1 include:_spf.example.com ~all"
selector1._domainkey.example.com. 300 IN TXT "v=DKIM1; k=rsa; p=MIGfMA0..."
_dmarc.example.com. 300 IN TXT "v=DMARC1; p=reject; rua=mailto:dmarc@example.com"
example.com. 300 IN TXT "google-site-verification=abc123"
_acme-challenge.example.com. 60 IN TXT "abc..."
```

Failure modes: too long for UDP without truncation; multiple TXTs concatenated incorrectly; quoting issues.

### PTR — Pointer (reverse DNS)

```text
34.216.184.93.in-addr.arpa. 300 IN PTR www.example.com.
6.4.9.1.8.c.5.2.3.9.8.1.8.4.2.0.1.0.0.0.0.2.2.0.0.0.8.2.6.0.6.2.ip6.arpa. IN PTR www.example.com.
```

Failure modes: missing PTR (mail servers reject your mail); PTR doesn't match HELO; reverse zone not delegated to you by RIR.

### SRV — Service location

```text
_xmpp-server._tcp.example.com. 300 IN SRV 10 100 5269 xmpp.example.com.
                                            ^^^ ^^^ ^^^^
                                            pri wt  port  target
```

Failure modes: target is a CNAME (RFC says no); port wrong; weight 0 with priority differences confuses clients.

### CAA — Certificate Authority Authorization

```text
example.com. 3600 IN CAA 0 issue "letsencrypt.org"
example.com. 3600 IN CAA 0 issuewild ";"
example.com. 3600 IN CAA 0 iodef "mailto:security@example.com"
```

Failure modes: issuer not listed → CA refuses to issue cert; wildcard mistaken for regular issue.

### DS / DNSKEY / NSEC / NSEC3 / RRSIG — DNSSEC chain

```text
example.com. 86400 IN DS 12345 8 2 abc...      ← in parent zone
example.com.   300 IN DNSKEY 257 3 8 ...       ← KSK in child
example.com.   300 IN DNSKEY 256 3 8 ...       ← ZSK
example.com.   300 IN RRSIG A 8 2 300 20240301 20240201 12345 example.com. <sig>
```

Failure modes: DS doesn't match DNSKEY; RRSIG expired; NSEC chain incomplete (proves wrong non-existence).

### SVCB / HTTPS — Service binding

```text
example.com. 300 IN HTTPS 1 . alpn="h2,h3" port=443 ipv4hint=93.184.216.34
example.com. 300 IN SVCB  1 . alpn="dot" port=853
```

Modern alternative to SRV for HTTPS; advertises ALPN, ECH, HTTP/3 ports. Failure modes: clients that don't understand SVCB ignore it; misconfigured ALPN values cause negotiation failures.

## Reverse DNS (PTR) Issues

```bash
dig -x 93.184.216.34
# 34.216.184.93.in-addr.arpa. 86400 IN PTR www.example.com.

dig -x 2001:db8::1
# 1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa.
```

Common errors:

```text
;; status: NXDOMAIN
;; QUESTION SECTION:
;34.216.184.93.in-addr.arpa.    IN PTR
```

= no PTR exists for that IP. Cloud-provider IPs often have a generic PTR; some have none.

Mail servers' PTR check:

```text
Apr 21 10:14:32 mail postfix/smtpd[1234]: NOQUEUE: reject: RCPT from unknown[203.0.113.5]:
   450 4.7.1 Client host rejected: cannot find your hostname, [203.0.113.5];
```

Or a PTR-mismatch:

```text
Apr 21 10:14:32 mail postfix/smtpd[1234]: NOQUEUE: reject: RCPT from server.bad.example[203.0.113.5]:
   554 5.7.1 ... reject_unverified_client: lookup error
```

Fix: configure forward-confirmed reverse DNS (FCrDNS) — `dig -x IP` and `dig PTR-result` round-trip and match.

```bash
# Verify FCrDNS
PTR=$(dig +short -x 203.0.113.5 | head -1)
dig +short "$PTR"
# Should equal 203.0.113.5
```

Configuring PTR usually requires the IP holder to delegate the reverse zone to you (or to provide a UI). For cloud:

- **AWS**: VPC EC2 → Console → "Update reverse DNS" on Elastic IP.
- **GCP**: only on Cloud DNS-managed reverse zones; or per-instance via `--public-ptr-domain-name`.
- **Azure**: per-public-IP property `reverseFqdn`.
- **Self-hosted on owned IP space**: ARIN/RIPE/APNIC delegates the reverse zone to your nameservers.

## CDN Edge / Anycast DNS Subtleties

Anycast: same IP announced from multiple BGP locations; routing picks the topologically nearest one. The resolver sees one IP but reaches a different server depending on which network they're on.

```bash
# Route to 1.1.1.1 from London vs Tokyo lands on different POPs
mtr -r -c10 1.1.1.1
```

GeoDNS: the authoritative server returns different A/AAAA based on the resolver's IP (or, with EDNS Client Subnet, the *client*'s prefix).

```bash
# Same query, different answers depending on where you are
dig @1.1.1.1 cdn.example.com           # answer reflects 1.1.1.1's subnet
dig @8.8.8.8 cdn.example.com           # different answer

# Force a specific client subnet (if auth supports ECS)
dig +subnet=203.0.113.0/24 cdn.example.com
```

The "different IPs from different ISPs" is not a bug — it's GeoDNS doing its job. A reproducible diagnostic always specifies `@server` and (where applicable) `+subnet=`.

EDNS Client Subnet (ECS, RFC 7871):

- Resolver appends a netmask of the client's IP to upstream queries.
- Auth uses that to pick a region-appropriate answer.
- Cloudflare and Google honour ECS; some privacy-focused resolvers (Quad9) strip it.
- Visible in dig with `+subnet=`:

```text
;; OPT PSEUDOSECTION:
; CLIENT-SUBNET: 203.0.113.0/24/0
```

## Mail-Related DNS

### SPF (Sender Policy Framework)

```text
example.com. 3600 IN TXT "v=spf1 include:_spf.google.com ip4:203.0.113.0/24 -all"
```

Mechanisms:

- `include:domain` — recursively include another SPF record (counts toward 10-lookup limit).
- `ip4:CIDR` / `ip6:CIDR` — literal address ranges.
- `a` / `mx` — the domain's own A/MX hosts.
- `~all` — softfail (treat as suspicious).
- `-all` — hardfail (reject).
- `?all` — neutral.
- `+all` — pass everything (don't do this).

Failure modes:

- **PermError**: more than 10 DNS lookups (the SPF spec limit). `dig TXT example.com` then count `include:` recursions. Fix: flatten `include:` chains to `ip4:` ranges, or split your SPF.
- Missing record entirely → some receivers treat as `?all`, others reject as a fail.
- Multiple SPF records on the same name (illegal; receivers PermError).

### DKIM (DomainKeys Identified Mail)

```text
selector1._domainkey.example.com. 300 IN TXT "v=DKIM1; k=rsa; p=MIGfMA0GCSqGSIb..."
```

The `selector` is arbitrary (Google uses `google._domainkey`, Mailgun uses `mta._domainkey`). The `p=` field is the public key.

Failure modes:

- Key longer than 255 chars: must be split into multiple quoted strings inside one TXT (DNS limits each string to 255 chars).
- Wrong selector: messages signed with `mta` selector but DNS only has `default` published.
- Stale public key after rotation: old signatures fail until receivers re-cache.

### DMARC (Domain-based Message Authentication, Reporting & Conformance)

```text
_dmarc.example.com. 300 IN TXT "v=DMARC1; p=reject; rua=mailto:dmarc@example.com; pct=100"
```

- `p=none` — monitor only.
- `p=quarantine` — send failures to spam.
- `p=reject` — reject failures outright.
- `pct=` — percentage of failing mail to apply the policy to (rollout knob).
- `rua=` — aggregate reports.
- `ruf=` — forensic reports.

Failure modes: DMARC misalignment (the From: domain doesn't align with SPF/DKIM passing domains); `p=reject` deployed before DKIM/SPF pass for all senders → legitimate mail rejected.

### MX hardfail / softfail SPF

`-all` (hardfail): receiver SHOULD reject. `~all` (softfail): receiver SHOULD accept-but-mark.

The 10-DNS-lookup limit:

```text
v=spf1 include:_spf.a.com include:_spf.b.com include:_spf.c.com ... -all
       ^^^^^^^               ^^^^^^^               ^^^^^^^
        each lookup +N (recursive includes)
```

Test with `spfquery` or online checkers. If you exceed 10, the receiver's evaluator returns `permerror` and treats the result as fail or neutral (depending on receiver policy).

## The "DNS Just Feels Broken" Diagnostic Ladder

Walk every step. Don't skip any. Most "DNS broken" reports are diagnosed before step 7.

### Step 1: does the query return anything at all?

```bash
dig example.com
```

- If you get an answer: DNS is fine; the problem is elsewhere (application, routing, TLS).
- If timeout: your default resolver is unreachable. Skip to step 2.
- If SERVFAIL/NXDOMAIN/REFUSED: capture the rcode and move on.

### Step 2: is your resolver the problem? Try Google.

```bash
dig @8.8.8.8 example.com
```

- If 8.8.8.8 works but your default doesn't: your local resolver is broken. Restart systemd-resolved, or change `/etc/resolv.conf` to 8.8.8.8 temporarily.
- If 8.8.8.8 also fails: it's not your resolver.

### Step 3: try Cloudflare too.

```bash
dig @1.1.1.1 example.com
```

Confirms whether the problem is upstream of all public resolvers. If both 8.8.8.8 and 1.1.1.1 disagree with your local but agree with each other, the issue is local. If they all return the same SERVFAIL, the problem is at the auth zone.

### Step 4: trace the delegation.

```bash
dig +trace example.com
```

Find the step where the chain breaks. Fix the parent that's pointing at a lame NS, or the NS that's not serving the zone.

### Step 5: does NSS see something different?

```bash
nslookup example.com
```

`nslookup` uses different code from `dig`; sometimes triggers different resolver behaviour.

### Step 6: does the system resolver agree?

```bash
getent hosts example.com
```

`getent` consults NSS — `/etc/hosts`, mDNS, then DNS. If `getent` and `dig` disagree, look at /etc/hosts and /etc/nsswitch.conf.

### Step 7: read the configuration.

```bash
cat /etc/resolv.conf
cat /etc/nsswitch.conf
```

- Wrong nameserver IP?
- Wrong search domain or `ndots:`?
- mDNS short-circuiting `.local`?

### Step 8: ask systemd-resolved.

```bash
resolvectl status
resolvectl query example.com
resolvectl monitor &      # start live monitor
resolvectl flush-caches
resolvectl query example.com
```

The `monitor` view shows every query and which interface/resolver it went to.

### Step 9: tcpdump.

```bash
sudo tcpdump -i any -n -s0 -w /tmp/dns.pcap port 53 or port 853 or port 443
# in another terminal
dig example.com
sudo pkill tcpdump
tshark -r /tmp/dns.pcap -Y dns
```

You see the actual queries leaving and answers returning. If no query leaves: stub is broken or routing is wrong. If query leaves but no answer returns: server problem or middlebox dropping.

### Step 10: TCP fallback.

```bash
dig +tcp example.com
dig +tcp @8.8.8.8 example.com
```

If UDP fails but TCP works: PMTU blackhole, EDNS issue, or middlebox rewriting UDP DNS.

## EDNS / EDNS0 Issues

EDNS0 (RFC 6891) extends the DNS header via an OPT pseudo-RR in the ADDITIONAL section. Visible in dig:

```text
;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags: do; udp: 1232
; COOKIE: f1e2d3c4...
; CLIENT-SUBNET: 203.0.113.0/24/0
; PADDING: 64
```

Common EDNS extensions:

- **DO bit** — DNSSEC OK; resolver wants RRSIG/NSEC.
- **Cookie (RFC 7873)** — replay/spoof mitigation.
- **Padding (RFC 7830)** — mask query length under DoT/DoH.
- **Client Subnet (RFC 7871)** — for GeoDNS.

Failure modes:

- **Old middlebox strips OPT** → resolver thinks server has no EDNS → falls back to 512-byte UDP → larger answers truncate → SERVFAIL.
- **EDNS buffer too large** → response fragmented at IP layer → fragments dropped → resolver retries with smaller buffer (or never retries, if it doesn't track this).
- **EDNS version unsupported** → BADVERS rcode (16). Drop to version 0.

```bash
# Probe for EDNS support
dig +noedns example.com           # no EDNS in query
dig +bufsize=1232 example.com     # explicit smaller buffer
dig +ednsflags=0x8000 example.com # DO bit
dig +nocookie example.com         # disable cookies
```

EDNS Client Subnet is privacy-leaky (the auth server learns which subnet you're on); modern resolvers like 1.1.1.1 deliberately strip it.

## DNS over TLS (DoT) and DNS over HTTPS (DoH)

Plain DNS is unencrypted and unauthenticated. DoT and DoH add confidentiality and (with cert validation) server authentication.

### DoT — DNS over TLS (RFC 7858)

```text
Port 853, TCP, TLS 1.2+. Plain DNS protocol over TLS.
```

```bash
# kdig (Knot DNS) supports DoT
kdig @1.1.1.1 +tls example.com
kdig @1.1.1.1 +tls-ca +tls-host=cloudflare-dns.com example.com

# Connect raw to verify cert
openssl s_client -connect 1.1.1.1:853 -servername cloudflare-dns.com </dev/null

# systemd-resolved with opportunistic DoT
# /etc/systemd/resolved.conf
DNSOverTLS=opportunistic
DNS=1.1.1.1#cloudflare-dns.com 1.0.0.1#cloudflare-dns.com
```

### DoH — DNS over HTTPS (RFC 8484)

```text
Port 443, HTTPS (often HTTP/2 or HTTP/3). DNS payload as message/dns content type.
```

```bash
# DoH with curl
curl -H 'accept: application/dns-message' \
     'https://1.1.1.1/dns-query?dns=AAABAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE'

# Or with a DoH client
doh-client -domains example.com -upstream-resolver https://1.1.1.1/dns-query

# kdig with DoH
kdig @1.1.1.1 +https example.com
```

DoH bypass: browsers (Firefox, Chrome) ship with built-in DoH that bypasses the OS resolver entirely. This means:

- `cat /etc/resolv.conf` lies — the browser uses its own resolver.
- Network policies that depend on DNS visibility (DNS-based ad blocking, parental controls) break.
- Diagnostics: enable `chrome://net-internals/#dns` view, or temporarily disable browser DoH.

### DoQ — DNS over QUIC (RFC 9250)

```text
Port 853, UDP (QUIC). Faster than DoT (no TCP/TLS handshake), encrypted.
```

```bash
# kdig with DoQ
kdig @1.1.1.1 +quic example.com

# AdGuard's DNS server supports DoQ
kdig @94.140.14.140 +quic example.com
```

## Authoritative Server Errors

### BIND named.conf syntax

```bash
sudo named-checkconf
sudo named-checkconf -z         # also load and check zones
```

Common error:

```text
zone 'example.com/IN': loading from master file db.example failed: file not found
```

= path wrong, or permissions wrong, or SELinux blocking.

```text
zone example.com/IN: NS 'ns1.example.com' has no address records (A or AAAA)
```

= you defined the NS but didn't add glue.

### Zone file errors

```text
zone example.com/IN: loaded serial 2024010101
zone example.com/IN: REFUSED zone transfer due to ACL
zone example.com/IN: SOA serial number is unchanged
```

```bash
named-checkzone example.com /etc/bind/zones/db.example.com
# good output:
zone example.com/IN: loaded serial 2024010101
OK
```

Common zone file mistakes:

```text
$ORIGIN example.com.
$TTL 3600

@   IN  SOA  ns1 admin (
    2024010101 ; serial
    7200       ; refresh
    3600       ; retry
    1209600    ; expire
    3600 )     ; minimum

@   IN  NS   ns1
@   IN  NS   ns2          ← whoops, no trailing dot — becomes ns2.example.com.

ns1 IN  A    192.0.2.1
ns2 IN  A    192.0.2.2
www IN  A    192.0.2.10
www IN  CNAME app          ← ILLEGAL: CNAME and A on same name
```

### dnssec-signzone failures

```text
dnssec-signzone: fatal: missing key: example.com/RSASHA256
```

= the KSK or ZSK private key isn't in the keys directory.

```text
dnssec-signzone: warning: 'example.com/A/93.184.216.34': RRSIG has expired
```

= last signing was more than `expire` seconds ago. Re-sign and reload.

### Tools

```bash
named-checkconf                    # syntax of named.conf
named-checkconf -z                 # also load zones
named-checkzone <zone> <file>      # zone file syntax + integrity
rndc reload                        # reload BIND
rndc flush                         # flush cache (recursor mode)
rndc dumpdb -all                   # dump cache to file
rndc retransfer <zone>             # secondary: force transfer from primary
rndc notify <zone>                 # primary: notify secondaries

# Knot DNS equivalents
knotc reload
knotc zone-status example.com
knotc zone-flush example.com
```

## Cloud-Specific DNS Gotchas

### AWS Route 53

- **ALIAS records** — synthetic A/AAAA records that resolve to the target's IP at query time. Allow CNAME-like behaviour at apex. Free queries when target is an AWS resource.
- **Propagation** — usually <60s, sometimes longer for new zones during initial NS distribution.
- **Health checks** — only relevant for failover routing; don't help with caching downstream.
- **Private hosted zones** — only resolvable from within VPCs you associate.

### Cloudflare

- **Orange-cloud (proxy)** — Cloudflare answers DNS with its own anycast IPs (104.16.0.0/12, 172.64.0.0/13, 2606:4700::/32). The origin IP is hidden.
- **Diagnostics confused by orange-cloud** — `dig www.example.com` returns 104.x; you ping/curl 104.x; you don't know if your origin is even up. Bypass: `--resolve www.example.com:443:<origin-ip>` in curl, or grey-cloud temporarily.
- **CNAME flattening** — Cloudflare resolves the CNAME at the apex internally and returns A.
- **Always Online** — serves a cached copy when origin is down.

### Google Cloud DNS

- **TTL minimum is 0** per record set — useful for fast failover, but caches downstream still apply.
- **Private zones** — split-horizon DNS within a VPC.
- **DNSSEC** — managed signing; KSK rotation requires DS update at registrar.

### Azure DNS

- Per-record TTL.
- Private DNS zones are separate resource type from public.
- No DNSSEC support as of writing (legacy gap).

### GoDaddy / Namecheap / etc.

- TTL handling sometimes ignored — caches stale data longer than declared.
- Some refuse to publish certain TXT formats (DKIM with long quoted strings).
- Web UIs sometimes lose trailing dots silently.

## Common Gotchas — broken→fixed pairs

### 1. Forgetting trailing dot in zone file

```text
# Broken
@   IN  MX  10  mail.example.com    ← no trailing dot
                                       becomes mail.example.com.example.com.
```

```text
# Fixed
@   IN  MX  10  mail.example.com.   ← trailing dot makes it absolute
```

### 2. SOA serial not incremented after change

```text
# Broken
example.com. IN SOA ns admin 2024010100 ...   ← unchanged after edit
```

Secondaries see the old serial, refuse to refresh. New record exists on primary only.

```text
# Fixed
example.com. IN SOA ns admin 2024010101 ...   ← incremented
```

Convention: `YYYYMMDDNN` (year-month-day-revision).

### 3. Old DNS records cached past expected TTL

```text
# Broken: record changed, but resolver returns old IP
$ dig +short www.example.com
93.184.216.34       ← stale; you changed it 5 min ago
```

Even with TTL=60, some resolvers cap minimums (e.g. cap at 5 min) or have buggy expiry.

```bash
# Fixed: check what TTL the resolver actually used
dig www.example.com         # see the TTL counting down
sudo resolvectl flush-caches
# Or query a different resolver
dig @8.8.8.8 www.example.com
```

### 4. /etc/hosts override

```text
# Broken: stale entry from years ago
$ getent hosts example.com
192.168.99.99  example.com    ← from /etc/hosts, not DNS
```

```bash
# Diagnose: compare getent (NSS) to dig (DNS)
getent hosts example.com
dig +short example.com
# Mismatch → check /etc/hosts
grep example.com /etc/hosts
```

### 5. Search-domain infinite suffix

```text
# Broken
search corp.example.com
$ dig www.example.com    ← becomes www.example.com.corp.example.com.example.com...
```

```bash
# Fixed: trailing dot to bypass search
dig www.example.com.        ← absolute; no search applied
```

### 6. Lame delegation

```text
# Broken
example.com. IN NS ns1.someother.org.       ← parent says this
$ dig @ns1.someother.org example.com
;; status: REFUSED                           ← child doesn't host the zone
```

```text
# Fixed: update parent NS records to a server that actually hosts the zone, or
# load the zone on ns1.someother.org.
```

### 7. Missing glue for in-bailiwick NS

```text
# Broken
example.com. IN NS ns1.example.com.
                    ← parent doesn't include A record for ns1.example.com
                    ← chicken-and-egg: can't resolve NS to ask it about its zone
```

```text
# Fixed: add glue at the parent
example.com. IN NS ns1.example.com.
ns1.example.com. IN A 192.0.2.1   ← glue (in ADDITIONAL of parent's referral)
```

### 8. Underscore-prefixed names dropped by old resolvers

```text
# Broken: ancient resolver strips RFC-non-compliant labels
_dmarc.example.com    ← some legacy systems treat _ as illegal
```

```bash
# Diagnose
dig _dmarc.example.com TXT @8.8.8.8        ← works on modern
dig _dmarc.example.com TXT @<old-resolver> ← may fail
```

Modern resolvers all support underscore labels (DKIM, DMARC, SRV, ACME).

### 9. Mail rejection due to missing reverse DNS

```text
# Broken
$ tail /var/log/mail.log
NOQUEUE: reject: ... Client host rejected: cannot find your hostname, [203.0.113.5]
```

```bash
# Fixed: configure PTR for the mail server's IP
$ dig -x 203.0.113.5 +short
mail.example.com.            ← PTR present and matches HELO
```

### 10. SPF lookup limit exceeded

```text
# Broken
v=spf1 include:_spf.gmail.com include:_spf.mailgun.com include:spf.protection.outlook.com include:_spf.salesforce.com -all
                                                                                           ^^^ chain pushes past 10 lookups
$ result: PermError (treated as Fail by many receivers)
```

```bash
# Fixed: flatten and consolidate
# Use an SPF flattening service or replace include: chains with explicit ip4:/ip6: ranges.
v=spf1 ip4:35.190.247.0/24 ip4:64.233.160.0/19 include:spf.protection.outlook.com -all
```

### 11. DNSSEC validator clock skew

```text
# Broken: server clock 4 days behind real time
$ dig +dnssec example.com
;; status: SERVFAIL          ← all RRSIGs look "not yet valid" (or all expired)
```

```bash
# Diagnose
date
chronyc tracking
timedatectl status

# Fixed: re-sync NTP
sudo systemctl restart chrony
# or: sudo ntpdate -s pool.ntp.org
```

### 12. CNAME at apex

```text
# Broken
example.com. IN CNAME www.example.com.   ← illegal; coexists with required SOA/NS
```

```text
# Fixed: use the provider's ALIAS / ANAME / flattening
# Route53: ALIAS to ELB
# Cloudflare: CNAME flattening enabled by default at apex
# Or: A records pointing directly to the target IPs (lose dynamic resolution)
```

### 13. ndots:5 inside Kubernetes pod

```text
# Broken
$ time nslookup google.com
# 5 NXDOMAINs (cluster.local suffixes), then finally google.com.
real  0m0.250s
```

```yaml
# Fixed: lower ndots in Pod spec
dnsConfig:
  options:
    - name: ndots
      value: "1"
```

### 14. AAAA blackhole

```text
# Broken
$ time curl https://example.com
real  1m15.234s              ← waited for kernel TCP timeout on v6
```

```bash
# Fixed: enable Happy Eyeballs in your client; or disable v6 if unworkable
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1
```

### 15. EDNS-stripping middlebox

```text
# Broken: large answers always truncate to 512
$ dig www.example.com +noedns
;; flags: ... TC=1
$ dig +tcp www.example.com   ← works
```

```text
# Fixed: replace/upgrade the middlebox; pin firewall rules to permit OPT records.
```

## Diagnostic Tools

### dig (BIND) — the standard

```bash
dig example.com                    # full output
dig +short example.com             # rdata only
dig +trace example.com             # iterative
dig @8.8.8.8 example.com
dig -x 8.8.8.8                     # reverse PTR
dig -t MX example.com              # specific type
```

### drill (NLnet Labs / ldns)

Alternative to dig with tighter DNSSEC integration.

```bash
drill -D example.com               # DNSSEC chain
drill -T example.com               # trace from root
drill @1.1.1.1 example.com
```

### host — concise

```bash
host example.com
host -t MX example.com
host -a example.com                # ANY (often filtered)
host -v example.com 8.8.8.8
```

### nslookup — interactive

```bash
nslookup example.com
nslookup example.com 8.8.8.8

# Interactive
nslookup
> set type=MX
> example.com
> server 1.1.1.1
> example.com
> exit
```

### kdig (Knot DNS) — DoT/DoH

```bash
kdig @1.1.1.1 example.com
kdig @1.1.1.1 +tls example.com                                  # DoT
kdig @1.1.1.1 +https example.com                                # DoH
kdig @1.1.1.1 +quic example.com                                 # DoQ
kdig @1.1.1.1 +tls +tls-host=cloudflare-dns.com example.com     # SNI
```

### whois — domain registration

```bash
whois example.com                  # registration info
whois -h whois.iana.org example.com
whois 8.8.8.8                      # IP allocation
```

### dnstop — live DNS query monitoring

```bash
sudo dnstop eth0
# Press 1/2/3/etc. to switch views (sources, dests, types, RRtypes...).
```

### tcpdump — raw packets

```bash
sudo tcpdump -i any -n -s0 'port 53'
sudo tcpdump -i any -n -s0 -w /tmp/dns.pcap 'port 53 or port 853'
# View with tshark or Wireshark
tshark -r /tmp/dns.pcap -Y dns
```

### getent hosts — system resolver

```bash
getent hosts example.com
getent ahosts example.com          # all addresses
getent ahostsv4 example.com
getent ahostsv6 example.com
```

### resolvectl — systemd-resolved

```bash
resolvectl status
resolvectl query example.com
resolvectl statistics
resolvectl flush-caches
resolvectl monitor                 # live query stream
```

### dnsperf — performance testing

```bash
# Generate a query file
echo 'example.com A' > queries.txt
echo 'www.example.com A' >> queries.txt

# Run a load test
dnsperf -s 8.8.8.8 -d queries.txt -l 60 -c 10
```

### dnsviz — DNSSEC visualisation

```bash
dnsviz print example.com           # text summary
dnsviz query example.com           # query and analyse
dnsviz graph example.com -O graph.png
```

## Idioms

- "dig is your friend, especially with `+trace` and `+short`."
- "Always test with `@8.8.8.8` and `@1.1.1.1` to rule out resolver-side bugs."
- "Incrementing the SOA serial is the secondary's only signal that data changed."
- "Use TTL of 300s for records you might change soon, 86400s for stable records."
- "DNS propagation = max(authoritative TTL, recursive resolver cache, browser cache, application cache)."
- "If `dig` works but the application doesn't, the application has its own resolver."
- "If `getent hosts` differs from `dig`, check `/etc/hosts` and `/etc/nsswitch.conf`."
- "If `+cd` succeeds and the bare query SERVFAILs, DNSSEC is broken."
- "If UDP times out but TCP works, you have an EDNS or PMTU problem."
- "If TLD says NS X but X says REFUSED, you have lame delegation."
- "If a record doesn't exist where you expect it, check the SOA you're served — you might be on the wrong zone."
- "Anycast means same IP, different server. Don't `dig @1.1.1.1` and assume the answer is consistent across the world."
- "Trailing dots matter. A name without a trailing dot in a zone file is relative to `$ORIGIN`."
- "EDNS extends DNS; middleboxes that don't understand it are why your zone transfers and DNSSEC don't work."

## See Also

- verify
- dns
- troubleshooting/http-errors
- troubleshooting/tls-errors
- troubleshooting/linux-errors

## References

- RFC 1034 — Domain Names: Concepts and Facilities
- RFC 1035 — Domain Names: Implementation and Specification
- RFC 1912 — Common DNS Operational and Configuration Errors
- RFC 2136 — Dynamic Updates in the Domain Name System
- RFC 2181 — Clarifications to the DNS Specification
- RFC 2182 — Selection and Operation of Secondary DNS Servers
- RFC 2308 — Negative Caching of DNS Queries (DNS NCACHE)
- RFC 4033 / 4034 / 4035 — DNSSEC Resource Records and Protocol Modifications
- RFC 4592 — The Role of Wildcards in the Domain Name System
- RFC 4635 — HMAC SHA TSIG Algorithm Identifiers
- RFC 5321 — Simple Mail Transfer Protocol (PTR/MX rules)
- RFC 6891 — Extension Mechanisms for DNS (EDNS(0))
- RFC 7208 — Sender Policy Framework (SPF)
- RFC 7489 — Domain-based Message Authentication, Reporting & Conformance (DMARC)
- RFC 7766 — DNS Transport over TCP — Implementation Requirements
- RFC 7816 — DNS Query Name Minimisation to Improve Privacy (obsoleted by RFC 9156)
- RFC 7858 — Specification for DNS over Transport Layer Security (DoT)
- RFC 7871 — Client Subnet in DNS Queries (ECS)
- RFC 7873 — Domain Name System (DNS) Cookies
- RFC 8305 — Happy Eyeballs Version 2: Better Connectivity Using Concurrency
- RFC 8484 — DNS Queries over HTTPS (DoH)
- RFC 9156 — DNS Query Name Minimisation to Improve Privacy
- RFC 9250 — DNS over Dedicated QUIC Connections (DoQ)
- iana.org/assignments/dns-parameters — RCODEs, RRTYPEs, EDNS option codes
- bind9.readthedocs.io — BIND 9 administrator reference
- knot-dns.cz/documentation — Knot DNS documentation
- nlnetlabs.nl/projects/unbound — Unbound documentation
- dnsviz.net — DNSSEC visualisation
- dnsop working group at IETF — current drafts on DNS operations
