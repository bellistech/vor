# dnsmasq (Lightweight DNS/DHCP Server)

Combined DNS forwarder, DHCP server, and TFTP server for small networks and development.

## DNS Configuration

### Basic config (/etc/dnsmasq.conf)

```bash
# Listen on specific interface
listen-address=127.0.0.1,10.0.0.1

# Do not use /etc/resolv.conf (set upstream manually)
no-resolv

# Upstream DNS servers
server=8.8.8.8
server=8.8.4.4
server=1.1.1.1

# Cache size (default 150)
cache-size=1000

# Do not forward plain names (without dots)
domain-needed

# Never forward reverse lookups for private ranges
bogus-priv

# Log queries (for debugging)
log-queries
log-facility=/var/log/dnsmasq.log
```

## DNS Forwarding

### Forward specific domains to specific servers

```bash
# Internal domain to corporate DNS
server=/corp.example.com/10.0.0.53

# Consul service discovery
server=/consul/127.0.0.1#8600

# Forward .internal to a local resolver
server=/internal/192.168.1.1
```

### Conditional forwarding

```bash
# Reverse DNS for 10.x to internal DNS
server=/10.in-addr.arpa/10.0.0.53

# Reverse DNS for 192.168.x
server=/168.192.in-addr.arpa/192.168.1.1
```

## Static DNS Entries

### Address records

```bash
# Map hostname to IP
address=/myapp.local/192.168.1.100
address=/api.local/192.168.1.101

# Wildcard — all subdomains resolve to one IP
address=/dev.example.com/192.168.1.50
# *.dev.example.com AND dev.example.com -> 192.168.1.50

# Block a domain (resolve to 0.0.0.0)
address=/ads.example.com/0.0.0.0

# Return NXDOMAIN
address=/blocked.example.com/
```

### Using /etc/hosts

```bash
# dnsmasq reads /etc/hosts by default
# 192.168.1.100  myapp.local
# 192.168.1.101  api.local db.local
```

### Additional hosts file

```bash
addn-hosts=/etc/dnsmasq.hosts
```

### CNAME records

```bash
cname=www.example.local,example.local
cname=cdn.example.local,example.local
```

## Wildcard DNS

### Resolve all subdomains

```bash
# All *.local.dev -> 127.0.0.1
address=/local.dev/127.0.0.1

# All *.test -> 127.0.0.1
address=/test/127.0.0.1
```

This is commonly used for local development to route all subdomains to localhost.

## DHCP

### Basic DHCP server

```bash
# Enable DHCP on a range
dhcp-range=192.168.1.100,192.168.1.200,24h

# Set gateway
dhcp-option=3,192.168.1.1

# Set DNS server (itself)
dhcp-option=6,192.168.1.1

# Set domain
domain=home.local

# Lease file
dhcp-leasefile=/var/lib/dnsmasq/dnsmasq.leases
```

### Static DHCP leases

```bash
# Assign fixed IP by MAC address
dhcp-host=00:1A:2B:3C:4D:5E,workstation,192.168.1.50
dhcp-host=AA:BB:CC:DD:EE:FF,printer,192.168.1.51

# Set hostname without fixed IP
dhcp-host=00:1A:2B:3C:4D:5E,myhost
```

### DHCP options

```bash
dhcp-option=option:router,192.168.1.1          # gateway
dhcp-option=option:dns-server,192.168.1.1      # DNS
dhcp-option=option:ntp-server,192.168.1.1      # NTP
dhcp-option=option:domain-search,home.local    # search domain
dhcp-option=42,192.168.1.1                     # NTP by number
```

### Multiple subnets

```bash
dhcp-range=lan,192.168.1.100,192.168.1.200,24h
dhcp-range=guest,192.168.2.100,192.168.2.200,1h
```

## PXE Boot

### TFTP and PXE

```bash
# Enable TFTP
enable-tftp
tftp-root=/var/lib/tftpboot

# PXE boot file
dhcp-boot=pxelinux.0

# PXE for UEFI
dhcp-match=set:efi-x86_64,option:client-arch,7
dhcp-boot=tag:efi-x86_64,bootx64.efi
```

## Listen Address

### Bind to specific interfaces

```bash
listen-address=127.0.0.1
listen-address=10.0.0.1

# Or bind to interface name
interface=eth0
interface=lo

# Do not listen on specific interface
except-interface=docker0

# Bind only to specified interfaces (safer)
bind-interfaces
```

## Cache

### Cache settings

```bash
cache-size=10000                           # max cached entries
no-negcache                                # do not cache NXDOMAIN
local-ttl=300                              # override TTL for local entries
min-cache-ttl=60                           # minimum TTL to cache
```

### View cache statistics

```bash
kill -USR1 $(pidof dnsmasq)                # dump stats to log
# Check /var/log/syslog or journal for cache hit/miss stats
```

## Operations

### Test config

```bash
dnsmasq --test                             # syntax check
```

### Start/restart

```bash
systemctl start dnsmasq
systemctl restart dnsmasq
systemctl status dnsmasq
```

### Query via dnsmasq

```bash
dig @127.0.0.1 myapp.local
dig @10.0.0.1 example.com
nslookup myapp.local 127.0.0.1
```

### View leases

```bash
cat /var/lib/dnsmasq/dnsmasq.leases
```

### macOS usage (via Homebrew)

```bash
brew install dnsmasq
# Config: /opt/homebrew/etc/dnsmasq.conf (Apple Silicon)
# Config: /usr/local/etc/dnsmasq.conf (Intel)
sudo brew services start dnsmasq
# Point resolver: sudo mkdir -p /etc/resolver && echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/local.dev
```

## Tips

- `address=/domain/ip` handles wildcard subdomains. It matches the domain and everything under it.
- `server=/domain/ip#port` forwards only that domain to a specific upstream (useful for split DNS).
- `no-resolv` plus explicit `server=` lines gives you full control over upstream DNS.
- `bogus-priv` prevents leaking internal reverse lookups to upstream DNS servers.
- `bind-interfaces` is safer than the default. Without it, dnsmasq binds to 0.0.0.0 and filters by address.
- On macOS, use `/etc/resolver/` directory to route specific TLDs to dnsmasq without changing system DNS.
- `log-queries` is invaluable for debugging but generates a lot of output. Disable in production.
- dnsmasq reads `/etc/hosts` on startup. Changes to `/etc/hosts` require a SIGHUP or restart.
