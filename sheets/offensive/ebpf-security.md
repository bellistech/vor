# eBPF Security (Attack Surface and Security Research)

> For authorized security testing, CTF competitions, and educational purposes only.

eBPF is a powerful in-kernel virtual machine allowing sandboxed programs to run in
the Linux kernel without modifying kernel source or loading modules. Its deep kernel
integration creates a unique attack surface spanning the verifier, helper functions,
JIT compiler, and map subsystem.

---

## Verifier Bypass Techniques

### Complexity Bombs

```bash
# The verifier enforces a 1M instruction complexity limit
# Complexity bombs craft programs expensive to verify but ultimately
# pass verification with unintended behavior

# Check current verifier and JIT limits
sysctl kernel.unprivileged_bpf_disabled
cat /proc/sys/kernel/bpf_jit_limit

# Deeply nested conditionals exhaust verifier state tracking
# Each branch doubles verification paths — 30 nested ifs = 2^30 paths
# Verifier may prune paths incorrectly, allowing OOB access

# Monitor verifier statistics
cat /proc/sys/net/core/bpf_jit_enable
```

### Back-Edge Confusion

```bash
# eBPF originally banned loops (back-edges in the CFG)
# Bounded loops added in kernel 5.3+ expand the attack surface
# Confusion: verifier miscalculates loop bounds → excess iterations → OOB

# Check kernel version for bounded loop support
uname -r
# 5.3+ supports bounded loops, increasing attack surface significantly
```

### Dead Code Exploitation

```bash
# Verifier prunes "unreachable" code paths
# Dead code exploits insert instructions the verifier ignores
# but the JIT compiler or runtime actually executes

# Technique: conditional branch with a predicate the verifier
# statically evaluates as always-false, but runtime disagrees
# due to speculative execution or register state divergence

# Compare verified vs compiled output for discrepancies
bpftool prog dump xlated id <PROG_ID>    # verified bytecode
bpftool prog dump jited id <PROG_ID>     # native JIT output
# Discrepancies between xlated and jited indicate potential issues
```

---

## Helper Function Abuse

### Dangerous Helpers

```bash
# List available BPF helpers for a given program type
bpftool feature probe | grep -A 100 "Helper"
bpftool feature probe kernel

# High-risk helpers for exploitation:
# bpf_probe_read_kernel — reads arbitrary kernel memory
# bpf_probe_write_user — writes to user-space memory
# bpf_override_return — overrides function return values (kprobes)
# bpf_send_signal — sends signals to processes
# bpf_d_path — leaks file path information
```

### bpf_override_return Abuse

```bash
# Available only with CONFIG_BPF_KPROBE_OVERRIDE
# Allows kprobe BPF programs to override traced function return values

# Attack: override security_file_open() to return 0
# This bypasses LSM file access controls entirely

# Check if override is available (should be disabled in production)
grep CONFIG_BPF_KPROBE_OVERRIDE /boot/config-$(uname -r)
# RHEL/CentOS typically disable it; Ubuntu may have it enabled
```

---

## Map Exploitation

### Map Type Confusion

```bash
# BPF maps are the primary data structure for BPF programs
# Type confusion: verifier accepts array access on a hash map
# Memory layout mismatch enables OOB access

# Enumerate all BPF maps on the system
bpftool map list
bpftool map show id <MAP_ID>
bpftool map dump id <MAP_ID>

# Map types with different memory layouts:
# BPF_MAP_TYPE_ARRAY      — fixed-size contiguous memory
# BPF_MAP_TYPE_HASH       — hash table with per-bucket locking
# BPF_MAP_TYPE_RINGBUF    — lock-free ring buffer
# BPF_MAP_TYPE_PROG_ARRAY — holds program file descriptors
```

### Map-of-Maps Nested Confusion

