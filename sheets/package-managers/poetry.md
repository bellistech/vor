# poetry

Python dependency management, packaging, and publishing — pyproject.toml-driven, deterministic resolver, integrated venv automation, single-tool replacement for pip+venv+setuptools+twine.

## Setup

```bash
# Recommended: pipx (isolated, upgradable, never collides with project venvs)
pipx install poetry
pipx upgrade poetry
pipx uninstall poetry

# Official installer (curl) — installs to ~/.local/share/pypoetry
curl -sSL https://install.python-poetry.org | python3 -
curl -sSL https://install.python-poetry.org | python3 - --version 1.8.4
curl -sSL https://install.python-poetry.org | python3 - --preview
curl -sSL https://install.python-poetry.org | python3 - --uninstall
curl -sSL https://install.python-poetry.org | POETRY_HOME=/opt/poetry python3 -

# Older URL (deprecated — install-poetry.py replaced get-poetry.py in 1.2)
curl -sSL https://install.python-poetry.org -o install-poetry.py
python3 install-poetry.py --version 1.8.4
python3 install-poetry.py --uninstall

# Homebrew (macOS / Linuxbrew) — pinned by formula maintainer, may lag upstream
brew install poetry
brew upgrade poetry
brew uninstall poetry

# Add to PATH after curl install
export PATH="$HOME/.local/bin:$PATH"
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc

# Verify install
poetry --version
poetry --help
poetry about

# Self management (1.2+) — manage Poetry's own venv, plugins, updates
poetry self update
poetry self update 1.8.4
poetry self update --preview
poetry self show
poetry self show plugins
poetry self show --tree
poetry self add poetry-plugin-export
poetry self add poetry-plugin-bundle
poetry self add poetry-plugin-shell
poetry self add poetry-dynamic-versioning
poetry self remove poetry-plugin-export
poetry self lock

# Shell completions (1.2+)
poetry completions bash > /etc/bash_completion.d/poetry.bash-completion
poetry completions zsh > ~/.zfunc/_poetry
poetry completions fish > ~/.config/fish/completions/poetry.fish

# Diagnose install
which poetry
poetry --version
poetry config --list
ls $(poetry env info --path)
```

## Why Poetry

```bash
# vs pip + requirements.txt
#   pip:     manual venv, pinned reqs ≠ resolved tree, no lockfile, install != reproduce
#   poetry:  one tool — venv + resolver + lockfile + build + publish, true reproducibility

# vs pipenv
#   pipenv:  Pipfile/Pipfile.lock, slow resolver (deprecated 1-year hiatus), no build/publish
#   poetry: faster resolver, builds wheels/sdists, publishes to PyPI, plugin ecosystem

# vs uv (Astral, 2024+)
#   uv:     Rust-fast (10–100x), pip-compatible, single static binary, native pyproject.toml
#   poetry: mature, plugin ecosystem, builds + publishes natively, PEP 517 build backend
#   Mix:    uv lock + uv pip install for speed, poetry build/publish — or stick with one

# vs setuptools + twine + pip-tools
#   classic: 4 tools (setup.py, pip-tools for lock, twine for publish, venv manually)
#   poetry: one config, one CLI, one lockfile

# pyproject.toml layouts you will see in 2025
#   PEP 621 (standard):     [project] name = "...", dependencies = [...]   (uv, hatch, flit)
#   Poetry (pre-2.0):       [tool.poetry] name = "...", [tool.poetry.dependencies]
#   Poetry 2.0+:            BOTH supported — [project] is preferred, [tool.poetry] for poetry-only knobs
# Until you migrate to Poetry 2.x, you will write [tool.poetry] (this sheet).

# Deterministic resolver
#   * SAT-style backtracking — finds compatible versions across full graph
#   * Records resolved versions + content hashes in poetry.lock
#   * `poetry install` from lock = byte-identical tree everywhere

# Built-in build/publish
#   poetry build     -> dist/foo-1.0-py3-none-any.whl + foo-1.0.tar.gz
#   poetry publish   -> twine-equivalent, with TOML config for repos + tokens

# Venv automation
#   * First `poetry install` creates venv automatically (~/.cache/pypoetry/virtualenvs/...)
#   * `poetry run X`, `poetry env activate` — never need `source venv/bin/activate` ritual
#   * `virtualenvs.in-project = true` puts .venv/ next to pyproject.toml (IDE-friendly)
```

## Project Init

```bash
# Create new package skeleton (flat layout)
poetry new project-name
# Generates:
#   project-name/
#     pyproject.toml
#     README.md
#     project_name/__init__.py
#     tests/__init__.py

# src-layout (recommended for libraries — prevents importing from cwd)
poetry new --src project-name
# Generates:
#   project-name/
#     pyproject.toml
#     README.md
#     src/project_name/__init__.py
#     tests/__init__.py

# Override import-package name (when distribution name != module name)
poetry new my-cool-lib --name mycoollib

# Existing directory — interactive prompt
cd existing-project
poetry init
# Prompts: name, version, description, author, license, python compat, deps, dev-deps

# Non-interactive — perfect for scripts and templates
poetry init --no-interaction \
  --name my-app \
  --description "Example app" \
  --author "Jane Doe <jane@example.com>" \
  --license MIT \
  --python "^3.11" \
  --dependency "requests:^2.31" \
  --dependency "pydantic:^2.0" \
  --dev-dependency "pytest:^8.0" \
  --dev-dependency "ruff:^0.5"

# Just create pyproject.toml with sane defaults
poetry init -n

# After init: install + create venv
poetry install
```

## pyproject.toml — [tool.poetry]

```bash
cat > pyproject.toml <<'TOML'
[tool.poetry]
name = "my-app"                       # distribution name (PyPI)
version = "0.1.0"                     # semver — bumped via `poetry version`
description = "An example Poetry project"
authors = ["Jane Doe <jane@example.com>", "John Smith <john@example.com>"]
maintainers = ["Jane Doe <jane@example.com>"]
license = "MIT"                       # SPDX identifier (also accepts "Proprietary", file = "LICENSE")
readme = "README.md"                  # or ["README.md", "CHANGELOG.md"]
homepage = "https://example.com/my-app"
repository = "https://github.com/example/my-app"
documentation = "https://my-app.readthedocs.io"

keywords = ["example", "demo", "cli"]
classifiers = [                       # PyPI Trove classifiers
  "Development Status :: 3 - Alpha",
  "Intended Audience :: Developers",
  "License :: OSI Approved :: MIT License",
  "Operating System :: OS Independent",
  "Programming Language :: Python :: 3",
  "Programming Language :: Python :: 3.11",
  "Programming Language :: Python :: 3.12",
  "Topic :: Software Development :: Libraries",
]

# Package layout — what gets included in the wheel
packages = [
  { include = "my_app" },                                # flat layout
  { include = "my_app", from = "src" },                  # src layout
  { include = "my_app/**/*.py", format = "wheel" },      # wheel-only
  { include = "tests", format = "sdist" },               # sdist-only
  { include = "my_app/extra.py", to = "renamed_extra" }, # rename inside wheel
]

# Extra files to include / exclude (glob patterns)
include = [
  { path = "tests", format = ["sdist"] },
  "CHANGELOG.md",
  "my_app/data/*.json",
]
exclude = [
  "my_app/internal_only.py",
  "**/*.pyc",
]

# Console scripts (entry-points) — install creates `my-cli` in PATH
[tool.poetry.scripts]
my-cli   = "my_app.cli:main"          # `my-cli` -> my_app.cli.main()
my-tool  = "my_app.tool:run"
gui-app  = { reference = "my_app.gui:start", type = "console", extras = ["gui"] }

# GUI scripts (Windows-only — no console window)
[tool.poetry.gui_scripts]
my-gui = "my_app.gui:main"

# Plugins — register entry points other tools discover
[tool.poetry.plugins."pytest11"]
my-pytest-plugin = "my_app.pytest_plugin"

[tool.poetry.plugins."console_scripts"]
# rarely used — prefer [tool.poetry.scripts] above
TOML
```

