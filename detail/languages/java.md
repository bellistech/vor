# The Internals of the JVM — HotSpot, GC, Class Loading, and Concurrency

> *Java is not the JVM, and the JVM is not just "Java". HotSpot is a sophisticated dynamic optimizer: an interpreter, two JIT compilers, several garbage collectors, a class loading and verification system, and a memory model that took the better part of a decade to formalize. Understanding the runtime is what separates "I write Java" from "I run Java in production at scale."*

---

## 1. The JVM Architecture

### 1.1 Big Picture

A `.java` source file is compiled by `javac` to platform-independent JVM bytecode (`.class` files), packaged into JARs, then executed by a Java Virtual Machine implementation. HotSpot — the reference implementation shipped with OpenJDK and Oracle JDK — is the dominant production JVM.

```
Source (.java)
  └─► javac (compile-time)
        └─► Bytecode (.class)
              └─► JVM (runtime)
                    ├─► Class loader subsystem
                    ├─► Bytecode verifier
                    ├─► Runtime data areas (heap, stacks, metaspace)
                    ├─► Execution engine
                    │     ├─► Interpreter (template interpreter)
                    │     ├─► C1 (client compiler)
                    │     └─► C2 (server compiler)
                    ├─► Garbage collector (one of: Serial, Parallel, G1, ZGC, Shenandoah, Epsilon)
                    └─► JNI / Panama bridge to native code
```

### 1.2 The Execution Engine: Interpreter + Two JITs

HotSpot uses **tiered compilation**, which is *not* a fall-back system but a continuous spectrum:

| Tier | Compiler | What it does |
|:----:|:---------|:-------------|
| 0 | Interpreter | Walk bytecode opcodes one at a time; collect invocation/branch counters |
| 1 | C1 | Quick compile, no profiling — for code that won't get hot |
| 2 | C1 | Limited profiling (invocation + back-edge counters) |
| 3 | C1 | Full profiling (type profile, branch profile, null/range checks) |
| 4 | C2 | Aggressive optimization; uses the tier-3 profile |

A method usually walks `0 → 3 → 4`: interpreted while cold, compiled by C1 with profiling once warm, then recompiled by C2 once hot. C1 produces code in milliseconds; C2 may take seconds for a complex method but produces near-handwritten-assembly quality.

### 1.3 Bytecode → IR → Machine Code

C2's pipeline:

```
Bytecode
  └─► Sea-of-nodes IR (graph: data + control)
        └─► Inlining + escape analysis + GVN + loop opts
              └─► Register allocation (graph coloring)
                    └─► Instruction selection (per-arch matcher)
                          └─► Machine code in code cache
```

The **code cache** is a fixed-size memory region (default ~240 MiB on 64-bit) where JIT-compiled methods live. When it fills, compilation stops and a warning is logged. Tune with `-XX:ReservedCodeCacheSize=...`.

### 1.4 Metaspace (Replaces PermGen, Java 8+)

Pre-Java 8, classes lived in **PermGen** — a fixed region inside the heap. Heavy class-loading apps (servlet containers reloading WARs) hit `OutOfMemoryError: PermGen space`.

Java 8 replaced PermGen with **Metaspace**, allocated in **native memory** (off-heap):

```bash
-XX:MetaspaceSize=128m       # Initial (will grow)
-XX:MaxMetaspaceSize=512m    # Cap (default unlimited)
```

Metaspace stores:
- Class metadata (`Klass*`)
- Method bytecode
- Constant pools
- Annotations
- Method counters

Class data is allocated per-classloader, freed when the classloader becomes unreachable.

### 1.5 Class Data Sharing (CDS) and AppCDS

**CDS** dumps a curated set of system classes into a memory-mapped archive (`classes.jsa`) at JDK build time, so multiple JVMs share the same parsed metadata pages. **AppCDS** (Java 10+, default-on in 13+) extends this to application classes.

```bash
# Java 13+: dynamic CDS archive on shutdown
java -XX:ArchiveClassesAtExit=app.jsa -cp app.jar com.example.Main
java -XX:SharedArchiveFile=app.jsa  -cp app.jar com.example.Main
```

Startup wins of 20–50% are common for cold starts of medium apps. With `-Xshare:on`, the JVM mmaps the archive and *fails* if it can't (useful for tests). `-Xshare:auto` (default) falls back gracefully. `-Xshare:dump` regenerates the system archive.

### 1.6 AOT Compilation

Two distinct stories:

- **`jaotc`** (Java 9–16, removed in 17): an experimental AOT compiler using Graal as a backend. Deprecated in favor of GraalVM.
- **GraalVM Native Image**: closed-world AOT compilation to a single native binary. No JIT at runtime, no class loading after image build, no reflection without configuration. Trade-off: blazing startup (single-digit ms) and tiny memory (sometimes 10× less RSS), but loses JIT peak throughput and dynamic features.

```bash
# GraalVM
native-image -cp app.jar -H:Class=com.example.Main -H:Name=app
./app   # single static binary
```

### 1.7 Major HotSpot Tunables Cheat Sheet

```bash
# Memory
-Xms4g -Xmx4g                          # initial = max (avoid resize)
-Xss512k                               # per-thread stack
-XX:MaxDirectMemorySize=1g             # direct ByteBuffer cap

# JIT
-XX:+TieredCompilation                 # default true
-XX:CompileThreshold=10000             # interpreter→C1 trigger (untiered)
-XX:ReservedCodeCacheSize=240m

# GC (pick exactly one)
-XX:+UseSerialGC
-XX:+UseParallelGC
-XX:+UseG1GC                           # default since Java 9
-XX:+UseZGC -XX:+ZGenerational         # Java 21+
-XX:+UseShenandoahGC                   # OpenJDK build
-XX:+UseEpsilonGC                      # no-op, testing only

# Logging
-Xlog:gc*:file=gc.log:time,level,tags
-XX:+PrintCompilation
-XX:+UnlockDiagnosticVMOptions -XX:+PrintInlining
```

---

## 2. Class Loading

### 2.1 The Five Phases

The JVM Specification (JVMS §5) breaks loading into:

| Phase | What happens |
|:------|:-------------|
| Loading | Read `.class` bytes, build internal `Klass*` |
| Linking — Verification | Prove bytecode is well-typed and stack-safe |
| Linking — Preparation | Allocate static fields, initialize to default values |
| Linking — Resolution | Resolve symbolic references (`#NN` constant-pool entries) to direct pointers |
| Initialization | Run `<clinit>` (class initializer) — static initializers + static field assignments |

**Loading is lazy.** A class is loaded the first time it is *actively used*: a `new` creates an instance, a static field is read/written, a static method is invoked, `Class.forName()` is called, or a subclass triggers parent loading.

**Initialization is even lazier.** A class can be loaded and verified without `<clinit>` running — `<clinit>` runs only on active use. `Class.forName(name, false, loader)` loads and links without initializing.

### 2.2 The Parent-Delegation Model

```
                    +-------------------+
                    | Bootstrap (null)  |   loads java.base, java.lang, ...
                    +---------+---------+
                              │
                    +---------v---------+
                    |   Platform (was   |   loads java.sql, java.xml, ...
                    |   "Extension"     |   (since Java 9 modules)
                    |   pre-9)          |
                    +---------+---------+
                              │
                    +---------v---------+
                    |   Application     |   loads classpath / module path
                    |   (System)        |
                    +---------+---------+
                              │
                    +---------v---------+
                    | Custom (URL/Web/  |   one per WAR / OSGi bundle / etc
                    | OSGi/SecMgr/...)  |
                    +-------------------+
```

When a classloader is asked to load a class, it **first delegates to its parent**, only loading the class itself if the parent fails. This guarantees that, e.g., `java.lang.String` always comes from the bootstrap loader — a malicious user-supplied `java/lang/String.class` on the classpath cannot shadow it.

### 2.3 ClassLoader API

```java
public class IsolatingLoader extends ClassLoader {
    public IsolatingLoader(ClassLoader parent) { super(parent); }

    @Override
    protected Class<?> findClass(String name) throws ClassNotFoundException {
        byte[] bytes = readClassBytes(name);
        return defineClass(name, bytes, 0, bytes.length);
    }
}
```

