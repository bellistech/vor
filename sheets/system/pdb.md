# pdb (Python Debugger)

The stdlib Python debugger — interactive breakpoints, post-mortem inspection, stepping, and stack walking, all with zero install.

## Setup

pdb ships with the Python standard library. Nothing to install for basic use.

```bash
python --version
```

```bash
python3 --version
```

```bash
python -c "import pdb; print(pdb.__file__)"
```

```bash
python -c "import sys; print(sys.version_info)"
```

The `breakpoint()` builtin requires Python 3.7 or newer. For older Python use `import pdb; pdb.set_trace()`.

```bash
python -c "import sys; assert sys.version_info >= (3,7), 'breakpoint() needs 3.7+'"
```

Optional upgrades — install one for a richer experience:

```bash
pip install ipdb
```

```bash
pip install pdbpp
```

```bash
pip install debugpy
```

```bash
pip install web-pdb
```

```bash
pip install rpdb
```

Verify install:

```bash
python -c "import ipdb; print(ipdb.__version__)"
```

```bash
python -c "import pdb; print(pdb.__file__)"
```

When pdbpp is installed it shadows stdlib pdb (`import pdb` returns pdbpp). Verify with:

```bash
python -c "import pdb; print(pdb.__file__)"
```

If the path mentions `pdbpp`, pdb++ is active.

For project-local install (recommended over global):

```bash
python -m venv .venv
```

```bash
source .venv/bin/activate
```

```bash
pip install ipdb
```

## Starting

There are three canonical ways to start a pdb session.

### From the command line

Launch the script under pdb. Execution stops at the first line:

```bash
python -m pdb script.py
```

```bash
python -m pdb script.py arg1 arg2
```

Pdb prompt looks like:

```bash
(Pdb) 
```

### Inline with breakpoint() (Python 3.7+)

Drop a `breakpoint()` call where you want to stop:

```bash
python -c "x=1; breakpoint(); print(x)"
```

```bash
echo 'def main():
    x = 42
    breakpoint()
    print(x)
main()' > demo.py
```

```bash
python demo.py
```

Execution stops at the breakpoint() line and you get the (Pdb) prompt.

### Inline legacy (pre-3.7 or explicit pdb)

```bash
python -c "import pdb; x=1; pdb.set_trace(); print(x)"
```

This always uses stdlib pdb, regardless of `PYTHONBREAKPOINT`.

### Post-mortem on uncaught exception

`-c continue` lets the script run; on unhandled exception, control drops into pdb at the failing frame:

```bash
python -m pdb -c continue script.py
```

```bash
echo 'd = {}
print(d["missing"])' > crash.py
```

```bash
python -m pdb -c continue crash.py
```

You land in the (Pdb) prompt at the KeyError, with the full stack walkable.

### From inside the REPL

```bash
python -c "import pdb; pdb.run('x=1/0')"
```

```bash
python -c "
import pdb
def f(): return 1/0
try: f()
except: pdb.post_mortem()
"
```

## PYTHONBREAKPOINT — Customizing breakpoint()

`PYTHONBREAKPOINT` is the environment variable that picks which debugger `breakpoint()` invokes. The canonical "switch debugger without code change."

Default (uses pdb):

```bash
unset PYTHONBREAKPOINT
```

```bash
python demo.py
```

Switch to ipdb:

```bash
export PYTHONBREAKPOINT=ipdb.set_trace
```

```bash
python demo.py
```

Switch to pudb (curses TUI):

```bash
pip install pudb
```

```bash
export PYTHONBREAKPOINT=pudb.set_trace
```

Switch to web-pdb (browser):

```bash
export PYTHONBREAKPOINT=web_pdb.set_trace
```

Disable all `breakpoint()` calls (production safe):

```bash
export PYTHONBREAKPOINT=0
```

```bash
python demo.py
```

With `PYTHONBREAKPOINT=0` every `breakpoint()` is a no-op — safe to commit code with `breakpoint()` if your prod env sets this.

Per-invocation override:

```bash
PYTHONBREAKPOINT=ipdb.set_trace python demo.py
```

```bash
PYTHONBREAKPOINT=0 python demo.py
```

Custom hook function:

```bash
echo 'def my_hook():
    print("hook fired")
    import pdb; pdb.set_trace()' > my_hooks.py
```

```bash
PYTHONBREAKPOINT=my_hooks.my_hook python demo.py
```

`breakpoint()` accepts kwargs that pass through to the hook:

```bash
python -c "breakpoint(header='entering critical section')"
```

## Commands — Core

Every command has a single-letter shortcut. Help is always available.

```bash
(Pdb) h
```

```bash
(Pdb) help
```

```bash
(Pdb) help break
```

### List source

```bash
(Pdb) l
```

```bash
(Pdb) list
```

```bash
(Pdb) l 1, 50
```

```bash
(Pdb) l .
```

`l` (no args) lists 11 lines around the current line. `l .` re-centers on the current line. `l first, last` lists an explicit range.

### Long list (full function)

```bash
(Pdb) ll
```

```bash
(Pdb) longlist
```

`ll` prints the entire current function or frame body — usually what you want over `l`.

### Step into

```bash
(Pdb) s
```

```bash
(Pdb) step
```

Executes one line; if it's a function call, descends into it.

### Step over (next)

```bash
(Pdb) n
```

```bash
(Pdb) next
```

Executes the current line; treats function calls as a single step.

### Run until function returns

```bash
(Pdb) r
```

