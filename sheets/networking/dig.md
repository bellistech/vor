# dig (Domain Information Groper)

The canonical DNS query tool from BIND — flexible, scriptable, and the de facto standard for inspecting DNS responses, debugging delegation, validating DNSSEC, and probing authoritative or recursive resolvers. Replaces the legacy `nslookup` for any non-trivial DNS troubleshooting.

## Setup

`dig` ships in BIND's utility package. Its presence depends on distro — but the binary is identical across systems.

### Linux / BSD

```bash
# Debian / Ubuntu
sudo apt-get install -y dnsutils
# or on newer Debian / Ubuntu
sudo apt-get install -y bind9-dnsutils

# RHEL / CentOS / Rocky / Alma / Fedora
sudo dnf install -y bind-utils
# or on older RHEL family
sudo yum install -y bind-utils

# Arch / Manjaro
sudo pacman -S bind

# Alpine
sudo apk add bind-tools

# FreeBSD
sudo pkg install bind-tools

# OpenBSD (drill is preinstalled — dig from ports if desired)
sudo pkg_add isc-bind
```

### macOS

`dig` is bundled with macOS — no install required.

```bash
which dig                            # /usr/bin/dig
dig -v                               # report installed BIND version
```

If you need a newer version (for `+yaml`, `+json`, modern DNSSEC):

```bash
brew install bind                    # installs newer BIND tools
brew link --force bind               # may collide with system dig
# or invoke explicitly
/opt/homebrew/opt/bind/bin/dig example.com
```

### Windows

`dig` does not ship by default. Three good options:

```bash
# Option 1: install BIND tools from ISC
choco install bind-toolsonly

# Option 2: install via winget
winget install isc.bind

# Option 3: use the native PowerShell cmdlet
Resolve-DnsName example.com -Type A

# Option 4: use legacy nslookup (limited but available)
nslookup example.com 8.8.8.8
```

### WSL / Cygwin / MSYS2

```bash
# WSL — use Linux package above
sudo apt-get install -y dnsutils

# MSYS2
pacman -S bind-tools

# Cygwin
setup-x86_64.exe -P bind-utils
```

### Verify install

```bash
dig -v                               # version line: DiG 9.18.x ...
dig -h                               # condensed help (BIND 9.18+)
which dig                            # path to binary
```

`dig -v` reports the BIND version. Some flags (e.g. `+yaml`, `+json`) require BIND 9.16+/9.18+.

## Basic Syntax

The command line is positional and forgiving:

```bash
dig [@server] [name] [type] [class] [+queryoptions] [-globaloptions]
```

Defaults if omitted:

- `@server` — the first nameserver listed in `/etc/resolv.conf` (or the system stub resolver)
- `name` — required (no default)
- `type` — `A`
- `class` — `IN`

The canonical `dig example.com` is therefore equivalent to `dig @<resolv.conf-ns> example.com A IN`.

```bash
dig example.com                      # A record via local resolver
dig example.com A                    # explicit type
dig example.com MX                   # mail exchanger
dig @1.1.1.1 example.com             # use Cloudflare resolver
dig @ns1.iana.org example.com NS IN  # explicit server, type, class
```

Argument order is loose — any token starting with `@` is treated as the server, recognised RR type names jump to the type slot, and `IN`/`CH`/`HS` mark the class. So these are equivalent:

```bash
dig example.com MX @8.8.8.8
dig @8.8.8.8 example.com MX
dig MX example.com @8.8.8.8
```

Multiple queries can be batched on one command line — each `name [type] [class]` group becomes a separate query in one invocation:

```bash
dig example.com A example.com AAAA example.com MX
```

Each query produces its own header, sections, and stats block.

## Query Types

`dig` accepts any RR type name registered with IANA. The most common (and a few obscure-but-useful) ones:

```bash
dig example.com A                    # IPv4 host address (RFC 1035)
dig example.com AAAA                 # IPv6 host address (RFC 3596)
dig example.com CNAME                # canonical name alias
dig example.com MX                   # mail exchanger (priority + host)
dig example.com NS                   # delegation nameservers
dig example.com SOA                  # zone start of authority (serial, refresh, retry, expire, minTTL)
dig example.com TXT                  # arbitrary text — SPF, DKIM, verification, _acme-challenge
dig example.com SPF                  # legacy SPF type — deprecated, use TXT (RFC 7208)
dig example.com PTR                  # reverse pointer (mostly via `dig -x`)
dig example.com SRV                  # service location (priority, weight, port, target)
dig example.com CAA                  # certification authority authorization (RFC 8659)
dig example.com NAPTR                # naming authority pointer (used by ENUM, SIP)
dig example.com URI                  # URI mapping (RFC 7553)
dig example.com LOC                  # geographic location (RFC 1876)
dig example.com HINFO                # host info (CPU + OS) — historically refused since RFC 8482
dig example.com OPENPGPKEY           # PGP key publication (RFC 7929)
dig example.com SMIMEA               # S/MIME cert publication (RFC 8162)
```

DNSSEC types:

```bash
dig example.com DS                   # delegation signer (parent → child)
dig example.com DNSKEY               # zone signing keys
dig example.com RRSIG                # signature over RRset
dig example.com NSEC                 # next secure (authenticated denial)
dig example.com NSEC3                # hashed NSEC for opt-out
dig example.com NSEC3PARAM           # NSEC3 hashing parameters
dig example.com CDS                  # child DS for parental sync (RFC 7344)
dig example.com CDNSKEY              # child DNSKEY for parental sync
```

TLS/security types:

```bash
dig _443._tcp.example.com TLSA       # DANE — cert/key publication for TLS
dig user._smimecert.example.com SMIMEA
dig host.example.com SSHFP           # SSH host key fingerprints (RFC 4255)
```

Modern routing/discovery:

```bash
dig example.com SVCB                 # service binding (RFC 9460)
dig example.com HTTPS                # HTTPS service binding (RFC 9460) — used by browsers for ECH/HTTP3
dig example.com TYPE65               # numeric form for HTTPS pre-9.18 dig
```

Wildcard / meta queries:

```bash
dig example.com ANY                  # all cached types (RFC 8482: most recursors return minimal/HINFO)
dig example.com AXFR                 # full zone transfer (only authoritative + ACL-permitted)
dig example.com IXFR=2024010100      # incremental from given serial
```

The canonical "clean output" idiom:

```bash
dig +short example.com TXT           # just the text strings, one per line
```

Numeric type form is always accepted — useful when your `dig` predates a type name:

```bash
dig example.com TYPE257              # equivalent to CAA on old dig
dig example.com TYPE65               # equivalent to HTTPS pre-9.18
```

## dig Flags — Output Control

Output options are introduced with `+` (or `+no` to disable). They are toggleable and combine freely.

