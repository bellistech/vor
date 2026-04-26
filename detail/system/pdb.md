# pdb — Deep Dive

The Python Debugger, internals and theory. Why pdb works the way it does, what `sys.settrace` actually does at the C level, how Bdb maps breakpoints, why a 10x slowdown is fundamental, and how the ecosystem (ipdb, pdbpp, debugpy, web-pdb, remote-pdb, pdb-attach) builds on the same Bdb mechanics.

## Setup

`pdb` is the canonical Python source-level debugger. It is a stdlib module — no installation, no pip — every CPython distribution since 1.4 has shipped it. The module lives at `Lib/pdb.py` in the CPython source tree and the file is roughly 1900 lines of pure Python.

`pdb` is not a separate program. It is a Python module that, when imported, registers itself as the trace function via `sys.settrace`. There is no out-of-process attach the way gdb works against a native binary; pdb runs *inside the same Python process* it is debugging. This is fundamental: pdb is the same interpreter, sharing the same heap, the same GIL, the same import system, the same threading model. There is no "debugger process" and "debuggee process" — there is one process, and pdb pauses it by raising the trace function and dropping into a REPL.

The class hierarchy is:

```
Bdb            (Lib/bdb.py)            — generic breakpoint/dispatch framework
  └── Pdb     (Lib/pdb.py)            — interactive command-line UI on top of Bdb
        ├── Restart                   — small exception type for `restart`
        └── (subclassed by ipdb, pdbpp, etc.)
```

`Bdb` is the *Basic Debugger* — a base class meant to be subclassed. It encapsulates everything related to the trace function, breakpoints, frame dispatch, and step semantics. It does *not* know how to interact with a user. `Bdb` exposes hooks like `user_call`, `user_line`, `user_return`, `user_exception` that subclasses override to react to events.

