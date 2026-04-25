# Lua (Programming Language)

> Small, fast, embeddable scripting language built around tables and first-class functions; the canonical extension language for games, Neovim, Redis, Nginx/OpenResty, and Wireshark.

## Setup

Lua ships in two ecosystems: PUC-Rio reference implementation (`lua`/`luac`) and LuaJIT (`luajit`), a high-performance JIT for the 5.1 dialect with FFI. Most modern PUC-Rio code targets 5.4 (released 2020); embedded contexts (Redis, OpenResty, World of Warcraft, Roblox until 2022, many games) still pin Lua 5.1 because LuaJIT only follows 5.1 + select 5.2 extensions.

```bash
# # macOS — Homebrew
# brew install lua          # 5.4 (default)
# brew install lua@5.3
# brew install lua@5.1
# brew install luajit       # LuaJIT 2.1 beta (5.1 + extensions)
# brew install luarocks     # Package manager (defaults to highest installed Lua)

# # Debian / Ubuntu
# sudo apt install lua5.4 liblua5.4-dev luarocks

# # Arch
# sudo pacman -S lua luarocks
# sudo pacman -S luajit

# # Build from source (PUC-Rio)
# curl -R -O https://www.lua.org/ftp/lua-5.4.7.tar.gz
# tar zxf lua-5.4.7.tar.gz && cd lua-5.4.7
# make all test           # auto-detects platform
# sudo make install       # installs to /usr/local

# # Build LuaJIT
# git clone https://luajit.org/git/luajit.git
# cd luajit && make && sudo make install

# # Verify install
# lua -v                  # Lua 5.4.7  Copyright (C) 1994-2024 Lua.org, PUC-Rio
# luajit -v               # LuaJIT 2.1.0-beta3 -- Copyright (C) 2005-2023 ...
# luarocks --version
```

### Version differences (high-level)

```bash
# # Lua 5.1  (2006) — LuaJIT base, OpenResty, Redis, ComputerCraft
# #   - module() function, setfenv/getfenv (DEPRECATED in 5.2)
# #   - no goto, no integer/float split, no bitwise operators
# #   - # operator on tables undefined for holey tables (still true)

# # Lua 5.2  (2011)
# #   - _ENV replaces setfenv/getfenv
# #   - goto + ::label::, bit32 library (DROPPED in 5.4 → use bitwise ops)

# # Lua 5.3  (2015)
# #   - integer/float subtypes (math.type, math.tointeger, math.maxinteger)
# #   - native bitwise operators: & | ~ << >> ~ (xor with two args, not with one)
# #   - // floor division operator
# #   - utf8 library, string.pack/unpack/packsize

# # Lua 5.4  (2020)
# #   - <const> and <close> attributes for locals
# #   - generational GC (collectgarbage("generational"))
# #   - integer for loop overflow check
# #   - new warn() function

# # Trap: scripts written for 5.4 will NOT run on LuaJIT (5.1) without changes.
# #       Always test on the deployment runtime.
```

## REPL

```bash
# lua                     # Start REPL  →  Lua 5.4.7 ...  >
# > 1 + 1                 # In 5.3+ shows expression result automatically
# 2
# > print("hi")
# hi
# > = 1 + 1               # = prefix in 5.1 / 5.2 (NOT 5.3+ — they made it implicit)
# 2
# > x = 10                # Statement: no auto-print
# > x                     # In 5.3+ auto-prints; in 5.1 you'd need `= x` or `print(x)`
# 10
# > <CTRL-D>              # Exit (or os.exit())

# # Multi-line: the REPL detects incomplete blocks and switches to >> prompt
# > function f(n)
# >>   return n * 2
# >> end
# > f(21)
# 42
```

## Run scripts

```bash
# lua hello.lua                       # Run script
# lua hello.lua arg1 arg2             # Args available as global `arg`: arg[1]="arg1"
# lua -e 'print(1+1)'                 # Inline code (no script file)
# lua -l json hello.lua               # Pre-load module `json` then run script
# lua -i hello.lua                    # Run script then drop into REPL (inspect state)
# lua -                               # Read script from stdin
# lua -E hello.lua                    # Ignore LUA_INIT (clean env)
# lua -v                              # Print version then exit
# lua -W hello.lua                    # 5.4: enable warn() output

# # Shebang
# #!/usr/bin/env lua                  ← first line of script
# chmod +x hello.lua
# ./hello.lua

# # Bytecode compile / run
# luac -o hello.luac hello.lua        # Compile
# luac -p hello.lua                   # Parse only (syntax check)
# luac -l hello.lua                   # List opcodes (disassemble)
# luac -s -o stripped.luac hello.lua  # Strip debug info (smaller binary)
# lua hello.luac                      # Run bytecode (CAUTION: bytecode is NOT portable across versions)

# # luac flags summary
# # -l  list bytecode (use -l -l for full)
# # -o  output file (default luac.out)
# # -p  parse only, no output
# # -s  strip debug info
# # -v  show version
```

## Variables and Scope

The single biggest Lua footgun: **assignment without `local` creates a global**. There is no `var`/`let` syntax — bare `x = 1` writes to the global table `_G` (or `_ENV` in 5.2+). Always prefer `local`.

```bash
# # Broken — pollutes globals
# x = 10                  -- Global!
# function f()
#     y = 20              -- Global! Even inside a function.
# end

# # Fixed — locals everywhere
# local x = 10
# local function f()
#     local y = 20
# end

# # Scope is lexical and block-structured
# do
#     local secret = "hidden"
# end
# print(secret)           -- nil (out of scope)

# # Shadowing is allowed
# local x = 1
# do
#     local x = 2         -- New binding, shadows outer x
#     print(x)            -- 2
# end
# print(x)                -- 1

# # 5.4 attributes
# local PI <const> = 3.14159        -- Compile-time constant; reassignment is an error
# local file <close> = io.open("f") -- Calls __close metamethod on scope exit (RAII)

# # Inspecting globals
# for k, v in pairs(_G) do print(k, type(v)) end

# # 5.2+ environments via _ENV (advanced)
# -- A function's "globals" really mean `_ENV.x`. Override _ENV to sandbox.
# local sandbox = setmetatable({}, {__index = _G})
# local f = load("x = 5; print(x)", "sb", "t", sandbox)
# f()                     -- prints 5; sandbox.x == 5; _G.x untouched
```

## The 8 Types

Lua has exactly 8 first-class types (some sources say 9 if you split number into integer/float subtypes, but `type()` returns just `"number"`).

```bash
# print(type(nil))         -- "nil"     — only nil, the absence of value
# print(type(true))        -- "boolean" — true / false
# print(type(42))          -- "number"  — integer or float (subtypes 5.3+)
# print(type("hi"))        -- "string"  — immutable, interned, 8-bit clean
# print(type({}))          -- "table"   — the only data structure
# print(type(print))       -- "function" — first class, closures
# print(type(io.stdin))    -- "userdata" — opaque C pointer (e.g., file handle)
# print(type(coroutine.create(function() end)))  -- "thread" — coroutines

# # Truthiness — ONLY nil and false are falsy. Crucial!
# if 0 then print("0 is truthy") end           -- prints
# if "" then print("'' is truthy") end         -- prints
# if {} then print("{} is truthy") end         -- prints
# if nil then print("never") end               -- skipped
# if false then print("never") end             -- skipped
```

## Numbers

Lua 5.3 introduced an integer/float subtype split. Operations preserve subtype where possible; `/` always yields float, `//` (floor division) preserves integer when both operands are integer.

