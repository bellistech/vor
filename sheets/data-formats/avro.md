# Apache Avro (Schema-Based Serialization)

> Row-oriented binary serialization with JSON-defined schemas, built-in schema evolution with compatibility guarantees, tight Kafka integration via Schema Registry, and self-describing container files — designed for data-intensive applications where schemas change over time.

## Schema Definition

### Primitive Types

```json
{
  "type": "null"
}
// Other primitives: "boolean", "int", "long", "float", "double", "bytes", "string"
```

### Record Schema

```json
{
  "type": "record",
  "name": "User",
  "namespace": "com.example.avro",
  "doc": "A user in the system",
  "fields": [
    {"name": "id", "type": "long"},
    {"name": "name", "type": "string"},
    {"name": "email", "type": ["null", "string"], "default": null},
    {"name": "age", "type": "int", "default": 0},
    {"name": "tags", "type": {"type": "array", "items": "string"}, "default": []},
    {"name": "created_at", "type": {"type": "long", "logicalType": "timestamp-millis"}}
  ]
}
```

### Complex Types

```json
// Enum
{
  "type": "enum",
  "name": "Status",
  "symbols": ["ACTIVE", "INACTIVE", "BANNED"],
  "default": "ACTIVE"
}

// Array
{
  "type": "array",
  "items": "string"
}

// Map (keys are always strings)
{
  "type": "map",
  "values": "long"
}

// Union (nullable field pattern)
["null", "string"]
["null", "int", "string"]
```

### Logical Types

```json
// Date (days since epoch)
{"type": "int", "logicalType": "date"}

// Time (milliseconds since midnight)
{"type": "int", "logicalType": "time-millis"}

// Timestamp (milliseconds since epoch)
{"type": "long", "logicalType": "timestamp-millis"}

// Timestamp (microseconds since epoch)
{"type": "long", "logicalType": "timestamp-micros"}

// Decimal (arbitrary precision)
{"type": "bytes", "logicalType": "decimal", "precision": 10, "scale": 2}

// UUID
{"type": "string", "logicalType": "uuid"}
```

## Schema Evolution

### Compatibility Levels

```
Level                 Rules
─────────────────────────────────────────────────────────
BACKWARD              New schema can read old data.
                      - Can add fields with defaults
                      - Can remove fields without defaults

FORWARD               Old schema can read new data.
                      - Can remove fields with defaults
                      - Can add fields without defaults

FULL                  Both BACKWARD and FORWARD.
                      - Can add/remove fields WITH defaults only

BACKWARD_TRANSITIVE   BACKWARD across all versions
FORWARD_TRANSITIVE    FORWARD across all versions
FULL_TRANSITIVE       FULL across all versions
NONE                  No compatibility checks
```

### Evolution Examples

```json
// Version 1: Original schema
{
  "type": "record",
  "name": "Event",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "timestamp", "type": "long"},
    {"name": "payload", "type": "string"}
  ]
}

// Version 2: Add optional field (BACKWARD compatible)
{
  "type": "record",
  "name": "Event",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "timestamp", "type": "long"},
    {"name": "payload", "type": "string"},
    {"name": "source", "type": ["null", "string"], "default": null}
  ]
}

// Version 3: Add field with default (FULL compatible)
{
  "type": "record",
  "name": "Event",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "timestamp", "type": "long"},
    {"name": "payload", "type": "string"},
    {"name": "source", "type": ["null", "string"], "default": null},
    {"name": "priority", "type": "int", "default": 0}
  ]
}
```

## Confluent Schema Registry

### REST API

```bash
# List subjects
curl http://localhost:8081/subjects

# Get latest schema for a subject
curl http://localhost:8081/subjects/my-topic-value/versions/latest

# Register a new schema
curl -X POST -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  http://localhost:8081/subjects/my-topic-value/versions \
  --data '{"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"}]}"}'

# Check compatibility
curl -X POST -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  http://localhost:8081/compatibility/subjects/my-topic-value/versions/latest \
  --data '{"schema": "{\"type\":\"record\",\"name\":\"User\",\"fields\":[{\"name\":\"name\",\"type\":\"string\"},{\"name\":\"age\",\"type\":\"int\",\"default\":0}]}"}'

# Set compatibility level
curl -X PUT -H "Content-Type: application/vnd.schemaregistry.v1+json" \
  http://localhost:8081/config/my-topic-value \
  --data '{"compatibility": "FULL"}'

# Get schema by ID
curl http://localhost:8081/schemas/ids/1

# Delete a subject (soft delete)
curl -X DELETE http://localhost:8081/subjects/my-topic-value

# Delete a subject (hard delete)
curl -X DELETE http://localhost:8081/subjects/my-topic-value?permanent=true
```

## Java/Kotlin Usage

### GenericRecord (Schema at Runtime)

```java
import org.apache.avro.Schema;
import org.apache.avro.generic.GenericData;
import org.apache.avro.generic.GenericRecord;

// Parse schema
Schema schema = new Schema.Parser().parse(new File("user.avsc"));

// Create record
GenericRecord user = new GenericData.Record(schema);
user.put("id", 1L);
user.put("name", "Alice");
user.put("email", "alice@example.com");

// Read field
String name = user.get("name").toString();
```

