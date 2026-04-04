# Vim (Vi IMproved)

> Modal text editor — fast, ubiquitous, and endlessly configurable.

## Modes

```bash
i          # insert mode (before cursor)
a          # insert mode (after cursor)
I          # insert at beginning of line
A          # insert at end of line
o          # open line below and insert
O          # open line above and insert
v          # visual mode (character)
V          # visual mode (line)
Ctrl-v     # visual block mode
R          # replace mode
r          # replace single character then back to normal
gR         # virtual replace mode (respects tabs)
Esc        # return to normal mode
Ctrl-[     # return to normal mode (alternative)
Ctrl-c     # return to normal mode (no abbreviation check)
:          # command-line mode
```

## Motions

### Character and Word

```bash
h j k l          # left, down, up, right
w                # next word start
W                # next WORD start (whitespace-delimited)
b                # previous word start
B                # previous WORD start
e                # next word end
E                # next WORD end
ge               # previous word end
gE               # previous WORD end
0                # start of line
^                # first non-blank character
$                # end of line
g_               # last non-blank character of line
f{char}          # forward to char (inclusive)
F{char}          # backward to char
t{char}          # forward to char (exclusive)
T{char}          # backward to char (exclusive)
;                # repeat last f/F/t/T
,                # repeat last f/F/t/T (reverse)
```

### Line and Document

```bash
gg               # go to first line
G                # go to last line
42G              # go to line 42
:42              # go to line 42 (command form)
{                # previous paragraph
}                # next paragraph
(                # previous sentence
)                # next sentence
%                # matching bracket
Ctrl-d           # half page down
Ctrl-u           # half page up
Ctrl-f           # full page down
Ctrl-b           # full page up
Ctrl-e           # scroll down one line (cursor stays)
Ctrl-y           # scroll up one line (cursor stays)
H                # top of screen
M                # middle of screen
L                # bottom of screen
zz               # center cursor on screen
zt               # cursor to top of screen
zb               # cursor to bottom of screen
```

## Text Objects

```bash
# used with operators (d, c, y, v, etc.)
iw               # inner word
aw               # a word (includes surrounding space)
iW               # inner WORD
aW               # a WORD (includes surrounding space)
is               # inner sentence
as               # a sentence
ip               # inner paragraph
ap               # a paragraph
i"               # inside double quotes
a"               # around double quotes (includes quotes)
i'               # inside single quotes
a'               # around single quotes
i`               # inside backticks
a`               # around backticks
i(  or  ib       # inside parentheses
a(  or  ab       # around parentheses
i{  or  iB       # inside braces
a{  or  aB       # around braces
it               # inside HTML/XML tag
at               # around HTML/XML tag (includes tags)
i[               # inside brackets
a[               # around brackets
i<               # inside angle brackets
a<               # around angle brackets
```

### Common Combinations

```bash
diw              # delete inner word
ciw              # change inner word
ci"              # change inside quotes
ca"              # change around quotes (including the quotes)
yap              # yank around paragraph
da(              # delete around parentheses
vi{              # select inside braces
dit              # delete inside tag
dat              # delete around tag (including the tags)
yi[              # yank inside brackets
ci'              # change inside single quotes
gUiw             # uppercase inner word
guaw             # lowercase a word
```

## Operators

```bash
d                # delete
c                # change (delete + insert mode)
y                # yank (copy)
>                # indent right
<                # indent left
=                # auto-indent
gq               # format/wrap text
gw               # format/wrap text (cursor stays)
gu               # lowercase
gU               # uppercase
g~               # toggle case
~                # toggle case (single char, moves forward)
!                # filter through external command
```

### Operator + Motion Examples

```bash
dd               # delete line
yy               # yank line
cc               # change entire line
D                # delete to end of line
C                # change to end of line
Y                # yank entire line (use y$ for end of line)
dw               # delete word
d$               # delete to end of line
d0               # delete to start of line
dG               # delete to end of file
dgg              # delete to start of file
3dd              # delete 3 lines
d3w              # delete 3 words
dfx              # delete through next x
dtx              # delete up to next x
d/pattern        # delete to next match of pattern
c3w              # change 3 words
>3j              # indent current line and 3 below
=G               # auto-indent from cursor to end of file
gUw              # uppercase next word
gu$              # lowercase to end of line
```

## Registers

### Basic Registers

```bash
"ayy             # yank line into register a
"ap              # paste from register a
"Ayy             # append yank to register a (capital = append)
"+y              # yank to system clipboard
"+p              # paste from system clipboard
"*y              # yank to primary selection (X11)
"*p              # paste from primary selection
"0p              # paste last yank (not delete)
""               # unnamed register (default)
"_dd             # delete to black hole (don't save)
:reg             # show all registers
:reg a           # show register a
```

### Advanced Registers