```bash
(Pdb) return
```

Executes until the current function returns; you stop on the return statement.

### Continue

```bash
(Pdb) c
```

```bash
(Pdb) cont
```

```bash
(Pdb) continue
```

Resume normal execution until the next breakpoint or program end.

### Until line

```bash
(Pdb) unt
```

```bash
(Pdb) until
```

```bash
(Pdb) until 50
```

`until` (no arg) runs until the next line greater than the current one — escapes a loop body. `until N` runs until line N.

### Quit

```bash
(Pdb) q
```

```bash
(Pdb) quit
```

`q` exits pdb AND the program — it raises `BdbQuit`.

### Restart

```bash
(Pdb) restart
```

```bash
(Pdb) run arg1 arg2
```

`restart` re-execs the script under the same pdb session, preserving breakpoints.

## Breakpoints

### Set a breakpoint

By line number in current file:

```bash
(Pdb) b 42
```

By file and line:

```bash
(Pdb) b script.py:42
```

By function name:

```bash
(Pdb) b mymodule.myfunc
```

```bash
(Pdb) b MyClass.method
```

### Conditional breakpoint

`break file:line, condition` — comma separates. The condition is any Python expression evaluated in the breakpoint's frame.

```bash
(Pdb) b worker.py:50, len(queue) > 100
```

```bash
(Pdb) b 42, x == "target"
```

```bash
(Pdb) b mymod.process, item.priority > 5
```

### List breakpoints

```bash
(Pdb) b
```

```bash
(Pdb) break
```

(no args) lists all breakpoints with numbers, hits, conditions.

### Clear breakpoint

```bash
(Pdb) cl 1
```

```bash
(Pdb) clear 1
```

```bash
(Pdb) cl
```

`cl N` clears breakpoint N. `cl` (no args, with confirmation) clears all.

### Disable / enable

```bash
(Pdb) disable 1
```

```bash
(Pdb) enable 1
```

Disabled breakpoints are kept but skipped — handy for toggling.

### Temporary breakpoint

```bash
(Pdb) tbreak 42
```

```bash
(Pdb) tbreak script.py:42
```

`tbreak` fires once and removes itself.

### Multi-line commands attached to breakpoint

`commands N` opens a sub-prompt; lines you type run when breakpoint N hits. End with `end`.

```bash
(Pdb) commands 1
(com) p locals()
(com) p self.state
(com) c
(com) end
```

The trailing `c` makes the breakpoint silently log + continue — a poor man's tracepoint.

### Ignore N hits

```bash
(Pdb) ignore 1 100
```

Skip the next 100 hits of breakpoint 1, then resume normal stopping. Counts down.

## Inspection

### Print

```bash
(Pdb) p x
```

```bash
(Pdb) p len(items)
```

```bash
(Pdb) p {k: v for k, v in d.items() if v}
```

`p` prints `repr(expr)`. Any Python expression works.

### Pretty-print

```bash
(Pdb) pp d
```

```bash
(Pdb) pp [x.__dict__ for x in xs]
```

`pp` uses `pprint.pformat` — useful for nested dicts and lists.

### Args of current function

```bash
(Pdb) a
```

```bash
(Pdb) args
```

Prints the arguments of the current frame.

### Stack trace (where)

```bash
(Pdb) w
```

```bash
(Pdb) where
```

```bash
(Pdb) bt
```

Shows the full call stack with `>` marking the current frame.

### Type / inspect

```bash
(Pdb) whatis x
```

```bash
(Pdb) whatis self.foo
```

Prints the type of the expression.

### Drop into Python REPL

```bash
(Pdb) interact
```

`interact` opens a real Python REPL with the current frame's locals — full tab-completion, multi-line, no pdb-command-name conflicts. Exit with Ctrl-D to return to (Pdb).

```bash
(Pdb) interact
>>> import asyncio
>>> loop = asyncio.get_event_loop()
>>> ^D
(Pdb)
```

Note: assignments inside `interact` mutate a copy of locals, NOT the running frame's locals (CPython limitation).

### Auto-display on every stop

```bash
(Pdb) display x
```

```bash
(Pdb) display len(queue)
```

```bash
(Pdb) display
```

`display expr` re-evaluates the expression on every prompt and shows it if changed. `display` (no args) lists tracked expressions.

```bash
(Pdb) undisplay
```

```bash
(Pdb) undisplay 1
```

## Stack

### Move up the stack

```bash
(Pdb) u
```

```bash
(Pdb) up
```

```bash
(Pdb) up 3
```

Moves toward the caller. After moving, `p`, `l`, `args` operate on the new frame.

### Move down the stack

```bash
(Pdb) d
```

```bash
(Pdb) down
```

```bash
(Pdb) down 2
```

Moves toward the callee.

### Show current stack with marker

```bash
(Pdb) w
```

The line with `>` is the current frame.

### Frame switching workflow

```bash
(Pdb) w
(Pdb) u
(Pdb) p locals()
(Pdb) u
(Pdb) ll
(Pdb) d
```

## Source

### Show numbered source

```bash
(Pdb) l
```

```bash
(Pdb) l 1, 100
```

`l` shows 11 lines centered on the current line (or the last listed line). Marker `->` shows current line; `B` marks breakpoints.

### Show full current function

```bash
(Pdb) ll
```

Almost always preferred over `l` for context.

### Show source for arbitrary frame

After `u`/`d` to switch frame, `ll` shows that frame's function.

