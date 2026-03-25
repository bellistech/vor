# Go (Programming Language)

Statically typed, compiled language with built-in concurrency, garbage collection, and a rich standard library.

## Types

### Basic types

```bash
# var s string = "hello"
# var n int = 42
# var f float64 = 3.14
# var b bool = true
# var by byte = 'A'          // alias for uint8
# var r rune = 'Z'           // alias for int32 (Unicode code point)
# x := 10                    // short declaration (infers type)
# const Pi = 3.14159
```

### Type conversions

```bash
# i := 42
# f := float64(i)
# s := strconv.Itoa(i)                    // int -> string
# n, err := strconv.Atoi("42")            // string -> int
# bs := []byte("hello")                   // string -> byte slice
```

## Slices

```bash
# s := []int{1, 2, 3, 4, 5}
# s = append(s, 6, 7)
# s2 := make([]int, 0, 100)               // len=0, cap=100
# copy(dst, src)
# s[1:3]                                   // elements at index 1,2
# s[:3]                                    // first 3
# s[2:]                                    // from index 2 to end
# len(s), cap(s)
# slices.Contains(s, 3)                   // Go 1.21+
# slices.Sort(s)
```

## Maps

```bash
# m := map[string]int{"alice": 1, "bob": 2}
# m["carol"] = 3
# val, ok := m["alice"]                    // comma-ok idiom
# delete(m, "bob")
# for k, v := range m { ... }
# len(m)
# m2 := make(map[string]int, 100)         // pre-allocate
```

## Structs

```bash
# type User struct {
#     Name  string `json:"name"`
#     Email string `json:"email"`
#     Age   int    `json:"age,omitempty"`
# }
# u := User{Name: "Alice", Email: "alice@example.com"}
# u.Name
# p := &u                                  // pointer to struct
# p.Name                                   // auto-dereference
```

### Methods

```bash
# func (u User) String() string {
#     return fmt.Sprintf("%s <%s>", u.Name, u.Email)
# }
# func (u *User) SetEmail(email string) {
#     u.Email = email                       // pointer receiver mutates
# }
```

## Interfaces

```bash
# type Writer interface {
#     Write(p []byte) (n int, err error)
# }
# // Interfaces are satisfied implicitly -- no "implements" keyword.
# // The empty interface (any / interface{}) accepts any type.
# func process(v any) {
#     switch t := v.(type) {
#     case string: fmt.Println("string:", t)
#     case int:    fmt.Println("int:", t)
#     }
# }
```

## Goroutines & Channels

### Launch goroutines

```bash
# go doWork()
# go func() {
#     fmt.Println("anonymous goroutine")
# }()
```

### Channels

```bash
# ch := make(chan int)        // unbuffered
# ch := make(chan int, 100)   // buffered
# ch <- 42                    // send
# val := <-ch                 // receive
# close(ch)
# for v := range ch { ... }  // receive until closed
```

### Select

```bash
# select {
# case msg := <-ch1:
#     handle(msg)
# case ch2 <- result:
#     // sent
# case <-time.After(5 * time.Second):
#     // timeout
# case <-ctx.Done():
#     return ctx.Err()
# }
```

### WaitGroup

```bash
# var wg sync.WaitGroup
# for i := 0; i < 10; i++ {
#     wg.Add(1)
#     go func(id int) {
#         defer wg.Done()
#         process(id)
#     }(i)
# }
# wg.Wait()
```

## Error Handling

```bash
# f, err := os.Open("file.txt")
# if err != nil {
#     return fmt.Errorf("open file: %w", err)   // wrap error
# }
# defer f.Close()
#
# // errors.Is checks the chain
# if errors.Is(err, os.ErrNotExist) { ... }
# // errors.As unwraps to a specific type
# var pathErr *os.PathError
# if errors.As(err, &pathErr) { ... }
#
# // Custom error type
# type NotFoundError struct{ ID string }
# func (e *NotFoundError) Error() string {
#     return fmt.Sprintf("not found: %s", e.ID)
# }
```

## Testing

```bash
go test ./...
go test ./... -race -count=1           # with race detector, no caching
go test -v -run TestSpecific ./pkg/    # run one test
go test -bench=. ./...                 # benchmarks
go test -cover ./...                   # coverage summary
go test -coverprofile=cover.out ./... && go tool cover -html=cover.out
```

### Test file (foo_test.go)

```bash
# func TestAdd(t *testing.T) {
#     got := Add(2, 3)
#     if got != 5 {
#         t.Errorf("Add(2,3) = %d, want 5", got)
#     }
# }
# func BenchmarkAdd(b *testing.B) {
#     for i := 0; i < b.N; i++ {
#         Add(2, 3)
#     }
# }
```

## Modules

```bash
go mod init github.com/user/project
go mod tidy                            # add missing, remove unused deps
go get github.com/pkg/errors@v0.9.1    # add dependency at version
go get -u ./...                        # update all deps
go mod vendor                          # vendor dependencies
go mod graph                           # show dependency graph
go list -m all                         # list all modules
```

## Build & Run

```bash
go run main.go
go build -o myapp .
go build -ldflags="-s -w" -o myapp .   # strip debug info, smaller binary
go install ./cmd/myapp                 # install to $GOPATH/bin
GOOS=linux GOARCH=amd64 go build -o myapp-linux .   # cross-compile
CGO_ENABLED=0 go build -o myapp .     # static binary (no cgo)
```

## Common Patterns

### Context

```bash
# ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
# defer cancel()
# req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
```

### HTTP server

```bash
# mux := http.NewServeMux()
# mux.HandleFunc("GET /api/users", listUsers)
# mux.HandleFunc("POST /api/users", createUser)  // Go 1.22+ method patterns
# log.Fatal(http.ListenAndServe(":8080", mux))
```

### JSON

```bash
# data, err := json.Marshal(user)
# err = json.Unmarshal(data, &user)
# err = json.NewDecoder(r.Body).Decode(&user)
# err = json.NewEncoder(w).Encode(user)
```

### io

```bash
# data, err := io.ReadAll(r)
# _, err = io.Copy(dst, src)
# r := io.LimitReader(src, 1<<20)         // limit to 1MB
```

## Tips

- Always check errors. `_` discarding errors leads to silent failures.
- Use `defer` for cleanup (close files, unlock mutexes). Defers run LIFO.
- `go test -race` catches data races. Run it in CI.
- Wrap errors with `fmt.Errorf("context: %w", err)` to build error chains.
- Prefer `context.Context` as the first parameter of functions that do I/O or may be cancelled.
- Slices and maps are reference types. Passing them to functions shares the underlying data.
- `sync.Once` is the safest way to do one-time initialization in concurrent code.
- `go vet` catches common mistakes (printf format errors, unreachable code). Run it alongside tests.
