# uv — Internals & Theory

uv is Astral's Python package manager, written in Rust. It replaces (or subsumes) pip, pip-tools, pipx, virtualenv, pyenv, and Poetry, presenting a single binary that handles every step of the Python dependency lifecycle. Its design goals are clear: speed (10-100× over pip in many workflows), correctness (a modern resolver with reproducible builds), and ergonomics (a minimal, opinionated CLI).

This document explores how uv achieves those goals: the resolver algorithm, the parallel architecture, the cache and lockfile design, the Python interpreter management, and the integration with the broader Python packaging ecosystem.

## Setup

uv was first released in 2024 by Astral, the company founded by Charlie Marsh (creator of the Ruff linter). The pitch was direct: every Python developer's daily tooling has bottlenecks — pip is slow on metadata fetch, pip-tools has poor parallelism, virtualenv is a Python-bootstrap problem, pyenv requires compilation. A single Rust binary that solves all of them at once was the aim.

The sequencing was deliberate. Astral first shipped Ruff (a lint/format tool) — small scope, easy to validate the "Python tooling in Rust" thesis. Ruff's success (millions of installs, adoption by major projects within months) bought credibility for the more ambitious uv project.

uv is distributed as a single static binary per platform. There's no Python dependency for uv itself: you can install uv on a fresh machine with no Python installed, then have uv install Python (Section 11). The binary is delivered via:

```bash
# macOS / Linux installer script
curl -LsSf https://astral.sh/uv/install.sh | sh

# Homebrew
brew install uv

# Cargo (Rust toolchain)
cargo install --git https://github.com/astral-sh/uv uv

# Pip-style (bootstrap from existing Python)
pip install uv
```

On installation, the binary lives in `~/.local/bin/uv` (Linux), `/opt/homebrew/bin/uv` (macOS Homebrew), or wherever the chosen installer puts it. Updating happens via `uv self update`.

The CLI surface is comprehensive and intentionally evokes existing tool muscle memory:

```bash
# pip-compatible interface
uv pip install requests
uv pip install -r requirements.txt
uv pip freeze
uv pip list
uv pip uninstall requests

# Project-level (Poetry/PEP 621-style)
uv init                    # scaffold pyproject.toml
uv add requests            # add dep, update lock, install
uv remove requests
uv sync                    # install per pyproject + lock
uv lock                    # regenerate lock without install
uv run python script.py    # run inside project venv

# Tool installation (pipx-style)
uv tool install black
uv tool run black .        # ephemeral run
uvx black .                # alias for tool run

# Python interpreter management (pyenv-style)
uv python install 3.12
uv python list
uv python pin 3.12

# Workspace / scripts
uv run --script script.py
```

The same binary handles all surface area. Internally, the CLI is a Rust program built with `clap` (the standard Rust CLI library) and dispatches to subsystems based on subcommand.

## PubGrub Resolver

The dependency resolver is the algorithmic core of uv. It uses PubGrub — the same family of CDCL (conflict-driven clause learning) solver that Poetry's Mixology implements. uv's PubGrub implementation is in Rust, in the `pubgrub` crate (originally extracted from uv as a standalone library).

PubGrub was designed for Dart's pub package manager. The design paper (linked in references) is short and readable; it formalizes the notion of *incompatibility-based resolution* and proves correctness and termination properties.

The high-level loop:

1. **Decision**: Pick a package whose version range hasn't yet been narrowed to a single version. Choose the highest version satisfying the current range (newest-first preference).
2. **Propagation**: Apply known incompatibilities to derive new constraints. If two terms imply a third, add the third to the partial solution.
3. **Conflict detection**: If propagation derives a constraint that contradicts the partial solution, you've hit a conflict. Compute the *root cause* — the minimal set of decisions/derivations that caused the conflict — and add it as a new incompatibility.
4. **Backjumping**: Use the new incompatibility to backjump to the latest decision level where the conflict could have been avoided. Try a different decision.
5. **Termination**: When no decisions remain (all packages have a single version) and no conflicts hold, you've found a solution. If you derive an incompatibility that's implied by the root constraints (the project's top-level requirements alone), the project is unsatisfiable.

The key insight is that step 3's *learning* prunes far more of the search space than naive backtracking would. Each conflict adds a clause that prevents a whole equivalence class of bad partial solutions, not just the specific one being explored.

uv's implementation has several optimizations over a textbook PubGrub:

**Parallel metadata fetch.** When the resolver needs the dependency list for `requests==2.31.0`, it issues an async HTTP request. While that's pending, it can continue exploring other branches of the search. Rust's `tokio` runtime and async/await make this lock-free and efficient. Poetry, in contrast, fetches metadata serially per branch (Python's GIL bottlenecks the I/O).

**Universal resolution.** uv resolves once for all platforms (Linux/macOS/Windows × multiple Pythons × multiple architectures) and produces a single lockfile that's portable. The PubGrub solver is extended with marker-aware terms: a term can be conditional on `sys_platform == "linux"` or `python_version >= "3.10"`. The lockfile records both unconditional and conditional resolutions.

**Incompatibility caching across runs.** Common incompatibilities (e.g. "package X is yanked", "package Y has no Linux wheel for Python 3.13") are cached on disk. Subsequent resolves reuse these without re-computing.

**Pre-extracted metadata via PEP 658.** Instead of downloading the full wheel just to read its METADATA file, uv uses PEP 658 (where supported) to fetch only the metadata. This is essential for resolution speed: a typical resolver might need metadata for 1000+ packages, and downloading full wheels for all of them would be hundreds of MB.

The resolver supports several conflict-handling strategies:

- **First match**: pick the first compatible version found.
- **Newest** (default): pick the highest compatible version, preferring stable releases over pre-releases.
- **Lowest**: pick the lowest compatible version (useful for testing minimum supported versions).
- **Lowest-direct**: lowest for direct deps, newest for transitive (a useful middle ground for "test min versions of my own constraints").

Configurable via `--resolution=lowest|highest|lowest-direct`.

## Dependency Graph Walking

What makes uv 10-100× faster than pip in practice? Multiple factors compound:

**(1) Parallelism.** uv issues hundreds of HTTP requests concurrently. Python's GIL makes pip's parallelism limited (the `pip` binary uses threads but contention slows them). Rust has no GIL; uv's tokio runtime saturates the network on any reasonably-fast connection.

**(2) Native code performance.** Pure Rust execution is 10-100× faster than equivalent Python for CPU-bound work. Resolver inner loops, hash computations, archive extraction, version-range arithmetic — all benefit.

**(3) Cached resolution.** The first time you resolve a project, uv builds the dependency graph from scratch. Subsequent operations (sync, add, remove) reuse the cached graph and only re-resolve affected portions.

**(4) Lock-free architecture.** uv uses Rust's ownership model to ensure data-race freedom without explicit locks. Multiple worker tasks can read shared caches concurrently; writes are coordinated via channels (CSP-style).

**(5) Metadata-only fetches.** PEP 658 plus aggressive use of HTTP range requests means uv often pulls just kilobytes of metadata where pip would pull megabytes of full wheels.

**(6) Tarball pre-extraction in workers.** For sdists that need building, uv runs the build in a worker pool, parallelized across packages.

The net effect: a typical "install pandas + scikit-learn + jupyter" cold install that takes pip 30 seconds takes uv 2-3 seconds. Warm cache (re-install of the same project), pip takes 10-15 seconds; uv takes ~200ms.

The benchmarks (linked in references) are reproducible. Astral publishes scripts that run identical operations under uv, pip, pip-tools, Poetry, and pdm, on standardized hardware. The numbers consistently favor uv.

## Wheel + sdist Handling

A Python distribution is either a *wheel* (`.whl`, pre-built) or an *sdist* (`.tar.gz`, source). Wheels are zip files with a specific layout (PEP 427) that's directly installable: extract, copy files into site-packages, done. sdists require a build step: invoke a build backend (per PEP 517) to produce a wheel.

For resolution, uv needs metadata — name, version, Python-version requirements, dependencies. Where does it come from?

**Fast path: PEP 658.** When a registry exposes per-distribution metadata files alongside the wheel/sdist, uv fetches just that metadata file. PyPI supports PEP 658 for wheels uploaded since 2023.

**Fast path: PEP 691 JSON simple API.** The simple-index endpoint (`https://pypi.org/simple/<pkg>/`) returns JSON when requested with `Accept: application/vnd.pypi.simple.v1+json`. The JSON includes per-distribution metadata fields, eliminating per-file fetches.

**Slow path: HTTP range request.** For older wheels without PEP 658 sidecars, uv issues an HTTP range request for the wheel's central directory record (a few KB at the end of the zip), parses the central directory to find METADATA's offset, and issues another range request for METADATA only. Total bandwidth: tens of KB instead of MB.

**Fallback path: full download.** If range requests fail (some mirrors don't support them) or the file is an sdist, uv downloads the full archive and extracts metadata.

For sdists specifically, "extract metadata" can mean running the build backend. uv runs `python -m build --wheel --no-isolation --metadata` (PEP 643's `prepare_metadata_for_build_wheel` API) to extract metadata without actually compiling. If the backend doesn't support this, uv falls back to a full build (which is slow — see Section 13).

For the install phase (post-resolution), uv handles wheels and sdists differently:

**Wheels.** Decompress the zip into the target site-packages directory. Compute SHA-256 over each file and verify against the lockfile. Update RECORD and `*.dist-info/INSTALLER` files. Hard-linked or copied from the cache.

**sdists.** Run the build backend in an isolated environment (per PEP 517). uv creates a temporary venv, installs the backend's `[build-system].requires`, calls `build_wheel`, then proceeds as for a wheel.

For projects that need many sdist builds (cold install, no wheels for the target platform), uv parallelizes builds in a worker pool. Each worker runs a single build at a time; the pool size defaults to the number of CPU cores.

## python-build-standalone

uv can install Python interpreters: `uv python install 3.12`. The interpreters come from `python-build-standalone` — a project originally created by Gregory Szorc (indygreg) that distributes pre-built CPython binaries for all major platforms.

The problem this solves: building Python from source is slow (5-15 minutes), error-prone (depends on system libraries, headers, configure options), and produces a Python that's tied to the build machine's libraries. `python-build-standalone` produces *static* binaries (or near-static, with carefully-selected dynamic dependencies) that work on a wide range of Linux distributions, macOS versions, and Windows versions.

The build pipeline is:

1. Build CPython from upstream source on a controlled environment.
2. Statically link as much as feasible (libffi, libffi, openssl, etc.).
3. Test against a matrix of target platforms.
4. Package as a .tar.gz containing the entire interpreter + standard library.

uv's `uv python install` downloads the pre-built tarball from `https://github.com/indygreg/python-build-standalone/releases` (or GitHub API), extracts it to `~/.local/share/uv/python/<version>/`, and creates symlinks for `python`, `python3`, `python3.12` etc.

Per-project pinning:

```bash
uv python pin 3.12
```

writes `.python-version` (a single line: `3.12`) at the project root. Subsequent `uv run`, `uv sync`, `uv venv` use that version. Useful for "this project requires Python 3.12 minimum" scenarios.

uv supports both:

- **uv-managed Pythons**: installed via `uv python install`, lives in uv's cache, fully controlled by uv.
- **System Pythons**: any `python3` in PATH, or a specific path. Discovered via `uv python list`.

The `--python` flag overrides the auto-selection: `uv venv --python 3.11`, `uv run --python /opt/python3.13/bin/python`, etc.

The big win: `python-build-standalone` works *everywhere*. No "install build-essential, libssl-dev, libffi-dev, ..." rituals. No "Python 3.13 is broken on Ubuntu 22.04 because of OpenSSL 1.1 vs 3.0". Just download a binary, run it.

## uv.lock

The lockfile is `uv.lock`, TOML-format, designed to align with PEP 751 (the universal Python lockfile standard, in progress as of 2024).

```toml
version = 1
requires-python = ">=3.10"

[[package]]
name = "requests"
version = "2.31.0"
source = { registry = "https://pypi.org/simple" }
dependencies = [
    { name = "charset-normalizer" },
    { name = "idna" },
    { name = "urllib3" },
    { name = "certifi" },
]
sdist = {
    url = "https://files.pythonhosted.org/packages/.../requests-2.31.0.tar.gz",
    hash = "sha256:942c5a..."
}
wheels = [
    { url = "https://files.pythonhosted.org/packages/.../requests-2.31.0-py3-none-any.whl", hash = "sha256:58cd..." },
]

[[package]]
name = "urllib3"
version = "2.0.7"
source = { registry = "https://pypi.org/simple" }
sdist = { ... }
wheels = [...]
markers = "python_version < '3.10'"
```

Key elements:

- **`version`** — schema version. uv reads any version it knows; refuses newer.
- **`requires-python`** — the project's Python range; lockfile is valid only for that range.
- **`[[package]]`** — one entry per resolved package. Records all candidate distributions (wheels + sdist), full hashes, and the dependency adjacency list.
- **`source`** — where the package came from: `registry = "url"`, `git = "url"`, `path = "..."`, or `direct = "url"`.
- **`markers`** — optional environment marker for conditional installation. e.g. `markers = "sys_platform == 'win32'"` for Windows-only deps.
- **`hash`** — SHA-256 hex string. Verified on every download and install.

The lockfile is *cross-platform*. A single `uv.lock` from a Linux developer's machine works for Mac and Windows colleagues without re-resolving. The resolver explores all platform/Python combinations and records the union.

Compared to `requirements.txt` (pip's typical lockfile, often produced by `pip freeze` or `pip-compile`):

| Feature | requirements.txt | uv.lock |
|---------|-----------------|---------|
| Format | Plain text | TOML |
| Hashes | Optional, --hash-mode | Required |
| Cross-platform | No (frozen for resolving env) | Yes |
| Conditional deps | Markers in-line | Structured markers |
| Source info | URL only | Structured (registry/git/path) |
| Editable installs | -e prefix | Source = path with editable flag |
| Roundtripping | Lossy | Lossless |

PEP 751 is the emerging standard for a universal Python lockfile. uv's format is closely aligned; the goal is for `uv.lock` to *be* a PEP 751 lockfile when the spec finalizes.

`uv lock` regenerates the lockfile without installing. `uv sync` reads the lockfile and installs the resolved versions, creating/updating a venv at `.venv/`. The two phases are separated for CI workflows that lock in one job, install in another.

## Workspaces

uv workspaces are inspired by Cargo's workspaces: a single `pyproject.toml` at the repo root declares member packages, and a single lockfile spans the whole workspace.

```toml
[tool.uv.workspace]
members = ["packages/*", "apps/*"]
exclude = ["apps/legacy"]
```

Members are subdirectories containing their own `pyproject.toml`. Each member has its own metadata (name, version, dependencies) but shares the workspace's lockfile and venv.

Internal cross-references use the `workspace = true` flag:

```toml
# packages/web/pyproject.toml
[project]
name = "myorg-web"
version = "0.1.0"
dependencies = ["myorg-core"]

[tool.uv.sources]
myorg-core = { workspace = true }
```

`[tool.uv.sources]` is uv's mechanism for overriding where a dependency comes from. `workspace = true` says "resolve from a workspace member". When `myorg-web` is published, the dependency on `myorg-core` is published as a normal version-pinned dep; locally, it's a path reference to `packages/core/`.

Other source overrides:

```toml
[tool.uv.sources]
mypackage = { git = "https://github.com/example/mypackage", rev = "abc123" }
otherpkg = { path = "../local-pkg", editable = true }
ironkey = { url = "https://example.com/ironkey-1.0.0.tar.gz" }
```

uv resolves all members together — the lockfile contains the union of dependencies. When you `uv sync`, the active member's deps are installed; `uv sync --all` installs every member's deps.

This is a deliberate analogue of Cargo: members "share a lockfile", so all version conflicts surface at lock time, not at runtime when two members happen to be active.

## Tool Installation (uvx)

For tools that don't belong to any specific project — `black`, `ruff`, `pre-commit`, `cookiecutter` — Python developers traditionally used pipx. pipx creates a per-tool isolated venv and exposes the tool's entry-point script in PATH.

uv subsumes pipx with two command paths:

**Persistent install:**

```bash
uv tool install black
uv tool install --python 3.12 cookiecutter
```

Creates `~/.local/share/uv/tools/black/` (a venv), installs `black` into it, links `black` into `~/.local/bin/` (or `~/Library/Application Support/uv/bin/` on macOS).

**Ephemeral run:**

```bash
uv tool run black .
uvx black .                  # alias
uvx --python 3.13 black .
uvx git+https://github.com/example/tool@main mycommand
```

`uvx` resolves the tool's dependencies on the fly into a temp venv (or reuses a cached venv if available), runs the tool, then leaves the venv (cached for next time).

The cached ephemeral venvs live in `~/.cache/uv/tool/` and are GC'd by `uv cache prune`. Unique cache key per (tool, version, Python, platform) tuple.

The trade-off vs pipx:

- pipx writes the tool's binary directly to a stable path with stable behavior.
- uv tool install behaves the same.
- `uvx` is more like `pipx run` — ephemeral, and faster on warm cache (uv's resolver + cached store make a re-run sub-second).

## PEP 723 Inline Metadata

PEP 723 (accepted 2024) allows scripts to declare their dependencies inline:

```python
#!/usr/bin/env python
# /// script
# requires-python = ">=3.10"
# dependencies = ["requests", "rich"]
# ///

import requests
from rich import print

resp = requests.get("https://api.example.com")
print(resp.json())
```

The `# /// script ... # ///` block is parsed as TOML. `requires-python` and `dependencies` are the only currently-required fields.

uv handles PEP 723 scripts with `uv run --script`:

```bash
uv run --script ./myscript.py
```

uv:

1. Parses the inline metadata.
2. Creates a temp venv (cached in `~/.cache/uv/scripts/<hash>/`).
3. Installs the declared dependencies.
4. Runs the script in that venv.
5. Cleans up — but caches the venv keyed by the metadata hash, so re-runs are fast.

If the metadata changes (you add a dependency), the venv is regenerated. Otherwise, the cached venv is reused.

This is transformative for "run a one-off script" workflows. Previously, you'd `pip install requests rich` (polluting the system or active venv), or create a venv just for this script. With PEP 723 + uv, the script declares its needs and uv handles the rest.

`uv add --script myscript.py requests` adds a dependency to the script's inline metadata (rewriting the file).

## PEP 735 Dependency Groups

PEP 735 (accepted 2024) standardizes named dependency groups in pyproject.toml:

```toml
[dependency-groups]
test = ["pytest>=7", "pytest-cov"]
lint = ["ruff>=0.1", "mypy"]
docs = ["sphinx", "sphinx-rtd-theme"]

# A group can include another group:
dev = [{ include-group = "test" }, { include-group = "lint" }]
```

Activation:

```bash
uv sync --group test
uv sync --all-groups
uv sync --no-group docs
```

Groups are *not* installed by default — only the project's main dependencies are. You opt in via `--group <name>`.

This replaces the older patterns:

- `setup.cfg` extras (`extras_require`).
- `[project.optional-dependencies]` (PEP 621). Still works, but `[dependency-groups]` is preferred for non-published groups.
- `[tool.poetry.group.<name>.dependencies]` (Poetry-specific).

uv reads PEP 735 natively. Poetry 2.x reads PEP 735 as a complement to its own `[tool.poetry.group.*]`. The trend across the ecosystem is convergence on PEP 735 as the standard.

The semantic difference: `[project.optional-dependencies]` defines extras that are part of the *published* package metadata. Users `pip install mypackage[test]` to opt in. `[dependency-groups]` is *unpublished* — purely development-time configuration, not visible to consumers of your distributed wheel.

## Python Toolchain Manager

uv replaces pyenv for many users. The mechanics:

```bash
uv python install 3.12         # download python-build-standalone build
uv python install 3.10 3.11 3.12  # multiple versions at once
uv python list                  # show installed + system Pythons
uv python pin 3.12             # write .python-version
uv python find 3.11            # show path of a specific version
```

Discovery search order:

1. `.python-version` in the current directory or ancestors.
2. `UV_PYTHON` environment variable.
3. `--python` flag.
4. Newest installed uv-managed Python.
5. System `python3` in PATH.

The auto-selection is debuggable via `uv python find <spec>` which prints the resolved path.

Per-project venv interaction:

```bash
uv venv                # create .venv/ using the current effective Python
uv venv --python 3.13  # specify Python
uv venv --seed         # also install pip + setuptools (for compatibility with old workflows)
uv sync                # auto-creates .venv/ if missing
```

The default venv path is `.venv/` next to pyproject.toml. The `[tool.uv]` section can override:

```toml
[tool.uv]
managed = true               # enable uv's auto-management
python-preference = "managed"  # "system", "managed", "only-managed", "only-system"
```

`managed = true` means uv treats `.venv/` as fully owned — `uv sync` may delete and recreate it freely. With `managed = false`, uv treats it as user-managed (won't delete on its own).

## Cache Architecture

uv's cache is content-addressable, similar to pnpm's. Located at:

- Linux: `~/.cache/uv/`
- macOS: `~/Library/Caches/uv/`
- Windows: `%LOCALAPPDATA%\uv\Cache\`

Subdirectories:

- `archive-v0/` — downloaded wheels and sdists, keyed by hash.
- `built-wheels-v0/` — wheels built from sdists, cached.
- `simple-v0/` — cached PEP 503/691 simple-index responses.
- `wheels-v0/` — extracted wheel directories (post-install layout, ready to copy/link).
- `interpreter-v0/` — discovered Python interpreters' metadata (their `sysconfig`, version, etc.).
- `python/` — uv-managed Python installations.
- `tool/` — venvs for `uvx` ephemeral runs.

File-locking is used to coordinate parallel access. Each cache write goes through a `.tmp` file + atomic rename pattern, so concurrent uv processes never see partial writes.

The cache supports installs *into* the cache: when uv installs a wheel into a venv, it can hardlink (or copy/clone) from `wheels-v0/` instead of re-extracting. This makes per-venv install nearly free.

`uv cache prune` removes cache entries not referenced by any active project. By default keeps recent entries; `--ci` mode is more aggressive.

`uv cache clean` removes everything (rare; mostly for debugging).

## Compared to pip

pip is the de facto standard Python installer. Its strengths: ubiquitous, simple, well-understood. Its weaknesses: slow, single-threaded, lacking a real lockfile, dependency-resolution failures on complex graphs.

| Operation | pip | uv |
|-----------|-----|-----|
| Cold install (no cache) | Baseline | 5-10× faster |
| Warm install (full cache) | 3-5s | 50-200ms |
| Resolution (1000 packages) | 30-60s | 2-5s |
| Lock generation | (no native support) | Built-in |
| Cross-platform lock | (no support) | Built-in |
| Parallel metadata fetch | Limited (GIL) | Yes |
| Native code | No | Yes |

The "warm install 100x faster" claim refers to the round-trip time of `uv sync` on a project where the lockfile and cache are both populated. pip's `pip install -r requirements.txt` re-resolves and re-fetches metadata even when nothing changed; uv recognizes the lock matches and skips that work.

The "slow first-time fetch" caveat: uv's first install on a fresh machine has to download everything once. That's bandwidth-bound and similar to pip's time. The wins come on subsequent installs.

uv's pip-compatible interface (`uv pip install`, `uv pip freeze`, `uv pip list`, `uv pip uninstall`) is intentional. It lets users replace `pip` with `uv pip` in scripts and CI without rewriting workflows. The behavior matches pip's where possible, except faster.

There are limits. uv's pip mode doesn't support every pip flag (rare ones like `--proxy` got added later). Some workflows that depend on pip's specific output format need adjustment. For most users, the migration is `s/pip /uv pip /` and done.

## Build backends

uv doesn't ship a build backend (unlike Poetry, which ships `poetry-core`). Projects using uv typically pick a backend separately:

- `hatchling` — Pythonic, opinionated, good defaults. The Astral team often recommends this.
- `setuptools` — the legacy default. Most compatible but verbose.
- `flit-core` — minimal, simple-projects only.
- `poetry-core` — Poetry's backend, can be used standalone.
- `pdm-backend` — PDM's backend.

A typical `[build-system]`:

```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

When uv builds a wheel for your project (e.g. for `uv build`), it spawns the configured backend in an isolated environment. The backend produces the wheel; uv handles distribution and installation.

`uv build` produces a wheel + sdist:

```bash
uv build              # both
uv build --wheel      # wheel only
uv build --sdist      # sdist only
uv build --out dist/  # custom output directory
```

`uv publish` uploads to PyPI:

```bash
uv publish                 # default registry
uv publish -r private      # to a configured alternate registry
uv publish --token "$PYPI_TOKEN"
```

Authentication is via `~/.pypirc` (legacy), env vars, or the `--token` flag. uv can also use OIDC trusted publishing for GitHub Actions workflows.

## Alternative resolvers

While PubGrub is uv's default, some projects need different semantics. Notable alternatives in the broader ecosystem:

- **pip's resolver** — backtracking, no clause learning. Reasonable for simple cases; pathological for large monorepos.
- **Mixology (Poetry)** — same family as PubGrub; pure-Python implementation.
- **resolvelib** — a generic resolver framework used by pip 20.3+. Not specifically PubGrub but borrows similar ideas.

uv's resolver is implemented as a separate Rust crate (`pubgrub`). It's been carefully tuned for Python's specific constraints (markers, extras, pre-releases) but the core is a faithful PubGrub. The crate is reusable — non-Python tools can adopt it.

## Versioning policy

uv follows semver loosely but with practical pragmatism. Major versions (uv 1.0, 2.0, ...) introduce breaking changes. Minor versions add features. Patch versions are bug fixes.

The lockfile schema versions independently. Lockfile version 1 is current; future versions might bump if PEP 751 changes the spec. uv reads any lockfile version it knows; refuses newer.

uv's CLI surface is officially "stable" but pragmatic. Rare flags may be reorganized; deprecated flags emit warnings before removal.

## Configuration

Configuration lives in three places, in priority order:

1. CLI flags (highest priority).
2. Environment variables (`UV_*`).
3. Project `pyproject.toml` `[tool.uv]` section.
4. User config: `~/.config/uv/uv.toml` (Linux), `~/Library/Application Support/uv/uv.toml` (macOS).

Common settings:

```toml
# pyproject.toml or uv.toml
[tool.uv]
managed = true
python-preference = "managed"
python-downloads = "automatic"
index-strategy = "first-match"          # or "unsafe-best-match"
keyring-provider = "subprocess"         # use system keyring for auth
preview = false                          # opt into experimental features

[[tool.uv.index]]
name = "pypi"
url = "https://pypi.org/simple"

[[tool.uv.index]]
name = "private"
url = "https://npm.example.com/simple"
default = false
explicit = true                          # only used when explicitly named
```

`index-strategy = "first-match"` means uv stops searching at the first index that has the package. `"unsafe-best-match"` means search all indexes and pick the highest version — risky because it can pull from a public index a package with the same name as your private one (dependency confusion attack).

For private indexes, `explicit = true` requires `[tool.uv.sources]` to opt-in: a dependency uses the explicit index only if you write `mypkg = { index = "private" }` in `[tool.uv.sources]`. This is the safer default for private package distribution.

## Common workflows

**New project from scratch:**

```bash
uv init --app my-project
cd my-project
uv add "django>=4.2"
uv add --dev pytest
uv sync
uv run pytest
```

**Existing pip-based project:**

```bash
cd existing-project
uv venv                          # create .venv from current Python
uv pip install -r requirements.txt
# OR migrate to uv-native:
uv init --no-readme              # convert to pyproject if needed
uv add ...                       # convert requirements.txt entries
```

**Existing Poetry project:**

```bash
cd poetry-project
uv sync                          # uv reads pyproject.toml directly
# uv treats [tool.poetry.dependencies] as project deps
# but ignores Poetry-specific groups
```

**Run a script:**

```bash
uv run --script myscript.py
```

**Install a tool globally:**

```bash
uv tool install black
black --version
```

**Switch Python versions:**

```bash
uv python install 3.13
uv python pin 3.13
uv sync                          # rebuild venv with 3.13
```

## Performance instrumentation

uv has built-in tracing support. Set `UV_LOG_TIMINGS=1` and `UV_VERBOSE=1` to see what's slow:

```bash
UV_VERBOSE=1 UV_LOG_TIMINGS=1 uv sync 2>&1 | head -100
```

Output includes:

- DNS resolution time per host.
- TLS handshake time.
- HTTP request/response sizes.
- Per-package resolution branches taken.
- Wheel build durations.

Helpful for diagnosing "why is this install slow" — usually a single misbehaving registry or a large sdist that needs building.

## Common errors and fixes

**`No matching distribution found for X`** — uv couldn't find any (Python, platform, arch)-compatible wheel and the sdist also failed. Often: missing build tools (gcc, openssl-dev) for native extensions. Install them, or pin a version with a wheel for your platform.

**`hash mismatch`** — the downloaded file doesn't match the lockfile hash. Rare; usually a flaky mirror. `uv cache clean <pkg>` and retry.

**`Failed to determine Python interpreter version`** — uv tried to invoke a Python that doesn't exist or is broken. Check `uv python list` and pin a known-good version.

**`Conflicting requirements`** — the resolver couldn't find a satisfying version. Read the error carefully; it shows the chain of conflicts. Use `--resolution=lowest-direct` to try alternative resolution strategies, or relax constraints.

**`Workspace not found`** — `[tool.uv.workspace]` references a member that doesn't exist or doesn't have a pyproject.toml.

**`Editable install failed`** — for path/git deps with `editable = true`, the project must use a backend that supports editable installs. setuptools, hatchling, poetry-core all support; some custom backends don't.

## Going deeper

uv's source is in Rust, available at https://github.com/astral-sh/uv. Key crates:

- `crates/uv-cli` — CLI entry.
- `crates/uv` — main library.
- `crates/uv-resolver` — the resolver, wrapping the `pubgrub` crate.
- `crates/uv-cache` — content-addressable cache.
- `crates/uv-installer` — the installer (linking, hardlinks, etc.).
- `crates/uv-python` — Python interpreter management.
- `crates/uv-distribution` — wheel/sdist handling.

The `pubgrub` crate (https://github.com/pubgrub-rs/pubgrub) is a separate library that uv depends on. It's a generic PubGrub implementation usable by other tools.

Astral's documentation at https://docs.astral.sh/uv/ is comprehensive and includes deep-dive guides on resolution, caching, and migration from pip/Poetry.

For the algorithmic foundation, read the PubGrub paper (linked in references). It's accessible to anyone who's understood SAT solving at a high level.

## Compatibility matrix

uv aims for broad compatibility with existing Python tooling:

| Tool / Format | Status |
|--------------|--------|
| pip CLI | `uv pip` — most flags supported |
| pip-tools (compile) | `uv pip compile` |
| pipx | `uv tool install`, `uvx` |
| virtualenv | `uv venv` |
| pyenv | `uv python install/pin/find` |
| Poetry | reads pyproject.toml deps; can resolve any Poetry project |
| pdm | reads pyproject.toml deps |
| pep517 backends | full support (any backend works) |
| requirements.txt | `uv pip install -r`, `uv pip freeze` |
| pyproject.toml [project] | full support |
| pyproject.toml [tool.poetry] | dep extraction; no Poetry-specific resolution |
| PEP 723 inline metadata | full support |
| PEP 735 dependency groups | full support |
| PEP 751 universal lock | aligned, in progress |

The practical effect: in most projects, `uv` is a drop-in replacement for the existing tool, with speed wins.

## References

- uv documentation — https://docs.astral.sh/uv/
- uv source — https://github.com/astral-sh/uv
- pubgrub-rs — https://github.com/pubgrub-rs/pubgrub
- PubGrub design paper — https://medium.com/@nex3/pubgrub-2fb6470504f
- python-build-standalone — https://github.com/indygreg/python-build-standalone
- PEP 503 — Simple Repository API — https://peps.python.org/pep-0503/
- PEP 517 — Build system independent format — https://peps.python.org/pep-0517/
- PEP 518 — Build system requirements — https://peps.python.org/pep-0518/
- PEP 621 — Project metadata in pyproject.toml — https://peps.python.org/pep-0621/
- PEP 658 — Serve distribution metadata in PyPI Simple — https://peps.python.org/pep-0658/
- PEP 691 — JSON Simple API — https://peps.python.org/pep-0691/
- PEP 723 — Inline script metadata — https://peps.python.org/pep-0723/
- PEP 735 — Dependency Groups — https://peps.python.org/pep-0735/
- PEP 751 — Universal lockfile (in progress)
- PEP 440 — Version specification — https://peps.python.org/pep-0440/
- PEP 427 — Wheel format — https://peps.python.org/pep-0427/
- Charlie Marsh on uv — https://astral.sh/blog/uv
- uv vs pip benchmarks — https://github.com/astral-sh/uv/blob/main/BENCHMARKS.md
- Tokio async runtime — https://tokio.rs/
- Cargo workspaces — https://doc.rust-lang.org/cargo/reference/workspaces.html
- Ruff (Astral's lint tool) — https://docs.astral.sh/ruff/
- pyproject.toml specification — https://packaging.python.org/en/latest/specifications/pyproject-toml/
- Python Packaging User Guide — https://packaging.python.org/
- PyPI JSON API — https://warehouse.pypa.io/api-reference/json.html
- pipx — https://pypa.github.io/pipx/
- pyenv — https://github.com/pyenv/pyenv
- The case for a fast Python package manager — https://blog.crysis.io/a-fast-package-manager/ (general background)
