# bpftrace (High-Level Tracing Language for eBPF)

A compact awk-inspired DSL that compiles to eBPF and attaches to kprobes, uprobes, tracepoints, USDT, profile timers, and software/hardware events for ad-hoc Linux performance analysis and dynamic instrumentation.

## Setup

### What bpftrace is (and is not)

bpftrace is a high-level tracing language for Linux eBPF. You write short scripts in a DSL (one-liners or `.bt` files) and bpftrace compiles them into BPF bytecode, loads them into the kernel, attaches them to probes, collects data into BPF maps, and prints results. It is the rapid-prototyping alternative to writing raw eBPF with libbpf or BCC.

It is Linux-only. It does not run on macOS, FreeBSD, Windows, or any non-Linux kernel. The macOS workaround is Lima/colima/UTM/Docker Desktop with a Linux VM, OrbStack, or a remote Linux box. WSL2 works (kernel ships with BPF and BTF on recent Windows builds).

### Install

```bash
# Debian/Ubuntu
sudo apt-get update
sudo apt-get install -y bpftrace

# Fedora/RHEL/Rocky/Alma
sudo dnf install -y bpftrace

# Arch
sudo pacman -S bpftrace

# Alpine
sudo apk add bpftrace

# openSUSE
sudo zypper install bpftrace

# macOS via Lima (Linux VM — bpftrace itself runs inside the VM)
brew install lima
limactl start --name=ebpf template://ubuntu
limactl shell ebpf -- sudo apt-get install -y bpftrace

# Build from source (when distro version is too old)
git clone https://github.com/bpftrace/bpftrace
cd bpftrace
mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
make -j$(nproc)
sudo make install
```

### Verify install

```bash
bpftrace --version
# bpftrace v0.21.2 (or similar)

# Smoke test: run BEGIN+END for one second
bpftrace -e 'BEGIN { printf("ok\n"); } interval:s:1 { exit(); }'

# Confirm BPF + BTF support
ls -l /sys/kernel/btf/vmlinux
# -r--r--r-- 1 root root 5234567 Jan  1 00:00 /sys/kernel/btf/vmlinux
```

### Kernel + headers + libbpf

bpftrace requires:

- Linux kernel ≥ 4.9 (minimum), but realistically ≥ 5.4 for usable feature set.
- Kernel built with `CONFIG_BPF=y`, `CONFIG_BPF_SYSCALL=y`, `CONFIG_BPF_JIT=y`, `CONFIG_HAVE_EBPF_JIT=y`, `CONFIG_BPF_EVENTS=y`, `CONFIG_KPROBES=y`, `CONFIG_UPROBES=y`, `CONFIG_DEBUG_INFO_BTF=y` (BTF), `CONFIG_FTRACE=y`, `CONFIG_FTRACE_SYSCALLS=y`.
- libbpf (statically linked into bpftrace by default).
- BTF at `/sys/kernel/btf/vmlinux` (or matching kernel-headers package). Kernels ≥ 5.5 typically ship BTF.
- For uprobes: target binary must have symbols (not stripped) or be matched via DWARF/PLT.

### Capabilities and privileges

bpftrace traditionally needs root (`sudo bpftrace …`). Modern kernels (≥ 5.8) split BPF privileges:

```bash
# CAP_BPF: load BPF programs
# CAP_PERFMON: attach to perf-style probes (kprobe/uprobe/profile)
# CAP_NET_ADMIN: required for some networking probe types

# Grant on the binary
sudo setcap cap_bpf,cap_perfmon,cap_net_admin=eip $(which bpftrace)

# Run unprivileged
bpftrace -e 'kprobe:vfs_read { @[comm] = count(); exit(); }'
```

For older kernels (< 5.8), there is only `CAP_SYS_ADMIN`, which is effectively root.

### Version cadence

bpftrace versions ship roughly every 6–10 weeks. Major features by version:

- v0.7 — first official release with stable language (2018).
- v0.10 — `loop` (bounded), better string handling.
- v0.12 — explicit type system (`uint64`, `int8`, `string`).
- v0.13 — BTF integration; kfunc/kretfunc probes.
- v0.16 — `iter:task` and `iter:task_file` (BPF iterators).
- v0.17 — improved CO-RE; `for` loops over maps.
- v0.20+ — better perf and stability; structured ahead-of-time compilation.

Always check `bpftrace --version` and the upstream changelog when a feature in this sheet seems missing.

## The bpftrace Language

### Syntax overview

bpftrace is an awk-inspired DSL: a script is a list of `probe { action }` blocks. The language is C-like in its expressions but tracing-specific: maps with `@`, scratch variables with `$`, predicates with `/cond/`, and a small standard library of built-ins.

```bash
bpftrace -e '
BEGIN { printf("Tracing... Hit Ctrl-C to end.\n"); }
kprobe:vfs_read /comm == "myapp"/ { @reads[pid] = count(); }
END { print(@reads); clear(@reads); }
'
```

### Comparison with raw eBPF / libbpf / BCC

| Tool | Language | Compile | Boilerplate | Speed of dev | Speed of run |
|---|---|---|---|---|---|
| Raw eBPF (asm/C) | BPF asm or restricted C | LLVM | Hundreds of LOC | Slow | Fastest |
| libbpf + CO-RE | C + BTF | clang -target bpf | ~50 LOC + skeleton | Medium | Very fast |
| BCC | Python + C templates | LLVM at runtime | ~30 LOC | Fast | Slow start (recompile) |
| bpftrace | DSL | LLVM at runtime | One line possible | Fastest | Fast for typical probes |

bpftrace is the fastest path from "I want to know X" to "data on screen". For long-lived productionized tools, port to libbpf+CO-RE (skeleton-based, no runtime LLVM dependency).

### Canonical script structure

```bash
#!/usr/bin/env bpftrace
// optional shebang; chmod +x and run directly

BEGIN {
    printf("%-8s %-16s %s\n", "PID", "COMM", "EVENT");
}

kprobe:vfs_read {
    @start[tid] = nsecs;
}

kretprobe:vfs_read /@start[tid]/ {
    $delta = nsecs - @start[tid];
    @latency[comm] = hist($delta);
    delete(@start[tid]);
}

END {
    print(@latency);
    clear(@latency);
    clear(@start);
}
```

Three phases: setup (`BEGIN`), per-event work (one or more probe blocks), teardown (`END`). The `@map` syntax accumulates results; `print` dumps them.

## Probe Types — Catalog

### kprobe / kretprobe — kernel function entry/exit

Attach to any non-inlined kernel function. Wildcards allowed.

```bash
kprobe:vfs_read              # entry of vfs_read()
kretprobe:vfs_read           # return of vfs_read()
kprobe:tcp_*                 # all kernel functions starting with tcp_
kprobe:do_sys_open*          # variants like do_sys_openat2
```

### uprobe / uretprobe — userspace function entry/exit

Attach to any symbol in a user binary or shared library.

```bash
uprobe:/usr/bin/bash:readline
uretprobe:/usr/lib/x86_64-linux-gnu/libc.so.6:malloc
uprobe:./myapp:my_function
uprobe:/usr/lib/libssl.so.3:SSL_read
```

### tracepoint — stable kernel tracepoints

Statically compiled-in trace points. Stable ABI across kernel versions. Always prefer over kprobe when available.

```bash
tracepoint:syscalls:sys_enter_open
tracepoint:syscalls:sys_enter_openat
tracepoint:sched:sched_switch
tracepoint:block:block_rq_issue
tracepoint:net:net_dev_xmit
tracepoint:raw_syscalls:sys_enter      # all syscall entry
tracepoint:raw_syscalls:sys_exit       # all syscall exit
```

### usdt — userspace statically-defined tracepoints

