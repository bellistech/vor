# Python (Programming Language)

High-level, dynamically typed language with a vast standard library, expressive syntax, and a thriving ecosystem from web to ML to systems glue.

## Setup, REPL & Versions

### Run a script / one-liner

```bash
# python3 script.py                            # canonical interpreter
# python3 -c 'print("hi")'                    # one-liner
# python3 -m http.server 8000                 # run a stdlib module
# python3 -i script.py                        # run then drop into REPL
# python3 -X dev script.py                    # dev mode: warnings on
# python3 -O script.py                        # strip asserts (-OO also drops docstrings)
```

### REPL

```bash
# python3                                      # basic REPL (3.13+ has multiline editing + colour)
# pip install ipython && ipython              # nicer: tab-complete, magic, %timeit, %debug
# >>> exit()  or Ctrl-D                       # leave the REPL
# >>> _                                       # last expression result
# >>> help(str)                               # interactive help
# >>> dir(obj)                                # list attributes
```

### Interpreter discovery

```bash
# python3 --version                            # 3.12.x ...
# python3 -c 'import sys; print(sys.version)' # full build info
# python3 -c 'import sys; print(sys.executable)' # path to interpreter
# python3 -c 'import sys; print(sys.path)'    # module search path
# which python3 python                         # OS PATH resolution
```

### Virtual environments

```bash
# python3 -m venv .venv                        # create venv in .venv/
# source .venv/bin/activate                    # POSIX shells
# . .venv/bin/activate.fish                    # fish
# .venv\Scripts\activate                       # Windows
# deactivate                                   # leave venv
# python -m venv --upgrade-deps .venv          # bring pip/setuptools to latest
```

### pyenv (multiple Pythons)

```bash
# brew install pyenv                           # macOS
# pyenv install 3.12.4                         # download + build a specific version
# pyenv install --list | grep 3.13             # list available 3.13 builds
# pyenv global 3.12.4                          # default for the user
# pyenv local 3.11.9                           # write .python-version into cwd
# pyenv shell 3.10.14                          # current shell only
# pyenv versions                               # what is installed
```

### uv (fast modern package + venv manager)

```bash
# curl -LsSf https://astral.sh/uv/install.sh | sh   # install uv
# uv venv                                       # create .venv/ (auto-detects Python)
# uv venv --python 3.12                         # pin Python version for venv
# uv pip install requests                       # install a package
# uv pip sync requirements.txt                  # exact lockfile install
# uv add httpx                                  # add to pyproject.toml + lock
# uv run python script.py                       # run inside project venv
# uv tool install ruff                          # install a CLI globally (isolated)
```

### pipx (isolated CLI tools)

```bash
# brew install pipx                             # or: python3 -m pip install --user pipx
# pipx install black                            # CLI in its own venv
# pipx upgrade-all                              # update everything
# pipx list                                     # show installed tools
```

## Variables, Type Hints, and Annotations

### Plain assignment + type hint

```bash
# x = 42                                       # untyped (still int by inference)
# x: int = 42                                  # PEP 526 variable annotation
# pi: float = 3.14159
# name: str = "Alice"
# is_admin: bool = False
# data: list[int] = [1, 2, 3]                  # 3.9+ builtin generics
# config: dict[str, int] = {"x": 1}
```

### `from __future__ import annotations`

```bash
# from __future__ import annotations            # 3.7+: all annotations become strings
# # Now you can forward-reference without quotes:
# class Tree:
#     def parent(self) -> Tree | None: ...      # no quotes needed
# # Annotations are NOT evaluated at runtime — speeds import, breaks runtime introspection
# # tools that rely on get_type_hints(); use typing.get_type_hints() instead of __annotations__.
```

### Final + ClassVar

```bash
# from typing import Final, ClassVar
# MAX: Final[int] = 100                         # checker-enforced "do not reassign"
# class Config:
#     VERSION: ClassVar[str] = "1.0"            # class attribute, not instance
#     name: str                                 # instance attribute (annotation only)
```

### Multiple assignment

```bash
# a = b = c = 0                                 # all three bound to same int
# a, b, c = 1, 2, 3                             # tuple unpacking
# a, b = b, a                                   # idiomatic swap (no temp)
# first, *rest = [1, 2, 3, 4]                   # first=1, rest=[2,3,4]
# *init, last = [1, 2, 3, 4]                    # init=[1,2,3], last=4
```

## Numbers

### Integer

```bash
# n = 42                                        # int — arbitrary precision, no overflow
# big = 2**1000                                 # works fine; ~301 decimal digits
# h = 0xff                                       # hex literal
# o = 0o755                                      # octal
# b = 0b1010                                     # binary
# m = 1_000_000                                  # underscore separator (PEP 515)
# n.bit_length()                                # 6 — minimum bits to represent
# (-7).bit_count()                              # 3.10+ — number of 1 bits
# int("ff", 16); int("0b1010", 0)               # parse with base
```

### Float

```bash
# x = 3.14                                       # 64-bit IEEE-754
# x = 1e-3                                       # 0.001
# x = float('inf'); float('-inf'); float('nan') # special values
# x.is_integer()                                # True if 1.0 / 2.0 / etc.
# round(2.675, 2)                               # 2.67 (banker's rounding to nearest even)
# import math; math.isclose(0.1 + 0.2, 0.3)     # True — handles FP imprecision
# math.isnan(x); math.isinf(x); math.copysign(1.0, -0.0)
```

### Complex

```bash
# z = 2 + 3j                                    # imaginary literal uses j (engineering)
# z.real, z.imag                                # 2.0, 3.0
# abs(z)                                        # magnitude sqrt(13)
# z.conjugate()                                 # 2 - 3j
# import cmath; cmath.sqrt(-1)                  # 1j  (math.sqrt would raise ValueError)
```

### Decimal (exact base-10 arithmetic)

```bash
# from decimal import Decimal, getcontext, ROUND_HALF_EVEN
# Decimal("0.1") + Decimal("0.2")               # Decimal('0.3') exactly
# # NEVER pass float to Decimal — pass strings:
# Decimal(0.1)                                  # 0.10000000000000000555...  (BAD)
# Decimal("0.1")                                # 0.1                          (GOOD)
# getcontext().prec = 50                        # set precision for the context
# (Decimal(1) / Decimal(7)).quantize(Decimal("0.0001"))
```

### Fractions (exact rationals)

```bash
# from fractions import Fraction
# Fraction(1, 3) + Fraction(1, 6)               # Fraction(1, 2)
# Fraction("3.14")                              # Fraction(157, 50)
# Fraction.from_float(0.5)                      # Fraction(1, 2)
# float(Fraction(22, 7))                        # 3.142857142857143
```

### Built-in math operations

```bash
# divmod(17, 5)                                 # (3, 2) — quotient and remainder
# abs(-5); pow(2, 10); pow(2, 10, 1000)         # pow(b, e, m) is modpow
# round(3.7); round(2.5); round(3.5)            # 4, 2, 4 (bankers' rounding!)
# sum([1, 2, 3], start=10)                      # 16
# min([3, 1, 2]); max(1, 2, 3)
# import math
# math.gcd(24, 36); math.lcm(4, 6)              # 12, 12
# math.floor(3.7); math.ceil(3.2); math.trunc(-3.7)  # 3, 4, -3
```

## Strings & f-strings

### Literals & quoting

```bash
# s = "hello"                                   # double quotes
# s = 'world'                                   # single quotes (interchangeable)
# s = """multi
# line"""                                       # triple-quoted = literal newlines
# r = r"C:\path\no\escape"                      # raw — backslashes literal
# b = b"\x00\x01\xff"                           # bytes literal
# u = "café"                                    # str is Unicode by default in Py3
```

### f-string format mini-language

```bash
# name = "Alice"; n = 42; x = 3.14159
# f"{name}"                                     # "Alice"
# f"{name!r}"                                   # "'Alice'"  — repr()
# f"{name!s}"                                   # "Alice"     — str()  (default)
# f"{name!a}"                                   # ascii() — escape non-ASCII
# f"{n:5}"                                      # "   42"     — width 5
# f"{n:05}"                                     # "00042"     — zero-fill
# f"{n:>10}"                                    # right-align (default for nums)
# f"{n:<10}"                                    # left-align
# f"{n:^10}"                                    # center
# f"{n:,}"                                      # "42"        — thousands separator
# f"{1000000:,}"                                # "1,000,000"
# f"{1000000:_}"                                # "1_000_000"
# f"{x:.2f}"                                    # "3.14"      — 2-decimal float
# f"{x:.4e}"                                    # "3.1416e+00" — scientific
# f"{x:.2%}"                                    # "314.16%"   — percent
# f"{255:b}"; f"{255:o}"; f"{255:x}"; f"{255:#x}"  # bin/oct/hex; # adds prefix
# f"{n=}"                                       # "n=42"      — debug form (3.8+)
# f"{x = :.3f}"                                 # "x = 3.142"
```

### String methods