```bash
# # 5.3+ integer/float
# print(math.type(1))           -- "integer"
# print(math.type(1.0))         -- "float"
# print(math.type("1"))         -- nil (not a number value)
# print(1 // 2)                 -- 0     (integer floor div)
# print(1 / 2)                  -- 0.5   (always float)
# print(math.tointeger(3.0))    -- 3
# print(math.tointeger(3.5))    -- nil   (not exact)
# print(math.maxinteger)        -- 9223372036854775807 (64-bit)
# print(math.mininteger)        -- -9223372036854775808

# # Operators
# 5 + 3        -- 8       addition
# 5 - 3        -- 2       subtraction
# 5 * 3        -- 15
# 5 / 3        -- 1.6666... always float
# 5 // 3       -- 1       floor div (5.3+)
# 5 % 3        -- 2       modulo (mathematical: result has sign of divisor)
# 2 ^ 10       -- 1024.0  exponentiation, always float
# -5           -- -5      unary minus

# # Bitwise (5.3+) — integer only
# 0x0F & 0xF0  -- 0       AND
# 0x0F | 0xF0  -- 0xFF    OR
# 0x0F ~ 0xFF  -- 0xF0    XOR (binary ~)
#  ~0          -- -1      bitwise NOT (unary ~)
# 1 << 4       -- 16      left shift
# 256 >> 2     -- 64      right shift (logical, not arithmetic)

# # Number literals
# 42           -- decimal integer
# 0xFF         -- hex integer (255)
# 3.14         -- float
# 1e3          -- 1000.0 (float)
# 0x1.8p2      -- 6.0    hex float (5.3+)

# # math library highlights
# math.pi                        -- 3.14159...
# math.huge                      -- inf
# math.abs(-5)                   -- 5
# math.floor(3.7) / math.ceil(3.2)
# math.sqrt(16)                  -- 4.0
# math.exp(1)  / math.log(math.exp(1))    -- log defaults to natural; pass base as 2nd arg
# math.sin / cos / tan / asin / acos / atan
# math.fmod(10, 3)               -- 1.0  (truncated remainder, sign of dividend)
# math.modf(3.7)                 -- 3.0, 0.7  (integer + fractional parts)
# math.random()                  -- [0, 1)
# math.random(10)                -- [1, 10] integer
# math.random(1, 6)              -- [1, 6] integer
# math.randomseed(os.time())     -- 5.4: returns the seed used
```

## Strings

Strings are 8-bit clean immutable byte sequences. They are interned, so `==` is O(1) pointer equality (after a hash). The string library is reachable as both `string.foo(s, ...)` and `s:foo(...)` because every string value has the string library as its metatable.

```bash
# # Literals
# 'single quotes'
# "double quotes"
# [[long bracket — multi-line; preserves newlines verbatim]]
# [==[level-2 bracket — useful when content contains ]] ]==]

# # Length and indexing
# #s                       -- byte length (NOT codepoint count)
# string.len(s)            -- same as #s
# s:byte(1)                -- numeric byte at index 1
# string.char(65, 66)      -- "AB"

# # Concatenation
# "hello" .. " " .. "world"        -- "hello world"
# # Each .. allocates; in loops use table.concat instead.

# # Substring (1-indexed, inclusive on both ends; negatives count from end)
# s = "hello world"
# s:sub(1, 5)              -- "hello"
# s:sub(7)                 -- "world"
# s:sub(-5)                -- "world"
# s:sub(-5, -1)            -- "world"

# # Case
# ("HeLLo"):upper()        -- "HELLO"
# ("HeLLo"):lower()        -- "hello"

# # Repeat / reverse
# ("ab"):rep(3)            -- "ababab"
# ("ab"):rep(3, "-")       -- "ab-ab-ab"  (5.3+ separator)
# ("hello"):reverse()      -- "olleh"

# # format — printf-style
# string.format("%s = %d", "x", 42)        -- "x = 42"
# string.format("%.3f", math.pi)           -- "3.142"
# string.format("%05d", 42)                -- "00042"
# string.format("%q", 'He said "hi"')      -- '"He said \"hi\""' (Lua-readable)
# %d / %i  integer    %u unsigned (5.2)    %x / %X hex
# %f / %e / %g  float (decimal / scientific / shortest)
# %s string    %q quoted string    %c char from int    %%  literal %

# # Search and match (Lua patterns — see next section)
# string.find("hello", "ll")               -- 3, 4
# string.find("hello", "ll", 1, true)      -- plain text find (no patterns)
# string.match("v1.2.3", "(%d+)%.(%d+)")    -- "1", "2"
# string.gmatch("a=1; b=2", "(%w+)=(%d+)")  -- iterator yielding pairs
# for k, v in string.gmatch("a=1; b=2", "(%w+)=(%d+)") do print(k, v) end
# string.gsub("hello", "l", "L")           -- "heLLo", 2  (count is 2nd return)
# string.gsub("hello", "l", "L", 1)        -- "heLlo", 1  (max replacements)
```

## String Patterns

Lua patterns are NOT regex. They have no alternation (`|`), no `\b` word-boundary, no lookbehind/lookahead, no `{m,n}` counted repetition. They are smaller and faster.

```bash
# # Character classes (uppercase = complement)
# .       any character
# %a / %A   letter / non-letter
# %d / %D   digit / non-digit
# %l / %L   lowercase / non-lowercase
# %u / %U   uppercase / non-uppercase
# %w / %W   alphanumeric / non-alphanumeric
# %s / %S   whitespace / non-whitespace
# %p / %P   punctuation / non-punctuation
# %c / %C   control / non-control
# %x / %X   hex digit / non-hex
# %%        literal %

# # Sets
# [abc]            a, b, or c
# [a-z]            range
# [^0-9]           anything but a digit
# [%a%d_]          letter, digit, or underscore

# # Quantifiers (apply to single class — NO grouping!)
# *      0 or more  (greedy)
# +      1 or more  (greedy)
# -      0 or more  (lazy / shortest match)   ← Lua-specific; NOT subtraction
# ?      0 or 1

# # Anchors
# ^pat    match must start at string start (when at front)
# pat$    match must end at string end (when at back)
# %f[set] frontier: position where prev is NOT in set, next IS

# # Captures
# (pattern)    capture
# ()           position capture: returns the index instead of substring (5.1+)
# %0           whole match in gsub replacement
# %1 .. %9     captures 1..9 in gsub replacement; in match they're returned values

# # Examples
# # Trim
# s = s:match("^%s*(.-)%s*$")                  -- ".-" so it's lazy

# # Split on whitespace
# for w in s:gmatch("%S+") do print(w) end

# # Parse "key=value" pairs
# for k, v in s:gmatch("(%w+)=([^;]+)") do ... end

# # Replace digits with #
# s:gsub("%d", "#")

# # gsub with function
# s:gsub("%w+", function(w) return w:upper() end)

# # gsub with table
# s:gsub("%w+", {hello = "world", foo = "bar"})    -- replace if key present, else keep

# # Why no alternation? Use multiple gsub passes or lpeg (PEG library).
```

## Tables — the only data structure

A Lua table is a hybrid array + hash map. Internally it has an array part (dense integer keys 1..N) and a hash part (everything else); the implementation auto-balances. Arrays, dictionaries, sets, objects, modules, namespaces — every aggregate is a table.

```bash
# # Constructor syntax
# local t = {}                              -- empty
# local t = {10, 20, 30}                    -- array
# local t = {x = 1, y = 2}                  -- record (sugar for ["x"] = 1)
# local t = {10, 20, name = "n", [99] = 1}  -- mixed; array part is 1,2; hash for rest
# local t = {                               -- trailing comma OK
#     "first",
#     "second",
#     "third",
# }
```

## Tables as arrays

Lua arrays start at index 1. The `#` operator returns "a border": an `n` such that `t[n] ~= nil` and `t[n+1] == nil`. For sequence (no holes 1..n) tables this equals the length. For holey tables `#` may return ANY border — undefined!