USDT probes are explicit instrumentation points compiled into a binary (sys/sdt.h, dtrace -h). PostgreSQL, MySQL, Python (PEP 669), Ruby, Node, libc, Java JVM all ship USDT.

```bash
usdt:/usr/lib/postgresql/16/bin/postgres:postgresql:transaction__start
usdt:/path/to/python3:python:function__entry
usdt:./myapp:myprovider:myprobe
```

### profile / interval — timer probes

Periodic timer-driven probes (no event needed). Runs on every CPU at the rate specified, or globally.

```bash
profile:hz:99                # 99 times/sec on every CPU (flamegraph standard)
profile:s:1                  # once per second on every CPU
profile:ms:10                # every 10ms on every CPU
profile:us:100               # every 100us on every CPU

interval:s:5                 # globally once every 5 seconds (printf-style)
interval:ms:100
```

### software / hardware events — perf events

Backed by Linux perf_events. Sample on counter overflow.

```bash
software:cpu-clock:1000000           # CPU clock, every 1ms
software:page-faults:1               # every page fault
software:context-switches:1
software:cpu-migrations:1
software:minor-faults:1
software:major-faults:1
software:emulation-faults:1
software:alignment-faults:1
software:dummy:1

hardware:cycles:1000000              # CPU cycles
hardware:instructions:1000000
hardware:cache-misses:10000
hardware:branches:1000000
hardware:branch-misses:10000
hardware:bus-cycles:1000000
hardware:cache-references:10000
hardware:ref-cycles:1000000
```

### BEGIN / END — script lifecycle

```bash
BEGIN { printf("hello\n"); }     # runs once at script start
END   { printf("bye\n"); }       # runs once at script exit (Ctrl-C, exit(), or signal)
```

### kfunc / kretfunc — BTF-based kernel function probes (5.11+)

Stable, BTF-typed kernel function probes (faster than kprobe; typed args).

```bash
kfunc:vfs_read              # typed entry probe
kretfunc:vfs_read           # typed return probe
```

### iter — BPF iterators (5.12+)

Walk kernel state without per-event probing.

```bash
iter:task                   # walk every task_struct
iter:task_file              # walk every open fd of every task
iter:task_vma               # walk every VMA of every task (5.18+)
```

## Probe Syntax

### Function probes — exact, prefix, glob

```bash
# Exact match
bpftrace -e 'kprobe:vfs_read { printf("vfs_read by %s\n", comm); }'

# Glob/wildcard — match many at once
bpftrace -e 'kprobe:vfs_* { @[probe] = count(); }'
bpftrace -e 'kprobe:tcp_send* { @[probe] = count(); }'

# Multiple probes share an action
bpftrace -e 'kprobe:vfs_read,kprobe:vfs_write { @[probe, comm] = count(); }'
```

### Tracepoint probes

```bash
bpftrace -e 'tracepoint:syscalls:sys_enter_open { printf("%s: %s\n", comm, str(args.filename)); }'

# Glob
bpftrace -e 'tracepoint:syscalls:sys_enter_* { @[probe] = count(); }'

# All syscalls (one probe)
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[args.id] = count(); }'
```

### Userspace probes

```bash
# Library probe (PID-agnostic — attaches to every process loading libc)
bpftrace -e 'uprobe:/lib/x86_64-linux-gnu/libc.so.6:malloc { @ = hist(arg0); }'

# Specific binary
bpftrace -e 'uretprobe:./bin/myapp:compute { @rv = hist(retval); }'

# USDT
bpftrace -e 'usdt:/path/to/postgres:postgresql:transaction__start { printf("xact begin\n"); }'

# Attach to one running process
bpftrace -p 12345 -e 'uprobe:/path/to/lib:func { @ = count(); }'
```

### Timer probes

```bash
bpftrace -e 'profile:hz:99 { @[ustack, comm] = count(); }'           # flamegraph data
bpftrace -e 'profile:hz:99 /pid == 1234/ { @[ustack] = count(); }'    # one PID
bpftrace -e 'interval:s:5 { time("%H:%M:%S "); print(@); clear(@); }' # report every 5s
```

### Software / hardware events

```bash
bpftrace -e 'software:cpu-clock:1000000 { @[comm] = count(); }'      # 1ms CPU sample
bpftrace -e 'software:page-faults:1 { @[comm] = count(); }'          # every page fault
bpftrace -e 'hardware:cache-misses:10000 { @[ustack] = count(); }'   # every 10k cache miss
```

## Built-in Variables

```bash
# Process / thread
pid           # current process ID (TGID)
tid           # current thread ID (LWP)
uid           # user ID
gid           # group ID
comm          # process name (16-byte truncated, like /proc/<pid>/comm)
nspid         # PID in current namespace (≥ v0.13)
nsname        # namespace name (≥ v0.13)
cgroup        # current cgroup ID
curtask       # pointer to current task_struct (cast and walk for advanced data)

# CPU
cpu           # current CPU number
numaid        # NUMA node ID

# Time
nsecs         # nanoseconds since boot (uses CLOCK_MONOTONIC)
elapsed       # nanoseconds since BEGIN

# Probe context
probe         # full probe name (e.g. "kprobe:vfs_read")
func          # current function name (kprobe/uprobe)

# Kernel/userspace function arguments
arg0, arg1, …, argN     # kprobe/uprobe positional args (uint64)
args                    # tracepoint args struct (typed); use args.field
retval                  # kretprobe/uretprobe return value (uint64)

# Stacks
kstack        # kernel stack trace
ustack        # user stack trace

# Script CLI args
$1, $2, …     # positional CLI args after the script
$#            # number of CLI args (≥ v0.16)

# Maps
@             # the unnamed/default global map
@name         # named map
@name[k]      # map indexed by key
```

### Examples for each builtin

```bash
bpftrace -e 'kprobe:do_sys_openat2 { printf("pid=%d tid=%d uid=%d comm=%s\n", pid, tid, uid, comm); }'
bpftrace -e 'profile:hz:1 { printf("cpu=%d nsecs=%llu elapsed=%llu\n", cpu, nsecs, elapsed); }'
bpftrace -e 'kprobe:vfs_read { printf("probe=%s func=%s\n", probe, func); }'
bpftrace -e 'kprobe:vfs_read { printf("arg0=%llx arg1=%llx arg2=%llu\n", arg0, arg1, arg2); }'
bpftrace -e 'tracepoint:syscalls:sys_enter_open { printf("file=%s\n", str(args.filename)); }'
bpftrace -e 'kretprobe:vfs_read { @rv = hist(retval); }'
```

## Variables

### Scratch variables — `$name` (per-thread, per-action)

Local to a single execution of a probe action. Per-thread, ephemeral, no persistence between events.

```bash
bpftrace -e '
kprobe:vfs_read {
    $bytes = arg2;
    $msg = "reading";
    printf("%s %d bytes\n", $msg, $bytes);
}
'
```

### Map variables — `@name` (global)

Persisted in BPF maps. Visible across probes and across CPUs (per-CPU aggregation handled internally).

```bash
@count = 0;             # unnamed global (rare; use $ for scratch)
@reads = 0;
@reads += 1;
@reads[comm] = count(); # keyed map
@stats[pid] = stats(arg2);
```

### Typed maps and explicit declarations

```bash
// Most maps are inferred from first assignment; force a type:
@hist[comm] = hist(arg2);          // histogram-per-comm
@avg[pid]  = avg(arg2);            // running avg per pid
@cnt[probe, comm] = count();       // multi-key

// Explicit type (≥ v0.12):
let @counts: hash<string, uint64> = (hash<string, uint64>) 0;
```

### Canonical timing pattern