```bash
(Pdb) u
(Pdb) ll
```

### Show source for a function by name

`ll` only works on the current frame. To inspect a function not yet called, use:

```bash
(Pdb) import inspect
(Pdb) p inspect.getsource(mymod.myfunc)
```

## Variables and Expressions

Any Python expression works at the (Pdb) prompt. The result is printed via `p`.

```bash
(Pdb) p x + y
```

```bash
(Pdb) p [i for i in range(10) if i % 2]
```

```bash
(Pdb) p {**a, **b}
```

### Assignment — use the `!` prefix

A bare `x = 5` looks like a comparison or an unknown command to pdb. Prefix with `!` to disambiguate as a Python statement:

```bash
(Pdb) !x = 5
```

```bash
(Pdb) !import json
```

```bash
(Pdb) !self.state = "fixed"
```

The `!` prefix tells pdb "treat the rest as Python, not as a pdb command."

### Calling functions

```bash
(Pdb) p f(42)
```

```bash
(Pdb) p list(gen)
```

Caveat: calling functions from pdb runs them in the live process — side effects are real.

### Full REPL access

For complex expressions or imports, drop into `interact`:

```bash
(Pdb) interact
>>> import json
>>> json.dumps(self.__dict__, default=str, indent=2)
>>> ^D
```

### Inspecting locals and globals

```bash
(Pdb) p locals()
```

```bash
(Pdb) p globals()
```

```bash
(Pdb) p sorted(locals().keys())
```

### Inspecting an object

```bash
(Pdb) p vars(self)
```

```bash
(Pdb) p dir(self)
```

```bash
(Pdb) p {k: getattr(self, k) for k in dir(self) if not k.startswith("_")}
```

## Source Path Mapping

Python is interpreted, so source paths are usually correct. But when sources have moved (e.g., installed-package debugging), use:

```bash
(Pdb) import sys
(Pdb) p sys.path
```

```bash
(Pdb) !sys.path.insert(0, "/path/to/sources")
```

For frozen modules and zipped eggs, force-extract by setting `PYTHONDONTWRITEBYTECODE`:

```bash
PYTHONDONTWRITEBYTECODE=1 python -m pdb script.py
```

For installed packages whose source is missing, use `pip install --editable`:

```bash
pip install -e ./mypackage
```

When stepping into a stdlib function, source comes from `$(python -c 'import sys; print(sys.prefix)')/lib/python3.X/`.

```bash
python -c "import sys; print(sys.prefix)"
```

## Post-Mortem Debugging

The single most useful pdb mode: drop into the debugger at the moment of the unhandled exception, with the full stack still alive.

### Auto post-mortem on script

```bash
python -m pdb -c continue script.py
```

`-c continue` runs `continue` as the first command. The script runs normally; on uncaught exception, pdb takes the prompt at the failing frame.

```bash
echo 'def f(d):
    return d["k"]
f({})' > pm.py
```

```bash
python -m pdb -c continue pm.py
```

You land at the line raising KeyError, full stack walkable with `u`/`d`.

### Manual post-mortem in REPL

After an exception in an interactive session:

```bash
python
>>> import pdb
>>> try:
...     1/0
... except:
...     pdb.post_mortem()
```

```bash
python -i script.py
>>> import pdb; pdb.pm()
```

`pdb.pm()` resumes the LAST exception. Run with `python -i` so you get a REPL on exit.

### post_mortem with explicit traceback

```bash
python -c "
import pdb, sys
try:
    1/0
except:
    pdb.post_mortem(sys.exc_info()[2])
"
```

### Drop into pdb on every uncaught exception (sitecustomize)

Edit `~/.pythonrc` or use a sitecustomize.py:

```bash
echo 'import sys
def excepthook(t, v, tb):
    import traceback, pdb
    traceback.print_exception(t, v, tb)
    pdb.post_mortem(tb)
sys.excepthook = excepthook' > ~/.pythonrc
```

```bash
export PYTHONSTARTUP=~/.pythonrc
```

### post_mortem inside pytest

```bash
pytest --pdb
```

```bash
pytest --pdb --pdbcls=IPython.terminal.debugger:Pdb
```

`pytest --pdb` drops into pdb at the first failure. `--pdbcls` swaps debugger.

## Async / asyncio

pdb works inside coroutines but switching frames across the event loop can be confusing — frames include the asyncio scheduler.

### Set a breakpoint inside a coroutine

```bash
echo 'import asyncio
async def task():
    await asyncio.sleep(0.1)
    breakpoint()
    return 42
asyncio.run(task())' > async_demo.py
```

```bash
python async_demo.py
```

Once stopped, `w` shows the coroutine frame plus asyncio internals; `u` walks past the awaitable.

### Inspect from inside the running loop

Use `interact` to access the live loop:

```bash
(Pdb) interact
>>> import asyncio
>>> loop = asyncio.get_running_loop()
>>> tasks = asyncio.all_tasks(loop)
>>> for t in tasks: print(t)
>>> ^D
```

### Caveat — await is not allowed at the (Pdb) prompt

You cannot type `await coro()` directly at the (Pdb) prompt. Workaround:

```bash
(Pdb) interact
>>> import asyncio
>>> result = asyncio.get_event_loop().run_until_complete(coro())
```

Or schedule a task and let it run on `c`:

```bash
(Pdb) !asyncio.ensure_future(coro())
(Pdb) c
```

### trio / anyio debuggers

trio has its own diagnostic tools:

```bash
pip install trio
```

