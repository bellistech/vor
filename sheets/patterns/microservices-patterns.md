# Microservices Patterns (Resilience, Communication, and Decomposition)

A field guide to patterns that keep distributed services running reliably under real-world failure conditions.

## Circuit Breaker

### States and Transitions

```
     ┌──────────┐
     │  CLOSED  │ ← Normal operation, requests pass through
     └────┬─────┘
          │ failure count >= threshold
          v
     ┌──────────┐
     │   OPEN   │ ← All requests fail-fast (no downstream calls)
     └────┬─────┘
          │ timeout expires
          v
     ┌──────────┐
     │HALF-OPEN │ ← Allow limited probe requests
     └────┬─────┘
          │ probe succeeds → CLOSED
          │ probe fails    → OPEN
```

### Go Implementation

```go
type State int

const (
    Closed   State = iota
    Open
    HalfOpen
)

type CircuitBreaker struct {
    mu               sync.Mutex
    state            State
    failureCount     int
    successCount     int
    failureThreshold int
    successThreshold int // required successes in half-open to close
    timeout          time.Duration
    lastFailure      time.Time
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    cb.mu.Lock()
    if cb.state == Open {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = HalfOpen
            cb.successCount = 0
        } else {
            cb.mu.Unlock()
            return ErrCircuitOpen
        }
    }
    cb.mu.Unlock()

    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failureCount++
        cb.lastFailure = time.Now()
        if cb.failureCount >= cb.failureThreshold {
            cb.state = Open
        }
        return err
    }

    if cb.state == HalfOpen {
        cb.successCount++
        if cb.successCount >= cb.successThreshold {
            cb.state = Closed
            cb.failureCount = 0
        }
    } else {
        cb.failureCount = 0
    }
    return nil
}
```

### Configuration Guidelines

| Parameter | Typical Value | Notes |
|---|---|---|
| Failure threshold | 5-10 | Consecutive failures to trip |
| Success threshold | 3-5 | Successes in half-open to close |
| Timeout | 30-60s | Time in open before probing |
| Monitor window | 60s | Sliding window for failure rate |

## Bulkhead

### Thread Pool Isolation

```go
// Bulkhead limits concurrent access to a resource
type Bulkhead struct {
    sem chan struct{}
}

func NewBulkhead(maxConcurrent int) *Bulkhead {
    return &Bulkhead{
        sem: make(chan struct{}, maxConcurrent),
    }
}

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
    select {
    case b.sem <- struct{}{}:
        defer func() { <-b.sem }()
        return fn()
    case <-ctx.Done():
        return fmt.Errorf("bulkhead rejected: %w", ctx.Err())
    }
}

// Per-service bulkheads prevent cascade
var (
    paymentBulkhead   = NewBulkhead(20)
    inventoryBulkhead = NewBulkhead(50)
    userBulkhead      = NewBulkhead(100)
)
```

## Sidecar Pattern

```yaml
# Kubernetes pod with sidecar
apiVersion: v1
kind: Pod
metadata:
  name: app-with-sidecar
spec:
  containers:
  - name: app
    image: myapp:latest
    ports:
    - containerPort: 8080
  - name: envoy-proxy
    image: envoyproxy/envoy:v1.28
    ports:
    - containerPort: 9901
  - name: log-collector
    image: fluentd:latest
    volumeMounts:
    - name: shared-logs
      mountPath: /var/log/app
  volumes:
  - name: shared-logs
    emptyDir: {}
```

### Common Sidecar Uses

| Sidecar | Purpose | Examples |
|---|---|---|
| Service mesh proxy | mTLS, routing, observability | Envoy, Linkerd-proxy |
| Log collector | Ship logs to central store | Fluentd, Filebeat |
| Config watcher | Reload config on change | Custom, Consul template |
| Cert manager | Auto-rotate TLS certificates | cert-manager |

## Ambassador Pattern

```go
// Ambassador wraps external service calls with cross-cutting concerns
type Ambassador struct {
    client         *http.Client
    circuitBreaker *CircuitBreaker
    retrier        *Retrier
    tracer         trace.Tracer
}

func (a *Ambassador) Call(ctx context.Context, req *http.Request) (*http.Response, error) {
    ctx, span := a.tracer.Start(ctx, "ambassador.call")
    defer span.End()

    var resp *http.Response
    err := a.circuitBreaker.Execute(func() error {
        return a.retrier.Do(ctx, func() error {
            var err error
            resp, err = a.client.Do(req.WithContext(ctx))
            if err != nil {
                return err
            }
            if resp.StatusCode >= 500 {
                return fmt.Errorf("server error: %d", resp.StatusCode)
            }
            return nil
        })
    })
    return resp, err
}
```

## Saga Pattern

