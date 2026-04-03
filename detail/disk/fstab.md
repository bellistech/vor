# The Mathematics of fstab — Filesystem Table Internals

> *fstab maps devices to mount points with options. The math covers mount ordering dependencies, performance tuning parameters, swap priority weighting, and tmpfs memory allocation.*

---

## 1. fstab Entry Structure — The Six Fields

### The Format

```
<device>  <mountpoint>  <type>  <options>  <dump>  <pass>
```

### Field Semantics and Types

| Field | Type | Values | Purpose |
|:---|:---|:---|:---|
| device | String | UUID=, LABEL=, /dev/sdX, path | Block device or remote |
| mountpoint | Path | /home, /data, none (swap) | Where to attach |
| type | Enum | ext4, xfs, btrfs, nfs, swap, tmpfs | Filesystem type |
| options | CSV | defaults, noatime, ro, ... | Mount flags |
| dump | Int | 0 or 1 | Backup flag (obsolete) |
| pass | Int | 0, 1, or 2 | fsck order |

---

## 2. Mount Order and fsck Pass — Dependency Graph

### The Model

The `pass` field determines the order of filesystem checks at boot. This creates a dependency graph.

### fsck Pass Rules

$$\text{fsck order} = \begin{cases} \text{skip} & \text{if pass} = 0 \\ \text{first (serial)} & \text{if pass} = 1 \\ \text{second (parallel)} & \text{if pass} = 2 \end{cases}$$

| Pass | Behavior | Typical Use |
|:---:|:---|:---|
| 0 | No fsck | Network mounts, swap, tmpfs |
| 1 | Check first, alone | Root filesystem only |
| 2 | Check after pass 1, in parallel | All other local filesystems |

### Parallel fsck Time

$$T_{fsck} = T_{pass1} + \max_{i \in pass2}(T_i)$$

Where $T_i$ = check time for filesystem $i$. Pass 2 entries on different disks run in parallel.

### Mount Order at Boot

systemd constructs a dependency tree from fstab. Mount order follows path depth:

$$\text{Order} = \text{depth}(\text{mountpoint})$$

`/` must mount before `/home`, which must mount before `/home/user/data`.

---

## 3. Performance Options — The Numbers Behind the Flags

### noatime vs relatime vs atime

$$\text{Write Overhead per Read} = \begin{cases} 1 \text{ metadata write} & \text{atime (default)} \\ 1 \text{ write if atime < mtime} & \text{relatime} \\ 0 \text{ writes} & \text{noatime} \end{cases}$$

### IOPS Impact

For a workload of $R$ reads/second:

$$\text{Extra IOPS}_{atime} = R$$

$$\text{Extra IOPS}_{relatime} \approx 0.01 \times R \quad (\text{~1\% of reads trigger update})$$

$$\text{Extra IOPS}_{noatime} = 0$$

| Reads/sec | atime IOPS overhead | relatime | noatime |
|:---:|:---:|:---:|:---:|
| 100 | 100 | 1 | 0 |
| 1,000 | 1,000 | 10 | 0 |
| 10,000 | 10,000 | 100 | 0 |
| 100,000 | 100,000 | 1,000 | 0 |

### commit= Option (ext4/btrfs)

Controls how often dirty data is flushed to disk:

$$\text{Max Data Loss Window} = \text{commit interval (seconds)}$$

| commit= | Flush Interval | Risk Window | Performance |
|:---:|:---:|:---:|:---|
| 1 | 1 second | 1 sec data loss | Worst (many syncs) |
| 5 (default) | 5 seconds | 5 sec data loss | Default balance |
| 30 | 30 seconds | 30 sec data loss | Better throughput |
| 60 | 60 seconds | 60 sec data loss | Best throughput |

### barrier= Option

$$\text{Write Order Guarantee} = \begin{cases} \text{Guaranteed} & \text{barrier=1 (default)} \\ \text{Not guaranteed} & \text{barrier=0 (dangerous)} \end{cases}$$

Disabling barriers increases write throughput by ~20-30% but risks data corruption on power loss.

---

## 4. Swap Priority — Weighted Distribution

### The Model

Multiple swap entries can have different priorities. Higher priority = used first. Same priority = round-robin (striped).

### Priority Rules

$$\text{Swap Selection} = \begin{cases} \text{Use highest priority first} & \text{if priorities differ} \\ \text{Round-robin across all} & \text{if priorities equal} \end{cases}$$

### Worked Example

