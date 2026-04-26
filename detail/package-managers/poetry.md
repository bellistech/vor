# Poetry — Internals & Theory

Poetry is a Python dependency manager and packaging tool that combines what historically required several tools (pip, virtualenv, twine, setuptools) into a single command. Its central technical contributions are: a deterministic resolver (Mixology), an opinionated lockfile (poetry.lock), an integrated build backend (poetry-core), and the unification of project metadata in pyproject.toml. Understanding Poetry means understanding how each of those subsystems is implemented, why each design decision was made, and where the trade-offs live.

## Setup

Poetry was created by Sébastien Eustace in 2018, born from frustration with the fragmented Python packaging ecosystem. Before Poetry, a typical Python project required: a setup.py for distribution metadata; a requirements.txt for runtime dependencies; a requirements-dev.txt for development tools; a Pipfile or constraints file for dependency pinning; a virtualenv to isolate site-packages; a separate twine invocation to upload to PyPI; and often a tox.ini just to test against multiple Python versions. Poetry's pitch was: one tool, one config file (pyproject.toml), one lockfile (poetry.lock).

The initial design predates PEP 621 (the standard [project] table) and PEP 631 by years. Poetry chose to put metadata under [tool.poetry] in pyproject.toml — at the time, no PEP existed for cross-tool metadata. Years later, PEP 621 standardized [project], and Poetry has had to reconcile its legacy schema with the modern standard. Poetry 2.0 (released 2024) added [project] support, but the [tool.poetry] schema is preserved for backward compatibility — see Section 7.

Poetry's command-line surface is intentionally narrow:

```bash
poetry new myproject       # scaffold a new project
poetry init                # interactive pyproject.toml creation
poetry add requests        # add a dependency, update lock, install
poetry remove requests     # remove + reverse
poetry install             # install from lock
poetry update              # bump locked versions per constraints
poetry lock                # regenerate lock without installing
poetry build               # produce sdist + wheel
poetry publish             # upload to PyPI (or another index)
poetry run python script.py  # run inside the project venv
poetry shell               # open shell with venv activated (deprecated 2.0+)
poetry env use 3.12        # select Python interpreter
poetry show                # list installed packages
poetry show --tree         # dependency tree
poetry export -f requirements.txt > req.txt  # legacy export (now via plugin)
```

Behind the scenes, every command goes through three subsystems: the parser (TOML deserialization → in-memory Project model), the resolver (Mixology), and the executor (download, install, link).

The dependency tree printed by `poetry show --tree` is generated from the lockfile's `[[package]]` array — Poetry stores not just resolved versions but also the dependency edges, so it can render the tree without re-resolving.

Installation of Poetry itself is intentionally separate from any project's pip:

```bash
curl -sSL https://install.python-poetry.org | python3 -
```

The recommended installer is a standalone Python script that creates a dedicated virtualenv at `~/.local/share/pypoetry/venv/` (or platform equivalent), installs Poetry into it, and creates a launcher script in `~/.local/bin/poetry`. This means Poetry never coexists with project dependencies — your project's `requests==2.20.0` cannot break Poetry's own runtime. (uv goes further: ships as a single Rust binary, no Python at all.)

## Mixology Resolver

The Mixology resolver is Poetry's most academically interesting component. It's a CDCL (Conflict-Driven Clause Learning) SAT solver adapted for the package-resolution domain. CDCL is the same family of algorithm used in modern industrial SAT solvers like MiniSat and Glucose — Poetry didn't invent it, but it adapted it carefully to handle the specific constraint patterns of Python package metadata.

The intellectual lineage is: Bundler (Ruby) wrote Molinillo, the first widely-deployed CDCL-style package resolver; Dart's pub team, advised by some of the SAT-solving research community, designed PubGrub, which formalized "incompatibility-based" resolution; and Poetry's Mixology is a Python reimplementation that draws on both. uv's resolver (Section 14) descends directly from PubGrub.

The classical "pip" resolver is a backtracking depth-first search: pick a candidate version for the first requirement, recursively try to satisfy its sub-requirements, backtrack on conflict. This works for simple cases but becomes pathologically slow on dense dependency graphs because it doesn't *learn* from failures. Mixology learns: when it discovers that two constraints are mutually unsatisfiable, it records the *incompatibility* — a clause stating "this combination cannot all be true" — and uses that incompatibility to prune future search.

Concretely, an incompatibility is a set of *terms*. A term is a package + a version range (positive or negative). For example:

```
incompatibility: { django ∈ [3.0, 3.1), psycopg2 ∈ [2.9, ∞) }
```

means "you cannot have both Django 3.0.x and psycopg2 ≥ 2.9.0". When the solver later considers Django 3.0.5, it knows it must derive `psycopg2 < 2.9` or backtrack. This is the same propagation that Boolean Constraint Propagation (BCP) does in CDCL SAT solvers, but extended to version ranges instead of single Boolean variables.

The resolver maintains a *partial solution* — an assignment of version ranges to packages — and a *queue* of pending decisions. The main loop:

1. *Unit propagation*: For each incompatibility, check if all but one of its terms is already implied by the partial solution. If so, the remaining term must be derived (its negation added).
2. *Decision*: Pick a package with a non-empty version range whose version isn't yet decided. Pick a specific version (typically the highest available that satisfies the range).
3. *Conflict*: If unit propagation derives a term that contradicts the partial solution, you have a conflict. Compute the *root cause* by walking the implication graph; that becomes a new incompatibility (clause learning).
4. *Backjump*: Use the new incompatibility to backjump to the level where the conflict could have been avoided, and try a different decision there.
5. *Termination*: If no decisions remain and no incompatibility holds, you've found a satisfying solution. If you derive an incompatibility containing only the root term (the project's top-level requirements), the constraints are unsatisfiable.

The brilliance of CDCL is in step 3: clause learning means each conflict eliminates not just the current bad path but a whole *family* of bad paths. Poetry's resolver routinely solves dependency graphs with thousands of packages and millions of (package, version) candidate pairs in seconds — a brute-force backtracker would take hours.

The implementation in Poetry is in `poetry/mixology/` and is a faithful port of PubGrub's reference implementation, with Python idioms.

There's a price: the resolver is CPU-intensive in pure Python. Poetry releases over the years have steadily optimized the inner loop — caching version-range intersection, deduplicating package metadata, eagerly fetching metadata in parallel — but it's still slower than uv's Rust implementation, which uses identical algorithms but with native-code performance and lock-free parallel metadata fetching.

## Resolver Phase

Resolution in Poetry happens in three phases:

**Phase 1: Flattening requirements.** Poetry reads `[tool.poetry.dependencies]` and `[tool.poetry.group.<name>.dependencies]` from pyproject.toml. Each entry becomes a *root incompatibility*: the project requires this constraint to hold. For markers (e.g. `python_version >= "3.10"`), Poetry creates a conditional incompatibility that's only active when the marker matches.

```toml
[tool.poetry.dependencies]
python = ">=3.10,<4.0"
django = "^4.2"
psycopg2 = { version = "^2.9", optional = true }
redis = { version = "^5.0", markers = "python_version >= '3.11'" }
```

The flattening produces:

```
project requires python ∈ [3.10, 4.0)
project requires django ∈ [4.2, 5.0)        (^4.2 expansion)
project requires redis ∈ [5.0, 6.0) when python ≥ 3.11
psycopg2 is optional → only active if extra activated
```

Note that `^4.2` (caret) expands to `>=4.2.0,<5.0.0` per Poetry's semver semantics. Tilde `~4.2` expands to `>=4.2.0,<4.3.0`. Both differ from npm's identical-syntax semantics in subtle ways for pre-1.0 versions; Poetry's docs document the expansion table.

**Phase 2: Solving.** The Mixology loop runs against a `Source` object that fetches package metadata from configured repositories (PyPI by default, plus any sources declared in `[tool.poetry.source]`). For each package the solver decides on, the source returns:

- The list of available versions (filtered by the current incompatibilities).
- For each candidate version, the dependencies, Python-version constraints, and platform markers.

Poetry caches metadata aggressively. The first time a package is queried, it goes through the JSON simple API (PEP 691) or the legacy HTML simple API (PEP 503). Subsequent queries hit the in-process cache; across runs, the cache lives in `~/.cache/pypoetry/cache/repositories/<repo>/`.

The solver picks the *highest* version satisfying the active range. This is "newest-first" preference and is the universal default in package managers — older versions are tried only as a result of backtracking from a conflict.

**Phase 3: Lockfile production.** Once the solver terminates with a satisfying assignment, Poetry writes `poetry.lock`. The lockfile records:

- The project's content hash (a SHA-256 of the relevant pyproject.toml fields), so Poetry can detect if pyproject changed without re-locking.
- For each resolved package: name, version, dependencies, Python markers, optional-extras, source URL, and the package's content hashes (one per distribution file — wheels and sdists).

```toml
[[package]]
name = "django"
version = "4.2.7"
description = "A high-level Python Web framework..."
optional = false
python-versions = ">=3.8"
files = [
    {file = "Django-4.2.7-py3-none-any.whl", hash = "sha256:e1d3..."},
    {file = "Django-4.2.7.tar.gz", hash = "sha256:8e0f..."},
]

[package.dependencies]
asgiref = ">=3.6.0,<4"
sqlparse = ">=0.3.1"
"backports.zoneinfo" = {version = "*", markers = "python_version < \"3.9\""}
```

The lockfile is the source of truth for `poetry install`. Even if pyproject says `django = "^4.2"`, the lockfile pins exactly `4.2.7` until you `poetry update` (or `poetry update django` to update only that package).

## poetry.lock

The lockfile is plain TOML, designed to be human-readable and diff-friendly in code review. Its structure:

```toml
[[package]]                              # one entry per resolved package
name = "..."
version = "..."
files = [...]                            # per-file SHA-256 hashes
[package.dependencies]                   # adjacency list

[metadata]
lock-version = "2.0"                     # schema version
python-versions = ">=3.10,<4.0"          # union of supported Pythons
content-hash = "abc123..."               # SHA-256 of pyproject critical fields

[metadata.files]                         # legacy hash duplication (older lock-versions)
```

The `content-hash` is computed over a canonical representation of the dependency-relevant parts of pyproject.toml. If you edit a comment or rename a script, the hash doesn't change. If you bump `django = "^4.2"` to `django = "^4.3"`, it does — and `poetry install` will refuse to proceed without `poetry lock --no-update` (regenerate hash without resolving) or `poetry update` (resolve and update lock).

This drift detection is one of Poetry's most important guarantees. In teams, it means CI can verify "the lockfile matches the spec" without re-resolving, and "the lockfile was regenerated when the spec changed" without trusting humans to remember.

The lockfile schema has evolved. `lock-version = "1.1"` was the original; `1.2` added Python-version markers in dependency entries; `2.0` (Poetry 1.5+) cleaned up the file-hash representation, moving them inside each package entry instead of a separate `[metadata.files]` table. Poetry reads any version it knows; it writes the latest by default.

The `python-versions` field at top level records the *intersection* of all dependency Python-version constraints, which becomes the project's effective supported range. If your project says `python = ">=3.10,<4.0"` but a dependency requires `python = ">=3.11"`, the resolver will either downgrade that dependency or surface an unsatisfiable constraint.

## The metadata.python-versions field

Python's tagging system makes lockfiles complicated. A wheel is tagged with a Python ABI (e.g. `cp311` for CPython 3.11), an OS (e.g. `manylinux_2_17_x86_64`), and an architecture. A single resolved version of a package may have ten or more wheels — one per (Python, OS, arch) combination — plus an sdist (source distribution) for everything else.

Poetry's lockfile records *all* candidate files for the chosen version, with their hashes. At install time, the executor selects the file matching the current interpreter. This means the lockfile is *cross-platform* and *cross-Python*: a single poetry.lock works on Linux x86_64 with Python 3.10, macOS arm64 with Python 3.11, and Windows x86_64 with Python 3.12.

This wasn't always true. Older versions of Poetry (and pip's freeze approach, and pipenv at one point) produced lockfiles that pinned the specific wheel for the resolving machine. Switching machines meant re-generating the lock. Poetry's design — record all candidates, let the installer pick — was a deliberate generalization that hugely improved CI portability.

The `metadata.python-versions` field at the top level of the lockfile records the *resolution-time* Python-version constraint. The resolver assumes a project Python range and prunes versions that don't satisfy it. If you change `python = "..."` in pyproject, the lock's content-hash mismatches and you must re-lock.

There's a subtle case: a dependency's wheels may be tagged for a narrower Python range than the dependency's metadata declares. Poetry records the metadata range in the lockfile (`python-versions = ">=3.8"` for Django 4.2), but at install time, the executor checks the actual wheel tags. If no compatible wheel exists, Poetry falls back to the sdist and triggers a build (which may pull in a build-system, see Section 6).

## PEP 517/518 Build System

PEP 518 introduced the `[build-system]` table in pyproject.toml — the canonical way to declare what's needed to *build* a Python project. PEP 517 defined the API that a build *backend* must implement (functions like `build_wheel`, `build_sdist`, `prepare_metadata_for_build_wheel`).

Before PEP 517/518, every project had to use setuptools, because pip hard-coded `import setuptools` into its build process. PEP 518 broke that monopoly. Now any tool can declare its own build backend:

```toml
[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"
```

This means: when pip (or uv, or `python -m build`) wants to build this project's wheel, it should:

1. Create an isolated environment.
2. Install `poetry-core>=1.0.0` into it.
3. Import `poetry.core.masonry.api` and call its `build_wheel` function.

`poetry-core` is a stripped-down library — just the build backend, no resolver, no CLI. This separation matters: `poetry-core` has minimal dependencies, so it installs quickly and reliably even in build-isolated environments. The full Poetry CLI lives in `poetry`, which depends on `poetry-core` plus the resolver, network code, etc.

The build backend's job is to take a project's source tree (pyproject.toml + source files) and produce:

- A wheel (`.whl`): a zip containing pre-built artifacts ready to install.
- An sdist (`.tar.gz`): a source archive that can be rebuilt anywhere.

`poetry-core`'s wheel builder reads `[tool.poetry]` (or `[project]` in 2.0+), the package layout (which files belong in the wheel, controlled by `packages` and `include`/`exclude`), and produces a PEP 427-compliant wheel.

For C-extension projects, the picture is more complex. `poetry-core` does not (yet) support compiled extensions natively; projects that need them either use the `build.py` extension-build hook or fall back to setuptools as the backend. This is one of the few feature-gaps where pip's setuptools-based build remains more capable than Poetry's.

## pyproject.toml [tool.poetry] vs [project]

PEP 621 standardized the `[project]` table:

```toml
[project]
name = "myproject"
version = "0.1.0"
description = "..."
authors = [{name = "Stevie", email = "stevie@bellis.tech"}]
requires-python = ">=3.10"
dependencies = ["django>=4.2", "redis>=5.0"]

[project.optional-dependencies]
test = ["pytest>=7"]
```

Poetry's legacy schema:

```toml
[tool.poetry]
name = "myproject"
version = "0.1.0"
description = "..."
authors = ["Stevie <stevie@bellis.tech>"]

[tool.poetry.dependencies]
python = ">=3.10,<4.0"
django = "^4.2"
redis = "^5.0"

[tool.poetry.group.test.dependencies]
pytest = "^7"
```

Differences:

- `python = "..."` lives inside `[tool.poetry.dependencies]` in the legacy schema; it lives as `requires-python = "..."` outside in PEP 621.
- Poetry's caret/tilde version operators (`^4.2`, `~4.2`) are not valid PEP 440 syntax. PEP 621 uses pure PEP 440 (`>=4.2,<5.0`).
- Authors are a string list with embedded emails in legacy; PEP 621 uses structured tables with `name` and `email` fields.
- Optional dependencies use `[project.optional-dependencies]` (PEP 621) vs `[tool.poetry.dependencies]` with `optional = true` and an `[tool.poetry.extras]` table to define which extras pull which dependencies.

Poetry 2.0 supports both schemas. You can write `[project]` and Poetry will read it; you can write `[tool.poetry]` and Poetry will read that too. If both exist, `[project]` wins for the fields it defines, and `[tool.poetry]` provides Poetry-specific extensions (groups, source priorities, build customization).

The migration tension is real: tools downstream of pyproject (linters, IDE plugins, dependency scanners) increasingly expect `[project]`. Pure `[tool.poetry]` projects look like "no project metadata" to those tools. But Poetry's caret syntax and group features are heavily used, and writing them in pure PEP 621 means losing semantic meaning. The recommended path is to use `[project]` for static metadata (name, version, description, dependencies as PEP 440) and `[tool.poetry]` for tool-specific configuration only.

## Dependency Groups (Poetry 1.2+)

Before Poetry 1.2, dev dependencies were a single magic group: `[tool.poetry.dev-dependencies]`. This was rigid — what if you want separate groups for testing, linting, and documentation?

Poetry 1.2 introduced general dependency groups:

```toml
[tool.poetry.dependencies]
python = ">=3.10"
django = "^4.2"

[tool.poetry.group.test.dependencies]
pytest = "^7"
pytest-django = "^4"

[tool.poetry.group.lint.dependencies]
ruff = "^0.1"
mypy = "^1.5"

[tool.poetry.group.docs.dependencies]
sphinx = "^7"
sphinx-rtd-theme = "^1"
```

Group activation:

```bash
poetry install                        # all non-optional groups
poetry install --without docs         # exclude one
poetry install --only test            # ONLY test (no main, no lint)
poetry install --with docs            # include normally-optional docs
```

To make a group optional (not installed by default):

```toml
[tool.poetry.group.docs]
optional = true
```

Groups are just resolver inputs — they're flattened during Phase 1 of resolution. The resolver doesn't treat groups specially; it just considers more or fewer constraints depending on which groups are active.

Group dependencies are *not* included in the published wheel. When you `poetry build`, only `[tool.poetry.dependencies]` ends up as the wheel's runtime requirements. Test/lint/docs dependencies stay in pyproject.toml as development-time configuration.

PEP 735 (accepted 2024) standardizes dependency groups across tools. Poetry's syntax is a superset of PEP 735's minimum, and Poetry 2.x reads PEP 735 groups (`[dependency-groups]` at the top level) in addition to `[tool.poetry.group.*]`. uv reads PEP 735 natively.

## Sources & Repos

By default, Poetry resolves against PyPI. To use a private index:

```toml
[[tool.poetry.source]]
name = "private"
url = "https://private.example.com/simple/"
priority = "supplemental"
```

Priority levels:

- `primary` — searched first for any package. (Deprecated; replaced by explicit priority logic.)
- `default` — the implicit fallback; PyPI is `default` unless explicitly overridden.
- `supplemental` — searched only if a package isn't found in higher-priority sources.
- `explicit` — never searched implicitly; only used when a dependency declares `source = "..."` explicitly.

The priority system addresses the "dependency confusion" attack: an attacker uploads a malicious package to public PyPI with the same name as your private package; if the resolver treats public PyPI as higher priority, it pulls the attacker's code. By marking your private repo `primary` (or PyPI as `supplemental`), you protect against this.

For per-dependency source pinning:

```toml
[tool.poetry.dependencies]
mylib = { version = "^1.0", source = "private" }
```

This guarantees `mylib` is only resolved from `private`, regardless of priority logic.

The simple index protocol (PEP 503) is HTML-based: a single page listing all packages, each linking to a per-package page with hyperlinks to file URLs. PEP 691 added a JSON variant — same data, but parseable without an HTML parser, and with cleaner metadata. Poetry queries JSON-simple if available (`Accept: application/vnd.pypi.simple.v1+json`) and falls back to HTML.