### Orchestration (Central Coordinator)

```go
type SagaOrchestrator struct {
    steps []SagaStep
}

type SagaStep struct {
    Name       string
    Execute    func(ctx context.Context, data any) (any, error)
    Compensate func(ctx context.Context, data any) error
}

func (s *SagaOrchestrator) Run(ctx context.Context, data any) error {
    var completed []int
    current := data

    for i, step := range s.steps {
        result, err := step.Execute(ctx, current)
        if err != nil {
            // Compensate in reverse order
            for j := len(completed) - 1; j >= 0; j-- {
                idx := completed[j]
                if compErr := s.steps[idx].Compensate(ctx, current); compErr != nil {
                    log.Printf("compensation failed for %s: %v",
                        s.steps[idx].Name, compErr)
                }
            }
            return fmt.Errorf("saga failed at step %s: %w", step.Name, err)
        }
        completed = append(completed, i)
        current = result
    }
    return nil
}

// Usage
saga := &SagaOrchestrator{
    steps: []SagaStep{
        {Name: "reserve_inventory", Execute: reserveInventory, Compensate: releaseInventory},
        {Name: "charge_payment", Execute: chargePayment, Compensate: refundPayment},
        {Name: "create_shipment", Execute: createShipment, Compensate: cancelShipment},
        {Name: "send_confirmation", Execute: sendConfirmation, Compensate: noOp},
    },
}
```

### Choreography (Event-Driven)

```
Order Created  ──→  Inventory Service ──→  Inventory Reserved
                                                  │
Payment Service  ←─────────────────────────────────┘
       │
Payment Charged  ──→  Shipping Service ──→  Shipment Created
                                                  │
Notification Service  ←───────────────────────────┘
```

| Aspect | Orchestration | Choreography |
|---|---|---|
| Coupling | Central coordinator | Loosely coupled |
| Visibility | Easy to trace flow | Harder to follow |
| Single point of failure | Orchestrator | None |
| Complexity | In orchestrator | Spread across services |

## Service Discovery

### Client-Side Discovery

```go
// Client queries registry and load-balances
type ServiceRegistry interface {
    Register(name, addr string, port int) error
    Deregister(name, addr string) error
    Discover(name string) ([]Endpoint, error)
    Watch(name string) (<-chan []Endpoint, error)
}

func (c *Client) callService(ctx context.Context, name string) error {
    endpoints, err := c.registry.Discover(name)
    if err != nil {
        return err
    }
    target := c.loadBalancer.Pick(endpoints) // round-robin, random, least-conn
    return c.call(ctx, target)
}
```

### Server-Side Discovery

```
Client ──→ Load Balancer ──→ Service Instance
                │
           Registry Query
```

## Health Checks

