# io_uring Internals -- Ring Buffers, Zero-Copy, and High-Performance Networking

> *io_uring replaces the traditional syscall-per-operation model with shared memory ring buffers, enabling millions of I/O operations per second with minimal kernel transitions. Understanding its memory layout, submission flow, and advanced features is essential for building the next generation of Linux network services.*

---

## 1. Ring Buffer Shared Memory Layout

### The Core Design

io_uring's performance comes from eliminating syscalls via shared memory. The kernel and userspace communicate through three memory-mapped regions:

1. **SQ Ring** -- submission queue metadata (head, tail, flags, array of SQE indices)
2. **CQ Ring** -- completion queue metadata (head, tail, flags, inline CQE array)
3. **SQE Array** -- contiguous array of submission queue entries

```
              Userspace Process                    Kernel
              ================                    ======

  mmap'd region 1: SQ Ring
  +---------------------------+
  | sq_head (kernel updates)  |-----> kernel reads SQEs from here
  | sq_tail (user updates)    |
  | sq_ring_mask              |
  | sq_ring_entries           |
  | sq_flags                  |      IORING_SQ_NEED_WAKEUP flag
  | sq_array[N]               |      indirection: array[i] -> SQE index
  +---------------------------+

  mmap'd region 2: SQE Array
  +---------------------------+
  | sqe[0]                    |      64 bytes each
  | sqe[1]                    |
  | ...                       |
  | sqe[N-1]                  |
  +---------------------------+

  mmap'd region 3: CQ Ring
  +---------------------------+
  | cq_head (user updates)    |
  | cq_tail (kernel updates)  |-----> kernel writes CQEs here
  | cq_ring_mask              |
  | cq_ring_entries           |
  | cq_overflow               |      count of dropped CQEs (CQ was full)
  | cqes[M]                   |      16 bytes each, inline in CQ ring
  +---------------------------+
```

### Memory Mapping Details

`io_uring_setup()` returns a file descriptor and populates `struct io_uring_params` with offsets for mmap:

| Offset Constant         | What It Maps          | Size Formula                                    |
|:------------------------|:----------------------|:------------------------------------------------|
| `IORING_OFF_SQ_RING`    | SQ ring metadata      | `params.sq_off.array + entries * sizeof(__u32)` |
| `IORING_OFF_CQ_RING`    | CQ ring + CQE array   | `params.cq_off.cqes + cq_entries * sizeof(struct io_uring_cqe)` |
| `IORING_OFF_SQES`       | SQE array             | `entries * sizeof(struct io_uring_sqe)`         |

Key sizing rules:
- SQ and SQE entries are always a power of two (rounded up from the requested count).
- CQ entries default to 2x SQ entries. Override with `IORING_SETUP_CQSIZE` in params.flags and set `params.cq_entries`.
- SQE is 64 bytes. CQE is 16 bytes (or 32 bytes with `IORING_SETUP_CQE32`).
- The SQ has an indirection array (`sq_array`) that maps SQ slots to SQE indices. This allows out-of-order SQE reuse.

### Ring Index Arithmetic

Both rings are single-producer, single-consumer (SPSC) with power-of-two masking:

```
available_slots = sq_entries - (sq_tail - sq_head)
next_index      = sq_tail & sq_ring_mask
sqe_to_fill     = &sqes[sq_array[next_index]]

// After filling:
write_barrier();          // ensure SQE data visible before tail update
sq_tail++;                // atomic store (kernel reads this)
```

For the CQ ring, the roles are reversed: kernel updates `cq_tail`, userspace updates `cq_head`.

The memory barriers are critical. On x86, a compiler barrier suffices (store ordering is guaranteed by hardware). On ARM/RISC-V, explicit `dmb` / `fence` instructions are required. liburing handles this portably with `io_uring_smp_store_release()` and `io_uring_smp_load_acquire()`.

---

## 2. Submission and Completion Flow

### Without SQPOLL (Normal Mode)

```
1. Userspace fills SQE(s) in the SQE array
2. Userspace updates sq_tail (with write barrier)
3. Userspace calls io_uring_enter(fd, to_submit, min_complete, flags)
   - to_submit > 0: kernel consumes SQEs from SQ ring
   - min_complete > 0: kernel blocks until that many CQEs are available
   - flags & IORING_ENTER_GETEVENTS: enable waiting for completions
4. Kernel processes SQEs, posts CQEs to CQ ring, updates cq_tail
5. Userspace reads CQEs, updates cq_head (with write barrier)
```

Cost: one syscall per batch. Batching amortizes the cost -- submitting 32 operations in one `io_uring_enter()` call is 32x more efficient than 32 individual `read()`/`write()` calls.

