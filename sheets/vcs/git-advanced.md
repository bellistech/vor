# Git Advanced

Advanced Git techniques for history rewriting, debugging, large repository management, and workflow optimization beyond everyday branching and merging.

## Interactive Rebase

```bash
# Rebase last N commits interactively
git rebase -i HEAD~5

# Rebase onto a branch
git rebase -i main

# Commands in the rebase editor:
# pick   = keep commit as-is
# reword = keep commit, edit message
# edit   = pause to amend the commit
# squash = meld into previous commit (keep message)
# fixup  = meld into previous commit (discard message)
# drop   = remove commit entirely
# exec   = run a shell command

# Autosquash: fixup commits by convention
git commit --fixup=abc1234
git commit --squash=abc1234
git rebase -i --autosquash main

# Rebase preserving merge commits
git rebase -i --rebase-merges main

# Abort a rebase in progress
git rebase --abort

# Continue after resolving conflicts
git rebase --continue

# Skip the current commit
git rebase --skip
```

## Bisect (Binary Search for Bugs)

```bash
# Start bisect session
git bisect start

# Mark current state as bad
git bisect bad

# Mark a known-good commit
git bisect good v2.0.0

# Git checks out a midpoint -- test it, then mark
git bisect good   # or
git bisect bad

# Automate with a test script (exit 0 = good, exit 1 = bad)
git bisect start HEAD v2.0.0
git bisect run ./test_for_bug.sh

# Automate with a make/test command
git bisect run make test

# View bisect log
git bisect log

# Replay a bisect session
git bisect replay bisect.log

# End bisect session
git bisect reset

# Skip untestable commits
git bisect skip
```

## Reflog (Safety Net)

```bash
# Show reflog for HEAD
git reflog

# Show reflog for a specific branch
git reflog show feature/auth

# Recover a deleted branch
git reflog | grep "feature/deleted"
git checkout -b feature/recovered abc1234

# Undo a hard reset
git reflog
git reset --hard HEAD@{2}

# Recover after a bad rebase
git reflog
git reset --hard ORIG_HEAD

# Reflog entries expire (default 90 days for reachable, 30 for unreachable)
git reflog expire --expire=now --all
git gc --prune=now   # actually remove unreachable objects
```

## Subtree

```bash
# Add a subtree (embed another repo in a subdirectory)
git subtree add --prefix=vendor/lib https://github.com/org/lib.git main --squash

# Pull updates from upstream
git subtree pull --prefix=vendor/lib https://github.com/org/lib.git main --squash

# Push changes back upstream
git subtree push --prefix=vendor/lib https://github.com/org/lib.git main

# Split subtree into its own branch (for extraction)
git subtree split --prefix=vendor/lib --branch=lib-standalone

# Use a remote alias for convenience
git remote add lib-upstream https://github.com/org/lib.git
git subtree pull --prefix=vendor/lib lib-upstream main --squash
```

## Sparse Checkout

```bash
# Enable sparse checkout
git sparse-checkout init --cone

# Check out only specific directories
git sparse-checkout set src/frontend docs

# Add more directories
git sparse-checkout add tests/frontend

# List sparse checkout patterns
git sparse-checkout list

# Disable sparse checkout (restore full working tree)
git sparse-checkout disable

# Non-cone mode (gitignore-style patterns)
git sparse-checkout init --no-cone
git sparse-checkout set '/*.md' '/src/core/**' '!/src/core/vendor/**'

# Combine with partial clone for huge repos
git clone --filter=blob:none --sparse https://github.com/org/monorepo.git
cd monorepo
git sparse-checkout set src/my-service
```

## Worktrees

```bash
# Create a worktree for a branch
git worktree add ../hotfix-worktree hotfix/critical

# Create a worktree with a new branch
git worktree add -b feature/new ../feature-worktree main

# List all worktrees
git worktree list

# Remove a worktree
git worktree remove ../hotfix-worktree

# Prune stale worktree metadata
git worktree prune

# Lock a worktree (prevent pruning on removable media)
git worktree lock ../usb-worktree

# Worktrees share the same .git object store
# Each worktree can have a different branch checked out simultaneously
```

## filter-repo (History Rewriting)

```bash
# Install
pip install git-filter-repo

# Remove a file from entire history
git filter-repo --invert-paths --path secrets.env

# Remove a directory from history
git filter-repo --invert-paths --path vendor/

# Move everything into a subdirectory (for monorepo migration)
git filter-repo --to-subdirectory-filter services/api

# Rename/rewrite paths
git filter-repo --path-rename old-dir/:new-dir/

# Replace text in all files (e.g., remove credentials)
git filter-repo --replace-text expressions.txt
# expressions.txt format: literal:old==>new  or  regex:pattern==>replacement

# Rewrite author info
git filter-repo --email-callback '
    return email.replace(b"old@example.com", b"new@example.com")
'

# Strip large blobs
git filter-repo --strip-blobs-bigger-than 10M

# Analyze repo (find large files, frequent paths)
git filter-repo --analyze
cat .git/filter-repo/analysis/blob-shas-and-paths.txt
```

