# The Internals of Neovim — Architecture, Lua API, and LSP

> *Neovim is a refactored fork of Vim with a client-server architecture (MessagePack RPC), first-class Lua scripting (LuaJIT), built-in LSP client, Tree-sitter integration for syntax-aware editing, and an asynchronous event loop (libuv). It separates the core editor from the UI, enabling multiple frontends to control one Neovim instance.*

---

## 1. Architecture — Separation of Concerns

### Vim vs Neovim Architecture

```
Vim:
┌────────────────────────────┐
│  Terminal UI + Editor Core  │  (monolithic)
└────────────────────────────┘

Neovim:
┌──────────┐  MsgPack-RPC  ┌──────────────┐
│ UI Client│ ◄────────────► │  nvim core   │
│ (TUI,GUI)│                │  (headless)  │
└──────────┘                └──────┬───────┘
                                   │ libuv
                              ┌────┴────┐
                              │ Event   │
                              │  Loop   │
                              └─────────┘
```

### MessagePack-RPC API

Every Neovim operation is exposed via a typed RPC API:

```
nvim_buf_set_lines(buffer, start, end, strict, lines)
nvim_win_set_cursor(window, [row, col])
nvim_exec_lua(code, args)
nvim_create_autocmd(event, opts)
```

This enables:
- **Remote plugins** (any language with MsgPack support)
- **Multiple UIs** (terminal, Qt, Electron, web)
- **Embedding** (Neovim as a library in other applications)
- **Testing** (programmatic control via RPC)

### Event Loop (libuv)

Neovim uses **libuv** (same as Node.js) for async I/O:

```
┌─────── Event Loop ───────┐
│  Timer callbacks         │
│  I/O callbacks (files)   │
│  RPC message handling    │
│  Job/process management  │
│  Signal handling         │
└──────────────────────────┘
```

All I/O is non-blocking. This is why Neovim can run LSP servers, linters, and formatters without freezing the UI.

---

## 2. Lua Integration — First-Class Scripting

### Lua vs Vimscript

| Feature | Vimscript | Lua (LuaJIT) |
|:--------|:----------|:-------------|
| Speed | Interpreted, slow | JIT-compiled, ~10-100x faster |
| Type system | Dynamic, weak | Dynamic, stronger |
| Data structures | Lists, Dicts | Tables (unified) |
| Error handling | `try/catch` (limited) | `pcall`, `xpcall` |
| Ecosystem | Vim plugins | LuaRocks + Vim plugins |
| API access | Direct | `vim.api.nvim_*` |

### The `vim` Global Object

```lua
-- API functions
vim.api.nvim_buf_set_lines(0, 0, -1, false, {"hello", "world"})

-- Options
vim.opt.number = true
vim.opt.shiftwidth = 4
vim.opt.completeopt = {"menu", "noselect"}

-- Key mappings
vim.keymap.set('n', '<leader>f', function()
    vim.lsp.buf.format()
end, { desc = "Format buffer" })

-- Autocommands
vim.api.nvim_create_autocmd("BufWritePre", {
    pattern = "*.go",
    callback = function()
        vim.lsp.buf.format()
    end,
})
```

### Module System

```
~/.config/nvim/
├── init.lua                    # entry point
├── lua/
│   ├── config/
│   │   ├── options.lua         # vim.opt settings
│   │   ├── keymaps.lua         # vim.keymap.set
│   │   └── autocmds.lua        # autocommands
│   └── plugins/
│       ├── lsp.lua             # LSP configuration
│       └── telescope.lua       # plugin config
```

`require("config.options")` loads `lua/config/options.lua`.

### Performance: LuaJIT vs Vimscript

LuaJIT achieves near-C performance for computational tasks:

| Operation | Vimscript | Lua | Speedup |
|:----------|:----------|:----|:--------|
| Loop 1M iterations | ~500ms | ~5ms | 100x |
| String manipulation | ~200ms | ~10ms | 20x |
| Table/dict operations | ~300ms | ~15ms | 20x |
| Regex matching | ~100ms | ~50ms | 2x |

---

## 3. Tree-sitter Integration

### What Tree-sitter Provides

Tree-sitter is an **incremental parsing** library that builds and maintains a concrete syntax tree (CST) for source code:

```
Traditional regex highlighting:
  Line-by-line → misidentifies multi-line strings, nested structures

Tree-sitter:
  Full syntax tree → correct highlighting of every token
```

### Incremental Parsing

When you edit a file, Tree-sitter re-parses only the affected region:

$$\text{Re-parse cost} = O(\text{edit size} + \log n)$$

Not $O(n)$ for the entire file. This enables real-time syntax tree updates.

### Tree-sitter Query Language

Highlights, indentation, and text objects are defined via **S-expression queries**:

```scheme
;; Highlight function names
(function_declaration
  name: (identifier) @function)

;; Highlight string literals
(string) @string

;; Highlight comments
(comment) @comment
```

