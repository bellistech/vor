# systemd (Linux Init System and Service Manager)

The PID 1 process and unified service manager on most modern Linux distributions, replacing SysV init / Upstart with declarative unit files, dependency-graph parallel boot, cgroup-based process tracking, journald logging, socket activation, timers, and a full ecosystem of helper daemons.

## Setup

Most modern Linux distributions ship systemd as PID 1: Debian (since 8/Jessie), Ubuntu (since 15.04), Fedora (since 15), RHEL/CentOS/Rocky/Alma (since 7), openSUSE/SLES, Arch, Manjaro. Notable exceptions: Alpine (OpenRC + busybox-init), Devuan (sysvinit), Void (runit), Slackware (BSD-style init), Gentoo (OpenRC by default).

```bash
# Verify systemd is PID 1
ps -p 1 -o comm=
# expected output: systemd

# Verify systemctl is on PATH and check version
systemctl --version
# example: systemd 254 (254-1-arch)
#          +PAM +AUDIT -SELINUX -APPARMOR ...

# Show feature flags compiled into this systemd build
systemctl --version | head -2
```

### Unit File Search Path

systemd looks for unit files in a fixed precedence order. The first match wins. Higher-precedence directories override lower ones, which is how distributions ship defaults that admins can override without touching package files.

```bash
# System unit search order (highest precedence first):
# /etc/systemd/system/                 admin overrides (write here)
# /run/systemd/system/                 runtime-generated units
# /usr/lib/systemd/system/             distro-shipped vendor units
# /lib/systemd/system/                 (often a symlink to /usr/lib on Debian)

# User unit search order (per-user):
# ~/.config/systemd/user/              user overrides (write here)
# /etc/systemd/user/                   admin-defined defaults for all users
# /run/systemd/user/                   runtime-generated user units
# /usr/lib/systemd/user/               distro-shipped user units

# Show the full search path systemd is using right now
systemd-analyze --system unit-paths
systemd-analyze --user unit-paths
```

### System vs User Instances

systemd runs two distinct instances. The **system** instance is PID 1, manages system-wide services, owns `system.slice`, and is reached with `systemctl` (no flag) or `systemctl --system`. The **user** instance runs once per logged-in user under `systemd --user`, manages per-user services, owns `user-UID.slice`, and is reached with `systemctl --user`. The two instances do not share unit files, do not share state, and have separate journals.

```bash
# Talk to the system instance (default)
systemctl status

# Talk to your user instance
systemctl --user status

# Check whether your user instance is even running
systemctl --user is-system-running

# Start a user instance for another user as root (rare)
systemctl --machine=alice@.host --user status
```

### Distro-Specific Notes

```bash
# Debian/Ubuntu: vendor units in /lib/systemd/system (symlink to /usr/lib)
# Use 'systemctl edit X' rather than editing in /lib directly

# Fedora/RHEL: vendor units in /usr/lib/systemd/system
# Most services ship as both .service and .socket

# Arch Linux: vendor units in /usr/lib/systemd/system
# Pacman never touches /etc/systemd/system, so admin overrides are safe

# CentOS Stream / Rocky / Alma: same layout as RHEL
# SELinux contexts on units: system_u:object_r:systemd_unit_file_t:s0
```

## Architecture

systemd was created by Lennart Poettering and Kay Sievers (announced 2010, first release 2010-03-30). It replaces the imperative shell-script-driven SysV init with a declarative dependency graph executed in parallel.

### PID 1

```bash
# systemd is the first user-space process, started by the kernel after init=
# Check the kernel command line that selected it
cat /proc/cmdline
# typical: BOOT_IMAGE=... root=UUID=... rw quiet splash init=/usr/lib/systemd/systemd

# The /sbin/init symlink points to systemd on systemd-based distros
ls -l /sbin/init
# lrwxrwxrwx 1 root root 22 ... /sbin/init -> ../lib/systemd/systemd
```

### Unit Lifecycle

A unit goes through a finite state machine: `inactive` → `activating` → `active` → `deactivating` → `inactive` (or `failed`). Sub-states refine this: `active (running)`, `active (exited)`, `active (waiting)`, `failed`.

```bash
# Show high-level state and sub-state of a unit
systemctl show -p ActiveState -p SubState nginx.service

# Watch a unit transition through states in real time
watch -n0.5 'systemctl show -p ActiveState -p SubState nginx.service'
```

### Dependency Graph Executor

systemd builds a directed graph from `Wants=`, `Requires=`, `After=`, `Before=`, etc. and executes nodes in topological order, in parallel where possible. `Wants=` is a soft pull (start if not started); `After=` is pure ordering. Confusing the two is the most common gotcha — see Gotchas section.

```bash
# Print the full transitive dependency tree
systemctl list-dependencies multi-user.target

# Print reverse dependencies (who pulls this unit?)
systemctl list-dependencies --reverse nginx.service

# Show only the units that are currently active in the tree
systemctl list-dependencies --all multi-user.target | head -50
```

### D-Bus IPC

systemd exposes its API over D-Bus on the well-known name `org.freedesktop.systemd1`. `systemctl` is itself a D-Bus client. Any language with a D-Bus binding can drive systemd directly.

```bash
# List all systemd units via raw D-Bus
busctl call org.freedesktop.systemd1 \
  /org/freedesktop/systemd1 \
  org.freedesktop.systemd1.Manager \
  ListUnits

# Introspect the systemd1 service
busctl introspect org.freedesktop.systemd1 /org/freedesktop/systemd1
```

### cgroup Integration

Every service runs in its own cgroup under `/sys/fs/cgroup/system.slice/<unit>.service`. Modern systems use cgroup-v2 unified hierarchy. systemd uses cgroups to track which processes belong to a unit (no PID-file races) and to enforce resource limits.

```bash
# Show the cgroup tree as systemd sees it
systemd-cgls

# Show top-style live cgroup resource usage
systemd-cgtop

# Find which unit owns a PID
systemctl status 1234
# or
ps -o cgroup= -p 1234
```

### Socket Activation

systemd binds the listening socket itself, then passes the file descriptor to the service via `$LISTEN_FDS`. This enables on-demand startup, graceful upgrade (the socket survives service restarts), and parallel boot (clients can connect before the service is ready — connections queue in the socket buffer).

```bash
# Check whether a service has an associated socket
systemctl list-sockets --all

# Show which file descriptors a socket-activated service inherited
systemctl show -p ListenStream -p ListenDatagram cups.socket
```

## Unit Types Catalog

The unit type is determined by the file suffix. Each type has type-specific directives in addition to the common `[Unit]` and `[Install]` sections.

| Suffix       | Purpose                                                            |
|--------------|--------------------------------------------------------------------|
| `.service`   | A daemon or one-shot program — the most common unit type.          |
| `.socket`    | An IPC/network socket whose connections trigger an associated `.service`. |
| `.target`    | A synchronization point — like a SysV runlevel, but pure dep-graph. |
| `.timer`     | A schedule that triggers a `.service` (cron replacement).          |
| `.path`      | A filesystem watch that triggers a `.service` when paths appear/change. |
| `.mount`     | A filesystem mount point — auto-generated from `/etc/fstab` or hand-written. |
| `.automount` | Lazy mount — the kernel autofs triggers the matching `.mount` on first access. |
| `.swap`      | A swap partition or swap file.                                     |
| `.device`    | A udev-managed device — usually auto-generated; rarely written by hand. |
| `.scope`     | A transient unit wrapping already-running processes (e.g., a tmux session). |
| `.slice`     | A cgroup hierarchy node for grouping and limiting resources.       |

```bash
# List units of a specific type
systemctl list-units --type=service
systemctl list-units --type=timer
systemctl list-units --type=socket
systemctl list-units --type=target
systemctl list-units --type=mount

# List all unit files of a type (regardless of state)
systemctl list-unit-files --type=service
```

## Service Units (.service)

A `.service` unit describes how to start, stop, monitor, and restart a daemon or one-shot program. It is the workhorse unit type.

### Skeleton

```bash
# /etc/systemd/system/myapp.service
[Unit]
Description=My Application server
Documentation=https://example.com/myapp/docs
After=network-online.target postgresql.service
Wants=network-online.target
Requires=postgresql.service

[Service]
Type=simple
User=myapp
Group=myapp
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/bin/server --config /etc/myapp/config.yaml
ExecReload=/bin/kill -HUP $MAINPID
ExecStop=/bin/kill -TERM $MAINPID
Restart=on-failure
RestartSec=5s
TimeoutStartSec=30s
TimeoutStopSec=30s
Environment=NODE_ENV=production
Environment=PORT=8080
EnvironmentFile=-/etc/myapp/env

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/myapp /var/log/myapp
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096
MemoryMax=2G
CPUQuota=200%

[Install]
WantedBy=multi-user.target
```

### [Unit] Directives

```bash
# Identification and docs
Description=One-line description shown in systemctl status
Documentation=https://... or man:foo(1) or file:/usr/share/doc/...

# Ordering (does NOT pull units in)
After=postgresql.service network-online.target
Before=httpd.service

# Soft pull (units start, but failure is non-fatal)
Wants=elasticsearch.service

# Hard pull (units start; failure aborts this unit)
Requires=postgresql.service

# Hard pull AND already running (failure if not active when this starts)
Requisite=mounted-data.mount

# Stop together (if listed unit stops, this one stops too)
BindsTo=hardware-device.device

# This unit is part of a group; restart/stop with the group
PartOf=apache.service

# Stop me if this conflicts with another unit
Conflicts=shutdown.target

# Run this on failure
OnFailure=alert-admin@%n.service

# Skip if condition fails (no failure recorded)
ConditionPathExists=/var/lib/myapp
ConditionPathIsDirectory=/var/lib/myapp
ConditionFileNotEmpty=/etc/myapp/config.yaml
ConditionDirectoryNotEmpty=/var/spool/myapp
ConditionFileIsExecutable=/opt/myapp/bin/server
ConditionKernelCommandLine=foo=bar
ConditionVirtualization=container
ConditionHost=db1.prod.example.com
ConditionACPower=true

# Like Condition*= but missing the condition is a hard failure
AssertPathExists=/etc/myapp/license.key
```

### [Service] Type=

The `Type=` directive controls how systemd decides the service is "started":

| Type        | Started when...                                                          |
|-------------|--------------------------------------------------------------------------|
| `simple`    | (default if no `BusName=`) `ExecStart` is `fork()`+`execve()`'d.         |
| `exec`      | `execve()` returns successfully — stricter than `simple`.                |
| `forking`   | The initial process forks and exits; child becomes the daemon. Use `PIDFile=`. |
| `oneshot`   | `ExecStart` runs to completion (and exits 0). Combine with `RemainAfterExit=true` to stay "active". |
| `dbus`      | The service registers `BusName=` on D-Bus.                               |
| `notify`    | The service calls `sd_notify(READY=1)`.                                  |
| `notify-reload` | Like `notify`, with `sd_notify(RELOADING=1)` semantics for reload (v253+). |
| `idle`      | Like `simple`, but waits up to 5s for other jobs to finish first (cosmetic — clean console). |

```bash
# Type=simple — the default for foreground daemons
[Service]
Type=simple
ExecStart=/usr/bin/myserver --foreground

# Type=forking — for traditional daemons that double-fork
[Service]
Type=forking
PIDFile=/run/myserver.pid
ExecStart=/usr/sbin/myserver

# Type=oneshot — for setup/teardown scripts
[Service]
Type=oneshot
RemainAfterExit=true
ExecStart=/usr/local/bin/setup.sh
ExecStop=/usr/local/bin/teardown.sh

# Type=notify — for sd_notify-aware daemons (postgres, sshd, networkd)
[Service]
Type=notify
NotifyAccess=main
ExecStart=/usr/bin/myserver

# Type=dbus — for services that publish a D-Bus name
[Service]
Type=dbus
BusName=org.example.MyApp
ExecStart=/usr/bin/myapp
```

### [Service] Exec*= directives