```bash
# local arr = {10, 20, 30}
# arr[1]                   -- 10
# arr[3]                   -- 30
# arr[4]                   -- nil
# #arr                     -- 3

# # ipairs — stops at first nil; safe for sequences
# for i, v in ipairs(arr) do print(i, v) end

# # pairs — iterates ALL keys (including hash keys); order undefined
# for k, v in pairs(arr) do print(k, v) end

# # Holey table gotcha
# local h = {1, 2, nil, 4}
# print(#h)                 -- could be 2 OR 4! Implementation-defined.
# # Fix: track length yourself, or use table.pack/select("#", ...)

# table.pack(1, 2, nil, 4)   -- {1, 2, nil, 4, n = 4}
# select("#", 1, 2, nil, 4)  -- 4 (counts trailing nils for varargs)

# # Append / prepend / pop
# table.insert(arr, 40)            -- append: arr = {10,20,30,40}
# table.insert(arr, 1, 0)          -- insert at 1: arr = {0,10,20,30,40}
# table.remove(arr)                -- pop last
# table.remove(arr, 1)             -- shift first

# # Concat to string
# table.concat({"a","b","c"}, ", ")        -- "a, b, c"
# table.concat({"a","b","c"}, ", ", 2, 3)  -- "b, c"

# # Sort (in place)
# table.sort(arr)
# table.sort(arr, function(a, b) return a > b end)   -- descending

# # Move (5.3+) — fast slice/copy
# table.move(src, 1, 3, 1, dst)    -- src[1..3] → dst[1..3]
```

## Tables as records

```bash
# local user = {name = "Stevie", age = 42}
# user.name                        -- "Stevie"  (sugar for user["name"])
# user["name"]                     -- "Stevie"
# user.email = "x@y.z"             -- add field
# user.age = nil                   -- delete field

# # Iterate (order NOT guaranteed)
# for k, v in pairs(user) do print(k, v) end

# # Test field presence
# if user.email ~= nil then ... end           -- safest
# if user.email then ... end                  -- false-positives if value is false/0 — wait, 0 is truthy in Lua, so this is fine for non-boolean fields

# # Table as namespace / module
# local M = {}
# M.greet = function(n) return "hi " .. n end
# return M
```

## Tables as sets / multisets

Idiom: store the element as the key, value `true`. Membership test is O(1).

```bash
# # Set
# local seen = {apple = true, banana = true}
# if seen.apple then ... end
# seen.cherry = true                -- add
# seen.apple = nil                  -- remove
# # Iterate
# for k in pairs(seen) do print(k) end

# # Multiset (count occurrences)
# local count = {}
# for _, w in ipairs(words) do
#     count[w] = (count[w] or 0) + 1     -- "or 0" handles missing key
# end

# # Union / intersection
# local function union(a, b)
#     local r = {}
#     for k in pairs(a) do r[k] = true end
#     for k in pairs(b) do r[k] = true end
#     return r
# end
```

## Metatables — the magic

A metatable is a regular table whose keys (`__index`, `__add`, ...) describe behavior overrides for another table. Set with `setmetatable(t, mt)`, retrieve with `getmetatable(t)`. This single mechanism gives Lua its OOP, operator overloading, default values, and proxy patterns.

```bash
# # Default values via __index
# local defaults = {x = 0, y = 0}
# local p = setmetatable({}, {__index = defaults})
# print(p.x)                        -- 0   (looked up via metatable)
# p.x = 10
# print(p.x)                        -- 10  (own field wins; defaults untouched)

# # __index can be a function
# local store = setmetatable({}, {
#     __index = function(t, k) return "missing: " .. k end
# })
# print(store.foo)                  -- "missing: foo"

# # __newindex — intercept ASSIGNMENT to absent keys
# local readonly = setmetatable({x = 1}, {
#     __newindex = function(t, k, v) error("readonly: " .. k, 2) end
# })
# readonly.y = 2                    -- error! "readonly: y"
# # Note: __newindex only triggers on absent keys. Use rawset to bypass.

# # Operator overloading — full list
# __add  __sub  __mul  __div  __mod  __pow  __unm   (- + * / % ^ unary-)
# __idiv  __band  __bor  __bxor  __bnot  __shl  __shr   (5.3+ // & | ~ ~ << >>)
# __concat                         -- ..
# __len                            -- #
# __eq  __lt  __le                 -- == < <=  (>= and > derive automatically)
# __index  __newindex              -- table indexing
# __call                           -- t(...)  — table is callable
# __tostring                       -- tostring(t) and print(t)
# __metatable                      -- protect; getmetatable returns this value, setmetatable errors
# __gc                             -- finalizer; runs at GC time (5.2+ tables, 5.0+ userdata)
# __close                          -- 5.4: called when <close> local goes out of scope
# __pairs                          -- 5.2+: customize pairs() iteration
# __mode                           -- weak references: "k", "v", or "kv"

# # Operator overload example
# local v = setmetatable({3, 4}, {
#     __add = function(a, b) return {a[1]+b[1], a[2]+b[2]} end,
#     __tostring = function(a) return "("..a[1]..","..a[2]..")" end,
# })

# # Protect a metatable
# setmetatable(t, {__metatable = "locked"})
# getmetatable(t)                   -- "locked"   (not the actual metatable)
# setmetatable(t, {})               -- error: cannot change a protected metatable
```

## Object Orientation via Metatables

Lua has no `class` keyword. The canonical pattern uses a table as a class, sets `__index = self` so instances inherit, and uses the colon operator `:` for methods (which makes `self` an implicit first parameter).

```bash
# -- Animal "class"
# local Animal = {}
# Animal.__index = Animal           -- instances look up methods on Animal

# function Animal.new(name, sound)
#     local self = setmetatable({}, Animal)
#     self.name = name
#     self.sound = sound
#     return self
# end

# function Animal:speak()           -- colon: implicit `self`
#     print(self.name .. " says " .. self.sound)
# end

# local cat = Animal.new("Mittens", "meow")
# cat:speak()                        -- equivalent to Animal.speak(cat)

# -- Inheritance: subclass via metatable chain
# local Dog = setmetatable({}, {__index = Animal})
# Dog.__index = Dog

# function Dog.new(name)
#     local self = Animal.new(name, "woof")
#     return setmetatable(self, Dog)
# end

# function Dog:fetch()
#     print(self.name .. " fetches the ball")
# end

# local d = Dog.new("Rex")
# d:speak()                          -- inherited from Animal
# d:fetch()                          -- defined on Dog

# -- Colon vs dot
# obj:method(a, b)                   -- sugar for obj.method(obj, a, b)
# function T:m(a, b) end             -- sugar for function T.m(self, a, b) end
```

## Control Flow

```bash
# # if / elseif / else
# if x > 0 then
#     print("positive")
# elseif x < 0 then
#     print("negative")
# else
#     print("zero")
# end

# # No ternary keyword — use the and/or trick
# local label = (x >= 0) and "pos" or "neg"
# # CAUTION: fails if the "true" branch is itself falsy.
# # Broken:  local v = cond and false or "default"   -- always "default"
# # Fixed:   local v = cond and {false} or {"default"}; v = v[1]
# # Or just use if/else.

# # No switch — chain ifs, or use a dispatch table
# local handlers = {
#     [1] = function() print("one") end,
#     [2] = function() print("two") end,
# }
# (handlers[n] or function() print("other") end)()
```

## Loops

