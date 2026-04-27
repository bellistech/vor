# GitHub Actions — ELI5 (Robot Assembly Line for Your Repo)

> GitHub Actions is a robot assembly line that wakes up every time you push code, runs through a checklist (build, test, lint, deploy), and tells you what broke.

## Prerequisites

(none — but `cs ramp-up git-eli5` helps if you don't know what a push is)

This sheet is the very first stop for GitHub Actions. You do not need to know what "CI" is. You do not need to know YAML. You do not need to know what a "runner" is or what a "job" is. By the end of this sheet you will know all of those things in plain English, and you will have a real workflow file in your head and you will know how to read one and how to write one and how to figure out why one broke.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

If you see something like `name:` or `runs-on:` in a code block, that is **YAML**. YAML is a way of writing down lists and labels. It looks weird at first. We will explain every line we show. You do not need to learn YAML before you read this sheet. YAML will become familiar by the time you are done.

## What Even Is GitHub Actions?

### Imagine your code is a factory order

Picture a really busy factory. Every morning, somebody rolls a cart up to the front door with a big stack of paper on it. The paper is the order: "build me a hundred toys, paint them red, put them in boxes, ship them."

The factory has a long assembly line inside. The line has stations. Station 1 cuts wood. Station 2 sands the wood. Station 3 paints the wood. Station 4 boxes it up. Station 5 puts a label on it. The cart goes in one end of the line, and at the other end of the line, finished boxes come out.

At each station there is a robot. The robot at station 1 has saws. The robot at station 2 has sandpaper. The robot at station 3 has paintbrushes. Each robot does its little job, and then it slides the work down to the next station.

Now imagine the cart at the front door is not a stack of paper. Imagine the cart is your code. Every time you save your code on GitHub (we call that "pushing"), a cart with your code rolls up to the door of the GitHub Actions factory. Inside, an assembly line starts up. Robot 1 grabs your code. Robot 2 builds it. Robot 3 tests it. Robot 4 paints it (we call this "linting"). Robot 5 packs it into a box. Robot 6 ships the box to your server.

When the line finishes, the factory tells you, "Yep, all good, every robot did its job," with a little green check mark next to your push. If anything broke at any station, the factory tells you, "Robot 3 burst into flames at the test station, here is the smoke report," with a little red X.

That is GitHub Actions. It is a robot assembly line bolted to your code. Every time your code shows up at the door, the line runs.

### Imagine a really obedient sous-chef

Here is another way to think about it. Pretend you are a head chef. You are very busy. You write recipes on index cards and put the cards on a hook. Every time a customer orders, your sous-chef (your assistant) reads the next card off the hook and follows the recipe step by step. They never skip a step. They never forget the salt. They do exactly what is on the card.

GitHub Actions is your sous-chef. The recipe cards are your **workflow files**. The customers are events on your repo: somebody pushed code, somebody opened a pull request, somebody clicked the "run this manually" button, the clock ticked over to 3 AM. Each event is a customer ordering something.

When a customer orders, the sous-chef grabs the right card and starts cooking. They build, test, lint, deploy. They tell you when it's done. They tell you when it's burnt.

The huge advantage is: the sous-chef never gets tired. The sous-chef never goes on vacation. The sous-chef works at 3 AM. The sous-chef does the same thing every single time, the same way, without getting bored. That is the whole point of automation. You get a tireless cook, and you only have to write the recipe once.

### Imagine a vending machine for your repo

Here is one more picture. Pretend GitHub Actions is a vending machine bolted to the side of your project. The vending machine has buttons. One button says "build." One button says "test." One button says "deploy." One button says "publish a release."

Some of the buttons get pressed automatically. The "build" button gets pressed every time you push code. The "test" button gets pressed every time somebody opens a pull request. The "deploy" button gets pressed every time somebody pushes a tag that starts with `v`. The "publish a release" button gets pressed when somebody clicks "Create release" in the GitHub UI.

Other buttons are pressed by hand. You walk up to the vending machine, click the "deploy" button, and the machine starts whirring. We call that a **manual trigger** or a `workflow_dispatch` (a fancy name for "I dispatched the workflow myself").

That is GitHub Actions. A vending machine of buttons attached to your repo. You write the buttons. You decide what each one does. You decide when each one fires.

### Why so many pictures?

You might wonder why we have a factory picture, a sous-chef picture, and a vending machine picture. The answer is: GitHub Actions is invisible. You cannot see the robots. You cannot see the kitchen. You cannot see the vending machine. So we use pictures to imagine what is happening.

The **factory** picture is best for understanding that the line has stations and each station hands off to the next.

The **sous-chef** picture is best for understanding that you write a recipe once and it gets followed every single time, perfectly, forever.

The **vending machine** picture is best for understanding that there are different buttons, and different events press different buttons.

If one picture is not clicking, switch to another. Whichever one feels right is the one you should keep in your head.

### What the robots actually do

In real life, the robots are computers. The "stations" are little boxed-off computers that GitHub turns on for a few minutes, lets your code run on, and then turns off and throws away. We call these little computers **runners.** The runner is the robot's body. The robot's brain is your workflow file telling it what to do. The robot's arms are the tools (compilers, test frameworks, deploy scripts) you tell it to use.

Your workflow file says, "Robot, please go to a fresh Ubuntu computer. Now go fetch my code from the repo. Now install Go. Now run my tests. Now build a binary. Now upload the binary to S3." The runner is the body. The robot is the runner doing the steps you wrote.

When the runner is done, GitHub turns it off. The runner disappears. The next time the workflow runs, GitHub turns on a brand new runner. This is one of the most important things to understand: **runners are throwaway.** They get a fresh computer every time. Anything you put on the runner gets wiped when the run is over. If you want to keep something, you have to upload it (as an **artifact**, or to S3, or to your registry, or wherever). Otherwise, poof, it is gone.

This is also why builds in GitHub Actions are usually pretty repeatable. Every run starts from a clean slate. You don't have leftover junk from yesterday's run. You don't have files mysteriously sticking around. Every run is brand new.

### Why does this matter?

Before automation like GitHub Actions existed, somebody on your team had to push a button on their laptop every time the code changed. They had to remember to run the tests. They had to remember to build the artifact. They had to remember to upload it. Humans forget. Humans get tired. Humans go home at 5 PM. Humans take vacations.

The robot never forgets. The robot is at the office at 3 AM on a Sunday. The robot does the same thing every single time. The robot does not get bored.

That is why every serious software project uses something like this. The robot does the boring, repetitive, error-prone parts. The humans do the interesting parts (deciding what to build, talking to users, designing the system).

## Workflows, Jobs, Steps, Actions

There are four nesting levels you have to know. They show up everywhere.

```
+------------------------------------------------------+
|  WORKFLOW (a file in .github/workflows/*.yml)       |
|                                                      |
|  +-----------------------+  +---------------------+ |
|  |  JOB "build"          |  |  JOB "test"         | |
|  |  (runs on its own     |  |  (runs on its own   | |
|  |   runner, parallel    |  |   runner, parallel  | |
|  |   to other jobs)      |  |   to other jobs)    | |
|  |                       |  |                     | |
|  |  Step 1: checkout     |  |  Step 1: checkout   | |
|  |  Step 2: setup Go     |  |  Step 2: setup Go   | |
|  |  Step 3: go build     |  |  Step 3: go test    | |
|  |  (steps share state,  |  |                     | |
|  |   run in order)       |  |                     | |
|  +-----------------------+  +---------------------+ |
+------------------------------------------------------+
```

### Workflow

A **workflow** is one YAML file inside the special folder `.github/workflows/` in your repo. Every file in that folder is a separate workflow. You can have one workflow file or fifty. GitHub finds them automatically.

Each workflow has a name (the `name:` line at the top), a list of triggers (the `on:` block, what events fire this workflow), and a list of jobs (the `jobs:` block, the work it does).

If your workflow file is at `.github/workflows/ci.yml`, that is the file GitHub reads. If you rename it to `.github/workflows/build.yml`, GitHub reads it as `build.yml`. The folder is fixed (`.github/workflows`), the filename you choose. The file extension can be `.yml` or `.yaml`. Both work.

### Job

A **job** is one chunk of work that runs on one runner. Inside a workflow, you can have many jobs. By default, all the jobs run **at the same time** (in parallel). The runner for job A is a different computer than the runner for job B. They do not share files. They do not share environment variables. They are isolated.

Jobs can wait for other jobs to finish first. If you write `needs: [build]` on a job, that job waits until `build` is done before it starts. This is how you build a chain: "first build, then test, then deploy" — three jobs, each `needs:` the previous one, runs in order.

Each job runs on a runner. You pick which runner with the `runs-on:` line. `runs-on: ubuntu-latest` means "use a fresh Ubuntu Linux machine." `runs-on: macos-latest` means "use a fresh macOS machine." `runs-on: windows-latest` means "use a fresh Windows machine."

### Step

A **step** is one little task inside a job. Steps run in **order**, top to bottom, on the same runner. Step 1 finishes, then step 2 starts, then step 3, and so on. If any step fails, by default the rest of the steps in that job are skipped.

Steps share state. If step 1 makes a file, step 2 can read it. If step 1 sets an environment variable in `$GITHUB_ENV`, step 2 sees it. If step 1 changes the working directory, step 2 starts from where step 1 left off (well, sort of — `working-directory:` is per-step, but files persist).

There are two kinds of steps:

- **`run:`** steps run a shell command. `run: go test ./...` runs `go test ./...` in the runner's shell.
- **`uses:`** steps run an **action** (a reusable bundle of work somebody else wrote). `uses: actions/checkout@v4` runs the official "check out my code" action.

### Action

An **action** is a reusable robot. Somebody wrote it once, published it to the GitHub Marketplace, and now anybody in the world can plug it into their workflow with one line: `uses: name/repo@version`.

There are three kinds of actions:

- **JavaScript actions** — run as Node.js code on the runner.
- **Docker actions** — run as a Docker container on the runner.
- **Composite actions** — bundle a bunch of `run:` and `uses:` steps into one reusable thing.

You probably don't need to know which kind something is most of the time. You just plug it in with `uses:` and pass it inputs with `with:`.

The most famous actions are `actions/checkout@v4` (clones your repo onto the runner), `actions/setup-go@v5` (installs Go), `actions/setup-node@v4` (installs Node), `actions/setup-python@v5` (installs Python), `actions/cache@v4` (saves and restores cache), `actions/upload-artifact@v4` (saves a file from the runner), `actions/download-artifact@v4` (gets that file back). You will see these in almost every workflow file in the world.

## A Hello-World Workflow

Here is a workflow file. It is the simplest CI you can write.

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./...
```

Let's walk through every single line.

**`name: CI`** — gives the workflow a human-readable name. This is what shows up in the GitHub UI on the "Actions" tab. You can call it whatever you want. "CI" is a common choice (CI = continuous integration, fancy name for "run my tests on every change").

**`on: [push, pull_request]`** — the triggers. This workflow fires when somebody pushes to any branch (`push`) and also when somebody opens or updates a pull request (`pull_request`). The square brackets are YAML's way of writing a list of strings.

**`jobs:`** — this is where the list of jobs starts. Below this line, every job is its own block.

**`  test:`** — the name of the first (and only) job is `test`. The two-space indent is important. YAML uses indentation to mean "this thing is inside that thing." `test:` is inside `jobs:`. Get the indentation wrong and YAML breaks.

**`    runs-on: ubuntu-latest`** — this job runs on a fresh Ubuntu Linux machine. GitHub will boot one up for you and throw it away when you're done.

**`    steps:`** — list of steps starts here.

**`      - uses: actions/checkout@v4`** — the first step uses an action called `actions/checkout`, version `v4`. This action clones your repo onto the runner. Without this step, the runner has no copy of your code. You will basically always have this as the first step. The dash at the beginning means "this is one item in a list."

**`      - uses: actions/setup-go@v5`** — second step. Uses an action called `actions/setup-go`, version `v5`. This installs Go on the runner.

**`        with:`** — passes inputs to the action. The action takes parameters; we list them under `with:`.

**`          go-version: '1.22'`** — tells `setup-go` to install Go 1.22. The single quotes around `1.22` keep it as a string, not a number, because YAML sometimes interprets numbers weirdly (it might read `1.22` as the float 1.22 and then try to install Go 1.22.0.0.0 or something silly).

**`      - run: go test ./...`** — third step. Runs the shell command `go test ./...` on the runner. This is your actual test command. The runner now has Go installed (from step 2) and your code (from step 1), so it can compile and run your tests.

That's the whole file. Save it as `.github/workflows/ci.yml`, commit it, push it to GitHub. The next push to any branch (and the next pull request) will trigger this workflow. You will see the run in the "Actions" tab of your repo on github.com.

## Triggers (the "on:" block)

The `on:` block decides when the workflow fires. There are tons of events. Here are the ones you will use most.

### `push`

```yaml
on:
  push:
    branches: [main, develop]
    paths:
      - 'src/**'
      - '!src/docs/**'
    tags:
      - 'v*'
```

Fires when somebody pushes commits to the repo. You can filter by branch (`branches:` list), by file path (`paths:` list — and `!` means "exclude this path"), and by tag (`tags:` list).

`branches: [main]` means "only fire on pushes to main." `branches: ['feature/*']` means "fire on any branch whose name starts with `feature/`." `branches-ignore: [draft/*]` means "fire on every branch except ones starting with `draft/`."

### `pull_request`

```yaml
on:
  pull_request:
    types: [opened, synchronize, reopened]
    branches: [main]
```

Fires when somebody opens or updates a pull request. The default `types:` are `[opened, synchronize, reopened]`, which is "PR was just opened, somebody pushed a new commit to the PR, or somebody reopened a closed PR."

There is also `pull_request_target`, which is dangerous. Read the section below on it before you ever use it.

### `workflow_dispatch`

```yaml
on:
  workflow_dispatch:
    inputs:
      env:
        description: 'Which environment'
        required: true
        default: 'staging'
        type: choice
        options:
          - staging
          - production
```

Fires when somebody clicks the "Run workflow" button in the GitHub UI, or when somebody runs `gh workflow run` from the CLI. You can ask for inputs (the user fills them in before clicking the button). This is the manual trigger. Great for "deploy to production now."

### `schedule`

```yaml
on:
  schedule:
    - cron: '0 4 * * *'
```

Fires on a cron schedule. The format is the standard `m h dom mon dow` — minute, hour, day-of-month, month, day-of-week. `0 4 * * *` means "every day at 4:00 AM UTC." `*/15 * * * *` means "every 15 minutes."

Important: scheduled workflows run from the **default branch** (usually `main`). Whatever the workflow file looks like on `main` is what runs. If you modify the workflow on a feature branch, the cron does not pick up the changes until they merge to `main`.

### `workflow_run`

```yaml
on:
  workflow_run:
    workflows: ['CI']
    types: [completed]
```

Fires after another workflow finishes. Useful for "after CI passes, deploy." Note: this also runs from the default branch, which trips a lot of people up.

### `repository_dispatch`

```yaml
on:
  repository_dispatch:
    types: [my-custom-event]
```

Fires when somebody calls the GitHub API with a custom event type. This lets external systems (or other repos) kick off your workflows. You'd use this to chain things across repos, or to let a chatbot trigger a deploy.

### `release`

```yaml
on:
  release:
    types: [published]
```

Fires when a GitHub release is published, edited, deleted, etc. Great for "when somebody hits the Release button, build and ship the artifacts."

### Other triggers worth knowing

- `issues:` — when an issue is opened, edited, closed, etc.
- `issue_comment:` — when somebody comments on an issue or PR.
- `label:` — when a label is created, edited, deleted.
- `deployment_status:` — when a deployment changes state.
- `check_run:`, `check_suite:` — for the checks API.
- `page_build:` — when GitHub Pages rebuilds.
- `registry_package:` — when a package is published.
- `secret_scanning_alert:`, `dependabot_alert:`, `code_scanning_alert:` — security alerts.
- `fork:`, `watch:` — when somebody forks or stars your repo.

## Runners

The runner is the robot's body. It is a fresh computer that GitHub boots up just for your job, runs your steps on, and then throws away.

```
+----------------------------------------------------+
|  GitHub-hosted runner (fresh VM, ~14 GB RAM)      |
|                                                    |
|  +----------+  +-----------+  +----------------+  |
|  |  /home/  |  |  pre-     |  |  Docker        |  |
|  |  runner/ |  |  installed|  |  socket        |  |
|  |  work/   |  |  tools    |  |  available     |  |
|  +----------+  +-----------+  +----------------+  |
|                                                    |
|  Lifetime: a few minutes (the length of the job)  |
|  Disk: ~14 GB free                                 |
|  Network: open outbound, no inbound               |
+----------------------------------------------------+
```

### GitHub-hosted runners

These are runners GitHub provides. You don't have to manage anything. You just say `runs-on: ubuntu-latest` and GitHub spins up a Linux VM. When the job finishes, GitHub destroys it.

The most common labels:

- `ubuntu-latest` (currently Ubuntu 22.04 or 24.04 — points to the most recent supported Ubuntu)
- `ubuntu-22.04`, `ubuntu-24.04` — pin a specific Ubuntu version
- `macos-latest` (currently macOS 14)
- `macos-13`, `macos-14`, `macos-15`
- `windows-latest` (Windows Server 2022)
- `windows-2022`, `windows-2019`

GitHub-hosted runners are **free for public repositories**, with very generous limits. For private repos, they are metered: minutes per month included with your plan, then per-minute pricing. macOS minutes are most expensive, Windows next, Linux cheapest. There is a 10x multiplier for macOS in private repos (one macOS minute counts as 10 Linux minutes against your quota).

Pre-installed tools include common languages (Node, Python, Go, Ruby, Java, .NET), Docker, Git, gh CLI, AWS CLI, Azure CLI, gcloud, and many more. The runner image is documented and updated weekly. See `actions/runner-images` on GitHub for the full list.

### Self-hosted runners

You can also run your own runners. You install the runner agent on a machine you own (a VM, a bare-metal server, a Raspberry Pi, whatever) and the agent connects to GitHub and waits for jobs. When a job comes in, your machine runs it.

You'd do this when:

- You have specific hardware your job needs (GPU, lots of RAM, ARM, FPGA).
- You need access to your private network.
- You have very high volumes and the GitHub-hosted minutes get expensive.
- You have compliance requirements (data must stay on your servers).

Self-hosted runners can be standalone or organized into groups. You target them with labels: `runs-on: [self-hosted, linux, gpu]` means "any self-hosted runner that has all three labels."

For Kubernetes shops, there's **ARC (Actions Runner Controller)** — an operator that runs on your cluster and creates ephemeral runner pods on demand. ARC is the modern way to scale self-hosted runners. Each job gets a fresh pod, the pod runs the job, the pod gets destroyed. Same model as GitHub-hosted runners but on your own k8s cluster.

### Runner architecture

```
GitHub.com  <----long-poll---- Runner agent
                                  |
                                  +--> spawns a worker process per job
                                  +--> downloads actions
                                  +--> runs your steps
                                  +--> uploads logs/artifacts
                                  +--> reports status back to GitHub
```

The runner agent is a small Go-ish program (mostly C# actually) that connects out to GitHub over HTTPS and waits for work. When GitHub has a job for it, GitHub sends down the workflow data. The agent downloads the actions, runs your steps, streams logs back to GitHub in real time, uploads artifacts, and reports the final status.

## Secrets and Variables

Workflows often need secrets: API keys, deploy tokens, certificates. You **never** put secrets in the YAML file. You put them in GitHub's secret store and reference them with `${{ secrets.MY_KEY }}`.

```yaml
- run: deploy.sh
  env:
    AWS_ACCESS_KEY_ID:     ${{ secrets.AWS_ACCESS_KEY_ID }}
    AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
    AWS_REGION:            ${{ vars.AWS_REGION }}
```

### Secrets

Secrets are encrypted at rest. They are masked in logs (if a secret value shows up in a log, GitHub replaces it with `***`). Once a secret is set, you can't read it back from the UI — you can only overwrite it.

Secrets live at three levels:

- **Repository secrets** — only this repo can see them.
- **Organization secrets** — every repo in the org can see them (you can scope to specific repos).
- **Environment secrets** — only jobs targeting a specific environment can see them.

You set them in **Settings → Secrets and variables → Actions**, or with `gh secret set NAME`.

### Variables

Variables are the same idea but **not secret**. Use these for things like region names, account IDs (some teams treat account IDs as not-secret), feature flags. They're not encrypted and they're visible in logs.

Reference with `${{ vars.NAME }}`.

### `env:` block

You can also set plain environment variables in the workflow file:

```yaml
env:                              # workflow-level
  GLOBAL_VAR: hello

jobs:
  test:
    env:                          # job-level
      JOB_VAR: world
    steps:
      - run: echo $GLOBAL_VAR $JOB_VAR
        env:                      # step-level
          STEP_VAR: !
```

Order of precedence (most specific wins): step-level `env:` > job-level `env:` > workflow-level `env:`. Step env beats job env beats workflow env.

### Why secrets aren't shared with forks

If your repo is public, anybody can fork it and open a pull request. If a forked PR could read your secrets, a malicious fork could just print your AWS keys. So GitHub does **not** share secrets with `pull_request` workflows from forks. The workflow runs, but `secrets.X` is empty.

This is annoying when you want to test things, but it is the right behavior. There is a separate event called `pull_request_target` which **does** get secrets, but it runs the workflow from your default branch (not the fork's code), which is the safe direction. Use `pull_request_target` carefully — if you check out the fork's code in a `pull_request_target` workflow, you have re-introduced the security hole.

## Matrix Builds

Matrix builds let you run the same job many times with different inputs. Common use: "test on Linux, macOS, and Windows; on Go 1.21, 1.22, and 1.23." That's a 3×3 = 9-job matrix.

```yaml
jobs:
  test:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.21', '1.22', '1.23']
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}
      - run: go test ./...
```

This expands into nine jobs:

```
ubuntu-latest    + go 1.21   --> job 1
ubuntu-latest    + go 1.22   --> job 2
ubuntu-latest    + go 1.23   --> job 3
macos-latest     + go 1.21   --> job 4
macos-latest     + go 1.22   --> job 5
macos-latest     + go 1.23   --> job 6
windows-latest   + go 1.21   --> job 7
windows-latest   + go 1.22   --> job 8
windows-latest   + go 1.23   --> job 9
```

All nine run in parallel (subject to your runner availability).

### `fail-fast`

`fail-fast: true` (the default) means: if any job in the matrix fails, cancel all the others. Saves time but you don't see whether other combinations would have passed.

`fail-fast: false` means: let all the matrix combinations finish even if one fails. Better when you want to know "does it fail on Windows only, or also on macOS?"

### `max-parallel`

`max-parallel: 3` limits how many matrix jobs run at the same time. Useful if your matrix is huge and you don't want to swarm your runners.

### `include:` and `exclude:`

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest]
    go: ['1.21', '1.22']
    include:
      - os: ubuntu-latest
        go: '1.20'
        experimental: true
    exclude:
      - os: macos-latest
        go: '1.21'
```

`include:` adds extra combinations beyond the cross-product. The example above adds an extra `ubuntu-latest + go 1.20` job and tags it with `experimental: true`.

`exclude:` removes specific combinations. The example removes `macos-latest + go 1.21`.

The `experimental: true` from the `include:` is now available in the job as `${{ matrix.experimental }}`. You can use it for conditionals.

## Reusable Workflows

You can write a workflow once and call it from many other workflows. Reusable workflows are top-level workflow files that have `workflow_call:` as a trigger.

```yaml
# .github/workflows/reusable-test.yml
name: Reusable test
on:
  workflow_call:
    inputs:
      go-version:
        required: true
        type: string
    secrets:
      MY_TOKEN:
        required: false

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ inputs.go-version }}
      - run: go test ./...
        env:
          TOKEN: ${{ secrets.MY_TOKEN }}
```

Then call it from another workflow:

```yaml
# .github/workflows/ci.yml
on: [push]

jobs:
  call-test:
    uses: ./.github/workflows/reusable-test.yml
    with:
      go-version: '1.22'
    secrets:
      MY_TOKEN: ${{ secrets.PROD_TOKEN }}
```

You can call workflows from the same repo with `./.github/workflows/file.yml`, or from another repo with `owner/repo/.github/workflows/file.yml@ref`.

Reusable workflows are great for "every team uses the same deploy pipeline, written once."

## Composite Actions

Composite actions bundle a sequence of steps into a single reusable action. Different from a reusable workflow: a composite action is a **step** you can drop into any job. A reusable workflow is a whole **job** (or set of jobs) you call.

```yaml
# .github/actions/setup-app/action.yml
name: 'Setup app'
description: 'Set up Go and download deps'
inputs:
  go-version:
    description: 'Go version'
    required: true
    default: '1.22'
runs:
  using: 'composite'
  steps:
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ inputs.go-version }}
    - run: go mod download
      shell: bash
```

Then use it in any workflow:

```yaml
- uses: ./.github/actions/setup-app
  with:
    go-version: '1.23'
```

Composite actions are for bundling **steps**. Reusable workflows are for bundling **jobs**. If you find yourself copy-pasting three steps into every workflow, make a composite action. If you find yourself copy-pasting a whole job, make a reusable workflow.

## OIDC Token Auth (No Static Cloud Secrets)

Old way: store an AWS access key as a secret, use it to deploy. Problem: if the secret leaks, the leak is permanent until you rotate.

New way: GitHub gives each workflow run a short-lived **OIDC token** (think "ID badge that expires in 5 minutes"). You configure your cloud provider to trust GitHub's OIDC. The workflow exchanges the OIDC token for a temporary cloud credential (lasts an hour, scoped to one repo, one branch). No long-lived secret on either side.

```
+--------------------+        OIDC token       +-----------+
|   GitHub Actions   |  -------------------->  |   AWS     |
|   workflow run     |  <------------------    |  STS API  |
|                    |    temporary creds      |           |
+--------------------+   (1 hr, scoped to       +-----------+
                          repo/branch)
```

```yaml
permissions:
  id-token: write       # required to mint the OIDC token
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/MyDeployRole
          aws-region: us-east-1
      - run: aws s3 sync ./build s3://my-bucket
```

On the AWS side, you create an IAM role with a trust policy that says "trust GitHub's OIDC issuer, but only for repo `myorg/myrepo`, only for branch `main`." Now even if somebody steals your workflow file, they can't deploy: their fork's OIDC token won't match the trust policy.

GCP equivalent: workload identity federation. Azure equivalent: federated identity credentials. AWS equivalent: `AssumeRoleWithWebIdentity`. All three providers support this exact pattern. It is the right way to authenticate workflows to clouds.

## Caching

Builds repeat the same downloads every time: dependencies, packages, Docker layers. The `actions/cache@v4` action saves directories between runs to make them faster.

```yaml
- uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-go-
```

How it works:

1. At step start, the action looks for a cache entry under `key`. If found, it downloads it to `path:`.
2. If not found, it falls back to the first `restore-keys:` prefix that matches.
3. At end of job, if the exact `key:` was not a hit (i.e. you got a partial restore or no restore), the action saves whatever is in `path:` under `key:` for future runs.

Cache key resolution diagram:

```
key: linux-go-abc123 (hash of go.sum)

      v
+--------------------------+
| Look for exact key match |
+--------------------------+
      |
      | not found
      v
+----------------------------------+
| Look for restore-key prefix:     |
|   linux-go-                      |
+----------------------------------+
      |
      | found "linux-go-def456"
      v
+--------------------------+
| Restore that cache       |
+--------------------------+
      |
      v
  job runs, downloads any extra deps,
  hashFiles changes, key now is
  linux-go-abc123 (no exact match
  was found earlier), so save now
  under linux-go-abc123
```

Caches are scoped to the branch they were created on, plus the default branch. Other branches can read a default-branch cache but cannot read a sibling branch's cache. Caches expire after 7 days of no access. There is a 10 GB limit per repo.

## Artifacts

Artifacts are files you save from a workflow run. Think build outputs, test reports, screenshots from a flaky test. Different from cache (which is for "make the build faster") — artifacts are for "I want this file."

```yaml
- run: go build -o myapp ./cmd/myapp
- uses: actions/upload-artifact@v4
  with:
    name: myapp-binary
    path: myapp
    retention-days: 7
```

Then in another job (or even just to download from the GitHub UI):

```yaml
- uses: actions/download-artifact@v4
  with:
    name: myapp-binary
```

Artifacts persist after the run finishes. By default they live for 90 days. Anyone with access to the repo can download them from the run's summary page.

Artifacts are how you pass files **between jobs**. Job A uploads. Job B downloads. Without this, jobs can't share files (different runners, no shared disk).

## Permissions and the GITHUB_TOKEN

Every workflow run gets a magical token called `GITHUB_TOKEN`. It is a short-lived credential that lets the workflow talk to GitHub's API as if it were your repo's bot. Read commits, write comments, create checks, push tags — depending on its scopes.

You should always think about what scopes the token has. The default permissions changed in 2023: GitHub flipped the default to **read-only** to be safer. If your workflow needs to write something (push a commit, create a release, comment on a PR), you must explicitly grant the scope.

```yaml
permissions:
  contents: write           # push commits, create tags
  pull-requests: write      # comment on PRs
  issues: read
  packages: read
  id-token: write           # mint OIDC tokens
  attestations: write       # for build provenance
  statuses: write
  deployments: write
  pages: write
  security-events: write    # for code scanning uploads
  actions: read
```

You can put `permissions:` at the workflow level or per-job. Per-job is more secure ("only the deploy job can write contents; the test job is read-only").

Special form: `permissions: read-all` (everything read) and `permissions: write-all` (everything write — only when you really need it) and `permissions: {}` (everything denied).

## Environments and Approvals

Environments are a wrapper around deploys. You define an environment (`production`, `staging`, `qa`) and you can attach **protection rules**:

- **Required reviewers** — one or more humans must click "approve" before the job runs.
- **Wait timer** — sit and do nothing for N minutes after triggering (lets you cancel).
- **Deployment branches** — only certain branches can deploy here. ("Only `main` can deploy to production.")
- **Environment secrets** — secrets only exposed to jobs targeting this environment.

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment:
      name: production
      url: https://example.com
    steps:
      - run: ./deploy.sh
```

When this job starts, GitHub pauses it and waits for the environment's protection rules to be satisfied. Reviewers get notified. They click approve. The job continues. If they reject, the job fails.

```
+-----------+    +-------------+    +----------+    +--------+
|  trigger  | -> | wait timer  | -> | reviewer | -> | deploy |
|  fires    |    | (e.g. 5min) |    | approves |    |        |
+-----------+    +-------------+    +----------+    +--------+
                                          |
                                          | rejects
                                          v
                                     +---------+
                                     |  fail   |
                                     +---------+
```

This is your manual gate. Every serious production deploy should be protected by an environment.

## Common Errors

### "The job was canceled"

Either:

- A concurrency rule canceled it because a newer run started.
- Somebody clicked "Cancel run" in the UI.
- A `needs:` dependency failed and the cascade canceled this one.

Check the run's "concurrency" group and any upstream jobs.

### "Process completed with exit code N"

Your script returned a non-zero exit code. The runner reports this verbatim. Look at the step's log: the actual error from your tool is right above this message. The exit code itself rarely tells you anything; the lines above tell you everything.

### "No matching workflow run"

You triggered a workflow but nothing happened. Causes:

- Your `on:` block doesn't match the event (wrong branch filter, wrong event type).
- The workflow file has a YAML syntax error and GitHub silently ignored it.
- The workflow file is on a feature branch but the trigger only fires from the default branch (true for `schedule:` and `workflow_run:`).

Run `gh workflow view ci.yml` to confirm GitHub even knows about the workflow.

### "Resource not accessible by integration"

The `GITHUB_TOKEN` doesn't have the scope it needs to do what you asked. Add the scope under `permissions:`. For example, if commenting on a PR, you need `pull-requests: write`.

### "Required secret X not set"

The secret name in the YAML doesn't match a secret name in your repo's settings. Watch out for case (`MY_TOKEN` vs `my_token`) and scope (org-level vs repo-level vs environment-level). Run `gh secret list` to see what's defined.

### "Could not resolve to a User with the username"

You referenced an actor (`github.actor`) or user that doesn't exist. Usually a copy-paste typo.

### "Workflow file is invalid"

YAML syntax error or workflow schema violation. The error message tells you the line. Most common causes:

- Inconsistent indentation (tabs vs spaces, or wrong number of spaces).
- A list (`-`) where a map was expected, or vice versa.
- Unquoted special characters (`@`, `:`, `#`).

`actionlint` catches most of these before you push.

### "actions/checkout@vN: Reference @vN is not a tag or branch"

You used a version of an action that doesn't exist. Maybe a typo (`@v44` instead of `@v4`), maybe the action was renamed. Look up the action on github.com and find a real tag.

### "matrix variable X is not defined"

You used `${{ matrix.X }}` but `X` isn't in your matrix. Often a typo or a leftover reference after refactoring.

### "Error: actions/checkout@v3 has been deprecated"

GitHub deprecates old major versions of official actions. The fix is usually to bump the version: `@v3` → `@v4`.

## Hands-On

Set up `gh` (the GitHub CLI). On macOS: `brew install gh`. On Linux: see cli.github.com. Authenticate with `gh auth login`. The exercises below assume you are inside a repo on GitHub that has at least one workflow.

### List workflows

```
$ gh workflow list
NAME    STATE   ID
CI      active  12345678
Deploy  active  12345679
Nightly active  12345680
```

`gh workflow list` shows every workflow GitHub knows about for this repo.

### Show a workflow's YAML

```
$ gh workflow view ci.yml
CI - .github/workflows/ci.yml
ID: 12345678

  Total runs 247
  Recent runs

  status  conclusion  event  branch  workflow  ID
  ✓       success     push   main    CI        9876543
  ✓       success     push   main    CI        9876542
```

You can also pipe the YAML out: `gh workflow view ci.yml --yaml`.

### List recent runs

```
$ gh run list --workflow=ci.yml --limit 5
STATUS  TITLE                            WORKFLOW  BRANCH  EVENT  ID         AGE
✓       Fix flaky test (#123)            CI        main    push   9876543    5m
✓       Update deps                      CI        main    push   9876542    3h
X       Add new feature                  CI        feat/x  pr     9876541    1d
✓       Bump go version                  CI        main    push   9876540    2d
✓       Refactor handler                 CI        main    push   9876539    3d
```

`X` means failed, `✓` means succeeded. Use `--limit` to control how many.

### Watch a run live

```
$ gh run watch
? Select a workflow run  [Use arrows to move, type to filter]
> 9876543  CI  Fix flaky test
  9876542  CI  Update deps
  ...
```

After you pick, `gh run watch` streams the run to your terminal until it finishes. You see jobs starting, finishing, failing.

### View logs for a specific run

```
$ gh run view 9876543 --log
CI / test (ubuntu-latest, 1.22)  Set up job  2024-12-01T10:00:00.0000000Z Current runner version: '2.310.0'
CI / test (ubuntu-latest, 1.22)  Set up job  2024-12-01T10:00:00.0000000Z Operating System
CI / test (ubuntu-latest, 1.22)  Set up job  2024-12-01T10:00:00.0000000Z   Ubuntu 22.04.3 LTS
...
```

This dumps the entire log. Pipe through `less` or `grep` to find what you want.

### View only failed steps

```
$ gh run view 9876541 --log-failed
CI / test (windows-latest, 1.22)  Run go test ./...  2024-12-01T10:30:15.1234567Z FAIL: TestFoo
CI / test (windows-latest, 1.22)  Run go test ./...  2024-12-01T10:30:15.1234567Z   path_test.go:42: expected forward slash
```

Saves you scrolling.

### Re-run a failed run

```
$ gh run rerun 9876541
✓ Requested rerun of run 9876541
```

`--failed` re-runs only the failed jobs. Without `--failed` it re-runs everything.

### List secrets

```
$ gh secret list
NAME                       UPDATED
AWS_ACCESS_KEY_ID          2024-11-01
AWS_SECRET_ACCESS_KEY      2024-11-01
NPM_TOKEN                  2024-09-15
```

You can't see the values, only the names and when they were last updated.

### Set a secret

```
$ gh secret set MY_TOKEN
? Paste your secret  ****************
✓ Set Actions secret MY_TOKEN for owner/repo
```

Or pipe a value from a command: `op read op://Vault/Item/token | gh secret set MY_TOKEN`.

### List variables

```
$ gh variable list
NAME           VALUE                       UPDATED
AWS_REGION     us-east-1                   2024-11-01
DEPLOY_BRANCH  main                        2024-09-15
```

Variables are visible (not encrypted). Use for non-secret config.

### Manually trigger a workflow

```
$ gh workflow run ci.yml -f env=staging
✓ Created workflow_dispatch event for ci.yml at main
```

The `-f` flag passes inputs declared in the `workflow_dispatch:` block.

### Use the API directly

```
$ curl -H "Authorization: Bearer $GH_TOKEN" \
       https://api.github.com/repos/owner/repo/actions/runs?per_page=5
{
  "total_count": 247,
  "workflow_runs": [
    {
      "id": 9876543,
      "name": "CI",
      "head_branch": "main",
      "status": "completed",
      "conclusion": "success",
      ...
    },
    ...
  ]
}
```

Anything `gh` does, you can do with `curl` against the REST API. The `gh` CLI is just a friendly wrapper.

### Run a workflow locally with `act`

```
$ act
[CI/test] Start image=catthehacker/ubuntu:act-latest
[CI/test]   git clone /home/runner/work/repo/repo
[CI/test]   actions/setup-go@v5
[CI/test]   go test ./...
[CI/test]   PASS
```

`act` (from `nektos/act`) runs your workflow locally in Docker. Great for debugging without burning runner minutes. Install with `brew install act` or `gh extension install nektos/gh-act`.

### Run one specific job locally

```
$ act -j test
[CI/test] Start image=catthehacker/ubuntu:act-latest
[CI/test]   ...
```

`-j <job-id>` runs just that one job. Use the job key from your YAML.

### Pass secrets to act

```
$ act --secret-file .env.local
[CI/test]   AWS_ACCESS_KEY_ID=*****
[CI/test]   ...
```

The `.env.local` file is a list of `KEY=value` lines. Don't commit it.

### Slice the YAML with yq

```
$ yq '.jobs.test.steps' .github/workflows/ci.yml
- uses: actions/checkout@v4
- uses: actions/setup-go@v5
  with:
    go-version: '1.22'
- run: go test ./...
```

`yq` is the YAML cousin of `jq`. Great for grepping deep into workflow files.

### Parse with Python

```
$ python -c "import yaml; print(yaml.safe_load(open('.github/workflows/ci.yml')))"
{'name': 'CI', 'on': ['push', 'pull_request'], 'jobs': {'test': {'runs-on': 'ubuntu-latest', 'steps': [...]}}}
```

If you ever need to programmatically inspect a workflow, this is the easiest way.

### Validate YAML with yamllint

```
$ find .github -name '*.yml' -exec yamllint {} \;
.github/workflows/ci.yml
  3:1   error  too many blank lines (2 > 1)  (empty-lines)
  10:81 error  line too long (95 > 80 characters)  (line-length)
```

`yamllint` catches generic YAML problems (indentation, line length, trailing whitespace).

### Validate workflows with actionlint

```
$ actionlint .github/workflows/ci.yml
.github/workflows/ci.yml:7:9: "uses" of "actions/checkout@v3" is outdated.
                                "v4" is the latest version. [reusable-workflows-tags]
```

`actionlint` (from `rhysd/actionlint`) is a static analyzer specifically for GitHub Actions YAML. It catches:

- Outdated action versions.
- Unknown contexts (`${{ secrets.X }}` where X isn't a real secret).
- Shell script issues (passes `run:` blocks through `shellcheck`).
- Invalid expressions.
- Wrong matrix references.

Run it locally before every push. CI configs catch a lot of waste.

### Install actionlint globally

```
$ brew install actionlint
==> Pouring actionlint--1.7.1.arm64_sequoia.bottle.tar.gz
🍺  /opt/homebrew/Cellar/actionlint/1.7.1: 4 files, 6.2MB
$ actionlint --version
1.7.1
installed by Homebrew
built with go1.22
```

### Install gh-act extension

```
$ gh extension install nektos/gh-act
✓ Installed extension nektos/gh-act
$ gh act --help
Run GitHub Actions locally
...
```

Now you can run `gh act` instead of installing `act` separately.

### Cancel a running workflow

```
$ gh run cancel 9876543
✓ Requested cancellation of run 9876543
```

Stops a workflow that's still running.

### Delete an old workflow run

```
$ gh run delete 9876541
? Are you sure you want to delete the run 9876541? Yes
✓ Run 9876541 deleted
```

Cleans up old runs. Useful for failed runs you don't want cluttering history.

### List artifacts from a run

```
$ gh run view 9876543
✓ main CI · 9876543
Triggered via push about 1 hour ago

JOBS
✓ test in 1m20s (ID 27654321)

ARTIFACTS
test-results (4.2 MB)
coverage-report (1.1 MB)
```

The `ARTIFACTS` block lists what was uploaded.

### Download an artifact

```
$ gh run download 9876543 -n test-results
$ ls test-results/
results.xml  coverage.html
```

`-n` selects which artifact to grab. Without `-n`, all artifacts are downloaded.

### Show workflow billing usage

```
$ gh api /repos/owner/repo/actions/workflows/12345678/timing
{
  "billable": {
    "UBUNTU": { "total_ms": 4200000 },
    "MACOS": { "total_ms": 1500000 },
    "WINDOWS": { "total_ms": 0 }
  },
  "run_duration_ms": 5700000
}
```

`run_duration_ms` is the wall-clock total. `billable.X.total_ms` is what counts against your minutes.

### Watch logs as they stream

```
$ gh run watch 9876543 --interval 5
✓ test (ubuntu-latest, 1.22) [5s elapsed]
  ▶ Set up job
  ▶ actions/checkout@v4
  ▶ actions/setup-go@v5
  ▶ go test ./...
```

Streams in real time. `--interval` controls poll frequency.

### Trigger a `repository_dispatch` from curl

```
$ curl -X POST \
       -H "Authorization: Bearer $GH_TOKEN" \
       -H "Accept: application/vnd.github.v3+json" \
       https://api.github.com/repos/owner/repo/dispatches \
       -d '{"event_type":"deploy","client_payload":{"env":"prod"}}'
```

The repo's workflows triggered by `on: repository_dispatch` (filtered by `types: [deploy]`) will fire. The `client_payload` shows up in `${{ github.event.client_payload }}`.

### Inspect default token permissions

```
$ gh api /repos/owner/repo/actions/permissions
{
  "enabled": true,
  "allowed_actions": "all",
  "selected_actions_url": "https://api.github.com/repos/owner/repo/actions/permissions/selected-actions"
}
```

Shows what actions the repo allows.

## Common Confusions

### "Why doesn't my secret work in a fork PR?"

Forks don't get secrets, on purpose. If they did, anybody could fork your public repo and steal your AWS keys with a malicious workflow. The right way to test fork PRs that need secrets is `pull_request_target` (which gets secrets but runs your default-branch code, not the fork's code), but understand the security model first or you'll re-introduce the hole.

### "Why is my matrix job failing for one combination?"

Probably an OS-specific issue (path separators, file permissions, available tools) or a version-specific issue (deprecated stdlib function, new compiler check). Check the failed job's logs in isolation. Sometimes adding `experimental: true` via `include:` and using `continue-on-error: ${{ matrix.experimental }}` lets you keep an unstable combination as a "we know it's broken, don't fail the build" canary.

### "What's the difference between a step and a job?"

Steps run on the **same runner**, in **order**. They share state (files, env vars).

Jobs run on **separate runners**, in **parallel** by default. They do **not** share state. To pass data between jobs, you have to use artifacts or job outputs.

### "Why did my workflow_run trigger never fire?"

`workflow_run` only triggers from the **default branch**'s workflow file. If you're testing your `workflow_run` workflow on a feature branch, the trigger ignores it. Merge to `main` (or whatever your default is) before you can test it.

### "Why does my GITHUB_TOKEN not have write access?"

Since 2023, the default workflow permissions are read-only. You need to either bump the repo's default permissions in **Settings → Actions → General → Workflow permissions**, or grant per-workflow with `permissions:` blocks. The latter is best practice.

### "Should I use composite or reusable workflows?"

Composite for **bundling steps** (drop-in `uses:` step). Reusable workflow for **bundling jobs** (whole pipeline you call). Composite shares the runner. Reusable workflow gets a fresh runner.

### "Why does the matrix expand to 1 job when I expected 9?"

YAML interpreted your matrix arrays wrong. `os: ubuntu-latest` (no brackets) is a single string, not a list. You need `os: [ubuntu-latest]` (brackets) to make it a list. Same for the `go:` key. Always use brackets for matrix lists, even one-element ones.

### "Why does my workflow not see the file the previous step wrote?"

It does see it, **as long as** that previous step is in the same job. If steps are in different jobs, they're on different runners and don't share files. Use artifacts to pass between jobs.

### "Why is my OIDC step failing with 'no permission to mint id token'?"

Add `permissions: id-token: write` to your workflow or job. By default, `id-token` is `none` (because it grants real cloud access).

### "Why is my schedule running from main even though I'm on a feature branch?"

Schedules always run from the default branch. Cron entries on feature branches do not fire. Same for `workflow_run` triggered from another branch's workflow.

### "Why does `gh run watch` say 'no in-progress runs'?"

The run might have already finished. Or it might have been canceled before it started. Or the workflow file has a syntax error and never started. Check `gh run list --limit 1` to see the latest state.

### "Why does my action keep installing the same dependency every run?"

You don't have caching set up. Add `actions/cache@v4` keyed on your lockfile (`go.sum`, `package-lock.json`, `Cargo.lock`, etc.).

### "Why does my workflow take 10 minutes when it used to take 2?"

Likely cache miss. Look at the cache hit/miss reported by `actions/cache@v4` in the logs. If your cache key is unstable (hash of a file that changes every commit), you'll never get a hit.

### "Why does my `if:` condition never match?"

Common mistake: `if: github.event_name == 'push'` works at job/step level. Inside a step's `with:` block you can't use `if:` (use the action's own input or a wrapper step). Also: `if:` is a string expression — quote it: `if: "github.event_name == 'push'"` if YAML gets confused.

### "Why did my `actions/checkout` not bring my submodules?"

Default is to skip submodules. Add `with: { submodules: true }` (or `recursive`) to fetch them.

### "Why does my workflow run twice on every push to a PR branch?"

You probably have both `push:` and `pull_request:` triggers without filters. Push to the PR branch fires the `push:`, GitHub also re-runs the PR workflow. Filter `push:` to specific branches (like `main` only) and let `pull_request:` handle PR branches.

## Vocabulary

Word | Meaning
---- | -------
GitHub Actions | GitHub's CI/CD service that runs workflows on events.
workflow | A YAML file in `.github/workflows/` describing what to run when.
workflow file | The `.yml` or `.yaml` file with the workflow definition.
.github/workflows | The fixed folder where GitHub looks for workflow files.
jobs | The block listing the chunks of work in a workflow.
steps | The list of tasks inside one job.
run | A single execution of a workflow (one trigger = one run).
uses | A step keyword that loads an action.
with | The block that passes inputs to an action.
env | A block setting environment variables.
if | A keyword that conditionally runs a job or step.
name | Human-readable label for a workflow, job, or step.
runs-on | Which runner type the job uses.
runner | The machine (real or VM) that runs your job.
GitHub-hosted runner | A throwaway VM provided by GitHub.
self-hosted runner | A machine you manage that GitHub sends jobs to.
ARC | Actions Runner Controller — Kubernetes operator for self-hosted runners.
label | A tag on a self-hosted runner for targeting.
on | The trigger block.
push | Trigger on git push.
pull_request | Trigger on PR open/update from the same repo.
pull_request_target | PR trigger that runs from the default branch (gets secrets).
workflow_dispatch | Manual trigger via UI or `gh workflow run`.
workflow_run | Trigger that fires when another workflow finishes.
schedule | Cron-style trigger.
cron | The `m h dom mon dow` time format.
repository_dispatch | API-driven trigger with a custom event type.
release | Trigger on GitHub release events.
deployment | Trigger on the deployment API.
deployment_status | Trigger on deployment status changes.
status | Trigger on commit status changes.
page_build | Trigger when GitHub Pages rebuilds.
registry_package | Trigger when a package is published.
fork | Trigger when somebody forks the repo.
watch | Trigger when somebody stars the repo.
issues | Trigger on issue lifecycle events.
issue_comment | Trigger on comments on issues or PRs.
labels (matrix) | The matrix dimension for OS/version combos.
check_run | Trigger when a check run is created/updated.
check_suite | Trigger when a check suite is created/updated.
secret_scanning_alert | Trigger when secret scanning finds something.
dependabot_alert | Trigger when Dependabot finds a vulnerability.
code_scanning_alert | Trigger when code scanning finds something.
secrets | Encrypted key/value store for sensitive config.
vars | Plain (non-secret) key/value config.
env (block) | Block to set environment variables.
action | A reusable building block referenced via `uses:`.
composite action | An action that bundles multiple steps.
JavaScript action | An action implemented in Node.js.
Docker action | An action that runs in a Docker container.
Marketplace | github.com/marketplace — directory of public actions.
action.yml | The metadata file for an action (name, inputs, runs).
runs | Inside `action.yml`: how the action runs.
main | Inside `action.yml`: the entry point script.
post | Inside `action.yml`: cleanup script after the job.
pre | Inside `action.yml`: setup script before the job.
inputs | Declared parameters an action accepts.
outputs | Values an action emits for later steps.
branding | Inside `action.yml`: icon/color for Marketplace.
GITHUB_TOKEN | Auto-generated short-lived token for the workflow run.
permissions | Block declaring what scopes GITHUB_TOKEN gets.
contents | Permission scope: repo content (read/write).
packages | Permission scope: GitHub Packages.
id-token | Permission scope: required to mint OIDC tokens.
attestations | Permission scope: build attestations.
statuses | Permission scope: commit statuses.
deployments | Permission scope: deployment API.
pages | Permission scope: GitHub Pages.
security-events | Permission scope: code scanning.
actions | Permission scope: Actions API itself.
OIDC | OpenID Connect; how Actions mints short-lived cloud creds.
federated identity | The trust setup that lets a cloud accept OIDC tokens.
workload identity (GCP) | GCP's name for federated identity.
AssumeRoleWithWebIdentity (AWS) | The AWS STS call backing OIDC auth.
service principal (Azure) | Azure's identity for an OIDC-bound app.
strategy | Block configuring matrix and fail-fast.
matrix | The combinatorial expansion of job parameters.
fail-fast | Cancel the rest of the matrix when one fails.
max-parallel | Cap on concurrent matrix jobs.
include | Add extra matrix combinations.
exclude | Remove specific matrix combinations.
environment | A named protected deploy target.
environment protection | Rules guarding an environment.
required reviewers | Humans who must approve before deploy.
wait timer | Mandatory delay before deploy.
deployment branch policy | Which branches can deploy to this env.
concurrency | Block to control overlapping runs.
concurrency group | A name that groups runs for cancellation logic.
cancel-in-progress | Cancel running members of a concurrency group.
jobs.<id>.needs | Declare a job depends on another.
dependency | A job listed under `needs:`.
services (containers) | Sidecar containers (DBs, queues) for tests.
services.<id>.image | Docker image for a service.
services.<id>.ports | Port mappings for a service.
services.<id>.options | Extra Docker run options.
container (job-level) | Run the whole job inside a container.
volumes | Docker volume mappings.
network | Docker network configuration.
port mapping | Map runner ports to service ports.
defaults | Block for default `run:` settings.
defaults.run | Default shell, working dir for `run:` steps.
working-directory | Default cwd for shell steps.
shell | Which shell to run `run:` blocks in.
bash | Default shell on Linux/macOS runners.
pwsh | PowerShell Core — default shell on Windows runners.
python | Run the step's script as Python.
sh | POSIX sh.
cmd | Old Windows command shell.
powershell | Windows PowerShell.
github context | `${{ github.X }}` — info about the run.
env context | `${{ env.X }}` — env vars.
vars context | `${{ vars.X }}` — non-secret config.
secrets context | `${{ secrets.X }}` — secret values.
runner context | `${{ runner.X }}` — runner info.
job context | `${{ job.X }}` — current job state.
steps context | `${{ steps.<id>.X }}` — outputs from prior steps.
matrix context | `${{ matrix.X }}` — current matrix values.
strategy context | `${{ strategy.X }}` — matrix metadata.
inputs context | `${{ inputs.X }}` — for reusable workflows.
needs context | `${{ needs.<id>.outputs.X }}` — upstream job outputs.
runner.os | The runner OS name (`Linux`, `macOS`, `Windows`).
runner.arch | CPU arch (`X64`, `ARM64`).
runner.temp | Path to a temp dir on the runner.
runner.tool_cache | Where preinstalled tools live.
GITHUB_ACTIONS | Env var; always `true` in Actions.
CI | Env var; always `true`.
GITHUB_WORKFLOW | The workflow's name.
GITHUB_RUN_ID | Unique numeric ID for the run.
GITHUB_RUN_NUMBER | Per-workflow auto-incrementing counter.
GITHUB_JOB | Current job's ID.
GITHUB_ACTION | Current action's ID.
GITHUB_ACTOR | Username of who triggered the run.
GITHUB_REPOSITORY | `owner/repo`.
GITHUB_EVENT_NAME | The trigger event (`push`, `pull_request`, ...).
GITHUB_EVENT_PATH | Path to the event payload JSON.
GITHUB_SHA | Commit SHA being built.
GITHUB_REF | Full ref (e.g. `refs/heads/main`).
GITHUB_REF_NAME | Short ref name (e.g. `main`).
GITHUB_REF_TYPE | `branch` or `tag`.
GITHUB_BASE_REF | Base ref of a PR.
GITHUB_HEAD_REF | Head ref of a PR.
GITHUB_SERVER_URL | `https://github.com` (or your enterprise URL).
GITHUB_API_URL | REST API base URL.
GITHUB_GRAPHQL_URL | GraphQL API base URL.
GITHUB_OUTPUT | File for setting step outputs.
GITHUB_STEP_SUMMARY | File for writing job summary markdown.
GITHUB_PATH | File for prepending to PATH.
GITHUB_ENV | File for setting env vars seen by later steps.
RUNNER_OS | Same as `runner.os`.
RUNNER_ARCH | Same as `runner.arch`.
RUNNER_TEMP | Same as `runner.temp`.
RUNNER_TOOL_CACHE | Same as `runner.tool_cache`.
gh CLI | `gh` — official GitHub command-line tool.
act | nektos/act — runs workflows locally in Docker.
actionlint | rhysd/actionlint — static analyzer for workflows.
yamllint | Generic YAML linter.

## Try This

These are little experiments to do on a real repo. Pick one (or all) and try.

### Experiment 1: A minimum workflow

In any repo of yours, create `.github/workflows/hello.yml`:

```yaml
name: Hello
on: [push, workflow_dispatch]
jobs:
  greet:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Hello, $GITHUB_ACTOR! It is $(date)."
```

Push it. Open the Actions tab on github.com. Watch your robot say hello. Click "Run workflow" to fire it manually.

### Experiment 2: Inspect every context

Add a step that dumps the entire `github` context to the log:

```yaml
      - run: echo '${{ toJSON(github) }}'
```

Push, look at the log. Every field is documented in the GitHub Actions context docs. Now you know what's available.

### Experiment 3: Set an output and read it in another step

```yaml
      - id: gen
        run: echo "color=blue" >> $GITHUB_OUTPUT
      - run: echo "The color was ${{ steps.gen.outputs.color }}"
```

This is how steps pass values to each other within a job.

### Experiment 4: Two jobs, one depends on the other

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.v.outputs.version }}
    steps:
      - id: v
        run: echo "version=1.2.3" >> $GITHUB_OUTPUT
  notify:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - run: echo "Built version ${{ needs.build.outputs.version }}"
