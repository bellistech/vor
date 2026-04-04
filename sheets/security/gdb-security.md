# GDB for Security Research (Exploit Development and Crash Analysis)

> For authorized security testing, CTF competitions, and educational purposes only.

GDB (GNU Debugger) is the foundational tool for security research on Linux, enabling crash triage, heap analysis, memory corruption investigation, and exploit development. Enhanced with extensions like pwndbg, GEF, and PEDA, GDB becomes a comprehensive platform for understanding vulnerabilities at the instruction level, inspecting heap metadata, scripting exploit logic, and bypassing anti-debugging protections.

---

## GDB Essentials for Security

### Starting and Attaching

```bash
# Start debugging a binary
gdb ./binary
gdb -q ./binary                    # quiet mode (no banner)
gdb --args ./binary arg1 arg2      # pass arguments

# Attach to running process
gdb -p $(pidof target_process)
sudo gdb -p 1234                   # needs privileges

# Load a core dump
gdb ./binary core.12345
gdb -c core.12345 ./binary

# Remote debugging
gdb ./binary -ex "target remote localhost:1234"

# Run with input
gdb -q ./binary -ex "run < input.bin"
gdb -q ./binary -ex "run <<< 'AAAA'"

# Batch mode (non-interactive)
gdb -batch -ex run -ex bt -ex quit --args ./binary crash_input
```

### Breakpoints and Execution Control

```bash
# Breakpoints
# break main                       # break at function
# break *0x00401234                # break at address
# break *main+42                   # break at offset
# break target.c:55               # break at source line
# break strcmp                     # break at library function

# Conditional breakpoints
# break *0x00401234 if $rdi == 0x41414141
# break malloc if $rdi > 1024     # large allocations only
# break *main+42 if *(int*)$rsp == 0xdeadbeef

# Hardware breakpoints (limited to 4 on x86)
# hbreak *0x00401234              # hardware breakpoint
# watch *(int*)0x00603010         # write watchpoint
# rwatch *(int*)0x00603010        # read watchpoint
# awatch *(int*)0x00603010        # access watchpoint (r+w)

# Catchpoints (break on events)
# catch syscall write             # break on write() syscall
# catch syscall execve            # break on execve()
# catch signal SIGSEGV            # break on segfault
# catch fork                     # break on fork()
# catch throw                    # break on C++ exceptions

# Execution control
# run                             # start execution
# continue                       # resume execution
# stepi                          # step one instruction
# nexti                          # step over call
# finish                         # run until return
# until *0x00401300              # run to address
# advance *0x00401300            # like until, skips recursive

# Reverse debugging (requires recording)
# record                         # start recording
# reverse-stepi                  # step backward
# reverse-continue               # continue backward
```

## Memory Examination

### Inspecting and Modifying Memory

```bash
# Examine memory (x command)
# x/10x $rsp                      # 10 hex words at stack pointer
# x/20wx $rsp                     # 20 32-bit hex words
# x/10gx $rsp                     # 10 64-bit hex words (giant)
# x/s 0x00402000                  # string at address
# x/10s 0x00402000                # 10 strings
# x/10i $rip                      # 10 instructions at RIP
# x/10bx $rsp                     # 10 bytes in hex
# x/10c 0x00402000                # 10 characters

# Examine formats: x=hex, d=decimal, u=unsigned, o=octal,
#                  t=binary, a=address, c=char, s=string, i=instruction
# Sizes: b=byte, h=halfword(2), w=word(4), g=giant(8)

# Print expressions
# p $rax                          # register value
# p/x $rax                        # register in hex
# p/x *(int*)$rdi                 # dereference as int
# p (char*)$rdi                   # dereference as string
# p/x $rip - 0x00400000           # calculate offset
# p {int}0x00603010               # read int at address

# Modify memory
# set *(int*)0x00603010 = 0x41414141
# set $rax = 0xdeadbeef
# set $rip = 0x00401234           # redirect execution
# set {char[8]}$rsp = "AAAA"

# Dump memory to file
# dump binary memory heap.bin 0x602000 0x603000
# dump binary memory stack.bin $rsp-0x200 $rsp+0x200

# Search memory
# find /b 0x00400000, 0x00500000, 0x48, 0x89, 0xe5  # byte pattern
# find /s 0x00400000, 0x00500000, "password"          # string search
# find /g 0x602000, 0x700000, 0xdeadbeef              # 64-bit value

# Info commands
# info registers                  # all registers
# info registers rax rbx          # specific registers
# info frame                     # current stack frame
# info locals                    # local variables
# info args                      # function arguments
# info proc mappings             # memory map (/proc/pid/maps)
# info sharedlibrary             # loaded shared libraries
# info threads                   # thread list
```

