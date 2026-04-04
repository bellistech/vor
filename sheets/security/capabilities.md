# Capabilities (Linux Capabilities)

Linux divides the monolithic root privilege into distinct units called capabilities, allowing processes to hold only the specific privileges they need instead of full superuser access.

## Capability Sets

### Per-Thread Sets

```
Effective (E)    # Caps actually checked by kernel for permission
Permitted (P)    # Upper bound of caps thread can assume
Inheritable (I)  # Caps preserved across execve()
Bounding (B)     # Limit on caps that can ever be gained
Ambient (A)      # Caps automatically added to E and P on execve()
                 # (since Linux 4.3, for unprivileged programs)
```

### Transformation on execve()

```
# New process capability sets after exec:
P'(new) = (P(old) & B) | (F_P & B) | (F_I & I)
E'(new) = F_E ? P'(new) : A'(new)
I'(new) = I(old)
A'(new) = (A(old) & I(old)) & B

# F_P = file permitted, F_I = file inheritable, F_E = file effective bit
```

## Common Capabilities

### Network

```
CAP_NET_BIND_SERVICE   # Bind to ports < 1024
CAP_NET_RAW            # Use RAW/PACKET sockets (ping, tcpdump)
CAP_NET_ADMIN          # Network configuration (iptables, routes, interfaces)
CAP_NET_BROADCAST      # Allow broadcasting and multicast
```

### System

```
CAP_SYS_ADMIN          # Catch-all admin (mount, sethostname, many more)
CAP_SYS_PTRACE         # Trace/inspect any process
CAP_SYS_CHROOT         # Use chroot()
CAP_SYS_TIME           # Set system clock
CAP_SYS_BOOT           # Reboot the system
CAP_SYS_NICE           # Set process scheduling priority
CAP_SYS_RESOURCE       # Override resource limits
CAP_SYS_MODULE         # Load/unload kernel modules
CAP_SYS_RAWIO          # Direct I/O access (/dev/mem, iopl)
```

### File and Process

```
CAP_DAC_OVERRIDE       # Bypass file read/write/execute permission checks
CAP_DAC_READ_SEARCH    # Bypass file read and directory search
CAP_FOWNER             # Bypass file ownership checks
CAP_CHOWN              # Change file ownership
CAP_SETUID             # Arbitrary setuid()
CAP_SETGID             # Arbitrary setgid()
CAP_KILL               # Send signals to any process
CAP_FSETID             # Set setuid/setgid bits on files
CAP_MKNOD              # Create special files with mknod()
CAP_SETFCAP            # Set file capabilities
CAP_AUDIT_WRITE        # Write to kernel audit log
CAP_SETPCAP            # Modify process capabilities
```

## Managing Capabilities

### capsh — Capability Shell

```bash
# Show current process capabilities
capsh --print

# Run shell with specific caps only
capsh --caps="cap_net_bind_service+eip" -- -c "id"

# Decode capability hex value
capsh --decode=00000000a80425fb

# Drop all caps and run command
capsh --drop=all -- -c "./myserver"

# Run with specific caps from bounding set
capsh --keep=1 --uid=1000 \
  --caps="cap_net_bind_service+eip" -- -c "./server"
```

### setcap / getcap — File Capabilities

```bash
# Set capabilities on a binary
sudo setcap 'cap_net_bind_service=+ep' /usr/local/bin/myserver
sudo setcap 'cap_net_raw,cap_net_admin=+ep' /usr/sbin/tcpdump

# View capabilities on a file
getcap /usr/local/bin/myserver
# /usr/local/bin/myserver cap_net_bind_service=ep

# Recursively find all files with capabilities
getcap -r /usr/bin/ 2>/dev/null
getcap -r / 2>/dev/null

# Remove capabilities from a file
sudo setcap -r /usr/local/bin/myserver

# Set inheritable + permitted
sudo setcap 'cap_net_bind_service=+ip' /usr/local/bin/myserver
```

### Process Inspection

