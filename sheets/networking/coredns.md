# CoreDNS (Pluggable DNS Server)

CoreDNS is a flexible, extensible DNS server written in Go that uses a plugin chain architecture for composing DNS functionality, serving as the default cluster DNS in Kubernetes and supporting zone files, forwarding, caching, service discovery, and Prometheus metrics.

## Installation

```bash
# Download binary
wget https://github.com/coredns/coredns/releases/latest/download/coredns_1.12.1_linux_amd64.tgz
tar xf coredns_*.tgz
sudo mv coredns /usr/local/bin/

# Build from source
git clone https://github.com/coredns/coredns.git
cd coredns
make

# Docker
docker run -d --name coredns \
  -p 53:53/udp -p 53:53/tcp \
  -v $(pwd)/Corefile:/Corefile \
  coredns/coredns

# Check version
coredns -version
```

## Corefile Syntax

```text
# Basic Corefile structure
# Each server block: zone:port { plugins }

.:53 {
    forward . 8.8.8.8 8.8.4.4
    cache 30
    log
    errors
}

# Multiple zones
example.com:53 {
    file /etc/coredns/zones/example.com.zone
    log
    errors
}

internal.local:53 {
    file /etc/coredns/zones/internal.local.zone
    log
}

# Catch-all forwarding
.:53 {
    forward . /etc/resolv.conf
    cache 300
    errors
}
```

## Core Plugins

### Forward (Recursive Resolution)

```text
.:53 {
    # Forward to upstream DNS
    forward . 8.8.8.8 8.8.4.4 {
        policy round_robin             # round_robin | random | sequential
        health_check 5s
        max_fails 3
        expire 10s
        tls_servername dns.google
    }

    # Forward with DNS-over-TLS
    forward . tls://8.8.8.8 tls://8.8.4.4 {
        tls_servername dns.google
    }

    # Forward with DNS-over-HTTPS
    forward . https://dns.google/dns-query {
        except internal.local
    }

    # Conditional forwarding
    forward internal.corp 10.0.0.53 {
        policy sequential
    }
}
```

### Cache

```text
.:53 {
    cache {
        success 9984 30                # cache NOERROR for 30s (up to 9984 entries)
        denial 9984 5                  # cache NXDOMAIN/NODATA for 5s
        prefetch 10 1h 20%             # prefetch if >10 hits in 1h, at 20% TTL remaining
        serve_stale 1h                 # serve stale records for up to 1h on upstream failure
    }
}
```

### File (Zone Files)

```text
example.com:53 {
    file /etc/coredns/zones/example.com.zone {
        transfer to *                  # allow zone transfers
        reload 10s                     # check for zone file changes
    }
}
```

### Zone File Example

```text
; /etc/coredns/zones/example.com.zone
$ORIGIN example.com.
$TTL 3600

@       IN  SOA  ns1.example.com. admin.example.com. (
            2024010101  ; Serial
            3600        ; Refresh
            900         ; Retry
            604800      ; Expire
            86400       ; Minimum TTL
        )

@       IN  NS   ns1.example.com.
@       IN  NS   ns2.example.com.
@       IN  A    203.0.113.10
@       IN  MX   10 mail.example.com.

ns1     IN  A    203.0.113.10
ns2     IN  A    203.0.113.11
www     IN  A    203.0.113.10
mail    IN  A    203.0.113.20
api     IN  A    203.0.113.30
db      IN  A    10.0.1.50
*.dev   IN  CNAME dev.example.com.
```

### Log and Errors

```text
.:53 {
    log {
        class denial error             # only log denied/error queries
    }
    errors {
        consolidate 5m ".* i]o timeout"
    }
}
```

### Health and Ready

```text
.:53 {
    health :8080                       # liveness: GET /health on :8080
    ready :8181                        # readiness: GET /ready on :8181
}
```

## Kubernetes Plugin