### With SQPOLL

```
1. Userspace fills SQE(s) and updates sq_tail
2. Kernel SQPOLL thread detects new SQEs (polls sq_tail continuously)
3. SQPOLL thread processes SQEs, posts CQEs
4. Userspace reads CQEs from CQ ring

No io_uring_enter() needed in steady state.
```

The only exception: if the SQPOLL thread has gone idle (no new SQEs for `sq_thread_idle` milliseconds), it sets the `IORING_SQ_NEED_WAKEUP` flag in `sq_flags`. Userspace must then call `io_uring_enter()` with `IORING_ENTER_SQ_WAKEUP` to restart it.

```c
// Correct SQPOLL submission pattern
io_uring_smp_store_release(sq_tail, new_tail);

if (IO_URING_READ_ONCE(*sq_flags) & IORING_SQ_NEED_WAKEUP) {
    io_uring_enter(ring_fd, 0, 0, IORING_ENTER_SQ_WAKEUP, NULL);
}
```

### CQ Overflow Handling

If the CQ ring is full when the kernel needs to post a CQE, the kernel increments `cq_overflow` and stores the CQE in an internal overflow list. Userspace can retrieve overflowed CQEs by calling `io_uring_enter()` with `IORING_ENTER_GETEVENTS` after draining the CQ ring.

To avoid overflow entirely: size the CQ ring large enough (4x SQ is a safe margin for multishot workloads) and drain CQEs promptly.

---

## 3. SQPOLL Thread Model

### Architecture

The SQPOLL thread is a kernel thread (`io_uring-sq` in `ps` output) bound to the io_uring instance. It runs in kernel context and has direct access to the shared rings.

```
           +-----------+
           | Userspace |
           |  thread   |
           +-----+-----+
                 |
        writes SQEs, reads CQEs
        (no syscall)
                 |
    =============|============ shared memory boundary
                 |
           +-----+-----+
           | SQPOLL    |
           | kthread   |-----> processes SQEs, executes I/O
           +-----------+
                 |
           polls sq_tail continuously
           sleeps after sq_thread_idle ms
```

### CPU Affinity

By default, the SQPOLL thread runs on any CPU. Pin it with:

```c
params.flags = IORING_SETUP_SQPOLL | IORING_SETUP_SQ_AFF;
params.sq_thread_cpu = 3;   // pin to CPU 3
```

For networking servers, pin the SQPOLL thread to the same CPU that handles the NIC's interrupt (check `/proc/interrupts`). This maximizes cache locality for packet data.

### Privilege Requirements

- Kernel 5.1-5.12: `CAP_SYS_ADMIN` required for SQPOLL
- Kernel 5.13+: `CAP_SYS_NICE` sufficient (or root)
- Kernel 5.19+: unprivileged SQPOLL allowed if `io_uring_group` GID matches the process

### When to Use SQPOLL

SQPOLL eliminates syscall overhead but burns a CPU core. The break-even point:

- Below ~100,000 IOPS: syscall overhead is negligible. SQPOLL wastes CPU.
- 100,000-500,000 IOPS: SQPOLL starts to win. Syscall overhead is 1-5 microseconds each.
- Above 500,000 IOPS: SQPOLL is essential. A single `io_uring_enter()` costs ~200ns, but at 1M IOPS that is 200ms/sec of syscall overhead.

---

## 4. Zero-Copy Send and Receive

### Fixed Buffers (io_uring_register)

Registering buffers with `IORING_REGISTER_BUFFERS` pins the pages in kernel memory and creates a persistent mapping. Subsequent `IORING_OP_READ_FIXED` / `IORING_OP_WRITE_FIXED` operations skip `get_user_pages()` entirely.

Performance impact:
- `get_user_pages()` costs 1-3 microseconds per call (TLB walk, page table lock, refcount)
- Fixed buffers reduce this to near zero
- Most impactful for small I/O (4KB-64KB) where per-operation overhead dominates

### Zero-Copy Send (IORING_OP_SEND_ZC)

Available since kernel 6.0. The kernel sends data directly from the userspace buffer without copying to a kernel socket buffer.

```c
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_send_zc(sqe, sockfd, buf, len, msg_flags, zc_flags);
```

Important constraints:
- The userspace buffer must not be modified until the kernel signals completion.
- Two CQEs are generated: first with `IORING_CQE_F_NOTIF` (notification that the buffer is safe to reuse), second with the actual send result.
- Most beneficial for large sends (64KB+). For small messages, the copy cost is negligible compared to the notification overhead.

### Zero-Copy Receive

True zero-copy receive (kernel to userspace without copy) is not yet available as a general feature. The current approach uses buffer rings, where the kernel selects pre-registered buffers and writes directly into them, avoiding one copy (kernel intermediate buffer to user buffer) but not the NIC-to-kernel copy.

