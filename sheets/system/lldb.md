# LLDB (LLVM Debugger)

macOS default debugger; works on Linux too. Modern LLVM-based replacement for gdb with first-class Python scripting, Swift/Objective-C support, and the canonical `image lookup` symbol resolver.

## Setup

LLDB ships with the macOS Xcode Command Line Tools. On Linux it's a separate package from the LLVM project.

```bash
xcode-select --install
xcrun lldb --version
which lldb
```

```bash
sudo apt-get install -y lldb
sudo apt-get install -y lldb-15 clang-tools-15
sudo dnf install -y lldb
sudo pacman -S lldb
```

```bash
lldb --version
lldb -v
lldb --help | head -40
```

```bash
sudo ln -s /usr/bin/lldb-15 /usr/local/bin/lldb
brew install --cask llvm
export PATH="/opt/homebrew/opt/llvm/bin:$PATH"
```

The historical `lldb-mi` tool (Machine Interface for IDE integration) was removed from LLVM 12+; modern IDEs use the LLDB DAP (Debug Adapter Protocol) implementation `lldb-vscode` or `lldb-dap` instead.

```bash
lldb-vscode --help
lldb-dap --help
which lldb-dap || which lldb-vscode
```

```bash
clang -g -O0 -o myprog myprog.c
clang++ -g -O0 -std=c++20 -o myprog myprog.cc
swiftc -g -Onone -o myprog myprog.swift
go build -gcflags='all=-N -l' -o myprog .
```

## lldb vs gdb — Translation Table

The canonical command-equivalence chart. Most LLDB commands have a verbose form (`breakpoint set -n main`) and a short alias (`b main`).

```bash
b main
breakpoint set --name main
b file.c:42
breakpoint set --file file.c --line 42
break main
b -[NSString length]
breakpoint set --selector length
```

```bash
run
r
process launch
process launch -- arg1 arg2
process launch --stop-at-entry
```

```bash
continue
c
process continue
```

```bash
next
n
thread step-over
step
s
thread step-in
finish
thread step-out
```

```bash
print var
p var
expression var
expr var
po obj
expression -O -- obj
```

```bash
bt
thread backtrace
bt all
thread backtrace all
bt 5
thread backtrace -c 5
```

```bash
thread list
thread select 2
process status
```

```bash
breakpoint list
br list
breakpoint disable 1
breakpoint enable 1
breakpoint delete 1
```

```bash
memory read 0x100000000
memory read --size 4 --format x --count 16 0x100000000
x/16xw 0x100000000
```

| gdb | lldb (full) | lldb (alias) |
|---|---|---|
| `break main` | `breakpoint set --name main` | `b main` |
| `break file.c:42` | `breakpoint set --file file.c --line 42` | `b file.c:42` |
| `break Foo::*` | `breakpoint set --regex "^Foo::.*"` | `b -r ^Foo::.*` |
| `condition 1 x>10` | `breakpoint modify -c "x>10" 1` | — |
| `run arg1 arg2` | `process launch -- arg1 arg2` | `r arg1 arg2` |
| `continue` | `process continue` | `c` |
| `next` | `thread step-over` | `n` |
| `step` | `thread step-in` | `s` |
| `nexti` | `thread step-inst-over` | `ni` |
| `stepi` | `thread step-inst` | `si` |
| `finish` | `thread step-out` | `finish` |
| `until N` | `thread until N` | `u N` |
| `print x` | `expression x` | `p x` |
| `print/x x` | `expression -f x -- x` | `p/x x` (alias) |
| `info threads` | `thread list` | — |
| `thread N` | `thread select N` | — |
| `bt` | `thread backtrace` | `bt` |
| `info breakpoints` | `breakpoint list` | `br l` |
| `info registers` | `register read` | — |
| `info frame` | `frame info` | — |
| `info locals` | `frame variable` | — |
| `info args` | `frame variable -a` | — |
| `info shared` | `image list` | — |
| `info functions Foo` | `image lookup -n Foo` | — |
| `disas` | `disassemble` | `dis` |
| `disas /m` | `disassemble -m` | — |
| `x/4xw addr` | `memory read --size 4 --format x --count 4 addr` | `x/4xw addr` |
| `set var x=5` | `expression x = 5` | `p x = 5` |
| `watch var` | `watchpoint set variable var` | — |
| `attach PID` | `process attach --pid PID` | — |
| `detach` | `process detach` | — |
| `quit` | `quit` | `q` |
| `define cmd` | `command alias cmd ...` | — |

```bash
help
help breakpoint
help breakpoint set
apropos thread
gdb-remote localhost:1234
```

## Starting

LLDB launches in five canonical ways: with a binary, attached to a PID, with a core file, with arguments, or empty.

```bash
lldb /path/to/myprog
lldb ./myprog
lldb -- ./myprog arg1 arg2 arg3
lldb -p 12345
lldb --attach-pid 12345
lldb --attach-name Safari
lldb -c /cores/core.12345
lldb -c core.dump ./myprog
```

```bash
lldb
(lldb) target create ./myprog
(lldb) settings set target.run-args arg1 arg2
(lldb) settings show target.run-args
(lldb) run
```

```bash
lldb -o "b main" -o "run" ./myprog
lldb -s commands.lldb ./myprog
lldb -O "settings set stop-disassembly-display always" ./myprog
lldb --no-lldbinit ./myprog
lldb -X ./myprog
lldb --batch -o "run" -o "bt" -k "quit" ./myprog
```

```bash
lldb /usr/bin/ls
(lldb) settings set target.run-args -la /tmp
(lldb) process launch
```

The canonical workflow: launch lldb with the binary, set breakpoints, then `run` from inside the prompt. The dashes (`--`) separate lldb's own arguments from the inferior's arguments.

## Command Structure

