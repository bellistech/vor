# Flamegraph (CPU Profile Visualization)

A flamegraph is a stack-trace visualization that turns thousands of profiler samples into a single SVG where width represents time and depth represents call hierarchy — the canonical tool for finding hot paths in CPU, off-CPU, memory, and lock-contention profiles.

## Setup

```bash
# Canonical install — Brendan Gregg's original Perl scripts
git clone https://github.com/brendangregg/FlameGraph ~/FlameGraph
cd ~/FlameGraph

# Add to PATH for convenience
export PATH="$HOME/FlameGraph:$PATH"

# Verify
which flamegraph.pl
which stackcollapse-perf.pl
```

```bash
# What's in the repo
ls ~/FlameGraph/*.pl
# flamegraph.pl                 — SVG renderer (the main tool)
# stackcollapse-perf.pl         — Linux perf script -> folded
# stackcollapse-bpftrace.pl     — bpftrace output -> folded
# stackcollapse-stap.pl         — SystemTap -> folded
# stackcollapse-jstack.pl       — Java jstack thread dumps -> folded
# stackcollapse-go.pl           — Go pprof -raw -> folded
# stackcollapse-elfutils.pl     — elfutils stack samples
# stackcollapse-instruments.pl  — macOS Instruments deep copy
# stackcollapse-pmc.pl          — FreeBSD pmcstat
# stackcollapse-vsprof.pl       — Visual Studio profiler
# stackcollapse-vtune.pl        — Intel VTune
# stackcollapse-recursive.pl    — fold recursive calls into one frame
# stackcollapse-chrome.pl       — Chrome JSON timeline
# difffolded.pl                 — diff two folded files
# range-perf.pl                 — slice perf data by time range
```

```bash
# Alternative 1: Rust port (faster, drop-in)
cargo install inferno
# Provides: inferno-collapse-perf, inferno-collapse-dtrace, inferno-flamegraph
# Same args/output as the Perl scripts but ~10x faster on large traces

# Alternative 2: speedscope — interactive web viewer
npm install -g speedscope
# Drag-and-drop a folded or pprof or chrome JSON at https://www.speedscope.app/

# Alternative 3: Pyroscope — continuous profiling product
docker run -it -p 4040:4040 grafana/pyroscope
# Continuous flamegraphs, time-travel, diffs

# Alternative 4: Parca — eBPF-based always-on
# https://www.parca.dev/ — unprivileged whole-system profiling
```

```bash
# The canonical pipeline (memorize this)
profiler-output | stackcollapse-X.pl | flamegraph.pl > flame.svg
#       |                  |                  |
#       |                  |                  +-- renders SVG
#       |                  +-- normalizes to "frame1;frame2 N" lines
#       +-- raw stack samples from the profiler
```

## Reading a Flamegraph — Visual Anatomy

```bash
# x-axis is NOT time. It is alphabetical sort of frames at each level.
#   Reading left-to-right tells you NOTHING about chronology.
#   Width = sample count = CPU time in that frame (and its children).
#
# y-axis IS depth. Bottom = entry point (main, _start). Top = on-CPU leaf.
#
# Each rectangle = one stack frame seen at depth Y, with width
#   proportional to how many samples included it at that depth.
#
# Colors: by default, warm random palette (no semantic meaning).
#   Use --colors=mem / io / java / js / red / green / blue / yellow
#   to encode meaning (memory, I/O, runtime, etc.).
```

```bash
# The 4 patterns to look for:
#
# 1. Wide PLATEAU at the top of a tower
#    -> a function with high SELF time. This is your hot leaf.
#       Optimize this code or call it less.
#
# 2. Wide PYRAMID with a small tip
#    -> a high-level path that fans out into many cheap leaves.
#       Optimization target: reduce calls to the wide base.
#
# 3. RECURSIVE TOWER (same name stacked vertically)
#    -> deep recursion. Often fine, sometimes a bug.
#       Use stackcollapse-recursive.pl to fold it.
#
# 4. NARROW SPIKE
#    -> rare, deep stack. Usually noise. Ignore unless investigating
#       a specific path.
```

```bash
# Interaction (in a browser, on the SVG):
#  - CLICK a frame      -> zoom into that subtree (rest of graph dims)
#  - "Reset Zoom" link  -> back to full view
#  - "Search" link      -> highlight all frames matching a regex
#  - Hover              -> tooltip with function name + sample count + %
```

```bash
# THE rule: "Look for plateaus, not spikes."
# A wide flat top = CPU stuck in one function. That's your bottleneck.
```

## The Pipeline

```bash
# Stage 1: capture raw stacks from a profiler
#   - perf record + perf script (Linux)
#   - bpftrace
#   - dtrace (macOS, FreeBSD, Solaris)
#   - py-spy / async-profiler / rbspy (language-specific)
#   - go tool pprof -raw (Go)

# Stage 2: collapse to folded format
#   stackcollapse-X.pl out.raw > out.folded
#
# Folded format (one stack per line):
#   main;parser_run;parse_token;hashmap_get 142
#   main;io_read;syscall_read 88
#   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ ^^^
#   semicolon-joined stack (root first)        sample count

# Stage 3: render SVG
#   flamegraph.pl out.folded > flame.svg

# Open the SVG in any browser. The SVG has embedded JS for zoom/search.
```

```bash
# Folded-format examples — note semicolons join, root is leftmost
cat out.folded | head -3
# main;event_loop;handle_request;parse_json;json_lex 1042
# main;event_loop;handle_request;db_query;tcp_send 318
# main;event_loop;timer_tick 7
```

```bash
# You can write folded files by hand for testing
cat > demo.folded <<'EOF'
main;a;b 50
main;a;c 30
main;d 20
EOF
flamegraph.pl demo.folded > demo.svg
open demo.svg
```

## perf — On-CPU Flamegraph

```bash
# Linux only. Standard CPU profiling pipeline.
# 99 Hz (NOT 100 Hz) avoids aliasing with NTP/timer interrupts.

# Profile the entire system for 30s
sudo perf record -F 99 -a -g -- sleep 30
sudo perf script > out.perf
stackcollapse-perf.pl out.perf > out.folded
flamegraph.pl out.folded > on-cpu.svg
```

```bash
# Profile a specific PID for 30s
sudo perf record -F 99 -p $(pgrep -f myapp) -g -- sleep 30
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > app.svg
```

```bash
# Profile a process you launch
sudo perf record -F 99 -g -- ./myapp --workload=heavy
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > launch.svg
```

```bash
# The one-liner everyone uses
sudo perf record -F 99 -a -g -- sleep 30 \
  && sudo perf script \
  | stackcollapse-perf.pl \
  | flamegraph.pl --title="On-CPU $(date +%F)" > on-cpu.svg
```

```bash
# When frame pointers are missing — DWARF unwinding
sudo perf record -F 99 -p $PID --call-graph dwarf -- sleep 30
# Slower, larger output, but works on binaries built with -fomit-frame-pointer
```

```bash
# Last Branch Record (Intel-only, fast, shallow stacks)
sudo perf record -F 99 -p $PID --call-graph lbr -- sleep 30
# LBR depth limited to ~16-32 entries depending on CPU generation
```

```bash
# Read perf.data later (no recapture needed)
sudo perf script -i perf.data | stackcollapse-perf.pl | flamegraph.pl > later.svg
```

## perf — Off-CPU Flamegraph

