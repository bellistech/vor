# io_uring

Linux async I/O interface (kernel 5.1+) using shared memory ring buffers between userspace and kernel, eliminating syscall overhead for high-throughput I/O and networking.

## Overview

io_uring provides two ring buffers shared between userspace and kernel:
- **Submission Queue (SQ)**: userspace writes I/O requests (SQEs) for the kernel to consume
- **Completion Queue (CQ)**: kernel writes results (CQEs) for userspace to consume

```
Userspace                          Kernel
   |                                  |
   |--- SQE (submit request) ------->|
   |                                  |--- execute I/O
   |<--- CQE (completion result) ----|
   |                                  |
   No syscall needed if SQPOLL is on
```

## Syscalls

```
io_uring_setup(entries, params)    — create a new io_uring instance, returns fd
io_uring_enter(fd, to_submit, min_complete, flags, sig)
                                   — submit SQEs and/or wait for CQEs
io_uring_register(fd, opcode, arg, nr_args)
                                   — register files, buffers, eventfd for zero-copy
```

```c
// Minimal setup (C)
struct io_uring_params params = {};
int ring_fd = io_uring_setup(256, &params);  // 256 entries

// mmap the SQ and CQ rings
void *sq_ptr = mmap(0, params.sq_off.array + params.sq_entries * sizeof(__u32),
                    PROT_READ | PROT_WRITE, MAP_SHARED | MAP_POPULATE,
                    ring_fd, IORING_OFF_SQ_RING);
void *cq_ptr = mmap(0, params.cq_off.cqes + params.cq_entries * sizeof(struct io_uring_cqe),
                    PROT_READ | PROT_WRITE, MAP_SHARED | MAP_POPULATE,
                    ring_fd, IORING_OFF_CQ_RING);
void *sqes = mmap(0, params.sq_entries * sizeof(struct io_uring_sqe),
                  PROT_READ | PROT_WRITE, MAP_SHARED | MAP_POPULATE,
                  ring_fd, IORING_OFF_SQES);
```

## SQE and CQE Structures

### Submission Queue Entry (SQE)

```c
struct io_uring_sqe {
    __u8    opcode;       // operation type (IORING_OP_*)
    __u8    flags;        // IOSQE_FIXED_FILE, IOSQE_IO_LINK, etc.
    __u16   ioprio;       // I/O priority
    __s32   fd;           // file descriptor (or index if fixed)
    __u64   off;          // offset
    __u64   addr;         // buffer address (or buf_index for fixed bufs)
    __u32   len;          // buffer length
    union {
        __kernel_rwf_t  rw_flags;
        __u32           fsync_flags;
        __u16           poll_events;
        __u32           sync_range_flags;
        __u32           msg_flags;        // for send/recv
        __u32           timeout_flags;
        __u32           accept_flags;
        __u32           cancel_flags;
        __u32           splice_flags;
    };
    __u64   user_data;    // opaque value returned in CQE
    // ... additional fields for buf_index, personality, splice_fd_in
};
```

### Completion Queue Entry (CQE)

```c
struct io_uring_cqe {
    __u64   user_data;    // copied from SQE — identifies which request completed
    __s32   res;          // result (bytes transferred, or negative errno)
    __u32   flags;        // IORING_CQE_F_BUFFER (buffer selected), IORING_CQE_F_MORE (multishot)
};
```

## liburing Helper Library

liburing abstracts away mmap setup, memory barriers, and ring management.

```c
#include <liburing.h>

// Setup
struct io_uring ring;
io_uring_queue_init(256, &ring, 0);            // 256 entries, no flags

// Prepare a read
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_read(sqe, fd, buf, buf_len, offset);
io_uring_sqe_set_data(sqe, user_data_ptr);     // tag for identification

// Submit
io_uring_submit(&ring);                         // calls io_uring_enter internally

// Reap completion
struct io_uring_cqe *cqe;
io_uring_wait_cqe(&ring, &cqe);                // blocks until one CQE ready
int result = cqe->res;                          // bytes read, or -errno
void *data = io_uring_cqe_get_data(cqe);
io_uring_cqe_seen(&ring, cqe);                 // advance CQ head

// Cleanup
io_uring_queue_exit(&ring);
```