LLDB uses a strict "noun verb [options] [args]" hierarchy: `breakpoint set`, `thread step-over`, `target create`, `frame variable`. Every long form has built-in aliases.

```bash
help breakpoint
help breakpoint set
help thread
help thread step-over
```

```bash
command alias bf breakpoint set --file %1 --line %2
command alias bn breakpoint set --name %1
command unalias bf
command alias
```

```bash
command source ~/.lldbinit-extra
command script import ~/scripts/mycmds.py
```

```bash
apropos memory
apropos breakpoint
type lookup int
help expression
```

The "object verb subject" pattern is rigid:
- `breakpoint set/list/delete/modify/enable/disable/command/name`
- `thread list/select/step-in/step-over/step-out/step-inst/until/return/continue/backtrace`
- `frame select/info/variable`
- `target create/list/select/delete/modules/symbols`
- `process launch/attach/detach/continue/kill/status/signal/connect/load`
- `memory read/write/find/region/history`
- `register read/write/info`
- `image list/lookup/dump/show-unwind`
- `expression -- ...` (or `expr` / `p` / `print` / `po`)
- `watchpoint set/list/delete/modify/enable/disable/command`
- `settings set/show/list/clear/append`
- `command alias/unalias/source/script/regex`
- `type lookup/summary/format/synthetic/category`

## Breakpoints

The `breakpoint set` family is exhaustive — by name, file:line, regex, address, selector, or condition. Every breakpoint gets a numeric ID for later reference.

```bash
b main
b myfile.c:42
b MyClass::method
breakpoint set --name main
breakpoint set --file myfile.c --line 42
breakpoint set --address 0x100003f50
```

```bash
breakpoint set --name malloc
breakpoint set --name malloc --shlib libsystem_malloc.dylib
breakpoint set --regex "^MyClass::.*"
breakpoint set --func-regex "^test_.*"
breakpoint set --selector viewDidLoad
breakpoint set --method foo
```

```bash
breakpoint set --name main --condition 'argc > 1'
breakpoint set --file foo.c --line 42 --condition 'i == 100'
breakpoint modify -c 'x > 10' 1
breakpoint modify -c '' 1
breakpoint modify --ignore-count 5 1
```

```bash
breakpoint list
br l
breakpoint disable 1
breakpoint enable 1
breakpoint delete 1
breakpoint delete
breakpoint clear --file foo.c --line 42
```

```bash
breakpoint set --name main --one-shot
breakpoint set --name main -o true
tbreak main
```

```bash
breakpoint command add 1
> bt 5
> frame variable
> continue
> DONE
breakpoint command list 1
breakpoint command delete 1
```

```bash
breakpoint command add -s python 1
> print('hit', frame.GetFunctionName())
> return False
> DONE
```

```bash
breakpoint set --name foo --auto-continue 1
breakpoint set --name foo --thread-index 1
breakpoint set --name foo --queue-name com.apple.main-thread
breakpoint set --name foo --thread-name worker
```

```bash
breakpoint name configure mybp --condition 'x > 0'
breakpoint name add mybp 1
breakpoint set --breakpoint-name mybp --name foo
breakpoint name list
```

## Watchpoints

Watchpoints fire when memory is read or written. Hardware watchpoints are limited (typically 4 on x86-64, 2-4 on ARM64) and the variable must fit in a register-sized chunk.

```bash
watchpoint set variable my_var
watchpoint set variable -w write my_var
watchpoint set variable -w read my_var
watchpoint set variable -w read_write my_var
watchpoint set expression -- &my_var
watchpoint set expression --size 8 -- 0x100008000
```

```bash
watchpoint list
watchpoint disable 1
watchpoint enable 1
watchpoint delete 1
watchpoint modify -c 'my_var > 100' 1
```

```bash
watchpoint command add 1
> bt 3
> frame variable
> continue
> DONE
```

```bash
watchpoint set expression -- (char*)&my_struct.field
watchpoint set expression --size 4 -- (uint32_t*)&my_struct.field
```

The same hardware-support limitation as gdb: too many watchpoints fail with `error: Watchpoint creation failed (No hardware resources available)`. Reduce active watchpoints or watch smaller regions.

## Execution

The execution commands map gdb directly: `run`, `continue`, `next`, `step`, `finish`, plus LLDB's instruction-level variants and the `thread until` line target.

```bash
run
r
process launch
process launch -- arg1 arg2
process launch --stop-at-entry
process launch --environment FOO=bar
process launch --working-dir /tmp
process launch --tty
process launch --no-stdio
```

```bash
continue
c
process continue
process continue --ignore-count 5
```

```bash
next
n
thread step-over
step
s
thread step-in
thread step-in --target foo
thread step-in --end-line 42
finish
thread step-out
```

```bash
nexti
ni
thread step-inst-over
stepi
si
thread step-inst
```

```bash
thread until 100
thread until 100 105 110
thread until --address 0x100003f50
```

```bash
thread return
thread return 42
thread return -- (int)0
```

```bash
process kill
kill
process detach
process signal SIGUSR1
process interrupt
process handle SIGUSR1 --notify true --pass true --stop true
```

```bash
process status
process status --verbose
```

## Stack

Stack frames are 0-indexed (innermost = 0). `frame variable` shows locals + args, `frame info` shows current PC.

```bash
bt
thread backtrace
thread backtrace -c 10
thread backtrace all
bt all
thread backtrace --extended true
```

```bash
frame select 0
frame select 5
frame select --relative 1
frame select --relative -1
up
down
up 3
down 3
```

```bash
frame info
frame variable
frame variable arg1
frame variable -a
frame variable --regex "^my_.*"
frame variable --depth 3
frame variable --show-types
frame variable --raw
```

```bash
frame variable --location
frame variable --format x my_var
frame variable -F x my_var
frame variable --summary "${var.field}"
```

