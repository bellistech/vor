# Delta Lake (Lakehouse Storage Layer)

Open-source storage layer providing ACID transactions, scalable metadata, time travel, and schema enforcement on top of data lakes (Parquet + JSON transaction log).

## Installation

```bash
# PySpark
pip install delta-spark pyspark

# Python standalone (delta-rs)
pip install deltalake

# Spark session with Delta Lake
```

```python
from pyspark.sql import SparkSession
spark = SparkSession.builder \
    .appName("delta") \
    .config("spark.jars.packages", "io.delta:delta-spark_2.12:3.1.0") \
    .config("spark.sql.extensions", "io.delta.sql.DeltaSparkSessionExtension") \
    .config("spark.sql.catalog.spark_catalog",
            "org.apache.spark.sql.delta.catalog.DeltaCatalog") \
    .getOrCreate()
```

## Creating Delta Tables

```python
# Write DataFrame as Delta table
df.write.format("delta").save("/data/events")

# Write with partitioning
df.write.format("delta") \
    .partitionBy("date", "region") \
    .save("/data/events")

# Create table in metastore
df.write.format("delta").saveAsTable("events")
```

```sql
-- SQL: Create Delta table
CREATE TABLE events (
    event_id STRING,
    user_id STRING,
    event_type STRING,
    payload STRING,
    event_time TIMESTAMP,
    date DATE GENERATED ALWAYS AS (CAST(event_time AS DATE))
) USING DELTA
PARTITIONED BY (date)
LOCATION '/data/events';

-- Create from existing data
CREATE TABLE events USING DELTA AS
SELECT * FROM parquet.`/raw/events/`;

-- Convert Parquet to Delta (in-place)
CONVERT TO DELTA parquet.`/data/events/` PARTITIONED BY (date DATE);
```

## Read and Write

```python
# Read
df = spark.read.format("delta").load("/data/events")

# Append
new_data.write.format("delta").mode("append").save("/data/events")

# Overwrite
new_data.write.format("delta").mode("overwrite").save("/data/events")

# Overwrite specific partitions
new_data.write.format("delta") \
    .mode("overwrite") \
    .option("replaceWhere", "date = '2024-01-15'") \
    .save("/data/events")
```

## Merge (Upsert)

```python
from delta.tables import DeltaTable

target = DeltaTable.forPath(spark, "/data/events")
target.alias("t").merge(
    source=updates.alias("s"),
    condition="t.event_id = s.event_id"
).whenMatchedUpdateAll() \
 .whenNotMatchedInsertAll() \
 .execute()
```

```sql
-- SQL merge
MERGE INTO events AS t
USING updates AS s
ON t.event_id = s.event_id
WHEN MATCHED THEN UPDATE SET *
WHEN NOT MATCHED THEN INSERT *
WHEN NOT MATCHED BY SOURCE AND t.date < '2024-01-01'
    THEN DELETE;
```

## Time Travel

```python
# Read specific version
df = spark.read.format("delta").option("versionAsOf", 5).load("/data/events")

# Read at specific timestamp
df = spark.read.format("delta") \
    .option("timestampAsOf", "2024-01-15 10:00:00") \
    .load("/data/events")
```

```sql
-- SQL time travel
SELECT * FROM events VERSION AS OF 5;
SELECT * FROM events TIMESTAMP AS OF '2024-01-15 10:00:00';

-- Show version history
DESCRIBE HISTORY events;
DESCRIBE HISTORY events LIMIT 10;
```

## Schema Enforcement and Evolution

```python
# Schema enforcement is automatic — mismatched schema raises error

# Schema evolution (add new columns)
df.write.format("delta") \
    .mode("append") \
    .option("mergeSchema", "true") \
    .save("/data/events")
```

```sql
-- Enable auto schema evolution
SET spark.databricks.delta.schema.autoMerge.enabled = true;

-- Add column
ALTER TABLE events ADD COLUMNS (source STRING AFTER event_type);

-- Rename column
ALTER TABLE events RENAME COLUMN payload TO event_payload;

-- Change column type (safe widening only)
ALTER TABLE events ALTER COLUMN user_id TYPE BIGINT;
```

## OPTIMIZE and Z-ORDER

```sql
-- Compact small files into larger ones
OPTIMIZE events;

-- Compact specific partition
OPTIMIZE events WHERE date = '2024-01-15';

-- Z-ORDER for multi-dimensional clustering
OPTIMIZE events ZORDER BY (user_id, event_type);

-- Z-ORDER on specific partition
OPTIMIZE events WHERE date >= '2024-01-01' ZORDER BY (user_id);
```

## VACUUM

