# Signals (Unix Process Signals)

Unix signals are asynchronous notifications sent to processes to indicate events such as termination requests, child process status changes, or user-defined triggers, forming the primary inter-process communication mechanism for process lifecycle management.

## Signal Reference Table

| Signal | Number | Default Action | Catchable | Description |
|--------|--------|---------------|-----------|-------------|
| SIGHUP | 1 | Terminate | Yes | Hangup / config reload |
| SIGINT | 2 | Terminate | Yes | Interrupt (Ctrl+C) |
| SIGQUIT | 3 | Core dump | Yes | Quit (Ctrl+\) |
| SIGILL | 4 | Core dump | Yes | Illegal instruction |
| SIGTRAP | 5 | Core dump | Yes | Trace/breakpoint trap |
| SIGABRT | 6 | Core dump | Yes | Abort |
| SIGBUS | 7 | Core dump | Yes | Bus error |
| SIGFPE | 8 | Core dump | Yes | Floating-point exception |
| SIGKILL | 9 | Terminate | **No** | Force kill (uncatchable) |
| SIGUSR1 | 10 | Terminate | Yes | User-defined signal 1 |
| SIGSEGV | 11 | Core dump | Yes | Segmentation fault |
| SIGUSR2 | 12 | Terminate | Yes | User-defined signal 2 |
| SIGPIPE | 13 | Terminate | Yes | Broken pipe |
| SIGALRM | 14 | Terminate | Yes | Alarm clock |
| SIGTERM | 15 | Terminate | Yes | Graceful termination |
| SIGCHLD | 17 | Ignore | Yes | Child process status change |
| SIGCONT | 18 | Continue | Yes | Resume stopped process |
| SIGSTOP | 19 | Stop | **No** | Stop process (uncatchable) |
| SIGTSTP | 20 | Stop | Yes | Terminal stop (Ctrl+Z) |
| SIGWINCH | 28 | Ignore | Yes | Terminal window size change |

## Sending Signals

```bash
# Send SIGTERM (default, graceful shutdown)
kill $PID
kill -TERM $PID
kill -15 $PID

# Send SIGKILL (force kill, cannot be caught)
kill -9 $PID
kill -KILL $PID

# Send SIGHUP (reload config by convention)
kill -HUP $PID
kill -1 $PID

# Send to all processes in a process group
kill -TERM -$PGID

# Send to all processes with a given name
killall -TERM nginx
pkill -TERM -f "python server.py"

# Send signal to all processes of a user
pkill -TERM -u username

# Send SIGUSR1 for custom behavior
kill -USR1 $PID

# List all signal names
kill -l
```

## Signal Handling in Bash

```bash
# Trap signals in a bash script
#!/bin/bash

cleanup() {
    echo "Caught signal, cleaning up..."
    rm -f /tmp/myapp.pid
    exit 0
}

# Register signal handlers
trap cleanup SIGTERM SIGINT SIGHUP
trap "echo 'SIGUSR1 received'" SIGUSR1

# Ignore a signal
trap '' SIGPIPE

# Reset signal to default behavior
trap - SIGTERM

# Trap EXIT (runs on any exit, including normal)
trap 'echo "Script exiting"' EXIT

# Trap ERR (runs on any command failure)
trap 'echo "Error on line $LINENO"' ERR

# Show current traps
trap -p

# Background process with signal handling
long_running_task &
BGPID=$!
trap "kill $BGPID; wait $BGPID; exit" SIGTERM SIGINT
wait $BGPID
```

## Signal Handling in C

```c
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

volatile sig_atomic_t got_signal = 0;

void handler(int sig) {
    got_signal = sig;  // Only async-signal-safe operations
}

int main() {
    struct sigaction sa;
    sa.sa_handler = handler;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = SA_RESTART;  // Restart interrupted syscalls

    sigaction(SIGTERM, &sa, NULL);
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGHUP, &sa, NULL);

    // Ignore SIGPIPE
    signal(SIGPIPE, SIG_IGN);

    // Block SIGUSR1 during critical section
    sigset_t mask, oldmask;
    sigemptyset(&mask);
    sigaddset(&mask, SIGUSR1);
    sigprocmask(SIG_BLOCK, &mask, &oldmask);

    // ... critical section ...

    // Unblock
    sigprocmask(SIG_SETMASK, &oldmask, NULL);

    while (!got_signal) {
        pause();  // Wait for signal
    }

    printf("Received signal %d\n", got_signal);
    return 0;
}
```

## Signal Handling in Go

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // Create channel for signals
    sigCh := make(chan os.Signal, 1)

    // Register for specific signals
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

    // Context-based cancellation (preferred pattern)
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGTERM, syscall.SIGINT)
    defer stop()

    go func() {
        <-ctx.Done()
        fmt.Println("Shutting down gracefully...")
        // Cleanup logic here
    }()

    // Or use channel directly
    sig := <-sigCh
    fmt.Printf("Received: %v\n", sig)

    // Ignore a signal
    signal.Ignore(syscall.SIGPIPE)

    // Reset to default behavior
    signal.Reset(syscall.SIGTERM)
}
```

## Signal Handling in Python

```python
import signal
import sys
import os

