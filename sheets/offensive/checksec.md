# Checksec (Binary Hardening Verification and Bypass)

> For authorized security testing, CTF competitions, and educational purposes only.

Checksec is the practice of verifying the security hardening features applied to compiled binaries. Modern operating systems and compilers provide multiple layers of exploit mitigation -- ASLR, NX, stack canaries, RELRO, PIE, and FORTIFY_SOURCE -- each making exploitation more difficult. Understanding what each protection does, how to verify it, and how attackers bypass it is fundamental to both defensive hardening and offensive security research.

---

## checksec.sh (Quick Binary Audit)

### Installation and Basic Usage

```bash
# Install checksec
git clone https://github.com/slimm609/checksec.sh
cd checksec.sh && sudo install -m 755 checksec /usr/local/bin/

# Or via package manager
apt install checksec               # Debian/Ubuntu
brew install checksec              # macOS

# Check a single binary
checksec --file=/usr/bin/ssh
# RELRO           STACK CANARY      NX            PIE
# Full RELRO      Canary found      NX enabled    PIE enabled

# Check all binaries in a directory
checksec --dir=/usr/bin/

# Check a running process
checksec --proc-all
checksec --proc=1234

# Check kernel security features
checksec --kernel

# Output formats
checksec --file=./binary --format=json
checksec --file=./binary --format=csv
checksec --file=./binary --format=xml

# Batch check with detailed output
for bin in /usr/sbin/*; do
    checksec --file="$bin" 2>/dev/null
done | sort
```

## ASLR (Address Space Layout Randomization)

### Checking and Understanding ASLR

```bash
# Check system ASLR level
cat /proc/sys/kernel/randomize_va_space
# 0 = disabled
# 1 = conservative (stack, mmap, VDSO)
# 2 = full (stack, mmap, VDSO, heap)

# Verify ASLR is randomizing addresses
for i in $(seq 1 5); do
    ldd /usr/bin/cat | grep libc
done
# Each run should show different libc addresses

# Check address randomization in a process
cat /proc/self/maps | grep -E "stack|heap|libc"
# Addresses change between runs

# Verify PIE binary gets randomized base
readelf -h ./binary | grep Type
# DYN (Position-Independent Executable) = PIE enabled
# EXEC (Executable file) = no PIE, fixed base

# ASLR entropy on x86-64 Linux
# Stack:       30 bits of entropy
# mmap/libs:   28 bits of entropy
# Heap:        13 bits of entropy
# PIE binary:  28 bits of entropy
# VDSO:        11 bits of entropy

# Disable ASLR for testing (requires root)
echo 0 | sudo tee /proc/sys/kernel/randomize_va_space

# Disable ASLR for single process
setarch $(uname -m) -R ./binary

# Re-enable ASLR
echo 2 | sudo tee /proc/sys/kernel/randomize_va_space
```

### ASLR Bypass Techniques

```bash
# Information leak: read a pointer from the program's output
# then compute base addresses from known offsets
# leaked_addr = 0x7f3a4c069420
# libc_base = leaked_addr - known_offset_of_puts

# Brute force (viable on 32-bit, impractical on 64-bit)
# 32-bit stack ASLR: ~8 bits entropy = 256 attempts
# 64-bit stack ASLR: ~30 bits = 1 billion attempts
while true; do
    ./exploit 2>/dev/null && break
done

# Partial overwrite (only overwrite low bytes of address)
# If ASLR randomizes bits 12-39, bits 0-11 are fixed (page offset)
# Overwriting 1-2 bytes keeps high bytes intact

# Return-to-PLT (PLT addresses are fixed in non-PIE binaries)
# Non-PIE binary: .text, .plt, .got addresses are fixed
# Use plt_puts to leak GOT entries, then compute libc base

# fork()-based servers reuse parent's address space
# Child processes inherit same ASLR layout
# One leak or brute-force attempt is reusable across all children
```

## NX / DEP (No-Execute / Data Execution Prevention)

### Checking NX Status

