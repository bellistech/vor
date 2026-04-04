# NATS (Cloud-Native Messaging System)

Lightweight, high-performance messaging system for pub/sub, request/reply, and streaming with JetStream.

## Core Concepts

### Architecture overview

```text
# Key concepts:
# Subject      - a named channel for messages (hierarchical: orders.new, orders.>)
# Pub/Sub      - publish a message, all subscribers receive it
# Request/Reply - synchronous request with a response (built on pub/sub)
# Queue Group  - load-balanced consumption (each message goes to one member)
# JetStream    - persistence layer for at-least-once / exactly-once delivery
# Stream       - JetStream storage for subjects (append-only log)
# Consumer     - JetStream client reading from a stream with ack tracking

# Subject wildcards:
#   *   matches a single token     (orders.* matches orders.new, not orders.us.new)
#   >   matches one or more tokens (orders.> matches orders.new, orders.us.new)
```

## NATS CLI (nats)

### Pub/Sub

```bash
# Subscribe to a subject (blocks, prints received messages)
nats sub "orders.>"

# Subscribe and show only N messages then exit
nats sub "orders.new" --count 5

# Publish a message
nats pub orders.new "new order 123"

# Publish with headers
nats pub orders.new "order data" --header "Priority:high" --header "Source:web"

# Publish from stdin
echo '{"id": 123}' | nats pub orders.new
```

### Request/Reply

```bash
# Send a request and wait for a reply
nats req orders.status '{"id": 123}'

# Start a reply service (responds to each request)
nats reply orders.status "order is shipped"

# Reply with a command (dynamic response)
nats reply 'time.>' --command "date +%s"
```

### Queue groups

```bash
# Subscribe as part of a queue group (load-balanced)
# Only one member of the group receives each message
nats sub orders.new --queue workers

# Start multiple instances for load balancing
nats sub orders.new --queue workers &
nats sub orders.new --queue workers &
nats pub orders.new "task 1"   # delivered to one worker
```

### Server info and benchmarks

```bash
# Show server info
nats server info

# List all servers in the cluster
nats server list

# Check server health
nats server ping

# Benchmark pub/sub throughput
nats bench test --pub 1 --sub 1 --msgs 1000000 --size 128
```

## JetStream

### Stream management

```bash
# Add a stream (stores messages for subjects matching "orders.>")
nats stream add ORDERS \
  --subjects "orders.>" \
  --storage file \
  --retention limits \
  --max-msgs -1 \
  --max-bytes -1 \
  --max-age 72h \
  --replicas 3

# List streams
nats stream list

# Show stream info and stats
nats stream info ORDERS

# View messages in a stream
nats stream view ORDERS

# Purge all messages from a stream
nats stream purge ORDERS

# Delete a stream
nats stream delete ORDERS

# Edit a stream
nats stream edit ORDERS --max-age 168h
```

### Consumer management

```bash
# Add a push consumer (messages delivered to a subject)
nats consumer add ORDERS push-consumer \
  --deliver-subject orders.deliver \
  --ack explicit \
  --deliver all

# Add a pull consumer (client pulls messages on demand)
nats consumer add ORDERS pull-consumer \
  --pull \
  --ack explicit \
  --deliver all \
  --max-pending 100

# List consumers for a stream
nats consumer list ORDERS

# Show consumer info
nats consumer info ORDERS pull-consumer

# Pull and ack messages from a pull consumer
nats consumer next ORDERS pull-consumer --count 10

# Delete a consumer
nats consumer delete ORDERS pull-consumer
```

### Key-Value store

```bash
# Create a KV bucket
nats kv add CONFIG --history 5 --ttl 24h

# Put a key
nats kv put CONFIG db.host "postgres.example.com"

# Get a key
nats kv get CONFIG db.host

# List keys
nats kv ls CONFIG

# Watch for changes
nats kv watch CONFIG

# Delete a key
nats kv del CONFIG db.host

# Delete the bucket
nats kv rm CONFIG
```

### Object store

```bash
# Create an object store bucket
nats object add ASSETS --max-bucket-size 1GB

# Put a file
nats object put ASSETS ./image.png

# Get a file
nats object get ASSETS image.png -O ./downloaded.png

# List objects
nats object ls ASSETS

# Delete an object
nats object del ASSETS image.png
```

## Server Configuration

### Basic nats-server.conf

```text
# Listen address
listen: 0.0.0.0:4222

# Enable JetStream
jetstream {
    store_dir: /data/nats
    max_mem: 1G
    max_file: 10G
}

# Authentication
authorization {
    users = [
        { user: admin, password: s3cret, permissions: { publish: ">", subscribe: ">" } }
        { user: reader, password: r3ad, permissions: { subscribe: ">" } }
    ]
}

# Logging
log_file: /var/log/nats/nats.log
debug: false
trace: false
```

### TLS configuration

```text
tls {
    cert_file: "/etc/nats/server-cert.pem"
    key_file:  "/etc/nats/server-key.pem"
    ca_file:   "/etc/nats/ca-cert.pem"
    verify:    true
}
```

## Clustering

### Cluster setup

```text
# Node 1: nats-server.conf
cluster {
    name: my-cluster
    listen: 0.0.0.0:6222
    routes: [
        nats-route://nats-2:6222
        nats-route://nats-3:6222
    ]
}

# Superclusters: connect multiple clusters via gateway
gateway {
    name: region-us
    listen: 0.0.0.0:7222
    gateways: [
        { name: region-eu, urls: ["nats://eu-nats-1:7222"] }
    ]
}
```

### Leaf nodes

```text
# Leaf node connects to a hub cluster (useful for edge/IoT)
leafnodes {
    remotes [
        { urls: ["nats-leaf://hub-nats:7422"] }
    ]
}

# Hub cluster allows leaf connections
leafnodes {
    listen: 0.0.0.0:7422
}
```

## Tips

- Use `nats context` to save and switch between server connection configs.
- Use `nats account info` to check JetStream usage and limits.
- Subject naming convention: use dots for hierarchy (`service.action.entity`).
- Use queue groups for horizontal scaling without JetStream when at-most-once is fine.
- For exactly-once, use JetStream with `--ack explicit` and idempotent processing.
- Use `nats events` to monitor server events (connects, disconnects, errors).
- Replicas must be an odd number (1, 3, 5) for JetStream consensus.

## See Also

- kafka
- rabbitmq
- redis

## References

- [NATS Documentation](https://docs.nats.io/)
- [NATS CLI Reference](https://docs.nats.io/running-a-nats-service/clients#nats)
- [JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [NATS Server Configuration](https://docs.nats.io/running-a-nats-service/configuration)
- [NATS Subject-Based Messaging](https://docs.nats.io/nats-concepts/subjects)
- [NATS Clustering](https://docs.nats.io/running-a-nats-service/configuration/clustering)
- [NATS Leaf Nodes](https://docs.nats.io/running-a-nats-service/configuration/leafnodes)
- [NATS Key-Value Store](https://docs.nats.io/nats-concepts/jetstream/key-value-store)
- [NATS Authentication and Authorization](https://docs.nats.io/running-a-nats-service/configuration/securing_nats)
- [NATS Server GitHub Repository](https://github.com/nats-io/nats-server)
- [NATS CLI GitHub Repository](https://github.com/nats-io/natscli)
