# Apache Beam (Unified Batch and Streaming)

Unified programming model for batch and streaming data processing, with portable pipelines that run on multiple execution engines (Flink, Spark, Dataflow, and more).

## Installation

```bash
# Python SDK
pip install apache-beam
pip install apache-beam[gcp]                     # Google Cloud Dataflow runner
pip install apache-beam[aws]                     # AWS runner
pip install apache-beam[interactive]             # Jupyter support

# Java SDK (Maven)
# <dependency>
#   <groupId>org.apache.beam</groupId>
#   <artifactId>beam-sdks-java-core</artifactId>
#   <version>2.55.0</version>
# </dependency>

# Go SDK
go get github.com/apache/beam/sdks/v2
```

## Basic Pipeline

```python
import apache_beam as beam

# Minimal pipeline
with beam.Pipeline() as p:
    (p
     | 'Read' >> beam.io.ReadFromText('input.txt')
     | 'Transform' >> beam.Map(str.upper)
     | 'Write' >> beam.io.WriteToText('output.txt'))

# With runner options
from apache_beam.options.pipeline_options import PipelineOptions
options = PipelineOptions([
    '--runner=DirectRunner',        # local execution
    '--direct_num_workers=4',
])
with beam.Pipeline(options=options) as p:
    # pipeline steps...
    pass
```

## Core Transforms

```python
# Map — apply function to each element
results = items | beam.Map(lambda x: x * 2)
results = items | beam.Map(process_fn, extra_arg)

# FlatMap — one-to-many mapping
words = lines | beam.FlatMap(lambda line: line.split())

# Filter
adults = people | beam.Filter(lambda p: p['age'] >= 18)

# ParDo — most general element-wise transform
class ProcessFn(beam.DoFn):
    def setup(self):
        """Called once per worker initialization."""
        self.client = connect_to_service()

    def process(self, element, timestamp=beam.DoFn.TimestampParam):
        """Yields zero or more output elements."""
        if element['value'] > threshold:
            yield element
        # Emit to tagged output
        yield beam.pvalue.TaggedOutput('errors', element)

results = items | beam.ParDo(ProcessFn()).with_outputs('errors', main='valid')
valid = results.valid
errors = results.errors
```

## GroupByKey and CoGroupByKey

```python
# GroupByKey — group values by key
# Input: [('cat', 1), ('dog', 2), ('cat', 3)]
# Output: [('cat', [1, 3]), ('dog', [2])]
grouped = pairs | beam.GroupByKey()

# CombinePerKey — aggregate per key (more efficient than GroupByKey)
totals = pairs | beam.CombinePerKey(sum)
stats = pairs | beam.CombinePerKey(
    beam.combiners.MeanCombineFn())

# CoGroupByKey — join two PCollections by key
joined = ({'emails': emails, 'phones': phones}
          | beam.CoGroupByKey())
# Output: [('user1', {'emails': ['a@b.com'], 'phones': ['555-1234']})]
```

## Combine (Global Aggregation)

```python
# Global combine
total = numbers | beam.CombineGlobally(sum)
mean = numbers | beam.CombineGlobally(beam.combiners.MeanCombineFn())

# Top N
top_10 = numbers | beam.combiners.Top.Of(10)
top_scores = records | beam.combiners.Top.Of(
    10, key=lambda x: x['score'])

# Count
counts = items | beam.combiners.Count.Globally()
per_key = items | beam.combiners.Count.PerKey()

# Custom CombineFn (associative + commutative)
class AverageFn(beam.CombineFn):
    def create_accumulator(self):
        return (0, 0)  # (sum, count)

    def add_input(self, accumulator, input):
        s, c = accumulator
        return s + input, c + 1

    def merge_accumulators(self, accumulators):
        sums, counts = zip(*accumulators)
        return sum(sums), sum(counts)

    def extract_output(self, accumulator):
        s, c = accumulator
        return s / c if c else 0

avg = numbers | beam.CombineGlobally(AverageFn())
```

## Windowing

```python
from apache_beam import window

# Fixed windows (tumbling) — non-overlapping
windowed = events | beam.WindowInto(
    window.FixedWindows(60))                     # 60-second windows

# Sliding windows — overlapping
windowed = events | beam.WindowInto(
    window.SlidingWindows(300, 60))              # 5-min window, 1-min slide

# Session windows — gap-based
windowed = events | beam.WindowInto(
    window.Sessions(600))                        # 10-min gap timeout

# Global window (default for batch)
windowed = events | beam.WindowInto(
    window.GlobalWindows())

# Window with allowed lateness
windowed = events | beam.WindowInto(
    window.FixedWindows(60),
    allowed_lateness=beam.Duration(seconds=3600))
```

## Triggers

```python
from apache_beam.transforms.trigger import (
    AfterWatermark, AfterProcessingTime, AfterCount,
    AccumulationMode, Repeatedly, AfterAny
)

# Default trigger (fire at watermark)
windowed = events | beam.WindowInto(
    window.FixedWindows(60),
    trigger=AfterWatermark(),
    accumulation_mode=AccumulationMode.DISCARDING)

# Early and late firings
windowed = events | beam.WindowInto(
    window.FixedWindows(60),
    trigger=AfterWatermark(
        early=AfterProcessingTime(10),           # early results every 10s
        late=AfterCount(1)),                     # fire on each late element
    accumulation_mode=AccumulationMode.ACCUMULATING)

# Repeatedly trigger
windowed = events | beam.WindowInto(
    window.FixedWindows(60),
    trigger=Repeatedly(AfterCount(100)),         # fire every 100 elements
    accumulation_mode=AccumulationMode.DISCARDING)
```

