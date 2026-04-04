# Sanitizers (Runtime Memory and Behavior Error Detection)

> For authorized security testing, CTF competitions, and educational purposes only.

Sanitizers are compiler-instrumented runtime detectors that catch memory errors, data races, undefined behavior, and uninitialized memory reads during program execution. By inserting checks at compile time, they transform silent corruption into immediate, actionable crash reports with stack traces, making them essential for vulnerability discovery alongside fuzzing and manual security review.

---

## AddressSanitizer (ASAN)

### Compilation and Basic Usage

```bash
# Compile with ASAN (Clang)
clang -fsanitize=address -fno-omit-frame-pointer -g -O1 \
    -o target target.c

# Compile with ASAN (GCC)
gcc -fsanitize=address -fno-omit-frame-pointer -g -O1 \
    -o target target.c

# Compile C++ with ASAN
clang++ -fsanitize=address -fno-omit-frame-pointer -g -O1 \
    -o target target.cpp -lstdc++

# Link shared libraries with ASAN
clang -fsanitize=address -shared -fPIC -o libfoo.so foo.c

# Run with default settings
./target

# Run with custom ASAN options
ASAN_OPTIONS="detect_leaks=1:halt_on_error=0:print_stats=1" ./target

# Common ASAN_OPTIONS
# detect_leaks=1              # enable leak detection (default on Linux)
# halt_on_error=0             # continue after first error
# print_stats=1               # print memory stats at exit
# malloc_context_size=30      # deeper stack traces for allocs
# quarantine_size_mb=256      # increase quarantine for UAF detection
# redzone_size=128            # larger redzones for overflow detection
# detect_stack_use_after_return=1  # detect stack UAR (slower)
# symbolize=1                 # symbolize stack traces
# suppressions=asan.supp      # suppression file
# log_path=asan.log           # write reports to file

# Suppression file format (asan.supp)
# interceptor_via_fun:libthirdparty.so
# leak:known_benign_leak_function
```

### What ASAN Detects

```bash
# Heap buffer overflow
# char *buf = malloc(10);
# buf[10] = 'x';  // ERROR: heap-buffer-overflow

# Stack buffer overflow
# char buf[10];
# buf[10] = 'x';  // ERROR: stack-buffer-overflow

# Global buffer overflow
# char g[10];
# g[10] = 'x';    // ERROR: global-buffer-overflow

# Use-after-free
# char *p = malloc(10);
# free(p);
# *p = 'x';       // ERROR: heap-use-after-free

# Use-after-return (requires detect_stack_use_after_return=1)
# char *p;
# void f() { char buf[10]; p = buf; }
# f(); *p = 'x';  // ERROR: stack-use-after-return

# Double-free
# char *p = malloc(10);
# free(p);
# free(p);         // ERROR: attempting double-free

# Memory leaks (at exit)
# char *p = malloc(10);
# return 0;        // ERROR: detected memory leaks

# Stack buffer underflow
# char buf[10];
# buf[-1] = 'x';   // ERROR: stack-buffer-underflow
```

### Interpreting ASAN Reports

```bash
# Sample ASAN output:
# ==12345==ERROR: AddressSanitizer: heap-buffer-overflow on address 0x60200000001a
#   WRITE of size 1 at 0x60200000001a thread T0
#     #0 0x4a3c8f in main /path/to/source.c:42:15
#     #1 0x7f... in __libc_start_main
#
# 0x60200000001a is located 0 bytes to the right of 10-byte region
#   [0x602000000010,0x60200000001a)
# allocated by thread T0 here:
#     #0 0x493e2d in malloc (asan_interceptors.cpp)
#     #1 0x4a3c4f in main /path/to/source.c:40:17

# Key information:
# - Error type: heap-buffer-overflow
# - Access type: WRITE of size 1
# - Crash location: source.c:42
# - Buffer info: 10-byte region, overflow at byte 10
# - Allocation site: source.c:40

# Symbolize addresses manually if symbols are missing
ASAN_OPTIONS="symbolize=1" ASAN_SYMBOLIZER_PATH=/usr/bin/llvm-symbolizer ./target

# Or use addr2line
addr2line -e ./target 0x4a3c8f
```

## MemorySanitizer (MSAN)

### Detecting Uninitialized Memory Reads