```

`notify` waits for `build`. `notify` reads `build`'s output via `needs.build.outputs.version`.

### Experiment 5: Run it locally with act

```
$ act -j greet
[Hello/greet] 🚀  Start image=catthehacker/ubuntu:act-latest
[Hello/greet]   ⭐ Run Main echo "Hello, ..."
[Hello/greet] | Hello, you! It is Mon Apr 27 14:22:00 UTC 2026.
[Hello/greet]   ✅  Success - Main echo "..."
```

Same workflow, no GitHub round-trip. You can now iterate without pushing.

### Experiment 6: Cache something

Add caching to a Go workflow:

```yaml
      - uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: ${{ runner.os }}-go-
```

Push twice. The second push should be much faster. Check the cache hit message in the action's log.

### Experiment 7: Make a composite action and use it

Create `.github/actions/say-hi/action.yml`:

```yaml
name: 'Say Hi'
description: 'Greet the user'
inputs:
  name:
    description: 'Who to greet'
    required: true
runs:
  using: 'composite'
  steps:
    - run: echo "Hi, ${{ inputs.name }}!"
      shell: bash
```

In your workflow:

```yaml
      - uses: ./.github/actions/say-hi
        with:
          name: ${{ github.actor }}
```

Now you have your own custom action.

### Experiment 8: Lint your workflow

```
$ actionlint .github/workflows/hello.yml
```

If it says nothing, you're clean. If it complains, fix and re-run.

### Experiment 9: Use OIDC to talk to AWS (advanced)

If you have an AWS account, set up an OIDC provider for GitHub, create an IAM role, then in your workflow:

```yaml
permissions:
  id-token: write
  contents: read
