# The Internals of Lua — VM, GC, Metatables, and LuaJIT Tracing

> *Lua is a tiny register-based virtual machine that fits in 250KB of object code, embeds in everything from World of Warcraft to nginx, and — through LuaJIT — competes with C for raw speed. To understand Lua deeply is to understand a language designed for embedding: a single number type (or two since 5.3), a single data structure (the table), a tiny C API, and a metamechanism (metatables) that turns that single data structure into objects, modules, environments, classes, prototypes, namespaces, and proxies. This deep dive crawls the Proto, the CallInfo, the iABC encoding, the open/closed upvalue distinction, the tri-color incremental collector, the LuaJIT trace recorder, and the FFI — everything the practical sheet glosses over.*

---

## 1. The Lua VM Architecture

Lua's runtime is a **register-based virtual machine** that operates on a per-coroutine value stack. Since Lua 5.0 — when Roberto Ierusalimschy switched from a stack-based VM — every Lua function executes against an array of "registers" that are simply slots in the calling thread's value stack. There is no separate register file; the registers *are* the stack.

### 1.1 The Big Picture

```
                    lua_State (one per coroutine)
                  ┌─────────────────────────────────┐
                  │  stack[]  ── array of TValue    │
                  │  base ───────► R(0) for cur fn  │
                  │  top  ───────► first free slot  │
                  │  ci   ───────► CallInfo chain   │
                  │  l_G  ───────► global_State     │
                  └─────────────────────────────────┘
                                │
                                ▼
                    global_State (one per Lua universe)
                  ┌─────────────────────────────────┐
                  │  strt   ── string intern table  │
                  │  l_registry ── LUA_REGISTRY     │
                  │  mainthread ── primary lua_State│
                  │  GC fields (gray, allgc, ...)   │
                  └─────────────────────────────────┘
```

Every coroutine is a `lua_State` but they all share one `global_State`. This is why coroutines can pass tables around: tables are GC objects in shared memory, and the per-coroutine `lua_State` only owns the call stack and resume context.

### 1.2 The TValue — Tagged Union

Every Lua value is a tagged union called `TValue`:

```c
/* lobject.h — simplified */
typedef struct TValue {
    union Value {
        struct GCObject *gc;   /* collectable: string, table, fn, ... */
        void *p;               /* light userdata */
        int b;                 /* boolean */
        lua_CFunction f;       /* light C function */
        lua_Integer i;         /* integer (5.3+) */
        lua_Number n;          /* float */
    } value_;
    lu_byte tt_;               /* tag: type+variant+collectable bit */
} TValue;
```

The tag byte encodes three things in 8 bits: the basic type (`LUA_TNIL`, `LUA_TBOOLEAN`, `LUA_TNUMBER`, `LUA_TSTRING`, `LUA_TTABLE`, `LUA_TFUNCTION`, `LUA_TUSERDATA`, `LUA_TTHREAD`), a sub-variant (e.g. `LUA_TSHRSTR` vs `LUA_TLNGSTR` for short vs long strings), and a "collectable" bit that tells the GC whether this value owns a heap object.

Since Lua 5.3 there are two number subtypes: `LUA_VNUMINT` and `LUA_VNUMFLT`. The integer subtype was a major change — until 5.2 every numeric was a `double`.

### 1.3 The Proto — Compiled Function Prototype

When the Lua compiler chews a chunk, it produces one `Proto` per function literal in the source. The `Proto` holds the bytecode, the constant table, the upvalue descriptors, the line info, the names of parameters, and pointers to nested `Proto`s for inner functions.

```c
typedef struct Proto {
    CommonHeader;
    lu_byte numparams;       /* number of fixed parameters */
    lu_byte is_vararg;
    lu_byte maxstacksize;    /* max # registers used by this fn */
    int sizeupvalues;
    int sizek;               /* # constants */
    int sizecode;            /* # bytecode instructions */
    int sizelineinfo;
    int sizep;               /* # nested protos */
    int sizelocvars;
    Instruction *code;       /* the bytecode */
    TValue *k;               /* constant table */
    struct Proto **p;        /* nested fn prototypes */
    Upvaldesc *upvalues;     /* upvalue descriptors */
    LocVar *locvars;         /* debug: local var names */
    TString *source;         /* source name */
} Proto;
```

A `Proto` is GC-managed but immutable. Many `Closure` objects can share a single `Proto` — the closure is the marriage of a `Proto` (code) and a set of upvalue references (captured environment).

### 1.4 The Closure — Code + Captured Environment

```c
typedef struct LClosure {     /* Lua closure */
    CommonHeader;
    lu_byte nupvalues;
    GCObject *gclist;
    struct Proto *p;
    UpVal *upvals[1];          /* flexible array */
} LClosure;

typedef struct CClosure {     /* C closure (for lua_pushcclosure) */
    CommonHeader;
    lu_byte nupvalues;
    GCObject *gclist;
    lua_CFunction f;
    TValue upvalue[1];         /* flexible array */
} CClosure;
```

Two flavors. Lua closures wrap a `Proto`. C closures wrap a `lua_CFunction` and store any "upvalues" the C code wants to bind via `lua_pushcclosure(L, fn, n)` — which is how you implement curried C functions.

### 1.5 The CallInfo — Frame Descriptor

When you call a function, Lua doesn't allocate a new frame off the heap. It reuses a `CallInfo` record from a doubly-linked list that grows on demand:

```c
typedef struct CallInfo {
    StkId func;                /* function index in the stack */
    StkId top;                 /* top for this function */
    struct CallInfo *previous, *next;
    union {
        struct {                  /* only for Lua functions */
            const Instruction *savedpc;
            volatile l_signalT trap;
            int nextraargs;       /* for varargs */
        } l;
        struct {                  /* only for C functions */
            lua_KFunction k;       /* continuation (yieldable C calls) */
            ptrdiff_t old_errfunc;
            lua_KContext ctx;
        } c;
    } u;
    ptrdiff_t extra;
    short nresults;
    unsigned short callstatus;
} CallInfo;
```

`previous`/`next` chain frames; `savedpc` is the program counter saved on call; `nresults` is the number of values the caller wants back. The `callstatus` flags include `CIST_LUA` (Lua call), `CIST_HOOKED` (debug hook active), `CIST_TAIL` (this is a tail call), `CIST_YPCALL` (yieldable pcall), and so on.

### 1.6 Why Register-Based?

Register-based VMs reduce the number of bytecode dispatches. Stack VMs need separate `PUSH`, `PUSH`, `ADD`, `STORE` for `x = a + b`. Register VMs do `ADD R0, R1, R2` — one instruction. The trade-off is wider instructions (32 bits in Lua vs 8 bits in CPython) and more complex codegen, but on modern CPUs with branch predictors and superscalar pipelines, fewer dispatches wins.

The tradeoff was studied by Ierusalimschy et al. in "The Implementation of Lua 5.0" (Journal of Universal Computer Science, 2005). They measured 30–50% speedup over the 4.0 stack VM on numeric benchmarks.

---

## 2. The Bytecode Instruction Set

Lua bytecode is 32 bits per instruction. There are a handful of instruction shapes — `iABC`, `iABx`, `iAsBx`, `iAx`, `isJ` — and each opcode picks one.

### 2.1 Encoding Formats

```
iABC    [opcode:7][A:8][k:1][B:8][C:8]
iABx    [opcode:7][A:8][Bx:17]
iAsBx   [opcode:7][A:8][sBx:17]    (sBx is signed via bias)
iAx     [opcode:7][Ax:25]
isJ     [opcode:7][sJ:25]          (signed jump, 5.4)
```

Field widths are not arbitrary: `A` is always 8 bits because most operations target a register and 256 registers per function is enough. `B` and `C` are 8 bits each in iABC, with the high bit (`k`) sometimes selecting "this is a constant index" vs "this is a register index".

### 2.2 Register/Constant Operands — RK(B)

Many instructions accept either a register or a constant. Lua encodes this with the `k` bit (in 5.4) or the high bit of B/C (in 5.3 and earlier):

```c
/* 5.3 style */
#define ISK(x)   ((x) & BITRK)
#define INDEXK(r) ((int)(r) & ~BITRK)
#define RKASK(x)  ((x) | BITRK)
```

So `ADD A B C` with `B = RKASK(2)` means "B is constant K[2]". This is how the compiler avoids loading every numeric literal into a register before using it.

### 2.3 Core Opcode Catalog

The 5.4 opcode table has roughly 80 instructions. The essentials:

| Opcode | Format | Effect |
|:-------|:-------|:-------|
| `MOVE A B` | iABC | `R(A) := R(B)` |
| `LOADI A sBx` | iAsBx | `R(A) := sBx` (5.4 integer literal) |
| `LOADF A sBx` | iAsBx | `R(A) := (lua_Number)sBx` (5.4 float literal) |
| `LOADK A Bx` | iABx | `R(A) := K(Bx)` (load constant) |
| `LOADKX A` | iABC | `R(A) := K(extra arg)` (for >2^17 constants) |
| `LOADBOOL A B C` | iABC | `R(A) := (Bool)B; if (C) pc++` |
| `LOADNIL A B` | iABC | `R(A), R(A+1), ..., R(A+B) := nil` |
| `GETUPVAL A B` | iABC | `R(A) := UpValue[B]` |
| `SETUPVAL A B` | iABC | `UpValue[B] := R(A)` |
| `GETTABUP A B C` | iABC | `R(A) := UpValue[B][RK(C)]` (typically `_ENV[name]`) |
| `SETTABUP A B C` | iABC | `UpValue[A][RK(B)] := RK(C)` |
| `GETTABLE A B C` | iABC | `R(A) := R(B)[RK(C)]` |
| `SETTABLE A B C` | iABC | `R(A)[RK(B)] := RK(C)` |
| `NEWTABLE A B C` | iABC | `R(A) := {} (size hint B array, C hash)` |
| `SELF A B C` | iABC | Method call setup: `R(A+1) := R(B); R(A) := R(B)[RK(C)]` |
| `ADD A B C` | iABC | `R(A) := RK(B) + RK(C)` |
| `SUB/MUL/DIV/MOD/POW/IDIV` | iABC | Arithmetic |
| `BAND/BOR/BXOR/SHL/SHR` | iABC | Bitwise (5.3+) |
| `UNM/BNOT/NOT/LEN` | iABC | Unary |
| `CONCAT A B C` | iABC | `R(A) := R(B).. ... ..R(C)` |
| `JMP A sJ` | isJ | `pc += sJ; if (A) close upvalues` |
| `EQ A B k` | iABC | `if ((R(A) == RK(B)) ~= k) pc++` |
| `LT/LE A B k` | iABC | Less-than / less-or-equal |
| `TEST A k` | iABC | `if not (R(A) <=> k) pc++` |
| `TESTSET A B k` | iABC | `if (R(B) <=> k) R(A) := R(B); else pc++` |
| `CALL A B C` | iABC | Call `R(A)(R(A+1), ..., R(A+B-1))`, expecting C-1 returns |
| `TAILCALL A B C` | iABC | Tail-call optimisation |
| `RETURN A B C` | iABC | Return `R(A), ..., R(A+B-2)` |
| `FORLOOP / FORPREP` | iAsBx | Numeric for-loop |
| `TFORCALL / TFORLOOP` | iABC | Generic for-loop (`for k,v in pairs(t)`) |
| `CLOSURE A Bx` | iABx | `R(A) := closure(KPROTO[Bx])` |
| `VARARG A C` | iABC | `R(A), R(A+1), ..., R(A+C-2) := vararg` |

### 2.4 luac -l — Reading the Bytecode

The reference implementation ships with `luac`, a compiler-front-end for inspecting bytecode:

```bash
echo 'local x = 1 + 2 * 3' > /tmp/foo.lua
luac -l -p /tmp/foo.lua
```

```
main <foo.lua:0,0> (3 instructions at 0x...)
0+ params, 2 slots, 1 upvalue, 1 local, 0 constants, 0 functions
        1       [1]     LOADI           0 7
        2       [1]     RETURN          1 1 1
```

The compiler folded `1 + 2 * 3` to `7` at compile time. Notice it loaded a single `LOADI 0 7` because 5.4 added integer literals direct in `sBx`. In 5.3 it would have been a `LOADK` with constant table entry.

A more illustrative example:

```bash
cat > /tmp/calc.lua <<'EOF'
local function add(a, b)
    return a + b
end
return add(3, 4)
EOF
luac -l -p /tmp/calc.lua
```

```
main <calc.lua:0,0> (5 instructions at 0x...)
0+ params, 3 slots, 1 upvalue, 1 local, 0 constants, 1 function
        1       [1]     CLOSURE         0 0     ; Proto[0]
        2       [4]     MOVE            1 0
        3       [4]     LOADI           2 3
        4       [4]     LOADI           3 4
        5       [4]     CALL            1 3 0
        6       [4]     RETURN          1 0 1

function <calc.lua:1,3> (3 instructions at 0x...)
2 params, 3 slots, 0 upvalues, 2 locals, 0 constants, 0 functions
        1       [2]     ADD             2 0 1
        2       [2]     RETURN          2 2 1
        3       [2]     RETURN          0 1 1
```

You can read the bytecode like assembly: `CLOSURE 0 0` builds a closure from nested `Proto[0]` into `R(0)`. `CALL 1 3 0` calls `R(1)` with two args (`B-1 = 2`) and asks for "all" returns (`C = 0`).

### 2.5 The `OP_CLOSURE` Magic

`CLOSURE` is special because it has to bind upvalues. After the `CLOSURE` instruction, the compiler emits a sequence of pseudo-instructions in the form of `MOVE` or `GETUPVAL` to describe each upvalue. The VM reads these as *upvalue descriptors*, not as instructions to execute. In 5.4 this was cleaned up — upvalues are described in the `Proto.upvalues` array directly.

```c
/* lvm.c - 5.4 */
case OP_CLOSURE: {
    Proto *p = cl->p->p[GETARG_Bx(i)];
    halfProtect(pushclosure(L, p, cl->upvals, base, ra));
    checkGC(L, ra + 1);
    vmbreak;
}

/* pushclosure binds upvals according to p->upvalues[i].instack flag */
```

---

## 3. The Stack and Calling Convention

Lua's C API is **stack-based**: every interaction with the C side pushes or pops values from a virtual stack. This is the inverse of the VM (register-based) but it makes the C API trivially small.

### 3.1 The Virtual Stack

Each `lua_State*` has a `stack` array. The C side only sees positive (1-indexed from bottom) or negative (relative-to-top) indices:

```c
lua_pushinteger(L, 42);    /* push: top++ */
lua_pushstring(L, "hi");
int v = lua_tointeger(L, -2);   /* peek: 42 */
const char *s = lua_tostring(L, -1);  /* peek: "hi" */
lua_pop(L, 2);             /* pop: top -= 2 */
```

### 3.2 Pseudo-Indices

Some indices don't correspond to real stack slots:

- `LUA_REGISTRYINDEX` — the C registry, a hidden table for C code to stash references.
- `LUA_GLOBALSINDEX` — *deprecated since 5.2*. Globals live in `_ENV` now, accessible as the first upvalue of every chunk.
- `lua_upvalueindex(i)` — upvalue `i` of the running C function (for C closures).

### 3.3 The Call Pattern

```c
/* Call lua_function(arg1, arg2) and get one result */
lua_getglobal(L, "process");      /* push function */
lua_pushinteger(L, 10);            /* push arg1 */
lua_pushstring(L, "hello");        /* push arg2 */
if (lua_pcall(L, 2, 1, 0) != LUA_OK) {
    fprintf(stderr, "error: %s\n", lua_tostring(L, -1));
    lua_pop(L, 1);
    return -1;
}
int result = lua_tointeger(L, -1);
lua_pop(L, 1);
```

`lua_pcall(L, nargs, nresults, errfunc)` pops the function and `nargs`, runs it under a protected boundary (errors caught), pushes `nresults` values. `lua_call` is the unprotected version — errors propagate up via `longjmp` and you'd better have set up a pcall earlier or you'll abort.

### 3.4 Stack Discipline

Every C function has a contract: how many values it pushes vs pops. A `lua_CFunction` registered into Lua receives its arguments as the first N stack slots and must return how many values it pushed. The Lua VM does the cleanup.

```c
static int l_double(lua_State *L) {
    int n = luaL_checkinteger(L, 1);  /* stack: [n] */
    lua_pushinteger(L, n * 2);         /* stack: [n, n*2] */
    return 1;                          /* return last 1 value */
}
```

`luaL_checkinteger` does a `lua_tointegerx` plus a type-check that raises a Lua error if the arg isn't a number convertible to integer. The auxiliary library (`lauxlib.h`) is full of these "check or die" helpers — they're how you write idiomatic, defensive C extensions.

---

## 4. Tables — Hybrid Array+Hash

### 4.1 Two Parts in One Object

```c
typedef struct Table {
    CommonHeader;
    lu_byte flags;           /* metamethod presence cache */
    lu_byte lsizenode;       /* log2 of hash size */
    unsigned int alimit;     /* "limit" of array part */
    TValue *array;           /* array part */
    Node *node;              /* hash part */
    Node *lastfree;          /* free-list head */
    struct Table *metatable;
    GCObject *gclist;
} Table;

typedef struct Node {
    TValue i_val;
    Key key;                /* TValuefields + next-link */
} Node;
```

Two heap blocks: `array[]` for integer keys 1..N, and `node[]` for everything else. Both can be empty; both grow.