## Supported Operations

```
IORING_OP_READV / WRITEV      — vectored read/write (readv/writev)
IORING_OP_READ / WRITE        — simple read/write (5.6+)
IORING_OP_ACCEPT              — accept incoming connection
IORING_OP_CONNECT             — initiate outbound connection
IORING_OP_SEND / RECV         — send/recv on connected socket
IORING_OP_SENDMSG / RECVMSG   — sendmsg/recvmsg (UDP, ancillary data)
IORING_OP_POLL_ADD / REMOVE   — level-triggered poll on fd
IORING_OP_TIMEOUT              — timer (relative or absolute)
IORING_OP_SPLICE               — splice between two fds (zero-copy pipe)
IORING_OP_PROVIDE_BUFFERS      — register buffer pool for kernel selection
IORING_OP_CANCEL               — cancel a pending operation
IORING_OP_SHUTDOWN             — shutdown a socket
IORING_OP_CLOSE                — close a file descriptor
IORING_OP_OPENAT / STATX       — file open / stat
IORING_OP_SEND_ZC              — zero-copy send (5.20+)
IORING_OP_RECV_MULTI           — multishot recv (6.0+, see below)
```

## Fixed Files and Fixed Buffers

Registering files/buffers avoids per-operation `fget()`/`fput()` overhead.

```c
// Register fixed files
int fds[64];
// ... populate fds with open file descriptors ...
io_uring_register(ring_fd, IORING_REGISTER_FILES, fds, 64);

// Use fixed file in SQE
sqe->flags |= IOSQE_FIXED_FILE;
sqe->fd = 3;   // index into registered array, not the real fd

// Update a registered file slot
struct io_uring_files_update up = { .offset = 3, .fds = &new_fd };
io_uring_register(ring_fd, IORING_REGISTER_FILES_UPDATE, &up, 1);

// Register fixed buffers (pins pages, avoids get_user_pages per I/O)
struct iovec iovs[8];
// ... populate iovs ...
io_uring_register(ring_fd, IORING_REGISTER_BUFFERS, iovs, 8);

// Use with IORING_OP_READ_FIXED / IORING_OP_WRITE_FIXED
io_uring_prep_read_fixed(sqe, fd, buf, len, offset, buf_index);
```

## SQPOLL Mode

Kernel-side polling thread consumes SQEs without any `io_uring_enter()` syscall.

```c
struct io_uring_params params = {};
params.flags = IORING_SETUP_SQPOLL;
params.sq_thread_idle = 2000;          // idle timeout in ms before thread sleeps

int ring_fd = io_uring_setup(256, &params);

// With liburing
struct io_uring ring;
io_uring_queue_init(256, &ring, IORING_SETUP_SQPOLL);

// Submit: just write SQE and update SQ tail — no syscall
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_recv(sqe, sockfd, buf, len, 0);
io_uring_submit(&ring);  // no-op if SQPOLL thread is awake; kicks it if asleep
```

- Requires `CAP_SYS_NICE` or `IORING_SETUP_SQPOLL` privilege (sysctl `io_uring_group` in 6.x)
- SQ polling thread consumes one CPU core when active
- `sq_thread_idle`: if no new SQEs for this duration, thread sleeps (needs `io_uring_enter` to wake)

## Linked SQEs (Chaining Operations)

