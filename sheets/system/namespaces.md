# Namespaces (Linux Process Isolation)

Linux namespaces provide lightweight process isolation by partitioning kernel resources so that each group of processes sees its own independent set of PIDs, network stacks, mount points, users, and more.

## Namespace Types

| Namespace | Clone Flag | Isolates | Kernel Version |
|-----------|-----------|----------|----------------|
| PID | `CLONE_NEWPID` | Process IDs | 3.8 |
| NET | `CLONE_NEWNET` | Network stack | 2.6.29 |
| MNT | `CLONE_NEWNS` | Mount points | 2.4.19 |
| USER | `CLONE_NEWUSER` | UID/GID mappings | 3.8 |
| IPC | `CLONE_NEWIPC` | System V IPC, POSIX MQ | 2.6.19 |
| UTS | `CLONE_NEWUTS` | Hostname, domain name | 2.6.19 |
| Cgroup | `CLONE_NEWCGROUP` | Cgroup root | 4.6 |
| Time | `CLONE_NEWTIME` | Boot/monotonic clocks | 5.6 |

## Inspecting Namespaces

```bash
# List namespaces of a process
ls -la /proc/$PID/ns/
# lrwxrwxrwx 1 root root 0 cgroup -> 'cgroup:[4026531835]'
# lrwxrwxrwx 1 root root 0 ipc -> 'ipc:[4026531839]'
# lrwxrwxrwx 1 root root 0 mnt -> 'mnt:[4026531840]'
# lrwxrwxrwx 1 root root 0 net -> 'net:[4026531969]'
# lrwxrwxrwx 1 root root 0 pid -> 'pid:[4026531836]'
# lrwxrwxrwx 1 root root 0 user -> 'user:[4026531837]'
# lrwxrwxrwx 1 root root 0 uts -> 'uts:[4026531838]'

# Compare namespace IDs between two processes
readlink /proc/1/ns/net
readlink /proc/$PID/ns/net
# Different inode numbers = different namespaces

# List all namespaces on the system
lsns
# NS         TYPE   NPROCS   PID USER  COMMAND
# 4026531835 cgroup    150     1 root  /sbin/init
# 4026531836 pid       150     1 root  /sbin/init
# 4026532450 mnt         1  1234 root  nginx

# List specific namespace type
lsns -t net
```

## PID Namespace

```bash
# Create a new PID namespace
unshare --pid --fork --mount-proc bash

# Inside: PID 1 is the new bash
ps aux
# USER  PID %CPU %MEM COMMAND
# root    1  0.0  0.0 bash

# PID namespaces are nested — parent sees child PIDs
# but child cannot see parent PIDs

# Run a process in a new PID namespace
unshare -fp --mount-proc /bin/sh -c 'echo PID=$$; ps aux'
```

## Network Namespace

```bash
# Create a named network namespace
ip netns add myns

# List network namespaces
ip netns list

# Run command in network namespace
ip netns exec myns ip addr

# Create veth pair to connect namespaces
ip link add veth0 type veth peer name veth1
ip link set veth1 netns myns

# Configure the veth pair
ip addr add 10.0.0.1/24 dev veth0
ip link set veth0 up
ip netns exec myns ip addr add 10.0.0.2/24 dev veth1
ip netns exec myns ip link set veth1 up
ip netns exec myns ip link set lo up

# Test connectivity
ip netns exec myns ping 10.0.0.1

# Delete network namespace
ip netns delete myns
```

## Mount Namespace

```bash
# Create a new mount namespace
unshare --mount bash

# Mounts inside this namespace are invisible to the host
mount -t tmpfs tmpfs /mnt/private
echo "secret" > /mnt/private/data.txt
# /mnt/private/data.txt does not exist on the host

# Propagation types
mount --make-private /mnt     # No propagation
mount --make-shared /mnt      # Bidirectional propagation
mount --make-slave /mnt       # Host->NS only
mount --make-unbindable /mnt  # Cannot be bind-mounted

# Create overlay filesystem in namespace
mount -t overlay overlay \
  -o lowerdir=/lower,upperdir=/upper,workdir=/work /merged
```

## User Namespace

```bash
# Create user namespace (maps to root inside)
unshare --user --map-root-user bash

# Check mappings
cat /proc/self/uid_map
#          0       1000          1

# Manual UID/GID mapping (from parent namespace)
# Format: inside_id  outside_id  count
echo "0 1000 1" > /proc/$PID/uid_map
echo "0 1000 1" > /proc/$PID/gid_map

# Deny setgroups (required before writing gid_map as unprivileged)
echo "deny" > /proc/$PID/setgroups

# User namespaces enable unprivileged container creation
unshare --user --pid --fork --mount-proc --map-root-user bash
# Now root inside the namespace, unprivileged outside
```

