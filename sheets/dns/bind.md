# BIND (DNS Server)

Authoritative and recursive DNS server, the most widely deployed DNS software.

## named.conf

### Main configuration

```bash
# /etc/bind/named.conf or /etc/named.conf
# options {
#     directory "/var/cache/bind";
#     listen-on { 127.0.0.1; 10.0.0.1; };
#     listen-on-v6 { ::1; };
#     allow-query { any; };
#     allow-recursion { 10.0.0.0/8; 127.0.0.1; };
#     forwarders { 8.8.8.8; 8.8.4.4; };
#     dnssec-validation auto;
#     recursion yes;
# };
```

### Zone declarations

```bash
# zone "example.com" {
#     type master;
#     file "/etc/bind/zones/db.example.com";
#     allow-transfer { 10.0.0.2; };     # secondary NS
# };
#
# zone "0.0.10.in-addr.arpa" {
#     type master;
#     file "/etc/bind/zones/db.10.0.0";  # reverse zone
# };
#
# zone "example.com" {
#     type slave;
#     file "/var/cache/bind/db.example.com";
#     masters { 10.0.0.1; };
# };
```

## Zone Files

### Forward zone (db.example.com)

```bash
# $TTL 3600
# @   IN  SOA  ns1.example.com. admin.example.com. (
#         2025010101  ; Serial (YYYYMMDDNN)
#         3600        ; Refresh (1 hour)
#         900         ; Retry (15 min)
#         604800      ; Expire (1 week)
#         86400       ; Negative TTL (1 day)
# )
#
# ; Name servers
# @       IN  NS   ns1.example.com.
# @       IN  NS   ns2.example.com.
#
# ; A records
# @       IN  A    93.184.216.34
# ns1     IN  A    10.0.0.1
# ns2     IN  A    10.0.0.2
# www     IN  A    93.184.216.34
# mail    IN  A    93.184.216.35
# db      IN  A    10.0.0.10
#
# ; AAAA records
# @       IN  AAAA 2606:2800:220:1:248:1893:25c8:1946
#
# ; CNAME records
# blog    IN  CNAME www.example.com.
# ftp     IN  CNAME www.example.com.
#
# ; MX records
# @       IN  MX   10  mail.example.com.
# @       IN  MX   20  mail2.example.com.
#
# ; TXT records
# @       IN  TXT  "v=spf1 mx a ~all"
# _dmarc  IN  TXT  "v=DMARC1; p=reject; rua=mailto:dmarc@example.com"
#
# ; SRV records
# _sip._tcp  IN  SRV  10 60 5060 sip.example.com.
#
# ; Wildcard
# *       IN  A    93.184.216.34
```

## Record Types

### Common record types

```bash
# A       IPv4 address
# AAAA    IPv6 address
# CNAME   Canonical name (alias) — cannot coexist with other records at same name
# MX      Mail exchange (priority + hostname)
# NS      Name server
# TXT     Text (SPF, DKIM, DMARC, verification)
# SRV     Service locator (priority weight port target)
# PTR     Pointer (reverse DNS)
# SOA     Start of authority (zone metadata)
# CAA     Certificate Authority Authorization
```

### Reverse zone (db.10.0.0)

```bash
# $TTL 3600
# @   IN  SOA  ns1.example.com. admin.example.com. (
#         2025010101 3600 900 604800 86400
# )
# @   IN  NS   ns1.example.com.
# 1   IN  PTR  ns1.example.com.
# 2   IN  PTR  ns2.example.com.
# 10  IN  PTR  db.example.com.
```

## rndc (Remote Name Daemon Control)

### Reload configuration

```bash
rndc reload                                # reload all zones
rndc reload example.com                    # reload one zone
```

### Check status

```bash
rndc status
```

### Flush cache

```bash
rndc flush                                 # flush entire cache
rndc flushname www.example.com             # flush one name
```

### Freeze/thaw (for manual zone edits)

```bash
rndc freeze example.com
# edit zone file, update serial
rndc thaw example.com
```

### Dump and query stats

