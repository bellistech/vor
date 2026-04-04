# Kafka Streams (Stream Processing Library)

Kafka Streams is a client-side Java/Scala library for building real-time streaming applications on top of Apache Kafka, providing exactly-once processing, stateful operations, and interactive queries without requiring a separate processing cluster.

## Topology Building

### KStream and KTable basics

```java
import org.apache.kafka.streams.*;
import org.apache.kafka.streams.kstream.*;
import org.apache.kafka.common.serialization.Serdes;

// Configuration
Properties props = new Properties();
props.put(StreamsConfig.APPLICATION_ID_CONFIG, "my-stream-app");
props.put(StreamsConfig.BOOTSTRAP_SERVERS_CONFIG, "broker:9092");
props.put(StreamsConfig.DEFAULT_KEY_SERDE_CLASS_CONFIG, Serdes.String().getClass());
props.put(StreamsConfig.DEFAULT_VALUE_SERDE_CLASS_CONFIG, Serdes.String().getClass());
props.put(StreamsConfig.PROCESSING_GUARANTEE_CONFIG, StreamsConfig.EXACTLY_ONCE_V2);

// Build topology
StreamsBuilder builder = new StreamsBuilder();

// KStream: unbounded stream of events
KStream<String, String> events = builder.stream("input-events");

// KTable: changelog stream (latest value per key)
KTable<String, String> users = builder.table("users",
    Materialized.<String, String, KeyValueStore<Bytes, byte[]>>as("users-store")
        .withKeySerde(Serdes.String())
        .withValueSerde(Serdes.String()));

// GlobalKTable: full copy on every instance (for small reference data)
GlobalKTable<String, String> config = builder.globalTable("app-config");

// Start the application
KafkaStreams streams = new KafkaStreams(builder.build(), props);
streams.start();

// Shutdown hook
Runtime.getRuntime().addShutdownHook(new Thread(streams::close));
```

## Stream Operations

### Stateless transformations

```java
// Filter
KStream<String, Order> orders = events
    .filter((key, value) -> value.getAmount() > 0);

// Map (change key and value)
KStream<String, String> mapped = events
    .map((key, value) -> KeyValue.pair(value.getUserId(), value.toJson()));

// MapValues (change value only, preserves key — avoids repartition)
KStream<String, Double> amounts = events
    .mapValues(event -> event.getAmount());

// FlatMap (one-to-many)
KStream<String, String> words = events
    .flatMapValues(value -> Arrays.asList(value.split("\\s+")));

// Branch (split stream)
Map<String, KStream<String, Order>> branches = orders.split(Named.as("split-"))
    .branch((key, order) -> order.getAmount() > 1000, Branched.as("large"))
    .branch((key, order) -> order.getAmount() > 100, Branched.as("medium"))
    .defaultBranch(Branched.as("small"));

KStream<String, Order> largeOrders = branches.get("split-large");

// SelectKey (rekey — triggers repartition)
KStream<String, Order> rekeyedByProduct = orders
    .selectKey((key, order) -> order.getProductId());

// Merge streams
KStream<String, Order> merged = largeOrders.merge(branches.get("split-medium"));

// Peek (side effect, non-transforming)
events.peek((key, value) -> log.info("Processing: {}", key));
```

### Stateful transformations

```java
// GroupByKey + aggregate
KTable<String, Long> eventCounts = events
    .groupByKey()
    .count(Materialized.as("event-counts-store"));

// GroupBy (rekey) + aggregate
KTable<String, Double> categoryTotals = orders
    .groupBy((key, order) -> KeyValue.pair(order.getCategory(), order),
        Grouped.with(Serdes.String(), orderSerde))
    .aggregate(
        () -> 0.0,
        (key, order, total) -> total + order.getAmount(),
        (key, order, total) -> total - order.getAmount(),
        Materialized.as("category-totals")
    );

// Reduce
KTable<String, Order> latestOrders = orders
    .groupByKey()
    .reduce((oldVal, newVal) -> newVal,
        Materialized.as("latest-orders"));
```

## Windowing

### Window types

```java
// Tumbling window (fixed, non-overlapping)
KTable<Windowed<String>, Long> hourlyCounts = events
    .groupByKey()
    .windowedBy(TimeWindows.ofSizeWithNoGrace(Duration.ofHours(1)))
    .count(Materialized.as("hourly-counts"));

// Hopping window (overlapping)
KTable<Windowed<String>, Long> slidingCounts = events
    .groupByKey()
    .windowedBy(TimeWindows.ofSizeAndGrace(
        Duration.ofMinutes(10), Duration.ofMinutes(2))
        .advanceBy(Duration.ofMinutes(5)))
    .count();

// Session window (gap-based)
KTable<Windowed<String>, Long> sessionCounts = events
    .groupByKey()
    .windowedBy(SessionWindows.ofInactivityGapAndGrace(
        Duration.ofMinutes(30), Duration.ofMinutes(5)))
    .count();

// Sliding window (for joins)
// Automatically used in stream-stream joins

// Suppress (emit only final window result)
hourlyCounts
    .suppress(Suppressed.untilWindowCloses(
        Suppressed.BufferConfig.unbounded()))
    .toStream()
    .foreach((windowedKey, count) ->
        System.out.println(windowedKey.key() + ": " + count));
```