```bash
python -c "import trio; print(trio.__version__)"
```

For trio, prefer `trio-monitor` or stop with `breakpoint()` inside the trio task — same semantics.

### aiomonitor — alternative for live introspection

```bash
pip install aiomonitor
```

```bash
python -c "
import aiomonitor, asyncio
async def main():
    with aiomonitor.start_monitor(loop=asyncio.get_event_loop()):
        await asyncio.sleep(60)
asyncio.run(main())
"
```

Connect with:

```bash
nc localhost 50101
```

## ipdb — IPython pdb

`ipdb` is the canonical "pdb is just better with IPython" upgrade. Tab-completion, syntax highlighting, magic commands.

### Install

```bash
pip install ipdb
```

### Use as drop-in

```bash
python -c "import ipdb; x=1; ipdb.set_trace(); print(x)"
```

```bash
echo 'import ipdb
def f():
    x = 42
    ipdb.set_trace()
    return x
f()' > ipdb_demo.py
```

```bash
python ipdb_demo.py
```

### Use via PYTHONBREAKPOINT

```bash
export PYTHONBREAKPOINT=ipdb.set_trace
```

```bash
python demo.py
```

Now every `breakpoint()` lands in ipdb instead of pdb. Same commands, plus IPython magics:

```bash
ipdb> %who
```

```bash
ipdb> %timeit f()
```

```bash
ipdb> %hist
```

### ipdb post-mortem

```bash
python -c "
import ipdb
def f(d): return d['k']
try: f({})
except: ipdb.post_mortem()
"
```

### ipdb command-line entry

```bash
python -m ipdb script.py
```

```bash
python -m ipdb -c continue script.py
```

### pytest + ipdb

```bash
pytest --pdb --pdbcls=IPython.terminal.debugger:TerminalPdb
```

## pdbpp / pdb++

`pdbpp` (pronounced "pdb plus plus") is a sibling: install and pdb is replaced. Sticky mode shows the source pane above the prompt.

### Install

```bash
pip install pdbpp
```

`import pdb` now imports pdbpp. Verify:

```bash
python -c "import pdb; print(pdb.__file__)"
```

### Sticky mode

Inside pdb++:

```bash
(Pdb) sticky
```

The terminal splits — source on top, prompt on bottom. Re-toggle to disable.

### Configuration file

`~/.pdbrc.py` (Python) or `~/.pdbrc` (commands):

```bash
echo 'import pdb
class Config(pdb.DefaultConfig):
    sticky_by_default = True
    use_pygments = True
    bg = "dark"' > ~/.pdbrc.py
```

Per-project `.pdbrc` runs automatically when pdb starts in that directory.

### pdbpp vs ipdb

- pdbpp: sticky source view, syntax highlighting, drop-in replacement (replaces stdlib pdb).
- ipdb: IPython kernel, magics, tab-completion, NOT a drop-in (you must call `ipdb.set_trace()` or set PYTHONBREAKPOINT).

You can install both — `pdb++` replaces stdlib pdb; `ipdb` you import explicitly.

## Remote pdb

When the process you want to debug isn't on a TTY (Django/Flask in container, daemon, systemd unit), you need a remote debugger.

### rpdb (telnet to debugger)

```bash
pip install rpdb
```

```bash
echo 'import rpdb
x = 1
rpdb.set_trace()
print(x)' > rpdb_demo.py
```

```bash
python rpdb_demo.py &
```

The script blocks waiting for connection on default port 4444:

```bash
nc 127.0.0.1 4444
```

Custom port:

```bash
python -c "
import rpdb
rpdb.set_trace(addr='0.0.0.0', port=12345)
"
```

```bash
nc localhost 12345
```

### web-pdb (browser UI)

```bash
pip install web-pdb
```

```bash
echo 'import web_pdb
x = 1
web_pdb.set_trace()
print(x)' > web_pdb_demo.py
```

```bash
python web_pdb_demo.py
```

Open `http://localhost:5555` in your browser.

Custom port:

```bash
python -c "
import web_pdb
web_pdb.set_trace(host='0.0.0.0', port=8765)
"
```

### Inside Docker

```bash
docker run -it -p 5678:5678 --rm myapp python app.py
```

The container needs to expose the debugger port (`-p 5678:5678`), and the script must bind to `0.0.0.0` (not `127.0.0.1`).

## debugpy

Microsoft's debug adapter — the protocol VS Code uses. Modern, async-aware, multi-thread, attach or launch.

### Install

```bash
pip install debugpy
```

### Listen and wait for client

```bash
python -m debugpy --listen 5678 --wait-for-client app.py
```

```bash
python -m debugpy --listen 0.0.0.0:5678 --wait-for-client app.py
```

`--wait-for-client` blocks until VS Code (or any DAP client) attaches.

### Attach without waiting

```bash
python -m debugpy --listen 5678 app.py
```

The script runs immediately; you can attach any time.

### Programmatic API

```bash
python -c "
import debugpy
debugpy.listen(5678)
debugpy.wait_for_client()
debugpy.breakpoint()
print('attached')
"
```

### Integrate with breakpoint() builtin

```bash
export PYTHONBREAKPOINT=debugpy.breakpoint
```

After `debugpy.listen(...)` was called, `breakpoint()` triggers a DAP breakpoint instead of pdb.

### VS Code launch.json (attach)

```bash
echo '{"version":"0.2.0","configurations":[{"name":"Python: Remote Attach","type":"python","request":"attach","connect":{"host":"localhost","port":5678}}]}' > .vscode/launch.json
```

