# eBPF (Extended Berkeley Packet Filter)

In-kernel virtual machine for safe, programmable tracing, networking, and security at the kernel level.

## Overview

### What eBPF enables

```bash
# Tracing:     instrument kernel and user-space functions without recompilation
# Networking:  XDP for high-speed packet processing, TC for traffic shaping
# Security:    syscall filtering, LSM hooks, runtime policy enforcement
# Observability: custom metrics, latency histograms, flamegraphs with near-zero overhead
```

### Architecture

```bash
# User space:  write BPF program (C, bpftrace, etc.)
# Compiler:    LLVM compiles to BPF bytecode
# Verifier:    kernel verifies safety (no loops, bounded memory, valid accesses)
# JIT:         bytecode compiled to native machine code
# Maps:        shared data structures between BPF programs and user space
```

## BCC Tools (BPF Compiler Collection)

### Process and thread tracing

```bash
execsnoop                                  # trace new processes (exec calls)
execsnoop -t                               # with timestamps
execsnoop -x                               # include failed execs
```

```bash
threadsnoop                                # trace new threads
exitsnoop                                  # trace process exits with duration
```

### File and I/O tracing

```bash
opensnoop                                  # trace file opens
opensnoop -p 12345                         # filter by PID
opensnoop -n nginx                         # filter by process name
opensnoop -x                               # show only failed opens
```

```bash
filelife                                   # trace short-lived files
fileslower 10                              # files with I/O slower than 10ms
filetop                                    # top-like for file I/O
```

### Block I/O

```bash
biolatency                                 # block I/O latency histogram
biolatency -D                              # per-disk histograms
biolatency -m                              # milliseconds
biosnoop                                   # trace each block I/O with latency
biotop                                     # top-like for block I/O
```

### Network tracing

```bash
tcpconnect                                 # trace outbound TCP connections
tcpconnect -p 12345                        # by PID
tcpaccept                                  # trace inbound TCP connections
tcplife                                    # trace TCP sessions with duration and bytes
tcpretrans                                 # trace TCP retransmits
tcpdrop                                    # trace dropped TCP packets
```

### Memory

```bash
memleak -p 12345                           # detect memory leaks
cachestat                                  # page cache hit/miss statistics
cachetop                                   # top-like for page cache
oomkill                                    # trace OOM killer events
```

### CPU and scheduling

```bash
cpudist                                    # on-CPU time distribution
runqlat                                    # run queue (scheduler) latency
runqlen                                    # run queue length
offcputime -p 12345                        # off-CPU time (blocked)
softirqs                                   # soft interrupt time
hardirqs                                   # hard interrupt time
```

### Syscall tracing

```bash
syscount                                   # count syscalls by type
syscount -p 12345                          # by PID
syscount -L                                # show latency
funccount 'vfs_*'                          # count kernel function calls
funclatency vfs_read                       # latency distribution for a function
```

### DNS

```bash
gethostlatency                             # trace DNS lookup latency
```

## bpftool

### List loaded programs

```bash
bpftool prog list
bpftool prog show id 42
bpftool prog dump xlated id 42             # show BPF instructions
bpftool prog dump jited id 42              # show JIT output
```

### List maps

```bash
bpftool map list
bpftool map show id 15
bpftool map dump id 15                     # dump map contents
```

### Pin a program

```bash
bpftool prog pin id 42 /sys/fs/bpf/my_prog
```

### Attach and detach

```bash
bpftool net attach xdp id 42 dev eth0
bpftool net detach xdp dev eth0
bpftool net list                           # show attached programs
```

### Feature detection

```bash
bpftool feature probe                      # check kernel BPF support
bpftool feature probe kernel full          # detailed feature list
```

## Map Types

### Common map types

```bash
# BPF_MAP_TYPE_HASH             hash table
# BPF_MAP_TYPE_ARRAY            fixed-size array
# BPF_MAP_TYPE_PERCPU_HASH      per-CPU hash (no lock contention)
# BPF_MAP_TYPE_PERCPU_ARRAY     per-CPU array
# BPF_MAP_TYPE_LRU_HASH         hash with LRU eviction
# BPF_MAP_TYPE_RINGBUF          efficient ring buffer (events to user space)
# BPF_MAP_TYPE_PERF_EVENT_ARRAY perf event output
# BPF_MAP_TYPE_LPM_TRIE         longest prefix match (IP routing)
# BPF_MAP_TYPE_STACK_TRACE      stack trace storage
# BPF_MAP_TYPE_BLOOM_FILTER     probabilistic set membership
```

### Map operations from user space

```bash
bpftool map create /sys/fs/bpf/mymap type hash key 4 value 8 entries 1024
bpftool map update id 15 key 0x01 0x00 0x00 0x00 value 0x42 0x00 0x00 0x00 0x00 0x00 0x00 0x00
bpftool map lookup id 15 key 0x01 0x00 0x00 0x00
bpftool map delete id 15 key 0x01 0x00 0x00 0x00
```

## Program Types

### Common program types

