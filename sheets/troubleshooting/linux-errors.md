# Linux Errors

Verbatim errno values, dmesg patterns, kernel oops, signal exit codes, and diagnostic recipes for terminal-bound Linux troubleshooting.

## Setup

`errno` is a thread-local integer set by failing system calls and library functions. The integer maps to a symbolic constant in `<errno.h>` (e.g. `2 == ENOENT`). It is *not* set on success; check the syscall return first, then read `errno`. The standard says `errno` is reset to zero by the program if it wants a clean baseline; the kernel does not zero it on success.

```bash
# strerror(3) — print the string for a numeric errno
man 3 strerror
man 3 errno
man 7 signal
man 5 proc
```

```bash
# Print the exit status of the last command
ls /no/such/path
echo $?           # 2  (ENOENT, but bash also remaps to 2 generically)

# After a process exits via signal, $? = 128 + signal_number
sleep 60 &
kill -9 $!
wait $!; echo $?  # 137  (128 + 9 = SIGKILL)
```

```bash
# Look up errno number → name
errno 2           # from moreutils
# ENOENT 2 No such file or directory

errno -l          # list everything
errno -s "permission"   # search by substring
```

```python
# Python
import errno, os
try:
    os.open("/nope", os.O_RDONLY)
except OSError as e:
    print(e.errno, errno.errorcode[e.errno], os.strerror(e.errno))
    # 2 ENOENT No such file or directory
    if e.errno == errno.ENOENT:
        ...
```

```go
// Go
import (
    "errors"
    "io/fs"
    "os"
    "syscall"
)

_, err := os.Open("/nope")
if errors.Is(err, fs.ErrNotExist) { /* ENOENT */ }
if errors.Is(err, fs.ErrPermission) { /* EACCES or EPERM */ }

// Lower-level: extract syscall.Errno
var en syscall.Errno
if errors.As(err, &en) {
    switch en {
    case syscall.ENOENT: // ...
    case syscall.EACCES: // ...
    }
}
```

```c
/* C */
#include <errno.h>
#include <string.h>
#include <stdio.h>

if (open("/nope", O_RDONLY) == -1) {
    fprintf(stderr, "open: %s (errno=%d)\n", strerror(errno), errno);
    perror("open");          /* same effect, prefixed */
}
```

```bash
# Reference headers (Linux glibc)
ls /usr/include/asm-generic/errno-base.h   # 1-34 (POSIX)
ls /usr/include/asm-generic/errno.h        # 35-133 (Linux extensions)
ls /usr/include/errno.h                    # frontend that includes both

grep -E '^#define E' /usr/include/asm-generic/errno-base.h \
                     /usr/include/asm-generic/errno.h
```

```bash
# Quick lookup script
errno-name() { perl -MErrno -e 'for(keys%!){$!{$_}&&$_[0]==$!+0&&print"$_\n"}' "$1"; }
errno-name 13       # EACCES
```

## errno Catalog — Standard Errors (1-34)

These are POSIX errors, defined in `errno-base.h`. Numbers are stable across architectures (Linux MIPS/Alpha/SPARC differ for 35+).

### 1 EPERM — Operation not permitted

The user is not privileged for this operation (compare with `EACCES` which is about permission bits). Returned by `setuid()`, `mount()`, `chown()` to a different owner as non-root, `kill()` to a process you don't own, `nice()` to a higher priority, opening `/dev/kmem`, raw socket, `iptables` rules.

Fix: run as root (`sudo`), grant a capability (`setcap cap_net_raw=eip ./binary`), or restructure to not need privilege.

```bash
# verbatim
$ chown root:root /tmp/foo
chown: changing ownership of '/tmp/foo': Operation not permitted
$ ping -c1 8.8.8.8
ping: socktype: SOCK_RAW   # or "Operation not permitted" without setcap
```

### 2 ENOENT — No such file or directory

