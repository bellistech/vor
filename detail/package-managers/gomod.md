# Go Modules — Internals & Theory

Go modules are the canonical dependency-management system for Go since Go 1.11 (2018). Unlike most package managers, Go's design choices privilege determinism, simplicity, and a content-addressable trust model over flexibility. The result is a system that's small enough to understand end-to-end, fast in practice, and produces builds that are bit-for-bit reproducible.

This document explores how Go modules work under the hood: the Minimum Version Selection algorithm and its mathematical properties, the module path/version semantics, the checksum database, the proxy protocol, the module cache layout, the directives in go.mod, and the toolchain features added across Go 1.16 through 1.22+.

## Setup

Pre-modules Go (Go 1.10 and earlier) used GOPATH — a single workspace directory at `$GOPATH/src/github.com/user/repo/`. Every dependency was checked out to a path mirroring its import statement. There was no version pinning; you got whatever was checked into the repo at that path. Tools like `dep`, `glide`, and `godep` provided manifest-and-lock workflows on top, but they were external and inconsistent.

The original module proposal was authored by Russ Cox in 2018 in a series of blog posts ("vgo" — versioned go) that became Go 1.11's experimental module support. Go 1.13 made modules the default; Go 1.17 made GOPATH-mode opt-in. By Go 1.21, most idiomatic Go code uses modules.

Cox's design principles from the original posts:

1. **Reproducibility** — same source + same go.mod = identical built binary.
2. **Compatibility** — a module's API is a contract; semver is enforced.
3. **Verification** — the go.sum file plus the checksum database guarantee no one (not even the registry) can substitute a different version of code.
4. **Minimal version selection** — pick the *minimum* version satisfying all constraints, not the maximum. This eliminates the "dependency lock file" problem because the spec already implies the lock.

Initialize a module:

```bash
go mod init github.com/user/myproject
```

This creates `go.mod`:

```
module github.com/user/myproject

go 1.22
```

Adding dependencies happens implicitly via `go get` or `go build`/`go test`:

```bash
go get github.com/gorilla/mux@v1.8.0
go build ./...
```

Updating dependencies:

```bash
go get -u ./...                  # update all to latest patch+minor
go get -u=patch ./...            # update only patch versions
go get github.com/gorilla/mux@latest
go mod tidy                      # remove unused, add missing
```

Inspecting:

```bash
go list -m all                   # all modules in build
go list -m -versions github.com/gorilla/mux  # available versions
go mod graph                     # dependency graph
go mod why github.com/x/y        # why is this dep needed
```

## Minimum Version Selection (MVS) Algorithm

MVS is the most distinctive technical decision in Go modules. While most package managers use SAT-style solvers (PubGrub, Mixology) that pick the *maximum* version satisfying all constraints, Go picks the *minimum*.

The intuition: every requirement in go.mod is a *minimum* requirement. `require github.com/gorilla/mux v1.8.0` means "I need at least v1.8.0". MVS computes: for each module mentioned anywhere in the transitive build, take the maximum of all the minimum-version requirements. That maximum is the version used.

