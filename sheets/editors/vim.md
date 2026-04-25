# Vim (Modal Text Editor)

> Vi IMproved — modal, ubiquitous, scriptable; the editor that ships on every Unix and runs in every container.

## Setup

Vim is preinstalled on virtually every Unix-like system. The binary may be `vim`, `vi`, `vim.tiny`, `vim.basic`, `vim.huge`, or `nvim`. What you get depends on the build flavor.

```bash
which vim                     # path resolution
vim --version                 # version + compile-time features (the +/- list)
vim --version | head -1       # single-line: VIM - Vi IMproved 9.1
vim --version | grep -E 'clipboard|python3|lua|ruby|perl|terminal'

apt install vim               # Debian/Ubuntu — installs vim-runtime
apt install vim-gtk3          # +clipboard via X11/GTK on Linux
brew install vim              # macOS Homebrew (huge build, +clipboard)
dnf install vim-enhanced      # Fedora
pacman -S vim gvim            # Arch — gvim package gives +clipboard

vi                            # often a symlink to vim, sometimes BSD nposix vi
ex                            # ex mode (line editor, no screen)
view                          # vim -R (read-only)
gvim                          # GUI wrapper
rvim                          # restricted (no shell, no :!), use carefully
vimdiff a b                   # 4-way diff mode (also vim -d a b)
```

The +clipboard, +python3, +lua, +terminal, +xterm-clipboard build flags determine what works. A sign with `-` means missing. The "tiny" Debian build has `-clipboard -eval` and breaks half of all plugins.

```bash
vim --version | grep -o '[+-]clipboard'   # +clipboard or -clipboard
vim --version | grep -o '[+-]python3'     # is :py3 available
vim --version | grep -o '[+-]terminal'    # :terminal command
vim --version | grep -o '[+-]xterm_clipboard'  # X11 selection over SSH
```

vim vs vi vs nvim differences:

```bash
vi          # POSIX vi, often vim in compatibility mode (set compatible)
vim         # Bram Moolenaar's Vi IMproved (current: 9.1, released 2024)
nvim        # Neovim — fork with Lua config and native LSP, see neovim sheet
vim -u NONE # skip all rc files (clean test env)
vim -N      # nocompatible mode, even if invoked as vi
```

Vim 9.0 introduced Vim9 script (10x faster than legacy vimscript) and is the recommended baseline as of 2024. Vim 8.x added async jobs (`job_start`), packages (`pack/*/start`), and `:terminal`. Anything older than 8 lacks plugins-as-packages.

```bash
:version              # full version + features inside Vim
:echo has('clipboard') :echo has('python3')   # 1 if compiled in
:echo v:version       # 901 = 9.1
```

`:help vim-modes` `:help startup` `:help feature-list`

## Modes

Vim is modal. The mode determines what every keypress means.

```bash
Normal           # default — keys are commands (h, d, y, :)
Insert           # keys insert text (entered via i, a, o, I, A, O, s, S, c)
Visual           # selecting (v), Visual-Line (V), Visual-Block (Ctrl-v)
Select           # like visual but typing replaces (rarely used outside GUI)
Command-line     # ex commands after :, /, ?, !
Replace          # R — overwrites text, r — single char then back
Operator-pending # after d, y, c, etc. waiting for motion or text-object
Terminal         # inside :terminal; Ctrl-w N to exit to normal
```

Mode transitions (the "modes diagram"):

```bash
                  Normal  <----- ESC / Ctrl-[ / Ctrl-c
                 /  |  | \
                v   v  v  v
            Insert Visual : Cmdline
                  Select       /
                       Replace
```

Returning to Normal mode:

```bash
ESC          # canonical exit-to-normal
Ctrl-[       # equivalent (ESC alternative, won't trigger meta on terminals)
Ctrl-c       # exits but skips abbreviations and aborts pending op
Ctrl-\ Ctrl-n  # universal "go to normal" — works from terminal mode too
```

Show current mode in statusline:

```bash
set showmode                  # :h 'showmode' — prints -- INSERT -- in cmd area
set noshowmode                # let plugins like lualine/airline render it
```

Gotcha: pressing `Ctrl-c` instead of ESC sometimes leaves an autocmd half-finished (`InsertLeave` may not fire). Use ESC for normal workflow; `Ctrl-c` only as a panic abort.

`:help vim-modes` `:help mode-switching` `:help i_CTRL-O`

## Motion Alphabet

Motions move the cursor and define the range an operator acts on. Master these and the editor disappears.

### Character and word

```bash
h j k l       # left, down, up, right
w             # next word start (lowercase: word = letters, digits, underscore)
W             # next WORD start (uppercase: WORD = whitespace-delimited)
b             # previous word start
B             # previous WORD start
e             # next word end
E             # next WORD end
ge            # previous word end
gE            # previous WORD end
```

### Line bounds

```bash
0             # column 0 (very first column)
^             # first non-blank char of line
$             # end of line
g_            # last non-blank char of line
+             # first non-blank of next line (also Enter)
-             # first non-blank of previous line
_             # first non-blank of [count] line below ([count]_ = count)
g0            # first column of screen line (when wrap on)
g^            # first non-blank of screen line
g$            # end of screen line
gm            # middle of screen line
```

### File bounds and screen positioning

```bash
gg            # first line of file
G             # last line of file (also :$)
{count}G      # go to line {count} (also :{count})
{count}gg     # same
H             # top of visible screen (High)
M             # middle of visible screen
L             # bottom of visible screen (Low)
zt            # scroll: cursor to top
zz            # scroll: cursor to middle
zb            # scroll: cursor to bottom
Ctrl-d        # half-page down
Ctrl-u        # half-page up
Ctrl-f        # full page forward
Ctrl-b        # full page back
Ctrl-e        # scroll line down (cursor stays)
Ctrl-y        # scroll line up
```

### Bracket matching and structural

```bash
%             # matching ( ) [ ] { } /* */, also #if/#endif via matchit.vim
[[            # previous { in column 0 (function start in C)
]]            # next { in column 0
[]            # previous } in column 0
][            # next } in column 0
[{            # previous unmatched {
]}            # next unmatched }
[(            # previous unmatched (
])            # next unmatched )
(             # previous sentence
)             # next sentence
{             # previous paragraph (blank line)
}             # next paragraph
```

### Character search on current line

```bash
f{c}          # forward to char c (cursor lands ON c)
F{c}          # backward to c
t{c}          # forward 'til c (cursor lands BEFORE c) — use with d/c
T{c}          # backward 'til c
;             # repeat last f/F/t/T forward
,             # repeat last reversed
```

### Search-based motion

```bash
/pat<Enter>   # search forward for pat
?pat<Enter>   # search backward
n             # next match (same direction as last search)
N             # previous match
*             # search word under cursor forward (\<word\>)
#             # search word under cursor backward
g*            # like * but no \< \> (substring match)
g#            # like #, substring
```

### Jumping history

```bash
Ctrl-o        # jump to older position (think "out")
Ctrl-i        # jump to newer position (Tab — same as Tab key)
:jumps        # list jump history
g;            # jump to older change
g,            # jump to newer change
:changes      # list change history
``            # to position before last jump
''            # to line of last jump
```

`:help motion.txt` `:help word-motions` `:help jump-motions` `:help mark-motions`

## Operators

An operator + a motion = a command. The operator says WHAT, the motion says WHERE.

```bash
d{motion}     # delete
y{motion}     # yank (copy)
c{motion}     # change (delete then enter insert)
v{motion}     # visual-select range
={motion}     # auto-indent
>{motion}     # shift right
<{motion}     # shift left
gu{motion}    # lowercase
gU{motion}    # UPPERCASE
g~{motion}    # toggle case
gq{motion}    # format (textwidth-wrap)
gw{motion}    # format without moving cursor
g?{motion}    # rot13 (yes really)
zf{motion}    # create fold
!{motion}     # filter through external command (then cmd)
```

Doubled operator = whole line:

```bash
dd            # delete current line
yy            # yank line (also Y in some configs, default = y$)
cc            # change line
==            # auto-indent line
>>            # shift line right by shiftwidth
<<            # shift line left
guu / gUU     # lowercase / UPPERCASE line
~             # toggle case of char under cursor (no motion needed)
```

The combo: counts go anywhere. `3dw`, `d3w`, `3d3w` all work (the latter = 9 words).

```bash
d3w           # delete 3 words
dw3           # WRONG — 3 is a separate motion (3 columns? no, error)
d$            # delete to end of line (also D)
d0            # delete to start of line
dgg           # delete to beginning of file
dG            # delete to end of file
y$            # yank to end of line (also Y in modern vim)
c}            # change to end of paragraph
=G            # auto-indent from cursor to end of file
gg=G          # auto-indent ENTIRE file (canonical reformat)
>i{           # indent inside braces (one shiftwidth)
```

Special: `D = d$`, `C = c$`, `S = cc`, `Y = yy` (or `y$` if `:set Y` mapped — see :h Y).

`:help operator` `:help d` `:help c` `:help motion-count`

## Text Objects

Text objects describe a region rather than a motion to its boundary. Format: `[a|i]{object}` where `a` = "around" (includes delimiters/whitespace) and `i` = "inner" (just the contents).

```bash
aw  iw        # a word / inner word
aW  iW        # a WORD / inner WORD
as  is        # a sentence / inner sentence
ap  ip        # a paragraph / inner paragraph
a"  i"        # a "string" / inner string (between quotes)
a'  i'        # a 'string' / inner
a`  i`        # a `string` / inner
a(  i(        # a (parens) / inner (also ab / ib for "block")
a)  i)        # same as a( / i(
a[  i[        # a [brackets] / inner
a]  i]        # same
a{  i{        # a {braces} / inner (also aB / iB)
a}  i}        # same
a<  i<        # a <angle> / inner
a>  i>        # same
at  it        # a tag / inner tag (HTML/XML <p>...</p>)
```

The canonical daily-driver examples:

```bash
daw           # delete a word (with surrounding space)
diw           # delete inner word (just the word, leave space)
ciw           # change inner word — most common rename action
yi"           # yank inside quotes
ci"           # change inside quotes
va(           # visual-select including parentheses
vi{           # visual-select inside braces
dap           # delete a paragraph
yip           # yank a paragraph
dat           # delete an HTML tag (full <p>...</p>)
cit           # change inside an HTML tag (just the contents)
```

Plugin-extended (with kana/vim-textobj-* or vim-textobj-user):

```bash
ai  ii        # a/inner indentation block (vim-textobj-indent)
af  if        # a/inner function (vim-textobj-function or LSP-aware)
ac  ic        # a/inner comment
al  il        # a/inner line (text on the line)
```

`:help text-objects` `:help v_aw` `:help v_i(` `:help v_at`

## Search

Forward and backward incremental search, with regex flavors.

```bash
/pat<Enter>   # search forward
?pat<Enter>   # search backward
n             # next match (same direction)
N             # previous match (opposite direction)
/             # repeat last forward
?             # repeat last backward
*             # search word under cursor forward — \<word\>
#             # search word under cursor backward
g*            # like * but no word boundaries
g#            # like # but no word boundaries
```

