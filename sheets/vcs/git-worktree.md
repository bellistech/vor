# Git Worktree (Multiple Working Trees)

Git worktrees let you check out multiple branches simultaneously in separate directories, all sharing the same repository object database, avoiding the overhead of multiple clones.

## Basic Operations

### Add a Worktree

```bash
# Create worktree with existing branch
git worktree add ../hotfix hotfix/urgent-fix

# Create worktree with new branch
git worktree add -b feature/auth ../auth-work

# Create worktree at detached HEAD
git worktree add --detach ../bisect-work HEAD~20

# Create worktree from remote branch
git worktree add ../review origin/feature/review

# Create worktree with specific commit
git worktree add ../investigate abc1234
```

### List Worktrees

```bash
git worktree list                      # list all worktrees
# /home/user/project          abc1234 [main]
# /home/user/hotfix           def5678 [hotfix/urgent-fix]
# /home/user/auth-work        ghi9012 [feature/auth]

git worktree list --porcelain          # machine-readable output
```

### Remove a Worktree

```bash
git worktree remove ../hotfix          # remove worktree (must be clean)
git worktree remove --force ../hotfix  # force remove (even with changes)

# Manual cleanup (if directory already deleted)
git worktree prune                     # remove stale worktree entries
git worktree prune --dry-run           # show what would be pruned
```

### Move a Worktree

```bash
git worktree move ../hotfix ../hotfix-new-location
```

### Lock and Unlock

```bash
# Prevent pruning (useful for network/removable drives)
git worktree lock ../hotfix
git worktree lock --reason "on USB drive" ../hotfix

git worktree unlock ../hotfix
```

## Worktree Internals

### How It Works

```bash
# Main repo stores worktree metadata
ls .git/worktrees/
# hotfix/
#   HEAD          → ref: refs/heads/hotfix/urgent-fix
#   gitdir        → /home/user/hotfix/.git
#   commondir     → ../..

# Linked worktree has a file (not directory) as .git
cat ../hotfix/.git
# gitdir: /home/user/project/.git/worktrees/hotfix
```

### Shared vs Per-Worktree

```bash
# Shared across all worktrees (in main .git):
# - objects/       → all git objects (blobs, trees, commits)
# - refs/          → branches, tags, remote refs
# - config         → repository configuration
# - hooks/         → hook scripts
# - packed-refs    → packed references

# Per-worktree (in .git/worktrees/<name>/):
# - HEAD           → current branch/commit
# - index          → staging area
# - MERGE_HEAD     → merge state
# - REBASE_HEAD    → rebase state
# - logs/HEAD      → reflog for this worktree's HEAD
```

## Workflow Patterns

### Hotfix While Working on Feature

```bash
# You're in the middle of feature work on main repo
# Urgent bug comes in — don't stash, create a worktree

git worktree add -b hotfix/CVE-2024 ../hotfix main
cd ../hotfix

# Fix the bug
vim src/auth.go
go test ./...
git commit -am "fix: patch CVE-2024 auth bypass"
git push origin hotfix/CVE-2024

# Done — go back and clean up
cd ../project
git worktree remove ../hotfix
```

### Code Review in Separate Directory

```bash
# Review a PR without disrupting your current work
git fetch origin
git worktree add ../review origin/pull/123/head

cd ../review
# Read code, run tests, check behavior
go test ./...
make lint

# Clean up
cd ../project
git worktree remove ../review
```

### Parallel Testing Across Branches

```bash
# Test multiple branches simultaneously
git worktree add ../test-v1 release/v1.0
git worktree add ../test-v2 release/v2.0
git worktree add ../test-main main

# Run tests in parallel (separate terminals or background jobs)
(cd ../test-v1 && go test ./... > /tmp/v1-results.txt 2>&1) &
(cd ../test-v2 && go test ./... > /tmp/v2-results.txt 2>&1) &
(cd ../test-main && go test ./... > /tmp/main-results.txt 2>&1) &
wait

# Compare results
diff /tmp/v1-results.txt /tmp/v2-results.txt
```