```bash
# Exec lines run in order: Pre → Start → Post; on stop: ExecStop → ExecStopPost
ExecStartPre=/usr/local/bin/check-config.sh
ExecStartPre=-/bin/rm -f /run/myapp.lock      # leading "-" = ignore failure
ExecStart=/usr/bin/myapp
ExecStartPost=/usr/local/bin/notify-started.sh
ExecReload=/bin/kill -HUP $MAINPID
ExecStop=/bin/kill -TERM $MAINPID
ExecStopPost=/usr/local/bin/cleanup.sh

# ExecStart with multiple processes (rare; only Type=oneshot)
[Service]
Type=oneshot
ExecStart=/bin/echo step 1
ExecStart=/bin/echo step 2
ExecStart=/bin/echo step 3

# Special prefixes:
#   -    ignore failure
#   @    set argv[0] explicitly: ExecStart=@/path/program argv0 arg1 arg2
#   :    do not perform variable expansion
#   +    run with full privileges, ignoring User=/Group= (v231+)
#   !    run with credentials but without manipulating capabilities (v240+)
#   !!   like ! but ambient caps still applied if supported (v240+)
```

### [Service] Restart and Watchdog

```bash
# Restart policy:
#   no              never restart (default)
#   always          restart no matter what
#   on-success      only on clean exit (rare)
#   on-failure      non-zero exit, signal, timeout, or watchdog
#   on-abnormal     signal, timeout, watchdog (NOT non-zero exit)
#   on-watchdog     only watchdog timeout
#   on-abort        only on uncaught signal (e.g., SIGSEGV)
Restart=on-failure
RestartSec=5s
RestartSteps=10
RestartMaxDelaySec=5min     # exponential backoff cap (v254+)

# Limit restart-loop:
StartLimitIntervalSec=300
StartLimitBurst=5
# After 5 starts in 300s, refuse further restarts; clear with:
#   systemctl reset-failed myapp.service

# Watchdog:
WatchdogSec=30s
# Service must call sd_notify(WATCHDOG=1) every <30s or systemd kills it.

# Timeouts:
TimeoutStartSec=90s
TimeoutStopSec=30s
TimeoutAbortSec=10s
TimeoutSec=60s              # shortcut for both Start and Stop
```

### [Service] User, Group, and Filesystem

```bash
User=myapp
Group=myapp
DynamicUser=true            # v235+: ephemeral UID/GID, auto-managed
SupplementaryGroups=audio video

WorkingDirectory=/var/lib/myapp
RootDirectory=/var/empty    # chroot
RootImage=/var/lib/myapp.img  # mount disk image as root
RootImageOptions=ro

UMask=0027
```

### [Service] Environment

```bash
Environment="NODE_ENV=production"
Environment="DATABASE_URL=postgres://db/app" "LOG_LEVEL=info"

EnvironmentFile=/etc/myapp/env
EnvironmentFile=-/etc/myapp/env.optional   # leading "-" = OK if missing

PassEnvironment=HOME PATH                  # inherit specific vars
UnsetEnvironment=LD_PRELOAD                # explicitly clear
```

### [Service] Standard I/O

```bash
# StandardInput=:  null (default), tty, tty-force, tty-fail, data, file:/path,
#                  socket, fd:NAME
# StandardOutput=: inherit, null, tty, journal (default), kmsg, journal+console,
#                  file:/path, append:/path, truncate:/path, socket, fd:NAME
# StandardError=:  same options as StandardOutput

StandardInput=null
StandardOutput=journal
StandardError=journal
StandardOutput=append:/var/log/myapp.log     # v240+
StandardOutput=file:/var/log/myapp.log
SyslogIdentifier=myapp
SyslogFacility=daemon
SyslogLevel=info
```

### [Service] Sandboxing (the security goldmine)

```bash
# Filesystem isolation
PrivateTmp=true                              # private /tmp /var/tmp
ProtectSystem=strict                         # /usr /boot /etc read-only (or: full, yes)
ProtectHome=true                             # /home /root /run/user empty (or: read-only, tmpfs)
ReadWritePaths=/var/lib/myapp /var/log/myapp
ReadOnlyPaths=/etc/myapp
InaccessiblePaths=/srv/sensitive
PrivateDevices=true                          # only /dev/null, zero, random, urandom, tty, ptmx, log
PrivateMounts=true                           # private mount namespace; mounts don't leak (v239+)
PrivateUsers=true                            # private UID namespace; root inside ≠ root outside
PrivateNetwork=true                          # no network at all
PrivateIPC=true                              # private SysV IPC namespace (v248+)
ProtectClock=true                            # cannot set system clock (v246+)
ProtectHostname=true                         # cannot set hostname (v242+)
ProtectKernelTunables=true                   # /proc/sys, /sys read-only
ProtectKernelModules=true                    # cannot load/unload modules
ProtectKernelLogs=true                       # cannot read kernel ring buffer (v244+)
ProtectControlGroups=true                    # /sys/fs/cgroup read-only
ProtectProc=invisible                        # /proc only your processes (v247+)
ProcSubset=pid                               # /proc has only pid dirs (v247+)

# Capabilities
NoNewPrivileges=true                         # block setuid escalation
CapabilityBoundingSet=                       # empty = drop all
CapabilityBoundingSet=CAP_NET_BIND_SERVICE   # only allow this one
AmbientCapabilities=CAP_NET_BIND_SERVICE     # grant on exec (v229+)

# Misc hardening
LockPersonality=true                         # lock personality(2) (v235+)
MemoryDenyWriteExecute=true                  # block W^X violations (v231+)
RestrictRealtime=true                        # no SCHED_FIFO/RR (v231+)
RestrictSUIDSGID=true                        # no setuid/setgid bits (v242+)
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
RestrictNamespaces=true                      # no unshare(2) namespaces
RemoveIPC=true                               # clean up SysV IPC at exit
KeyringMode=private                          # keyring isolation (v235+)

# System call filtering (see SystemCallFilter section)
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources
SystemCallErrorNumber=EPERM
SystemCallArchitectures=native
```

### [Service] Resource Limits

```bash
# rlimit-style (per-process):
LimitNOFILE=65536            # max open file descriptors
LimitNPROC=4096              # max processes
LimitCORE=infinity           # core dump size
LimitMEMLOCK=8M              # mlock'd memory
LimitSTACK=8M
LimitFSIZE=infinity
LimitCPU=infinity
LimitRSS=infinity            # ignored on modern kernels — use MemoryMax

# cgroup-v2 (per-unit):
MemoryMax=2G
MemoryHigh=1500M             # soft throttle threshold
MemorySwapMax=0              # disable swap for this unit (v232+)
MemoryLow=512M               # protected from reclaim
TasksMax=512                 # process+thread cap

CPUQuota=200%                # 2 cores worth
CPUWeight=200                # relative weight (default 100, range 1..10000)
AllowedCPUs=0-3              # CPU pinning (v244+)

IOWeight=100
IOReadBandwidthMax=/dev/sda 50M
IOWriteBandwidthMax=/dev/sda 50M
IOReadIOPSMax=/dev/sda 1000

# Scheduling
Nice=10                      # niceness (-20..19)
CPUSchedulingPolicy=batch    # other, batch, idle, fifo, rr
CPUSchedulingPriority=50     # 1..99 for fifo/rr
IOSchedulingClass=best-effort # none, realtime, best-effort, idle
IOSchedulingPriority=4       # 0..7 (lower = higher priority)
```

### [Service] Signals

```bash
KillMode=control-group       # SIGTERM the whole cgroup (default)
KillMode=mixed               # SIGTERM main, SIGKILL the rest
KillMode=process             # SIGTERM only main process
KillMode=none                # don't send anything (rare)

KillSignal=SIGTERM           # default
KillSignal=SIGINT
RestartKillSignal=SIGTERM
SendSIGHUP=false             # also send SIGHUP before SIGTERM
SendSIGKILL=true             # send SIGKILL after TimeoutStopSec
FinalKillSignal=SIGKILL
```

### [Install] section

```bash
# WantedBy/RequiredBy create symlinks when 'systemctl enable' runs.
# multi-user.target.wants/myapp.service → ../myapp.service

WantedBy=multi-user.target           # most common — start at non-graphical boot
WantedBy=graphical.target            # GUI boot
RequiredBy=critical-target.target    # like Wants= but Requires=

Alias=mywebserver.service            # additional symlink names
Also=myapp-helper.service            # enable this unit too

DefaultInstance=production           # for foo@.service templates
```

## Service Types — When to Use

### Type=simple (default)

Most modern daemons. systemd considers the service "started" the moment `fork()` + `execve()` succeed — it does not wait for readiness. Use this when your program runs in the foreground and writes logs to stdout.

```bash
[Service]
Type=simple
ExecStart=/usr/bin/python3 /opt/app/server.py
```

Drawback: dependents start immediately, even if your service is still initializing. If you need true readiness, use `Type=notify`.

### Type=exec

Stricter than `simple`: systemd waits for `execve()` to return successfully before marking the unit started. Catches cases where the binary doesn't exist or has the wrong permissions. Available in v240+.

```bash
[Service]
Type=exec
ExecStart=/usr/bin/myapp
```

### Type=forking

For traditional Unix daemons that fork into the background and exit. systemd waits for the parent to exit, then the surviving child is the "main" process. Almost always combine with `PIDFile=` so systemd knows which child.

```bash
[Service]
Type=forking
PIDFile=/run/sshd.pid
ExecStart=/usr/sbin/sshd
```

Modern advice: prefer `Type=simple` and let your daemon stay in the foreground. systemd doesn't need backgrounding.

### Type=oneshot

For setup/teardown scripts that run to completion and exit. systemd considers the unit "active (exited)" once `ExecStart` returns 0. Combine with `RemainAfterExit=true` to keep the unit "active" after the script finishes (so its dependents stay activated).

```bash
[Service]
Type=oneshot
RemainAfterExit=true
ExecStart=/usr/local/bin/iptables-restore /etc/iptables.rules
ExecStop=/usr/local/bin/iptables-restore /etc/iptables.empty
```

Used heavily for cron-replacement timer-driven jobs (see Timer-Driven Backup Pattern).

### Type=notify

The gold standard for daemons. The service signals readiness via `sd_notify(3)`, and systemd holds dependents until then. PostgreSQL, sshd, systemd-networkd, and most modern daemons use this.

```bash
[Service]
Type=notify
NotifyAccess=main
ExecStart=/usr/bin/postgres -D /var/lib/postgres/data
```

In your code:
```c
#include <systemd/sd-daemon.h>
sd_notify(0, "READY=1\n"
             "STATUS=Listening on port 5432\n"
             "MAINPID=12345\n");
```

### Type=dbus

The service publishes a name on the system bus. systemd considers it ready when `BusName=` appears.

```bash
[Service]
Type=dbus
BusName=org.example.MyApp
ExecStart=/usr/bin/myapp
```

### Type=idle

Cosmetic only — runs after other jobs complete (up to 5s wait). Used for `getty@` so login prompt doesn't spam over boot messages.

## Targets

A target is a synchronization point — a "named milestone" in the dependency graph. Targets do not "run" anything; they exist when their `Wants=`/`Requires=` chain is satisfied. They replace SysV runlevels.

### Canonical Targets

| Target              | Purpose                                                          |
|---------------------|------------------------------------------------------------------|
| `default.target`    | Symlink to graphical or multi-user — what to boot into.         |
| `multi-user.target` | Multi-user CLI, networking, daemons. (≈ runlevel 3)              |
| `graphical.target`  | multi-user + display manager (gdm/sddm/lightdm). (≈ runlevel 5)  |
| `rescue.target`     | Single-user, root shell, minimal services. (≈ runlevel 1, single) |
| `emergency.target`  | Even more minimal — only emergency shell, root rw not guaranteed. |
| `network.target`    | Network configured (interfaces brought up).                      |
| `network-online.target` | Network has L3 connectivity.                                |
| `getty.target`      | Pulls in `getty@ttyN.service` instances.                         |
| `sleep.target`      | Generic sleep state.                                              |
| `suspend.target`    | RAM suspend (S3).                                                 |
| `hibernate.target`  | Disk hibernate (S4).                                              |
| `hybrid-sleep.target` | RAM + disk.                                                     |
| `halt.target`       | Halt CPU.                                                         |
| `poweroff.target`   | Power off.                                                        |
| `reboot.target`     | Reboot.                                                           |
| `shutdown.target`   | Aborts running services on shutdown.                             |
| `local-fs.target`   | All local filesystems mounted.                                   |
| `remote-fs.target`  | Network filesystems (NFS, CIFS) mounted.                         |
| `swap.target`       | All swap activated.                                              |
| `timers.target`     | All timers ready.                                                |
| `sockets.target`    | All socket-activated services have their sockets ready.          |
| `paths.target`      | All path units watching.                                          |
| `basic.target`      | All sysinit, sockets, timers, paths, slices ready.               |
| `sysinit.target`    | Early boot done — fs mounted, swap on, devices probed.           |

