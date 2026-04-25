# Go Modules & Workspaces

Go's official dependency management — `go.mod`, `go.sum`, `go work`, the module cache, GOPROXY chains, and every flag, env var, and error message you'll meet at a terminal.

## Setup

Go modules shipped experimentally in **Go 1.11 (Aug 2018)**, became default in **Go 1.13**, and the legacy `GOPATH`-only build mode was removed in **Go 1.17** (modules are mandatory for new projects). Workspaces (`go work`) shipped in **Go 1.18 (Mar 2022)**. The `toolchain` directive arrived in **Go 1.21 (Aug 2023)**. The `tool` directive (used with `go get -tool`) arrived in **Go 1.24 (Feb 2025)**.

```bash
# Check your Go version (need 1.17+ for required, 1.21+ for toolchain, 1.24+ for tool dirs)
go version
# go version go1.24.0 darwin/arm64

# Print full env (modules, proxy, sumdb, cache locations)
go env

# Print specific vars
go env GOMODCACHE GOPROXY GOSUMDB GOPRIVATE GO111MODULE GOWORK GOFLAGS GOTOOLCHAIN
```

### Install Go

```bash
# 1) Official tarball (Linux x86_64) — most reliable
curl -LO https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
# Add to ~/.profile or ~/.zshrc:
#   export PATH=$PATH:/usr/local/go/bin
```

```bash
# 2) Homebrew (macOS / Linuxbrew)
brew install go
brew upgrade go
brew info go        # show install prefix; GOROOT is auto-detected
```

```bash
# 3) asdf (per-project pinning via .tool-versions)
asdf plugin add golang
asdf install golang 1.24.0
asdf global  golang 1.24.0
echo "golang 1.24.0" > .tool-versions   # project-local pin
```

```bash
# 4) gvm (Go Version Manager — popular for switching toolchains)
bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
source ~/.gvm/scripts/gvm
gvm install go1.24.0 -B    # -B uses prebuilt binary
gvm use     go1.24.0 --default
```

```bash
# 5) GOTOOLCHAIN auto — Go 1.21+ can fetch the toolchain pinned in go.mod automatically
go env GOTOOLCHAIN              # default: auto
GOTOOLCHAIN=go1.22.5 go build .  # one-shot override
```

### GOROOT vs GOPATH (today)

```bash
# GOROOT  — where the Go distribution itself lives (toolchain, stdlib).
#           Set automatically; do NOT set unless you know why.
go env GOROOT     # /usr/local/go

# GOPATH  — workspace for downloaded modules + 'go install' binaries.
#           In modules mode, GOPATH/src is unused. GOPATH/pkg/mod and GOPATH/bin still matter.
go env GOPATH     # /Users/you/go (default if unset)

# GOMODCACHE — where module cache lives. Default: $GOPATH/pkg/mod
go env GOMODCACHE # /Users/you/go/pkg/mod

# GOBIN  — where 'go install' drops binaries. Default: $GOPATH/bin
go env GOBIN
```

```bash
# Persistent override (recommended add to shell profile)
go env -w GOPATH=$HOME/code/go
go env -w GOBIN=$HOME/.local/bin
```

```bash
# Reset to defaults (clear an override)
go env -u GOPATH
```

## Why Modules

Before modules: `$GOPATH/src/github.com/owner/repo` was the only valid place for code, dependencies were the `HEAD` of whatever was on disk, and tools like `dep`, `glide`, `godep`, `govendor` competed without consensus. Modules replaced all of that with one official mechanism.

Key properties:

- **Reproducible builds** — `go.mod` pins direct dep versions; `go.sum` records cryptographic hashes (`h1:...`) for every module zip + `go.mod` ever consumed. A clean checkout + `go build` produces identical output.
- **Minimum Version Selection (MVS)** — a deterministic algorithm: for each module, Go picks the **highest minimum version** required by anyone in the dep graph. No "latest wins"; nothing changes silently.
- **Semantic Import Versioning (SIV)** — major version `v2+` is *part of the import path* (`example.com/lib/v2`). Two majors of the same lib coexist in one binary without conflict.
- **Module-aware everything** — `go build`, `go test`, `go vet`, `go install`, `go run` all read `go.mod`/`go.sum` and resolve through the proxy.
- **Workspaces (1.18+)** — `go.work` lets you develop several modules together without committing `replace` directives.

```bash
# Inspect MVS choices for the current build
go list -m all                  # every module in the build's module graph + chosen version
go list -m -versions golang.org/x/sync   # all available versions on the proxy
```

## go mod init

Creates a new `go.mod` in the current directory. The argument is the **module path** — the import prefix every package in this module will live under.

```bash
# Public module hosted on GitHub — the path must match the eventual repo URL
go mod init github.com/bellistech/cheatsheet

# Private module on a corporate forge
go mod init git.corp.example.com/platform/widget

# Internal-only / scratch project (any unique string works, but pick something stable)
go mod init example.com/internal/tool
```

Resulting `go.mod`:

```bash
cat go.mod
# module github.com/bellistech/cheatsheet
#
# go 1.24
```

### Module path conventions

- **Match the repo URL.** `github.com/owner/repo` resolves directly via `go-import` meta tags or the standard GitHub layout. Mismatch ⇒ "module declares its path as: X but was required as: Y".
- **Lowercase only on case-insensitive filesystems** — see Gotchas. Pick lowercase always to be safe.
- **No trailing slashes, no protocol** — `https://github.com/...` is wrong; use `github.com/...`.
- **Major version suffix for v2+** — module path becomes `github.com/owner/repo/v2`, and the directory `/v2` is *not* a real subdirectory; it's encoded in the module path itself.

### Importing local code

Within one module, packages are imported by `<module-path>/<sub-dir>`:

```bash
# Repo layout
# .
# |-- go.mod                  module github.com/me/app
# |-- main.go                 import "github.com/me/app/internal/db"
# |-- internal/
# |   `-- db/
# |       `-- db.go           package db
# `-- pkg/
#     `-- api/
#         `-- api.go          package api  (importable by other modules)
```

```bash
# internal/ packages can ONLY be imported by code rooted at github.com/me/app/...
# Any external module trying to import github.com/me/app/internal/db gets:
#   use of internal package github.com/me/app/internal/db not allowed
```

## go mod tidy

Reconciles `go.mod` + `go.sum` with the actual `import` statements in your code. Run after **every** dep change.

```bash
go mod tidy             # add missing, remove unused, normalize go.sum
go mod tidy -v          # verbose — print every module added/removed
go mod tidy -e          # keep going on errors (don't abort on first failure)
go mod tidy -compat=1.21   # keep go.sum entries needed by Go 1.21 too (lower-bound compat)
go mod tidy -go=1.22    # also bump the 'go' directive in go.mod to 1.22
go mod tidy -x          # echo every command tidy runs (useful for diagnosing)
```

What it does, in order:

1. Walks every `.go` file (including `_test.go` and `// +build`/`//go:build` filtered files for default GOOS/GOARCH **and** the cross-compile matrix unless `-compat` lowers it).
2. Adds any `require` line missing for an imported package.
3. Removes any `require` line whose module no longer appears in the import graph.
4. Adds/removes `// indirect` markers as needed.
5. Rewrites `go.sum` so it contains exactly the hashes needed for the current graph (and, with `-compat`, the previous Go release's graph).

```bash
# Common workflow after editing imports
go mod tidy && go build ./... && go test ./...
```

When NOT to run `go mod tidy`:

- **In CI on a release branch** — use `go mod download` + `go build` + `go vet` instead. Tidy on CI hides forgotten local commits.
- **On a sub-module's directory if a parent workspace already drives it** — let `go work sync` flow through.

## go mod download

Pre-populates the module cache. Doesn't change `go.mod`. Useful for warming Docker layers or air-gapped builds.

```bash
go mod download                      # download every module in the build list
go mod download -x                   # echo each curl/git command
go mod download -json                # one JSON object per module on stdout
go mod download golang.org/x/sync@latest  # download a specific module@version

# Combine for offline build prep
go mod download -x | tee download.log
```

`-json` output (one record per module):

```bash
go mod download -json github.com/spf13/cobra@v1.8.0
# {
#   "Path":     "github.com/spf13/cobra",
#   "Version":  "v1.8.0",
#   "Info":     "/root/go/pkg/mod/cache/download/github.com/spf13/cobra/@v/v1.8.0.info",
#   "GoMod":    "/root/go/pkg/mod/cache/download/github.com/spf13/cobra/@v/v1.8.0.mod",
#   "Zip":      "/root/go/pkg/mod/cache/download/github.com/spf13/cobra/@v/v1.8.0.zip",
#   "Dir":      "/root/go/pkg/mod/github.com/spf13/cobra@v1.8.0",
#   "Sum":      "h1:e5/vxKd/rZsfSJMUX1agtjeTDf+qv1/JdBF8gg5k9ZM=",
#   "GoModSum": "h1:1pl1NeTyaaB6Ch59NsB28+Vc+xkcz0wzM7LxDdKQ12g="
# }
```

### Warming GOPROXY in Docker

```bash
# Two-stage build that caches deps separately from source
FROM golang:1.24 AS deps
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download -x

FROM deps AS build
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app
```

## go mod why

Explains why a module/package is in your build graph — i.e., the import chain that pulls it in.

```bash
go mod why github.com/spf13/pflag
# # github.com/spf13/pflag
# github.com/me/app
# github.com/spf13/cobra
# github.com/spf13/pflag
```

```bash
# -m treats the argument as a module path (vs a package path)
go mod why -m golang.org/x/sys

# Multiple paths in one invocation
go mod why -m golang.org/x/sys golang.org/x/text

# Vendor mode — compute against vendored set
go mod why -mod=vendor github.com/some/dep
```

If a module is in `go.mod` but unused, `go mod why` prints:

```bash
go mod why -m github.com/unused/pkg
# # github.com/unused/pkg
# (main module does not need module github.com/unused/pkg)
```

`go mod tidy` will remove these.

## go mod graph

Dumps the module graph as `<from> <to>@<version>` lines on stdout — perfect for piping.

```bash
go mod graph | head -5
# github.com/me/app github.com/spf13/cobra@v1.8.0
# github.com/me/app golang.org/x/sync@v0.5.0
# github.com/spf13/cobra@v1.8.0 github.com/inconshreveable/mousetrap@v1.1.0
# github.com/spf13/cobra@v1.8.0 github.com/spf13/pflag@v1.0.5
```

Recipes:

```bash
# Direct dependencies only (no @version means "main module")
go mod graph | awk '$1 == "github.com/me/app"'

# Anyone depending on a specific module
go mod graph | grep ' golang.org/x/sys@'

# Count distinct modules
go mod graph | awk '{print $2}' | sort -u | wc -l

# Render as DOT for Graphviz
go mod graph | awk '{printf "  \"%s\" -> \"%s\";\n",$1,$2} BEGIN{print "digraph G {"} END{print "}"}' > deps.dot
dot -Tpng deps.dot -o deps.png
```

## go mod verify

Validates that every cached module still hashes to the value recorded in `go.sum`. Run in CI immediately after checkout to detect cache tampering.

```bash
go mod verify
# all modules verified
```

On a mismatch:

```bash
go mod verify
# github.com/x/y v1.2.3: dir has been modified (/root/go/pkg/mod/github.com/x/y@v1.2.3)
```

Fixes:

```bash
# Wipe the bad cache entry and refetch from the proxy
chmod -R +w "$(go env GOMODCACHE)/github.com/x/y@v1.2.3"
rm -rf      "$(go env GOMODCACHE)/github.com/x/y@v1.2.3"
go mod download github.com/x/y
go mod verify
```

## go mod edit

Programmatic edits to `go.mod` (without hand-editing). Always run `go mod tidy` afterwards unless you know what you're doing.

```bash
# Pin / bump direct require
go mod edit -require=github.com/spf13/cobra@v1.8.0

# Drop a require (won't remove from go.sum until 'tidy')
go mod edit -droprequire=github.com/old/dep

# Local-path replace for active development
go mod edit -replace=github.com/me/lib=../lib

# Replace one version with another (or with a fork)
go mod edit -replace=github.com/upstream/lib@v1.0.0=github.com/me/lib@v1.0.0-fork.1
go mod edit -replace=github.com/upstream/lib=github.com/me/lib@v1.1.0

# Drop a replace
go mod edit -dropreplace=github.com/upstream/lib

# Exclude a known-broken version (forces MVS to pick something else)
go mod edit -exclude=github.com/x/y@v1.2.3
go mod edit -dropexclude=github.com/x/y@v1.2.3

# Retract — used by MAINTAINERS in their own go.mod to mark bad releases
go mod edit -retract=v1.0.5
go mod edit -retract='[v1.1.0,v1.1.3]'
go mod edit -dropretract=v1.0.5

# Bump language level
go mod edit -go=1.22

# Pin toolchain (1.21+) — Go will auto-fetch this toolchain if GOTOOLCHAIN=auto
go mod edit -toolchain=go1.24.0

# Reformat in place (safe canonicalize)
go mod edit -fmt

# Print the result instead of writing
go mod edit -print -require=github.com/x/y@v1.2.0

# Emit JSON of the parsed module file
go mod edit -json
# {
#   "Module": { "Path": "github.com/me/app" },
#   "Go": "1.24",
#   "Toolchain": "go1.24.0",
#   "Require": [ { "Path": "github.com/spf13/cobra", "Version": "v1.8.0" } ],
#   "Replace": null,
#   "Exclude": null,
#   "Retract": null
# }
```

## go mod vendor

Copies all dependencies into `./vendor/` so the build is hermetic against the proxy. Required by some old build systems and air-gapped environments.

```bash
go mod vendor          # populate ./vendor and write vendor/modules.txt
go mod vendor -v       # verbose — list every package copied
go mod vendor -e       # continue on errors

# Build using the vendor tree (default behavior since Go 1.14 if vendor/ exists and go.mod says go >= 1.14)
go build -mod=vendor ./...
go test  -mod=vendor ./...
```

`vendor/modules.txt` format:

```bash
cat vendor/modules.txt
# # github.com/spf13/cobra v1.8.0
# ## explicit; go 1.15
# github.com/spf13/cobra
# github.com/spf13/cobra/doc
# # github.com/spf13/pflag v1.0.5
# ## explicit; go 1.12
# github.com/spf13/pflag
```

- Lines starting with `# <module> <version>` introduce a module.
- `## explicit` means listed in `go.mod`'s `require`.
- Bare lines list each package being vendored from that module.

When to vendor:

- **Air-gapped builds** — no network in CI/build envs.
- **Long-lived branches** — guarantee reproducibility even if a module is yanked from the proxy.
- **Auditing** — vendored code is reviewable in PRs.

When NOT to vendor:

- Day-to-day open-source dev — adds churn, bloats diffs, masks `go.sum`-level issues.
- When you trust GOPROXY (`proxy.golang.org` keeps deleted modules forever).

## go.mod Syntax

Full grammar with every block annotated:

```bash
cat go.mod
# module github.com/bellistech/cheatsheet
#
# go 1.24
#
# toolchain go1.24.0
#
# require (
#     github.com/spf13/cobra v1.8.0
#     golang.org/x/sync     v0.5.0
#     golang.org/x/sys      v0.15.0 // indirect
# )
#
# replace github.com/upstream/lib => ../local-fork
#
# exclude github.com/known/bad v1.2.3
#
# retract v0.0.1   // published with a critical bug, do not use
# retract [v0.1.0, v0.1.5]   // range
#
# // tool blocks (Go 1.24+)
# tool github.com/golangci/golangci-lint/cmd/golangci-lint
```

### Directives

- `module <path>` — exactly one. Defines the module's import-prefix.
- `go <version>` — minimum Go language version. Affects compiler features + tidy behavior. Setting `go 1.21` enables loop-var-per-iteration scoping; `go 1.22` enables ranged `for i := range N`. `go 1.17+` causes `go.mod` to list **every** transitive dep — this is the "lazy module loading" pivot.
- `toolchain <name>` — optional, 1.21+. The exact toolchain version this module wants. With `GOTOOLCHAIN=auto`, the `go` command will download and re-exec into this toolchain if its own version is lower.
- `require ( ... )` — direct + indirect deps. `// indirect` marks deps that don't appear in the module's own `import` statements (they got pulled in transitively but are recorded for MVS reproducibility).
- `replace <module>[@version] => <module-or-path>[@version]` — redirect resolution. Local-path replaces have no version. **Replace does NOT transit to your dependents** — it only applies when *this* module is the main module.
- `exclude <module>@<version>` — ignore a specific version during MVS (forces selection of next-highest).
- `retract <version-or-range>` — used by *maintainers* in their *own* go.mod to mark releases as withdrawn. Surfaces in `go list -m -retracted` and `pkg.go.dev`.
- `tool <package>` — Go 1.24+. Declares a tool dependency. `go tool <name>` runs it; `go get -tool` adds it.

### Pseudo-versions

When code isn't tagged with semver, Go synthesizes a version:

```bash
# Format: vX.Y.Z-<TIMESTAMP>-<12-char-commit-hash>
# Timestamp is UTC, encoded as YYYYMMDDhhmmss (Coordinated Universal Time of commit).
v0.0.0-20250115093045-a1b2c3d4e5f6
v0.5.1-0.20250220120000-deadbeef0123    # pre-release of v0.5.1
v1.2.4-0.20250220120000-deadbeef0123    # pre-release of v1.2.4 (next patch)
```

Base version inference:

- No tags reachable from the commit ⇒ base is `v0.0.0`.
- Most recent tag `v1.2.3` reachable ⇒ base is the *next* unreleased version, encoded as `v1.2.4-0.<ts>-<hash>`.

### `+incompatible` suffix

Repos that have a `v2.0.0+` tag but **no** `/v2` module path are pre-modules legacy code. Go tolerates them but appends `+incompatible`:

```bash
require github.com/old/lib v2.5.0+incompatible
```

You can't fix this from the consumer side — the *upstream* needs to either retag with `/v2` in the module path or drop back to `v1.x`.

## go.sum

Cryptographic ledger. Every module zip and every `go.mod` ever consumed by your build has an entry.

```bash
cat go.sum
# github.com/spf13/cobra v1.8.0 h1:e5/vxKd/rZsfSJMUX1agtjeTDf+qv1/JdBF8gg5k9ZM=
# github.com/spf13/cobra v1.8.0/go.mod h1:6tWZHNZGjz0pPaXf2nrUF7Y4LcLvXmnsxw0aHOM6vCM=
```

- `h1:` is **base64-encoded SHA-256** of an authoritative module zip layout (the `1` denotes hash algorithm version 1).
- The `<version>/go.mod` line hashes just the `go.mod` file — used to short-circuit fetching the full zip if only the graph is needed.

### `go.sum` is *additive across the build matrix*

`go.sum` may contain hashes for modules NOT currently selected in your build, because tests on other GOOS/GOARCH combos can pull them in. `go mod tidy` cleans entries the **whole matrix** doesn't need.

### Checksum DB (`sum.golang.org`)

When Go fetches a new (module, version) pair, it cross-checks the hash against Google's transparent log at `sum.golang.org`. If the proxy returns a different zip than the one the log endorsed, Go aborts with `verifying module: checksum mismatch`.

```bash
# Bypass for a private host (don't do this for public modules)
go env -w GONOSUMCHECK='*.corp.example.com,*.intra.example.net'
# Or per-build
GONOSUMCHECK='*.corp.example.com' go build ./...
```

## Pseudo-Versions

Generated by `go get <module>@<commit>` when you reference an untagged commit, or by `go get <module>@<branch>`.

```bash
go get github.com/me/lib@main           # rewritten to a pseudo-version
go get github.com/me/lib@a1b2c3d        # short hash → pseudo-version
go get github.com/me/lib@HEAD           # latest commit on default branch
go get github.com/me/lib@latest         # latest tagged release (NOT @main)
```

`@latest` semantics:

- Picks the highest tagged version that is **not** retracted, **not** a prerelease (no `-alpha`/`-rc`), and **not** `+incompatible`.
- Falls back to the highest pre-release if no stable exists.
- Falls back to a pseudo-version of the default branch if no tags exist.

Branches/tags resolution (the order Go tries):

1. The module proxy (default `https://proxy.golang.org`).
2. Direct VCS (`git`/`hg`/`svn`/`bzr`/`fossil`) if the proxy returns 404 or `GOPROXY` includes `direct`.

## Semver in Modules (SIV)

Go enforces semantic import versioning at the *path* level: **major versions ≥ 2 must include `/vN` in the import path AND the module path**.

### Releasing v2 of a library

```bash
# Working in github.com/me/lib (currently v1.x)
git checkout -b v2-prep

# 1) Update go.mod's module line
go mod edit -module=github.com/me/lib/v2

# 2) Search-replace import paths in your own code
grep -rl 'github.com/me/lib' --include='*.go' . | xargs sed -i.bak 's|github.com/me/lib|github.com/me/lib/v2|g'

# 3) Tidy + commit + tag
go mod tidy
git add . && git commit -m "v2: rename module path"
git tag v2.0.0
git push origin v2-prep
git push origin v2.0.0
```

### Importing v2

```bash
# go.mod
require github.com/me/lib/v2 v2.0.0

# Go source
import "github.com/me/lib/v2/sub/pkg"
```

### `+incompatible` (escape hatch)

If upstream tagged `v2.0.0` *without* renaming to `/v2`, you can still consume it:

```bash
require github.com/old/lib v2.5.0+incompatible
```

…but neighboring modules that *did* migrate properly are not interchangeable with the `+incompatible` form. Migrate when possible.

### v0 / v1

`v0.x.y` and `v1.x.y` use the same path (no `/vN`). Breaking changes within `v0` are explicitly allowed by semver — Go will let you bump within `v0` freely.

## Module Path Resolution

Go has to translate `import "example.com/widget"` into "where do I fetch this?"

### Vanity imports (`go-import` meta tag)

`example.com/widget` doesn't host code on `example.com`. Instead, the server returns HTML with:

```bash
# GET https://example.com/widget?go-get=1
# <meta name="go-import" content="example.com/widget git https://github.com/me/widget">
# <meta name="go-source" content="example.com/widget https://github.com/me/widget https://github.com/me/widget/tree/main{/dir} https://github.com/me/widget/blob/main{/dir}/{file}#L{line}">
```

- `go-import` content fields: `<root> <vcs> <repo>`. `vcs` is one of `git`, `hg`, `svn`, `bzr`, `fossil`, or `mod` (proxy-style).
- `go-source` content fields: `<root> <home> <directory> <file>` — used by `pkg.go.dev` and `godoc` for "view source" links.

### `gopkg.in`

Vanity host that serves Git over HTTP with the major-version baked into the path:

```bash
import "gopkg.in/yaml.v3"        # tracks the latest v3.x tag in gopkg.in/yaml repo
import "gopkg.in/user/repo.v2"   # third-party repo with major v2
```

Aliases the GitHub repo automatically — no separate hosting needed.

### Standard Git layouts

- `github.com/owner/repo` — Go infers `git https://github.com/owner/repo`.
- `gitlab.com/owner/repo` — same.
- `bitbucket.org/owner/repo` — same.

## GOPATH vs Modules

```bash
# Legacy GOPATH layout (pre-1.11)
$GOPATH/
|-- bin/
|-- pkg/
`-- src/
    `-- github.com/
        `-- me/
            `-- app/
                |-- main.go
                `-- vendor/
```

In modules mode (default since 1.16), code lives **anywhere** on disk — `$GOPATH/src` is unused.

### `GO111MODULE`

Tri-state knob. Defaults to `on` since Go 1.16; can no longer be unset to disable modules in current Go (1.17+).

```bash
go env GO111MODULE          # on
go env -w GO111MODULE=on    # explicit, recommended
go env -w GO111MODULE=off   # legacy GOPATH mode (BREAKS most modern tooling)
go env -w GO111MODULE=auto  # 1.15-and-earlier behavior: GOPATH if inside src, modules elsewhere
```

If you see `go: modules disabled by GO111MODULE=off`, run `go env -w GO111MODULE=on` and try again.

## GOPROXY

Comma-or-pipe-separated list of module proxies.

```bash
go env GOPROXY
# https://proxy.golang.org,direct
```

- **Comma `,`** — try next on 410 / 404 only.
- **Pipe `|`** — try next on **any** error (network, 5xx, etc.). Use for fallbacks.

### Special tokens

```bash
direct          # bypass any proxy; fetch from VCS directly
off             # disable downloads entirely (offline; only cache)
```

### Common configurations

```bash
# 1) Default — proxy.golang.org with VCS fallback
go env -w GOPROXY='https://proxy.golang.org,direct'

# 2) Corporate Athens / JFrog / Artifactory
go env -w GOPROXY='https://athens.corp.example.com,https://proxy.golang.org,direct'

# 3) Strict — only the corporate proxy, no public fetches
go env -w GOPROXY='https://athens.corp.example.com'
go env -w GOSUMDB='off'   # corp proxy serves its own sums

# 4) Offline (CI on a sealed network)
go env -w GOPROXY='off'

# 5) Per-build override
GOPROXY=https://proxy.golang.org go build ./...
```

### Private modules

Two pieces have to align:

```bash
# 1) Where NOT to fetch through a public proxy:
go env -w GOPRIVATE='*.corp.example.com,github.com/myorg/*'

# 2) GOPRIVATE implies GONOPROXY + GONOSUMDB unless you set them:
go env -w GONOPROXY='*.corp.example.com'   # never proxy these
go env -w GONOSUMDB='*.corp.example.com'   # never check sumdb for these
```

`GOPRIVATE` accepts comma-separated glob patterns. `*` matches a path component; multiple components match if you use `/`-separated wildcards.

For Git auth on private hosts, use `~/.netrc` or an SSH replace via `~/.gitconfig`:

```bash
# ~/.netrc — HTTPS basic auth (use a token, not a real password)
cat ~/.netrc
# machine git.corp.example.com
# login bellistech
# password ghp_xxxxxxxxxxxxxxxxxxxx

# ~/.gitconfig — rewrite HTTPS to SSH for everything under your forge
git config --global url."git@git.corp.example.com:".insteadOf "https://git.corp.example.com/"
```

## GOPRIVATE / GONOSUMCHECK / GONOSUMDB / GOSUMDB

Four overlapping toggles for separating public from private resolution.

```bash
go env -w GOSUMDB='sum.golang.org'    # default — checksum DB to consult
go env -w GOSUMDB='off'               # disable globally (DON'T for public modules)

# Run-once override for a specific build
GOSUMDB=off go build ./...

# Skip checksum DB lookup for private patterns ONLY
go env -w GONOSUMDB='*.corp.example.com'

# GONOSUMCHECK (rarer) — skip the cache-vs-sum verification entirely for these
go env -w GONOSUMCHECK='*.corp.example.com'
```

Mental model:

- **GOPRIVATE** — "I never want this proxied or sum-checked." Sets the other two implicitly unless overridden.
- **GONOPROXY** — "Don't proxy these specifically." Modules still get summed unless GONOSUMDB.
- **GONOSUMDB** — "Don't consult sum.golang.org for these." Local hashes still get checked against `go.sum`.
- **GOSUMDB=off** — kill switch. **Avoid** in shared go.mods; use targeted `GONOSUMDB`.

## go work — Workspaces

Edit several modules together without committing `replace` directives.

```bash
# Create a workspace at the current dir
go work init                     # empty go.work
go work init ./api ./worker      # init + add two modules
```

Resulting `go.work`:

```bash
cat go.work
# go 1.24
#
# use (
#     ./api
#     ./worker
# )
```

```bash
# Add another module to the workspace
go work use ./libs/auth
go work use -r ./libs           # recursively add every module under ./libs

# Remove
go work edit -dropuse=./libs/auth

# Replace at the workspace level (overrides each module's go.mod replace)
go work edit -replace=github.com/upstream/lib=./libs/lib

# Sync all modules' go.mod / go.sum to be consistent with the workspace
go work sync

# Print parsed JSON of go.work
go work edit -json
```

### `go.work` semantics

- `use ./path` is a *replace* with a local path. Every module listed becomes the active version of itself in the build.
- `replace` at the workspace level wins over per-module `replace`.
- Workspaces are ignored on `go install pkg@version` (since that mode pins to a specific version).
- **Don't commit `go.work`** for libraries — workspaces are a developer-environment concept, not a release artifact. Add `go.work` and `go.work.sum` to `.gitignore` for libraries; commit them only for monorepos that *require* a workspace to build.

### `GOWORK` env

```bash
go env GOWORK              # path to active go.work, or 'off'
GOWORK=off go build ./...  # ignore any go.work in the tree (ad-hoc)
GOWORK=/abs/path/go.work go build ./...   # use a specific workspace file
go env -w GOWORK=off       # disable workspaces globally
```

### When to use `go work`

- Cloned 3 repos (`api`, `worker`, `shared-lib`) and editing `shared-lib` while testing in the others. Without workspaces you'd need to `replace` in `api/go.mod` and `worker/go.mod` and **un-replace** before pushing.
- Monorepo with N modules that need to evolve atomically.
- Reproducing an end-to-end test that crosses module boundaries.

## go install

Builds and installs binaries to `GOBIN` (or `$GOPATH/bin` if unset).

```bash
# Install a tool at a specific version (since Go 1.16) — RECOMMENDED form
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/spf13/cobra-cli@v1.3.0

# At the latest release tag
go install example.com/tool@latest

# At a specific commit (pseudo-version)
go install example.com/tool@a1b2c3d4

# Build with extra flags
go install -tags 'netgo osusergo' -ldflags '-s -w -X main.version=1.2.3' example.com/tool@v1.2.3

# Install from current module (uses go.mod's resolution rules)
go install .
go install ./cmd/cs
```

### Module-mode install rules (Go 1.16+)

`go install pkg@version` runs in **isolated** mode — it does NOT consult or modify the current module's `go.mod`. The binary is built with the dependencies declared in the *target's* `go.mod`. This is why `go install` is the canonical way to fetch global tooling without polluting your project's deps.

`go install ./...` (no `@version`) does use the current `go.mod`.

### Where binaries land

```bash
go env GOBIN          # explicit override (recommended: $HOME/.local/bin)
go env GOPATH         # GOBIN defaults to $GOPATH/bin
ls $(go env GOBIN || echo $(go env GOPATH)/bin)
```

Add to PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## go get

In Go 1.17+, `go get` is **only** for managing module dependencies. To install binaries, use `go install`.

```bash
# Add (or upgrade) a direct dependency
go get github.com/spf13/cobra
go get github.com/spf13/cobra@v1.8.0
go get github.com/spf13/cobra@latest
go get github.com/spf13/cobra@main

# Upgrade ALL deps to their latest minor/patch
go get -u ./...
go get -u=patch ./...   # patch-level only

# Downgrade
go get github.com/spf13/cobra@v1.7.0

# Remove (replaced by tidy in modern usage; kept for muscle memory)
go get github.com/old/dep@none

# Add a tool dependency (Go 1.24+)
go get -tool github.com/golangci/golangci-lint/cmd/golangci-lint@latest
# This adds a 'tool' directive to go.mod so 'go tool golangci-lint' works.
```

### Deprecation note

Pre-1.16, `go get -u` would also install binaries. That dual role caused endless confusion. Today:

- Use `go install pkg@version` to install global binaries.
- Use `go get pkg@version` (run inside a module) to add/update deps.
- Use `go get -tool` (1.24+) for tool deps tracked in `go.mod`.

## go list -m

The Swiss-army knife for inspecting module state.

```bash
# Every module in the build (main first, then alphabetical)
go list -m all

# A single module
go list -m github.com/spf13/cobra
# github.com/spf13/cobra v1.8.0

# JSON for scripting
go list -m -json all | jq '.[] | select(.Indirect != true) | .Path'

# Show available versions on the proxy
go list -m -versions golang.org/x/sync
# golang.org/x/sync v0.1.0 v0.2.0 v0.3.0 v0.4.0 v0.5.0

# Show retracted versions (default hides them)
go list -m -versions -retracted github.com/me/lib

# Are there updates available?
go list -m -u all
# github.com/spf13/cobra v1.7.0 [v1.8.0]
# golang.org/x/sync     v0.4.0 [v0.5.0]

# Force read-only mode (CI-safe — fails if anything would be downloaded)
go list -m -mod=readonly all

# Use vendored set
go list -m -mod=vendor all
```

`-mod` flag values:

- `mod` — may modify go.mod / go.sum (default outside read-only mode).
- `readonly` — fail if go.mod or go.sum need writing.
- `vendor` — use ./vendor; ignore the cache and proxy.

Useful one-liners:

```bash
# Direct deps only
go list -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{end}}' all

