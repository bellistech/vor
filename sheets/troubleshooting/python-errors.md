# Python Errors

Exhaustive catalog of Python exceptions — verbatim text, root cause, and one-line fix for every common error from BaseException down through OS, asyncio, and major library hierarchies. Built so a terminal-bound developer never needs to web-search a Python traceback.

## Setup

An *exception* in Python is an object that signals an error condition — every exception is an instance of a class derived from `BaseException`. Raising an exception unwinds the stack until something catches it; uncaught exceptions print a traceback and exit with status `1`.

The hierarchy (3.12):

```text
BaseException
 ├── BaseExceptionGroup        (3.11+)
 ├── GeneratorExit
 ├── KeyboardInterrupt
 ├── SystemExit
 └── Exception
      ├── ArithmeticError
      │    ├── FloatingPointError
      │    ├── OverflowError
      │    └── ZeroDivisionError
      ├── AssertionError
      ├── AttributeError
      ├── BufferError
      ├── EOFError
      ├── ExceptionGroup       (3.11+)
      ├── ImportError
      │    └── ModuleNotFoundError
      ├── LookupError
      │    ├── IndexError
      │    └── KeyError
      ├── MemoryError
      ├── NameError
      │    └── UnboundLocalError
      ├── OSError              (== EnvironmentError == IOError)
      │    ├── BlockingIOError
      │    ├── ChildProcessError
      │    ├── ConnectionError
      │    │    ├── BrokenPipeError
      │    │    ├── ConnectionAbortedError
      │    │    ├── ConnectionRefusedError
      │    │    └── ConnectionResetError
      │    ├── FileExistsError
      │    ├── FileNotFoundError
      │    ├── InterruptedError
      │    ├── IsADirectoryError
      │    ├── NotADirectoryError
      │    ├── PermissionError
      │    ├── ProcessLookupError
      │    └── TimeoutError
      ├── ReferenceError
      ├── RuntimeError
      │    ├── NotImplementedError
      │    ├── PythonFinalizationError  (3.13+)
      │    └── RecursionError
      ├── StopAsyncIteration
      ├── StopIteration
      ├── SyntaxError
      │    └── IndentationError
      │         └── TabError
      ├── SystemError
      ├── TypeError
      ├── ValueError
      │    └── UnicodeError
      │         ├── UnicodeDecodeError
      │         ├── UnicodeEncodeError
      │         └── UnicodeTranslateError
      └── Warning              (DeprecationWarning, FutureWarning, etc.)
```

`except Exception` catches every "normal" error. `except BaseException` (or bare `except:`) also catches `KeyboardInterrupt`, `SystemExit`, and `GeneratorExit` — almost always a bug.

```python
import sys

try:
    1 / 0
except ZeroDivisionError:
    exc_type, exc_value, exc_tb = sys.exc_info()  # legacy 3-tuple
    # 3.11+: prefer except <Type> as e: e.__traceback__
```

Exception chaining (PEP 3134):

```python
try:
    int("abc")
except ValueError as orig:
    raise RuntimeError("could not parse config") from orig  # explicit cause
    # raise RuntimeError(...)             # implicit __context__ chain
    # raise RuntimeError(...) from None   # suppress chain
```

Multi-type catch — one tuple, one block:

```python
try:
    risky()
except (ValueError, TypeError) as e:   # catches either
    handle(e)

try:
    risky()
except ValueError as e:                # different handler per type
    handle_value(e)
except TypeError as e:
    handle_type(e)
```

3.11+ exception groups (PEP 654):

```python
try:
    raise ExceptionGroup("multi", [ValueError("v"), TypeError("t")])
except* ValueError as eg:    # except* unpacks groups
    handle(eg.exceptions)
except* TypeError as eg:
    handle(eg.exceptions)
```

## How to Read a Traceback

Default 3.10 format:

```text
Traceback (most recent call last):
  File "/home/u/app/main.py", line 42, in <module>
    process(data)
  File "/home/u/app/main.py", line 17, in process
    return d["key"].upper()
AttributeError: 'NoneType' object has no attribute 'upper'
```

Read **bottom-up**: the last line is the exception type and message; above that is the deepest frame; above that, the caller chain back to module-level. Each frame: `File "path", line N, in func` then the source line.

3.11+ adds **PEP 657 ↑ markers** pointing at the exact sub-expression:

```text
Traceback (most recent call last):
  File "main.py", line 17, in process
    return d["key"].upper()
           ~~~~~~~~^^^^^^^^
AttributeError: 'NoneType' object has no attribute 'upper'
```

The carets (`^^^^^^^^`) under `.upper()` show the precise call that failed; tildes (`~~~~~~~~`) underline the surrounding expression.

Chained exceptions print twice with a connector phrase:

- "**The above exception was the direct cause of the following exception:**" — used when `raise X from Y` (explicit `__cause__`).
- "**During handling of the above exception, another exception occurred:**" — used when an exception is raised inside an `except` block (implicit `__context__`).

Example chain:

```text
Traceback (most recent call last):
  File "a.py", line 3, in <module>
    int("x")
ValueError: invalid literal for int() with base 10: 'x'

The above exception was the direct cause of the following exception:

Traceback (most recent call last):
  File "a.py", line 5, in <module>
    raise RuntimeError("parse failed") from e
RuntimeError: parse failed
```

3.10+ "Did you mean…" hints — Python suggests likely names from the local namespace:

```text
NameError: name 'lenght' is not defined. Did you mean: 'length'?
AttributeError: 'list' object has no attribute 'appned'. Did you mean: 'append'?
```

Get clean tracebacks:

```python
import traceback
traceback.print_exc()                   # current exception to stderr
traceback.format_exc()                  # same as string
traceback.print_exception(exc_type, exc_value, exc_tb)
traceback.print_stack()                 # current stack (no exception)
```

Force PEP 657 caret printing in older code: `python3.11 -X no_debug_ranges` disables; `-X dev` enables more diagnostics.

```bash
python -X dev script.py            # development mode (extra warnings)
python -X tracemalloc=10 script.py # show memory allocation tracebacks
PYTHONDEVMODE=1 python script.py   # equivalent to -X dev
PYTHONFAULTHANDLER=1 python ...    # dump stack on segfault
```

## AttributeError

Raised when an attribute reference or assignment fails on an object.

`AttributeError: 'NoneType' object has no attribute 'X'`

```text
AttributeError: 'NoneType' object has no attribute 'upper'
```

Cause: a function returned `None` (often because no `return` statement, or the last branch fell through) and you called a method on it.

```python
# Broken
def first_word(s):
    if s:
        return s.split()[0]   # None when s is empty
print(first_word("").upper())

# Fix — guard or return a sentinel
def first_word(s):
    return s.split()[0] if s else ""
```

`AttributeError: module 'X' has no attribute 'Y'`

```text
AttributeError: module 'datetime' has no attribute 'datetime.now'
```

Cause: typo, wrong import (`import datetime` vs `from datetime import datetime`), or a name removed in your installed version.

```python
# Broken
import datetime
datetime.now()                    # module has no .now()

# Fix
from datetime import datetime
datetime.now()
# or
import datetime as dt
dt.datetime.now()
```

`AttributeError: __enter__`

```text
AttributeError: __enter__
```

Cause: used a non-context-manager in `with`. Common when forgetting to call a factory:

```python
# Broken
with open                              # function reference, not file
    pass

with my_pool                           # forgot ()
    pass

# Fix
with open("f.txt") as f:
    ...
with my_pool() as p:
    ...
```

`AttributeError: 'X' object has no attribute 'Y'`

```text
AttributeError: 'list' object has no attribute 'appned'. Did you mean: 'append'?
```

Cause: typo in attribute or wrong type assumption. The 3.10+ "Did you mean" heuristic suggests the closest name.

```python
# Broken
class User: pass
u = User()
u.name        # AttributeError: 'User' object has no attribute 'name'

# Fix — set in __init__
class User:
    def __init__(self, name): self.name = name
```

Defensive read patterns:

```python
val = getattr(obj, "name", "anon")     # default if missing
if hasattr(obj, "save"):
    obj.save()
val = getattr(obj, "deeply.nested", None)  # NOTE: literal name, no dots
```

Walrus + chained guard:

```python
if (m := response.get("meta")) and (u := m.get("user")):
    name = u["name"]
```

## TypeError

Operation applied to an operand of inappropriate type.

`TypeError: can only concatenate str (not "int") to str`

```text
TypeError: can only concatenate str (not "int") to str
```

```python
# Broken
"page " + 42

# Fix
f"page {42}"        # f-string
"page " + str(42)
"page %d" % 42
```

`TypeError: 'int' object is not iterable`

```text
TypeError: 'int' object is not iterable
```

```python
# Broken — for needs an iterable
for i in 5: ...

# Fix
for i in range(5): ...
```

`TypeError: 'list' object is not callable`

```text
TypeError: 'list' object is not callable
```

Cause: shadowed builtin (`list = [1,2]; list("abc")`), or `()` instead of `[]` for indexing.

```python
# Broken
list = [1, 2, 3]    # shadowed builtin
list((1, 2))        # TypeError

# Fix — never shadow builtins (lst, items, values)
items = [1, 2, 3]
list((1, 2))        # works
```

`TypeError: missing N required positional argument`

```text
TypeError: greet() missing 1 required positional argument: 'name'
```