```bash
# s = "  Hello, World  "
# s.strip(); s.lstrip(); s.rstrip()             # whitespace trim
# s.strip(",.!? ")                              # strip a charset (NOT a substring)
# "abc".upper(); "ABC".lower(); "abc".title()   # "ABC", "abc", "Abc"
# "Abc".swapcase(); "abc".capitalize()          # "aBC", "Abc"
# "abc".casefold()                              # aggressive lower (locale-aware: ß→ss)
# "hello world".replace("world", "py")          # "hello py"
# "a,b,c".split(",")                            # ["a", "b", "c"]
# "a, b, c".split(", ", 1)                      # ["a", "b, c"] — maxsplit
# "abc\ndef".splitlines()                       # ["abc", "def"] — handles \r\n too
# ", ".join(["a", "b", "c"])                    # "a, b, c"
# "abc".startswith(("a", "b"))                  # True — accepts tuple of prefixes
# "abc".endswith("c"); "ab" in "abc"            # True
# "abc".find("b"); "abc".index("b")             # 1, 1   (find=-1 if missing, index=raises)
# "abc".count("b")                              # 1
# "Hello".encode("utf-8")                       # b'Hello'
# b'caf\xc3\xa9'.decode("utf-8")                # "café"
```

### String alignment shortcuts

```bash
# "abc".center(11, "-")                         # "----abc----"
# "42".zfill(5)                                 # "00042"
# "abc".ljust(10, "."); "abc".rjust(10, ".")    # "abc......." / ".......abc"
```

### String prefix/suffix removal (3.9+)

```bash
# "test_user".removeprefix("test_")             # "user"
# "page.html".removesuffix(".html")             # "page"
# # Pre-3.9 idiom: s[len(pre):] if s.startswith(pre) else s
```

## Lists

### Construction & basics

```bash
# v = [1, 2, 3, 4, 5]
# v = list(range(10))                            # [0..9]
# v = [0] * 10                                   # ten zeros
# v = list("hello")                              # ['h','e','l','l','o']
# v = []                                         # empty
# len(v); sum(v); min(v); max(v)
# 3 in v; v.count(3); v.index(3)
```

### Indexing & slicing

```bash
# v[0]                                           # first
# v[-1]                                          # last
# v[1:4]                                         # indices 1..3
# v[:3]                                          # first three
# v[-2:]                                         # last two
# v[::2]                                         # every second item
# v[::-1]                                        # reversed copy
# v[1:4] = [10, 20]                              # replace slice with different length OK
# del v[1:4]                                     # delete slice
```

### Mutation

```bash
# v.append(6)                                    # add one
# v.extend([7, 8])                               # add many; equivalent: v += [7, 8]
# v.insert(0, 0)                                 # at index
# v.pop()                                        # remove + return last
# v.pop(0)                                       # remove + return first (O(n))
# v.remove(3)                                    # first occurrence of value (ValueError if absent)
# v.clear()                                      # empty in place
# v.reverse()                                    # mutates
# v.sort()                                       # mutates, ascending
# v.sort(key=lambda x: -x, reverse=True)
# v.sort(key=str.lower)                          # case-insensitive sort of strings
```

### Sort vs sorted

```bash
# v.sort()                                       # in place, returns None
# new = sorted(v)                                # returns a new list, leaves v alone
# new = sorted(v, key=lambda u: u.name)
# # Stable: equal-key items keep relative order
```

### Copying — shallow vs deep

```bash
# a = [1, 2, 3]
# b = a                                          # alias — same list!
# b is a                                         # True
# c = a.copy()                                    # shallow — new list, same elements
# c = a[:]                                        # shallow — same effect, idiomatic
# c = list(a)                                     # shallow
# import copy
# d = copy.deepcopy(nested)                       # recursive — required for nested lists/dicts
```

### List comprehensions

```bash
# [x**2 for x in range(10)]                      # 0,1,4,9,...,81
# [x for x in items if x > 0]                    # filter
# [x*2 for x in items if x > 0]                  # filter + map
# [(x, y) for x in [1,2] for y in [3,4]]         # cartesian
# [x if x > 0 else 0 for x in items]             # ternary inside expression
```

## Tuples

### Basic tuple

```bash
# t = (1, 2, 3)                                  # immutable
# t = 1, 2, 3                                    # parens optional
# t = ()                                         # empty
# t = (1,)                                       # one-element — comma is REQUIRED
# (1)                                            # NOT a tuple — just int 1
# a, b, c = t                                    # unpacking
# a, *rest = t                                   # a=1, rest=[2,3]
# t[0]                                           # access
# t.count(1); t.index(2)
# # Tuples are hashable (if elements are): use as dict keys / set members
```

### namedtuple

```bash
# from collections import namedtuple
# Point = namedtuple("Point", ["x", "y"])
# p = Point(1, 2)
# p.x; p.y; p[0]                                 # field or index access
# p._asdict()                                    # OrderedDict-like dict
# p._replace(x=10)                               # NEW namedtuple, x=10
# Point._fields                                  # ('x', 'y')
# # Lightweight, immutable, tuple-compatible. Useful for hot paths.
```

### typing.NamedTuple (typed)

```bash
# from typing import NamedTuple
# class Point(NamedTuple):
#     x: float
#     y: float
#     label: str = "origin"
# p = Point(1.0, 2.0)
# # Same runtime semantics as namedtuple, but with type hints + class body.
```

## Dictionaries

### Construction & access

```bash
# d = {"name": "Alice", "age": 30}
# d = dict(name="Alice", age=30)                 # kwargs form (str keys only)
# d = dict([("a", 1), ("b", 2)])                 # from iterable of pairs
# d = {k: 0 for k in "abc"}                      # comprehension
# d = {}                                          # empty
# d["name"]                                       # KeyError if absent
# d.get("name"); d.get("name", "default")
# "name" in d                                     # membership
# len(d)
```

### Mutation

```bash
# d["email"] = "a@b.c"
# d.update({"age": 31, "role": "admin"})
# d.setdefault("count", 0)                       # insert if missing, return value
# d.pop("age")                                    # remove + return; KeyError if absent
# d.pop("age", None)                              # safe form
# d.popitem()                                     # remove + return last inserted (LIFO)
# del d["name"]
# d.clear()
```

### Iteration & merging (3.9+)

```bash
# for k in d: ...                                 # iterates keys
# for k, v in d.items(): ...
# for v in d.values(): ...
# d1 | d2                                         # NEW dict, d2 wins on conflict (3.9+)
# d1 |= d2                                        # in-place merge (3.9+)
# {**d1, **d2}                                    # pre-3.9 merge
# {k: v for k, v in d.items() if v}              # comprehension filter
```

### Insertion order (3.7+)

```bash
# # CPython 3.6 made dicts ordered as an implementation detail; PEP 468 (3.7) made it official.
# d = {"a": 1, "b": 2}; d["c"] = 3
# list(d)                                         # ['a', 'b', 'c'] — guaranteed
# # Use OrderedDict only if you need .move_to_end() / .popitem(last=False) semantics.
```

### ChainMap

```bash
# from collections import ChainMap
# defaults = {"timeout": 30, "retries": 3}
# overrides = {"timeout": 5}
# config = ChainMap(overrides, defaults)         # search left to right
# config["timeout"]                               # 5
# config["retries"]                               # 3
# # Writes go to the FIRST dict — useful for layered config (CLI > env > file > default).
```

### defaultdict

```bash
# from collections import defaultdict
# # Group by key:
# groups = defaultdict(list)
# for word in words: groups[len(word)].append(word)
# # Counters from scratch:
# counts = defaultdict(int)
# for x in items: counts[x] += 1
# # Pass a callable, not a value:
# bad = defaultdict([])                           # TypeError — list isn't callable
# good = defaultdict(list)                        # OK
```

### Counter

```bash
# from collections import Counter
# c = Counter("abracadabra")
# c.most_common(3)                                # [('a',5),('b',2),('r',2)]
# c["a"]                                          # 5
# c["z"]                                          # 0 — never raises
# Counter("abc") + Counter("aab")                 # multiset addition
# Counter("abc") - Counter("aab")                 # multiset subtraction (drops <= 0)
```

## Sets & Frozensets

### Set basics

```bash
# s = {1, 2, 3}
# s = set([1, 2, 3])
# s = set()                                       # NOTE: {} is an empty dict, not set
# s.add(4); s.update([5, 6])
# s.discard(2)                                    # no error if missing
# s.remove(2)                                     # KeyError if missing
# s.pop()                                         # remove arbitrary element
# 3 in s; len(s); s.clear()
```

### Set algebra

```bash
# a, b = {1, 2, 3}, {2, 3, 4}
# a | b                                           # union {1,2,3,4}
# a & b                                           # intersection {2,3}
# a - b                                           # difference {1}
# a ^ b                                           # symmetric difference {1,4}
# a <= b                                          # subset
# a < b                                           # proper subset
# a.isdisjoint(b)                                 # no overlap
# # Method form: a.union(b), a.intersection(b), a.difference(b), a.symmetric_difference(b)
```

