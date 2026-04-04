# Reverse Engineering (Binary Analysis and Malware Dissection)

Techniques and tools for analyzing compiled binaries, understanding program behavior without source code, and dissecting malware through static and dynamic analysis.

## Ghidra (NSA Reverse Engineering Framework)

```bash
# Install Ghidra
# Download from https://ghidra-sre.org/
unzip ghidra_*.zip
cd ghidra_*/
./ghidraRun  # requires JDK 17+

# Command-line analysis (headless mode)
./support/analyzeHeadless /projects MyProject \
  -import /path/to/binary \
  -postScript ExportDecompilation.java \
  -deleteProject

# Ghidra CodeBrowser workflow:
# 1. File -> Import File -> select binary
# 2. Auto-analysis runs (accept defaults)
# 3. Symbol Tree -> Functions -> navigate code
# 4. Decompiler window shows C pseudocode
# 5. Right-click -> References -> Find references to

# Key Ghidra windows:
# Listing      - Disassembly view
# Decompiler   - C pseudocode
# Symbol Tree  - Functions, labels, namespaces
# Data Types   - Struct/enum definitions
# Bytes        - Hex editor view
# Function Graph - Control flow graph

# Ghidra scripting (Java/Python)
# Tools -> Script Manager -> New Script
# Example: find all calls to malloc
from ghidra.program.model.symbol import SymbolType
fm = currentProgram.getFunctionManager()
for func in fm.getFunctions(True):
    if "malloc" in func.getName():
        refs = getReferencesTo(func.getEntryPoint())
        for ref in refs:
            print(f"malloc called from {ref.getFromAddress()}")
```

## Radare2 / Rizin

```bash
# Install
brew install radare2    # macOS
apt install radare2     # Debian

# Open binary for analysis
r2 -A ./binary          # -A runs auto-analysis (aaa)
r2 -d ./binary          # debug mode
r2 -w ./binary          # write mode (patching)

# Core commands
afl                     # list all functions
afl~main                # grep for main
pdf @ main              # print disassembly of main
pdf @ sym.check_password
VV @ main               # visual graph mode (V then V)

# Seeking and navigation
s main                  # seek to main
s 0x00401000           # seek to address
s+10                    # seek forward 10 bytes
axt @ sym.strcmp         # cross-references to strcmp
axf @ main              # cross-references from main

# Analysis commands
aa                      # analyze all (basic)
aaa                     # analyze all autoname
aaaa                    # full analysis (slow, thorough)
afr                     # analyze function recursively
afl | sort -k2 -n       # functions sorted by size

# String analysis
iz                      # strings in data section
izz                     # strings in entire binary
iz~password             # grep strings for "password"
iz~http                 # find URLs

# Information
iI                      # binary info (arch, bits, endian)
iS                      # sections
iE                      # exports
ii                      # imports
ie                      # entrypoint
iR                      # relocations

# Hex and data
px 64 @ main            # hexdump 64 bytes at main
ps @ 0x00402000         # print string at address
pf S @ 0x00402000       # print formatted (struct)

# Patching
wa "nop" @ 0x00401234   # write assembly instruction
wx 90 @ 0x00401234      # write hex byte (NOP)
"wa jmp 0x00401300" @ 0x00401250  # redirect jump

# Visual mode
V                       # hex view
VV                      # function graph
Vp                      # visual panel mode
# In visual: p/P cycle views, hjkl navigate, q quit

# Rizin (radare2 fork, same concepts)
rizin -A ./binary
rz-bin -I ./binary      # binary info
rz-bin -z ./binary      # strings
```

## Binary Format Analysis

