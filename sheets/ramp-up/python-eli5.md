# Python — ELI5

> Python is what happens when you write robot orders in something close to English and a translator turns each line into machine-friendly orders one at a time.

## Prerequisites

- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md) — what a "program" actually is, what user space is, what a process is. You don't strictly need it, but a lot of words ("process", "memory", "syscall") become much easier after you've read it.
- [ramp-up/bash-eli5](bash-eli5.md) — how to type a command into a terminal, how to know which folder you're in. Not required, but if you have never opened a terminal in your life, that sheet first will save you a lot of confusion.

If you have never read a single line of code, that's fine. Stay. We will explain every word. Every weird name has a one-line definition in the **Vocabulary** table near the bottom. If a word feels weird, that is your cue to scroll down, glance at the table, and come back. Nothing in this sheet expects you to already know anything.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath it that don't have a `$` are what your computer prints back at you. We call that "output."

If you see `>>>` at the start of a line, that means you are inside Python's interactive shell. The Python prompt is `>>>`. You don't type `>>>` either. We will see how to start that shell in the Hands-On section.

## What Even Is Python

### A long, slow story before any code

Imagine you have a robot. Not a sci-fi robot from a movie. Imagine a really dumb robot, the kind that only does what you tell it, exactly. It has arms. It has legs. It has a couple of grippers. It has a tiny brain that can only follow really, really simple orders.

The orders the robot understands are awful. They look like this:

```
LIFT_LEFT_GRIPPER 23 millimeters
ROTATE_LEFT_WRIST 12 degrees clockwise
CLOSE_LEFT_GRIPPER 90 percent
WAIT 100 milliseconds
LIFT_LEFT_ARM 50 millimeters
```

That is the only language the robot understands. We call this **machine code**. Every computer in the world only really understands its own version of this — long lists of tiny, dumb orders. Move this number into that slot. Add these two slots. Jump to that line. Each order is so small you can barely tell what the program is supposed to do by looking at the orders.

Now imagine that you, a human, want the robot to make a sandwich. You don't want to write out two thousand wrist-rotations. That would take you all day. You'd lose track. You'd make mistakes. You'd cry. So you hire a translator. You write down the recipe in plain English:

```
Take two slices of bread.
Spread peanut butter on one slice.
Spread jelly on the other slice.
Press them together.
Cut on a diagonal.
```

You hand the recipe to the translator. The translator reads each line, turns it into the long boring list of wrist-rotations and gripper-squeezes, and tells the robot what to do. You wrote five lines. The translator turned them into five thousand machine orders. The robot makes you a sandwich. Everybody is happy.

**Python is the recipe.** Python is the close-to-English language you write your wishes in.

**The Python interpreter is the translator.** It reads each line of your Python file, turns it into the boring tiny machine-friendly orders, and runs them.

That is the whole idea. Everything else in this sheet is detail.

### Why is it called "Python"

The man who made Python is named Guido van Rossum. He was a fan of a British comedy show called *Monty Python's Flying Circus*. He thought "Python" sounded short and a bit silly and would be fun. The language has nothing to do with snakes. The logo has snakes on it because everybody assumed snakes after the fact, but the name was a comedy show.

You will see jokes about Spam, parrots, and dead-but-not-dead birds in Python documentation. Those are all *Monty Python* references. If you don't get them, that's totally fine. They're just goofs from the people who built the language.

### Why do people love Python

A few reasons people pick Python over other languages:

- **It reads like English.** A line of Python often looks like a sentence. `if temperature > 30: open_window()` reads almost the way you'd say it.
- **You skip the build step.** In a lot of languages, you have to compile your code into a binary first, then run the binary. In Python, you just say "run my file" and Python runs it. The translator is built into Python itself.
- **Tons of libraries.** Almost any thing you might want to do — read a CSV file, talk to a database, draw a chart, build a website, train a neural network — somebody has already written a Python library for it. You don't write it from scratch.
- **It's everywhere.** Sysadmins use it for scripts. Scientists use it for math. Web developers use it for backends. Game developers use it for tools. Data scientists use it for everything. ML engineers use it for everything. It is the most common first language taught at universities.

A few reasons people complain about Python:

- **It's slow.** A line of Python takes the translator longer to handle than the same line in a faster language like C or Rust. If you write a math-heavy program in pure Python it will probably be 10x to 100x slower than the same program in C. People work around this by calling fast libraries (NumPy, PyTorch) which were themselves written in C, but pure Python is not fast.
- **The GIL.** We will explain this later. Short version: Python kind of can't really run two pieces of Python code on two CPU cores at the same time, because of a thing called the Global Interpreter Lock. There are ways around it. We'll cover them.
- **Whitespace is meaningful.** In Python, you don't use `{` and `}` to group code. You use indentation. If you mess up the indentation, your program breaks in weird ways. Most people get used to it. Some people hate it.

### The recipe again, but real this time

Here is a real Python file. It looks like this:

```python
# greet.py
name = "Sandwich"
print("Hello,", name)
```

Three lines.

Line 1 is a comment. The `#` means "ignore the rest of this line, it's just a note for humans." Comments are free; the translator skips them.

Line 2 says "make a name called `name` and stick the text `\"Sandwich\"` to it."

Line 3 says "print the words `Hello,` and then whatever `name` is pointing at."

If you save that as `greet.py` and run `python greet.py` in a terminal, your computer prints:

```
Hello, Sandwich
```

That's it. That is a complete Python program. You don't need a `main` function. You don't need to declare types. You don't need to compile anything. You just write the lines and run the file.

### Where Python lives

When you "install Python" on your computer, you are installing a few things at once.

- **The interpreter** itself. This is a binary called `python` (or `python3` on a lot of systems). It is the translator.
- **The standard library.** This is a giant folder of pre-written Python code that comes for free with Python. Code for reading files, talking to the network, doing math, parsing JSON, picking dates apart, and a thousand other things. You don't have to install any of it; it just works the moment Python is installed. The Python team calls this "batteries included."
- **`pip`**, the tool that lets you install more libraries from the internet. We'll cover it later.

When you run `python myfile.py`, the interpreter reads `myfile.py` from your hard drive, translates it on the fly, and runs the result.

## Interpreter, Bytecode, .pyc Files

This is where we explain what the translator actually does, in slow motion.

### The pipeline

When you say `python greet.py`, here is roughly what happens, from your file going in to the computer doing things.

```
+---------------+      +-----------+      +-------------+      +------------+
|  source code  |  ->  |    AST    |  ->  |  bytecode   |  ->  | eval loop  |
| (your .py)    |      |  (tree)   |      | (.pyc tape) |      | (executes) |
+---------------+      +-----------+      +-------------+      +------------+
       |                     |                   |                    |
       |                     |                   |                    |
   text on disk        parsed shape         tiny opcodes         CPU work
```

Five stages:

1. **Source code.** Your `.py` file. Plain text. Humans wrote it. Looks like English.
2. **Parsing.** Python reads the text and turns it into an **AST** — short for Abstract Syntax Tree. The AST is the shape of your program: `assignment` here, `function call` there, `if-statement` over there. The parser doesn't care what the program does; it just figures out the shape.
3. **Bytecode compilation.** The AST gets walked and turned into **bytecode**. Bytecode is a long list of tiny tape-instructions: LOAD this, STORE that, ADD these two, CALL that function. There are about 200 of these little instructions. They are not the actual machine code your CPU runs, but they are way smaller and simpler than your source.
4. **`.pyc` cache.** Python writes the bytecode to a hidden folder called `__pycache__` next to your file. The file inside is named something like `greet.cpython-313.pyc`. Next time you run the same `.py` file, if it hasn't changed since, Python skips the parse and bytecode steps and just loads the `.pyc` straight from the cache. That's why a second run of the same script feels a tiny bit snappier.
5. **Eval loop.** The interpreter has a giant loop inside called the **eval loop** (sometimes "the bytecode dispatcher"). It picks the next bytecode instruction off the tape, does what it says, picks the next one, does what it says, picks the next one. Forever, until your program ends. That's the engine room.

### The thing called CPython

When you say "Python," 99 times out of 100 you mean **CPython**. CPython is the version of the Python interpreter written in the C language. It is the official one. It is the one you get when you go to python.org and click download. It is what `python3` on Linux is. It is what `python` from Homebrew is. When this sheet says "Python", it means CPython.

There are other versions:

- **PyPy** — a faster Python interpreter, written in a special variant of Python itself. It uses a JIT (just-in-time compiler) to make hot loops much faster. Compatible with most Python code, but a separate binary.
- **Jython** — Python that runs on the Java Virtual Machine. Old, mostly historical.
- **IronPython** — Python that runs on .NET. Also mostly historical.
- **MicroPython** — a tiny Python that runs on microcontrollers (small chips with kilobytes of RAM). Used for hobby electronics.
- **GraalPy** — Python that runs on Oracle's GraalVM. Newer, experimental.

You can check which one you've got with `python -c 'import sys; print(sys.implementation.name)'`. If it says `cpython`, you're on the official one.

### What "interpreter" really means

Old languages like C have a **compiler**. The compiler reads your whole source file, turns the whole thing into machine code, writes a binary, and you're done. Then you run the binary later. The compiler is one program. The binary is another.

Python doesn't usually work that way. Python has the parsing-and-compiling step built into the same binary as the running step. You don't end up with a separate `.exe`; you end up with `.pyc` files in `__pycache__`, but those are not standalone binaries — they only mean something to a Python interpreter. So we say Python is **interpreted**, not compiled. (Technically, Python compiles to bytecode and the bytecode is interpreted, but for an ELI5 view: think of Python as not needing a separate build step.)

### A picture of the bytecode

Here is the Python source:

```python
def add(a, b):
    return a + b
```

If you ask Python "show me the bytecode," it looks like this:

