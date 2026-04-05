# Network CI/CD (Continuous Integration/Delivery for Network Infrastructure)

Automated pipelines that lint, validate, test, and deploy network configuration changes using GitOps principles.

## Pipeline Overview

### Minimal network CI/CD pipeline

```bash
# .gitlab-ci.yml / GitHub Actions equivalent stages:
# 1. lint        — YAML/Python syntax, style checks
# 2. validate    — config correctness (Batfish, offline checks)
# 3. test        — lab deployment + verification
# 4. deploy      — production push (canary → full)
# 5. verify      — post-deploy validation
# 6. rollback    — automatic if verify fails
```

## Pre-Commit Hooks

### Install pre-commit

```bash
pip install pre-commit
```

### .pre-commit-config.yaml for network repos

```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-json

  - repo: https://github.com/adrienverge/yamllint
    rev: v1.33.0
    hooks:
      - id: yamllint
        args: [-c, .yamllint.yml]

  - repo: https://github.com/psf/black
    rev: 24.3.0
    hooks:
      - id: black

  - repo: https://github.com/PyCQA/pylint
    rev: v3.1.0
    hooks:
      - id: pylint
        args: [--disable=C0114,C0115,C0116]

  - repo: local
    hooks:
      - id: validate-inventory
        name: Validate Nornir Inventory
        entry: python scripts/validate_inventory.py
        language: python
        files: 'inventory/.*\.yaml$'
```

### Install and run

```bash
pre-commit install
pre-commit run --all-files
```

## Config Validation with Batfish

### Start Batfish container

```bash
docker run -d --name batfish \
  -p 9997:9997 -p 9996:9996 \
  -v $(pwd)/configs:/configs \
  batfish/allinone
```

### Validate with pybatfish

```python
from pybatfish.client.session import Session

bf = Session(host="localhost")
bf.set_network("my_network")
bf.init_snapshot("configs/", name="candidate")

# Check for undefined references
undef = bf.q.undefinedReferences().answer().frame()
assert undef.empty, f"Undefined references found:\n{undef}"

# Check for unused structures
unused = bf.q.unusedStructures().answer().frame()

# Verify ACL behavior
acl_reach = bf.q.searchFilters(
    headers={"srcIps": "10.0.0.0/8", "dstIps": "192.168.1.0/24"},
    action="permit",
).answer().frame()

# Traceroute validation
traceroute = bf.q.traceroute(
    startLocation="spine1",
    headers={"dstIps": "10.0.0.11"},
).answer().frame()
assert len(traceroute) > 0, "No path found"

# Detect routing loops
loops = bf.q.detectLoops().answer().frame()
assert loops.empty, f"Routing loops detected:\n{loops}"
```

### Batfish in CI pipeline

```yaml
# .github/actions/batfish-validate/action.yml
name: Batfish Validation
runs:
  using: composite
  steps:
    - run: |
        docker run -d --name batfish -p 9997:9997 batfish/allinone
        sleep 10
        python scripts/batfish_validate.py
        docker stop batfish
```

## Config Validation with pyATS

### Install pyATS

```bash
pip install "pyats[full]"
```

### Testbed file (testbed.yaml)

```yaml
devices:
  spine1:
    os: iosxe
    type: router
    connections:
      cli:
        protocol: ssh
        ip: 10.0.0.1
        port: 22
    credentials:
      default:
        username: admin
        password: "%ENV{DEVICE_PASSWORD}"
```

### pyATS validation script

```python
from genie.testbed import load
from genie.utils.diff import Diff

testbed = load("testbed.yaml")
device = testbed.devices["spine1"]
device.connect(log_stdout=False)

# Learn BGP state
bgp = device.learn("bgp")

# Verify neighbors are established
for neighbor, data in bgp.info["instance"]["default"]["vrf"]["default"]["neighbor"].items():
    state = data["session_state"]
    assert state == "established", f"BGP neighbor {neighbor} is {state}"

# Config diff
pre = device.execute("show running-config")
# ... apply changes ...
post = device.execute("show running-config")
diff = Diff(pre, post)
diff.findDiff()
print(diff)
```

## GitHub Actions Pipeline

### Complete network CI/CD workflow