## Crash Triage

### Analyzing Crashes and Core Dumps

```bash
# Generate core dumps
ulimit -c unlimited
echo "/tmp/core.%e.%p" | sudo tee /proc/sys/kernel/core_pattern

# Analyze a crash
gdb -q ./binary core.12345
# bt                              # backtrace (call stack)
# bt full                         # backtrace with locals
# frame 3                         # switch to frame 3
# info registers                  # register state at crash
# x/i $rip                        # instruction that crashed
# x/10gx $rsp                     # stack at crash point

# Automated crash analysis script
# gdb -batch -q ./binary core.12345 \
#     -ex "bt" \
#     -ex "info registers" \
#     -ex "x/i \$rip" \
#     -ex "x/20gx \$rsp" \
#     -ex "info proc mappings"

# Classify crash type
# SIGSEGV (signal 11): memory access violation
#   - Read from unmapped: null deref, UAF, OOB read
#   - Write to unmapped: null deref, UAF, OOB write
#   - Execute from non-exec: DEP/NX violation
#
# SIGABRT (signal 6): assertion/abort
#   - Stack smashing detected (__stack_chk_fail)
#   - Double free or corruption
#   - Heap corruption detected
#
# SIGFPE (signal 8): division by zero / overflow
# SIGBUS (signal 7): alignment error
# SIGILL (signal 4): illegal instruction

# Determine exploitability
# Check if attacker controls:
# x/i $rip      # Is RIP overwritten? -> code execution
# x/gx $rsp     # Is RSP corrupted? -> stack pivot
# p $rdi         # Are function args controlled? -> parameter injection
# info registers # Which registers have 0x41414141 pattern?

# Crash reproduction
gdb -q ./binary -ex "run < crash_input"
# If crash is non-deterministic, use rr for record-replay:
# rr record ./binary < crash_input
# rr replay
```

## Heap Analysis

### Understanding Heap State

```bash
# With pwndbg/GEF extensions (heap commands):

# pwndbg heap commands:
# heap                           # heap overview
# bins                           # all bin contents
# fastbins                       # fastbin freelists
# unsortedbin                    # unsorted bin contents
# smallbins                      # small bin contents
# largebins                      # large bin contents
# tcache                         # tcache entries per thread
# tcachebins                     # tcache bin contents
# vis_heap_chunks                # visual heap layout
# find_fake_fast 0x601060        # find fake fastbin candidates
# malloc_chunk 0x603000          # parse chunk at address
# mp                             # malloc parameters
# top_chunk                      # wilderness chunk info
# arena                          # main arena info

# GEF heap commands:
# heap chunks                    # list all chunks
# heap bins                      # list all bins
# heap arenas                    # list arenas

# Manual heap inspection (no extensions)
# Main arena location
# p/x &main_arena
# x/30gx &main_arena             # inspect arena struct

# Examine a heap chunk
# chunk at addr:
# x/4gx addr-0x10               # prev_size, size, fd, bk
# Chunk header: [prev_size(8)] [size(8)] [data...]
# Size includes flags in low 3 bits:
#   bit 0 (PREV_INUSE): previous chunk is in use
#   bit 1 (IS_MMAPPED): chunk from mmap
#   bit 2 (NON_MAIN_ARENA): not main arena

# Check for heap corruption
# Verify chunk size consistency
# x/gx addr                      # chunk size
# x/gx addr+size                 # next chunk should be valid
# Verify fd/bk pointers of free chunks
# fd should point to another free chunk or bin
# bk should point to previous free chunk or bin

# Tcache structure (glibc 2.26+)
# Tcache is per-thread, stored at heap start
# Each bin holds up to 7 entries
# LIFO order (last freed = first allocated)
# x/70gx $tls_tcache_addr        # tcache counts + entries

# Detect double-free
# Freed chunk in tcache: fd points to next free chunk
# Double free: chunk appears twice in the same list
# tcache key field (glibc 2.29+): chunk+0x10 == tcache_addr -> freed
```

## Watchpoints for Memory Corruption

### Tracking Memory Writes

