# AWS Lambda

Event-driven, serverless compute service that runs code in response to triggers without managing servers.

## Handler Function

```python
# Python handler — lambda_function.py
def lambda_handler(event, context):
    """
    event:   dict — input data from the trigger
    context: LambdaContext — runtime info (request_id, memory, time_remaining)
    """
    print(f"Request ID: {context.aws_request_id}")
    print(f"Time remaining: {context.get_remaining_time_in_millis()}ms")
    return {
        "statusCode": 200,
        "body": "Hello from Lambda"
    }
```

```javascript
// Node.js handler — index.mjs
export const handler = async (event, context) => {
    return {
        statusCode: 200,
        body: JSON.stringify({ message: "Hello from Lambda" }),
    };
};
```

```go
// Go handler — main.go
package main

import (
    "context"
    "github.com/aws/aws-lambda-go/lambda"
)

type Event struct {
    Name string `json:"name"`
}

func handler(ctx context.Context, event Event) (string, error) {
    return "Hello, " + event.Name, nil
}

func main() {
    lambda.Start(handler)
}
```

## Event Sources

```bash
# API Gateway (REST/HTTP API)
# Synchronous invocation, returns response to caller
aws lambda create-function-url-config \
    --function-name my-func \
    --auth-type NONE

# S3 (object events)
aws s3api put-bucket-notification-configuration \
    --bucket my-bucket \
    --notification-configuration '{
        "LambdaFunctionConfigurations": [{
            "LambdaFunctionArn": "arn:aws:lambda:us-east-1:123456:function:process",
            "Events": ["s3:ObjectCreated:*"],
            "Filter": {"Key": {"FilterRules": [{"Name": "suffix", "Value": ".json"}]}}
        }]
    }'

# SQS (message queue)
aws lambda create-event-source-mapping \
    --function-name my-func \
    --event-source-arn arn:aws:sqs:us-east-1:123456:my-queue \
    --batch-size 10

# DynamoDB Streams
aws lambda create-event-source-mapping \
    --function-name my-func \
    --event-source-arn arn:aws:dynamodb:us-east-1:123456:table/my-table/stream/timestamp \
    --starting-position LATEST \
    --batch-size 100

# EventBridge (scheduled/rule-based)
aws events put-rule \
    --name "every-5-minutes" \
    --schedule-expression "rate(5 minutes)"
aws events put-targets \
    --rule "every-5-minutes" \
    --targets "Id"="1","Arn"="arn:aws:lambda:us-east-1:123456:function:my-func"
```

## Create and Deploy

```bash
# Create function from zip
zip function.zip lambda_function.py
aws lambda create-function \
    --function-name my-func \
    --runtime python3.12 \
    --role arn:aws:iam::123456:role/lambda-role \
    --handler lambda_function.lambda_handler \
    --zip-file fileb://function.zip \
    --timeout 30 \
    --memory-size 256

# Update function code
aws lambda update-function-code \
    --function-name my-func \
    --zip-file fileb://function.zip

# Update configuration
aws lambda update-function-configuration \
    --function-name my-func \
    --timeout 60 \
    --memory-size 512

# Invoke manually
aws lambda invoke \
    --function-name my-func \
    --payload '{"key": "value"}' \
    --cli-binary-format raw-in-base64-out \
    output.json
```

## Environment Variables

```bash
# Set environment variables
aws lambda update-function-configuration \
    --function-name my-func \
    --environment "Variables={DB_HOST=db.example.com,API_KEY=secret123}"

# Access in code
import os
db_host = os.environ['DB_HOST']

# Encrypt with KMS
aws lambda update-function-configuration \
    --function-name my-func \
    --kms-key-arn arn:aws:kms:us-east-1:123456:key/key-id \
    --environment "Variables={DB_PASSWORD=encrypted_value}"
```

## Lambda Layers

```bash
# Create a layer (shared dependencies)
mkdir -p python/lib/python3.12/site-packages
pip install requests -t python/lib/python3.12/site-packages/
zip -r layer.zip python/

aws lambda publish-layer-version \
    --layer-name my-deps \
    --zip-file fileb://layer.zip \
    --compatible-runtimes python3.12

# Attach layer to function (max 5 layers)
aws lambda update-function-configuration \
    --function-name my-func \
    --layers arn:aws:lambda:us-east-1:123456:layer:my-deps:1
```

## Cold Starts and Provisioned Concurrency

