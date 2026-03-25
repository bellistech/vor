# perf (Linux Performance Profiler)

Linux profiling tool for CPU performance counters, tracing, and sampling.

## perf stat (Counting)

### Count hardware events for a command

```bash
perf stat ls
perf stat -d ls                            # detailed (L1 cache, branch misses)
perf stat -dd ls                           # more detail
perf stat -ddd ls                          # maximum detail
```

### Specific events

```bash
perf stat -e cycles,instructions,cache-misses,branch-misses ./myapp
perf stat -e L1-dcache-loads,L1-dcache-load-misses ./myapp
```

### Profile a running process

```bash
perf stat -p 12345                         # attach to PID, Ctrl-C to stop
perf stat -p 12345 sleep 10               # collect for 10 seconds
```

### System-wide

```bash
perf stat -a sleep 5                       # all CPUs for 5 seconds
perf stat -a -e context-switches,cpu-migrations sleep 10
```

### Per-CPU breakdown

```bash
perf stat -a -A -e cycles sleep 5          # per-CPU counts
```

## perf record (Sampling)

### Record a command

```bash
perf record ./myapp                        # creates perf.data
perf record -g ./myapp                     # with call graphs (stack traces)
perf record -F 99 -g ./myapp              # sample at 99 Hz
```

### Record a running process

```bash
perf record -p 12345 -g                    # attach to PID, Ctrl-C to stop
perf record -p 12345 -g sleep 30          # record for 30 seconds
```

### System-wide recording

```bash
perf record -a -g sleep 10                 # all CPUs for 10 seconds
```

### Specific events

```bash
perf record -e cache-misses -g ./myapp
perf record -e page-faults -g ./myapp
perf record -e cpu-clock -g -p 12345 sleep 5
```

### Call graph modes

```bash
perf record --call-graph dwarf ./myapp     # DWARF unwinding (most accurate)
perf record --call-graph fp ./myapp        # frame pointer (fast, needs -fno-omit-frame-pointer)
perf record --call-graph lbr ./myapp       # Last Branch Record (Intel, no overhead)
```

## perf report (Analysis)

### Interactive report

```bash
perf report                                # from perf.data
perf report --no-children                  # show self time only (not cumulative)
perf report --sort=dso                     # sort by shared library
perf report --stdio                        # text output (no TUI)
```

### Filter

```bash
perf report --dsos=myapp                   # filter to specific binary
perf report --symbols=hot_function         # filter to specific symbol
perf report --percent-limit 1              # hide entries below 1%
```

## perf annotate (Source-Level)

### Annotate hot functions

```bash
perf annotate                              # annotate from perf.data
perf annotate hot_function                 # annotate specific function
perf annotate --stdio                      # text output
```

Requires debug symbols (`-g` compiler flag or debuginfo packages).

## perf top (Live Profiling)

### Live CPU profiling

```bash
perf top                                   # system-wide, live updating
perf top -p 12345                          # specific process
perf top -g                                # with call graphs
perf top -e cache-misses                   # specific event
```

## perf probe (Dynamic Tracing)

### Add a probe

```bash
perf probe --add tcp_sendmsg               # kernel function entry
perf probe --add 'tcp_sendmsg size'        # with argument
perf probe -x /usr/bin/myapp --add my_func # user-space function
```

### List and remove probes

```bash
perf probe --list
perf probe --del tcp_sendmsg
```

### Record with probe

```bash
perf record -e probe:tcp_sendmsg -a sleep 5
perf script                                # view trace output
```

## perf script (Raw Trace Output)

### Dump raw samples

```bash
perf script                                # full trace output
perf script --header                       # include metadata
perf script -F time,pid,comm,sym           # select fields
```

## Flamegraphs

### Generate flamegraph from perf data

```bash
perf record -F 99 -g ./myapp
perf script > out.perf
# Using Brendan Gregg's FlameGraph tools:
stackcollapse-perf.pl out.perf > out.folded
flamegraph.pl out.folded > flamegraph.svg
```

### One-liner

```bash
perf record -F 99 -g -p 12345 sleep 30 && \
  perf script | stackcollapse-perf.pl | flamegraph.pl > flame.svg
```

### Off-CPU flamegraph

```bash
perf record -e sched:sched_switch -a -g sleep 10
perf script | stackcollapse-perf.pl | flamegraph.pl --color=io > offcpu.svg
```

## Common Events

### Hardware events

```bash
# cycles                CPU cycles
# instructions          instructions retired
# cache-references      cache accesses
# cache-misses          cache misses
# branch-instructions   branches
# branch-misses         branch mispredictions
# bus-cycles            bus cycles
```

### Software events

```bash
# cpu-clock             CPU clock
# task-clock            task clock
# page-faults           page faults
# context-switches      context switches
# cpu-migrations        CPU migrations
# minor-faults          minor page faults
# major-faults          major page faults (disk I/O)
```

### List available events

```bash
perf list                                  # all events
perf list hw                               # hardware events
perf list sw                               # software events
perf list tracepoint                       # kernel tracepoints
```

## Tips

- `perf record -g` with `--call-graph dwarf` gives the most accurate stacks. Frame pointer mode requires binaries compiled with `-fno-omit-frame-pointer`.
- `-F 99` (not 100) avoids lock-step aliasing with timer interrupts.
- `perf report --no-children` shows where time is actually spent (self), not cumulative time including callees.
- Flamegraphs are the most intuitive way to read profiling data. The x-axis is stack depth, width is sample count.
- `perf stat` is zero-overhead counting. Use it first to identify whether the bottleneck is CPU, cache, or branches.
- `perf top` is like `top` for functions. Good for live diagnosis of CPU-bound processes.
- Kernel symbols require `/proc/kallsyms` access (may need `sysctl kernel.kptr_restrict=0`).
- `perf record` writes to `perf.data` in the current directory. Specify `-o filename` to change.

## References

- [man perf(1)](https://man7.org/linux/man-pages/man1/perf.1.html)
- [man perf-record(1)](https://man7.org/linux/man-pages/man1/perf-record.1.html)
- [man perf-stat(1)](https://man7.org/linux/man-pages/man1/perf-stat.1.html)
- [man perf-report(1)](https://man7.org/linux/man-pages/man1/perf-report.1.html)
- [man perf-top(1)](https://man7.org/linux/man-pages/man1/perf-top.1.html)
- [perf Wiki](https://perf.wiki.kernel.org/index.php/Main_Page)
- [perf Wiki — Tutorial](https://perf.wiki.kernel.org/index.php/Tutorial)
- [Kernel perf Documentation](https://www.kernel.org/doc/html/latest/admin-guide/perf-security.html)
- [Brendan Gregg — perf Examples](https://www.brendangregg.com/perf.html)
- [Brendan Gregg — Linux perf Flame Graphs](https://www.brendangregg.com/FlameGraphs/cpuflamegraphs.html)
- [Red Hat — Performance Observability with perf](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/getting-started-with-perf_monitoring-and-managing-system-status-and-performance)
- [Arch Wiki — Perf](https://wiki.archlinux.org/title/Perf)