### 4.2 The Array-Part Limit

When you do `t[i] = v`, Lua first checks: is `i` an integer in `[1, alimit]`? If yes, update `array[i-1]`. If `i` is just past the limit and the array is dense enough, extend. Otherwise fall through to the hash part.

The "dense enough" decision uses the rule: at least 50% of `array[1..2^k]` is non-nil. The resize algorithm in `ltable.c::luaH_resize` walks both parts, counts populated keys, picks the new array size as the largest 2^k where occupancy ≥ 50%, then puts the rest into the hash.

### 4.3 The Hash Part — Brent's Algorithm

The hash uses **open addressing with a twist**: collisions are resolved by chaining *within the same array* via `Node.next` indices. When you insert a colliding key, Lua first tries to evict the existing occupant of the home slot — but only if that occupant is a *foreigner* (its hash doesn't map to this slot). This is Brent's variation of double hashing and it keeps lookup chains short.

```c
TValue *luaH_set(lua_State *L, Table *t, const TValue *key) {
    const TValue *p = luaH_get(t, key);  /* exists? */
    if (!ttisnil(p))
        return cast(TValue *, p);
    return luaH_newkey(L, t, key);  /* allocate new node */
}
```

When `lastfree` runs out the table is rehashed (doubled). Rehashing is amortized O(n).

### 4.4 The "Holes" Gotcha and `#t`

```lua
local t = {1, 2, nil, 4, 5}
print(#t)       -- could be 2 or 5 — implementation-defined!
```

`#t` is the **length operator**, defined as "any border": any `n` where `t[n] ~= nil` and `t[n+1] == nil`. With holes there are multiple borders. The reference implementation uses binary search over the array part:

```c
/* ltable.c - simplified */
lua_Unsigned luaH_getn(Table *t) {
    unsigned int j = t->alimit;
    if (j > 0 && ttisnil(&t->array[j - 1])) {
        /* binary search in array part */
        unsigned int i = 0;
        while (j - i > 1) {
            unsigned int m = (i + j) / 2;
            if (ttisnil(&t->array[m - 1])) j = m;
            else i = m;
        }
        return i;
    }
    /* check hash part for keys > alimit */
    return hash_search(t, j);
}
```

Result: for a hole-free sequence, `#t` is exact and O(log n). For a sequence with holes, `#t` returns *one* of possibly many borders — don't rely on it.

### 4.5 `rawset` vs `settable`

Two ways to set a key:

```lua
t.x = 1                -- triggers __newindex if mt has it
rawset(t, "x", 1)      -- bypasses metatable
```

In the C API, `lua_settable(L, idx)` is metatable-aware (calls `__newindex`). `lua_rawset(L, idx)` is not. When implementing low-level table primitives in C extensions, `lua_rawset` is mandatory to avoid infinite recursion through `__newindex`.

### 4.6 NEWTABLE Size Hints

`NEWTABLE A B C` accepts hints `B` for the array size and `C` for the hash size, both encoded with a special "floating-point byte" (`fb2int`) that compresses sizes into 7 bits. The compiler emits hints based on the table constructor:

```lua
local t = {1, 2, 3, x = 10, y = 20}
-- compiler hints: array=3, hash=2
```

This avoids quadratic rehash during construction.

---

## 5. Metatables and the Method-Resolution Algorithm

### 5.1 The Slot Lookup Algorithm

When you write `t[k]`, the VM does:

```
1. raw_get(t, k)
2. if non-nil → return it
3. if mt = metatable(t), and mt.__index exists:
    a. if __index is a function → tail-call __index(t, k)
    b. if __index is a table → recurse: lookup k in __index
4. else → return nil
```

The "tail-call" in step 3a is real: `t[k]` for a function `__index` becomes a Lua call, including a new `CallInfo` and a possible coroutine yield boundary.

### 5.2 The `__newindex` Mirror

`t[k] = v` is symmetric:

```
1. if raw_get(t, k) is non-nil → raw_set(t, k, v)  [direct overwrite]
2. else if mt.__newindex exists:
    a. function → call __newindex(t, k, v)
    b. table   → recurse: set k in __newindex
3. else → raw_set(t, k, v)
```

Critical: `__newindex` only fires for **new** keys. Updating an existing key bypasses it. This is the foundation of "read-only proxy tables":

```lua
local function readonly(t)
    return setmetatable({}, {
        __index = t,
        __newindex = function(_, k, _)
            error("attempt to modify readonly table at key '"..k.."'", 2)
        end,
        __metatable = "locked",  -- prevents getmetatable() override
    })
end
```

### 5.3 The Full Metamethod Catalog

| Metamethod | Trigger | Notes |
|:-----------|:--------|:------|
| `__index` | `t[k]` if missing | Function or table |
| `__newindex` | `t[k] = v` if missing | Function or table |
| `__call` | `t(args)` | Make tables callable |
| `__tostring` | `tostring(t)` | Used by print |
| `__metatable` | `getmetatable(t)` returns this; `setmetatable` errors | Lock metatable |
| `__name` | Used by error messages | Type name (5.3+) |
| `__eq` | `a == b` (only if same type and not raw-equal) | |
| `__lt` | `a < b` | |
| `__le` | `a <= b` (5.3+: independent; <5.3 fell back to `not __lt`) | |
| `__add` | `a + b` | Integer/float subtype rules apply |
| `__sub` | `a - b` | |
| `__mul` | `a * b` | |
| `__div` | `a / b` (float div) | |
| `__idiv` | `a // b` (floor div, 5.3+) | |
| `__mod` | `a % b` | |
| `__pow` | `a ^ b` | |
| `__unm` | `-a` | |
| `__band` | `a & b` (5.3+) | |
| `__bor` | `a \| b` | |
| `__bxor` | `a ~ b` | |
| `__bnot` | `~a` | |
| `__shl` | `a << b` | |
| `__shr` | `a >> b` | |
| `__concat` | `a .. b` | |
| `__len` | `#a` | |
| `__pairs` | `pairs(t)` (5.2+) | Custom iterator factory |
| `__ipairs` | `ipairs(t)` (5.2 only — removed 5.3) | |
| `__close` | `<close>` variable scope exit (5.4+) | RAII for Lua |
| `__gc` | Finalization | userdata always; tables only if pre-set |
| `__mode` | Weak table mode | "k", "v", "kv" |

### 5.4 The `flags` Optimization

Look at the `Table` struct again — there's a `lu_byte flags` field. This is a **metamethod presence cache**. When the VM goes to call `__add`, instead of walking the metatable hash to look for `"__add"`, it checks bit 0 of `flags`. If set, the metamethod is *absent* (negative cache), so skip the lookup. If clear, it might be present — do the lookup, and on miss, set the bit.

This is why the order of metamethods in the source matters: the first 8 (give or take) get the cache. Also why mutating a metatable's metamethods later is slow — every mutation has to clear the cache on every table that uses this metatable, which is why Lua just clears all the bits whenever any metamethod is set.

### 5.5 The Canonical Class Pattern

```lua
local Animal = {}
Animal.__index = Animal       -- self-referential __index

function Animal.new(name, sound)
    local self = setmetatable({}, Animal)
    self.name = name
    self.sound = sound
    return self
end

function Animal:speak()
    return self.name .. " says " .. self.sound
end

local dog = Animal.new("Rex", "woof")
print(dog:speak())  --> Rex says woof
```

Method-call syntax `dog:speak()` is sugar for `dog.speak(dog)`. The lookup for `dog.speak` finds `Animal.speak` via `__index`. Inheritance composes by chaining `__index`:

```lua
local Dog = setmetatable({}, {__index = Animal})
Dog.__index = Dog

function Dog.new(name)
    local self = Animal.new(name, "woof")
    return setmetatable(self, Dog)
end

function Dog:fetch() return self.name .. " fetches" end
```

The `SELF A B C` opcode optimizes method calls: it does both the table lookup for the method and pushes the receiver as the first argument in one instruction.

---

## 6. Closures and Upvalues

### 6.1 The Lifetime Problem

Consider:

```lua
local function counter()
    local n = 0
    return function() n = n + 1; return n end
end

local c = counter()
print(c(), c(), c())  -- 1 2 3
```

When `counter` returns, its local `n` should be gone. But the closure still references it. Lua's solution: **upvalues**.

### 6.2 Open vs Closed Upvalues

While `counter` is still on the stack, `n` lives in a stack slot. The inner closure's upvalue reference points *directly to that stack slot*. This is called an **open upvalue**.

When `counter` returns, its stack frame is about to be destroyed. The VM scans every closure that has an open upvalue pointing into this frame, and **closes** them: copy the value out of the stack into the upvalue object's own storage, redirect the upvalue's pointer to its internal storage.

```c
typedef struct UpVal {
    CommonHeader;
    union {
        TValue *p;        /* points to stack OR own value */
    } v;
    union {
        struct {           /* when open */
            struct UpVal *next;
            struct UpVal **previous;
        } open;
        TValue value;     /* when closed */
    } u;
} UpVal;
```

The `v.p` pointer is the indirection. While open, `v.p == &thread->stack[i]`. When closed, `v.p == &this->u.value`. The `OP_CLOSE` instruction (or the `JMP` with the close flag) triggers the closing.

### 6.3 Sharing — Multiple Closures, One Upvalue

```lua
local function make_pair()
    local x = 0
    local get = function() return x end
    local set = function(v) x = v end
    return get, set
end

local g, s = make_pair()
s(42); print(g())  -- 42
```

Both `get` and `set` reference the same `UpVal` object. The compiler tracks this: when emitting `CLOSURE` for `set`, if `x` is already shared with `get`, the upvalue descriptor reuses the existing `UpVal`.

Open upvalues are stored in a per-thread linked list (`L->openupval`) sorted by stack depth. Closing closes everything ≥ a target stack level in one walk.

### 6.4 Counter Walked Through Bytecode

```bash
cat > /tmp/ctr.lua <<'EOF'
local function counter()
    local n = 0
    return function()
        n = n + 1
        return n
    end
end
return counter
EOF
luac -l -p /tmp/ctr.lua
```

```
main <ctr.lua:0,7>:
    1   CLOSURE  0 0    ; counter
    2   RETURN   0 2 1

function counter <ctr.lua:1,7>:
    1   LOADI    0 0       ; R0 = 0  (this is local n)
    2   CLOSURE  1 0       ; R1 = closure of inner Proto[0]
    3   RETURN   1 2 1     ; return R1

function inner <ctr.lua:3,6>:
    1   GETUPVAL 0 0       ; R0 = upval[0]  (the n)
    2   ADDI     0 0 1     ; R0 = R0 + 1
    3   SETUPVAL 0 0       ; upval[0] = R0
    4   GETUPVAL 0 0       ; R0 = upval[0]
    5   RETURN   0 2 1     ; return R0
```

When `counter` returns at instruction 3, the VM checks for open upvalues pointing into `counter`'s frame. It finds the inner closure's upvalue 0 pointing at `R(0)` of `counter`. It closes: copies the integer `0` into the upvalue's own storage. Now the inner closure carries `n` with it, even though `counter`'s frame is reaped.

---

## 7. Garbage Collection

### 7.1 The Tri-Color Incremental Algorithm

Lua uses incremental, tri-color, mark-and-sweep collection. Every GC object is in one of three colors:

- **White** — not yet visited; tentatively garbage.
- **Gray** — visited but children not yet visited.
- **Black** — visited and all children marked.

The algorithm:

```
1. (atomic) Mark roots gray.
2. (incremental) While any gray exists:
     - Pop a gray object.
     - Mark its children gray (if white).
     - Mark it black.
3. (atomic) Visit "remembered" set, weak tables, finalize candidates.
4. (incremental) Sweep: free any remaining white; flip white→white' for next cycle.
```

The interleaving with mutation ("incremental") is the tricky part. While the mutator is running, it can create new references that violate the **tri-color invariant**: *no black object points to a white object*. Lua enforces this with a **forward write barrier**:

```c
/* lgc.h - simplified */
#define luaC_barrier(L, p, v) \
    (iscollectable(v) && isblack(p) && iswhite(gcvalue(v)) ? \
     luaC_barrier_(L, obj2gco(p), gcvalue(v)) : (void)0)
```

When you write a black object's field with a white value, the barrier either marks the white value gray (forward) or marks the black object gray again (backward, used for tables: it's cheaper to revisit a table than to scan every key/value at write time).