```bash
# Cold start = initialization time when no warm container exists
# Typical cold starts:
#   Python:  ~200-500ms
#   Node.js: ~200-400ms
#   Java:    ~1-5s (JVM startup)
#   Go:      ~50-100ms (compiled binary)

# Provisioned concurrency — pre-warm N instances
aws lambda put-provisioned-concurrency-config \
    --function-name my-func \
    --qualifier prod \
    --provisioned-concurrent-executions 10

# Auto-scaling provisioned concurrency
aws application-autoscaling register-scalable-target \
    --service-namespace lambda \
    --resource-id "function:my-func:prod" \
    --scalable-dimension lambda:function:ProvisionedConcurrency \
    --min-capacity 5 --max-capacity 50
```

## IAM Execution Role

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "logs:CreateLogGroup",
                "logs:CreateLogStream",
                "logs:PutLogEvents"
            ],
            "Resource": "arn:aws:logs:*:*:*"
        },
        {
            "Effect": "Allow",
            "Action": ["s3:GetObject"],
            "Resource": "arn:aws:s3:::my-bucket/*"
        }
    ]
}
```

## Destinations (Async Invocation)

```bash
# Route async results to other services
aws lambda put-function-event-invoke-config \
    --function-name my-func \
    --destination-config '{
        "OnSuccess": {"Destination": "arn:aws:sqs:us-east-1:123456:success-queue"},
        "OnFailure": {"Destination": "arn:aws:sqs:us-east-1:123456:dlq"}
    }'
```

## Lambda@Edge / CloudFront Functions

```bash
# Lambda@Edge — runs at CloudFront edge locations
# Use cases: A/B testing, auth, URL rewrite, header manipulation
# Restrictions: Node.js/Python only, 5s timeout (viewer), 30s (origin)

aws lambda publish-version --function-name my-edge-func

# Associate with CloudFront distribution
# Event types: viewer-request, viewer-response, origin-request, origin-response
```

## SAM (Serverless Application Model)

```yaml
# template.yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31

Resources:
  MyFunction:
    Type: AWS::Serverless::Function
    Properties:
      Handler: app.lambda_handler
      Runtime: python3.12
      MemorySize: 256
      Timeout: 30
      Events:
        Api:
          Type: Api
          Properties:
            Path: /hello
            Method: get
```

```bash
# SAM CLI
sam init --runtime python3.12
sam build
sam local invoke MyFunction --event event.json
sam local start-api
sam deploy --guided
```

## CDK (Cloud Development Kit)

```typescript
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';

const fn = new lambda.Function(this, 'MyFunc', {
    runtime: lambda.Runtime.PYTHON_3_12,
    handler: 'app.lambda_handler',
    code: lambda.Code.fromAsset('lambda'),
    memorySize: 256,
    timeout: cdk.Duration.seconds(30),
});

new apigateway.LambdaRestApi(this, 'Api', { handler: fn });
```

## Monitoring

```bash
# View logs
aws logs tail /aws/lambda/my-func --follow

# Key CloudWatch metrics
# Invocations, Duration, Errors, Throttles, ConcurrentExecutions
# IteratorAge (for stream-based sources)

# X-Ray tracing
aws lambda update-function-configuration \
    --function-name my-func \
    --tracing-config Mode=Active
```

## Tips

- Memory allocation also scales CPU proportionally; 1769 MB = 1 full vCPU
- Keep deployment packages small; use layers for shared dependencies
- Initialize SDK clients outside the handler to reuse across invocations (connection reuse)
- Set `reserved concurrency` to prevent a single function from consuming all account concurrency
- Use `/tmp` (512 MB, or up to 10 GB configured) for ephemeral file storage between invocations
- For Java, use GraalVM native-image or SnapStart to reduce cold starts from seconds to milliseconds
- Dead-letter queues (DLQ) catch failed async invocations; always configure one
- Async invocations retry twice by default (configurable 0-2 retries)
- Use Lambda Power Tuning to find optimal memory/cost balance
- VPC-attached Lambdas use Hyperplane ENIs; cold starts are no longer penalized (since 2019)
- ARM64 (Graviton2) is ~20% cheaper and often faster than x86_64
- Connection pooling matters: reuse HTTP clients and DB connections across invocations

## See Also

- API Gateway (HTTP/REST frontend for Lambda)
- Step Functions (orchestrate multiple Lambda functions)
- EventBridge (event bus for routing events to Lambda)
- DynamoDB (serverless database, common Lambda backend)
- SAM / CDK (infrastructure-as-code for serverless)

## References

- [AWS Lambda Developer Guide](https://docs.aws.amazon.com/lambda/latest/dg/welcome.html)
- [Lambda Quotas and Limits](https://docs.aws.amazon.com/lambda/latest/dg/gettingstarted-limits.html)
- [Lambda Power Tuning](https://github.com/alexcasalboni/aws-lambda-power-tuning)
- [SAM CLI Documentation](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/)
- [CDK Lambda Construct](https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_lambda-readme.html)
- [Lambda Operator Guide](https://docs.aws.amazon.com/lambda/latest/operatorguide/intro.html)