```bash
thread backtrace --start 5 --count 10
thread backtrace --extended true
```

The frame-pointer-omitted (`-fomit-frame-pointer`) builds may produce incomplete backtraces; rebuild with `-fno-omit-frame-pointer` for clean stacks.

## Examining

LLDB has no `print/format` shortcut — use `expression -f FORMAT --` or `frame variable --format FORMAT`. The `expression` family evaluates real C/C++/Obj-C/Swift expressions in the inferior's context.

```bash
expression my_var
expr my_var
p my_var
print my_var
expression my_var + 1
expression my_func(42)
```

```bash
expression -f x -- my_var
expression -f hex -- my_var
expression -f octal -- my_var
expression -f binary -- my_var
expression -f decimal -- my_var
expression -f char -- my_var
expression -f cstring -- my_ptr
expression -f pointer -- my_ptr
p/x my_var
```

```bash
expression -O -- my_obj
po my_obj
po [my_obj description]
po self
```

```bash
expression -l objc -- [obj method]
expression -l swift -- self.property
expression -l c++ -- (MyClass*)ptr
```

```bash
expression -- int $x = 42
expression -- $x + 1
expression -- (void*)$rax
```

```bash
memory read 0x100000000
memory read --size 4 --format x --count 16 0x100000000
memory read -s 4 -f x -c 16 0x100000000
memory read -s 8 -f x -c 32 $rsp
memory read --force 0x100000000
memory read --outfile /tmp/dump.bin --binary 0x100000000
```

```bash
memory read --format c --count 256 my_string
memory read --format x --size 1 --count 64 buffer
x my_var
x/16xw $rsp
x/16gx 0x100000000
x/s my_string
```

```bash
memory write 0x100000000 0xff 0x00 0xff 0x00
memory write --infile /tmp/data.bin 0x100000000
memory write -s 4 0x100000000 0xdeadbeef
```

```bash
memory find -e 'hello' 0x100000000 0x100100000
memory find -s "needle" 0x100000000 0x100100000
memory find --expression '(uint64_t)0xdeadbeef' 0x100000000 0x100100000
```

```bash
memory region 0x100000000
memory history 0x100000000
```

## Type System

LLDB has a powerful type-formatter system: type summaries (one-line strings) and synthetic providers (Python-driven children).

```bash
type lookup int
type lookup MyClass
type lookup --show-help std::vector
type lookup -r ".*Foo.*"
```

```bash
type summary list
type summary add --summary-string "size=${var.size}" MyVector
type summary add -s "x=${var.x}, y=${var.y}" Point
type summary add --inline-children --omit-names Vec3
type summary delete MyVector
type summary clear
```

```bash
type format add --format hex MyHandle
type format list
type format delete MyHandle
type format clear
```

```bash
type synthetic add --python-class my_module.MyVecProvider MyVec
type synthetic list
type synthetic delete MyVec
```

```bash
type category list
type category enable VectorTypes
type category disable VectorTypes
type category list libcxx
```

```bash
type filter add --child x --child y --child z Vec3
type filter list
```

```bash
type summary add --regex --summary-string "len=${var.length}" "^MyArray<.*>$"
```

The libc++ formatters (`std::string`, `std::vector`, `std::map`, etc.) ship with LLDB and are auto-loaded when libc++ symbols are detected.

## Threads

Each thread has an index (1-based, stable across the session) and a Thread ID (TID). LLDB's `thread` noun owns step, list, backtrace, until, and return.

```bash
thread list
thread info
thread info 2
thread select 2
thread select --thread-index 2
thread select --thread-id 0x1234abcd
```

```bash
thread step-over
thread step-in
thread step-out
thread step-inst
thread step-inst-over
thread continue
```

```bash
thread step-in --target my_func
thread step-in --step-in-target my_func
thread step-in --end-line 42
thread step-in --run-mode this-thread
thread step-in --run-mode all-threads
```

```bash
thread until 100
thread until 100 110 120
thread until --address 0x100003f50
```

```bash
thread return
thread return -- 42
thread jump --line 50
thread jump --address 0x100003f50
```

```bash
thread backtrace
thread backtrace all
thread backtrace --count 5
thread backtrace --start 2 --count 10
thread backtrace --extended true
```

```bash
thread plan list
thread plan discard 1
thread plan prune
```

```bash
process status
process status --verbose
thread list
```

For deadlock diagnosis, `thread backtrace all` is the universal first move; look for threads blocked in `pthread_mutex_lock`, `pthread_cond_wait`, or `__psynch_cvwait`.

## Registers

Register read/write is mostly architecture-symmetric; `$rax` syntax works in `expression`.

```bash
register read
register read --all
register read rax rbx rcx
register read --format hex rax
register read -f x rax
register read -f d rax
```

```bash
register read --set general
register read --set floating
register read --set vector
register read --set "Floating Point"
```

```bash
register write rax 0x10
register write rip 0x100003f50
register write rflags 0x246
```

```bash
register info rax
register info pc
```

```bash
expression $rax
expression $rax + 8
expression -- $rax = 42
memory read $rsp
memory read $rsp -c 16 -s 8 -f x
```

x86-64 generic registers: `rax`, `rbx`, `rcx`, `rdx`, `rsi`, `rdi`, `rbp`, `rsp`, `r8`-`r15`, `rip`, `rflags`. ARM64: `x0`-`x30`, `sp`, `pc`, `cpsr`. Generic aliases work cross-arch: `$pc`, `$sp`, `$fp`, `$arg1`, `$arg2`.

## Source

Source listing and source-map remapping for cross-machine builds.

```bash
source list
list
l
source list -c 30
source list -n main
source list -f myfile.c -l 100
source list -f myfile.c -l 100 -c 20
source list -a 0x100003f50
```