```bash
# CONTRAST:
#   On-CPU  = where the CPU is busy (compute hot paths)
#   Off-CPU = where threads SLEEP (I/O wait, lock contention, mutex,
#             disk read, network recv, sched delay)
#
# The two views answer different questions:
#   "Where is my code burning CPU?"   -> on-CPU
#   "Where is my code blocked?"       -> off-CPU
```

```bash
# Old-school perf-based off-CPU (slow, high overhead — use bpftrace instead)
sudo perf record -e sched:sched_switch -a -g -- sleep 30
sudo perf script | stackcollapse-perf.pl \
  | flamegraph.pl --colors=io --title="Off-CPU" > off-cpu.svg
```

```bash
# Best practice: use bpftrace below — orders of magnitude cheaper
# because it aggregates in-kernel and never copies stacks until done.
```

## bpftrace Off-CPU

```bash
# Capture per-thread off-CPU time with kernel stacks (root required)
sudo bpftrace -e '
kprobe:finish_task_switch {
  $prev = (struct task_struct *)arg0;
  @start[$prev->pid] = nsecs;
}
kprobe:try_to_wake_up {
  $waking = (struct task_struct *)arg0;
  $pid = $waking->pid;
  if (@start[$pid]) {
    @stacks[kstack($pid), $waking->comm] = sum(nsecs - @start[$pid]);
    delete(@start[$pid]);
  }
}
' > off-cpu.bt
# Run the workload, then Ctrl-C bpftrace.
```

```bash
# Convert bpftrace map output to folded format
stackcollapse-bpftrace.pl off-cpu.bt > off-cpu.folded
flamegraph.pl --colors=io --countname=ns --title="Off-CPU" \
  off-cpu.folded > off-cpu.svg
```

```bash
# User-stack version (off-CPU userspace stacks)
sudo bpftrace -e '
kprobe:finish_task_switch { @start[tid] = nsecs; }
kprobe:try_to_wake_up /@start[tid]/ {
  @us[ustack, comm] = sum(nsecs - @start[tid]);
  delete(@start[tid]);
}
' > off-cpu-user.bt
```

```bash
# Filter by PID
sudo bpftrace -e '
kprobe:finish_task_switch /pid == '"$PID"'/ { @start[tid] = nsecs; }
kprobe:try_to_wake_up /@start[tid]/ {
  @stacks[kstack, ustack] = sum(nsecs - @start[tid]);
  delete(@start[tid]);
}'
```

## bpftrace On-CPU Profile

```bash
# Sampled on-CPU profile via bpftrace (alternative to perf)
sudo bpftrace -e '
profile:hz:99 {
  @[ustack, comm] = count();
}
' > stacks.bt
# Ctrl-C after 30s.
```

```bash
# Convert with awk to folded
awk '
  /^@/ { stack = ""; next }
  /^\t/ { stack = $0 ";" stack; next }
  /^]/ { print stack count; stack = ""; next }
  { count = $NF }
' stacks.bt > stacks.folded
# Or just use stackcollapse-bpftrace.pl which handles the format
stackcollapse-bpftrace.pl stacks.bt > stacks.folded
flamegraph.pl stacks.folded > on-cpu-bpftrace.svg
```

```bash
# Userspace + kernel stacks combined
sudo bpftrace -e '
profile:hz:99 {
  @[kstack, ustack, comm] = count();
}'
```

## flamegraph.pl Flags

```bash
# Title and metadata
flamegraph.pl --title="My App, prod, 30s" \
              --subtitle="Captured 2024-12-01" \
              --countname="samples" \
              --nametype="Function:" \
              out.folded > out.svg
```

```bash
# Sizing
flamegraph.pl --width=1800 \
              --height=18 \
              --fontsize=11 \
              --fontwidth=0.59 \
              --minwidth=0.05 \
              out.folded > big.svg
# --width N      -> SVG width in pixels (default 1200)
# --height N     -> rectangle height in pixels (default 16)
# --fontsize N   -> label font size (default 12)
# --minwidth %   -> hide frames narrower than %.% (good for de-noising)
```

```bash
# Color palettes
flamegraph.pl --colors=hot     out.folded > hot.svg     # default warm reds/oranges
flamegraph.pl --colors=mem     out.folded > mem.svg     # green (memory)
flamegraph.pl --colors=io      out.folded > io.svg      # blue (I/O / off-CPU)
flamegraph.pl --colors=java    out.folded > java.svg    # JVM-aware (kernel/jit/inline)
flamegraph.pl --colors=js      out.folded > js.svg      # JS-aware
flamegraph.pl --colors=perl    out.folded > perl.svg
flamegraph.pl --colors=red     out.folded > red.svg
flamegraph.pl --colors=green   out.folded > green.svg
flamegraph.pl --colors=blue    out.folded > blue.svg
flamegraph.pl --colors=yellow  out.folded > yellow.svg
flamegraph.pl --colors=purple  out.folded > purple.svg
flamegraph.pl --colors=orange  out.folded > orange.svg
flamegraph.pl --colors=aqua    out.folded > aqua.svg
flamegraph.pl --colors=chain   out.folded > chain.svg   # color by depth
```

```bash
# Hash-based stable colors (same function = same color across captures)
flamegraph.pl --hash out.folded > stable.svg
```

```bash
# Background gradient
flamegraph.pl --bgcolors=grey   out.folded > grey-bg.svg
flamegraph.pl --bgcolors=yellow out.folded > yellow-bg.svg
flamegraph.pl --bgcolors=blue   out.folded > blue-bg.svg
flamegraph.pl --bgcolors=green  out.folded > green-bg.svg
```

```bash
# Layout variants
flamegraph.pl --reverse  out.folded > icicle.svg     # icicle: root at TOP, leaves at BOTTOM
flamegraph.pl --inverted out.folded > flipped.svg    # flip vertical without re-sorting
flamegraph.pl --flamechart out.folded > chart.svg    # do NOT sort frames; preserve order
```

```bash
# Differential rendering
flamegraph.pl --negate before.folded after.folded > diff.svg
# After --negate: red = removed time, blue = added time
```

```bash
# Help
flamegraph.pl --help
```

## stackcollapse Variants — When to Use Each

```bash
# stackcollapse-perf.pl       -> input: perf script
sudo perf script | stackcollapse-perf.pl > out.folded

# stackcollapse-bpftrace.pl   -> input: bpftrace map dump (kstack/ustack)
stackcollapse-bpftrace.pl bpf.txt > out.folded

# stackcollapse-stap.pl       -> input: SystemTap with print_ubacktrace()
stackcollapse-stap.pl stap.out > out.folded

# stackcollapse-elfutils.pl   -> input: eu-stack -1p PID  (elfutils package)
eu-stack -1p $PID > stacks.txt
stackcollapse-elfutils.pl stacks.txt > out.folded

# stackcollapse-instruments.pl-> input: macOS Instruments "deep copy" of Time Profiler
# (in Instruments: select call tree -> right-click -> Deep Copy -> paste)
pbpaste | stackcollapse-instruments.pl > out.folded

# stackcollapse-jstack.pl     -> input: jstack thread dump (multiple snapshots)
for i in {1..30}; do jstack $JAVA_PID; sleep 1; done > jstacks.txt
stackcollapse-jstack.pl jstacks.txt > out.folded

# stackcollapse-go.pl         -> input: go tool pprof -raw cpu.pprof
go tool pprof -raw -output=cpu.txt cpu.pprof
stackcollapse-go.pl cpu.txt > out.folded

# stackcollapse-pmc.pl        -> input: FreeBSD pmcstat -G
pmcstat -P CPU_CLK_UNHALTED.THREAD_P -O pmc.out -- sleep 30
pmcstat -R pmc.out -G - | stackcollapse-pmc.pl > out.folded

# stackcollapse-vsprof.pl     -> input: Visual Studio profiler CSV
stackcollapse-vsprof.pl report.csv > out.folded

# stackcollapse-vtune.pl      -> input: Intel VTune amplxe-cl text report
amplxe-cl -report callstacks -r vtune-r000pp/ -format text > vtune.txt
stackcollapse-vtune.pl vtune.txt > out.folded

# stackcollapse-recursive.pl  -> POST-PROCESS folded to fold A;A;A -> A
stackcollapse-perf.pl out.perf | stackcollapse-recursive.pl > out.folded

# stackcollapse-chrome.pl     -> input: Chrome DevTools Performance .json
stackcollapse-chrome.pl trace.json > out.folded
```