```
LOAD_FAST     a
LOAD_FAST     b
BINARY_OP     +
RETURN_VALUE
```

Four tiny tape-instructions. `LOAD_FAST` means "push the value of this local variable onto a little stack." `BINARY_OP` means "pop the top two things off the stack, apply the operation, push the result." `RETURN_VALUE` means "send the top of the stack back to whoever called us."

You can see this for any function in Python by importing the `dis` module ("dis" is short for "disassemble"):

```python
>>> import dis
>>> def add(a, b): return a + b
>>> dis.dis(add)
```

You will see the actual list of bytecode instructions. It is one of the more magical things you can do as a beginner — peek under the hood at what your code becomes. Don't worry if you can't read it. You don't need to. But it's good to know it's there.

### Why care about `.pyc` files

You will sometimes notice a `__pycache__` folder appear next to your Python files. Don't worry about it. Don't delete it (well, you can, Python will just recreate it the next time). Don't commit it to git — every Python project has `__pycache__/` in its `.gitignore`. The folder is just a cache to make the next run a bit faster.

If you ever see a stale `.pyc` causing weird behaviour, you can blow away every cache in your project with:

```
$ find . -name __pycache__ -type d -exec rm -rf {} +
```

Doing that is fine. Python will rebuild what it needs.

## Names and Objects

This is the single most important section of the sheet. If you only really understand one thing about Python, understand this.

### Everything is an object

In Python, **everything you can poke at is an object.** Numbers are objects. Strings are objects. Lists are objects. Functions are objects. Classes are objects. Modules are objects. The `None` thing is an object. Even `type` itself is an object.

An **object** is just a glob of stuff in memory with three things attached to it:

1. A **type** (am I a number, a string, a list, a function, a thing-that-Steve-defined?)
2. A **value** (what data am I actually carrying?)
3. An **identity** (a unique number that tells you which exact object this is, even if there are two with the same value).

You can ask Python any of those things at any time:

```python
>>> x = 7
>>> type(x)
<class 'int'>
>>> x
7
>>> id(x)
4302718192
```

Type, value, identity. Every object has all three.

### Names are not boxes

Now the trick. In a lot of older languages, you imagine variables as **boxes**. You have a box called `x` and you put `7` inside it. If you say `x = 8`, you go to the same box and replace what's inside.

**Python does not work that way.** In Python, names are not boxes. Names are **labels**. Names are **sticky notes that point at objects.**

Picture an empty room with a bunch of objects floating around. Numbers, strings, lists. Each object has a unique identity number. When you do `x = 7`, what really happens is:

1. Python finds (or creates) the object whose value is `7`.
2. Python takes a sticky note that says `x` and slaps it onto the `7` object.

When you say `y = x`, Python does **not** make a new copy. It just slaps another sticky note (`y`) onto the same `7` object. Now there are two sticky notes on the same object.

When you say `x = 8`, the sticky note `x` peels off the `7` object and gets stuck on the `8` object. The `7` object is still floating there with the `y` sticky note still on it. The `7` doesn't change. The label moved.

```
                Before:
                x ---+
                     v
                 [ object: 7 ]  <--- y (also pointing here)


                After x = 8:
                x ---+
                     v
                 [ object: 8 ]

                 [ object: 7 ]  <--- y (still pointing here)
```

This is **assignment binds names to objects.** That is the actual rule. The right-hand side evaluates to an object. The left-hand side becomes a name pointing at that object. We say "assignment is binding," and you'll hear that word a lot.

### Why this matters: lists

Let's see why this trips people up. Try this:

```python
>>> a = [1, 2, 3]
>>> b = a
>>> b.append(4)
>>> a
[1, 2, 3, 4]
```

Wait — we appended to `b`, but `a` changed too. Why?

Because `b = a` did not make a copy of the list. It put a second sticky note on the same list object. There is one list. Two names point at it. When you mutate the list through `b`, you are mutating the same list `a` is also pointing at. Of course `a` sees the change. It's the same list.

If you actually wanted a copy, you have to make one explicitly:

```python
>>> a = [1, 2, 3]
>>> b = a.copy()         # or list(a), or a[:]
>>> b.append(4)
>>> a
[1, 2, 3]
>>> b
[1, 2, 3, 4]
```

Now there are two list objects. Two sticky notes, two lists, two stories.

This is the source of more beginner bugs than any other single thing in Python. Internalize it: **`=` does not copy, it binds.**

### `is` versus `==`

There are two ways in Python to ask "are these the same?"

- `==` asks: "do these two objects have the same value?"
- `is` asks: "are these two names pointing at the *same object*?" (Same identity, same `id()`.)

Most of the time you want `==`. The exception is comparing to `None`: you should write `if x is None:`, not `if x == None:`. `None` is a singleton (there is only one `None` object in the entire interpreter), so `is` is the right tool for it.

### Scope: where Python looks for names

When Python sees a name like `x`, it doesn't just guess where to find it. It searches in a specific order. The order is called **LEGB**.

```
   L  --  Local       (the function you're inside)
   E  --  Enclosing   (any outer function wrapping this one)
   G  --  Global      (the module / file you're in)
   B  --  Built-in    (Python itself: print, len, range, ...)
```

Python checks Local first. If `x` isn't a local name, it climbs out one layer to Enclosing. If still not found, it goes to Global. Finally, it falls through to Built-in. If Python runs all four and still can't find the name, you get a `NameError`.

```
        +-----------------------+
        |  built-ins (B)        |  print, len, list, dict, ...
        |   +-----------------+ |
        |   |  globals (G)    | |  module-level names
        |   |  +-----------+  | |
        |   |  | enclosing | <----- outer function
        |   |  | +-------+ |  | |
        |   |  | | local | |  | |  inner function
        |   |  | +-------+ |  | |
        |   |  +-----------+  | |
        |   +-----------------+ |
        +-----------------------+
```

You can override LEGB in two ways: `global x` says "when I write to `x`, write to the global one." `nonlocal x` says "when I write to `x`, write to the enclosing-function's `x`, not a new local."

### Mutable vs immutable: the second-most-important rule

Python objects come in two flavors.

**Immutable** objects cannot be changed after they are made. If you want a new value, you have to make a new object. Examples:

- `int` — `7` is `7` forever. You can't sneak inside it and change it to `8`.
- `float` — same.
- `str` — `"hello"` is `"hello"` forever. There is no way to change a character in place.
- `tuple` — `(1, 2, 3)` is fixed. You can't append.
- `frozenset` — same as a set, but can't add or remove.
- `bytes` — like a string of bytes, but immutable.

**Mutable** objects can be changed in place. Same object, new contents. Examples:

- `list` — you can `.append`, `.pop`, change items.
- `dict` — you can add and remove keys.
- `set` — you can `.add` and `.remove`.
- `bytearray` — like `bytes`, but mutable.
- Most user-defined classes you write yourself.

Why is this such a big deal? Because of the sticky-note thing above. If two names point at the same mutable object, changing it through one name changes it for both. If two names point at the same immutable object, you can't change it at all — you can only rebind one of the names to a different object.

This is why functions in Python should rarely take a list as a default argument. We will see that trap in the **Common Confusions** section — it bites everyone exactly once.

## The GIL (Global Interpreter Lock)

Now the elephant in the room. Threading in Python.

### What the GIL actually is

CPython has, deep inside, a single big lock. It is called the **Global Interpreter Lock**, the **GIL** for short. The GIL makes a simple promise:

> Only one thread can be running Python bytecode at a time.

That's it. That's the whole rule.

Imagine the eval loop (the engine room from earlier). The GIL is a key. There is exactly one key. Whichever thread holds the key gets to run Python bytecode. Other threads wait. The thread holding the key runs for a tiny bit, drops the key (every few milliseconds), and another thread might pick it up.

```
   thread A --(holds GIL)-- runs bytecode for ~5ms
                |
                v  drops GIL
   thread B --(holds GIL)-- runs bytecode for ~5ms
                |
                v  drops GIL
   thread A --(picks GIL back up)-- runs bytecode for ~5ms
```

So even if your laptop has 8 CPU cores, **plain Python threads will not give you 8x the speed for CPU-bound work.** Only one is really running Python at any given moment.

### Why does the GIL exist

CPython's memory model is built on **reference counting** (we'll cover that in a minute). Reference counts have to be updated every time a name is bound, an object is passed around, or an object is dropped. If two threads tried to update a reference count at the same time on the same object, you'd get a race condition: the count goes wrong, and the object either gets freed too soon (crash) or never (memory leak). Putting a tiny lock on every single reference count would slow CPython to a crawl. So instead they put one big lock on the whole interpreter and called it a day. Simple, fast for single-threaded code, terrible for multi-core CPU work.

### When the GIL hurts

The GIL hurts when your work is **CPU-bound**. That means: your program is doing math, image processing, sorting big lists, anything where the bottleneck is "the CPU is busy thinking." In that case, threads in Python will not help you. They might even slow things down because the GIL handoff has overhead.

### When the GIL doesn't hurt

The GIL is released during **I/O**. I/O means input/output: reading from a file, talking to the network, waiting for a database. While a thread is waiting on I/O, it drops the GIL so other threads can run. So if your program is **I/O-bound** — like a web scraper that's mostly waiting on network — threads work great.

### Ways around the GIL

- **`multiprocessing`**: instead of threads, use multiple **processes**. Each process has its own Python interpreter and its own GIL. They run in parallel for real. The downside is processes are heavier (more memory) and don't share data as easily.
- **C extensions**: libraries like NumPy and PyTorch do their hot work in C, and the C code can release the GIL while it runs. So even though you're calling NumPy from a single Python thread, NumPy might be using all 8 cores under the hood.
- **`asyncio`**: doesn't get rid of the GIL, but lets one thread juggle thousands of I/O tasks. Great for network-heavy apps.
- **Free-threaded Python (3.13+)**: starting in Python 3.13, there is a special build of CPython called the **free-threaded build** (also called PEP 703 / "no-GIL build"). It removes the GIL entirely. It is opt-in for now (you have to download a special build), and many libraries don't fully support it yet. Over the next few years this will likely become the default, but for now it's an experimental option.
- **PyPy** has a GIL too, but for many workloads PyPy is so much faster on a single thread that it doesn't matter.