```bash
# # Numeric for: for var = start, stop [, step] do ... end
# for i = 1, 10 do print(i) end             -- 1..10 inclusive
# for i = 1, 10, 2 do print(i) end          -- 1, 3, 5, 7, 9
# for i = 10, 1, -1 do print(i) end         -- 10..1
# # In 5.4 a for loop with integer overflow STOPS instead of wrapping.

# # Generic for: works with any iterator function
# for k, v in pairs(t) do ... end           -- order undefined
# for i, v in ipairs(t) do ... end          -- order 1..n, stops at first nil
# for line in io.lines("f.txt") do ... end  -- file iteration
# for w in s:gmatch("%w+") do ... end       -- pattern iteration

# # while
# while cond do
#     ...
# end

# # repeat / until — note `until` INVERTS the condition vs do/while
# repeat
#     line = io.read()
# until line == "quit"                       -- exits when line == "quit"
# # The until expression has access to locals declared in the loop body — handy.

# # Iterator pattern (factory + state + control)
# local function range(n)
#     local i = 0
#     return function()
#         i = i + 1
#         if i <= n then return i end
#     end
# end
# for x in range(5) do print(x) end
```

## Break, Continue

Lua has `break` but **no `continue` keyword**. Use `goto` (5.2+) with a `::continue::` label inside the loop body.

```bash
# # Skip even numbers
# for i = 1, 10 do
#     if i % 2 == 0 then goto continue end
#     print(i)
#     ::continue::                   -- must be the LAST statement in the loop block
# end

# # break — exits innermost loop
# for i = 1, 100 do
#     if found then break end
# end

# # No labeled break either; emulate with flag or goto out_of_outer
# for i = 1, 10 do
#     for j = 1, 10 do
#         if cond then goto done end
#     end
# end
# ::done::
```

## Functions — multiple return values, varargs

```bash
# # Multiple return values
# function div(a, b) return a // b, a % b end
# local q, r = div(17, 5)              -- 3, 2
# local q   = div(17, 5)               -- q=3, rest discarded
# local t   = {div(17, 5)}             -- {3, 2}

# # Adjustment rule: in the MIDDLE of an expression list, only the first return is kept
# print(div(17, 5), "x")               -- 3   x        (one value used)
# print((div(17, 5)), "x")             -- 3   x        (extra parens force one)
# print(div(17, 5))                    -- 3   2        (last position keeps all)

# # Varargs ...
# function log(level, fmt, ...)
#     io.write("[" .. level .. "] " .. string.format(fmt, ...) .. "\n")
# end
# log("INFO", "user=%s id=%d", "alice", 42)

# # table.pack / unpack — handle nils correctly
# function f(...)
#     local t = table.pack(...)        -- t = {..., n = select("#", ...)}
#     for i = 1, t.n do print(t[i]) end
# end
# table.unpack(t)                       -- explode array back into values (5.2+)
# unpack(t)                             -- 5.1 name (still works in LuaJIT)

# # select — manipulate vararg list
# select("#", "a", nil, "c")           -- 3 (count, including nil)
# select(2, "a", "b", "c")             -- "b", "c" (from index 2 to end)

# # "Named arguments" via table literal
# function spawn(opts)
#     local name = opts.name or "anon"
#     local x    = opts.x or 0
# end
# spawn{name = "Alice", x = 10}        -- parens optional when single table arg
```

## Closures and Upvalues

Functions capture surrounding locals by reference. Each closure invocation can mutate the captured upvalue, which is shared across all closures that captured the same binding.

```bash
# function counter()
#     local n = 0
#     return function()                -- closes over n
#         n = n + 1
#         return n
#     end
# end
# local c = counter()
# c() c() c()                           -- 1, 2, 3

# # Two closures sharing an upvalue
# function make_pair()
#     local v
#     return function(x) v = x end,    -- setter
#            function()  return v end  -- getter
# end
# local set, get = make_pair()
# set(42); print(get())                 -- 42

# # Per-iteration vs shared closure (classic gotcha in some langs — Lua does it RIGHT)
# local fns = {}
# for i = 1, 3 do
#     fns[i] = function() return i end  -- each i is a fresh local
# end
# print(fns[1](), fns[2](), fns[3]())   -- 1   2   3   (not all 3!)

# # Tail calls — proper TCO (no stack growth)
# function loop(n)
#     if n <= 0 then return end
#     return loop(n - 1)               -- TAIL call; "return" before the call
# end
# loop(1e6)                             -- works; no stack overflow
# # Note: `return f() + 1` is NOT a tail call.
```

## Modules

A module is a file that returns a value (almost always a table). `require("name")` searches `package.path` (Lua) and `package.cpath` (C libs), loads the file once, caches the result in `package.loaded[name]`, and returns it.

```bash
# # mymod.lua
# local M = {}
# 
# function M.greet(name)
#     return "hi " .. name
# end
# 
# local function private() end          -- local: not exported
# 
# return M

# # Caller
# local mymod = require("mymod")        -- searches package.path
# mymod.greet("world")
# 
# # Search path semantics — module names are NOT file paths!
# print(package.path)
# -- ./?.lua;./?/init.lua;/usr/local/share/lua/5.4/?.lua;...
# 
# # require("foo.bar")   →  searches  ./foo/bar.lua  ./foo/bar/init.lua  ...
# # The dots are translated to the platform path separator.

# # Override search path
# package.path = "./mylibs/?.lua;" .. package.path

# # Force reload (skip cache)
# package.loaded["mymod"] = nil
# local mymod = require("mymod")

# # Old 5.1 module() function — DEPRECATED. Don't use.
# # If you see a file starting with `module(...)`, it's pre-2011 style.

# # Loading C modules
# # require("socket.core")   →  searches package.cpath for socket/core.so / .dll
# # The C lib must export luaopen_socket_core (lowercase, replaces dots with underscores).
```

## Error Handling

Lua uses `error()` to raise and `pcall`/`xpcall` to catch. Errors can be ANY value (string, table, custom object). The `level` argument to `error` lets a function blame its caller.

```bash
# # Raising
# error("something broke")              -- string error includes "file:line:" prefix
# error("plain", 0)                     -- level 0: no location prefix
# error("blame caller", 2)              -- level 2: report caller's location
# error({code = 404, msg = "not found"})-- table error — pcall gets the table back

# # Catching
# local ok, err = pcall(function()
#     error("boom")
# end)
# -- ok = false, err = "file.lua:2: boom"

# local ok, a, b = pcall(some_function, arg1, arg2)
# -- if ok, a/b are returns; if not, a is the error

# # xpcall — pcall + custom error handler (gets stack BEFORE unwind, so traceback works)
# local ok, err = xpcall(function() error("boom") end,
#                        function(e) return debug.traceback(e, 2) end)

# # assert — error-on-falsy shorthand
# local f = assert(io.open("missing"))  -- error if io.open returns nil, msg
# # equivalent to: f = io.open(...); if not f then error(msg) end

# # Catching specific error types
# local ok, err = pcall(do_thing)
# if not ok then
#     if type(err) == "table" and err.code == 404 then
#         -- handle specifically
#     else
#         error(err)                    -- rethrow
#     end
# end

# # 5.4 warn() — non-fatal warnings
# warn("@on")                           -- enable warnings (off by default)
# warn("disk almost full")
```

## Coroutines

Cooperative threads — NOT OS threads, NOT preemptive. They yield voluntarily. Useful for iterators, state machines, async control flow, generators, lazy sequences.

