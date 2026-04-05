# Twelve-Factor App (Go + Docker)

Complete reference for the Twelve-Factor methodology with Go and Docker examples — all 12 factors with practical implementation patterns, plus modern additions.

## I. Codebase — One Codebase, Many Deploys

One repo tracked in version control, many deploys (dev, staging, production).

```
myapp/                     # one git repository
  cmd/server/main.go       # entry point
  internal/                # business logic
  go.mod                   # dependency manifest
  Dockerfile               # build recipe
  docker-compose.yml       # local dev
  docker-compose.prod.yml  # production overlay
```

```bash
# Same codebase, different deploys
git remote -v
# origin  git@github.com:org/myapp.git

# Deploy to staging
git push staging main

# Deploy to production
git push production main
```

Anti-pattern: multiple apps in separate repos sharing code via copy-paste. Use a monorepo or shared modules instead.

## II. Dependencies — Explicitly Declare and Isolate

Never rely on system-wide packages. Declare all dependencies explicitly.

```go
// go.mod — explicit dependency declaration
module github.com/org/myapp

go 1.24

require (
    github.com/gorilla/mux v1.8.1
    github.com/lib/pq v1.10.9
    go.uber.org/zap v1.27.0
)
```

```dockerfile
# Dockerfile — isolated dependency installation
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download          # download dependencies in isolation

COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server
```

```bash
# Verify no implicit dependencies
go mod verify
go mod tidy

# Vendor for reproducible builds
go mod vendor
go build -mod=vendor ./...
```

## III. Config — Store Config in the Environment

Strict separation of config from code. Config varies between deploys; code does not.

```go
// config.go — read from environment
type Config struct {
    Port        int    `env:"PORT" envDefault:"8080"`
    DatabaseURL string `env:"DATABASE_URL,required"`
    RedisURL    string `env:"REDIS_URL" envDefault:"localhost:6379"`
    LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
    Debug       bool   `env:"DEBUG" envDefault:"false"`
}

func LoadConfig() (*Config, error) {
    var cfg Config
    if err := env.Parse(&cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }
    return &cfg, nil
}
```

```go
// Alternative: standard library only
func LoadConfig() *Config {
    return &Config{
        Port:        getEnvInt("PORT", 8080),
        DatabaseURL: mustGetEnv("DATABASE_URL"),
        RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
    }
}

func mustGetEnv(key string) string {
    val := os.Getenv(key)
    if val == "" {
        log.Fatalf("required environment variable %s is not set", key)
    }
    return val
}

func getEnv(key, fallback string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return fallback
}
```

```bash
# Set config per deploy
PORT=8080 DATABASE_URL=postgres://user:pass@db:5432/myapp ./server

# Docker
docker run -e PORT=8080 -e DATABASE_URL=postgres://... myapp
```

Anti-pattern: config files committed to the repo (`config.production.yml`), feature flags in code constants.

## IV. Backing Services — Treat as Attached Resources

Databases, queues, caches, SMTP servers — all are attached resources accessed via URL/config.

```go
// No distinction between local and third-party services
type App struct {
    DB    *sql.DB           // could be local PostgreSQL or RDS
    Cache *redis.Client     // could be local Redis or ElastiCache
    Queue *amqp.Connection  // could be local RabbitMQ or CloudAMQP
}

func NewApp(cfg *Config) (*App, error) {
    db, err := sql.Open("pgx", cfg.DatabaseURL)    // swap by changing env var
    if err != nil {
        return nil, err
    }

    cache := redis.NewClient(&redis.Options{
        Addr: cfg.RedisURL,                         // swap by changing env var
    })

    return &App{DB: db, Cache: cache}, nil
}
```

```yaml
# docker-compose.yml — local backing services
services:
  app:
    build: .
    environment:
      DATABASE_URL: postgres://user:pass@postgres:5432/myapp
      REDIS_URL: redis:6379
    depends_on:
      - postgres
      - redis
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_PASSWORD: pass
      POSTGRES_USER: user
      POSTGRES_DB: myapp
  redis:
    image: redis:7-alpine
```

## V. Build, Release, Run — Strictly Separate Stages

```
Build:   Code + Dependencies → Executable artifact
Release: Artifact + Config → Deployable release
Run:     Execute the release in the execution environment
```

