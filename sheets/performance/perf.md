# perf (Linux Performance Analysis)

The canonical Linux profiler — counter sampling, event tracing, call-graph profiling, dynamic probes, and microarchitecture analysis, all driven by the kernel's `perf_events` subsystem.

## Setup

`perf` ships as a userspace tool tightly coupled to the running kernel. The `perf` binary version must match the kernel version, otherwise event lists, ABI shapes, and tracepoint names drift out of sync.

### Debian / Ubuntu

```bash
sudo apt-get update
sudo apt-get install linux-tools-common linux-tools-generic
sudo apt-get install linux-tools-$(uname -r)
which perf
perf --version
```

The `linux-tools-common` package only provides the wrapper `/usr/bin/perf` which dispatches to the kernel-version-specific binary. Without `linux-tools-$(uname -r)` installed, the wrapper prints:

```bash
WARNING: perf not found for kernel 6.5.0-26-generic
  You may need to install the following packages for this specific kernel:
    linux-tools-6.5.0-26-generic
    linux-cloud-tools-6.5.0-26-generic
```

This is the most common gotcha — installing `linux-tools-generic` is necessary but not sufficient.

### Fedora / RHEL / CentOS / Rocky / Alma

```bash
sudo dnf install perf
perf --version
```

The Fedora package is a single `perf-tools` (sometimes just `perf`) that always tracks the running kernel.

### Arch / Manjaro

```bash
sudo pacman -S perf
perf --version
```

### Alpine

```bash
sudo apk add perf
```

### From source (latest kernel tree)

```bash
git clone --depth 1 https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git
cd linux/tools/perf
make -j$(nproc)
sudo cp perf /usr/local/bin/perf
```

Building from source is required if you need a feature that landed in mainline but hasn't shipped in your distro's `linux-tools` yet (e.g. recent `--topdown` levels, new tracepoints, BTF support).

### macOS — perf does not exist

Linux `perf` requires the `perf_events` kernel subsystem. macOS has no equivalent. Alternatives on macOS:

```bash
sudo dtrace -n 'profile-99 /pid == $target/ { @[ustack()] = count(); }' -p PID
sample PID 5 -file /tmp/sample.txt
xctrace record --template "Time Profiler" --launch -- ./myapp
instruments -t "Time Profiler" ./myapp
```

For Linux-on-Mac profiling, run `perf` inside a Linux VM (UTM, Lima, Multipass, OrbStack) or a container with `--privileged` and `--cap-add=PERFMON`.

### Verifying perf works

```bash
perf list | head
perf stat -e cycles -- sleep 1
sudo perf top
```

If `perf list` is empty and `perf stat` reports `<not supported>` for everything, the kernel was built without `CONFIG_PERF_EVENTS=y`. Check with `zgrep PERF_EVENTS /proc/config.gz` or `grep PERF_EVENTS /boot/config-$(uname -r)`.

### Permissions: perf_event_paranoid

```bash
cat /proc/sys/kernel/perf_event_paranoid
```

Values:

```bash
# -1 = allow everything (development boxes)
#  0 = allow access to CPU + kernel measurements (root not required for system-wide)
#  1 = disallow CPU events for unpriv users (default on most distros pre-6.6)
#  2 = disallow kernel profiling for unpriv users (default on Debian/Ubuntu)
#  3 = disallow user profiling for unpriv users (some hardened distros)
#  4 = additionally disallow raw tracepoint access (newest)
```

To loosen for an interactive session:

```bash
sudo sysctl -w kernel.perf_event_paranoid=-1
```

To persist:

```bash
echo 'kernel.perf_event_paranoid = -1' | sudo tee /etc/sysctl.d/99-perf.conf
sudo sysctl --system
```

The modern alternative is `CAP_PERFMON` (added in 5.8), narrower than `CAP_SYS_ADMIN`:

```bash
sudo setcap cap_perfmon,cap_sys_ptrace+ep $(which perf)
```

### Permissions: kptr_restrict

```bash
cat /proc/sys/kernel/kptr_restrict
sudo sysctl -w kernel.kptr_restrict=0
```

When set to 1 or 2, `/proc/kallsyms` returns zeroed addresses to non-root users, so `perf report` shows kernel functions as `[unknown]`.

## perf Subcommands Catalog

```bash
perf --help
perf list-cmds
```

The full subcommand catalog (Linux 6.x):

```bash
# Core profiling
perf stat              # event counter aggregation
perf record            # sample-based profiling, writes perf.data
perf report            # interactive analysis of perf.data
perf top               # live system-wide profiling (top-style)
perf annotate          # source/asm annotation of hot functions
perf script            # dump samples as text or run scripts on them
perf list              # enumerate all available events
perf archive           # tar perf.data + symbols for cross-machine analysis

# Specialized analyzers
perf bench             # built-in micro-benchmarks
perf c2c               # cache-line contention / false sharing analyzer
perf data              # convert perf.data to/from CTF, JSON, etc.
perf diff              # diff two perf.data files
perf evlist            # list events recorded in a perf.data
perf ftrace            # frontend to kernel ftrace
perf inject            # post-process perf.data (build-ids, jitdump)
perf kallsyms          # show kernel symbol resolution
perf kmem              # kernel slab allocator analyzer
perf kvm               # KVM guest profiling from host
perf lock              # kernel lock contention analyzer
perf mem               # memory load/store sampling (PEBS)
perf probe             # add dynamic kernel/userspace probes
perf sched             # scheduler timing/latency analyzer
perf test              # built-in self-test suite
perf timechart         # generate Gantt-chart SVG of system activity
perf trace             # strace-like syscall tracer (lower overhead)

# Configuration / cache
perf buildid-cache     # manage ~/.debug build-id symbol cache
perf buildid-list      # list build-ids in perf.data
perf config            # read/write ~/.perfconfig
perf daemon            # run perf record sessions as a service
```

Most subcommands take `-h` for short help and have a man page (`man perf-record`, `man perf-stat`, etc).

## perf stat — Counter Sampling

`perf stat` is the simplest profiler: it runs (or attaches to) a workload, programs hardware/software counters, and prints aggregate counts when the workload exits. Zero call-stack data — pure counter sums.

### Default output

```bash
perf stat -- ls
```

Default counter set (varies by kernel/CPU):

```bash
 Performance counter stats for 'ls':

              0.46 msec task-clock                #    0.711 CPUs utilized
                 1      context-switches          #    0.002 M/sec
                 0      cpu-migrations            #    0.000 K/sec
               101      page-faults               #    0.220 M/sec
         1,432,051      cycles                    #    3.117 GHz
         1,802,184      instructions              #    1.26  insn per cycle
           360,418      branches                  #  784.500 M/sec
            10,253      branch-misses             #    2.85% of all branches

       0.000647412 seconds time elapsed
       0.000718000 seconds user
       0.000000000 seconds sys
```

The `insn per cycle` (IPC) is the headline metric — modern x86 cores can retire 4-6 IPC at peak. Below 1.0 means significant stalling.

### Counting a long-running workload

```bash
perf stat -- ./batch_job input.csv
perf stat ./batch_job input.csv      # same; -- is optional
```

### Attaching to a running PID

```bash
perf stat -p 12345
perf stat -p 12345 -- sleep 30      # attach for exactly 30s
```

Ctrl-C ends the count and prints aggregates.

### System-wide

```bash
sudo perf stat -a -- sleep 5
```

Counts events across every CPU. Requires `CAP_PERFMON` or `perf_event_paranoid <= 0`.

## perf stat — Common Flags