```bash
# BPF_MAP_TYPE_ARRAY_OF_MAPS and HASH_OF_MAPS hold map references
# Attack: inner map replacement race condition
# 1. Create outer map with inner map A
# 2. BPF program begins accessing inner map A
# 3. Userspace replaces inner map A with map B (different type/size)
# 4. BPF program accesses map B with map A's assumptions

bpftool map list | grep -i "array_of_maps\|hash_of_maps"
```

---

## BPF-to-BPF Calls and Tail Calls

### Function Call Gadgets

```bash
# BPF-to-BPF calls (kernel 4.16+): each subprogram verified separately
# Gadget: call chain where pointer bounds differ between caller/callee
# Function A passes validated pointer to B; B may have different bounds

# Inspect subprogram calls
bpftool prog dump xlated id <PROG_ID> | grep "call pc"
```

### Tail Call Exploitation

```bash
# Tail calls: one BPF program jumps to another (max chain: 33)
# Stored in BPF_MAP_TYPE_PROG_ARRAY maps

# Attack: circular tail call chain → kernel hang (DoS)
# Program A tail-calls B, B tail-calls A
# 33-call limit prevents infinite loops but causes significant delays

bpftool map list | grep prog_array

# Monitor BPF program CPU time
echo 1 > /proc/sys/kernel/bpf_stats_enabled
bpftool prog list                        # shows run_time_ns, run_cnt
# Tail call depth approaching 33 is suspicious
```

---

## BTF and JIT Exploitation

### BTF Mismatch

```bash
# BPF Type Format (BTF) provides type information for BPF programs
# BTF mismatches between compile-time and runtime types → OOB access

# Check available BTF information
bpftool btf list
bpftool btf dump id <BTF_ID>
ls /sys/kernel/btf/vmlinux

# Attack: compile BPF against one kernel's BTF, load on another
# Field offsets will be wrong — enables OOB reads/writes
# CO-RE relocations can mask mismatches if not properly validated
```

### JIT Spray

```bash
# BPF JIT compiles verified bytecode to native instructions
# Without hardening, controlled immediates appear in JIT output
# Attacker embeds controlled byte sequences as native code gadgets

cat /proc/sys/net/core/bpf_jit_enable   # 0=off, 1=on, 2=debug
cat /proc/sys/net/core/bpf_jit_harden   # 0=off, 1=unpriv, 2=all

# Enable JIT hardening (constant blinding) — defensive
echo 2 > /proc/sys/net/core/bpf_jit_harden

# With constant blinding, immediates are XORed with random values:
# C → (C XOR R), R — prevents predictable gadget placement
```

---

## BPF LSM Bypass

### Hook Enumeration and Policy Gaps

```bash
# BPF LSM (kernel 5.7+) allows BPF programs as security hooks
# If misconfigured, these policies can be bypassed

cat /sys/kernel/security/lsm             # check if "bpf" is listed
bpftool prog list | grep lsm            # list LSM BPF programs
bpftool link list | grep "type lsm"     # list attachments

# Common policy gaps:
# 1. hook on open() but not openat2() — use openat2 to bypass
# 2. missing hooks on mmap/mprotect — code execution bypass
# 3. no hook on bpf() syscall itself — BPF self-loading
# 4. incomplete network hooks (connect but not bind or sendmsg)

# Test for gaps via alternative syscall paths
strace -e trace=openat,openat2,open -f ./target_binary 2>&1
```

---

## Unprivileged BPF and Namespace Isolation

### Privilege Escalation via Unprivileged BPF

```bash
# Check if unprivileged BPF is enabled
cat /proc/sys/kernel/unprivileged_bpf_disabled
# 0 = enabled (DANGEROUS), 1 = disabled, 2 = permanently disabled

# CVE-2021-3490, CVE-2021-31440, CVE-2021-4204
# All exploited unprivileged BPF for kernel privilege escalation

# Disable unprivileged BPF (defensive)
echo 1 > /proc/sys/kernel/unprivileged_bpf_disabled
echo 2 > /proc/sys/kernel/unprivileged_bpf_disabled   # permanent

# Capability requirements (kernel 5.8+):
# CAP_BPF         — load programs, create maps
# CAP_PERFMON     — attach tracing programs
# CAP_NET_ADMIN   — attach networking programs
# CAP_SYS_ADMIN   — legacy catch-all (pre-5.8)
capsh --print
```