```yaml
# .github/workflows/network-cicd.yml
name: Network CI/CD

on:
  pull_request:
    paths:
      - 'configs/**'
      - 'inventory/**'
      - 'templates/**'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - run: pip install yamllint pylint black
      - run: yamllint -c .yamllint.yml configs/ inventory/
      - run: black --check scripts/
      - run: pylint scripts/ --disable=C0114

  validate:
    runs-on: ubuntu-latest
    needs: lint
    services:
      batfish:
        image: batfish/allinone
        ports:
          - 9997:9997
    steps:
      - uses: actions/checkout@v4
      - run: pip install pybatfish
      - run: python scripts/batfish_validate.py

  test-lab:
    runs-on: self-hosted
    needs: validate
    steps:
      - uses: actions/checkout@v4
      - run: pip install nornir nornir_netmiko nornir_utils
      - run: |
          python scripts/deploy.py --env lab --dry-run
          python scripts/deploy.py --env lab
          python scripts/verify.py --env lab

  deploy-canary:
    runs-on: self-hosted
    needs: test-lab
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    environment: canary
    steps:
      - uses: actions/checkout@v4
      - run: |
          python scripts/deploy.py --env prod --filter role=canary --dry-run
          python scripts/deploy.py --env prod --filter role=canary
          python scripts/verify.py --env prod --filter role=canary

  deploy-prod:
    runs-on: self-hosted
    needs: deploy-canary
    environment: production
    steps:
      - uses: actions/checkout@v4
      - run: |
          python scripts/deploy.py --env prod --dry-run
          python scripts/deploy.py --env prod
          python scripts/verify.py --env prod
```

## GitLab CI Pipeline

### .gitlab-ci.yml for network automation

```yaml
stages:
  - lint
  - validate
  - test
  - deploy
  - verify

variables:
  PIP_CACHE_DIR: "$CI_PROJECT_DIR/.pip-cache"

lint:
  stage: lint
  image: python:3.12
  script:
    - pip install yamllint pylint
    - yamllint -c .yamllint.yml configs/ inventory/
    - pylint scripts/

validate:
  stage: validate
  image: python:3.12
  services:
    - name: batfish/allinone
      alias: batfish
  script:
    - pip install pybatfish
    - python scripts/batfish_validate.py

test-lab:
  stage: test
  tags: [network-runner]
  script:
    - python scripts/deploy.py --env lab
    - python scripts/verify.py --env lab

deploy-canary:
  stage: deploy
  tags: [network-runner]
  script:
    - python scripts/deploy.py --env prod --filter role=canary
  when: manual
  only:
    - main

deploy-prod:
  stage: deploy
  tags: [network-runner]
  script:
    - python scripts/deploy.py --env prod
  needs: [deploy-canary]
  when: manual
  only:
    - main

verify:
  stage: verify
  tags: [network-runner]
  script:
    - python scripts/verify.py --env prod
  needs: [deploy-prod]
```

## Config Backup and Diff

### Automated backup with Nornir

```python
from nornir import InitNornir
from nornir_netmiko.tasks import netmiko_send_command
from nornir_utils.plugins.tasks.files import write_file
from datetime import datetime
import os

nr = InitNornir(config_file="config.yaml")

def backup_config(task):
    date = datetime.now().strftime("%Y%m%d")
    backup_dir = f"backups/{date}"
    os.makedirs(backup_dir, exist_ok=True)
    r = task.run(
        task=netmiko_send_command,
        command_string="show running-config",
    )
    task.run(
        task=write_file,
        filename=f"{backup_dir}/{task.host.name}.cfg",
        content=r.result,
    )

nr.run(task=backup_config)
```

### Git-based config diff

```bash
# Store configs in git, diff on each run
cd backups/
git add -A
git diff --cached --stat
git diff --cached -- spine1.cfg
git commit -m "Config backup $(date +%Y-%m-%d)"
```

### Diff with difflib

```python
import difflib

def diff_configs(old_file, new_file):
    with open(old_file) as f:
        old = f.readlines()
    with open(new_file) as f:
        new = f.readlines()
    diff = difflib.unified_diff(old, new, fromfile="before", tofile="after")
    return "".join(diff)
```

## Nornir in Pipelines

### Deploy script pattern (scripts/deploy.py)

