# DNS-over-HTTPS and DNS-over-TLS (Encrypted DNS Transport)

Encrypted DNS protocols that protect query privacy by wrapping DNS traffic in TLS (DoT, port 853) or HTTPS (DoH, port 443), preventing on-path observers from reading or tampering with DNS queries.

## The Problem: Plaintext DNS

```bash
# Traditional DNS (Do53) sends queries in cleartext on UDP/TCP port 53
# Any on-path observer (ISP, coffee shop WiFi, government) can:
#   - See every domain you resolve
#   - Modify responses (DNS hijacking)
#   - Build browsing profiles from query logs
#   - Inject ads or redirect traffic

# Capture plaintext DNS with tcpdump to see the exposure
sudo tcpdump -i eth0 -n port 53 -l
# 10.0.0.5.41234 > 8.8.8.8.53: 12345+ A? secret-project.example.com. (44)

# Even with HTTPS websites, the DNS query for the domain leaks metadata
# TLS encrypts the payload, but DNS resolution happens before the TLS handshake
```

## DNS-over-TLS (DoT) -- RFC 7858

```bash
# DoT wraps standard DNS wire format inside a TLS session on port 853
# The DNS message is unchanged; only the transport is encrypted

# Query with kdig (knot-dns-utils)
kdig @1.1.1.1 +tls example.com A
kdig @1.1.1.1 +tls-ca example.com AAAA    # strict mode: verify CA certificate
kdig @8.8.8.8 +tls-pin=... example.com A  # pin the server's TLS certificate

# Query with dog (modern DNS client)
dog example.com A @tls:1.1.1.1
dog example.com AAAA @tls:9.9.9.9

# DoT uses a dedicated port (853), making it easy to identify and block
# Firewalls can block TCP/853 to disable DoT without affecting other traffic

# Verify DoT is working with openssl
openssl s_client -connect 1.1.1.1:853 -servername cloudflare-dns.com </dev/null 2>/dev/null | head -20
```

## DNS-over-HTTPS (DoH) -- RFC 8484

```bash
# DoH sends DNS queries as HTTPS requests on port 443
# Queries use the application/dns-message content type (binary wire format)
# or application/dns-json for JSON responses (Cloudflare, Google)

# curl: binary wire format (RFC 8484 compliant)
# Build a DNS query for example.com A record, base64url-encode it
echo -n 'AAABAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=' | \
  base64 -d | \
  curl -s -H 'Accept: application/dns-message' \
       --data-binary @- \
       -H 'Content-Type: application/dns-message' \
       'https://cloudflare-dns.com/dns-query'

# curl: GET with dns parameter (base64url-encoded query)
curl -s -H 'Accept: application/dns-message' \
  'https://cloudflare-dns.com/dns-query?dns=AAABAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE'

# curl: JSON API (non-standard but widely supported)
curl -s -H 'Accept: application/dns-json' \
  'https://cloudflare-dns.com/dns-query?name=example.com&type=A' | jq .

curl -s 'https://dns.google/resolve?name=example.com&type=A' | jq .
curl -s 'https://dns.quad9.net:5053/dns-query?name=example.com&type=A'

# DoH is indistinguishable from normal HTTPS traffic on port 443
# Blocking DoH requires blocking the resolver's IP or using SNI inspection
```

## DNS-over-QUIC (DoQ) -- RFC 9250

```bash
# DoQ uses QUIC (UDP-based) on port 853 for encrypted DNS
# Lower latency than DoT: QUIC 0-RTT handshake vs TLS 1-RTT or 2-RTT
# Each query gets its own QUIC stream -- no head-of-line blocking

# Query with kdig (requires knot-dns-utils 3.1+)
kdig @dns.adguard-dns.com +quic example.com A

# Query with q (dns client with DoQ support)
q example.com A @quic://dns.adguard-dns.com

# DoQ is newer; fewer resolvers support it compared to DoT/DoH
# AdGuard DNS and NextDNS are early adopters
```

## Wire Format Differences