### Containerized debugging

```bash
docker run -it -p 5678:5678 --rm -v $(pwd):/app myapp \
  python -m debugpy --listen 0.0.0.0:5678 --wait-for-client /app/main.py
```

VS Code attaches to `localhost:5678`; container forwards.

### debugpy + pytest

```bash
python -m debugpy --listen 5678 --wait-for-client -m pytest -x test_my.py
```

## Conditional Breakpoints

Stop only when the predicate is true. Save eternity scrolling through hits.

### Syntax

```bash
(Pdb) b file.py:42, condition_expr
```

The comma is required; `condition_expr` is any Python expression evaluated at the breakpoint frame.

### Examples

```bash
(Pdb) b worker.py:50, len(queue) > 100
```

```bash
(Pdb) b parser.py:200, token.type == "ERROR"
```

```bash
(Pdb) b loop.py:30, i == 999
```

```bash
(Pdb) b parse.py:42, isinstance(node, ast.Call) and node.func.attr == "exec"
```

### Modify condition on existing breakpoint

```bash
(Pdb) condition 1 i > 50
```

```bash
(Pdb) condition 1
```

`condition N expr` updates breakpoint N's condition. `condition N` (no expr) removes the condition.

### Common patterns

Stop on the Nth iteration:

```bash
(Pdb) b 42, i == 999
```

Or use `ignore`:

```bash
(Pdb) b 42
(Pdb) ignore 1 999
```

Stop on a specific request:

```bash
(Pdb) b views.py:50, request.user.id == 42
```

Stop on edge case:

```bash
(Pdb) b parse.py:30, len(input) > 1024 * 1024
```

## Watchpoints

pdb has NO native watchpoints — you cannot say "stop when self.foo changes." Workarounds:

### Monkey-patch __setattr__

```bash
echo 'class Watched:
    pass
def trace_setattr(self, name, value):
    if name == "foo":
        import pdb; pdb.set_trace()
    object.__setattr__(self, name, value)
Watched.__setattr__ = trace_setattr
w = Watched()
w.foo = 1
w.foo = 2' > watch.py
```

```bash
python watch.py
```

### Property + breakpoint

```bash
echo 'class C:
    _foo = None
    @property
    def foo(self): return self._foo
    @foo.setter
    def foo(self, v):
        if v != self._foo:
            breakpoint()
        self._foo = v
c = C()
c.foo = 5
c.foo = 7' > prop_watch.py
```

```bash
python prop_watch.py
```

### sys.settrace for variable change

```bash
echo 'import sys
last = {}
def trace(frame, event, arg):
    if event == "line":
        cur = frame.f_locals.get("x")
        if last.get("x") != cur:
            print(f"x changed: {last.get(\"x\")} -> {cur}")
            last["x"] = cur
    return trace
sys.settrace(trace)
x = 1
x = 2
x = 3' > settrace_watch.py
```

```bash
python settrace_watch.py
```

This is high-overhead — only useful for tight scopes.

### Pdb subclass with custom dispatch

```bash
echo 'import pdb
class WatchPdb(pdb.Pdb):
    def __init__(self, *a, **kw):
        super().__init__(*a, **kw)
        self.last = None
    def user_line(self, frame):
        cur = frame.f_locals.get("x")
        if cur != self.last:
            self.last = cur
            super().user_line(frame)
        else:
            self.set_continue()
WatchPdb().set_trace()' > watch_pdb.py
```

## Inspecting Generators and Iterators

Stepping into a generator does NOT iterate it — it just enters the generator object's `__next__` machinery. You'll often see surprising behavior.

### The gotcha

```bash
echo 'def gen():
    yield 1
    yield 2
    yield 3
g = gen()
breakpoint()' > gen_demo.py
```

```bash
python gen_demo.py
```

At the prompt:

```bash
(Pdb) s
```

This enters the iterator protocol code, NOT the generator body.

### Step into the generator body

Use `next(g)` from `interact`:

```bash
(Pdb) interact
>>> next(g)
1
>>> next(g)
2
>>> ^D
```

Or set a breakpoint INSIDE the generator function and continue:

```bash
(Pdb) b gen
(Pdb) c
```

Now `next(g)` from interact stops on the first `yield`-or-after.

### Step over a generator-consuming loop

```bash
(Pdb) until
```

Or set a breakpoint past the loop:

```bash
(Pdb) b 50
(Pdb) c
```

### Materialize a generator for inspection

```bash
(Pdb) p list(g)
```

This consumes the generator! Make a copy with `itertools.tee` first:

```bash
(Pdb) !import itertools
(Pdb) !g, peek = itertools.tee(g)
(Pdb) p list(peek)
```

## Inspecting Async Code

Under asyncio, the call stack includes the event loop's machinery — `_run`, `__step`, callbacks. Walking up too far drops you into asyncio internals.

### Walk to user code

```bash
(Pdb) w
```

Look for frames that match your file paths; `u` to those frames. Skip ones in `asyncio/`.

### Inspect the running loop

```bash
(Pdb) interact
>>> import asyncio
>>> loop = asyncio.get_running_loop()
>>> [t for t in asyncio.all_tasks(loop)]
>>> [t.get_stack() for t in asyncio.all_tasks(loop)]
>>> ^D
```

### Force a specific coroutine to run

```bash
(Pdb) !import asyncio
(Pdb) !asyncio.get_event_loop().run_until_complete(my_coro())
```