### Managing the Default

```bash
# What target does this system boot into?
systemctl get-default

# Boot into multi-user (no GUI) by default
systemctl set-default multi-user.target
# → symlinks /etc/systemd/system/default.target to multi-user.target

# Boot into GUI by default
systemctl set-default graphical.target

# Switch right now without reboot (like SysV "init 3" / "init 5")
systemctl isolate multi-user.target
systemctl isolate graphical.target

# Drop to single-user shell now (root password required for login)
systemctl isolate rescue.target

# Truly minimal recovery shell (filesystem may be ro)
systemctl isolate emergency.target
```

### Targets are Synchronization-Only

A target is "active" if its dependency closure is satisfied. There is no process associated. You cannot `ExecStart=` a target. To boot into a target, you `WantedBy=` services into it.

```bash
# Inspect a target's contents
systemctl cat multi-user.target

# What is currently pulled into multi-user.target?
ls /etc/systemd/system/multi-user.target.wants/
ls /usr/lib/systemd/system/multi-user.target.wants/
```

## Socket Units (.socket)

A `.socket` unit binds the listening socket itself, then activates a paired `.service` (default: same name) when traffic arrives. Two activation styles:

- **Accept=no** (default, "Inetd-style"): the service inherits the listening fd via `$LISTEN_FDS` and accepts connections itself — one service handles all connections.
- **Accept=yes**: systemd accepts each connection and spawns one instance of `foo@N.service` per connection — pair with `foo@.service` template.

### Inetd-style (Accept=no)

```bash
# /etc/systemd/system/myweb.socket
[Unit]
Description=Web server socket

[Socket]
ListenStream=80
ListenStream=443
Accept=no

[Install]
WantedBy=sockets.target

# /etc/systemd/system/myweb.service
[Unit]
Description=Web server
Requires=myweb.socket
After=myweb.socket

[Service]
Type=simple
ExecStart=/usr/local/bin/myweb
StandardInput=socket

[Install]
# (no [Install] needed — pulled in by myweb.socket)
```

```bash
# Enable and start (the socket — not the service)
systemctl enable --now myweb.socket
# First connection wakes myweb.service.
```

### Per-Connection (Accept=yes) — classic inetd model

```bash
# /etc/systemd/system/myecho.socket
[Socket]
ListenStream=2023
Accept=yes

[Install]
WantedBy=sockets.target

# /etc/systemd/system/myecho@.service   (note the @)
[Service]
ExecStart=/usr/bin/cat
StandardInput=socket
StandardOutput=socket
```

### Listen Directives

```bash
ListenStream=80                     # TCP port
ListenStream=[::]:80                # explicit IPv6 + IPv4
ListenStream=192.0.2.1:80           # bind specific interface
ListenStream=/run/myapp.sock        # Unix stream socket
ListenDatagram=53                   # UDP port
ListenDatagram=/run/myapp-udp.sock  # Unix datagram
ListenSequentialPacket=/run/myapp.seq
ListenFIFO=/run/myapp.fifo
ListenSpecial=/dev/log
ListenNetlink=kobject-uevent 1
ListenMessageQueue=/myqueue
ListenUSBFunction=/sys/...

# Socket options
BindIPv6Only=both                   # default, ipv6-only, both
SocketUser=root
SocketGroup=root
SocketMode=0660                     # for Unix sockets
DirectoryMode=0755
Backlog=128
KeepAlive=true
KeepAliveTimeSec=7200
TCPCongestion=bbr
ReusePort=true
SendBuffer=4M
ReceiveBuffer=4M
PassCredentials=true                # SCM_CREDENTIALS over Unix
PassSecurity=true                   # SELinux context
PassPacketInfo=true                 # IP_PKTINFO

FreeBind=true                       # bind to non-existent IP
Transparent=true                    # IP_TRANSPARENT
NoDelay=true                        # TCP_NODELAY
IPTOS=low-delay
IPTTL=64
Mark=0xFF                           # SO_MARK

# Triggering the service
Service=otherthing.service          # override default name
```

### Verify Socket Activation

```bash
# List all socket units
systemctl list-sockets --all

# Show which fds the service inherited
systemctl show -p ListenStream cups.socket

# Once running, the service should see:
#   echo $LISTEN_FDS                  → 1 (or however many)
#   echo $LISTEN_FDNAMES              → "myweb.socket"
#   echo $LISTEN_PID                  → matches getpid()
```

## Timer Units (.timer)

A `.timer` unit triggers a paired `.service` (default name). Timers are systemd's cron replacement — better logging (journald), better dependencies (`After=network-online.target`), better hardening (full `[Service]` sandboxing).

### Calendar-Based Timer (replaces cron)

```bash
# /etc/systemd/system/backup.timer
[Unit]
Description=Daily backup

[Timer]
OnCalendar=daily                    # 00:00:00 every day
OnCalendar=*-*-* 02:30:00          # 02:30 every day
OnCalendar=Mon..Fri 09:00:00       # weekday 09:00
OnCalendar=Mon *-*-* 09:00:00      # Monday 09:00
OnCalendar=*:0/15                   # every 15 minutes
OnCalendar=2024-01-01 00:00:00     # one specific moment
Persistent=true                     # catch up missed runs after downtime
RandomizedDelaySec=15min            # spread load across machines
AccuracySec=1min                    # default 1min; tighter = more wakeups
Unit=backup.service                 # default: same basename, but explicit is clearer

[Install]
WantedBy=timers.target
```

```bash
# /etc/systemd/system/backup.service
[Unit]
Description=Backup job

[Service]
Type=oneshot
ExecStart=/usr/local/bin/backup.sh
User=backup
Nice=19
IOSchedulingClass=idle
ProtectSystem=strict
ReadWritePaths=/srv/backups
```

### Monotonic (Boot/Activation-Relative) Timer

```bash
[Timer]
OnBootSec=15min                     # 15 min after boot
OnStartupSec=10min                  # 10 min after systemd started (rare)
OnUnitActiveSec=1h                  # 1h after unit last became active
OnUnitInactiveSec=30min             # 30 min after unit last became inactive
OnActiveSec=5min                    # 5 min after timer activated
```

```bash
# Combine: run 15min after boot, then every hour
[Timer]
OnBootSec=15min
OnUnitActiveSec=1h
```

### OnCalendar= Syntax Reference

```bash
# General: DayOfWeek Year-Month-Day Hour:Minute:Second [Timezone]

OnCalendar=minutely       # *:*:00
OnCalendar=hourly         # *:00:00
OnCalendar=daily          # *-*-* 00:00:00
OnCalendar=weekly         # Mon *-*-* 00:00:00
OnCalendar=monthly        # *-*-01 00:00:00
OnCalendar=quarterly      # *-01,04,07,10-01 00:00:00
OnCalendar=semiannually   # *-01,07-01 00:00:00
OnCalendar=yearly         # *-01-01 00:00:00

# Lists, ranges, modulo:
OnCalendar=Mon,Wed,Fri *-*-* 08:00:00
OnCalendar=Mon..Fri *-*-* 09:00:00
OnCalendar=*-*-* 0/4:00:00          # every 4 hours
OnCalendar=*-*-* *:0/30             # every 30 min

# Year/month/day specifics:
OnCalendar=2025-12-25 00:00:00      # Christmas 2025
OnCalendar=*-*-01 03:00:00          # 1st of every month at 03:00
OnCalendar=*-*-* 23:00..23:59:00    # range within hour

# Timezone:
OnCalendar=Mon *-*-* 09:00:00 America/New_York
```

### Verify a Calendar Spec

```bash
# Validate the spec and show the next 5 trigger times
systemd-analyze calendar "Mon..Fri 09:00"
# Output:
#   Original form: Mon..Fri 09:00
#   Normalized form: Mon..Fri *-*-* 09:00:00
#   Next elapse: Mon 2025-04-28 09:00:00 UTC
#   From now: 2 days left

# Compute next 5 occurrences
systemd-analyze calendar --iterations=5 "*-*-* 0/4:00:00"
```

### List Timers

```bash
# All timers, when they last triggered, when they fire next
systemctl list-timers

# Include disabled/inactive
systemctl list-timers --all
```

### Persistent= Behaviour

`Persistent=true` makes the timer remember the last firing on disk under `/var/lib/systemd/timers/`. If the system was off when the trigger should have fired, the service runs as soon as the timer is loaded after boot. Required for "daily backup" semantics on machines that aren't always on.

## Path Units (.path)

A `.path` unit watches the filesystem and triggers a paired `.service` when conditions match. Replaces ad-hoc `inotifywait` loops.

```bash
# /etc/systemd/system/incoming.path
[Unit]
Description=Watch /srv/incoming for new files

[Path]
PathExists=/srv/incoming/trigger    # exists at start, or appears
PathExistsGlob=/srv/incoming/*.tar.gz
PathChanged=/etc/myapp.conf         # any close-after-write
PathModified=/var/log/myapp.log     # any write
DirectoryNotEmpty=/srv/incoming
Unit=process-incoming.service       # default: same basename

[Install]
WantedBy=multi-user.target
```

```bash
# /etc/systemd/system/process-incoming.service
[Service]
Type=oneshot
ExecStart=/usr/local/bin/process-incoming.sh
```

```bash
systemctl enable --now incoming.path
# Drop a *.tar.gz in /srv/incoming → process-incoming.service runs.
```

### Path Directive Semantics

| Directive            | Triggers when...                                     |
|----------------------|------------------------------------------------------|
| `PathExists=`        | File/dir/symlink exists.                             |
| `PathExistsGlob=`    | Glob pattern matches at least one entry.             |
| `PathChanged=`       | Watched path changes (close after write, rename, attr). |
| `PathModified=`      | Watched path or any file under it is written.        |
| `DirectoryNotEmpty=` | Directory has at least one entry.                    |
| `MakeDirectory=true` | Create the watched dir if missing.                    |
| `DirectoryMode=0755` | Mode for `MakeDirectory`.                             |
| `TriggerLimitIntervalSec=`/`TriggerLimitBurst=` | Rate-limit re-triggers.   |

## Mount/Automount/Swap Units

### .mount Units

systemd auto-generates `.mount` units from `/etc/fstab` at boot. You rarely need to write them by hand. The unit name is the path with `/` replaced by `-` (then `.mount`): `/var/log` → `var-log.mount`.

```bash
# Translate a path to a unit name
systemd-escape -p --suffix=mount /var/log
# var-log.mount

# Hand-written mount unit
# /etc/systemd/system/srv-data.mount
[Unit]
Description=Data partition

[Mount]
What=/dev/disk/by-uuid/12345678-1234-1234-1234-123456789012
Where=/srv/data
Type=ext4
Options=defaults,noatime
TimeoutSec=60s

[Install]
WantedBy=local-fs.target
```

### .automount Units

Lazy mounts: the kernel's autofs mounts the corresponding `.mount` only when something accesses the path. Useful for slow filesystems (NFS over WAN, encrypted volumes) and rarely-used mounts.

```bash
# /etc/systemd/system/mnt-archive.automount
[Unit]
Description=NFS archive mount on first access

[Automount]
Where=/mnt/archive
TimeoutIdleSec=600                  # unmount after 10 min idle

[Install]
WantedBy=multi-user.target

# /etc/systemd/system/mnt-archive.mount
[Mount]
What=nfs.example.com:/archive
Where=/mnt/archive
Type=nfs
Options=ro,soft,intr
```