```dockerfile
# Multi-stage Docker build — Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /server ./cmd/server

# Release stage — artifact + minimal runtime
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /server /server
ENTRYPOINT ["/server"]
```

```bash
# Build (creates artifact)
docker build -t myapp:v1.2.3 .

# Release (tag with config context)
docker tag myapp:v1.2.3 registry.example.com/myapp:v1.2.3

# Run (execute with config)
docker run -e DATABASE_URL=... registry.example.com/myapp:v1.2.3
```

## VI. Processes — Execute the App as Stateless Processes

Processes are stateless and share-nothing. Any persistent data is stored in a backing service.

```go
// GOOD — stateless handler, session in Redis
func (a *App) HandleRequest(w http.ResponseWriter, r *http.Request) {
    sessionID := r.Cookie("session_id")
    session, err := a.Cache.Get(r.Context(), "session:"+sessionID.Value).Result()
    // ...
}

// BAD — in-memory state (lost on restart/scale)
var sessions = map[string]*Session{} // DON'T DO THIS
```

```go
// GOOD — file processing with external storage
func (a *App) ProcessUpload(w http.ResponseWriter, r *http.Request) {
    file, _, err := r.FormFile("upload")
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    defer file.Close()

    // Store in object storage, not local filesystem
    key := fmt.Sprintf("uploads/%s/%s", userID, uuid.New())
    _, err = a.S3.PutObject(r.Context(), &s3.PutObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String(key),
        Body:   file,
    })
}
```

## VII. Port Binding — Export Services via Port Binding

The app is self-contained and binds to a port to serve requests.

```go
func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/api/v1/users", usersHandler)

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      mux,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Printf("listening on port %s", port)
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("server error: %v", err)
    }
}
```

No Apache, no Nginx in front (for the app itself). The app IS the web server.

## VIII. Concurrency — Scale Out via the Process Model

Scale by running more processes, not by making one process bigger.

```go
// Goroutines for internal concurrency
func (a *App) ProcessBatch(ctx context.Context, items []Item) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(runtime.NumCPU()) // bounded concurrency

    for _, item := range items {
        g.Go(func() error {
            return a.processItem(ctx, item)
        })
    }
    return g.Wait()
}
```

```yaml
# Horizontal scaling with Docker Compose
services:
  web:
    build: .
    deploy:
      replicas: 4
    environment:
      PORT: 8080

  worker:
    build: .
    command: ["/server", "--mode=worker"]
    deploy:
      replicas: 8
```

```yaml
# Kubernetes HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: myapp
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: myapp
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

## IX. Disposability — Maximize Robustness with Fast Startup and Graceful Shutdown

Processes can be started and stopped at a moment's notice.

```go
func main() {
    srv := &http.Server{
        Addr:    ":" + os.Getenv("PORT"),
        Handler: newRouter(),
    }

    // Start server
    go func() {
        log.Printf("server starting")
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("listen: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Printf("shutting down gracefully...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("forced shutdown: %v", err)
    }

    log.Printf("server stopped")
}
```

```dockerfile
# Use exec form to receive signals properly
ENTRYPOINT ["/server"]
# NOT: ENTRYPOINT /server  (shell form wraps in /bin/sh, signals go to sh)

# Health check for fast readiness detection
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s \
    CMD wget -q --spider http://localhost:8080/health || exit 1
```

## X. Dev/Prod Parity — Keep Development, Staging, and Production as Similar as Possible

Minimize the gap between development and production.

```yaml
# docker-compose.yml — mirrors production stack
services:
  app:
    build: .
    environment:
      DATABASE_URL: postgres://user:pass@postgres:5432/myapp
      REDIS_URL: redis:6379
      LOG_LEVEL: debug      # only difference from prod
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine    # SAME version as production
    environment:
      POSTGRES_PASSWORD: pass
      POSTGRES_USER: user
      POSTGRES_DB: myapp
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d myapp"]
      interval: 2s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine        # SAME version as production
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 2s
      timeout: 5s
      retries: 10
```

Anti-patterns:
- SQLite in dev, PostgreSQL in prod
- In-memory cache in dev, Redis in prod
- macOS file system in dev, ext4 in prod (case sensitivity differs)

## XI. Logs — Treat Logs as Event Streams

Write to stdout/stderr. Let the execution environment handle log routing.

```go
func main() {
    // Structured logging to stdout
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: parseLogLevel(os.Getenv("LOG_LEVEL")),
    }))
    slog.SetDefault(logger)

    slog.Info("server starting",
        "port", os.Getenv("PORT"),
        "version", version,
    )
}

