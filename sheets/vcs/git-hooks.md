# Git Hooks (Automation Scripts)

Git hooks are scripts that run automatically before or after git events like commit, push, and merge, enabling enforcement of code standards, running tests, and automating workflows without manual intervention.

## Client-Side Hooks

### Pre-Commit

```bash
# .git/hooks/pre-commit
#!/usr/bin/env bash
# Runs before commit is created — exit non-zero to abort

# Lint staged files only
STAGED=$(git diff --cached --name-only --diff-filter=ACM)

# Run gofmt on staged Go files
GOFMT_ERRORS=$(echo "$STAGED" | grep '\.go$' | xargs -r gofmt -l)
if [ -n "$GOFMT_ERRORS" ]; then
    echo "gofmt failed on:"
    echo "$GOFMT_ERRORS"
    exit 1
fi

# Prevent committing large files (>5MB)
for file in $STAGED; do
    size=$(wc -c < "$file" 2>/dev/null || echo 0)
    if [ "$size" -gt 5242880 ]; then
        echo "ERROR: $file is $(( size / 1048576 ))MB (limit 5MB)"
        exit 1
    fi
done

# Check for secrets (basic)
if git diff --cached --diff-filter=ACM -p | grep -qiE '(password|secret|api_key)\s*=\s*["\x27]'; then
    echo "ERROR: possible secret detected in staged changes"
    exit 1
fi
```

### Commit-Msg

```bash
# .git/hooks/commit-msg
#!/usr/bin/env bash
# Validate commit message format — $1 is the temp file with the message

MSG=$(cat "$1")

# Enforce conventional commits
if ! echo "$MSG" | grep -qE '^(feat|fix|docs|style|refactor|test|chore|perf|ci|build|revert)(\(.+\))?: .{10,}'; then
    echo "ERROR: commit message must match: type(scope): description (min 10 chars)"
    echo "Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert"
    exit 1
fi

# Enforce max line length
if echo "$MSG" | head -1 | grep -qE '.{73,}'; then
    echo "ERROR: first line must be 72 characters or less"
    exit 1
fi
```

### Prepare-Commit-Msg

```bash
# .git/hooks/prepare-commit-msg
#!/usr/bin/env bash
# Prepopulate commit message — $1 = msg file, $2 = source, $3 = SHA

BRANCH=$(git symbolic-ref --short HEAD 2>/dev/null)

# Prepend ticket number from branch name (e.g., feature/JIRA-123-description)
TICKET=$(echo "$BRANCH" | grep -oE '[A-Z]+-[0-9]+')
if [ -n "$TICKET" ] && ! grep -q "$TICKET" "$1"; then
    sed -i.bak "1s/^/[$TICKET] /" "$1"
fi
```

### Pre-Push

```bash
# .git/hooks/pre-push
#!/usr/bin/env bash
# Runs before push — receives remote name and URL as args
# Stdin: lines of "<local ref> <local sha> <remote ref> <remote sha>"

REMOTE="$1"

# Prevent force-push to main/master
while read local_ref local_sha remote_ref remote_sha; do
    if echo "$remote_ref" | grep -qE 'refs/heads/(main|master)$'; then
        echo "ERROR: direct push to main/master is not allowed"
        echo "Use a pull request instead"
        exit 1
    fi
done

# Run tests before push
echo "Running tests before push..."
go test ./... -count=1 -race -timeout 120s
if [ $? -ne 0 ]; then
    echo "ERROR: tests failed — push aborted"
    exit 1
fi
```

### Post-Merge

```bash
# .git/hooks/post-merge
#!/usr/bin/env bash
# Runs after a successful merge — $1 is squash flag (0=merge, 1=squash)

CHANGED=$(git diff-tree -r --name-only --no-commit-id ORIG_HEAD HEAD)

# Auto-install if dependencies changed
if echo "$CHANGED" | grep -q 'go.mod'; then
    echo "go.mod changed — running go mod tidy"
    go mod tidy
fi

if echo "$CHANGED" | grep -q 'package.json'; then
    echo "package.json changed — running npm install"
    npm install
fi

# Rebuild if Makefile changed
if echo "$CHANGED" | grep -q 'Makefile'; then
    echo "Makefile changed — rebuilding"
    make build
fi
```

### Post-Checkout

```bash
# .git/hooks/post-checkout
#!/usr/bin/env bash
# Runs after checkout/switch — $1=prev HEAD, $2=new HEAD, $3=branch flag (1=branch, 0=file)

PREV_HEAD="$1"
NEW_HEAD="$2"
BRANCH_CHECKOUT="$3"

if [ "$BRANCH_CHECKOUT" = "1" ]; then
    # Only on branch switches, not file checkouts
    CHANGED=$(git diff --name-only "$PREV_HEAD" "$NEW_HEAD")

    if echo "$CHANGED" | grep -q 'go.mod'; then
        echo "go.mod differs — running go mod download"
        go mod download
    fi
fi
```

## Server-Side Hooks

### Pre-Receive

