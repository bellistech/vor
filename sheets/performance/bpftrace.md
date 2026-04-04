# bpftrace (Dynamic Tracing Language)

High-level tracing language for Linux using eBPF, for quick one-liners and scripts.

## One-Liners

### List available probes

```bash
bpftrace -l 'tracepoint:syscalls:*'
bpftrace -l 'kprobe:tcp_*'
bpftrace -l 'uprobe:/usr/bin/bash:*'
```

### Trace new processes

```bash
bpftrace -e 'tracepoint:syscalls:sys_enter_execve { printf("%s %s\n", comm, str(args.filename)); }'
```

### Count system calls by process

```bash
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[comm] = count(); }'
```

### Trace file opens

```bash
bpftrace -e 'tracepoint:syscalls:sys_enter_openat { printf("%s %s\n", comm, str(args.filename)); }'
```

### Trace TCP connections

```bash
bpftrace -e 'kprobe:tcp_connect { printf("%s -> %s\n", comm, ntop(((struct sock *)arg0)->__sk_common.skc_daddr)); }'
```

### Read latency histogram

```bash
bpftrace -e 'tracepoint:syscalls:sys_enter_read { @start[tid] = nsecs; }
tracepoint:syscalls:sys_exit_read /@start[tid]/ { @usecs = hist((nsecs - @start[tid]) / 1000); delete(@start[tid]); }'
```

## Probe Types

### Kernel probes

```bash
# kprobe:function_name         # kernel function entry
# kretprobe:function_name      # kernel function return
bpftrace -e 'kprobe:do_sys_open { printf("%s\n", comm); }'
bpftrace -e 'kretprobe:vfs_read { @bytes = hist(retval); }'
```

### User-space probes

```bash
# uprobe:/path/to/binary:function        # function entry
# uretprobe:/path/to/binary:function     # function return
bpftrace -e 'uprobe:/usr/bin/bash:readline { printf("readline called\n"); }'
bpftrace -e 'uretprobe:/usr/bin/bash:readline { printf("%s\n", str(retval)); }'
```

### Tracepoints (stable kernel interface)

```bash
# tracepoint:category:event
bpftrace -e 'tracepoint:block:block_rq_issue { printf("%s %d\n", comm, args.bytes); }'
bpftrace -e 'tracepoint:sched:sched_switch { printf("%s -> %s\n", args.prev_comm, args.next_comm); }'
bpftrace -e 'tracepoint:net:netif_rx { @[comm] = count(); }'
```

### Software events

```bash
# software:event:count
bpftrace -e 'software:page-faults:1 { @[comm] = count(); }'
bpftrace -e 'software:cpu-clock:1000000 { @[comm] = count(); }'
```

### Profile (sampling)

```bash
# profile:hz:frequency
bpftrace -e 'profile:hz:99 { @[kstack] = count(); }'    # kernel stack sampling
bpftrace -e 'profile:hz:99 { @[ustack] = count(); }'    # user stack sampling
bpftrace -e 'profile:hz:99 /comm == "myapp"/ { @[ustack] = count(); }'
```

### Interval and BEGIN/END

```bash
bpftrace -e 'interval:s:1 { printf("tick\n"); }'
bpftrace -e 'BEGIN { printf("tracing...\n"); } END { printf("done\n"); }'
```

## Builtins

```bash
# pid         process ID
# tid         thread ID
# uid         user ID
# comm        process name (16 char max)
# nsecs       nanosecond timestamp
# elapsed     nanoseconds since bpftrace start
# kstack      kernel stack trace
# ustack      user-space stack trace
# arg0-argN   function arguments (kprobe/uprobe)
# retval      return value (kretprobe/uretprobe)
# args        tracepoint arguments struct
# cpu         current CPU number
# cgroup      cgroup ID
# curtask     current task_struct pointer
```

## Maps (Aggregations)

### Count

```bash
bpftrace -e 'tracepoint:syscalls:sys_enter_read { @reads[comm] = count(); }'
```

### Sum

```bash
bpftrace -e 'tracepoint:syscalls:sys_exit_read /retval > 0/ { @bytes[comm] = sum(retval); }'
```

