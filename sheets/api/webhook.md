# Webhooks

HTTP callbacks that deliver real-time event notifications from a source system to a consumer endpoint.

## How Webhooks Work

```
1. Consumer registers a callback URL with the provider
2. Event occurs in the provider system
3. Provider sends HTTP POST to the callback URL with event payload
4. Consumer processes the event and returns 2xx response
5. Provider retries on failure (4xx/5xx/timeout)

Provider                          Consumer
   |                                 |
   |  POST /webhook                  |
   |  { "event": "order.created" }   |
   | ------------------------------> |
   |                                 | process event
   |  HTTP 200 OK                    |
   | <------------------------------ |
   |                                 |
```

## Receiving Webhooks (Express.js)

```javascript
const express = require('express');
const crypto = require('crypto');
const app = express();

app.use('/webhook', express.raw({ type: 'application/json' }));

app.post('/webhook', (req, res) => {
    // Verify signature first
    const signature = req.headers['x-webhook-signature'];
    if (!verifySignature(req.body, signature)) {
        return res.status(401).send('Invalid signature');
    }

    const event = JSON.parse(req.body);
    console.log(`Received: ${event.type}`);

    // Acknowledge immediately, process async
    res.status(200).send('OK');
    processEventAsync(event);
});

app.listen(3000);
```

## Receiving Webhooks (Python/Flask)

```python
from flask import Flask, request, abort
import hmac, hashlib

app = Flask(__name__)
WEBHOOK_SECRET = "whsec_..."

@app.route('/webhook', methods=['POST'])
def webhook():
    # Verify HMAC signature
    signature = request.headers.get('X-Webhook-Signature')
    payload = request.get_data()

    expected = hmac.new(
        WEBHOOK_SECRET.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()

    if not hmac.compare_digest(signature, f"sha256={expected}"):
        abort(401)

    event = request.get_json()

    # Idempotency check
    if already_processed(event['id']):
        return 'OK', 200

    # Process
    handle_event(event)
    mark_processed(event['id'])
    return 'OK', 200
```

## HMAC Signature Verification

```bash
# Provider signs payload with shared secret
# HMAC-SHA256 is the standard algorithm

# Compute expected signature
echo -n '{"event":"test"}' | \
    openssl dgst -sha256 -hmac "webhook_secret" | \
    awk '{print $2}'

# Verify with curl (testing)
PAYLOAD='{"event":"order.created","id":"evt_123"}'
SIG=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "secret" | awk '{print $2}')
curl -X POST https://example.com/webhook \
    -H "Content-Type: application/json" \
    -H "X-Webhook-Signature: sha256=$SIG" \
    -d "$PAYLOAD"
```

```python
# Stripe-style signature verification (timestamp + payload)
import hmac, hashlib, time

def verify_stripe_signature(payload, header, secret, tolerance=300):
    parts = dict(item.split('=', 1) for item in header.split(','))
    timestamp = parts['t']
    signature = parts['v1']

    # Check timestamp tolerance (prevent replay attacks)
    if abs(time.time() - int(timestamp)) > tolerance:
        raise ValueError("Timestamp too old")

    # Compute expected signature
    signed_payload = f"{timestamp}.{payload}"
    expected = hmac.new(
        secret.encode(),
        signed_payload.encode(),
        hashlib.sha256
    ).hexdigest()

    if not hmac.compare_digest(expected, signature):
        raise ValueError("Invalid signature")
```

## Delivery Guarantees (At-Least-Once)

```
Webhooks provide AT-LEAST-ONCE delivery:
- Provider retries until consumer acknowledges (2xx)
- Network failures, timeouts, or consumer errors trigger retries
- Consumer may receive the same event multiple times

Consequences:
- Consumer MUST be idempotent
- Consumer MUST handle duplicate events gracefully
- Provider cannot guarantee exactly-once delivery
```

## Retry with Exponential Backoff