```bash
bpftrace -e '
kprobe:vfs_read {
    @start[tid] = nsecs;
}
kretprobe:vfs_read /@start[tid]/ {
    @ns[comm] = hist(nsecs - @start[tid]);
    delete(@start[tid]);
}
'
```

### `@` alias for unnamed map

```bash
bpftrace -e 'kprobe:vfs_read { @ = count(); }'    # equivalent to @[/* none */] = count();
bpftrace -e 'kretprobe:vfs_read { @ = hist(retval); }'
```

## Functions — printf and Friends

### printf

C-style format string. Supported conversions: `%d %u %x %llx %s %c %p %f`.

```bash
printf("pid=%d comm=%s\n", pid, comm);
printf("rate=%.2f%%\n", $rate);              // floats — limited; prefer integer math
printf("%-8d %-16s %s\n", pid, comm, str(args.filename));
```

### print

Print a map or scalar. Special-cased for histograms (renders as buckets).

```bash
print(@latency);        // dump entire map
print(@latency, 10);    // top 10 entries
print(123);             // print scalar
```

### clear / delete / zero

```bash
clear(@map);            // remove all keys
delete(@map[key]);      // remove one key
zero(@map);             // zero all values, keep keys
```

### exit

```bash
exit();                 // terminate bpftrace (runs END)
```

### time / strftime

```bash
time();                          // prints "HH:MM:SS\n"
time("%Y-%m-%d %H:%M:%S\n");     // strftime-style
strftime("%Y-%m-%dT%H:%M:%S", nsecs)   // returns string (≥ v0.14)
```

### system / cat

```bash
system("date");                                  // run shell command (privileged!)
cat("/proc/%d/cmdline", pid);                    // print file contents
cat("/sys/kernel/debug/tracing/trace");
```

`system()` is gated behind `--unsafe`. Do not enable in production.

## Functions — Aggregations

The killer feature. Aggregations are merged in-kernel via per-CPU maps and rendered on dump.

```bash
count()                  // count events
sum(N)                   // running sum
avg(N)                   // running average
min(N) / max(N)          // running min/max
stats(N)                 // count + avg + total — three at once
hist(N)                  // power-of-2 histogram
lhist(N, min, max, step) // linear histogram
```

### Examples

```bash
// count
bpftrace -e 'kprobe:vfs_read { @reads[comm] = count(); }'

// sum (bytes)
bpftrace -e 'kretprobe:vfs_read /retval > 0/ { @bytes[comm] = sum(retval); }'

// avg + min + max
bpftrace -e 'kretprobe:vfs_read { @avg[comm] = avg(retval); @min[comm] = min(retval); @max[comm] = max(retval); }'

// stats — combines count, avg, total
bpftrace -e 'kretprobe:vfs_read { @stats[comm] = stats(retval); }'

// power-of-2 histogram (canonical latency dump)
bpftrace -e '
kprobe:vfs_read { @start[tid] = nsecs; }
kretprobe:vfs_read /@start[tid]/ {
    @latency_us = hist((nsecs - @start[tid]) / 1000);
    delete(@start[tid]);
}'

// linear histogram with custom buckets
bpftrace -e 'kretprobe:vfs_read { @size = lhist(retval, 0, 1000000, 50000); }'
```

### Canonical "latency-on-exit" idiom

```bash
bpftrace -e '
kprobe:vfs_read { @start[tid] = nsecs; }
kretprobe:vfs_read /@start[tid]/ {
    @ns = hist(nsecs - @start[tid]);
    delete(@start[tid]);
}
END { print(@ns); clear(@ns); clear(@start); }
'
```

bpftrace prints `@ns` automatically on END if you do not. The histogram renders as ASCII buckets — wire-ready for terminals.

## Functions — String Manipulation

```bash
str(addr)                // copy a NUL-terminated C string from kernel/user pointer
str(addr, len)           // bounded length copy
buf(addr, len)           // raw byte buffer (hex-printable)
strncmp(s1, s2, n)       // 0 == match, 1 == differ
strcontains(s1, s2)      // 1 if s2 found in s1, else 0   (≥ v0.18)
strerror(errno)          // "No such file or directory" etc. (≥ v0.21)
```

### Examples

```bash
// safe string from user pointer
bpftrace -e 'tracepoint:syscalls:sys_enter_open { printf("%s\n", str(args.filename)); }'

// fixed-length read (avoid overflow)
bpftrace -e 'uprobe:/bin/bash:readline { $cmd = str(retval, 64); printf("%s\n", $cmd); }'

// raw hex
bpftrace -e 'kprobe:tcp_v4_connect { printf("sock=%r\n", buf(arg0, 16)); }'

// equality check (== on strings is unsupported on older bpftrace; use strncmp)
bpftrace -e 'tracepoint:syscalls:sys_enter_openat /strncmp(str(args.filename), "/etc/", 5) == 0/ {
    printf("etc-access by %s\n", comm);
}'

// substring search
bpftrace -e 'tracepoint:syscalls:sys_enter_openat /strcontains(str(args.filename), ".so")/ {
    @[comm] = count();
}'
```

## Functions — Stack Traces

```bash
kstack          // kernel stack
kstack(N)       // kernel stack, max N frames
ustack          // user stack
ustack(N)       // user stack, max N frames
kstack(perf)    // perf-style format
ustack(perf, 10)
```

### Examples

```bash
// Stack-counting profile (basis for flamegraphs)
bpftrace -e 'profile:hz:99 { @[kstack] = count(); }'
bpftrace -e 'profile:hz:99 { @[ustack] = count(); }'
bpftrace -e 'profile:hz:99 { @[kstack, ustack, comm] = count(); }'

// Where in the kernel are TCP packets dropped?
bpftrace -e 'kprobe:kfree_skb { @[kstack] = count(); }'

// What user stacks lead to malloc()?
bpftrace -e 'uprobe:/lib/x86_64-linux-gnu/libc.so.6:malloc { @[ustack] = count(); }'

// Limit depth
bpftrace -e 'profile:hz:99 { @[ustack(8)] = count(); }'
```

### Symbol resolution caveat

User stacks need debug symbols or BTF for the userspace binary, otherwise frames render as `0xdeadbeef` or `[unknown]`. Solutions:

```bash
# Install -dbgsym packages (Debian/Ubuntu)
sudo apt-get install libc6-dbgsym

# Build app with frame pointers (mandatory for ustack on x86_64)
go build -gcflags='-m' ...                       # Go: keeps frame pointers by default since 1.18
gcc -fno-omit-frame-pointer ...                  # C/C++: needed for stack walking
rustc -C force-frame-pointers=yes ...            # Rust

# DWARF unwinding (≥ kernel 5.5, with libdw / libunwind)
# bpftrace will use frame pointers first; fallback to DWARF if compiled with --enable-libdw
```

## Functions — Symbols

Address ↔ symbol-name conversion.

```bash
ksym(addr)        // kernel address → symbol (resolve from /proc/kallsyms)
usym(addr)        // user address → symbol (requires -p PID or attached uprobe)
kaddr("name")     // symbol → kernel address
uaddr("name")     // userspace symbol → address  (≥ v0.10)
```

### Examples

```bash
// Resolve a function pointer
bpftrace -e 'kprobe:do_softirq { printf("at %s\n", ksym(arg0)); }'

// Track which file_operations is dispatching reads
bpftrace -e 'kprobe:vfs_read { printf("file=%s op=%s\n", comm, ksym(((struct file *)arg0)->f_op->read)); }'

// Patch / hook detection (read-only): print address of a known symbol
bpftrace -e 'BEGIN { printf("vfs_read at %p\n", kaddr("vfs_read")); exit(); }'
```

Resolution caveat: on busy systems with frequent module loads, symbols may resolve incorrectly if `/proc/kallsyms` changes mid-trace. For correctness, capture addresses raw and resolve them in `END` or in post-processing.

