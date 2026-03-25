# nslookup (Name Server Lookup)

Simple DNS query tool — interactive and non-interactive modes for quick lookups.

## Forward Lookups

### Basic queries
```bash
nslookup example.com                  # default DNS server, A record
nslookup example.com 8.8.8.8         # query specific DNS server
nslookup example.com 1.1.1.1         # query Cloudflare
```

## Record Types

### Query specific record types
```bash
nslookup -type=A example.com          # IPv4 address
nslookup -type=AAAA example.com       # IPv6 address
nslookup -type=MX example.com         # mail exchange
nslookup -type=NS example.com         # name servers
nslookup -type=TXT example.com        # TXT records (SPF, DKIM)
nslookup -type=CNAME www.example.com  # canonical name
nslookup -type=SOA example.com        # start of authority
nslookup -type=SRV _sip._tcp.example.com  # service records
nslookup -type=CAA example.com        # certificate authority auth
nslookup -type=PTR 34.216.184.93.in-addr.arpa  # manual PTR
nslookup -type=ANY example.com        # all records (may be refused)
```

## Reverse Lookups

### IP to hostname
```bash
nslookup 93.184.216.34                # reverse DNS lookup
nslookup 2606:2800:220:1:248:1893:25c8:1946  # IPv6 reverse
```

## Specific DNS Server

### Query authoritative or public servers
```bash
nslookup example.com ns1.example.com  # query authoritative NS
nslookup example.com 8.8.4.4         # Google secondary
nslookup example.com 9.9.9.9         # Quad9
nslookup example.com 208.67.222.222  # OpenDNS
```

## Interactive Mode

### Enter interactive mode
```bash
nslookup
# Then at the > prompt:
> server 8.8.8.8                      # switch DNS server
> set type=MX                         # set record type
> example.com                         # query
> set type=NS
> example.com
> set debug                           # enable debug output
> set nodebug                         # disable debug
> exit                                # quit
```

### Useful interactive commands
```bash
nslookup
> set type=any
> set timeout=10                      # 10 second timeout
> set retry=3                         # 3 retries
> set domain=example.com              # default domain suffix
> web                                 # resolves web.example.com
> exit
```

## Debug Mode

### Verbose output
```bash
nslookup -debug example.com           # show query/response details
nslookup -debug -type=MX example.com  # debug MX lookup
```

## Common Tasks

### Check all MX records and priority
```bash
nslookup -type=MX gmail.com
```

### Verify name server delegation
```bash
nslookup -type=NS example.com
```

### Check SPF record
```bash
nslookup -type=TXT example.com
```

### Verify PTR matches forward
```bash
# Forward lookup
nslookup example.com
# Reverse lookup on the IP
nslookup 93.184.216.34
```

### Check if a domain exists
```bash
nslookup nonexistent.example.com
# "** server can't find" = NXDOMAIN
```

## Scripting with nslookup

### Parse output
```bash
nslookup example.com | grep 'Address' | tail -1
nslookup -type=MX example.com | grep 'mail exchanger'
nslookup example.com 8.8.8.8 2>/dev/null | awk '/^Address:/{print $2}' | tail -1
```

## Tips

- `nslookup` is simpler than `dig` but less powerful — fine for quick checks
- `dig` is preferred for scripting because its output is more consistent and parseable
- `nslookup` is available by default on Windows, macOS, and most Linux distributions
- In interactive mode, `set debug` reveals the raw DNS response — useful for troubleshooting
- `nslookup` always queries a DNS server (no `/etc/hosts` file); use `getent hosts` to test full resolution chain
- The "Non-authoritative answer" header means the response came from a cache, not the authoritative NS
- Some DNS servers refuse `type=ANY` queries (RFC 8482) — query specific types instead
- On Windows, `nslookup` is often the only DNS tool available, making it essential to know

## References

- [man nslookup](https://man7.org/linux/man-pages/man1/nslookup.1.html)
- [BIND 9 — nslookup Reference](https://bind9.readthedocs.io/en/latest/manpages.html#nslookup)
- [RFC 1035 — Domain Names: Implementation and Specification](https://www.rfc-editor.org/rfc/rfc1035)
- [IANA DNS Parameters — Resource Record Types](https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml)
- [man dig — More Capable DNS Lookup Tool](https://man7.org/linux/man-pages/man1/dig.1.html)
- [man host — Simplified DNS Lookup](https://man7.org/linux/man-pages/man1/host.1.html)