```bash
# # Lifecycle
# local co = coroutine.create(function(a, b)
#     print("start", a, b)
#     local x = coroutine.yield(a + b)    -- send a+b out, receive x in
#     print("resumed", x)
#     return "done"
# end)
# 
# print(coroutine.status(co))             -- "suspended"
# print(coroutine.resume(co, 1, 2))       -- start  1  2  →  true, 3 (the yielded value)
# print(coroutine.status(co))             -- "suspended" (after yield)
# print(coroutine.resume(co, 10))         -- resumed  10  →  true, "done"
# print(coroutine.status(co))             -- "dead"
# print(coroutine.resume(co))             -- false, "cannot resume dead coroutine"

# # coroutine.wrap — returns a function that resumes; errors propagate (vs pcall in resume)
# local gen = coroutine.wrap(function()
#     for i = 1, 3 do coroutine.yield(i) end
# end)
# print(gen(), gen(), gen())              -- 1   2   3

# # Use with generic for
# for v in coroutine.wrap(function()
#     for i = 1, 5 do coroutine.yield(i*i) end
# end) do
#     print(v)                            -- 1, 4, 9, 16, 25
# end

# # Producer-consumer
# function producer()
#     for line in io.lines() do coroutine.yield(line) end
# end
# function consumer(p)
#     while true do
#         local line = coroutine.resume(p)
#         if not line then break end
#         print(":: " .. line)
#     end
# end

# # API summary
# coroutine.create(f)     -> co (suspended)
# coroutine.resume(co, ...) -> true, ...   |  false, err
# coroutine.yield(...)    -> ... (values from next resume)
# coroutine.status(co)    -> "running" | "suspended" | "normal" | "dead"
# coroutine.wrap(f)       -> function (resume w/ error propagation)
# coroutine.running()     -> co, ismain   (5.2+)
# coroutine.close(co)     -> (5.4) close suspended coroutine, run __close
# coroutine.isyieldable() -> bool         (5.3+)
```

## The string library

```bash
# string.byte(s, i, j)            -- numeric byte values from i to j (default i=1, j=i)
# string.char(...)                -- string from byte values
# string.dump(f)                  -- bytecode of function f (no upvalues)
# string.find(s, pat, init, plain)-- start, end, captures...
# string.format(fmt, ...)         -- printf
# string.gmatch(s, pat)           -- iterator
# string.gsub(s, pat, repl, n)    -- substitute, returns new, count
# string.len(s)                   -- byte length (#s)
# string.lower(s) / upper(s)
# string.match(s, pat, init)      -- captures or whole match
# string.rep(s, n, sep)           -- n copies, optional separator (5.3+ for sep)
# string.reverse(s)
# string.sub(s, i, j)             -- substring (1-indexed; negatives count from end)
# string.pack / unpack / packsize -- 5.3+ binary serialization
# string.utf8.* (utf8 lib in 5.3+ is its own table — utf8.char, utf8.len, utf8.codepoint, utf8.offset)

# # All accessible via colon: s:upper(), s:find(...), etc.
```

## The table library

```bash
# table.insert(t, v)              -- append
# table.insert(t, pos, v)         -- insert at pos, shift right
# table.remove(t)                 -- remove and return last
# table.remove(t, pos)            -- remove and return; shift left
# table.concat(t, sep, i, j)      -- join (only string/number elements)
# table.sort(t, comp)             -- in-place; comp(a,b) returns true if a < b
# table.pack(...)                 -- {..., n = select("#", ...)}
# table.unpack(t, i, j)           -- explode (5.2+; in 5.1 it's the global `unpack`)
# table.move(a1, f, e, t, a2)     -- copy a1[f..e] to a2[t..t+e-f] (5.3+)
```

## The math library

```bash
# math.pi                          -- 3.14159...
# math.huge                        -- inf
# math.mininteger / maxinteger    -- 5.3+
# math.abs / ceil / floor / sqrt
# math.exp / log(x [, base])       -- log defaults to ln; second arg = base
# math.sin / cos / tan / asin / acos / atan / atan(y, x)
# math.deg(rad) / math.rad(deg)
# math.min(...) / max(...)
# math.fmod(x, y)                  -- truncated remainder (sign of x)
# math.modf(x)                     -- integer, fractional parts
# math.random()                    -- [0, 1)
# math.random(n)                   -- [1, n]
# math.random(m, n)                -- [m, n]
# math.randomseed(seed [, seed2])  -- 5.4: returns the seed used
# math.type(x)                     -- "integer" | "float" | nil (5.3+)
# math.tointeger(x)                -- exact int conversion or nil (5.3+)
# math.ult(a, b)                   -- unsigned integer less-than (5.3+)
```

## The io library

```bash
# # Default streams
# io.stdin / io.stdout / io.stderr
# io.write("hello\n")              -- to stdout, no newline added
# io.read()                        -- read line from stdin
# print("x", "y")                  -- TAB-separated, newline appended

# # Open / read modes
# local f = io.open("file.txt", "r")        -- "r" read, "w" write, "a" append,
#                                            -- "rb"/"wb"/"ab" binary, "r+"/"w+"/"a+" update
# if not f then error(err) end

# # Read formats (5.3+ drop the asterisk; 5.2 needs "*l", "*n", "*a")
# f:read("l")        -- one line WITHOUT newline (default)
# f:read("L")        -- one line WITH newline (5.3+)
# f:read("a")        -- whole file
# f:read("n")        -- a number (returns number or nil)
# f:read(8)          -- exactly 8 bytes (or less at EOF)
# f:read("l", "n")   -- multiple at once: line, then number

# # Write — accepts strings and numbers
# f:write("text", 42, "\n")

# # Iterate lines
# for line in f:lines() do ... end          -- doesn't auto-close
# for line in io.lines("f.txt") do ... end  -- auto-opens AND auto-closes

# # Seek
# f:seek("set", 0)          -- absolute
# f:seek("cur", 16)         -- relative
# f:seek("end")             -- end (returns size)

# # Buffering
# f:setvbuf("no")           -- unbuffered
# f:setvbuf("line")         -- line buffered
# f:setvbuf("full", 4096)   -- block buffered

# # Close
# f:close()                 -- forget = file lingers until GC; close explicitly!
# # In 5.4: local f <close> = io.open(...)   -- auto-closes via __close metamethod

# # Subprocess
# local p = io.popen("ls -la", "r")         -- "r" = read its stdout
# for line in p:lines() do print(line) end
# p:close()                                  -- returns true/nil + "exit"/"signal" + code (5.2+)

# # Temp files
# io.tmpfile()              -- returns a file handle, deleted at close
```

## The os library

```bash
# os.time()                                  -- current epoch seconds (or table → epoch)
# os.time({year=2024, month=1, day=15})      -- convert table to epoch
# os.date()                                  -- "Sun Apr 27 12:30:00 2024"
# os.date("%Y-%m-%d %H:%M:%S")               -- formatted (strftime-ish)
# os.date("*t")                              -- table: year, month, day, hour, min, sec, wday, yday, isdst
# os.date("!%Y-%m-%d")                       -- ! = UTC
# os.difftime(t2, t1)                        -- seconds between
# os.clock()                                 -- CPU time in seconds (process)

# os.getenv("HOME")                          -- env var or nil
# os.execute("ls")                           -- runs via system shell
# # Returns differ across versions: 5.1 returns exit code; 5.2+ returns true|nil + "exit"|"signal" + code
# os.exit(0)                                 -- terminate (or os.exit(true))
# os.exit(1, true)                           -- 5.2+: close state first

# os.remove(path)                            -- delete file
# os.rename(old, new)                        -- rename / move
# os.tmpname()                               -- string path; you must clean up. (POSIX may warn.)
# os.setlocale("en_US.UTF-8")                -- affects %a, %l etc in patterns!
# os.setlocale("C")                          -- restore portable behavior
```

## The debug library

For tooling and inspection — NOT for production logic. It can break invariants.

