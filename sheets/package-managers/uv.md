# uv

An extremely fast Python package and project manager, written in Rust by Astral. Single static binary; replaces pip, pip-tools, virtualenv, pipx, pyenv, and parts of poetry/conda. PEP 517/518/621/660/723/735 native, with a deterministic PubGrub-based resolver and a global content-addressable cache. 10-100x faster than pip on cold installs, often 1000x on warm cache.

## Setup

### Install on macOS / Linux

```bash
# Astral install script — fetches the latest static binary into ~/.local/bin
curl -LsSf https://astral.sh/uv/install.sh | sh

# Pin a specific version of uv at install time
curl -LsSf https://astral.sh/uv/0.5.13/install.sh | sh

# Install without modifying shell profile
curl -LsSf https://astral.sh/uv/install.sh | env INSTALLER_NO_MODIFY_PATH=1 sh

# Install to a custom directory
curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR=/opt/uv sh
```

### Install on Windows

```bash
# PowerShell — Astral install script
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"

# Pin a specific version
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/0.5.13/install.ps1 | iex"
```

### Install via package managers

```bash
# Homebrew (macOS / Linuxbrew)
brew install uv

# pipx — install uv into a managed venv (note: ironic given uv replaces pipx)
pipx install uv

# pip — install into the system or active venv
pip install uv

# Cargo — build from source (requires Rust 1.83+)
cargo install --git https://github.com/astral-sh/uv uv

# Arch Linux (community repo)
pacman -S uv

# Alpine
apk add uv

# Conda / mamba
conda install -c conda-forge uv

# Nix
nix-env -iA nixpkgs.uv

# Docker — official image
docker pull ghcr.io/astral-sh/uv:latest
docker pull ghcr.io/astral-sh/uv:0.5-python3.12-bookworm-slim
```

### Self-update

```bash
uv self update                  # update to latest
uv self update 0.5.13           # update to a specific version
uv self update --token=$GH_TOKEN  # supply GH token if rate-limited
uv --version                    # check installed version
uv version                      # newer subcommand form (uv 0.5+)
```

### Shell completions

```bash
# bash
uv generate-shell-completion bash > /etc/bash_completion.d/uv

# zsh
uv generate-shell-completion zsh > "${fpath[1]}/_uv"

# fish
uv generate-shell-completion fish > ~/.config/fish/completions/uv.fish

# powershell
uv generate-shell-completion powershell | Out-String | Invoke-Expression
```

### Version pinning per-project

```bash
# .python-version — pinning a Python interpreter version (read by uv + pyenv + asdf)
echo "3.12" > .python-version
echo "3.12.7" > .python-version       # full triple, more strict

# .tool-versions — asdf / mise compatible
printf "uv 0.5.13\npython 3.12.7\n" > .tool-versions

# pyproject.toml — Python version constraint
[project]
requires-python = ">=3.10"
```

### Verify install

```bash
uv --version                    # uv 0.5.13 (a3b8d04 2025-01-15)
which uv                        # /Users/you/.local/bin/uv
uv help                         # top-level help
uv help pip install             # subcommand help
uv help pip                     # group help
```

### Uninstall

```bash
# Remove uv binary
rm ~/.local/bin/uv ~/.local/bin/uvx

# Remove cache + managed Python toolchains + tools
uv cache clean
rm -rf ~/.local/share/uv
rm -rf ~/.cache/uv

# Or all in one step
rm -rf ~/.cache/uv ~/.local/share/uv ~/.local/bin/uv ~/.local/bin/uvx
```

## Why uv

- **Speed** — Rust core, parallel network IO, content-addressable cache; 10-100x faster than pip on cold installs; near-instant on warm cache via copy-on-write hardlinking.
- **Replaces a stack of tools**:
  - `pip` (`uv pip install`)
  - `pip-tools` (`uv pip compile`, `uv pip sync`)
  - `virtualenv` (`uv venv`)
  - `pipx` (`uv tool install`, `uvx`)
  - `pyenv` (`uv python install`)
  - parts of `poetry` and `hatch` (`uv add/lock/sync`)
  - some `conda` use cases (Python-only — caveat: no non-Python deps)
- **Single binary** — one static Rust executable, ~30MB, no Python dependency for uv itself; bootstraps the Python interpreter from python-build-standalone.
- **Deterministic resolver** — PubGrub-based, like Cargo and Dart's pub. Produces minimal conflict explanations. The resolver runs cross-platform: a single lockfile resolves for Linux/macOS/Windows, glibc/musl, multiple architectures.
- **PEP 621 native** — pyproject.toml is the source of truth; no requirements.txt drift; generates a uv.lock for reproducibility.
- **Standards-first** — implements PEP 517 (build backends), PEP 518 (build-system table), PEP 621 ([project] metadata), PEP 660 (editable installs), PEP 723 (script inline metadata), PEP 735 (dependency groups).
- **Universal lockfile** — `uv.lock` resolves for every supported platform/marker, not just the CI machine; `uv sync` materialises only what the current platform needs.
- **Hardlinked installs** — when cache + venv share a filesystem, uv hardlinks wheels into the venv (zero copy, zero disk usage), with copy fallback for cross-fs.
- **Workspaces** — Cargo-style multi-package monorepo support with shared lockfile.
- **Tooling-agnostic** — does not force its own build backend; works with hatchling, poetry-core, flit-core, setuptools, maturin, scikit-build-core, pdm-backend, or its own `uv_build` (uv 0.5+).

## Architecture

### Global cache

- Default location: `~/.cache/uv` on Linux; `~/Library/Caches/uv` on macOS; `%LOCALAPPDATA%\uv\cache` on Windows.
- Override with `UV_CACHE_DIR` env var or `--cache-dir`.
- Content-addressable: wheels keyed by hash, deduplicated across all projects.
- Two main pools:
  - `wheels-v0/` — pre-built `.whl` files, indexed by SHA + URL.
  - `built-wheels-v0/` — wheels uv built locally from sdists, indexed by source dist hash.
  - `archive-v0/` — extracted wheel archives (the install-ready tree).
  - `interpreter-v0/` — Python interpreter metadata cache.
  - `git-v0/` — cloned git repos for `git+https://...` deps.
  - `simple-v0/` — JSON metadata fetched from PyPI Simple API.

### Wheel + source-distribution download flow

1. Resolver asks index (PyPI Simple API) for available versions of each requirement.
2. Resolver picks a version per platform/marker; queries metadata (preferring `.whl` METADATA via range requests for speed).
3. Selected wheel/sdist is downloaded into the cache.
4. Source distributions are built into wheels via PEP 517 (uses uv-managed Python or system Python).
5. Wheels are extracted into `archive-v0/`, then hardlinked / copy-on-written into the venv's `site-packages/`.

### The PubGrub resolver

- Same algorithm as Cargo (Rust) and Pub (Dart). Produces minimal conflict explanations rather than raw "incompatible deps" errors.
- Output is a single complete solution per platform marker. uv runs the resolver once, producing the universal `uv.lock`.
- Resolution strategies (`--resolution`):
  - `highest` (default) — newest compatible version
  - `lowest` — oldest version satisfying the constraint (good for CI lower-bound checks)
  - `lowest-direct` — lowest for direct deps, highest for transitives
- Pre-release handling (`--prerelease`):
  - `disallow` (default for direct deps without explicit pre-release marker)
  - `allow` — include pre-releases
  - `if-necessary` — only if no stable release satisfies
  - `explicit` — only allow pre-releases for packages with explicit pre-release constraint
  - `if-necessary-or-explicit`

### PEP relationships

- **PEP 517** — Build-system independence. uv calls the project's declared build backend (hatchling, setuptools, etc.) via subprocess to build sdists/wheels.
- **PEP 518** — `[build-system]` table in pyproject.toml. uv reads `requires` and `build-backend` from there.
- **PEP 621** — `[project]` metadata. uv reads `name`, `version`, `dependencies`, `optional-dependencies`, etc., directly.
- **PEP 660** — Editable installs (`uv pip install -e .` and `uv sync` with the project itself).
- **PEP 723** — Inline script metadata for single-file scripts (`# /// script`).
- **PEP 735** — Dependency groups, the standard replacement for `[tool.uv.dev-dependencies]` (uv 0.5+).
- **PEP 508** — Dependency specifiers (e.g., `numpy>=1.26; python_version>='3.10'`).
- **PEP 440** — Version specifier syntax (`==`, `>=`, `~=`, `===`).

### File layout in a uv-managed project

```bash
my-project/
├── pyproject.toml       # source of truth: deps, metadata, build config
├── uv.lock              # locked resolution (commit this!)
├── .python-version      # pinned Python interpreter (commit this)
├── .venv/               # auto-managed virtual env (gitignore this)
│   ├── bin/python       # symlink/junction to managed Python
│   └── lib/python3.12/site-packages/
├── src/
│   └── my_project/
│       └── __init__.py
└── tests/
```

## pip-Compatible Mode vs Project Mode

uv operates in two distinct modes. Picking the right one matters.

### pip-compatible mode — `uv pip ...`

Imperatively manipulates an active venv, just like classic pip. Useful for ad-hoc work, legacy projects with `requirements.txt`, or scripts that must remain pip-compatible.

