# mDNS and DNS-SD (Multicast DNS and DNS Service Discovery)

A reference for zero-configuration name resolution (mDNS, RFC 6762) and service discovery (DNS-SD, RFC 6763) covering .local domains, multicast addressing, record types, service registration and browsing, conflict resolution, and tooling on macOS and Linux.

## mDNS Fundamentals

### Multicast Addresses and Ports

```bash
# mDNS uses multicast group addresses on UDP port 5353
# IPv4: 224.0.0.251:5353
# IPv6: ff02::fb:5353

# The .local top-level domain is reserved for mDNS (RFC 6762)
# Names ending in .local are resolved via multicast, not unicast DNS

# mDNS is link-local scope only — packets are not forwarded by routers
# IPv4 TTL = 255; receivers MUST discard packets with TTL != 255
# IPv6 hop limit = 255; same verification applies
```

### How mDNS Resolution Works

```bash
# 1. Host wants to resolve "myprinter.local"
# 2. Sends DNS query to multicast group 224.0.0.251:5353
# 3. The device owning "myprinter.local" responds (also via multicast)
# 4. All hosts on the link can cache the response

# mDNS reuses standard DNS message format (same header, question, answer sections)
# Responses are authoritative (AA bit set)
# Default TTL for mDNS records: 120 seconds (for host addresses)
# Goodbye packets: record with TTL=0 announces departure
```

### Querying with Command-Line Tools

```bash
# macOS — dns-sd (built-in, talks to mDNSResponder)
dns-sd -G v4v6 myhost.local            # look up A/AAAA for a .local name
dns-sd -q myhost.local A               # query specific record type

# Linux — avahi-resolve
avahi-resolve -n myhost.local           # resolve .local name to address
avahi-resolve -a 192.168.1.42          # reverse lookup (address to name)

# dig can query mDNS with explicit multicast target
dig @224.0.0.251 -p 5353 myhost.local  # direct multicast query (unreliable)
```

## DNS-SD (DNS Service Discovery)

### Core Concept

```bash
# DNS-SD (RFC 6763) uses standard DNS record types to advertise services
# Three record types work together:
#   PTR — enumerates service instances (browsing)
#   SRV — locates a service instance (host + port)
#   TXT — carries service metadata (key=value pairs)

# Service instance name format:
#   <Instance Name>.<Service Type>.<Domain>
#   Example: "Office Printer._ipp._tcp.local"
#   - Instance Name: human-readable, can contain spaces/punctuation
#   - Service Type: _protocol._transport (e.g., _ipp._tcp)
#   - Domain: typically .local for mDNS
```

### Service Type Naming

```bash
# Service types follow the format: _service._proto
# _proto is either _tcp or _udp

# Common service types:
# _http._tcp        — HTTP web server
# _https._tcp       — HTTPS web server
# _ssh._tcp         — SSH remote login
# _ipp._tcp         — Internet Printing Protocol
# _printer._tcp     — LPR/LPD printing
# _airplay._tcp     — Apple AirPlay
# _raop._tcp        — Remote Audio Output Protocol (AirPlay audio)
# _smb._tcp         — SMB/CIFS file sharing
# _afpovertcp._tcp  — AFP file sharing (Apple)
# _nfs._tcp         — NFS file sharing
# _ftp._tcp         — FTP file transfer
# _daap._tcp        — Digital Audio Access Protocol (iTunes)
# _googlecast._tcp  — Google Chromecast
# _mqtt._tcp        — MQTT message broker
# _coap._udp        — Constrained Application Protocol (IoT)
# _hap._tcp         — HomeKit Accessory Protocol

# Full registry: https://www.iana.org/assignments/service-names-port-numbers
```

### PTR Records (Browsing)

```bash
# PTR records answer "what instances of this service type exist?"
# Query: _ipp._tcp.local PTR ?
# Answer: _ipp._tcp.local PTR "Office Printer._ipp._tcp.local"
#         _ipp._tcp.local PTR "Lab Printer._ipp._tcp.local"

# Browse all service types on the network:
# Query: _services._dns-sd._udp.local PTR ?
# Returns PTR records for each registered service type

# macOS
dns-sd -B _http._tcp local             # browse for HTTP services
dns-sd -B _ipp._tcp local              # browse for printers
dns-sd -B _services._dns-sd._udp local # browse all service types

# Linux (Avahi)
avahi-browse _http._tcp                 # browse HTTP services
avahi-browse -a                         # browse all services
avahi-browse -a -t                      # browse all, terminate after listing
avahi-browse -a -r                      # browse all, resolve addresses
```

### SRV Records (Locating)