## pyproject.toml — [tool.poetry.dependencies]

```bash
cat >> pyproject.toml <<'TOML'
[tool.poetry.dependencies]
python = "^3.10"                      # Poetry-specific — equivalent to PEP 621 requires-python = ">=3.10,<4.0"

# Caret (^) — semver-aware: ^1.2.3 = >=1.2.3,<2.0.0; ^0.2.3 = >=0.2.3,<0.3.0; ^0.0.3 = >=0.0.3,<0.0.4
requests   = "^2.31"
fastapi    = "^0.110"

# Tilde (~) — patch-level: ~1.2.3 = >=1.2.3,<1.3.0
typing-extensions = "~4.7"

# Wildcard (*) — major-level: 1.* = >=1.0,<2.0
pydantic = "1.*"

# Exact pin
ujson = "5.10.0"
"==5.10.0" # equivalent inline form: ujson = "==5.10.0"

# Range
sqlalchemy = ">=2.0,<2.1"
numpy      = ">=1.24,!=1.25.0,<2.0"

# Any version (avoid in production — defeats lockfile purpose)
some-pkg = "*"

# Inline table — full dep spec in one line
boto3 = { version = "^1.34", optional = true, extras = ["crt"] }
psycopg2 = { version = "^2.9", optional = true, markers = "platform_system == 'Linux'" }

# Multiple constraints — different versions per python
[tool.poetry.dependencies.dataclasses]
version = "^0.7"
markers = "python_version < '3.7'"

# Multiple-constraint dependency (advanced — array of tables)
[[tool.poetry.dependencies.foo]]
version = "^1.0"
markers = "python_version < '3.10'"

[[tool.poetry.dependencies.foo]]
version = "^2.0"
markers = "python_version >= '3.10'"

# Git dependency — branch/tag/rev
internal-lib = { git = "https://github.com/example/internal-lib.git", branch = "main" }
auth-helper  = { git = "git@github.com:example/auth-helper.git",      tag    = "v1.2.3" }
proto-defs   = { git = "https://github.com/example/proto-defs.git",   rev    = "abc123def" }
sub-pkg      = { git = "https://github.com/example/mono.git", subdirectory = "packages/sub-pkg" }

# Path dependency (local) — handy for monorepos
shared-utils = { path = "../shared-utils", develop = true }   # develop = pip install -e
vendored-lib = { path = "./vendor/some-lib-1.0.tar.gz" }      # archive

# URL dependency (direct download)
weird-pkg = { url = "https://example.com/weird-pkg-1.0.tar.gz" }

# Markers — PEP 508 environment markers (any Python conditional)
pytest-asyncio = { version = "^0.23", markers = "python_version >= '3.10'" }
pywin32        = { version = "*",     markers = "sys_platform == 'win32'" }
TOML
```

## pyproject.toml — Dependency Groups (1.2+)

```bash
cat >> pyproject.toml <<'TOML'
# Modern grouping — replaces `--dev`. Add as many groups as you like.
[tool.poetry.group.dev.dependencies]
pytest         = "^8.0"
pytest-cov     = "^5.0"
ruff           = "^0.5"
mypy           = "^1.10"
ipython        = "^8.20"

[tool.poetry.group.test.dependencies]
pytest-xdist   = "^3.5"
pytest-mock    = "^3.12"
hypothesis     = "^6.100"

[tool.poetry.group.docs.dependencies]
sphinx               = "^7.3"
sphinx-rtd-theme     = "^2.0"
myst-parser          = "^3.0"

[tool.poetry.group.lint.dependencies]
ruff       = "^0.5"
black      = "^24.0"
isort      = "^5.13"

# Optional group — NOT installed by default (must opt-in via --with)
[tool.poetry.group.gpu]
optional = true

[tool.poetry.group.gpu.dependencies]
torch        = { version = "^2.3", source = "pytorch-cu121" }
nvidia-cuda  = "^12.1"

[tool.poetry.group.plotting]
optional = true

[tool.poetry.group.plotting.dependencies]
matplotlib = "^3.8"
seaborn    = "^0.13"
TOML

# Behavior cheatsheet:
#   poetry install                 -> main + all NON-optional groups (dev, test, docs, lint above)
#   poetry install --with gpu      -> add the optional gpu group
#   poetry install --without dev   -> exclude dev group
#   poetry install --only main     -> ONLY [tool.poetry.dependencies] (production install)
#   poetry install --only dev,test -> ONLY dev + test groups (CI matrix split)
#   poetry install --sync          -> remove anything not in lock + selected groups (true sync)
```

## pyproject.toml — Extras

```bash
cat >> pyproject.toml <<'TOML'
# Extras = optional install profiles for END USERS doing `pip install my-app[redis]`.
# (Groups are for DEVS; extras are for users of your published package.)

[tool.poetry.dependencies]
python = "^3.10"
requests = "^2.31"
# Mark deps as optional — they exist in the lockfile but skip default install
redis    = { version = "^5.0", optional = true }
hiredis  = { version = "^2.3", optional = true }
boto3    = { version = "^1.34", optional = true }
sqlalchemy = { version = "^2.0", optional = true }
psycopg2-binary = { version = "^2.9", optional = true }

[tool.poetry.extras]
# Each extra = list of optional deps to pull in
redis = ["redis", "hiredis"]
aws   = ["boto3"]
postgres = ["sqlalchemy", "psycopg2-binary"]
all   = ["redis", "hiredis", "boto3", "sqlalchemy", "psycopg2-binary"]
TOML

# Users install via:
#   pip install my-app[redis]
#   pip install my-app[redis,aws]
#   pip install my-app[all]
# Devs install locally via:
#   poetry install --extras "redis aws"
#   poetry install -E redis -E aws
#   poetry install --all-extras
```

## pyproject.toml — Scripts

