# Kernel Hardening (Linux Security)

Harden the Linux kernel attack surface with boot parameters, sysctl tunables, module signing, and layered security modules.

## KASLR — Kernel Address Space Layout Randomization

```bash
# Check if KASLR is active (0 = fixed base, nonzero = randomized)
grep -c "nokaslr" /proc/cmdline && echo "KASLR DISABLED" || echo "KASLR ENABLED"

# Verify kernel text base is randomized across boots
dmesg | grep "Kernel Offset"

# GRUB: ensure KASLR is on (remove nokaslr if present)
# /etc/default/grub
# GRUB_CMDLINE_LINUX_DEFAULT="quiet splash"

# Enable user-space ASLR (should be 2 by default)
sysctl kernel.randomize_va_space
sysctl -w kernel.randomize_va_space=2   # 0=off, 1=stack/mmap, 2=full (+ brk)

# Check ASLR effectiveness per region
cat /proc/self/maps | head -20
```

## SMEP/SMAP — Supervisor Mode Execution/Access Prevention

```bash
# Check CPU support for SMEP and SMAP
grep -o 'smep\|smap' /proc/cpuinfo | sort -u

# Verify both are active (shown in CPU flags)
grep -E 'smep|smap' /proc/cpuinfo | head -1

# SMEP prevents kernel from executing user-space code
# SMAP prevents kernel from accessing user-space memory
# Both are hardware (CR4 register) — no sysctl, enabled by default on supported CPUs

# Disable for debugging only (boot param — NEVER in production)
# nosmap nosmep
```

## Lockdown Mode

```bash
# Check current lockdown state
cat /sys/kernel/security/lockdown

# Lockdown levels:
#   none         — no restrictions
#   integrity    — blocks unsigned modules, kexec, /dev/mem writes
#   confidentiality — integrity + blocks /proc/kcore, bpf read, perf

# Set via boot param (cannot be relaxed at runtime)
# GRUB_CMDLINE_LINUX_DEFAULT="lockdown=integrity"
# GRUB_CMDLINE_LINUX_DEFAULT="lockdown=confidentiality"

# Runtime: can only escalate, never relax
echo integrity > /sys/kernel/security/lockdown    # if currently 'none'
echo confidentiality > /sys/kernel/security/lockdown  # if currently 'integrity'
```

## Kernel Module Signing

```bash
# Check if module signature enforcement is active
cat /proc/sys/kernel/modules_disabled   # 1 = no new modules can load at all
grep CONFIG_MODULE_SIG /boot/config-$(uname -r)

# Key kernel config options:
# CONFIG_MODULE_SIG=y              — enable module signing
# CONFIG_MODULE_SIG_FORCE=y        — reject unsigned modules
# CONFIG_MODULE_SIG_ALL=y          — sign all modules during build
# CONFIG_MODULE_SIG_SHA512=y       — hash algorithm for signatures

# List loaded modules and check signatures
modinfo <module_name> | grep sig

# Prevent loading new modules entirely (one-way, survives until reboot)
echo 1 > /proc/sys/kernel/modules_disabled

# Check if a specific module is signed
modinfo -F signer ext4

# Persist via sysctl (sets on boot, no new modules after init)
echo "kernel.modules_disabled = 1" >> /etc/sysctl.d/50-hardening.conf
```

## Security Sysctl Parameters

### Information Leak Prevention

```bash
# Restrict kernel pointer exposure in /proc/kallsyms, dmesg, etc.
sysctl -w kernel.kptr_restrict=2    # 0=visible, 1=hide from non-CAP_SYSLOG, 2=hide from all

# Restrict dmesg to root (hide kernel ring buffer from unprivileged users)
sysctl -w kernel.dmesg_restrict=1

# Restrict perf_event (performance monitoring)
sysctl -w kernel.perf_event_paranoid=3   # 3=deny all, 2=deny kernel, 1=restrict, 0=open

# Disable unprivileged BPF (prevents spectre-class BPF attacks)
sysctl -w kernel.unprivileged_bpf_disabled=1

# Restrict eBPF JIT to root
sysctl -w net.core.bpf_jit_harden=2
```

### Process Isolation

```bash
# Restrict ptrace to parent-child only (Yama LSM)
sysctl -w kernel.yama.ptrace_scope=1
# 0 = classic (any process can ptrace), INSECURE
# 1 = parent-child only (default on Ubuntu)
# 2 = admin only (CAP_SYS_PTRACE)
# 3 = completely disabled

# Disable core dumps for SUID binaries
sysctl -w fs.suid_dumpable=0

# Restrict userns cloning (reduces container escape surface)
sysctl -w kernel.unprivileged_userns_clone=0   # Debian/Ubuntu only
```