```bash
# BPF_PROG_TYPE_KPROBE           kernel function tracing
# BPF_PROG_TYPE_TRACEPOINT       kernel tracepoint
# BPF_PROG_TYPE_PERF_EVENT       perf event handling
# BPF_PROG_TYPE_XDP              eXpress Data Path (packet processing)
# BPF_PROG_TYPE_SCHED_CLS        traffic control classifier
# BPF_PROG_TYPE_SOCKET_FILTER    socket filtering
# BPF_PROG_TYPE_CGROUP_SKB       cgroup socket buffer
# BPF_PROG_TYPE_LSM              Linux Security Module hooks
# BPF_PROG_TYPE_STRUCT_OPS       kernel struct operations
```

## XDP (eXpress Data Path)

### XDP actions

```bash
# XDP_PASS       pass to normal network stack
# XDP_DROP       drop the packet (fastest possible drop)
# XDP_TX         transmit back out same interface
# XDP_REDIRECT   redirect to another interface or CPU
# XDP_ABORTED    error, drop and trigger trace
```

### Attach XDP program

```bash
ip link set dev eth0 xdp obj xdp_prog.o sec xdp
ip link set dev eth0 xdpgeneric obj xdp_prog.o sec xdp  # generic (slower, any driver)
ip link set dev eth0 xdp off                              # detach
```

### XDP statistics

```bash
bpftool net list                           # show XDP attachments
ip -d link show eth0                       # show XDP info on interface
```

## Diagnostics

### Check kernel support

```bash
uname -r                                  # kernel version (4.18+ for most BPF features)
cat /boot/config-$(uname -r) | grep BPF   # kernel config
bpftool feature probe kernel              # runtime feature check
```

### Debug BPF programs

```bash
cat /sys/kernel/debug/tracing/trace_pipe   # trace_printk output
bpftool prog tracelog                      # same as above
```

### Verifier output

```bash
# When a BPF program fails to load, the verifier prints why.
# Common issues:
#   "R0 invalid mem access" — accessing memory beyond verified bounds
#   "back-edge from insn X to Y" — loop detected (not allowed without bpf_loop)
#   "unreachable insn" — dead code
```

## Common Patterns

### High-level tracing stack

```bash
# bpftrace        quick one-liners and scripts (highest level)
# BCC tools        ready-made tools for common tasks (execsnoop, biolatency, etc.)
# libbpf + C       full control, CO-RE (Compile Once, Run Everywhere)
# cilium/ebpf (Go) production eBPF programs in Go
```

### Typical workflow

```bash
# 1. Start with BCC tools to identify the problem area
biolatency                                 # is disk I/O slow?
runqlat                                    # is the scheduler queue long?
tcpretrans                                 # are there network retransmits?

# 2. Use bpftrace for custom investigation
bpftrace -e 'kprobe:vfs_read /comm == "myapp"/ { @[kstack] = count(); }'

# 3. Build a custom BPF program with libbpf/cilium for production use
```

## Tips

- BCC tools are the fastest way to diagnose performance issues. Start with `execsnoop`, `opensnoop`, `biolatency`, `tcpconnect`, and `runqlat`.
- eBPF programs are verified for safety before loading. They cannot crash the kernel, access arbitrary memory, or loop indefinitely.
- CO-RE (Compile Once, Run Everywhere) with libbpf eliminates the need to recompile BPF programs for different kernel versions.
- XDP processes packets before the kernel network stack, achieving line-rate packet processing.
- Use `BPF_MAP_TYPE_RINGBUF` over `BPF_MAP_TYPE_PERF_EVENT_ARRAY` for event streaming to user space. It is more efficient and supports variable-length data.
- `bpftool feature probe` shows exactly which BPF features your kernel supports.
- Kernel 5.8+ added BPF LSM hooks for security policy enforcement without kernel modules.
- The `CAP_BPF` capability (kernel 5.8+) allows non-root users to load BPF programs.

## See Also

- bpftrace, perf, kernel, strace, docker

## References

- [eBPF.io — Official Documentation](https://ebpf.io/)
- [eBPF.io — What is eBPF?](https://ebpf.io/what-is-ebpf/)
- [eBPF.io — Project Landscape](https://ebpf.io/projects/)
- [Kernel BPF Documentation](https://www.kernel.org/doc/html/latest/bpf/)
- [Kernel BPF Design Q&A](https://www.kernel.org/doc/html/latest/bpf/bpf_design_QA.html)
- [Kernel BPF Instruction Set](https://www.kernel.org/doc/html/latest/bpf/standardization/instruction-set.html)
- [man bpf(2) — BPF System Call](https://man7.org/linux/man-pages/man2/bpf.2.html)
- [BCC — BPF Compiler Collection](https://github.com/iovisor/bcc)
- [libbpf — C Library for BPF](https://github.com/libbpf/libbpf)
- [Brendan Gregg — BPF Performance Tools](https://www.brendangregg.com/bpf-performance-tools-book.html)
- [Cilium — eBPF Reference Guide](https://docs.cilium.io/en/latest/bpf/)
- [Red Hat — eBPF Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_monitoring_and_updating_the_kernel/assembly_understanding-extended-bpf_managing-monitoring-and-updating-the-kernel)