```python
# Broken
def greet(name): print(f"hi {name}")
greet()

# Fix — provide arg, or give default
def greet(name="world"): print(f"hi {name}")
```

`TypeError: got an unexpected keyword argument 'X'`

```text
TypeError: open() got an unexpected keyword argument 'enconding'
```

Cause: typo (`enconding` → `encoding`) or argument added/removed in another version.

```python
# Broken
open("f.txt", enconding="utf-8")

# Fix
open("f.txt", encoding="utf-8")
```

`TypeError: unhashable type: 'list'`

```text
TypeError: unhashable type: 'list'
```

Cause: using a mutable as a dict key or set element.

```python
# Broken
{[1,2]: "a"}
{frozenset([1,2])}    # OK — frozenset is hashable
{[1,2]}               # TypeError

# Fix — convert to tuple/frozenset
{(1, 2): "a"}
{frozenset([1, 2])}
```

`TypeError: object of type 'X' has no len()`

```text
TypeError: object of type 'generator' has no len()
```

Cause: applied `len()` to something that doesn't implement `__len__` (generators, iterators, ints).

```python
# Broken
len(x for x in range(5))

# Fix — materialize or count
len(list(x for x in range(5)))
sum(1 for _ in (x for x in range(5)))
```

`TypeError: 'NoneType' object is not subscriptable`

```text
TypeError: 'NoneType' object is not subscriptable
```

```python
# Broken — dict.get returns None on miss
config.get("db")["host"]

# Fix
db = config.get("db") or {}
db.get("host")
```

`TypeError: cannot unpack non-iterable X object`

```text
TypeError: cannot unpack non-iterable int object
```

```python
# Broken
a, b = 7

# Fix — must come from iterable
a, b = (7, 8)
```

`TypeError: descriptor 'X' for 'Y' objects doesn't apply to a 'Z' object`

```text
TypeError: descriptor 'append' for 'list' objects doesn't apply to a 'tuple' object
```

Cause: calling an unbound method as `Class.method(other_type_instance)`.

```python
# Broken
list.append((1,2,3), 4)

# Fix
[1,2,3].append(4)
```

`TypeError: unsupported operand type(s) for +: 'X' and 'Y'`

```text
TypeError: unsupported operand type(s) for +: 'NoneType' and 'int'
```

```python
# Broken
total = None
total + 5

# Fix
total = total or 0
```

`TypeError: '<' not supported between instances of 'X' and 'Y'`

```text
TypeError: '<' not supported between instances of 'str' and 'int'
```

Python 3 forbids cross-type ordering.

```python
# Broken
sorted(["10", 2, "a"])

# Fix — coerce
sorted(["10", "2", "a"])
sorted(map(int, ["10", "2"]))   # if only ints
```

## NameError

Identifier not found in any accessible scope.

`NameError: name 'X' is not defined`

```text
NameError: name 'lenght' is not defined. Did you mean: 'length'?
```

Causes:

1. Typo (commonly `lenght`/`length`).
2. Used before assignment.
3. Forgot `import`.
4. Wrong scope (function-local hides module-level).

```python
# Broken — typo
lenght = 5
print(length)

# Broken — used before defined
def f():
    print(x)
    x = 1
f()                # UnboundLocalError (subclass of NameError)

# Fix — declare global or use parameter
def f(x):
    print(x)
```

Late-binding in closures (the **classic for-lambda bug**):

```python
# Broken — all lambdas print 4
fns = [lambda: i for i in range(5)]
for fn in fns: print(fn())   # 4 4 4 4 4

# Fix — bind via default arg
fns = [lambda i=i: i for i in range(5)]
print([fn() for fn in fns])  # [0,1,2,3,4]
```

`global` and `nonlocal`:

```python
counter = 0

def inc():
    global counter        # without this, counter is local
    counter += 1

def make_counter():
    n = 0
    def inc():
        nonlocal n        # walks up enclosing function scopes (not module)
        n += 1
        return n
    return inc
```

`nonlocal` requires the name already bound in an enclosing function scope; using `nonlocal` in a top-level function gives `SyntaxError: no binding for nonlocal 'X' found`.

## KeyError

Dict (or other Mapping) lookup failed.

`KeyError: 'X'`

```text
KeyError: 'database'
```

```python
# Broken
config["database"]

# Fix — .get() with default, or explicit handle
host = config.get("database", "localhost")
try:
    host = config["database"]
except KeyError:
    host = "localhost"
```

Membership test:

```python
if "database" in config:
    host = config["database"]
```

`dict.setdefault` to insert + return:

```python
groups.setdefault(key, []).append(item)
```

`collections.defaultdict` for auto-create:

```python
from collections import defaultdict
groups = defaultdict(list)
groups[key].append(item)        # never KeyError
```

Python 3.7+ guarantees insertion-order iteration on dicts — relying on that is now official, not just CPython behaviour.

`os.environ` is a special mapping — accessing a missing var raises `KeyError`, not returns `None`:

```python
# Broken — KeyError if HOME unset
home = os.environ["HOME"]

# Fix
home = os.environ.get("HOME", "/tmp")
home = os.getenv("HOME", "/tmp")    # equivalent
```

JSON load missing-key pattern:

```python
data = json.loads(payload)
user = data.get("user", {})
name = user.get("name", "anon")
# or pydantic / dataclass with defaults
```

## IndexError

Sequence subscript out of range.

`IndexError: list index out of range`

```text
IndexError: list index out of range
```

```python
# Broken — off-by-one
items = [1, 2, 3]
items[3]          # max valid index is 2

# Fix — len(items) is the boundary
items[len(items) - 1]    # last element
items[-1]                # idiomatic
```

`IndexError: string index out of range`

```python
s = ""
s[0]              # IndexError

# Fix
s[0] if s else ""
next(iter(s), "")
```

Negative indexing wraps but still errors past the start:

```python
"abc"[-3]    # 'a'
"abc"[-4]    # IndexError
```

Slicing never raises:

```python
[1,2,3][10:20]   # [] — silent empty
"abc"[10:]       # ""
```

Empty-tuple unpack:

```python
# Broken
(a,) = ()        # ValueError: not enough values to unpack
```

## ValueError

Right type but wrong value.

`ValueError: invalid literal for int() with base 10: 'X'`

```text
ValueError: invalid literal for int() with base 10: '12.5'
```

```python
# Broken
int("12.5")
int("")
int("0x1f")        # base 10 only

# Fix — convert through float, or specify base
int(float("12.5"))           # 12
int("0x1f", 16)              # 31
int("0x1f", 0)               # auto-detect prefix
```

`ValueError: too many values to unpack (expected N)`

```text
ValueError: too many values to unpack (expected 2)
```

```python
# Broken
a, b = (1, 2, 3)

# Fix — match arity, or use star
a, b, c = (1, 2, 3)
a, *rest = (1, 2, 3)
a, b, *rest = (1, 2, 3)
*head, last = (1, 2, 3)
```

`ValueError: not enough values to unpack (expected N, got M)`

```python
# Broken
a, b, c = (1, 2)

# Fix
a, b, c = (1, 2, None)
```

`ValueError: substring not found`

```text
ValueError: substring not found
```

```python
# Broken — str.index raises
"abc".index("z")

# Fix — use .find (returns -1) or check first
"abc".find("z")             # -1
if "z" in "abc": "abc".index("z")
```

`ValueError: I/O operation on closed file`

```python
# Broken — using f after with-block
with open("f.txt") as f:
    pass
data = f.read()             # closed

# Fix — read inside the block
with open("f.txt") as f:
    data = f.read()
```

`ValueError: Mixing iteration and read methods would lose data`

```python
# Broken — alternates iter and read
with open("f.txt") as f:
    for line in f:
        rest = f.read()      # ValueError on some streams

# Fix — pick one mode
with open("f.txt") as f:
    rest = f.read()
```

Other common:

```text
ValueError: math domain error          # math.sqrt(-1), math.log(0)
ValueError: time data 'X' does not match format '%Y'
ValueError: max() arg is an empty sequence
```

```python
# Fix patterns
math.sqrt(abs(x))
datetime.strptime(s, "%Y-%m-%d")           # match exactly
max(seq, default=0)                         # 3.4+
```

## ImportError vs ModuleNotFoundError

`ModuleNotFoundError` (3.6+) — subclass of `ImportError` — fires when the module itself can't be located.

`ModuleNotFoundError: No module named 'X'`

```text
ModuleNotFoundError: No module named 'requests'
```

```bash
# Diagnose — is it installed in the active interpreter?
python -c "import sys; print(sys.executable)"
python -m pip list | grep -i requests
which python && which pip

# Fix
python -m pip install requests              # install in CURRENT python
pip install requests                        # may use a different python
uv pip install requests                     # if using uv
```

The `python -m pip ...` form guarantees you install into the same interpreter you're running.

`ImportError: cannot import name 'X' from 'Y'`

```text
ImportError: cannot import name 'cached_property' from 'functools'
```

Causes: name removed/renamed in your version, circular import, or typo.

```python
# Diagnose by version
import sys; sys.version_info       # (3, 7, x) -> cached_property added in 3.8

# Fix — install backport, or polyfill
try:
    from functools import cached_property
except ImportError:
    from cached_property import cached_property  # PyPI backport
```

`ImportError: attempted relative import with no known parent package`

