# Ghidra (NSA Reverse Engineering Framework)

> For authorized security testing, CTF competitions, and educational purposes only.

Ghidra is the NSA's open-source software reverse engineering suite providing disassembly,
decompilation, scripting, and binary analysis capabilities. It supports dozens of processor
architectures and executable formats, with an extensible plugin system and a powerful
P-Code intermediate representation that enables cross-architecture analysis.

---

## Project Setup

### Creating and Importing

```bash
# Launch Ghidra from install directory
/opt/ghidra/ghidraRun

# Headless import (no GUI)
/opt/ghidra/support/analyzeHeadless /path/to/project ProjectName \
  -import /path/to/binary

# Import with specific language/compiler
/opt/ghidra/support/analyzeHeadless /path/to/project ProjectName \
  -import firmware.bin \
  -processor "ARM:LE:32:v7" \
  -cspec "default"

# Import multiple files at once
/opt/ghidra/support/analyzeHeadless /path/to/project ProjectName \
  -import /path/to/binaries/*.elf

# Create a shared project (for team collaboration)
# File -> New Project -> Shared Project -> ghidra://server:port/repo
```

### Project Organization

```bash
# Recommended project folder structure
# /project_root/
#   ├── original/          # Unmodified binaries
#   ├── unpacked/          # Unpacked/decrypted samples
#   ├── scripts/           # Custom Ghidra scripts
#   └── exports/           # Exported analysis artifacts

# Set analysis options before auto-analysis
# Analysis -> Auto Analyze -> configure analyzers:
#   - Decompiler Parameter ID (enable for better signatures)
#   - Aggressive Instruction Finder (enable for obfuscated code)
#   - Stack (enable for local variable recovery)
#   - Non-Returning Functions (enable for better control flow)
```

---

## CodeBrowser Navigation

### Essential Keyboard Shortcuts

```bash
# Navigation
# G            — Go to address
# Ctrl+E       — Go to label/symbol
# Ctrl+Shift+E — Go to external symbol
# Alt+Left     — Navigate back
# Alt+Right    — Navigate forward
# Ctrl+T       — Open symbol table
# Ctrl+Shift+T — Open function call tree

# Editing
# L            — Rename label/variable
# ;            — Set comment (EOL)
# Ctrl+;       — Set pre-comment
# Shift+;      — Set post-comment
# /            — Set plate comment
# T            — Set data type on variable
# D            — Convert to data (cycle: byte/word/dword/qword)
# C            — Convert to code (disassemble)
# U            — Undefine (clear code/data)
# P            — Create function at cursor
# F            — Edit function signature

# Display
# Space        — Toggle listing/decompiler
# Ctrl+Shift+G — Toggle function graph
# Ctrl+E       — Symbol table
```

### Window Management

```bash
# Essential windows for analysis:
# Window -> Bytes           — Raw hex view
# Window -> Decompiler      — C-like pseudo-code
# Window -> Function Graph  — Control flow graph
# Window -> Symbol Table    — All defined symbols
# Window -> Data Type Manager — Type definitions
# Window -> Bookmarks       — Analysis bookmarks
# Window -> Script Manager  — Run/edit scripts
# Window -> Memory Map      — Memory layout
# Window -> Register Manager — CPU registers

# Snap windows side by side:
# Drag decompiler next to listing for synchronized view
# Both views auto-sync cursor position
```

---

## Decompiler Window

### Working with Decompiled Code

```bash
# In Decompiler window:
# Right-click variable -> Rename Variable (or L)
# Right-click variable -> Retype Variable (or T)
# Right-click function call -> Override Signature
# Ctrl+L    — Retype return value
# Right-click -> Commit Locals — save variable names to listing

# Fix incorrect decompilation:
# 1. Set correct function signature (F on function start)
# 2. Define structs in Data Type Manager
# 3. Apply struct types to parameters/variables
# 4. Mark calling conventions (stdcall, cdecl, fastcall, thiscall)
# 5. Set storage locations for parameters

# Decompiler options (Edit -> Tool Options -> Decompiler):
#   - Max Payload Bytes: increase for large functions
#   - Simplification Style: "normalize" vs "decompile"
#   - Eliminate unreachable code: toggle for obfuscated binaries
```

### Data Type Manager