```bash
# debug.traceback(msg, level)         -- multi-line stack trace string
# debug.getinfo(level, what)          -- table about a stack frame
#   what flags: "n" name, "S" source, "l" current line, "u" upvalues, "f" function, "L" valid lines
# debug.getlocal(level, idx)          -- name, value of local at index
# debug.setlocal(level, idx, val)     -- mutate local
# debug.getupvalue(f, idx)            -- inspect closure upvalue
# debug.setupvalue(f, idx, val)
# debug.sethook(fn, mask, count)      -- "c" call, "r" return, "l" line; count = every N instr
# debug.gethook()
# debug.getmetatable(t) / setmetatable(t, mt)   -- bypass __metatable protection
# debug.getregistry()                  -- the C registry table

# # Common: error handler that adds traceback
# xpcall(fn, function(e) return debug.traceback(e, 2) end)
```

## Common Gotchas (broken + fixed)

```bash
# # 1) 1-indexed arrays catching C/Python brains
# # Broken:
# for i = 0, #arr - 1 do print(arr[i]) end   -- skips arr[#arr], shows nil at i=0
# # Fixed:
# for i = 1, #arr do print(arr[i]) end
# for i, v in ipairs(arr) do ... end

# # 2) Truthiness — only nil and false are falsy
# if 0 == nil then ... end                   -- never (0 is a number)
# if x then ... end                          -- TRUE for 0, "", {}
# # Fixed:
# if x ~= nil then ... end                   -- explicit nil check

# # 3) Global by default
# # Broken:
# function counter()
#     n = 0                                   -- GLOBAL!
#     return function() n = n + 1; return n end
# end
# # Fixed:
# function counter()
#     local n = 0
#     return function() n = n + 1; return n end
# end

# # 4) #t on holey tables
# # Broken:
# local t = {1, 2, nil, 4}
# print(#t)                                   -- 2 OR 4 — undefined!
# # Fixed: track length explicitly, or use table.pack / store n yourself

# # 5) pairs vs ipairs
# # Broken (expects order, but pairs is unordered):
# for i, v in pairs({"a", "b", "c"}) do print(v) end   -- order not guaranteed
# # Fixed:
# for i, v in ipairs({"a", "b", "c"}) do print(v) end

# # 6) Bare equality on tables = identity
# # Broken:
# {1,2,3} == {1,2,3}                         -- false! Different tables.
# # Fixed: deep-compare manually or via library; or define __eq metamethod.

# # 7) Coercion in concatenation
# # Broken:
# "x" .. nil                                 -- error: attempt to concatenate a nil value
# # Fixed:
# "x" .. tostring(nil)                       -- "xnil"

# # 8) and/or "ternary" trap
# # Broken:
# local v = cond and false or "default"      -- always "default" because false is falsy!
# # Fixed: use full if/else, or wrap: cond and {false} or {"default"}; v = v[1]

# # 9) Forgetting `return` from a module
# # Broken:
# -- mymod.lua:
# local M = {}
# M.x = 1
# -- (no return)
# require("mymod")                           -- returns true (default), not M
# # Fixed: end module file with `return M`

# # 10) Wrong require name (path vs module name)
# # Broken:
# require("./mymod.lua")                     -- error: module './mymod.lua' not found
# # Fixed:
# require("mymod")                           -- file is mymod.lua, name is "mymod"

# # 11) Mutating an array during iteration
# # Broken:
# for i, v in ipairs(t) do
#     if v == bad then table.remove(t, i) end -- skips next element!
# end
# # Fixed:
# for i = #t, 1, -1 do
#     if t[i] == bad then table.remove(t, i) end
# end

# # 12) Confusing `;` with significance — Lua doesn't need it; semicolons optional
# local x = 1 local y = 2                    -- works (Lua disambiguates)
# local x = 1; local y = 2                   -- clearer

# # 13) Calling string methods on numbers
# # Broken:
# (42):sub(1,1)                              -- error: attempt to index a number value
# # Fixed:
# tostring(42):sub(1,1)
```

## Embedding patterns

```bash
# # Redis EVAL — Redis 7+ ships Lua 5.1
# redis-cli EVAL 'return redis.call("INCR", KEYS[1])' 1 counter
# redis-cli EVALSHA <sha> numkeys key... arg...
# # KEYS / ARGV / redis.call / redis.pcall / redis.status_reply / redis.error_reply
# # No io, no os, no require — sandboxed.

# # OpenResty / Nginx
# # location / { content_by_lua_block { ngx.say("hi") } }
# # Phases: init_by_lua, set_by_lua, rewrite_by_lua, access_by_lua,
# #         content_by_lua, header_filter_by_lua, body_filter_by_lua, log_by_lua
# # Globals: ngx (req/resp/log/timer), ndk, cosocket TCP/UDP

# # Neovim API (Lua 5.1 via LuaJIT)
# vim.opt.number = true
# vim.keymap.set("n", "<leader>w", ":w<CR>")
# vim.api.nvim_buf_get_lines(0, 0, -1, false)        -- direct API
# vim.fn.expand("%:p")                                -- call vimscript funcs
# vim.cmd("colorscheme habamax")
# vim.notify / vim.print / vim.inspect

# # LÖVE 2D — game framework
# function love.load() end
# function love.update(dt) end
# function love.draw() love.graphics.print("hi", 100, 100) end

# # World of Warcraft / Roblox / Defold / Solar2D — all use Lua 5.1-flavored sandboxes

# # LuaSocket (TCP/UDP, HTTP via socket.http)
# local socket = require("socket")
# local server = socket.bind("*", 8080)
# local c, err = server:accept()

# # LuaFileSystem (lfs) — directory ops not in stdlib
# local lfs = require("lfs")
# for f in lfs.dir(".") do print(f) end
# lfs.attributes("file").size
# lfs.mkdir("newdir")

# # busted — RSpec-like test framework
# describe("math", function()
#     it("adds", function()
#         assert.are.equal(2, 1 + 1)
#     end)
# end)
```

## C API hint

The C API is the single biggest reason Lua exists. It's a ~250-function stack-based interface for embedding the interpreter and writing C extensions.

```bash
# # Skeleton: register a C function as Lua callable
# #include <lua.h>
# #include <lauxlib.h>
# #include <lualib.h>
#
# static int l_add(lua_State *L) {
#     lua_Number a = luaL_checknumber(L, 1);    /* arg #1 from stack */
#     lua_Number b = luaL_checknumber(L, 2);
#     lua_pushnumber(L, a + b);                  /* result onto stack */
#     return 1;                                   /* number of return values */
# }
#
# int main(void) {
#     lua_State *L = luaL_newstate();
#     luaL_openlibs(L);                           /* load standard libs */
#     lua_register(L, "add", l_add);
#     if (luaL_dofile(L, "script.lua") != LUA_OK) {
#         fprintf(stderr, "%s\n", lua_tostring(L, -1));
#     }
#     lua_close(L);
#     return 0;
# }

# # Key API families
# # State:   lua_newstate / lua_close / luaL_newstate / luaL_openlibs
# # Stack:   lua_gettop / lua_settop / lua_pushvalue / lua_remove / lua_insert
# # Push:    lua_pushnil/boolean/integer/number/string/lstring/cfunction/lightuserdata
# # Get:     lua_isXXX / lua_toXXX / lua_tonumberx / luaL_checkXXX (errors if wrong type)
# # Tables:  lua_createtable / lua_settable / lua_gettable / lua_setfield / lua_getfield
# # Call:    lua_call (no protection) / lua_pcall (catches) / lua_callk (continuations)
# # Refs:    luaL_ref / luaL_unref — store Lua values in C across boundaries
# # Module:  luaL_setfuncs(L, reg, 0); /* reg is a {const char*, lua_CFunction} array */

# # Build a C module (Linux):
# # cc -O2 -fPIC -shared -I/usr/include/lua5.4 mymod.c -o mymod.so

# # Then in Lua:  local m = require("mymod")  -- looks up package.cpath
# # Entry point must be named luaopen_<modulename>
```

## LuaJIT specifics