## JIT Languages — JIT Map Files

```bash
# Problem: perf walks process maps, but JIT-compiled code lives in
# anonymous mmap regions with NO symbols. Stacks show "[unknown]" or
# raw addresses like "0x7f1e23c40123".
#
# Solution: each JIT runtime can write a "perf map" listing addresses:
#   /tmp/perf-PID.map
#
# Format (one symbol per line):
#   START_ADDR_HEX SIZE_HEX symbol_name
#   7f1e23c40000 4d Lcom/foo/Bar.method
#
# perf script reads this file at the time you run perf script (NOT
# during record), so the map must still exist when you process.
```

```bash
# Per-runtime setup:

# Java -- async-profiler writes the map automatically when invoked
./asprof -d 30 -f flame.html $JAVA_PID
# Or use perf-map-agent:
git clone https://github.com/jvm-profiling-tools/perf-map-agent
cd perf-map-agent && cmake . && make
java -agentpath:./out/libperfmap.so -XX:+PreserveFramePointers -jar app.jar
# Or attach at runtime:
jcmd $JAVA_PID JVMTI.agent_load /path/to/libperfmap.so

# Node.js -- write /tmp/perf-PID.map
node --perf-basic-prof app.js              # JS-only frames
node --perf-basic-prof-only-functions app.js  # cleaner (no V8 internals)
node --perf-prof app.js                    # for JIT'd code

# .NET -- dotnet-trace can output speedscope/perf-collapsed
dotnet tool install -g dotnet-trace
dotnet-trace collect --format Speedscope --process-id $PID
# Or use perfcollect:
curl -o perfcollect https://raw.githubusercontent.com/microsoft/perfview/main/src/perfcollect/perfcollect
chmod +x perfcollect && ./perfcollect collect mytrace

# Symptom: stacks show [unknown] or hex addresses
#   -> perf-PID.map missing, malformed, or wrong PID
#   -> map file deleted before perf script ran
#   -> in containers, /tmp is namespaced — bind-mount or run perf inside
```

## Java Flamegraphs

```bash
# THE tool: async-profiler. Low overhead, accurate stacks, no safepoint
# bias (a problem JFR and jstack-sampling tools both have).

# Download
ASYNC_PROFILER=async-profiler-3.0-linux-x64
curl -Lo ap.tar.gz \
  https://github.com/async-profiler/async-profiler/releases/download/v3.0/$ASYNC_PROFILER.tar.gz
tar xzf ap.tar.gz
cd $ASYNC_PROFILER
```

```bash
# CPU flamegraph -- interactive HTML output
./bin/asprof -d 30 -f cpu.html $JAVA_PID
# Open cpu.html in a browser. Click frames to zoom.
```

```bash
# Other event types
./bin/asprof -d 30 -e alloc -f alloc.html $JAVA_PID  # allocation profile
./bin/asprof -d 30 -e lock  -f lock.html  $JAVA_PID  # lock contention
./bin/asprof -d 30 -e wall  -f wall.html  $JAVA_PID  # wall-clock (incl off-CPU)
./bin/asprof -d 30 -e cache-misses -f cm.html $JAVA_PID
./bin/asprof -d 30 -e itlb_misses.miss_causes_a_walk -f tlb.html $JAVA_PID
```

```bash
# Output collapsed format (compatible with flamegraph.pl pipeline)
./bin/asprof -d 30 -o collapsed -f stacks.folded $JAVA_PID
flamegraph.pl --colors=java stacks.folded > stacks.svg
```

```bash
# Start/stop control instead of fixed duration
./bin/asprof start -e cpu $JAVA_PID
# ... run workload ...
./bin/asprof stop -f profile.html $JAVA_PID
```

```bash
# Alternative: perf + perf-map-agent + frame pointers
java -XX:+UnlockDiagnosticVMOptions -XX:+PreserveFramePointers \
     -agentpath:./libperfmap.so \
     -jar app.jar &
JPID=$!
sudo perf record -F 99 -p $JPID -g -- sleep 30
sudo perf script | stackcollapse-perf.pl | flamegraph.pl --colors=java > java.svg
# Tradeoff: -XX:+PreserveFramePointers loses one register for compiled code,
# typically 0-3% throughput cost, but stacks become walkable by perf.
```

## Node.js Flamegraphs

```bash
# Approach 1: --perf-basic-prof + perf (Linux only)
node --perf-basic-prof app.js &
NPID=$!
sudo perf record -F 99 -p $NPID -g -- sleep 30
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > node.svg
# Filter V8 internals for cleaner JS-only output
sudo perf script | stackcollapse-perf.pl \
  | grep -v 'v8::internal' \
  | flamegraph.pl > node-clean.svg
```

```bash
# Approach 2: --perf-prof (newer, captures JIT'd code in /tmp/perf-PID.map)
node --perf-prof app.js
```

```bash
# Approach 3: 0x — opinionated wrapper that does it all
npm install -g 0x
0x -- node app.js
# Generates an interactive HTML flamegraph in 0x-PID/flamegraph.html
```

```bash
# Approach 4: clinic.js flame
npm install -g clinic
clinic flame -- node app.js
# Opens browser automatically when done
```

```bash
# Live profiling with --inspect + Chrome DevTools
node --inspect app.js
# In Chrome: open chrome://inspect, click "inspect" -> Performance tab
# Then "Save profile" -> .cpuprofile -> drag-drop into speedscope.app
```

## Python Flamegraphs

```bash
# THE tool: py-spy. Sampling profiler that doesn't require code changes
# and works on running processes via /proc/PID/maps + ptrace.

pip install py-spy
# (or: cargo install py-spy)
```

```bash
# Record an SVG directly (no folded step needed)
sudo py-spy record -o profile.svg --pid $PID --duration 30
```

```bash
# Record by launching
sudo py-spy record -o profile.svg -- python myapp.py
```

```bash
# Include subprocesses (e.g. multiprocessing, gunicorn workers)
sudo py-spy record -o profile.svg --pid $PID --subprocesses
```

```bash
# Include native (C) stacks for numpy/pandas/cython
sudo py-spy record -o profile.svg --pid $PID --native
```

```bash
# Live top-style view (no recording)
sudo py-spy top --pid $PID
```

```bash
# Dump current stacks of all threads (one-shot, no sampling)
sudo py-spy dump --pid $PID
```

```bash
# Output formats (besides SVG)
sudo py-spy record -o profile.json    --format speedscope --pid $PID
sudo py-spy record -o profile.folded  --format raw         --pid $PID  # folded
sudo py-spy record -o profile.svg     --format flamegraph  --pid $PID  # default
```

```bash
# Why ptrace? CPython's GIL means most threads aren't actively running;
# py-spy reads the interpreter state structure from another process's
# memory using ptrace, snapshotting where each Python thread is "stuck".
# This is why root (or CAP_SYS_PTRACE) is required.
```