### Set comprehensions

```bash
# {x % 3 for x in range(10)}                     # {0, 1, 2}
# {w.lower() for w in words if w.isalpha()}
```

### frozenset (immutable, hashable)

```bash
# fs = frozenset([1, 2, 3])
# fs.add(4)                                       # AttributeError
# d = {fs: "value"}                               # frozenset can be a dict key
# # All set algebra methods work; only mutation is forbidden.
```

## Comprehensions & Generator Expressions

### Forms

```bash
# [x for x in v]                                  # list comp — eager
# {x for x in v}                                  # set comp
# {k: v for k, v in pairs}                        # dict comp
# (x for x in v)                                  # generator expression — lazy
# # Generator: lazy, single-pass, constant memory; consumes itself when iterated.
# # Drop parens when passing to a single-arg call:
# sum(x*2 for x in v)                             # no double parens needed
```

### When to choose which

```bash
# # List comp:  you need to index, slice, or iterate twice
# # Set comp:   deduplicating
# # Dict comp:  building lookups
# # Generator:  one-pass, large/streaming, or when you might short-circuit
# any(x > 100 for x in big_list)                  # stops at first match — never builds list
```

## Conditionals & Pattern Matching

### if / elif / else

```bash
# if score >= 90:
#     grade = "A"
# elif score >= 80:
#     grade = "B"
# else:
#     grade = "F"
# grade = "A" if score >= 90 else "F"             # ternary
```

### Truthiness

```bash
# # Falsy:  False, None, 0, 0.0, 0j, "", [], {}, set(), range(0), and zero-length objects
# # Truthy: everything else, including [0], {0}, " "
# if items:                                       # idiomatic "non-empty"
# if items is not None:                           # explicit None check
# # NEVER:  if items == None  (use `is None`)
```

### match / case (3.10+)

```bash
# def describe(x):
#     match x:
#         case 0:
#             return "zero"
#         case int() | float() if x < 0:
#             return "negative number"
#         case [a, b]:
#             return f"two-list: {a}, {b}"
#         case [a, *rest]:
#             return f"head={a}, tail={rest}"
#         case {"type": "user", "name": n}:
#             return f"user {n}"
#         case Point(x=0, y=0):
#             return "origin"
#         case Point(x=x, y=y) if x == y:
#             return f"diagonal at {x}"
#         case _:
#             return "other"
# # Patterns: literal, capture, wildcard _, sequence, mapping, class, OR (|), guard (if).
# # CAPS are TREATED AS LITERALS (constants) only with dotted access (e.g. Color.RED).
# # Bare `RED` would re-bind, not match.
```

## Loops & else clause

### for / while

```bash
# for i in range(10): print(i)
# for i, x in enumerate(v): print(i, x)
# for k, v in d.items(): print(k, v)
# for a, b in zip(xs, ys): ...
# for a, b in zip(xs, ys, strict=True): ...      # 3.10+ — raise if lengths differ
# while cond:
#     ...
#     if done: break
```

### range

```bash
# range(10)                                       # 0..9
# range(2, 10)                                    # 2..9
# range(0, 10, 2)                                 # 0,2,4,6,8
# range(10, 0, -1)                                # 10..1
# list(range(5))                                  # materialise
```

### `else` on loops (oddly underused)

```bash
# # for/else: else runs if loop completed WITHOUT break
# for x in items:
#     if x.matches(): break
# else:
#     raise ValueError("no match")
# # Same on while:
# while attempts:
#     if try_thing(): break
#     attempts -= 1
# else:
#     log.error("ran out of attempts")
```

## Functions

### Definition & signatures

```bash
# def add(a, b): return a + b
# def add(a: int, b: int) -> int: return a + b
# def greet(name: str, greeting: str = "Hello") -> str:
#     return f"{greeting}, {name}!"
# greet("Alice"); greet("Bob", greeting="Hi")
```

### Positional-only / keyword-only

```bash
# def f(a, b, /, c, d, *, e, f):                 # 3.8+
#     ...
# # Before /  -> positional-only (cannot use a=)
# # Between /  and *  -> positional or keyword
# # After *   -> keyword-only (must pass as e=)
# f(1, 2, 3, d=4, e=5, f=6)                      # OK
# f(1, 2, c=3, d=4, e=5, f=6)                    # OK
# f(a=1, ...)                                     # TypeError — a is positional-only
```

### *args / **kwargs

```bash
# def f(*args, **kwargs):
#     # args is a tuple, kwargs is a dict
#     for a in args: ...
#     for k, v in kwargs.items(): ...
# f(1, 2, 3, x=10, y=20)                          # args=(1,2,3) kwargs={'x':10,'y':20}
# # Forwarding:
# def wrapper(*args, **kwargs): return inner(*args, **kwargs)
# # Unpacking on call:
# args = (1, 2); kwargs = {"name": "Alice"}
# greet(*args, **kwargs)
```

### Default-argument trap (the #1 Python footgun)

```bash
# # WRONG — mutable default is created ONCE at def time, shared across calls:
# def append_to(item, target=[]):
#     target.append(item)
#     return target
# append_to(1)                                    # [1]
# append_to(2)                                    # [1, 2]    <-- surprise!
#
# # RIGHT — sentinel + assign inside:
# def append_to(item, target=None):
#     if target is None:
#         target = []
#     target.append(item)
#     return target
```

### Closures & Scope (LEGB)

```bash
# # LEGB: Local, Enclosing, Global, Built-in
# x = "global"
# def outer():
#     x = "enclosing"
#     def inner():
#         # Reads see x="enclosing" via LEGB.
#         # WRITE without nonlocal/global creates a NEW local!
#         return x
#     return inner
# outer()()                                       # "enclosing"
#
# def writer():
#     x = "outer"
#     def inner():
#         nonlocal x                              # rebind enclosing variable
#         x = "rebound"
#     inner()
#     return x
#
# def globwriter():
#     global x
#     x = "rebound at module level"
```

### Late-binding closure trap

```bash
# # WRONG — all lambdas see the SAME `i` variable (the final value):
# fns = [lambda: i for i in range(3)]
# [f() for f in fns]                              # [2, 2, 2]
#
# # RIGHT — bind via default argument (evaluated at def time):
# fns = [lambda i=i: i for i in range(3)]
# [f() for f in fns]                              # [0, 1, 2]
#
# # ALSO RIGHT — partial:
# from functools import partial
# fns = [partial(lambda i: i, i) for i in range(3)]
```

## Lambda & functools

### lambda

```bash
# square = lambda x: x * x                        # one expression only — no statements
# # Almost always prefer `def`. Use lambda for short callbacks:
# sorted(users, key=lambda u: u.age)
# max(items, key=lambda x: x.priority)
```

### functools.reduce

```bash
# from functools import reduce
# import operator
# reduce(operator.add, [1, 2, 3, 4], 0)          # 10
# reduce(lambda a, b: a * b, range(1, 6))        # 120
# # For sums and products, prefer sum() and math.prod() — they're clearer.
```

### partial

```bash
# from functools import partial
# def power(base, exp): return base ** exp
# square = partial(power, exp=2)                  # fixes exp=2
# square(5)                                       # 25
# # Useful for plumbing callbacks with pre-bound config.
```

### lru_cache / cache (memoisation)

```bash
# from functools import lru_cache, cache
# @lru_cache(maxsize=128)
# def fib(n): return n if n < 2 else fib(n-1) + fib(n-2)
# fib.cache_info()                                # CacheInfo(hits=..., misses=..., ...)
# fib.cache_clear()
#
# @cache                                           # 3.9+ — unbounded
# def expensive(x): ...
#
# # Args MUST be hashable. No mutation in/out.
```

### singledispatch

```bash
# from functools import singledispatch
# @singledispatch
# def dump(obj): return repr(obj)
# @dump.register
# def _(obj: list): return ", ".join(map(str, obj))
# @dump.register
# def _(obj: dict): return "; ".join(f"{k}={v}" for k, v in obj.items())
# # Type-dispatched function — Python's "method on a non-method".
```

## Classes

### Plain class

```bash
# class User:
#     def __init__(self, name: str, email: str):
#         self.name = name
#         self.email = email
#
#     def __repr__(self) -> str:                  # what dev sees in REPL / logs
#         return f"User(name={self.name!r}, email={self.email!r})"
#
#     def __str__(self) -> str:                   # what user sees in print()
#         return self.name
#
#     def domain(self) -> str:
#         return self.email.split("@", 1)[1]
```

### @dataclass

```bash
# from dataclasses import dataclass, field
# @dataclass
# class Point:
#     x: float
#     y: float
#     label: str = "origin"
#     tags: list[str] = field(default_factory=list)   # never default=[] !
#
# p = Point(1.0, 2.0)
# # Free: __init__, __repr__, __eq__
# # Options: @dataclass(frozen=True, slots=True, kw_only=True, order=True)
# # frozen=True  -> immutable + hashable
# # slots=True   -> 3.10+, faster + lower memory
# # kw_only=True -> all fields keyword-only
# # order=True   -> generates __lt__/__le__/__gt__/__ge__
```

