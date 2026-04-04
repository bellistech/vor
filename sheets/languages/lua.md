# Lua (Lightweight Scripting Language)

> Fast, embeddable scripting language with simple syntax, first-class functions, and tables as the universal data structure.

## Basics

### Variables and Types

```lua
-- Variables (global by default)
x = 10
local y = 20             -- Local variable (preferred)

-- Types
type(42)                  -- "number"
type("hello")             -- "string"
type(true)                -- "boolean"
type(nil)                 -- "nil"
type({})                  -- "table"
type(print)               -- "function"

-- Multiple assignment
local a, b, c = 1, 2, 3
a, b = b, a               -- Swap values

-- String concatenation
local s = "hello" .. " " .. "world"
local n = #s              -- String length: 11
```

## Tables

### Arrays (1-Indexed)

```lua
local arr = {10, 20, 30, 40, 50}
print(arr[1])             -- 10 (1-indexed!)
print(#arr)               -- 5 (length operator)

-- Iterate array
for i, v in ipairs(arr) do
    print(i, v)
end

-- Append
table.insert(arr, 60)           -- Append to end
table.insert(arr, 1, 0)         -- Insert at position 1

-- Remove
table.remove(arr, 3)            -- Remove element at index 3
table.remove(arr)               -- Remove last element

-- Sort
table.sort(arr)
table.sort(arr, function(a, b) return a > b end)  -- Descending
```

### Dictionaries (Hash Maps)

```lua
local config = {
    host = "localhost",
    port = 8080,
    debug = true,
}

print(config.host)            -- "localhost"
print(config["port"])         -- 8080

-- Iterate all key-value pairs
for k, v in pairs(config) do
    print(k, v)
end

-- Delete a key
config.debug = nil
```

### Metatables

```lua
local Vector = {}
Vector.__index = Vector

function Vector.new(x, y)
    return setmetatable({x = x, y = y}, Vector)
end

function Vector:length()
    return math.sqrt(self.x^2 + self.y^2)
end

-- Operator overloading
function Vector.__add(a, b)
    return Vector.new(a.x + b.x, a.y + b.y)
end

function Vector.__tostring(v)
    return string.format("(%g, %g)", v.x, v.y)
end

local v1 = Vector.new(3, 4)
print(v1:length())            -- 5
local v2 = v1 + Vector.new(1, 2)
```

### Metamethod Reference

```lua
__index       -- Lookup missing keys (table or function)
__newindex     -- Intercept new key assignment
__add          -- + operator
__sub          -- - operator
__mul          -- * operator
__div          -- / operator
__mod          -- % operator
__pow          -- ^ operator
__concat       -- .. operator
__eq           -- == operator
__lt           -- < operator
__le           -- <= operator
__len          -- # operator
__tostring     -- tostring() conversion
__call         -- Call table as function
__gc           -- Garbage collection finalizer
```

## Functions

### Basic Functions

```lua
-- Function declaration
function greet(name)
    return "Hello, " .. name
end

-- Local function
local function add(a, b)
    return a + b
end

-- Anonymous function
local square = function(x) return x * x end

-- Multiple return values
function divmod(a, b)
    return math.floor(a / b), a % b
end
local q, r = divmod(17, 5)    -- 3, 2
```

### Closures

```lua
function counter(start)
    local count = start or 0
    return function()
        count = count + 1
        return count
    end
end

local next = counter(10)
print(next())    -- 11
print(next())    -- 12
```

### Varargs

```lua
function printf(fmt, ...)
    io.write(string.format(fmt, ...))
end

function sum(...)
    local total = 0
    for _, v in ipairs({...}) do
        total = total + v
    end
    return total
end

print(sum(1, 2, 3, 4))   -- 10
```

## Coroutines

### Create, Resume, Yield

```lua
local co = coroutine.create(function(x)
    print("start:", x)
    local y = coroutine.yield(x * 2)     -- Suspend, return x*2
    print("resumed:", y)
    return x + y
end)

local ok, val = coroutine.resume(co, 5)   -- start: 5, val = 10
print(val)                                  -- 10
ok, val = coroutine.resume(co, 20)          -- resumed: 20, val = 25
print(val)                                  -- 25
print(coroutine.status(co))                 -- "dead"
```

### Iterator with Coroutines