```bash
# Do53 (traditional):
#   Transport: UDP/TCP port 53
#   Encryption: None
#   Message format: Raw DNS wire format (RFC 1035)
#   Padding: None

# DoT (RFC 7858):
#   Transport: TCP port 853 + TLS 1.2/1.3
#   Encryption: TLS
#   Message format: 2-byte length prefix + DNS wire format
#   Padding: Optional (RFC 7830, RFC 8467)

# DoH (RFC 8484):
#   Transport: HTTPS (TCP port 443 + TLS 1.2/1.3)
#   Encryption: TLS (via HTTPS)
#   Message format: HTTP/2 frames containing DNS wire format
#   Content-Type: application/dns-message
#   Methods: GET (query in ?dns= param) or POST (query in body)
#   Padding: Via EDNS(0) padding + HTTP/2 PADDING frames

# DoQ (RFC 9250):
#   Transport: QUIC (UDP port 853)
#   Encryption: TLS 1.3 (built into QUIC)
#   Message format: DNS wire format in QUIC streams
#   Padding: Via QUIC PADDING frames
```

## Opportunistic vs Strict Mode

```bash
# Opportunistic mode:
#   - Try encrypted DNS; fall back to plaintext if it fails
#   - Does NOT verify the server's TLS certificate
#   - Protects against passive eavesdropping but not active attacks
#   - An attacker can force a downgrade to plaintext

# Strict mode:
#   - Require encrypted DNS; fail if it is unavailable
#   - Verify the server's TLS certificate against a trusted CA
#   - Authenticate the resolver's identity (hostname or SPKI pin)
#   - Protects against both passive and active attacks
#   - Can cause resolution failures if the resolver is unreachable

# systemd-resolved: opportunistic vs strict
# DNSOverTLS=opportunistic   -- try TLS, fall back to plaintext
# DNSOverTLS=yes             -- require TLS, fail if unavailable (strict)
```

## Stub Resolver Configuration

### systemd-resolved (DoT)

```bash
# /etc/systemd/resolved.conf
# [Resolve]
# DNS=1.1.1.1#cloudflare-dns.com 1.0.0.1#cloudflare-dns.com
# DNS=2606:4700:4700::1111#cloudflare-dns.com
# DNSOverTLS=yes
# DNSSEC=allow-downgrade
# Domains=~.
# CacheFromLocalhost=no

# The #hostname suffix tells resolved the TLS authentication name
# Without it, resolved cannot verify the server certificate in strict mode

# Apply the configuration
sudo systemctl restart systemd-resolved

# Verify DoT is active
resolvectl status
resolvectl query example.com
# Look for "+DNSOverTLS" in the output

# Monitor DoT connections
sudo ss -tnp | grep ':853'
```

### Unbound (DoT forwarding)

```bash
# /etc/unbound/unbound.conf
# server:
#     tls-cert-bundle: /etc/ssl/certs/ca-certificates.crt
#
# forward-zone:
#     name: "."
#     forward-tls-upstream: yes
#     forward-addr: 1.1.1.1@853#cloudflare-dns.com
#     forward-addr: 1.0.0.1@853#cloudflare-dns.com
#     forward-addr: 9.9.9.9@853#dns.quad9.net
#     forward-addr: 149.112.112.112@853#dns.quad9.net

# Test the configuration
unbound-checkconf
sudo systemctl restart unbound
dig @127.0.0.1 example.com A
```

### stubby (dedicated DoT stub resolver)

```bash
# /etc/stubby/stubby.yml
# resolution_type: GETDNS_RESOLUTION_STUB
# dns_transport_list:
#   - GETDNS_TRANSPORT_TLS
# tls_authentication: GETDNS_AUTHENTICATION_REQUIRED
# tls_query_padding_blocksize: 128
# round_robin_upstreams: 1
# upstream_recursive_servers:
#   - address_data: 1.1.1.1
#     tls_auth_name: "cloudflare-dns.com"
#   - address_data: 9.9.9.9
#     tls_auth_name: "dns.quad9.net"

sudo systemctl restart stubby
# Point /etc/resolv.conf to 127.0.0.1 (stubby's listen address)
```

## Browser DoH Configuration

```bash
# Firefox:
#   Settings > Privacy & Security > DNS over HTTPS
#   Or about:config:
#     network.trr.mode = 2 (DoH preferred, fall back to system)
#     network.trr.mode = 3 (DoH only, no fallback)
#     network.trr.uri = https://cloudflare-dns.com/dns-query
#     network.trr.custom_uri = <your resolver>

# Chrome/Chromium:
#   Settings > Privacy and security > Security > Use secure DNS
#   Or launch flag:
#     --enable-features=DnsOverHttps
#     --force-fieldtrials=DnsOverHttps/Enabled

# Brave:
#   Settings > Privacy and security > Use secure DNS
#   Supports Cloudflare, Google, Quad9, custom

# Edge:
#   Settings > Privacy, search, and services > Use secure DNS

# Safari:
#   No built-in DoH toggle; use system-level configuration
#   macOS: install a DNS profile (.mobileconfig) or use dns-sd
```