```bash
# ELF format (Linux)
readelf -h ./binary         # ELF header
readelf -l ./binary         # program headers (segments)
readelf -S ./binary         # section headers
readelf -s ./binary         # symbol table
readelf -d ./binary         # dynamic section (shared libs)
readelf -r ./binary         # relocations
readelf --notes ./binary    # notes (build ID, ABI)

# Key ELF sections:
# .text    - executable code
# .rodata  - read-only data (strings, constants)
# .data    - initialized global variables
# .bss     - uninitialized globals
# .plt     - procedure linkage table (lazy binding)
# .got     - global offset table
# .dynamic - dynamic linking info
# .init/.fini - constructor/destructor

# PE format (Windows)
# Using pefile (Python)
python3 -c "
import pefile
pe = pefile.PE('malware.exe')
print(f'Entry point: {hex(pe.OPTIONAL_HEADER.AddressOfEntryPoint)}')
print(f'Sections:')
for s in pe.sections:
    print(f'  {s.Name.decode().strip(chr(0)):8s} VA:{hex(s.VirtualAddress)} Size:{s.SizeOfRawData}')
print(f'Imports:')
for entry in pe.DIRECTORY_ENTRY_IMPORT:
    print(f'  {entry.dll.decode()}')
    for imp in entry.imports[:5]:
        print(f'    {imp.name.decode() if imp.name else hex(imp.ordinal)}')
"

# objdump (cross-platform)
objdump -d ./binary | head -100    # disassemble
objdump -t ./binary                 # symbol table
objdump -R ./binary                 # dynamic relocations
objdump -s -j .rodata ./binary     # dump rodata section

# file and magic
file ./binary
file ./suspicious.doc
```

## x86/ARM Disassembly Patterns

```bash
# Common x86-64 patterns to recognize:

# Function prologue
# push rbp
# mov rbp, rsp
# sub rsp, 0x20     ; allocate stack frame

# Function epilogue
# leave             ; mov rsp, rbp; pop rbp
# ret

# If-else: cmp eax, 0x41 / jne addr / ... / jmp skip
# For loop: mov ecx, 0 / .loop: cmp ecx, 10 / jge .end / inc ecx / jmp .loop
# Switch/case: cmp eax, 5 / ja .default / lea rdx,[rip+table] / jmp [rdx+rax*8]
# Syscall: mov rax, 59 / mov rdi, filename / syscall (sys_execve)

# ARM patterns:
# BL label (call), BX LR (return), LDR R0, =value (load constant)
# PUSH {R4-R7, LR} (save), POP {R4-R7, PC} (restore+return)

# Capstone disassembly (Python)
python3 -c "
from capstone import *
code = open('./binary', 'rb').read()[0x1000:0x1100]
md = Cs(CS_ARCH_X86, CS_MODE_64)
for insn in md.disasm(code, 0x401000):
    print(f'0x{insn.address:x}: {insn.mnemonic:8s} {insn.op_str}')
"
```

## Dynamic Analysis

```bash
# GDB essentials
gdb ./binary
# break main / break *0x00401234   ; set breakpoints
# run / continue / stepi / nexti   ; execution control
# info registers / bt              ; inspect state
# x/10x $rsp / x/s 0x402000       ; examine memory
# set $rax = 1                     ; modify register
# watch *0x00603010                ; hardware watchpoint
# catch syscall write              ; break on syscall

# GDB with GEF (GDB Enhanced Features)
bash -c "$(curl -fsSL https://gef.blah.cat/sh)"
gdb -q ./binary
# GEF adds: heap analysis, format string helpers,
# pattern create/search, vmmap, checksec

# strace (system call tracing)
strace ./binary                    # trace all syscalls
strace -e trace=network ./binary   # network calls only
strace -e trace=file ./binary      # file operations only
strace -e trace=write -s 200 ./binary  # write calls, 200 char strings
strace -p $(pidof target) -f       # attach to running process
strace -c ./binary                 # syscall statistics

# ltrace (library call tracing)
ltrace ./binary                    # trace libc calls
ltrace -e strcmp+strlen ./binary   # specific functions
ltrace -s 200 ./binary             # longer string output

# Frida (dynamic instrumentation)
pip install frida-tools
frida -l hook.js -f ./binary

# hook.js example:
# Interceptor.attach(Module.findExportByName(null, "strcmp"), {
#   onEnter: function(args) {
#     console.log("strcmp:", Memory.readUtf8String(args[0]),
#                 "vs", Memory.readUtf8String(args[1]));
#   },
#   onLeave: function(retval) {
#     console.log("  result:", retval.toInt32());
#   }
# });
```

## Anti-Debugging and Unpacking