### __slots__

```bash
# class Point:
#     __slots__ = ("x", "y")
#     def __init__(self, x, y): self.x, self.y = x, y
# # Pros: ~40% less memory per instance, faster attribute access, prevents typos
# # Cons: no __dict__, can't add attributes dynamically; subclasses must also declare slots
# p = Point(1, 2); p.z = 3                        # AttributeError
```

## Properties, descriptors, classmethod, staticmethod

### @property

```bash
# class Celsius:
#     def __init__(self, t): self._t = t
#     @property
#     def fahrenheit(self): return self._t * 9 / 5 + 32
#     @fahrenheit.setter
#     def fahrenheit(self, f): self._t = (f - 32) * 5 / 9
#     @fahrenheit.deleter
#     def fahrenheit(self): del self._t
# c = Celsius(100); c.fahrenheit                  # 212.0
# c.fahrenheit = 32; c._t                          # 0.0
```

### @classmethod & @staticmethod

```bash
# class User:
#     def __init__(self, name): self.name = name
#     @classmethod
#     def from_dict(cls, d): return cls(d["name"])    # cls = the (sub)class
#     @staticmethod
#     def is_valid_email(s): return "@" in s          # plain function, lives in class namespace
# User.from_dict({"name": "A"})                       # User
# User.is_valid_email("a@b")                          # True
```

### Descriptors (advanced)

```bash
# class Validated:
#     def __set_name__(self, owner, name): self.name = name
#     def __get__(self, obj, objtype=None): return obj.__dict__[self.name]
#     def __set__(self, obj, value):
#         if value < 0: raise ValueError(self.name)
#         obj.__dict__[self.name] = value
# class Account:
#     balance = Validated()
# a = Account(); a.balance = 100; a.balance = -1     # ValueError
# # Foundation of @property, @classmethod, ORM fields, etc.
```

## Dunder methods

### Identity, equality, hashing

```bash
# class Point:
#     def __init__(self, x, y): self.x, self.y = x, y
#     def __repr__(self): return f"Point({self.x}, {self.y})"
#     def __eq__(self, other):
#         return isinstance(other, Point) and (self.x, self.y) == (other.x, other.y)
#     def __hash__(self): return hash((self.x, self.y))     # MUST match __eq__
# # Defining __eq__ without __hash__ makes instances unhashable. @dataclass(frozen=True) does both.
```

### Iteration

```bash
# class Range:
#     def __init__(self, n): self.n = n
#     def __iter__(self):
#         self.i = 0
#         return self
#     def __next__(self):
#         if self.i >= self.n: raise StopIteration
#         x = self.i; self.i += 1; return x
# # Or simpler: yield from a generator method
# class Range:
#     def __init__(self, n): self.n = n
#     def __iter__(self): yield from range(self.n)
```

### Context manager

```bash
# class Timer:
#     def __enter__(self):
#         import time; self.start = time.perf_counter(); return self
#     def __exit__(self, exc_type, exc, tb):
#         import time; self.elapsed = time.perf_counter() - self.start
#         return False                            # don't suppress exceptions
# with Timer() as t: do_work()
# t.elapsed
```

### Callable

```bash
# class Adder:
#     def __init__(self, n): self.n = n
#     def __call__(self, x): return x + self.n
# add5 = Adder(5); add5(10)                       # 15
```

### Attribute access

```bash
# class Lazy:
#     def __getattr__(self, name):                # called ONLY when normal lookup fails
#         if name.startswith("get_"): return lambda: name
#         raise AttributeError(name)
# # __getattribute__ intercepts ALL access; rarely needed.
# # __setattr__, __delattr__ for writes/deletes.
```

### Other useful dunders

```bash
# __len__, __bool__, __contains__              # in / len() / truthiness
# __getitem__, __setitem__, __delitem__        # subscript: obj[k]
# __add__, __sub__, __mul__, ...                # arithmetic operators
# __lt__, __le__, __gt__, __ge__                # ordering (or @functools.total_ordering)
# __format__                                     # f"{obj:<spec>}"
# __init_subclass__                              # hook for subclass creation
```

## Inheritance, MRO, super(), ABC

### Inheritance

```bash
# class Animal:
#     def __init__(self, name): self.name = name
#     def speak(self): raise NotImplementedError
# class Dog(Animal):
#     def speak(self): return f"{self.name} barks"
# # Multi-base:
# class C(A, B): ...
```

### super() & MRO

```bash
# class A:
#     def __init__(self): print("A"); super().__init__()
# class B(A):
#     def __init__(self): print("B"); super().__init__()
# class C(A):
#     def __init__(self): print("C"); super().__init__()
# class D(B, C):
#     def __init__(self): print("D"); super().__init__()
# D()                                              # prints D, B, C, A — C3 linearisation
# D.__mro__                                        # (D, B, C, A, object)
# # super() is NOT "the parent class" — it follows the MRO.
```

### Abstract base classes

```bash
# from abc import ABC, abstractmethod
# class Repository(ABC):
#     @abstractmethod
#     def save(self, item): ...
#     @abstractmethod
#     def find(self, id): ...
# class InMemoryRepo(Repository):
#     def save(self, item): ...
#     def find(self, id): ...
# Repository()                                     # TypeError — abstract
# InMemoryRepo()                                   # OK
```

## Modules, packages, imports

### Importing

```bash
# import json
# import os.path
# from collections import OrderedDict, deque
# from typing import Optional, Iterable
# import numpy as np                               # alias
# from . import helpers                            # relative — only inside packages
# from ..utils import tool                         # parent package
# # Avoid: from foo import *  (pollutes namespace)
```

### Package layout

```bash
# myproj/
#   src/myproj/
#     __init__.py            # marks the package; runs on first import
#     __main__.py            # python -m myproj  -> runs this
#     core.py
#     utils/
#       __init__.py
#       text.py
#   tests/
#   pyproject.toml
# # In __init__.py:
# from .core import main_thing
# __all__ = ["main_thing"]                         # what `from myproj import *` exports
# __version__ = "0.1.0"
```

### __main__ guard

```bash
# # script.py
# def main(): ...
# if __name__ == "__main__":
#     main()
# # Lets the file be both importable and runnable.
# # Without it, top-level code fires on `import script` — a footgun for tests.
```

### Module discovery

```bash
# python3 -c 'import sys; print("\n".join(sys.path))'
# # Order: cwd (or script dir), PYTHONPATH, stdlib, site-packages
# # Inside a venv, site-packages points to .venv/lib/pythonX.Y/site-packages.
# python3 -m mymodule.sub                          # run sub as script
```

## Iterators & Generators

### Iterator protocol

```bash
# # An iterable has __iter__ that returns an iterator.
# # An iterator has __next__ that returns next value or raises StopIteration.
# it = iter([1, 2, 3])
# next(it); next(it); next(it)                    # 1, 2, 3
# next(it)                                         # StopIteration
# next(it, "default")                              # provide a sentinel
```

### Generator functions

```bash
# def fib():
#     a, b = 0, 1
#     while True:
#         yield a
#         a, b = b, a + b
#
# from itertools import islice
# list(islice(fib(), 10))                          # [0, 1, 1, 2, 3, 5, 8, 13, 21, 34]
```

### yield from (generator delegation)

```bash
# def flatten(xs):
#     for x in xs:
#         if isinstance(x, list):
#             yield from flatten(x)                 # delegate
#         else:
#             yield x
# list(flatten([1, [2, [3, 4]], 5]))               # [1, 2, 3, 4, 5]
```

### Send / close / throw

```bash
# def echo():
#     while True:
#         x = yield
#         print("got:", x)
# g = echo(); next(g); g.send("hi"); g.close()
# # send() resumes and supplies value of the yield expression.
# # close() raises GeneratorExit inside; throw() raises an arbitrary exception.
```

## Context Managers

### `with`

```bash
# with open("p") as f:
#     data = f.read()
# # File closed automatically — even on exception.
# with open("a") as fa, open("b") as fb:           # multiple in one
#     ...
```

### contextlib.contextmanager

```bash
# from contextlib import contextmanager
# import time
# @contextmanager
# def timer(label):
#     start = time.perf_counter()
#     try:
#         yield
#     finally:
#         print(f"{label}: {time.perf_counter() - start:.3f}s")
# with timer("query"): run_query()
```

### contextlib.ExitStack (dynamic count)

```bash
# from contextlib import ExitStack
# with ExitStack() as stack:
#     files = [stack.enter_context(open(p)) for p in paths]
#     # all closed when the with block exits
```

### Other contextlib helpers

```bash
# from contextlib import suppress, redirect_stdout, closing, nullcontext
# with suppress(FileNotFoundError):
#     os.remove("maybe.txt")
# with redirect_stdout(io.StringIO()) as buf:
#     print("captured")
# with closing(thing_with_close_method) as t: ...
# cm = nullcontext() if not need_lock else lock
# with cm: ...                                     # placeholder context
```

