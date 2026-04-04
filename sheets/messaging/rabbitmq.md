# RabbitMQ (Advanced Message Queuing Protocol Broker)

Feature-rich message broker supporting multiple protocols with flexible routing, reliability, and clustering.

## Core Concepts

### Architecture overview

```text
# Key components:
# Producer     - sends messages to an exchange
# Exchange     - routes messages to queues based on rules (bindings + routing keys)
# Queue        - buffer that stores messages until consumed
# Binding      - rule linking an exchange to a queue (with optional routing key)
# Consumer     - receives messages from a queue
# Vhost        - virtual host for logical isolation (like a namespace)
# Connection   - TCP connection from client to broker
# Channel      - multiplexed lightweight connection within a TCP connection

# Message flow:
# Producer -> Exchange --(binding/routing key)--> Queue -> Consumer

# Exchange types:
# direct   - routes to queues with exact routing key match
# topic    - routes by routing key pattern (*.error, log.#)
# fanout   - broadcasts to all bound queues (ignores routing key)
# headers  - routes based on message header attributes
```

## rabbitmqctl

### Server and node management

```bash
# Check server status
rabbitmqctl status

# Check cluster status
rabbitmqctl cluster_status

# Stop / start the application (not the Erlang VM)
rabbitmqctl stop_app
rabbitmqctl start_app

# Reset a node (removes all data, use with caution)
rabbitmqctl stop_app
rabbitmqctl reset
rabbitmqctl start_app
```

### Queue operations

```bash
# List queues with message counts
rabbitmqctl list_queues name messages messages_ready messages_unacknowledged

# List queues in a specific vhost
rabbitmqctl list_queues -p /staging name messages

# Purge a queue (delete all messages)
rabbitmqctl purge_queue my-queue

# Delete a queue
rabbitmqctl delete_queue my-queue
```

### Exchange operations

```bash
# List exchanges
rabbitmqctl list_exchanges name type durable auto_delete

# List bindings
rabbitmqctl list_bindings source_name destination_name routing_key
```

### User management

```bash
# List users
rabbitmqctl list_users

# Add a user
rabbitmqctl add_user myuser mypassword

# Change password
rabbitmqctl change_password myuser newpassword

# Delete a user
rabbitmqctl delete_user myuser

# Set user tags (administrator, monitoring, management, etc.)
rabbitmqctl set_user_tags myuser administrator
```

### Permissions

```bash
# Set permissions (configure, write, read regexes)
# Grant full access to all resources in vhost /
rabbitmqctl set_permissions -p / myuser ".*" ".*" ".*"

# Grant limited permissions (only queues starting with "app.")
rabbitmqctl set_permissions -p / myuser "^app\..*" "^app\..*" "^app\..*"

# List permissions
rabbitmqctl list_permissions -p /

# List user permissions across all vhosts
rabbitmqctl list_user_permissions myuser
```

### Virtual hosts

```bash
# List vhosts
rabbitmqctl list_vhosts

# Add a vhost
rabbitmqctl add_vhost /staging

# Delete a vhost (deletes all its exchanges, queues, bindings)
rabbitmqctl delete_vhost /staging
```

## Management Plugin

### HTTP API and UI

```bash
# Enable the management plugin (provides web UI on port 15672)
rabbitmq-plugins enable rabbitmq_management

# Access the web UI
# http://localhost:15672  (default: guest/guest, localhost only)

# API examples with curl
# List queues
curl -u guest:guest http://localhost:15672/api/queues | jq

# Get a specific queue
curl -u guest:guest http://localhost:15672/api/queues/%2F/my-queue | jq

# Publish a message via API
curl -u guest:guest -X POST http://localhost:15672/api/exchanges/%2F/amq.default/publish \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {},
    "routing_key": "my-queue",
    "payload": "hello from API",
    "payload_encoding": "string"
  }'

# Export definitions (exchanges, queues, users, permissions)
curl -u guest:guest http://localhost:15672/api/definitions > definitions.json

# Import definitions
curl -u guest:guest -X POST http://localhost:15672/api/definitions \
  -H "Content-Type: application/json" -d @definitions.json
```

## Exchange Types

### Direct exchange

```text
# Routes messages to queues where binding key == routing key (exact match)
# Use case: task distribution, RPC

# Example:
# Exchange "orders" bound to queue "order-processor" with key "new-order"
# Message with routing_key="new-order" -> delivered to "order-processor"
# Message with routing_key="cancel-order" -> dropped (no matching binding)
```

### Topic exchange