### The two-line summary

If you're doing math: use multiprocessing or C-backed libraries. If you're doing I/O: threads or asyncio are fine. The GIL is only a problem when you're doing pure-Python math on multiple threads at once.

## Reference Counting + Cycle Collector

How Python manages memory.

### The simple part: reference counts

Every Python object has a hidden number attached to it called its **reference count**. The number is "how many sticky notes are currently pointing at me." When you do `x = [1, 2, 3]`, the new list object is born with a refcount of 1 (because `x` points at it). If you then do `y = x`, the list's refcount goes up to 2 (now `x` and `y` both point at it). If you do `del x`, the refcount drops back to 1. If you also do `del y`, the refcount drops to 0, and at that exact moment Python frees the memory.

```
   x = [1,2,3]      list object refcount = 1   [x]
   y = x            list object refcount = 2   [x, y]
   del x            list object refcount = 1   [y]
   del y            list object refcount = 0   <-- freed immediately
```

This is **deterministic**. The instant the count hits zero, the object is gone. You don't have to wait for any garbage-collector to come along later. Python knows immediately that nothing is using this thing anymore, so it cleans up.

Most of Python's memory is freed this way — quickly, immediately, no fuss.

### The annoying part: cycles

Reference counting has one flaw. Cycles.

Imagine you make two lists, and each list contains a reference to the other list:

```python
a = []
b = []
a.append(b)
b.append(a)
del a
del b
```

After `del a` and `del b`, are the lists gone?

Not by reference counting. The first list still has a refcount of 1 because the second list is pointing at it. The second list still has a refcount of 1 because the first list is pointing at it. They're both pointing at each other. Reference counting will never bring either count to zero. The lists are **leaked**.

```
        +-------+         +-------+
        |   a   |  --->   |   b   |
        |       |  <---   |       |
        +-------+         +-------+
        refcount 1        refcount 1
        (but no outside name points at either)
```

This is called a **reference cycle**. To clean up cycles, Python runs a second mechanism: a **cycle collector** (also called the cyclic garbage collector, or just "the gc"). It periodically wakes up, looks for islands of objects that only point at each other, and frees them.

You can poke the cycle collector through the `gc` module:

```python
import gc
gc.collect()        # run a full collection right now
gc.disable()        # stop running automatic collections
gc.enable()
gc.get_count()      # see how many objects are tracked at each generation
```

Most of the time, you ignore it. The cycle collector runs on its own. You only think about it if you're profiling a memory leak or writing weird metaclass tricks.

### What about __del__

Some objects have a special `__del__` method that runs when the object is freed. It's like a destructor. Don't rely on it. The exact moment it fires depends on refcounts and cycles, and if you mix `__del__` with cycles, the cycle collector used to refuse to clean up such cycles. (This was fixed in Python 3.4, but `__del__` is still a smell.) For "I want to clean up when I'm done with this thing," use a context manager (`with` statement) instead. We'll cover those.

## Mutable vs Immutable

You met this in the names-and-objects section. Let's lay it out as a table because you will look back at this.

| Type | Mutable? | Hashable? | Notes |
|------|----------|-----------|-------|
| `int`, `float`, `complex`, `bool` | no | yes | numbers are immutable |
| `str` | no | yes | strings are immutable; "modifying" makes a new string |
| `tuple` | no | yes (if all items are) | like a list but frozen |
| `frozenset` | no | yes | frozen version of a set |
| `bytes` | no | yes | immutable byte string |
| `list` | yes | no | the workhorse mutable sequence |
| `dict` | yes | no | hash map, key-value |
| `set` | yes | no | unordered unique collection |
| `bytearray` | yes | no | mutable byte string |

A few things to take away:

**Hashable means it can be a dict key or a set member.** You cannot use a list as a dict key, because lists are mutable and Python doesn't want the key to change out from under it. You can use a tuple as a dict key (as long as the tuple's items are hashable too).

**"Modifying" an immutable thing makes a new one.** When you write `s = "hi"; s += "!"`, you don't change `"hi"` — you build a new string `"hi!"` and rebind `s` to it. This matters for performance: building a string by `+=`-ing in a loop is quadratic. Use `"".join(parts)` instead.

**Default arguments evaluate once.** This is *the* Python beginner trap, and it has bitten everyone. Look at this:

```python
def add(item, bag=[]):
    bag.append(item)
    return bag

print(add("a"))  # ['a']
print(add("b"))  # ['a', 'b']  -- WAT
```

The `[]` in the function definition is evaluated **once**, when the function is defined, not every time you call it. So every call shares the *same* list. This is why you should never use a mutable default argument. Use `None` and create the real default inside:

```python
def add(item, bag=None):
    if bag is None:
        bag = []
    bag.append(item)
    return bag
```

Burn this into your brain. Mutable default = trap.

## Iterators and Generators

### What's an iterator

An **iterator** is anything you can ask "give me the next thing" until it runs out. The official rule: an iterator is an object with a `__next__()` method that returns the next value, and raises `StopIteration` when there are no more.

You almost never call `__next__` yourself. You use a `for` loop:

```python
for fruit in ["apple", "banana", "cherry"]:
    print(fruit)
```

Under the hood, the `for` loop:

1. Calls `iter(["apple", "banana", "cherry"])` to get an iterator.
2. Calls `next(iterator)` over and over.
3. Stops when it sees `StopIteration`.

Lists, tuples, strings, dicts, sets, files — all of these are **iterable**, meaning you can ask them for an iterator with `iter()`. Not all of them are iterators themselves. A list is iterable but not an iterator. You call `iter(my_list)` and get a new fresh iterator over the list.

### Generators: iterators you write with `yield`

Writing an iterator class by hand (with `__iter__` and `__next__`) is annoying. Python gives you a much nicer way: **generators**.

A generator is a function that uses the `yield` keyword. The instant Python sees `yield` anywhere inside the function, that function becomes a generator function. Calling it doesn't run the code. It returns a generator object — a paused iterator.

```python
def count_up_to(n):
    i = 0
    while i < n:
        yield i
        i += 1

for x in count_up_to(3):
    print(x)
# 0
# 1
# 2
```

What's happening:

- `count_up_to(3)` doesn't run the function body. It hands you a paused generator.
- The first `next()` call starts the function. It runs until `yield i`, hands `i` (which is `0`) to you, and pauses **right there** with all its local variables intact.
- The next `next()` resumes from the `yield`. The loop runs `i += 1`, comes back around, sees `i < n`, yields `1`. Pause.
- And so on. When the function falls off the end (or hits `return`), the generator raises `StopIteration` and the `for` loop ends.

Generators are great because they are **lazy**. You don't build a list of a million numbers in memory; you produce them one at a time. That makes them perfect for processing huge files or infinite streams. (Yes, infinite — `while True: yield random.random()` is fine because you only ever pull what you need.)

### `yield from`

If you have one generator and want to delegate to another, you can use `yield from`:

```python
def outer():
    yield from range(3)
    yield from range(10, 13)

list(outer())  # [0, 1, 2, 10, 11, 12]
```

It chains generators together cleanly.

## Comprehensions and Generator Expressions

Python has shortcut syntax for "build a sequence by transforming another sequence." It looks weird at first and reads great later.

### List comprehension

```python
squares = [x*x for x in range(10)]
# [0, 1, 4, 9, 16, 25, 36, 49, 64, 81]
```

That builds a list of squares. The shape is `[<expression> for <var> in <iterable>]`. You can add a filter:

```python
even_squares = [x*x for x in range(10) if x % 2 == 0]
# [0, 4, 16, 36, 64]
```

You can nest:

```python
pairs = [(x, y) for x in range(3) for y in range(3) if x != y]
```

The order of `for` clauses matches what you'd write as nested loops.

### Dict comprehension

Same shape, with `{key: value}` syntax:

```python
squares = {x: x*x for x in range(10)}
```

### Set comprehension

Same shape with `{x}`:

```python
unique_lengths = {len(word) for word in ["hi", "hello", "yo", "world"]}
# {2, 5}
```

### Generator expression

Same shape, but with `()` instead of `[]`:

```python
total = sum(x*x for x in range(10))
```

Generator expressions are **lazy**. They don't build a list; they yield items one at a time. Use them when you don't need the whole list — when you're going to feed it to `sum`, `max`, `any`, `all`, or any function that consumes an iterator.

```python
[x for x in range(1_000_000_000)]    # tries to build a billion-item list, blows up your RAM
(x for x in range(1_000_000_000))    # fine, lazy, uses constant memory
```

A common rule of thumb: if the result is going to be passed to a single function that consumes it lazily, use a generator expression. If you need to keep the whole result around to use multiple times, use a list comprehension.

## Decorators

A **decorator** is a function that takes another function and returns a (usually wrapped) version of it. The `@something` syntax is just sugar.

```python
def shout(func):
    def wrapper(*args, **kwargs):
        result = func(*args, **kwargs)
        return result.upper()
    return wrapper

@shout
def greet(name):
    return f"hello, {name}"

print(greet("steve"))  # HELLO, STEVE
```

That `@shout` line is the same as writing `greet = shout(greet)` after the function definition. It's a shortcut.

Decorators are used everywhere:

- `@staticmethod` and `@classmethod` — change how a method is bound (we'll cover these later).
- `@property` — turn a method into something that looks like an attribute.
- `@functools.lru_cache` — memoize the function.
- `@dataclass` — auto-generate `__init__`, `__repr__`, etc. for a class.
- Web frameworks use them for routes: `@app.route("/")`.
- Test frameworks use them for fixtures: `@pytest.fixture`.

### Decorators with arguments

If a decorator takes arguments, it's actually a **factory** that returns a decorator:

```python
def repeat(n):
    def decorator(func):
        def wrapper(*args, **kwargs):
            for _ in range(n):
                func(*args, **kwargs)
        return wrapper
    return decorator

@repeat(3)
def hi():
    print("hi")

hi()
# hi
# hi
# hi
```

Three layers of functions. It looks weird the first time. It's just functions returning functions returning functions. Re-read it twice if it's not clicking.

### `functools.wraps`

When you wrap a function, you usually want to preserve its name, docstring, and other metadata. Use `functools.wraps`:

```python
import functools

def shout(func):
    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        return func(*args, **kwargs).upper()
    return wrapper
```

Without `@functools.wraps(func)`, the wrapped function looks like it's named `wrapper` and has no docstring. With it, all of `func`'s metadata is copied onto `wrapper`. Always use `@functools.wraps` when you write decorators.

## Context Managers

A **context manager** is something you use with the `with` statement to set up and tear down a resource cleanly.

The classic example is files:

```python
with open("data.txt") as f:
    contents = f.read()
# file is automatically closed here, even if an exception was raised
```

That's the whole point. The `with` block guarantees that the cleanup runs no matter what. You don't have to remember to call `f.close()`. You don't have to write a `try/finally`.

A context manager is any object with `__enter__` and `__exit__` methods. `__enter__` runs at the top of the `with` block; whatever it returns is bound to the `as` name. `__exit__` runs when the block is done (success or failure).

You can write your own with the `contextlib` module, which is much easier than writing a class:

```python
from contextlib import contextmanager
import time

@contextmanager
def timer(label):
    start = time.perf_counter()
    yield
    elapsed = time.perf_counter() - start
    print(f"{label}: {elapsed:.3f}s")

with timer("loading"):
    do_something_slow()
```

Common context managers in the wild:

- `open(...)` — files
- `threading.Lock()` — locks
- `tempfile.NamedTemporaryFile()` — temp files
- `unittest.mock.patch(...)` — mocking
- `contextlib.suppress(SomeError)` — swallow specific errors

If you want to manage **multiple** resources, you can stack them:

```python
with open("a") as fa, open("b") as fb:
    ...
```

Or use `contextlib.ExitStack` for dynamic cases.

## Type Hints and the typing module

Python is **dynamically typed** — types are checked at runtime, not at compile time. But since 2014 (Python 3.5) you can add **type hints** to your code that humans, IDEs, and external type checkers can use. Python itself ignores them at runtime; they're just annotations.

```python
def greet(name: str) -> str:
    return f"hello, {name}"
```

`name: str` says "this argument is expected to be a string." `-> str` says "this function is expected to return a string." If you call `greet(42)`, Python won't complain at runtime — but a type checker like mypy will tell you the call is wrong.

### Common type hint shapes

```python
from typing import Optional, Union, Any, Callable

def f(x: int) -> int: ...
def g(items: list[str]) -> None: ...
def h(table: dict[str, int]) -> int: ...
def i(x: Optional[int] = None) -> None: ...     # int or None
def j(x: int | None = None) -> None: ...        # same, modern syntax (3.10+)
def k(x: Union[int, str]) -> None: ...          # int or str
def l(x: int | str) -> None: ...                # same, modern syntax (3.10+)
def m(callback: Callable[[int, int], int]) -> None: ...
def n(x: Any) -> None: ...                      # escape hatch
```

In Python 3.10+, you can use `int | str` instead of `Union[int, str]` and `int | None` instead of `Optional[int]`. The new syntax is shorter and is the preferred style going forward.

In Python 3.9+, you can use `list[str]` and `dict[str, int]` directly. Before 3.9 you had to use `List[str]` and `Dict[str, int]` from `typing`.

In Python 3.12+, type parameter syntax is built-in:

```python
def first[T](items: list[T]) -> T:
    return items[0]
```

That `[T]` declares a type variable inline. Before 3.12, you had to import `TypeVar`.

### Protocols (structural typing)

Sometimes you want "anything that quacks like a duck" — you don't care about the class, just the shape:

```python
from typing import Protocol

class HasName(Protocol):
    name: str

def hello(x: HasName) -> None:
    print(f"hi {x.name}")
```

Anything with a `.name` attribute satisfies `HasName`. No inheritance needed. This is called **structural typing**.

### Type checkers

Python doesn't enforce type hints at runtime. To catch type bugs, you run a separate tool over your code:

- **mypy** — the original, strict, pickier.
- **pyright** — Microsoft's, faster, used by Pylance / VS Code.
- **pyre** — Facebook's, less common.
- **pytype** — Google's, less common.

Most teams pick one (often pyright in VS Code, mypy in CI) and run it as part of their lint step.

## Exceptions

When something goes wrong, Python raises an **exception**. An exception is an object that "unwinds" the call stack until something catches it.

### Try / except / else / finally

```python
try:
    risky_thing()
except ValueError as e:
    print("got a value error:", e)
except KeyError:
    print("got a key error")
except (FileNotFoundError, PermissionError):
    print("disk problem")
else:
    print("ran with no exceptions")
finally:
    print("always runs, success or failure")
```

The order:

- `try` runs first.
- If it raises, the `except` clauses are checked in order. The first matching one runs.
- If nothing raised, the `else` block runs (this is for "this code only runs if the try succeeded").
- The `finally` block runs no matter what.

### Catching everything

```python
try:
    ...
except Exception as e:
    log.error(f"something broke: {e}")
```

Catching `Exception` is broad but acceptable in a top-level handler. **Don't** catch `BaseException` (that's even broader and includes things like `KeyboardInterrupt` and `SystemExit`, which you almost always want to let through).

A bare `except:` is the same as `except BaseException:`. Don't use it.

### Raising exceptions

```python
raise ValueError("temperature out of range")
```

You can also raise from a custom exception class:

```python
class TemperatureError(Exception):
    pass

raise TemperatureError("too hot")
```

### Chaining: raise X from Y

If you catch one exception and want to raise a more specific one, use `raise X from Y` to keep the chain visible:

```python
try:
    int(x)
except ValueError as e:
    raise ConfigError("bad temperature config") from e
```

When this happens, the traceback shows both exceptions: "while handling the above exception, another occurred." Without `from`, Python still shows both, with a slightly different message. Either way, you don't lose information.

If you want to **suppress** the original cause, use `raise X from None`.

### Common built-in exceptions

We list these in **Common Errors** below — read them carefully, you will see all of them.

## Async/Await

This is its own world inside Python. We'll go slow.

### The problem

Network code spends most of its life waiting. Your program asks the database for a row and waits 5 milliseconds for the answer. While it waits, the CPU is idle. If you're handling 1000 connections, with regular blocking I/O you'd need 1000 threads, which is heavy. With **async I/O** you can handle them all on a single thread, by switching to whichever connection has data ready right now.

### Coroutines

A **coroutine** is like a function but pausable. You define one with `async def`:

```python
async def fetch(url):
    response = await http.get(url)
    return response.text
```

Calling `fetch(...)` does not run the body. It returns a coroutine object — a thing you can later `await` or hand to the event loop.

The `await` keyword inside means "pause here until that other coroutine finishes; while paused, let other coroutines run."

### The event loop

Async Python relies on an **event loop**. The event loop is a giant dispatcher inside one thread. It holds a queue of coroutines that are ready to make progress. It pulls one off the queue, runs it until it hits an `await` that has to wait, parks it, picks up the next ready one, and so on.

```
                +----------------------+
                |      Event Loop      |
                |  (single thread)     |
                +----+----+-----+------+
                     |    |     |
              +------+    +     +-----+
              v          v            v
          coroutine A  coroutine B  coroutine C
          (waiting       (running     (waiting
           on socket)     a step)      on timer)
```

The standard library module is **`asyncio`**. The simple way to start it up is `asyncio.run`:

```python
import asyncio

async def main():
    await asyncio.sleep(1)
    print("hello after 1 second")

asyncio.run(main())
```

`asyncio.run(main())` does three things: creates a new event loop, runs `main()` until it finishes, then closes the loop. It is **the** entry point. Don't manually fiddle with event loops unless you have a very specific reason.

### Tasks

A **Task** is a coroutine that has been scheduled on the event loop and is running concurrently with whatever else is running. You create one with `asyncio.create_task` or, in modern code, with a **task group**:

```python
async def main():
    async with asyncio.TaskGroup() as tg:
        tg.create_task(fetch("https://a.com"))
        tg.create_task(fetch("https://b.com"))
        tg.create_task(fetch("https://c.com"))
```

`TaskGroup` (Python 3.11+) is the right way to spawn concurrent tasks. It waits for all of them and cancels any unfinished tasks if one of them raises. Before 3.11 people used `asyncio.gather`, which is still around but trickier with cancellation.

### Threads vs asyncio

- Threads run real code on multiple OS threads (still under the GIL for Python bytecode). Use them when you have blocking calls in libraries that aren't async-aware.
- `asyncio` runs on **one** thread but juggles many tasks via cooperative pausing at `await` points. Use it when you have lots of network connections and your library is async-aware (or you can wrap blocking calls with `asyncio.to_thread`).

### Don't mix them carelessly

Common mistake: calling a synchronous blocking function inside an async function. The whole event loop freezes for the duration of that call. If you must call a blocking function, use `await asyncio.to_thread(blocking_func, args)` — that runs the blocking function on a thread pool, awaits its result, and returns to your async code without freezing the loop.

### Other async libraries