```bash
cat >> pyproject.toml <<'TOML'
# Console scripts — installed into the venv's bin/ as executables
[tool.poetry.scripts]
my-cli         = "my_app.cli:main"             # most common form: module:function
serve          = "my_app.server:run"
migrate        = "my_app.db:migrate"
admin-shell    = "my_app.admin:shell"

# With extras (script only available when extra is installed)
gui = { reference = "my_app.gui:start", type = "console", extras = ["gui"] }

# Reference an executable from a dependency (rare)
black = { reference = "black", type = "file" }
TOML

# After `poetry install`, these are real executables:
poetry run my-cli --help
poetry run serve --port 8000
ls $(poetry env info --path)/bin/        # see them
```

## pyproject.toml — [build-system]

```bash
cat >> pyproject.toml <<'TOML'
# REQUIRED — PEP 517 build backend declaration
[build-system]
requires      = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"

# Without [build-system] you get:
#   "Can not execute setup.py since setuptools is not available"
#   "ERROR: Could not build wheels for X, which is required to install pyproject.toml-based projects"

# Pin poetry-core for reproducible CI builds
[build-system]
requires      = ["poetry-core==1.9.0"]
build-backend = "poetry.core.masonry.api"

# Custom build script (compile C extensions etc.)
[tool.poetry.build]
script          = "build.py"
generate-setup-file = false
TOML
```

## Install

```bash
# Full install (main + all non-optional groups) — the default for devs
poetry install

# Production install (no dev/test/docs)
poetry install --only main
poetry install --without dev,test,docs

# Specific groups only
poetry install --with dev
poetry install --with dev,test
poetry install --without docs
poetry install --only test            # ONLY test group, nothing else
poetry install --only main,test       # main + test

# Sync — remove anything not in lockfile + selected groups (true clean state)
poetry install --sync
poetry install --sync --without dev   # production-style sync

# Skip installing the ROOT package itself (lib-only install — common in CI)
poetry install --no-root

# All extras
poetry install --all-extras
poetry install --extras "redis aws"
poetry install --extras redis --extras aws
poetry install -E redis -E aws

# Compile .py to .pyc on install (cold-start optimisation)
poetry install --compile

# Quiet / verbose
poetry install -q
poetry install -v
poetry install -vv          # show resolver decisions
poetry install -vvv         # full debug

# Dry run — show what would happen
poetry install --dry-run

# Skip the lockfile — danger, only for one-off rebuild
poetry install --no-interaction --no-ansi
```

## Add

```bash
# Add to main deps — `add` does BOTH `poetry.lock update` AND `poetry install`
poetry add requests
poetry add "requests>=2.31,<3.0"
poetry add requests@^2.31
poetry add requests==2.31.0
poetry add "requests>=2.31"
poetry add requests@latest

# Add to a group (replaces deprecated `--dev`)
poetry add --group dev pytest
poetry add --group dev pytest ruff mypy
poetry add --group test pytest-cov pytest-xdist
poetry add --group docs sphinx myst-parser
poetry add -G dev pytest                       # short form

# Multiple at once
poetry add requests httpx pydantic

# Optional dep (mark as optional in pyproject — must be paired with [tool.poetry.extras])
poetry add --optional redis
poetry add --optional hiredis

# Extras of the dep
poetry add "fastapi[all]"
poetry add fastapi --extras "all"
poetry add fastapi -E all -E uvicorn

# Constrain Python compat for this dep
poetry add "tensorflow@^2.15" --python "^3.10"

# Pre-releases / RCs
poetry add "django@^5.0" --allow-prereleases

# Specific source (private repo)
poetry add my-internal-lib --source internal

# Update lock only — don't install
poetry add requests --lock

# Dry run — show resolution without modifying anything
poetry add requests --dry-run

# Bypass cache (force fresh fetch)
poetry add requests --no-cache

# Editable / path / git / url installs
poetry add --editable ../shared-lib
poetry add ../shared-lib                       # implicit non-editable
poetry add "git+https://github.com/example/foo.git#main"
poetry add "git+https://github.com/example/foo.git@v1.2.3"
poetry add "git+ssh://git@github.com/example/foo.git#abc123"
poetry add "https://example.com/pkg-1.0.tar.gz"

# Common idiom — install dev tooling group in one line
poetry add --group dev pytest pytest-cov pytest-xdist ruff mypy black
```

## Remove

```bash
poetry remove requests
poetry remove requests httpx                   # multiple
poetry remove --group dev pytest
poetry remove -G dev pytest mypy
poetry remove --dry-run requests
```

## Update

```bash
# Update everything to latest within constraints + rewrite lockfile
poetry update

# Single package
poetry update requests
poetry update requests httpx

# Group filtering
poetry update --with dev
poetry update --without docs
poetry update --only main

# Update LOCK only — don't install (CI-friendly)
poetry update --lock

# Dry run
poetry update --dry-run

# Verbose — see resolver
poetry update -v
poetry update -vvv

# Legacy: `--no-dev` (deprecated, still parsed) — use `--without dev`
# poetry update --no-dev               # DEPRECATED — silently maps to --without dev
```

## Lock

```bash
# Re-resolve and rewrite poetry.lock from pyproject.toml
poetry lock

# Verify lock is up-to-date (CI gate) — exits non-zero if pyproject changed
poetry lock --no-update
poetry check --lock                            # equivalent in newer Poetry

# Force regenerate from scratch (Poetry 1.7+) — discard old lock entirely
poetry lock --regenerate

# Pre-resolve without modifying — print plan
poetry lock --dry-run

# Verbose
poetry lock -vvv

# After bumping pyproject.toml manually:
$EDITOR pyproject.toml
poetry lock                                    # re-resolve
poetry install                                 # apply lock to venv
```

## Show

```bash
# Top-level deps only
poetry show

# Tree (with transitive deps + version constraints)
poetry show --tree
poetry show --tree requests                    # subtree of one package

# Latest available versions vs. installed
poetry show --latest

# Outdated only (filter)
poetry show --outdated

# Top-level only (no transitives)
poetry show --top-level

# Why is X installed? — shows reverse-deps chain
poetry show --why pytest
poetry show pytest                             # detailed info on one pkg

# Filter by group
poetry show --with dev
poetry show --without docs
poetry show --only main

# Output formats
poetry show --no-ansi                          # plain text for piping
```

## Run

```bash
# Run any command inside the project's venv (no activation needed)
poetry run python script.py
poetry run python -c "import sys; print(sys.executable)"
poetry run python -m my_app
poetry run pytest
poetry run pytest -xvs tests/test_foo.py
poetry run ruff check .
poetry run mypy .
poetry run my-cli --help                       # console_scripts from [tool.poetry.scripts]

# Pass through env vars
LOG_LEVEL=DEBUG poetry run python -m my_app

# Chain inside one shell process
poetry run sh -c "ruff check . && mypy . && pytest"

# Inspect what venv `poetry run` uses
poetry run which python
poetry run python --version
```

## Shell

