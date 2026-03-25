# Apache Kafka (Distributed Event Streaming Platform)

High-throughput, fault-tolerant messaging system for building real-time data pipelines and streaming applications.

## Core Concepts

### Architecture overview

```text
# Key components:
# Broker       - a Kafka server; a cluster has multiple brokers
# Topic        - a named feed/category of messages
# Partition    - ordered, immutable log within a topic (unit of parallelism)
# Offset       - sequential ID for each message within a partition
# Producer     - publishes messages to topics
# Consumer     - reads messages from topics
# Consumer Group - set of consumers sharing the work of reading a topic
# Replication  - each partition has leader + follower replicas across brokers
# ZooKeeper/KRaft - cluster coordination (KRaft replaces ZooKeeper in newer versions)

# Data flow:
# Producer -> Topic (Partition 0, 1, ..., N) -> Consumer Group -> Consumers
```

## Topic Management

### kafka-topics

```bash
# List all topics
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Create a topic
kafka-topics.sh --bootstrap-server localhost:9092 \
  --create --topic my-topic \
  --partitions 6 \
  --replication-factor 3

# Describe a topic (partitions, replicas, ISR)
kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe --topic my-topic

# Delete a topic
kafka-topics.sh --bootstrap-server localhost:9092 \
  --delete --topic my-topic

# Increase partitions (cannot decrease)
kafka-topics.sh --bootstrap-server localhost:9092 \
  --alter --topic my-topic --partitions 12

# List topics with under-replicated partitions
kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe --under-replicated-partitions
```

## Producing Messages

### kafka-console-producer

```bash
# Produce messages interactively (one per line, Ctrl+D to stop)
kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic my-topic

# Produce with keys (key separator is tab by default)
kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic my-topic \
  --property "parse.key=true" \
  --property "key.separator=:"
# Then type: mykey:myvalue

# Produce from a file
kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic my-topic < messages.txt

# Produce with acks=all for durability
kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic my-topic \
  --producer-property acks=all
```

## Consuming Messages

### kafka-console-consumer

```bash
# Consume new messages (from latest offset)
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic my-topic

# Consume from the beginning
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic my-topic --from-beginning

# Consume with a consumer group
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic my-topic --group my-group

# Show keys, values, timestamps, and partition info
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic my-topic --from-beginning \
  --property print.key=true \
  --property print.timestamp=true \
  --property print.partition=true \
  --property print.offset=true

# Consume a specific number of messages
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic my-topic --from-beginning --max-messages 10

# Consume from a specific partition
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic my-topic --partition 0 --offset 42
```

## Consumer Groups

### kafka-consumer-groups

```bash
# List consumer groups
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list

# Describe a group (show lag per partition)
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --describe --group my-group

# Reset offsets to earliest (dry run first)
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group my-group --topic my-topic \
  --reset-offsets --to-earliest --dry-run

# Reset offsets to earliest (execute)
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group my-group --topic my-topic \
  --reset-offsets --to-earliest --execute

# Reset to a specific offset
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group my-group --topic my-topic \
  --reset-offsets --to-offset 100 --execute

# Reset to a timestamp
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group my-group --topic my-topic \
  --reset-offsets --to-datetime "2026-01-01T00:00:00.000" --execute

# Delete a consumer group (must have no active members)
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --delete --group my-group
```

## Configuration

### Key server.properties settings

```properties
# Broker identity
broker.id=0
listeners=PLAINTEXT://0.0.0.0:9092
advertised.listeners=PLAINTEXT://kafka-host:9092

# Log (data) storage
log.dirs=/var/kafka-logs
num.partitions=6                       # default partitions for new topics
default.replication.factor=3

# Retention
log.retention.hours=168                # 7 days (default)
log.retention.bytes=-1                 # unlimited (-1)
log.segment.bytes=1073741824           # 1 GB segment files
log.cleanup.policy=delete             # or "compact" for compacted topics

# Replication
min.insync.replicas=2                  # with acks=all, at least 2 replicas must ack
unclean.leader.election.enable=false   # prevent data loss on leader failure

# Performance
num.io.threads=8
num.network.threads=3
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400

# KRaft mode (no ZooKeeper)
process.roles=broker,controller
controller.quorum.voters=0@kafka-0:9093,1@kafka-1:9093,2@kafka-2:9093
```