LuaJIT tracks Lua 5.1 syntax + selected 5.2/5.3 extensions. It is several times faster than PUC-Rio and adds a phenomenal FFI for direct C interop without writing C.

```bash
# # FFI — call C from Lua, no glue
# local ffi = require("ffi")
# ffi.cdef[[
#     int printf(const char *fmt, ...);
#     typedef struct { double x, y; } point_t;
# ]]
# ffi.C.printf("hello %d\n", 42)
# local p = ffi.new("point_t", 1.0, 2.0)
# print(p.x, p.y)

# # Loading shared libs
# local zlib = ffi.load("z")     -- libz.so / z.dll
# ffi.cdef[[
#     unsigned long compressBound(unsigned long sourceLen);
# ]]
# zlib.compressBound(1024)

# # JIT control
# jit.on() / jit.off()
# jit.flush()                    -- drop all compiled traces
# jit.status()                   -- on/off, flags
# jit.opt.start(...)             -- tweak optimizer
# require("jit.dump").on()       -- print traces (or "jit.v" for terse)

# # NYI (Not Yet Implemented) — operations that fall back to interpreter
# # Common offenders: pcall/xpcall (in some versions), unbalanced returns,
# # string.dump, debug.*, coroutine.* in some hot paths.
# # Run with -jdump=+rs to find NYI sites.

# # Compatibility flags (build time)
# # -DLUAJIT_ENABLE_LUA52COMPAT  → goto, ::label::, "\z" escape, integer-style division
# # Most distros enable this.
```

## Common Error Messages

Exact text and what to check. The line/column prefix is `chunk:line:` (e.g., `script.lua:7:`).

```bash
# # "attempt to index a nil value (global 'x')"
# # → You wrote x.foo or x[k] but x is nil. Check spelling, require return value, scope.
# # Variations: "(local 'x')"  "(field 'name')"  "(upvalue 'x')"

# # "attempt to call a nil value (global 'f')"
# # → f() but f is nil. Module didn't export f, or you called .name not :name and f is method.

# # "attempt to call a XXX value"  (XXX = string / number / table)
# # → You called something that isn't a function. e.g. forgot to return module table.

# # "attempt to perform arithmetic on a string value"
# # → "5" + 1 in 5.3+ may coerce; "abc" + 1 won't. Use tonumber("5") + 1.
# # In 5.4 string-arith coercion was REMOVED — always tonumber explicitly.

# # "attempt to concatenate a nil value"
# # → ".." with a nil operand. Use tostring(x) or guard.

# # "attempt to compare two XXX values"  /  "attempt to compare XXX with YYY"
# # → < <= > >= require same type (or __lt metamethod). nil < 1 errors.

# # "stack overflow"
# # → Infinite recursion. The default C stack is ~200 levels for protected calls.
# # Look for missing base case; consider iteration or tail calls (return f()).

# # "bad argument #2 to 'insert' (number expected, got string)"
# # → Arg type mismatch in stdlib or luaL_check function. Trace the call site.

# # "module 'X' not found:"  (followed by all paths searched)
# # → require failed. Module file not in package.path / cpath. Add to LUA_PATH /
# # set package.path. Remember dots become slashes.

# # "loop in gettable" / "C stack overflow"
# # → __index metamethod recursion (a metatable that points back to itself).

# # "table index is NaN"  /  "table index is nil"
# # → Cannot use NaN or nil as a table key. (false IS allowed.)

# # "'<eof>' expected near 'X'"
# # → Syntax error: stray token. Check unmatched 'end', missing 'then'/'do'.

# # "'end' expected (to close 'function' at line N) near '<eof>'"
# # → Forgot an end. Lua does NOT use indentation; all blocks need explicit end.

# # "'=' expected near 'X'"
# # → Forgot = in a local declaration:  local x 5  →  local x = 5

# # "ambiguous syntax (function call x new statement) near '('"
# # → Two statements run together; previous line returned a function. Add `;` or split.
# #   Example:  local f = g
# #             (h or i)()              -- parsed as g(h or i)()
# # Fix:        local f = g;
# #             (h or i)()
```

## Performance Tips

