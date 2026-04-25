# Neovim (Hyperextensible Vim-Based Editor)

> Lua-first, LSP-native, Treesitter-powered editor — differences from vim live in editors/vim sheet — focus here on Lua + LSP + Treesitter + modern plugin ecosystem.

## Setup

### Install

```bash
# macOS
brew install neovim                                # stable
brew install --HEAD neovim                         # nightly
brew install neovim --cask                         # GUI variant where available

# Debian / Ubuntu — distro versions are ALWAYS stale
sudo apt install neovim                            # avoid: usually 0.7
# preferred: PPA or AppImage
sudo add-apt-repository ppa:neovim-ppa/unstable
sudo apt update && sudo apt install neovim
# or AppImage (works on any glibc>=2.31 distro)
curl -LO https://github.com/neovim/neovim/releases/latest/download/nvim-linux-x86_64.appimage
chmod u+x nvim-linux-x86_64.appimage
sudo mv nvim-linux-x86_64.appimage /usr/local/bin/nvim

# Arch / Manjaro
sudo pacman -S neovim                              # stable
yay -S neovim-git                                  # nightly via AUR

# Fedora
sudo dnf install neovim python3-neovim

# Nix / NixOS
nix-env -iA nixpkgs.neovim
# flake-style:
# environment.systemPackages = [ pkgs.neovim ];

# Windows
winget install Neovim.Neovim
choco install neovim
scoop install main/neovim
```

### Verify

```bash
nvim --version
# NVIM v0.10.2
# Build type: Release
# LuaJIT 2.1.1713484068
# Run "nvim -V1 -v" for more info

nvim --headless "+lua print(vim.version().major, vim.version().minor)" +qa
# 0	10
```

### Version Differences (0.8 → 0.11)

```bash
# 0.8 (Oct 2022) — last "old way" baseline
#   nvim_set_keymap was idiomatic, no native inlay hints

# 0.9 (Apr 2023)
#   - vim.lsp.inlay_hint introduced (toggleable inlay hints)
#   - vim.loader.enable() bytecode caching for faster require()
#   - :terminal got better keybindings
#   - statuscolumn option added

# 0.10 (May 2024) — current default-config era
#   - Default colorscheme is no longer torte: it's a 256-color "default"
#   - Built-in default keymaps: gra (code action), grn (rename),
#     grr (references), gri (implementation), [d / ]d (diagnostics)
#   - vim.snippet.expand / jump native (no plugin needed for basic)
#   - tree-sitter highlights enabled by default for some filetypes
#   - vim.deprecate warning system tightened
#   - vim.ui.open for cross-platform "open URL/file"

# 0.11 (May 2025)
#   - vim.lsp.config / vim.lsp.enable — declarative LSP without lspconfig
#   - Snippet completion in built-in vim.lsp.completion
#   - Default LSP keymaps wired to vim.lsp.buf.* automatically
#   - :checkhealth now async

# nightly (HEAD) — 0.12-dev features land here
#   - Track NEWS.txt: :help news / nvim/runtime/doc/news.txt
```

### Smoke Test

```bash
# launch in clean state — no init.lua, no plugins
nvim --clean
nvim -u NONE                                       # no config
nvim -u NORC                                       # no config, but plugins
nvim --noplugin                                    # config only, no plugins

# headless eval (CI / scripting)
nvim --headless "+echo 'hi'" +qa
nvim --headless -u NONE \
     "+lua print(_VERSION)" +qa

# print runtimepath
nvim --headless "+lua print(vim.o.runtimepath)" +qa
```

## Configuration Layout

### Canonical Directory

```bash
~/.config/nvim/                                    # XDG_CONFIG_HOME
├── init.lua                                       # canonical entrypoint
├── lazy-lock.json                                 # plugin lockfile
├── lua/
│   └── <namespace>/                               # e.g. "user", "myname"
│       ├── init.lua                               # require'user' loads this
│       ├── options.lua                            # vim.opt.* settings
│       ├── keymaps.lua                            # vim.keymap.set
│       ├── autocmds.lua                           # autocommands
│       ├── plugins.lua                            # lazy.nvim spec OR…
│       ├── plugins/                               # split spec per plugin
│       │   ├── init.lua                           # returns table of specs
│       │   ├── lsp.lua
│       │   ├── treesitter.lua
│       │   ├── telescope.lua
│       │   └── colors.lua
│       ├── config/
│       │   ├── lsp.lua                            # LSP setup helpers
│       │   └── cmp.lua                            # completion setup
│       └── utils.lua
├── after/                                         # loaded AFTER plugins/runtime
│   ├── plugin/                                    # post-plugin overrides
│   │   └── overrides.lua
│   └── ftplugin/                                  # filetype overrides
│       ├── go.lua
│       ├── python.lua
│       └── markdown.lua
├── ftdetect/                                      # custom filetype detect
│   └── myext.lua
├── snippets/                                      # LuaSnip / native snippets
│   └── all.lua
├── colors/                                        # custom colorschemes
└── spell/                                         # custom dictionaries
```

### XDG Locations

```bash
# config:  $XDG_CONFIG_HOME/nvim   (defaults: ~/.config/nvim)
# data:    $XDG_DATA_HOME/nvim     (defaults: ~/.local/share/nvim)
# state:   $XDG_STATE_HOME/nvim    (defaults: ~/.local/state/nvim)
# cache:   $XDG_CACHE_HOME/nvim    (defaults: ~/.cache/nvim)

# inspect at runtime
:echo stdpath('config')
:echo stdpath('data')
:echo stdpath('state')
:echo stdpath('cache')
:echo stdpath('config_dirs')                       # extra dirs

# multiple configs side-by-side via NVIM_APPNAME
NVIM_APPNAME=nvim-test nvim                        # uses ~/.config/nvim-test
```

### runtimepath Order

```bash
# &runtimepath is searched IN ORDER for runtime files
# default order (simplified):
#   1. $XDG_CONFIG_HOME/nvim                       # user config
#   2. site dirs ($XDG_DATA_HOME/nvim/site)
#   3. pack/*/start/<plugin>                       # auto-load plugins
#   4. $VIMRUNTIME                                 # bundled runtime
#   5. pack/*/opt/<plugin>                         # opt-in plugins
#   6. site/after, $XDG_CONFIG_HOME/nvim/after     # 'after' overrides

# pack/*/start vs pack/*/opt
#   start  — auto-loaded on startup
#   opt    — load on demand via :packadd <name>

# debug — see exact order
:set runtimepath?
:lua =vim.opt.runtimepath:get()                    # Lua list form
```

### init.lua vs init.vim

```bash
# nvim looks for, in order:
#   $XDG_CONFIG_HOME/nvim/init.lua
#   $XDG_CONFIG_HOME/nvim/init.vim
# whichever is found first wins; do NOT have both

# migrate via:
mv ~/.config/nvim/init.vim ~/.config/nvim/init.vim.bak
touch ~/.config/nvim/init.lua
# then in init.lua:
# vim.cmd("source ~/.config/nvim/init.vim.bak")    # incremental migration
```

## init.lua — Canonical Boilerplate

### The Modern Starter

```lua
-- ~/.config/nvim/init.lua

-- 1. LEADER KEYS FIRST — must be set before any keymap that uses <leader>
vim.g.mapleader = " "
vim.g.maplocalleader = "\\"

-- 2. Bootstrap lazy.nvim plugin manager
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not (vim.uv or vim.loop).fs_stat(lazypath) then
    local lazyrepo = "https://github.com/folke/lazy.nvim.git"
    local out = vim.fn.system({
        "git", "clone", "--filter=blob:none", "--branch=stable", lazyrepo, lazypath,
    })
    if vim.v.shell_error ~= 0 then
        vim.api.nvim_echo({
            { "Failed to clone lazy.nvim:\n", "ErrorMsg" },
            { out, "WarningMsg" },
            { "\nPress any key to exit..." },
        }, true, {})
        vim.fn.getchar()
        os.exit(1)
    end
end
vim.opt.rtp:prepend(lazypath)

-- 3. Load core options and keymaps BEFORE plugins (so plugins see them)
require("user.options")
require("user.keymaps")
require("user.autocmds")

-- 4. Plugin specs
require("lazy").setup("user.plugins", {
    install = { colorscheme = { "tokyonight", "habamax" } },
    checker = { enabled = true, notify = false },
    change_detection = { notify = false },
    performance = {
        rtp = {
            disabled_plugins = {
                "gzip", "matchit", "matchparen", "netrwPlugin",
                "tarPlugin", "tohtml", "tutor", "zipPlugin",
            },
        },
    },
})

-- 5. Colorscheme last (after plugins are loaded)
vim.cmd.colorscheme("tokyonight")
```

### init.vim → init.lua Migration

```lua
-- vimscript                     -- lua
-- let mapleader=" "             vim.g.mapleader = " "
-- set number                    vim.opt.number = true
-- set tabstop=4                 vim.opt.tabstop = 4
-- nnoremap <leader>w :w<CR>     vim.keymap.set("n", "<leader>w", ":w<CR>")
-- autocmd BufRead *.go set ts=4 vim.api.nvim_create_autocmd("BufRead", {
--                                  pattern = "*.go",
--                                  callback = function()
--                                      vim.opt_local.tabstop = 4
--                                  end,
--                               })
-- colorscheme tokyonight        vim.cmd.colorscheme("tokyonight")
-- source other.vim              vim.cmd("source other.vim")
-- function! Foo()...endfunction local function foo() ... end
```

## vim.opt vs vim.o vs vim.bo vs vim.wo

### The Four Scopes

```lua
-- vim.opt — Lua-table OOP API (RECOMMENDED for most use)
vim.opt.number = true                              -- :set number
vim.opt.tabstop = 4                                -- :set tabstop=4
vim.opt.listchars = { tab = "» ", trail = "·" }    -- table → comma-list
vim.opt.shortmess:append("c")                      -- :set shortmess+=c
vim.opt.wildignore:remove("*.git")                 -- :set wildignore-=*.git
vim.opt.path:prepend("**")                         -- :set path^=**
print(vim.opt.tabstop:get())                       -- 4 (must call :get() to read)

-- vim.o — single-string get/set, NO table sugar
vim.o.number = true
vim.o.listchars = "tab:» ,trail:·"                 -- must build the string yourself
print(vim.o.tabstop)                               -- 4 (direct read)

-- vim.bo[bufnr] — buffer-local; bufnr=0 means current
vim.bo.tabstop = 2                                 -- only this buffer
vim.bo[5].filetype = "lua"                         -- buffer 5 specifically

-- vim.wo[winid] — window-local; winid=0 means current
vim.wo.number = true                               -- only this window
vim.wo[1001].cursorline = true                     -- specific window

-- vim.go — global only (rarely needed)
vim.go.title = true

-- vim.opt_local / vim.opt_global — explicit local/global
vim.opt_local.tabstop = 4                          -- like :setlocal
vim.opt_global.hidden = true                       -- like :setglobal
```

### The Global-vs-Local-Default Trap

```lua
-- many options are GLOBAL-LOCAL: changing :set affects only the current buffer/win
-- but new buffers inherit the GLOBAL value. This bites with:
--   tabstop, shiftwidth, expandtab, wrap, list, foldmethod, number, signcolumn

-- BAD: in init.lua, vim.bo.tabstop = 2 affects only the FIRST buffer
-- GOOD: use vim.opt (or vim.o) for global default; ftplugin/<ft>.lua for per-ft
vim.opt.tabstop = 2                                -- sets BOTH global and current
vim.opt_global.tabstop = 2                         -- sets only the global default
```

## Common vim.opt Settings

### Display

```lua
vim.opt.number = true                              -- absolute line numbers
vim.opt.relativenumber = true                      -- relative numbers (gj 5j etc)
vim.opt.cursorline = true                          -- highlight current line
vim.opt.cursorcolumn = false                       -- highlight current col (slow)
vim.opt.signcolumn = "yes"                         -- always show, prevents jump
vim.opt.colorcolumn = "80,100,120"                 -- ruler lines
vim.opt.scrolloff = 8                              -- vertical context
vim.opt.sidescrolloff = 8                          -- horizontal context
vim.opt.wrap = false                               -- soft wrap off
vim.opt.linebreak = true                           -- if wrap on, break at words
vim.opt.breakindent = true                         -- wrapped lines keep indent
vim.opt.showbreak = "↳ "                           -- prefix for wrapped lines
vim.opt.list = true                                -- show invisibles
vim.opt.listchars = {
    tab      = "» ",
    trail    = "·",
    nbsp     = "␣",
    extends  = "›",
    precedes = "‹",
}
vim.opt.fillchars = {
    eob       = " ",                               -- hide ~ on empty buffers
    fold      = " ",
    foldopen  = "",
    foldsep   = " ",
    foldclose = "",
}
vim.opt.termguicolors = true                       -- 24-bit color
vim.opt.background = "dark"
vim.opt.showmode = false                           -- statusline shows mode
vim.opt.cmdheight = 1                              -- 0 with noice.nvim
vim.opt.pumheight = 10                             -- popup max height
vim.opt.pumblend = 10                              -- popup transparency 0-100
vim.opt.winblend = 0                               -- floating-window transparency
vim.opt.laststatus = 3                             -- global statusline (0.7+)
```