```bash
# numbered registers
"0               # last yank
"1               # last delete (1+ lines or motions)
"2 - "9          # previous deletes (shifted from "1 on each delete)

# read-only registers
"%               # current filename
"#               # alternate filename (last edited file)
".               # last inserted text
":               # last ex command executed

# special registers
"-               # small delete register (less than one line)
"/               # last search pattern
"=               # expression register (evaluate expression, result is used)

# using expression register in insert mode
Ctrl-r =         # opens expression prompt at bottom
                 # type 2+2<Enter> to insert 4
                 # type strftime('%Y-%m-%d')<Enter> for current date
                 # type expand('%')<Enter> for current filename

# using registers in insert mode
Ctrl-r a         # insert contents of register a
Ctrl-r "         # insert unnamed register
Ctrl-r +         # insert system clipboard
Ctrl-r /         # insert last search pattern
Ctrl-r :         # insert last command
Ctrl-r .         # insert last inserted text
Ctrl-r %         # insert current filename
Ctrl-r Ctrl-w    # insert word under cursor (in command-line mode)
```

## Search and Replace

### Basic Search

```bash
/pattern         # search forward
?pattern         # search backward
n                # next match
N                # previous match
*                # search word under cursor (forward)
#                # search word under cursor (backward)
g*               # like * but partial match (no \b word boundaries)
g#               # like # but partial match
gd               # go to local declaration of word under cursor
gD               # go to global declaration of word under cursor
```

### Substitute

```bash
:s/old/new/      # replace first on current line
:s/old/new/g     # replace all on current line
:%s/old/new/g    # replace all in file
:%s/old/new/gc   # replace all with confirmation
:5,20s/old/new/g # replace in lines 5-20
:'<,'>s/old/new/g # replace in visual selection
:%s/old/new/gi   # replace all, case insensitive
:%s/old/new/gI   # replace all, case sensitive (override ignorecase)
:%s//new/g       # replace last search pattern with new
:&&              # repeat last substitution with same flags
:~               # repeat last substitute with last search pattern
:%s/\v(\w+)/\U\1/g     # uppercase all words (very magic mode)
:%s/\s\+$//e            # remove trailing whitespace
:%s/\n\n\+/\r\r/g       # collapse multiple blank lines to one
:%s/\t/    /g            # replace tabs with 4 spaces
```

### Advanced Search Patterns

```bash
# very magic mode — all special chars are magic (like PCRE)
/\v(foo|bar)     # alternation without escaping parens
/\vfoo\d+        # digits without escaping +
/\v<word>        # word boundary match

# magic mode (default) — some chars need escaping
/foo\|bar        # alternation (must escape pipe)
/foo\(bar\)      # grouping (must escape parens)

# match delimiters — control what the match returns
/foo\zsbar\ze    # matches "foobar" but only "bar" is the match
                 # useful in :s — only replaces the \zs..\ze portion

# case control
/\cfoo           # case insensitive search for foo
/\Cfoo           # case sensitive search for foo (overrides ignorecase)

# lookahead and lookbehind
/foo\@=bar       # NOT how you think — use \v for clarity
/\v(foo)@=bar    # bar preceded by foo (not really — see below)
/\vbar(baz)@=    # bar followed by baz (positive lookahead)
/\vbar(baz)@!    # bar NOT followed by baz (negative lookahead)
/\v(foo)@<=bar   # bar preceded by foo (positive lookbehind)
/\v(foo)@<!bar   # bar NOT preceded by foo (negative lookbehind)

# useful search patterns
/\v^\s*$         # blank lines
/\v\S+@\S+       # rough email pattern
/\v(TODO|FIXME|HACK|XXX)  # find code annotations
/\v^(.*)$\n\1$   # find duplicate adjacent lines
```

## Global Command

```bash
# :g/pattern/command — execute command on every line matching pattern
# :g!/pattern/command or :v/pattern/command — execute on non-matching lines

# deletion
:g/pattern/d     # delete all lines matching pattern
:g!/pattern/d    # delete all lines NOT matching pattern (keep matches)
:v/pattern/d     # same as :g!/pattern/d
:g/^$/d          # delete all blank lines
:g/^\s*$/d       # delete all blank/whitespace-only lines
:g/^#/d          # delete all comment lines (# style)

# moving and copying
:g/pattern/t$    # copy all matching lines to end of file
:g/pattern/m0    # move all matching lines to top of file
:g/TODO/t$       # copy all TODO lines to end of file

# executing normal mode commands on matches
:g/pattern/norm A;         # append semicolon to every matching line
:g/pattern/norm @a         # run macro a on every matching line
:g/pattern/norm I//        # comment out every matching line (C-style)
:g/^/norm J                # join every line with the next (collapse file)
:g/pattern/norm dd         # same as :g/pattern/d but via normal mode

# combining with ranges
:g/pattern/s/foo/bar/g     # on matching lines, substitute foo with bar
:g/^Chapter/.,/^Chapter/-1sort  # sort sections between Chapter headings

# display matches
:g/pattern/p     # print matching lines (with line numbers)
:g/pattern/#     # print matching lines with line numbers
:g/TODO/         # print all lines containing TODO

# inverse and chaining
:v/keep_this/d   # delete everything except lines containing keep_this
:g/pattern/norm >>         # indent all matching lines
```