### 7.2 The Two White Colors

Lua actually has **two whites** that alternate. After a sweep, all surviving objects are flipped from `currentwhite` to `otherwhite`. New allocations during the next mark phase use `currentwhite`. This way, sweep never accidentally collects an object allocated during the cycle.

### 7.3 Tunables — `collectgarbage`

```lua
collectgarbage("collect")        -- full cycle now
collectgarbage("stop")           -- disable
collectgarbage("restart")        -- enable
collectgarbage("count")          -- KB used
collectgarbage("step", n)        -- run one step of size n
collectgarbage("setpause", 200)  -- threshold = 200% (default)
collectgarbage("setstepmul", 100)-- step / alloc ratio (default 100)
```

- **gcpause** (default 200): how much memory must grow above the previous live set before triggering a new cycle. 200 = 2× growth before next GC.
- **gcstepmul** (default 100): how many bytes get scanned per byte allocated. 100 = 1:1; higher = more aggressive.

### 7.4 Generational GC (5.4)

Lua 5.4 added a generational mode:

```lua
collectgarbage("generational", 20, 100)  -- minormul, majormajor
collectgarbage("incremental")             -- back to incremental
```

Generational mode keeps two generations: young (recently allocated) and old (survived a major collection). Minor cycles collect only young; major cycles collect both. The hypothesis: most objects die young.

Internally, generational mode reuses the tri-color machinery but with extra sets: a "remembered" set of old objects with pointers to young, used as roots during minor cycles.

### 7.5 Weak Tables

```lua
local cache = setmetatable({}, {__mode = "v"})  -- weak values
cache[k1] = create_expensive(k1)
-- if no other reference exists to the value, GC will reclaim it
```

Modes:
- `"k"` — weak keys; entry removed if key is otherwise unreachable.
- `"v"` — weak values; entry removed if value is otherwise unreachable.
- `"kv"` — both. Often called "ephemerons" but Lua's semantics are simpler than true ephemerons.

Weak tables interact carefully with the mark phase: the mark walks through them but doesn't mark the weak side. After mark, the GC sweeps weak tables, deleting entries whose weak side is white.

### 7.6 Finalization — `__gc`

```lua
local resource = setmetatable({}, {
    __gc = function(self)
        print("releasing", self)
    end,
})
```

For *tables*, the `__gc` field must be set **before** `setmetatable`, or Lua won't recognize it as finalizable. (Subtle 5.4 rule.) For *userdata*, the `__gc` runs whenever the userdata is unreachable. Finalization runs in a dedicated phase: white objects with `__gc` are first resurrected, marked, and queued; `__gc` runs in the mutator after the cycle; the next cycle reaps them for real.

---

## 8. Coroutines as Continuations

### 8.1 Symmetric Coroutines

Lua's coroutines are **symmetric, asymmetric, stackful**:
- Stackful: each has its own stack and can yield from any depth.
- Asymmetric: coroutines have a parent-child resume relationship — yield returns to the resumer, not to a peer.

This is more powerful than Python's generators (which yield only from the top frame) and equivalent in expressiveness to call/cc plus mutable state.

### 8.2 The Basic API

```lua
local co = coroutine.create(function(x, y)
    print("start", x, y)
    local r1 = coroutine.yield(x + y)
    print("resumed1", r1)
    local r2 = coroutine.yield(r1 * 2)
    print("resumed2", r2)
    return "done"
end)

print(coroutine.resume(co, 3, 4))   -- start 3 4 / true 7
print(coroutine.resume(co, "a"))    -- resumed1 a / true (a*2 — error here)
print(coroutine.status(co))         -- suspended (or dead)
```

`resume` returns `true, ...values...` on success or `false, err` on error. `yield` returns the values passed to the next `resume`.

### 8.3 The State Machine

```
                resume()
[suspended] ─────────────► [running]
     ▲                          │
     │      yield()             │
     └──────────────────────────┘
                                │
                                │ function returns
                                ▼
                            [dead]
```

A coroutine that errored is also `dead`. `coroutine.status(co)` reports the state. There's also a `normal` state — a coroutine that resumed another coroutine. The chain ends at the *main thread*.

### 8.4 `coroutine.wrap`

```lua
local gen = coroutine.wrap(function()
    for i = 1, 3 do coroutine.yield(i * 10) end
end)

for v in gen do print(v) end  -- 10 20 30
```

`wrap` returns a function that resumes the coroutine and propagates errors. Cleaner than manual resume in iterators. Returns the values directly (not `true, ...`).

### 8.5 Producer-Consumer

```lua
local function producer()
    for i = 1, 5 do coroutine.yield("item " .. i) end
end

local function consumer(p)
    while true do
        local ok, item = coroutine.resume(p)
        if not ok or item == nil then break end
        print("consumed", item)
    end
end

consumer(coroutine.create(producer))
```

This is the classic dataflow pattern. Each `yield`/`resume` is a context switch — but unlike OS threads, no kernel involvement. Each switch is a few hundred nanoseconds: save PC, base, top; load the other's.

### 8.6 The C-API Side