Regex magic levels:

```bash
/foo          # default ('magic') — most metas need no escape, () \( required
/\vfoo(bar|baz)+    # very-magic — all chars regex unless escaped (PCRE-ish)
/\Vfoo.bar    # very-nomagic — only \ has meaning, . is literal dot
/\Mfoo        # nomagic — even less magic
```

Useful flags inline:

```bash
/foo\c        # force case-INsensitive (overrides ignorecase setting)
/foo\C        # force case-sensitive
/^foo$        # full line match
/\<foo\>      # word boundaries
/foo\zsbar    # match foo+bar, but only highlight bar (\zs = match start)
/foo\zebar    # match foo+bar, only highlight foo (\ze = match end)
/foo\&bar     # match foo AND bar at same position (rare but useful)
```

The canonical search settings combo:

```bash
set hlsearch     # highlight all matches
set incsearch    # show match as you type
set ignorecase   # /Foo finds foo
set smartcase    # but /Foo when has uppercase matches only foo (case-sensitive)
nnoremap <leader>h :nohlsearch<CR>   # toggle off the lingering highlight
```

Search history navigation:

```bash
/             # then Up / Down arrows = browse history
q/            # open search history window (use Ctrl-c to close)
q?            # backward search history
```

`:help search-commands` `:help pattern-overview` `:help /\v` `:help /\zs`

## Substitute

The find-and-replace engine. Range, then `s/pat/rep/flags`.

```bash
:s/old/new/             # replace first occurrence on current line
:s/old/new/g            # all on current line
:s/old/new/gc           # all + confirm each
:%s/old/new/g           # all in file (% = whole buffer = 1,$)
:%s/old/new/gc          # confirm each — y/n/a/q/l/^E/^Y
:'<,'>s/old/new/g       # within last visual selection
:.,$s/old/new/g         # current line to end
:.,+5s/old/new/g        # next 5 lines
:5,10s/old/new/g        # lines 5 through 10
:/start/,/end/s/old/new/g    # between two patterns
```

Flags:

```bash
g     # global (all in line)
c     # confirm each
i     # ignore case (this substitute only)
I     # case sensitive (this substitute)
&     # use flags from previous :s
e     # don't error on no match (useful in scripts)
n     # report match count, don't substitute
```

Backreferences and special replacements:

```bash
:%s/\(\w\+\)\s\+\1/\1/g  # collapse double-words (default magic)
:%s/\v(\w+)\s+\1/\1/g    # same with very-magic
:%s/foo/\U&/g            # uppercase the match (\U through \E)
:%s/foo/\L&/g            # lowercase
:%s/\(.\)/\1\1/g         # double every char
:%s/\v(\d+)/\=submatch(1)*2/g   # math: double every number (sub-replace-expression)
:%s/^/    /              # add 4 spaces to start of every line
:%s/\s\+$//              # trim trailing whitespace (canonical)
:%s/\r//g                # remove DOS carriage returns
:%s//new/g               # empty pattern reuses last search
&                        # repeat last :s (no flags) — use g& for whole-file
:&&                      # repeat last :s with same flags
:~                       # repeat with last regex from any cmd
```

The :g and :v family — substitute's siblings:

```bash
:g/pattern/d             # delete all matching lines
:g/pattern/p             # print matching (mostly for scripts)
:v/pattern/d             # delete NON-matching lines (v = vice-versa)
:g/^$/d                  # delete blank lines
:g/TODO/normal A FIXME   # append " FIXME" to every line containing TODO
:g/^/m0                  # reverse all lines (canonical g+normal trick)
:g/^/y A                 # yank every line into register a (capital A appends)
```

`:help :s` `:help :s_flags` `:help sub-replace-special` `:help :g`

## Marks

Marks are bookmarks. Lowercase = buffer-local; uppercase = global (cross-file).

```bash
ma            # set mark a at cursor
mA            # set GLOBAL mark A (file + line + column)
'a            # jump to LINE of mark a (first non-blank)
`a            # jump to exact line+column of mark a
'A            # jump to file containing mark A and to its line
g'a           # like 'a but doesn't change jump list
:marks        # list all marks
:marks abc    # list marks a, b, c only
:delmarks a   # delete mark a
:delmarks!    # delete all lowercase marks
```

Special automatic marks:

```bash
'.            # last change position
'^            # last insert mode position
''            # position before last jump (toggle: '' bounces back)
'"            # cursor position when last exited buffer
'[            # start of last yank/change/paste
']            # end of last yank/change/paste
'<            # start of last visual selection
'>            # end of last visual selection
'(            # start of current sentence
')            # end of current sentence
'{            # start of current paragraph
'}            # end of current paragraph
```

Useful idioms:

```bash
ma            # bookmark current spot
... wander ...
'a            # come back
y'a           # yank from cursor to mark a
d`a           # delete from cursor to exact position of mark a
:'a,'b!sort   # sort lines from mark a to mark b through external sort
```

`:help mark-motions` `:help :marks` `:help '.` `:help '"`

## Registers

Registers are named clipboards. Prefix any yank/delete/paste with `"x` to use register x.

```bash
""            # default unnamed register (last yank/delete)
"0            # last YANK (preserved through deletes)
"1 .. "9      # delete history ring (most recent in 1)
"-            # small-delete (less than one line)
"a-z          # named registers (lowercase = overwrite)
"A-Z          # APPEND to corresponding lowercase register
"_            # blackhole — discard, don't fill any register
"+            # SYSTEM CLIPBOARD (requires +clipboard build)
"*            # X11 PRIMARY selection (middle-click clipboard)
"/            # last search pattern
":            # last ex command
"."           # last inserted text
"%            # current filename
"#            # alternate filename (the # buffer)
"=            # expression register (prompt for expression, paste result)
```

Common operations:

```bash
"ayy          # yank line into a
"Ayy          # APPEND line to a (capital)
"ap           # paste from a after cursor
"aP           # paste from a before cursor
:reg          # show all registers and their contents
:reg a + 0    # show only registers a, +, 0
:let @a = ''  # clear register a
:let @a = 'hello'   # set register a from script
:put a        # ex command: paste a as new line(s)
```

System clipboard idioms (the daily driver):

```bash
"+y           # yank to system clipboard (Ctrl-C-equivalent)
"+yy          # yank line to clipboard
"+yiw         # yank inner word to clipboard
"+p           # paste from system clipboard
gg"+yG        # yank entire file to clipboard
:%y+          # ex equivalent of yank-all-to-clipboard
```

To always use system clipboard:

```bash
set clipboard=unnamedplus       # Linux/Wayland — yanks go to "+
set clipboard=unnamed           # macOS — yanks go to "* (which IS macOS clipboard)
set clipboard=unnamedplus,unnamed   # both
```

Expression register — paste computed values:

```bash
"=            # then type expression, e.g. strftime("%Y-%m-%d") + Enter
              # in insert mode: Ctrl-r = then expr Enter
i Ctrl-r =5*8 <Enter>   # inserts "40" at cursor
```

`:help registers` `:help quote_+` `:help quote_=` `:help :reg`

## Macros

Record a sequence of normal commands, replay arbitrarily.

```bash
qa            # start recording into register a (statusline shows "recording @a")
... commands ...
q             # stop recording
@a            # replay macro a
@@            # replay LAST played macro
5@a           # play macro a five times
100@a         # play 100 times — stops on error (e.g., end of file)
:%normal! @a  # run macro on every line
:'<,'>normal! @a    # run macro on visual selection
```

A macro is just text in a register. You can edit it:

```bash
:let @a = 'IHello <Esc>'        # build a macro from script
:put a                          # paste macro to inspect/edit
"ap                             # paste, edit, then "add to record
```

Recursive macros — useful when you want "until end":

```bash
qa qa         # clear register a (start recording, stop immediately)
qa            # start recording fresh
... commands ...
@a            # call self at end (recursive — only works if a was empty before)
q             # stop
@a            # runs to error (often EOF) — handy "do until done"
```

Capital A appends:

```bash
qA            # APPEND to existing macro a
... more commands ...
q
```

Gotchas:
- Macros don't replay register usage perfectly (`"+y` may behave oddly under macros).
- Use `qq` then `@q` for ad-hoc one-off macros — q is the throwaway register by convention.
- If a macro stops early, the failed motion (e.g., `f;` finding nothing) aborts the macro. Use `:set nowrapscan` carefully.

`:help recording` `:help q` `:help @` `:help complex-repeat`

## Ex Commands

Ex commands are typed after `:`. They take an optional range, a command, optional arguments, and bang `!` for force.

### Range syntax

```bash
:5            # line 5
:5,10         # lines 5 to 10
:5,10d        # delete lines 5-10
:.            # current line
:$            # last line
:%            # all lines (synonym for 1,$)
:.,$          # current to end
:.,+5         # current and next 5 lines
:.-3,.+3      # 3 above to 3 below cursor
:'a,'b        # mark a to mark b
:'<,'>        # last visual selection (auto-inserted by :)
:/pat/        # next line matching pat
:?pat?        # previous line matching pat
:/start/,/end/d   # delete from line matching start to line matching end
```

### File operations

```bash
:w            # write current file
:w!           # force write (readonly, etc.)
:w foo.txt    # write to foo.txt (does NOT change current filename)
:saveas foo   # write AND switch buffer to foo.txt
:w >> log     # APPEND to log
:wa           # write all changed buffers
:wq           # write and quit
:wq!          # force write and quit
:x            # write IF changed and quit (preferred — preserves mtime)
ZZ            # = :x (normal mode shortcut)
ZQ            # = :q! (normal mode shortcut)
:q            # quit (fails if buffer modified)
:q!           # quit without saving
:qa           # quit all
:qa!          # quit all forcefully
:e foo.txt    # edit foo.txt (replaces current buffer if not modified)
:e!           # reload current file (DISCARDING unsaved changes)
:e            # reload current file (only if not modified)
:e #          # edit alternate file (the # buffer)
:e %:h/other  # edit file in same directory as current
```

### Reading and shelling

```bash
:r foo.txt    # read foo.txt content AFTER cursor line
:r !date      # read output of `date` after cursor
:r !curl https://...   # insert HTTP response
:!ls          # run shell command (output displayed)
:!python %    # run current file with python
:.!cmd        # filter current line through cmd
:%!sort       # filter whole buffer through sort
:%!jq .       # pretty-print JSON via external jq
:'<,'>!fmt    # filter visual selection through fmt
```

### Built-in :sort and friends

```bash
:sort                # sort all lines (case-sensitive)
:sort i              # ignore case
:sort u              # unique (dedupe + sort)
:sort n              # numeric sort
:sort!               # reverse sort
:sort /regex/        # sort by what comes AFTER regex match
:sort r /regex/      # sort by what regex matches
:%!uniq -c           # frequency count via external
:'<,'>!awk '{print NR, $0}'   # external awk
```

### Tabs, splits, buffers (covered in their own sections below)

```bash
:tabnew foo
:split foo
:vsplit foo
:hide buffer 3       # switch to buffer 3, hiding modified
:diffsplit foo       # open foo in split + diff
```