```python
# Provider-side retry logic
import time, math

def deliver_webhook(url, payload, max_retries=5):
    for attempt in range(max_retries + 1):
        try:
            response = requests.post(url, json=payload, timeout=30)
            if response.status_code < 300:
                return True  # Success
            if response.status_code < 500:
                return False  # Client error, don't retry (except 429)
        except requests.exceptions.RequestException:
            pass  # Network error, retry

        # Exponential backoff with jitter
        delay = min(300, (2 ** attempt) + random.uniform(0, 1))
        # Attempt 0: ~1s, 1: ~2s, 2: ~4s, 3: ~8s, 4: ~16s, 5: ~32s
        time.sleep(delay)

    return False  # All retries exhausted -> send to DLQ

# Typical retry schedule (real-world providers):
# Stripe:  2 min, 4 min, 8 min, ... up to 3 days (16 attempts)
# GitHub:  10s timeout, then retry after 1 min (no exponential)
# Twilio:  1s, 5s, 20s, 1m, 5m, 1h (3 attempts total)
```

## Idempotency Keys

```python
# Consumer-side idempotency with Redis
import redis
r = redis.Redis()

def process_webhook(event):
    idempotency_key = event['id']  # Provider-assigned unique event ID

    # Atomic check-and-set with TTL
    if not r.set(f"webhook:{idempotency_key}", "processing",
                 nx=True, ex=86400):  # 24h TTL
        return  # Already processed or in-progress

    try:
        handle_event(event)
        r.set(f"webhook:{idempotency_key}", "completed", ex=86400)
    except Exception:
        r.delete(f"webhook:{idempotency_key}")  # Allow retry
        raise
```

```sql
-- Database-based idempotency
CREATE TABLE webhook_events (
    event_id    VARCHAR(255) PRIMARY KEY,
    event_type  VARCHAR(100) NOT NULL,
    processed   BOOLEAN DEFAULT FALSE,
    payload     JSONB,
    received_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP
);

-- Insert with conflict handling (PostgreSQL)
INSERT INTO webhook_events (event_id, event_type, payload)
VALUES ($1, $2, $3)
ON CONFLICT (event_id) DO NOTHING;
-- If rows affected = 0, event was already received
```

## Event Ordering

```
Webhooks do NOT guarantee ordered delivery:
- Retries can arrive after newer events
- Concurrent deliveries may arrive out of order
- Network latency varies per request

Mitigation strategies:
1. Include a sequence number or timestamp in the payload
2. Consumer checks: is this event newer than current state?
3. Use event sourcing: append all events, derive state from history
4. Accept eventual consistency as the default model
```

```python
# Ordering check using timestamps
def handle_webhook(event):
    current = get_resource(event['resource_id'])
    if current and current['updated_at'] >= event['timestamp']:
        return  # Stale event, skip
    apply_event(event)
```

## Payload Design

```json
{
    "id": "evt_1234567890",
    "type": "order.completed",
    "created": "2024-01-15T10:30:00Z",
    "api_version": "2024-01-01",
    "data": {
        "object": {
            "id": "ord_abc123",
            "status": "completed",
            "total": 4200,
            "currency": "usd"
        },
        "previous_attributes": {
            "status": "pending"
        }
    },
    "livemode": true
}
```

```
Best practices for payload design:
- Include event ID for idempotency
- Include event type for routing
- Include timestamp for ordering
- Include API version for compatibility
- Provide both current and previous state for change events
- Use fat payloads (include data) to reduce follow-up API calls
- Alternative: thin payloads (ID only) + consumer fetches current state
```

## Dead Letter Queues

