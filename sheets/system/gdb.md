# gdb (GNU Debugger)

The canonical source-level debugger for C, C++, Rust, Go, Ada, Fortran, D, Objective-C, and any language emitting DWARF debug info — process attach, breakpoint, watchpoint, single-step, backtrace, memory examination, register manipulation, reverse-execution, remote-debug, core-file forensics, scripted automation.

## Setup

### Installation — Linux

```bash
sudo apt update
sudo apt install -y gdb gdb-doc
```

```bash
sudo dnf install -y gdb gdb-doc
```

```bash
sudo pacman -S gdb
```

```bash
sudo zypper install gdb
```

```bash
sudo apk add gdb
```

### Installation — macOS

```bash
brew install gdb
```

macOS gotcha: gdb is not signed by default. Without a code-signing certificate it cannot attach to processes, leading to `Unable to find Mach task port for process-id`. Apple's official lldb is the path of least resistance on Darwin; if you must use gdb, generate a self-signed certificate via Keychain Access (`Certificate Assistant -> Create a Certificate -> "gdb-cert" -> Code Signing -> Always Trust`), then:

```bash
codesign -fs gdb-cert "$(which gdb)"
```

```bash
sudo killall taskgated
```

You may also need `/etc/gdb.conf`:

```bash
echo "set startup-with-shell off" | sudo tee -a /etc/gdb.conf
```

### Verify Install

```bash
gdb --version
```

```bash
gdb --configuration
```

```bash
gdb -batch -ex 'show version'
```

### Build with Debug Info

```bash
gcc -O0 -g3 -ggdb -o prog prog.c
```

```bash
g++ -O0 -g3 -ggdb -std=c++20 -o prog prog.cc
```

```bash
clang -O0 -g3 -gdwarf-4 -o prog prog.c
```

`-O0` disables optimization (variables stay in scope). `-g3` includes preprocessor macros. `-ggdb` emits DWARF tuned for gdb. `-Og` is the official "optimize for debugging" flag — use it when `-O0` is too slow.

### Rust

```bash
RUSTFLAGS="-C debuginfo=2" cargo build
```

```bash
rust-gdb target/debug/myprog
```

### Go

```bash
go build -gcflags="all=-N -l" -o prog .
```

```bash
gdb -ex 'add-auto-load-safe-path /' ./prog
```

Use `delve` (`dlv`) for Go in practice; gdb's Go support is incomplete (goroutines visible only as OS threads).

## Starting

### Open a Binary

```bash
gdb prog
```

```bash
gdb ./prog
```

```bash
gdb -q prog
```

`-q` (quiet) skips the startup banner.

### Pass Arguments to the Program

```bash
gdb --args prog --flag value positional1 positional2
```

```bash
gdb -ex 'set args --flag value' prog
```

### Attach to a Running Process

```bash
gdb -p 12345
```

```bash
gdb prog 12345
```

```bash
gdb attach 12345
```

Linux gotcha: kernel ptrace_scope often blocks attaching to non-child processes. Either run gdb as root, or:

```bash
sudo sysctl -w kernel.yama.ptrace_scope=0
```

```bash
echo 0 | sudo tee /proc/sys/kernel/yama/ptrace_scope
```

### Open a Core Dump

```bash
gdb prog core
```

```bash
gdb prog core.12345
```

```bash
gdb -c core prog
```

```bash
coredumpctl debug
```

```bash
coredumpctl gdb 12345
```

### Run a Script Non-Interactively

```bash
gdb -batch -ex 'run' -ex 'bt' --args ./prog arg1
```

```bash
gdb -batch -x script.gdb prog
```

```bash
gdb -nx -batch -ex 'set pagination off' -ex 'bt full' prog core
```

`-nx` skips `.gdbinit`. `-batch` exits after running the commands.

### Connect to a Remote gdbserver

```bash
gdb -ex 'target remote 192.168.1.42:1234' prog
```

```bash
gdb -ex 'target extended-remote /dev/ttyUSB0' prog
```

### Useful Startup Flags

```bash
gdb -tui prog
```

```bash
gdb -batch -ex run -ex bt prog
```

```bash
gdb -ex 'break main' -ex run prog
```

```bash
gdb -nh prog
```

```bash
gdb -nx prog
```

```bash
gdb -d /path/to/source prog
```

```bash
gdb -ix init.gdb prog
```

```bash
gdb --return-child-result -batch ./prog
```

`-tui` enables textual UI. `-batch` runs commands and exits. `-nh` ignores `~/.gdbinit` only. `-nx` ignores both. `-d` adds source path. `-ix` reads an extra init file.

## Breakpoints

### Setting Basic Breakpoints

```bash
break main
```

```bash
b main
```

```bash
break compute
```

```bash
break Class::method
```

```bash
break Namespace::Class::method
```

```bash
break operator+
```

```bash
break src/main.c:42
```

```bash
break main.c:42
```

```bash
break 42
```

```bash
break +5
```

```bash
break -5
```

```bash
break *0x4006a0
```

```bash
break *main+24
```

`*ADDR` breaks at a raw address. `+N`/`-N` are relative to the current line. `*func+offset` is offset from a symbol — useful for breaking at a specific instruction inside a function.

### Conditional Breakpoints

```bash
break main.c:42 if i == 100
```

```bash
break compute if argc > 1 && strcmp(argv[1], "debug") == 0
```

```bash
break tree.c:88 if node && node->key == target
```

```bash
condition 3 i > 100
```

```bash
condition 3
```

`condition N expr` adds a condition to existing breakpoint #N. `condition N` (no expression) clears it.

### Temporary Breakpoints

```bash
tbreak main
```

```bash
tbreak main.c:42 if argc > 1
```

```bash
thbreak main
```

`tbreak` deletes itself after the first hit. `thbreak` is a hardware-assisted temporary breakpoint.

### Regex Breakpoints

```bash
rbreak ^process_
```

```bash
rbreak Class::
```

```bash
rbreak file.c:^helper_
```

`rbreak` matches every function whose mangled or demangled name matches the regex; useful for breaking on every method of a class.

### Hardware Breakpoints

```bash
hbreak main
```

```bash
hbreak *0x4006a0
```

Hardware breakpoints are required when debugging ROM/flash; processors typically allow only 4 simultaneously.

### Watchpoints

```bash
watch x
```

```bash
watch arr[3]
```

```bash
watch *(int*)0x7ffe0010
```

```bash
rwatch x
```

```bash
awatch x
```

```bash
watch -location x
```

```bash
watch *p->next
```

`watch` breaks on writes; `rwatch` on reads; `awatch` on either. `-location` watches by address — survives scope exit.

### Listing, Deleting, Disabling

```bash
info breakpoints
```

```bash
info b
```

```bash
info watchpoints
```

```bash
delete
```

```bash
delete 3
```

```bash
delete 3 5 7
```

```bash
clear
```

```bash
clear main
```

```bash
disable 3
```

```bash
enable 3
```

```bash
enable once 3
```

```bash
enable count 5 3
```

```bash
disable display
```

`enable once N` is one-shot. `enable count N M` auto-disables breakpoint M after N hits.

### Ignore Counts

```bash
ignore 3 100
```

Skip the first 100 hits of breakpoint #3, then break on the 101st.

### Save/Load Breakpoints

```bash
save breakpoints bps.gdb
```

```bash
source bps.gdb
```

The saved file contains plain `break ...` commands and can be edited.

### Commands at Breakpoints

```bash
commands 3
silent
printf "called with i=%d\n", i
continue
end
```

`silent` suppresses the usual "Breakpoint 3, foo()..." banner — combined with `continue`, this is the canonical lightweight tracepoint trick. Use `commands` (no number) to attach to the most recent breakpoint.

### Breakpoint at Future-Loaded Symbol

```bash
break libfoo.so:foo
```

```bash
set breakpoint pending on
```

### Breaking at Process Entry

```bash
starti
```

