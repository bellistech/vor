# Apache Flink (Stateful Stream Processing)

Apache Flink is a distributed stream processing framework for stateful computations over bounded and unbounded data streams with exactly-once semantics, event time processing, and millisecond-level latency.

## DataStream API

### Basic stream operations

```java
// Create execution environment
StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
env.setParallelism(4);

// Source: read from Kafka
KafkaSource<String> source = KafkaSource.<String>builder()
    .setBootstrapServers("broker:9092")
    .setTopics("input-events")
    .setGroupId("flink-consumer")
    .setStartingOffsets(OffsetsInitializer.earliest())
    .setValueOnlyDeserializer(new SimpleStringSchema())
    .build();

DataStream<String> stream = env.fromSource(
    source, WatermarkStrategy.noWatermarks(), "Kafka Source");

// Transformations
DataStream<Event> events = stream
    .map(json -> Event.fromJson(json))
    .filter(event -> event.getType().equals("purchase"))
    .keyBy(Event::getUserId)
    .process(new UserEventProcessor());

// Sink: write to Kafka
KafkaSink<String> sink = KafkaSink.<String>builder()
    .setBootstrapServers("broker:9092")
    .setRecordSerializer(
        KafkaRecordSerializationSchema.builder()
            .setTopic("output-events")
            .setValueSerializationSchema(new SimpleStringSchema())
            .build()
    )
    .setDeliveryGuarantee(DeliveryGuarantee.EXACTLY_ONCE)
    .build();

events.map(Event::toJson).sinkTo(sink);
env.execute("Event Processing Job");
```

### Python (PyFlink)

```python
from pyflink.datastream import StreamExecutionEnvironment
from pyflink.table import StreamTableEnvironment

env = StreamExecutionEnvironment.get_execution_environment()
env.set_parallelism(4)
t_env = StreamTableEnvironment.create(env)

# Define source with SQL DDL
t_env.execute_sql("""
    CREATE TABLE events (
        event_id STRING,
        user_id STRING,
        amount DOUBLE,
        event_time TIMESTAMP(3),
        WATERMARK FOR event_time AS event_time - INTERVAL '5' SECOND
    ) WITH (
        'connector' = 'kafka',
        'topic' = 'events',
        'properties.bootstrap.servers' = 'broker:9092',
        'properties.group.id' = 'flink-sql',
        'format' = 'json',
        'scan.startup.mode' = 'latest-offset'
    )
""")

# Windowed aggregation
result = t_env.sql_query("""
    SELECT user_id,
           TUMBLE_START(event_time, INTERVAL '1' HOUR) AS window_start,
           COUNT(*) AS event_count,
           SUM(amount) AS total_amount
    FROM events
    GROUP BY user_id, TUMBLE(event_time, INTERVAL '1' HOUR)
""")
```

## Event Time and Watermarks

### Watermark strategies

```java
// Bounded out-of-orderness watermark (most common)
WatermarkStrategy<Event> strategy = WatermarkStrategy
    .<Event>forBoundedOutOfOrderness(Duration.ofSeconds(10))
    .withTimestampAssigner((event, timestamp) -> event.getTimestamp())
    .withIdleness(Duration.ofMinutes(1));

DataStream<Event> withWatermarks = stream
    .assignTimestampsAndWatermarks(strategy);

// Custom periodic watermark generator
WatermarkStrategy<Event> custom = WatermarkStrategy
    .forGenerator(ctx -> new WatermarkGenerator<Event>() {
        private long maxTimestamp = Long.MIN_VALUE;

        @Override
        public void onEvent(Event event, long ts, WatermarkOutput output) {
            maxTimestamp = Math.max(maxTimestamp, event.getTimestamp());
        }

        @Override
        public void onPeriodicEmit(WatermarkOutput output) {
            output.emitWatermark(new Watermark(maxTimestamp - 5000));
        }
    });
```

## Windowing

### Window types

```java
// Tumbling window (fixed, non-overlapping)
stream.keyBy(Event::getUserId)
    .window(TumblingEventTimeWindows.of(Time.minutes(5)))
    .aggregate(new EventAggregator());

// Sliding window (overlapping)
stream.keyBy(Event::getUserId)
    .window(SlidingEventTimeWindows.of(Time.minutes(10), Time.minutes(5)))
    .reduce((a, b) -> a.merge(b));

// Session window (gap-based)
stream.keyBy(Event::getUserId)
    .window(EventTimeSessionWindows.withGap(Time.minutes(30)))
    .process(new SessionWindowFunction());

// Global window with custom trigger
stream.keyBy(Event::getUserId)
    .window(GlobalWindows.create())
    .trigger(CountTrigger.of(100))
    .process(new BatchProcessor());

// Allowed lateness (handle late events)
stream.keyBy(Event::getUserId)
    .window(TumblingEventTimeWindows.of(Time.hours(1)))
    .allowedLateness(Time.minutes(10))
    .sideOutputLateData(lateOutputTag)
    .aggregate(new HourlyAggregator());
```

## State Management

### Keyed state types

```java
public class StatefulProcessor extends KeyedProcessFunction<String, Event, Result> {
    // Value state (single value per key)
    private ValueState<Long> countState;
    // List state (list per key)
    private ListState<Event> bufferState;
    // Map state (map per key)
    private MapState<String, Double> metricsState;
    // Reducing state (aggregates automatically)
    private ReducingState<Long> sumState;

    @Override
    public void open(Configuration parameters) {
        countState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("count", Long.class));
        bufferState = getRuntimeContext().getListState(
            new ListStateDescriptor<>("buffer", Event.class));
        metricsState = getRuntimeContext().getMapState(
            new MapStateDescriptor<>("metrics", String.class, Double.class));
        sumState = getRuntimeContext().getReducingState(
            new ReducingStateDescriptor<>("sum", Long::sum, Long.class));
    }

    @Override
    public void processElement(Event event, Context ctx, Collector<Result> out) {
        Long count = countState.value();
        count = (count == null) ? 1L : count + 1;
        countState.update(count);

        // Register event-time timer
        ctx.timerService().registerEventTimeTimer(
            event.getTimestamp() + 60000);
    }

    @Override
    public void onTimer(long timestamp, OnTimerContext ctx, Collector<Result> out) {
        // Timer fired — emit result and clear state
        out.collect(new Result(ctx.getCurrentKey(), countState.value()));
        countState.clear();
    }
}
```

