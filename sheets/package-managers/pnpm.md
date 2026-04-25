# pnpm

Fast, disk-efficient Node.js package manager. Content-addressable global store, hardlink-and-symlink layout, strict-by-default node_modules, first-class workspaces, deterministic lockfile.

## Setup

Pinned versioning is the recommended path on Node 16.13+ via Corepack — it reads the `packageManager` field in `package.json` and downloads the exact version on first invocation, so contributors and CI agree on bytes-for-bytes the same pnpm.

```bash
# Corepack (ships with Node >= 16.13)
corepack enable
corepack prepare pnpm@9.12.3 --activate
pnpm --version          # 9.12.3
```

```bash
# Homebrew (macOS / Linuxbrew)
brew install pnpm
brew upgrade pnpm
brew uninstall pnpm
```

```bash
# Global npm install (fallback when Corepack is disabled)
npm install -g pnpm
npm install -g pnpm@9
npm install -g pnpm@latest
npm install -g pnpm@next-9
```

```bash
# Standalone official install script (no Node required)
curl -fsSL https://get.pnpm.io/install.sh | sh -
curl -fsSL https://get.pnpm.io/install.sh | env PNPM_VERSION=9.12.3 sh -
curl -fsSL https://get.pnpm.io/install.sh | env SHELL="$(which zsh)" sh -
```

```bash
# Windows PowerShell standalone
iwr https://get.pnpm.io/install.ps1 -useb | iex
$env:PNPM_VERSION="9.12.3"; iwr https://get.pnpm.io/install.ps1 -useb | iex
```

```bash
# winget / scoop / choco
winget install pnpm.pnpm
scoop install nodejs-lts pnpm
choco install pnpm
```

```bash
# Snap
sudo snap install node --classic
sudo npm install -g pnpm
```

```bash
# Self-update once installed
pnpm self-update           # to latest
pnpm self-update 9.12.3    # to specific version
pnpm self-update next      # to next-tagged release
```

```bash
# Pin in package.json (single source of truth for the repo)
{
  "name": "my-app",
  "packageManager": "pnpm@9.12.3"
}
```

```bash
# Pin with integrity hash (Corepack will verify)
{
  "packageManager": "pnpm@9.12.3+sha512.abcdef...."
}
```

```bash
# Pin via .npmrc engine-strict
engine-strict=true
```

```bash
# Engines field gates the version
{
  "engines": {
    "node": ">=20.10.0",
    "pnpm": ">=9.0.0"
  }
}
```

```bash
# Disable global pnpm fallback when Corepack is the source of truth
COREPACK_ENABLE_STRICT=1 pnpm install
```

```bash
# Verify install + diagnostics
pnpm --version
pnpm env list
pnpm config list
which pnpm
pnpm doctor          # 9.x: prints store info, integrity, perms
```

```bash
# Path layout after install
~/Library/pnpm                       # PNPM_HOME (macOS user)
~/.local/share/pnpm                  # PNPM_HOME (Linux user)
%LOCALAPPDATA%\pnpm                  # PNPM_HOME (Windows user)
$PNPM_HOME                           # honors override
$PATH must include $PNPM_HOME       # global bins live there
```

```bash
# Set PNPM_HOME explicitly (zsh/bash)
export PNPM_HOME="$HOME/.local/share/pnpm"
case ":$PATH:" in
  *":$PNPM_HOME:"*) ;;
  *) export PATH="$PNPM_HOME:$PATH" ;;
esac
```

```bash
# Uninstall completely
rm -rf "$PNPM_HOME"
rm -rf ~/.local/state/pnpm
rm -rf ~/.cache/pnpm
rm -rf ~/.npmrc                # only if you no longer use npm either
brew uninstall pnpm            # Homebrew route
npm uninstall -g pnpm          # npm route
```

## Why pnpm

```bash
# vs npm:
#   npm                                     pnpm
#   - flat node_modules with phantom deps   - strict isolated tree, no phantom deps
#   - duplicates same package per project   - global content-addressable store
#   - heavy disk use                        - hardlinks, ~50% less disk
#   - slow installs (~45s for big repos)    - ~2-3x faster (~15-20s same repo)
#   - workspaces work but no filtering DSL  - first-class --filter selectors
```

```bash
# vs yarn classic (1.x):
#   - yarn classic flat node_modules, dead PnP support
#   - pnpm strict + isolated, supports node-linker=pnp / hoisted as escape hatches
#   - pnpm lockfile (pnpm-lock.yaml) is YAML, deterministic, mergeable
```

```bash
# vs yarn berry (3.x/4.x):
#   - yarn berry default: PnP (.pnp.cjs), zero-installs
#   - pnpm default: real node_modules with symlinks (works with every tool)
#   - both have content stores; pnpm's hardlink layout is closer to the real FS
#   - pnpm has simpler config (.npmrc), berry has richer .yarnrc.yml
```

```bash
# Quantitative wins (typical mid-size monorepo, ~150 deps, 8 workspaces):
#   - install (cold):   npm 65s   yarn 50s   pnpm 22s
#   - install (warm):   npm 18s   yarn 12s   pnpm  3s
#   - disk usage:       npm 1.2GB yarn 1.1GB pnpm 380MB (per-machine, shared store)
#   - CI determinism:   only --frozen-lockfile is guaranteed deterministic; pnpm enforces it by default in CI
```

```bash
# Strict-by-default benefits
# A package can ONLY require modules listed in its own dependencies/peerDependencies.
# Phantom deps (a transitive that "happens to be there") will fail at require-time.
# Catches bugs early; production crashes from missing-but-implicit deps disappear.
```

## Architecture: Content-Addressable Store

```bash
# Default store path (override with PNPM_HOME or store-dir)
~/.local/share/pnpm/store/v3        # Linux  (XDG)
~/Library/pnpm/store/v3             # macOS
%LOCALAPPDATA%\pnpm\store\v3        # Windows
```

```bash
# Inside the store
~/.local/share/pnpm/store/v3/
├── files/                  # raw blobs, indexed by integrity hash
│   ├── 00/00ab1f2...
│   ├── 01/01cd3e4...
│   └── ff/fffefdc...
├── tmp/                    # in-flight downloads
├── index/                  # mapping pkg@version -> integrity
└── store.json
```

```bash
# Each package version exists as ONE blob set globally; node_modules links into it
# Layout per project:
node_modules/
├── .pnpm/                          # virtual store (hidden real packages)
│   ├── lodash@4.17.21/
│   │   └── node_modules/
│   │       └── lodash/             # hardlink chain to ~/.local/share/pnpm/store/v3/files/...
│   ├── react@18.3.1/
│   │   └── node_modules/
│   │       ├── react/
│   │       └── loose-envify        # peer dep symlinked to its own .pnpm dir
│   └── lock.yaml                   # integrity manifest mirroring pnpm-lock.yaml
├── lodash -> .pnpm/lodash@4.17.21/node_modules/lodash
└── react  -> .pnpm/react@18.3.1/node_modules/react
```

```bash
# Why hardlinks: zero copy, instant install, identical inode across projects,
# but each project's node_modules is still a real directory.
# Caveat: if a script writes to node_modules/<pkg>/file it edits the GLOBAL blob.
# pnpm sets package files read-only by default; toggle with package-import-method=copy.
```

```bash
# Inspect store layout
pnpm store path                      # absolute path of active store
pnpm store status                    # report any missing files
pnpm store prune                     # remove blobs not referenced by any project
du -sh ~/.local/share/pnpm/store     # total size
ls ~/.local/share/pnpm/store/v3/index | wc -l
```

```bash
# Migrate the store between disks
pnpm store path
mv ~/.local/share/pnpm/store /new/disk/pnpm/store
pnpm config set store-dir /new/disk/pnpm/store
pnpm install                          # relinks projects to new path
```