`Pdb` extends `Bdb` and adds the command-line interface. It also extends `cmd.Cmd` (Python's stdlib REPL framework), giving it the prompt, command parsing, help-text autoderivation, and history mechanism.

This separation matters for the ecosystem. `ipdb` doesn't reimplement breakpoints — it inherits Bdb. `pdbpp` doesn't reimplement breakpoints — it inherits Pdb. The differences are entirely at the UI layer.

## Architecture

`Bdb` extends nothing in the inheritance sense (other than `object`), but conceptually it *is* a tracer that turns Python's low-level trace events into high-level "is this a breakpoint?" decisions. The dispatch graph looks like this:

```
sys.settrace(self.trace_dispatch)
        │
        ├── on 'call'      event ──▶ Bdb.dispatch_call(frame, arg)
        │                                ├── stops_here? ──▶ self.user_call(frame, arg)
        │                                └── return self.trace_dispatch (re-arm per-frame)
        │
        ├── on 'line'      event ──▶ Bdb.dispatch_line(frame)
        │                                ├── stop_here(frame)?
        │                                ├── break_here(frame)?
        │                                └── self.user_line(frame)
        │
        ├── on 'return'    event ──▶ Bdb.dispatch_return(frame, arg)
        │                                ├── self.user_return(frame, arg)
        │                                └── reset stoplineno if leaving
        │
        └── on 'exception' event ──▶ Bdb.dispatch_exception(frame, arg)
                                         └── self.user_exception(frame, arg)
```

The four canonical events are `call`, `line`, `return`, `exception`. The trace function is a function with the signature `trace(frame, event, arg) -> trace`. Critically, the *return value* of the trace function becomes the *per-frame* trace function for the next event in that frame. This is how pdb selectively turns line tracing on or off in different frames — by returning `None` from `dispatch_call`, no per-line trace is set up for uninteresting frames; by returning `self.trace_dispatch`, line events fire for every line of that frame.

The reason this matters for performance: the trace function is called *for every line of Python source executed*. If you return `self.trace_dispatch` from a `call`, every subsequent line in that frame causes a full Python-level callback. Bdb's optimization story is mostly about *not* returning `self.trace_dispatch` for frames you don't care about.

## sys.settrace

`sys.settrace(func)` is the canonical mechanism. It registers a Python callable as the global trace function for the current thread. Internally, the CPython runtime calls `PyEval_SetTrace(tracefunc, arg)` in `Python/sysmodule.c`, which writes into the current thread state's `c_tracefunc` slot:

```c
// Python/sysmodule.c (paraphrased)
void PyEval_SetTrace(Py_tracefunc func, PyObject *arg) {
    PyThreadState *tstate = _PyThreadState_GET();
    PyObject *temp = tstate->c_traceobj;
    Py_XINCREF(arg);
    tstate->c_tracefunc = NULL;
    tstate->c_traceobj = NULL;
    /* Must make sure that profilefunc is not called
       if 'temp' is freed */
    tstate->use_tracing = tstate->c_profilefunc != NULL;
    Py_XDECREF(temp);
    tstate->c_tracefunc = func;
    tstate->c_traceobj = arg;
    /* Flag that tracing or profiling is turned on */
    tstate->use_tracing = ((func != NULL)
                           || (tstate->c_profilefunc != NULL));
}
```

The interpreter's evaluation loop in `Python/ceval.c` (the giant `_PyEval_EvalFrameDefault` function) checks `tstate->use_tracing` at every bytecode dispatch. If set, it calls `call_trace_protected(tstate->c_tracefunc, tstate->c_traceobj, ...)` for the appropriate event. This is where the cost comes from: every Python bytecode that corresponds to a new line, every function call, every return, every exception triggers a C function call that ultimately calls back into Python to invoke the trace function.

The key insight on overhead: in normal execution, the interpreter evaluates a bytecode and moves on. With tracing on, every line-corresponding bytecode triggers:

1. A check of `tstate->use_tracing`
2. A C-level call to `call_trace`
3. An acquisition of the GIL re-entry (already held, but accounted for)
4. A construction of the Python args (frame, event string, arg)
5. A `PyObject_Call` into the user's Python trace function
6. Inside the trace function (Bdb), dictionary lookups for the breakpoint set
7. Possibly an `eval()` of a condition expression
8. Return through all the layers

Even when no breakpoint matches, the trace function still runs. This is "what slows down a debugger by 10x" — for naive Python code, the per-line overhead can dominate. CPython 3.12 introduced PEP 669 (`sys.monitoring`) specifically to address this by allowing fine-grained event subscriptions, but pdb itself still uses `sys.settrace` for compatibility.

`sys.settrace` is *thread-local*. Setting it in one thread does not affect other threads. This is why multithreaded debugging is hard (see Multithreading section).

## sys.setprofile vs sys.settrace

There are two trace mechanisms in CPython:

| Mechanism | Events fired | Cost |
|-----------|--------------|------|
| `sys.settrace` | `call`, `line`, `return`, `exception`, `opcode` | Per-line overhead. ~10x slower. |
| `sys.setprofile` | `call`, `return`, `c_call`, `c_return`, `c_exception` | Per-call overhead only. ~2-3x slower. |

The critical difference: `setprofile` does *not* fire on every line. It only fires on function call boundaries. This makes it *much* cheaper for profiling purposes — a profiler doesn't need per-line resolution; it needs which function is on the stack and how long it's been there. `cProfile` and `profile` (the stdlib profilers) use `setprofile`.

`setprofile` also adds the `c_call` and `c_return` events, which fire when Python code calls into a C extension. `settrace` does *not* report C-level transitions — from settrace's perspective, a call into `numpy.dot` looks like a single instruction that takes a long time. This is a structural limitation: the trace function is called from the Python evaluation loop, and C extensions execute outside that loop.

For pdb, this means: if you set a breakpoint in a Python function called by a C extension (e.g., a callback into Python from a sort comparator), pdb works because eventually the Python eval loop runs again. But if you want to step *into* a C function, pdb can't help — there is no Python eval loop running there. You'd need gdb or lldb attached to the Python process.

## Frame Object

The `PyFrameObject` in CPython (`Include/internal/pycore_frame.h` and `Objects/frameobject.c`) is the runtime representation of an executing function. Every Python function call creates a frame. pdb's entire model is built around frames.

Key fields exposed at the Python level via `frame.f_*`:

- **`f_back`** — pointer to the caller's frame. Forms a singly-linked stack via parent links. Walking `f_back` repeatedly gives you the call stack. This is what `pdb`'s `up`/`down` commands traverse.
- **`f_code`** — pointer to the `PyCodeObject` for this function. This is the compiled bytecode plus metadata; a single code object can be the target of many frames (every call to a function creates a new frame for the same code object).
- **`f_lineno`** — current source line number being executed. Computed from `f_lasti` via the line-number table (see Code Object section).
- **`f_lasti`** — *last instruction* index, the bytecode offset (in bytes) of the most recently executed instruction. This is the actual program counter for pdb's stepping logic. `f_lineno` is derived; `f_lasti` is authoritative.
- **`f_locals`** — dict of local variables. *Note*: in CPython, locals are stored in a fixed array (`f_localsplus`) on the frame, not a dict. `f_locals` is a dict view that is *materialized* on access via `PyFrame_FastToLocalsWithError`. This means pdb sees a *snapshot* of locals; assigning to `f_locals['x']` does NOT change the actual local variable in the running frame.
- **`f_globals`** — dict of module globals.
- **`f_builtins`** — dict of builtins (usually `__builtins__`).
- **`f_trace`** — *per-frame* trace function. This is what makes selective tracing work. When the global trace function returns a callable from `dispatch_call`, that callable is stored in `frame.f_trace` and called for subsequent line events in that frame.
- **`f_trace_lines`** — boolean (3.7+); if False, line events are suppressed for this frame even if `f_trace` is set.
- **`f_trace_opcodes`** — boolean (3.7+); if True, fire trace events on every opcode (not just every line).

The `f_locals` materialization quirk is a recurring source of confusion. If you do `frame.f_locals['x'] = 42` in pdb, the change does not propagate to the running code. To actually mutate a local from pdb, you have to use the `!` prefix (or just type the assignment) which compiles and executes Python in the frame's namespace, and then in 3.13+ there's `PyFrame_LocalsToFast` / runtime updates that handle this — but historically, mutating locals from a debugger has been semi-broken.

## Code Object

The `PyCodeObject` (`Objects/codeobject.c`, `Include/cpython/code.h`) is the immutable result of compiling Python source. It is shared across all invocations of a function. pdb often inspects code objects to map between source lines and bytecode offsets.

Key fields:

- **`co_code`** — bytes object containing the bytecode. Each instruction is 2 bytes: opcode + operand.
- **`co_consts`** — tuple of constants referenced by the code (literals, nested code objects for closures).
- **`co_varnames`** — tuple of local variable names (parameters first, then locals).
- **`co_freevars`** — tuple of free variable names (closed-over from outer scopes).
- **`co_cellvars`** — tuple of cell variable names (variables bound in this scope and used by inner scopes via closure).
- **`co_names`** — tuple of names used by the code (global lookups, attribute accesses).
- **`co_filename`** — source filename (string).
- **`co_firstlineno`** — first source line of the function.
- **`co_lnotab`** — *line-number table*. Compact byte-encoded mapping from bytecode offset to source line. Used to compute `f_lineno` from `f_lasti`. **In Python 3.10+, replaced by `co_linetable`** which uses the PEP 626 format and supports column information.

The `co_lnotab` format (pre-3.10) is a sequence of `(byte_increment, line_increment)` pairs. To find the line for a given `f_lasti`, walk the table summing increments until you hit the offset. This is a tiny encoding designed to fit in a few bytes per function.

```python
# Example: inspect line table
import dis
def f(x):
    y = x + 1
    return y
print(f.__code__.co_lnotab)  # bytes: e.g. b'\x00\x01\x08\x01'
print(list(dis.findlinestarts(f.__code__)))
# [(0, 2), (8, 3)]  (offset, line)
```

PEP 626 (Python 3.10) reformatted this to the `co_linetable` (format documented in `Objects/lnotab_notes.txt`). PEP 657 (Python 3.11) added column information so error messages can point to the *exact subexpression* that errored. pdb's `list` and `longlist` commands use `dis.findlinestarts` to know where each line begins.

The relationship `f_lasti -> f_lineno` is what makes the difference between "next" and "step" decisions. `next` waits until `f_lineno` changes within the same frame. `step` stops at the next frame change.

## The Bdb Class

`Bdb` (`Lib/bdb.py`) is the framework. The most important methods:

- **`run(cmd, globals=None, locals=None)`** — enable tracing, then `exec(cmd, globals, locals)`. Used by `pdb.run(...)`.
- **`runeval(expr, globals=None, locals=None)`** — same but evaluates and returns a value.
- **`runcall(func, *args, **kwargs)`** — calls `func(*args, **kwargs)` with tracing on. Used internally by `pdb.runcall`.
- **`runctx(cmd, globals, locals)`** — convenience wrapper; common entry point.
- **`set_trace(frame=None)`** — enable tracing starting at the given frame (or caller). Used by `pdb.set_trace()`.
- **`set_step()` / `set_next(frame)` / `set_return(frame)` / `set_until(frame, lineno)` / `set_continue()` / `set_quit()`** — change the stop semantics for the next event.
- **`canonic(filename)`** — normalize a filename (resolve symlinks, absolute path, lowercase on Windows). Used as the key in the breakpoint map so equivalent paths are unified.
- **`set_break(filename, lineno, temporary=False, cond=None, funcname=None)`** — register a new breakpoint.
- **`clear_break(filename, lineno)`** — remove.
- **`get_break(filename, lineno)`** — query.

The `Breakpoint` class is a separate class within `bdb.py`. Each breakpoint has:

```python
class Breakpoint:
    next = 1                    # auto-increment ID across all breakpoints
    bplist = {}                 # dict: (file, line) -> [Breakpoint, ...]
    bpbynumber = [None]         # list: [None, bp1, bp2, ...] indexed by number

    def __init__(self, file, line, temporary=False, cond=None, funcname=None):
        self.file = file
        self.line = line
        self.temporary = temporary
        self.cond = cond
        self.enabled = True
        self.ignore = 0         # hit count to skip
        self.hits = 0
        self.number = Breakpoint.next
        Breakpoint.next += 1
        Breakpoint.bplist.setdefault((file, line), []).append(self)
        Breakpoint.bpbynumber.append(self)
```

The class-level `bplist` dict is keyed by `(canonical_filename, lineno)`. Lookups during line events are O(1) hashmap lookups. Multiple `Breakpoint` instances at the same `(file, line)` are stored as a list — useful when you have a conditional and an unconditional breakpoint on the same line (rare in practice).

`Bdb.set_break` calls `self.canonic(filename)` first, then constructs a `Breakpoint`. This is why typing `b ./foo.py:10` and `b /full/path/foo.py:10` both work: they canonicalize to the same key.

## Breakpoint Implementation

The hot path is `Bdb.dispatch_line(frame)`, called for every line in a traced frame:

```python
def dispatch_line(self, frame):
    if self.stop_here(frame) or self.break_here(frame):
        self.user_line(frame)
        if self.quitting: raise BdbQuit
    return self.trace_dispatch
```

`stop_here(frame)` checks if we're stepping (`step`/`next`/`return`/`until`).
`break_here(frame)` checks the breakpoint map:

```python
def break_here(self, frame):
    filename = self.canonic(frame.f_code.co_filename)
    if filename not in self.breaks:
        return False
    lineno = frame.f_lineno
    if lineno not in self.breaks[filename]:
        # The line itself has no breakpoint, but maybe the line is the
        # first line of a function which has a breakpoint
        lineno = frame.f_code.co_firstlineno
        if lineno not in self.breaks[filename]:
            return False
    # flag says ok to delete temp. bp
    (bp, flag) = effective(filename, lineno, frame)
    if bp:
        self.currentbp = bp.number
        if flag and bp.temporary:
            self.do_clear(str(bp.number))
        return True
    return False
```

`effective(file, line, frame)` evaluates the breakpoint's condition (if any), checks ignore count, increments hits, and decides whether to actually stop. This is where conditional breakpoints get evaluated.

The cost: even with no breakpoints, every traced line goes through `dispatch_line`, which does a dict membership check. The check is fast (hash lookup) but it's *every line*. If you trace 10 million lines of Python, that's 10 million dict lookups. This is why `sys.settrace` is structurally slower than sampling profilers like `py-spy` (which read frame state from outside the process via `process_vm_readv` — no callback into Python at all).

## Conditional Breakpoints

A conditional breakpoint stops only when an expression evaluates to truthy in the frame's namespace. The implementation in `bdb.effective`:

```python
def effective(file, line, frame):
    possibles = Breakpoint.bplist[file, line]
    for b in possibles:
        if not b.enabled:
            continue
        if not b.cond:
            # Unconditional and enabled
            ...
        else:
            try:
                val = eval(b.cond, frame.f_globals, frame.f_locals)
                if val:
                    if b.ignore > 0:
                        b.ignore -= 1
                        continue
                    else:
                        return (b, True)
            except Exception:
                # if eval fails, treat as if it was true and stop
                return (b, False)
    return (None, None)
```

Two things stand out:

1. `eval(b.cond, frame.f_globals, frame.f_locals)` — the condition is compiled (cached as a code object on the Breakpoint) and run in the frame's namespace. This means it has access to all local and global variables.
2. If the condition raises, pdb treats it as "stop anyway and tell the user." This is a design choice — better to surface a broken condition than silently skip.

The performance impact: a conditional breakpoint is *more* expensive than an unconditional one, because eval runs every time the line is hit. But it's vastly cheaper than stepping through every line manually until the condition is met.

```python
import pdb

def f():
    for i in range(10000):
        # set conditional breakpoint: b f:N, i == 9999
        x = i * 2
    return x
```

Without a conditional breakpoint, you'd `next` 9999 times. With it, pdb evaluates `i == 9999` once per iteration (10000 evals) and stops only on match. Even with the eval cost, it's far faster than user input.

## Stepping

The stepping commands map to flags on the Bdb instance:

| Command | Method | Mechanism |
|---------|--------|-----------|
| `step` (s) | `set_step()` | `self._set_stopinfo(None, None)` — stop on *any* next event |
| `next` (n) | `set_next(frame)` | `self._set_stopinfo(frame, frame)` — stop only when current frame is `frame` *or* parent of `frame` (i.e. we returned to it) |
| `return` (r) | `set_return(frame)` | `self._set_stopinfo(frame, frame.f_back)` — stop when current frame is `frame.f_back` (we returned) |
| `until N` (unt) | `set_until(frame, N)` | step within frame until line >= N |
| `continue` (c) | `set_continue()` | `self._set_stopinfo(self.botframe, None, -1)` — only stop at breakpoints |

`stop_here(frame)` is the unified check:

```python
def stop_here(self, frame):
    if frame is self.stopframe:
        if self.stoplineno == -1:
            return False
        return frame.f_lineno >= self.stoplineno
    if not self.stopframe:
        return True
    while frame is not None and frame is not self.stopframe:
        if frame is self.botframe:
            return True
        frame = frame.f_back
    return False
```

The semantics:

- **step** sets `stopframe = None`, `stoplineno = 0` → stop on first event in any frame.
- **next** sets `stopframe = current_frame`, `stoplineno = 0` → stop only when execution is within `current_frame` (not in a callee).
- **return** sets `stopframe = current_frame`, `stoplineno = -1` and waits for the `return` event in that frame.
- **continue** sets `stopframe = botframe`, `stoplineno = -1` → only breakpoints can stop.

The mechanism is elegant: by adjusting `stopframe` and `stoplineno`, all five stepping behaviors fall out of one `stop_here` function.

## The "F" command (continue/until)

`until [lineno]` resumes execution within the current frame until either:

1. A line with number greater than the current line is reached.
2. The current frame returns.
3. A breakpoint is hit.

Internally, `set_until(frame, lineno=None)` sets `stoplineno = lineno or frame.f_lineno + 1`. The next event check becomes "are we past this line?" This is useful for skipping over loops without setting a breakpoint after the loop.

```python
def f():
    for i in range(1000000):  # set breakpoint inside, then `until` skips to after loop
        process(i)
    return  # `until` lands here
```

Without `until`, you'd have to set a breakpoint on the line after the loop and `continue`. With `until`, no breakpoint needed.

## continue (c)

`continue` clears the step-into flag (`stopframe = self.botframe`, `stoplineno = -1`). The trace function still runs (because tracing is on globally), but `stop_here` always returns False unless the frame is the bottom frame. Only `break_here` can fire from this point. This is the lowest-overhead pdb mode: every line still goes through the trace function and dispatch_line, but stop_here short-circuits quickly.

To *fully* stop tracing, you'd call `sys.settrace(None)`. pdb doesn't do this on `continue` because it still wants to honor breakpoints. The cost stays at ~10x. To get back to native speed, you have to `quit` pdb or have `pdb.set_trace()` exit cleanly.

## The pdb.py CLI

`Pdb` extends `cmd.Cmd`, the stdlib REPL framework. `cmd.Cmd` provides:

- A `cmdloop()` method that reads input, parses the first word as the command name, and dispatches to `do_<name>`.
- Auto-help: `help foo` prints the docstring of `do_foo`.
- Tab completion via `complete_<name>` methods.
- History via readline integration.

So `pdb.Pdb` defines methods like:

```python
def do_break(self, arg):
    """Set a breakpoint at LINENO or FILE:LINENO. ..."""
    # parse arg, call self.set_break(...)

def do_continue(self, arg):
    """Continue execution, only stop when a breakpoint is encountered."""
    if not self.nosigint:
        try:
            Pdb._previous_sigint_handler = \
                signal.signal(signal.SIGINT, self.sigint_handler)
        except ValueError:
            pass
    self.set_continue()
    return 1
```

The return value from `do_*` matters: a truthy return from `do_*` exits the cmdloop, resuming program execution. `do_continue` returns 1 (resume); `do_step` returns 1 (resume); `do_help` returns nothing (loop again to read next command).

Help text is auto-derived from docstrings. `pdb.do_break.__doc__` is what `help break` prints. This is why every `do_*` method has a careful docstring — it's the user-facing help.

Aliases like `b` for `break` are wired via `do_b = do_break` (just a binding) in the Pdb class. `cmd.Cmd` sees them as separate methods and dispatches accordingly.

Tab completion: `complete_break(text, line, begidx, endidx)` returns candidate strings. Pdb implements completion for filenames and breakpoint numbers. ipdb extends this with completion for Python identifiers.

## breakpoint() Builtin (3.7+, PEP 553)

PEP 553 (Python 3.7) added the `breakpoint()` builtin. The reason: `import pdb; pdb.set_trace()` is verbose and hardcodes pdb. The builtin is a single name that respects an environment variable for swapping out the debugger.

The implementation in `Python/bltinmodule.c`:

```c
static PyObject *
builtin_breakpoint(PyObject *self, PyObject *const *args,
                   Py_ssize_t nargs, PyObject *keywords)
{
    PyObject *hook = PySys_GetObject("breakpointhook");
    if (hook == NULL) {
        PyErr_SetString(PyExc_RuntimeError,
                        "lost sys.breakpointhook");
        return NULL;
    }
    return _PyObject_FastCallKeywords(hook, args, nargs, keywords);
}
```

It calls `sys.breakpointhook(*args, **kwargs)`. The default `sys.breakpointhook` is set up in `Lib/sys.py`:

```python
def breakpointhook(*args, **kws):
    hookname = os.environ.get('PYTHONBREAKPOINT')
    if hookname is None or len(hookname) == 0:
        hookname = 'pdb.set_trace'
    elif hookname == '0':
        return None  # no-op; useful in production
    modname, dot, funcname = hookname.rpartition('.')
    if not modname:
        raise RuntimeError(...)
    import importlib
    module = importlib.import_module(modname)
    hook = getattr(module, funcname)
    return hook(*args, **kws)
```

So:

- `breakpoint()` → calls pdb.set_trace by default.
- `PYTHONBREAKPOINT=ipdb.set_trace breakpoint()` → routes to ipdb.
- `PYTHONBREAKPOINT=0 breakpoint()` → no-op. Production-safe.
- `PYTHONBREAKPOINT=web_pdb.set_trace breakpoint()` → web-pdb.

This decoupling is the actual contribution of PEP 553. You can sprinkle `breakpoint()` calls in code, ship to production with `PYTHONBREAKPOINT=0`, and develop with whatever debugger you prefer.

## post_mortem()

`pdb.post_mortem(traceback=None)` enters pdb on a traceback. If `traceback` is None, it uses `sys.last_traceback` (the last unhandled exception). Implementation:

```python
def post_mortem(t=None):
    if t is None:
        t = sys.exc_info()[2]
        if t is None:
            t = getattr(sys, 'last_traceback', None)
    if t is None:
        raise ValueError("no traceback...")
    p = Pdb()
    p.reset()
    p.interaction(None, t)
```

`p.interaction(None, t)` walks the traceback to its bottom frame and starts the REPL there. You can `up`/`down` through the traceback frames, inspect locals, and see exactly the state when the exception was raised.

The "drop into pdb on uncaught exception" idiom uses `sys.excepthook`:

```python
import sys, pdb, traceback

def excepthook(type_, value, tb):
    traceback.print_exception(type_, value, tb)
    pdb.post_mortem(tb)

sys.excepthook = excepthook
```

Or more directly, run with `python -m pdb script.py` which automatically does post-mortem on unhandled exceptions, or `python -i` which drops into a regular REPL (where you can then `import pdb; pdb.pm()` — `pm` is the alias for `post_mortem`).

`pdb.pm()` reads `sys.last_traceback` (set by the REPL when an exception is unhandled at the prompt). Useful in interactive sessions.

## Multithreading

pdb is single-threaded by design. `sys.settrace` is thread-local: setting it in thread A does not affect thread B. When you `pdb.set_trace()` in thread A, pdb's REPL runs on thread A. Other threads continue executing concurrently (subject to the GIL).

This causes confusion:

- Setting a breakpoint with `b foo.py:10` registers it in the Bdb instance, which is shared. But the breakpoint only triggers in threads that have the trace function set.
- If a worker thread is started *before* `pdb.set_trace()` is called, the worker's tracing is not configured. To make a thread debuggable, you'd have to call `sys.settrace` in the thread or use `threading.settrace`.

`threading.settrace(func)` registers a trace function for *future* threads created via the `threading` module. Existing threads are unaffected.

```python
import threading, sys, pdb

# Make all FUTURE threads inherit pdb tracing
threading.settrace(pdb.Pdb().trace_dispatch)
```

The catch: `trace_dispatch` is a method, and the Bdb state (breakpoints, step flags) is per-instance. Sharing one Pdb across threads is a recipe for confusion. Practical advice: use `breakpoint()` in the specific thread you want to debug, accept that other threads will continue, and use thread synchronization to pause workers if needed (e.g., put a `threading.Event` they wait on).

For real multithreaded debugging, use `debugpy` with VS Code, which has explicit support for multiple threads in the DAP protocol.

## asyncio Debug Mode

`asyncio.run(coro, debug=True)` (or `loop.set_debug(True)`) enables several diagnostics:

- Slow callback warnings (over 100ms by default).
- Logging of un-awaited coroutines.
- More detailed "coroutine was never awaited" tracebacks.
- Setting `PYTHONASYNCIODEBUG=1` env var has the same effect.

For pdb itself: you can `breakpoint()` inside a coroutine. The await suspension model means pdb pauses on the line *before* the await, you `step` into the awaited coroutine, but `step` over an `await` actually suspends the current task and may resume in a different task entirely. This makes async stepping confusing — your stack trace can change discontinuously.

debugpy is significantly better for async because it understands task switching at the protocol level.

```python
import asyncio

async def fetch(url):
    breakpoint()  # works; locals visible
    await asyncio.sleep(1)
    return url

async def main():
    return await fetch('http://example.com')

asyncio.run(main(), debug=True)
```

A common gotcha: pdb's REPL runs synchronously. While you're paused, the event loop is *not* running. This means timers don't fire, other tasks don't make progress, and any `await` you try to evaluate as a pdb expression won't work directly (you have to `asyncio.run` a fresh coroutine, but you'd be re-entering the loop).