`starti` breaks at the very first instruction of the executable, before `_start` runs the dynamic linker.

## Execution Control

### Run / Restart

```bash
run
```

```bash
r
```

```bash
run arg1 arg2 < input.txt > out.txt
```

```bash
start
```

```bash
starti
```

```bash
kill
```

```bash
set args --flag value
```

```bash
show args
```

`start` breaks at `main`, then runs. `starti` breaks at the first instruction (before `_start`). `kill` terminates the inferior but keeps the gdb session alive.

### Step / Next / Continue

```bash
step
```

```bash
s
```

```bash
step 5
```

```bash
next
```

```bash
n
```

```bash
stepi
```

```bash
si
```

```bash
nexti
```

```bash
ni
```

```bash
continue
```

```bash
c
```

```bash
continue 3
```

```bash
finish
```

```bash
until
```

```bash
until 88
```

```bash
advance compute
```

`step` enters function calls; `next` steps over them. `stepi`/`nexti` operate on machine instructions. `continue 3` ignores the next 2 hits and stops on the 3rd. `finish` runs until the current frame returns. `until` runs to the next source line in the current frame (skips loops). `advance LOC` runs until reaching a location.

### Skip — Don't Step Into

```bash
skip function std::vector
```

```bash
skip file boost/.*
```

```bash
info skip
```

```bash
skip enable 2
```

`skip` tells gdb to step over (rather than into) calls to matching functions/files — invaluable for stepping past STL internals.

### Force Return / Force Branch

```bash
return
```

```bash
return 42
```

```bash
jump 100
```

```bash
jump *0x4006a0
```

```bash
signal SIGINT
```

```bash
signal 0
```

`return` immediately leaves the current frame; `return EXPR` returns the given value. `jump` is dangerous — it bypasses cleanup and stack adjustment. Use rarely. `signal 0` continues without delivering a pending signal.

## Stack

### Backtrace

```bash
backtrace
```

```bash
bt
```

```bash
bt 10
```

```bash
bt -10
```

```bash
bt full
```

```bash
bt full 5
```

```bash
bt no-filters
```

```bash
bt hide
```

```bash
where
```

`bt N` shows the innermost N frames; `bt -N` shows the outermost N. `bt full` prints locals for each frame. `bt no-filters` bypasses Python frame filters (useful when a custom filter hides what you want to see).

### Navigate Frames

```bash
frame
```

```bash
frame 3
```

```bash
f 3
```

```bash
up
```

```bash
up 3
```

```bash
down
```

```bash
down 2
```

```bash
select-frame 3
```

```bash
select-frame view 0x7fff1234
```

`up` moves toward the caller; `down` toward the callee. `frame N` jumps to absolute frame N.

### Inspect a Frame

```bash
info frame
```

```bash
info args
```

```bash
info locals
```

```bash
info reg
```

```bash
info stack
```

### All Threads

```bash
thread apply all bt
```

```bash
thread apply all bt full
```

```bash
thread apply all -- print errno
```

```bash
thread apply 1-4 bt
```

The canonical "what is every thread doing" snapshot.

### Saved Frames

```bash
maintenance print frame-id
```

```bash
maintenance print msymbols
```

## Examining Memory

### print

```bash
print x
```

```bash
p x
```

```bash
p/x x
```

```bash
p/d x
```

```bash
p/u x
```

```bash
p/o x
```

```bash
p/t x
```

```bash
p/c x
```

```bash
p/f x
```

```bash
p/s x
```

```bash
p/a x
```

```bash
p (char*)x
```

```bash
p *p
```

```bash
p *p@10
```

```bash
p arr@5
```

```bash
p sizeof(*p)
```

```bash
p &x
```

```bash
p {int}0x7fff1234
```

```bash
p {struct foo}0x7fff1234
```

`/x` hex, `/d` decimal, `/u` unsigned, `/o` octal, `/t` binary, `/c` char, `/f` float, `/s` string, `/a` address. `*p@N` treats `p` as a pointer to an N-element array. `arr@N` treats `arr` as an N-element array. `{TYPE}ADDR` interprets the address as a value of TYPE.

### x — Examine Memory

`x/[count][format][unit] address`

| Field   | Values                                                                            |
|---------|-----------------------------------------------------------------------------------|
| count   | how many units                                                                    |
| format  | `x` hex, `d` dec, `u` unsigned, `o` octal, `t` binary, `a` address, `c` char, `f` float, `s` string, `i` instruction |
| unit    | `b` 1 byte, `h` 2 bytes, `w` 4 bytes, `g` 8 bytes                                  |

```bash
x/16xb 0x7fff1234
```

```bash
x/8wx &arr
```

```bash
x/4gx $rsp
```

```bash
x/s 0x4006a0
```

```bash
x/10i $pc
```

```bash
x/100c buffer
```

```bash
x/64xb &header
```

```bash
x/wx &x
```

```bash
x $rsp
```

```bash
x
```

`/i` is shorthand for "instruction." gdb remembers the format and address — pressing Enter (or typing `x`) repeats the last `x` walking forward.

### Display — Auto-Print at Each Stop

```bash
display x
```

```bash
display/x reg
```

```bash
display/i $pc
```

```bash
info display
```

```bash
undisplay 2
```

```bash
disable display 2
```

```bash
enable display 2
```

`display/i $pc` is the canonical "show me the next instruction every time gdb stops" idiom.

### Dump / Restore Binary

```bash
dump binary memory dump.bin 0x7ffe0000 0x7ffe1000
```

```bash
dump binary value foo.bin foo
```

```bash
restore dump.bin binary 0x7ffe0000
```

```bash
append binary memory more.bin 0x4000 0x5000
```

### printf

```bash
printf "i=%d s=%s\n", i, s
```

```bash
printf "0x%08x\n", *(unsigned int*)addr
```

## Registers

### Inspect

```bash
info registers
```

```bash
info reg
```

```bash
info reg rax rbx rcx
```

```bash
info reg rflags
```

```bash
info reg all
```

```bash
info reg float
```

```bash
info reg sse
```

```bash
info reg system
```

```bash
print $rax
```

```bash
p/x $rsp
```

```bash
p $pc
```

```bash
p/t $eflags
```

### Modify

```bash
set $rax = 0
```

```bash
set $rip = 0x4006a0
```

```bash
set $eflags |= (1 << 0)
```

```bash
set $rflags &= ~(1 << 6)
```

```bash
set $rdi = (uint64_t)"hello"
```

### Common ABI Registers (x86-64 SysV)

| Register       | Role                              |
|----------------|-----------------------------------|
| `$rdi $rsi $rdx $rcx $r8 $r9` | first 6 int/ptr args  |
| `$rax`         | return value                      |
| `$rsp`         | stack pointer                     |
| `$rbp`         | frame pointer                     |
| `$rip`         | instruction pointer               |
| `$xmm0`        | first float arg / return          |
| `$rflags`      | status flags                      |

### ARM64

```bash
info reg
```

```bash
p $x0
```

```bash
p $sp
```

ARM64 passes args in `x0..x7`, returns in `x0`, has stack pointer `sp` and program counter `pc`.

## Threads

### Listing

```bash
info threads
```

```bash
info thread 3
```

```bash
thread name worker-pool
```

```bash
thread find regex
```

`thread name NAME` names the current thread. `thread find` searches by regex.

### Switching

```bash
thread 3
```

```bash
thread 3.2
```

`3.2` means inferior 3, thread 2 — handy in multi-process sessions.

### Apply to Many Threads

```bash
thread apply all bt
```

```bash
thread apply all bt full
```

```bash
thread apply 1-4 print errno
```

```bash
thread apply all -ascending bt
```

```bash
thread apply all -- frame 0
```

### Scheduler Locking

```bash
set scheduler-locking on
```

```bash
set scheduler-locking step
```

```bash
set scheduler-locking off
```

```bash
show scheduler-locking
```

`step` is the practical default: stepping doesn't accidentally let other threads make progress and trigger the bug elsewhere.