```bash
# Alternative: austin (no ptrace required on Linux with kernel.yama.ptrace_scope)
pip install austin-python
austin -i 100 -o profile.austin python myapp.py
austin2speedscope profile.austin profile.json
```

```bash
# Note: pyflame is DEPRECATED. Don't use it for new work. py-spy is
# the modern replacement and supports CPython 3.6 - 3.12.
```

## Go Flamegraphs

```bash
# Native: go tool pprof. Built-in flamegraph view since Go 1.11.

# Add net/http/pprof endpoints to your app
# (in your main package, side-effect import)
# import _ "net/http/pprof"
# go func() { log.Fatal(http.ListenAndServe("localhost:6060", nil)) }()
```

```bash
# Capture a 30s CPU profile from a running app
curl -o cpu.pprof 'http://localhost:6060/debug/pprof/profile?seconds=30'
```

```bash
# Open the interactive web UI (includes a flamegraph view)
go tool pprof -http=:8080 cpu.pprof
# Then in the browser: View -> Flame Graph
```

```bash
# Alternative: command-line top
go tool pprof cpu.pprof
# (pprof) top10
# (pprof) list MyHotFunction
# (pprof) web   -- generates a callgraph SVG via graphviz
```

```bash
# Heap (allocations) flamegraph
curl -o heap.pprof http://localhost:6060/debug/pprof/heap
go tool pprof -http=:8080 heap.pprof
# Switch sample type with -alloc_space (total bytes allocated) or
# -inuse_space (currently held).
go tool pprof -alloc_space -http=:8080 heap.pprof
go tool pprof -inuse_space -http=:8080 heap.pprof
```

```bash
# Other live profiles
curl -o goroutine.pprof 'http://localhost:6060/debug/pprof/goroutine'
curl -o block.pprof    'http://localhost:6060/debug/pprof/block'
curl -o mutex.pprof    'http://localhost:6060/debug/pprof/mutex'
curl -o trace.bin      'http://localhost:6060/debug/pprof/trace?seconds=5'
go tool trace trace.bin
```

```bash
# Convert pprof to flamegraph.pl format (rare, but possible)
go tool pprof -raw -output=cpu.txt cpu.pprof
stackcollapse-go.pl cpu.txt | flamegraph.pl > go.svg
```

```bash
# Alternative viewer: speedscope
go tool pprof -output cpu.json -format=speedscope cpu.pprof
# (pprof has -format=speedscope only via the 3rd-party "github.com/google/pprof"
#  binary; for the stdlib version, use this instead:)
go install github.com/google/pprof@latest
pprof -output=cpu.json -format=speedscope cpu.pprof
# Then drag-and-drop cpu.json at https://www.speedscope.app/
```

## Rust Flamegraphs

```bash
# Idiomatic: cargo flamegraph (wraps perf on Linux, dtrace on macOS)
cargo install flamegraph
```

```bash
# Profile a binary
cd path/to/crate
cargo flamegraph --bin myapp
# Output: flamegraph.svg in current dir
```

```bash
# Profile a benchmark
cargo flamegraph --bench mybench
```

```bash
# Profile with arguments
cargo flamegraph --bin myapp -- --workload=heavy --threads=4
```

```bash
# Profile a release build (default), or dev
cargo flamegraph --dev --bin myapp   # debug build, slower
cargo flamegraph --bin myapp         # release (default)
```

```bash
# Important: enable frame pointers in Cargo.toml for accurate stacks
# [profile.release]
# debug = true
# # On stable, also:
# # RUSTFLAGS="-C force-frame-pointers=yes" cargo flamegraph
RUSTFLAGS="-C force-frame-pointers=yes" cargo flamegraph --bin myapp
```

```bash
# macOS uses dtrace (no perf). Requires running as root or with
# csrutil-disabled SIP for system-wide; for own processes is fine.
sudo cargo flamegraph --bin myapp
```

```bash
# Pure-Rust pipeline (no Perl): inferno
cargo install inferno
sudo perf record -F 99 -g -- ./target/release/myapp
sudo perf script | inferno-collapse-perf | inferno-flamegraph > flame.svg
# inferno is a drop-in replacement for stackcollapse-perf.pl + flamegraph.pl
# typically 5-10x faster, identical output
```

## Ruby Flamegraphs

```bash
# THE tool: rbspy. Same model as py-spy (ptrace + sampling).

cargo install rbspy
# or download a release binary from
#   https://github.com/rbspy/rbspy/releases
```

```bash
# Record by PID, output an SVG flamegraph
sudo rbspy record --pid $PID --duration 30 --format flamegraph -o flame.svg
```

```bash
# Record by launching
sudo rbspy record --format flamegraph -o flame.svg -- ruby myapp.rb
```

```bash
# Live top-style view
sudo rbspy snapshot --pid $PID
```

```bash
# Other formats
sudo rbspy record --format speedscope -o profile.json --pid $PID
sudo rbspy record --format summary    -o summary.txt  --pid $PID
sudo rbspy record --format callgrind  -o cg.out       --pid $PID
```

```bash
# Alternative: stackprof gem (in-process sampling)
gem install stackprof
# In your Ruby code:
#   StackProf.run(mode: :cpu, out: 'stackprof.dump') { run_workload }
stackprof stackprof.dump --text --limit 20
stackprof stackprof.dump --d3-flamegraph > flame.html
```

## PHP Flamegraphs

```bash
# Approach 1: XHProf (sampling, low overhead)
pecl install xhprof
# Enable in php.ini:
#   extension=xhprof.so
#   xhprof.output_dir="/tmp/xhprof"

# In your PHP code:
#   xhprof_enable(XHPROF_FLAGS_CPU + XHPROF_FLAGS_MEMORY);
#   run_workload();
#   $data = xhprof_disable();
#   file_put_contents("/tmp/xhprof/run.json", json_encode($data));

# Convert to folded then flamegraph
git clone https://github.com/whatsapp/xhprof
php xhprof/utils/xhprof_to_flamegraph.php /tmp/xhprof/run.json > run.folded
flamegraph.pl run.folded > xhprof.svg
```

```bash
# Approach 2: SPX (more modern, sampling + tracing)
pecl install spx
# In php.ini:
#   extension=spx.so
#   spx.http_enabled=1
#   spx.http_key="dev"
#   spx.http_ip_whitelist="127.0.0.1"

# Trigger by adding ?SPX_KEY=dev&SPX_UI_URI=/ to any URL
# Then visit /?SPX_UI_URI=/ for the report dashboard
```

```bash
# Approach 3: Tideways (commercial, production-grade)
# https://tideways.com/ — APM with built-in flamegraphs

# PHP-FPM caveat: each FPM worker is a separate process. Either
# use APM or attach a sampler to a specific worker PID.
```

## C/C++ Flamegraphs

```bash
# Build with frame pointers enabled (CRITICAL for accurate stacks)
gcc -O2 -fno-omit-frame-pointer -g myapp.c -o myapp
# (or for clang: -fno-omit-frame-pointer -gline-tables-only)
```

```bash
# Standard perf workflow
sudo perf record -F 99 -g -- ./myapp
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > app.svg
```

```bash
# When you can't rebuild — use DWARF unwinding
sudo perf record -F 99 -g --call-graph dwarf -p $PID -- sleep 30
# DWARF works without frame pointers but:
#  - perf.data is 5-10x larger
#  - script is slower (must reconstruct stacks from .eh_frame)
#  - max stack depth defaults to 8192 bytes — bump with --call-graph dwarf,16384
```

