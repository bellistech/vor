# fish (friendly interactive shell)

A smart and user-friendly command-line shell that provides autosuggestions, syntax highlighting, and sane defaults out of the box without requiring complex configuration, while offering a clean scripting language that breaks from POSIX conventions.

## Variables

### Setting Variables

```bash
# Set a local variable (current scope only)
set myvar "hello world"

# Set a global variable (current session)
set -g myvar "hello"

# Set a universal variable (persists across all sessions and restarts)
set -U EDITOR nvim
set -U fish_greeting ""      # disable greeting

# Set an exported variable (available to child processes)
set -x MY_ENV_VAR "value"

# Set exported universal variable (persists + exported)
set -Ux GOPATH $HOME/go

# Append to a list variable
set -a PATH $HOME/.local/bin

# Prepend to a list variable
set -p PATH $HOME/.cargo/bin

# Erase a variable
set -e myvar

# List all variables
set

# Show specific variable
set -S PATH
```

### PATH Management

```bash
# fish PATH is a list, not colon-separated
# Add to PATH universally (persists)
fish_add_path $HOME/.local/bin
fish_add_path $HOME/.cargo/bin
fish_add_path /usr/local/go/bin

# Prepend (default) vs append
fish_add_path --prepend $HOME/bin
fish_add_path --append /opt/bin

# Remove from PATH
set PATH (string match -v '/path/to/remove' $PATH)

# View PATH as list
printf '%s\n' $PATH
```

## Functions

### Defining Functions

```bash
# Define a function
function greet
    echo "Hello, $argv[1]!"
end

# Function with description
function ll --description "Long listing with hidden files"
    ls -lah $argv
end

# Function wrapping a command
function rm --wraps rm --description "Safe rm"
    command rm -i $argv
end

# Save function permanently (to ~/.config/fish/functions/)
funcsave greet

# Edit a function
funced greet
# Then save: funcsave greet

# List all functions
functions

# Show function definition
functions greet

# Delete a function
functions -e greet

# Autoloaded functions: place in ~/.config/fish/functions/name.fish
# File must match function name
```

### Event Handlers

```bash
# Run function on variable change
function on_pwd_change --on-variable PWD
    echo "Changed to $PWD"
end

# Run on signal
function on_exit --on-event fish_exit
    echo "Goodbye!"
end

# Run on process exit
function on_job_done --on-process-exit %self
    echo "Job finished"
end
```

## Abbreviations

### Managing Abbreviations

```bash
# Add an abbreviation (expands on Space/Enter)
abbr -a g git
abbr -a gst "git status"
abbr -a gco "git checkout"
abbr -a gp "git push"
abbr -a gcm "git commit -m"
abbr -a dc "docker compose"
abbr -a k kubectl

# Position-dependent abbreviation (only at command position)
abbr -a --position command l "ls -la"

# Regex abbreviation (fish 3.6+)
abbr -a --regex "^!!\$" --function last_history_item

# Remove abbreviation
abbr -e g

# List all abbreviations
abbr --list
abbr --show     # with expansions

# Abbreviations are stored universally (persist across sessions)
# No need for funcsave
```

## Completions

### Writing Completions

```bash
# Basic completion for a command
complete -c mycommand -s h -l help -d "Show help"
complete -c mycommand -s v -l verbose -d "Verbose output"
complete -c mycommand -l output -r -F -d "Output file"

# Subcommand completions
complete -c mycommand -n "__fish_use_subcommand" -a "start" -d "Start service"
complete -c mycommand -n "__fish_use_subcommand" -a "stop" -d "Stop service"
complete -c mycommand -n "__fish_seen_subcommand_from start" -s d -l daemon -d "Run as daemon"

# Dynamic completions (from command output)
complete -c kubectl -n "__fish_use_subcommand" -a "(kubectl get namespaces -o name | string replace 'namespace/' '')"

# File type completions
complete -c convert -s o -l output -r -F   # -F = accept files
complete -c convert -s o -l output -r -d "Output" -a "(__fish_complete_suffix .png .jpg)"

# Completion with conditions
complete -c git -n "__fish_git_using_command checkout" -a "(git branch -a | string trim)"

# Save completions to ~/.config/fish/completions/mycommand.fish
```

## String Manipulation

### String Builtins

```bash
# String match (glob or regex)
string match "*.txt" file.txt        # glob match
string match -r "^[0-9]+" "42abc"   # regex match, prints "42"
string match -rg '(\d+)' "port:8080" # capture group, prints "8080"

# String replace
string replace "old" "new" "old string"   # first match
string replace -a "o" "0" "foo"           # all matches: "f00"
string replace -r '(\w+)@(\w+)' '$2/$1' "user@host"  # regex

# String split
string split "," "a,b,c"            # outputs a, b, c on separate lines
string split -m1 "=" "key=val=ue"   # max 1 split: "key", "val=ue"
string split0                        # split on null bytes

# String join
string join ", " a b c               # "a, b, c"
string join \n line1 line2            # join with newlines

# String trim
string trim "  hello  "              # "hello"
string trim -l "  hello  "           # "hello  " (left only)
string trim -c "/" "/path/"          # "path" (custom chars)

# String length / sub / upper / lower
string length "hello"               # 5
string sub -s 2 -l 3 "hello"        # "ell" (start at 2, length 3)
string upper "hello"                 # "HELLO"
string lower "HELLO"                 # "hello"

# String repeat / pad
string repeat -n 3 "ab"             # "ababab"
string pad -w 10 "hello"            # "     hello"
string pad -w 10 --char=0 "42"      # "0000000042"

# String collect (join stdin into single string)
printf '%s\n' a b c | string collect  # "a\nb\nc" as single arg
```