```bash
# SRV records answer "where is this service instance?"
# Format: priority weight port target
# Example:
# "Office Printer._ipp._tcp.local" SRV 0 0 631 printer1.local

# macOS — resolve a service instance
dns-sd -L "Office Printer" _ipp._tcp local

# Linux
avahi-browse _ipp._tcp -r               # browse and resolve (shows SRV + TXT)
```

### TXT Records (Metadata)

```bash
# TXT records carry key=value metadata for a service instance
# Each key=value pair is a single DNS TXT string
# Keys are case-insensitive, ASCII only, max 9 characters recommended
# Values are opaque bytes (often UTF-8 text)
# Total TXT record size should not exceed 1300 bytes

# Example TXT record for an IPP printer:
# txtvers=1
# qtotal=1
# pdl=application/postscript,image/jpeg
# rp=printers/Office_Printer
# ty=HP LaserJet Pro
# note=Room 204, 2nd Floor

# A key with no "=" sign is a boolean flag (present = true)
# An empty value "key=" means key exists with empty value
# Missing key means false/default
```

## Conflict Resolution

### Probing and Announcing

```bash
# When a host starts or claims a new name:
# 1. PROBING (3 queries, 250ms apart)
#    - Send query for the desired name (QU bit set, unicast response requested)
#    - If any response contains a conflicting record, conflict detected
#
# 2. CONFLICT RESOLUTION
#    - Compare conflicting records lexicographically (by rdata)
#    - Loser must choose a new name (typically appends " (2)", " (3)", etc.)
#    - Example: "MyLaptop.local" -> "MyLaptop (2).local"
#
# 3. ANNOUNCING (2 announcements, 1 second apart)
#    - After probing succeeds, announce ownership via unsolicited responses
#    - Announcements are sent to the multicast group

# Hosts must defend their names:
# - If a conflicting query is received, respond within 250ms
# - If conflict cannot be resolved, one host must yield
```

## Service Registration

### macOS (dns-sd / mDNSResponder)

```bash
# Register a service (stays registered while command runs)
dns-sd -R "My Web Server" _http._tcp local 8080 path=/index.html
dns-sd -R "My SSH" _ssh._tcp . 22       # register SSH on default domain

# Proxy registration (register on behalf of another device)
dns-sd -P "Dumb Printer" _ipp._tcp local 631 printer1.local 192.168.1.50 \
    txtvers=1 pdl=application/postscript
```

### Linux (Avahi)

```bash
# avahi-publish — register a service
avahi-publish -s "My Web Server" _http._tcp 8080 "path=/index.html"
avahi-publish -s "My SSH" _ssh._tcp 22

# Register a host address
avahi-publish -a mydevice.local 192.168.1.100

# Avahi static service files (/etc/avahi/services/*.service)
# Persistent registration without running avahi-publish
cat <<'EOF' > /etc/avahi/services/myhttp.service
<?xml version="1.0" standalone='no'?>
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">
<service-group>
  <name>My Web Server</name>
  <service>
    <type>_http._tcp</type>
    <port>8080</port>
    <txt-record>path=/index.html</txt-record>
  </service>
</service-group>
EOF
```

## Implementation Platforms

### Bonjour (Apple)

```bash
# mDNSResponder — system daemon on macOS, iOS, tvOS, watchOS
# Provides mDNS + DNS-SD for all Apple zero-conf networking
# Powers: AirPrint, AirPlay, AirDrop discovery, HomeKit, Finder sharing

# Check mDNSResponder status (macOS)
sudo launchctl list | grep mDNSResponder
log show --predicate 'process == "mDNSResponder"' --last 5m

# Bonjour Browser (GUI) or dns-sd (CLI) for inspection
dns-sd -B _airplay._tcp local           # find AirPlay devices
dns-sd -B _raop._tcp local              # find AirPlay audio receivers
dns-sd -B _companion-link._tcp local    # find Apple devices
```

### Avahi (Linux)

```bash
# avahi-daemon — open source mDNS/DNS-SD implementation for Linux
# Configuration: /etc/avahi/avahi-daemon.conf

# Key config options
# [server]
# host-name=myhost                      # hostname to publish
# domain-name=local                     # domain (almost always "local")
# use-ipv4=yes
# use-ipv6=yes
# allow-interfaces=eth0,wlan0           # restrict to specific interfaces
#
# [wide-area]
# enable-wide-area=no                   # wide-area DNS-SD (usually off)
#
# [publish]
# publish-addresses=yes                 # publish A/AAAA records
# publish-hinfo=no                      # host info (security concern)
# publish-workstation=no                # workstation service

# Service management
systemctl status avahi-daemon
systemctl enable --now avahi-daemon
journalctl -u avahi-daemon -f           # follow daemon logs

# avahi-daemon --check                  # check if daemon is running
# avahi-daemon --reload                 # reload configuration
```

### systemd-resolved