```c
int co_resume(lua_State *L) {
    lua_State *co = lua_tothread(L, 1);
    int nargs = lua_gettop(L) - 1;
    int status = lua_resume(co, L, nargs, &nargs);
    if (status == LUA_OK || status == LUA_YIELD) {
        lua_xmove(co, L, nargs);
        return nargs;
    }
    lua_xmove(co, L, 1);  /* error message */
    return -1;
}
```

`lua_resume(co, from, nargs, nresults)` runs `co` until it yields or returns. `lua_xmove(from, to, n)` shuffles values between two threads' stacks. Notice: `lua_yield` takes the running thread's `L` and longjmps out — your C code that called `lua_yield` *must* be a continuation function or be careful about C-stack state.

### 8.7 Yieldable C Calls

If your C function calls Lua code that might yield, you need a continuation:

```c
static int k(lua_State *L, int status, lua_KContext ctx) {
    /* runs after yield resumes */
    int n = lua_tointeger(L, -1);
    return 1;  /* return value already on stack */
}

static int my_func(lua_State *L) {
    /* call something that might yield */
    int rc = lua_pcallk(L, 0, 1, 0, 0, k);
    return k(L, rc, 0);  /* if no yield, fall through */
}
```

`pcallk`/`callk` versions accept a continuation function `k` and a context. If the call yields, when it resumes Lua re-enters `k` instead of unwinding the C stack.

---

## 9. The C API in Depth

### 9.1 lua_State and lua_newstate

```c
lua_State *L = luaL_newstate();    /* with default allocator */
luaL_openlibs(L);                   /* load std libs */

if (luaL_dofile(L, "script.lua")) {
    fprintf(stderr, "%s\n", lua_tostring(L, -1));
}

lua_close(L);
```

`luaL_newstate` is `lua_newstate(default_alloc, NULL)`. You can provide your own allocator — useful for fixed-memory embedded systems:

```c
static void *my_alloc(void *ud, void *ptr, size_t osize, size_t nsize) {
    if (nsize == 0) { free(ptr); return NULL; }
    return realloc(ptr, nsize);
}
lua_State *L = lua_newstate(my_alloc, NULL);
```

### 9.2 The Push/To/Check Family

| Push (C → Lua) | To (Lua → C, lenient) | Check (Lua → C, strict) |
|:----|:----|:----|
| `lua_pushnil` | — | — |
| `lua_pushboolean(L, b)` | `lua_toboolean` | — (use `lua_isboolean`) |
| `lua_pushinteger(L, n)` | `lua_tointeger` | `luaL_checkinteger` |
| `lua_pushnumber(L, n)` | `lua_tonumber` | `luaL_checknumber` |
| `lua_pushstring(L, s)` | `lua_tostring` | `luaL_checkstring` |
| `lua_pushlstring(L, s, n)` | `lua_tolstring` | `luaL_checklstring` |
| `lua_pushcfunction(L, fn)` | — | — |
| `lua_pushcclosure(L, fn, n)` | — | — |
| `lua_pushlightuserdata(L, p)` | `lua_touserdata` | — |
| `lua_pushvalue(L, idx)` | — | — |
| `lua_pushfstring(L, fmt, ...)` | — | — |

The "to" family returns 0/NULL on failure. The "check" family raises a Lua error. Most C extension code uses `check*` functions.

### 9.3 Registering Functions

Two patterns:

```c
/* 1. Single function as global */
lua_pushcfunction(L, my_func);
lua_setglobal(L, "my_func");

/* 2. Library table */
static const luaL_Reg mylib[] = {
    {"foo", l_foo},
    {"bar", l_bar},
    {NULL, NULL}
};

int luaopen_mylib(lua_State *L) {
    luaL_newlib(L, mylib);   /* creates table, registers fns */
    return 1;
}
```

Pattern 2 is the standard: a `luaopen_X` function returns the library as a table. Lua's `require("X")` will dlopen the shared library, look up `luaopen_X`, call it, cache the returned table.

### 9.4 Light vs Full Userdata

```c
/* Light userdata: just a void* tagged with type=USERDATA */
lua_pushlightuserdata(L, my_ptr);

/* Full userdata: GC-managed memory block with optional metatable */
MyType *u = (MyType *)lua_newuserdata(L, sizeof(MyType));
luaL_setmetatable(L, "MyType");
```

Light userdata is just a tagged pointer — no GC, no metatable, equal-by-pointer-value. Full userdata is a GC-managed block — has a metatable, can have `__gc` for cleanup, equal-by-identity unless `__eq` overrides.

```c
/* Define metatable for MyType */
luaL_newmetatable(L, "MyType");
lua_pushcfunction(L, my_gc); lua_setfield(L, -2, "__gc");
lua_pushcfunction(L, my_index); lua_setfield(L, -2, "__index");
lua_pop(L, 1);  /* pop metatable from stack */
```

### 9.5 The Registry — Hidden References

Sometimes a C extension needs to keep a Lua value alive across calls without exposing it to Lua code. The **registry** is a hidden table accessible at `LUA_REGISTRYINDEX`:

```c
/* Stash a Lua value and get an integer reference */
int ref = luaL_ref(L, LUA_REGISTRYINDEX);

/* Later, push it back */
lua_rawgeti(L, LUA_REGISTRYINDEX, ref);

/* Free */
luaL_unref(L, LUA_REGISTRYINDEX, ref);
```

`luaL_ref` pops a value from the stack and returns an integer key into the registry. The registry holds the value until you `unref`. This is how callbacks-from-C-back-into-Lua are typically stored.

### 9.6 Error Handling

```c
if (lua_pcall(L, nargs, nresults, msgh) != LUA_OK) {
    const char *err = lua_tostring(L, -1);
    /* handle */
    lua_pop(L, 1);
}
```

Inside a `pcall` boundary, errors `longjmp` back to the pcall, leaving the error value on top of stack. The optional `msgh` is the index of a message handler — typically `debug.traceback` so you get a stack trace appended:

```c
lua_pushcfunction(L, traceback_handler);
int msgh = lua_gettop(L);
lua_call_setup_args(...);
lua_pcall(L, nargs, nresults, msgh);
lua_remove(L, msgh);
```

### 9.7 The C Continuation API (Yieldable C)

If your C function might call Lua that yields:

```c
static int my_step(lua_State *L, int status, lua_KContext ctx) {
    /* called after each yield */
    return 1;
}

static int my_func(lua_State *L) {
    /* push args... */
    return lua_callk(L, nargs, nresults, ctx, my_step);
}
```

The continuation receives the status (`LUA_OK`, `LUA_YIELD`, error) and the context you passed. Without continuations, your C frame would block the yield.

---

## 10. LuaJIT Internals

LuaJIT is Mike Pall's tracing JIT for Lua 5.1. It's a separate codebase from PUC-Rio Lua and implements its own VM, GC, parser, and JIT compiler. For numerics-heavy code it often beats hand-written C.

### 10.1 The Trace Compiler — Brief

Tracing JITs work by:
1. Running everything in an interpreter.
2. Counting iterations of every loop and call site.
3. When a counter exceeds a threshold (`hotloop`, default 56 in LuaJIT), start **recording**: the interpreter logs every operation it executes.
4. When the recording reaches a backward branch matching the start, the trace is closed. Compile to machine code.
5. Replace the loop entry with a jump to the compiled trace.

Side branches off the main trace (e.g., a different value seen in a comparison) become **side traces** that are recorded the same way.

### 10.2 The IR — SSA Form

LuaJIT uses a clean SSA IR. You can see it:

```bash
luajit -jdump=isr file.lua > dump.txt
```

Output:

```
---- TRACE 1 start
0001  ADDVN  R(2) R(1) +1
0002  SLOAD  #1   T  R
0003  ISEQ   R(2) +10
...
---- TRACE 1 IR
0001  int SLOAD  #2   PI
0002  num SLOAD  #3   T  
0003  > num CONV  num int   0002
0004  + num ADD   0003 +1
...
---- TRACE 1 mcode 256
0xfffe2c20  mov ecx, [rdx+0x8]
...
```

The phases: bytecode trace (`isr` = interpreter trace + IR + mcode) → IR → optimized IR → machine code. The optimizer does dead code elimination, common subexpression elimination, loop-invariant code motion, allocation sinking, narrowing (replacing 64-bit ops with 32-bit when safe).

### 10.3 NYI — Operations That Abort

LuaJIT can't compile every operation. The "Not Yet Implemented" list ([http://wiki.luajit.org/NYI](http://wiki.luajit.org/NYI)) includes:
- `pairs`, `ipairs` on tables with `__pairs`/`__ipairs` (5.2 metas)
- `string.gmatch`, `string.gsub` callback variant
- `pcall`/`xpcall` with errors caught (uncaught is fine)
- `coroutine.resume`/`coroutine.yield` — the trace boundary
- `string.format` with `%a`/`%g` in some cases
- `os.*` (most), `io.*` (most)
- Calls into yieldable C frames
- `setfenv`/`getfenv` (Lua 5.1)

