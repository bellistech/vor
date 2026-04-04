# Apache Parquet (Columnar Storage)

> Columnar file format for analytics — organizes data by columns instead of rows, enabling aggressive compression, predicate pushdown, and column pruning that make it the standard storage format for data lakes, Spark, Hive, DuckDB, and virtually every big data query engine.

## File Structure

### Anatomy of a Parquet File

```
┌──────────────────────────┐
│       Magic: PAR1         │  4 bytes
├──────────────────────────┤
│     Row Group 1           │
│  ┌────────────────────┐  │
│  │ Column Chunk (col1) │  │
│  │  ├─ Page 1 (data)   │  │
│  │  ├─ Page 2 (data)   │  │
│  │  └─ Page 3 (dict)   │  │
│  ├────────────────────┤  │
│  │ Column Chunk (col2) │  │
│  │  ├─ Page 1 (data)   │  │
│  │  └─ Page 2 (data)   │  │
│  └────────────────────┘  │
├──────────────────────────┤
│     Row Group 2           │
│  ┌────────────────────┐  │
│  │ Column Chunk (col1) │  │
│  │ Column Chunk (col2) │  │
│  └────────────────────┘  │
├──────────────────────────┤
│    Footer (metadata)      │
│  ├─ Schema                │
│  ├─ Row group metadata    │
│  ├─ Column chunk offsets  │
│  └─ Statistics            │
├──────────────────────────┤
│   Footer Length (4 bytes) │
│   Magic: PAR1 (4 bytes)  │
└──────────────────────────┘
```

### Row Groups

```
Row Group Size:
- Default: 128 MB (configurable)
- Controls parallelism — each row group can be processed independently
- Larger = better compression, fewer metadata entries
- Smaller = lower memory footprint, finer-grained predicate pushdown

# Spark configuration
spark.sql.parquet.rowGroupSize = 134217728  # 128 MB
spark.sql.files.maxRecordsPerFile = 0       # unlimited
```

### Pages

```
Page Types:
- Data Page:     Encoded column values (default 1 MB)
- Dictionary Page: Dictionary for dictionary encoding (one per column chunk)
- Index Page:    Column and offset indexes for page-level filtering

Page Size:
- Default: 1 MB
- Controls encoding granularity and I/O unit size
- Smaller pages = better predicate pushdown precision
- Larger pages = better compression ratio
```

## Encodings

### Dictionary Encoding (PLAIN_DICTIONARY)

```
How it works:
1. Build dictionary of unique values in column chunk
2. Replace each value with its dictionary index (integer)
3. Encode indices with RLE/bit-packing

Best for: Low-cardinality string columns (status, country, category)
Falls back to: PLAIN encoding when dictionary exceeds page size

Example:
  Values:  ["US", "UK", "US", "DE", "US", "UK", "US"]
  Dict:    {0: "US", 1: "UK", 2: "DE"}
  Encoded: [0, 1, 0, 2, 0, 1, 0]  ← integers, not strings
```

### RLE / Bit-Packing (RLE_DICTIONARY)

```
Run-Length Encoding:
  Input:  [0, 0, 0, 0, 1, 1, 2, 2, 2]
  RLE:    [(4, 0), (2, 1), (3, 2)]  ← (count, value)

Bit-Packing:
  If max dictionary index = 3 → need 2 bits per value
  8 values packed into 2 bytes instead of 8 bytes
  Compression: 4x for 2-bit values

Hybrid RLE/Bit-Pack:
  Parquet alternates between RLE runs (repeated values)
  and bit-packed groups (varying values) automatically
```

### Delta Encoding (DELTA_BINARY_PACKED)

```
For sorted or sequential integer columns:
  Values:  [1000, 1001, 1002, 1005, 1006]
  Deltas:  [1000, 1, 1, 3, 1]  ← first value + differences

DELTA_LENGTH_BYTE_ARRAY (for strings):
  Stores lengths as delta-encoded, then concatenated byte values

DELTA_BYTE_ARRAY:
  Prefix + suffix encoding for sorted string columns
  Values:  ["apple", "application", "apply"]
  Encoded: [("apple", ""), ("appl", "ication"), ("appl", "y")]
```

## Compression

### Compression Codecs

```bash
# Available codecs
UNCOMPRESSED   # no compression
SNAPPY         # fast, moderate ratio (default in many tools)
GZIP           # slow, high ratio
LZ4            # very fast, moderate ratio
ZSTD           # good balance of speed and ratio
BROTLI         # high ratio, slower than ZSTD

# Spark setting
spark.sql.parquet.compression.codec = zstd

# PyArrow
import pyarrow.parquet as pq
pq.write_table(table, 'data.parquet', compression='zstd',
               compression_level=3)
```

### Compression Comparison

```
Codec       Compress    Decompress    Ratio
────────────────────────────────────────────
Snappy      ~500 MB/s   ~1000 MB/s    2-3x
LZ4         ~700 MB/s   ~2000 MB/s    2-3x
ZSTD-3      ~250 MB/s   ~1000 MB/s    3.5-4.5x
GZIP-6      ~50 MB/s    ~300 MB/s     4-5x
```

## Predicate Pushdown

### How Statistics Enable Skipping

```
Parquet stores per-column-chunk and per-page statistics:
- min value
- max value
- null count
- distinct count (optional)

Query: SELECT * FROM data WHERE price > 100

Row Group 1: price min=10, max=80    → SKIP (max < 100)
Row Group 2: price min=50, max=200   → READ (range overlaps)
Row Group 3: price min=150, max=500  → READ (range overlaps)

Result: Only 2 of 3 row groups read from disk
```