```bash
# Compile with MSAN (Clang only, not available in GCC)
clang -fsanitize=memory -fno-omit-frame-pointer -g -O1 \
    -o target target.c

# With origin tracking (shows where uninitialized value originated)
clang -fsanitize=memory -fsanitize-memory-track-origins=2 \
    -fno-omit-frame-pointer -g -O1 -o target target.c

# Run
MSAN_OPTIONS="halt_on_error=0:print_stats=1" ./target

# Common MSAN_OPTIONS
# halt_on_error=0             # continue after first error
# print_stats=1               # print stats at exit
# origin_history_size=4       # depth of origin tracking
# wrap_signals=0              # disable signal interceptors

# What MSAN detects:
# int x;
# if (x > 0) { ... }         # ERROR: use-of-uninitialized-value
#
# char buf[100];
# read(fd, buf, 50);
# if (buf[75]) { ... }       # ERROR: use-of-uninitialized-value
#
# int *p = malloc(sizeof(int));
# printf("%d\n", *p);        # ERROR: use-of-uninitialized-value

# IMPORTANT: All code and libraries must be compiled with MSAN
# Mixing instrumented and uninstrumented code causes false positives
# Build entire dependency tree with MSAN:
CC=clang CFLAGS="-fsanitize=memory" ./configure && make

# MSAN report with origin tracking:
# ==12345==WARNING: MemorySanitizer: use-of-uninitialized-value
#     #0 0x4a3c8f in process_data /path/to/source.c:55:9
#   Uninitialized value was created by a heap allocation
#     #0 0x493e2d in malloc
#     #1 0x4a3c4f in init_buffer /path/to/source.c:40:17
```

## ThreadSanitizer (TSAN)

### Detecting Data Races

```bash
# Compile with TSAN (Clang)
clang -fsanitize=thread -g -O1 -o target target.c -lpthread

# Compile with TSAN (GCC)
gcc -fsanitize=thread -g -O1 -o target target.c -lpthread

# Run
TSAN_OPTIONS="halt_on_error=0:second_deadlock_stack=1" ./target

# Common TSAN_OPTIONS
# halt_on_error=0             # continue after first race
# second_deadlock_stack=1     # show both deadlock stacks
# history_size=7              # memory access history (0-7, higher=slower)
# force_seq_cst_atomics=1     # force sequential consistency
# suppressions=tsan.supp      # suppression file
# log_path=tsan.log           # write reports to file

# What TSAN detects:
# Data race (two threads accessing same memory, at least one write)
# int counter = 0;
# Thread 1: counter++;        # WRITE
# Thread 2: printf("%d", counter);  # READ -- DATA RACE

# Lock order inversion (potential deadlock)
# Thread 1: lock(A); lock(B);
# Thread 2: lock(B); lock(A);  # WARNING: lock-order-inversion

# Use-after-free with threads
# Thread 1: free(p);
# Thread 2: *p = 1;           # DATA RACE + use-after-free

# TSAN report example:
# ==12345==WARNING: ThreadSanitizer: data race (pid=12345)
#   Write of size 4 at 0x7f... by thread T2:
#     #0 increment /path/to/source.c:15:5
#   Previous read of size 4 at 0x7f... by thread T1:
#     #0 read_value /path/to/source.c:22:12
#   Location is global 'counter' of size 4 at 0x...

# Suppression file format (tsan.supp)
# race:third_party_lib_function
# deadlock:known_false_positive
# mutex:benign_race_on_logging
```

## UndefinedBehaviorSanitizer (UBSAN)

### Catching Undefined Behavior

```bash
# Compile with UBSAN (Clang)
clang -fsanitize=undefined -fno-omit-frame-pointer -g -O1 \
    -o target target.c

# Compile with UBSAN (GCC)
gcc -fsanitize=undefined -fno-omit-frame-pointer -g -O1 \
    -o target target.c

# Specific UBSAN checks
clang -fsanitize=signed-integer-overflow -o target target.c
clang -fsanitize=null -o target target.c
clang -fsanitize=alignment -o target target.c
clang -fsanitize=bounds -o target target.c

# All checks combined
clang -fsanitize=undefined,float-divide-by-zero,integer \
    -fno-sanitize-recover=all -o target target.c

# Run
UBSAN_OPTIONS="halt_on_error=1:print_stacktrace=1" ./target

# Common UBSAN_OPTIONS
# halt_on_error=1             # abort on first UB (default: continue)
# print_stacktrace=1          # include stack trace
# suppressions=ubsan.supp     # suppression file
# silence_unsigned_overflow=1 # suppress unsigned overflow reports

# What UBSAN detects:
# Signed integer overflow
# int x = INT_MAX; x++;       # ERROR: signed integer overflow

# Null pointer dereference
# int *p = NULL; *p = 1;      # ERROR: null pointer dereference

# Misaligned access
# char buf[16]; int *p = (int*)(buf+1); *p = 42;  # ERROR: misaligned

# Shift overflow
# int x = 1 << 32;            # ERROR: shift exponent too large

# Out-of-bounds array access
# int a[10]; a[10] = 1;       # ERROR: index 10 out of bounds

# Division by zero
# int x = 42 / 0;             # ERROR: division by zero

# Invalid bool/enum value
# bool b = *(bool*)"x";       # ERROR: load of value, not valid for bool

# Implicit conversion truncation (Clang only)
# clang -fsanitize=implicit-conversion
# uint8_t x = 256;            # ERROR: implicit conversion loses data
```