### Settings, mappings, sourcing

```bash
:set option          # query current value (also :set option?)
:set option=value
:set nooption        # unset (boolean)
:set option!         # toggle
:setlocal            # only this buffer
:setglobal
:let g:var = 'x'     # set Vim variable
:let &option = ...   # set option via :let (allows expressions)
:map  / :nmap / etc. # see Mappings section
:unmap <key>
:source ~/.vimrc     # reload config
:source %            # source current buffer
:runtime plugin/foo.vim   # source via runtimepath
:scriptnames         # list every sourced script (debug load order)
:messages            # see hidden messages / errors
:redraw!             # force screen redraw (useful after :!cmd)
:silent !cmd         # run shell silently
:verbose set option? # show where the option was last set (which file/line)
:verbose map <C-p>   # show where mapping was set
:version
:help topic
```

`:help :w` `:help cmdline-ranges` `:help :!` `:help :sort` `:help :verbose`

## Windows and Splits

Windows are viewports onto buffers. All window commands start with `Ctrl-w` (or `:wincmd`).

```bash
Ctrl-w s        # horizontal split (current buffer)
Ctrl-w v        # vertical split
:sp foo         # horizontal split + edit foo
:vsp foo        # vertical split + edit foo
:new            # new horizontal split with empty buffer
:vnew           # new vertical
Ctrl-w c        # close current window (= :close)
Ctrl-w o        # only — close all OTHER windows (= :only)
Ctrl-w q        # quit window (= :q, may close last buffer)
```

Navigation:

```bash
Ctrl-w h        # move to LEFT window
Ctrl-w j        # move DOWN
Ctrl-w k        # move UP
Ctrl-w l        # move RIGHT
Ctrl-w w        # cycle to next window
Ctrl-w W        # cycle to previous
Ctrl-w t        # top-left window
Ctrl-w b        # bottom-right window
Ctrl-w p        # previous (most-recent) window
```

Move/exchange:

```bash
Ctrl-w H        # move current window to FAR LEFT (full height)
Ctrl-w J        # move to BOTTOM (full width)
Ctrl-w K        # move to TOP
Ctrl-w L        # move to FAR RIGHT
Ctrl-w r        # rotate windows
Ctrl-w x        # exchange with neighbor
Ctrl-w T        # move current window to its OWN TAB
```

Resize:

```bash
Ctrl-w =        # equalize all windows
Ctrl-w _        # max height of current window
Ctrl-w |        # max width
Ctrl-w +        # +1 line height
Ctrl-w -        # -1 line height
Ctrl-w >        # +1 column width
Ctrl-w <        # -1 column width
5 Ctrl-w +      # 5 lines taller
:resize 30      # explicit height
:vertical resize 80   # explicit width
```

Recommended mappings for fewer keystrokes:

```bash
nnoremap <C-h> <C-w>h
nnoremap <C-j> <C-w>j
nnoremap <C-k> <C-w>k
nnoremap <C-l> <C-w>l
```

`:help windows.txt` `:help CTRL-W` `:help :split` `:help winheight`

## Tabs

Tabs are workspaces (collections of windows), NOT files. Each tab can have any number of windows; each window shows a buffer. Tabs do not own files — buffers do.

```bash
:tabnew              # new empty tab
:tabnew foo.txt      # new tab editing foo.txt
:tabedit foo         # alias
:tabe                # short
gt                   # next tab
gT                   # previous tab
{n}gt                # go to tab n (1-indexed)
:tabfirst / :tabr    # first tab
:tablast             # last tab
:tabclose            # close current tab
:tabonly             # close all OTHER tabs
:tabmove 0           # move tab to position 0 (first)
:tabmove $           # move to last
:tabmove +1          # move right
:tabs                # list all tabs and their windows
:tabdo cmd           # run cmd in every tab's current window
```

Mental model: think of tabs as "layout snapshots". Splits show MORE of the SAME workspace; tabs show DIFFERENT workspaces.

`:help tab-page` `:help :tabnew` `:help gt`

## Buffers

A buffer is the in-memory representation of a file. Listed buffers persist across windows/tabs.

```bash
:ls                  # list buffers
:buffers             # synonym
:files               # synonym
:b 3                 # switch to buffer #3
:b foo               # switch to buffer matching "foo" (partial)
:bn                  # next buffer
:bp                  # previous buffer
:bf / :blast         # first / last
:bd                  # delete buffer (unloads)
:bd!                 # force (discard changes)
:bw                  # WIPEOUT (also removes marks/options for buffer)
:bufdo cmd           # run cmd in every buffer
:bufdo! cmd          # ignore errors
```

Buffer state characters in `:ls` output:

```bash
%       # currently displayed in this window
#       # alternate buffer (last edited; <C-^> jumps here)
a       # active (loaded + visible somewhere)
h       # hidden (loaded but not visible)
u       # unlisted (won't show without :ls!)
=       # readonly
+       # MODIFIED (unsaved changes)
-       # 'modifiable' off
x       # read errors
```

Combinations:

```bash
:ls!                 # include unlisted buffers (help, scratch, terminal)
:ls +                # only modified buffers
:ls a                # only active buffers
%a +    file.txt    "current, active, modified"
```

Daily idioms:

```bash
Ctrl-^               # toggle to alternate buffer (the # one)
:bd                  # close current buffer (don't quit window)
:bufdo update        # save all modified buffers
:%bd|e#              # close all buffers EXCEPT current (popular trick)
```

Why hidden buffers matter:

```bash
set hidden           # allow switching from a modified buffer without saving
                     # without 'hidden': :bn fails saying "No write since last change"
```

`:help buffers` `:help :ls` `:help 'hidden'`

## Folds

Hide regions of text. The fold opens and closes; the content stays.

```bash
zf{motion}    # create fold over motion (manual foldmethod)
zf3j          # fold next 3 lines
zfap          # fold a paragraph
zfa{          # fold around braces
zo            # open one fold under cursor
zO            # open all folds at cursor (recursive)
zc            # close one fold
zC            # close all at cursor
za            # toggle fold (open if closed, close if open)
zA            # toggle recursively
zR            # reduce — open ALL folds in buffer
zM            # more — close ALL folds
zr            # open one fold level
zm            # close one fold level
zd            # delete fold under cursor (manual method)
zE            # eliminate ALL folds in buffer
zj            # move to next fold
zk            # move to previous fold
zn            # 'foldenable' off (show everything temporarily)
zN            # 'foldenable' on
zi            # toggle 'foldenable'
```

Foldmethods:

```bash
set foldmethod=manual    # zf creates folds (default)
set foldmethod=indent    # fold by indent level (great for Python)
set foldmethod=marker    # fold between {{{ and }}} markers
set foldmethod=syntax    # use syntax file definitions
set foldmethod=expr      # use 'foldexpr' (custom)
set foldmethod=diff      # fold unchanged regions in diff mode

set foldlevel=99         # start fully unfolded (handy with indent/syntax)
set foldlevelstart=99    # default for new buffers
set foldnestmax=3        # don't fold deeper than 3 levels
set foldcolumn=2         # show fold indicators in left column
set foldminlines=3       # don't fold less than 3 lines
```

Marker syntax:

```bash
" some code {{{
function foo() { ... }
" }}}
```

`:help folding` `:help fold-commands` `:help foldmethod`

## Insert Mode Tricks

Insert mode is mostly typing, but there's a small but vital command set.

```bash
Ctrl-h           # backspace one char
Ctrl-w           # delete previous WORD
Ctrl-u           # delete to start of line
Ctrl-t           # indent line one shiftwidth
Ctrl-d           # un-indent one shiftwidth
Ctrl-r {reg}     # insert contents of register
Ctrl-r =         # insert result of expression
Ctrl-r Ctrl-w    # insert word under cursor (from where insert started)
Ctrl-r Ctrl-a    # insert WORD under cursor
Ctrl-r Ctrl-l    # insert current line
Ctrl-o {cmd}     # do ONE normal-mode command, then back to insert
Ctrl-c           # exit to normal (skips InsertLeave autocmd)
Ctrl-[           # exit to normal (= ESC)
Ctrl-v {char}    # literal insert (for ESC, Tab, etc.)
Ctrl-q {char}    # same as Ctrl-v on Windows
Ctrl-k {ab}      # digraph insert (Ctrl-k a' = á; :digraphs lists)
Ctrl-a           # repeat last inserted text
Ctrl-@           # repeat last inserted text + leave insert (rare)
Ctrl-y           # copy char from line ABOVE (great for ASCII art)
Ctrl-e           # copy char from line BELOW
```

Completion (autocomplete from various sources):

```bash
Ctrl-n           # next completion (from current buffer words)
Ctrl-p           # previous completion
Ctrl-x Ctrl-l    # whole-line completion
Ctrl-x Ctrl-n    # buffer keyword completion
Ctrl-x Ctrl-k    # dictionary completion (set 'dictionary')
Ctrl-x Ctrl-t    # thesaurus
Ctrl-x Ctrl-i    # included files (#include) keyword
Ctrl-x Ctrl-]    # tag completion (ctags)
Ctrl-x Ctrl-f    # filename completion (paths)
Ctrl-x Ctrl-d    # macro definition
Ctrl-x Ctrl-v    # vim command-line completion
Ctrl-x Ctrl-u    # user-defined completion ('completefunc')
Ctrl-x Ctrl-o    # OMNI completion ('omnifunc') — language-aware
Ctrl-x Ctrl-s    # spelling suggestions
Ctrl-x s         # spelling (alternate)
```

While completion menu is up:

```bash
Ctrl-n / Ctrl-p   # next / previous candidate
Ctrl-y            # accept (Yes)
Ctrl-e            # cancel (Escape menu)
Ctrl-l            # extend the longest common prefix
```

Idioms:

```bash
i Ctrl-r"        # paste unnamed register without leaving insert
i Ctrl-r +       # paste system clipboard at cursor
i Ctrl-r =strftime("%c") <Enter>   # insert current timestamp
Ctrl-o dd        # delete current line, stay in insert
Ctrl-o zz        # recenter screen, stay in insert
```

`:help insert.txt` `:help i_CTRL-R` `:help ins-completion` `:help i_CTRL-X_CTRL-O`

## Visual Mode Operations

Selection, then operate.

```bash
v           # enter character visual
V           # line-wise visual
Ctrl-v      # block visual (column edits)
gv          # re-select last visual region
o           # swap cursor to other END of selection (refine the boundary)
O           # in block visual: swap to other CORNER (diagonal)
```

Operations on selection:

```bash
d           # delete
y           # yank
c           # change
~           # toggle case
u           # lowercase
U           # uppercase
=           # auto-indent
>           # shift right
<           # shift left
gq          # textwidth wrap
gw          # textwidth wrap (cursor stays)
J           # join lines (with single space)
:           # opens cmdline with :'<,'> already inserted
!           # filter through external command
```

Block-mode column edits — the canonical visual-block trick:

```bash
Ctrl-v          # enter block visual
{move}          # select rectangle (e.g., j j j to span 4 lines, l l l for width)
I               # insert at LEFT edge of every selected line
{type text}     # appears only on top line during typing
ESC             # propagates the inserted text to ALL selected lines

Ctrl-v
{select column}
A               # APPEND at right edge (note: A in block-visual, not at end of buffer)
{type}
ESC

Ctrl-v
{select region}
c               # change — type replacement, ESC propagates
r{c}            # replace EVERY selected char with c (no need to ESC)
```

Examples:

```bash
Ctrl-v G $ A ;<Esc>     # add ; to end of every line from cursor to EOF
Ctrl-v 5j I // <Esc>    # comment 5 lines (insert "// " at left of each)
Ctrl-v 5j x             # delete first column from 5 lines
gv                      # reselect last selection (after operating)
```

`:help visual.txt` `:help v_o` `:help v_O` `:help blockwise-visual` `:help v_b_I`

## Indentation

Vim has half a dozen indent systems. They interact subtly.

```bash
>>          # shift line right by shiftwidth
<<          # shift left
>i{         # indent inside braces
=           # auto-indent (operator) — uses 'indentexpr' or 'cindent'
==          # auto-indent current line
=ip         # auto-indent paragraph
gg=G        # auto-indent ENTIRE file
G=gg        # equivalent (motion direction differs)
```

Settings (the four that matter):

```bash
set tabstop=4        # how many columns a TAB character displays as
set softtabstop=4    # how many spaces TAB key inserts (or backspaces)
set shiftwidth=4     # how many spaces > and < and auto-indent uses
set expandtab        # TAB key inserts spaces, NOT a tab character
set noexpandtab      # TAB key inserts a literal tab (Makefile-friendly)
```

Common combos:

```bash
" 4-space indent (Python, JS, Go-with-spaces):
set tabstop=4 softtabstop=4 shiftwidth=4 expandtab

" 2-space indent (HTML, Lua, YAML):
set ts=2 sts=2 sw=2 et

" Tab-indented (Go canonical, Makefile required):
set ts=4 sw=4 noexpandtab

" Filetype-specific via autocmd:
autocmd FileType python  setlocal ts=4 sts=4 sw=4 et
autocmd FileType yaml    setlocal ts=2 sts=2 sw=2 et
autocmd FileType make    setlocal noexpandtab
```

Indent engines (mutually exclusive — last one wins):

```bash
set autoindent       # carry indent from previous line on Enter
set smartindent      # like autoindent + extra rules for { and #
set cindent          # C-aware indent (prefer over smartindent)
filetype indent on   # use language-specific indent files (RECOMMENDED)
                     # — installs files from runtime/indent/
```

Re-tabify file:

```bash
set tabstop=8        # tabs are EIGHT (the historical view)
set noexpandtab      # don't replace tabs with spaces
:retab               # convert based on current settings
:retab!              # convert tabs ⇄ spaces aggressively
:%s/\t/    /g        # blunt: replace every tab with 4 spaces
```

`:help indent.txt` `:help 'tabstop'` `:help 'shiftwidth'` `:help filetype-indent`

## Settings — Common Reference

The grab-bag of options that 90% of `.vimrc` files set. Each shown with its `:help` reference.

### Numbers and signs

```bash
set number              " absolute line numbers (also: set nu)
set relativenumber      " relative numbers (set rnu) — combine with set nu for hybrid
set numberwidth=4
set signcolumn=yes      " always show sign column (avoids text shift on diagnostic)
set signcolumn=yes:2    " 2 columns wide (room for git + LSP)
set cursorline          " highlight current line
set cursorcolumn        " highlight current column (visual but expensive)
set colorcolumn=80      " visual ruler at column 80
set colorcolumn=+1      " ruler at textwidth+1
```

### Search

```bash
set hlsearch            " highlight matches
set incsearch           " incremental — show matches as you type
set ignorecase          " /Foo finds foo
set smartcase           " UNLESS pattern has uppercase, then case-sensitive
set wrapscan            " search wraps at end of file (default)
```

### Indent / tabs

```bash
set tabstop=4 softtabstop=4 shiftwidth=4 expandtab
set autoindent
set smarttab            " <Tab> at line start uses shiftwidth
set shiftround          " round indent to multiple of shiftwidth
filetype plugin indent on
syntax on
```

### Display

```bash
set wrap                " wrap long lines visually (default)
set nowrap              " don't wrap (horizontal scroll)
set linebreak           " when wrap is on, break at word boundaries
set breakindent         " visually indent wrapped lines to match first line
set showbreak=↳         " marker for wrapped continuation
set scrolloff=8         " keep 8 lines visible above/below cursor
set sidescrolloff=8
set termguicolors       " enable 24-bit color (most modern terminals)
set background=dark     " or 'light' — affects colorscheme
set list listchars=tab:>·,trail:·,nbsp:¬,extends:>,precedes:<,eol:¶
set conceallevel=2      " hide concealed text (used by markdown plugins)
set fillchars=eob:\ ,vert:│,fold:·
```

### Splits

```bash
set splitbelow          " :sp opens below current
set splitright          " :vsp opens to right
```

### Undo

```bash
set undofile            " persistent undo (survives close)
set undodir=~/.vim/undo " where to store
set undolevels=10000
set undoreload=10000
```

### Clipboard / mouse

```bash
set clipboard=unnamedplus  " yank goes to system clipboard (Linux)
set clipboard=unnamed,unnamedplus  " both (macOS-friendly)
set mouse=a             " enable mouse in all modes
set mouse=nvi           " mouse only in normal/visual/insert
```

### Wildmenu (cmdline completion)

```bash
set wildmenu
set wildmode=longest:full,full
set wildignore=*.o,*.pyc,*.swp,*/node_modules/*,*/.git/*
set wildignorecase
```

### Performance / responsiveness

```bash
set lazyredraw          " skip redraws during macros (faster, less flicker)
set updatetime=300      " ms — affects CursorHold, swapfile write
set timeoutlen=500      " ms — wait for mapping completion
set ttimeoutlen=10      " ms — for key-code completion (low for snappy ESC)
set redrawtime=10000    " ms — cap on syntax/regex highlighting
set regexpengine=0      " 0=auto, 1=NFA-old, 2=NFA-new (try 1 if slow)
set synmaxcol=240       " don't syntax-highlight past col 240 on huge lines
```

### Bells and behavior

```bash
set noerrorbells
set novisualbell
set t_vb=               " no bell at all
set belloff=all
set hidden              " allow modified hidden buffers
set autoread            " reload file if changed externally
set autowrite           " save when switching buffers
set confirm             " prompt instead of failing on unsaved
set backspace=indent,eol,start  " backspace works "naturally"
set whichwrap+=<,>,h,l,[,]      " left/right wrap to prev/next line
```

### Folding

```bash
set foldmethod=indent   " or marker, syntax, manual
set foldlevel=99
set foldlevelstart=99
set foldcolumn=1
```

### Spell

```bash
set spell
set spelllang=en_us
set spellfile=~/.vim/spell/en.utf-8.add
" inside a buffer:
" zg = good word (add to file)
" zw = wrong word
" z= = suggestions
" ]s / [s = next/prev misspelling
```

### Encoding / line endings

```bash
set encoding=utf-8
set fileencoding=utf-8
set fileformat=unix
set fileformats=unix,dos
```

### Backup / swap

```bash
set nobackup            " or set backup + backupdir
set nowritebackup       " disable backup before overwrite
set noswapfile          " no .swp (less safety, fewer files)
set directory=~/.vim/swap//   " // = full path encoded into swap name
```

`:help options` `:help option-list` `:help quickref`

## .vimrc Structure

A canonical layout that scales from 50 to 5000 lines.

```bash
" ~/.vimrc — Vim 9 with plenty of legacy compat
" 1. Reset to known state
set nocompatible                  " in case launched as 'vi'

" 2. Leader keys (set BEFORE any mapping that uses <leader>)
let mapleader = ' '
let maplocalleader = ','

" 3. Plugin manager block (vim-plug example)
call plug#begin('~/.vim/plugged')
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-surround'
Plug 'tpope/vim-commentary'
Plug 'tpope/vim-repeat'
Plug 'junegunn/fzf', { 'do': { -> fzf#install() } }
Plug 'junegunn/fzf.vim'
Plug 'neoclide/coc.nvim', { 'branch': 'release' }
Plug 'mhinz/vim-signify'
call plug#end()

" 4. Core settings
syntax on
filetype plugin indent on
set encoding=utf-8 number relativenumber hidden hlsearch incsearch ignorecase smartcase
set tabstop=4 softtabstop=4 shiftwidth=4 expandtab autoindent smartindent
set scrolloff=8 sidescrolloff=8 splitbelow splitright termguicolors
set undofile undodir=~/.vim/undo updatetime=300 timeoutlen=500
set clipboard=unnamedplus mouse=a wildmenu wildmode=longest:full,full
set list listchars=tab:>·,trail:·,nbsp:¬

" 5. Colorscheme (after termguicolors)
silent! colorscheme habamax       " always-available scheme
silent! colorscheme tokyonight    " try preferred (silent! = ignore if missing)

" 6. Mappings
nnoremap <leader>w :w<CR>
nnoremap <leader>q :q<CR>
nnoremap <leader>h :nohlsearch<CR>
nnoremap <C-h> <C-w>h
nnoremap <C-j> <C-w>j
nnoremap <C-k> <C-w>k
nnoremap <C-l> <C-w>l
xnoremap > >gv                    " keep selection after indent
xnoremap < <gv

" 7. Autocommands (each in its own augroup for idempotent reload)
augroup vimrc_trim_whitespace
    autocmd!
    autocmd BufWritePre * %s/\s\+$//e
augroup END

augroup vimrc_restore_cursor
    autocmd!
    autocmd BufReadPost * if line("'\"") > 0 && line("'\"") <= line("$") | exe "normal! g`\"" | endif
augroup END

augroup vimrc_filetype
    autocmd!
    autocmd FileType python  setlocal ts=4 sts=4 sw=4 et
    autocmd FileType yaml    setlocal ts=2 sts=2 sw=2 et
    autocmd FileType go      setlocal noexpandtab
augroup END

" 8. Conditional features (portable across builds)
if has('clipboard') && has('unnamedplus')
    set clipboard=unnamedplus
endif

if has('persistent_undo')
    set undofile
    silent !mkdir -p ~/.vim/undo
endif

if has('terminal')
    tnoremap <Esc> <C-\><C-n>     " ESC exits terminal mode
endif
```

The `has('feature')` guard is critical for portability — your config should work on a tiny build, a huge build, and Neovim alike.

`:help vimrc` `:help has()` `:help feature-list` `:help :augroup`

## Mappings

The map-family lets you bind keys per-mode.

```bash
:map     <key> <action>   " ALL modes (normal+visual+select+op-pending)
:nmap    <key> <action>   " normal mode only
:vmap    <key> <action>   " visual + select
:xmap    <key> <action>   " visual ONLY (not select)
:smap    <key> <action>   " select only
:omap    <key> <action>   " operator-pending
:imap    <key> <action>   " insert
:cmap    <key> <action>   " command-line
:tmap    <key> <action>   " terminal mode
:lmap    <key> <action>   " language mode (input methods)