Formally (paraphrased from Cox's MVS paper):

> Given the main module M and the build list (the set of modules in the build), the MVS algorithm:
> 
> 1. Start with the set of direct requirements of M: each requirement names a (module, minimum version) pair.
> 2. For each (M', v') in the current set, add the direct requirements of M' at version v'.
> 3. For each module M'' that appears in the set, retain only the *maximum* version requirement.
> 4. Repeat from step 2 until the set is stable.
> 5. The final set is the build list.

This is a least-fixed-point computation over a join-semilattice (versions ordered by semver, with max as the join). It always terminates (versions are bounded), and the result is unique.

The deterministic nature is the breakthrough. Given any go.mod tree, MVS produces exactly one build list. There's no "lock file" needed because the spec already has all the information. (go.sum is for *verification* of contents, not for *selecting* versions.)

Compare to SAT-based resolvers like PubGrub: those have ample search space, may require backtracking, and can be slow on pathological inputs. They also need lock files because the spec alone is underdetermined — many possible solutions exist, and the resolver picks one.

MVS's apparent downside is "you may be using older versions than necessary". If module A requires `gorilla/mux v1.8.0` and module B requires `gorilla/mux v1.9.0`, MVS picks v1.9.0 (the max of the minima). Both A and B are satisfied. This is fine; no functional regression.

But: what if v1.10.0 has a critical security fix? MVS *won't* automatically pick it up. You must explicitly `go get github.com/gorilla/mux@v1.10.0` to add a stronger minimum requirement.

This is *deliberate*. MVS gives you the version your dependencies *promised* would work. Newer versions might be better or might break — the system stays out of that decision. You opt in to upgrades explicitly.

In practice, `go get -u ./...` exists for exactly the case "I want to opportunistically pick up new minor/patch versions". It's a deliberate human decision; not something MVS does automatically.

The result: Go builds are reproducible, deterministic, and updates are explicit. No "I ran `pip install` and got a different version than yesterday because someone published a new patch."

## Module Path & Versioning

Every module has a *path* — its canonical identity:

```
github.com/gorilla/mux
go.uber.org/zap
gopkg.in/yaml.v3
golang.org/x/exp
```

The path serves three roles:

1. **Identity** — uniquely identifies the module across the universe.
2. **Discovery** — the path tells `go` where to fetch the module (more on this in Section 5).
3. **Import** — code imports `github.com/gorilla/mux` using exactly that string.

Major-version paths embed the major version in the path itself for v2+. This is "semantic import versioning":

```
github.com/foo/bar     # v0.x.x or v1.x.x
github.com/foo/bar/v2  # v2.x.x
github.com/foo/bar/v3  # v3.x.x
```

Why? Because v2 may make breaking changes, and code that imports `github.com/foo/bar` (expecting v1) shouldn't suddenly get v2's API. By making the import path encode the major version, two majors can coexist in the same build — a v1 user and a v2 user both get exactly the API they expect.

This is unique among major package managers. npm/pip/cargo allow semver "X.Y.Z where major X is incompatible" but don't change the import name across majors. Go does. The trade-off is more typing in import statements, but the win is that the module graph never has "you have two majors of the same package and one of them is implicitly broken".

There's a special case for early modules (before this rule was enforced): `+incompatible`. A module that has v2+ tags but doesn't follow the major-version-in-path rule is tagged `vX.Y.Z+incompatible`:

```
require github.com/foo/legacy v2.3.4+incompatible
```

The `+incompatible` suffix tells Go "this module has a semver tag claiming v2 but doesn't have `/v2` in its path; assume the author didn't follow the rule but use this version anyway". Old modules that pre-date Go modules often need this. New modules should always use proper major-version paths.

**Pseudo-versions** handle the case "I want to depend on an unreleased commit":

```
require github.com/foo/bar v0.0.0-20230415120000-abc123def456
```

The format is `vX.Y.Z-YYYYMMDDhhmmss-abcdef123456`:

- `v0.0.0` if no prior tag exists, or the next minor pre-release if you're between tags.
- The timestamp (UTC) of the commit.
- The first 12 hex characters of the commit SHA.

Pseudo-versions are *valid* semver versions in Go's interpretation. They sort correctly: a pseudo-version on top of `v1.2.3` is `v1.2.4-0.YYYYMMDDhhmmss-abc123` which sorts between v1.2.3 and v1.2.4.

The deterministic format means two developers asking `go get github.com/foo/bar@<commit-sha>` get identical pseudo-versions, identical go.mod, identical builds.

## go.sum

The go.sum file is the integrity-verification log. For every (module, version) ever pulled into the build, go.sum records two entries:

```
github.com/gorilla/mux v1.8.0 h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=
github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
```

The first line is the hash of the entire module's content (extracted code + module zip metadata). The second is the hash of just the go.mod file.

The `h1:` prefix indicates hash algorithm version 1, which is currently SHA-256 of a canonical encoding (the module zip's file list, sorted, with each file's content hashed independently and concatenated). The exact encoding is documented in `golang.org/x/mod/sumdb/dirhash`.

When `go build` fetches a module, it computes both hashes and compares them to go.sum. Any mismatch is a hard build failure:

```
verifying github.com/gorilla/mux@v1.8.0: checksum mismatch
        downloaded: h1:abc...
        go.sum:     h1:def...
```

This catches:

- A registry serving different content than recorded.
- A man-in-the-middle attack rewriting downloads.
- A typo in go.sum (rare; tooling generates it).
- A re-publication of a tag (which Go's ecosystem strongly discourages but isn't enforced by all servers).

The go.sum file is committed to the repo. Anyone building gets the same hashes and verifies the same content.

**The checksum database (sum.golang.org).** This is Google-operated infrastructure that records hashes for every public module/version it has ever served. When you ask `go` to fetch a new (module, version), it:

1. Fetches the content from the proxy.
2. Computes the hash.
3. Queries sum.golang.org for the canonical hash.
4. If they don't match: build fails with "mismatch with sum.golang.org".

This is the trust-on-first-use plus public-log model: even if your local proxy or your hosted git is compromised, the sum.golang.org log is an append-only, signed record. To poison it, an attacker must compromise both your proxy *and* sum.golang.org.

The signed log uses a Merkle tree (the same data structure as Certificate Transparency). Each entry is signed by Google's key; consumers can verify entries don't change after publication. A malicious actor can't retroactively rewrite the log without breaking the signatures.

Configuration:

```
GOSUMDB=sum.golang.org              # default
GOSUMDB=off                          # skip sum check (insecure!)
GOSUMDB=mysum.example.com           # private sum database
GONOSUMCHECK="*.private.example"    # skip for these patterns
GONOSUMDB="*.private.example"       # alias
```

`GOSUMDB=off` is sometimes necessary in air-gapped environments, but it should be a deliberate decision — it removes one of Go's strongest supply-chain guarantees.

## Module Proxy Protocol

The module proxy is Go's distribution mechanism. By default, `go` fetches modules through `proxy.golang.org`, a Google-operated HTTP proxy.

The proxy protocol is simple:

```
GET <proxy>/<module>/@v/list
GET <proxy>/<module>/@v/<version>.info
GET <proxy>/<module>/@v/<version>.mod
GET <proxy>/<module>/@v/<version>.zip
GET <proxy>/<module>/@latest
```

- `/@v/list` — newline-separated list of available versions.
- `/@v/<version>.info` — JSON: `{ "Version": "v1.2.3", "Time": "2023-01-15T..." }`.
- `/@v/<version>.mod` — the module's go.mod content.
- `/@v/<version>.zip` — the module's source as a zip.
- `/@latest` — same as .info for the latest stable version.

This is a *content protocol*, not a service. Any HTTP server that responds correctly is a valid proxy. There's no authentication, no API keys, no protocol-level state.

The benefits are huge:

1. **Caching is trivial.** Any HTTP cache (Varnish, CloudFront, NGINX caching proxy) caches modules for free.
2. **Mirrors are trivial.** A mirror is just a server that pre-fetches and serves the same paths.
3. **Air-gapped builds.** Run an internal proxy that holds a frozen set of modules.
4. **Migration is trivial.** Switch proxies via env var; no protocol differences.

`GOPROXY` controls the proxy chain:

```
GOPROXY=https://proxy.golang.org,direct
```

Comma-separated. `direct` means "fetch directly from the source repository (git, hg, etc.)" — no proxy. The chain is tried in order: try proxy.golang.org first; on a 404 or 410, fall through to `direct`.

Other special values:

- `off` — disable network entirely. Builds use only the local cache.
- `<url>` — a single proxy. Useful for "must use our internal proxy".

Proxy fail-open semantics: if a proxy returns 404 or 410, Go *falls through* to the next entry. If a proxy returns 5xx (server error) or times out, Go *stops* — it doesn't try the next entry. This is intentional: a proxy that's down might just be a network blip, but a proxy that says "this module doesn't exist" is making a positive statement.

`GOPRIVATE` patterns mark modules as private; these skip the public proxy and go direct:

```
GOPRIVATE=*.example.com,*.internal
```

Equivalent to:

```
GONOSUMCHECK=*.example.com,*.internal
GOPROXY=direct  # for these patterns
```

Useful for internal modules behind authenticated git remotes. The Go tool uses your git credentials to fetch; the proxy doesn't see private code.

## Module Cache Layout

The module cache is at `$GOPATH/pkg/mod/`. Layout:

```
$GOPATH/pkg/mod/
├── cache/
│   ├── download/                                # raw downloads (.zip, .mod, .info)
│   │   └── github.com/
│   │       └── gorilla/
│   │           └── mux/
│   │               └── @v/
│   │                   ├── list                 # version list
│   │                   ├── v1.8.0.info          # JSON metadata
│   │                   ├── v1.8.0.mod           # the go.mod
│   │                   ├── v1.8.0.ziphash       # h1:... hash
│   │                   └── v1.8.0.zip           # module contents zip
│   ├── lock                                      # process lock for concurrent safety
│   └── sumdb/                                    # checksum database cache
└── github.com/                                   # extracted module source
    └── gorilla/
        └── mux@v1.8.0/                          # one directory per (module, version)
            ├── README.md
            ├── go.mod
            ├── mux.go
            └── ...
```

The `cache/download/` tree has the raw archives; the top-level `github.com/` etc. tree has extracted source ready for compilation.

**Concurrent safety.** Multiple `go` processes can run simultaneously (e.g. `go build` and `go test` in different terminals). The cache uses file locks (Linux: flock; macOS: flock; Windows: LockFileEx) on the `cache/lock` file to coordinate. Module extraction is atomic: extract to a temp directory, rename to the final path. Two processes asking for the same module simultaneously: one extracts, the other waits, both finish with a populated cache.

**Read-only mode.** By default, the cache is read-only after extraction. Files have restricted perms (444 / 555). This catches the bug "I accidentally edited a vendored dep". To intentionally edit, use `replace` directive (Section 8) or `go mod vendor`.

**Cleaning.** `go clean -modcache` deletes everything. Useful to reclaim disk space or to force re-fetching. The cache regenerates on subsequent builds.

**Cache size.** Typical developer machine: a few GB. CI environments that build many distinct projects: 10-50 GB. Pruning is rare but available.

## go.mod Directives

The go.mod file uses a small set of directives:

```
module github.com/user/myproject

go 1.22
toolchain go1.22.3

require (
    github.com/gorilla/mux v1.8.0
    github.com/stretchr/testify v1.8.4
    golang.org/x/exp v0.0.0-20230415120000-abc123def456
)

require (
    github.com/davecgh/go-spew v1.1.1 // indirect
    github.com/pmezard/go-difflib v1.0.0 // indirect
)

replace github.com/foo/bar => ../local-fork

exclude github.com/bad/version v1.0.0

retract v1.5.0  // CVE-2024-XXXX
```

**`module`** — declares the module's import path. Must match the path used to fetch the module.

**`go`** — declares the Go language version. Affects features (generics require 1.18+) and behavior (1.21 changed loop variable semantics). Builds on older Go versions can fail if `go 1.22` is set but you have Go 1.20.

**`toolchain`** — declares a specific toolchain version that should be used. Triggers the auto-download of that toolchain (Section 14). Optional; without it, the local Go toolchain is used.

**`require`** — declares a dependency at a minimum version. The `// indirect` comment marks transitive deps that aren't directly imported by your code but are needed by your direct deps.

**`replace`** — overrides the location/version of a require (Section 8).

**`exclude`** — prevents a specific (module, version) from being selected by MVS. Rare; usually for "this version is broken, force the resolver to pick the next one".

**`retract`** — declares "this version of *my* module shouldn't be used". Goes into your own go.mod and signals to consumers. Used when you publish a bad version.

The format is a custom (but simple) syntax. Comments (`//`) are preserved by Go's tooling — `go mod tidy` rewrites the file but keeps your structure where reasonable.

The order of `require` blocks is significant only for readability. Go conventionally puts direct deps in one `require` block and indirect deps in another, but both are functionally equivalent.

## Replace Directive

`replace` is one of Go's most useful debugging tools. It lets you override a dependency's location — point to a local fork, a specific git ref, or a different module entirely.

Forms:

```
replace github.com/foo/bar => github.com/myuser/bar v1.2.4-fix
replace github.com/foo/bar => ../local-fork
replace github.com/foo/bar v1.0.0 => ./testdata/bar
replace github.com/foo/bar => /absolute/path/to/bar
```

The left-hand side names a module (optionally a specific version). The right-hand side specifies the replacement: another module path + version, or a filesystem path.

**Critical detail: `replace` does not transit.** Your `replace` only applies when *your module* is the main module. If module M depends on your module, M's build does not see your `replace`. They build with the canonical version.

This is by design. It would be hostile to silently change a downstream user's dependencies. If you need to publish a fix, fork the module and depend on your fork explicitly:

```go
require github.com/myuser/bar v1.2.4-fix
```

Then in code:

```go
import "github.com/myuser/bar"  // not "github.com/foo/bar"
```

The local-development workflow is:

```
# In project A's go.mod, while debugging:
replace github.com/foo/bar => ../local-bar

# Edit ../local-bar, run go test in project A. Changes are visible.

# Once fix is committed/tagged in the upstream repo:
go get github.com/foo/bar@v1.2.5

# Remove the replace directive.
```

The pattern: temporary replace during dev, never committed long-term in published modules.

For multi-module dev (work on several modules together), see Workspaces in Section 13.

## Vendor Directory

`go mod vendor` copies all dependencies into a `vendor/` directory at the project root:

```
vendor/
├── github.com/
│   └── gorilla/
│       └── mux/
│           └── (all the source files)
└── modules.txt   # the vendor manifest
```

`modules.txt` lists which modules are vendored and at what versions:

```
# github.com/gorilla/mux v1.8.0
## explicit
github.com/gorilla/mux
# github.com/davecgh/go-spew v1.1.1
github.com/davecgh/go-spew/spew
```

Each section records the module + version, optional `## explicit` marker (was directly required), and the packages used.

When building:

```bash
go build -mod=vendor ./...
```

`-mod=vendor` tells Go to use the `vendor/` directory and ignore the module cache. This is useful for:

- Building in environments without network access.
- Auditing dependencies (vendored code is in your repo).
- Pinning to specific commits beyond what go.sum guarantees.
- Reducing CI time (no network fetches needed).

If `vendor/modules.txt` is consistent with go.mod (which `go mod vendor` ensures), Go uses vendor automatically when `vendor/` exists. You can still build with the normal cache via `-mod=mod` to override.

The downside: vendoring duplicates source, increasing repo size. A medium project might add 100-500 MB of vendor source. Some teams accept this; others prefer module-cache-only builds.

`go mod vendor -e` is the "preserve as much as possible" mode for cleaning up partial vendor states.

## Import Resolution

Go's import resolution is based on the *longest matching prefix*. When you `import "github.com/foo/bar/baz"`:

1. The compiler searches go.sum for module paths that prefix-match `github.com/foo/bar/baz`:
   - `github.com/foo/bar` (matches; package path = `baz`)
   - `github.com/foo/bar/baz` (also matches; package path = empty/root)
   - `github.com/foo` (also matches if such a module exists, package path = `bar/baz`)
2. The longest match wins.

This is why module paths can be (and often are) sub-paths of organizational/user roots. `github.com/foo/repo/v2/internal` resolves correctly even though `github.com/foo` is also a valid module path *if there were such a module*.

The resolution is performed by the `go` tool against go.mod, go.sum, and the module cache. The compiler itself doesn't know about modules; it works with extracted source trees.

For the curious: the algorithm also handles `replace` directives (effective module path may be different), pseudo-versions (resolution still works at the path level), and major-version paths (`/v2` is part of the module path, considered for prefix matching).

## Build Cache Interaction

Go has a *build cache* separate from the module cache. The build cache is at `$GOCACHE` (default: `~/Library/Caches/go-build` on macOS, `~/.cache/go-build` on Linux).

The build cache stores compiled package archives keyed by an *action ID* — a hash of:

- The package's source files' contents.
- The compiler version and flags.
- The dependencies' action IDs (transitively).
- Build constraints (GOOS, GOARCH, build tags).

Building a package recomputes the action ID; if a cached archive matches, it's reused. Otherwise, the compiler runs and the result is cached.

This is why `go build` is fast on repeated invocations: most packages don't need recompilation. Edit one file in your project, only the directly-affected packages and their dependents recompile.

The action graph extends across module boundaries. A change in your code triggers recompilation of your packages but reuses cached archives for unchanged dependencies.

`go clean -cache` clears the build cache. Useful for rare cases where you suspect cache poisoning (in practice, very rare; the action ID hash is robust).

`GOCACHE=off` disables the cache entirely (slow; not recommended).

## Lazy Module Loading (1.17+)

Go 1.17 introduced *lazy module loading*. Before 1.17, every module mentioned anywhere in the dependency graph was loaded into memory and considered for MVS. For large projects this could be slow.

After 1.17, only modules whose *packages are actually used* in the build are loaded. The go.mod file gained a fuller `require` block listing all transitive deps (so MVS can still compute correctly without descending into unused modules), but the runtime loading is incremental.

Concretely:

- Before 1.17: a project with 1000 transitive modules paid the parsing cost of all 1000 on every build.
- After 1.17: only modules whose packages are imported are loaded.

The performance win is ~30-50% reduction in `go build` startup time for large projects. The downside is that go.mod's `// indirect` block grew significantly — every transitive module now appears, not just those whose go.mod conflicts mattered.

`go mod tidy` cleans up the indirect list, removing entries that turn out to not be needed.

For very large monorepos (1000+ packages), 1.17's lazy loading was transformative.

## Workspaces (1.18+)

Workspaces let you develop multiple modules together without committing `replace` directives.

Create a workspace:

```bash
mkdir my-workspace
cd my-workspace
go work init
```

This creates `go.work`:

```
go 1.22

use (
    ./api
    ./client
    ./shared
)
```

Add modules:

```bash
go work use ./api ./client ./shared
```

Or create new modules in the workspace:

```bash
mkdir -p api client shared
cd api && go mod init github.com/user/api
cd ../client && go mod init github.com/user/client
cd ../shared && go mod init github.com/user/shared
go work use ./api ./client ./shared
```

Now, builds across the workspace see all three modules as if their go.mod were `replace`d to the local path. Edit `shared/util.go`; `client` and `api` see the change immediately.

Crucially, *go.work is not committed*. It's a developer-local file. CI builds without go.work (or with `GOWORK=off`) and uses the canonical published versions.

This solves the multi-module dev problem without polluting any module's go.mod. The pattern is:

- `go.work` for local dev.
- Each module's go.mod has canonical version dependencies.
- When changes are tested + committed in `shared`, you tag a version, push, and update `client`/`api` go.mod via `go get github.com/user/shared@vNew`.

`go work sync` syncs the workspace's module versions back into individual go.mod files (uncommon; mostly for "I want my workspace's resolution to become the real go.mod").

## Go 1.21+ Toolchain Directive

Go 1.21 introduced the `toolchain` directive in go.mod:

```
go 1.21
toolchain go1.22.3
```

This says: "the language version is 1.21, but actually use toolchain 1.22.3 for building". When invoked with an older Go (e.g. Go 1.20), the `go` command auto-downloads Go 1.22.3 and re-invokes itself with that toolchain.

The download mechanism uses `dl.google.com/go/...`. First invocation:

```
$ go build .
go: downloading go1.22.3 (linux/amd64)
go: downloaded go1.22.3 (linux/amd64) (success)
[normal build proceeds]
```

The downloaded toolchain is cached at `$GOPATH/pkg/mod/golang.org/toolchain@vX.Y.Z/`. Subsequent invocations are fast.

Configuration:

```
GOTOOLCHAIN=auto                    # default: download as needed
GOTOOLCHAIN=local                   # use local toolchain only; fail if newer required
GOTOOLCHAIN=go1.22.3                # always use this toolchain
GOTOOLCHAIN=path                    # use whatever's in PATH
```

This is huge for ecosystems where projects diverge in required Go version. Before 1.21, you'd manage Go versions via a separate tool (gvm, asdf, etc.). After 1.21, the Go tool itself manages toolchain versions per project.

The signing/verification model: the downloaded toolchain is itself a Go module (`golang.org/toolchain`). It's published to proxy.golang.org with the same checksum-database protections as any other module. Downloading a toolchain is no less safe than downloading any dependency.

## Module Vulnerability Database

Go has a vulnerability database at `pkg.go.dev/vuln/`. It's a curated list of known security issues affecting Go modules.

The `govulncheck` tool reads the database and analyzes your build:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

It reports:

- Vulnerabilities affecting the modules in your go.sum.
- Whether the vulnerable code paths are *actually called* from your code (call-graph analysis).
- Recommended fixes (which version to upgrade to).

The call-graph analysis is the key feature. Many vulnerabilities affect only specific functions; if your code doesn't call those functions, you're not actually exposed. govulncheck reports vulnerabilities by callability, reducing noise vs naive "any version-affected dep is a problem" tools.

The database itself is at https://vuln.go.dev/ and is open: anyone can submit a CVE for triage. Reviewed entries become part of the data feed at https://vuln.go.dev/ID/{GO-YYYY-NNNN}.json.

For CI integration:

```bash
govulncheck -mode=binary -test ./...
```

This fails the build on any callable vulnerability, useful for "block PRs that introduce known vulns".

## Module publishing

Publishing a module is the inverse of consuming one. To publish:

1. Push your code to a public git host (GitHub, GitLab, etc.).
2. Tag a version: `git tag v1.0.0 && git push --tags`.
3. The first time anyone runs `go get github.com/you/repo@v1.0.0`, the proxy fetches from your git, validates, and caches.
4. The hash gets recorded in sum.golang.org.

That's it. No registration, no API keys, no `go publish` command.

For pre-1.0 versions (`v0.x.y`), no major-version path is needed. For v2+:

1. Update go.mod: `module github.com/you/repo/v2`.
2. Update import statements throughout the code: `github.com/you/repo/v2/...`.
3. Tag `v2.0.0`.

Both v1 and v2 can coexist — anyone using `import github.com/you/repo` gets v1; anyone using `import github.com/you/repo/v2` gets v2.

There's no central repo for module discovery; pkg.go.dev (Google-operated) crawls and indexes public modules but isn't mandatory. To "publish" a private module, just have a git remote that the consumer's environment can reach.

## Pre-release versions

Pre-release version syntax: `v1.0.0-rc.1`, `v1.0.0-beta`, `v1.0.0-alpha.2`. Per semver, these sort before `v1.0.0`.

By default, `go get module@latest` does not include pre-releases. You must explicitly request them:

```bash
go get github.com/foo/bar@v1.0.0-rc.1
```

`@latest` means "latest stable", `@upgrade` means "upgrade to highest including pre-releases of the current major", and so on.

This separation is intentional: pre-releases are for testing, not production. Having to explicitly opt in prevents "I ran go get -u and accidentally got an alpha version".

## Multiple go.mod files (modules in subdirectories)

A repo can contain multiple modules. Each module has its own `go.mod` in its own subdirectory:

```
repo/
├── api/
│   ├── go.mod         # module github.com/repo/api
│   └── ...
└── client/
    ├── go.mod         # module github.com/repo/client
    └── ...
```

For consumers, `import github.com/repo/api/v2/handlers` resolves to the `api/handlers` package in the v2 of the api module.

Tagging multi-module repos requires care:

```
git tag api/v1.0.0
git tag client/v1.0.0
```

The tag prefix `<subdir>/` tells Go which module's tag this is. Without the prefix, the tag would apply to a (nonexistent or wrong) root-level module.

This pattern is used by Kubernetes, Google APIs, and other large projects with many independently-versioned components.

## Goimports & code formatting

`goimports` (`golang.org/x/tools/cmd/goimports`) is a code-formatting tool that, in addition to `gofmt`'s formatting, manages import statements:

- Adds missing imports.
- Removes unused imports.
- Reformats imports into canonical groups (stdlib, third-party, local).

It works at the package level, looking at which symbols are used and matching against installed modules.

```bash
goimports -w ./...
```

Modern editors (VS Code's gopls, GoLand, etc.) run goimports on save. It's part of every Go developer's workflow.

## Common errors and fixes

**`go: module github.com/foo/bar: reading https://proxy.golang.org/...: 410 Gone`** — the version was retracted by the publisher. Either upgrade or use the next available version.

**`module github.com/foo/bar found, but does not contain package github.com/foo/bar/baz`** — the import path doesn't match a package in the module. Often a version mismatch (the package was moved between versions).

**`module github.com/foo/bar@latest found (v1.0.0), but does not contain package github.com/foo/bar/v2`** — you tried to import `/v2` but the latest stable version is v1. Upgrade with `go get github.com/foo/bar/v2@latest`.

**`go.sum: checksum mismatch`** — the downloaded content doesn't match go.sum. Run `go clean -modcache` and try again. Persistent mismatch: investigate (could be MITM, registry compromise, or a mistakenly committed bad go.sum).

**`go: requires Go >= 1.22 (running Go 1.20)`** — the module's go.mod requires a newer language version. Upgrade Go (`go install golang.org/dl/go1.22@latest && go1.22 download`) or use a toolchain-aware Go version.

**`cannot find module providing package`** — typo in the import path, or the module isn't yet in go.mod. Run `go mod tidy` to add missing modules.

**`directory prefix github.com/foo/bar/v2 does not contain main module`** — you tried `go run` in a subdirectory but the go.mod is elsewhere. Run from the module root.

## The proxy and corporate environments

For air-gapped or restricted networks, you typically run an internal Go proxy:

- **Athens** (https://docs.gomods.io/) — open-source proxy. Fetches from upstream, caches locally.
- **JFrog Artifactory** — proprietary, supports Go module repos.
- **Sonatype Nexus** — supports Go module repos.

Configure clients:

```
GOPROXY=https://athens.example.com/
GOSUMDB=off                    # if using internal sum database
GOPRIVATE=*.example.com        # private modules go direct
```

The internal proxy can pre-warm with a known set of modules:

```bash
# Inside Athens or similar
GOPROXY=https://proxy.golang.org go mod download github.com/foo/bar@v1.0.0
```

Once cached, the proxy serves from cache without contacting upstream.

For private modules requiring authentication, use `.netrc` or git credential helpers:

```
machine github.com
  login mytoken
  password x-oauth-basic
```

The Go tool invokes git, which uses standard authentication. No special Go-specific auth.

## Reproducibility

Go builds are bit-for-bit reproducible if:

- Same source code (verified via go.sum).
- Same compiler version.
- Same build flags.
- Same OS/arch (cross-compilation is also reproducible if you fix GOOS/GOARCH).

`-trimpath` removes file path information from binaries, making reproducibility insensitive to where the source was built:

```bash
go build -trimpath -o myapp ./cmd/myapp
```

Two developers, on two different machines, with the same go.mod/go.sum, the same `-trimpath -ldflags="..."`, get identical binaries. Useful for verifying releases (sigstore, debian reproducible builds, etc.).

The Go binary itself is also reproducible, given the same source. This is why Go is popular for distributing single-binary tools — `cs` itself is built with `-trimpath` so end users can verify they have the canonical binary.

## Comparison to other ecosystems

| Aspect | Go | npm | pip | Cargo |
|--------|-----|-----|-----|-------|
| Resolver | MVS (deterministic, min-version) | npm-resolver (max compatible) | resolvelib (max compatible) | resolver (max compatible) |
| Lock file | go.sum (verification) + go.mod (selection) | package-lock.json | requirements.txt or pyproject lock | Cargo.lock |
| Major-version handling | Path embedding (/vN) | Allow conflicting majors | Allow conflicting majors | Allow conflicting majors |
| Reproducibility | Strong | Medium | Medium | Strong |
| Default proxy | proxy.golang.org | registry.npmjs.org | pypi.org | crates.io |
| Tamper protection | sum.golang.org (signed log) | Optional | Optional | Optional |
| Toolchain auto-management | Yes (1.21+) | No | No | rustup separately |

Go's design tilts strongly toward simplicity and reproducibility. The trade-off is loss of flexibility — you can't have two majors of the same package transparently coexist (must explicitly use both `import X` and `import X/v2`). For the use cases Go targets (production servers, CLI tools, infrastructure), the constraint is generally welcomed.

## Going deeper

The Go module source is in the Go runtime: `src/cmd/go/internal/modload/`, `src/cmd/go/internal/modfetch/`, etc. The MVS algorithm itself is in `src/cmd/go/internal/mvs/`.

Russ Cox's original "vgo" blog posts (https://research.swtch.com/vgo) are the canonical introduction. Read them in order — they build the conceptual framework piece by piece.

The Go modules reference (https://go.dev/ref/mod) is the comprehensive specification. Long but worth reading sections on: MVS, the proxy protocol, go.sum format, and pseudo-versions.

For practical operational details, the Go blog has a series on modules:

- "Using Go modules" — basics.
- "Migrating to Go modules" — moving from GOPATH.
- "Publishing Go modules" — how to release.
- "Go modules: v2 and beyond" — major-version mechanics.

For the ecosystem of proxy/sumdb internals: the Athens project's docs and the sum.golang.org transparency-log spec.

## References

- Go modules reference — https://go.dev/ref/mod
- Russ Cox vgo blog — https://research.swtch.com/vgo
- MVS paper — Russ Cox, "Minimal Version Selection" — https://research.swtch.com/vgo-mvs
- Go modules tutorial — https://go.dev/blog/using-go-modules
- Migrating to Go modules — https://go.dev/blog/migrating-to-go-modules
- Publishing Go modules — https://go.dev/blog/publishing-go-modules
- Go modules: v2 and beyond — https://go.dev/blog/v2-go-modules
- Module proxy protocol — https://go.dev/ref/mod#module-proxy
- Module zip format — https://pkg.go.dev/golang.org/x/mod/zip
- Checksum database — https://sum.golang.org/
- Checksum DB Merkle log — https://research.swtch.com/tlog
- proxy.golang.org — https://proxy.golang.org/
- Go vulnerability database — https://vuln.go.dev/
- govulncheck — https://golang.org/x/vuln/cmd/govulncheck
- Go workspace mode — https://go.dev/ref/mod#workspaces
- Toolchain directive — https://go.dev/blog/toolchain
- Athens proxy — https://docs.gomods.io/
- pkg.go.dev — https://pkg.go.dev/
- Semantic import versioning — https://research.swtch.com/vgo-import
- Pseudo-versions spec — https://go.dev/ref/mod#pseudo-versions
- go.mod format — https://go.dev/ref/mod#go-mod-file
- go.sum format — https://go.dev/ref/mod#go-sum-files
- GOPROXY documentation — https://go.dev/ref/mod#environment-variables
- Module cache layout — https://go.dev/ref/mod#module-cache
- Build cache — https://pkg.go.dev/cmd/go/internal/cache
- Reproducible builds — https://reproducible-builds.org/
- Go's reproducible builds — https://go.dev/blog/rebuild
- Sigstore — https://www.sigstore.dev/
- The Go Programming Language Specification — https://go.dev/ref/spec