# Anything still on a v0.x.y
go list -m -f '{{.Path}}@{{.Version}}' all | grep '@v0\.'

# Anything pinned to a pseudo-version
go list -m -f '{{.Path}}@{{.Version}}' all | grep -E '@v[0-9]+\.[0-9]+\.[0-9]+-[0-9]{14}-'
```

## go test

Module flags interact with testing in a few subtle ways.

```bash
go test ./...                  # standard
go test -mod=readonly ./...    # CI: fail if go.mod would be modified
go test -mod=vendor ./...      # build against vendored deps only
go test -race ./...            # data-race detector (modules must allow CGO if needed for runtime)
go test -cover -coverprofile=c.out ./...
go test -fuzz=FuzzMyFunc -fuzztime=30s ./pkg/foo  # fuzz harness; corpora live in testdata/fuzz
```

The fuzz cache lives under `$GOCACHE/fuzz`, separate from the module cache. `go clean -fuzzcache` resets it.

## Replace Directive Patterns

`replace` is for **local** development overrides only — it does not transit to your dependents.

```bash
# 1) Local path — point at a sibling working tree
replace github.com/me/lib => ../lib

# 2) Forked repo — redirect imports to your fork at a specific tag
replace github.com/upstream/lib v1.2.0 => github.com/me/lib v1.2.0-fork.1