```bash
# Activate a venv, then use uv pip like pip
source .venv/bin/activate
uv pip install requests          # installs into ./.venv

# Or specify the target Python explicitly
uv pip install --python /usr/bin/python3.11 requests
uv pip install --python .venv/bin/python requests
```

When uv pip runs without a venv active, it errors out unless `--system` is passed:

```bash
uv pip install --system numpy    # install into system Python (use sparingly!)
uv pip install --system --break-system-packages numpy  # honour PEP 668 override
```

### Project mode — `uv add/remove/lock/sync/run`

Declarative — pyproject.toml + uv.lock are the source of truth. uv manages `.venv/` for you, runs the resolver against the full project graph, and produces a deterministic install.

```bash
uv init my-project               # scaffold pyproject.toml
cd my-project
uv add requests                  # adds to [project].dependencies, locks, syncs
uv add --dev pytest              # adds to dev group
uv sync                          # materialise the lockfile into .venv
uv run pytest                    # run inside the project env
```

### How uv decides which mode

- `uv pip <verb>` always pip-mode.
- `uv add/remove/lock/sync/run/tree/export/init` always project-mode (requires pyproject.toml).
- `uv venv` is mode-neutral — creates a venv usable in either mode.

### When to use each

- **pip mode** — quick experiments, single-script venvs, legacy projects, CI installs from a hand-curated requirements.txt, environments not meant to be checked in.
- **Project mode** — anything you intend to develop, distribute, or reproduce. Lockfile guarantees identical installs across machines.

### Mixing them — caveat

Avoid running `uv pip install foo` in a project-mode `.venv`. The lockfile won't know about `foo`, so the next `uv sync` will remove it. Either use project mode end-to-end, or use a separate venv outside the project.

## Python Toolchain Management