### Indent / Tabs

```lua
vim.opt.tabstop = 4                                -- visual tab width
vim.opt.softtabstop = 4                            -- chars BS deletes
vim.opt.shiftwidth = 4                             -- >> << indent width (0=tabstop)
vim.opt.expandtab = true                           -- tab → spaces
vim.opt.smartindent = true                         -- C-like indent guess
vim.opt.autoindent = true                          -- copy prev line indent
vim.opt.shiftround = true                          -- round to shiftwidth
```

### Search

```lua
vim.opt.ignorecase = true                          -- case-insensitive search
vim.opt.smartcase = true                           -- case-sensitive if uppercase
vim.opt.hlsearch = true                            -- highlight matches
vim.opt.incsearch = true                           -- show matches as you type
vim.opt.inccommand = "split"                       -- :s preview pane (nvim only)
vim.opt.gdefault = false                           -- :s/x/y/ adds /g implicitly
```

### Splits / Windows

```lua
vim.opt.splitbelow = true                          -- :split puts new window below
vim.opt.splitright = true                          -- :vsplit puts new window right
vim.opt.splitkeep = "screen"                       -- minimize jump on resize (0.9+)
vim.opt.equalalways = false                        -- don't force equal sizes
vim.opt.winminheight = 0                           -- allow zero-height windows
vim.opt.winminwidth = 0
```

### Files / Persistence

```lua
vim.opt.undofile = true                            -- persistent undo across sessions
vim.opt.undodir = vim.fn.stdpath("state") .. "/undo"
vim.opt.swapfile = false                           -- no swap files
vim.opt.backup = false                             -- no ~ backup
vim.opt.writebackup = false                        -- no temp backup during save
vim.opt.autoread = true                            -- reload on external change
vim.opt.autowrite = false                          -- save on :next etc
vim.opt.confirm = true                             -- prompt instead of fail
vim.opt.fileencoding = "utf-8"
vim.opt.fileformat = "unix"
vim.opt.hidden = true                              -- :next on dirty buffer
```

### Editing

```lua
vim.opt.mouse = "a"                                -- mouse in all modes
vim.opt.mousemodel = "extend"                      -- right-click extends select
vim.opt.clipboard = "unnamedplus"                  -- system clipboard for y/p
vim.opt.virtualedit = "block"                      -- block-mode past EOL
vim.opt.backspace = "indent,eol,start"             -- backspace through everything
vim.opt.iskeyword:append("-")                      -- - is part of word
vim.opt.formatoptions = "jcroqlnt"                 -- see :h fo-table
vim.opt.completeopt = { "menu", "menuone", "noselect", "noinsert" }
vim.opt.shortmess:append("scIWF")                  -- silence noisy messages
vim.opt.spelllang = { "en_us" }
vim.opt.spell = false                              -- enable per-buffer
```

### Folds

```lua
vim.opt.foldmethod = "expr"                        -- treesitter does the work
vim.opt.foldexpr = "nvim_treesitter#foldexpr()"
vim.opt.foldlevel = 99                             -- start fully unfolded
vim.opt.foldlevelstart = 99
vim.opt.foldenable = true
vim.opt.foldnestmax = 4
vim.opt.foldcolumn = "0"                           -- "1" to show fold gutter
```

### Performance

```lua
vim.opt.updatetime = 200                           -- CursorHold delay (default 4000)
vim.opt.timeoutlen = 300                           -- mapped sequence timeout
vim.opt.ttimeoutlen = 10                           -- key code timeout
vim.opt.lazyredraw = false                         -- DO NOT enable with noice.nvim
vim.opt.redrawtime = 1500                          -- syntax giveup ms (large files)
vim.opt.synmaxcol = 240                            -- syntax highlight col cap
vim.opt.history = 1000
vim.opt.maxmempattern = 2000                       -- KB for pattern memory
```

## vim.api — The Public API

### Core Functions

```lua
-- autocommands
local grp = vim.api.nvim_create_augroup("MyGroup", { clear = true })
vim.api.nvim_create_autocmd({ "BufWritePre" }, {
    group   = grp,
    pattern = "*.go",
    desc    = "Format Go on save",
    callback = function(args)
        vim.lsp.buf.format({ bufnr = args.buf, async = false })
    end,
})

-- delete / get autocmds
vim.api.nvim_clear_autocmds({ group = grp })
vim.api.nvim_get_autocmds({ group = "MyGroup", event = "BufWritePre" })

-- user commands
vim.api.nvim_create_user_command("Format", function(opts)
    vim.lsp.buf.format({ async = true })
end, {
    desc  = "Format current buffer via LSP",
    range = true,
    nargs = "?",
    bang  = true,
    complete = "filetype",
})
vim.api.nvim_del_user_command("Format")

-- keymaps (modern API)
vim.keymap.set("n", "<leader>w", ":write<CR>", { silent = true, desc = "Save file" })
vim.keymap.del("n", "<leader>w")

-- LEGACY — does NOT accept lua callbacks
-- vim.api.nvim_set_keymap("n", "<leader>w", ":write<CR>", { noremap = true, silent = true })

-- buffer text
local lines = vim.api.nvim_buf_get_lines(0, 0, -1, false)
vim.api.nvim_buf_set_lines(0, 0, -1, false, { "new", "content" })
vim.api.nvim_buf_get_text(0, 0, 0, 0, 10, {})      -- partial line range
vim.api.nvim_buf_set_text(0, 0, 0, 0, 0, { "x" })

-- buffer / window handles
local bufnr = vim.api.nvim_get_current_buf()
local winid = vim.api.nvim_get_current_win()
vim.api.nvim_win_set_cursor(0, { 10, 0 })          -- {row(1), col(0)}
vim.api.nvim_buf_get_name(0)                       -- absolute path
vim.api.nvim_buf_set_name(0, "/tmp/foo")

-- runtime files
local files = vim.api.nvim_get_runtime_file("colors/*.lua", true)
vim.api.nvim_get_runtime_file("plugin/foo.vim", false)  -- first match only

-- exec lua / vimscript from Lua (rarely needed)
vim.api.nvim_exec_lua("return 1+1", {})
vim.api.nvim_eval("expand('%:p')")                 -- vimscript expr → lua
vim.api.nvim_command("write")                      -- run Ex command (use vim.cmd)
```

### Floating Windows

```lua
-- create a floating window
local buf = vim.api.nvim_create_buf(false, true)   -- (listed=false, scratch=true)
vim.api.nvim_buf_set_lines(buf, 0, -1, false, { "line 1", "line 2" })

local win = vim.api.nvim_open_win(buf, true, {     -- (buf, enter, config)
    relative = "editor",                           -- "editor"|"win"|"cursor"|"mouse"
    row      = math.floor(vim.o.lines / 2 - 5),
    col      = math.floor(vim.o.columns / 2 - 20),
    width    = 40,
    height   = 10,
    border   = "rounded",                          -- "none"|"single"|"double"|"rounded"|"solid"|"shadow"
    title    = " Hello ",
    title_pos = "center",
    footer   = " press q to close ",
    footer_pos = "right",
    style    = "minimal",                          -- no line numbers, statuscol etc
    anchor   = "NW",                               -- corner anchor
    focusable = true,
    zindex   = 50,
})

vim.api.nvim_win_set_config(win, { width = 60 })   -- update config
vim.api.nvim_win_close(win, true)                  -- (force=true)

-- close-on-q helper
vim.keymap.set("n", "q", "<cmd>close<CR>", { buffer = buf, silent = true })

-- LSP-style hover preview
vim.lsp.util.open_floating_preview(
    { "**Hello**", "More text" },
    "markdown",
    { border = "rounded", focus = false }
)
```

### Extmarks (Inline Decorations)

```lua
local ns = vim.api.nvim_create_namespace("my_decor")

-- virtual text after a line
vim.api.nvim_buf_set_extmark(0, ns, 5, 0, {
    virt_text = { { " ← here", "WarningMsg" } },
    virt_text_pos = "eol",                         -- "eol"|"overlay"|"right_align"|"inline"
})

-- virtual line ABOVE
vim.api.nvim_buf_set_extmark(0, ns, 5, 0, {
    virt_lines = { { { "▶ note", "Comment" } } },
    virt_lines_above = true,
})

-- highlight a range
vim.api.nvim_buf_set_extmark(0, ns, 0, 0, {
    end_row = 0, end_col = 5,
    hl_group = "Visual",
})

-- sign column
vim.api.nvim_buf_set_extmark(0, ns, 5, 0, {
    sign_text = ">",
    sign_hl_group = "DiagnosticSignError",
    number_hl_group = "DiagnosticError",
})

-- clear all marks in namespace
vim.api.nvim_buf_clear_namespace(0, ns, 0, -1)
```

## vim.fn — Vimscript Function Bridge

### The Rule

```lua
-- ANY function listed in :h func-list works as vim.fn.<Name>(...)
-- Lua-incompatible names use vim.fn['Name']
vim.fn.expand("%:p")                               -- absolute path of current buffer
vim.fn.expand("<cword>")                           -- word under cursor
vim.fn.expand("~")                                 -- home dir
vim.fn.fnamemodify("/a/b/c.txt", ":t")             -- "c.txt"
vim.fn.fnamemodify("/a/b/c.txt", ":h")             -- "/a/b"
vim.fn.fnamemodify("/a/b/c.txt", ":r")             -- "/a/b/c"
vim.fn.fnamemodify("/a/b/c.txt", ":e")             -- "txt"

vim.fn.getcwd()                                    -- pwd
vim.fn.chdir("/tmp")                               -- :cd
vim.fn.glob("*.go")                                -- shell glob → string
vim.fn.glob("*.go", false, true)                   -- glob → list

vim.fn.input("Name: ", "default")                  -- blocking prompt
vim.fn.confirm("Save?", "&Yes\n&No\n&Cancel", 1)
vim.fn.inputlist({ "Pick:", "1. one", "2. two" })

vim.fn.system("ls -1")                             -- run shell, return stdout (string)
vim.fn.systemlist("ls -1")                         -- same → list of lines
vim.fn.shellescape("'tricky path'")
vim.fn.has("nvim-0.10")                            -- 1 if version ≥ 0.10
vim.fn.has("mac")                                  -- 1 on macOS
vim.fn.has("unix")
vim.fn.has("win32")
vim.fn.executable("git")                           -- 1 if found in PATH

vim.fn.line(".")                                   -- current line number
vim.fn.col(".")                                    -- current column
vim.fn.getpos(".")                                 -- {bufnum, lnum, col, off}
vim.fn.setpos(".", { 0, 5, 1, 0 })

vim.fn.bufnr("%")                                  -- current buffer number
vim.fn.winnr()                                     -- current window number

-- vim.fn calls cross the lua/vimscript boundary — prefer vim.api or vim.* when possible
```

## vim.cmd — Ex Command Bridge

```lua
-- string form (legacy, fine for one-off)
vim.cmd("colorscheme tokyonight")
vim.cmd("write")
vim.cmd("source ~/.vimrc")

-- function-call form (modern, type-safe-ish)
vim.cmd.colorscheme("tokyonight")
vim.cmd.write()
vim.cmd.edit("foo.txt")
vim.cmd.help("autocmd")

-- multi-line via long bracket
vim.cmd[[
    syntax enable
    highlight Normal guibg=NONE ctermbg=NONE
    augroup my_highlights
        autocmd ColorScheme * highlight Comment gui=italic
    augroup END
]]

-- with bang / range / args
vim.cmd.bdelete({ args = { "5" }, bang = true })   -- :bdelete! 5
vim.cmd({ cmd = "substitute", args = { "/foo/bar/g" }, range = { 1, 10 } })

-- silent
vim.cmd.silent({ args = { "!touch /tmp/x" }, bang = true })
```

## vim.keymap.set

### Signature

