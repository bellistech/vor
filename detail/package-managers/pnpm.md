# pnpm — Internals & Theory

pnpm (performant npm) is a JavaScript package manager designed around a single architectural insight: *content-addressable storage*. Instead of copying packages into each project's `node_modules`, pnpm maintains a global store keyed by content hash and uses hardlinks/symlinks to make those stored files appear in each project's tree. The result is dramatic savings in disk space, faster installs, and stricter module resolution that catches a class of "phantom dependency" bugs that npm and Yarn-classic silently allowed.

This document goes deep on how pnpm achieves this — the storage model, the linking strategy, the lockfile format, the resolver, and the workspace/filtering features built on top.

## Setup

pnpm was created by Zoltan Kochan in 2017. The motivation was direct: the author was tired of running `npm install` on a 50-project monorepo and watching the disk fill with hundreds of identical copies of `lodash`. The first version was a thin wrapper around npm that replaced the install step with hardlinks from a shared store.

Subsequent versions diverged significantly: a custom resolver, a custom installer, the `.pnpm/` virtual store layout (Section 3), workspace support, filtering, patches, and a comprehensive set of CLI commands. By 2024, pnpm was the third-most-used Node package manager (after npm and Yarn), with notable adopters including Vue, Vite, Vercel, and Microsoft's monorepo tooling.

pnpm installs as a single Node.js script (or as a self-contained binary via `corepack` or the official installer). The CLI surface mirrors npm's where reasonable:

```bash
pnpm install                  # install per package.json + lockfile
pnpm add lodash               # add dependency
pnpm add -D vitest            # add devDependency
pnpm add -O webpack           # add optionalDependency
pnpm remove lodash            # remove dependency
pnpm update                   # bump versions per ranges
pnpm outdated                 # show outdated deps
pnpm ls                       # tree view
pnpm dlx create-vite          # ephemeral execute (like npx)
pnpm exec eslint              # exec from local node_modules/.bin
pnpm run test                 # run a script from package.json
pnpm test                     # shorthand (well-known scripts)
pnpm publish                  # publish to npm registry
pnpm patch lodash             # create local patch
pnpm patch-commit             # finalize patch
pnpm import                   # convert npm/yarn lockfile → pnpm-lock.yaml
```

The default config lives in `~/.npmrc` (yes, .npmrc — pnpm reads npm-compatible config), with pnpm-specific keys layered on top. Per-project config is in `.npmrc` at the project root.

## Hardlinks + Symlinks

The architectural heart of pnpm is the distinction between the *store* and the *node_modules tree*.

**The store** lives at `~/.local/share/pnpm/store/v3/` (Linux), `~/Library/pnpm/store/v3/` (macOS), or similar on Windows. Inside, files are organized content-addressably:

```
~/.local/share/pnpm/store/v3/
├── files/
│   ├── 00/
│   │   ├── 00abc...d (file content)
│   │   └── 00xyz...e
│   ├── 01/
│   ├── 02/
│   ...
│   └── ff/
└── index/
    └── (per-tarball index files mapping package version → file hashes)
```

When pnpm fetches a package's tarball from the registry, it:

1. Verifies the integrity hash matches the registry-supplied SHA-512 (or SHA-1 for very old packages).
2. Extracts each file individually and computes its SHA-512.
3. Stores each file under `files/<first-2-hex-chars>/<remaining-62-chars>` — keyed by content hash.
4. Records the package's file manifest (package version → list of (relative-path, hash) pairs) in `index/`.

The fragmentation into `files/<NN>/<rest>` exists because some filesystems don't perform well with hundreds of thousands of files in a single directory. Two-character prefix sharding keeps each directory under a few thousand entries.

**The node_modules tree** uses hardlinks to the store. When pnpm installs `lodash@4.17.21` for a project, it doesn't copy the files; it creates hardlinks:

```
project/node_modules/.pnpm/lodash@4.17.21/node_modules/lodash/
├── lodash.js   →  hardlink to store/v3/files/2a/...
├── package.json → hardlink to store/v3/files/9b/...
└── ... (all hardlinked)
```

Hardlinks share inodes — `cp` style copies create new inodes, but hardlinks make multiple paths refer to the same on-disk data. Disk usage of N projects depending on the same `lodash@4.17.21` is the size of `lodash@4.17.21` *once*, regardless of how many projects use it.

**Cross-volume fallback.** Hardlinks only work within a single filesystem. If your store is on `/home` (one filesystem) and your project is on `/mnt/external` (a different filesystem), hardlinks cannot bridge the gap. pnpm detects this at install time and falls back to:

1. **Reflinks** (CoW copy) — on filesystems supporting copy-on-write semantics (XFS with `reflink=1`, btrfs, APFS), pnpm uses `clonefile()`/`copy_file_range()` with the reflink hint. The copy is initially shared on disk; only modified blocks diverge.
2. **Symlinks** — pointers to the store files. Works across filesystems but breaks tools that don't follow symlinks (rare, but real).
3. **Copy** — the last-resort fallback. Costs disk and time but always works.

The choice is configurable via `package-import-method` in `.npmrc`:

```
package-import-method=auto         # default: hardlink, reflink, copy in order
package-import-method=hardlink     # force hardlink, fail if cross-volume
package-import-method=clone        # force reflink (CoW)
package-import-method=copy         # always copy
```

Most users want `auto`. The performance difference is significant: hardlinks are nearly free (just a directory entry); reflinks involve a copy_file_range call but no data copy; full copies hit the disk for every byte.

**Why hardlinks and not symlinks for the store?** Two reasons.

First, *editor safety*. If you accidentally `rm -rf node_modules`, hardlinks remove only the directory entry — the store file remains intact. Symlinks would point to a deleted target after store cleanup. Hardlinks degrade gracefully.

Second, *portability*. Symlinks have OS-specific semantics (Windows treats them differently from Unix; some build tools follow them, some don't). Hardlinks behave like ordinary files everywhere — the kernel resolves them at open() and the rest of the system never knows.

## .pnpm/ Virtual Store

The `node_modules/.pnpm/` directory is pnpm's most distinctive structural feature. It's where the actual package contents (well, the hardlinks to the store) live, in a flat layout per package version:

```
node_modules/
├── .pnpm/
│   ├── lodash@4.17.21/
│   │   └── node_modules/
│   │       └── lodash/                # actual package files (hardlinked)
│   ├── react@18.2.0_/
│   │   └── node_modules/
│   │       └── react/                 # actual package files
│   ├── react-dom@18.2.0_react@18.2.0/
│   │   └── node_modules/
│   │       ├── react-dom/             # actual package files
│   │       └── react/                 # SYMLINK to ../../../react@18.2.0/node_modules/react/
│   └── ...
├── lodash → .pnpm/lodash@4.17.21/node_modules/lodash    (top-level symlink)
├── react → .pnpm/react@18.2.0_/node_modules/react
└── react-dom → .pnpm/react-dom@18.2.0_react@18.2.0/node_modules/react-dom
```

Two kinds of links:

1. **Top-level symlinks** in `node_modules/<pkg>` point into `.pnpm/`. Only the project's *direct* dependencies appear at the top level. Indirect (transitive) dependencies don't.
2. **Inside .pnpm/<pkg-id>/node_modules/**, each package's own dependencies are present as symlinks to their respective `.pnpm/<dep-id>/node_modules/<dep>` directories.

The "alphabet soup" directory names (`react-dom@18.2.0_react@18.2.0`) encode peer-dependency resolutions. When a package has a peer dependency, its actual on-disk identity depends on which version of the peer is satisfying the requirement. So `react-dom@18.2.0` paired with `react@18.2.0` becomes `react-dom@18.2.0_react@18.2.0`. This is essential for correctness — multiple versions of `react` can coexist in a workspace, and each `react-dom` instance must point at its specific paired `react`.

**The strict resolution guarantee.** Because only direct dependencies appear at the top level of `node_modules`, code in your project can `require('react')` only if `react` is in your `package.json`. If you accidentally use a transitive dependency (a "phantom dependency"), the require fails with `MODULE_NOT_FOUND`. npm and Yarn-classic flatten everything, so phantom dependencies "work" until the day a transitive bumps a version or removes the package entirely.

This stricter resolution catches real bugs. Migrating from npm to pnpm typically reveals a handful of phantom dependencies — packages your code uses that aren't actually in package.json. Adding them explicitly is the fix.

The Node.js module resolution algorithm (https://nodejs.org/api/modules.html#all-together) walks up directories looking for `node_modules/`. With pnpm's layout, when resolving from `.pnpm/react-dom@18.2.0_react@18.2.0/node_modules/react-dom/index.js`, the resolver finds `.pnpm/react-dom@.../node_modules/react/` (the symlink). This works because Node.js by default *follows* symlinks during module resolution (see Section 13 for the exception with `--preserve-symlinks`).

## Lockfile (pnpm-lock.yaml)

The pnpm lockfile is YAML, not JSON. The choice was deliberate: YAML is more diff-friendly for large files (block-style nested structures), supports comments, and is easier for humans to read.

Structure:

```yaml
lockfileVersion: '6.0'

settings:
  autoInstallPeers: true
  excludeLinksFromLockfile: false

importers:
  .:
    dependencies:
      lodash:
        specifier: ^4.17.21
        version: 4.17.21
      react:
        specifier: ^18.2.0
        version: 18.2.0
    devDependencies:
      vitest:
        specifier: ^1.0.0
        version: 1.0.4

packages:

  /lodash@4.17.21:
    resolution: {integrity: sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs17LhbZVGedAJv8XZ1tvj5FvSg==}
    dev: false

  /react@18.2.0:
    resolution: {integrity: sha512-/3IjMdb2L9QbBdWiW5e3P2/npwMBaU9mHCSCUzNln0ZCYbcfTsGbTJrU/kGemdH2IWmB2ioZ+zkxtmq6g09fGQ==}
    engines: {node: '>=0.10.0'}
    dependencies:
      loose-envify: 1.4.0
    dev: false

  /react-dom@18.2.0(react@18.2.0):
    resolution: {integrity: sha512-...}
    peerDependencies:
      react: ^18.2.0
    dependencies:
      react: 18.2.0
      scheduler: 0.23.0
    dev: false
```

Key sections:

**`lockfileVersion`** — schema version. pnpm refuses to read newer versions; older versions are read but a warning is shown. Major-version bumps (5.0 → 6.0 → 9.0) often correspond to pnpm major releases.

**`settings`** — captures pnpm config that affects resolution. Notably `autoInstallPeers` (silently install missing peers), `excludeLinksFromLockfile` (whether `link:` deps appear), and a few others. Stored so that two developers with different .npmrc settings still produce identical locks if they share the same `settings` block.

**`importers`** — for monorepos, one entry per workspace package (keyed by relative path); for single-package projects, just `.`. Each importer lists its declared dependencies (from package.json) with both the *specifier* (what was written, like `^4.17.21`) and the *version* (what was resolved, like `4.17.21`). The specifier preservation lets pnpm detect drift: if you edit `^4.17.21` to `^5.0.0` in package.json, the lockfile's specifier no longer matches and pnpm knows to re-resolve.

**`packages`** — one entry per (package, version, peer-resolution) triple. Each entry has:

- `resolution` — the integrity hash, plus optional `tarball` URL for non-registry packages.
- `engines` — the package's engines requirement (warning, not enforced unless you set `engine-strict=true`).
- `dependencies` / `peerDependencies` / `optionalDependencies` — adjacency lists.
- `dev: true|false` — whether this package is dev-only (not needed in production install).
- `cpu` / `os` — platform-specific filters; lockfile records all candidates and installer picks at install time.

The `(react@18.2.0)` suffix on `react-dom@18.2.0(react@18.2.0)` mirrors the on-disk directory naming. The same package version with different peer resolutions appears as multiple entries.

The integrity hash is SHA-512 (`sha512-...`), encoded as base64. This matches the npm registry's metadata format. pnpm verifies the hash on every download and on every store import.

Lockfile size for a typical mid-sized project is 5,000-50,000 lines. For a monorepo with 50+ packages, it can reach hundreds of thousands of lines. YAML's block structure keeps it diffable; tools that pretty-print or normalize the file (sorted keys, consistent indentation) help PR reviews enormously.

## Resolver

pnpm's resolver is a depth-first traversal of the dependency graph, with caching and deduplication.

For each direct dependency:

1. Parse the specifier (`^1.2.3`, `latest`, `git+https://...`, `file:../local-pkg`, etc.).
2. Query the registry for matching versions (or read git/file as appropriate).
3. Apply the specifier's selection rule (caret/tilde/exact/range) to pick a version.
4. Recurse into the chosen version's dependencies.

For each transitive dependency, pnpm uses *pickfun* selection: prefer a version already chosen for some other path in the graph, if it's compatible. This is the standard "deduplication" optimization — avoid having `lodash@4.17.20` and `lodash@4.17.21` if both projects could use `4.17.21`.

Peer dependencies complicate this. A peer dependency is a package that the dependent expects to be provided by the *parent* in the dependency tree, not by the package itself. Classic case: `react-dom` peer-depends on `react`. The `react-dom` package author doesn't list `react` as a regular dependency because they want the user's `react` (potentially different versions in different parts of the tree) to be the actual one used. pnpm's `.pnpm/<pkg>@<ver>(<peer>@<ver>)` naming is precisely the mechanism for tracking which `react` is paired with which `react-dom`.

When the peer is missing or unresolvable:

- `autoInstallPeers: true` (default in pnpm 8+): pnpm silently adds the peer to the project's resolution.
- `autoInstallPeers: false`: pnpm warns about the missing peer and the dependent may fail at runtime.

**Strictness evolution.** Earlier pnpm versions (1.x – 4.x) were strict by default: a missing peer was a hard install failure. This caught real bugs but also blocked installs on packages with overly-aggressive peer ranges. pnpm 5+ relaxed this to warnings; pnpm 8 added `autoInstallPeers` to fix the warning without manual intervention. Some teams set `strict-peer-dependencies=true` to restore the old behavior for production correctness.

**Conflict detection.** If two packages peer-depend on different versions of the same peer (`A peer-needs react ^18`, `B peer-needs react ^17`, both in the same project), pnpm produces a warning. The resolver still completes — picking one version of `react` and reporting the conflict — but the application may misbehave at runtime if both `A` and `B` are used. `resolutions` (npm's term: `overrides`) in package.json can force a specific version.

Override syntax in package.json:

```json
{
  "pnpm": {
    "overrides": {
      "react": "18.2.0",
      "lodash@<4.17.21": "4.17.21"
    }
  }
}
```

The first form pins `react` to exactly `18.2.0` everywhere; the second uses a "selector" syntax to upgrade only specific old versions. Useful for security backports.

## Strict by Default

pnpm's strictness manifests in several places:

**Strict node_modules layout.** Only direct deps at the top level. Phantom-dependency code fails. Already covered in Section 3.

**Strict peer dependencies (configurable).** Set `strict-peer-dependencies=true` in `.npmrc` to fail installs on missing peers, mismatched peers, etc.

**Strict integrity verification.** Every downloaded package is checked against its integrity hash. If the registry serves a different file (compromised registry, MITM, cache corruption), pnpm refuses.

**Strict workspace protocol.** In a workspace, `"my-internal-pkg": "workspace:^1.0"` declares that the dependency must come from a workspace member. If no workspace member has that name + version, install fails. This catches misconfigured workspaces that would silently install from npm registry instead.

**Strict engine checks (opt-in).** `engine-strict=true` makes the `engines` field in package.json a hard fail rather than a warning. If your package requires Node >=18 and the user has Node 16, `pnpm install` errors out.

This strictness was sometimes painful in early adoption — projects designed against npm's loose semantics encountered failures. The pnpm team and ecosystem have absorbed the lessons, and the strictness has become widely-praised: the failures it surfaces are real bugs, just made visible earlier.

## Workspaces

A workspace is a multi-package repository sharing a common dependency graph. pnpm's workspace support is one of its strongest features.

Define workspaces with `pnpm-workspace.yaml` at the repo root:

```yaml
packages:
  - "packages/*"
  - "apps/*"
  - "!**/test/**"
```

Globs match directories containing `package.json`. The `!` prefix excludes patterns.

Inside each workspace package, you can reference siblings:

```json
{
  "name": "@myorg/web",
  "dependencies": {
    "@myorg/core": "workspace:^1.0.0"
  }
}
```

The `workspace:` protocol tells pnpm to resolve from the local workspace, not the npm registry. The version after `workspace:` (or `workspace:*`, `workspace:^`, `workspace:~`) controls how the version is rewritten when the package is published:

- `workspace:^1.0.0` — replaced with `^1.0.0` on publish (the workspace's actual version becomes the floor).
- `workspace:^` — replaced with `^<workspace-version>` (resolved at publish time).
- `workspace:*` — replaced with the exact workspace version.

Without rewriting, an external user installing your published package wouldn't be able to resolve `workspace:` because they don't have your workspace.

In `node_modules`, a workspace dep appears as a symlink to the source directory (no copy):

```
packages/web/node_modules/@myorg/core → ../../../core/
```

This means edits in `packages/core/src/index.ts` are immediately visible to `packages/web` — no rebuild step. This live-linking is the core developer-experience win of workspaces.

Workspace-aware commands:

```bash
pnpm -r run build              # run "build" in every workspace
pnpm -r --parallel run dev     # run "dev" in every workspace, in parallel
pnpm --filter @myorg/web run test  # run "test" only in @myorg/web
pnpm --filter "./apps/*" run lint  # filter by path glob
pnpm --filter "@myorg/web^..." build  # build @myorg/web and all its dependents (including transitively)
pnpm --filter "...^@myorg/core" build  # build @myorg/core and everything it depends on
```

The `^...` and `...^` operators are dependency-graph walks. `^...` means "this and everything that depends on it (downstream)"; `...^` means "this and everything it depends on (upstream)". This is essential for incremental builds: change a low-level package, build it and all consumers; change a high-level app, build it and all of its dependencies.

## Filtering

Filtering syntax extends beyond workspace selection. The `--filter` flag accepts:

- `<pkg-name>` — exact match by name.
- `"<glob>"` — name glob, e.g. `"@myorg/*"`.
- `"./path/glob"` — directory glob, e.g. `"./apps/*"`.
- `"...<sel>"` — sel and all dependencies (transitive).
- `"<sel>..."` — sel and all dependents (transitive).
- `"...^<sel>"` — sel and all dependencies (excluding sel).
- `"<sel>^..."` — sel and all dependents (excluding sel).
- `"[<since-ref>]"` — packages changed since a git ref. e.g. `"[origin/main]"`.

The git-since filter is particularly powerful in CI:

```bash
pnpm --filter "[origin/main]" run test
```

means "run tests only on packages whose source files have changed since `origin/main`". For large monorepos, this can cut CI time from hours to minutes. Combined with dependency-graph walking:

```bash
pnpm --filter "...[origin/main]" run test
```

means "and also test everything that depends on changed packages (since they may break)". This is conservative-correct: any test that could be affected by the change runs.

The implementation walks the workspace graph: each `<sel>` resolves to a set of packages; `^...` adds dependents; `...^` adds dependencies; the filter set is the union. The graph is the same one in `pnpm-lock.yaml`, so this works without re-resolving.

## Patches

Sometimes you need to fix a bug in a third-party package without waiting for upstream. npm's traditional answer was `patch-package` (a separate tool). pnpm has it built in:

```bash
pnpm patch lodash@4.17.21
```

This:

1. Copies `lodash@4.17.21` from the store into a temp directory.
2. Tells you the temp path. You edit the files however you want.
3. Run `pnpm patch-commit <temp-path>` when done.
4. pnpm computes a unified diff and writes it to `patches/lodash@4.17.21.patch`.
5. pnpm adds an entry in package.json:

```json
{
  "pnpm": {
    "patchedDependencies": {
      "lodash@4.17.21": "patches/lodash@4.17.21.patch"
    }
  }
}
```

On the next install, pnpm:

1. Fetches `lodash@4.17.21` from the store (or registry if not cached).
2. Applies the patch.
3. Stores the *patched* version in a separate store entry, keyed by the patch's hash.

Subsequent installs share the patched version across projects (just like the unpatched version). The patch file is committed; the patched files are not (they're in the store and `node_modules`).

Patches work well for surgical fixes: a single line change, a missing null check, a wrong default. They don't scale to large refactors — at that point, fork the package or wait for upstream.

## Migration

Switching from npm/yarn to pnpm is mostly a matter of:

```bash
pnpm import      # convert package-lock.json or yarn.lock → pnpm-lock.yaml
rm -rf node_modules
pnpm install
```

`pnpm import` reads the existing lockfile and produces an equivalent pnpm-lock.yaml. Versions are preserved exactly; the only changes are in graph structure (peer-resolution paths) and lockfile schema.

Common migration pitfalls:

**Phantom dependencies.** Code that worked under npm fails under pnpm because it requires a transitive package not in package.json. Fix: add the package as a direct dependency.

**Hoisted resolution assumptions.** Some build tools (older webpack configs, certain Babel plugins) assume "find module X anywhere in node_modules". pnpm's strict layout breaks them. Workarounds:

- `public-hoist-pattern[]=*types*` in .npmrc — selectively hoist patterns to the top level.
- `node-linker=hoisted` in .npmrc — disable the .pnpm/ layout entirely; install in the npm-style flat tree (loses some benefits but maximizes compatibility).
- Fix the tool to use proper module resolution.

**Symlink-unaware tools.** Old tools that don't follow symlinks see empty node_modules. Rare, but if you encounter one, file an issue or use `node-linker=hoisted`.

**Mixing.** If a single project has both `package-lock.json` and `pnpm-lock.yaml`, pnpm errors. CI must use one consistently. The recommendation: commit only the chosen lockfile and add the others to .gitignore.

## The disk-savings math

Why does pnpm save disk? Suppose you have 10 projects, each depending on `lodash@4.17.21`, `react@18.2.0`, and `webpack@5.x`.

Under npm:

```
project-1/node_modules/lodash/  (4 MB)
project-1/node_modules/react/   (300 KB)
project-1/node_modules/webpack/ (3 MB)
project-2/node_modules/lodash/  (4 MB, different inodes)
project-2/node_modules/react/   (300 KB)
...
```

Total: 10 × (4 + 0.3 + 3) MB = ~73 MB.

Under pnpm:

```
~/.local/share/pnpm/store/v3/files/  (one copy of each package's files: 4 + 0.3 + 3 ≈ 7.3 MB)
project-1/node_modules/.pnpm/lodash@4.17.21/  (hardlinks, 0 MB additional)
project-2/node_modules/.pnpm/lodash@4.17.21/  (hardlinks, 0 MB additional)
...
```

Total: ~7.3 MB regardless of project count.

In practice, exact savings depend on:

- *Reuse rate*: more projects sharing the same package versions → bigger savings.
- *Package-set churn*: if every project uses different versions, savings approach zero.
- *Filesystem*: hardlinks are most efficient on ext4/xfs/apfs/ntfs; they degrade on networked filesystems.

Real-world reports on monorepos with 50-200 packages often show 50-80% savings versus npm. For a CI runner that builds dozens of branches per day, store-mediated installs are a major perf and cost win.

## Speed math

Installing a fresh project under npm takes time proportional to:

1. Network: download every tarball.
2. Disk: extract every tarball.
3. CPU: compute integrity hashes (SHA-512) for every file.
4. Resolution: build the dependency graph (CPU + small network for metadata).

pnpm's wins:

**(1) Network with warm store**: zero downloads. The store has the tarballs (or rather, their extracted files). Install is purely linking + symlinks.

**(2) Disk with warm store**: zero extraction. Files are already in the store. Linking is a directory-write (microseconds per package).

**(3) CPU**: hash computed once at store-import time, never repeated for subsequent installs.

**(4) Resolution**: lockfile-based — no resolver needed if the lock is valid. (Fresh resolution is comparable to npm/yarn.)

Cold install (empty store, fresh `pnpm install` on a new machine): pnpm is roughly comparable to npm in network/disk time. The advantage is small.

Warm install (subsequent `pnpm install` after dependency change): pnpm is significantly faster — often 5-10× — because the store hits make almost all per-package work instantaneous.

CI install (clean checkout + pre-warmed store via cache key): pnpm shines. The store can be cached across CI runs, and a typical install becomes seconds.

The biggest win is *across projects*: a developer's machine, with N projects sharing a store, gets a free speed boost on every project after the first.

## Symlink Resolution Caveats

Node.js's module resolution algorithm follows symlinks by default. When you `require('react-dom')`, Node:

1. Looks for `node_modules/react-dom` in the current directory.
2. Walks up parent directories.
3. Finds `node_modules/react-dom` (which may be a symlink to `.pnpm/react-dom@.../node_modules/react-dom`).
4. Resolves the symlink to the real path.
5. Continues resolution from the real path's perspective.

This is the standard behavior and what pnpm relies on. When `react-dom` does `require('react')`, Node's resolver, *now operating in `.pnpm/react-dom@.../node_modules/react-dom/`*, walks up to `.pnpm/react-dom@.../node_modules/`, finds `react` (symlink to `.pnpm/react@.../node_modules/react/`), and resolves through.

**`--preserve-symlinks` mode.** Node has a flag, `--preserve-symlinks`, that instructs the resolver *not* to follow symlinks during resolution. This is occasionally desired for esoteric reasons (mostly: ensuring relative paths work consistently in modules linked via `npm link`). When set, pnpm's layout breaks: the resolver, sitting in `node_modules/react-dom` (a symlink), doesn't follow to its target, and can't find `react` because it doesn't see the `.pnpm/react-dom@.../node_modules/` parent directory.

The workaround is `--preserve-symlinks-main` (preserves only the entry point, not the resolution lookups), or simply not using `--preserve-symlinks` with pnpm-managed projects.

Most mainstream tools (Vite, webpack, esbuild, tsc, jest) work fine with pnpm's default resolution. Edge cases involve tools that:

- Compute paths via `__dirname` and assume they're "real" rather than symlinked.
- Cache resolution by inode and get confused by symlink targets.
- Use `realpath()` on intermediate paths and skip pnpm's layered structure.

When such tools fail, the usual fix is `node-linker=hoisted` (turn off pnpm's strictness, install in npm-style).

For monorepo scenarios, peer-dependency-paired packages have *multiple* on-disk copies (different `.pnpm/<pkg>@<ver>(<peer>@<ver>)` directories per peer combination). This is correct and necessary, but tools that walk `node_modules/` linearly may visit the same package multiple times.

## Hoisting (the `node-linker` setting)

By default, pnpm uses the strict isolated layout (`node-linker=isolated`). Alternatives:

`node-linker=hoisted` — falls back to the npm/yarn-classic flat layout. No `.pnpm/`, no symlinks, all packages at the top level of `node_modules`. Compatibility with old tools is maximized; phantom-dependency protection and disk savings are largely lost. Useful for projects that just can't be made to work with isolated layout.

`node-linker=pnp` — Yarn 2/Plug'n'Play style. No `node_modules` at all; instead, a `.pnp.cjs` file at the root tells Node where each package lives via a custom resolver. Fastest layout (no filesystem ops at all), but requires patching every Node.js process to load the resolver, which is incompatible with many tools. Marginal use; mostly a Yarn-PnP migration aid.

The trade-off is real: isolated catches bugs but exposes incompatibilities; hoisted is permissive but loses correctness; pnp is fastest but requires deep tool support.

`public-hoist-pattern[]=<glob>` — selectively hoist matching packages to the top level even in isolated mode. Common patterns:

```
public-hoist-pattern[]=*types*       # hoist @types/* for IDE support
public-hoist-pattern[]=*eslint*      # hoist ESLint plugins
public-hoist-pattern[]=*prettier*
```

This selectively breaks isolation for tools known to require flat resolution (older ESLint plugins, IDE TypeScript servers, etc.).

## Lockfile vs. shrinkwrap, why YAML

npm 5+ uses `package-lock.json`. Yarn classic uses `yarn.lock` (custom format). pnpm uses `pnpm-lock.yaml`.

Why YAML? When pnpm's lockfile schema was designed (around 2018), JSON was the obvious choice but had three downsides:

1. **No comments**, so settings/rationale must live elsewhere.
2. **Verbose nesting** for large objects — lots of `{`, `}` characters that are noise.
3. **Diff-unfriendly** — minor reordering by tooling produces large diffs.

YAML solves all three. The lockfile is also stable: pnpm sorts keys deterministically, indents consistently, and emits the same file for the same input. Diffs in PRs show actual semantic changes — version bumps, integrity-hash updates, new packages added.

The downside is parsing speed (YAML is slower to parse than JSON) and the risk of YAML's edge cases (the famous Norway problem: `country: NO` parses as boolean false). pnpm uses a strict YAML subset and validates the parsed result, so practical issues are rare.

## Registry

The default registry is `https://registry.npmjs.org/` — same as npm. Configured in `.npmrc`:

```
registry=https://registry.npmjs.org/
```

Per-scope registries:

```
@myorg:registry=https://npm.myorg.com/
```

means "for any package in the `@myorg/` scope, use this registry instead". Useful for private packages.

Authentication:

```
//registry.npmjs.org/:_authToken=npm_xxxxxxxxxxxxxxxxxx
//npm.myorg.com/:username=myuser
//npm.myorg.com/:_password=base64-encoded-password
//npm.myorg.com/:email=me@myorg.com
```

These can be in `~/.npmrc` (global) or `.npmrc` at the project root (project-specific). pnpm reads npm's config files unchanged.

For CI, environment variables are preferred:

```
NPM_TOKEN=npm_xxxxxx
```

with .npmrc:

```
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
```

pnpm expands `${VAR}` from the environment, so credentials never appear in committed files.

## Hooks

pnpm supports lifecycle hooks via package.json scripts (same names as npm: `prepare`, `prepublish`, `postinstall`, etc.). It also adds:

- `pnpm:devPreinstall` / `pnpm:postinstall` — pnpm-specific hooks at the workspace root.
- `.pnpmfile.cjs` — programmatic hook for read-only or read-write modification of resolved packages.

Example `.pnpmfile.cjs`:

```javascript
function readPackage(pkg) {
  // remove unwanted peer deps
  if (pkg.name === 'some-bad-package') {
    delete pkg.peerDependencies;
  }
  return pkg;
}

module.exports = { hooks: { readPackage } };
```

This is invoked for every package as it's resolved. You can rewrite dependencies, peer dependencies, hooks — useful for working around upstream bugs or enforcing organization-wide policy.

For security, postinstall scripts (the `postinstall` field in package.json) are widely-recognized as a malware vector. pnpm 9+ defaults to *not* running postinstall scripts unless explicitly allowlisted via `onlyBuiltDependencies` in package.json:

```json
{
  "pnpm": {
    "onlyBuiltDependencies": ["esbuild", "node-sass"]
  }
}
```

Without this, packages relying on postinstall (compilation of native code, etc.) may silently fail to build. The trade-off is explicit security: you opt in to running each package's scripts.

## Cache pruning

The store grows over time. To reclaim space:

```bash
pnpm store prune       # remove orphans (no longer referenced by any project)
pnpm store path        # show store location
pnpm store status      # show store size
```

`prune` walks every project's lockfile (configurable via `pnpm config get globalDir`) and finds files in the store not referenced by any project. Those become candidates for deletion. It's safe to run manually periodically.

Some teams set up a cron job: weekly `pnpm store prune` keeps the store from growing unboundedly. Total store size on a busy developer machine is typically 5-20 GB.

## Why pnpm exists / why it succeeded

The Node ecosystem went through three generations of package managers:

1. **npm 1.x – 2.x** — nested `node_modules`, deeply nested directories, "Windows long path" issues, slow.
2. **npm 3+, Yarn classic** — flat hoisted `node_modules`, fast(er), but enables phantom dependencies and the "hoisting tax" of more files.
3. **pnpm, Yarn PnP, Yarn Berry** — content-addressable store, strict resolution, smaller disk footprint.

pnpm's specific innovation — hardlinks + isolated layout — is the most ergonomically successful of the third generation. PnP's "no node_modules at all" is technically beautiful but breaks too many tools. pnpm's design works with the existing Node module resolution algorithm, just leverages it differently.

The success has been steady: 2018 had ~thousands of pnpm users; by 2024, single-digit-million weekly downloads, adoption by large companies and major open-source projects, and the de facto choice for new monorepos. The hardlinks-and-symlinks idea has also influenced npm's own thinking: `npm install --omit=optional` and other features hint at moving toward content-addressable optimization.

## Configuration reference (selective)

The `.npmrc` keys most relevant to pnpm:

```
# Storage
store-dir=~/.local/share/pnpm/store/v3
package-import-method=auto    # hardlink, clone, copy, auto

# Layout
node-linker=isolated           # isolated, hoisted, pnp
public-hoist-pattern[]=*types*
hoist-pattern[]=*eslint*       # hoisted to .pnpm/<pkg>/node_modules but not top-level

# Strictness
strict-peer-dependencies=false
auto-install-peers=true
prefer-frozen-lockfile=true     # CI-friendly

# Performance
side-effects-cache=true
side-effects-cache-readonly=false

# Network
registry=https://registry.npmjs.org/
network-concurrency=16          # parallel downloads
fetch-retries=2
fetch-retry-mintimeout=10000

# Security
verify-store-integrity=true
ignore-scripts=false           # block postinstall scripts globally
```

Most defaults are sensible. The most-tweaked are `node-linker` (when an old tool needs hoisting) and `auto-install-peers` (when CI wants strict peer behavior).

## Common errors and fixes

**`ELIFECYCLE Command failed`** — a script (postinstall, prepare, etc.) returned non-zero. Look at the previous lines for the actual error.

**`peer X is required but not installed`** — a peer dependency wasn't satisfied. Set `auto-install-peers=true` or add it manually.

**`ERR_PNPM_NO_MATCHING_VERSION_INSIDE_WORKSPACE`** — a `workspace:` reference doesn't match any workspace member. Check the version specifier.

**`ENOTEMPTY` on rmdir** — pnpm tried to remove a directory that wasn't empty. Often caused by an editor or build tool holding open file handles. Close the editor and retry.

**`tarball cannot be extracted`** — corrupt download. Run `pnpm store prune` and retry.

**`Cannot find module 'X'`** at runtime — phantom dependency. Add `X` to package.json.

## Going deeper

pnpm's source is in TypeScript and is well-organized. Key directories:

- `pnpm/cli/` — CLI entry and commands.
- `pnpm/store/` — content-addressable store implementation.
- `pnpm/resolver-base/`, `pnpm/npm-resolver/` — the resolver(s).
- `pnpm/lockfile-file/` — lockfile read/write.
- `pnpm/headless/` — the headless installer (no resolution, just install from lock).

For the algorithmic story: read `lockfile/lockfile-utils/src/satisfiesPackageManifest.ts` (lockfile validation), `resolve-dependencies/src/index.ts` (resolver entry), and the `lockfile-walker` package.

For the linking story: read `package-store/src/createPackageRequester.ts` (store imports) and `link-bins/src/index.ts` (creating .bin/ entries).

The pnpm team blogs at https://pnpm.io/blog and posts deep-dives on architectural decisions. Recommended: "The case for pnpm" (early-2020) and "Why pnpm v9 changes the default for security" (mid-2024).

## References

- pnpm documentation — https://pnpm.io/
- pnpm source — https://github.com/pnpm/pnpm
- pnpm benchmarks — https://github.com/pnpm/benchmarks-of-javascript-package-managers
- npm CLI documentation — https://docs.npmjs.com/cli
- Yarn documentation — https://yarnpkg.com/
- Node.js module resolution — https://nodejs.org/api/modules.html
- Node.js --preserve-symlinks documentation — https://nodejs.org/api/cli.html#--preserve-symlinks
- Hardlink vs symlink overview — https://en.wikipedia.org/wiki/Hard_link
- Reflink (copy-on-write) — https://btrfs.wiki.kernel.org/index.php/Reflink
- npm package-lock.json schema — https://docs.npmjs.com/cli/v10/configuring-npm/package-lock-json
- npm registry API — https://github.com/npm/registry/blob/main/docs/REGISTRY-API.md
- pnpm-lock.yaml format — https://pnpm.io/git#lockfile
- pnpm-workspace.yaml — https://pnpm.io/pnpm-workspace_yaml
- pnpm filtering — https://pnpm.io/filtering
- pnpm patches — https://pnpm.io/cli/patch
- Phantom dependencies — https://rushjs.io/pages/advanced/phantom_deps/
- Doppelganger packages — https://rushjs.io/pages/advanced/npm_doppelgangers/
- The ".pnpm" directory layout — https://pnpm.io/symlinked-node-modules-structure
- Yarn PnP — https://yarnpkg.com/features/pnp
- npm doppelgangers — https://github.com/npm/cli/issues/3953
- pnpm vs npm vs yarn comparison — https://pnpm.io/feature-comparison
