# ShellCheck (Shell Script Static Analyzer)

Static analysis tool for shell scripts that finds bugs, pitfalls, and portability issues before they bite.

## Installation

### Package managers

```bash
# macOS
brew install shellcheck

# Ubuntu/Debian
apt-get install shellcheck

# Fedora
dnf install ShellCheck

# Arch
pacman -S shellcheck

# From source (Haskell)
cabal update && cabal install ShellCheck

# Docker
docker run --rm -v "$PWD:/mnt" koalaman/shellcheck:stable script.sh
```

## Basic Usage

### Running ShellCheck

```bash
shellcheck script.sh                    # check single file
shellcheck -s bash script.sh            # specify shell dialect
shellcheck -s sh script.sh              # POSIX sh
shellcheck -s dash script.sh            # Dash
shellcheck scripts/*.sh                 # check multiple files
shellcheck -x script.sh                 # follow source/. includes
shellcheck -f json script.sh            # JSON output
shellcheck -f gcc script.sh             # GCC-style output
shellcheck -f diff script.sh            # diff/patch output
shellcheck -f checkstyle script.sh      # Checkstyle XML (CI)
shellcheck -S error script.sh           # only errors (no warnings)
shellcheck -S warning script.sh         # errors + warnings
shellcheck -S info script.sh            # errors + warnings + info
shellcheck -S style script.sh           # all (default)
```

## Common SC Codes

### SC2086 -- Double-quote to prevent word splitting

```bash
# Bad -- word splitting on spaces in filename
rm $file
cp $src $dst

# Good
rm "$file"
cp "$src" "$dst"

# Also applies to command substitution
path=$(find . -name "config")
cat "$path"                              # not: cat $path
```

### SC2046 -- Quote to prevent globbing and word splitting

```bash
# Bad -- output undergoes globbing
files=$(ls *.txt)
echo $files

# Good
files=$(ls *.txt)
echo "$files"

# Better -- use arrays for file lists
files=(*.txt)
echo "${files[@]}"
```

### SC2006 -- Use $(...) instead of backticks

```bash
# Bad -- backticks are harder to nest and read
result=`command`
nested=`echo \`date\``

# Good
result=$(command)
nested=$(echo "$(date)")
```

### SC2039/SC3054 -- Bash-only features in sh scripts

```bash
# Bad (#!/bin/sh) -- arrays are bash-only
arr=(one two three)
[[ $x == y ]]

# Good (#!/bin/sh) -- POSIX alternatives
set -- one two three
[ "$x" = y ]

# Or change shebang to #!/bin/bash
```

### SC2034 -- Variable appears unused

```bash
# Triggers when a variable is assigned but never read
unused_var="hello"

# Fix: use it or remove it
# shellcheck disable=SC2034
EXPORTED_VAR="hello"    # intentionally unused (sourced by other scripts)
```

### SC2155 -- Declare and assign separately

```bash
# Bad -- masks return code of command
local output=$(command)

# Good -- preserves exit status
local output
output=$(command)
```

### SC2164 -- Use cd ... || exit

```bash
# Bad -- script continues if cd fails
cd /some/path
rm -rf *                # deletes wrong files if cd failed!