```bash
# Per-project virtual store + integrity manifest
node_modules/.pnpm/lock.yaml          # integrity sidecar (NOT pnpm-lock.yaml)
node_modules/.modules.yaml            # snapshot of options used at install time
```

```bash
# Verify everything is intact
pnpm install --frozen-lockfile        # CI-grade verify
pnpm store status                     # store-side verify
pnpm rebuild                          # re-run lifecycle scripts on stored pkgs
```

## Install

```bash
pnpm install                           # default: respect lockfile, write what's missing
pnpm i                                 # alias
pnpm add                               # alias for install when no pkg name given (deprecated)
```

```bash
# CI / reproducible install
pnpm install --frozen-lockfile         # FAIL if pnpm-lock.yaml needs updates
pnpm install --frozen-lockfile=false   # force unfreeze
PNPM_FROZEN_LOCKFILE=1 pnpm install    # env-var equivalent
# Note: --frozen-lockfile is the DEFAULT in CI environments (CI=true)
```

```bash
# Lockfile-only / no node_modules
pnpm install --lockfile-only           # update lockfile, do NOT touch node_modules
pnpm install --no-lockfile             # do NOT read or write a lockfile
pnpm install --fix-lockfile            # rewrite lockfile if it has stale entries
```

```bash
# Cache and offline modes
pnpm install --prefer-offline          # use cache when possible, network as fallback
pnpm install --offline                 # NEVER hit network; fail if cache miss
pnpm install --force                   # refetch all packages even if cached
pnpm install --resolution-only         # only resolve graph; no fetch, no link
```

```bash
# Optional / dev / production
pnpm install --no-optional             # skip optionalDependencies
pnpm install --prod                    # skip devDependencies
pnpm install --production              # alias of --prod
pnpm install --dev                     # only devDependencies
NODE_ENV=production pnpm install       # implicit --prod (note: NODE_ENV honored)
```

```bash
# Hoisting controls
pnpm install --shamefully-hoist        # flatten EVERYTHING into top-level node_modules
pnpm install --public-hoist-pattern='*eslint*'   # hoist matches to top-level
pnpm install --public-hoist-pattern='*types*'    # multiple via repeat or .npmrc
pnpm install --hoist-pattern='@types/*'          # hoist to .pnpm/node_modules only
```

```bash
# Lifecycle scripts
pnpm install --ignore-scripts          # skip pre/install/post/prepare across the tree
pnpm install --ignore-pnpmfile         # skip .pnpmfile.cjs hooks
pnpm install --ignore-workspace-cycles # do not error on cyclic workspace deps
```

```bash
# Verbose / debug
pnpm install --reporter=default
pnpm install --reporter=append-only    # CI-friendly
pnpm install --reporter=ndjson         # one JSON event per line
pnpm install --silent
pnpm install --loglevel=debug
```

```bash
# Fix vs strict modes
pnpm install --strict-peer-dependencies        # bail if any peer is unmet
pnpm install --no-strict-peer-dependencies     # warn only (default since 8.x)
pnpm install --fix                              # attempt to repair common mismatches
```

```bash
# Exit codes
0   # success
1   # generic install error
129 # network failure during fetch
130 # SIGINT
```

## Add

```bash
pnpm add lodash                          # install + record in dependencies
pnpm add lodash@4.17.21                  # exact version (still uses ^ unless --save-exact)
pnpm add lodash@latest                   # latest tagged version
pnpm add lodash@next                     # tag-based
pnpm add lodash@4                        # any 4.x
pnpm add 'lodash@>=4.17 <5'              # range
```

```bash
# Dep type
pnpm add -P lodash                       # --save-prod (default)
pnpm add -D vitest                       # --save-dev
pnpm add -O fsevents                     # --save-optional
pnpm add --save-peer react              # peerDependencies entry
pnpm add --save-exact lodash@4.17.21     # writes "4.17.21" not "^4.17.21"
pnpm add --save-prefix='~' lodash        # "~4.17.21"
```

```bash
# Workspace targeting
pnpm add lodash -w                       # add to workspace ROOT package.json
pnpm add lodash --workspace-root         # long form
pnpm add lodash --filter @scope/api      # add to one package in the monorepo
pnpm add lodash --filter './apps/**'     # path-glob filter
```

```bash
# Workspace-protocol references (cross-package deps inside the monorepo)
pnpm add @myorg/utils@workspace:*        # any version in workspace
pnpm add @myorg/utils@workspace:^        # caret of workspace version
pnpm add @myorg/utils@workspace:~        # tilde
pnpm add @myorg/utils@workspace:1.2.3    # pinned, but pull from workspace
```

```bash
# Global installs (use a real bin directory, not the project)
pnpm add -g typescript
pnpm add -g typescript@latest
pnpm add --global eslint prettier
pnpm list -g                              # what's globally installed
pnpm root -g                              # where global bins live
```

```bash
# Tarball / git / local / file
pnpm add ./packages/internal              # local directory
pnpm add file:../other-repo               # explicit file: protocol
pnpm add github:lodash/lodash#4.17.21     # github shorthand
pnpm add https://github.com/lodash/lodash.git#4.17.21
pnpm add git+ssh://git@github.com/x/y.git#main
pnpm add ./pkg-1.0.0.tgz                  # tarball
pnpm add https://example.com/pkg.tgz
```

```bash
# Adding adds AND writes lockfile + symlinks; use --lockfile-only to defer linking
pnpm add lodash --lockfile-only
pnpm install                              # later, materialize node_modules
```

```bash
# Common patterns
pnpm add -D typescript@^5 vitest@^1 eslint@^9 prettier@^3
pnpm add -E react@18.3.1 react-dom@18.3.1            # match versions exactly
pnpm add -P 'lodash@~4.17' 'zod@~3.23'
```

## Remove

```bash
pnpm remove lodash
pnpm rm lodash                            # alias
pnpm uninstall lodash                     # alias
pnpm un lodash                            # short alias
```

```bash
# Dep-type aware (rarely needed; pnpm finds the right field automatically)
pnpm rm -P lodash         # remove from dependencies
pnpm rm -D vitest         # remove from devDependencies
pnpm rm -O fsevents       # remove from optionalDependencies
```

```bash
# Workspace
pnpm rm lodash -w                       # remove from root
pnpm rm lodash --filter @scope/api      # remove from one package
pnpm -r rm lodash                        # remove from EVERY package in workspace
pnpm rm --recursive lodash               # long form
```

```bash
# Global
pnpm remove -g typescript
pnpm rm --global eslint prettier
```

```bash
# Bulk
pnpm rm lodash underscore ramda
pnpm rm '@types/*' --filter ./apps/*     # path-glob + bulk
```

## Update

```bash
pnpm update                               # update all deps within current semver ranges
pnpm up                                   # alias
pnpm upgrade                              # alias
```

```bash
# To latest, ignoring semver
pnpm up --latest                          # bump major/minor/patch to "latest" tag
pnpm up -L                                # short
pnpm up react@latest react-dom@latest
pnpm up '@types/*' --latest               # glob + latest
```

```bash
# Interactive picker (TUI)
pnpm up --interactive                     # check + uncheck which deps to bump
pnpm up -i
pnpm up -i --latest                       # interactive AND ignore semver
pnpm up -i -r                             # interactive across full workspace
```

```bash
# Targeted
pnpm up lodash                            # only this package
pnpm up lodash@4.17.21                    # specific version
pnpm up 'lodash@>=4.17.21 <5'             # constraint
pnpm up 'eslint*'                         # glob
pnpm up react react-dom --latest          # paired bumps
```

```bash
# Recursion
pnpm -r update                            # update across all workspace packages
pnpm -r up --latest                       # bring whole repo to latest
pnpm -r up --filter './apps/*'            # only matching workspace
```

```bash
# Depth control
pnpm up --depth 9999                      # update transitive deps too (lockfile rewrite)
pnpm up --depth 0                          # update only direct deps (default behavior)
```