## Go Race Detector

### Built-In Data Race Detection

```bash
# Run tests with race detector
go test -race ./...

# Run specific test with race detector
go test -race -run TestConcurrentAccess ./pkg/server/

# Build binary with race detector
go build -race -o server ./cmd/server/

# Run binary with race detector
./server  # race detector active, reports at runtime

# Race detector environment variables
GORACE="log_path=race.log" ./server
GORACE="halt_on_error=1" ./server         # abort on first race
GORACE="history_size=5" ./server          # more history (1-7)
GORACE="atexit_sleep_ms=2000" ./server    # wait before exit

# What Go race detector catches:
# var counter int
# go func() { counter++ }()    # goroutine 1 writes
# fmt.Println(counter)          # main goroutine reads -- RACE

# Race report example:
# ==================
# WARNING: DATA RACE
# Write at 0x00c0000b4008 by goroutine 7:
#   main.increment()
#       /path/to/main.go:15 +0x4a
# Previous read at 0x00c0000b4008 by goroutine 6:
#   main.readCounter()
#       /path/to/main.go:22 +0x3e
# Goroutine 7 (running) created at:
#   main.main()
#       /path/to/main.go:30 +0x96
# ==================

# Fix: use sync.Mutex, sync/atomic, or channels
# var mu sync.Mutex
# mu.Lock(); counter++; mu.Unlock()
# or: atomic.AddInt64(&counter, 1)
```

## Rust Miri (MIR Interpreter)

### Detecting Undefined Behavior in Unsafe Rust

```bash
# Install Miri
rustup +nightly component add miri

# Run tests under Miri
cargo +nightly miri test

# Run a specific binary under Miri
cargo +nightly miri run

# Miri flags
MIRIFLAGS="-Zmiri-disable-isolation" cargo +nightly miri test
MIRIFLAGS="-Zmiri-tag-raw-pointers" cargo +nightly miri test

# Common MIRIFLAGS
# -Zmiri-disable-isolation        # allow file/env access
# -Zmiri-tag-raw-pointers         # detect aliasing violations
# -Zmiri-check-number-validity    # validate number types
# -Zmiri-symbolic-alignment-check # strict alignment
# -Zmiri-seed=42                  # deterministic execution

# What Miri detects:
# Out-of-bounds access in unsafe blocks
# Dangling pointer dereference
# Invalid enum discriminant values
# Unaligned pointer access
# Use of uninitialized memory
# Data races (in concurrent code with -Zmiri-preemption-rate)
# Stacked Borrows violations (aliasing rules)
# Memory leaks

# Miri report example:
# error: Undefined Behavior: dereferencing pointer failed:
#   alloc1384 has been freed
#   --> src/main.rs:42:18
#    |
# 42 |     unsafe { *ptr = 5; }
#    |              ^^^^^^^^ dereferencing pointer failed
```

## Combining Sanitizers

### Multi-Sanitizer Build Strategies

```bash
# ASAN + UBSAN (compatible, recommended combination)
clang -fsanitize=address,undefined -fno-omit-frame-pointer \
    -g -O1 -o target target.c

# ASAN + UBSAN + fuzzing (for fuzz targets)
clang -fsanitize=fuzzer,address,undefined \
    -fno-omit-frame-pointer -g -O1 \
    -o fuzz_target fuzz_target.c

# MSAN + UBSAN (compatible)
clang -fsanitize=memory,undefined -fno-omit-frame-pointer \
    -g -O1 -o target target.c

# INCOMPATIBLE combinations (do NOT mix):
# ASAN + TSAN    -- both intercept memory, will conflict
# ASAN + MSAN    -- both shadow memory, will conflict
# TSAN + MSAN    -- both shadow memory, will conflict

# Build matrix for CI (separate builds)
# Build 1: ASAN + UBSAN    (memory errors + undefined behavior)
# Build 2: TSAN            (data races)
# Build 3: MSAN + UBSAN    (uninitialized reads + UB)
# Build 4: UBSAN alone     (lightest, fastest)

# Performance impact (approximate):
# ASAN:  2x slowdown, 3x memory
# TSAN:  5-15x slowdown, 5-10x memory
# MSAN:  3x slowdown, 2x memory
# UBSAN: 1.2x slowdown, minimal memory
```

## CI Integration Patterns

### Continuous Sanitizer Testing