```bash
rndc dumpdb -cache                         # dump cache to file
rndc querylog on                           # enable query logging
rndc querylog off
```

## dig (DNS Queries)

### Basic queries

```bash
dig example.com                            # default (A record)
dig example.com A                          # explicit A record
dig example.com AAAA                       # IPv6
dig example.com MX                         # mail servers
dig example.com TXT                        # text records
dig example.com NS                         # name servers
dig example.com ANY                        # all records (may be limited)
```

### Query a specific server

```bash
dig @8.8.8.8 example.com
dig @ns1.example.com example.com
dig @127.0.0.1 example.com
```

### Short output

```bash
dig +short example.com
dig +short example.com MX
```

### Trace resolution path

```bash
dig +trace example.com
```

### Reverse lookup

```bash
dig -x 93.184.216.34
```

### Check SOA serial

```bash
dig SOA example.com +short
```

### TCP query (for large responses)

```bash
dig +tcp example.com AXFR                  # zone transfer
```

## Validation & Testing

### Check zone file syntax

```bash
named-checkzone example.com /etc/bind/zones/db.example.com
```

### Check named.conf syntax

```bash
named-checkconf /etc/bind/named.conf
```

### Test resolution

```bash
dig @localhost example.com
nslookup example.com 127.0.0.1
host example.com 127.0.0.1
```

## DNSSEC

### Generate keys

```bash
dnssec-keygen -a ECDSAP256SHA256 -n ZONE example.com      # ZSK
dnssec-keygen -a ECDSAP256SHA256 -n ZONE -f KSK example.com  # KSK
```

### Sign a zone

```bash
dnssec-signzone -o example.com -N INCREMENT db.example.com
```

### Enable in named.conf

```bash
# zone "example.com" {
#     type master;
#     file "/etc/bind/zones/db.example.com.signed";
#     key-directory "/etc/bind/keys";
#     auto-dnssec maintain;
#     inline-signing yes;
# };
```

### Verify DNSSEC

```bash
dig +dnssec example.com
dig example.com DNSKEY +short
delv @8.8.8.8 example.com                 # DNSSEC-aware lookup
```

## Tips

- Always increment the SOA serial number when editing zone files. Use YYYYMMDDNN format.
- `named-checkzone` and `named-checkconf` catch errors before reload. Always run them.
- CNAME records cannot coexist with other record types at the same name. Use A records at zone apex.
- Trailing dots in zone files (e.g., `ns1.example.com.`) are mandatory. Without the dot, BIND appends the zone name.
- `dig +trace` shows the full resolution path from root servers down. Invaluable for debugging delegation.
- Set `allow-transfer` to restrict zone transfers (AXFR) to known secondary servers only.
- `rndc querylog on` enables query logging for debugging, but generates heavy log volume.
- MX record values must be hostnames (not IPs) and should have matching A records.

## References

- [BIND 9 Administrator Reference Manual](https://bind9.readthedocs.io/en/latest/)
- [BIND 9 — named.conf Configuration Reference](https://bind9.readthedocs.io/en/latest/reference.html)
- [BIND 9 — DNSSEC Guide](https://bind9.readthedocs.io/en/latest/dnssec-guide.html)
- [BIND 9 — Man Pages](https://bind9.readthedocs.io/en/latest/manpages.html)
- [ISC BIND 9 Downloads and Release Notes](https://www.isc.org/bind/)
- [RFC 1035 — Domain Names: Implementation and Specification](https://www.rfc-editor.org/rfc/rfc1035)
- [RFC 4033 — DNS Security Introduction and Requirements (DNSSEC)](https://www.rfc-editor.org/rfc/rfc4033)
- [RFC 5936 — DNS Zone Transfer Protocol (AXFR)](https://www.rfc-editor.org/rfc/rfc5936)
- [RFC 1996 — A Mechanism for Prompt Notification of Zone Changes (DNS NOTIFY)](https://www.rfc-editor.org/rfc/rfc1996)
- [IANA Root Zone Database](https://www.iana.org/domains/root/db)
- [man named](https://man7.org/linux/man-pages/man8/named.8.html)