```c
// Chain: read -> process -> write (sequential dependency)
struct io_uring_sqe *sqe1 = io_uring_get_sqe(&ring);
io_uring_prep_read(sqe1, in_fd, buf, len, 0);
sqe1->flags |= IOSQE_IO_LINK;         // next SQE depends on this one

struct io_uring_sqe *sqe2 = io_uring_get_sqe(&ring);
io_uring_prep_write(sqe2, out_fd, buf, len, 0);
// sqe2 executes only if sqe1 succeeds

io_uring_submit(&ring);                // submit entire chain at once

// IOSQE_IO_HARDLINK — next SQE runs even if this one fails
// IOSQE_IO_DRAIN    — wait for all prior SQEs to complete first
```

## Multishot Operations

Single SQE produces multiple CQEs. Eliminates resubmission overhead.

```c
// Multishot accept (5.19+): one SQE, CQE per accepted connection
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_multishot_accept(sqe, listen_fd, NULL, NULL, 0);
io_uring_submit(&ring);

// Each CQE has:
//   cqe->res = new client fd (or -errno)
//   cqe->flags & IORING_CQE_F_MORE  — more CQEs coming (still active)
//   If IORING_CQE_F_MORE is NOT set, the multishot was terminated — resubmit

// Multishot recv (6.0+): receives into kernel-selected buffers
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_recv_multishot(sqe, client_fd, NULL, 0, 0);
sqe->flags |= IOSQE_BUFFER_SELECT;
sqe->buf_group = buf_group_id;         // buffer ring group
```

## Buffer Ring (io_uring_buf_ring)

Kernel-managed buffer pool. Kernel picks a buffer for each recv/read.

```c
// Setup buffer ring (5.19+)
struct io_uring_buf_reg reg = {
    .ring_addr = 0,                     // let kernel allocate
    .ring_entries = 128,                // number of buffers
    .bgid = 1,                          // buffer group ID
};
io_uring_register_buf_ring(&ring, &reg, 0);

struct io_uring_buf_ring *br = (void *)reg.ring_addr;
io_uring_buf_ring_init(br);

// Add buffers to the ring
for (int i = 0; i < 128; i++) {
    io_uring_buf_ring_add(br, bufs[i], BUF_SIZE, i, 127, i);
}
io_uring_buf_ring_advance(br, 128);

// In CQE, extract which buffer was used:
int buf_id = cqe->flags >> IORING_CQE_BUFFER_SHIFT;
```

## Networking Patterns

### TCP Echo Server

```c
// 1. Setup listening socket and submit multishot accept
io_uring_prep_multishot_accept(sqe, listen_fd, NULL, NULL, 0);
io_uring_submit(&ring);

// 2. Event loop
while (1) {
    io_uring_wait_cqe(&ring, &cqe);
    struct conn_info *info = io_uring_cqe_get_data(cqe);

    switch (info->type) {
    case ACCEPT:
        int client_fd = cqe->res;
        // Submit recv for new client
        sqe = io_uring_get_sqe(&ring);
        io_uring_prep_recv(sqe, client_fd, buf, BUF_SIZE, 0);
        io_uring_sqe_set_data(sqe, make_conn(client_fd, RECV));
        break;
    case RECV:
        if (cqe->res <= 0) { close(info->fd); break; }
        // Echo back: submit send
        sqe = io_uring_get_sqe(&ring);
        io_uring_prep_send(sqe, info->fd, buf, cqe->res, 0);
        io_uring_sqe_set_data(sqe, make_conn(info->fd, SEND));
        break;
    case SEND:
        // Resubmit recv
        sqe = io_uring_get_sqe(&ring);
        io_uring_prep_recv(sqe, info->fd, buf, BUF_SIZE, 0);
        io_uring_sqe_set_data(sqe, make_conn(info->fd, RECV));
        break;
    }
    io_uring_cqe_seen(&ring, cqe);
    io_uring_submit(&ring);
}
```

### Proxy Pattern (splice)