" Non-recursive (PREFERRED for safety):
:noremap, :nnoremap, :inoremap, :vnoremap, :xnoremap, :onoremap, :cnoremap, :tnoremap
```

Why `nnoremap` and not `nmap`?

```bash
" BAD — recursive:
nmap j gj
nmap gj jk     " gj triggers j (which is gj!) — INFINITE LOOP risk

" GOOD — non-recursive:
nnoremap j gj  " j now means gj, but gj keeps original meaning
```

Special key syntax:

```bash
<CR>          " Enter
<Esc>         " escape
<Tab> <S-Tab> " tab and shift-tab
<Space>
<BS>          " backspace
<Up> <Down> <Left> <Right>
<F1>...<F12>
<C-x>         " Ctrl-x
<S-x>         " Shift-x
<A-x>         " Alt-x (also <M-x>)
<D-x>         " Cmd on macOS GUI
<leader>      " value of mapleader
<localleader> " value of maplocalleader
<Plug>        " plugin-defined entry point
<Cmd>         " modeless cmd execution (Neovim, also Vim 9): no <CR> needed
<silent>      " don't echo
<expr>        " RHS is evaluated as expression
<buffer>      " buffer-local mapping
<unique>      " fail if mapping exists
<nowait>      " don't wait for longer mapping
```

Daily-driver mappings (canonical):

```bash
let mapleader = ' '

" Save / quit
nnoremap <leader>w :w<CR>
nnoremap <leader>q :q<CR>
nnoremap <leader>Q :qa!<CR>

" Clear search highlight
nnoremap <silent> <leader>h :nohlsearch<CR>

" Window nav
nnoremap <C-h> <C-w>h
nnoremap <C-j> <C-w>j
nnoremap <C-k> <C-w>k
nnoremap <C-l> <C-w>l

" Buffer nav
nnoremap <leader>bn :bnext<CR>
nnoremap <leader>bp :bprevious<CR>
nnoremap <leader>bd :bd<CR>

" Keep cursor centered on big jumps
nnoremap n nzzzv
nnoremap N Nzzzv
nnoremap <C-d> <C-d>zz
nnoremap <C-u> <C-u>zz

" Don't lose selection on indent
xnoremap > >gv
xnoremap < <gv

" Move selected lines up/down
xnoremap J :move '>+1<CR>gv=gv
xnoremap K :move '<-2<CR>gv=gv

" Quick edit/reload of vimrc
nnoremap <leader>ev :edit $MYVIMRC<CR>
nnoremap <leader>sv :source $MYVIMRC<CR>

" Better paste (don't yank what you replaced)
xnoremap <leader>p "_dP

" System clipboard shortcuts
nnoremap <leader>y "+y
xnoremap <leader>y "+y
nnoremap <leader>Y "+yg_
nnoremap <leader>p "+p

" Insert-mode quick exit
inoremap jk <Esc>
inoremap kj <Esc>
```

Inspect mappings:

```bash
:map               " all mappings
:nmap              " normal-mode mappings
:nmap <leader>w    " what does <leader>w do?
:verbose nmap <C-p>   " show file:line where mapping was set
:unmap <key>
:nunmap <key>
:mapclear          " clear ALL (DANGER)
```

`:help :map` `:help map-modes` `:help <Plug>` `:help map-arguments`

## Auto-commands

Run commands on Vim events.

```bash
:autocmd Event Pattern Cmd         " add an autocmd
:autocmd! Event Pattern            " remove all matching autocmds
:autocmd                           " list all
:autocmd BufRead                   " list all BufRead autocmds
```

Common events:

```bash
BufRead, BufReadPre, BufReadPost   " before/after reading a file
BufWrite, BufWritePre, BufWritePost
BufNew, BufNewFile                 " new (non-existing) file
BufEnter, BufLeave                 " window enters/leaves a buffer
BufWinEnter, BufWinLeave
BufDelete, BufWipeout, BufUnload
FileType                           " when 'filetype' is set
VimEnter                           " after all init done
VimLeave, VimLeavePre              " before quit
GUIEnter
ColorScheme                        " after :colorscheme
CursorHold                         " idle (after 'updatetime' ms)
CursorHoldI                        " same in insert mode
CursorMoved                        " any cursor move (expensive!)
CursorMovedI
TextChanged, TextChangedI          " buffer modified
InsertEnter, InsertLeave, InsertChange
WinEnter, WinLeave
TabEnter, TabLeave
TermOpen, TermClose                " :terminal
QuickFixCmdPre, QuickFixCmdPost
User MyEvent                       " custom event (fired with :doautocmd User MyEvent)
```

The augroup pattern (idempotent on reload):

```bash
augroup vimrc_trim_whitespace
    autocmd!                                " clear group's existing autocmds
    autocmd BufWritePre * %s/\s\+$//e
augroup END
```

Without `autocmd!`, sourcing your vimrc twice DOUBLES every autocmd. Always wrap in named augroup with autocmd! at top.

Common autocmd recipes:

```bash
" Restore last cursor position
augroup restore_cursor
    autocmd!
    autocmd BufReadPost * 
        \ if line("'\"") > 0 && line("'\"") <= line("$") |
        \   exe "normal! g`\"" |
        \ endif
augroup END

" Trim trailing whitespace on save (preserves cursor)
augroup trim_trailing
    autocmd!
    autocmd BufWritePre * call setline(1, getline(1, '$'))
    autocmd BufWritePre * %s/\s\+$//e
augroup END

" Filetype-specific settings
augroup ft_settings
    autocmd!
    autocmd FileType python setlocal ts=4 sw=4 et tw=88
    autocmd FileType make   setlocal noexpandtab
    autocmd FileType yaml   setlocal ts=2 sw=2 et
    autocmd FileType go     setlocal noexpandtab ts=4 sw=4
    autocmd FileType gitcommit setlocal spell tw=72
augroup END

" Highlight yanked text briefly (Neovim has built-in; for Vim 9):
augroup yank_highlight
    autocmd!
    autocmd TextYankPost * silent! call highlight#yank_flash()
augroup END

" Auto-resize splits when window is resized
augroup auto_resize
    autocmd!
    autocmd VimResized * wincmd =
augroup END

" Reload file when changed externally
augroup auto_reload
    autocmd!
    autocmd FocusGained,BufEnter * silent! checktime
augroup END
```

`:help autocmd-events` `:help :augroup` `:help BufWritePre`

## File Operations

```bash
:e file              " edit file
:e!                  " reload current file (DISCARD changes)
:e #                 " edit alternate file
:enew                " new empty buffer in current window
:find foo            " search 'path' option for foo
:browse e            " GUI file picker
:w                   " write
:w!                  " force write
:w newname           " write to newname (current file unchanged)
:saveas newname      " write to newname AND switch to it
:w !sudo tee %       " write file using sudo (the canonical sudo-save trick)
:r file              " insert file content after cursor
:r !date             " insert command output
:r !curl https://...
:!cmd                " run shell command
:!cmd %              " run cmd on current file (% = filename)
:!cmd %:p            " full path
:!cmd %:h            " head (directory)
:!cmd %:t            " tail (basename)
:!cmd %:r            " root (no extension)
:!cmd %:e            " extension only
:%!cmd               " filter buffer through cmd
:.,$!sort            " filter from cursor to end
```

Built-in netrw file browser:

```bash
:Ex                  " open netrw in current window (Explorer)
:Sex                 " open in horizontal split
:Vex                 " open in vertical split
:Tex                 " open in new tab
:Lex                 " open as left tree-style sidebar
-                    " from any buffer: open netrw at parent dir (vim-vinegar style)
```

Inside netrw:

```bash
<CR>                 " open file/enter dir
-                    " up one directory
%                    " create new file
d                    " create new directory
D                    " delete
R                    " rename
mt                   " mark target dir
mf                   " mark file
mc                   " copy marked to target
mm                   " move marked to target
mb                   " bookmark
qb                   " list bookmarks
gh                   " toggle hidden files
i                    " toggle list/long/wide views
o                    " open in horizontal split
v                    " open in vertical split
t                    " open in new tab
```

`:help :w` `:help filename-modifiers` `:help netrw-quickmap`

## Sessions and Views

Sessions save the entire layout — windows, buffers, marks, settings.

```bash
:mksession ~/.vim/sess/myproj.vim
:mksession! ~/.vim/sess/myproj.vim    " overwrite
:source ~/.vim/sess/myproj.vim        " restore session
vim -S ~/.vim/sess/myproj.vim         " start vim and load session

set sessionoptions=blank,buffers,curdir,folds,help,tabpages,winsize,terminal
" defaults are usually fine
```

Views save only the current window/buffer state (folds, cursor):

```bash
:mkview              " save view for current file
:loadview            " load saved view
:mkview 1            " save as numbered view
:loadview 1
" Auto-save/restore views per file:
augroup auto_view
    autocmd!
    autocmd BufWinLeave *.* mkview
    autocmd BufWinEnter *.* silent! loadview
