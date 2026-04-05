# Go Testing (Standard Library)

Comprehensive reference for Go's built-in testing framework — table-driven tests, subtests, benchmarks, coverage, race detection, and all the flags you need.

## Test File Conventions

### File Naming

```
mypackage/
  handler.go
  handler_test.go       # same package — white-box tests
  handler_ext_test.go   # package mypackage_test — black-box tests
```

Test files must end in `_test.go`. They are excluded from normal builds automatically.

### Function Signatures

```go
func TestXxx(t *testing.T)       // unit test (Xxx starts with uppercase)
func BenchmarkXxx(b *testing.B)  // benchmark
func FuzzXxx(f *testing.F)       // fuzz test (Go 1.18+)
func ExampleXxx()                // runnable doc example
func TestMain(m *testing.M)      // global setup/teardown
```

## Table-Driven Tests

### Basic Pattern

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 2, 3, 5},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
        {"mixed", -1, 5, 4},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.expected)
            }
        })
    }
}
```

### With Error Cases

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Config
        wantErr bool
    }{
        {
            name:  "valid json",
            input: `{"port": 8080}`,
            want:  &Config{Port: 8080},
        },
        {
            name:    "invalid json",
            input:   `{broken`,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Parse() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Subtests and t.Run

### Nested Subtests

```go
func TestHTTPHandler(t *testing.T) {
    t.Run("GET", func(t *testing.T) {
        t.Run("returns 200 for valid ID", func(t *testing.T) {
            // ...
        })
        t.Run("returns 404 for missing ID", func(t *testing.T) {
            // ...
        })
    })
    t.Run("POST", func(t *testing.T) {
        t.Run("creates resource", func(t *testing.T) {
            // ...
        })
    })
}
```

Run specific subtests:

```bash
go test -run "TestHTTPHandler/GET/returns_200"
# spaces in names become underscores in -run pattern
```

## Parallel Tests

### t.Parallel

```go
func TestParallel(t *testing.T) {
    tests := []struct {
        name  string
        input int
    }{
        {"case1", 1},
        {"case2", 2},
        {"case3", 3},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // mark this subtest as safe for parallel execution
            // CRITICAL: tt is captured by the loop variable
            // Go 1.22+ fixes this; before 1.22 you need:
            //   tt := tt
            result := SlowCompute(tt.input)
            if result != tt.input*2 {
                t.Errorf("got %d, want %d", result, tt.input*2)
            }
        })
    }
}
```

Control parallelism:

```bash
go test -parallel 4    # max 4 parallel subtests (default GOMAXPROCS)
```

## Test Helpers

### t.Helper

```go
func assertEqual(t *testing.T, got, want interface{}) {
    t.Helper() // marks this function as a helper
    // error reports will show caller's line, not this line
    if !reflect.DeepEqual(got, want) {
        t.Errorf("got %v, want %v", got, want)
    }
}

func mustParseURL(t *testing.T, raw string) *url.URL {
    t.Helper()
    u, err := url.Parse(raw)
    if err != nil {
        t.Fatalf("failed to parse URL %q: %v", raw, err)
    }
    return u
}
```

### t.Cleanup

```go
func TestWithTempDB(t *testing.T) {
    db := setupTestDB(t)
    t.Cleanup(func() {
        db.Close()
        os.Remove(db.Path())
    })
    // test code — cleanup runs after test completes
    // multiple Cleanup calls run in LIFO order
}
```

### t.TempDir

```go
func TestFileWrite(t *testing.T) {
    dir := t.TempDir() // auto-cleaned up after test
    path := filepath.Join(dir, "output.txt")
    err := WriteFile(path, []byte("hello"))
    if err != nil {
        t.Fatal(err)
    }
}
```

## TestMain

### Global Setup and Teardown

```go
func TestMain(m *testing.M) {
    // setup: runs before ALL tests in this package
    pool, err := dockertest.NewPool("")
    if err != nil {
        log.Fatalf("could not connect to docker: %s", err)
    }

    resource, err := pool.Run("postgres", "15", []string{
        "POSTGRES_PASSWORD=test",
        "POSTGRES_DB=testdb",
    })
    if err != nil {
        log.Fatalf("could not start resource: %s", err)
    }

    // run tests
    code := m.Run()

    // teardown: runs after ALL tests
    if err := pool.Purge(resource); err != nil {
        log.Fatalf("could not purge resource: %s", err)
    }

    os.Exit(code)
}
```

## Golden Files

### Pattern with testdata/

```go
var update = flag.Bool("update", false, "update golden files")

