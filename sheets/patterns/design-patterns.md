# Design Patterns (GoF, SOLID, and Go-Idiomatic)

A practitioner's guide to the 23 Gang of Four patterns, SOLID principles, and Go-specific idioms for building maintainable software.

## Creational Patterns

### Factory Method

```go
// Define the interface
type Storage interface {
    Save(key string, data []byte) error
    Load(key string) ([]byte, error)
}

// Factory function — Go-idiomatic replacement for Factory Method
func NewStorage(kind string, cfg Config) (Storage, error) {
    switch kind {
    case "s3":
        return NewS3Storage(cfg)
    case "disk":
        return NewDiskStorage(cfg)
    case "memory":
        return NewMemoryStorage()
    default:
        return nil, fmt.Errorf("unknown storage kind: %s", kind)
    }
}
```

```python
# Python — class-based factory method
from abc import ABC, abstractmethod

class Notifier(ABC):
    @abstractmethod
    def send(self, message: str) -> None: ...

class EmailNotifier(Notifier):
    def send(self, message: str) -> None:
        send_email(message)

class SlackNotifier(Notifier):
    def send(self, message: str) -> None:
        post_to_slack(message)

def create_notifier(channel: str) -> Notifier:
    factories = {"email": EmailNotifier, "slack": SlackNotifier}
    return factories[channel]()
```

### Abstract Factory

```go
// Family of related objects
type UIFactory interface {
    CreateButton() Button
    CreateCheckbox() Checkbox
    CreateTextField() TextField
}

type MaterialFactory struct{}
func (m MaterialFactory) CreateButton() Button       { return &MaterialButton{} }
func (m MaterialFactory) CreateCheckbox() Checkbox    { return &MaterialCheckbox{} }
func (m MaterialFactory) CreateTextField() TextField  { return &MaterialTextField{} }

type CupertinoFactory struct{}
func (c CupertinoFactory) CreateButton() Button      { return &CupertinoButton{} }
func (c CupertinoFactory) CreateCheckbox() Checkbox   { return &CupertinoCheckbox{} }
func (c CupertinoFactory) CreateTextField() TextField { return &CupertinoTextField{} }
```

### Builder

```go
// Functional options pattern — Go's idiomatic Builder
type Server struct {
    host    string
    port    int
    timeout time.Duration
    maxConn int
    tls     bool
}

type Option func(*Server)

func WithPort(port int) Option {
    return func(s *Server) { s.port = port }
}

func WithTimeout(d time.Duration) Option {
    return func(s *Server) { s.timeout = d }
}

func WithTLS(enabled bool) Option {
    return func(s *Server) { s.tls = enabled }
}

func WithMaxConnections(n int) Option {
    return func(s *Server) { s.maxConn = n }
}

func NewServer(host string, opts ...Option) *Server {
    s := &Server{
        host:    host,
        port:    8080,
        timeout: 30 * time.Second,
        maxConn: 100,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Usage
srv := NewServer("0.0.0.0",
    WithPort(9090),
    WithTLS(true),
    WithTimeout(60*time.Second),
)
```

```python
# Python — classic Builder with method chaining
class QueryBuilder:
    def __init__(self, table: str):
        self._table = table
        self._conditions = []
        self._order = None
        self._limit = None

    def where(self, condition: str) -> "QueryBuilder":
        self._conditions.append(condition)
        return self

    def order_by(self, field: str) -> "QueryBuilder":
        self._order = field
        return self

    def limit(self, n: int) -> "QueryBuilder":
        self._limit = n
        return self

    def build(self) -> str:
        q = f"SELECT * FROM {self._table}"
        if self._conditions:
            q += " WHERE " + " AND ".join(self._conditions)
        if self._order:
            q += f" ORDER BY {self._order}"
        if self._limit:
            q += f" LIMIT {self._limit}"
        return q

query = (QueryBuilder("users")
    .where("age > 18")
    .where("active = true")
    .order_by("name")
    .limit(50)
    .build())
```

### Singleton

```go
// Go — sync.Once guarantees thread-safe single initialization
var (
    dbInstance *Database
    dbOnce    sync.Once
)

func GetDatabase() *Database {
    dbOnce.Do(func() {
        dbInstance = &Database{
            pool: openConnectionPool(),
        }
    })
    return dbInstance
}
```