# 3) Replace ALL versions of a module with a single one
replace github.com/upstream/lib => github.com/me/lib v1.2.0-fork.1

# 4) Pin across module boundaries (use sparingly — usually MVS is enough)
replace golang.org/x/sys => golang.org/x/sys v0.15.0
```

### Replace does NOT transit

If `mod-A` has `replace foo => ../foo` in its `go.mod`, and you import `mod-A` from `mod-B`, **`mod-B` does not see that replace**. Each main module is responsible for its own `replace` directives.

This is by design — it stops a library from rerouting your dep graph behind your back. But it means local-path replaces are useless for downstream consumers, and forks need their own tagged release.

### When the fork is "just like upstream + 2 patches"

Cleanest pattern:

```bash
# 1) Tag your fork as v1.2.0-fork.1 (semver-prerelease form)
git tag v1.2.0-fork.1 && git push origin v1.2.0-fork.1

# 2) Consumers add to their go.mod:
replace github.com/upstream/lib => github.com/me/lib v1.2.0-fork.1
require github.com/upstream/lib v1.2.0-fork.1
```

## Retract Directive

Used by *maintainers*, **inside their own module's go.mod**, to tell the world "do not use this version."

```bash
# In github.com/me/lib's go.mod, after publishing a bad release:
retract (
    v0.5.0           // critical bug — see issue #42
    [v0.4.0, v0.4.5] // panic on Linux ARM64
    v0.0.0-20250115093045-deadbeef0123  // accidental commit, ignore
)
```

Effect on consumers:

- `go list -m -versions <module>` hides retracted versions by default.
- `go list -m -versions -retracted <module>` shows them (with a `(retracted)` marker).
- `go get <module>@latest` skips retracted versions.
- If a user's `go.mod` already pinned a retracted version, `go list -m -u` flags it.

```bash
go list -m -u -retracted github.com/me/lib
# github.com/me/lib v0.5.0 (retracted) [v0.6.0]
```

## Module Cache

Content-addressable cache of every module zip + extracted source.

```bash
go env GOMODCACHE          # default: $GOPATH/pkg/mod
ls $(go env GOMODCACHE)
# cache  github.com  golang.org  ...
```

Layout:

```bash
$GOMODCACHE/
|-- cache/
|   |-- download/                       # raw .zip / .info / .mod files (proxy-aligned)
|   |   `-- github.com/spf13/cobra/@v/v1.8.0.{info,mod,zip,ziphash}
|   `-- lock                            # advisory file lock
|-- github.com/spf13/cobra@v1.8.0/      # extracted, READ-ONLY
|-- golang.org/x/sync@v0.5.0/
`-- ...
```