```lua
-- vim.keymap.set(mode, lhs, rhs, opts?)
--   mode: string ("n"|"i"|"v"|"x"|"s"|"o"|"c"|"t"|"l"|"!"|"") OR list { "n", "v" }
--   lhs:  string keys (use "<leader>", "<C-x>", "<CR>")
--   rhs:  string Ex/cmd OR function callback
--   opts: { silent, noremap, expr, buffer, nowait, desc, callback, replace_keycodes, remap }

vim.keymap.set("n", "<leader>w", ":write<CR>",
    { silent = true, desc = "Save file" })

-- function callback form (cannot use string + callback simultaneously)
vim.keymap.set("n", "<leader>q", function()
    print("Quitting!")
    vim.cmd.quit()
end, { desc = "Quit" })

-- buffer-local (used heavily in LSP on_attach)
vim.keymap.set("n", "K", vim.lsp.buf.hover, { buffer = bufnr, desc = "Hover" })

-- multiple modes at once
vim.keymap.set({ "n", "v" }, "<leader>y", '"+y', { desc = "Yank to system" })

-- expr mapping (rhs is evaluated, must return keys)
vim.keymap.set("i", "<Tab>", function()
    return vim.fn.pumvisible() == 1 and "<C-n>" or "<Tab>"
end, { expr = true })

-- delete a mapping
vim.keymap.del("n", "<leader>w")
vim.keymap.del("n", "<leader>w", { buffer = bufnr })

-- noremap is TRUE by default in vim.keymap.set; set remap=true to allow remap
vim.keymap.set("n", "X", "<leader>x", { remap = true })

-- nowait — fire instantly even if a longer mapping exists
vim.keymap.set("n", "<leader>e", ":Explore<CR>", { nowait = true })
```

### Modes Reference

```bash
n   normal
i   insert
v   visual + select
x   visual only
s   select only
o   operator-pending     # after d, c, y etc
c   command-line
t   terminal
l   "lang-mode" (caret in insert)
""  normal+visual+operator (default :map)
!   insert + cmdline
```

## Auto-commands via Lua

### Canonical Pattern

```lua
-- the "no need for autocmd!" idiom — clear=true wipes the group on reload
local grp = vim.api.nvim_create_augroup("MyConfig", { clear = true })

-- yank highlight (one of the most copied snippets)
vim.api.nvim_create_autocmd("TextYankPost", {
    group = grp,
    desc  = "Highlight on yank",
    callback = function()
        vim.highlight.on_yank({ higroup = "IncSearch", timeout = 200 })
    end,
})

-- restore cursor on file open
vim.api.nvim_create_autocmd("BufReadPost", {
    group = grp,
    callback = function(args)
        local mark = vim.api.nvim_buf_get_mark(args.buf, '"')
        local lcount = vim.api.nvim_buf_line_count(args.buf)
        if mark[1] > 0 and mark[1] <= lcount then
            pcall(vim.api.nvim_win_set_cursor, 0, mark)
        end
    end,
})

-- per-filetype on FileType event (preferred over BufRead pattern)
vim.api.nvim_create_autocmd("FileType", {
    group   = grp,
    pattern = { "go", "rust", "c", "cpp" },
    callback = function()
        vim.opt_local.tabstop = 4
        vim.opt_local.shiftwidth = 4
        vim.opt_local.expandtab = false
    end,
})

-- close special buffers with q
vim.api.nvim_create_autocmd("FileType", {
    group   = grp,
    pattern = { "help", "qf", "lspinfo", "checkhealth", "man", "git" },
    callback = function(ev)
        vim.bo[ev.buf].buflisted = false
        vim.keymap.set("n", "q", "<cmd>close<CR>",
            { buffer = ev.buf, silent = true })
    end,
})
```

### Common Events

```bash
BufRead, BufReadPre, BufReadPost     # opening a file
BufNew, BufNewFile                   # new (unsaved) buffer
BufWrite, BufWritePre, BufWritePost  # saving
BufEnter, BufLeave                   # buffer focus
BufWinEnter, BufWinLeave             # buffer in window
BufHidden, BufDelete                 # buffer lifecycle
WinEnter, WinLeave, WinNew, WinClosed
TabEnter, TabLeave, TabNew, TabClosed
FileType                             # filetype detected
ColorScheme                          # :colorscheme run
VimEnter, VimLeavePre, VimLeave
VimResized
TermOpen, TermClose, TermEnter, TermLeave
InsertEnter, InsertLeave, InsertCharPre
TextChanged, TextChangedI
CursorHold, CursorHoldI              # uses 'updatetime'
CursorMoved, CursorMovedI
ModeChanged
LspAttach, LspDetach                 # LSP-specific (very useful)
LspProgress                          # streaming progress
DiagnosticChanged
User <event>                         # custom — fire via :doautocmd User MyEvent
```

## User Commands via Lua

```lua
-- basic
vim.api.nvim_create_user_command("Format", function()
    vim.lsp.buf.format({ async = true })
end, { desc = "LSP format buffer" })

-- with arguments
vim.api.nvim_create_user_command("Greet", function(opts)
    print("Hello, " .. opts.args .. "!")
    print("fargs:", vim.inspect(opts.fargs))       -- args split by whitespace
    print("range:", opts.range, opts.line1, opts.line2)
    print("count:", opts.count)
    print("bang:",  opts.bang)
end, {
    nargs    = "*",                                -- 0 | 1 | "*" | "?" | "+"
    range    = true,                               -- true | "%" | N
    count    = true,
    bang     = true,
    desc     = "Say hello",
    complete = function(arg_lead, cmdline, cursor_pos)
        return { "world", "earth", "moon" }
    end,
})
-- :Greet world earth     -> args="world earth", fargs={"world","earth"}
-- :%Format!              -> range=2, bang=true

-- buffer-local user command
vim.api.nvim_buf_create_user_command(0, "BufLocal", function() end, {})

-- delete
vim.api.nvim_del_user_command("Greet")
```

## Floating Windows

### Reusable Helper

```lua
local function popup(text, opts)
    opts = opts or {}
    local lines = type(text) == "string" and vim.split(text, "\n") or text
    local width = 0
    for _, l in ipairs(lines) do
        if #l > width then width = #l end
    end
    local height = #lines
    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
    vim.bo[buf].modifiable = false
    vim.bo[buf].bufhidden = "wipe"

    local win = vim.api.nvim_open_win(buf, true, {
        relative = "editor",
        row      = math.floor((vim.o.lines  - height) / 2),
        col      = math.floor((vim.o.columns - width) / 2),
        width    = width + 4,
        height   = height,
        border   = opts.border or "rounded",
        style    = "minimal",
        title    = opts.title,
        title_pos = "center",
    })
    vim.keymap.set("n", "q",     "<cmd>close<CR>", { buffer = buf, silent = true })
    vim.keymap.set("n", "<Esc>", "<cmd>close<CR>", { buffer = buf, silent = true })
    return win, buf
end
popup("Hello\nworld", { title = " greet " })
```

## lazy.nvim — The Modern Plugin Manager

### Bootstrap

```lua
-- canonical bootstrap (init.lua snippet)
local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not (vim.uv or vim.loop).fs_stat(lazypath) then
    vim.fn.system({
        "git", "clone", "--filter=blob:none", "--branch=stable",
        "https://github.com/folke/lazy.nvim.git", lazypath,
    })
end
vim.opt.rtp:prepend(lazypath)

require("lazy").setup("user.plugins")              -- loads lua/user/plugins/*.lua
-- or: require("lazy").setup({ ...specs... })
```

### Plugin Spec Anatomy

```lua
-- in lua/user/plugins/init.lua (or split across files)
return {
    -- string form: just a github "owner/repo"
    "tpope/vim-fugitive",

    -- full table form
    {
        "folke/tokyonight.nvim",
        lazy     = false,                          -- load at startup (themes usually)
        priority = 1000,                           -- load before others
        config   = function()
            require("tokyonight").setup({ style = "night" })
            vim.cmd.colorscheme("tokyonight")
        end,
    },

    -- LAZY-LOAD ON EVENT
    {
        "lewis6991/gitsigns.nvim",
        event = { "BufReadPre", "BufNewFile" },    -- list of events
        opts  = {                                  -- shorthand for setup(opts)
            current_line_blame = true,
        },
    },

    -- LAZY-LOAD ON CMD
    {
        "nvim-telescope/telescope.nvim",
        cmd = "Telescope",
    },

    -- LAZY-LOAD ON FILETYPE
    {
        "fatih/vim-go",
        ft = "go",
    },

    -- LAZY-LOAD ON KEYS — also creates the mapping
    {
        "phaazon/hop.nvim",
        keys = {
            { "<leader>j", "<cmd>HopWord<CR>", desc = "Hop word" },
            { "<leader>l", "<cmd>HopLine<CR>", desc = "Hop line", mode = { "n", "v" } },
        },
    },

    -- DEPENDENCIES
    {
        "nvim-telescope/telescope.nvim",
        dependencies = {
            "nvim-lua/plenary.nvim",
            { "nvim-telescope/telescope-fzf-native.nvim", build = "make" },
        },
    },

    -- BUILD STEP
    {
        "nvim-treesitter/nvim-treesitter",
        build = ":TSUpdate",                       -- run after install/update
    },

    -- VERSION PINS
    {
        "folke/lazy.nvim",
        version  = "*",                            -- latest tag
        -- version = "10.x",                       -- semver range
        -- branch  = "main",
        -- tag     = "v10.0.0",
        -- commit  = "abc1234",
    },

    -- DISABLE A PLUGIN CONDITIONALLY
    {
        "github/copilot.vim",
        enabled = function()
            return vim.fn.executable("node") == 1
        end,
    },

    -- INIT vs CONFIG
    {
        "L3MON4D3/LuaSnip",
        init = function()                          -- before plugin loads
            vim.g.snip_some_var = 1
        end,
        config = function()                        -- after plugin loads
            require("luasnip").setup({})
        end,
    },

    -- DEV / LOCAL PATH
    {
        dir  = "~/code/my-plugin",
        name = "my-plugin",
        dev  = true,
    },
}
```

### Lazy-Load Triggers

```lua
-- event       — autocmd events ("BufReadPre", "VeryLazy", "InsertEnter" ...)
-- cmd         — Ex command name(s)
-- ft          — filetype(s)
-- keys        — keymap definitions (also creates mapping)
-- VeryLazy    — special pseudo-event after UI is ready (for "load eventually")

-- defer until UI is ready (good for statusline, dashboards)
{ "nvim-lualine/lualine.nvim", event = "VeryLazy" }
```

### lazy.nvim Commands

```bash
:Lazy                 # open UI
:Lazy install         # install missing
:Lazy update          # update all
:Lazy sync            # install + update + clean
:Lazy clean           # remove orphans
:Lazy check           # check for updates without applying
:Lazy log             # commit log of last update
:Lazy restore         # restore lazy-lock.json state (reproducible)
:Lazy profile         # startup profiling
:Lazy debug           # debug active spec
:Lazy help            # built-in help
:Lazy reload <name>   # reload one plugin
:Lazy load <name>     # force-load a lazy plugin
```

### Lockfile

```bash
~/.config/nvim/lazy-lock.json     # commit hashes per plugin
# COMMIT this for reproducibility — restore via :Lazy restore
git -C ~/.config/nvim add lazy-lock.json && git commit -m "lazy lock"
```

## Plugin Spec Catalog

### Themes

```lua
{ "folke/tokyonight.nvim",   priority = 1000, lazy = false },
{ "catppuccin/nvim",         name     = "catppuccin", priority = 1000 },
{ "ellisonleao/gruvbox.nvim" },
{ "EdenEast/nightfox.nvim" },
{ "rebelot/kanagawa.nvim" },
{ "rose-pine/neovim",        name = "rose-pine" },
```

### UI

```lua
-- statusline
{ "nvim-lualine/lualine.nvim",
  dependencies = { "nvim-tree/nvim-web-devicons" } },

-- bufferline / tab-as-buffer
{ "akinsho/bufferline.nvim", version = "*",
  dependencies = "nvim-tree/nvim-web-devicons" },

-- cmdline / messages / popups overhaul (replaces cmd line at bottom)
{ "folke/noice.nvim",
  event = "VeryLazy",
  dependencies = { "MunifTanjim/nui.nvim", "rcarriga/nvim-notify" } },

-- replaces vim.ui.input/select with pretty UI (telescope picker)
{ "stevearc/dressing.nvim", event = "VeryLazy" },

-- icons
{ "nvim-tree/nvim-web-devicons" },                 -- nerdfont needed
{ "echasnovski/mini.icons", version = "*" },       -- alternative

-- notifications
{ "rcarriga/nvim-notify" },

-- which-key (popup of mappings)
{ "folke/which-key.nvim", event = "VeryLazy" },

-- todo-comments highlight
{ "folke/todo-comments.nvim",
  dependencies = "nvim-lua/plenary.nvim" },

-- color preview (#ff0000 → swatch)
{ "norcalli/nvim-colorizer.lua" },

-- indent guides
{ "lukas-reineke/indent-blankline.nvim", main = "ibl" },
```

### File Explorer

