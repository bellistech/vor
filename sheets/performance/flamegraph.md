# Flame Graphs (Performance Visualization)

Stack trace visualization that shows which code paths consume the most resources (CPU, memory, off-CPU time).

## Core Pipeline

### perf record to flame graph (Linux)

```bash
# 1. Record CPU samples (30 seconds, all CPUs)
sudo perf record -F 99 -a -g -- sleep 30

# 2. Generate human-readable stacks
sudo perf script > out.perf

# 3. Collapse stacks (one line per stack)
./stackcollapse-perf.pl out.perf > out.folded

# 4. Generate SVG flame graph
./flamegraph.pl out.folded > flamegraph.svg

# All in one pipeline
sudo perf record -F 99 -a -g -- sleep 30 && \
  sudo perf script | ./stackcollapse-perf.pl | ./flamegraph.pl > flamegraph.svg
```

### Record a specific process

```bash
# By PID
sudo perf record -F 99 -p $(pgrep myapp) -g -- sleep 30

# By command
sudo perf record -F 99 -g -- ./myapp --flag

# With DWARF unwinding (when frame pointers are missing)
sudo perf record -F 99 -p $PID --call-graph dwarf -- sleep 30

# With LBR (Last Branch Record, Intel CPUs)
sudo perf record -F 99 -p $PID --call-graph lbr -- sleep 30
```

## Install Flame Graph Tools

```bash
# Clone Brendan Gregg's flame graph repo
git clone https://github.com/brendangregg/FlameGraph.git
cd FlameGraph

# Key scripts:
# stackcollapse-perf.pl   — collapse perf script output
# stackcollapse-jstack.pl — collapse jstack thread dumps
# stackcollapse-stap.pl   — collapse SystemTap output
# stackcollapse-dtrace.pl — collapse DTrace output
# stackcollapse-go.pl     — collapse Go pprof text output
# flamegraph.pl           — generate SVG from collapsed stacks
# difffolded.pl           — differential flame graphs
```

## CPU Flame Graphs

### Go (pprof)

```bash
# Collect CPU profile (30 seconds)
curl -o cpu.pprof 'http://localhost:6060/debug/pprof/profile?seconds=30'

# View in browser (built-in flame graph)
go tool pprof -http=:8080 cpu.pprof

# Generate collapsed stacks for flamegraph.pl
go tool pprof -raw cpu.pprof | ./stackcollapse-go.pl | ./flamegraph.pl > go-cpu.svg

# Or use pprof's built-in SVG
go tool pprof -svg cpu.pprof > cpu.svg
```

### Java (async-profiler)

```bash
# Download async-profiler
curl -Lo async-profiler.tar.gz \
  https://github.com/async-profiler/async-profiler/releases/latest/download/async-profiler-3.0-linux-x64.tar.gz
tar xzf async-profiler.tar.gz

# Profile a running JVM (CPU, 30 seconds)
./asprof -d 30 -f flamegraph.html $JAVA_PID

# Profile with specific event
./asprof -d 30 -e cpu -f cpu.html $JAVA_PID
./asprof -d 30 -e alloc -f alloc.html $JAVA_PID
./asprof -d 30 -e lock -f lock.html $JAVA_PID
./asprof -d 30 -e wall -f wall.html $JAVA_PID

# Output collapsed stacks
./asprof -d 30 -o collapsed -f out.folded $JAVA_PID
```

### Python (py-spy)

```bash
# Install py-spy
pip install py-spy

# Record to SVG flame graph
sudo py-spy record -o flamegraph.svg --pid $PID

# Record for specific duration
sudo py-spy record -o flamegraph.svg --pid $PID --duration 30

# Record with subprocesses
sudo py-spy record -o flamegraph.svg --pid $PID --subprocesses

# Native extensions included
sudo py-spy record -o flamegraph.svg --pid $PID --native

# Top-like live view
sudo py-spy top --pid $PID
```

### Node.js

```bash
# Using 0x (generates flame graph directly)
npx 0x -- node app.js

# Using perf with Node.js
node --perf-basic-prof app.js &
sudo perf record -F 99 -p $! -g -- sleep 30
sudo perf script | ./stackcollapse-perf.pl | \
  grep -v 'v8::internal' | ./flamegraph.pl > node.svg

# Using clinic.js
npx clinic flame -- node app.js
```

### Rust / C / C++

```bash
# perf (frame pointers enabled: -C force-frame-pointers=yes for Rust, -fno-omit-frame-pointer for C/C++)
sudo perf record -F 99 -p $PID -g -- sleep 30
sudo perf script | ./stackcollapse-perf.pl | ./flamegraph.pl > rust.svg

# Using cargo-flamegraph (Rust)
cargo install flamegraph
cargo flamegraph --bin myapp

# With DWARF info
sudo perf record -F 99 -p $PID --call-graph dwarf -- sleep 30
```

## Off-CPU Flame Graphs