## Rerere (Reuse Recorded Resolution)

```bash
# Enable rerere globally
git config --global rerere.enabled true

# After resolving a conflict, rerere records it automatically
# Next time the same conflict appears, it's resolved automatically

# View recorded resolutions
git rerere status
git rerere diff

# Forget a specific resolution
git rerere forget path/to/file.txt

# Clean old resolutions (default: 60 days resolved, 15 days unresolved)
git rerere gc
```

## Notes, Bundle, Replace

```bash
# Notes: attach metadata to commits without changing SHA
git notes add -m "Reviewed by: Alice" abc1234
git notes show abc1234
git notes list
git notes remove abc1234

# Push/fetch notes
git push origin refs/notes/*
git fetch origin refs/notes/*:refs/notes/*

# Bundle: offline transfer (sneakernet)
git bundle create repo.bundle --all
git bundle create update.bundle main ^origin/main   # incremental

# Clone or fetch from bundle
git clone repo.bundle myrepo
git fetch update.bundle main:refs/remotes/origin/main

# Verify bundle
git bundle verify repo.bundle

# Replace: substitute one object for another
git replace abc1234 def5678    # make abc1234 point to def5678
git replace --graft abc1234 parent1 parent2   # rewrite parentage
git replace -l                  # list replacements
git replace -d abc1234          # delete replacement
```

## Maintenance and Performance

```bash
# Run maintenance tasks
git maintenance start          # register repo for background maintenance
git maintenance run            # run all tasks now
git maintenance run --task=gc  # specific task

# Commit graph (speeds up log, merge-base, reachability)
git commit-graph write --reachable
git commit-graph verify

# Multi-pack index (speeds up object lookup across packs)
git multi-pack-index write
git multi-pack-index repack --batch-size=500m

# Filesystem monitor (speeds up status/diff on large repos)
git config core.fsmonitor true       # built-in FSMonitor (Git 2.37+)
git config core.untrackedcache true  # cache untracked files

# Partial clone (fetch objects on demand)
git clone --filter=blob:none https://github.com/org/huge-repo.git     # blobless
git clone --filter=tree:0 https://github.com/org/huge-repo.git        # treeless

# Shallow clone
git clone --depth=1 https://github.com/org/repo.git
git fetch --deepen=10    # fetch 10 more commits
git fetch --unshallow     # fetch full history
```

## Commit Signing

```bash
# Configure GPG signing
git config --global user.signingkey ABCDEF1234567890
git config --global commit.gpgsign true
git config --global tag.gpgsign true

# Sign a commit
git commit -S -m "Signed commit"

# Sign with SSH key (Git 2.34+)
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub

# Create allowed signers file
echo "user@example.com ssh-ed25519 AAAA..." > ~/.config/git/allowed_signers
git config --global gpg.ssh.allowedSignersFile ~/.config/git/allowed_signers

# Verify commits
git log --show-signature
git verify-commit abc1234

# Verify tags
git tag -v v1.0.0
```

## Tips

- Use `git reflog` as your safety net -- it records every HEAD movement for 90 days, making almost any mistake recoverable
- Automate bug hunting with `git bisect run` -- write a script that exits 0 for good and 1 for bad, and bisect tests it in $O(\log n)$ commits
- Enable `rerere` globally to avoid re-resolving the same merge conflicts during long-lived rebase workflows
- Use worktrees instead of stashing or switching branches -- they let you work on multiple branches simultaneously without context-switching costs
- Prefer `git filter-repo` over the deprecated `git filter-branch` -- it is 10-100x faster and handles edge cases correctly
- Use sparse checkout with partial clone (`--filter=blob:none`) for monorepos -- download only the code you need
- Run `git maintenance start` on large repositories to enable background optimization (prefetch, commit-graph, incremental repack)
- Sign commits with SSH keys (Git 2.34+) instead of GPG for simpler key management -- most developers already have SSH keys
- Use `--fixup` and `--autosquash` during development to keep a clean history without manual rebase editing
- Use `git bundle` for transferring repositories across air-gapped networks or as offline backups
- Check `git reflog expire` settings before running `git gc --prune=now` -- once reflog entries expire, those commits are truly gone
- Use `git notes` for code review metadata, deployment tracking, or CI results without altering commit history

## See Also

- Git internals (objects, packfiles, refs)
- GitHub Actions CI/CD
- Conventional Commits specification
- Monorepo management (Nx, Turborepo)
- Pre-commit hooks framework

## References

- [Pro Git Book (Scott Chacon)](https://git-scm.com/book/en/v2)
- [Git Reference Manual](https://git-scm.com/docs)
- [git-filter-repo Documentation](https://github.com/newren/git-filter-repo)
- [Git Internals (Plumbing and Porcelain)](https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain)
- [Git Maintenance Documentation](https://git-scm.com/docs/git-maintenance)
- [Git Worktree Documentation](https://git-scm.com/docs/git-worktree)
