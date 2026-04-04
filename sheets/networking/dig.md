# dig (DNS Lookup)

DNS query tool from BIND — the standard for flexible, detailed DNS lookups.

## Basic Queries

### Query a domain
```bash
dig example.com                       # default: A record
dig example.com A                     # explicit A record
dig example.com AAAA                  # IPv6 address
dig example.com MX                    # mail exchange
dig example.com NS                    # name servers
dig example.com TXT                   # TXT records (SPF, DKIM, etc.)
dig example.com CNAME                 # canonical name
dig example.com SOA                   # start of authority
dig example.com ANY                   # all records (some servers refuse)
```

### Short output
```bash
dig +short example.com                # just the answer, no fluff
dig +short example.com MX             # short MX records
dig +short example.com NS             # short NS records
```

## Specific Server

### Query a particular DNS server
```bash
dig @8.8.8.8 example.com             # Google Public DNS
dig @1.1.1.1 example.com             # Cloudflare
dig @ns1.example.com example.com     # authoritative server directly
dig @127.0.0.1 example.com           # local resolver
```

## Reverse Lookups

### PTR records
```bash
dig -x 93.184.216.34                  # reverse DNS lookup
dig -x 93.184.216.34 +short           # short reverse
dig -x 2606:2800:220:1:248:1893:25c8:1946  # IPv6 reverse
```

## Tracing and Debugging

### Full delegation trace
```bash
dig +trace example.com                # trace from root servers down
dig +trace +nodnssec example.com      # trace without DNSSEC noise
```

### Verbose output control
```bash
dig +noall +answer example.com        # answer section only
dig +noall +authority example.com     # authority section only
dig +noall +additional example.com    # additional section only
dig +noall +answer +authority example.com  # answer + authority
dig +stats example.com               # query time and server info
```

## DNSSEC

### Verify DNSSEC
```bash
dig +dnssec example.com               # request DNSSEC records
dig +dnssec +cd example.com           # check disabled (see unvalidated)
dig example.com DNSKEY                 # show DNSSEC keys
dig example.com DS                     # delegation signer
dig example.com RRSIG                  # DNSSEC signatures
dig +sigchase example.com             # chase DNSSEC chain (if supported)
```

## Zone Transfers

### AXFR
```bash
dig @ns1.example.com example.com AXFR   # full zone transfer (if allowed)
dig @ns1.example.com example.com IXFR=2024010100  # incremental from serial
```

## Batch and Scripting

### Multiple queries
```bash
dig example.com +short A
dig example.com +short AAAA
dig example.com +short MX

# Batch mode from file
dig -f queries.txt                     # one domain per line
```

### Control output for scripts
```bash
dig +noall +answer +nocomments example.com       # clean parseable output
dig +noall +answer example.com | awk '{print $5}' # extract just the value
dig +short example.com || echo "DNS lookup failed"
```

## SRV and Other Records

### Service records
```bash
dig _sip._tcp.example.com SRV         # SIP service
dig _http._tcp.example.com SRV        # HTTP service
dig _ldap._tcp.dc._msdcs.example.com SRV  # AD domain controller
```

### CAA records
```bash
dig example.com CAA                    # certificate authority authorization
```

### NAPTR
```bash
dig example.com NAPTR                  # naming authority pointer
```

## Output Format

### Control sections
```bash
dig +noall +answer example.com        # just answers
dig +nocmd +noall +answer example.com # even cleaner (no dig version line)
dig +noall +answer +ttlunits example.com  # human-readable TTLs (1h vs 3600)
```

### JSON output (BIND 9.11+)
```bash
dig +json example.com                 # JSON format output
```

## Tips

- `+short` is your go-to for quick checks and scripts
- `+trace` bypasses your local resolver and follows the delegation chain — essential for debugging propagation
- `+noall +answer` gives clean output without the noise of authority/additional sections
- `dig @authoritative-ns domain` tests the source of truth directly, bypassing caches
- TTL in the output tells you how long the record is cached — low TTL means a change is expected
- `dig` returns exit code 0 even when the query returns NXDOMAIN; check the status line or `+short` output
- On macOS, system `dig` may lag behind; install BIND tools via Homebrew for the latest version
- For quick lookups where you don't need detail, `host` or `nslookup` are shorter to type

## See Also

- nslookup, dns, curl, mtr, ipv4

## References

- [dig Man Page](https://man7.org/linux/man-pages/man1/dig.1.html)
- [BIND 9 dig Reference](https://bind9.readthedocs.io/en/latest/manpages.html#dig)
- [RFC 1035 — Domain Names: Implementation and Specification](https://www.rfc-editor.org/rfc/rfc1035)
- [RFC 8484 — DNS Queries over HTTPS (DoH)](https://www.rfc-editor.org/rfc/rfc8484)
- [IANA DNS Parameters — Resource Record Types](https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml)
- [Cloudflare — DNS Query Debugging Guide](https://developers.cloudflare.com/1.1.1.1/encryption/dns-over-https/)
- [ISC BIND 9 Administrator Reference Manual](https://bind9.readthedocs.io/en/latest/)