## Macros

### Basic Macros

```bash
qa               # start recording macro into register a
q                # stop recording
@a               # play macro a
@@               # replay last macro
5@a              # play macro a 5 times
:%norm @a        # run macro a on every line in the file
:'<,'>norm @a    # run macro a on every line in visual selection
```

### Advanced Macros

```bash
# recursive macro — runs until it fails (e.g., end of file)
qaq              # clear register a first
qa               # start recording
0f,r;            # example: go to start, find comma, replace with semicolon
j                # move to next line
@a               # call itself recursively
q                # stop recording
@a               # run — stops when j fails at last line or f fails

# visual mode macros
# select lines in visual mode, then:
:'<,'>norm @a    # run macro a on each selected line

# editing a macro stored in register a
"ap              # paste macro contents into buffer
# edit the text as needed
"ayy             # yank edited macro back into register a
# or use :let
:let @a = 'iHello World^[' # set register a directly (^[ is Esc, type Ctrl-v Esc)

# appending to a macro (capital letter appends)
qA               # append more steps to macro in register a
# additional steps
q                # stop

# view macro contents
:reg a           # show what's in register a
```

## Marks

```bash
ma               # set mark a at cursor (local to buffer)
mA               # set mark A at cursor (global — across files)
'a               # jump to mark a (beginning of line)
`a               # jump to mark a (exact position)
''               # position before last jump (line)
``               # position before last jump (exact)
'.               # last change (line)
`.               # last change (exact position)
'^               # last insert position
'[               # start of last yank/change (line)
`]               # end of last yank/change (exact)
'"               # position when last exited file
:marks           # list all marks
:delmarks a      # delete mark a
:delmarks a-d    # delete marks a through d
:delmarks!       # delete all lowercase marks

# automatic marks
`<               # start of last visual selection
`>               # end of last visual selection
```

## Splits and Tabs

### Splits

```bash
:split file      # horizontal split
:sp file         # horizontal split (short form)
:vsplit file     # vertical split
:vsp file        # vertical split (short form)
:new             # new empty horizontal split
:vnew            # new empty vertical split
Ctrl-w s         # split current window horizontal
Ctrl-w v         # split current window vertical
Ctrl-w h/j/k/l  # navigate between splits
Ctrl-w H/J/K/L  # move split to edge
Ctrl-w =         # equalize split sizes
Ctrl-w _         # maximize height
Ctrl-w |         # maximize width
Ctrl-w q         # close split
Ctrl-w c         # close split (same as :close)
Ctrl-w o         # close all splits except current
Ctrl-w r         # rotate splits downward/rightward
Ctrl-w R         # rotate splits upward/leftward
Ctrl-w x         # exchange current split with next
Ctrl-w T         # move current split to new tab
Ctrl-w +         # increase height by 1
Ctrl-w -         # decrease height by 1
Ctrl-w >         # increase width by 1
Ctrl-w <         # decrease width by 1
10 Ctrl-w +      # increase height by 10
:resize 20       # set height to 20
:vertical resize 80  # set width to 80
```

### Tabs

```bash
:tabnew file     # open file in new tab
:tabnew          # new empty tab
:tabn            # next tab
:tabp            # previous tab
:tabclose        # close tab
:tabonly          # close all tabs except current
:tabmove 0       # move tab to first position
:tabmove $       # move tab to last position
gt               # next tab
gT               # previous tab
2gt              # go to tab 2
:tabs            # list all tabs
```

## Visual Mode

```bash
v                # character-wise selection
V                # line-wise selection
Ctrl-v           # block selection
gv               # reselect last visual selection
o                # move to other end of selection
O                # move to other corner of block selection

# after selecting:
d                # delete selection
c                # change selection
y                # yank selection
>                # indent
<                # outdent
=                # auto-indent
:                # command on selection
U                # uppercase
u                # lowercase
~                # toggle case
J                # join lines
gJ               # join lines without spaces
gq               # format/wrap selection
!                # filter through external command
:sort            # sort selected lines
```

### Block Mode Tricks

```bash
# insert text on multiple lines:
Ctrl-v           # enter block mode
jjj              # select lines
I                # insert at block start
# type text
Esc              # apply to all lines

# append to multiple lines:
Ctrl-v jjj $     # select to end of lines
A                # append
# type text
Esc

# change block (replace column):
Ctrl-v jjj       # select block
c                # change — type new text
Esc              # apply to all lines

# increment numbers in column:
Ctrl-v jjj       # select numbers in a column
g Ctrl-a         # create incrementing sequence (1,2,3,4...)

