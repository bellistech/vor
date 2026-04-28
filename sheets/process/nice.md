# nice (process priority)

Control CPU and I/O scheduling priority for processes.

## CPU Priority (nice)

### Launch with Modified Priority

```bash
# Nice values range from -20 (highest priority) to 19 (lowest)
# Default nice value is 0

# Run with lower priority (nice to other processes)
nice -n 10 /usr/local/bin/heavy_job.sh

# Run with even lower priority
nice -n 19 make -j8

# Run with higher priority (requires root)
nice -n -5 /usr/local/bin/critical.sh

# Highest priority (requires root)
nice -n -20 /usr/local/bin/realtime_critical.sh
```

### Change Running Process Priority

```bash
# Renice a running process
renice -n 10 -p 1234

# Renice all processes of a user
renice -n 15 -u deploy

# Renice a process group
renice -n 5 -g 5678

# Make a running process higher priority (requires root)
renice -n -5 -p 1234
```

### Check Current Nice Value

```bash
# Show nice value in ps output
ps -eo pid,ni,comm --sort=-ni

# Check a specific process
ps -o pid,ni,comm -p 1234
```

## I/O Priority (ionice)

### Scheduling Classes

```bash
# Classes:
#   0 — None (use CPU nice to derive I/O priority)
#   1 — Realtime (highest, requires root, priority 0-7)
#   2 — Best-effort (default, priority 0-7)
#   3 — Idle (only when no other I/O, no priority level)

# Run with idle I/O priority (won't compete for disk)
ionice -c 3 rsync -a /data/ /backup/

# Run with best-effort, low priority
ionice -c 2 -n 7 tar czf backup.tar.gz /var/data/

# Run with realtime I/O (requires root)
ionice -c 1 -n 0 dd if=/dev/sda of=/dev/sdb

# Check I/O priority of a running process
ionice -p 1234

# Change I/O class of a running process
ionice -c 3 -p 1234
```

## Combining CPU and I/O

### Low-Impact Background Jobs

```bash
# Low CPU priority + idle I/O = minimal system impact
nice -n 19 ionice -c 3 find / -type f -name "*.log" -mtime +30 -delete

# Background compilation
nice -n 15 ionice -c 2 -n 7 make -j$(nproc)

# Low-priority backup
nice -n 19 ionice -c 3 rsync -a --bwlimit=10m /data/ /backup/
```

## Tips

- Regular users can only increase nice values (lower priority). Only root can set negative nice values.
- Nice value affects CPU scheduling share, not hard limits -- a process at nice 19 will still use 100% CPU if nothing else wants it.
- `ionice -c 3` (idle) is extremely useful for backups and maintenance tasks -- they only get I/O when the disk is otherwise idle.
- The CFQ I/O scheduler respects ionice classes. If you use `mq-deadline` or `none` (common on NVMe), ionice has no effect.
- `chrt` is the tool for real-time CPU scheduling (FIFO, round-robin) -- different from nice values.
- On systemd-managed services, use `Nice=` and `IOSchedulingClass=` in the unit file instead of wrapping with nice/ionice.

## chrt — Real-Time Scheduling (different beast)

`nice` only adjusts the CFS (Completely Fair Scheduler) weight. For *real-time* scheduling — guaranteed time slices, fixed priority queues — use `chrt`.

```bash
# Show current scheduling policy + priority for a pid
chrt -p 1234
# pid 1234's current scheduling policy: SCHED_OTHER
# pid 1234's current scheduling priority: 0

# Run a command under SCHED_FIFO (real-time, runs until it yields/blocks)
sudo chrt -f 50 /usr/local/bin/audio-engine
# -f = FIFO, 50 = static priority (1-99 for RT classes)

# SCHED_RR (real-time, round-robin) — same priority levels share CPU
sudo chrt -r 50 ./worker

# SCHED_BATCH — non-RT but hint to the scheduler this is a CPU-bound batch job
chrt -b 0 ./bulk_compress

# SCHED_IDLE — even lower than nice 19; only runs when nothing else wants CPU
chrt -i 0 ./background_indexer

# Drop back to SCHED_OTHER (the default CFS class)
sudo chrt -o -p 0 1234

# List supported policies on this kernel
chrt -m
# SCHED_OTHER min/max priority    : 0/0
# SCHED_FIFO min/max priority     : 1/99
# SCHED_RR min/max priority       : 1/99
# SCHED_BATCH min/max priority    : 0/0
# SCHED_IDLE min/max priority     : 0/0
# SCHED_DEADLINE min/max priority : 0/0
```