### Non-Stop Mode

```bash
set non-stop on
```

```bash
interrupt -a
```

```bash
interrupt &
```

Non-stop mode lets one thread be paused while others continue running — essential for debugging GUIs and event loops.

## Watchpoints

### Variants

```bash
watch x
```

```bash
rwatch x
```

```bash
awatch x
```

```bash
watch x if x > 100
```

```bash
watch *(int*)0x7fff1234
```

```bash
watch -location p->next
```

### Limitations

Hardware watchpoints rely on the CPU's debug registers (DR0–DR3 on x86). You usually get 4 watchpoints, each watching 1, 2, 4, or 8 bytes. Beyond that, gdb falls back to software watchpoints — single-stepping the entire program — which is unusably slow for non-trivial code.

```bash
show can-use-hw-watchpoints
```

```bash
set can-use-hw-watchpoints 0
```

### Diagnose "Watchpoint No Longer in Scope"

```bash
watch -location x
```

```bash
watch *(int*)&x
```

By-address watchpoints survive scope exit; the variable name binding does not.

## Expressions

### Any C/C++ Expression

```bash
p i + 1
```

```bash
p arr[3] * 2
```

```bash
p strlen(s)
```

```bash
p strcmp(a, b)
```

```bash
p sizeof(struct foo)
```

```bash
p (int[5]){1,2,3,4,5}
```

```bash
p ((MyClass*)p)->method()
```

```bash
p obj.field.subfield
```

```bash
p &obj.field
```

### call — Invoke a Function

```bash
call printf("hello\n")
```

```bash
call malloc(64)
```

```bash
call dump_state()
```

```bash
p (void)dump_state()
```

The function actually runs in the inferior — side effects persist. Useful for invoking dump/log helpers compiled into the binary. `(void)EXPR` discards the return value silently.

### ptype / whatis

```bash
ptype x
```

```bash
ptype struct foo
```

```bash
ptype/o struct foo
```

```bash
ptype/m Class
```

```bash
ptype/r struct foo
```

```bash
whatis x
```

```bash
whatis &x
```

```bash
info types
```

```bash
info types ^my_
```

```bash
info functions
```

```bash
info functions process_
```

```bash
info variables
```

```bash
info variables ^g_
```

`ptype/o` shows offsets and sizes. `ptype/m` prints methods. `ptype/r` is raw, no typedef substitution. `whatis` gives just the type without expanding.

### Convenience Variables

```bash
set $i = 0
```

```bash
print $i
```

```bash
set $p = (int*)malloc(64)
```

```bash
print *$p@16
```

```bash
show convenience
```

`$_` holds the last `print` value. `$__` is a side history. `$_exitcode`, `$_siginfo`, `$_thread`, `$_inferior` are auto-set by gdb on relevant events.

### Casting

```bash
p *(struct foo*)0x7fff1234
```

```bash
p (char*)$rdi
```

```bash
p (void(*)(int))0x4006a0
```

```bash
p (typeof(*p))*p
```

### Loops in Expressions

```bash
set $i = 0
while $i < 10
printf "arr[%d] = %d\n", $i, arr[$i]
set $i = $i + 1
end
```

## Source

### list

```bash
list
```

```bash
l
```

```bash
list 50
```

```bash
list main
```

```bash
list main.c:1
```

```bash
list 10,30
```

```bash
list ,30
```

```bash
list 30,
```

```bash
list -
```

```bash
set listsize 30
```

```bash
show listsize
```

`list -` lists 10 lines before the previous `list`. `set listsize 0` shows the entire function.

### Source Path

```bash
directory /home/me/src
```

```bash
directory /home/me/src:/usr/src/lib
```

```bash
show directories
```

```bash
set substitute-path /build/abc /home/me/src
```

```bash
show substitute-path
```

```bash
info source
```

```bash
info sources
```

`set substitute-path` is the canonical fix for "cores from CI machines on different paths" — translate the build server's `/build/foo` to your local checkout.

### info source

```bash
info source
```

Prints the current file's compile flags, language, and debug format.

## Search

```bash
search printf
```

```bash
reverse-search compute
```

```bash
forward-search FOO
```

`search` and `reverse-search` walk the current source file matching a regex.

## Find

`find addr_low, addr_high, value [, value]...`

```bash
find 0x7fff0000, 0x7fff1000, 0xdeadbeef
```

```bash
find 0x7fff0000, +0x1000, "hello"
```

```bash
find/w 0x7fff0000, 0x7fff1000, 0x12345678
```

```bash
find/h 0x7fff0000, +0x1000, 0x1234
```

```bash
find/g $rsp, $rsp+4096, (long)0xcafebabe
```

```bash
find/1 $rsp, +4096, 0xdeadbeef
```

`/b` byte, `/h` halfword, `/w` word, `/g` giant (8 bytes). `/N` limits to N matches. Returns a list of addresses; `$_` holds the count of matches. Useful for hunting magic numbers, structure markers, or known strings.

## Pretty Printers

### Built-in libstdc++ printers

```bash
p vec
```

```bash
p mp
```

```bash
p s
```

```bash
p sp
```

```bash
info pretty-printer
```

```bash
disable pretty-printer global libstdc++.*
```

```bash
enable pretty-printer global libstdc++.*
```

The libstdc++ pretty printers ship with gcc as `python/libstdcxx/v6/printers.py`. They auto-load when the `.so` matches; otherwise:

```bash
python sys.path.insert(0, '/usr/share/gcc-13/python')
```

```bash
python from libstdcxx.v6.printers import register_libstdcxx_printers
```

```bash
python register_libstdcxx_printers(None)
```

### Toggling

```bash
p/r vec
```

```bash
set print pretty on
```

```bash
set print pretty off
```

```bash
set print elements 0
```

```bash
set print elements 200
```

```bash
set print object on
```

```bash
set print vtbl on
```

`/r` prints raw, bypassing the pretty printer.

## Python Scripting

### Inline Python

```bash
python print("hello from gdb python")
```

```bash
python
i = 0
while i < 10:
    gdb.execute("print arr[%d]" % i)
    i += 1
end
```

### Source a .py File

```bash
source script.py
```

```bash
python exec(open("script.py").read())
```

### Useful gdb Module APIs

```bash
python import gdb; gdb.execute("bt")
```

```bash
python import gdb; out = gdb.execute("print x", to_string=True); print(out)
```

```bash
python import gdb; v = gdb.parse_and_eval("x + 1"); print(int(v))
```

```bash
python import gdb; gdb.write("hello\n"); gdb.flush()
```

```bash
python import gdb; print(gdb.selected_inferior())
```

```bash
python import gdb; print(gdb.selected_thread())
```

```bash
python import gdb; print(gdb.selected_frame())
```

```bash
python import gdb; print(gdb.lookup_symbol("main"))
```

`gdb.execute(cmd, to_string=True)` captures output. `gdb.parse_and_eval(EXPR)` returns a `gdb.Value` you can convert with `int()`, `str()`, `.address`, `.type`, etc.

### Custom Command

```bash
cat > hello.py <<'PY'
import gdb
class HelloCmd(gdb.Command):
    """Print a friendly hello."""
    def __init__(self):
        super().__init__("hello", gdb.COMMAND_USER)
    def invoke(self, arg, from_tty):
        gdb.write("hello %s\n" % arg)
HelloCmd()
PY
```

```bash
source hello.py
```

```bash
hello world
```

### Custom Pretty Printer

```bash
cat > intpair_printer.py <<'PY'
import gdb
class IntPairPrinter:
    def __init__(self, val):
        self.val = val
    def to_string(self):
        return "(%d, %d)" % (int(self.val['a']), int(self.val['b']))
def lookup(val):
    if str(val.type.tag) == "IntPair":
        return IntPairPrinter(val)
    return None
gdb.pretty_printers.append(lookup)
PY
```

```bash
source intpair_printer.py
```