# delete column:
Ctrl-v jjj       # select column
d                # delete the column
```

## Folds

```bash
zf{motion}       # create fold (manual mode)
zfap             # fold around paragraph
zo               # open fold
zO               # open fold recursively
zc               # close fold
zC               # close fold recursively
za               # toggle fold
zA               # toggle fold recursively
zR               # open all folds
zM               # close all folds
zd               # delete fold
zE               # delete all folds
zj               # move to next fold
zk               # move to previous fold
[z               # go to start of current open fold
]z               # go to end of current open fold
:set foldmethod=manual   # create folds manually with zf
:set foldmethod=indent   # fold by indentation
:set foldmethod=syntax   # fold by syntax
:set foldmethod=marker   # fold by markers ({{{ and }}})
:set foldmethod=expr     # fold by expression
:set foldlevel=2         # fold depth to keep open
:set nofoldenable        # disable folding entirely
```

## Ex Commands

```bash
:w               # save
:w filename      # save as
:q               # quit
:wq              # save and quit
:x               # save and quit (only writes if changes)
:q!              # quit without saving
:qa              # quit all windows
:qa!             # quit all without saving
:wqa             # save and quit all
:e file          # edit file
:e!              # reload current file (discard changes)
:e #             # edit alternate file (last edited)
:r file          # read file into buffer (below cursor)
:0r file         # read file at top of buffer
:r !cmd          # read command output into buffer
:!cmd            # run shell command
:.!cmd           # replace current line with command output
:%!sort          # sort entire file
:%!column -t     # align columns in entire file
:%!jq .          # format JSON (requires jq)
:set nu          # show line numbers
:set nonu        # hide line numbers
:set rnu         # relative line numbers
:set paste       # paste mode (no auto-indent)
:set nopaste     # exit paste mode
:noh             # clear search highlighting
:earlier 5m      # undo to 5 minutes ago
:later 5m        # redo to 5 minutes later
:earlier 3f      # undo to 3 file-writes ago
:sort            # sort selected/all lines
:sort u          # sort and remove duplicates
:sort!           # sort reverse
:sort n          # sort numerically
:sort /pattern/  # sort by match of pattern
:changes         # list change history
:jumps           # list jump history
:undolist        # show undo tree branches
```

### Advanced Ex Commands

```bash
# :norm — execute normal mode commands from command line
:%norm A;          # append semicolon to every line
:%norm I//         # comment every line with //
:'<,'>norm @a      # run macro a on selection

# multi-buffer commands
:argdo %s/foo/bar/ge | update  # substitute in all argument files
:bufdo %s/foo/bar/ge | update  # substitute in all buffers
:windo diffthis                # diff all visible windows
:tabdo %s/foo/bar/ge           # substitute in all tabs

# :cfdo — execute on each file in quickfix list
:cfdo %s/foo/bar/ge | update   # substitute in all quickfix files

# ranges with patterns
:/start/,/end/d                # delete from line matching start to line matching end
:/start/,/end/s/foo/bar/g      # substitute between pattern-matched lines
:/start/+1,/end/-1d            # delete between patterns (exclusive of pattern lines)
:.,+5d                         # delete current line and 5 below
:-3,.d                         # delete 3 lines above through current
:1,/pattern/d                  # delete from line 1 to first match

# reading and writing ranges
:10,20w outfile.txt            # write lines 10-20 to file
:10,20w >> outfile.txt         # append lines 10-20 to file

# executing register content as ex command
:@a                            # execute register a as ex command
:@:                            # repeat last ex command
```

## Quickfix and Location Lists

```bash
# quickfix list — global to the vim instance
:copen           # open quickfix window
:cclose          # close quickfix window
:cnext           # go to next item       (also: ]q with unimpaired)
:cprev           # go to previous item   (also: [q with unimpaired)
:cfirst          # go to first item
:clast           # go to last item
:cc 5            # go to item number 5
:colder          # go to older quickfix list
:cnewer          # go to newer quickfix list

# populating quickfix
:grep pattern **/*.go       # grep and load results into quickfix
:vimgrep /pattern/ **/*.go  # use vim's internal grep
:vimgrep /TODO/ %           # find all TODOs in current file
:cexpr system('make 2>&1')  # load command output into quickfix

# operating on quickfix entries
:cdo s/old/new/g | update   # substitute in each quickfix entry (line-level)
:cfdo %s/old/new/ge | update # substitute in each quickfix file (file-level)

# location list — local to each window (same commands, l prefix)
:lopen           # open location list window
:lclose          # close location list
:lnext           # go to next item
:lprev           # go to previous item
:lfirst          # go to first item
:llast           # go to last item
:lgrep pattern **/*.go      # populate location list with grep
:lvimgrep /pattern/ **/*.go # populate with vimgrep
:ldo s/old/new/g | update   # substitute in each location entry
```

## Diff Mode

```bash
# starting diff mode
vimdiff file1 file2          # open two files in diff mode from shell
vim -d file1 file2 file3     # diff up to 4 files
:diffthis                    # mark current window for diffing
:diffoff                     # turn off diff mode for current window
:diffoff!                    # turn off diff mode for all windows