## Functions — Process and System

```bash
pid, tid, uid, gid, comm    // process info (covered above)
ntop(addr)                   // network address → string ("1.2.3.4" or "::1")
ntop(af, addr)               // explicit address family (AF_INET / AF_INET6)
uptime()                     // seconds since boot
getopt("flag", default)      // (rare) read CLI flag
system("cmd")                // run shell command (--unsafe)
cat("path")                  // print file contents
join(arr)                    // join string array (e.g. argv)
print(scalar_or_map)
```

### Examples

```bash
// IPv4 connect destination
bpftrace -e 'kprobe:tcp_v4_connect {
    $sk = (struct sock *)arg0;
    printf("%s -> %s\n", comm, ntop($sk->__sk_common.skc_daddr));
}'

// IPv6 explicit
bpftrace -e 'kprobe:tcp_v6_connect {
    $sk = (struct sock *)arg0;
    printf("%s -> %s\n", comm, ntop(AF_INET6, $sk->__sk_common.skc_v6_daddr.in6_u.u6_addr8));
}'

// argv join (≥ v0.13)
bpftrace -e 'tracepoint:syscalls:sys_enter_execve {
    join(args.argv);
}'

// Print a /proc file when condition met
bpftrace -e 'kprobe:do_exit /comm == "doomed"/ {
    cat("/proc/%d/status", pid);
}'
```

## Functions — Time

```bash
nsecs                            // nanoseconds since boot (raw)
elapsed                          // nanoseconds since BEGIN
time()                           // print "HH:MM:SS\n"
time("%Y-%m-%d %H:%M:%S")        // strftime-style
strftime("%H:%M:%S", nsecs)      // returns string (≥ v0.14)
```

### Canonical latency-measurement pattern

```bash
bpftrace -e '
BEGIN { printf("Tracing read latency (us). Hit Ctrl-C.\n"); }
kprobe:vfs_read { @start[tid] = nsecs; }
kretprobe:vfs_read /@start[tid]/ {
    $delta_us = (nsecs - @start[tid]) / 1000;
    @us = hist($delta_us);
    delete(@start[tid]);
}
END { print(@us); clear(@us); clear(@start); }
'
```

### Time-stamping output rows

```bash
bpftrace -e '
tracepoint:syscalls:sys_enter_openat {
    time("%H:%M:%S ");
    printf("%s %s\n", comm, str(args.filename));
}'
```

## Functions — Signal

Send a UNIX signal to the traced process.

```bash
signal("SIGSTOP")
signal("SIGKILL")
signal(9)           // numeric form
signal("SIGUSR1")
exit()              // terminate bpftrace itself
```

### Examples

```bash
// Stop the process the moment it does something forbidden
bpftrace -e '
tracepoint:syscalls:sys_enter_unlink /comm == "sus"/ {
    printf("unlink by sus, stopping\n");
    signal("SIGSTOP");
}'

// Auto-stop bpftrace after 10 seconds
bpftrace -e 'BEGIN { printf("starting\n"); } interval:s:10 { exit(); }'

// Kill a misbehaving process
bpftrace -e 'kprobe:do_sys_open /uid == 0 && comm == "evil"/ { signal("SIGKILL"); }'
```

`signal()` is gated by `--unsafe`. Do not enable in production unless you know what you're doing.

## Maps and Aggregates

### Maps as hash tables

`@name[key1, key2, …] = aggregator()` builds a hash table keyed on the tuple, aggregated per-CPU and merged at print time.

```bash
@counts[comm] = count();
@latencies[pid] = hist(elapsed);
@bytes[comm, args.fd] = sum(arg2);
@by_cpu[cpu, probe] = count();
```

### Multi-key

```bash
bpftrace -e 'kretprobe:vfs_read { @[comm, retval > 0] = count(); }'
bpftrace -e 'tracepoint:sched:sched_switch { @[args.prev_comm, args.next_comm] = count(); }'
```

### Per-CPU implicit aggregation

bpftrace allocates per-CPU maps for aggregations; values merge on `print`/`END`. There is no contention between CPUs at probe time, so high-frequency probes scale.

### Dump-on-END automatic behavior

When the script exits (Ctrl-C, `exit()`, or signal), bpftrace prints all maps in declaration order, then runs `END`. If you don't want that, `clear()` them before `END`.

```bash
END {
    print(@counts);     // explicit dump (controls order/format)
    clear(@counts);
    clear(@start);      // suppress auto-print
}
```

## Predicates

The `/cond/` between probe and action is the predicate. Block runs only when predicate is non-zero.

```bash
kprobe:vfs_read /pid == 1234/ { @reads = count(); }
kprobe:vfs_read /comm == "myapp"/ { … }
kprobe:vfs_read /uid != 0/ { … }
kretprobe:vfs_read /retval < 0/ { @errors = count(); }
tracepoint:syscalls:sys_enter_openat /str(args.filename) == "/etc/passwd"/ { … }
```

### Combining conditions

```bash
/pid == 1234 && comm == "myapp"/
/comm == "nginx" || comm == "envoy"/
/(arg0 > 1000 && arg0 < 5000) || arg1 == 0/
```

### Filter by process

```bash
bpftrace -e 'kprobe:vfs_read /comm == "redis-server"/ { @ = hist(arg2); }'
bpftrace -e 'kprobe:vfs_read /pid == 1234/ { @ = hist(arg2); }'
```

### Filter via map presence (paired probes)

```bash
// Only fire kretprobe if the kprobe set a start time:
kprobe:vfs_read     { @start[tid] = nsecs; }
kretprobe:vfs_read /@start[tid]/ {
    @latency = hist(nsecs - @start[tid]);
    delete(@start[tid]);
}
```

## Conditionals and Control Flow

### if / else

```bash
if ($x > 100) {
    @big = count();
} else if ($x > 10) {
    @medium = count();
} else {
    @small = count();
}
```

### Ternary (limited)

```bash
$category = $x > 100 ? "big" : "small";
```

### while loops (kernel 5.3+ for bounded loops)

```bash
$i = 0;
while ($i < 10) {
    @arr[$i] = $i * 2;
    $i++;
}
```

The verifier requires bounded loops. Pre-5.3 you must unroll; bpftrace will do this for fixed iterations.

### unrolled / for loops

```bash
// Unroll a fixed-size loop
unroll(10) {
    @sum += 1;
}

// for over a map (≥ v0.17)
for ($kv : @map) {
    printf("k=%s v=%d\n", $kv.0, $kv.1);
}
```

### Verifier-friendly subset

The eBPF verifier rejects:
- Unbounded loops (pre-5.3 entirely; 5.3+ only with `bpf_loop` helper backing).
- Pointer arithmetic without explicit bounds checks.
- Stack > 512 bytes.
- Programs > 1M instructions (post 5.2; was 4096 before).

bpftrace handles most of this for you, but huge unrolled loops, oversized strings, or deeply nested map operations can still trip the verifier.

## Common One-Liners — Brendan Gregg's bpftrace Tools Catalog

The canonical "what does this in bpftrace" reference. Most of these are paste-runnable.

### Process / syscall — execsnoop / opensnoop / killsnoop

```bash
# execsnoop — trace new processes
bpftrace -e 'tracepoint:syscalls:sys_enter_execve {
    printf("%-8d %-16s %s\n", pid, comm, str(args.filename));
}'

# opensnoop — trace file opens
bpftrace -e 'tracepoint:syscalls:sys_enter_openat {
    printf("%-6d %-16s %s\n", pid, comm, str(args.filename));
}'

# killsnoop — trace signals sent
bpftrace -e 'tracepoint:syscalls:sys_enter_kill {
    printf("%s -> pid=%d sig=%d\n", comm, args.pid, args.sig);
}'

# top syscalls by count
bpftrace -e 'tracepoint:syscalls:sys_enter_* { @[probe] = count(); }'
```

