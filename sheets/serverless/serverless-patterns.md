# Serverless Patterns

Architectural patterns for building reliable, scalable event-driven serverless applications.

## Cold Start Mitigation

```bash
# Problem: first invocation after idle period incurs initialization overhead
# Typical cold starts: 100ms (Go) to 5s (Java)

# Pattern 1: Provisioned concurrency (pre-warmed instances)
aws lambda put-provisioned-concurrency-config \
    --function-name my-func \
    --qualifier prod \
    --provisioned-concurrent-executions 10

# Pattern 2: Scheduled warming (ping function on interval)
# EventBridge rule every 5 minutes
aws events put-rule --name "warm-lambda" \
    --schedule-expression "rate(5 minutes)"

# Pattern 3: Optimize package size
# Smaller deployment = faster cold start
# Remove unused dependencies, use tree-shaking
pip install --no-cache-dir -t . requests  # only what you need

# Pattern 4: Use lightweight runtimes
# Go, Rust (custom runtime) >> Python, Node >> Java, .NET

# Pattern 5: Lazy initialization
import boto3
_client = None
def get_client():
    global _client
    if _client is None:
        _client = boto3.client('dynamodb')
    return _client
```

## Fan-Out / Fan-In

```bash
# Fan-out: one event triggers many parallel functions
# Fan-in:  aggregate results from parallel executions

# SNS fan-out (one message -> many subscribers)
aws sns publish --topic-arn arn:aws:sns:us-east-1:123456:orders \
    --message '{"orderId": "123"}'

# Step Functions fan-out with Map state
# Distributes items across parallel Lambda invocations
```

```json
{
    "Type": "Map",
    "ItemsPath": "$.items",
    "MaxConcurrency": 40,
    "Iterator": {
        "StartAt": "ProcessItem",
        "States": {
            "ProcessItem": {
                "Type": "Task",
                "Resource": "arn:aws:lambda:us-east-1:123456:function:process",
                "End": true
            }
        }
    }
}
```

## Event Sourcing

```bash
# Store state as a sequence of immutable events
# Benefits: full audit trail, temporal queries, replay capability

# DynamoDB as event store
aws dynamodb put-item --table-name events --item '{
    "pk": {"S": "ORDER#123"},
    "sk": {"S": "EVENT#2024-01-15T10:30:00Z#OrderCreated"},
    "data": {"S": "{\"items\": [\"A\", \"B\"], \"total\": 42.00}"},
    "version": {"N": "1"}
}'

# EventBridge as event bus
aws events put-events --entries '[{
    "Source": "orders",
    "DetailType": "OrderCreated",
    "Detail": "{\"orderId\": \"123\", \"total\": 42.00}"
}]'
```

```python
# Event sourcing handler
def handle_order(event):
    events = get_events(event['order_id'])  # load event history
    state = replay(events)                   # rebuild current state
    new_events = process(state, event)       # produce new events
    store_events(new_events)                 # append to event store
    publish(new_events)                      # notify subscribers
```

## Saga Pattern (Distributed Transactions)

```bash
# Problem: no distributed transactions in serverless
# Solution: sequence of local transactions with compensating actions

# Step Functions orchestrated saga
```

```json
{
    "StartAt": "ReserveInventory",
    "States": {
        "ReserveInventory": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:reserve-inventory",
            "Catch": [{"ErrorEquals": ["States.ALL"], "Next": "CancelOrder"}],
            "Next": "ProcessPayment"
        },
        "ProcessPayment": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:process-payment",
            "Catch": [{"ErrorEquals": ["States.ALL"], "Next": "ReleaseInventory"}],
            "Next": "FulfillOrder"
        },
        "FulfillOrder": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:fulfill-order",
            "Catch": [{"ErrorEquals": ["States.ALL"], "Next": "RefundPayment"}],
            "End": true
        },
        "RefundPayment": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:refund-payment",
            "Next": "ReleaseInventory"
        },
        "ReleaseInventory": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:release-inventory",
            "Next": "CancelOrder"
        },
        "CancelOrder": {
            "Type": "Fail",
            "Error": "SagaFailed",
            "Cause": "Compensating transactions completed"
        }
    }
}
```

## Circuit Breaker

```python
import time, os
import boto3

dynamodb = boto3.resource('dynamodb')
table = dynamodb.Table('circuit-breaker')

def circuit_breaker(service_name, fn, *args):
    """
    States: CLOSED (normal), OPEN (failing), HALF_OPEN (testing)
    """
    state = get_state(service_name)

    if state['status'] == 'OPEN':
        if time.time() - state['last_failure'] > int(os.environ.get('RESET_TIMEOUT', 60)):
            state['status'] = 'HALF_OPEN'
        else:
            raise Exception(f"Circuit OPEN for {service_name}")

    try:
        result = fn(*args)
        if state['status'] == 'HALF_OPEN':
            reset_circuit(service_name)
        return result
    except Exception as e:
        record_failure(service_name, state)
        if state.get('failure_count', 0) >= int(os.environ.get('THRESHOLD', 5)):
            trip_circuit(service_name)
        raise

def trip_circuit(service_name):
    table.update_item(
        Key={'service': service_name},
        UpdateExpression='SET #s = :s, last_failure = :t, failure_count = :c',
        ExpressionAttributeNames={'#s': 'status'},
        ExpressionAttributeValues={':s': 'OPEN', ':t': int(time.time()), ':c': 0}
    )
```

## Idempotency (DynamoDB Conditional Writes)