- **trio** — alternative async library, focuses on safety (structured concurrency baked in).
- **anyio** — works on top of either asyncio or trio. Library authors target anyio so users can pick.
- **uvloop** — a faster event loop implementation, drop-in replacement for asyncio's default.

## Modules and Packages

### Modules

A **module** is just a Python file. If you have a file `tools.py`, it's a module called `tools`. Other code can `import tools` and then call `tools.something()`.

```python
# tools.py
def shout(s):
    return s.upper() + "!"

# main.py
import tools
print(tools.shout("hi"))
```

You can import in many ways:

```python
import tools                  # import the whole module
import tools as t             # rename it
from tools import shout       # bring just one name in
from tools import shout as sh # bring it in renamed
from tools import *           # bring everything (frowned upon)
```

### Packages

A **package** is a folder of modules, with a special `__init__.py` file inside that says "I'm a package." It looks like:

```
mypkg/
    __init__.py
    tools.py
    helpers.py
    sub/
        __init__.py
        more.py
```

You import with dots: `from mypkg.tools import shout`, `from mypkg.sub.more import something`.

The `__init__.py` is the package's "front page." It runs when you import the package, and anything defined or imported there is part of the package's namespace.

### Namespace packages

Since Python 3.3, you can have **namespace packages** — folders without an `__init__.py` that still act as packages. They can span multiple directories on disk. This is useful for splitting a big package across different installs, but it's an advanced feature; for most projects, just put a regular `__init__.py` in your packages and call it done.

### `__main__` and `if __name__ == "__main__":`

When you run a Python file directly, Python sets a special variable `__name__` to the string `"__main__"`. When you import it as a module, `__name__` is set to the module's name. So a common pattern is:

```python
def main():
    ...

if __name__ == "__main__":
    main()
```

This means "if I'm being run directly, call `main()`. If I'm being imported, don't." This way the file works as both a script and a library.

### `python -m`

`python -m something` runs the module `something` as a script. It's used a lot:

```
$ python -m venv .venv
$ python -m http.server
$ python -m json.tool < file.json
$ python -m timeit "x = sum(range(100))"
```

`python -m foo` is roughly equivalent to "find the module named `foo` and run its `__main__.py` (if it's a package) or its top-level code (if it's a module)."

## Virtual Environments

### Why

Different Python projects want different versions of different libraries. Project A wants Django 4.2. Project B wants Django 5.0. If you `pip install` into your system Python, you can only have one. So we use **virtual environments** (or "venvs"). A venv is a folder that contains its own Python interpreter (a copy or a symlink) and its own `site-packages` (where libraries get installed). When you "activate" a venv, your shell prefers that interpreter.

### `venv` (the built-in)

```
$ python -m venv .venv          # create
$ source .venv/bin/activate     # activate (mac/linux)
$ .venv\Scripts\activate        # activate (windows)
(.venv) $ pip install requests
(.venv) $ deactivate            # leave
```

Now `pip install` puts packages into `.venv/lib/python3.X/site-packages/`. They are isolated from your other projects.

### `virtualenv`

`virtualenv` is the older, third-party tool that `venv` was based on. Its main reason to exist now is supporting older Pythons (it works on Python 2 still) and a few extra features. For new code, just use `python -m venv`.

### `pyenv`

`pyenv` lets you install and switch between **multiple Python versions**. Different from `venv`, which isolates packages but uses your system's Python. With `pyenv`, you can have Python 3.11, 3.12, and 3.13 all installed side by side.

```
$ pyenv install 3.13.0
$ pyenv versions
$ pyenv shell 3.13.0
$ pyenv local 3.13.0     # set the version for this folder
$ pyenv global 3.13.0    # set the default
```

It uses **shims** — tiny wrappers in your PATH that figure out which Python to use based on your folder.

### `uv` (the new fast one)

`uv` is a much newer (2024+) tool from Astral (the people who made `ruff`). It's written in Rust and is very fast. It can:

- Manage virtualenvs (`uv venv`)
- Install packages (`uv pip install X`)
- Resolve dependencies (`uv pip compile`)
- Run scripts (`uv run script.py`)
- Install Python itself (`uv python install 3.13`)

It's basically trying to replace `pyenv + pip + virtualenv + pip-tools` with one tool. Many teams have already moved to it. For new projects, it's worth a look.

### `conda`

`conda` is a different beast, mainly used in scientific Python. Instead of a venv, it has its own environments. Instead of installing from PyPI, it has its own package repos (Anaconda, conda-forge). It is very good for things that ship binary deps (NumPy, SciPy, certain CUDA libraries) but heavier than venv. If you're doing machine learning or research, you'll see conda everywhere. For general-purpose Python, venv + pip (or uv) is fine.

### `poetry`, `hatch`, `pdm`

These are higher-level project management tools. They handle creating a venv, installing packages, locking versions, building wheels, publishing — all in one. They use `pyproject.toml`. Pick one if your project needs full lifecycle management. Or just use `uv`, which now does most of the same things.

## pip and PyPI

`pip` is the package installer. **PyPI** (the Python Package Index, pronounced "pie-pee-eye", at pypi.org) is the giant central public repository of Python packages. There are over 500,000 packages on PyPI.

### Basic usage

```
$ pip install requests
$ pip install requests==2.31.0
$ pip install 'requests>=2.30,<3'
$ pip install -r requirements.txt
$ pip install -e .                  # editable install of current project
$ pip uninstall requests
$ pip list
$ pip show requests
$ pip freeze > requirements.txt
```

`pip install -r requirements.txt` reads a list of pinned versions from a file and installs them all. It's the classic way to lock your project's dependencies.

### `requirements.txt`

A plain text file like:

```
requests==2.31.0
flask==3.0.0
psycopg[binary]==3.1.18
```

Every line is a package and version. `pip freeze` dumps your currently installed packages in this format.

### `pip-tools`

`pip-tools` is a small tool that gives you a cleaner workflow:

```
$ pip install pip-tools
$ pip-compile requirements.in    # turns a list of unpinned reqs into a fully-pinned requirements.txt
$ pip-sync requirements.txt      # makes your venv match the file exactly
```

The pinned `requirements.txt` includes every transitive dependency, so builds are reproducible.

### `pipx`

`pipx` is for installing **command-line tools** written in Python (like `black`, `ruff`, `httpie`, `youtube-dl`). It puts each tool in its own little venv automatically and exposes the binary on your PATH. So you don't have to think about it.

```
$ pipx install black
$ pipx install ruff
$ pipx list
```

Use `pipx` for tools you want everywhere. Use `pip install` inside a venv for libraries your project needs.

### `pip` vs `pip3`

On systems with both Python 2 and Python 3, `pip` was sometimes the Python-2 one and `pip3` was the Python-3 one. Today, Python 2 is dead. On most modern systems `pip` and `pip3` point at the same thing — your Python 3's pip. Inside a virtualenv, `pip` is the venv's pip. The safe form, if you're not sure, is `python -m pip install X` — that runs the pip that belongs to whatever `python` is.

### PEP 668: externally managed environments

On many newer Linux distros, if you try `pip install` outside a venv, you'll get an error like:

```
error: externally-managed-environment
```

That's the OS saying "don't put packages into the system Python; use a venv or use pipx." It's a good rule. Use a venv. Or if you're installing a CLI tool, use pipx.

## Project Layout

### `pyproject.toml`

The modern way to describe a Python project is a single file: `pyproject.toml`. It looks like:

```toml
[project]
name = "myproject"
version = "0.1.0"
dependencies = ["requests>=2.30", "flask>=3"]
requires-python = ">=3.11"

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

This file replaces the old `setup.py` for almost everything. The `[project]` table is standard (PEP 621). The `[build-system]` table tells pip what tool to use to build wheels.

### `setup.py` (legacy)

You'll still see `setup.py` in older projects:

```python
from setuptools import setup
setup(name="myproject", version="0.1.0", install_requires=["requests"])
```

It's not deprecated yet, but new projects should use `pyproject.toml`. Don't write a new `setup.py` if you can avoid it.

### `src/` layout vs flat layout

There are two common ways to lay out a project.

**Flat layout:**

```
myproject/
    pyproject.toml
    mypkg/
        __init__.py
        ...
    tests/
        ...
```

**Src layout:**

```
myproject/
    pyproject.toml
    src/
        mypkg/
            __init__.py
            ...
    tests/
        ...
```

The `src/` layout is slightly safer because it forces you to install your project (with `pip install -e .`) before running tests. That guarantees you're testing the *installed* version, not just whatever happens to be in your CWD. Most modern tutorials prefer `src/`. Either works.

## Testing

### `pytest`

`pytest` is the de-facto standard. It's not in the standard library, but it's so common that you should just use it.

```python
# test_math.py
def add(a, b):
    return a + b

def test_add():
    assert add(2, 3) == 5
```

Run with `pytest`. It finds files starting with `test_`, collects all functions starting with `test_`, runs them, and tells you what failed.

```
$ pytest -v                        # verbose
$ pytest -k 'add'                  # only tests with 'add' in the name
$ pytest --pdb                     # drop into debugger on failure
$ pytest --cov=mypkg               # measure coverage of mypkg
$ pytest tests/test_math.py::test_add   # run one specific test
```

### `unittest` (built-in)

Python ships with `unittest` in the standard library. It's class-based, more verbose, but works with no external deps:

```python
import unittest

class TestMath(unittest.TestCase):
    def test_add(self):
        self.assertEqual(2 + 3, 5)

if __name__ == "__main__":
    unittest.main()
```

Run with `python -m unittest discover`. Most projects use pytest because it's nicer, but `unittest` is fine if you can't add deps.

### `doctest`

You can put little examples in your docstrings, and `doctest` will check that they still work:

```python
def add(a, b):
    """
    >>> add(2, 3)
    5
    """
    return a + b