The extracted directories are mode `0444` (read-only). Editing them is a bug — your changes won't survive `go mod download`, and `go mod verify` will complain.

```bash
# Reset the cache (rare — usually you want -modcache to be safe across projects)
go clean -modcache              # nukes the entire $GOMODCACHE
go clean -modcache -n           # dry run — print what would be deleted
go clean -cache                 # build cache — different from module cache
go clean -fuzzcache             # fuzz corpora cache
go clean -testcache             # test result cache
```

```bash
# Surgical removal (force remove read-only files first)
chmod -R +w "$(go env GOMODCACHE)/github.com/x/y@v1.2.3"
rm -rf      "$(go env GOMODCACHE)/github.com/x/y@v1.2.3"
```

## Deprecation

Module-level deprecation is a free-text comment on the `module` directive.

```bash
# In the deprecated module's go.mod
// Deprecated: use github.com/me/replacement instead.
module github.com/me/oldlib

go 1.18
```

Effect:

- `go list -m -u <module>` flags it: `(deprecated)`.
- `pkg.go.dev` shows a banner.
- `go mod tidy -v` warns once per tidy run.

The library's *own* go.mod carries the marker. There's no separate deprecation registry.

## Common Errors

### `go: module declares its path as: X but was required as: Y`

Exact text:

```bash
go: module github.com/me/lib@v1.0.0: parsing go.mod:
	module declares its path as: github.com/me/lib
	        but was required as: github.com/myorg/lib
```

**Cause**: someone forked `github.com/me/lib` to `github.com/myorg/lib` without updating the `module` line in `go.mod`. The `module` line is canonical; the URL it lives at is checked at fetch time.

**Fix**: either update the consumer's import paths to use the canonical `github.com/me/lib`, or in the fork rewrite go.mod with `go mod edit -module=github.com/myorg/lib` and retag. Or, if you can't change the fork, use `replace`:

```bash
replace github.com/me/lib => github.com/myorg/lib v1.0.0
```

### `ambiguous import: found package X in multiple modules`

Exact text:

```bash
ambiguous import: found package golang.org/x/sys/unix in multiple modules:
	golang.org/x/sys v0.15.0 (/root/go/pkg/mod/golang.org/x/sys@v0.15.0/unix)
	example.com/forked-x-sys v0.0.1 (/root/go/pkg/mod/example.com/forked-x-sys@v0.0.1/unix)
```

**Cause**: two different modules export the same package path. Almost always a botched `replace` that doesn't actually re-route.

**Fix**: pick one. Either drop the bad `replace`, or expand it to cover all the affected versions:

```bash
replace golang.org/x/sys => example.com/forked-x-sys v0.0.1
```

### `unknown revision X` / `invalid version: X`

Exact text:

```bash
go: github.com/me/lib@v1.5.0: invalid version: unknown revision v1.5.0
```

**Cause**: the tag/branch/commit doesn't exist on the remote, or your proxy can't see it.

**Fix**: verify the tag actually exists (`git ls-remote --tags <repo>`), check `GOPROXY` is reaching the right host, and for private repos check `GOPRIVATE`. For freshly-pushed tags, the proxy has a ~10-minute lag; `GOPROXY=direct go get …` bypasses it.

### `verifying module: checksum mismatch`

Exact text:

```bash
verifying github.com/me/lib@v1.2.3: checksum mismatch
	downloaded: h1:abc...
	go.sum:     h1:def...

SECURITY ERROR
This download does NOT match an earlier download recorded in go.sum.
```

**Cause** (in order of likelihood): (1) someone retagged a version in place — the zip changed but the version didn't; (2) corrupted local cache; (3) actual supply-chain compromise.

**Fix**: investigate. Check `GOSUMDB` for the canonical hash:

```bash
curl -s "https://sum.golang.org/lookup/github.com/me/lib@v1.2.3"
```