```bash
# Common upgrade flows
pnpm up '@types/*' --latest                                            # safe TS type bump
pnpm up --latest --interactive --filter @myorg/web                     # surgical major bump
pnpm up --latest --recursive --filter '...[origin/main]'               # update all changed pkgs
pnpm install --no-frozen-lockfile && pnpm dedupe                       # clean up after big bump
```

## Outdated

```bash
pnpm outdated                              # show outdated deps in current package
pnpm outdated --long                       # extra columns: deprecated, dependency type
pnpm outdated --recursive                  # entire workspace
pnpm outdated -r --filter './apps/*'       # workspace subset
pnpm outdated --compatible                 # only ones that are compatible per semver
pnpm outdated --no-table                   # plain text
pnpm outdated --format json                # JSON output (for CI)
pnpm outdated --format list                # list output
pnpm outdated --format table               # default
pnpm outdated lodash react                 # filter by name
pnpm outdated 'eslint*'                    # glob filter
pnpm outdated --depth 9999                 # include transitive deps
```

```bash
# Sample table output
# Package           Current   Wanted   Latest   Dependency type
# lodash             4.17.20   4.17.21  4.17.21  dependencies
# eslint             8.57.0    8.57.0   9.13.0   devDependencies
```

```bash
# Pipe into upgrades
pnpm outdated --format json | jq -r 'keys[]' | xargs pnpm up --latest
pnpm outdated -r --format json | jq 'to_entries[] | select(.value.latest != .value.current)'
```

## Audit

```bash
pnpm audit                                 # report vulnerabilities for the resolved tree
pnpm audit --prod                          # only production deps
pnpm audit --dev                           # only dev deps
pnpm audit --json                          # machine-readable
pnpm audit --audit-level low               # threshold: low | moderate | high | critical
pnpm audit --audit-level moderate
pnpm audit --audit-level high
pnpm audit --audit-level critical
pnpm audit --ignore-registry-errors        # treat registry errors as 0 vulns (NOT recommended)
pnpm audit --no-audit-level
pnpm audit --recursive                     # whole workspace
pnpm audit --filter @myorg/api             # subset
pnpm audit --fix                           # write override entries to package.json
```

```bash
# Exit codes
0   # no vulns at or above threshold
1   # vulns found
129 # registry error (override with --ignore-registry-errors)
```

```bash
# Manual override after audit
# package.json
{
  "pnpm": {
    "overrides": {
      "minimatch@<3.0.5": ">=3.0.5",
      "ws@<8.17.1": ">=8.17.1",
      "follow-redirects@<1.15.6": ">=1.15.6"
    },
    "auditConfig": {
      "ignoreCves": ["CVE-2023-44270"],
      "ignoreGhsas": ["GHSA-xxxx-yyyy-zzzz"]
    }
  }
}
```

```bash
# CI gate
pnpm audit --prod --audit-level=high && echo OK || exit 1
```

## Workspaces

```bash
# pnpm-workspace.yaml at repo root (sibling of package.json)
packages:
  - 'apps/*'
  - 'libs/*'
  - 'packages/*'
  - 'tools/*'
  - '!**/test-fixtures/**'        # negation
```

```bash
# Project layout
my-monorepo/
├── package.json
├── pnpm-workspace.yaml
├── pnpm-lock.yaml                # ONE lockfile for the entire workspace
├── apps/
│   ├── web/package.json          # name: "@myorg/web"
│   └── api/package.json          # name: "@myorg/api"
└── libs/
    ├── ui/package.json           # name: "@myorg/ui"
    └── utils/package.json        # name: "@myorg/utils"
```

```bash
# Cross-package deps with workspace: protocol
# apps/web/package.json
{
  "name": "@myorg/web",
  "dependencies": {
    "@myorg/ui":    "workspace:*",   # any version present in workspace
    "@myorg/utils": "workspace:^",   # caret of current workspace version
    "react":        "^18.3.1"
  },
  "devDependencies": {
    "@myorg/types": "workspace:~"    # tilde semver
  }
}
```

```bash
# Variants
"workspace:*"          # ANY workspace version
"workspace:^"          # caret-pinned to current workspace version
"workspace:~"          # tilde
"workspace:^1.2.3"     # caret of specific version (must match local)
"workspace:1.2.3"      # exact (still resolves to local symlink)
```

```bash
# Publish behavior — workspace: gets rewritten to a real semver
# By default at publish time:
#   workspace:^ -> ^1.2.3
#   workspace:~ -> ~1.2.3
#   workspace:* -> 1.2.3
# Override with .npmrc:
#   prefer-workspace-packages=true
#   link-workspace-packages=deep
#   save-workspace-protocol=rolling | true | false
#   publishConfig.preserveWorkspaceProtocol=false   # keeps "workspace:" raw (not recommended)
```

```bash
# Initialize a new workspace
mkdir myrepo && cd myrepo
pnpm init
printf "packages:\n  - 'apps/*'\n  - 'libs/*'\n" > pnpm-workspace.yaml
mkdir apps libs
pnpm -F nope init                        # creates a sub-package interactively
```

```bash
# Enumerate
pnpm -r ls                               # list workspace packages
pnpm m ls                                # alias
pnpm -r ls --json                        # JSON, useful for scripts
pnpm -r ls --depth -1                    # only top-level workspace projects
```

## Workspace Filters

```bash
# Filter by name
pnpm --filter @myorg/web build
pnpm -F @myorg/web build                 # short
pnpm -F '@myorg/*' build                 # glob over names
```

```bash
# Filter by path
pnpm --filter './apps/*' test
pnpm --filter './apps/**' test
pnpm --filter '!./test-fixtures/**' run lint
```

```bash
# Change-aware (git-diff)
pnpm --filter '...[origin/main]' build           # changed since origin/main + dependents
pnpm --filter '[origin/main]' test               # ONLY changed packages
pnpm --filter '...[HEAD~1]' lint                 # against last commit
pnpm --filter '...[HEAD~3..HEAD]' build          # range
pnpm -F '...{apps/web}[origin/main]' build       # path glob + git
```

```bash
# Topology selectors
pnpm --filter "@myorg/web..."           # @myorg/web AND its dependents (downstream)
pnpm --filter "...@myorg/utils"         # @myorg/utils AND its dependencies (upstream)
pnpm --filter "...^@myorg/ui"           # ONLY direct ancestors of @myorg/ui (changed-aware)
pnpm --filter "@myorg/web..."           # web + everything web depends on transitively
pnpm --filter "@myorg/web^..."          # web + just direct dependents
```

```bash
# Combine
pnpm --filter "@myorg/web..." --filter "!@myorg/legacy" build

# Cap concurrency
pnpm --workspace-concurrency=4 -r run build       # at most 4 parallel
pnpm -r --parallel --workspace-concurrency=8 run dev

# Negate
pnpm --filter '!./apps/legacy' run lint
```

```bash
# Inspect what a filter resolves to without running
pnpm -r --filter '...[origin/main]' ls --depth -1 --json
```

## Recursive Commands

```bash
pnpm -r run build                          # run "build" in EVERY workspace package that defines it
pnpm -r run test --if-present              # skip pkgs without "test" script
pnpm -r --parallel run dev                 # all dev servers in parallel
pnpm -r --stream run build                 # interleave logs prefixed by package name
pnpm -r --aggregate-output run build       # buffer + flush per-package
pnpm -r --include-workspace-root run lint  # also run in root package.json
pnpm -r exec eslint .                      # exec a binary, not an npm script
pnpm -r exec rm -rf dist                   # any shell command
pnpm -r --filter '...[origin/main]' run build      # change-aware build
pnpm -r --workspace-concurrency=1 run migrate      # serialize when needed
pnpm -r --reporter=append-only run build           # CI-friendly logs
pnpm -r --no-bail run test                          # run all even if one fails
pnpm -r --bail run test                             # stop on first failure (default)
```