jobs:
  list-buckets:
    runs-on: ubuntu-latest
    steps:
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/MyRole
          aws-region: us-east-1
      - run: aws s3 ls
```

No long-lived secret in your repo. Watch `aws s3 ls` print buckets. This is the secure pattern for cloud deploys.

### Experiment 10: Add a manual approval gate

Define an environment in **Settings → Environments → New environment**, name it `production`, add yourself as a required reviewer. Then:

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production
    steps:
      - run: echo "deploying"
```

Trigger the workflow. The job pauses. You get an email. Click approve. The job continues. This is the primitive every serious deploy pipeline is built on.

## Where to Go Next

- `cs ci-cd github-actions` — dense reference cheat sheet for everyday lookups
- `cs detail ci-cd/github-actions` — internals deep dive (runner architecture, OIDC, queue model)
- `cs ci-cd gitlab-ci` — alternative pipeline syntax
- `cs ci-cd jenkins` — old-school CI server
- `cs ci-cd argocd` — GitOps deploy controller
- `cs orchestration kubernetes` — container orchestration the workflows often deploy to
- `cs orchestration argocd` — GitOps continuous deploy
- `cs ramp-up git-eli5` — what triggers your actions in the first place
- `cs ramp-up docker-eli5` — for container-based jobs and self-hosted runners
- `cs ramp-up kubernetes-eli5` — ARC for self-hosted runners

## See Also

- `ci-cd/github-actions`
- `ci-cd/gitlab-ci`
- `ci-cd/jenkins`
- `orchestration/argocd`
- `vcs/git`
- `vcs/git-worktree`
- `containers/docker`
- `orchestration/kubernetes`
- `orchestration/argocd`
- `ramp-up/git-eli5`
- `ramp-up/docker-eli5`
- `ramp-up/kubernetes-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- docs.github.com/en/actions
- "GitHub Actions in Action" by Kaufmann, De Jong, et al. (Manning, 2024)
- github.com/actions/checkout, setup-go, setup-node, setup-python, cache, upload-artifact, download-artifact
- github.com/nektos/act
- github.com/rhysd/actionlint
- man gh, man gh-workflow, man gh-run, man gh-secret, man gh-variable
- GitHub OIDC docs: docs.github.com/en/actions/deployment/security-hardening-your-deployments
- The Marketplace: github.com/marketplace?type=actions
- GitHub REST API: docs.github.com/en/rest/actions
- Runner images: github.com/actions/runner-images
- Actions Runner Controller (ARC): github.com/actions/actions-runner-controller
