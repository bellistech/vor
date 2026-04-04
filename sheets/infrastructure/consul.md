# Consul (Service Discovery & Service Mesh)

Discover services, store configuration in a distributed KV store, and enforce service-to-service encryption and authorization using HashiCorp Consul's gossip protocol, Raft consensus, and Connect service mesh.

## Agent Operations

### Starting Consul

```bash
# Development mode (single node, in-memory, no persistence)
consul agent -dev

# Production server mode
consul agent -server \
  -bootstrap-expect=3 \
  -data-dir=/opt/consul/data \
  -config-dir=/etc/consul.d \
  -bind=10.0.1.10 \
  -client=0.0.0.0 \
  -ui

# Client mode (joins cluster)
consul agent \
  -data-dir=/opt/consul/data \
  -config-dir=/etc/consul.d \
  -bind=10.0.1.20 \
  -retry-join=10.0.1.10

# Join an existing cluster
consul join 10.0.1.10
consul join -wan 10.1.1.10        # WAN join for multi-DC

# Check cluster members
consul members
consul members -wan               # WAN members across datacenters
consul operator raft list-peers   # Raft peer status
```

### Agent Configuration

```hcl
# /etc/consul.d/consul.hcl
datacenter = "dc1"
data_dir   = "/opt/consul/data"
log_level  = "INFO"

server           = true
bootstrap_expect = 3

bind_addr   = "{{ GetInterfaceIP \"eth0\" }}"
client_addr = "0.0.0.0"

ui_config {
  enabled = true
}

retry_join = ["10.0.1.10", "10.0.1.11", "10.0.1.12"]

encrypt = "pUqJrVyVRj5jsiYEkM/tFQYfWyJIv4s3XkvDwy7Cu5s="

performance {
  raft_multiplier = 1
}

telemetry {
  prometheus_retention_time = "24h"
}
```

## Service Registration

### Service Definition

```hcl
# /etc/consul.d/web.hcl
service {
  name = "web"
  port = 8080
  tags = ["v1.2.3", "production"]

  meta {
    version = "1.2.3"
    env     = "production"
  }

  check {
    http     = "http://localhost:8080/health"
    interval = "10s"
    timeout  = "3s"
  }

  connect {
    sidecar_service {}
  }
}
```

### HTTP API Registration

```bash
# Register service via API
curl -X PUT http://localhost:8500/v1/agent/service/register \
  -d '{
    "Name": "web",
    "Port": 8080,
    "Tags": ["v1.2.3"],
    "Check": {
      "HTTP": "http://localhost:8080/health",
      "Interval": "10s"
    }
  }'

# Deregister service
curl -X PUT http://localhost:8500/v1/agent/service/deregister/web

# List registered services
curl http://localhost:8500/v1/agent/services | jq

# Get service health
curl "http://localhost:8500/v1/health/service/web?passing" | jq
```

## Service Discovery

### DNS Interface

```bash
# Query service via DNS (default port 8600)
dig @127.0.0.1 -p 8600 web.service.consul SRV
dig @127.0.0.1 -p 8600 web.service.consul A

# Tag-based filtering
dig @127.0.0.1 -p 8600 v1.web.service.consul

# Cross-datacenter query
dig @127.0.0.1 -p 8600 web.service.dc2.consul

# Node lookup
dig @127.0.0.1 -p 8600 node1.node.consul

# Forward DNS to Consul (dnsmasq)
# /etc/dnsmasq.d/consul
# server=/consul/127.0.0.1#8600

# Forward DNS to Consul (systemd-resolved)
# [Resolve]
# DNS=127.0.0.1:8600
# Domains=~consul
```

### HTTP API Discovery

```bash
# Discover healthy service instances
curl "http://localhost:8500/v1/health/service/web?passing" | \
  jq '.[].Service | {Address, Port}'

# Catalog query (all instances including unhealthy)
curl http://localhost:8500/v1/catalog/service/web | jq

# List all services
curl http://localhost:8500/v1/catalog/services | jq

# Prepared queries (load-balanced, cross-DC failover)
curl -X POST http://localhost:8500/v1/query \
  -d '{
    "Name": "web-near",
    "Service": {
      "Service": "web",
      "Failover": {
        "NearestN": 3,
        "Datacenters": ["dc2", "dc3"]
      }
    }
  }'

# Execute prepared query
curl http://localhost:8500/v1/query/web-near/execute | jq
```

## Health Checks

### Check Types

```hcl
# HTTP check
check {
  http     = "http://localhost:8080/health"
  interval = "10s"
  timeout  = "3s"
}

# TCP check
check {
  tcp      = "localhost:5432"
  interval = "10s"
  timeout  = "3s"
}

# Script check
check {
  args     = ["/usr/local/bin/check_disk.sh"]
  interval = "30s"
  timeout  = "10s"
}

# gRPC check
check {
  grpc                  = "localhost:50051"
  grpc_use_tls          = true
  interval              = "10s"
  tls_skip_verify       = false
}

# TTL check (service reports its own status)
check {
  ttl = "30s"
}

# Alias check (mirrors another service's health)
check {
  alias_service = "database"
}
```