### Namespace and Cgroup Isolation Weaknesses

```bash
# BPF maps are global — not namespace-scoped
bpftool map list                         # maps visible across namespaces

# Pinned objects in /sys/fs/bpf visible if mount is shared
ls -la /sys/fs/bpf/

# Parent cgroup BPF programs affect all child cgroups (containers)
# Containers cannot override or detach parent cgroup BPF programs
bpftool cgroup tree /sys/fs/cgroup/
bpftool cgroup show /sys/fs/cgroup/system.slice/docker-<id>.scope
```

---

## Defensive Controls

### seccomp-BPF

```bash
# Block bpf() syscall in containers via seccomp profile
# Docker default seccomp profile blocks bpf() — verify:
docker inspect --format '{{.HostConfig.SecurityOpt}}' <container>

# Apply custom seccomp profile
docker run --security-opt seccomp=./bpf-block.json <image>
```

### Capability Hardening

```bash
# Drop BPF-related capabilities from containers
docker run --cap-drop=ALL --cap-add=NET_BIND_SERVICE <image>

# Verify a process has no BPF capabilities
getpcaps <PID>

# Audit BPF syscall usage with auditd
auditctl -a always,exit -F arch=b64 -S bpf -k bpf_audit
ausearch -k bpf_audit
```

### Full System Assessment

```bash
# BPF security assessment checklist
cat /proc/sys/kernel/unprivileged_bpf_disabled  # should be 1 or 2
cat /proc/sys/net/core/bpf_jit_enable           # JIT status
cat /proc/sys/net/core/bpf_jit_harden           # should be 2
cat /proc/sys/net/core/bpf_jit_kallsyms         # symbol visibility

# Enumerate all loaded BPF objects
bpftool prog list
bpftool map list
bpftool link list

# Find pinned BPF objects
find /sys/fs/bpf -type f 2>/dev/null

# Check kernel config for BPF features
zcat /proc/config.gz 2>/dev/null | grep -i bpf
grep -i bpf /boot/config-$(uname -r) 2>/dev/null
```

---

## Tips

- Always check `unprivileged_bpf_disabled` first — set to 2 blocks most userspace attacks
- Verifier bypass exploits are kernel-version-specific; match exploit to exact kernel
- JIT hardening (`bpf_jit_harden=2`) blocks constant-based JIT spray but adds overhead
- Container escapes via BPF require CAP_SYS_ADMIN or unprivileged BPF enabled
- Use `bpftool prog dump xlated` vs `jited` comparison to find JIT inconsistencies
- Tail call chain depth approaching 33 is suspicious — monitor with bpf_stats
- BPF LSM gaps are common in early deployments; audit with systematic syscall testing
- CVE databases list 20+ BPF privilege escalation bugs since 2020
- Map-of-maps race conditions require precise timing; use usleep calibration
- BPF maps are not namespace-scoped — this is a known container isolation weakness

---

## See Also

- container-escape
- sanitizers
- seccomp

## References

- [Linux Kernel BPF Documentation](https://docs.kernel.org/bpf/)
- [eBPF.io - Introduction to eBPF](https://ebpf.io/)
- [bpftool Manual](https://man7.org/linux/man-pages/man8/bpftool.8.html)
- [CVE-2021-3490 eBPF Exploit Analysis](https://www.graplsecurity.com/post/kernel-pwning-with-ebpf-a-love-story)
- [seccomp-BPF Documentation](https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html)
- [BPF LSM Documentation](https://docs.kernel.org/bpf/prog_lsm.html)
