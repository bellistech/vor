# The Internals of Ruby — Object Model, Method Lookup, and YARV

> *Ruby's object model is built on three principles: everything is an object (including classes and modules), method lookup follows a precise ancestor chain, and blocks/procs/lambdas are closures with different control flow semantics. The YARV virtual machine compiles source to bytecode and executes it on a stack-based VM.*

---

## 1. Object Model — Everything Is an Object

### The Object Graph

```
                      BasicObject
                          │
                        Object
                       /      \
                    Module    Kernel (mixed in)
                      │
                    Class
                   /     \
          String  Integer  Array  Hash  ...  YourClass
```

Every object has:
- A **class pointer** (which class it's an instance of)
- An **instance variable table** (hash map of `@vars`)
- An optional **singleton class** (eigenclass) for per-object methods

### Classes Are Objects

```ruby
String.class          #=> Class
Class.class           #=> Class  (circular!)
Class.superclass      #=> Module
Module.superclass     #=> Object
Object.superclass     #=> BasicObject
BasicObject.superclass #=> nil
```

The circular dependency: `Class` is an instance of itself.

### Singleton Classes (Eigenclasses)

Every object can have an invisible **singleton class** inserted between itself and its actual class:

```
obj → (singleton class of obj) → ActualClass → superclass → ...
```

This is how `def obj.method` works — the method is defined on the singleton class.

For classes, the singleton class hierarchy mirrors the regular hierarchy:

```
MyClass → (singleton MyClass) → (singleton SuperClass) → (singleton Object) → (singleton BasicObject) → Class
```

---

## 2. Method Lookup Algorithm

### The Ancestor Chain

When you call `obj.method`, Ruby searches in this exact order:

$$\text{lookup}(obj, m) = \text{first } c \in \text{ancestors}(obj.\text{class}) \text{ where } m \in c.\text{methods}$$

```ruby
obj.class.ancestors
# => [MyClass, IncludedModule, Object, Kernel, BasicObject]
```

### Module Inclusion Order

| Operation | Insertion Point |
|:----------|:---------------|
| `include M` | After the including class |
| `prepend M` | Before the including class |
| `extend M` | Into the singleton class |

### Worked Example

```ruby
module A; end
module B; end
class C
  prepend A
  include B
end

C.ancestors  #=> [A, C, B, Object, Kernel, BasicObject]
```

Lookup order: A (prepended) → C → B (included) → Object → Kernel → BasicObject.

### `method_missing` — The Safety Net

If no method is found in the entire ancestor chain:

```
lookup fails → call method_missing(name, *args) on the object
             → if method_missing not defined: NoMethodError
```

This is how `ActiveRecord` dynamic finders (`find_by_name`) work — they intercept `method_missing` and generate methods.

---

## 3. Blocks, Procs, and Lambdas

### Three Closure Types

| Feature | Block | Proc | Lambda |
|:--------|:------|:-----|:-------|
| Object? | No (syntax only) | Yes (`Proc.new`) | Yes (`lambda` / `->`) |
| `return` | Returns from **enclosing method** | Returns from **enclosing method** | Returns from **lambda only** |
| Arity check | No (extra args ignored, missing = nil) | No | Yes (ArgumentError) |
| Created by | `{ }` / `do...end` | `Proc.new`, `proc {}` | `lambda {}`, `->() {}` |

### The `return` Difference

```ruby
def test_proc
  p = Proc.new { return 42 }   # return exits test_proc!
  p.call
  puts "never reached"
end
test_proc  #=> 42

def test_lambda
  l = lambda { return 42 }     # return exits lambda only
  l.call
  puts "reached!"               # this prints
end
```

### Block-to-Proc Conversion

The `&` operator converts between blocks and Procs:

```ruby
def takes_block(&blk)    # block → Proc (explicit capture)
  blk.call
end

my_proc = Proc.new { puts "hi" }
[1,2,3].each(&my_proc)   # Proc → block (passed to each)
```

### Yield Optimization

`yield` is faster than `block.call` because it avoids creating a Proc object:

```ruby
# Fast — no Proc allocation:
def fast; yield; end

# Slower — allocates Proc:
def slow(&block); block.call; end
```

---

## 4. YARV — Yet Another Ruby VM

### Compilation Pipeline

```
Source (.rb)
    │ parse.y (Bison grammar)
    ▼
AST (Abstract Syntax Tree)
    │ compile.c
    ▼
YARV bytecode (instruction sequences)
    │ vm_exec.c
    ▼
Execution (stack-based VM)
```

### Key Instructions

```ruby
RubyVM::InstructionSequence.disasm(method(:example))
```

```
== disasm: <RubyVM::InstructionSequence:example>
0000 putself                        # push self
0001 putobject  1                   # push 1
0003 putobject  2                   # push 2
0005 opt_plus                       # pop 2 operands, push result
0006 opt_send_without_block :puts   # call puts
0008 leave                          # return
```

### Instruction Categories

| Category | Examples | Purpose |
|:---------|:---------|:--------|
| Stack | `putself`, `putobject`, `dup`, `pop` | Manage operand stack |
| Variable | `getlocal`, `setlocal`, `getinstancevariable` | Variable access |
| Method call | `opt_send_without_block`, `send` | Method dispatch |
| Optimized | `opt_plus`, `opt_lt`, `opt_aref` | Fast-path for common ops |
| Control | `jump`, `branchif`, `branchunless` | Flow control |

### Inline Method Cache

YARV caches method lookup results at each call site:

```
call site → (class, method) cache
  if obj.class == cached_class: call cached_method  (fast path)
  else: full method lookup, update cache             (slow path)
```

---

## 5. Garbage Collector — Generational Mark-and-Sweep

### Evolution

| Ruby Version | GC Algorithm |
|:-------------|:-------------|
| 1.8 | Conservative mark-and-sweep (stop-the-world) |
| 2.1 | Generational (minor + major GC) |
| 2.2 | Incremental marking (reduced pause times) |
| 3.x | Compaction (reduce fragmentation) |

### Generational Hypothesis

Most objects die young. Ruby tracks **old** vs **new** objects:

| GC Type | Scans | Frequency | Pause |
|:--------|:------|:----------|:------|
| Minor GC | New objects only | Frequent | Short |
| Major GC | All objects | Infrequent | Longer |

### Write Barrier

When an old object references a new object, a **write barrier** records this. Without it, minor GC could miss new objects referenced only by old objects.

$$\text{If old} \to \text{new: add old to remembered set}$$

### Object Size

Ruby objects (RObject) are typically **40 bytes** on 64-bit systems. The GC allocates in **pages** of 400+ objects. Objects larger than 40 bytes use external heap allocation.

---

## 6. Symbol Table and String Interning

### Symbols vs Strings

| Feature | Symbol | String |
|:--------|:-------|:-------|
| Mutable | No | Yes |
| Interned | Always (one copy) | Only with `freeze` |
| GC collected | Yes (since Ruby 2.2) | Yes |
| Use case | Identifiers, hash keys | Text data |

### Frozen String Literals

```ruby
# frozen_string_literal: true
```

All string literals become frozen (immutable, interned). Reduces object allocation and GC pressure significantly.

---

## 7. Thread and Fiber Model

### GVL (Global VM Lock)

Like Python's GIL, Ruby has a **Global VM Lock** (GVL, formerly GIL):

$$\text{Ruby threads executing bytecode} \leq 1$$

Released during I/O, C extensions that explicitly release it, and `sleep`.

### Ractors (Ruby 3.0+)

Ractors provide true parallelism by isolating object spaces:

| Feature | Thread | Fiber | Ractor |
|:--------|:-------|:------|:-------|
| Parallelism | No (GVL) | No | Yes |
| Scheduling | Preemptive | Cooperative | Preemptive |
| Shared state | Yes (dangerous) | Yes | No (message passing) |
| Overhead | ~8KB stack | ~4KB | Separate object space |

### Fiber Scheduler (3.1+)

Non-blocking fibers with a pluggable scheduler — enables async I/O without callbacks:

```ruby
Fiber.schedule do
  io = IO.popen("sleep 1")
  io.read  # fiber yields, scheduler runs other fibers
end
```

---

## 8. Summary of Key Internals

| Concept | Mechanism | Key Detail |
|:--------|:----------|:-----------|
| Method lookup | Ancestor chain traversal | prepend → class → include → superclass |
| Singleton class | Hidden class per object | Stores `def obj.method` |
| VM architecture | Stack-based (YARV) | Inline method cache at call sites |
| GC | Generational mark-sweep + compaction | Minor (new) + Major (all) |
| Object size | 40 bytes (RObject, 64-bit) | Pages of 400+ objects |
| Closures | Block (syntax) vs Proc vs Lambda | `return` semantics differ |
| Concurrency | GVL limits to 1 thread | Ractors for true parallelism |

---

*Ruby's philosophy is programmer happiness, but its implementation is serious engineering: a generational GC, inline method caches, instruction specialization, and a precisely defined object model. The "magic" of Ruby (method_missing, open classes, eval) is built on deterministic, well-specified machinery.*