```text
ImportError: attempted relative import with no known parent package
```

Cause: ran a file inside a package directly (`python pkg/sub.py`) instead of via `-m`.

```bash
# Broken
python myapp/cli.py        # has `from . import util`

# Fix — run as module
python -m myapp.cli
# or refactor to absolute imports if running script directly
```

Circular imports — A imports B, B imports A:

```python
# a.py
from b import B
class A: ...

# b.py
from a import A            # ImportError: cannot import name 'A'
class B: ...

# Fix 1 — move import inside function (lazy)
# b.py
class B:
    def use(self):
        from a import A
        return A()

# Fix 2 — extract shared parts to c.py imported by both
```

`sys.path` debugging:

```python
import sys
print("\n".join(sys.path))     # what import searches
sys.path.insert(0, "/extra")   # only as a last resort
```

```bash
# .pth files in site-packages auto-extend sys.path
python -c "import site; print(site.getsitepackages())"

# Editable install lays down a .pth pointing at your source
pip install -e .
```

Namespace package vs regular package — directory **without** `__init__.py` is a namespace package (PEP 420). Two namespace dirs of same name across `sys.path` *merge*; a regular package shadows everything else. If your tests can't import a sibling module, missing `__init__.py` is a common cause.

```bash
find . -type d -not -path "*/.*" -exec test -e {}/__init__.py \; -o -print
```

## SyntaxError

Tokenizer/parser failure — code never ran.

`SyntaxError: invalid syntax`

```text
  File "x.py", line 3
    if x = 5:
       ^
SyntaxError: invalid syntax
```

```python
# Broken
if x = 5:           # = is assignment, not comparison

# Fix
if x == 5:
```

3.11+ uses PEP 657 caret ranges — `^^^^^^^` underlines exact span, not just one char.

`SyntaxError: unexpected EOF while parsing`

```text
SyntaxError: unexpected EOF while parsing
```

Cause: unbalanced bracket / paren / colon at end of file.

```python
# Broken
def f():
    print("hi"

# Fix — close paren
def f():
    print("hi")
```

`SyntaxError: f-string: empty expression not allowed`

```python
# Broken
f"value: {}"

# Fix
f"value: {x}"
```

3.12 (PEP 701) relaxes f-string nesting and quote reuse — older Pythons forbid `f"{ "nested" }"`.

`SyntaxError: cannot assign to literal`

```text
SyntaxError: cannot assign to literal
```

```python
# Broken
3 = x
"key" = "v"

# Fix
x = 3
d["key"] = "v"
```

`SyntaxError: 'await' outside function`

```text
SyntaxError: 'await' outside function
```

```python
# Broken
await main()              # at module level

# Fix — wrap in async function
async def run():
    await main()
asyncio.run(run())
# or, in Python REPL 3.8+, run with: python -m asyncio
```

`SyntaxError: 'return' outside function`

```python
# Broken
return 1                  # at module top-level

# Fix
def f(): return 1
```

`SyntaxError: positional argument follows keyword argument`

```text
SyntaxError: positional argument follows keyword argument
```

```python
# Broken
f(a=1, 2)

# Fix — positional first
f(2, a=1)
```

`SyntaxError: non-default argument follows default argument`

```python
# Broken
def f(a=1, b): ...

# Fix
def f(b, a=1): ...
```

`SyntaxError: duplicate argument 'X' in function definition`

```python
def f(a, a): ...   # error
```

`SyntaxError: 'break' outside loop`, `'continue' not properly in loop`

```python
# Fix — must be inside while/for
```

3.12+ improved messages — e.g. `SyntaxError: invalid syntax. Perhaps you forgot a comma?`.

## IndentationError + TabError

`IndentationError: unexpected indent`

```text
IndentationError: unexpected indent
```

Cause: a line indented when it shouldn't be (after `def`/`if`/etc, the *first* line sets the level).

```python
# Broken
def f():
    x = 1
        y = 2     # too deep

# Fix
def f():
    x = 1
    y = 2
```

`IndentationError: expected an indented block`

```python
# Broken
def f():
pass            # not indented

# Fix
def f():
    pass
```

`IndentationError: unindent does not match any outer indentation level`

```python
# Broken — mixing 4 spaces and 2 spaces
if True:
    if True:
      x = 1     # 6 spaces, not matching outer 4

# Fix — pick one indent (PEP 8 = 4 spaces)
```

`TabError: inconsistent use of tabs and spaces in indentation`

```text
TabError: inconsistent use of tabs and spaces in indentation
```

```bash
# Diagnose
python -tt script.py            # error on inconsistent tabs/spaces
cat -A script.py | head         # ^I = tab, $ = end-of-line

# Fix — convert tabs to 4 spaces
expand -t 4 script.py > /tmp/x && mv /tmp/x script.py
# or in vim
:retab
```

Editor settings: configure your editor to *show whitespace* and *insert spaces for tab*. PEP 8 requires 4-space indents.

## RuntimeError

Generic runtime errors that don't fit other categories.

`RuntimeError: dictionary changed size during iteration`

```text
RuntimeError: dictionary changed size during iteration
```

```python
# Broken
for k in d:
    if cond(k): del d[k]

# Fix — iterate over a copy of keys
for k in list(d):
    if cond(k): del d[k]

# Or — comprehension to new dict
d = {k: v for k, v in d.items() if not cond(k)}
```

`RuntimeError: Set changed size during iteration`

```python
# Same fix — iterate over set(s) copy
for x in set(s):
    if cond(x): s.discard(x)
```

`RuntimeError: This event loop is already running`

```text
RuntimeError: This event loop is already running
```

Cause: calling `asyncio.run()` (or `loop.run_until_complete()`) inside an already-running loop (notebooks, FastAPI handlers).

```python
# Broken
async def task(): pass
def sync_caller():
    asyncio.run(task())   # inside async context

# Fix — await directly
async def caller():
    await task()

# In Jupyter
import nest_asyncio; nest_asyncio.apply()
```

`RuntimeError: cannot reuse already awaited coroutine`

```python
# Broken
c = my_coro()
await c
await c                # error second time

# Fix — call factory each await
await my_coro()
await my_coro()
```

`RuntimeError: coroutine 'X' was never awaited`

```text
RuntimeWarning: coroutine 'fetch' was never awaited
```

```python
# Broken
fetch()                # creates coroutine, discards

# Fix
await fetch()                  # in async context
asyncio.run(fetch())           # at top level
```

`RuntimeError: maximum recursion depth exceeded` — see RecursionError below; same root cause; raised by deeply-nested calls or `__repr__` recursion.

`RuntimeError: generator raised StopIteration` — see PEP 479 in StopIteration section.

`RuntimeError: There is no current event loop in thread 'X'.` — see asyncio section.

## RecursionError

`RecursionError: maximum recursion depth exceeded while calling a Python object`

```text
RecursionError: maximum recursion depth exceeded while calling a Python object
```

```python
# Broken — naive recursion
def fact(n):
    return 1 if n == 0 else n * fact(n-1)
fact(5000)              # exceeds default 1000 limit

# Diagnose
import sys; sys.getrecursionlimit()         # default 1000

# Fix 1 — bump limit (rarely the right answer)
sys.setrecursionlimit(10000)

# Fix 2 — iterative
def fact(n):
    r = 1
    for i in range(2, n+1): r *= i
    return r

# Fix 3 — explicit stack
def walk(root):
    stack = [root]
    while stack:
        node = stack.pop()
        stack.extend(node.children)
```

`__repr__` recursion (an instance whose `__repr__` returns `repr(self)`) is a subtle infinite recursion — debug with `reprlib`.

CPython tail-call optimization: there is none. Even pure tail recursion blows the stack.

3.13+ raised the default limit and added a `PYTHON_RECURSION_LIMIT` env var.

## StopIteration / StopAsyncIteration

Used internally to signal generator exhaustion; `for` swallows them.

```python
g = iter([1,2])
next(g)              # 1
next(g)              # 2
next(g)              # StopIteration (or use default)
next(g, None)        # None  — recommended
```

PEP 479 (3.7+): a `StopIteration` raised *inside* a generator function is converted to `RuntimeError: generator raised StopIteration`. Before 3.7 this silently terminated the generator.

```python
# Broken — accidentally bubbles up
def gen():
    yield 1
    yield next(other_gen)     # if other_gen empty -> StopIteration -> RuntimeError

# Fix — handle explicitly
def gen():
    yield 1
    try:
        yield next(other_gen)
    except StopIteration:
        return
```

`StopAsyncIteration` — same role for `async for` / `__aiter__`.

```python
class Counter:
    def __init__(self, n): self.n, self.i = n, 0
    def __aiter__(self): return self
    async def __anext__(self):
        if self.i >= self.n: raise StopAsyncIteration
        self.i += 1
        return self.i
```

## FileNotFoundError + PermissionError + IsADirectoryError + FileExistsError + NotADirectoryError

All inherit `OSError`; carry `errno` and `strerror`.

`FileNotFoundError: [Errno 2] No such file or directory: 'X'`

```text
FileNotFoundError: [Errno 2] No such file or directory: 'config.toml'
```

```python
# Broken — relative path resolves vs cwd, not the script dir
open("config.toml")

# Fix — resolve relative to the script
from pathlib import Path
HERE = Path(__file__).resolve().parent
open(HERE / "config.toml")

# Pre-check
if not Path("config.toml").exists():
    raise SystemExit("config.toml not found")
```