**Why this matters:** a CPU-bound `SCHED_FIFO` task at priority 99 can lock out the entire system, including kernel threads, until it sleeps. Use real-time classes ONLY when you understand the trade-off (audio engines, hard real-time control loops, latency-sensitive trading systems). Mistakes here can require power-cycling the box.

## Cgroups Era — Modern Priority Control

On modern systems (cgroup v2 + systemd), `nice` and `ionice` are still useful but cgroups are the proper tool for resource control. `nice` says "this process should get less CPU"; cgroups say "this group of processes can use at most X CPU and Y MB/s of disk".

### systemd-run for One-Shot Limits

```bash
# Run a command under a transient cgroup with CPU and I/O caps
systemd-run --scope --user \
  -p CPUWeight=20 \
  -p IOWeight=20 \
  -p MemoryHigh=2G \
  -p MemoryMax=4G \
  ./resource-hungry-job

# Equivalent of "nice 19 + ionice idle" with hard memory ceiling.
```

### Service Unit Equivalents

For long-running services, write the policy into the unit file rather than wrapping the command:

```ini
[Service]
ExecStart=/usr/local/bin/myservice
Nice=10
IOSchedulingClass=best-effort
IOSchedulingPriority=7
CPUWeight=50           # cgroup v2 (1-10000, default 100)
IOWeight=50            # cgroup v2 (1-10000, default 100)
CPUQuota=80%           # hard cap at 80% of one CPU
MemoryMax=2G
TasksMax=200
```

After editing, `systemctl daemon-reload && systemctl restart myservice`.

### cgroup v2 Direct Inspection

```bash
# Find the cgroup of a process
cat /proc/1234/cgroup
# 0::/user.slice/user-1000.slice/session-3.scope

# Inspect limits applied
cat /sys/fs/cgroup/user.slice/user-1000.slice/session-3.scope/cpu.weight
cat /sys/fs/cgroup/user.slice/user-1000.slice/session-3.scope/io.weight
cat /sys/fs/cgroup/user.slice/user-1000.slice/session-3.scope/memory.max
```

## Worked Recipes

### Background Build That Doesn't Stutter the Desktop

```bash
# Compile a large project on your workstation while continuing to use it.
# Combines low CPU nice + idle I/O + memory soft-limit via systemd-run.
systemd-run --user --scope \
  -p CPUWeight=10 \
  -p IOWeight=10 \
  -p MemoryHigh=8G \
  bash -c 'cd ~/code/big-project && make -j$(nproc)'
```

### Backup That Yields to Real Work

```bash
# rsync at idle I/O priority + low CPU + bandwidth cap
nice -n 19 ionice -c 3 rsync -aHAX \
  --bwlimit=20m \
  --info=progress2 \
  /home/ /backup/home/

# Why every flag matters:
#   nice -n 19    — CFS weight floor (~5% share when contended)
#   ionice -c 3   — idle I/O class (only runs when disk is otherwise quiet)
#   --bwlimit=20m — 20 MB/s ceiling (avoids saturating slow uplinks)
#   -aHAX         — preserve hardlinks, ACLs, xattrs (full fidelity)
#   --info=progress2 — total-byte progress, not per-file
```

### One-Off Cleanup Job

```bash
# Delete old logs without touching foreground performance
nice -n 19 ionice -c 3 find /var/log -type f -name "*.log.*.gz" -mtime +90 -delete

# Equivalent with parallel gnu-parallel:
nice -n 19 ionice -c 3 find /var/log -name "*.log.*.gz" -mtime +90 \
  | parallel --bar -j 2 rm
```

### Realtime Audio Server

```bash
# JACK / PulseAudio / PipeWire on RT-PREEMPT kernel
sudo chrt -f 70 -p $(pgrep -x jackd)

# Or via the unit file (the right way):
# /etc/systemd/system/jackd.service:
[Service]
ExecStart=/usr/bin/jackd -d alsa
CPUSchedulingPolicy=fifo
CPUSchedulingPriority=70
LimitRTPRIO=99
LimitMEMLOCK=infinity
```