Override `findClass`, not `loadClass`, unless you need to break the parent-delegation contract (e.g., a servlet container loading webapp classes *before* delegating to the app classloader so the WAR's bundled `slf4j` wins).

### 2.4 The Classloader Leak

A `ClassLoader` is unreachable — and thus reclaimable — only if **none** of its loaded classes are reachable from outside. Common leaks:

- A static field in a JDK class (`Thread.contextClassLoader`, `ThreadLocal`, `LogManager`, `JdbcDriverManager`) holds a webapp class.
- A custom thread spawned by the webapp keeps running after WAR undeploy.
- A timer / scheduler with a webapp `Runnable` lives in a JDK static map.
- An `ApplicationListener` registered in a singleton.

Symptom: redeploy the WAR a few times, watch metaspace creep upward, eventually `OutOfMemoryError: Metaspace`. Tools: `jcmd <pid> GC.class_histogram`, Eclipse Memory Analyzer with the "Leak Suspects" report.

### 2.5 The `<clinit>` Static Initializer

The compiler synthesizes a special method `<clinit>` for each class containing:
- All static-field assignments (in source order)
- All `static { ... }` blocks (in source order)

It runs **once**, **under a lock** held on the class's `Class` object. This is why a circular `<clinit>` between two classes can deadlock — and why singletons via static init are thread-safe (the "initialization-on-demand holder" idiom).

```java
public class Holder {
    private Holder() {}
    private static class Inner {                 // not loaded until referenced
        static final Holder INSTANCE = new Holder();
    }
    public static Holder get() { return Inner.INSTANCE; } // triggers Inner <clinit>
}
```

---

## 3. Bytecode Primer

### 3.1 Reading `javap -c`

```java
public class Greet {
    public static int add(int a, int b) { return a + b; }
    public static void main(String[] args) {
        System.out.println("hello");
        int s = add(2, 3);
        System.out.println(s);
    }
}
```

Disassembled with `javap -c -p Greet`:

```asm
public static int add(int, int);
  Code:
    0: iload_0          ; push local 0 (a)
    1: iload_1          ; push local 1 (b)
    2: iadd             ; pop two, push sum
    3: ireturn          ; pop, return int

public static void main(java.lang.String[]);
  Code:
    0: getstatic     #2  ; Field java/lang/System.out:Ljava/io/PrintStream;
    3: ldc           #3  ; String "hello"
    5: invokevirtual #4  ; PrintStream.println:(Ljava/lang/String;)V
    8: iconst_2
    9: iconst_3
   10: invokestatic  #5  ; Greet.add:(II)I
   13: istore_1
   14: getstatic     #2
   17: iload_1
   18: invokevirtual #6  ; PrintStream.println:(I)V
   21: return
```

### 3.2 The Operand Stack and Locals

The JVM is a **stack machine**. Every method has:

- A fixed-size **operand stack** (max depth declared in the `.class`).
- A fixed-size **local variable array** (slots; `long`/`double` take two).

Opcode `iload_1` pushes local slot 1 onto the stack. `iadd` pops two and pushes one. `istore_2` pops one and writes it to local slot 2.

This is intentionally simpler than register-machine bytecode (Dalvik, Lua VM, Wasm pre-spec) — it makes the verifier's job tractable.

### 3.3 Common Opcode Families

| Family | Opcodes | Purpose |
|:-------|:--------|:--------|
| Constants | `iconst_<i>`, `lconst_<l>`, `fconst_<f>`, `bipush`, `sipush`, `ldc` | Push constants |
| Loads | `iload`, `lload`, `aload`, `aload_0` ... `aload_3` | Local → stack |
| Stores | `istore`, `lstore`, `astore`, `astore_<n>` | Stack → local |
| Arithmetic | `iadd`, `isub`, `imul`, `idiv`, `irem`, `ineg`, `iinc`, ... | Math (per type) |
| Logical | `iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr` | Bitwise |
| Object | `new`, `dup`, `getfield`, `putfield`, `getstatic`, `putstatic` | Allocation + field access |
| Method | `invokevirtual`, `invokestatic`, `invokespecial`, `invokeinterface`, `invokedynamic` | Calls |
| Control | `goto`, `ifeq`, `ifne`, `if_icmpge`, `tableswitch`, `lookupswitch`, `return`, `areturn`, `ireturn` | Flow |
| Conversions | `i2l`, `l2i`, `f2d`, `d2i`, ... | Type widening / narrowing |

`aload_0` is *the* opcode you see most: it pushes `this` for any non-static method call.

### 3.4 The Constant Pool

Every `.class` carries a **constant pool** indexed from `#1`. References in code are symbolic (`#NN`) and resolved at link time. You'll see in `javap -v`:

```
Constant pool:
   #1 = Methodref     #8.#22       // java/lang/Object."<init>":()V
   #2 = Fieldref      #23.#24      // java/lang/System.out:Ljava/io/PrintStream;
   #3 = String        #25          // hello
   #4 = Methodref     #26.#27      // java/io/PrintStream.println:(Ljava/lang/String;)V
   ...
```

This indirection lets the verifier and resolver do their work without rewriting bytecode in place.

### 3.5 Walking a Method End-to-End

```java
boolean hasNonNull(String[] xs) {
    for (String s : xs) if (s != null) return true;
    return false;
}
```

```asm
 0: aload_1            ; xs
 1: astore_2           ; arr := xs
 2: aload_2
 3: arraylength
 4: istore_3           ; len := arr.length
 5: iconst_0
 6: istore        4    ; i := 0
 8: iload         4
10: iload_3
11: if_icmpge     33   ; if i >= len goto 33
14: aload_2
15: iload         4
17: aaload             ; s := arr[i]
18: astore        5
20: aload         5
22: ifnull        27   ; if s == null skip return
25: iconst_1
26: ireturn
27: iinc          4, 1 ; i++
30: goto          8
33: iconst_0
34: ireturn
```

---

## 4. Method Dispatch

### 4.1 The Five `invoke*` Opcodes

| Opcode | Receiver? | Resolution | Used for |
|:-------|:----------|:-----------|:---------|
| `invokestatic` | none | Resolved at link, no receiver | `static` methods |
| `invokespecial` | yes | Resolved at link, no virtual lookup | `<init>`, `private`, `super.foo()` |
| `invokevirtual` | yes | Virtual dispatch via class's vtable | Normal instance methods |
| `invokeinterface` | yes | Itable / hashed dispatch | Interface methods |
| `invokedynamic` | yes/no | Bootstrap method picks call-site target, can change | Lambdas, string concat (Java 9+), pattern matching |

### 4.2 vtables and itables

For a class `C` with method `foo()`, HotSpot builds a **virtual method table (vtable)** indexed by method offset. `invokevirtual` becomes:

```asm
mov   rax, [receiver + class_offset]   ; load Klass*
mov   rax, [rax + vtable_offset]       ; load method entry
call  rax
```

Interfaces are messier: a class can implement many interfaces, so a single linear vtable doesn't work. HotSpot uses **itables** — a table of `(Interface, vtable-slice)` pairs — searched (linearly or via a small hash) at the call site. This is why `invokeinterface` is microscopically slower than `invokevirtual`.

### 4.3 Inline Caches and Megamorphism

A real `invokevirtual` call site rarely sees a different receiver type from one call to the next. HotSpot exploits this with an **inline cache (IC)**:

```
Call site state machine:
  uninitialized
       │  first call
       ▼
  monomorphic (cached: SingleClass → method)
       │  cache miss
       ▼
  bimorphic (cached: ClassA, ClassB)
       │  third type seen
       ▼
  megamorphic (fall back to vtable; profile lost)
```

A monomorphic IC compiles to a class-pointer compare and a direct call — almost free, and inlinable. A megamorphic site is a vtable lookup and a non-inlined call — the type profile is gone, escape analysis can't fire, and dependent optimizations collapse.

This is *why* the advice "don't use a generic `Object handler` field across many implementations" matters: if a single field dispatches to 50 implementations, you ruin every call site that touches it.

### 4.4 `final` / `private` and Static Dispatch

A `final` (or `private`, or `static`) method has only one possible target. The compiler emits `invokespecial` / `invokestatic` — no virtual lookup at all. C2 inlines aggressively at these sites.

The historical advice "make methods `final` for performance" is mostly obsolete: with class-hierarchy analysis (CHA), C2 detects that a `non-final` method has a *single* implementation in the loaded hierarchy and inlines it just as aggressively, *guarded by a deopt point* in case a new subclass is loaded later.

### 4.5 `MethodHandle` and `LambdaMetafactory`

`MethodHandle` is a typed, JIT-friendly function pointer. Unlike `java.lang.reflect.Method.invoke`, calls go through `invokedynamic` and inline. The performance hierarchy:

```
direct call   <  invokedynamic / MethodHandle  <  Method.invoke  <  Constructor.newInstance with setAccessible
   ~1ns                ~1-3ns                       ~30-100ns           ~200ns+
```

Lambdas (`(x) -> x + 1`) are bootstrapped via `LambdaMetafactory`. The *first* call to a lambda triggers ASM-generated synthesis of an inner-class implementing the target functional interface, cached for subsequent calls. After warmup, a lambda call is the same cost as an `invokevirtual` to a final method — far cheaper than the historical anonymous-inner-class approach because the lambda capture site is `invokedynamic`, leaving the class-loader-side work to runtime instead of compile time.

---

## 5. The Java Memory Model (JMM)

### 5.1 Why a JMM?

A multi-threaded program's "obvious" execution (sequentially consistent) is not what hardware provides. CPUs reorder, cache, and forward; compilers reorder and hoist. The JMM (JSR-133, Java 5+) defines what *can* and *cannot* happen, so portable concurrent code is possible.

### 5.2 Happens-Before

The fundamental relation: **`a` happens-before `b`** ($a \xrightarrow{hb} b$) means the effects of `a` are visible to `b`, and `a` is ordered before `b`. The JMM defines happens-before via composition rules:

- **Program order**: within a single thread, `a` before `b` in source ⇒ $a \xrightarrow{hb} b$.
- **Monitor lock**: `unlock(m)` $\xrightarrow{hb}$ subsequent `lock(m)`.
- **Volatile**: write to `volatile v` $\xrightarrow{hb}$ subsequent read of `v`.
- **Thread start**: `Thread.start()` $\xrightarrow{hb}$ first action of the new thread.
- **Thread join**: last action of joined thread $\xrightarrow{hb}$ return from `join()`.
- **Final fields**: end of the constructor $\xrightarrow{hb}$ first read of any `final` field via a reference that didn't escape.
- **Transitivity**: $a \xrightarrow{hb} b$ and $b \xrightarrow{hb} c$ ⇒ $a \xrightarrow{hb} c$.

Formally, $\xrightarrow{hb}$ is the transitive closure:

$$\xrightarrow{hb} = (\xrightarrow{po} \cup \xrightarrow{so})^+$$

### 5.3 Volatile Semantics

```java
volatile int v;
```

A `volatile` write is a **release**: every prior store (program-ordered before) is published before `v` becomes visible. A `volatile` read is an **acquire**: every subsequent load sees state at least as fresh as what the matching writer published.

Hardware fence pattern (x86 is permissive — only `StoreLoad` matters):

```
volatile write: ... ; mov [v], val ; lock orq $0, (rsp)   ; StoreLoad fence
volatile read:  ... ; mov rax, [v] ; (no fence on x86)
```

ARM is weaker: volatile writes get a `dmb ishst` after, volatile reads get a `dmb ishld` before.

### 5.4 Final-Field Semantics

After a constructor returns *without leaking `this`*, all `final` fields are guaranteed visible to any thread that subsequently obtains a reference to the object — even without synchronization. This is what makes `String` (with its `private final char[] value`) safely shareable.

```java
class Pair {
    final int a, b;
    Pair(int a, int b) { this.a = a; this.b = b; }   // do NOT escape `this` here
}
// Any thread reading p.a / p.b after the construction completes sees the final values.
```

Caveat: if the constructor publishes `this` to a static field *before* completing (anti-pattern), other threads can see the partially-constructed object with default-zero `final` fields.

### 5.5 The JSR-133 Cookbook

For compiler authors, JSR-133 distills the JMM into a fence table:

|   1st op \ 2nd op   | Normal Load | Normal Store | Volatile Load (acquire) | Volatile Store (release) |
|:--------------------|:------------|:-------------|:------------------------|:-------------------------|
| Normal Load         | —           | —            | —                       | LoadStore                |
| Normal Store        | —           | —            | —                       | StoreStore               |
| Volatile Load       | LoadLoad    | LoadStore    | LoadLoad                | LoadStore                |
| Volatile Store      | —           | —            | StoreLoad               | StoreStore               |

Each cell tells the compiler which fence to emit between the two ops to preserve JMM semantics.

### 5.6 Sequential Consistency for DRF Programs

The JMM's central guarantee: **data-race-free programs have sequentially consistent semantics**. If every shared variable access is properly synchronized (locks, volatile, `final`-publish), the program behaves as if all ops interleave in some global order. This is the "if you do it right, you don't have to think about it" promise.

### 5.7 JMM vs C++ Memory Model

| Aspect | JMM | C++11 |
|:-------|:----|:------|
| Default access | Plain (racy is undefined) | Plain (racy is UB) |
| Atomic flavors | `volatile`, AtomicX, VarHandle | `memory_order_{relaxed, consume, acquire, release, acq_rel, seq_cst}` |
| Default volatile-like | `volatile` ≈ `seq_cst` atomic | atomic with explicit order |
| `final` publication | Special-cased | No analog |
| Out-of-thin-air values | Forbidden (mostly) | Forbidden (mostly) |

The JMM is *less granular* than C++: `volatile` is sequentially consistent (not relaxed/acquire/release). `VarHandle` (Java 9+) closes that gap.

---

## 6. Garbage Collection — Algorithms and When to Use Each

### 6.1 The Collector Matrix

| Collector | Style | Pause goal | Heap range | Best for | Flag |
|:----------|:------|:-----------|:-----------|:---------|:-----|
| Serial | Single-thread, stop-the-world | Whatever | < 100 MiB | Embedded, tiny services | `-XX:+UseSerialGC` |
| Parallel | Multi-thread STW | Throughput-first | 1–10 GiB | Batch jobs, ETL | `-XX:+UseParallelGC` |
| G1 | Region-based, mostly concurrent | 200ms target | 1 GiB – 64 GiB | Latency-sensitive servers | `-XX:+UseG1GC` (default 9+) |
| ZGC | Concurrent, colored pointers | sub-1ms | Up to 16 TiB | Large-heap, latency-critical | `-XX:+UseZGC -XX:+ZGenerational` |
| Shenandoah | Concurrent compaction | sub-10ms | 1 GiB+ | OpenJDK / RH builds | `-XX:+UseShenandoahGC` |
| Epsilon | No-op | n/a (no GC) | Any | Benchmarks, allocation budgets | `-XX:+UseEpsilonGC` |

### 6.2 Serial

Single GC thread. Mark-sweep-compact in the old generation, copying in the young. Lowest overhead, highest pauses. Used by default on tiny single-CPU containers.

### 6.3 Parallel

Multiple GC threads, but still stop-the-world. Optimizes for **throughput** — minimizes the fraction of time spent in GC, regardless of pause length. Pre-Java 9 default.

### 6.4 G1 (Garbage First)

The default since Java 9. Heap is split into ~2,048 fixed-size **regions** (1–32 MiB depending on heap size). Some regions are eden, some survivor, some old, some humongous (objects > 50% of region size).

Cycle:
1. **Concurrent marking** finds liveness across the heap.
2. **Mixed collections** evacuate young + a handful of "garbage-first" old regions per pause, in order to hit a pause-time goal (`-XX:MaxGCPauseMillis=200`).
3. **Full GC** is a fallback (single-threaded, embarrassing — tune to avoid it).

Predictability comes from a *cost model*: G1 measures region copy/scan times and only schedules as much work per pause as fits the budget.

### 6.5 ZGC (Generational since Java 21)

Region-based, **concurrent everything**. Uses **colored pointers** — extra metadata bits in the upper bits of pointers — and a **load barrier** that intercepts every reference load. Marking, relocation, and remap all run concurrently; STW pauses are only for root scanning (sub-ms even at TB heaps).

Generational ZGC (JEP 439, Java 21): adds young/old separation so common short-lived garbage doesn't trigger expensive full-heap concurrent cycles.

```bash
java -XX:+UseZGC -XX:+ZGenerational -Xmx128g ...
```

### 6.6 Shenandoah

Like ZGC, but invented at Red Hat for OpenJDK. Uses a **Brooks pointer** (or, in newer versions, a **load reference barrier**) for concurrent compaction. Differences vs ZGC: simpler to port to 32-bit / non-amd64, slightly higher mutator overhead, no colored-pointer requirement.

### 6.7 Epsilon

Allocates only — never collects. The JVM dies of OOM as soon as the heap fills. Use cases:
- Performance benchmarks: separate allocator perf from collector perf.
- Short-lived utilities where allocation < heap.
- Determining a true "allocation budget" for your workload.

### 6.8 GC Tuning Starting Points

```bash
# Baseline: G1, log everything to a rotating file
-XX:+UseG1GC
-Xms8g -Xmx8g                        # equal: avoid resize STW
-XX:MaxGCPauseMillis=200             # G1 budget target
-XX:+ParallelRefProcEnabled
-Xlog:gc*,safepoint:file=gc.log:utctime,level,tags:filecount=10,filesize=50M
```

Rule of thumb: heap size ≈ 2 × steady-state live set. If "live after full GC" is 4 GiB, use `-Xmx8g`. Smaller forces frequent collections; larger wastes RAM and lengthens pauses.

For ZGC: just set `-Xmx`, mostly. Pause time is heap-independent. The expensive resource is *CPU*: ZGC concurrent threads consume cores during marking.

### 6.9 Reading GC Logs (`-Xlog:gc*`)

```
[2024-04-25T14:32:01.123+0000][info][gc] GC(42) Pause Young (Normal) (G1 Evacuation Pause)
  1024M->312M(8192M) 18.523ms
```

Decoding: GC #42, young pause, evacuation, before/after/total = 1024 MiB → 312 MiB of an 8 GiB heap, took 18.5 ms. Repeated young pauses with little reduction = high allocation rate. Mixed pauses growing in duration = old generation filling up. Promotion failure or full GC = re-tune.

---

## 7. GC — Generational Hypothesis and Heap Layout

### 7.1 The Hypothesis

Empirically, **most objects die young**. A survival curve looks roughly like:

$$P(\text{survive past age } t) = e^{-\lambda t}, \quad \lambda \approx 1\text{-}3$$

After 3 epochs, > 95% of allocations are dead. This justifies cheap-frequent young collections plus rare-expensive old collections.

### 7.2 The Young Generation

Three sub-regions:

```
+----------------+----------------+----------------+
|      Eden      |  Survivor S0   |  Survivor S1   |
+----------------+----------------+----------------+
       ▲                  ▲                ▲
   allocations       to-space         from-space
                  (this cycle)     (this cycle)
```

- New objects are bump-allocated into Eden.
- A young (minor) collection copies live objects from Eden + active Survivor → other Survivor.
- The "active" Survivor flips each cycle.
- Objects surviving N cycles (`-XX:MaxTenuringThreshold`, default 15) are **promoted** to the old generation.

### 7.3 Bump-Pointer Allocation and TLABs

Inside Eden, allocation is a single atomic add to a pointer:

```c
old = atomic_fetch_add(&eden_top, size);
return old;
```

But every thread doing atomic CAS on a single pointer is a contention disaster. Solution: each thread gets a **Thread-Local Allocation Buffer (TLAB)** — a private slice of Eden. Allocation inside a TLAB is just `top += size; if (top > end) refill()`, no atomics.

TLAB sizing is dynamic; typically tens of KiB. Tunables:

```bash
-XX:+UseTLAB                 # default true
-XX:TLABSize=512k
-XX:+ResizeTLAB              # default true: heuristic resizing
```

### 7.4 The Write Barrier (Card Table)

Old → young pointers are rare but matter: a young GC needs to know which old objects might reference young objects (to add as roots). Scanning the entire old gen per young GC defeats the purpose.

The **card table** divides the old gen into 512-byte cards (one byte per card). Every reference store `obj.field = ref` runs through a write barrier:

```c
*(obj.field) = ref;
card_table[(obj >> 9) - card_table_base] = DIRTY;
```

A young GC scans only the dirty cards in the old gen, then resets them clean.

### 7.5 G1's Remembered Sets

G1 generalizes the card table per region. Each region has a **remembered set (RSet)** that records "which other regions hold pointers into me." Evacuating a region needs only its RSet, not the whole heap. RSets cost ~5–10% of heap space.

### 7.6 Promotion and Tenuring

```bash
-XX:MaxTenuringThreshold=15      # max age before forced promotion
-XX:InitialTenuringThreshold=7
-XX:+PrintTenuringDistribution
```

The tenuring distribution log shows the age spectrum of survivors. Big bumps at low ages = lots of medium-lived objects (caches, request scopes); shifting tenuring threshold can avoid premature promotion.

### 7.7 Premature Promotion = "Promotion Failure"

If old gen has no room for a young promotion, G1/Parallel falls back to a full GC. That's an emergency: the heap is too small, allocation rate is too high, or `-XX:G1HeapWastePercent` / `-XX:G1MixedGCLiveThresholdPercent` are misconfigured.

---

## 8. JIT Compilation

### 8.1 Tiered Compilation Pipeline

| Tier | Description | Profile? |
|:----:|:------------|:--------:|
| 0 | Interpreter (template) | Counts |
| 1 | C1 simple, no profiling | No |
| 2 | C1 + invocation/back-edge counters | Limited |
| 3 | C1 + full profiling | Full |
| 4 | C2 (consumes tier-3 profile) | No (uses prior) |

Default thresholds (tunable; see `-XX:Tier3CompileThreshold` etc.):

```
T0 → T3:  ~2,000 invocations OR backedge-counter overflow
T3 → T4:  ~10,000 invocations of a tier-3-compiled method
```

### 8.2 OSR — On-Stack Replacement

Imagine a `main()` that runs a single hot loop without ever returning. The method is *running*, so you can't recompile it the normal way (you can't redirect the entry point — there's no fresh entry point coming).

OSR compiles a **loop entry point** instead of a method entry. When the back-edge counter overflows, the JVM patches the running stack frame: saves locals + operand stack, transfers to a JIT-compiled version of the loop, resumes. The original interpreter frame is discarded.

`-XX:+PrintCompilation` shows OSR with a `% ` flag:

```
  342  234 %     4   com.example.Foo::compute @ 23 (180 bytes)
```

### 8.3 Deoptimization

C2's optimizations are speculative. When an assumption breaks, the compiled code **deoptimizes** back to the interpreter:

- **Uncommon trap**: a branch the profile said was cold actually fires.
- **Class-hierarchy invalidation**: a new class loaded that breaks a CHA-based devirtualization.
- **Predicate failure**: a null-check, bounds-check, or type-check we elided actually wasn't safe.
- **Lock biasing failure**: a lock we biased to a thread is contended.

Deopt isn't catastrophic — the method falls back to the interpreter, then re-tiers. But repeated deopts thrash. Look for `-XX:+PrintCompilation` showing `made not entrant` events on the same method.

### 8.4 Inlining

Inlining is C2's most impactful optimization — without it, almost no other optimization fires (escape analysis can't see across calls; constants don't propagate; loops can't fuse).