```python
import argparse
from nornir import InitNornir
from nornir_netmiko.tasks import netmiko_send_config
from nornir_jinja2.plugins.tasks import template_file
from nornir_utils.plugins.functions import print_result

def deploy(task, dry_run=False):
    r = task.run(
        task=template_file,
        template=f"{task.host.name}.j2",
        path="templates/",
        **task.host.items(),
    )
    if dry_run:
        print(f"[DRY RUN] {task.host.name}:\n{r.result}")
        return
    task.run(
        task=netmiko_send_config,
        config_commands=r.result.splitlines(),
    )

parser = argparse.ArgumentParser()
parser.add_argument("--env", required=True, choices=["lab", "prod"])
parser.add_argument("--dry-run", action="store_true")
parser.add_argument("--filter", help="key=value filter")
args = parser.parse_args()

nr = InitNornir(config_file=f"config_{args.env}.yaml")

if args.filter:
    key, value = args.filter.split("=")
    nr = nr.filter(**{key: value})

result = nr.run(task=deploy, dry_run=args.dry_run)
print_result(result)

if result.failed:
    raise SystemExit(1)
```

## Ansible in Pipelines

### Ansible network playbook in CI

```yaml
# .github/workflows/ansible-network.yml
name: Ansible Network Deploy

on:
  push:
    branches: [main]
    paths: ['ansible/**']

jobs:
  deploy:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
      - run: pip install ansible ansible-pylibssh netaddr
      - run: ansible-lint ansible/
      - run: |
          ansible-playbook ansible/site.yml \
            --check --diff \
            -i ansible/inventory/production.yml
      - run: |
          ansible-playbook ansible/site.yml \
            -i ansible/inventory/production.yml
```

## Rollback Automation

### Automatic rollback on failure

```python
def deploy_with_rollback(task):
    # Step 1: Backup
    backup = task.run(
        task=netmiko_send_command,
        command_string="show running-config",
    )
    task.host["backup_config"] = backup.result

    # Step 2: Deploy
    task.run(
        task=netmiko_send_config,
        config_commands=task.host["desired_config"].splitlines(),
    )

    # Step 3: Verify
    try:
        verify = task.run(
            task=netmiko_send_command,
            command_string="show ip bgp summary",
        )
        if "Established" not in verify.result:
            raise Exception("BGP neighbors not established")
    except Exception:
        # Step 4: Rollback
        task.run(
            task=netmiko_send_config,
            config_commands=task.host["backup_config"].splitlines(),
        )
        raise
```

### Config replace rollback (IOS-XE)

```python
def safe_deploy_iosxe(task):
    # Set rollback timer
    task.run(
        task=netmiko_send_command,
        command_string="configure replace flash:rollback.cfg timer 5",
    )
    # Apply config
    task.run(
        task=netmiko_send_config,
        config_commands=task.host["desired_config"].splitlines(),
    )
    # Verify
    ok = verify_device(task)
    if ok:
        # Confirm (cancel rollback timer)
        task.run(
            task=netmiko_send_command,
            command_string="configure confirm",
        )
    # If not confirmed, device auto-rolls back after 5 minutes
```

## Source of Truth Integration

### NetBox as source of truth

```python
import pynetbox

nb = pynetbox.api("https://netbox.example.com", token="your-token")

# Get all devices
devices = nb.dcim.devices.filter(site="dc1", role="spine")

# Generate Nornir inventory from NetBox
hosts = {}
for device in devices:
    hosts[device.name] = {
        "hostname": str(device.primary_ip4).split("/")[0],
        "platform": device.platform.slug,
        "data": {
            "site": device.site.slug,
            "role": device.device_role.slug,
            "serial": device.serial,
        },
    }
```

### Nautobot inventory plugin

```python
# pip install nornir_nautobot
nr = InitNornir(
    inventory={
        "plugin": "NautobotInventory",
        "options": {
            "nautobot_url": "https://nautobot.example.com",
            "nautobot_token": "your-token",
            "filter_parameters": {"site": "dc1"},
        },
    },
)
```

## See Also

- Nornir
- Ansible
- NAPALM
- GitHub Actions
- GitLab CI
- Batfish

## References

- Batfish: https://www.batfish.org/
- pyATS: https://developer.cisco.com/pyats/
- NetBox: https://netbox.dev/
- Nautobot: https://www.networktocode.com/nautobot/
- "Network Programmability and Automation" — O'Reilly