```lua
function range(n)
    return coroutine.wrap(function()
        for i = 1, n do
            coroutine.yield(i)
        end
    end)
end

for v in range(5) do
    print(v)    -- 1, 2, 3, 4, 5
end
```

## String Patterns

```lua
-- Lua patterns (NOT regex — simpler)
.       -- Any character
%a      -- Letters          %A  non-letters
%d      -- Digits           %D  non-digits
%l      -- Lowercase        %L  non-lowercase
%u      -- Uppercase        %U  non-uppercase
%w      -- Alphanumeric     %W  non-alphanumeric
%s      -- Whitespace       %S  non-whitespace
%p      -- Punctuation      %P  non-punctuation
%%      -- Literal %

-- Quantifiers
*       -- 0 or more (greedy)
+       -- 1 or more (greedy)
-       -- 0 or more (lazy)
?       -- 0 or 1

-- String functions
string.find("hello world", "world")         -- 7, 11
string.match("2024-01-15", "(%d+)-(%d+)-(%d+)")  -- "2024", "01", "15"
string.gsub("hello", "l", "L")             -- "heLLo", 2
string.gmatch("one two three", "%S+")       -- Iterator over words
string.format("%.2f", 3.14159)              -- "3.14"
```

## Modules

### Creating a Module

```lua
-- mymodule.lua
local M = {}

function M.greet(name)
    return "Hello, " .. name
end

local function private_helper()
    -- Not exported
end

return M
```

### Using Modules

```lua
local mymod = require("mymodule")
print(mymod.greet("World"))

-- package.path controls search locations
print(package.path)
-- ./?.lua;./?/init.lua;/usr/share/lua/5.4/?.lua;...
```

## C API Basics

```c
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>

// C function callable from Lua
static int l_add(lua_State *L) {
    double a = luaL_checknumber(L, 1);
    double b = luaL_checknumber(L, 2);
    lua_pushnumber(L, a + b);
    return 1;  // Number of return values
}

// Register and run
int main(void) {
    lua_State *L = luaL_newstate();
    luaL_openlibs(L);
    lua_register(L, "add", l_add);
    luaL_dofile(L, "script.lua");
    lua_close(L);
    return 0;
}
```

## Common Use Cases

```lua
-- Neovim configuration (init.lua)
vim.opt.number = true
vim.keymap.set('n', '<leader>w', ':w<CR>')

-- Nginx/OpenResty
-- access_by_lua_block { ngx.say("Hello") }

-- Game engines (LOVE2D)
function love.draw()
    love.graphics.print("Hello", 400, 300)
end
```

## Tips

- Tables are the only data structure; arrays, dicts, objects, and modules are all tables.
- Lua arrays are 1-indexed by convention; the `#` operator and `ipairs` rely on this.
- Use `local` for all variables; globals pollute the environment and are slower.
- The colon syntax `obj:method(args)` is sugar for `obj.method(obj, args)`.
- LuaJIT is dramatically faster than PUC Lua and supports FFI for calling C directly.
- Lua patterns are not regular expressions; they lack alternation (`|`) and many PCRE features.
- `nil` removes a key from a table; check existence with `if value ~= nil then`.

## See Also

- c, python, neovim, ruby, javascript, regex

## References

- [Lua 5.4 Reference Manual](https://www.lua.org/manual/5.4/) -- official language and library reference
- [Programming in Lua (PiL)](https://www.lua.org/pil/) -- authoritative book (first edition free online)
- [Lua 5.4 Source Code](https://www.lua.org/source/5.4/) -- annotated C source of the interpreter
- [LuaJIT](https://luajit.org/) -- high-performance JIT compiler for Lua 5.1
- [LuaJIT FFI Tutorial](https://luajit.org/ext_ffi_tutorial.html) -- calling C from Lua via FFI
- [LuaRocks](https://luarocks.org/) -- package manager and module repository
- [Lua Users Wiki](http://lua-users.org/wiki/) -- community tutorials, patterns, and recipes
- [Neovim Lua Guide](https://neovim.io/doc/user/lua-guide.html) -- Lua integration in Neovim
- [Lua Style Guide](http://lua-users.org/wiki/LuaStyleGuide) -- community conventions
- [Lua mailing list archive](https://www.lua.org/lua-l.html) -- official discussion list
