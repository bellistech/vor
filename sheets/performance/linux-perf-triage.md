# Linux Performance Triage (First 60 Seconds, USE / RED Methods)

A paste-runnable cookbook for diagnosing Linux performance problems fast. Brendan Gregg's "first 60 seconds" checklist, the USE method (Utilization, Saturation, Errors), the RED method (Rate, Errors, Duration), tool flag references, and real-world diagnostic recipes — every command runnable, root requirements marked, expected output and interpretation included.

## Setup

Install the core triage toolkit. Most distros ship a subset; install the rest from official repos.

### Debian / Ubuntu

```bash
sudo apt update
sudo apt install -y procps htop iotop iftop atop dstat sysstat \
                    psmisc lsof strace ltrace tcpdump ncdu \
                    smartmontools nethogs mtr-tiny iproute2 \
                    net-tools bind9-dnsutils linux-tools-common \
                    linux-tools-generic bpftrace bpfcc-tools \
                    numactl util-linux
```

`sysstat` provides `sar`, `iostat`, `mpstat`, `pidstat`. After install, enable data collection:

```bash
sudo sed -i 's/^ENABLED="false"/ENABLED="true"/' /etc/default/sysstat
sudo systemctl enable --now sysstat
```

### RHEL / CentOS / Rocky / Alma / Fedora

```bash
sudo dnf install -y procps-ng htop iotop iftop atop dstat sysstat \
                    psmisc lsof strace ltrace tcpdump ncdu \
                    smartmontools nethogs mtr iproute net-tools \
                    bind-utils perf bcc-tools bpftrace numactl \
                    util-linux
sudo systemctl enable --now sysstat
```

### Arch / Manjaro

```bash
sudo pacman -S --needed procps-ng htop iotop iftop atop dstat sysstat \
                       psmisc lsof strace ltrace tcpdump ncdu \
                       smartmontools nethogs mtr iproute2 net-tools \
                       bind perf bpf bcc bcc-tools bpftrace numactl
```

### Alpine (containers / minimal)

```bash
sudo apk add procps htop iotop iftop atop dstat sysstat lsof strace \
            tcpdump bind-tools mtr ncdu smartmontools nethogs perf \
            bcc-tools bpftrace
```

### Verify install

```bash
for t in top htop iotop iftop atop dstat sar vmstat pidstat mpstat \
         free df du ss lsof strace tcpdump perf bpftrace; do
  command -v "$t" >/dev/null && printf "%-10s OK\n" "$t" || \
                                printf "%-10s MISSING\n" "$t"
done
```

### Capability requirements

| Tool             | Root? | Notes                                        |
|------------------|-------|----------------------------------------------|
| top, htop, free  | No    | Read-only /proc                              |
| vmstat, mpstat   | No    | Read-only                                    |
| iostat, sar      | No    | sar needs `sysstat` collector active         |
| iotop            | YES   | Needs CAP_NET_ADMIN + tracepoints            |
| iftop, nethogs   | YES   | Raw socket capture                           |
| ss               | No    | Minor info; root for some socket details     |
| strace -p PID    | YES   | Unless same UID; ptrace_scope may block      |
| lsof             | Mixed | Other UIDs need root                         |
| perf record      | YES   | Or `kernel.perf_event_paranoid=-1`           |
| bpftrace         | YES   | Or CAP_BPF + CAP_PERFMON (5.8+)              |
| tcpdump          | YES   | Or CAP_NET_RAW                               |
| smartctl         | YES   | Direct device access                         |
| sysctl -w        | YES   | Setting kernel tunables                      |

## The First 60 Seconds

Brendan Gregg's "Linux Performance Analysis in 60 Seconds" — ten commands that triage 80% of incidents. Run them in this exact order, copy the output, then analyze.

### The checklist

```bash
uptime
dmesg | tail -20
vmstat 1 5
mpstat -P ALL 1 5
pidstat 1 5
iostat -xz 1 5
free -m
sar -n DEV 1 5
sar -n TCP,ETCP 1 5
top -b -n 1 | head -30
```

### 1. `uptime` — load averages

```bash
uptime
```

Sample output:

```
14:23:01 up 12 days,  3:42,  4 users,  load average: 8.42, 5.91, 3.18
```

Interpretation: load is climbing (1m > 5m > 15m). On a 4-core box, 8.42 = 2.1× saturated. On 16 cores, fine. **Always compare to `nproc`.**

### 2. `dmesg | tail` — kernel messages

```bash
sudo dmesg -T | tail -20
```

Look for: OOM kills, segfaults, TCP "Out of memory", disk errors, NIC link flaps, hung tasks (`INFO: task X blocked for more than 120 seconds`). The `-T` adds human timestamps.

### 3. `vmstat 1` — virtual memory + CPU

```bash
vmstat 1 5
```

Sample:

```
procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
 8  0      0 124312   2104 387210    0    0    52   180  890 1240 38  4 56  2  0
 9  0      0 124100   2104 387210    0    0     0     0 1023 1456 42  5 51  2  0
```

Key columns: `r` (runnable) > CPU count = saturation; `b` (blocked I/O) > 0 sustained = disk bottleneck; `si`/`so` > 0 = swapping; `wa` high = waiting on disk; `st` > 0 = hypervisor steal time.

### 4. `mpstat -P ALL 1` — per-CPU breakdown

```bash
mpstat -P ALL 1 5
```

Look for hot CPUs (single-thread bottleneck) vs balanced load. `%steal` > 0 on a VM means the hypervisor is throttling you.

### 5. `pidstat 1` — per-process CPU

```bash
pidstat 1 5
```

Like top but rolling — won't miss short-lived processes.

### 6. `iostat -xz 1` — extended disk stats

```bash
iostat -xz 1 5
```

Critical columns: `%util` > 80 = saturation; `r_await`/`w_await` > 10 ms = slow; `aqu-sz` > 1 = queue building.

### 7. `free -m` — memory

```bash
free -m
```

Look at `available` (not `free`). Linux uses RAM aggressively for cache; "free" is misleading.

### 8. `sar -n DEV 1` — network throughput

```bash
sar -n DEV 1 5
```

Compare `rxkB/s` and `txkB/s` against NIC link speed. 100MB/s sustained on a 1Gbps link = saturated.

### 9. `sar -n TCP,ETCP 1` — TCP errors

```bash
sar -n TCP,ETCP 1 5
```

Watch `retrans/s`, `isegerr/s`. Retransmits > 1% of segments = network problem.

### 10. `top -b -n 1` — top processes

```bash
top -b -n 1 -o %CPU | head -20
top -b -n 1 -o %MEM | head -20
```

Quick read of who's burning CPU and who's eating RAM. Use `-b` (batch) so it's pasteable.

### Capture all in one shot

```bash
SNAP="/tmp/snapshot-$(date +%Y%m%d-%H%M%S).txt"
{
  echo "=== uptime ===";        uptime
  echo "=== dmesg tail ==="; sudo dmesg -T | tail -20
  echo "=== vmstat 1 5 ===";    vmstat 1 5
  echo "=== mpstat ===";        mpstat -P ALL 1 5
  echo "=== pidstat ===";       pidstat 1 5
  echo "=== iostat ===";        iostat -xz 1 5
  echo "=== free -m ===";       free -m
  echo "=== sar DEV ===";       sar -n DEV 1 5
  echo "=== sar TCP,ETCP ==="; sar -n TCP,ETCP 1 5
  echo "=== top ===";           top -b -n 1 | head -40
} > "$SNAP" 2>&1
echo "Saved: $SNAP"
```

## The USE Method

Brian Cantrill / Brendan Gregg's USE method: for every resource, check three things — Utilization (% of time busy), Saturation (queue depth or wait), Errors. Iterate every resource. If you cover them all, you'll find the bottleneck.

### USE cheat-table

| Resource    | Utilization                   | Saturation                       | Errors                       |
|-------------|-------------------------------|----------------------------------|------------------------------|
| CPU         | `mpstat -P ALL 1` `%idle` inv | `vmstat 1` `r` column > nproc    | `dmesg` MCE / thermal        |
| Memory      | `free -m` used%               | `vmstat 1` `si`/`so` > 0         | `dmesg` OOM / mcelog         |
| Disk IO     | `iostat -xz 1` `%util`        | `iostat` `aqu-sz` > 1, `await`   | `dmesg`, `smartctl -a`       |
| Network IF  | `sar -n DEV 1` rx/tx vs link  | `ifconfig` dropped, `nstat` drop | `ip -s link`, `ethtool -S`   |
| Disk Space  | `df -h` used%                 | n/a (binary: full or not)        | `dmesg` "no space left"      |
| Swap        | `free -m` swap used           | `vmstat 1` `si`/`so` rate        | OOM in dmesg                 |
| Filesys     | `df -i` inodes                | open file descriptors near limit | EIO in dmesg                 |
| TCP sockets | `ss -s` summary               | `ss -ant state syn-recv` count   | `nstat -az` retransmits      |
| Sched runq  | `vmstat 1` `r`                | `runqlat.bt` (bpftrace)          | n/a                          |