func TestRender(t *testing.T) {
    got := Render(input)

    golden := filepath.Join("testdata", t.Name()+".golden")

    if *update {
        os.MkdirAll("testdata", 0o755)
        os.WriteFile(golden, got, 0o644)
        return
    }

    want, err := os.ReadFile(golden)
    if err != nil {
        t.Fatalf("failed to read golden file: %v", err)
    }

    if !bytes.Equal(got, want) {
        t.Errorf("output mismatch:\n%s", diff(want, got))
    }
}
```

Update goldens:

```bash
go test -run TestRender -update
```

The `testdata/` directory is ignored by `go build` but included in test binaries.

## httptest

### Testing HTTP Handlers

```go
func TestHealthHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()

    HealthHandler(w, req)

    resp := w.Result()
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
    }

    body, _ := io.ReadAll(resp.Body)
    if string(body) != `{"status":"ok"}` {
        t.Errorf("body = %s", body)
    }
}
```

### Test Server

```go
func TestAPIClient(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/v1/users" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintln(w, `[{"id":1,"name":"Alice"}]`)
    }))
    defer srv.Close()

    client := NewAPIClient(srv.URL)
    users, err := client.ListUsers()
    if err != nil {
        t.Fatal(err)
    }
    if len(users) != 1 {
        t.Errorf("got %d users, want 1", len(users))
    }
}
```

### TLS Test Server

```go
func TestTLSClient(t *testing.T) {
    srv := httptest.NewTLSServer(handler)
    defer srv.Close()

    client := srv.Client() // pre-configured to trust the test cert
    resp, err := client.Get(srv.URL + "/secure")
    // ...
}
```

## Benchmarks

### Basic Benchmark

```go
func BenchmarkFibonacci(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Fibonacci(20)
    }
}
```

### With Setup

```go
func BenchmarkParse(b *testing.B) {
    data, err := os.ReadFile("testdata/large.json")
    if err != nil {
        b.Fatal(err)
    }
    b.ResetTimer() // exclude setup from timing

    for i := 0; i < b.N; i++ {
        Parse(data)
    }
}
```

### Memory Reporting

```go
func BenchmarkAllocations(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        result := BuildLargeStruct()
        _ = result
    }
}
```

### Parallel Benchmark

```go
func BenchmarkConcurrentMap(b *testing.B) {
    m := NewConcurrentMap()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            m.Set(fmt.Sprintf("key%d", i), i)
            i++
        }
    })
}
```

### Sub-Benchmarks

```go
func BenchmarkSort(b *testing.B) {
    for _, size := range []int{100, 1000, 10000} {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := generateData(size)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                sort.Ints(append([]int{}, data...))
            }
        })
    }
}
```

Run benchmarks:

```bash
go test -bench=.                          # all benchmarks
go test -bench=BenchmarkSort -benchmem    # with memory stats
go test -bench=. -benchtime=5s            # longer sampling
go test -bench=. -count=10 | tee old.txt  # for benchstat
```

## Build Tags for Integration Tests

### Using Build Constraints

```go
//go:build integration

package mypackage

func TestDatabaseIntegration(t *testing.T) {
    // only runs with: go test -tags=integration
}
```

### Using testing.Short

```go
func TestSlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping slow test in short mode")
    }
    // long-running integration test
}
```

```bash
go test -short ./...    # skip slow tests
go test ./...           # run everything
```

## Test Flags Reference

```bash
# Execution control
go test -v                    # verbose output
go test -run TestFoo          # regex filter for test names
go test -run TestFoo/subtest  # run specific subtest
go test -count=1              # disable test caching
go test -count=5              # run each test 5 times
go test -timeout 30s          # per-test timeout (default 10m)
go test -short                # set testing.Short() = true
go test -shuffle=on           # randomize test order (Go 1.17+)
go test -shuffle=12345        # reproducible random order with seed
go test -failfast             # stop on first failure
go test -parallel 8           # max parallel subtests