```lua
-- tree-style
{ "nvim-neo-tree/neo-tree.nvim",
  dependencies = {
    "nvim-lua/plenary.nvim",
    "nvim-tree/nvim-web-devicons",
    "MunifTanjim/nui.nvim",
  },
  cmd  = "Neotree",
  keys = { { "<leader>e", "<cmd>Neotree toggle<CR>" } } },
{ "nvim-tree/nvim-tree.lua",
  dependencies = "nvim-tree/nvim-web-devicons" },

-- buffer-as-directory (text-based, edit fs like a buffer)
{ "stevearc/oil.nvim",
  opts = {},
  dependencies = "nvim-tree/nvim-web-devicons" },
{ "echasnovski/mini.files", version = "*" },
```

### Fuzzy Finder

```lua
-- canonical
{ "nvim-telescope/telescope.nvim", branch = "0.1.x",
  dependencies = {
    "nvim-lua/plenary.nvim",
    { "nvim-telescope/telescope-fzf-native.nvim", build = "make" },
  } },

-- fastest, vim.ui.select replacement
{ "ibhagwan/fzf-lua",
  dependencies = "nvim-tree/nvim-web-devicons" },

-- snacks.nvim (newer, by folke — picker + many UI primitives)
{ "folke/snacks.nvim", priority = 1000, lazy = false, opts = {} },
```

### LSP

```lua
{ "neovim/nvim-lspconfig" },                       -- canonical config presets
{ "williamboman/mason.nvim", build = ":MasonUpdate" },     -- LSP/DAP installer
{ "williamboman/mason-lspconfig.nvim" },           -- bridges mason ↔ lspconfig
{ "WhoIsSethDaniel/mason-tool-installer.nvim" },   -- :MasonToolsInstall
-- progress UI (LSP "indexing…" notifications)
{ "j-hui/fidget.nvim", opts = {} },
-- inlay-hint helpers, code-action lightbulb, etc
{ "kosayoda/nvim-lightbulb" },
{ "ray-x/lsp_signature.nvim" },
```

### Completion

```lua
{ "hrsh7th/nvim-cmp",
  dependencies = {
    "hrsh7th/cmp-nvim-lsp",
    "hrsh7th/cmp-buffer",
    "hrsh7th/cmp-path",
    "hrsh7th/cmp-cmdline",
    "saadparwaiz1/cmp_luasnip",
    "L3MON4D3/LuaSnip",
    "rafamadriz/friendly-snippets",
    -- icons in completion items
    "onsails/lspkind.nvim",
  } },

-- alternative — blink.cmp (newer, fast, written in Rust)
{ "saghen/blink.cmp", version = "v0.*" },
```

### Treesitter

```lua
{ "nvim-treesitter/nvim-treesitter", build = ":TSUpdate" },
{ "nvim-treesitter/nvim-treesitter-textobjects" },
{ "nvim-treesitter/nvim-treesitter-context" },
{ "windwp/nvim-ts-autotag" },                      -- HTML/JSX auto-close
{ "JoosepAlviste/nvim-ts-context-commentstring" }, -- per-region comment chars
{ "nvim-treesitter/playground" },                  -- :TSPlaygroundToggle (debug)
```

### Git

```lua
{ "lewis6991/gitsigns.nvim", event = { "BufReadPre", "BufNewFile" } },
{ "tpope/vim-fugitive", cmd = { "G", "Git", "Gdiff", "Gblame" } },
{ "NeogitOrg/neogit", dependencies = "nvim-lua/plenary.nvim",
  cmd = "Neogit" },
{ "sindrets/diffview.nvim", cmd = { "DiffviewOpen", "DiffviewFileHistory" } },
```

### Editing

```lua
{ "kylechui/nvim-surround", event = "VeryLazy", opts = {} },     -- ysiw" cs"' ds"
{ "numToStr/Comment.nvim",  opts = {} },                          -- gcc gc
{ "echasnovski/mini.pairs", version = "*", event = "InsertEnter", opts = {} },
{ "windwp/nvim-autopairs",  event = "InsertEnter", opts = {} },   -- alt to mini
{ "ggandor/leap.nvim" },                                          -- s/S motion
{ "folke/flash.nvim", event = "VeryLazy", opts = {} },            -- newer leap
```

### Diagnostics / Quickfix

```lua
{ "folke/trouble.nvim", dependencies = "nvim-tree/nvim-web-devicons" },
{ "kevinhwang91/nvim-bqf" },                                      -- prettier qf
```

### DAP

```lua
{ "mfussenegger/nvim-dap" },
{ "rcarriga/nvim-dap-ui",  dependencies = { "mfussenegger/nvim-dap", "nvim-neotest/nvim-nio" } },
{ "jay-babu/mason-nvim-dap.nvim" },
{ "theHamsta/nvim-dap-virtual-text" },
{ "leoluz/nvim-dap-go" },                                         -- go-specific helper
{ "mfussenegger/nvim-dap-python" },
```

### Snippets

```lua
{ "L3MON4D3/LuaSnip", version = "v2.*", build = "make install_jsregexp" },
{ "rafamadriz/friendly-snippets" },                               -- shared corpus
```

### AI Assistants

```lua
{ "zbirenbaum/copilot.lua",                                       -- official Copilot, Lua
  cmd = "Copilot", event = "InsertEnter" },
{ "github/copilot.vim",                                           -- vim-script edition
  cmd = "Copilot", event = "InsertEnter" },
{ "Exafunction/codeium.nvim",
  dependencies = { "nvim-lua/plenary.nvim", "hrsh7th/nvim-cmp" } },
{ "yetone/avante.nvim", event = "VeryLazy",
  build = "make", opts = { provider = "claude" } },
{ "olimorris/codecompanion.nvim",
  dependencies = { "nvim-lua/plenary.nvim", "nvim-treesitter/nvim-treesitter" } },
```

### Notes / Markdown

```lua
{ "epwalsh/obsidian.nvim", version = "*", lazy = true, ft = "markdown" },
{ "renerocksai/telekasten.nvim", dependencies = "nvim-telescope/telescope.nvim" },
{ "MeanderingProgrammer/render-markdown.nvim",
  ft = { "markdown", "Avante" } },
{ "iamcco/markdown-preview.nvim", build = "cd app && yarn install",
  ft = "markdown" },
```

### Testing

```lua
{ "nvim-neotest/neotest",
  dependencies = {
    "nvim-lua/plenary.nvim",
    "nvim-treesitter/nvim-treesitter",
    "antoinemadec/FixCursorHold.nvim",
    "nvim-neotest/nvim-nio",
  } },
{ "vim-test/vim-test" },
```

### Sessions / Workflow

```lua
{ "rmagatti/auto-session" },
{ "folke/persistence.nvim", event = "BufReadPre", opts = {} },
{ "tpope/vim-sleuth" },                                           -- auto-detect ts/sw
{ "ThePrimeagen/harpoon", branch = "harpoon2",
  dependencies = "nvim-lua/plenary.nvim" },
```

## LSP — Native Setup

### Path A: nvim-lspconfig (works everywhere)

```lua
-- lua/user/config/lsp.lua
local lspconfig = require("lspconfig")
local cmp_caps  = require("cmp_nvim_lsp").default_capabilities()

local on_attach = function(client, bufnr)
    local map = function(mode, lhs, rhs, desc)
        vim.keymap.set(mode, lhs, rhs, { buffer = bufnr, desc = desc, silent = true })
    end
    map("n", "K",          vim.lsp.buf.hover,            "Hover")
    map("n", "gd",         vim.lsp.buf.definition,       "Goto definition")
    map("n", "gD",         vim.lsp.buf.declaration,      "Goto declaration")
    map("n", "gi",         vim.lsp.buf.implementation,   "Goto implementation")
    map("n", "gr",         vim.lsp.buf.references,       "References")
    map("n", "gy",         vim.lsp.buf.type_definition,  "Type definition")
    map("n", "<leader>rn", vim.lsp.buf.rename,           "Rename symbol")
    map({ "n", "v" }, "<leader>ca", vim.lsp.buf.code_action, "Code action")
    map("n", "<leader>f",  function() vim.lsp.buf.format({ async = true }) end, "Format")
    map("i", "<C-s>",      vim.lsp.buf.signature_help,   "Signature")

    -- inlay hints (0.10+)
    if client.supports_method and client.supports_method("textDocument/inlayHint") then
        vim.lsp.inlay_hint.enable(true, { bufnr = bufnr })
        map("n", "<leader>ih", function()
            vim.lsp.inlay_hint.enable(not vim.lsp.inlay_hint.is_enabled({ bufnr = bufnr }),
                                      { bufnr = bufnr })
        end, "Toggle inlay hints")
    end

    -- format on save (per-buffer)
    if client.supports_method and client.supports_method("textDocument/formatting") then
        local grp = vim.api.nvim_create_augroup("LspFormat" .. bufnr, { clear = true })
        vim.api.nvim_create_autocmd("BufWritePre", {
            group = grp, buffer = bufnr,
            callback = function() vim.lsp.buf.format({ bufnr = bufnr, async = false }) end,
        })
    end
end

lspconfig.gopls.setup({
    on_attach    = on_attach,
    capabilities = cmp_caps,
    settings = {
        gopls = {
            analyses = { unusedparams = true, shadow = true },
            staticcheck   = true,
            gofumpt       = true,
            usePlaceholders = true,
            completeUnimported = true,
        },
    },
})

lspconfig.lua_ls.setup({
    on_attach    = on_attach,
    capabilities = cmp_caps,
    settings = {
        Lua = {
            runtime    = { version = "LuaJIT" },
            diagnostics = { globals = { "vim" } },
            workspace  = {
                library = vim.api.nvim_get_runtime_file("", true),
                checkThirdParty = false,
            },
            telemetry  = { enable = false },
            completion = { callSnippet = "Replace" },
        },
    },
})
```

### Path B: vim.lsp.config / vim.lsp.enable (0.11+, no lspconfig needed)

```lua
-- 0.11+ declarative API — register a server
vim.lsp.config["gopls"] = {
    cmd      = { "gopls" },
    filetypes = { "go", "gomod", "gowork", "gotmpl" },
    root_markers = { "go.work", "go.mod", ".git" },
    settings = { gopls = { analyses = { unusedparams = true } } },
}
vim.lsp.enable({ "gopls", "lua_ls", "rust_analyzer" })

-- the on_attach pattern moves to LspAttach autocmd:
vim.api.nvim_create_autocmd("LspAttach", {
    callback = function(args)
        local bufnr  = args.buf
        local client = vim.lsp.get_client_by_id(args.data.client_id)
        if not client then return end
        vim.keymap.set("n", "K", vim.lsp.buf.hover, { buffer = bufnr })
        -- ...
    end,
})
```

### Path C: mason.nvim Auto-Install

```lua
require("mason").setup({
    ui = { border = "rounded", icons = { package_installed = "✓" } },
})
require("mason-lspconfig").setup({
    ensure_installed = { "lua_ls", "gopls", "rust_analyzer", "ts_ls", "pyright" },
    automatic_installation = true,
})

-- mason-tool-installer for non-LSP tools (formatters, linters, debuggers)
require("mason-tool-installer").setup({
    ensure_installed = {
        "stylua", "shfmt", "black", "isort", "prettier",
        "eslint_d", "shellcheck", "delve", "codelldb",
    },
})
```

### Useful LSP Commands

```bash
:LspInfo            # show attached clients (lspconfig)
:LspLog             # tail the LSP log file
:LspStart <name>    # manually start a server
:LspStop  <name>    # stop a server
:LspRestart         # restart all
:Mason              # mason UI: i install, X uninstall, U update
:checkhealth lspconfig
```

## LSP — Common Servers

### Filetype → Server Matrix

```bash
# language       lspconfig name        mason package          notes
# ----------     -------------------   --------------------   --------------------------
# Lua            lua_ls                lua-language-server    sumneko fork
# Go             gopls                 gopls
# Rust           rust_analyzer         rust-analyzer          install via rustup also OK
# TypeScript/JS  ts_ls (was tsserver)  typescript-language-server
# TypeScript     vtsls                 vtsls                  faster than ts_ls
# Python         pyright               pyright                types
# Python         basedpyright          basedpyright           pyright fork, more checks
# Python         ruff_lsp / ruff       ruff                   linter+formatter
# C/C++          clangd                clangd                 needs compile_commands.json
# CMake          cmake                 cmake-language-server
# JSON           jsonls                json-lsp
# YAML           yamlls                yaml-language-server
# TOML           taplo                 taplo
# Bash           bashls                bash-language-server
# Docker         dockerls              dockerfile-language-server
# Docker compose docker_compose_language_service
# Terraform      terraformls           terraform-ls
# HCL            terraformls           terraform-ls
# HTML           html                  html-lsp
# CSS / SCSS     cssls                 css-lsp
# Tailwind       tailwindcss           tailwindcss-language-server
# Vue            volar                 vue-language-server
# Svelte         svelte                svelte-language-server
# Nix            nil_ls                nil
# Nix            nixd                  nixd                   richer eval
# Markdown       marksman              marksman
# LaTeX          texlab                texlab
# SQL            sqls                  sqls
# Java           jdtls                 jdtls
# Kotlin         kotlin_language_server
# Solidity       solidity_ls
# Vimscript      vimls                 vim-language-server
# OCaml          ocamllsp
# Haskell        hls                   haskell-language-server
# Zig            zls                   zls
# Elixir         elixirls              elixir-ls
```

