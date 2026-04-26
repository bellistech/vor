# Python Errors — CPython Runtime and Exception Internals

A deep dive into how CPython implements exceptions, from the C-level `PyExc_BaseException` struct through PEP 657 fine-grained tracebacks, exception groups, and the bytecode mechanisms that make `try`/`except` work. This is not a catalog of error messages — for that, see `sheets/troubleshooting/python-errors.md`. This document is about the machinery underneath.

## Setup

Python's primary error-signalling mechanism is the exception. Unlike C's return-code idiom or Go's explicit `error` interface, Python wires non-local control flow into the language: any expression may raise, and the interpreter unwinds the call stack until a matching `except` is found. The choice has consequences: every Python frame is potentially a re-entry point for an exception, every C extension function returning to Python must check for set errors, and every line of Python code carries the cost of being a possible exception source.

The `BaseException` hierarchy:

```text
BaseException
 +-- BaseExceptionGroup            (3.11+)
 +-- GeneratorExit
 +-- KeyboardInterrupt
 +-- SystemExit
 +-- Exception
      +-- ArithmeticError
      |    +-- FloatingPointError
      |    +-- OverflowError
      |    +-- ZeroDivisionError
      +-- AssertionError
      +-- AttributeError
      +-- BufferError
      +-- EOFError
      +-- ExceptionGroup            (3.11+)
      +-- ImportError
      |    +-- ModuleNotFoundError
      +-- LookupError
      |    +-- IndexError
      |    +-- KeyError
      +-- MemoryError
      +-- NameError
      |    +-- UnboundLocalError
      +-- OSError
      |    +-- BlockingIOError
      |    +-- ChildProcessError
      |    +-- ConnectionError
      |    |    +-- BrokenPipeError
      |    |    +-- ConnectionAbortedError
      |    |    +-- ConnectionRefusedError
      |    |    +-- ConnectionResetError
      |    +-- FileExistsError
      |    +-- FileNotFoundError
      |    +-- InterruptedError
      |    +-- IsADirectoryError
      |    +-- NotADirectoryError
      |    +-- PermissionError
      |    +-- ProcessLookupError
      |    +-- TimeoutError
      +-- ReferenceError
      +-- RuntimeError
      |    +-- NotImplementedError
      |    +-- RecursionError
      +-- StopAsyncIteration
      +-- StopIteration
      +-- SyntaxError
      |    +-- IndentationError
      |         +-- TabError
      +-- SystemError
      +-- TypeError
      +-- ValueError
      |    +-- UnicodeError
      |         +-- UnicodeDecodeError
      |         +-- UnicodeEncodeError
      |         +-- UnicodeTranslateError
      +-- Warning
           +-- DeprecationWarning
           +-- PendingDeprecationWarning
           +-- RuntimeWarning
           +-- SyntaxWarning
           +-- UserWarning
           +-- FutureWarning
           +-- ImportWarning
           +-- UnicodeWarning
           +-- BytesWarning
           +-- ResourceWarning
```

`BaseException` (not `Exception`) is the actual root. `KeyboardInterrupt` and `SystemExit` are deliberately outside `Exception` — a bare `except Exception` won't catch Ctrl+C, which is what most code wants.

## CPython Exception Object Layout

`PyExc_BaseException` is defined in `Objects/exceptions.c`. The C struct underneath every Python exception is `PyBaseExceptionObject`:

```c
typedef struct {
    PyObject_HEAD
    PyObject *dict;
    PyObject *args;
    PyObject *notes;          /* PEP 678: __notes__ list */
    PyObject *traceback;
    PyObject *context;
    PyObject *cause;
    char suppress_context;
} PyBaseExceptionObject;
```

The five Python-visible attributes:

- `args` — tuple passed to the constructor; `str(exc)` typically formats from this.
- `__traceback__` — the linked traceback chain; `None` until the exception is raised.
- `__context__` — implicit chain. Set automatically when an exception is raised inside an `except` or `finally` block.
- `__cause__` — explicit chain. Set by `raise X from Y`. Setting `__cause__` also sets `__suppress_context__` to `True` so the implicit context isn't printed.
- `__notes__` — list of strings added by `exc.add_note(s)` (PEP 678, 3.11+). Printed below the traceback.

`PyErr_SetString(PyExc_TypeError, "msg")` constructs a `TypeError("msg")` and stores it on the thread state without raising it through the C call stack. C extension code returns `NULL` (or `-1` for int-returning functions) and the interpreter checks `PyErr_Occurred()` on its way back.

The `__suppress_context__` flag exists because of a UX issue: when you `raise X from Y`, you want only Y in the chain, not whatever exception happened to be active. Without the flag, both would be printed.

## The Interpreter's Exception State

CPython doesn't use C++ exceptions or longjmp. Instead, each thread has a `PyThreadState` with three slots: `curexc_type`, `curexc_value`, `curexc_traceback`. (Pre-3.11 these were three separate slots; 3.12+ collapses them into a single `current_exception` field that points to a normalized exception object.)