```bash
dig +short example.com               # minimal output — just the rdata
dig +noall +answer example.com       # show only the ANSWER section
dig +noall +authority example.com    # show only the AUTHORITY section
dig +noall +additional example.com   # show only the ADDITIONAL section
dig +noall +answer +authority example.com   # mix
```

Cherry-pick what you want by toggling individual sections:

```bash
dig +nocomments example.com          # drop the ;; comment lines between sections
dig +noquestion example.com          # hide the QUESTION section
dig +noauthority example.com         # hide the AUTHORITY section
dig +noadditional example.com        # hide the ADDITIONAL section
dig +nostats example.com             # hide the trailing stats block
dig +nocmd example.com               # hide the leading "; <<>> DiG ... <<>>" line
```

The cleanest possible output:

```bash
dig +nocmd +noall +answer +nocomments example.com
```

Formatting:

```bash
dig +multiline example.com SOA       # split SOA fields onto multiple lines, with field comments
dig +multiline example.com DNSKEY    # break long base64 keys
dig +ttlunits example.com            # TTL as 1h / 2d instead of 3600 / 172800
dig +nottlid example.com             # hide the TTL column entirely (rare)
dig +nottlunits example.com          # raw seconds (default)
dig +rrcomments example.com DNSKEY   # add `; KSK; alg=...` comments
dig +norrcomments example.com        # suppress them
dig +crypto example.com DNSKEY       # show full crypto rdata (default)
dig +nocrypto example.com DNSKEY     # mask cryptographic rdata (compact)
dig +unknownformat example.com A     # output rdata in RFC 3597 generic form
```

Structured output:

```bash
dig +yaml example.com                # YAML-structured response (BIND 9.16+)
dig +json example.com                # JSON-structured response (BIND 9.18+)
dig +zonefile example.com            # render answer in zone-file presentation form
```

`+short` understands a "short option flag" character to choose what to print:

```bash
dig +short example.com A             # just the IP
dig +short example.com NS            # just the NS hostnames
dig +short example.com MX            # priority + host per line
dig +short example.com SOA           # SOA rdata fields on one line
```

You can override which RR field is printed with `+rdflag`:

```bash
dig +nosplit example.com TXT         # don't break long strings onto continuation lines
dig +split=80 example.com TXT        # split base64/hex output every 80 chars
```

Identification:

```bash
dig +cmd example.com                 # echo the original command line (default)
dig +nocmd example.com               # suppress
dig +identify example.com            # print server IP responding to +short
```

## dig Flags — Resolution Control

These shape how the query is sent and which servers are walked.

```bash
dig +recurse example.com             # set RD bit (default — ask for recursion)
dig +norecurse example.com           # clear RD — ask the server to answer locally only
```

Direct authoritative testing — the canonical "bypass cache" idiom:

```bash
dig +norecurse @ns1.example.com example.com SOA
```

If the server returns an answer with `aa` flag, you have authoritative ground truth.

Trace mode — walk delegation from root down:

```bash
dig +trace example.com               # root → TLD → authoritative
dig +trace +nodnssec example.com     # quiet trace (skip DNSKEY/DS dumps)
dig +trace +additional example.com   # show glue at each level
dig +topdown example.com             # legacy alias for +trace
dig +trace example.com NS            # trace specifically for NS records
```

The trace starts from `.` (root) using the dig built-in root hints (or `/etc/named/named.cache` if installed). Each step is a fresh, independent query — the local resolver is bypassed.

Search a delegation for SOA agreement:

```bash
dig +nssearch example.com            # query SOA of every authoritative NS, compare serials
```

This is the canonical "are my secondaries in sync?" test.

Sigchase / DNSSEC chasing:

```bash
dig +sigchase example.com            # request and validate the DNSSEC chain (BIND 9.10+, deprecated)
dig +trusted-key=./trusted-key.key +sigchase example.com   # custom anchor
dig +topdown +sigchase example.com   # top-down chase (default is bottom-up)
```

`+sigchase` is deprecated in modern dig; use `delv` instead.

Transport selection — dig defaults to UDP and falls back to TCP only on a truncated (`tc=1`) reply.

```bash
dig +notcp example.com               # never retry over TCP, even on TC bit (default)
dig +tcp example.com                 # force TCP from the start
dig +vc example.com                  # virtual circuit = TCP (alias)
dig +ignore example.com              # ignore TC bit, don't auto-retry over TCP
dig +keepopen example.com a.com b.com   # reuse single TCP connection across queries
```

Modern transport options:

```bash
dig +tls @1.1.1.1 example.com        # DNS over TLS (DoT, port 853, BIND 9.18+)
dig +https @1.1.1.1 example.com      # DNS over HTTPS (DoH, BIND 9.18+)
dig +https-get @1.1.1.1 example.com  # DoH using GET (default is POST)
dig +tls-ca=./ca.pem @1.1.1.1 example.com   # validate DoT cert against given CA bundle
dig +tls-hostname=cloudflare-dns.com @1.1.1.1 example.com   # SNI / cert-name override
```

Ports:

```bash
dig -p 5353 @127.0.0.1 example.com   # query non-standard port (mDNS, dev resolver)
dig -p 853 +tls @1.1.1.1 example.com # explicit DoT port
```

Address family pinning:

```bash
dig -4 example.com                   # only use IPv4 transport to the resolver
dig -6 example.com                   # only use IPv6 transport
```

(These are flags, not `+options`.)

## dig Flags — DNSSEC

```bash
dig +dnssec example.com              # set the DO bit; ask for RRSIG/NSEC alongside the answer
dig +nodnssec example.com            # clear DO bit (default if your dig wasn't built for DNSSEC)
dig +cdflag example.com              # set Checking Disabled — don't validate at the recursor
dig +nocdflag example.com            # clear CD (default)
dig +adflag example.com              # ask for AD on response (the default; query AD is mostly ceremonial)
dig +noadflag example.com            # clear it
```

The flags returned in the response header are the ones that matter — see "Output Anatomy" below for `aa`, `ad`, `cd`. The canonical "is DNSSEC validating?" test:

```bash
dig +dnssec example.com | grep -E '^;; flags:'
# Look for "ad" — if present, the recursor validated the chain.
```

Manual chain inspection:

```bash
dig . DNSKEY +dnssec
dig com. DS +dnssec
dig example.com. DS +dnssec
dig example.com DNSKEY +dnssec
dig example.com A +dnssec            # check RRSIG covers the A
```

Trust anchor setup for `+sigchase`:

```bash
dig . DNSKEY | grep -E '257 [0-9]+ [0-9]+' > /etc/trusted-key.key
dig +sigchase +trusted-key=/etc/trusted-key.key example.com
```

## dig Flags — Tuning