```bash
# # 1) Locals are stack slots; globals are hash lookups.
# local insert = table.insert
# for i = 1, 1e6 do insert(t, i) end          -- faster than table.insert(t, i)

# # 2) Cache frequently used library functions
# local sqrt, sin, cos = math.sqrt, math.sin, math.cos

# # 3) Avoid string concat in loops — quadratic cost
# # Broken:
# local s = ""
# for i = 1, 1e4 do s = s .. tostring(i) end  -- O(n^2)
# # Fixed:
# local parts = {}
# for i = 1, 1e4 do parts[#parts+1] = tostring(i) end
# local s = table.concat(parts)

# # 4) Pre-size tables when known (PUC-Rio: no API; LuaJIT has table.new)
# local table_new = require("table.new")        -- LuaJIT only
# local t = table_new(narray, nhash)

# # 5) Avoid table indexing in hot loops
# # Broken:
# for i = 1, n do f(self.field[i]) end
# # Fixed:
# local field = self.field
# for i = 1, n do f(field[i]) end

# # 6) Numeric for is fastest; ipairs faster than pairs.

# # 7) `t[#t+1] = x` is usually faster than table.insert(t, x).

# # 8) Avoid creating closures in hot loops; reuse outside.

# # 9) GC tuning
# collectgarbage("count")              -- KB used
# collectgarbage("collect")            -- full GC
# collectgarbage("stop") / "restart"   -- pause / resume
# collectgarbage("incremental")        -- 5.4 modes: incremental | generational
# collectgarbage("setpause", 100)
# collectgarbage("setstepmul", 200)

# # 10) On LuaJIT, watch for NYI bytecode falling back to interpreter (use -jv or -jdump).
```

## Idioms

```bash
# # Default arg
# function greet(name)
#     name = name or "stranger"
#     return "hi " .. name
# end

# # Optional table arg
# function spawn(opts)
#     opts = opts or {}
#     local x = opts.x or 0
# end

# # Swap (multi-assignment)
# a, b = b, a

# # Multiple-return chaining
# return next(t)                       -- forwards both key and value

# # Nil-safe field access (no operator; use 'and' chain)
# local v = a and a.b and a.b.c        -- nil if any link missing

# # Singleton via require cache
# -- singleton.lua: returns table; require returns same table everywhere

# # Lazy initialization
# local _data
# local function data()
#     if not _data then _data = expensive() end
#     return _data
# end

# # Counting with default-zero
# count[k] = (count[k] or 0) + 1

# # Build-a-string
# local buf = {}
# buf[#buf+1] = "line1"
# buf[#buf+1] = "line2"
# return table.concat(buf, "\n")

# # Reverse iterate an array
# for i = #t, 1, -1 do ... end

# # Inline if (ternary surrogate)
# local v = (cond and a) or b           -- be sure a is truthy

# # Variadic forwarding
# function wrapper(...) return inner(...) end
```

## Modules and Packaging

```bash
# # luarocks basics
# luarocks search lua-cjson
# luarocks install lua-cjson
# luarocks install --local lua-cjson         -- ~/.luarocks  (no sudo)
# luarocks list
# luarocks remove lua-cjson
# luarocks --lua-version=5.1 install ...     -- target a specific Lua
# luarocks make path/to/rockspec             -- install from local rockspec
# luarocks build path/to/rockspec
# luarocks pack mymod                         -- create .src.rock
# luarocks upload mymod-1.0-0.rockspec       -- to luarocks.org (needs api key)

# # rockspec skeleton — mymod-1.0-0.rockspec
# package = "mymod"
# version = "1.0-0"                            -- semver-MAJOR.MINOR-REVISION
# source = {
#     url = "git+https://github.com/me/mymod",
#     tag = "v1.0",
# }
# description = {
#     summary = "...",
#     license = "MIT",
# }
# dependencies = {
#     "lua >= 5.1",                            -- or "lua >= 5.3, < 5.5"
#     "lpeg ~> 1.0",                           -- pessimistic
# }
# build = {
#     type = "builtin",                        -- builtin | make | cmake | command
#     modules = {
#         ["mymod"]      = "src/mymod.lua",
#         ["mymod.util"] = "src/mymod/util.lua",
#         ["mymod.core"] = {                   -- C module
#             sources = {"src/core.c"},
#             libraries = {"z"},
#         },
#     },
# }

# # Activate rocks tree on the fly (paths to LUA_PATH/LUA_CPATH)
# eval "$(luarocks path)"
# # or
# eval "$(luarocks --local path)"
```

## Testing

```bash
# # busted (luarocks install busted)
# # spec/calc_spec.lua
# describe("calc", function()
#     local calc = require("calc")
#     it("adds", function()
#         assert.are.equal(3, calc.add(1, 2))
#     end)
#     it("rejects non-numbers", function()
#         assert.has_error(function() calc.add("a", 1) end)
#     end)
#     before_each(function() ... end)
#     after_each(function()  ... end)
# end)
# 
# # CLI
# busted                                       -- runs ./spec/*_spec.lua
# busted -o utfTerminal                         -- pretty output
# busted --coverage                              -- with luacov
# busted -p '_test'                              -- different pattern
# busted --tags=fast                              -- filter by tag

# # luaunit alternative — single-file, simpler
# # local lu = require("luaunit")
# # function TestStuff:testAdd() lu.assertEquals(2, 1+1) end
# # os.exit(lu.LuaUnit.run())

# # luacov — coverage
# luarocks install luacov
# lua -lluacov myscript.lua          -- runs and writes luacov.stats.out
# luacov                              -- generate luacov.report.out

# # CI
# # GitHub Actions: leafo/gh-actions-lua + leafo/gh-actions-luarocks
```

## Tools

```bash
# lua          # interpreter (PUC-Rio)            man lua
# luac         # bytecode compiler                man luac
# luajit       # JIT interpreter (5.1 dialect)
# luarocks     # package manager                  luarocks help <cmd>

# # Linter
# luacheck script.lua
# luacheck . --no-self            # don't warn on missing self in methods
# luacheck . --globals foo bar    # whitelist additional globals
# # Config: .luacheckrc — std = "lua54", globals, ignore, codes ...

# # Formatter
# stylua .                          # in-place format (Rust-based, fast)
# stylua --check .                  # CI mode
# # Config: stylua.toml — column_width, indent_type (Spaces/Tabs), call_parentheses ...

# # LSP
# lua-language-server               # github.com/LuaLS/lua-language-server
# # Diagnostics, completion, hover, goto-def. Strongly recommended in editors.
# # Settings (.luarc.json): runtime.version "Lua 5.4", workspace.library, diagnostics.globals

# # Profiler — LuaJIT
# luajit -jp=fl3 script.lua          # function/line/3-char view of CPU time
# luajit -jp=v script.lua            # verbose
# luajit -jdump=Trs script.lua       # IR/asm dump

# # Profiler — PUC-Rio
# # Use lua-profile, ProFi, or hand-roll via debug.sethook("c", ...)

# # REPL upgrade
# # luarocks install ilua            -- nicer REPL with auto-print, tab complete
# # luarocks install lua-repl
```

## Tips

- Write `local` everywhere — it's faster, scoped, and shields you from globals collisions.
- The colon `obj:method(...)` is sugar for `obj.method(obj, ...)`. When defining, `function T:m(...)` is sugar for `function T.m(self, ...)`. Use it consistently.
- Tables are 1-indexed by convention; do not invent 0-indexed schemes — `ipairs` and `#` will betray you.
- Only `nil` and `false` are falsy. `0`, `""`, `{}` are truthy. Always check `~= nil` for nil-specifically.
- Lua patterns are not regex. They have no `|`, no `\b`, no `(?:...)`. Use `lpeg` if you need real grammars.
- Modules return a table; `require` caches it. To reload, set `package.loaded["mod"] = nil` then `require` again.
- Errors are values — they can be tables. Use `pcall`/`xpcall` to catch; `xpcall` keeps the stack live.
- Coroutines are cooperative — they yield, never preempt. They model iterators and async control beautifully.
- LuaJIT is a different deployment target than PUC-Rio. Test on both if you target Redis / OpenResty / Neovim.
- Tail calls do NOT grow the C stack: `return f(x)`. They DO require `return` first.
- The `#` operator on holey tables is undefined — store your own `n` if nils may appear.
- Default to `local` upvalues over closures-over-globals; faster, and immune to environment changes.
- Use `string.format` over `..` chains for anything formatted; readable and avoids accidental coercion.
- 5.4 introduced `<close>` for RAII; pair with `__close` metamethod and forget about manual `f:close()`.

## See Also

- polyglot, c, rust, go, python, javascript, typescript, java, ruby, make, webassembly, bash, regex

## References

- [Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/) -- official language and library reference
- [Lua 5.3 Reference Manual](https://www.lua.org/manual/5.3/) -- still common in embedded contexts
- [Lua 5.1 Reference Manual](https://www.lua.org/manual/5.1/) -- LuaJIT, Redis, OpenResty target
- [Programming in Lua (PiL)](https://www.lua.org/pil/) -- Roberto Ierusalimschy's authoritative book
- [Lua Source Code](https://www.lua.org/source/5.4/) -- ~25k lines of clean, readable C
- [LuaJIT](https://luajit.org/) -- high-performance JIT for Lua 5.1
- [LuaJIT FFI Tutorial](https://luajit.org/ext_ffi_tutorial.html) -- calling C from Lua
- [LuaJIT NYI list](https://wiki.luajit.org/NYI) -- operations not yet jit-compiled
- [LuaRocks](https://luarocks.org/) -- module repository and package manager
- [LuaRocks rockspec format](https://github.com/luarocks/luarocks/wiki/Rockspec-format) -- packaging reference
- [lua-language-server](https://github.com/LuaLS/lua-language-server) -- canonical LSP implementation
- [stylua](https://github.com/JohnnyMorganz/StyLua) -- opinionated formatter (Rust)
- [luacheck](https://github.com/lunarmodules/luacheck) -- static linter
- [busted](https://lunarmodules.github.io/busted/) -- BDD-style testing
- [LuaUnit](https://github.com/bluebird75/luaunit) -- xUnit-style testing
- [luacov](https://lunarmodules.github.io/luacov/) -- coverage analyzer
- [LPeg](http://www.inf.puc-rio.br/~roberto/lpeg/) -- pattern-matching via PEGs (alternative to patterns)
- [OpenResty](https://openresty.org/) -- Nginx + LuaJIT web platform
- [Neovim Lua guide](https://neovim.io/doc/user/lua-guide.html) -- editor integration
- [Redis EVAL](https://redis.io/commands/eval/) -- scripting commands
- [Lua-users wiki](http://lua-users.org/wiki/) -- community recipes and idioms
- [Lua Style Guide](http://lua-users.org/wiki/LuaStyleGuide) -- community conventions
- [Lua mailing list](https://www.lua.org/lua-l.html) -- official discussion
- [LÖVE 2D](https://love2d.org/) -- 2D game framework
- [LuaSocket](https://lunarmodules.github.io/luasocket/) -- networking
- [LuaFileSystem](https://lunarmodules.github.io/luafilesystem/) -- directory operations
- [lua-cjson](https://github.com/openresty/lua-cjson) -- fast JSON
- [lpeg patterns](https://www.inf.puc-rio.br/~roberto/lpeg/) -- modern alternative