```bash
# Creating custom structs:
# 1. Open Data Type Manager (Window -> Data Type Manager)
# 2. Right-click archive -> New -> Structure
# 3. Add fields with types and names
# 4. Apply to memory: right-click address -> Data -> struct_name

# Import types from C headers:
# File -> Parse C Source -> add header files
# Preprocessor: set -I include paths, -D defines

# Example: define a network packet struct
# struct packet_header {
#     uint32_t magic;
#     uint16_t version;
#     uint16_t type;
#     uint32_t length;
#     uint8_t  payload[0];  // flexible array member
# };

# Apply struct to memory range:
# Select address -> T -> choose struct -> Apply
# Array of structs: right-click -> Data -> Create Array
```

---

## Cross-References (Xrefs)

### Finding References

```bash
# On any symbol or address:
# Ctrl+Shift+F — Find all references TO this address
# X            — Show references (quick xref popup)

# Reference types:
# - CALL: function call to this address
# - DATA: data read/write of this address
# - UNCONDITIONAL_JUMP: direct branch
# - CONDITIONAL_JUMP: conditional branch
# - INDIRECTION: indirect reference (pointer table, vtable)

# Trace data flow:
# Right-click variable in decompiler -> Forward Slice
# Right-click variable in decompiler -> Backward Slice
# Highlights all code touching that variable

# Search for string references:
# Search -> For Strings -> filter results
# Double-click string -> X to find code referencing it

# Search for scalar operand values:
# Search -> For Scalars -> enter value
# Finds all instructions using a specific constant (magic numbers, sizes)
```

### Function Call Trees

```bash
# Incoming calls (who calls this function):
# Window -> Function Call Trees -> "Incoming Calls" tab
# Or: right-click function -> References -> Show References To

# Outgoing calls (what does this function call):
# Window -> Function Call Trees -> "Outgoing Calls" tab

# Full call graph:
# Graph -> Function Call Graph
# Adjust depth in graph options

# Find unreferenced functions (potential entry points):
# Search -> For Functions -> filter by "No References"
# These may be callback handlers, thread entries, or dead code
```

---

## Function Graphs

### Control Flow Graphs

```bash
# Toggle graph view: Ctrl+Shift+G (or Space in some configs)
# Graph types:
#   - Function Graph (default) — basic blocks with edges
#   - Call Graph — inter-function relationships

# Graph navigation:
# Mouse wheel — zoom
# Click+drag  — pan
# Double-click block — navigate to address in listing
# Right-click -> Group Vertices — collapse related blocks

# Color coding:
# - Green edges = conditional true (branch taken)
# - Red edges = conditional false (fall-through)
# - Blue edges = unconditional jump
# - Highlighted blocks = current selection

# Export graph:
# Right-click graph -> Export -> PNG/SVG/DOT
# DOT format can be processed with Graphviz
```

---

## Scripting

### Java (GhidraScript)

```bash
# Create script: Script Manager -> New -> Java
# Scripts location: ~/ghidra_scripts/ or <project>/scripts/

# Example: enumerate all functions and their sizes
# @category Analysis
# import ghidra.app.script.GhidraScript;
# import ghidra.program.model.listing.*;
#
# public class ListFunctions extends GhidraScript {
#     @Override
#     public void run() throws Exception {
#         FunctionManager fm = currentProgram.getFunctionManager();
#         for (Function f : fm.getFunctions(true)) {
#             printf("%s @ %s (size: %d)\n",
#                 f.getName(), f.getEntryPoint(), f.getBody().getNumAddresses());
#         }
#     }
# }

# Run via headless mode:
/opt/ghidra/support/analyzeHeadless /path/to/project ProjectName \
  -process binary.exe \
  -postScript ListFunctions.java \
  -noanalysis

# Useful GhidraScript API methods:
# currentAddress       — cursor position
# currentProgram       — active program
# currentSelection     — selected range
# getMonitor()         — progress monitor
# ask*() methods       — user input dialogs
# toAddr(long)         — convert to Address
# getBytes(addr, len)  — read raw bytes
# setBytes(addr, bytes)— write bytes (patch)
# createFunction(addr) — define function
# createLabel(addr, n) — set label
```

### Python via Ghidrathon (Python 3)

```bash
# Install Ghidrathon plugin for Python 3 support
# https://github.com/mandiant/Ghidrathon

# Python scripts use same API as Java but with Python syntax
# Example: find all calls to a specific function
# @category Analysis
# from ghidra.program.model.symbol import ReferenceManager

# target_name = "memcpy"
# fm = currentProgram.getFunctionManager()
# for func in fm.getFunctions(True):
#     if func.getName() == target_name:
#         refs = getReferencesTo(func.getEntryPoint())
#         for ref in refs:
#             caller = getFunctionContaining(ref.getFromAddress())
#             caller_name = caller.getName() if caller else "unknown"
#             print(f"  Called from {caller_name} @ {ref.getFromAddress()}")

# Headless Python script execution:
/opt/ghidra/support/analyzeHeadless /path/to/project ProjectName \
  -process binary.exe \
  -postScript find_memcpy.py \
  -scriptPath /path/to/scripts/
```