### Per-process VFS read count

```bash
bpftrace -e 'kprobe:vfs_read { @[comm] = count(); }'
```

### Read latency histogram per process

```bash
bpftrace -e '
BEGIN { @start[tid] = 0; }
kprobe:vfs_read   { @start[tid] = nsecs; }
kretprobe:vfs_read /@start[tid]/ {
    @latency[comm] = hist(nsecs - @start[tid]);
    delete(@start[tid]);
}
'
```

### biolatency — block I/O latency

```bash
bpftrace -e '
tracepoint:block:block_rq_issue {
    @start[args.dev, args.sector] = nsecs;
}
tracepoint:block:block_rq_complete /@start[args.dev, args.sector]/ {
    @us = hist((nsecs - @start[args.dev, args.sector]) / 1000);
    delete(@start[args.dev, args.sector]);
}
'
```

### biosnoop — every block I/O

```bash
bpftrace -e '
tracepoint:block:block_rq_issue {
    @start[args.dev, args.sector] = nsecs;
    @comm[args.dev, args.sector] = comm;
}
tracepoint:block:block_rq_complete /@start[args.dev, args.sector]/ {
    $ms = (nsecs - @start[args.dev, args.sector]) / 1000000;
    printf("%s dev=%d sector=%d %dms\n",
        @comm[args.dev, args.sector], args.dev, args.sector, $ms);
    delete(@start[args.dev, args.sector]);
    delete(@comm[args.dev, args.sector]);
}
'
```

### runqlat — scheduler run-queue latency

```bash
bpftrace -e '
tracepoint:sched:sched_wakeup     { @start[args.pid] = nsecs; }
tracepoint:sched:sched_wakeup_new { @start[args.pid] = nsecs; }
tracepoint:sched:sched_switch /@start[args.next_pid]/ {
    @us = hist((nsecs - @start[args.next_pid]) / 1000);
    delete(@start[args.next_pid]);
}
'
```

### Off-CPU profiling

```bash
bpftrace -e '
tracepoint:sched:sched_switch /args.prev_state == 0/ {
    @off[args.prev_pid] = nsecs;
}
tracepoint:sched:sched_switch /@off[args.next_pid]/ {
    @offcpu[ustack] = hist(nsecs - @off[args.next_pid]);
    delete(@off[args.next_pid]);
}
'
```

### CPU profiling (flamegraph data)

```bash
bpftrace -e 'profile:hz:99 { @[ustack, comm] = count(); }'

# Pipe to FlameGraph (collapsed format):
bpftrace -e 'profile:hz:99 { @[ustack, comm] = count(); }' \
  > out.bt
# (post-process out.bt → flamegraph.pl)
```

### tcptracer / tcpconnect / tcpaccept

```bash
# tcpconnect — outbound TCP
bpftrace -e '
kprobe:tcp_v4_connect {
    $sk = (struct sock *)arg0;
    @[comm, ntop($sk->__sk_common.skc_daddr)] = count();
}
'

# tcpaccept — inbound TCP
bpftrace -e '
kretprobe:inet_csk_accept /retval/ {
    $sk = (struct sock *)retval;
    printf("%s accepted from %s\n", comm, ntop($sk->__sk_common.skc_daddr));
}
'

# tcptracer — both
bpftrace -e '
kprobe:tcp_v4_connect { @conn[pid] = arg0; }
kretprobe:tcp_v4_connect /retval == 0 && @conn[pid]/ {
    $sk = (struct sock *)@conn[pid];
    printf("%-6d %-16s -> %s\n", pid, comm, ntop($sk->__sk_common.skc_daddr));
    delete(@conn[pid]);
}
'
```

### Per-process TCP send bytes

```bash
bpftrace -e '
kprobe:tcp_sendmsg {
    @bytes[comm] = sum(arg2);
}
'
```

### tcpretrans — TCP retransmits

```bash
bpftrace -e '
tracepoint:tcp:tcp_retransmit_skb {
    @[comm] = count();
}
'
```

### oomkill — OOM killer

```bash
bpftrace -e '
kprobe:oom_kill_process {
    $oc = (struct oom_control *)arg0;
    time("%H:%M:%S ");
    printf("OOM kill: %s pid=%d\n", $oc->chosen->comm, $oc->chosen->pid);
}
'
```

### tcptop / nettop equivalents (per-process bytes)

```bash
bpftrace -e '
kprobe:tcp_sendmsg { @send[comm] = sum(arg2); }
kprobe:tcp_recvmsg { @recv[comm] = sum(arg2); }
interval:s:1 {
    time("%H:%M:%S\n");
    print(@send); clear(@send);
    print(@recv); clear(@recv);
}
'
```

### File system top by bytes

```bash
bpftrace -e '
kprobe:vfs_read,kprobe:vfs_write {
    @[probe, comm] = sum(arg2);
}
'
```

### USDT — Postgres queries

```bash
bpftrace -e '
usdt:/usr/lib/postgresql/16/bin/postgres:postgresql:query__start {
    printf("query: %s\n", str(arg0));
}
'
```

### Memory — page faults

```bash
bpftrace -e '
software:page-faults:1 {
    @[comm, ustack] = count();
}
'
```

### Profile + report every N seconds

```bash
bpftrace -e '
profile:hz:99 { @[comm] = count(); }
interval:s:10 {
    time("%H:%M:%S\n");
    print(@, 10);    // top 10
    clear(@);
}
'
```

## Multi-Script — Files

### Running scripts

```bash
bpftrace script.bt                     # run a script file
bpftrace -e '<one-liner>'              # inline
bpftrace -p 12345 script.bt            # attach uprobes to one PID
bpftrace -c "./myapp arg1 arg2" script.bt   # spawn target, trace, cleanup on exit
bpftrace -B line script.bt             # line-buffered output
bpftrace -B none script.bt             # unbuffered
bpftrace -d script.bt                  # debug: dump LLVM IR
bpftrace -v script.bt                  # verbose: dump probe attachment, BTF info
```

### Shebang for self-running .bt files

```bash
#!/usr/bin/env bpftrace

BEGIN {
    printf("Tracing read latency. Ctrl-C to end.\n");
}

kprobe:vfs_read { @start[tid] = nsecs; }
kretprobe:vfs_read /@start[tid]/ {
    @ns[comm] = hist(nsecs - @start[tid]);
    delete(@start[tid]);
}

END {
    print(@ns);
    clear(@start);
}
```

```bash
chmod +x runqlat.bt
sudo ./runqlat.bt
```

### Canonical multi-section script — runqlat.bt

```bash
#!/usr/bin/env bpftrace
//
// runqlat.bt — measure scheduler run-queue latency in microseconds
//

BEGIN {
    printf("Tracing run-queue latency... Hit Ctrl-C to end.\n");
}

tracepoint:sched:sched_wakeup     { @start[args.pid] = nsecs; }
tracepoint:sched:sched_wakeup_new { @start[args.pid] = nsecs; }

tracepoint:sched:sched_switch /@start[args.next_pid]/ {
    @us = hist((nsecs - @start[args.next_pid]) / 1000);
    delete(@start[args.next_pid]);
}

END {
    print(@us);
    clear(@us);
    clear(@start);
}
```

## Bcc-vs-bpftrace Decision

