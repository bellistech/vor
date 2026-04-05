# Benchmarking (Go, Rust, CLI)

Complete reference for performance measurement — Go testing.B, benchstat, criterion (Rust), hyperfine (CLI), pprof integration, and the art of avoiding measurement traps.

## Go Benchmarks (testing.B)

### Basic Benchmark

```go
func BenchmarkConcat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = "hello" + " " + "world"
    }
}
```

The framework adjusts `b.N` automatically until the measurement is stable (minimum 1 second by default).

### Timer Control

```go
func BenchmarkWithSetup(b *testing.B) {
    // expensive setup
    data := loadTestData()
    b.ResetTimer() // zero the clock after setup

    for i := 0; i < b.N; i++ {
        b.StopTimer()  // pause for per-iteration setup
        input := prepareInput(data)
        b.StartTimer() // resume measurement

        Process(input)
    }
}
```

**Warning**: `b.StopTimer()`/`b.StartTimer()` in the hot loop adds overhead. Prefer `b.ResetTimer()` with bulk setup when possible.

### Memory Reporting

```go
func BenchmarkAllocations(b *testing.B) {
    b.ReportAllocs() // report allocs/op and bytes/op
    for i := 0; i < b.N; i++ {
        buf := make([]byte, 1024)
        _ = buf
    }
}
```

Output:

```
BenchmarkAllocations-8    5000000    234 ns/op    1024 B/op    1 allocs/op
```

### Sub-Benchmarks

```go
func BenchmarkEncode(b *testing.B) {
    sizes := []int{1, 10, 100, 1000, 10000}
    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := generatePayload(size)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                Encode(data)
            }
        })
    }
}
```

```bash
go test -bench=BenchmarkEncode/size=1000
```

### Parallel Benchmarks

```go
func BenchmarkConcurrentCache(b *testing.B) {
    cache := NewCache()
    // prefill
    for i := 0; i < 1000; i++ {
        cache.Set(fmt.Sprintf("key%d", i), i)
    }
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        rng := rand.New(rand.NewSource(time.Now().UnixNano()))
        for pb.Next() {
            key := fmt.Sprintf("key%d", rng.Intn(1000))
            cache.Get(key)
        }
    })
}
```

### Custom Metrics

```go
func BenchmarkThroughput(b *testing.B) {
    data := make([]byte, 1<<20) // 1MB
    b.SetBytes(int64(len(data))) // enables MB/s reporting

    for i := 0; i < b.N; i++ {
        Compress(data)
    }
}

func BenchmarkCustomMetric(b *testing.B) {
    var items int
    for i := 0; i < b.N; i++ {
        items += ProcessBatch()
    }
    b.ReportMetric(float64(items)/float64(b.N), "items/op")
}
```

### Preventing Compiler Optimization

```go
// BAD — compiler may optimize away the result
func BenchmarkBad(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ComputeHash("hello") // result discarded, may be eliminated
    }
}

// GOOD — use a package-level sink variable
var sink interface{}

func BenchmarkGood(b *testing.B) {
    for i := 0; i < b.N; i++ {
        sink = ComputeHash("hello")
    }
}

// ALSO GOOD — use b.StopTimer trick or runtime.KeepAlive
func BenchmarkAlsoGood(b *testing.B) {
    var result []byte
    for i := 0; i < b.N; i++ {
        result = ComputeHash("hello")
    }
    runtime.KeepAlive(result)
}
```

## Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkEncode -benchmem

# Custom duration
go test -bench=. -benchtime=5s

# Fixed iteration count
go test -bench=. -benchtime=1000x

# Multiple runs for statistical analysis
go test -bench=. -count=10 | tee bench.txt

# With CPU profile
go test -bench=BenchmarkEncode -cpuprofile=cpu.out
go tool pprof cpu.out

# With memory profile
go test -bench=BenchmarkEncode -memprofile=mem.out
go tool pprof mem.out

# Disable tests, only run benchmarks
go test -run='^$' -bench=.

