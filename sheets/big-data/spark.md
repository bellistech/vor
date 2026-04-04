# Apache Spark (Unified Analytics Engine)

Apache Spark is a multi-language engine for large-scale data processing, providing in-memory computation for batch, streaming, SQL, machine learning, and graph workloads with up to 100x faster performance than disk-based MapReduce.

## Spark Submit

### Launching applications

```bash
# Local mode (development)
spark-submit --master local[4] \
  --class com.example.MyApp \
  target/myapp-1.0.jar \
  --input /data/input --output /data/output

# YARN cluster mode
spark-submit --master yarn \
  --deploy-mode cluster \
  --num-executors 20 \
  --executor-memory 8g \
  --executor-cores 4 \
  --driver-memory 4g \
  --conf spark.dynamicAllocation.enabled=true \
  --class com.example.ETLJob \
  s3://bucket/jars/etl-2.0.jar

# YARN client mode (for interactive debugging)
spark-submit --master yarn \
  --deploy-mode client \
  --num-executors 10 \
  --executor-memory 4g \
  app.py

# Kubernetes
spark-submit --master k8s://https://k8s-api:6443 \
  --deploy-mode cluster \
  --conf spark.kubernetes.container.image=myrepo/spark:3.5 \
  --conf spark.kubernetes.namespace=spark-jobs \
  local:///opt/spark/work-dir/app.py

# PySpark with packages
spark-submit --master local[*] \
  --packages org.apache.spark:spark-sql-kafka-0-10_2.12:3.5.0 \
  --py-files utils.zip \
  stream_processor.py
```

## Spark SQL and DataFrames

### DataFrame operations

```python
from pyspark.sql import SparkSession
from pyspark.sql import functions as F
from pyspark.sql.window import Window

spark = SparkSession.builder \
    .appName("ETL Pipeline") \
    .config("spark.sql.adaptive.enabled", "true") \
    .config("spark.sql.adaptive.coalescePartitions.enabled", "true") \
    .getOrCreate()

# Read data
df = spark.read.parquet("s3://bucket/data/events/")
csv_df = spark.read.csv("data.csv", header=True, inferSchema=True)
json_df = spark.read.json("data.jsonl")

# Basic transformations
result = df \
    .filter(F.col("event_date") >= "2024-01-01") \
    .withColumn("year", F.year("event_date")) \
    .withColumn("revenue_usd", F.col("amount") * F.col("exchange_rate")) \
    .groupBy("year", "category") \
    .agg(
        F.count("*").alias("event_count"),
        F.sum("revenue_usd").alias("total_revenue"),
        F.avg("revenue_usd").alias("avg_revenue"),
        F.approx_count_distinct("user_id").alias("unique_users")
    ) \
    .orderBy(F.desc("total_revenue"))

# Window functions
window_spec = Window.partitionBy("category").orderBy(F.desc("total_revenue"))
ranked = result.withColumn("rank", F.row_number().over(window_spec))

# Write output
result.write \
    .mode("overwrite") \
    .partitionBy("year") \
    .parquet("s3://bucket/output/summary/")

# Write to a single CSV file
result.coalesce(1).write \
    .mode("overwrite") \
    .option("header", "true") \
    .csv("output/report/")
```

### SQL interface

```python
# Register temp view
df.createOrReplaceTempView("events")

# Run SQL
top_users = spark.sql("""
    SELECT user_id,
           COUNT(*) as event_count,
           SUM(amount) as total_spent,
           PERCENTILE_APPROX(amount, 0.95) as p95_amount
    FROM events
    WHERE event_date >= '2024-01-01'
    GROUP BY user_id
    HAVING COUNT(*) > 10
    ORDER BY total_spent DESC
    LIMIT 100
""")
```

## RDD Operations

### Core RDD API

```python
# Create RDDs
rdd = sc.textFile("hdfs:///data/logs/*.log")
rdd = sc.parallelize(range(1000000), numSlices=100)

# Transformations (lazy)
words = rdd.flatMap(lambda line: line.split()) \
           .map(lambda word: (word, 1)) \
           .reduceByKey(lambda a, b: a + b) \
           .filter(lambda pair: pair[1] > 5) \
           .sortBy(lambda pair: pair[1], ascending=False)

# Actions (trigger execution)
top_10 = words.take(10)
total = words.count()
words.saveAsTextFile("hdfs:///output/wordcount/")

# Partition control
rdd.repartition(200)      # shuffle to exact partition count
rdd.coalesce(50)           # reduce partitions without full shuffle
rdd.getNumPartitions()     # check current partition count
```

## Caching and Persistence

### Storage levels