### One-liner USE check per resource

CPU:

```bash
mpstat -P ALL 1 1; vmstat 1 2 | tail -1
```

Memory:

```bash
free -m; vmstat 1 2 | tail -1 | awk '{print "si="$7" so="$8}'
```

Disk:

```bash
iostat -xz 1 2 | tail -n +4
```

Network:

```bash
sar -n DEV 1 1
ip -s link
nstat -az | grep -iE 'retrans|drop|error'
```

## The RED Method

Tom Wilkie's RED method (for services): Rate (req/s), Errors (err/s), Duration (latency distribution). USE is for resources; RED is for services. Use both — USE finds the saturated subsystem, RED finds the broken endpoint.

### RED in practice

For an HTTP service behind nginx:

```bash
sudo tail -f /var/log/nginx/access.log | \
  awk '{ rate++; if ($9 >= 500) errs++; sum += $NF }
       END { print "rate="rate" errs="errs" avg_dur="sum/rate }'
```

Extract per-endpoint p99 with `awk` quantiles or push to Prometheus. The `histogram_quantile()` function on a Prometheus histogram gives you percentile latencies.

### RED with `ss` for TCP services

```bash
ss -tin state established '( sport = :443 )' | \
  awk '/cwnd/{print $0}' | head
```

`rtt:` is per-connection round-trip time + variance; `cwnd:` is congestion window. Anomalies = network or app stalls.

## Load Averages

`uptime` and `w` show 1-minute, 5-minute, 15-minute load averages.

### What they actually mean (Linux-specific)

```bash
uptime
w
cat /proc/loadavg
```

Sample `/proc/loadavg`:

```
2.41 1.97 1.42 4/1234 56789
```

Fields: 1m, 5m, 15m, runnable/total tasks, last PID. **On Linux, load includes both runnable AND uninterruptible (D-state) tasks** — disk-blocked processes count. Other Unixes count only runnable. So a Linux load of 8 with `r=1` means 7 processes are stuck in D-state on something (usually disk or NFS).

### Decoding the trend

| Pattern                  | Interpretation                                    |
|--------------------------|---------------------------------------------------|
| 1m < 5m < 15m            | Load decreasing; recovering                       |
| 1m > 5m > 15m            | Load increasing; problem starting                 |
| 1m ≈ 5m ≈ 15m            | Steady state                                      |
| 1m >> 5m, sudden spike   | Acute event in the last minute (e.g., burst)      |

### Saturation threshold

```bash
NPROC=$(nproc); LOAD=$(awk '{print $1}' /proc/loadavg); \
echo "load=$LOAD cpus=$NPROC ratio=$(awk -v l=$LOAD -v c=$NPROC 'BEGIN{printf "%.2f", l/c}')"
```

Ratio > 1.0 = saturated. Ratio > 2.0 = sustained queueing. **But** check D-state count first — a load of 16 with 16 D-state processes is an I/O problem, not a CPU problem.

### Find D-state processes

```bash
ps -eo pid,stat,comm,wchan | awk '$2 ~ /^D/'
```

`wchan` shows where they're stuck (e.g., `io_schedule`, `nfs_wait_on_request`).

## CPU — top / htop

### `top` essentials

```bash
top
```

Interactive keys:

| Key | Action                                  |
|-----|-----------------------------------------|
| 1   | Toggle per-CPU view                     |
| t   | Toggle CPU summary mode                 |
| m   | Toggle memory summary mode              |
| P   | Sort by %CPU                            |
| M   | Sort by %MEM                            |
| T   | Sort by time                            |
| c   | Toggle full command path                |
| H   | Toggle threads (show LWP)               |
| f   | Field selector (add %wa per process)    |
| W   | Save config to `~/.toprc`               |
| k   | Kill (asks PID + signal)                |
| r   | Renice (asks PID + value)               |
| u   | Filter by user                          |
| o   | Add filter expression (e.g. `%CPU>5.0`) |
| q   | Quit                                    |

### CPU summary line — `us / sy / ni / id / wa / hi / si / st`

```
%Cpu(s):  35.2 us,  4.1 sy,  0.0 ni, 56.3 id,  3.8 wa,  0.0 hi,  0.6 si,  0.0 st
```

| Field | Meaning                                                 |
|-------|---------------------------------------------------------|
| us    | User-space CPU                                          |
| sy    | Kernel CPU                                              |
| ni    | Niced (lowered priority) user CPU                       |
| id    | Idle                                                    |
| wa    | I/O wait (CPU idle but tasks waiting on I/O)            |
| hi    | Hardware IRQ servicing                                  |
| si    | Software IRQ (softirq, e.g., network bottom halves)     |
| st    | Steal time — hypervisor took CPU from this guest VM     |

`wa` high = disk bottleneck. `si` high on network-heavy box = NIC IRQ saturation; check `cat /proc/interrupts` and consider RPS/RFS. `st` > 5% on a VM = noisy neighbor or oversubscribed host.

### `htop`

```bash
htop
```

Better defaults: per-CPU bars, tree mode (`F5`), process search (`F3`), filter (`F4`), kill (`F9`), nice (`F7`/`F8`). Reads `/proc` like top but renders prettily.

Set up tree view:

```bash
htop -t
```

### Batch-mode `top` for scripts

```bash
top -b -n 1 -o %CPU | head -25       # one-shot, CPU sort
top -b -d 1 -n 5 > /tmp/top-snap.txt  # 5 samples 1s apart
```

## CPU — `mpstat -P ALL 1`

Per-CPU statistics — find single-CPU bottlenecks invisible in aggregate.

```bash
mpstat -P ALL 1 5
```

Sample output:

```
CPU    %usr   %nice    %sys %iowait    %irq   %soft  %steal  %idle
all   38.21    0.00    4.10    1.42    0.00    0.61    0.00   55.66
  0   97.00    0.00    2.00    0.00    0.00    1.00    0.00    0.00
  1   12.00    0.00    3.00    1.00    0.00    0.00    0.00   84.00
  2    8.00    0.00    4.00    2.00    0.00    1.00    0.00   85.00
  3   10.00    0.00    5.00    1.00    0.00    1.00    0.00   83.00
```

CPU 0 is pinned at 97% user — single-threaded hot loop. Aggregate `all` only shows 38% — you'd miss it without per-CPU.

### Steal column on VMs

```bash
mpstat 1 10 | awk 'NR>3 && $NF != "%idle" {print $0}'
```

`%steal` is the % of time the vCPU was ready but the host scheduled another guest. Persistent > 5% = oversubscribed host or neighbor stomping.

### Compare to `/proc/interrupts`

```bash
watch -n 1 'cat /proc/interrupts | head -25'
```

If one CPU is hot and `cat /proc/interrupts` shows that CPU receiving most NIC IRQs, enable RPS:

```bash
echo ffff | sudo tee /sys/class/net/eth0/queues/rx-0/rps_cpus
```

## CPU — `pidstat 1`

Per-process CPU rolling — like top, but emits a line per process per interval. Won't miss short-lived processes.

```bash
pidstat 1 5
```

### Per-thread

```bash
pidstat -t 1 5
```

`-t` adds threads (TID). The aggregate process line has `TID=-`.

### Specific PID

```bash
pidstat -p 12345 1 10
pidstat -t -p 12345 1 10           # with threads
```

### Per-process I/O

```bash
pidstat -d 1 5
```

Columns: `kB_rd/s` `kB_wr/s` `kB_ccwr/s` `iodelay` — find which process is reading/writing most.

### Per-process page faults

```bash
pidstat -r 1 5
```

`majflt/s` = major faults (disk read needed). High = working set bigger than RAM, or aggressive cache eviction.

### Context switches per process

```bash
pidstat -w 1 5
```

`cswch/s` voluntary; `nvcswch/s` involuntary. Many involuntary cs = CPU contention.

## Memory — `free -m`

```bash
free -m
```

Sample:

```
              total        used        free      shared  buff/cache   available
Mem:          15876        3221         421         123       12233       12101
Swap:          2047           0        2047
```

**Read `available`, not `free`.** Linux uses unused RAM as page cache. `available` = approximation of how much is usable for new processes without swapping (free + reclaimable cache).

### Other useful flags

```bash
free -h           # human-readable
free -w           # split buff and cache columns
free -s 1         # repeat every 1s
free -t           # add total row (RAM + swap)
```

### When `available` is low

| `available` size  | Action                                                  |
|-------------------|---------------------------------------------------------|
| > 20% RAM         | Healthy                                                 |
| 5–20%             | Watch — check working set, biggest consumers            |
| < 5%              | Warning — likely page-cache pressure                    |
| Approaching 0     | Imminent swapping or OOM                                |

```bash
ps aux --sort=-%mem | head -10                  # top RAM consumers
ps -eo pid,rss,comm --sort=-rss | head -10
```

## Memory — `vmstat 1`

Header columns explained — memorize these.

```bash
vmstat 1 5
```