```bash
# Activate venv interactively — DEPRECATED in Poetry 1.8+
poetry shell                                   # works but prints deprecation warning

# 1.8+ replacement — print activation script
poetry env activate                            # prints something like: source /path/.venv/bin/activate
eval $(poetry env activate)                    # ACTUAL activation in current shell
deactivate                                     # leave the venv (standard venv command)

# Restore the old `poetry shell` — install the plugin
poetry self add poetry-plugin-shell
poetry shell                                   # works again, no deprecation warning

# Without activating — just run things via `poetry run`
poetry run python script.py
```

## env

```bash
# List all venvs poetry knows about for THIS project
poetry env list
poetry env list --full-path

# Detailed info on the active venv
poetry env info
poetry env info --path                         # just the path (scriptable)
poetry env info --executable                   # just the python path

# Switch python interpreters (creates a new venv if missing)
poetry env use python3.11
poetry env use python3.12
poetry env use /opt/python-3.13/bin/python
poetry env use 3.12                            # short form (uses py launcher / pyenv)
poetry env use system                          # use system python (no isolated venv)

# Create venv inside the project as ./.venv  (IDE-friendly — VSCode, JetBrains autodetect)
poetry config virtualenvs.in-project true
rm -rf $(poetry env info --path)               # nuke old global-cache venv
poetry install                                 # creates ./.venv

# Remove a venv
poetry env remove python3.11
poetry env remove $(poetry env info --path)
poetry env remove --all                        # nuke ALL venvs for this project

# Activate the env (1.8+ way)
poetry env activate                            # prints activation command
eval $(poetry env activate)

# Force-create venv even if already in one (avoid PYTHONHOME inheritance)
poetry config virtualenvs.create true
```

## Build

```bash
# Build both sdist and wheel (default) — outputs to dist/
poetry build
ls dist/
#   my-app-0.1.0.tar.gz                # sdist
#   my_app-0.1.0-py3-none-any.whl      # wheel

# Wheel only
poetry build --format wheel
poetry build -f wheel

# Sdist only
poetry build --format sdist
poetry build -f sdist

# Both (explicit)
poetry build --format both

# Clean before build (avoid stale dist/ files)
rm -rf dist/
poetry build

# Verbose
poetry build -vvv

# Inspect the wheel
unzip -l dist/my_app-0.1.0-py3-none-any.whl
tar tzf dist/my-app-0.1.0.tar.gz | head -30
python -m zipfile -l dist/my_app-0.1.0-py3-none-any.whl
```

## Publish

```bash
# Publish to PyPI (default repo)
poetry publish                                 # requires prior `poetry build`
poetry publish --build                         # build + publish in one step

# Publish to TestPyPI first (always do this for new releases)
poetry config repositories.testpypi https://test.pypi.org/legacy/
poetry config pypi-token.testpypi pypi-AgENd...
poetry publish --repository testpypi --build
poetry publish -r testpypi --build             # short form

# Private index
poetry config repositories.internal https://nexus.example.com/repository/pypi-internal/
poetry config http-basic.internal myuser mypass
poetry publish --repository internal --build

# Token-based auth (preferred over user/pass for PyPI)
poetry config pypi-token.pypi pypi-AgEIcHlwaS5vcmcCJDA...
poetry publish

# One-shot creds (no config persistence)
poetry publish --username __token__ --password pypi-AgEIcHl...
poetry publish -u __token__ -p $PYPI_TOKEN

# Idempotency — skip if version already published (CI re-run safety)
poetry publish --skip-existing

# Dry run — validate everything without uploading
poetry publish --dry-run --build

# Combined: build + publish + skip-existing for CI
poetry publish --build --skip-existing -r internal

# Verbose — see HTTP requests
poetry publish -vvv

# Inspect what would be uploaded
ls -lh dist/
```

## Source / Repos

```bash
# Add a custom index
poetry source add internal https://nexus.example.com/repository/pypi-internal/simple/
poetry source add --priority=primary    internal https://nexus.example.com/...
poetry source add --priority=supplemental gpu     https://download.pytorch.org/whl/cu121
poetry source add --priority=explicit   pytorch  https://download.pytorch.org/whl/cu121

# Priorities (1.5+):
#   primary       — replaces PyPI as the default
#   supplemental  — searched only if pkg not found on primary
#   explicit      — used ONLY when a dep specifies `source = "pytorch"`
#   secondary     — DEPRECATED (was: search PyPI first, then this — confusing semantics)

# List configured sources
poetry source show
poetry source show --json

# Remove a source
poetry source remove internal

# Source URL must be a PEP 503 simple index (ends with /simple/ on most servers)
#   PyPI:           https://pypi.org/simple/
#   TestPyPI:       https://test.pypi.org/simple/
#   AWS CodeArtifact: https://<domain>-<acct>.d.codeartifact.<region>.amazonaws.com/pypi/<repo>/simple/
#   GCP Artifact Reg: https://<region>-python.pkg.dev/<project>/<repo>/simple/
#   Azure Artifacts: https://pkgs.dev.azure.com/<org>/_packaging/<feed>/pypi/simple/
#   GitHub Packages: https://maven.pkg.github.com/<owner>/<repo> (NOT supported — use a proxy)

# Pin a dep to an explicit source (in pyproject.toml)
#   torch = { version = "^2.3", source = "pytorch" }

# Show effective resolution order
poetry config --list | grep -i repo
```

## Auth

```bash
# Token-based (PyPI)
poetry config pypi-token.pypi pypi-AgEIcHlwaS5vcmcCJDA...
poetry config pypi-token.testpypi pypi-AgENd...
poetry config pypi-token.<repo-name> <token>

# Basic auth (user/pass) — for private indexes
poetry config http-basic.<repo-name> <username> <password>
poetry config http-basic.internal myuser 'p@ssw0rd!'
poetry config http-basic.internal myuser   # password prompted interactively

# Remove stored creds
poetry config --unset pypi-token.pypi
poetry config --unset http-basic.internal

# CI-friendly via env vars (skip `poetry config` entirely)
export POETRY_HTTP_BASIC_INTERNAL_USERNAME=myuser
export POETRY_HTTP_BASIC_INTERNAL_PASSWORD=$INTERNAL_PASSWORD
export POETRY_PYPI_TOKEN_PYPI=pypi-AgEI...
export POETRY_PYPI_TOKEN_INTERNAL=pypi-AgEI...

# Pattern: POETRY_<KEY>_<REPO_UPPER>_<USERNAME|PASSWORD>
#   repo "my-corp" -> POETRY_HTTP_BASIC_MY_CORP_USERNAME (hyphens become underscores!)
#   repo "internal" -> POETRY_HTTP_BASIC_INTERNAL_USERNAME

# Keyring integration (uses macOS Keychain / GNOME Keyring / Windows Credential Locker)
poetry config keyring.enabled true            # default since 1.2
poetry config keyring.enabled false           # opt out (necessary in CI without keyring)

# Bypass keyring temporarily
PYTHON_KEYRING_BACKEND=keyring.backends.null.Keyring poetry install

# AWS CodeArtifact token rotation (12h expiry)
export CODEARTIFACT_TOKEN=$(aws codeartifact get-authorization-token \
    --domain my-domain --query authorizationToken --output text)
poetry config http-basic.codeartifact aws $CODEARTIFACT_TOKEN

# GCP Artifact Registry — uses gcloud as keyring helper
pip install keyrings.google-artifactregistry-auth
poetry config keyring.enabled true
# poetry now picks up gcloud's application-default credentials automatically
```