### Prototype

```go
type Prototype interface {
    Clone() Prototype
}

type Document struct {
    Title    string
    Content  string
    Metadata map[string]string
}

func (d *Document) Clone() *Document {
    meta := make(map[string]string, len(d.Metadata))
    for k, v := range d.Metadata {
        meta[k] = v
    }
    return &Document{
        Title:    d.Title,
        Content:  d.Content,
        Metadata: meta,
    }
}
```

## Structural Patterns

### Adapter

```go
// Adapt an incompatible interface to the expected one
type OldLogger struct{}
func (o *OldLogger) LogMessage(msg string, severity int) { /* ... */ }

// Target interface
type Logger interface {
    Info(msg string)
    Error(msg string)
}

// Adapter
type LoggerAdapter struct {
    old *OldLogger
}

func (a *LoggerAdapter) Info(msg string)  { a.old.LogMessage(msg, 1) }
func (a *LoggerAdapter) Error(msg string) { a.old.LogMessage(msg, 3) }
```

### Bridge

```go
// Decouple abstraction from implementation
type Renderer interface {
    RenderCircle(radius float64)
    RenderRect(width, height float64)
}

type Shape struct {
    renderer Renderer
}

type Circle struct {
    Shape
    radius float64
}

func (c *Circle) Draw() {
    c.renderer.RenderCircle(c.radius)
}

// Implementations
type OpenGLRenderer struct{}
func (o *OpenGLRenderer) RenderCircle(r float64) { /* OpenGL calls */ }
func (o *OpenGLRenderer) RenderRect(w, h float64) { /* OpenGL calls */ }

type VulkanRenderer struct{}
func (v *VulkanRenderer) RenderCircle(r float64) { /* Vulkan calls */ }
func (v *VulkanRenderer) RenderRect(w, h float64) { /* Vulkan calls */ }
```

### Composite

```go
// Tree structure where leaves and composites share an interface
type Component interface {
    Execute() string
    Add(child Component)
    GetChildren() []Component
}

type File struct {
    name string
}
func (f *File) Execute() string         { return f.name }
func (f *File) Add(child Component)     { /* no-op for leaf */ }
func (f *File) GetChildren() []Component { return nil }

type Directory struct {
    name     string
    children []Component
}
func (d *Directory) Execute() string {
    results := d.name + "/\n"
    for _, c := range d.children {
        results += "  " + c.Execute() + "\n"
    }
    return results
}
func (d *Directory) Add(child Component)     { d.children = append(d.children, child) }
func (d *Directory) GetChildren() []Component { return d.children }
```

### Decorator

```go
// Wrap behavior around an existing interface implementation
type Handler interface {
    Handle(req Request) Response
}

// Logging decorator
func WithLogging(h Handler, logger *slog.Logger) Handler {
    return HandlerFunc(func(req Request) Response {
        logger.Info("handling request", "path", req.Path)
        start := time.Now()
        resp := h.Handle(req)
        logger.Info("request complete", "duration", time.Since(start))
        return resp
    })
}

// Auth decorator
func WithAuth(h Handler, auth Authenticator) Handler {
    return HandlerFunc(func(req Request) Response {
        if !auth.Validate(req.Token) {
            return Response{Status: 401}
        }
        return h.Handle(req)
    })
}

// Compose decorators
handler := WithLogging(WithAuth(baseHandler, auth), logger)
```

### Facade

```go
// Simplified interface over a complex subsystem
type OrderFacade struct {
    inventory *InventoryService
    payment   *PaymentService
    shipping  *ShippingService
    notify    *NotificationService
}

func (f *OrderFacade) PlaceOrder(order Order) error {
    if err := f.inventory.Reserve(order.Items); err != nil {
        return fmt.Errorf("inventory: %w", err)
    }
    if err := f.payment.Charge(order.Total, order.PaymentMethod); err != nil {
        f.inventory.Release(order.Items)
        return fmt.Errorf("payment: %w", err)
    }
    trackingID, err := f.shipping.Ship(order.Address, order.Items)
    if err != nil {
        f.payment.Refund(order.Total)
        f.inventory.Release(order.Items)
        return fmt.Errorf("shipping: %w", err)
    }
    f.notify.Send(order.CustomerEmail, "Order shipped: "+trackingID)
    return nil
}
```