| Concern | bpftrace | BCC |
|---|---|---|
| Syntax | DSL (~50 line scripts) | C + Python (~300 LOC tools) |
| Compile | Runtime LLVM (every run) | Runtime LLVM (every run) |
| Speed of dev | Minutes | Hours |
| Speed of run | Fast for typical | Faster for complex/long programs |
| Persistent service | Awkward | OK |
| CO-RE | Yes (transparent) | Mixed (libbpf-tools yes, bcc python no) |
| Distribution | Single binary OK | Heavy (Python + clang + headers) |

**Workflow:** prototype in bpftrace → if you need to ship as a daemon or to many hosts, port to libbpf+CO-RE (the modern path; the bcc Python framework is in maintenance mode).

```bash
# bpftrace one-liner — 30 seconds
bpftrace -e 'tracepoint:syscalls:sys_enter_open { @[comm] = count(); }'

# BCC equivalent — ~50 lines, two files
# (see github.com/iovisor/bcc/blob/master/tools/opensnoop.py)

# libbpf+CO-RE — production-grade
# (see github.com/iovisor/bcc/blob/master/libbpf-tools/opensnoop.bpf.c)
```

## Limitations and Workarounds

### BPF stack limit (512 bytes)

Each program has 512 bytes of stack. Big strings, structs, arrays in scratch (`$`) variables can blow it.

**Workaround:** keep large data in maps (`@`), not scratch. Avoid `str(addr, 4096)`; use `str(addr, 64)` or whatever's actually needed.

### Bounded loops (5.3+ only)

Pre-5.3 kernels reject any loop. From 5.3, `bpf_loop` is allowed; bpftrace 0.16+ uses it transparently for `while` and `for`.

**Workaround for older kernels:** unroll manually (`unroll(N)`) or split work across multiple probes.

### No arbitrary string compare

`s1 == s2` historically didn't work in bpftrace. Newer versions partially support it via `strncmp`-equivalent codegen, but for portability use `strncmp(s1, s2, n)`.

### No floating point

eBPF has no FPU. All arithmetic is integer. `printf("%.2f", x)` requires bpftrace to do user-space formatting.

**Workaround:** scale by integer (`microseconds * 100 / total` for two-decimal percentage).

### Verifier program-too-large rejection

```
ERROR: BPF program too large. Processed 1000001 insn
```

Causes: too many helpers, too many unrolled iterations, deeply nested ifs.

**Workaround:** split into multiple smaller probes; remove unused branches; upgrade kernel (limit grew from 4096 to 1M).

### Per-CPU map merge cost

Aggregations are per-CPU; `print(@map)` merges across all CPUs. On 256-core systems, the merge step itself takes time.

**Workaround:** print less frequently; use `interval:s:5` not `interval:s:1`.

### High-rate probe overhead

Per-event `printf` at 1M events/sec floods stdout and back-pressures the kernel. Output ring buffer fills, events get dropped (`bpftrace: lost N events`).

**Workaround:** aggregate via maps; print on `END` or `interval:s:N`.

## CO-RE and BTF

CO-RE (Compile-Once-Run-Everywhere) is the eBPF portability story. A bpftrace script written today runs on kernels from 4.18 to 6.10 unchanged, because:

- Type info comes from BTF (`/sys/kernel/btf/vmlinux`) at run time.
- libbpf rewrites field offsets for the target kernel.
- bpftrace embeds libbpf and does this transparently.

```bash
# Verify BTF
ls -l /sys/kernel/btf/vmlinux
# -r--r--r-- 1 root root 5234567 Jan  1 00:00

# List all kfunc probes (BTF-typed)
bpftrace -l 'kfunc:*' | head

# List all struct types known via BTF
bpftrace -lv 'struct task_struct' | head
```

### When BTF is missing

Distros built without `CONFIG_DEBUG_INFO_BTF=y` (e.g. older Ubuntu LTS, some embedded). Fix:

```bash
# Option 1: install kernel-headers matching running kernel
sudo apt-get install linux-headers-$(uname -r)

# Option 2: BTFhub — downloadable BTF for stock kernels
# https://github.com/aquasecurity/btfhub-archive
sudo cp ./5.4.0-77-generic.btf /sys/kernel/btf/vmlinux  # mock; usually loaded another way

# Option 3: build a kernel with CONFIG_DEBUG_INFO_BTF=y (server / cloud images often need this)
```

### Kernel version portability

```bash
# Probes that need ≥ specific kernel
kfunc:vfs_read       # 5.5+ (BTF) — typed, fast
iter:task            # 5.12+
iter:task_vma        # 5.18+
software:dummy:1     # always
profile:hz:N         # 4.9+

# Tracepoints — most are stable since 4.x
tracepoint:syscalls:sys_enter_*   # since 3.x
tracepoint:sched:sched_switch     # since 3.x
```

## Profile + Stack Trace Workflow

The canonical CPU profiling pipeline.

```bash
# Step 1: collect 30 seconds of stacks at 99Hz
sudo bpftrace -e '
profile:hz:99 /pid == '"$(pgrep myapp)"'/ {
    @[ustack] = count();
}
interval:s:30 { exit(); }
' > out.bt

# Step 2: convert to FlameGraph collapsed format
# (out.bt format is: stack frames + count; needs reformat)

# Step 3: render flamegraph
git clone https://github.com/brendangregg/FlameGraph
cd FlameGraph
cat ../out.bt | ./stackcollapse-bpftrace.pl | ./flamegraph.pl > flame.svg
xdg-open flame.svg
```

### Off-CPU flamegraph

```bash
sudo bpftrace -e '
tracepoint:sched:sched_switch /args.prev_state == 0/ {
    @off[args.prev_pid] = nsecs;
}
tracepoint:sched:sched_switch /@off[args.next_pid]/ {
    @offcpu[ustack, comm] = hist(nsecs - @off[args.next_pid]);
    delete(@off[args.next_pid]);
}
' > offcpu.bt

# Render — see Brendan Gregg's "Off-CPU Flame Graphs" post for stackcollapse
```

### Wakeup flamegraph

```bash
bpftrace -e '
tracepoint:sched:sched_wakeup {
    @[kstack, comm] = count();
}
'
```

## JSON Output

```bash
bpftrace -f json script.bt
# {"type": "map", "data": {...}}
# {"type": "printf", "data": "..."}
```

JSON output is line-delimited, suitable for piping to `jq`, Vector, Loki, or scripted post-processing.

```bash
sudo bpftrace -f json -e '
tracepoint:syscalls:sys_enter_openat { printf("%s %s\n", comm, str(args.filename)); }
' | jq 'select(.type == "printf")'
```

Caveats:
- JSON output gets large for high-frequency probes.
- Maps print once at end; pipe through `jq` to reshape.
- Printf format strings carry through verbatim — they don't auto-jsonify their arguments.

## Performance Considerations

### Probe overhead — order from cheapest to most expensive

| Probe | Overhead per event | Notes |
|---|---|---|
| tracepoint | ~80 ns | Static instrumentation; cheapest |
| kfunc / kretfunc (BTF) | ~150 ns | Faster than kprobe (no int3) |
| kprobe / kretprobe | ~250 ns | Software trap into BPF |
| profile:hz:99 | ~250 ns | Timer-driven |
| uprobe / uretprobe | ~1.2 µs | Crosses kernel/user boundary, mprotect |
| usdt | ~150 ns | Pre-instrumented; static |

Overhead × event-rate = total cost. A 1M-events/sec kprobe is 25% of one CPU just on probe overhead. Pick tracepoints over kprobes when available.

### Aggregation discipline

```bash
# BAD — per-event printf at high rate (floods, drops events)
bpftrace -e 'kprobe:vfs_read { printf("%s %d\n", comm, arg2); }'

# GOOD — in-kernel aggregation, print on exit
bpftrace -e 'kprobe:vfs_read { @[comm] = sum(arg2); }'
```

### Reduce probe scope

