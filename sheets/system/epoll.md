# epoll (Linux I/O Event Notification)

Scalable I/O event notification mechanism for monitoring multiple file descriptors, the foundation of high-performance event loops on Linux.

## Core API

### epoll_create

```bash
# Create an epoll instance (returns epoll fd)
# int epoll_create1(int flags);
#   flags: 0 or EPOLL_CLOEXEC
```

```c
#include <sys/epoll.h>

// Create epoll instance
int epfd = epoll_create1(0);
if (epfd == -1) {
    perror("epoll_create1");
    exit(EXIT_FAILURE);
}

// With close-on-exec flag
int epfd = epoll_create1(EPOLL_CLOEXEC);
```

### epoll_ctl (add/modify/delete)

```c
// struct epoll_event
struct epoll_event {
    uint32_t     events;    // EPOLLIN, EPOLLOUT, EPOLLERR, etc.
    epoll_data_t data;      // user data (fd, ptr, u32, u64)
};

// Add a file descriptor to epoll
struct epoll_event ev;
ev.events = EPOLLIN;                    // watch for readable
ev.data.fd = client_fd;
epoll_ctl(epfd, EPOLL_CTL_ADD, client_fd, &ev);

// Modify watched events
ev.events = EPOLLIN | EPOLLOUT;         // now also watch for writable
epoll_ctl(epfd, EPOLL_CTL_MOD, client_fd, &ev);

// Remove a file descriptor
epoll_ctl(epfd, EPOLL_CTL_DEL, client_fd, NULL);
```

### epoll_wait (wait for events)

```c
#define MAX_EVENTS 64
struct epoll_event events[MAX_EVENTS];

// Block until events arrive (timeout -1 = infinite)
int nfds = epoll_wait(epfd, events, MAX_EVENTS, -1);

// With timeout (milliseconds)
int nfds = epoll_wait(epfd, events, MAX_EVENTS, 1000);  // 1 second

// Non-blocking check
int nfds = epoll_wait(epfd, events, MAX_EVENTS, 0);

// Process events
for (int i = 0; i < nfds; i++) {
    if (events[i].events & EPOLLIN) {
        handle_read(events[i].data.fd);
    }
    if (events[i].events & EPOLLOUT) {
        handle_write(events[i].data.fd);
    }
    if (events[i].events & (EPOLLERR | EPOLLHUP)) {
        handle_error(events[i].data.fd);
    }
}
```

## Event Flags

```c
// Input events (set in epoll_ctl)
EPOLLIN      // fd is readable (data available)
EPOLLOUT     // fd is writable (buffer space available)
EPOLLPRI     // urgent data (TCP OOB)
EPOLLRDHUP   // peer closed connection (half-close detection)

// Behavior modifiers
EPOLLET      // edge-triggered mode (default is level-triggered)
EPOLLONESHOT // disable fd after one event (must re-arm with EPOLL_CTL_MOD)
EPOLLEXCLUSIVE // avoid thundering herd (one thread wakes per event)

// Output-only events (returned by epoll_wait)
EPOLLERR     // error condition on fd
EPOLLHUP     // hangup (peer closed both read and write)
```

## Edge-Triggered vs Level-Triggered

### Level-Triggered (default)

```c
// LT: epoll_wait returns as long as the condition is true
// Safe: even if you don't read all data, epoll_wait will notify again
struct epoll_event ev;
ev.events = EPOLLIN;                    // level-triggered (default)
ev.data.fd = fd;
epoll_ctl(epfd, EPOLL_CTL_ADD, fd, &ev);

// Simple read — partial reads are OK, epoll will fire again
ssize_t n = read(fd, buf, sizeof(buf));
```

### Edge-Triggered

