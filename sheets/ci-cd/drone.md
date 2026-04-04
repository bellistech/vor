# Drone (Continuous Integration and Delivery)

Drone is a container-native CI/CD platform that executes pipelines defined in `.drone.yml` using isolated Docker containers for each step, with support for multiple runners, plugins, secrets management, matrix builds, and cron scheduling.

## Pipeline Basics

### Minimal Pipeline

```yaml
kind: pipeline
type: docker
name: default

steps:
  - name: test
    image: golang:1.22
    commands:
      - go test ./...

  - name: build
    image: golang:1.22
    commands:
      - go build -o app .
```

### Pipeline with Services

```yaml
kind: pipeline
type: docker
name: integration

services:
  - name: postgres
    image: postgres:16
    environment:
      POSTGRES_DB: testdb
      POSTGRES_PASSWORD: secret

  - name: redis
    image: redis:7-alpine

steps:
  - name: test
    image: golang:1.22
    environment:
      DATABASE_URL: postgres://postgres:secret@postgres:5432/testdb?sslmode=disable
      REDIS_URL: redis://redis:6379
    commands:
      - go test -tags integration ./...
```

## Conditions and Triggers

### Branch and Event Filtering

```yaml
kind: pipeline
type: docker
name: deploy

trigger:
  branch:
    - main
    - release/*
  event:
    - push
    - tag

steps:
  - name: deploy
    image: alpine/k8s:1.29
    commands:
      - kubectl apply -f manifests/
    when:
      branch:
        - main
      event:
        - push
```

### Step Conditions

```yaml
steps:
  - name: notify-slack
    image: plugins/slack
    settings:
      webhook:
        from_secret: slack_webhook
      channel: builds
    when:
      status:
        - failure
        - success
      event:
        exclude:
          - pull_request
```

## Plugins

### Docker Plugin (Build and Push)

```yaml
steps:
  - name: publish
    image: plugins/docker
    settings:
      repo: registry.example.com/myapp
      tags:
        - latest
        - ${DRONE_TAG}
      dockerfile: Dockerfile
      username:
        from_secret: docker_username
      password:
        from_secret: docker_password
      build_args:
        - VERSION=${DRONE_TAG}
```

## Secrets Management

### CLI Secret Operations

```bash
# Add a secret to a repository
drone secret add \
  --repository org/repo \
  --name docker_password \
  --data 's3cur3p4ss'

# List secrets
drone secret ls --repository org/repo

# Remove a secret
drone secret rm --repository org/repo --name docker_password

# Add organization secret
drone orgsecret add org docker_password 's3cur3p4ss'

# Add secret limited to specific events
drone secret add \
  --repository org/repo \
  --name deploy_key \
  --data @/path/to/key \
  --event push \
  --event tag
```

### Using Secrets in Pipelines

```yaml
steps:
  - name: deploy
    image: alpine
    environment:
      API_KEY:
        from_secret: api_key
      DB_PASSWORD:
        from_secret: db_password
    commands:
      - ./deploy.sh
```

## Matrix Builds

### Build Matrix

```yaml
kind: pipeline
type: docker
name: matrix

steps:
  - name: test
    image: golang:${GO_VERSION}
    commands:
      - go test ./...

  - name: build
    image: golang:${GO_VERSION}
    commands:
      - GOOS=${GOOS} GOARCH=${GOARCH} go build -o app .

matrix:
  include:
    - GO_VERSION: "1.21"
      GOOS: linux
      GOARCH: amd64
    - GO_VERSION: "1.22"
      GOOS: linux
      GOARCH: amd64
    - GO_VERSION: "1.22"
      GOOS: linux
      GOARCH: arm64
```

## Runners

### Docker Runner

```bash
# Install Docker runner
docker run -d \
  --volume=/var/run/docker.sock:/var/run/docker.sock \
  --env=DRONE_RPC_PROTO=https \
  --env=DRONE_RPC_HOST=drone.example.com \
  --env=DRONE_RPC_SECRET=shared-secret \
  --env=DRONE_RUNNER_CAPACITY=4 \
  --env=DRONE_RUNNER_NAME=runner-1 \
  --restart=always \
  --name=drone-runner \
  drone/drone-runner-docker:1
```

## Promotion and Cron

### Promotion (Deploy Targets)

```yaml
kind: pipeline
type: docker
name: deploy

trigger:
  event:
    - promote
  target:
    - production

steps:
  - name: deploy
    image: alpine/k8s:1.29
    commands:
      - kubectl set image deployment/app app=registry/app:${DRONE_COMMIT_SHA:0:8}
```

```bash
# Promote a build to production
drone build promote org/repo 42 production

# Promote with parameters
drone build promote org/repo 42 staging --param=IMAGE_TAG=v1.2.3
```

### Cron Jobs

```bash
# Create a cron job
drone cron add org/repo nightly-build "0 0 * * *" --branch main

# List cron jobs
drone cron ls org/repo

# Remove a cron job
drone cron rm org/repo nightly-build
```

## Jsonnet Extensions

### Using Jsonnet for DRY Pipelines

```jsonnet
// .drone.jsonnet
local Pipeline(name, goVersion) = {
  kind: "pipeline",
  type: "docker",
  name: name,
  steps: [
    {
      name: "test",
      image: "golang:" + goVersion,
      commands: [
        "go test ./...",
      ],
    },
    {
      name: "build",
      image: "golang:" + goVersion,
      commands: [
        "go build -o app .",
      ],
    },
  ],
};

[
  Pipeline("go-1.21", "1.21"),
  Pipeline("go-1.22", "1.22"),
]
```

```bash
# Convert jsonnet to YAML
drone jsonnet --stream
drone jsonnet --stream --source .drone.jsonnet --target .drone.yml
```

## CLI Operations

```bash
# List recent builds / get info / restart
drone build ls org/repo
drone build info org/repo 42
drone build restart org/repo 42

# Approve or decline a blocked build
drone build approve org/repo 42
drone build decline org/repo 42
```

## Tips

- Use `from_secret` for all sensitive values rather than hardcoding them in pipeline definitions
- Leverage services for integration testing with real databases and caches in ephemeral containers
- Use `when: status: [ failure ]` on notification steps to alert only on broken builds
- Use Jsonnet to template repetitive pipelines and eliminate YAML duplication
- Set `DRONE_RUNNER_CAPACITY` to control parallelism on each runner based on available resources
- Pin plugin images to specific tags (e.g., `plugins/docker:20.17`) to avoid unexpected behavior
- Use `trigger.event.exclude` to skip pipelines for pull requests or other event types
- Use promotion events and deploy targets to separate CI from CD with manual gates
- Cache dependencies using host volumes to speed up builds across pipeline runs
- Matrix builds are ideal for testing across Go versions, OS targets, or feature flag combinations

## See Also

- github-actions, gitlab-ci, jenkins, tekton, docker

## References

- [Drone Documentation](https://docs.drone.io/)
- [Drone Plugins Registry](https://plugins.drone.io/)
- [Drone CLI Reference](https://docs.drone.io/cli/)
- [Drone Docker Runner](https://docs.drone.io/runner/docker/overview/)
- [Drone Jsonnet Extension](https://docs.drone.io/pipeline/scripting/jsonnet/)