```bash
# m / multi alias
pnpm m run build         # same as -r
pnpm multi run build
```

```bash
# Recursive with env
NODE_OPTIONS=--max-old-space-size=4096 pnpm -r run build
TURBO_TOKEN=$T pnpm -r --filter '@myorg/*' run ci
```

## Run Scripts

```bash
pnpm run build                              # explicit form
pnpm build                                   # implicit shorthand (works because there's no built-in build cmd)
pnpm test                                    # built-in shortcut
pnpm start                                   # built-in shortcut
pnpm run                                     # list all scripts
pnpm run --if-present custom:script          # silent no-op if script missing
```

```bash
# Pre/post hooks (npm-compatible)
# package.json
{
  "scripts": {
    "prebuild":  "rm -rf dist",
    "build":     "tsc",
    "postbuild": "node scripts/copy-assets.js"
  }
}
# pnpm pre/post are OFF by default since pnpm 7+; enable via .npmrc:
# enable-pre-post-scripts=true
```

```bash
# Pass arguments through to the script
pnpm run test -- --watch                    # forward "--watch" to underlying command
pnpm test -- --coverage --reporter=verbose
pnpm vitest -- --run                        # works for any registered bin
```

```bash
# Differences vs npm run
# - pnpm runs scripts with the package's local node_modules/.bin first, then workspace bins
# - pnpm exposes workspace siblings on PATH automatically
# - pnpm has --if-present built-in (npm has it too since 7.x but pnpm honors it everywhere)
# - pnpm runs sequentially by default; npm has --parallel only inside "npm exec"
# - pnpm 9+: pre/post scripts disabled by default (security)
```

```bash
# Recursive run with topo order
pnpm -r run build                            # respects package dependency graph (deps first)
pnpm -r --no-sort run build                  # alphabetical, ignore graph
pnpm -r --reverse run cleanup                # reverse topo
```

```bash
# Working directory and aliases
pnpm -C ./apps/web run dev                   # change dir before run
pnpm --dir=./apps/web run dev                # long form of -C
```

## Exec / dlx

```bash
# Run a binary already in node_modules/.bin or workspace bin
pnpm exec eslint .
pnpm exec tsc --noEmit
pnpm exec vitest run
pnpm exec -- prettier --check .              # `--` to stop pnpm flag parsing
```

```bash
# dlx — ephemeral install + run, like npx
pnpm dlx create-vite my-app --template react-ts
pnpm dlx create-next-app@latest my-app
pnpm dlx tsc --init
pnpm dlx eslint --init
pnpm dlx serve ./dist
pnpm dlx http-server -p 8080
```

```bash
# dlx flags
pnpm dlx --package=create-vite create-vite my-app
pnpm dlx --silent serve ./dist
pnpm dlx --use-node-version=20.10.0 cmd
PNPM_HOME=/tmp/pnpm-temp pnpm dlx some-cli   # isolate temp store
```

```bash
# Cache reuse
# dlx caches into the global store like any other install:
#   ~/.local/share/pnpm/store/v3/
# Subsequent runs of the same package + version are essentially instant.
# Force a refresh:
pnpm dlx --no-prefer-offline create-vite my-app
```

```bash
# Equivalent npm/yarn surface
# pnpm exec  ~ npx --no-install / yarn run
# pnpm dlx   ~ npx (with install) / yarn dlx
```

## Patches

```bash
# Open a temp copy of the package for editing
pnpm patch lodash@4.17.21
# pnpm prints something like:
# You can now edit the following folder: /tmp/abc1234567/node_modules/lodash
# Once you're done with your changes, run "pnpm patch-commit '/tmp/abc1234567'"

# Edit files in that folder, then commit
pnpm patch-commit /tmp/abc1234567
# pnpm writes:  patches/lodash@4.17.21.patch
# and updates package.json:
#   "pnpm": {
#     "patchedDependencies": {
#       "lodash@4.17.21": "patches/lodash@4.17.21.patch"
#     }
#   }
```

```bash
# Inspect / re-apply patches
ls patches/
pnpm install            # re-applies patches automatically
pnpm patch-remove lodash@4.17.21       # remove patch entry
```

```bash
# Patch flags
pnpm patch --edit-dir ./tmp/lodash-patch lodash@4.17.21      # explicit dir
pnpm patch --ignore-existing lodash@4.17.21                  # discard existing patch
```

```bash
# Patch shape (unified diff)
diff --git a/index.js b/index.js
index 1234567..89abcde 100644
--- a/index.js
+++ b/index.js
@@ -10,7 +10,7 @@ var lodash = (function () {
-  var FOO = 'old';
+  var FOO = 'new';
```

```bash
# Workspace-aware patches
# pnpm-workspace.yaml or root package.json defines patches; they apply across all packages
# in the workspace that resolve to the patched version.
```

## Overrides

```bash
# package.json — pin transitive dep versions across the entire dep graph
{
  "pnpm": {
    "overrides": {
      "minimist": "^1.2.8",
      "lodash@<4.17.21": "^4.17.21",
      "follow-redirects": "^1.15.6",
      "axios>follow-redirects": "^1.15.6",
      "react@^17": "^17.0.2",
      "@types/node": "20.x"
    }
  }
}
```

```bash
# Selector grammar
# "pkg":             every occurrence
# "pkg@<range>":     only when the resolved version satisfies <range>
# "parent>child":    only when the parent depends on the child
# "parent>child@<range>": parent + child range together
# "@scope/pkg":      scoped pkg
# Empty string keys are illegal.
```

```bash
# vs npm overrides
# npm overrides:
#   "overrides": { "minimist": "^1.2.8" }
# pnpm reads npm-style overrides too, but pnpm.overrides wins on conflict.
# pnpm supports parent>child syntax; npm does the same since 8.3.
# pnpm has additional patchedDependencies + neverBuiltDependencies hooks.
```

```bash
# Other pnpm-only fields next to overrides
{
  "pnpm": {
    "overrides": { ... },
    "packageExtensions": {
      "react-router@^6": {
        "peerDependencies": { "react": "^18 || ^19" }
      }
    },
    "peerDependencyRules": {
      "ignoreMissing": ["@babel/*"],
      "allowedVersions": { "react": "18 || 19" },
      "allowAny": ["eslint"]
    },
    "neverBuiltDependencies": ["fsevents"],
    "onlyBuiltDependencies": ["esbuild", "sharp"],
    "allowedDeprecatedVersions": { "request": "*" }
  }
}
```

```bash
# Apply changes (must reinstall)
pnpm install
pnpm install --force         # re-resolve everything if overrides changed deeply
```

## Peer Dependency Rules

```bash
# Defaults (pnpm 7.13+)
auto-install-peers=true                # missing peers will be auto-added
strict-peer-dependencies=false         # warn instead of error
dedupe-peer-dependents=true            # one copy of each peer per dep tree
resolve-peers-from-workspace-root=true # workspace root wins for peer resolution
```

```bash
# .npmrc tweaks
auto-install-peers=false               # disable auto add
strict-peer-dependencies=true          # FAIL on unmet/incompatible peers
strict-peer-dependencies=false         # WARN only
```

```bash
# Per-rule overrides in package.json
{
  "pnpm": {
    "peerDependencyRules": {
      "ignoreMissing": ["@babel/*", "webpack"],
      "allowedVersions": {
        "react": "18 || 19",
        "react-dom": "18 || 19",
        "@types/react": "*"
      },
      "allowAny": ["eslint"]
    }
  }
}
```

```bash
# CLI on the fly
pnpm install --strict-peer-dependencies
pnpm install --no-strict-peer-dependencies
pnpm install --no-auto-install-peers
pnpm install --no-dedupe-peer-dependents
```

```bash
# Common peer fix workflow
# 1. pnpm install fails with ERR_PNPM_PEER_DEP_ISSUES
# 2. pnpm why <peer-pkg>          -> see who needs which version
# 3. add peerDependencyRules.allowedVersions or upgrade the peer
# 4. pnpm install --force         -> rebuild graph
```

