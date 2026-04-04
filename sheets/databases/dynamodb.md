# DynamoDB (AWS NoSQL Database)

Amazon DynamoDB is a fully managed key-value and document database providing single-digit millisecond performance at any scale, with automatic partitioning, optional secondary indexes, streams for change data capture, and both provisioned and on-demand capacity modes.

## Table Operations

```bash
# Create table with partition key
aws dynamodb create-table \
  --table-name Users \
  --attribute-definitions \
    AttributeName=UserId,AttributeType=S \
  --key-schema \
    AttributeName=UserId,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST

# Create table with partition + sort key (composite)
aws dynamodb create-table \
  --table-name Orders \
  --attribute-definitions \
    AttributeName=CustomerId,AttributeType=S \
    AttributeName=OrderDate,AttributeType=S \
  --key-schema \
    AttributeName=CustomerId,KeyType=HASH \
    AttributeName=OrderDate,KeyType=RANGE \
  --billing-mode PROVISIONED \
  --provisioned-throughput ReadCapacityUnits=10,WriteCapacityUnits=5

# List tables
aws dynamodb list-tables

# Describe table
aws dynamodb describe-table --table-name Users

# Delete table
aws dynamodb delete-table --table-name Users

# Update table (change capacity)
aws dynamodb update-table \
  --table-name Orders \
  --provisioned-throughput ReadCapacityUnits=20,WriteCapacityUnits=10
```

## Item Operations (CRUD)

```bash
# Put item
aws dynamodb put-item \
  --table-name Users \
  --item '{
    "UserId": {"S": "user-001"},
    "Name": {"S": "Alice"},
    "Email": {"S": "alice@example.com"},
    "Age": {"N": "30"},
    "Tags": {"SS": ["admin", "premium"]}
  }'

# Get item
aws dynamodb get-item \
  --table-name Users \
  --key '{"UserId": {"S": "user-001"}}' \
  --consistent-read

# Update item
aws dynamodb update-item \
  --table-name Users \
  --key '{"UserId": {"S": "user-001"}}' \
  --update-expression "SET Age = :age, #n = :name" \
  --expression-attribute-names '{"#n": "Name"}' \
  --expression-attribute-values '{":age": {"N": "31"}, ":name": {"S": "Alice B"}}' \
  --return-values UPDATED_NEW

# Delete item
aws dynamodb delete-item \
  --table-name Users \
  --key '{"UserId": {"S": "user-001"}}'

# Conditional write (only if exists)
aws dynamodb put-item \
  --table-name Users \
  --item '{"UserId": {"S": "user-001"}, "Name": {"S": "Alice"}}' \
  --condition-expression "attribute_not_exists(UserId)"
```

## Query and Scan

```bash
# Query by partition key
aws dynamodb query \
  --table-name Orders \
  --key-condition-expression "CustomerId = :cid" \
  --expression-attribute-values '{":cid": {"S": "cust-001"}}'

# Query with sort key range
aws dynamodb query \
  --table-name Orders \
  --key-condition-expression "CustomerId = :cid AND OrderDate BETWEEN :start AND :end" \
  --expression-attribute-values '{
    ":cid": {"S": "cust-001"},
    ":start": {"S": "2024-01-01"},
    ":end": {"S": "2024-12-31"}
  }'

# Query with filter (applied after read, still consumes RCU)
aws dynamodb query \
  --table-name Orders \
  --key-condition-expression "CustomerId = :cid" \
  --filter-expression "Total > :min" \
  --expression-attribute-values '{":cid": {"S": "cust-001"}, ":min": {"N": "100"}}'

# Scan (reads entire table — expensive)
aws dynamodb scan \
  --table-name Users \
  --filter-expression "Age > :age" \
  --expression-attribute-values '{":age": {"N": "25"}}' \
  --max-items 100

# Parallel scan
aws dynamodb scan \
  --table-name Users \
  --total-segments 4 \
  --segment 0
```

## Global Secondary Indexes (GSI)

```bash
# Create table with GSI
aws dynamodb create-table \
  --table-name Orders \
  --attribute-definitions \
    AttributeName=CustomerId,AttributeType=S \
    AttributeName=OrderDate,AttributeType=S \
    AttributeName=Status,AttributeType=S \
  --key-schema \
    AttributeName=CustomerId,KeyType=HASH \
    AttributeName=OrderDate,KeyType=RANGE \
  --global-secondary-indexes '[{
    "IndexName": "StatusIndex",
    "KeySchema": [
      {"AttributeName": "Status", "KeyType": "HASH"},
      {"AttributeName": "OrderDate", "KeyType": "RANGE"}
    ],
    "Projection": {"ProjectionType": "ALL"},
    "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
  }]' \
  --billing-mode PROVISIONED \
  --provisioned-throughput ReadCapacityUnits=10,WriteCapacityUnits=5

# Query GSI
aws dynamodb query \
  --table-name Orders \
  --index-name StatusIndex \
  --key-condition-expression "#s = :status" \
  --expression-attribute-names '{"#s": "Status"}' \
  --expression-attribute-values '{":status": {"S": "SHIPPED"}}'

# Add GSI to existing table
aws dynamodb update-table \
  --table-name Orders \
  --attribute-definitions AttributeName=Email,AttributeType=S \
  --global-secondary-index-updates '[{
    "Create": {
      "IndexName": "EmailIndex",
      "KeySchema": [{"AttributeName": "Email", "KeyType": "HASH"}],
      "Projection": {"ProjectionType": "KEYS_ONLY"},
      "ProvisionedThroughput": {"ReadCapacityUnits": 5, "WriteCapacityUnits": 5}
    }
  }]'
```