### Persist All Hardening Sysctls

```bash
cat > /etc/sysctl.d/50-hardening.conf << 'EOF'
kernel.kptr_restrict = 2
kernel.dmesg_restrict = 1
kernel.perf_event_paranoid = 3
kernel.unprivileged_bpf_disabled = 1
kernel.yama.ptrace_scope = 1
kernel.randomize_va_space = 2
net.core.bpf_jit_harden = 2
fs.suid_dumpable = 0
fs.protected_hardlinks = 1
fs.protected_symlinks = 1
fs.protected_fifos = 2
fs.protected_regular = 2
EOF
sysctl --system
```

## Stack Protector

```bash
# Check kernel config for stack protector strength
grep CONFIG_STACKPROTECTOR /boot/config-$(uname -r)

# CONFIG_STACKPROTECTOR=y            — basic stack canary
# CONFIG_STACKPROTECTOR_STRONG=y     — strong: covers more functions (arrays, local addr-taken vars)
# CONFIG_STACKPROTECTOR_NONE is not set  — verify this is absent

# Userspace: compile with strong stack protector
gcc -fstack-protector-strong -o binary source.c

# Verify stack canary presence in a binary
objdump -d binary | grep __stack_chk_fail

# Also enable stack clash protection (guard pages)
gcc -fstack-clash-protection -fstack-protector-strong -o binary source.c
```

## KASAN/UBSAN — Runtime Sanitizers (Development)

```bash
# Check if KASAN is enabled in running kernel (development/debug kernels only)
grep CONFIG_KASAN /boot/config-$(uname -r)

# CONFIG_KASAN=y                  — Kernel Address Sanitizer (detects OOB, UAF, double-free)
# CONFIG_KASAN_GENERIC=y          — software instrumentation (2-3x slowdown)
# CONFIG_KASAN_SW_TAGS=y          — ARM64 memory tagging (lower overhead)
# CONFIG_KASAN_HW_TAGS=y          — ARM64 MTE hardware tags (minimal overhead)

# Check for UBSAN (Undefined Behavior Sanitizer)
grep CONFIG_UBSAN /boot/config-$(uname -r)

# CONFIG_UBSAN=y                  — detects signed overflow, alignment, shift issues
# CONFIG_UBSAN_TRAP=y             — panic on UB (strict mode)

# View KASAN reports in dmesg
dmesg | grep -A 20 "BUG: KASAN"

# KASAN is compile-time only — you need a debug kernel
# Never run KASAN in production (massive memory and CPU overhead)
```

## Secure Boot Chain

```bash
# Check Secure Boot status
mokutil --sb-state

# Check enrolled keys
mokutil --list-enrolled

# Verify shim is installed and signed
efibootmgr -v | grep -i shim

# Check UEFI Secure Boot variables
efivar -l | grep SecureBoot
cat /sys/firmware/efi/efivars/SecureBoot-*/data | xxd | head

# Enroll a new MOK (Machine Owner Key) for custom module signing
openssl req -new -x509 -newkey rsa:2048 -keyout MOK.priv -outform DER \
  -out MOK.der -nodes -days 36500 -subj "/CN=Custom Kernel Module Signing/"
mokutil --import MOK.der   # prompts for password, requires reboot

# Sign a kernel module with MOK
/usr/src/linux-headers-$(uname -r)/scripts/sign-file sha256 \
  MOK.priv MOK.der /path/to/module.ko

# Verify module signature
modinfo /path/to/module.ko | grep signer
```

## LSM Stacking — AppArmor + Landlock + Yama

```bash
# Check which LSMs are active
cat /sys/kernel/security/lsm

# Typical stacked output: lockdown,capability,yama,apparmor,landlock

# GRUB: configure LSM stack order
# GRUB_CMDLINE_LINUX_DEFAULT="lsm=lockdown,capability,yama,apparmor,landlock"

# Verify AppArmor status
aa-status

# Check Yama restrictions
sysctl kernel.yama.ptrace_scope

# Check Landlock ABI version
cat /sys/kernel/security/landlock/abi_version 2>/dev/null || echo "Landlock not available"

# Landlock is per-process sandboxing (filesystem access control)
# Applications must opt-in via landlock_create_ruleset(2)
# Works alongside AppArmor — complementary, not conflicting

# SELinux alternative (if using Fedora/RHEL instead of AppArmor)
getenforce         # Enforcing, Permissive, or Disabled
sestatus           # detailed status
```

## Practical Hardening Profiles

### Minimal Server (CIS Benchmark Style)