### Walk a Linked List in Python

```bash
python
head = gdb.parse_and_eval("head")
node = head
i = 0
while int(node) != 0 and i < 1000:
    print(int(node['key']))
    node = node['next']
    i += 1
end
```

### Frame Decorators / Filters

The `gdb.FrameFilter` and `gdb.FrameDecorator` APIs drive `bt` output — used by `gdb-dashboard`, `pwndbg`, `gef`, and language-specific tools (e.g. `rust-gdb`'s pretty backtraces).

## Catchpoints

### Syscalls

```bash
catch syscall
```

```bash
catch syscall open
```

```bash
catch syscall openat
```

```bash
catch syscall read write
```

```bash
catch syscall 1
```

When triggered, `$_thread`, `$_siginfo`, and the relevant register set are valid.

### C++ Exceptions

```bash
catch throw
```

```bash
catch throw std::out_of_range
```

```bash
catch rethrow
```

```bash
catch catch
```

```bash
catch exception
```

```bash
catch assert
```

`catch throw` breaks at the throw site; `catch catch` breaks at the catch handler.

### Signals

```bash
catch signal
```

```bash
catch signal SIGSEGV SIGFPE
```

```bash
handle SIGUSR1 nostop noprint pass
```

```bash
handle SIGSEGV stop print nopass
```

```bash
info handle
```

`handle SIG ... pass` forwards the signal to the inferior. `nopass` discards it.

### Process Lifecycle

```bash
catch fork
```

```bash
catch vfork
```

```bash
catch exec
```

### Library Load/Unload

```bash
catch load
```

```bash
catch load libssl
```

```bash
catch unload
```

```bash
catch unload libssl
```

## Reverse Debugging

```bash
record
```

```bash
record full
```

```bash
record btrace
```

```bash
record btrace bts
```

```bash
record stop
```

```bash
record save trace.gdb
```

```bash
record restore trace.gdb
```

`record full` is software-emulated and ~1000× slower than native execution. `record btrace` uses Intel Processor Trace if available — much faster but only records branch decisions, not full state.

### Reverse Step / Continue

```bash
reverse-step
```

```bash
rs
```

```bash
reverse-next
```

```bash
rn
```

```bash
reverse-stepi
```

```bash
reverse-nexti
```

```bash
reverse-continue
```

```bash
rc
```

```bash
reverse-finish
```

`reverse-finish` runs back to the caller — the inverse of `finish`.

### Look Through History

```bash
record instruction-history
```

```bash
record instruction-history /m
```

```bash
record function-call-history
```

```bash
info record
```

```bash
set record full insn-number-max 200000
```

`/m` prints mixed source and assembly.

### rr (Mozilla)

For non-toy reverse debugging, `rr` from Mozilla wins. It records once, replays deterministically, and integrates with gdb's reverse commands without `record`'s 1000× slowdown:

```bash
sudo apt install rr
```

```bash
rr record ./prog
```

```bash
rr replay
```

```bash
reverse-continue
```

`rr` requires Intel CPUs with PMU (or a kernel module); doesn't work on AMD pre-Zen 3 or virtualized CPUs without PMU passthrough.

## Multi-Process

### Follow fork / exec

```bash
set follow-fork-mode child
```

```bash
set follow-fork-mode parent
```

```bash
set detach-on-fork off
```

```bash
set follow-exec-mode new
```

```bash
set follow-exec-mode same
```

`follow-fork-mode` defaults to `parent`. Set to `child` to debug forked child processes (essential for daemons). `detach-on-fork off` keeps both as inferiors so you can switch.

### Inferiors

```bash
info inferiors
```

```bash
inferior 2
```

```bash
add-inferior
```

```bash
add-inferior -copies 3
```

```bash
clone-inferior
```

```bash
remove-inferiors 2
```

```bash
detach inferior 2
```

### Switching Between Children

```bash
info inferiors
```

```bash
inferior 2
```

The output of `info inferiors` shows pid, executable, and connection per inferior; `*` marks the current one.

### gdbserver Multi-Process

```bash
gdbserver --multi :1234
```

```bash
gdb -ex 'target extended-remote :1234' -ex 'set remote exec-file /usr/bin/myprog' -ex run
```

## Core Files

### Enabling Core Dumps

```bash
ulimit -c unlimited
```

```bash
ulimit -c
```

```bash
echo "* soft core unlimited" | sudo tee -a /etc/security/limits.conf
```

```bash
cat /proc/sys/kernel/core_pattern
```

```bash
echo "/var/cores/core.%e.%p.%t" | sudo tee /proc/sys/kernel/core_pattern
```

```bash
sudo sysctl -w kernel.core_pattern='/var/cores/core.%e.%p.%t'
```

```bash
sudo mkdir -p /var/cores
```

```bash
sudo chmod 1777 /var/cores
```

`%e` is exe name, `%p` pid, `%t` time, `%h` hostname, `%s` signal.

### systemd-coredump

```bash
coredumpctl list
```

```bash
coredumpctl info
```

```bash
coredumpctl info 12345
```

```bash
coredumpctl gdb 12345
```

```bash
coredumpctl debug
```

```bash
coredumpctl dump 12345 -o core.12345
```

```bash
coredumpctl --since "10 minutes ago"
```

`coredumpctl debug` opens gdb on the most recent crash — fastest path from crash to backtrace on systemd boxes.

### Open the Core

```bash
gdb prog core
```

```bash
bt full
```

```bash
info threads
```

```bash
thread apply all bt
```

```bash
info registers
```

```bash
info sharedlibrary
```

```bash
x/i $pc
```

### Generate a Core From a Live Process

```bash
gcore 12345
```

```bash
generate-core-file
```

```bash
generate-core-file /tmp/snap.core
```

`gcore PID` (CLI tool) and `generate-core-file` (gdb command) produce ELF cores from a live process without killing it.

### macOS

macOS uses `~/Library/Logs/DiagnosticReports/*.crash` and `*.ips` files — not ELF cores. Open in Console.app or use `lldb`. gdb cannot read mach-o core files.

## Remote Debugging

### Server

```bash
gdbserver :1234 ./prog arg1 arg2
```

```bash
gdbserver --once :1234 ./prog
```

```bash
gdbserver --attach :1234 12345
```

```bash
gdbserver --multi :1234
```

```bash
gdbserver localhost:1234 ./prog
```

```bash
gdbserver /dev/ttyUSB0 ./prog
```

### Client

```bash
target remote 192.168.1.42:1234
```

```bash
target extended-remote 192.168.1.42:1234
```

```bash
target remote /dev/ttyUSB0
```

```bash
target remote | ssh host gdbserver - prog
```

```bash
disconnect
```

```bash
detach
```

`extended-remote` lets you `run` the program multiple times; `remote` is one-shot.

### Cross Architectures

```bash
sudo apt install gdb-multiarch
```

```bash
set architecture arm
```

```bash
show architecture
```

```bash
set sysroot /path/to/target/rootfs
```

```bash
show sysroot
```

```bash
set solib-search-path /path/to/target/rootfs/lib
```

```bash
file /path/to/cross-built-binary
```

```bash
target remote 10.0.0.42:1234
```

### File Transfer to Target

```bash
remote put localprog /tmp/prog
```

```bash
remote get /tmp/log /tmp/log.local
```

```bash
remote delete /tmp/prog
```

### qemu-user

```bash
qemu-arm -L /usr/arm-linux-gnueabi -g 1234 ./prog
```

```bash
gdb-multiarch ./prog
```

```bash
target remote :1234
```

## TUI Mode

### Enter / Exit

```bash
gdb -tui prog
```

```bash
tui enable
```

```bash
tui disable
```

| Keys      | Action                                  |
|-----------|-----------------------------------------|
| `Ctrl-x a`| toggle TUI                              |
| `Ctrl-x 1`| 1-window layout (source)                |
| `Ctrl-x 2`| 2-window layout (cycles src/asm/regs)   |
| `Ctrl-x s`| SingleKey mode                          |
| `Ctrl-l`  | refresh                                 |
| `Ctrl-p / Ctrl-n` | command history                |
| `PgUp / PgDn`     | scroll source                  |
| `Up / Down`       | move cursor in source          |

### Layouts

```bash
layout src
```

```bash
layout asm
```

```bash
layout split
```

```bash
layout regs
```

```bash
layout next
```

```bash
layout prev
```

`layout split` shows source + assembly side by side. `layout regs` adds the register window above whatever else is showing.

### Focus

```bash
focus cmd
```

```bash
focus src
```

```bash
focus asm
```

```bash
focus regs
```

```bash
focus next
```

```bash
focus prev
```

### Colours

```bash
set tui border-mode bold-standout
```

```bash
set tui active-border-mode bold
```

```bash
set style enabled on
```

```bash
set style address foreground cyan
```

```bash
set style function foreground green
```

```bash
set style filename foreground yellow
```

```bash
show style
```

## .gdbinit

`gdb` reads `~/.gdbinit` (and, if `set auto-load local-gdbinit on`, the `.gdbinit` in the current directory). The canonical productivity boilerplate:

```bash
set history save on
set history filename ~/.gdb_history
set history size 10000
set history expansion on

set print pretty on
set print object on
set print static-members off
set print vtbl on
set print array on
set print array-indexes on
set print elements 0
set print null-stop on
set print thread-events off

set pagination off
set confirm off
set verbose off
set logging file ~/.gdb.log
set logging overwrite on

set disassembly-flavor intel
set disable-randomization on
set debuginfod enabled on

set auto-load safe-path /
add-auto-load-safe-path /
set follow-fork-mode child
set detach-on-fork off

define hookpost-run
end
define eb
  enable breakpoints
end
define db
  disable breakpoints
end
```

### Project-local .gdbinit

```bash
directory ./src
file ./build/debug/prog
break main
break src/parser.c:88 if token_kind == TOK_IDENT
```

If you see "warning: File ... auto-loading has been declined", add the path:

```bash
add-auto-load-safe-path /home/me/proj/.gdbinit
```

Or in `~/.gdbinit`:

```bash
add-auto-load-safe-path /home/me/proj
```

### Hook Functions

```bash
define hook-stop
  printf "[stopped, pc=%p]\n", $pc
end
define hookpost-run
  echo running...
end
```

`hook-CMD` runs before CMD; `hookpost-CMD` runs after. Useful for keeping state synced (e.g. dump globals at every stop).

## set Options

### Pretty Printing

```bash
set print pretty on
```

```bash
set print object on
```

```bash
set print elements 0
```

```bash
set print elements 200
```

```bash
set print array on
```

```bash
set print array-indexes on
```

```bash
set print null-stop on
```

```bash
set print symbol on
```

```bash
set print address on
```

```bash
set print pid on
```

```bash
set print sevenbit-strings off
```

```bash
set print union on
```

```bash
set print frame-arguments scalars
```

```bash
set print frame-info short-location
```

```bash
set print thread-events on
```

### Process Control

```bash
set follow-fork-mode child
```

```bash
set detach-on-fork off
```

```bash
set follow-exec-mode new
```

```bash
set disable-randomization on
```

```bash
set startup-with-shell off
```

```bash
set environment LD_PRELOAD=/path/to/lib.so
```

```bash
unset environment LD_PRELOAD
```

```bash
show environment
```

### UI

```bash
set pagination off
```

```bash
set confirm off
```

```bash
set verbose off
```

```bash
set complaints 0
```

```bash
set width 0
```

```bash
set height 0
```

```bash
set prompt (myprog)
```

```bash
set output-radix 16
```

```bash
set input-radix 16
```

`set complaints 0` silences DWARF complaints — useful when working with old or unusual debug info.

### Logging

```bash
set logging file gdb.log
```

```bash
set logging overwrite on
```

```bash
set logging redirect on
```

```bash
set logging on
```

```bash
set logging off
```

```bash
show logging
```

### Symbols / Debuginfo

```bash
set debug-file-directory /usr/lib/debug
```

```bash
show debug-file-directory
```

```bash
set debuginfod enabled on
```

```bash
set debuginfod urls https://debuginfod.fedoraproject.org/
```

```bash
show debuginfod urls
```

## Symbols and Debug Info

### Compile Flags

```bash
gcc -O0 -g3 -ggdb prog.c -o prog
```

```bash
gcc -Og -g3 prog.c -o prog
```

```bash
gcc -O0 -g3 -gdwarf-5 prog.c -o prog
```

```bash
gcc -O0 -g3 -gsplit-dwarf prog.c -o prog
```

`-g3` adds macro debug info (so `print MACRO_NAME` works in gdb). `-gsplit-dwarf` keeps debug info in side files for faster linking.

### Strip vs Debug

```bash
strip --strip-debug prog
```

```bash
strip prog
```

```bash
file prog
```

A stripped binary leaves you with only mangled-name symbols. Recompile with `-g`. If you must debug a stripped production binary, find the corresponding debug package or build-id'd companion file.

### Separate Debug Info

```bash
objcopy --only-keep-debug prog prog.debug
```

```bash
strip --strip-unneeded prog
```

```bash
objcopy --add-gnu-debuglink=prog.debug prog
```

```bash
readelf -n prog | grep -A2 'GNU.build-id'
```

gdb finds separated debug info via:

1. The path embedded by `--add-gnu-debuglink`
2. `/usr/lib/debug/<bin path>.debug`
3. `/usr/lib/debug/.build-id/AB/CDEF....debug`

```bash
info sharedlibrary
```

```bash
info files
```

```bash
maint print msymbols
```

### debuginfod (Fedora, Ubuntu, Debian)

`debuginfod` is the modern centralized "fetch debug info on demand" service:

```bash
export DEBUGINFOD_URLS="https://debuginfod.elfutils.org/"
```

```bash
gdb prog
```

```bash
set debuginfod enabled on
```

The first `bt` against a system library transparently downloads the symbols for that exact build.

### Add Symbols Manually

```bash
symbol-file prog
```

```bash
symbol-file prog.debug
```

```bash
add-symbol-file libfoo.so 0x7f8000000000
```

```bash
info symbol 0x4006a0
```

```bash
info address main
```

### "No symbol table" Diagnosis

The canonical message:

```bash
No debugging symbols found in prog
```

means one of:

- compiled without `-g`
- `strip` ran post-link
- separate `.debug` file isn't on the search path
- `set debug-file-directory` is misconfigured
- the binary you launched is not the one whose source you're staring at

`info files` prints the actual binary path, sections, and entry point — verify it's the one you think it is.

## Inspecting C++

### Demangling

```bash
p _ZN3foo3barEv
```

```bash
maint demangle _ZN3foo3barEv
```

```bash
set print demangle on
```

```bash
set print asm-demangle on
```

```bash
c++filt _ZN3foo3barEv
```

`c++filt` is the standalone CLI demangler — useful in shell pipelines.

### Classes

```bash
ptype MyClass
```

```bash
ptype/m MyClass
```

```bash
ptype/o MyClass
```

```bash
info functions ^MyClass::
```

```bash
info variables ^MyClass::
```

### Virtual Dispatch / vtable

```bash
p obj
```

```bash
p *obj
```

```bash
set print object on
```

```bash
p *((Derived*)basePtr)
```

```bash
info vtbl obj
```

```bash
p obj->_vptr
```

```bash
x/8wx obj
```

```bash
x/8a *(void**)obj
```

`set print object on` makes gdb print the dynamic type rather than the static type — useful when polymorphism hides what an object actually is.

### Templates

```bash
p vec<int>::size
```

```bash
info functions ^std::vector<int>::
```

```bash
ptype std::vector<int>
```

```bash
rbreak ^std::vector<int>::
```

### Smart Pointers (libstdc++ pretty printer)

```bash
p sp
```

```bash
p *sp.get()
```

```bash
p *sp._M_ptr
```

`sp._M_ptr` accesses the raw pointer when bypassing the pretty printer — the field name is libstdc++-internal and may differ between versions.

### Lambdas

Lambdas appear as nameless `operator()` of a compiler-generated class. To break:

```bash
info functions ^.*operator\(\)
```

```bash
rbreak compute::\{lambda
```

## Common Recipes

### Crash Diagnosis

```bash
ulimit -c unlimited
```

```bash
./prog
```

```bash
gdb prog core
```

```bash
bt
```

```bash
bt full
```

```bash
frame 2
```

```bash
info args
```

```bash
info locals
```

```bash
p *suspectPtr
```

```bash
x/64xb suspectAddr
```

```bash
info threads
```

```bash
thread apply all bt
```

The canonical "run, crash, backtrace" recipe: enable cores, run, open the core, navigate the stack to the suspect frame, print arguments and locals, dump suspect bytes.

### Hang / Deadlock

```bash
gdb -p $(pgrep prog)
```

```bash
thread apply all bt
```

```bash
info threads
```

```bash
thread 4
```

```bash
up
```

```bash
p *mutex
```

```bash
p mutex->__data.__owner
```

```bash
thread find <owner-tid>
```

Look for `__lll_lock_wait`, `pthread_cond_wait`, `futex_wait`, `sched_yield`. The mutex's `__owner` field reveals which thread holds it.

### Use-After-Free

```bash
gcc -O0 -g3 -fsanitize=address prog.c -o prog
```

```bash
./prog
```

```bash
gdb prog
```

```bash
break __asan_report_load4
```

```bash
run
```

```bash
bt
```

Pair with rr for record-replay UAF reproduction:

```bash
rr record ./prog
```

```bash
rr replay
```

```bash
continue
```

```bash
reverse-stepi
```

### Hot Function Tracing (no recompile)

```bash
break compute
```

```bash
commands
silent
printf "compute(%d)\n", n
continue
end
```

```bash
run
```

The `silent`+`printf`+`continue`+`end` block turns a breakpoint into a printf-tracepoint without rebuilding.

### Spam Print Every Iteration

```bash
break main.c:42
```

```bash
commands
silent
printf "i=%d total=%lld\n", i, total
continue
end
```

### Walk a Linked List

```bash
set $n = head
```

```bash
while $n
printf "node @ %p key=%d\n", $n, $n->key
set $n = $n->next
end
```

### Conditional Print on Nth Hit

```bash
break parse
```

```bash
ignore 3 999
```

```bash
condition 3 token_kind == TOK_BAD
```

### Patch a Variable Live

```bash
set var debug_level = 3
```

```bash
set debug_level = 3
```

```bash
set {int}0x602010 = 42
```

```bash
call (void)reload_config()
```

`set var` is safer than bare `set` when the variable name might collide with a gdb option.

### Capture stdout/stderr Inside gdb

```bash
set logging file run.log
```

```bash
set logging on
```

```bash
run
```

```bash
set logging off
```

### Check errno on Linux

```bash
p errno
```

```bash
p (char*)strerror(errno)
```

```bash
p *((int*)__errno_location())
```

### Disassemble

```bash
disassemble
```

```bash
disassemble main
```

```bash
disassemble main, +50
```

```bash
disassemble 0x4006a0, 0x4006f0
```

```bash
disassemble /m main
```

```bash
disassemble /s main
```

```bash
disassemble /r main
```

```bash
set disassembly-flavor intel
```

```bash
set disassembly-flavor att
```

```bash
x/20i $pc
```

`/m` interleaves source lines (deprecated, use `/s`). `/s` is the modern source-interleaved form. `/r` shows raw bytes.

### Step Over Library Internals

```bash
skip file /usr/include/c++/.*
```

```bash
skip function std::__throw_bad_alloc
```

```bash
info skip
```

### Dump a Registry of Globals at Each Stop

```bash
define dumpglobs
printf "g_count=%d g_state=%s\n", g_count, g_state_names[g_state]
end
```

```bash
define hook-stop
dumpglobs
end
```

## Common Errors

### "No symbol table is loaded. Use the 'file' command."

Compile with `-g` (or `-g3 -ggdb`). For a binary that already shipped without symbols, use the matching `*-dbgsym`/`*-debuginfo` package or recover via `debuginfod`.

```bash
file ./prog
```

```bash
info files
```

### "Function 'X' not defined."

Either the name is wrong (typo, missing namespace, not yet linked), or it's in another translation unit not in the binary, or the symbol was eliminated by `-O2`/`LTO`.

```bash
info functions process_
```

```bash
rbreak ^X
```

### "Cannot find bounds of current function."

The instruction pointer landed in code without DWARF (stripped library, JIT, kernel). Either set `set backtrace past-main on`, attach the right debug file, or accept the partial backtrace.

### "warning: Source file is more recent than executable."

Your source has been edited since the binary was built. Line numbers may be off by one or more. Rebuild.

### "warning: GDB: Failed to set controlling terminal: Operation not permitted"

The inferior cannot grab a controlling terminal — common with `pty`s in IDEs. Workaround:

```bash
set startup-with-shell off
```

```bash
set tty /dev/pts/3
```

Or run gdb in a real terminal (not the IDE's embedded one).

### "Couldn't find equivalent PT_LOAD segment for ..."

The core file was generated by a different binary than the one you opened. Check `info files` against the binary's mtime, build-id, or version stamp:

```bash
readelf -n prog | grep build-id
```

```bash
file core
```

```bash
strings core | head -30
```

### "Remote 'g' packet reply is too long"

You're connected to a gdbserver of a different architecture (e.g. 64-bit gdb to 32-bit gdbserver). Use `gdb-multiarch` and `set architecture` correctly.

```bash
set architecture i386
```

```bash
target remote :1234
```

### "Could not load shared library symbols for ..."

Set `solib-search-path` to point at the cross-compiled rootfs lib directory:

```bash
set sysroot /opt/target-root
```

```bash
set solib-search-path /opt/target-root/usr/lib
```

### "ptrace: Operation not permitted."

Linux Yama LSM blocks attaching:

```bash
sudo sysctl -w kernel.yama.ptrace_scope=0
```

Or run gdb under sudo, or set the binary's caps:

```bash
sudo setcap cap_sys_ptrace=eip $(which gdb)
```

### "Mach kernel error 5: (os/kern) failure"

macOS without code signing. Sign gdb with a self-signed code-signing cert (see Setup).

### "Cannot insert breakpoint N: Cannot access memory at address 0x..."

Either the binary is PIE and you broke before relocation (use `starti`), or the address is in unmapped memory (text segment not loaded yet). Solutions: break by symbol, or use `start` instead of `run`.

### "warning: GDB can't find the start of the function at ..."

Stripped binary or you're in JIT-emitted code with no symbols. Provide debug info or break by raw address.

## Common Gotchas

### Stripped Binary

Bad:

```bash
gcc -O2 prog.c -o prog
strip prog
gdb prog
```

```bash
bt
# 0  0x0000000000400550 in ?? ()
```

Fixed:

```bash
gcc -O0 -g3 -ggdb prog.c -o prog
gdb prog
```

```bash
bt
# 0  compute (n=42) at prog.c:17
```

### PIE Address Drift

Bad:

```bash
gdb prog
```

```bash
break *0x4006a0
```

Address moves every run because of ASLR/PIE, so the breakpoint lands in random memory.

Fixed:

```bash
break compute
```

```bash
set disable-randomization on
```

Symbol-relative breakpoints survive ASLR. gdb defaults `disable-randomization` to on, but verify.

### Optimized-Out Variables

Bad:

```bash
gcc -O2 -g prog.c -o prog
gdb prog
```

```bash
p i
# $1 = <optimized out>
```

Fixed:

```bash
gcc -O0 -g3 prog.c -o prog
```

```bash
gcc -Og -g3 prog.c -o prog
```

`-Og` is the official "optimize for debugging" flag — fewer optimizations remove variables.

### macOS gdb Without Codesign

Bad:

```bash
brew install gdb
```

```bash
gdb ./prog
```

```bash
run
# Unable to find Mach task port for process-id 12345
```

Fixed:

```bash
codesign -fs gdb-cert "$(which gdb)"
```

(See Setup for creating the cert.) Or use `lldb`, which is the path of least resistance on macOS.

### Forgetting `set follow-fork-mode child`

Bad:

```bash
gdb -p $(pgrep server)
```

```bash
break handle_request
```

```bash
continue
```

The breakpoint never fires because each request is handled by a forked child.

Fixed:

```bash
set follow-fork-mode child
```

```bash
set detach-on-fork off
```

```bash
continue
```

Now gdb follows the child of the next fork.

### Wrong Source for Binary

Bad:

```bash
gdb prog
```

```bash
list main
# warning: Source file is more recent than executable.
```

Fixed:

```bash
make
```

```bash
gdb ./prog
```

### Auto-Load Refused

Bad:

```bash
gdb prog
# warning: File "/home/me/proj/.gdbinit" auto-loading has been declined
```

Fixed:

```bash
echo 'add-auto-load-safe-path /home/me/proj' >> ~/.gdbinit
```

### Calling a Function That Hangs

Bad:

```bash
call do_blocking_io()
```

gdb session locks up forever waiting for the inferior call to return.

Fixed:

```bash
set unwindonsignal on
```

```bash
set unwind-on-terminating-exception on
```

Now Ctrl-C aborts the inferior call back into gdb.

### Pretty Printer Eats Errors

Bad:

```bash
p mp
# std::map pretty printer crashes silently or hides corruption
```

Fixed:

```bash
p/r mp
```

```bash
set print frame-arguments none
```

`/r` raw mode bypasses pretty printers. `set print frame-arguments none` prevents the printer running during automatic backtrace.

### Symbol Mismatch on Cores

Bad:

```bash
gdb /usr/bin/myapp /var/cores/core.12345
# warning: Couldn't find equivalent PT_LOAD segment...
```

Fixed:

```bash
file /var/cores/core.12345
```

```bash
strings -a /var/cores/core.12345 | grep build-id
```

```bash
gdb /opt/release/v1.2.3/myapp /var/cores/core.12345
```

Match the exact binary that produced the core.

### Forgetting to Compile Tests with -g

Bad:

```bash
make tests
```

```bash
gdb ./tests
# (No debugging symbols found)
```

Fixed:

```bash
cmake -DCMAKE_BUILD_TYPE=Debug ..
```

```bash
make tests
```

Or:

```bash
cmake -DCMAKE_BUILD_TYPE=RelWithDebInfo ..
```

### "Single-Stepping" While Other Threads Run

Bad: stepping in thread A but the bug is triggered by thread B's racing modification, and you can't reproduce.

Fixed:

```bash
set scheduler-locking step
```

Now stepping freezes the other threads.

### Inferior Stops on Trivial Signals

Bad:

```bash
run
# Program received signal SIGUSR1, User defined signal 1.
```

Fixed:

```bash
handle SIGUSR1 nostop noprint pass
```

```bash
handle SIGCHLD nostop noprint pass
```

### Calling C++ Member Without `this`

Bad:

```bash
call compute()
# No symbol "compute" in current context.
```

Fixed:

```bash
call obj.compute()
```

```bash
call ((Foo*)0x7fff1234)->compute()
```

### Forgetting `set print pretty on`

Bad:

```bash
p config
# {host = 0x4006a0 "localhost", port = 8080, tls = false, retry = {count = 3, backoff = 1500}}
```

Fixed:

```bash
set print pretty on
```

```bash
p config
```

Output now multi-line with one field per line — much easier to scan.

## gdb-dashboard / pwndbg / gef

Modern UI overlays that turn raw gdb into a Bloomberg-terminal-grade reverse-engineering / debugging environment. All three are pure Python and respect existing breakpoints, scripts, and Python extensions.

### gdb-dashboard

Clean panels (source, asm, regs, stack, threads, breakpoints, expressions). Single-file install:

```bash
wget -P ~ https://git.io/.gdbinit
```

```bash
mv ~/.gdbinit ~/.gdbinit.dashboard
```

```bash
echo "source ~/.gdbinit.dashboard" >> ~/.gdbinit
```

```bash
dashboard
```

```bash
dashboard -layout source assembly registers stack
```

```bash
dashboard -enabled on
```

```bash
dashboard source -style height 20
```

### pwndbg

Designed for exploit development and binary CTFs. Heap inspection, ROP, vmmap, telescope:

```bash
git clone https://github.com/pwndbg/pwndbg ~/pwndbg
```

```bash
cd ~/pwndbg && ./setup.sh
```

```bash
checksec
```

```bash
vmmap
```

```bash
telescope $rsp 32
```

```bash
heap
```

```bash
bins
```

```bash
rop --grep 'pop rdi'
```

```bash
cyclic 200
```

```bash
cyclic -l 0x6161616a
```

```bash
ksymaddr __libc_start_main
```

### gef (GDB Enhanced Features)

Sister project with a different feature ramp; one-line install:

```bash
bash -c "$(curl -fsSL https://gef.blah.cat/sh)"
```

```bash
context
```

```bash
vmmap
```

```bash
checksec
```

```bash
heap chunks
```

```bash
heap bins
```

```bash
pattern create 200
```

```bash
pattern search $rsp
```

```bash
elf-info
```

### Coexistence

You can only run one of dashboard/pwndbg/gef at a time (they all hijack the prompt). Pick one per profile and switch via:

```bash
gdb -nx -x ~/.gdbinit.pwndbg
```

## Idioms

### The Catch-Segfault Breakpoint

```bash
catch signal SIGSEGV
```

```bash
run
```

```bash
bt full
```

Doesn't depend on a core file or `ulimit -c`.

### Loop-Print an Array

```bash
set $i = 0
while $i < 10
printf "arr[%d] = %d\n", $i, arr[$i]
set $i = $i + 1
end
```

The canonical "show me the whole array" idiom — works for arrays whose size is dynamic.

### Deadlock Diagnostic

```bash
gdb -p $(pgrep myserver)
```

```bash
thread apply all bt
```

Scan for: `__lll_lock_wait`, `pthread_cond_wait`, `futex_wait`, `sched_yield`. A thread blocked on a mutex is the consumer; the holder is found via `mutex->__data.__owner`.

### Print Every Call to a Hot Function

```bash
break compute
```

```bash
commands
silent
printf "compute(n=%d)\n", n
continue
end
```

```bash
run
```

### Invoke an In-Process Dump Helper

```bash
call dump_state()
```

```bash
call (void)debug_print_tree(root)
```

### Conditional Break on Map Key

```bash
break parse if strcmp(key, "deadbeef") == 0
```

### Break Inside Stdlib

```bash
break operator new
```

```bash
break std::vector<int>::push_back
```

```bash
rbreak ^std::__cxx11::basic_string<.*>::basic_string
```

### Find a String in Memory

```bash
find /b 0x7f0000000000, 0x7f0010000000, 'p','a','s','s','w','d'
```

```bash
find 0, -1, "DEADBEEF"
```

### Lightweight Tracepoint

```bash
trace foo
```

```bash
actions
collect $regs, x, y
end
```

```bash
tstart
```

```bash
continue
```

```bash
tstop
```

```bash
tfind 0
```

```bash
tdump
```

True tracepoints require `gdbserver` and are non-stop, but the breakpoint+`commands silent`+`continue` pattern is the practical equivalent for most uses.

### Re-exec From Inside

```bash
set args ARG_NEW
```

```bash
run
```

```bash
shell ./rebuild.sh && echo OK
```

```bash
file ./prog
```

```bash
run
```

Re-reads symbols after rebuild without quitting gdb.

### Save Whole Session State

```bash
save breakpoints session.bps
```

```bash
save tracepoints session.tps
```

```bash
save gdb-index .
```

`save gdb-index DIR` precomputes a `.gdb-index` for every binary in `DIR` — drastically speeds up the next gdb session on those binaries.

### Auto-Continue Through Spurious Stops

```bash
handle SIGPIPE nostop noprint pass
```

```bash
handle SIGUSR1 nostop noprint pass
```

```bash
handle SIG34 nostop noprint pass
```

### Quick "is this pointer valid"

```bash
info symbol p
```

```bash
info address p
```

```bash
p p
```

```bash
x/x p
```

```bash
maint info sections
```

```bash
info proc mappings
```

`info proc mappings` (Linux) prints `/proc/<pid>/maps` from inside gdb — tells you whether a pointer lives in heap, stack, .bss, mapped lib, or unmapped void.

### One-Liner Crash Capture in CI

```bash
gdb -batch -ex run -ex 'thread apply all bt full' --args ./prog testarg 2>&1 | tee crash.log
```

Wrap test commands in this and you always have a backtrace when they crash.

### One-Liner "What Is This Process Doing"

```bash
gdb --batch --pid=12345 --ex 'thread apply all bt'
```

Doesn't require attaching interactively — perfect for `kubectl exec` debugging.

### Enable debuginfod For This Session Only

```bash
DEBUGINFOD_URLS=https://debuginfod.elfutils.org/ gdb prog core
```

### Add a Watchpoint That Survives Optimizer Reorderings

```bash
watch -location *p
```

```bash
watch *(int*)&x
```

By-address watchpoints don't suffer from "variable optimized out" because they bind to memory, not to a DWARF variable description.

### Print a Buffer as Hex Dump

```bash
x/64xb buffer
```

```bash
x/64bx buffer
```

```bash
dump binary memory dump.bin buffer buffer+64
```

The `xxd` of gdb-land. `dump` lets you analyze the bytes outside gdb.

### Conditional Logging with Counter

```bash
set $cnt = 0
break process
commands
silent
set $cnt = $cnt + 1
if $cnt % 1000 == 0
  printf "iter %d\n", $cnt
end
continue
end
```

Logs every 1000th call without flooding the terminal.

## Tips

- `Ctrl-X A` toggles TUI mode mid-session — quickest way to "see code while debugging."
- `Ctrl-X 2` then `Ctrl-X 2` repeatedly cycles src→asm→regs layouts.
- Use `start` instead of `run`; it auto-breaks at `main` so PIE relocation has happened and you can set address-based breakpoints.
- `tbreak` + `continue` is the cleanest way to skip many iterations and break exactly once.
- `commands silent ... continue end` is gdb's poor-man's tracepoint; faster than rebuilding with `printf`.
- `set print elements 0` once in `~/.gdbinit` saves you from "..." truncated arrays forever.
- `gdb -batch -ex run -ex bt prog` in CI captures crash stacks on every test failure.
- `coredumpctl gdb` is the fastest path from "process died 30 seconds ago" to a working backtrace on systemd boxes.
- `info sharedlibrary` then `set debuginfod enabled on` is the canonical "I'm missing libc symbols" fix on modern distros.
- `set scheduler-locking step` makes single-stepping in multithreaded code actually work.
- `set follow-fork-mode child` plus `set detach-on-fork off` is mandatory when debugging server processes.
- `gef`, `pwndbg`, or `gdb-dashboard` is mandatory in 2026 — vanilla gdb's UX hasn't changed since 1996.
- `rr` is the right answer for non-deterministic bugs; `record` is the wrong answer.
- `delve` (dlv) for Go, `lldb` for macOS / Swift, `pdb` for Python — gdb is best at C/C++/Rust on Linux.
- For embedded ARM, gdb + OpenOCD over `target remote :3333` is the canonical JTAG flow.
- Use `set disable-randomization off` only when you specifically want to repro ASLR-dependent bugs; gdb defaults to ASLR-off.
- `info proc mappings` (Linux) is `/proc/<pid>/maps` — use it to ID code/data/heap/stack regions when chasing bad pointers.
- `maint info breakpoints` shows internal-use breakpoints (longjmp, exception, etc.) — useful when wondering why a `step` lands somewhere weird.
- `gdb --batch --pid=12345 --ex 'thread apply all bt'` is a one-liner "what is this hung process doing right now."
- `set logging on` + replay-paste-into-IDE is faster than scrolling backwards in the gdb buffer.
- For very long sessions, `set history save on` and `set history size 10000` give you Bash-like history persistence across runs.
- `set confirm off` after you trust your fingers — saves dozens of `y<enter>` per session.
- `set print thread-events off` silences "[New Thread ...]" spam in heavily-threaded programs.
- `info auxv` shows AT_* auxiliary vector — useful for finding the loader, vDSO base, page size, hwcap.
- `maint print symbols /tmp/sym.txt` dumps every symbol; faster to grep than to repeat `info functions`.
- `maint set show-debug-regs on` makes hardware breakpoints/watchpoints visible — confirm slot allocation when you suspect "no more debug registers."
- `directory $cwd/include:$cwd/src` is a one-liner to add multiple source paths.
- gdb interprets `\n` and `\t` in `printf` format strings — use `\\n` to print a literal backslash-n.
- `set $a = (struct foo*)$rdi` makes the first arg accessible as a typed pointer for the rest of the frame.
- `bt -frame-arguments scalars` produces compact tracebacks ideal for sharing in chat.
- `print $_thread, $_inferior` reveals which inferior+thread you're stopped in.
- `info source` after attaching to a process tells you whether gdb actually found source for the current PC.
- `add-symbol-file ... -s .text 0x...` is needed for relocatable objects whose `.text` is at an offset.
- `set print finish on` shows the return value of `finish` — convenient when chasing return-value bugs.
- Use `define` macros for project-specific shortcuts — `define dump_req` then `commands` to call it from any breakpoint.
- The `gdb` Python module's `gdb.events.stop`, `gdb.events.exited`, `gdb.events.new_objfile` hooks let scripts react to inferior events.

## See Also

- `cs lldb` — LLVM's debugger (default on macOS); includes a gdb-vs-lldb command translation table
- `cs delve` — Go-aware debugger (goroutines, channels, runtime types)
- `cs pdb` — Python's interactive debugger
- `cs perf` — Linux profiler (cycle/cache/syscall sampling)
- `cs valgrind` — memcheck/helgrind/cachegrind dynamic analysis
- `cs polyglot` — language-comparison cheat sheet
- `cs bash` — shell scripting reference (gdb scripting via `set logging`/`shell`)

## References

- `man gdb`
- `info gdb` — full Texinfo manual locally
- https://sourceware.org/gdb/ — official GDB project home
- https://sourceware.org/gdb/current/onlinedocs/gdb/ — full HTML manual
- https://sourceware.org/gdb/wiki/ — wiki, FAQ, recipes
- "The Art of Debugging with GDB, DDD, and Eclipse" — Norman Matloff & Peter Jay Salzman, No Starch Press
- "GDB Pocket Reference" — Arnold Robbins, O'Reilly
- "Debugging with GDB" — Richard Stallman, Roland Pesch, Stan Shebs (FSF, free PDF)
- https://sourceware.org/elfutils/ — debuginfod, elf utilities
- https://github.com/cyrus-and/gdb-dashboard — modern panel UI
- https://github.com/pwndbg/pwndbg — exploit-dev plugin
- https://github.com/hugsy/gef — GDB Enhanced Features
- https://rr-project.org/ — record-and-replay reverse debugging
- https://www.gnu.org/software/gdb/ — GNU project page
- DWARF Debugging Standard: https://dwarfstd.org/
- https://www.kernel.org/doc/html/latest/dev-tools/gdb-kernel-debugging.html — kernel debugging
- https://lldb.llvm.org/use/map.html — lldb<->gdb command translation