## ipdb

`ipdb` is the IPython-flavored pdb. It is `pip install ipdb` and ships as a thin wrapper around `IPython.terminal.debugger.TerminalPdb`, which extends `Pdb`.

What it adds:

- IPython's prompt with syntax-aware tab completion (not just identifiers — full attribute/item completion).
- Colored output (syntax highlighting of source listings).
- Magic-aware: `?foo` prints help, `??foo` prints source, in some IPython contexts.
- Compatible with IPython kernels (notebooks).

What it preserves:

- All Bdb mechanics (breakpoints, stepping, conditions).
- All pdb commands.
- `breakpoint()` integration via `PYTHONBREAKPOINT=ipdb.set_trace`.

ipdb is a strict superset of pdb at the command level. If you can use pdb, you can use ipdb. The reason to use ipdb is purely UX: better completion and color matter when you're staring at a stack trace at 2am.

```bash
pip install ipdb
PYTHONBREAKPOINT=ipdb.set_trace python script.py
```

## pdbpp (pdb++)

`pdbpp` (pronounced "pdb plus plus") is `pip install pdbpp`. It *monkey-patches* pdb on import. The first thing `import pdbpp` does is replace `pdb.Pdb` with `pdbpp.Pdb`, so subsequent `import pdb; pdb.set_trace()` invocations get the enhanced version.

