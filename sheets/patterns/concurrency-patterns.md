# Concurrency Patterns (Multi-Language Reference)

A comprehensive guide to concurrent and parallel programming patterns in Go, Rust, and Python — from primitives to advanced composition.

## Go Concurrency Primitives

### Goroutines

```go
// Launch a goroutine
go func() {
    fmt.Println("running concurrently")
}()

// With parameters (avoid closure capture pitfalls)
for i := 0; i < 10; i++ {
    go func(n int) {
        fmt.Printf("goroutine %d\n", n)
    }(i)
}

// Goroutine leak detection — always ensure goroutines can exit
func worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return // clean exit
        default:
            doWork()
        }
    }
}
```

### Channels — Unbuffered

```go
// Unbuffered: sender blocks until receiver is ready (synchronous handoff)
ch := make(chan int)

go func() {
    ch <- 42 // blocks until someone reads
}()

val := <-ch // blocks until someone writes
fmt.Println(val)

// Directional channels for type safety
func producer(out chan<- int) { out <- 1 }
func consumer(in <-chan int)  { v := <-in; _ = v }
```

### Channels — Buffered

```go
// Buffered: sender blocks only when buffer is full
ch := make(chan int, 100)

// Non-blocking send/receive with select
select {
case ch <- value:
    // sent successfully
default:
    // channel full, drop or handle backpressure
}

// Check buffer state
fmt.Printf("len=%d cap=%d\n", len(ch), cap(ch))
```

### Select Statement

```go
// Multiplex across multiple channels
select {
case msg := <-inbox:
    handle(msg)
case result := <-results:
    process(result)
case <-time.After(5 * time.Second):
    fmt.Println("timeout")
case <-ctx.Done():
    fmt.Println("cancelled")
}

// Non-blocking check
select {
case v := <-ch:
    use(v)
default:
    // channel empty, move on
}

// Priority select (Go has no built-in priority — nest selects)
select {
case <-highPriority:
    handleHigh()
default:
    select {
    case <-highPriority:
        handleHigh()
    case <-lowPriority:
        handleLow()
    }
}
```

### sync.Mutex and sync.RWMutex

```go
// Mutex — exclusive access
var mu sync.Mutex
var balance int

func deposit(amount int) {
    mu.Lock()
    defer mu.Unlock()
    balance += amount
}

// RWMutex — multiple readers, single writer
var rwmu sync.RWMutex
var cache map[string]string

func read(key string) string {
    rwmu.RLock()
    defer rwmu.RUnlock()
    return cache[key]
}

func write(key, val string) {
    rwmu.Lock()
    defer rwmu.Unlock()
    cache[key] = val
}
```

### sync.WaitGroup

```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        process(id)
    }(i)
}

wg.Wait() // block until all goroutines finish
```

### sync.Once

```go
var (
    instance *Database
    once     sync.Once
)

func GetDB() *Database {
    once.Do(func() {
        instance = connectDB()
    })
    return instance
}
```

### sync.Map

```go
// Thread-safe map (useful when keys are stable or disjoint goroutine access)
var m sync.Map

m.Store("key", "value")

if v, ok := m.Load("key"); ok {
    fmt.Println(v.(string))
}

// LoadOrStore: returns existing value or stores new one
actual, loaded := m.LoadOrStore("key", "default")

// Range: iterate all entries
m.Range(func(key, value any) bool {
    fmt.Printf("%v: %v\n", key, value)
    return true // continue iteration
})
```

## Go Advanced Patterns

### errgroup — Goroutines with Error Propagation

```go
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(context.Background())

for _, url := range urls {
    url := url // capture loop variable
    g.Go(func() error {
        return fetch(ctx, url)
    })
}

if err := g.Wait(); err != nil {
    log.Fatalf("fetch failed: %v", err)
}
```

### Semaphore Pattern

```go
// Limit concurrency to N goroutines
sem := make(chan struct{}, 10) // max 10 concurrent

for _, task := range tasks {
    sem <- struct{}{} // acquire
    go func(t Task) {
        defer func() { <-sem }() // release
        process(t)
    }(task)
}

// Or use golang.org/x/sync/semaphore
import "golang.org/x/sync/semaphore"

sem := semaphore.NewWeighted(10)

for _, task := range tasks {
    if err := sem.Acquire(ctx, 1); err != nil {
        break
    }
    go func(t Task) {
        defer sem.Release(1)
        process(t)
    }(task)
}
```

### Context Cancellation and Timeout

