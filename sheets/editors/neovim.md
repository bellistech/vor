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

## Lua API Patterns

### vim.api.nvim_* (Core API)

```lua
-- buffer operations
local buf = vim.api.nvim_create_buf(false, true)     -- unlisted scratch buffer
vim.api.nvim_buf_set_lines(buf, 0, -1, false, {"line1", "line2"})
local lines = vim.api.nvim_buf_get_lines(0, 0, -1, false)  -- all lines in current buf
vim.api.nvim_buf_set_name(buf, "scratch")
vim.api.nvim_buf_delete(buf, { force = true })

-- window operations
local win = vim.api.nvim_get_current_win()
vim.api.nvim_win_set_cursor(win, { 10, 0 })          -- row 10, col 0
local cursor = vim.api.nvim_win_get_cursor(0)         -- {row, col}
vim.api.nvim_win_set_option(0, "wrap", true)

-- set buffer-local and window-local options
vim.api.nvim_set_option_value("filetype", "lua", { buf = buf })
vim.api.nvim_set_option_value("number", true, { win = win })

-- create user commands
vim.api.nvim_create_user_command("Greet", function(opts)
    print("Hello, " .. opts.args)
end, { nargs = 1, desc = "Greet someone" })
```

### vim.fn.* (Vimscript Functions)

```lua
-- call any vimscript function via vim.fn
local home = vim.fn.expand("~")                       -- expand ~ to home dir
local exists = vim.fn.filereadable("/tmp/foo")         -- 1 if readable, 0 if not
local cwd = vim.fn.getcwd()
local lines = vim.fn.line("$")                        -- last line number
local input = vim.fn.input("Enter name: ")            -- prompt user
local data_dir = vim.fn.stdpath("data")               -- ~/.local/share/nvim
local config_dir = vim.fn.stdpath("config")            -- ~/.config/nvim

-- system commands
local output = vim.fn.system("git branch --show-current")
local code = vim.v.shell_error                         -- exit code of last system()

-- list/dict operations
local joined = vim.fn.join({"a", "b", "c"}, ", ")
local idx = vim.fn.index({"a", "b", "c"}, "b")        -- 1
```

### vim.cmd and vim.keymap.set

```lua
-- execute ex commands
vim.cmd("colorscheme tokyonight")
vim.cmd.edit("/tmp/scratch.txt")                       -- vim.cmd.{command} form
vim.cmd.highlight({ "Normal", "guibg=NONE", bang = true })

-- multi-line vim commands
vim.cmd([[
    augroup MyGroup
        autocmd!
        autocmd FileType go setlocal tabstop=4
    augroup END
]])

-- vim.keymap.set supports lua callbacks
vim.keymap.set("n", "<leader>x", function()
    local line = vim.api.nvim_get_current_line()
    print("Current line: " .. line)
end, { desc = "Print current line" })

-- set keymap for specific buffer
vim.keymap.set("n", "K", vim.lsp.buf.hover, { buffer = 0, desc = "LSP hover" })
```

### vim.opt and vim.g

```lua
-- vim.opt wraps :set — returns Option objects
vim.opt.completeopt = { "menu", "menuone", "noselect" }
vim.opt.shortmess:append("c")                         -- append to string option
vim.opt.path:prepend("/usr/include")
vim.opt.wildignore:remove("*.o")

-- get raw value from option object
local tabstop = vim.opt.tabstop:get()                  -- returns number
local rtp = vim.opt.rtp:get()                          -- returns table

-- vim.g = global variables, vim.b = buffer, vim.w = window
vim.g.loaded_netrw = 1                                 -- disable netrw
vim.g.loaded_netrwPlugin = 1
vim.b.my_buffer_var = "hello"
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

### Lazy-Loading Patterns

```lua
-- load on multiple events
{ "plugin", event = { "BufReadPost", "BufNewFile" } }

-- load on VeryLazy (after UI is ready, non-blocking)
{ "plugin", event = "VeryLazy" }

-- build step (runs after install/update)
{ "plugin", build = "make" }
{ "plugin", build = ":TSUpdate" }
{ "plugin", build = function() vim.fn.system("make install") end }

