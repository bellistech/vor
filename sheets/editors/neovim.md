# Neovim (nvim)

> Hyperextensible Vim fork with Lua-first configuration, built-in LSP, and Treesitter.

## Lua Configuration

### init.lua Basics

```bash
# config location
~/.config/nvim/init.lua                    # main entry point
~/.config/nvim/lua/                        # lua modules
~/.config/nvim/lua/plugins/                # plugin configs
~/.config/nvim/after/plugin/               # loaded after plugins
```

### Settings in Lua

```lua
-- ~/.config/nvim/init.lua
vim.g.mapleader = " "                     -- space as leader key
vim.g.maplocalleader = " "

-- options (equivalent to :set)
vim.opt.number = true                     -- line numbers
vim.opt.relativenumber = true             -- relative numbers
vim.opt.tabstop = 4
vim.opt.shiftwidth = 4
vim.opt.expandtab = true                  -- spaces, not tabs
vim.opt.smartindent = true
vim.opt.wrap = false
vim.opt.signcolumn = "yes"
vim.opt.scrolloff = 8
vim.opt.updatetime = 250
vim.opt.termguicolors = true
vim.opt.clipboard = "unnamedplus"         -- system clipboard
vim.opt.ignorecase = true
vim.opt.smartcase = true                  -- case-sensitive if uppercase used
vim.opt.undofile = true                   -- persistent undo
vim.opt.splitright = true
vim.opt.splitbelow = true
vim.opt.cursorline = true
```

### Keymaps in Lua

```lua
local keymap = vim.keymap.set

-- normal mode
keymap("n", "<leader>w", ":w<CR>", { desc = "Save file" })
keymap("n", "<leader>q", ":q<CR>", { desc = "Quit" })
keymap("n", "<Esc>", ":noh<CR>", { desc = "Clear search highlight" })

-- better window navigation
keymap("n", "<C-h>", "<C-w>h")
keymap("n", "<C-j>", "<C-w>j")
keymap("n", "<C-k>", "<C-w>k")
keymap("n", "<C-l>", "<C-w>l")

-- move lines up/down
keymap("v", "J", ":m '>+1<CR>gv=gv", { desc = "Move selection down" })
keymap("v", "K", ":m '<-2<CR>gv=gv", { desc = "Move selection up" })

-- keep cursor centered on scroll
keymap("n", "<C-d>", "<C-d>zz")
keymap("n", "<C-u>", "<C-u>zz")

-- paste without overwriting register
keymap("x", "<leader>p", '"_dP')

-- delete without yanking
keymap("n", "<leader>d", '"_d')
keymap("v", "<leader>d", '"_d')
```

### Autocommands

```lua
local augroup = vim.api.nvim_create_augroup("UserConfig", { clear = true })

-- highlight on yank
vim.api.nvim_create_autocmd("TextYankPost", {
    group = augroup,
    callback = function()
        vim.highlight.on_yank({ timeout = 200 })
    end,
})

-- remove trailing whitespace on save
vim.api.nvim_create_autocmd("BufWritePre", {
    group = augroup,
    pattern = "*",
    command = [[%s/\s\+$//e]],
})

-- restore cursor position
vim.api.nvim_create_autocmd("BufReadPost", {
    group = augroup,
    callback = function()
        local mark = vim.api.nvim_buf_get_mark(0, '"')
        local lcount = vim.api.nvim_buf_line_count(0)
        if mark[1] > 0 and mark[1] <= lcount then
            pcall(vim.api.nvim_win_set_cursor, 0, mark)
        end
    end,
})
```

## lazy.nvim (Plugin Manager)

### Bootstrap

```lua
-- ~/.config/nvim/init.lua (top of file)
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not vim.loop.fs_stat(lazypath) then
    vim.fn.system({
        "git", "clone", "--filter=blob:none",
        "https://github.com/folke/lazy.nvim.git",
        "--branch=stable", lazypath,
    })
end
vim.opt.rtp:prepend(lazypath)

require("lazy").setup({
    -- inline plugin specs
    { "folke/tokyonight.nvim", priority = 1000 },
    { import = "plugins" },   -- load all files from lua/plugins/
}, {
    change_detection = { notify = false },
})
```