```go
// Cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    // long-running work
    select {
    case <-ctx.Done():
        log.Println("cancelled:", ctx.Err())
        return
    case result <- doWork():
        handleResult(result)
    }
}()

cancel() // signal all goroutines using this ctx

// Timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := longOperation(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("operation timed out")
}

// Deadline (absolute time)
deadline := time.Now().Add(30 * time.Second)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()
```

### Fan-Out / Fan-In

```go
// Fan-out: distribute work to multiple goroutines
func fanOut(input <-chan int, workers int) []<-chan int {
    channels := make([]<-chan int, workers)
    for i := 0; i < workers; i++ {
        channels[i] = worker(input)
    }
    return channels
}

// Fan-in: merge multiple channels into one
func fanIn(channels ...<-chan int) <-chan int {
    var wg sync.WaitGroup
    merged := make(chan int)

    output := func(ch <-chan int) {
        defer wg.Done()
        for val := range ch {
            merged <- val
        }
    }

    wg.Add(len(channels))
    for _, ch := range channels {
        go output(ch)
    }

    go func() {
        wg.Wait()
        close(merged)
    }()

    return merged
}
```

### Pipeline Pattern

```go
func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            out <- n
        }
    }()
    return out
}

func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * n
        }
    }()
    return out
}

func filter(in <-chan int, pred func(int) bool) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            if pred(n) {
                out <- n
            }
        }
    }()
    return out
}

// Compose: generate -> square -> filter
pipeline := filter(square(generate(1, 2, 3, 4, 5)), func(n int) bool {
    return n > 10
})
for v := range pipeline {
    fmt.Println(v) // 16, 25
}
```

### Worker Pool

```go
func workerPool(ctx context.Context, jobs <-chan Job, results chan<- Result, numWorkers int) {
    var wg sync.WaitGroup
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for {
                select {
                case job, ok := <-jobs:
                    if !ok {
                        return
                    }
                    results <- process(job)
                case <-ctx.Done():
                    return
                }
            }
        }(i)
    }

    go func() {
        wg.Wait()
        close(results)
    }()
}
```

### Or-Done Channel

```go
// Read from a channel respecting cancellation
func orDone(ctx context.Context, ch <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for {
            select {
            case <-ctx.Done():
                return
            case v, ok := <-ch:
                if !ok {
                    return
                }
                select {
                case out <- v:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    return out
}
```

### Tee Channel

```go
// Split one channel into two identical streams
func tee(ctx context.Context, in <-chan int) (<-chan int, <-chan int) {
    out1, out2 := make(chan int), make(chan int)
    go func() {
        defer close(out1)
        defer close(out2)
        for val := range orDone(ctx, in) {
            o1, o2 := out1, out2
            for i := 0; i < 2; i++ {
                select {
                case o1 <- val:
                    o1 = nil
                case o2 <- val:
                    o2 = nil
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    return out1, out2
}
```

### Bridge Channel

```go
// Flatten a channel of channels into a single channel
func bridge(ctx context.Context, chanStream <-chan <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for {
            var stream <-chan int
            select {
            case maybe, ok := <-chanStream:
                if !ok {
                    return
                }
                stream = maybe
            case <-ctx.Done():
                return
            }
            for val := range orDone(ctx, stream) {
                select {
                case out <- val:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    return out
}
```

## Rust Concurrency

### Tokio Spawning and Async/Await

```rust
use tokio;

#[tokio::main]
async fn main() {
    // Spawn a task (like a goroutine)
    let handle = tokio::spawn(async {
        expensive_computation().await
    });

    // Await the result
    let result = handle.await.unwrap();

    // Spawn multiple tasks
    let mut handles = vec![];
    for i in 0..10 {
        handles.push(tokio::spawn(async move {
            process(i).await
        }));
    }

    // Join all
    for handle in handles {
        handle.await.unwrap();
    }

    // select! macro — race multiple futures
    tokio::select! {
        val = future_a() => println!("a finished: {}", val),
        val = future_b() => println!("b finished: {}", val),
        _ = tokio::time::sleep(Duration::from_secs(5)) => {
            println!("timeout");
        }
    }
}
```

### Arc and Mutex

```rust
use std::sync::{Arc, Mutex, RwLock};

// Shared mutable state across threads/tasks
let counter = Arc::new(Mutex::new(0));

let mut handles = vec![];
for _ in 0..10 {
    let counter = Arc::clone(&counter);
    handles.push(tokio::spawn(async move {
        let mut num = counter.lock().unwrap();
        *num += 1;
    }));
}

for handle in handles {
    handle.await.unwrap();
}

// RwLock for read-heavy workloads
let data = Arc::new(RwLock::new(HashMap::new()));

// Tokio-aware locks (don't hold across .await with std::sync)
use tokio::sync::Mutex as TokioMutex;
let data = Arc::new(TokioMutex::new(vec![]));

let mut lock = data.lock().await; // non-blocking acquire
lock.push(42);
```