Modern indexes (PyPI, AWS CodeArtifact, GitHub Packages, Artifactory) all support PEP 691. Older custom servers (devpi, simple file servers) often only support PEP 503.

## Auth

Poetry supports several auth schemes for private repos:

**HTTP Basic (username + password):**

```bash
poetry config http-basic.private myuser mypass
```

This stores credentials in `auth.toml` (or in the system keyring; see below). At fetch time, Poetry sends `Authorization: Basic base64(myuser:mypass)`.

**Bearer tokens (PyPI tokens):**

```bash
poetry config pypi-token.private my-token-value
```

Sends `Authorization: token <value>` on requests to that source. PyPI's API tokens are bearer tokens; many private indexes (CodeArtifact, GitHub Packages) accept either format.

**Environment variables:**

```bash
export POETRY_HTTP_BASIC_PRIVATE_USERNAME=myuser
export POETRY_HTTP_BASIC_PRIVATE_PASSWORD=mypass
```

The pattern is `POETRY_HTTP_BASIC_<SOURCE_NAME_UPPERCASED>_USERNAME`/`_PASSWORD`. Useful for CI where you don't want credentials in files. Poetry reads env vars in addition to (and overriding) the config files.

**Keyring integration:**

By default, Poetry stores credentials via Python's `keyring` library — on macOS, this is the system Keychain; on Linux, secret-service (gnome-keyring or KWallet); on Windows, Credential Manager. This is more secure than a plain-text auth.toml but requires a graphical session or appropriately-configured headless keyring.

To disable keyring (e.g. in Docker):

```bash
poetry config keyring.enabled false
```

Credentials then go into `auth.toml`. To set credentials non-interactively in CI:

```bash
poetry config http-basic.private $USERNAME $PASSWORD
```