## Config

```bash
# View everything
poetry config --list
poetry config --list --local                   # project-level overrides
poetry config <key>                            # read single value
poetry config <key> <value>                    # write single value
poetry config <key> --unset                    # delete key

# Project-local config (~/.config/pypoetry/config.toml is global; --local is in-tree)
poetry config virtualenvs.in-project true --local
ls poetry.toml                                 # generated

# Most useful keys
poetry config virtualenvs.create true                 # auto-create venv on install (default)
poetry config virtualenvs.in-project true             # ./.venv instead of ~/.cache (IDE-friendly)
poetry config virtualenvs.path ~/Code/venvs           # global venv root (default ~/.cache/pypoetry/virtualenvs)
poetry config virtualenvs.options.no-pip true         # don't bootstrap pip into venv
poetry config virtualenvs.options.no-setuptools true  # don't install setuptools
poetry config virtualenvs.options.system-site-packages false
poetry config virtualenvs.prompt "{project_name}-py{python_version}"
poetry config virtualenvs.prefer-active-python true   # use $(which python) instead of patch-detected
poetry config virtualenvs.use-poetry-python false     # 2.0+ — use Python that runs Poetry itself

poetry config cache-dir ~/.cache/pypoetry              # central cache root
poetry config cache-dir /tmp/pypoetry-cache --local    # per-project cache override

poetry config installer.parallel true                  # parallel installs (default)
poetry config installer.max-workers 8                  # downloader threads
poetry config installer.no-binary :all:                # never use wheels (force build from sdist)
poetry config installer.no-binary "pkg-a,pkg-b"        # build these from sdist only
poetry config installer.only-binary "pkg-a,pkg-b"      # never build these from sdist
poetry config installer.modern-installation true       # 1.4+ — newer installer (default 1.6+)
poetry config installer.re-resolve false               # 1.7+ — skip re-resolution if lock matches

poetry config experimental.system-git-client true      # use system `git` instead of bundled dulwich
poetry config solver.lazy-wheel true                   # 1.8+ — partial-download wheel metadata (faster)

poetry config repositories.<name> <url>                # equivalent of `poetry source add` for legacy
poetry config http-basic.<repo> <user> <pass>
poetry config pypi-token.<repo> <token>
poetry config keyring.enabled true

# Reset everything
rm -rf ~/.config/pypoetry/                             # Linux global config
rm -rf ~/Library/Application\ Support/pypoetry/        # macOS global config
rm poetry.toml                                         # local override
```

## poetry.lock

```bash
# WHEN TO COMMIT
#   * Applications:  ALWAYS commit poetry.lock (reproducibility)
#   * Libraries:     ALSO commit poetry.lock — but it doesn't constrain consumers
#                     (consumers re-resolve when they `poetry add your-lib`)

# Lockfile semantics
#   * Records EXACT versions of every transitive dep
#   * Records SHA-256 hashes for each artifact (wheel + sdist)
#   * Records resolution markers (python_version, sys_platform, extras)
#   * On `poetry install`, hashes are RE-VERIFIED — tampering = failure

# Inspect
head -30 poetry.lock
grep '^name = ' poetry.lock | head -20         # all locked package names
grep -A2 '^name = "requests"' poetry.lock      # version + hash for one pkg

# content-hash field
grep content-hash poetry.lock                  # ties lockfile to pyproject.toml

# python-versions field (under [metadata])
grep python-versions poetry.lock               # lock's python-compat range

# Verify lock matches pyproject without modifying anything
poetry lock --no-update
poetry check --lock

# CI gate (recommended)
poetry check --lock --strict || exit 1

# Update flow when you change pyproject.toml
$EDITOR pyproject.toml                          # tweak constraints
poetry lock --no-update                         # FAIL — lock and pyproject diverged
poetry lock                                     # re-resolve
poetry install                                  # apply

# Update single package without bumping the rest
poetry update requests                          # rewrites lock for requests + its sub-deps only

# Lockfile bloat in monorepos — generated by relative paths
#   Path deps with develop=true bake the FULL local path into lock —
#   commit only on the canonical machine, NOT in CI.

# Rebuild from scratch (1.7+)
poetry lock --regenerate
```

## Plugins

```bash
# Manage plugins via `poetry self <cmd>` (1.2+)
poetry self show plugins
poetry self add poetry-plugin-export                   # export to requirements.txt
poetry self add poetry-plugin-bundle                   # bundle venv into a tarball / zipapp
poetry self add poetry-plugin-shell                    # restore deprecated `poetry shell`
poetry self add poetry-dynamic-versioning              # version from git tags
poetry self add poetry-plugin-up                       # bulk update commands
poetry self add poetry-plugin-poetryup                 # similar — bumps caret/tilde
poetry self add poetry-plugin-mono-add                 # monorepo workflows

# Pin plugin version
poetry self add "poetry-plugin-export@^1.7"

# Remove
poetry self remove poetry-plugin-export
poetry self remove poetry-dynamic-versioning

# Inspect plugin tree
poetry self show --tree

# Plugin install location
ls $(poetry config cache-dir 2>/dev/null || echo ~/.cache/pypoetry)/virtualenvs/

# poetry-dynamic-versioning — version derived from git tags
cat >> pyproject.toml <<'TOML'
[tool.poetry-dynamic-versioning]
enable = true
vcs = "git"
style = "semver"
format-jinja = "{{ base }}{% if distance %}.dev{{ distance }}+{{ commit }}{% endif %}"

[build-system]
requires = ["poetry-core>=1.0.0", "poetry-dynamic-versioning>=1.0.0"]
build-backend = "poetry_dynamic_versioning.backend"
TOML
```

## Export

```bash
# Requires the export plugin (was bundled <1.2, now external)
poetry self add poetry-plugin-export

# requirements.txt for legacy tools (Lambda, Docker images using pip, etc.)
poetry export -f requirements.txt -o requirements.txt

# With dev deps
poetry export -f requirements.txt -o requirements-dev.txt --with dev

# Without hashes (smaller file, lower trust — only if your private index doesn't expose them)
poetry export -f requirements.txt -o requirements.txt --without-hashes

# Specific groups
poetry export -f requirements.txt -o requirements-test.txt --only test
poetry export -f requirements.txt -o requirements-prod.txt --without dev,test,docs

# Include extras
poetry export -f requirements.txt -o requirements.txt --extras "redis aws"
poetry export -f requirements.txt -o requirements.txt -E redis -E aws --all-extras

# Constraints file (for `pip install -c`)
poetry export -f constraints.txt -o constraints.txt

# Multi-format
poetry export -f requirements.txt -o requirements.txt
poetry export -f requirements.txt -o requirements-dev.txt --with dev
poetry export -f requirements.txt -o requirements-docs.txt --only docs

# Verify exported file installs cleanly
python -m venv /tmp/verify-export
/tmp/verify-export/bin/pip install -r requirements.txt
/tmp/verify-export/bin/pip check

# Common Docker idiom (multi-stage)
poetry export -f requirements.txt --output requirements.txt --without-hashes --with dev
# then COPY requirements.txt and `pip install -r requirements.txt` in the image
```