### Configuration Highlights

```lua
-- pyright + ruff combo (types + lint/format)
lspconfig.pyright.setup({
    on_attach = on_attach,
    settings = {
        python = { analysis = {
            typeCheckingMode = "basic",
            autoSearchPaths  = true,
            useLibraryCodeForTypes = true,
        }},
    },
})
lspconfig.ruff.setup({
    on_attach = function(client, bufnr)
        client.server_capabilities.hoverProvider = false  -- let pyright hover
        on_attach(client, bufnr)
    end,
})

-- rust_analyzer
lspconfig.rust_analyzer.setup({
    on_attach    = on_attach,
    capabilities = cmp_caps,
    settings = {
        ["rust-analyzer"] = {
            cargo  = { allFeatures = true },
            checkOnSave = { command = "clippy" },
            inlayHints = { lifetimeElisionHints = { enable = "always" } },
        },
    },
})

-- clangd
lspconfig.clangd.setup({
    on_attach = on_attach,
    cmd = {
        "clangd", "--background-index", "--clang-tidy",
        "--completion-style=detailed", "--header-insertion=iwyu",
    },
})
```

## LSP — Built-in Mappings

### Default Buffer Mappings (0.10+)

```bash
# fired by LspAttach automatically — no config needed
K        vim.lsp.buf.hover
gd       vim.lsp.buf.definition       # 0.11+ default
grn      vim.lsp.buf.rename           # 0.10+ default
grr      vim.lsp.buf.references       # 0.10+ default
gri      vim.lsp.buf.implementation   # 0.10+ default
gra      vim.lsp.buf.code_action      # 0.10+ default (n + v)
gO       vim.lsp.buf.document_symbol  # 0.11+ default
[d / ]d  vim.diagnostic.goto_prev/next
<C-W>d   vim.diagnostic.open_float    # 0.10+ default
<C-S>    vim.lsp.buf.signature_help   # insert mode, 0.11+
```

### Common API Calls

```lua
vim.lsp.buf.hover()
vim.lsp.buf.definition()
vim.lsp.buf.declaration()
vim.lsp.buf.implementation()
vim.lsp.buf.type_definition()
vim.lsp.buf.references()
vim.lsp.buf.document_symbol()
vim.lsp.buf.workspace_symbol("query")
vim.lsp.buf.rename()
vim.lsp.buf.code_action()
vim.lsp.buf.format({ async = true, timeout_ms = 2000 })
vim.lsp.buf.signature_help()
vim.lsp.buf.add_workspace_folder()
vim.lsp.buf.remove_workspace_folder()
vim.lsp.buf.list_workspace_folders()
vim.lsp.buf.incoming_calls()
vim.lsp.buf.outgoing_calls()

-- inlay hints (0.10+)
vim.lsp.inlay_hint.enable(true)
vim.lsp.inlay_hint.is_enabled()

-- get attached clients
local clients = vim.lsp.get_clients({ bufnr = 0 })
for _, c in ipairs(clients) do print(c.name) end

-- semantic tokens (0.9+)
vim.lsp.semantic_tokens.start(bufnr, client_id)
vim.lsp.semantic_tokens.stop(bufnr, client_id)
```

## vim.diagnostic API

```lua
-- configure global diagnostic UI
vim.diagnostic.config({
    virtual_text   = {                             -- inline messages
        prefix  = "●",
        spacing = 2,
        severity = { min = vim.diagnostic.severity.HINT },
    },
    -- virtual_lines = { current_line = true },    -- 0.11+ alt UI
    signs          = {                             -- gutter signs
        text = {
            [vim.diagnostic.severity.ERROR] = "",
            [vim.diagnostic.severity.WARN]  = "",
            [vim.diagnostic.severity.INFO]  = "",
            [vim.diagnostic.severity.HINT]  = "",
        },
    },
    underline      = true,
    update_in_insert = false,                      -- defer until InsertLeave
    severity_sort  = true,
    float = {
        border    = "rounded",
        source    = "if_many",
        header    = "",
        prefix    = "",
    },
})

-- navigation
vim.diagnostic.goto_next({ severity = vim.diagnostic.severity.ERROR })
vim.diagnostic.goto_prev()
vim.diagnostic.open_float({ scope = "line" })      -- "line"|"buffer"|"cursor"
vim.diagnostic.setqflist({ severity = vim.diagnostic.severity.WARN })
vim.diagnostic.setloclist()                        -- per-window loc list

-- toggle virtual_text on the fly
local function toggle_vt()
    local cfg = vim.diagnostic.config()
    vim.diagnostic.config({ virtual_text = not cfg.virtual_text })
end

-- programmatic — fetch current diagnostics
local diags = vim.diagnostic.get(0, { severity = { min = vim.diagnostic.severity.WARN }})
for _, d in ipairs(diags) do
    print(d.lnum, d.col, d.message, d.source)
end

-- severity values
-- vim.diagnostic.severity.ERROR = 1
-- vim.diagnostic.severity.WARN  = 2
-- vim.diagnostic.severity.INFO  = 3
-- vim.diagnostic.severity.HINT  = 4
```

## Treesitter

### Setup

```lua
require("nvim-treesitter.configs").setup({
    ensure_installed = {
        "lua", "vim", "vimdoc", "query",
        "go", "gomod", "gosum", "gowork",
        "python", "rust", "javascript", "typescript", "tsx",
        "json", "yaml", "toml", "markdown", "markdown_inline",
        "bash", "html", "css", "regex", "diff",
        "c", "cpp", "make", "cmake", "dockerfile",
    },
    sync_install = false,
    auto_install = true,                           -- install missing on open
    ignore_install = {},
    highlight = {
        enable  = true,
        disable = function(lang, bufnr)
            -- skip on huge files
            return vim.api.nvim_buf_line_count(bufnr) > 50000
        end,
        additional_vim_regex_highlighting = false,
    },
    indent = { enable = true, disable = { "yaml", "python" }},  -- python indent is buggy
    incremental_selection = {
        enable = true,
        keymaps = {
            init_selection    = "<C-Space>",
            node_incremental  = "<C-Space>",
            node_decremental  = "<BS>",
            scope_incremental = "<C-s>",
        },
    },
    textobjects = {                                -- requires nvim-treesitter-textobjects
        select = {
            enable = true,
            lookahead = true,
            keymaps = {
                ["af"] = "@function.outer",
                ["if"] = "@function.inner",
                ["ac"] = "@class.outer",
                ["ic"] = "@class.inner",
                ["aa"] = "@parameter.outer",
                ["ia"] = "@parameter.inner",
                ["al"] = "@loop.outer",
                ["il"] = "@loop.inner",
            },
        },
        move = {
            enable = true, set_jumps = true,
            goto_next_start     = { ["]f"] = "@function.outer", ["]c"] = "@class.outer" },
            goto_next_end       = { ["]F"] = "@function.outer" },
            goto_previous_start = { ["[f"] = "@function.outer", ["[c"] = "@class.outer" },
            goto_previous_end   = { ["[F"] = "@function.outer" },
        },
        swap = {
            enable = true,
            swap_next     = { ["<leader>a"] = "@parameter.inner" },
            swap_previous = { ["<leader>A"] = "@parameter.inner" },
        },
    },
})

-- folding via treesitter
vim.opt.foldmethod = "expr"
vim.opt.foldexpr   = "nvim_treesitter#foldexpr()"
vim.opt.foldenable = false                         -- start unfolded
```

### Treesitter Commands

```bash
:TSInstall <lang>           # install parser
:TSInstallSync <lang>       # blocking
:TSInstallInfo
:TSUninstall <lang>
:TSUpdate                   # update all
:TSUpdateSync
:TSBufEnable <module>       # e.g. highlight
:TSBufDisable <module>
:TSBufToggle <module>
:TSEnable highlight
:TSModuleInfo
:Inspect                    # show TS captures + hl groups under cursor (0.9+)
:InspectTree                # open syntax tree side-buffer (0.9+, replaces Playground)
:checkhealth nvim-treesitter
```

## Treesitter Queries

### .scm DSL

```scheme
; lua/queries/lua/highlights.scm — example
((identifier) @variable)

; capture children
(function_call
  name: (identifier) @function.call)

; pattern with predicate
((identifier) @keyword
  (#eq? @keyword "self"))

; alternation
[
  (true)
  (false)
] @boolean

; common predicates
; #eq?       — exact match
; #not-eq?
; #match?    — vim regex
; #any-of?   — list match
; #set!      — attach metadata (e.g. "priority" "120")
```

### Custom Queries

```bash
~/.config/nvim/queries/<lang>/highlights.scm     # extend built-in highlights
~/.config/nvim/queries/<lang>/textobjects.scm    # extend nvim-treesitter-textobjects

# tree-sitter inheritance order (the file is ADDED to the parser's queries):
# 1. parser's bundled queries
# 2. queries/<lang>/*.scm in runtimepath
# 3. user's after/queries/<lang>/*.scm

# debug a query interactively
:EditQuery                  # 0.10+ scratch query editor
:Inspect                    # captures under cursor

# install missing parser when filetype detected
:TSInstall <lang>
```

### Fallback to Regex

```lua
-- if no parser is installed, neovim silently falls back to regex syntax
-- to make it explicit:
require("nvim-treesitter.configs").setup({
    highlight = {
        enable = true,
        additional_vim_regex_highlighting = false,  -- avoid double-highlighting
    },
})
-- :TSBufDisable highlight  -- disable for current buffer (debug)
```

## Completion (nvim-cmp)

### Canonical Setup

```lua
-- lua/user/config/cmp.lua
local cmp     = require("cmp")
local luasnip = require("luasnip")
local lspkind = require("lspkind")

require("luasnip.loaders.from_vscode").lazy_load() -- friendly-snippets

local has_words_before = function()
    local line, col = unpack(vim.api.nvim_win_get_cursor(0))
    return col ~= 0 and vim.api.nvim_buf_get_lines(0, line - 1, line, true)[1]
        :sub(col, col):match("%s") == nil
end

cmp.setup({
    snippet = {
        expand = function(args) luasnip.lsp_expand(args.body) end,
    },
    window = {
        completion    = cmp.config.window.bordered(),
        documentation = cmp.config.window.bordered(),
    },
    mapping = cmp.mapping.preset.insert({
        ["<C-b>"]     = cmp.mapping.scroll_docs(-4),
        ["<C-f>"]     = cmp.mapping.scroll_docs(4),
        ["<C-Space>"] = cmp.mapping.complete(),
        ["<C-e>"]     = cmp.mapping.abort(),
        ["<CR>"]      = cmp.mapping.confirm({ select = false }),
        ["<Tab>"]     = cmp.mapping(function(fallback)
            if cmp.visible() then       cmp.select_next_item()
            elseif luasnip.expand_or_jumpable() then luasnip.expand_or_jump()
            elseif has_words_before() then cmp.complete()
            else fallback() end
        end, { "i", "s" }),
        ["<S-Tab>"]   = cmp.mapping(function(fallback)
            if cmp.visible() then cmp.select_prev_item()
            elseif luasnip.jumpable(-1) then luasnip.jump(-1)
            else fallback() end
        end, { "i", "s" }),
    }),
    sources = cmp.config.sources(
        {
            { name = "nvim_lsp", priority = 1000 },
            { name = "luasnip",  priority = 750  },
        },
        {
            { name = "buffer",   priority = 500, keyword_length = 3 },
            { name = "path",     priority = 250 },
        }
    ),
    formatting = {
        format = lspkind.cmp_format({
            mode = "symbol_text", maxwidth = 50, ellipsis_char = "...",
            menu = ({
                nvim_lsp = "[LSP]", luasnip = "[Snip]",
                buffer = "[Buf]",    path = "[Path]",
            }),
        }),
    },
    experimental = { ghost_text = true },
})

-- per-filetype source list
cmp.setup.filetype("gitcommit", {
    sources = cmp.config.sources({ { name = "git" } }, { { name = "buffer" } }),
})

-- cmdline completion
cmp.setup.cmdline({ "/", "?" }, {
    mapping = cmp.mapping.preset.cmdline(),
    sources = { { name = "buffer" } },
})
cmp.setup.cmdline(":", {
    mapping = cmp.mapping.preset.cmdline(),
    sources = cmp.config.sources(
        { { name = "path" } },
        { { name = "cmdline" } }
    ),
})
```