When the trace recorder hits an NYI it aborts. The interpreter continues running. If a hot loop keeps aborting, LuaJIT eventually blacklists it.

### 10.4 jit.* Control

```lua
require "jit"
print(jit.version)        -- LuaJIT 2.1.0-beta3
print(jit.os, jit.arch)   -- Linux x64

jit.off()                 -- disable JIT for the rest of execution
jit.on()                  -- re-enable
jit.flush()               -- discard all compiled traces

require "jit.opt".start("hotloop=10")  -- trace earlier
```

```bash
luajit -jv prog.lua          # one line per trace event
luajit -jdump=+rs prog.lua   # full dump with regs and snapshots
luajit -joff prog.lua        # interpreter only
luajit -O0 prog.lua          # disable optimizer
luajit -p -1,prof.txt prog.lua  # profiler
```

### 10.5 Exit Snapshots

When a trace exits (taking a side branch, encountering a guard failure), LuaJIT must restore the interpreter state — registers, stack values, PC. The IR records *snapshots* at every exit point, listing exactly which IR values map to which interpreter state. The exit handler walks the snapshot, materializes the values, returns to the interpreter. Snapshots also enable allocation sinking — an object allocated only to satisfy a guard's snapshot can be elided in the compiled code, materialized only on actual exit.

### 10.6 Tuning Tips

- Avoid NYI operations in hot loops.
- Localize globals: `local sin = math.sin` outside the loop.
- Use the FFI for C struct access — avoid the cost of metatables.
- Profile with `-jp=v` or `-jp=Fl` to see where time goes.

---

## 11. The FFI Library

LuaJIT's FFI is its killer feature: parse C declarations at runtime and call C functions, allocate structs, pointer-arith — without writing any C glue.

### 11.1 Basic ffi.cdef

```lua
local ffi = require("ffi")

ffi.cdef[[
typedef struct { double x, y; } point_t;

int printf(const char *fmt, ...);
double sqrt(double x);
void *malloc(size_t size);
void free(void *ptr);
]]

local p = ffi.new("point_t", 3, 4)
local d = ffi.C.sqrt(p.x*p.x + p.y*p.y)
ffi.C.printf("distance = %f\n", d)
```

`ffi.cdef` parses C declarations (a subset of C — no preprocessor, no inline functions). `ffi.C` is a namespace for the loaded C library symbols.

### 11.2 ffi.new and Memory Layout

```lua
local p = ffi.new("point_t")          -- zero-init
local q = ffi.new("point_t", 1, 2)    -- positional
local r = ffi.new("point_t", {x=1, y=2})  -- table init

local arr = ffi.new("int[10]")        -- 10-int array
arr[0] = 42                            -- 0-indexed!

local sz = ffi.sizeof("point_t")      -- 16 on x64
local ofs = ffi.offsetof("point_t", "y")  -- 8
```

FFI struct access is *zero-copy*: `p.x` reads directly from the C memory. Compare with PUC-Rio Lua, where you'd allocate a userdata, expose getters/setters via metatable, copy values across boundaries.

### 11.3 ffi.cast and Pointer Arithmetic

```lua
local buf = ffi.new("uint8_t[1024]")
local p = ffi.cast("uint32_t*", buf)
p[0] = 0xdeadbeef                       -- writes 4 bytes

-- Pointer arithmetic
local ip = ffi.cast("int*", buf)
local ip2 = ip + 1                       -- advance 4 bytes
print(ip2 - ip)                          -- 1 (in int units)
```

`ffi.cast` reinterprets without copying. `ffi.fill(buf, 1024, 0)` is the FFI equivalent of `memset`.

### 11.4 ffi.metatype — Adding Methods

```lua
ffi.cdef[[
typedef struct { double x, y; } vec2_t;
]]

local vec2_mt = {
    __add = function(a, b) return ffi.new("vec2_t", a.x+b.x, a.y+b.y) end,
    __tostring = function(v) return string.format("(%g, %g)", v.x, v.y) end,
    __index = {
        length = function(v) return math.sqrt(v.x*v.x + v.y*v.y) end,
    },
}

local vec2 = ffi.metatype("vec2_t", vec2_mt)

local a = vec2(1, 2)
local b = vec2(3, 4)
print(a + b)                  -- (4, 6)
print((a+b):length())          -- 7.211...
```

`ffi.metatype` permanently associates a metatable with a cdata type. Now you have struct-with-methods at C speed. The trace compiler inlines these calls aggressively.

### 11.5 Loading Shared Libraries

```lua
local lib = ffi.load("z")     -- libz.so / z.dll
ffi.cdef[[
unsigned long crc32(unsigned long crc, const unsigned char *buf, unsigned int len);
]]
local crc = lib.crc32(0, "hello", 5)
```

`ffi.load` mimics `dlopen`/`LoadLibrary`. With `ffi.C` you get the symbols of the host process; with `ffi.load` you get a specific library.

### 11.6 Why FFI Is Fast

The trace compiler treats FFI calls as inlinable opcodes. A call to `ffi.C.sqrt(x)` in a hot loop becomes a `sqrtsd` x86 instruction. A read of `p.x` becomes a `movsd [reg+0]`. There is no marshalling, no Lua-stack manipulation, no GC pressure. This is why LuaJIT + FFI often beats traditional C extensions: the C extension goes through the C API, while FFI goes directly through the JIT.

---

## 12. Embedding Patterns

### 12.1 Redis Lua Scripting

Redis embeds Lua 5.1 (LuaJIT in some forks) for atomic, deterministic scripting:

```lua
-- KEYS and ARGV are pre-populated by Redis
local current = redis.call("GET", KEYS[1])
if current and tonumber(current) >= tonumber(ARGV[1]) then
    redis.call("INCR", KEYS[1])
    return 1
end
return 0
```

Constraints:
- **Sandboxed**: `os`, `io`, `package`, `loadfile` are removed.
- **Deterministic**: scripts must produce same output for same input — Redis blocks `math.random` after first call seeding via input.
- **Atomic**: script runs to completion under the global lock; no other commands interleave.
- **Time-bounded**: `lua-time-limit` (default 5s) terminates runaway scripts.
- **No side effects outside Redis**: no networking, no file I/O.

`EVAL script numkeys k1 k2 ... a1 a2 ...` runs a script. `EVALSHA sha1 ...` runs a previously-loaded script by SHA1 hash, avoiding the cost of recompiling and resending.

### 12.2 Nginx + OpenResty

OpenResty embeds LuaJIT into nginx with a coroutine-per-request model:

```nginx
location /api {
    content_by_lua_block {
        local cjson = require "cjson"
        local res = ngx.location.capture("/upstream")
        ngx.say(cjson.encode({status = res.status, body = res.body}))
    }
}
```

Hooks: `init_by_lua` (master init), `init_worker_by_lua`, `set_by_lua`, `rewrite_by_lua`, `access_by_lua`, `content_by_lua`, `header_filter_by_lua`, `body_filter_by_lua`, `log_by_lua`. Each runs in its own coroutine; cosocket I/O (`ngx.socket.tcp`) yields the coroutine while waiting on the kernel, giving you cooperative async I/O without callbacks.

The `lua-resty-*` ecosystem provides Redis, MySQL, memcached, DNS, and many other clients all built on cosocket. Performance: tens of thousands of req/s per worker.

### 12.3 Neovim

Neovim embeds LuaJIT and exposes editor features through `vim.api`, `vim.fn`, `vim.cmd`:

```lua
vim.api.nvim_set_keymap('n', '<leader>w', ':w<CR>', {noremap=true, silent=true})

vim.cmd [[
  augroup MyGroup
    autocmd!
    autocmd BufWritePost *.lua echo "saved Lua file"
  augroup END
]]

local lines = vim.api.nvim_buf_get_lines(0, 0, -1, false)
print(#lines)
```

`vim.fn` calls Vim functions. `vim.cmd` runs Ex commands. `vim.api` calls Neovim's C-level API. The configuration ecosystem (lazy.nvim, packer, telescope) is entirely Lua-based since Neovim 0.5.

### 12.4 Game Engines — LÖVE, World of Warcraft, Roblox

LÖVE (love2d) is a pure Lua game framework: define `love.update(dt)` and `love.draw()` and the engine drives them at 60Hz. World of Warcraft's UI is entirely Lua (PUC-Rio 5.1). Roblox uses Luau, a fork with type annotations. Garry's Mod, the Cryengine, the Witcher, Don't Starve — Lua-scripted.

The pattern is consistent: the engine is C/C++ for performance; Lua scripts game logic; the Lua sandbox prevents user-mod scripts from breaking out.

---

## 13. Debugging and Introspection

### 13.1 Stack Traces