# Race detection
go test -race                 # enable race detector
go test -race -count=5        # race + repeated runs (recommended)

# Coverage
go test -cover                # show coverage percentage
go test -coverprofile=c.out   # write coverage data
go test -covermode=atomic     # mode: set, count, or atomic
go test -coverpkg=./...       # include all packages in coverage
go tool cover -html=c.out     # open HTML coverage report
go tool cover -func=c.out     # per-function coverage

# Benchmarks
go test -bench=.              # run all benchmarks
go test -bench=. -benchmem    # include memory allocation stats
go test -bench=. -benchtime=3s  # benchmark duration
go test -bench=. -cpuprofile=cpu.out  # CPU profile

# Build
go test -tags=integration     # build tag for conditional compilation
go test -json                 # JSON output (for CI parsing)
go test -list ".*"            # list tests without running them
```

## Testify Assertions

### Common Assertions

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWithTestify(t *testing.T) {
    // assert — logs failure but continues
    assert.Equal(t, expected, actual)
    assert.NotNil(t, obj)
    assert.True(t, condition)
    assert.Contains(t, "hello world", "hello")
    assert.Len(t, slice, 3)
    assert.ErrorIs(t, err, ErrNotFound)
    assert.ErrorContains(t, err, "not found")
    assert.WithinDuration(t, expected, actual, time.Second)
    assert.JSONEq(t, `{"a":1}`, `{"a": 1}`)

    // require — fails immediately (like t.Fatal)
    require.NoError(t, err) // if err != nil, test stops here
    require.NotNil(t, result)
}
```

### Suite Pattern

```go
import "github.com/stretchr/testify/suite"

type UserServiceSuite struct {
    suite.Suite
    db      *sql.DB
    service *UserService
}

func (s *UserServiceSuite) SetupSuite() {
    s.db = connectTestDB()
}

func (s *UserServiceSuite) TearDownSuite() {
    s.db.Close()
}

func (s *UserServiceSuite) SetupTest() {
    s.db.Exec("DELETE FROM users")
    s.service = NewUserService(s.db)
}

func (s *UserServiceSuite) TestCreateUser() {
    user, err := s.service.Create("alice", "alice@test.com")
    s.Require().NoError(err)
    s.Equal("alice", user.Name)
}

func TestUserService(t *testing.T) {
    suite.Run(t, new(UserServiceSuite))
}
```

## Tips

- Always use `t.Helper()` in test utility functions so failure messages point to the caller
- Use `t.Parallel()` liberally for independent tests to speed up the suite
- Prefer `require` over `assert` when a failure makes subsequent checks meaningless
- Run `go test -race -count=3 ./...` in CI to catch races reliably
- Use `testdata/` for fixtures — it is invisible to `go build` but available to tests
- Use `-shuffle=on` to detect order-dependent tests
- Keep golden files under version control; update with `-update` flag
- Use `t.Cleanup()` instead of `defer` in test helpers — cleanup runs even if the helper returns early
- Avoid `init()` in test files; use `TestMain` for global setup
- For flaky tests, `go test -count=100 -run TestFlaky` helps reproduce

## See Also

- `sheets/testing/benchmarking.md` — deep dive on Go benchmarks and benchstat
- `sheets/testing/mocking.md` — gomock, testify/mock, mockery
- `sheets/testing/coverage.md` — coverage profiles, modes, and tooling
- `sheets/testing/integration-testing.md` — testcontainers, docker-compose patterns

## References

- https://pkg.go.dev/testing — official testing package docs
- https://go.dev/doc/tutorial/add-a-test — Go tutorial
- https://go.dev/blog/subtests — subtests and sub-benchmarks
- https://github.com/stretchr/testify — testify assertion library
- https://go.dev/doc/articles/race_detector — race detector documentation