```python
from pyspark import StorageLevel

# Cache in memory (deserialized)
df.cache()  # equivalent to persist(MEMORY_ONLY)

# Persist with specific level
df.persist(StorageLevel.MEMORY_AND_DISK)
df.persist(StorageLevel.MEMORY_ONLY_SER)    # serialized, less memory
df.persist(StorageLevel.DISK_ONLY)
df.persist(StorageLevel.OFF_HEAP)

# Unpersist when done
df.unpersist()

# Check if cached
df.is_cached

# Check storage level
df.storageLevel
```

## Configuration

### Key tuning parameters

```bash
# Memory settings
--conf spark.executor.memory=8g
--conf spark.executor.memoryOverhead=2g
--conf spark.driver.memory=4g
--conf spark.memory.fraction=0.6
--conf spark.memory.storageFraction=0.5

# Parallelism
--conf spark.sql.shuffle.partitions=200
--conf spark.default.parallelism=200
--conf spark.sql.adaptive.enabled=true
--conf spark.sql.adaptive.coalescePartitions.enabled=true
--conf spark.sql.adaptive.skewJoin.enabled=true

# Shuffle
--conf spark.shuffle.compress=true
--conf spark.shuffle.spill.compress=true
--conf spark.sql.shuffle.partitions=auto

# Serialization
--conf spark.serializer=org.apache.spark.serializer.KryoSerializer
--conf spark.kryoserializer.buffer.max=128m

# Dynamic allocation
--conf spark.dynamicAllocation.enabled=true
--conf spark.dynamicAllocation.minExecutors=5
--conf spark.dynamicAllocation.maxExecutors=100
--conf spark.dynamicAllocation.executorIdleTimeout=60s

# Speculation (relaunch slow tasks)
--conf spark.speculation=true
--conf spark.speculation.multiplier=1.5
--conf spark.speculation.quantile=0.9
```

## Structured Streaming

### Stream processing

```python
# Read from Kafka
stream_df = spark.readStream \
    .format("kafka") \
    .option("kafka.bootstrap.servers", "broker:9092") \
    .option("subscribe", "events") \
    .option("startingOffsets", "latest") \
    .load()

# Parse and transform
from pyspark.sql.types import StructType, StringType, TimestampType

schema = StructType() \
    .add("user_id", StringType()) \
    .add("event", StringType()) \
    .add("timestamp", TimestampType())

parsed = stream_df \
    .select(F.from_json(F.col("value").cast("string"), schema).alias("data")) \
    .select("data.*") \
    .withWatermark("timestamp", "10 minutes") \
    .groupBy(
        F.window("timestamp", "5 minutes"),
        "event"
    ).count()

# Write stream
query = parsed.writeStream \
    .outputMode("update") \
    .format("console") \
    .option("checkpointLocation", "/checkpoints/event_counts") \
    .trigger(processingTime="30 seconds") \
    .start()

query.awaitTermination()
```

## Monitoring

### Spark UI and metrics

```bash
# Spark UI (driver)
# http://driver-host:4040

# History Server
$SPARK_HOME/sbin/start-history-server.sh
# http://history-host:18080

# REST API for application info
curl http://driver-host:4040/api/v1/applications
curl http://driver-host:4040/api/v1/applications/{appId}/stages
curl http://driver-host:4040/api/v1/applications/{appId}/executors

# Enable event logging for history server
--conf spark.eventLog.enabled=true
--conf spark.eventLog.dir=hdfs:///spark-logs/
```

## Tips

- Use Adaptive Query Execution (AQE) in Spark 3.x to auto-optimize shuffle partitions and handle skew
- Prefer DataFrames over RDDs; the Catalyst optimizer generates significantly better execution plans
- Set `spark.sql.shuffle.partitions` based on data size, not the default 200; aim for 128-256 MB per partition
- Cache DataFrames only when reused multiple times; unnecessary caching wastes executor memory
- Use `coalesce()` instead of `repartition()` when reducing partition count to avoid a full shuffle
- Enable Kryo serialization for 2-10x faster serialization than Java default
- Broadcast small tables in joins using `F.broadcast(small_df)` to avoid shuffle
- Monitor the Spark UI Stages tab for skewed tasks (one task taking much longer than others)
- Use `spark.sql.files.maxPartitionBytes` to control input split size for file-based sources
- Write output in Parquet or ORC with snappy compression for best read performance downstream
- Set `spark.executor.memoryOverhead` to at least 10% of executor memory for PySpark jobs
- Use checkpointing in long lineage chains to truncate the DAG and prevent stack overflows

## See Also

- hadoop, yarn, hive, kafka-streams, flink, parquet

## References

- [Apache Spark Documentation](https://spark.apache.org/docs/latest/)
- [Spark SQL Guide](https://spark.apache.org/docs/latest/sql-programming-guide.html)
- [Spark Configuration Reference](https://spark.apache.org/docs/latest/configuration.html)
- [Structured Streaming Guide](https://spark.apache.org/docs/latest/structured-streaming-programming-guide.html)
- [Spark Tuning Guide](https://spark.apache.org/docs/latest/tuning.html)