```bash
-e EVENT[,EVENT,...]     # explicit event list (overrides default)
-p PID                   # attach to running process
-t TID                   # attach to specific thread
-a                       # system-wide across all CPUs
-A                       # don't aggregate per CPU (--no-aggr)
-C 0,1,2-4               # restrict to specific CPUs
-d                       # detailed (adds L1, LLC, dTLB)
-dd                      # more detail (adds frontend/backend stalls)
-ddd                     # even more detail (adds L2, page walk cycles)
--topdown                # Top-Down Microarchitecture Analysis
--td-level 2             # topdown level 2 (more granular)
-r N                     # repeat N times, compute mean ± stdev
-R                       # raw event values
-x ,                     # CSV output (separator)
--json-output            # JSON output (recent perf)
-o file                  # output file
--append                 # append to output file
-I MS                    # interval print every MS milliseconds
-G CGROUP                # cgroup-scoped counting
--per-socket             # aggregate per socket
--per-core               # aggregate per physical core
--per-thread             # aggregate per thread
--per-die                # aggregate per die (multi-die CPUs)
--for-each-cgroup foo,bar # multi-cgroup compare
--summary                # print global summary at end
--no-csv-summary         # for CSV mode
--big-num                # 1,234,567 instead of 1234567
-B                       # synonym for --big-num
-n                       # null run, just measure overhead
-S                       # synthesize counts (rare)
-v                       # verbose
--all-user               # only count user-space events
--all-kernel             # only count kernel events
--metric-only            # print only metrics, not raw counts
--metric-no-group        # don't group when measuring metrics
```

### Repeat for variance

```bash
perf stat -r 10 -- ./bench
```

Output shows `+- N%` standard deviation. Useful for "did my optimization actually move the needle".

### Interval mode

```bash
perf stat -I 1000 -e cycles,instructions -a
```

Prints counts every 1000 ms (1 second). Useful for time-series profiling.

### CSV / JSON output for piping

```bash
perf stat -x , -e cycles,instructions -- ./bench 2>perf.csv
perf stat --json-output -e cycles,instructions -- ./bench 2>perf.json
```

### Cgroup scoping

```bash
sudo perf stat -G my_cgroup -e cycles,instructions -a -- sleep 10
```

Counts only events from tasks in the specified cgroup. Useful for container profiling — pass the container's cgroup path.

### Microarch detail levels

```bash
perf stat -d -- ./bench         # adds L1-dcache-loads, L1-dcache-load-misses, LLC-loads, LLC-load-misses, dTLB-loads, dTLB-load-misses
perf stat -dd -- ./bench        # also adds L1-icache-load-misses, iTLB-load-misses
perf stat -ddd -- ./bench       # also adds L2 + node misses
```

## perf list — Event Catalog

```bash
perf list
perf list hw                  # hardware events only
perf list sw                  # software events only
perf list cache               # cache events
perf list tracepoint          # static kernel tracepoints
perf list pmu                 # raw PMU events for this CPU
perf list metric              # named metrics (IPC, CPI, etc)
perf list sdt                 # USDT (User-Statically-Defined Tracing) probes
perf list 'sched:*'           # glob filtering
```

### Hardware events (CPU PMU)

```bash
cpu-cycles  OR  cycles                # core clock cycles
instructions                          # instructions retired
cache-references                      # last-level cache accesses
cache-misses                          # last-level cache misses
branch-instructions OR branches       # branches executed
branch-misses                         # branch mispredicts
bus-cycles                            # bus clock cycles
stalled-cycles-frontend               # frontend stalls (decode-bound)
stalled-cycles-backend                # backend stalls (memory/exec-bound)
ref-cycles                            # cycles ignoring DVFS (constant freq reference)
```

### Software events (kernel-emulated)

```bash
cpu-clock              # high-res CPU clock (per-CPU)
task-clock             # task-attached CPU clock (per-task)
page-faults  OR  faults
context-switches  OR  cs
cpu-migrations  OR  migrations
minor-faults
major-faults
alignment-faults
emulation-faults
dummy                  # placeholder, useful as a side-band channel
bpf-output             # BPF perf-event-array writes
cgroup-switches        # cgroup-scoped context switches
```

### Cache events

Generic shape: `<level>-<type>-<op>-<result>`:

```bash
L1-dcache-loads
L1-dcache-load-misses
L1-dcache-stores
L1-dcache-store-misses
L1-icache-loads
L1-icache-load-misses
LLC-loads             # last-level cache (L2 or L3 depending on CPU)
LLC-load-misses
LLC-stores
LLC-store-misses
dTLB-loads
dTLB-load-misses
dTLB-stores
dTLB-store-misses
iTLB-loads
iTLB-load-misses
branch-loads
branch-load-misses
node-loads            # NUMA node-local vs remote
node-load-misses
```

### Tracepoints

```bash
perf list 'sched:*'
perf list 'syscalls:*'
perf list 'block:*'
perf list 'net:*'
perf list 'kvm:*'
perf list 'irq:*'
perf list 'timer:*'
perf list 'workqueue:*'
perf list 'writeback:*'
```

Hot examples:

```bash
sched:sched_switch           # context switch
sched:sched_wakeup
sched:sched_stat_runtime
sched:sched_stat_iowait
sched:sched_stat_sleep
sched:sched_process_exec
syscalls:sys_enter_openat
syscalls:sys_exit_openat
block:block_rq_issue
block:block_rq_complete
net:net_dev_xmit
net:netif_receive_skb
kvm:kvm_entry
kvm:kvm_exit
irq:softirq_entry
```

### Vendor-specific PMU events

```bash
perf list pmu | head -50
```

Intel Skylake example:

```bash
cpu/event=0xc0,umask=0x00/        # raw INST_RETIRED.ANY
cpu/event=0xc4,umask=0x20/        # BR_INST_RETIRED.NEAR_TAKEN
cpu/mem-loads,ldlat=30/           # PEBS load-latency >= 30 cycles
```

You can name them inline:

```bash
perf stat -e 'cpu/event=0xc0,umask=0x00,name=insns_alt/' -- ./bench
```

### Dynamic probes appear in `perf list` after creation

```bash
sudo perf probe --add tcp_sendmsg
perf list 'probe:*'
```

## perf record — Profiling

`perf record` samples events and writes `perf.data` for offline analysis. Where `stat` aggregates, `record` keeps per-sample detail (CPU, time, IP, callstack, raw event data).

### Basic recording

```bash
perf record -- ./myapp
perf record -p 12345 -- sleep 30
sudo perf record -a -- sleep 10
```

The `--` separates perf flags from the workload command and its args.

### Frequency sampling

```bash
perf record -F 99 -- ./myapp
perf record -F 999 -- ./myapp
perf record -F 4000 -p 12345 -- sleep 5     # very high; expensive
```

`-F HZ` samples roughly HZ times per second. `99` instead of `100` avoids aliasing with NTP-synced 100Hz timer ticks. `997` and `1999` are similarly prime-flavored choices.

### Period sampling

```bash
perf record -c 1000000 -e cycles -- ./myapp
```

`-c COUNT` triggers a sample every COUNT events. Deterministic, but workload-dependent rate.

### Specific events

```bash
perf record -e cache-misses -- ./myapp
perf record -e 'cycles,instructions,cache-misses,branch-misses' -- ./myapp
perf record -e 'syscalls:sys_enter_openat' -a -- sleep 10
perf record -e 'sched:sched_switch' -a -- sleep 5
```

### Call graphs

```bash
perf record -g -- ./myapp                    # default call-graph mode (fp on most distros)
perf record --call-graph=fp -- ./myapp       # frame-pointer (fastest, needs -fno-omit-frame-pointer)
perf record --call-graph=dwarf -- ./myapp    # DWARF-based unwinding (best, needs debuginfo, slow)
perf record --call-graph=dwarf,16384 -- ./myapp  # DWARF with 16KB stack capture (default 8KB)
perf record --call-graph=lbr -- ./myapp      # Last Branch Record (Intel only, low overhead, shallow)
perf record --call-graph=fp,5 -- ./myapp     # cap depth at 5 frames
perf record --no-call-graph -- ./myapp       # disable
```

