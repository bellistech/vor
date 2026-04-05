# Event-Driven Architecture (Sourcing, CQRS, and Streaming)

A comprehensive reference for event-driven patterns covering event sourcing, CQRS, messaging semantics, and stream processing in production systems.

## Event Sourcing

### Core Concept

```
Traditional: Store current state
  users table: {id: 1, name: "Alice", email: "alice@new.com", balance: 150}

Event Sourcing: Store sequence of events
  events table:
    {id: 1, type: "UserCreated",      data: {name: "Alice", email: "alice@old.com"}}
    {id: 2, type: "EmailChanged",     data: {email: "alice@new.com"}}
    {id: 3, type: "BalanceCredited",  data: {amount: 200}}
    {id: 4, type: "BalanceDebited",   data: {amount: 50}}

Current state = replay all events: balance = 0 + 200 - 50 = 150
```

### Event Store Implementation

```go
type Event struct {
    ID            string    `json:"id"`
    AggregateID   string    `json:"aggregate_id"`
    AggregateType string    `json:"aggregate_type"`
    Type          string    `json:"type"`
    Version       int       `json:"version"`
    Data          json.RawMessage `json:"data"`
    Metadata      json.RawMessage `json:"metadata"`
    Timestamp     time.Time `json:"timestamp"`
}

type EventStore interface {
    // Append events — optimistic concurrency via expected version
    Append(ctx context.Context, aggregateID string, expectedVersion int, events []Event) error

    // Load all events for an aggregate
    Load(ctx context.Context, aggregateID string) ([]Event, error)

    // Load events from a specific version
    LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error)

    // Subscribe to new events (for projections)
    Subscribe(ctx context.Context, fromPosition uint64) (<-chan Event, error)
}

// SQL schema for event store
// CREATE TABLE events (
//     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
//     aggregate_id VARCHAR(255) NOT NULL,
//     aggregate_type VARCHAR(255) NOT NULL,
//     event_type VARCHAR(255) NOT NULL,
//     version INTEGER NOT NULL,
//     data JSONB NOT NULL,
//     metadata JSONB,
//     timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
//     UNIQUE(aggregate_id, version)  -- optimistic concurrency
// );
```

### Projections (Read Models)

```go
type Projection interface {
    Handle(event Event) error
}

type UserBalanceProjection struct {
    db *sql.DB
}

func (p *UserBalanceProjection) Handle(event Event) error {
    switch event.Type {
    case "BalanceCredited":
        var data struct{ Amount float64 }
        json.Unmarshal(event.Data, &data)
        _, err := p.db.Exec(
            `INSERT INTO user_balances (user_id, balance)
             VALUES ($1, $2)
             ON CONFLICT (user_id) DO UPDATE SET balance = user_balances.balance + $2`,
            event.AggregateID, data.Amount,
        )
        return err

    case "BalanceDebited":
        var data struct{ Amount float64 }
        json.Unmarshal(event.Data, &data)
        _, err := p.db.Exec(
            `UPDATE user_balances SET balance = balance - $1 WHERE user_id = $2`,
            data.Amount, event.AggregateID,
        )
        return err
    }
    return nil
}

// Projection runner — processes events sequentially
type ProjectionRunner struct {
    store       EventStore
    projection  Projection
    checkpoint  CheckpointStore
    name        string
}

func (r *ProjectionRunner) Run(ctx context.Context) error {
    pos, _ := r.checkpoint.Load(r.name)
    events, err := r.store.Subscribe(ctx, pos)
    if err != nil {
        return err
    }

    for {
        select {
        case event := <-events:
            if err := r.projection.Handle(event); err != nil {
                return fmt.Errorf("projection %s failed: %w", r.name, err)
            }
            r.checkpoint.Save(r.name, event.Position)
        case <-ctx.Done():
            return nil
        }
    }
}
```

### Snapshots