Setting and fetching:

```c
/* Set */
PyErr_SetString(PyExc_ValueError, "bad input");
PyErr_SetObject(type, value);
PyErr_Format(PyExc_TypeError, "expected %s, got %s", a, b);

/* Check */
if (PyErr_Occurred()) {
    /* an exception is currently set */
}

/* Fetch (clears tstate) */
PyObject *type, *value, *tb;
PyErr_Fetch(&type, &value, &tb);

/* Restore (sets tstate) */
PyErr_Restore(type, value, tb);

/* Normalize: convert (type, args) pair into instance */
PyErr_NormalizeException(&type, &value, &tb);
```

`PyErr_Fetch` is "borrow ownership and clear the slot". `PyErr_Restore` is the inverse. `PyErr_Occurred` is a cheap pointer comparison and is called millions of times per second on a busy interpreter — every C function returning to Python checks it.

The "lazy normalization" trick: when you `raise ValueError("x")` from C, the interpreter may store the type and a tuple `("x",)` separately, deferring the actual `ValueError("x")` instance creation until something needs it. `PyErr_NormalizeException` forces the instance creation. This is invisible from Python.

## Exception Chaining (PEP 3134)

Two kinds of chaining:

**Implicit (`__context__`):** when an exception is raised while another is being handled, the new exception's `__context__` is set to the active one.

```python
try:
    1/0
except ZeroDivisionError:
    raise ValueError("oops")  # __context__ is ZeroDivisionError
```

Output:

```text
During handling of the above exception, another exception occurred:

Traceback (most recent call last):
  File ..., line N, in <module>
    raise ValueError("oops")
ValueError: oops
```

**Explicit (`__cause__`):** `raise X from Y` sets `X.__cause__ = Y` and `X.__suppress_context__ = True`.

```python
try:
    1/0
except ZeroDivisionError as e:
    raise ValueError("oops") from e
```

Output:

```text
The above exception was the direct cause of the following exception:

Traceback (most recent call last):
  File ..., line N, in <module>
    raise ValueError("oops") from e
ValueError: oops
```

The "direct cause" framing tells the user the second exception was *meant* to wrap the first; "during handling" means they probably wanted only the first and have a second bug.

`raise X from None` is the only way to suppress chaining entirely — useful when re-raising a translated exception from an unrelated low-level error.

## Traceback Object Internals

A traceback is a singly-linked list of `PyTracebackObject`s, each pointing to a frame:

```c
typedef struct _traceback {
    PyObject_HEAD
    struct _traceback *tb_next;
    struct _frame *tb_frame;
    int tb_lasti;        /* bytecode index where exception occurred */
    int tb_lineno;       /* line number derived from tb_lasti */
} PyTracebackObject;
```

The chain is built as the exception unwinds. When `gen_op_RAISE` (or any opcode that propagates an exception) discovers that no handler in the current frame matches, it:

1. Calls `PyTraceBack_Here(frame)` which prepends a new `tb` node pointing to the current frame.
2. Pops the frame, returning to the caller.
3. The caller's loop sees an active exception and repeats.