```bash
source info
source info -f myfile.c
```

```bash
settings set target.source-map /build/path /local/path
settings set target.source-map /build/old1 /local/new1 /build/old2 /local/new2
settings show target.source-map
settings clear target.source-map
settings append target.source-map /build/extra /local/extra
```

```bash
settings set target.exec-search-paths /opt/build/bin
settings set target.debug-file-search-paths /opt/build/dsym
settings set symbols.enable-external-lookup true
```

The source-map setting is mandatory when debugging binaries built in CI containers — DWARF embeds absolute build paths.

## Symbols

The `image` noun owns shared-library introspection. `image lookup` is the canonical "where is this symbol" tool.

```bash
image list
image list --verbose
image list myprog
image list -f
image list -g
image list -t
```

```bash
image lookup --name main
image lookup -n main
image lookup -n malloc
image lookup --regex --name "^MyClass::.*"
image lookup -r -n "^test_.*"
```

```bash
image lookup --address 0x100003f50
image lookup -a 0x100003f50
image lookup --address $pc
image lookup --address $pc --verbose
```

```bash
image lookup --type MyStruct
image lookup -t MyStruct
image lookup -t int
```

```bash
image dump symtab
image dump symtab myprog
image dump sections
image dump sections myprog
image dump line-table myfile.c
image dump objfile myprog
```

```bash
image show-unwind --name main
image show-unwind -a 0x100003f50
```

```bash
target symbols add /path/to/symbols.dSYM
target symbols add --shlib libfoo.dylib /path/to/foo.dSYM
add-dsym /path/to/symbols.dSYM
```

```bash
settings set symbols.enable-external-lookup true
settings set plugin.symbol-file.dwarf.comp-dir-symlink-paths /tmp/build
```

The `image lookup -n` command searches mangled and demangled names. For C++ overloads, use the regex form: `image lookup -r -n "^Foo::method.*"`.

## Memory

Read, write, find, and inspect memory regions.

```bash
memory read 0x100000000
memory read --size 8 --format x --count 16 0x100000000
memory read -s 1 -f c -c 256 my_string
memory read -f x -s 4 -c 1 $rsp
```

```bash
memory write 0x100000000 0xff 0x00 0xff 0x00
memory write -s 4 0x100000000 0xdeadbeef
memory write --infile /tmp/payload.bin 0x100000000
```

```bash
memory find -e 'hello' 0x100000000 0x100100000
memory find -s "ELF" 0x100000000 0x100100000
memory find -e '(uint64_t)0xdeadbeef' 0x100000000 0x100100000
memory find --count 5 -s "match" 0x100000000 0x100100000
```

```bash
memory region 0x100000000
memory region --all
memory history 0x100000000
```

```bash
memory read --outfile /tmp/dump.bin --binary 0x100000000 0x100100000
memory read --outfile /tmp/dump.txt --force 0x100000000
```

```bash
expression -- (void*)malloc(1024)
expression -- memset((void*)0x100008000, 0, 1024)
expression -- memcpy((void*)dst, (void*)src, 64)
```

The default `memory read` chunk is 32 bytes formatted as `bytes-with-ASCII`. The `memory history` command works only with MallocStackLogging or AddressSanitizer enabled.

## Python Scripting

LLDB embeds a full Python 3 interpreter via the SBAPI (Script Bridge API). All commands are scriptable from day 1.

```bash
script
>>> print(lldb.frame.GetFunctionName())
>>> print(lldb.thread.GetNumFrames())
>>> for f in lldb.thread:
...     print(f.GetFunctionName())
>>> exit()
```

```bash
script print(lldb.frame.GetFunctionName())
script print(hex(lldb.frame.FindVariable("my_var").GetValueAsUnsigned()))
script lldb.target.BreakpointCreateByName("malloc")
```

```bash
script import os; os.system("ls /tmp")
script import sys; print(sys.version)
```

```bash
command script import ~/scripts/mycmds.py
command script add -f mymodule.my_function mycmd
command script list
command script delete mycmd
command script clear
```

```python
def my_function(debugger, command, result, internal_dict):
    target = debugger.GetSelectedTarget()
    process = target.GetProcess()
    thread = process.GetSelectedThread()
    frame = thread.GetSelectedFrame()
    result.AppendMessage(f"PC: {hex(frame.GetPC())}")
    result.AppendMessage(f"Function: {frame.GetFunctionName()}")
```

```bash
script
>>> debugger = lldb.SBDebugger.Create()
>>> target = debugger.CreateTarget("./myprog")
>>> bp = target.BreakpointCreateByName("main")
>>> process = target.LaunchSimple(None, None, ".")
```

The canonical SBAPI objects: `SBDebugger`, `SBTarget`, `SBProcess`, `SBThread`, `SBFrame`, `SBValue`, `SBBreakpoint`, `SBSymbol`, `SBAddress`, `SBModule`. Full API docs at `lldb.llvm.org/python_api.html`.

```bash
breakpoint set --name foo --script-type python --python-function mymodule.bp_callback
breakpoint command add -s python -F mymodule.bp_callback 1
```

```python
def bp_callback(frame, bp_loc, internal_dict):
    print("hit:", frame.GetFunctionName())
    print("rax:", hex(frame.FindRegister("rax").GetValueAsUnsigned()))
    return False
```

## .lldbinit

The startup file. `~/.lldbinit` runs once on every lldb session; `./.lldbinit` runs only if `settings set target.load-cwd-lldbinit true` (security-sensitive).