```go
type Aggregate interface {
    Apply(event Event)
    State() any
    LoadSnapshot(state any, version int)
}

type SnapshotStore interface {
    Save(aggregateID string, version int, state any) error
    Load(aggregateID string) (state any, version int, err error)
}

func LoadAggregate(ctx context.Context, agg Aggregate, id string,
    eventStore EventStore, snapStore SnapshotStore) error {

    // Try loading snapshot first
    state, snapVersion, err := snapStore.Load(id)
    if err == nil && state != nil {
        agg.LoadSnapshot(state, snapVersion)
    }

    // Replay events after snapshot
    events, err := eventStore.LoadFrom(ctx, id, snapVersion+1)
    if err != nil {
        return err
    }
    for _, event := range events {
        agg.Apply(event)
    }

    // Save new snapshot every 100 events
    if len(events) > 100 {
        snapStore.Save(id, snapVersion+len(events), agg.State())
    }
    return nil
}
```

## CQRS (Command Query Responsibility Segregation)

### Architecture

```
Commands (writes)                    Queries (reads)
    │                                    │
    v                                    v
┌──────────┐                     ┌──────────────┐
│ Command  │                     │  Query       │
│ Handler  │                     │  Handler     │
└────┬─────┘                     └──────┬───────┘
     │                                  │
     v                                  v
┌──────────┐    events/CDC       ┌──────────────┐
│  Write   │ ────────────────→   │  Read Model  │
│  Store   │                     │  (optimized) │
└──────────┘                     └──────────────┘
  (event store                    (denormalized,
   or RDBMS)                      materialized views)
```

```go
// Command side
type CreateOrderCommand struct {
    CustomerID string
    Items      []OrderItem
}

type CommandHandler interface {
    Handle(ctx context.Context, cmd any) error
}

type OrderCommandHandler struct {
    eventStore EventStore
}

func (h *OrderCommandHandler) Handle(ctx context.Context, cmd any) error {
    switch c := cmd.(type) {
    case CreateOrderCommand:
        order := NewOrder(c.CustomerID, c.Items)
        events := order.UncommittedEvents()
        return h.eventStore.Append(ctx, order.ID, 0, events)
    }
    return fmt.Errorf("unknown command type: %T", cmd)
}

// Query side — optimized read model
type OrderQueryService struct {
    readDB *sql.DB
}

func (q *OrderQueryService) GetOrderSummary(ctx context.Context, orderID string) (*OrderSummary, error) {
    var summary OrderSummary
    err := q.readDB.QueryRowContext(ctx,
        `SELECT order_id, customer_name, total, status, item_count
         FROM order_summaries WHERE order_id = $1`, orderID,
    ).Scan(&summary.OrderID, &summary.CustomerName, &summary.Total,
        &summary.Status, &summary.ItemCount)
    return &summary, err
}
```

## Pub/Sub Patterns

### Topic-Based Publish/Subscribe

```go
type MessageBroker interface {
    Publish(ctx context.Context, topic string, msg Message) error
    Subscribe(ctx context.Context, topic string, group string) (<-chan Message, error)
    Ack(ctx context.Context, msg Message) error
    Nack(ctx context.Context, msg Message) error
}

type Message struct {
    ID          string
    Topic       string
    Key         string            // partition key
    Value       []byte
    Headers     map[string]string
    Timestamp   time.Time
    Partition   int
    Offset      int64
}
```

### CloudEvents Specification

```json
{
    "specversion": "1.0",
    "id": "evt-abc123",
    "source": "/orders/order-service",
    "type": "com.example.order.created",
    "datacontenttype": "application/json",
    "time": "2026-04-04T10:30:00Z",
    "subject": "order-456",
    "data": {
        "orderId": "order-456",
        "customerId": "cust-789",
        "total": 99.99,
        "items": [
            {"sku": "WIDGET-1", "qty": 2, "price": 49.995}
        ]
    }
}
```

```go
// CloudEvents in Go
type CloudEvent struct {
    SpecVersion     string          `json:"specversion"`
    ID              string          `json:"id"`
    Source          string          `json:"source"`
    Type           string          `json:"type"`
    DataContentType string         `json:"datacontenttype,omitempty"`
    Time           time.Time       `json:"time"`
    Subject        string          `json:"subject,omitempty"`
    Data           json.RawMessage `json:"data"`
}
```

