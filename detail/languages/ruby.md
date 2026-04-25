# The Internals of CRuby — YARV, GVL, GC, and the Object Model

> *CRuby (also known as MRI — Matz's Ruby Interpreter) is the reference implementation Yukihiro "Matz" Matsumoto wrote in C. Every other Ruby (JRuby, TruffleRuby, mruby) is measured against its semantics. Beneath the surface of "everything is an object" lies a hand-written Bison parser, a stack-based bytecode VM called YARV, a tagged-pointer object model, a generational incremental GC, a global VM lock that survived three decades of debate, a true-parallelism Ractor system, a Rust-based YJIT, and an inline cache architecture rebuilt around basic-block versioning. This page is the bridge from "I write Rails" to "I can read `vm_insnhelper.c`."*

---

## 1. CRuby (MRI) Architecture

### The Pipeline

```
Source (.rb)
   │  parse.y (Bison/Yacc grammar — the longest .y file in the world)
   ▼
AST (struct RNode tree, defined in node.h)
   │  compile.c — iseq_compile_each / iseq_setup_insn
   ▼
ISeq (rb_iseq_t — instruction sequence, the YARV bytecode object)
   │  vm_exec.c — vm_exec_core / TC_DISPATCH dispatch loop
   ▼
Execution: control frames push/pop on rb_thread_t.ec.cfp
   │  GVL acquired, operand stack manipulated
   ▼
Result (a VALUE — tagged pointer)
```

The pipeline is single-process and single-image: CRuby loads `.rb` files, parses each into an AST, compiles each AST into an ISeq, caches the ISeq in memory, and dispatches through `vm_exec_core`. There is no separate compile step, no `.pyc`-style on-disk bytecode cache by default, and (until 3.2) no native code generation. YJIT changed that last bit but the rest still holds.

### The C Source Layout (github.com/ruby/ruby)

| Path | Purpose |
|:-----|:--------|
| `parse.y` | Bison grammar — produces `parse.c` at build time |
| `compile.c` | AST → ISeq compiler |
| `iseq.c` | ISeq object lifecycle, disasm, marshal |
| `vm.c` / `vm_eval.c` / `vm_exec.c` | Top-level VM dispatch |
| `vm_insnhelper.c` | Per-instruction implementations |
| `vm_method.c` | Method definition / lookup / cache |
| `vm_core.h` | Core structs: `rb_thread_t`, `rb_vm_t`, `rb_iseq_t`, `rb_control_frame_t` |
| `gc.c` | Garbage collector (mark/sweep/compact) |
| `string.c` / `array.c` / `hash.c` | Core data types |
| `class.c` / `object.c` | Object model |
| `thread.c` / `thread_pthread.c` | Native thread integration, GVL |
| `ractor.c` | Ractor (3.0+) |
| `yjit/` | YJIT JIT compiler (Rust) |
| `ext/` | Bundled C extensions (StringIO, Digest, OpenSSL, Socket, etc.) |
| `lib/` | Pure-Ruby standard library |

The total source tree is about 1.3 million lines of C plus 600 thousand lines of Ruby, including bundled gems.

### Core Structures (vm_core.h)

```c
// The per-VM (per-process) singleton — there is exactly ONE rb_vm_t.
typedef struct rb_vm_struct {
    VALUE self;                  // the VM as a Ruby object
    rb_global_vm_lock_t gvl;     // the Global VM Lock
    rb_nativethread_lock_t workqueue_lock;
    struct ccan_list_head workqueue;
    rb_serial_t fork_gen;        // generation counter, bumps on fork
    struct ccan_list_head waiting_pids;
    struct ccan_list_head waiting_grps;
    struct ccan_list_head waiting_fds;
    rb_postponed_job_queue_t *postponed_job_queue;
    int ractor_count;
    rb_objspace_t *objspace;     // GC heap
    /* ... ~150 more fields ... */
} rb_vm_t;

// Per-thread state. Holds the execution context.
typedef struct rb_thread_struct {
    struct ccan_list_node lt_node;
    VALUE self;
    rb_ractor_t *ractor;
    rb_vm_t *vm;
    rb_execution_context_t *ec;  // points to per-thread EC
    /* ... */
} rb_thread_t;

// Execution context — contains the call stack.
typedef struct rb_execution_context_struct {
    VALUE *vm_stack;             // operand + frame stack
    size_t vm_stack_size;
    rb_control_frame_t *cfp;     // current frame pointer
    rb_thread_t *thread_ptr;
    /* ... */
} rb_execution_context_t;

// One stack frame. Allocated in vm_stack, not on the C stack.
typedef struct rb_control_frame_struct {
    const VALUE *pc;             // program counter into iseq->iseq_encoded
    VALUE *sp;                   // stack pointer (operand stack top)
    const rb_iseq_t *iseq;       // currently executing iseq
    VALUE self;                  // method receiver
    const VALUE *ep;             // environment pointer (locals + block + flags)
    const void *block_code;
    VALUE *__bp__;
    /* ... */
} rb_control_frame_t;

// The bytecode object.
typedef struct rb_iseq_struct {
    VALUE flags;
    VALUE wrapper;
    struct rb_iseq_constant_body *body;  // the actual instructions + metadata
} rb_iseq_t;
```

The single most important fact is that **the operand stack is part of the same allocation as the call stack** — both grow on `ec->vm_stack`. This is why pushing too deep (`SystemStackError: stack level too deep`) can occur from either frame nesting OR runaway operand-stack growth in pathological code.

### The Dispatch Loop

`vm_exec_core` is a giant switch (or a computed-goto threaded dispatcher when built with `-fno-gcse`) over the YARV opcodes. Conceptually:

```c
INSN_ENTRY(getlocal):
{
    VALUE val = *(GET_EP() - operand_offset);
    *GET_SP() = val;
    INC_SP(1);
    NEXT_INSN();   // either goto *labels[*pc] (threaded) or break
}
```

CRuby's actual implementation generates the dispatch table from `insns.def` at build time via `tool/ruby_vm/`, producing `vm.inc`. The switch isn't hand-written.

---

## 2. The Parser and AST

### parse.y as Bison Grammar

`parse.y` is famously the largest single Bison `.y` file in any major open-source project — over 14,000 lines as of Ruby 3.3. The grammar is hand-tuned with mid-rule actions, custom lexer hooks (`parser_yylex`), heredoc state machines, regular-expression context tracking, and "command call" disambiguation that distinguishes `puts foo` (a call) from `puts.foo` (a method on `puts`).

```
program        : top_compstmt
top_compstmt   : top_stmts opt_terms
top_stmts      : none
               | top_stmt
               | top_stmts terms top_stmt
top_stmt       : stmt
               | keyword_BEGIN '{' top_compstmt '}'
stmt           : keyword_alias fitem fitem
               | keyword_undef undef_list
               | stmt keyword_if_mod expr_value
               | stmt keyword_unless_mod expr_value
               | stmt keyword_while_mod expr_value
               /* ... 60+ alternatives ... */
```

### The Heredoc Parsing Complexity

Heredocs are the canonical example of why Ruby's grammar resists straightforward LALR parsing. A heredoc body is gathered **after** the line that introduces it, but parsed **as if** it were inline:

```ruby
puts <<~END, "second arg"
  body line 1
  body line 2
END
```

The lexer maintains a queue of pending heredoc identifiers. When it hits a newline, it flushes the heredoc body before continuing on the next logical line. This is encoded in `parser_params->heredoc_indent`, `lex_strterm`, and a dedicated state machine in `parser_yylex`.

### Inspecting the AST

```ruby
require 'ripper'

src = "x = 1 + 2"
puts Ripper.sexp(src).inspect
# => [:program,
#     [[:assign,
#       [:var_field, [:@ident, "x", [1, 0]]],
#       [:binary, [:@int, "1", [1, 4]], :+, [:@int, "2", [1, 8]]]]]]
```

`Ripper` is a public stable wrapper around the parser. `RubyVM::AbstractSyntaxTree.parse` (3.0+) gives a richer view backed by `RNode`:

```ruby
ast = RubyVM::AbstractSyntaxTree.parse("def foo(x); x + 1; end")
ast.children
# => [[], [], #<RubyVM::AbstractSyntaxTree::Node:DEFN(id: :foo, line: 1)>]
ast.type   # => :SCOPE
```

### The ruby-parser Gem

For tooling that must run on multiple Ruby versions or parse legacy syntax, the `ruby-parser` gem (Ryan Davis) and the modern `parser` gem (Yuki "whitequark" Yano) reimplement Ruby's grammar in pure Ruby with explicit AST node classes (`s(:def, :foo, ...)`). RuboCop, Rubocop-AST, Sorbet's resolver, and Solargraph all build on `parser`. CRuby itself does not use these.

### Common Node Types

| Node | Meaning |
|:-----|:--------|
| `SCOPE` | A binding scope (top-level, def, class) — holds locals table |
| `BLOCK` | Sequence of statements |
| `IF` / `UNLESS` / `WHILE` / `UNTIL` | Control flow |
| `CALL` / `FCALL` / `VCALL` / `OPCALL` | Method calls (foo.bar / bar / bar / a + b) |
| `ITER` | Block iteration (`.each { }`) |
| `LASGN` / `IASGN` / `CVASGN` / `GASGN` | Local / instance / class / global var assignment |
| `LIT` / `STR` / `DSTR` | Literal int/sym, plain string, dynamic (interpolated) string |
| `ARRAY` / `HASH` | Container literals |
| `LAMBDA` | `->() { }` literal |
| `DEFN` / `DEFS` | def foo / def self.foo |

---

## 3. YARV — Yet Another Ruby VM

YARV (designed by Koichi Sasada) replaced the original AST-walking interpreter ("matz ruby") in **Ruby 1.9** (2007). It is a stack-based bytecode VM with about 100 opcodes, optimized opcodes (`opt_*`), and inline caches.

### The Core Instruction Set

| Opcode | Operands | Stack effect | Notes |
|:-------|:---------|:-------------|:------|
| `nop` | — | — | Used as a placeholder |
| `pop` | — | -1 | Discard top |
| `dup` | — | +1 | Duplicate top |
| `swap` | — | 0 | Swap top two |
| `topn N` | int | +1 | Push stack[top-N] |
| `setn N` | int | 0 | stack[top-N] = top |
| `putobject obj` | VALUE | +1 | Push frozen literal |
| `putobject_INT2FIX_0_` | — | +1 | Optimized push of `0` |
| `putobject_INT2FIX_1_` | — | +1 | Optimized push of `1` |
| `putstring str` | string | +1 | Push a fresh String (unless frozen-literal) |
| `putnil` / `putself` / `puttrue` / `putfalse` | — | +1 | Push special |
| `getlocal idx, level` | int, int | +1 | Read local from frame |
| `setlocal idx, level` | int, int | -1 | Write local |
| `getinstancevariable id, ivc` | id, cache | +1 | `@x` read with inline cache |
| `setinstancevariable id, ivc` | id, cache | -1 | `@x = ...` write |
| `getconstant id` | id | -1/+1 | Resolve `Constant` (pops scope) |
| `setconstant id` | id | -2 | `K = v` |
| `getclassvariable id, cvc` | id, cache | +1 | `@@x` |
| `setclassvariable id, cvc` | id, cache | -1 | `@@x = v` |
| `getglobal id` | id | +1 | `$x` |
| `setglobal id` | id | -1 | `$x = v` |
| `send ci, blockiseq` | callinfo, iseq | varies | General method call |
| `opt_send_without_block ci` | callinfo | varies | Specialized — no block |
| `invokesuper ci, blockiseq` | callinfo, iseq | varies | `super` |
| `invokeblock ci` | callinfo | varies | `yield` |
| `branchif label` | label | -1 | Jump if truthy |
| `branchunless label` | label | -1 | Jump if falsy |
| `branchnil label` | label | -1 | Jump if nil |
| `jump label` | label | 0 | Unconditional |
| `leave` | — | -1 | Return — pops frame |
| `throw type` | int | varies | Throw (return / break / next / retry) |
| `concatstrings n` | int | -n+1 | String interpolation builder |
| `concatarray` | — | -1 | `*a` splat in array literal |

The "optimized" opcodes (`opt_plus`, `opt_lt`, `opt_aref`, `opt_aset`, `opt_length`, `opt_size`, `opt_empty_p`, `opt_succ`, `opt_not`, `opt_neq`, `opt_regexpmatch2`) are fast paths for built-in operations on Integer/Float/String/Array/Hash. They check the receiver's class against a baked-in expectation; on mismatch they fall back to a full `send`.

### Inline Caches

YARV maintains two caches per call site:

**1. Inline Method Cache (IMC).** The `CALL_DATA` struct contains a `cc` (`call_cache`) field with the last-seen receiver class and a pointer to the resolved `rb_callable_method_entry_t`. On dispatch:

```c
if (cc->klass == CLASS_OF(receiver) && cc->method_state == GET_GLOBAL_METHOD_STATE()) {
    // Fast path: invoke cc->me directly
} else {
    // Slow path: lookup in receiver class, refill cc
}
```

A global counter (`rb_serial_t ruby_vm_global_method_state`) is bumped on any method definition, which invalidates every IMC at once. CRuby has refined this over time — modern caches use per-class versioning rather than the global state to reduce thrash.

**2. Constant Inline Cache (CIC).** `getconstant` checks a per-call-site cache against the current constant lookup state. Constant resolution otherwise walks the lexical scope chain (`cref`) plus the inheritance chain — expensive in the common case.

### Disassembly Walk-Through

```ruby
code = <<~RUBY
  def add(a, b)
    a + b
  end
  add(1, 2)
RUBY

puts RubyVM::InstructionSequence.compile(code).disasm
```

```
== disasm: #<ISeq:<compiled>@<compiled>:1 (1,0)-(4,9)>
0000 definemethod                   :add, add
0003 putself
0004 putobject_INT2FIX_1_
0005 putobject                      2
0007 opt_send_without_block         <calldata!mid:add, argc:2, FCALL|ARGS_SIMPLE>
0009 leave

== disasm: #<ISeq:add@<compiled>:1 (1,0)-(3,3)>
local table (size: 2, argc: 2 [opts: 0, rest: -1, post: 0, block: -1, kw: -1@-1, kwrest: -1])
[ 2] a@0<Arg>   [ 1] b@1<Arg>
0000 getlocal_WC_0                  a@0
0002 getlocal_WC_0                  b@1
0004 opt_plus                       <calldata!mid:+, argc:1, ARGS_SIMPLE>
0006 leave
```

`getlocal_WC_0` is the level-0 specialization of `getlocal` (locals in the current frame, not a captured outer scope) — the most common case, so it gets its own opcode.

### Compiling a Block

```ruby
puts RubyVM::InstructionSequence.compile("[1,2,3].each { |x| puts x }").disasm
```

```
== disasm: <main>
0000 duparray                       [1, 2, 3]
0002 send                           <calldata!mid:each>, block in <main>
0005 leave

== disasm: <block in main>
local table (size: 1, argc: 1)
[ 1] x@0<Arg>
0000 putself
0001 getlocal_WC_0                  x@0
0003 opt_send_without_block         <calldata!mid:puts, argc:1, FCALL>
0005 leave
```

Each block compiles to its own ISeq, attached as the `blockiseq` operand of `send`. The block is an ordinary instruction sequence that runs in a child frame whose `ep` chains to the parent's `ep` for closure capture.

---

## 4. The Object Model

### The Ancestor Spine

```
                    BasicObject              (Ruby 1.9+; near-empty class)
                         │
                       Object                (where Kernel methods land)
                         │
                       Module                (an instance is a Module — i.e., a Class is a Module)
                         │
                        Class                (instances of Class are classes)
```

`Kernel` is a `Module` mixed into `Object` — it provides `puts`, `p`, `gets`, `raise`, `lambda`, `caller`, `Integer()`, and ~150 other near-builtin "global" methods. Every object that descends from `Object` gets `Kernel`.

```ruby
String.ancestors      # => [String, Comparable, Object, Kernel, BasicObject]
String.class          # => Class
Class.superclass      # => Module
Module.superclass     # => Object
Object.superclass     # => BasicObject
BasicObject.superclass # => nil
```

### VALUE — The Tagged Pointer

In CRuby, every Ruby value is a C `unsigned long` called `VALUE`. On 64-bit platforms it's 8 bytes; on 32-bit, 4 bytes. The low bits are pointer tags:

```
64-bit VALUE (low bits):
   ...xxxxxxxx 1   →  Fixnum (integer in [-2^62, 2^62-1])
   ...xxxxxxxx 0   →  Pointer to RObject/RArray/RHash/RString/etc. (heap)
   ...xxxxx 1100   →  Flonum (small Float — embedded)
   ...xxxxxx 0100  →  Symbol (immediate)
   00000000 0000   →  false  (Qfalse = 0)
   00000000 1000   →  nil    (Qnil   = 8)
   00000010 0000   →  true   (Qtrue  = 20)
   00000110 0100   →  Qundef (sentinel — 'no value', distinct from nil)
```

So checking "is this value an integer?" is just `value & 1`. Checking "is this nil?" is `value == Qnil`. **No allocation** for small ints, true/false/nil, or symbols — they live in registers.

```c
// include/ruby/internal/value_type.h and value.h
#define RB_FIXNUM_P(f)    (((int)(SIGNED_VALUE)(f)) & RUBY_FIXNUM_FLAG)
#define RB_SYMBOL_P(x)    (((VALUE)(x) & ~((~(VALUE)0) << RUBY_SPECIAL_SHIFT)) \
                            == RUBY_SYMBOL_FLAG)
#define RB_NIL_P(v)       ((VALUE)(v) == RUBY_Qnil)
#define RB_TRUE_P(v)      ((VALUE)(v) == RUBY_Qtrue)
```

### RBasic — The Heap-Object Header

Every heap-allocated Ruby object starts with `RBasic`:

```c
struct RBasic {
    VALUE flags;     // GC mark bit, frozen flag, T_* type, embedded flags
    const VALUE klass;  // the object's class
};
```

`flags` is a 64-bit packed bitfield: low 5 bits are the **type tag** (`T_OBJECT`, `T_STRING`, `T_ARRAY`, `T_HASH`, `T_DATA`, `T_FILE`, `T_REGEXP`, ...). Other bits encode `FROZEN`, `FL_TAINT` (removed in 3.2), `FL_EXIVAR` (has external instance variables), `WB_PROTECTED` (write-barrier protected), generational mark bits, and per-type flags.

### Type-Specific Layouts

**T_OBJECT** (a generic instance — `class Foo; end; Foo.new`):

```c
struct RObject {
    struct RBasic basic;
    union {
        struct {
            uint32_t numiv;        // # of inline ivars (small obj fast path)
            VALUE *ivptr;          // heap-allocated ivar table (large obj)
            struct rb_id_table *iv_index_tbl;
        } heap;
        VALUE ary[ROBJECT_EMBED_LEN_MAX];  // 3 inline ivar slots (64-bit)
    } as;
};
```

Small objects with ≤3 instance variables embed them inline. Larger objects get a heap-allocated table.

**T_STRING** (RString — see Section 11 for the embedded/shared distinction).

**T_ARRAY** (RArray — see Section 14).

**T_HASH** (RHash — see Section 12).

**T_DATA** is the C-extension escape hatch — wraps an arbitrary `void*` with a `dmark`/`dfree` callback.

**T_NONE** is a freed slot in the GC heap awaiting reuse.

### Singleton Classes (Eigenclasses)

Every object can have a hidden **singleton class** between itself and its actual class:

```
obj  →  (singleton class of obj)  →  ActualClass  →  superclass  →  ...
```

```ruby
obj = Object.new
def obj.greet; "hi"; end

obj.singleton_class
# => #<Class:#<Object:0x...>>

obj.singleton_class.superclass
# => Object

obj.singleton_class.instance_methods(false)
# => [:greet]
```

For Class objects, the singleton chain mirrors the regular hierarchy (this is how class methods inherit):

```
MyClass  →  «MyClass»  →  «SuperClass»  →  «Object»  →  «BasicObject»  →  Class
```

Where `«X»` denotes "singleton class of X". This is why `def self.foo` in `MyClass` is callable from `MyClass.foo` AND inherited by subclasses.

### Method Resolution Order (MRO)

```ruby
class A; end
module M; end
class B < A; include M; end

B.ancestors
# => [B, M, A, Object, Kernel, BasicObject]
```

The algorithm — see `rb_class_inherited_p`, `rb_method_search`, and `class_search_ancestor` in `class.c`:

1. Start at the receiver's class (or its singleton class, if any).
2. Walk linearly through the precomputed ancestors list.
3. At each ancestor, check the method table (`m_tbl`) for the requested name.
4. First hit wins.

`prepend M` inserts `M` **before** the class itself in the chain (more on this in Section 17). `include M` inserts `M` immediately after the class. `extend M` is shorthand for `singleton_class.include(M)`.

---

## 5. Method Dispatch

### The Four Call Kinds

| Instruction | Use | Receiver |
|:------------|:----|:---------|
| `send` / `opt_send_without_block` | `obj.foo(args)` or `foo(args)` | Explicit/self |
| `invokesuper` | `super` | Self, but skip current method's class |
| `invokeblock` | `yield` | The block passed to the enclosing method |
| `invokesuper` | `super` with no args | Same as above, args from caller frame |

Each opcode reads a `CALL_DATA` operand bundling the method name, argc, kw flags, splat info, and inline cache slots.

### The Lookup Path (vm_method.c)

```c
const rb_callable_method_entry_t *
rb_callable_method_entry(VALUE klass, ID id) {
    rb_method_entry_t *me;
    VALUE defined_class;
    me = search_method(klass, id, &defined_class);
    if (UNDEFINED_METHOD_ENTRY_P(me)) return NULL;
    if (me->def->type == VM_METHOD_TYPE_NOTIMPLEMENTED) return NULL;
    return prepare_callable_method_entry(defined_class, id, me, FALSE);
}
```

`search_method` walks the ancestor chain storing the result in a per-class `m_tbl` lookup. The `me` (method entry) carries the `def` (definition: ISeq, C function, alias, attr_reader, etc.) and the original visibility.

### Method Cache Invalidation

Any of these bumps the global method state and invalidates IMCs across the whole VM:

- `def` of a new method
- `define_method` / `alias_method`
- `undef_method` / `remove_method`
- `Module#prepend` / `Module#include`
- Refinement activation (with finer-grained scoping)
- Reopening any class

In hot loops with monomorphic dispatch, the IMC is a one-comparison fast path. In megamorphic call sites (think Rails view rendering with many model types), the cache misses constantly and dispatch falls back to the slow path. YJIT specifically targets this pattern (see Section 10).

### Refinement-Aware Dispatch

Refinements (Ruby 2.0+, semantics finalized in 2.4) require dispatch to consider the **lexical scope** of the call site, not just the receiver's class. The compiler emits special call-site metadata when refinements are visible (`using SomeRefinement`), and dispatch consults the per-cref refinement table:

```ruby
module StringExt
  refine String do
    def shout; upcase + "!"; end
  end
end

module Greeter
  using StringExt
  def self.go(name) = "hello #{name.shout}"
end

Greeter.go("ruby")  # => "hello RUBY!"
"x".shout            # NoMethodError outside Greeter
```

The cost: refinement-aware call sites cannot use the simple monomorphic IMC.

### method_missing — The Safety Net

If lookup fails through the entire ancestor chain plus `Kernel`, CRuby calls `method_missing(name, *args, &block)` on the receiver. The default implementation (in `BasicObject#method_missing`) raises `NoMethodError`.

```ruby
class DynamicProxy
  def method_missing(name, *args, **kw, &blk)
    return "stubbed: #{name}" if name.to_s.start_with?("get_")
    super
  end

  def respond_to_missing?(name, include_private = false)
    name.to_s.start_with?("get_") || super
  end
end

DynamicProxy.new.get_user  # => "stubbed: get_user"
```

`respond_to_missing?` must be overridden in tandem so `respond_to?` returns truthful answers — the `Method` object machinery and `define_method` introspection both consult it. ActiveRecord uses this to support `find_by_email_and_status(...)` style methods.

---

## 6. The Global VM Lock (GVL)

The GVL (originally GIL — Global Interpreter Lock; renamed when Ractors arrived) is a `pthread_mutex_t` that **only one Ruby thread can hold while executing Ruby bytecode**.

$$\text{Ruby threads in YARV dispatch} \leq 1 \quad\text{(per Ractor)}$$

### Why It Exists

Like CPython's GIL, the GVL exists because CRuby's data structures are not internally thread-safe. Method tables, global constants, the instance-variable index table, the symbol pool, the GC mark bit array, the freelist — all assume single-writer access. Adding fine-grained locking would mean millions of atomic operations per second on objects that almost never see contention.

### When It Releases

The GVL is released:

1. **During blocking I/O.** `read(2)`, `write(2)`, `select(2)`, `poll(2)`, `kqueue`/`epoll_wait`, `accept(2)` — all wrapped in `rb_thread_call_without_gvl` or `rb_thread_io_blocking_region`.
2. **In C extensions that opt in.** `rb_thread_call_without_gvl(func, data1, ubf, data2)` runs `func` without the GVL (and `ubf` as the unblock function on signal). Writers must promise not to touch any Ruby VALUE inside `func`.
3. **At explicit thread-switch points.** Every ~10ms the running thread checks `RUBY_VM_INTERRUPTED_ANY` and may yield via `rb_thread_schedule`.
4. **On `sleep`, `Mutex#sleep`, `ConditionVariable#wait`, `Queue#pop` etc.** — all release while waiting.

### The Acquire/Release Cycle

```c
// thread_pthread.c (simplified)
static void
gvl_release(rb_global_vm_lock_t *gvl) {
    pthread_mutex_lock(&gvl->lock);
    gvl->owner = NULL;
    if (gvl->waiting > 0) pthread_cond_signal(&gvl->cond);
    pthread_mutex_unlock(&gvl->lock);
}

static void
gvl_acquire_common(rb_global_vm_lock_t *gvl, rb_thread_t *th) {
    pthread_mutex_lock(&gvl->lock);
    while (gvl->owner) {
        gvl->waiting++;
        pthread_cond_wait(&gvl->cond, &gvl->lock);
        gvl->waiting--;
    }
    gvl->owner = th;
    pthread_mutex_unlock(&gvl->lock);
}
```

CRuby actually uses a more complex multi-threaded scheduler with a designated `timer_thread` that pings the running thread to force preemption.

### Practical Impact

| Workload | Multi-thread benefit | Why |
|:---------|:---------------------|:----|
| I/O-bound (HTTP fetch, DB) | Yes — near-linear | GVL released during I/O |
| CPU-bound (pure Ruby compute) | No (often slower) | GVL contention + scheduler overhead |
| C ext that releases GVL (e.g. ZIP compression) | Yes | Ext explicitly drops GVL |
| `Process.fork` | Yes (separate VMs) | Each child has its own GVL |
| `Ractor` | Yes | Per-Ractor GVL since 3.0 |

### What `fork` Does to the GVL

`Process.fork` is supported on POSIX. The child process inherits the parent's memory image including the GVL state. CRuby resets the GVL (`fork_gen` increments) and **terminates all non-main threads in the child**, since their stacks are stale. This is why `fork` after thread spawning is a known footgun: connection pools, background loops, and timer threads vanish in the child unless explicitly recreated.

### Why It Hasn't Been Removed

Multiple attempts to remove the GVL have been proposed and (mostly) abandoned:

- **MVM (Multiple VM)** — never merged.
- **Guild** — renamed to Ractor and reframed as additive (per-Ractor GVL) rather than removal.
- **GIL-free MRI** — the C extension API exposes too many "I assume single-thread" assumptions; rewriting would break the gem ecosystem.

The compromise is Ractors (Section 7) and YJIT (Section 10) for performance, while keeping single-threaded code semantics unchanged.

---

## 7. Ractor — True Parallelism (3.0+)

`Ractor` (Ruby Actor — Koichi Sasada) ships in **Ruby 3.0** (Christmas 2020) as the answer to "true parallelism without removing the GVL." Each Ractor has its own GVL, its own method/constant cache state, and an isolated object space (with explicit message-passing).

```ruby
r = Ractor.new do
  msg = Ractor.receive       # block until message arrives
  Ractor.yield(msg.upcase)   # send back
end

r.send("hello")              # async send to r's mailbox
puts r.take                  # => "HELLO"
```

### Isolation Rules

A value can cross Ractor boundaries only if it is **shareable**:

| Shareable | Why |
|:----------|:----|
| Integer, Float, true, false, nil | Immutable atoms |
| Symbol | Interned, immutable |
| Frozen String / Array / Hash with frozen elements | Deeply frozen |
| Class / Module | Globally registered |
| `Ractor::Shareable`-marked objects | Explicit |

Mutable references — ordinary `Array`, `Hash`, `String`, `MyClass.new` — are **not shareable**. To pass them you must either:

1. **Copy** — `Ractor.send(obj)` (default) marshals and unmarshals.
2. **Move** — `Ractor.send(obj, move: true)` transfers ownership; sender can no longer reference `obj`.

### A Concurrent Pipeline

```ruby
PIPE = Ractor.new do
  loop do
    raw = Ractor.receive
    Ractor.yield(raw.bytesize)
  end
end

10.times.map { |i| PIPE.send("payload-#{i}"); PIPE.take }
```

### Limitations (As of 3.3)

- **Experimental warning** — emits `warning: Ractor is experimental` on use.
- **Globals are per-Ractor**, but most stdlib mutates implicit global state (e.g. `Time.zone`, `Encoding.default_internal`) and was not designed for it.
- **Many gems** (ActiveRecord, anything calling `require` lazily, anything with class-level mutex) raise `Ractor::IsolationError`.
- **Performance** — message-passing throughput is much lower than shared-memory; only worth it for embarrassingly-parallel CPU-bound work.
- **Rails support is essentially nil** as of Rails 7.x.

The realistic 3.x use cases: parallel JSON/CSV parsing, parallel cryptographic verification, parallel image processing, and benchmarks. For typical web apps, prefer Sidekiq workers (separate processes) or `Thread` with the GVL.

### "No GVL Contention Between Ractors"

Each Ractor has an independent execution context with its own GVL. Within a Ractor, threads still share that Ractor's GVL. So:

- 1 Ractor + N threads = serial Ruby execution (classic GVL behavior).
- N Ractors + 1 thread each = N parallel Ruby executions on N cores.
- N Ractors + M threads each = N parallel paths, M-way contended within each.

---

## 8. Fiber and the Fiber.scheduler API

Fibers are **cooperative coroutines** — green threads that yield explicitly. They ship since Ruby 1.9 (they predate Ractors by over a decade).

```ruby
fib = Fiber.new do
  Fiber.yield 1
  Fiber.yield 2
  3
end

fib.resume   # => 1
fib.resume   # => 2
fib.resume   # => 3
fib.resume   # FiberError: dead fiber called
```

### Implementation — m:1 in CRuby

CRuby implements fibers via **stack switching** — each fiber gets its own C stack (default 1 MB on glibc, adjustable via `RUBY_FIBER_VM_STACK_SIZE` and `RUBY_FIBER_MACHINE_STACK_SIZE`). `Fiber#resume` saves the current native stack pointer/registers and swaps in the fiber's stack. There is no scheduler — control transfers explicitly.

Crucially, all fibers in a thread share that thread's GVL. A fiber doing CPU work does **not** unblock other fibers; only blocking I/O or explicit `Fiber.yield` does. Fibers are an m:1 model (m fibers : 1 OS thread), still bound by the GVL.

### Fiber.scheduler (3.0+)

Ruby 3.0 introduced a pluggable **fiber scheduler** that hooks into blocking operations. Once a scheduler is installed for the current thread, `IO#read`, `IO#write`, `Kernel#sleep`, `Mutex#lock`, etc., transparently yield the fiber to the scheduler instead of blocking the thread:

```ruby
require "async"          # the canonical scheduler implementation

Async do
  3.times.map do |i|
    Async do
      sleep 1            # yields to scheduler — does NOT block the thread
      puts "done #{i}"
    end
  end.map(&:wait)
end
# All three "done" lines print after ~1 second total, not 3.
```

The `Async` gem (Samuel Williams) is the de-facto scheduler. `Falcon` is an Async-powered web server. `async-http`, `async-redis`, etc., provide non-blocking clients. None of this changes the GVL — it just lets one thread juggle many in-flight I/Os.

### Comparing the Three

| | Thread | Fiber | Ractor |
|:-|:-:|:-:|:-:|
| Parallelism | No (GVL) | No | Yes |
| Scheduling | OS preemptive | Cooperative | OS preemptive |
| Shared state | Yes (dangerous) | Yes (within thread) | Isolated (msg pass) |
| Stack | 8 MB native | 1 MB per fiber | own VM, own stacks |
| Switch cost | OS context switch | Stack swap (~ns) | Cross-Ractor msg |
| Use case | I/O-bound concurrency | High-fanout async I/O | CPU-bound parallelism |

---

## 9. Garbage Collection

CRuby's GC is a **generational, incremental, mark-and-sweep collector** with optional compaction (since 2.7). It runs single-threaded under the GVL.

### Evolution Timeline

| Version | GC milestone |
|:--------|:-------------|
| 1.8 | Conservative mark-and-sweep, stop-the-world |
| 1.9 | Lazy sweep |
| 2.0 | Bitmap marking (mark bits in side tables, not in object) |
| 2.1 | **Generational** (RGenGC) — minor / major split |
| 2.2 | Incremental marking (RIncGC) — chunked mark phase |
| 2.7 | `GC.compact` — optional compaction |
| 3.0 | Auto-compaction on major GC (opt-in) |
| 3.2 | Variable-width allocation (VWA) — multi-size heaps |
| 3.3 | Improved compaction, MMTk experimental backend |

### Generational Hypothesis

Most objects die young. CRuby tracks **young** (just allocated) vs **old** (survived `RVALUE_AGE_INC * 3` GCs, by default) objects:

| GC type | Scans | Frequency | Pause |
|:--------|:------|:----------|:------|
| **Minor GC** | Young objects + remembered set | Frequent | Short (~ms) |
| **Major GC** | All objects | Infrequent | Longer (~10ms-100ms) |

### Write Barrier

When an old object writes a reference to a young object, a **write barrier** records this so minor GC doesn't miss the young object. CRuby uses two flag bits:

- `WB_PROTECTED` — this object's mutators always invoke the write barrier.
- `WB_UNPROTECTED` — this object is "shady" — it can write without barrier.

Shady objects force a **major GC** more often. Most built-in types (Array, Hash, String, Object) are WB_PROTECTED. Some C extensions create shady objects, hurting GC.

```ruby
ObjectSpace.count_objects[:T_ARRAY]  # how many arrays are alive
ObjectSpace.count_objects(GC.stat)
```

### GC.stat Anatomy

```ruby
GC.stat
# {
#   :count                          => 28,        # total GC runs
#   :heap_allocated_pages           => 89,        # pages owned by GC
#   :heap_sorted_length             => 90,
#   :heap_allocatable_pages         => 0,
#   :heap_available_slots           => 36251,     # all slots
#   :heap_live_slots                => 35012,
#   :heap_free_slots                => 1239,
#   :heap_final_slots               => 0,
#   :heap_marked_slots              => 18324,
#   :heap_eden_pages                => 89,        # pages in active heap
#   :heap_tomb_pages                => 0,         # pages slated for free
#   :total_allocated_pages          => 89,
#   :total_freed_pages              => 0,
#   :total_allocated_objects        => 1825110,   # cumulative allocs
#   :total_freed_objects            => 1790098,
#   :malloc_increase_bytes          => 1245184,
#   :malloc_increase_bytes_limit    => 16777216,
#   :minor_gc_count                 => 25,
#   :major_gc_count                 => 3,
#   :compact_count                  => 0,
#   :read_barrier_faults            => 0,
#   :total_moved_objects            => 0,
#   :remembered_wb_unprotected_objects        => 0,
#   :remembered_wb_unprotected_objects_limit  => 162,
#   :old_objects                    => 17500,
#   :old_objects_limit              => 35000,
#   :oldmalloc_increase_bytes       => 524288,
#   :oldmalloc_increase_bytes_limit => 16777216
# }
```

### Tunables (Environment Variables)

| Variable | Default | Purpose |
|:---------|:-------:|:--------|
| `RUBY_GC_HEAP_INIT_SLOTS` | 10000 | Initial slot count |
| `RUBY_GC_HEAP_FREE_SLOTS` | 4096 | Min free slots after GC |
| `RUBY_GC_HEAP_GROWTH_FACTOR` | 1.8 | Heap grows by this factor when needed |
| `RUBY_GC_HEAP_GROWTH_MAX_SLOTS` | 0 (no max) | Cap on growth |
| `RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR` | 2.0 | Major GC trigger ratio for old objects |
| `RUBY_GC_MALLOC_LIMIT` | 16 MB | Triggers GC if malloc bytes exceed |
| `RUBY_GC_MALLOC_LIMIT_MAX` | 32 MB | Hard cap on malloc limit |
| `RUBY_GC_MALLOC_LIMIT_GROWTH_FACTOR` | 1.4 | How fast malloc limit grows |
| `RUBY_GC_OLDMALLOC_LIMIT` | 16 MB | Old-object malloc limit |

The Discourse and GitHub blog posts on tuning these (especially `RUBY_GC_HEAP_INIT_SLOTS=600000` for Rails warm starts) are well-known production lore.

### Bitmap Marking vs Inline Mark Bits

Pre-2.0 CRuby stored the mark bit inside `RBasic.flags`. This dirties every page during GC marking, defeating copy-on-write after fork (catastrophic for Unicorn/Puma forking models). 2.0 moved mark bits to a side bitmap (one bit per slot, packed in pages of 4 KB), preserving CoW pages.

### Compaction

`GC.compact` (manual) or `GC.auto_compact = true` runs a compaction pass: live objects are relocated to fill freelist gaps, references are updated, and tomb pages can be returned to the OS. Compaction requires the **write barrier** to be respected (objects with C-extension references using `T_DATA` may pin themselves and not move).

### Inspecting at Runtime

```ruby
ObjectSpace.count_objects
# => {:TOTAL=>132042, :FREE=>389, :T_OBJECT=>4123, :T_CLASS=>1234, :T_MODULE=>87, ...}

ObjectSpace.each_object(String).count   # all live Strings
ObjectSpace.memsize_of(some_obj)        # bytes used by this object

require "objspace"
ObjectSpace.dump_all(output: File.open("heap.json", "w"))   # heap dump for analysis
```

The `objspace` stdlib + `heapy` gem + Sam Saffron's `derailed_benchmarks` give per-allocation-site profiling.

---

## 10. JIT — YJIT and RJIT

### Timeline

| Year | Version | JIT |
|:----:|:-------:|:----|
| 2018 | 2.6 | **MJIT** (method-based, generates C, calls cc) — landed but slow |
| 2021 | 3.0 | MJIT improved, still niche |
| 2022 | 3.1 | **YJIT** introduced (initial release, x86_64 only) |
| 2023 | 3.2 | **YJIT goes default-built** (compiled in by default), arm64 added; **RJIT** (Ruby-written experimental JIT) replaces MJIT |
| 2024 | 3.3 | YJIT major perf gains, code-GC, splat support |

### YJIT — Basic Block Versioning

YJIT (Yet another JIT, Maxime Chevalier-Boisvert et al., originally at Shopify) is a **method-tracing JIT using basic-block versioning (BBV)**. Written in **Rust**, it compiles directly to native code (no C-compiler dependency).

The key insight of BBV: instead of speculating on a single type and bailing out, YJIT compiles **multiple specialized versions of each basic block**, each tied to a specific receiver/argument type signature. The runtime selects the right version by tagged-pointer check.

```ruby
# yjit-stats output (--yjit-stats)
$ ruby --yjit --yjit-stats -e 'sum = 0; 1_000_000.times { |i| sum += i }; puts sum'
500000500000
yjit: ratio_in_yjit:           99.7%
yjit: avg_len_in_yjit:         12.4
yjit: total_exits:               87
yjit: side_exit:                 12
yjit: invalidation_count:         3
```

### Enabling YJIT

```bash
# Command-line
ruby --yjit script.rb

# Environment
RUBYOPT="--yjit" rails server

# In code
RubyVM::YJIT.enable

# Stats (requires --yjit-stats build flag)
ruby --yjit --yjit-stats script.rb
```

### Workload Fit

YJIT is tuned for **megamorphic, deeply-nested, dispatch-heavy** code — exactly Rails. Reported wins:

| Benchmark | Speedup vs interpreter |
|:----------|:----------------------:|
| Liquid template rendering | 1.6x |
| ActiveRecord query building | 1.4x |
| Optcarrot (NES emulator) | 1.3x |
| Pure-tight-loop numeric | 1.05x (already fast in interp) |
| Memory overhead | +20-50 MB resident |

Shopify reported ~10% wall-clock improvement on production Storefront Renderer workloads after switching to YJIT 3.2 — at the scale of Shopify's traffic, that's meaningful.

### RJIT — The Educational JIT

RJIT (Takashi Kokubun, replaces MJIT in 3.2) is **written in Ruby itself**. It is a research/educational project, not a production target. It exists to lower the barrier for experimenting with Ruby JIT techniques without writing C/Rust.

```bash
ruby --rjit script.rb     # opt-in, experimental
```

### MJIT (Deprecated)

MJIT generated C code per hot method, dropped it to disk, invoked the system C compiler (`cc`), and `dlopen`-ed the result. It worked but had three flaws: (1) C compiler dependency in production, (2) huge per-method code size, (3) compile time dominated for short-lived programs. Removed in 3.2.

---

## 11. String — RString Internals

`RString` (`include/ruby/internal/core/rstring.h`, implementation in `string.c`) is one of the most heavily optimized objects in CRuby.

### Embedded Strings

Strings up to **23 bytes (on 64-bit)** are stored inline in the RString struct itself — no separate heap allocation:

```c
struct RString {
    struct RBasic basic;
    union {
        struct {
            long len;
            char *ptr;
            union {
                long capa;            // capacity (heap)
                VALUE shared;         // shared root (shared)
            } aux;
        } heap;
        struct {
            char ary[RSTRING_EMBED_LEN_MAX + 1];   // 23 bytes + NUL
        } embed;
    } as;
};
```

Whether a string is embedded or heap-allocated is determined by a flag bit in `RBasic.flags`. `RSTRING_LEN(str)` and `RSTRING_PTR(str)` branch on this flag.

### Shared Strings (Substring Without Copy)

`String#[range]` and similar substring operations return a **shared string** that references the original's buffer with an offset. The shared string's `ptr` points into the parent's buffer, and the parent is held alive via `aux.shared`.

```ruby
parent = "x" * 1000
sub = parent[100, 50]
# sub does not allocate a new char buffer; it references parent's buffer.
```

When the shared string is mutated (`sub << "!"`), CRuby allocates a fresh buffer for it (copy-on-write at the string level). The flag distinguishing shared from owned is `STR_SHARED` in flags.

### Encoding-Aware Design

Every string carries an **encoding** (an `rb_encoding *` — UTF-8, ASCII-8BIT, US-ASCII, Shift_JIS, EUC-JP, ...). Encoding affects `length`, `each_char`, `[index]`, and equality.

```ruby
"héllo".length       # => 5  (in UTF-8)
"héllo".bytesize     # => 6  (UTF-8 encodes é as 2 bytes)
"héllo".encoding     # => #<Encoding:UTF-8>

"héllo".force_encoding("ASCII-8BIT").length  # => 6 (now byte-counted)
```

Operations between strings of incompatible encodings raise `Encoding::CompatibilityError` unless one is ASCII-only. The `ASCII compatible` flag is cached on each string for fast comparison.

### frozen_string_literal Magic Comment

```ruby
# frozen_string_literal: true

s = "hello"     # already frozen, interned in the iseq's literal pool
s << "!"        # FrozenError
```

When set, every string literal in the file is a frozen, deduplicated value baked into the ISeq. No allocation per literal. This single magic comment can cut allocations in a Rails app by 30%+.

### Canonical Optimization

```ruby
# Bad — allocates a new String per call
def status; "active"; end

# Good — frozen literal, single allocation in lifetime
def status; "active".freeze; end       # Ruby <3.0 idiom

# Better — one frozen literal across the file
# frozen_string_literal: true
def status; "active"; end
```

In Ruby 3.0+, the file-level magic comment is the standard. RuboCop's `Style/FrozenStringLiteralComment` cop enforces it.

### Mutation in Place

`String#<<`, `String#concat`, `String#gsub!`, `String#replace`, `String#clear`, `String#chomp!`, `String#strip!` mutate in place and return `self` (or `nil` if no change for the `!` variants). Allocation count is the dominating cost in pure-Ruby string-heavy code; in-place ops crush a tight loop budget.

---

## 12. Hash — RHash Internals

The `Hash` implementation has been rewritten twice in CRuby's history. The current design (since **2.4**) is an **open-addressing hash table with a linear probing scheme**, plus an **array-stub fast path** for small hashes.

### Insertion Order Preservation

Since **Ruby 1.9**, Hash preserves **insertion order** — iteration follows the order keys were first inserted. This was implemented by linking entries in a doubly-linked list within the bucket array. The 2.4 rewrite kept this property using a packed entry array.

### Array Stub for Small Hashes

Hashes with **6 or fewer entries** skip the hash table entirely. They store entries in a packed array and do linear scans on lookup:

```c
struct RHash {
    struct RBasic basic;
    union {
        struct st_table *st;        // full hash table (>= 7 entries)
        rb_hash_ar_table_t ar;      // array stub (<= 6 entries)
    } as;
    /* ... */
};
```

For `{a: 1, b: 2}` style configuration hashes, this avoids the overhead of building a full hash table. Lookup at `n=6` linear-scan is faster than hash + probe in practice because of cache effects.

### The Hash Table

For 7+ entries, `RHash` uses `st_table` — Ruby's traditional open-address hash:

```c
struct st_table {
    const struct st_hash_type *type;
    st_index_t num_entries;
    unsigned int entry_power;       // bins = 1 << entry_power
    unsigned int bin_power;
    unsigned int size_ind;
    st_index_t num_bins;
    st_index_t rebuilds_num;
    st_index_t *bins;               // bin -> entry index
    st_table_entry *entries;        // ordered entries (insertion order)
};
```

Two arrays: `bins` is the open-addressing index (linear probing on collision), `entries` is the insertion-ordered storage. Lookup hashes the key, probes `bins` for the entry index, then dereferences `entries`. Iteration walks `entries` directly — preserving insertion order with no extra overhead.

Tombstones mark deleted slots. After enough deletes, the table rebuilds (`rebuilds_num` increments). Loadfactor stays below 0.5.

### Allocation Cost

```ruby
# Allocates a new Hash on every call:
def options; { mode: :fast, retries: 3 }; end

# Pre-allocate, freeze, return:
OPTIONS = { mode: :fast, retries: 3 }.freeze
def options; OPTIONS; end
```

Each `{ ... }` literal allocates a new Hash. The frozen-hash-literal optimization does not exist (unlike frozen-string-literal). Frozen module-level constants are the workaround.

### Common Pitfall — Default Block

```ruby
# Default value:
h = Hash.new(0)
h[:x] += 1                          # h => {x: 1}

# Default block — runs each missing key:
h = Hash.new { |hash, key| hash[key] = [] }
h[:x] << "a"
h[:x] << "b"                        # h => {x: ["a", "b"]}

# WRONG — shared default:
h = Hash.new([])
h[:x] << "a"                        # mutates the default!
h[:y] << "b"                        # h[:y] is the SAME array as h[:x]!
```

---

## 13. Symbol — Two-Tier Pool

Symbols (`:foo`) are **interned identifiers**. There is exactly one Symbol object per name in the VM.

### The DoS Vulnerability That Forced Mortal Symbols

Pre-Ruby-2.2, **all symbols were immortal** — once created, never freed. This was a memory exhaustion attack vector:

```ruby
# Pre-2.2: every distinct string here becomes a permanent Symbol
loop { rand(1_000_000).to_s.to_sym }
# Process eventually OOMs.
```

Frameworks doing `params[:user_id].to_sym` on user-controlled input (Rails before its mitigation) were vulnerable. The fix in **2.2** introduced the **two-tier symbol pool**:

| Tier | Source | GC'd? |
|:-----|:-------|:------|
| **Immortal** | Source code (`:foo`, `def foo`), `rb_intern("foo")` | Never |
| **Mortal** | `String#to_sym` at runtime | Yes — collected when no references |

```ruby
# Immortal — pinned forever
def foo; end
:bar
Symbol.all_symbols.length    # snapshot of current pool

# Mortal — eligible for GC when last reference drops
sym = "user_input".to_sym
sym = nil
GC.start
# the symbol may now be reclaimed
```

The distinction is internal — `Symbol#to_s`, `==`, and inspection look identical. The `rb_intern` C function returns immortal symbols; `rb_to_symbol`/`String#to_sym` returns mortal ones.

### Symbol Identity

```ruby
:foo.equal?(:foo)            # => true — same object
"foo".equal?("foo")          # => false — fresh allocations
"foo".freeze.equal?("foo".freeze)
# => true under Ruby 3.0+ frozen-string-literal pragma; depends on context
```

Symbol equality is pointer comparison — `O(1)`. String equality is `O(min(len_a, len_b))`. For hash keys hot in dispatch, symbols win.

### Symbol#to_proc

```ruby
[1, 2, 3].map(&:to_s)        # => ["1", "2", "3"]
```

`&:to_s` invokes `Symbol#to_proc`, which returns `proc { |x| x.to_s }`. Cached on the Symbol since 3.0+ — `Symbol#to_proc` returns the same Proc each time, so `&:to_s` allocates no Proc on hot calls.

---

## 14. Array — RArray Internals

`RArray` (`array.c`) is a **dynamically-sized contiguous VALUE array** with embedded vs heap distinction (like RString).

### Embedded vs Heap

Arrays with **3 or fewer elements (on 64-bit)** are stored inline:

```c
struct RArray {
    struct RBasic basic;
    union {
        struct {
            long len;
            union {
                long capa;
                VALUE shared_root;
            } aux;
            const VALUE *ptr;
        } heap;
        const VALUE ary[RARRAY_EMBED_LEN_MAX];   // 3 elements (64-bit)
    } as;
};
```

`RARRAY_LEN(ary)` and `RARRAY_AREF(ary, i)` branch on the embedded flag. For tiny arrays (function arguments, splatted return values), this avoids the heap.

### Geometric Growth

Heap arrays grow geometrically — `capa` doubles when full (with a tunable cap). Amortized `O(1)` push.

```ruby
a = []
1_000_000.times { |i| a << i }    # ~20 capacity reallocs total, not 1M
```

### Shared Arrays (Copy-on-Write Slices)

Like strings, `Array#dup` and `Array#[range]` for large slices may return shared arrays that point into the parent's buffer until mutation:

```ruby
parent = (0..999).to_a
slice = parent[100, 100]    # shared — no copy of the 100 ints
slice << :extra              # now slice gets its own buffer
```

### freeze and frozen?

```ruby
a = [1, 2, 3].freeze
a.frozen?     # => true
a << 4        # FrozenError

a = [1, 2, 3]
b = a.freeze.dup    # b is mutable; freeze returns self
b << 4              # ok
```

`Array#freeze` flips the `FL_FREEZE` bit. Subsequent mutation raises. `Array#dup` clears it on the copy.

### Array#each Hot Path

`Array#each` is implemented as a C loop in `array.c`:

```c
VALUE
rb_ary_each(VALUE ary) {
    long i;
    RETURN_SIZED_ENUMERATOR(ary, 0, 0, ary_enum_length);
    for (i=0; i<RARRAY_LEN(ary); i++) {
        rb_yield(RARRAY_AREF(ary, i));
    }
    return ary;
}
```

`rb_yield` invokes the block via `invokeblock`. The whole loop is one C function with no Ruby-level dispatch, which is why `Array#each` is ~3x faster than a hand-written `while` loop in Ruby.

---

## 15. Block, Proc, Lambda — The Three Callables

Three closure-like things, three different control-flow contracts.

### Comparison

| | Block | Proc | Lambda |
|:-|:------|:-----|:-------|
| Object? | No (syntax) | Yes (`Proc.new`) | Yes (`lambda` / `->`) |
| Lambda? | n/a | `false` | `true` |
| `return` | Returns from **enclosing method** | Returns from **enclosing method** (LocalJumpError if escaped) | Returns from **the lambda only** |
| Arity check | No (extras ignored, missing = nil) | No | Yes (ArgumentError) |
| Created by | `{ ... }` / `do ... end` after a call | `Proc.new`, `proc { }`, `&block` | `lambda { }`, `->() { }` |
| Conversion | `&` to/from Proc | `to_proc` | `to_proc` |
| Capture | Local lexical scope | Same | Same |

### The `return` Difference

```ruby
def proc_demo
  p = Proc.new { return 42 }       # return inside Proc returns from proc_demo
  p.call
  puts "unreachable"
end
proc_demo                          # => 42

def lambda_demo
  l = lambda { return 42 }         # return inside lambda returns from the lambda
  l.call
  puts "reachable"                 # this prints
end
lambda_demo                        # => "reachable"
```

If a Proc with `return` outlives its enclosing method:

```ruby
def make_proc
  Proc.new { return 42 }
end
make_proc.call
# LocalJumpError: unexpected return
```

### Arity Enforcement

```ruby
p = proc { |a, b| [a, b] }
p.call(1)                          # => [1, nil]   — missing arg becomes nil
p.call(1, 2, 3)                    # => [1, 2]     — extra args dropped

l = lambda { |a, b| [a, b] }
l.call(1)                          # ArgumentError: wrong number of arguments (given 1, expected 2)
```

### Block-to-Proc Conversion (`&`)

```ruby
def takes_block(&blk)              # converts the block to a Proc named blk
  blk.call(10)                     # explicit invocation
end
takes_block { |x| x * 2 }          # => 20

# Reverse direction:
my_proc = proc { |x| x.upcase }
["hi", "yo"].map(&my_proc)         # => ["HI", "YO"]

# Symbol#to_proc (a frequent idiom):
["hi", "yo"].map(&:upcase)         # => ["HI", "YO"]
```

The `&` operator in a method definition captures the block as a Proc. In a method call, it converts a Proc (or anything responding to `to_proc`) back into a block. `Method#to_proc` and `Symbol#to_proc` are the two stdlib `to_proc` implementations.

### yield vs block.call

`yield` is faster — no Proc allocation:

```ruby
# Fast — no allocation
def each_fast(arr)
  i = 0
  while i < arr.length
    yield arr[i]
    i += 1
  end
end

# Slower — allocates Proc on every call
def each_slow(arr, &block)
  i = 0
  while i < arr.length
    block.call(arr[i])
    i += 1
  end
end
```

In tight loops the difference is measurable. In normal code, prefer `&block` for clarity unless profiling says otherwise.

### Method Objects

`method(:name)` returns an `UnboundMethod` (when on a Class) or `Method` (when on an instance). `to_proc` converts:

```ruby
m = "hello".method(:upcase)
m.call                             # => "HELLO"
m.to_proc.call                     # => "HELLO"

["hi", "yo"].map(&"".method(:upcase))   # awkward but legal
```

---

## 16. Enumerable and Lazy Evaluation

`Enumerable` is a mixin requiring `#each`. Including it grants 50+ derived methods (`map`, `select`, `reject`, `reduce`, `find`, `group_by`, `tally`, `partition`, `min`, `max`, `sum`, `count`, `chunk`, `slice_when`, `each_with_index`, `each_with_object`, `flat_map`, `zip`, `take`, `drop`, `take_while`, `drop_while`, ...).

### The Canonical Implementation

Most Enumerable methods are implemented in `enum.c` (C) for performance, but the conceptual Ruby implementation is illustrative:

```ruby
module MyEnumerable
  # Required: #each yielding each element

  def map
    result = []
    each { |x| result << yield(x) }
    result
  end

  def select
    result = []
    each { |x| result << x if yield(x) }
    result
  end

  def reduce(initial = nil)
    acc = initial
    each do |x|
      acc = acc.nil? ? x : yield(acc, x)
    end
    acc
  end

  def each_with_object(memo)
    each { |x| yield(x, memo) }
    memo
  end
end
```

### Including It

```ruby
class LinkedList
  include Enumerable

  def initialize; @head = nil; end

  def each
    node = @head
    while node
      yield node.value
      node = node.next
    end
    self
  end
end

list = LinkedList.new
# Now list.map, list.select, list.tally, etc. all work.
```

### Enumerator and Lazy

`#each` without a block returns an `Enumerator`:

```ruby
e = [1, 2, 3].each
e.next    # => 1
e.next    # => 2
e.next    # => 3
e.next    # StopIteration
```

`Enumerator::Lazy` (`#lazy`) builds a chain of pending operations without executing them until forced:

```ruby
result = (1..Float::INFINITY)
  .lazy
  .map  { |x| x * x }
  .select { |x| x.even? }
  .first(5)
# => [4, 16, 36, 64, 100]
```

Without `.lazy`, `(1..Float::INFINITY).map { ... }` would loop forever building an infinite array. With `.lazy`, each element is processed through the chain only as far as needed for the final consumer (`first(5)`).

`Enumerator::Lazy#force` realizes the lazy chain into an Array:

```ruby
(1..100).lazy.map { |x| x * x }.select { |x| x.even? }.force
# => [4, 16, 36, 64, ..., 10000]
```

### Lazy vs Eager Performance

| | Eager | Lazy |
|:-|:------|:-----|
| Allocations | One Array per intermediate step | None until force/first |
| Termination | Always full traversal | Stops at first consumer satisfaction |
| Best for | Small bounded collections | Infinite or filtering early-out |

For finite small collections, eager is usually faster (less overhead per element). Lazy wins on infinite streams or when only the first N matches are needed.

### tally — A Recent Addition (2.7+)

```ruby
%w[apple banana apple cherry banana apple].tally
# => {"apple"=>3, "banana"=>2, "cherry"=>1}
```

`tally` is implemented in C as a single-pass `Hash.new(0); each { |x| h[x] += 1 }; h` — drastically faster than the Ruby equivalent for large collections.

---

## 17. Module Lookup and Refinements

### include vs prepend vs extend

```ruby
module Greet
  def greet; "hi"; end
end

class A
  include Greet
end

class B
  prepend Greet
end

class C; end
C.new.extend(Greet)

A.ancestors    # => [A, Greet, Object, Kernel, BasicObject]
B.ancestors    # => [Greet, B, Object, Kernel, BasicObject]
C.new.singleton_class.ancestors   # => [#<Class:#<C:0x...>>, Greet, C, Object, Kernel, BasicObject]
```

| Operation | Insertion in MRO |
|:----------|:----------------|
| `include M` | After the including class |
| `prepend M` | **Before** the including class |
| `extend M` (instance) | Into the singleton class of the instance |
| `extend M` (class) | Into the singleton class of the class (= class methods) |

### Why prepend Matters

`prepend` lets you wrap a method **before** the class itself in the lookup chain — so calling `obj.foo` calls the prepended `foo`, which can `super` to the original:

```ruby
module LoggingFoo
  def foo
    puts "calling foo with #{caller.first}"
    result = super
    puts "foo returned #{result.inspect}"
    result
  end
end

class MyService
  prepend LoggingFoo
  def foo; 42; end
end

MyService.new.foo
# calling foo with ...
# foo returned 42
# => 42
```

This is how Rails's `ActiveSupport::Concern` and various aspect-oriented patterns work. Pre-2.0 it was done with `alias_method_chain`, which was clunky; `prepend` cleaned it up.

### Constant Lookup (rb_const_lookup)

Constant resolution is **distinct from method lookup** — it walks the **lexical scope chain (`cref`) first**, then the inheritance chain:

```ruby
module Outer
  X = 1
  class Foo
    X = 2
    def show; X; end           # => 2 — lexically nearest wins
  end
end
Outer::Foo.new.show            # => 2

class Bar
  X = 3
  Outer::Foo.new.show          # => 2 (lexical scope of show is Outer::Foo, NOT call site)
end
```

This is why `Module.nesting` matters and why constants leaked from `eval` strings can confuse: `eval` defaults to a different `cref`.

### Refinements (2.0+, finalized in 2.4)

Refinements provide **scoped monkey-patching** — modifications visible only inside a `using` block:

```ruby
module StringExt
  refine String do
    def shout; upcase + "!"; end
  end
end

class Greeter
  using StringExt
  def go(name) = "hello #{name.shout}"
end

Greeter.new.go("ruby")    # => "hello RUBY!"
"x".shout                 # NoMethodError — refinement not active here
```

Refinement-aware dispatch costs more than monomorphic IMC. Each call site in a refinement-using scope must consult the per-cref refinement map. For this reason, refinements never quite displaced full monkey-patching for performance-critical code; they're best for library authors who want safer, scoped extensions.

### Per-Call-Site Cache Invalidation

Adding a method to a module at runtime invalidates **every method cache for any class with that module in its ancestors**. In a long-running Rails process, gem authors `define_method` at boot time and (mostly) stop, so caches stabilize. Hot-code-reloading dev mode pays the invalidation cost continuously.

---

## 18. Bundler Resolver Algorithm

Bundler — the de facto dependency manager for Ruby gems — solves a **constraint-satisfaction problem**: find the latest set of gem versions that satisfy every dependency in the Gemfile and all transitive `.gemspec` requirements.

### Molinillo

The resolver is `Molinillo` (Spanish for "little mill" — it resolves), extracted from Bundler in 2014 and now used by RubyGems, CocoaPods, and others. It is a **DPLL-style backtracking solver** with conflict-directed backjumping.

```
state := { activated: {}, working_set: [...root_deps...], frontier: [] }

loop:
  if working_set.empty: return activated  // done
  dep := working_set.pop
  for each candidate version of dep (newest-first):
    if candidate compatible with activated:
      push candidate to activated
      push candidate's deps onto working_set
      recurse
      if recurse succeeded: return
      else: pop candidate, try next version
  // no candidate worked — backtrack: report conflict
```

In practice, the resolver:

1. Builds a graph of gem dependency requirements.
2. Picks an unresolved gem.
3. Tries the newest matching version.
4. Recurses into its dependencies.
5. On conflict (e.g., `gemA` requires `rack >= 2.2`, `gemB` requires `rack < 2`), backtracks to the most recent decision that could change the conflicting subset.
6. Records the conflict to avoid revisiting equivalent states.

### "Resolving dependencies..." — Why It Spins

Worst-case complexity is exponential in the number of conflicting gems. Real-world Gemfiles with hundreds of transitive deps and a few overlapping constraints can take many seconds. The 2017 `bundler` blog post on "Why is Bundler slow?" walks through the conflict-tracking improvements that brought typical resolution from minutes to seconds.

### Lockfile

After resolution, Bundler writes `Gemfile.lock` pinning every gem to an exact version. Subsequent `bundle install` reads the lockfile and skips resolution entirely — that's the fast path. `bundle update` re-resolves from scratch.

---

## 19. RBS and Sorbet — Static Typing

Ruby is dynamically typed. Two community-led type systems retrofit static checking.

### RBS (Ruby Signature)

**RBS** is the standard Ruby type signature language, included with **Ruby 3.0+**. Type signatures live in **separate `.rbs` files** — Ruby source remains untyped.

```rbs
# user.rbs
class User
  attr_reader name: String
  attr_reader age: Integer

  def initialize: (name: String, age: Integer) -> void
  def greet: () -> String
  def adult?: () -> bool
end

module Admin
  def grant: (User) -> User
end
```

`rbs` ships with the core distribution. The standard library has comprehensive `.rbs` files. `gem_rbs_collection` is the community-maintained set for popular gems.

### TypeProf

**TypeProf** (also bundled with 3.0+) is a **static type analyzer / inferer**. It runs your code abstractly — without executing it — and emits inferred RBS:

```bash
typeprof app/models/user.rb -o sig/user.rbs
```

TypeProf's analysis is unsound (it sometimes misses paths) but pragmatic.

### Steep

**Steep** is a stand-alone type checker that consumes RBS and Ruby source and reports errors:

```bash
gem install steep
steep init
steep check
```

Steep integrates with VS Code via Solargraph or its own LSP server. It's the typical tool for "I've adopted RBS, now check it."

### Sorbet

**Sorbet** is **Stripe's** typed Ruby system. It predates RBS (open-sourced 2019) and uses a fundamentally different approach:

1. Type annotations are **inline in Ruby source**, using the `T::Sig` DSL:

```ruby
require "sorbet-runtime"

class User
  extend T::Sig

  sig { params(name: String, age: Integer).void }
  def initialize(name:, age:); @name = name; @age = age; end

  sig { returns(String) }
  def greet; "hi #{@name}"; end
end
```

2. A `# typed: true` magic comment per file controls the strictness level (`ignore`, `false`, `true`, `strict`, `strong`).

3. The Sorbet checker is a **C++ binary** that processes thousands of files per second — much faster than Steep on large codebases.

4. **Runtime checks** — `sorbet-runtime` validates types at runtime by wrapping methods with sig-checking shims. Has measurable overhead (~5-15%); production deployments often disable runtime checks via `T::Configuration.default_checked_level = :never`.

### Comparison

| | RBS + Steep | Sorbet |
|:-|:-----------|:-------|
| Annotation location | Separate `.rbs` files | Inline in `.rb` source |
| Type system | Structural-ish, with generics | Nominal, with generics |
| Runtime cost | None (static only) | Optional `sorbet-runtime` |
| Large-codebase speed | Slower | Faster (C++ checker) |
| Community | Ruby core team + matz | Stripe + community |
| Generics syntax | `Array[Integer]` | `T::Array[Integer]` |
| Tooling | `steep`, LSP via Solargraph | `srb`, Sorbet LSP |

In 2026, **RBS is the standard** Ruby community endorses, but **Sorbet is the de facto choice** at large engineering organizations (Stripe, Shopify, GitHub, Coinbase) due to its speed and inline-annotation ergonomics. Both can coexist — Sorbet has tooling to consume RBS files.

---

## 20. C Extensions

The Ruby C API lets you write native extensions — gems implemented in C that load like pure-Ruby gems.

### Hello, Extension

`ext/hello/hello.c`:

```c
#include "ruby.h"

static VALUE
hello_greet(VALUE self, VALUE name) {
    Check_Type(name, T_STRING);
    char *cstr = StringValueCStr(name);
    char buf[256];
    snprintf(buf, sizeof(buf), "Hello, %s, from C!", cstr);
    return rb_str_new_cstr(buf);
}

void
Init_hello(void) {
    VALUE mod = rb_define_module("Hello");
    rb_define_singleton_method(mod, "greet", hello_greet, 1);
}
```

`ext/hello/extconf.rb`:

```ruby
require "mkmf"
create_makefile("hello/hello")
```

Build:

```bash
cd ext/hello
ruby extconf.rb           # generates Makefile
make                      # produces hello.bundle (macOS) or hello.so (Linux)
```

Use:

```ruby
require_relative "ext/hello/hello"
Hello.greet("world")      # => "Hello, world, from C!"
```

### Key API Functions

| C function | Purpose |
|:-----------|:--------|
| `rb_define_module(name)` | Create a Module |
| `rb_define_class(name, super)` | Create a Class |
| `rb_define_method(klass, name, fn, argc)` | Add an instance method |
| `rb_define_singleton_method(obj, name, fn, argc)` | Add a singleton method |
| `rb_define_const(klass, name, value)` | Define a constant |
| `rb_intern(str)` | Get or create an immortal Symbol ID |
| `rb_str_new_cstr(str)` / `rb_str_new(buf, len)` | Make a String |
| `RSTRING_PTR(s)` / `RSTRING_LEN(s)` | Read a String |
| `INT2NUM(i)` / `LONG2NUM(l)` | C int → Ruby Integer |
| `NUM2INT(v)` / `NUM2LONG(v)` | Ruby Integer → C int |
| `rb_ary_new()` / `rb_ary_push(ary, v)` | Build an Array |
| `RARRAY_LEN(a)` / `RARRAY_AREF(a, i)` | Read Array |
| `rb_hash_new()` / `rb_hash_aset(h, k, v)` / `rb_hash_aref(h, k)` | Hash |
| `rb_funcall(obj, mid, argc, ...)` | Call a method |
| `rb_yield(value)` | yield to the block |
| `rb_raise(klass, fmt, ...)` | Raise an exception |
| `rb_protect(fn, arg, &state)` | Run with exception protection |
| `Data_Wrap_Struct(klass, mark, free, ptr)` | Wrap a C struct as T_DATA |
| `rb_gc_mark(value)` | Mark a VALUE during GC |
| `rb_thread_call_without_gvl(fn, data, ubf, ubf_data)` | Run without GVL |

### GC Integration

When wrapping C state in a Ruby object, you must tell the GC how to mark held VALUEs:

```c
typedef struct {
    VALUE name;
    VALUE buffer;
    int counter;
} my_state_t;

static void my_state_mark(void *ptr) {
    my_state_t *st = ptr;
    rb_gc_mark(st->name);
    rb_gc_mark(st->buffer);
}

static void my_state_free(void *ptr) {
    free(ptr);
}

static const rb_data_type_t my_state_type = {
    "MyState",
    { my_state_mark, my_state_free, NULL, },
    NULL, NULL,
    RUBY_TYPED_FREE_IMMEDIATELY,
};

static VALUE my_state_new(VALUE klass) {
    my_state_t *st;
    VALUE obj = TypedData_Make_Struct(klass, my_state_t, &my_state_type, st);
    st->name = Qnil;
    st->buffer = Qnil;
    st->counter = 0;
    return obj;
}
```

Without `rb_gc_mark` calls, the held VALUEs would be eligible for GC and could vanish under the C extension's feet.

### Releasing the GVL

```c
static void *do_blocking_work(void *data) {
    sleep(1);                     // pretend we're blocking on something
    return NULL;
}

static VALUE my_blocking_call(VALUE self) {
    rb_thread_call_without_gvl(do_blocking_work, NULL, RUBY_UBF_IO, NULL);
    return Qnil;
}
```

Inside `do_blocking_work`, you must NOT touch any VALUE — no Ruby allocation, no method calls, no string ops. The GVL is released, the GC may run, and your thread is operating outside Ruby's invariants.

### The Gem Package

```
my_gem/
├── lib/
│   └── my_gem.rb              # Ruby entry: require "my_gem/my_gem"
├── ext/
│   └── my_gem/
│       ├── extconf.rb
│       ├── my_gem.c
│       └── my_gem.h
├── my_gem.gemspec             # spec.extensions = ["ext/my_gem/extconf.rb"]
└── Rakefile                   # task :compile via rake-compiler
```

`bundle install` runs `extconf.rb` and `make` for each extension. `gem install` does the same.

---

## 21. Idiomatic vs Performant Patterns

A short reference for the common cost-vs-clarity tradeoffs.

### Method Call Overhead

A method call costs ~100-300 ns of pure interpreter overhead — IMC lookup, frame push, ep setup, leave, return. In a tight inner loop with millions of iterations, this dominates. YJIT can cut this by 5-10x for monomorphic call sites.

### Instance Variable Access

```ruby
class Foo
  def initialize; @x = 1; end
  def get_attr; @x; end                    # fastest — direct RObject ivar slot
end

class Bar
  attr_reader :x
  def initialize; @x = 1; end
  def get_attr; x; end                     # slower — calls reader method
end
```

`attr_reader` IS implemented as a C-level fast path (it generates an `getinstancevariable` instruction without a method call), so the gap is small. But `@x` directly is still a single instruction; calling `x` is an opt_send + a frame.

### Frozen String Literal

```ruby
# Without # frozen_string_literal: true
def fmt(name); "hello, #{name}"; end       # allocates a new String each call

# With # frozen_string_literal: true at top of file
def fmt(name); "hello, #{name}"; end       # the LITERAL "hello, " is frozen,
                                           # but interpolation still allocates.
                                           # Just no extra dup of "hello, ".
```

Heredocs, plain literals, and the `"" + x` form all benefit. Interpolation `"x #{y}"` always allocates the result string (since it's a fresh String).

### Avoid Symbol Creation in Hot Loops

```ruby
# BAD — creates a fresh mortal symbol per iteration
loop do
  request_data["#{prefix}_user".to_sym]
end

# GOOD — interns the symbol once at boot
USER_KEY = :"#{prefix}_user".freeze    # immortal at this point
loop do
  request_data[USER_KEY]
end
```

### tally vs each_with_object

```ruby
# Pre-2.7 idiom:
words.each_with_object(Hash.new(0)) { |w, h| h[w] += 1 }

# 2.7+ — built-in C implementation, faster:
words.tally
```

### map.compact vs filter_map

```ruby
# Two passes over the array:
ary.map { |x| transform(x) }.compact

# Single pass, 2.7+:
ary.filter_map { |x| transform(x) }   # nil results are dropped
```

### Lazy vs Eager

```ruby
# If you only need the first 10 matches and the array is huge, lazy wins:
data.lazy.select { |x| expensive_test(x) }.first(10)

# If the array is small or you need them all, eager is faster:
data.select { |x| expensive_test(x) }
```

### Hash Default Block — Beware Shared Defaults

```ruby
# WRONG — default Array is shared
groups = Hash.new([])
groups[:a] << "one"
groups[:b] << "two"
# Both groups[:a] and groups[:b] are the SAME array now.

# RIGHT — each missing key gets a fresh Array
groups = Hash.new { |h, k| h[k] = [] }
groups[:a] << "one"
groups[:b] << "two"
```

### String Building

```ruby
# Allocates many intermediate Strings:
result = ""
data.each { |x| result += x.to_s }

# In-place — single buffer that grows:
result = String.new
data.each { |x| result << x.to_s }

# Best for known-final-size:
result = data.map(&:to_s).join
```

### Avoid Block Capture When Not Needed

```ruby
# Slow — captures block as Proc on every call
def each_double(arr, &block)
  arr.each { |x| block.call(x * 2) }
end

# Fast — yields directly, no Proc allocation
def each_double(arr)
  arr.each { |x| yield x * 2 }
end
```

### Cost Model Summary

| Operation | Order |
|:----------|:------|
| Local variable read/write | < 5 ns |
| `@ivar` read/write | ~ 10-20 ns |
| Method call (cache hit) | ~ 100-300 ns |
| Method call (cache miss) | ~ 1-5 µs |
| Block yield | ~ 100 ns |
| Proc.new + call | ~ 500 ns |
| Hash lookup (small) | ~ 50-100 ns |
| Hash lookup (large) | ~ 100-200 ns |
| String allocation | ~ 100-500 ns + memcpy |
| Symbol comparison | ~ 5 ns (pointer eq) |
| String comparison | ~ O(min len) memcmp |
| GC minor pause | ~ 1-5 ms |
| GC major pause | ~ 10-100 ms |

---

## Prerequisites

- **Object-oriented programming** — classes, instances, inheritance, mixins, message-passing.
- **Closures and higher-order functions** — capturing scope, blocks vs procs vs lambdas, partial application.
- **Garbage collection fundamentals** — generational hypothesis, write barriers, mark/sweep, compaction.
- **Stack-based virtual machines** — operand stacks, frame pointers, instruction dispatch, inline caches.
- **C basics** — `struct`, pointers, `void*`, the preprocessor — for reading CRuby internals and writing extensions.
- **Pthreads or equivalent** — mutex/condvar semantics, the meaning of "release the lock around blocking IO."
- **Bison/Yacc grammar mental model** — for parse.y.
- **Familiarity with the Ruby language at user level** — the companion sheet `sheets/languages/ruby.md` has the practical reference.

## Complexity

| Operation | Complexity | Notes |
|:----------|:-----------|:------|
| Method dispatch (cache hit) | O(1) | Single class-pointer comparison |
| Method dispatch (cache miss) | O(depth × method-table-size) | Walk ancestors |
| Constant lookup | O(cref-depth + ancestor-depth) | Lexical first, then inheritance |
| Hash lookup (open addressing) | Expected O(1), worst O(n) | Linear probing |
| Hash lookup (small-array stub) | O(n) for n ≤ 6 | Faster than hash for tiny |
| Array random access | O(1) | Contiguous VALUE buffer |
| Array push | Amortized O(1) | Geometric growth |
| Array prepend (`unshift`) | O(n) | Memmove |
| String concatenation (`<<`) | Amortized O(len of rhs) | In-place geometric |
| String concatenation (`+`) | O(a + b) | Allocates new String |
| GC minor | O(young + remembered) | Most objects skipped |
| GC major | O(all objects) | Full mark + sweep |
| Bundler resolve | Worst-case exponential | DPLL with conflict learning |
| Symbol comparison | O(1) | Pointer equality |
| String hashing | O(n) | Cached after first hash |

## See Also

- [ruby](../../sheets/languages/ruby.md) — practical reference companion sheet (syntax, idioms, stdlib).
- [polyglot](../../sheets/languages/polyglot.md) — Ruby's place among modern dynamic languages, FFI patterns.
- [python](../../detail/languages/python.md) — CPython internals (GIL, refcount + GC, bytecode VM) — direct architectural sibling.
- [javascript](../../detail/languages/javascript.md) — V8 engine, hidden classes, inline caches in Crankshaft/TurboFan/Ignition — same ideas, different VM.

## References

- **The Ruby Language Specification** — https://www.ruby-lang.org/en/documentation/ — official docs (en + ja).
- **Ruby Core Documentation** — https://docs.ruby-lang.org/en/master/ — `String`, `Array`, `Hash`, `Module`, `Class`, `Thread`, `Fiber`, `Ractor`, `GC`, `RubyVM`, `ObjectSpace`.
- **Ruby Under a Microscope** — Pat Shaughnessy (No Starch, 2013). The deep-dive book on YARV, the GC, the parser, and method lookup. Required reading.
- **Programming Ruby (the "Pickaxe")** — Dave Thomas et al. The canonical user-level book; covers the language and stdlib at depth.
- **Eloquent Ruby** — Russ Olsen. Style and idiom — when to use blocks, when to use modules, the Ruby Way of writing readable code.
- **Metaprogramming Ruby 2** — Paolo Perrotta. The clearest explanation of the object model, eigenclasses, and how method_missing/define_method/eval interact.
- **The Ruby Source Code Itself** — https://github.com/ruby/ruby — `parse.y`, `compile.c`, `vm_core.h`, `gc.c`, `string.c`, `array.c`, `hash.c`, `vm_method.c`, `class.c`. The truth is in the C.
- **Koichi Sasada's papers and talks** — YARV (RubyKaigi 2007), Ractor (RubyKaigi 2020), Fiber.scheduler (RubyKaigi 2020).
- **Maxime Chevalier-Boisvert et al., "YJIT: Building a New JIT Compiler for CRuby"** — MPLR 2023.
- **Aaron Patterson's blog** — https://tenderlovemaking.com — GC compaction, write barriers, ObjectSpace dumps.
- **Sam Saffron, Discourse blog** — Ruby memory tuning, RUBY_GC_HEAP_INIT_SLOTS lore.
- **The Rubinius project (historical)** — alternative implementation that influenced YARV's instruction set design.
- **JRuby** — https://www.jruby.org — JVM-hosted Ruby; useful contrast on parallelism.
- **TruffleRuby** — https://github.com/oracle/truffleruby — GraalVM-hosted Ruby; the partial-evaluation school.
- **mruby** — https://mruby.org — minimal embedded Ruby (matz's reference for ISO standardization).
- **RBS Documentation** — https://github.com/ruby/rbs.
- **Sorbet Documentation** — https://sorbet.org/docs.
- **Bundler / Molinillo** — https://github.com/CocoaPods/Molinillo, https://bundler.io/blog.
- **Ruby Issue Tracker** — https://bugs.ruby-lang.org — the canonical place to follow CRuby development. Search the `Feature` tracker for proposals.
- **The Ruby Weekly newsletter** — https://rubyweekly.com — keeps current with releases, gems, and CRuby internals discussion.