### System-wide vs per-process

```bash
sudo perf record -a -g -- sleep 30
sudo perf record -C 0,1,2,3 -g -- sleep 10
perf record -p 12345 -g -- sleep 30
perf record -t 12348 -g -- sleep 30           # specific TID
perf record -u alice -g -- sleep 30           # all processes of user alice (recent perf)
```

### Output file

```bash
perf record -o /tmp/myapp.data -- ./myapp
perf record -o /tmp/myapp.data --append -- ./bench2
```

### Common record flags

```bash
-F HZ              # sample frequency
-c COUNT           # period (samples every COUNT events)
-e EVENTS          # event list
-g                 # enable call graphs (default mode varies)
--call-graph TYPE  # fp|dwarf|lbr [,SIZE]
-p PID             # attach
-t TID             # specific thread
-a                 # system-wide
-C CPUS            # restrict CPUs
-o FILE            # output file
--append           # append to existing perf.data
-S                 # snapshot mode (with --switch-output)
--switch-output    # rotate perf.data files
--switch-events    # rotate based on events
-z                 # compress traces (zstd)
-m PAGES           # mmap size in pages
--no-buffering     # disable mmap buffer
-D MS              # delay MS ms before sampling (warmup skip)
--running-time     # record runtime info
-T                 # include timestamps in samples (default for many events)
--per-thread       # per-thread sampling
--user-callchains  # only userspace callstacks
--kernel-callchains # only kernel callstacks
-N                 # don't save build IDs (faster but breaks cross-machine analysis)
--all-user         # only sample user-mode
--all-kernel       # only sample kernel-mode
--exclude-guest    # exclude KVM guest samples
--include-guest
-q                 # quieter
-v                 # more verbose
```

### Recording with timestamps for time-correlated analysis

```bash
perf record -e cycles -e sched:sched_switch -a -g --timestamp -- sleep 10
```

## perf report — Analyzing perf.data

```bash
perf report
perf report -i /tmp/myapp.data
```

### TUI navigation

```bash
# Up/Down or j/k     navigate functions
# Enter              expand/collapse children (callgraph)
# +                  expand callgraph one level
# -                  collapse callgraph one level
# E                  expand all
# C                  collapse all
# /                  filter by regex
# d                  filter to current dso
# c                  filter to current comm
# t                  filter to current thread
# s                  switch sort order
# a                  annotate (jump into source/asm)
# h                  help
# Esc / q            back / quit
```

### Common flags

```bash
perf report -g                          # enable call graphs in display
perf report -g graph,0.5,caller         # caller-based, 0.5% threshold
perf report -g graph,0.5,callee         # callee-based (inverse)
perf report --no-children               # show "self" only, not cumulative
perf report --stdio                     # plain text, no TUI
perf report --sort=comm,dso,symbol      # custom sort columns
perf report --sort=overhead,symbol
perf report --percent-limit 1           # hide entries below 1%
perf report --dsos=myapp,libc           # include only these binaries
perf report --comms=myapp               # filter by command name
perf report --symbols=hot_func          # filter by symbol
perf report --tid=12345
perf report --cpu=0,1
perf report -F overhead,sample,period,symbol  # custom fields
perf report -n                          # show sample counts alongside %
perf report --header                    # show recording metadata
perf report --kallsyms=/proc/kallsyms   # kernel symbols
perf report --vmlinux=/path/to/vmlinux  # for kernel symbol details
perf report --buildid-dir=~/.debug      # build-id symbol cache
perf report --time 100ms-500ms          # restrict to time window
perf report --hierarchy                 # collapsible hierarchical view
perf report --inline                    # show inline functions
```

### Self vs Children semantics

```bash
# "Self":     samples taken when this function was directly executing
# "Children": self + all samples from functions this called (cumulative)
```

A leaf function (no callees) has `self == children`. A wrapper that delegates everything has `self ≈ 0` and `children` close to 100%.

### Anatomy of a report line

```bash
# Overhead   Children   Self  Command   Shared Object         Symbol
# 12.34%      45.67%   8.91%  myapp     libc-2.31.so          [.] strstr
```

### Diff two recordings

```bash
perf diff before.data after.data
perf diff --baseline-if-not-zero before.data after.data
```

## perf top — Live Profiling

```bash
sudo perf top
sudo perf top -F 99 -g
sudo perf top -p 12345 -g
sudo perf top -e cache-misses -g
sudo perf top --no-children
sudo perf top --hide-kernel-symbols
sudo perf top --hide-user-symbols
sudo perf top -K                # hide kernel symbols (shorthand)
sudo perf top -U                # hide user symbols (shorthand)
sudo perf top --comms=nginx
sudo perf top --dsos=nginx,libssl.so
```

A live `top(1)`-style view that refreshes about once a second. Hot functions percolate to the top. Use it as the immediate "what's burning CPU right now" answer.

Interactive keys:

```bash
# Enter   focus on a function (annotate)
# h       help
# z       toggle zero-rate symbol filter
# q       quit
```

## perf annotate — Source / Asm Mixing

```bash
perf annotate -- after running perf record
perf annotate strstr
perf annotate --stdio strstr
perf annotate --stdio2 strstr             # cleaner format
perf annotate --asm-raw                   # show raw bytes
perf annotate --no-source                 # asm only
perf annotate -i /tmp/myapp.data symbol_name
```

The annotate view interleaves source lines (if debuginfo present) with the disassembled instructions, showing per-instruction sample percentages:

```bash
       │     for (i = 0; i < n; i++) {
  0.12 │ 10:   add    $0x1,%rax
       │       sum += arr[i] * arr[i];
 32.41 │       movsd  (%rdx,%rax,8),%xmm0
 41.83 │       mulsd  %xmm0,%xmm0
 18.22 │       addsd  %xmm0,%xmm1
  4.91 │       cmp    %rcx,%rax
  2.51 │       jne    10
```

The hot instruction (32%, 41%, 18%) is your bottleneck — typically a memory-load or an arithmetic stall.

Requires debug symbols (`-g` compile flag, or distro `*-debuginfo` / `*-dbgsym` packages).

## perf script — Scripted Output

```bash
perf script
perf script --header
perf script -i /tmp/myapp.data
perf script -F comm,pid,tid,cpu,time,event,ip,sym,dso
perf script -F +srcline,+brstack
perf script --ns                          # nanosecond timestamps
perf script --time 100s,200s              # time range
perf script --reltime                     # relative timestamps
perf script --tid=12345
```

### The flamegraph pipeline

```bash
perf record -F 99 -g -p $(pgrep myapp) -- sleep 30
perf script > out.perf
~/FlameGraph/stackcollapse-perf.pl out.perf > out.folded
~/FlameGraph/flamegraph.pl out.folded > flame.svg
xdg-open flame.svg
```

### Built-in scripts

```bash
perf script -l                           # list installed scripts
perf script syscall-counts               # syscall histogram
perf script failed-syscalls              # per-PID failed syscalls
perf script wakeup-latency               # wakeup latency analysis
perf script sched-migration              # sched migration tracker
perf script -s ~/my_analyzer.py          # run custom Python script
```

### Custom Python integration

```bash
perf script -g python > my_analyzer.py    # generate skeleton
perf script -s my_analyzer.py             # run it on perf.data
```

The skeleton has callbacks per event class. See `man perf-script-python`.

### Custom Perl integration

```bash
perf script -g perl > my_analyzer.pl
perf script -s my_analyzer.pl
```

## perf trace — Syscall Tracing