## Errors & Exceptions

### try/except/else/finally

```bash
# try:
#     x = risky()
# except ValueError as e:
#     log.error("bad input: %s", e)
# except (KeyError, IndexError):
#     log.error("missing data")
# except Exception:
#     log.exception("unexpected")                  # auto-includes traceback
#     raise
# else:
#     # runs only if try succeeded
#     use(x)
# finally:
#     # always runs (even on return / break / re-raise)
#     cleanup()
```

### raise & raise from

```bash
# raise ValueError("bad")
# raise ValueError("bad") from None                # suppress the implicit cause
# try: parse(s)
# except ParseError as e:
#     raise MyDomainError("could not parse") from e   # explicit cause chain
```

### Custom exceptions

```bash
# class AppError(Exception): pass
# class ValidationError(AppError):
#     def __init__(self, field, msg):
#         super().__init__(f"{field}: {msg}")
#         self.field = field
# try: ...
# except AppError as e: handle(e)
# # Always inherit from Exception (or one of its subclasses), NEVER BaseException.
```

### Exception groups (3.11+)

```bash
# raise ExceptionGroup("multiple", [ValueError("a"), TypeError("b")])
#
# try:
#     run_concurrent_tasks()
# except* ValueError as eg:
#     for e in eg.exceptions: handle_value(e)
# except* TypeError as eg:
#     for e in eg.exceptions: handle_type(e)
# # `except*` matches a subset and re-raises the rest. Used by asyncio.TaskGroup.
```

### add_note (3.11+)

```bash
# try: parse(s)
# except ValueError as e:
#     e.add_note(f"while parsing line {n}")
#     raise
# # Notes appear in the traceback.
```

## Type System

### Hints & basics

```bash
# def f(x: int, y: float = 1.0) -> str: ...
# def g(items: list[str], lookup: dict[str, int]) -> None: ...    # 3.9+ builtin generics
# from typing import Optional, Union
# def h(x: Optional[int]) -> int: return x or 0                   # int | None
# def i(x: int | str) -> str: ...                                  # 3.10+ union syntax
# # Use `X | None` over `Optional[X]` in modern code.
```

### Protocols (structural typing)

```bash
# from typing import Protocol
# class SupportsClose(Protocol):
#     def close(self) -> None: ...
# def shutdown(x: SupportsClose) -> None: x.close()
# # Anything with a close() method satisfies — no inheritance needed.
# # Add @runtime_checkable to enable isinstance checks.
```

### TypeVar / generics

```bash
# from typing import TypeVar
# T = TypeVar("T")
# def first(xs: list[T]) -> T: return xs[0]
# # 3.12+ syntactic sugar:
# def first[T](xs: list[T]) -> T: return xs[0]
#
# # Bound:
# Number = TypeVar("Number", bound=int | float)
# def sum2(a: Number, b: Number) -> Number: return a + b
#
# # Constrained:
# AnyStr = TypeVar("AnyStr", str, bytes)
```

### ParamSpec (decorator-friendly generics)

```bash
# from typing import ParamSpec, TypeVar, Callable
# P = ParamSpec("P"); R = TypeVar("R")
# def trace(f: Callable[P, R]) -> Callable[P, R]:
#     def wrapper(*args: P.args, **kwargs: P.kwargs) -> R:
#         print(f"call {f.__name__}({args}, {kwargs})")
#         return f(*args, **kwargs)
#     return wrapper
# # Preserves the decorated function's exact signature for the type checker.
```

### Literal & TypedDict

```bash
# from typing import Literal, TypedDict
# Mode = Literal["read", "write", "append"]
# def open_file(p: str, mode: Mode) -> None: ...
# open_file("p", "exec")                          # type checker error
#
# class User(TypedDict):
#     name: str
#     age: int
#     active: bool
# u: User = {"name": "A", "age": 30, "active": True}
# # Use total=False for optional keys, or NotRequired[...] (3.11+) per field.
```

### Generic classes

```bash
# from typing import Generic, TypeVar
# T = TypeVar("T")
# class Stack(Generic[T]):
#     def __init__(self): self._items: list[T] = []
#     def push(self, x: T) -> None: self._items.append(x)
#     def pop(self) -> T: return self._items.pop()
# # 3.12+ syntactic sugar:
# class Stack[T]:
#     def __init__(self): self._items: list[T] = []
```

## asyncio

### async / await basics

```bash
# import asyncio
# async def fetch(n):
#     await asyncio.sleep(1)                       # non-blocking sleep
#     return n
#
# async def main():
#     result = await fetch(1)                      # one at a time
#     # Run concurrently:
#     a, b = await asyncio.gather(fetch(1), fetch(2))
#     # Same, with task objects:
#     t1 = asyncio.create_task(fetch(1))
#     t2 = asyncio.create_task(fetch(2))
#     a, b = await t1, await t2
#
# asyncio.run(main())                              # entry point
```

### TaskGroup (3.11+) — preferred

```bash
# import asyncio
# async def main():
#     async with asyncio.TaskGroup() as tg:
#         t1 = tg.create_task(fetch(1))
#         t2 = tg.create_task(fetch(2))
#     # All tasks awaited at end; if any raises, others are cancelled.
#     # Errors surface as ExceptionGroup.
#     print(t1.result(), t2.result())
# # Replaces the older gather(...) idiom in most cases — better cancellation.
```

### Timeout

```bash
# # 3.11+ context manager:
# async with asyncio.timeout(5.0):
#     await slow_op()
# # Older form:
# await asyncio.wait_for(slow_op(), timeout=5.0)   # TimeoutError on expiry
```

### Queues & events

```bash
# q = asyncio.Queue(maxsize=10)
# await q.put(1); x = await q.get(); q.task_done()
# await q.join()                                   # wait until all marked done
#
# ev = asyncio.Event()
# await ev.wait()                                  # blocks until set
# ev.set(); ev.clear(); ev.is_set()
#
# lock = asyncio.Lock()
# async with lock: ...
```

### Common pitfalls

```bash
# # 1. NEVER call asyncio.run() inside an existing loop. Use create_task or await.
# # 2. await a function that ISN'T async => TypeError. Mark it async or wrap in run_in_executor.
# # 3. Forget to await => returns a coroutine object (RuntimeWarning at GC).
# # 4. CPU-bound work blocks the loop. Offload with loop.run_in_executor(None, fn).
```

## Threading, multiprocessing, the GIL

### threading

```bash
# import threading
# def worker(n): print(f"working on {n}")
# t = threading.Thread(target=worker, args=(1,))
# t.start(); t.join()
# # Lock:
# lock = threading.Lock()
# with lock: shared += 1
# # Threads share memory but the GIL serialises Python bytecode — great for I/O, useless for CPU.
```

### concurrent.futures

```bash
# from concurrent.futures import ThreadPoolExecutor, ProcessPoolExecutor, as_completed
# with ThreadPoolExecutor(max_workers=8) as pool:
#     futures = [pool.submit(fetch, u) for u in urls]
#     for f in as_completed(futures):
#         data = f.result()
# # Same API for ProcessPoolExecutor when CPU-bound.
# with ProcessPoolExecutor() as pool:
#     results = list(pool.map(cpu_heavy, items))
```

### multiprocessing

```bash
# from multiprocessing import Process, Queue
# def worker(q, x): q.put(x * x)
# q = Queue()
# ps = [Process(target=worker, args=(q, i)) for i in range(4)]
# for p in ps: p.start()
# for p in ps: p.join()
# # Bypasses the GIL by forking — but each process has its own memory.
# # Pickling + IPC overhead — measure before assuming it's faster.
```

### The GIL

```bash
# # CPython's Global Interpreter Lock: only one thread executes Python bytecode at a time.
# # Implications:
# #   - I/O-bound workloads scale fine across threads (they release the GIL while waiting)
# #   - CPU-bound workloads do NOT — use multiprocessing or extension code (numpy / C ext)
# # PEP 703: free-threaded Python (3.13+ as opt-in build, --disable-gil).
# #   - Removes the GIL behind a build flag; ABI3 wheels still work but extensions need updates.
# # 3.13 also introduced experimental subinterpreter-based concurrency (PEP 684 / 734).
```

## File I/O

### Text

```bash
# with open("p", "r", encoding="utf-8") as f:
#     content = f.read()
# with open("p", "w", encoding="utf-8") as f:
#     f.write("hi\n")
# with open("p", "a", encoding="utf-8") as f:
#     f.write("appended\n")
# # ALWAYS specify encoding=. Default is platform-dependent (locale on Linux/macOS, cp1252 on Windows).
# # 3.10+: open(...) issues EncodingWarning under -X warn_default_encoding.
```

### Line by line

```bash
# with open("p", encoding="utf-8") as f:
#     for line in f:                               # iterates lazily, one at a time
#         process(line.rstrip("\n"))
# # NEVER:  for line in f.readlines():  -- loads the whole file
```

### Binary