### Tree-sitter Powered Features

| Feature | Traditional | Tree-sitter |
|:--------|:-----------|:------------|
| Syntax highlighting | Regex patterns | AST node types |
| Indentation | Heuristic rules | Grammar-based |
| Code folding | Marker/indent based | Syntax node folding |
| Text objects | Pattern matching | `@function.outer`, `@class.inner` |
| Selection | Manual | Incremental node selection |

---

## 4. Built-in LSP Client

### Language Server Protocol Architecture

```
Neovim ◄──── JSON-RPC ────► Language Server
(client)     (stdio/TCP)     (gopls, pyright, rust-analyzer, ...)
```

### LSP Message Types

| Type | Direction | Example |
|:-----|:----------|:--------|
| Request | Client → Server | `textDocument/completion` |
| Response | Server → Client | Completion items |
| Notification | Either direction | `textDocument/didOpen` |

### Key LSP Capabilities

| Capability | Method | Description |
|:-----------|:-------|:-----------|
| Completion | `textDocument/completion` | Code completion |
| Hover | `textDocument/hover` | Type info, documentation |
| Go to definition | `textDocument/definition` | Jump to definition |
| Find references | `textDocument/references` | All usages |
| Rename | `textDocument/rename` | Symbol rename |
| Code action | `textDocument/codeAction` | Quick fixes, refactors |
| Diagnostics | `textDocument/publishDiagnostics` | Errors, warnings |
| Formatting | `textDocument/formatting` | Code formatting |

### LSP Configuration

```lua
vim.lsp.start({
    name = "gopls",
    cmd = {"gopls"},
    root_dir = vim.fs.dirname(vim.fs.find({"go.mod"}, {upward = true})[1]),
    capabilities = vim.lsp.protocol.make_client_capabilities(),
})
```

---

## 5. Plugin Manager Architecture

### Lazy Loading

Modern plugin managers (lazy.nvim) defer loading until needed:

| Trigger | Description | Example |
|:--------|:-----------|:--------|
| `event` | Autocommand event | `BufReadPost` |
| `cmd` | Ex command | `:Telescope` |
| `ft` | File type | `go`, `python` |
| `keys` | Key mapping | `<leader>f` |
| `module` | Lua require | `require("telescope")` |

### Startup Time Impact

| Loading Strategy | Plugins | Startup Time |
|:----------------|:--------|:-------------|
| Eager (all at start) | 50 | ~300-500ms |
| Lazy (on demand) | 50 | ~30-80ms |
| Minimal (essentials only) | 10 | ~15-30ms |

---

## 6. Async Job Control

### Vim's Problem: Synchronous External Commands

```vim
:!make          " Vim freezes until make completes
```

### Neovim's Solution: Async Jobs

```lua
vim.fn.jobstart({"make", "build"}, {
    stdout_buffered = true,
    on_stdout = function(_, data)
        -- process output lines
    end,
    on_exit = function(_, exit_code)
        -- handle completion
    end,
})
```

The event loop handles job I/O alongside user input — no freezing.

### Channels

| Channel Type | Transport | Use Case |
|:-------------|:----------|:---------|
| Job | stdin/stdout | External processes |
| TCP | Socket | Remote language servers |
| Stdio | stdin/stdout of nvim | Embedding |

---

## 7. Diagnostic Framework

### Unified Diagnostics

Neovim provides a single diagnostic API that multiple sources feed into:

```
LSP server ─────┐
Linter (lint) ──┤──► vim.diagnostic ──► Signs, virtual text,
Compiler ───────┘                       float window, quickfix
```

### Diagnostic Severity Levels

| Level | Number | Default Sign |
|:------|:------:|:-------------|
| Error | 1 | `E` (red) |
| Warning | 2 | `W` (yellow) |
| Information | 3 | `I` (blue) |
| Hint | 4 | `H` (green) |

---

## 8. Summary of Key Differences from Vim

| Feature | Vim | Neovim |
|:--------|:----|:-------|
| Architecture | Monolithic | Client-server (MsgPack-RPC) |
| Scripting | Vimscript (+ Lua 5.1) | Lua first-class (LuaJIT) |
| Async I/O | Limited (`job_start`) | libuv event loop |
| LSP | Plugin required | Built-in client |
| Tree-sitter | Not available | Built-in integration |
| Defaults | Minimal | Sensible defaults out of box |
| Terminal | `:terminal` (basic) | Full terminal emulator |
| UI | Built-in TUI only | Pluggable (any UI via RPC) |
| Diagnostics | Plugin-dependent | Built-in `vim.diagnostic` |

---

*Neovim's core insight is separation: separate the editor from the UI, separate configuration from implementation (Lua over Vimscript), separate syntax understanding from regex heuristics (Tree-sitter), and separate language intelligence from the editor (LSP). Each separation makes the system more composable, testable, and extensible.*