```bash
# hooks/pre-receive (bare repo)
#!/usr/bin/env bash
# Runs once per push — stdin: "<old-sha> <new-sha> <ref>" per updated ref

while read old_sha new_sha ref; do
    # Block force-push (non-fast-forward) to protected branches
    if echo "$ref" | grep -qE 'refs/heads/(main|release/.*)'; then
        if [ "$old_sha" != "0000000000000000000000000000000000000000" ]; then
            MERGE_BASE=$(git merge-base "$old_sha" "$new_sha" 2>/dev/null)
            if [ "$MERGE_BASE" != "$old_sha" ]; then
                echo "ERROR: non-fast-forward push to $ref is not allowed"
                exit 1
            fi
        fi
    fi
done
```

### Update

```bash
# hooks/update (bare repo)
#!/usr/bin/env bash
# Runs once per ref — $1=ref, $2=old-sha, $3=new-sha

REF="$1"
OLD="$2"
NEW="$3"

# Restrict tag deletion
if echo "$REF" | grep -q 'refs/tags/' && [ "$NEW" = "0000000000000000000000000000000000000000" ]; then
    echo "ERROR: tag deletion is not allowed"
    exit 1
fi
```

### Post-Receive

```bash
# hooks/post-receive (bare repo)
#!/usr/bin/env bash
# Runs after all refs updated — trigger deployments, notifications

while read old_sha new_sha ref; do
    if [ "$ref" = "refs/heads/main" ]; then
        echo "Deploying main to production..."
        GIT_WORK_TREE=/var/www/app git checkout -f main
    fi
done
```

## Hook Management

### Installation and Permissions

```bash
# Hooks live in .git/hooks/ (not tracked by git)
ls .git/hooks/                         # see sample hooks
chmod +x .git/hooks/pre-commit         # hooks must be executable
cp scripts/hooks/* .git/hooks/         # manual install from project

# Shared hooks directory (git 2.9+)
git config core.hooksPath .githooks    # use tracked .githooks/ dir
git config --global core.hooksPath ~/.git-hooks  # global hooks
```

### Husky (Node.js Projects)

```bash
# Install husky
npm install --save-dev husky
npx husky init                         # creates .husky/ directory

# Add hooks
echo 'npm test' > .husky/pre-commit
echo 'npx commitlint --edit $1' > .husky/commit-msg

# .husky/pre-commit is automatically executable and git-tracked
```

### Pre-Commit Framework (Python)

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
        args: ['--maxkb=500']
      - id: detect-private-key

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.59.0
    hooks:
      - id: golangci-lint

  - repo: https://github.com/shellcheck-py/shellcheck-py
    rev: v0.10.0
    hooks:
      - id: shellcheck
```

```bash
pip install pre-commit
pre-commit install                     # install git hooks
pre-commit install --hook-type commit-msg
pre-commit run --all-files             # run on all files (not just staged)
pre-commit autoupdate                  # update hook versions
pre-commit uninstall                   # remove hooks
```

### Lefthook (Language-Agnostic)

```yaml
# lefthook.yml
pre-commit:
  parallel: true
  commands:
    lint:
      glob: "*.go"
      run: golangci-lint run {staged_files}
    fmt:
      glob: "*.go"
      run: gofmt -l {staged_files}

pre-push:
  commands:
    test:
      run: go test ./... -count=1 -race
```

```bash
lefthook install                       # install hooks
lefthook run pre-commit                # run manually
```

## Bypassing Hooks

```bash
git commit --no-verify                 # skip pre-commit and commit-msg
git commit -n                          # same as --no-verify
git push --no-verify                   # skip pre-push
SKIP=golangci-lint git commit          # pre-commit framework: skip specific hooks
```

## Tips

- Always use `#!/usr/bin/env bash` (not `#!/bin/bash`) for portability across systems.
- Hook scripts in `.git/hooks/` are not tracked by git -- use `core.hooksPath` to point to a tracked directory instead.
- `pre-commit` runs only on staged files by default; use `git diff --cached` to get the staged file list.
- `commit-msg` receives the message file path as `$1` -- read with `cat "$1"`, write with `sed -i`.
- `pre-push` receives info on stdin, not as arguments -- use `while read` to parse it.
- Server-side `pre-receive` is the last line of defense -- it cannot be bypassed with `--no-verify`.
- The `pre-commit` framework and husky both solve the "hooks not in version control" problem differently -- pick one per project.
- Test hooks by running them directly: `.git/hooks/pre-commit` should work standalone.
- Keep hooks fast (under 5 seconds) or developers will start using `--no-verify` habitually.
- Use `exit 0` at the end of informational hooks (post-merge, post-checkout) -- a non-zero exit does not abort but prints a warning.
- Chain multiple checks in one hook with `&&` or track a failure flag -- exit on the first failure or collect all errors.
- Hooks run with the repo root as the working directory, regardless of where the git command was invoked.

## See Also

git, git-worktree, github-actions, make, shell-scripting, bash

## References

- [Git Hooks Documentation](https://git-scm.com/docs/githooks) -- official reference for all hook types
- [Pro Git: Customizing Git - Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks) -- chapter on hooks with examples
- [man githooks](https://man7.org/linux/man-pages/man5/githooks.5.html) -- man page with hook specifications
- [pre-commit framework](https://pre-commit.com/) -- multi-language hook management tool
- [Husky](https://typicode.github.io/husky/) -- git hooks for Node.js projects
- [Lefthook](https://github.com/evilmartians/lefthook) -- fast polyglot git hooks manager
- [Conventional Commits](https://www.conventionalcommits.org/) -- commit message specification
- [commitlint](https://commitlint.js.org/) -- lint commit messages against conventional commits