For true zero-copy receive, use AF_XDP (XDP sockets) which map NIC ring buffers directly into userspace.

---

## 5. Performance: io_uring vs epoll for Echo Server

### Methodology

Benchmark: TCP echo server handling 1000 concurrent connections, each sending 64-byte messages in a tight loop. Measured on a single core, kernel 6.1, AMD EPYC 7763.

### Results

```
                        epoll            io_uring         io_uring+SQPOLL
Requests/sec            412,000          687,000          891,000
Avg latency (p50)       48 us            29 us            22 us
Tail latency (p99)      215 us           89 us            51 us
Syscalls/req            3 (wait+recv+send) 0.5 (batched)  0 (SQPOLL)
CPU user/sys split      35%/65%          62%/38%          78%/22%
```

### Why io_uring Wins

1. **Syscall elimination**: epoll requires `epoll_wait` + `recv` + `send` = 3 syscalls per request. io_uring batches all operations into shared memory writes.

2. **Reduced context switches**: Each syscall triggers a user-to-kernel transition (~200ns on modern x86). At 400K ops/sec, that is 240ms/sec of pure transition overhead.

3. **Kernel-side batching**: The kernel processes multiple SQEs in a single pass, amortizing lock acquisition and scheduling overhead.

4. **Cache efficiency**: With SQPOLL, the polling thread keeps the ring buffer data hot in L1/L2 cache. epoll's `epoll_wait` path touches more kernel data structures (red-black tree, wait queues).

5. **Completion-driven I/O**: epoll tells you a socket is readable; you must then call `recv()`. io_uring combines "check readiness" and "do the I/O" into one operation, halving the kernel interactions.

### When epoll is Still Fine

- Low connection counts (< 1000 concurrent)
- Low throughput (< 50,000 ops/sec)
- Applications where I/O is not the bottleneck (CPU-bound processing)
- Need for portability (epoll works on older kernels, kqueue on BSD; io_uring is Linux 5.1+ only)

---

## 6. Multishot Accept Pattern

### The Problem with Traditional Accept

With epoll or basic io_uring, each `accept()` call returns one connection. For a server handling thousands of new connections per second, the resubmission overhead accumulates:

```
Traditional: submit_accept -> CQE(fd=10) -> submit_accept -> CQE(fd=11) -> ...
Multishot:   submit_accept -> CQE(fd=10) -> CQE(fd=11) -> CQE(fd=12) -> ...
```

### Implementation

```c
// Submit once
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_multishot_accept(sqe, listen_fd, NULL, NULL, 0);
sqe->flags |= IOSQE_FIXED_FILE;      // listen_fd is registered
io_uring_sqe_set_data64(sqe, ACCEPT_TAG);
io_uring_submit(&ring);

// Reap loop
while (1) {
    io_uring_wait_cqe(&ring, &cqe);

    if (cqe->user_data == ACCEPT_TAG) {
        if (cqe->res >= 0) {
            int client_fd = cqe->res;
            setup_client(client_fd);
        }
        // Check if multishot is still active
        if (!(cqe->flags & IORING_CQE_F_MORE)) {
            // Multishot terminated (error or resource limit)
            // Resubmit
            sqe = io_uring_get_sqe(&ring);
            io_uring_prep_multishot_accept(sqe, listen_fd, NULL, NULL, 0);
            io_uring_submit(&ring);
        }
    }
    io_uring_cqe_seen(&ring, cqe);
}
```

### Multishot Accept Termination

The kernel terminates multishot accept when:
- The CQ ring is full (no room for another CQE)
- An error occurs on the listening socket
- The io_uring instance is being shut down

Always check `IORING_CQE_F_MORE` and resubmit if absent. A robust server should treat termination as routine, not exceptional.

---

## 7. Buffer Ring Management

### Why Buffer Rings Exist

Traditional recv requires pre-allocating a buffer per connection. With 100,000 connections and 4KB buffers, that is 400MB of memory mostly sitting idle (most connections are inactive at any moment).

Buffer rings let the kernel select a buffer at recv time from a shared pool. Only active connections consume buffers.

### Ring Structure

```
struct io_uring_buf_ring {
    union {
        struct {
            __u64 resv1;
            __u32 resv2;
            __u16 resv3;
            __u16 tail;          // userspace updates this
        };
        struct io_uring_buf bufs[];
    };
};

struct io_uring_buf {
    __u64 addr;                   // buffer virtual address
    __u32 len;                    // buffer length
    __u16 bid;                    // buffer ID (returned in CQE)
    __u16 resv;
};
```

### Lifecycle