### TTL Check Updates

```bash
# Pass
curl -X PUT http://localhost:8500/v1/agent/check/pass/service:web

# Warn
curl -X PUT http://localhost:8500/v1/agent/check/warn/service:web

# Fail
curl -X PUT "http://localhost:8500/v1/agent/check/fail/service:web?note=disk+full"
```

## KV Store

### Key-Value Operations

```bash
# Put a value
consul kv put config/database/host 10.0.1.50
consul kv put config/database/port 5432
consul kv put -flags=42 config/app/feature-flag true

# Get a value
consul kv get config/database/host
consul kv get -detailed config/database/host   # with metadata

# List keys by prefix
consul kv get -recurse config/database/
consul kv get -keys config/                    # keys only, no values

# Delete
consul kv delete config/database/host
consul kv delete -recurse config/database/     # delete prefix tree

# CAS (Check-And-Set) for atomic updates
consul kv put -cas -modify-index=123 config/app/counter 42

# Export/Import
consul kv export config/ > backup.json
consul kv import @backup.json

# Watch for changes
consul watch -type=key -key=config/app/feature-flag cat
consul watch -type=keyprefix -prefix=config/ cat
```

### HTTP API KV

```bash
# PUT with flags
curl -X PUT -d 'myvalue' http://localhost:8500/v1/kv/config/key

# GET (base64 encoded value)
curl http://localhost:8500/v1/kv/config/key | jq '.[0].Value' -r | base64 -d

# Blocking query (long poll, wait for changes)
curl "http://localhost:8500/v1/kv/config/key?index=42&wait=5m"

# CAS update
curl -X PUT -d 'newvalue' "http://localhost:8500/v1/kv/config/key?cas=123"

# Acquire lock
curl -X PUT -d 'locked' "http://localhost:8500/v1/kv/locks/mylock?acquire=$SESSION_ID"
```

## Connect (Service Mesh)

### Intentions (Service-to-Service Authorization)

```bash
# Allow web to talk to database
consul intention create web database

# Deny api to talk to admin
consul intention create -deny api admin

# List intentions
consul intention list

# Check if connection is allowed
consul intention check web database

# Delete intention
consul intention delete web database
```

### Intentions via Config

```hcl
# /etc/consul.d/intentions.hcl
Kind = "service-intentions"
Name = "database"

Sources = [
  {
    Name   = "web"
    Action = "allow"
  },
  {
    Name   = "api"
    Action = "allow"
  },
  {
    Name   = "*"
    Action = "deny"
  }
]
```

## ACLs

### Bootstrap and Token Management

```bash
# Bootstrap ACL system (first time only)
consul acl bootstrap

# Create a policy
consul acl policy create -name "web-read" \
  -rules='service_prefix "web" { policy = "read" } node_prefix "" { policy = "read" }'

# Create a token with policy
consul acl token create -description "Web service token" \
  -policy-name "web-read"

# List tokens
consul acl token list

# Set token for CLI
export CONSUL_HTTP_TOKEN="your-token-here"
```

## Tips

- Always run 3 or 5 server nodes for Raft quorum -- 2-node clusters lose quorum on a single failure with no way to recover
- Use `retry-join` with cloud auto-join (`provider=aws tag_key=consul`) instead of hardcoding IP addresses
- Set `encrypt` with a gossip encryption key (`consul keygen`) on every agent to prevent unauthorized cluster joins
- Use blocking queries (`?index=N&wait=5m`) instead of polling the KV store -- they long-poll and return immediately on changes
- Prefer prepared queries over raw service lookups for automatic failover across datacenters with `NearestN`
- Configure DNS forwarding (dnsmasq or systemd-resolved) so applications resolve `*.consul` domains transparently
- Use Connect intentions in default-deny mode and explicitly allow only required service-to-service paths
- Set `leave_on_terminate = true` on client agents so they gracefully deregister services on shutdown instead of waiting for failure detection
- Run `consul operator raft list-peers` regularly to verify all servers are voting members and none are stuck
- Enable Prometheus telemetry (`prometheus_retention_time`) and scrape the `/v1/agent/metrics` endpoint for monitoring
- Use `-bootstrap-expect` rather than `-bootstrap` to prevent split-brain during initial cluster formation

## See Also

- nomad, vault, terraform, envoy, dns, service-mesh, etcd

## References

- [Consul Documentation](https://developer.hashicorp.com/consul/docs)
- [Consul Architecture](https://developer.hashicorp.com/consul/docs/architecture)
- [Raft Consensus Algorithm](https://raft.github.io/raft.pdf)
- [Serf Gossip Protocol](https://www.serf.io/docs/internals/gossip.html)
- [Consul Connect (Service Mesh)](https://developer.hashicorp.com/consul/docs/connect)