## Snippets (LuaSnip)

### Node Types

```lua
local ls = require("luasnip")
local s, t, i, c, d, f = ls.snippet, ls.text_node,
    ls.insert_node, ls.choice_node, ls.dynamic_node, ls.function_node

ls.add_snippets("go", {
    s("ifer", {                                    -- trigger
        t("if err != nil {"), t({ "", "\treturn " }),
        i(1, "err"),                               -- placeholder #1, default "err"
        t({ "", "}" }),
        i(0),                                      -- final cursor
    }),

    s("fn", {
        t("func "), i(1, "Name"), t("("), i(2), t(") "), i(3, "error"),
        t({ " {", "\t" }), i(0), t({ "", "}" }),
    }),

    s("test", {
        t("func Test"), i(1, "Name"), t("(t *testing.T) {"),
        t({ "", "\tt.Run(\"" }), c(2, { t("happy"), t("error"), t("edge") }), t("\", func(t *testing.T) {"),
        t({ "", "\t\t" }), i(0),
        t({ "", "\t})", "}" }),
    }),
})

-- load JSON snippets from friendly-snippets
require("luasnip.loaders.from_vscode").lazy_load()
require("luasnip.loaders.from_vscode").lazy_load({ paths = { "~/.config/nvim/snippets" }})

-- bindings
vim.keymap.set({ "i", "s" }, "<C-k>", function()
    if ls.expand_or_jumpable() then ls.expand_or_jump() end
end, { silent = true })
vim.keymap.set({ "i", "s" }, "<C-j>", function()
    if ls.jumpable(-1) then ls.jump(-1) end
end, { silent = true })
vim.keymap.set("i", "<C-l>", function()
    if ls.choice_active() then ls.change_choice(1) end
end)
```

## Telescope

### Built-in Pickers

```lua
local builtin = require("telescope.builtin")
vim.keymap.set("n", "<leader>ff", builtin.find_files, { desc = "Find files" })
vim.keymap.set("n", "<leader>fg", builtin.git_files,  { desc = "Find git files" })
vim.keymap.set("n", "<leader>/",  builtin.live_grep,  { desc = "Live grep" })
vim.keymap.set("n", "<leader>fb", builtin.buffers,    { desc = "Buffers" })
vim.keymap.set("n", "<leader>fh", builtin.help_tags,  { desc = "Help tags" })
vim.keymap.set("n", "<leader>fr", builtin.oldfiles,   { desc = "Recent files" })
vim.keymap.set("n", "<leader>fc", builtin.commands,   { desc = "Commands" })
vim.keymap.set("n", "<leader>fk", builtin.keymaps,    { desc = "Keymaps" })
vim.keymap.set("n", "<leader>fs", builtin.lsp_document_symbols, { desc = "Doc symbols" })
vim.keymap.set("n", "<leader>fS", builtin.lsp_dynamic_workspace_symbols, { desc = "WS symbols" })
vim.keymap.set("n", "<leader>fd", builtin.diagnostics, { desc = "Diagnostics" })
vim.keymap.set("n", "<leader>gs", builtin.git_status,  { desc = "Git status" })
vim.keymap.set("n", "<leader>gb", builtin.git_branches,{ desc = "Git branches" })
vim.keymap.set("n", "<leader>gc", builtin.git_commits, { desc = "Git commits" })
```

### Setup with FZF Native

```lua
local actions = require("telescope.actions")
require("telescope").setup({
    defaults = {
        prompt_prefix    = "  ",
        selection_caret  = " ",
        sorting_strategy = "ascending",
        layout_strategy  = "horizontal",
        layout_config    = {
            horizontal = { prompt_position = "top", preview_width = 0.55 },
            vertical   = { mirror = false },
        },
        file_ignore_patterns = { "%.git/", "node_modules/", "%.lock", "dist/", "build/" },
        mappings = {
            i = {
                ["<C-j>"] = actions.move_selection_next,
                ["<C-k>"] = actions.move_selection_previous,
                ["<C-q>"] = actions.send_to_qflist + actions.open_qflist,
                ["<Esc>"] = actions.close,
            },
        },
    },
    pickers = {
        find_files = { hidden = true, find_command = { "rg", "--files", "--hidden", "--glob=!.git" } },
    },
    extensions = {
        fzf = {                                    -- requires telescope-fzf-native build = "make"
            fuzzy = true, override_generic_sorter = true,
            override_file_sorter = true, case_mode = "smart_case",
        },
    },
})
require("telescope").load_extension("fzf")
```

### Custom Picker

```lua
local pickers   = require("telescope.pickers")
local finders   = require("telescope.finders")
local conf      = require("telescope.config").values
local actions   = require("telescope.actions")
local astate    = require("telescope.actions.state")

local function pick_color()
    pickers.new({}, {
        prompt_title = "Colors",
        finder = finders.new_table({ results = { "red", "green", "blue" } }),
        sorter = conf.generic_sorter({}),
        attach_mappings = function(prompt_bufnr, _)
            actions.select_default:replace(function()
                actions.close(prompt_bufnr)
                local sel = astate.get_selected_entry()
                vim.print("Picked: " .. sel[1])
            end)
            return true
        end,
    }):find()
end
```

## DAP — Debug Adapter Protocol

### Base Setup

```lua
local dap, dapui = require("dap"), require("dapui")
dapui.setup({})
dap.listeners.after.event_initialized["dapui_config"] = function() dapui.open() end
dap.listeners.before.event_terminated["dapui_config"] = function() dapui.close() end
dap.listeners.before.event_exited["dapui_config"]    = function() dapui.close() end

-- key bindings
local map = vim.keymap.set
map("n", "<F5>",       dap.continue,          { desc = "DAP continue" })
map("n", "<F10>",      dap.step_over,         { desc = "DAP step over" })
map("n", "<F11>",      dap.step_into,         { desc = "DAP step into" })
map("n", "<F12>",      dap.step_out,          { desc = "DAP step out" })
map("n", "<leader>b",  dap.toggle_breakpoint, { desc = "Toggle breakpoint" })
map("n", "<leader>B",  function() dap.set_breakpoint(vim.fn.input("Cond: ")) end,
                                              { desc = "Conditional breakpoint" })
map("n", "<leader>dr", dap.repl.open,         { desc = "DAP REPL" })
map("n", "<leader>du", dapui.toggle,          { desc = "DAP UI toggle" })
```

### Adapters

```lua
-- Go (delve)
dap.adapters.delve = {
    type = "server", port = "${port}",
    executable = { command = "dlv", args = { "dap", "-l", "127.0.0.1:${port}" }},
}
dap.configurations.go = {
    { type = "delve", name = "Debug",      request = "launch", program = "${file}" },
    { type = "delve", name = "Debug test", request = "launch", mode = "test", program = "${file}" },
    { type = "delve", name = "Debug pkg",  request = "launch", mode = "test",
      program = "./${relativeFileDirname}" },
}

-- Rust / C / C++ (codelldb)
dap.adapters.codelldb = {
    type = "server", port = "${port}",
    executable = { command = "codelldb", args = { "--port", "${port}" } },
}
for _, lang in ipairs({ "c", "cpp", "rust" }) do
    dap.configurations[lang] = {
        { type = "codelldb", name = "Launch", request = "launch",
          program = function()
              return vim.fn.input("Path to executable: ", vim.fn.getcwd() .. "/", "file")
          end,
          cwd = "${workspaceFolder}", stopOnEntry = false },
    }
end

-- Python (debugpy)
require("dap-python").setup("python3")             -- nvim-dap-python helper

-- Node / browser
dap.adapters["pwa-node"] = {
    type = "server", host = "localhost", port = "${port}",
    executable = { command = "node",
        args = { vim.fn.stdpath("data") .. "/mason/packages/js-debug-adapter/js-debug/src/dapDebugServer.js",
                 "${port}" }},
}
```

### Mason Bridge

```lua
require("mason-nvim-dap").setup({
    ensure_installed = { "delve", "codelldb", "debugpy", "js-debug-adapter" },
    automatic_installation = true,
    handlers = {},
})
```

## Statusline (lualine.nvim)

```lua
require("lualine").setup({
    options = {
        theme = "auto",
        component_separators = { left = "│", right = "│" },
        section_separators   = { left = "", right = "" },
        globalstatus         = true,               -- single statusline (laststatus=3)
        disabled_filetypes   = { statusline = { "dashboard", "alpha", "neo-tree" } },
    },
    sections = {
        lualine_a = { "mode" },
        lualine_b = { "branch", "diff",
                      { "diagnostics", sources = { "nvim_diagnostic" }} },
        lualine_c = {
            { "filename", path = 1, symbols = { modified = "●", readonly = "" }},
        },
        lualine_x = { "encoding", "fileformat", "filetype" },
        lualine_y = { "progress" },
        lualine_z = { "location" },
    },
    extensions = { "fugitive", "neo-tree", "lazy", "trouble", "quickfix" },
})

-- custom component example: LSP server names
local function lsp_status()
    local clients = vim.lsp.get_clients({ bufnr = 0 })
    if #clients == 0 then return "" end
    local names = {}
    for _, c in ipairs(clients) do table.insert(names, c.name) end
    return " " .. table.concat(names, ", ")
end
-- include in section: lualine_x = { lsp_status, "filetype" }
```

## Buffer-line / Tabline (bufferline.nvim)

```lua
require("bufferline").setup({
    options = {
        mode = "buffers",                          -- treat each buffer as a tab
        diagnostics = "nvim_lsp",
        diagnostics_indicator = function(_, _, diag)
            return (diag.error and (" ✘ " .. diag.error) or "")
                .. (diag.warning and (" ⚠ " .. diag.warning) or "")
        end,
        offsets = {
            { filetype = "neo-tree", text = "Files", text_align = "center", separator = true },
        },
        show_buffer_close_icons = true,
        show_close_icon = false,
        always_show_bufferline = true,
    },
})

vim.keymap.set("n", "<S-h>", "<cmd>BufferLineCyclePrev<CR>", { desc = "Prev buffer" })
vim.keymap.set("n", "<S-l>", "<cmd>BufferLineCycleNext<CR>", { desc = "Next buffer" })
vim.keymap.set("n", "<leader>bp","<cmd>BufferLineTogglePin<CR>", { desc = "Pin buffer" })
vim.keymap.set("n", "<leader>bo","<cmd>BufferLineCloseOthers<CR>",{ desc = "Close others" })
```

## Git Integration (gitsigns.nvim)

```lua
require("gitsigns").setup({
    signs = {
        add          = { text = "▎" },
        change       = { text = "▎" },
        delete       = { text = "" },
        topdelete    = { text = "" },
        changedelete = { text = "▎" },
        untracked    = { text = "▎" },
    },
    signcolumn = true,
    numhl      = false,
    linehl     = false,
    word_diff  = false,
    current_line_blame = false,
    current_line_blame_opts = { delay = 200, virt_text_pos = "eol" },
    on_attach = function(bufnr)
        local gs  = package.loaded.gitsigns
        local map = function(mode, l, r, desc)
            vim.keymap.set(mode, l, r, { buffer = bufnr, desc = desc })
        end
        -- navigation
        map("n", "]c", function()
            if vim.wo.diff then return "]c" end
            vim.schedule(function() gs.next_hunk() end)
            return "<Ignore>"
        end, "Next hunk")
        map("n", "[c", function()
            if vim.wo.diff then return "[c" end
            vim.schedule(function() gs.prev_hunk() end)
            return "<Ignore>"
        end, "Prev hunk")
        -- actions
        map({ "n", "v" }, "<leader>hs", "<cmd>Gitsigns stage_hunk<CR>",  "Stage hunk")
        map({ "n", "v" }, "<leader>hr", "<cmd>Gitsigns reset_hunk<CR>",  "Reset hunk")
        map("n", "<leader>hS", gs.stage_buffer,                          "Stage buffer")
        map("n", "<leader>hu", gs.undo_stage_hunk,                       "Undo stage")
        map("n", "<leader>hR", gs.reset_buffer,                          "Reset buffer")
        map("n", "<leader>hp", gs.preview_hunk,                          "Preview hunk")
        map("n", "<leader>hb", function() gs.blame_line({ full = true }) end, "Blame line")
        map("n", "<leader>tb", gs.toggle_current_line_blame,             "Toggle blame")
        map("n", "<leader>hd", gs.diffthis,                              "Diff this")
        map("n", "<leader>hD", function() gs.diffthis("~") end,          "Diff vs HEAD~")
        map("n", "<leader>td", gs.toggle_deleted,                        "Toggle deleted")
        -- text object
        map({ "o", "x" }, "ih", "<cmd>Gitsigns select_hunk<CR>",          "Hunk text object")
    end,
})
```