---

## Headless Analysis

### Batch Processing

```bash
# Full headless analysis pipeline
/opt/ghidra/support/analyzeHeadless /tmp/ghidra_projects AutoProject \
  -import /samples/*.exe \
  -postScript ExportFunctions.java \
  -postScript FindCrypto.java \
  -log /tmp/ghidra_analysis.log \
  -max-cpu 4

# Analyze without importing (already imported)
/opt/ghidra/support/analyzeHeadless /tmp/ghidra_projects AutoProject \
  -process "malware.exe" \
  -postScript ThreatHunt.java \
  -readOnly   # don't save changes

# Export analysis results
/opt/ghidra/support/analyzeHeadless /tmp/ghidra_projects AutoProject \
  -process "target.elf" \
  -postScript ExportXML.java \
  -deleteProject   # clean up after export

# Scripted bulk triage
for sample in /samples/*.bin; do
  /opt/ghidra/support/analyzeHeadless /tmp/triage BulkProject \
    -import "$sample" \
    -postScript TriageReport.java \
    -deleteProject \
    -log "/tmp/triage_$(basename "$sample").log" 2>&1
done

# Common headless flags:
# -overwrite         — replace existing program in project
# -recursive         — import directory recursively
# -readOnly          — don't save analysis changes
# -deleteProject     — remove project after processing
# -noanalysis        — skip auto-analysis (just import)
# -max-cpu N         — limit CPU threads
# -analysisTimeoutPerFile N — timeout in seconds
```

---

## Binary Diffing and Version Tracking

### Comparing Binaries

```bash
# Version Tracking (built-in binary diffing):
# 1. Import both versions into same project
# 2. Tools -> Version Tracking -> New Session
# 3. Select source (old) and destination (new)
# 4. Run correlators:
#   - Exact Function Hash Match
#   - Exact Data Match
#   - Symbol Name Match
#   - Function Reference Match
#   - Combined Function and Data Reference

# Accept matches:
# Review matches -> Accept (green check) or reject
# Apply accepted matches: copies labels, comments, types from source

# BinDiff integration (requires BinDiff plugin):
# Export both binaries as BinExport files
# Tools -> BinDiff -> Diff current program against...
# Results show: matched/unmatched functions, similarity scores

# Export for external diff tools:
# File -> Export Program -> choose format:
#   - C/C++ (decompiled source for text diff)
#   - ASCII (listing for text diff)
#   - Intel HEX / Binary (raw comparison)
```

---

## P-Code Intermediate Language

### Understanding P-Code

```bash
# P-Code is Ghidra's intermediate representation (IR)
# Abstracts away architecture-specific details
# Enables cross-architecture analysis and decompilation

# View P-Code: in listing, right-click instruction -> PCode
# Or: Window -> PCode

# Key P-Code operations:
# COPY       — register/memory copy
# LOAD       — memory read
# STORE      — memory write
# INT_ADD    — integer addition
# INT_SUB    — integer subtraction
# INT_AND    — bitwise AND
# INT_OR     — bitwise OR
# CALL       — function call
# BRANCH     — unconditional branch
# CBRANCH    — conditional branch
# RETURN     — function return

# P-Code is useful for:
# - Writing architecture-agnostic analysis scripts
# - Understanding how decompiler interprets instructions
# - Debugging decompiler output issues
# - Building custom analyses (taint tracking, symbolic execution)

# Access P-Code from scripts:
# Instruction instr = getInstructionAt(addr);
# PcodeOp[] pcode = instr.getPcode();
# for (PcodeOp op : pcode) {
#     println(op.toString());
# }
```

---

## Plugin Development

### Creating Extensions