`PermissionError: [Errno 13] Permission denied: 'X'`

```python
# Cause — wrong owner, mode, or directory not executable
# Fix
import os, stat
os.chmod(path, stat.S_IRUSR | stat.S_IWUSR)        # 0o600
# Or run with sufficient privileges; check parent dir +x
```

`IsADirectoryError: [Errno 21] Is a directory: 'X'`

```python
# Broken
open("/tmp")

# Fix — use os.path.isdir / Path.is_dir guard
from pathlib import Path
p = Path("/tmp")
if p.is_dir():
    for child in p.iterdir(): print(child)
else:
    open(p)
```

`FileExistsError: [Errno 17] File exists: 'X'`

```python
# Broken
os.makedirs("/tmp/out")     # second time -> FileExistsError

# Fix — exist_ok
os.makedirs("/tmp/out", exist_ok=True)
Path("/tmp/out").mkdir(parents=True, exist_ok=True)
open("f.txt", "x")          # exclusive — fails if exists (use intentionally)
```

`NotADirectoryError: [Errno 20] Not a directory: 'X'`

```python
# Broken — treating file path as dir
os.listdir("/tmp/file.txt")

# Fix — guard
if Path(p).is_dir(): ...
```

`os.path` vs `pathlib.Path`:

```python
# Old style
import os
os.path.join("a", "b", "c")
os.path.exists(p)
os.path.basename(p)
os.path.splitext(p)[1]

# Modern (pathlib)
from pathlib import Path
Path("a") / "b" / "c"          # operator overload
Path(p).exists()
Path(p).name
Path(p).suffix
Path(p).read_text()            # one-shot
Path(p).write_bytes(b)
```

`pathlib` understands Windows backslashes and drive letters; `os.path` is platform-aware too but stringly-typed.

## UnicodeDecodeError + UnicodeEncodeError + UnicodeError

`UnicodeDecodeError: 'utf-8' codec can't decode byte 0xN in position M`

```text
UnicodeDecodeError: 'utf-8' codec can't decode byte 0xff in position 0: invalid start byte
```

Cause: file isn't UTF-8 (often Latin-1 / cp1252 / UTF-16 BOM).

```python
# Broken
open("data.txt").read()      # locale-dependent default encoding

# Fix — declare encoding
open("data.txt", encoding="utf-8").read()
open("data.txt", encoding="latin-1").read()        # any byte decodes
open("data.txt", encoding="utf-8", errors="replace").read()  # ?? for bad bytes

# Detect
import chardet
chardet.detect(open("data.txt", "rb").read(4096))
```

`UnicodeEncodeError: 'ascii' codec can't encode character 'X' in position M`

```text
UnicodeEncodeError: 'ascii' codec can't encode character '—' in position 4: ordinal not in range(128)
```

Cause: writing non-ASCII to a stream whose encoding is ASCII (some Windows consoles, redirected stdout in Docker without locale).

```python
# Diagnose
import sys, locale
print(sys.stdout.encoding)
print(locale.getpreferredencoding())

# Fix 1 — UTF-8 mode
PYTHONUTF8=1 python script.py
PYTHONIOENCODING=utf-8 python script.py

# Fix 2 — reconfigure stdout (3.7+)
sys.stdout.reconfigure(encoding="utf-8", errors="replace")

# Fix 3 — open file with encoding
open("out.txt", "w", encoding="utf-8")
```

3.10+ supports `encoding="locale"` as the explicit "use locale default"; 3.15 plans to make `open()` default to UTF-8 unconditionally.

```bash
# Linux/macOS sanity
locale
# en_US.UTF-8 expected; LC_ALL=C will break Unicode output
LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8 python script.py

# Docker baseline
ENV LANG=C.UTF-8 LC_ALL=C.UTF-8
```

## ZeroDivisionError + OverflowError + FloatingPointError

`ZeroDivisionError: division by zero`

```text
ZeroDivisionError: division by zero
```

```python
# Broken
1 / 0
1 // 0
1 % 0
divmod(1, 0)
math.log(0)            # ValueError, not ZeroDivisionError

# Fix — guard
def safe_div(a, b):
    return a / b if b else float("inf")
```

`OverflowError: int too large to convert to float`

```python
# Broken
float(10**400)          # int is unbounded, float is IEEE-754

# Fix
math.log(10**400)       # log handles big ints
from decimal import Decimal
Decimal(10) ** 400
```

`OverflowError: math range error`

```text
OverflowError: math range error
```

```python
# Broken
math.exp(1000)

# Fix
import math
math.log(...)           # use log space
mpmath.exp(1000)        # arbitrary precision
```

`FloatingPointError` — only raised when `numpy.errstate` or `fpectl` is enabled; default Python silently produces `inf` / `nan`.

```python
import numpy as np
with np.errstate(divide="raise"):
    np.array([1.0]) / 0     # FloatingPointError
```

`Decimal` for predictable arithmetic:

```python
from decimal import Decimal, getcontext, ROUND_HALF_EVEN
getcontext().prec = 28
Decimal("0.1") + Decimal("0.2")     # Decimal('0.3')
```

## MemoryError

Raised when allocation fails. Often fatal — process may already be in OOM-killer territory.

```text
MemoryError
```

```bash
# Diagnose
python -X tracemalloc=10 script.py
```

```python
import tracemalloc, resource

tracemalloc.start(10)
# ... run code ...
snap = tracemalloc.take_snapshot()
for stat in snap.statistics("lineno")[:10]:
    print(stat)

# Limit address space (Linux/macOS)
resource.setrlimit(resource.RLIMIT_AS, (2 * 1024**3, 2 * 1024**3))   # 2 GiB
```

Common causes: building a giant list when a generator would do; loading a whole CSV instead of streaming with `csv.reader`; slurping a 10 GiB file with `read()`.

```python
# Bad
data = open("big.csv").read().split("\n")

# Good
with open("big.csv") as f:
    for line in f:
        ...
```

## NotImplementedError vs raise NotImplemented

`NotImplementedError` is a real exception; `NotImplemented` is a singleton constant returned (not raised) by `__eq__` / `__lt__` / arithmetic dunders to signal "I don't know — try the other operand". They are different objects.

```python
# Correct — abstract method
class Base:
    def step(self):
        raise NotImplementedError("subclass must implement step()")

# Wrong — does not raise; returns NotImplemented as a value
class Wrong:
    def step(self):
        raise NotImplemented            # TypeError at runtime

# Better — abc module
from abc import ABC, abstractmethod
class Base(ABC):
    @abstractmethod
    def step(self): ...
# Instantiating Base or a subclass without step() -> TypeError at construction
```

`__eq__` returning `NotImplemented` lets Python try the right-hand side:

```python
class A:
    def __eq__(self, other):
        if not isinstance(other, A):
            return NotImplemented       # the value, not raise
        return self.x == other.x
```

## AssertionError

`AssertionError: <message>`

```python
assert x > 0, f"x must be positive, got {x}"
# AssertionError: x must be positive, got -1
```

`-O` flag strips assertions:

```bash
python -O script.py        # __debug__=False; assert is no-op
python -OO script.py       # also strips docstrings
```

**Never use assertions for security checks** or for validating untrusted input — they vanish under `-O`. Use explicit `if not …: raise ValueError(…)`.

```python
# Wrong — disabled with -O
def withdraw(amount):
    assert amount > 0
    ...

# Right
def withdraw(amount):
    if amount <= 0:
        raise ValueError("amount must be positive")
```

`pytest` rewrites assertion expressions to print rich diffs:

```python
assert response.status_code == 200
# pytest output:
# AssertionError: assert 500 == 200
#  +  where 500 = <Response [500]>.status_code
```

## KeyboardInterrupt

Sent on `Ctrl+C` (SIGINT). Subclass of `BaseException`, **not** `Exception`.

```python
try:
    long_running()
except KeyboardInterrupt:
    print("\ncancelled")
    sys.exit(130)             # convention: 128 + signal number
```

A naked `except:` catches `KeyboardInterrupt` — almost always a bug:

```python
# Broken — Ctrl+C ignored
while True:
    try:
        do()
    except:
        continue

# Fix
while True:
    try:
        do()
    except Exception:
        continue              # Ctrl+C now propagates
```

Custom `SIGINT` handling:

```python
import signal

def handler(signum, frame):
    cleanup()
    sys.exit(0)

signal.signal(signal.SIGINT, handler)
```

`KeyboardInterrupt` vs `SystemExit` — both inherit `BaseException`. `SystemExit` is raised by `sys.exit(N)` and propagates; the interpreter catches it at the top and exits with `N`. `KeyboardInterrupt` is raised by the SIGINT handler default behaviour.

## SystemExit

```python
sys.exit()                    # exit 0
sys.exit(0)
sys.exit(1)                   # generic error
sys.exit("oops")              # prints "oops" to stderr, exits 1
```

Bare `except:` swallows it:

```python
# Broken — sys.exit() is caught and ignored
try:
    sys.exit(1)
except:
    pass

# Fix — narrow
try:
    risky()
except Exception:
    handle()
```

`argparse` exits with code 2 on parse errors and code 0 on `--help`. `argparse.ArgumentError` is internal; for tests use `pytest.raises(SystemExit)`:

```python
import pytest
with pytest.raises(SystemExit) as ei:
    parser.parse_args(["--bad"])
assert ei.value.code == 2
```

## GeneratorExit

Raised inside a generator when it is garbage collected or `.close()`-d. Inherits `BaseException` so it doesn't get caught by `except Exception`.

```python
def gen():
    try:
        yield 1
        yield 2
    except GeneratorExit:
        # cleanup; do NOT re-raise something else
        cleanup()
        raise

g = gen()
next(g)
g.close()                    # triggers GeneratorExit inside gen
```

Use `finally` for unconditional cleanup:

```python
def gen():
    try:
        for x in source:
            yield x
    finally:
        source.close()       # runs on close() / GC
```

A generator that `yield`s while handling `GeneratorExit` raises `RuntimeError: generator ignored GeneratorExit`.

## EOFError + BrokenPipeError + ConnectionError family

`EOFError: EOF when reading a line`

```text
EOFError: EOF when reading a line
```

```python
# Broken — input() at end of stdin
name = input("name? ")        # piped input ran out

# Fix — handle EOF
try:
    name = input("name? ")
except EOFError:
    name = "anon"
```

`BrokenPipeError: [Errno 32] Broken pipe`

```text
BrokenPipeError: [Errno 32] Broken pipe
```

Cause: writing to a pipe whose reader closed (`python script.py | head` after head exits).

```python
# Mitigate at exit
import sys, signal
signal.signal(signal.SIGPIPE, signal.SIG_DFL)   # POSIX only
try:
    for line in lines:
        sys.stdout.write(line + "\n")
except BrokenPipeError:
    sys.stderr.close()                          # avoid further writes
```

`ConnectionRefusedError: [Errno 111] Connection refused`

```python
# Cause — nothing listening on host:port (or wrong port)
# Fix — verify service running, retry with backoff
import socket, time
for attempt in range(5):
    try:
        s = socket.create_connection(("host", 80), timeout=2)
        break
    except ConnectionRefusedError:
        time.sleep(2 ** attempt)
```

`ConnectionResetError: [Errno 104] Connection reset by peer`

```python
# Cause — peer sent RST (kernel-level abort), often firewall / load balancer
# Fix — retry with backoff; verify keep-alives; check ulimit
```

`ConnectionAbortedError`

```python
# Cause — local TCP stack aborted the connection (often Windows-specific)
```

## TimeoutError + socket.timeout

`socket.timeout` was deprecated in 3.10 and is now an alias of `TimeoutError`.

`TimeoutError: [Errno 110] Connection timed out`

```text
TimeoutError: [Errno 110] Connection timed out
```

```python
# socket
sock.settimeout(5.0)           # blocks max 5s; timeout -> TimeoutError

# requests
import requests
requests.get(url, timeout=(3, 10))   # (connect, read)
# requests.exceptions.ConnectTimeout / ReadTimeout

# httpx
httpx.get(url, timeout=10.0)
# httpx.ConnectTimeout / ReadTimeout / WriteTimeout / PoolTimeout

# aiohttp
async with aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=10)) as s:
    await s.get(url)
# asyncio.TimeoutError on miss

# asyncio
await asyncio.wait_for(coro, timeout=5)   # asyncio.TimeoutError
```

`select` / `poll` / `epoll`:

```python
import select
ready, _, _ = select.select([sock], [], [], 5.0)
if not ready:
    raise TimeoutError
```

## OSError family

The parent of nearly every I/O error. Use `e.errno` to dispatch:

```python
import errno
try:
    open(p, "w")
except OSError as e:
    if e.errno == errno.EACCES: ...
    elif e.errno == errno.ENOSPC: ...
    elif e.errno == errno.EMFILE: ...
    else: raise
```

Common errnos:

```text
ENOENT      2    No such file or directory   FileNotFoundError
EACCES     13    Permission denied           PermissionError
EEXIST     17    File exists                 FileExistsError
ENOTDIR    20    Not a directory             NotADirectoryError
EISDIR     21    Is a directory              IsADirectoryError
EMFILE     24    Too many open files         OSError
ENAMETOOLONG 36  File name too long          OSError
ENOSPC     28    No space left on device     OSError
ECONNREFUSED 111 Connection refused          ConnectionRefusedError
ECONNRESET 104   Connection reset by peer    ConnectionResetError
EPIPE      32    Broken pipe                 BrokenPipeError
ETIMEDOUT 110    Connection timed out        TimeoutError
```

`OSError: [Errno 28] No space left on device`

```bash
df -h .
du -sh /tmp/* | sort -h | tail
```

`OSError: [Errno 24] Too many open files`

```bash
# Diagnose
ulimit -n               # soft limit
lsof -p <pid> | wc -l

# Fix at run-time
import resource
resource.setrlimit(resource.RLIMIT_NOFILE, (10000, 10000))

# Persistent
echo "ulimit -n 10000" >> ~/.zshrc
```

```python
# Most common cause — leak from open()
# Wrong
for path in paths:
    f = open(path)
    process(f)         # f never closed

# Right
for path in paths:
    with open(path) as f:
        process(f)
```

`OSError: [Errno 36] File name too long`

```python
# Linux PATH_MAX=4096, NAME_MAX=255 per component
# Fix — hash long filenames
import hashlib
fn = hashlib.sha1(long_name.encode()).hexdigest() + ".dat"
```

`OSError: [WinError N] message` — Windows-specific. `WinError 5` = access denied; `WinError 32` = file in use.

## asyncio CancelledError

3.8+: `CancelledError` inherits `BaseException`, not `Exception`. So `except Exception` no longer catches it — that's intentional, so cleanup code can run without absorbing the cancellation.

```python
# Wrong — eats cancellation
try:
    await long_op()
except Exception:
    handle()

# Right — re-raise
try:
    await long_op()
except asyncio.CancelledError:
    cleanup()
    raise
```

Pattern for graceful shutdown:

```python
async def worker():
    try:
        while True:
            await do_step()
    except asyncio.CancelledError:
        await flush()
        raise
```

`trio` analogue: `trio.Cancelled` — same "do not swallow" rule.

## asyncio Errors

`RuntimeError: coroutine 'X' was never awaited`

```text
sys:1: RuntimeWarning: coroutine 'fetch' was never awaited
```

```python
# Broken
async def fetch(): ...
fetch()                # creates, never awaited

# Fix
await fetch()
asyncio.run(fetch())
asyncio.create_task(fetch())     # schedule for later
```

Find with `python -W error::RuntimeWarning script.py`.

`RuntimeError: There is no current event loop in thread 'X'.`

```python
# Broken — 3.10+ deprecated implicit creation
loop = asyncio.get_event_loop()

# Fix
loop = asyncio.new_event_loop()
asyncio.set_event_loop(loop)
# Or — high-level
asyncio.run(main())
```

`RuntimeError: Task was destroyed but it is pending!`

Cause: program exited while a Task was still running; loop was torn down.

```python
# Fix — keep a strong reference, await on shutdown
tasks = []
tasks.append(asyncio.create_task(worker()))
...
await asyncio.gather(*tasks, return_exceptions=True)
```

`RuntimeError: Event loop stopped before Future completed`

```python
# Broken — loop.stop() while futures pending
# Fix — let asyncio.run() manage lifetime; or await all before stopping
```

`asyncio.TimeoutError` — raised by `asyncio.wait_for(coro, timeout)`. In 3.11+ it's an alias of the builtin `TimeoutError`.

```python
try:
    await asyncio.wait_for(slow(), timeout=5)
except (asyncio.TimeoutError, TimeoutError):
    log.warning("slow op timed out")
```

3.11+ `asyncio.TaskGroup` propagates exceptions in an `ExceptionGroup`:

```python
async with asyncio.TaskGroup() as tg:
    tg.create_task(a())
    tg.create_task(b())
# If both raise, you get ExceptionGroup at exit
```

## Pydantic + dataclass + typing exceptions

`pydantic.ValidationError: N validation errors for X`

```text
pydantic.ValidationError: 2 validation errors for User
name
  Field required [type=missing, input_value={...}, input_type=dict]
age
  Input should be a valid integer [type=int_parsing, input_value='x', input_type=str]
```

```python
# v2 access
from pydantic import BaseModel, ValidationError
class User(BaseModel):
    name: str; age: int

try:
    User(age="x")
except ValidationError as e:
    print(e.errors())          # list[dict]
    print(e.json())            # JSON string
```

dataclass:

```text
TypeError: __init__() missing 1 required positional argument: 'name'
TypeError: __init__() got an unexpected keyword argument 'fullname'
```

```python
# Broken — mutable default
from dataclasses import dataclass
@dataclass
class A:
    items: list = []           # ValueError: mutable default for field

# Fix
from dataclasses import dataclass, field
@dataclass
class A:
    items: list = field(default_factory=list)
```

`TypeError: dataclass() got an unexpected keyword argument 'X'` — likely using a kw added in a later Python (`slots=` is 3.10+, `kw_only=` is 3.10+).

`typing.get_type_hints` failures:

```text
NameError: name 'X' is not defined
```

Cause: forward reference (string-quoted type) referencing a name not yet imported. Fix: ensure the name is in scope at call time, or use `from __future__ import annotations` and resolve carefully.