## Exactly-Once Semantics

### Idempotency + Deduplication

```go
// Idempotent consumer with deduplication
type IdempotentConsumer struct {
    handler    func(ctx context.Context, event Event) error
    seenStore  DeduplicationStore
    ttl        time.Duration
}

func (c *IdempotentConsumer) Process(ctx context.Context, event Event) error {
    // Check if already processed
    seen, err := c.seenStore.Exists(ctx, event.ID)
    if err != nil {
        return fmt.Errorf("dedup check failed: %w", err)
    }
    if seen {
        return nil // already processed, skip
    }

    // Process the event
    if err := c.handler(ctx, event); err != nil {
        return err
    }

    // Mark as processed (with TTL for cleanup)
    return c.seenStore.Mark(ctx, event.ID, c.ttl)
}

// Idempotency key for API requests
func (h *PaymentHandler) ChargePayment(ctx context.Context, req ChargeRequest) error {
    // Use idempotency key to prevent duplicate charges
    existing, err := h.store.GetByIdempotencyKey(ctx, req.IdempotencyKey)
    if err == nil && existing != nil {
        return nil // already processed
    }

    result, err := h.gateway.Charge(ctx, req.Amount, req.CardToken)
    if err != nil {
        return err
    }

    return h.store.Save(ctx, result, req.IdempotencyKey)
}
```

## Dead Letter Queues

```go
type DLQHandler struct {
    mainQueue  MessageBroker
    dlq        MessageBroker
    maxRetries int
}

func (h *DLQHandler) ProcessWithDLQ(ctx context.Context, msg Message, handler func(Message) error) {
    retryCount := getRetryCount(msg)

    if err := handler(msg); err != nil {
        if retryCount >= h.maxRetries {
            // Send to dead letter queue with error metadata
            msg.Headers["dlq-reason"] = err.Error()
            msg.Headers["dlq-original-topic"] = msg.Topic
            msg.Headers["dlq-retry-count"] = strconv.Itoa(retryCount)
            msg.Headers["dlq-timestamp"] = time.Now().Format(time.RFC3339)

            if dlqErr := h.dlq.Publish(ctx, "dlq."+msg.Topic, msg); dlqErr != nil {
                log.Printf("failed to send to DLQ: %v (original error: %v)", dlqErr, err)
            }
            h.mainQueue.Ack(ctx, msg) // remove from main queue
            return
        }

        // Retry with backoff
        msg.Headers["retry-count"] = strconv.Itoa(retryCount + 1)
        h.mainQueue.Nack(ctx, msg)
    } else {
        h.mainQueue.Ack(ctx, msg)
    }
}
```

## Transactional Outbox Pattern

```go
// Write event to outbox table in same transaction as state change
func (s *OrderService) CreateOrder(ctx context.Context, order Order) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 1. Write state change
    _, err = tx.ExecContext(ctx,
        `INSERT INTO orders (id, customer_id, total, status)
         VALUES ($1, $2, $3, $4)`,
        order.ID, order.CustomerID, order.Total, "created",
    )
    if err != nil {
        return err
    }

    // 2. Write event to outbox (same transaction!)
    eventData, _ := json.Marshal(order)
    _, err = tx.ExecContext(ctx,
        `INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload)
         VALUES ($1, $2, $3, $4, $5)`,
        uuid.New().String(), "Order", order.ID, "OrderCreated", eventData,
    )
    if err != nil {
        return err
    }

    return tx.Commit()
}

// Outbox poller publishes events to message broker
func (p *OutboxPoller) Poll(ctx context.Context) error {
    rows, err := p.db.QueryContext(ctx,
        `SELECT id, aggregate_type, aggregate_id, event_type, payload
         FROM outbox WHERE published = false
         ORDER BY created_at LIMIT 100`)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var event OutboxEvent
        rows.Scan(&event.ID, &event.AggregateType, &event.AggregateID,
            &event.EventType, &event.Payload)

        if err := p.broker.Publish(ctx, event.EventType, event.ToMessage()); err != nil {
            return err
        }

        p.db.ExecContext(ctx, `UPDATE outbox SET published = true WHERE id = $1`, event.ID)
    }
    return nil
}
```