```bash
systemctl enable --now mnt-archive.automount
# Accessing /mnt/archive triggers the mount; idle for 10 min unmounts.
```

### .swap Units

Auto-generated from `/etc/fstab`. To enable a swap file:

```bash
# /etc/systemd/system/swapfile.swap
[Unit]
Description=8G swap file

[Swap]
What=/swapfile
Priority=10

[Install]
WantedBy=swap.target
```

```bash
fallocate -l 8G /swapfile
chmod 600 /swapfile
mkswap /swapfile
systemctl daemon-reload
systemctl enable --now swapfile.swap
```

## Device Units

`.device` units are auto-generated by udev for every device with `SYSTEMD_WANTS=` or that something else `BindsTo=`. You almost never write these by hand. Their utility is in **ordering**: `After=sys-subsystem-net-devices-eth0.device` waits for the interface to exist before starting your service.

```bash
# List currently active device units
systemctl list-units --type=device

# Show the device unit name for /dev/sdb1
systemd-escape -p --suffix=device /dev/sdb1
# dev-sdb1.device

# Wait for an interface in a service:
[Unit]
Wants=sys-subsystem-net-devices-eth0.device
After=sys-subsystem-net-devices-eth0.device
```

## Scope and Slice

### .scope Units

A scope groups already-running processes (started outside systemd) into a managed cgroup. `systemd-run` and `machinectl` create transient scopes. Login sessions are scopes (`session-c1.scope`).

```bash
# Wrap a long-running command in a transient scope with resource limits
systemd-run --scope --slice=batch.slice -p MemoryMax=2G \
  /usr/local/bin/heavy-job

# Show all current scopes
systemctl list-units --type=scope
```

### .slice Units

A slice is a node in the cgroup tree — purely a grouping/resource-limiting boundary. Default top-level slices:

| Slice               | Contains                                              |
|---------------------|-------------------------------------------------------|
| `system.slice`      | All system services (`*.service`).                    |
| `user.slice`        | All user sessions, broken into `user-UID.slice`.      |
| `machine.slice`     | All machinectl-managed containers/VMs.                |
| `init.scope`        | systemd itself + early-boot scopes.                   |

```bash
# Show the slice/service tree
systemd-cgls

# Define a custom slice for batch jobs
# /etc/systemd/system/batch.slice
[Unit]
Description=Batch jobs slice

[Slice]
CPUWeight=20
MemoryMax=4G
TasksMax=200
```

```bash
systemctl daemon-reload
systemctl set-property batch.slice CPUQuota=100%

# Start a service in this slice
# In the .service:
[Service]
Slice=batch.slice
```

### systemctl set-property (runtime tuning)

```bash
# Apply persistently (writes drop-in under /etc/systemd/system.control/)
systemctl set-property nginx.service MemoryMax=1G CPUQuota=50%

# Apply just for this boot (--runtime)
systemctl set-property --runtime nginx.service IOWeight=200

# View the effective property
systemctl show -p MemoryMax nginx.service
```

## Drop-Ins

Drop-ins are the canonical way to customize a vendor unit without editing it. systemd merges any number of `*.conf` snippets in priority-ordered directories on top of the base unit.

### Drop-In Directories (priority order)

```
/etc/systemd/system/<unit>.d/*.conf       admin overrides (highest)
/run/systemd/system/<unit>.d/*.conf       runtime
/usr/lib/systemd/system/<unit>.d/*.conf   distro defaults

# Type-wide drop-ins (apply to all units of a type):
/etc/systemd/system/service.d/*.conf      apply to all services
/etc/systemd/system/timer.d/*.conf        apply to all timers
```

### Edit a Drop-In

```bash
# Open the drop-in editor (creates dir + file as needed)
sudo systemctl edit nginx.service
# This opens /etc/systemd/system/nginx.service.d/override.conf in $EDITOR.
# Reload-and-restart guidance is printed on save.

# Replace the entire vendor unit (full edit)
sudo systemctl edit --full nginx.service

# Just rename the file you'd create instead of opening editor:
sudo systemctl edit --drop-in=mem-limit.conf nginx.service
```

### Common Drop-In Patterns

```bash
# /etc/systemd/system/nginx.service.d/override.conf
# Override Restart policy and add memory limit
[Service]
Restart=always
RestartSec=2s
MemoryMax=512M
```

```bash
# Append to ExecStart (note: must clear with empty line first)
[Service]
ExecStart=
ExecStart=/usr/sbin/nginx -g 'daemon off;' -c /etc/nginx/custom.conf
```

```bash
# Append an environment variable
[Service]
Environment="DEBUG=1"
```

### View Effective Configuration

```bash
# Show the merged result of vendor + all drop-ins
systemctl cat nginx.service

# Show the original vendor unit only
cat /usr/lib/systemd/system/nginx.service

# List all drop-ins applied
systemctl status nginx.service | grep Drop-In
```

## systemctl — Common Commands

```bash
# Lifecycle
systemctl start   <unit>
systemctl stop    <unit>
systemctl restart <unit>
systemctl reload  <unit>            # send SIGHUP / call ExecReload=

# Boot-time enable
systemctl enable      <unit>        # create symlinks per [Install]
systemctl disable     <unit>        # remove [Install] symlinks
systemctl enable --now <unit>       # enable + start
systemctl disable --now <unit>      # disable + stop
systemctl mask        <unit>        # symlink to /dev/null — cannot start
systemctl unmask      <unit>

# Inspect
systemctl status      <unit>        # state + recent journal lines
systemctl is-active   <unit>        # exit 0 if active
systemctl is-enabled  <unit>        # exit 0 if enabled
systemctl is-failed   <unit>        # exit 0 if failed
systemctl is-system-running         # initializing/starting/running/degraded/maintenance/stopping/offline

# Listings
systemctl list-units                # currently loaded units
systemctl list-units --all          # include inactive
systemctl list-units --failed       # only failed
systemctl list-units --type=service
systemctl list-units --state=failed
systemctl list-unit-files           # all unit files (loaded or not)
systemctl list-jobs                 # in-flight jobs
systemctl list-dependencies <unit>
systemctl list-timers
systemctl list-sockets
systemctl list-machines             # virtual machines / containers via systemd-machined

# Edit / view
systemctl cat   <unit>              # full unit + drop-ins
systemctl show  <unit>              # all properties
systemctl show -p ExecStart -p Restart <unit>
systemctl edit  <unit>              # drop-in
systemctl edit --full <unit>        # full replacement

# Reload systemd's own config
systemctl daemon-reload             # after editing units
systemctl daemon-reexec             # re-exec PID 1 (rare; for systemd upgrades)
```

## systemctl — Less-Common Useful Commands

```bash
# Restart only if running (no-op if stopped)
systemctl try-restart   <unit>
systemctl reload-or-restart <unit>  # reload if possible, else restart
systemctl try-reload-or-restart <unit>
systemctl condrestart   <unit>      # alias

# Forceful stop
systemctl kill <unit>                       # SIGTERM whole cgroup
systemctl kill --signal=SIGKILL <unit>
systemctl kill -s SIGUSR1 --kill-who=main <unit>

# cgroup-v2 freezer (v243+)
systemctl freeze <unit>             # SIGSTOP all processes in cgroup
systemctl thaw   <unit>

# Reset failed-state (clear "failed", allow restart)
systemctl reset-failed
systemctl reset-failed nginx.service

# Garbage collect a oneshot's leftover runtime state
systemctl clean <unit>              # state=runtime/state/cache/logs/configuration/all (v243+)

# Manager environment block (passed to all spawned children)
systemctl show-environment
systemctl set-environment FOO=bar BAZ=qux
systemctl unset-environment FOO
systemctl import-environment HTTPS_PROXY    # import from caller's env

# Presets (distro-defined enable/disable defaults)
systemctl preset      <unit>        # apply preset for one unit
systemctl preset-all                # apply all presets (dangerous)

# Job control
systemctl list-jobs
systemctl cancel <job-id>

# Stop everything cleanly and shut down
systemctl halt
systemctl poweroff
systemctl reboot
systemctl reboot --boot-loader-menu=20s     # reboot to firmware menu (v240+)
systemctl reboot --firmware-setup           # reboot directly to UEFI setup
systemctl suspend
systemctl hibernate
systemctl hybrid-sleep
systemctl suspend-then-hibernate            # sleep, then hibernate later
systemctl rescue                            # isolate rescue.target
systemctl emergency                         # isolate emergency.target

# Switch root (initrd → real root)
systemctl switch-root /sysroot

# Filter list-units by state
systemctl list-units --state=active
systemctl list-units --state=failed,activating

# Show pretty info via D-Bus
systemctl show -p MainPID,ActiveState,SubState,Result,ExecMainPID,LoadState <unit>
```

## journalctl — Reading Logs

journald is systemd's logging daemon. Logs are structured key-value records, indexed and queryable.

```bash
# Per-unit logs
journalctl -u nginx.service
journalctl -u nginx.service -u redis.service

# Follow (tail -f)
journalctl -u nginx.service -f

# Reverse chronological
journalctl -u nginx.service -r

# Jump to end
journalctl -u nginx.service -e

# Last N lines
journalctl -u nginx.service -n 100

# Kernel messages only (dmesg replacement)
journalctl -k
journalctl --dmesg

# Boots
journalctl -b                       # current boot
journalctl -b -1                    # previous boot
journalctl -b -2                    # two boots ago
journalctl --list-boots             # show all boots with IDs

# Time ranges
journalctl --since "2 hours ago"
journalctl --since today
journalctl --since yesterday
journalctl --since "2025-04-25 00:00:00" --until "2025-04-25 12:00:00"
journalctl --since "10 min ago" --until "5 min ago"

# Priority filter (0-7 or names)
journalctl -p err                   # err and above
journalctl -p warning..err          # range
# Levels: 0 emerg, 1 alert, 2 crit, 3 err, 4 warning, 5 notice, 6 info, 7 debug

# Pattern match
journalctl -u nginx.service --grep "404"
journalctl --grep "fatal" --case-sensitive=false

# Output format
journalctl -u nginx -o short        # default
journalctl -u nginx -o short-iso    # ISO timestamps
journalctl -u nginx -o cat          # message only
journalctl -u nginx -o verbose      # all fields
journalctl -u nginx -o json         # one JSON per line
journalctl -u nginx -o json-pretty
journalctl -u nginx -o json-seq     # RFC 7464 sequence
journalctl -u nginx -o export       # binary-safe export

# Pick fields
journalctl -u nginx --output-fields=MESSAGE,_PID,_COMM -o json

# By PID, UID, executable, hostname
journalctl _PID=1234
journalctl _UID=1000
journalctl _COMM=sshd
journalctl _SYSTEMD_UNIT=nginx.service
journalctl _HOSTNAME=db1
journalctl /usr/bin/myapp           # by path

# By transport (kernel/syslog/journal/stdout/audit)
journalctl _TRANSPORT=kernel

# User journal (your own user instance)
journalctl --user
journalctl --user -u syncthing.service

# No pager (script-friendly)
journalctl -u nginx --no-pager

# Disk usage
journalctl --disk-usage

# Vacuum (manually clean up)
journalctl --vacuum-size=500M
journalctl --vacuum-time=2weeks
journalctl --vacuum-files=10

# Verify journal integrity
journalctl --verify
```

## journalctl — Persistence and Storage

By default, the journal is **volatile** (in `/run/log/journal/`) and lost at reboot. Persistent storage requires creating `/var/log/journal/`.

```bash
# Create persistent journal
sudo mkdir -p /var/log/journal
sudo systemd-tmpfiles --create --prefix /var/log/journal
# OR manually:
sudo chown root:systemd-journal /var/log/journal
sudo chmod 2755 /var/log/journal
sudo systemctl restart systemd-journald
```

### journald.conf

