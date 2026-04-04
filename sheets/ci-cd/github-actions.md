# GitHub Actions (CI/CD Workflow Automation)

Automate build, test, and deploy pipelines directly in your GitHub repository with YAML workflows.

## Workflow Basics

### Minimal workflow

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: echo "Hello from CI"
```

### Trigger events

```yaml
on:
  push:
    branches: [main, develop]
    paths: ['src/**', '*.go']        # only run when these paths change
    tags: ['v*']                      # trigger on version tags
  pull_request:
    branches: [main]
    types: [opened, synchronize, reopened]
  schedule:
    - cron: '0 6 * * 1'              # every Monday at 06:00 UTC
  workflow_dispatch:                   # manual trigger from UI
    inputs:
      environment:
        description: 'Target environment'
        required: true
        default: 'staging'
        type: choice
        options: [staging, production]
  release:
    types: [published]
```

## Jobs and Steps

### Job structure

```yaml
jobs:
  test:
    runs-on: ubuntu-latest           # runner OS
    timeout-minutes: 15               # kill job if it hangs
    steps:
      - uses: actions/checkout@v4     # use an action
      - name: Run tests
        run: go test ./...            # run a shell command
        working-directory: ./backend  # optional working dir
        env:
          CGO_ENABLED: "0"            # step-level env var

  deploy:
    needs: test                       # run after test job
    runs-on: ubuntu-latest
    steps:
      - run: echo "deploying"
```

### Multi-line commands

```yaml
steps:
  - name: Build and push
    run: |
      docker build -t myapp .
      docker push myapp:latest
```

## Common Actions

### Setup and caching

```yaml
# Checkout code
- uses: actions/checkout@v4
  with:
    fetch-depth: 0                    # full history (for tags, etc.)

# Setup Node.js
- uses: actions/setup-node@v4
  with:
    node-version: '20'
    cache: 'npm'                      # built-in caching

# Setup Python
- uses: actions/setup-python@v5
  with:
    python-version: '3.12'
    cache: 'pip'

# Setup Go
- uses: actions/setup-go@v5
  with:
    go-version: '1.24'
    cache: true

# Generic cache action
- uses: actions/cache@v4
  with:
    path: ~/.cache/my-tool
    key: ${{ runner.os }}-my-tool-${{ hashFiles('**/lockfile') }}
    restore-keys: |
      ${{ runner.os }}-my-tool-
```

## Matrix Builds

### Test across multiple versions/OSes

```yaml
jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        node-version: [18, 20, 22]
        exclude:
          - os: windows-latest
            node-version: 18
        include:
          - os: ubuntu-latest
            node-version: 22
            experimental: true
      fail-fast: false                # don't cancel others on failure
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: ${{ matrix.node-version }}
      - run: npm test
```

## Conditional Steps

### Using if expressions

```yaml
steps:
  - name: Deploy to production
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    run: ./deploy.sh prod

  - name: Comment on PR
    if: github.event_name == 'pull_request'
    run: echo "This is a PR"

  - name: Run only on failure
    if: failure()
    run: echo "Something failed"

  - name: Always run (cleanup)
    if: always()
    run: echo "cleanup"

  - name: Skip on forks
    if: github.repository == 'owner/repo'
    run: echo "not a fork"
```

## Secrets and Environment Variables

### Using secrets

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    env:
      APP_ENV: production             # job-level env var
    steps:
      - name: Deploy
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        run: aws s3 sync ./dist s3://my-bucket

      - name: Use GitHub token (auto-provided)
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: gh pr comment --body "Deployed"
```

## Artifacts

### Upload and download build artifacts

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm run build
      - uses: actions/upload-artifact@v4
        with:
          name: dist
          path: dist/
          retention-days: 7

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v4
        with:
          name: dist
          path: dist/
      - run: ls dist/
```

## Environments

### Deployment environments with protection

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment:
      name: production
      url: https://myapp.example.com
    steps:
      - run: ./deploy.sh
        env:
          DEPLOY_KEY: ${{ secrets.DEPLOY_KEY }}  # environment-scoped secret
```

## Reusable Workflows

### Call a reusable workflow

```yaml
# .github/workflows/deploy.yml (reusable)
on:
  workflow_call:
    inputs:
      environment:
        required: true
        type: string
    secrets:
      deploy-key:
        required: true

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Deploying to ${{ inputs.environment }}"
        env:
          KEY: ${{ secrets.deploy-key }}
```

```yaml
# .github/workflows/ci.yml (caller)
jobs:
  call-deploy:
    uses: ./.github/workflows/deploy.yml
    with:
      environment: staging
    secrets:
      deploy-key: ${{ secrets.STAGING_KEY }}
```

## Composite Actions

### Create a local composite action

```yaml
# .github/actions/setup-env/action.yml
name: Setup Environment
description: Install deps and configure env
inputs:
  node-version:
    description: Node.js version
    default: '20'
runs:
  using: composite
  steps:
    - uses: actions/setup-node@v4
      with:
        node-version: ${{ inputs.node-version }}
    - run: npm ci
      shell: bash
```

```yaml
# Usage in a workflow
steps:
  - uses: actions/checkout@v4
  - uses: ./.github/actions/setup-env
    with:
      node-version: '22'
```

## Self-Hosted Runners

### Runner configuration

```yaml
jobs:
  build:
    runs-on: self-hosted                # use a self-hosted runner
    # or use labels
    # runs-on: [self-hosted, linux, gpu]
    steps:
      - uses: actions/checkout@v4
      - run: make build
```

## Key Environment Variables

```yaml
# Available in every workflow run
steps:
  - run: |
      echo "Repository: $GITHUB_REPOSITORY"       # owner/repo
      echo "Ref:        $GITHUB_REF"               # refs/heads/main
      echo "SHA:        $GITHUB_SHA"               # full commit SHA
      echo "Actor:      $GITHUB_ACTOR"             # who triggered
      echo "Run ID:     $GITHUB_RUN_ID"            # unique run ID
      echo "Run Number: $GITHUB_RUN_NUMBER"        # incrementing number
      echo "Workspace:  $GITHUB_WORKSPACE"         # checkout dir
      echo "Event:      $GITHUB_EVENT_NAME"        # push, pull_request, etc.
      echo "API URL:    $GITHUB_API_URL"            # https://api.github.com
```

## Tips

- Use `concurrency` to cancel in-progress runs on the same branch.
- Use `permissions` at job/workflow level to restrict the GITHUB_TOKEN scope.
- Pin actions to a full SHA instead of a tag for supply-chain security.
- Use `continue-on-error: true` on non-critical steps.
- Store reusable workflows in a central `.github` repository for the org.
- Use `gh act` or `act` (third-party) to test workflows locally.
- Set `ACTIONS_STEP_DEBUG` secret to `true` for verbose step logs.

## See Also

- gitlab-ci
- jenkins
- docker
- kubernetes
- helm
- git

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Workflow Syntax Reference](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
- [Context and Expression Syntax](https://docs.github.com/en/actions/learn-github-actions/contexts)
- [Events That Trigger Workflows](https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows)
- [Reusable Workflows](https://docs.github.com/en/actions/using-workflows/reusing-workflows)
- [Creating Composite Actions](https://docs.github.com/en/actions/creating-actions/creating-a-composite-action)
- [Encrypted Secrets](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions)
- [Using Environments for Deployment](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)
- [Caching Dependencies](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [Self-Hosted Runners](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners/about-self-hosted-runners)
- [Actions Marketplace](https://github.com/marketplace?type=actions)
- [Security Hardening for GitHub Actions](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