```bash
cat > /etc/sysctl.d/99-cis-hardening.conf << 'EOF'
# 1. Information leak prevention
kernel.kptr_restrict = 2
kernel.dmesg_restrict = 1
kernel.perf_event_paranoid = 3

# 2. BPF hardening
kernel.unprivileged_bpf_disabled = 1
net.core.bpf_jit_harden = 2

# 3. Process isolation
kernel.yama.ptrace_scope = 1
fs.suid_dumpable = 0

# 4. Filesystem hardening
fs.protected_hardlinks = 1
fs.protected_symlinks = 1
fs.protected_fifos = 2
fs.protected_regular = 2

# 5. ASLR
kernel.randomize_va_space = 2

# 6. ExecShield / NX enforcement (legacy, modern kernels enforce via hardware)
# kernel.exec-shield = 1  # deprecated on modern kernels

# 7. SysRq restriction (allow only sync+remount+reboot = 176)
kernel.sysrq = 176

# 8. Restrict loading of TTY line disciplines
dev.tty.ldisc_autoload = 0

# 9. Restrict userfaultfd to root
vm.unprivileged_userfaultfd = 0

# 10. Kexec restriction
kernel.kexec_load_disabled = 1
EOF
sysctl --system
```

### Boot Parameter Hardening

```bash
# /etc/default/grub — GRUB_CMDLINE_LINUX_DEFAULT additions:
# Security boot params:
#   init_on_alloc=1          — zero pages on allocation (prevents info leaks)
#   init_on_free=1           — zero pages on free (prevents use-after-free data leaks)
#   page_alloc.shuffle=1     — randomize page allocator freelists
#   slab_nomerge             — prevent slab cache merging (isolates UAF per cache)
#   iommu=force              — force IOMMU even if not needed (DMA attack prevention)
#   lockdown=integrity       — kernel lockdown
#   lsm=lockdown,capability,yama,apparmor,landlock
#   randomize_kstack_offset=on   — per-syscall kernel stack offset

# Apply
update-grub
```

### Audit Hardening State

```bash
# Quick audit script
echo "=== KASLR ==="
grep -c "nokaslr" /proc/cmdline && echo "DISABLED" || echo "ENABLED"

echo "=== SMEP/SMAP ==="
grep -oE 'smep|smap' /proc/cpuinfo | sort -u

echo "=== Lockdown ==="
cat /sys/kernel/security/lockdown 2>/dev/null || echo "Not available"

echo "=== LSMs ==="
cat /sys/kernel/security/lsm

echo "=== Key sysctls ==="
sysctl kernel.kptr_restrict kernel.dmesg_restrict kernel.perf_event_paranoid \
  kernel.unprivileged_bpf_disabled kernel.yama.ptrace_scope kernel.randomize_va_space

echo "=== Secure Boot ==="
mokutil --sb-state 2>/dev/null || echo "mokutil not available"

echo "=== Module signing ==="
grep CONFIG_MODULE_SIG_FORCE /boot/config-$(uname -r) 2>/dev/null
```

## Tips

- Start with `lockdown=integrity` and CIS sysctl profile; escalate to `confidentiality` only if needed
- `kernel.modules_disabled=1` is a one-way switch per boot; set it in late init only after all modules are loaded
- `kptr_restrict=2` can break tracing tools (`perf`, `bpftrace`); use `kptr_restrict=1` on dev systems
- `perf_event_paranoid=3` blocks all userspace `perf`; use `2` if you need application profiling
- Always test hardening in staging; strict settings can break container runtimes and debugging tools
- Combine KASLR with `init_on_alloc=1` and `slab_nomerge` for best memory safety
- Secure Boot without module signing enforcement is security theater
- Landlock adds per-process sandboxing without needing root to write AppArmor profiles
- Use `lynis audit system` for automated hardening audits against CIS and other benchmarks
- Keep `kernel.sysrq=176` (not 0) so emergency sync+reboot still works on hung systems

## See Also

- memory-tuning
- network-stack-tuning
- cgroups
- selinux
- apparmor
- iptables

## References

- Linux kernel documentation: `Documentation/admin-guide/kernel-parameters.txt`
- Linux kernel documentation: `Documentation/admin-guide/LSM/`
- Linux kernel documentation: `Documentation/security/`
- CIS Benchmark for Ubuntu Linux (cisecurity.org)
- Kernel Self-Protection Project (KSPP): kernsec.org/wiki
- `man 7 capabilities`, `man 2 ptrace`, `man 2 landlock_create_ruleset`
- Kees Cook, "Kernel Self-Protection" — Linux Security Summit
- UEFI Secure Boot documentation: `man mokutil`, `man sbsign`
- Intel SDM Vol. 3A, Section 2.5: Control Registers (CR4 — SMEP/SMAP bits)
- Greg Kroah-Hartman, "Signed Kernel Modules" — LWN.net
