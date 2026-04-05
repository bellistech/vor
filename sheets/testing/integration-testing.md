# Integration Testing (Go + Docker Patterns)

Complete reference for integration testing — testcontainers-go, database setup/teardown, API testing, docker-compose for test infra, test isolation, fixtures, and environment-gated tests.

## Test Isolation Strategies

### The Spectrum

```
Unit Tests          Integration Tests          E2E Tests
  (fast, isolated)    (real deps, slower)        (full stack, slowest)
  Mock everything     Real DB, real cache        Real everything
  ms per test         s per test                 10s+ per test
  1000s of them       100s of them               10s of them
```

### Isolation Approaches

| Strategy | Speed | Isolation | Complexity |
|----------|-------|-----------|------------|
| Separate DB per test | Slow | Perfect | High |
| Transaction rollback | Fast | Good | Medium |
| Truncate tables | Medium | Good | Low |
| Schema per test | Slow | Perfect | High |
| Shared DB, unique data | Fast | Fragile | Low |

## Environment-Gated Tests

### Build Tags

```go
//go:build integration

package mypackage_test

import "testing"

func TestDatabaseQuery(t *testing.T) {
    // only runs with: go test -tags=integration
}
```

```bash
# Run unit tests only
go test ./...

# Run integration tests
go test -tags=integration ./...

# Run both
go test -tags=integration ./...
```

### testing.Short

```go
func TestSlowIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // slow test with real database
}
```

```bash
go test -short ./...    # skip integration tests
go test ./...           # run all tests
```

## testcontainers-go

### Basic Container

```go
package mypackage_test

import (
    "context"
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestWithPostgres(t *testing.T) {
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image:        "postgres:16-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_USER":     "test",
            "POSTGRES_PASSWORD": "test",
            "POSTGRES_DB":       "testdb",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections").
            WithOccurrence(2).
            WithStartupTimeout(30 * time.Second),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start container: %v", err)
    }
    t.Cleanup(func() {
        if err := container.Terminate(ctx); err != nil {
            t.Logf("failed to terminate container: %v", err)
        }
    })

    host, err := container.Host(ctx)
    if err != nil {
        t.Fatal(err)
    }
    port, err := container.MappedPort(ctx, "5432")
    if err != nil {
        t.Fatal(err)
    }

    dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { db.Close() })

    // Run your actual tests against `db`
    err = db.Ping()
    if err != nil {
        t.Fatalf("failed to ping database: %v", err)
    }
}
```

### Redis Container

```go
func setupRedis(t *testing.T) *redis.Client {
    t.Helper()
    ctx := context.Background()

    req := testcontainers.ContainerRequest{
        Image:        "redis:7-alpine",
        ExposedPorts: []string{"6379/tcp"},
        WaitingFor:   wait.ForLog("Ready to accept connections"),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start redis: %v", err)
    }
    t.Cleanup(func() { container.Terminate(ctx) })

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "6379")

    client := redis.NewClient(&redis.Options{
        Addr: fmt.Sprintf("%s:%s", host, port.Port()),
    })
    t.Cleanup(func() { client.Close() })

    return client
}
```

### Module-Based Containers (Preferred)

```go
import (
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestWithPostgresModule(t *testing.T) {
    ctx := context.Background()

    pgContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready").WithOccurrence(2),
        ),
    )
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { pgContainer.Terminate(ctx) })

    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatal(err)
    }

    // use connStr to connect
}
```

## TestMain for Global Setup

### Shared Container Across Tests

```go
var testDB *sql.DB

func TestMain(m *testing.M) {
    ctx := context.Background()

    container, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    if err != nil {
        log.Fatalf("failed to start postgres: %v", err)
    }

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        log.Fatalf("failed to get connection string: %v", err)
    }

    testDB, err = sql.Open("pgx", connStr)
    if err != nil {
        log.Fatalf("failed to open database: %v", err)
    }

    // Run migrations
    if err := runMigrations(testDB); err != nil {
        log.Fatalf("failed to run migrations: %v", err)
    }

    code := m.Run()

    testDB.Close()
    container.Terminate(ctx)
    os.Exit(code)
}
```

## Database Test Patterns

### Transaction Rollback

```go
func TestUserRepository(t *testing.T) {
    // Start a transaction
    tx, err := testDB.Begin()
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() {
        tx.Rollback() // always rollback — test isolation
    })

    repo := NewUserRepository(tx)

    user, err := repo.Create(context.Background(), &User{
        Name:  "Alice",
        Email: "alice@test.com",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, user.ID)

    // Verify
    found, err := repo.GetByID(context.Background(), user.ID)
    require.NoError(t, err)
    assert.Equal(t, "Alice", found.Name)
    // tx.Rollback() runs via t.Cleanup — no data persists
}
```

### Table Truncation

```go
func truncateTables(t *testing.T, db *sql.DB, tables ...string) {
    t.Helper()
    for _, table := range tables {
        _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
        if err != nil {
            t.Fatalf("failed to truncate %s: %v", table, err)
        }
    }
}

func TestWithCleanTables(t *testing.T) {
    truncateTables(t, testDB, "users", "orders")
    t.Cleanup(func() {
        truncateTables(t, testDB, "users", "orders")
    })

    // test code
}
```

### Fixtures

