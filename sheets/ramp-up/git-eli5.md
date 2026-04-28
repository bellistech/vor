# Git — ELI5 (Time Machine for Code)

> Git is a time machine for your code: every save is a snapshot you can travel back to, and every branch is an alternate timeline where you can try things without breaking the original.

## Prerequisites

(none)

This sheet starts at the very beginning. You do not need to know what "version control" is. You do not need to have ever used the command line for anything. You do not need to know what GitHub is. By the end of this sheet you will know what Git is, why it exists, what every common command does, what every common error means, and how to dig yourself out of trouble. You will also know enough vocabulary to read a co-worker's "I rebased onto main and force-pushed" sentence without flinching.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't start with `$` are what your computer prints back at you. We call that "output."

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

## What Even Is Git?

### Imagine a time machine

Picture a time machine. Not the science-fiction kind that breaks paradoxes. A simple, friendly time machine that has one button on it: **"save right now."** Every time you press the button, the time machine takes a perfect photograph of your project. The whole project. Every file. Every folder. Every byte. Then it stuffs that photograph into a giant filing cabinet, with a sticker on it that says when it was taken and a unique fingerprint nobody else's photograph has.

Years later, you can walk up to the filing cabinet, find any photograph, and step right back into that exact moment. The files are exactly as they were. The folders are exactly as they were. Nothing is missing. Nothing is corrupted. Nothing is approximate. It is the project, frozen in time, ready to be revisited.

That time machine is **Git.**

The button is `git commit`. The photograph is a **commit**. The filing cabinet is the **repository** (or "repo" for short). The unique fingerprint is the **commit hash**. Stepping back into a moment is `git checkout` (or the newer `git switch` and `git restore`).

Git is the world's most popular time machine for code. Every modern company that writes software uses Git or something very much like it. When you push a button on Twitter, or open a file on Dropbox, or stream a movie on Netflix, you are using software that lives inside Git. The codebase that runs the Linux kernel is in Git. The codebase that runs your phone's operating system is in Git. The website you bought your toothbrush from is probably in Git.

### Imagine branching timelines

A regular time machine just goes backward and forward through one timeline. Git's time machine is more powerful than that. Git lets you make **alternate timelines.**

Picture this. You have your project. You have been clicking the save button (committing) for weeks, building up a long line of photographs in the filing cabinet. Each photograph is the project after one more day of work.

Now you want to try something risky. Maybe you want to rip out the whole login system and replace it with something completely different. Maybe you want to redesign the home page. Maybe you want to experiment with a feature that you're not sure you'll keep.

If you just start changing files, you are messing with the original timeline. Bad changes can wreck what was working. So instead, Git lets you press a button that says **"start a new timeline from right here."** That button is `git branch` (or `git switch -c` to create and jump to it in one step).

Now there are two timelines. The old one keeps going from where it was (this is usually called **`main`** or **`master`**). The new one is a copy that diverges from that point forward. You can keep saving (committing) to the new timeline. Nothing you do in the new timeline can affect the original. The original is safe. You can go nuts in the new timeline. Try crazy things. Break stuff. Delete files. Rewrite everything.

If your experiment works, you can **merge** the two timelines. Git takes all the changes you made in your alternate timeline and folds them carefully back into the original timeline. Now your changes are part of the main project.

If your experiment fails, you can **delete the branch.** The original timeline is untouched. Nothing was lost. You walked into a possible future, didn't like it, and walked back out.

This is the killer feature of Git. The ability to fork off an alternate reality, work in it freely, and then either fold it back into the main reality or throw it away. No other system before Git made this so easy or so cheap. Branches in Git are nearly free. You can have a thousand of them. Most projects have dozens of them at any given time, one per feature or fix.

### Imagine a shared time machine

The third magic trick is that the time machine isn't just on your computer. There is also a **shared time machine** that lives in the cloud. Everyone on your team has their own copy of the time machine on their own computer. There is one in the cloud (called the **remote**, often hosted on GitHub or GitLab). Everyone's local time machine talks to the cloud time machine.

When you click "save" on your local time machine, the photograph goes into your local filing cabinet. To share it with your team, you push it up to the shared cloud time machine. That is `git push`. To download photographs other people pushed up, you pull them down. That is `git pull` (or its more careful cousin `git fetch`).

Everyone has the entire history. Everyone. Not just the latest version. The entire history, all the way back to the very first commit, sitting on every developer's laptop. This is what people mean when they say Git is **"distributed."** No single computer is the source of truth. The cloud copy is just one copy among many. If the cloud blew up, you could rebuild it from any developer's laptop.

This is also why Git is fast. When you ask Git to show you the history, it doesn't reach out over the network. It looks at your local copy. When you switch branches, no network. When you make a commit, no network. The only times Git talks to the cloud are `git push`, `git pull`, `git fetch`, and `git clone`. Everything else is offline. You can work on Git on a plane with no Wi-Fi all day long.

### Snapshots, not deltas

Other version control systems before Git (CVS, Subversion, Perforce) saved space by storing **deltas** — only the changes between versions. They would say "version 5 of file.txt is version 4 plus these three line changes." This sounds clever and efficient, but it makes everything slow. To reconstruct the file at version 5, you have to start from version 1 and play forward all the changes. To compare version 5 with version 50, you have to play forward 45 sets of changes.

Git does it differently. Git stores **full snapshots.** Every commit contains the complete state of every file. If a file didn't change between commit 1 and commit 2, Git just stores a pointer to the same data — it doesn't duplicate it. But conceptually, every commit is a complete photograph, not a list of changes.

This is why Git is so fast. Want the file at commit 50? Git looks up commit 50, finds the pointer to that file's content, hands it to you. Done. No replaying. No reconstruction. Want to compare two arbitrary commits? Git finds the snapshots and diffs them on the fly. No replaying.

The trick that makes this affordable in disk space is **content-addressed storage**. We will get to that in a moment, but the short version is: identical content is stored once, no matter how many commits include it.

### Local-first by design

A consequence of the "everyone has the whole history" model is that Git is **local-first.** The center of the world, for you, is your own computer. The remote is just one of several places you could push to. You could push to multiple remotes. You could have no remote at all. You could collaborate by emailing patches to each other and never use a remote.

This was a deliberate design choice. The original use case for Git was the Linux kernel, where thousands of developers around the world contribute to the same codebase. There was no central server they could all hit reliably. So Git was built so each developer has full power on their own machine, and "syncing" with anyone else is a separate, optional operation.

For most teams today, the workflow has settled around having one central remote (typically GitHub or GitLab) that everyone treats as the source of truth. But the underlying tool doesn't care. It works just as well with no central server, two remotes, or fifty remotes.

## The Three Trees

Git has a model of your project that involves three "trees." A tree, in Git terminology, is just a snapshot of files and folders. The three trees are:

1. **The working directory** — the actual files on your disk that you edit with your editor. If you `ls` your project folder, this is what you see. This is where you make changes.

2. **The index** (also called the **staging area**) — a draft of what your next commit will look like. Files you have added with `git add` are in the index. Files you haven't added are not in the index yet.

3. **HEAD** (also called the **repository** or just "the committed history") — the latest commit, plus everything before it. The permanent record. The filing cabinet of photographs.

The flow is:

```
       edit                git add               git commit
working ───────────► index ───────────► HEAD ───────────► history
directory            (staging)         (latest commit)
```

You edit a file in the working directory. You `git add` it to move that change into the index. You `git commit` to bake the index into a new commit, which becomes the new HEAD.

### A picture of the three trees

```
+----------------------------------+
|        WORKING DIRECTORY         |   what you see when you ls
|  (real files on your disk)       |
+----------------------------------+
              |
              |  git add
              v
+----------------------------------+
|             INDEX                |   "what my next commit will be"
|         (staging area)           |
+----------------------------------+
              |
              |  git commit
              v
+----------------------------------+
|              HEAD                |   "the latest commit"
|     (most recent snapshot)       |
+----------------------------------+
              |
              |  parent pointers
              v
+----------------------------------+
|       OLDER COMMITS              |   the rest of history
|   (the filing cabinet)           |
+----------------------------------+
```

### Why three trees and not two?

Other version control systems have two stages: edit, then commit. Git has three because the middle stage (the index) is incredibly useful.

The index lets you **prepare** a commit. You can stage some changes from a file but not others. You can stage one file and not another. You can review what's about to go into the commit before committing. You can build up a commit piece by piece. This is the difference between "I'll just commit everything I changed" and "I'll commit a clean, focused change with exactly the right files in it."

A common workflow is:

1. Edit five files in your working directory.
2. `git add` only the three that belong to one logical change.
3. `git commit -m "fix the bug"` — those three files are now committed.
4. `git add` the other two.
5. `git commit -m "update docs"` — those two are now a separate commit.

You ended up with two clean, focused commits instead of one giant grab-bag commit. The index is the buffer that lets you do this.

If three trees feels like too much, you can mostly ignore the index by using `git commit -a`, which stages and commits all tracked changes in one step. But if you ever want to make a clean commit history, the index is your friend.

## Commits — The Snapshots

A **commit** is a single photograph of your project. Each commit contains:

- **Tree** — the snapshot of every file and folder, frozen at this moment.
- **Parent commit(s)** — usually one parent (the previous commit on this branch), but sometimes two (a merge commit) or zero (the very first commit, called the "root commit").
- **Author** — who originally wrote the change (name + email).
- **Committer** — who actually committed it (name + email). Usually the same as the author, but they can differ if someone else applies your patch.
- **Author date** — when the change was originally written.
- **Commit date** — when the commit was made (these can differ if you rebase or amend).
- **Message** — a human-readable description of what changed and why.
- **Hash** — a unique fingerprint of the entire contents above. The hash is computed from all the other fields, so any change to any field produces a totally different hash.

### Hashes are content-addressed

The fingerprint is calculated by running a cryptographic hash function (SHA-1 historically, with SHA-256 being phased in) over the entire contents of the commit. The same content always produces the same hash. Different content always produces a different hash.

This is called **content-addressed storage**. Instead of giving every commit a sequential number (commit 1, commit 2, commit 3), Git gives every commit a name that *is* a fingerprint of the content. Two commits with the same parent, same files, same author, same message at the same instant would have the same hash — they are literally the same commit.

This has a few magical consequences:

- **Tamper-evident.** If anyone changes any byte of any commit anywhere, the hash changes, and Git will notice. You cannot secretly edit history without Git knowing.
- **Deduplication.** If two branches happen to produce the same commit, they share the same hash and the same storage. No duplication.
- **Distributed.** Two computers that both make a commit with the same content will produce the same hash, automatically. There is no central authority handing out IDs.
- **Cryptographic chain.** Each commit's hash includes its parent's hash. So if you change anything anywhere in history, every later commit's hash changes too. The whole chain is bound together by hashes. This is the same trick used by blockchains.

Hashes are 40 hex characters long for SHA-1 (`a3c5e8b9d2f1...`) or 64 for SHA-256. Most of the time you can use just the first 7 or so characters because they are unique enough — Git will accept any unambiguous prefix. So `a3c5e8b` is just as good as the full hash for most commands.

### What a commit looks like internally

If you peek inside `.git/objects` and unpack a commit with `git cat-file -p`, you see something like:

```
tree 4f7a1b2c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a
parent 9f8e7d6c5b4a3210fedcba9876543210abcdef12
author Stevie <stevie@example.com> 1761588000 -0400
committer Stevie <stevie@example.com> 1761588000 -0400

Fix the off-by-one bug in pagination

The cursor was advancing one too many rows when the
result set was exactly equal to the page size.
```

That's the entire commit. A handful of lines of plain text, hashed to produce its fingerprint. The `tree` line points to the snapshot of all files. The `parent` line points to the previous commit. Everything else is metadata.

### Writing good commit messages

Convention says a commit message should look like this:

```
Short summary line, max 72 chars, imperative mood

Optional longer body explaining the WHY. Wrap at 72 columns.
Talk about the motivation, the tradeoffs you considered, and
anything reviewers should know.

Refs #1234
```

The first line is the headline. Imperative mood means "Fix the bug" not "Fixed the bug" or "Fixes the bug" — read it as "if applied, this commit will Fix the bug." A blank line follows. Then a body explaining the why, not the what (the diff already shows the what).

Some teams use **Conventional Commits**, which adds a structured prefix:

```
feat(auth): add OAuth2 support
fix(db): close connection after migration
docs: update README install steps
chore: bump dependency versions
refactor(api): split user controller into modules
```

The prefix lets tooling (changelog generators, semantic-version bumpers) parse messages programmatically. We will see this again in the vocabulary section.

## Branches and HEAD

### A branch is just a moveable pointer

A branch sounds fancy. It is not. A branch in Git is **a single file that contains the hash of one commit.** That's it.

When you say `git branch feature-x`, Git creates a tiny file at `.git/refs/heads/feature-x` that contains, say, `a3c5e8b9d2f1...`. Now there is a name, `feature-x`, that points at that commit.

When you commit on the branch, Git replaces the contents of that file with the hash of the new commit. The branch "moves forward" automatically. This is why branches are so cheap — creating a branch creates a 41-byte file. Git could have a million branches and barely use any disk space.

```
                               main
                                 v
o ─── o ─── o ─── o ─── o ─── o
               \
                o ─── o ─── o
                            ^
                       feature-x
```

Each `o` is a commit. `main` is a label pointing at the latest commit on the main timeline. `feature-x` is a label pointing at the latest commit on the feature timeline. Both labels are just strings stored in tiny files.

### HEAD: where am I right now?

**HEAD** is "where I am right now." HEAD is also a tiny file (`.git/HEAD`) that usually contains the name of a branch, like `ref: refs/heads/main`.

When you run `git switch feature-x`, Git updates HEAD to point at `feature-x`. Now any commit you make will move `feature-x`, not `main`. When you run `git switch main`, HEAD goes back to pointing at `main`, and `main` resumes moving when you commit.

So the layered picture is:

```
                              main          feature-x
                                v               v
o ─── o ─── o ─── o ─── o ─── o ─── o ─── o ─── o
                                    ^
                                  HEAD
                              (currently main)
```

Git uses HEAD to know where to put your next commit, what your working directory should match, what `git diff` is comparing against, and so on. It is the cursor that says "you are here."

### Detached HEAD

Sometimes HEAD points directly at a commit instead of at a branch. This is **detached HEAD** state. It happens if you `git checkout <hash>` directly, or `git switch --detach <hash>`. You are looking at a moment in history but not on any branch.

Git will warn you about this:

```
$ git checkout a3c5e8b
Note: switching to 'a3c5e8b'.

You are in 'detached HEAD' state. You can look around, make experimental
changes and commit them, and you can discard any commits you make in this
state without impacting any branches by switching back to a branch.

If you want to create a new branch to retain commits you create, you may
do so (now or later) by using -c with the switch command. Example:

  git switch -c <new-branch-name>

HEAD is now at a3c5e8b old commit message
```

If you commit in detached HEAD, those commits aren't on any branch. If you switch away, they become orphaned (still in the database, but unreachable). The reflog will save them for ~90 days, but they are otherwise lost.

This isn't a bug — it is sometimes useful (e.g., bisecting). But it is a footgun if you didn't mean to detach. The fix is always to `git switch -c new-branch` and convert your detached commits into a real branch.

### A branch and HEAD picture

```
+---------+
| HEAD    |  ──►  refs/heads/main
+---------+

+---------+
| main    |  ──►  commit a3c5e8b
+---------+

+---------+
| a3c5e8b |  ──►  parent: 9f8e7d6c
+---------+      tree: ...
                 author: ...
                 message: "Fix bug"
```

HEAD points at a branch. The branch points at a commit. The commit points at its parent and at a snapshot of files. The whole graph is built out of pointers.

## Merging vs Rebasing

You have two timelines and you want to combine them. Git has two ways to do that: **merge** and **rebase**. They produce different histories. Both are valid. Both have tradeoffs.

### Merge: weld the timelines together

`git merge feature-x` while on `main` does this:

1. Find the common ancestor of `main` and `feature-x` (the last commit they share).
2. Look at what changed on `main` since that ancestor.
3. Look at what changed on `feature-x` since that ancestor.
4. Combine the two sets of changes into a single new commit, the **merge commit**, which has *two* parents — one on each branch.

```
Before:
            o ─── o ─── o   <- main
           /
o ─── o ─── o
           \
            o ─── o ─── o   <- feature-x

After git merge feature-x (while on main):
            o ─── o ─── o ─── M   <- main
           /                 /
o ─── o ─── o               /
           \               /
            o ─── o ─── o   <- feature-x
```

The merge commit `M` has two parent arrows. The history now has a "diamond" shape. The full path of how each branch developed is preserved.

If `main` did not change since `feature-x` started, Git can do a **fast-forward** instead of a real merge. There is nothing to combine — `main` just leaps forward to where `feature-x` is. No merge commit is created.

```
Before:                          After fast-forward:
                                       main
                                        v
o ─── o ─── o   <- main         o ─── o ─── o ─── o ─── o
           \                                            ^
            o ─── o ─── o   <- feature-x                feature-x
                        ^
                        feature-x
```

The branches are now identical. The "feature-x" history is the new "main" history.

You can force a merge commit even when fast-forward is possible with `git merge --no-ff`. Some teams require this to make every feature branch visible in the history.

### Rebase: replay the commits onto a new base

`git rebase main` while on `feature-x` does something completely different:

1. Find the common ancestor of `main` and `feature-x`.
2. Take all the commits from that ancestor up to `feature-x`'s tip.
3. Set them aside.
4. Move `feature-x` to point at `main`'s tip.
5. Replay those saved commits, one by one, onto the new tip. Each replay produces a *new* commit with a new hash. The original commits are abandoned (still in the database, reachable via reflog, but no longer on the branch).

```
Before:
            o ─── o ─── o   <- main
           /
o ─── o ─── o
           \
            A ─── B ─── C   <- feature-x

After git rebase main (while on feature-x):
                          A' ─── B' ─── C'   <- feature-x
                         /
            o ─── o ─── o   <- main
           /
o ─── o ─── o
           \
            A ─── B ─── C   <- (orphaned, in reflog)
```

`A'`, `B'`, `C'` are *new* commits with the same content but new parents (and thus new hashes). The original `A`, `B`, `C` are still in the object database but no longer reachable from any branch.

The result is a **linear history.** `feature-x` now sits on top of `main` as if it had always been written there.

### Merge vs rebase tradeoffs

**Merge preserves true history.** You can see exactly when a feature branched off, when it merged back, and how it developed in parallel with main. The history is messy but accurate.