### Highlight Groups

```bash
GitSignsAdd, GitSignsChange, GitSignsDelete, GitSignsTopdelete,
GitSignsChangedelete, GitSignsUntracked
GitSignsAddNr, GitSignsChangeNr, GitSignsDeleteNr,
GitSignsAddLn, GitSignsChangeLn, GitSignsDeleteLn,
GitSignsCurrentLineBlame
```

## Filetype-specific Setup

### ftdetect

```lua
-- ~/.config/nvim/ftdetect/myext.lua
vim.filetype.add({
    extension = {
        myext = "myft",
        gotmpl = "gotmpl",
    },
    filename = {
        Brewfile = "ruby",
        ["docker-compose.yml"] = "yaml.docker-compose",
    },
    pattern = {
        [".*%.config/git/config"] = "gitconfig",
        ["${HOME}/%.config/foo/.*"] = "json",
    },
})
```

### ftplugin

```lua
-- ~/.config/nvim/after/ftplugin/python.lua  (after/ overrides bundled ftplugin)
vim.opt_local.tabstop     = 4
vim.opt_local.shiftwidth  = 4
vim.opt_local.expandtab   = true
vim.opt_local.colorcolumn = "88"

-- buffer-local mapping
vim.keymap.set("n", "<localleader>r", "<cmd>!python3 %<CR>", { buffer = 0 })
```

### Autocmd-based Alternative

```lua
vim.api.nvim_create_autocmd("FileType", {
    pattern = "go",
    callback = function()
        vim.opt_local.expandtab = false
        vim.opt_local.tabstop   = 4
        vim.opt_local.shiftwidth = 4
    end,
})
```

## Health Check

```bash
:checkhealth                 # full system report
:checkhealth nvim            # core
:checkhealth provider        # python/node/perl/ruby providers
:checkhealth lsp             # 0.11+
:checkhealth lspconfig
:checkhealth nvim-treesitter
:checkhealth mason
:checkhealth telescope
:checkhealth nvim-cmp
:checkhealth lazy
:checkhealth vim.lsp         # log location, attached clients

# Common "missing" complaints and fixes
# - "Python 3 provider (optional)" missing  → pip install pynvim
# - "Node provider (optional)" missing       → npm i -g neovim
# - "ripgrep" not found (telescope live_grep)→ brew/apt install ripgrep
# - "fd" not found (telescope find_files)    → brew/apt install fd / fd-find
# - "tree-sitter CLI" missing                → cargo install tree-sitter-cli
#                                              or :TSInstallSync (uses bundled)
# - "git" not found                          → install git
# - "compile_commands.json" missing (clangd) → bear -- make / cmake -DCMAKE_EXPORT_COMPILE_COMMANDS=1
```

## Common Errors and Fixes

### Lua / require Errors

```bash
# ERROR: module 'foo' not found
#   E5108: Error executing lua [string ":lua"]:1: module 'foo' not found:
#       no field package.preload['foo']
#       no file './foo.lua'
#       no file '/usr/local/share/luajit-2.1/foo.lua'
#       ...
# CAUSE:    file not in runtimepath OR plugin not installed/loaded
# FIX:      :Lazy install ; :scriptnames ; :=vim.api.nvim_get_runtime_file("lua/foo*", true)
#           confirm path is lua/foo.lua under a runtimepath dir

# ERROR: E5108: Error executing lua [string ":lua"]:1: attempt to index a nil value (field 'foo')
# CAUSE:    `local x = require("bar"); x.foo.bar()` — module returned nil OR foo not a table
# FIX:      check spelling, that the module returns `M`, that it loaded:
#           :lua print(vim.inspect(require("bar")))
```

### LSP Errors

```bash
# ERROR: vim.lsp.buf.format(): attempt to call method 'format' (a nil value)
# CAUSE:    on_attach didn't run — server not actually attached to buffer
# FIX:      :LspInfo (is the server attached?); :checkhealth lsp; check filetype detection
#           confirm file matches server's `filetypes` config

# ERROR: LSP[gopls] timeout error reading message: connection closed
# CAUSE:    server crashed (panic, OOM, version mismatch)
# FIX:      :LspLog ; check ~/.local/state/nvim/lsp.log ; :LspRestart ; rebuild server
#           gopls: `go install golang.org/x/tools/gopls@latest`

# ERROR: client_id <N> sent invalid response: ...
# CAUSE:    server sent malformed JSON-RPC
# FIX:      update server; pin to a working version

# ERROR: No client attached
# CAUSE:    server failed to spawn
# FIX:      run server binary manually from terminal; check PATH;
#           :lua =vim.fn.executable("gopls")  -- 1 if reachable

# ERROR: server stopped, exiting [unable to find source X]
# CAUSE:    rust-analyzer can't locate Cargo workspace
# FIX:      open at workspace root; ensure Cargo.toml exists
```

### Treesitter Errors

```bash
# ERROR: Treesitter: language 'go' not installed
# CAUSE:    parser not present
# FIX:      :TSInstall go      or :TSInstallSync go
#           ensure_installed in setup is the long-term fix

# ERROR: ABI version mismatch for parser
# CAUSE:    parser compiled for older neovim ABI
# FIX:      :TSUpdate

# ERROR: query: invalid node type at position ...
# CAUSE:    custom highlights.scm references node missing in current grammar
# FIX:      check tree via :InspectTree; align with grammar version
```

### Configuration Errors

```bash
# ERROR: Error executing lua: ...vim/_meta/options.lua: 'X' is not a valid option name
# CAUSE:    typo OR removed option OR version-specific
# FIX:      :help 'X' ; :help options-removed ; check nvim --version

# ERROR: E5113: Error while calling lua chunk: ...
# CAUSE:    init.lua syntax/runtime error
# FIX:      :messages to read full traceback ; :luafile % to reload current file

# ERROR: source $MYVIMRC fails silently, no errors
# CAUSE:    init.lua errors caught and suppressed
# FIX:      nvim --headless "+luafile $MYVIMRC" +qa  (will print error)
#           or `:lua dofile(vim.env.MYVIMRC)`

# ERROR: Plug not loaded (lazy.nvim never bootstrapped)
# CAUSE:    bootstrap snippet missing or path wrong
# FIX:      :lua print(vim.fn.stdpath("data") .. "/lazy/lazy.nvim")
#           verify directory exists
```

### Plugin / Mason Errors

```bash
# ERROR: failed to download package <name>
# CAUSE:    no network / missing tools (npm, cargo, go)
# FIX:      :checkhealth mason ; install required toolchain

# ERROR: server <name> not configured (mason-lspconfig)
# CAUSE:    server name not in lspconfig list
# FIX:      use the lspconfig name (gopls not "go-language-server")

# ERROR: lazy.nvim: spec key 'X' is invalid
# CAUSE:    typo in plugin spec
# FIX:      see :h lazy.nvim-spec ; double-check key names
```

## Common Gotchas

### Clipboard

```bash
# BAD: linux without clipboard provider — yank to + register silently fails
:lua vim.fn.setreg("+", "test")    # returns success but nothing in system clipboard
:checkhealth provider               # reports "no clipboard tool available"

# FIX: install ONE of
sudo apt install xclip              # X11
sudo apt install xsel               # X11 alternative
sudo apt install wl-clipboard       # Wayland (wl-copy / wl-paste)

# verify
:lua print(vim.fn.has("clipboard"))  -- 1
:lua vim.opt.clipboard = "unnamedplus"
```

### Truecolor

```bash
# BAD: vim.opt.termguicolors = true on a terminal without truecolor → washed colors
echo $COLORTERM                     # should be "truecolor" or "24bit"

# FIX: set conditionally
if vim.env.COLORTERM == "truecolor" or vim.env.COLORTERM == "24bit" then
    vim.opt.termguicolors = true
else
    vim.opt.termguicolors = false   -- fall back to 256 colors
end

# tmux must opt-in (in ~/.tmux.conf):
#   set-option -sa terminal-overrides ",xterm-256color:RGB"
```

### lazy.nvim "true lazy load"

```bash
# BAD — loads on startup even though we set lazy = true
{
    "X/Y",
    lazy   = true,
    config = require("Y").setup({}),    -- WRONG: require runs IMMEDIATELY at parse time
}

# FIX — wrap require in a function
{
    "X/Y",
    event  = "VeryLazy",
    config = function()
        require("Y").setup({})           -- now deferred
    end,
}

# or use opts shorthand (lazy auto-calls setup(opts))
{ "X/Y", event = "VeryLazy", opts = {} }
```

### Legacy nvim_set_keymap

```bash
# BAD — does NOT accept Lua callbacks; rhs must be a string
vim.api.nvim_set_keymap("n", "<leader>q",
    function() print("hi") end,            -- IGNORED, treated as nil
    { noremap = true })

# FIX — use vim.keymap.set
vim.keymap.set("n", "<leader>q",
    function() print("hi") end,
    { desc = "say hi" })
```

### vim.cmd("set …")

```bash
# BAD — slow + opaque, no Lua type help
vim.cmd("set tabstop=4 shiftwidth=4 expandtab")

# FIX — vim.opt.* (one option per line so diffs are clean)
vim.opt.tabstop    = 4
vim.opt.shiftwidth = 4
vim.opt.expandtab  = true
```

### LuaRocks

```bash
# BAD — plugin spec says "needs luarocks"; manually pinning rocks gets messy
# many plugins NOW require lua-rocks (e.g. older versions of telescope-fzy-native)

# FIX 1 — prefer pure-lua alternatives (telescope-fzf-native, blink.cmp)
# FIX 2 — let lazy handle it (rocks support landed in lazy 11+)
require("lazy").setup({
    rocks = { hererocks = true },        # auto-install hererocks for lua deps
    ...
})
```

### Treesitter Silent Fallback

```bash
# BAD — no parser installed, neovim falls back to vim regex highlights silently
# you THINK you have TS highlights but it's actually old-school syntax/

# FIX — be explicit
require("nvim-treesitter.configs").setup({
    ensure_installed = { "lua", "go", "python", "rust" },
    auto_install     = true,            -- install as you open new filetypes
    highlight = {
        enable = true,
        additional_vim_regex_highlighting = false,   -- don't double-up
    },
})
:TSModuleInfo                            -- confirm highlights enabled per buffer
```

### apt-installed LSP Not Detected

```bash
# BAD — `apt install gopls` ; nvim says "No LSP attached"
# CAUSE — distro binary in /usr/bin but the wrong version; OR not in nvim's PATH

# FIX — use mason.nvim (puts everything in $XDG_DATA_HOME/nvim/mason/bin and adds to PATH)
:Mason
# install via UI; lspconfig auto-detects mason path
:LspInfo                                 -- verify command field points to mason install
```

### Wrong Filetype on Open

```bash
# BAD — open foo.tf, filetype is "" because vim doesn't know
# CAUSE — filetype detection rule missing

# FIX — add to ftdetect/
vim.filetype.add({ extension = { tf = "terraform", tfvars = "terraform" } })
:set filetype?                           -- check; :set filetype=terraform to force
```

### autocmd Double-Fire

```bash
# BAD — reloading init.lua adds a second autocmd of the same definition
:autocmd                                 -- shows duplicates

# FIX — group with clear=true (idiomatic since 0.7)
local grp = vim.api.nvim_create_augroup("MyGroup", { clear = true })   -- wipes prior
vim.api.nvim_create_autocmd("BufWritePre", { group = grp, callback = ... })
```

## Performance Tips

### vim.loader

```lua
-- 0.9+: bytecode cache for require()  → dramatically faster startup
vim.loader.enable()
-- (call BEFORE any require — typically first line of init.lua)
```

### lazy.nvim Profiling

```bash
:Lazy profile        # timeline of plugin load
# typical target: <50ms total startup with 100+ plugins
# any plugin >5ms on load needs investigation

nvim --startuptime startup.log
sort -k2 -n startup.log | tail -50
```

### Treesitter Surgery

```lua
-- disable highlight on huge files
require("nvim-treesitter.configs").setup({
    highlight = {
        enable  = true,
        disable = function(_, bufnr)
            local ok, stats = pcall(vim.uv.fs_stat,
                vim.api.nvim_buf_get_name(bufnr))
            return ok and stats and stats.size > 100 * 1024  -- 100 KB cap
        end,
    },
})
```

### Disable Built-in Plugins

```lua
-- in lazy.nvim setup{}
performance = {
    rtp = {
        disabled_plugins = {
            "gzip", "matchit", "matchparen", "netrwPlugin",
            "tarPlugin", "tohtml", "tutor", "zipPlugin",
            "spellfile_plugin", "rplugin",
        },
    },
},
```

### Lazy-Load Triggers