-- conditional loading
{ "plugin", enabled = vim.fn.executable("cargo") == 1 }
{ "plugin", cond = not vim.g.vscode }                     -- skip in vscode-neovim

-- priority (higher loads first, default 50)
{ "colorscheme-plugin", lazy = false, priority = 1000 }

-- pin to a specific version
{ "plugin", tag = "v2.0.0" }
{ "plugin", commit = "abc1234" }

-- init runs before plugin loads (use for vim.g settings)
{ "plugin", init = function() vim.g.plugin_setting = 1 end }
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

### Treesitter Text Objects

```lua
-- ~/.config/nvim/lua/plugins/treesitter-textobjects.lua
return {
    "nvim-treesitter/nvim-treesitter-textobjects",
    dependencies = "nvim-treesitter/nvim-treesitter",
    event = "BufReadPost",
    config = function()
        require("nvim-treesitter.configs").setup({
            textobjects = {
                select = {
                    enable = true,
                    lookahead = true,              -- jump forward to match
                    keymaps = {
                        ["af"] = "@function.outer", -- around function
                        ["if"] = "@function.inner", -- inside function
                        ["ac"] = "@class.outer",
                        ["ic"] = "@class.inner",
                        ["aa"] = "@parameter.outer",
                        ["ia"] = "@parameter.inner",
                        ["ai"] = "@conditional.outer",
                        ["ii"] = "@conditional.inner",
                        ["al"] = "@loop.outer",
                        ["il"] = "@loop.inner",
                    },
                },
                move = {
                    enable = true,
                    set_jumps = true,
                    goto_next_start = {
                        ["]f"] = "@function.outer",
                        ["]c"] = "@class.outer",
                        ["]a"] = "@parameter.inner",
                    },
                    goto_previous_start = {
                        ["[f"] = "@function.outer",
                        ["[c"] = "@class.outer",
                        ["[a"] = "@parameter.inner",
                    },
                },
                swap = {
                    enable = true,
                    swap_next = { ["<leader>sn"] = "@parameter.inner" },
                    swap_previous = { ["<leader>sp"] = "@parameter.inner" },
                },
            },
        })
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

### LSP Capabilities with nvim-cmp

```lua
-- extend capabilities for completion
local capabilities = vim.lsp.protocol.make_client_capabilities()
capabilities = require("cmp_nvim_lsp").default_capabilities(capabilities)

-- pass to each server
lspconfig.gopls.setup({ capabilities = capabilities })
```

### LSP Formatting

```lua
-- format on save via LspAttach
vim.api.nvim_create_autocmd("LspAttach", {
    callback = function(event)
        local client = vim.lsp.get_client_by_id(event.data.client_id)

        -- format on save if server supports it
        if client and client.supports_method("textDocument/formatting") then
            vim.api.nvim_create_autocmd("BufWritePre", {
                buffer = event.buf,
                callback = function()
                    vim.lsp.buf.format({
                        bufnr = event.buf,
                        timeout_ms = 3000,
                    })
                end,
            })
        end
    end,
})

-- manual format keymap
vim.keymap.set("n", "<leader>lf", function()
    vim.lsp.buf.format({ async = true })
end, { desc = "LSP format" })
```

### LSP Border Configuration

```lua
-- rounded borders for hover and signature help
vim.lsp.handlers["textDocument/hover"] = vim.lsp.with(
    vim.lsp.handlers.hover, { border = "rounded" }
)
vim.lsp.handlers["textDocument/signatureHelp"] = vim.lsp.with(
    vim.lsp.handlers.signature_help, { border = "rounded" }
)
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

## Completion (nvim-cmp)

### Setup

