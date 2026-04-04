# pre-commit (Git Hook Framework)

Language-agnostic framework for managing and running git pre-commit hooks with automatic environment isolation.

## Installation

### Installing pre-commit

```bash
# pip
pip install pre-commit

# Homebrew
brew install pre-commit

# Conda
conda install -c conda-forge pre-commit

# pipx (isolated install)
pipx install pre-commit

# Verify
pre-commit --version
```

### Setting up hooks

```bash
# Install hooks defined in .pre-commit-config.yaml
pre-commit install                      # install pre-commit hook
pre-commit install --hook-type commit-msg  # install commit-msg hook
pre-commit install --hook-type pre-push    # install pre-push hook
pre-commit install --install-hooks      # install and download hook envs

# Uninstall
pre-commit uninstall                    # remove pre-commit hook
pre-commit uninstall --hook-type commit-msg
```

## Configuration

### Basic .pre-commit-config.yaml

```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-json
      - id: check-added-large-files
        args: ['--maxkb=500']
      - id: check-merge-conflict
      - id: detect-private-key
      - id: no-commit-to-branch
        args: ['--branch', 'main', '--branch', 'master']

  - repo: https://github.com/psf/black
    rev: 24.3.0
    hooks:
      - id: black
        language_version: python3.12

  - repo: https://github.com/pre-commit/mirrors-eslint
    rev: v9.0.0
    hooks:
      - id: eslint
        types: [javascript, tsx, typescript]
        additional_dependencies:
          - eslint@9.0.0
          - eslint-config-prettier@9.1.0
```

### Hook configuration options

```yaml
repos:
  - repo: https://github.com/example/hooks
    rev: v1.0.0
    hooks:
      - id: my-hook
        name: "My Custom Hook"          # display name
        files: '^src/.*\.py$'           # only match these files (regex)
        exclude: '^src/vendor/'         # skip these files (regex)
        types: [python]                 # file types to check
        types_or: [python, pyi]         # any of these types
        stages: [commit]                # when to run (commit, push, etc.)
        language: python                # hook language runtime
        entry: my-linter                # command to run
        args: ['--strict', '--config=.config.yaml']
        pass_filenames: true            # pass matched files as args
        always_run: false               # run even if no matching files
        verbose: false                  # show output even on success
        require_serial: false           # don't parallelize
```

## Running Hooks

### Manual execution

```bash
pre-commit run                          # run on staged files
pre-commit run --all-files              # run on all files
pre-commit run trailing-whitespace      # run specific hook
pre-commit run --files src/main.py      # run on specific files
pre-commit run --from-ref HEAD~1 --to-ref HEAD  # run on changed files

# Skip specific hooks during commit
SKIP=eslint,black git commit -m "wip"

# Skip all hooks
git commit --no-verify -m "emergency fix"
```

### Hook environments

```bash
pre-commit clean                        # remove cached hook environments
pre-commit gc                           # garbage collect unused environments
pre-commit autoupdate                   # update all hooks to latest rev
pre-commit autoupdate --repo https://github.com/psf/black  # update specific
pre-commit autoupdate --freeze          # pin to exact commit hash
```

## Common Hook Repositories

### pre-commit-hooks (official)

```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      # Whitespace and formatting
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: mixed-line-ending
        args: ['--fix=lf']

      # File checks
      - id: check-yaml
      - id: check-json
      - id: check-toml
      - id: check-xml
      - id: check-added-large-files
      - id: check-case-conflict
      - id: check-symlinks

      # Security
      - id: detect-private-key
      - id: detect-aws-credentials

      # Git
      - id: check-merge-conflict
      - id: no-commit-to-branch
        args: ['--branch', 'main']

      # Python
      - id: check-ast
      - id: debug-statements
      - id: requirements-txt-fixer
```

### Language-specific hooks

```yaml
repos:
  # Python
  - repo: https://github.com/psf/black
    rev: 24.3.0
    hooks:
      - id: black

  - repo: https://github.com/astral-sh/ruff-pre-commit
    rev: v0.3.4
    hooks:
      - id: ruff
        args: ['--fix']
      - id: ruff-format

  - repo: https://github.com/pre-commit/mirrors-mypy
    rev: v1.9.0
    hooks:
      - id: mypy
        additional_dependencies: [types-requests]

  # JavaScript/TypeScript
  - repo: https://github.com/pre-commit/mirrors-prettier
    rev: v4.0.0-alpha.8
    hooks:
      - id: prettier
        types_or: [javascript, typescript, css, json, markdown]

  # Go
  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
      - id: go-vet
      - id: go-build

  # Shell
  - repo: https://github.com/koalaman/shellcheck-precommit
    rev: v0.10.0
    hooks:
      - id: shellcheck

  # Docker
  - repo: https://github.com/hadolint/hadolint
    rev: v2.12.0
    hooks:
      - id: hadolint
```