```
procs -----------memory---------- ---swap-- -----io---- -system-- ------cpu-----
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
 4  0      0 423012   2104 12233048    0    0    52   180  890 1240 38  4 56  2  0
```

| Col   | Meaning                                                                |
|-------|------------------------------------------------------------------------|
| r     | Tasks runnable (running + queued). Saturation if `r > nproc`.          |
| b     | Tasks blocked (uninterruptible, usually disk). Saturation if > 0 sustained. |
| swpd  | Virtual mem swapped to disk (KB)                                       |
| free  | Idle RAM (KB) — usually small, that's fine                             |
| buff  | Buffers (filesystem metadata cache, KB)                                |
| cache | Page cache (file content cache, KB)                                    |
| si    | Swap-in rate KB/s (memory paged from swap)                             |
| so    | Swap-out rate KB/s (memory paged to swap)                              |
| bi    | Block-in KB/s (disk reads)                                             |
| bo    | Block-out KB/s (disk writes)                                           |
| in    | Interrupts/s                                                           |
| cs    | Context switches/s                                                     |
| us    | User CPU %                                                             |
| sy    | System CPU %                                                           |
| id    | Idle %                                                                 |
| wa    | I/O wait %                                                             |
| st    | Steal % (VM only)                                                      |

### Wide format

```bash
vmstat -wS M 1 5      # wide cols, MB instead of KB
```

### Slab info

```bash
vmstat -m | head -20
```

### Active/inactive memory

```bash
vmstat -a 1 5
```

Replaces `buff/cache` columns with `inact/active`. Active = recently used; inactive = candidate for reclaim.

## Memory — `/proc/meminfo` decoded

```bash
cat /proc/meminfo
```

Key fields:

| Field            | Meaning                                                       |
|------------------|---------------------------------------------------------------|
| MemTotal         | Total usable RAM                                              |
| MemFree          | Truly idle RAM                                                |
| MemAvailable     | Estimate of available without swapping (kernel computed)      |
| Buffers          | Block-device buffer cache (filesystem metadata)               |
| Cached           | Page cache (file contents)                                    |
| SwapCached       | Pages swapped out then back, still in swap too                |
| Active           | Recently used pages (anon + file)                             |
| Inactive         | Less-recently-used (reclaim candidates)                       |
| Active(anon)     | Active anonymous (heap, stack)                                |
| Active(file)     | Active page cache                                             |
| Unevictable      | Locked (mlock) or ramfs                                       |
| Mlocked          | Locked by mlock()                                             |
| SwapTotal        | Swap configured                                               |
| SwapFree         | Swap available                                                |
| Dirty            | Modified page cache pending writeback                         |
| Writeback        | Currently being written to disk                               |
| AnonPages        | Anonymous pages mapped (heap)                                 |
| Mapped           | Files mapped into memory                                      |
| Shmem            | Shared memory + tmpfs                                         |
| Slab             | Kernel slab allocator (data structures)                       |
| SReclaimable     | Slab portion that can be reclaimed (e.g. dentry, inode)       |
| SUnreclaim       | Slab pinned in kernel                                         |
| KernelStack      | Per-thread kernel stack                                       |
| PageTables       | Page table structures                                         |
| CommitLimit      | Total commit allowance (RAM × overcommit_ratio + swap)        |
| Committed_AS     | Currently committed virtual memory                            |
| VmallocTotal     | Vmalloc area total                                            |
| HugePages_Total  | Number of huge pages allocated                                |
| HugePages_Free   | Free huge pages                                               |
| Hugepagesize     | Default hugepage size                                         |

### Quick ratios

```bash
awk '/^MemTotal/{tot=$2}/^MemAvailable/{av=$2}END{
  printf "available=%.1f%% (%d MB / %d MB)\n", 100*av/tot, av/1024, tot/1024
}' /proc/meminfo
```

```bash
awk '/^Dirty:/{d=$2}/^Writeback:/{w=$2}END{
  printf "dirty=%d KB writeback=%d KB\n", d, w
}' /proc/meminfo
```

### When huge `Dirty:` and tiny `Writeback:`

You're filling page cache with writes faster than disk drains. Tune:

```bash
sudo sysctl vm.dirty_ratio=10              # max % dirty before sync flush
sudo sysctl vm.dirty_background_ratio=5    # bg flush threshold
sudo sysctl vm.dirty_expire_centisecs=1500
```

## Memory — Swap

```bash
swapon -s
swapon --show
free -m | grep -i swap
```

### Why is swap being used?

```bash
sysctl vm.swappiness            # 0–100, default 60. Lower = avoid swap.
sysctl vm.overcommit_memory     # 0 heuristic, 1 always allow, 2 strict
sysctl vm.overcommit_ratio
```

### Per-process swap usage

```bash
for f in /proc/*/status; do
  awk '/^Pid|^Name|^VmSwap/{
    if($1=="Pid:") pid=$2;
    if($1=="Name:") name=$2;
    if($1=="VmSwap:") sw=$2;
  } END { if(sw>0) printf "%-25s pid=%-7d swap=%d KB\n", name, pid, sw }' "$f"
done | sort -k4 -n -t= -r | head -20
```

Or simpler with `smem`:

```bash
sudo smem -t -k -s swap
```

### Swap thrashing

```bash
vmstat 1 | awk 'NR<=2 || $7>0 || $8>0'   # only show lines with si/so > 0
```

### Disable swap to triage

```bash
sudo swapoff -a            # may take time; needs RAM headroom
sudo swapon -a             # re-enable
```

### OOM killer events

```bash
sudo dmesg -T | grep -iE 'killed process|oom|out of memory'
sudo journalctl -k --since "1 hour ago" | grep -iE 'killed|oom'
```

Sample dmesg:

```
[Mon Apr 25 13:42:11 2026] Out of memory: Killed process 18472 (java) total-vm:8392132kB, anon-rss:6128304kB
```

Interpretation: Linux killed PID 18472 (java) using 6 GB RSS. Tune `oom_score_adj` per critical service:

```bash
echo -1000 | sudo tee /proc/$(pidof critical_service)/oom_score_adj
```

## Memory — `slabtop`

Kernel object cache view.

```bash
sudo slabtop -o | head -30
sudo slabtop -s c              # sort by cache size
sudo slabtop -s a              # sort by active objs
```

Sample:

```
 OBJS ACTIVE  USE OBJ SIZE  SLABS OBJ/SLAB CACHE SIZE NAME
210034 198120  94%    0.19K   5001       42     40008K dentry
182500 175200  95%    1.05K  18250       10    584000K xfs_inode
```

If `dentry` or `inode_cache` is huge, you're filesystem-walking a lot. Drop caches as a triage hack (don't do this in production unattended):

```bash
sync; echo 2 | sudo tee /proc/sys/vm/drop_caches      # dentries + inodes
sync; echo 3 | sudo tee /proc/sys/vm/drop_caches      # + page cache
```

## Disk IO — `iostat -xz 1`

The single most important disk command.

```bash
iostat -xz 1 5
```

`-x` extended fields, `-z` skip idle devices.

Sample:

```
Device      r/s    w/s    rkB/s    wkB/s rrqm/s wrqm/s  %rrqm  %wrqm r_await w_await aqu-sz rareq-sz wareq-sz svctm  %util
sda      102.00  48.00  4096.00  3072.00   0.00   2.00   0.00   4.00   12.31    8.92   1.84    40.16    64.00  0.62  93.20
nvme0n1   12.00  84.00   192.00 12288.00   0.00   1.00   0.00   1.18    0.45    0.92   0.08    16.00   146.29  0.04   3.21
```

| Col      | Meaning                                                      |
|----------|--------------------------------------------------------------|
| r/s      | Reads/sec                                                    |
| w/s      | Writes/sec                                                   |
| rkB/s    | Read throughput KB/s                                         |
| wkB/s    | Write throughput KB/s                                        |
| rrqm/s   | Read requests merged per second by I/O scheduler             |
| wrqm/s   | Write requests merged per second                             |
| %rrqm    | % of read requests merged                                    |
| %wrqm    | % of write requests merged                                   |
| r_await  | Avg ms a read request waited (queue + service)               |
| w_await  | Avg ms a write request waited                                |
| aqu-sz   | Avg queue depth (was `avgqu-sz`). > 1 = building queue       |
| rareq-sz | Avg read size KB                                             |
| wareq-sz | Avg write size KB                                            |
| svctm    | Avg service time (deprecated; misleading on multi-queue)     |
| %util    | % of time device had at least one outstanding I/O            |

### Thresholds

| Indicator              | Healthy           | Suspect           | Bad                |
|------------------------|-------------------|-------------------|--------------------|
| %util                  | < 60              | 60–80             | > 80               |
| r_await / w_await (HDD) | < 10 ms          | 10–20 ms          | > 20 ms            |
| r_await / w_await (SSD) | < 1 ms           | 1–5 ms            | > 5 ms             |
| r_await / w_await (NVMe)| < 0.5 ms         | 0.5–2 ms          | > 2 ms             |
| aqu-sz                 | < 1               | 1–4               | > 4                |