```lua
-- ~/.config/nvim/lua/plugins/cmp.lua
return {
    "hrsh7th/nvim-cmp",
    event = "InsertEnter",
    dependencies = {
        "hrsh7th/cmp-nvim-lsp",            -- LSP completions
        "hrsh7th/cmp-buffer",              -- buffer words
        "hrsh7th/cmp-path",                -- file paths
        "L3MON4D3/LuaSnip",               -- snippet engine
        "saadparwaiz1/cmp_luasnip",        -- snippet completions
        "rafamadriz/friendly-snippets",    -- snippet collection
    },
    config = function()
        local cmp = require("cmp")
        local luasnip = require("luasnip")

        require("luasnip.loaders.from_vscode").lazy_load()

        cmp.setup({
            snippet = {
                expand = function(args)
                    luasnip.lsp_expand(args.body)
                end,
            },
            mapping = cmp.mapping.preset.insert({
                ["<C-b>"] = cmp.mapping.scroll_docs(-4),
                ["<C-f>"] = cmp.mapping.scroll_docs(4),
                ["<C-Space>"] = cmp.mapping.complete(),
                ["<C-e>"] = cmp.mapping.abort(),
                ["<CR>"] = cmp.mapping.confirm({ select = true }),
                ["<Tab>"] = cmp.mapping(function(fallback)
                    if cmp.visible() then
                        cmp.select_next_item()
                    elseif luasnip.expand_or_jumpable() then
                        luasnip.expand_or_jump()
                    else
                        fallback()
                    end
                end, { "i", "s" }),
                ["<S-Tab>"] = cmp.mapping(function(fallback)
                    if cmp.visible() then
                        cmp.select_prev_item()
                    elseif luasnip.jumpable(-1) then
                        luasnip.jump(-1)
                    else
                        fallback()
                    end
                end, { "i", "s" }),
            }),
            sources = cmp.config.sources({
                { name = "nvim_lsp" },          -- highest priority
                { name = "luasnip" },
                { name = "path" },
            }, {
                { name = "buffer" },            -- fallback group
            }),
            formatting = {
                format = function(entry, item)
                    local source_names = {
                        nvim_lsp = "[LSP]",
                        luasnip = "[Snip]",
                        buffer = "[Buf]",
                        path = "[Path]",
                    }
                    item.menu = source_names[entry.source.name] or ""
                    return item
                end,
            },
        })
    end,
}
```

## Telescope (Fuzzy Finder)

### Basic Setup

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

### Telescope Extensions and Custom Pickers

```lua
-- load extensions after telescope.setup()
telescope.load_extension("fzf")                       -- faster fuzzy matching

-- custom picker: find dotfiles
vim.keymap.set("n", "<leader>f.", function()
    require("telescope.builtin").find_files({
        prompt_title = "Dotfiles",
        cwd = vim.fn.expand("~"),
        hidden = true,
        find_command = { "fd", "--type", "f", "--hidden", "--glob", ".*" },
    })
end, { desc = "Find dotfiles" })

-- custom picker: grep in current buffer's directory
vim.keymap.set("n", "<leader>fD", function()
    local dir = vim.fn.expand("%:p:h")
    require("telescope.builtin").live_grep({ cwd = dir })
end, { desc = "Grep in buffer directory" })

-- custom picker: git commits for current file
vim.keymap.set("n", "<leader>gc", function()
    require("telescope.builtin").git_bcommits()
end, { desc = "Git commits for buffer" })

-- custom picker using actions
local actions = require("telescope.actions")
telescope.setup({
    defaults = {
        mappings = {
            i = {                                      -- insert mode mappings
                ["<C-j>"] = actions.move_selection_next,
                ["<C-k>"] = actions.move_selection_previous,
                ["<C-q>"] = actions.send_selected_to_qflist + actions.open_qflist,
            },
        },
        layout_strategy = "horizontal",
        layout_config = { preview_width = 0.55 },
    },
})
```

## DAP (Debug Adapter Protocol)

### Setup with nvim-dap