If the upstream legitimately re-tagged (it shouldn't have), delete the go.sum entry and re-run `go mod download` — Go will record the new hash. If you can't explain the change, treat it as a security incident.

### `invalid version: should be vN or later`

Exact text:

```bash
go: github.com/me/lib@v2.0.0: invalid version: module contains a go.mod file,
        so major version must be compatible: should be v0 or v1, not v2
```

**Cause**: you required `github.com/me/lib v2.0.0` but the module path lacks `/v2`. Per SIV, v2+ requires `/v2` in the import path.

**Fix**: change your import to `github.com/me/lib/v2` everywhere, and require `github.com/me/lib/v2 v2.0.0`. If the upstream module didn't migrate to `/v2`, you must use `+incompatible`:

```bash
require github.com/me/lib v2.0.0+incompatible
```

### `module github.com/X/Y/v2 found, but does not contain package github.com/X/Y/v2/Z`

**Cause**: the `/v2` tag exists but doesn't contain the subpackage you're importing — usually because the tag was created on an old branch.

**Fix**: bump to a more recent v2 tag (`go list -m -versions github.com/X/Y/v2`) or check if `Z` was renamed/removed.

### `missing go.sum entry for module providing package X`

Exact text:

```bash
missing go.sum entry for module providing package github.com/spf13/cobra/doc;
	to add: go mod download github.com/spf13/cobra
```

**Cause**: you're in `-mod=readonly` (default for `go build` since 1.16 inside a module with `go 1.16+`) and a sum is missing.

**Fix**:

```bash
go mod download github.com/spf13/cobra
# or
go mod tidy
```

### `build constraints exclude all Go files in`

Exact text:

```bash
package github.com/me/lib/internal/foo
	imports github.com/me/lib/internal/foo: build constraints exclude all Go files in /root/go/pkg/mod/github.com/me/lib@v1.0.0/internal/foo
```

**Cause**: every file in that package has `//go:build <tag>` constraints that don't match your current `GOOS`/`GOARCH` or a custom build tag you haven't set.

**Fix**: set the appropriate build tag (`go build -tags=<tag>`), cross-compile (`GOOS=linux GOARCH=amd64 go build`), or check if the package supports your target at all.

### `go: modules disabled by GO111MODULE=off`

**Cause**: someone set `GO111MODULE=off` in your env or a wrapper script.

**Fix**:

```bash
go env -u GO111MODULE          # remove user override
go env -w GO111MODULE=on       # or set explicitly
unset GO111MODULE              # clear env var inherited from parent shell
```

### `go: cannot find main module, but found .git/config in`

Exact text:

```bash
go: cannot find main module, but found .git/config in /home/me/code/scratch
	to create a module there, run:
	go mod init
```

**Cause**: you ran `go build`, `go test`, etc. outside any module.

**Fix**: `go mod init <module-path>`, or `cd` into a module, or use `go install pkg@version` (which doesn't need a current module).

### `package X is not in std`

Exact text:

```bash
package github.com/me/lib/foo is not in std (/usr/local/go/src/github.com/me/lib/foo)
```

**Cause**: typo in the import path, or modules disabled (so Go is looking under GOROOT/src).

**Fix**: double-check the import; verify `GO111MODULE=on`; check you're inside a module.

### `go: cannot use path@version syntax in GOPATH mode`

**Cause**: legacy GOPATH mode is active; `pkg@version` only works in module mode.

**Fix**: enable modules — `go env -w GO111MODULE=on` and re-run.

## Common Gotchas

### 1) Nested `go.mod` files

Broken:

```bash
# Layout
# .
# |-- go.mod          module github.com/me/app
# |-- main.go
# `-- internal/
#     |-- go.mod      module github.com/me/app/internal   <-- WRONG
#     `-- internal.go
```

The build tool sees `internal/go.mod` and treats `internal/` as a separate module. Your imports of `github.com/me/app/internal` from `main.go` then fail with `module not found`.

Fixed: delete `internal/go.mod`. Sub-packages live in the same module unless you intentionally split.

### 2) Missing `/v2` path

Broken:

```bash
# go.mod
require github.com/me/lib v2.0.0
import "github.com/me/lib/foo"
# go: github.com/me/lib@v2.0.0: invalid version: module contains a go.mod file, ...
```

Fixed:

```bash
# go.mod
require github.com/me/lib/v2 v2.0.0
import "github.com/me/lib/v2/foo"
```

### 3) `replace` not transiting

Broken: library `mod-A` has `replace foo => ../foo-fork` in its go.mod. Consumer `mod-B` imports `mod-A` and gets the upstream `foo`, not the fork.

Fixed: `mod-A` publishes its fork as a tagged release (`github.com/me/foo-fork v1.2.3`) and uses `replace foo => github.com/me/foo-fork v1.2.3`. Consumers either accept upstream `foo` (the fork's transitive impact is invisible to them) or add their own `replace`.

### 4) `vendor/` vs cache mismatch

Broken: you ran `go mod tidy`, which updated go.mod/go.sum, but forgot `go mod vendor`. Now CI builds with `-mod=vendor` and gets stale code.

Fixed: always re-vendor after `tidy`, and add a CI guard:

```bash
go mod tidy
go mod vendor
git diff --exit-code go.mod go.sum vendor/
```

### 5) `GOFLAGS=-mod=vendor` set unintentionally

Broken: `~/.bashrc` has `export GOFLAGS=-mod=vendor` from an old project. Now in a new project (no `vendor/`) every `go` command fails with `go: -mod=vendor but vendor directory does not exist`.

Fixed:

```bash
unset GOFLAGS
go env -u GOFLAGS
```

Prefer `go env -w` / `go env -u` over shell exports for Go-specific knobs.

### 6) Capitalization on case-insensitive filesystems

Broken: macOS dev pushes `import "github.com/Me/Lib"` (capital `M`). CI on Linux fails because the actual repo is `github.com/me/lib`. Locally everything builds because HFS+/APFS are case-insensitive.

Fixed: always lowercase. Add a CI grep:

```bash
grep -RnE '"github\.com/[A-Z]' --include='*.go' . && echo "uppercase import path!" && exit 1
```

### 7) Indirect deps suddenly direct

Broken: you upgraded your direct dep `cobra`, which dropped its dependency on `pflag`. Suddenly your code (which `import "github.com/spf13/pflag"`) breaks because `pflag` is gone from go.mod.

Fixed: `go mod tidy` will keep `pflag` because *your* code imports it — but it'll be moved from `// indirect` to a direct require. Test imports count too. Always `go mod tidy && go test ./...` after a dep bump.

### 8) Editing files in `$GOMODCACHE`

Broken: you patched a bug in `~/go/pkg/mod/github.com/x/y@v1.0.0/foo.go` for a hot fix. Next `go mod download` overwrites your changes (if you can even chmod the dir).

Fixed: use a `replace` directive pointing at a local clone:

```bash
git clone https://github.com/x/y ~/code/x-y-fork
cd ~/code/x-y-fork && # ... apply patch
go mod edit -replace=github.com/x/y=$HOME/code/x-y-fork
```

### 9) `go.sum` conflicts in monorepo merges

Broken: two PRs both touch deps. Their go.sum diffs conflict (both add hashes for the *same* module at *different* versions). A naive merge keeps both, which leaves stale hashes.

Fixed: resolve by re-running `go mod tidy` *after* the merge, then `git add go.sum`. Never hand-merge go.sum.

### 10) Toolchain auto-fetch surprises

Broken: project's `go.mod` has `toolchain go1.24.0`. Old machine has `go1.21`. Build silently downloads a 200MB toolchain to `$GOMODCACHE/toolchain`, surprising the user.

Fixed: pin in CI explicitly with `GOTOOLCHAIN=local` to fail fast, or `GOTOOLCHAIN=auto` to opt in. Document the policy in repo README.

```bash
GOTOOLCHAIN=local go build ./...    # use the installed go binary; no auto-download
GOTOOLCHAIN=auto  go build ./...    # default since 1.21 — auto-fetch if go.mod requires
GOTOOLCHAIN=path  go build ./...    # like local but consult $PATH for newer toolchains too
```

## Idioms

### `go mod tidy` after every dep change

```bash
# Add a dep
go get github.com/new/dep@latest
go mod tidy
go build ./... && go test ./...
git add go.mod go.sum
git commit -m "deps: add new/dep"
```

### `-locked` style with `go.sum` verify in CI

```bash
# CI script — fail if anything would change go.mod/go.sum/vendor
go mod download
go mod verify
go vet -mod=readonly ./...
go test -mod=readonly -race ./...

# Optional: detect uncommitted go.mod/go.sum drift
git diff --exit-code go.mod go.sum
```

### Workspaces for fast multi-repo iteration

```bash
mkdir ~/work && cd ~/work
git clone git@github.com:me/api.git
git clone git@github.com:me/worker.git
git clone git@github.com:me/shared.git

go work init ./api ./worker ./shared
go work use -r ./           # add any further modules under ./

# Edit ./shared, run from ./api or ./worker — changes flow through immediately
cd api && go test ./...
```

Done iterating? `cd ../shared && git commit -m '...' && git push`. Tag a release. Update the *committed* go.mod files in `api` and `worker`. Don't commit go.work.

### Private GOPROXY + skip GOSUMDB only for the private patterns

```bash
go env -w GOPROXY='https://athens.corp.example.com,https://proxy.golang.org,direct'
go env -w GOPRIVATE='*.corp.example.com,github.com/myorg/*'
# GOPRIVATE implies GONOPROXY+GONOSUMDB, but be explicit:
go env -w GONOPROXY='*.corp.example.com'
go env -w GONOSUMDB='*.corp.example.com,github.com/myorg/*'
# Public modules still go through proxy.golang.org + sum.golang.org.
```

### Reproducible release builds

```bash
# In a release branch / tag
go mod download                                  # warm cache
go mod verify                                    # check sums
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -buildid= -X main.version=$(git describe --tags)" \
    -mod=readonly -o dist/app ./cmd/app
sha256sum dist/app > dist/app.sha256
```

`-trimpath` removes local paths from the binary; `-buildid=` zeroes the build ID; combined with a fixed `SOURCE_DATE_EPOCH` env (Go 1.21+ honors it for some build steps), this gets you toward bit-identical builds.

### Tool dependencies (Go 1.24+)

```bash
# Add a dev tool that lives alongside the project
go get -tool github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go get -tool github.com/goreleaser/goreleaser/v2@latest

# Run them
go tool golangci-lint run ./...
go tool goreleaser release --snapshot --clean

# go.mod now contains:
# tool github.com/golangci/golangci-lint/cmd/golangci-lint
# tool github.com/goreleaser/goreleaser/v2
```

The tool entries pin the version (via the regular `require` block), so a fresh checkout + `go tool X` produces the same tool you used.

### Inspecting "what changed in deps"

```bash
# Diff modules between two refs (e.g., last release and HEAD)
git show v1.0.0:go.mod > /tmp/old.mod
git show HEAD:go.mod   > /tmp/new.mod
diff -u /tmp/old.mod /tmp/new.mod

# Or use go list to compute the actual selected versions
git checkout v1.0.0  && go list -m all > /tmp/old.list
git checkout main    && go list -m all > /tmp/new.list
diff -u /tmp/old.list /tmp/new.list
```

### Detecting unused indirect deps

```bash
# Anything in 'require' marked indirect but no longer needed will be removed by tidy.
go mod tidy -v 2>&1 | grep '^remove '
```

### Pinning the Go language for older environments

```bash
# Library targeting Go 1.21+ but not requiring 1.22 features
go mod edit -go=1.21
go mod edit -toolchain=none      # don't pin a toolchain; let consumers choose
go mod tidy -go=1.21 -compat=1.21
```

### Vendoring policy in CI

```bash
# Enforce vendored builds in CI
go build -mod=vendor ./...
# Detect vendor drift
go mod vendor
git diff --exit-code vendor/ go.mod go.sum
```

### Fast onboarding for new contributors

```bash
# README snippet
cat <<'EOF' > scripts/bootstrap.sh
#!/usr/bin/env bash
set -euo pipefail
go env -w GOFLAGS=-mod=readonly
go env -w GOTOOLCHAIN=auto
go mod download
go mod verify
go vet ./...
go test -short ./...
EOF
chmod +x scripts/bootstrap.sh
```

### Force-bypass the proxy for one command

```bash
GOPROXY=direct GOSUMDB=off go get github.com/me/lib@HEAD
```

Use only for debugging — `GOSUMDB=off` skips supply-chain verification.

### Resetting a wedged module state

```bash
# Nuclear option for "I have no idea what's in my cache anymore"
go clean -modcache
rm -f go.sum
go mod download
go mod tidy
go mod verify
git diff go.mod go.sum            # review what changed
```

### Reading proxy-served version metadata

```bash
# What does the proxy say about a module?
curl -s "https://proxy.golang.org/github.com/spf13/cobra/@v/list"
# v0.0.1
# v0.0.2
# ...
# v1.8.0

curl -s "https://proxy.golang.org/github.com/spf13/cobra/@v/v1.8.0.info"
# {"Version":"v1.8.0","Time":"2024-01-04T00:46:30Z"}

curl -s "https://proxy.golang.org/github.com/spf13/cobra/@v/v1.8.0.mod"
# module github.com/spf13/cobra
# go 1.15
# require ( ... )
```

### CI cache key for go modules

```bash
# GitHub Actions / GitLab — cache the module dir keyed on go.sum
# key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
# paths:
#   ~/go/pkg/mod
#   ~/.cache/go-build
```

### Detecting modules that haven't released in a long time

```bash
go list -m -json all | \
    jq -r '. | select(.Time != null) | "\(.Time)  \(.Path)@\(.Version)"' | \
    sort | head -20
```

### When a dep ships a "v1.0.0+incompatible" tag

```bash
# Upstream has tag v2.0.0 but no /v2 module path. You'll see:
require github.com/old/lib v2.5.0+incompatible

# To migrate the upstream cleanly (if you're the maintainer):
# 1) cd into the repo
# 2) Move main code under v2/ subdir (or v2 branch with /v2 in module path)
# 3) Update module line: go mod edit -module=github.com/old/lib/v2
# 4) Tag v2.0.0 on the new layout
# 5) Update README to direct users at the /v2 import path
```

### Inspecting a module zip directly

```bash
# Module zips live under $GOMODCACHE/cache/download/
unzip -l "$(go env GOMODCACHE)/cache/download/github.com/spf13/cobra/@v/v1.8.0.zip" | head
# Archive: ...
#   Length      Date    Time    Name
# ---------  ---------- -----   ----
#       128  1970-01-01 00:00   github.com/spf13/cobra@v1.8.0/.gitignore
#       ...
```

Notice timestamps are zeroed — the proxy normalizes to make hashes reproducible.

### `go env -w` is the right knob

Prefer `go env -w` over shell exports for Go-specific config — it survives shells, CI containers, IDE launches, etc., and `go env -u` cleanly reverts.

```bash
# One-time machine-wide setup
go env -w \
    GOPROXY='https://proxy.golang.org,direct' \
    GOSUMDB='sum.golang.org' \
    GOPRIVATE='' \
    GOFLAGS='-mod=readonly' \
    GOBIN=$HOME/.local/bin \
    GOTOOLCHAIN=auto

# Inspect
go env -json | jq '.GOPROXY, .GOSUMDB, .GOPRIVATE, .GOFLAGS'
```

### Catching dependency pinning regressions

```bash
# Verify nothing pulls a v0.x or pseudo-version into the build graph
go list -m -f '{{.Path}} {{.Version}}' all | \
    awk '$2 ~ /^v0\./ || $2 ~ /-[0-9]{14}-/ { print "WARN:",$0 }'
```

### Mirroring a public module to an internal proxy

```bash
# With Athens / Artifactory, simply set GOPROXY to the corp host and the proxy fetches+caches.
# To pre-warm the mirror:
GOPROXY=https://athens.corp.example.com go mod download -x golang.org/x/sync@v0.5.0

# To bypass and fetch direct (debugging mirror health):
GOPROXY=direct go list -m -versions golang.org/x/sync
```

## See Also

- [go](../languages/go.md) — language reference, stdlib, build flags, gotchas
- [cargo](cargo.md) — Rust's modules-and-build counterpart; instructive contrasts (Cargo.lock vs go.sum, workspaces vs go.work)
- [pnpm](pnpm.md) — content-addressable store similar in spirit to GOMODCACHE
- [npm](npm.md) — semver lockfiles vs MVS
- [pip](pip.md) — `requirements.txt`/`pyproject.toml` analog
- [brew](brew.md) — installing Go itself

## References

- Official module reference: <https://go.dev/ref/mod>
- Go modules wiki (legacy but useful): <https://github.com/golang/go/wiki/Modules>
- Module proxy protocol: <https://go.dev/ref/mod#module-proxy>
- `proxy.golang.org` (default GOPROXY): <https://proxy.golang.org>
- `sum.golang.org` (checksum DB): <https://sum.golang.org>
- `pkg.go.dev` (module/package discovery): <https://pkg.go.dev>
- Workspaces design doc: <https://go.googlesource.com/proposal/+/master/design/45713-workspace.md>
- Toolchain proposal (1.21): <https://go.googlesource.com/proposal/+/master/design/57001-gotoolchain.md>
- Tool directive proposal (1.24): <https://github.com/golang/go/issues/48429>
- Semantic Import Versioning paper: <https://research.swtch.com/vgo-import>
- Minimum Version Selection paper: <https://research.swtch.com/vgo-mvs>
- Russ Cox's vgo blog series (background, deeply illuminating): <https://research.swtch.com/vgo>
- `go help mod`, `go help modules`, `go help work`, `go help get`, `go help install`, `go help build` — built-in docs; read these before Stack Overflow.