```python
from __future__ import annotations  # all annotations become strings
# get_type_hints needs to resolve them — provide globalns/localns if needed
typing.get_type_hints(MyClass, globalns=globals())
```

## NumPy / Pandas Common Errors

`ValueError: setting an array element with a sequence`

```text
ValueError: setting an array element with a sequence. The requested array has an inhomogeneous shape after 1 dimensions.
```

```python
# Broken — ragged list
np.array([[1, 2, 3], [4, 5]])

# Fix — pad to rectangular, or use object dtype
np.array([[1,2,3],[4,5,0]])
np.array([[1,2,3],[4,5]], dtype=object)
```

`ValueError: operands could not be broadcast together with shapes (3,) (4,)`

```text
ValueError: operands could not be broadcast together with shapes (3,) (4,)
```

```python
# Broken
np.array([1,2,3]) + np.array([1,2,3,4])

# Fix — same shape, or compatible (broadcasting rules)
a = np.array([1,2,3])
b = np.array([10])           # (1,) broadcasts to (3,)
a + b                        # [11,12,13]
```

`KeyError: 'X' not in index` (pandas)

```python
# Broken
df["missing_col"]

# Fix
df.get("missing_col")        # None if missing
df.reindex(columns=["a","b"], fill_value=0)
```

`ValueError: cannot reindex on an axis with duplicate labels`

```python
# Cause — duplicate index entries
df = df[~df.index.duplicated(keep="first")]
```

`SettingWithCopyWarning`

```text
SettingWithCopyWarning:
A value is trying to be set on a copy of a slice from a DataFrame
```

```python
# Broken — chained indexing
df[df.x > 0]["y"] = 1

# Fix — single .loc
df.loc[df.x > 0, "y"] = 1
# or copy explicitly when you want a separate frame
sub = df[df.x > 0].copy()
sub["y"] = 1
```

`FutureWarning` / `DeprecationWarning` — feature changes in upcoming version. Treat as actionable. Run with `-W error::DeprecationWarning` to fail tests.

`PerformanceWarning` (pandas) — usually for fragmented dataframes after many `insert` / `concat` ops; defragment with `df = df.copy()`.

```python
# Convert all warnings to errors during testing
PYTHONWARNINGS=error pytest
```

## Django Common Errors

`django.db.utils.OperationalError: no such table: X`

```text
django.db.utils.OperationalError: no such table: auth_user
```

```bash
# Cause — migrations not run
python manage.py migrate
python manage.py showmigrations         # what's pending
```

`django.urls.exceptions.NoReverseMatch`

```text
NoReverseMatch: Reverse for 'view-name' not found.
```

```python
# Cause — typo in {% url 'name' %}, or url not named, or kwargs missing
# Fix — verify name in urls.py
path("posts/<int:pk>/", views.detail, name="post-detail")
{% url 'post-detail' pk=post.pk %}
```

`django.core.exceptions.AppRegistryNotReady: Apps aren't loaded yet.`

```python
# Cause — importing models before django.setup() in a script
import django
django.setup()         # before any model import
from myapp.models import User
```

`django.core.exceptions.ImproperlyConfigured: settings.DATABASES is improperly configured`

```bash
# Cause — DJANGO_SETTINGS_MODULE not set, or env var missing
export DJANGO_SETTINGS_MODULE=myproj.settings
```

`TemplateSyntaxError`

```text
TemplateSyntaxError: Invalid block tag: 'endif', expected 'endblock'
```

```python
# Cause — unbalanced {% block %} / {% if %} tags
# Fix — pair tags, check inheritance
```

`OperationalError: FATAL: password authentication failed for user "X"`

```bash
# Diagnose
psql "$DATABASE_URL" -c "select 1"
# Fix — credentials in .env or settings
```

## Flask + FastAPI Errors

`RuntimeError: Working outside of application context.`

```python
# Cause — accessing current_app or db outside a request
# Fix
with app.app_context():
    db.create_all()
```

`RuntimeError: Working outside of request context.`

```python
# Cause — accessing request / session in a thread / Celery task
# Fix
with app.test_request_context():
    ...
```

`TypeError: View function did not return a response.`

```python
# Broken
@app.route("/")
def index():
    pass            # returns None

# Fix
@app.route("/")
def index():
    return "hello"
```

FastAPI:

`fastapi.exceptions.RequestValidationError`

```text
{"detail":[{"type":"int_parsing","loc":["query","page"],"msg":"Input should be a valid integer"}]}
```

```python
# 422 default; customize
from fastapi import Request
from fastapi.responses import JSONResponse
@app.exception_handler(RequestValidationError)
async def handler(req: Request, exc: RequestValidationError):
    return JSONResponse({"errors": exc.errors()}, status_code=400)
```

`pydantic.ValidationError` inside response models — your code returned data that doesn't match `response_model`. Diagnose with `e.errors()`.

## SQLAlchemy Errors

`sqlalchemy.exc.OperationalError: (psycopg2.OperationalError) ...`

```text
sqlalchemy.exc.OperationalError: (psycopg2.OperationalError) FATAL: database "X" does not exist
```

Wraps the underlying DB driver error. `e.orig` is the driver exception.

```python
# Diagnose
try:
    db.execute(...)
except OperationalError as e:
    print(e.orig)              # the real psycopg2 error
```

`sqlalchemy.exc.IntegrityError`

```text
IntegrityError: (psycopg2.errors.UniqueViolation) duplicate key value violates unique constraint "users_email_key"
```

```python
# Pattern — try/except + rollback
try:
    session.add(u); session.commit()
except IntegrityError:
    session.rollback()
    # surface as 409 Conflict in API
```

`sqlalchemy.exc.PendingRollbackError`

```text
PendingRollbackError: This Session's transaction has been rolled back due to a previous exception
```

Cause: prior error left the session in a failed state; you must `rollback()` before reusing.

```python
@contextmanager
def session_scope():
    s = SessionLocal()
    try:
        yield s
        s.commit()
    except:
        s.rollback()
        raise
    finally:
        s.close()
```

`DetachedInstanceError: Instance <X> is not bound to a Session`

```python
# Cause — accessing lazy attribute after session closed
# Fix — eager-load with options(joinedload(...)), or expire_on_commit=False
session = Session(engine, expire_on_commit=False)
```

`InvalidRequestError` — covers many cases (e.g. "Object is already attached to session", "no such relationship"). Read message text for specifics.

## requests + httpx + urllib3 Errors

```python
import requests
try:
    r = requests.get(url, timeout=10)
    r.raise_for_status()
except requests.exceptions.ConnectionError as e:
    # DNS failure, refused, network down
except requests.exceptions.SSLError as e:
    # TLS failure — see SSL section
except requests.exceptions.Timeout as e:        # ConnectTimeout, ReadTimeout
    # network slow / unresponsive
except requests.exceptions.HTTPError as e:      # raise_for_status on 4xx/5xx
    print(e.response.status_code, e.response.text)
except requests.exceptions.RequestException as e:
    # parent of all the above
```

`urllib3.exceptions.MaxRetryError`

```text
urllib3.exceptions.MaxRetryError: HTTPSConnectionPool(host='api', port=443): Max retries exceeded
```

Cause: connection failures repeated past `Retry()` budget. Increase or fix root cause.

```python
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

s = requests.Session()
s.mount("https://", HTTPAdapter(max_retries=Retry(total=5, backoff_factor=0.5)))
```

`httpx`:

```python
import httpx
try:
    r = httpx.get(url, timeout=10)
    r.raise_for_status()
except httpx.ConnectError: ...
except httpx.TimeoutException: ...     # Connect/Read/Write/PoolTimeout
except httpx.HTTPStatusError as e:
    print(e.response.status_code)
```

## SSL / Certificate Errors

`ssl.SSLCertVerificationError: [SSL: CERTIFICATE_VERIFY_FAILED] certificate verify failed: unable to get local issuer certificate`

```text
ssl.SSLCertVerificationError: [SSL: CERTIFICATE_VERIFY_FAILED] certificate verify failed: unable to get local issuer certificate (_ssl.c:1006)
```

Cause: trust store missing the CA. Common on macOS where Python ships with no system CA bridge.

```bash
# macOS — install certifi-managed CAs into Python
/Applications/Python\ 3.12/Install\ Certificates.command
# or
python -m pip install --upgrade certifi

# Find truststore
python -c "import ssl, certifi; print(ssl.get_default_verify_paths()); print(certifi.where())"
```

```python
# Force certifi
import certifi, requests
requests.get(url, verify=certifi.where())

# Use system trust store (3.10+)
import ssl, truststore       # PyPI package
truststore.inject_into_ssl()   # now uses OS keychain / WinINet
```

`ssl.SSLError: [SSL: WRONG_VERSION_NUMBER]`

Cause: speaking HTTPS to a plain-HTTP port (or vice versa), or TLS version mismatch.

```bash
# Diagnose
openssl s_client -connect host:443 -servername host </dev/null
```

`ssl.SSLError: [SSL: TLSV1_ALERT_PROTOCOL_VERSION]` — peer doesn't support your TLS version. Update OpenSSL / Python; very old servers may need `ssl.PROTOCOL_TLSv1_2`.

`ssl.SSLError: [SSL: SSLV3_ALERT_HANDSHAKE_FAILURE]` — usually a cipher mismatch.