```
1. Register buffer ring: io_uring_register_buf_ring()
   - Allocates shared memory for the ring metadata
   - Assigns a buffer group ID (bgid)

2. Populate buffers: io_uring_buf_ring_add() for each buffer
   - Adds buffer address, length, and ID to the ring
   - Call io_uring_buf_ring_advance() to make buffers visible to kernel

3. Submit recv with IOSQE_BUFFER_SELECT flag and buf_group set
   - Kernel picks a buffer from the ring when data arrives

4. On CQE completion:
   - buf_id = cqe->flags >> IORING_CQE_BUFFER_SHIFT
   - Process data in bufs[buf_id]
   - Return buffer: io_uring_buf_ring_add() + io_uring_buf_ring_advance()
```

### Sizing Guidelines

| Concurrent Active Connections | Buffer Size | Buffer Count | Total Memory |
|:------------------------------|:------------|:-------------|:-------------|
| 1,000                         | 4 KB        | 2,048        | 8 MB         |
| 10,000                        | 4 KB        | 16,384       | 64 MB        |
| 100,000                       | 2 KB        | 32,768       | 64 MB        |

Over-provision by 2-4x the expected concurrent active connections. If all buffers are exhausted, the kernel returns `-ENOBUFS` in the CQE and terminates multishot recv for that connection.

### Combining with Multishot Recv

Buffer rings and multishot recv are designed to work together:

```c
// One-time setup per client connection
struct io_uring_sqe *sqe = io_uring_get_sqe(&ring);
io_uring_prep_recv_multishot(sqe, client_fd, NULL, 0, 0);
sqe->flags |= IOSQE_BUFFER_SELECT;
sqe->buf_group = RECV_BUF_GROUP;
io_uring_sqe_set_data64(sqe, make_tag(client_fd, RECV));
```

Each incoming message produces a CQE with the buffer ID. The application processes the data and returns the buffer to the ring. No resubmission of the recv SQE is needed as long as `IORING_CQE_F_MORE` is set.

---

## 8. Kernel Version Feature Matrix

| Feature                        | Minimum Kernel | Notes                                      |
|:-------------------------------|:--------------:|:-------------------------------------------|
| Basic file I/O (readv, writev) | 5.1            | Initial release                            |
| `IORING_SETUP_SQPOLL`         | 5.1            | Required CAP_SYS_ADMIN until 5.13         |
| Linked SQEs                   | 5.3            | `IOSQE_IO_LINK`                            |
| `accept`, `connect`           | 5.5            | Networking support begins                  |
| `send`, `recv`                | 5.6            | Non-vectored socket I/O                    |
| `IORING_OP_PROVIDE_BUFFERS`   | 5.7            | Kernel-selected buffers (legacy API)       |
| Fixed files update             | 5.12           | `IORING_REGISTER_FILES_UPDATE`             |
| Unprivileged SQPOLL           | 5.13           | CAP_SYS_NICE sufficient                   |
| `IORING_OP_CANCEL`            | 5.5            | Cancel by user_data                        |
| Cancel by fd                   | 5.19           | `IORING_ASYNC_CANCEL_FD`                  |
| Multishot accept               | 5.19           | `IORING_ACCEPT_MULTISHOT`                  |
| Buffer rings (`buf_ring`)     | 5.19           | Replaces `PROVIDE_BUFFERS`                 |
| `IORING_SETUP_SINGLE_ISSUER` | 5.20           | Optimization for single-threaded use       |
| Zero-copy send (`SEND_ZC`)   | 6.0            | Two CQEs per operation                    |
| Multishot recv                 | 6.0            | Requires buffer ring                       |
| `IORING_SETUP_DEFER_TASKRUN` | 6.1            | Deferred work processing                  |
| `IORING_OP_WAITID`           | 6.7            | Wait for child process state change       |
| `IORING_OP_FUTEX_WAIT/WAKE`  | 6.7            | Futex operations via io_uring             |

### Checking Kernel Support at Runtime

```c
// Probe supported opcodes
struct io_uring_probe *probe = io_uring_get_probe();
if (io_uring_opcode_supported(probe, IORING_OP_SEND_ZC)) {
    // Zero-copy send available
}
io_uring_free_probe(probe);
```

### Distribution Kernel Versions (Approximate)

```
Ubuntu 20.04 LTS    — 5.4   (basic networking)
Ubuntu 22.04 LTS    — 5.15  (full networking, no multishot)
Ubuntu 24.04 LTS    — 6.8   (all features)
Debian 12           — 6.1   (all features)
RHEL 9 / Rocky 9    — 5.14  (networking, no multishot/buf_ring)
Fedora 40           — 6.8   (all features)
```