## Checkpointing and Savepoints

### Checkpoint configuration

```java
// Enable checkpointing (every 60 seconds)
env.enableCheckpointing(60000, CheckpointingMode.EXACTLY_ONCE);

// Checkpoint config
CheckpointConfig config = env.getCheckpointConfig();
config.setMinPauseBetweenCheckpoints(30000);
config.setCheckpointTimeout(600000);
config.setMaxConcurrentCheckpoints(1);
config.setTolerableCheckpointFailureNumber(3);

// Retain checkpoints on cancellation
config.setExternalizedCheckpointRetention(
    ExternalizedCheckpointRetention.RETAIN_ON_CANCELLATION);

// State backend
env.setStateBackend(new EmbeddedRocksDBStateBackend());
config.setCheckpointStorage("hdfs:///flink/checkpoints");
```

### Savepoint operations

```bash
# Trigger a savepoint
flink savepoint <jobId> hdfs:///flink/savepoints/

# Cancel with savepoint
flink cancel -s hdfs:///flink/savepoints/ <jobId>

# Resume from savepoint
flink run -s hdfs:///flink/savepoints/savepoint-abc123 \
  -c com.example.MyJob myapp.jar

# List savepoints
hdfs dfs -ls /flink/savepoints/

# Dispose a savepoint
flink savepoint -d hdfs:///flink/savepoints/savepoint-abc123
```

## Flink CLI

### Job management

```bash
# Submit a job
flink run -c com.example.MyJob \
  -p 8 \
  -d \
  myapp.jar --input kafka --output hdfs

# Submit in detached mode
flink run -d -c com.example.StreamJob app.jar

# List running jobs
flink list

# List all jobs (including finished)
flink list -a

# Cancel a job
flink cancel <jobId>

# Get job details
flink info myapp.jar

# Modify job parallelism (rescale with savepoint)
flink modify <jobId> -p 16
```

### Cluster management

```bash
# Start standalone cluster
$FLINK_HOME/bin/start-cluster.sh

# Stop cluster
$FLINK_HOME/bin/stop-cluster.sh

# Start/stop task managers
$FLINK_HOME/bin/taskmanager.sh start
$FLINK_HOME/bin/taskmanager.sh stop

# Web UI: http://jobmanager:8081
```

## Configuration

### flink-conf.yaml essentials

```yaml
# JobManager
jobmanager.rpc.address: jobmanager-host
jobmanager.rpc.port: 6123
jobmanager.memory.process.size: 4096m

# TaskManager
taskmanager.memory.process.size: 8192m
taskmanager.numberOfTaskSlots: 4
taskmanager.memory.managed.fraction: 0.4

# State backend
state.backend: rocksdb
state.backend.rocksdb.localdir: /tmp/flink-rocksdb
state.checkpoints.dir: hdfs:///flink/checkpoints
state.savepoints.dir: hdfs:///flink/savepoints

# Restart strategy
restart-strategy: exponential-delay
restart-strategy.exponential-delay.initial-backoff: 1s
restart-strategy.exponential-delay.max-backoff: 60s
restart-strategy.exponential-delay.backoff-multiplier: 2.0
restart-strategy.exponential-delay.reset-backoff-threshold: 120s

# Network
taskmanager.network.memory.fraction: 0.1
taskmanager.network.memory.min: 64mb
taskmanager.network.memory.max: 1gb

# Parallelism
parallelism.default: 4
```

## Tips

- Use RocksDB state backend for production jobs with large state; HashMapStateBackend is faster but limited by JVM heap
- Set watermark out-of-orderness based on actual data lateness observed in production, not arbitrary values
- Enable incremental checkpointing with RocksDB to reduce checkpoint time for large state jobs
- Use savepoints for planned maintenance and upgrades; checkpoints for automatic failure recovery
- Avoid using `KeyedProcessFunction` timers with very fine granularity; millions of timers degrade checkpoint performance
- Set `setMaxConcurrentCheckpoints(1)` to prevent checkpoint storms under backpressure
- Monitor backpressure via the Flink Web UI; it indicates slow operators that need more parallelism or optimization
- Use side outputs for late data rather than dropping it silently; route to a dead-letter topic for investigation
- When upgrading job logic, always take a savepoint first and use `--allowNonRestoredState` only as a last resort
- Configure restart strategy with exponential backoff to prevent rapid restart loops that overwhelm external systems
- Use `uid()` on every operator to enable savepoint compatibility across code changes

## See Also

- kafka-streams, spark, kafka, hadoop, yarn, airflow

## References

- [Apache Flink Documentation](https://nightlies.apache.org/flink/flink-docs-stable/)
- [Flink DataStream API](https://nightlies.apache.org/flink/flink-docs-stable/docs/dev/datastream/overview/)
- [Flink Checkpointing](https://nightlies.apache.org/flink/flink-docs-stable/docs/dev/datastream/fault-tolerance/checkpointing/)
- [Flink State Backends](https://nightlies.apache.org/flink/flink-docs-stable/docs/ops/state/state_backends/)
- [Flink Configuration Reference](https://nightlies.apache.org/flink/flink-docs-stable/docs/deployment/config/)
