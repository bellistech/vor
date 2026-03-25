# pip (Python Package Installer)

> Install and manage Python packages from PyPI and other indexes.

## Virtual Environments

### Creating and Using

```bash
python3 -m venv .venv                  # create virtual environment
source .venv/bin/activate              # activate (bash/zsh)
source .venv/bin/activate.fish         # activate (fish)
.venv/bin/activate.ps1                 # activate (PowerShell)
deactivate                             # deactivate

# verify active environment
which python                           # should point to .venv/bin/python
python -m site --user-site             # show site-packages path
```

## Install

```bash
pip install requests                   # install latest version
pip install requests==2.31.0           # exact version
pip install "requests>=2.28,<3.0"      # version range
pip install requests~=2.31.0           # compatible release (>=2.31.0, <2.32.0)
pip install ./local-package/           # install from local directory
pip install ./package.whl              # install wheel file
pip install ./package.tar.gz           # install source distribution

pip install -e .                       # editable install (development mode)
pip install -e ".[dev,test]"           # editable with extras

pip install --user requests            # install to user directory (no venv)
pip install --pre requests             # include pre-release versions
pip install --no-deps requests         # skip dependencies
pip install --force-reinstall requests # reinstall even if up to date
pip install --no-cache-dir requests    # skip cache
```

### From Requirements File

```bash
pip install -r requirements.txt
pip install -r requirements.txt -r requirements-dev.txt

# with constraints
pip install -c constraints.txt -r requirements.txt
```

### From Other Sources

```bash
pip install git+https://github.com/user/repo.git
pip install git+https://github.com/user/repo.git@v2.0    # specific tag
pip install git+https://github.com/user/repo.git@main    # specific branch
pip install --index-url https://pypi.example.com/simple/ private-pkg
pip install --extra-index-url https://pypi.example.com/simple/ private-pkg
```

## Uninstall

```bash
pip uninstall requests                 # uninstall package
pip uninstall -y requests flask        # uninstall multiple, skip confirmation
pip uninstall -r requirements.txt      # uninstall everything in file
```

## Freeze and Requirements

```bash
pip freeze                             # list installed packages (pip format)
pip freeze > requirements.txt          # save to file
pip freeze --exclude-editable          # skip editable installs

# requirements.txt format
requests==2.31.0
flask>=3.0,<4.0
gunicorn~=21.2
-e ./my-local-package                  # editable local package
```

## Upgrade

```bash
pip install --upgrade requests         # upgrade to latest
pip install --upgrade pip              # upgrade pip itself
pip install --upgrade pip setuptools wheel  # upgrade build tools

# upgrade all packages (no built-in command, use this pattern)
pip list --outdated --format=columns
pip list --outdated --format=json | python3 -c "
import json, sys
for p in json.load(sys.stdin):
    print(p['name'])
" | xargs -n1 pip install --upgrade
```

## Show and List

```bash
pip show requests                      # package metadata
pip show -f requests                   # metadata + installed files
pip list                               # list all installed packages
pip list --outdated                     # packages with newer versions
pip list --not-required                 # packages not depended on by others
pip list --format=json                 # JSON output
pip list --editable                    # list editable installs
```

## Cache

```bash
pip cache info                         # cache location and size
pip cache list                         # list cached packages
pip cache list requests                # cached versions of specific package
pip cache purge                        # clear entire cache
pip cache remove requests              # remove specific package from cache
```

## Search and Check

```bash
pip check                              # verify dependency compatibility
pip debug --verbose                    # show pip configuration
pip config list                        # show pip config values
pip config set global.index-url https://pypi.example.com/simple/
```

## pip.conf (Configuration)

```bash
# location:
# Linux:   ~/.config/pip/pip.conf
# macOS:   ~/Library/Application Support/pip/pip.conf
# Per-project: ./pip.conf (in venv root)

# example pip.conf
[global]
index-url = https://pypi.org/simple/
timeout = 30
trusted-host = pypi.example.com

[install]
no-cache-dir = false
find-links = /local/wheels/
```

## Building and Distributing

```bash
pip install build twine                # install build tools
python -m build                        # build sdist + wheel
twine check dist/*                     # validate built packages
twine upload dist/*                    # upload to PyPI
twine upload --repository testpypi dist/*  # upload to test PyPI
```

## Tips

- Always use virtual environments. Never `pip install` into the system Python.
- `pip install -e ".[dev]"` for development installs -- changes to source are reflected immediately.
- `pip freeze` includes transitive dependencies. For cleaner requirements, use `pip-compile` from pip-tools.
- `pip check` catches dependency conflicts. Run it after installing to verify consistency.
- `--no-cache-dir` is useful in Docker to reduce image size.
- `pip install --upgrade pip` should be the first command in a fresh venv.
- Use `constraints.txt` to pin transitive dependencies without declaring them as direct requirements.
- `pip list --not-required` shows top-level packages -- useful for auditing what you actually depend on.
- Environment markers in requirements: `pywin32; sys_platform == "win32"` for platform-specific deps.
- Prefer `python3 -m pip` over bare `pip` to ensure you are using the right Python interpreter.