uv ships its own Python toolchain manager. Replaces pyenv. Uses python-build-standalone (statically linked, relocatable Python builds from indygreg's project, now an Astral-maintained project).

### List interpreters

```bash
uv python list                            # available + installed Python versions
uv python list --all-versions             # every version (huge list)
uv python list --all-platforms            # cross-platform listings
uv python list --only-installed           # only what you've already installed
uv python list --output-format json       # machine-readable
```

Output columns: `cpython-3.12.7-linux-x86_64-gnu`, install path, download URL, install state.

### Install / uninstall

```bash
uv python install 3.12                    # latest 3.12.x
uv python install 3.12.7                  # exact version
uv python install 3.10 3.11 3.12 3.13     # multiple at once
uv python install pypy@3.10               # PyPy
uv python install cpython-3.12.7-linux-x86_64-gnu   # explicit toolchain id
uv python install --reinstall 3.12        # force redownload + reinstall
uv python install --mirror https://my.org/python  # custom mirror

uv python uninstall 3.10
uv python uninstall --all                 # uninstall every managed Python
```

### Find an interpreter

```bash
uv python find                            # path of the resolved Python for cwd
uv python find 3.12                       # path of installed 3.12
uv python find '>=3.11,<3.13'             # match a specifier
uv python find --no-project               # ignore pyproject's requires-python
```

### Pin a project to a Python version

```bash
uv python pin 3.12                        # writes .python-version
uv python pin 3.12.7
uv python pin --resolved                  # write the full triple, not the minor
cat .python-version                       # 3.12
```

`uv run`, `uv venv`, `uv sync` all read `.python-version` automatically.

### Where managed Pythons live

```bash
uv python dir                             # ~/.local/share/uv/python
ls "$(uv python dir)"
# cpython-3.12.7-darwin-aarch64-none/
# cpython-3.13.1-darwin-aarch64-none/
```

Override with `UV_PYTHON_INSTALL_DIR`.

### Offline / air-gapped installs

```bash
# Mirror python-build-standalone tarballs to your own server
# Then point uv at it:
export UV_PYTHON_INSTALL_MIRROR=https://my.corp/python-build-standalone

uv python install 3.12

# Permanent setting via env var or in pyproject.toml:
[tool.uv]
python-install-mirror = "https://my.corp/python-build-standalone"
```

### Force-prefer managed vs system

```bash
# Always prefer managed (default)
uv run --python-preference managed script.py

# Only managed — error if not installed
uv run --python-preference only-managed script.py

# Prefer system, fall back to managed
uv run --python-preference system script.py

# Only system — never download
uv run --python-preference only-system script.py
```

### python-build-standalone caveats

- Statically linked OpenSSL — slightly different cipher suite vs distro Python.
- No `tkinter` on minimal builds (request `+tkinter` variant).
- Some C extensions that vendor `_ssl` paths may need rebuilding.
- musl support: alpine builds use the `-musl` triple.

## venv Management

### Basic creation

```bash
uv venv                                   # create .venv with Python from .python-version (or current)
uv venv .venv-myname                      # custom path
uv venv --python 3.12                     # specific Python
uv venv --python 3.12.7                   # exact triple
uv venv --python /usr/bin/python3.11      # use system Python at this path
uv venv --python python3.12               # search PATH
```

### Useful flags

```bash
uv venv --seed                            # pre-install pip/setuptools/wheel into venv
uv venv --prompt myproj                   # custom prompt name shown in PS1
uv venv --system-site-packages            # inherit system site-packages (rare; use --no-system-site-packages to disable)
uv venv --allow-existing                  # don't error if .venv already exists
uv venv --no-project                      # don't read pyproject.toml requires-python
uv venv --python-preference only-managed  # ensure managed Python only
uv venv --link-mode copy                  # copy files instead of hardlink
uv venv --link-mode hardlink              # default when on same fs
uv venv --link-mode symlink               # symlinks (Linux/macOS only)
uv venv --link-mode clone                 # APFS / btrfs / xfs reflink (CoW)
uv venv --relocatable                     # produce a venv whose paths are relative
```

### Activate / deactivate

```bash
# bash / zsh
source .venv/bin/activate
deactivate

# fish
source .venv/bin/activate.fish

# Windows cmd
.venv\Scripts\activate.bat

# Windows PowerShell
.venv\Scripts\Activate.ps1
```

Many uv commands work without an activate — `uv run`, `uv pip --python .venv/bin/python`, etc.

### Inspect the venv

```bash
.venv/bin/python --version
.venv/bin/python -c "import sys; print(sys.executable, sys.prefix)"
uv pip list --python .venv/bin/python
uv pip freeze --python .venv/bin/python
```

### Delete the venv

```bash
rm -rf .venv
# Or recreate from scratch
uv venv --allow-existing
uv sync
```

## pip-Compat Subcommands

Drop-in replacements for `pip`. Use these inside an active venv or with `--python`.

### `uv pip install`

```bash
uv pip install requests                              # latest
uv pip install 'requests>=2.31,<3'                   # specifier
uv pip install requests==2.31.0                      # exact
uv pip install requests urllib3                      # multiple
uv pip install -r requirements.txt                   # from file
uv pip install -r requirements.txt -r dev.txt        # multiple files
uv pip install -e .                                  # editable install of cwd
uv pip install -e /path/to/pkg                       # editable install of path
uv pip install -e '.[dev,test]'                      # editable + extras
uv pip install '.[dev]'                              # extras only
uv pip install git+https://github.com/psf/requests.git
uv pip install git+https://github.com/psf/requests.git@v2.31.0
uv pip install git+ssh://git@github.com/org/private.git
uv pip install 'requests @ git+https://github.com/psf/requests.git'
uv pip install https://example.com/pkg-1.0-py3-none-any.whl
uv pip install ./local-wheel.whl

# Upgrade flags
uv pip install --upgrade requests                    # bump requests
uv pip install -U requests                           # short
uv pip install --upgrade-package requests            # only this dep, leave others
uv pip install -U requests --upgrade-package urllib3 # mix
uv pip install --reinstall requests                  # force reinstall
uv pip install --reinstall-package requests          # only this dep
uv pip install --refresh                             # bypass cache, redownload
uv pip install --refresh-package requests            # bypass for this dep

# Resolution flags
uv pip install --pre 'requests>=3'                   # allow pre-releases
uv pip install --no-deps requests                    # skip transitive deps
uv pip install --resolution highest requests         # default
uv pip install --resolution lowest requests          # oldest compatible
uv pip install --resolution lowest-direct requests   # lowest for top-level only

# Index / find-links
uv pip install --index-url https://pypi.org/simple requests
uv pip install --extra-index-url https://my.corp/simple my-private-pkg
uv pip install --find-links /path/to/wheels requests
uv pip install --find-links https://my.org/wheels/ requests
uv pip install --no-index --find-links ./wheels requests   # offline-only

# Build flags
uv pip install --no-build numpy                      # require pre-built wheel
uv pip install --no-binary numpy                     # force build from sdist
uv pip install --only-binary :all: numpy             # all packages binary-only
uv pip install --no-build-isolation -e .             # use current env's build deps
uv pip install --no-build-isolation-package torch    # for one package only
uv pip install --config-setting='--build-option=--global-option' pkg

# Constraints / overrides
uv pip install -r reqs.txt -c constraints.txt        # bound transitive versions
uv pip install -r reqs.txt --override overrides.txt  # force versions

# Target / prefix
uv pip install --target ./vendor requests            # install to a directory
uv pip install --prefix /usr/local requests          # install with this prefix
uv pip install --root /tmp/sysroot requests          # install with chrooted root

# Offline
uv pip install --offline requests                    # cache-only, fail if missing
```

### `uv pip uninstall`

```bash
uv pip uninstall requests
uv pip uninstall -r requirements.txt
uv pip uninstall requests urllib3 idna
uv pip uninstall --break-system-packages requests    # PEP 668 override
```

### `uv pip list`

```bash
uv pip list                                          # all installed
uv pip list --outdated                               # show updates available
uv pip list --format json                            # JSON output
uv pip list --format freeze                          # requirements.txt-style
uv pip list --format columns                         # default human-readable
uv pip list --editable                               # only editable installs
uv pip list --exclude-editable                       # hide editable
uv pip list --exclude pkg1 --exclude pkg2            # hide specific
```

### `uv pip show`

```bash
uv pip show requests                                 # metadata
uv pip show -f requests                              # include file list
uv pip show --files requests                         # same
uv pip show requests urllib3                         # multiple
```

### `uv pip freeze`

```bash
uv pip freeze                                        # all installed, requirements-style
uv pip freeze > requirements.lock
uv pip freeze --exclude-editable                     # hide -e installs
uv pip freeze --strict                               # fail if env is inconsistent
uv pip freeze --exclude pkg1                         # hide specific
```

### `uv pip compile` — the pip-tools replacement

Reads `requirements.in` (or pyproject.toml extras), resolves, writes a fully pinned `requirements.txt`.

```bash
uv pip compile requirements.in -o requirements.txt
uv pip compile pyproject.toml -o requirements.txt    # compile from [project].dependencies
uv pip compile pyproject.toml --extra dev -o dev-requirements.txt
uv pip compile pyproject.toml --all-extras -o requirements.txt
uv pip compile -r requirements.in -r dev.in -o all.txt

# Strategies
uv pip compile --resolution highest requirements.in
uv pip compile --resolution lowest requirements.in       # CI lower-bound test
uv pip compile --resolution lowest-direct requirements.in

# Targeting
uv pip compile --python-version 3.10 requirements.in     # resolve as if on 3.10
uv pip compile --python-version 3.12 --python-platform linux requirements.in
uv pip compile --python-platform x86_64-unknown-linux-gnu requirements.in
uv pip compile --python-platform aarch64-apple-darwin requirements.in
uv pip compile --universal requirements.in               # cross-platform lockfile

# Upgrade
uv pip compile --upgrade requirements.in                 # bump everything
uv pip compile -U requirements.in
uv pip compile --upgrade-package requests requirements.in # bump just this
uv pip compile -P requests -P urllib3 requirements.in    # multiple specific bumps

# Output formatting
uv pip compile --output-file requirements.txt requirements.in
uv pip compile -o reqs.txt requirements.in
uv pip compile --no-header requirements.in               # omit "by uv compile" header
uv pip compile --emit-index-url requirements.in          # include --index-url in output
uv pip compile --emit-find-links requirements.in
uv pip compile --emit-build-options requirements.in
uv pip compile --no-emit-package pkg1 requirements.in    # exclude from output
uv pip compile --no-strip-extras requirements.in         # keep [extras] in output
uv pip compile --no-strip-markers requirements.in        # keep ; markers
uv pip compile --no-annotate requirements.in             # no "# via X" comments
uv pip compile --annotation-style=line                   # one-line comments
uv pip compile --annotation-style=split                  # default

# Hashes
uv pip compile --generate-hashes requirements.in         # SHA hashes per pkg

# Constraints / overrides
uv pip compile -c constraints.txt requirements.in
uv pip compile --override overrides.txt requirements.in

# Resolver tuning
uv pip compile --resolver pubgrub requirements.in        # default
uv pip compile --pre requirements.in                     # allow pre-releases
uv pip compile --prerelease allow requirements.in
uv pip compile --index-strategy first-index requirements.in   # default
uv pip compile --index-strategy unsafe-best-match requirements.in
uv pip compile --index-strategy unsafe-first-match requirements.in
```

### `uv pip sync`

Replace the venv contents with exactly what's in the requirements file. Anything not listed is uninstalled.

```bash
uv pip sync requirements.txt                         # sync to lock state
uv pip sync requirements.txt dev-requirements.txt    # multi-file sync
uv pip sync --strict requirements.txt                # error on inconsistent env after
uv pip sync --reinstall requirements.txt             # full reinstall
uv pip sync --reinstall-package requests requirements.txt
uv pip sync --no-deps requirements.txt               # skip transitive resolution
uv pip sync --no-allow-empty-requirements requirements.txt
uv pip sync --refresh requirements.txt               # bypass cache
uv pip sync --break-system-packages requirements.txt # PEP 668 override
uv pip sync --python /usr/bin/python3.11 requirements.txt
```

### `uv pip check`

```bash
uv pip check                                         # detect missing/conflicting deps
# Prints "<pkg> X.Y requires <pkg2><Z, but you have <pkg2> W"
```

### `uv pip tree`

```bash
uv pip tree                                          # dep graph in current env
uv pip tree --depth 1                                # top-level only
uv pip tree --invert                                 # reverse-deps view
uv pip tree --package requests                       # subtree
uv pip tree --no-dedupe                              # show repeats
uv pip tree --outdated                               # mark outdated
```

## Project Mode Subcommands

Source of truth is `pyproject.toml` + `uv.lock`.

### `uv init`

```bash
uv init                                              # init in cwd, app project
uv init my-project                                   # create directory + init
uv init --name my-package my-project                 # custom package name
uv init --lib                                        # library project (uses src/ layout)
uv init --app                                        # application (default; flat layout)
uv init --package                                    # treat as installable package
uv init --no-package                                 # script-only project, no build
uv init --build-backend hatch                        # hatchling
uv init --build-backend uv                           # uv_build (uv 0.5+)
uv init --build-backend setuptools
uv init --build-backend flit                         # flit-core
uv init --build-backend poetry                       # poetry-core
uv init --build-backend maturin                      # Rust extensions
uv init --build-backend scikit                       # scikit-build-core (C/C++)
uv init --vcs git                                    # init git repo + .gitignore
uv init --vcs none
uv init --python 3.12                                # set Python pin
uv init --description "My cool project"
uv init --author-from git                            # use git config user.name/email
uv init --no-readme
uv init --no-pin-python                              # skip .python-version
uv init --bare                                       # minimal — only pyproject.toml
```

### `uv add`

```bash
uv add requests                                      # add to [project].dependencies, lock+sync
uv add 'requests>=2.31,<3'                           # with constraint
uv add requests urllib3                              # multiple
uv add --dev pytest                                  # legacy [tool.uv.dev-dependencies]
uv add --group test pytest                           # PEP 735 group
uv add --group lint ruff                             # add to [dependency-groups].lint
uv add --optional gpu torch                          # to [project.optional-dependencies].gpu
uv add --extra dev .                                 # add self with extras (rare)

# Source-specified
uv add 'mypkg @ git+https://github.com/me/mypkg.git'
uv add 'mypkg @ git+https://github.com/me/mypkg.git@main'
uv add 'mypkg @ git+ssh://git@github.com/org/private.git'
uv add 'mypkg @ git+https://github.com/me/mypkg.git@v1.0.0'    # tag
uv add 'mypkg @ file:///path/to/mypkg'
uv add ../my-local-pkg                               # local path
uv add --editable ../my-local-pkg                    # editable local

# Workspace
uv add --workspace ./packages/my-pkg

# Behaviour flags
uv add --no-sync requests                            # update pyproject + lock, don't install
uv add --frozen requests                             # use existing lock; error if needed update
uv add --locked requests                             # assert lock is up-to-date
uv add --no-build-package mypkg                      # require wheel for this dep
uv add --no-binary-package mypkg                     # build from sdist
uv add --build-constraint constraints.txt requests   # constrain build deps
uv add --raw-sources requests                        # don't write [tool.uv.sources]
uv add --refresh requests                            # bypass cache
uv add --upgrade requests                            # also upgrade existing
uv add --resolution lowest-direct requests
uv add --prerelease allow 'foo>=2.0.0a1'
```

### `uv remove`

```bash
uv remove requests                                   # remove from [project].dependencies
uv remove --dev pytest                               # from dev-dependencies
uv remove --group test pytest                        # from [dependency-groups].test
uv remove --optional gpu torch                       # from optional-deps
uv remove --no-sync requests                         # update files, don't uninstall
uv remove --frozen requests
```

### `uv lock`

Regenerates `uv.lock` from `pyproject.toml`. Does not touch `.venv`.

```bash
uv lock                                              # resolve & write uv.lock
uv lock --upgrade                                    # bump all versions to latest compatible
uv lock --upgrade-package requests                   # bump only this dep
uv lock -P requests -P urllib3                       # multiple bumps (short form)
uv lock --frozen                                     # check-only; error if lock is stale
uv lock --locked                                     # alias for --frozen on lock
uv lock --check                                      # explicit check-only mode
uv lock --refresh                                    # ignore cache, refetch metadata
uv lock --refresh-package requests                   # for one dep
uv lock --resolution lowest                          # for compat tests
uv lock --resolution lowest-direct
uv lock --prerelease allow
uv lock --no-cache                                   # skip cache entirely
uv lock --offline                                    # cache-only
uv lock --python 3.10                                # resolve as if on 3.10
uv lock --python-platform linux                      # cross-platform target
uv lock --build-constraint constraints.txt
```

### `uv sync`

Materialises `uv.lock` into `.venv`. The principal install command for project mode.

```bash
uv sync                                              # install full default set into .venv
uv sync --frozen                                     # error if uv.lock is stale (don't regenerate)
uv sync --locked                                     # check lock is up-to-date but don't update it
uv sync --upgrade                                    # alias for `uv lock --upgrade && uv sync`
uv sync --upgrade-package requests
uv sync --reinstall                                  # purge + reinstall everything
uv sync --reinstall-package requests
uv sync --refresh                                    # bypass cache
uv sync --refresh-package requests

# Group / extras selection
uv sync --all-extras                                 # install every extra in [project.optional-dependencies]
uv sync --extra gpu                                  # specific extra
uv sync --extra gpu --extra ml
uv sync --no-dev                                     # skip dev deps
uv sync --only-dev                                   # only dev deps (no project itself)
uv sync --group test                                 # PEP 735 group
uv sync --group test --group lint
uv sync --no-default-groups                          # skip [tool.uv].default-groups
uv sync --only-group test                            # only this group
uv sync --all-groups                                 # every dependency group

# Project-self handling
uv sync --no-install-project                         # don't install the current pkg itself
uv sync --no-install-package mypkg                   # skip a workspace member
uv sync --inexact                                    # don't uninstall stale packages
uv sync --no-editable                                # install non-editable even if [tool.uv.sources] says editable

# Reproducibility
uv sync --frozen                                     # the canonical CI flag
uv sync --no-build                                   # require pre-built wheels for everything
uv sync --no-binary                                  # build everything from sdist
uv sync --offline                                    # cache-only

# Compile bytecode
uv sync --compile-bytecode                           # pre-compile .pyc (slower install, faster startup)
```

### `--frozen` vs `--locked` semantics

- `--frozen` — use the existing `uv.lock` exactly. Don't regenerate. Fail if the lock can't satisfy the install.
- `--locked` — check the lock is up-to-date with pyproject.toml; fail if not. Don't regenerate.
- Default — regenerate the lock if pyproject.toml changed, then sync.

CI rule of thumb: use `--frozen` to ensure no resolver drift between dev and CI.

### `uv export`

Emit a requirements.txt-style file from the lock for tools that don't speak `uv.lock`.

```bash
uv export                                            # to stdout, requirements-txt format
uv export -o requirements.txt
uv export --format requirements-txt -o requirements.txt
uv export --no-hashes -o reqs.txt                    # skip --hash entries
uv export --no-emit-project                          # exclude the project itself
uv export --no-emit-package mypkg                    # exclude specific
uv export --no-dev                                   # production reqs
uv export --only-dev                                 # only dev reqs
uv export --extra gpu                                # include an extra
uv export --all-extras
uv export --group test
uv export --frozen                                   # use existing lock; don't regenerate
uv export --locked
uv export --no-annotate                              # remove "# via foo" comments
uv export --no-header                                # remove the uv banner comment
```

### `uv tree`

```bash
uv tree                                              # full project graph
uv tree --depth 1                                    # top-level deps only
uv tree --depth 2
uv tree --invert                                     # reverse-deps mode
uv tree --package requests                           # subtree only
uv tree --no-dedupe                                  # repeat deps shown each time
uv tree --outdated                                   # show "(latest: X.Y)" hints
uv tree --frozen                                     # use existing lock
uv tree --locked
uv tree --universal                                  # show all platform variants
uv tree --no-dev                                     # exclude dev
uv tree --only-dev
uv tree --group test
uv tree --all-groups
uv tree --python-version 3.10                        # resolve as if on 3.10
uv tree --python-platform linux
```

### `uv run`

The "do thing inside the project's environment" command. Auto-syncs first (unless `--no-sync` or `--frozen`).

```bash
uv run python                                        # REPL inside .venv
uv run python script.py                              # run a script
uv run python -m pytest                              # module
uv run pytest                                        # entry-point installed in .venv
uv run -- ruff check .                               # `--` ends uv flag parsing
uv run pytest -- -k 'test_foo' --tb=short            # pass args to pytest after `--`

# With ad-hoc deps (don't pollute pyproject)
uv run --with rich script.py
uv run --with 'rich>=13' --with httpx script.py
uv run --with-requirements extras.txt script.py
uv run --with-editable ../mylib script.py            # ad-hoc editable

# Sync controls
uv run --frozen pytest                               # use lock as-is, error on staleness
uv run --locked pytest                               # check lock is current
uv run --no-sync pytest                              # skip auto-sync
uv run --refresh pytest                              # bypass cache
uv run --reinstall pytest

# Python selection
uv run --python 3.12 script.py                       # use a specific Python
uv run --python /usr/bin/python3.11 script.py
uv run --python-preference only-managed script.py

# Group / extras
uv run --all-extras pytest
uv run --extra gpu python -m torch.something
uv run --group test pytest
uv run --no-dev script.py
uv run --only-dev pytest

# Project context
uv run --no-project script.py                        # ignore pyproject.toml in cwd
uv run --directory ../other-project pytest           # run inside that project
uv run --package mypkg pytest                        # run from a specific workspace member

# Environment
uv run --env-file .env script.py                     # load dotenv into env
uv run --isolated --with rich script.py              # ephemeral venv, no project
```

### PEP 723 inline-metadata scripts — `uv run --script`

Single-file scripts can declare deps inline:

```bash
#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "rich",
#     "httpx>=0.27",
# ]
# ///

import httpx
from rich import print

print(httpx.get("https://httpbin.org/get").json())
```

Then:

```bash
uv run --script foo.py                               # auto-detect script header
uv run --script foo.py arg1 arg2                     # pass args
uv run foo.py                                        # if `# /// script` present, autodetect

# Manage script dependencies
uv add --script foo.py rich                          # add inline dep
uv add --script foo.py 'httpx>=0.27' polars
uv remove --script foo.py rich
uv lock --script foo.py                              # generate `# requires-lock` block
uv sync --script foo.py                              # ensure script env is built
uv tree --script foo.py                              # show script's resolved tree
```

uv stores script venvs in `~/.cache/uv/environments-v0/`, keyed by script hash.

## pyproject.toml — the [project] table (PEP 621)

```bash
[project]
name = "my-project"
version = "0.1.0"
description = "A short summary."
readme = "README.md"
requires-python = ">=3.10"
license = { text = "MIT" }
license-files = ["LICEN[CS]E*"]                      # PEP 639
keywords = ["cli", "tool"]
authors = [
    { name = "Stevie Bellis", email = "stevie@bellis.tech" },
]
maintainers = [
    { name = "Stevie Bellis", email = "stevie@bellis.tech" },
]
classifiers = [
    "Development Status :: 4 - Beta",
    "Programming Language :: Python :: 3",
    "Programming Language :: Python :: 3.12",
    "License :: OSI Approved :: MIT License",
    "Operating System :: OS Independent",
]
dependencies = [
    "httpx>=0.27",
    "rich>=13",
    "pydantic>=2.6",
]
dynamic = ["version"]                                # version comes from VCS / build backend

[project.optional-dependencies]
gpu = ["torch>=2.0"]
ml = ["scikit-learn>=1.3", "pandas>=2.1"]
all = ["my-project[gpu,ml]"]

[project.urls]
Homepage = "https://github.com/me/my-project"
Documentation = "https://my-project.readthedocs.io"
Repository = "https://github.com/me/my-project.git"
Issues = "https://github.com/me/my-project/issues"
Changelog = "https://github.com/me/my-project/blob/main/CHANGELOG.md"

[project.scripts]
my-cli = "my_project.cli:main"
my-tool = "my_project.tool:run"

[project.gui-scripts]
my-gui = "my_project.gui:start"

[project.entry-points."pytest11"]
my-plugin = "my_project.pytest_plugin"

[project.entry-points."console_scripts"]
my-extra = "my_project.cli:extra"
```

## pyproject.toml — the [tool.uv] table

```bash
[tool.uv]
# Legacy dev deps (predates PEP 735); still supported but prefer [dependency-groups].dev
dev-dependencies = [
    "pytest>=7",
    "ruff>=0.4",
    "mypy>=1.8",
]

# Default groups installed by `uv sync`
default-groups = ["dev", "lint"]

# Override transitive resolution
override-dependencies = [
    "urllib3>=2.0,<3",                               # force urllib3 into this range
    "certifi>=2024.0.0",
]

# Constrain build-time deps
constraint-dependencies = [
    "setuptools<70",
    "wheel<0.43",
]

# Build-isolation overrides
no-build-isolation-package = [
    "torch",                                         # use current env's build deps
    "vllm",
]

# Source / wheel preferences
no-binary = false                                    # default: prefer wheels
no-binary-package = ["torch"]                        # force build from sdist for torch
only-binary = false
only-binary-package = ["scipy"]                      # require pre-built wheel

# Index pinning
find-links = ["https://download.pytorch.org/whl/cu121"]
index-url = "https://pypi.org/simple"
extra-index-url = ["https://my.corp/simple"]

# Conflict markers (uv 0.5+) — e.g. extras that can't coexist
conflicts = [
    [
        { extra = "cu121" },
        { extra = "cu118" },
    ],
]

# Per-environment specific resolution (uv 0.5+)
environments = [
    "sys_platform == 'linux'",
    "sys_platform == 'darwin'",
]

# Required uv version for this project
required-version = ">=0.5"

# Cache control
cache-keys = [
    { file = "pyproject.toml" },
    { file = "uv.lock" },
    { git = { commit = true } },
]

# Workspace declaration
[tool.uv.workspace]
members = ["packages/*"]
exclude = ["packages/legacy-*"]
```

## pyproject.toml — [tool.uv.sources]

Override the URL/path/git source for any dep declared in `[project].dependencies`. Doesn't change the version specifier — just where the package comes from.

```bash
[project]
dependencies = [
    "my-lib",                                        # version comes from sources or registry
    "torch",
    "private-pkg",
]

[tool.uv.sources]
# Git source with various pin types
my-lib = { git = "https://github.com/me/my-lib.git" }
my-lib = { git = "https://github.com/me/my-lib.git", rev = "abc1234" }      # commit
my-lib = { git = "https://github.com/me/my-lib.git", tag = "v1.0.0" }       # tag
my-lib = { git = "https://github.com/me/my-lib.git", branch = "main" }      # branch
my-lib = { git = "https://github.com/me/my-lib.git", subdirectory = "subpkg" }
my-lib = { git = "ssh://git@github.com/org/private.git", tag = "v1.0" }

# Local path
my-utils = { path = "../my-utils" }
my-utils = { path = "../my-utils", editable = true }

# Direct URL to wheel/sdist
torch = { url = "https://download.pytorch.org/whl/cu121/torch-2.3.0%2Bcu121-cp312-cp312-linux_x86_64.whl" }

# Workspace member (Cargo-style)
my-utils = { workspace = true }

# Environment-specific source
my-lib = [
    { git = "https://github.com/me/my-lib.git", marker = "sys_platform == 'linux'" },
    { path = "../my-lib", marker = "sys_platform == 'darwin'" },
]

# Index pinning per package
private-pkg = { index = "my-corp" }                  # require this index by name

# Multiple sources via marker
torch = [
    { index = "pytorch-cu121", marker = "platform_system == 'Linux'" },
    { index = "pytorch-cpu", marker = "platform_system == 'Darwin'" },
]

[[tool.uv.index]]
name = "my-corp"
url = "https://my.corp/simple"
explicit = true                                      # only use when referenced

[[tool.uv.index]]
name = "pytorch-cu121"
url = "https://download.pytorch.org/whl/cu121"
explicit = true
```

## pyproject.toml — [dependency-groups] (PEP 735)

The standard replacement for `[tool.uv.dev-dependencies]`. Multiple named groups, each independently installable.

```bash
[dependency-groups]
dev = [
    "pytest>=7",
    "ruff>=0.4",
    {include-group = "test"},                        # transclude another group
]
test = [
    "pytest>=7",
    "pytest-cov>=4",
    "pytest-xdist>=3",
]
lint = [
    "ruff>=0.4",
    "mypy>=1.8",
]
docs = [
    "sphinx>=7",
    "myst-parser>=2",
]
typecheck = [
    "mypy>=1.8",
    {include-group = "test"},
]

[tool.uv]
default-groups = ["dev"]                             # what `uv sync` installs by default
```

Use:

```bash
uv add --group test pytest
uv sync --group test --group lint
uv run --group test pytest
uv export --group test -o test-requirements.txt
uv sync --no-default-groups                          # only project deps, no groups
uv sync --all-groups                                 # every group
```

Group ordering is controlled by `default-groups`. Explicit `--group` flags override defaults.

## Workspaces

Cargo-style monorepo support. Multiple Python packages share a single venv and lockfile.

### Layout

```bash
my-monorepo/
├── pyproject.toml              # workspace root
├── uv.lock                     # single shared lockfile
├── .venv/                      # single shared venv
└── packages/
    ├── core/
    │   ├── pyproject.toml      # member 1
    │   └── src/core/
    ├── api/
    │   ├── pyproject.toml      # member 2
    │   └── src/api/
    └── cli/
        ├── pyproject.toml
        └── src/cli/
```

### Workspace root pyproject.toml

```bash
[project]
name = "my-monorepo"
version = "0.0.0"
requires-python = ">=3.12"
dependencies = []

[tool.uv.workspace]
members = ["packages/*"]
exclude = ["packages/legacy-*", "packages/.skip"]

[tool.uv.sources]
core = { workspace = true }
api = { workspace = true }
cli = { workspace = true }
```

### Member pyproject.toml

```bash
[project]
name = "api"
version = "0.1.0"
dependencies = [
    "core",                                          # source from workspace
    "fastapi>=0.110",
]

[tool.uv.sources]
core = { workspace = true }
```

### Workspace commands

```bash
uv sync                                              # install all members + their deps into root .venv
uv sync --package api                                # only `api` and its deps
uv sync --package api --package cli                  # multiple
uv sync --no-install-package legacy                  # skip a member
uv add --package api fastapi                         # add a dep to specific member
uv add --workspace ./packages/new-pkg                # add a new workspace member
uv run --package api uvicorn api.main:app
uv lock                                              # locks all members
uv tree --package api                                # tree for one member
uv build --package api                               # build a single member's wheel
uv build --all-packages                              # build every member
```

### Workspace design rules

- Single `uv.lock` at root — no per-package locks.
- Single `.venv` at root — every member's deps coexist.
- Members can declare each other as deps using `{ workspace = true }`.
- Editable installs are the default for workspace members.
- Use `[tool.uv.sources]` at root to declare workspace source overrides for shared transitive deps.

## Tools — global isolated apps

Replaces `pipx`. Installs CLI tools into isolated venvs, not the project's venv.

```bash
# Install
uv tool install ruff                                 # global isolated venv for ruff
uv tool install 'ruff==0.5.0'                        # pin
uv tool install ruff black mypy                      # multiple
uv tool install --python 3.12 ruff                   # specific Python
uv tool install --with pytest-cov pytest             # extra deps in tool's env
uv tool install --editable .                         # install local pkg as a tool
uv tool install -U ruff                              # upgrade if installed
uv tool install --reinstall ruff                     # force reinstall

# Run ephemerally (no install)
uv tool run ruff check .                             # download + run, cache for next time
uv tool run ruff@0.5.0 check .                       # specific version
uv tool run --with rich httpx                        # add deps to tool's ephemeral env
uv tool run --from httpx http                        # run http binary from httpx pkg

# uvx alias
uvx ruff check .                                     # shorthand for `uv tool run`
uvx ruff@latest check .
uvx --with pytest-cov pytest

# List / inspect
uv tool list                                         # installed tools + their commands
uv tool list --show-paths                            # include venv paths
uv tool list --show-version-specifiers
uv tool dir                                          # ~/.local/share/uv/tools
uv tool dir --bin                                    # ~/.local/bin (where shims live)

# Update
uv tool upgrade ruff                                 # upgrade single tool
uv tool upgrade --all                                # upgrade all
uv tool upgrade --python 3.12 ruff                   # change Python while upgrading

# Remove
uv tool uninstall ruff
uv tool uninstall --all

# Update PATH
uv tool update-shell                                 # add ~/.local/bin to PATH in shell rc
```

### How it works

- Each tool gets its own venv at `~/.local/share/uv/tools/<tool-name>/`.
- Entry-point scripts are placed in `~/.local/bin/` (or `%USERPROFILE%\.local\bin` on Windows).
- The shim script execs the tool's venv Python directly — no PATH manipulation.

## Scripts with PEP 723

Standalone single-file scripts with embedded dep metadata. uv automatically materialises an ephemeral venv per script, cached by script hash.

### Inline-metadata header

```bash
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "httpx>=0.27",
#     "rich",
# ]
#
# [tool.uv]
# exclude-newer = "2024-01-01T00:00:00Z"
# ///
```

Notes:
- The opening `# /// script` line is required.
- The closing `# ///` is required.
- Each metadata line starts with `# ` (hash + space).
- Body is TOML.
- `requires-python` and `dependencies` are required.

### Run

```bash
uv run script.py                                     # auto-detects header
uv run --script script.py
uv run --script script.py arg1 arg2 --flag

# Override Python at run time
uv run --python 3.13 script.py

# Add ad-hoc deps without editing the file
uv run --with polars script.py
```

### Manage from CLI

```bash
uv add --script script.py polars
uv add --script script.py 'httpx>=0.27' rich
uv remove --script script.py rich
uv lock --script script.py                           # adds # requires-lock block (uv 0.5+)
uv sync --script script.py                           # build the script's venv
uv tree --script script.py
```

### Shebang

Make a uv script executable:

```bash
#!/usr/bin/env -S uv run --script
# /// script
# dependencies = ["rich"]
# ///
from rich import print
print("hi")
```

```bash
chmod +x script.py
./script.py
```

### Script venv cache

- Stored at `~/.cache/uv/environments-v0/<hash>/` keyed by the script's metadata hash.
- Reused across runs until metadata changes.
- Garbage-collected by `uv cache prune`.

## Cache

```bash
uv cache dir                                         # show cache root
uv cache clean                                       # delete the entire cache
uv cache clean requests                              # delete only this pkg's entries
uv cache prune                                       # delete only stale/unreferenced entries (recommended)
uv cache prune --ci                                  # tighter pruning meant for CI volume

# Bypass cache for a single command
uv pip install --no-cache requests
uv sync --no-cache
uv lock --refresh                                    # ignore cache, refetch metadata
uv lock --refresh-package requests                   # refresh one dep's metadata

# Permanent: env var or flag
export UV_NO_CACHE=1
uv pip install --refresh requests                    # bypass for one invocation
```

### Cache structure

```bash
~/.cache/uv/
├── archive-v0/         # extracted wheel trees, install-ready
├── built-wheels-v0/    # wheels uv built locally from sdists
├── environments-v0/    # ephemeral venvs (PEP 723 scripts, uvx)
├── git-v0/             # cloned git repos
├── interpreter-v0/     # Python interpreter metadata
├── simple-v0/          # PyPI Simple API metadata cache
└── wheels-v0/          # pre-built .whl files from registries
```

### Link modes (cache → venv copy strategy)

```bash
uv venv --link-mode hardlink                         # default when same fs
uv venv --link-mode copy                             # full copy
uv venv --link-mode symlink                          # POSIX only
uv venv --link-mode clone                            # APFS / btrfs / xfs reflink (zero-cost copy)

# Or set permanently
export UV_LINK_MODE=clone
```

## Index Configuration

### One-shot

```bash
uv pip install --index-url https://my.corp/simple my-pkg
uv pip install --extra-index-url https://my.corp/simple my-pkg
uv pip install --find-links ./wheels --no-index pkg

# uv 0.5+ alternative
uv pip install --default-index https://my.corp/simple my-pkg
uv pip install --index https://my.corp/simple my-pkg
```

### Named indexes via [[tool.uv.index]]

```bash
[[tool.uv.index]]
name = "pypi"
url = "https://pypi.org/simple"
default = true                                       # this is the default (highest priority unless explicit)

[[tool.uv.index]]
name = "my-corp"
url = "https://my.corp/simple"
explicit = true                                      # only use when [tool.uv.sources] references "my-corp"

[[tool.uv.index]]
name = "pytorch-cu121"
url = "https://download.pytorch.org/whl/cu121"
explicit = true

[tool.uv.sources]
my-private = { index = "my-corp" }
torch = { index = "pytorch-cu121" }
```

### Index strategies

```bash
uv pip install --index-strategy first-index pkg        # default — first index that has the pkg wins
uv pip install --index-strategy unsafe-best-match pkg  # pick best version across indexes
uv pip install --index-strategy unsafe-first-match pkg # first match wins (any version)
uv pip install --index-strategy unsafe-any-match pkg   # any match
```

`first-index` is the secure default; switching may expose dependency-confusion attacks.

### Authentication

```bash
# Per-index env var pattern (uv 0.5+)
export UV_INDEX_MY_CORP_USERNAME=stevie
export UV_INDEX_MY_CORP_PASSWORD=$TOKEN
# Pattern: UV_INDEX_<NAME>_USERNAME / UV_INDEX_<NAME>_PASSWORD where <NAME> is uppercased

# Or embed in URL (avoid for shared configs)
uv pip install --index-url https://stevie:$TOKEN@my.corp/simple my-pkg

# System keyring
uv pip install --keyring-provider subprocess pkg
# uv shells out to `keyring get <host> <user>` to fetch the password

# In pyproject.toml
[[tool.uv.index]]
name = "my-corp"
url = "https://my.corp/simple"
authenticate = "always"                              # always send auth header
# username/password come from UV_INDEX_MY_CORP_USERNAME/PASSWORD
```

### .netrc

```bash
# ~/.netrc
machine my.corp
login stevie
password TOKEN

# uv reads ~/.netrc automatically when no other auth is set
chmod 600 ~/.netrc
```

## Environment Variables

### Index / network

```bash
UV_INDEX_URL=https://pypi.org/simple
UV_DEFAULT_INDEX=https://pypi.org/simple             # uv 0.5+
UV_EXTRA_INDEX_URL=https://my.corp/simple,https://my.corp/extra
UV_INDEX=name=https://my.corp/simple                 # named index
UV_FIND_LINKS=https://my.corp/wheels
UV_NO_INDEX=1                                        # disable PyPI lookup
UV_INDEX_STRATEGY=first-index                        # or unsafe-best-match
UV_KEYRING_PROVIDER=subprocess
UV_INDEX_<NAME>_USERNAME=stevie
UV_INDEX_<NAME>_PASSWORD=secret
UV_HTTP_TIMEOUT=30                                   # seconds
UV_NATIVE_TLS=1                                      # use OS TLS stack instead of rustls
UV_OFFLINE=1                                         # cache-only, fail on network access
UV_NO_CACHE=1                                        # bypass cache
UV_CACHE_DIR=/tmp/uv-cache
```

### Python toolchain

```bash
UV_PYTHON=3.12                                       # default Python for cwd
UV_PYTHON=python3.12                                 # any Python with that name
UV_PYTHON=/usr/bin/python3.12                        # absolute path
UV_PYTHON_INSTALL_DIR=$HOME/.local/share/uv/python   # override managed Python location
UV_PYTHON_INSTALL_MIRROR=https://my.org/python-build-standalone
UV_PYTHON_PREFERENCE=managed                         # managed | system | only-managed | only-system
UV_PYTHON_DOWNLOADS=automatic                        # automatic | manual | never
```

### Project / venv

```bash
UV_PROJECT_ENVIRONMENT=.venv-prod                    # custom venv location for project
UV_LINK_MODE=clone                                   # clone | copy | hardlink | symlink
UV_RESOLUTION=highest                                # highest | lowest | lowest-direct
UV_PRERELEASE=disallow                               # disallow | allow | if-necessary | ...
UV_FROZEN=1                                          # default `--frozen` for everything
UV_LOCKED=1                                          # default `--locked` for everything
UV_NO_SYNC=1                                         # default `--no-sync` for `uv run`
UV_NO_PROGRESS=1                                     # disable progress bars
UV_NO_BUILD_ISOLATION=1                              # disable build isolation globally
UV_BUILD_BACKEND=hatch                               # default backend for `uv init`
UV_VCS=git                                           # default VCS for `uv init`
UV_NO_EDITABLE=1                                     # never install editably
UV_PUBLISH_TOKEN=$TOKEN                              # for `uv publish`
UV_PUBLISH_USERNAME=stevie
UV_PUBLISH_PASSWORD=$TOKEN
UV_PUBLISH_URL=https://upload.pypi.org/legacy/
```

### Logging / output

```bash
UV_VERBOSE=1                                         # equivalent to -v
RUST_LOG=uv=debug                                    # trace-level Rust logging
RUST_BACKTRACE=1                                     # backtraces on panics
NO_COLOR=1                                           # disable ANSI colour
UV_LOG_FILE=/tmp/uv.log                              # write all logs here
```

## Build Backends

uv is build-backend agnostic. The `[build-system]` table chooses one. Common pairings:

### uv_build (uv 0.5+)

```bash
[build-system]
requires = ["uv_build>=0.5.0,<0.6"]
build-backend = "uv_build"
```

- Lightweight backend produced by uv itself.
- Fast pure-Python builds; no support for compiled extensions yet.
- Reads everything from `[project]` + `[tool.uv]`.

### Hatchling (modern, default for `uv init`)

```bash
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.version]
path = "src/my_project/__about__.py"

[tool.hatch.build.targets.wheel]
packages = ["src/my_project"]
```

### Setuptools (the old reliable)

```bash
[build-system]
requires = ["setuptools>=64", "setuptools_scm>=8"]
build-backend = "setuptools.build_meta"

[tool.setuptools.packages.find]
where = ["src"]

[tool.setuptools_scm]                                # version from VCS tag
```

### Flit-core (small pure-Python projects)

```bash
[build-system]
requires = ["flit_core>=3.2,<4"]
build-backend = "flit_core.buildapi"

[tool.flit.module]
name = "my_project"
```

### Poetry-core (poetry-style projects without poetry CLI)

```bash
[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"
```

### Maturin (Rust extensions)

```bash
[build-system]
requires = ["maturin>=1.5,<2"]
build-backend = "maturin"

[tool.maturin]
features = ["pyo3/extension-module"]
```

### Scikit-build-core (CMake/C++)

```bash
[build-system]
requires = ["scikit-build-core>=0.10"]
build-backend = "scikit_build_core.build"
```

### `uv build`

Wraps the chosen backend to produce sdist + wheel.

```bash
uv build                                             # build cwd's project
uv build --sdist                                     # only sdist
uv build --wheel                                     # only wheel
uv build --out-dir dist/                             # default
uv build --package my-pkg                            # workspace member
uv build --all-packages                              # every workspace member
uv build --no-sources                                # ignore [tool.uv.sources] (publish-ready build)
uv build --build-constraint constraints.txt
uv build /path/to/sdist.tar.gz                       # rebuild a wheel from sdist
```

### `uv publish`

```bash
uv publish dist/*.whl dist/*.tar.gz                  # upload to PyPI
uv publish --token $PYPI_TOKEN dist/*
uv publish --publish-url https://test.pypi.org/legacy/ dist/*
uv publish --username __token__ --password $PYPI_TOKEN dist/*
uv publish --check-url https://pypi.org/simple/ dist/*  # skip if already uploaded
uv publish --trusted-publishing automatic dist/*     # GitHub Actions OIDC
```

## Migration

### pip + venv → uv

```bash
# Old
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt

# New, drop-in
uv venv
. .venv/bin/activate
uv pip install -r requirements.txt

# Or stay declarative — convert requirements.txt to pyproject.toml
uv init --bare
# Then for each line in requirements.txt:
uv add $(grep -v '^#' requirements.txt)
```

### pipx → uv tool

```bash
# Old
pipx install ruff
pipx run black .
pipx list

# New
uv tool install ruff
uvx black .
uv tool list

# Bulk migrate
pipx list --short | awk '{print $1}' | xargs -n1 uv tool install
```

### poetry → uv

```bash
# pyproject.toml conversion: most fields under [tool.poetry] move to [project]
# poetry deps → [project].dependencies
# poetry dev-deps → [dependency-groups].dev
# poetry.lock is replaced by uv.lock (regenerate)

# Manual steps:
# 1. Move deps from [tool.poetry.dependencies] to [project].dependencies (PEP 508 syntax)
#    e.g.  requests = "^2.31"  →  "requests>=2.31,<3"
# 2. Move dev deps from [tool.poetry.group.dev.dependencies] to [dependency-groups].dev
# 3. Update [build-system]:
#       requires = ["poetry-core"]   # if you still want poetry-core as backend
#       build-backend = "poetry.core.masonry.api"
# 4. Delete poetry.lock
# 5. Run `uv lock && uv sync`

# Helper tool: pdm-backend ships a converter; or use poetry-to-uv community scripts.
```

### pip-tools → uv

```bash
# Old
pip-compile requirements.in -o requirements.txt
pip-sync requirements.txt

# New
uv pip compile requirements.in -o requirements.txt
uv pip sync requirements.txt

# Or skip compile altogether — use uv project mode + uv.lock
```

### conda → uv

```bash
# uv covers Python-only stacks. For non-Python deps (CUDA, OpenSSL, BLAS, R, Java),
# you still need conda OR system packages OR pre-built wheels carrying the deps.
#
# Migration:
# 1. List Python packages in your env:
conda list --explicit > conda-explicit.txt
# 2. Filter to Python-only deps; ignore conda system packages.
# 3. Add to pyproject.toml:
uv add numpy pandas scikit-learn
# 4. For CUDA: pin via [tool.uv.sources] to PyTorch's CUDA index
[tool.uv.sources]
torch = { index = "pytorch-cu121" }
[[tool.uv.index]]
name = "pytorch-cu121"
url = "https://download.pytorch.org/whl/cu121"
explicit = true
```

### pyenv → uv python

```bash
# Old
pyenv install 3.12.7
pyenv local 3.12.7

# New
uv python install 3.12.7
uv python pin 3.12.7

# Migration of existing .python-version: uv reads pyenv's .python-version directly.
# Just delete pyenv from your PATH, run `uv python install <ver>` for each version
# you used, and uv takes over.
```

## Common Errors

### "Because <pkg> requires <pkg2>>=X and you require <pkg2><Y, we can conclude that the requirements are unsatisfiable."

**Cause:** PubGrub found a real conflict — direct deps + transitive deps disagree.

**Fix:**
```bash
uv tree --invert pkg2                                # find who pulls in pkg2 with what range
uv add 'pkg2>=X'                                     # widen your direct constraint
# Or override:
[tool.uv]
override-dependencies = ["pkg2>=X,<Z"]               # force a specific range
```

### "error: No solution found when resolving dependencies"

**Cause:** Python version constraints conflict, or a dep has no compatible release.

**Fix:**
```bash
uv lock -v                                           # verbose explanation
uv tree --resolution lowest                          # see the conflicting bounds
# Often: requires-python is too narrow; widen it.
[project]
requires-python = ">=3.10"                           # was ">=3.12", widen
```

### "Failed to build: <pkg>" — build-isolation issue

**Cause:** A package needs build-time deps already installed in your env (numpy + Cython for some sci-py builds, or torch's custom setup).

**Fix:**
```bash
[tool.uv]
no-build-isolation-package = ["torch"]
# Then ensure the build-time deps are available:
uv add --dev numpy cython
uv sync
```

### "error: HTTP transport error: ... 401 Unauthorized" (private index auth)

**Cause:** uv hit a private index without credentials.

**Fix:**
```bash
# Set per-index auth
export UV_INDEX_MY_CORP_USERNAME=stevie
export UV_INDEX_MY_CORP_PASSWORD=$TOKEN

# Or via keyring
uv pip install --keyring-provider subprocess my-pkg

# Or .netrc
chmod 600 ~/.netrc
```

### "Project virtual environment directory already exists and is not a valid Python environment"

**Cause:** `.venv` exists but is broken (e.g., the Python interpreter it points to has been deleted).

**Fix:**
```bash
rm -rf .venv
uv sync
```

### "Lockfile is outdated, please run `uv lock`"

**Cause:** pyproject.toml changed since `uv.lock` was generated.

**Fix:**
```bash
uv lock                                              # regenerate
uv sync
# In CI: this means someone forgot to commit a fresh lockfile — prefer `uv sync --frozen`
# in CI so this surfaces as a hard error rather than auto-relock.
```

### "The Python interpreter at .venv/bin/python does not exist"

**Cause:** Managed Python was uninstalled, or `.venv` was created on another machine with a different Python path.

**Fix:**
```bash
rm -rf .venv
uv python install                                    # ensure pinned Python is installed
uv sync
```

### "error: Failed to inspect Python interpreter from cached environment"

**Cause:** Cached interpreter metadata is stale.

**Fix:**
```bash
uv cache clean
uv sync
```

### "warning: requirements.txt was generated with an older uv version"

**Cause:** The compiled requirements.txt header indicates it was made by an older uv.

**Fix:**
```bash
uv self update
uv pip compile -o requirements.txt requirements.in
```

### "uv.lock is outdated"

**Cause:** Same as the "Lockfile is outdated" error in different wording — pyproject.toml drift.

**Fix:**
```bash
uv lock
git add uv.lock
git commit -m "Refresh uv.lock"
```

### "error: Failed to parse `pyproject.toml`"

**Cause:** Syntax error in TOML or missing required PEP 621 fields (`name`, `version`).

**Fix:**
```bash
uv lock --offline -v 2>&1 | head -30                 # uv prints the offending line
python -c "import tomllib; tomllib.load(open('pyproject.toml','rb'))"
```

### "Distribution not found at: <url>"

**Cause:** A wheel referenced by direct URL or git rev no longer exists.

**Fix:**
```bash
uv lock --upgrade-package mypkg                      # let resolver pick a new source
# Or update the source in pyproject.toml
```

### "warning: The package `<pkg>` was already installed from a different source"

**Cause:** Mixing `uv pip install` (no source recorded) with project-mode `uv sync` that has a `[tool.uv.sources]` entry for the same package.

**Fix:**
```bash
rm -rf .venv
uv sync
```

### "error: distribution requires a different `requires-python`"

**Cause:** A dep needs a Python version your project doesn't allow.

**Fix:**
```bash
[project]
requires-python = ">=3.10"                           # widen if practical
# Or pin the dep to an older version:
uv add 'old-pkg<2.0'
```

### "Could not find a distribution that satisfies the requirement"

**Cause:** Wrong index, typo, no wheel for current platform/Python combo.

**Fix:**
```bash
uv pip install -v requests                           # verbose to see indexes tried
uv pip install --index-url https://pypi.org/simple requests
```

### "warning: `pyproject.toml` does not declare `requires-python`"

**Cause:** Missing field; uv warns because it can't decide which Python to use.

**Fix:**
```bash
[project]
requires-python = ">=3.12"
```

### "Hash mismatch for distribution"

**Cause:** A wheel changed at the index between lock and download. Could be a republished release, a poisoned index, or a corrupted cache.

**Fix:**
```bash
uv cache clean
uv lock --refresh                                    # refetch metadata
# If hash truly changed — investigate the upstream pkg release.
```

## Common Gotchas

### 1) Forgetting `uv sync` after `uv add --no-sync`

```bash
# Broken
uv add --no-sync requests
python -c "import requests"
# ModuleNotFoundError: No module named 'requests'

# Fixed
uv add requests                                      # auto-syncs
# Or
uv add --no-sync requests
uv sync                                              # explicit sync
```

### 2) CI without `--frozen` — lock churn

```bash
# Broken (CI surface area drifts)
- run: uv sync                                       # may regenerate lock if pyproject changed

# Fixed (lock is authoritative; CI fails loud on stale lock)
- run: uv sync --frozen                              # error if lock is stale
- run: uv run --frozen pytest
```

### 3) Mixing `uv pip` with project mode

```bash
# Broken — installs requests, but next `uv sync` removes it
. .venv/bin/activate
uv pip install requests
uv sync                                              # requests now uninstalled

# Fixed — use project mode
uv add requests
uv sync
```

### 4) Build isolation breaks native extensions needing pre-existing deps

```bash
# Broken — torch's custom build wants numpy already in the env
uv add torch
# Failed to build torch: numpy not found

# Fixed
[tool.uv]
no-build-isolation-package = ["torch"]
[dependency-groups]
build-deps = ["numpy", "cython"]
[tool.uv]
default-groups = ["build-deps"]
```

### 5) `--resolution lowest` in CI as a compat check

```bash
# Lower-bound CI matrix
- run: uv lock --resolution lowest
- run: uv sync --frozen
- run: uv run --frozen pytest
# Catches: "I declared 'requests>=2.20' but actually I need a 2.31+ feature"
```

### 6) `UV_PYTHON` set globally pinning the wrong version

```bash
# Broken — UV_PYTHON=3.10 in shell rc, but project requires 3.12
uv sync
# error: Python 3.10 doesn't satisfy requires-python >=3.12

# Fixed — let .python-version drive
unset UV_PYTHON
uv python pin 3.12
uv sync
```

### 7) `.python-version` vs `requires-python` contradiction

```bash
# Broken
# .python-version → 3.10
# pyproject → requires-python = ">=3.12"
uv sync
# Error: pinned Python 3.10 violates requires-python

# Fixed
uv python pin 3.12
# Or relax requires-python:
[project]
requires-python = ">=3.10"
```

### 8) Forgetting to commit `uv.lock`

```bash
# Broken — .gitignore had uv.lock
echo "uv.lock" >> .gitignore                         # NEVER do this for apps
git push                                             # CI sees no lock, regenerates → drift

# Fixed
# In .gitignore, exclude:
#   .venv/
#   __pycache__/
# Include:
#   pyproject.toml
#   uv.lock
#   .python-version
git add uv.lock
git commit -m "Lock"
```

### 9) Using `uv add ../foo` without `[tool.uv.sources]` awareness

```bash
# What you typed
uv add ../foo
# What uv writes:
# [project].dependencies → "foo"
# [tool.uv.sources].foo → { path = "../foo" }
# Now publishing this project breaks because external users have no `../foo`.

# Fixed — for a publishable package, use --raw-sources or remove the source override:
uv add --raw-sources foo
# Or build with --no-sources:
uv build --no-sources
```

### 10) `uvx tool@latest` not bypassing cache

```bash
# Surprising — uvx caches the resolution
uvx ruff check .                                     # may use a stale ruff
uvx --refresh ruff check .                           # force refresh
uvx ruff@latest check .                              # explicit "latest"
```

### 11) Workspace member with the same name as a registry package

```bash
# Broken — workspace member "requests" shadows PyPI's requests
# uv resolves to local; transitive deps get confused.

# Fixed — never name workspace members the same as published packages.
```

### 12) Editable installs surviving `--no-editable`

```bash
# uv sync --no-editable still treats workspace members as editable by default.
# To disable editability for workspace members:
[tool.uv.sources]
my-utils = { workspace = true, editable = false }
```

### 13) Hardlink mode breaking when cache and venv are on different filesystems

```bash
# Symptom: silently slow installs (hardlink falls back to copy)
# Or: "Failed to hardlink files; falling back to full copy"

# Fixed — use clone mode on APFS/btrfs/xfs
export UV_LINK_MODE=clone
# Or move cache onto the same fs as your project
export UV_CACHE_DIR=/path/on/same/fs
```

### 14) Missing `--all-extras` in CI hides bugs

```bash
# Broken — CI installs default deps only
- run: uv sync --frozen

# Fixed — match dev exactly
- run: uv sync --frozen --all-extras --all-groups
```

### 15) PEP 723 script header not detected — wrong comment style

```bash
# Broken
'''
# /// script
# dependencies = ["rich"]
# ///
'''
# uv ignores this — must be `#`-prefixed line comments, not a docstring.

# Fixed
# /// script
# dependencies = ["rich"]
# ///
```

### 16) Forgetting `uv pip` vs `uv tool`

```bash
# Broken — pollutes project venv with linters
uv add ruff black mypy                               # now part of every install

# Fixed — install once, globally
uv tool install ruff
uv tool install black
uv tool install mypy
```

## Idioms

### CI install (the canonical recipe)

```bash
- name: Install uv
  run: curl -LsSf https://astral.sh/uv/install.sh | sh
- name: Sync
  run: uv sync --frozen --all-extras --all-groups
- name: Test
  run: uv run --frozen pytest
- name: Lint
  run: uv run --frozen ruff check .
```

### Reproducible build (Docker)

```bash
FROM python:3.12-slim AS base
COPY --from=ghcr.io/astral-sh/uv:0.5 /uv /uvx /bin/

WORKDIR /app

# Layer 1: copy lock + manifest only (changes rarely)
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-install-project --no-dev

# Layer 2: copy source (changes often)
COPY src/ ./src/
RUN uv sync --frozen --no-dev

ENV PATH="/app/.venv/bin:$PATH"
CMD ["python", "-m", "myapp"]
```

### Ephemeral tool one-liners

```bash
uvx ruff check .
uvx black .
uvx mypy src/
uvx --from httpie http get https://example.com
uvx --from build pyproject-build
```

### Per-PR test matrix with low-bound check

```bash
strategy:
  matrix:
    resolution: [highest, lowest-direct]
    python-version: ["3.10", "3.11", "3.12"]
steps:
  - uses: astral-sh/setup-uv@v3
  - run: uv lock --resolution ${{ matrix.resolution }}
  - run: uv sync --frozen --python ${{ matrix.python-version }}
  - run: uv run --frozen pytest
```

### Self-updating script for ops

```bash
#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = ["typer", "rich", "httpx"]
# ///
import typer
from rich import print
import httpx

app = typer.Typer()

@app.command()
def status(url: str):
    r = httpx.get(url)
    print({"status": r.status_code, "size": len(r.content)})

if __name__ == "__main__":
    app()
```

### Vendoring for air-gapped install

```bash
# On a connected machine
uv pip compile pyproject.toml -o requirements.txt --generate-hashes
uv pip download -r requirements.txt -d ./vendor

# Tar + ship vendor/ + uv binary
tar czf vendor.tgz vendor/ requirements.txt

# On the air-gapped machine
tar xzf vendor.tgz
uv venv
uv pip install --no-index --find-links ./vendor -r requirements.txt
```

### Pre-commit hook (bytecode + lock check)

```bash
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/astral-sh/uv-pre-commit
    rev: 0.5.13
    hooks:
      - id: uv-lock                                  # ensure uv.lock is up-to-date
      - id: uv-export                                # auto-regen requirements.txt
```

### Quick experiment without polluting any project

```bash
# A throwaway venv with a few deps — gone after the shell exits
mkdir -p /tmp/scratch && cd /tmp/scratch
uv init --bare
uv add httpx polars
uv run python                                        # poke around
cd ..
rm -rf /tmp/scratch
```

### "Lock now, install later" CI split

```bash
# Lock job: produce the lockfile artefact
- run: uv lock
- uses: actions/upload-artifact@v4
  with:
    name: lock
    path: uv.lock

# Install jobs (matrixed): consume the lockfile
- uses: actions/download-artifact@v4
  with:
    name: lock
- run: uv sync --frozen
```

### Mixing uv tool and uv project

```bash
# Linters as global tools (don't pollute project venv)
uv tool install ruff
uv tool install mypy
uv tool install pre-commit

# Project-specific test deps stay in the project
uv add --group test pytest pytest-cov
uv run --group test pytest                           # uses project venv
```

## See Also

- python — Python language reference and stdlib idioms
- polyglot — multi-language ecosystem comparisons
- cargo — Cargo-style workflows, the inspiration for uv's resolver and workspaces
- pnpm — pnpm content-addressable store, conceptually similar to uv's cache
- gomod — Go modules, another deterministic resolver / lockfile design

## References

- Official docs — https://docs.astral.sh/uv/
- GitHub repository — https://github.com/astral-sh/uv
- Astral blog announcement — https://astral.sh/blog/uv
- python-build-standalone — https://github.com/astral-sh/python-build-standalone
- PEP 517 — Build-system independence — https://peps.python.org/pep-0517/
- PEP 518 — pyproject.toml [build-system] — https://peps.python.org/pep-0518/
- PEP 621 — Project metadata in pyproject.toml — https://peps.python.org/pep-0621/
- PEP 660 — Editable installs for pyproject-based builds — https://peps.python.org/pep-0660/
- PEP 723 — Inline script metadata — https://peps.python.org/pep-0723/
- PEP 735 — Dependency groups — https://peps.python.org/pep-0735/
- PEP 508 — Dependency specifiers — https://peps.python.org/pep-0508/
- PEP 440 — Version identification — https://peps.python.org/pep-0440/
- PEP 639 — License expressions in pyproject.toml — https://peps.python.org/pep-0639/
- PEP 668 — Marking Python base environments as "externally managed" — https://peps.python.org/pep-0668/
- PubGrub algorithm — https://nex3.medium.com/pubgrub-2fb6470504f
- uv pre-commit — https://github.com/astral-sh/uv-pre-commit
- setup-uv GitHub Action — https://github.com/astral-sh/setup-uv
- Astral Discord — https://discord.gg/astral-sh