## UTS Namespace

```bash
# Create new UTS namespace with custom hostname
unshare --uts bash
hostname container-01
hostname
# container-01

# Host hostname is unchanged
```

## IPC Namespace

```bash
# Create new IPC namespace
unshare --ipc bash

# IPC objects (semaphores, shared memory, message queues) are isolated
ipcs  # Empty — no IPC objects from host visible

# Create a shared memory segment
ipcmk -M 1024
ipcs -m  # Visible only inside this namespace
```

## Cgroup Namespace

```bash
# Create new cgroup namespace
unshare --cgroup bash

# Inside the namespace, cgroup paths are relative
cat /proc/self/cgroup
# 0::/                    (instead of full host path)
```

## Using unshare

```bash
# Combine multiple namespaces
unshare --pid --net --mount --uts --ipc --fork --mount-proc bash

# Run a command in new namespaces
unshare -pmn --fork /bin/sh -c '
  hostname isolated
  ip link set lo up
  mount -t proc proc /proc
  ps aux
'

# Common flags
# --pid           New PID namespace
# --net           New network namespace
# --mount         New mount namespace
# --uts           New UTS namespace
# --ipc           New IPC namespace
# --user          New user namespace
# --cgroup        New cgroup namespace
# --fork          Fork before exec (required for --pid)
# --mount-proc    Mount new /proc (requires --mount + --pid)
# --map-root-user Map current UID to root in new user NS
# -r              Shorthand for --map-root-user
```

## Using nsenter

```bash
# Enter all namespaces of a running container
nsenter -t $PID --all

# Enter specific namespaces
nsenter -t $PID --pid --net --mount

# Enter a Docker container's namespaces
CPID=$(docker inspect --format '{{.State.Pid}}' mycontainer)
nsenter -t $CPID --net ip addr

# Enter namespace from a file reference
nsenter --net=/var/run/netns/myns ip addr

# Debug a container's networking without exec
nsenter -t $CPID --net ss -tlnp
nsenter -t $CPID --net iptables -L -n
nsenter -t $CPID --net tcpdump -i eth0
```

## Namespace Lifecycle

```bash
# Namespaces persist as long as:
# 1. A process is inside them
# 2. A bind mount holds them open
# 3. A file descriptor references them

# Persist a namespace via bind mount
touch /var/run/ns/mynet
mount --bind /proc/$PID/ns/net /var/run/ns/mynet
# Namespace survives even if the process exits

# Open namespace via file descriptor (C)
# int fd = open("/proc/$PID/ns/net", O_RDONLY);
# setns(fd, CLONE_NEWNET);

# Persist namespace with unshare + mount
unshare --net --mount bash -c '
  mount --bind /proc/self/ns/net /var/run/ns/persistent
  sleep infinity
' &
```

## Tips

- Always use `--fork` with `--pid` in unshare; without it, the calling process becomes PID 1 and may confuse signal handling
- Use `--mount-proc` with PID namespaces so `ps` and `/proc` reflect the new PID space
- Network namespaces start with only a loopback interface; you must create veth pairs for connectivity
- User namespaces are the key to rootless containers; they map unprivileged UIDs to root inside
- `lsns` is your best friend for auditing namespace usage across the system
- Bind-mounting namespace files from `/proc/$PID/ns/` keeps namespaces alive after the process exits
- `nsenter` is more surgical than `docker exec` because you can pick exactly which namespaces to enter
- Mount propagation (shared/slave/private) controls whether mounts leak between namespaces
- Combining user + PID + mount namespaces creates a minimal unprivileged sandbox
- Each container runtime (Docker, Podman, LXC) uses the same namespace syscalls under the hood
- Time namespaces (kernel 5.6+) let you give containers different boot timestamps for testing

## See Also

cgroups, proc-sys, signals, ulimit

## References

- [Linux Kernel Namespaces Documentation](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [unshare(1) Man Page](https://man7.org/linux/man-pages/man1/unshare.1.html)
- [nsenter(1) Man Page](https://man7.org/linux/man-pages/man1/nsenter.1.html)
- [ip-netns(8) Man Page](https://man7.org/linux/man-pages/man8/ip-netns.8.html)
- [LWN.net: Namespaces in Operation](https://lwn.net/Articles/531114/)