```c
// Zero-copy proxy: splice from client -> pipe -> backend
int pipefd[2];
pipe(pipefd);

struct io_uring_sqe *sqe1 = io_uring_get_sqe(&ring);
io_uring_prep_splice(sqe1, client_fd, -1, pipefd[1], -1, SPLICE_LEN, 0);
sqe1->flags |= IOSQE_IO_LINK;

struct io_uring_sqe *sqe2 = io_uring_get_sqe(&ring);
io_uring_prep_splice(sqe2, pipefd[0], -1, backend_fd, -1, SPLICE_LEN, 0);

io_uring_submit(&ring);
```

## Cancel Operations

```c
// Cancel by user_data
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_cancel(sqe, user_data_ptr, 0);
io_uring_submit(&ring);

// Cancel all matching a file descriptor (5.19+)
io_uring_prep_cancel_fd(sqe, target_fd, IORING_ASYNC_CANCEL_FD);

// The cancelled operation's CQE will have res = -ECANCELED
```

## io_uring vs epoll vs select/poll

```
                  select/poll     epoll            io_uring
Mechanism         copy fd set     kernel list      shared ring buffer
Per-call cost     O(n) fds        O(1) ready       O(0) with SQPOLL
Readiness only    yes             yes              no (does actual I/O)
Max fds           FD_SETSIZE(1024) unlimited       unlimited
Batching          no              epoll_wait       submit+reap batched
Syscalls/op       2 (poll+rw)     2 (wait+rw)      0-1 (combined)
Kernel thread     no              no               SQPOLL option
Zero-copy         no              no               fixed bufs, send_zc
```

- **select/poll**: O(n) scan per call, 1024 fd limit (select). Legacy.
- **epoll**: O(1) readiness notification, but still requires separate read/write syscalls.
- **io_uring**: Combines readiness + I/O in one step. With SQPOLL, zero syscalls in steady state.

## Tips

- Start with liburing, not raw syscalls. The memory barrier and ring index management is error-prone.
- Size the CQ at least 2x the SQ. The kernel may generate more CQEs than SQEs submitted (multishot, errors). Use `IORING_SETUP_CQSIZE` to set CQ size independently.
- Always check `cqe->res` for negative values -- they are negated errno codes, not positive error numbers.
- SQPOLL burns a CPU core. Only use it when syscall overhead is the actual bottleneck (very high IOPS workloads).
- Use multishot accept instead of resubmitting accept after each connection. Resubmit only if `IORING_CQE_F_MORE` is not set in the CQE flags.
- For buffer rings, over-provision buffers (2-4x expected concurrent operations). Running out of buffers causes `-ENOBUFS` and drops the multishot.
- Linked SQEs fail the entire chain if one link fails. Use `IOSQE_IO_HARDLINK` if subsequent operations should proceed regardless.
- io_uring is Linux-only. For portable async I/O, use a library that falls back to epoll/kqueue (e.g., Tokio in Rust, libuv for C).
- Kernel 5.1 has basic file I/O only. Networking operations require 5.5+. Multishot accept needs 5.19+. Buffer rings need 5.19+. Zero-copy send needs 6.0+.
- Use `IORING_SETUP_SINGLE_ISSUER` (5.20+) and `IORING_SETUP_DEFER_TASKRUN` (6.1+) for single-threaded servers to reduce kernel overhead.

## See Also

- tcp, udp, system/epoll, xdp

## References

- [io_uring man page (io_uring_setup)](https://man7.org/linux/man-pages/man2/io_uring_setup.2.html)
- [liburing GitHub Repository](https://github.com/axboe/liburing)
- [Lord of the io_uring -- Unofficial Guide](https://unixism.net/loti/)
- [Kernel Documentation -- io_uring](https://www.kernel.org/doc/html/latest/userspace-api/io_uring.html)
- [Efficient IO with io_uring -- Jens Axboe (Original Design Paper)](https://kernel.dk/io_uring.pdf)
- [io_uring and networking in 2023 -- Jens Axboe (LPC Talk)](https://lpc.events/event/17/contributions/1550/)