### Histogram

```bash
bpftrace -e 'tracepoint:syscalls:sys_exit_read /retval > 0/ { @size = hist(retval); }'
```

### Linear histogram

```bash
bpftrace -e 'kretprobe:vfs_read { @size = lhist(retval, 0, 10000, 1000); }'
```

### Min / Max / Avg

```bash
bpftrace -e 'kretprobe:vfs_read { @min_bytes = min(retval); @max_bytes = max(retval); @avg_bytes = avg(retval); }'
```

### Stats (count, avg, total)

```bash
bpftrace -e 'kretprobe:vfs_read { @bytes = stats(retval); }'
```

### Clear maps periodically

```bash
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[comm] = count(); }
interval:s:5 { print(@); clear(@); }'
```

## Filters

### Filter by process name

```bash
bpftrace -e 'kprobe:vfs_read /comm == "nginx"/ { @[comm] = count(); }'
```

### Filter by PID

```bash
bpftrace -e 'kprobe:vfs_read /pid == 12345/ { @bytes = hist(retval); }'
```

### Filter by UID

```bash
bpftrace -e 'tracepoint:syscalls:sys_enter_openat /uid == 1000/ { printf("%s\n", str(args.filename)); }'
```

## Common Scripts

### Syscall count by process (top-like)

```bash
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[comm] = count(); }
interval:s:5 { print(@, 10); clear(@); }'
```

### Block I/O latency

```bash
bpftrace -e 'tracepoint:block:block_rq_issue { @start[args.dev, args.sector] = nsecs; }
tracepoint:block:block_rq_complete /@start[args.dev, args.sector]/ {
    @usecs = hist((nsecs - @start[args.dev, args.sector]) / 1000);
    delete(@start[args.dev, args.sector]);
}'
```

### DNS lookups

```bash
bpftrace -e 'uprobe:/lib/x86_64-linux-gnu/libc.so.6:getaddrinfo {
    printf("%s looking up %s\n", comm, str(arg0));
}'
```

### Malloc size distribution

```bash
bpftrace -e 'uprobe:/lib/x86_64-linux-gnu/libc.so.6:malloc {
    @size = hist(arg0);
}'
```

### Context switch tracing

```bash
bpftrace -e 'tracepoint:sched:sched_switch {
    @[args.prev_comm] = count();
}'
```

## Tips

- Run bpftrace as root or with `CAP_BPF` + `CAP_PERFMON` capabilities.
- `bpftrace -l` lists probes. Pipe through grep to find what you need.
- Maps (`@`) are automatically printed on exit. Name them descriptively: `@bytes[comm]`.
- `hist()` produces power-of-2 histograms. `lhist()` produces linear histograms with custom buckets.
- `kstack` and `ustack` capture stack traces. Pipe output to FlameGraph tools for visualization.
- Tracepoints are stable across kernel versions. Prefer them over kprobes when available.
- Filter early (`/comm == "myapp"/`) to reduce overhead on busy systems.
- bpftrace scripts (`.bt` files) can be run with `bpftrace script.bt` for reuse.

## See Also

- ebpf, perf, strace, kernel, prometheus

## References

- [bpftrace Reference Guide](https://github.com/bpftrace/bpftrace/blob/master/docs/reference_guide.md)
- [bpftrace One-Liners Tutorial](https://github.com/bpftrace/bpftrace/blob/master/docs/tutorial_one_liners.md)
- [bpftrace GitHub Repository](https://github.com/bpftrace/bpftrace)
- [bpftrace Internals](https://github.com/bpftrace/bpftrace/blob/master/docs/internals_development.md)
- [man bpftrace(8)](https://man7.org/linux/man-pages/man8/bpftrace.8.html)
- [Kernel BPF Documentation](https://www.kernel.org/doc/html/latest/bpf/)
- [eBPF.io — bpftrace](https://ebpf.io/projects/#bpftrace)
- [Brendan Gregg — bpftrace Cheat Sheet](https://www.brendangregg.com/BPF/bpftrace-cheat-sheet.html)
- [Brendan Gregg — BPF Performance Tools](https://www.brendangregg.com/bpf-performance-tools-book.html)