**Rebase creates clean history.** Every commit is in a single line. There are no diamonds. Reading history is simpler. But the history is fictional — it pretends the branch never existed in parallel.

**Merge is safe.** Merging never rewrites existing commits. The hashes you had yesterday are still the same hashes today.

**Rebase is dangerous on shared branches.** If you rebase a branch other people are working on, their copy and your copy now have completely different hashes. When they pull, Git can't reconcile and you get angry conflicts. **Golden rule: never rebase commits that have been pushed to a shared branch.** Only rebase your own private branches.

**Merge is good for "this feature was developed by a team."** Rebase is good for "this commit should look like it was always meant to be this way."

Many teams have a policy: rebase your feature branch on top of `main` to keep it up to date, then merge it (with a merge commit) when it's ready. That gives both clean per-commit history and a visible "feature merged here" point.

### Three-way merge and merge conflicts

When Git has to combine two timelines, it does a **three-way merge.** It looks at:

1. The version of the file on `main`.
2. The version of the file on `feature-x`.
3. The common ancestor's version of the file.

If both branches changed *different* parts of the file, Git can usually combine them automatically. If both branches changed the *same* lines, Git can't decide which version is right. This is a **merge conflict.**

Git marks the conflicted region in the file with markers:

```
<<<<<<< HEAD
print("Hello, World!")
=======
print("Hello, Git!")
>>>>>>> feature-x
```

You edit the file to keep the right version (or combine them yourself), remove the markers, `git add` the file, and `git commit` to finish the merge. If you're in the middle of a rebase, you `git rebase --continue` instead.

If you panic and want to back out: `git merge --abort` (or `git rebase --abort` during a rebase) restores the state from before you started.

## The Index (Staging Area)

The **index** is a buffer between your working directory and your committed history. It holds the files you have explicitly said "yes, include this in the next commit" by running `git add`.

### Why have an index?

The index lets you **construct** a commit deliberately, instead of just dumping every file you happened to change. You can:

- Stage some files and not others. Maybe you fixed a bug AND made a typo while you were in there. You want to commit only the bug fix and leave the typo for later.
- Stage some changes within a file, not others. `git add -p` walks you through each "hunk" (chunk of changes) and asks if you want to stage it.
- Review the staged changes before committing with `git diff --cached`.
- Unstage with `git restore --staged file.txt` if you change your mind.

### `git diff` versus `git diff --cached`

`git diff` (no flags) shows the difference between **the working directory and the index.** That is, "what have I changed since I last staged?"

`git diff --cached` (or `git diff --staged`, same thing) shows the difference between **the index and HEAD.** That is, "what have I staged that isn't yet committed?"

`git diff HEAD` shows the difference between **the working directory and HEAD.** That is, "what would change if I staged everything and committed it?"

```
working dir ──[git diff]──► index ──[git diff --cached]──► HEAD
                                                            ^
                              ◄────────[git diff HEAD]──────┘
```

These three diff modes correspond to the three trees. Knowing which one to use saves a lot of confusion.

### Partial staging with `git add -p`

`git add -p` is one of Git's hidden gems. It walks through every change in your working directory, hunk by hunk, and asks what to do:

```
$ git add -p
diff --git a/main.go b/main.go
index 1234567..89abcde 100644
--- a/main.go
+++ b/main.go
@@ -10,6 +10,8 @@ func main() {
        x := 1
        y := 2
+       // bugfix: account for off-by-one
+       z := x + y - 1
        fmt.Println(z)
 }
(1/1) Stage this hunk [y,n,q,a,d,e,?]?
```

You answer each prompt with a single letter: `y` (yes, stage this hunk), `n` (no, skip), `q` (quit), `a` (yes to all remaining), `d` (no to all remaining), `s` (split this hunk into smaller pieces), `e` (edit this hunk manually). The `?` shows the full menu.

This is how you make small, focused commits even when you've made a big mess of changes. Stage just the bug fix. Commit. Stage the typo fix. Commit. Stage the refactor. Commit. The fact that you did them all in one editing session doesn't matter — your history reads as if you did them one at a time.

## Reflog — Your Safety Net

The **reflog** is Git's "undo history." Every time HEAD moves — every commit, every checkout, every reset, every rebase, every merge — Git writes a line to the reflog. So even if a branch gets deleted, even if you `git reset --hard` over your work, even if you rebase and lose commits, **they are still in the reflog** for ~90 days by default.

```
$ git reflog
a3c5e8b HEAD@{0}: rebase (finish): returning to refs/heads/feature-x
a3c5e8b HEAD@{1}: rebase: pick: third commit
9f8e7d6 HEAD@{2}: rebase: pick: second commit
1234567 HEAD@{3}: rebase (start): checkout main
abcdef0 HEAD@{4}: commit: third commit
fedcba9 HEAD@{5}: commit: second commit
0987654 HEAD@{6}: commit: first commit
```

Each line is a position HEAD was in. `HEAD@{0}` is now. `HEAD@{1}` is one move ago. `HEAD@{2}` is two moves ago. And so on.

### "I deleted my branch"

You committed a bunch of work to a branch, then you accidentally `git branch -D` it. The branch is gone. Are your commits gone?

No. They are in the reflog.

```
$ git reflog
abcdef0 HEAD@{4}: commit: my important work
$ git branch recovered abcdef0
$ git switch recovered
```

Branch resurrected. Work intact.

### "I rebased and lost commits"

You rebased and a conflict resolution went wrong, and now your branch looks weird and a commit seems to be missing. Find the pre-rebase position in the reflog:

```
$ git reflog
a3c5e8b HEAD@{2}: rebase (finish): returning to refs/heads/feature-x
9f8e7d6 HEAD@{3}: rebase (start): checkout main
1234567 HEAD@{4}: commit: my work right before the rebase
$ git reset --hard HEAD@{4}
```

You are now exactly where you were before the rebase. Try again.

### "I ran reset --hard"

You ran `git reset --hard origin/main` and threw away all your local work. Disaster!

```
$ git reflog
1234567 HEAD@{0}: reset: moving to origin/main
abcdef0 HEAD@{1}: commit: my brilliant work I just lost
$ git reset --hard HEAD@{1}
```

Restored.

The reflog is local-only. It is not pushed to the remote. It is not pulled from the remote. It is your private safety net for *your* HEAD movements on *your* machine.

By default the reflog keeps reachable entries for 90 days and unreachable entries for 30 days, but `git gc` can prune expired entries. You can change these with `gc.reflogExpire` and `gc.reflogExpireUnreachable`.

## Stash

Sometimes you are in the middle of editing and you need to switch branches *right now* to fix something else. But your changes aren't ready to commit — they're half-done. You don't want to commit garbage just to clean your working directory.

Enter `git stash`. It takes all your uncommitted changes (both staged and unstaged), tucks them into a special "stash stack," and restores your working directory to a clean state.

```
$ git status
On branch feature-x
Changes not staged for commit:
        modified:   main.go
        modified:   utils.go
$ git stash
Saved working directory and index state WIP on feature-x: a3c5e8b previous commit
$ git status
On branch feature-x
nothing to commit, working tree clean
```

Your changes are gone from your working directory but stored in the stash. Now you can switch branches, do whatever you need, then come back and:

```
$ git stash pop
On branch feature-x
Changes not staged for commit:
        modified:   main.go
        modified:   utils.go
Dropped refs/stash@{0} (a3c5e8b)
```

Your changes are back. The stash entry is removed.

### Stash list

You can have multiple stashes:

```
$ git stash list
stash@{0}: WIP on feature-x: a3c5e8b previous commit
stash@{1}: On main: experiment with new layout
stash@{2}: WIP on hotfix-1.2: e1f2a3b before lunch
```

Apply a specific one with `git stash pop stash@{1}` or `git stash apply stash@{1}` (apply keeps the stash entry; pop removes it).

### Why stash instead of commit-and-uncommit?

You could fake-commit your work-in-progress, switch branches, then come back and `git reset HEAD~1` to uncommit. People do this. But it's clunky and easy to mess up. Stash is purpose-built and easier.

Stash entries do not get pushed to remotes. They are local. If you want to share work-in-progress, use a real commit on a real branch.

## Tags

A **tag** is a name for a specific commit. Usually used to mark releases.

There are two kinds:

**Lightweight tags** are just a name pointing at a commit, like a branch that doesn't move. Created with `git tag v1.0.0`.

**Annotated tags** are full Git objects that include a tagger name, email, date, message, and optionally a GPG signature. Created with `git tag -a v1.2.3 -m 'Release 1.2.3'` (or `git tag -s` to also sign with GPG).

Annotated tags are the right choice for releases. Lightweight tags are fine for personal "remember this commit" markers.

```
$ git tag -a v1.2.3 -m 'Release 1.2.3'
$ git show v1.2.3
tag v1.2.3
Tagger: Stevie <stevie@example.com>
Date:   Mon Apr 27 12:00:00 2026 -0400

Release 1.2.3

commit a3c5e8b9d2f1...
Author: Stevie <stevie@example.com>
Date:   Mon Apr 27 11:55:00 2026 -0400

    Bump version to 1.2.3
...
```

Tags are not pushed by default — `git push` does NOT push tags. You have to `git push origin v1.2.3` for one tag, or `git push origin --tags` to push them all.

## Remotes

A **remote** is a named URL that points at another copy of your repository. The default remote name is `origin`. You can have multiple remotes.