```bash
# Compile flags reference (gcc/clang)
gcc -O2 -fno-omit-frame-pointer -gdwarf-4 myapp.c     # frame pointers + DWARF
gcc -O2 -fno-omit-frame-pointer -g1     myapp.c     # frame pointers + minimal DI
gcc -O0 -g myapp.c                                  # debug build (slow but accurate)
```

```bash
# macOS uses dtrace; xctrace (Xcode 11+) replaces older Instruments CLI
xcrun xctrace record --template 'CPU Profiler' --launch -- ./myapp
# Outputs a .trace bundle. Open in Instruments. To get a flamegraph:
#  1. Open in Instruments
#  2. Select Time Profiler track
#  3. Edit -> Deep Copy on the call tree
#  4. pbpaste | stackcollapse-instruments.pl | flamegraph.pl > app.svg
```

```bash
# Stack symbol resolution issues
sudo perf script --no-inline       # disable inline expansion
sudo perf script --max-stack=128   # cap unwind depth
sudo perf record --buildid-all     # capture build-ids for later resolution
```

## Continuous Profiling — Pyroscope / Parca / Grafana Phlare

```bash
# Continuous profiling = always-on, low-overhead sampling stored in a
# time-series database. Compare flamegraphs at any two points in time
# without redeploying or running ad-hoc captures.

# Pyroscope (now part of Grafana, merged into Phlare)
docker run -it -p 4040:4040 grafana/pyroscope:latest
# Web UI at http://localhost:4040
```

```bash
# Pyroscope agent for Go
go get github.com/grafana/pyroscope-go
# In your code:
#   import "github.com/grafana/pyroscope-go"
#   pyroscope.Start(pyroscope.Config{
#     ApplicationName: "my.app",
#     ServerAddress:   "http://pyroscope:4040",
#     ProfileTypes:    pyroscope.DefaultProfileTypes,
#   })
```

```bash
# Pyroscope agent for Java
java -javaagent:pyroscope.jar=server=http://pyroscope:4040 \
     -Dpyroscope.application.name=my.app \
     -jar app.jar
```

```bash
# Parca — eBPF-based, unprivileged, whole-system
docker run -p 7070:7070 ghcr.io/parca-dev/parca:latest \
  /parca --config-path=/parca.yaml
# parca-agent runs on each host (DaemonSet on k8s)
docker run --privileged --pid=host \
  -v /sys:/sys -v /proc:/proc \
  ghcr.io/parca-dev/parca-agent:latest
```

```bash
# Grafana Phlare (now Pyroscope) — query via PromQL-like syntax
# https://pyroscope.example.com/
# Compare flamegraphs of "now vs 1 hour ago"
# Compare across deployments via labels
```

```bash
# The continuous profiling workflow
#   1. App emits profiles every 10-15s with labels (env, version, region)
#   2. Backend stores per-label time-series of folded stacks
#   3. UI lets you pick time range + label filter -> flamegraph
#   4. Diff mode lets you subtract two flamegraphs to find regressions
```

## Differential Flamegraphs

```bash
# Goal: see what changed between baseline and current.

# Step 1: capture baseline
sudo perf record -F 99 -a -g -o perf-before.data -- sleep 60
sudo perf script -i perf-before.data | stackcollapse-perf.pl > before.folded

# Step 2: deploy changes / change workload

# Step 3: capture current
sudo perf record -F 99 -a -g -o perf-after.data -- sleep 60
sudo perf script -i perf-after.data | stackcollapse-perf.pl > after.folded
```

```bash
# Method 1: difffolded.pl — produces a folded with per-stack delta counts
~/FlameGraph/difffolded.pl before.folded after.folded \
  | flamegraph.pl --title="Diff: before vs after" > diff.svg
```

```bash
# Method 2: --negate on flamegraph.pl directly
flamegraph.pl --negate before.folded after.folded > diff.svg
# Color reading after --negate:
#   red   = time REMOVED (got faster / called less)
#   blue  = time ADDED   (got slower / called more)
```

```bash
# Method 3: side-by-side — easier to eyeball
flamegraph.pl --title="Before" before.folded > before.svg
flamegraph.pl --title="After"  after.folded  > after.svg
# Open both; compare widths visually
```

```bash
# Pyroscope/Phlare diff mode
# In the web UI: pick two time ranges -> click "Diff" -> renders
# automatic differential flamegraph with red/green coloring.
```

## Heatmaps and FlameScope

```bash
# Problem: a flat 30s flamegraph hides BURSTS — a 1s spike of CPU
# 5s into the capture is averaged away.
# Solution: FlameScope renders a 2D heatmap of (time x flamegraph)
# so you can see exactly when the workload patterns happen.

git clone https://github.com/Netflix/flamescope ~/flamescope
cd ~/flamescope
pip install -r requirements.txt
python run.py
# Opens http://localhost:5000
```

```bash
# Drop perf script output into ~/flamescope/profiles/
sudo perf record -F 99 -a -g -- sleep 60
sudo perf script > ~/flamescope/profiles/run.txt
# Refresh the browser. Click the file -> heatmap view.
# X-axis: seconds elapsed. Y-axis: subsecond offset (default 20ms rows).
# Brightness: number of samples in that millisecond bucket.
# Drag to select a time region -> flamegraph for ONLY that window.
```

```bash
# Reading a heatmap:
#  - Steady bright band -> uniform load
#  - Bright spikes      -> bursts (drag-select for forensics)
#  - Dark gaps          -> idle / off-CPU
#  - Diagonal pattern   -> often timer-driven (GC, log flush)
```

## Reading Flamegraphs — What to Look For

```bash
# 1. PLATEAUS at the top
#    A wide flat top means CPU is spent IN that function's code
#    (high self-time). This is your hot leaf. Optimize the body.

# 2. PYRAMIDS (wide base, narrow top)
#    Self-time is concentrated at the leaf. Often you'll see a
#    single hot function fanned out across many call sites.

# 3. RECURSIVE TOWERS (same name stacked)
#    Recursive function. May be normal (parser, tree walk) or
#    accidental (forgot a memoization). Use stackcollapse-recursive.pl
#    to collapse identical adjacent frames.

# 4. INVERTED / ICICLE (--reverse)
#    Useful for "what calls X?" — the leaf is at the bottom, callers
#    fan upward. Click on a leaf to see who's invoking it.

# 5. SPIKES
#    Tall narrow stacks: rare deep paths. Often noise. Investigate
#    only if the deep path corresponds to something you care about.

# 6. THE BIGGEST PLATEAU IS YOUR BOTTLENECK
#    Optimize wide-but-shallow before deep-and-narrow. Width = time.
```

```bash
# Good things to do BEFORE looking:
#  - Know the workload (synthetic? prod? benchmark?)
#  - Know the duration (30s of warmup vs 30s of steady state matter)
#  - Know what "expected" looks like for this app
```

```bash
# Quick interpretation script for the impatient
flamegraph.pl --title="$(basename $PWD) — $(date +%F-%H%M)" \
  out.folded > flame.svg
open flame.svg
# Then: scan for the widest plateau at the top half of the graph
```

## Common Mistakes Reading Flamegraphs

