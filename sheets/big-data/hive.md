# Apache Hive (Data Warehouse on Hadoop)

Hive provides SQL-like query language (HiveQL) over large datasets stored in HDFS, translating queries into MapReduce, Tez, or Spark jobs with support for partitioning, bucketing, and columnar storage formats like ORC and Parquet.

## HiveQL Basics

### Database and table operations

```sql
-- Create database
CREATE DATABASE IF NOT EXISTS analytics
  COMMENT 'Analytics data warehouse'
  LOCATION '/warehouse/analytics';

-- Use database
USE analytics;

-- Create managed table
CREATE TABLE events (
  event_id    STRING,
  user_id     STRING,
  event_type  STRING,
  payload     STRING,
  amount      DOUBLE,
  created_at  TIMESTAMP
)
COMMENT 'Raw event data'
STORED AS ORC
TBLPROPERTIES ('orc.compress' = 'SNAPPY');

-- Create external table (data managed outside Hive)
CREATE EXTERNAL TABLE raw_logs (
  log_line STRING
)
ROW FORMAT DELIMITED FIELDS TERMINATED BY '\t'
STORED AS TEXTFILE
LOCATION '/data/raw/logs/';

-- Create table as select (CTAS)
CREATE TABLE daily_summary
STORED AS ORC AS
SELECT date_trunc('day', created_at) AS event_date,
       event_type,
       COUNT(*) AS event_count,
       SUM(amount) AS total_amount
FROM events
GROUP BY date_trunc('day', created_at), event_type;

-- Drop table
DROP TABLE IF EXISTS staging_events;

-- Alter table
ALTER TABLE events ADD COLUMNS (source STRING COMMENT 'Event source');
ALTER TABLE events SET TBLPROPERTIES ('auto.purge' = 'true');
ALTER TABLE events RENAME TO raw_events;

-- Show table details
DESCRIBE FORMATTED events;
SHOW CREATE TABLE events;
SHOW TABLES IN analytics;
```

## Partitioning

### Static and dynamic partitions

```sql
-- Create partitioned table
CREATE TABLE events_partitioned (
  event_id    STRING,
  user_id     STRING,
  event_type  STRING,
  amount      DOUBLE,
  created_at  TIMESTAMP
)
PARTITIONED BY (event_date STRING, region STRING)
STORED AS ORC;

-- Add partition manually (static)
ALTER TABLE events_partitioned
  ADD PARTITION (event_date='2024-01-15', region='us-east');

-- Enable dynamic partitioning
SET hive.exec.dynamic.partition=true;
SET hive.exec.dynamic.partition.mode=nonstrict;
SET hive.exec.max.dynamic.partitions=10000;
SET hive.exec.max.dynamic.partitions.pernode=1000;

-- Insert with dynamic partitions
INSERT OVERWRITE TABLE events_partitioned
PARTITION (event_date, region)
SELECT event_id, user_id, event_type, amount, created_at,
       date_format(created_at, 'yyyy-MM-dd') AS event_date,
       region
FROM raw_events;

-- Show partitions
SHOW PARTITIONS events_partitioned;

-- Drop a specific partition
ALTER TABLE events_partitioned
  DROP IF EXISTS PARTITION (event_date='2024-01-01', region='us-east');

-- Repair partitions (sync metastore with HDFS)
MSCK REPAIR TABLE events_partitioned;
```

## Bucketing

### Hash-based data organization

```sql
-- Create bucketed table
CREATE TABLE user_events_bucketed (
  event_id    STRING,
  user_id     STRING,
  event_type  STRING,
  amount      DOUBLE
)
PARTITIONED BY (event_date STRING)
CLUSTERED BY (user_id) SORTED BY (event_id) INTO 32 BUCKETS
STORED AS ORC;

-- Enable bucketing enforcement
SET hive.enforce.bucketing=true;
SET hive.enforce.sorting=true;

-- Insert into bucketed table
INSERT OVERWRITE TABLE user_events_bucketed
PARTITION (event_date='2024-01-15')
SELECT event_id, user_id, event_type, amount
FROM raw_events
WHERE date_format(created_at, 'yyyy-MM-dd') = '2024-01-15';

-- Bucket map join (both tables bucketed on join key)
SET hive.optimize.bucketmapjoin=true;
SET hive.optimize.bucketmapjoin.sortedmerge=true;
```