```bash
# Check NX bit in ELF program headers
readelf -l ./binary | grep GNU_STACK
# GNU_STACK      0x000000 0x00000000 0x00000000 0x00000 0x00000 RW  0x10
#                                                                ^^
# RW  = NX enabled (stack is read-write, NOT executable)
# RWE = NX disabled (stack is read-write-EXECUTE)

# Check with checksec
checksec --file=./binary
# NX: NX enabled / NX disabled

# Compile without NX (for testing)
gcc -z execstack -o vuln vuln.c

# Compile with NX (default)
gcc -o vuln vuln.c  # NX enabled by default

# Check at runtime
cat /proc/self/maps | grep -E "\[stack\]|\[heap\]"
# rw-p = not executable (NX enforced)
# rwxp = executable (NX NOT enforced)

# Verify hardware NX support
grep -c ' nx ' /proc/cpuinfo   # should be > 0
```

### NX Bypass Techniques

```bash
# Return-Oriented Programming (ROP)
# Chain existing executable code snippets (gadgets)
# Each gadget ends in RET, chaining them via stack addresses

# Find gadgets with ROPgadget
ROPgadget --binary ./binary
ROPgadget --binary ./binary --only "pop|ret"
ROPgadget --binary /lib/x86_64-linux-gnu/libc.so.6 --ropchain

# ret2libc: call system("/bin/sh") without injecting code
# Stack: [padding] [pop_rdi_ret] [addr_of_binsh] [addr_of_system]

# mprotect() to make memory executable
# ROP chain calls mprotect(shellcode_addr, size, PROT_READ|PROT_WRITE|PROT_EXEC)
# Then jump to shellcode

# JIT spray (in applications with JIT compilers)
# Embed shellcode within JIT-compiled constants
```

## Stack Canaries

### Checking Canary Protection

```bash
# Check for stack canary
checksec --file=./binary
# Stack: Canary found / No canary found

# Check in disassembly
objdump -d ./binary | grep -A2 "__stack_chk_fail"
# Presence of __stack_chk_fail references = canary enabled

# Look for canary setup in function prologue
objdump -d ./binary | grep -A5 "main>:"
# mov    rax, QWORD PTR fs:0x28    # load canary from TLS
# mov    QWORD PTR [rbp-0x8], rax  # store on stack

# Compile with canary
gcc -fstack-protector-all -o target target.c    # all functions
gcc -fstack-protector-strong -o target target.c # functions with arrays/ptrs
gcc -fstack-protector -o target target.c        # heuristic selection

# Compile without canary (for testing)
gcc -fno-stack-protector -o target target.c

# Canary value structure (Linux)
# Byte 7: always 0x00 (null terminator to stop string operations)
# Bytes 0-6: random (from /dev/urandom at program start)
```

### Canary Bypass Techniques

```bash
# Information leak: read the canary value from memory
# Then include correct canary in overflow payload
# payload = [buffer_padding] [correct_canary] [saved_rbp] [ret_addr]

# Format string leak
# %11$p might leak the canary value from the stack

# Brute-force (fork servers only)
# Canary is preserved across fork()
# Brute-force byte-by-byte: 256 * 7 = 1,792 attempts (null byte known)

# Overwrite canary and __stack_chk_fail GOT entry simultaneously
# If Partial RELRO: overwrite GOT[__stack_chk_fail] to skip the check

# Thread-local storage (TLS) overwrite
# If you can write to arbitrary addresses, overwrite the reference
# canary in TLS to match your overwritten stack canary
```

## RELRO (Relocation Read-Only)

### Checking RELRO Status

```bash
# Check RELRO level
checksec --file=./binary
# RELRO: Full RELRO / Partial RELRO / No RELRO

# Check with readelf
readelf -l ./binary | grep GNU_RELRO
# Presence of GNU_RELRO segment = at least Partial RELRO

readelf -d ./binary | grep BIND_NOW
# BIND_NOW present = Full RELRO

# Partial RELRO: GOT is writable, but .dynamic/.init/.fini are read-only
# Full RELRO: entire GOT is read-only after startup (lazy binding disabled)

# Compile with Full RELRO
gcc -Wl,-z,relro,-z,now -o target target.c

# Compile with Partial RELRO (usually default)
gcc -Wl,-z,relro -o target target.c

# Compile with No RELRO
gcc -Wl,-z,norelro -o target target.c

# Check GOT writability
readelf -S ./binary | grep -E "\.got|\.got\.plt"
# .got.plt with W flag = writable (Partial RELRO, GOT overwrite possible)
# .got without W flag = read-only (Full RELRO, GOT overwrite blocked)
```