Defaults:

```
-XX:MaxInlineSize=35             # max bytecode of a non-hot callee to inline
-XX:FreqInlineSize=325           # max bytecode of a hot callee to inline
-XX:InlineSmallCode=2500         # callee already-compiled size limit
-XX:MaxInlineLevel=15            # call-chain depth cap
```

A callee bigger than these thresholds is *not* inlined, even if hot. Refactoring a 400-byte hot method into two ~200-byte halves can flip both into "inlinable" and yield big wins.

### 8.5 The Compilation Log

```bash
java -XX:+UnlockDiagnosticVMOptions \
     -XX:+LogCompilation \
     -XX:+PrintInlining \
     -XX:+PrintCompilation \
     -XX:LogFile=jit.log MyApp
```

`jit.log` is XML; visualize with **JITWatch** (https://github.com/AdoptOpenJDK/jitwatch).

`PrintCompilation` line decoding:

```
123  234   ! 3       com.example.Foo::bar (bytes)  <details>
^^^  ^^^   ^ ^       ^^^^^^^^^^^^^^^^^^^^
 │    │    │ │              method
 │    │    │ tier (3 = C1+profile, 4 = C2)
 │    │    has-exception-handler flag
 │    compile id
 timestamp ms since start
```

---

## 9. JIT Optimizations

### 9.1 Escape Analysis → Scalar Replacement → Stack Allocation

If the JIT proves an object **doesn't escape its allocating method**, it can decompose the object into individual scalars (registers / stack slots) — **scalar replacement** — and skip heap allocation entirely.

```java
// Looks like an allocation. Often, it isn't.
int distance(Point a, Point b) {
    Point delta = new Point(a.x - b.x, a.y - b.y);
    return (int) Math.hypot(delta.x, delta.y);
}
```

If `delta` doesn't escape, C2 turns it into two locals (`dx`, `dy`), zero heap allocation, zero GC pressure. This is why "don't use objects, use primitives" is largely outdated advice — the JIT does that for you, *as long as the object doesn't escape*.

What kills it: storing the object into a static field, returning it, passing it to a method the JIT can't inline, putting it in a megamorphic call site.

### 9.2 Lock Elision and Biased Locking

If a synchronized block protects an object that doesn't escape, C2 elides the lock entirely. For shared objects, **biased locking** historically optimized the uncontended case (no CAS, just a pointer compare).

```bash
# Java 15+: biased locking deprecated, off by default in 18+
-XX:-UseBiasedLocking
```

Modern HotSpot replaces it with **lightweight locking** + adaptive spinning — performance is similar in the common case, simpler in the contended case.

### 9.3 Devirtualization via Type Profile

If the tier-3 type profile shows a virtual call site sees one or two receiver types ≥ 95% of the time, C2 emits an inlined fast path:

```c
if (receiver.klass == ProfileSeenClass) {
    // inlined body
} else {
    // fall back to vtable lookup + uncommon trap
}
```

A megamorphic site (3+ types) loses this entirely.

### 9.4 Bounds-Check Elimination

```java
for (int i = 0; i < arr.length; i++) sum += arr[i];
```

Naively, every `arr[i]` checks `i >= 0 && i < arr.length`. The JIT proves `i ∈ [0, arr.length)` from the loop bounds and elides the check. Manual bounds-check-friendly idioms:

- Iterate over `arr.length` (not a function-call-derived bound).
- Avoid `i % arr.length` in the index — confuses the analysis.
- Avoid intervening method calls that could (in theory) modify `arr.length` — only matters for multi-dim arrays.

### 9.5 Loop Optimizations

- **Unrolling**: replicate loop body N times to amortize loop overhead.
- **Range check elimination**: as above.
- **Loop-invariant code motion (LICM)**: hoist computations whose result doesn't depend on the iteration variable.
- **Loop peeling**: extract first/last iterations to handle edge cases without per-iteration branches.
- **Vectorization (superword)**: detect parallel loops and emit SSE/AVX SIMD. Less effective than auto-vectorization in C/Rust; for serious vectorization, use the **Vector API** (incubator/preview) which exposes explicit SIMD intrinsics.

### 9.6 Prefetch

C2 inserts software prefetch instructions ahead of large array sweeps and during array copy, masking memory latency. Tunable:

```bash
-XX:AllocatePrefetchStyle=N     # 0..3
-XX:AllocatePrefetchLines=N
```

### 9.7 The Deoptimization-on-Class-Load Event

```
java -XX:+UnlockDiagnosticVMOptions -XX:+TraceDeoptimization ...
```

Loading a new class can invalidate CHA-based devirtualization. Example: a hot loop calls `service.execute()` where `Service` had only one impl. C2 inlines. Later, a plugin loads a new `Service` subclass — every compiled method that relied on the assumption is *immediately* deoptimized (`made not entrant`). Subsequent invocations re-tier with the wider profile.

---

## 10. String and the String Pool

### 10.1 Immutability and the Hash Cache

`java.lang.String` is final and immutable. Two slots that matter:

```java
public final class String {
    @Stable private final byte[] value;        // since 9; was char[]
    private final byte coder;                   // 0 = LATIN1, 1 = UTF16
    private int hash;                           // cached, 0 means "not yet computed"
    private boolean hashIsZero;                 // tells "0 means real zero, not unset"
}
```

`hashCode()` is lazily computed on first call and cached. The `+1` over a naive cache is the `hashIsZero` boolean to disambiguate "zero hash because string is empty/`""`" from "hash uncomputed."

### 10.2 Compact Strings (Java 9+)

Pre-9, `String` was always `char[]` (UTF-16). Most strings in real programs are pure ASCII — wasting half the bytes. Compact Strings (JEP 254) add a `coder` byte:

- `coder == 0` (LATIN1): `value` is one byte per char, ASCII-compatible.
- `coder == 1` (UTF16): `value` is two bytes per char.

Memory wins of 30–60% are routine on text-heavy workloads. Tunable:

```bash
-XX:-CompactStrings   # turn off (rare, e.g., bug-compat)
```

### 10.3 The String Pool (`StringTable`)

String literals (`"foo"`) and explicit `intern()` calls live in a JVM-internal hash table — the **`StringTable`**, a native (non-Java) hashtable in C++. Two lookups of the same literal return the same `String` instance:

```java
String a = "hi";
String b = "hi";
assert a == b;   // same object — both interned at class load
```

Tuning the table size (helps if you intern many strings):

```bash
-XX:StringTableSize=1000003   # prime number, default 60013
```

### 10.4 The `substring` Trap (Pre-7u6)

Until Java 7u6, `String.substring(i, j)` returned a *view* sharing the parent's `char[]`. This made substringing $O(1)$ — and produced lurking memory leaks: a huge file slurped into a string, then a tiny substring kept, would retain the entire file's bytes.

7u6 changed `substring` to copy. It's now $O(j - i)$, but no GC mystery. Modern code: don't think about it.

### 10.5 String Concatenation (`invokedynamic` since Java 9)

Pre-9: `"a" + b + c` compiled to `new StringBuilder().append("a").append(b).append(c).toString()`. Allocations scale with chain length.

Java 9+ (JEP 280): the compiler emits `invokedynamic` with a bootstrap method `StringConcatFactory.makeConcatWithConstants`. At runtime, the JVM picks an implementation strategy — often a single direct memory copy with no intermediate allocations. Up to 5× faster on small concatenations.

---

## 11. Reflection vs MethodHandles vs LambdaMetafactory

### 11.1 The Cost Hierarchy

| Mechanism | First call | Steady state | Notes |
|:----------|:-----------|:-------------|:------|
| Direct method call | Inlined / vtable | ~1 ns | The baseline |
| `invokedynamic` (lambda) | Generates impl | ~1–3 ns | Effectively a final-method call after warmup |
| `MethodHandle.invokeExact` | Type check | ~2–5 ns | JIT-friendly, inline-able |
| `Method.invoke` | Reflection bridge | ~30–100 ns | Native call, access check, boxing of args |
| `Constructor.newInstance` | Reflection bridge | ~200 ns + alloc | Same as above plus allocation |
| `setAccessible(true)` | One-time | (negligible after) | Bypasses access checks |

### 11.2 Why Reflection Is Slow

`Method.invoke` boxes primitives, walks an `Object[]` of arguments, performs an access check on every call (mitigated by `setAccessible(true)`), and dispatches via a generated bytecode "accessor" or a native bridge. Each call site is opaque to the JIT — no inlining, no escape analysis, no constant propagation.

### 11.3 MethodHandles

```java
import java.lang.invoke.*;

MethodHandles.Lookup L = MethodHandles.lookup();
MethodHandle mh = L.findVirtual(
    String.class, "length",
    MethodType.methodType(int.class)
);
int n = (int) mh.invokeExact("hello");   // 5
```

`MethodHandle` is a typed function reference. Critically, calls are **`invokedynamic`-shaped** under the hood — the JIT can inline the entire chain when the handle is reachable as a constant.

### 11.4 LambdaMetafactory and Why Lambdas Beat Anon Classes

Pre-Java 8, a "function" was an anonymous inner class:

```java
Runnable r = new Runnable() {
    @Override public void run() { ... }
};
// At compile: synthesizes Outer$1.class, on every load it gets verified, linked, etc.
```

Java 8+ compiles a lambda to an `invokedynamic` site whose bootstrap is `LambdaMetafactory.metafactory`. The first call:
1. ASM-generates an inner class implementing the target interface.
2. Caches the result.
3. Returns a `CallSite` whose target is a `MethodHandle`.

Subsequent calls hit the cached call site; the JIT inlines through it. End result: lambdas are **cheaper than anonymous classes** at load time (no synthetic class until first use) and **the same** at steady state.

---

## 12. Modules (Project Jigsaw / JPMS)

### 12.1 Why Modules?

Java 9 (JEP 261) introduced the Java Platform Module System to:

- Strongly encapsulate internal JDK APIs (`sun.misc.Unsafe`, `com.sun.*`).
- Make missing dependencies a build/start error, not a runtime `ClassNotFoundException`.
- Enable `jlink` to build minimal custom runtimes.
- Allow layered class loaders (multiple module versions in one JVM).

### 12.2 `module-info.java`

```java
module com.acme.payments {
    requires java.sql;
    requires com.acme.config;
    requires transitive com.acme.api;     // re-exported

    exports com.acme.payments.api;
    exports com.acme.payments.internal to com.acme.tests;

    opens com.acme.payments.model;        // deep reflection allowed (e.g., for ORM)

    provides com.acme.spi.Provider with com.acme.payments.PaymentProvider;
    uses com.acme.spi.Logger;
}
```

### 12.3 Module Path vs Classpath

Two disjoint worlds:

- **Classpath**: the legacy. Everything is in one big "unnamed module." Reflection wide open. Backwards compatible.
- **Module path**: explicit graph. `--module-path` (or `-p`). Encapsulation enforced.

You can mix: classpath JARs become "automatic modules" (name derived from filename, all packages exported). This is the migration path.

### 12.4 The `--add-opens` / `--add-exports` / `--add-reads` Escape Hatches

When a library needs deep reflection into a JDK module that doesn't `opens` to it (e.g., older Hibernate, Lombok, ByteBuddy on JDK 17):

```bash
java --add-opens java.base/java.lang=ALL-UNNAMED \
     --add-exports java.base/sun.nio.ch=ALL-UNNAMED \
     --add-reads java.base=ALL-UNNAMED \
     -jar app.jar
```

Document these in `META-INF/MANIFEST.MF`'s `Add-Opens` entry or a launcher script — they're load-bearing.

### 12.5 jlink and Custom Runtimes

```bash
jlink \
  --module-path "$JAVA_HOME/jmods:mods" \
  --add-modules com.acme.payments \
  --launcher pay=com.acme.payments/com.acme.payments.Main \
  --output dist/runtime
```

Produces a `dist/runtime/` containing exactly the JDK modules your app needs (often 30–50 MiB instead of 200 MiB+) plus a `bin/pay` launcher. Combined with AppCDS, this delivers small, fast cold-start packages — the JVM's answer to single-binary deploys.

---

## 13. Virtual Threads (Project Loom, stable in Java 21+)

### 13.1 What They Are

**Virtual threads** are user-mode threads scheduled by the JVM onto a small pool of OS-backed **carrier threads** (a `ForkJoinPool`). A virtual thread:

- Has a tiny initial stack (~1 KiB), grown on demand.
- Costs ~200 bytes when parked.
- Is mounted on a carrier thread when running.
- *Unmounts* when it blocks (I/O, `Lock.lock()`, `LockSupport.park()`), releasing the carrier for other virtuals.

You can have a million virtual threads where you'd have a few thousand platform threads.

### 13.2 Continuations Under the Hood

A virtual thread is a `Continuation` (still an internal API, exposed via Loom plumbing). When it parks, the JVM **freezes** the continuation: walks the stack, copies live frames into an array on the heap, and unmounts. Unparking **thaws** the continuation: copies frames back onto a carrier's native stack and resumes.

This is exactly the Goroutine model — but built on top of arbitrary Java code, no special syntax required.

### 13.3 Creating Virtual Threads

```java
Thread vt = Thread.ofVirtual().start(() -> {
    System.out.println("hi from " + Thread.currentThread());
});
vt.join();

// Executor with one virtual thread per task — no pooling needed
try (var es = Executors.newVirtualThreadPerTaskExecutor()) {
    for (int i = 0; i < 1_000_000; i++) {
        es.submit(() -> { httpCall(); });
    }
}
```

### 13.4 Pinning

A virtual thread is **pinned** to its carrier (cannot unmount) when:
- Inside a `synchronized` block (Java 21–23; relaxed in JEP 491 / Java 24+).
- Inside a native frame (JNI / Panama downcall).
- Doing certain file I/O on filesystems without async support (e.g., synchronous `FileInputStream.read()` on Linux without `io_uring`).

Pinning means a parked virtual thread holds its carrier, defeating the scaling story. Find pinning with:

```bash
-Djdk.tracePinnedThreads=full
```

Migration: prefer `ReentrantLock` over `synchronized` in libraries that virtual threads will use heavily.

### 13.5 Structured Concurrency

A preview API (JEP 462 / 480) that scopes a parent task to a set of child tasks: the parent doesn't return until *all* children terminate, and a child failure can cancel siblings.

```java
try (var scope = new StructuredTaskScope.ShutdownOnFailure()) {
    Subtask<String> a = scope.fork(() -> fetchA());
    Subtask<String> b = scope.fork(() -> fetchB());
    scope.join().throwIfFailed();
    return a.get() + b.get();
}
```

Goal: make concurrent code as reasonable as sequential — exceptions, cancellation, and lifetimes scope-bound.

---

## 14. Foreign Function & Memory API (Project Panama, stable in Java 22+)

### 14.1 The Replacement for JNI

JNI is awful: write C, compile per-platform, manage GC interaction manually, fall off cliffs around exceptions and pinning. Panama replaces it with a Java-only API for calling native functions and managing native memory.

### 14.2 Core Types

| Type | Purpose |
|:-----|:--------|
| `Arena` | Lifetime scope for native memory; auto-freed on close |
| `MemorySegment` | A pointer + length + scope (bounds + lifetime checked) |
| `ValueLayout` | Describes a C type (`JAVA_INT`, `JAVA_LONG`, `ADDRESS`) |
| `Linker` | Bridges Java to C ABI |
| `FunctionDescriptor` | Signature for a downcall |
| `SymbolLookup` | Find a symbol in a loaded native library |

### 14.3 Calling `strlen`

```java
import java.lang.foreign.*;
import static java.lang.foreign.ValueLayout.*;

try (Arena arena = Arena.ofConfined()) {
    Linker linker = Linker.nativeLinker();
    SymbolLookup libc = linker.defaultLookup();

    MethodHandle strlen = linker.downcallHandle(
        libc.find("strlen").orElseThrow(),
        FunctionDescriptor.of(JAVA_LONG, ADDRESS));

    MemorySegment hello = arena.allocateUtf8String("hello, world");
    long n = (long) strlen.invokeExact(hello);    // 12
}
```

### 14.4 Why It's Safer

- **Bounds checks**: every `MemorySegment` access is bounds-checked.
- **Lifetimes**: closing the `Arena` invalidates all its segments — subsequent access throws `IllegalStateException`, never SIGSEGV.
- **Confined arenas**: pinned to creating thread; sharing requires `Arena.ofShared()` (with extra cost).

### 14.5 jextract

For a real C library, generating bindings by hand is tedious. `jextract` (separate tool) parses headers and emits ready-to-use Java:

```bash
jextract -t com.acme.libsodium --include-dir /usr/include sodium.h
```

Generates a `com.acme.libsodium` package with `MethodHandle`s for every function and `MemoryLayout`s for every struct.

---

## 15. `sun.misc.Unsafe` and `VarHandle`

### 15.1 What `Unsafe` Was

`sun.misc.Unsafe` exposed JVM internals: low-level memory ops (`getInt`, `putInt`, `allocateMemory`), object field offsets, CAS primitives, parking, fence ops. It was supposed to be JDK-internal but every nontrivial library used it — Netty, Cassandra, Hadoop, Hibernate, Kryo.

```java
// Pre-Java 9 idiom (now discouraged)
Unsafe U = Unsafe.getUnsafe();
long offset = U.objectFieldOffset(Foo.class.getDeclaredField("count"));
U.compareAndSwapInt(foo, offset, expected, updated);
```

### 15.2 Why It's Going Away

- **Type-unsafe**: the JIT can't see the access pattern, no bounds checks, easy to corrupt the heap.
- **Module barriers**: in JPMS, `sun.misc.Unsafe` is in `jdk.unsupported` and produces a warning then (eventually) an error.

### 15.3 `VarHandle` (Java 9+, JEP 193)

```java
import java.lang.invoke.*;

private static final VarHandle COUNT;
static {
    try {
        COUNT = MethodHandles.lookup()
            .findVarHandle(Counter.class, "count", int.class);
    } catch (ReflectiveOperationException e) { throw new ExceptionInInitializerError(e); }
}

class Counter {
    private volatile int count;

    void increment() {
        int prev;
        do { prev = (int) COUNT.getVolatile(this); }
        while (!COUNT.compareAndSet(this, prev, prev + 1));
    }
}
```

Modes (per access):

| Mode | Memory order |
|:-----|:-------------|
| `get`/`set` | Plain (no fence) |
| `getOpaque`/`setOpaque` | Opaque (atomicity, no ordering) |
| `getAcquire`/`setRelease` | Acquire/release (one-way fence) |
| `getVolatile`/`setVolatile` | Sequentially consistent |
| `compareAndSet`, `getAndAdd`, ... | Per-method semantics, default seq-cst |

`VarHandle` gives you the C++11 `memory_order_*` flexibility within the JMM. The JIT recognizes call patterns and emits the optimal fence per architecture.

### 15.4 `Atomic*` and Friends

`AtomicInteger`, `AtomicLong`, `AtomicReference` were the old-school wrappers around `Unsafe.compareAndSwap*`. Since Java 9 they're implemented atop `VarHandle`. `LongAdder` / `LongAccumulator` (Java 8) shard counters across cells (per-CPU stripes) to reduce CAS contention — at the cost of a more expensive `sum()`.

---

## 16. Class Data Sharing — Deep Dive

### 16.1 The `.jsa` Archive

A CDS archive is a memory-mapped file containing pre-loaded, pre-verified, partially-resolved class metadata. Multiple JVMs map the same archive read-only — **shared pages across processes**.

### 16.2 System CDS (Out of the Box)

`$JAVA_HOME/lib/server/classes.jsa` ships pre-built. The JVM mmaps it at startup; system classes are ready instantly.

### 16.3 AppCDS (Java 10+)

Add your application classes to the archive.

```bash
# Java 13+: dynamic dump
java -XX:ArchiveClassesAtExit=app.jsa -jar app.jar
java -XX:SharedArchiveFile=app.jsa -jar app.jar    # uses it

# Java 10–12: static dump (two-step, list-based)
java -XX:DumpLoadedClassList=classes.lst -jar app.jar
java -Xshare:dump -XX:SharedClassListFile=classes.lst -XX:SharedArchiveFile=app.jsa
java -Xshare:on  -XX:SharedArchiveFile=app.jsa -jar app.jar
```

### 16.4 Modes

| Flag | Behavior |
|:-----|:---------|
| `-Xshare:on` | Require archive; fail if it can't be used |
| `-Xshare:auto` | Use archive if possible, fall back silently (default) |
| `-Xshare:off` | Don't use archive |
| `-Xshare:dump` | Build archive (dumping run) |

### 16.5 Wins and Caveats

Cold start improves by 20–50%, RSS drops by 50–150 MiB on shared servers, peak GC behavior is unchanged. Caveat: AppCDS captures classes *as loaded*. If you load classes dynamically (plugins, ServiceLoader-bound modules), they're not in the archive.

---

## 17. Performance Tooling

### 17.1 The JDK Toolbox

| Tool | What it does |
|:-----|:-------------|
| `jcmd` | Generic command channel: GC, threads, flight recorder, heap dump |
| `jstack` | Thread dump |
| `jmap` | Heap histogram, heap dump |
| `jstat` | Live GC / class loading / JIT statistics |
| `jfr` | Java Flight Recorder control |
| `jhsdb` | Live debugger / postmortem (replaces `jhat`/`jdb`) |

### 17.2 `jcmd` Cookbook

```bash
jcmd <pid> help                          # list commands
jcmd <pid> VM.flags                      # JVM flags + values
jcmd <pid> GC.heap_info                  # heap state
jcmd <pid> GC.class_histogram            # class instance counts
jcmd <pid> Thread.print                  # stack dump (= jstack)
jcmd <pid> JFR.start duration=60s filename=rec.jfr
jcmd <pid> JFR.stop name=1
jcmd <pid> GC.heap_dump /tmp/heap.hprof
jcmd <pid> VM.system_properties
jcmd <pid> VM.command_line
```

### 17.3 JFR — Java Flight Recorder

Built-in, low-overhead (< 1%) profiler. Produces `.jfr` files openable in **JDK Mission Control** (JMC).

```bash
java -XX:StartFlightRecording=duration=120s,filename=rec.jfr -jar app.jar
```

Captures: GC events, allocation samples, lock contention, exception throws, file/socket I/O, JIT events, JVM internal events.

### 17.4 async-profiler

When JFR's allocation sampling isn't enough (it's biased toward TLAB-refill points), `async-profiler` uses CPU performance counters and AsyncGetCallTrace to produce flame graphs:

```bash
./profiler.sh -d 60 -f flame.html <pid>     # CPU time
./profiler.sh -d 60 -e alloc -f alloc.html <pid>     # allocation
./profiler.sh -d 60 -e lock -f lock.html <pid>       # lock contention
```

The output flame graph collapses identical stacks into proportional-width columns — visually obvious where time/allocation is spent.

### 17.5 Production Troubleshooting Workflow

When a JVM is misbehaving in prod:

1. **`jcmd <pid> VM.flags`** — what are we actually running?
2. **`jcmd <pid> GC.heap_info`** — heap usage; full GC history?
3. **`jcmd <pid> Thread.print`** — any threads stuck on a lock? On socket read?
4. **`jcmd <pid> JFR.start duration=60s ...`** — capture a snapshot, open in JMC.
5. If memory-shaped: **`jcmd <pid> GC.class_histogram`** to spot suspicious classes; if hot, **`GC.heap_dump`** and offline analysis with Eclipse MAT.
6. If CPU-shaped: **async-profiler** flame graph.
7. Cross-reference deopts in `-XX:+PrintCompilation` log if available.

---

## 18. Common Performance Pitfalls

### 18.1 Autoboxing in Hot Loops

```java
// Bad: each iteration allocates a Long
long sum = 0;
for (Long x : list) sum += x;

// Good: avoid the wrapper
long sum = 0;
for (long x : longArray) sum += x;
```

A `Long` allocates ~24 bytes on the heap; a tight loop allocates millions of them, throws GC pressure, and makes escape analysis fight a battle it usually loses.

### 18.2 Megamorphic Call Sites

A field of type `Handler` shared across 50 implementations becomes megamorphic at every dispatch point. Fixes:
- Specialize callers (one method per concrete handler class) where reasonable.
- Use a `switch` on a `kind` enum and explicit branches (the JIT can profile the branch instead).
- Tame the polymorphism: keep hot paths to ≤ 2 implementations.

### 18.3 Escaping Stack-Allocatable Objects

Returning, storing in a static field, or passing to a method the JIT can't inline kills escape analysis and forces heap allocation.

### 18.4 Excessive Short-Lived Allocation

Even with TLABs, allocations cost. A `parse()` that builds three intermediate `List<String>` per call generates GC pressure that escape analysis can sometimes — but not always — eliminate. Consider:
- `StringBuilder` reuse.
- Pre-sized collections (`new ArrayList<>(expected)`).
- Object pools (rare; usually not worth the complexity unless allocation is *the* bottleneck).

### 18.5 Lock Contention

A `synchronized` block held for ~100 ns under heavy contention can degrade throughput by 100×. Fixes in escalating order:

1. Hold the lock for less time (move work outside the critical section).
2. `ReentrantReadWriteLock` if reads dominate.
3. `StampedLock` with optimistic reads if reads vastly dominate.
4. Lock striping (`ConcurrentHashMap`-style).
5. Lock-free (`Atomic*`, `VarHandle.compareAndSet`).
6. `LongAdder` / `LongAccumulator` for write-heavy counters.

### 18.6 Improper `equals` / `hashCode`

Hot maps that violate the `equals`/`hashCode` contract (e.g., mutating a key after insertion) silently misbehave. A `HashMap` with bad hashCode collides into long bucket lists; since 8, the bucket falls back to a tree at threshold, but tree comparison still requires `Comparable` or pays an `O(n)` walk.

---

## 19. Concurrency Primitives — Internals

### 19.1 `synchronized` and the Object Header

Every Java object has a header (12 or 16 bytes on 64-bit, depending on compressed oops) including a **mark word** that doubles as the lock state:

| Lock state | Mark word encoding |
|:-----------|:-------------------|
| Unlocked | hash + age + state bits |
| Lightweight locked | pointer to lock record on holder's stack |
| Heavyweight locked | pointer to JVM `ObjectMonitor` (off-heap) |
| GC-marked | reserved bits |

Path through `synchronized`:

```
1. CAS the mark word: unlocked → "I own this" (lightweight).
   - Success: enter critical section.
   - Failure: another thread holds it → contention.
2. Inflate: allocate an ObjectMonitor; mark word now points to it.
3. Enter ObjectMonitor: park on its wait queue if necessary.
```

The fast path (step 1) is one CAS — uncontended `synchronized` is nearly free. Contention is expensive (kernel `futex` syscall, scheduler involvement).

### 19.2 `ReentrantLock` and AQS

`ReentrantLock` is not a primitive — it's a Java class atop **`AbstractQueuedSynchronizer`** (AQS):

```
AQS state:
  - int state                        ; lock count, semaphore permits, etc.
  - LinkedList<Node> wait queue      ; FIFO of waiting threads (CLH variant)

Acquire:
  if CAS(state, 0, 1) succeed → owned (fast path).
  else enqueue self, park, on wake retry.

Release:
  state := 0;
  unpark head of queue.
```

AQS is the substrate for `ReentrantLock`, `ReentrantReadWriteLock`, `Semaphore`, `CountDownLatch`, `CyclicBarrier`, `FutureTask`, `Phaser`. It's exposed for custom synchronizers via `AbstractQueuedSynchronizer.tryAcquire/tryRelease`.

### 19.3 `synchronized` vs `ReentrantLock`

| Feature | `synchronized` | `ReentrantLock` |
|:--------|:--------------:|:---------------:|
| Reentrancy | yes | yes |
| Try-lock | no | `tryLock()` |
| Timed acquire | no | `tryLock(time, unit)` |
| Interruptible | no | `lockInterruptibly()` |
| Fairness | no (always unfair) | optional (`new ReentrantLock(true)`) |
| Multiple condition variables | one (`wait`/`notify`) | many (`newCondition()`) |
| JMM | full | full |
| Virtual-thread-friendly | pre-JEP-491: pins | no pinning |

Modern advice: prefer `ReentrantLock` in code that may be touched by virtual threads; use `synchronized` for simple cases where its terseness wins.

### 19.4 `ReentrantReadWriteLock`

Two AQS-based modes: shared (read) and exclusive (write). State is split: high 16 bits = read count, low 16 bits = write count. Many readers, single writer; readers exclude writers and vice versa.

Beware **writer starvation** in unfair mode: a steady stream of readers can lock out writers indefinitely.

### 19.5 `StampedLock` (Java 8)

Three modes: write, read, **optimistic read**. An optimistic read returns a *stamp*; you then read the data, then call `validate(stamp)` to check no writer ran in between.

```java
private final StampedLock sl = new StampedLock();
double x, y;

double distFromOrigin() {
    long stamp = sl.tryOptimisticRead();
    double cx = x, cy = y;
    if (!sl.validate(stamp)) {                         // a writer happened
        stamp = sl.readLock();
        try { cx = x; cy = y; } finally { sl.unlockRead(stamp); }
    }
    return Math.hypot(cx, cy);
}
```

When reads vastly outnumber writes, optimistic reads are ~10× faster than `ReentrantReadWriteLock` because they take no locks at all in the common case.

### 19.6 `LongAdder` vs `AtomicLong`

`AtomicLong.incrementAndGet()` is one CAS on the same memory location across all CPUs — under contention, every CPU's cache line ping-pongs. `LongAdder` shards into a `Cell[]`, one cell per "stripe"; CPUs hash to a cell and CAS *that* cell's count. `sum()` walks all cells.

| Primitive | Increment cost | Sum cost | When |
|:----------|:--------------:|:--------:|:----:|
| `AtomicLong` | 1 CAS, contended | $O(1)$ | Low contention or read-heavy |
| `LongAdder` | 1 CAS, sharded | $O(\text{cells})$ | High write contention |

### 19.7 `ConcurrentHashMap` Internals (Java 8+)

Pre-8: a fixed-size array of segments, each a mini hashtable with its own lock — coarse-grained.

Java 8+: a single `Node[]` table. Per-bucket locks via `synchronized` on the bucket's first node. Bucket types:
- Linked list (collisions ≤ 8).
- Red-black tree (collisions > 8 *and* table size ≥ 64).
- "Forwarding node" during resize.

Resize is **incremental and concurrent**: multiple threads cooperate to migrate buckets from old table to new, while reads and writes continue on whichever copy of the bucket they find first.

### 19.8 `LinkedBlockingQueue` and Producer-Consumer

A standard FIFO bounded queue. Two locks (head and tail), so producers and consumers don't contend on the same monitor when the queue is non-empty. Use as the substrate for `ThreadPoolExecutor`'s task queue.

```java
ExecutorService es = new ThreadPoolExecutor(
    4, 8, 60, SECONDS,
    new LinkedBlockingQueue<>(1024),
    new ThreadPoolExecutor.CallerRunsPolicy());     // back-pressure
```

`CallerRunsPolicy` is the bouncer: when the queue is full, the submitting thread runs the task itself — automatic flow control without dropping work.

---

## Prerequisites

- Comfortable reading bytecode-style assembly (operand stack model).
- Working knowledge of multi-threaded programming (locks, condition variables, atomics).
- Familiarity with the layered runtime model: source → compiler → VM → JIT → machine code.
- Conceptual grasp of garbage collection (mark-sweep, copying, generational).
- Understanding of CPU memory models (cache coherence, store/load fences).
- Fluency with Java 8+ syntax (lambdas, streams, modules, records, sealed classes).
- Comfort with the JDK CLI tools (`javac`, `java`, `javap`, `jar`, `jlink`, `jcmd`).

## Complexity

- Object allocation in TLAB: **$O(1)$** (bump pointer).
- Object allocation when TLAB exhausted: **$O(1)$** amortized (atomic refill).
- Young GC: **$O(\text{live young})$** (copying collector).
- Old GC (G1 mixed): **$O(\text{regions evacuated})$** with work bounded by pause goal.
- Old GC (full): **$O(\text{heap size})$** — avoid in production.
- Bytecode verification: **$O(\text{method size})$** (linear in bytecode + locals).
- Class loading: **$O(\text{methods + fields + constant pool entries})$**.
- Reflection cache lookup: **$O(1)$** with cached `Method`/`Field`; **$O(\text{methods})$** without.
- `HashMap` get/put: **$O(1)$** average, **$O(\log n)$** worst-case (treeified buckets, since 8).
- `ConcurrentHashMap` get: **$O(1)$** average, lock-free; put: **$O(1)$** with bucket lock.
- AQS acquire (uncontended): **$O(1)$** one CAS; contended: **$O(\text{queue depth})$** with parking.
- JIT compilation: **$O(\text{method size}^k)$** for some optimizer-internal $k > 1$ (bounded by per-method limits).
- ZGC pause: **$O(|\text{roots}|)$**, independent of heap size.

## See Also

- [java](java) — practical Java reference (syntax, syntax-level features, common APIs)
- [polyglot](polyglot) — language/runtime comparison
- [c](c) — the language HotSpot is implemented in
- [rust](rust) — different memory and concurrency models, no GC
- [go](go) — different runtime model: goroutines vs virtual threads, tracing GC

## References

- **The Java Language Specification (JLS), Java SE 21 Edition** — Gosling, Joy, Steele, Bracha, Buckley, Smith. Oracle, 2023. https://docs.oracle.com/javase/specs/jls/se21/html/index.html
- **The Java Virtual Machine Specification (JVMS), Java SE 21 Edition** — Lindholm, Yellin, Bracha, Buckley, Smith. Oracle, 2023. https://docs.oracle.com/javase/specs/jvms/se21/html/index.html
- **JSR-133: Java Memory Model and Thread Specification** — JCP, 2004. https://www.jcp.org/en/jsr/detail?id=133
- **JSR-133 Cookbook for Compiler Writers** — Doug Lea. https://gee.cs.oswego.edu/dl/jmm/cookbook.html
- **Java Concurrency in Practice** — Brian Goetz et al., Addison-Wesley, 2006. The reference text on the JMM, threading, and the `java.util.concurrent` library.
- **Effective Java, 3rd Edition** — Joshua Bloch, Addison-Wesley, 2018. Idiomatic Java; many chapters on object lifecycle, equality, and concurrency that interplay with the runtime.
- **Java Performance: The Definitive Guide, 2nd Edition** — Scott Oaks, O'Reilly, 2020. JIT, GC, and JVM tuning at depth.
- **The Garbage Collection Handbook, 2nd Edition** — Jones, Hosking, Moss, CRC Press, 2023. The textbook on GC algorithms.
- **OpenJDK** — the source: https://github.com/openjdk/jdk
- **OpenJDK JEP Index** — every JVM feature has a JEP: https://openjdk.org/jeps/0
- **HotSpot Wiki on OpenJDK** — https://wiki.openjdk.org/display/HotSpot
- **Aleksey Shipilëv's blog** — JMH, JMM, and JIT internals from the source. https://shipilev.net
- **Cliff Click's blog (HP, Azul)** — original C2 architect's commentary. https://www.cliffc.org/blog/
- **Mechanical Sympathy mailing list / Martin Thompson's blog** — lock-free programming, false sharing, NUMA. https://mechanical-sympathy.blogspot.com/
- **The Garbage Collection Tuning Guide for HotSpot** — Oracle. https://docs.oracle.com/en/java/javase/21/gctuning/index.html
- **JEP 425: Virtual Threads (Preview)** — https://openjdk.org/jeps/425; **JEP 444: Virtual Threads (Final)** — https://openjdk.org/jeps/444
- **JEP 439: Generational ZGC** — https://openjdk.org/jeps/439
- **JEP 454: Foreign Function & Memory API** — https://openjdk.org/jeps/454
- **JITWatch** — visualize the JIT compilation log: https://github.com/AdoptOpenJDK/jitwatch
- **async-profiler** — https://github.com/async-profiler/async-profiler
- **Eclipse Memory Analyzer (MAT)** — heap dump analysis: https://www.eclipse.org/mat/
- **JDK Mission Control (JMC)** — JFR analysis: https://www.oracle.com/java/technologies/jdk-mission-control.html