## Hoisting / Public Hoist

```bash
# .npmrc keys
node-linker=isolated          # default: strict, isolated layout
node-linker=hoisted           # flat node_modules (npm-like)
node-linker=pnp               # Yarn-like Plug'n'Play (.pnp.cjs)
hoist=true                    # hoist non-public deps into .pnpm/node_modules (default true)
hoist=false                   # disable internal hoisting
hoist-pattern[]=*types*       # which transitive names get hoisted (default '*')
public-hoist-pattern[]=*eslint*   # additionally hoist these to top-level node_modules
public-hoist-pattern[]=*prettier*
shamefully-hoist=true         # SHORTCUT: public-hoist-pattern[]=*  (full flat tree)
shamefully-hoist=false        # default
shamefully-hoist=true is equivalent to public-hoist-pattern[]=*
```

```bash
# Symlink resolution algorithm (simplified)
# require('lodash') from apps/web/src/index.ts:
#   1. apps/web/node_modules/lodash -> .pnpm/lodash@4.17.21/node_modules/lodash (symlink)
#   2. lodash's own require() looks inside ITS .pnpm dir, NOT the project root
#   3. so lodash can ONLY see its own declared deps (strict isolation)
# This is why phantom deps fail under pnpm.
```

```bash
# Examples
public-hoist-pattern[]=eslint*
public-hoist-pattern[]=*prettier*
public-hoist-pattern[]=tslib            # makes tslib visible to non-deps (sometimes needed)

# Apply:
pnpm install
ls node_modules/eslint                    # should now exist at top level
```

```bash
# When to choose each linker
# isolated  - default; strictest; best for libraries; smallest disk
# hoisted   - tools that scan node_modules naively (Next.js standalone, some bundlers, electron-builder)
# pnp       - Yarn berry compatibility; smallest disk; needs PnP-aware tooling
```

## Lockfile

```bash
pnpm-lock.yaml                  # the canonical lockfile, COMMIT it
node_modules/.pnpm/lock.yaml    # per-project integrity snapshot (do NOT commit)
.modules.yaml                   # snapshot of options used at install
```

```bash
# Top-level shape
lockfileVersion: '9.0'
settings:
  autoInstallPeers: true
  excludeLinksFromLockfile: false
importers:
  .:
    dependencies:
      lodash: { specifier: ^4.17.21, version: 4.17.21 }
    devDependencies: { ... }
  apps/web:
    dependencies: { ... }
packages:
  lodash@4.17.21:
    resolution: { integrity: sha512-... }
    engines: { node: '>=4' }
snapshots:
  lodash@4.17.21:
    {}
```

```bash
# Sections
# importers/   workspace projects -> their direct deps with specifier+resolved
# packages/    package metadata: integrity, engines, peerDeps, license
# snapshots/   per-resolution-context snapshot (peer deps included)
# settings/    pnpm options that affect lockfile shape
```

```bash
# Integrity hashes
# - sha512 base64 (subresource integrity format)
# - mismatched hash = pnpm install error ERR_PNPM_INTEGRITY_CHECK_FAILED
# - to refresh: rm pnpm-lock.yaml && pnpm install
```

```bash
# When to commit
# ALWAYS commit pnpm-lock.yaml in libraries and apps.
# pnpm-lock.yaml goes in source control, NEVER pnpm-lock.yaml.gz, NEVER node_modules/.
```

```bash
# Frozen-lockfile semantics
# pnpm install --frozen-lockfile:
#   - read pnpm-lock.yaml
#   - if any importer's specifier doesn't match locked version: ERR_PNPM_OUTDATED_LOCKFILE
#   - DO NOT modify pnpm-lock.yaml under any circumstance
#   - DEFAULT in CI (when CI=true env var)
# pnpm install --no-frozen-lockfile:
#   - update lockfile to satisfy specifiers; develop locally
```

```bash
# Diagnose lockfile drift
pnpm install --frozen-lockfile             # CI-style verify
pnpm install --lockfile-only               # only update lockfile, no node_modules
pnpm dedupe                                 # remove redundant entries
pnpm install --fix-lockfile                 # rewrite stale snapshots/peers
```

```bash
# Older lockfile versions
# 5.x ->  v5.4   (pnpm 6)
# 6.x ->  v6     (pnpm 7)
# 7.x ->  v7     (pnpm 8)
# 9.x ->  v9.0   (pnpm 9+)
# Upgrading pnpm major usually rewrites the lockfile; commit the rewrite.
```

## .npmrc / config

```bash
pnpm config get <key>
pnpm config set <key> <value>
pnpm config delete <key>
pnpm config list                              # ALL active values, with origin
pnpm config list --json
pnpm config list --location=project
pnpm config list --location=user
pnpm config list --location=global
```

```bash
# Cascade order (last wins):
# 1. project   ./.npmrc          (and ./.pnpmrc — DEPRECATED, use .npmrc)
# 2. user      ~/.npmrc
# 3. global    $PREFIX/etc/npmrc
# 4. builtin   shipped with pnpm
# .pnpmrc was deprecated in pnpm 6+; use .npmrc.
```

```bash
# Common keys (.npmrc)
registry=https://registry.npmjs.org/
@myorg:registry=https://npm.pkg.github.com/
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
always-auth=true
strict-ssl=true
ca=/etc/ssl/cert.pem
cafile=/etc/ssl/cert.pem
fetch-retries=2
fetch-retry-mintimeout=10000
fetch-retry-maxtimeout=60000
network-concurrency=16
fetch-timeout=60000
proxy=http://proxy.corp:3128
https-proxy=http://proxy.corp:3128
noproxy=.corp,localhost,127.0.0.1
```

```bash
# pnpm-specific keys
store-dir=~/.local/share/pnpm/store
cache-dir=~/.cache/pnpm
state-dir=~/.local/state/pnpm
virtual-store-dir=node_modules/.pnpm
package-import-method=auto                  # auto | hardlink | copy | clone
node-linker=isolated                        # isolated | hoisted | pnp
auto-install-peers=true
strict-peer-dependencies=false
hoist=true
public-hoist-pattern[]=*types*
shamefully-hoist=false
side-effects-cache=true
side-effects-cache-readonly=false
shared-workspace-lockfile=true
link-workspace-packages=deep                # true | false | deep
prefer-workspace-packages=false
save-workspace-protocol=rolling             # true | false | rolling
recursive-install=true
ignore-workspace-root-check=false
manage-package-manager-versions=true
auto-install-peers=true
dedupe-peer-dependents=true
enable-pre-post-scripts=false               # default false on 9+
```

```bash
# Per-project override
echo "shamefully-hoist=true" >> .npmrc
echo "registry=https://internal.example.com/" >> .npmrc
echo "@myorg:registry=https://npm.pkg.github.com" >> .npmrc
echo "//npm.pkg.github.com/:_authToken=\${GITHUB_TOKEN}" >> .npmrc
```

```bash
# Env-var equivalents (uppercase + underscore)
# .npmrc: store-dir=...     -> NPM_CONFIG_STORE_DIR=...
# .npmrc: registry=...      -> NPM_CONFIG_REGISTRY=...
# .npmrc: cache-dir=...     -> NPM_CONFIG_CACHE_DIR=...
NPM_CONFIG_STORE_DIR=/data/pnpm-store pnpm install
NPM_CONFIG_REGISTRY=https://internal pnpm install
```

## Registries / Auth

```bash
# Single registry
registry=https://registry.npmjs.org/
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
```

```bash
# Per-scope routing
@myorg:registry=https://npm.pkg.github.com
@types:registry=https://registry.npmjs.org
@private:registry=https://npm.internal.example.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
//npm.internal.example.com/:_authToken=${INTERNAL_TOKEN}
```