def handler(signum, frame):
    print(f"Received signal {signum}")
    sys.exit(0)

# Register handlers
signal.signal(signal.SIGTERM, handler)
signal.signal(signal.SIGINT, handler)
signal.signal(signal.SIGHUP, handler)

# Ignore a signal
signal.signal(signal.SIGPIPE, signal.SIG_IGN)

# Reset to default
signal.signal(signal.SIGTERM, signal.SIG_DFL)

# Block signals temporarily (Python 3.3+)
signal.pthread_sigmask(signal.SIG_BLOCK, {signal.SIGUSR1})
# ... critical section ...
signal.pthread_sigmask(signal.SIG_UNBLOCK, {signal.SIGUSR1})

# Set alarm (SIGALRM after N seconds)
signal.alarm(30)

# Wait for any signal
signal.pause()
```

## Process Groups and Sessions

```bash
# View process group IDs
ps -eo pid,pgid,sid,comm

# Send signal to entire process group
kill -TERM -$(ps -o pgid= -p $PID | tr -d ' ')

# Create a new process group
setsid bash -c 'echo new session $$'

# Send signal to session leader's group
pkill -TERM -s $SID
```

## Signal Debugging

```bash
# Trace signals received by a process
strace -e signal -p $PID

# Check pending signals
cat /proc/$PID/status | grep -E "^Sig"
# SigQ:    0/63432          # queued/max
# SigPnd:  0000000000000000 # pending for thread
# ShdPnd:  0000000000000000 # pending for process
# SigBlk:  0000000000000000 # blocked mask
# SigIgn:  0000000000001000 # ignored mask
# SigCgt:  0000000180004002 # caught mask

# Decode signal mask (hex to signal list)
python3 -c "
mask = 0x0000000180004002
for i in range(1, 65):
    if mask & (1 << (i-1)):
        print(f'Signal {i}')
"
# Signal 2 (SIGINT), Signal 15 (SIGTERM), Signal 32, Signal 33

# Watch for zombie processes (need SIGCHLD handling)
ps aux | awk '$8 ~ /Z/ {print}'
```

## Graceful Shutdown Pattern

```bash
#!/bin/bash
# Production-grade graceful shutdown

SHUTDOWN=0
PIDS=()

shutdown_handler() {
    SHUTDOWN=1
    echo "Shutting down ${#PIDS[@]} workers..."
    for pid in "${PIDS[@]}"; do
        kill -TERM "$pid" 2>/dev/null
    done
    # Wait with timeout
    for pid in "${PIDS[@]}"; do
        timeout 30 tail --pid="$pid" -f /dev/null 2>/dev/null
        if kill -0 "$pid" 2>/dev/null; then
            echo "Force killing $pid"
            kill -9 "$pid"
        fi
    done
    exit 0
}

trap shutdown_handler SIGTERM SIGINT

# Start workers
for i in $(seq 1 4); do
    worker_process &
    PIDS+=($!)
done

# Wait for all workers
while [ $SHUTDOWN -eq 0 ]; do
    wait -n 2>/dev/null || true
done
```

## Tips

- Always send SIGTERM before SIGKILL; give processes time to clean up (Docker uses 10s default)
- SIGKILL and SIGSTOP cannot be caught, blocked, or ignored -- they are kernel-enforced
- Use `signal.NotifyContext` in Go for clean context-based shutdown patterns
- SIGCHLD must be handled (or explicitly ignored) to prevent zombie processes
- In signal handlers, only call async-signal-safe functions; `printf` and `malloc` are NOT safe
- SIGPIPE kills processes writing to closed pipes; always ignore it in network daemons
- Use `SA_RESTART` flag in C to automatically restart interrupted system calls
- Real-time signals (SIGRTMIN to SIGRTMAX) are queued and delivered in order, unlike standard signals
- `kill -0 $PID` tests whether a process exists without sending any signal
- In containers, PID 1 has special signal behavior: it only receives signals it explicitly handles
- `pkill -f` matches the full command line, not just the process name
- Use `trap cleanup EXIT` in bash scripts for reliable cleanup regardless of how the script ends

## See Also

proc-sys, namespaces, oom-killer, ulimit

## References

- [signal(7) Man Page](https://man7.org/linux/man-pages/man7/signal.7.html)
- [sigaction(2) Man Page](https://man7.org/linux/man-pages/man2/sigaction.2.html)
- [Go os/signal Package](https://pkg.go.dev/os/signal)
- [Python signal Module](https://docs.python.org/3/library/signal.html)
- [Proper handling of SIGINT/SIGQUIT](https://www.cons.org/cracauer/sigint.html)