## File Formats and SerDe

### Format comparison and usage

```sql
-- ORC (Optimized Row Columnar) — best for Hive
CREATE TABLE orc_table (id INT, name STRING, value DOUBLE)
STORED AS ORC
TBLPROPERTIES (
  'orc.compress' = 'ZLIB',        -- or SNAPPY, LZO, NONE
  'orc.stripe.size' = '67108864', -- 64 MB stripes
  'orc.row.index.stride' = '10000',
  'orc.bloom.filter.columns' = 'name'
);

-- Parquet — best for cross-engine compatibility
CREATE TABLE parquet_table (id INT, name STRING, value DOUBLE)
STORED AS PARQUET
TBLPROPERTIES ('parquet.compression' = 'SNAPPY');

-- Avro with schema
CREATE TABLE avro_table
ROW FORMAT SERDE 'org.apache.hadoop.hive.serde2.avro.AvroSerDe'
STORED AS AVRO
TBLPROPERTIES ('avro.schema.url' = 'hdfs:///schemas/events.avsc');

-- JSON SerDe
CREATE EXTERNAL TABLE json_events (
  event_id STRING,
  user_id  STRING,
  metadata MAP<STRING, STRING>
)
ROW FORMAT SERDE 'org.apache.hive.hcatalog.data.JsonSerDe'
STORED AS TEXTFILE
LOCATION '/data/json/events/';

-- CSV with OpenCSV SerDe
CREATE EXTERNAL TABLE csv_data (
  col1 STRING, col2 STRING, col3 STRING
)
ROW FORMAT SERDE 'org.apache.hadoop.hive.serde2.OpenCSVSerde'
WITH SERDEPROPERTIES (
  'separatorChar' = ',',
  'quoteChar' = '"',
  'escapeChar' = '\\'
)
STORED AS TEXTFILE
LOCATION '/data/csv/';

-- Convert between formats
INSERT OVERWRITE TABLE orc_table
SELECT * FROM csv_data;
```

## Query Optimization

### Execution engine and join hints

```sql
-- Use Tez engine (much faster than MapReduce)
SET hive.execution.engine=tez;

-- Enable vectorized execution
SET hive.vectorized.execution.enabled=true;
SET hive.vectorized.execution.reduce.enabled=true;

-- Cost-based optimizer
SET hive.cbo.enable=true;
SET hive.compute.query.using.stats=true;
SET hive.stats.fetch.column.stats=true;
SET hive.stats.fetch.partition.stats=true;

-- Compute statistics for CBO
ANALYZE TABLE events COMPUTE STATISTICS;
ANALYZE TABLE events COMPUTE STATISTICS FOR COLUMNS;
ANALYZE TABLE events_partitioned PARTITION (event_date) COMPUTE STATISTICS;

-- Map join (broadcast small table)
SET hive.auto.convert.join=true;
SET hive.mapjoin.smalltable.filesize=25000000; -- 25 MB threshold

-- Explicit map join hint
SELECT /*+ MAPJOIN(d) */
  e.event_id, d.dimension_name
FROM events e
JOIN dimensions d ON e.dim_id = d.id;

-- Predicate pushdown
SET hive.optimize.ppd=true;
SET hive.optimize.ppd.storage=true;

-- Merge small files
SET hive.merge.mapfiles=true;
SET hive.merge.mapredfiles=true;
SET hive.merge.size.per.task=256000000;
SET hive.merge.smallfiles.avgsize=16000000;
```

## LLAP (Live Long and Process)