```bash
# Common anti-debugging techniques:

# ptrace self-attach (Linux)
# if (ptrace(PTRACE_TRACEME, 0, 0, 0) == -1) exit(1);
# Bypass: patch ptrace call or LD_PRELOAD fake ptrace

# Timing checks
# rdtsc before/after code block; large delta = debugger
# Bypass: set hardware breakpoint past the check

# /proc/self/status TracerPid check
cat /proc/self/status | grep TracerPid
# Bypass: mount overlay on /proc or patch the check

# IsDebuggerPresent (Windows)
# Bypass: set PEB.BeingDebugged = 0

# UPX unpacking
upx -d packed_binary -o unpacked_binary
upx -l packed_binary    # list packing info

# Manual unpacking workflow:
# 1. Find OEP (Original Entry Point)
#    - Set breakpoint on VirtualProtect/mprotect
#    - Watch for execution transfer to unpacked code
# 2. Dump memory at OEP
#    - gdb: dump binary memory unpacked.bin 0x400000 0x450000
# 3. Fix imports (if needed)
#    - Use Scylla/Imprec for PE import reconstruction

# Detect packing
python3 -c "
import pefile, math
pe = pefile.PE('suspect.exe')
for s in pe.sections:
    data = s.get_data()
    entropy = 0
    if data:
        freq = [0]*256
        for b in data: freq[b] += 1
        for f in freq:
            if f > 0:
                p = f/len(data)
                entropy -= p * math.log2(p)
    name = s.Name.decode().strip(chr(0))
    print(f'{name:8s} entropy={entropy:.2f} size={len(data)}')
    # entropy > 7.0 suggests packed/encrypted
"
```

## Firmware and Embedded Analysis

```bash
# binwalk - firmware extraction
binwalk firmware.bin                    # scan for signatures
binwalk -e firmware.bin                 # extract embedded files
binwalk -Me firmware.bin                # recursive extraction
binwalk --entropy firmware.bin          # entropy graph

# Extract and inspect filesystem
cd _firmware.bin.extracted/ && unsquashfs squashfs-root.img
strings -n 8 firmware.bin | grep -i "password\|key\|secret\|token"

# QEMU for running extracted ARM binaries
qemu-arm-static -L ./squashfs-root/ ./squashfs-root/usr/bin/httpd
```

## Tips

- Start with static analysis (strings, imports, sections) before dynamic analysis to form hypotheses
- High entropy (>7.0) in code sections indicates packing or encryption; unpack before deep analysis
- Name functions and variables in Ghidra/IDA as you understand them; future-you will thank present-you
- Use cross-references (xrefs) religiously; they reveal data flow and call graphs faster than reading linearly
- Set breakpoints on interesting API calls (CreateFile, connect, VirtualAlloc) rather than stepping through code
- Compare strings output against known malware families and YARA rules for quick classification
- Use snapshots in VM-based dynamic analysis; malware may detect and evade sandbox environments
- Check for anti-debugging before dynamic analysis; patch or bypass before attempting to trace
- Document your findings as you go; RE sessions can span days and context is easily lost
- Use diffing tools (BinDiff, Diaphora) to compare malware variants and identify new capabilities
- Extract and hash unique strings, mutexes, and C2 URLs for threat intelligence sharing
- For firmware, always check for hardcoded credentials, debug interfaces, and outdated libraries

## See Also

- MITRE ATT&CK for mapping malware behaviors to techniques
- osquery for detecting artifacts of reverse-engineered malware on endpoints
- Suricata for network signatures derived from malware C2 analysis
- SIEM for correlating indicators discovered during analysis
- CIS Benchmarks for hardening against common exploitation targets

## References

- [Ghidra Official Documentation](https://ghidra-sre.org/CheatSheet.html)
- [Radare2 Book](https://book.rada.re/index.html)
- [Practical Binary Analysis (Dennis Andriesse)](https://practicalbinaryanalysis.com/)
- [Malware Unicorn RE Workshops](https://malwareunicorn.org/)
- [x86 Assembly Reference](https://www.felixcloutier.com/x86/)
- [GEF (GDB Enhanced Features)](https://hugsy.github.io/gef/)
- [binwalk Documentation](https://github.com/ReFirmLabs/binwalk)
- [Frida Dynamic Instrumentation](https://frida.re/docs/home/)