# With trace
go test -bench=BenchmarkEncode -trace=trace.out
go tool trace trace.out
```

## benchstat

### Installation and Usage

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

### Comparing Two Runs

```bash
# Collect baseline
go test -bench=. -count=10 > old.txt

# Make changes, collect new measurements
go test -bench=. -count=10 > new.txt

# Compare
benchstat old.txt new.txt
```

Output:

```
name        old time/op    new time/op    delta
Encode-8    45.2ms ± 3%    38.7ms ± 2%   -14.38%  (p=0.000 n=10+10)
Decode-8    12.1ms ± 1%    12.3ms ± 1%     ~       (p=0.095 n=10+10)
```

The `~` means no statistically significant difference (Welch's t-test, $p > 0.05$).

### Single Run Analysis

```bash
benchstat bench.txt
```

Shows median, confidence interval, and whether the measurement is stable.

### Filtering

```bash
benchstat -filter '.name:Encode' old.txt new.txt
```

## pprof Integration

### CPU Profile from Benchmark

```bash
go test -bench=BenchmarkHot -cpuprofile=cpu.out
go tool pprof -http=:6060 cpu.out
```

### Memory Profile

```bash
go test -bench=BenchmarkAllocHeavy -memprofile=mem.out -memprofilerate=1
go tool pprof -http=:6060 mem.out
```

### In-Code Profiling

```go
func BenchmarkWithProfile(b *testing.B) {
    for i := 0; i < b.N; i++ {
        result := ComplexOperation()
        _ = result
    }

    // After benchmark, you can examine profiles:
    // go test -bench=BenchmarkWithProfile -cpuprofile=cpu.out -memprofile=mem.out
}
```

### Common pprof Commands

```bash
# Top consumers
go tool pprof -top cpu.out

# Web UI
go tool pprof -http=:8080 cpu.out

# Flamegraph (built into web UI)
go tool pprof -http=:8080 -flame cpu.out

# Text call graph
go tool pprof -text cpu.out

# Focus on specific function
go tool pprof -focus=Encode cpu.out
```

## Rust: criterion

### Setup

```toml
# Cargo.toml
[dev-dependencies]
criterion = { version = "0.5", features = ["html_reports"] }

[[bench]]
name = "my_benchmark"
harness = false
```

### Basic Benchmark

```rust
// benches/my_benchmark.rs
use criterion::{criterion_group, criterion_main, Criterion, BenchmarkId};

fn fibonacci(n: u64) -> u64 {
    match n {
        0 | 1 => n,
        _ => fibonacci(n - 1) + fibonacci(n - 2),
    }
}

fn bench_fibonacci(c: &mut Criterion) {
    c.bench_function("fib 20", |b| b.iter(|| fibonacci(20)));
}

fn bench_fibonacci_group(c: &mut Criterion) {
    let mut group = c.benchmark_group("fibonacci");
    for size in [10, 15, 20, 25].iter() {
        group.bench_with_input(
            BenchmarkId::from_parameter(size),
            size,
            |b, &size| b.iter(|| fibonacci(size)),
        );
    }
    group.finish();
}

criterion_group!(benches, bench_fibonacci, bench_fibonacci_group);
criterion_main!(benches);
```

### Preventing Optimization

```rust
use criterion::black_box;

fn bench_hash(c: &mut Criterion) {
    c.bench_function("hash", |b| {
        b.iter(|| compute_hash(black_box("hello")))
    });
}
```

### Running

```bash
cargo bench                            # run all benchmarks
cargo bench -- fibonacci               # filter by name
cargo bench -- --save-baseline before   # save baseline
# make changes
cargo bench -- --baseline before        # compare against baseline
```

Criterion generates HTML reports in `target/criterion/report/index.html`.

## hyperfine (CLI Benchmarking)

### Installation

```bash
# macOS
brew install hyperfine

# cargo
cargo install hyperfine
```

### Basic Usage

```bash
# Simple benchmark
hyperfine 'sleep 0.3'