```sql
-- Remove old files no longer referenced (default 7-day retention)
VACUUM events;

-- Aggressive vacuum (retain only 24 hours)
VACUUM events RETAIN 24 HOURS;

-- Dry run (show files to delete)
VACUUM events DRY RUN;

-- Disable retention check (DANGEROUS — breaks time travel)
SET spark.databricks.delta.retentionDurationCheck.enabled = false;
VACUUM events RETAIN 0 HOURS;
```

## Deletion Vectors

```sql
-- Enable deletion vectors (mark rows as deleted without rewriting files)
ALTER TABLE events SET TBLPROPERTIES ('delta.enableDeletionVectors' = true);

-- Deletes now use deletion vectors (faster, less I/O)
DELETE FROM events WHERE event_type = 'debug';

-- Purge deletion vectors (rewrite files to physically remove rows)
REORG TABLE events APPLY (PURGE);
```

## Liquid Clustering

```sql
-- Replace partitioning + Z-ORDER with liquid clustering
CREATE TABLE events_v2 (
    event_id STRING,
    user_id STRING,
    event_type STRING,
    event_time TIMESTAMP
) USING DELTA
CLUSTER BY (user_id, event_type);

-- Change clustering columns without rewrite
ALTER TABLE events_v2 CLUSTER BY (event_time, user_id);

-- Trigger clustering
OPTIMIZE events_v2;

-- Remove clustering
ALTER TABLE events_v2 CLUSTER BY NONE;
```

## Change Data Feed (CDF)

```sql
-- Enable change data feed
ALTER TABLE events SET TBLPROPERTIES ('delta.enableChangeDataFeed' = true);

-- Read changes between versions
SELECT * FROM table_changes('events', 2, 5);

-- Read changes by timestamp
SELECT * FROM table_changes('events',
    '2024-01-15 00:00:00', '2024-01-16 00:00:00');
```

```python
# Python CDF read
changes = spark.read.format("delta") \
    .option("readChangeFeed", "true") \
    .option("startingVersion", 2) \
    .option("endingVersion", 5) \
    .table("events")

# Streaming CDF
stream = spark.readStream.format("delta") \
    .option("readChangeFeed", "true") \
    .option("startingVersion", 0) \
    .table("events")
```

## Table Properties and Metadata

```sql
-- Show table details
DESCRIBE DETAIL events;
DESCRIBE EXTENDED events;

-- Table properties
ALTER TABLE events SET TBLPROPERTIES (
    'delta.logRetentionDuration' = 'interval 30 days',
    'delta.deletedFileRetentionDuration' = 'interval 7 days',
    'delta.autoOptimize.optimizeWrite' = 'true',
    'delta.autoOptimize.autoCompact' = 'true',
    'delta.dataSkippingNumIndexedCols' = 8
);

-- Show properties
SHOW TBLPROPERTIES events;
```

## delta-rs (Rust/Python Standalone)

```python
from deltalake import DeltaTable, write_deltalake
import pandas as pd

# Read without Spark
dt = DeltaTable("/data/events")
df = dt.to_pandas()

# Write from Pandas
write_deltalake("/data/events", df, mode="append")

# History and vacuum
dt.history()
dt.vacuum(retention_hours=168, dry_run=False)

# Version info
print(dt.version())
print(dt.metadata())
print(dt.schema())
```

## Tips

- Use liquid clustering over partitioning + Z-ORDER for new tables; it adapts without data rewrites
- Run OPTIMIZE regularly to compact small files; aim for 256 MB - 1 GB file sizes
- Set VACUUM retention to match your time travel needs; 7 days is the safe default
- Enable deletion vectors for tables with frequent DELETE/UPDATE to avoid costly file rewrites
- Use Change Data Feed for CDC pipelines instead of diffing snapshots
- Partition only by low-cardinality columns (date, region) to avoid small file problems
- Enable autoOptimize.optimizeWrite for streaming workloads to reduce small files
- Schema enforcement catches type mismatches at write time; use mergeSchema for intentional evolution
- Use DESCRIBE HISTORY to audit who changed what and when
- Combine time travel with MERGE for idempotent pipeline reruns
- Set dataSkippingNumIndexedCols to cover your most common filter columns
- Use delta-rs for lightweight reads outside Spark (Python, Rust) without JVM overhead

## See Also

- sql
- postgresql

## References

- [Delta Lake Documentation](https://docs.delta.io/latest/index.html)
- [Delta Lake GitHub](https://github.com/delta-io/delta)
- [delta-rs (Rust/Python)](https://github.com/delta-io/delta-rs)
- [Liquid Clustering](https://docs.delta.io/latest/delta-clustering.html)
- [Delta Lake Paper — VLDB 2020](https://www.vldb.org/pvldb/vol13/p3411-armbrust.pdf)