### Rust Channels

```rust
// mpsc — multiple producer, single consumer
use tokio::sync::mpsc;

let (tx, mut rx) = mpsc::channel(100); // buffered

tokio::spawn(async move {
    tx.send("hello").await.unwrap();
});

while let Some(msg) = rx.recv().await {
    println!("received: {}", msg);
}

// broadcast — multiple producers, multiple consumers
use tokio::sync::broadcast;

let (tx, _) = broadcast::channel(16);
let mut rx1 = tx.subscribe();
let mut rx2 = tx.subscribe();

tx.send("event").unwrap();
// both rx1 and rx2 receive "event"

// oneshot — single value, single send
use tokio::sync::oneshot;

let (tx, rx) = oneshot::channel();
tx.send(42).unwrap();
let val = rx.await.unwrap();
```

## Python Concurrency

### asyncio

```python
import asyncio

async def fetch_data(url: str) -> str:
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as resp:
            return await resp.text()

async def main():
    # Run concurrently
    results = await asyncio.gather(
        fetch_data("https://api.example.com/a"),
        fetch_data("https://api.example.com/b"),
        fetch_data("https://api.example.com/c"),
    )

    # With timeout
    try:
        result = await asyncio.wait_for(
            slow_operation(), timeout=5.0
        )
    except asyncio.TimeoutError:
        print("timed out")

    # Task groups (Python 3.11+)
    async with asyncio.TaskGroup() as tg:
        task1 = tg.create_task(fetch_data(url1))
        task2 = tg.create_task(fetch_data(url2))
    # Both complete here; exceptions propagate

asyncio.run(main())
```

### Threading and Multiprocessing

```python
from threading import Thread, Lock, Event
from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor

# Threading (I/O-bound) — GIL limits CPU parallelism
lock = Lock()
shared = []

def worker(item):
    with lock:
        shared.append(process(item))

threads = [Thread(target=worker, args=(i,)) for i in range(10)]
for t in threads:
    t.start()
for t in threads:
    t.join()

# ThreadPoolExecutor
with ThreadPoolExecutor(max_workers=10) as executor:
    futures = [executor.submit(fetch, url) for url in urls]
    results = [f.result() for f in futures]

# ProcessPoolExecutor (CPU-bound) — bypasses GIL
with ProcessPoolExecutor(max_workers=4) as executor:
    results = list(executor.map(cpu_heavy_task, data_chunks))

# concurrent.futures — as_completed for streaming results
from concurrent.futures import as_completed

with ThreadPoolExecutor(max_workers=5) as executor:
    future_to_url = {executor.submit(fetch, u): u for u in urls}
    for future in as_completed(future_to_url):
        url = future_to_url[future]
        try:
            data = future.result()
        except Exception as e:
            print(f"{url} generated exception: {e}")
```

## Tips

- Always use `defer mu.Unlock()` immediately after `mu.Lock()` to prevent deadlocks on panics
- Prefer channels for communication between goroutines, mutexes for protecting shared state
- Never copy a `sync.Mutex` or `sync.WaitGroup` after first use (pass by pointer)
- Use `-race` flag in Go tests to detect data races: `go test -race ./...`
- Buffered channels decouple producers and consumers but can mask backpressure problems
- In Rust, prefer `tokio::sync::Mutex` over `std::sync::Mutex` when holding locks across `.await`
- Python's GIL means threads are only useful for I/O-bound work; use multiprocessing for CPU-bound
- Close channels from the sender side only; never close from the receiver
- Context cancellation propagates down the call tree, not up

## See Also

- `detail/patterns/concurrency-patterns.md` — CSP theory, happens-before, Amdahl's law
- `sheets/patterns/distributed-systems.md` — distributed concurrency and consensus
- `sheets/patterns/microservices-patterns.md` — circuit breakers and bulkheads

## References

- Go Concurrency Patterns (Rob Pike, 2012): https://go.dev/talks/2012/concurrency.slide
- Advanced Go Concurrency Patterns (Sameer Ajmani, 2013): https://go.dev/talks/2013/advconc.slide
- Tokio Tutorial: https://tokio.rs/tokio/tutorial
- Python asyncio Documentation: https://docs.python.org/3/library/asyncio.html
- "Concurrency in Go" by Katherine Cox-Buday (O'Reilly, 2017)