# Compare commands
hyperfine 'fd -e go' 'find . -name "*.go"'

# With warmup
hyperfine --warmup 3 'my-program input.txt'

# Specific number of runs
hyperfine --min-runs 20 'my-program'

# Export results
hyperfine --export-json results.json 'cmd1' 'cmd2'
hyperfine --export-markdown results.md 'cmd1' 'cmd2'
```

### Parameter Sweeps

```bash
# Vary a parameter
hyperfine --parameter-scan threads 1 8 'my-program --threads {threads}'

# List of values
hyperfine --parameter-list lang go,rust,python './{lang}-impl input.txt'
```

### Setup and Cleanup

```bash
hyperfine \
    --setup 'make build' \
    --prepare 'sync; echo 3 | sudo tee /proc/sys/vm/drop_caches' \
    --cleanup 'rm -f output.tmp' \
    'my-program > output.tmp'
```

### Shell Selection

```bash
# Use specific shell
hyperfine --shell=none './my-binary arg1 arg2'  # no shell overhead
```

## Common Measurement Traps

### Trap 1: JIT/Cache Cold Start

```go
// BAD — first iteration is cold, rest are warm
func BenchmarkCold(b *testing.B) {
    for i := 0; i < b.N; i++ {
        result := ExpensiveInit() // cached after first call
        _ = result
    }
}

// GOOD — measure what you mean
func BenchmarkWarm(b *testing.B) {
    ExpensiveInit() // warm up
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        result := CachedOperation()
        _ = result
    }
}
```

### Trap 2: Input Size Matters

```go
// BAD — benchmarks only one size
func BenchmarkSort(b *testing.B) {
    data := []int{3, 1, 4, 1, 5}
    for i := 0; i < b.N; i++ {
        sort.Ints(append([]int{}, data...))
    }
}

// GOOD — parameterize
func BenchmarkSort(b *testing.B) {
    for _, n := range []int{10, 100, 1000, 10000, 100000} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            data := rand.Perm(n)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                tmp := append([]int{}, data...)
                sort.Ints(tmp)
            }
        })
    }
}
```

### Trap 3: Power Management / Thermal Throttling

```bash
# Lock CPU frequency on Linux
sudo cpupower frequency-set -g performance

# Check frequency
cat /proc/cpuinfo | grep "cpu MHz"

# macOS — disable Turbo Boost
sudo pmset -a lidwake 0  # limited control
```

### Trap 4: Garbage Collection

```go
func BenchmarkGCPressure(b *testing.B) {
    // Force GC before benchmark
    runtime.GC()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        AllocHeavyOperation()
    }
}
```

## Tips

- Always use `-count=10` or more for benchstat — single runs are statistically meaningless
- Use `b.ReportAllocs()` routinely — allocation count often matters more than CPU time
- Use sink variables or `runtime.KeepAlive` to prevent dead code elimination
- Sub-benchmarks let you measure scaling behavior across input sizes
- benchstat's Welch t-test needs at least 5 samples per group ($n \geq 5$)
- Close background applications during benchmarks — browser tabs cause variance
- Pin CPU governor to `performance` on Linux to reduce noise
- hyperfine's `--warmup` flag is essential for disk-heavy benchmarks
- criterion auto-detects significant changes and generates comparison reports
- Profile first, benchmark second — pprof tells you *where*, benchmarks tell you *how much*

## See Also

- `sheets/testing/go-testing.md` — Go test runner and flags
- `detail/testing/benchmarking.md` — statistical theory behind benchstat
- `sheets/testing/coverage.md` — coverage-guided optimization

## References

- https://pkg.go.dev/testing#hdr-Benchmarks — Go benchmark documentation
- https://pkg.go.dev/golang.org/x/perf/cmd/benchstat — benchstat
- https://bheisler.github.io/criterion.rs/book/ — criterion user guide
- https://github.com/sharkdp/hyperfine — hyperfine
- https://go.dev/blog/pprof — profiling Go programs