```bash
cat > ~/.lldbinit <<'EOF'
settings set target.x86-disassembly-flavor intel
settings set target.skip-prologue false
settings set target.inline-breakpoint-strategy always
settings set stop-disassembly-display always
settings set stop-disassembly-count 8
settings set stop-line-count-before 3
settings set stop-line-count-after 3
settings set frame-format "frame #${frame.index}: ${frame.pc}{ ${module.file.basename}\`${function.name-with-args}{${frame.no-debug}${function.pc-offset}}}{ at ${line.file.basename}:${line.number}}\n"
command alias ll thread list
command alias bb breakpoint list
command alias hh help
command regex bf 's/(.+):(.+)/breakpoint set --file "%1" --line %2/'
command script import ~/.lldb/mycmds.py
EOF
```

```bash
settings set target.load-cwd-lldbinit true
ls -la ~/.lldbinit
lldb --no-lldbinit ./myprog
```

```bash
settings list
settings list target
settings show target.x86-disassembly-flavor
settings show frame-format
settings clear target.x86-disassembly-flavor
```

```bash
settings set target.env-vars FOO=bar BAZ=qux
settings set target.run-args --verbose --input data.txt
settings set target.disable-aslr false
settings set target.detach-on-error false
```

```bash
settings set auto-confirm true
settings set prompt "(lldb) "
settings set use-color true
```

The canonical productivity boilerplate: Intel-flavor disassembly (instead of AT&T), full disassembly on stop, and 3-line source context. Add to `~/.lldbinit` once.

## macOS-Specific

LLDB on macOS requires codesigning and entitlements to debug arbitrary processes due to System Integrity Protection (SIP) and Apple Mobile File Integrity (AMFI).

```bash
codesign --display --verbose=4 /usr/bin/lldb
codesign --display --entitlements - /usr/bin/lldb
csrutil status
```

```bash
sudo lldb -p $(pgrep MyApp)
sudo lldb /path/to/myapp
```

```bash
cat > /tmp/lldb.entitlements <<'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.cs.debugger</key>
    <true/>
    <key>com.apple.security.get-task-allow</key>
    <true/>
</dict>
</plist>
EOF
codesign -s - --entitlements /tmp/lldb.entitlements -f /usr/local/bin/lldb-custom
```

```bash
ls -la /Library/Developer/CommandLineTools/usr/bin/debugserver
ls /Applications/Xcode.app/Contents/SharedFrameworks/LLDB.framework/Resources/debugserver
xcrun -find debugserver
```

```bash
debugserver localhost:1234 ./myprog
debugserver localhost:1234 --attach=12345
debugserver --help
```

```bash
lldb
(lldb) process connect connect://localhost:1234
(lldb) process connect connect://10.0.0.5:1234
```

```bash
xcrun lldb --version
xcode-select -p
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
```

For SIP-protected processes (anything in `/System/`, `/usr/`, hardened-runtime apps): debugging is blocked even with sudo unless SIP is partially disabled in Recovery Mode (`csrutil enable --without debug`). Not recommended for production machines.

```bash
spctl --status
spctl --assess --type execute /path/to/MyApp.app
sudo log stream --predicate 'subsystem == "com.apple.AppleMobileFileIntegrity"'
```

## Multi-Process

LLDB can manage multiple targets and inferiors in one session.

```bash
settings set target.detach-on-error false
target create ./prog1
target create ./prog2
target list
target select 0
target select 1
target delete 0
```

```bash
process attach --pid 12345
process attach --name Safari
process attach --name Safari --waitfor
process list
process detach
```

```bash
settings set target.process.detach-keeps-stopped false
settings set target.process.follow-fork-mode child
settings set target.process.follow-fork-mode parent
```

```bash
target list
target select 1
process status
thread list
```

```bash
target stop-hook add --name foo
> bt
> frame variable
> DONE
target stop-hook list
target stop-hook delete 1
target stop-hook disable 1
```

```bash
target create --core /cores/core.dump ./myprog
target create --symfile myprog.dSYM ./myprog
target create --arch x86_64 ./myprog
```

The fork-mode setting controls whether lldb follows child or parent across `fork()`. The default is `parent` on macOS.

## Core Files

Post-mortem debugging: open a core dump and inspect state without running the program.

```bash
ulimit -c
ulimit -c unlimited
sysctl kern.corefile
sudo sysctl -w kern.corefile=/cores/core.%P
ls -la /cores/
```

```bash
lldb -c /cores/core.12345
lldb -c core.dump ./myprog
lldb --core core.dump ./myprog
target create --core core.dump ./myprog
```

```bash
(lldb) thread list
(lldb) thread backtrace all
(lldb) frame select 0
(lldb) frame variable
(lldb) register read
(lldb) image list
(lldb) memory region 0x100000000
```

```bash
process save-core /tmp/myprog.core
process save-core --plugin-name minidump /tmp/myprog.dmp
```

```bash
launchctl limit core unlimited
sudo /System/Library/CoreServices/CrashReporter.app
ls /Library/Logs/DiagnosticReports/
ls ~/Library/Logs/DiagnosticReports/
```

```bash
echo '/cores/core.%e.%p' | sudo tee /proc/sys/kernel/core_pattern
sysctl -w kernel.core_pattern='/cores/core.%e.%p'
ulimit -c unlimited
./myprog
ls /cores/
lldb -c /cores/core.myprog.12345 ./myprog
```

On macOS, core files are not generated by default; the kernel routes crashes to CrashReporter. Set `ulimit -c unlimited` and ensure `/cores` is writable. Use `process save-core` from inside lldb to dump on demand.

## Remote Debugging

The split: `debugserver` (macOS) or `lldb-server` (Linux) on the target, `lldb` on the host.

```bash
debugserver localhost:1234 ./myprog
debugserver localhost:1234 --attach=12345
debugserver *:1234 ./myprog
debugserver --help
```

```bash
lldb-server platform --listen "*:1234" --server
lldb-server gdbserver localhost:1234 ./myprog
lldb-server gdbserver --attach 12345 localhost:1234
```