### Flyweight

```go
// Share common state across many objects
type CharacterStyle struct {
    Font  string
    Size  int
    Color color.RGBA
}

type CharacterStyleFactory struct {
    cache map[string]*CharacterStyle
}

func (f *CharacterStyleFactory) GetStyle(font string, size int, c color.RGBA) *CharacterStyle {
    key := fmt.Sprintf("%s-%d-%v", font, size, c)
    if style, ok := f.cache[key]; ok {
        return style // reuse existing
    }
    style := &CharacterStyle{Font: font, Size: size, Color: c}
    f.cache[key] = style
    return style
}
```

### Proxy

```go
// Control access to an object
type DataStore interface {
    Get(key string) (string, error)
    Set(key, value string) error
}

type CachingProxy struct {
    store DataStore
    cache map[string]string
    mu    sync.RWMutex
}

func (p *CachingProxy) Get(key string) (string, error) {
    p.mu.RLock()
    if val, ok := p.cache[key]; ok {
        p.mu.RUnlock()
        return val, nil
    }
    p.mu.RUnlock()

    val, err := p.store.Get(key)
    if err != nil {
        return "", err
    }

    p.mu.Lock()
    p.cache[key] = val
    p.mu.Unlock()
    return val, nil
}
```

## Behavioral Patterns

### Chain of Responsibility

```go
type Middleware func(http.Handler) http.Handler

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "unauthorized", 401)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Chain: logging -> auth -> handler
handler := LoggingMiddleware(AuthMiddleware(appHandler))
```

### Command

```go
type Command interface {
    Execute() error
    Undo() error
}

type InsertTextCommand struct {
    doc      *Document
    position int
    text     string
}

func (c *InsertTextCommand) Execute() error {
    return c.doc.Insert(c.position, c.text)
}

func (c *InsertTextCommand) Undo() error {
    return c.doc.Delete(c.position, len(c.text))
}

// Command history for undo/redo
type History struct {
    commands []Command
    current  int
}

func (h *History) Execute(cmd Command) error {
    if err := cmd.Execute(); err != nil {
        return err
    }
    h.commands = append(h.commands[:h.current], cmd)
    h.current++
    return nil
}
```

### Observer

```go
type EventType string

type Event struct {
    Type    EventType
    Payload any
}

type EventBus struct {
    mu       sync.RWMutex
    handlers map[EventType][]func(Event)
}

func (b *EventBus) Subscribe(t EventType, handler func(Event)) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.handlers[t] = append(b.handlers[t], handler)
}

func (b *EventBus) Publish(e Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, h := range b.handlers[e.Type] {
        go h(e) // async notification
    }
}
```

### Strategy

```go
type Compressor interface {
    Compress(data []byte) ([]byte, error)
}

type GzipCompressor struct{}
func (g GzipCompressor) Compress(data []byte) ([]byte, error) { /* gzip */ return nil, nil }

type ZstdCompressor struct{}
func (z ZstdCompressor) Compress(data []byte) ([]byte, error) { /* zstd */ return nil, nil }

type Archiver struct {
    compressor Compressor
}

func (a *Archiver) Archive(files []File) ([]byte, error) {
    combined := combineFiles(files)
    return a.compressor.Compress(combined)
}

// Swap strategy at runtime
archiver := &Archiver{compressor: ZstdCompressor{}}
```

### State

```go
type State interface {
    Handle(ctx *TCPConnection, data []byte) State
}

type ListenState struct{}
func (s *ListenState) Handle(ctx *TCPConnection, data []byte) State {
    if isSYN(data) {
        ctx.sendSYNACK()
        return &SynReceivedState{}
    }
    return s
}

type SynReceivedState struct{}
func (s *SynReceivedState) Handle(ctx *TCPConnection, data []byte) State {
    if isACK(data) {
        return &EstablishedState{}
    }
    return s
}

type EstablishedState struct{}
func (s *EstablishedState) Handle(ctx *TCPConnection, data []byte) State {
    if isFIN(data) {
        ctx.sendACK()
        return &CloseWaitState{}
    }
    ctx.processData(data)
    return s
}
```

## SOLID Principles

### Single Responsibility