```bash
# MISTAKE 1: Reading x-axis as time
# Flamegraphs sort frames ALPHABETICALLY at each level. The leftmost
# frame is NOT the first call. To see chronological order, use
# --flamechart on flamegraph.pl, or use a "flame chart" viewer
# (Chrome DevTools, speedscope's "Time Order" view).

# MISTAKE 2: Reading y-axis as time
# Y is depth (call hierarchy), not duration. A tall stack means deep
# calls; it does NOT mean it took a long time.

# MISTAKE 3: Ignoring [unknown] frames
# These are usually JIT'd code with no perf-PID.map. Fix the symbols
# (see JIT Map Files section) — those frames could be your real hotspot.

# MISTAKE 4: Profiling without frame pointers
# You'll see broken stacks: shallow plateaus, missing call hierarchy.
# Rebuild with -fno-omit-frame-pointer or use --call-graph=dwarf.

# MISTAKE 5: Sampling at 100 Hz
# 100 Hz aliases with the kernel timer interrupt and NTP, biasing your
# samples. Use 99 Hz. Always 99 Hz.

# MISTAKE 6: Profiling startup
# The first few seconds are loaders/JIT warmup, not steady-state.
# Wait until the workload is stable before recording.

# MISTAKE 7: One profile = truth
# Always capture 2-3 profiles to check consistency. Outliers happen.

# MISTAKE 8: Forgetting kernel frames
# Wide plateau labeled [kernel] often means I/O syscalls. Pair with
# off-CPU graph to see what's blocking.
```

## Common Errors and Fixes

```bash
# Error: "[unknown]" frames everywhere, or "[no stacks]"
# Cause: frame pointers missing OR JIT-map missing
# Fix:
#   - Rebuild with -fno-omit-frame-pointer
#   - Use --call-graph=dwarf
#   - For JIT runtimes, generate /tmp/perf-PID.map (see JIT section)
```

```bash
# Error: "perl: warning: Setting locale failed"
# Cause: missing locale on minimal containers
# Fix: harmless; set LC_ALL=C
export LC_ALL=C
flamegraph.pl out.folded > flame.svg
```

```bash
# Error: "Use of uninitialized value $x in numeric comparison"
# Cause: malformed folded format (line missing sample count, or
#        unexpected characters in stack)
# Fix:
awk 'NF<2 {print "BAD:", NR, $0}' out.folded   # find bad lines
# Common culprits:
#   - extra blank lines at end of file
#   - "+" or "-" in symbol names that confuse stackcollapse-perf.pl
#   - non-UTF-8 bytes in symbols
```

```bash
# Error: Wide bar at top of every stack labeled [kernel]
# Cause: kernel symbols not readable
# Fix:
sudo sysctl kernel.kptr_restrict=0
sudo sysctl kernel.perf_event_paranoid=-1
ls /proc/kallsyms                         # should be readable
sudo perf record -F 99 -g -- sleep 30     # re-record
```

```bash
# Error: perf record fails with "Permission denied"
# Cause: paranoid kernel default
# Fix:
sudo sysctl kernel.perf_event_paranoid=1   # or =-1 for full access
# Or run perf as root
```

```bash
# Error: "Cannot find .map file" / hex addresses in JIT stacks
# Cause: process exited before perf script ran (map file is per-PID)
# Fix: either copy /tmp/perf-PID.map BEFORE process exits, or
#      symbolize at record time:
sudo perf record -F 99 -p $PID -g -- sleep 30
sudo cp /tmp/perf-$PID.map ./perf-saved.map
# ... process can exit now ...
sudo perf script --symfs=. > out.perf
```

```bash
# Error: stacks are 2 frames deep — clearly wrong
# Cause: built with -fomit-frame-pointer (default at -O2 on many distros)
# Fix:
gcc -O2 -fno-omit-frame-pointer ...        # rebuild
# or:
sudo perf record --call-graph dwarf ...    # use DWARF unwinding
# or:
sudo perf record --call-graph lbr ...      # Intel LBR (limited depth)
```

```bash
# Error: "no symbol found for IP 0x..."
# Cause: stripped binaries, or debuginfo not installed
# Fix:
debuginfo-install $(rpm -qf /path/to/binary)   # RHEL/CentOS
apt install $(dpkg -S /path/to/binary | cut -d: -f1)-dbgsym  # Debian
```

```bash
# Error: SVG renders blank or misaligned in browser
# Cause: missing minwidth, very small stacks rendered sub-pixel
# Fix:
flamegraph.pl --minwidth=0.5 out.folded > flame.svg
```

## Performance Cost

```bash
# Profiling overhead (typical, on x86_64 Linux):
#
# perf at 99 Hz, frame pointers:        ~1-2% throughput overhead
# perf at 99 Hz, --call-graph dwarf:    ~3-5% (more I/O, more CPU)
# perf at 999 Hz, frame pointers:       ~5-10%
# perf at 4999 Hz, frame pointers:      ~25-40%  (don't do this)
#
# bpftrace at 99 Hz with stack capture:  ~1-3%
# bpftrace off-CPU (kprobes):            ~2-5% under heavy contention
#
# py-spy ptrace sampling:                ~1-2%
# rbspy:                                 ~1-2%
# async-profiler CPU:                    ~1-3%
# async-profiler alloc (every alloc):    ~5-15%
#
# go tool pprof CPU:                     ~5%  (built-in setitimer)
# go tool pprof block / mutex:           depends on rate setting
#
# Pyroscope continuous (10s every 15s):  ~0.5-1% steady state
# Parca eBPF:                            ~0.5-1.5%
```

```bash
# Rule of thumb: 99 Hz is safe for production on busy machines.
# If you go above 999 Hz, run on a canary host first.
```

```bash
# Reduce overhead further:
sudo perf record -F 49 -g -- sleep 60         # half the rate, longer window
sudo perf record -F 99 -p $PID -g -- sleep 30 # one process, not -a
```

## Best Practices

```bash
# 1. Profile in production traffic, not synthetic.
#    Synthetic tests miss real-world patterns (cache effects, contention).

# 2. Sample 30-60s for steady-state apps.
#    Longer (5-10 min) for bursty workloads.

# 3. Always capture multiple profiles.
#    Compare them. One profile can be misleading; three tell a story.

# 4. Archive baselines BEFORE optimizing.
#    Save before.folded so you can do a differential after.

# 5. Use 99 Hz, not 100 Hz.
#    Avoids aliasing with kernel timers.

# 6. Build with -fno-omit-frame-pointer for production binaries.
#    Cost: 0-3% throughput. Benefit: cheap, accurate profiling forever.

# 7. Tag profiles with metadata.
#    Include date, version, host, workload in --title and --subtitle.

# 8. Capture both CPU and off-CPU.
#    On-CPU answers "what's hot?". Off-CPU answers "what's blocked?".

# 9. Keep folded files. They are tiny (~1MB) and let you re-render
#    or differentiate later without re-profiling.

# 10. Use continuous profiling for prod. Ad-hoc profiling for dev.
```

```bash
# A reusable "profile this" script
profile_pid() {
  local pid=$1
  local dur=${2:-30}
  local out=${3:-flame-$(date +%Y%m%d-%H%M%S)}
  sudo perf record -F 99 -p "$pid" -g -o "$out.data" -- sleep "$dur"
  sudo perf script -i "$out.data" > "$out.perf"
  stackcollapse-perf.pl "$out.perf" > "$out.folded"
  flamegraph.pl --title="PID $pid, ${dur}s, $(date +%F)" \
    "$out.folded" > "$out.svg"
  echo "Wrote $out.svg"
}
profile_pid 12345 30
```

## Common Patterns Discovered

```bash
# PATTERN: JSON encoding hot path
#   Symptom: wide plateau on json.Marshal / encoding/json.encode
#   Fix:    pre-allocate buffers, use json.Encoder with bufio.Writer,
#           consider easyjson / jsoniter / sonic for hot paths
```