```python
# After exhausting retries, move to DLQ
import boto3

sqs = boto3.client('sqs')
DLQ_URL = "https://sqs.us-east-1.amazonaws.com/123456/webhook-dlq"

def send_to_dlq(event, error, attempts):
    sqs.send_message(
        QueueUrl=DLQ_URL,
        MessageBody=json.dumps({
            'event': event,
            'error': str(error),
            'attempts': attempts,
            'failed_at': datetime.utcnow().isoformat()
        }),
        MessageAttributes={
            'EventType': {'DataType': 'String', 'StringValue': event['type']},
            'ConsumerUrl': {'DataType': 'String', 'StringValue': event['url']}
        }
    )

# Process DLQ manually or via scheduled Lambda
# 1. Inspect failures
# 2. Fix consumer bugs
# 3. Replay events from DLQ
```

## Sending Webhooks (Provider Side)

```python
import requests, hmac, hashlib, json, time, uuid

def send_webhook(url, event_data, secret):
    event = {
        'id': f"evt_{uuid.uuid4().hex}",
        'type': event_data['type'],
        'created': int(time.time()),
        'data': event_data['payload']
    }

    payload = json.dumps(event, separators=(',', ':'))
    timestamp = str(int(time.time()))

    # Sign: timestamp.payload
    signature = hmac.new(
        secret.encode(),
        f"{timestamp}.{payload}".encode(),
        hashlib.sha256
    ).hexdigest()

    headers = {
        'Content-Type': 'application/json',
        'X-Webhook-ID': event['id'],
        'X-Webhook-Timestamp': timestamp,
        'X-Webhook-Signature': f"sha256={signature}",
    }

    response = requests.post(url, data=payload, headers=headers, timeout=30)
    return response.status_code < 300
```

## Testing Webhooks

```bash
# ngrok — expose local server to the internet
ngrok http 3000
# Use the https://xxxx.ngrok.io URL as webhook endpoint

# webhook.site — inspect payloads online (no code needed)
# https://webhook.site — gives you a unique URL

# smee.io — GitHub webhook proxy for local development
npx smee -u https://smee.io/your-channel -t http://localhost:3000/webhook

# curl — manual testing
curl -X POST http://localhost:3000/webhook \
    -H "Content-Type: application/json" \
    -H "X-Webhook-Signature: sha256=abc123" \
    -d '{"type": "test", "id": "evt_test"}'

# requestbin (Pipedream) — hosted request inspector
# https://requestbin.com
```

## Tips

- Always verify webhook signatures before processing; unsigned webhooks are trivially spoofable
- Respond with 200 immediately, then process asynchronously; providers typically timeout at 5-30 seconds
- Implement idempotency from day one; every webhook consumer will eventually receive duplicates
- Use a queue (SQS, RabbitMQ) between your webhook endpoint and processor for reliability
- Store raw webhook payloads before processing; they are invaluable for debugging and replay
- Rotate webhook secrets periodically; support multiple active secrets during rotation windows
- Monitor webhook processing lag and failure rates; alert on DLQ depth
- Use fat payloads to minimize API calls, but always validate by fetching current state for sensitive operations
- Set appropriate timeouts on your webhook endpoint (5-10 seconds max for the initial response)
- Log the full request (headers + body) for failed signature verifications
- Use IP allowlisting as a defense-in-depth measure alongside HMAC verification
- Test webhook recovery by intentionally returning errors and verifying retry behavior

## See Also

- API Gateway (front-end for webhook endpoints)
- Event-Driven Architecture (broader pattern webhooks implement)
- Message Queues (SQS, RabbitMQ, Kafka for reliable event processing)
- Server-Sent Events (alternative: server push over HTTP)
- WebSocket (alternative: bidirectional real-time communication)

## References

- [Standard Webhooks Specification](https://www.standardwebhooks.com/)
- [Stripe Webhook Best Practices](https://docs.stripe.com/webhooks/best-practices)
- [GitHub Webhooks Documentation](https://docs.github.com/en/webhooks)
- [Webhook.site (Testing Tool)](https://webhook.site/)
- [ngrok Documentation](https://ngrok.com/docs)
- [Svix Webhook Service](https://www.svix.com/resources/faq/what-are-webhooks/)