```lua
-- prefer event-based over `lazy = true` alone (nothing to trigger it)
{ "lewis6991/gitsigns.nvim", event = "BufReadPre" }      -- not at startup
{ "windwp/nvim-autopairs",   event = "InsertEnter" }     -- not until typing
{ "folke/which-key.nvim",    event = "VeryLazy" }        -- after UI ready
```

### Rendering / UI

```lua
vim.opt.lazyredraw   = false         -- DO NOT set true with noice/cmp (breaks UI)
vim.opt.synmaxcol    = 240
vim.opt.redrawtime   = 1500
vim.opt.updatetime   = 200
vim.opt.timeoutlen   = 300
```

### Profiling

```bash
# built-in startup profile
nvim --startuptime st.log +q && cat st.log

# function-level lua profile (plugin: profile.nvim)
:lua require("profile").start("*")
:lua require("profile").stop()
:lua require("profile").export("/tmp/prof.json")
# view in chrome://tracing or speedscope.app
```

## Migration from Vim

### Incremental Path

```lua
-- step 1 — keep init.vim alongside, source from init.lua
-- ~/.config/nvim/init.lua
vim.cmd("source " .. vim.fn.stdpath("config") .. "/init.vim.bak")
-- now Lua runs, vimscript still loads

-- step 2 — translate one section at a time
-- vimscript:                     lua:
-- let g:foo = 1                  vim.g.foo = 1
-- set number                     vim.opt.number = true
-- nnoremap j gj                  vim.keymap.set("n", "j", "gj")
-- autocmd BufRead *.go set ts=4  see autocommands section

-- step 3 — plugins still work via lazy.nvim with `dir`
{ dir = "~/.vim/plugged/vim-fugitive", lazy = false }

-- step 4 — switch package manager
-- vim-plug → lazy.nvim (one-shot via :PlugClean then bootstrap)
```

### Option Mapping

```lua
-- common let g:* → vim.g.*
-- vimscript                       lua
-- let g:loaded_python_provider=0  vim.g.loaded_python_provider = 0
-- let g:python3_host_prog="..."   vim.g.python3_host_prog = "/usr/bin/python3"
-- let g:netrw_banner=0            vim.g.netrw_banner = 0

-- common variables
vim.g                             -- global g:vars
vim.b                             -- buffer b:vars (vim.b[bufnr])
vim.w                             -- window w:vars
vim.t                             -- tab    t:vars
vim.v                             -- vim    v:vars (read-only mostly)
vim.env                           -- environment $VARS
```

### Vimscript Functions Inside Lua

```lua
-- if you really need to call vimscript (e.g. legacy plugin's API)
vim.fn.MyVimscriptFunc(arg1, arg2)
-- bracket-name access for non-Lua-friendly names
vim.fn["s:LocalFunc"]              -- script-local — NOT callable from outside
vim.fn["plugin#namespace#Foo"](arg)
```

## Migration from VS Code / Other Editors

### Modal Shock — Day 1 Survival

```bash
# the three modes
i / a / o    enter insert mode (a = after cursor, o = open new line)
<Esc>        return to normal mode  (or <C-[> or <C-c>)
v            character visual
V            line visual
<C-v>        block visual

# essential motions
hjkl         left/down/up/right (arrows work but learn hjkl)
w / b        next / previous word
0 / $        line start / end
^            first non-blank
gg / G       file start / end
{ / }        paragraph back / forward
%           matching bracket
*           search word under cursor

# essential operators (operator + motion = action)
d            delete    →  dw, d$, dd (whole line)
c            change    →  cw, cc
y            yank      →  yw, yy
p / P        paste after / before
u            undo
<C-r>        redo
.            repeat last change

# the "leader" key (default \ ; configured to space here) prefixes user mappings
```

### Starter Configs (Pick One, Don't Copy Many)

```bash
# kickstart.nvim — single-file ~600-line teaching config
#   pros: read EVERY line, understand every choice, customize freely
#   cons: you build the rest yourself
#   git clone https://github.com/nvim-lua/kickstart.nvim ~/.config/nvim

# LazyVim — opinionated full-stack distribution
#   pros: works out of the box, fast, well-maintained
#   cons: opinionated; harder to dig into
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/LazyVim/starter/main/install.sh)"

# AstroNvim — community-driven, plugin-rich
#   pros: lots of "just works" features (UI heavy)
#   cons: many layers of indirection

# NvChad — minimal, fast, theme-focused
#   pros: very fast, pretty
#   cons: heavy custom abstractions over lazy.nvim

# LunarVim — Lua-first, language-server aware
#   pros: bundled lvim CLI for management
#   cons: heavyweight install
```

### Recommended Path

```bash
# 1. Use kickstart.nvim for 1-2 weeks (read it cover to cover)
# 2. Fork into your own dotfiles, prune what you don't need
# 3. Add plugins one at a time (commit between each)
# 4. Resist the temptation to install distros; they hide what's happening
```

## Idioms

### The Lua Module Pattern

```lua
-- ~/.config/nvim/lua/user/utils.lua
local M = {}                                       -- public table

local cache = {}                                   -- private

function M.find_root(markers, start)
    start = start or vim.fn.expand("%:p:h")
    local found = vim.fs.find(markers, { upward = true, path = start })[1]
    return found and vim.fs.dirname(found) or nil
end

function M.toggle(opt)
    vim.opt[opt] = not vim.opt[opt]:get()
    vim.notify(opt .. " = " .. tostring(vim.opt[opt]:get()))
end

return M

-- usage:  require("user.utils").toggle("number")
```

### Augroup + autocmd

```lua
local grp = vim.api.nvim_create_augroup("UserFmt", { clear = true })
vim.api.nvim_create_autocmd("BufWritePre", {
    group = grp, pattern = { "*.lua", "*.go" },
    callback = function() vim.lsp.buf.format({ async = false }) end,
})
```

### Deferred Work

```lua
-- vim.schedule — run on next event-loop tick (use after fast-event)
vim.schedule(function() vim.notify("hi") end)

-- vim.defer_fn — run after ms
vim.defer_fn(function() print("delayed") end, 500)

-- vim.uv.new_timer for repeating
local t = vim.uv.new_timer()
t:start(0, 1000, vim.schedule_wrap(function() print("tick") end))
-- t:stop(); t:close()
```

### Safe Require

```lua
local ok, mod = pcall(require, "maybe-missing")
if not ok then return end
mod.setup({})

-- or
local function safe_require(name)
    local ok, m = pcall(require, name)
    if not ok then vim.notify("failed: " .. name, vim.log.levels.WARN) end
    return ok and m or nil
end
```

### vim.notify

```lua
vim.notify("hello")
vim.notify("error happened", vim.log.levels.ERROR)
vim.notify("warn", vim.log.levels.WARN, { title = "User" })

-- override with nvim-notify for pretty popups
vim.notify = require("notify")
```

### Buffer-Local Helpers

```lua
local function buf_set(bufnr, opt, val)
    vim.api.nvim_set_option_value(opt, val, { buf = bufnr })
end
local function win_set(winid, opt, val)
    vim.api.nvim_set_option_value(opt, val, { win = winid })
end
```

## Tips

### Daily Maintenance

```bash
:checkhealth          # quick "is everything OK"
:Lazy                 # one-key (U) to update plugins
:Mason update         # update LSPs/formatters
:TSUpdate             # update treesitter parsers
```

### Workflow Cheat-Sheet

```bash
K            hover info on symbol under cursor (LSP)
gd           goto definition
gri          goto implementation                (0.10 default)
grr          references                         (0.10 default)
grn          rename symbol                      (0.10 default)
gra          code action                        (0.10 default)
[d / ]d      next / prev diagnostic
<C-W>d       diagnostic float

<leader>ff   find file (telescope)
<leader>/    live grep
<leader>fb   buffers

<C-^> / <C-6>  alternate buffer
:b<n>          jump to buffer N
:bd            delete buffer
:bp / :bn      prev / next buffer

<C-w>v        vertical split
<C-w>s        horizontal split
<C-w>w        cycle windows
<C-w>q        quit window

zR / zM       open / close all folds
zf / zo / zc  create / open / close fold
za            toggle fold

q<letter>     start macro into register
@<letter>     play macro
@@           replay last macro

:'<,'>norm    apply normal-mode keys to visual selection
:%s/foo/bar/g substitute file-wide
:cdo / :cfdo  apply Ex command to each quickfix entry
```

### Ad-hoc Lua

```bash
:lua = expr         # print expr (= prefix is shorthand for print)
:lua =vim.bo.filetype
:lua =vim.lsp.get_clients()
:lua vim.opt.number = false
:luafile %          # source current file as lua
:source %           # source current file (auto-detect)
:!lua -e "..."      # external lua oneliner
```

### Useful Built-ins

```bash
:%!sort                # pipe whole buffer through sort
:%!jq .               # format JSON via jq
:read !date           # insert command output below cursor
:terminal             # open shell in buffer (Esc <C-\><C-n> exits term-mode)
:te lazygit
:Inspect              # show TS captures + hl groups under cursor (0.9+)
:Telescope highlights # find a highlight group
:messages             # last messages (also g<)
:scriptnames          # files sourced this session
:verbose set foo?     # where was 'foo' last set
:verbose map <leader>w # where was the mapping defined
```

### .editorconfig Support

```bash
# neovim 0.9+ has built-in .editorconfig support
:lua print(vim.g.editorconfig)        # true if enabled
# disable per-buffer: vim.b.editorconfig = false
```

### NVIM_APPNAME Trick

```bash
# run a fully separate config (great for testing)
NVIM_APPNAME=nvim-trial nvim
# uses ~/.config/nvim-trial, ~/.local/share/nvim-trial, etc

# alias for permanent split
alias nvtrial='NVIM_APPNAME=nvim-trial nvim'
```

### SSH / Remote

```bash
# one-shot edit a remote file via scp
nvim scp://user@host//etc/nginx/nginx.conf

# proper remote-server (0.9+ wip / 0.11): vim.lsp + vim.fs.exec_remote()
# practical alternative — sshfs mount
sshfs user@host:/ /mnt/host
nvim /mnt/host/etc/nginx/nginx.conf
```

### Headless Scripting

```bash
# format all .go files in a tree, then quit
nvim --headless -c "args **/*.go" -c "argdo lua vim.lsp.buf.format()" -c "wa" -c "qa"

# bulk substitute
nvim --headless -c "args **/*.md" -c "argdo %s/old/new/ge | update" -c "qa"

# run a lua script
nvim --headless -u NONE -c "luafile script.lua" -c "qa"
```

### See Also Sub-tip — Pair with tmux

```bash
# launch in same tmux pane vs. new window
nvim                   # current pane
tmux new-window nvim   # new window
tmux split-window -h nvim  # right split

# inside neovim :terminal blends well; map it to a key
vim.keymap.set("n", "<leader>tt", "<cmd>terminal<CR>", { desc = "Term" })
vim.keymap.set("t", "<C-\\><C-n>", [[<C-\><C-n>]])   -- Esc-equivalent in term
```

## See Also

- vim — base editor reference (modes, motions, operators, registers, ex commands, vimscript)
- tmux — terminal multiplexer pairs naturally with neovim for session/pane management
- lua — language reference for writing config (LuaJIT 2.1 dialect, what neovim uses)
- bash — shell knowledge for `:!` external commands and `:terminal` work
- zsh — shell pairing inside `:terminal` and for completion/history wiring
- polyglot — multi-language reference for matching the right LSP/parser per filetype

## References

- neovim.io/doc — official user manual (`:help` mirror)
- neovim.io/doc/user/lua-guide.html — Lua-specific guide
- neovim.io/doc/user/api.html — full vim.api surface
- neovim.io/doc/user/lsp.html — LSP client reference
- neovim.io/doc/user/treesitter.html — Treesitter integration
- neovim.io/doc/user/news.html — version-by-version change log (read after upgrade)
- github.com/nanotee/nvim-lua-guide — community Lua + nvim crash course
- github.com/nvim-lua/kickstart.nvim — single-file teaching config
- github.com/folke/lazy.nvim — plugin-manager docs
- github.com/neovim/nvim-lspconfig — LSP server config presets
- github.com/williamboman/mason.nvim — LSP/DAP/formatter installer
- github.com/nvim-treesitter/nvim-treesitter — parser collection + module manager
- github.com/hrsh7th/nvim-cmp — completion engine
- github.com/L3MON4D3/LuaSnip — snippet engine
- github.com/nvim-telescope/telescope.nvim — fuzzy finder
- github.com/lewis6991/gitsigns.nvim — git decorations
- github.com/folke/which-key.nvim — keymap help popup
- github.com/mfussenegger/nvim-dap — DAP client
- LazyVim.org — turn-key distribution and reference
- astronvim.com — community distro docs
- nvchad.com — opinionated minimal distro
- :help news — built-in change log per release
- :help vim_diff — full Neovim vs Vim difference list