## Batch Operations

```bash
# Batch write (up to 25 items)
aws dynamodb batch-write-item --request-items '{
  "Users": [
    {"PutRequest": {"Item": {"UserId": {"S": "u1"}, "Name": {"S": "Alice"}}}},
    {"PutRequest": {"Item": {"UserId": {"S": "u2"}, "Name": {"S": "Bob"}}}},
    {"DeleteRequest": {"Key": {"UserId": {"S": "u3"}}}}
  ]
}'

# Batch get (up to 100 items)
aws dynamodb batch-get-item --request-items '{
  "Users": {
    "Keys": [
      {"UserId": {"S": "u1"}},
      {"UserId": {"S": "u2"}}
    ],
    "ProjectionExpression": "UserId, Name"
  }
}'
```

## PartiQL (SQL-Compatible Queries)

```bash
# Select
aws dynamodb execute-statement \
  --statement "SELECT * FROM Users WHERE UserId = 'user-001'"

# Insert
aws dynamodb execute-statement \
  --statement "INSERT INTO Users VALUE {'UserId': 'u4', 'Name': 'Carol', 'Age': 28}"

# Update
aws dynamodb execute-statement \
  --statement "UPDATE Users SET Age = 29 WHERE UserId = 'u4'"

# Delete
aws dynamodb execute-statement \
  --statement "DELETE FROM Users WHERE UserId = 'u4'"

# Batch execute
aws dynamodb batch-execute-statement --statements '[
  {"Statement": "SELECT * FROM Users WHERE UserId = '\''u1'\''"},
  {"Statement": "SELECT * FROM Users WHERE UserId = '\''u2'\''"}
]'
```

## DynamoDB Streams

```bash
# Enable streams on table
aws dynamodb update-table \
  --table-name Users \
  --stream-specification StreamEnabled=true,StreamViewType=NEW_AND_OLD_IMAGES

# Stream view types: KEYS_ONLY | NEW_IMAGE | OLD_IMAGE | NEW_AND_OLD_IMAGES

# Describe stream
aws dynamodbstreams describe-stream \
  --stream-arn arn:aws:dynamodb:us-east-1:123456789:table/Users/stream/2024-01-01

# Get shard iterator
aws dynamodbstreams get-shard-iterator \
  --stream-arn $STREAM_ARN \
  --shard-id $SHARD_ID \
  --shard-iterator-type TRIM_HORIZON

# Read records
aws dynamodbstreams get-records --shard-iterator $ITERATOR
```

## TTL (Time to Live)

```bash
# Enable TTL on a numeric attribute (epoch seconds)
aws dynamodb update-time-to-live \
  --table-name Sessions \
  --time-to-live-specification Enabled=true,AttributeName=ExpiresAt

# Set item with TTL
aws dynamodb put-item \
  --table-name Sessions \
  --item '{
    "SessionId": {"S": "sess-001"},
    "ExpiresAt": {"N": "1735689600"}
  }'
# Items are deleted within 48 hours after TTL expiry (not instant)
```

## Single-Table Design Patterns

```bash
# Overloaded partition key pattern
# PK = "USER#alice"     SK = "PROFILE"          -> user profile
# PK = "USER#alice"     SK = "ORDER#2024-001"   -> user's order
# PK = "USER#alice"     SK = "ADDR#home"        -> user's address
# PK = "ORDER#2024-001" SK = "ITEM#sku-123"     -> order line item

# GSI for inverted access:
# GSI1PK = "ORDER#2024-001"  GSI1SK = "USER#alice"  -> find user for order
# GSI1PK = "STATUS#shipped"  GSI1SK = "2024-03-15"  -> orders by status+date
```

## DAX (DynamoDB Accelerator)

```bash
# Create DAX cluster (via CLI)
aws dax create-cluster \
  --cluster-name my-dax \
  --node-type dax.r5.large \
  --replication-factor 3 \
  --iam-role-arn arn:aws:iam::123456789:role/DAXRole \
  --subnet-group my-dax-subnet

# DAX endpoint replaces DynamoDB endpoint in application code
# Provides microsecond read latency for cached items
# Write-through: writes go to DynamoDB and invalidate DAX cache
```

## Tips

- Design partition keys for uniform distribution; hot partitions cause throttling even with available capacity
- Use sort keys to model hierarchical and time-series data within a single partition
- GSIs have their own throughput; under-provisioned GSIs throttle the base table on writes
- Filter expressions reduce response size but NOT read capacity consumed; push filtering into key conditions
- Batch operations have no transactional guarantees; use TransactWriteItems for ACID across up to 100 items
- TTL deletes are eventually consistent (up to 48 hours delay); do not rely on TTL for real-time expiry
- On-demand mode eliminates capacity planning but costs ~6.5x more per request than well-provisioned tables
- Use ProjectionExpression to return only needed attributes and reduce network transfer
- Single-table design reduces the number of queries but increases complexity; start simple, denormalize when needed
- DynamoDB Streams + Lambda is the standard pattern for event-driven architectures and materialized views

## See Also

- redis, cassandra, mongodb, elasticsearch

## References

- [DynamoDB Developer Guide](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/)
- [DynamoDB Best Practices](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/best-practices.html)
- [Single-Table Design (Alex DeBrie)](https://www.alexdebrie.com/posts/dynamodb-single-table/)
- [DynamoDB API Reference](https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/)
- [PartiQL Reference for DynamoDB](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/ql-reference.html)