## Custom Hooks

### Local hooks

```yaml
repos:
  - repo: local
    hooks:
      - id: go-test
        name: Go Tests
        entry: go test ./... -count=1 -race
        language: system
        pass_filenames: false
        types: [go]

      - id: check-todos
        name: Check for TODO comments
        entry: grep -rn "TODO\|FIXME\|HACK" --include="*.py"
        language: system
        pass_filenames: false
        always_run: true

      - id: validate-schema
        name: Validate JSON Schema
        entry: python scripts/validate_schema.py
        language: python
        files: '^config/.*\.json$'
        additional_dependencies: [jsonschema==4.21.1]
```

### Script hooks

```yaml
repos:
  - repo: local
    hooks:
      - id: custom-linter
        name: Custom Linter
        entry: scripts/lint.sh
        language: script
        files: '\.py$'
```

## Stages

### Hook stages

```yaml
repos:
  - repo: local
    hooks:
      # Run during commit
      - id: lint
        stages: [commit]
        entry: make lint
        language: system
        pass_filenames: false

      # Run during push
      - id: test
        stages: [push]
        entry: make test
        language: system
        pass_filenames: false

      # Run during commit message validation
      - id: commitlint
        stages: [commit-msg]
        entry: npx commitlint --edit
        language: system
        pass_filenames: false

      # Run manually only
      - id: full-suite
        stages: [manual]
        entry: make test-all
        language: system
        pass_filenames: false
```

```bash
# Install hooks for specific stages
pre-commit install --hook-type pre-commit
pre-commit install --hook-type pre-push
pre-commit install --hook-type commit-msg

# Run manual stage hooks
pre-commit run --hook-stage manual --all-files
```

## CI Integration

### pre-commit.ci (hosted service)

```yaml
# .pre-commit-config.yaml
ci:
  autofix_prs: true                     # auto-fix and push
  autofix_commit_msg: 'style: auto-fix pre-commit hooks'
  autoupdate_schedule: weekly           # weekly, monthly, quarterly
  autoupdate_commit_msg: 'chore: pre-commit autoupdate'
  skip: [go-test, mypy]                # skip slow hooks in CI
  submodules: false
```

### GitHub Actions

```yaml
# .github/workflows/pre-commit.yml
jobs:
  pre-commit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - uses: pre-commit/action@v3.0.1
        with:
          extra_args: --all-files
```

### GitLab CI

```yaml
pre-commit:
  image: python:3.12
  variables:
    PRE_COMMIT_HOME: ${CI_PROJECT_DIR}/.cache/pre-commit
  cache:
    paths:
      - .cache/pre-commit
  script:
    - pip install pre-commit
    - pre-commit run --all-files
```

## Tips

- Run `pre-commit autoupdate` monthly to get latest hook versions and security fixes
- Use `pre-commit run --all-files` in CI to catch issues on files that weren't staged
- Put slow hooks (tests, type checking) in the `push` stage, fast hooks (formatting) in `commit`
- Use `SKIP=hook-id git commit` to bypass specific hooks during development emergencies
- Cache `.cache/pre-commit` in CI to avoid re-downloading hook environments on every run
- Use `additional_dependencies` to pin transitive dependencies for reproducible hook behavior
- Use `types` or `files` to limit hooks to relevant files and avoid unnecessary runs
- Use `repo: local` hooks for project-specific checks that don't exist as published hooks
- Set `pass_filenames: false` for hooks that should run once (not per-file), like `make test`
- Use `no-commit-to-branch` to prevent accidental commits directly to main/master
- Use `pre-commit.ci` for automatic autoupdate PRs and hook running without CI config
- Use `--freeze` with `autoupdate` to pin to exact commit SHAs for maximum reproducibility

## See Also

- shellcheck
- hadolint
- black
- eslint
- prettier

## References

- [pre-commit Official Documentation](https://pre-commit.com/)
- [pre-commit Supported Hooks](https://pre-commit.com/hooks.html)
- [pre-commit-hooks Repository](https://github.com/pre-commit/pre-commit-hooks)
- [pre-commit.ci Documentation](https://pre-commit.ci/)
- [pre-commit GitHub Action](https://github.com/pre-commit/action)