```go
// BAD: User struct handles persistence and validation
// GOOD: Separate concerns
type User struct { Name, Email string }
type UserValidator struct{}
func (v *UserValidator) Validate(u User) error { /* ... */ return nil }
type UserRepository struct{ db *sql.DB }
func (r *UserRepository) Save(u User) error { /* ... */ return nil }
```

### Open/Closed

```go
// Open for extension, closed for modification
// Use interfaces and composition instead of modifying existing code
type Notifier interface {
    Notify(user User, message string) error
}

// Add new notification types without changing existing code
type PushNotifier struct{ /* ... */ }
func (p *PushNotifier) Notify(u User, msg string) error { /* ... */ return nil }
```

### Liskov Substitution

```go
// Any implementation of an interface must be substitutable
type Shape interface {
    Area() float64
}

// Rectangle and Square should both satisfy Shape
// but Square should NOT extend Rectangle (classic violation)
type Rectangle struct{ Width, Height float64 }
func (r Rectangle) Area() float64 { return r.Width * r.Height }

type Square struct{ Side float64 }
func (s Square) Area() float64 { return s.Side * s.Side }
```

### Interface Segregation

```go
// BAD: fat interface
type Worker interface {
    Work()
    Eat()
    Sleep()
}

// GOOD: small, focused interfaces
type Workable interface { Work() }
type Feedable interface { Eat() }
type Sleepable interface { Sleep() }

// Compose as needed
type HumanWorker struct{}
func (h *HumanWorker) Work()  {}
func (h *HumanWorker) Eat()   {}
func (h *HumanWorker) Sleep() {}

type RobotWorker struct{}
func (r *RobotWorker) Work() {} // robots don't eat or sleep
```

### Dependency Inversion

```go
// High-level modules should not depend on low-level modules
// Both should depend on abstractions

// Abstraction
type MessageQueue interface {
    Publish(topic string, msg []byte) error
    Subscribe(topic string) (<-chan []byte, error)
}

// High-level module depends on abstraction
type OrderService struct {
    queue MessageQueue // not *RabbitMQ or *Kafka
}

// Low-level modules implement the abstraction
type RabbitMQ struct{ /* ... */ }
func (r *RabbitMQ) Publish(topic string, msg []byte) error { return nil }
func (r *RabbitMQ) Subscribe(topic string) (<-chan []byte, error) { return nil, nil }
```

## Go-Idiomatic Patterns

### Interface Embedding

```go
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
type Closer interface { Close() error }

// Compose interfaces
type ReadWriter interface {
    Reader
    Writer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

### Constructor Functions

```go
// Go convention: NewXxx returns a pointer, validates inputs
func NewServer(addr string, port int) (*Server, error) {
    if port < 1 || port > 65535 {
        return nil, fmt.Errorf("invalid port: %d", port)
    }
    return &Server{
        addr:    addr,
        port:    port,
        started: time.Now(),
    }, nil
}
```

## Anti-Patterns to Avoid

| Anti-Pattern | Problem | Fix |
|---|---|---|
| God Object | Single struct does everything | Split by responsibility |
| Premature Abstraction | Interface before 2nd implementation | Wait for concrete need |
| Singleton Abuse | Global mutable state | Dependency injection |
| Deep Inheritance | Fragile base class (N/A in Go) | Composition |
| Shotgun Surgery | One change touches many files | Improve cohesion |
| Feature Envy | Method uses another struct's data more | Move method to that struct |

## Tips

- In Go, accept interfaces and return structs — this maximizes flexibility
- Keep interfaces small (1-3 methods); the `io.Reader` interface has one method
- Prefer composition over inheritance in every language, but especially in Go
- The functional options pattern replaces the need for Builder in most Go code
- Not every problem needs a design pattern — simplicity beats cleverness
- Test with interfaces: inject mocks/stubs through constructor parameters

## See Also

- `detail/patterns/design-patterns.md` — coupling metrics, complexity analysis
- `sheets/patterns/microservices-patterns.md` — distributed system patterns
- `sheets/patterns/concurrency-patterns.md` — concurrent design patterns

## References

- "Design Patterns" by Gamma, Helm, Johnson, Vlissides (GoF, 1994)
- "Effective Go": https://go.dev/doc/effective_go
- SOLID Principles (Robert C. Martin)
- "Go Proverbs" by Rob Pike: https://go-proverbs.github.io/