```go
// Three types of health checks for Kubernetes
type HealthChecker struct {
    checks map[string]func() error
}

// Liveness — is the process alive? Failure = restart container
func (h *HealthChecker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
    // Check for deadlocks, unrecoverable state
    if err := h.checks["liveness"](); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{"status": "dead", "error": err.Error()})
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// Readiness — can the process serve traffic? Failure = remove from LB
func (h *HealthChecker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
    // Check dependencies: DB, cache, downstream services
    errors := make(map[string]string)
    for name, check := range h.checks {
        if err := check(); err != nil {
            errors[name] = err.Error()
        }
    }
    if len(errors) > 0 {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]any{"status": "not_ready", "errors": errors})
        return
    }
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// Startup — has the process finished initializing? Failure during startup = wait
func (h *HealthChecker) StartupHandler(w http.ResponseWriter, r *http.Request) {
    if !h.initialized.Load() {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

## Retry with Exponential Backoff and Jitter

```go
func RetryWithBackoff(ctx context.Context, maxRetries int, fn func() error) error {
    baseDelay := 100 * time.Millisecond
    maxDelay := 30 * time.Second

    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }

        if attempt == maxRetries {
            return fmt.Errorf("all %d retries exhausted: %w", maxRetries, err)
        }

        // Exponential backoff with full jitter
        expDelay := baseDelay * time.Duration(1<<uint(attempt))
        if expDelay > maxDelay {
            expDelay = maxDelay
        }
        jitter := time.Duration(rand.Int63n(int64(expDelay)))

        select {
        case <-time.After(jitter):
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return nil
}
```

## Deadline and Timeout Propagation

```go
// Propagate deadlines across service boundaries
func (s *OrderService) CreateOrder(ctx context.Context, req OrderRequest) (*Order, error) {
    // Inherit deadline from incoming request
    deadline, ok := ctx.Deadline()
    if !ok {
        // Set default if no upstream deadline
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
        defer cancel()
        deadline, _ = ctx.Deadline()
    }

    // Subtract buffer for local processing
    remaining := time.Until(deadline) - 500*time.Millisecond
    if remaining <= 0 {
        return nil, status.Error(codes.DeadlineExceeded, "insufficient time remaining")
    }

    downstreamCtx, cancel := context.WithTimeout(ctx, remaining)
    defer cancel()

    // Propagate via gRPC metadata
    md := metadata.Pairs("x-deadline", deadline.Format(time.RFC3339Nano))
    downstreamCtx = metadata.NewOutgoingContext(downstreamCtx, md)

    return s.inventoryClient.Reserve(downstreamCtx, req.Items)
}
```

## Distributed Tracing (W3C Trace Context)

```go
// W3C Trace Context headers
// traceparent: 00-<trace-id>-<parent-span-id>-<trace-flags>
// tracestate:  vendor1=value1,vendor2=value2

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (s *Service) HandleRequest(ctx context.Context, req Request) (Response, error) {
    tracer := otel.Tracer("order-service")
    ctx, span := tracer.Start(ctx, "HandleRequest",
        trace.WithAttributes(
            attribute.String("order.id", req.OrderID),
            attribute.String("customer.id", req.CustomerID),
        ),
    )
    defer span.End()

    // Downstream calls automatically propagate trace context
    result, err := s.paymentClient.Charge(ctx, req.Amount)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return Response{}, err
    }

    span.AddEvent("payment_completed",
        trace.WithAttributes(attribute.String("transaction_id", result.TxID)),
    )
    return Response{TxID: result.TxID}, nil
}
```

## Strangler Fig

```go
// Gradually migrate from monolith to microservices
func NewStranglerProxy(legacy, modern *url.URL) http.Handler {
    legacyProxy := httputil.NewSingleHostReverseProxy(legacy)
    modernProxy := httputil.NewSingleHostReverseProxy(modern)

    // Route table: paths migrated to new service
    migrated := map[string]bool{
        "/api/v2/users":    true,
        "/api/v2/products": true,
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if migrated[r.URL.Path] {
            modernProxy.ServeHTTP(w, r)
        } else {
            legacyProxy.ServeHTTP(w, r)
        }
    })
}
```

## Backends for Frontends (BFF)

```
Mobile App  ──→  Mobile BFF  ──→  User Service
                      │            Order Service
                      │            Product Service

Web App     ──→  Web BFF     ──→  User Service
                      │            Order Service
                      │            Product Service

CLI/API     ──→  API Gateway ──→  (direct access)
```

```go
// Mobile BFF — aggregates and shapes data for mobile clients
func (bff *MobileBFF) GetHomeFeed(ctx context.Context, userID string) (*MobileFeed, error) {
    g, ctx := errgroup.WithContext(ctx)

    var user *User
    var orders []*OrderSummary
    var recommendations []*Product

    g.Go(func() error {
        var err error
        user, err = bff.userClient.GetProfile(ctx, userID)
        return err
    })
    g.Go(func() error {
        var err error
        orders, err = bff.orderClient.GetRecent(ctx, userID, 5) // only 5 for mobile
        return err
    })
    g.Go(func() error {
        var err error
        recommendations, err = bff.recClient.GetForUser(ctx, userID, 10)
        return err
    })

    if err := g.Wait(); err != nil {
        return nil, err
    }

    // Shape response for mobile — minimal payload
    return &MobileFeed{
        Greeting:        fmt.Sprintf("Hi %s", user.FirstName),
        RecentOrders:    summarizeForMobile(orders),
        Recommendations: compactProducts(recommendations),
    }, nil
}
```

## Tips

- Circuit breakers should be per-downstream-service, not global
- Use bulkheads to prevent one failing dependency from consuming all threads/goroutines
- Prefer choreography for simple flows, orchestration for complex multi-step transactions
- Always propagate deadlines — a request without a deadline can block indefinitely
- Health check endpoints should be unauthenticated and fast (< 100ms)
- Retry only on transient errors (5xx, timeouts); never retry 4xx client errors
- The strangler fig pattern works best when the monolith has clear URL-based boundaries
- BFFs should be owned by the frontend team that consumes them

## See Also

- `detail/patterns/microservices-patterns.md` — circuit breaker state machines, backoff math
- `sheets/patterns/distributed-systems.md` — consensus, consistency, replication
- `sheets/patterns/event-driven-architecture.md` — event sourcing and CQRS

## References

- "Release It!" by Michael Nygard (2nd edition, 2018)
- "Building Microservices" by Sam Newman (2nd edition, 2021)
- W3C Trace Context: https://www.w3.org/TR/trace-context/
- Microsoft Cloud Design Patterns: https://learn.microsoft.com/en-us/azure/architecture/patterns/