```text
# In-cluster CoreDNS config (typically via ConfigMap)
.:53 {
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure                  # resolve pod IPs
        fallthrough in-addr.arpa ip6.arpa
        ttl 30
    }
    forward . /etc/resolv.conf {
        max_concurrent 1000
    }
    cache 30
    loop
    reload
    loadbalance
    health :8080
    ready :8181
    prometheus :9153
    errors
    log
}
```

### Kubernetes Stub Domains

```text
# Forward specific domains to custom DNS servers
.:53 {
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
    }

    # Stub domain: forward corp queries to internal DNS
    forward corp.example.com 10.0.0.53 10.0.0.54

    # Default upstream
    forward . 8.8.8.8 8.8.4.4
    cache 30
}
```

### Kubernetes ConfigMap

```yaml
# kubectl edit configmap coredns -n kube-system
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
        }
        forward . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
    }
```

## Prometheus Metrics

```text
.:53 {
    prometheus :9153                   # expose metrics on :9153/metrics
}
```

```bash
# Key metrics
# coredns_dns_requests_total           — total queries by zone, proto, type
# coredns_dns_responses_total          — responses by rcode
# coredns_dns_request_duration_seconds — latency histogram
# coredns_cache_hits_total             — cache hit rate
# coredns_cache_misses_total           — cache miss rate
# coredns_forward_requests_total       — forwarded queries
# coredns_forward_healthcheck_failures_total — upstream health failures

# Scrape config for Prometheus
curl -s http://localhost:9153/metrics | grep coredns_dns_requests_total
```

## Custom Plugins

```bash
# Build CoreDNS with custom plugins
# 1. Clone the repo
git clone https://github.com/coredns/coredns.git && cd coredns

# 2. Edit plugin.cfg to add/remove plugins
# Add a line like: myplug:github.com/user/myplug

# 3. Build
make

# Plugin ordering in plugin.cfg determines execution order in the chain
# Plugins execute in the order listed (top to bottom)
```

## Running CoreDNS

```bash
# Run with default Corefile in current directory
coredns

# Specify Corefile
coredns -conf /etc/coredns/Corefile

# Run on a different port (useful for testing)
coredns -dns.port 1053

# Test with dig
dig @127.0.0.1 -p 53 example.com A
dig @127.0.0.1 -p 53 example.com AAAA
dig @127.0.0.1 -p 53 example.com MX
dig @127.0.0.1 -p 53 example.com ANY +short
```

## Systemd Integration

```ini
# /etc/systemd/system/coredns.service
[Unit]
Description=CoreDNS DNS Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/coredns -conf /etc/coredns/Corefile
Restart=on-failure
User=coredns
Group=coredns
LimitNOFILE=1048576
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable coredns
sudo systemctl start coredns
sudo systemctl status coredns
journalctl -u coredns -f
```

## Tips

- Plugin execution order follows the order in `plugin.cfg` at compile time, not the Corefile order
- Use `cache` with `prefetch` to reduce upstream query load and improve response times for popular domains
- The `loop` plugin detects forwarding loops and halts CoreDNS; always include it in Kubernetes configs
- Use `reload` plugin to automatically pick up Corefile changes without restarting
- Serve stale cache entries during upstream failures with `serve_stale` for better availability
- The `ready` endpoint differs from `health`: ready means all plugins are loaded, health means the process is alive
- In Kubernetes, always use `fallthrough` with the kubernetes plugin so non-cluster queries reach the forward plugin
- Use `forward` with `except` to prevent specific zones from being forwarded upstream
- Prometheus metrics at `:9153` provide cache hit ratio, latency percentiles, and upstream health
- Test Corefile changes with `coredns -dns.port 1053` before deploying to production

## See Also

- openvpn, tailscale, zerotier, dns

## References

- [CoreDNS Documentation](https://coredns.io/manual/toc/)
- [CoreDNS Plugin Reference](https://coredns.io/plugins/)
- [Kubernetes DNS Specification](https://github.com/kubernetes/dns/blob/master/docs/specification.md)
- [CoreDNS GitHub](https://github.com/coredns/coredns)
- [Customizing DNS in Kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/dns-custom-nameservers/)