```bash
# View capabilities of a running process
cat /proc/$PID/status | grep Cap
# CapInh: 0000000000000000
# CapPrm: 00000000a80425fb
# CapEff: 00000000a80425fb
# CapBnd: 00000000a80425fb
# CapAmb: 0000000000000000

# Decode capability bits
capsh --decode=$(cat /proc/$PID/status | grep CapEff | awk '{print $2}')

# List all capabilities with numbers
grep -r '' /proc/sys/kernel/cap_last_cap
# Output: max capability number supported by kernel

# getpcaps utility
getpcaps $PID
```

## Docker Capabilities

### Default Capabilities (Docker grants 14)

```
AUDIT_WRITE    CHOWN          DAC_OVERRIDE   FOWNER
FSETID         KILL           MKNOD          NET_BIND_SERVICE
NET_RAW        SETFCAP        SETGID         SETPCAP
SETUID         SYS_CHROOT
```

### Managing Container Capabilities

```bash
# Drop all capabilities
docker run --cap-drop=ALL alpine sh

# Drop all, add only what's needed
docker run --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  --cap-add=CHOWN \
  alpine sh

# Add specific capability
docker run --cap-add=SYS_PTRACE alpine sh

# Add ALL capabilities (dangerous)
docker run --cap-add=ALL alpine sh

# Check container capabilities
docker exec CONTAINER cat /proc/1/status | grep Cap

# Inspect default caps
docker inspect --format '{{.HostConfig.CapAdd}} {{.HostConfig.CapDrop}}' CONTAINER
```

## Kubernetes Capabilities

### SecurityContext

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: secure-pod
spec:
  containers:
  - name: app
    image: myapp:latest
    securityContext:
      capabilities:
        drop:
        - ALL                     # Drop everything first
        add:
        - NET_BIND_SERVICE        # Then add only what's needed
      runAsNonRoot: true
      readOnlyRootFilesystem: true
```

### Pod Security Standards

```yaml
# Restricted profile (strictest)
# Capabilities MUST drop ALL
# May only add NET_BIND_SERVICE
# No privileged containers
# Must run as non-root

# Baseline profile (moderate)
# Cannot add: SYS_ADMIN, NET_RAW, SYS_PTRACE, etc.
# Allows most default Docker caps

# Privileged profile (unrestricted)
# No restrictions on capabilities
```

## Ambient Capabilities

### Setting Ambient Caps (Linux 4.3+)

```bash
# Ambient caps let unprivileged binaries inherit caps
# Without file caps or setuid bits

# Using prctl
prctl(PR_CAP_AMBIENT, PR_CAP_AMBIENT_RAISE, CAP_NET_BIND_SERVICE, 0, 0);

# Using capsh
capsh --addamb=cap_net_bind_service --uid=1000 -- -c "./server"

# Rules:
# - Cap must be in both Permitted and Inheritable to be ambient
# - Cleared on setuid/setgid exec
# - Inherited across execve() to non-capability-aware programs
```

## Tips

- Always follow the principle of least privilege: drop ALL caps, then add back only what is needed
- `CAP_SYS_ADMIN` is the most dangerous capability and almost equivalent to full root access
- File capabilities (setcap) are better than setuid root for granting specific privileges to binaries
- Use `getcap -r /` to audit all capability-enhanced binaries on a system
- Ambient capabilities solve the inheritance problem for non-setuid, non-file-cap binaries
- In Docker, `--cap-drop=ALL --cap-add=NEEDED` is the recommended security pattern
- The bounding set can only be reduced, never expanded, providing a hard ceiling on privilege
- Capabilities are per-thread, not per-process, which matters for multi-threaded applications
- `CAP_NET_BIND_SERVICE` is the most commonly needed cap for web servers on privileged ports
- Kubernetes Pod Security Standards enforce capability restrictions at the namespace level
- Use `capsh --print` to debug capability issues before and after privilege changes
- Some capabilities (like `CAP_SYS_MODULE`) should never be granted to containers

## See Also

seccomp, apparmor, selinux, container-security, docker, hardening-linux

## References

- [capabilities(7) Man Page](https://man7.org/linux/man-pages/man7/capabilities.7.html)
- [Docker Runtime Privilege and Capabilities](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities)
- [Kubernetes Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)
- [Linux Kernel Credentials Documentation](https://www.kernel.org/doc/html/latest/security/credentials.html)
- [capsh(1) Man Page](https://man7.org/linux/man-pages/man1/capsh.1.html)