```python
# Last resort — disable verification (DANGEROUS, never in prod)
requests.get(url, verify=False)            # warns
import urllib3; urllib3.disable_warnings()
```

## concurrent.futures + multiprocessing + threading errors

`concurrent.futures._base.TimeoutError` — 3.11+ aliased to builtin `TimeoutError`.

```python
from concurrent.futures import ThreadPoolExecutor, TimeoutError as FTimeout
with ThreadPoolExecutor() as ex:
    fut = ex.submit(work)
    try:
        fut.result(timeout=5)
    except FTimeout:
        ...
```

`concurrent.futures.process.BrokenProcessPool`

```text
concurrent.futures.process.BrokenProcessPool: A process in the process pool was terminated abruptly
```

Cause: a worker died (segfault / OOM / `os._exit`). Pool is dead; create a new one.

```python
# Diagnose
PYTHONFAULTHANDLER=1 python script.py    # dump on segfault
```

`AssertionError: daemonic processes are not allowed to have children`

```python
# Cause — Pool inside a multiprocessing daemon worker
# Fix — restructure; daemon processes can't fork further
```

`_pickle.PicklingError: Can't pickle <function X.<locals>.Y>`

```text
_pickle.PicklingError: Can't pickle <function inner at 0x7f...>: it's not found as outer.inner
```

Cause: `multiprocessing` (with the default "spawn" or "forkserver" start method) pickles the target — lambdas, local functions, and methods of unpicklable classes fail.

```python
# Broken
def outer():
    def inner(x): return x*2
    p = Pool().map(inner, range(10))   # PicklingError

# Fix — top-level function
def inner(x): return x*2
p = Pool().map(inner, range(10))

# Or — use cloudpickle / pathos
from pathos.multiprocessing import ProcessPool
```

Start methods:

```text
fork       (default Linux <3.14)  fast; copies whole process; threads/locks broken; not on macOS reliably
spawn      (default macOS, Windows; default Linux 3.14+) clean import in child; slower; everything must be picklable
forkserver  fork from a clean parent; best of both
```

```python
import multiprocessing as mp
mp.set_start_method("spawn", force=True)     # call before any Pool/Process
```

Threading:

```python
# RuntimeError: cannot schedule new futures after interpreter shutdown
# Fix — use the executor as a context manager or shutdown(wait=True)
```

GIL note: CPU-bound work doesn't speed up with threads — use processes or `nogil` builds (3.13+ free-threaded).

## pytest Common Errors

`ImportError while loading conftest 'X/conftest.py'`

```bash
# Diagnose
pytest --collect-only -q
PYTHONPATH=. pytest

# Common fix — pyproject.toml
[tool.pytest.ini_options]
pythonpath = ["src", "."]
```

`fixture 'X' not found`

```python
# Cause — fixture defined in a sibling test module, no conftest
# Fix — move to conftest.py at appropriate scope (project root, package, or test dir)
```

`PytestUnraisableExceptionWarning`

```text
PytestUnraisableExceptionWarning: Exception ignored in: <function X.__del__>
```

Cause: exception raised in `__del__` or other finalizer; harmless but worth fixing.

`ScopeMismatch: You tried to access the function-scoped fixture X with a session-scoped request object`

```python
# Fix — match scopes, or pull function fixture out of session-scoped
```

`pytest.UsageError: file not found: X` — typo in test path.

`Errors in conftest`:

```bash
pytest --capture=no            # show prints during collection
pytest --no-header             # cleaner output
```

Run a single test:

```bash
pytest tests/test_x.py::TestY::test_z
pytest -k "auth and not slow"  # by name expression
pytest -m "smoke"              # by marker
```

## Build / Setup / Packaging Errors

`ERROR: Could not find a version that satisfies the requirement X`

```text
ERROR: Could not find a version that satisfies the requirement requests>=999 (from versions: 0.0.1, ..., 2.32.3)
ERROR: No matching distribution found for requests>=999
```

Causes: typo in name, version doesn't exist, Python too old/new for any wheel, package private (need `--index-url`).

```bash
# Diagnose
pip install --dry-run X
pip index versions X        # if index supports it
pip install X==             # error lists available versions
```

`ERROR: pip's dependency resolver does not currently take into account all the packages that are installed`

Pip prints this on conflicts but still completes; check the warning text for which versions clash. `pip check` audits.

```bash
pip check
pip install pipdeptree && pipdeptree --warn fail
```

`error: Microsoft Visual C++ 14.0 or greater is required` (Windows)

```bash
# Fix — install Build Tools
# https://visualstudio.microsoft.com/visual-cpp-build-tools/
# Or — use a wheel
pip install --only-binary=:all: pkg
```

`error: command 'gcc' failed with exit status 1` (missing dev headers on Linux)

```bash
# Debian / Ubuntu
sudo apt install build-essential python3-dev libssl-dev libffi-dev

# RHEL / Fedora
sudo dnf install gcc python3-devel openssl-devel libffi-devel

# Alpine
apk add build-base python3-dev libffi-dev openssl-dev
```

`ERROR: Failed building wheel for X`

```bash
# Diagnose — read the C compile error above this line
pip install -v X 2>&1 | tee build.log

# Often fixed with system deps; sometimes need older Python
pip install --no-binary=:all: X        # force source build (rarely helpful)
```

`error: externally-managed-environment` (Debian/Ubuntu Python 3.11+)

```bash
# Cause — system Python protected by PEP 668
# Fix — venv or pipx, never sudo pip
python3 -m venv .venv && . .venv/bin/activate
pipx install <tool>
```

## PYTHONPATH / sys.path Issues

The "import works in shell but not in script" diagnosis ladder:

```bash
# 1. Which python?
which python; python -c "import sys; print(sys.executable)"

# 2. Where does it look?
python -c "import sys; print('\n'.join(sys.path))"

# 3. Is the package installed?
python -m pip show <pkg>
python -c "import <pkg>; print(<pkg>.__file__)"

# 4. Editable install pointing at source?
pip install -e .
cat $(python -c "import site; print(site.getsitepackages()[0])")/easy-install.pth

# 5. Conftest rootdir?
pytest --rootdir=. --collect-only

# 6. PYTHONPATH override
echo $PYTHONPATH
unset PYTHONPATH               # clean slate
PYTHONPATH=src python -m mypkg.cli
```

Virtualenv vs system Python:

```bash
# Verify activation
echo $VIRTUAL_ENV
which python                   # should be inside the venv
python -m pip --version        # should show venv path

# Recreate from scratch
deactivate; rm -rf .venv
python3.12 -m venv .venv
. .venv/bin/activate
pip install -U pip
pip install -e .
```

`uv` equivalent:

```bash
uv venv
uv pip install -e .
uv run python -m mypkg
```

## Logging Common Issues

Duplicate log lines — handlers added twice (often from `basicConfig` after a library already added one):

```python
# Broken — repeated import of module that calls basicConfig
import logging
logging.basicConfig(level=logging.INFO)

# Fix — call basicConfig once at top of __main__, or guard
if not logging.getLogger().handlers:
    logging.basicConfig(level=logging.INFO)
```

`basicConfig` only takes effect on the **first** call (unless `force=True` in 3.8+).

```python
logging.basicConfig(level=logging.DEBUG, force=True, format="%(asctime)s %(levelname)s %(message)s")
```

Root logger pollution — every module-level `logging.info(...)` writes to root. Best practice:

```python
log = logging.getLogger(__name__)        # per-module logger
log.info("started")
```

Encoding on Windows console — emit `UnicodeEncodeError` on non-ASCII. Fix:

```python
import sys, logging
handler = logging.StreamHandler(sys.stdout)
handler.setStream(open(sys.stdout.fileno(), "w", encoding="utf-8", closefd=False))
# or PYTHONUTF8=1
```

Logger propagation:

```python
log = logging.getLogger("myapp.sub")
log.propagate = False        # don't bubble up to root (avoids dup if root has handler)
```

Don't use the print/log mix in libraries — let the application configure logging.

## Locale / Encoding on Different OSes

Windows cp1252 vs UTF-8:

```bash
# Symptom — UnicodeEncodeError on print of em-dash
chcp                          # current code page; 65001 = UTF-8
chcp 65001                    # switch to UTF-8 in the current session
PYTHONUTF8=1 python script.py
```

macOS LC_ALL=C breaking pip install:

```bash
# Symptom — pip install crashes with UnicodeDecodeError when reading README
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
pip install pkg
```

The `PYTHONIOENCODING=utf-8` fix (universal):

```bash
export PYTHONIOENCODING=utf-8
```

Docker containers without locale:

```dockerfile
# Debian / Ubuntu base
RUN apt-get update && apt-get install -y locales \
 && sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen \
 && locale-gen
ENV LANG=en_US.UTF-8 LANGUAGE=en_US:en LC_ALL=en_US.UTF-8

# Or — use C.UTF-8 (Debian 11+, no locales pkg needed)
ENV LANG=C.UTF-8 LC_ALL=C.UTF-8

# Alpine — install musl-locales or stick to C.UTF-8
```

3.7+ added the **UTF-8 mode** flag — superset fix:

```bash
PYTHONUTF8=1 python script.py
python -X utf8 script.py
```

## Common "Did You Mean" hints (3.10+)