```bash
lldb
(lldb) platform select remote-linux
(lldb) platform connect connect://10.0.0.5:1234
(lldb) target create ./myprog
(lldb) run
```

```bash
lldb
(lldb) process connect connect://10.0.0.5:1234
(lldb) target create ./myprog
(lldb) continue
```

```bash
lldb -o "platform select remote-linux" \
     -o "platform connect connect://10.0.0.5:1234" \
     -o "target create ./myprog" \
     -o "run"
```

```bash
ssh user@target "debugserver localhost:1234 --attach=$(pgrep myprog)" &
ssh -L 1234:localhost:1234 user@target
lldb
(lldb) process connect connect://localhost:1234
```

```bash
settings set platform.plugin.remote-android.package-name com.example.app
platform select remote-android
platform connect connect://device:5039
```

Cross-platform debugging: an x86_64 Linux target debugged from an arm64 macOS host works seamlessly because lldb understands the wire format from either side.

## Objective-C Specifics

LLDB knows about the Objective-C runtime, message dispatch, and Foundation/AppKit/UIKit types out of the box.

```bash
po obj
po [obj description]
po self
po [self class]
po NSStringFromClass([self class])
po [obj respondsToSelector:@selector(foo)]
```

```bash
expression -O -- obj
expression -l objc -- [obj method]
expression -l objc -- (NSString*)[NSString stringWithUTF8String:"hello"]
expression -l objc -- (id)[NSArray arrayWithObjects:@"a", @"b", nil]
```

```bash
breakpoint set --selector viewDidLoad
breakpoint set --selector dealloc
breakpoint set --name "-[NSObject init]"
breakpoint set --name "+[NSString stringWithFormat:]"
```

```bash
breakpoint set --regex "-\[MyClass .*\]"
breakpoint set --regex "\+\[MyClass .*\]"
breakpoint set --regex "-\[.* viewDidLoad\]"
```

```bash
po [NSThread callStackSymbols]
po [[NSBundle mainBundle] bundlePath]
po [[NSProcessInfo processInfo] environment]
po [[NSFileManager defaultManager] currentDirectoryPath]
```

```bash
expression -O -- (id)NSApp
expression -O -- (id)[NSApp keyWindow]
expression -l objc -- (BOOL)[obj isKindOfClass:[NSString class]]
```

```bash
type summary add --summary-string "${var._fname%@} ${var._lname%@}" Person
type summary add --python-script 'return "name=" + valobj.GetChildMemberWithName("_name").GetSummary()' MyObj
```

LLDB injects the Foundation framework symbols when the inferior loads `Foundation.framework`. The `po` shortcut equals `expression -O --` (object-print).

## Swift Specifics

The full LLDB Swift integration ships in Xcode's lldb. Open-source LLDB has partial Swift support.

```bash
po self
po self.property
expression --language swift -- self.viewModel
expression -l swift -- self.array.count
expression -l swift -- self.dict.keys.sorted()
```

```bash
po type(of: self)
po String(describing: self)
po self.debugDescription
po Mirror(reflecting: self).children.map { $0.label }
```

```bash
expression -l swift -- import Foundation
expression -l swift -- let x = 42
expression -l swift -- $x + 1
```

```bash
breakpoint set --name MyApp.MyClass.viewDidLoad
breakpoint set --func-regex "MyApp.MyClass.*"
breakpoint set --selector viewDidLoad
```

```bash
po self.children.first?.value
po self.array[0]
expression -l swift -- self.callback?()
```

```bash
xcrun --toolchain swift lldb
xcrun -find swift
xcrun -sdk macosx --show-sdk-path
```

The canonical "po self" in a Swift breakpoint context prints the Swift mirror representation. Mixed Swift/Obj-C targets accept `expression -l objc --` to drop into Obj-C dispatch and `expression -l swift --` to switch back.

## Common Recipes

The recipes that come up daily.

```bash
lldb -c /cores/core.12345 ./myprog
(lldb) thread backtrace all
(lldb) frame select 0
(lldb) frame variable
(lldb) register read
(lldb) image lookup --address $pc
```

```bash
sudo lldb -p $(pgrep -f hung_app)
(lldb) thread backtrace all > /tmp/threads.txt
(lldb) script
>>> for t in lldb.process:
...     for f in t:
...         if 'lock' in (f.GetFunctionName() or ''):
...             print(t.idx, f)
>>> exit()
(lldb) detach
```

```bash
b file.c:42
breakpoint modify -c "i == 100" 1
breakpoint command add 1
> bt 5
> frame variable i
> continue
> DONE
run
```

```bash
b main
run
thread until 100
```

```bash
b foo
run
finish
frame variable
```

```bash
b malloc
breakpoint modify -c '$rdi > 1024' 1
breakpoint command add 1
> bt 3
> continue
> DONE
run
```

```bash
b -[NSException raise]
b objc_exception_throw
run
```

```bash
b __sanitizer::Report
b __asan::ReportError
run
```

```bash
breakpoint set --file foo.c --line 42 --auto-continue 1
breakpoint command add 1
> p i
> DONE
run
```

```bash
target create ./myprog
b main
run
register read --set general
disassemble --frame
memory read $rsp -c 16 -s 8 -f x
```

```bash
process save-core /tmp/snapshot.core
expression -- (void)abort()
```

```bash
breakpoint set --regex "^test_.*"
breakpoint command add 1
> bt 1
> continue
> DONE
run
```

## Common Errors

The error messages you actually see, with fixes.

```bash
(lldb) run
Process 12345 launched: './myprog' (x86_64)
Process 12345 exited with status = 1 (0x00000001)
(lldb) process status
(lldb) script print(lldb.process.GetExitStatus())
```

Fix: the program exited cleanly. Set a breakpoint *before* `run`, e.g. `b main`, then `run`.