```bash
# Generate extension skeleton:
/opt/ghidra/support/buildExtension.sh \
  -e /path/to/extension \
  -g /opt/ghidra

# Extension structure:
# MyExtension/
#   ├── extension.properties    # name, version, description
#   ├── Module.manifest         # module metadata
#   ├── build.gradle            # build configuration
#   ├── src/main/java/          # Java source
#   ├── src/main/resources/     # icons, help
#   └── lib/                    # dependencies

# Build extension:
cd /path/to/MyExtension
gradle -PGHIDRA_INSTALL_DIR=/opt/ghidra

# Install extension:
# File -> Install Extensions -> Add -> select .zip
# Restart Ghidra

# Analyzer plugin template:
# Extend AbstractAnalyzer, implement:
#   - canAnalyze(Program) — return true if applicable
#   - added(Program, AddressSetView, TaskMonitor, MessageLog)
#   - Main analysis logic, register with tool

# Common extension types:
# - Analyzers (auto-run during analysis)
# - Loaders (new file format support)
# - Exporters (new export formats)
# - Plugins (new GUI components)
# - Scripts (standalone analysis tasks)
```

---

## Common Analysis Workflows

### Malware Triage

```bash
# Quick malware triage workflow:
# 1. Import sample -> let auto-analysis complete
# 2. Search -> For Strings -> filter for:
#   - URLs (http://, https://)
#   - File paths (C:\, /tmp/)
#   - Registry keys (HKEY_, Software\)
#   - IP addresses (regex: \d+\.\d+\.\d+\.\d+)
# 3. Check imports: Window -> Symbol Table -> filter "External"
#   - Network: WSAStartup, connect, send, recv, InternetOpenA
#   - File: CreateFileA, WriteFile, DeleteFileA
#   - Process: CreateProcessA, VirtualAlloc, WriteProcessMemory
#   - Registry: RegOpenKeyEx, RegSetValueEx
#   - Crypto: CryptEncrypt, CryptDecrypt, BCryptEncrypt
# 4. Find entry point -> trace execution flow
# 5. Look for anti-analysis: IsDebuggerPresent, NtQueryInformationProcess

# Crypto identification:
# Search -> For Scalars -> common crypto constants:
#   0x67452301 (MD5/SHA-1 init)
#   0x6A09E667 (SHA-256 init)
#   0x61707865 (ChaCha20 "expa")
#   0x63707978 (Salsa20 "cpyx" / "nd 3")
# Or use FindCrypt script/plugin
```

### Firmware Analysis

```bash
# Firmware reverse engineering:
# 1. Extract firmware: binwalk -e firmware.bin
# 2. Identify architecture (ARM, MIPS, PPC, etc.)
# 3. Import with correct processor/endianness:
#   - File -> Import -> Options -> Language
#   - Set base address if known (from bootloader, linker script)
# 4. Define memory map:
#   - Window -> Memory Map
#   - Add regions: Flash, SRAM, MMIO registers
#   - Mark permissions: R/W/X per region
# 5. Find reset vector / entry point
# 6. Identify RTOS (FreeRTOS, VxWorks, ThreadX) by string signatures
# 7. Map peripheral registers using datasheets

# Useful scripts for firmware:
# - FindInstructionByPcode.java — find specific operations
# - DefineIORegisters.java — map MMIO from SVD/datasheet
# - FindCryptConstants.java — locate crypto implementations
```

---

## Tips

- Always let auto-analysis complete before making manual changes; interrupting can leave the database in an inconsistent state
- Use bookmarks liberally to mark interesting addresses during analysis — they persist across sessions
- Create a "common types" archive in Data Type Manager and reuse it across projects for consistent struct definitions
- For stripped binaries, use the Function ID (FID) plugin to identify known library functions by matching against signature databases
- Enable the Decompiler Parameter ID analyzer for significantly better function signatures and calling convention detection
- When analyzing malware, use a snapshot/VM and consider disabling network analyzers that might trigger callbacks
- Export partial analysis often — if the database corrupts you can rebuild from the exported data
- Use headless mode for batch processing and CI/CD integration; it is much faster than GUI analysis
- The Script Manager search box supports regex — use it to quickly find community scripts by functionality
- Version Tracking is invaluable for patch diffing; import old and new versions and let correlators find changes automatically

---

## See Also

- reverse-engineering
- malware-analysis
- checksec

## References

- [Ghidra Official Documentation](https://ghidra.re/ghidra_docs/api/)
- [Ghidra GitHub Repository](https://github.com/NationalSecurityAgency/ghidra)
- [Ghidrathon Python 3 Plugin](https://github.com/mandiant/Ghidrathon)
- [Ghidra Cheat Sheet (SANS)](https://www.sans.org/posters/ghidra-cheat-sheet/)
- [Ghidra P-Code Reference](https://ghidra.re/courses/languages/html/pcoderef.html)
- [The Ghidra Book by Chris Eagle](https://nostarch.com/GhidraBook)
