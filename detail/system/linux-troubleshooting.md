# Linux Troubleshooting — Theory and Methodology

> *Effective troubleshooting is a disciplined process, not guesswork. Understanding the frameworks, decision trees, and analysis methodologies transforms random poking into systematic root cause identification.*

---

## 1. Troubleshooting Framework

### The Scientific Method Applied

Every troubleshooting engagement follows the same fundamental cycle:

```
IDENTIFY → ANALYZE → HYPOTHESIZE → TEST → VERIFY → DOCUMENT
    ↑                                         |
    └─────────── If not resolved ─────────────┘
```

### Identify Phase

**Critical questions:**
1. What is the **exact** symptom? (Not "it's broken" — what specific behavior?)
2. When did it **start**? (Correlate with changes)
3. What is the **scope**? (One user? All users? One service? All services?)
4. Is it **reproducible**? (Every time? Intermittent? Under load?)
5. What **changed**? (Deployments, config changes, OS updates, hardware)

### Analyze Phase — Data Collection Priority

Collect data in order of **least invasive** to **most invasive**:

$$\text{Priority} = \frac{diagnostic\_value}{service\_impact}$$

| Priority | Method | Impact |
|:---|:---|:---|
| 1 | Read logs (journalctl, /var/log) | None |
| 2 | Check metrics (top, vmstat, iostat) | None |
| 3 | Query state (ss, ip, systemctl) | None |
| 4 | Trace (strace, tcpdump, perf) | Minor overhead |
| 5 | Reproduce in test environment | None on production |
| 6 | Change configuration | Possible disruption |
| 7 | Restart services | Service interruption |

### Occam's Razor in Troubleshooting

$$P(simple\_cause) \gg P(complex\_cause)$$

Always check the simplest explanations first:
- Is the service running?
- Is the network cable plugged in?
- Is the disk full?
- Did someone change a config file?
- Was there a recent deployment?

---

## 2. Boot Failure Decision Tree

### Boot Sequence Stages

```
Power On
  ↓
UEFI/BIOS → POST
  ↓ (failure: no display, beep codes)
Bootloader (GRUB2)
  ↓ (failure: grub> or grub rescue> prompt)
Kernel load + initramfs
  ↓ (failure: kernel panic, dracut shell)
systemd PID 1 (initrd.target → switch-root)
  ↓ (failure: emergency.target, sulogin)
sysinit.target → basic.target
  ↓ (failure: dependency failures, stuck mounts)
multi-user.target or graphical.target
  ↓ (failure: specific service failures)
Login prompt
```

### Decision Tree

```
System doesn't boot
├── No display at all
│   └── Hardware: PSU, RAM, CPU, POST failure
├── GRUB prompt appears
│   ├── grub> → GRUB found but config missing
│   │   └── Manually boot: set root, linux, initrd, boot
│   └── grub rescue> → GRUB core found, modules missing
│       └── set prefix, insmod normal, normal
├── Kernel panic
│   ├── "not syncing: VFS: Unable to mount root fs"
│   │   └── Wrong root= parameter, missing initramfs, or missing driver
│   ├── "not syncing: Attempted to kill init"
│   │   └── Corrupted /sbin/init or missing libraries
│   └── General kernel panic
│       └── Hardware issue, kernel bug, corrupted kernel image
├── Dracut emergency shell
│   └── initramfs couldn't mount root — check fstab, LVM, LUKS, multipath
├── systemd emergency.target
│   └── Critical unit failed — check journalctl -b, systemctl --failed
└── Hangs during boot
    ├── Check: systemd-analyze blame (which unit?)
    ├── Network timeout: systemd-networkd-wait-online or NetworkManager-wait-online
    └── Mount timeout: bad fstab entry (remove or add nofail)
```

### Root Cause Categories