**Caveat:** on multi-queue NVMe, `%util` can show 100% with plenty of headroom (it's "any outstanding I/O" not "saturated"). Trust `await` and queue size on NVMe.

### Per-partition only

```bash
iostat -xz -p ALL 1 5         # every partition + device
iostat -xz -p sda 1 5         # just sda + its partitions
```

### Watch with timestamps

```bash
iostat -xtz 1
```

## Disk IO — `iotop`

Per-process disk I/O. **Requires root.**

```bash
sudo iotop
```

Interactive keys:

| Key | Action                              |
|-----|-------------------------------------|
| o   | Show only processes doing I/O       |
| a   | Toggle accumulated I/O (since start)|
| p   | Toggle PID/threads view             |
| ←/→ | Sort column                         |
| r   | Reverse sort                        |
| q   | Quit                                |

Batch mode:

```bash
sudo iotop -b -o -n 5 -d 1
```

Per-thread:

```bash
sudo iotop -P -o
```

### biotop / biolatency from bcc

```bash
sudo biotop                  # per-process disk I/O top (BPF)
sudo biolatency 1 10         # disk latency histogram, 10s windows
sudo biosnoop                # log every disk I/O with PID + latency
```

## Disk Space — `df`, `du`, `ncdu`

### Filesystem usage

```bash
df -h                  # human-readable
df -hT                 # with FS type
df -i                  # inodes (often the real culprit when "no space" fires)
df -h --output=source,size,used,avail,pcent,target,fstype
```

### Find big directories

```bash
du -sh /var/* 2>/dev/null | sort -rh | head -20
sudo du -sh /var/log/* 2>/dev/null | sort -rh | head
sudo du -h --max-depth=1 / 2>/dev/null | sort -rh | head -20
```

### Big files

```bash
sudo find / -xdev -type f -size +500M 2>/dev/null -printf '%s %p\n' | \
  sort -nr | head -20 | numfmt --to=iec --field=1
```

### Interactive — `ncdu`

```bash
sudo ncdu /
```

Navigate, mark for delete (`d`), size/itemcount sort (`n`/`s`/`C`).

### Open-but-unlinked files (the silent disk hog)

```bash
sudo lsof +L1 | head             # +L1 = link count < 1 = deleted but held
```

A process keeping a deleted log file open will hold its space until restarted. Restart the process or recreate via `/proc/PID/fd/N`:

```bash
sudo cat /proc/PID/fd/N > /dev/null    # while open, content is recoverable
```

### Inode exhaustion

```bash
df -i
sudo find /var -xdev -type d -printf '%i %p\n' 2>/dev/null | sort -n | uniq -c | sort -rn | head
```

Or count files per directory:

```bash
sudo find /var/spool -xdev -type f 2>/dev/null | awk -F/ '{print $1"/"$2"/"$3}' | sort | uniq -c | sort -rn | head
```

## Disk Errors

### Kernel ring buffer

```bash
sudo dmesg -T | grep -iE 'error|fail|bad|reset|sense|media|i/o|ata|sata|nvme|scsi|ext4|xfs|btrfs'
```

Look for `medium error`, `I/O error`, `ata1.00: status: { DRDY ERR }`, `nvme nvme0: I/O timeout`.

### SMART self-assessment

```bash
sudo smartctl -a /dev/sda
sudo smartctl -a /dev/nvme0
sudo smartctl -H /dev/sda                # one-line PASSED/FAILED
sudo smartctl -t short /dev/sda          # run short self-test
sudo smartctl -l selftest /dev/sda       # self-test history
```

Watch attributes: `Reallocated_Sector_Ct`, `Pending_Sector`, `Offline_Uncorrectable`, `UDMA_CRC_Error_Count`. Any non-zero, climbing = drive failing.

For NVMe:

```bash
sudo smartctl -a /dev/nvme0 | grep -iE 'percentage|critical|media'
sudo nvme smart-log /dev/nvme0
sudo nvme list
```

### Filesystem check

```bash
sudo dmesg -T | grep -i 'EXT4-fs error'
sudo tune2fs -l /dev/sda1 | grep -E 'Filesystem state|Last mount|Mount count'
```

After unmounting:

```bash
sudo umount /dev/sda1
sudo fsck -n /dev/sda1               # read-only check
sudo fsck -y /dev/sda1               # auto-repair
```

## Network — `ss` (socket stats)

`ss` replaces deprecated `netstat`. Faster, more info, talks to kernel via netlink.

### Listening sockets

```bash
ss -tlnp                    # TCP, listening, numeric, with PID/name
ss -ulnp                    # UDP, listening
ss -tunlp                   # TCP+UDP listening
sudo ss -tlnp               # show PID even for other users' sockets
```

Sample:

```
State   Recv-Q  Send-Q   Local Address:Port    Peer Address:Port  Process
LISTEN  0       128            0.0.0.0:22           0.0.0.0:*      users:(("sshd",pid=812,fd=3))
LISTEN  0       4096         127.0.0.1:5432         0.0.0.0:*      users:(("postgres",pid=1023,fd=7))
```

### Established connections

```bash
ss -tan                     # all TCP
ss -tan state established
ss -tan state syn-sent
ss -tan state syn-recv      # SYN flood symptom
ss -tan state time-wait     # client churn
```

Counts by state:

```bash
ss -tan | awk 'NR>1{c[$1]++} END{for(s in c) print s, c[s]}'
```

### Per-process socket count

```bash
ss -tanp | awk '/users/ {gsub(/.*"/,"",$NF); print}' | sort | uniq -c | sort -rn | head
```

### Socket summary

```bash
ss -s
```

```
Total: 412 (kernel 0)
TCP:   158 (estab 102, closed 38, orphaned 0, synrecv 0, timewait 38)
Transport Total     IP        IPv6
*         412       -         -
RAW       1         0         1
UDP       12        8         4
TCP       120       96        24
```

### TCP info — cwnd, rtt, retrans

```bash
ss -ti | head
ss -tin '( dport = :443 or sport = :443 )' | head
```

Sample:

```
ESTAB 0      0   192.168.1.10:50341 142.250.190.78:443
   cubic wscale:7,7 rto:228 rtt:24.137/4.612 ato:40 mss:1460 pmtu:1500
   rcvmss:1460 advmss:1460 cwnd:18 ssthresh:9 bytes_sent:9824
   bytes_retrans:0 segs_out:18 segs_in:14 send 8.7Mbps lastsnd:24
   lastrcv:24 lastack:24 pacing_rate 17.4Mbps delivery_rate 5.2Mbps
   busy:412ms rcv_space:14600 minrtt:21.402
```

`rtt:24.137/4.612` = mean / variance ms. `cwnd:18` segments in flight. `bytes_retrans:0` — clean. Persistent `bytes_retrans` growing = path loss.

### Per-port

```bash
ss -tan '( sport = :80 or sport = :443 )'
sudo ss -tlnp '( sport = :443 )'
```

### Filter by address

```bash
ss -tan dst 10.0.0.1
ss -tan dst 10.0.0.0/24
```

## Network — `sar -n` for nic + tcp

```bash
sar -n DEV 1 5
```

```
IFACE   rxpck/s   txpck/s    rxkB/s    txkB/s   rxcmp/s   txcmp/s  rxmcst/s   %ifutil
eth0    14252.0    9921.0   17182.4    9214.7      0.0      0.0       0.4     14.10
lo        128.0     128.0      32.0      32.0      0.0      0.0       0.0      0.00
```

`%ifutil` = NIC saturation. > 80 = bonding / NIC bottleneck.

### TCP rate + errors

```bash
sar -n TCP,ETCP 1 5
```

```
       active/s passive/s    iseg/s    oseg/s
        12.00     85.00   18250.00   12104.00

       atmptf/s   estres/s  retrans/s isegerr/s   orsts/s
         0.00       0.00      14.20      0.00      4.00
```

`retrans/s` > 1% of `oseg/s` = TCP retransmits hurting throughput. Check the path.

### Other `sar` views

```bash
sar -n SOCK 1 5            # socket counts (tcp,udp,raw,frag)
sar -n IP 1 5              # IP layer
sar -n EIP 1 5             # IP errors
sar -n EDEV 1 5            # NIC errors (rx/tx errs/drops/collisions)
sar -n ICMP,EICMP 1 5      # ICMP
```

### Historical sar (after sysstat collector running)

```bash
sar -A                     # everything from today
sar -n DEV -f /var/log/sa/sa$(date +%d -d yesterday)
sar -n TCP,ETCP -s 14:00:00 -e 15:00:00
```

## Network — `iftop`, `nethogs`, `tcpdump`, `mtr`, `ping`

### `iftop` — bandwidth per flow

```bash
sudo iftop -i eth0
sudo iftop -i eth0 -nNP        # -n no DNS, -N no service names, -P show ports
sudo iftop -i eth0 -F 10.0.0.0/8
```

### `nethogs` — bandwidth per process

```bash
sudo nethogs eth0
sudo nethogs -d 1 eth0
```

### `tcpdump` — packet capture

```bash
sudo tcpdump -i eth0 -nn -c 50 'tcp port 443'
sudo tcpdump -i eth0 -nn 'host 10.0.0.5 and port 443'
sudo tcpdump -i any -nn -w /tmp/capture.pcap 'tcp and (port 80 or port 443)'
sudo tcpdump -i eth0 -nn -A -c 20 'tcp[tcpflags] & tcp-syn != 0'
```

| Filter                          | Purpose                                  |
|---------------------------------|------------------------------------------|
| `host 10.0.0.5`                 | Either direction to this host            |
| `src 10.0.0.5`                  | From host                                |
| `dst 10.0.0.5`                  | To host                                  |
| `port 443`                      | Either side on port                      |
| `dst port 443`                  | Destination port                         |
| `tcp[tcpflags] & tcp-syn != 0`  | Any SYN (incl SYN-ACK)                   |
| `tcp[tcpflags] == tcp-syn`      | Pure SYN (connection start)              |
| `tcp[tcpflags] & tcp-rst != 0`  | RST packets                              |
| `icmp[icmptype] == icmp-echo`   | Pings                                    |
| `not host 10.0.0.5`             | Negate                                   |

Read pcap:

```bash
sudo tcpdump -r /tmp/capture.pcap -nn | head
```

### `mtr` — combined ping + traceroute

```bash
mtr 8.8.8.8
mtr -rwc 10 8.8.8.8       # report mode, 10 cycles, wide
mtr -T -P 443 example.com # TCP probe to port 443
mtr -u -P 53 8.8.8.8      # UDP
```

Look for: jumps in `Loss%` mid-path = upstream issue at that hop. Loss only at final hop = destination filter (often ICMP rate-limiting, not real loss).

### `ping`

```bash
ping -c 4 8.8.8.8
ping -c 100 -i 0.2 8.8.8.8 | tee /tmp/p.txt
ping -M do -s 1472 8.8.8.8       # don't fragment, find PMTU
ping -A 8.8.8.8                  # adaptive — fast as possible
```

### `traceroute`

```bash
traceroute -n 8.8.8.8
traceroute -T -p 443 example.com
traceroute -I 8.8.8.8                # ICMP probe
```

### NIC error counters

```bash
ip -s link
ip -s -s link show eth0          # double -s = extended
ethtool -S eth0 | grep -iE 'err|drop|crc|miss|fifo'
ethtool eth0                     # link, speed, duplex
ethtool -k eth0                  # offload features
ethtool -c eth0                  # interrupt coalescing
ethtool -g eth0                  # ring buffer size
```

If NIC ring buffer full → drops. Increase rx ring:

```bash
sudo ethtool -G eth0 rx 4096
```

## `atop` and `dstat`

### `atop` — full-system snapshots, with history

```bash
atop                       # interactive
atop 1                     # 1-second updates
atop -m                    # memory view first
atop -d                    # disk
atop -n                    # network (needs netatop kmod)
atop -r /var/log/atop/atop_$(date +%Y%m%d)   # replay archived
```

`atop` records every 10 minutes by default to `/var/log/atop/`. Massive after-the-fact value.

```bash
sudo systemctl enable --now atop
sudo systemctl enable --now atopacct          # process accounting
```

Interactive in atop:

| Key | View                              |
|-----|-----------------------------------|
| g   | Generic                           |
| m   | Memory                            |
| d   | Disk                              |
| n   | Network (needs netatop)           |
| s   | Sched                             |
| v   | Various                           |
| t   | Next sample                       |
| T   | Previous sample                   |

### `dstat` — combined live view

```bash
dstat                              # default cpu/disk/net/paging/system
dstat -tcndlmprs 1                 # time, cpu, net, disk, load, mem, paging, proc, swap
dstat --top-cpu --top-mem 1
dstat --tcp 1                      # tcp connection states
dstat --output /tmp/d.csv 1 60     # log to CSV
```

`dstat` is unmaintained on some distros; replacement is `dool` (drop-in fork) or `pcp-dstat`.

### Field cheat for dstat

| Flag           | Shows                                    |
|----------------|------------------------------------------|
| -c             | CPU usr/sys/idl/wai/stl                  |
| -d             | Disk read/writ                           |
| -n             | Net recv/send                            |
| -m             | Mem used/buff/cach/free                  |
| -p             | Procs run/blk/new                        |
| -r             | IO read/write requests                   |
| -s             | Swap used/free                           |
| -l             | Load 1m/5m/15m                           |
| -t             | Time stamp                               |
| --top-cpu      | Top CPU process                          |
| --top-io       | Top IO process                           |
| --top-mem      | Top memory process                       |
| --tcp          | TCP states                               |
| --socket       | Socket counts                            |

## Process Investigation

When you've narrowed to a single PID — go deep.

### `strace` — syscall trace

**Slows the target.** Never on a production-critical PID without a maintenance window.

```bash
sudo strace -p 12345                       # attach
sudo strace -p 12345 -f                    # follow children/threads
sudo strace -p 12345 -e trace=network      # only network syscalls
sudo strace -p 12345 -e trace=read,write
sudo strace -c -p 12345                    # summary on detach (Ctrl-C)
sudo strace -tt -T -p 12345                # timestamps + duration
sudo strace -y -p 12345                    # show fd's path
sudo strace -e trace=openat,close,read,write -p 12345
```

Sample summary:

```
% time     seconds  usecs/call     calls    errors syscall
------ ----------- ----------- --------- --------- ----------------
 71.42    0.082313           4     20512           epoll_wait
 12.31    0.014194           3      4096           writev
 ...
```

If `epoll_wait` dominates with low usecs/call, process is idle. If `read` errors high, look at fd path with `-y`.

### `ltrace` — library calls

```bash
sudo ltrace -p 12345
sudo ltrace -c -p 12345     # summary
```

### `lsof` — open files / sockets

```bash
sudo lsof -p 12345                       # everything PID has open
sudo lsof -p 12345 -nP                   # numeric (no DNS / no port-name)
sudo lsof -i :443                        # who has port 443
sudo lsof -i tcp:443
sudo lsof -i @10.0.0.5                   # any conn to/from host
sudo lsof -u steve                       # all of user steve's open files
sudo lsof /var/log/messages              # who has this file open
sudo lsof +D /var/log                    # everything under directory
sudo lsof +L1                            # deleted but held files
sudo lsof -nP -iTCP -sTCP:LISTEN         # every TCP listener
```

### `/proc/PID/`

```bash
ls /proc/12345/
```

Key files:

```bash
cat /proc/12345/status         # name, state, uid, threads, VM stats, signals
cat /proc/12345/stat           # one-line raw stats
cat /proc/12345/cmdline | tr '\0' ' '; echo
cat /proc/12345/environ | tr '\0' '\n'
cat /proc/12345/limits         # rlimits (open files, processes, mem...)
cat /proc/12345/maps           # virtual memory map (libs, heap, stack)
cat /proc/12345/smaps          # detailed memory per region
cat /proc/12345/wchan          # what kernel function is it sleeping in
cat /proc/12345/stack          # kernel call stack
cat /proc/12345/cgroup         # cgroup membership
cat /proc/12345/mountinfo
cat /proc/12345/numa_maps      # NUMA placement
ls /proc/12345/fd              # open file descriptors
ls -l /proc/12345/fd/0         # stdin's actual target
ls /proc/12345/task            # threads (each is a TID dir)
cat /proc/12345/sched          # scheduler stats
cat /proc/12345/io             # bytes read/written
```

### Show kernel stack of a stuck process

```bash
sudo cat /proc/12345/stack
sudo cat /proc/12345/wchan; echo
```

If `wchan` is `io_schedule_timeout` → blocked on disk; `nfs_wait_on_request` → NFS hang; `futex_wait_queue_me` → waiting on futex (probably a userspace lock).

### `pmap` — process memory map

```bash
pmap 12345                      # summary
pmap -x 12345                   # extended (RSS per region)
pmap -X 12345                   # very detailed
pmap -p 12345 | tail            # numeric paths, total at end
```

## Cgroup-Level

systemd-managed services run in cgroups under `/sys/fs/cgroup/`. Cgroup v2 unified hierarchy is now standard.

### `systemd-cgtop` — top per-cgroup

```bash
systemd-cgtop
systemd-cgtop -d 1
systemd-cgtop --order=cpu
systemd-cgtop --order=memory
systemd-cgtop --order=io
```

### Direct cgroup files (v2)

```bash
ls /sys/fs/cgroup/system.slice/
cat /sys/fs/cgroup/system.slice/nginx.service/cpu.stat
cat /sys/fs/cgroup/system.slice/nginx.service/memory.current
cat /sys/fs/cgroup/system.slice/nginx.service/memory.max
cat /sys/fs/cgroup/system.slice/nginx.service/memory.peak
cat /sys/fs/cgroup/system.slice/nginx.service/memory.events    # oom counts
cat /sys/fs/cgroup/system.slice/nginx.service/io.stat
cat /sys/fs/cgroup/system.slice/nginx.service/pids.current
cat /sys/fs/cgroup/system.slice/nginx.service/pids.max
```

### systemd unit resource view

```bash
systemctl status nginx | head -20
systemd-cgls /system.slice/nginx.service
sudo systemctl set-property nginx.service MemoryMax=2G CPUQuota=200%
```

### Find top mem cgroup

```bash
for d in /sys/fs/cgroup/system.slice/*.service; do
  m=$(cat "$d/memory.current" 2>/dev/null) || continue
  printf "%12d %s\n" "$m" "$(basename "$d")"
done | sort -rn | head -10 | numfmt --to=iec --field=1
```

### Container-specific (Docker)

```bash
docker stats --no-stream
docker stats --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}'
```

## Kernel Tunables

Read the live values:

```bash
sysctl -a 2>/dev/null | grep -E 'vm\.swappiness|vm\.dirty_ratio|fs\.file-max|net\.core\.somaxconn|net\.ipv4\.tcp_max_syn_backlog|net\.ipv4\.ip_local_port_range'
```

### Memory / swap

| Tunable                          | Default       | Effect                                              |
|----------------------------------|---------------|-----------------------------------------------------|
| vm.swappiness                    | 60            | 0–200; higher = more eager to swap                  |
| vm.dirty_ratio                   | 20            | % RAM dirty before sync writeback                   |
| vm.dirty_background_ratio        | 10            | % RAM dirty before bg writeback starts              |
| vm.dirty_expire_centisecs        | 3000          | Pages older than this flushed                       |
| vm.dirty_writeback_centisecs     | 500           | Writeback wakeup interval                           |
| vm.overcommit_memory             | 0             | 0 heuristic, 1 always allow, 2 strict               |
| vm.overcommit_ratio              | 50            | %RAM (only with vm.overcommit_memory=2)             |
| vm.min_free_kbytes               | varies        | Reserve to avoid allocation failure                 |
| vm.vfs_cache_pressure            | 100           | Reclaim dentries/inodes (higher = more)             |
| kernel.numa_balancing            | 1             | Auto-NUMA migration                                 |

### Files / FDs

| Tunable                 | Default   | Effect                                  |
|-------------------------|-----------|-----------------------------------------|
| fs.file-max             | varies    | System-wide max open files              |
| fs.nr_open              | 1048576   | Per-process max open files              |
| fs.inotify.max_user_watches | 8192  | inotify watches/uid                     |
| fs.aio-max-nr           | 65536     | Max async-io requests in flight         |

Per-user/process via `ulimit`:

```bash
ulimit -n              # current soft
ulimit -Hn             # hard
sudo prlimit --pid 12345
sudo prlimit --pid 12345 --nofile=65536:65536
```

systemd unit:

```ini
[Service]
LimitNOFILE=1048576
TasksMax=8192
```

### TCP / sockets

| Tunable                                   | Default       | Effect                              |
|-------------------------------------------|---------------|-------------------------------------|
| net.core.somaxconn                        | 4096 (5.4+)   | Listener accept queue depth         |
| net.core.netdev_max_backlog               | 1000          | Per-cpu netif backlog               |
| net.core.rmem_default / rmem_max          | varies        | Socket recv buffer defaults / cap   |
| net.core.wmem_default / wmem_max          | varies        | Socket send buffer defaults / cap   |
| net.ipv4.tcp_max_syn_backlog              | 1024          | SYN queue depth                     |
| net.ipv4.tcp_syncookies                   | 1             | Defend SYN flood                    |
| net.ipv4.tcp_fin_timeout                  | 60            | TIME_WAIT / FIN_WAIT2 timeout       |
| net.ipv4.tcp_tw_reuse                     | 2 (5.x)       | Reuse TIME_WAIT for outgoing        |
| net.ipv4.tcp_keepalive_time               | 7200          | Idle before keepalive               |
| net.ipv4.tcp_keepalive_intvl              | 75            | Interval between keepalive probes   |
| net.ipv4.tcp_keepalive_probes             | 9             | Probes before declaring dead        |
| net.ipv4.ip_local_port_range              | 32768 60999   | Ephemeral port range                |
| net.ipv4.tcp_rmem                         | 4096 ...      | min/default/max recv (auto)         |
| net.ipv4.tcp_wmem                         | 4096 ...      | min/default/max send (auto)         |
| net.ipv4.tcp_congestion_control           | cubic/bbr     | CC algorithm                        |
| net.ipv4.tcp_notsent_lowat                | inf           | Limit unsent buffer                 |

### Apply tunables

```bash
sudo sysctl -w vm.swappiness=10
sudo sysctl -w net.core.somaxconn=65535

echo 'vm.swappiness=10' | sudo tee /etc/sysctl.d/99-perf.conf
sudo sysctl --system          # reload all conf.d files
```

### NIC ring + offload

```bash
sudo ethtool -G eth0 rx 4096 tx 4096
sudo ethtool -K eth0 gro on tso on gso on
sudo ethtool -C eth0 adaptive-rx on adaptive-tx on
```

### Scheduler — CPU governor

```bash
cpupower frequency-info
sudo cpupower frequency-set -g performance
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor
```

## Real-World Diagnostic Recipes

Each: starting symptom → exact commands → expected output snippet → interpretation.

### Recipe — "Server is slow"

The first-60-seconds sequence. Zero context — one of CPU, mem, disk, net is the culprit.

```bash
uptime
sudo dmesg -T | tail -20
vmstat 1 5
mpstat -P ALL 1 5
pidstat 1 5
iostat -xz 1 5
free -m
sar -n DEV 1 5
sar -n TCP,ETCP 1 5
top -b -n 1 | head -25
```

| Smoking gun in output                | Subsystem     | Next sheet           |
|--------------------------------------|---------------|----------------------|
| `r > nproc` in vmstat                | CPU saturated | CPU recipes          |
| `b > 0` sustained, `wa` high         | Disk          | Disk recipes         |
| `si/so > 0` in vmstat                | Memory/swap   | Memory recipes       |
| `%steal > 5`                         | Hypervisor    | Hosting / move       |
| `retrans/s` high in sar -n ETCP      | Network       | Network recipes      |
| `dmesg` OOM / fail                   | Kernel event  | Specific incident    |

### Recipe — "Process eating CPU"

```bash
top -b -n 1 -o %CPU | head -10
```

Identify PID. Then:

```bash
pidstat -t -p $PID 1 10
```

Sample:

```
   UID       PID    TID    %usr %system  %CPU   Command
  1000     12345    -     97.00    1.00 98.00   compactor
  1000         -  12347   88.00    1.00 89.00    |__compactor
  1000         -  12348    5.00    0.00  5.00    |__compactor
```

TID 12347 is the hot thread.

```bash
sudo perf top -p $PID
```

Sample:

```
Samples: 24K of event 'cycles', Event count (approx.): 18,239,012
  18.42%  compactor  [.] hash_lookup
  12.31%  libc.so    [.] memcpy
   8.92%  compactor  [.] crc32_iso
```

Make a flame graph (depth view):

```bash
sudo perf record -F 99 -p $PID -g -- sleep 30
sudo perf script > /tmp/out.perf
git clone https://github.com/brendangregg/FlameGraph /tmp/FG 2>/dev/null
/tmp/FG/stackcollapse-perf.pl /tmp/out.perf > /tmp/out.folded
/tmp/FG/flamegraph.pl /tmp/out.folded > /tmp/cpu.svg
open /tmp/cpu.svg
```

Interpretation: width of a frame = % time. Wide leaf = the actual hot function. Wide intermediate but narrow leaves = many siblings, optimize the dispatcher.

### Recipe — "OOM kills happening"

```bash
sudo dmesg -T | grep -iE 'killed process|out of memory|oom-kill'
```

Sample:

```
[Mon Apr 25 13:42:11 2026] cgroup-out-of-memory: Killed process 18472 (java)
total-vm:8392132kB, anon-rss:6128304kB, file-rss:0kB, shmem-rss:0kB,
UID:1001 pgtables:12508kB oom_score_adj:0
```

Cgroup OOM (not system OOM). Check cgroup limits:

```bash
ls /sys/fs/cgroup
systemctl show <unit> -p MemoryMax,MemoryHigh,MemoryCurrent,MemoryPeak
cat /sys/fs/cgroup/system.slice/<unit>/memory.events
```

```
oom 5
oom_kill 5
high 0
max 12
low 0
```

Action: raise `MemoryMax`, find leak with `pmap -x` or heap profiler, or set `oom_score_adj=-1000` for critical work.

System-wide OOM (not cgroup):

```bash
free -m
swapon --show
ps aux --sort=-%mem | head
```

### Recipe — "Disk seems slow"

```bash
iostat -xz 1 5
```

Sample:

```
Device   r/s   w/s  rkB/s  wkB/s aqu-sz r_await w_await %util
sda    14.20 102.0  227.2 16384  4.21    11.32   42.80  98.4
```

`%util=98.4` and `w_await=42.80 ms` on a spinning disk = saturated and slow. Or, if it's an SSD/NVMe, > 10 ms `await` is unusually high.

Per-process:

```bash
sudo iotop -b -o -n 5 -d 1
sudo biotop 5             # bcc
```

Drive health:

```bash
sudo smartctl -a /dev/sda | grep -iE 'reallocated|pending|crc|uncorrectable'
sudo dmesg -T | grep -iE 'sda|ata|nvme|i/o error'
```

If clean drives but slow: check filesystem (XFS journal contention, ext4 jbd2), scheduler (`cat /sys/block/sda/queue/scheduler`), or fragmentation.

```bash
cat /sys/block/sda/queue/scheduler            # current is in [brackets]
echo mq-deadline | sudo tee /sys/block/sda/queue/scheduler
```

### Recipe — "Connection refused"

```bash
sudo ss -tlnp '( sport = :443 )'
```

If empty → service isn't listening.

```bash
sudo systemctl status nginx
sudo journalctl -u nginx -n 50
```

If listening but still refused remote:

```bash
sudo firewall-cmd --list-all                   # firewalld
sudo iptables -L -n -v | head                  # legacy
sudo nft list ruleset | head                   # nftables
sudo ufw status                                # ubuntu
```

Then routes:

```bash
ip route
ip route get 10.0.0.5
ip rule
```

And SELinux / AppArmor:

```bash
sudo getenforce
sudo journalctl -t setroubleshoot --since "1 hour ago"
sudo aa-status
```

Test from another host:

```bash
nc -zv server.host 443
curl -v telnet://server.host:443
```

### Recipe — "Connection timed out"

(Different from refused — refused = RST, timeout = no response.)

```bash
mtr -rwc 20 server.host
```

Look for hop with sustained loss. If only the last hop loses but real traffic still reaches, ICMP filter — not a real loss.

```bash
sudo tcpdump -i eth0 -nn host server.host -c 30
```

Watch for retransmitted SYN with no SYN-ACK — silent drop somewhere.

```bash
sudo traceroute -T -p 443 server.host
```

PMTU black hole?

```bash
ping -c 4 -M do -s 1472 server.host
ping -c 4 -M do -s 1452 server.host
```

If 1472 fails but 1452 works, fragmentation needed but blocked. Workaround:

```bash
sudo ip route change <route> mtu 1400
```

### Recipe — "Swap thrashing"

```bash
vmstat 1 | head -1; vmstat 1 | awk '$7>0 || $8>0'
```

Sample:

```
 r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
 4  2  524288  10212    104   2123 8421 12042  9032 12508 4023 8410  4  6 22 67  0
```

`si=8421 so=12042` → constantly paging in and out. `wa=67%` waiting for those page operations. CPU is mostly waiting.

Reduce swap pressure:

```bash
sudo sysctl -w vm.swappiness=10
sudo swapoff -a && sudo swapon -a            # if you have RAM headroom
```

Find the consumer:

```bash
ps -eo pid,comm,rss,vsz --sort=-rss | head -10
```

Long-term: add RAM, fix the leak, reduce working set, or set per-cgroup memory limits to keep one service from blowing out cache.

### Recipe — "fork: Resource temporarily unavailable"

PID exhaustion or `LimitNOFILE` blown:

```bash
ps -eLf | wc -l                           # running threads
cat /proc/sys/kernel/threads-max
sysctl kernel.pid_max
ulimit -u                                 # max user processes
sudo prlimit --pid $PID
```

```bash
ls /proc/$PID/fd | wc -l
cat /proc/$PID/limits | head -10
```

Increase limits in systemd unit:

```ini
[Service]
TasksMax=infinity
LimitNPROC=65535
LimitNOFILE=1048576
```

### Recipe — "Too many TIME_WAIT"

```bash
ss -tan state time-wait | wc -l
```

If client churn (e.g., short-lived HTTP), enable reuse and shrink fin timeout:

```bash
sudo sysctl -w net.ipv4.tcp_tw_reuse=1
sudo sysctl -w net.ipv4.tcp_fin_timeout=15
```

For server side, ensure ephemeral port pool is wide enough:

```bash
sudo sysctl -w net.ipv4.ip_local_port_range="10000 65000"
```

Don't enable `tcp_tw_recycle` (removed in 4.12; was always dangerous behind NAT).

### Recipe — "Conntrack table full"

```bash
sudo dmesg -T | grep -i 'nf_conntrack'
cat /proc/sys/net/netfilter/nf_conntrack_count
cat /proc/sys/net/netfilter/nf_conntrack_max
```

Sample dmesg:

```
nf_conntrack: nf_conntrack: table full, dropping packet
```

Raise:

```bash
sudo sysctl -w net.netfilter.nf_conntrack_max=1048576
echo 'net.netfilter.nf_conntrack_max=1048576' | sudo tee /etc/sysctl.d/99-conntrack.conf
```

Drop unnecessary conntrack:

```bash
sudo iptables -t raw -A PREROUTING -p tcp --dport 80 -j NOTRACK
```

### Recipe — "Too many retransmits"

```bash
sar -n ETCP 1 5
nstat -az TcpRetrans* TcpExt*
```

Per-connection:

```bash
ss -tin | awk '/retrans/{print}' | head
```

If only a single peer: path issue (mtr that peer). If broad: NIC / driver / queue issue:

```bash
ethtool -S eth0 | grep -iE 'drop|err|miss'
ip -s link show eth0
sudo dmesg -T | grep -iE 'eth0|nic'
```

### Recipe — "Container memory throttle"

```bash
docker stats --no-stream
cat /sys/fs/cgroup/.../memory.events
```

Sample:

```
low 0
high 14025
max 3
oom 3
oom_kill 3
```

`high` = throttled at memory.high; `oom_kill` = killed inside cgroup. Raise `--memory` or `MemoryMax`, fix leak.

### Recipe — "NUMA imbalance"

```bash
numactl --hardware
numactl --show
numastat -m | head -30
numastat -p $PID
```

If allocations on remote node:

```bash
sudo systemctl set-property myapp.service NUMAPolicy=bind NUMAMask=0
numactl --membind=0 --cpunodebind=0 my_app
```

### Recipe — "Random DNS failure"

```bash
dig @1.1.1.1 example.com +noall +stats
sudo tcpdump -i any -nn 'udp port 53'
sudo journalctl -u systemd-resolved -n 100
sudo journalctl -u nscd -n 100
sudo journalctl -u dnsmasq -n 100
```

Conntrack drops on UDP/53 also fit here. And `nss-resolve` ordering in `/etc/nsswitch.conf`.

### Recipe — "Time skew"

```bash
timedatectl status
chronyc tracking
chronyc sources -v
ntpq -p           # if ntpd
```

Symptoms: TLS cert errors out of nowhere, Kerberos auth failures, distributed-system clock issues.

```bash
sudo chronyc -a 'burst 4/4'
sudo chronyc -a makestep
```

### Recipe — "kworker pegged at 100%"

```bash
top -b -n 1 -o %CPU | grep kworker | head
```

Find the work source via tracing:

```bash
sudo perf top -e cycles -g
sudo perf sched record -- sleep 10 && sudo perf sched latency
sudo bpftrace -e 'kprobe:wakeup_kworker { @[kstack] = count(); }'
```

Common culprits: bcache flush, md raid resync, ACPI events, broken driver.

```bash
cat /proc/mdstat
cat /sys/block/bcache0/bcache/state
sudo dmesg -T | grep -iE 'acpi|firmware'
```

## Performance Tuning vs Triage

Distinct disciplines.

| Triage                         | Tuning                              |
|--------------------------------|-------------------------------------|
| Find the bottleneck            | Improve a known bottleneck          |
| Time-pressured (live incident) | Engineering project                 |
| Tools above                    | Plus profilers, BPF, benchmarks     |
| Output: a hypothesis           | Output: a configuration / patch     |
| Risk: misdiagnosis             | Risk: regressions in other workloads|

The mistake is to start tuning before triaging. **Don't change a knob until USE points at the resource it controls.**

When triage points at CPU → tune scheduler / governor / process pinning, build flame graphs, profile.

When triage points at memory → measure working set (`ps -o rss`, smaps), tune `vm.swappiness`, NUMA placement, hugepages, or fix the leak.

When triage points at disk → measure with `fio`, tune scheduler / queue depth / read-ahead, consider hardware refresh.

When triage points at network → measure with `iperf3`, tune `tcp_rmem`/`tcp_wmem`, congestion control (`bbr` for high-BDP paths), NIC ring + offloads.

## Idioms

### `ps` shortcuts

```bash
alias psm='ps aux --sort=-%mem | head -15'      # top mem
alias psc='ps aux --sort=-%cpu | head -15'      # top cpu
alias pst='ps -eLf | wc -l'                     # total threads
```

### Count threads per process

```bash
ps -eo nlwp,pid,comm --sort=-nlwp | head
```

### `watch` snapshots

```bash
watch -n 1 'free -m; echo; vmstat 1 2 | tail -1'
watch -n 2 'ss -s'
watch -n 1 'cat /proc/loadavg; cat /proc/sys/fs/file-nr'
```

### Snapshot capture pattern

```bash
TS=$(date +%Y%m%d-%H%M%S); D=/tmp/snap-$TS; mkdir -p "$D"
{ date; uname -a; uptime; } > "$D/0_meta.txt"
sudo dmesg -T > "$D/1_dmesg.txt"
vmstat 1 5 > "$D/2_vmstat.txt"
mpstat -P ALL 1 5 > "$D/3_mpstat.txt"
pidstat 1 5 > "$D/4_pidstat.txt"
iostat -xz 1 5 > "$D/5_iostat.txt"
free -m > "$D/6_free.txt"
sar -n DEV 1 5 > "$D/7_sar_dev.txt"
sar -n TCP,ETCP 1 5 > "$D/8_sar_tcp.txt"
top -b -n 1 > "$D/9_top.txt"
ss -s > "$D/A_ss.txt"
ss -tan > "$D/B_ss_tan.txt"
ps auxf > "$D/C_psauxf.txt"
echo "Snapshot at $D"
tar czf "$D.tar.gz" -C /tmp "snap-$TS"
echo "Bundle: $D.tar.gz"
```

Save the function:

```bash
perf_snapshot() {
  local TS=$(date +%Y%m%d-%H%M%S) D=/tmp/snap-$TS
  mkdir -p "$D"
  uptime > "$D/up.txt"
  vmstat 1 5 > "$D/vm.txt" &
  mpstat -P ALL 1 5 > "$D/mp.txt" &
  iostat -xz 1 5 > "$D/io.txt" &
  sar -n DEV 1 5 > "$D/dev.txt" &
  wait
  free -m > "$D/free.txt"
  top -b -n 1 > "$D/top.txt"
  echo "Snapshot: $D"
}
```

### Always check the obvious

- Is it DNS? (It's always DNS.)
- Did you check `dmesg`?
- Did you check disk space (`df -h`, `df -i`)?
- Did you check time skew (`timedatectl`)?
- Did you check certs expiry?
- Is it a recent deploy? (`git log -1`, `journalctl --since`)
- Is the change in the last 24h? (`find /etc -mtime -1`)

### Quick capture of what changed

```bash
sudo find /etc -type f -mtime -1 2>/dev/null
journalctl --since "1 hour ago" --priority=err
last reboot | head
```

### Compare host to baseline

```bash
diff <(ssh good-host 'sysctl -a 2>/dev/null' | sort) \
     <(sudo sysctl -a 2>/dev/null | sort) | head -50
```

### Don't trust averages

```bash
sudo bpftrace -e 'tracepoint:block:block_rq_complete /args->bytes/ {
   @us[args->dev] = hist((nsecs - @t[args->dev])/1000);
}'
```

p99 disk latency > 100× p50 = bursty workload, smooth average misleads.

### `vmtouch` — see file cache

```bash
sudo vmtouch -v /var/lib/postgres/data
sudo vmtouch -t /var/lib/postgres/data         # pre-warm
sudo vmtouch -e /var/lib/postgres/data         # evict
```

### `pcstat` — page-cache state

```bash
pcstat /var/log/messages /etc/hosts
```

### `numastat`

```bash
numastat
numastat -m | head -30
numastat -p $(pidof postgres)
```

### `chrt` — set scheduling class

```bash
chrt -p $PID                                   # show
sudo chrt -f -p 50 $PID                        # SCHED_FIFO prio 50
sudo chrt -r -p 50 $PID                        # SCHED_RR
sudo chrt -i -p 0 $PID                         # SCHED_IDLE
```

### `taskset` — CPU pin

```bash
taskset -pc $PID                               # show affinity
sudo taskset -pc 2,3 $PID                      # pin to CPU 2 and 3
sudo taskset -c 0-3 my_command                 # launch pinned
```

### `ionice` — IO priority

```bash
ionice -p $PID                                 # show
sudo ionice -c 3 -p $PID                       # idle class
sudo ionice -c 2 -n 0 my_command               # best-effort highest
```

### `nice` / `renice`

```bash
nice -n 19 my_command                          # lowest priority
sudo renice -n -5 -p $PID                      # raise priority
```

## Tips

### Read in this order under pressure

1. Load avg (`uptime`) — context.
2. `dmesg | tail` — kernel events you'd otherwise miss.
3. `vmstat 1 5` — CPU + mem + swap + IO + steal in one screen.
4. `mpstat -P ALL 1` if CPU implicated.
5. `iostat -xz 1` if `wa` or `b` non-zero.
6. `free -m` if mem implicated.
7. `sar -n DEV 1`, `sar -n TCP,ETCP 1` if network implicated.
8. `top` for process attribution.

### Save baselines

Run the snapshot pattern weekly on a healthy host. When it goes bad, diff.

### Beware tool overhead

`strace`, `tcpdump` (on busy NICs), and `perf record -F 999` on a hot box can themselves be the trigger. Always have a kill plan (`Ctrl-C`, `kill -TERM`).

### Don't change two things at once

If you tune `swappiness` AND raise `LimitNOFILE` AND change CC, you can't attribute the result. One change per measurement window.

### Container surprises

- `top` inside a container shows host RAM/CPU — you're seeing the host kernel. Use `cgroup` files (or `docker stats` from outside).
- `/proc/meminfo` inside the container is the host's. lxcfs / fakeproc projects exist; or just look at cgroup.
- `dmesg` is the host kernel; in unprivileged containers may be empty.

### When `top` itself eats CPU

`top` reads `/proc/PID/*` for every PID every interval. On a host with 100k PIDs (e.g., container fleet) that's heavy. Use `top -p PID1,PID2,...` or `pidstat -p PID 1`.

### Logging is performance

Log to ramdisk + ship; don't sync per-line on a hot path. `journalctl` rate-limit:

```bash
sudo journalctl --disk-usage
sudo vacuumctl                                 # if available
sudo journalctl --vacuum-time=2d
```

### Profile, don't guess

When you've narrowed to "this binary is hot," use perf or bpftrace. Don't optimize on intuition — flame graph.

### Use `--help` aggressively

`vmstat --help`, `iostat --help`, `mpstat --help`, `pidstat --help`, `sar --help` all print usable summaries faster than re-reading the man page.

### Keep a triage notebook

Per-host (or in chatops): `date`, symptom, hypothesis, command run, output snippet, diagnosis, fix, follow-up. Patterns emerge.

### Production readiness

Before declaring an incident over:

- Working set fits in RAM (`free -m available` stable).
- p99 latency back to baseline (RED).
- No new errors in `dmesg` or `journalctl --priority=err`.
- All resources < 60% utilization (USE).
- 30-min stable trend.

## See Also

- perf
- bpftrace
- bpftool
- ebpf
- flamegraph
- polyglot
- bash

## References

- Brendan Gregg, "Linux Performance Analysis in 60,000 milliseconds" — Netflix Tech Blog, https://netflixtechblog.com/linux-performance-analysis-in-60-000-milliseconds-accc10403c55
- Brendan Gregg, "Linux Performance" landing page — https://www.brendangregg.com/linuxperf.html
- Brendan Gregg, "Systems Performance: Enterprise and the Cloud" (2nd ed., Pearson, 2020)
- Brendan Gregg, "BPF Performance Tools" (Addison-Wesley, 2019) — https://www.brendangregg.com/bpf-performance-tools-book.html
- Brendan Gregg, "The USE Method" — https://www.brendangregg.com/usemethod.html
- Tom Wilkie, "The RED Method: How To Instrument Your Services" — https://grafana.com/blog/2018/08/02/the-red-method-how-to-instrument-your-services/
- Linux man-pages project — https://man7.org/linux/man-pages/
- `man 1 top`, `man 1 vmstat`, `man 1 iostat`, `man 1 mpstat`, `man 1 pidstat`, `man 1 sar`, `man 1 free`, `man 1 ss`, `man 1 lsof`, `man 1 strace`, `man 1 perf`, `man 5 proc`, `man 7 cgroups`, `man 8 sysctl`
- Documentation/admin-guide/sysctl/ in the kernel source tree — https://www.kernel.org/doc/html/latest/admin-guide/sysctl/index.html
- "Linux Insides" — https://0xax.gitbooks.io/linux-insides/
- Julia Evans, "Linux performance" zines and posts — https://jvns.ca/
- Tanel Poder, "0x.tools" — https://0x.tools/
- bcc — https://github.com/iovisor/bcc
- bpftrace — https://github.com/iovisor/bpftrace
- FlameGraph — https://github.com/brendangregg/FlameGraph
- perf-tools — https://github.com/brendangregg/perf-tools
- atop — https://www.atoptool.nl/
- sysstat — https://github.com/sysstat/sysstat
- The Netflix-Skunkworks "linux-perf-out-of-the-box" snapshot script
- "Brendan Gregg's Performance Checklists for SREs" (SREcon talk archives)