```bash
# BAD — fires on every TCP packet
bpftrace -e 'tracepoint:tcp:* { … }'

# GOOD — fires only on retransmits
bpftrace -e 'tracepoint:tcp:tcp_retransmit_skb { … }'
```

### Filter early

```bash
# BAD — collect, then filter in user space
bpftrace -e 'kprobe:vfs_read { @[comm] = count(); }'  | grep myapp

# GOOD — predicate filters in kernel; map stays small
bpftrace -e 'kprobe:vfs_read /comm == "myapp"/ { @ = count(); }'
```

## Common Errors and Fixes

### "ERROR: BPF program too large. Processed N insn"

```
ERROR: Loading program failed: Argument list too long
ERROR: BPF program too large. Processed 1000001 insn
```

The compiled program exceeded the verifier's instruction limit (1M on modern kernels, 4096 pre-5.2). Causes:
- Too many `unroll`d iterations.
- Wildcards expanding to hundreds of probes (`kprobe:*` attaches one program per matched fn).
- Huge string operations.

**Fix:**
```bash
# narrow the wildcard
bpftrace -e 'kprobe:vfs_read { … }'        # NOT kprobe:vfs_*
# split into multiple scripts
# upgrade kernel for higher limits (≥ 5.2)
```

### "ERROR: Could not find program 'X' for type 'kprobe'"

```
ERROR: Could not resolve probe: kprobe:tcp_send_msg
```

Symbol typo (the function is `tcp_sendmsg`, no underscore between send and msg) or function inlined.

**Fix:**
```bash
# list candidates
bpftrace -l 'kprobe:*tcp*send*'
# use exact name
bpftrace -e 'kprobe:tcp_sendmsg { … }'
```

### "ERROR: kernel version mismatch" / "BTF not found"

```
ERROR: kernel does not support BTF
ERROR: vmlinux BTF is not available
```

Kernel built without `CONFIG_DEBUG_INFO_BTF=y`, or `/sys/kernel/btf/vmlinux` missing.

**Fix:**
```bash
# Verify
ls /sys/kernel/btf/vmlinux

# Install matching headers
sudo apt-get install linux-headers-$(uname -r)

# Or fetch from BTFhub (Aqua Security)
# https://github.com/aquasecurity/btfhub-archive
```

### "Address of program is null"

`CONFIG_BPF_JIT=n` or JIT disabled in sysctl.

**Fix:**
```bash
sudo sysctl -w net.core.bpf_jit_enable=1
# or rebuild kernel with CONFIG_BPF_JIT=y
```

### "Permission denied" / "Operation not permitted"

```
ERROR: failed to open perf event for kprobe:vfs_read: Operation not permitted
```

Not running as root (or no capabilities).

**Fix:**
```bash
sudo bpftrace -e '…'
# OR
sudo setcap cap_bpf,cap_perfmon,cap_net_admin=eip $(which bpftrace)
```

### "ERROR: Failed to attach probe"

```
ERROR: Failed to attach probe: kprobe:my_function
```

Function inlined (no symbol), tail-call optimized away, or symbol stripped.

**Fix:**
```bash
# check it exists
grep my_function /proc/kallsyms
bpftrace -l 'kprobe:my_*'

# for userspace, ensure binary is not stripped
file ./mybinary    # should NOT say "stripped"
nm ./mybinary | grep my_function

# rebuild without -O2 inlining of the function, or pick a non-inlined caller
```

### "lost N events"

```
bpftrace: lost 1234 events
```

Output ring buffer overflowed. Too many printf events.

**Fix:**
```bash
# Aggregate, don't printf-per-event
# OR increase buffer
bpftrace -B none …           # unbuffered (worse)
ulimit -l unlimited           # raise locked memory
sudo sysctl -w kernel.perf_event_max_stack=128
```

### "Unsafe builtin not allowed"

```
ERROR: signal() is unsafe; pass --unsafe to enable
ERROR: system() is unsafe; pass --unsafe to enable
```

**Fix:** add `--unsafe` (only when you actually need to send signals or run shell).

```bash
sudo bpftrace --unsafe -e 'kprobe:vfs_read /comm == "evil"/ { signal("SIGKILL"); }'
```

### "bpftrace: command not found" on macOS

bpftrace is Linux-only.

**Fix:**
```bash
# Use Lima
limactl start
limactl shell default sudo apt-get install -y bpftrace
```

## Common Gotchas

### Bad: comparing strings with `==` (older versions)

```bash
# BAD (older bpftrace; silently always false)
kprobe:do_sys_open /str(arg1) == "/etc/passwd"/ { … }

# GOOD
kprobe:do_sys_open /strncmp(str(arg1), "/etc/passwd", 11) == 0/ { … }
# OR
kprobe:do_sys_open /strcontains(str(arg1), "passwd")/ { … }
```

### Bad: missing predicate

```bash
# BAD — fires for every process; floods stdout
bpftrace -e 'kprobe:vfs_read { printf("%s read\n", comm); }'

# GOOD — narrow to one process
bpftrace -e 'kprobe:vfs_read /comm == "myapp"/ { printf("%s read\n", comm); }'
```

### Bad: per-event printf at high rate

```bash
# BAD — overwhelms ring buffer, drops events
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { printf("%d %s\n", args.id, comm); }'

# GOOD — aggregate, dump on END
bpftrace -e 'tracepoint:raw_syscalls:sys_enter { @[args.id, comm] = count(); }'
```

### Bad: forgetting clear()

```bash
# BAD — when this END runs, @start is still full of stale tids
END { print(@latency); }

# GOOD — clean up
END { print(@latency); clear(@latency); clear(@start); }
```

### Bad: assuming uretprobe always fires

```bash
# BAD — tail-call-optimized functions never reach the return probe
uprobe:./app:short_func   { @start[tid] = nsecs; }
uretprobe:./app:short_func /@start[tid]/ {  // may NEVER fire
    @latency = hist(nsecs - @start[tid]);
    delete(@start[tid]);
}

# Diagnose via objdump
objdump -d ./app | grep -A 5 short_func
# If you see `jmp <other_func>` instead of `ret`, the function tail-calls

# GOOD — use kfunc/kprobe on a non-inlined caller, or instrument the caller's exit
```

### Bad: assuming `arg0/arg1/...` are signed

bpftrace args are `uint64`. Negative kernel return values look enormous.

```bash
# BAD
kretprobe:vfs_read /retval > 0/ { … }   # works for ssize_t but…
# (-1 as uint64 is 0xFFFF…FFFF, which IS > 0)

# GOOD — cast to signed when needed
kretprobe:vfs_read /(int64)retval > 0/ { @bytes = sum(retval); }
```

### Bad: deleting while iterating

```bash
# BAD — may skip entries
END { for ($kv : @map) { delete(@map[$kv.0]); } }

# GOOD
END { print(@map); clear(@map); }
```

### Bad: wildcards that match too much

```bash
# BAD — attaches kprobes to every function whose name starts with t
bpftrace -e 'kprobe:t* { @ = count(); }'   # likely tens of thousands of probes; verifier-explosion

# GOOD — narrow the glob
bpftrace -e 'kprobe:tcp_send* { @ = count(); }'
```

### Bad: stale pids in @start map

```bash
# Process exits between kprobe and kretprobe — @start[tid] leaks
# Add an exit cleanup:
tracepoint:sched:sched_process_exit { delete(@start[tid]); }
```

## Listing Available Probes