What it adds:

- Tab completion (Python expressions, not just commands).
- Syntax highlighting (uses Pygments).
- "Sticky mode": `sticky` toggles a mode where the source listing is continuously redisplayed at the top of the screen, so you always see context. This is the killer feature.
- Better tracebacks (with full source context).
- Smarter `display`/`undisplay` (watchpoints).
- `interact` command drops you into a regular Python REPL with the frame's locals.
- `track FOO` follows a variable through stepping.

The monkey-patching approach is controversial. It means once pdbpp is installed, *every* Python program in that environment that imports pdb gets pdbpp. This is intentional (so `breakpoint()` works without config) but can surprise users. Uninstall is `pip uninstall pdbpp`.

```bash
pip install pdbpp
python script.py
# When breakpoint() hits, you get pdbpp's enhanced UI automatically.
```

## debugpy (vs pdb)

`debugpy` is Microsoft's debugger backend for VS Code. It is *not* Bdb-based at the protocol level. It implements the **Debug Adapter Protocol (DAP)** — a JSON-RPC protocol over a TCP socket — that lets editors talk to debuggers.

Architecture:

```
VS Code (DAP client) <----TCP/JSON-RPC----> debugpy (DAP server) ───── inside Python process
                                                  └── uses sys.settrace under the hood
```