# workflow: diff two open buffers
:vsp file2       # open second file in vertical split
:windo diffthis  # diff both windows

# navigating diffs
]c               # jump to next change
[c               # jump to previous change

# applying changes
do               # diff obtain — pull change from other window into current
dp               # diff put — push change from current window to other

# maintenance
:diffupdate      # recalculate diff (after manual edits)
:diffget         # same as do (can specify buffer: :diffget 2)
:diffput         # same as dp (can specify buffer: :diffput 3)

# options
:set diffopt+=vertical       # always split vertically
:set diffopt+=iwhite         # ignore whitespace changes
:set diffopt+=context:3      # show 3 lines of context
```

## Command-Line Window

```bash
q:               # open command-line window (command history)
q/               # open command-line window (search history forward)
q?               # open command-line window (search history backward)
Ctrl-f           # switch from command-line to command-line window

# inside command-line window:
# - it's a normal vim buffer — navigate, edit, search
# - press Enter on a line to execute that command
# - press Ctrl-c to close without executing
# - you can yank commands, edit them, paste them
```

## Insert Mode Completion

```bash
Ctrl-n           # complete next keyword match (from multiple sources)
Ctrl-p           # complete previous keyword match
Ctrl-x Ctrl-f    # complete filename/path
Ctrl-x Ctrl-l    # complete whole line (from buffer)
Ctrl-x Ctrl-o    # omni completion (language-aware, needs filetype plugin)
Ctrl-x Ctrl-k    # complete from dictionary (:set dictionary=/usr/share/dict/words)
Ctrl-x Ctrl-]    # complete from tags file
Ctrl-x Ctrl-n    # complete keyword from current buffer only
Ctrl-x Ctrl-i    # complete keyword from included files
Ctrl-x Ctrl-d    # complete macro/definition names
Ctrl-x Ctrl-v    # complete vim command

# during completion popup:
Ctrl-n           # select next match
Ctrl-p           # select previous match
Ctrl-y           # accept current match
Ctrl-e           # cancel completion and return to typed text
```

## Insert Mode Tricks

```bash
Ctrl-o           # execute one normal mode command, then back to insert
Ctrl-w           # delete word before cursor
Ctrl-u           # delete to start of line
Ctrl-t           # indent current line
Ctrl-d           # unindent current line
Ctrl-a           # insert previously inserted text
Ctrl-r "         # paste unnamed register
Ctrl-r =         # evaluate expression and insert result
Ctrl-r Ctrl-p a  # paste register a literally (no auto-indent)
Ctrl-v u0041     # insert unicode character by hex (A)
Ctrl-v 065       # insert character by decimal (A)
Ctrl-k e'        # insert digraph (e with acute: e)
Ctrl-j           # insert newline (same as Enter)
```

## Spell Checking

```bash
:set spell                   # enable spell checking
:set nospell                 # disable spell checking
:set spelllang=en_us         # set spell language to US English
:set spelllang=en_us,es      # check English and Spanish
:set spellfile=~/.vim/spell/custom.utf-8.add  # custom word list

