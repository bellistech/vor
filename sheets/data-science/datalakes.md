# Data Lakes (Modern Lakehouse Architecture)

Data lakes are centralized repositories that store structured, semi-structured, and unstructured data at any scale, with modern lakehouse architectures combining data lake flexibility with data warehouse reliability through open table formats like Apache Iceberg, Delta Lake, and Apache Hudi, organized in medallion tiers for progressive data refinement.

## Medallion Architecture
### Bronze / Silver / Gold Tiers
```
Bronze (Raw/Landing)
  - Exact copy of source data, append-only
  - No transformations, no schema enforcement
  - Full history preserved for reprocessing
  - Formats: JSON, CSV, Avro, raw Parquet
  - Retention: indefinite (cold storage tier)

Silver (Cleansed/Conformed)
  - Deduplication, schema enforcement, type casting
  - Null handling, join enrichment, PII masking
  - Slowly Changing Dimensions (SCD Type 2)
  - Format: Parquet with table format (Iceberg/Delta)
  - Retention: 2-7 years (warm storage)

Gold (Business/Aggregated)
  - Business-level aggregations, KPIs, feature stores
  - Denormalized star schemas for BI consumption
  - Materialized views, summary tables
  - Format: Parquet with table format, optimized
  - Retention: rolling window (1-3 years)
```

### Medallion Implementation (PySpark)
```python
from pyspark.sql import SparkSession
from pyspark.sql.functions import col, current_timestamp, sha2, concat_ws

spark = SparkSession.builder \
    .appName("medallion-pipeline") \
    .config("spark.sql.extensions",
            "org.apache.iceberg.spark.extensions.IcebergSparkSessionExtensions") \
    .config("spark.sql.catalog.lakehouse",
            "org.apache.iceberg.spark.SparkCatalog") \
    .getOrCreate()

# Bronze: ingest raw data
bronze_df = spark.read.json("s3://raw-bucket/events/2024/01/")
bronze_df = bronze_df.withColumn("_ingested_at", current_timestamp()) \
                      .withColumn("_source_file", col("_metadata.file_path"))
bronze_df.writeTo("lakehouse.bronze.events").append()

# Silver: cleanse and conform
silver_df = spark.read.table("lakehouse.bronze.events") \
    .filter(col("event_id").isNotNull()) \
    .dropDuplicates(["event_id"]) \
    .withColumn("email_hash",
                sha2(col("email"), 256)) \
    .drop("email") \
    .withColumn("event_date",
                col("event_timestamp").cast("date"))

silver_df.writeTo("lakehouse.silver.events") \
    .tableProperty("write.merge.mode", "merge-on-read") \
    .overwritePartitions()

# Gold: business aggregations
gold_df = spark.sql("""
    SELECT
        event_date,
        event_type,
        COUNT(*) as event_count,
        COUNT(DISTINCT user_id) as unique_users,
        AVG(duration_ms) as avg_duration
    FROM lakehouse.silver.events
    WHERE event_date >= current_date() - INTERVAL 90 DAYS
    GROUP BY event_date, event_type
""")
gold_df.writeTo("lakehouse.gold.daily_metrics").overwritePartitions()
```

## Table Formats
### Apache Iceberg
```sql
-- Create Iceberg table
CREATE TABLE lakehouse.silver.orders (
    order_id    BIGINT,
    customer_id BIGINT,
    amount      DECIMAL(10, 2),
    status      STRING,
    order_date  DATE,
    updated_at  TIMESTAMP
)
USING iceberg
PARTITIONED BY (days(order_date))
TBLPROPERTIES (
    'format-version' = '2',
    'write.metadata.compression-codec' = 'gzip'
);

-- Schema evolution (add/rename/drop columns without rewrite)
ALTER TABLE lakehouse.silver.orders ADD COLUMN discount DECIMAL(5, 2);
ALTER TABLE lakehouse.silver.orders RENAME COLUMN status TO order_status;
ALTER TABLE lakehouse.silver.orders ALTER COLUMN amount TYPE DECIMAL(12, 2);

-- Time travel
SELECT * FROM lakehouse.silver.orders
  FOR SYSTEM_TIME AS OF TIMESTAMP '2024-01-15 10:00:00';

SELECT * FROM lakehouse.silver.orders
  FOR SYSTEM_VERSION AS OF 42;  -- snapshot ID

-- View snapshot history
SELECT * FROM lakehouse.silver.orders.snapshots;
SELECT * FROM lakehouse.silver.orders.history;
SELECT * FROM lakehouse.silver.orders.metadata_log_entries;
```