`tb_frame` keeps the frame alive after the function returned. This is a major source of memory retention: a long-lived caught exception holds tracebacks, which hold frames, which hold `f_locals` (everything in the function's namespace).

`traceback.format_exception(exc)` walks `__cause__`/`__context__` first, then walks `__traceback__.tb_next` chain to build the "most recent call last" output.

## PEP 657 (3.11+)

Pre-3.11 traceback output:

```text
File "x.py", line 5, in foo
    return obj.attr.method()[0]
AttributeError: 'NoneType' object has no attribute 'method'
```

You couldn't tell which `.attr` or `.method` failed without re-running with print statements. PEP 657 added column-anchored carets:

```text
File "x.py", line 5, in foo
    return obj.attr.method()[0]
           ^^^^^^^^^^^^^^^^^
AttributeError: 'NoneType' object has no attribute 'method'
```

Implementation:

- `co_positions` table on each `PyCodeObject` maps each bytecode instruction to `(start_line, end_line, start_col, end_col)`.
- The table is varint-encoded; only changes between adjacent instructions are stored, so the overhead is small.
- `tb_lasti` indexes into this table, giving exact source span.
- `dis.dis(func, show_caches=True)` shows the position info; `code.co_positions()` is the runtime API.

The cost: each `.pyc` file is slightly larger. The benefit: tracebacks finally point to the actual failing subexpression. This is one of the largest user-facing improvements in 3.11.

## The GIL + Exception Propagation

C extensions hold the GIL while running unless they explicitly release it via `Py_BEGIN_ALLOW_THREADS` / `Py_END_ALLOW_THREADS`. Inside that block, the C code must not call any Python API and must not access Python objects.

When a C function detects an error:

1. Call `PyErr_SetString(type, msg)` (or related).
2. Return `NULL` (for `PyObject *` returners) or `-1` (for `int` returners).
3. The interpreter's eval loop, on its next iteration after the C call returns, sees `PyErr_Occurred()` is true and unwinds.

There is no way for an exception to "cross" a thread without explicit handing-off. If a worker thread raises, the exception terminates that thread (printing to stderr via `sys.unraisablehook`) but does not propagate to the main thread. This is why `threading.Thread.run` swallows exceptions by default and `concurrent.futures` exists — `Future.result()` re-raises in the calling thread.

`sys.excepthook` handles uncaught exceptions in the main thread; `threading.excepthook` (3.8+) handles them in worker threads; `sys.unraisablehook` handles ones the runtime can't propagate (e.g., in `__del__`).

## The "Did You Mean" Hints (3.10+)

```python
>>> import collectons
ModuleNotFoundError: No module named 'collectons'. Did you mean: 'collections'?

>>> {}.iteritems()
AttributeError: 'dict' object has no attribute 'iteritems'. Did you mean: 'items'?

>>> def foo():
...     prnt("hi")
... 
>>> foo()
NameError: name 'prnt' is not defined. Did you mean: 'print'?
```

Mechanism:

- `Lib/_suggestions.py` (or the C equivalent in 3.12+: `Python/suggestions.c`) computes Levenshtein-distance edit distance between the unknown name and candidates.
- Candidates for `NameError`: builtins + locals + globals + nonlocals.
- Candidates for `AttributeError`: `dir(obj)` (filtered).
- Candidates for `ImportError`/`ModuleNotFoundError`: `sys.modules` + `sys.path`-discovered modules.
- A maximum edit distance of `max(2, len(name) // 2)` is used to filter.

The hint is added as a suffix to the exception's args during normalization. It's always a guess — the runtime can't know if `prnt` was meant to be `print` or your own helper.

## Pickle / Unpickle Exception

Exceptions are picklable because they implement `__reduce__`:

```python
def __reduce__(self):
    return (type(self), self.args)
```

When unpickled, the receiver imports the same exception class and constructs it with `args`. This is why exceptions raised in a `multiprocessing.Pool` worker can be re-raised in the parent: the worker pickles the exception over the pipe; the parent unpickles and re-raises.

The catch: the exception class must be importable on the receiver. A custom exception defined in `__main__` of the worker won't unpickle in a different parent. `concurrent.futures.ProcessPoolExecutor` works around this by wrapping unpicklable exceptions in `_ExceptionWrapper`.

`__traceback__` is *not* picklable — tracebacks contain frames which contain locals which may be arbitrarily non-picklable. `multiprocessing` formats the traceback as a string and reconstructs as `RemoteTraceback`.

## Bytecode + Try/Except

Pre-3.11 used a try-block stack. Each `SETUP_EXCEPT` / `SETUP_FINALLY` opcode pushed an entry; a raise looked up the top entry, jumped to the handler, and `POP_BLOCK` cleaned up.

3.11+ replaced this with the **exception table** stored in `co_exceptiontable`. Each entry is `(start, end, target, depth, lasti)`:

- `start`, `end` — bytecode range covered.
- `target` — handler's bytecode address.
- `depth` — value-stack depth at handler entry.
- `lasti` — push `tb_lasti` for re-raising.

The table is a sorted list. On exception, the runtime binary-searches for the innermost entry covering the current `lasti`. No per-instruction stack maintenance. This is the **zero-cost exception** model: the no-exception path executes zero extra instructions.

```python
import dis
def f():
    try:
        return 1/0
    except ZeroDivisionError:
        return 0
dis.dis(f, show_caches=False)
```

Pre-3.11 you'd see `SETUP_FINALLY` / `POP_BLOCK`. In 3.11+ those vanish; you'll see only the actual logic, plus a separate `ExceptionTable` printout.

The benefit is real: try/except has been ~10% faster overall in 3.11 due to this change alone.

## Asyncio Exception Patterns

`asyncio.CancelledError` was promoted from `Exception` to `BaseException` in 3.8. The reason: prior code did `try: ... except Exception:` to catch real errors, which inadvertently swallowed cancellations. Promoting to `BaseException` means a normal `except Exception` lets cancellation propagate.

Propagation through `await`:

```python
async def inner():
    raise ValueError("inner")

async def outer():
    await inner()  # ValueError propagates here

asyncio.run(outer())  # ValueError raised
```

`asyncio.gather(coro1, coro2, ...)`:

- Default: first exception is raised in the gather's caller; other coroutines continue and their results/exceptions are silently swallowed.
- `return_exceptions=True`: gather returns a list with exceptions and results mixed; the caller decides how to handle.

`TaskGroup` (3.11+, PEP 654):

```python
async with asyncio.TaskGroup() as tg:
    tg.create_task(inner1())
    tg.create_task(inner2())
# If both raised, you get an ExceptionGroup with both
```

If one task raises, the group cancels the others and waits for them; all their exceptions are aggregated into a single `ExceptionGroup`. This is the modern replacement for `gather`.

## PEP 654 Exception Groups (3.11+)

`ExceptionGroup` is an exception that contains other exceptions:

```python
eg = ExceptionGroup("multiple errors", [
    ValueError("a"),
    KeyError("b"),
    OSError("c"),
])
```

The split/subgroup API:

```python
matched, rest = eg.split(ValueError)
# matched is ExceptionGroup with ValueError(s) only
# rest is ExceptionGroup with the others, or None

only_oserror = eg.subgroup(OSError)
# subset; doesn't return the leftover
```

`try: ... except*` walks the group:

```python
try:
    raise ExceptionGroup("e", [ValueError("v"), KeyError("k")])
except* ValueError as eg_v:
    print("got value error(s)")
except* KeyError as eg_k:
    print("got key error(s)")
```

Each handler receives only the matching subgroup. Multiple handlers can fire (unlike regular `except`).

`BaseExceptionGroup` is the parent; if all contained exceptions are `Exception`, you get `ExceptionGroup` (a subclass of both `BaseExceptionGroup` and `Exception`); otherwise plain `BaseExceptionGroup`.

## Garbage Collection + Exception Cycles

The retention pattern:

```python
exc.__traceback__ -> tb -> frame -> frame.f_locals -> ... -> exc (cycle)
```

If `exc` is held in a long-lived structure (e.g., a global error log), this cycle keeps the frame and all locals alive. Python's cycle collector will eventually free it, but until then memory is held.

Mitigations:

```python
try:
    risky()
except Exception as e:
    log.error("failed", exc_info=True)
    # Optionally:
    # del e   <- inside except block, e is unbound at end anyway
    save_exc = e.with_traceback(None)  # detach traceback
    archive.append(save_exc)
```

In a `try: ... except E as e:` block, `e` is implicitly `del`'d at the end of the except. This was added precisely to break the cycle. So:

```python
try:
    risky()
except Exception as e:
    handle(e)
# Here, e is no longer bound — del happened automatically
```

But:

```python
try:
    risky()
except Exception as e:
    self.last_error = e  # cycle until self is freed
```

## Frame Lifetime + sys.exc_info

`sys.exc_info()` returns `(type, value, traceback)` of the currently-handled exception. Outside an `except` block, returns `(None, None, None)`.

In Python 3, `sys.exc_info()` is per-coroutine: each generator and each task maintains its own exception context, so awaiting another coroutine doesn't disturb the caller's exception state. Pre-3.7 this was global per thread and caused subtle bugs.

`sys.exception()` (3.11+) returns just the exception object — equivalent to `sys.exc_info()[1]` but more direct.

PEP 3134's implicit chaining works because the runtime, on `RAISE`, peeks at `sys.exception()` and stashes it as `__context__` before propagating.

## Logging + Exceptions

```python
import logging
log = logging.getLogger(__name__)

try:
    risky()
except Exception:
    log.exception("operation failed")
    # equivalent to: log.error("...", exc_info=True)
```

`log.exception` calls `traceback.format_exception(*sys.exc_info())` and includes the formatted traceback in the log record. This is why you should call it inside the `except` — outside, `sys.exc_info()` returns `(None, None, None)` and there's no traceback to log.

`exc_info=True` works on all log methods. `exc_info=exc` (passing an exception) lets you log an exception you have a handle to without it being currently raised.

The threading concern: if you have a logger writing to a file with no lock, two threads logging exceptions concurrently can interleave their multi-line tracebacks. The stdlib `StreamHandler` uses a `RLock` to prevent this.

## C Extension Errors

The `PyErr_SetFromXxx` family translates C-level errors into Python exceptions:

- `PyErr_SetFromErrno(type)` — read C `errno`, set the matching exception. `errno=ENOENT` becomes `FileNotFoundError`.
- `PyErr_SetFromErrnoWithFilename(type, filename)` — same, but also sets `.filename`.
- `PyErr_SetFromWindowsErr(int)` — Windows GetLastError variant.
- `PyErr_SetExcFromWindowsErrWithFilename(type, ierr, filename)` — combo.
- `PyErr_NoMemory()` — set MemoryError.
- `PyErr_BadInternalCall()` — SystemError, used when a C function detects misuse by another C function.

The C function returns NULL/-1 to signal "exception is set". The eval loop unwinds.

Common pattern:

```c
PyObject *
my_open(PyObject *self, PyObject *args)
{
    const char *path;
    if (!PyArg_ParseTuple(args, "s", &path)) {
        return NULL;
    }
    int fd = open(path, O_RDONLY);
    if (fd < 0) {
        return PyErr_SetFromErrnoWithFilename(PyExc_OSError, path);
    }
    return PyLong_FromLong(fd);
}
```

If you're calling another Python API (e.g., `PyObject_GetItem`) and want to catch its exception:

```c
PyObject *result = PyObject_GetItem(d, key);
if (result == NULL) {
    if (PyErr_ExceptionMatches(PyExc_KeyError)) {
        PyErr_Clear();
        /* handle missing key */
    } else {
        return NULL;  /* propagate */
    }
}
```

## The errno.h Mapping

Since 3.3, OSError has subclass-per-errno:

```text
EACCES, EPERM            -> PermissionError
EAGAIN, EALREADY,
EWOULDBLOCK, EINPROGRESS -> BlockingIOError
ECHILD                   -> ChildProcessError
ECONNABORTED             -> ConnectionAbortedError
ECONNREFUSED             -> ConnectionRefusedError
ECONNRESET               -> ConnectionResetError
EEXIST                   -> FileExistsError
EISDIR                   -> IsADirectoryError
ENOENT                   -> FileNotFoundError
ENOTDIR                  -> NotADirectoryError
EINTR                    -> InterruptedError
ESRCH                    -> ProcessLookupError
ETIMEDOUT                -> TimeoutError
EPIPE, ESHUTDOWN         -> BrokenPipeError
```

The mapping happens automatically in `PyErr_SetFromErrno`. So:

```python
try:
    open("/nonexistent")
except FileNotFoundError as e:
    print(e.errno)  # 2 (ENOENT)
```

You can still catch `OSError` to handle all cases, or use the specific subclass for fine-grained handling. Don't compare `e.errno == errno.ENOENT` manually — use the subclass.

## Memory Errors

`MemoryError` is raised when allocation fails. CPython has multiple allocators:

- `PyMem_Malloc` — for small objects, uses pymalloc (an arena allocator with block recycling).
- `PyObject_Malloc` — same as PyMem_Malloc by default in Python.
- `PyMem_RawMalloc` — direct system malloc, doesn't hold the GIL.

When pymalloc's arena pool runs out and the underlying `mmap` fails, `MemoryError` is raised. On Linux, `mmap` rarely fails because of overcommit; the OOM killer will fire first. So `MemoryError` in Python on Linux usually means you've hit `ulimit -v` or you're in a container with a memory cgroup limit.

Soft vs hard memory pressure:

- **Soft:** `MemoryError` raised at allocation site. Handler can free things and retry.
- **Hard:** OS kills the process. No Python-level recovery.

There's no `gc.callbacks` for memory pressure in CPython; you'd have to poll `psutil.Process().memory_info().rss` and free manually. Some tools use `resource.setrlimit(resource.RLIMIT_AS, ...)` to convert hard kills to `MemoryError`.

## Recursion Limit

```python
import sys
print(sys.getrecursionlimit())  # 1000 default
sys.setrecursionlimit(10000)
```

The limit exists because each Python frame allocates a C stack frame, and each C frame consumes ~500-1000 bytes. The default 8 MB thread stack on Linux can hold ~8000 Python frames before SIGSEGV. The 1000 limit gives headroom.

When the limit is hit:

```text
RecursionError: maximum recursion depth exceeded
```

`RecursionError` is a subclass of `RuntimeError` so old code catching `RuntimeError` still works.

The check is per-frame in the C eval loop (`tstate->py_recursion_remaining` in 3.12+, `tstate->recursion_remaining` earlier). Each call decrements; each return increments. Cython-compiled or C-extension code that creates Python frames must call `Py_EnterRecursiveCall` / `Py_LeaveRecursiveCall`.

Python 3.11 reduced per-frame overhead by inlining frames; this slightly increases the practical depth at default limit, though the `setrecursionlimit` value didn't change.

For deep recursion, convert to iteration using an explicit stack. Tail-call optimization is not in CPython and is unlikely to be added — Guido's stance is that explicit iteration is more debuggable.

## PEP 626 — Code Object Line Numbers

Pre-3.10, the `co_lnotab` table mapped bytecode offset to line number, but the resolution was per-line. If multiple instructions came from one source line, the table compressed them. This was efficient but lost information.

PEP 626 introduced `co_lines()`:

```python
def f():
    x = 1
    return x

for start, end, line in f.__code__.co_lines():
    print(start, end, line)
```

Each entry is a contiguous bytecode range and its source line, including end-instruction. This means tracebacks now point to the correct line even when the failing instruction is the last of a multi-instruction line.

A subtle change: pre-3.10 a traceback in:

```python
result = (foo() +
          bar())
```

Would point to the first line. 3.10+ points to the line containing the actual call. PEP 657 (3.11+) extends this further with column info.

## Common Library Exception Patterns

**Pydantic v2 ValidationError:**

```python
from pydantic import BaseModel, ValidationError

class User(BaseModel):
    name: str
    age: int

try:
    User(name=123, age="abc")
except ValidationError as e:
    for err in e.errors():
        print(err["loc"], err["msg"])
```

`ValidationError` collects all field errors in one exception; the `.errors()` returns a list. This is similar in spirit to `ExceptionGroup` but predates it.

**SQLAlchemy IntegrityError:**

```python
from sqlalchemy.exc import IntegrityError

try:
    session.commit()
except IntegrityError as e:
    print(e.orig)  # the underlying DB-API exception
    print(e.statement)
    print(e.params)
```

SQLAlchemy wraps DB-API-2.0 exceptions in its own hierarchy. `.orig` is the database driver's original exception (e.g., `psycopg.errors.UniqueViolation`).

**requests/httpx hierarchy:**

```text
requests.exceptions.RequestException (base)
 +-- ConnectionError
 |    +-- ConnectTimeout
 |    +-- ProxyError
 |    +-- SSLError
 +-- HTTPError      <- raised only by raise_for_status()
 +-- Timeout
 |    +-- ConnectTimeout
 |    +-- ReadTimeout
 +-- TooManyRedirects
 +-- URLRequired
 +-- MissingSchema
 +-- InvalidSchema
 +-- InvalidURL
```

Catching `RequestException` covers all network-level failures. `HTTPError` (4xx/5xx) is *not* raised automatically — call `response.raise_for_status()`.

httpx mirrors this with `httpx.HTTPError` as base and `httpx.RequestError`, `httpx.HTTPStatusError`, etc.

## Exception Performance

The cost model:

- **try/except (no exception):** essentially free in 3.11+ (zero-cost). Pre-3.11 was a small constant per `try`.
- **raise:** allocates exception object, walks frames building traceback. Each frame in the traceback is a `PyTracebackObject` allocation. Cost is O(stack depth).
- **except + match:** the exception's MRO is walked to check `isinstance`. Linear in inheritance depth.
- **re-raise (`raise`):** uses the existing exception, no new traceback allocation, but `tb_lasti` may be appended.

The advice "don't use exceptions for control flow" was real pre-3.11. Now the no-exception path is genuinely free, so:

```python
# Slow if 'a' often missing
try:
    return d[a]
except KeyError:
    return default

# Faster if 'a' often missing (avoid raising)
return d.get(a, default)
```

The "slow" version is fast when `a` is usually present; the `dict.get` check itself has overhead. Profile before optimizing.

## Best Practices

**Most-specific-exception-first:**

```python
try:
    ...
except FileNotFoundError:
    ...
except OSError:
    ...
except Exception:
    ...
```

If `FileNotFoundError` came after `OSError`, it would never match because `OSError` catches its subclass.

**Never bare `except`:**

```python
# WRONG: catches BaseException, including Ctrl+C and SystemExit
try:
    ...
except:
    ...

# Right
try:
    ...
except Exception:
    ...
```

**Never catch + ignore without logging:**

```python
# WRONG: silent failure
try:
    risky()
except Exception:
    pass

# Right
try:
    risky()
except Exception:
    log.exception("risky failed")
```

**Prefer narrow scopes:**

```python
# WRONG: 50 lines under one try
try:
    a()
    b()
    c()
    ...
except Exception:
    handle()

# Right: only the call that can fail
result = a()
try:
    val = parse(result)
except ValueError:
    val = default
b(val)
c(val)
```

**Use exception groups for async/concurrent code:**

```python
async with asyncio.TaskGroup() as tg:
    tg.create_task(work_a())
    tg.create_task(work_b())
# All errors aggregated; partial successes lost. Use return_exceptions
# style only when you specifically need to inspect each.
```

**Preserve chains when translating:**

```python
try:
    raw = parse(data)
except json.JSONDecodeError as e:
    raise MyAppError("invalid input") from e
```

Always `from e` (or `from None` to suppress). Bare `raise MyAppError(...)` inside `except` still chains via `__context__`, which is fine, but explicit `from` is clearer intent.

**Don't over-broaden:**

```python
# WRONG: catches AttributeError from bug in cleanup, hides it
try:
    obj.cleanup()
except Exception:
    pass

# Right: catch only the exception cleanup() is documented to raise
try:
    obj.cleanup()
except OSError:
    pass
```

**Use `add_note` for context (3.11+):**

```python
try:
    process(item)
except Exception as e:
    e.add_note(f"while processing {item.id}")
    raise
```

Cleaner than wrapping in a new exception class when you just want to attach context.

## Frame Object Internals

The `PyFrameObject` (Include/cpython/frameobject.h) is the runtime stack frame for each Python function call. Key fields used in exception handling:

```c
typedef struct _frame {
    PyObject_VAR_HEAD
    struct _frame *f_back;      /* previous frame (caller) — frame chain */
    PyCodeObject  *f_code;      /* compiled code object — bytecode */
    PyObject      *f_builtins;  /* builtin namespace */
    PyObject      *f_globals;   /* module-level namespace */
    PyObject      *f_locals;    /* local variables (when accessed via locals()) */
    PyObject     **f_valuestack; /* the value stack used by bytecode */
    PyObject      *f_trace;     /* trace function (set by sys.settrace) */
    char           f_trace_lines;
    char           f_trace_opcodes;
    int            f_lasti;     /* index of last attempted instruction (in bytecode) */
    int            f_lineno;    /* current line number (computed from f_lasti via line table) */
    int            f_iblock;    /* depth of try-block / loop-block stack */
} PyFrameObject;
```

When an exception unwinds, the interpreter walks `f_back` building the traceback. Each `PyTracebackObject` holds:

```c
typedef struct _traceback {
    PyObject_HEAD
    struct _traceback *tb_next;
    PyFrameObject     *tb_frame;
    int                tb_lasti;
    int                tb_lineno;
} PyTracebackObject;
```

## Bytecode-Level Try/Except (Python 3.10 and Earlier)

Pre-3.11, try/except was implemented with a block-stack via SETUP_EXCEPT and POP_BLOCK opcodes:

```python
# Python source:
try:
    risky()
except ValueError:
    handle()

# Bytecode (3.10):
  2           0 SETUP_FINALLY            14 (to 16)
  3           2 LOAD_GLOBAL              0 (risky)
              4 CALL_FUNCTION            0
              6 POP_TOP
              8 POP_BLOCK
             10 JUMP_FORWARD            22 (to 34)
  4     >>   16 DUP_TOP
             18 LOAD_GLOBAL              1 (ValueError)
             20 JUMP_IF_NOT_EXC_MATCH    32
             22 POP_TOP
             24 POP_TOP
             26 POP_TOP
  5          28 LOAD_GLOBAL              2 (handle)
             30 CALL_FUNCTION            0
             32 POP_EXCEPT
             34 LOAD_CONST               0 (None)
             36 RETURN_VALUE
```

Each try block pushed a frame onto the `f_iblock` stack; exception unwind popped to find handlers.

## Bytecode-Level Try/Except (Python 3.11+)

Python 3.11 replaced the block-stack with a static **exception table** stored alongside bytecode:

```python
# Python source same as above

# Bytecode (3.11+):
  2           0 RESUME                   0
  3           2 LOAD_GLOBAL              1 (NULL + risky)
             14 PRECALL                  0
             18 CALL                     0
             28 POP_TOP
             30 LOAD_CONST               0 (None)
             32 RETURN_VALUE
  4     >>   34 PUSH_EXC_INFO
             36 LOAD_GLOBAL              2 (ValueError)
             48 CHECK_EXC_MATCH
             50 POP_JUMP_FORWARD_IF_FALSE 14 (to 80)
             52 POP_TOP
  5          54 LOAD_GLOBAL              5 (NULL + handle)
             66 PRECALL                  0
             70 CALL                     0
             80 POP_TOP
             82 POP_EXCEPT
             84 LOAD_CONST               0 (None)
             86 RETURN_VALUE

# Exception table (compact format, see Python/compile.c):
#   start_offset, end_offset, target, depth, lasti
#       2          30          34       0      0
```

The table is parsed when an exception is raised: find the smallest range containing the current `f_lasti` and jump to the handler. **Zero-cost on the no-exception path** — there's no SETUP_EXCEPT to execute.

## sys.settrace at the C Level

The `sys.settrace(fn)` API hooks into `tstate->c_tracefunc` and `tstate->use_tracing`:

```c
// In ceval.c, the eval loop checks tstate->use_tracing before each opcode dispatch:
if (cframe.use_tracing) {
    if (call_trace_protected(tstate->c_tracefunc, tstate->c_traceobj,
                             tstate, frame, &instr_prev, PyTrace_LINE, Py_None) < 0) {
        goto error;
    }
}
```

`tstate->use_tracing` is a bitfield combining global trace state and per-frame `f_trace_lines` / `f_trace_opcodes` flags. The trace function is called with `(frame, event, arg)` where `event` is one of:

- `'call'` — entering a function
- `'line'` — about to execute a new line (the most common — used by pdb breakpoints)
- `'return'` — function returning
- `'exception'` — exception raised
- `'opcode'` — about to execute an instruction (only if `f_trace_opcodes` set)

The performance cost is significant: tracing slows execution by ~10x, which is why pdb is sluggish during step-through.

## PEP 657 Fine-Grained Tracebacks

Python 3.11 introduced position-aware tracebacks. The code object now stores a `co_positions` table mapping each instruction to (start_line, end_line, start_col, end_col). When formatting a traceback:

```python
def f():
    a = obj.x.y.z
    #        ^   ← traceback caret points here

# Pre-3.11: just "line 2, in f"
# 3.11+: shows the carets:
#   File "X.py", line 2, in f
#     a = obj.x.y.z
#         ~~~~~^^^^
```

The `co_positions` table is encoded in `PyCode_LinesIterator` via a varint sequence; each entry is the delta from the previous instruction.

## PEP 654 Exception Groups (3.11+)

Multiple unrelated exceptions surfaced together (typical in async / concurrent code):

```python
async def process_all(urls):
    async with asyncio.TaskGroup() as tg:
        for url in urls:
            tg.create_task(fetch(url))

# If 3 of N tasks raise different exceptions, raises:
#   ExceptionGroup("unhandled errors in a TaskGroup", [
#     HTTPError("404 for url1"),
#     TimeoutError("url2 took too long"),
#     ConnectionError("url3 reset"),
#   ])

# Pattern matching with except* (PEP 654):
try:
    asyncio.run(process_all(urls))
except* HTTPError as eg:
    for exc in eg.exceptions:
        log.warning("HTTP error: %s", exc)
except* (TimeoutError, ConnectionError) as eg:
    for exc in eg.exceptions:
        log.error("Network error: %s", exc)
```

`except* X` matches any exception in the group whose type is `X` (or descended); other exceptions in the group are re-raised in a new ExceptionGroup. The `.split()` and `.subgroup()` methods on ExceptionGroup let you partition programmatically.

## PEP 626 Code Object Line Tables

Pre-3.10: `co_lnotab` was a sequence of `(byte_delta, line_delta)` pairs.
3.10+: `co_linetable` uses a more compact varint-based encoding with negative deltas and "no line" markers (for compiler-generated bytecode that has no source line).

The `co_lines()` generator iterates entries:

```python
import dis
for start, end, line in code.co_lines():
    print(f"bytecode {start:4}-{end:4} → source line {line}")
```

This enables PEP 657's column-aware tracebacks: each (byte_offset, line) pair is augmented with column info via `co_positions()`.

## CPython Memory Model + Exceptions

Exceptions are heap-allocated PyObjects. Reference cycles are common because:

```text
exc.__traceback__ → frame
frame.f_locals    → all local vars
local "exc" var   → exc itself  ← cycle!
```

CPython's reference-counting GC won't free this; the cyclic GC eventually collects it. Best practice: explicitly `del exc` in the handler or use `try/except/else/finally` to ensure the local goes out of scope.

The `traceback` module's `format_exception` walks `__traceback__` allocating new strings per frame; for a deep stack this can be expensive (10s of µs per frame).

## Library Exception Patterns Catalog

### Pydantic v2 ValidationError

```python
from pydantic import BaseModel, ValidationError

class User(BaseModel):
    name: str
    age: int

try:
    User(name="Alice", age="not a number")
except ValidationError as e:
    # e.errors() returns a list of dicts, each with type/loc/msg/input
    for err in e.errors():
        print(f"{err['loc']}: {err['msg']} (input: {err['input']!r})")
    # e.json() for JSON-serializable
    # str(e) for human-readable multi-line summary
```

### asyncio TaskGroup error aggregation

Exceptions from spawned tasks are gathered into ExceptionGroup. CancelledError suppression: a CancelledError in the body of TaskGroup propagates to siblings via cancel scope.

### contextlib.ExitStack exception aggregation

```python
with contextlib.ExitStack() as stack:
    files = [stack.enter_context(open(p)) for p in paths]
    # If any file's __exit__ raises during cleanup:
    # ExitStack chains the new exception to __context__ but doesn't aggregate.
    # In 3.11+, you can use contextlib.aclosing for async with similar semantics.
```

### sqlalchemy DetachedInstanceError

```python
sqlalchemy.orm.exc.DetachedInstanceError: Instance <Foo> is not bound to a Session
# Cause: object accessed after its session was closed; lazy-loaded attribute traversal
# Fix: eager-load (joinedload, selectinload) or keep session open longer
```

### requests SSLError chain

```text
requests.exceptions.SSLError: HTTPSConnectionPool(host='example.com', port=443):
  Max retries exceeded with url: /api/...
  (Caused by SSLError(SSLCertVerificationError(1, '[SSL: CERTIFICATE_VERIFY_FAILED] ...')))

# The .__cause__ chain: requests.exceptions.SSLError ← urllib3 SSLError ← ssl.SSLCertVerificationError
# To inspect:
try:
    requests.get(...)
except requests.exceptions.SSLError as e:
    while e:
        print(type(e).__name__, str(e))
        e = e.__cause__ or e.__context__
```

## References

- Python Language Reference, "Exceptions": https://docs.python.org/3/reference/executionmodel.html#exceptions
- CPython source `Objects/exceptions.c`: https://github.com/python/cpython/blob/main/Objects/exceptions.c
- CPython source `Python/ceval.c` (eval loop, exception handling)
- PEP 3134, "Exception Chaining and Embedded Tracebacks" (Yee, 2005)
- PEP 409, "Suppressing exception context" (Stinner/Coghlan, 2012)
- PEP 415, "Implement context suppression with `raise ... from None`" (Storchaka, 2012)
- PEP 654, "Exception Groups and except*" (Shaheed, 2021)
- PEP 657, "Include Fine Grained Error Locations in Tracebacks" (Galindo, 2021)
- PEP 678, "Enriching Exceptions with Notes" (Wieland, 2022)
- PEP 626, "Precise line numbers for debugging and other tools" (Shannon, 2020)
- "Python Language Summit 2022: Faster CPython" (Shannon)
- Brett Cannon, "Why Python Exceptions Are Cool" blog series
- `traceback` module documentation
- `sys.exc_info`, `sys.exception`, `sys.unraisablehook` documentation
- Python C API "Exception Handling": https://docs.python.org/3/c-api/exceptions.html
- See Also: `sheets/troubleshooting/python-errors.md`, `detail/python/cpython-internals.md`, `detail/troubleshooting/asyncio-debugging.md`