```bash
dig +timeout=5 example.com           # per-attempt timeout in seconds (default: 5)
dig +tries=3 example.com             # number of attempts before giving up (default: 3)
dig +retry=2 example.com             # number of retries on no answer (different from +tries)
dig +ndots=1 example.com             # min dots in name before treating as FQDN
dig +search example.com              # apply search list from /etc/resolv.conf (default for short names)
dig +nosearch example.com            # treat name as fully-qualified (default for names with dots)
dig +domain=corp.example.com example.com   # override search domain
dig +bufsize=4096 example.com        # advertised UDP EDNS0 buffer size (default 1232 in modern dig)
dig +bufsize=0 example.com           # disable EDNS0 buffer extension (cap at 512)
dig +edns=0 example.com              # enable EDNS0 version 0 (default)
dig +edns=1 example.com              # request EDNS version 1 (very rarely useful)
dig +noedns example.com              # disable EDNS entirely
```

EDNS options:

```bash
dig +nocookie example.com            # don't send EDNS cookies
dig +cookie example.com              # send/use EDNS cookies (default since BIND 9.11)
dig +cookie=0123456789abcdef example.com    # send a specific client cookie
dig +padding=128 example.com         # add EDNS padding to the given block size (RFC 7830)
dig +nopadding example.com           # disable padding
dig +zflag example.com               # set the reserved Z bit in the query header
```

DNS over a SOCKS proxy or specific source:

```bash
dig -b 192.0.2.5 example.com         # bind to specific source IP
dig -b 192.0.2.5#5300 example.com    # source IP + source port
dig -b ::1 example.com               # IPv6 source address
```

## dig Flags — Subnet / Identification

EDNS Client Subnet (ECS) — controversial, used by CDNs to localise responses:

```bash
dig +subnet=1.2.3.0/24 example.com   # tell server "client lives in 1.2.3.0/24"
dig +subnet=0.0.0.0/0 example.com    # explicit empty ECS — request "no subnet" answer
dig +subnet=::/0 example.com         # IPv6 form
dig +nosubnet example.com            # don't include ECS option
```

NSID — server identification (RFC 5001):

```bash
dig +nsid @1.1.1.1 example.com       # ask which Cloudflare anycast node responded
# Response includes:  ;; NSID: ... ("DFW" or "AMS-1234")
```

EDNS Expire (RFC 7314) for secondary cache lifetime:

```bash
dig +expire @ns1.example.com example.com SOA
```

EDNS Key Tag option (RFC 8145) for trust anchor reporting:

```bash
dig +ednsopt=KEY-TAG:0030 . DNSKEY
```

Generic EDNS options:

```bash
dig +ednsopt=10:abcd1234 example.com # send raw EDNS option code 10 with hex data
dig +ednsflags=0x80 example.com      # raw EDNS flags
```

## dig Flags — Class

DNS classes are mostly historical, but the CHAOS class is alive for server diagnostics.

```bash
dig +class=IN example.com            # Internet (default)
dig example.com IN A                 # equivalent positional form
```

CHAOS class — server self-identification queries:

```bash
dig @ns1.example.com version.bind CH TXT    # BIND version string
dig @ns1.example.com hostname.bind CH TXT   # hostname of responding instance
dig @ns1.example.com id.server CH TXT       # standard server ID (RFC 4892)
dig @ns1.example.com authors.bind CH TXT    # historical BIND authors list
```

These are the canonical "what is this server?" queries. Many operators block or scrub them.

Hesiod — historical user/host/group lookup at MIT/CMU:

```bash
dig +class=HS user.passwd HS TXT
```

## Reverse Lookups

The shorthand `-x` reverses an IP into the in-addr.arpa / ip6.arpa hierarchy and queries PTR.

```bash
dig -x 93.184.216.34                 # equivalent to: dig 34.216.184.93.in-addr.arpa PTR
dig -x 93.184.216.34 +short          # short form
dig -x 2606:2800:220:1:248:1893:25c8:1946   # IPv6 — nibble-reversed under ip6.arpa
```

What `-x` actually does on an IPv6 address:

```bash
# 2001:db8::1 expands and reverses to:
# 1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa
dig -x 2001:db8::1
```

If you want explicit control:

```bash
dig 4.3.2.1.in-addr.arpa PTR         # equivalent to dig -x 1.2.3.4
dig 1.0.0.0...8.b.d.0.1.0.0.2.ip6.arpa PTR
```

Reverse lookups against a specific resolver (canonical "what does my ISP think?" check):

```bash
dig -x 1.1.1.1 @8.8.8.8
```

For RFC 1918 / unallocated space, expect NXDOMAIN unless your local resolver overrides:

```bash
dig -x 10.0.0.1                      # likely NXDOMAIN from public resolvers
dig -x 10.0.0.1 @192.168.1.1         # internal resolver may have the PTR
```

## Selecting Servers

The `@server` argument can be an IP, hostname, or IPv6 literal in brackets.

```bash
dig @8.8.8.8 example.com             # Google Public DNS v4
dig @8.8.4.4 example.com             # Google secondary
dig @1.1.1.1 example.com             # Cloudflare 1.1.1.1
dig @1.0.0.1 example.com             # Cloudflare secondary
dig @9.9.9.9 example.com             # Quad9 (DNSSEC + filtering)
dig @149.112.112.112 example.com     # Quad9 secondary
dig @208.67.222.222 example.com      # OpenDNS
dig @64.6.64.6 example.com           # Verisign Public
dig @64.6.65.6 example.com           # Verisign secondary
dig @127.0.0.53 example.com          # systemd-resolved local stub (Linux)
dig @127.0.0.1 example.com           # local resolver / Unbound / dnsmasq
dig @::1 example.com                 # local IPv6 resolver
dig @[2606:4700:4700::1111] example.com   # IPv6 Cloudflare
```

By hostname — dig will resolve the @name with the system resolver first:

```bash
dig @ns1.example.com example.com NS
dig @resolver1.opendns.com example.com
```

The canonical "compare resolvers" diagnostic:

```bash
for ns in 8.8.8.8 1.1.1.1 9.9.9.9 208.67.222.222; do
  echo "== $ns =="
  dig @$ns +short example.com
done
```

A staggered version that compares answers for sanity:

```bash
diff <(dig @8.8.8.8 +short example.com | sort) \
     <(dig @1.1.1.1 +short example.com | sort)
```

## Output Anatomy

A standard `dig example.com` response is structured:

```bash
; <<>> DiG 9.18.24 <<>> example.com
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 26542
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;example.com.            IN  A

;; ANSWER SECTION:
example.com.        300  IN  A   93.184.216.34

;; Query time: 16 msec
;; SERVER: 1.1.1.1#53(1.1.1.1) (UDP)
;; WHEN: Fri Apr 25 12:00:00 UTC 2026
;; MSG SIZE  rcvd: 56
```

Line by line:

- `; <<>> DiG 9.18.24 <<>> example.com` — the dig version + the original command (suppressible with `+nocmd`).
- `;; global options: +cmd` — global options in effect.
- `;; Got answer:` — separator before the response message.
- `;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 26542` — the DNS header (RFC 1035 §4.1.1).
  - `opcode` — `QUERY` (0), `IQUERY` (1, obsolete), `STATUS` (2), `NOTIFY` (4), `UPDATE` (5).
  - `status` — RCODE (see "Status Codes" below).
  - `id` — 16-bit transaction id matching query and response.
- `;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1` — flag bits + section counts.
  - `qr` — Query Response (1 = response, 0 = query).
  - `aa` — Authoritative Answer (server is authoritative for the zone).
  - `tc` — Truncated (response was too large for UDP).
  - `rd` — Recursion Desired (set by client in query; echoed in response).
  - `ra` — Recursion Available (server offers recursion).
  - `ad` — Authenticated Data (DNSSEC validated successfully).
  - `cd` — Checking Disabled (client asked recursor not to validate).
  - `z` — reserved zero bit (must be 0; presence indicates non-conformance).
- `;; OPT PSEUDOSECTION:` — EDNS0 pseudo-record (RFC 6891).
  - `version: 0` — EDNS version.
  - `flags: do` — `do` if DNSSEC OK was set.
  - `udp: 1232` — advertised reassembly buffer.
  - May include `COOKIE`, `NSID`, `CLIENT-SUBNET`, `PADDING` etc.
- `;; QUESTION SECTION:` — what was asked.
  - `;example.com.   IN  A` — name, class, type. Leading `;` because `dig` formats it as a comment.
- `;; ANSWER SECTION:` — RRset(s) directly satisfying the QUESTION.
  - `example.com.  300  IN  A  93.184.216.34` — name, TTL (seconds), class, type, rdata.
- `;; AUTHORITY SECTION:` — NS records for the zone of the answer (or hint if delegation pending).
- `;; ADDITIONAL SECTION:` — glue (A/AAAA for NS hostnames in AUTHORITY), OPT pseudo-RR.
- `;; Query time: 16 msec` — RTT measured by dig.
- `;; SERVER: 1.1.1.1#53(1.1.1.1) (UDP)` — `<addr>#<port>(<@arg>) (transport)`.
- `;; WHEN: Fri Apr 25 12:00:00 UTC 2026` — local clock when the response arrived.
- `;; MSG SIZE  rcvd: 56` — wire size of the response in bytes.

When status is `NOERROR` but ANSWER is empty, look at AUTHORITY — an SOA there means "the name exists but this RR type does not" (NODATA).

When status is `NXDOMAIN`, AUTHORITY usually carries the SOA of the closest enclosing zone with a `negative TTL` you can use to size negative caches.

## Status Codes

The `status:` field in the header is the response RCODE (RFC 1035 §4.1.1, RFC 6895).

```bash
NOERROR    # 0  — query answered (may still have empty ANSWER = NODATA)
FORMERR    # 1  — server thought the query was malformed
SERVFAIL   # 2  — server failure (often DNSSEC validation, broken upstream, or zone misconfig)
NXDOMAIN   # 3  — name does not exist (anywhere in the queried name's hierarchy)
NOTIMP     # 4  — server does not implement the request (e.g. AXFR refused)
REFUSED    # 5  — server is refusing for policy reasons (ACL, recursion not offered)
YXDOMAIN   # 6  — name exists when it shouldn't (Dynamic Update prerequisite)
YXRRSET    # 7  — RRset exists when it shouldn't (Dynamic Update)
NXRRSET    # 8  — RRset doesn't exist when it should (Dynamic Update)
NOTAUTH    # 9  — server not authoritative for zone, or TSIG/SIG(0) failed
NOTZONE    # 10 — name not contained in zone (Dynamic Update)
DSOTYPENI  # 11 — DSO type not implemented
BADVERS    # 16 — EDNS version not supported (also BADSIG for TSIG)
BADKEY     # 17 — TSIG key not recognized
BADTIME    # 18 — TSIG signature out of time window
BADMODE    # 19 — TKEY mode not supported
BADNAME    # 20 — duplicate key name
BADALG     # 21 — TSIG algorithm not supported
BADTRUNC   # 22 — bad truncation
BADCOOKIE  # 23 — bad/missing server cookie
```

What to do for each:

- `NOERROR` + empty ANSWER → look at AUTHORITY for SOA (NODATA — type doesn't exist).
- `NXDOMAIN` → look at AUTHORITY for the closest enclosing zone; check spelling, delegation, wildcards.
- `SERVFAIL` → the recursor failed. Try `+cd` to bypass DNSSEC validation, query authoritative servers directly, check zone health.
- `REFUSED` → ACL or non-recursive server. Try a different resolver, or for AXFR check `allow-transfer`.
- `FORMERR` → broken server or unsupported EDNS option. Try `+noedns` or `+nocookie`.
- `NOTIMP` → server doesn't speak this opcode/feature (often AXFR off public resolvers).
- `BADCOOKIE` → DNS cookie mismatch. Most clients retry automatically.

## Trace Mode Deep

`+trace` is the canonical "I think delegation is broken" diagnostic.

```bash
dig +trace example.com
```

What it actually does:

1. Reads root nameserver hints (built into dig binary, or from `named.cache` if present).
2. Issues a non-recursive query for `example.com NS` to a root server.
3. Receives a referral to `com.` NS records.
4. Issues a non-recursive query to a `com.` NS for `example.com NS`.
5. Receives a referral to `example.com.` NS records.
6. Issues a non-recursive query to an `example.com.` NS for the original record type.

Because each step is non-recursive, the local resolver and recursor caches are bypassed entirely. You see ground truth from the delegation chain.

```bash
dig +trace +dnssec example.com       # add DNSSEC chain at every level
dig +trace +nodnssec example.com     # quiet — skip DS/DNSKEY chatter
dig +trace example.com NS            # trace specifically for NS records
dig +trace -t MX example.com         # trace MX
dig +trace +additional example.com   # show glue at each level
```

Common trace pathologies:

- "no servers could be reached" mid-trace → outbound port 53 blocked or NS hostname unresolvable.
- Different NS sets at parent vs child (`com.` says `[a,b].ns.example.com`, but `example.com` SOA says `[a,c].ns.example.com`) → lame delegation. Fix at the registrar.
- DS at parent doesn't match DNSKEY at child → DNSSEC chain broken; re-roll DS records.
- Slow trace stalling at TLD → TLD-side slow / rate-limiting / GeoDNS issue.

For modular debugging, manual stepping:

```bash
dig +norecurse @a.root-servers.net example.com NS
dig +norecurse @a.gtld-servers.net example.com NS
dig +norecurse @ns1.example.com example.com A
```

## AXFR Zone Transfer

`AXFR` requests a full zone copy. It only succeeds against a server that:

1. Is authoritative for the zone, AND
2. Has an `allow-transfer` ACL that includes your IP (or no ACL at all — uncommon).

```bash
dig @ns1.example.com example.com AXFR              # full zone transfer
dig @ns1.example.com example.com IXFR=2024010100   # incremental from given serial
dig @ns1.example.com -y hmac-sha256:keyname:base64key example.com AXFR   # with TSIG
```

Output is presented as an answer section, ordered as the zone — start with SOA, then all RRs, ending with the same SOA.

```bash
dig @ns1.example.com example.com AXFR | head -20
dig @ns1.example.com example.com AXFR > example.com.zone
```

Refused transfer typically returns `Transfer failed.` with `; Transfer failed.` and status `REFUSED` — expected on every public-facing authoritative server in 2026, since open AXFR is a textbook information disclosure.

For internal testing where you control the server:

```bash
# in named.conf
zone "example.com" {
  type master;
  file "example.com.zone";
  allow-transfer { 192.0.2.0/24; };
};
```

Then:

```bash
dig @ns1.example.com -b 192.0.2.42 example.com AXFR
```

For TSIG-secured transfers:

```bash
dnssec-keygen -a HMAC-SHA256 -b 256 -n HOST transfer-key
KEY=$(awk '/Key:/{print $2}' Ktransfer-key.+163+*.private)
dig @ns1.example.com -y "hmac-sha256:transfer-key:$KEY" example.com AXFR
```

## dig +short — Scripting

`+short` is the canonical scriptable form. It strips everything except the rdata of the ANSWER section.

```bash
dig +short example.com               # 93.184.216.34
dig +short example.com MX            # 0 .
dig +short example.com NS            # a.iana-servers.net.
                                     # b.iana-servers.net.
dig +short example.com TXT           # "v=spf1 -all"
```

Bash idioms:

```bash
IP=$(dig +short example.com | head -1)
[[ -z "$IP" ]] && { echo "no IPv4"; exit 1; }

# Robust check — bail on no answer or NXDOMAIN
if ! dig +short example.com | grep -q '.'; then
  echo "DNS lookup failed for example.com" >&2
  exit 1
fi

# Get the first MX target (skip the priority)
PRIMARY_MX=$(dig +short example.com MX | sort -n | head -1 | awk '{print $2}')

# Collect all A and AAAA addresses
mapfile -t IPS < <(dig +short example.com A; dig +short example.com AAAA)
```

`+short` returns one rdata per line; multiple records produce multiple lines. For a single value, use `head -1` after sorting if order matters:

```bash
dig +short example.com A | sort | head -1
dig +short example.com MX | sort -n | awk 'NR==1 {print $2}'
```

Exit status: `dig` returns 0 on a successful protocol exchange even if the RCODE is `NXDOMAIN` or the ANSWER is empty. Always check the output, not the exit code, unless you want "couldn't reach any server".

```bash
# Bad — dig succeeds even on NXDOMAIN
dig +short notarealdomain.example && echo "found"       # prints "found"

# Good
[[ -n "$(dig +short notarealdomain.example)" ]] && echo "found"
```

## dig +noall +answer — Display Only Answer Section

Cleaner than full output, more inspectable than `+short` (you keep TTL, type, class):

```bash
dig +noall +answer example.com
# example.com.        300  IN  A   93.184.216.34
```

Combine with `+multiline` for big records:

```bash
dig +noall +answer +multiline example.com SOA
# example.com.        3600 IN SOA (
#                             ns.icann.org.    ; primary master
#                             noc.dns.icann.org. ; rname (responsible)
#                             2024010100  ; serial
#                             7200        ; refresh (2 hours)
#                             3600        ; retry (1 hour)
#                             1209600     ; expire (2 weeks)
#                             3600        ; minimum (1 hour)
#                             )
```

For DNSKEY / DS readability:

```bash
dig +noall +answer +multiline example.com DNSKEY
```

Parseable but human-readable script idiom:

```bash
dig +noall +answer +nocomments example.com | awk '{print $1, $5}'
```

## Diagnostic Workflows

A toolbox of recipes — each runnable as-is.

### "Is my zone authoritative?"

```bash
dig @auth_ns1.example.com example.com SOA +norecurse
# Look for `aa` flag in response. Match SOA serials across NS:
dig +nssearch example.com
```

### "Is delegation correct?"

```bash
# Compare parent's view vs child's view
dig @a.gtld-servers.net example.com NS +norecurse +noall +authority
dig @ns1.example.com example.com NS +norecurse +noall +answer
diff <(dig @a.gtld-servers.net example.com NS +norecurse +short | sort) \
     <(dig @ns1.example.com example.com NS +short | sort)
```

### "Is DNSSEC validating?"

```bash
dig +dnssec example.com | grep -E '^;; flags:'
# Need "ad" in flags for validated.
# Then sanity-check chain:
delv +rtrace example.com
```

### "What's the TTL of this record?"

```bash
dig example.com | awk '/^[^;]/ && /IN[[:space:]]+A/ { print $2 }'
# Each subsequent query against the same recursor decrements until 0, then refreshes.
```

### "Compare resolvers"

```bash
for ns in 8.8.8.8 1.1.1.1 9.9.9.9; do
  printf "%-15s %s\n" "$ns" "$(dig @$ns +short example.com)"
done
```

### "Is this MX correctly configured?"

```bash
dig +short example.com MX
# Each MX target must resolve to A/AAAA — no CNAME, per RFC 5321:
for host in $(dig +short example.com MX | awk '{print $2}'); do
  echo -n "$host "
  dig +short "$host" A | head -1
done
```

### "Show all records for a domain"

```bash
# ANY is unreliable post-RFC 8482; common types one by one is more honest:
for t in A AAAA NS MX TXT SOA CAA CNAME; do
  echo "== $t =="
  dig +noall +answer example.com $t
done
```

### "Detailed DNSSEC chain"

```bash
dig +trace +dnssec example.com
# Or, more focused:
delv +rtrace example.com
delv +cd example.com                 # what the chain says without trust anchor
```

### "Cache TTL behaviour"

```bash
T1=$(dig example.com | awk '/^example\.com\./ {print $2; exit}')
sleep 5
T2=$(dig example.com | awk '/^example\.com\./ {print $2; exit}')
echo "TTL went from $T1 to $T2 — expected delta ≈ 5"
```

### "Authoritative answer?"

```bash
dig @ns1.example.com example.com SOA | grep -E '^;; flags:'
# Look for `aa`. From a public recursor (Cloudflare/Google), `aa` is rarely set.
```

### "Recursive answer?"

```bash
dig @1.1.1.1 example.com | grep -E '^;; flags:'
# Look for `ra` — recursion available. Public resolvers always set this.
```

### "Did the resolver lie? Cross-check authoritative."

```bash
NS=$(dig +short example.com NS | head -1)
echo "Authoritative says: $(dig @$NS +short example.com)"
echo "Recursor says:      $(dig @1.1.1.1 +short example.com)"
```

### "What does the registrar's parent zone say?"

```bash
TLD_NS=$(dig +short com NS | head -1)
dig @$TLD_NS example.com NS +norecurse +noall +authority
```

### "Is my SPF valid?"

```bash
dig +short example.com TXT | grep -E '^"v=spf1' | head -1
# Lookup count must be < 10 per RFC 7208.
```

### "Is my DMARC valid?"

```bash
dig +short _dmarc.example.com TXT
```

### "Is DKIM published for selector s1?"

```bash
dig +short s1._domainkey.example.com TXT
```

### "What does my CAA say?"

```bash
dig +noall +answer example.com CAA
# Should list `0 issue "letsencrypt.org"` etc. Check empty CAA = any CA may issue.
```

### "Find DNS hijacking / poisoning"

```bash
# Compare an authoritative answer with a public recursor:
dig @ns1.example.com +short example.com A
dig @1.1.1.1 +short example.com A
# If different, suspect cache poisoning, DNS hijack, split-horizon, or stale slave.
```

### "Detect open recursive resolver"

```bash
# From outside the network:
dig @target-ip example.com +norecurse
# If status NOERROR with an answer, the server is acting as an open recursor
# (a security/DDoS-amplification hazard).
```

## dig vs nslookup vs host

| Tool       | Status                  | Strengths                         | Weaknesses                                |
|------------|-------------------------|------------------------------------|-------------------------------------------|
| `dig`      | Maintained (BIND)       | Verbose, scriptable, every flag    | Verbose by default — needs `+short`       |
| `nslookup` | Legacy, dual-mode       | Ubiquitous (Windows includes it)   | Inconsistent across platforms; stale      |
| `host`     | Maintained (BIND)       | Compact one-line output            | Limited control of flags / output format  |

The canonical recommendation: **use `dig` for any non-trivial DNS work**. `nslookup` is fine for a quick PTR on Windows, `host` is fine for a one-liner — but `dig` is the one true tool for delegation, DNSSEC, AXFR, and EDNS debugging.

```bash
# These are roughly equivalent for "what's the IP of example.com?":
dig +short example.com
host example.com | awk '/has address/ {print $4; exit}'
nslookup example.com 2>/dev/null | awk '/^Address: / && NR>2 {print $2; exit}'
```

## drill

`drill` is the alternative dig from NLnet Labs (LDNS package). Same purpose, slightly different syntax, often present where `dnsutils` isn't.

```bash
sudo apt-get install -y ldnsutils    # Debian/Ubuntu
sudo dnf install -y ldns-utils       # RHEL/Fedora
sudo apk add ldns-tools              # Alpine
brew install ldns                    # macOS
```

Common equivalences:

```bash
drill example.com                    # ≈ dig example.com
drill -t example.com                 # ≈ dig +trace example.com
drill -D example.com                 # ≈ dig +dnssec example.com
drill -DT example.com                # ≈ dig +trace +dnssec — compact tracing
drill -k anchor.key example.com      # validate against a trust anchor
drill -x 1.1.1.1                     # reverse PTR
drill -p 5353 @127.0.0.1 example.com # custom port
```

`drill -DT` is the canonical "compact full chain trace" idiom for ops who don't want dig's verbose `+trace`.

## delv

`delv` (DNSSEC Look-aside Validator) ships with BIND 9.10+. It is the proper modern replacement for `dig +sigchase` — performs full DNSSEC validation client-side using a trust anchor (typically the IANA root anchor from `/etc/bind/bind.keys`).

```bash
delv example.com                     # validate full DNSSEC chain for A
delv example.com DNSKEY              # show + validate DNSKEY
delv +rtrace example.com             # show resolution path while validating
delv +mtrace example.com             # message-level trace
delv +vtrace example.com             # validation-level trace
delv +cd example.com                 # set CD bit — see what arrives without validation
delv +nodnssec example.com           # disable DNSSEC processing entirely
delv -a /etc/bind/bind.keys example.com   # custom anchor file
delv +unknownformat example.com      # generic RR format
delv @1.1.1.1 example.com            # send via specific recursor (still validates locally)
```

Output ends with `; fully validated` on success or an error like `resolution failed: ...`. The latter is your DNSSEC bug.

## Common Errors and Fixes

```bash
;; connection timed out; no servers could be reached
```
Resolver unreachable. Check `/etc/resolv.conf`, network, firewall. Try `dig @1.1.1.1 example.com` to bypass the system resolver — if that works, fix `resolv.conf`.

```bash
;; reply from unexpected source: 198.51.100.5#53, expected 192.0.2.1#53
```
Spoofed reply or NAT/load-balancer rewriting source. Common on misconfigured firewalls. `dig` rejects the reply for safety. Investigate: `tcpdump -ni any port 53 and host 192.0.2.1`.

```bash
;; ->>HEADER<<- opcode: QUERY, status: SERVFAIL, id: 12345
```
Server-side failure. Causes:

- DNSSEC validation failed at the recursor — try `+cd` to confirm.
- Authoritative server unreachable / lame.
- Misconfigured zone.
- Rate-limited at the resolver.

Diagnostic:

```bash
dig +cd example.com                  # bypass DNSSEC at recursor
dig @ns1.example.com example.com    # ask authoritative directly
dig +trace example.com               # walk delegation
```

```bash
;; ANSWER SECTION:
(empty)
;; AUTHORITY SECTION:
example.com.   3600 IN SOA ns.icann.org. ...
```
With status `NOERROR` — this is **NODATA**. The name exists, but the requested RR type does not. Common when querying AAAA on an A-only host or TXT on a domain with no TXT.

```bash
;; Truncated, retrying in TCP mode.
```
Response exceeded the UDP buffer (default 1232 in modern dig). Informational — dig will retry over TCP. To suppress retry: `+ignore`. To enlarge UDP buffer: `+bufsize=4096`.

```bash
;; Got bad packet: bad label type
```
Malformed wire data. Either dig hit a buggy resolver or the wire was corrupted in transit. Try a different transport (`+tcp`) or different resolver.

```bash
; communications error to 192.0.2.1#53: timed out
```
A specific server timed out — dig will try the next from `/etc/resolv.conf`. If all timeout, see "connection timed out" above.

```bash
;; warning: recursion requested but not available
```
You queried a non-recursive (authoritative-only) server with `+recurse`. Either query the authoritative for a name it serves, or use a recursor (`@1.1.1.1`).

```bash
status: REFUSED
```
Server refuses to answer. Common causes: queried a recursor outside its allowed clients, attempted AXFR without permission, hit a server policy block. Try a public recursor (`@1.1.1.1`) or check the authoritative's ACL.

```bash
;; AXFR query: Transfer failed.
```
The server refused or aborted the zone transfer. Almost always intentional (`allow-transfer` ACL). Don't retry — it's policy, not a bug.

```bash
;; bad cookie - retry
```
EDNS Cookie mismatch — usually transient, dig will retry. If persistent, try `+nocookie`.

## Common Gotchas

Each item: the broken pattern, then the fix.

### Cached vs authoritative drift

Bad — querying a public recursor and assuming it reflects the authoritative state immediately after a change:

```bash
dig +short example.com               # may return stale cached answer
```

Fixed — explicitly query an authoritative NS:

```bash
NS=$(dig +short example.com NS | head -1)
dig +short @$NS example.com
```

### `ANY` query expecting all records

Bad — assuming `ANY` returns every RR type:

```bash
dig example.com ANY                  # returns minimal HINFO since RFC 8482 (Cloudflare/Google)
```

Fixed — query individual types or use AXFR if authoritative + permitted:

```bash
for t in A AAAA NS MX TXT SOA CAA HTTPS; do
  dig +noall +answer example.com $t
done
```

### UDP truncation surprise

Bad — querying a name with a huge response (e.g. a DKIM TXT or a DNSSEC-signed zone) over single-packet UDP and getting truncation:

```bash
dig example.com TXT                  # ;; Truncated...
```

Fixed — force TCP or raise the EDNS buffer:

```bash
dig +tcp example.com TXT
dig +bufsize=4096 example.com TXT
```

### `+short` hides multiple values

Bad — reading the first `+short` line and assuming it's the only answer:

```bash
IP=$(dig +short example.com | head -1)
# But example.com may have multiple A records — IP only gets one.
```

Fixed — inspect full output, or process all lines:

```bash
mapfile -t IPS < <(dig +short example.com)
echo "Got ${#IPS[@]} addresses: ${IPS[*]}"
```

### MX hostname extraction

Bad — `dig example.com MX` shows priorities + targets, but you only want hostnames:

```bash
dig example.com MX
# 10 mx1.example.com.
# 20 mx2.example.com.
```

Fixed — short + awk:

```bash
dig +short example.com MX | awk '{print $2}'
```

### Split-horizon DNS

Bad — running `dig` on the office VPN, getting an internal answer, and assuming it's what the rest of the world sees:

```bash
dig +short example.com               # 10.0.0.5  (internal)
```

Fixed — always test from an external resolver during DNS migrations:

```bash
dig @1.1.1.1 +short example.com      # 93.184.216.34 (external)
diff <(dig +short example.com) <(dig @1.1.1.1 +short example.com)
```

### Trailing dot omission

Bad — `dig example.com.com` accidentally because `+search` appended a search domain:

```bash
echo "search example.com" >> /etc/resolv.conf
dig www                              # actually queries www.example.com
```

Fixed — add the trailing dot to fully-qualify, or use `+nosearch`:

```bash
dig www.                             # rooted name
dig +nosearch www                    # disables search list
```

### `+trace` with a broken local resolver

Bad — `dig +trace` failing because dig still uses the local resolver to resolve the @-arg or NS hostnames:

```bash
dig +trace example.com               # connection timed out at root
```

Fixed — either fix `resolv.conf` first, or pass an explicit IPv4/IPv6 resolver:

```bash
echo "nameserver 1.1.1.1" | sudo tee /etc/resolv.conf
dig +trace example.com
```

### EDNS / cookie incompatibility with old servers

Bad — querying an ancient authoritative server that chokes on EDNS:

```bash
dig @legacy-ns.example.com example.com   # FORMERR
```

Fixed — disable EDNS or cookies:

```bash
dig +noedns @legacy-ns.example.com example.com
dig +nocookie @legacy-ns.example.com example.com
```

### IPv6 disabled but querying AAAA

Bad — your kernel/network has no IPv6, you hit a v6-only resolver hostname:

```bash
dig @resolver.example.com example.com    # times out trying ::1 / AAAA
```

Fixed — pin to IPv4 or use IP literals:

```bash
dig -4 @resolver.example.com example.com
dig @1.1.1.1 example.com
```

### Old `dig` lacking new RR types

Bad — `dig example.com HTTPS` on BIND 9.10 returns generic `TYPE65` rdata:

```bash
dig example.com HTTPS                # rdata not parsed
```

Fixed — upgrade dig (BIND 9.18+), or fall back to numeric types:

```bash
dig example.com TYPE65 +unknownformat
```

### `dig +trace` fingerprinting / rate limiting

Bad — many `+trace` runs in a tight loop trigger TLD or root rate-limiting:

```bash
for i in $(seq 1 1000); do dig +trace example.com; done   # SERVFAIL after a while
```

Fixed — sleep, or query the authoritative directly once you have it cached:

```bash
NS=$(dig +short example.com NS | head -1)
for i in $(seq 1 1000); do dig +short @$NS example.com; done
```

### CNAME on apex

Bad — having a `CNAME` at the zone apex:

```bash
example.com.    300 IN CNAME target.elb.amazonaws.com.
```

This violates RFC 1034 §3.6.2 and breaks SOA/MX/NS coexistence. Many resolvers tolerate it (Cloudflare CNAME flattening, AWS Route 53 ALIAS), but it's not portable.

Fixed — use the registrar/provider's flattening / ALIAS / ANAME equivalent, or switch to A records:

```bash
dig example.com A                    # provider serves flattened A
```

### Wildcard masking

Bad — assuming `*.example.com` wildcard means every subdomain — but a more specific record overrides it:

```bash
dig random.example.com               # matches *.example.com
dig www.example.com                  # explicit record overrides wildcard
```

Fixed — list and inspect:

```bash
dig +short '*.example.com'
dig +short www.example.com
```

(Direct wildcard query rarely works — wildcards are synthesised on lookup.)

## Performance Tips

Fast-fail probes for monitoring:

```bash
dig +tries=1 +timeout=2 +short @1.1.1.1 example.com
```

Bulk parallel queries:

```bash
# GNU parallel
parallel -j 50 'dig +short {} A' :::: domains.txt

# xargs
xargs -n1 -P50 -I{} dig +short {} A < domains.txt
```

Reduce resolver chatter:

```bash
dig +nocookie example.com            # for old servers that mishandle cookies
dig +bufsize=1232 example.com        # match DNS-flag-day default
dig +keepopen +tcp example.com a.com b.com c.com   # one TCP connection
```

Skip retries on a single test:

```bash
dig +tries=1 example.com
```

Pre-fetch + cache (in scripts that run dig many times for the same name):

```bash
# Resolve once, cache, then reuse
NS_IPS=$(dig +short example.com NS | xargs -I{} dig +short {} | sort -u)
for ip in $NS_IPS; do
  dig +short @$ip example.com SOA &
done
wait
```

For very large zones (`AXFR` of 1M+ records):

```bash
dig +tcp +bufsize=65535 @ns1.example.com example.com AXFR > zone.txt
```

## Idioms

The recurring patterns. Memorise these:

```bash
# 1. Quick lookup, scriptable
dig +short example.com

# 2. Clean human output
dig +noall +answer example.com

# 3. Full chain debug
dig +trace +dnssec example.com

# 4. Authoritative ground truth (bypass cache)
dig @ns1.example.com example.com +norecurse

# 5. Compare resolvers
for ns in 8.8.8.8 1.1.1.1 9.9.9.9; do dig @$ns +short example.com; done

# 6. Reverse with compact output
dig -x 1.1.1.1 +short

# 7. Server self-identification
dig @1.1.1.1 id.server CH TXT

# 8. DNSSEC validation check
dig +dnssec example.com | grep -E '^;; flags:.* ad'

# 9. Bulk NS sync check
dig +nssearch example.com

# 10. EDNS / DoT / DoH probes
dig +tls @1.1.1.1 example.com
dig +https @1.1.1.1 example.com

# 11. Compact full-chain trace via drill
drill -DT example.com

# 12. Modern DNSSEC validation via delv
delv +rtrace example.com

# 13. Get the SOA serial
dig +short example.com SOA | awk '{print $3}'

# 14. List all NS targets and IPs
dig +short example.com NS | xargs -I{} dig +short {}

# 15. "Is recursion offered?"
dig @resolver-ip example.com | grep -E '^;; flags:.* ra'

# 16. "Did the recursor validate DNSSEC?"
dig +dnssec example.com | grep -E '^;; flags:.* ad'

# 17. Find the smallest TTL in a zone (informational)
dig @ns1.example.com example.com AXFR | awk '$3=="IN" {print $2}' | sort -n | head -1

# 18. Test EDNS Client Subnet impact
dig +subnet=8.8.8.8/24 @1.1.1.1 example.com
dig +subnet=192.0.2.0/24 @1.1.1.1 example.com
```

## Tips

- `+short` is your friend in shell scripts. Always pair with explicit error checking, since dig exits 0 even on NXDOMAIN.
- `+trace` bypasses every cache. Always reach for it before suspecting your code — DNS is usually the culprit.
- `+noall +answer` is the cleanest "show me what I asked about" form, with TTL preserved.
- `dig @authoritative-ns ...` is the only way to test changes immediately. Public recursors will lie to you for the duration of the previous TTL.
- TTL counts down as you query the same recursor. A TTL of 300 that drops to 290 then 280 means the recursor cached it; a stable TTL means each query re-fetched.
- `dig` exit codes: 0 = success (any RCODE), 8 = couldn't open batch file, 9 = no reply received, 10 = internal error. Don't rely on exit codes for "name resolved".
- macOS bundled `dig` lags BIND mainline. For modern features (`+yaml`, `+json`, DoT, DoH, HTTPS RR), install `bind` from Homebrew.
- For one-shot lookups in a CI pipeline, prefer `getent ahosts example.com` (uses NSS) when you want the same answer the application would get.
- `dig +trace` is great for delegation, but it cannot diagnose stub-resolver bugs. For that, run `strace -f -e trace=network` on the failing process.
- DNSSEC validation failures are silent in `+short` — always inspect the header flags or use `delv`.
- The order of `+options` doesn't matter except that later options override earlier ones (`+all +noall +answer` shows only the answer).
- Leading dot in a name to dig (`. NS`) means "the root zone".
- Names without a trailing dot are subject to the search list. `dig www` may become `www.corp.example.com` if `/etc/resolv.conf` has `search corp.example.com`.

## See Also

- dns, tls, openssl, polyglot, bash

## References

- man pages: `man dig`, `man delv`, `man drill`, `man named.conf`, `man resolv.conf`
- BIND 9 Administrator Reference Manual — https://bind9.readthedocs.io/
- ISC dig manpage — https://bind9.readthedocs.io/en/latest/manpages.html#dig
- ISC delv manpage — https://bind9.readthedocs.io/en/latest/manpages.html#delv
- Cricket Liu & Paul Albitz, "DNS and BIND" (5th ed.), O'Reilly, 2006 — the canonical reference
- Cricket Liu, "DNS and BIND on IPv6", O'Reilly, 2011
- RFC 1034 — Domain Names: Concepts and Facilities
- RFC 1035 — Domain Names: Implementation and Specification
- RFC 1876 — LOC RR
- RFC 1995 — IXFR
- RFC 1996 — NOTIFY
- RFC 2136 — DNS Update
- RFC 2181 — Clarifications to the DNS Specification
- RFC 2308 — Negative Caching of DNS Queries
- RFC 2782 — SRV RR
- RFC 2845 — TSIG (revised by RFC 8945)
- RFC 3596 — DNS Extensions for IPv6 (AAAA)
- RFC 3597 — Handling of Unknown DNS Resource Record Types
- RFC 4033 — DNSSEC Introduction and Requirements
- RFC 4034 — DNSSEC Resource Records
- RFC 4035 — DNSSEC Protocol Modifications
- RFC 4255 — SSHFP RR
- RFC 4892 — Requirements for a Mechanism Identifying a Name Server Instance (id.server)
- RFC 5001 — DNS Name Server Identifier (NSID)
- RFC 5155 — DNSSEC Hashed Authenticated Denial of Existence (NSEC3)
- RFC 5395 / 6195 — DNS IANA Considerations
- RFC 5936 — DNS Zone Transfer Protocol (AXFR)
- RFC 6376 — DKIM Signatures
- RFC 6698 — DNS-Based Authentication of Named Entities (DANE / TLSA)
- RFC 6891 — EDNS(0)
- RFC 6895 — DNS IANA Considerations
- RFC 7208 — SPF (deprecates the SPF RR type)
- RFC 7344 — Automating DNSSEC Delegation Trust Maintenance (CDS/CDNSKEY)
- RFC 7553 — URI Resource Record
- RFC 7766 — DNS Transport over TCP
- RFC 7830 — EDNS(0) Padding Option
- RFC 7858 — DNS over TLS (DoT)
- RFC 7929 — OPENPGPKEY
- RFC 8094 — DNS over DTLS
- RFC 8145 — Signaling Trust Anchor Knowledge in DNSSEC (Key Tag)
- RFC 8162 — SMIMEA
- RFC 8482 — Providing Minimal-Sized Responses to DNS Queries with QTYPE=ANY
- RFC 8484 — DNS Queries over HTTPS (DoH)
- RFC 8499 — DNS Terminology
- RFC 8659 — Certification Authority Authorization (CAA)
- RFC 8945 — TSIG (current)
- RFC 9460 — Service Binding and Parameter Specification via the DNS (SVCB / HTTPS)
- IANA DNS Parameters — https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml
- IANA Root Zone Database — https://www.iana.org/domains/root/db
- DNS Flag Day 2020 — https://dnsflagday.net/2020/
- ICANN ITHI Project (DNSSEC Adoption Metrics) — https://ithi.research.icann.org/