### Interactive query daemon

```bash
# Start LLAP daemons
hive --service llap --name llap0 \
  --instances 10 \
  --cache 50g \
  --size 60g \
  --executors 4 \
  --xmx 40g \
  --args "-XX:+UseG1GC" \
  --startImmediately

# Check LLAP status
hive --service llapstatus -n llap0

# Connect to LLAP via Beeline
beeline -u "jdbc:hive2://hiveserver2:10000/default;transportMode=binary" \
  -n hive -p password
```

## Metastore

### Configuration and management

```bash
# Initialize metastore schema (first time)
schematool -dbType mysql -initSchema

# Upgrade metastore schema
schematool -dbType mysql -upgradeSchema

# Validate metastore schema
schematool -dbType mysql -validate
```

```xml
<!-- hive-site.xml metastore config -->
<property>
  <name>javax.jdo.option.ConnectionURL</name>
  <value>jdbc:mysql://metastore-db:3306/hive_metastore?useSSL=true</value>
</property>
<property>
  <name>javax.jdo.option.ConnectionDriverName</name>
  <value>com.mysql.cj.jdbc.Driver</value>
</property>
<property>
  <name>hive.metastore.warehouse.dir</name>
  <value>/warehouse</value>
</property>
<property>
  <name>hive.metastore.uris</name>
  <value>thrift://metastore-host:9083</value>
</property>
```

## Beeline CLI

### Connection and usage

```bash
# Connect to HiveServer2
beeline -u "jdbc:hive2://hiveserver2:10000/default" -n user -p pass

# Execute query from command line
beeline -u "jdbc:hive2://localhost:10000/default" \
  -e "SELECT COUNT(*) FROM events WHERE event_date='2024-01-15'"

# Execute SQL file
beeline -u "jdbc:hive2://localhost:10000/default" \
  -f /path/to/query.sql

# Output to CSV
beeline -u "jdbc:hive2://localhost:10000/default" \
  --outputformat=csv2 \
  -e "SELECT * FROM summary" > output.csv

# Silent mode (no headers, no progress)
beeline -u "jdbc:hive2://localhost:10000/default" \
  --silent=true --showHeader=false \
  -e "SELECT count(*) FROM events"
```

## Tips

- Always use ORC with Snappy compression for Hive-native tables; it provides predicate pushdown and vectorized reads
- Partition by date columns used in WHERE clauses; over-partitioning (millions of partitions) degrades metastore performance
- Run `ANALYZE TABLE ... COMPUTE STATISTICS FOR COLUMNS` after loading data to enable cost-based query optimization
- Use bucketing on join keys when two large tables are frequently joined; sort-merge bucket joins avoid full shuffles
- Set `hive.execution.engine=tez` instead of MapReduce for 3-10x faster query execution
- Enable LLAP for sub-second interactive queries on frequently accessed datasets
- Use external tables for raw/landing data and managed tables for curated/processed data
- Prefer `INSERT OVERWRITE` over `INSERT INTO` for partition-level idempotent loads
- Use `MSCK REPAIR TABLE` when partitions are added to HDFS outside of Hive (e.g., by Spark or distcp)
- Set `hive.exec.parallel=true` to execute independent stages of a query concurrently
- Use bloom filter indexes in ORC tables on high-cardinality columns used in point lookups
- Enable `hive.vectorized.execution.enabled` for 2-5x speedup on ORC-backed queries

## See Also

- hadoop, spark, yarn, hive-metastore, parquet, orc, tez

## References

- [Apache Hive Documentation](https://hive.apache.org/)
- [Hive Language Manual](https://cwiki.apache.org/confluence/display/Hive/LanguageManual)
- [ORC File Format Specification](https://orc.apache.org/specification/)
- [Hive Performance Tuning](https://cwiki.apache.org/confluence/display/Hive/Configuration+Properties)
- [LLAP Documentation](https://cwiki.apache.org/confluence/display/Hive/LLAP)