```bash
sudo perf trace -- ls
sudo perf trace -p 12345
sudo perf trace -a -- sleep 5
sudo perf trace -e openat,close
sudo perf trace -e !brk,!mmap          # exclude noisy syscalls
sudo perf trace --summary -p 12345 -- sleep 30
sudo perf trace --summary-only -a -- sleep 10
sudo perf trace --duration 1 -p 12345  # only show syscalls > 1ms
sudo perf trace --max-events 100
sudo perf trace --no-syscalls -e 'sched:*' -a -- sleep 5
sudo perf trace -F all -p 12345        # full mmap fault tracking
sudo perf trace -e 'block:*'           # tracepoint mode
```

Lower-overhead alternative to `strace`. The `--summary` mode prints a per-syscall histogram by count and total time at exit:

```bash
 Summary of events:

 myapp (12345), 2415 events, 100.0%

   syscall            calls  errors  total       min       avg       max     stddev
                                     (msec)    (msec)    (msec)    (msec)        (%)
   --------------- --------  ------ -------- --------- --------- ---------     ------
   read                 800       0   12.345     0.001     0.015     2.345    12.34%
   write                400       0    8.123     0.002     0.020     0.987     5.67%
   futex                500       0  120.456     0.001     0.241    50.123    34.56%
```

## perf sched — Scheduler Analysis

```bash
sudo perf sched record -- sleep 10
sudo perf sched latency
sudo perf sched latency --sort max
sudo perf sched timehist
sudo perf sched timehist --summary
sudo perf sched timehist -V                # verbose, with migration data
sudo perf sched map                        # textual CPU activity map
sudo perf sched script                     # raw events
sudo perf sched replay                     # synthesize identical workload (test scheduler)
```

`sched latency` output anatomy:

```bash
 ---------------------------------------------------------------------------------------------------------
  Task                  |   Runtime ms  | Switches | Avg delay ms    | Max delay ms    | Max delay start
 ---------------------------------------------------------------------------------------------------------
  myapp:12345           |     1234.567  |     8901 |    avg:  0.045  |    max:  3.456  |    max start: 12345.6789
  swapper:0/1           |      234.567  |      890 |    avg:  0.012  |    max:  0.987  |
```

`Max delay` is the longest a task waited on the runqueue between wakeup and execution. Spikes usually mean noisy neighbors, lock contention, or priority inversion.

## perf lock — Lock Contention

```bash
sudo perf lock record -- ./myapp
sudo perf lock report
sudo perf lock contention                  # newer subcommand
sudo perf lock info -k                     # known lock keys
sudo perf lock script
```

`perf lock contention` (recent perf) sorts kernel locks by contention time:

```bash
 contended   total wait     max wait     avg wait         type   caller
        87       4.56 ms      1.23 ms     52.41 us       mutex   inode_lookup+0x80
        45       2.34 ms      0.87 ms     52.00 us    spinlock   tcp_v4_rcv+0x230
```

Cross-reference with kernel source to identify which subsystem's lock is hot.

## perf mem — Memory Profiling

```bash
sudo perf mem record -- ./myapp
sudo perf mem record -t store -- ./myapp     # only stores
sudo perf mem record --ldlat 30 -- ./myapp   # PEBS load-latency >= 30 cycles
sudo perf mem report
sudo perf mem report --sort=mem
```

Records load and store events with PEBS-precise addresses. The `mem` column in the report shows hit type:

```bash
# L1 hit | L2 hit | LLC hit | Local RAM | Remote RAM | RFO miss | etc.
```

Useful for diagnosing LLC misses tied to specific data structures. Requires Intel PEBS or AMD IBS.

## perf c2c — Cache-Line Contention

```bash
sudo perf c2c record -- ./myapp
sudo perf c2c record -a -- sleep 30
sudo perf c2c report
sudo perf c2c report --stats
sudo perf c2c report --coalesce tid,iaddr
```

The canonical false-sharing detector. Report shows cache lines being accessed by multiple cores with HITM (modified-hit) events:

```bash
 -------------------------------------------------------------
 Cacheline                       Total      HITM      Local
 -------------------------------------------------------------
 0xffffabc012340000              45,123     1,234     43,456
   Function: ./myapp+0x...
   Function: ./myapp+0x...
```

If two unrelated fields share a 64-byte line and are written by different cores, you'll see massive HITM counts. Fix: pad/align the struct so each field gets its own cache line (`__attribute__((aligned(64)))`).

## perf probe — Dynamic Probing

### Add probes

```bash
sudo perf probe --add tcp_sendmsg
sudo perf probe --add 'tcp_sendmsg sk size=$arg2'
sudo perf probe --add 'do_sys_openat2 filename:string'
sudo perf probe -x /lib/x86_64-linux-gnu/libc.so.6 --add 'malloc size'
sudo perf probe -x /usr/bin/myapp --add 'main argc'
sudo perf probe -x ./myapp --add 'mycode.c:42'
```

### List, delete, vars

```bash
sudo perf probe --list
sudo perf probe --del tcp_sendmsg
sudo perf probe --del 'probe:*'                # delete all
sudo perf probe -V tcp_sendmsg                 # show available variables at function entry
sudo perf probe -L tcp_sendmsg                 # show source lines
sudo perf probe -F                             # list functions perf can probe
```

### Use the probe

```bash
sudo perf record -e probe:tcp_sendmsg -aR -- sleep 10
sudo perf script
```

### Limitations

Probes need symbol info — for kernel: `vmlinux` with debug data (or BTF on recent kernels). For userspace: ELF + DWARF debug info. If `perf probe -V` says `Failed to find ...`, you need debuginfo.

## perf ftrace — ftrace Frontend

```bash
sudo perf ftrace -- sleep 1
sudo perf ftrace -p 12345 -- sleep 5
sudo perf ftrace -T 'tcp_*' -- sleep 5         # function tracer
sudo perf ftrace -G 'do_sys_openat2' -- ls     # graph tracer
sudo perf ftrace -t function_graph -p 12345 -- sleep 1
sudo perf ftrace -F                            # list available tracers
sudo perf ftrace --notrace 'tcp_recvmsg' -p 12345
```

Wraps `/sys/kernel/debug/tracing` so you don't have to manually echo into trace files. Best for tracing kernel function call paths inline.

## perf kvm — KVM Profiling

```bash
sudo perf kvm stat -a -- sleep 30
sudo perf kvm stat live
sudo perf kvm stat report
sudo perf kvm record -a -- sleep 30
sudo perf kvm --host --guest top
sudo perf kvm --guestmount=/var/lib/libvirt/qemu --guestvmlinux=/path/vmlinux record
```

Host-side analysis of guest VMs. `perf kvm stat` summarizes VM-EXIT reasons (e.g. EPT_VIOLATION, EXTERNAL_INTERRUPT, MSR_READ) — high counts of any particular reason point at virtualization overhead.

## perf bench — Built-in Benchmarks

```bash
perf bench
perf bench all
perf bench sched
perf bench sched all
perf bench sched messaging
perf bench sched pipe
perf bench mem
perf bench mem memcpy -l 1GB
perf bench mem memset -l 1GB
perf bench mem memcpy --function default,prefault
perf bench futex
perf bench futex all
perf bench futex hash
perf bench futex wake
perf bench futex requeue
perf bench numa
perf bench syscall basic
perf bench internals synthesize
```

A built-in micro-benchmark suite covering scheduler, memory, futex, syscall, and NUMA primitives. Useful baseline tests when comparing kernels or hardware.

## Topdown Microarchitecture Analysis

```bash
sudo perf stat --topdown -a -- sleep 5
sudo perf stat --topdown --td-level 2 -a -- sleep 5
sudo perf stat -M TopdownL1 -- ./myapp        # explicit metric group
sudo perf stat -M TopdownL2 -- ./myapp
```

The Top-Down Methodology (Yasin, Intel) classifies every issue slot into one of four buckets:

```bash
# Frontend Bound      — fetch/decode can't deliver enough uops
#   Frontend Latency    (e.g. icache misses, iTLB misses)
#   Frontend Bandwidth
# Bad Speculation     — uops issued but flushed (mispredicts, machine clears)
# Backend Bound       — execution units stalled
#   Memory Bound        (cache misses, store buffer full)
#   Core Bound          (long-latency arithmetic, port contention)
# Retiring            — useful work; want this high
```

Sample output:

```bash
 Performance counter stats for 'system wide':

       retiring       bad speculation       frontend bound       backend bound
            38.5%                  4.2%                21.3%               36.0%
```

Diagnosis cheat:

```bash
# High Frontend Bound  → icache/iTLB misses, branch density; reduce code footprint
# High Bad Speculation → branch mispredicts; restructure conditionals, profile-guided opt
# High Backend Memory  → LLC misses, prefetch issues; better data layout, blocking
# High Backend Core    → ALU port pressure; vectorize, reduce serial chains
# High Retiring        → close to peak; remaining wins are algorithmic
```

Requires recent kernel + recent perf + supported PMU. Both Intel (Sandy Bridge+) and AMD (Zen 3+) support topdown to varying degrees.

## Hardware PMU Events

```bash
perf list pmu
perf list pmu | head -100
perf list 'cpu/*'
```

### Raw event syntax

```bash
perf stat -e 'cpu/event=0xc0,umask=0x00/' -- ./bench
perf stat -e 'cpu/event=0xd1,umask=0x01,name=mem_load_l1_hit/' -- ./bench
perf stat -e 'cpu/event=0xc4,umask=0x20,cmask=1,inv,edge/' -- ./bench
```

Available encoding modifiers (Intel):

```bash
event=0xNN     # event selector
umask=0xMM     # unit mask
edge           # edge-detect mode
inv            # invert cmask comparison
cmask=N        # counter mask
any            # measure on any thread of core (HT-aware)
pc             # pin (PEBS-Capable)
in_tx          # transactional memory inside-tx
in_tx_cp       # tx checkpoint
period=N       # sampling period
freq=N         # sampling frequency
name=str       # display name
ldlat=N        # load latency threshold (PEBS)
```

### Common goal-driven raw events

```bash
# L1 dcache replacement
perf stat -e 'cpu/event=0x51,umask=0x01,name=l1d_replacement/' -- ./bench

# Branch mispredict by category (Skylake)
perf stat -e 'cpu/event=0xc5,umask=0x00/' -e 'cpu/event=0xc5,umask=0x20/' -- ./bench

# Cycles stalled on memory
perf stat -e 'cpu/event=0xa3,umask=0x14,cmask=0x14/' -- ./bench
```

Always cross-reference with the vendor's optimization manual for your specific microarchitecture — event codes differ between generations.

## Sampling Frequency vs Period

```bash
perf record -F 99 -- ./bench       # frequency: ~99 samples per second per CPU, kernel adapts period
perf record -c 1000000 -- ./bench  # period: one sample every 1,000,000 events
```

| Mode | Pros | Cons |
|------|------|------|
| Frequency (`-F`) | Adapts to workload, predictable rate | Less deterministic for analysis |
| Period (`-c`) | Deterministic, can correlate with absolute counts | Heavy/light workloads sample at very different wall-clock rates |

### The 99 Hz convention

```bash
perf record -F 99 -g -- ./bench
```

Instead of 100 Hz (which often aliases with NTP-aligned 100Hz timer interrupts), 99 Hz is mutually prime with most periodic kernel work. 999 and 9973 are similar choices for higher-rate sampling.

### Frequency clamping

The kernel caps `-F` at `kernel.perf_event_max_sample_rate` (default 100000). To raise:

```bash
sudo sysctl -w kernel.perf_event_max_sample_rate=200000
```

You may see:

```bash
WARNING: Maximum frequency rate (100,000 Hz) exceeded.
```

## The Flamegraph Pipeline

The Brendan Gregg canonical workflow:

```bash
git clone https://github.com/brendangregg/FlameGraph ~/FlameGraph

perf record -F 99 -g -p $(pgrep -f myapp) -- sleep 30
perf script > out.perf
~/FlameGraph/stackcollapse-perf.pl out.perf > out.folded
~/FlameGraph/flamegraph.pl out.folded > flame.svg
```

### Color schemes

```bash
~/FlameGraph/flamegraph.pl --color=hot         out.folded > flame.svg     # default
~/FlameGraph/flamegraph.pl --color=mem         out.folded > flame.svg
~/FlameGraph/flamegraph.pl --color=io          out.folded > flame.svg
~/FlameGraph/flamegraph.pl --color=java        out.folded > flame.svg
~/FlameGraph/flamegraph.pl --color=red         out.folded > flame.svg
~/FlameGraph/flamegraph.pl --color=green       out.folded > flame.svg
~/FlameGraph/flamegraph.pl --colors=java       out.folded > flame.svg     # palette by language
```

### Useful flags

```bash
~/FlameGraph/flamegraph.pl --width 1600 --height 24 --title "myapp CPU" out.folded > flame.svg
~/FlameGraph/flamegraph.pl --reverse --inverted out.folded > icicle.svg     # icicle (top-down)
~/FlameGraph/flamegraph.pl --hash out.folded > flame.svg                    # consistent colors across runs
~/FlameGraph/flamegraph.pl --countname=samples out.folded > flame.svg
```

### Differential flamegraphs

```bash
~/FlameGraph/difffolded.pl before.folded after.folded | ~/FlameGraph/flamegraph.pl > diff.svg
```

Red-tinted means slower in `after.folded`, blue means faster.

## Off-CPU Flamegraphs

On-CPU flamegraphs only show what's running. Off-CPU flamegraphs show what's *waiting*. Combined, they tell the full latency story.

### perf-based off-CPU (heavy)

```bash
sudo perf record -e sched:sched_switch -e sched:sched_stat_sleep -e sched:sched_stat_iowait \
                  -a -g --call-graph dwarf -- sleep 30
sudo perf script | ~/FlameGraph/stackcollapse-perf.pl --kernel | \
                   ~/FlameGraph/flamegraph.pl --color=io --title="Off-CPU" > offcpu.svg
```

This is high-overhead because every context switch is recorded. On busy boxes, perf.data can balloon to GB.

### eBPF-based off-CPU (light, recommended)

```bash
sudo offcputime-bpfcc -df -p $(pgrep myapp) 30 > out.stacks
sudo offcputime -df -p $(pgrep myapp) 30 > out.stacks    # bcc-tools or libbpf-tools variant
~/FlameGraph/flamegraph.pl --color=io --countname=us --title="Off-CPU Time" < out.stacks > offcpu.svg
```

Roughly 100x lighter than the perf method. The bcc/bpftrace `offcputime` only emits a sum of off-CPU time per stack rather than every individual switch event.

## Real-World Workflows

### Why is my server CPU at 100%?

```bash
sudo perf top -F 99 -g
```

Identifies the hot functions in real time. If kernel functions dominate, check `softirq`, network, filesystem. If user functions dominate, those are your code's hot path.

### Why is this batch slow?

```bash
perf stat -d -- ./batch input.csv
perf stat -- ./batch input.csv 2>&1 | tee baseline.txt
# tweak code...
perf stat -- ./batch input.csv 2>&1 | tee after.txt
diff baseline.txt after.txt
```

### Where in my code is the hot path?

```bash
perf record -F 99 -g -- ./bin args
perf report                          # interactive
# or
perf script | ~/FlameGraph/stackcollapse-perf.pl | ~/FlameGraph/flamegraph.pl > flame.svg
```

### Is my code mispredicting branches?

```bash
perf stat -e branches,branch-misses -- ./bench
```