## Popular Encrypted DNS Resolvers

```bash
# Cloudflare (1.1.1.1)
#   DoT:  1.1.1.1:853, 1.0.0.1:853  (TLS name: cloudflare-dns.com)
#   DoH:  https://cloudflare-dns.com/dns-query
#   DoH:  https://1.1.1.1/dns-query
#   Filtering variants:
#     1.1.1.2 / 1.0.0.2  — malware blocking
#     1.1.1.3 / 1.0.0.3  — malware + adult content blocking

# Google (8.8.8.8)
#   DoT:  8.8.8.8:853, 8.8.4.4:853  (TLS name: dns.google)
#   DoH:  https://dns.google/dns-query
#   JSON: https://dns.google/resolve?name=example.com&type=A

# Quad9 (9.9.9.9) -- threat-blocking, non-profit
#   DoT:  9.9.9.9:853, 149.112.112.112:853  (TLS name: dns.quad9.net)
#   DoH:  https://dns.quad9.net/dns-query
#   Unfiltered: 9.9.9.10 (no threat blocking)

# NextDNS (custom filtering)
#   DoT:  <config-id>.dns.nextdns.io:853
#   DoH:  https://dns.nextdns.io/<config-id>
#   DoQ:  quic://<config-id>.dns.nextdns.io

# AdGuard DNS
#   DoT:  dns.adguard-dns.com:853
#   DoH:  https://dns.adguard-dns.com/dns-query
#   DoQ:  quic://dns.adguard-dns.com
```

## DNSCrypt

```bash
# DNSCrypt (pre-dates DoT/DoH) -- encrypts DNS with its own protocol
# Uses X25519 key exchange + XSalsa20-Poly1305 or XChaCha20-Poly1305
# Authenticates the server via a short-term signing key published as a DNS TXT record

# Install dnscrypt-proxy
sudo apt install dnscrypt-proxy          # Debian/Ubuntu
brew install dnscrypt-proxy              # macOS

# Configuration: /etc/dnscrypt-proxy/dnscrypt-proxy.toml
# server_names = ['cloudflare', 'google', 'quad9-dnscrypt-ip4-filter-pri']
# listen_addresses = ['127.0.0.1:53', '[::1]:53']
# max_clients = 250
# require_dnssec = true
# require_nofilter = false

sudo systemctl restart dnscrypt-proxy
dig @127.0.0.1 example.com A

# DNSCrypt also supports anonymized DNS (relays) to hide client IP from resolver
# Anonymous DNSCrypt routes queries through relay servers
```

## Encrypted Client Hello (ECH)

```bash
# Even with DoH/DoT, the TLS Client Hello still leaks the target hostname via SNI
# ECH (formerly ESNI) encrypts the SNI field in the TLS handshake
# Requires the server to publish an ECH configuration in DNS (HTTPS/SVCB record)

# Check if a domain publishes ECH keys
dig +short TYPE65 cloudflare.com
# Returns HTTPS/SVCB record with ech= parameter containing the ECH config

kdig cloudflare.com HTTPS
# Look for "ech" in the SVCB parameters

# ECH + DoH together close the two main metadata leaks:
#   DoH encrypts the DNS query (hides what domain you are resolving)
#   ECH encrypts the SNI (hides what domain you are connecting to)
# The combination makes it much harder for on-path observers to determine
# which websites a user is visiting
```

## Privacy vs Network Visibility Tradeoffs

```bash
# For users and privacy advocates:
#   + Prevents ISP/network operator from logging DNS queries
#   + Stops DNS-based censorship and ad injection
#   + Protects against DNS spoofing and cache poisoning
#   + Reduces metadata leakage when combined with ECH

# For network operators and enterprises:
#   - Encrypted DNS bypasses local DNS-based security filtering
#   - Breaks split-horizon DNS (internal vs external resolution)
#   - Reduces visibility into network traffic for threat detection
#   - Makes parental controls and compliance filtering harder
#   - Users can exfiltrate data via DoH (looks like HTTPS traffic)

# Mitigation strategies for enterprises:
#   - Deploy internal DoH/DoT resolvers (encrypted but under org control)
#   - Use canary domains: if "use-application-dns.net" returns NXDOMAIN,
#     Firefox disables DoH and uses system DNS
#   - Block known public DoH resolver IPs at the firewall
#   - Configure managed devices to use the org's encrypted resolver
#   - Implement DNS-based policies on the internal resolver
```