```lua
-- ~/.config/nvim/lua/plugins/dap.lua
return {
    "mfussenegger/nvim-dap",
    dependencies = {
        "rcarriga/nvim-dap-ui",
        "nvim-neotest/nvim-nio",           -- required by dap-ui
        "leoluz/nvim-dap-go",              -- Go adapter
    },
    keys = {
        { "<leader>db", function() require("dap").toggle_breakpoint() end, desc = "Toggle breakpoint" },
        { "<leader>dc", function() require("dap").continue() end, desc = "Continue" },
        { "<leader>do", function() require("dap").step_over() end, desc = "Step over" },
        { "<leader>di", function() require("dap").step_into() end, desc = "Step into" },
        { "<leader>dO", function() require("dap").step_out() end, desc = "Step out" },
        { "<leader>dr", function() require("dap").repl.open() end, desc = "Open REPL" },
        { "<leader>du", function() require("dapui").toggle() end, desc = "Toggle DAP UI" },
        { "<leader>dB", function()
            require("dap").set_breakpoint(vim.fn.input("Condition: "))
        end, desc = "Conditional breakpoint" },
    },
    config = function()
        local dap = require("dap")
        local dapui = require("dapui")

        dapui.setup()
        require("dap-go").setup()

        -- auto open/close dap-ui
        dap.listeners.after.event_initialized["dapui"] = function() dapui.open() end
        dap.listeners.before.event_terminated["dapui"] = function() dapui.close() end
        dap.listeners.before.event_exited["dapui"] = function() dapui.close() end

        -- custom adapter example (Node.js)
        dap.adapters.node2 = {
            type = "executable",
            command = "node",
            args = { vim.fn.stdpath("data") .. "/mason/packages/node-debug2-adapter/out/src/nodeDebug.js" },
        }
        dap.configurations.javascript = {
            {
                type = "node2",
                request = "launch",
                name = "Launch file",
                program = "${file}",
                cwd = vim.fn.getcwd(),
            },
        }
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

## Neovim-Specific Features

### Floating Windows

```lua
-- create a floating window
local buf = vim.api.nvim_create_buf(false, true)        -- unlisted, scratch
vim.api.nvim_buf_set_lines(buf, 0, -1, false, { "Hello from float!", "", "Press q to close" })

local width = 40
local height = 5
local win = vim.api.nvim_open_win(buf, true, {
    relative = "editor",
    width = width,
    height = height,
    col = (vim.o.columns - width) / 2,                  -- centered
    row = (vim.o.lines - height) / 2,
    style = "minimal",
    border = "rounded",                                  -- none/single/double/rounded/solid
    title = " My Float ",
    title_pos = "center",
})

-- close float with q
vim.keymap.set("n", "q", function()
    vim.api.nvim_win_close(win, true)
end, { buffer = buf })
```

### Virtual Text and Extmarks

```lua
local ns = vim.api.nvim_create_namespace("my_plugin")

-- add virtual text at end of line 5
vim.api.nvim_buf_set_extmark(0, ns, 4, 0, {            -- 0-indexed line
    virt_text = { { " -- this is virtual text", "Comment" } },
    virt_text_pos = "eol",                               -- eol/overlay/right_align
})

-- add virtual lines below line 3
vim.api.nvim_buf_set_extmark(0, ns, 2, 0, {
    virt_lines = { { { "  virtual line below", "WarningMsg" } } },
})

-- highlight a range (line 1, col 0 to line 1, col 10)
vim.api.nvim_buf_set_extmark(0, ns, 0, 0, {
    end_row = 0,
    end_col = 10,
    hl_group = "Visual",
})