```bash
# /etc/systemd/journald.conf
[Journal]
Storage=persistent              # persistent | volatile | auto | none
Compress=yes
Seal=yes                        # FSS forward-secure sealing (v189+)
SplitMode=uid                   # journal per-uid | none (single)
RateLimitIntervalSec=30s
RateLimitBurst=10000
SystemMaxUse=2G                 # max disk for /var/log/journal
SystemKeepFree=4G               # always keep this much free on the partition
SystemMaxFileSize=128M          # max per-file size
SystemMaxFiles=100              # max number of journal files
RuntimeMaxUse=200M              # same, for /run
RuntimeKeepFree=400M
RuntimeMaxFileSize=64M
MaxRetentionSec=1month          # delete entries older than this
MaxFileSec=1week                # rotate file every week
ForwardToSyslog=no
ForwardToKMsg=no
ForwardToConsole=no
ForwardToWall=yes
TTYPath=/dev/console
MaxLevelStore=debug
MaxLevelSyslog=debug
MaxLevelKMsg=notice
MaxLevelConsole=info
MaxLevelWall=emerg
LineMax=48K
ReadKMsg=yes
Audit=yes
```

```bash
# Apply changes
sudo systemctl restart systemd-journald
```

### Drop-In for journald

```bash
# /etc/systemd/journald.conf.d/size.conf
[Journal]
SystemMaxUse=500M
MaxRetentionSec=2weeks
```

## Logging — Standard Output / Error

The `StandardInput=`, `StandardOutput=`, and `StandardError=` directives in `[Service]` route fds 0/1/2.

| Value             | Behaviour                                                           |
|-------------------|---------------------------------------------------------------------|
| `inherit`         | Inherit from systemd (usually `/dev/null`).                         |
| `null`            | Connect to `/dev/null`.                                              |
| `tty`             | Connect to a TTY (uses `TTYPath=`).                                  |
| `tty-force`       | Like `tty` but disconnect any other process on it.                   |
| `tty-fail`        | Like `tty`, but fail if TTY is taken.                                |
| `journal`         | Connect to journald (default for `Output`/`Error` since v183+).      |
| `syslog`          | Connect to syslog.                                                    |
| `kmsg`            | Connect to kernel ring buffer.                                       |
| `journal+console` | journald + the console.                                              |
| `syslog+console`  | syslog + the console.                                                |
| `kmsg+console`    | kmsg + the console.                                                  |
| `file:/path`      | Open file (truncating); v236+.                                       |
| `append:/path`    | Open file in append mode; v240+.                                     |
| `truncate:/path`  | Open file truncating; v248+.                                         |
| `socket`          | Inherit from socket (only valid via socket activation).              |
| `fd:NAME`         | Inherit named file descriptor (v246+, with `OpenFile=`/socket FDs).  |
| `data`            | Read inline data (`StandardInputData=` / `StandardInputText=`).      |

```bash
# Send all stderr to a custom log file
[Service]
StandardOutput=journal
StandardError=append:/var/log/myapp/errors.log

# Per-line tagging in journal:
SyslogIdentifier=myapp
SyslogLevelPrefix=true              # service can prefix lines with <3>err <6>info etc.
```

## Cgroup Integration

Every service is in its own cgroup, named after the unit. Modern systems use cgroup-v2 unified hierarchy at `/sys/fs/cgroup/`.

```bash
# Show the cgroup tree
systemd-cgls

# Live top-style view
systemd-cgtop

# Find what cgroup a PID is in
cat /proc/1234/cgroup
# 0::/system.slice/nginx.service

# Inspect cgroup-v2 controllers
cat /sys/fs/cgroup/system.slice/nginx.service/cgroup.controllers
cat /sys/fs/cgroup/system.slice/nginx.service/memory.current
cat /sys/fs/cgroup/system.slice/nginx.service/cpu.stat
```

### Resource Control Directives

```bash
# CPU
CPUWeight=200                    # 1-10000, default 100; relative weight (cgroup-v2)
CPUQuota=50%                     # at most 0.5 CPU
CPUQuota=200%                    # at most 2 CPUs
CPUQuotaPeriodSec=100ms          # window for the quota
AllowedCPUs=0-3                  # cpuset (v244+)
AllowedMemoryNodes=0             # NUMA nodes
CPUAffinity=0 1 4-7              # taskset-style (v243+)

# Memory
MemoryMin=100M                   # never reclaim below this
MemoryLow=200M                   # soft protection from reclaim
MemoryHigh=1500M                 # throttle when exceeded
MemoryMax=2G                     # OOM-kill if exceeded
MemorySwapMax=0                  # disable swap (v232+)
MemoryZSwapMax=                  # zswap budget (v253+)

# I/O (cgroup-v2 io controller)
IOWeight=100                     # 1-10000, default 100
StartupIOWeight=200              # weight only during startup
IOReadBandwidthMax=/dev/sda 50M
IOWriteBandwidthMax=/dev/sda 50M
IOReadIOPSMax=/dev/sda 1000
IOWriteIOPSMax=/dev/sda 1000
IODeviceLatencyTargetSec=/dev/sda 50ms

# Tasks (process+thread cap)
TasksMax=512
TasksMax=infinity

# Apply at runtime (writes /etc/systemd/system.control/<unit>.d/)
systemctl set-property nginx.service MemoryMax=1G CPUWeight=500
systemctl set-property --runtime nginx.service IOWeight=200
```

## Security Hardening

systemd ships a remarkable amount of sandboxing for free. Use `systemd-analyze security` to score a unit on a 0-10 "exposure" scale and get a checklist.

```bash
# Score a single unit
systemd-analyze security nginx.service

# Score everything
systemd-analyze security

# Quick score only
systemd-analyze security --no-pager nginx.service | tail -5
```

### The Hardening Checklist

```bash
[Service]
# Identity
User=myapp
Group=myapp
DynamicUser=true                 # v235+: ephemeral UID, automatic /var/lib

# Privilege escalation
NoNewPrivileges=true
LockPersonality=true             # v235+
RestrictRealtime=true
RestrictSUIDSGID=true            # v242+

# Capabilities — drop everything, add back only what's needed
CapabilityBoundingSet=
AmbientCapabilities=
# To allow binding < 1024:
#   CapabilityBoundingSet=CAP_NET_BIND_SERVICE
#   AmbientCapabilities=CAP_NET_BIND_SERVICE

# Memory protection
MemoryDenyWriteExecute=true      # block W^X bypass

# Filesystem
ProtectSystem=strict             # /usr /boot /etc read-only
ProtectHome=true                 # no /home, /root, /run/user
PrivateTmp=true                  # private /tmp /var/tmp
PrivateDevices=true              # /dev limited to safe nodes
ReadWritePaths=/var/lib/myapp /var/log/myapp
TemporaryFileSystem=/var:ro      # any write below /var fails (v238+)
ProtectKernelTunables=true       # /proc/sys, /sys read-only
ProtectKernelModules=true        # cannot load modules
ProtectKernelLogs=true           # v244+
ProtectControlGroups=true
ProtectClock=true                # v246+
ProtectHostname=true             # v242+
ProtectProc=invisible            # v247+
ProcSubset=pid                   # v247+

# Namespaces
PrivateNetwork=false             # set true if no network needed
PrivateUsers=true                # private UID namespace
PrivateMounts=true               # v239+
PrivateIPC=true                  # v248+
RestrictNamespaces=true          # block unshare(2) namespace creation
RemoveIPC=true                   # clean SysV IPC at exit

# Network
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
SocketBindAllow=tcp:80           # eBPF-based bind allowlist (v249+)
SocketBindDeny=any
IPAddressAllow=10.0.0.0/8        # eBPF-based egress allow (v235+)
IPAddressDeny=any

# System call filtering
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources @debug @cpu-emulation @obsolete
SystemCallErrorNumber=EPERM
SystemCallArchitectures=native

# Misc
KeyringMode=private
ProtectKeyring=true
UMask=0077
```

### What Each Pillar Buys You

| Directive               | Blocks                                                    |
|-------------------------|-----------------------------------------------------------|
| `NoNewPrivileges=true`  | setuid/setgid escalation; `execve` cannot raise privs.    |
| `PrivateTmp=true`       | `/tmp` symlink races; cross-service tmp file leakage.     |
| `ProtectSystem=strict`  | Writes to `/usr`, `/boot`, `/etc` (entire root except `/dev`, `/proc`, `/sys`, `/tmp`, `/var`, plus `ReadWritePaths=`). |
| `ProtectHome=true`      | Reading or writing `/home`, `/root`, `/run/user`.         |
| `PrivateDevices=true`   | Access to disks, USB, GPUs, etc.                          |
| `PrivateNetwork=true`   | All networking — service is in an empty net namespace.    |
| `RestrictNamespaces=true` | `unshare(CLONE_*)` namespace escapes.                   |
| `SystemCallFilter=`     | seccomp-bpf — block syscalls outside allowlist.           |
| `MemoryDenyWriteExecute=true` | `mmap PROT_WRITE|PROT_EXEC` JIT-spray attacks.      |
| `LockPersonality=true`  | `personality(2)` switching to legacy ABIs.                |
| `CapabilityBoundingSet=`| Raising any capability not in the set.                    |

## SystemCallFilter

`SystemCallFilter=` installs a seccomp-bpf filter that allows only listed syscalls. Predefined "syscall sets" (prefix `@`) cover common needs.

```bash
# Common predefined sets:
@system-service        # safe baseline for typical services
@basic-io              # read, write, close, openat, lseek, dup, etc.
@network-io            # socket, connect, accept, send, recv, etc.
@file-system           # open, stat, mkdir, unlink, etc.
@io-event              # epoll, poll, select, kqueue
@ipc                   # shm, msg, sem, futex
@process               # fork, clone, execve, kill, signal
@signal                # rt_sig*, sigaction, sigprocmask
@timer                 # nanosleep, clock_*, timerfd
@memlock               # mlock, mlockall

# Sets that should usually be DENIED via ~ (negation):
@privileged            # setuid, capset, syslog, mount, etc.
@resources             # setpriority, sched_setscheduler, ioprio_set, prlimit
@debug                 # ptrace, kcmp, perf_event_open
@module                # init_module, finit_module, delete_module
@reboot                # reboot, kexec_load
@swap                  # swapon, swapoff
@mount                 # mount, umount2, pivot_root
@raw-io                # iopl, ioperm
@cpu-emulation         # modify_ldt, vm86, vm86old
@obsolete              # syslog, ustat, sysfs, ...
@aio                   # io_*
@chown                 # chown, fchown, lchown
@clock                 # adjtimex, settimeofday
@keyring               # add_key, request_key, keyctl

# Combine: allow @system-service, deny @resources, @privileged
SystemCallFilter=@system-service
SystemCallFilter=~@resources @privileged @obsolete

# Custom syscall: allow openat but block open
SystemCallFilter=openat
SystemCallFilter=~open

# What happens on a denied syscall?
SystemCallErrorNumber=EPERM        # default — return -EPERM
SystemCallErrorNumber=kill         # SIGSYS-kill the process
```

```bash
# Architecture restriction (block 32-on-64 syscall surface)
SystemCallArchitectures=native
```

```bash
# Validate a filter against a binary
systemd-analyze syscall-filter @system-service
systemd-analyze syscall-filter --quiet @privileged | wc -l
```

## Capabilities

Linux capabilities partition root's privileges into ~40 independent flags (`CAP_NET_BIND_SERVICE`, `CAP_NET_ADMIN`, `CAP_SYS_ADMIN`, etc.). systemd lets you constrain which a service can hold.

```bash
# Drop everything, then re-add what's needed
CapabilityBoundingSet=                       # empty = drop all
CapabilityBoundingSet=CAP_NET_BIND_SERVICE   # only this one

# Ambient = capabilities granted *to the process at exec*
# Required for non-root services that need privileges (v229+)
AmbientCapabilities=CAP_NET_BIND_SERVICE
```

### Common Capabilities Cheat Sheet

