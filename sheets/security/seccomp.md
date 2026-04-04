# Seccomp (Secure Computing Mode)

Linux kernel feature that restricts the system calls a process can make, using BPF filter programs to define allow/deny policies per syscall number and arguments.

## Seccomp Modes

### Mode 1 — Strict

```bash
# Only allows read(), write(), _exit(), sigreturn()
# Process killed with SIGKILL on any other syscall
prctl(PR_SET_SECCOMP, SECCOMP_MODE_STRICT);
```

### Mode 2 — Filter (BPF)

```bash
# Allows custom BPF filter programs to decide per-syscall
prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, &prog);

# Or via seccomp() syscall (preferred since Linux 3.17)
seccomp(SECCOMP_SET_MODE_FILTER, flags, &prog);
```

### Check Seccomp Status

```bash
# Check if seccomp is enabled for a process
grep Seccomp /proc/$PID/status
# Seccomp:    0   (disabled)
# Seccomp:    1   (strict)
# Seccomp:    2   (filter)

# Check available seccomp actions
cat /proc/sys/kernel/seccomp/actions_avail
# kill_process kill_thread trap errno trace log allow
```

## BPF Filter Actions

### Return Values

```
SECCOMP_RET_KILL_PROCESS   # Kill entire process (since 4.14)
SECCOMP_RET_KILL_THREAD    # Kill offending thread
SECCOMP_RET_TRAP           # Send SIGSYS to process
SECCOMP_RET_ERRNO          # Return errno to caller
SECCOMP_RET_TRACE          # Notify ptrace tracer
SECCOMP_RET_LOG            # Allow but log (since 4.14)
SECCOMP_RET_ALLOW          # Allow syscall
```

### Filter Evaluation Order

```
# Highest priority wins (lowest numeric value)
# KILL_PROCESS > KILL_THREAD > TRAP > ERRNO > TRACE > LOG > ALLOW
# Multiple filters: all evaluated, strictest result applies
```

## Docker Seccomp Profiles

### Default Profile

```bash
# Docker blocks ~44 syscalls by default
# Run with default profile (automatic)
docker run --rm alpine sh

# Run with NO seccomp (dangerous)
docker run --security-opt seccomp=unconfined alpine sh

# Run with custom profile
docker run --security-opt seccomp=profile.json alpine sh

# Check if seccomp is active in container
docker inspect --format '{{.HostConfig.SecurityOpt}}' CONTAINER
```

### Custom Profile Format

```json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "defaultErrnoRet": 1,
  "architectures": ["SCMP_ARCH_X86_64", "SCMP_ARCH_AARCH64"],
  "syscalls": [
    {
      "names": ["read", "write", "exit", "exit_group",
                "openat", "close", "fstat", "mmap"],
      "action": "SCMP_ACT_ALLOW"
    },
    {
      "names": ["clone"],
      "action": "SCMP_ACT_ALLOW",
      "args": [
        {
          "index": 0,
          "value": 2114060288,
          "op": "SCMP_CMP_MASKED_EQ"
        }
      ]
    },
    {
      "names": ["mount", "umount2", "ptrace", "reboot"],
      "action": "SCMP_ACT_ERRNO",
      "errnoRet": 1
    }
  ]
}
```

### Profile Actions

```
SCMP_ACT_KILL_PROCESS    # Kill process
SCMP_ACT_KILL_THREAD     # Kill thread (alias: SCMP_ACT_KILL)
SCMP_ACT_TRAP            # SIGSYS
SCMP_ACT_ERRNO           # Return errno
SCMP_ACT_TRACE           # Notify tracer
SCMP_ACT_LOG             # Log and allow
SCMP_ACT_ALLOW           # Allow
```

## Kubernetes Seccomp

### Pod Security Context (v1.19+)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: secure-pod
spec:
  securityContext:
    seccompProfile:
      type: RuntimeDefault          # Use container runtime default
  containers:
  - name: app
    image: myapp:latest
    securityContext:
      seccompProfile:
        type: Localhost              # Custom profile
        localhostProfile: profiles/fine-grained.json