```bash
# Always send auth (private registries that require it for tarball downloads)
always-auth=true
//registry.npmjs.org/:always-auth=true
```

```bash
# Login flows
pnpm login                                 # login to default registry
pnpm login --registry=https://npm.pkg.github.com --scope=@myorg
pnpm logout
pnpm logout --registry=https://npm.pkg.github.com
pnpm whoami
pnpm whoami --registry=https://npm.pkg.github.com
```

```bash
# Token environment variables (commonly read by tooling)
export NPM_TOKEN=abc123
export NODE_AUTH_TOKEN=abc123
export GITHUB_TOKEN=ghp_xxx
```

```bash
# Self-hosted (Verdaccio, Sonatype Nexus, JFrog Artifactory, GitHub Packages)
# Verdaccio:
registry=https://verdaccio.corp/
//verdaccio.corp/:_authToken=${VERDACCIO_TOKEN}

# Artifactory:
registry=https://artifactory.corp/api/npm/npm-virtual/
//artifactory.corp/api/npm/npm-virtual/:_authToken=${ARTI_TOKEN}
//artifactory.corp/api/npm/npm-virtual/:always-auth=true

# GitHub Packages:
@myorg:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```bash
# Mirror / replace
replaceRegistryHost=true                     # replace registry hostname in lockfile resolution URLs
strict-ssl=false                             # only for self-signed mirrors (use cafile instead in prod)
cafile=/etc/ssl/internal-ca.pem
```

```bash
# Verify routing
pnpm config get @myorg:registry
pnpm view @myorg/utils                       # makes a real call; will 401 if auth wrong
pnpm view @myorg/utils versions
```

## Fetching / Tarballs

```bash
# pnpm fetch — populate the local store from pnpm-lock.yaml WITHOUT touching node_modules
pnpm fetch                                # fetch every dep
pnpm fetch --prod                         # only production deps
pnpm fetch --dev                          # only dev deps
pnpm fetch --filter @myorg/api            # subset
```

```bash
# Layered Docker build pattern (cache-friendly)
# Dockerfile
# syntax=docker/dockerfile:1.7
FROM node:20.10-bookworm AS base
RUN corepack enable && corepack prepare pnpm@9.12.3 --activate
WORKDIR /app

FROM base AS deps
COPY pnpm-lock.yaml package.json pnpm-workspace.yaml ./
COPY apps apps
COPY libs libs
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm fetch --prod

FROM base AS prod
COPY --from=deps /app /app
COPY . .
RUN pnpm install --offline --prod --frozen-lockfile
RUN pnpm -r --filter @myorg/api run build
CMD ["node", "apps/api/dist/index.js"]
```

```bash
# pnpm install with offline tarballs
pnpm install --offline --prefer-offline --frozen-lockfile
# Together: never hit network, error if any tarball is missing from store.
```

```bash
# Manual store warm-up
pnpm store add lodash@4.17.21              # download a single package into the store
pnpm store add 'react@18.x'
pnpm store prune                            # GC untracked store entries
pnpm store status                           # integrity report
```

## create-* / init

```bash
pnpm init                                    # generate package.json
pnpm init --yes                              # accept all defaults