### Key producer/consumer configs

```properties
# Producer
acks=all                               # strongest durability
retries=2147483647                     # retry indefinitely
enable.idempotence=true                # exactly-once semantics
linger.ms=5                            # batch delay for throughput
batch.size=16384                       # batch size in bytes
compression.type=lz4                   # lz4, snappy, gzip, zstd

# Consumer
group.id=my-group
auto.offset.reset=earliest             # or "latest"
enable.auto.commit=true
auto.commit.interval.ms=5000
max.poll.records=500
session.timeout.ms=45000
```

## Monitoring

### Checking cluster health

```bash
# Describe the cluster (broker list, controller)
kafka-metadata.sh --snapshot /var/kafka-logs/__cluster_metadata-0/00000000000000000000.log \
  --cluster-id

# Check broker configs
kafka-configs.sh --bootstrap-server localhost:9092 \
  --entity-type brokers --entity-name 0 --describe

# Check topic configs
kafka-configs.sh --bootstrap-server localhost:9092 \
  --entity-type topics --entity-name my-topic --describe

# JMX metrics (key ones to monitor)
# kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec
# kafka.server:type=ReplicaManager,name=UnderReplicatedPartitions
# kafka.consumer:type=consumer-fetch-manager-metrics,client-id=*,name=records-lag-max
```

## Kafka Connect

### Connectors for data integration

```bash
# List installed connector plugins
curl localhost:8083/connector-plugins | jq

# List active connectors
curl localhost:8083/connectors | jq

# Create a connector (example: file source)
curl -X POST localhost:8083/connectors \
  -H "Content-Type: application/json" \
  -d '{
    "name": "file-source",
    "config": {
      "connector.class": "FileStreamSource",
      "tasks.max": "1",
      "file": "/tmp/input.txt",
      "topic": "file-topic"
    }
  }'

# Check connector status
curl localhost:8083/connectors/file-source/status | jq

# Delete a connector
curl -X DELETE localhost:8083/connectors/file-source
```

## Schema Registry

### Schema management basics

```bash
# List subjects
curl localhost:8081/subjects | jq

# Get latest schema for a subject
curl localhost:8081/subjects/my-topic-value/versions/latest | jq

# Register a schema
curl -X POST localhost:8081/subjects/my-topic-value/versions \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"schema": "{\"type\":\"record\",\"name\":\"Event\",\"fields\":[{\"name\":\"id\",\"type\":\"string\"}]}"}'

# Check compatibility
curl -X POST localhost:8081/compatibility/subjects/my-topic-value/versions/latest \
  -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  -d '{"schema": "..."}'
```

## Tips

- Use `kafka-dump-log.sh` to inspect raw segment files for debugging.
- Set `min.insync.replicas=2` with `acks=all` for strong durability guarantees.
- Use compacted topics (`log.cleanup.policy=compact`) for changelog/state topics.
- Consumer lag is the primary metric to monitor; alerts when lag grows.
- Use `kafka-reassign-partitions.sh` to rebalance partitions across brokers.
- For local development, use Docker Compose with Confluent or Redpanda images.
- Partition count determines max consumer parallelism within a group.

## References

- [Apache Kafka Documentation](https://kafka.apache.org/documentation/)
- [Kafka Operations and CLI Tools](https://kafka.apache.org/documentation/#operations)
- [Kafka Configuration Reference](https://kafka.apache.org/documentation/#configuration)
- [Kafka Producer Configuration](https://kafka.apache.org/documentation/#producerconfigs)
- [Kafka Consumer Configuration](https://kafka.apache.org/documentation/#consumerconfigs)
- [KRaft Mode (ZooKeeper Replacement)](https://kafka.apache.org/documentation/#kraft)
- [Kafka Connect](https://kafka.apache.org/documentation/#connect)
- [Confluent Platform Documentation](https://docs.confluent.io/platform/current/overview.html)
- [Confluent Schema Registry](https://docs.confluent.io/platform/current/schema-registry/)
- [Kafka Design Documentation](https://kafka.apache.org/documentation/#design)
- [Apache Kafka GitHub Repository](https://github.com/apache/kafka)