### SpecificRecord (Code Generation)

```bash
# Generate Java classes from schema
java -jar avro-tools-1.12.0.jar compile schema user.avsc output/

# Maven plugin
# <plugin>
#   <groupId>org.apache.avro</groupId>
#   <artifactId>avro-maven-plugin</artifactId>
#   <version>1.12.0</version>
# </plugin>
```

```java
// Generated class usage
User user = User.newBuilder()
    .setId(1L)
    .setName("Alice")
    .setEmail("alice@example.com")
    .setAge(30)
    .build();
```

## Avro Container Files

### Reading and Writing .avro Files

```java
import org.apache.avro.file.DataFileWriter;
import org.apache.avro.file.DataFileReader;
import org.apache.avro.specific.SpecificDatumWriter;
import org.apache.avro.specific.SpecificDatumReader;

// Write
DataFileWriter<User> writer = new DataFileWriter<>(new SpecificDatumWriter<>(User.class));
writer.setCodec(CodecFactory.snappyCodec());
writer.create(User.getClassSchema(), new File("users.avro"));
writer.append(user);
writer.close();

// Read
DataFileReader<User> reader = new DataFileReader<>(
    new File("users.avro"), new SpecificDatumReader<>(User.class));
for (User u : reader) {
    System.out.println(u.getName());
}
reader.close();
```

### avro-tools CLI

```bash
# Convert JSON to Avro
java -jar avro-tools-1.12.0.jar fromjson --schema-file user.avsc input.json > users.avro

# Convert Avro to JSON
java -jar avro-tools-1.12.0.jar tojson users.avro

# Get schema from .avro file
java -jar avro-tools-1.12.0.jar getschema users.avro

# Get record count
java -jar avro-tools-1.12.0.jar count users.avro

# Concatenate Avro files
java -jar avro-tools-1.12.0.jar concat file1.avro file2.avro output.avro
```

## Kafka Integration

### Producer with Schema Registry

```java
import io.confluent.kafka.serializers.KafkaAvroSerializer;

Properties props = new Properties();
props.put("bootstrap.servers", "localhost:9092");
props.put("key.serializer", "org.apache.kafka.common.serialization.StringSerializer");
props.put("value.serializer", KafkaAvroSerializer.class);
props.put("schema.registry.url", "http://localhost:8081");
props.put("auto.register.schemas", true);

KafkaProducer<String, GenericRecord> producer = new KafkaProducer<>(props);

GenericRecord record = new GenericData.Record(schema);
record.put("id", 1L);
record.put("name", "Alice");

producer.send(new ProducerRecord<>("users", "key-1", record));
```

### Consumer with Schema Registry

```java
import io.confluent.kafka.serializers.KafkaAvroDeserializer;

Properties props = new Properties();
props.put("bootstrap.servers", "localhost:9092");
props.put("group.id", "my-group");
props.put("key.deserializer", "org.apache.kafka.common.serialization.StringDeserializer");
props.put("value.deserializer", KafkaAvroDeserializer.class);
props.put("schema.registry.url", "http://localhost:8081");
props.put("specific.avro.reader", true);

KafkaConsumer<String, User> consumer = new KafkaConsumer<>(props);
consumer.subscribe(Collections.singletonList("users"));

while (true) {
    ConsumerRecords<String, User> records = consumer.poll(Duration.ofMillis(100));
    for (ConsumerRecord<String, User> record : records) {
        System.out.println(record.value().getName());
    }
}
```

## Tips

- Always define union fields with `["null", "type"]` and `"default": null` for optional fields — this is the idiomatic nullable pattern
- Use FULL_TRANSITIVE compatibility in Schema Registry for production topics — it prevents any breaking change across all versions
- Include `"default"` values on every new field added to a schema — this enables both forward and backward compatibility
- Use logical types (`timestamp-millis`, `date`, `uuid`) instead of raw primitives for semantic clarity
- Store schemas in version control alongside your code — do not rely solely on Schema Registry as the source of truth
- Use `avro-tools getschema` to extract the embedded schema from any `.avro` file for debugging
- Prefer SpecificRecord (code generation) over GenericRecord for compile-time type safety in production code
- Set `auto.register.schemas: false` in production Kafka producers — register schemas explicitly in CI/CD pipelines
- Use Snappy compression in Avro container files — it provides a good balance of speed and compression ratio
- Enum evolution requires adding new symbols only at the end — reordering or removing symbols breaks readers
- The Avro container file header includes the writer schema — readers always know what schema was used to write

## See Also

- protobuf, parquet, json, yaml, xml

## References

- [Apache Avro Specification](https://avro.apache.org/docs/current/specification/)
- [Apache Avro Getting Started (Java)](https://avro.apache.org/docs/current/getting-started-java/)
- [Confluent Schema Registry Documentation](https://docs.confluent.io/platform/current/schema-registry/index.html)
- [Schema Evolution and Compatibility](https://docs.confluent.io/platform/current/schema-registry/fundamentals/schema-evolution.html)
- [Apache Avro GitHub Repository](https://github.com/apache/avro)