| Stage | Common Causes |
|:---|:---|
| GRUB | Corrupted MBR/GPT, missing grub.cfg, wrong partition UUIDs |
| Kernel | Missing initramfs, wrong root= param, missing drivers |
| initrd | LVM not activated, LUKS key missing, multipath not assembled |
| systemd early | Bad fstab entry, corrupted filesystem, SELinux relabel needed |
| systemd late | Service dependency loop, failed mount, network timeout |

---

## 3. Filesystem Recovery Theory

### Filesystem Consistency

A filesystem is **consistent** when all metadata structures agree:

$$\text{consistent} \iff \begin{cases} \text{superblock valid} \\ \text{inode table intact} \\ \text{block bitmap matches allocations} \\ \text{directory entries point to valid inodes} \\ \text{journal replayed (if journaled)} \end{cases}$$

### Journal Recovery

Journaled filesystems (ext4, XFS) maintain a **write-ahead log**:

```
Transaction flow:
1. Write metadata changes to journal
2. Commit journal transaction
3. Write metadata to actual location
4. Mark journal transaction complete
```

On crash recovery:
- **Complete transactions** in journal → replay to filesystem
- **Incomplete transactions** → discard (data written after journal but before commit is lost)

### ext4 vs XFS Recovery

| Aspect | ext4 (fsck) | XFS (xfs_repair) |
|:---|:---|:---|
| Journal replay | Automatic on mount | Automatic on mount |
| Offline repair | `fsck.ext4 -y` | `xfs_repair` |
| Force check | `fsck.ext4 -f` | `xfs_repair -n` (dry run) |
| Dirty log | Auto-replay | `xfs_repair -L` (clears log, data loss risk) |
| Online check | `tune2fs -l` (limited) | `xfs_scrub` (Linux 4.20+) |
| Superblock backup | Multiple copies | Secondary superblock at AG boundaries |

### Data Loss Hierarchy

$$\text{metadata loss} > \text{journal loss} > \text{data loss}$$

Metadata loss (inode table, superblock) is catastrophic and may require backup restoration. Journal loss means recent transactions may be lost. Data loss in individual files is recoverable if metadata is intact.

---

## 4. Network Troubleshooting Layers

### OSI Layer Mapping to Linux Tools

| Layer | Name | What Fails | Tools |
|:---|:---|:---|:---|
| L1 | Physical | Cable, NIC, speed/duplex | `ethtool`, `ip link`, `dmesg` |
| L2 | Data Link | MAC, VLAN, ARP, switching | `ip neigh`, `bridge`, `tcpdump -e` |
| L3 | Network | IP, routing, MTU, fragmentation | `ip addr/route`, `ping`, `traceroute`, `mtr` |
| L4 | Transport | TCP/UDP, ports, firewalls, connection state | `ss`, `tcpdump`, `nftables`, `conntrack` |
| L5-6 | Session/Presentation | TLS, encoding | `openssl s_client`, `curl -v` |
| L7 | Application | HTTP, DNS, app logic | `curl`, `dig`, `strace`, app logs |

### Systematic Approach

**Always start at Layer 1 and work up.** A L3 problem cannot be solved if L1 is down.

```
L1: Is the link UP? (ip link show)
 ↓ YES
L2: Can we ARP to the gateway? (arping)
 ↓ YES
L3: Can we ping the gateway? (ping)
 ↓ YES
L3: Can we ping the destination? (ping, traceroute)
 ↓ YES
L4: Can we connect to the port? (ss, curl, telnet)
 ↓ YES
L7: Does the application respond correctly? (curl -v, app logs)
```

### MTU/Fragmentation Issues

Path MTU Discovery (PMTUD) failures are a common cause of mysterious connectivity issues:

$$\text{if } packet\_size > path\_MTU \text{ and DF bit set} \implies \text{ICMP "Fragmentation Needed"}$$

If ICMP is blocked (bad firewall), the sender never learns about the MTU limit — connections **hang** after the TCP handshake (small packets work, large data transfers stall).

**Test:** `ping -M do -s 1472 destination` (1472 + 28 byte header = 1500 MTU)

---