```c
// ET: epoll_wait returns ONLY when state changes (new data arrives)
// You MUST read all available data in a loop, or data may be lost
struct epoll_event ev;
ev.events = EPOLLIN | EPOLLET;         // edge-triggered
ev.data.fd = fd;
epoll_ctl(epfd, EPOLL_CTL_ADD, fd, &ev);

// MUST use non-blocking fd + drain loop
fcntl(fd, F_SETFL, fcntl(fd, F_GETFL) | O_NONBLOCK);

while (1) {
    ssize_t n = read(fd, buf, sizeof(buf));
    if (n == -1) {
        if (errno == EAGAIN || errno == EWOULDBLOCK)
            break;                      // all data consumed
        perror("read");
        break;
    }
    if (n == 0) {
        // EOF — peer closed
        close(fd);
        break;
    }
    process_data(buf, n);
}
```

## EPOLLONESHOT

```c
// One-shot mode: fd is disabled after delivering one event
// Prevents multiple threads from processing the same fd simultaneously
struct epoll_event ev;
ev.events = EPOLLIN | EPOLLONESHOT;
ev.data.fd = fd;
epoll_ctl(epfd, EPOLL_CTL_ADD, fd, &ev);

// After processing, re-arm the fd
ev.events = EPOLLIN | EPOLLONESHOT;
epoll_ctl(epfd, EPOLL_CTL_MOD, fd, &ev);
```

## Complete Event Loop Pattern

```c
#include <sys/epoll.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <fcntl.h>
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>

#define MAX_EVENTS 1024
#define PORT 8080

void set_nonblocking(int fd) {
    fcntl(fd, F_SETFL, fcntl(fd, F_GETFL) | O_NONBLOCK);
}

int main() {
    // Create listening socket
    int listen_fd = socket(AF_INET, SOCK_STREAM | SOCK_NONBLOCK, 0);
    int opt = 1;
    setsockopt(listen_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    struct sockaddr_in addr = {
        .sin_family = AF_INET,
        .sin_port = htons(PORT),
        .sin_addr.s_addr = INADDR_ANY
    };
    bind(listen_fd, (struct sockaddr *)&addr, sizeof(addr));
    listen(listen_fd, SOMAXCONN);

    // Create epoll
    int epfd = epoll_create1(0);
    struct epoll_event ev;
    ev.events = EPOLLIN;
    ev.data.fd = listen_fd;
    epoll_ctl(epfd, EPOLL_CTL_ADD, listen_fd, &ev);

    struct epoll_event events[MAX_EVENTS];

    // Event loop
    while (1) {
        int nfds = epoll_wait(epfd, events, MAX_EVENTS, -1);
        for (int i = 0; i < nfds; i++) {
            if (events[i].data.fd == listen_fd) {
                // Accept new connections
                while (1) {
                    int client = accept(listen_fd, NULL, NULL);
                    if (client == -1) break;
                    set_nonblocking(client);
                    ev.events = EPOLLIN | EPOLLET;
                    ev.data.fd = client;
                    epoll_ctl(epfd, EPOLL_CTL_ADD, client, &ev);
                }
            } else {
                // Handle client data
                char buf[4096];
                while (1) {
                    ssize_t n = read(events[i].data.fd, buf, sizeof(buf));
                    if (n <= 0) {
                        if (n == 0 || errno != EAGAIN) {
                            close(events[i].data.fd);
                        }
                        break;
                    }
                    write(events[i].data.fd, buf, n); // echo
                }
            }
        }
    }
    close(epfd);
    close(listen_fd);
}
```

## Thundering Herd Problem

```c
// Problem: multiple threads call epoll_wait on same epfd
// All wake up for the same event, but only one can handle it

// Solution 1: EPOLLEXCLUSIVE (Linux 4.5+)
ev.events = EPOLLIN | EPOLLEXCLUSIVE;
epoll_ctl(epfd, EPOLL_CTL_ADD, listen_fd, &ev);
// Only ONE thread wakes per event

// Solution 2: SO_REUSEPORT (separate sockets, separate epolls)
int opt = 1;
setsockopt(fd, SOL_SOCKET, SO_REUSEPORT, &opt, sizeof(opt));
// Each thread gets its own socket + epoll, kernel distributes connections

// Solution 3: EPOLLONESHOT (re-arm after processing)
ev.events = EPOLLIN | EPOLLONESHOT;
// Only one thread gets the event; must re-arm after handling
```

## Comparison: select vs poll vs epoll vs kqueue vs io_uring