```
/dev/sda2  none  swap  pri=100   0  0
/dev/sdb2  none  swap  pri=100   0  0
/dev/sdc2  none  swap  pri=10    0  0
```

**Behavior:** sda2 and sdb2 are used in parallel (striped, both pri=100). sdc2 is only used when both are full.

### Striped Swap Throughput

$$\text{Swap BW}_{striped} = n \times \text{Single Disk BW}$$

| Configuration | Priority | Effective BW |
|:---|:---:|:---:|
| 1 HDD swap | - | 150 MB/s |
| 2 HDD swap (striped) | Same | 300 MB/s |
| 1 SSD swap | - | 500 MB/s |
| SSD (pri=100) + HDD (pri=10) | Tiered | 500 MB/s (then 150) |

### Swap Size Recommendations

$$\text{Swap Size} = \begin{cases} \text{RAM} \times 2 & \text{if RAM} \leq 2 \text{ GiB} \\ \text{RAM} + 2 \text{ GiB} & \text{if RAM} \leq 8 \text{ GiB} \\ \text{RAM} \times 1.5 & \text{if hibernate needed} \\ 8 \text{ GiB (fixed)} & \text{if RAM} > 16 \text{ GiB, no hibernate} \end{cases}$$

---

## 5. tmpfs Sizing — RAM-Backed Filesystems

### The Model

tmpfs uses RAM (and swap when needed). It has a configurable size limit.

### Size Formula

$$\text{tmpfs Max} = \begin{cases} \frac{\text{RAM}}{2} & \text{default (no size= option)} \\ \text{specified size} & \text{size=2G} \\ \text{RAM} \times \text{fraction} & \text{size=50\%} \end{cases}$$

### Memory Pressure

$$\text{Available for Apps} = \text{RAM} - \text{tmpfs Used} - \text{Kernel}$$

$$\text{OOM Risk} = \frac{\text{tmpfs Used} + \text{App Memory}}{\text{RAM}} > 1.0$$

### Common tmpfs Entries

| Mountpoint | Typical Size | Purpose |
|:---|:---:|:---|
| /tmp | 50% RAM or 2 GiB | Temporary files |
| /run | 10% RAM | Runtime state (PIDs, sockets) |
| /dev/shm | 50% RAM | POSIX shared memory |
| /run/lock | 5 MiB | Lock files |

### Worked Example

*"32 GiB RAM server, /tmp=4G, /dev/shm=default, /run=10%."*

$$\text{/tmp max} = 4 \text{ GiB}$$

$$\text{/dev/shm max} = 16 \text{ GiB (50\% of 32)}$$

$$\text{/run max} = 3.2 \text{ GiB}$$

$$\text{Worst case tmpfs usage} = 4 + 16 + 3.2 = 23.2 \text{ GiB}$$

$$\text{Remaining for apps} = 32 - 23.2 = 8.8 \text{ GiB}$$

---

## 6. NFS Mount Options — Network Tuning

### rsize/wsize Impact

$$\text{NFS Throughput} \approx \frac{\text{rsize}}{\text{RTT} + T_{server}}$$

| rsize/wsize | Packets per MiB | Throughput (1ms RTT) |
|:---:|:---:|:---:|
| 8 KiB | 128 | ~8 MB/s |
| 32 KiB | 32 | ~31 MB/s |
| 64 KiB | 16 | ~60 MB/s |
| 1 MiB | 1 | ~500 MB/s |

### NFS Timeout Formula

$$\text{Total Timeout} = \text{timeo} \times \sum_{i=0}^{\text{retrans}} 2^i = \text{timeo} \times (2^{\text{retrans}+1} - 1)$$

Default: timeo=600 (60 seconds), retrans=3:

$$\text{Total} = 60 \times (2^4 - 1) = 60 \times 15 = 900 \text{ seconds} = 15 \text{ minutes}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $R \times \text{write overhead}$ | Linear scaling | atime IOPS cost |
| $n \times \text{BW}$ | Linear scaling | Striped swap throughput |
| $\frac{\text{RAM}}{2}$ | Fraction | tmpfs default sizing |
| $\text{timeo} \times (2^{r+1}-1)$ | Geometric series | NFS timeout |
| $\frac{\text{rsize}}{\text{RTT}}$ | Rate equation | NFS throughput |

---

*Every line in /etc/fstab is parsed by mount(8) at boot — six whitespace-separated fields that determine how your entire storage topology is assembled.*