## check

```bash
# Validate pyproject.toml — required fields, license, classifiers, syntactic sanity
poetry check

# Strict mode — promote warnings to errors (CI gate)
poetry check --strict

# Verify lockfile matches pyproject (1.6+)
poetry check --lock

# Combined CI gate
poetry check --strict --lock

# Sample warnings poetry check catches
#   * "License classifier is deprecated, use SPDX expression"
#   * "Declared license expression doesn't match SPDX classifier"
#   * "The 'description' field is required"
#   * "Dependency 'X' is duplicated"
#   * "The lock file is not up to date with the latest changes in pyproject.toml"

# Use in pre-commit
poetry check --strict --lock || { echo "Run: poetry lock"; exit 1; }
```

## Common Errors

```bash
# 1. Resolver conflict — most common error
#
#   "SolverProblemError
#    Because my-app depends on both pkg-a (^1.0) and pkg-b (^2.0) which depends on pkg-a (>=2,<3),
#    version solving failed."
#
#   CAUSE: two deps want incompatible versions of a transitive dep.
#   FIX:
poetry show --tree pkg-a                       # find who pulls pkg-a in
poetry show --why pkg-a
# Then: relax constraints, override transitive, or upgrade conflicting top-level dep.
poetry add "pkg-b@^3.0"                        # if pkg-b@^3 is compatible
# Or override directly:
#   [tool.poetry.dependencies]
#   pkg-a = "^2.5"      # force a version both can satisfy

# 2. Group not found
#
#   "Group(s) not found: dev"
#
#   CAUSE: pyproject.toml has no [tool.poetry.group.dev.dependencies] section.
#   FIX: add the section, or fix typo in --group flag.
cat >> pyproject.toml <<'TOML'
[tool.poetry.group.dev.dependencies]
pytest = "^8.0"
TOML
poetry add --group dev pytest                  # now works

# 3. Invalid version
#
#   "InvalidVersion: Invalid version 'X'"
#
#   CAUSE: version field doesn't match PEP 440 (semver-like).
#   FIX: use canonical version strings.
sed -i 's/version = "v1.0"/version = "1.0.0"/' pyproject.toml      # remove the leading v
# Valid:   "1.0", "1.0.0", "1.0.0a1", "1.0.0.dev1", "1.0.0+build.42"
# Invalid: "v1.0", "1.0-rc1", "1.0_rc1"

# 4. Build backend missing
#
#   "Can not execute setup.py since setuptools is not available in the environment."
#   "ERROR: Could not build wheels for X, which is required to install pyproject.toml-based projects"
#
#   CAUSE: project (or a dep) lacks [build-system] section.
#   FIX: add it.
cat >> pyproject.toml <<'TOML'
[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"
TOML

# 5. Missing pyproject.toml
#
#   "ProjectNotFound
#    Poetry could not find a pyproject.toml file in /path/or/parent/dirs"
#
#   "RuntimeError
#    Poetry could not find a pyproject.toml file in /current/dir or its parents"
#
#   FIX: cd to the right dir, or create one.
cd $(git rev-parse --show-toplevel)            # jump to project root
ls pyproject.toml || poetry init -n            # create if truly missing

# 6. Package not on index
#
#   "PackageNotFound
#    Package 'my-internal-lib' not found"
#
#   CAUSE: poetry can't see the index that hosts this package.
#   FIX: add the source.
poetry source add internal https://nexus.example.com/repository/pypi-internal/simple/
poetry config http-basic.internal user pass
poetry add my-internal-lib --source internal

# 7. Network failure
#
#   "ConnectionError: HTTPSConnectionPool(host='pypi.org', port=443):
#    Max retries exceeded with url: /simple/requests/"
#
#   "URLError: <urlopen error [Errno -2] Name or service not known>"
#   "URLError: <urlopen error [Errno -3] Temporary failure in name resolution>"
#
#   FIX: check network, proxy, DNS.
curl -I https://pypi.org/simple/
export HTTPS_PROXY=http://proxy.corp.example.com:3128
poetry install -vvv                            # see exact URL it failed on

# 8. Auth failure
#
#   "AuthenticationError
#    HTTPError 401 Client Error: Unauthorized for url: ..."
#   "HTTPError: 403 Client Error: Forbidden"
#
#   CAUSE: missing/wrong creds for a private index, or token revoked.
#   FIX:
poetry config --list | grep -i internal        # check current creds
poetry config http-basic.internal user newpass
# Or env:
export POETRY_HTTP_BASIC_INTERNAL_USERNAME=user
export POETRY_HTTP_BASIC_INTERNAL_PASSWORD=newpass

# 9. Add reverts pyproject
#
#   "Failed to add packages, reverting the pyproject.toml file to its original content."
#
#   CAUSE: resolver failed mid-add — common after a stale lockfile or transitive conflict.
#   FIX:
poetry lock                                    # rebuild lock first
poetry add the-package                         # retry
# Or with verbose to see the real failure:
poetry add the-package -vvv

# 10. Invalid pyproject
#
#   "The Poetry configuration is invalid:
#     - Property 'name' is required
#     - 'authors' must be an array"
#
#   FIX: re-read pyproject.toml against this sheet's [tool.poetry] section.
poetry check --strict                          # gives line-by-line errors

# 11. Cannot determine package name
#
#   "Poetry was unable to determine the package's name. Please specify it."
#
#   CAUSE: building from a directory that lacks [tool.poetry] name field.
#   FIX:
sed -i '/^\[tool.poetry\]/a name = "my-app"' pyproject.toml
# Or explicitly:
#   [tool.poetry]
#   name = "my-app"
#   version = "0.1.0"

# 12. Lockfile out of sync (CI)
#
#   "pyproject.toml changed significantly since poetry.lock was last generated.
#    Run `poetry lock` to fix the lock file."
#
#   FIX:
poetry lock                                    # regenerate
git add poetry.lock
git commit -m "chore: refresh poetry.lock"

# 13. Hash mismatch
#
#   "Failed to install <pkg>: hash mismatch in lock file"
#   "Retrieved digest for link <pkg>-1.0.tar.gz (sha256:abc...)
#    not in installed-hashes (sha256:def...)"
#
#   CAUSE: package was re-uploaded to PyPI (yanked + replaced) — extremely rare.
#   FIX: clear cache + re-lock.
poetry cache clear --all PyPI
poetry lock --regenerate

# 14. Python version mismatch
#
#   "The current project's Python requirement (^3.9) is not compatible with
#    some of the required packages Python requirement"
#
#   FIX: relax python constraint or pin a compatible dep version.
sed -i 's/python = "\^3.9"/python = "^3.10"/' pyproject.toml
poetry lock

# 15. Keyring failure in CI
#
#   "ERROR Backend keyring.backends.fail.Keyring is not configured properly"
#
#   FIX: disable keyring in CI.
export POETRY_VIRTUALENVS_CREATE=true
poetry config keyring.enabled false
# or per-invocation:
PYTHON_KEYRING_BACKEND=keyring.backends.null.Keyring poetry install
```