### Bloom Filters

```
Parquet 2.0+ supports per-column Bloom filters:
- Probabilistic set membership for equality predicates
- Configurable false positive rate (default ~1%)
- Best for high-cardinality columns (user_id, session_id)

# PyArrow
pq.write_table(table, 'data.parquet',
               write_bloom_filter_for=['user_id'])
```

## Column Pruning

### Reading Only Needed Columns

```python
# PyArrow — read specific columns
import pyarrow.parquet as pq

# Read only 2 of 50 columns
table = pq.read_table('data.parquet', columns=['name', 'age'])

# Read with row group filtering
pf = pq.ParquetFile('data.parquet')
print(pf.metadata)                    # file metadata
print(pf.metadata.num_row_groups)     # number of row groups
print(pf.metadata.row_group(0))       # first row group stats

# Read single row group
table = pf.read_row_group(0, columns=['name', 'age'])

# Read with predicate (using filters)
table = pq.read_table('data.parquet',
    filters=[('age', '>', 25), ('country', '=', 'US')])
```

```sql
-- DuckDB — automatic column pruning
SELECT name, age FROM read_parquet('data.parquet')
WHERE country = 'US';
-- Only reads name, age, country columns; skips row groups via stats
```

## Nested Types

### Dremel Encoding (Repetition/Definition Levels)

```
Parquet uses Dremel-style encoding for nested/repeated fields:
- Repetition levels: 0 = new record, 1+ = repeated field depth
- Definition levels: how many optional/repeated ancestors are defined (non-null)

Example for repeated group A { optional string B }:
  Values: ["x", "y", "z"]
  Rep:    [0, 1, 1, 0, 0]
  Def:    [2, 1, 2, 0, 2]
```

## parquet-tools / parquet-cli

### File Inspection

```bash
# Install parquet-cli (Java)
# Via Homebrew (parquet-tools is deprecated, use parquet-cli)
brew install parquet-cli

# Python alternative
pip install parquet-tools

# Show schema
parquet schema data.parquet
# or
python -m parquet_tools schema data.parquet

# Show metadata (row groups, sizes, encodings)
parquet meta data.parquet

# Show first N rows
parquet head -n 10 data.parquet

# Show row count
parquet rowcount data.parquet

# Convert to JSON
parquet cat --json data.parquet

# Inspect column statistics
parquet column-index data.parquet

# Show per-column sizes
parquet column-size data.parquet
```

### DuckDB for Parquet

```sql
-- Read Parquet directly
SELECT * FROM read_parquet('data.parquet') LIMIT 10;

-- Get file metadata
SELECT * FROM parquet_metadata('data.parquet');

-- Get schema
SELECT * FROM parquet_schema('data.parquet');

-- Get per-row-group stats
SELECT * FROM parquet_file_metadata('data.parquet');

-- Write Parquet
COPY (SELECT * FROM my_table) TO 'output.parquet'
  (FORMAT PARQUET, CODEC 'ZSTD', ROW_GROUP_SIZE 100000);

-- Read from S3
SELECT * FROM read_parquet('s3://bucket/path/*.parquet');

-- Hive-partitioned datasets
SELECT * FROM read_parquet('data/*/*.parquet', hive_partitioning=true);
```

## Writing Parquet

### PyArrow

```python
import pyarrow as pa
import pyarrow.parquet as pq

# From Python dict
table = pa.table({
    'name': ['Alice', 'Bob', 'Carol'],
    'age': [30, 25, 35],
    'score': [95.5, 87.3, 91.8]
})

# Write with options
pq.write_table(table, 'output.parquet',
    compression='zstd',
    row_group_size=1_000_000,
    use_dictionary=True,
    write_statistics=True)

# Partitioned dataset
pq.write_to_dataset(table, 'output/',
    partition_cols=['country', 'year'])
```

## Tips

- Use ZSTD compression for the best balance of speed and ratio — it outperforms Snappy in almost every scenario
- Set row group size based on your query patterns: 128 MB for full scans, 32-64 MB for selective queries
- Sort data by commonly filtered columns before writing — this maximizes predicate pushdown effectiveness via min/max stats
- Use dictionary encoding (enabled by default) for string columns with fewer than ~100K unique values per row group
- Enable Bloom filters for equality predicates on high-cardinality columns like user_id or session_id
- Always write statistics (enabled by default in modern writers) — without them, predicate pushdown cannot work
- Partition datasets by date or category using Hive-style partitioning (`year=2026/month=04/`) for coarse-grained pruning
- Use DuckDB for quick ad-hoc Parquet analysis — it reads Parquet natively without loading into a database
- Monitor column sizes with `parquet column-size` to identify columns that compress poorly and may need type changes
- For nested data, flatten structures where possible — deeply nested Dremel encoding adds repetition/definition level overhead
- Prefer fewer large Parquet files over many small ones — small files cause excessive metadata overhead

## See Also

- avro, protobuf, json, yaml

## References

- [Apache Parquet Format Specification](https://parquet.apache.org/documentation/latest/)
- [Apache Parquet GitHub Repository](https://github.com/apache/parquet-format)
- [Dremel Paper: A Decade of Interactive SQL Analysis at Web Scale](https://research.google/pubs/dremel-a-decade-of-interactive-sql-analysis-at-web-scale/)
- [PyArrow Parquet Documentation](https://arrow.apache.org/docs/python/parquet.html)
- [DuckDB Parquet Documentation](https://duckdb.org/docs/data/parquet/overview.html)
- [Parquet Encoding Definitions](https://github.com/apache/parquet-format/blob/master/Encodings.md)