The path component does not exist. Returned by `open()`, `stat()`, `execve()` (if the binary path is missing — *not* if its interpreter is missing, that's `ENOEXEC` in older kernels or success-then-exec-failure today).

Confusing case: `execve` of a script returns `ENOENT` if the `#!/path` interpreter doesn't exist — you'll see `bash: ./script: /usr/bin/python3: bad interpreter: No such file or directory`.

```bash
$ cat /etc/this-file-does-not-exist
cat: /etc/this-file-does-not-exist: No such file or directory
$ ls -la /usr/bin/missing
ls: cannot access '/usr/bin/missing': No such file or directory
```

Fix: verify spelling, check `ls -la` of the parent dir, on a script check the shebang.

### 3 ESRCH — No such process

`kill(pid, ...)` to a non-existent PID; `ptrace()` against a non-tracee.

```bash
$ kill 999999
bash: kill: (999999) - No such process
```

### 4 EINTR — Interrupted system call

A blocking syscall was interrupted by a signal before it completed. POSIX-2001 says many syscalls auto-restart with `SA_RESTART`, but `read`/`write` on slow devices, `sigwait`, `pause` may still return `EINTR`.

Fix: retry the syscall in a loop. In libc programs use `TEMP_FAILURE_RETRY()` macro. In Go `read` from `os.File` is auto-retried.

```c
ssize_t n;
do { n = read(fd, buf, len); } while (n == -1 && errno == EINTR);
```

### 5 EIO — Input/output error

The hardware reported a transfer error (bad sector, disconnected USB, RAID degraded, network filesystem timeout that's been mapped to EIO).

```bash
$ cat /tmp/file_on_failing_disk
cat: /tmp/file_on_failing_disk: Input/output error
```

Fix: check `dmesg | tail` for kernel messages from the block layer (`ata1.00: failed command:`), `smartctl -a /dev/sda`, replace the drive.

### 6 ENXIO — No such device or address

The path exists but the device behind it is gone (unplugged USB tty, missing major/minor in `/dev`).

```bash
$ cat /dev/ttyUSB99
cat: /dev/ttyUSB99: No such device or address
```

### 7 E2BIG — Argument list too long

`execve(argv, envp)` where the combined size of `argv + envp + auxv` exceeds `MAX_ARG_STRLEN` (currently 1/4 of stack limit, default 128 KiB per arg, 32 pages total ≈ 128 KiB combined on x86_64 with 8 MiB stack).

```bash
$ rm /tmp/big-glob/*
bash: /usr/bin/rm: Argument list too long
```

Fix: pipe through `xargs`:

```bash
find /tmp/big-glob -mindepth 1 -delete
# or
find /tmp/big-glob -type f -print0 | xargs -0 rm
# or
printf '%s\0' /tmp/big-glob/* | xargs -0 rm
```

### 8 ENOEXEC — Exec format error

The kernel can't recognise the file format for `execve` — wrong architecture (ARM binary on x86_64), missing ELF magic, corrupted file. With binfmt_misc you may also see this if the registered handler is missing.

```bash
$ ./armbinary
bash: ./armbinary: cannot execute binary file: Exec format error
$ file ./armbinary
./armbinary: ELF 32-bit LSB executable, ARM, EABI5
```

Fix: install qemu-user-static for cross-arch, install correct architecture, or `dpkg --add-architecture armhf`.

### 9 EBADF — Bad file descriptor

The fd number is closed, never opened, or is a write-only fd being read (or vice versa). Frequent in code bugs after `close()` then accidentally reusing the variable.

```c
write(-1, "x", 1);  /* EBADF */
close(fd); read(fd, buf, 1);  /* EBADF after close */
```

### 10 ECHILD — No child processes

`wait()` / `waitpid()` returned this because no child exists or all have been reaped. Also if `SIGCHLD` is set to `SIG_IGN` — the kernel auto-reaps and `wait` finds nothing.

### 11 EAGAIN / EWOULDBLOCK — Resource temporarily unavailable

These are *the same number* on Linux (`#define EWOULDBLOCK EAGAIN`). Common returners:

- Non-blocking I/O when there's nothing to do yet (read on empty pipe with `O_NONBLOCK`).
- `fork()` hitting `RLIMIT_NPROC` (`ulimit -u`).
- `epoll`/`kqueue` style polling with no event.
- `SO_RCVTIMEO`/`SO_SNDTIMEO` socket timeout expiring.

```bash
$ for i in $(seq 1 100000); do ./fork-bomb & done
bash: fork: retry: Resource temporarily unavailable
```

Fix: raise `ulimit -u`, retry the call later, or restructure to reuse threads/processes.

### 12 ENOMEM — Out of memory

`malloc` would fail (with `vm.overcommit_memory=2`); `mmap` exceeds available; `fork` exceeds copy-on-write reservation; `pthread_create` with no stack space.

```bash
$ python -c 'x = b"x" * (1024**4)'
MemoryError
```

Fix: see OOM-killer section. Add swap, reduce working set, increase RLIMIT_AS.

### 13 EACCES — Permission denied

The permission bits / ACL don't allow the operation. Compare with `EPERM` (privileged-only operation). Returned by `open(O_RDONLY)` on a 0o600 file you don't own, `execve` of a non-executable file (`+x` missing), `bind()` on a privileged port (<1024) without `CAP_NET_BIND_SERVICE`.

```bash
$ cat /etc/shadow
cat: /etc/shadow: Permission denied
$ ./script.sh
bash: ./script.sh: Permission denied      # missing +x
```

Fix: `chmod +x`, fix ownership, run as the right user, grant capability for ports.

### 14 EFAULT — Bad address

The buffer pointer passed to a syscall points outside the process's address space. Almost always a programming bug — indicates the application passed a wild pointer or freed memory.

### 15 ENOTBLK — Block device required

`mount()` of something that isn't a block device (regular file without loopback). Use `mount -o loop`.

```bash
$ mount disk.img /mnt/img
mount: /mnt/img: failed to setup loop device for /home/.../disk.img.
$ mount -o loop disk.img /mnt/img       # fix
```

### 16 EBUSY — Device or resource busy

The resource is in use. `umount` while a process has a CWD or open fd in the mount; `rmdir` a directory that's a mount point; opening an exclusive device a second time; `rename` over an in-use binary on some filesystems.

```bash
$ umount /mnt/data
umount: /mnt/data: target is busy.
$ lsof +f -- /mnt/data        # find who holds it
$ fuser -mv /mnt/data
```

Fix: kill / migrate the user processes, or `umount -l /mnt/data` (lazy unmount).

### 17 EEXIST — File exists

`open(O_CREAT|O_EXCL)` when the file is there; `mkdir` of an existing directory; `link()` target exists.

```bash
$ mkdir /tmp/existing
mkdir: cannot create directory '/tmp/existing': File exists
$ mkdir -p /tmp/existing       # idempotent
```

### 18 EXDEV — Cross-device link

`link()` cannot make a hard link across filesystems; `rename()` cannot atomically move across filesystems.

```bash
$ ln /tmp/file /mnt/usb/file
ln: failed to create hard link '/mnt/usb/file' => '/tmp/file': Invalid cross-device link
```

Fix: use `cp` (and `rm`) for cross-fs moves; `mv` falls back to copy+unlink itself.

### 19 ENODEV — No such device

`open()` of a device node where the kernel has no driver for that major number; `mount(-t fstype)` where fstype isn't compiled in.

```bash
$ mount -t reiserfs /dev/sdb1 /mnt
mount: /mnt: unknown filesystem type 'reiserfs'.    # NODEV maps to this msg
$ modprobe reiserfs
```

### 20 ENOTDIR — Not a directory

A path component was expected to be a directory but is a regular file: `open("/etc/passwd/foo")`.

```bash
$ ls /etc/passwd/
ls: cannot access '/etc/passwd/': Not a directory
```

### 21 EISDIR — Is a directory

`open(O_WRONLY)` on a directory; `read()` on a directory (use `readdir`/`getdents64`); `unlink()` on a directory (use `rmdir`).

```bash
$ cat /etc
cat: /etc: Is a directory
```

### 22 EINVAL — Invalid argument

The kernel didn't like one of the parameters. The most underspecified errno in Linux — every syscall has its own list of triggers. Common: `mmap` with unaligned offset, `setsockopt` with wrong `optlen`, `ioctl` with wrong cmd for the device.

Fix: `strace -e trace=syscall_name -f ./binary` to see exactly which syscall and which parameter.

### 23 ENFILE — Too many open files in system

System-wide fd limit reached. Distinct from `EMFILE` (per-process). Read `/proc/sys/fs/file-nr` for current/max.

```bash
$ cat /proc/sys/fs/file-nr
4096    0       1048576   # allocated, free, max
$ sysctl fs.file-max=2097152
```

### 24 EMFILE — Too many open files

Per-process fd limit reached (`ulimit -n`, default 1024 historically, 1048576 on modern systemd). `accept()` / `open()` / `socket()` / `pipe()` all return this.

```bash
$ python3 -c 'import os; [os.open("/dev/null", 0) for _ in range(100000)]'
OSError: [Errno 24] Too many open files: '/dev/null'
$ ulimit -n
1024
$ ulimit -n 65536              # raise for current shell
```

Fix: `LimitNOFILE=65536` in systemd unit, `*  soft  nofile  65536` in `/etc/security/limits.conf`.

### 25 ENOTTY — Inappropriate ioctl for device

The fd you ran `ioctl()` on isn't a tty / isn't the type the cmd expects. Frequent surprise: `tcgetattr(stdin)` when stdin is a pipe — programs use it to detect "is this a tty".

```bash
$ echo | mc
Error opening terminal: dumb     # mc detects not-a-tty
```

### 26 ETXTBSY — Text file busy

Cannot `open(O_WRONLY)` or `unlink()` (on some filesystems / procfs setup) a binary that's currently being executed by some process.

```bash
$ cp newbinary /usr/bin/runningprog
cp: cannot create regular file '/usr/bin/runningprog': Text file busy
```

Fix: stop the running process, or `mv newbinary /usr/bin/runningprog.new && mv -f /usr/bin/runningprog.new /usr/bin/runningprog` (rename is allowed; the inode of the old binary stays valid for the running process).

### 27 EFBIG — File too large

Write past per-process file size limit (`ulimit -f` / `RLIMIT_FSIZE`) or past filesystem max-file-size (ext4 default 16 TiB).

### 28 ENOSPC — No space left on device

Filesystem full. Two distinct conditions:

```bash
$ df -h /                # block-level (bytes)
$ df -i /                # inode-level (count)
```

Inode exhaustion is common with many tiny files on ext4 (created with too few inodes); `df -i` shows IFree=0.

```bash
$ touch /tmp/x
touch: cannot touch '/tmp/x': No space left on device
$ du -sh /var/log /var/cache    # find the offender
$ find /var -size +100M -type f
```

### 29 ESPIPE — Illegal seek

`lseek()` on a pipe, FIFO, or socket. Use `read` to consume sequentially.

### 30 EROFS — Read-only file system

Filesystem mounted read-only, often after the kernel saw fs corruption and remounted-ro to protect itself.

```bash
$ touch /var/foo
touch: cannot touch '/var/foo': Read-only file system
$ dmesg | grep -i 'remount'
[12345.6] EXT4-fs (sda1): Remounting filesystem read-only
$ mount -o remount,rw /         # if disk is healthy
```

Fix: investigate `dmesg` for the *cause*. Don't blindly remount-rw if the disk is failing — that's how you corrupt further. Run `fsck` from rescue media.

### 31 EMLINK — Too many links

`link()` exceeds `LINK_MAX` (ext4: 65000 hard links per file).

### 32 EPIPE — Broken pipe

`write()` to a pipe whose read end was closed. Process also receives `SIGPIPE` (default action: terminate, exit code 141). Common with `head`:

```bash
$ yes | head -n 1
y
# yes process killed by SIGPIPE; exits 141
```

Fix: in C, `signal(SIGPIPE, SIG_IGN)` and check `EPIPE` from `write`. In Python, the runtime handles this. In Go, `os.Stdout.Write` returns the error.

### 33 EDOM — Numerical argument out of domain

Math function got out-of-domain input: `acos(2.0)`, `log(-1)`, `sqrt(-1)`.

### 34 ERANGE — Numerical result out of range

Result overflows the type: `strtol("99999999999999999999")`, `pow(10, 1000)` returns ±HUGE_VAL with errno=ERANGE.

## errno Catalog — Linux Extended Errors (35-130)

These are Linux additions, defined in `errno.h`. Numbers may differ on non-x86 architectures (MIPS/Alpha/SPARC use different values for compatibility with their old userland) — never compare with hardcoded numbers, use the symbolic name.

### 35 EDEADLK — Resource deadlock avoided

`fcntl(F_SETLKW)` would create a deadlock with another process; `pthread_mutex_lock` on a recursive mutex from the wrong owner.

### 36 ENAMETOOLONG — File name too long

A path component exceeds `NAME_MAX` (ext4: 255 bytes) or full path exceeds `PATH_MAX` (4096).

```bash
$ touch $(printf 'x%.0s' {1..300})
touch: cannot touch 'xxxxxx...': File name too long
```

### 37 ENOLCK — No locks available

Out of file-locking resources; rare on modern Linux; common on NFSv2/v3 if `lockd` is down.

### 38 ENOSYS — Function not implemented

Syscall doesn't exist on this kernel. Returned by glibc as a wrapper for `__NR_NotImplemented`. May indicate seccomp filter blocking the syscall (BPF_RET&SECCOMP_RET_ERRNO can return any errno; ENOSYS is conventional).

### 39 ENOTEMPTY — Directory not empty

`rmdir` of a non-empty directory; `rename` over a non-empty target on the same fs.

### 40 ELOOP — Too many levels of symbolic links

> 40 symlink resolutions while resolving a path. Often a self-loop: `ln -s a b; ln -s b a; cat a`.

```bash
$ ln -s loop loop && cat loop
cat: loop: Too many levels of symbolic links
```

### 88 ENOTSOCK — Socket operation on non-socket

`recv()` / `send()` / `bind()` on a non-socket fd (regular file).

### 95 EOPNOTSUPP — Operation not supported

Often `ENOTSUP` is the same: `#define ENOTSUP EOPNOTSUPP`. Filesystem doesn't support an operation (xattrs on FAT), socket doesn't support flag (`SO_REUSEPORT` on AF_UNIX).

### 97 EAFNOSUPPORT — Address family not supported by protocol

`socket(AF_INET6, ...)` on a kernel without IPv6; `getaddrinfo` family hint mismatch.

### 98 EADDRINUSE — Address already in use

`bind()` to a port already bound. Frequent post-restart of a server: kernel still has the socket in `TIME_WAIT`. Use `SO_REUSEADDR` (and `SO_REUSEPORT` for the multi-listener case).

```bash
$ ./server &
$ ./server                  # second instance
bind: Address already in use
$ ss -tlnp 'sport = :8080'  # find who has it
```

### 99 EADDRNOTAVAIL — Cannot assign requested address

`bind()` to an IP not on any local interface, or `bind()` to a port number out of range.

### 100 ENETDOWN — Network is down

The interface is administratively down; `ip link set X up`.

### 101 ENETUNREACH — Network is unreachable

No route for the destination subnet; `ip route show` shows the table.

### 104 ECONNRESET — Connection reset by peer

Peer sent a TCP RST. Often peer crashed mid-connection, peer's NAT/firewall dropped state, or peer's app explicitly `SO_LINGER 0` reset.

### 105 ENOBUFS — No buffer space available

Kernel out of socket buffer memory; raise `net.core.rmem_max` / `wmem_max`.

### 106 EISCONN — Transport endpoint is already connected

`connect()` on a socket already connected.

### 107 ENOTCONN — Transport endpoint is not connected

`send()` / `recv()` on a TCP socket not yet connected, or after `shutdown()`.

### 110 ETIMEDOUT — Connection timed out

TCP timed out (no SYN-ACK after SYN retransmits — default ~127 seconds). Also `connect()` with `SO_SNDTIMEO`.

```bash
$ curl http://10.255.255.1/
curl: (28) Failed to connect to 10.255.255.1 port 80 after 130000 ms: Connection timed out
```

### 111 ECONNREFUSED — Connection refused

Peer sent TCP RST in response to SYN. The host is up; the port is closed (no listener) or firewalled with REJECT.

```bash
$ ssh nothost
ssh: connect to host nothost port 22: Connection refused
```

### 113 EHOSTUNREACH — No route to host

ICMP unreachable received, or ARP failed for local subnet, or routing table has no path.

### 114 EALREADY — Operation already in progress

`connect()` on a non-blocking socket already mid-connect.

### 115 EINPROGRESS — Operation now in progress

`connect()` on a non-blocking socket — normal; poll for `POLLOUT` then `getsockopt(SOL_SOCKET, SO_ERROR)`.

### 116 ESTALE — Stale file handle

NFS file handle no longer valid (file deleted on server, server restarted with different fsid).

```bash
$ cat /mnt/nfs/file
cat: /mnt/nfs/file: Stale file handle
$ ls /mnt/nfs/                # often re-validates after readdir
```

### 121 EREMOTEIO — Remote I/O error

Remote (USB / network) I/O failed; common with USB device disconnect mid-transfer.

### 122 EDQUOT — Disk quota exceeded

User or group quota hit. Different from `ENOSPC` — the filesystem has space but *you* used your share.

```bash
$ dd if=/dev/zero of=~/big bs=1M count=1024
dd: error writing '/home/me/big': Disk quota exceeded
$ quota -v          # see your quotas
$ repquota -a       # admin: all users
```

### 130 EOWNERDEAD — Owner died (robust mutex)

`pthread_mutex_lock` on a `PTHREAD_MUTEX_ROBUST` mutex whose previous owner died holding it. The lock is granted to you in an inconsistent state — call `pthread_mutex_consistent`.

## dmesg / journalctl Patterns

`dmesg` is the kernel ring buffer; `journalctl -k` is the same plus the journal's persistence. `journalctl -p err` shows error-priority and above.

```bash
dmesg -T              # human-readable timestamps (since boot)
dmesg -W              # follow (like tail -f)
dmesg --level=err,warn
journalctl -k          # kernel only
journalctl -k -b      # current boot
journalctl -k -b -1   # previous boot
journalctl -fu sshd   # follow unit
```

### Out-of-Memory kill

```text
[12345.6789] Out of memory: Killed process 1234 (firefox) total-vm:8388608kB, anon-rss:7340032kB, file-rss:0kB, shmem-rss:0kB, UID:1000 pgtables:14336kB oom_score_adj:0
[12345.6790] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/user.slice/user-1000.slice/session-3.scope,task=firefox,pid=1234,uid=1000
```

Older kernel format:

```text
[12345.6] Out of memory: Kill process 1234 (firefox) score 800 or sacrifice child
[12345.6] Killed process 1234 (firefox) total-vm:8388608kB, anon-rss:7340032kB, file-rss:0kB
```

The `score` is `oom_badness()` output (process RSS + swap + adj). The `oom_score_adj` is the user-tunable bias (-1000 to 1000).

### Stack traceback / call stack

```text
[ 4321.0] Stack traceback for pid 1234
[ 4321.0]  ? __schedule+0x1a2/0x600
[ 4321.0]  ? schedule+0x46/0xb0
[ 4321.0]  ? io_schedule+0x16/0x40
[ 4321.0]  ? do_read_cache_page+0x394/0x420
```

### General protection fault

```text
[ 5432.1] general protection fault: 0000 [#1] SMP NOPTI
[ 5432.1] CPU: 2 PID: 1234 Comm: myapp Tainted: G        W   5.15.0 #1
[ 5432.1] RIP: 0010:do_something+0x12/0x40
```

The `[#1]` means "this is the 1st oops since boot." `SMP NOPTI` flags. `Tainted: G W` means kernel was already unhappy.

### NULL pointer dereference

```text
[ 6543.2] BUG: kernel NULL pointer dereference, address: 0000000000000018
[ 6543.2] #PF: supervisor read access in kernel mode
[ 6543.2] #PF: error_code(0x0000) - not-present page
[ 6543.2] PGD 0 P4D 0
[ 6543.2] Oops: 0000 [#1] SMP NOPTI
[ 6543.2] CPU: 0 PID: 567 Comm: kworker/0:1 Tainted: G        W   5.15.0
```

### Soft lockup

```text
[ 7654.3] watchdog: BUG: soft lockup - CPU#3 stuck for 22s! [stress-ng:8901]
[ 7654.3] Modules linked in: tcp_diag inet_diag nf_log_ipv4 ...
```

CPU did not voluntarily yield for 22 seconds (default `kernel.watchdog_thresh=10`, threshold 2x). Often a kernel module bug, or an interrupt storm.

### RCU stall

```text
[ 8765.4] INFO: rcu_sched detected stalls on CPUs/tasks:
[ 8765.4]  3-...: (1 GPs behind) idle=2af/1/0x4000000000000000 softirq=12345/12345 fqs=0
[ 8765.4]  (detected by 0, t=21002 jiffies, g=12345, q=42)
```

RCU grace period stalled — usually a CPU stuck in a tight loop with preemption disabled.

### Disk error

```text
[ 9876.5] ata1.00: exception Emask 0x0 SAct 0x0 SErr 0x0 action 0x0
[ 9876.5] ata1.00: irq_stat 0x40000008
[ 9876.5] ata1.00: failed command: READ FPDMA QUEUED
[ 9876.5] ata1.00: cmd 60/80:00:80:5a:6f/00:00:01:00:00/40 tag 0 ncq dma 65536 in
[ 9876.5]          res 41/40:00:80:5a:6f/00:00:01:00:00/00 Emask 0x409 (media error) <F>
[ 9876.5] ata1.00: status: { DRDY ERR }
[ 9876.5] ata1.00: error: { UNC }
```

`UNC` = Uncorrectable. The drive is failing — `smartctl -a /dev/sda` and replace.

### EXT4 filesystem error

```text
[10987.6] EXT4-fs error (device sda1): ext4_lookup:1602: inode #131073: comm myapp: deleted inode referenced: 9173653
[10987.6] EXT4-fs (sda1): Remounting filesystem read-only
```

Boot to rescue, `e2fsck -fy /dev/sda1`.

### NIC checksum failure

```text
[11098.7] eth0: hw csum failure
```

Hardware offload reported a bad checksum — usually firmware bug; disable with `ethtool -K eth0 rx off tx off`.

### TCP memory pressure

```text
[12109.8] TCP: out of memory -- consider tuning tcp_mem
```

Raise `net.ipv4.tcp_mem`, `net.core.rmem_max`, `net.core.wmem_max`.

### Conntrack table full

```text
[13210.9] nf_conntrack: nf_conntrack: table full, dropping packet
```

Raise `net.netfilter.nf_conntrack_max`, lower `nf_conntrack_tcp_timeout_*`.

### Kernel panic

```text
[14322.0] Kernel panic - not syncing: VFS: Unable to mount root fs on unknown-block(0,0)
[14322.0] CPU: 0 PID: 1 Comm: swapper/0 Not tainted 5.15.0 #1
[14322.0] Call Trace:
[14322.0]  dump_stack_lvl+0x46/0x5e
[14322.0]  panic+0x101/0x2e3
[14322.0]  mount_block_root+0x14a/0x215
```

Cannot continue; reboot. Investigate: missing initrd module, wrong root= cmdline, bad fstab.

## OOM-Killer

The OOM killer fires when memory allocations cannot be satisfied even after reclaim. The dmesg block for an OOM kill has many lines:

```text
[12345.0] myapp invoked oom-killer: gfp_mask=0xcc0(GFP_KERNEL), order=0, oom_score_adj=0
[12345.0] CPU: 2 PID: 1234 Comm: myapp Not tainted 5.15.0 #1
[12345.0] Call Trace:
[12345.0]  dump_stack_lvl+0x46/0x5e
[12345.0]  dump_header+0x4a/0x1ff
[12345.0]  oom_kill_process.cold+0xb/0x10
[12345.0]  out_of_memory+0xed/0x2d0
[12345.0]  __alloc_pages_slowpath.constprop.0+0xc4f/0xd80
[12345.0]  __alloc_pages+0x1ee/0x210
[12345.0] Mem-Info:
[12345.0] active_anon:1024 inactive_anon:1900000 isolated_anon:0
[12345.0]  active_file:32 inactive_file:0 isolated_file:0
[12345.0]  unevictable:0 dirty:0 writeback:0
[12345.0]  slab_reclaimable:5000 slab_unreclaimable:8000
[12345.0]  mapped:1024 shmem:128 pagetables:14336 bounce:0
[12345.0]  kernel_misc_reclaimable:0
[12345.0]  free:1234 free_pcp:0 free_cma:0
[12345.0] Node 0 active_anon:4096kB inactive_anon:7600000kB ...
[12345.0] Tasks state (memory values in pages):
[12345.0] [   pid ]   uid  tgid total_vm      rss pgtables_bytes swapents oom_score_adj name
[12345.0] [    345]     0   345    61500       42       65536        0          -1000 systemd
[12345.0] [   1234]  1000  1234  2097152  1900000      14680064        0              0 myapp
[12345.0] [   2345]  1000  2345    65536     2000        262144        0              0 bash
[12345.0] oom-kill:constraint=CONSTRAINT_NONE,nodemask=(null),cpuset=/,mems_allowed=0,global_oom,task_memcg=/,task=myapp,pid=1234,uid=1000
[12345.0] Out of memory: Killed process 1234 (myapp) total-vm:8388608kB, anon-rss:7600000kB, file-rss:0kB, shmem-rss:512kB, UID:1000 pgtables:14336kB oom_score_adj:0
```

Key lines:

- `gfp_mask=...GFP_KERNEL`, `order=0` — what allocation was attempted.
- `oom_score_adj=0` — adj of the *invoker*, not the victim.
- `Tasks state` table — every candidate ranked. Reading: `rss` is what matters most, `oom_score_adj` is the user bias (-1000 means immune, +1000 means kill first).
- `constraint=CONSTRAINT_NONE` — global OOM. `CONSTRAINT_MEMCG` = container hit cgroup memory.max.

### cgroup-v2 memory OOM

```text
[12346.0] Memory cgroup out of memory: Killed process 1234 (myapp) total-vm:524288kB, anon-rss:511000kB, file-rss:1024kB, shmem-rss:0kB, UID:1000 pgtables:1024kB oom_score_adj:0
[12346.0] memory: usage 524288kB, limit 524288kB, failcnt 1
```

Set / inspect with cgroup-v2:

```bash
cat /sys/fs/cgroup/user.slice/user-1000.slice/session-3.scope/memory.max
cat /sys/fs/cgroup/.../memory.current
cat /sys/fs/cgroup/.../memory.events     # oom, oom_kill counters
```

In systemd: `MemoryMax=512M` in the unit.

### Tuning vm.overcommit_memory

```bash
sysctl vm.overcommit_memory
# 0 = heuristic (default) — kernel guesses if alloc fits
# 1 = always overcommit — malloc never fails, OOM-killer runs
# 2 = strict — alloc fails (ENOMEM) at malloc time, no overcommit

sysctl vm.overcommit_ratio          # 50 — used when overcommit_memory=2
# Total = swap + overcommit_ratio% of RAM

sysctl vm.panic_on_oom              # 0 = oom-kill, 1 = panic, 2 = panic if global
```

### Per-process oom_score_adj

```bash
# View
cat /proc/$$/oom_score_adj         # 0
cat /proc/$$/oom_score             # current computed score

# Set (range -1000 to 1000)
echo -1000 | sudo tee /proc/1234/oom_score_adj   # immune
echo 1000  | sudo tee /proc/1234/oom_score_adj   # kill first

# choom — wrap a command at launch
choom -n -1000 sshd                # protect sshd
choom -p 1234                      # show current
choom -p 1234 -n 500               # adjust running

# systemd
# In unit: OOMScoreAdjust=-1000
# Or globally: DefaultOOMPolicy=stop
```

## Kernel BUG / Oops Patterns

### Oops format

```text
[12345.6] Oops: 0000 [#1] SMP NOPTI
```

- `0000` is the page-fault error code (bit 0 = present, bit 1 = write, bit 2 = user, bit 3 = reserved-bit, bit 4 = inst fetch).
- `[#1]` is "this many oops events since boot." A second `[#2]` means another oops happened — kernel might be too tainted to trust.

### WARNING (non-fatal)

```text
[12346.0] WARNING: CPU: 0 PID: 1 at kernel/sched/core.c:1234 do_thing+0x12/0x40
[12346.0] Modules linked in: nf_log_ipv4 nf_log_common ...
[12346.0] CPU: 0 PID: 1 Comm: myapp Tainted: G        W   5.15.0 #1
```

`WARN_ON()` was triggered — assertion-style check; kernel keeps running but logs it.

### Scheduling while atomic

```text
[12347.0] BUG: scheduling while atomic: ksoftirqd/0/9/0x00000100
```

Code with preemption disabled tried to sleep (called `schedule()` indirectly).

### Sleeping in atomic context

```text
[12348.0] BUG: sleeping function called from invalid context at mm/slab.h:494
[12348.0] in_atomic(): 1, irqs_disabled(): 0, non_block: 0, pid: 9, name: ksoftirqd/0
```

Driver bug — used `kmalloc(GFP_KERNEL)` from interrupt context. Should be `GFP_ATOMIC`.

### Soft / hard lockup

```text
[12349.0] watchdog: BUG: soft lockup - CPU#3 stuck for 22s! [myapp:1234]
[12349.0] Modules linked in: ...
[12349.0] CPU: 3 PID: 1234 Comm: myapp Tainted: G        W   5.15.0 #1
[12349.0] RIP: 0010:my_function+0x42/0x80

[12350.0] NMI watchdog: Watchdog detected hard LOCKUP on cpu 3
```

Soft lockup: scheduler is fine but task didn't yield. Hard lockup: NMI couldn't reach the CPU — completely stuck (often hardware fault).

### Modules linked in

```text
[12351.0] Modules linked in: tcp_diag inet_diag bnep nls_iso8859_1 ...
```

Loaded modules at oops time. If a third-party / out-of-tree module is present, the bug is *probably* there.

### Kernel taint flags

`/proc/sys/kernel/tainted` is a bitmask. Letters in oops headers map to bits:

```bash
$ cat /proc/sys/kernel/tainted
4096
$ for i in $(seq 0 18); do echo "$i: $((4096 >> i & 1))"; done
```

Decoded:

| Char | Bit | Meaning |
|------|-----|---------|
| G/P  | 0   | Proprietary module loaded (P) / all GPL (G) |
| F    | 1   | Module force-loaded |
| S    | 2   | SMP with non-SMP-safe CPU |
| R    | 3   | Module force-unloaded |
| M    | 4   | Machine check exception |
| B    | 5   | Bad page reference / unexpected page flags |
| U    | 6   | Userspace requested taint |
| D    | 7   | Kernel died — OOPS or BUG |
| A    | 8   | ACPI table overridden |
| W    | 9   | WARNING was triggered |
| C    | 10  | staging driver loaded |
| I    | 11  | Workaround for buggy firmware |
| O    | 12  | Out-of-tree module loaded |
| E    | 13  | Unsigned module loaded |
| L    | 14  | Soft lockup |
| K    | 15  | Live-patched kernel |
| X    | 16  | Auxiliary taint (distro-specific) |
| T    | 17  | Built with struct randomization |

`Tainted: G        W   5.15.0` means `G` (all GPL clean) and `W` (WARN fired).

```bash
# Decode the bitmask
cat /proc/sys/kernel/tainted | python3 -c '
import sys
flags = "PFSR MBUDA WCIO ELKXT"
v = int(sys.stdin.read())
for i, c in enumerate(flags.replace(" ","")):
    if v & (1<<i): print(c, end=" ")
print()
'
```

## Signal Exit Codes

When a process is killed by signal `N`, shells report exit code `128+N`. (POSIX says the wait status `WIFSIGNALED && WTERMSIG` is the source of truth; the 128+N convention is bash/dash/zsh shell-level.)

| Exit | Signal       | Name      | Cause                                                 |
|------|--------------|-----------|--------------------------------------------------------|
| 1    | —            | exit(1)   | Generic error (program-defined)                       |
| 2    | —            | exit(2)   | POSIX: misuse of shell builtin / argument error       |
| 126  | —            |           | Command found but not executable (no `+x` or wrong arch) |
| 127  | —            |           | Command not found in `$PATH`                          |
| 128  | —            |           | Invalid arg to `exit` (out of range)                  |
| 129  | 1            | SIGHUP    | Terminal hangup / parent shell exit                   |
| 130  | 2            | SIGINT    | Ctrl+C in terminal                                    |
| 131  | 3            | SIGQUIT   | Ctrl+\ — produces core                                |
| 132  | 4            | SIGILL    | Illegal instruction                                   |
| 133  | 5            | SIGTRAP   | Breakpoint / debug trap                               |
| 134  | 6            | SIGABRT   | `abort()` / `assert()` / glibc malloc corruption     |
| 135  | 7            | SIGBUS    | Misaligned access / mmap'd file truncation            |
| 136  | 8            | SIGFPE    | FP error / integer divide-by-zero                     |
| 137  | 9            | SIGKILL   | `kill -9` or OOM-killer                               |
| 138  | 10           | SIGUSR1   | App-specific (only if app didn't catch)               |
| 139  | 11           | SIGSEGV   | Segfault — bad memory access                          |
| 140  | 12           | SIGUSR2   | App-specific                                          |
| 141  | 13           | SIGPIPE   | Wrote to pipe with no reader                          |
| 142  | 14           | SIGALRM   | `alarm()` timeout fired                               |
| 143  | 15           | SIGTERM   | Polite kill (default `kill PID`)                      |
| 152  | 24           | SIGXCPU   | CPU time limit (`ulimit -t`) exceeded                |
| 153  | 25           | SIGXFSZ   | File-size limit (`ulimit -f`) exceeded                |
| 159  | 31           | SIGSYS    | Bad syscall (also: seccomp killed)                    |
| 255  | —            |           | Out-of-range exit value (wraparound: 256+ → 0..255)   |

```bash
# Quick one-liners
sleep 99 &
kill $!;      wait; echo $?       # 143  (SIGTERM)
sleep 99 &
kill -9 $!;   wait; echo $?       # 137  (SIGKILL)
kill -INT $$; echo $?             # 130  (SIGINT to self)

# Decode with kill -l
kill -l 137                       # KILL
kill -l 143                       # TERM
kill -l                           # full numeric table
```

## SIGKILL vs SIGTERM vs SIGINT — Matrix

| Signal     | # | Default Action  | Catchable? | Source                           |
|------------|---|-----------------|-----------|----------------------------------|
| SIGHUP     | 1 | Terminate       | Yes       | terminal hangup / `kill -HUP`    |
| SIGINT     | 2 | Terminate       | Yes       | Ctrl+C                           |
| SIGQUIT    | 3 | Core            | Yes       | Ctrl+\                            |
| SIGILL     | 4 | Core            | Yes       | bad instruction (CPU)            |
| SIGTRAP    | 5 | Core            | Yes       | breakpoint                       |
| SIGABRT    | 6 | Core            | Yes       | `abort()` / `assert()`           |
| SIGBUS     | 7 | Core            | Yes       | bus error                        |
| SIGFPE     | 8 | Core            | Yes       | divide by zero                   |
| SIGKILL    | 9 | Terminate       | **No**    | `kill -9`, OOM-killer            |
| SIGUSR1    | 10| Terminate       | Yes       | app-specific                     |
| SIGSEGV    | 11| Core            | Yes       | segfault                         |
| SIGUSR2    | 12| Terminate       | Yes       | app-specific                     |
| SIGPIPE    | 13| Terminate       | Yes       | write to closed pipe             |
| SIGALRM    | 14| Terminate       | Yes       | `alarm()`                         |
| SIGTERM    | 15| Terminate       | Yes       | `kill` default                    |
| SIGCHLD    | 17| Ignore          | Yes       | child exited (sent to parent)    |
| SIGCONT    | 18| Continue        | Yes (mostly) | `kill -CONT`                  |
| SIGSTOP    | 19| Stop            | **No**    | `kill -STOP`                     |
| SIGTSTP    | 20| Stop            | Yes       | Ctrl+Z                            |
| SIGTTIN    | 21| Stop            | Yes       | bg read on tty                    |
| SIGTTOU    | 22| Stop            | Yes       | bg write on tty                   |
| SIGURG     | 23| Ignore          | Yes       | OOB data on socket                |
| SIGXCPU    | 24| Core            | Yes       | CPU limit                         |
| SIGXFSZ    | 25| Core            | Yes       | file size limit                   |
| SIGSYS     | 31| Core            | Yes       | bad syscall / seccomp             |

```bash
# Send signals
kill PID                  # SIGTERM (15)
kill -9 PID               # SIGKILL
kill -HUP $(pidof nginx)  # graceful reload (nginx convention)
kill -USR1 $(pidof rsyslogd)  # rsyslog rotates logs
killall -9 firefox

# Catch in C
#include <signal.h>
signal(SIGTERM, my_handler);
sigaction(SIGTERM, &(struct sigaction){.sa_handler=my_handler}, NULL);

# Catch in bash
trap 'echo got SIGTERM; cleanup; exit 0' TERM
trap 'cleanup' EXIT INT TERM
```

```bash
# Why SIGKILL won't work (rare):
# - process is in uninterruptible sleep (D state) waiting on a stuck driver
# - process is a zombie (already dead, just unreaped)
# - process is being ptraced
ps -eo pid,stat,comm | awk '$2 ~ /D/'       # find D-state
```

## SIGSEGV Diagnosis

A segfault means a memory access violated the page table permissions. Causes: NULL deref, dangling pointer, stack overflow, executing data, writing read-only.

```bash
$ ./buggy
Segmentation fault (core dumped)
$ echo $?
139
```

### Core dump location

```bash
# Where do cores go?
sysctl kernel.core_pattern
# Modern systemd: |/usr/lib/systemd/systemd-coredump %P %u %g %s %t %c %h
# Older: core   (CWD-relative) or /tmp/core.%e.%p

cat /proc/sys/kernel/core_pattern
```

`%`-tokens: `%p` pid, `%u` uid, `%g` gid, `%s` signal, `%t` time, `%h` hostname, `%e` exe-name, `%E` exe-path (with / →  !), `%c` core-size limit.

### Enabling core dumps

```bash
# Per-shell
ulimit -c unlimited       # 0 by default → no core
ulimit -c                 # check

# Persistent (limits.conf)
echo '*  soft  core  unlimited' | sudo tee -a /etc/security/limits.conf
echo '*  hard  core  unlimited' | sudo tee -a /etc/security/limits.conf

# systemd unit
# [Service]
# LimitCORE=infinity
```

### Reading cores

```bash
# systemd-coredump
coredumpctl list                         # all cores
coredumpctl info myapp                   # show last
coredumpctl gdb myapp                    # open gdb on it
coredumpctl dump myapp -o core.myapp     # extract to file

# Manual
gdb /usr/bin/myapp /var/lib/systemd/coredump/core.myapp.1000.abc.123.456.lz4
(gdb) bt                                 # backtrace
(gdb) bt full                            # with locals
(gdb) info reg                           # registers
(gdb) frame 3                            # jump to frame
(gdb) print *somevar
(gdb) thread apply all bt                # all threads
```

### Symbolicate stack traces

If the program logs a stack of addresses (e.g. `glibc` backtrace from `backtrace_symbols`), use `addr2line`:

```bash
# Need binary built with -g (debug info)
addr2line -e /usr/bin/myapp -f -C -i 0x402a4f
# main
# /home/me/src/main.c:42

# For PIE binaries, subtract load address from /proc/PID/maps
```

### Common SIGSEGV signatures

```text
"Segmentation fault (core dumped)"            # generic
"address 0x0"  / "near (nil)"                 # NULL deref
"stack overflow detected"                     # gcc -fstack-protector caught it
"buffer overflow detected"                    # _FORTIFY_SOURCE caught a strcpy
"double free or corruption (out)"             # glibc malloc detected
"free(): invalid next size"                   # heap corruption
"munmap_chunk(): invalid pointer"             # heap corruption
```

`_FORTIFY_SOURCE` and stack-protector trigger via `SIGABRT` (134), not SIGSEGV (139), but the message matters.

### NULL pointer vs stack overflow

```bash
# In gdb after crash:
(gdb) info reg rsp
# RSP near 0x7fffXXXXXXXX = normal
# RSP at top of stack region (e.g. 0x7ffff7ffXXXX with no room) = stack overflow
(gdb) info proc mappings | grep stack
# 0x7ffffffde000  0x7ffffffff000  0x21000  rw-p  [stack]
```

## SIGBUS

SIGBUS (signal 7, exit 135) means the access *was* permitted by page tables but the underlying I/O failed. Causes:

- **Misaligned access on architectures that care** — ARM (pre-v6) / SPARC trap on unaligned 32-bit loads. x86 silently fixes most.
- **mmap'd file truncated** — process mmaps a file, then someone `truncate`s it shorter than the mapping. Accessing beyond EOF → SIGBUS.
- **mmap of a sparse file's hole that fails to allocate** — disk full while writing through mmap.
- **Hardware memory error** — MCE on bad RAM (rare, with ECC the kernel may recover).

```c
// Reproducing mmap truncation SIGBUS:
int fd = open("/tmp/f", O_RDWR | O_CREAT, 0644);
ftruncate(fd, 4096);
char *p = mmap(NULL, 4096, PROT_READ|PROT_WRITE, MAP_SHARED, fd, 0);
ftruncate(fd, 0);       // shrink it
p[0] = 'x';             // SIGBUS!
```

Fix: never truncate behind a live mmap; use `MAP_SHARED_VALIDATE | MAP_SYNC` for DAX; preallocate with `fallocate`.

## ulimit / Resource Limits

`ulimit` is a shell builtin wrapping `setrlimit(2)`. Each process has soft (current) and hard (cap) limits per resource.

```bash
ulimit -a                # all current limits
ulimit -aS               # soft
ulimit -aH               # hard
```

| Flag | Resource         | rlimit          | Failure mode                                             |
|------|------------------|-----------------|----------------------------------------------------------|
| -c   | core size (KB)   | RLIMIT_CORE     | no core file when crashed (silent)                       |
| -d   | data seg (KB)    | RLIMIT_DATA     | malloc → ENOMEM                                           |
| -e   | nice prio        | RLIMIT_NICE     | EPERM on `nice -n -10`                                    |
| -f   | file size (KB)   | RLIMIT_FSIZE    | EFBIG → SIGXFSZ if soft hit                               |
| -i   | pending signals  | RLIMIT_SIGPENDING| EAGAIN on `kill`                                         |
| -l   | locked mem (KB)  | RLIMIT_MEMLOCK  | EPERM on `mlock`                                          |
| -m   | RSS (KB) — IGNORED on Linux since 2.4.30 | RLIMIT_RSS | (no effect)                              |
| -n   | open files       | RLIMIT_NOFILE   | EMFILE on open/accept/socket                              |
| -p   | pipe size (512B) | RLIMIT_PIPESIZE | (Linux: read-only, returns 8 = 4096 bytes)               |
| -q   | POSIX MQ bytes   | RLIMIT_MSGQUEUE | EMFILE on mq_open                                         |
| -r   | RT prio          | RLIMIT_RTPRIO   | EPERM on chrt                                              |
| -s   | stack (KB)       | RLIMIT_STACK    | SIGSEGV at top of stack                                    |
| -t   | CPU time (sec)   | RLIMIT_CPU      | SIGXCPU (signal 24, exit 152)                              |
| -u   | max procs/user   | RLIMIT_NPROC    | EAGAIN on fork — "Resource temporarily unavailable"        |
| -v   | virtual mem (KB) | RLIMIT_AS       | mmap/malloc → ENOMEM                                        |
| -x   | file locks       | RLIMIT_LOCKS    | EAGAIN on flock                                            |

### Common limits.conf

```text
# /etc/security/limits.conf — for PAM-based logins
# <domain>   <type>    <item>    <value>
*            soft      nofile    65536
*            hard      nofile    1048576
*            soft      nproc     unlimited
*            hard      nproc     unlimited
@developers  soft      core      unlimited
root         hard      nofile    1048576
```

`/etc/security/limits.d/*.conf` — drop-in directory.

For `pam_limits.so` to apply: ensure `session  required  pam_limits.so` in `/etc/pam.d/login`, `/etc/pam.d/sshd`, etc.

### systemd LimitX

```ini
[Service]
LimitCORE=infinity
LimitNOFILE=1048576
LimitNPROC=65536
LimitSTACK=8M
LimitCPU=3600        # seconds
TasksMax=4915        # cgroup pids.max
MemoryMax=512M       # cgroup memory.max
```

```bash
systemctl show foo.service | grep -i limit
prlimit --pid 1234         # see limits of running process
prlimit --pid 1234 --nofile=65536:65536    # adjust live
```

## /proc Common Investigations

```bash
# Per-process
ls /proc/PID/

cat /proc/PID/status         # human-readable: VmRSS, VmSize, Threads, FDSize, State
cat /proc/PID/stat           # space-separated: ps reads this
cat /proc/PID/maps           # virtual address space (heap/stack/mmaps)
cat /proc/PID/smaps          # detailed per-mapping (Pss, Rss, Swap)
cat /proc/PID/cmdline | tr '\0' ' '   # full argv
cat /proc/PID/environ | tr '\0' '\n'  # env (root or own pid)
cat /proc/PID/limits         # rlimits
ls -la /proc/PID/fd/         # open fds: targets show files/sockets
ls -la /proc/PID/cwd         # CWD as symlink
ls -la /proc/PID/exe         # binary as symlink
cat /proc/PID/stack          # in-kernel stack (CONFIG_STACKTRACE)
cat /proc/PID/wchan          # what kernel function is the process sleeping in
cat /proc/PID/io             # bytes read/written
cat /proc/PID/sched          # detailed scheduler stats
cat /proc/PID/syscall        # current syscall # + args (if blocked)
ls /proc/PID/task/           # threads (each has same /proc/PID/task/TID/)
```

```bash
# Decode VmRSS lines
$ grep ^Vm /proc/$$/status
VmPeak:    11116 kB     # peak VmSize
VmSize:    11116 kB     # virtual address space
VmLck:         0 kB     # locked (mlock)
VmPin:         0 kB     # pinned
VmHWM:      4304 kB     # peak RSS ("high water mark")
VmRSS:      4304 kB     # resident set
VmData:     1612 kB     # data + heap
VmStk:       136 kB     # stack
VmExe:       968 kB     # text segment
VmLib:      2604 kB     # libraries
VmPTE:        56 kB     # page table entries
VmSwap:        0 kB     # swapped out
```

```bash
# System-wide /proc/sys
sysctl -a | less                       # all keys
sysctl vm.swappiness                   # 60 default
sysctl vm.dirty_ratio                  # % of RAM dirty before write blocks
sysctl fs.file-max
sysctl fs.nr_open
sysctl kernel.pid_max
sysctl kernel.core_pattern
sysctl kernel.printk                   # console log level
sysctl net.ipv4.tcp_fin_timeout
sysctl net.ipv4.ip_local_port_range
sysctl net.core.somaxconn
sysctl net.ipv4.tcp_max_syn_backlog
sysctl net.netfilter.nf_conntrack_max

# Persist
echo 'net.core.somaxconn = 4096' | sudo tee /etc/sysctl.d/99-tune.conf
sudo sysctl --system
```

```bash
# /proc/meminfo — system memory
cat /proc/meminfo
# MemTotal:        16384000 kB
# MemFree:           500000 kB
# MemAvailable:     8000000 kB    # USE THIS, not MemFree
# Buffers:           300000 kB
# Cached:           7000000 kB
# SwapCached:             0 kB
# Active:           4000000 kB
# Inactive:         3000000 kB
# Dirty:               5000 kB
# Writeback:              0 kB
# Slab:              500000 kB
# SReclaimable:      300000 kB
# SUnreclaim:        200000 kB
```

## systemd Unit Failures

```bash
systemctl status foo.service     # current state + last 10 lines
journalctl -u foo.service        # all logs for this unit
journalctl -fu foo.service       # follow
journalctl -u foo --since '1 hour ago'
journalctl -u foo -p err         # error and worse
journalctl _PID=1234             # by pid
journalctl _COMM=nginx           # by comm
journalctl -b                    # current boot
journalctl -b -1                 # previous
journalctl --list-boots
journalctl -k                    # kernel only
journalctl -p err -b             # errors this boot
```

### Common status lines (verbatim)

```text
● foo.service - My App
     Loaded: loaded (/etc/systemd/system/foo.service; enabled; vendor preset: enabled)
     Active: failed (Result: exit-code) since Mon 2024-01-15 12:34:56 UTC; 2min ago
   Main PID: 1234 (code=exited, status=1/FAILURE)
        CPU: 12ms
```

```text
Active: failed (Result: signal) since ... — main exited via signal
Main PID: 1234 (code=killed, signal=KILL)

Active: failed (Result: oom-kill)            — cgroup memory.max killed it
Active: failed (Result: timeout)              — startup/stop took too long
Active: failed (Result: protocol)             — Type=notify or Type=dbus failed handshake
Active: failed (Result: resources)            — fork failed / EAGAIN
Active: failed (Result: core-dump)            — exited via SIGSEGV/SIGABRT
Active: failed (Result: start-limit-hit)      — restarted too many times
```

```text
Service hold-off time over, scheduling restart.
Stopped Foo Service.
Start request repeated too quickly.
Failed to start Foo Service.
foo.service: Failed with result 'exit-code'.
foo.service: Failed with result 'oom-kill'.
foo.service: Main process exited, code=killed, status=9/KILL
```

### Restart loops

```text
foo.service: Scheduled restart job, restart counter is at 5.
foo.service: Start request repeated too quickly.
foo.service: Failed with result 'start-limit-hit'.
```

Tune in unit:

```ini
[Service]
Restart=on-failure
RestartSec=5s
StartLimitBurst=5
StartLimitIntervalSec=300

[Unit]
StartLimitBurst=5         # alternative location (older systemd)
StartLimitIntervalSec=300
```

```bash
systemctl reset-failed foo.service     # clear the rate-limit bookkeeping
systemctl daemon-reload                 # after editing units
systemctl edit foo.service              # drop-in /etc/systemd/system/foo.service.d/override.conf
```

### Common fixes

```bash
# unit fails immediately
journalctl -u foo --since today | tail -50

# binary not found?
systemd-analyze verify foo.service

# permissions
systemctl cat foo.service          # see effective unit
ls -la $(systemctl show foo -p ExecStart --value | awk '{print $1}')
```

## systemd-resolved / NSS Errors

```bash
resolvectl status                  # show DNS servers per link
resolvectl query example.com
resolvectl flush-caches
resolvectl statistics

# Older
systemd-resolve --status
systemd-resolve example.com
```

```text
example.com: resolve call failed: 'example.com' not found
example.com: resolve call failed: All attempts to contact name servers or networks failed
```

```bash
# /etc/nsswitch.conf — order of name resolvers
hosts:          files mdns4_minimal [NOTFOUND=return] dns
# files = /etc/hosts
# mdns4_minimal = avahi/mDNS for .local
# dns = system DNS (resolved or libc resolver)
```

```bash
# Debug
getent hosts example.com           # uses NSS chain
host example.com                   # direct DNS only (no /etc/hosts)
dig +short example.com
nslookup example.com

# Override via /etc/hosts
echo '192.0.2.1  example.com' | sudo tee -a /etc/hosts
```

## Filesystem Errors

### Read-only file system

```bash
$ touch /var/foo
touch: cannot touch '/var/foo': Read-only file system

# Why?
mount | grep ' / '          # see ro flag?
dmesg | grep -i 'remount'    # was it kernel-forced?
dmesg | grep -i 'EXT4-fs error'
```

If it's a real fs error, **don't** blindly remount-rw — boot to rescue and run `e2fsck -fy /dev/sdaN`. If it's intentional (e.g. `/usr/` on a server), you need to remount:

```bash
mount -o remount,rw /
mount -o remount,ro /
```

### No space left

```bash
df -h /var                  # by bytes
df -i /var                  # by inodes
du -sh /var/* | sort -h     # who's using space
find /var -size +100M -type f -exec ls -lh {} +
ncdu /                      # interactive

# Inode exhaustion fix:
# - delete many small files (logs, cache)
# - or recreate the fs with -N inodes (for ext4, mkfs.ext4 -N 50000000)
```

Common offenders:

- `/var/log/journal/` — `journalctl --vacuum-size=200M` or `--vacuum-time=2weeks`
- `/var/cache/apt/archives/` — `apt clean`
- `/tmp/` — clean it
- `/var/lib/docker/` — `docker system prune -af`
- core files in `/var/lib/systemd/coredump/`
- a deleted-but-open file (`lsof | grep deleted`) — restart the holder

### Disk quota exceeded

```bash
$ dd if=/dev/zero of=~/big bs=1M count=1000
dd: error writing '/home/me/big': Disk quota exceeded

quota -v                    # your quota
quota -u alice              # someone's quota (root)
repquota -a                 # all quotas (root)
edquota alice               # edit
quotacheck -avum            # rebuild
quotaon -av                 # enable on all fs
```

### Stale NFS file handle

```bash
$ cat /mnt/nfs/file
cat: /mnt/nfs/file: Stale file handle

# Server replaced the file or changed fsid (NFSv3 export). Re-readdir often heals:
ls /mnt/nfs/                # fresh dirent reads new handle

# Or remount
sudo umount /mnt/nfs
sudo mount /mnt/nfs
```

### fsck output

```text
e2fsck 1.46.5 (30-Dec-2021)
/dev/sda1: clean, 12345/61054976 files, 9876543/244189184 blocks

# When dirty:
/dev/sda1: recovering journal
Pass 1: Checking inodes, blocks, and sizes
Pass 2: Checking directory structure
Pass 3: Checking directory connectivity
Pass 4: Checking reference counts
Pass 5: Checking group summary information

Block bitmap differences: -1234 -1235
Fix<y>?
```

```bash
e2fsck -fy /dev/sda1        # force, answer yes
e2fsck -nv /dev/sda1        # dry run
e2fsck -p /dev/sda1         # only auto-fixable
xfs_repair /dev/sda1         # XFS equivalent
btrfs check --repair /dev/sda1   # use sparingly; --readonly first
```

## Network Errors

```text
ping: lookup failure                            # DNS
ping: Destination Host Unreachable              # ARP / no route
ping: Network is unreachable                    # no default route
From 192.168.1.1 icmp_seq=1 Destination Host Unreachable
From X icmp_seq=1 Time to live exceeded         # routing loop
```

### Diagnostic ladder

```bash
# 1. DNS works?
getent hosts example.com
dig +short example.com

# 2. ARP for default gw?
ip route show default
ip neigh show           # show ARP cache; STALE / REACHABLE / FAILED

# 3. Routing for the destination?
ip route get 1.1.1.1
ip route show

# 4. Reach gw at L3?
ping -c 2 $(ip route show default | awk '/default/{print $3}')

# 5. ICMP to remote
ping -c 4 -W 2 example.com

# 6. TCP to specific port?
nc -zv example.com 443
ss -tan dst :443        # all conns to port 443

# 7. Path with continuous hops
mtr -rwzbc 30 example.com
```

### arping vs ping

`ping` uses ICMP (works only if ICMP is allowed). `arping` uses ARP — only works on the same L2 segment. Useful to check if a host *exists* on the LAN even when it firewalls ICMP.

```bash
arping -c 3 192.168.1.10           # send ARP to LAN host
arping -I eth0 -D 192.168.1.10     # duplicate-address detection
```

## iptables / nftables Common Issues

```bash
# Show counters per rule (legacy iptables)
iptables -L -v -n
iptables -t nat -L -v -n
iptables -L FORWARD -v -n --line-numbers

# nftables
nft list ruleset
nft list table inet filter
nft list chain inet filter input -a    # with handles

# Watch counters live
watch -n1 'iptables -L INPUT -v -n'
```

### Conntrack table full

```text
nf_conntrack: nf_conntrack: table full, dropping packet
```

```bash
sysctl net.netfilter.nf_conntrack_max
sysctl net.netfilter.nf_conntrack_count
sysctl net.netfilter.nf_conntrack_buckets

# Raise table size (max should be 4x buckets typically)
sysctl -w net.netfilter.nf_conntrack_max=524288

# Lower idle TIME_WAIT timeouts
sysctl -w net.netfilter.nf_conntrack_tcp_timeout_time_wait=30
sysctl -w net.netfilter.nf_conntrack_tcp_timeout_established=86400

# Inspect
conntrack -L                       # all entries
conntrack -L -p tcp --dport 80
conntrack -S                       # stats
conntrack -F                       # flush
```

### Dropping unexplained traffic — bisect

```bash
# Insert LOG rules to see what hits where (legacy)
iptables -I INPUT -j LOG --log-prefix "INPUT: "
iptables -I FORWARD -j LOG --log-prefix "FWD: "
dmesg -W                           # watch
# Remove when done!
iptables -D INPUT -j LOG --log-prefix "INPUT: "
```

For nftables:

```bash
nft add rule inet filter input log prefix \"INPUT: \" level info
```

## SELinux / AppArmor Denials

### SELinux

```bash
getenforce                        # Enforcing | Permissive | Disabled
setenforce 0                      # to permissive (transient)
setenforce 1                      # back to enforcing
# Persistent: edit /etc/selinux/config
```

Denials show up as AVC entries in audit log:

```text
type=AVC msg=audit(1700000000.123:456): avc: denied { read } for pid=1234 comm="myapp" name="passwd" dev="dm-0" ino=12345 scontext=system_u:system_r:httpd_t:s0 tcontext=system_u:object_r:shadow_t:s0 tclass=file permissive=0
```

Decoded:

- `denied { read }` — what was blocked
- `pid=1234 comm="myapp"` — who tried
- `scontext` — the source context (what the process is)
- `tcontext` — the target context (what the file is)
- `tclass=file` — type of object
- `permissive=0` — was permissive mode? 0 means it was actually blocked

```bash
# Find recent denials
ausearch -m AVC -ts recent
ausearch -m AVC,USER_AVC -ts today
journalctl -t setroubleshoot

# Get human-readable advice
sealert -a /var/log/audit/audit.log

# Generate a policy for the denial
ausearch -m AVC -ts recent | audit2allow -M mymodule
semodule -i mymodule.pp

# Quick toggles
setsebool -P httpd_can_network_connect 1
restorecon -Rv /var/www         # reset contexts
chcon -t httpd_sys_content_t /var/www/file
ls -lZ /var/www                 # show contexts
ps -eZ | grep myapp             # process context
```

### AppArmor (Ubuntu / Debian / SUSE)

```bash
aa-status                       # profiles + counts
aa-complain /etc/apparmor.d/usr.bin.myapp     # to permissive
aa-enforce /etc/apparmor.d/usr.bin.myapp      # to enforcing
journalctl | grep -i 'apparmor.*DENIED'
```

Denial format:

```text
audit: type=1400 audit(1700000000.000:1234): apparmor="DENIED" operation="open" profile="myapp" name="/etc/passwd" pid=1234 comm="myapp" requested_mask="r" denied_mask="r" fsuid=1000 ouid=0
```

## Memory Diagnostics

```bash
# Top-level
free -h
#                total        used        free      shared  buff/cache   available
# Mem:           15Gi        4.0Gi       1.0Gi       300Mi        10Gi        10Gi
# Swap:          2.0Gi       100Mi       1.9Gi

# RIGHT NUMBER: "available" — what apps could allocate without swapping (Linux 3.14+)
# WRONG: "free" — buff/cache is reclaimable; "free" omits this

vmstat 1                       # free, buff, cache, swap, io, system, cpu (run forever)
sar -r 1 5                     # historical memory if sysstat collecting
```

```bash
# Per-process memory
ps aux --sort=-rss | head -20
top -o RES                       # sort by RSS
htop                             # F6 → MEM%
smem -tk -s rss                  # PSS-aware (better for shared mem)
pmap -x 1234                     # detailed map of pid

# Detailed
cat /proc/1234/smaps_rollup      # Pss/Rss/Swap totals
cat /proc/meminfo
```

```bash
# Find leaks
valgrind --leak-check=full --show-leak-kinds=all ./buggy
heaptrack ./buggy && heaptrack_gui heaptrack.buggy.123.gz

# Check for swap thrashing
vmstat 1                          # high si/so columns
sar -W 1 5                        # pswpin/pswpout
```

```bash
# OOM relationship
sysctl vm.overcommit_memory       # 0/1/2 (see OOM section)
sysctl vm.swappiness              # 60 default (0 = avoid swap, 100 = aggressive)
sysctl vm.vfs_cache_pressure      # 100 default
sysctl vm.min_free_kbytes         # safety reserve
```

## CPU Diagnostics

```bash
# Live
top                              # interactive
htop                             # nicer top
atop                             # historical + live
btop                             # eye-candy

# Per-CPU breakdown
mpstat -P ALL 1                  # %user/%sys/%idle/%iowait per CPU
sar -u 1 5                       # historical / spot CPU

# Per-process
pidstat 1                        # %CPU per process
pidstat -p 1234 1                # one process
pidstat -t 1                     # threads

# Load average
uptime
# load average: 0.50, 1.20, 0.80   (1min, 5min, 15min)
# scaled to total CPUs: load 8.0 on 8-CPU = 100% saturated

cat /proc/loadavg
# 0.50 1.20 0.80 1/1234 56789
# 1m 5m 15m running/total last_pid

nproc                            # # of CPUs
```

```bash
# vmstat fields
vmstat 1
# procs ----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
#  r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
#  1  0      0  500000 300000 7000000   0    0     5    10   234  456 12  3 84  1  0

# r = runnable (running + ready); >> nproc means CPU starved
# b = uninterruptible sleep (D-state); high = I/O bound
# wa = % CPU time in I/O wait
# st = % CPU stolen by hypervisor
# us+sy = % CPU work
```

```bash
# CPU frequency / governor
cpupower frequency-info
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
# powersave / performance / ondemand

# Per-core utilisation history
sar -P ALL 1 5
```

```bash
# Profile a hot process
perf top -p 1234
perf record -F 99 -p 1234 -g -- sleep 30
perf report
perf script | flamegraph.pl > flame.svg

# strace for syscall-bound
strace -c -p 1234 &        # summarise syscalls; let run, then SIGINT
strace -fttT -p 1234 -o trace.log
```

## I/O Diagnostics

```bash
# Live
iotop                            # per-process I/O (root)
iotop -oP                        # only active, processes (not threads)
pidstat -d 1                     # per-process disk

# Per-device
iostat -xz 1                     # extended; skip idle
sar -d 1 5                       # historical

# iostat columns:
# r/s  w/s  rkB/s  wkB/s  rrqm/s  wrqm/s  %rrqm  %wrqm  r_await  w_await  aqu-sz  rareq-sz  wareq-sz  svctm  %util
# r_await/w_await ms — total time per request including queue
# aqu-sz — average queue length
# %util — % time device had outstanding requests (NOT saturation; for SSDs >100% normal)
```

```bash
# Whose I/O is it?
sudo iotop -oa                    # accumulated
sudo strace -e trace=read,write,pread64,pwrite64 -p 1234

# dstat — unified
dstat -tcdngyrm 1                # time/cpu/disk/net/page/sys/io/mem
dstat --top-io --top-cpu --top-mem
```

```bash
# Block layer detail
blktrace -d /dev/sda -o trace
blkparse trace.blktrace.0
btt -i trace                     # summary

# fio for synthetic
fio --name=test --filename=/tmp/f --rw=randread --bs=4k --size=1G --runtime=30 --iodepth=32
```

## Network Diagnostics

```bash
# Listeners + connections
ss -tlnp                         # TCP listeners + program (-p needs root)
ss -tan                          # all TCP
ss -tan state established
ss -tan '( sport = :80 or dport = :80 )'
ss -tunap                        # tcp+udp+listen+all+process

# netstat (deprecated, slower)
netstat -tlnp
netstat -tunap
netstat -i                       # per-interface stats
netstat -s                       # protocol-level stats

# ss is faster (parses /proc/net/tcp directly via netlink)
```

```bash
# Packet capture
tcpdump -i eth0 -nn -vv 'tcp port 80'
tcpdump -i any -w /tmp/cap.pcap -s 0
tcpdump -r /tmp/cap.pcap -A      # show ASCII payload

tshark -i eth0 -Y 'http.request'
tshark -r /tmp/cap.pcap -z conv,tcp     # conversation list

# need root or CAP_NET_RAW
```

```bash
# Continuous hops
mtr -rwzbc 30 example.com        # ASCII report; -r=report -w=wide -z=ASN -b=both ip+name -c=count

# Throughput live
iftop -i eth0                    # by connection
nethogs eth0                     # by process
nload eth0                       # by interface
bmon -p eth0
sar -n DEV 1 5                   # historical
```

```bash
# Latency & loss between two hosts
hping3 -S -p 443 -c 100 example.com    # SYN; latency from 3WHS
nping --tcp --dest-port 443 -c 100 example.com

# DNS detail
dig +trace example.com
dig @1.1.1.1 example.com AAAA
dog example.com                  # nice cli

# Path MTU
ping -M do -s 1472 example.com   # IPv4: 1500 - 28 = 1472
tracepath example.com            # discovers path MTU per hop
```

## Process Diagnostics

```bash
# Tree
ps auxf
ps -ejH                          # session/jobs hierarchy
pstree -palT
pstree -p 1                      # systemd's tree

# Find/kill
pgrep -af nginx                  # full match incl args
pgrep -u alice
pkill -HUP nginx
pkill -f 'python myapp'
killall myapp                    # exact name match

# Open files / sockets
lsof -p 1234                     # all fds for pid
lsof +f -- /var/log              # what's holding files in /var/log
lsof -nP -iTCP:80                # whose listening on tcp/80
lsof -i :22                      # all on port 22
lsof | grep deleted              # phantom open files (deleted but held)

fuser /var/log/syslog            # who has this file open
fuser -mv /home                   # any process using /home (mountpoint)
fuser -k /var/lock/foo            # kill them all (CAREFUL)

# Per-process detail
ps -o pid,ppid,user,nice,rss,vsz,stat,cmd -p 1234

# State letters in STAT:
# R running
# S sleeping (interruptible)
# D uninterruptible sleep (usu I/O); cannot be killed by SIGKILL until I/O completes
# T stopped (Ctrl+Z)
# Z zombie (dead, not yet reaped)
# I idle kernel thread
# Plus suffixes:
# < high priority
# N low priority
# L locked pages in mem
# s session leader
# l multi-threaded
# + foreground process group
```

## Common Errors (verbatim) — non-errno

```text
bash: command not found
sudo: command not found
$ ./script
bash: ./script: No such file or directory     # often: missing interpreter (#!)
bash: ./script: cannot execute: required file not found

bash: ./script: Permission denied
bash: ./script: cannot execute binary file: Exec format error
bash: cannot create temp file for here-document: No space left on device

cp: cannot stat 'src': No such file or directory
cp: omitting directory 'src'                          # need -r/-R
cp: cannot create regular file 'dst': Permission denied

mv: cannot move 'a' to 'b': Device or resource busy
mv: cannot move 'a' to 'b': Invalid cross-device link
rm: cannot remove 'foo': Is a directory               # need -r
rm: cannot remove 'foo': No such file or directory

chmod: cannot access 'x': No such file or directory
chown: changing ownership of 'x': Operation not permitted

mkdir: cannot create directory 'x': File exists
mkdir: cannot create directory 'x': No space left on device
mkdir: cannot create directory 'x': Permission denied
ln: failed to create symbolic link 'x': File exists

tar: ./foo: file changed as we read it
tar: Removing leading '/' from member names
tar: Cowardly refusing to create an empty archive
tar: This does not look like a tar archive
gzip: stdin: not in gzip format
gzip: stdin: invalid compressed data--format violated

ssh: connect to host X port 22: Connection refused
ssh: connect to host X port 22: Connection timed out
ssh: connect to host X port 22: No route to host
ssh: Could not resolve hostname X: Name or service not known
Permission denied (publickey).
Permission denied (publickey,password).
Connection closed by X port 22
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
The fingerprint for the ECDSA key sent by the remote host is ...
Host key verification failed.

bash: fork: retry: Resource temporarily unavailable
bash: fork: Cannot allocate memory
bash: fork: Operation not permitted

curl: (6) Could not resolve host: example.com
curl: (7) Failed to connect to example.com port 443: Connection refused
curl: (28) Connection timed out after 30001 milliseconds
curl: (35) OpenSSL SSL_connect: SSL_ERROR_SYSCALL in connection to ...
curl: (51) SSL: no alternative certificate subject name matches target host
curl: (52) Empty reply from server
curl: (56) Recv failure: Connection reset by peer
curl: (60) SSL certificate problem: self signed certificate
curl: (60) SSL: no alternative certificate subject name matches target host
curl: (77) Problem with the SSL CA cert (path? access rights?)

apt: Unable to acquire the dbus lock /var/lib/dpkg/lock-frontend
dpkg: error: dpkg frontend lock is locked by another process
E: Could not get lock /var/lib/dpkg/lock - open (11: Resource temporarily unavailable)
E: Unable to fetch some archives, maybe run apt-get update or try with --fix-missing?

systemctl: command not found                         # using SysV?
Failed to connect to bus: No such file or directory   # systemctl in container w/o systemd

dnf: Failed to synchronize cache for repo 'X'
yum: Cannot find a valid baseurl for repo

docker: Got permission denied while trying to connect to the Docker daemon socket
docker: Error response from daemon: pull access denied for X
docker: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?

mount: special device /dev/sdb1 does not exist
mount: wrong fs type, bad option, bad superblock on /dev/sdb1
mount: only root can do that
mount: /mnt/x: target is busy.

fdisk: cannot open /dev/sda: Permission denied
mkfs.ext4: Device or resource busy

git: 'foo' is not a git command. See 'git --help'.
fatal: not a git repository (or any of the parent directories): .git
fatal: refusing to merge unrelated histories
error: failed to push some refs to 'X'
error: pathspec 'X' did not match any file(s) known to git
hint: Updates were rejected because the tip of your current branch is behind
```

## Common Gotchas

### 1. `rm -rf` with shell expansion that includes `-`

```bash
# BROKEN
$ ls
-rf  important.txt
$ rm *               # expands to: rm -rf important.txt — disaster

# FIXED
$ rm -- *            # -- ends option parsing
$ rm ./*             # paths starting with ./ aren't taken as flags
```

### 2. `cp -r` trailing-slash semantics

```bash
# BROKEN — creates /dst/src
$ cp -r src /dst/

# FIXED — copies *contents* of src into /dst
$ cp -r src/. /dst/
$ rsync -a src/ /dst/      # rsync's trailing slash is the standard
```

### 3. `chown user file` but parent dir not readable

```bash
# BROKEN
$ chown alice /home/bob/private/file
chown: cannot access '/home/bob/private/file': Permission denied
# (unable to traverse /home/bob/private/)

# FIXED — get +x on every parent component
$ namei -m /home/bob/private/file        # show every component's mode
```

### 4. `sudo cmd > file` — redirect runs as user

```bash
# BROKEN — root runs cmd, but the shell does the > as YOU before cmd starts
$ sudo echo data > /etc/protected
bash: /etc/protected: Permission denied

# FIXED
$ echo data | sudo tee /etc/protected
$ echo data | sudo tee -a /etc/protected     # append
$ sudo sh -c 'echo data > /etc/protected'
$ sudo bash -c 'cmd > /etc/protected'
```

### 5. `find -exec rm {} \;` vs `find -delete`

```bash
# WORKS but slow — fork rm per file
$ find . -name '*.tmp' -exec rm {} \;

# Faster — batch
$ find . -name '*.tmp' -exec rm {} +

# Fastest — built-in
$ find . -name '*.tmp' -delete
```

Note: `find -delete` requires the predicate before it (POSIX-find-portable: not all finds support it; GNU find does).

### 6. `awk` vs `gawk` vs `mawk`

```bash
# BROKEN — gensub() is gawk extension, fails on mawk (Debian default)
$ awk '{print gensub(/a/, "b", "g")}' file
awk: line 1: function gensub never defined

# FIXED
$ awk '{gsub(/a/, "b"); print}' file       # POSIX gsub
$ gawk '{print gensub(/a/, "b", "g")}' file  # use gawk explicitly
```

### 7. `xargs` without `-0` on filenames with spaces

```bash
# BROKEN — splits "my file" into "my" and "file"
$ find . -name '*.log' | xargs rm
rm: cannot remove 'my'
rm: cannot remove 'file'

# FIXED — null-separated
$ find . -name '*.log' -print0 | xargs -0 rm
$ find . -name '*.log' -exec rm {} +        # also fine
```

### 8. `cron` without absolute paths or `PATH`

```bash
# BROKEN — cron has minimal env; no /usr/local/bin
0 * * * * mybinary
# email: /bin/sh: 1: mybinary: not found

# FIXED
0 * * * * /usr/local/bin/mybinary
# OR set PATH at top of crontab
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
SHELL=/bin/bash
0 * * * * mybinary >> /var/log/myjob.log 2>&1
```

### 9. `tail -f` vs `tail -F` when log rotates

```bash
# BROKEN — keeps fd to old (now-rotated) file
$ tail -f /var/log/foo.log
# logrotate runs: foo.log → foo.log.1; tail keeps reading the empty old fd

# FIXED
$ tail -F /var/log/foo.log
# -F = -f --retry --follow=name; reopens by name when rotated
```

### 10. `tcpdump` needs root + the right interface

```bash
# BROKEN — wrong interface or no permission
$ tcpdump
tcpdump: no suitable device found
$ tcpdump -i eth0
tcpdump: eth0: You don't have permission to perform this capture

# FIXED
$ sudo tcpdump -i any -nn -vv 'tcp port 80'
# -i any   pseudo-interface for all
# -nn      don't resolve names or ports
# -vv      verbose
# Or grant capability:
$ sudo setcap cap_net_raw,cap_net_admin=eip $(which tcpdump)
```

### 11. `ip rule lookup table N` vs default route

Multiple routing tables: `ip rule` decides which table, then `ip route show table N` decides where. If a packet's source-IP matches a `from X lookup mytable` rule, it ignores `main`.

```bash
ip rule show
# 0:     from all lookup local
# 32766: from all lookup main
# 32767: from all lookup default

# Custom rule example (often shows up after VPN setup)
ip rule show
# 100:   from 10.8.0.0/24 lookup vpn

ip route show table main
ip route show table vpn       # if exists
ip route show table all       # everything
```

### 12. `-` as filename swallowed by tools

```bash
# BROKEN — tar -cf - is "stdout"; cat - is "stdin"
$ cat -
(reads stdin until EOF)

# FIXED — use ./- to disambiguate
$ cat ./-
$ rm ./-                        # rm a file literally named -
```

### 13. `bash` arithmetic with leading zero

```bash
# BROKEN — 08 is parsed as octal, fails
$ x=08; echo $((x + 1))
bash: 08: value too great for base (error token is "08")

# FIXED — force base-10
$ x=08; echo $((10#$x + 1))     # 9
```

### 14. `read` losing leading whitespace

```bash
# BROKEN — IFS strips leading whitespace
while read line; do echo "[$line]"; done < file

# FIXED
while IFS= read -r line; do echo "[$line]"; done < file
# IFS=  — disable splitting on any IFS
# -r    — don't process backslash escapes
```

### 15. `set -e` doesn't catch pipe errors

```bash
# BROKEN — exit code is from grep (0 because something matched), curl error swallowed
set -e
curl http://nope/missing | grep something

# FIXED
set -euo pipefail
# pipefail — pipeline returns first non-zero exit
```

## Idioms

```bash
# Always quote variables
"$var"          # right
$var            # wrong — splits on IFS, glob-expands
"$@"            # right way to pass argv
"$*"            # all args as single string (joined by IFS[0])

# Strict mode at top of scripts
set -euo pipefail
IFS=$'\n\t'                # fewer surprises with $@ expansion

# Cleanup on exit
trap 'rc=$?; rm -rf "$tmpdir"; exit $rc' EXIT
trap 'echo interrupted; exit 130' INT TERM

# Don't parse `ls`
for f in *; do ...; done             # right
for f in $(ls); do ...; done         # wrong — breaks on whitespace

# journalctl idioms
journalctl -fu nginx                 # follow this unit
journalctl -p err -b                 # errors this boot
journalctl --since '5 min ago' -p warn

# ss instead of netstat
ss -tlnp                             # listeners
ss -tan state established '( sport = :443 )'

# When systemd unit dies mysteriously
journalctl -u svc -n 50 --no-pager
dmesg | tail                         # kernel might've killed it (OOM etc)
coredumpctl info svc                 # if SIGSEGV/SIGABRT

# Find leaks of fds
ls -la /proc/PID/fd/ | wc -l
cat /proc/PID/limits | grep 'open files'

# Find what's eating disk fast
du -x -d 1 / 2>/dev/null | sort -h
ncdu -x /
find / -xdev -type f -size +500M 2>/dev/null

# Find what's eating mem fast
ps -eo pid,user,rss,vsz,comm --sort=-rss | head -20
smem -tk -s rss -r | head -20

# What was killed and why
dmesg -T | grep -i 'killed\|oom\|panic\|segfault'
journalctl -k -p err --since today

# Network sanity 60-sec script
echo --- DNS ---;     getent hosts example.com
echo --- ROUTE ---;   ip route get 1.1.1.1
echo --- ARP ---;     ip neigh show
echo --- LISTEN ---;  ss -tlnp
echo --- ESTAB ---;   ss -tan state established | head -20
echo --- DROPS ---;   ip -s -s link
echo --- CONNTR ---;  conntrack -S 2>/dev/null | head -5
```

## See Also

- bash
- systemd
- iptables
- dns
- troubleshooting/ssh-errors
- troubleshooting/git-errors
- troubleshooting/dns-errors

## References

- `man 3 errno` — the C errno API
- `man 3 strerror` — `strerror`, `strerror_r`
- `man 7 signal` — full signal disposition table; default actions
- `man 5 proc` — every `/proc/PID/*` and `/proc/sys/*` documented
- `man 5 systemd.exec` — systemd unit `Limit*=`, `OOMScoreAdjust=`, `Restart=`
- `man 5 systemd.service` — `Type=`, `ExecStart=`, `Restart*`
- `man 8 systemd-coredump` — kernel core integration
- `man 5 coredump.conf` — coredump storage
- `man 8 coredumpctl` — list/inspect cores
- `man 1 journalctl` — log query
- `man 1 dmesg` — kernel ring buffer
- `man 8 sysctl` — kernel parameter access
- Linux source: `include/uapi/asm-generic/errno-base.h` (1-34) and `errno.h` (35-133)
- Linux source: `mm/oom_kill.c` — OOM killer scoring logic
- POSIX 2017 §2.3 — Error Numbers
- POSIX 2017 `<signal.h>` — signal disposition
- Greg Kroah-Hartman, "Linux Kernel in a Nutshell" — boot, modules, taint flags
- Brendan Gregg, "Systems Performance" 2nd ed — diagnostic methodology, USE method
- RFC 1700 (historical) — assigned numbers including signal names
- Robert Love, "Linux System Programming" 2nd ed — errno, signals, /proc
- W. Richard Stevens, "Advanced Programming in the UNIX Environment" 3rd ed — POSIX errors and signals
- man-pages project: https://man7.org/linux/man-pages/
- Linux kernel docs: https://www.kernel.org/doc/html/latest/admin-guide/sysctl/
- systemd docs: https://www.freedesktop.org/software/systemd/man/