### Plugin Spec Format

```lua
-- ~/.config/nvim/lua/plugins/example.lua
return {
    "author/plugin-name",
    dependencies = { "dep1", "dep2" },
    event = "BufReadPost",             -- lazy load on event
    ft = { "go", "lua" },             -- lazy load on filetype
    cmd = "PluginCommand",            -- lazy load on command
    keys = {                          -- lazy load on keymap
        { "<leader>f", "<cmd>PluginCmd<cr>", desc = "Do thing" },
    },
    opts = {                          -- passed to plugin.setup()
        setting = true,
    },
    config = function(_, opts)        -- custom setup (overrides opts)
        require("plugin-name").setup(opts)
    end,
}
```

### Managing Plugins

```bash
:Lazy                  # open lazy.nvim UI
:Lazy sync             # install + update + clean
:Lazy update           # update plugins
:Lazy clean            # remove unused plugins
:Lazy health           # check plugin health
```

## Treesitter

### Setup

```lua
-- ~/.config/nvim/lua/plugins/treesitter.lua
return {
    "nvim-treesitter/nvim-treesitter",
    build = ":TSUpdate",
    event = "BufReadPost",
    opts = {
        ensure_installed = {
            "bash", "c", "go", "javascript", "json", "lua",
            "markdown", "python", "rust", "typescript", "yaml",
        },
        auto_install = true,
        highlight = { enable = true },
        indent = { enable = true },
        incremental_selection = {
            enable = true,
            keymaps = {
                init_selection = "<C-space>",
                node_incremental = "<C-space>",
                scope_incremental = false,
                node_decremental = "<bs>",
            },
        },
    },
    config = function(_, opts)
        require("nvim-treesitter.configs").setup(opts)
    end,
}
```

### Commands

```bash
:TSInstall go          # install parser for go
:TSUpdate              # update all parsers
:TSModuleInfo          # show enabled modules
:InspectTree           # show syntax tree (nvim 0.9+)
```

## Built-in LSP

### Setup with lspconfig

```lua
-- ~/.config/nvim/lua/plugins/lsp.lua
return {
    "neovim/nvim-lspconfig",
    dependencies = {
        "mason.nvim",
        "williamboman/mason-lspconfig.nvim",
    },
    event = "BufReadPre",
    config = function()
        local lspconfig = require("lspconfig")
        local capabilities = vim.lsp.protocol.make_client_capabilities()

        -- keymaps when LSP attaches
        vim.api.nvim_create_autocmd("LspAttach", {
            callback = function(event)
                local map = function(keys, func, desc)
                    vim.keymap.set("n", keys, func, {
                        buffer = event.buf, desc = "LSP: " .. desc,
                    })
                end
                map("gd", vim.lsp.buf.definition, "Go to definition")
                map("gr", vim.lsp.buf.references, "Go to references")
                map("gI", vim.lsp.buf.implementation, "Go to implementation")
                map("K", vim.lsp.buf.hover, "Hover documentation")
                map("<leader>rn", vim.lsp.buf.rename, "Rename symbol")
                map("<leader>ca", vim.lsp.buf.code_action, "Code action")
                map("<leader>D", vim.lsp.buf.type_definition, "Type definition")
                map("[d", vim.diagnostic.goto_prev, "Previous diagnostic")
                map("]d", vim.diagnostic.goto_next, "Next diagnostic")
            end,
        })

        -- configure servers
        lspconfig.gopls.setup({ capabilities = capabilities })
        lspconfig.lua_ls.setup({
            capabilities = capabilities,
            settings = {
                Lua = {
                    workspace = { checkThirdParty = false },
                    telemetry = { enable = false },
                },
            },
        })
        lspconfig.ts_ls.setup({ capabilities = capabilities })
        lspconfig.pyright.setup({ capabilities = capabilities })
    end,
}
```