```

Run with `python -m doctest -v module.py`. Useful for documenting library behaviour with verifiable examples.

### Mocking

`unittest.mock` (in the standard library) handles patching — replacing a function or method during a test:

```python
from unittest.mock import patch

@patch("mymodule.requests.get")
def test_thing(mock_get):
    mock_get.return_value.json.return_value = {"x": 1}
    ...
```

Or as a context manager: `with patch("mymodule.thing") as m: ...`.

## Linting and Formatting

A formatter rewrites your code into a canonical style. A linter looks for bugs and style issues. A type checker looks for type errors. You usually want all three.

### `ruff`

`ruff` is a very fast Rust-written linter and formatter. It replaces or supersedes flake8, isort, pylint (partially), pyupgrade, and many other tools in one binary.

```
$ ruff check                # lint
$ ruff check --fix          # auto-fix what it can
$ ruff format               # reformat all your code
```

Configure it in `pyproject.toml` under `[tool.ruff]`. New projects almost universally use ruff.

### `black`

`black` is a strict, opinionated code formatter. "Any color you want, as long as it's black." It takes your code and rewrites it in a fixed style. No options. The lack of options is the feature.

```
$ black .                   # reformat the whole project
$ black --check .           # exit non-zero if anything would change (use in CI)
```

`ruff format` is mostly compatible with `black`. Many teams now use just ruff for both lint and format.

### `isort`

Sorts your imports. Mostly subsumed by ruff and black these days.

### `mypy` and `pyright`

Type checkers. Run them on your code; they read type hints and tell you about violations.

```
$ mypy mypkg
$ pyright
```

Pick one. Most VS Code users get `pyright` for free via the Pylance extension. Many CI pipelines use `mypy` because it's been around longer and is more configurable.

### `flake8`, `pylint`, `pyre`

Older tools. `flake8` was the previous default linter; `pylint` is much pickier and slower. `pyre` is Facebook's. You'll see them in legacy projects. New projects mostly use ruff.

## Common Errors

These are the exact texts of errors you will see again and again. Learn to recognize each one and what it means.

### `NameError: name 'x' is not defined`

You used a name that Python can't find anywhere in the LEGB chain. Either you spelled it wrong, you forgot to import it, or you used it before assigning it.

```python
print(x)        # NameError if x was never defined
```

### `AttributeError: 'NoneType' object has no attribute 'foo'`

You did `something.foo` on an object that doesn't have a `foo`. The `'NoneType'` flavor is the most common: a function returned `None` and you tried to call a method on the `None`.

```python
result = my_func()  # returned None unexpectedly
result.upper()      # AttributeError: 'NoneType' object has no attribute 'upper'
```

### `TypeError: 'int' object is not iterable`

You tried to iterate something that isn't iterable.

```python
for x in 5:        # TypeError
    print(x)
```

Or:

### `TypeError: unsupported operand type(s) for +: 'int' and 'str'`

You tried to add an int and a string. Python won't auto-convert. Wrap one in `str(...)` or `int(...)`.

### `TypeError: f() missing 1 required positional argument: 'x'`

You called `f()` but forgot to give it `x`.

### `IndexError: list index out of range`

You did `mylist[5]` but the list has fewer than 6 items.

### `KeyError: 'foo'`

You did `mydict['foo']` but `'foo'` is not a key. Use `mydict.get('foo')` to get `None` instead, or `mydict.get('foo', default_value)` for a default.

### `ValueError: invalid literal for int() with base 10: 'abc'`

You tried `int('abc')`. The string isn't a valid integer.

### `ImportError: cannot import name 'X' from 'mypkg'`

The module exists, but `X` isn't in it. Spelling? Was it removed in a newer version?

### `ModuleNotFoundError: No module named 'requests'`

The package isn't installed in the current environment. `pip install requests` (in the right venv).

### `RecursionError: maximum recursion depth exceeded`

A function called itself too many times. Default limit is 1000. Either fix the recursion (it's probably a bug) or, if you really need deeper recursion, you can `sys.setrecursionlimit(10000)`. Better: rewrite as a loop.

### `IndentationError: expected an indented block`

You had an `if`, `for`, `def`, etc., and forgot to indent the body. Or you mixed tabs and spaces.

### `SyntaxError: invalid syntax`

Python couldn't even parse your file. Usually a missing `:`, an unmatched paren, a stray comma. Look at the line number Python points at; the actual mistake is usually on or just before that line.

### `UnicodeDecodeError: 'utf-8' codec can't decode byte 0x... in position N: invalid start byte`

You tried to read a file as text but its bytes aren't valid UTF-8. Either it's actually a binary file (open with `'rb'`) or it's text in a different encoding (open with `encoding='latin-1'` or whatever it actually is).

### `FileNotFoundError: [Errno 2] No such file or directory: 'foo.txt'`

The path doesn't exist. Check the cwd. Check the spelling. Use absolute paths if you're not sure.

### `PermissionError: [Errno 13] Permission denied: '/etc/something'`

You don't have permission to read or write that file. Either run with the right user or pick a different path.

### `ConnectionError`

Network call failed. Could be a DNS lookup that didn't resolve, a refused connection, a timeout. Check the wifi, check the URL.

### `asyncio.CancelledError`

The current task was cancelled. Often expected (a parent task group is shutting things down). If you catch it, you should usually re-raise it, otherwise you mess up cancellation.

### `RuntimeError: dictionary changed size during iteration`

You modified a dict (or set, or list) while iterating over it.

```python
for k in mydict:
    if condition:
        del mydict[k]   # boom
```

Iterate over a snapshot: `for k in list(mydict): ...`.

### `RuntimeError: cannot reuse already awaited coroutine`

You called `await` on the same coroutine object twice. Coroutines are single-shot.

```python
c = some_coro()
await c
await c     # RuntimeError
```

If you want to run the same code twice, call `some_coro()` twice — that gives you two separate coroutine objects.

## Hands-On

Open a real terminal. Type these. (Don't worry if you don't know what some do yet — try them anyway, then read what they did.)

```
$ python --version
Python 3.13.0
```

Tells you which Python is on your PATH.

```
$ python -c 'print("hello")'
hello
```

`-c` runs a single line of Python.

```
$ python -m venv .venv
```

Creates a virtual environment in the folder `.venv`.

```
$ source .venv/bin/activate
(.venv) $
```

Activates the venv. Your prompt now shows `(.venv)`.

```
$ pip install -r requirements.txt
```

Installs all dependencies listed in `requirements.txt` into the active venv.

```
$ pip install pip-tools
$ pip-compile requirements.in
$ pip-sync requirements.txt
```

Pin dependencies and sync your venv to match.

```
$ pipx install black
$ pipx install ruff
$ pipx install httpie
```

Install command-line tools globally without polluting your venv.

```
$ pyenv install 3.13.0
$ pyenv versions
$ pyenv shell 3.13
```

Install and switch to a specific Python version (after installing pyenv).

```
$ uv venv
$ uv pip install requests
$ uv pip compile requirements.in
$ uv run python script.py
```

The fast Rust-based replacement for venv + pip + pip-tools.

```
$ ruff check
$ ruff check --fix
$ ruff format
```

Lint and auto-format your code.

```
$ black --check .
$ black .
```

Check for formatting issues, or apply formatting.

```
$ mypy .
$ pyright
```

Run type checkers across your whole project.

```
$ pytest -v
$ pytest --cov
$ pytest -k 'parser'
$ pytest --pdb
```

Run tests verbose, with coverage, by name pattern, or drop into the debugger on failure.

```
$ python -m timeit '"-".join(str(n) for n in range(100))'
```

Quickly benchmark a one-liner. `timeit` runs the code many times and reports the average.

```
$ python -m cProfile -s cumulative script.py
```

Profile a script. `-s cumulative` sorts the output by total time per function.

```
$ python -X importtime script.py 2>&1 | head
```

See where startup time is going. Each line shows how long an import took.

```
$ python -X dev script.py
```

Run in **developer mode** — extra warnings, deprecation warnings shown by default, more checks. Use it during development.

```
$ python -c 'import dis; dis.dis(lambda x: x + 1)'
```

Disassemble a function and see its bytecode. Eye-opening.

```
$ python -c 'import inspect, json; print(inspect.getsource(json.dumps))'
```

Get the source code of any function or class in the standard library. Useful for poking around.

```
$ python -c 'help(dict)'
```

Open the in-process help for any object. Quits when you press `q`.

```
$ python -m http.server
```

Spin up a tiny web server in the current folder on port 8000. Great for quickly serving static files.

```
$ python -m json.tool < file.json
```

Pretty-print JSON. Reads from stdin, writes pretty JSON to stdout.

```
$ python -m unittest discover
```

Find and run every `test_*.py` file. The built-in test runner.

```
$ python -m pdb script.py
```

Run a script under the Python debugger. Type `n` to step, `c` to continue, `p x` to print `x`, `q` to quit.

```
$ python -c 'import sys; print(sys.path)'
```

See where Python looks for modules. The first entry is usually `''` or the script's directory.

```
$ python -c 'import sys; print(list(sys.modules)[:20])'
```

See some of the modules currently loaded. `sys.modules` is the live cache of imports.

```
$ python -c 'import gc; gc.collect(); print(gc.get_count())'
```

Force a cycle collector run and see how many objects are in each generation.

```
$ python -c 'import sysconfig; print(sysconfig.get_paths())'
```

See the paths Python uses for installs (where `pip install` puts things).

```
$ python -i script.py
```

Run a script and then drop into the interactive shell with all of its names available. Great for poking at state after a script runs.

```
$ python -W error script.py
```

Turn all warnings into errors. Useful for finding deprecation warnings that you've been ignoring.

## Common Confusions

Pairs and trios of things that confuse people. If something here surprises you, you probably had it wrong before.

### List comprehension vs generator vs `map`