```bash
# PATTERN: Regex compiled per-request
#   Symptom: wide plateau on regexp.Compile / regex.compile inside
#            hot request handler
#   Fix:    cache compiled regexes at package init, never inside loop
```

```bash
# PATTERN: Memory allocation hot path
#   Symptom: wide plateau on runtime.mallocgc (Go), GC_collect (Python),
#            G1ParScanThreadState (Java)
#   Fix:    sync.Pool / object pooling, slice reuse, avoid string
#           concatenation, prefer Sprintf -> Buffer.WriteString
```

```bash
# PATTERN: Lock contention plateau
#   Symptom: wide plateau on futex_wait / pthread_mutex_lock /
#            sync.Mutex.Lock — visible only in OFF-CPU flamegraph
#   Fix:    shard the lock (one per worker), use lock-free structures,
#           replace mutex with atomic for simple counters
```

```bash
# PATTERN: Logging in hot path
#   Symptom: wide plateau under log.Printf / fmt.Sprintf chains
#   Fix:    structured logging with lazy serialization (zerolog/zap),
#           level-gate before format, drop debug logs in prod
```

```bash
# PATTERN: TLS handshake overhead
#   Symptom: wide plateau on crypto/tls.handshake / SSL_do_handshake
#   Fix:    enable session resumption, keep-alive, HTTP/2, use OCSP stapling
```

```bash
# PATTERN: Context-switch storm
#   Symptom: wide [kernel] plateau on schedule / __schedule, low %user
#   Fix:    reduce thread count, batch work, use epoll/kqueue,
#           pin worker threads to cores
```

```bash
# PATTERN: Excessive syscall traffic
#   Symptom: wide [kernel] plateau on syscall entries — read/write/futex
#   Fix:    batch syscalls (writev), increase buffer sizes,
#           use io_uring (Linux 5.6+) for high-throughput I/O
```

## Idioms

```bash
# IDIOM: The canonical one-liner
sudo perf record -F 99 -a -g -- sleep 30 \
  && sudo perf script \
  | stackcollapse-perf.pl \
  | flamegraph.pl > flame.svg
```

```bash
# IDIOM: Drop a folded file at speedscope.app
# 1. Create out.folded by any means
# 2. Open https://www.speedscope.app/ in browser
# 3. Drag-drop the file. Interactive flamegraph + sandwich + time-order
#    views. Click any frame to drill down.
```

```bash
# IDIOM: The before-after differential
# 1. Capture baseline:    out-before.folded
# 2. Apply optimization
# 3. Capture current:     out-after.folded
# 4. Render diff:
flamegraph.pl --negate out-before.folded out-after.folded > diff.svg
# Red = removed time (success). Blue = added time (regression).
```

```bash
# IDIOM: Profile a transient command end-to-end
sudo perf record -F 99 -g -- ./target/release/long-running-job
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > job.svg
```

```bash
# IDIOM: Live attach to a process you didn't launch
sudo perf record -F 99 -p $(pgrep -f 'my-binary') -g -- sleep 30
```

```bash
# IDIOM: Rapid filter — show stacks containing X
grep 'parse_json' out.folded | flamegraph.pl > parse-only.svg
```

```bash
# IDIOM: Strip kernel frames
grep -v ';\\[kernel\\]' out.folded | flamegraph.pl > userspace.svg
```

```bash
# IDIOM: Strip everything ABOVE a frame (find leaves under foo)
grep -oE '.*foo[^;]*' out.folded | flamegraph.pl > under-foo.svg
```

```bash
# IDIOM: Multiple captures, single graph (concat folded files)
cat run1.folded run2.folded run3.folded \
  | sort | uniq -c \
  | awk '{count=$1; $1=""; print substr($0,2)" "count}' \
  | flamegraph.pl > combined.svg
```

```bash
# IDIOM: Inferno (Rust drop-in) for big traces
sudo perf record -F 99 -a -g -- sleep 30
sudo perf script | inferno-collapse-perf | inferno-flamegraph > flame.svg
```

## Tips

```bash
# - Always record with -g (call graphs); without it perf produces flat
#   profiles with no hierarchy and flamegraphs become useless.
#
# - Frame pointers (-fno-omit-frame-pointer) are the cheap, accurate
#   default. DWARF unwinding works without them but is slower.
#
# - 99 Hz is the magic number. 100 Hz aliases with NTP/timer ticks.
#
# - Off-CPU graphs reveal lock contention and I/O waits invisible
#   in CPU graphs.
#
# - Differential graphs are the fastest way to find regressions.
#   Always keep a baseline.folded around.
#
# - Filter stacks with grep BEFORE flamegraph.pl to focus subsystems.
#
# - Use --reverse for icicle (root at top) when you want "callers of X".
#
# - x-axis is NEVER time. Use --flamechart or speedscope's time-order
#   view for chronological data.
#
# - For Go, prefer "go tool pprof -http" — built-in, no Perl needed.
#
# - For Java, use async-profiler. JFR has safepoint bias, jstack is
#   too slow.
#
# - Memory flamegraphs distinguish allocation site (where created)
#   from retention site (where kept alive). Use both.
#
# - Continuous profiling (Pyroscope/Parca) supersedes ad-hoc captures
#   for production triage.
#
# - speedscope.app is the easiest interactive viewer. No install,
#   drag-drop folded or pprof or chrome JSON.
#
# - In containers, perf may need:
#     --cap-add=SYS_ADMIN --cap-add=SYS_PTRACE
#   and bind-mount /tmp for perf-PID.map.
#
# - macOS replaces perf with dtrace / xctrace (Instruments). Use
#   stackcollapse-instruments.pl on the deep-copy output.
#
# - On ARM, frame pointers are mandatory for accurate stacks; DWARF
#   unwinding on ARM is buggier than on x86.
#
# - Don't profile during JIT warmup. Wait for steady state.
#
# - Capture metadata: hostname, version, workload — embed in --title
#   so you remember what each SVG was 6 months later.
```

## Quick Reference Cheatsheet

```bash
# THE 5 COMMANDS YOU'LL ACTUALLY USE
# ----------------------------------
# 1. CPU flamegraph of a running process
sudo perf record -F 99 -p $PID -g -- sleep 30
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > cpu.svg

# 2. CPU flamegraph of whole system
sudo perf record -F 99 -a -g -- sleep 30
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > sys.svg

# 3. Python instant flamegraph
sudo py-spy record -o py.svg --pid $PID --duration 30

# 4. Go from running app
curl -o cpu.pprof 'http://localhost:6060/debug/pprof/profile?seconds=30'
go tool pprof -http=:8080 cpu.pprof

# 5. Java with async-profiler
./asprof -d 30 -f java.html $JAVA_PID
```

```bash
# TROUBLESHOOTING DECISION TREE
# -----------------------------
# Q: stacks all show [unknown] / hex
# A: missing frame pointers OR missing perf-PID.map (JIT)
#    -> rebuild with -fno-omit-frame-pointer
#    -> or use --call-graph dwarf
#    -> for JIT: enable language-specific perf map (Node --perf-prof,
#       Java perf-map-agent or async-profiler)

# Q: stacks max 2 frames deep
# A: same as above, frame pointers omitted

# Q: kernel frames missing
# A: kernel.kptr_restrict=0 + kernel.perf_event_paranoid=-1

# Q: graph is flat (no hierarchy)
# A: forgot -g on perf record

# Q: cannot capture, "permission denied"
# A: kernel.perf_event_paranoid; run as root, or set sysctl

# Q: macOS has no perf
# A: use dtrace via Xcode Instruments / xctrace, or cargo flamegraph
#    which already wraps both
```