```
| Feature            | select       | poll         | epoll        | kqueue       | io_uring     |
|--------------------|------------- |------------- |------------- |------------- |------------- |
| Platform           | POSIX        | POSIX        | Linux        | BSD/macOS    | Linux 5.1+   |
| Max FDs            | FD_SETSIZE   | unlimited    | unlimited    | unlimited    | unlimited    |
|                    | (1024)       |              |              |              |              |
| Complexity (wait)  | O(n)         | O(n)         | O(ready)     | O(ready)     | O(ready)     |
| Complexity (add)   | O(1)         | O(1)         | O(1)         | O(1)         | O(1)         |
| State in kernel    | No           | No           | Yes          | Yes          | Yes          |
| Edge-triggered     | No           | No           | Yes          | Yes          | Yes          |
| Batch syscalls     | No           | No           | No           | Yes          | Yes          |
| Async I/O          | No           | No           | No           | No           | Yes          |
| Thundering herd    | Yes          | Yes          | EXCLUSIVE    | EV_CLEAR     | per-ring     |
```

## Go Runtime (netpoller)

```bash
# Go uses epoll internally for all network I/O
# The runtime netpoller (runtime/netpoll_epoll.go):
# 1. All goroutine socket reads/writes go through the netpoller
# 2. When a goroutine would block on I/O, it parks and registers with epoll
# 3. The sysmon thread calls epoll_wait and wakes goroutines when fds are ready
# 4. This is why Go can handle millions of goroutines doing I/O efficiently
```

## Debugging epoll

```bash
# Check epoll fd count for a process
ls -la /proc/$PID/fd | grep eventpoll

# Count watched fds per epoll instance
cat /proc/$PID/fdinfo/$(ls /proc/$PID/fd | head -1) 2>/dev/null

# Trace epoll syscalls
strace -e trace=epoll_create1,epoll_ctl,epoll_wait -p $PID

# Count epoll_wait calls per second
strace -c -e trace=epoll_wait -p $PID 2>&1 &
sleep 10; kill $!

# Check system-wide epoll stats
cat /proc/sys/fs/epoll/max_user_watches    # max watched fds per user
```

## Tips

- Always use `epoll_create1(EPOLL_CLOEXEC)` to prevent fd leaks across `exec()` calls
- Edge-triggered mode requires non-blocking fds and drain loops; forgetting either causes stalls or data loss
- Level-triggered is simpler and correct by default; use edge-triggered only when you need the performance
- Use `EPOLLRDHUP` to detect half-closed connections without an extra `read()` returning 0
- `EPOLLONESHOT` is essential in multi-threaded servers to prevent two threads handling the same fd
- `EPOLLEXCLUSIVE` (Linux 4.5+) solves the thundering herd problem for accept() on shared epoll instances
- `SO_REUSEPORT` with per-thread epoll instances scales better than shared epoll with `EPOLLEXCLUSIVE`
- Do not add the same fd to multiple epoll instances; the kernel allows it but behavior is confusing
- `epoll_wait` can return `EINTR` on signal delivery; always retry in a loop
- For timer events, use `timerfd_create` with epoll instead of `alarm()` or polling
- The kernel uses a red-black tree for the interest set and a linked list for the ready list
- On modern Linux (5.1+), consider `io_uring` for truly async I/O that eliminates syscall overhead entirely

## See Also

- inotify
- signals
- kernel
- strace
- namespaces

## References

- [epoll(7) man page](https://man7.org/linux/man-pages/man7/epoll.7.html)
- [epoll_create(2)](https://man7.org/linux/man-pages/man2/epoll_create.2.html)
- [epoll_ctl(2)](https://man7.org/linux/man-pages/man2/epoll_ctl.2.html)
- [epoll_wait(2)](https://man7.org/linux/man-pages/man2/epoll_wait.2.html)
- [The C10K Problem — Dan Kegel](http://www.kegel.com/c10k.html)
- [io_uring and epoll comparison](https://kernel.dk/io_uring.pdf)
- [Linux Kernel: fs/eventpoll.c](https://github.com/torvalds/linux/blob/master/fs/eventpoll.c)