```
$ git remote -v
origin  git@github.com:bellistech/vor.git (fetch)
origin  git@github.com:bellistech/vor.git (push)
upstream        git@github.com:original/cs.git (fetch)
upstream        git@github.com:original/cs.git (push)
```

This is the typical fork setup: `origin` is your fork, `upstream` is the original repo you forked from.

### `git fetch` vs `git pull`

`git fetch` downloads new commits, branches, and tags from a remote, but does **not** touch your working directory or current branch. It just updates your `origin/main`, `origin/feature-x`, etc., refs.

`git pull` is `git fetch` followed by either `git merge` or `git rebase` (depending on configuration) to integrate the fetched changes into your current branch.

If you want to see what's new without immediately integrating, use `git fetch`. Look around with `git log HEAD..origin/main`. Decide what to do. Then merge or rebase explicitly.

### Tracking branches

When you `git clone`, your local `main` is set up to **track** `origin/main`. That means:

- `git pull` knows where to pull from (`origin/main`).
- `git push` knows where to push to (`origin/main`).
- `git status` tells you "your branch is ahead/behind origin/main by N commits."

If you create a new branch locally and want it to track a remote branch:

```
$ git push -u origin feature-x
```

The `-u` (or `--set-upstream`) tells Git "this local branch should track `origin/feature-x` from now on." After that, plain `git push` and `git pull` work without arguments.

### Remote-tracking branches

`origin/main` is a **remote-tracking branch.** It is your local cache of where `main` was on the remote the last time you fetched. It does NOT update automatically — you have to `git fetch` or `git pull` for it to refresh.

The naming is "remote-name slash branch-name." So `origin/main` means "the `main` branch on the remote named `origin`," as of the last fetch.

You can see them with `git branch -r` (just remotes) or `git branch -a` (all branches, local and remote-tracking).

## Pull Requests / Merge Requests

A **pull request (PR)** on GitHub, or **merge request (MR)** on GitLab, is the collaboration mechanism that wraps Git for team workflows.

### The fork-and-PR flow

```
       +-------------------+
       |   upstream/main   |   <- the canonical repository
       +-------------------+
                |
                | fork (one-time copy on the server)
                v
       +-------------------+
       |    origin/main    |   <- your fork on the server
       +-------------------+
                |
                | clone
                v
       +-------------------+
       |    local/main     |   <- your laptop
       +-------------------+
                |
                | git switch -c feature-x
                | edit files
                | git commit
                v
       +-------------------+
       | local/feature-x   |
       +-------------------+
                |
                | git push -u origin feature-x
                v
       +-------------------+
       | origin/feature-x  |   <- on your fork on the server
       +-------------------+
                |
                | open pull request on GitHub
                v
       +-------------------+
       |  PR review & CI   |
       +-------------------+
                |
                | merge button clicked
                v
       +-------------------+
       |   upstream/main   |   <- your work is now in the real repo
       +-------------------+
```

The pull request is a *conversation* about a change. Reviewers leave comments on specific lines. CI runs tests automatically. The change can go through many rounds of revision before being merged. Once approved, somebody clicks "merge" and the branch is merged into `main`.

### Three ways to merge a PR

GitHub and GitLab let the maintainer pick how to merge:

1. **Merge commit** — creates a merge commit on `main`, preserving the branch's history. The PR's commits all show up as ancestors of `main`. Best for preserving real development history.

2. **Squash merge** — combines all the PR's commits into a single new commit on `main`. The original commits are discarded. Best for keeping `main` history super clean (one commit per PR).

3. **Rebase merge** — rebases the PR's commits onto `main` and applies them as fast-forwards. No merge commit. The PR's commits become straight-line history on `main`. Best when you want linear history but each PR commit is meaningful.

Different teams pick different strategies. Squash merge is most common for small/medium projects. Merge commit is most common for large projects where the branch development is meaningful.

## Submodules and Subtrees

Sometimes a project depends on another project, and you want to embed that other project's code inside yours. There are two ways to do this in Git.

### Submodules

A **submodule** is a reference to a specific commit of another Git repository, embedded as a subdirectory.

```
$ git submodule add git@github.com:user/lib.git vendor/lib
$ git status
        new file:   .gitmodules
        new file:   vendor/lib
```

The `.gitmodules` file records where the submodule comes from. The `vendor/lib` "file" in the index is actually a pointer to a specific commit of the lib repository. When someone else clones your repo, they need to also `git submodule init && git submodule update` to pull down the actual content.

Submodules cause real pain:

- People forget to `git submodule update` and get a stale or empty submodule.
- Updating a submodule requires committing the new pointer in the parent repo.
- Branching and merging across submodule boundaries is awkward.
- A `git clone` does not pull submodules by default (you need `--recurse-submodules`).

### Subtrees

A **subtree** is the alternative. With `git subtree`, you actually merge another repository's content into your repository as a subdirectory. The other repo's content lives in your repo. No external pointer.

```
$ git subtree add --prefix=vendor/lib git@github.com:user/lib.git main --squash
```

Now `vendor/lib` is just files in your repo. `git clone` works. `git pull` works. Nobody needs to know it came from another repo.

To pull updates from upstream:

```
$ git subtree pull --prefix=vendor/lib git@github.com:user/lib.git main --squash
```

Subtrees are simpler to use day-to-day but the subtree commands themselves are arcane. Most people Google them every time.

For most projects, neither submodules nor subtrees is the right answer. Use a package manager (npm, Cargo, Go modules, pip, etc.) instead. Use submodules/subtrees only when you really need to vendor source code that doesn't have a package distribution.

## Worktrees

A **worktree** is a second working directory attached to the same repository. You can have multiple checkouts of different branches simultaneously, all backed by one `.git` directory.

```
$ git worktree add ../feature-x feature-x
Preparing worktree (checking out 'feature-x')
HEAD is now at a3c5e8b last commit on feature-x
$ ls ../
my-project/        feature-x/
$ git worktree list
/path/to/my-project    a3c5e8b [main]
/path/to/feature-x     9f8e7d6 [feature-x]
```

Now you can edit `main` in one terminal and `feature-x` in another, without switching branches in either. They share objects (so disk space is small) but have independent working directories and indexes.

Worktrees are way better than cloning twice:

- One `.git` directory means objects are shared. No duplication.
- Commits made in one worktree are immediately visible to others.
- You can't have the same branch checked out in two worktrees (Git protects you from this).

Common use: keep `main` checked out in one worktree for hotfixes, work on a feature branch in another. No more "let me stash this and switch branches" dance.

Remove a worktree with `git worktree remove ../feature-x`.

## Hooks

Git **hooks** are scripts that run at specific points in Git operations. They live in `.git/hooks/` and are organized by name.

Common hooks:

- **`pre-commit`** — runs before a commit is made. Use it to lint, format, run tests, check for secrets. If it exits non-zero, the commit is aborted.
- **`commit-msg`** — runs after the message is written but before the commit is finalized. Use it to enforce commit message format (e.g., conventional commits).
- **`prepare-commit-msg`** — runs before the message editor opens. Use it to pre-fill the message with a template.
- **`pre-push`** — runs before `git push` actually pushes. Use it to run tests one last time. Aborts the push if it exits non-zero.
- **`post-merge`** — runs after a merge completes. Use it to e.g., re-install dependencies if a lockfile changed.
- **`post-checkout`** — runs after `git checkout`/`git switch`. Use it to e.g., clear caches that might be branch-specific.

Server-side (on the remote):

- **`pre-receive`** — runs on the remote before any references are updated. Use it to enforce policy (no force-pushes, mandatory signed commits, etc.).
- **`update`** — like pre-receive but runs once per ref instead of once per push.
- **`post-receive`** — runs after the push is accepted. Use it to trigger CI, deploy, or notify chat.