(Only safe outside an active loop. If a loop is running, use `asyncio.ensure_future` and `c`.)

### asyncio debug mode

```bash
PYTHONASYNCIODEBUG=1 python script.py
```

Logs slow coroutines, never-awaited tasks, double-await — preventive debugging without pdb.

### asyncio task dump

```bash
(Pdb) interact
>>> import asyncio, sys
>>> for task in asyncio.all_tasks():
...     task.print_stack(file=sys.stdout)
```

## Tracing — sys.settrace

`sys.settrace(callback)` is the underlying mechanism pdb uses. It calls back on every line, function, exception event.

### Cost

10x to 100x slowdown when active. Acceptable for debugging, ruinous for benchmarking.

### Minimal tracer

```bash
echo 'import sys
def tracer(frame, event, arg):
    if event == "line":
        print(f"{frame.f_code.co_filename}:{frame.f_lineno}")
    return tracer
sys.settrace(tracer)
def f(): return 1+2
f()' > trace_demo.py
```

```bash
python trace_demo.py
```

### Coverage as low-overhead alternative

For line tracking only:

```bash
pip install coverage
```

```bash
coverage run script.py
```

```bash
coverage report
```

```bash
coverage html
```

Coverage uses `sys.settrace` but only counts hits — much cheaper than full trace.

### viztracer for visual flame chart

```bash
pip install viztracer
```

```bash
viztracer script.py
```

```bash
vizviewer result.json
```

Uses `sys.setprofile` (lower overhead than settrace) and produces a Chrome-trace timeline.

### Disable tracing

```bash
python -c "import sys; sys.settrace(None)"
```

`sys.settrace(None)` removes the tracer; helpful if a leftover from pdb or coverage is hurting perf.

### sys.setprofile for function-level only

```bash
echo 'import sys
def prof(frame, event, arg):
    if event == "call":
        print(f"call {frame.f_code.co_name}")
sys.setprofile(prof)
def f(): pass
f()' > prof_demo.py
```

```bash
python prof_demo.py
```

`setprofile` only fires on function call/return, much cheaper than `settrace`'s line events.

## Common Errors

Real error messages and what to do.

### `*** AttributeError: 'NoneType' object has no attribute 'X'`

```bash
(Pdb) p obj.field
*** AttributeError: 'NoneType' object has no attribute 'field'
```

The object is None. The bug is upstream — go up the stack and find where it should have been set.

```bash
(Pdb) u
(Pdb) p obj
```

### `*** NameError: name 'X' is not defined`

```bash
(Pdb) p result
*** NameError: name 'result' is not defined
```

The name doesn't exist in this frame. Check `args` and `locals()`:

```bash
(Pdb) args
```

```bash
(Pdb) p locals()
```

```bash
(Pdb) p sorted(locals().keys())
```

It may exist in an enclosing frame:

```bash
(Pdb) u
(Pdb) p result
```

### `*** SyntaxError: invalid syntax`

```bash
(Pdb) x = 5
*** SyntaxError: invalid syntax
```

`x = 5` collides with pdb's parsing. Use the `!` prefix:

```bash
(Pdb) !x = 5
```

Same for any statement that pdb mistakes for a command:

```bash
(Pdb) !import json
```

### `BdbQuit`

```bash
Traceback (most recent call last):
  File "...", line N, in ...
bdb.BdbQuit
```

`q` exits pdb by raising `BdbQuit`. If your code wraps the failing call in a `try/except`, `BdbQuit` propagates up. Catch it explicitly only if you really need to recover:

```bash
echo 'import bdb
try:
    breakpoint()
except bdb.BdbQuit:
    print("user quit pdb")' > bdbquit.py
```

```bash
python bdbquit.py
```

### `*** Blank or comment`

```bash
(Pdb) # not a real comment
*** Blank or comment
```

pdb treats `#` lines as comments. To pass `#` through, use `!`:

```bash
(Pdb) !x = "# literal"
```

### `*** The specified object 'X' is not a function`

```bash
(Pdb) b X
*** The specified object 'X' is not a function or was not found along sys.path.
```

Either `X` isn't imported in the current frame, or you wrote a method without the class. Use `module.func` or `class.method`:

```bash
(Pdb) b mymodule.myfunc
```

```bash
(Pdb) b MyClass.method
```

### `*** Newest frame`

```bash
(Pdb) d
*** Newest frame
```

You're already at the bottom — you can't go further down.

### `*** Oldest frame`

```bash
(Pdb) u
*** Oldest frame
```

You're at the outermost frame; can't go up further.

### Frozen module / no source

```bash
(Pdb) ll
*** No source available
```

The frame is in a C extension or frozen module. Use `whatis` and `p` instead of source listing.

### Tab completion not working

Make sure readline is available:

```bash
python -c "import readline; print(readline.__doc__)"
```

On macOS the system Python may not link readline; use Homebrew or pyenv:

```bash
brew install python
```

```bash
pyenv install 3.12
```

## Common Gotchas

Each gotcha shows the broken pattern and the fixed pattern.

### Gotcha 1: `q` quits the program

bad — quits the script entirely:

```bash
(Pdb) q
```

After `q`, the script terminates with `bdb.BdbQuit`.

fixed — use `c` to continue, only `q` when really done:

```bash
(Pdb) c
```

To exit pdb but let the script continue, you must `c` until program completion.

### Gotcha 2: `next` collides with Python's builtin