```text
# Routes by routing key pattern matching with wildcards
#   * matches exactly one word
#   # matches zero or more words

# Example bindings on exchange "logs":
#   queue "all-errors"   bound with "*.error"
#   queue "kernel-logs"  bound with "kern.#"
#   queue "everything"   bound with "#"

# Message with routing_key="app.error"  -> all-errors, everything
# Message with routing_key="kern.info"  -> kernel-logs, everything
# Message with routing_key="kern.critical.disk" -> kernel-logs, everything
```

### Fanout exchange

```text
# Broadcasts to ALL bound queues (routing key is ignored)
# Use case: pub/sub, notifications, event broadcasting

# Example:
# Exchange "events" with 3 bound queues
# Any message published -> copied to all 3 queues
```

### Headers exchange

```text
# Routes based on message header attributes instead of routing key
# Binding specifies header key-value pairs and match type (x-match: all | any)
# Use case: complex routing based on message metadata
```

## Advanced Features

### Dead letter exchanges

```bash
# Declare a queue with dead-letter exchange via policy
rabbitmqctl set_policy DLX "^my-queue$" \
  '{"dead-letter-exchange":"dlx","dead-letter-routing-key":"dlq"}' \
  --apply-to queues

# Messages are dead-lettered when:
# - rejected (basic.reject / basic.nack) with requeue=false
# - TTL expires
# - queue length limit exceeded
```

### TTL (time-to-live)

```bash
# Set message TTL on a queue (milliseconds) via policy
rabbitmqctl set_policy TTL "^my-queue$" \
  '{"message-ttl":60000}' \
  --apply-to queues

# Per-message TTL is set by the publisher in message properties
# Queue-level TTL applies to all messages in that queue
```

### Priority queues

```bash
# Declare via client with x-max-priority argument (e.g., max priority 10)
# Higher priority messages are delivered first
# Set per-message priority in the message properties (0 = lowest)

# Via management API:
curl -u guest:guest -X PUT http://localhost:15672/api/queues/%2F/priority-queue \
  -H "Content-Type: application/json" \
  -d '{"arguments":{"x-max-priority":10}}'
```

### Shovel and federation

```bash
# Shovel: reliably move messages between brokers (point-to-point)
rabbitmq-plugins enable rabbitmq_shovel
rabbitmq-plugins enable rabbitmq_shovel_management

# Federation: replicate exchanges/queues across clusters (distributed)
rabbitmq-plugins enable rabbitmq_federation
rabbitmq-plugins enable rabbitmq_federation_management

# Configure federation upstream via policy
rabbitmqctl set_parameter federation-upstream my-upstream \
  '{"uri":"amqp://remote-host","expires":3600000}'

rabbitmqctl set_policy federate-me "^federated\." \
  '{"federation-upstream-set":"all"}' \
  --apply-to exchanges
```

## Tips

- Use `rabbitmqctl list_connections` and `list_channels` to debug connection issues.
- Use publisher confirms (`confirm.select`) for reliable publishing.
- Use consumer acknowledgements (`basic.ack`) to prevent message loss.
- Set `prefetch_count` (QoS) to control how many unacked messages a consumer holds.
- Use quorum queues (`x-queue-type: quorum`) for replicated, durable queues in production.
- Use lazy queues for very large backlogs (stores messages to disk early).
- Monitor with `rabbitmqctl list_queues name messages consumers` or the management UI.

## See Also

- kafka
- nats
- redis

## References

- [RabbitMQ Documentation](https://www.rabbitmq.com/docs)
- [rabbitmqctl Reference](https://www.rabbitmq.com/docs/rabbitmqctl)
- [Management Plugin and HTTP API](https://www.rabbitmq.com/docs/management)
- [AMQP 0-9-1 Concepts (Exchange Types)](https://www.rabbitmq.com/tutorials/amqp-concepts)
- [Queues and Queue Types](https://www.rabbitmq.com/docs/queues)
- [Quorum Queues](https://www.rabbitmq.com/docs/quorum-queues)
- [Dead Lettering](https://www.rabbitmq.com/docs/dlx)
- [Clustering Guide](https://www.rabbitmq.com/docs/clustering)
- [Publisher Confirms](https://www.rabbitmq.com/docs/confirms)
- [Access Control and Permissions](https://www.rabbitmq.com/docs/access-control)
- [RabbitMQ Tutorials](https://www.rabbitmq.com/tutorials)
- [RabbitMQ GitHub Repository](https://github.com/rabbitmq/rabbitmq-server)