```

### Profile Types

```yaml
# RuntimeDefault — container runtime's built-in profile
# Localhost      — custom profile from node filesystem
#                  Path relative to kubelet's seccomp profile root
#                  Default: /var/lib/kubelet/seccomp/
# Unconfined     — no seccomp filtering (not recommended)
```

## Audit Mode

### Generate Profile via Strace

```bash
# Trace syscalls used by an application
strace -f -o /tmp/trace.log -e trace=all ./myapp

# Extract unique syscall names
awk -F'(' '{print $1}' /tmp/trace.log | sort -u | grep -v '^---\|^+++'

# Use OCI seccomp-bpf-hook to auto-generate profiles
sudo podman run --annotation io.containers.trace-syscall=of:/tmp/profile.json \
  myimage:latest
```

### Docker Audit Logging

```bash
# Run with logging profile to discover required syscalls
docker run --security-opt seccomp=log-all.json alpine sh -c "echo test"

# log-all.json uses SCMP_ACT_LOG as defaultAction
# Check audit log for denied/logged calls
journalctl -k | grep audit | grep seccomp
# Or: dmesg | grep seccomp
# Fields: syscall=NNN (use ausyscall to resolve names)
ausyscall --dump | grep "^NNN"
```

### Libseccomp Tools

```bash
# Install seccomp tools
apt install libseccomp-dev libseccomp-tools    # Debian/Ubuntu
dnf install libseccomp-devel                     # RHEL/Fedora

# Dump BPF filter from running process
seccomp-tools dump ./program

# Disassemble seccomp BPF bytecode
seccomp-tools disasm filter.bpf

# Emulate filter against specific syscall
seccomp-tools emu filter.bpf -s openat
```

## Go Seccomp Integration

### Using libseccomp-golang

```go
import (
    "log"
    libseccomp "github.com/seccomp/libseccomp-golang"
)

func applySeccomp() {
    // Default deny
    filter, err := libseccomp.NewFilter(libseccomp.ActErrno.SetReturnCode(1))
    if err != nil {
        log.Fatal(err)
    }
    defer filter.Release()

    // Allow essential syscalls
    allowed := []string{"read", "write", "exit", "exit_group",
        "openat", "close", "fstat", "mmap", "brk", "futex"}
    for _, name := range allowed {
        call, _ := libseccomp.GetSyscallFromName(name)
        filter.AddRule(call, libseccomp.ActAllow)
    }

    filter.Load()   // Apply filter to current process
}
```

## Tips

- Always start with audit/log mode before enforcing deny rules to discover required syscalls
- Seccomp filters are inherited by child processes and cannot be removed once applied
- Docker's default profile blocks dangerous calls like `reboot`, `mount`, `ptrace`, and `kexec_load`
- Use `SCMP_ACT_LOG` during development to catch missing syscalls without killing processes
- Architecture must match in profiles: x86_64 syscall numbers differ from ARM64
- Kubernetes `RuntimeDefault` is the recommended baseline for production pods
- Combine seccomp with AppArmor/SELinux for defense in depth (different layers)
- The `--security-opt seccomp=unconfined` flag disables all filtering and should never be used in production
- Profile argument filtering (`args`) allows fine-grained control over syscall parameters
- Seccomp filters use classic BPF (not eBPF) and have limited instruction sets
- Use `strace` or `perf trace` to identify the minimal syscall set your application needs
- Keep profiles in version control alongside Dockerfiles for reproducible security

## See Also

apparmor, selinux, container-security, capabilities, docker, podman

## References

- [Linux Kernel Seccomp Documentation](https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html)
- [Docker Seccomp Security Profiles](https://docs.docker.com/engine/security/seccomp/)
- [Kubernetes Seccomp Tutorial](https://kubernetes.io/docs/tutorials/security/seccomp/)
- [libseccomp Library](https://github.com/seccomp/libseccomp)
- [seccomp(2) Man Page](https://man7.org/linux/man-pages/man2/seccomp.2.html)
- [OCI Runtime Spec — Linux Seccomp](https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#seccomp)