### RELRO Bypass Techniques

```bash
# Partial RELRO: overwrite GOT entries
# Replace GOT[exit] with address of win() or system()
# python3 -c "from pwn import *; print(fmtstr_payload(7, {elf.got['exit']: elf.sym['win']}))"

# Full RELRO: GOT is read-only, so target other writable structures
# - __malloc_hook / __free_hook (removed in glibc 2.34+)
# - __exit_funcs (atexit handler list)
# - .fini_array (destructor array)
# - vtables (C++ virtual function tables)
# - FILE structure (_IO_list_all)
# - return addresses on the stack (still writable)
# - TLS (thread-local storage) function pointers

# Check for alternative write targets
readelf -S ./binary | grep -E "\.fini_array|\.init_array"
objdump -s -j .fini_array ./binary
```

## PIE (Position-Independent Executable)

### Checking PIE Status

```bash
# Check PIE
checksec --file=./binary
# PIE: PIE enabled / No PIE

# Check with readelf
readelf -h ./binary | grep Type
# DYN = PIE enabled (position-independent)
# EXEC = no PIE (fixed base address)

# Compile with PIE (default on most modern systems)
gcc -pie -fPIE -o target target.c

# Compile without PIE
gcc -no-pie -o target target.c

# Check if PIE binary is actually randomized
for i in $(seq 1 3); do
    readelf -l ./binary | grep LOAD | head -1
done
# For PIE, virtual addresses start at 0x0
# At runtime, kernel adds random base

# PIE + ASLR = all code addresses randomized
# Without PIE, .text/.plt/.got have fixed addresses
```

## FORTIFY_SOURCE

### Checking FORTIFY Level

```bash
# Check if FORTIFY_SOURCE is enabled
checksec --file=./binary
# FORTIFY: Yes / No

# Check manually: look for fortified function variants
objdump -t ./binary | grep -i fortify
nm -D ./binary | grep __.*_chk
# __printf_chk, __memcpy_chk, __strcpy_chk = fortified

# Compile with FORTIFY_SOURCE
gcc -D_FORTIFY_SOURCE=2 -O2 -o target target.c  # level 2 (stricter)
gcc -D_FORTIFY_SOURCE=1 -O2 -o target target.c  # level 1 (basic)
gcc -D_FORTIFY_SOURCE=3 -O2 -o target target.c  # level 3 (GCC 12+)

# IMPORTANT: requires -O1 or higher; -O0 disables FORTIFY

# What FORTIFY does:
# Replaces unsafe functions with bounds-checked versions
# memcpy(dst, src, n) -> __memcpy_chk(dst, src, n, dst_size)
# strcpy(dst, src)    -> __strcpy_chk(dst, src, dst_size)
# sprintf(buf, fmt)   -> __sprintf_chk(buf, flag, buf_size, fmt)

# Functions protected by FORTIFY_SOURCE:
# memcpy, memmove, memset, strcpy, strncpy, strcat, strncat
# sprintf, snprintf, vsprintf, vsnprintf, gets, printf family
```

## RUNPATH and RPATH

### Checking Library Search Paths

```bash
# Check for RPATH/RUNPATH
readelf -d ./binary | grep -E "RPATH|RUNPATH"
checksec --file=./binary

# RPATH: embedded library search path (searched before system paths)
# RUNPATH: similar but searched after LD_LIBRARY_PATH
# Both can be exploited if pointing to writable directories

# Check with chrpath
chrpath -l ./binary

# Compile with RPATH
gcc -Wl,-rpath,/opt/mylibs -o target target.c

# Remove RPATH (harden)
chrpath -d ./binary
patchelf --remove-rpath ./binary

# Set RUNPATH instead of RPATH
patchelf --set-rpath /opt/mylibs ./binary

# Attack: if RPATH points to writable directory
# Place malicious shared library in that directory
# ls -la /rpath/directory/  # check if writable
# cp evil_libfoo.so /rpath/directory/libfoo.so.1
# ./binary  # loads evil library

# Check all shared library dependencies
ldd ./binary
readelf -d ./binary | grep NEEDED

# Verify actual loaded libraries at runtime
LD_DEBUG=libs ./binary 2>&1 | head -50
```