```lua
local function f()
    error("boom")
end

local ok, err = xpcall(f, debug.traceback)
print(err)
-- boom
-- stack traceback:
--     in function 'error'
--     in function <input:2>
--     ...
```

`xpcall(f, msgh)` calls `f` and on error invokes `msgh(err)` while the stack is still alive. `debug.traceback` walks the call stack and formats it. Without `xpcall + traceback`, you only get the error message.

### 13.2 debug.getinfo

```lua
local info = debug.getinfo(2, "Slnf")  -- level 2, source/line/name/func
-- info.source   "@/path/to/file.lua"
-- info.short_src "...file.lua"
-- info.linedefined 12
-- info.lastlinedefined 18
-- info.what       "Lua" | "C" | "main"
-- info.currentline 15
-- info.name       "myfunc"
-- info.namewhat   "local" | "global" | "method" | "field" | ""
-- info.func       function value
```

The level argument: 0 = `getinfo` itself, 1 = caller, 2 = caller's caller. The format string selects which fields to populate (cheaper than getting all).

### 13.3 debug.sethook

```lua
debug.sethook(function(event, line)
    if event == "line" then
        print("line", line)
    elseif event == "call" then
        print("call", debug.getinfo(2).name)
    elseif event == "return" then
        print("return")
    end
end, "lcr")  -- "l" line, "c" call, "r" return; "12" = count(12)
```

Modes:
- `"l"` — every line
- `"c"` — every function call
- `"r"` — every function return
- `"#count"` (e.g., `"100"`) — every N instructions

This is how profilers and step-debuggers are built. Beware: hooks have nontrivial overhead.

### 13.4 debug.getlocal / setlocal / getupvalue

```lua
local function f(x, y)
    local sum = x + y
    return sum
end

f(10, 20)
-- inside hook, after f returns:
-- debug.getlocal(level, idx) returns name, value
-- debug.getupvalue(closure, idx) for upvalues
```

Subtle: parameter slots are negative-indexed (`-1` for first param) when querying via `debug.getinfo` with `'u'` flag plus `getlocal`. In practice, `debug.getlocal(level, i)` from `i=1` walks all locals plus params. Returns `nil` past the last.

### 13.5 The Canonical Error-Wrap

```lua
local function safe_call(f, ...)
    local args = {...}
    return xpcall(function() return f(table.unpack(args)) end,
                  function(err)
                      return debug.traceback(tostring(err), 2)
                  end)
end

local ok, ret_or_err = safe_call(may_fail, arg1, arg2)
if not ok then
    log("ERROR: " .. ret_or_err)
end
```

Note `table.unpack` (5.3+) — was global `unpack` in 5.1. The level 2 in `traceback` skips the wrapper itself.

---

## 14. Performance Characteristics

### 14.1 Local vs Global Access

A local variable is a register slot — direct array index, no hashing. A global is `_ENV.x` — a `GETTABUP` opcode that hashes `"x"`, walks the metatable chain on `_ENV` (which is normally just the globals table), reads.

```lua
-- BAD: global lookup every iteration
for i = 1, 1e6 do
    s = string.format("%d", i)
    print(s)
end

-- GOOD: localize
local format = string.format
local print = print
for i = 1, 1e6 do
    local s = format("%d", i)
    print(s)
end
```

Benchmarks typically show 30–50% speedup from localizing hot library functions. With LuaJIT this matters less in compiled traces (the JIT inlines), but for code that hasn't yet been traced it still applies.

### 14.2 String Concatenation — `..` vs `table.concat`

```lua
-- O(N^2): each .. creates a new string
local s = ""
for i = 1, 1e5 do
    s = s .. tostring(i) .. ","
end

-- O(N): build table, concat once
local parts = {}
for i = 1, 1e5 do
    parts[#parts+1] = tostring(i)
end
local s = table.concat(parts, ",")
```

The first form builds 100,000 intermediate strings, hashes each, interns each. The second builds one final string. Order-of-magnitude difference at scale.

### 14.3 Tail Call Elimination

Lua mandates TCO:

```lua
local function fact(n, acc)
    acc = acc or 1
    if n <= 1 then return acc end
    return fact(n-1, acc * n)   -- tail call: zero stack growth
end

print(fact(10000))  -- works without stack overflow
```

`return fn(args)` is a tail call. The VM uses `OP_TAILCALL` which reuses the current call frame. But the call must be in a tail position with no wrapping:

```lua
return f(x) + 1     -- NOT tail call (needs to add)
return (f(x))       -- NOT tail call (parens force single value)
return f(x), 1      -- NOT tail call (multi-return)
return f(x)         -- tail call
```

### 14.4 Method Call Cost

`obj:method()` is `obj.method(obj)` — first lookup `method` via `__index` chain (potentially many table lookups), then call. With `flags` cache on metatables, the per-call overhead is small but nonzero. Hot paths often inline by hand:

```lua
-- Hot path: cache the method
local method = obj.method
for i = 1, 1e6 do method(obj) end
```

Or, for FFI types under LuaJIT, use `ffi.metatype` — the JIT inlines through it.

### 14.5 Avoiding Allocations

GC pressure dominates many Lua programs. Common allocation sources:

- `table.insert(t, x)` allocates if the array part needs to grow. Pre-size when possible.
- `string.format` allocates the result string.
- Closures allocate every time they're created.
- Tables allocate.

Patterns:
- Reuse tables: `for k in pairs(t) do t[k] = nil end` clears in place.
- Use pools for hot-path objects.
- Build once outside the loop.

### 14.6 LuaJIT-Specific Tips

- Trust the trace compiler — write idiomatic Lua, then profile.
- Cache `ffi.C.fn` in a local — saves the namespace lookup.
- Avoid `pairs` for hot loops over arrays; use `for i = 1, #t`.
- If profiling shows trace aborts, find and remove the NYI op.

---

## 15. Module System

### 15.1 require, package.path, package.cpath

```lua
-- Lua searches package.path (Lua files), then package.cpath (.so/.dll)
package.path  = "./?.lua;./?/init.lua;/usr/share/lua/5.4/?.lua"
package.cpath = "./?.so;/usr/lib/lua/5.4/?.so"

local foo = require("foo")
-- searches: ./foo.lua, ./foo/init.lua, /usr/share/lua/5.4/foo.lua, ...
-- if not found, ./foo.so, /usr/lib/lua/5.4/foo.so, ...
```

`require` does:
1. Look in `package.loaded[modname]` — if cached, return it.
2. Walk `package.searchers` (5.2+; was `package.loaders` in 5.1):
   a. Check `package.preload[modname]` for a preregistered loader.
   b. Search `package.path` and try to compile the file.
   c. Search `package.cpath` and `dlopen` + look up `luaopen_<modname>`.
   d. Search for combined Lua/C loader.
3. Call the loader function. Take its return value (or `true` if nil).
4. Cache in `package.loaded[modname]`.
5. Return the cached value.

The `?` is replaced with the modname (with `.` swapped to path separator). So `require "a.b.c"` searches `./a/b/c.lua`, `./a/b/c/init.lua`, etc.

### 15.2 The Module Pattern