## 5. Service Dependency Analysis

### systemd Dependency Model

systemd units have two types of ordering:

| Directive | Meaning |
|:---|:---|
| `After=` / `Before=` | **Ordering** — when to start (not whether) |
| `Requires=` / `Wants=` | **Dependency** — what must also run |
| `BindsTo=` | **Strong dependency** — stop if dependency stops |
| `PartOf=` | **Group** — restart/stop with parent |
| `Conflicts=` | **Mutual exclusion** — stop the other |

**Common mistake:** Using `After=` without `Requires=` only controls ordering, not dependency. The service won't wait for the dependency to exist.

### Dependency Debugging

```
Service fails to start
├── Check: systemctl status unit
│   └── Shows exit code, recent logs
├── Check: journalctl -u unit -b
│   └── Full logs since boot
├── Check: systemctl list-dependencies unit
│   └── Shows dependency tree
├── Check: systemctl list-dependencies --reverse unit
│   └── Shows what depends on this unit
└── Check: systemd-analyze verify unit
    └── Validates unit file syntax
```

### Target Units as Synchronization Points

```
sysinit.target      — early boot (udev, /proc, /sys)
    ↓
basic.target        — base system (sockets, timers, paths)
    ↓
network.target      — network interfaces configured
    ↓
network-online.target — network actually reachable
    ↓
multi-user.target   — full multi-user system
    ↓
graphical.target    — display manager
```

---

## 6. Performance Analysis Methodology

### USE Method (Brendan Gregg)

For every **resource**, check **Utilization, Saturation, and Errors**:

$$\text{USE} = \{U(\text{tilization}), S(\text{aturation}), E(\text{rrors})\}$$

| Resource | Utilization | Saturation | Errors |
|:---|:---|:---|:---|
| CPU | `mpstat`, `top` (%usr+%sys) | `vmstat` (r > nproc) | `dmesg` (MCE) |
| Memory | `free` (used/total) | `vmstat` (si/so > 0) | `dmesg` (OOM) |
| Disk I/O | `iostat` (%util) | `iostat` (avgqu-sz) | `smartctl`, `dmesg` |
| Network | `sar -n DEV` (rxkB+txkB vs bandwidth) | `ss -tnp` (Recv-Q) | `ip -s link` (errors, drops) |

### TSA Method (Thread State Analysis)

Instead of looking at resources, look at where **threads spend time**:

$$thread\_time = on\_CPU + off\_CPU$$

$$off\_CPU = \{runqueue, disk\_I/O, network\_I/O, lock, sleep, page\_fault, ...\}$$

Tools: `perf record -g`, `perf sched`, `offcputime` (BCC/bpftrace)

### RED Method (for Services)

For each service, measure:

- **R**ate — requests per second
- **E**rrors — failed requests per second
- **D**uration — latency distribution (p50, p95, p99)

$$availability = \frac{successful\_requests}{total\_requests} \times 100\%$$

### Saturation Indicators

| Resource | Saturated When |
|:---|:---|
| CPU | Run queue > 2x CPU count, %iowait high |
| Memory | Swapping (si/so > 0), OOM kills |
| Disk | avgqu-sz > 1, await >> svctm, %util = 100% |
| Network | Drops > 0, retransmits, Recv-Q > 0 |

---

## 7. Log Correlation Techniques

### Time-Based Correlation

When multiple components fail, correlating events by time reveals causality:

$$\text{if } |t_{event_A} - t_{event_B}| < \epsilon \implies \text{possibly related}$$

**Method:**
1. Identify the **first** error in the timeline
2. Check what happened **immediately before** (within seconds)
3. Look for cascading failures **after**

### Pattern Recognition

Common failure cascades:

```
Disk full → log writes fail → application errors → health check fails → restart loop
OOM → process killed → dependent services fail → cascading restarts
Network partition → timeout → connection pool exhaustion → all requests fail
Certificate expiry → TLS handshake fails → all HTTPS connections fail
```