### Delta Lake
```sql
-- Create Delta table
CREATE TABLE silver.orders (
    order_id    BIGINT,
    customer_id BIGINT,
    amount      DECIMAL(10, 2),
    status      STRING,
    order_date  DATE
)
USING DELTA
PARTITIONED BY (order_date)
TBLPROPERTIES ('delta.enableChangeDataFeed' = 'true');

-- MERGE (upsert)
MERGE INTO silver.orders AS target
USING staging.new_orders AS source
ON target.order_id = source.order_id
WHEN MATCHED THEN UPDATE SET *
WHEN NOT MATCHED THEN INSERT *;

-- Time travel
SELECT * FROM silver.orders VERSION AS OF 10;
SELECT * FROM silver.orders TIMESTAMP AS OF '2024-01-15';

-- Change Data Feed
SELECT * FROM table_changes('silver.orders', 5, 10);

-- DESCRIBE HISTORY
DESCRIBE HISTORY silver.orders;
```

### Apache Hudi
```python
# Hudi write (Copy-on-Write table)
hudi_options = {
    'hoodie.table.name': 'orders',
    'hoodie.datasource.write.recordkey.field': 'order_id',
    'hoodie.datasource.write.precombine.field': 'updated_at',
    'hoodie.datasource.write.partitionpath.field': 'order_date',
    'hoodie.datasource.write.operation': 'upsert',
    'hoodie.datasource.write.table.type': 'COPY_ON_WRITE',
}

df.write.format("hudi") \
    .options(**hudi_options) \
    .mode("append") \
    .save("s3://lake/silver/orders")

# Incremental query (only changed records)
incremental_df = spark.read.format("hudi") \
    .option("hoodie.datasource.query.type", "incremental") \
    .option("hoodie.datasource.read.begin.instanttime", "20240115100000") \
    .load("s3://lake/silver/orders")
```

## Partitioning Strategies
### Choosing Partition Keys
```
Strategy              Use Case              Granularity
─────────────────────────────────────────────────────────
Date (daily)          Event logs, txns      ~1K-100K files/year
Date (monthly)        Slowly growing data   ~12 files/year
Region + Date         Multi-region data     Moderate cardinality
Bucket (hash)         High-cardinality ID   Fixed bucket count
Hidden partitioning   Iceberg only          Transforms on columns
  - days(ts)          Daily from timestamp
  - months(ts)        Monthly from timestamp
  - hours(ts)         Hourly from timestamp
  - bucket(N, col)    Hash into N buckets
  - truncate(L, col)  First L chars
```

```python
# Iceberg hidden partitioning (no user-visible partition column)
spark.sql("""
    CREATE TABLE lakehouse.silver.events (
        event_id    STRING,
        event_time  TIMESTAMP,
        user_id     STRING,
        payload     STRING
    ) USING iceberg
    PARTITIONED BY (days(event_time), bucket(16, user_id))
""")
```

## Compaction and Optimization
### File Management
```sql
-- Iceberg: compact small files
CALL lakehouse.system.rewrite_data_files(
    table => 'lakehouse.silver.events',
    strategy => 'binpack',
    options => map('target-file-size-bytes', '134217728')  -- 128MB
);

-- Iceberg: sort to improve query performance
CALL lakehouse.system.rewrite_data_files(
    table => 'lakehouse.silver.events',
    strategy => 'sort',
    sort_order => 'event_date ASC, user_id ASC'
);

-- Iceberg: expire old snapshots
CALL lakehouse.system.expire_snapshots(
    table => 'lakehouse.silver.events',
    older_than => TIMESTAMP '2024-01-01 00:00:00',
    retain_last => 10
);

-- Iceberg: remove orphan files
CALL lakehouse.system.remove_orphan_files(
    table => 'lakehouse.silver.events',
    older_than => TIMESTAMP '2024-01-01 00:00:00'
);

-- Delta Lake: optimize + Z-order
OPTIMIZE silver.events ZORDER BY (user_id, event_type);

-- Delta Lake: vacuum unreferenced files
VACUUM silver.events RETAIN 168 HOURS;
```