```bash
error: attach failed: Failed to attach to process 12345: error: __attached__
```

Fix on macOS: `sudo lldb -p 12345`, or codesign your custom lldb with `com.apple.security.cs.debugger`. Fix on Linux: `sudo sysctl kernel.yama.ptrace_scope=0` or run as root.

```bash
error: 'my_var' is not in the lldb expression evaluator path
```

Fix: that symbol doesn't exist in the current frame. Verify with `frame variable` (locals) and `image lookup -n my_var` (globals). Recompile with `-g` if the symbol should be present.

```bash
error: no debug symbols available
warning: (x86_64) /path/to/myprog empty dSYM file detected
```

Fix: rebuild with `-g`. For C/C++/Obj-C: `clang -g -O0 -o myprog myprog.c`. For Swift: `swiftc -g -Onone`. For Go: `go build -gcflags='all=-N -l'`. For Rust: `cargo build` (debug profile is default).

```bash
error: process attach failed: operation not permitted
```

Fix on macOS: codesigning + entitlements + SIP. Try `sudo lldb -p PID` first. For protected processes, partial SIP disable in Recovery: `csrutil enable --without debug` (NOT recommended on production machines).

```bash
error: Couldn't apply expression side effects
```

Fix: the expression had a memory side effect (function call) that was rolled back. Use `expression --allow-jit true --` or run the side-effect manually.

```bash
warning: (x86_64) libfoo.dylib unable to load symbols
```

Fix: either the dSYM is missing (`add-dsym /path/to/foo.dSYM`) or the binary was stripped. Rebuild from source with debug info.

```bash
error: Watchpoint creation failed (No hardware resources available)
```

Fix: hardware watchpoints are limited to ~4 on x86-64. Delete unused watchpoints (`watchpoint delete N`) or reduce the size of the watched region.

```bash
error: invalid target, create a target with 'target create <executable>'
```

Fix: you're at an empty lldb prompt. `target create ./myprog` or restart with `lldb ./myprog`.

```bash
error: this version of lldb does not support source-level debugging of Go programs
```

Fix: use Delve (`dlv`) for Go debugging. LLDB has minimal Go support — only register/disassembly level.

## Common Gotchas

Each gotcha lists the broken case and the fixed case.

```bash
clang -O2 -o myprog myprog.c
lldb ./myprog
b main
run
warning: no debug symbols available
```

```bash
clang -g -O0 -o myprog myprog.c
lldb ./myprog
b main
run
```

```bash
clang -g -o myprog myprog.c
strip myprog
lldb ./myprog
warning: (x86_64) myprog empty dSYM file detected
```

```bash
clang -g -O0 -o myprog myprog.c
lldb ./myprog
```

```bash
sudo lldb -p $(pgrep WindowServer)
error: attach failed: operation not permitted
```

```bash
sudo lldb -p $(pgrep my_user_app)
```

```bash
apt install lldb
lldb --version
lldb version 6.0.0
```

```bash
wget https://apt.llvm.org/llvm.sh
chmod +x llvm.sh
sudo ./llvm.sh 17 all
sudo ln -sf /usr/bin/lldb-17 /usr/local/bin/lldb
lldb --version
```

```bash
(lldb) p my_objc_obj
(NSString *) $0 = 0x0000600000c000a0
```

```bash
(lldb) po my_objc_obj
hello world
```

```bash
clang -g -O3 -o myprog myprog.c
lldb ./myprog
b main
run
frame variable
warning: variable 'x' was optimized out
```

```bash
clang -g -O0 -o myprog myprog.c
lldb ./myprog
b main
run
frame variable
```

```bash
breakpoint set --regex "Foo::bar"
```

```bash
breakpoint set --regex "^Foo::bar$"
breakpoint set --method bar --shlib libfoo.dylib
```

```bash
expression my_global
error: 'my_global' is not in the lldb expression evaluator path
```

```bash
image lookup -n my_global
expression -- *(int*)0x100008000
expression -- (int)my_global
```

```bash
memory read 0x0
error: memory read failed for 0x0
```

```bash
memory region 0x100000000
memory read 0x100000000
```

```bash
process attach --name MyApp
error: more than one process named "MyApp"
```

```bash
pgrep -lf MyApp
process attach --pid 12345
```

```bash
b file.c:42
warning: 1 location was added to the breakpoint but the file 'file.c' was not located
```

```bash
settings set target.source-map /build/path /local/path
breakpoint clear --file file.c --line 42
b file.c:42
```

## Comparison with gdb

| Dimension | gdb | lldb |
|---|---|---|
| Default platform | Linux, BSDs | macOS, increasingly Linux |
| License | GPLv3+ | Apache-2.0 + LLVM exception |
| Scripting | Python (added later, less integrated) | Python (native, SBAPI from day 1) |
| C++ support | Excellent (longest history) | Excellent (Clang AST integration) |
| Swift support | None | Native (in Xcode lldb) |
| Obj-C support | Limited | Native, Foundation-aware |
| Wire protocol | gdb-remote (gdbserver) | gdb-remote (debugserver/lldb-server) |
| Type formatters | Python pretty-printers | Native `type summary` + Python `type synthetic` |
| Disassembly | AT&T (default) | AT&T (default), Intel via setting |
| Reverse debugging | Yes (`record`) | No |
| Process record/replay | Yes (rr-compatible) | No (use `rr` separately) |
| TUI mode | `gdb -tui` | None native; use external (gef-style for lldb) |
| Frontend | `gdb -tui`, `cgdb`, IDE plugins | LLDB DAP, Xcode, VS Code |
| Breakpoint syntax | terse (`b main`) | verbose (`breakpoint set -n main`) + alias |
| Variable format | `print/x var` | `expr -f x -- var` |
| GDB vs LLDB map | — | `lldb.llvm.org/use/map.html` |