```bash
# with open("p", "rb") as f: data = f.read()      # bytes
# with open("p", "wb") as f: f.write(b"\x00\x01")
```

### pathlib

```bash
# from pathlib import Path
# p = Path("/tmp/data")
# p.mkdir(parents=True, exist_ok=True)
# (p / "file.txt").write_text("hi", encoding="utf-8")
# (p / "file.txt").read_text(encoding="utf-8")
# (p / "file.bin").write_bytes(b"\xff")
# list(p.glob("*.txt"))                            # non-recursive
# list(p.rglob("*.py"))                            # recursive
# p.exists(); p.is_file(); p.is_dir()
# p.stem; p.suffix; p.parent; p.name; p.parts
# p.with_suffix(".bak"); p.with_name("other.txt")
# p.absolute(); p.resolve()                        # resolve symlinks + relative
# p.iterdir()                                      # like os.scandir
# Path.home(); Path.cwd()
```

## JSON, pickle, csv, tomllib

### JSON

```bash
# import json
# data = json.loads('{"k": 1}')                   # str -> obj
# text = json.dumps(data, indent=2, sort_keys=True)
# with open("p") as f: data = json.load(f)
# with open("p", "w") as f: json.dump(data, f, indent=2)
# # Custom encoder for unsupported types:
# class DTEncoder(json.JSONEncoder):
#     def default(self, o):
#         if isinstance(o, datetime): return o.isoformat()
#         return super().default(o)
# json.dumps(obj, cls=DTEncoder)
```

### pickle

```bash
# import pickle
# blob = pickle.dumps(obj)
# obj = pickle.loads(blob)
# with open("p.pkl", "wb") as f: pickle.dump(obj, f)
# with open("p.pkl", "rb") as f: obj = pickle.load(f)
# # WARNING: pickle.loads on untrusted data is RCE. Never load pickles from the network.
# # Use json/messagepack/protobuf for cross-trust boundaries.
```

### csv

```bash
# import csv
# # Read:
# with open("p.csv", newline="") as f:             # newline="" is REQUIRED on Windows
#     for row in csv.DictReader(f):
#         print(row["name"])
# # Write:
# with open("out.csv", "w", newline="") as f:
#     w = csv.DictWriter(f, fieldnames=["name", "age"])
#     w.writeheader(); w.writerow({"name": "A", "age": 30})
# # csv.reader/writer for positional rows; DictReader/Writer for keyed.
```

### tomllib (3.11+, read-only)

```bash
# import tomllib
# with open("pyproject.toml", "rb") as f:
#     cfg = tomllib.load(f)                        # MUST open in binary mode
# # No tomllib.dumps — write with `tomli_w` or `tomlkit` from PyPI.
```

## Regex (`re` module)

### Compile / match / search

```bash
# import re
# re.match(r"\d+", "42abc")                       # at START of string
# re.search(r"\d+", "abc 42 def")                 # anywhere
# re.fullmatch(r"\d+", "42")                      # entire string
# re.findall(r"\d+", "1 2 3")                     # ['1', '2', '3']
# list(re.finditer(r"\d+", "1 2 3"))              # iterator of Match objects
# pat = re.compile(r"\d+")                        # reuse — faster
# pat.findall("1 2 3")
```

### Groups & named groups

```bash
# m = re.search(r"(\w+)@(\w+\.\w+)", "alice@example.com")
# m.group(0)                                       # full match
# m.group(1); m.group(2)                          # alice, example.com
# m.groups()                                       # ('alice', 'example.com')
# m = re.search(r"(?P<user>\w+)@(?P<host>\S+)", s)
# m.group("user"); m["user"]; m.groupdict()
```

### Substitution

```bash
# re.sub(r"\d+", "N", "a1 b22")                   # 'aN bN'
# re.sub(r"(\w+)@(\S+)", r"\2:\1", s)             # backrefs
# re.sub(r"\d+", lambda m: str(int(m.group()) * 2), s)  # callable replacement
```

### Flags

```bash
# re.IGNORECASE / re.I
# re.MULTILINE / re.M                              # ^ and $ match per-line
# re.DOTALL / re.S                                 # . matches \n
# re.VERBOSE / re.X                                # ignore whitespace + #-comments
# re.compile(r"\d+", re.I | re.M)
```

### Common gotchas

```bash
# # 1. Use raw strings (r"...") to avoid double-escaping. r"\n" vs "\n".
# # 2. .match() is anchored at start. Use search() if you don't want that.
# # 3. Greedy vs non-greedy: .* vs .*? — almost always you want non-greedy.
# # 4. Catastrophic backtracking: nested quantifiers (a+)+ on adversarial input.
# # 5. re returns MatchObject or None — check before .group().
```

## Subprocess

### subprocess.run (preferred)

```bash
# import subprocess
# r = subprocess.run(["ls", "-la"], capture_output=True, text=True, check=True, timeout=10)
# r.stdout; r.stderr; r.returncode
# # text=True returns str (utf-8 decoded). Without it, bytes.
# # check=True raises CalledProcessError on non-zero exit.
# # timeout=N raises TimeoutExpired.
```

### Pipes

```bash
# # Pipe two commands without `shell=True`:
# p1 = subprocess.Popen(["ps", "aux"], stdout=subprocess.PIPE)
# p2 = subprocess.run(["grep", "python"], stdin=p1.stdout, capture_output=True, text=True)
# p1.stdout.close()                                # let p1 receive SIGPIPE
# p1.wait()
# print(p2.stdout)
```

### shell=True warnings

```bash
# # DANGEROUS — vulnerable to shell injection if any arg is user-controlled:
# subprocess.run(f"echo {user_input}", shell=True)        # NEVER
# # Safe alternative: pass list, no shell:
# subprocess.run(["echo", user_input])                    # ALWAYS
# # Need pipes? Either (a) Popen-chain like above or (b) shlex.quote each arg if you must use shell:
# import shlex
# subprocess.run(f"cat {shlex.quote(path)} | head -1", shell=True)
```

### check_output / check_call (older)

```bash
# out = subprocess.check_output(["echo", "hi"], text=True)   # captures stdout
# subprocess.check_call(["mkdir", "-p", "x"])                # raises if non-zero
# # Both are convenience wrappers around .run() — newer code should use .run() directly.
```

## Environment & CLI

### sys.argv

```bash
# import sys
# # python3 myscript.py a b c
# sys.argv                                         # ['myscript.py', 'a', 'b', 'c']
# sys.argv[0]                                      # script path
# sys.argv[1:]                                     # actual args
```

### argparse

```bash
# import argparse
# parser = argparse.ArgumentParser(description="Process files")
# parser.add_argument("filename", help="input file")
# parser.add_argument("-o", "--output", default="out.txt")
# parser.add_argument("-v", "--verbose", action="store_true")
# parser.add_argument("-n", "--count", type=int, default=10)
# parser.add_argument("--tag", action="append", default=[])      # --tag a --tag b
# sub = parser.add_subparsers(dest="cmd", required=True)
# build = sub.add_parser("build"); build.add_argument("--release", action="store_true")
# args = parser.parse_args()
# print(args.filename, args.output, args.verbose)
```

### click hint

```bash
# # Third-party: pip install click — decorator-style, common in CLIs:
# import click
# @click.command()
# @click.argument("name")
# @click.option("--count", default=1)
# def hello(name, count):
#     for _ in range(count): click.echo(f"Hello {name}")
# # Also worth knowing: typer (built on click, type-hint-driven).
```

### os.environ

```bash
# import os
# os.environ.get("HOME", "/root")                  # safe read with default
# os.environ["KEY"]                                # KeyError if absent
# os.environ["KEY"] = "value"                      # set in this process + children
# del os.environ["KEY"]                            # unset
# os.getenv("KEY")                                 # alias for os.environ.get
```

## Date & Time

### datetime basics

```bash
# from datetime import datetime, date, time, timedelta, timezone, UTC
# datetime.now()                                   # naive — local time, no tz
# datetime.now(tz=UTC)                             # 3.11+ — aware UTC
# datetime.now(tz=timezone.utc)                    # pre-3.11 form
# datetime.utcnow()                                # DEPRECATED (3.12+) — drops tz
# date.today()
# datetime(2024, 1, 15, 10, 30, tzinfo=UTC)
```

### Parse / format

```bash
# datetime.fromisoformat("2024-01-15T10:30:00+00:00")
# datetime.strptime("2024-01-15", "%Y-%m-%d")
# datetime.now().isoformat()                       # '2024-01-15T10:30:00.123456'
# datetime.now().strftime("%Y-%m-%d %H:%M:%S")
# datetime.fromtimestamp(1700000000, tz=UTC)
# datetime.now(UTC).timestamp()                    # epoch seconds
```

### Timezones with zoneinfo (3.9+)