| Capability                 | Allows                                                 |
|----------------------------|--------------------------------------------------------|
| `CAP_NET_BIND_SERVICE`     | Bind to ports < 1024.                                  |
| `CAP_NET_ADMIN`            | Network config, iptables, routing.                     |
| `CAP_NET_RAW`              | Raw sockets, ICMP (ping), packet capture.              |
| `CAP_SYS_ADMIN`            | "The new root" — many privileged ops.                  |
| `CAP_SYS_TIME`             | Set system clock.                                      |
| `CAP_SYS_PTRACE`           | ptrace any process.                                    |
| `CAP_SYS_NICE`             | Raise nice priority, set sched policy.                 |
| `CAP_SYS_RESOURCE`         | Override rlimits.                                      |
| `CAP_SYS_CHROOT`           | Call `chroot(2)`.                                      |
| `CAP_DAC_OVERRIDE`         | Bypass file mode checks (read/write any).              |
| `CAP_DAC_READ_SEARCH`      | Bypass file read/search checks.                        |
| `CAP_FOWNER`               | Bypass file ownership checks.                          |
| `CAP_KILL`                 | Send signals to any process.                           |
| `CAP_BPF`                  | Load BPF programs (v5.8+ split from `CAP_SYS_ADMIN`).  |
| `CAP_PERFMON`              | perf_event_open (v5.8+).                               |

### Non-Root Bind to <1024

```bash
# Run as user 'mywebd', let it bind port 80
[Service]
User=mywebd
Group=mywebd
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ExecStart=/usr/local/bin/myweb --listen :80
```

## Socket Activation Pattern

The canonical "lazy-start a service when first connection arrives" pattern. Especially useful for D-Bus services, dev-only services, and rare administrative interfaces.

```bash
# /etc/systemd/system/example.socket
[Unit]
Description=Example HTTP listener

[Socket]
ListenStream=80
Accept=no                       # service handles all conns

[Install]
WantedBy=sockets.target
```

```bash
# /etc/systemd/system/example.service
[Unit]
Description=Example HTTP server
Requires=example.socket

[Service]
Type=simple
ExecStart=/usr/local/bin/myserver --systemd-socket
NonBlocking=true                # set O_NONBLOCK on inherited fds
```

```bash
sudo systemctl enable --now example.socket
# 'systemctl status example.service' → inactive (dead)
# First curl http://host:80 → service starts, handles connection.
```

In your service code:

```c
/* libsystemd */
#include <systemd/sd-daemon.h>
int n = sd_listen_fds(0);          /* number of inherited fds */
if (n != 1) { /* error */ }
int fd = SD_LISTEN_FDS_START + 0;  /* fd 3 by convention */
/* fd is already bound and listening; just accept(2) */
```

```go
// Go (github.com/coreos/go-systemd/v22/activation)
listeners, _ := activation.Listeners()
http.Serve(listeners[0], handler)
```

```bash
# In a shell-script service: $LISTEN_FDS, $LISTEN_PID, $LISTEN_FDNAMES
[Service]
ExecStart=/bin/sh -c 'echo got $LISTEN_FDS fds, pid $LISTEN_PID; cat <&3'
```

## Timer-Driven Backup Pattern

The canonical cron replacement.

```bash
# /etc/systemd/system/backup.service
[Unit]
Description=Daily database backup
Wants=postgresql.service
After=postgresql.service network-online.target

[Service]
Type=oneshot
User=backup
Group=backup
ExecStart=/usr/local/bin/backup-db.sh
Nice=19
IOSchedulingClass=idle
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/srv/backups
NoNewPrivileges=true
StandardOutput=journal
StandardError=journal
SyslogIdentifier=backup
```

```bash
# /etc/systemd/system/backup.timer
[Unit]
Description=Daily database backup timer

[Timer]
OnCalendar=*-*-* 02:30:00
Persistent=true                  # run on next boot if missed
RandomizedDelaySec=15min
AccuracySec=1min
Unit=backup.service

[Install]
WantedBy=timers.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now backup.timer

# Inspect
systemctl list-timers backup.timer
journalctl -u backup.service --since "1 week ago"

# Run on demand
sudo systemctl start backup.service
```

### Why this beats cron

- **Logs**: structured journal, `journalctl -u backup`.
- **Dependencies**: `After=postgresql.service network-online.target` — won't run before deps.
- **Hardening**: full `[Service]` sandbox.
- **Catch-up**: `Persistent=true` runs missed jobs on boot.
- **Test runs**: `systemctl start backup.service` runs identically to scheduled invocation.
- **Per-user**: `~/.config/systemd/user/X.timer` is immediate; cron-as-user is awkward.
- **Resource limits**: `Nice=`, `IOSchedulingClass=`, `MemoryMax=` per-job.

## User Services

`systemctl --user` manages a per-user systemd instance. Unit files live under `~/.config/systemd/user/`. The user instance is started by `systemd-logind` when you log in (via `pam_systemd`).

```bash
# Where to put your unit files
mkdir -p ~/.config/systemd/user

# Example: ~/.config/systemd/user/syncthing.service
[Unit]
Description=Syncthing
After=network-online.target

[Service]
ExecStart=/usr/bin/syncthing --no-browser --no-restart --logflags=0
Restart=on-failure

[Install]
WantedBy=default.target

# Enable + start
systemctl --user daemon-reload
systemctl --user enable --now syncthing.service

# Check
systemctl --user status syncthing.service
journalctl --user -u syncthing.service
```

### XDG_RUNTIME_DIR

User services see `$XDG_RUNTIME_DIR=/run/user/$UID` — a private tmpfs cleaned at logout. Use it for sockets, locks, etc.

### loginctl enable-linger (services that survive logout)

By default the user instance dies when your last session closes. To keep it running (e.g., for a server that you SSH into), enable lingering:

```bash
# As that user
loginctl enable-linger
loginctl enable-linger alice         # as root, for another user

# Verify
loginctl show-user alice | grep Linger
# Linger=yes

# Disable
loginctl disable-linger alice
```

```bash
# When linger is enabled, the user manager starts at boot,
# and user services with WantedBy=default.target start.
```

### Common User Service Use Cases

```bash
# Music daemon (mpd) per user
~/.config/systemd/user/mpd.service

# Background sync (rclone, syncthing, kdeconnectd)
# Notifications via dunst
# Per-user wireguard tunnels
# Personal cron: ~/.config/systemd/user/*.{timer,service}
```

## systemd-tmpfiles

Manages volatile and temporary files declaratively. Replaces `/etc/init.d/*-tmpfiles` ad-hoc scripts and lots of mkdir-on-boot logic.

```bash
# /etc/tmpfiles.d/myapp.conf
# Type Path                        Mode User  Group Age Argument
d /var/lib/myapp                  0750 myapp myapp -   -
d /var/log/myapp                  0750 myapp myapp 30d -
f /var/log/myapp/app.log          0640 myapp myapp -   -
L /var/lib/myapp/cache            -    -     -     -   /var/cache/myapp
r! /run/myapp                     -    -     -     -   -
e /tmp/myapp                      -    -     -     1h  -
```

### Type Field