The modern (5.1+ that-isn't-deprecated) way:

```lua
-- mymod.lua
local M = {}

local secret = "hidden"  -- module-private

function M.greet(name)
    return "hello, " .. name
end

function M.compute(x)
    return x * 2 + #secret
end

return M
```

```lua
-- consumer.lua
local mymod = require("mymod")
print(mymod.greet("world"))
```

Key rules:
- Return a table (or any value) at end of file — that's what `require` caches.
- Don't pollute globals.
- Module-private state is just locals at file top.

### 15.3 The Deprecated `module()` Function

Lua 5.1 had:

```lua
module("mymod", package.seeall)
function greet(name) ... end  -- silently global within mymod
```

This had multiple footguns: it modified the global environment, used `setfenv` (removed in 5.2), and made testing painful. Removed in 5.2. If you see it, it's old code.

### 15.4 The Forgotten Return

```lua
-- mymod.lua
local M = {}
function M.foo() ... end
-- forgot: return M
```

`require("mymod")` returns `true` (the default), not your table. Calling `mymod.foo` then errors with "attempt to index boolean". A surprisingly common bug. Always end module files with `return M`.

### 15.5 Reload During Development

```lua
-- Force a re-require:
package.loaded.mymod = nil
local mymod = require("mymod")
```

Useful for REPLs, but be careful: any closures captured from the old module still reference the old module's locals. State doesn't migrate.

---

## 16. PUC-Rio vs LuaJIT Diff

LuaJIT is not "fast Lua" — it is a different implementation with a different language version. Understanding the gap matters for any project choosing between them.

### 16.1 Version Compatibility

| Feature | PUC-Rio 5.4 | LuaJIT 2.1 |
|:--------|:----:|:----:|
| Integer subtype (5.3+) | Yes | Partial via 64-bit FFI |
| Bitwise operators `& \| ~ << >>` | Yes (5.3+) | Via `bit.*` library / FFI |
| `goto` labels (5.2+) | Yes | Yes |
| Generational GC (5.4) | Yes | No |
| `__close` finalizers (5.4) | Yes | No |
| `<close>` / `<const>` attributes (5.4) | Yes | No |
| `__pairs`/`__ipairs` metas | Yes | `__pairs` no, must use `pairs` workaround |
| `string.pack`/`unpack` (5.3+) | Yes | Yes (LuaJIT 2.1 added) |
| `utf8` library (5.3+) | Yes | Yes (LuaJIT 2.1 added) |
| `ffi` library | No | Yes |
| `bit` library | No (use 5.3 ops) | Yes |
| `jit.*` namespace | No | Yes |

LuaJIT is fundamentally a **5.1 VM** with selected 5.2 and 5.3 features back-ported. The integer/float distinction does not exist at the value level — everything is `lua_Number` (double) plus FFI cdata.

### 16.2 Performance

For typical Lua-only code, LuaJIT is 5–50× faster than PUC-Rio. For numerics-heavy or FFI-using code, LuaJIT often matches or beats hand-written C. PUC-Rio's interpreter is itself fast (one of the fastest interpreters around) but has no JIT.

### 16.3 GC

PUC-Rio has incremental tri-color and a generational mode. LuaJIT has its own non-generational, non-incremental mark-sweep — older and simpler. There's a long-running "GC64" project for 64-bit pointer support and a planned new GC, but at time of writing the released LuaJIT 2.1 still uses the old GC.

### 16.4 String Library

PUC-Rio's `string.match` and friends are written in C with the unique Lua pattern syntax. LuaJIT also has these, with subtle differences in performance. Neither uses PCRE — Lua patterns are simpler (no alternation, no backtracking).

### 16.5 Choosing

Use LuaJIT when:
- Performance is critical.
- You need FFI for C struct access.
- 5.1 + selected 5.2/5.3 is enough.

Use PUC-Rio when:
- You need 5.4 features (`<close>`, generational GC, full integer support).
- You're embedding in a setting where JIT codegen is forbidden (iOS, some game consoles).
- You need a very small runtime and don't need the speed.

---

## 17. Idioms at the Language Level

### 17.1 Varargs

```lua
local function f(...)
    local n = select("#", ...)         -- count
    local first = select(1, ...)        -- first
    local rest = {...}                  -- pack into table
    local t = table.pack(...)           -- {n=N, ...}  (5.2+)
end

f(1, 2, 3)
```

`select("#", ...)` is the only way to count varargs including nils. `{...}` stops at the first nil (sequence rule). `table.pack` always sets `n` correctly.

### 17.2 Multi-Return Chains

```lua
local function divmod(a, b)
    return a // b, a % b
end

local q, r = divmod(17, 5)         -- 3, 2
print(divmod(17, 5))                -- 3 2 (passed as separate args)

local t = {divmod(17, 5)}           -- {3, 2}
local fst = (divmod(17, 5))         -- 3 (parens collapse to first)
```

Multi-return values fan out only at:
- The last position of an arg list: `f(a, divmod(x, y))` passes 3 args.
- The last position of a constructor: `{a, divmod(x, y)}` is `{a, q, r}`.
- The last position of a return statement.
- A `local` assignment's RHS at last position.

In other positions (`f(divmod(x, y), a)`), only the first return is taken.

### 17.3 The `...` in Tail Position

```lua
local function fwd(target, ...)
    return target(...)               -- forwards all varargs
end

fwd(print, "a", "b", "c")            -- a   b   c
```

Dropping `...` mid-list collapses to first value. Always tail-position to forward.

### 17.4 `goto` Continue (5.2+)

Lua has no `continue`. The idiom:

```lua
for i = 1, 10 do
    if i % 2 == 0 then goto continue end
    print(i)                            -- 1 3 5 7 9
    ::continue::
end
```

`goto` with `::label::` is also the cleanest way to break out of multiple loops.

### 17.5 Integer/Float Subtype (5.3+)

```lua
print(math.type(1))       -- integer
print(math.type(1.0))      -- float
print(math.type("1"))      -- nil (not a number)

print(1 // 2)               -- 0     (integer floor div)
print(1.0 // 2)             -- 0.0   (float floor div)
print(7 / 2)                -- 3.5   (always float)
print(7 % 2)                -- 1     (integer)

print(1 == 1.0)             -- true  (cross-subtype equal)
print(math.tointeger(2.0))  -- 2
print(math.tointeger(2.5))  -- nil
```

The `//` (floor division) and `%` (modulo) preserve subtype: int op int = int, but any float operand promotes. `/` is *always* float division. Integer overflow wraps two's-complement for `+`, `-`, `*`; raises for `//` and `%` by zero.

### 17.6 Bitwise Operators (5.3+)

```lua
print(0xff & 0x0f)         -- 15
print(0xff | 0x100)         -- 511
print(0xff ~ 0xaa)          -- 85  (XOR — `~` is binary)
print(~0xff)                -- -256 (unary NOT)
print(1 << 4)               -- 16
print(256 >> 4)             -- 16
```

Pre-5.3 Lua used `bit32` library or LuaJIT's `bit` library. LuaJIT supports the `bit` library across versions but not the operator syntax (which is parsed as `~ ~` separately).

### 17.7 `<close>` (5.4)

```lua
do
    local f <close> = io.open("file", "r")
    -- use f
end  -- f:close() called automatically (via __close metamethod)
```

RAII for Lua. The `__close` metamethod is called when the variable goes out of scope (normally or via error). The value must support `__close` or be `nil` or `false`. Replaces the `pcall + close` pattern for resource cleanup.

### 17.8 `<const>` (5.4)

```lua
local PI <const> = 3.14159
PI = 4   -- compile error: attempt to assign to const variable
```

Compile-time check. No runtime cost.

---

## 18. Prerequisites

- C basics: structs, unions, pointers, malloc/free, function pointers — to read the implementation.
- Hash table internals: open vs closed addressing, load factor, rehash.
- Tri-color mark-sweep GC: white/gray/black, write barriers, incremental vs stop-the-world.
- Tracing JIT compilation: hot-path detection, IR, SSA, machine code emission, side traces, deoptimization, NYI lists.
- Continuations and coroutines: stackful vs stackless, symmetric vs asymmetric.
- Operator overloading and metaobject protocols.

## Complexity

- Table read (array part hit): O(1).
- Table read (hash part hit, no metatable): O(1) average, O(n) worst case (catastrophic collision).
- Table read with `__index` chain of length k: O(k).
- Table write triggering rehash: O(n) amortized over inserts.
- String comparison after intern: O(1).
- String creation (short, ≤40 bytes 5.4): O(n) hash + intern check.
- String creation (long, >40 bytes 5.4): O(n) copy, no intern.
- Coroutine resume/yield: O(stack-frames-affected) for save/restore, no kernel involvement.
- Closure call: O(1) plus per-upvalue array slot indexing.
- Mark phase per cycle: O(live objects).
- Sweep phase per cycle: O(all objects).
- LuaJIT compiled trace: tens of nanoseconds for simple ops, often equivalent to C.

## See Also

- lua (sheet) — the practical sheet covering syntax, stdlib, and patterns.
- polyglot — language comparisons including Lua's place in the embedded-language space.
- c — the host language Lua is implemented in; the C API is the embedding contract.
- python — comparison with another high-level dynamic language; CPython's stack VM vs Lua's register VM.

## References

- Roberto Ierusalimschy, Luiz Henrique de Figueiredo, Waldemar Celes. "The Implementation of Lua 5.0", Journal of Universal Computer Science, 2005. The canonical paper on the register VM and table design.
- Roberto Ierusalimschy. *Programming in Lua*, fourth edition, lua.org. The book.
- "Lua Performance Tips" by Roberto Ierusalimschy — chapter in *Lua Programming Gems*.
- Lua 5.4 Reference Manual: https://www.lua.org/manual/5.4/
- Lua source code: https://github.com/lua/lua
- LuaJIT documentation: https://luajit.org/luajit.html
- LuaJIT NYI list: http://wiki.luajit.org/NYI
- LuaJIT source code: https://github.com/LuaJIT/LuaJIT
- Mike Pall on tracing: https://www.freelists.org/post/luajit/Some-techniques-used-by-LuaJIT
- OpenResty / lua-nginx-module: https://openresty.org and https://github.com/openresty/lua-nginx-module
- Redis Lua scripting: https://redis.io/docs/manual/programmability/eval-intro/
- Neovim Lua guide: https://neovim.io/doc/user/lua-guide.html
- Roblox Luau: https://luau-lang.org
- Lua-users wiki: http://lua-users.org/wiki/