# Bootstrappers via dlx (equivalent of npx create-X)
pnpm create vite my-app                      # -> pnpm dlx create-vite
pnpm create vite my-app --template react-ts
pnpm create next-app my-app --typescript --tailwind
pnpm create react-app my-app                 # legacy, still works
pnpm create svelte@latest my-app
pnpm create astro@latest
pnpm create t3-app@latest
pnpm create remix@latest
pnpm create expo my-rn-app
pnpm create vue@latest my-vue-app
pnpm create solid@latest
pnpm create qwik@latest
```

```bash
# All create-* commands route through dlx; they reuse the global store.
# Equivalences:
# pnpm create vite        ===  pnpm dlx create-vite
# npm  init  vite         ===  npm  exec  create-vite
# yarn create vite        ===  yarn dlx  create-vite
```

## Why?

```bash
pnpm why <pkg>                              # show every path that pulls in <pkg>
pnpm why react                              # full graph
pnpm why react --recursive                  # across the whole workspace
pnpm why react --json                       # machine-readable
pnpm why -D vitest                          # only consider devDependencies path
pnpm why react --filter @myorg/web          # scoped to one workspace package
pnpm why react --depth=Infinity             # all the way down
pnpm why react --long                       # extra info per node
```

```bash
# Sample output
# @myorg/web@1.0.0 /repo/apps/web
# └─┬ @myorg/ui@workspace:^1.0.0
#   └─┬ react@18.3.1
#     └── peer @myorg/web > react ^18.3.1
```

```bash
# Companions
pnpm list react                              # tree of where react is installed
pnpm list react --depth Infinity
pnpm list --parseable                        # shell-friendly paths
pnpm view react                              # metadata from registry
pnpm view react@18.3.1
pnpm view react versions --json
pnpm view react dist-tags
```

## Migrate

```bash
# from npm
rm -rf node_modules package-lock.json
pnpm import          # reads package-lock.json (yes, even after rm? backup first)
# Or, import BEFORE deleting:
pnpm import          # writes pnpm-lock.yaml from package-lock.json
rm package-lock.json
pnpm install
```

```bash
# from yarn classic (yarn.lock)
pnpm import          # reads yarn.lock
rm yarn.lock
pnpm install
```

```bash
# from yarn berry (yarn.lock v6/v8)
pnpm import          # works for berry yarn.lock too
# If the project uses .yarnrc.yml + .pnp.cjs, also:
rm -rf .yarn .pnp.cjs .pnp.loader.mjs .yarnrc.yml
pnpm install
```

```bash
# Commit hooks / scripts to migrate too
# Replace in scripts:
#   "yarn"            ->  "pnpm"
#   "yarn install"    ->  "pnpm install"
#   "yarn add"        ->  "pnpm add"
#   "yarn workspaces foreach"  ->  "pnpm -r"
#   "npm exec"        ->  "pnpm exec"
#   "npx"             ->  "pnpm dlx"
sed -i.bak 's/yarn /pnpm /g; s/npx /pnpm dlx /g' .github/workflows/*.yml
```

```bash
# Workspace migration:
# package.json "workspaces" -> pnpm-workspace.yaml
#   {
#     "workspaces": ["apps/*","libs/*"]
#   }
# becomes
#   packages:
#     - 'apps/*'
#     - 'libs/*'
# AND pnpm reads root package.json's "workspaces" key as a fallback if pnpm-workspace.yaml is absent.
```

```bash
# Sanity checks after migration
pnpm install --frozen-lockfile
pnpm -r run build
pnpm -r run test
pnpm doctor
```

## Common Errors

```bash
# 1) Unmet peer deps
ERR_PNPM_PEER_DEP_ISSUES  Unmet peer dependencies
# Cause: a transitive package declares a peer that nothing else provides.
# Fix:
pnpm why <peer-pkg>
pnpm add <peer-pkg>@<acceptable-version>
# OR add to peerDependencyRules.allowedVersions or peerDependencyRules.ignoreMissing
```

```bash
# 2) Lockfile out of date in CI
ERR_PNPM_OUTDATED_LOCKFILE  Cannot install with "frozen-lockfile" because pnpm-lock.yaml is not up to date with package.json
# Cause: someone edited package.json but didn't run install locally before pushing.
# Fix locally:
pnpm install
git add pnpm-lock.yaml
git commit -m "chore: refresh lockfile"
```

```bash
# 3) Version not on registry
ERR_PNPM_NO_MATCHING_VERSION  No matching version found for <pkg>@<spec>
# Cause: tag changed, package unpublished, scoped registry routing wrong.
# Fix:
pnpm view <pkg> versions --json
pnpm view <pkg> dist-tags
# Update package.json with a real version OR check @scope:registry routing.
```

```bash
# 4) 404 / package not found
ERR_PNPM_FETCH_404  GET https://registry.npmjs.org/<pkg>: Not Found
# Cause: typo, missing scope, registry mirror missing the package, or auth required.
# Fix:
pnpm view <pkg>
cat .npmrc          # check @scope:registry and tokens
pnpm whoami --registry=...
```

```bash
# 5) Registry mismatch
ERR_PNPM_REGISTRIES_MISMATCH  Some packages were installed from a different registry than recorded in the lockfile
# Cause: lockfile resolved against registry A; current .npmrc points at registry B.
# Fix:
pnpm install --force                     # re-resolve against current registry
# OR set replaceRegistryHost=true and re-run install.
```

```bash
# 6) Missing package.json
ENOENT: no such file or directory, open '.../package.json'
# Cause: cwd is outside any package; `pnpm install` needs a package.json.
# Fix:
pnpm init
# Or run the command in the right directory:
pnpm -C ./apps/web install
```

```bash
# 7) Disk full from store growth
ENOSPC: no space left on device, open '/.../store/v3/files/...'
# Cause: store accumulates blobs from every project that ever installed.
# Fix:
pnpm store prune                              # GC unreferenced blobs
pnpm store status                             # report integrity
df -h
# Move store to bigger disk:
pnpm config set store-dir /data/pnpm-store
pnpm install
```

```bash
# 8) Module not found after pulling code
Error: Cannot find module 'X'
# Cause: pulled new package.json/lockfile but didn't reinstall.
# Fix:
pnpm install
```

```bash
# 9) Cannot find module — hoisted vs isolated mismatch
Error: Cannot find module 'react' (resolved by tool that scans top-level node_modules)
# Cause: a tool (electron-builder, Next.js standalone, jest with old config) expects flat tree.
# Fix:
echo "shamefully-hoist=true" >> .npmrc      # full flat
# OR more surgical:
echo "public-hoist-pattern[]=react" >> .npmrc
echo "public-hoist-pattern[]=react-dom" >> .npmrc
pnpm install
```

```bash
# 10) Integrity failure
ERR_PNPM_INTEGRITY_CHECK_FAILED  Integrity check failed
# Cause: store blob corrupted or lockfile hash differs from registry.
# Fix:
rm -rf node_modules
pnpm store prune
pnpm install --force
```

```bash
# 11) Workspace cycle
ERR_PNPM_RECURSIVE_RUN_FIRST_FAIL  cyclic dependency
# Cause: package A depends on B, B depends on A.
# Fix:
pnpm -r ls --depth -1                         # find cycle
# Refactor to break cycle, OR add --ignore-workspace-cycles to bypass during dev.
```

```bash
# 12) Engine mismatch
ERR_PNPM_UNSUPPORTED_ENGINE  Unsupported engine
# Cause: package requires Node X but current Node is Y.
# Fix:
nvm use 20
node --version
pnpm install
```

```bash
# 13) Workspace package not linked
ERR_PNPM_WORKSPACE_PKG_NOT_FOUND  In workspace: <pkg> requires <local-pkg>@workspace:*, but no such package exists
# Cause: the referenced workspace package's name in its package.json doesn't match the import.
# Fix:
grep '"name"' libs/*/package.json
# Make sure the name field matches the workspace: spec.
```

```bash
# 14) Auth required / 401
ERR_PNPM_FETCH_401  Forbidden
# Cause: missing or expired auth token for a private scope.
# Fix:
echo "//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}" >> .npmrc
pnpm whoami --registry=https://npm.pkg.github.com
pnpm install
```

```bash
# 15) Network EAI_AGAIN / ETIMEDOUT
ERR_PNPM_FETCH_REQUEST_TIMEOUT  request timed out
# Cause: flaky network or slow proxy.
# Fix:
pnpm config set fetch-timeout 120000
pnpm config set fetch-retries 5
pnpm config set network-concurrency 4
```

## Common Gotchas

```bash
# Gotcha 1: phantom dep that worked under npm now fails under pnpm
# Symptom:
#   Error: Cannot find module 'lodash' from 'apps/web/...'
# Broken (worked accidentally under npm):
#   require('lodash')          # but lodash isn't in apps/web/package.json
# Fixed:
pnpm add lodash --filter @myorg/web
```

```bash
# Gotcha 2: a CLI binary from a transitive dep isn't on PATH
# Broken:
pnpm exec eslint        # "eslint" not found
# Fixed: list eslint as a direct dep
pnpm add -D eslint
# Or hoist publicly:
echo "public-hoist-pattern[]=*eslint*" >> .npmrc
pnpm install
```

```bash
# Gotcha 3: CI runs `pnpm install` without --frozen-lockfile and silently rewrites the lockfile
# Broken:
#   .github/workflows/ci.yml
#   - run: pnpm install
# Fixed:
#   - run: pnpm install --frozen-lockfile
# Note: pnpm 8+ auto-enables --frozen-lockfile when CI=true. Verify your CI sets CI=true.
```

```bash
# Gotcha 4: workspace dep referenced without workspace: protocol
# Broken: apps/web/package.json
#   "@myorg/utils": "1.0.0"            # resolves to public registry, not local
# Fixed:
#   "@myorg/utils": "workspace:*"      # forces local link
pnpm install
```

```bash
# Gotcha 5: postinstall scripts run nothing on pnpm 9+
# Broken:
#   "postinstall": "node scripts/setup.js"     # silently skipped
# Cause: pnpm 9 disables lifecycle scripts of dependencies by default for security.
# Fixed:
# package.json:
{
  "pnpm": {
    "onlyBuiltDependencies": ["esbuild", "sharp", "@prisma/client"]
  }
}
# Or run an explicit:
pnpm rebuild esbuild sharp
```

```bash
# Gotcha 6: postinstall in YOUR OWN package.json works, but pre/post hooks for your scripts don't
# Broken:
#   "scripts": { "prebuild": "...", "build": "...", "postbuild": "..." }
#   pnpm run build       # only runs "build", skips pre/post on pnpm 7+
# Fixed:
# .npmrc:
echo "enable-pre-post-scripts=true" >> .npmrc
pnpm run build           # now runs prebuild + build + postbuild
```

```bash
# Gotcha 7: hardlinks share inodes — editing node_modules edits the global store
# Broken (rage-debugging):
#   sed -i 's/foo/bar/' node_modules/some-pkg/index.js
#   # Now ALL projects on the machine see "bar"
# Fixed:
pnpm patch some-pkg@1.2.3
# Edit in the temp dir, then:
pnpm patch-commit /tmp/abc1234
# Or, force per-project copies:
echo "package-import-method=copy" >> .npmrc
```

```bash
# Gotcha 8: pnpm exec vs pnpm run pass-through args
# Broken:
pnpm exec vitest --watch        # might be parsed as pnpm flag in some shells
# Fixed:
pnpm exec -- vitest --watch     # `--` ends pnpm flag parsing
pnpm vitest -- --watch          # alternative
```

```bash
# Gotcha 9: store on a different filesystem than the project
# Symptom: fallback to copy instead of hardlink, slower installs, more disk
# Broken:
#   store on /home (ext4) but project on /mnt/data (ntfs/exfat/different fs)
# Fixed: put the store on the SAME filesystem as the project.
pnpm config set store-dir /mnt/data/.pnpm-store
pnpm install
```

```bash
# Gotcha 10: scoped private registry fails inside Docker because tokens aren't passed
# Broken:
#   .npmrc references ${NPM_TOKEN} but Docker build doesn't inherit env
# Fixed:
DOCKER_BUILDKIT=1 docker build --secret id=npmrc,src=$HOME/.npmrc .
# Dockerfile:
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc pnpm install --frozen-lockfile
```

```bash
# Gotcha 11: lockfile merge conflict
# Broken: git merge gives conflict markers in pnpm-lock.yaml
# Fixed:
git checkout --theirs pnpm-lock.yaml         # take their version
pnpm install                                  # re-resolve against your package.json
git add pnpm-lock.yaml
git commit
# OR use pnpm 9+ built-in:
pnpm install --fix-lockfile
```

```bash
# Gotcha 12: --filter doesn't match because workspace names have changed case
# Broken:
pnpm --filter @MyOrg/web build              # case-sensitive miss
# Fixed:
pnpm --filter @myorg/web build
```

```bash
# Gotcha 13: `pnpm dlx pkg@latest` reuses old cache
# Broken: stale cli version persists.
# Fixed:
pnpm dlx --no-prefer-offline pkg@latest
# OR force store refresh:
pnpm store prune
```

```bash
# Gotcha 14: workspaces with shared peer (React) cause duplicate React in bundle
# Symptom: "Invalid hook call" / two-Reacts error
# Broken: two libs each declare React as a regular dep
# Fixed: declare React as a peer dep + dev dep in libs:
{
  "peerDependencies": { "react": "^18 || ^19" },
  "devDependencies":  { "react": "^18.3.1" }
}
# Then in apps:
{
  "dependencies": { "react": "^18.3.1" }
}
# pnpm 8+ default dedupe-peer-dependents=true keeps one copy.
```

```bash
# Gotcha 15: ignore-scripts hides important builds
# Broken:
pnpm install --ignore-scripts          # forgot to rebuild later
node app.js                             # native modules (sharp, esbuild) missing
# Fixed:
pnpm rebuild
pnpm rebuild sharp esbuild
```

## pnpm vs npm vs yarn

```bash
# Action            npm                              pnpm                              yarn berry
# Install all       npm install                      pnpm install                      yarn install
# CI install        npm ci                           pnpm install --frozen-lockfile    yarn install --immutable
# Add prod          npm install pkg                  pnpm add pkg                      yarn add pkg
# Add dev           npm install -D pkg               pnpm add -D pkg                   yarn add -D pkg
# Add peer          npm install --save-peer pkg      pnpm add --save-peer pkg          (no shorthand)
# Add optional      npm install -O pkg               pnpm add -O pkg                   yarn add -O pkg
# Remove            npm uninstall pkg                pnpm remove pkg                   yarn remove pkg
# Update            npm update                       pnpm update                       yarn up
# Update latest     (manual)                         pnpm up --latest                  yarn up --latest
# Outdated          npm outdated                     pnpm outdated                     yarn outdated
# Audit             npm audit                        pnpm audit                        yarn npm audit
# Audit fix         npm audit fix                    pnpm audit --fix                  (manual)
# Run script        npm run x                        pnpm run x  / pnpm x              yarn x
# Run binary        npx pkg                          pnpm exec pkg / pnpm dlx pkg      yarn dlx pkg
# Why pkg           npm explain pkg                  pnpm why pkg                      yarn why pkg
# Workspaces        npm workspaces  -ws / -w         pnpm -r / --filter                yarn workspaces foreach
# Lockfile          package-lock.json                pnpm-lock.yaml                    yarn.lock
# Publish           npm publish                      pnpm publish                      yarn npm publish
# Login             npm login                        pnpm login                        yarn npm login
# Whoami            npm whoami                       pnpm whoami                       yarn npm whoami
# Version           npm version patch                pnpm version patch                yarn version --patch
# Init              npm init                         pnpm init                         yarn init
# Create-X          npm init x  / npx create-x       pnpm create x                     yarn create x
# Global add        npm i -g pkg                     pnpm add -g pkg                   yarn dlx -p pkg
# Cache clean       npm cache clean --force          pnpm store prune                  yarn cache clean
```

```bash
# Field translation
# package.json "workspaces" (npm/yarn classic) -> pnpm-workspace.yaml (pnpm)
# package.json "resolutions" (yarn classic)    -> "pnpm.overrides" (pnpm) / "overrides" (npm)
# package.json "patchedDependencies"           -> pnpm-only field
# .yarnrc.yml                                   -> .npmrc (pnpm reuses npm's config file)
```

## Idioms

```bash
# CI: deterministic install + verify
# .github/workflows/ci.yml
- uses: pnpm/action-setup@v4
  with:
    version: 9.12.3