## Common Gotchas

```bash
# 1. Forgetting that `poetry add` ALREADY installs.
#    BROKEN MENTAL MODEL — running both:
poetry add requests
poetry install                         # redundant — `add` already ran install

# FIX — `poetry add` does:
#   1. Resolve and update poetry.lock
#   2. Update pyproject.toml
#   3. Install into the venv (UNLESS --lock-only)
poetry add requests                    # done
# Use --lock-only when you want to split the steps (CI):
poetry add requests --lock             # only writes pyproject + lock, no install

# ---

# 2. `poetry shell` deprecated in 1.8+.
#    BROKEN — running on Poetry 1.8+:
poetry shell
# > "The shell command is deprecated. Use `poetry env activate` instead."

# FIX (option A) — use eval activate
eval $(poetry env activate)

# FIX (option B) — restore shell via plugin
poetry self add poetry-plugin-shell
poetry shell                           # works again

# FIX (option C) — never activate, use `poetry run`
poetry run python script.py

# ---

# 3. PEP 621 [project] vs [tool.poetry] confusion.
#    BROKEN on Poetry < 2.0:
cat > pyproject.toml <<'TOML'
[project]
name = "my-app"
version = "0.1.0"
dependencies = ["requests>=2.31"]
TOML
poetry install
# > "RuntimeError: Poetry could not find a pyproject.toml file with a [tool.poetry] section"

# FIX — pre-2.0 needs [tool.poetry]:
cat > pyproject.toml <<'TOML'
[tool.poetry]
name = "my-app"
version = "0.1.0"
description = "..."
authors = ["You <you@example.com>"]

[tool.poetry.dependencies]
python = "^3.10"
requests = "^2.31"

[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"
TOML
# Poetry 2.0+ supports BOTH [project] (preferred) and [tool.poetry] for legacy knobs.

# ---

# 4. Venv created in cache surprises devs.
#    BROKEN — VSCode can't find interpreter, devs assume venv is in `./venv`:
poetry install
ls -la                                 # no .venv/ visible — wat?
poetry env info --path                 # /home/user/.cache/pypoetry/virtualenvs/my-app-AbC123-py3.11

# FIX — make Poetry create venv in-tree (IDE-friendly):
poetry config virtualenvs.in-project true
poetry env remove --all
poetry install                         # now creates ./.venv
echo ".venv/" >> .gitignore

# ---

# 5. `poetry add --dev` deprecated, `--group dev` is the new syntax.
#    BROKEN on Poetry 1.2+:
poetry add --dev pytest
# > "The --dev option is deprecated, use --group dev instead."

# FIX:
poetry add --group dev pytest
poetry add -G dev pytest

# ---

# 6. Lockfile content-hash mismatch in CI.
#    BROKEN — CI fails after teammate updates pyproject without regenerating lock:
poetry install
# > "Warning: poetry.lock is not consistent with pyproject.toml. You may be getting outdated dependencies."
# > Or: "Pyproject.toml changed significantly since poetry.lock was last generated."

# FIX — fail fast in CI:
poetry check --lock --strict
# If fails, locally:
poetry lock
git add poetry.lock pyproject.toml
git commit -m "chore: sync poetry.lock"

# ---

# 7. `poetry update` updating WAY more than expected.
#    BROKEN — innocuously updating one dep yanks 200 packages:
poetry update                          # updates EVERY dep within constraints

# FIX — update only what you mean to:
poetry update requests                 # just requests + its sub-deps
poetry update --only main              # skip dev/test groups
poetry update --lock                   # write lock only, no install

# ---

# 8. `poetry install` skipping the project itself in src layout.
#    BROKEN — tests can't import the package:
#   src/my_app/__init__.py
#   tests/test_foo.py: from my_app import bar       # ModuleNotFoundError

poetry run pytest                      # > ModuleNotFoundError: No module named 'my_app'

# FIX — add explicit packages directive in pyproject.toml:
cat >> pyproject.toml <<'TOML'
[tool.poetry]
packages = [
  { include = "my_app", from = "src" },
]
TOML
poetry install                         # now installs editable copy of my_app

# ---

# 9. Path dependency leaks absolute path into lockfile.
#    BROKEN — `poetry add ../shared-lib` → lock contains `/Users/alice/code/shared-lib`,
#             CI on Linux fails with "no such directory".

# FIX — use relative path that resolves on every machine, OR factor shared-lib into a
#       proper internal index dep:
shared-lib = { path = "../shared-lib", develop = true }   # relative, NOT absolute
# Or: publish shared-lib to internal PyPI and depend on it normally.

# ---

# 10. `poetry build` shipping tests/dev files in the wheel.
#     BROKEN — wheel size 50MB instead of 1MB; pyproject lacks packages directive,
#              so Poetry includes everything top-level.

# FIX — explicit packages + exclude:
[tool.poetry]
packages = [{ include = "my_app", from = "src" }]
exclude = ["tests/**", "**/*.test.py", "docs/**", "build/**"]

# ---

# 11. Forgetting --no-root in CI causes "package not installable" loops.
#     BROKEN — CI image has no source, just pyproject + lock for caching:
poetry install                         # > "ProjectError: cannot install editable mode"

# FIX:
poetry install --no-root --no-interaction --sync --without dev,test
# Then COPY src/ + run app — no pip-install of self needed for non-library Docker images.

# ---

# 12. Caret semantics surprise on 0.x.y versions.
#     BROKEN — `^0.5.2` does NOT mean ">=0.5.2,<1.0.0".
#     Caret on 0.x is treated as "tilde" semantics: ^0.5.2 == >=0.5.2,<0.6.0.

# FIX — be explicit when you want major-1-and-up:
some-pkg = ">=0.5.2,<1.0.0"            # explicit range
some-pkg = "~0.5"                      # patch-only: >=0.5,<0.6
some-pkg = "^0.5"                      # equivalent to >=0.5.0,<0.6.0  (NOT <1.0!)
```

## Poetry vs uv vs pip+venv