bad — calling `next()` on an iterator at the prompt:

```bash
(Pdb) next(it)
```

This may be parsed as the `next` (n) pdb command with garbage args.

fixed — use `!` prefix or `p`:

```bash
(Pdb) !next(it)
```

```bash
(Pdb) p next(it)
```

### Gotcha 3: Assignment misread as comparison

bad — bare assignment:

```bash
(Pdb) x = 5
*** SyntaxError: invalid syntax
```

fixed — use the `!` prefix:

```bash
(Pdb) !x = 5
```

### Gotcha 4: `s` on multi-statement line

bad — expecting `s` to step through each statement on a line:

```bash
(Pdb) s
```

If line 42 is `a = f(); b = g(); c = h()`, `s` may step into `f` but the next `s` may not stop on `g` as you expect — pdb is line-based, not statement-based.

fixed — split lines or use `n` and re-list to track:

```bash
(Pdb) ll
(Pdb) n
(Pdb) ll
```

Better: write code one statement per line for debuggability.

### Gotcha 5: Committing pdb to production

bad — `import pdb; pdb.set_trace()` in committed code:

```bash
import pdb; pdb.set_trace()
```

If this ships to prod, the prod process hangs on a TTY that doesn't exist.

fixed — use `breakpoint()` (3.7+) and disable in prod with env:

```bash
breakpoint()
```

```bash
PYTHONBREAKPOINT=0 python app.py
```

In prod, set `PYTHONBREAKPOINT=0` (in systemd unit, k8s deployment, etc.). All `breakpoint()` calls become no-ops.

### Gotcha 6: pdb in a daemon process

bad — `breakpoint()` in a process started without TTY:

```bash
nohup python app.py &
```

The process blocks forever waiting for stdin.

fixed — use rpdb / web-pdb / debugpy for headless processes:

```bash
import rpdb; rpdb.set_trace()
```

```bash
import debugpy; debugpy.listen(5678); debugpy.wait_for_client()
```

### Gotcha 7: Stepping into asyncio internals

bad — `s` from inside a coroutine, ending up in `asyncio/base_events.py`:

```bash
(Pdb) s
(Pdb) s
(Pdb) s
> /usr/lib/python3.X/asyncio/base_events.py(...)_run()
```

You wandered into the loop machinery.

fixed — set a breakpoint at your destination, then `c`:

```bash
(Pdb) b my_module.my_coro
(Pdb) c
```

Or use `until` to skip to a later line.

### Gotcha 8: Generator `s` doesn't enter body

bad — expecting `s` to step into a generator function:

```bash
(Pdb) p g
<generator object gen at 0x...>
(Pdb) s
```

`s` enters `__next__` machinery, not the generator body.

fixed — set a breakpoint inside the generator body:

```bash
(Pdb) b gen_module.gen
(Pdb) c
```

Then `next(g)` (from interact or via `p next(g)`) stops on the first yield.

### Gotcha 9: `interact` assignments don't persist

bad — assigning in interact and expecting the running frame to see the change:

```bash
(Pdb) interact
>>> x = 999
>>> ^D
(Pdb) p x
1
```

`interact` operates on a copy of locals. CPython doesn't propagate changes back.

fixed — use `!` to assign in the actual frame:

```bash
(Pdb) !x = 999
(Pdb) p x
999
```

(Function locals are still tricky — Python optimizes away the dict; module-level globals work.)

### Gotcha 10: pdb breaks tab completion in vi/emacs mode

bad — pressing Tab to complete and getting nothing.

fixed — verify readline is linked:

```bash
python -c "import readline; print(readline)"
```

Install Python with readline support (pyenv, Homebrew). Or use ipdb / pdbpp which bundle their own line editor.

### Gotcha 11: Exception inside the debugger eats the frame

bad — typing an expression that raises:

```bash
(Pdb) p obj.method()
```

If `method()` raises, the traceback prints and you stay in pdb — but if it raises BdbQuit or KeyboardInterrupt, the session may end.

fixed — wrap risky expressions:

```bash
(Pdb) !try: r = obj.method()
(Pdb) !except Exception as e: r = e
(Pdb) p r
```

Or use `interact` for stronger isolation:

```bash
(Pdb) interact
>>> try: print(obj.method())
... except: import traceback; traceback.print_exc()
>>> ^D
```

### Gotcha 12: Conditional breakpoint syntax

bad — using `if` keyword:

```bash
(Pdb) b 42 if x > 0
*** Bad lineno: ...
```

fixed — use comma:

```bash
(Pdb) b 42, x > 0
```

### Gotcha 13: pdb in tests breaks pytest

bad — `breakpoint()` left in a test runs the prompt during CI:

```bash
def test_thing():
    breakpoint()
    assert do() == 1
```

fixed — use `pytest --pdb` to drop in only on failure:

```bash
pytest --pdb test_thing.py
```

Or set `PYTHONBREAKPOINT=0` in CI:

```bash
PYTHONBREAKPOINT=0 pytest
```

## Idioms

### Quick set-and-go

```bash
breakpoint()
```

That's it. 3.7+ everywhere.

### Switch debugger globally

```bash
export PYTHONBREAKPOINT=ipdb.set_trace
```

Now every `breakpoint()` lands in ipdb.

### Disable in prod

```bash
PYTHONBREAKPOINT=0 python app.py
```

### Post-mortem on any unhandled exception

```bash
python -m pdb -c continue script.py
```

### Conditional breakpoint for edge cases