## Timestamps and Watermarks

```python
# Assign timestamps from data
timestamped = events | beam.Map(
    lambda e: beam.window.TimestampedValue(e, e['timestamp']))

# Custom timestamp function
class AddTimestamp(beam.DoFn):
    def process(self, element):
        ts = element['event_time']
        yield beam.window.TimestampedValue(element, ts)

timestamped = events | beam.ParDo(AddTimestamp())

# Watermark is set by the source — controls when windows close
# Custom sources can override estimate_watermark()
```

## Side Inputs

```python
# Pass additional data to transforms
config = p | beam.Create([{'threshold': 100}])

# As singleton
processed = main | beam.Map(
    lambda elem, cfg: process(elem, cfg),
    cfg=beam.pvalue.AsSingleton(config))

# As list
all_items = p | beam.Create([1, 2, 3, 4, 5])
filtered = main | beam.Filter(
    lambda elem, items: elem in items,
    items=beam.pvalue.AsList(all_items))

# As dict (from key-value pairs)
lookup = p | beam.Create([('k1', 'v1'), ('k2', 'v2')])
enriched = main | beam.Map(
    lambda elem, lut: {**elem, 'extra': lut.get(elem['key'])},
    lut=beam.pvalue.AsDict(lookup))
```

## I/O Connectors

```python
# Text files
lines = p | beam.io.ReadFromText('gs://bucket/input*.txt')
output | beam.io.WriteToText('gs://bucket/output', file_name_suffix='.txt')

# BigQuery
rows = p | beam.io.ReadFromBigQuery(
    query='SELECT * FROM dataset.table WHERE date > "2024-01-01"',
    use_standard_sql=True)
output | beam.io.WriteToBigQuery(
    'project:dataset.table',
    schema='name:STRING,value:FLOAT',
    write_disposition=beam.io.BigQueryDisposition.WRITE_APPEND)

# Kafka
messages = p | beam.io.ReadFromKafka(
    consumer_config={'bootstrap.servers': 'broker:9092'},
    topics=['events'])

# Pub/Sub
msgs = p | beam.io.ReadFromPubSub(topic='projects/proj/topics/events')
output | beam.io.WriteToPubSub(topic='projects/proj/topics/results')
```

## Stateful Processing

```python
import apache_beam as beam
from apache_beam.transforms.userstate import (
    BagStateSpec, CombiningValueStateSpec, ReadModifyWriteStateSpec
)
from apache_beam.coders import VarIntCoder

class StatefulCounter(beam.DoFn):
    COUNT_STATE = CombiningValueStateSpec('count', VarIntCoder(), sum)

    def process(self, element, count=beam.DoFn.StateParam(COUNT_STATE)):
        key, value = element
        count.add(1)
        current = count.read()
        if current >= 100:
            yield f'{key}: batch of {current}'
            count.clear()
```

## Runners

```bash
# DirectRunner (local testing)
python pipeline.py --runner=DirectRunner

# Apache Flink
python pipeline.py --runner=FlinkRunner \
  --flink_master=localhost:8081

# Google Cloud Dataflow
python pipeline.py --runner=DataflowRunner \
  --project=my-project \
  --region=us-central1 \
  --temp_location=gs://bucket/temp \
  --staging_location=gs://bucket/staging \
  --machine_type=n1-standard-4 \
  --num_workers=10 \
  --max_num_workers=50

# Apache Spark
python pipeline.py --runner=SparkRunner \
  --spark_master_url=spark://host:7077
```

## Testing

```python
import apache_beam as beam
from apache_beam.testing.test_pipeline import TestPipeline
from apache_beam.testing.util import assert_that, equal_to

def test_word_count():
    with TestPipeline() as p:
        output = (
            p
            | beam.Create(['hello world', 'hello beam'])
            | beam.FlatMap(str.split)
            | beam.combiners.Count.PerElement()
        )
        assert_that(output, equal_to([
            ('hello', 2), ('world', 1), ('beam', 1)
        ]))
```

## Tips

- Use CombinePerKey instead of GroupByKey + manual aggregation; it enables partial combining and reduces shuffle
- Prefer Map/FlatMap for simple transforms; use ParDo only when you need setup/teardown or multiple outputs
- Set allowed_lateness for streaming pipelines to handle out-of-order data gracefully
- Use side inputs for small lookup tables; for large lookups, use CoGroupByKey or an external service
- Test with DirectRunner locally before deploying to Flink or Dataflow
- Use ACCUMULATING mode when downstream needs full results; DISCARDING when it only needs deltas
- Session windows are powerful for user activity analysis but expensive; use fixed windows when possible
- Monitor watermark progress in streaming pipelines; stuck watermarks indicate stuck sources
- Write custom CombineFns that are associative and commutative for optimal distributed aggregation
- Use beam.io.fileio for dynamic file destinations (write to different paths based on content)

## See Also

- sql
- postgresql

## References

- [Apache Beam Documentation](https://beam.apache.org/documentation/)
- [Beam Programming Guide](https://beam.apache.org/documentation/programming-guide/)
- [Streaming 101 — Tyler Akidau](https://www.oreilly.com/radar/the-world-beyond-batch-streaming-101/)
- [Beam I/O Connectors](https://beam.apache.org/documentation/io/connectors/)
- [Apache Beam GitHub](https://github.com/apache/beam)