# navigating misspellings
]s               # go to next misspelled word
[s               # go to previous misspelled word
]S               # go to next bad word (skip rare/regional)
[S               # go to previous bad word

# correcting
z=               # show spelling suggestions for word under cursor
1z=              # accept first suggestion immediately
zg               # mark word as good (add to spellfile)
zw               # mark word as wrong (add as bad to spellfile)
zug              # undo zg — remove word from good list
zuw              # undo zw — remove word from bad list
zG               # mark as good for session only (not saved)
zW               # mark as wrong for session only
```

## Autocommands

```bash
# basic syntax: autocmd {event} {pattern} {command}

# strip trailing whitespace on save
autocmd BufWritePre * %s/\s\+$//e

# set filetype-specific options
autocmd FileType python setlocal tabstop=4 shiftwidth=4 expandtab
autocmd FileType go setlocal tabstop=4 shiftwidth=4 noexpandtab
autocmd FileType make setlocal noexpandtab
autocmd FileType markdown setlocal spell textwidth=80

# return to last edit position when opening files
autocmd BufReadPost * if line("'\"") > 1 && line("'\"") <= line("$") | exe "normal! g'\"" | endif

# auto-source vimrc on save
autocmd BufWritePost $MYVIMRC source $MYVIMRC

# highlight yanked text briefly
autocmd TextYankPost * silent! lua vim.highlight.on_yank({timeout=200})

# augroup — prevent duplicate autocommands on re-source
augroup MyGroup
  autocmd!                           # clear existing autocommands in group
  autocmd BufWritePre * %s/\s\+$//e
  autocmd FileType python setlocal ts=4 sw=4 et
augroup END

# common events
# BufRead / BufReadPost     — after reading a file
# BufWrite / BufWritePre    — before writing a file
# BufNewFile                — starting to edit a new file
# FileType                  — filetype detected
# InsertEnter / InsertLeave — entering/leaving insert mode
# VimEnter                  — after all startup is done
# VimLeave                  — before exiting vim
# BufEnter / BufLeave       — entering/leaving a buffer
# WinEnter / WinLeave       — entering/leaving a window
# CursorHold                — cursor idle for 'updatetime' ms
# TextYankPost              — after yanking text
```

## Netrw File Explorer

```bash
:Explore         # open netrw in current window (current file's dir)
:Sexplore        # open netrw in horizontal split
:Vexplore        # open netrw in vertical split
:Texplore        # open netrw in new tab
:Lexplore        # toggle netrw in left sidebar

# navigation inside netrw
Enter            # open file or directory
-                # go up one directory
u                # go to previous directory in history
U                # go to next directory in history
gh               # toggle hidden files (dotfiles)
i                # cycle view type (thin, long, wide, tree)

# file operations in netrw
%                # create new file (prompts for name)
d                # create new directory
D                # delete file/directory under cursor
R                # rename file/directory under cursor
mt               # mark target directory
mf               # mark file
mc               # copy marked files to target
mm               # move marked files to target

# netrw settings for .vimrc
let g:netrw_banner = 0       # hide banner at top
let g:netrw_liststyle = 3    # tree view by default
let g:netrw_browse_split = 4 # open files in previous window
let g:netrw_winsize = 25     # netrw window takes 25% width
let g:netrw_altv = 1         # vsplit to the right
```

## Text Formatting

```bash
gq{motion}       # format text (respects textwidth)
gqap             # format current paragraph
gqq              # format current line
gw{motion}       # format text (cursor stays in place)
gwap             # format current paragraph (cursor stays)
gqip             # format inner paragraph

# join lines
J                # join current line with next (adds space)
gJ               # join current line with next (no space)
3J               # join 3 lines
:j               # join lines in range (:1,5j)

# alignment commands
:center          # center current line
:center 80       # center to width 80
:right           # right-align current line
:right 80        # right-align to width 80
:left            # left-align (remove leading whitespace)
:left 4          # left-align with 4 spaces indent

# text width
:set textwidth=80    # auto-wrap at 80 columns
:set textwidth=0     # disable auto-wrap
:set formatoptions+=t  # auto-wrap text using textwidth
:set formatoptions+=c  # auto-wrap comments using textwidth
:set formatoptions+=q  # allow formatting with gq
:set formatoptions-=o  # don't auto-insert comment leader with o/O

# external formatting
:%!fmt -w 80         # format with Unix fmt command
:'<,'>!column -t     # align columns in selection
:%!prettier --stdin-filepath %.js  # format with prettier
```

## Sessions

```bash
:mksession           # save session to Session.vim in current dir
:mksession mysess.vim  # save session to specific file
:source Session.vim  # restore session
:source mysess.vim   # restore specific session

# from shell
vim -S Session.vim   # start vim and load session
vim -S mysess.vim    # start with specific session

# session stores: buffers, windows, tabs, folds, current dir, options
# does NOT store: registers, command history, marks (unless in viminfo)

# workflow: per-project sessions
# 1. cd to project root
# 2. open all files you need
# 3. :mksession
# 4. next time: vim -S Session.vim

# auto-save session on exit (add to .vimrc)
autocmd VimLeave * if !empty(v:this_session) | mksession! | endif
```

## Terminal

```bash
:terminal            # open terminal in current window
:term                # short form
:term ++rows=10      # open with 10 rows height
:term ++cols=80      # open with 80 columns width
:term ++curwin       # open in current window (not split)
:vert term           # open terminal in vertical split
:tab term            # open terminal in new tab
:term cmd            # run cmd in terminal
:term make           # run make in terminal

# inside terminal buffer
Ctrl-\ Ctrl-n        # switch from terminal mode to normal mode
                     # (now you can navigate, yank, scroll, etc.)
i  or  a             # switch back to terminal mode
Ctrl-w N             # same as Ctrl-\ Ctrl-n (enter normal mode)
Ctrl-w :             # enter vim command from terminal
Ctrl-w Ctrl-w        # switch to next window

# terminal-normal mode
# once in normal mode (Ctrl-\ Ctrl-n), you can:
# - use / to search terminal output
# - yank text with y
# - scroll with Ctrl-u, Ctrl-d
# - press i or a to go back to terminal input
```

## File Navigation

```bash
:e .             # open file browser (netrw)
:e **/*foo*      # fuzzy open file matching foo
gf               # open file under cursor
gF               # open file under cursor at line number (file:42)
Ctrl-^           # toggle between last two files
Ctrl-w gf        # open file under cursor in new tab
:ls              # list buffers
:buffers         # same as :ls
:b2              # switch to buffer 2
:b name          # switch to buffer matching name (partial match)
:bn              # next buffer
:bp              # previous buffer
:bd              # delete buffer (close file)
:bd 3            # delete buffer 3
:%bd             # delete all buffers
:e #             # edit alternate file

# argument list
:args **/*.go    # set argument list to all Go files
:args            # show current argument list
:next            # next file in arglist
:prev            # previous file in arglist
:first           # first file in arglist
:last            # last file in arglist
```

## .vimrc Essentials

```bash
" --- Core Settings ---
set nocompatible             " disable vi compatibility
filetype plugin indent on    " enable filetype detection and plugins
syntax enable                " enable syntax highlighting

" --- Display ---
set number                   " show line numbers
set relativenumber           " relative line numbers (hybrid with number)
set cursorline               " highlight current line
set showmatch                " highlight matching bracket
set showcmd                  " show partial command in bottom bar
set laststatus=2             " always show status line
set ruler                    " show cursor position in status line
set signcolumn=yes           " always show sign column (for git/linting)
set colorcolumn=80           " show column marker at 80
set scrolloff=8              " keep 8 lines above/below cursor
set sidescrolloff=8          " keep 8 columns left/right of cursor
set wrap                     " wrap long lines
set linebreak                " wrap at word boundaries
set wildmenu                 " visual autocomplete for command menu
set wildmode=longest:full,full  " tab complete to longest common, then cycle

" --- Indentation ---
set expandtab                " use spaces instead of tabs
set tabstop=4                " tab displays as 4 spaces
set softtabstop=4            " tab key inserts 4 spaces
set shiftwidth=4             " indent by 4 spaces
set autoindent               " copy indent from current line on new line
set smartindent              " smart autoindenting on new lines

" --- Search ---
set incsearch                " search as characters are entered
set hlsearch                 " highlight matches
set ignorecase               " case insensitive search...
set smartcase                " ...unless uppercase letters used

" --- Editing ---
set backspace=indent,eol,start  " backspace works on everything
set clipboard=unnamedplus    " use system clipboard for y/p
set hidden                   " allow unsaved buffers in background
set autoread                 " reload files changed outside vim
set confirm                  " ask instead of failing on unsaved quit
set undofile                 " persistent undo across sessions
set undodir=~/.vim/undodir   " undo file directory (mkdir this first!)
set noswapfile               " disable swap files
set nobackup                 " disable backup files

" --- Performance ---
set lazyredraw               " don't redraw during macros
set ttyfast                  " faster redrawing
set updatetime=250           " faster CursorHold (default 4000)
set timeoutlen=500           " mapping timeout in ms

" --- Leader Key ---
let mapleader = " "          " space as leader key
let maplocalleader = ","     " comma as local leader

" --- Common Mappings ---
" clear search highlight
nnoremap <leader><space> :noh<CR>

" quick save and quit
nnoremap <leader>w :w<CR>
nnoremap <leader>q :q<CR>

" better window navigation
nnoremap <C-h> <C-w>h
nnoremap <C-j> <C-w>j
nnoremap <C-k> <C-w>k
nnoremap <C-l> <C-w>l

" move lines up and down
vnoremap J :m '>+1<CR>gv=gv
vnoremap K :m '<-2<CR>gv=gv

" keep cursor centered when scrolling
nnoremap <C-d> <C-d>zz
nnoremap <C-u> <C-u>zz

" keep cursor centered when searching
nnoremap n nzzzv
nnoremap N Nzzzv

" don't overwrite register when pasting over selection
xnoremap <leader>p "_dP

" yank to system clipboard explicitly
nnoremap <leader>y "+y
vnoremap <leader>y "+y
nnoremap <leader>Y "+Y

" delete to black hole
nnoremap <leader>d "_d
vnoremap <leader>d "_d

" quick buffer switching
nnoremap <leader>bn :bn<CR>
nnoremap <leader>bp :bp<CR>
nnoremap <leader>bd :bd<CR>
nnoremap <leader>ls :ls<CR>

" open netrw
nnoremap <leader>e :Explore<CR>

" resize splits with arrows
nnoremap <C-Up> :resize +2<CR>
nnoremap <C-Down> :resize -2<CR>
nnoremap <C-Left> :vertical resize -2<CR>
nnoremap <C-Right> :vertical resize +2<CR>

" stay in visual mode after indenting
vnoremap < <gv
vnoremap > >gv

" quick access to .vimrc
nnoremap <leader>ve :e $MYVIMRC<CR>
nnoremap <leader>vs :source $MYVIMRC<CR>

" --- Status Line (no plugin) ---
set statusline=
set statusline+=%#PmenuSel#
set statusline+=\ %f              " filename
set statusline+=\ %m              " modified flag
set statusline+=%r                " read-only flag
set statusline+=%=                " right-align from here
set statusline+=\ %y              " filetype
set statusline+=\ %{&fileencoding?&fileencoding:&encoding}
set statusline+=\ [%{&fileformat}]
set statusline+=\ %l:%c           " line:column
set statusline+=\ %p%%            " percentage through file
set statusline+=\
```

## Tips

### General Workflow

- Use `.` to repeat the last change. Structure edits as repeatable operations.
- `ciw` (change inner word) is usually better than `diwi` -- fewer keystrokes, one undo step.
- `Ctrl-o` and `Ctrl-i` jump backward and forward through the jump list.
- `g;` and `g,` move through the change list -- jump to where you last edited.
- `ZZ` is equivalent to `:wq`, `ZQ` is equivalent to `:q!`.
- Use `Ctrl-a` and `Ctrl-x` to increment/decrement numbers under the cursor.
- `ga` shows the ASCII/Unicode value of the character under the cursor.
- `g8` shows the UTF-8 byte sequence of the character under the cursor.
- `gv` reselects the last visual selection -- extremely useful after an operation.

### Search and Replace

- `:set hlsearch` highlights matches; `:noh` clears until next search.
- In insert mode, `Ctrl-r "` pastes from the unnamed register. `Ctrl-r =` evaluates an expression.
- `:%!jq .` formats JSON in the current buffer (requires jq).
- Use `\zs` and `\ze` in search patterns to control what part of the match gets replaced.
- The `c` flag on substitute (`:s/a/b/gc`) lets you confirm each replacement.
- `:%s//replacement/g` reuses the last search pattern.

### Registers and Clipboard

- `"0` always holds the last yank -- useful when a delete overwrites the unnamed register.
- Numbered registers `"1` through `"9` hold your last 9 deletes, shifting on each delete.
- `"_d` deletes to the black hole register -- nothing is saved.
- Capital register names append: `"Ayy` appends a yank to register `a`.
- `Ctrl-r` in insert mode lets you paste any register without leaving insert mode.

### Efficiency

- `:set scrolloff=5` keeps 5 lines of context above/below the cursor.
- `:set relativenumber` makes jumping N lines trivial: just `5j` or `12k`.
- Record macros starting at a consistent position (e.g., `0` or `^`) for reliability.
- Use `q:` to edit and re-run previous commands in a full editor buffer.
- `:g/pattern/norm @a` runs a macro on every line matching a pattern.
- `:earlier 5m` and `:later 5m` let you time-travel through your undo history.
- `Ctrl-o` in insert mode lets you run one normal mode command without leaving insert.
- `gi` returns to insert mode at the exact position where you last left it.

### Dealing with External Tools

- `:r !cmd` reads command output into the buffer below the cursor.
- `:.!cmd` replaces the current line with command output.
- `:%!sort -u` sorts and deduplicates the entire file.
- `:'<,'>!column -t` aligns selected text into columns.
- `:term` opens a terminal inside vim -- `Ctrl-\ Ctrl-n` to scroll/yank from it.

### Vim Gotchas

- `Y` yanks the whole line by default (unlike `D` and `C`). Remap: `nnoremap Y y$`.
- Vim and Neovim handle clipboard differently. Neovim needs a clipboard provider (e.g., `xclip`).
- `:set paste` disables autoindent for pasting but also disables insert-mode mappings. Toggle it off after.
- `Ctrl-c` is not exactly the same as `Esc` -- it does not trigger `InsertLeave` autocommands.
- `u` in visual mode lowercases. If you meant undo, press `Esc` first, then `u`.
- The unnamed register `""` is overwritten by almost every delete/change, not just yanks.
- `:wq` always writes, even if nothing changed. Use `:x` or `ZZ` to write only if modified.
- Recursive macros (`@a` calling itself inside `qa...q`) stop at the first error -- use this to your advantage.

## See Also

- neovim, emacs, tmux, regex, git, bash

## References

- [Vim Documentation](https://vimhelp.org/) -- online version of Vim's built-in help
- [Vim Reference Manual](https://vimdoc.sourceforge.net/) -- complete reference manual
- [Vim Tips Wiki](https://vim.fandom.com/wiki/Vim_Tips_Wiki) -- community tips and tricks
- [Learn Vimscript the Hard Way](https://learnvimscriptthehardway.stevelosh.com/) -- Vimscript programming guide
- [Vim Quick Reference Card](https://vimhelp.org/quickref.txt.html) -- concise command reference
- [Vim GitHub](https://github.com/vim/vim) -- source code and issue tracker
- [Vim Awesome](https://vimawesome.com/) -- searchable plugin directory
- [man vim](https://man7.org/linux/man-pages/man1/vim.1.html) -- vim man page
- [Vimcasts](http://vimcasts.org/) -- screencasts and articles on Vim techniques
- [Practical Vim (book site)](https://pragprog.com/titles/dnvim2/practical-vim-second-edition/) -- Drew Neil's tip-based guide