# Good
cd /some/path || exit 1
rm -rf ./*

# Also good
cd /some/path || { echo "cd failed"; exit 1; }
```

### SC2162 -- read without -r

```bash
# Bad -- backslashes are interpreted
read input

# Good -- raw input, no backslash processing
read -r input
```

### SC2129 -- Group redirections

```bash
# Bad -- repeated redirections
echo "line 1" >> file
echo "line 2" >> file
echo "line 3" >> file

# Good -- single redirection block
{
  echo "line 1"
  echo "line 2"
  echo "line 3"
} >> file
```

## Directives

### Inline disable

```bash
# Disable for next line
# shellcheck disable=SC2086
echo $unquoted_var

# Disable for entire file (place at top)
# shellcheck disable=SC2086,SC2046

# Disable for a block (function scope)
function legacy_code() {
    # shellcheck disable=SC2086
    command $args
}
```

### Source directives

```bash
# Tell shellcheck where sourced files are
# shellcheck source=./lib/utils.sh
source "$DIR/utils.sh"

# Mark as externally sourced (skip checking)
# shellcheck source=/dev/null
source "$DYNAMIC_PATH"

# Set source path for all includes
# shellcheck source-path=SCRIPTDIR
source lib/helpers.sh
```

### Shell directive

```bash
# Override detected shell
# shellcheck shell=bash
```

## Severity Levels

### Level breakdown

```bash
# error   -- likely bugs, syntax errors, code that won't work
# warning -- likely problems, common pitfalls
# info    -- style issues, minor improvements
# style   -- purely cosmetic suggestions

# Filter by severity
shellcheck -S error script.sh           # only errors
shellcheck -S warning script.sh         # errors + warnings
shellcheck -e SC2086 script.sh          # exclude specific code
shellcheck -i SC2086,SC2046 script.sh   # include only specific codes
```

## Editor Integration

### VS Code

```bash
# Install "ShellCheck" extension (timonwong.shellcheck)
# Automatically highlights issues in .sh files
# Settings:
# "shellcheck.executablePath": "/usr/local/bin/shellcheck"
# "shellcheck.customArgs": ["-x"]
```

### Vim/Neovim (ALE)

```vim
" .vimrc
let g:ale_linters = {'sh': ['shellcheck']}
let g:ale_sh_shellcheck_options = '-x'
```

### Emacs (Flycheck)

```elisp
;; flycheck-mode auto-detects shellcheck
(setq flycheck-shellcheck-follow-sources t)
```

## CI Integration

### GitHub Actions

```yaml
jobs:
  shellcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ludeeus/action-shellcheck@master
        with:
          severity: warning
          scandir: './scripts'
          additional_files: 'entrypoint.sh'
```

### GitLab CI

```yaml
shellcheck:
  image: koalaman/shellcheck-alpine:stable
  script:
    - find . -name '*.sh' -exec shellcheck -S warning {} +
```

### Pre-commit hook

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/koalaman/shellcheck-precommit
    rev: v0.10.0
    hooks:
      - id: shellcheck
        args: ["-S", "warning", "-x"]
```

### Makefile target

```makefile
.PHONY: shellcheck
shellcheck:
	find . -name '*.sh' -not -path './vendor/*' | xargs shellcheck -S warning -x
```

## Tips

- Always quote `"$variables"` -- SC2086 catches the most real-world bugs from word splitting
- Use `shellcheck -x` to follow `source`/`.` includes and check sourced files too
- Put `# shellcheck disable=` on the line before the violation, not on the same line
- Use `set -euo pipefail` at the top of bash scripts for strict error handling (ShellCheck validates this pattern)
- Run `shellcheck -f diff` to generate auto-fixable patches for simple issues
- Use severity `-S warning` in CI to catch bugs without blocking on style nits
- Check your CI scripts and Dockerfiles' `RUN` commands -- they are shell too
- Use `# shellcheck source-path=SCRIPTDIR` to resolve relative source paths correctly
- Pin the ShellCheck version in CI to avoid surprise new warnings breaking builds
- Use `shellcheck -f json` for programmatic processing and custom reporting
- Address SC2164 (`cd || exit`) religiously -- unguarded `cd` causes catastrophic `rm` bugs
- Combine with `shfmt` for formatting -- ShellCheck handles correctness, shfmt handles style

## See Also

- bash
- shfmt
- hadolint
- pre-commit

## References

- [ShellCheck Official Wiki](https://www.shellcheck.net/)
- [ShellCheck GitHub Repository](https://github.com/koalaman/shellcheck)
- [ShellCheck Error Code Wiki](https://github.com/koalaman/shellcheck/wiki/Checks)
- [ShellCheck VS Code Extension](https://marketplace.visualstudio.com/items?itemName=timonwong.shellcheck)
- [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html)
