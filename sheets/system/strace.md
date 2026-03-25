# strace (system call tracer)

Trace system calls and signals for a process. Essential for debugging.

## Tracing a Command

### Run a Command Under strace

```bash
# Trace all syscalls
strace ls /tmp

# Trace with output to file
strace -o /tmp/trace.log ls /tmp

# Follow child processes (forks)
strace -f ./my_server
```

## Attach to Running Process

### Trace by PID

```bash
strace -p 1234

# Attach and follow threads
strace -fp 1234

# Attach with output to file
strace -fp 1234 -o /tmp/debug.log
```

## Filter Syscalls

### Trace Specific Calls

```bash
# Only file-related calls
strace -e trace=file ls /tmp

# Only network-related calls
strace -e trace=network curl example.com

# Only open and read
strace -e trace=open,read cat /etc/hosts

# Only write calls (see what a process writes)
strace -e trace=write -p 1234

# Process management calls
strace -e trace=process ./my_program

# Memory-related calls
strace -e trace=memory ./my_program

# Signal-related calls
strace -e trace=%signal ./my_program
```

### Negate Filters

```bash
# Everything except mmap and mprotect (noisy on startup)
strace -e trace=!mmap,mprotect ./my_program
```

## Timing

### Timestamps and Duration

```bash
# Relative timestamp on each line
strace -r ls /tmp

# Wall-clock timestamp on each line
strace -t ls /tmp

# Microsecond timestamps
strace -tt ls /tmp

# Time spent in each syscall
strace -T ls /tmp
```

## Statistics

### Summarize Syscall Counts

```bash
# Count and time summary only (no trace output)
strace -c ls /tmp

# Summary with trace output combined
strace -C ls /tmp

# Sort summary by time spent
strace -c -S time ls /tmp
```

## Output Control

### String and Buffer Size

```bash
# Show more of string arguments (default is 32 chars)
strace -s 256 -e trace=write -p 1234

# Show full strings
strace -s 9999 -e trace=read,write -p 1234

# Show file paths being accessed
strace -y ls /tmp

# Show IP addresses instead of fd numbers for sockets
strace -yy -e trace=network curl example.com
```

### Multiple PIDs

```bash
# Trace multiple processes
strace -p 1234 -p 5678

# Trace all processes of a user (via pgrep)
strace -fp $(pgrep -d, -u deploy)
```

## Common Debugging Patterns

### Find What Files a Program Opens

```bash
strace -e trace=openat -f ./my_program 2>&1 | grep -v ENOENT
```

### Debug DNS Resolution

```bash
strace -e trace=network,open -f getent hosts example.com
```

### Find Why a Program Hangs

```bash
strace -fp $(pgrep my_server) -e trace=futex,poll,epoll_wait
```

## Tips

- strace output goes to stderr -- redirect with `2>file` or use `-o`.
- `-f` is critical for multi-threaded or forking programs; without it you only see the parent.
- strace adds significant overhead (~2-10x slowdown) -- do not leave it attached in production for long.
- Use `-e trace=!futex` to filter out the extremely noisy futex calls in threaded programs.
- `-y` translates file descriptors to paths and `-yy` adds socket details -- invaluable for network debugging.
- On newer kernels, `perf trace` is a lower-overhead alternative for syscall tracing.
- `ltrace` is the equivalent for library calls (e.g., `ltrace -e malloc ./my_program`).

## References

- [man strace(1)](https://man7.org/linux/man-pages/man1/strace.1.html)
- [man ptrace(2)](https://man7.org/linux/man-pages/man2/ptrace.2.html)
- [man syscalls(2)](https://man7.org/linux/man-pages/man2/syscalls.2.html)
- [man ltrace(1)](https://man7.org/linux/man-pages/man1/ltrace.1.html)
- [strace Project Site](https://strace.io/)
- [strace GitHub](https://github.com/strace/strace)
- [Arch Wiki — strace](https://wiki.archlinux.org/title/Strace)
- [Red Hat — Tracing System Calls with strace](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/developing_c_and_cpp_applications_in_rhel_9/debugging-applications_developing-applications)
- [Kernel ptrace Documentation](https://www.kernel.org/doc/html/latest/process/adding-syscalls.html)