## Enterprise: Split-Horizon DNS

```bash
# Split-horizon DNS returns different answers for internal vs external queries
# DoH/DoT to external resolvers breaks this: internal names fail to resolve

# Solution 1: Internal encrypted resolver
# Deploy DoH/DoT on the internal resolver so clients encrypt
# but the org retains control of resolution policy

# Solution 2: DNS discovery via DHCP/RA
# DDR (Discovery of Designated Resolvers, RFC 9462)
# The network advertises its encrypted resolver via DHCP option or
# a well-known SVCB record: _dns.resolver.arpa

# Check for DDR support
dig _dns.resolver.arpa SVCB

# Solution 3: Canary domain
# Firefox checks "use-application-dns.net" before enabling DoH
# If NXDOMAIN, DoH is disabled; if it resolves, DoH proceeds
# Enterprise DNS can return NXDOMAIN to keep clients on system DNS
dig use-application-dns.net A

# Solution 4: Managed device policy
# Push DoH settings via MDM (mobile device management)
# or group policy to point at the internal DoH resolver
```

## Tips

- Start with opportunistic mode when first deploying encrypted DNS. Switch to strict mode only after confirming the resolver is reachable and correctly configured.
- DoH is harder to block than DoT because it shares port 443 with all HTTPS traffic. If you need to bypass network-level DNS filtering, DoH is the better choice.
- For maximum privacy, combine DoH with ECH and a VPN or Tor. DoH alone still exposes metadata to the resolver operator.
- When using systemd-resolved with DoT, always include the TLS authentication name after the # symbol (e.g., `1.1.1.1#cloudflare-dns.com`). Without it, strict mode cannot verify the server certificate.
- DNS padding (RFC 8467) should be enabled on your stub resolver to prevent query size analysis. Stubby uses 128-byte block padding by default.
- Test encrypted DNS with `kdig +tls` or `dog @tls:` rather than `dig`, which has limited DoT/DoH support in older versions.
- Watch out for DoH bootstrap: the initial resolution of the DoH resolver hostname (e.g., `cloudflare-dns.com`) must happen over traditional DNS unless you hardcode the IP.
- Enterprise networks should deploy DDR (RFC 9462) to advertise their internal encrypted resolver rather than blocking public DoH, which leads to an arms race.
- Be aware that some DoH resolvers log queries. Review the privacy policy of your chosen resolver (Cloudflare publishes annual audits; Quad9 is a non-profit with a no-logging policy).
- DoQ offers the lowest latency of all encrypted DNS transports thanks to QUIC 0-RTT, but resolver support is still limited.

## See Also

- dns, tls, http2

## References

- [RFC 7858 -- DNS over Transport Layer Security (DoT)](https://www.rfc-editor.org/rfc/rfc7858)
- [RFC 8484 -- DNS Queries over HTTPS (DoH)](https://www.rfc-editor.org/rfc/rfc8484)
- [RFC 9250 -- DNS over Dedicated QUIC Connections (DoQ)](https://www.rfc-editor.org/rfc/rfc9250)
- [RFC 8467 -- Padding Policies for Extension Mechanisms for DNS (EDNS(0))](https://www.rfc-editor.org/rfc/rfc8467)
- [RFC 9462 -- Discovery of Designated Resolvers (DDR)](https://www.rfc-editor.org/rfc/rfc9462)
- [RFC 7830 -- The EDNS(0) Padding Option](https://www.rfc-editor.org/rfc/rfc7830)
- [Cloudflare -- DNS over HTTPS](https://developers.cloudflare.com/1.1.1.1/encryption/dns-over-https/)
- [Cloudflare -- DNS over TLS](https://developers.cloudflare.com/1.1.1.1/encryption/dns-over-tls/)
- [Google Public DNS -- DNS over HTTPS](https://developers.google.com/speed/public-dns/docs/doh)
- [Quad9 -- Encrypted DNS](https://www.quad9.net/service/service-addresses-and-features/)
- [curl wiki -- DNS over HTTPS](https://curl.se/docs/doh.html)
- [stubby -- DNS Privacy Stub Resolver](https://dnsprivacy.org/dns_privacy_daemon_-_stubby/)
- [DNSCrypt Protocol Specification](https://dnscrypt.info/protocol/)
- [Encrypted Client Hello (ECH) -- IETF Draft](https://datatracker.ietf.org/doc/draft-ietf-tls-esni/)