```bash
# Hardware watchpoints (precise, limited count)
# watch *(int*)0x00603010         # break on write to address
# watch -l variable_name          # watch location, not expression
# rwatch *(char*)0x00603010       # break on read
# awatch *(int*)0x00603010        # break on any access

# Watch a range (using multiple watchpoints)
# watch *(long*)0x603010
# watch *(long*)0x603018
# watch *(long*)0x603020
# watch *(long*)0x603028

# Conditional watchpoint
# watch *(int*)0x603010
# condition 1 *(int*)0x603010 == 0x41414141

# Software watchpoints (slow but unlimited)
# set can-use-hw-watchpoints 0    # force software watchpoints
# watch *(int*)0x603010           # now uses software watchpoint

# Common workflow: find who corrupts a variable
# 1. Run until the corruption is visible
# 2. Set watchpoint on the corrupted address
# 3. Re-run the program
# 4. GDB breaks at the exact instruction that writes the bad value

# Watch heap metadata corruption
# watch *(long*)(chunk_addr + 8)  # watch size field
# watch *(long*)(chunk_addr + 16) # watch fd pointer
# watch *(long*)(chunk_addr + 24) # watch bk pointer

# Watch stack canary
# x/gx $rbp-0x8                  # find canary location
# watch *(long*)($rbp-0x8)       # break when canary is overwritten
# Now overflow triggers breakpoint BEFORE __stack_chk_fail
```

## Python Scripting for Exploit Development

### GDB Python API

```bash
# GDB has a built-in Python interpreter
# python print(gdb.execute("info registers", to_string=True))

# Script: dump all readable memory regions
# python
# import gdb
# mappings = gdb.execute("info proc mappings", to_string=True)
# for line in mappings.split('\n'):
#     parts = line.split()
#     if len(parts) >= 5 and 'r' in parts[4]:
#         start = int(parts[0], 16)
#         end = int(parts[1], 16)
#         print(f"Readable: {hex(start)}-{hex(end)} ({end-start} bytes)")

# Script: auto-extract ROP gadgets
# python
# import gdb
# def find_gadgets():
#     text_start = 0x400000
#     text_end = 0x401000
#     for addr in range(text_start, text_end):
#         try:
#             insn = gdb.execute(f"x/2i {addr}", to_string=True)
#             if "ret" in insn.split('\n')[-1]:
#                 prev = insn.split('\n')[0]
#                 if "pop" in prev:
#                     print(f"Gadget at {hex(addr)}: {prev.strip()}")
#         except:
#             pass
# find_gadgets()

# Script: format string offset finder
# python
# import gdb, struct
# def find_fmt_offset():
#     for i in range(1, 50):
#         payload = f"%{i}$p"
#         gdb.execute(f"run <<< '{payload}'")
#         output = gdb.execute("x/s $output_buf", to_string=True)
#         print(f"Offset {i}: {output}")
# find_fmt_offset()

# Load Python script from file
# source /path/to/exploit_helper.py

# Define custom GDB commands in Python
# python
# class HexdumpCommand(gdb.Command):
#     def __init__(self):
#         super().__init__("hexdump_at", gdb.COMMAND_USER)
#     def invoke(self, arg, from_tty):
#         addr = int(arg, 16)
#         data = gdb.selected_inferior().read_memory(addr, 64)
#         for i in range(0, len(data), 16):
#             hex_str = ' '.join(f'{b:02x}' for b in data[i:i+16])
#             print(f'{addr+i:#010x}: {hex_str}')
# HexdumpCommand()
# Usage: hexdump_at 0x603000

# Breakpoint with Python callback
# python
# class NotifyBreakpoint(gdb.Breakpoint):
#     def stop(self):
#         rdi = int(gdb.parse_and_eval("$rdi"))
#         if rdi > 0x1000:
#             print(f"Interesting call with rdi={hex(rdi)}")
#             return True  # stop
#         return False  # continue
# NotifyBreakpoint("*0x401234")
```

## Pwndbg, GEF, and PEDA Extensions

### Installing and Using GDB Extensions