```python
[x*2 for x in items]              # list comprehension — builds a list eagerly
(x*2 for x in items)              # generator expression — lazy, no list built
map(lambda x: x*2, items)         # map object — lazy too, returns a map iterator
```

For small inputs, prefer the list comprehension (most readable). For huge or infinite inputs, use the generator expression. `map` is a bit weird in Python 3 because it returns an iterator, not a list — you have to wrap it in `list(...)` if you want a list.

### Mutable default argument trap

```python
def add(item, bag=[]):       # BAD — list is shared across calls
    bag.append(item)
    return bag

def add(item, bag=None):     # GOOD
    if bag is None:
        bag = []
    bag.append(item)
    return bag
```

Default values are evaluated once at function-definition time. If the default is a mutable object, all calls share it.

### `is` vs `==`

`is` asks "same object?" (same `id()`). `==` asks "same value?". For most checks, you want `==`. The exception is `is None`, `is True`, `is False` — these singletons should always be compared with `is`.

### `copy` vs `deepcopy`

```python
import copy
a = [[1, 2], [3, 4]]
b = a.copy()              # shallow: top-level new, inner lists shared
c = copy.deepcopy(a)      # deep: everything new, nothing shared
b[0].append(999)
print(a)   # [[1, 2, 999], [3, 4]] — shared inner list!
print(c)   # untouched
```

`.copy()` only copies the outer container. Inner objects are still shared.

### Class vs instance attribute

```python
class Foo:
    items = []         # CLASS attribute — shared across all instances
    def __init__(self):
        pass

f1 = Foo()
f2 = Foo()
f1.items.append(1)
print(f2.items)   # [1] — they share the same list!
```

Class-level mutable defaults are basically the same trap as the default argument trap. If you want per-instance state, set it in `__init__`.

### Bound vs unbound method

In Python 3, `Foo.method` is just a function. `f = Foo(); f.method` is a **bound method** — it knows that `f` should be passed as `self` automatically.

```python
class Foo:
    def m(self): pass

Foo.m         # function
Foo().m       # bound method
```

You almost never think about this anymore unless you're doing metaclass tricks.

### `staticmethod` vs `classmethod`

```python
class Foo:
    @staticmethod
    def bar():
        ...                # no self, no cls. Just a regular function inside a class.

    @classmethod
    def baz(cls):
        ...                # cls is the class itself; great for alternate constructors.
```

Use `@staticmethod` if your method doesn't use `self` or `cls`. Use `@classmethod` for "alternate constructors" (`MyClass.from_string(s)`).

### `__init__` vs `__new__`

`__new__` creates the instance. `__init__` configures it. You almost never override `__new__`. The only times you do are: subclassing immutable built-ins (`int`, `tuple`, `str`), or implementing singletons / metaclass shenanigans.

### What `asyncio.run` actually does

`asyncio.run(coro())` does three things:

1. Creates a new event loop.
2. Runs `coro()` until it returns.
3. Closes the loop.

It is **not** "make this code run async." It's the entry point. Don't call `asyncio.run` from inside an already-running event loop — you'll get a `RuntimeError`.

### Threading vs multiprocessing under the GIL

| | threading | multiprocessing |
|---|---|---|
| Real parallel CPU? | No (GIL blocks) | Yes |
| Memory shared? | Yes (same process) | No (separate processes) |
| Startup cost | Cheap | Expensive |
| Best for | I/O-bound work | CPU-bound work |
| Communication | Shared memory, locks | Queues, pipes, shared memory blocks |

Pick threading for "wait on lots of network calls." Pick multiprocessing for "do real math on lots of cores."

### `pip` vs `pip3`

On modern systems they're the same. Inside a venv, `pip` is the venv's pip. If you're not sure which Python a `pip` belongs to, use `python -m pip ...`.

### `venv` vs `virtualenv` vs `pipenv` vs `poetry` vs `uv`

| Tool | What it does |
|------|--------------|
| `venv` | Built-in, creates virtual environments. That's it. |
| `virtualenv` | Older third-party tool. Same idea as `venv`. Modern: use `venv`. |
| `pipenv` | venv + lockfile + dep tool, all-in-one. Was popular ~2018. Less used today. |
| `poetry` | venv + lockfile + build/publish + project metadata. Big and feature-rich. |
| `uv` | New (2024). Rust. Fast. Aiming to replace pyenv + pip + virtualenv + pip-tools. |

If you're starting a new project today: `uv` if you want speed and a modern toolchain; `poetry` if you want a mature, opinionated tool; plain `venv + pip + pyproject.toml` if you want minimal moving parts.

### `setup.py` legacy vs `pyproject.toml`

`setup.py` was the old way: a Python script that described your project. `pyproject.toml` is the new way: a static TOML file. Modern tools read `pyproject.toml`. Don't write a new `setup.py` if you can help it.

### `importlib` vs `__import__`

`__import__` is the underlying machinery; rarely used directly. `importlib` is the modern, friendlier API. If you need to import dynamically, use `importlib.import_module("foo.bar")`.

### `PYTHONPATH` vs `sys.path`

`PYTHONPATH` is an environment variable. Python reads it at startup and prepends its entries to `sys.path`. `sys.path` is the actual list Python checks when importing. You can modify `sys.path` at runtime (people do, sparingly), but it's much cleaner to install your project properly with `pip install -e .` than to hack `PYTHONPATH`.

## Vocabulary