Hooks are executable scripts (any language with a shebang). They are NOT versioned in your repo by default — `.git/hooks/` is per-clone. Tools like **pre-commit** (https://pre-commit.com) solve this by managing hooks declaratively in a `.pre-commit-config.yaml` file that IS versioned, and installing them into `.git/hooks/` from there.

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.56.0
    hooks:
      - id: golangci-lint
```

Then `pre-commit install` writes a `.git/hooks/pre-commit` that runs all the configured hooks. Now everyone on the team gets the same hooks just by cloning and running `pre-commit install`.

## Bisect

You shipped a bug, and you don't know which commit introduced it. The codebase is huge. You have hundreds of commits since the last known good version. Reading every diff would take all week.

Use **`git bisect`**. Git will binary-search through history to find the bad commit.

```
$ git bisect start
$ git bisect bad HEAD          # current commit is broken
$ git bisect good v1.0.0       # v1.0.0 was working
Bisecting: 250 revisions left to test after this (roughly 8 steps)
[abcdef0] Refactor pagination
```

Git checks out the commit halfway between `v1.0.0` and `HEAD`. You test it. If the bug is there:

```
$ git bisect bad
Bisecting: 125 revisions left to test after this (roughly 7 steps)
[1234567] Add caching to user service
```

If the bug is NOT there:

```
$ git bisect good
Bisecting: 62 revisions left to test after this (roughly 6 steps)
[fedcba9] Update CI config
```

After ~log2(N) iterations, Git names the exact commit that introduced the bug:

```
1234567 is the first bad commit
commit 1234567abc
Author: Someone <someone@example.com>
Date:   Tue Apr 21 14:23:00 2026 -0400

    Add caching to user service
...
```

Finish with:

```
$ git bisect reset
Previous HEAD position was 1234567
Switched to branch 'main'
```

You can also automate it with `git bisect run <test-script>` — Git will run your script at each commit and use the exit code to decide good/bad. With a fast test script, you can pinpoint a bug in a 10,000-commit history in under a minute.

## Cherry-Pick

`git cherry-pick <hash>` copies a single commit from somewhere else and applies it to your current branch.

```
$ git switch main
$ git cherry-pick abc123
[main def4567] Fix critical security bug
```

The commit `abc123` is replayed on top of `main` as a *new* commit `def4567`. Same content (same diff), new hash (different parent).

Common use: a critical fix was made on a feature branch, and you need it on `main` immediately without merging the whole branch.

Cherry-pick can produce conflicts, just like merge or rebase. Resolve them, `git add`, `git cherry-pick --continue`. Or `git cherry-pick --abort` to back out.

You can cherry-pick a range: `git cherry-pick abc123..def456` (from but not including abc123, up to and including def456).

## Common Errors

These are the error messages you will hit. Each one with its plain-English meaning and the fix.

### `fatal: not a git repository`

```
$ git status
fatal: not a git repository (or any of the parent directories): .git
```

You ran a Git command in a folder that isn't a Git repo. Either you are in the wrong folder, or you haven't run `git init` yet. Check with `pwd` and `ls -la` to see if `.git/` exists.

### `fatal: pathspec 'X' did not match any files`

```
$ git add maain.go
fatal: pathspec 'maain.go' did not match any files
```

You typed a filename that doesn't exist (probably a typo). Or you're trying to add a file that doesn't exist yet. Or you're trying to add a file that's already gitignored. Check with `ls`.

### `error: failed to push some refs to 'origin'`

```
$ git push
To github.com:user/repo.git
 ! [rejected]        main -> main (fetch first)
error: failed to push some refs to 'github.com:user/repo.git'
hint: Updates were rejected because the remote contains work that you do
hint: not have locally.
```

Someone else pushed to the remote since your last fetch. Your local `main` is behind `origin/main`. Fix:

```
$ git pull --rebase   # or git pull
$ git push
```

Don't `git push --force` to "fix" this — you'll overwrite their work.

### `error: src refspec X does not match any`

```
$ git push origin feature-x
error: src refspec feature-x does not match any
error: failed to push some refs to 'github.com:user/repo.git'
```

The branch `feature-x` doesn't exist locally. Probably a typo, or you forgot to commit, or you're on a different branch than you think. Check with `git branch`.

### `Auto-merging X CONFLICT (content): Merge conflict in X`

```
$ git merge feature-x
Auto-merging main.go
CONFLICT (content): Merge conflict in main.go
Automatic merge failed; fix conflicts and then commit the result.
```

Both branches changed the same lines. You need to resolve manually. Open the file, find the `<<<<<<<` markers, edit the file to the right content, remove the markers, `git add` the file, `git commit` (or `git merge --continue`).

If you panic: `git merge --abort` to back out.

### `error: Your local changes to the following files would be overwritten by checkout`

```
$ git switch main
error: Your local changes to the following files would be overwritten by checkout:
        main.go
Please commit your changes or stash them before you switch branches.
Aborting
```

You have uncommitted changes that conflict with the target branch. Either commit them, or stash them with `git stash`, or discard them with `git restore main.go`.

### `fatal: refusing to merge unrelated histories`

```
$ git pull origin main
fatal: refusing to merge unrelated histories
```

The local repo and the remote repo have completely different starting commits. Usually this happens when you `git init` a local repo, then add a remote that already had its own history. Fix:

```
$ git pull origin main --allow-unrelated-histories
```

Then resolve any conflicts.

### `error: cannot rebase: You have unstaged changes`

```
$ git rebase main
error: cannot rebase: You have unstaged changes.
error: Please commit or stash them.
```

Same as the checkout error. Commit, stash, or discard before rebasing.

### `fatal: bad object`

```
$ git show abc123
fatal: bad object abc123
```

The hash you gave doesn't refer to any object in the repository. Either it's a typo, or you're in a different repo than you thought, or the object was garbage-collected.

### `error: object file is empty`

```
$ git status
error: object file .git/objects/ab/c123def... is empty
fatal: loose object abc123def... (stored in .git/objects/ab/c123def...) is corrupt
```

A Git object on disk got corrupted. Causes: filesystem error, killed Git mid-write, disk full at the wrong moment. Try `git fsck --full`. Often the fix is to clone a fresh copy from the remote and copy in your local-only branches.

### `Permission denied (publickey)`

```
$ git push
git@github.com: Permission denied (publickey).
fatal: Could not read from remote repository.
```

Your SSH key isn't set up correctly with the remote. Check `ssh -T git@github.com` to test. Generate a key with `ssh-keygen -t ed25519`, add the public key to GitHub/GitLab in Settings > SSH Keys, ensure `ssh-add ~/.ssh/id_ed25519` is run.

### `fatal: Authentication failed for 'https://...'`

```
$ git push
remote: Support for password authentication was removed on August 13, 2021.
fatal: Authentication failed for 'https://github.com/user/repo.git/'
```

GitHub no longer accepts passwords for HTTPS. Either switch to SSH (`git remote set-url origin git@github.com:user/repo.git`) or use a Personal Access Token instead of a password.

### `error: GH001: Large files detected`

```
remote: error: GH001: Large files detected. You may want to try Git Large File Storage
remote: error: File big.bin is 150.00 MB; this exceeds GitHub's file size limit of 100.00 MB
```

GitHub rejects pushes containing files over 100 MB. Solutions: don't commit huge binaries (use a real artifact store), use Git LFS for large files, or remove the file from history with `git filter-repo`.

### `fatal: detected dubious ownership in repository`

```
$ git status
fatal: detected dubious ownership in repository at '/path/to/repo'
To add an exception for this directory, call:
    git config --global --add safe.directory /path/to/repo
```

Git detected the repo is owned by a different user than the one running Git. This is a security feature. If the repo really is yours and the warning is wrong, run the suggested command.

## Hands-On

Set up a sandbox and run real commands. Each one shows what to type, what happens, and what to expect.

### Initialize a new repo

```
$ mkdir hello-git && cd hello-git
$ git init
Initialized empty Git repository in /tmp/hello-git/.git/
```

A `.git/` directory was created. That's the time machine. Don't touch its contents directly.

### Check status

```
$ git status
On branch main

No commits yet

nothing to commit (create/copy files and use "git add" to track)
```

### Create and stage a file

```
$ echo "Hello, Git!" > greeting.txt
$ git status
On branch main

No commits yet

Untracked files:
  (use "git add <file>..." to include in what will be committed)
        greeting.txt

nothing added to commit but untracked files present (use "git add" to track)
$ git add greeting.txt
$ git status
On branch main

No commits yet

Changes to be committed:
  (use "git rm --cached <file>..." to unstage)
        new file:   greeting.txt
```

### Status in short form

```
$ echo "another line" >> greeting.txt
$ git status -s
A  greeting.txt
AM greeting.txt
```

`A` = added (staged). `M` = modified (unstaged). The two-letter code shows index status, then working tree status.

### Make the first commit

```
$ git add greeting.txt
$ git commit -m "First commit"
[main (root-commit) a1b2c3d] First commit
 1 file changed, 2 insertions(+)
 create mode 100644 greeting.txt
```

The hash `a1b2c3d` will be different on your machine — content-addressed, but timestamps and your author info change the hash.

### View the log

```
$ git log
commit a1b2c3d4e5f6789... (HEAD -> main)
Author: Stevie <stevie@bellis.tech>
Date:   Mon Apr 27 12:00:00 2026 -0400

    First commit
```

### Compact log

```
$ git log --oneline -10
a1b2c3d (HEAD -> main) First commit
```

The `-10` means "last 10 commits." Without it, `git log` would page through all of them.

### Make a change and view the diff

```
$ echo "third line" >> greeting.txt
$ git diff
diff --git a/greeting.txt b/greeting.txt
index 1234567..89abcde 100644
--- a/greeting.txt
+++ b/greeting.txt
@@ -1,2 +1,3 @@
 Hello, Git!
 another line
+third line
```

Lines starting with `+` are additions. Lines starting with `-` would be deletions. The `@@ -1,2 +1,3 @@` header says "in the original file, starting at line 1, 2 lines; in the new file, starting at line 1, 3 lines."

### Stage and view staged diff

```
$ git add greeting.txt
$ git diff
$ git diff --cached
diff --git a/greeting.txt b/greeting.txt
index 1234567..89abcde 100644
--- a/greeting.txt
+++ b/greeting.txt
@@ -1,2 +1,3 @@
 Hello, Git!
 another line
+third line
```

Plain `git diff` shows nothing now (working dir matches index). `git diff --cached` shows the staged change.

### Unstage with restore

```
$ git restore --staged greeting.txt
$ git status
On branch main
Changes not staged for commit:
        modified:   greeting.txt
```

The change is back in the unstaged area. Notice we used the modern `git restore` instead of the old `git reset HEAD`.

### Discard working changes

```
$ git restore greeting.txt
$ git status
On branch main
nothing to commit, working tree clean
$ cat greeting.txt
Hello, Git!
another line
```

The working file is back to what was committed. The "third line" addition is gone forever (it wasn't committed, so the reflog can't save you).

### Commit again

```
$ echo "third line" >> greeting.txt
$ git add . && git commit -m "Add third line"
[main 5f6g7h8] Add third line
 1 file changed, 1 insertion(+)
```

### Commit with --amend

```
$ git commit --amend -m "Add a third line"
[main 9i0j1k2] Add a third line
 Date: Mon Apr 27 12:01:00 2026 -0400
 1 file changed, 1 insertion(+)
```

The previous commit was rewritten with a new message. The hash changed (`5f6g7h8` is now `9i0j1k2`). The old commit is in the reflog.

### Amend without changing the message

```
$ echo "more text" >> greeting.txt
$ git add . && git commit --amend --no-edit
[main 3l4m5n6] Add a third line
 Date: Mon Apr 27 12:02:00 2026 -0400
 1 file changed, 2 insertions(+)
```

The new change was folded into the previous commit, keeping the same message. Useful for "oh, I forgot one tiny thing in my last commit."

### Compare commits

```
$ git diff HEAD~1 HEAD
diff --git a/greeting.txt b/greeting.txt
index 89abcde..fedcba9 100644
--- a/greeting.txt
+++ b/greeting.txt
@@ -1,2 +1,4 @@
 Hello, Git!
 another line
+third line
+more text
```

`HEAD~1` means "one commit before HEAD." `HEAD~2` means two before. `HEAD^` is the same as `HEAD~1`.

### Show a commit

```
$ git show HEAD
commit 3l4m5n6...
Author: Stevie <stevie@bellis.tech>
Date:   Mon Apr 27 12:02:00 2026 -0400

    Add a third line

diff --git a/greeting.txt b/greeting.txt
...
```

### Blame a file

```
$ git blame greeting.txt
^a1b2c3d (Stevie 2026-04-27 12:00:00 -0400 1) Hello, Git!
^a1b2c3d (Stevie 2026-04-27 12:00:00 -0400 2) another line
3l4m5n6  (Stevie 2026-04-27 12:02:00 -0400 3) third line
3l4m5n6  (Stevie 2026-04-27 12:02:00 -0400 4) more text
```

Each line is annotated with the commit, author, date that last touched it. Great for "who added this code and why?" investigations.

### Search history with pickaxe

```
$ git log -S 'third line'
commit 3l4m5n6...
Author: Stevie <stevie@bellis.tech>
Date:   Mon Apr 27 12:02:00 2026 -0400

    Add a third line
```

`-S` (the "pickaxe") finds commits that added or removed the given string. `-G` is the regex variant.

### List branches

```
$ git branch
* main
```

The `*` marks the current branch. With no other branches yet, just `main`.

### Create and switch to a branch

```
$ git switch -c feature/uppercase
Switched to a new branch 'feature/uppercase'
$ git branch
* feature/uppercase
  main
```

### Make a commit on the new branch

```
$ tr 'a-z' 'A-Z' < greeting.txt > greeting.UPPER.txt
$ mv greeting.UPPER.txt greeting.txt
$ git add greeting.txt
$ git commit -m "Uppercase the greeting"
[feature/uppercase abc1234] Uppercase the greeting
 1 file changed, 4 insertions(+), 4 deletions(-)
```

### View graph of all branches

```
$ git log --graph --oneline --all
* abc1234 (HEAD -> feature/uppercase) Uppercase the greeting
* 3l4m5n6 (main) Add a third line
* a1b2c3d First commit
```

### Switch back to main

```
$ git switch main
Switched to branch 'main'
$ cat greeting.txt
Hello, Git!
another line
third line
more text
```

The file is back to lowercase — you switched timelines. The uppercase version still exists on `feature/uppercase`.

### Merge

```
$ git merge feature/uppercase
Updating 3l4m5n6..abc1234
Fast-forward
 greeting.txt | 8 ++++----
 1 file changed, 4 insertions(+), 4 deletions(-)
```

Fast-forward, because `main` had not changed. Now `main` includes the uppercase commit.

### Cherry-pick (after creating another branch)

```
$ git switch -c feature/extra
$ echo "EXTRA LINE" >> greeting.txt
$ git add . && git commit -m "Add extra line"
[feature/extra def5678] Add extra line
$ git switch main
$ git cherry-pick def5678
[main 9z8y7x6] Add extra line
 Date: Mon Apr 27 12:10:00 2026 -0400
 1 file changed, 1 insertion(+)
```

Same content, new hash on `main`.

### Stash

```
$ echo "uncommitted" >> greeting.txt
$ git stash
Saved working directory and index state WIP on main: 9z8y7x6 Add extra line
$ git status
On branch main
nothing to commit, working tree clean
$ git stash list
stash@{0}: WIP on main: 9z8y7x6 Add extra line
$ git stash pop
On branch main
Changes not staged for commit:
        modified:   greeting.txt
Dropped refs/stash@{0} (5p4o3i2)
```

### Tag

```
$ git tag v1.0.0
$ git tag -a v1.1.0 -m 'First minor release'
$ git tag
v1.0.0
v1.1.0
$ git show v1.1.0
tag v1.1.0
Tagger: Stevie <stevie@bellis.tech>
Date:   Mon Apr 27 12:15:00 2026 -0400

First minor release

commit 9z8y7x6...
...
```

### Reset (soft)

```
$ echo "junk" >> greeting.txt
$ git add . && git commit -m "junk commit"
[main 1q2w3e4] junk commit
$ git reset HEAD~1
$ git log --oneline -3
9z8y7x6 (HEAD -> main, tag: v1.0.0) Add extra line
3l4m5n6 Add a third line
a1b2c3d First commit
$ git status
Changes not staged for commit:
        modified:   greeting.txt
```

`git reset HEAD~1` moved HEAD back one commit, kept the changes in the working directory. The "junk commit" is gone but the changes are still there to refine.

### Revert

```
$ git revert HEAD
[main 5r6t7y8] Revert "Add extra line"
 1 file changed, 1 deletion(-)
```

`revert` doesn't rewrite history — it creates a NEW commit that undoes a previous one. Safe to use on shared branches.

### Reflog after experiments

```
$ git reflog
5r6t7y8 (HEAD -> main) HEAD@{0}: revert: Revert "Add extra line"
9z8y7x6 (tag: v1.0.0) HEAD@{1}: reset: moving to HEAD~1
1q2w3e4 HEAD@{2}: commit: junk commit
9z8y7x6 (tag: v1.0.0) HEAD@{3}: cherry-pick: Add extra line
def5678 HEAD@{4}: checkout: moving from feature/extra to main
...
```

Every move is recorded.

### Worktree

```
$ git worktree add ../hello-git-feature feature/uppercase
Preparing worktree (checking out 'feature/uppercase')
HEAD is now at abc1234 Uppercase the greeting
$ git worktree list
/tmp/hello-git              5r6t7y8 [main]
/tmp/hello-git-feature      abc1234 [feature/uppercase]
```

### Clean

```
$ touch garbage.tmp
$ git status
On branch main
Untracked files:
        garbage.tmp
$ git clean -fd
Removing garbage.tmp
```

`-f` is required (clean refuses without it as a safety check). `-d` includes directories. Be careful — this deletes files immediately.

### rm and mv

```
$ git mv greeting.txt hello.txt
$ git status
        renamed:    greeting.txt -> hello.txt
$ git rm hello.txt
rm 'hello.txt'
$ git restore --staged --worktree hello.txt
```

### Config

```
$ git config --global user.name "Stevie"
$ git config --global user.email "stevie@bellis.tech"
$ git config --global core.editor nvim
$ git config --list
user.name=Stevie
user.email=stevie@bellis.tech
core.editor=nvim
...
```

### Add a remote

```
$ git remote add origin git@github.com:bellistech/hello-git.git
$ git remote -v
origin  git@github.com:bellistech/hello-git.git (fetch)
origin  git@github.com:bellistech/hello-git.git (push)
```

### Push

```
$ git push -u origin main
Enumerating objects: 12, done.
Counting objects: 100% (12/12), done.
Writing objects: 100% (12/12), 1.04 KiB | 1.04 MiB/s, done.
Total 12 (delta 4), reused 0 (delta 0)
To github.com:bellistech/hello-git.git
 * [new branch]      main -> main
branch 'main' set up to track 'origin/main' from 'remote'.
```

### Push tags

```
$ git push origin --tags
Total 0 (delta 0), reused 0 (delta 0)
To github.com:bellistech/hello-git.git
 * [new tag]         v1.0.0 -> v1.0.0
 * [new tag]         v1.1.0 -> v1.1.0
```

### Fetch and pull

```
$ git fetch
remote: Counting objects: 5, done.
...
From github.com:bellistech/hello-git
   5r6t7y8..abc9876  main       -> origin/main
$ git pull --rebase
Successfully rebased and updated refs/heads/main.
```

### Bisect

```
$ git bisect start
$ git bisect bad HEAD
$ git bisect good v1.0.0
Bisecting: 1 revision left to test after this (roughly 1 step)
[3l4m5n6] Add a third line
$ git bisect good
9z8y7x6 is the first bad commit
$ git bisect reset
```

## Common Confusions

### "Why didn't `git push` push my new branch?"

```
# WRONG
$ git push
fatal: The current branch feature-x has no upstream branch.
```

```
# RIGHT
$ git push -u origin feature-x
```

The `-u` (`--set-upstream`) tells Git to track the remote branch. After that, plain `git push` works.

### "Why did `git checkout file.txt` destroy my changes?"

```
# WRONG (no warning, no recovery)
$ git checkout file.txt
```

```
# RIGHT (modern, clearer)
$ git restore file.txt
```

`git checkout file.txt` discards your uncommitted changes to that file with no warning and no way to recover (the working tree changes were never in any object storage). Use `git restore` for the same operation in modern Git — same effect, but the verb actually says what it does.

### "I committed to `main` but I meant to be on a feature branch."

```
# WRONG
$ git switch -c feature-x
# (the commit is still on main, branch is created from current HEAD)
```

```
# RIGHT
$ git switch -c feature-x      # branch off here, takes the commit with you
$ git switch main
$ git reset --hard HEAD~1      # remove the commit from main
```

The trick: a new branch is created from wherever HEAD is, including any committed work. Move yourself to the new branch (which has the commit), then go back to main and remove it.

### "I want to undo my last commit, keeping my changes."

```
# WRONG (loses changes)
$ git reset --hard HEAD~1
```

```
# RIGHT (keeps changes in working dir)
$ git reset HEAD~1
```

`reset` defaults to `--mixed`, which keeps the changes in your working directory. `--hard` throws them away. Only use `--hard` when you really want destruction.

### "I want to undo a commit on a shared branch."

```
# WRONG (rewrites history, breaks teammates)
$ git reset --hard HEAD~1
$ git push --force
```

```
# RIGHT (creates a NEW commit that undoes the old one)
$ git revert HEAD
$ git push
```

`revert` is the safe way to undo on a shared branch. It creates a new commit instead of rewriting.

### "I want to merge my feature into main."

```
# WRONG (you're on feature-x and you merge main into feature-x)
$ git switch feature-x
$ git merge main
```

```
# RIGHT
$ git switch main
$ git merge feature-x
```

You merge INTO the current branch FROM another branch. Switch to the destination first.

### "I rebased and now my history is duplicated."

```
# Before rebase: you have 3 local commits, just rebased onto updated main
$ git log --oneline
A' B' C' main_new D E F
```

```
# WRONG (now you have BOTH old and new commits)
$ git pull
```

```
# RIGHT (force push your rebased branch, since it's only yours)
$ git push --force-with-lease
```

After rebasing, your local commits have new hashes. `git pull` will try to merge the old (remote) and new (local) versions, doubling everything. Force-push your rebase. Use `--force-with-lease` (safer than `--force`) so it refuses if someone else pushed since your last fetch.

### "git stash pop produced a conflict and now my stash is gone."

Actually, `pop` keeps the stash if there was a conflict — only successful pops drop it. Look at `git stash list`:

```
$ git stash list
stash@{0}: WIP on main: ...
```

It's still there. Resolve the conflict, `git add`, then `git stash drop` to remove it.

### "I cloned and there's no `main` branch."

```
$ git branch
$ git branch -r
  origin/HEAD -> origin/main
  origin/main
```

Local `main` only appears after you check it out. Modern clones do this automatically; if not:

```
$ git switch main
```

### "I want to throw away every change since my last commit."

```
# Remove unstaged changes
$ git restore .

# Also remove staged changes
$ git restore --staged . && git restore .

# Also remove untracked files
$ git clean -fd
```

Three operations — three trees, three commands.

### "I made commits on detached HEAD and lost them."

```
# WRONG (they're orphaned, will be GC'd in 90 days)
$ git switch main
```

```
# RIGHT (rescue them BEFORE switching)
$ git switch -c rescued-work
# Now they're on a real branch
```

If you already switched away, the reflog still has the hashes:

```
$ git reflog
abc1234 HEAD@{2}: commit: my work
$ git branch rescued-work abc1234
```

### ".gitignore isn't ignoring a file I committed."

```
# WRONG: just adding to .gitignore
$ echo "secret.txt" >> .gitignore
```

```
# RIGHT: untrack it first
$ git rm --cached secret.txt
$ echo "secret.txt" >> .gitignore
$ git add .gitignore
$ git commit -m "Ignore secret.txt"
```

`.gitignore` only prevents *new* files from being tracked. Already-tracked files are still tracked. You have to `git rm --cached` (which removes from the index but keeps the working file) to untrack.

### "I want to find when a file was deleted."

```
# WRONG (file isn't there, so log shows nothing)
$ git log path/to/deleted-file.txt
```

```
# RIGHT
$ git log --all --full-history -- path/to/deleted-file.txt
```

Or to find the deletion specifically:

```
$ git log --diff-filter=D -- path/to/deleted-file.txt
```

## Vocabulary

The list of terms you will run into. Each one has a one-line plain-English definition.

| Term | Plain English |
|------|---------------|
| git | The tool. The time machine. |
| repository (repo) | A project tracked by Git. Includes a `.git/` directory. |
| bare repo | A repo with no working directory, just the `.git/` data. Used as a server-side hub. |
| working directory | Your actual files on disk that you edit. |
| index | The "next commit draft" buffer. Same as staging area. |
| staging area | The "next commit draft" buffer. Same as index. |
| HEAD | A pointer to "where you are right now," usually the latest commit on the current branch. |
| ORIG_HEAD | A pointer to where HEAD was before a dangerous operation (reset, merge). |
| FETCH_HEAD | A pointer to whatever was last fetched. |
| MERGE_HEAD | The other branch's tip during an in-progress merge. |
| branch | A moveable name pointing at a commit. Used for parallel timelines. |
| tracking branch | A local branch that knows which remote branch it should push to and pull from. |
| remote-tracking branch | A local cache of where a remote branch was at last fetch (e.g., `origin/main`). |
| upstream | The remote branch your local branch tracks; or, a project you forked from. |
| downstream | A fork or derivative of an upstream. |
| ahead | Your branch has commits the remote doesn't. |
| behind | The remote has commits your branch doesn't. |
| diverged | Both your branch and the remote have commits the other doesn't. |
| fast-forward | A merge where the target branch can simply jump forward; no merge commit needed. |
| three-way merge | A merge that uses the common ancestor + both tips to produce a result. |
| recursive merge | The default merge strategy in Git; recursive in tricky cases involving criss-cross merges. |
| octopus merge | A merge of three or more branches simultaneously. |
| merge commit | A commit with two or more parents, produced by merging branches. |
| merge conflict | When a merge can't auto-resolve because both sides changed the same lines. |
| rebase | Replay one branch's commits onto another. Rewrites history. |
| interactive rebase | Rebase with a menu, letting you reorder, squash, drop, edit, reword commits. |
| fixup | An interactive-rebase action: combine this commit into the previous one and discard its message. |
| squash | An interactive-rebase action: combine this commit into the previous one and combine messages. |
| drop | An interactive-rebase action: throw this commit away. |
| reword | An interactive-rebase action: change just the commit message. |
| edit | An interactive-rebase action: stop here so I can amend more changes. |
| exec | An interactive-rebase action: run a shell command at this point. |
| break | An interactive-rebase action: pause the rebase here. |
| label | An interactive-rebase action: name the current spot for later reference. |
| reset | Move HEAD (and optionally the index/working tree) to a different commit. |
| autosquash | Auto-mark commits prefixed `fixup!` or `squash!` for fixup/squash during interactive rebase. |
| cherry-pick | Copy a single commit from another branch to the current branch. |
| revert | Create a new commit that undoes a previous commit. |
| reset --soft | Move HEAD only; keep index and working tree as-is. |
| reset --mixed | Default reset: move HEAD and index; keep working tree. |
| reset --hard | Move HEAD, index, and working tree. Destructive. |
| reset --keep | Move HEAD; keep local changes if they don't conflict with the move. |
| reflog | A log of every HEAD movement on your local machine. Your safety net. |
| stash | A temporary parking lot for uncommitted changes. |
| tag | A name for a specific commit. Used for releases. |
| lightweight tag | A tag that's just a name pointing at a commit. |
| annotated tag | A tag stored as a full Git object with author, message, and signature. |
| GPG-signed commit | A commit cryptographically signed with a GPG key, proving authorship. |
| commit | A snapshot of the project at a moment, with metadata and a hash. |
| commit message | The human-readable description attached to a commit. |
| conventional commits | A commit message convention with types like feat:, fix:, chore:, docs:. |
| signed-off-by | A line in a commit message claiming you have rights to contribute the change. |
| tree | A snapshot of files and folders. The "directory" in Git's object database. |
| blob | A snapshot of a single file's contents. |
| tag object | The Git object that backs an annotated tag. |
| packfile | A compressed file containing many objects, used to save disk and bandwidth. |
| loose object | An individual object stored as its own file in `.git/objects/`. |
| gc (garbage collection) | Cleanup that packs loose objects and prunes unreachable ones. |
| pruning | Removing unreachable objects from the database. |
| expire | The reflog policy that decides when old reflog entries are eligible for pruning. |
| refs/ | The directory inside .git that holds all references (branches, tags, etc.). |
| refs/heads/ | Branches. |
| refs/remotes/ | Remote-tracking branches. |
| refs/tags/ | Tags. |
| refs/stash | The stash reference. |
| packed-refs | A single file storing many refs in compact form. |
| .git/objects | Where Git stores all objects (commits, trees, blobs, tags). |
| .git/HEAD | The file that says where HEAD is. |
| .git/config | The repo-local config file. |
| .gitignore | A file listing patterns Git should ignore. |
| .gitattributes | A file specifying per-path attributes (line endings, merge strategy, etc.). |
| .gitmodules | A file recording submodule definitions. |
| .mailmap | A file mapping author identities (canonicalize who is who in `git log`). |
| hooks | Scripts run at specific Git events. |
| pre-commit | Hook run before a commit is finalized. |
| commit-msg | Hook run after the commit message is written. |
| prepare-commit-msg | Hook run before the commit message editor opens. |
| pre-push | Hook run before `git push` actually pushes. |
| post-merge | Hook run after a merge completes. |
| pre-receive | Server-side hook run before any refs are updated by a push. |
| update | Server-side hook run once per ref during a push. |
| post-receive | Server-side hook run after a push is accepted. |
| git lfs | Git Large File Storage, an extension for handling huge files outside Git proper. |
| large file storage | Same as Git LFS. |
| submodule | A reference to a specific commit of another Git repo, embedded as a subdirectory. |
| subtree | Another repo's content merged into your repo as a subdirectory. |
| worktree | A second working directory attached to the same repo. |
| sparse-checkout | Check out only some files from the repo to save disk space. |
| partial clone | Clone without all blob data; fetch on demand. |
| shallow clone | Clone with limited history depth. |
| --depth | Flag to control shallow clone depth. |
| --filter=blob:none | Flag to exclude blob data from the initial clone. |
| blame | Show which commit last touched each line of a file. |
| log | Show the commit history. |
| show | Show the contents of a commit, tree, blob, or tag. |
| diff | Show changes between two states. |
| log -S | Search history for commits that added or removed a string ("pickaxe"). |
| log -G | Search history for commits whose diff matches a regex. |
| log -p | Show patches with each commit in `git log`. |
| format-patch | Generate `.patch` files from a range of commits, ready to email. |
| am (apply mailbox) | Apply a mailbox of patches as commits. |
| apply | Apply a `.patch` file without committing. |
| send-email | Send commits as emails (used by the kernel community). |
| request-pull | Generate a pull request summary, by email. |
| fork | A copy of a repo on a hosting service, owned by you. |
| pull request (PR) | GitHub/GitLab/Gitea construct: a request to merge a branch with discussion and review. |
| merge request (MR) | GitLab's name for a pull request. |
| code review | A teammate reading your changes before they're merged. |
| fast-forward only | A policy that disallows merge commits, requiring rebase or fast-forward. |
| allow-merge-commits | A policy that permits merge commits in PRs. |
| squash-merge | A merge strategy where all PR commits become one new commit. |
| semantic versioning | A version-numbering convention: MAJOR.MINOR.PATCH. |
| signed tag | An annotated tag with a GPG/SSH signature. |
| signed commit | A commit with a GPG/SSH signature. |
| GPG | The OpenPGP implementation typically used to sign commits and tags. |
| SSH signing | Using SSH keys (instead of GPG) to sign commits and tags. |
| X.509 | A certificate-based signing scheme also supported by Git. |
| sigstore | A free, automated signing infrastructure now usable with Git. |
| attestation | A signed claim about a commit (e.g., "this passed CI"). |
| gitmoji | A convention of putting emojis in commit messages to indicate type. |
| .gitkeep | A convention: an empty file used to force Git to track an otherwise-empty directory. |
| origin | The default name for the remote a repo was cloned from. |
| HEAD~N | The N-th ancestor of HEAD (HEAD~1 = parent, HEAD~2 = grandparent). |
| HEAD^ | Same as HEAD~1 in most cases. |
| HEAD^2 | The second parent of HEAD (only meaningful for merge commits). |
| range A..B | Commits in B but not in A. |
| range A...B | Commits in either A or B but not both (symmetric difference). |
| pickaxe | Slang for `git log -S`. |
| dangling | An object reachable from no ref. Eligible for garbage collection. |
| pack | A bundle of objects in compressed form. |
| index file | The binary file at .git/index that holds the staging area. |
| commit graph | A pre-computed cache of the commit graph for fast traversal. |

## Try This

Real experiments. Set up a sandbox, run them, see what happens.

### Experiment 1: Watch a hash change with content

```
$ mkdir hash-test && cd hash-test && git init
$ echo "hello" > a.txt
$ git add a.txt
$ git commit -m "Add a.txt"
$ git rev-parse HEAD          # note this hash
$ git commit --amend -m "Add a.txt"   # same content, same message, but reword
$ git rev-parse HEAD          # different hash!
```

The hash changed even though the content didn't, because the commit's date metadata changed. Hashes depend on every byte of the commit, including timestamps.

### Experiment 2: Recover a "deleted" branch

```
$ git switch -c temp
$ echo "important" > work.txt
$ git add . && git commit -m "Important work"
$ HASH=$(git rev-parse HEAD)
$ git switch main
$ git branch -D temp          # branch is "gone"
$ git reflog | grep $HASH
$ git branch recovered $HASH
$ git switch recovered
$ ls work.txt                 # still there
```

The reflog kept the commit alive even after the branch was deleted.

### Experiment 3: See three-way merge in action

```
$ mkdir three-way && cd three-way && git init
$ echo "common" > file.txt && git add . && git commit -m "Common"
$ git switch -c branch1
$ echo "branch1 change" >> file.txt && git add . && git commit -m "B1"
$ git switch main
$ git switch -c branch2
$ echo "branch2 change" >> file.txt && git add . && git commit -m "B2"
$ git switch main
$ git merge branch1
$ git merge branch2
# CONFLICT! Both branches changed the same end of the file.
$ cat file.txt
common
<<<<<<< HEAD
branch1 change
=======
branch2 change
>>>>>>> branch2
```

Now you can see what a real conflict looks like, and resolve it by hand.

### Experiment 4: Compare merge vs rebase visually

```
$ mkdir compare && cd compare && git init
$ echo a > x && git add . && git commit -m "A"
$ echo b >> x && git add . && git commit -m "B"
$ git switch -c feature
$ echo c >> x && git add . && git commit -m "C"
$ git switch main
$ echo d >> x && git add . && git commit -m "D"
# Two divergent timelines. First, try merge:
$ git merge feature   # may conflict; resolve and commit
$ git log --graph --oneline --all
# Now reset and try rebase:
$ git reset --hard HEAD~1
$ git switch feature
$ git rebase main     # may conflict; resolve and continue
$ git log --graph --oneline --all
```

The two graphs look very different. Merge produces a diamond. Rebase produces a straight line.

### Experiment 5: Try git bisect on a planted bug

```
$ mkdir bisect-test && cd bisect-test && git init
$ for i in 1 2 3 4 5 6 7 8 9 10; do
    echo "version $i" > version.txt
    if [ $i -eq 7 ]; then echo "BUG" >> version.txt; fi
    git add . && git commit -m "Version $i"
  done
$ git bisect start
$ git bisect bad HEAD
$ git bisect good HEAD~9
# Git checks out a middle commit; you check for "BUG"
$ grep -q BUG version.txt && echo "bad" || echo "good"
$ git bisect bad   # or good, based on what you saw
# Continue until Git names the first bad commit
$ git bisect reset
```

You'll find commit 7 (the one labeled "Version 7") is the bad one — Git binary-searches and lands there.

### Experiment 6: Inspect Git's object database

```
$ mkdir objects-test && cd objects-test && git init
$ echo "hello" > greet.txt && git add greet.txt
$ git commit -m "First"
$ ls .git/objects/
$ find .git/objects -type f
$ HASH=$(git rev-parse HEAD)
$ git cat-file -t $HASH        # commit
$ git cat-file -p $HASH        # show the commit object
$ TREE=$(git cat-file -p $HASH | grep tree | cut -c6-)
$ git cat-file -p $TREE        # show the tree object
$ BLOB=$(git cat-file -p $TREE | grep greet | awk '{print $3}')
$ git cat-file -p $BLOB        # show the blob (file contents)
hello
```

You just walked Git's pointer graph by hand. Commit → tree → blob.

### Experiment 7: Stash multiple times

```
$ echo "change 1" >> file.txt
$ git stash push -m "first WIP"
$ echo "change 2" >> file.txt
$ git stash push -m "second WIP"
$ git stash list
stash@{0}: On main: second WIP
stash@{1}: On main: first WIP
$ git stash apply stash@{1}    # apply the older one
$ git stash list               # both still there
$ git stash drop stash@{0}
```

### Experiment 8: Worktree with two branches at once

```
$ git switch main
$ git worktree add ../sandbox feature-x
$ cd ../sandbox
$ ls   # files are at the feature-x state
$ cd ../original
$ ls   # files are at the main state
$ git worktree list
```

Two working directories, one repo. Edit both at once.

## Where to Go Next

- `cs vcs git` — dense reference for daily commands
- `cs vcs git-worktree` — worktree-specific deep dive
- `cs detail vcs/git` — internals: object store, packfiles, refs
- `cs ramp-up github-actions-eli5` — what runs in CI after you push
- `cs quality pre-commit` — a framework for managing pre-commit hooks across the team
- `cs quality code-review` — how to give and receive code review well

## See Also

- `vcs/git`
- `vcs/git-worktree`
- `ci-cd/github-actions`
- `ci-cd/gitlab-ci`
- `quality/pre-commit`
- `ramp-up/github-actions-eli5`
- `ramp-up/bash-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- git-scm.com/book — Pro Git, free official book
- "Git Internals" — Bitbucket's deep dive into the object model
- man git, man git-rebase, man git-config, man gitrevisions
- gitignore.io — community-maintained `.gitignore` templates
- conventionalcommits.org — Conventional Commits specification
- semver.org — Semantic Versioning 2.0.0 specification