-- clear all extmarks in namespace
vim.api.nvim_buf_clear_namespace(0, ns, 0, -1)
```

## Statusline (lualine)

```lua
-- ~/.config/nvim/lua/plugins/lualine.lua
return {
    "nvim-lualine/lualine.nvim",
    dependencies = "nvim-tree/nvim-web-devicons",
    event = "VeryLazy",
    opts = {
        options = {
            theme = "auto",
            component_separators = { left = "|", right = "|" },
            section_separators = { left = "", right = "" },
            globalstatus = true,                       -- single statusline for all windows
        },
        sections = {
            lualine_a = { "mode" },
            lualine_b = { "branch", "diff", "diagnostics" },
            lualine_c = { { "filename", path = 1 } },  -- 0=name, 1=relative, 2=absolute
            lualine_x = { "encoding", "fileformat", "filetype" },
            lualine_y = { "progress" },
            lualine_z = { "location" },
        },
        extensions = { "lazy", "fugitive", "quickfix" },
    },
}
```

## Git Integration

### Gitsigns

```lua
-- ~/.config/nvim/lua/plugins/gitsigns.lua
return {
    "lewis6991/gitsigns.nvim",
    event = "BufReadPre",
    opts = {
        signs = {
            add = { text = "+" },
            change = { text = "~" },
            delete = { text = "_" },
            topdelete = { text = "-" },
            changedelete = { text = "~" },
        },
        on_attach = function(bufnr)
            local gs = require("gitsigns")
            local map = function(mode, l, r, desc)
                vim.keymap.set(mode, l, r, { buffer = bufnr, desc = desc })
            end

            map("n", "]h", gs.next_hunk, "Next hunk")
            map("n", "[h", gs.prev_hunk, "Previous hunk")
            map("n", "<leader>hs", gs.stage_hunk, "Stage hunk")
            map("n", "<leader>hr", gs.reset_hunk, "Reset hunk")
            map("n", "<leader>hu", gs.undo_stage_hunk, "Undo stage hunk")
            map("n", "<leader>hS", gs.stage_buffer, "Stage buffer")
            map("n", "<leader>hR", gs.reset_buffer, "Reset buffer")
            map("n", "<leader>hp", gs.preview_hunk, "Preview hunk")
            map("n", "<leader>hb", function() gs.blame_line({ full = true }) end, "Blame line")
            map("n", "<leader>hd", gs.diffthis, "Diff this")
        end,
    },
}
```

### Fugitive

```lua
-- ~/.config/nvim/lua/plugins/fugitive.lua
return {
    "tpope/vim-fugitive",
    cmd = { "Git", "Gwrite", "Gread", "Gdiffsplit" },
    keys = {
        { "<leader>gs", "<cmd>Git<cr>", desc = "Git status" },
        { "<leader>gd", "<cmd>Gdiffsplit<cr>", desc = "Git diff" },
        { "<leader>gb", "<cmd>Git blame<cr>", desc = "Git blame" },
        { "<leader>gl", "<cmd>Git log --oneline<cr>", desc = "Git log" },
    },
}
```

```bash
# inside :Git status buffer
s                      # stage file
u                      # unstage file
=                      # toggle diff inline
cc                     # commit
X                      # discard changes (checkout)
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
:Inspect                               # show highlight groups under cursor
:terminal                              # open built-in terminal
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
- Use `vim.api.nvim_create_namespace()` to isolate extmarks and virtual text per plugin.
- nvim-cmp sources are grouped: first group is tried first, fallback group only if first yields nothing.
- DAP breakpoints persist across restarts if you use a session plugin like `nvim-dap`'s breakpoint persistence.
- Treesitter text objects let you operate on code structures (functions, classes, parameters) instead of just words and lines.
- Use `:Inspect` to see which highlight groups apply at the cursor -- essential for theme debugging.
- Gitsigns `on_attach` is the cleanest place to define per-buffer git keymaps.

## References

- [Neovim Documentation](https://neovim.io/doc/) -- built-in help and user manual
- [Neovim Lua Guide](https://neovim.io/doc/user/lua-guide.html) -- Lua API for configuration and plugins
- [Neovim API Reference](https://neovim.io/doc/user/api.html) -- vim.api, vim.fn, vim.lsp, etc.
- [Neovim LSP Documentation](https://neovim.io/doc/user/lsp.html) -- built-in Language Server Protocol client
- [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig) -- quickstart configs for LSP servers
- [Neovim GitHub](https://github.com/neovim/neovim) -- source, issues, and releases
- [Neovim Treesitter](https://github.com/nvim-treesitter/nvim-treesitter) -- syntax parsing and highlighting
- [lazy.nvim](https://github.com/folke/lazy.nvim) -- modern plugin manager
- [Awesome Neovim](https://github.com/rockerBOO/awesome-neovim) -- curated list of Neovim plugins
- [Neovim Discourse](https://neovim.discourse.group/) -- official community forum