```bash
# GitHub Actions example (multi-sanitizer matrix)
# jobs:
#   sanitizers:
#     strategy:
#       matrix:
#         sanitizer: [address, thread, memory, undefined]
#     steps:
#       - run: |
#           CC=clang CXX=clang++ \
#           CFLAGS="-fsanitize=${{ matrix.sanitizer }} -g -O1" \
#           CXXFLAGS="-fsanitize=${{ matrix.sanitizer }} -g -O1" \
#           LDFLAGS="-fsanitize=${{ matrix.sanitizer }}" \
#           make clean all test

# Go CI with race detector
# go test -race -count=1 -timeout 300s ./...

# Rust CI with Miri
# cargo +nightly miri test --all-targets

# CMake integration
# cmake -DCMAKE_C_COMPILER=clang \
#       -DCMAKE_C_FLAGS="-fsanitize=address,undefined -g" \
#       -DCMAKE_EXE_LINKER_FLAGS="-fsanitize=address,undefined" \
#       -DCMAKE_BUILD_TYPE=Debug ..
# cmake --build . && ctest

# Makefile pattern
# SANITIZER ?= address
# CFLAGS += -fsanitize=$(SANITIZER) -fno-omit-frame-pointer -g
# LDFLAGS += -fsanitize=$(SANITIZER)
# make SANITIZER=thread test

# Fail CI on any sanitizer finding
# ASAN_OPTIONS="halt_on_error=1:exitcode=42" ./test_suite
# TSAN_OPTIONS="halt_on_error=1:exitcode=42" ./test_suite
# UBSAN_OPTIONS="halt_on_error=1:exitcode=42" ./test_suite
```

## False Positive Handling

### Managing Suppression and Known Issues

```bash
# ASAN suppression file
# Create asan_suppressions.txt:
# interceptor_via_fun:libpng_read_row
# leak:libcrypto
# odr_violation:duplicate_symbol

# TSAN suppression file
# Create tsan_suppressions.txt:
# race:logger_write
# deadlock:test_helper_lock
# signal:sig_handler_race
# called_from_lib:libzmq

# UBSAN suppression file
# Create ubsan_suppressions.txt:
# signed-integer-overflow:third_party/parser.c
# alignment:legacy_struct_access

# Apply suppressions
ASAN_OPTIONS="suppressions=asan_suppressions.txt" ./target
TSAN_OPTIONS="suppressions=tsan_suppressions.txt" ./target

# Source-level suppression (Clang attribute)
# __attribute__((no_sanitize("address")))
# void known_benign_overflow() { ... }

# __attribute__((no_sanitize("undefined")))
# void intentional_wrap() { ... }

# Go: suppress race detector for specific tests
# if testing.Short() { t.Skip("skipping race-prone test") }

# Mark intentional behavior (not a real bug)
# // This is safe because we control the bounds externally
# #pragma clang diagnostic push
# #pragma clang diagnostic ignored "-Warray-bounds"
# buf[computed_index] = val;
# #pragma clang diagnostic pop
```

---

## Tips

- Always compile with `-fno-omit-frame-pointer` when using sanitizers; without it, stack traces are incomplete or missing
- Use `-O1` optimization with sanitizers; `-O0` is slower and `-O2` may optimize away the buggy code you are trying to catch
- Run ASAN and TSAN as separate CI jobs since they are mutually incompatible and intercept the same runtime functions
- Enable `detect_stack_use_after_return=1` for ASAN when testing code that returns pointers to stack variables
- Use MSAN with origin tracking (`-fsanitize-memory-track-origins=2`) to trace where uninitialized values first appeared
- The Go race detector adds approximately 10x overhead; use shorter test timeouts and fewer iterations in race-enabled CI runs
- Pair ASAN with fuzzing (`-fsanitize=fuzzer,address`) for maximum vulnerability discovery; most OSS-Fuzz findings use this combination
- Write suppression files for known false positives in third-party libraries rather than disabling sanitizers entirely
- UBSAN has minimal overhead and should be enabled in every debug build; it catches bugs that other sanitizers miss
- Run Rust unsafe code through Miri regularly; it detects aliasing violations that no other tool catches

---

## See Also

- fuzzing
- gdb-security
- checksec

## References

- [AddressSanitizer Documentation](https://clang.llvm.org/docs/AddressSanitizer.html)
- [MemorySanitizer Documentation](https://clang.llvm.org/docs/MemorySanitizer.html)
- [ThreadSanitizer Documentation](https://clang.llvm.org/docs/ThreadSanitizer.html)
- [UndefinedBehaviorSanitizer Documentation](https://clang.llvm.org/docs/UndefinedBehaviorSanitizer.html)
- [Go Race Detector](https://go.dev/doc/articles/race_detector)
- [Rust Miri](https://github.com/rust-lang/miri)
- [Google Sanitizers Wiki](https://github.com/google/sanitizers/wiki)
- [LLVM Sanitizer Coverage](https://clang.llvm.org/docs/SanitizerCoverage.html)