| Type | Meaning                                                                |
|------|------------------------------------------------------------------------|
| `f`  | Create file (don't overwrite). With `+`: also write `Argument` content. |
| `f+` | Create or overwrite file with `Argument` content.                       |
| `w`  | Write argument to existing file (don't create).                         |
| `w+` | Append argument to file.                                                 |
| `d`  | Create directory.                                                        |
| `D`  | Like `d` but empty contents on creation.                                 |
| `e`  | Adjust attributes of existing dir, clean entries older than `Age`.       |
| `v`  | Subvolume (btrfs) — fall back to `d` on non-btrfs.                       |
| `q`  | Subvolume + quota.                                                       |
| `Q`  | Subvolume + parent-quota.                                                |
| `p`  | Create FIFO.                                                              |
| `L`  | Create symlink (don't overwrite).                                        |
| `L+` | Force symlink (overwrite).                                               |
| `c`  | Create character device.                                                 |
| `b`  | Create block device.                                                     |
| `C`  | Recursive copy (don't overwrite).                                        |
| `x`  | Ignore (exclude from clean).                                             |
| `X`  | Ignore (exclude self and contents).                                      |
| `r`  | Remove file/directory if exists. `r!` = at boot only.                    |
| `R`  | Recursive remove.                                                        |
| `z`  | Adjust ACLs (non-recursive).                                             |
| `Z`  | Adjust ACLs (recursive).                                                 |
| `t`  | Set extended attributes.                                                 |
| `T`  | Set xattrs recursively.                                                  |
| `h`  | Set file attributes (chattr).                                            |
| `H`  | Set attributes recursively.                                              |
| `a`  | Set POSIX ACLs.                                                           |
| `A`  | Set POSIX ACLs recursively.                                              |

### Apply Now

```bash
# Apply at boot (handled by systemd-tmpfiles-setup.service)
sudo systemd-tmpfiles --create

# Apply only "boot" entries (those without "!")
sudo systemd-tmpfiles --create --boot

# Apply a specific config
sudo systemd-tmpfiles --create /etc/tmpfiles.d/myapp.conf

# Run cleanup (apply Age=)
sudo systemd-tmpfiles --clean

# Show what would happen
sudo systemd-tmpfiles --create --dry-run /etc/tmpfiles.d/myapp.conf

# Remove (process 'r' / 'R' lines)
sudo systemd-tmpfiles --remove
```

### File-with-Content

```bash
# /etc/tmpfiles.d/sysctl.conf
# Lay down a default sysctl when missing
f /etc/sysctl.d/99-myapp.conf 0644 root root - "vm.swappiness = 10\n"
```

## systemd-sysusers

Declarative system-user creation. Replaces `useradd` calls in package post-install scripts. `systemd-sysusers.service` runs at boot and applies all `/etc/sysusers.d/*.conf` plus `/usr/lib/sysusers.d/*.conf`.

```bash
# /etc/sysusers.d/myapp.conf
# Type Name        ID                  GECOS                 Home directory  Shell
u     myapp       -                    "MyApp service user"  /var/lib/myapp  /usr/sbin/nologin
g     myapp       -                    -
m     myapp       systemd-journal      -                     -               -
```

### Types

| Type | Purpose                                                            |
|------|--------------------------------------------------------------------|
| `u`  | Create user (and matching group).                                  |
| `u!` | Create user but mark account locked (no password).                 |
| `g`  | Create group.                                                      |
| `m`  | Add existing user to existing group.                               |
| `r`  | Reserve a UID/GID range for static-allocation use.                 |

### ID Field

```bash
# - means "any free ID"
u myapp -
# Specific numeric UID
u myapp 990
# Path-based: assign UID matching ownership of file
u myapp /usr/bin/myapp
# Range
r - 1000-1999
```

### Apply Now

```bash
# Apply on boot (sysusers.service) or manually
sudo systemd-sysusers
sudo systemd-sysusers /etc/sysusers.d/myapp.conf

# Verify
getent passwd myapp
getent group  myapp
```

## systemd-resolved

Systemd's local stub DNS resolver. Listens on 127.0.0.53:53. Provides per-link DNS, DNSSEC validation, mDNS/LLMNR, DNS-over-TLS.

```bash
# Status
resolvectl status
resolvectl status eth0           # per-link

# Query
resolvectl query example.com
resolvectl query --type=AAAA example.com
resolvectl query --type=MX example.com
resolvectl query example.com --interface=eth0

# Show statistics
resolvectl statistics

# Flush cache
sudo resolvectl flush-caches

# DNSSEC
resolvectl dnssec eth0 yes       # require DNSSEC on this link

# Set DNS for a link
sudo resolvectl dns eth0 1.1.1.1 1.0.0.1
sudo resolvectl domain eth0 ~.    # use eth0 as catch-all

# Reset to systemd-networkd / NetworkManager defaults
sudo resolvectl revert eth0

# Restart the resolver
sudo systemctl restart systemd-resolved
```

### /etc/resolv.conf

```bash
# Recommended — symlink to the systemd-resolved stub
ls -l /etc/resolv.conf
# /etc/resolv.conf -> /run/systemd/resolve/stub-resolv.conf

# stub-resolv.conf points all queries at 127.0.0.53
cat /etc/resolv.conf
# nameserver 127.0.0.53
# options edns0 trust-ad
# search example.com

# To restore the symlink:
sudo ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
```

### resolved.conf

```bash
# /etc/systemd/resolved.conf
[Resolve]
DNS=1.1.1.1 1.0.0.1 9.9.9.9
FallbackDNS=8.8.8.8 8.8.4.4
Domains=~example.com
DNSSEC=yes
DNSOverTLS=opportunistic
MulticastDNS=yes
LLMNR=yes
Cache=yes
DNSStubListener=yes
ReadEtcHosts=yes
```

```bash
sudo systemctl restart systemd-resolved
```

## systemd-networkd

systemd's network manager — perfect for headless servers. Less suited to laptops (NetworkManager wins for roaming, VPN GUIs).

### Enable

```bash
sudo systemctl enable --now systemd-networkd
sudo systemctl enable --now systemd-resolved   # (recommended companion)
```

### Static IPv4 + IPv6 (.network)

```bash
# /etc/systemd/network/10-eth0.network
[Match]
Name=eth0

[Network]
Address=192.0.2.10/24
Address=2001:db8::10/64
Gateway=192.0.2.1
Gateway=2001:db8::1
DNS=1.1.1.1
DNS=2606:4700:4700::1111
Domains=example.com
IPv6AcceptRA=true
```

### DHCP

```bash
# /etc/systemd/network/20-eth0.network
[Match]
Name=eth0

[Network]
DHCP=yes

[DHCPv4]
UseDNS=true
UseRoutes=true
RouteMetric=10
```

### VLAN

```bash
# /etc/systemd/network/30-eth0.network
[Match]
Name=eth0

[Network]
VLAN=eth0.100

# /etc/systemd/network/31-eth0.100.netdev
[NetDev]
Name=eth0.100
Kind=vlan

[VLAN]
Id=100

# /etc/systemd/network/32-eth0.100.network
[Match]
Name=eth0.100

[Network]
Address=10.100.0.10/24
```

### Bridge

```bash
# /etc/systemd/network/40-br0.netdev
[NetDev]
Name=br0
Kind=bridge

# /etc/systemd/network/41-br0.network
[Match]
Name=br0

[Network]
DHCP=yes

# /etc/systemd/network/42-eth0-bridge.network
[Match]
Name=eth0

[Network]
Bridge=br0
```

### Apply

```bash
sudo systemctl restart systemd-networkd
networkctl status
networkctl status eth0
networkctl reload                # apply config changes (v244+)
```

## systemd-analyze

The diagnostic Swiss-army-knife.

```bash
# Total boot time
systemd-analyze
# Startup finished in 1.234s (kernel) + 5.678s (userspace) = 6.912s
# graphical.target reached after 4.321s in userspace

# Slowest units to start
systemd-analyze blame
# 2.345s NetworkManager-wait-online.service
# 1.234s systemd-journal-flush.service
# ...

# Critical path — the chain that gates the longest
systemd-analyze critical-chain
systemd-analyze critical-chain nginx.service

# SVG visualization of the boot
systemd-analyze plot > boot.svg
xdg-open boot.svg

# Score a unit on hardening (0-10 exposure score; lower = better)
systemd-analyze security nginx.service
systemd-analyze security                  # all running services

# Validate a unit file's syntax + semantics (without loading it)
systemd-analyze verify /etc/systemd/system/myapp.service

# Cat resolved unit (with all overrides merged)
systemd-analyze cat-config systemd/system.conf

# Calendar spec validation
systemd-analyze calendar "Mon..Fri 09:00"
systemd-analyze calendar --iterations=10 "*-*-* 0/4:00:00"

# Time spec validation
systemd-analyze timespan "2h 30min"
# → 9000000000 microseconds, 2h 30min

# Show the dependency graph (DOT format)
systemd-analyze dot multi-user.target | dot -Tsvg > deps.svg

# Inspect manager state
systemd-analyze dump | head -50
systemd-analyze dump > /tmp/state.txt

# Show condition results for a unit
systemd-analyze condition /etc/systemd/system/myapp.service

# Inspect environment passed to units
systemd-analyze environment

# Show seccomp filter expansion
systemd-analyze syscall-filter @system-service

# Inspect EXIT codes / signals
systemd-analyze exit-status FAILURE-CODE

# Capability inspection
systemd-analyze capability CAP_NET_BIND_SERVICE
```

## Common systemctl Errors and Fixes

These are the exact strings you will see and the canonical fix.

### "Failed to start X.service: Unit X.service not found"

```bash
# Cause: file created but daemon-reload not run, or wrong path
sudo systemctl daemon-reload

# Or: file is in /tmp/, ~/Downloads — must be under
#   /etc/systemd/system/  (system)
#   ~/.config/systemd/user/  (user)
ls /etc/systemd/system/myapp.service
```

### "X.service: Failed with result 'exit-code'"

```bash
# Cause: ExecStart exited non-zero
journalctl -u X.service -e --no-pager | tail -50

# Common follow-ups: missing binary, wrong WorkingDirectory, missing env file
systemctl status X.service          # shows last lines + exit code
```

### "X.service: Service hold-off time over, scheduling restart"

```bash
# Cause: Restart= is firing; check the underlying failure
journalctl -u X.service -e
# To prevent runaway loops:
#   StartLimitIntervalSec=300
#   StartLimitBurst=3
# Then `systemctl reset-failed X.service` once fixed.
```

### "X.service: Service failed because the control process exited with error code"

```bash
# Cause: ExecStartPre, ExecStartPost, or ExecStop returned non-zero
# Inspect the precise step:
systemctl show X.service -p ExecStartPre -p ExecStartPost -p ExecStop
journalctl -u X.service -e
```

### "Job for X.service failed because the control process exited with error code. See \"systemctl status X.service\" and \"journalctl -xeu X.service\" for details."

```bash
systemctl status X.service
journalctl -xeu X.service           # -x adds explanatory hints
```

### "Failed to enable unit: Unit X.service does not exist"

```bash
# Cause: the file is not in any of the search dirs, OR
# the .service has no [Install] section.
systemctl cat X.service
# If [Install] is missing, add e.g. WantedBy=multi-user.target

# If the file is a custom location, link it:
sudo systemctl link /opt/custom/myapp.service
```

### "Failed to enable unit: Refusing to operate on linked unit file"

```bash
# Cause: file in /etc/systemd/system/ is a symlink pointing outside.
# Solutions:
sudo cp /opt/custom/myapp.service /etc/systemd/system/myapp.service
sudo systemctl daemon-reload
sudo systemctl enable myapp.service
```

### "Failed to start X.service: Unit X.service is masked."

```bash
# Cause: 'systemctl mask' was used at some point.
sudo systemctl unmask X.service
sudo systemctl start  X.service
```

### "X.service is not a native service, redirecting to systemd-sysv-install."

```bash
# Cause: SysV-init script — systemd is delegating to chkconfig.
# Solution: package usually ships a real .service file; rebuild with
#   apt install --reinstall <package>
# Or write a native unit.
```

### "Failed to add dependency on X.target: No such file or directory"

```bash
# Cause: WantedBy= or RequiredBy= references a non-existent target
# Fix: use a built-in target name (multi-user.target, etc.)
```

### "Active: failed (Result: exit-code) since ..."

```bash
# Look up the last cause:
systemctl status X.service
journalctl -u X.service -p err -b
```

### "PIDFile= references path below legacy directory /var/run/, updating /var/run/X.pid → /run/X.pid"

```bash
# Cause: cosmetic warning. Replace /var/run/ with /run/ in the unit.
# (/var/run is a symlink to /run on systemd systems.)
```

### "(code=killed, signal=KILL)"

```bash
# Cause: SIGKILL — usually OOM-killed, or TimeoutStopSec hit.
# Confirm OOM:
journalctl -k | grep -iE "oom|killed process"
dmesg -T | grep -i oom
```

## Common Gotchas

Each gotcha shows the broken pattern first, then the fix.

### Forgetting daemon-reload

```bash
# bad — edited the unit, restarted the service, no effect
sudo $EDITOR /etc/systemd/system/myapp.service
sudo systemctl restart myapp.service
# Restarted: Yes. New config in use: NO.

# good
sudo $EDITOR /etc/systemd/system/myapp.service
sudo systemctl daemon-reload
sudo systemctl restart myapp.service
```

### Shell features in ExecStart=

```bash
# bad — || ; * & < > don't work; ExecStart is execve, not a shell
[Service]
ExecStart=/usr/bin/myapp || /usr/bin/myapp-fallback
ExecStart=cd /opt && ./run.sh

# good — wrap in sh -c, or write a wrapper script
[Service]
ExecStart=/bin/sh -c '/usr/bin/myapp || /usr/bin/myapp-fallback'
WorkingDirectory=/opt
ExecStart=/opt/run.sh
```

### PIDFile= without Type=forking

```bash
# bad — service shows "active (exited)" almost instantly
[Service]
Type=simple
PIDFile=/run/myapp.pid
ExecStart=/usr/sbin/myapp

# good — pick the matching Type= for your daemon
[Service]
Type=forking          # daemon double-forks
PIDFile=/run/myapp.pid
ExecStart=/usr/sbin/myapp

# OR if it stays in foreground:
[Service]
Type=simple
ExecStart=/usr/sbin/myapp --foreground
# (no PIDFile)
```

### Missing WantedBy=

```bash
# bad — 'systemctl enable' silently does nothing
[Unit]
Description=My App
[Service]
ExecStart=/opt/myapp/bin/run

# good — add an [Install] section
[Install]
WantedBy=multi-user.target
```

### Writing under ProtectSystem=strict without ReadWritePaths=

```bash
# bad — service starts, then "permission denied" on first write
[Service]
ProtectSystem=strict
ExecStart=/opt/myapp/bin/run --log /var/log/myapp/app.log

# good — explicitly whitelist the writable paths
[Service]
ProtectSystem=strict
ReadWritePaths=/var/log/myapp /var/lib/myapp
ExecStart=/opt/myapp/bin/run --log /var/log/myapp/app.log
```

### Restart=always loops on misconfigured service

```bash
# bad — service hits a config error, dies, restarts in 100ms, repeat
[Service]
Restart=always
ExecStart=/opt/myapp/bin/run

# good — exponential backoff + start-limit
[Service]
Restart=always
RestartSec=10s
StartLimitIntervalSec=300
StartLimitBurst=3
# After 3 restarts in 5 min, systemd gives up. Fix it then:
#   systemctl reset-failed myapp.service
#   systemctl start myapp.service
```

### Wants= without After=

```bash
# bad — myapp starts in parallel with postgres; race condition
[Unit]
Wants=postgresql.service

# good — pull AND order
[Unit]
Wants=postgresql.service
After=postgresql.service
```

### After=network.target when you need network up

```bash
# bad — network.target only means "config applied", not "interface up"
[Unit]
After=network.target

# good — wait for actual L3 connectivity
[Unit]
Wants=network-online.target
After=network-online.target
# Plus enable: systemctl enable systemd-networkd-wait-online.service
#         OR: systemctl enable NetworkManager-wait-online.service
```

### Using EnvironmentFile= with quotes

```bash
# bad — Bash-style export; systemd treats whole line as VAR=value
# /etc/myapp/env
export DATABASE_URL="postgres://..."

# good — KEY=VALUE, no shell, no export, no surrounding quotes needed
# /etc/myapp/env
DATABASE_URL=postgres://...
LOG_LEVEL=info
```

### Forgetting daemon-reexec after upgrading systemd

```bash
# bad — pacman/apt upgrade replaced /usr/lib/systemd/systemd
# but PID 1 is still the old binary
sudo pacman -Syu

# good — re-exec PID 1 against the new binary
sudo systemctl daemon-reexec
```

### Custom unit with executable bit not set

```bash
# bad — ExecStart points to a script without +x
chmod 644 /opt/myapp/start.sh  # oops

# good
chmod 755 /opt/myapp/start.sh
sudo systemctl daemon-reload
sudo systemctl start myapp.service
```

### User= with relative WorkingDirectory=

```bash
# bad — relative path; systemd starts you in /
[Service]
User=myapp
WorkingDirectory=opt/myapp

# good — absolute path (and ensure User= can access it)
[Service]
User=myapp
WorkingDirectory=/opt/myapp
```

### Using `systemctl status` exit code in scripts

```bash
# bad — `status` returns 0..3 with semantics other than "active"
if systemctl status myapp.service; then
  echo running
fi

# good — use is-active
if systemctl is-active --quiet myapp.service; then
  echo running
fi
```

### Hot-editing unit files in /usr/lib

```bash
# bad — package upgrade overwrites your changes
sudo $EDITOR /usr/lib/systemd/system/nginx.service

# good — drop-in
sudo systemctl edit nginx.service
# Or fully replace if you really want:
sudo systemctl edit --full nginx.service
```

### Type=oneshot without RemainAfterExit and dependents that need it "active"

```bash
# bad — script runs, exits, unit goes inactive, dependents see it as down
[Service]
Type=oneshot
ExecStart=/usr/local/bin/setup.sh

# good — keep "active" after exit so dependents stay activated
[Service]
Type=oneshot
RemainAfterExit=true
ExecStart=/usr/local/bin/setup.sh
```

### `systemctl edit` then file is empty

```bash
# bad — quit the editor on the empty template; file remains, but blank
# Result: drop-in exists, but does nothing
ls /etc/systemd/system/nginx.service.d/override.conf
# 0 bytes

# good — write the override or delete the empty file
sudo systemctl revert nginx.service
# (revert deletes admin overrides)
```

## Idioms

### Canonical Long-Running Service Template

```bash
# /etc/systemd/system/myapp.service
[Unit]
Description=MyApp
Documentation=https://example.com/myapp
After=network-online.target postgresql.service
Wants=network-online.target
Requires=postgresql.service

[Service]
Type=notify
NotifyAccess=main
User=myapp
Group=myapp
ExecStart=/usr/local/bin/myapp
ExecReload=/bin/kill -HUP $MAINPID
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=300
StartLimitBurst=3
TimeoutStartSec=60s
TimeoutStopSec=30s
WorkingDirectory=/var/lib/myapp
EnvironmentFile=-/etc/myapp/env
StandardOutput=journal
StandardError=journal
SyslogIdentifier=myapp

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/myapp /var/log/myapp
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
ProtectClock=true
ProtectHostname=true
LockPersonality=true
MemoryDenyWriteExecute=true
RestrictRealtime=true
RestrictSUIDSGID=true
RestrictNamespaces=true
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
RemoveIPC=true
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources @debug
SystemCallErrorNumber=EPERM
CapabilityBoundingSet=
LimitNOFILE=65536
MemoryMax=2G

[Install]
WantedBy=multi-user.target
```

### Drop-In Override Pattern

```bash
# Don't touch the vendor unit — drop-in instead
sudo systemctl edit nginx.service
# Then insert (in $EDITOR):
[Service]
LimitNOFILE=100000
Environment=DEBUG=1

# This creates /etc/systemd/system/nginx.service.d/override.conf
# Vendor file is untouched; package upgrades don't clobber.
```

### Cron Replacement

```bash
# foo.timer:  OnCalendar=daily  Persistent=true
# foo.service: Type=oneshot  ExecStart=/path/to/script.sh
# Enable: systemctl enable --now foo.timer
# Inspect: journalctl -u foo.service
```

### Lazy Service via Socket Activation

```bash
# foo.socket:  ListenStream=8080
# foo.service: ExecStart=/usr/bin/foo --systemd
# Enable: systemctl enable --now foo.socket
# First connect wakes the service; service can self-exit on idle.
```

### Pre-Deploy Hardening Review

```bash
# Always run before shipping a new service
sudo systemctl daemon-reload
systemd-analyze verify /etc/systemd/system/myapp.service
systemd-analyze security myapp.service
# Aim for "exposure" < 2.0 ("OK") on production daemons.
```

### Inspect a Failure Quickly

```bash
systemctl status myapp.service
journalctl -xeu myapp.service       # -x adds hints, -e jumps to end
journalctl -u myapp.service -p err -b
systemctl reset-failed myapp.service  # clear failed state
systemctl start myapp.service
```

### One-Shot via systemd-run

```bash
# Run any command as a transient service / scope, with full sandboxing
systemd-run --unit=mybackup --scope --slice=batch.slice \
  -p MemoryMax=1G -p Nice=19 \
  /usr/local/bin/backup.sh

# Schedule a one-off
systemd-run --on-active=10min /usr/local/bin/cleanup.sh
systemd-run --on-calendar="Mon 09:00" --unit=mondays /usr/local/bin/report.sh

# Inspect / cancel
systemctl list-timers
systemctl stop mondays.timer
```

### Boot Override for Recovery

```bash
# At GRUB, append to kernel command line:
systemd.unit=rescue.target           # single-user-ish
systemd.unit=emergency.target        # most minimal
systemd.debug-shell=1                # spawn debug shell on tty9
init=/bin/bash                       # bypass systemd entirely
```

## Tips

- `daemon-reload` after every unit-file edit. The most common mistake by a wide margin.
- `mask` is much stronger than `disable` — it symlinks the unit to `/dev/null`, so nothing (including dependencies) can start it. Used to permanently prevent services like `getty@tty3` or `firewalld`.
- `systemctl edit X` is the safe way to override a vendor unit; `systemctl edit --full X` only when you need to fully replace.
- Use `Type=notify` whenever your code can call `sd_notify`. It's the only Type where dependents actually wait for readiness.
- `Restart=on-failure` only restarts on non-zero exit / signal / timeout / watchdog. `Restart=always` also restarts on clean exits — usually wrong.
- Always pair `Restart=always` with `StartLimitIntervalSec=` and `StartLimitBurst=` to avoid restart storms on misconfig.
- `systemctl --user` for per-user services. Combine with `loginctl enable-linger USER` to keep them up after logout.
- `journalctl -xeu X.service` is the canonical "what just broke" command — `-x` shows hints, `-e` jumps to end, `-u` filters by unit.
- `systemd-analyze blame` to find slow-boot offenders; `systemd-analyze critical-chain` to see the gating chain.
- `systemd-analyze security X.service` for a free hardening checklist with rationale.
- `systemd-run --scope` is a great way to run an ad-hoc command under a cgroup with limits.
- `systemctl cat X` shows the unit + all drop-ins merged — the canonical way to confirm "what is systemd actually running?"
- `journalctl --vacuum-size=500M` to manually reclaim journal disk space.
- `systemd-tmpfiles --create --boot` to apply boot-only tmpfiles entries (those without `!`).
- Use `systemctl reload-or-restart` when you don't care about minor downtime; `systemctl reload` only when you know `ExecReload` is correct.
- `systemd-cgtop` is the cgroup version of `top` — fantastic for resource-attribution debugging.
- `OnCalendar=` is wildly more expressive than cron; validate with `systemd-analyze calendar "spec"` before deploying.
- `Persistent=true` on timers is the cron-replacement secret sauce — catches up missed runs after downtime.
- For containerized workloads, `systemd-nspawn` is excellent and integrates with `machinectl`.
- Prefer `network-online.target` over `network.target` for services that actually need connectivity.
- `journalctl --user` works just like `journalctl` but for your user-instance services.
- `systemd-resolve` was renamed `resolvectl` in v239; older docs may say `systemd-resolve`.
- `loginctl show-session $XDG_SESSION_ID` to see your seat/user-instance environment.
- The legacy `chkconfig` / `service` commands still work on RHEL — they delegate to systemd.
- `systemctl preset` applies the distro's `*.preset` defaults — useful after a fresh install or for ensuring a clean baseline.

## See Also

- bash, zsh, dbus, polyglot

## References

- [man systemd(1)](https://man7.org/linux/man-pages/man1/systemd.1.html)
- [man systemctl(1)](https://man7.org/linux/man-pages/man1/systemctl.1.html)
- [man journalctl(1)](https://man7.org/linux/man-pages/man1/journalctl.1.html)
- [man systemd.unit(5)](https://man7.org/linux/man-pages/man5/systemd.unit.5.html)
- [man systemd.service(5)](https://man7.org/linux/man-pages/man5/systemd.service.5.html)
- [man systemd.exec(5)](https://man7.org/linux/man-pages/man5/systemd.exec.5.html)
- [man systemd.resource-control(5)](https://man7.org/linux/man-pages/man5/systemd.resource-control.5.html)
- [man systemd.socket(5)](https://man7.org/linux/man-pages/man5/systemd.socket.5.html)
- [man systemd.timer(5)](https://man7.org/linux/man-pages/man5/systemd.timer.5.html)
- [man systemd.path(5)](https://man7.org/linux/man-pages/man5/systemd.path.5.html)
- [man systemd.mount(5)](https://man7.org/linux/man-pages/man5/systemd.mount.5.html)
- [man systemd.automount(5)](https://man7.org/linux/man-pages/man5/systemd.automount.5.html)
- [man systemd.swap(5)](https://man7.org/linux/man-pages/man5/systemd.swap.5.html)
- [man systemd.target(5)](https://man7.org/linux/man-pages/man5/systemd.target.5.html)
- [man systemd.slice(5)](https://man7.org/linux/man-pages/man5/systemd.slice.5.html)
- [man systemd.scope(5)](https://man7.org/linux/man-pages/man5/systemd.scope.5.html)
- [man systemd.device(5)](https://man7.org/linux/man-pages/man5/systemd.device.5.html)
- [man systemd-analyze(1)](https://man7.org/linux/man-pages/man1/systemd-analyze.1.html)
- [man systemd-run(1)](https://man7.org/linux/man-pages/man1/systemd-run.1.html)
- [man systemd-tmpfiles(8)](https://man7.org/linux/man-pages/man8/systemd-tmpfiles.8.html)
- [man tmpfiles.d(5)](https://man7.org/linux/man-pages/man5/tmpfiles.d.5.html)
- [man systemd-sysusers(8)](https://man7.org/linux/man-pages/man8/systemd-sysusers.8.html)
- [man sysusers.d(5)](https://man7.org/linux/man-pages/man5/sysusers.d.5.html)
- [man systemd-resolved(8)](https://man7.org/linux/man-pages/man8/systemd-resolved.service.8.html)
- [man resolvectl(1)](https://man7.org/linux/man-pages/man1/resolvectl.1.html)
- [man systemd-networkd(8)](https://man7.org/linux/man-pages/man8/systemd-networkd.service.8.html)
- [man systemd.network(5)](https://man7.org/linux/man-pages/man5/systemd.network.5.html)
- [man systemd.netdev(5)](https://man7.org/linux/man-pages/man5/systemd.netdev.5.html)
- [man networkctl(1)](https://man7.org/linux/man-pages/man1/networkctl.1.html)
- [man journald.conf(5)](https://man7.org/linux/man-pages/man5/journald.conf.5.html)
- [man systemd.directives(7)](https://man7.org/linux/man-pages/man7/systemd.directives.7.html)
- [man systemd.kill(5)](https://man7.org/linux/man-pages/man5/systemd.kill.5.html)
- [man sd_notify(3)](https://man7.org/linux/man-pages/man3/sd_notify.3.html)
- [man sd_listen_fds(3)](https://man7.org/linux/man-pages/man3/sd_listen_fds.3.html)
- [systemd Documentation Index](https://www.freedesktop.org/software/systemd/man/latest/)
- [systemd Project Page](https://systemd.io/)
- [systemd for Administrators (Lennart Poettering blog series)](https://0pointer.de/blog/projects/systemd-for-admins-1.html)
- [Arch Wiki — systemd](https://wiki.archlinux.org/title/Systemd)
- [Arch Wiki — systemd FAQ](https://wiki.archlinux.org/title/Systemd/FAQ)
- [Arch Wiki — systemd/Journal](https://wiki.archlinux.org/title/Systemd/Journal)
- [Arch Wiki — systemd/Timers](https://wiki.archlinux.org/title/Systemd/Timers)
- [Arch Wiki — systemd/User](https://wiki.archlinux.org/title/Systemd/User)
- [Red Hat — Managing System Services](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-system-services-with-systemctl_configuring-basic-system-settings)
- [Ubuntu — systemd](https://manpages.ubuntu.com/manpages/noble/man1/systemd.1.html)
- [systemd by example](https://systemd-by-example.com/)
- [RFC 5424 — Syslog Protocol](https://www.rfc-editor.org/rfc/rfc5424)