>5% miss rate is a problem. Use `perf record -e branch-misses -g` to find the worst predictors, then audit hot conditionals.

### What's stalling my pipeline?

```bash
sudo perf stat --topdown -a -- sleep 5
sudo perf stat --topdown --td-level 2 -a -- sleep 5
```

### Cache miss city?

```bash
perf stat -e cache-references,cache-misses,L1-dcache-loads,L1-dcache-load-misses,LLC-loads,LLC-load-misses -- ./bench
```

LLC miss rate above ~10% is usually pathological for serial workloads. Compute hit rate as `1 - (misses / loads)`.

### Latency distribution?

```bash
perf record -e cycles --call-graph dwarf -- ./bench
perf script -F time,event,ip,sym | ./my_latency_histo.py
```

Or use `perf script -g python` to generate a Python skeleton and aggregate samples into a histogram.

### NUMA imbalance?

```bash
perf stat -e node-loads,node-load-misses -a -- sleep 10
sudo perf c2c record -a -- sleep 30 && sudo perf c2c report
```

### Disk I/O latency?

```bash
sudo perf record -e 'block:block_rq_issue,block:block_rq_complete' -a -g -- sleep 10
sudo perf script | awk '/block_rq_issue/{...}'
```

### Specific syscall slowness?

```bash
sudo perf trace -e openat,read --duration 1 -p $(pgrep myapp)
```

### Find which kernel function is on the hot path

```bash
sudo perf record -F 99 -ag -- sleep 10
sudo perf report --kallsyms=/proc/kallsyms
```

## Scripts and Custom Analysis

### Generate a Python skeleton

```bash
perf script -g python > my_analyzer.py
```

The skeleton has callbacks like:

```bash
def trace_begin():
def trace_end():
def cycles(event_name, context, ...):     # one callback per event
def sched__sched_switch(...):
```

### Run it

```bash
perf script -i perf.data -s my_analyzer.py
```

### Generate a Perl skeleton

```bash
perf script -g perl > my_analyzer.pl
perf script -i perf.data -s my_analyzer.pl
```

### Useful built-in scripts

```bash
perf script -l                          # list available
perf script wakeup-latency
perf script syscall-counts
perf script syscall-counts-by-pid
perf script failed-syscalls
perf script failed-syscalls-by-pid
perf script sctop                       # syscall top
perf script futex-contention
perf script net_dropmonitor
perf script rwtop
perf script intel-pt-events             # Intel PT specific
perf script power-usage
perf script powerpc-hcalls
```

## JIT Code Symbol Resolution

JIT-compiled code (Java, Node, Mono, V8, JIT-PHP) doesn't appear in static ELF symbol tables. perf supports JIT symbol maps via `/tmp/perf-PID.map`.

### Format

```bash
# /tmp/perf-PID.map — one line per JITted function:
# <hex_start_addr> <hex_size> <symbol_name>
0x7f1234560000 0x40 java/lang/String.charAt
0x7f1234560040 0x80 com/myapp/Foo.bar
```

### Java

```bash
java -XX:+PreserveFramePointer -agentpath:/usr/lib/libperf-jvmti.so -jar app.jar
java -XX:+UnlockDiagnosticVMOptions -XX:+DebugNonSafepoints \
     -agentpath:/path/to/perf-map-agent.so -jar app.jar
```

The `perf-map-agent` (github.com/jvm-profiling-tools/perf-map-agent) hooks the JVM and writes `/tmp/perf-<PID>.map` periodically. `-XX:+PreserveFramePointer` keeps RBP usable so frame-pointer call-graphs work.

### Node.js / V8

```bash
node --perf-basic-prof --perf-prof-unwinding-info myapp.js
node --perf-basic-prof-only-functions myapp.js
```

Writes `/tmp/perf-<PID>.map` automatically.

### Rust / native

Native compiled code already has symbols in the ELF. For `cargo build --release`, ensure debug info is kept:

```bash
[profile.release]
debug = 1
```

### Go

Go has frame pointers since 1.7 on amd64. Use `-gcflags="all=-N -l"` for inlining-free profiling.

### perf inject for jitdump

```bash
perf record -k 1 -- ./jit_app                    # write monotonic timestamps
perf inject -j -i perf.data -o perf.jit.data     # inject jitdump
perf report -i perf.jit.data
```

The `perf-map-agent` and the V8 / OpenJDK hooks can write a `jit-PID.dump` that `perf inject -j` will fold into perf.data along with proper symbol info.

## perf inject

```bash
perf inject -j -i perf.data -o perf.jit.data       # inject jitdump
perf inject --build-ids -i perf.data -o perf.b.data  # embed build-IDs (for cross-machine)
perf inject -i perf.data --strip                   # strip event-irrelevant data
perf inject -i perf.data -o perf.itrace.data --itrace=cr  # ITrace control-flow synthesis (Intel PT)
```

## perf archive

```bash
perf archive perf.data                  # creates perf.data.tar.bz2 with .debug entries
tar xjf perf.data.tar.bz2 -C ~/.debug   # extract on the analysis machine
perf report -i perf.data
```

The archive bundles the build-ID-keyed binaries and debug files referenced by the recording, so you can move analysis to a different host.

Alternative for JIT: copy `/tmp/perf-PID.map` alongside `perf.data` to the analysis host.

## Common Errors and Fixes

### perf_event_paranoid

```bash
# Error:
# perf_event_paranoid: Operation not permitted

# Fix (interactive):
sudo sysctl -w kernel.perf_event_paranoid=-1

# Fix (persistent):
echo 'kernel.perf_event_paranoid = -1' | sudo tee /etc/sysctl.d/99-perf.conf

# Or grant capability:
sudo setcap cap_perfmon,cap_sys_ptrace+ep $(which perf)
```

### Failed to mmap

```bash
# Error:
# Failed to mmap with 1 (Operation not supported)

# Cause: kernel built without hardware PMU support, or running in a VM with no PMU passthrough.

# Fix:
perf stat -e cpu-clock,task-clock,page-faults -- ./bench
perf record -e cpu-clock -- ./bench
```

Use software events that don't depend on hardware counters.

### No symbol-table

```bash
# Error:
# couldn't find a fitting symbol-table for ./myapp

# Fix:
sudo apt-get install linux-image-$(uname -r)-dbgsym       # kernel
sudo apt-get install libc6-dbg                            # glibc
gcc -g -fno-omit-frame-pointer -O2 ...                    # rebuild your code with debug info
```

### Kernel address maps restricted

```bash
# Warning:
# WARNING: Kernel address maps (/proc/{kallsyms,modules}) are restricted, check
#          /proc/sys/kernel/kptr_restrict and /proc/sys/kernel/perf_event_paranoid.

# Fix:
sudo sysctl -w kernel.kptr_restrict=0
sudo sysctl -w kernel.perf_event_paranoid=-1
```

### Unresolved JIT symbols

```bash
# Warning:
# WARNING: ... 0x... not resolved
# 99% [unknown]

# Cause: JIT code with no /tmp/perf-PID.map.

# Fix (Java):
java -XX:+PreserveFramePointer -agentpath:/usr/lib/libperf-jvmti.so -jar app.jar

# Fix (Node.js):
node --perf-basic-prof myapp.js

# Verify:
ls -l /tmp/perf-*.map
```

### Frame pointer stacks broken

```bash
# Symptom: --call-graph=fp produces 1-2 frame deep stacks ending at libc
# Cause: binary built with -fomit-frame-pointer (default in gcc -O2)

# Fix: rebuild with -fno-omit-frame-pointer, or switch:
perf record --call-graph=dwarf -- ./bin
perf record --call-graph=lbr -- ./bin       # Intel only
```

### perf.data too large