// Output (structured JSON to stdout):
// {"time":"2025-01-15T10:30:00Z","level":"INFO","msg":"server starting","port":"8080","version":"1.2.3"}
```

```go
// HTTP middleware — request logging to stdout
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        ww := &responseWriter{ResponseWriter: w, statusCode: 200}

        next.ServeHTTP(ww, r)

        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", ww.statusCode,
            "duration_ms", time.Since(start).Milliseconds(),
            "remote_addr", r.RemoteAddr,
        )
    })
}
```

```yaml
# Docker handles log routing
services:
  app:
    build: .
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## XII. Admin Processes — Run Admin/Management Tasks as One-Off Processes

Database migrations, console sessions, one-time scripts — run as one-off processes in the same environment.

```go
// cmd/migrate/main.go — separate binary, same codebase
func main() {
    cfg := config.LoadConfig()
    db, err := sql.Open("pgx", cfg.DatabaseURL)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    if err := runMigrations(db); err != nil {
        log.Fatalf("migration failed: %v", err)
    }
    log.Println("migrations complete")
}
```

```bash
# Run in the same environment as the app
docker run --env-file .env myapp /migrate

# Kubernetes Job for migration
kubectl run migrate --image=myapp:v1.2.3 --restart=Never \
    --env="DATABASE_URL=$DATABASE_URL" \
    -- /migrate
```

```yaml
# Kubernetes Job
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: myapp:v1.2.3
          command: ["/migrate"]
          envFrom:
            - secretRef:
                name: app-secrets
      restartPolicy: Never
  backoffLimit: 3
```

## Modern Additions

### XIII. API-First

Design APIs before implementation. Use OpenAPI/gRPC definitions as contracts.

```yaml
# openapi.yaml
openapi: 3.1.0
info:
  title: My API
  version: 1.0.0
paths:
  /api/v1/users:
    get:
      summary: List users
      responses:
        '200':
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
```

### XIV. Telemetry

Structured logs, metrics, and distributed tracing from day one.

```go
// OpenTelemetry setup
func initTracer() func() {
    exporter, _ := otlptrace.New(context.Background(),
        otlptracegrpc.NewClient(
            otlptracegrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
        ),
    )
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("myapp"),
        )),
    )
    otel.SetTracerProvider(tp)
    return func() { tp.Shutdown(context.Background()) }
}
```

### XV. Security

Security is not an afterthought. Scan dependencies, enforce least privilege, rotate secrets.

```bash
# Dependency vulnerability scanning
govulncheck ./...

# Container security scanning
trivy image myapp:v1.2.3

# Non-root container
```

```dockerfile
RUN adduser -D -u 1000 appuser
USER appuser
```

## Tips

- Factor III (Config) is the most commonly violated — avoid config files, use env vars
- Factor VI (Processes) means no sticky sessions — use external session stores
- Factor IX (Disposability) requires handling SIGTERM — add graceful shutdown to every service
- Factor X (Dev/Prod Parity) means using Docker for local development too
- Factor XI (Logs) means never writing to log files — always stdout/stderr
- Factor XII (Admin) means migrations run as separate processes, not on startup
- The 12 factors are guidelines, not laws — adapt to your context
- Modern cloud-native apps should also consider API-first, telemetry, and security
- Use `docker compose` locally to satisfy Factor X and IV simultaneously
- Test disposability by randomly killing containers during development

## See Also

- `sheets/testing/chaos-engineering.md` — testing Factor IX (Disposability)
- `sheets/auth/oidc.md` — authentication patterns for Factor III (Config) and XV (Security)
- `detail/quality/twelve-factor.md` — process algebra, scaling mathematics, deployment DAGs

## References

- https://12factor.net/ — The Twelve-Factor App (Heroku, Adam Wiggins)
- https://www.cncf.io/ — Cloud Native Computing Foundation
- https://go.dev/doc/ — Go documentation
- https://docs.docker.com/compose/ — Docker Compose
- https://kubernetes.io/docs/ — Kubernetes documentation