## Change Data Capture (CDC)

```
Database ──→ WAL/Binlog ──→ Debezium ──→ Kafka ──→ Consumers

Debezium captures:
  INSERT → create event with "after" state
  UPDATE → update event with "before" and "after" state
  DELETE → delete event with "before" state
```

```json
{
    "schema": {},
    "payload": {
        "before": null,
        "after": {
            "id": 1,
            "name": "Alice",
            "email": "alice@example.com"
        },
        "source": {
            "version": "2.5.0",
            "connector": "postgresql",
            "name": "dbserver1",
            "ts_ms": 1712234400000,
            "db": "mydb",
            "schema": "public",
            "table": "users"
        },
        "op": "c",
        "ts_ms": 1712234400123
    }
}
```

## Event Versioning

```go
// Upcasting: transform old event versions to new format
type EventUpcaster interface {
    CanUpcast(eventType string, version int) bool
    Upcast(event Event) Event
}

type OrderCreatedV1ToV2Upcaster struct{}

func (u *OrderCreatedV1ToV2Upcaster) CanUpcast(eventType string, version int) bool {
    return eventType == "OrderCreated" && version == 1
}

func (u *OrderCreatedV1ToV2Upcaster) Upcast(event Event) Event {
    // V1: {customerId, items, total}
    // V2: {customerId, items, total, currency, createdAt}
    var v1 map[string]any
    json.Unmarshal(event.Data, &v1)

    v1["currency"] = "USD"              // default for old events
    v1["createdAt"] = event.Timestamp   // derive from event timestamp

    event.Data, _ = json.Marshal(v1)
    event.Version = 2
    return event
}

// Apply upcasters when loading events
func LoadWithUpcasting(events []Event, upcasters []EventUpcaster) []Event {
    result := make([]Event, len(events))
    for i, event := range events {
        for _, upcaster := range upcasters {
            if upcaster.CanUpcast(event.Type, event.Version) {
                event = upcaster.Upcast(event)
            }
        }
        result[i] = event
    }
    return result
}
```

## Event-Carried State Transfer

```go
// Include enough state in the event that consumers don't need to query back
// BAD: minimal event — consumer must call back to order service
type OrderShippedMinimal struct {
    OrderID string `json:"order_id"`
}

// GOOD: carries necessary state — consumer is self-sufficient
type OrderShippedRich struct {
    OrderID       string    `json:"order_id"`
    CustomerID    string    `json:"customer_id"`
    CustomerEmail string    `json:"customer_email"`
    ShippingAddr  Address   `json:"shipping_address"`
    Items         []Item    `json:"items"`
    TrackingNum   string    `json:"tracking_number"`
    Carrier       string    `json:"carrier"`
    ShippedAt     time.Time `json:"shipped_at"`
}
// Notification service can send email without querying order service
```

## Tips

- Event sourcing is not required for CQRS — they are independent patterns often used together
- Snapshot every N events (50-100) to keep replay times reasonable
- Use the transactional outbox when you cannot afford to lose events (no dual-write)
- CloudEvents provides a standard envelope; use it for interoperability across teams
- Dead letter queues need monitoring and alerting — events there represent data loss risk
- Event versioning is inevitable; plan for upcasting from day one
- CDC with Debezium is lower-risk than the outbox pattern but requires infrastructure
- Idempotency keys should be provided by the client, not generated server-side

## See Also

- `detail/patterns/event-driven-architecture.md` — ordering guarantees, Lamport timestamps
- `sheets/patterns/distributed-systems.md` — consistency models, CRDTs
- `sheets/patterns/microservices-patterns.md` — sagas with events

## References

- "Event Sourcing" by Martin Fowler: https://martinfowler.com/eaaDev/EventSourcing.html
- CloudEvents Specification: https://cloudevents.io/
- "Designing Event-Driven Systems" by Ben Stopford (Confluent, 2018)
- Debezium Documentation: https://debezium.io/documentation/