## Comprehensive Security Audit

### Full Binary Assessment

```bash
# Complete security audit of a binary
echo "=== Binary Info ==="
file ./binary
readelf -h ./binary | grep -E "Type|Machine|Entry"

echo "=== Security Features ==="
checksec --file=./binary

echo "=== Sections ==="
readelf -S ./binary | grep -E "Name|\.got|\.plt|\.bss|\.text"

echo "=== Dynamic Dependencies ==="
readelf -d ./binary | grep -E "NEEDED|RPATH|RUNPATH|BIND_NOW|RELRO"

echo "=== Symbols ==="
nm -D ./binary 2>/dev/null | grep -E "__stack_chk|__fortify|_chk"

echo "=== Dangerous Functions ==="
objdump -t ./binary 2>/dev/null | grep -E "gets|strcpy|sprintf|scanf" | head -20

echo "=== Compiler Info ==="
readelf --string-dump=.comment ./binary 2>/dev/null

# Automated hardening recommendations
# Based on checksec output, recommend:
# No PIE       -> recompile with -pie -fPIE
# No Canary    -> recompile with -fstack-protector-strong
# Partial RELRO -> recompile with -Wl,-z,relro,-z,now
# No FORTIFY   -> recompile with -D_FORTIFY_SOURCE=2 -O2
# NX disabled  -> recompile without -z execstack
# RPATH found  -> remove with patchelf --remove-rpath

# GCC/Clang full hardening flags
gcc -O2 -Wall -Wextra -Werror \
    -fstack-protector-strong \
    -fstack-clash-protection \
    -fcf-protection \
    -D_FORTIFY_SOURCE=2 \
    -pie -fPIE \
    -Wl,-z,relro,-z,now \
    -Wl,-z,noexecstack \
    -o hardened target.c
```

---

## Tips

- Always check binaries with `checksec` before beginning exploit development; it determines which techniques are viable
- Full RELRO + PIE + NX + Canary + FORTIFY together make exploitation significantly harder but not impossible
- Non-PIE binaries give attackers fixed addresses for .text, .plt, and .got -- this is the most impactful single mitigation to enable
- Stack canaries are defeated by info leaks; if you find a format string vulnerability, leak the canary first
- Partial RELRO leaves the GOT writable; upgrade to Full RELRO with `-Wl,-z,relro,-z,now` in production
- FORTIFY_SOURCE requires optimization (`-O1` or higher); compiling with `-O0` silently disables it
- fork()-based servers preserve ASLR layout across children, enabling brute-force and leak reuse attacks
- Check RPATH/RUNPATH for writable directories; this is a commonly overlooked privilege escalation vector
- Use `-fstack-clash-protection` in addition to canaries to prevent stack clash attacks that jump over guard pages
- When auditing third-party binaries, also check all shared libraries they load with `ldd` and `checksec`

---

## See Also

- reverse-engineering
- pwntools
- sanitizers

## References

- [checksec.sh GitHub](https://github.com/slimm609/checksec.sh)
- [GCC Hardening Options](https://gcc.gnu.org/onlinedocs/gcc/Instrumentation-Options.html)
- [Clang Hardening Documentation](https://clang.llvm.org/docs/SafeStack.html)
- [Linux ASLR Implementation](https://www.kernel.org/doc/Documentation/sysctl/kernel.txt)
- [RELRO Explanation](https://www.redhat.com/en/blog/hardening-elf-binaries-using-relocation-read-only-relro)
- [Stack Canary Design](https://www.usenix.org/legacy/publications/library/proceedings/sec98/full_papers/cowan/cowan.pdf)
- [FORTIFY_SOURCE Documentation](https://man7.org/linux/man-pages/man7/feature_test_macros.7.html)
- [Exploiting Modern Binaries (LiveOverflow)](https://liveoverflow.com/)