```bash
# All kprobes (one per kernel function)
bpftrace -l 'kprobe:*' | wc -l    # 50,000+

# Kprobes matching a pattern
bpftrace -l 'kprobe:*read*'
bpftrace -l 'kprobe:tcp_*'

# All tracepoints
bpftrace -l 'tracepoint:*' | head

# Tracepoints in a category
bpftrace -l 'tracepoint:sched:*'
bpftrace -l 'tracepoint:syscalls:*'
bpftrace -l 'tracepoint:block:*'
bpftrace -l 'tracepoint:net:*'

# Userspace probes for a binary or library
bpftrace -l 'uprobe:/usr/bin/bash:*'
bpftrace -l 'uprobe:/lib/x86_64-linux-gnu/libc.so.6:*'

# USDTs in a binary
bpftrace -l 'usdt:/usr/lib/postgresql/16/bin/postgres:*'

# kfuncs (BTF-typed; 5.11+)
bpftrace -l 'kfunc:*' | head

# With argument types (verbose)
bpftrace -lv 'tracepoint:syscalls:sys_enter_openat'
# tracepoint:syscalls:sys_enter_openat
#     int dfd
#     const char * filename
#     int flags
#     umode_t mode

bpftrace -lv 'kfunc:vfs_read'
# kfunc:vfs_read
#     struct file * file
#     char * buf
#     size_t count
#     loff_t * pos

# Software/hardware events
bpftrace -l 'software:*'
bpftrace -l 'hardware:*'

# Profile / interval
bpftrace -l 'profile:*'
bpftrace -l 'interval:*'
```

### Find what's traceable for a given problem

```bash
# I want to trace TCP retransmits — what probes exist?
bpftrace -l '*retrans*'
# tracepoint:tcp:tcp_retransmit_skb
# tracepoint:tcp:tcp_retransmit_synack
# kprobe:tcp_retransmit_skb
# kprobe:__tcp_retransmit_skb
# kprobe:tcp_retransmit_timer

# I want to trace OOM
bpftrace -l '*oom*'
# kprobe:oom_kill_process
# kprobe:out_of_memory
# tracepoint:oom:mark_victim
```

## Idioms

### The canonical three-phase script

```bash
BEGIN { /* setup, header */ }
probe { /* per-event work — accumulate into @maps */ }
END   { /* dump, cleanup */ }
```

### The timing pattern

```bash
kprobe:fn        { @start[tid] = nsecs; }
kretprobe:fn /@start[tid]/ {
    @ns = hist(nsecs - @start[tid]);
    delete(@start[tid]);
}
```

### The histogram-on-exit dump

```bash
END { print(@latency); clear(@latency); }
```

bpftrace prints maps automatically; explicit `print` controls order and lets you `clear` to suppress the auto-dump.

### The filter-by-process predicate

```bash
/comm == "myapp"/
/pid == 1234/
/strncmp(comm, "redis", 5) == 0/
```

### The interval-based sliding window

```bash
interval:s:5 {
    time("%H:%M:%S\n");
    print(@, 10);    // top 10
    clear(@);
}
```

### Per-process running stats

```bash
@stats[comm] = stats(retval);    // count + avg + total in one
```

### Multi-key aggregation

```bash
@by_op_pid[probe, pid] = count();
@by_dst[ntop(arg0)] = sum(arg2);
```

### Spawn a process under tracing

```bash
sudo bpftrace -c './target arg1 arg2' -e '
uprobe:./target:hot_function { @hits[arg0] = count(); }
'
# bpftrace exits when target exits; END runs and prints @hits
```

### Trace one PID

```bash
sudo bpftrace -p 12345 -e '
profile:hz:99 { @[ustack] = count(); }
'
```

### Kill the script after N seconds

```bash
bpftrace -e '
profile:hz:99 { @[ustack] = count(); }
interval:s:30 { exit(); }
'
```

### Live "top" with refresh

```bash
bpftrace -e '
tracepoint:raw_syscalls:sys_enter { @[comm] = count(); }
interval:s:1 {
    printf("\033[2J\033[H");   // clear screen
    print(@, 20);              // top 20
    clear(@);
}'
```

## Tips

- **Always prefer tracepoints** over kprobes when one exists for what you want — stable ABI, lower overhead.
- **Aggregate in maps**, dump on `END` or `interval`. Per-event `printf` floods.
- **Filter early** with predicates `/cond/` to keep map cardinality bounded.
- **Use `bpftrace -l 'pattern'`** to discover probes; pipe through `grep` and `wc -l` to estimate scope.
- **Capture stack traces** with `kstack`/`ustack`; pipe to FlameGraph for visualization.
- **For uretprobes,** ensure the function actually returns (no tail-call elimination); check disassembly.
- **Build apps with frame pointers** (`-fno-omit-frame-pointer` in C/C++; default in Go ≥1.18; `-C force-frame-pointers=yes` in Rust) for usable user-space stacks.
- **Read `/sys/kernel/btf/vmlinux`** existence first; without it, kfunc probes and many CO-RE patterns won't work.
- **Run `bpftrace --info`** to see kernel BPF feature support.
- **Match versions:** features in this sheet noted with kernel/bpftrace version requirements.
- **For long-running tools,** port from bpftrace to libbpf+CO-RE; bpftrace's runtime LLVM compile is unfit for daemons.
- **Use `-f json`** for machine-readable output when piping to other tools.
- **Use `-c "cmd"`** to spawn-and-trace a target; bpftrace cleans up on its exit.
- **Use `-p PID`** to attach uprobes to one running process.
- **For high-cardinality maps**, periodically `clear()` to keep memory bounded — BPF maps have a fixed `max_entries` (default ~10k for hash maps in bpftrace).

## See Also

- ebpf, perf, flamegraph, polyglot, bash

## References

- [bpftrace — bpftrace.org](https://bpftrace.org/)
- [bpftrace GitHub](https://github.com/bpftrace/bpftrace)
- [bpftrace Reference Guide](https://github.com/bpftrace/bpftrace/blob/master/docs/reference_guide.md)
- [bpftrace One-Liners Tutorial](https://github.com/bpftrace/bpftrace/blob/master/docs/tutorial_one_liners.md)
- [bpftrace Internals & Development](https://github.com/bpftrace/bpftrace/blob/master/docs/internals_development.md)
- [man bpftrace(8)](https://man7.org/linux/man-pages/man8/bpftrace.8.html)
- [Brendan Gregg — bpftrace Cheat Sheet](https://www.brendangregg.com/BPF/bpftrace-cheat-sheet.html)
- [Brendan Gregg — BPF Performance Tools (book)](https://www.brendangregg.com/bpf-performance-tools-book.html)
- [Brendan Gregg — bpftrace Tools Catalog](https://github.com/bpftrace/bpftrace/tree/master/tools)
- [eBPF.io — bpftrace](https://ebpf.io/projects/#bpftrace)
- [Kernel BPF Documentation](https://www.kernel.org/doc/html/latest/bpf/)
- [Kernel Tracepoints](https://www.kernel.org/doc/html/latest/trace/tracepoints.html)
- [Kernel kprobes](https://www.kernel.org/doc/html/latest/trace/kprobes.html)
- [Kernel uprobes](https://www.kernel.org/doc/html/latest/trace/uprobetracer.html)
- [USDT — User Statically-Defined Tracing](https://sourceware.org/systemtap/wiki/UserSpaceProbeImplementation)
- [BTFhub Archive](https://github.com/aquasecurity/btfhub-archive)
- [libbpf](https://github.com/libbpf/libbpf)
- [BCC — BPF Compiler Collection](https://github.com/iovisor/bcc)
- [FlameGraph](https://github.com/brendangregg/FlameGraph)
- [Brendan Gregg — Off-CPU Flame Graphs](https://www.brendangregg.com/FlameGraphs/offcpuflamegraphs.html)
- [Brendan Gregg — Linux Performance Analysis in 60 Seconds](https://netflixtechblog.com/linux-performance-analysis-in-60-000-milliseconds-accc10403c55)