| Word | Meaning |
|------|---------|
| interpreter | The program that reads your Python source and runs it. CPython is the official one. |
| bytecode | The tiny tape of instructions Python compiles your source into before running. |
| `.pyc` | A file containing cached bytecode for a module. Lives in `__pycache__`. |
| `__pycache__` | Folder where Python stores `.pyc` files. Auto-managed; ignore it. |
| CPython | The reference Python implementation, written in C. The "Python" everyone means. |
| PyPy | Alternative Python interpreter with a JIT. Often faster than CPython. |
| MicroPython | Tiny Python for microcontrollers (kilobytes of RAM). |
| Jython | Python on the JVM. Old. |
| IronPython | Python on .NET. Old. |
| GraalPy | Python on GraalVM. Newer experiment. |
| GIL | Global Interpreter Lock. Only one thread runs Python bytecode at a time. |
| reference count | Number of names currently pointing at an object. When it hits 0, the object dies. |
| cycle collector | Periodically frees objects that point at each other but nowhere else. |
| `gc.collect()` | Force a full cycle collection right now. |
| object | Any thing in Python. Has a type, a value, and an identity. |
| type | What kind of thing this object is. `type(x)` tells you. |
| class | A blueprint for making objects. Defined with `class`. |
| instance | An object made from a class. |
| name | A label that points at an object. Sometimes called a variable. |
| binding | The act of attaching a name to an object. `x = 7` is a binding. |
| scope | Where a name is visible. Functions create new scopes. |
| LEGB | Local, Enclosing, Global, Built-in — the order Python searches for names. |
| module | A single `.py` file you can import. |
| package | A folder with `__init__.py`. Holds modules and subpackages. |
| namespace package | Package without `__init__.py`. Can span directories. Advanced. |
| regular package | Package with an `__init__.py`. The normal kind. |
| `__init__.py` | The file that makes a folder a regular package. |
| `__main__` | The module name Python sets when you run a file directly. |
| `__name__` | Variable holding the current module's name. `'__main__'` if run directly. |
| `__file__` | Path to the current source file. |
| `__doc__` | Docstring of the current module/function/class. |
| `__dict__` | Internal dict that holds an object's attributes. |
| `__slots__` | Class attribute that restricts which attribute names instances can have, and saves memory. |
| MRO | Method Resolution Order. Order Python searches base classes for a method. `Foo.__mro__`. |
| super | Refers to the parent class. `super().method()` calls parent's `method`. |
| dunder | "Double underscore." Methods like `__init__` are "dunder methods." |
| magic method | Same thing as a dunder. Methods with special meaning to Python. |
| `__getitem__` | Called for `obj[key]`. |
| `__setitem__` | Called for `obj[key] = value`. |
| `__iter__` | Called for `iter(obj)`. Returns an iterator. |
| `__next__` | Called for `next(obj)`. Returns the next value or raises `StopIteration`. |
| `__enter__` | Called when entering a `with` block. Returns the value bound by `as`. |
| `__exit__` | Called when leaving a `with` block. |
| `__call__` | Lets an instance be called like a function. |
| `__repr__` | Developer-friendly string representation. Used in REPL and debug logs. |
| `__str__` | User-friendly string representation. Used by `print()`. |
| descriptor | Object implementing `__get__`/`__set__`. The mechanism behind `property`. |
| property | A method exposed as an attribute. `@property` decorator. |
| `staticmethod` | Method that takes no `self` or `cls`. |
| `classmethod` | Method that takes `cls` instead of `self`. Good for alternate constructors. |
| mutable | Can be changed in place. List, dict, set. |
| immutable | Cannot be changed in place. Int, float, str, tuple, frozenset, bytes. |
| hashable | Can be used as a dict key or set element. Most immutables are hashable. |
| list | Ordered, mutable, indexed by int. The default sequence. |
| tuple | Ordered, immutable. Often used for fixed-size groupings. |
| dict | Hash map. Maps hashable keys to values. Insertion-ordered since 3.7. |
| set | Unordered collection of unique hashable items. |
| frozenset | Immutable set. |
| str | Sequence of Unicode code points. Immutable. |
| bytes | Immutable sequence of bytes. |
| bytearray | Mutable sequence of bytes. |
| memoryview | A view into another bytes-like object's memory, no copy. |
| range | Lazy sequence of integers. `range(10)`. |
| slice | Object representing `start:stop:step`. `mylist[1:5:2]`. |
| generator | Function that uses `yield`. Returns a paused iterator. |
| coroutine | `async def` function. Returns a thing you can `await`. |
| async generator | `async def` function with `yield`. Iterate with `async for`. |
| awaitable | Something you can `await`. Coroutines, futures, tasks. |
| Future | Low-level placeholder for a value that will arrive later. |
| Task | A coroutine scheduled on the event loop. Subclass of Future. |
| event loop | The dispatcher inside `asyncio` that runs coroutines and tasks. |
| asyncio | The standard library async runtime. |
| anyio | Library that runs on top of asyncio or trio. |
| trio | Alternative async runtime focused on safety. |
| threading | Standard library module for OS threads. |
| Thread | A thread of execution. `threading.Thread`. |
| Lock | Mutual exclusion primitive. `threading.Lock`. |
| Event | Threading signal you can set and wait on. |
| Semaphore | Bounded counting lock. |
| multiprocessing | Standard library module for spawning processes. |
| Process | An OS process. `multiprocessing.Process`. |
| Pool | A pool of worker processes. `multiprocessing.Pool`. |
| concurrent.futures | High-level wrapper for thread/process pools. |
| ProcessPoolExecutor | Pool that runs work in subprocesses. |
| ThreadPoolExecutor | Pool that runs work in threads. |
| decorator | Function that wraps another function. `@something`. |
| closure | Inner function that captures names from an outer scope. |
| late binding | Closures look up captured names when they run, not when they're defined. |
| comprehension | Inline loop syntax to build a list, set, or dict. |
| walrus operator | `:=`. Assigns and returns in one expression. New in 3.8. |
| f-string | `f"hi {name}"`. Inline string formatting. |
| raw string | `r"\n"`. Treats backslashes literally. Used in regexes. |
| bytes literal | `b"abc"`. Bytes object. |
| type hint | Annotation that says what type a value should be. Not enforced at runtime. |
| `typing.Any` | "Any type." Disables type checking on this value. |
| `typing.Optional` | `Optional[X]` is `X | None`. Older spelling. |
| `typing.Union` | `Union[A, B]` is `A | B`. Older spelling. |
| `typing.Literal` | `Literal[1, 2, 3]` — exactly one of those values. |
| `typing.Protocol` | Structural type. "Anything with these attrs." |
| `typing.TypeVar` | A type variable. `T = TypeVar("T")`. |
| `typing.Generic` | Base class for generic classes. |
| dataclass | `@dataclass` decorator that auto-generates `__init__`, `__repr__`, etc. |
| attrs | Third-party library, predecessor to `dataclasses`, more featureful. |
| pydantic | Library for data validation using type hints. |
| msgspec | Faster, stricter alternative to pydantic for serialization. |
| pickle | Standard library module for serializing Python objects. Don't load untrusted pickles. |
| json | Standard library module for JSON. |
| csv | Standard library module for CSV. |
| sqlite3 | Standard library SQLite client. |
| requests | Third-party HTTP client. The classic. |
| httpx | Modern HTTP client. Sync and async. |
| aiohttp | Async HTTP client and server. |
| fastapi | Async web framework built on Starlette + pydantic. |
| flask | Classic small sync web framework. |
| django | Big batteries-included sync web framework. |
| starlette | Async web toolkit underneath FastAPI. |
| uvicorn | ASGI server (runs FastAPI/Starlette apps). |
| gunicorn | WSGI server (runs Flask/Django apps). |
| hypercorn | ASGI server alternative to uvicorn. |
| asyncio.Queue | An async-aware queue. |
| anyio.Event | Cross-runtime event primitive. |
| asyncpg | Fast async PostgreSQL driver. |
| sqlalchemy | The big SQL toolkit + ORM for Python. |
| alembic | Database migrations for SQLAlchemy. |
| virtualenv | Original third-party venv tool. |
| venv | Built-in venv module. `python -m venv .venv`. |
| pyenv | Manages multiple Python versions. |
| conda | Different kind of environment manager popular in scientific computing. |
| uv | New, fast Rust-based replacement for pip + venv + pip-tools. |
| pip | Python's package installer. |
| pipx | Installs Python CLI tools in their own venvs. |
| poetry | Project management tool — venv + lockfile + build. |
| hatch | Project management tool. Lighter than poetry. |
| pdm | Project management tool, PEP 582 / 621 focused. |
| build | Standard tool to build wheels and sdists from a project. |
| twine | Tool to upload packages to PyPI. |
| setuptools | The classic Python build backend. |
| wheel | Built distribution format. `.whl` file. Pre-compiled. |
| sdist | Source distribution. `.tar.gz`. Built from source on install. |
| bdist_wheel | Old name for the wheel building command. |
| manylinux | Standard tag for Linux wheels that work across distros. |
| musllinux | Like manylinux but for musl-based Linux (Alpine). |
| ABI3 | Stable Python C ABI. Lets a compiled wheel work across many Python versions. |
| platform tag | Part of a wheel filename that says what OS/architecture it targets. |
| PEP 8 | Python style guide. |
| PEP 20 | Zen of Python (`import this`). |
| PEP 257 | Docstring conventions. |
| PEP 484 | Original type hints proposal. |
| PEP 517 | Build system spec. |
| PEP 518 | `pyproject.toml` spec. |
| PEP 621 | Project metadata in pyproject.toml. |
| PEP 660 | Editable installs from pyproject. |
| PEP 668 | Externally-managed environments (don't pip install into system Python). |
| ruff | Fast Rust-based linter and formatter. |
| black | Opinionated Python formatter. |
| isort | Sorts imports. Mostly subsumed by ruff. |
| flake8 | Older linter. |
| pylint | Stricter, slower linter. |
| mypy | Original static type checker. |
| pyright | Microsoft's fast type checker. |
| pyre | Facebook's type checker. |

## Try This

Before moving on, try each one. Don't just read.

- Make a `.venv`, activate it, install `requests`, fire up `python`, do `import requests`, hit `requests.get("https://example.com").status_code`.
- Run `python -c 'import dis; def f(x): return x+1; dis.dis(f)'` (you'll need to put it in a file or use a multi-line `-c`). Look at the bytecode. Notice `LOAD_FAST`, `BINARY_OP`, `RETURN_VALUE`.
- Trigger every error in the **Common Errors** section on purpose. Use the REPL. Read the message. Recognize each one.
- Write a tiny generator function with `yield`. Call `next()` on it three times by hand. Then put it in a `for` loop. Convince yourself the loop calls `next()` for you.
- Write a decorator that prints a function's name and arguments before calling it.
- Write a context manager with `@contextmanager` that prints "before" and "after." Use it in a `with` block.
- Write a tiny `asyncio` script with `asyncio.run` and `asyncio.sleep`. Add a `TaskGroup` with two concurrent sleeps and notice the total time is the longer of the two, not the sum.
- Add type hints to a small function. Run `mypy` on it. Break the types on purpose. See mypy yell at you.
- Run `pip-compile` on a `requirements.in` with one line, and look at the resulting fully-pinned `requirements.txt`.
- Write a `pyproject.toml` for a tiny package, run `pip install -e .`, then `import` it from elsewhere.

## Where to Go Next

- [languages/python](../languages/python.md) — the comprehensive Python sheet. Goes deep on the language and the standard library.
- [detail/languages/python](../../detail/languages/python.md) — the deep-dive theory page.
- [package-managers/pip](../package-managers/pip.md) — pip in depth.
- [ramp-up/bash-eli5](bash-eli5.md) — if you skipped it, the terminal sheet will fill in gaps.
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md) — what your Python interpreter is sitting on top of.
- [languages/go](../languages/go.md), [languages/rust](../languages/rust.md), [languages/typescript](../languages/typescript.md) — once Python feels boring, take a look at how other languages solve the same problems differently.

## Version Notes

- **Python 3.10** — pattern matching with `match` / `case`. Parenthesized context managers. Better error messages.
- **Python 3.11** — `asyncio.TaskGroup`. `tomllib` in the standard library (read TOML without a third-party lib). Much better tracebacks (with the exact column where the error happened). About 25% faster than 3.10 for many workloads.
- **Python 3.12** — Per-interpreter GIL groundwork (towards subinterpreters with their own GIL). New `type` statement. Type parameter syntax `def f[T](...)`. `typing.override` decorator.
- **Python 3.13** — Free-threaded build (no-GIL) is available as an experimental opt-in. JIT preview is available as a build option (mostly invisible to users right now). REPL is much nicer (multiline editing, color, history search). Removed several long-deprecated stdlib modules.

If you're starting a new project today, target Python 3.11 or later. Older versions are out of security support or close to it.

## See Also

- [languages/python](../languages/python.md)
- [detail/languages/python](../../detail/languages/python.md)
- [languages/typescript](../languages/typescript.md)
- [languages/go](../languages/go.md)
- [languages/rust](../languages/rust.md)
- [package-managers/pip](../package-managers/pip.md)
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md)
- [ramp-up/bash-eli5](bash-eli5.md)

## References

- [docs.python.org](https://docs.python.org/3/) — the official Python documentation. The single best Python reference. The tutorial, library reference, and language reference together cover almost everything.
- [peps.python.org](https://peps.python.org/) — Python Enhancement Proposals. Every change to the language is debated and decided in a PEP. Reading PEP 8 (style), PEP 20 (zen), PEP 257 (docstrings), PEP 484 (type hints), and PEP 621 (pyproject) will make you literate in modern Python culture.
- *Fluent Python* by Luciano Ramalho — the deepest single book on Python. Covers data model, descriptors, metaclasses, async, and what makes Python *Pythonic*. Read it after you're comfortable with the basics.
- *Effective Python* by Brett Slatkin — 90 practical items, each a self-contained tip. Read one a day.
- [realpython.com](https://realpython.com/) — long-form tutorials at every level.
- [bugs.python.org](https://bugs.python.org/) and [github.com/python/cpython](https://github.com/python/cpython) — the source code and the issue tracker. When you want to know how something is *really* implemented, the answer is here.