When to prefer gdb: legacy Linux codebases, kernel debugging (kgdb), reverse execution (`rr`), embedded targets with only gdbserver. When to prefer lldb: macOS native development, Swift / Obj-C, Python-heavy debugging scripts, Clang/LLVM toolchain integration. Both have feature parity for general C/C++ stop-and-inspect workflows; the choice is mostly platform-driven.

## Idioms

Battle-tested patterns.

```bash
cat > ~/.lldbinit <<'EOF'
settings set target.x86-disassembly-flavor intel
settings set target.skip-prologue false
settings set target.inline-breakpoint-strategy always
settings set stop-disassembly-display always
settings set stop-disassembly-count 8
settings set stop-line-count-before 3
settings set stop-line-count-after 3
type summary add --summary-string "${var.size}" std::vector
command alias bff breakpoint set --file %1 --line %2
command alias bnn breakpoint set --name %1
command alias brr breakpoint set --regex %1
command alias xx memory read --size 8 --format x --count 16
command alias dis disassemble --frame
command regex bf 's/(.+):(.+)/breakpoint set --file "%1" --line %2/'
EOF
```

```bash
breakpoint set --regex "^Foo::.*"
breakpoint set --regex "^MyClass::(get|set)_.*"
breakpoint set --func-regex "test_.*"
breakpoint set --selector viewDidLoad
breakpoint set --regex "-\[.* dealloc\]"
```

```bash
disassemble --frame
disassemble --pc
disassemble --name main
disassemble --address 0x100003f50
disassemble --start-address 0x100003f50 --count 20
disassemble --line
dis -m
```

```bash
watchpoint set expression -- &my_struct.field
watchpoint set expression --size 4 -- (uint32_t*)&my_struct.field
watchpoint set expression --size 8 -- &my_array[5]
```

```bash
expression -- (void*)dlopen("libfoo.dylib", 2)
expression -- (void*)dlsym($1, "my_func")
expression -- ((void(*)())$2)()
```

```bash
breakpoint set --name foo --auto-continue 1
breakpoint command add 1
> p (uint64_t)$rdi
> bt 1
> DONE
```

```bash
settings set target.process.thread.step-avoid-regexp '^std::'
settings set target.process.thread.step-avoid-regexp '^(std::|boost::|absl::)'
settings show target.process.thread.step-avoid-regexp
```

```bash
target stop-hook add
> bt 3
> register read pc sp
> DONE
target stop-hook list
```

```bash
command script add -f myutils.timing time_func
command script add -f myutils.dumphex dh
command script list
```

## Tips

```bash
settings set target.x86-disassembly-flavor intel
disassemble --frame
```

```bash
settings set auto-confirm true
settings set use-color true
settings set prompt "(lldb) "
```

```bash
help breakpoint set
apropos memory
type lookup MyClass
```

```bash
lldb -o "b main" -o "run" -o "bt" -o "quit" ./myprog
lldb -s commands.lldb ./myprog
lldb --batch -o "run" -k "bt" -k "quit" ./myprog
```

```bash
script
>>> dir(lldb.frame)
>>> help(lldb.SBFrame)
>>> exit()
```

```bash
expression -- $foo = 42
expression -- $bar = (int*)malloc(16)
expression -- $bar[0] = 1
```

```bash
breakpoint set --skip-prologue 0 --name main
settings set target.skip-prologue false
```

```bash
process attach --name MyApp --waitfor
process attach --pid $(pgrep -n MyApp)
```

```bash
settings set target.detach-on-error false
settings set target.process.follow-fork-mode child
```

```bash
gui
^X^A
^Cgui
```

```bash
log enable lldb api
log enable lldb breakpoints
log enable lldb expr
log enable lldb step
log disable lldb api
```

```bash
settings set target.use-fast-stepping true
settings set target.process.optimization-warnings false
settings set show-progress false
```

```bash
target modules dump symtab
target modules dump line-table myfile.c
target modules dump sections
target modules lookup --type MyStruct
```

```bash
script lldb.process.GetSelectedThread().GetSelectedFrame().FindVariable("x").SetValueFromCString("42")
script lldb.target.BreakpointCreateByLocation("foo.c", 42)
script lldb.process.Continue()
```

## See Also

- gdb — GNU debugger; many lldb users come from it (translation table inside this sheet)
- delve — Go-aware debugger; better than lldb for goroutines/channels
- pdb — Python's interactive debugger
- perf — Linux profiler
- polyglot — language-comparison cheat sheet
- bash — shell scripting reference

## References

- LLDB tutorial — https://lldb.llvm.org/use/tutorial.html
- GDB to LLDB command map — https://lldb.llvm.org/use/map.html
- Python scripting tutorial — https://lldb.llvm.org/use/python.html
- LLDB Python API reference — https://lldb.llvm.org/python_api.html
- LLDB variable formatting — https://lldb.llvm.org/use/variable.html
- LLDB symbolication — https://lldb.llvm.org/use/symbolication.html
- LLDB remote debugging — https://lldb.llvm.org/use/remote.html
- LLDB QEMU testing — https://lldb.llvm.org/use/qemu-testing.html
- "Advanced Apple Debugging & Reverse Engineering" by raywenderlich.com / Kodeco
- "Debugging with LLDB" — Apple WWDC sessions (e.g., WWDC 2018 #412, WWDC 2019 #429, WWDC 2022 #110370)
- Apple Technical Q&A QA1361 — debugserver and entitlements
- Apple Technical Note TN2123 — CrashReporter
- LLVM source — https://github.com/llvm/llvm-project/tree/main/lldb
- LLDB man page — `man lldb`
- LLDB built-in help — `(lldb) help`
- macOS code signing requirements — Apple Developer documentation on hardened runtime
- System Integrity Protection — Apple Developer documentation