```bash
# systemd-resolved has built-in mDNS support (responder and resolver)
# Configuration: /etc/systemd/resolved.conf

# Enable mDNS in resolved.conf:
# [Resolve]
# MulticastDNS=yes                      # enable mDNS globally

# Per-interface mDNS control via networkd or nmcli:
resolvectl mdns                         # show mDNS status per interface
resolvectl mdns eth0 yes                # enable mDNS on eth0

# systemd-resolved mDNS modes:
# yes    — full mDNS responder + resolver
# resolve — resolver only (no registration)
# no     — disabled

# Query .local names via resolvectl
resolvectl query myhost.local
resolvectl service _http._tcp.local     # DNS-SD service browsing
```

## Common Use Cases

```bash
# Printers — AirPrint uses _ipp._tcp and _ipps._tcp via mDNS
dns-sd -B _ipp._tcp local              # find printers
avahi-browse _ipp._tcp -r              # find + resolve printers

# AirPlay / screen mirroring
dns-sd -B _airplay._tcp local          # find AirPlay displays
dns-sd -B _raop._tcp local             # find AirPlay audio

# Chromecast / Google Cast
avahi-browse _googlecast._tcp -r       # find Chromecast devices

# SSH servers
avahi-browse _ssh._tcp -r              # find SSH servers on LAN

# File sharing
avahi-browse _smb._tcp -r              # find SMB shares
avahi-browse _afpovertcp._tcp -r       # find AFP shares (Apple)

# IoT / Home automation
avahi-browse _hap._tcp -r              # find HomeKit accessories
avahi-browse _mqtt._tcp -r             # find MQTT brokers
avahi-browse _coap._udp -r             # find CoAP devices
```

## Link-Local and Multicast Scope

```bash
# mDNS operates at link-local scope
# - Packets stay on the local network segment (not routed)
# - IPv4 link-local range: 169.254.0.0/16 (not required for mDNS)
# - mDNS works with any IP address on the link (not just link-local)

# Multicast scope control:
# - IPv4: TTL=255 (but multicast routers do not forward 224.0.0.x)
# - IPv6: ff02::fb is link-local scope (scope ID = 2)

# mDNS on VLANs:
# - Each VLAN is a separate multicast domain
# - Devices on different VLANs cannot discover each other via mDNS
# - mDNS reflectors/repeaters can bridge VLANs (e.g., avahi-daemon reflector mode)

# Avahi reflector mode (bridges mDNS across interfaces)
# In /etc/avahi/avahi-daemon.conf:
# [reflector]
# enable-reflector=yes                  # forward mDNS between interfaces
# reflect-ipv=no                        # do not reflect between IPv4/IPv6
```

## Tips

- Use `avahi-browse -a -r -t` for a one-shot snapshot of all services and their resolved addresses on the local network.
- On macOS, `dns-sd -B _services._dns-sd._udp` lists every service type currently advertised -- useful for discovering unknown devices.
- If .local resolution is slow on Linux, check that `/etc/nsswitch.conf` has `mdns4_minimal [NOTFOUND=return]` before `dns` in the `hosts:` line.
- Avahi and systemd-resolved can conflict if both are enabled for mDNS on the same interface. Disable one or scope them to different interfaces.
- mDNS names are limited to 63 characters per label and 255 characters total (same as unicast DNS). Service instance names can be up to 63 bytes of UTF-8.
- For IoT devices that do not support mDNS natively, use a proxy registration (`dns-sd -P` or Avahi static service files) from a host that does.
- Chromecast and AirPlay both rely on DNS-SD, so blocking multicast on the WLAN will break discovery for those devices.
- In Docker/container environments, mDNS requires `--net=host` or a macvlan network; default bridge networking isolates multicast.

## See Also

- dns, dhcp, slaac

## References

- [RFC 6762 -- Multicast DNS](https://www.rfc-editor.org/rfc/rfc6762)
- [RFC 6763 -- DNS-Based Service Discovery](https://www.rfc-editor.org/rfc/rfc6763)
- [RFC 3927 -- Dynamic Configuration of IPv4 Link-Local Addresses](https://www.rfc-editor.org/rfc/rfc3927)
- [IANA Service Name and Transport Protocol Port Number Registry](https://www.iana.org/assignments/service-names-port-numbers)
- [Apple Bonjour Overview](https://developer.apple.com/bonjour/)
- [Apple dns-sd(1) Man Page](https://developer.apple.com/library/archive/documentation/Networking/Reference/DNSServiceDiscovery_CRef/)
- [Avahi Project](https://avahi.org/)
- [Avahi Wiki](https://wiki.archlinux.org/title/Avahi)
- [systemd-resolved mDNS Documentation](https://www.freedesktop.org/software/systemd/man/systemd-resolved.service.html)