```bash
# from zoneinfo import ZoneInfo
# tz = ZoneInfo("America/New_York")
# now_local = datetime.now(tz)
# # Cross-tz arithmetic is safe with aware datetimes:
# now_utc = datetime.now(UTC)
# delta = now_utc - now_local                      # timedelta(0) (same instant)
# now_utc.astimezone(ZoneInfo("Europe/Berlin"))    # convert
# # Naive (tzinfo=None) and aware datetimes can't be compared/subtracted — TypeError.
```

### Arithmetic

```bash
# t = datetime.now(UTC)
# t + timedelta(days=7, hours=3)
# t - timedelta(seconds=30)
# (t2 - t1).total_seconds()
# # No "month" or "year" in timedelta — use python-dateutil's relativedelta for that.
```

### time module

```bash
# import time
# time.time()                                      # epoch seconds (float)
# time.monotonic()                                  # for measuring elapsed (won't go backwards)
# time.perf_counter()                               # highest-resolution monotonic clock
# time.sleep(0.5)
```

## Standard Library Highlights

### collections

```bash
# from collections import Counter, defaultdict, deque, OrderedDict, ChainMap, namedtuple
# dq = deque([1, 2, 3], maxlen=100)
# dq.append(4); dq.appendleft(0); dq.pop(); dq.popleft(); dq.rotate(1)
# # deque is O(1) at both ends; list is O(n) at the front.
```

### itertools

```bash
# from itertools import (
#     count, cycle, repeat,                        # infinite
#     accumulate, chain, compress, dropwhile, takewhile,
#     filterfalse, groupby, islice, starmap, tee, zip_longest,
#     product, permutations, combinations, combinations_with_replacement,
#     pairwise, batched,                           # 3.10 / 3.12
# )
# list(chain([1,2], [3,4]))                       # [1,2,3,4]
# list(islice(count(1), 5, 10))                    # [6,7,8,9,10]
# list(combinations("abc", 2))                     # [('a','b'),('a','c'),('b','c')]
# list(product([1,2], [3,4]))                      # [(1,3),(1,4),(2,3),(2,4)]
# list(pairwise([1,2,3,4]))                        # [(1,2),(2,3),(3,4)]
# list(batched("ABCDEFG", 3))                      # [('A','B','C'),('D','E','F'),('G',)]
```

### functools

```bash
# from functools import reduce, partial, lru_cache, cache, cached_property, wraps, total_ordering, singledispatch
# class C:
#     @cached_property
#     def expensive(self):                          # computed once, stored on instance
#         return sum(range(10**6))
# # @total_ordering: define __eq__ + one of __lt__/__le__/__gt__/__ge__; rest are filled in.
```

### enum

```bash
# from enum import Enum, IntEnum, StrEnum, Flag, auto
# class Color(Enum):
#     RED = 1
#     GREEN = 2
#     BLUE = 3
# Color.RED; Color.RED.name; Color.RED.value
# Color(1)                                          # Color.RED
# Color["RED"]
#
# class Perm(Flag):
#     READ = auto(); WRITE = auto(); EXEC = auto()
# Perm.READ | Perm.WRITE
#
# class Mode(StrEnum):                              # 3.11+
#     READ = "r"; WRITE = "w"
# Mode.READ == "r"                                  # True
```

### dataclasses

```bash
# from dataclasses import dataclass, field, asdict, astuple, replace, fields
# @dataclass(slots=True, frozen=True, kw_only=True)
# class Config:
#     host: str
#     port: int = 80
#     tags: list[str] = field(default_factory=list)
# c = Config(host="x"); asdict(c); astuple(c)
# c2 = replace(c, port=443)                         # functional update
```

### statistics

```bash
# import statistics as st
# st.mean([1,2,3,4]); st.median([1,2,3,4]); st.mode([1,1,2,3])
# st.stdev([1,2,3,4]); st.variance([1,2,3,4])
# st.quantiles([1,2,3,4,5], n=4)                    # quartiles
```

### Other essentials

```bash
# import os, sys, io, shutil, glob, tempfile, hashlib, secrets, uuid, base64
# tempfile.NamedTemporaryFile(delete=False)
# hashlib.sha256(b"hi").hexdigest()
# secrets.token_urlsafe(32); secrets.token_hex(16)  # cryptographic random
# uuid.uuid4()
# base64.b64encode(b"hi").decode()
```

## Packaging

### pyproject.toml (PEP 621)

```bash
# [project]
# name = "myproj"
# version = "0.1.0"
# requires-python = ">=3.11"
# dependencies = ["requests>=2.31", "click>=8.1"]
# [project.optional-dependencies]
# dev = ["pytest", "ruff", "mypy"]
# [project.scripts]
# myproj = "myproj.cli:main"
# [build-system]
# requires = ["hatchling"]
# build-backend = "hatchling.build"
```

### Build backends (pick one)

```bash
# # hatchling   — modern, fast, opinionated         (this project uses it)
# # setuptools  — long-standing, very flexible
# # poetry-core — used by poetry-managed projects
# # flit-core   — minimal, single-package
# # pdm-backend — used by PDM
# # maturin     — Rust + Python (pyo3)
# # scikit-build-core — CMake-based C/C++ extensions
```

### Install / build / publish

```bash
# pip install .                                    # install from local source
# pip install -e .                                 # editable install (PEP 660)
# pip install -e .[dev]                             # with optional deps
# python -m build                                   # builds sdist + wheel into dist/
# pipx run twine upload dist/*                      # publish to PyPI
# # Modern alternatives:
# uv pip install -e .[dev]
# poetry install / poetry add / poetry publish
```

### Lockfiles & reproducibility

```bash
# pip freeze > requirements.txt                    # snapshot installed
# pip install -r requirements.txt
# pip-tools: pip-compile pyproject.toml             # generates pinned requirements.txt
# uv lock                                           # generates uv.lock
# poetry lock                                       # generates poetry.lock
# # Always commit a lockfile for application repos. Libraries usually only commit pyproject.toml.
```

## Test, Format, Lint

### pytest

```bash
# pip install pytest
# # tests/test_thing.py
# def test_add(): assert add(2, 3) == 5
# def test_raises():
#     import pytest
#     with pytest.raises(ValueError, match="bad"):
#         f("bad")
# # Run:
# pytest                                            # discover tests/test_*.py
# pytest -k "name and not slow"                     # filter
# pytest -x --pdb                                    # stop on first failure, drop to debugger
# pytest -n auto                                     # parallel (pytest-xdist)
# pytest --cov=myproj                                # coverage (pytest-cov)
# # Fixtures:
# @pytest.fixture
# def db():
#     conn = connect()
#     yield conn
#     conn.close()
# def test_query(db): ...
# # Parametrize:
# @pytest.mark.parametrize("a,b,exp", [(1,2,3),(2,2,4)])
# def test_add(a, b, exp): assert add(a, b) == exp
```

### ruff (format + lint, fast)

```bash
# pip install ruff
# ruff format .                                     # like black + isort
# ruff check .                                       # lints
# ruff check --fix .                                # autofix
# # Configure in pyproject.toml under [tool.ruff]
# # Replaces black + isort + flake8 + pyflakes + pyupgrade in most setups.
```

### mypy / pyright (type checking)

```bash
# pip install mypy
# mypy src/
# # pyproject.toml:
# # [tool.mypy]
# # strict = true
# # python_version = "3.12"
#
# # Or pyright (Microsoft, faster, stricter by default):
# npm install -g pyright
# pyright src/
# # pyrightconfig.json or [tool.pyright] in pyproject.toml.
```

### tox / nox (multi-version testing)

```bash
# # tox.ini:
# # [tox]
# # envlist = py310,py311,py312
# # [testenv]
# # deps = pytest
# # commands = pytest
# tox                                               # runs across all listed Pythons
# # nox is similar but configured in Python (noxfile.py).
```

## Common Gotchas

### Mutable default arguments

```bash
# # WRONG:
# def append(item, target=[]):
#     target.append(item); return target
# append(1)                                         # [1]
# append(2)                                         # [1, 2]    <-- shared!
# # RIGHT:
# def append(item, target=None):
#     if target is None: target = []
#     target.append(item); return target
```

### Late binding in closures

```bash
# # WRONG:
# fns = [lambda: i for i in range(3)]
# [f() for f in fns]                                # [2, 2, 2]
# # RIGHT (default-arg trick):
# fns = [lambda i=i: i for i in range(3)]
# [f() for f in fns]                                # [0, 1, 2]
```

### `is` vs `==`

```bash
# # `is` checks identity (same object). `==` checks equality (same value).
# a = [1, 2]; b = [1, 2]
# a == b                                            # True
# a is b                                            # False — different lists
# # Use `is` for None / True / False / sentinels only.
# x is None                                          # idiomatic
# x == None                                          # works but lints flag it
```

### Integer caching

```bash
# # CPython interns small ints in [-5, 256] — so `is` accidentally works:
# x = 256; y = 256; x is y                          # True   (cached)
# x = 257; y = 257; x is y                          # False  (NOT cached)
# # Don't rely on this. Always use ==.
```

### Division `/` vs `//`