```bash
# Install pwndbg (recommended for heap/exploit work)
git clone https://github.com/pwndbg/pwndbg
cd pwndbg && ./setup.sh

# Install GEF (single-file, lightweight)
bash -c "$(curl -fsSL https://gef.blah.cat/sh)"

# Install PEDA (classic, Python 2/3)
git clone https://github.com/longld/peda.git ~/peda
echo "source ~/peda/peda.py" >> ~/.gdbinit

# Pwndbg-specific commands
# context                        # show registers, code, stack, backtrace
# vmmap                          # memory map with permissions
# checksec                       # binary security features
# cyclic 200                     # generate cyclic pattern
# cyclic -l 0x61616168           # find pattern offset
# rop                            # find ROP gadgets
# got                            # display GOT entries
# plt                            # display PLT entries
# search -s "flag{"              # search for string
# search -p 0xdeadbeef           # search for pointer
# distance $rsp $rbp             # calculate distance
# telescope $rsp 20              # smart stack display
# retaddr                        # find return addresses on stack

# GEF-specific commands
# gef config                     # configure GEF
# pattern create 200             # generate De Bruijn pattern
# pattern search $rip            # find offset
# heap chunks                    # heap chunk listing
# heap bins                      # bin listing
# vmmap                          # memory map
# checksec                       # security features
# xinfo $rsp                     # extended info about address
# format-string-helper           # format string exploit aid
# shellcode search linux x86_64  # search shellcode database

# PEDA-specific commands
# pdisas main                    # pretty disassemble
# patto 200                      # pattern create
# patts $rip                     # pattern search
# dumprop                        # dump ROP gadgets
# jmpcall                        # find jmp/call gadgets
# skeleton                       # generate exploit skeleton
# assemble                       # inline assembler
```

## Anti-Debug Detection and Bypass

### Defeating Anti-Debugging Techniques

```bash
# Bypass ptrace anti-debug
# Binary calls: ptrace(PTRACE_TRACEME, 0, 0, 0)
# If return -1, debugger detected

# Method 1: Patch the ptrace call
# set *(char*)0x401234 = 0x90   # NOP the call
# set *(char*)0x401235 = 0x90
# set *(char*)0x401236 = 0x90

# Method 2: Catch and modify return value
# catch syscall ptrace
# commands
#     silent
#     set $rax = 0              # pretend ptrace succeeded
#     continue
# end

# Method 3: LD_PRELOAD fake ptrace
# // fake_ptrace.c
# long ptrace(int req, ...) { return 0; }
# gcc -shared -o fake_ptrace.so fake_ptrace.c
# LD_PRELOAD=./fake_ptrace.so gdb ./binary

# Bypass /proc/self/status TracerPid check
# The binary reads /proc/self/status and checks TracerPid
# Redirect the read:
# catch syscall openat
# commands
#     silent
#     if $_streq((char*)$rsi, "/proc/self/status")
#         # let it open, but we'll modify the read result
#         printf "Intercepted /proc/self/status open\n"
#     end
#     continue
# end

# Bypass timing checks (rdtsc)
# Binary measures time between points; slow execution = debugger
# Skip the check:
# break *timing_check_addr
# commands
#     silent
#     set $rip = past_check_addr   # jump past the check
#     continue
# end

# Bypass signal-based anti-debug
# Binary sends SIGTRAP to itself; if debugger catches it, behavior changes
# handle SIGTRAP nostop noprint pass
# Or: catch signal SIGTRAP and modify behavior

# Bypass int3 (0xCC) self-checking
# Binary scans its own code for 0xCC (breakpoint opcode)
# Use hardware breakpoints instead:
# hbreak *0x401234               # hardware BP, no 0xCC inserted

# Common anti-debug patterns and bypasses summary:
# ptrace self-attach     -> catch syscall + set $rax=0
# /proc/self/status      -> redirect file read
# timing (rdtsc/clock)   -> skip check or set registers
# SIGTRAP/signal tricks  -> handle signal with pass
# int3 scanning          -> use hardware breakpoints
# IsDebuggerPresent (Win)-> set PEB.BeingDebugged=0
# NtQueryInformationProcess -> hook and return clean
```

## Core Dump Analysis

### Post-Mortem Debugging