### Mason (LSP/Tool Installer)

```lua
-- ~/.config/nvim/lua/plugins/mason.lua
return {
    "williamboman/mason.nvim",
    cmd = "Mason",
    opts = {},                         -- calls mason.setup({})
}
```

```bash
:Mason                 # open Mason UI
:MasonInstall gopls    # install a server
:LspInfo               # show active LSP clients
:LspLog                # view LSP logs
:LspRestart            # restart LSP servers
```

## Telescope (Fuzzy Finder)

```lua
-- ~/.config/nvim/lua/plugins/telescope.lua
return {
    "nvim-telescope/telescope.nvim",
    branch = "0.1.x",
    dependencies = {
        "nvim-lua/plenary.nvim",
        { "nvim-telescope/telescope-fzf-native.nvim", build = "make" },
    },
    keys = {
        { "<leader>ff", "<cmd>Telescope find_files<cr>", desc = "Find files" },
        { "<leader>fg", "<cmd>Telescope live_grep<cr>", desc = "Live grep" },
        { "<leader>fb", "<cmd>Telescope buffers<cr>", desc = "Buffers" },
        { "<leader>fh", "<cmd>Telescope help_tags<cr>", desc = "Help tags" },
        { "<leader>fr", "<cmd>Telescope oldfiles<cr>", desc = "Recent files" },
        { "<leader>fd", "<cmd>Telescope diagnostics<cr>", desc = "Diagnostics" },
        { "<leader>fs", "<cmd>Telescope lsp_document_symbols<cr>", desc = "Symbols" },
        { "<leader>/", "<cmd>Telescope current_buffer_fuzzy_find<cr>", desc = "Fuzzy find in buffer" },
    },
    config = function()
        local telescope = require("telescope")
        telescope.setup({
            defaults = {
                file_ignore_patterns = { "node_modules", ".git/", "vendor/" },
            },
        })
        telescope.load_extension("fzf")
    end,
}
```

## Diagnostics

```lua
-- configure diagnostic display
vim.diagnostic.config({
    virtual_text = { prefix = "●" },
    signs = true,
    underline = true,
    update_in_insert = false,
    severity_sort = true,
    float = { border = "rounded", source = "always" },
})
```

```bash
# in normal mode
[d                     # previous diagnostic
]d                     # next diagnostic
<leader>e              # open diagnostic float (if mapped)
<leader>q              # diagnostic list (if mapped)
```

## Health Checks

```bash
:checkhealth                   # full health check
:checkhealth nvim              # neovim core
:checkhealth lazy              # plugin manager
:checkhealth lsp               # LSP status
```

## Neovim-Specific Commands

```bash
:lua print(vim.fn.stdpath("data"))    # show data directory
:lua print(vim.inspect(vim.opt.rtp))  # inspect option value
:lua =vim.version()                    # show neovim version
:source %                              # reload current file
:messages                              # show message history
```

## Tips

- Start with a minimal `init.lua` and add plugins one at a time. Starter configs like kickstart.nvim are good references.
- Use `vim.keymap.set` over `vim.api.nvim_set_keymap` -- it supports lua callbacks and is simpler.
- lazy.nvim's `event`, `ft`, `cmd`, and `keys` options dramatically improve startup time.
- `vim.lsp.buf.format()` formats the current buffer using the attached LSP server.
- Treesitter replaces regex-based syntax highlighting with a real parser -- much more accurate.
- `:InspectTree` (nvim 0.9+) shows the live syntax tree -- essential for debugging Treesitter queries.
- Mason installs LSP servers, formatters, and linters to a local directory -- no system-wide install needed.
- `vim.opt` returns an option object. Use `vim.opt.thing:get()` if you need the raw value.
- Neovim's built-in terminal (`:terminal`) supports all normal mode commands for scrolling and copying.
- Keep plugin configs in separate files under `lua/plugins/` -- lazy.nvim auto-loads them with `{ import = "plugins" }`.