```bash
# Symptom: perf.data is 10s of GB after a short recording
# Cause: dwarf call-graph captures large stacks; system-wide; high frequency.

# Fix:
perf record -F 49 -- ./bin                      # lower frequency
perf record --call-graph=fp -- ./bin            # cheaper unwinding
perf record -p PID -- sleep 30                  # not -a
perf record -z -- ./bin                         # zstd compression
perf record -m 8 -- ./bin                       # smaller mmap
```

### perf top: Unresolvable symbol

```bash
# Symptom: perf top shows "[k] 0xffffffff8123abcd"

# Fix:
sudo sysctl -w kernel.kptr_restrict=0
sudo perf top
```

### "couldn't find symbols" but the binary is right there

```bash
# Cause: binary was rebuilt after recording; build-IDs no longer match.

# Fix:
perf buildid-list -i perf.data | head
perf archive perf.data         # capture matching binaries
# or rerun the recording after the build is stable
```

### Container profiling shows host symbols only

```bash
# Symptom: profiling a process inside a container shows /proc/PID/root paths missing.

# Fix:
sudo perf record --buildid-all -p $(pgrep -n myapp) -g -- sleep 30
sudo perf record --namespaces -- ./bin           # newer kernels
sudo perf inject --buildid-all -i perf.data -o perf.bid.data
```

### Hardware events disabled in VMs

```bash
# Error in a KVM/VMware/Hyper-V guest:
# <not supported> for cycles/instructions

# Fix (KVM, on host):
# qemu -cpu host,+pmu                              # enable virtualized PMU
# or libvirt: <feature policy='require' name='pmu'/>
```

## Common Gotchas

### Frame pointer without -fno-omit-frame-pointer

```bash
# bad:
gcc -O2 myapp.c -o myapp
perf record --call-graph=fp ./myapp
# stacks broken — most show 1-2 frames

# fixed:
gcc -O2 -fno-omit-frame-pointer myapp.c -o myapp
perf record --call-graph=fp ./myapp
# OR
perf record --call-graph=dwarf ./myapp
```

### DWARF on huge processes

```bash
# bad:
perf record -F 99 -g --call-graph=dwarf -p $(pgrep firefox) -- sleep 30
# generates 10+ GB perf.data on a busy browser

# fixed:
perf record -F 49 -g --call-graph=dwarf,4096 -p $(pgrep firefox) -- sleep 30
# OR on Intel:
perf record -F 99 -g --call-graph=lbr -p $(pgrep firefox) -- sleep 30
```

### 100Hz aliasing

```bash
# bad:
perf record -F 100 -g -- ./bench
# samples land in lock-step with timer interrupts → bias toward ISR code

# fixed:
perf record -F 99 -g -- ./bench
```

### Running unprivileged with paranoid kernel

```bash
# bad:
perf record -a -- sleep 5
# Error: You may not have permission ...

# fixed:
sudo perf record -a -- sleep 5
# OR
sudo sysctl -w kernel.perf_event_paranoid=-1
sudo setcap cap_perfmon,cap_sys_ptrace+ep $(which perf)
```

### JIT without perf-map

```bash
# bad:
java -jar app.jar
perf record -F 99 -g -p $(pgrep java) -- sleep 30
perf report
# all stacks show [unknown]

# fixed:
java -XX:+PreserveFramePointer -agentpath:/usr/lib/libperf-jvmti.so -jar app.jar
perf record -k 1 -F 99 -g -p $(pgrep java) -- sleep 30
perf inject -j -i perf.data -o perf.jit.data
perf report -i perf.jit.data
```

### Forgetting -a or -p

```bash
# bad (forgot to attach):
perf record -- sleep 30
# you profiled "sleep", not your workload

# fixed:
perf record -p $(pgrep myapp) -- sleep 30
# OR system-wide:
sudo perf record -a -- sleep 30
```

### Recording on the wrong CPUs

```bash
# bad: workload pinned to CPU 7, but you profile -C 0,1,2,3
sudo perf record -C 0,1,2,3 -- sleep 10

# fixed: profile the right CPUs or all
sudo perf record -C 7 -- sleep 10
sudo perf record -a -- sleep 10
```

### Forgetting --call-graph=lbr depth

```bash
# bad:
perf record --call-graph=lbr -- ./bench
# LBR captures only ~16-32 last branches → very shallow stacks

# fixed: combine, or use dwarf
perf record --call-graph=dwarf -- ./bench
```

### Mixing system-wide and PID for incompatible reasons

```bash
# bad: system-wide trace of millions of context switches:
sudo perf record -e sched:sched_switch -ag -- sleep 60

# fixed: scope or sample:
sudo perf record -e sched:sched_switch -p $(pgrep myapp) -g -- sleep 60
sudo perf record -e sched:sched_switch -ag --filter "prev_comm == \"myapp\"" -- sleep 60
```

### CPU frequency scaling skewing IPC

```bash
# bad: comparing IPC across runs on a laptop with thermal throttling
perf stat -- ./bench       # run 1: 4.0 GHz, IPC 1.5
perf stat -- ./bench       # run 2: 1.8 GHz, IPC 1.5  (looks identical, isn't)

# fixed: pin frequency
sudo cpupower frequency-set -g performance
sudo cpupower frequency-set --min 3.0GHz --max 3.0GHz
perf stat -- ./bench
```

### Stripping debug info

```bash
# bad:
strip myapp
perf record -g -- ./myapp
perf report     # all functions show as "[.] _start" or similar

# fixed: keep debug info, or split:
objcopy --only-keep-debug myapp myapp.debug
strip --strip-debug myapp
objcopy --add-gnu-debuglink=myapp.debug myapp
```

## Performance Tips

- **Prefer LBR on Intel** for low-overhead call graphs (`--call-graph=lbr`); ~16-32 frames but near-zero perturbation.
- **Sample shorter (`-F 49`)** for less data when you only need to find the rough shape.
- **`-p PID` over `-a`** when you can — avoids profiling the whole system.
- **Specific events over the default broad set** — `-e cycles,instructions` is often enough for IPC, faster than the default 10+ event group.
- **Compress with `-z`** if disk is the bottleneck (zstd, modern perf).
- **Pin CPU frequency** before benchmarking. `cpupower frequency-set -g performance` then min==max.
- **Disable Turbo Boost** for deterministic counts: `echo 1 > /sys/devices/system/cpu/intel_pstate/no_turbo`.
- **Disable ASLR** for reproducible symbol addresses across runs: `setarch x86_64 -R ./bench`.
- **Use perf record's `-D`** to skip warmup time in noisy startups.
- **Match `perf` and kernel versions** — features like topdown level 2, BTF support, certain tracepoints depend on it.
- **Build with `-fno-omit-frame-pointer`** even in release builds; the cost is ~1% but profiling becomes 100x easier.
- **Keep debuginfo split** so production binaries stay small but symbols are available via `--debug-file-directory`.
- **Use `perf record -m PAGES`** to size the mmap ring buffer; smaller saves memory, larger reduces drops at high event rates.
- **Use `perf record --switch-output=1G`** for long captures so files don't grow unbounded.
- **Annotate first, optimize second** — `perf annotate` tells you which exact instruction is hot, which often makes the fix obvious.
- **Cross-validate with `perf stat`** — flamegraphs show *where*, counters confirm *why*.

## Idioms

### The 99 Hz CPU flamegraph

```bash
sudo perf record -F 99 -ag -- sleep 30
sudo perf script | ~/FlameGraph/stackcollapse-perf.pl | ~/FlameGraph/flamegraph.pl > flame.svg
```

### The IPC baseline

```bash
perf stat -- ./bench 2>&1 | grep -E 'instructions|IPC|insn per cycle'
```

### The live diagnosis

```bash
sudo perf top -F 99 -g
```

### The pipeline-stall diagnosis

```bash
sudo perf stat --topdown -a -- sleep 5
```

### The before/after compare