## Control Flow

### Conditionals and Loops

```bash
# If/else
if test -f myfile.txt
    echo "File exists"
else if test -d mydir
    echo "Directory exists"
else
    echo "Not found"
end

# Switch/case
switch $argv[1]
    case start
        echo "Starting"
    case stop
        echo "Stopping"
    case '*'
        echo "Unknown command"
end

# For loop
for f in *.txt
    echo "Processing $f"
end

# While loop
while read -l line
    echo "Line: $line"
end < input.txt

# Command substitution
set files (ls *.txt)

# Status codes
if command -q git
    echo "git is installed"
end

# Status variable
false
echo $status   # 1

# Logical operators
test -f a.txt; and echo "exists"
test -f a.txt; or echo "missing"
command1; and command2; or command3
```

## Math

### Math Builtin

```bash
# Basic arithmetic
math "2 + 3"         # 5
math "10 / 3"        # 3.333333
math "2 ^ 10"        # 1024
math "sqrt(144)"     # 12
math "ceil(3.2)"     # 4
math "floor(3.8)"    # 3
math "round(3.5)"    # 4
math "abs(-42)"      # 42

# Integer division
math "10 // 3"       # 3

# Modulo
math "10 % 3"        # 1

# Bitwise operations
math "0xFF & 0x0F"   # 15
math "1 << 8"        # 256

# Use in variable assignment
set result (math "2 * $width + 1")

# Trigonometric functions
math "sin(3.14159)"  # ~0
math "cos(0)"        # 1
math "pi"            # 3.141593
math "e"             # 2.718282

# Scale/precision
math -s 2 "10 / 3"  # 3.33 (2 decimal places)
```

## Configuration

### Config Files

```bash
# Main config: ~/.config/fish/config.fish
# Runs on every shell start (interactive + login)

# conf.d directory: ~/.config/fish/conf.d/*.fish
# Each file is sourced alphabetically before config.fish
# Great for modular configuration

# Functions: ~/.config/fish/functions/funcname.fish
# Autoloaded on first use

# Completions: ~/.config/fish/completions/command.fish
# Autoloaded when completing

# Web config UI
fish_config                # opens browser-based config
fish_config theme          # theme picker
fish_config prompt         # prompt picker
```

## Fisher Plugin Manager

### Plugin Management

```bash
# Install fisher
curl -sL https://raw.githubusercontent.com/jorgebucaran/fisher/main/functions/fisher.fish | source && fisher install jorgebucaran/fisher

# Install a plugin
fisher install PatrickF1/fzf.fish
fisher install jethrokuan/z
fisher install IlanCosman/tide@v6
fisher install meaningful-ooo/sponge

# List installed plugins
fisher list

# Update all plugins
fisher update

# Update specific plugin
fisher update PatrickF1/fzf.fish

# Remove a plugin
fisher remove jethrokuan/z

# Popular plugins:
#   PatrickF1/fzf.fish     — fzf integration (Ctrl-R, Ctrl-F, Alt-C)
#   IlanCosman/tide         — async prompt (like powerlevel10k)
#   jethrokuan/z            — z directory jumping
#   meaningful-ooo/sponge   — clean command history (remove typos)
#   jorgebucaran/autopair.fish — auto-close brackets/quotes
```

## Tips

- Universal variables (`set -U`) persist across all sessions and reboots -- use them for `EDITOR`, `PATH` additions, and preferences.
- Use `fish_add_path` instead of manually modifying `$PATH` -- it handles deduplication and persistence.
- Abbreviations (`abbr`) expand inline so you can edit the expanded command before running, unlike aliases.
- The `string` command replaces `sed`, `awk`, `tr`, `cut` for most text processing -- learn it well.
- Place one function per file in `~/.config/fish/functions/` named `funcname.fish` for autoloading.
- Use `conf.d/` directory for modular config instead of one giant `config.fish`.
- `command -q tool` checks if a command exists without running it -- use it for conditional setup.
- The `math` builtin handles floats, trig functions, and bitwise ops -- no need for `bc` or `expr`.
- Press `Alt+e` or `Alt+v` to open the current command line in your `$EDITOR` for complex editing.
- Fish syntax highlighting shows invalid commands in red and valid ones in blue/green as you type.
- Use `status is-interactive` and `status is-login` guards in config.fish to avoid running interactive-only setup in scripts.

## See Also

- bash, zsh, tmux, fzf, zoxide, nushell

## References

- [fish shell Documentation](https://fishshell.com/docs/current/)
- [fish Tutorial](https://fishshell.com/docs/current/tutorial.html)
- [fisher Plugin Manager](https://github.com/jorgebucaran/fisher)
- [awesome-fish (Curated Plugins)](https://github.com/jorgebucaran/awesome-fish)
- [fish FAQ](https://fishshell.com/docs/current/faq.html)
- [fish for bash users](https://fishshell.com/docs/current/fish_for_bash_users.html)