## Joins

### Stream and table joins

```java
// KStream-KTable join (lookup enrichment)
KStream<String, EnrichedOrder> enriched = orders.join(
    users,
    (order, user) -> new EnrichedOrder(order, user)
);

// KStream-GlobalKTable join (broadcast lookup)
KStream<String, EnrichedOrder> globalEnriched = orders.join(
    config,
    (key, order) -> order.getRegion(),  // extract foreign key
    (order, configValue) -> new EnrichedOrder(order, configValue)
);

// KStream-KStream join (windowed)
KStream<String, MatchedEvent> matched = streamA.join(
    streamB,
    (a, b) -> new MatchedEvent(a, b),
    JoinWindows.ofTimeDifferenceAndGrace(
        Duration.ofMinutes(5), Duration.ofMinutes(1)),
    StreamJoined.with(Serdes.String(), serdeA, serdeB)
);

// KTable-KTable join (changelog join)
KTable<String, UserProfile> profiles = users.join(
    preferences,
    (user, pref) -> new UserProfile(user, pref)
);

// Left join (keep left even without match)
KStream<String, EnrichedOrder> leftJoined = orders.leftJoin(
    users,
    (order, user) -> new EnrichedOrder(order, user != null ? user : "unknown")
);
```

## State Stores and Interactive Queries

### Querying local state

```java
// Query a read-only state store
ReadOnlyKeyValueStore<String, Long> store =
    streams.store(StoreQueryParameters.fromNameAndType(
        "event-counts-store", QueryableStoreTypes.keyValueStore()));

// Point lookup
Long count = store.get("user-123");

// Range scan
KeyValueIterator<String, Long> range = store.range("user-100", "user-200");
while (range.hasNext()) {
    KeyValue<String, Long> entry = range.next();
    System.out.println(entry.key + " = " + entry.value);
}
range.close();

// All entries
KeyValueIterator<String, Long> all = store.all();

// Windowed store query
ReadOnlyWindowStore<String, Long> windowStore =
    streams.store(StoreQueryParameters.fromNameAndType(
        "hourly-counts", QueryableStoreTypes.windowStore()));

// Fetch windows for a key in a time range
WindowStoreIterator<Long> windows = windowStore.fetch(
    "user-123",
    Instant.parse("2024-01-01T00:00:00Z"),
    Instant.parse("2024-01-02T00:00:00Z"));

// Metadata for distributed queries
Collection<StreamsMetadata> metadata = streams.metadataForAllStreamsClients();
StreamsMetadata hostForKey = streams.queryMetadataForKey(
    "event-counts-store", "user-123", Serdes.String().serializer());
```

## Configuration

### Key properties

```properties
# Application identity
application.id=my-stream-app
client.id=stream-client-1

# Kafka connection
bootstrap.servers=broker1:9092,broker2:9092

# Processing guarantee
processing.guarantee=exactly_once_v2

# Threads and parallelism
num.stream.threads=4

# State directory
state.dir=/var/lib/kafka-streams

# Commit interval
commit.interval.ms=100

# Cache size (per thread, for deduplication)
statestore.cache.max.bytes=10485760

# Internal topic config
replication.factor=3

# Producer tuning
producer.acks=all
producer.compression.type=lz4
producer.batch.size=32768

# Consumer tuning
consumer.auto.offset.reset=earliest
consumer.max.poll.records=1000
```

## Tips

- Use `mapValues()` instead of `map()` when only transforming values; `map()` triggers repartitioning which is expensive
- Set `processing.guarantee=exactly_once_v2` for production (requires Kafka broker 2.5+); it uses fewer transactions than v1
- Increase `num.stream.threads` to match input topic partition count for maximum parallelism within a single JVM
- Use `GlobalKTable` only for small reference data (loaded in full on every instance); use `KTable` for large datasets
- Suppress windowed results with `Suppressed.untilWindowCloses()` to emit only final aggregates and avoid downstream duplicates
- Always set `state.dir` to a fast local disk (SSD); RocksDB state store performance depends heavily on disk I/O
- Use the `Topology.describe()` method to inspect and debug your processing topology before deployment
- Handle deserialization errors with `DEFAULT_DESERIALIZATION_EXCEPTION_HANDLER_CLASS_CONFIG` instead of crashing the app
- Monitor consumer lag on the internal repartition and changelog topics; high lag indicates processing bottlenecks
- Use `statestore.cache.max.bytes` to control the caching layer that deduplicates updates before flushing to RocksDB
- Name all processors and state stores explicitly with `.as()` for savepoint/upgrade compatibility

## See Also

- kafka, flink, spark, rabbitmq, redis, event-sourcing

## References

- [Kafka Streams Documentation](https://kafka.apache.org/documentation/streams/)
- [Kafka Streams Developer Guide](https://docs.confluent.io/platform/current/streams/developer-guide/overview.html)
- [Kafka Streams DSL](https://kafka.apache.org/documentation/streams/developer-guide/dsl-api.html)
- [Interactive Queries](https://kafka.apache.org/documentation/streams/developer-guide/interactive-queries.html)
- [Kafka Streams Architecture](https://docs.confluent.io/platform/current/streams/architecture.html)