```bash
perf stat -r 10 -e cycles,instructions -- ./bench > before.txt 2>&1
# (apply optimization)
perf stat -r 10 -e cycles,instructions -- ./bench > after.txt 2>&1
diff before.txt after.txt
```

### The cache-miss audit

```bash
perf stat -e cache-references,cache-misses,L1-dcache-loads,L1-dcache-load-misses,LLC-loads,LLC-load-misses -- ./bench
```

### The branch-mispredict audit

```bash
perf stat -e branches,branch-misses -- ./bench
perf record -e branch-misses -g -- ./bench
perf report
```

### The syscall hot list

```bash
sudo perf trace --summary-only -p $(pgrep myapp) -- sleep 10
```

### The scheduler-latency audit

```bash
sudo perf sched record -- sleep 10
sudo perf sched latency --sort max
```

### The lock-contention audit

```bash
sudo perf lock record -- ./bench
sudo perf lock contention
```

### The false-sharing audit

```bash
sudo perf c2c record -a -- sleep 30
sudo perf c2c report
```

### The annotate-then-fix

```bash
perf record -F 999 -g -- ./bench
perf report
# pick hot symbol, press 'a' for annotate
# look for the single hot instruction
# fix the source
```

### The off-CPU companion

```bash
# on-CPU
perf record -F 99 -g -- ./bench
perf script | ~/FlameGraph/stackcollapse-perf.pl | ~/FlameGraph/flamegraph.pl > on.svg
# off-CPU (eBPF preferred)
sudo offcputime -df 30 > off.stacks
~/FlameGraph/flamegraph.pl --color=io < off.stacks > off.svg
```

### The container-aware profile

```bash
sudo perf record --namespaces --buildid-all -F 99 -g -p $(docker top mycontainer | awk 'NR==2{print $2}') -- sleep 30
```

### The kernel-only deep dive

```bash
sudo perf record -F 99 -ag --all-kernel -- sleep 10
sudo perf report
```

### The user-only profile

```bash
perf record -F 99 -g --all-user -p $(pgrep myapp) -- sleep 30
perf report
```

### The perf.data diff

```bash
perf record -o before.data -- ./bench
# tweak
perf record -o after.data -- ./bench
perf diff before.data after.data
```

### The interval time-series

```bash
perf stat -I 1000 -e cycles,instructions -a 2>&1 | tee timeseries.txt
```

### The repeatable benchmark

```bash
perf stat -r 20 --table -e cycles,instructions,cache-misses -- ./bench
```

### The annotated hot loop

```bash
perf record -F 999 -g --call-graph=dwarf -- ./bench
perf annotate --stdio2 hot_function | less
```

### The bench-the-kernel sanity check

```bash
perf bench sched messaging -g 50 -l 1000
perf bench mem memcpy -l 1GB
perf bench futex hash
```

## Tips

- Always start with `perf stat` — it's cheap and tells you whether the bottleneck is compute, memory, or branching before you commit to record/report cycles.
- A flamegraph shows you *where* time is spent; `perf annotate` shows you *which exact instruction*; `perf stat -e ...` confirms the *why*.
- Off-CPU + on-CPU flamegraphs together cover both running and waiting time. Use eBPF (`offcputime`) for the off-CPU side; it's far cheaper than the perf-record-based variant.
- For language runtimes with JITs (Java, Node, .NET), the perf-map / jitdump integration is non-optional — without it your stacks are 90% `[unknown]`.
- The kernel changes constantly. If `perf list` shows fewer events than expected, your perf binary may be older than the kernel; rebuild from the kernel source tree's `tools/perf/`.
- `perf` and `eBPF` overlap: bcc/bpftrace tools are often lighter and easier for ad-hoc questions; `perf` is the canonical sampler and the only one with hardware-counter access on most distros.

## See Also

- ebpf
- bpftrace
- bpftool
- flamegraph
- polyglot
- bash

## References

- [man perf(1)](https://man7.org/linux/man-pages/man1/perf.1.html)
- [man perf-record(1)](https://man7.org/linux/man-pages/man1/perf-record.1.html)
- [man perf-stat(1)](https://man7.org/linux/man-pages/man1/perf-stat.1.html)
- [man perf-report(1)](https://man7.org/linux/man-pages/man1/perf-report.1.html)
- [man perf-top(1)](https://man7.org/linux/man-pages/man1/perf-top.1.html)
- [man perf-script(1)](https://man7.org/linux/man-pages/man1/perf-script.1.html)
- [man perf-trace(1)](https://man7.org/linux/man-pages/man1/perf-trace.1.html)
- [man perf-sched(1)](https://man7.org/linux/man-pages/man1/perf-sched.1.html)
- [man perf-lock(1)](https://man7.org/linux/man-pages/man1/perf-lock.1.html)
- [man perf-mem(1)](https://man7.org/linux/man-pages/man1/perf-mem.1.html)
- [man perf-c2c(1)](https://man7.org/linux/man-pages/man1/perf-c2c.1.html)
- [man perf-probe(1)](https://man7.org/linux/man-pages/man1/perf-probe.1.html)
- [man perf-ftrace(1)](https://man7.org/linux/man-pages/man1/perf-ftrace.1.html)
- [man perf-bench(1)](https://man7.org/linux/man-pages/man1/perf-bench.1.html)
- [man perf-kvm(1)](https://man7.org/linux/man-pages/man1/perf-kvm.1.html)
- [man perf-inject(1)](https://man7.org/linux/man-pages/man1/perf-inject.1.html)
- [man perf-archive(1)](https://man7.org/linux/man-pages/man1/perf-archive.1.html)
- [man perf-script-python(1)](https://man7.org/linux/man-pages/man1/perf-script-python.1.html)
- [perf Wiki — Main](https://perf.wiki.kernel.org/index.php/Main_Page)
- [perf Wiki — Tutorial](https://perf.wiki.kernel.org/index.php/Tutorial)
- [perf Wiki — Top-Down Analysis](https://perf.wiki.kernel.org/index.php/Top-Down_Analysis)
- [Kernel admin-guide — perf-security](https://www.kernel.org/doc/html/latest/admin-guide/perf-security.html)
- [Brendan Gregg — perf Examples](https://www.brendangregg.com/perf.html) — THE comprehensive perf intro
- [Brendan Gregg — Linux perf Flame Graphs](https://www.brendangregg.com/FlameGraphs/cpuflamegraphs.html)
- [Brendan Gregg — Off-CPU Flame Graphs](https://www.brendangregg.com/FlameGraphs/offcpuflamegraphs.html)
- [Brendan Gregg — Java Flame Graphs](https://www.brendangregg.com/blog/2017-06-30/java-flame-graphs.html)
- ["Systems Performance" by Brendan Gregg (2nd ed.)](https://www.brendangregg.com/systems-performance-2nd-edition-book.html)
- ["BPF Performance Tools" by Brendan Gregg](https://www.brendangregg.com/bpf-performance-tools-book.html)
- [github.com/brendangregg/FlameGraph](https://github.com/brendangregg/FlameGraph)
- [github.com/jvm-profiling-tools/perf-map-agent](https://github.com/jvm-profiling-tools/perf-map-agent)
- [Intel — Top-Down Microarchitecture Analysis Method](https://www.intel.com/content/www/us/en/docs/vtune-profiler/cookbook/2023-0/top-down-microarchitecture-analysis-method.html)
- [Yasin — A Top-Down Method for Performance Analysis (paper)](https://ieeexplore.ieee.org/document/6844459)
- [Red Hat — Performance Observability with perf](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/getting-started-with-perf_monitoring-and-managing-system-status-and-performance)
- [Arch Wiki — Perf](https://wiki.archlinux.org/title/Perf)
- [Linux kernel source — tools/perf/Documentation](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/tools/perf/Documentation)