```bash
# Action                       | poetry                                | uv                                  | pip + venv
# -----------------------------+---------------------------------------+-------------------------------------+----------------------------------
# Create venv                  | poetry install (auto)                 | uv venv                             | python -m venv .venv
# Activate venv                | eval $(poetry env activate)           | source .venv/bin/activate           | source .venv/bin/activate
# Add a dep                    | poetry add requests                   | uv add requests                     | pip install requests; freeze >> req
# Add to dev group             | poetry add --group dev pytest         | uv add --dev pytest                 | (none — manual)
# Install all deps             | poetry install                        | uv sync                             | pip install -r requirements.txt
# Reproducible lockfile        | poetry.lock (auto)                    | uv.lock (auto)                      | pip-compile -> requirements.txt
# Build wheel                  | poetry build                          | uv build                            | python -m build
# Publish                      | poetry publish                        | uv publish                          | twine upload dist/*
# Update one dep               | poetry update requests                | uv lock --upgrade-package requests  | pip install -U requests
# Show outdated                | poetry show --outdated                | uv pip list --outdated              | pip list --outdated
# Run a script                 | poetry run pytest                     | uv run pytest                       | source .venv/bin/activate && pytest
# Switch python                | poetry env use 3.12                   | uv python install 3.12              | (rebuild venv manually)
# Speed (cold install ~50 deps)| ~30s                                  | ~3s                                 | ~25s
# Static binary                | no (Python script)                    | yes (Rust)                          | no
# pyproject layout             | [tool.poetry] (or [project] in 2.0+)  | [project] (PEP 621)                 | n/a (setup.py / setup.cfg)
# Built-in build backend       | poetry-core                           | (uses any PEP 517 backend)          | n/a
# Plugin ecosystem             | yes                                   | minimal                             | n/a
```

## Idioms

```bash
# CI install — strict, no prompts, fail-fast
poetry install --no-interaction --no-ansi --no-root --sync --with dev

# Production install (Docker runtime stage)
poetry install --no-interaction --no-ansi --no-root --only main --sync

# Dockerfile (multi-stage, layer-cached deps)
cat > Dockerfile <<'DOCKER'
FROM python:3.12-slim AS builder
ENV POETRY_VERSION=1.8.4 \
    POETRY_HOME=/opt/poetry \
    POETRY_VIRTUALENVS_IN_PROJECT=true \
    POETRY_NO_INTERACTION=1 \
    PIP_NO_CACHE_DIR=1
RUN pip install "poetry==${POETRY_VERSION}"
WORKDIR /app
# Copy ONLY the dep-defining files first — caches the install layer
COPY pyproject.toml poetry.lock ./
RUN poetry install --no-root --only main
# Then copy the source — only this layer rebuilds on code change
COPY src/ src/
RUN poetry install --only main           # registers root package

FROM python:3.12-slim AS runtime
WORKDIR /app
COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /app/src /app/src
ENV PATH="/app/.venv/bin:${PATH}"
CMD ["python", "-m", "my_app"]
DOCKER

# CI: GitHub Actions
cat > .github/workflows/ci.yml <<'YAML'
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - name: Install Poetry
        run: pipx install poetry==1.8.4
      - name: Cache venv
        uses: actions/cache@v4
        with:
          path: ~/.cache/pypoetry
          key: poetry-${{ hashFiles('poetry.lock') }}
      - run: poetry config keyring.enabled false
      - run: poetry check --lock --strict
      - run: poetry install --no-interaction --no-root --sync --with dev
      - run: poetry run ruff check .
      - run: poetry run mypy .
      - run: poetry run pytest -xvs
YAML

# Private repo with token (CI)
poetry source add --priority=primary internal https://nexus.example.com/repository/pypi/simple/
poetry config http-basic.internal __token__ "$INTERNAL_PYPI_TOKEN"
poetry install --no-interaction --no-root --sync

# Pre-commit hook (verify lock + format)
cat > .pre-commit-config.yaml <<'YAML'
repos:
  - repo: https://github.com/python-poetry/poetry
    rev: 1.8.4
    hooks:
      - id: poetry-check
        args: ["--lock"]
      - id: poetry-lock
        args: ["--no-update"]
  - repo: https://github.com/astral-sh/ruff-pre-commit
    rev: v0.5.0
    hooks:
      - id: ruff
        args: [--fix]
      - id: ruff-format
YAML
poetry run pre-commit install

# One-shot publish (build + publish to private + skip-existing for re-runs)
poetry publish --build --skip-existing -r internal

# Bump version + tag + publish
poetry version patch                                    # 1.0.0 -> 1.0.1
# poetry version minor                                  # 1.0.0 -> 1.1.0
# poetry version major                                  # 1.0.0 -> 2.0.0
# poetry version 1.2.3                                  # set explicit
git add pyproject.toml
git commit -m "release: $(poetry version -s)"
git tag "v$(poetry version -s)"
git push --tags
poetry publish --build

# Reproducible CI lockfile gate
poetry lock --no-update --check 2>&1 | tee lock.log    # exits non-zero if drifted
poetry check --lock --strict

# Local sync to lock state (nuke stale deps)
poetry install --sync

# Run tests in fresh venv (sanity check)
poetry env remove --all
poetry install --sync --with dev,test
poetry run pytest -xvs

# Bulk update across all projects in monorepo
for d in services/*/; do (cd "$d" && poetry lock && poetry install --sync); done

# Dump effective config (debug CI surprises)
poetry config --list
poetry env info
poetry env list
poetry --version
poetry self show plugins
```

## See Also

- python — language reference (stdlib, dataclasses, typing, asyncio)
- pip — basic Python package install / uninstall
- cargo — Rust's equivalent of Poetry (build + publish + lock)
- npm — Node.js dependency manager (similar lockfile + scripts model)
- polyglot — cross-language `<command>` reference table

## References

- python-poetry.org/docs — official documentation root
- python-poetry.org/docs/cli — full CLI reference
- python-poetry.org/docs/managing-dependencies — groups, extras, sources
- python-poetry.org/docs/repositories — private indexes, auth, priorities
- python-poetry.org/docs/pyproject — pyproject.toml schema
- python-poetry.org/docs/dependency-specification — caret/tilde/markers semantics
- python-poetry.org/docs/configuration — config keys + env-var equivalents
- python-poetry.org/docs/plugins — official plugin list
- github.com/python-poetry/poetry — source, issues, releases
- github.com/python-poetry/poetry-core — build backend
- github.com/python-poetry/poetry-plugin-export — export-to-requirements plugin
- github.com/mtkennerly/poetry-dynamic-versioning — git-tag-driven versioning
- PEP 517 — A build-system independent format for source trees
- PEP 518 — Specifying minimum build-system requirements for Python projects ([build-system])
- PEP 621 — Storing project metadata in pyproject.toml ([project])
- PEP 660 — Editable installs for pyproject.toml-based builds (PEP 517)
- PEP 440 — Version Identification and Dependency Specification
- PEP 503 — Simple Repository API (the URL shape every index honours)
- PEP 508 — Dependency specification for Python Software Packages (env markers)
- packaging.python.org — PyPA packaging user guide
