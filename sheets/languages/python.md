# Python (Programming Language)

High-level, dynamically typed language with batteries-included standard library.

## Data Types

### Strings

```bash
# name = "Alice"
# f"Hello, {name}!"                        # f-string
# f"{price:.2f}"                            # format float
# f"{name!r}"                               # repr
# f"{count:>10,}"                           # right-align, comma separator
# "hello".upper(), "HELLO".lower()
# "  hello  ".strip()
# "a,b,c".split(",")                        # ['a', 'b', 'c']
# ", ".join(["a", "b", "c"])                # "a, b, c"
# "hello world".replace("world", "python")
# "hello" in "hello world"                  # True
# name.startswith("Al"), name.endswith("ce")
```

### Numbers

```bash
# x = 42                                    # int
# y = 3.14                                  # float
# z = 2 + 3j                               # complex
# divmod(17, 5)                             # (3, 2) -> quotient, remainder
# abs(-5), round(3.7), pow(2, 10)
# int("42"), float("3.14"), str(42)
```

### Lists

```bash
# nums = [1, 2, 3, 4, 5]
# nums.append(6)
# nums.extend([7, 8])
# nums.insert(0, 0)
# nums.pop()                                # remove last
# nums.pop(0)                               # remove first
# nums.remove(3)                            # remove first occurrence
# nums.sort(), nums.sort(reverse=True)
# sorted(nums, key=lambda x: -x)
# nums[:3], nums[-2:], nums[::2]            # slicing
# len(nums), sum(nums), min(nums), max(nums)
```

### Dictionaries

```bash
# d = {"name": "Alice", "age": 30}
# d["name"]                                 # KeyError if missing
# d.get("name", "default")                  # safe access
# d["email"] = "alice@example.com"           # set
# d.pop("age")                              # remove and return
# d.keys(), d.values(), d.items()
# d.update({"age": 31, "role": "admin"})
# d | {"new": "val"}                        # merge (3.9+)
# {k: v for k, v in d.items() if v}         # dict comprehension
```

### Sets

```bash
# s = {1, 2, 3}
# s.add(4)
# s.discard(2)                              # no error if missing
# s1 | s2                                   # union
# s1 & s2                                   # intersection
# s1 - s2                                   # difference
# s1 ^ s2                                   # symmetric difference
```

### Tuples

```bash
# t = (1, 2, 3)
# a, b, c = t                               # unpacking
# first, *rest = (1, 2, 3, 4)              # first=1, rest=[2,3,4]
```

## Comprehensions

```bash
# [x**2 for x in range(10)]                 # list comprehension
# [x for x in items if x > 0]               # with filter
# {k: v for k, v in pairs}                  # dict comprehension
# {x % 3 for x in range(10)}               # set comprehension
# (x**2 for x in range(10))                 # generator expression
```

## Classes

```bash
# class User:
#     def __init__(self, name: str, email: str):
#         self.name = name
#         self.email = email
#
#     def __repr__(self):
#         return f"User({self.name!r})"
#
#     @property
#     def domain(self):
#         return self.email.split("@")[1]
#
#     @classmethod
#     def from_dict(cls, data):
#         return cls(data["name"], data["email"])
#
#     @staticmethod
#     def validate_email(email):
#         return "@" in email
```

### Dataclasses

```bash
# from dataclasses import dataclass, field
# @dataclass
# class Point:
#     x: float
#     y: float
#     label: str = "origin"
#     tags: list = field(default_factory=list)
```

## Decorators

```bash
# import functools
# def retry(max_attempts=3):
#     def decorator(func):
#         @functools.wraps(func)
#         def wrapper(*args, **kwargs):
#             for attempt in range(max_attempts):
#                 try:
#                     return func(*args, **kwargs)
#                 except Exception:
#                     if attempt == max_attempts - 1:
#                         raise
#         return wrapper
#     return decorator
#
# @retry(max_attempts=5)
# def fetch_data(): ...
```

## Context Managers

```bash
# with open("file.txt", "r") as f:
#     content = f.read()
#
# from contextlib import contextmanager
# @contextmanager
# def timer(label):
#     start = time.time()
#     yield
#     print(f"{label}: {time.time() - start:.3f}s")
#
# with timer("query"):
#     run_query()
```

## Asyncio

```bash
# import asyncio
# async def fetch(url):
#     async with aiohttp.ClientSession() as session:
#         async with session.get(url) as resp:
#             return await resp.json()
#
# async def main():
#     results = await asyncio.gather(
#         fetch("https://api.example.com/a"),
#         fetch("https://api.example.com/b"),
#     )
#
# asyncio.run(main())
```

## Argparse

```bash
# import argparse
# parser = argparse.ArgumentParser(description="Process files")
# parser.add_argument("filename")
# parser.add_argument("-o", "--output", default="out.txt")
# parser.add_argument("-v", "--verbose", action="store_true")
# parser.add_argument("-n", "--count", type=int, default=10)
# args = parser.parse_args()
# print(args.filename, args.output, args.verbose)
```

## Common Standard Library

### os, sys, pathlib

```bash
# from pathlib import Path
# p = Path("/tmp/data")
# p.mkdir(parents=True, exist_ok=True)
# (p / "file.txt").write_text("hello")
# (p / "file.txt").read_text()
# list(p.glob("*.txt"))
# p.exists(), p.is_file(), p.is_dir()
# p.stem, p.suffix, p.parent, p.name
#
# import os
# os.environ.get("HOME", "/root")
# os.getenv("API_KEY")
#
# import sys
# sys.argv, sys.exit(1), sys.stdin, sys.stdout
```

### json

```bash
# import json
# data = json.loads('{"key": "value"}')
# text = json.dumps(data, indent=2)
# with open("data.json") as f: data = json.load(f)
# with open("out.json", "w") as f: json.dump(data, f, indent=2)
```

### subprocess

```bash
# import subprocess
# result = subprocess.run(["ls", "-la"], capture_output=True, text=True, check=True)
# result.stdout, result.returncode
# subprocess.run("echo hello | grep hello", shell=True)
```

### collections

```bash
# from collections import Counter, defaultdict, deque, namedtuple
# Counter("abracadabra").most_common(3)     # [('a',5), ('b',2), ('r',2)]
# dd = defaultdict(list); dd["key"].append(1)
# dq = deque([1,2,3], maxlen=100); dq.appendleft(0)
```

### itertools

```bash
# from itertools import chain, islice, groupby, product, combinations, permutations
# list(chain([1,2], [3,4]))                 # [1,2,3,4]
# list(islice(range(100), 5, 10))           # [5,6,7,8,9]
# list(combinations("abc", 2))              # [('a','b'), ('a','c'), ('b','c')]
```

## Tips

- Use f-strings over `.format()` or `%` for readability and speed.
- `pathlib.Path` is cleaner than `os.path` for all file path operations.
- `subprocess.run(..., check=True)` raises on non-zero exit codes. Use it instead of bare `os.system()`.
- `defaultdict` and `Counter` eliminate most manual key-existence checks.
- Generator expressions `(x for x in ...)` use constant memory vs. list comprehensions `[x for x in ...]`.
- Use `dataclasses` instead of plain `__init__` for simple data containers.
- `functools.lru_cache` memoizes pure functions. Use `@cache` (3.9+) for unbounded caching.
- Type hints (`def f(x: int) -> str:`) are documentation that `mypy` can verify at build time.