```text
NameError: name 'lenght' is not defined. Did you mean: 'length'?
NameError: name 'tru' is not defined. Did you mean: 'True'?
AttributeError: 'list' object has no attribute 'lenght'. Did you mean: 'length'?
AttributeError: module 'os' has no attribute 'getev'. Did you mean: 'getenv'?
ImportError: cannot import name 'datetimee' from 'datetime'. Did you mean: 'datetime'?
```

These hints come from a Levenshtein-distance scan over local names, attributes, and module exports. They are best-effort — absence of a hint doesn't mean no near-name exists.

## Common Gotchas — broken→fixed

1. **Mutable default argument**

```python
# Broken — shared across calls
def add(item, items=[]):
    items.append(item)
    return items
add(1); add(2)         # [1, 2]  — surprise

# Fix
def add(item, items=None):
    if items is None: items = []
    items.append(item)
    return items
```

2. **Late binding in closures**

```python
# Broken — every fn returns 4
fns = [lambda: i for i in range(5)]
print([fn() for fn in fns])  # [4,4,4,4,4]

# Fix — capture via default arg or partial
fns = [lambda i=i: i for i in range(5)]
from functools import partial
fns = [partial(lambda i: i, i) for i in range(5)]
```

3. **`is` vs `==`**

```python
# Broken — small ints are interned, but not all
x = 1000
x is 1000              # SyntaxWarning + False on some runs

# Fix — use is for identity (None, True, False), == for equality
if x is None: ...
if x == 1000: ...
```

4. **`from X import *` clobbering builtins**

```python
# Broken
from itertools import *      # imports `count` etc.
sum = 0                      # shadow builtin sum
sum([1,2,3])                 # TypeError: 'int' object is not callable

# Fix — explicit imports, or aliasing
from itertools import count, chain
```

5. **`__init__` calling self.X before X is defined**

```python
# Broken
class Box:
    def __init__(self, n):
        self.fill()         # uses self.items
        self.items = [n]

# Fix — set attributes before calling instance methods
class Box:
    def __init__(self, n):
        self.items = [n]
        self.fill()
```

6. **Floating-point equality**

```python
# Broken
0.1 + 0.2 == 0.3       # False

# Fix
import math
math.isclose(0.1 + 0.2, 0.3, rel_tol=1e-9)
# or use Decimal for money
```

7. **global vs nonlocal scope confusion**

```python
# Broken
total = 0
def add(n): total += n   # UnboundLocalError

# Fix
def add(n):
    global total
    total += n
```

8. **`lru_cache` on instance methods (memory leak)**

```python
# Broken — cache holds self forever, instance never GCs
from functools import lru_cache
class A:
    @lru_cache
    def calc(self, x): ...

# Fix — cache by hashable args, not bound to instance
class A:
    def calc(self, x): return _calc(self.k, x)

@lru_cache
def _calc(k, x): ...

# Or — cachetools.cached + per-instance cache
```

9. **Concurrent dict mutation**

```python
# Broken — RuntimeError mid-iteration
for k, v in d.items():
    if v is None: del d[k]

# Fix
for k in list(d):
    if d[k] is None: del d[k]
# or — comprehension
d = {k: v for k, v in d.items() if v is not None}
```

10. **`asyncio.get_event_loop()` vs `asyncio.new_event_loop()`**

```python
# Broken — 3.10+ deprecates implicit creation in main thread
loop = asyncio.get_event_loop()

# Fix
loop = asyncio.new_event_loop()
asyncio.set_event_loop(loop)
# Best — let asyncio.run() manage it
asyncio.run(main())
```

11. **`subprocess.run` with `check=False` silently swallowing errors**

```python
# Broken — exit code ignored
r = subprocess.run(["pkg", "install"])

# Fix
r = subprocess.run(["pkg", "install"], check=True)
# or inspect explicitly
if r.returncode != 0:
    raise RuntimeError(f"failed: {r.stderr.decode()}")
```

12. **`eval` / `exec` without restricted globals**

```python
# Dangerous
eval(user_input)             # arbitrary code execution

# Restrict — never sufficient for untrusted input
eval(expr, {"__builtins__": {}}, {})

# Better — ast.literal_eval for literals only
import ast
ast.literal_eval("[1, 2, 3]")   # safe
ast.literal_eval("__import__('os').system('rm -rf /')")  # ValueError

# For arithmetic, use a parser, not eval
```

13. **Iterating a generator twice**

```python
# Broken — second loop empty
g = (x*2 for x in range(3))
list(g)                       # [0, 2, 4]
list(g)                       # []  generator exhausted

# Fix — materialize once, or use a function that returns a fresh iterator
data = [x*2 for x in range(3)]
def gen(): return (x*2 for x in range(3))
list(gen()); list(gen())
```

14. **Forgetting `return` in `__exit__` propagates exception (intentional!)**

```python
class CM:
    def __enter__(self): return self
    def __exit__(self, et, ev, tb):
        return True            # suppresses exceptions — usually wrong
        # return False / None  # propagates
```

15. **`open()` without `encoding` is locale-dependent**

```python
open("f.txt").read()           # uses cp1252 on Windows by default
open("f.txt", encoding="utf-8").read()   # explicit
```

## Idioms

`try/except/else/finally` — `else` runs only if no exception; `finally` always runs.

```python
try:
    val = compute()
except ValueError as e:
    log.warning("bad value: %s", e)
    val = 0
else:
    log.info("ok: %s", val)         # only on success
finally:
    cleanup()                       # always
```

Exception chaining with `raise X from Y`:

```python
try:
    parse(text)
except ValueError as e:
    raise ConfigError("invalid config") from e        # chain explicit
    # raise ConfigError("invalid config") from None   # suppress chain
```

Never swallow exceptions silently:

```python
# Bad — invisible failure
try: risky()
except Exception: pass

# Bad — generic except catches BaseException effects too
try: risky()
except: pass

# Good
try:
    risky()
except SpecificError as e:
    log.exception("risky() failed")
    raise                           # re-raise after logging
```

Log at the appropriate level:

```python
log.debug("internal trace")
log.info("normal lifecycle event")
log.warning("recoverable, surprising")
log.error("failure, but service continues")
log.critical("service-level failure")
log.exception("error with traceback")  # use inside except blocks; log.error + traceback
```

Use specific exception classes — define your own for domain errors:

```python
class AppError(Exception): ...
class ValidationError(AppError): ...
class ExternalServiceError(AppError): ...

try:
    do()
except ValidationError as e:
    return 422, e.errors
except ExternalServiceError:
    return 503, "upstream unavailable"
except AppError:
    return 500, "internal"
```

Context manager pattern for cleanup:

```python
from contextlib import contextmanager

@contextmanager
def transaction():
    tx = begin()
    try:
        yield tx
    except:
        tx.rollback(); raise
    else:
        tx.commit()
    finally:
        tx.close()
```

`contextlib.suppress` for "ignore this specific error" (clearer than try/pass):

```python
from contextlib import suppress
with suppress(FileNotFoundError):
    os.remove("/tmp/may-not-exist")
```

`warnings` module — non-fatal version-shifts:

```python
import warnings
warnings.warn("foo() deprecated, use bar()", DeprecationWarning, stacklevel=2)

# Test-time: turn warnings into errors
warnings.simplefilter("error")
PYTHONWARNINGS=error pytest
```

Defensive coding — fail fast at boundaries:

```python
def fetch_user(uid: int) -> User:
    if not isinstance(uid, int) or uid <= 0:
        raise ValueError(f"uid must be positive int, got {uid!r}")
    ...
```

## See Also

- python — language reference, syntax, and stdlib
- polyglot — cross-language idioms
- pdb — Python debugger commands
- pip — package installer
- uv — fast Python package manager
- poetry — Python dependency management
- troubleshooting/javascript-errors — JS/Node error catalog
- troubleshooting/go-errors — Go error catalog

## References

- docs.python.org/3/library/exceptions.html — exception class reference
- docs.python.org/3/tutorial/errors.html — errors and exceptions tutorial
- docs.python.org/3/library/traceback.html — traceback formatting
- peps.python.org/pep-3134 — exception chaining (`raise X from Y`)
- peps.python.org/pep-0657 — fine-grained error locations (3.11+ `^^^^` markers)
- peps.python.org/pep-0479 — change StopIteration handling inside generators
- peps.python.org/pep-0654 — exception groups and `except*`
- peps.python.org/pep-0668 — externally-managed-environment marker
- docs.python.org/3/library/asyncio-exceptions.html — asyncio exception types
- docs.python.org/3/library/warnings.html — warnings framework
- docs.python.org/3/library/logging.html — logging facility
- docs.python.org/3/library/multiprocessing.html — start methods, pickling
- docs.python.org/3/library/ssl.html — SSL/TLS errors and trust stores
- docs.python.org/3/library/codecs.html — codecs and error handlers
- packaging.python.org/en/latest/tutorials/packaging-projects/ — packaging guide
- pip.pypa.io/en/stable/topics/dependency-resolution/ — pip dependency resolver
- docs.djangoproject.com/en/stable/ref/exceptions/ — Django exception classes
- fastapi.tiangolo.com/tutorial/handling-errors/ — FastAPI error handling
- docs.sqlalchemy.org/en/20/core/exceptions.html — SQLAlchemy exception API
- requests.readthedocs.io/en/latest/api/#exceptions — requests exceptions
- www.python.org/dev/peps/pep-3151/ — restructure of OSError hierarchy