```python
import hashlib, json, time
import boto3

dynamodb = boto3.resource('dynamodb')
table = dynamodb.Table('idempotency')

def idempotent_handler(event, context):
    # Generate idempotency key from event
    key = hashlib.sha256(json.dumps(event, sort_keys=True).encode()).hexdigest()

    # Try to claim the execution
    try:
        table.put_item(
            Item={
                'idempotency_key': key,
                'status': 'IN_PROGRESS',
                'ttl': int(time.time()) + 3600,  # 1 hour expiry
                'request_id': context.aws_request_id,
            },
            ConditionExpression='attribute_not_exists(idempotency_key) OR #s = :expired',
            ExpressionAttributeNames={'#s': 'status'},
            ExpressionAttributeValues={':expired': 'EXPIRED'}
        )
    except dynamodb.meta.client.exceptions.ConditionalCheckFailedException:
        # Already processed — return cached result
        existing = table.get_item(Key={'idempotency_key': key})
        return existing['Item'].get('result')

    # Process the event
    result = do_work(event)

    # Store result for future duplicate detection
    table.update_item(
        Key={'idempotency_key': key},
        UpdateExpression='SET #s = :s, #r = :r',
        ExpressionAttributeNames={'#s': 'status', '#r': 'result'},
        ExpressionAttributeValues={':s': 'COMPLETED', ':r': result}
    )
    return result
```

## Async Invocation Pattern

```bash
# Asynchronous Lambda invocation (fire-and-forget)
aws lambda invoke \
    --function-name my-func \
    --invocation-type Event \
    --payload '{"key": "value"}' \
    /dev/null

# Configure retry behavior
aws lambda put-function-event-invoke-config \
    --function-name my-func \
    --maximum-retry-attempts 1 \
    --maximum-event-age-in-seconds 3600 \
    --destination-config '{
        "OnSuccess": {"Destination": "arn:aws:sqs:...:success"},
        "OnFailure": {"Destination": "arn:aws:sqs:...:failure"}
    }'
```

## Step Functions Orchestration

```json
{
    "Comment": "Order processing workflow",
    "StartAt": "ValidateOrder",
    "States": {
        "ValidateOrder": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:validate",
            "Next": "CheckInventory"
        },
        "CheckInventory": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:check-inventory",
            "Next": "InStockChoice"
        },
        "InStockChoice": {
            "Type": "Choice",
            "Choices": [{
                "Variable": "$.inStock",
                "BooleanEquals": true,
                "Next": "ProcessPayment"
            }],
            "Default": "WaitForRestock"
        },
        "WaitForRestock": {
            "Type": "Wait",
            "Seconds": 300,
            "Next": "CheckInventory"
        },
        "ProcessPayment": {
            "Type": "Task",
            "Resource": "arn:aws:lambda:...:payment",
            "Retry": [{
                "ErrorEquals": ["TransientError"],
                "IntervalSeconds": 2,
                "MaxAttempts": 3,
                "BackoffRate": 2.0
            }],
            "End": true
        }
    }
}
```

## CQRS (Command Query Responsibility Segregation)

```bash
# Separate write (command) and read (query) paths
#
# Write path: API Gateway -> Lambda -> DynamoDB (source of truth)
#                                   -> DynamoDB Stream -> Lambda -> ElasticSearch (read model)
#
# Read path:  API Gateway -> Lambda -> ElasticSearch (optimized queries)

# DynamoDB Stream triggers Lambda to update read model
aws lambda create-event-source-mapping \
    --function-name update-read-model \
    --event-source-arn arn:aws:dynamodb:...:table/orders/stream/... \
    --starting-position LATEST
```

## Tips

- Always design for idempotency; event sources may deliver duplicates (SQS, EventBridge, S3)
- Use DynamoDB TTL on idempotency records to auto-expire after a safe window (1-24 hours)
- Prefer Step Functions over Lambda-to-Lambda calls; direct invocation couples functions tightly
- Set `maxConcurrency` on Map states to avoid overwhelming downstream services
- Use SQS as a buffer between services to absorb traffic spikes and decouple producers/consumers
- Dead-letter queues are mandatory; unprocessed events vanish without them
- Keep Lambda functions focused on a single responsibility; avoid monolithic handlers
- Use environment-specific configuration via SSM Parameter Store, not hardcoded values
- The saga pattern requires every step to have an idempotent compensating action
- Test locally with SAM CLI (`sam local invoke`) before deploying
- Monitor `IteratorAge` for stream-based triggers; high values indicate processing lag
- Use Express Step Functions for high-volume, short-duration workflows (up to 5 minutes)

## See Also

- AWS Lambda (the compute foundation for serverless)
- Step Functions (workflow orchestration)
- EventBridge (event routing and filtering)
- DynamoDB (serverless NoSQL database)
- SQS / SNS (message queuing and pub/sub)

## References

- [AWS Serverless Patterns Collection](https://serverlessland.com/patterns)
- [Serverless Application Lens (AWS Well-Architected)](https://docs.aws.amazon.com/wellarchitected/latest/serverless-applications-lens/welcome.html)
- [Lambda Powertools (Idempotency)](https://docs.powertools.aws.dev/lambda/python/latest/utilities/idempotency/)
- [Step Functions Developer Guide](https://docs.aws.amazon.com/step-functions/latest/dg/welcome.html)
- [Saga Pattern in Serverless (theburningmonk.com)](https://theburningmonk.com/2017/07/applying-the-saga-pattern-with-aws-lambda-and-step-functions/)
- [CQRS Pattern (Microsoft)](https://learn.microsoft.com/en-us/azure/architecture/patterns/cqrs)