```bash
# Record scheduling events (off-CPU = time spent sleeping/blocked)
sudo perf record -e sched:sched_switch -a -g -- sleep 30
sudo perf script | ./stackcollapse-perf.pl | \
  ./flamegraph.pl --color=io --title="Off-CPU Flame Graph" > offcpu.svg

# Using bpftrace (more efficient)
sudo bpftrace -e '
  kprobe:finish_task_switch {
    @[kstack] = count();
  }
' > /tmp/offcpu.bt 2>&1 &
sleep 30; kill $!
# Parse output with stackcollapse-bpftrace.pl
```

## Memory Flame Graphs

### Go Heap

```bash
# Collect heap profile
curl -o heap.pprof http://localhost:6060/debug/pprof/heap

# View in browser
go tool pprof -http=:8080 heap.pprof

# alloc_space vs inuse_space
go tool pprof -alloc_space -http=:8080 heap.pprof    # total allocated
go tool pprof -inuse_space -http=:8080 heap.pprof    # currently held
```

### perf (malloc tracing)

```bash
# Trace malloc calls with stack traces
sudo perf record -e probe_libc:malloc -a -g -- sleep 10
sudo perf script | ./stackcollapse-perf.pl | \
  ./flamegraph.pl --color=mem --title="Malloc Flame Graph" > malloc.svg
```

## Differential Flame Graphs

```bash
# 1. Collect baseline
sudo perf record -F 99 -a -g -o perf-before.data -- sleep 30
sudo perf script -i perf-before.data | ./stackcollapse-perf.pl > before.folded

# 2. Make changes / deploy new version

# 3. Collect comparison
sudo perf record -F 99 -a -g -o perf-after.data -- sleep 30
sudo perf script -i perf-after.data | ./stackcollapse-perf.pl > after.folded

# 4. Generate differential flame graph
./difffolded.pl before.folded after.folded | \
  ./flamegraph.pl --title="Differential: Before vs After" > diff.svg

# Red = regression (more samples), blue = improvement (fewer samples)
```

## Flame Graph Options

```bash
# flamegraph.pl options
./flamegraph.pl \
  --title="My Flame Graph" \
  --subtitle="PID 12345, 30s sample" \
  --width=1200 \
  --height=16 \                        # frame height in pixels
  --minwidth=0.1 \                     # hide frames below 0.1% width
  --fontsize=12 \
  --countname="samples" \
  --nametype="Function:" \
  --colors=hot \                       # hot, mem, io, java, perl, js
  --hash \                             # consistent colors by function name
  --reverse \                          # icicle graph (top-down)
  --inverted \                         # invert call order
  --negate \                           # swap differential colors
  out.folded > flamegraph.svg

# Grep filter (show only matching stacks)
grep 'http' out.folded | ./flamegraph.pl > http-only.svg

# Exclude kernel stacks
grep -v '^\[kernel' out.folded | ./flamegraph.pl > userspace.svg
```

## Reading Flame Graphs

```
x-axis = stack population (NOT time), wider = more samples
y-axis = stack depth (bottom = entry point, top = on-CPU function)
color  = random (CPU), or semantic (mem/io/differential)

Key patterns:
- Wide plateau at top    → hot function (high self-time)
- Wide tower             → deep call chain consuming CPU
- Narrow spikes          → infrequent but deep stacks
- Flat top               → leaf function (no children)
```

## Tips

- Always record with `-g` (call graphs) otherwise perf produces flat profiles, not flame graphs
- Use frame pointers (`-fno-omit-frame-pointer`) for reliable stack unwinding; DWARF is slower but works without them
- Set sample rate to 99 Hz (not 100) to avoid lockstep synchronization with system timers
- Off-CPU flame graphs reveal I/O bottlenecks and lock contention that CPU flame graphs miss entirely
- Differential flame graphs are the fastest way to diagnose performance regressions after deploys
- Filter collapsed stacks with `grep` before generating the SVG to focus on specific subsystems
- Use `--reverse` to generate icicle graphs (top-down) which some people find more intuitive
- The x-axis is NOT time; it is alphabetically sorted stack population -- do not read left-to-right
- For Go, prefer `go tool pprof -http` over manual flamegraph.pl; pprof has built-in interactive flame graphs
- async-profiler for Java avoids the safepoint bias problem that JFR and jstack-based tools suffer from
- Memory flame graphs distinguish allocation sites (where objects are created) from retention sites (where objects are kept alive)
- Combine CPU and off-CPU flame graphs to get a complete picture: CPU shows compute, off-CPU shows waiting

## See Also

- perf
- ebpf
- bpftrace
- pyroscope
- valgrind

## References

- [Flame Graphs — Brendan Gregg](https://www.brendangregg.com/flamegraphs.html)
- [FlameGraph GitHub Repository](https://github.com/brendangregg/FlameGraph)
- [async-profiler GitHub](https://github.com/async-profiler/async-profiler)
- [py-spy GitHub](https://github.com/benfred/py-spy)
- [Go pprof Documentation](https://pkg.go.dev/net/http/pprof)
- [Linux perf Wiki](https://perf.wiki.kernel.org/index.php/Main_Page)
- [Off-CPU Flame Graphs — Brendan Gregg](https://www.brendangregg.com/offcpuanalysis.html)