### Long-Running Build + Development

```bash
# Worktree for CI-like build that takes 20 minutes
git worktree add ../build-test main
(cd ../build-test && make docker-build-all 2>&1 | tee build.log) &

# Continue developing in main worktree while build runs
vim src/new-feature.go
```

### Bisect Without Losing Context

```bash
# Create a dedicated worktree for bisecting
git worktree add --detach ../bisect HEAD

cd ../bisect
git bisect start
git bisect bad HEAD
git bisect good v1.0.0
git bisect run ./test.sh

# Note the bad commit, clean up
cd ../project
git worktree remove ../bisect
```

## Worktree with Bare Repos

```bash
# Clone as bare repo (no working tree by default)
git clone --bare git@github.com:user/repo.git repo.git
cd repo.git

# Add worktrees for each branch
git worktree add ../main main
git worktree add ../develop develop
git worktree add -b feature/new ../feature

# This pattern keeps the bare repo as the "hub"
# and worktrees as the working directories
```

## Branch Restrictions

```bash
# A branch can only be checked out in ONE worktree at a time
git worktree add ../second main
# fatal: 'main' is already checked out at '/home/user/project'

# Workaround: use detached HEAD
git worktree add --detach ../second HEAD

# Or check which worktree has the branch
git worktree list | grep main
```

## Configuration

```bash
# Auto-prune stale worktrees on fetch
git config fetch.prune true

# Set default worktree location pattern
# (no built-in config — use shell aliases)

# Alias for quick worktree creation
git config --global alias.wt 'worktree'
git config --global alias.wta 'worktree add'
git config --global alias.wtl 'worktree list'
git config --global alias.wtr 'worktree remove'
```

## Shell Integration

```bash
# Function: create worktree and cd into it
gwt() {
    local branch="$1"
    local dir="${2:-../$branch}"
    git worktree add "$dir" "$branch" && cd "$dir"
}

# Function: remove current worktree and cd back
gwtr() {
    local wt_dir
    wt_dir=$(pwd)
    cd "$(git worktree list | head -1 | awk '{print $1}')"
    git worktree remove "$wt_dir"
}

# List worktrees with branch status
alias gwtls='git worktree list && echo "---" && git branch -vv'
```

## Tips

- Each worktree has its own index (staging area) and HEAD, so you can stage and commit independently in each one.
- A branch cannot be checked out in two worktrees simultaneously -- git prevents this to avoid confusion.
- Worktrees share the object database, so `git fetch` in any worktree makes objects available to all of them.
- Use `git worktree prune` after manually deleting a worktree directory to clean up stale metadata in `.git/worktrees/`.
- Hooks are shared across worktrees (they live in the main `.git/hooks/`), so a pre-commit hook applies everywhere.
- Worktrees are much cheaper than full clones -- they share objects and only add an index file and a HEAD pointer.
- The bare repo + worktree pattern is popular for managing multiple long-lived branches without a "main" working directory.
- Lock worktrees on removable media with `git worktree lock` to prevent `prune` from deleting their references.
- Worktrees work with submodules, but each worktree gets its own submodule checkout -- run `git submodule update` per worktree.
- `git worktree move` lets you relocate a worktree without removing and re-adding it.
- Worktrees are ideal for bisecting -- create a detached worktree, bisect there, and your main work stays untouched.
- Keep worktree directories as siblings of the main repo (e.g., `../hotfix`) for easy navigation.

## See Also

git, git-hooks, github-actions, bash, shell-scripting

## References

- [git-worktree Documentation](https://git-scm.com/docs/git-worktree) -- official reference
- [Pro Git: Git Tools](https://git-scm.com/book/en/v2/Git-Tools-Revision-Selection) -- advanced git features
- [man git-worktree](https://man7.org/linux/man-pages/man1/git-worktree.1.html) -- man page with full options
- [Git Worktree Tutorial (Atlassian)](https://www.atlassian.com/git/tutorials/git-worktree) -- practical guide
- [Git Internals - Plumbing and Porcelain](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain) -- how worktrees share the object database