- uses: actions/setup-node@v4
  with:
    node-version: 20
    cache: 'pnpm'
- run: pnpm install --frozen-lockfile
- run: pnpm -r run build
- run: pnpm -r run test
```

```bash
# CI with cache hits via @actions/cache
- name: Get pnpm store path
  id: pnpm-cache
  shell: bash
  run: echo "STORE_PATH=$(pnpm store path)" >> $GITHUB_OUTPUT
- uses: actions/cache@v4
  with:
    path: ${{ steps.pnpm-cache.outputs.STORE_PATH }}
    key: pnpm-store-${{ hashFiles('**/pnpm-lock.yaml') }}
    restore-keys: pnpm-store-
```

```bash
# Layered Docker build (pnpm fetch + offline install)
FROM node:20.10-bookworm AS base
RUN corepack enable && corepack prepare pnpm@9.12.3 --activate
WORKDIR /app

FROM base AS fetch
COPY pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm fetch

FROM fetch AS install
COPY . .
RUN pnpm install --offline --frozen-lockfile

FROM install AS build
RUN pnpm -r --filter @myorg/api run build

FROM node:20.10-bookworm AS run
COPY --from=build /app /app
WORKDIR /app/apps/api
CMD ["node", "dist/index.js"]
```

```bash
# Monorepo dev — every workspace dev server in parallel
pnpm -r --parallel run dev
# Only specific subset
pnpm --parallel --filter '@myorg/web' --filter '@myorg/api' run dev

# Build only what changed since main
pnpm --filter '...[origin/main]' run build

# Test only what changed
pnpm --filter '[origin/main]' run test
```

```bash
# Pre-commit hook (Husky + lint-staged)
# .husky/pre-commit
pnpm exec lint-staged

# package.json
{
  "lint-staged": {
    "*.{ts,tsx,js,jsx}": ["pnpm exec eslint --fix", "pnpm exec prettier --write"]
  }
}
```

```bash
# Version + changeset workflow (Changesets)
pnpm add -Dw @changesets/cli
pnpm exec changeset init
pnpm exec changeset                          # author a version bump
pnpm exec changeset version                  # apply pending changesets
pnpm install                                 # update lockfile
pnpm -r publish --access=public --no-git-checks
```

```bash
# Release engineering: publish only changed packages
pnpm -r --filter '...[origin/main]' publish --access=public --no-git-checks
```

```bash
# Reset everything (nuclear option, when in doubt)
rm -rf node_modules **/node_modules pnpm-lock.yaml
pnpm store prune
pnpm install
```

```bash
# Compress install logs in CI
pnpm install --reporter=append-only
pnpm install --reporter=ndjson | jq -r 'select(.level=="error") | .message'
```

```bash
# Telemetry (pnpm collects nothing by default)
# To explicitly opt out at corporate policy time:
pnpm config set --global telemetry false
```

## See Also

- npm
- javascript
- typescript
- polyglot
- cargo
- gomod
- uv

## References

- pnpm CLI reference — https://pnpm.io/cli
- pnpm configuring (.npmrc keys) — https://pnpm.io/configuring
- pnpm workspaces — https://pnpm.io/workspaces
- pnpm filtering DSL — https://pnpm.io/filtering
- pnpm feature comparison vs npm/yarn — https://pnpm.io/feature-comparison
- pnpm symlinked node_modules structure — https://pnpm.io/symlinked-node-modules-structure
- pnpm motivation / origin — https://pnpm.io/motivation
- pnpm hooks (.pnpmfile.cjs) — https://pnpm.io/pnpmfile
- Corepack docs — https://nodejs.org/api/corepack.html
- npm registry HTTP API — https://github.com/npm/registry/blob/master/docs/REGISTRY-API.md
- Subresource integrity (SRI) — https://www.w3.org/TR/SRI/