## Compiler Flags Reference

```bash
# C / C++ (gcc, clang) — ESSENTIAL for accurate flamegraphs
-O2 -fno-omit-frame-pointer    # the canonical pair
-g                             # symbols
-gdwarf-4                      # DWARF (for --call-graph dwarf)
-gline-tables-only             # minimal debuginfo (cheap)
-fno-inline                    # disable inlining (more frames, slower)
-mno-omit-leaf-frame-pointer   # keep FP even for leaf functions
```

```bash
# Rust (rustc) — pass via RUSTFLAGS
RUSTFLAGS="-C force-frame-pointers=yes -C debuginfo=2"
# Cargo.toml
# [profile.release]
# debug = true              # symbols in release
# split-debuginfo = "packed"  # smaller binary, separate dSYM/DWO
```

```bash
# Go — frame pointers are ALWAYS on (since Go 1.7 on amd64).
# No flag needed. Just build and capture.
go build -o app .
go build -gcflags="all=-N -l" -o app .   # disable optimizations + inlining
```

```bash
# Java — JVM flags
-XX:+PreserveFramePointers   # required for perf-based capture
-XX:+UnlockDiagnosticVMOptions
-XX:+DebugNonSafepoints      # enables async-profiler safepoint-bias-free capture
```

```bash
# Node.js — runtime flags
node --perf-basic-prof              # JS-only frames (cleaner)
node --perf-basic-prof-only-functions
node --perf-prof                    # JIT'd code, more detail
node --interpreted-frames-native-stack
```

```bash
# Python — runtime / interpreter
# (none needed for py-spy; it samples without cooperation)
PYTHONDONTWRITEBYTECODE=1   # avoid .pyc churn during profile
```

```bash
# Ruby — runtime
# (none needed for rbspy; it samples without cooperation)
```

## Output Format Compatibility

```bash
# Folded (the canonical exchange format)
#   stack;is;here count
#   ...one line per unique stack
flamegraph.pl, inferno-flamegraph, speedscope.app, FlameScope

# pprof (Google's profile.proto, gzipped binary)
#   .pprof / .pb.gz
go tool pprof, pprof CLI, speedscope.app, Pyroscope

# Speedscope JSON
#   { "$schema": "...", "profiles": [...] }
speedscope.app, Chrome DevTools (some), FlameScope (some)

# Chrome DevTools / V8 .cpuprofile
#   { "head": {...}, "samples": [...] }
Chrome DevTools, speedscope.app, stackcollapse-chrome.pl

# Linux perf.data (binary, perf-specific)
#   perf record output
perf script (then convert), Hotspot KDE GUI, perf report

# DTrace / SystemTap text
#   stack traces interleaved with counts
stackcollapse-stap.pl, stackcollapse for dtrace formats
```

```bash
# Conversion table (rough)
# perf script  -> folded:        stackcollapse-perf.pl
# bpftrace     -> folded:        stackcollapse-bpftrace.pl
# Go pprof     -> folded:        go tool pprof -raw | stackcollapse-go.pl
# Go pprof     -> speedscope:    pprof -format=speedscope
# Java jstack  -> folded:        stackcollapse-jstack.pl
# Java JFR     -> speedscope:    converter (jfr-flame-graph)
# Chrome JSON  -> folded:        stackcollapse-chrome.pl
# Instruments  -> folded:        stackcollapse-instruments.pl (deep copy)
# Folded       -> SVG:           flamegraph.pl  /  inferno-flamegraph
# Folded       -> speedscope:    speedscope CLI:  speedscope < file.folded
```

## Container and Kubernetes Notes

```bash
# Docker — containers usually need extra capabilities for perf
docker run --cap-add=SYS_ADMIN --cap-add=SYS_PTRACE \
           --security-opt=seccomp=unconfined \
           --pid=host \
           myimage perf record -F 99 -a -g -- sleep 30
```

```bash
# Or run perf on the host, target the container's PID
PID=$(docker inspect --format '{{.State.Pid}}' my-container)
sudo perf record -F 99 -p $PID -g -- sleep 30
```

```bash
# Kubernetes — sidecar approach
# Add a sidecar with privileged: true, then perf record against the
# main container's PID 1 (visible because they share the pod's PID
# namespace if shareProcessNamespace: true is set on the pod).

cat <<'EOF'
spec:
  shareProcessNamespace: true
  containers:
  - name: app
    image: myapp:latest
  - name: profiler
    image: brendangregg/perf-tools
    securityContext:
      privileged: true
    command: ["sleep", "infinity"]
EOF
# Then: kubectl exec -it pod -c profiler -- perf record ...
```

```bash
# Parca-Agent — runs as DaemonSet, profiles everything via eBPF
kubectl apply -f https://github.com/parca-dev/parca/releases/latest/download/parca-agent.yaml
```

```bash
# Pyroscope — language SDKs work in containers without privileges
# (the SDKs use in-process profilers — no perf, no eBPF)
```

## Sampling Math — Why 99 Hz

```bash
# 100 Hz problem: the kernel's HZ value (CONFIG_HZ) is often 100, 250,
# or 1000. If your sampling rate is a divisor of HZ, you sample at the
# same phase as the timer interrupt and miss intervals between ticks.
#
# 99 Hz is coprime with most HZ values (100, 250, 1000), so samples
# distribute evenly across the timer cycle.

# Brendan Gregg's rule: never use 100 / 1000 / 250 Hz for sampling.
# Use 99 / 999 / 49 / 49.5 Hz instead.

# Concrete example:
#   HZ=1000 (kernel ticks every 1ms)
#   Sample at 1000 Hz -> aligned with ticks -> bias toward functions
#     that happen to be running at tick time
#   Sample at 999 Hz  -> drifts through the cycle -> unbiased coverage
```

## See Also

- perf
- bpftrace
- bpftool
- ebpf
- polyglot
- bash

## References

- [FlameGraph repository — Brendan Gregg](https://github.com/brendangregg/FlameGraph)
- [Flame Graphs — brendangregg.com](https://www.brendangregg.com/flamegraphs.html)
- [Off-CPU Flame Graphs — brendangregg.com](https://www.brendangregg.com/offcpuanalysis.html)
- [The Flame Graph (ACM Queue, 2016) — Brendan Gregg](https://queue.acm.org/detail.cfm?id=2927301)
- [async-profiler GitHub](https://github.com/async-profiler/async-profiler)
- [py-spy GitHub](https://github.com/benfred/py-spy)
- [rbspy GitHub](https://github.com/rbspy/rbspy)
- [google/pprof](https://github.com/google/pprof)
- [speedscope.app](https://www.speedscope.app/)
- [Pyroscope / Grafana Phlare](https://grafana.com/oss/pyroscope/)
- [Parca — eBPF continuous profiling](https://www.parca.dev/)
- [FlameScope — Netflix](https://github.com/Netflix/flamescope)
- [inferno — Rust port of FlameGraph](https://github.com/jonhoo/inferno)
- [Linux perf wiki](https://perf.wiki.kernel.org/index.php/Main_Page)
- [bpftrace reference guide](https://github.com/bpftrace/bpftrace/blob/master/docs/reference_guide.md)
- "Systems Performance: Enterprise and the Cloud" — Brendan Gregg, 2nd ed., Pearson, 2020
- "BPF Performance Tools" — Brendan Gregg, Pearson, 2019