The actual tracing mechanism is still `sys.settrace` (or `sys.monitoring` on 3.12+). debugpy's contribution is:

- Translating DAP requests (set breakpoint, step over, evaluate, etc.) into trace function adjustments.
- Translating Python events (line, exception) into DAP notifications.
- Multithread support: properly enumerates threads, lets you select a thread to step.
- Variable inspection: returns DAP variable trees (suitable for VS Code's tree view).

Key features absent from pdb:

- Edit-and-continue for some changes.
- Visual breakpoint management in the editor.
- Conditional/hit-count breakpoints with proper UI.
- Logpoints (breakpoints that print without stopping).
- Function breakpoints by name.
- Exception breakpoints (filter by exception type).
- "Just My Code" (skip stdlib frames) — analogous to pdb's `skip_modules`.

debugpy is the production answer for editor-integrated Python debugging. pdb is the answer when all you have is a terminal.

```bash
pip install debugpy
# In VS Code, just press F5 with the right launch.json.
# Or attach to a running process:
python -m debugpy --listen 5678 --wait-for-client script.py
```

## web-pdb

`web-pdb` is `pip install web-pdb`. It launches a tiny web server and serves a pdb-like UI in the browser. Useful for headless servers where you can't easily get a terminal but you can hit a URL.

```python
import web_pdb
web_pdb.set_trace()
# Now connect a browser to http://localhost:5555/
```

The internals: web-pdb extends `pdb.Pdb` and overrides the input/output to read from / write to a web socket. The Bdb mechanics are unchanged. From the running Python process's perspective, it's pdb; from the user's perspective, it's a web UI.

Useful for:

- Debugging a long-running service in a Docker container.
- Debugging on a remote machine where SSH-and-pdb is awkward.
- Demoing pdb to people unfamiliar with terminal UIs.

## remote-pdb

`remote-pdb` is `pip install remote-pdb`. It binds pdb to a TCP socket so you can connect via `telnet` or `nc` from another machine.

```python
from remote_pdb import RemotePdb
RemotePdb('0.0.0.0', 4444).set_trace()
```

```bash
# On another machine:
nc 1.2.3.4 4444
# Now you have a pdb prompt.
```

Like web-pdb, it's pdb internally with redirected I/O. The protocol is just raw text — pdb's stdout to socket, socket input to pdb's stdin.

Security implication: anyone who can connect to the socket gets a Python REPL inside your process. Bind to localhost only or use SSH tunneling.

## pdb-attach

`pdb-attach` is `pip install pdb-attach`. It installs a signal handler that, on signal receipt, drops the process into pdb. The intended use: you have a running process, you want to debug it without restarting.

```python
import pdb_attach
pdb_attach.listen(50000)  # listen on port 50000
# Process runs normally. Later:
```

```bash
python -m pdb_attach <pid> 50000
# Connects to the listening port and gets a pdb prompt.
```

Alternatively, the SIGUSR2 trick (manual): install a signal handler that calls `pdb.set_trace()`:

```python
import signal, pdb

def handler(sig, frame):
    pdb.Pdb().set_trace(frame)

signal.signal(signal.SIGUSR2, handler)
# Now: kill -USR2 <pid> drops the process into pdb on whatever frame
# is current when the signal is delivered.
```

This is fragile: signals are delivered when the interpreter checks for pending signals (between bytecodes), so you don't get to choose exactly where you stop. Also pdb's I/O goes to the original stdin/stdout, which might be a daemon log.

For real "attach to a running Python process," the heavy-duty option is gdb with the CPython Python extension (`gdb -p <pid>` plus `python-stack` from the cpython gdb helpers). This walks the C stack, finds frame objects, and prints Python-level state. It can't *resume* with a Python REPL, but it shows you where the process is hung.

py-spy does similar via `process_vm_readv` (Linux) or task_for_pid + mach_vm_read (macOS) — pure read access, no settrace, no callback into Python at all.

## PEP 657 (3.11+)

PEP 657 (Python 3.11) added column information to tracebacks. The compiler now emits, alongside `co_linetable`, additional tables that map bytecode offsets to *column ranges* in the source.

Result: tracebacks now look like:

```
Traceback (most recent call last):
  File "x.py", line 1, in <module>
    print(d['a']['b']['c'])
          ~~~~~~~~~~~^^^^^
KeyError: 'c'
```

The carets point at the exact subexpression that errored. Before 3.11, you'd get the line but no column info, and you'd have to reason about which subscript failed.

Implementation: the new `_PyCode_Locations` data structure stores start_line, end_line, start_col, end_col for each bytecode offset. Costs a bit more memory but the developer ergonomics win is enormous.

For pdb: `where`/`w` and traceback-displaying commands now show column carets. `list` and `longlist` benefit too. The `f_lineno` field is still line-only, but pdb can ask the code object for the column range of the current `f_lasti`.

## Performance Cost

`sys.settrace` adds overhead at every Python instruction:

| Mode | Overhead |
|------|----------|
| No tracing | 1x baseline |
| settrace, return None per frame | ~1.2x (call/return only) |
| settrace, return self.trace_dispatch | ~10x (line events) |
| pdb running with breakpoints | ~10x to ~50x depending on conditions |
| cProfile (setprofile) | ~2-3x |
| py-spy (sampling, no callback) | ~0% (out of process) |

The 10x figure comes from microbenchmarks like a tight `for i in range(N): pass` loop. Real code spends more time per line (e.g., I/O, numpy), so the relative slowdown shrinks. But for CPU-bound Python, the cost is real.

This is why:

- You don't run profilers with `settrace`.
- You don't ship pdb in production hot paths.
- Sampling profilers (py-spy, austin) are preferred for production.

PEP 669 (Python 3.12, `sys.monitoring`) provides finer-grained subscriptions: a tool can ask for only `LINE` events in specific code objects, only `BRANCH` events, etc. The dispatch overhead is dropped from "always callback" to "only callback for subscribed events" — significantly cheaper. But pdb hasn't been fully ported to `sys.monitoring` in stdlib as of 3.12; it still uses `sys.settrace`.

## Bdb's Breakpoint Optimization

Bdb has several optimizations to reduce the per-line cost:

1. **Canonic filename caching**: `canonic` results are cached in `self.fncache`. Calling `canonic` on the same path repeatedly is O(1) after the first call.

2. **Per-frame trace decision**: `dispatch_call` checks if the called frame is in a "skipped" module (via `Bdb.skip_modules`). If so, return `None` (don't trace lines in this frame). This is how you skip stdlib frames.

   ```python
   pdb_instance = pdb.Pdb(skip=['django.*', 'pkg_resources.*'])
   ```

3. **`is_skipped_module(module_name)`** — fnmatch against the skip list.

4. **`break_anywhere(frame)`** — early check: does the frame's filename appear in `self.breaks` at all? If not, `break_here` returns False without doing per-line lookups.

5. **`stop_here` short-circuit**: if no stepping is active and no breakpoints exist, `stop_here` returns False immediately.

The "is this frame interesting?" pre-check is the most impactful. If a frame is in a skipped module, the per-frame trace function is None, which means *no line events fire for that frame at all* — the cost reverts to call/return only. This is why `skip` is a major performance lever in pdb when debugging code that calls into large frameworks.

## Display Expressions

`display EXPRESSION` adds an expression to a per-frame display list. Every time pdb pauses in that frame, it evaluates each display expression and prints `expr: old_value -> new_value` if the value changed.

Implementation: stored in `self.displaying[frame]` as a dict `{expr: last_value}`. On each `user_line`, pdb iterates the dict, evals each expression, compares to last value, prints diff, updates last value.

```
(Pdb) display x
display x: 0
(Pdb) n
> ...
display x: 0  --Now: 5
```

`undisplay [expr]` removes an entry. `display` with no argument lists current displays.

This is poor man's watch expression. ipdb and pdbpp expand it. The limitation: only fires when pdb pauses, not when the value changes. So if you `continue` past a change and stop later, you see the new value but you don't know exactly when it changed.

## Source-Aware Stepping

The `c_lasti` (last bytecode index) versus `f_lineno` (current line) distinction drives stepping decisions:

- `step` always returns True from `stop_here` (until exhausted).
- `next` checks if `frame is self.stopframe` AND `frame.f_lasti != original_lasti` (we moved bytecode).
- For multi-line statements (e.g., chained method calls split across lines), `f_lineno` may not change between bytecodes — but `f_lasti` does. So stepping uses `f_lasti` for "did anything happen" and `f_lineno` for "did we move to a new logical line."

`co_linetable` (PEP 626) handles the mapping. It's possible for two adjacent bytecodes to map to the same line (e.g., `LOAD_FAST` and `LOAD_FAST` for `a + b`), in which case `next` doesn't fire between them. But if a method call returns and you re-enter the same line, that's a new bytecode boundary — but pdb's logic suppresses it because the line didn't change.

This is why `next` over a line containing a complex expression "feels right": you stop on the next *logical* line, not the next bytecode.

## The pdb commands

Full command surface, with internal notes:

- **`continue`/`c`/`cont`** — `set_continue()`. Resumes until breakpoint.
- **`step`/`s`** — `set_step()`. Stops on next event.
- **`next`/`n`** — `set_next(self.curframe)`. Stops on next line in same or shallower frame.
- **`return`/`r`** — `set_return(self.curframe)`. Stops when current frame returns.
- **`until [N]`/`unt`** — `set_until(self.curframe, N)`. Stops at line >= N (default: current+1) or frame return.
- **`break`/`b`** — `do_break`. Adds a breakpoint.
- **`tbreak`** — temporary breakpoint, deleted on first hit.
- **`clear`/`cl`** — removes breakpoints.
- **`disable N`** — sets `bp.enabled = False`.
- **`enable N`** — sets `bp.enabled = True`.
- **`condition N expr`** — sets `bp.cond = expr`.
- **`ignore N count`** — sets `bp.ignore = count`.
- **`commands N`** — defines commands to run on hit (rarely used).
- **`where`/`w`/`bt`** — print stack trace via `print_stack_trace`.
- **`up`/`u`** — `self.curindex -= 1`; updates `self.curframe`.
- **`down`/`d`** — `self.curindex += 1`.
- **`list`/`l`** — `do_list`; prints source around current line.
- **`longlist`/`ll`** — prints whole function source.
- **`source EXPR`** — print source of an object (function/method).
- **`print`/`p EXPR`** — eval EXPR in frame and print.
- **`pp EXPR`** — pretty-print.
- **`args`/`a`** — print args of current frame from `co_varnames` and `f_locals`.
- **`locals`** (3.13+) — print all locals.
- **`globals`** (3.13+) — print all globals.
- **`!python_expr`** — execute as Python statement (`exec`).
- **`jump N`/`j`** — set `frame.f_lineno = N`. Only allowed at line events, not call/return. Skips/replays code.
- **`quit`/`q`** — raises `BdbQuit`.
- **`run [args]`/`restart`** — restart the program (raises `Restart`).
- **`debug EXPR`** — recursive debug: starts a *new* Pdb on the expression. So you can debug code from within pdb.
- **`alias`/`unalias`** — define/remove command aliases.
- **`whatis EXPR`** — prints `type(EXPR)`.
- **`help`/`h`** — auto-derived from docstrings.

The **`q` vs `quit` edge case**: `q` is short for `quit`, which raises `BdbQuit`. The Bdb run methods catch `BdbQuit` and return cleanly. But if you're inside a recursive `debug` session, `q` only quits the inner session — the outer Pdb is still active. Repeated `q` unwinds nested sessions.

## Conditional + Hit-Count

`condition N expr` sets `bp.cond = expr` for breakpoint N. The condition is compiled (cached) and evaluated on every hit.

`ignore N count` sets `bp.ignore = count`. The next `count` hits are skipped (effectively decrementing `ignore` each hit). Once `ignore` reaches 0, the breakpoint fires normally.

The two mechanisms compose: a conditional with `ignore 5` skips the first 5 condition-true hits.

```
(Pdb) b foo.py:10
Breakpoint 1 at foo.py:10
(Pdb) condition 1 i > 100
(Pdb) ignore 1 3
Will ignore next 3 crossings of breakpoint 1.
```

Now bp1 fires on the 4th hit where `i > 100`.

## Aliases

`alias [name [command]]` creates a user-defined macro. Without arguments, lists current aliases. With `name` only, prints that alias. With `name command`, defines.

```
(Pdb) alias pi for k in %1.__dict__.keys(): print("%1." + k, "=", %1.__dict__[k])
(Pdb) pi some_obj
some_obj.x = 1
some_obj.y = 2
```

Substitution: `%1`, `%2`, etc. are positional args. `%*` is all args.

Aliases are *not* persisted by default. The `.pdbrc` file (in cwd or `$HOME`) is read on Pdb startup and can contain commands (including `alias`), so you can persist aliases there:

```
# ~/.pdbrc
alias pi for k in %1.__dict__.keys(): print("%1." + k, "=", %1.__dict__[k])
alias ps for k in dir(%1): print(k)
```

This is the canonical way to customize pdb. ipdb and pdbpp also read `.pdbrc`.

## Watch Expressions

pdb itself has no `watch` command. The closest is `display`. ipdb adds `watch EXPR` which is a richer display (only prints when changed, with formatted diff). pdbpp adds true watchpoints with the `track` command.

True watchpoints (break when memory changes) require either:

- Hardware watchpoints (gdb on x86 has these via DR0-DR3 debug registers).
- Memory page protection tricks.

Python doesn't expose either. So Python-level "watchpoints" all reduce to "evaluate expression on every line and stop if it changed" — which is essentially conditional breakpoints with a side effect.

The performance cost is severe: every line evaluates every watch expression. A handful of watches multiply the per-line trace cost.

## Internals + Performance

Recap of the hot path:

1. CPython evaluates a bytecode.
2. If next bytecode is at a new line and `tstate->use_tracing` is set, call `c_tracefunc`.
3. `c_tracefunc` is `_PyEval_TraceCall` (in CPython internals), which calls the Python trace function.
4. Python trace function = `Bdb.trace_dispatch`.
5. `trace_dispatch` dispatches to `dispatch_line`, which calls `stop_here` and `break_here`.
6. `break_here` looks up `self.breaks[filename][lineno]`, runs `effective` to evaluate conditions.
7. If a stop is needed, `user_line` is called, which (in Pdb) calls `interaction`, which runs the cmdloop.
8. Otherwise, `dispatch_line` returns `self.trace_dispatch` so the next line in this frame also calls back.

Every step here is per-line. The dictionary lookups are O(1) but the Python-call overhead dominates.

For high-performance scenarios:

- **Don't use pdb.** Use a sampling profiler (py-spy, austin) that reads frame state from outside the process.
- **If you must trace, use sys.monitoring (3.12+)** to subscribe only to needed events.
- **Skip uninteresting modules** via `Pdb(skip=[...])` so their frames don't get per-line tracing.
- **Limit conditional breakpoints** — each one adds an `eval` per hit.

The fundamental architectural cost: pdb's "stop on breakpoint" model requires it to be informed about every line. The alternative — letting CPython hardware-trap on a specific bytecode — would require modifying bytecode in place (which CPython doesn't expose safely from Python). debugpy and PEP 669 inch closer to this but still operate within the trace function model.

## Common Internals Errors

- **`BdbQuit`** — raised when user types `q` or `quit`. Caught by `Bdb.run`, `Bdb.runeval`, etc., and translated into a clean exit. If you `pdb.set_trace()` from arbitrary code and the user types `q`, the exception propagates to your code — wrap in `try`/`except BdbQuit` if you want to continue without pdb (rare).

- **`*** AttributeError`** — when you reference a variable name in a pdb command that doesn't exist in the frame. Pdb tries `frame.f_locals[name]`, then `frame.f_globals[name]`, then `frame.f_builtins[name]`, then raises. The shape of the error is `AttributeError` rather than `NameError` because pdb wraps the lookup. Subtle but distinct.

- **`*** can't jump backwards into except block`** — `jump` has constraints. You can't jump into the middle of a try/except, into a `for` loop's body without going through the `for` statement, etc. The CPython JIT has a state machine for this and rejects invalid jumps.

- **`*** Ignored: function arguments`** — if you try to set a breakpoint via `b funcname` and the function has no source-line breakpoint location (e.g., a C function), pdb refuses.

- **"Hung pdb"** — happens when pdb is invoked from a subprocess whose stdin is not a tty. pdb's REPL reads from stdin; if stdin is closed or redirected, the loop gets EOF immediately and exits, looking like a no-op.

- **`pdb.set_trace()` does nothing in pytest** — pytest captures stdout/stderr/stdin by default. Use `pytest -s` to disable capture, or `pytest --pdb` to enter pdb on test failure.

## Idioms

- **`import pdb; pdb.set_trace()` vs `breakpoint()`**: prefer `breakpoint()` (3.7+). Concise, swappable via env var, easier to disable in production.

- **Global crash → post-mortem**: install `sys.excepthook` to drop into `pdb.post_mortem` on uncaught exceptions. Useful for development. Don't ship in production.

  ```python
  import sys, pdb
  def excepthook(*a):
      sys.__excepthook__(*a)
      pdb.post_mortem(a[2])
  sys.excepthook = excepthook
  ```

- **Use ipdb in interactive notebooks**: `from IPython import get_ipython; get_ipython().run_line_magic('pdb', 'on')` enables pdb on cell errors.

- **Use debugpy for editor integration**: VS Code's Python extension uses debugpy; PyCharm uses pydevd. Both surpass pdb for editor-mediated debugging.

- **Skip framework frames via skip_modules**: `Pdb(skip=['django.*', 'pytest.*'])`. Massively cleans up `where` output and avoids stepping into framework internals.

- **`.pdbrc` for muscle memory**: aliases and default settings. Common entries: `pp`, `args`, `locals`, custom inspection helpers.

- **`pp` for pretty-printing**: `pp my_dict` is far more readable than `p my_dict` for nested structures.

- **`!` prefix when names collide**: `! step = 5` assigns to local `step` (would otherwise be parsed as the step command). Equivalent: `step = 5` works most of the time, but `!` removes ambiguity.

- **`debug EXPR` to debug a single expression**: starts a new Pdb on the expression's evaluation. Useful for chasing a complex computation without restarting.

- **`commands N` for auto-actions**: rarely used but powerful — define commands to run automatically when bp N is hit. The gdb equivalent of "every break, do X."

- **Python `-m pdb script.py`**: runs the script under pdb from the start. Auto-restarts on `restart`. Drops into post-mortem on unhandled exceptions.

## See Also

- gdb (sheets/system/gdb.md) — native debugger; can attach to a running Python process and inspect the C stack and (with python helpers) the Python stack.
- lldb (sheets/system/lldb.md) — LLVM-based native debugger; macOS default; similar role to gdb.
- delve (sheets/system/delve.md) — Go debugger; similar source-level model but for Go binaries.
- python (sheets/languages/python.md) — Python language reference; pdb's host runtime.

## References

- [docs.python.org/3/library/pdb.html](https://docs.python.org/3/library/pdb.html) — official pdb reference.
- [docs.python.org/3/library/bdb.html](https://docs.python.org/3/library/bdb.html) — Bdb base class reference.
- [peps.python.org/pep-0553](https://peps.python.org/pep-0553/) — `breakpoint()` builtin.
- [peps.python.org/pep-0626](https://peps.python.org/pep-0626/) — Precise line numbers for debugging.
- [peps.python.org/pep-0657](https://peps.python.org/pep-0657/) — Include fine-grained error locations in tracebacks.
- [peps.python.org/pep-0669](https://peps.python.org/pep-0669/) — Low-impact monitoring (3.12+).
- CPython source: `Lib/pdb.py`, `Lib/bdb.py`, `Lib/cmd.py`.
- CPython source: `Python/ceval.c` — the eval loop, `call_trace`, `PyTrace_*` callbacks.
- CPython source: `Python/sysmodule.c` — `PyEval_SetTrace`, `PyEval_SetProfile`.
- CPython source: `Objects/frameobject.c`, `Objects/codeobject.c` — frame and code object internals.
- CPython source: `Objects/lnotab_notes.txt` — line number table format documentation.
- [docs.python.org/3/library/sys.html#sys.settrace](https://docs.python.org/3/library/sys.html#sys.settrace) — sys.settrace reference.
- [docs.python.org/3/library/sys.html#sys.setprofile](https://docs.python.org/3/library/sys.html#sys.setprofile) — sys.setprofile reference.
- [docs.python.org/3/library/sys.monitoring.html](https://docs.python.org/3/library/sys.monitoring.html) — PEP 669 sys.monitoring API.
- [pypi.org/project/ipdb](https://pypi.org/project/ipdb/) — IPython-flavored pdb.
- [pypi.org/project/pdbpp](https://pypi.org/project/pdbpp/) — pdb++ enhanced UI.
- [github.com/microsoft/debugpy](https://github.com/microsoft/debugpy) — DAP-compatible debugger.
- [pypi.org/project/web-pdb](https://pypi.org/project/web-pdb/) — browser-based pdb.
- [pypi.org/project/remote-pdb](https://pypi.org/project/remote-pdb/) — TCP-attached pdb.
- [pypi.org/project/pdb-attach](https://pypi.org/project/pdb-attach/) — signal-attached pdb.
- [py-spy](https://github.com/benfred/py-spy) — sampling profiler, no settrace overhead.
- [austin](https://github.com/P403n1x87/austin) — frame stack sampler.