```bash
# Enable core dumps
ulimit -c unlimited
echo "/tmp/core.%e.%p.%t" | sudo tee /proc/sys/kernel/core_pattern

# Analyze core dump
gdb ./binary /tmp/core.binary.12345.1234567890

# Essential analysis commands
# bt                              # where did it crash
# bt full                         # with all local variables
# info registers                  # register state
# x/i $rip                        # faulting instruction
# info signal SIGSEGV             # signal details
# x/gx $rsp                       # stack pointer contents
# info proc mappings              # memory layout at crash time

# Examine all threads
# info threads                    # list threads
# thread 2                        # switch to thread 2
# thread apply all bt             # backtrace all threads

# Check for corruption patterns
# x/100gx $rsp                   # look for 0x41414141 patterns
# find /g 0x600000, 0x700000, 0x4141414141414141

# Extract crash context for bug report
gdb -batch -q ./binary core.12345 \
    -ex "set pagination off" \
    -ex "bt" \
    -ex "info registers" \
    -ex "x/5i \$rip" \
    -ex "x/20gx \$rsp" \
    -ex "thread apply all bt" \
    -ex "info proc mappings" > crash_report.txt 2>&1

# Compare multiple core dumps
# Use stack trace hashes for deduplication
for core in /tmp/core.binary.*; do
    hash=$(gdb -batch -q ./binary "$core" -ex bt 2>&1 | \
        grep "^#" | md5sum | cut -d' ' -f1)
    echo "$core -> $hash"
done | sort -t'>' -k2

# Analyze ASAN core dump
# ASAN report is in stderr, core dump has shadow memory
# Check shadow memory: x/bx (addr >> 3) + 0x7fff8000
```

## Advanced GDB Techniques

### Security Research Workflows

```bash
# Follow fork (debug child processes)
set follow-fork-mode child        # debug child after fork
set follow-fork-mode parent       # stay in parent (default)
set detach-on-fork off            # debug both parent and child

# Non-stop mode (other threads continue while one is stopped)
set non-stop on
set pagination off

# Record and replay (deterministic debugging)
# record                         # start recording
# reverse-stepi                  # step backward
# reverse-continue               # run backward
# record stop                    # stop recording

# Checkpoint/restore (save and restore state)
# checkpoint                     # save current state
# info checkpoints               # list saved states
# restart 1                      # restore checkpoint 1

# Examine PLT/GOT entries
# x/gx 0x601018                  # GOT entry for puts
# x/3i 0x400410                  # PLT stub for puts
# info functions puts            # all puts-related symbols

# Trace function calls
# set logging on
# break puts
# commands
#     silent
#     printf "puts(%s)\n", (char*)$rdi
#     continue
# end

# Disassemble with Intel syntax
set disassembly-flavor intel

# Useful GDB settings for security work
set follow-exec-mode new          # follow execve
set disable-randomization off     # keep ASLR (default: on = disabled)
set print asm-demangle on         # demangle C++ symbols
```

---

## Tips

- Install pwndbg as your primary GDB extension for security work; its `context` display, heap commands, and ROP gadget finder are indispensable
- Use `set disable-randomization off` when debugging exploits that depend on ASLR behavior; GDB disables ASLR by default
- Hardware breakpoints (`hbreak`) bypass anti-debug checks that scan for software breakpoint opcodes (0xCC)
- Set watchpoints on stack canary locations to catch the exact instruction that corrupts them, rather than waiting for `__stack_chk_fail`
- Use `catch syscall execve` to detect shellcode execution during exploit development; it breaks right at the syscall
- Record execution with `record` for deterministic replay of non-deterministic crashes; step backward through the crash
- Always check `info proc mappings` to understand the memory layout before writing memory addresses in exploits
- Use Python scripting for repetitive analysis tasks; custom GDB commands persist across sessions via `.gdbinit`
- For heap exploitation, use `vis_heap_chunks` (pwndbg) to visualize chunk layout and verify your exploit's heap state
- Bypass ptrace-based anti-debug with `catch syscall ptrace` and setting `$rax = 0` rather than patching the binary

---

## See Also

- reverse-engineering
- pwntools
- sanitizers
- checksec

## References

- [GDB Documentation](https://sourceware.org/gdb/current/onlinedocs/gdb/)
- [pwndbg GitHub](https://github.com/pwndbg/pwndbg)
- [GEF Documentation](https://hugsy.github.io/gef/)
- [PEDA GitHub](https://github.com/longld/peda)
- [GDB Python API](https://sourceware.org/gdb/current/onlinedocs/gdb/Python-API.html)
- [how2heap](https://github.com/shellphish/how2heap)
- [Azeria Labs ARM Debugging](https://azeria-labs.com/debugging-with-gdb-introduction/)
- [Anti-Debugging Techniques (OpenSecurityTraining)](https://opensecuritytraining.info/)