### Structured Logging Analysis

```bash
# Count errors by type
journalctl -p err -b --no-pager -o json | jq -r '.MESSAGE' | sort | uniq -c | sort -rn

# Timeline of error frequency
journalctl -p err -b --no-pager -o short-iso | awk '{print substr($1,1,16)}' | uniq -c
```

---

## 8. OOM Killer Scoring Algorithm

### How the Kernel Chooses Victims

When the system runs out of memory and all reclaim attempts fail, the OOM killer selects a process to terminate.

### Badness Score Calculation

The kernel assigns a **badness score** to each process:

$$badness = \frac{process\_RSS + process\_swap + page\_table\_size}{total\_RAM} \times 1000$$

Then adjusted by `oom_score_adj`:

$$final\_score = badness + oom\_score\_adj$$

| oom_score_adj | Effect |
|:---|:---|
| -1000 | Never kill (OOM-immune) |
| -500 | Significantly less likely |
| 0 | Default |
| +500 | Significantly more likely |
| +1000 | Kill first |

### Selection Criteria

The OOM killer chooses the process that:
1. Has the **highest** badness score
2. Is not kernel thread (PID 0, kthreadd children)
3. Is not oom_score_adj = -1000

### Prevention Strategies

| Strategy | Method |
|:---|:---|
| Memory limits | cgroups memory.max / systemd MemoryMax= |
| Overcommit control | `vm.overcommit_memory` (0=heuristic, 1=always, 2=never) |
| Swap | Add swap for temporary overload |
| Monitoring | Alert when MemAvailable < threshold |
| Application tuning | Reduce memory footprint, fix leaks |

### Overcommit Accounting

When `vm.overcommit_memory=2`:

$$commit\_limit = swap\_total + RAM \times overcommit\_ratio / 100$$

Default `overcommit_ratio=50`: a system with 16 GB RAM and 4 GB swap can commit:

$$commit\_limit = 4 + 16 \times 0.5 = 12 \text{ GB}$$

---

## 9. Systematic Approach to Unknown Problems

### When You Have No Idea

```
1. Don't panic — gather data before changing anything
2. WHAT is the symptom? (Exact error, behavior)
3. WHEN did it start? (Check change logs, deployment history)
4. WHERE is it happening? (One host? All hosts? One region?)
5. WHO is affected? (All users? Specific users? Internal only?)
6. HOW OFTEN? (Constant? Intermittent? Under load?)
```

### Binary Search for Root Cause

When the cause is unknown, use **bisection** to narrow the scope:

$$iterations = \lceil \log_2(n) \rceil$$

Where $n$ is the number of possible causes.

**Example:** 64 config changes since last known good state:
- Test with first 32 reverted → still broken → problem in remaining 32
- Test with 16 of those reverted → works → problem in those 16
- Continue: 8, 4, 2, 1 → exact change found in 6 iterations

### Intermittent Problems

The hardest category. Strategies:

| Approach | When |
|:---|:---|
| Increase logging | Need more data when it happens |
| Continuous monitoring | Capture metrics at failure time |
| Load testing | If load-related, reproduce in controlled environment |
| Statistical analysis | Correlate failure times with external events (cron jobs, backups, batch processing) |
| Packet capture | For network intermittents, long-running tcpdump with rotation |

### The "Changed Nothing" Myth

Something **always** changed. Common hidden changes:
- Certificate expiry
- Log rotation filled disk
- DNS TTL expired, resolved to different IP
- Cron job ran (backup, cleanup, report)
- Kernel or library auto-update
- Upstream dependency changed
- Cloud provider maintenance
- Daylight saving time transition

---

## References

- Brendan Gregg: Systems Performance, 2nd Edition
- Brendan Gregg: BPF Performance Tools
- USE Method: brendangregg.com/usemethod.html
- Red Hat Troubleshooting Guide
- kernel.org: Documentation/admin-guide/sysctl/vm.rst
- kernel.org: Documentation/filesystems/ext4/