### Container CPU/IO Limits (Docker / Kubernetes)

```bash
# Docker — uses the same cgroup primitives under the hood
docker run --cpus=0.5 --memory=1g --blkio-weight=200 myimage

# Kubernetes pod spec
resources:
  requests:
    cpu: "100m"        # 0.1 CPU guaranteed
    memory: "256Mi"
  limits:
    cpu: "500m"        # cap at 0.5 CPU
    memory: "1Gi"
```

## Common Errors and Fixes

```bash
# "renice: failed to set priority for 1234: Permission denied"
$ renice -n -5 -p 1234
# Cause: only root can lower the nice value (raise priority).
# Fix:   sudo renice -n -5 -p 1234

# "renice: failed to set priority for 1234: Operation not permitted"
$ renice -n 5 -p 1234
# Cause: trying to renice another user's process without root.
# Fix:   sudo renice -n 5 -p 1234

# nice/ionice has no apparent effect
# Cause 1: scheduler is mq-deadline / none / kyber (NVMe defaults).
#          ionice classes only respect the CFQ/BFQ schedulers.
$ cat /sys/block/sda/queue/scheduler
[mq-deadline] kyber bfq none      # current = mq-deadline → ionice classes ignored
# Fix: switch to BFQ if you want ionice to bite (impacts throughput on NVMe):
echo bfq | sudo tee /sys/block/sda/queue/scheduler

# Cause 2: kernel CFS auto-grouping merges your terminal's nice values
#          across all foreground processes — your -n 19 may not show.
$ cat /proc/sys/kernel/sched_autogroup_enabled
1
# Workaround: setsid the command into its own session, or use systemd-run.

# "chrt: failed to set pid 1234's policy: Operation not permitted"
$ chrt -f 50 -p 1234
# Cause: real-time scheduling requires CAP_SYS_NICE or root.
# Fix:   sudo chrt -f 50 -p 1234
```

## Tips

- Regular users can only INCREASE nice values (lower priority). Only root can set negative nice values.
- Nice value affects CPU scheduling SHARE, not hard limits — a process at nice 19 will still use 100% CPU if nothing else wants it.
- `ionice -c 3` (idle) is extremely useful for backups and maintenance tasks — they only get I/O when the disk is otherwise idle.
- The CFQ/BFQ I/O schedulers respect ionice classes. If you use `mq-deadline` or `none` (common on NVMe), ionice has no effect — see "Common Errors" above.
- `chrt` is the tool for real-time CPU scheduling (FIFO, round-robin) — different from nice values, and dangerous if misused.
- On systemd-managed services, use `Nice=`, `IOSchedulingClass=`, `CPUWeight=`, `IOWeight=`, `MemoryHigh=`, `MemoryMax=` in the unit file instead of wrapping with nice/ionice.
- **NEVER** set negative nice values on database servers or anything latency-sensitive without measuring — you can starve kernel threads and trigger soft-lockup warnings.
- A child process inherits the parent's nice value. `nice -n 10 bash` then everything launched from that shell is nice 10.
- `taskset` pins to specific CPUs (cgroup-style affinity) — orthogonal to nice; combine for "low priority and only on cores 4-7".
- `nohup nice -n 19 ionice -c 3 long_job &` is the classic "fire and forget" combo.

## See Also

- system/htop, system/iostat, system/lsof, system/cgroups, system/systemd

## References

- [man nice(1)](https://man7.org/linux/man-pages/man1/nice.1.html)
- [man nice(2) — System Call](https://man7.org/linux/man-pages/man2/nice.2.html)
- [man renice(1)](https://man7.org/linux/man-pages/man1/renice.1.html)
- [man ionice(1)](https://man7.org/linux/man-pages/man1/ionice.1.html)
- [man getpriority(2) / setpriority(2)](https://man7.org/linux/man-pages/man2/getpriority.2.html)
- [man sched(7) — Scheduling Policies](https://man7.org/linux/man-pages/man7/sched.7.html)
- [Kernel CFS Scheduler](https://www.kernel.org/doc/html/latest/scheduler/sched-design-CFS.html)
- [Arch Wiki — Process Management](https://wiki.archlinux.org/title/Process)
- [Ubuntu Manpage — nice](https://manpages.ubuntu.com/manpages/noble/man1/nice.1.html)