(In Poetry 1.5+, the recommended pattern is env vars — they're explicit, scoped to the process, and never persisted.)

## Plugin Architecture

Poetry's plugin system lets third parties extend the CLI without modifying core. Plugins are pip-installable Python packages that declare entry points under `poetry.plugin` or `poetry.application.plugin`.

Three plugin classes:

**`Plugin`** — extends Poetry's core (resolver, installer, etc.).

```python
from poetry.plugins.plugin import Plugin

class MyPlugin(Plugin):
    def activate(self, poetry, io):
        # poetry: the Poetry object (project, config, etc.)
        # io: the IO object (input/output)
        # Modify config, register custom resolvers, etc.
        ...
```

**`ApplicationPlugin`** — adds new top-level commands.

```python
from poetry.plugins.application_plugin import ApplicationPlugin
from cleo.commands.command import Command

class HelloCommand(Command):
    name = "hello"
    description = "Say hello."
    def handle(self):
        self.line("Hello!")
        return 0

class HelloPlugin(ApplicationPlugin):
    @property
    def commands(self):
        return [HelloCommand]
```

After `poetry self add my-plugin`, `poetry hello` works.

**`BaseProjectPlugin`** — hooks into project-level lifecycle (uncommon).

The `poetry self` command manages plugins isolated from your project's environment. `poetry self add` installs into Poetry's own venv; `poetry self remove` uninstalls. This avoids the chicken-and-egg problem of "I need a plugin to install plugins."

Notable plugins:

- `poetry-plugin-export` — re-adds the `poetry export` command, which was removed from core in Poetry 1.2 to keep the CLI lean. Lets you produce requirements.txt from poetry.lock.
- `poetry-plugin-shell` — re-adds `poetry shell` (deprecated/removed from core in 2.0+).
- `poetry-plugin-up` — bulk upgrade of dependencies (uplifts version constraints in pyproject.toml).
- `poetry-dynamic-versioning` — derives version from git tags at build time.

## Build/Publish

`poetry build` produces both an sdist and a wheel by default:

```bash
$ poetry build
Building myproject (0.1.0)
  - Building sdist
  - Built myproject-0.1.0.tar.gz
  - Building wheel
  - Built myproject-0.1.0-py3-none-any.whl
```

Outputs go to `dist/`. The sdist is just a tar.gz of the source tree (with files filtered per `[tool.poetry].include`/`exclude` and the standard ignore patterns). The wheel is a zip containing:

- `myproject/` — the package's source files (or compiled artifacts).
- `myproject-0.1.0.dist-info/` — METADATA, WHEEL, RECORD, entry_points.txt.

The METADATA file is PEP 643 / PEP 753 metadata: name, version, summary, requires, classifiers, etc.

The WHEEL file declares the wheel's tags:

```
Wheel-Version: 1.0
Generator: poetry-core 1.7.0
Root-Is-Purelib: true
Tag: py3-none-any
```

`Root-Is-Purelib: true` means the package contains only Python files (no compiled C). `py3-none-any` means "any Python 3, no specific ABI, any OS, any architecture" — a universal wheel.

For C-extension projects, you'd see tags like `cp311-cp311-manylinux_2_17_x86_64`. `poetry-core` doesn't build these natively (yet); projects use `build.py` extension scripts or fall back to setuptools.

`poetry publish` uploads the contents of `dist/` to a configured repository:

```bash
poetry publish                      # to PyPI (default)
poetry publish -r private           # to a named source
poetry publish --build              # build and upload in one command
poetry publish --skip-existing      # don't fail if version already uploaded
```

Historically, twine (`pip install twine; twine upload dist/*`) was the standard upload tool. Poetry's integrated `publish` command does the same job — talks to the index's upload API (PyPI's legacy `/legacy/` endpoint, or a private equivalent), authenticates via the configured credentials, and uploads each file.

PyPI's upload endpoint is HTTP POST multipart/form-data with the wheel/sdist as a file part and metadata as form fields. The HTTP response indicates success/failure with a (somewhat irregular) status-code convention. Poetry handles all of this transparently.

## virtualenvs.in-project

By default, Poetry creates virtualenvs in a centralized cache: `~/.cache/pypoetry/virtualenvs/<project-hash>/`. Each project gets a uniquely-named venv. This avoids cluttering your project directory with `.venv` and works well when you have many projects sharing a Python interpreter.

But it means the venv path is unpredictable and tied to the project's hash. IDE integration is harder. Some teams prefer a `.venv` directory next to pyproject.toml — predictable, easily deleted, easy for `vscode`/`pycharm`/`vim` to detect.

```bash
poetry config virtualenvs.in-project true
```

After this, Poetry creates `.venv/` in the project root. (Existing centralized venvs aren't migrated; you'd need `poetry env remove --all` and `poetry install` to recreate.)

Other useful virtualenv-related config:

```bash
poetry config virtualenvs.create false        # don't create venv; install into current Python
poetry config virtualenvs.prefer-active-python true  # use `which python`, not Poetry's bundled version
poetry config virtualenvs.path /path/to/dir   # change the centralized cache location
poetry config virtualenvs.options.system-site-packages true  # inherit global packages
```

`virtualenvs.create false` is useful in Docker, where the image *is* the environment — you don't want a nested venv inside the container. Just install into the system Python.

`virtualenvs.prefer-active-python` lets you control Python version via `pyenv`/`asdf`/`uv python`, with Poetry following along instead of choosing on its own. The default behavior was for Poetry to pick a Python it found in PATH; `prefer-active-python` makes it use whatever `python3` resolves to in the current shell.

For multiple Python versions:

```bash
poetry env use 3.11           # create/select venv with Python 3.11
poetry env use 3.12           # switch to (or create) one with 3.12
poetry env list               # show all envs for this project
poetry env remove 3.11        # delete the 3.11 env
```

Each `env use <version>` either creates a new venv or selects an existing matching one. The current selection is stored in a small file in the cache; `poetry install` and `poetry run` use that selection.

## Compared to uv

Both Poetry and uv solve the same problem: resolve, lock, and install Python dependencies. The differences:

**Algorithm:** Both use PubGrub-style CDCL resolution. They produce semantically-equivalent results in nearly all cases.

**Implementation language:** Poetry is pure Python; uv is Rust. uv is 10-100× faster for typical workflows. The gap widens with project size — a 500-package project that takes Poetry 30 seconds to lock takes uv under 1 second.

**Scope:** uv's design philosophy is "replace pip + pip-tools + pipx + virtualenv + pyenv + poetry". A single binary subsumes the whole stack. Poetry's scope is narrower: it doesn't manage Python interpreters (you use pyenv/asdf), doesn't replace pip for non-Poetry projects, and is itself a Python package (bootstrap problem).

**Lockfile format:** Both use TOML. Both record full file hashes and per-package metadata. uv's `uv.lock` is aligned with the emerging PEP 751 standard (universal lockfile for Python); Poetry's `poetry.lock` predates PEP 751 and uses its own schema.

**Workspaces:** uv has first-class workspace support (`[tool.uv.workspace]`) modeled after Cargo. Poetry has had workspace-like features through path dependencies and `[tool.poetry.dependencies] mylib = { path = "../mylib" }`, but no integrated workspace lock.

**Plugin ecosystem:** Poetry has a mature plugin system and many third-party plugins. uv is newer and currently has no plugin API — features are either built-in or out-of-scope.

**Build backend:** Poetry ships `poetry-core`, a PEP 517 backend, that many projects use even when not using the Poetry CLI. uv doesn't ship a build backend; uv-based projects typically use hatchling, setuptools, or poetry-core as their backend.

**Python install:** uv can install Python interpreters (`uv python install 3.12.1`) using Astral's `python-build-standalone` distribution. Poetry assumes Python is already installed.

**Migration:** `uv pip install -r requirements.txt` works; uv reads pyproject.toml from any Poetry project (resolves the same `[tool.poetry.dependencies]`, though it ignores Poetry-specific fields like groups). Poetry has no native uv migration path; you'd manually rewrite uv.lock to poetry.lock or vice versa.

**The honest take:** for a new project in 2025+, uv is the recommended choice — faster, simpler, more aligned with emerging standards. Poetry remains an excellent choice for existing projects that depend on its plugins or its mature build backend. Many large open-source projects (FastAPI, Pydantic, Black) still use Poetry for the build pipeline and may continue to do so for years.

## Performance Internals

Where does Poetry spend its time? A typical `poetry install` from cold cache breaks down as:

1. **Metadata fetch** (40-70%) — querying PyPI's simple API for each package, downloading per-version JSON metadata. Each request is a TLS handshake + HTTP roundtrip; Poetry parallelizes via a thread pool, but Python's GIL bottlenecks the CPU-bound TLS work.
2. **Resolution** (10-30%) — Mixology's CDCL loop. Pure Python, ~1ms per propagation step; large projects may execute hundreds of thousands of steps.
3. **Wheel download** (10-20%) — fetching `.whl` files. Bottlenecked by network bandwidth.
4. **Install** (5-10%) — unzipping wheels, writing files, running post-install scripts (rarely).

uv's win is largely in (1) and (2): parallel metadata fetch with no GIL, and a 10-100× faster resolver. Wheel download (3) is bandwidth-bound and benefits less.

For repeated installs from a warm cache, Poetry skips (1) and (3) entirely — but (2) and (4) still run. uv's warm install can be sub-second; Poetry's is several seconds.

Optimizations Poetry has shipped over time:

- **Parallel metadata fetch** (1.0+): originally serial; now uses a thread pool.
- **Resolver caching** (1.2+): reuses partial-solution data across re-runs.
- **Lockfile content-hash** (always): skip re-resolving if pyproject didn't change.
- **Stripped metadata in lockfile** (2.0+): reduces lockfile size and parse time.

Optimizations Poetry could ship but hasn't:

- **Streaming wheel install**: pip and uv unpack wheels in parallel; Poetry serial-unpacks.
- **PEP 658 metadata-only fetch**: skip downloading wheel ZIPs just for metadata. Poetry uses this where supported but the implementation is conservative.
- **Native code in hot loops**: porting the resolver inner loop to a C extension or Rust extension would close most of the perf gap with uv. The Poetry team has discussed this; no concrete roadmap.

## Caching

Poetry maintains several caches:

**Repository cache** — `~/.cache/pypoetry/cache/repositories/<repo>/`. Per-source cached metadata. Subdirectories per package; files contain JSON metadata. Cleared with `poetry cache clear --all <repo>`.

**Wheel cache** — `~/.cache/pypoetry/artifacts/`. Downloaded wheel files. Reused across projects; install from cache is much faster than re-download.

**Virtualenv cache** — `~/.cache/pypoetry/virtualenvs/<project-hash>/` (when not in-project). The actual installed Python environments.

Cache eviction is manual (`poetry cache clear`). Poetry doesn't auto-prune like uv does. Long-running developers may accumulate gigabytes of cached wheels for old projects; a `poetry cache clear --all PyPI` periodically frees space.

## Lockfile drift detection

A common workflow bug: developer adds a dependency in pyproject but forgets to commit the updated lockfile. CI then runs `poetry install` against the committed lockfile, which doesn't include the new dependency, and the test passes — even though the project is broken in the sense that the lockfile and pyproject disagree.

Poetry's content-hash detects this:

```bash
$ poetry install
Installing dependencies from lock file

pyproject.toml changed significantly since poetry.lock was last generated. Run `poetry lock` to fix the lock file.
```

Modern CI configurations should add `poetry check --lock` to validate the lock matches pyproject *before* running tests. This catches the bug locally instead of in PR review.

`poetry lock --no-update` regenerates the lockfile to match pyproject's content-hash *without* re-resolving versions. This is useful when you've made a non-semantic edit to pyproject (e.g. added a comment in a dependency declaration) but don't want to bump versions.

## Constraint syntax cheat sheet

Poetry inherits and extends PEP 440 syntax:

| Syntax | Meaning | Equivalent PEP 440 |
|--------|---------|-------------------|
| `^1.2.3` | Compatible release | `>=1.2.3,<2.0.0` |
| `^0.2.3` | Pre-1.0 caret (different!) | `>=0.2.3,<0.3.0` |
| `^0.0.3` | Pre-0.1 caret | `>=0.0.3,<0.0.4` |
| `~1.2.3` | Tilde | `>=1.2.3,<1.3.0` |
| `~1.2` | Tilde, less specific | `>=1.2,<1.3` |
| `1.2.3` | Exact (caret is implied? no, exact) | `==1.2.3` |
| `==1.2.3` | Exact | `==1.2.3` |
| `>=1.2.3,<2.0.0` | Range | (same) |
| `*` | Any version | (any) |
| `>=1.2.3 || <1.0.0` | Disjunction | (no PEP 440 equiv) |

The caret operator is the one most likely to surprise. For Python, `^X.Y.Z` allows up to but not including the next *major* version when `X >= 1`, but for `X = 0` it constrains differently — pre-1.0 versions don't have the same compatibility guarantees in semver. Poetry follows the npm-style interpretation.

For testing constraint expansions:

```bash
poetry show django --tree         # see resolved version
poetry show -v django             # verbose: show why this version
```

## Lockfile portability between Poetry versions

A poetry.lock file generated by Poetry 1.5 is readable by Poetry 1.4, *most of the time*. The lock-version field signals the schema; older Poetry versions may not understand newer schema features and refuse to install.

Best practice: pin Poetry version in CI via `poetry self update --version X.Y.Z` or use a Poetry-installer Docker image with a fixed version. Otherwise, a developer who upgrades Poetry locally regenerates the lock in a newer schema, and CI (still on the older Poetry) fails to read it.

For team consistency, some projects commit a `tool-versions` file or use `mise`/`asdf` to pin Poetry alongside Python.

## Common errors and fixes

**`Because no versions of X match …`** — the resolver couldn't find any version satisfying all constraints. Run `poetry add X --dry-run` to see why; often it's an indirect conflict between two transitive dependencies.

**`SolverProblemError`** — Poetry's wrapped error for unsatisfiable constraints. The error message lists the chain of incompatibilities — read it bottom-up to understand the conflict.

**`The current project's Python requirement is incompatible with some of the required packages Python requirement`** — your `python = "..."` is too narrow (or too wide) for some dependency. Either widen the range or pin the dependency to a version compatible with your Python range.

**`Package not found in repository`** — a dependency isn't on PyPI but is expected to be on a private repo. Check `[tool.poetry.source]` configuration and credentials.

**`HashMismatch`** — the wheel/sdist downloaded doesn't match the lockfile's hash. Either the package was re-published (PyPI prevents this for the same version, but private indexes may not), or your network/cache is corrupt. Run `poetry cache clear --all PyPI` and retry.

**`No matching distribution found for X (from Y)`** — none of the wheels for X match the current Python interpreter, and no sdist exists. Either install missing build tools (gcc, etc.) so the sdist can build, or upgrade Python.

## Going deeper

Poetry's source code is a great read for anyone interested in package-management internals. Key directories:

- `src/poetry/core/packages/` — the Package model, version specifiers, dependency types.
- `src/poetry/mixology/` — the resolver. Read `version_solver.py` first.
- `src/poetry/repositories/` — source backends (PyPI, simple-index, git, path).
- `src/poetry/installation/` — the executor that downloads and installs.
- `src/poetry/utils/env/` — virtualenv management.

The Mixology implementation is annotated with references to the PubGrub paper and Dart's reference implementation, so you can compare side-by-side.

For the CDCL theory beneath everything: read the SAT-solver chapter in *Handbook of Satisfiability* (Biere et al., 2009). The package-resolution adaptation is a relatively thin layer over standard CDCL.

## References

- Poetry documentation — https://python-poetry.org/docs/
- Poetry source — https://github.com/python-poetry/poetry
- poetry-core source — https://github.com/python-poetry/poetry-core
- PEP 517 — A build-system independent format for source trees — https://peps.python.org/pep-0517/
- PEP 518 — Specifying minimum build system requirements — https://peps.python.org/pep-0518/
- PEP 621 — Storing project metadata in pyproject.toml — https://peps.python.org/pep-0621/
- PEP 631 — Dependency specification in pyproject.toml using PEP 508 — https://peps.python.org/pep-0631/
- PEP 503 — Simple Repository API — https://peps.python.org/pep-0503/
- PEP 691 — JSON-based Simple API — https://peps.python.org/pep-0691/
- PEP 735 — Dependency groups — https://peps.python.org/pep-0735/
- PEP 751 — Universal lockfile (draft, in progress)
- PEP 440 — Version specification — https://peps.python.org/pep-0440/
- PubGrub paper — Natalie Weizenbaum, "PubGrub: Next-Generation Version Solving" — https://medium.com/@nex3/pubgrub-2fb6470504f
- Mixology source — https://github.com/python-poetry/poetry/tree/master/src/poetry/mixology
- Bundler's Molinillo — https://github.com/CocoaPods/Molinillo
- Dart pub source — https://github.com/dart-lang/pub
- Handbook of Satisfiability — Biere, Heule, van Maaren, Walsh (eds.), IOS Press, 2009
- The PubGrub algorithm explained — https://github.com/dart-lang/pub/blob/master/doc/solver.md
- Python Packaging User Guide — https://packaging.python.org/
- pyproject.toml specification — https://packaging.python.org/en/latest/specifications/pyproject-toml/
- Wheel format (PEP 427) — https://peps.python.org/pep-0427/
- Source distribution format (PEP 517 / PEP 643) — https://peps.python.org/pep-0643/
- Poetry plugins guide — https://python-poetry.org/docs/plugins/
- Cleo (Poetry's CLI framework) — https://github.com/python-poetry/cleo
- TOML 1.0.0 specification — https://toml.io/en/v1.0.0
- Keyring library — https://github.com/jaraco/keyring
- Comparison: Poetry vs uv — https://docs.astral.sh/uv/pip/compatibility/