```go
type TestFixtures struct {
    Admin  *User
    Member *User
    Org    *Organization
}

func setupFixtures(t *testing.T, db *sql.DB) *TestFixtures {
    t.Helper()
    repo := NewRepository(db)
    ctx := context.Background()

    org, err := repo.CreateOrg(ctx, &Organization{Name: "TestOrg"})
    require.NoError(t, err)

    admin, err := repo.CreateUser(ctx, &User{Name: "Admin", Role: "admin", OrgID: org.ID})
    require.NoError(t, err)

    member, err := repo.CreateUser(ctx, &User{Name: "Member", Role: "member", OrgID: org.ID})
    require.NoError(t, err)

    return &TestFixtures{Admin: admin, Member: member, Org: org}
}
```

## API Integration Tests

### Full Server Test

```go
func TestAPIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup real dependencies
    db := setupTestDB(t)
    cache := setupTestRedis(t)

    // Create real server
    app := NewApp(db, cache)
    srv := httptest.NewServer(app.Handler())
    t.Cleanup(srv.Close)

    client := &http.Client{Timeout: 5 * time.Second}

    t.Run("create and get user", func(t *testing.T) {
        // Create
        body := strings.NewReader(`{"name":"Alice","email":"alice@test.com"}`)
        resp, err := client.Post(srv.URL+"/api/v1/users", "application/json", body)
        require.NoError(t, err)
        require.Equal(t, http.StatusCreated, resp.StatusCode)

        var created User
        json.NewDecoder(resp.Body).Decode(&created)
        resp.Body.Close()

        // Get
        resp, err = client.Get(srv.URL + "/api/v1/users/" + created.ID)
        require.NoError(t, err)
        require.Equal(t, http.StatusOK, resp.StatusCode)

        var fetched User
        json.NewDecoder(resp.Body).Decode(&fetched)
        resp.Body.Close()

        assert.Equal(t, "Alice", fetched.Name)
    })
}
```

### HTTP Client Integration

```go
func TestExternalAPIClient(t *testing.T) {
    // Use httptest.NewServer to simulate external API
    externalAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/api/users/123":
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(map[string]string{
                "id":   "123",
                "name": "External User",
            })
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
    t.Cleanup(externalAPI.Close)

    client := NewExternalClient(externalAPI.URL)
    user, err := client.GetUser(context.Background(), "123")
    require.NoError(t, err)
    assert.Equal(t, "External User", user.Name)
}
```

## docker-compose for Test Infra

### docker-compose.test.yml

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: testdb
    ports:
      - "5433:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U test -d testdb"]
      interval: 2s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 2s
      timeout: 5s
      retries: 10

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
```

### Makefile Integration

```makefile
.PHONY: test-integration
test-integration:
	docker compose -f docker-compose.test.yml up -d --wait
	go test -tags=integration -v -count=1 ./...
	docker compose -f docker-compose.test.yml down -v

.PHONY: test-integration-clean
test-integration-clean:
	docker compose -f docker-compose.test.yml down -v --remove-orphans
```

## Parallel Integration Tests

### Safe Parallelism

```go
func TestParallelDB(t *testing.T) {
    tests := []struct {
        name  string
        email string
    }{
        {"create alice", "alice@test.com"},
        {"create bob", "bob@test.com"},
        {"create charlie", "charlie@test.com"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // safe because each test uses unique data

            repo := NewUserRepository(testDB)
            user, err := repo.Create(context.Background(), &User{
                Name:  tt.name,
                Email: tt.email, // unique per test
            })
            require.NoError(t, err)

            found, err := repo.GetByEmail(context.Background(), tt.email)
            require.NoError(t, err)
            assert.Equal(t, user.ID, found.ID)
        })
    }
}
```

### Per-Test Schema (Maximum Isolation)

```go
func setupIsolatedSchema(t *testing.T, db *sql.DB) *sql.DB {
    t.Helper()
    schema := fmt.Sprintf("test_%s_%d", t.Name(), time.Now().UnixNano())
    schema = strings.ReplaceAll(schema, "/", "_")
    schema = strings.ReplaceAll(schema, " ", "_")

    _, err := db.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema))
    require.NoError(t, err)

    _, err = db.Exec(fmt.Sprintf("SET search_path TO %s", schema))
    require.NoError(t, err)

    t.Cleanup(func() {
        db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
    })

    return db
}
```

## Tips

- Use `testcontainers-go` modules when available — they handle wait strategies and connection strings
- TestMain containers are shared across all tests in a package — faster but requires careful isolation
- Transaction rollback is the fastest isolation strategy for database tests
- Always use `t.Cleanup()` instead of `defer` in helper functions — cleanup runs even on `t.Fatal`
- Tag integration tests with `//go:build integration` to keep `go test ./...` fast
- Use `-parallel 1` for integration tests that share database state
- docker-compose is simpler than testcontainers for multi-service setups
- Set realistic timeouts — integration tests should not hang forever
- Use unique data per test (UUIDs, timestamps) for parallel safety
- Run integration tests in CI with `docker compose up --wait` before `go test`

## See Also

- `sheets/testing/go-testing.md` — Go testing fundamentals and TestMain
- `sheets/testing/mocking.md` — when to mock vs use real dependencies
- `sheets/testing/chaos-engineering.md` — fault injection for integration tests

## References

- https://golang.testcontainers.org/ — testcontainers-go documentation
- https://pkg.go.dev/testing — Go testing package
- https://docs.docker.com/compose/ — Docker Compose documentation
- https://martinfowler.com/bliki/IntegrationTest.html — Fowler on integration tests