```bash
(Pdb) b worker.py:50, len(queue) > 100
```

### Drop into REPL with full scope

```bash
(Pdb) interact
```

### One-liner post-mortem after caught exception

```bash
import pdb
try:
    risky()
except:
    pdb.post_mortem()
```

### Sticky source pane (pdbpp)

```bash
(Pdb) sticky
```

### Auto-display variable on every step

```bash
(Pdb) display len(queue)
```

### Skip first N hits of a hot breakpoint

```bash
(Pdb) ignore 1 1000
```

### Auto-pdb on uncaught exception

```bash
echo 'import sys
def excepthook(t, v, tb):
    import traceback, pdb
    traceback.print_exception(t, v, tb)
    pdb.post_mortem(tb)
sys.excepthook = excepthook' > ~/.pythonrc
```

```bash
export PYTHONSTARTUP=~/.pythonrc
```

### .pdbrc for default commands

```bash
echo 'alias pl pp locals()
alias pa pp args' > .pdbrc
```

`.pdbrc` in CWD or `~` runs each time pdb starts.

### Break on every call to a function

```bash
(Pdb) b mymodule.f
(Pdb) commands 1
(com) p locals()
(com) c
(com) end
```

This logs `locals()` on every call without stopping.

### Run-script-in-pdb-from-Python

```bash
python -c "
import pdb
pdb.run('exec(open(\"script.py\").read())')
"
```

## Tips

- Always prefer `ll` (longlist) over `l` (list) — full-function context beats 11 lines.
- Always use `!` for assignments and statements — saves the SyntaxError detour.
- Always use `breakpoint()` over `import pdb; pdb.set_trace()` — env-disable-able.
- `c` to continue, `q` only when truly done — `q` ends the program.
- For CI/prod, set `PYTHONBREAKPOINT=0` defensively.
- Install `ipdb` system-wide; you'll thank yourself.
- Use `display` for "watch this expression" — pdb has no real watchpoints.
- `pytest --pdb` is the canonical "debug failing test" — no code change.
- For Django/Flask in containers, install `debugpy` and attach from VS Code.
- Set `PYTHONASYNCIODEBUG=1` even outside pdb — finds bugs preventively.
- The post-mortem `python -m pdb -c continue script.py` is the killer feature — print/log debugging is rarely needed.
- `interact` to escape pdb's parser quirks when you need full Python.
- For deeply nested structures, `pp` (pretty print) over `p`.
- When pdb seems unresponsive, you may be in `interact` — Ctrl-D to return.
- Use a `.pdbrc` per project for aliases like `alias pl pp locals()`.
- `where` (or `bt`) early — orient yourself before stepping.
- `args` to confirm what was actually passed to the current function.
- Use `tbreak` for "stop here once" — saves the cleanup.
- Use `commands N` for tracepoint-style logging without stepping.
- Use `condition N expr` to refine an existing breakpoint without recreating it.
- Stick to `breakpoint()` over manual `import pdb; pdb.set_trace()` everywhere.
- For async, prefer `interact` + `asyncio.all_tasks()` over manual `u`/`d` walking.
- pdbpp's sticky mode is a game-changer for navigating large functions — keep it on by default.
- Use `coverage run` instead of `sys.settrace` if you only need line counts.
- viztracer beats pdb for performance investigations — pdb is for correctness, viztracer for speed.
- When debugging an iterator, `list(iter)` consumes it — use `itertools.tee` to peek.
- Multi-process pdb: each child process writes to its own TTY; use `rpdb` with unique ports per child.
- Threading: pdb stops the thread that hit the breakpoint, but other threads keep running. Use `threading.settrace` for cross-thread.
- Global `excepthook` + `pdb.post_mortem` is the single most powerful pdb idiom — set it in `~/.pythonrc`.
- For C extensions, pdb cannot step in — use gdb on the Python interpreter for that.

## See Also

- gdb — analogous interactive debugger for C/C++ (and CPython itself with python-gdb)
- lldb — LLVM equivalent; default on macOS
- delve — Go's interactive debugger; the dlv-vs-pdb command shapes are similar
- bash — set/unset PYTHONBREAKPOINT, PYTHONASYNCIODEBUG, and other env vars covered above
- polyglot — language comparisons including Python debugging entry points

## References

- Official pdb documentation — https://docs.python.org/3/library/pdb.html
- bdb (debugger framework) — https://docs.python.org/3/library/bdb.html
- breakpoint() and PYTHONBREAKPOINT (PEP 553) — https://peps.python.org/pep-0553/
- ipdb — https://github.com/gotcha/ipdb
- pdbpp — https://github.com/pdbpp/pdbpp
- debugpy — https://github.com/microsoft/debugpy
- web-pdb — https://github.com/romanvm/python-web-pdb
- rpdb — https://github.com/tamentis/rpdb
- aiomonitor — https://github.com/aio-libs/aiomonitor
- pytest --pdb — https://docs.pytest.org/en/stable/how-to/failures.html
- coverage.py — https://coverage.readthedocs.io/
- viztracer — https://github.com/gaogaotiantian/viztracer
- sys.settrace — https://docs.python.org/3/library/sys.html#sys.settrace
- sys.setprofile — https://docs.python.org/3/library/sys.html#sys.setprofile
- asyncio debug mode — https://docs.python.org/3/library/asyncio-dev.html#debug-mode
- Python tutorial: debugging — https://docs.python.org/3/tutorial/errors.html