augroup END
```

`:help session-file` `:help 'sessionoptions'` `:help mkview`

## Diff Mode

Side-by-side diff with merge support.

```bash
vimdiff a b          " open two files in diff mode
vimdiff a b c        " 3-way diff
vim -d a b           " same
:diffsplit other     " open other in split + diff
:vert diffsplit other
:diffthis            " mark current window for diff
:diffoff             " turn off diff in current window
:diffoff!            " turn off in all windows of current tab
:diffupdate          " refresh diff after edits
:windo diffthis      " mark all windows for diff
```

Diff navigation and merging:

```bash
]c                   " next change
[c                   " previous change
do                   " diff obtain — pull change from OTHER side
dp                   " diff put — push change to OTHER side
zo                   " open fold (unchanged regions are folded)
zc                   " close fold
zr / zm              " open/close one fold level
```

Three-way merge (git):

```bash
git config --global merge.tool vimdiff
git config --global mergetool.vimdiff.cmd \
  'vim -f -d -c "wincmd J" "$MERGED" "$LOCAL" "$BASE" "$REMOTE"'
git mergetool
```

`:help diff` `:help vimdiff` `:help do` `:help dp`

## Quickfix and Location Lists

Quickfix is a single global list (per Vim instance). Location list is window-local.

```bash
:copen               " open quickfix window
:cclose              " close
:cwindow             " open IF not empty (handy in scripts)
:cnext / :cn         " next quickfix entry
:cprev / :cp         " previous
:cfirst / :clast
:cnewer / :colder    " navigate quickfix HISTORY (multiple lists)
:cdo cmd             " run cmd on every quickfix entry
:cfdo cmd            " run cmd on each FILE in quickfix
:caddexpr expr       " add to quickfix
:cexpr expr          " set quickfix from expression
```

Populating quickfix:

```bash
:grep pattern files            " external grep — :h 'grepprg'
:grep pattern **/*.py
:grep -r pattern .
:vimgrep /pattern/g **/*.py    " Vim's internal grep (slower, regex flavor differs)
:vimgrep /pattern/gj **/*.go   " j = don't jump to first match
:make                          " runs 'makeprg', errors → quickfix
:helpgrep pattern              " search :help
:lgrep                         " location list version
:lvimgrep                      " location list version
```

Set 'grepprg' to use ripgrep (much faster):

```bash
set grepprg=rg\ --vimgrep\ --smart-case\ --no-heading
set grepformat=%f:%l:%c:%m
```

Then `:grep pattern` populates qf via rg.

Location list — same commands prefixed with `l`:

```bash
:lopen / :lclose / :lwindow
:lnext / :lprev / :lfirst / :llast
:lgrep / :lvimgrep / :lmake
:ldo cmd
```

The canonical "fix everything" workflow:

```bash
:vimgrep /TODO/gj **/*.py     " populate qf
:copen                         " inspect
:cdo s/TODO/DONE/gc | update   " confirm-replace + save in every match
```

`:help quickfix` `:help :grep` `:help :cdo` `:help 'errorformat'`

## Plugins — Manager Choices

Installing/updating plugins. Modern Vim has multiple options; pick ONE.

### vim-plug (most popular, simple, async)

```bash
" Install in ~/.vim/autoload/plug.vim:
curl -fLo ~/.vim/autoload/plug.vim --create-dirs \
  https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim

" In ~/.vimrc:
call plug#begin('~/.vim/plugged')
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-surround'
Plug 'junegunn/fzf', { 'do': { -> fzf#install() } }
Plug 'neoclide/coc.nvim', { 'branch': 'release' }
Plug 'lazy/loaded', { 'on': 'CmdName' }      " load on command
Plug 'language/specific', { 'for': 'go' }    " load for filetype
Plug 'branch/specific', { 'branch': 'main' }
Plug 'tag/specific', { 'tag': 'v1.0.0' }
call plug#end()

" Commands:
:PlugInstall         " install missing
:PlugUpdate          " update all
:PlugClean           " remove unused
:PlugStatus
:PlugDiff            " show pending updates
:PlugUpgrade         " update vim-plug itself
```

### Native Vim 8+ packages (no manager required)

```bash
mkdir -p ~/.vim/pack/myplugins/start
cd ~/.vim/pack/myplugins/start
git clone https://github.com/tpope/vim-fugitive

" Loaded automatically on Vim start.
" Use 'opt' instead of 'start' for lazy load + :packadd plugin-name
mkdir ~/.vim/pack/lazy/opt
cd ~/.vim/pack/lazy/opt
git clone https://github.com/lazy/plugin
" In .vimrc: :packadd plugin-name (when needed)

:helptags ~/.vim/pack/myplugins/start/vim-fugitive/doc
" Generate help tags for new plugins
```

### Other managers

```bash
" pathogen (oldest, just adds to runtimepath)
" vundle (legacy)
" dein.vim (powerful, async, ~/.cache/dein/)
" minpac (uses native packages under the hood)
" lazy.nvim (Neovim-only — see neovim sheet)
```

For serious config and modern features (LSP, treesitter, async): consider switching to Neovim. See the `neovim` sheet.

`:help packages` `:help packadd` `https://github.com/junegunn/vim-plug`

## Essential Plugins

The "everyone has these" set. Most are by Tim Pope (the gold standard for Vim plugin design).

```bash
" Editing power-tools
tpope/vim-surround       " ds( cs([ ysiw" — change/delete/add surrounds
tpope/vim-commentary     " gcc / gc{motion} — comment toggle
tpope/vim-repeat         " makes . work for plugins (REQUIRED for surround/commentary)
tpope/vim-unimpaired     " [b ]b [q ]q [<Space> — bracket pairs for nav
tpope/vim-abolish        " :Subvert/old/new/g — case-aware substitute
tpope/vim-eunuch         " :Rename :Delete :Mkdir :SudoWrite — Unix from Vim

" Git
tpope/vim-fugitive       " :Git, :Gblame, :Gdiffsplit, :Gread, :Gwrite
tpope/vim-rhubarb        " :GBrowse for GitHub
airblade/vim-gitgutter   " sign-column git diff (legacy)
mhinz/vim-signify        " sign-column for ANY VCS (git/hg/svn)
junegunn/gv.vim          " :GV git log browser

" Motion
justinmk/vim-sneak       " s{c}{c} — two-char f-style motion
ggandor/leap.nvim        " (Neovim) modern alternative
easymotion/vim-easymotion  " <leader><leader>w — pick visible target

" Fuzzy finder
junegunn/fzf            " core
junegunn/fzf.vim        " :Files :Buffers :Rg :GFiles
ctrlpvim/ctrlp.vim      " legacy fuzzy finder
yegappan/ctrlp          " (still works, no external deps)

" File browser
preservim/nerdtree      " classic tree (heavyweight)
tpope/vim-vinegar       " netrw enhancements (- to open parent)
justinmk/vim-dirvish    " minimal directory viewer

" Statusline
vim-airline/vim-airline " full-featured (slower)
itchyny/lightline.vim   " minimal, fast, opinionated

" LSP / completion
neoclide/coc.nvim       " node-based, mature, popular (LSP + snippets)
prabirshrestha/vim-lsp  " pure-vimscript LSP
dense-analysis/ale      " linting via async — also LSP-light
ycm-core/YouCompleteMe  " legacy compiled completion

" Git blame inline
APZelos/blamer.nvim     " virtual-text blame
f-person/git-blame.nvim " (Neovim) similar

" Undo
mbbill/undotree         " visualize the undo tree (:UndotreeToggle)

" Snippets
SirVer/ultisnips        " classic snippet engine
honza/vim-snippets      " massive snippet library

" Misc essentials
machakann/vim-highlightedyank   " flash yanked region (Vim 9)
junegunn/goyo.vim       " distraction-free writing
junegunn/limelight.vim  " dim non-current paragraphs
ryanoasis/vim-devicons  " filetype icons (requires patched font)
psliwka/vim-smoothie    " smooth scrolling
```

The "Tim Pope starter pack" alone (surround / commentary / repeat / fugitive / unimpaired) covers 80% of daily editing wins.

`:help plugin` `:help write-plugin`

## LSP Approaches in Vim

Vim 9 has no built-in LSP. Pick a layer.

### coc.nvim (most popular)

```bash
" Requires Node.js
Plug 'neoclide/coc.nvim', { 'branch': 'release' }

:CocInstall coc-tsserver coc-pyright coc-rust-analyzer coc-go coc-json
:CocList extensions
:CocConfig                " edit ~/.vim/coc-settings.json

" Mappings (canonical):
inoremap <silent><expr> <Tab> coc#pum#visible() ? coc#pum#next(1) : "\<Tab>"
inoremap <silent><expr> <S-Tab> coc#pum#visible() ? coc#pum#prev(1) : "\<S-Tab>"
inoremap <silent><expr> <CR> coc#pum#visible() ? coc#pum#confirm() : "\<CR>"
nmap <silent> gd <Plug>(coc-definition)
nmap <silent> gy <Plug>(coc-type-definition)
nmap <silent> gi <Plug>(coc-implementation)
nmap <silent> gr <Plug>(coc-references)
nmap <silent> [g <Plug>(coc-diagnostic-prev)
nmap <silent> ]g <Plug>(coc-diagnostic-next)
nnoremap <silent> K :call CocActionAsync('doHover')<CR>
nmap <leader>rn <Plug>(coc-rename)
xmap <leader>f  <Plug>(coc-format-selected)
nmap <leader>f  <Plug>(coc-format)
nmap <leader>ca <Plug>(coc-codeaction)
```

### vim-lsp + asyncomplete (pure vimscript, lighter)

```bash
Plug 'prabirshrestha/vim-lsp'
Plug 'mattn/vim-lsp-settings'         " auto-install servers
Plug 'prabirshrestha/asyncomplete.vim'
Plug 'prabirshrestha/asyncomplete-lsp.vim'

:LspInstallServer pylsp
:LspInstallServer gopls
:LspStatus
```

### ALE (mostly linting, but has LSP support)

```bash
Plug 'dense-analysis/ale'
let g:ale_linters = {'python': ['pylsp', 'ruff'], 'go': ['gopls', 'golangci-lint']}
let g:ale_fixers  = {'python': ['black'], 'go': ['gofmt'], 'javascript': ['prettier']}
let g:ale_fix_on_save = 1
```

For a TRULY native LSP experience: switch to Neovim, which has built-in `vim.lsp` and excellent plugins (mason.nvim, nvim-cmp, lspconfig). See the `neovim` sheet.

`:help lsp` (Vim 9.1+ has experimental built-in lsp client) `:CocList` `:ALEInfo`

## Search and Replace Across Files

Three reliable strategies.

### vimgrep + cdo

```bash
:vimgrep /old/gj **/*.py     " populate quickfix (j = don't jump)
:copen
:cdo s/old/new/gc | update   " run on every match, save modified
:cfdo %s/old/new/gc | update " same but per-FILE (faster for many matches in same file)
```

The `| update` writes only if buffer changed (cheaper than `| write`).

### grep with external tool

```bash
" Configure ripgrep:
set grepprg=rg\ --vimgrep\ --smart-case\ --hidden\ --no-heading
set grepformat=%f:%l:%c:%m

:grep 'old' src/             " populate quickfix
:cdo s/old/new/gc | update
```

### fzf + ripgrep (interactive)

```bash
:Rg pattern                  " from fzf.vim
" results in fzf — Tab to select multiple, Enter to send to qf
" :Rg uses :Rg!Bang for fullscreen
```

### fugitive's :Ggrep

```bash
:Ggrep pattern               " uses git grep, populates qf
" Then :cdo as before
```

### the canonical bulk-rename

```bash
:args **/*.py                " populate arglist
:argdo %s/old/new/gc | update
```

`:help :vimgrep` `:help :cdo` `:help :argdo` `:help fugitive-:Ggrep`

## Multi-File Edits

Run a command across many buffers/windows/tabs/files.

```bash
:argdo cmd               " each file in arglist
:bufdo cmd               " each buffer
:tabdo cmd               " each tab
:windo cmd               " each window
:cdo cmd                 " each quickfix entry
:cfdo cmd                " each FILE in quickfix
:ldo / :lfdo             " location list versions
```

Always pair edit commands with `update` (writes only if modified):

```bash
:bufdo %s/old/new/g | update
:argdo set ff=unix | update
:cfdo %s/foo/bar/g | update
```

Build an arglist:

```bash
:args **/*.py            " expand glob
:args `find . -name '*.go'`   " from shell
:argadd file             " add to current arglist
:argdelete file
:argument 3              " switch to 3rd file in arglist
:next / :prev            " navigate arglist
```

`:help argument-list` `:help :bufdo` `:help :update`

## The Help System

Vim's `:help` is its killer feature. Spend 10 minutes here and save weeks.

```bash
:help                   " open help (also F1)
:help topic             " jump to topic
:help :command          " ex command help (with colon)
:help i_CTRL-N          " insert-mode Ctrl-N (mode prefix syntax!)
:help v_o               " visual-mode 'o'
:help c_CTRL-R          " cmdline-mode Ctrl-R
:help map.txt           " entire map.txt help file
:help options.txt
:help eval.txt          " vimscript expressions
:help quickref          " one-page summary
:help index             " all commands by mode
```

Mode prefixes for help topics:

```bash
:help i_topic     " insert mode topic
:help v_topic     " visual mode
:help c_topic     " cmdline mode
:help t_topic     " terminal mode
:help o_topic     " operator-pending
:help n_topic     " (rare — normal-mode topic with ambiguity)
```

Navigation in help:

```bash
Ctrl-]            " follow link under cursor (the |link| syntax)
Ctrl-T            " jump back
Ctrl-O            " jump back (general jump-list)
:helpclose        " close help window
q                 " close help window (mapped by default)
:helpgrep pattern " search across all help files (populates qf)
:lhelpgrep pattern  " same, location list
```

Tab-completion magic:

```bash
:help c_<Tab>     " cycle through cmdline-mode topics
:help v_<C-D>     " list all visual-mode topics
:help options-<Tab>   " all options
```

External resource: vimhelp.org is the official help rendered as HTML.

`:help help` `:help help-context` `:help :helpgrep`

## Common Gotchas

Real, frequent, time-wasting traps — each shown broken then fixed.

### Modeless typing

```bash
" BAD: opening Vim and typing 'i love this' — first 'i' enters insert mode, but the rest looks fine. Until you press ESC and ':wq' is typed without the colon.
" FIX: ALWAYS check the mode indicator (:set showmode or statusline). When in doubt, mash ESC twice (idempotent in normal mode).
```

### Visual-block insert affects only first line during typing

```bash
" BAD: Ctrl-v jjj I // <type, only top line shows //, panic, undo>
" FIX: Don't panic. The '//' propagates to ALL selected lines AFTER you press ESC.
"     Watch for this — Vim previews on top line only.
```

### Clobbering register 0 with delete

```bash
" BAD:
yy                  " yank line — goes to "0
dd                  " delete line — goes to "" but NOT to "0
p                   " pastes the yank (good)
" but later:
dap                 " delete paragraph — overwrites "" with the paragraph
" you wanted to paste the original yank — it's still in "0:
"0p                 " explicit paste from yank register
" Or use blackhole to NOT overwrite:
"_dd                " delete without saving to any register
```

### :w fails on readonly file

```bash
" BAD:
:w
" E45: 'readonly' option is set (add ! to override)
" FIX:
:w!                 " force (works if YOU own the file)
" If the FILE is read-only on disk:
:!chmod +w %        " make writable
:w
" Or if it needs sudo:
:w !sudo tee %      " classic sudo-save
```

### Recursive mappings cause weird behavior

```bash
" BAD:
nmap <leader>w :w<CR>
nmap w :write<CR>      " now <leader>w may run :write twice
" FIX:
nnoremap <leader>w :w<CR>   " non-recursive — <leader>w is JUST :w<CR>
```

### Autocmd duplicates from re-sourcing vimrc

```bash
" BAD:
autocmd BufWritePre * %s/\s\+$//e
:source ~/.vimrc       " now ALL autocmds fire TWICE
:source ~/.vimrc       " now THREE times — slow, plus side effects
" FIX:
augroup my_trim
    autocmd!           " clears any existing autocmds in this group
    autocmd BufWritePre * %s/\s\+$//e
augroup END
" Now sourcing N times has the same effect as sourcing once.
```

### hlsearch lingers

```bash
" BAD: search /foo, then 5 minutes later all 'foo' instances still highlighted, distracting.
" FIX:
:noh<CR>           " or :nohlsearch — clears for current search
nnoremap <silent> <leader>h :nohlsearch<CR>
" Or auto-clear on cursor move:
augroup auto_nohl
    autocmd!
    autocmd CursorMoved * if !v:hlsearch | nohlsearch | endif
augroup END
```

### Pasting in insert mode mangles indent

```bash
" BAD: (mostly when pasting from terminal that simulates typing)
" Indent levels go ridiculous.
" FIX (terminal vim):
:set paste          " disables auto-indent / abbrev / autocmd for paste
" paste your content
:set nopaste

" BETTER FIX:
" Use the system clipboard from NORMAL mode:
"+p                 " paste from + register, no auto-indent involved
" Or use bracketed-paste-aware vim (modern Vim auto-detects):
" :h xterm-bracketed-paste — Vim 8+ usually handles this
```

### Lost work — Vim crashed or system rebooted

```bash
" BAD: opened file again, see "ATTENTION: Found a swap file" warning.
" FIX:
" 1. Read the warning — Vim explains.
" 2. Choose:
:r ~/.vim/swap/file.swp~     " read swap content into a buffer
" Or use :recover at startup:
vim -r myfile.txt
" After recovering, save AS A NEW FILE first to preserve the swap:
:saveas myfile.recovered.txt
" Then compare with original, decide which to keep, delete .swp:
rm ~/.vim/swap/.myfile.txt.swp
" To prevent future swap files (NOT recommended in production):
:set noswapfile
```

### Clipboard doesn't work

```bash
" BAD: "+y does nothing or pastes the wrong thing.
" CHECK:
:version | grep clipboard
" -clipboard means your build doesn't support it.
" FIX (Linux):
sudo apt install vim-gtk3      " or vim-gnome / vim-athena
" or:
brew install vim               " macOS

" Over SSH with X11 forwarding:
ssh -X user@host
" needs +xterm_clipboard build:
vim --version | grep xterm_clipboard
" Use OSC 52 escape codes for clipboard over plain SSH (no X):
" — see vim-oscyank plugin
```

### Undo doesn't survive close

```bash
" BAD: edit file, save, close, reopen — :undo says "already at oldest change"
" FIX:
set undofile
set undodir=~/.vim/undo
silent !mkdir -p ~/.vim/undo
" Now undo persists across sessions per file.
```

### Macro hits an error and aborts mid-way

```bash
" BAD: Macro deletes 5 lines but stops at line 3 because f; finds nothing.
" FIX:
:set nowrapscan      " then f; doesn't wrap (predictable failure)
" OR design macro to handle absence:
" - use :s instead of f-based motions for line-wide ops
" - use 'normal!' (with bang) to disable user mappings during replay
:'<,'>normal! @a
```

### "E486: Pattern not found" on %s

```bash
" BAD:
:%s/old/new/g     " errors out if 'old' not found
" FIX:
:%s/old/new/ge    " e flag = no error on no match
" or:
silent! %s/old/new/g
```

### Filename modifiers gone wrong

```bash
" BAD:
:!mv % %.bak     " % is the FILE name. But % from cmdline IS expanded.
" FIX:
:!mv % %:r.bak   " :r strips extension; full ref:
"   %     full filename relative to cwd
"   %:p   absolute path
"   %:h   directory part
"   %:t   tail (basename)
"   %:r   root (no extension)
"   %:e   extension only
" Test what % expands to:
:echo expand('%:p')
```

`:help message-history` `:help :messages` `:help recover.txt`

## Recovery

Vim creates `.swp` files (swap files) for crash recovery. They live next to the file or in `~/.vim/swap/`.

```bash
" When you reopen a file with a stale .swp:
" Vim shows the ATTENTION dialog:
"   (1) Another program may be editing the same file.
"   (2) An edit session for this file crashed.
" Options:
[O]pen Read-Only       " safe — view only
(E)dit anyway          " open the file (DANGEROUS if another vim has it)
(R)ecover              " load swap content into buffer
(Q)uit
(A)bort
(D)elete swap file     " just remove the swap (only if you're SURE)
```

The proper recovery flow:

```bash
" If you suspect a crash:
ls -la ~/.vim/swap/
ls -la /path/to/file/.*.swp

" Recover from command line:
vim -r myfile.txt              " loads swap
:saveas myfile.recovered.txt   " save AS NEW FILE — preserve swap until verified
" Compare manually:
:!diff myfile.recovered.txt myfile.txt
" Once happy:
rm /path/to/.myfile.txt.swp
```

Detecting external file changes:

```bash
:checktime                     " check if any buffer changed externally
set autoread                   " auto-reload changed files (when safe)
" autoread doesn't fire on its own — pair with autocmd:
augroup auto_check
    autocmd!
    autocmd FocusGained,BufEnter,CursorHold * silent! checktime
augroup END
```

If Vim is unresponsive (huge regex hang, runaway syntax):

```bash
Ctrl-c                  " usually breaks
" If terminal frozen, send signal from another shell:
killall -TERM vim       " polite — Vim writes swap then exits
killall -KILL vim       " last resort — leaves swap (recoverable)
```

`:help swap-file` `:help :recover` `:help E325`

## Performance

Vim is fast, but huge files, complex regex, or runaway syntax can still tank you.

```bash
set lazyredraw          " skip screen redraws during macros and scripts
set ttyfast             " (default in modern Vim) — fast terminal connection
set updatetime=300      " ms — affects CursorHold, swap, signs
set synmaxcol=240       " stop syntax highlighting past col 240 (long-line saver)
set redrawtime=10000    " cap (ms) on syntax/regex highlighting per cycle
set regexpengine=1      " 0=auto, 1=old NFA, 2=new NFA — try 1 if syntax slow
set re=1                " short alias
syntax sync minlines=200    " how far back to sync syntax (per-language)
syntax sync fromstart   " always re-parse from start (SLOW but correct)
```

Engine choice — when does each win:

```bash
" New NFA engine (default) is faster for most patterns but pathologically slow
" on certain patterns (look-aheads, deeply nested groups).
" If editing feels laggy on a specific filetype:
:set re=1               " switch to old engine, often a 10x speedup
:syntime on             " enable syntax timing
:syntime report         " see which patterns are slow
```

Profiling Vim startup:

```bash
vim --startuptime startup.log
" lines like:
" 003.123  003.123: sourcing /etc/vim/vimrc
" 050.456  047.333: sourcing ~/.vim/plugged/coc.nvim/...
" Sort biggest contributors:
sort -k2 -n startup.log | tail -20
```

Plugin profiling at runtime:

```bash
:profile start profile.log
:profile func *
:profile file *
" reproduce slowness
:profile pause
:noautocmd qa!
" Then read profile.log to see hot functions.
```

Big-file mode (the canonical "open this 5GB log without dying"):

```bash
" Detect and disable expensive features for huge files:
augroup big_files
    autocmd!
    autocmd BufReadPre * if getfsize(expand('%')) > 10*1024*1024
        \ | syntax off
        \ | setlocal noundofile noswapfile bufhidden=unload
        \ | setlocal eventignore+=FileType
        \ | setlocal foldmethod=manual
        \ | endif
augroup END

" Or quick interactive:
:syntax off
:set noundofile noswapfile
:set bufhidden=unload
```

`:help slow-terminal` `:help :syntime` `:help --startuptime` `:help :profile`

## Migration to Neovim

Neovim is a fork of Vim with major modernizations. 99% of Vim configs work unchanged.

What's the same:

```bash
" All motions, operators, text-objects, modes, ex commands.
" Vimscript (legacy and Vim9 — Neovim accepts BOTH but prefers Lua).
" The :help system (Neovim's help is a superset).
" Most plugins (especially Tim Pope's, vim-fugitive, vim-surround).
" The .vimrc loads if you alias init.vim → ~/.config/nvim/init.vim
```

What's different (and why you'd migrate):

```bash
" - Lua config: ~/.config/nvim/init.lua (faster, real language)
" - Native LSP: vim.lsp built-in (no coc.nvim needed)
" - Native treesitter: incremental parsing for highlighting/folding
" - Async by default: jobs and timers without compat layer
" - Floating windows, virtual text, extmarks
" - :terminal is more reliable
" - vim.api lua API for plugins
" - Built-in package manager via vim.pack (Neovim 0.12+)
" - lazy.nvim is the de facto modern package manager
" - Better default keymaps in some cases
```

To migrate gradually:

```bash
" 1. Symlink init:
ln -s ~/.vimrc ~/.config/nvim/init.vim
nvim                        " runs your existing config

" 2. Replace plugins with Neovim-native equivalents:
"    coc.nvim → nvim-cmp + nvim-lspconfig
"    NERDTree → nvim-tree.lua
"    vim-airline → lualine.nvim

" 3. Move config to Lua incrementally:
mv ~/.config/nvim/init.vim ~/.config/nvim/init.lua
" Wrap legacy in vim.cmd([[ ... ]]) blocks.
```

See the `neovim` sheet for the full Neovim setup.

`:help nvim-from-vim` `:help nvim-features` `https://neovim.io`

## Idioms

The high-leverage daily-driver patterns. Memorize these.

### Text-object operators

```bash
ciw           " change inner word — most-used rename
cit           " change inside HTML tag
ci"           " change inside double quotes
ci'           " change inside single
ci(           " change inside parens (also cib)
ci{           " change inside braces (also ciB)
da"           " delete a "string" (with quotes)
yi}           " yank inside braces
viw           " visual-select word
vap           " visual-select paragraph
=ip           " auto-indent paragraph
gqap          " textwidth-reformat paragraph
```

### Macros for repetitive edits

```bash
qq            " start recording into q
... do the edit on one line ...
q             " stop
@q            " replay
99@q          " do it 99 times (stops on error)
:%normal! @q  " do it on every line
```

### The * trick

```bash
*             " search word under cursor — no need to type it
cgn           " change next match (use * then cgn, then . to repeat)
:%s//new/g    " empty pattern reuses last search — handy after *
```

### :g for bulk operations

```bash
:g/TODO/d                  " delete every line with TODO
:g/^$/d                    " delete blank lines
:g/^/m0                    " reverse all lines (move every line to top)
:g/^/y A                   " yank every line into register a (capital appends)
:g/pat/normal @q           " run macro q on every matching line
:g/pat1/.,/pat2/d          " delete from pat1 to pat2 (block delete)
:v/keep/d                  " delete lines NOT matching 'keep'
```

### Replace word under cursor

```bash
" Word under cursor:
:%s/\<<C-r><C-w>\>/new/g   " <C-r><C-w> inserts cursor word literally
" Or with confirmation:
:%s/\<<C-r><C-w>\>/new/gc

" Even faster — using *:
*                           " search-word
:%s//new/g                  " sub with empty pattern (uses last search)
```

### Replace with clipboard

```bash
" Visual-select target, paste from clipboard, blackhole the deleted:
viw"_d"+P                   " inner-word, blackhole-delete, paste before
" Or as a one-shot map:
xnoremap <leader>p "_dP
" Then: viw <leader>p   pastes clipboard over selection without yank-loss
```

### Increment numbers

```bash
Ctrl-a            " increment number under cursor
Ctrl-x            " decrement
5 Ctrl-a          " add 5
" Visual-block + g Ctrl-a creates an arithmetic sequence:
" select 10 lines all containing '0', press g Ctrl-a → 1 2 3 ... 10
```

### Surround (vim-surround idioms — install vim-surround first)

```bash
ysiw"           " surround inner-word with "
ysiw)           " surround with ) (no inner space)
ysiw(           " surround with ( ) — WITH spaces (parens with space)
yss"            " surround whole line with "
S"              " (visual mode) surround selection with "
ds"             " delete surrounding "
cs"'            " change surrounding " to '
cs)t<em>        " change surrounding ) to <em></em>
```

### Comment toggle (vim-commentary idioms)

```bash
gcc             " toggle comment current line
gc{motion}      " toggle comment over motion
gcap            " toggle comment a paragraph
gcG             " toggle to end of file
gc (visual)     " toggle comment on selection
```

### Quickfix → bulk edit

```bash
:vimgrep /pattern/gj **/*.py   " populate
:copen                          " review
:cdo s/pattern/replace/gc | update   " confirm + save
```

### Save and source vimrc instantly

```bash
nnoremap <leader>ev :edit $MYVIMRC<CR>
nnoremap <leader>sv :source $MYVIMRC<CR>
```

### Quick word-boundary substitute

```bash
:%s/\<old\>/new/g           " whole-word match (avoids 'older' → 'newer')
:%s/\<<C-r><C-w>\>/new/gc   " word-under-cursor with confirm
```

### Use :normal for "do this normal-mode thing on a range"

```bash
:%normal A;                 " append ; to every line
:%normal! @q                " run macro q on every line
:'<,'>normal I// <Esc>      " comment selected lines (insert "// ")
```

`:help text-objects` `:help :g` `:help :normal` `:help v_v`

## Tips

Small wins that compound.

```bash
" Open Vim at a specific location:
vim file +30                   " jump to line 30
vim file +/pattern             " jump to first match of pattern
vim file -c 'normal Gzz'       " run a command on open

" Multiple files:
vim file1 file2 file3          " arglist; navigate with :next / :prev
vim -p file1 file2             " open in tabs
vim -o file1 file2             " open in horizontal splits
vim -O file1 file2             " open in vertical splits

" Read stdin:
ls | vim -                     " - means stdin
git diff | vim -

" Pipe Vim output:
vim -e -s -c '%s/old/new/g' -c 'wq' file   " ed-like batch edit (ex mode + silent)
ex -s -c '%s/old/new/g | x' file           " same effect via ex

" Hex editor mode:
:%!xxd                         " hex dump in buffer
:%!xxd -r                      " convert back

" Sort + dedupe in 2 keystrokes:
ggVG :sort u

" Reformat code (gq) over a range:
gggqG                          " format whole file (uses 'formatprg' or textwidth)
gqip                           " format paragraph

" Numbered jumps via marks:
'0                             " back to last cursor position when Vim closed
'1, '2, ...                    " older positions

" :g + :m for sorting:
:g/^/m0                        " reverse every line (each line moves to top)
:g/foo/m$                      " move every line containing foo to bottom

" Visual-select last paste:
gp                             " paste then put cursor AFTER
gP                             " paste before, cursor after
`[v`]                          " visual-select last yank/paste/change
gv                             " reselect last visual

" Insert literal control chars:
i Ctrl-v Ctrl-]                " inserts ] (literal escape)
i Ctrl-v Tab                   " literal tab even with expandtab

" Count matches:
:%s/pattern//gn                " 'n' flag = report only, don't substitute

" Reload a file:
:e!                            " discard changes and reload from disk
:edit                          " reload (only if not modified)
:checktime                     " trigger autoread check

" Useful $MYVIMRC:
:echo $MYVIMRC                 " print the path of YOUR vimrc
:edit $MYVIMRC                 " edit it
:source $MYVIMRC               " reload

" Find your runtimepath / scripts loaded:
:set runtimepath?
:scriptnames                   " every file Vim has sourced this session

" Help in pop-up window:
:help! topic                   " open help WITHOUT splitting (replaces current)
:vert help topic               " open help in vertical split

" The most underrated motion:
gi                             " resume insert at last insert position
g;                             " jump to PREVIOUS change
g,                             " jump to NEWER change
gv                             " reselect last visual

" Auto-complete from any buffer (works without LSP):
i Ctrl-n                       " word-completion from open buffers
i Ctrl-x Ctrl-l                " whole-line completion

" Quick-edit related files:
:e %:h/<Tab>                   " complete sibling files
:e %:r.h                       " sibling .h to current .c
" Or use vim-eunuch's :Move and :Rename

" Force write a file even if missing parent dir:
:!mkdir -p %:h
:w

" Diff current buffer against saved:
:w !diff % -

" The hidden-clipboard trick (when no +clipboard):
:!cat % | pbcopy               " macOS — pipe file to clipboard
:'<,'>w !pbcopy                " write visual selection to clipboard

" Print current word as ASCII / hex:
ga                             " under cursor: char info (dec/hex/oct)
g8                             " UTF-8 byte sequence under cursor

" The :! trick — execute current line as shell:
:.!sh                          " replaces current line with output of running it as shell

" Filter buffer/section through external sort:
:.,$!sort -u
:%!python -m json.tool         " pretty-print JSON
:%!yq .                        " pretty-print YAML

" Show changes made in this session:
:changes                       " list of changes
:earlier 5m                    " roll back to 5 minutes ago
:later 5m                      " roll forward
:earlier 100                   " 100 changes ago
g-                             " step back through undo tree
g+                             " step forward
:undolist                      " (Neovim) view tree

" Visual-block on-the-fly column insert:
Ctrl-v Gg I // <Esc>           " comment from cursor to top of file

" Quick move line up/down (without plugin):
:m +1                          " current line down by 1
:m -2                          " current line up by 1 (move to N-2 of self)

" Quick swap two lines:
ddp                            " delete line, paste below = swap with line below
ddkP                           " swap with line above
```

## See Also

- [neovim](../editors/neovim.md) — Lua-based fork with native LSP and treesitter
- [tmux](../shell/tmux.md) — terminal multiplexer often paired with Vim
- [bash](../shell/bash.md) — shell for `:!` calls and `vim` invocation
- [zsh](../shell/zsh.md) — alternative shell with deeper completion
- [polyglot](../languages/polyglot.md) — language-agnostic patterns useful inside Vim

## References

- `:help` — Vim's built-in help (try `:help quickref`, `:help index`, `:help help`)
- `:help options.txt` — every option, alphabetical
- `:help map.txt` — mappings reference
- `:help eval.txt` — vimscript expressions and functions
- `:help vim9.txt` — Vim9 script (faster scripting, Vim 9.0+)
- `:help recover.txt` — swap and recovery
- `:help quickfix.txt` — qf and location lists
- vimhelp.org — official help rendered as HTML, fully searchable
- `learnvimscriptthehardway.stevelosh.com` — Steve Losh's free book on vimscript
- "Practical Vim" by Drew Neil — the canonical book; tip-based, deep
- "Modern Vim" by Drew Neil — async, terminal, tmux, Neovim
- vimcasts.org — Drew Neil's video screencasts
- github.com/iggredible/Learn-Vim — modern free Vim guide
- `vim-galore` (github.com/mhinz/vim-galore) — comprehensive cheat-sheet wiki
- `vim.fandom.com/wiki/Vim_Tips_Wiki` — community recipes (uneven but enormous)
- `vim --version` — your build's feature list (`+clipboard`, `+python3`, etc.)
- `vimtutor` — interactive 30-minute tutorial included with Vim