```bash
# 7 / 2                                              # 3.5  — true division (always float in Py3)
# 7 // 2                                             # 3    — floor division
# -7 // 2                                            # -4   — floors toward -inf, not zero!
# import math; math.trunc(-7 / 2)                    # -3   — truncates toward zero
# divmod(-7, 2)                                       # (-4, 1)  — quotient + remainder
```

### Tuple-of-one comma

```bash
# (1)                                                # int
# (1,)                                               # 1-tuple
# 1,                                                 # 1-tuple (no parens needed)
# # Common bug: forgetting the comma when constructing single-element tuples.
```

### Circular imports

```bash
# # a.py: from b import B
# # b.py: from a import A
# # ImportError or partially-initialised module.
# # Fixes: (1) lift shared types into c.py, (2) import inside the function, (3) restructure.
# def use_b():
#     from b import B                                # late import
#     return B()
```

### GIL-bound CPU loops

```bash
# # Threading does NOT speed up pure-Python CPU loops — the GIL serialises bytecode.
# # Use ProcessPoolExecutor, multiprocessing, numpy/numba/cython, or 3.13+ free-threaded build.
```

### Keyword arguments with mutable values

```bash
# def f(**kwargs): kwargs["seen"] = True; return kwargs
# d = {"a": 1}; f(**d); d                           # d unchanged — **d copies
# # But if you assign kwargs back into a shared dict, you mutate it. Be explicit.
```

### `==` on floats

```bash
# 0.1 + 0.2 == 0.3                                   # False  — IEEE-754
# import math; math.isclose(0.1 + 0.2, 0.3)         # True
# # Never compare floats with ==. Use isclose() with explicit tolerances.
```

## Performance Tips

### Use the right structure

```bash
# # list:        O(1) end ops, O(n) front ops
# # collections.deque: O(1) at both ends
# # set / dict:  O(1) membership and lookup
# # tuple:       like list but immutable + slightly less memory + hashable
# # frozenset:   set + hashable
# # Don't `if x in big_list:` in a hot loop. Convert to set() once.
```

### `__slots__`

```bash
# # Saves ~40% memory per instance, prevents __dict__ creation, faster attribute access.
# class Point:
#     __slots__ = ("x", "y")
# # @dataclass(slots=True) adds it for you.
```

### Generators over lists

```bash
# # Lazy — constant memory, short-circuits naturally:
# any(x > 100 for x in big_iter)
# sum(x*x for x in big_iter)
# # Materialise only when you need to index, slice, or iterate twice.
```

### Local-variable lookup is faster than global

```bash
# def hot_loop(items):
#     local_sqrt = math.sqrt           # cache attribute outside the loop
#     return [local_sqrt(x) for x in items]
# # CPython resolves locals via LOAD_FAST (array index) vs LOAD_GLOBAL (dict lookup).
```

### Batch & vectorise

```bash
# # Replace per-element Python loops with numpy / pandas / polars when working on numeric data:
# import numpy as np
# a = np.arange(10_000_000); (a * 2 + 1).sum()      # ~100x faster than a Python loop
```

### Profile before optimising

```bash
# python -m cProfile -o profile.out script.py
# python -m pstats profile.out                       # interactive: sort cumulative; stats 20
# pip install snakeviz; snakeviz profile.out         # flamegraph view
# python -m timeit -s "from m import f" "f()"
# # In notebooks: %timeit, %prun, %lprun (line_profiler).
```

## Idioms (the Pythonic stack)

### EAFP vs LBYL

```bash
# # Easier to Ask Forgiveness than Permission — preferred:
# try:
#     value = d[k]
# except KeyError:
#     value = default
# # Look Before You Leap — readable but racey if anything mutates between checks:
# value = d[k] if k in d else default
# # In a tight inner loop, EAFP is also faster when the happy path dominates.
```

### Duck typing

```bash
# # If it walks like a duck and quacks like a duck...
# def write_all(f, lines): f.writelines(lines)
# # Anything with .writelines satisfies — file, BytesIO, custom class.
# # Use Protocols (typing) to formalise without inheritance.
```

### Use dunder methods to "be" a type

```bash
# class Money:
#     def __init__(self, amount): self.amount = amount
#     def __add__(self, other): return Money(self.amount + other.amount)
#     def __eq__(self, other): return self.amount == other.amount
#     def __hash__(self): return hash(self.amount)
#     def __lt__(self, other): return self.amount < other.amount
#     def __format__(self, spec): return format(self.amount, spec)
# # Now Money instances work with +, ==, sorted(), f"{m:.2f}", set(), dict.
```

### `if __name__ == "__main__":`

```bash
# def main():
#     ...
# if __name__ == "__main__":
#     main()
# # Allows the file to be both imported (no side effects) and executed (runs main).
```

### Walrus operator (3.8+)

```bash
# while (chunk := f.read(8192)):
#     process(chunk)
# if (n := len(items)) > 10:
#     print(f"too many: {n}")
# # Use sparingly — clarity beats cleverness.
```

### Unpacking everywhere

```bash
# a, *rest = [1, 2, 3, 4]
# first, *middle, last = [1, 2, 3, 4, 5]
# def f(*args, **kwargs): ...
# d3 = {**d1, **d2, "extra": 1}
# combined = [*lst1, *lst2, 99]
# *path, ext = filename.rsplit(".", 1)
```

## Tips

- Prefer f-strings; reach for `.format()` only when you need to defer the template.
- `pathlib.Path` over `os.path` for new code — composable, OS-aware, type-safe.
- `subprocess.run([...], check=True)` over `os.system` — never silent failures.
- `dataclass(slots=True, frozen=True)` for value types you pass around.
- `@cache` (3.9+) for memoising pure functions; clear it with `f.cache_clear()`.
- Type hints + `mypy --strict` (or pyright) catch ~30% of refactors before they ship.
- `ruff format && ruff check --fix` replaces black + isort + flake8 + pyupgrade.
- `from __future__ import annotations` for any module that crosses Python versions.
- Use `is None` / `is not None`, never `== None`.
- Generator expressions over list comprehensions when you might short-circuit.
- `with open(...)` always — never rely on GC to close files.
- `subprocess.run(..., shell=False)` and pass a list — always.
- Pin `python_requires` in pyproject.toml — `>=3.11` is a fine modern baseline.
- Lockfile (`uv.lock`, `poetry.lock`, or `pip-compile`) for applications; SemVer ranges for libraries.
- `pytest -x --pdb` to drop into a debugger at the first failure.
- Profile (cProfile + snakeviz) before optimising — Python intuition lies.

## See Also

- polyglot
- javascript
- typescript
- go
- rust
- c
- java
- ruby
- lua
- bash
- regex
- json
- toml
- pytest
- numpy
- pandas
- jupyter
- pip

## References

- [Python Documentation](https://docs.python.org/3/) -- official docs, tutorials, and library reference
- [Python Language Reference](https://docs.python.org/3/reference/) -- formal grammar and semantics
- [Python Standard Library](https://docs.python.org/3/library/) -- every built-in module documented
- [Python Tutorial](https://docs.python.org/3/tutorial/) -- the canonical introduction
- [Python What's New](https://docs.python.org/3/whatsnew/) -- changelog per version (read this for upgrades)
- [Python Glossary](https://docs.python.org/3/glossary.html) -- definitions of Python-specific terms
- [Python Package Index (PyPI)](https://pypi.org/) -- third-party package registry
- [Python Packaging User Guide](https://packaging.python.org/) -- pip, venvs, building & publishing
- [PEP Index](https://peps.python.org/) -- Python Enhancement Proposals
- [PEP 8 -- Style Guide](https://peps.python.org/pep-0008/) -- official coding conventions
- [PEP 20 -- The Zen of Python](https://peps.python.org/pep-0020/) -- `import this`
- [PEP 484 -- Type Hints](https://peps.python.org/pep-0484/) -- foundational typing PEP
- [PEP 526 -- Variable Annotations](https://peps.python.org/pep-0526/)
- [PEP 557 -- Data Classes](https://peps.python.org/pep-0557/)
- [PEP 585 -- Builtin Generics](https://peps.python.org/pep-0585/) -- `list[int]` instead of `List[int]`
- [PEP 604 -- Union Syntax](https://peps.python.org/pep-0604/) -- `int | str` instead of `Union[int, str]`
- [PEP 634 -- Structural Pattern Matching](https://peps.python.org/pep-0634/)
- [PEP 654 -- Exception Groups](https://peps.python.org/pep-0654/) -- 3.11+
- [PEP 695 -- Type Parameter Syntax](https://peps.python.org/pep-0695/) -- 3.12+ generics
- [PEP 703 -- Making the GIL Optional](https://peps.python.org/pep-0703/) -- free-threaded Python
- [Real Python](https://realpython.com/) -- tutorials and guides for all levels
- [Awesome Python](https://github.com/vinta/awesome-python) -- curated library list
- [Python Cookbook (Beazley & Jones)](https://www.oreilly.com/library/view/python-cookbook-3rd/9781449357337/) -- recipe-style reference