## Storage Layers
### Cloud Object Storage Configuration
```bash
# AWS S3
aws s3 ls s3://data-lake-bucket/silver/events/ --recursive --summarize

# Lifecycle policy for tiering
cat <<'EOF' > lifecycle.json
{
  "Rules": [{
    "ID": "BronzeToGlacier",
    "Filter": {"Prefix": "bronze/"},
    "Status": "Enabled",
    "Transitions": [
      {"Days": 90, "StorageClass": "GLACIER_IR"},
      {"Days": 365, "StorageClass": "DEEP_ARCHIVE"}
    ]
  }, {
    "ID": "SilverToIA",
    "Filter": {"Prefix": "silver/"},
    "Status": "Enabled",
    "Transitions": [
      {"Days": 180, "StorageClass": "STANDARD_IA"}
    ]
  }]
}
EOF
aws s3api put-bucket-lifecycle-configuration \
  --bucket data-lake-bucket \
  --lifecycle-configuration file://lifecycle.json

# Azure ADLS Gen2
az storage fs directory list \
  --file-system lake \
  --path silver/events \
  --account-name mystorageaccount

# GCS
gsutil ls -l gs://data-lake-bucket/silver/events/
```

## Query Engines
### Engine Comparison
```
Engine          Strengths                    Best For
──────────────────────────────────────────────────────────
Trino/Presto    Federation, ANSI SQL         Ad-hoc analytics, cross-source
Spark SQL       Batch + streaming, ML        ETL, large-scale transforms
DuckDB          In-process, zero config      Local analysis, small-medium data
Athena          Serverless, pay-per-query     AWS-native ad-hoc queries
BigQuery        Serverless, massive scale     GCP analytics, ML integration
Redshift        Cluster-based warehouse       AWS structured analytics
Snowflake       Multi-cloud, time travel      Cross-cloud analytics
Databricks      Unified analytics + ML        Full lakehouse platform
```

```sql
-- Trino: query across catalogs
SELECT
    o.order_id,
    c.customer_name,
    p.product_name
FROM iceberg.silver.orders o
JOIN postgresql.public.customers c ON o.customer_id = c.id
JOIN mysql.inventory.products p ON o.product_id = p.id
WHERE o.order_date >= DATE '2024-01-01';
```

## Tips
- Start with a single table format (Iceberg is the most vendor-neutral) rather than mixing formats within the same lake
- Set target file sizes between 128MB and 512MB for Parquet files to balance query parallelism with overhead
- Use hidden partitioning (Iceberg) instead of Hive-style partitioning to avoid user-facing partition columns in queries
- Schedule compaction jobs during off-peak hours to merge small files without impacting query performance
- Implement schema evolution at the silver layer; bronze should always store raw data without schema constraints
- Use Change Data Capture (CDC) with merge operations for incremental silver layer updates instead of full reprocessing
- Enable column statistics and bloom filters on high-cardinality join columns to accelerate predicate pushdown
- Apply storage lifecycle policies to move bronze data to cold storage after 90 days to control costs
- Use catalog services (AWS Glue, Hive Metastore, Nessie, Polaris) for centralized metadata management
- Test time travel queries regularly to ensure snapshot retention policies support your audit and rollback requirements
- Separate compute and storage from the beginning; avoid tightly coupled architectures that prevent engine flexibility

## See Also
- pandas, numpy, spark, trino, parquet, avro

## References
- [Apache Iceberg Documentation](https://iceberg.apache.org/docs/latest/)
- [Delta Lake Documentation](https://docs.delta.io/latest/)
- [Apache Hudi Documentation](https://hudi.apache.org/docs/overview/)
- [Databricks Medallion Architecture](https://www.databricks.com/glossary/medallion-architecture)
- [Trino Documentation](https://trino.io/docs/current/)
