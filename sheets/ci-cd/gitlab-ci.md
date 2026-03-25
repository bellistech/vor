# GitLab CI/CD (.gitlab-ci.yml Pipeline Configuration)

Define build, test, and deploy pipelines as code in your GitLab repository with `.gitlab-ci.yml`.

## Pipeline Basics

### Minimal pipeline

```yaml
# .gitlab-ci.yml
stages:
  - build
  - test
  - deploy

build-job:
  stage: build
  script:
    - echo "Building the app"
    - make build

test-job:
  stage: test
  script:
    - echo "Running tests"
    - make test

deploy-job:
  stage: deploy
  script:
    - echo "Deploying"
  only:
    - main
```

## Job Configuration

### Script blocks

```yaml
my-job:
  stage: test
  before_script:
    - apt-get update && apt-get install -y curl  # runs before main script
  script:
    - echo "Main job commands"
    - ./run-tests.sh
  after_script:
    - echo "Cleanup (runs even if script fails)"
    - rm -rf tmp/
```

### Image and services

```yaml
test-job:
  image: node:20-alpine               # Docker image for the job
  services:
    - name: postgres:16
      alias: db                        # accessible as hostname "db"
    - redis:7
  variables:
    POSTGRES_DB: testdb
    POSTGRES_USER: runner
    POSTGRES_PASSWORD: secret
  script:
    - npm ci
    - npm test
```

## Rules and Conditions

### Modern rules syntax (preferred over only/except)

```yaml
deploy-staging:
  stage: deploy
  script:
    - ./deploy.sh staging
  rules:
    - if: $CI_COMMIT_BRANCH == "main"           # branch condition
      when: always
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
      when: manual                                # manual trigger
    - if: $CI_COMMIT_TAG                          # run on tags
    - when: never                                 # default: skip

test-job:
  script:
    - make test
  rules:
    - changes:                                    # only when files change
        - src/**/*
        - tests/**/*
    - if: $CI_PIPELINE_SOURCE == "schedule"       # scheduled pipelines
```

### Legacy only/except

```yaml
deploy-prod:
  script:
    - ./deploy.sh
  only:
    - main
    - tags
  except:
    - schedules
```

## Variables

### Define and use variables

```yaml
variables:
  APP_NAME: "my-app"                   # pipeline-level variable
  DEPLOY_ENV: "staging"

build-job:
  variables:
    BUILD_FLAGS: "--release"           # job-level variable
  script:
    - echo "Building $APP_NAME with $BUILD_FLAGS"
    - echo "CI variables: $CI_COMMIT_SHA $CI_PIPELINE_ID"

# Predefined CI variables (subset):
# $CI_COMMIT_SHA        - full commit SHA
# $CI_COMMIT_BRANCH     - branch name
# $CI_COMMIT_TAG        - tag name (if tagged)
# $CI_PIPELINE_ID       - pipeline ID
# $CI_PROJECT_DIR       - project checkout directory
# $CI_REGISTRY_IMAGE    - container registry image path
# $CI_JOB_TOKEN         - token for API/registry auth
```

## Artifacts

### Save and pass build outputs

```yaml
build-job:
  stage: build
  script:
    - make build
  artifacts:
    paths:
      - build/                         # save the build/ directory
      - binary
    expire_in: 1 week                  # auto-cleanup
    when: always                       # save even on failure

test-job:
  stage: test                          # automatically downloads build-job artifacts
  script:
    - ./build/run-tests

# JUnit test reports
test-job:
  script:
    - go test -v ./... 2>&1 | go-junit-report > report.xml
  artifacts:
    reports:
      junit: report.xml                # displayed in merge request UI
```

## Cache

### Speed up jobs with dependency caching

```yaml
test-job:
  image: node:20
  cache:
    key: ${CI_COMMIT_REF_SLUG}         # cache per-branch
    paths:
      - node_modules/
    policy: pull-push                  # pull: read-only, push: write-only
  script:
    - npm ci
    - npm test

# Multiple caches
build-job:
  cache:
    - key: go-mod-${CI_COMMIT_REF_SLUG}
      paths:
        - .go-cache/
    - key: npm-${CI_COMMIT_REF_SLUG}
      paths:
        - node_modules/
```

## Docker-in-Docker

### Build Docker images in CI

```yaml
build-image:
  image: docker:24
  services:
    - docker:24-dind
  variables:
    DOCKER_TLS_CERTDIR: "/certs"
  before_script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
```

## Environments

### Track deployments

```yaml
deploy-staging:
  stage: deploy
  script:
    - ./deploy.sh staging
  environment:
    name: staging
    url: https://staging.example.com
    on_stop: stop-staging              # link to stop job

stop-staging:
  stage: deploy
  script:
    - ./teardown.sh staging
  environment:
    name: staging
    action: stop
  when: manual
```

## Include and Extends

### Reuse configuration

```yaml
# Include templates from other files or repos
include:
  - local: '.gitlab/ci/test.yml'            # from same repo
  - project: 'group/shared-ci'              # from another project
    ref: main
    file: '/templates/docker-build.yml'
  - template: 'Auto-DevOps.gitlab-ci.yml'   # GitLab templates
  - remote: 'https://example.com/ci.yml'    # remote URL

# Extend a hidden job (template)
.test-template:
  image: node:20
  before_script:
    - npm ci
  cache:
    paths:
      - node_modules/

unit-tests:
  extends: .test-template
  script:
    - npm run test:unit

integration-tests:
  extends: .test-template
  script:
    - npm run test:integration
  services:
    - postgres:16
```

## Parallel and Advanced

### Parallel execution

```yaml
# Run a job N times in parallel (for splitting tests)
rspec:
  stage: test
  parallel: 5
  script:
    - bundle exec rspec --format progress $(./split-tests $CI_NODE_INDEX $CI_NODE_TOTAL)
```

### Retry and timeout

```yaml
flaky-test:
  script:
    - ./run-flaky-test.sh
  retry:
    max: 2
    when:
      - runner_system_failure
      - stuck_or_timeout_failure
  timeout: 10 minutes

# Allow failure (non-blocking)
lint-job:
  script:
    - npm run lint
  allow_failure: true
```

## Pages Deployment

### Deploy static site to GitLab Pages

```yaml
pages:
  stage: deploy
  script:
    - npm run build
    - mv dist/ public/                 # must output to public/
  artifacts:
    paths:
      - public
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
```

## Tips

- Use `needs` for DAG pipelines to run jobs out of stage order when dependencies allow.
- Use `!reference` to extract specific keys from templates without full `extends`.
- Define secrets in Settings > CI/CD > Variables, mark as "Masked" and "Protected."
- Use `workflow: rules:` at the top level to control when pipelines run at all.
- Use `trigger` to start downstream/child pipelines in other projects.
- Lint your `.gitlab-ci.yml` at `CI/CD > Pipelines > CI Lint` in the GitLab UI.
- Use `interruptible: true` on jobs that can be safely cancelled by newer pushes.

## References

- [GitLab CI/CD YAML Reference](https://docs.gitlab.com/ee/ci/yaml/)
- [GitLab CI/CD Overview](https://docs.gitlab.com/ee/ci/)
- [Predefined CI/CD Variables](https://docs.gitlab.com/ee/ci/variables/predefined_variables.html)
- [GitLab CI/CD Examples](https://docs.gitlab.com/ee/ci/examples/)
- [GitLab CI/CD Pipeline Configuration](https://docs.gitlab.com/ee/ci/pipelines/)
- [Rules for Controlling Jobs](https://docs.gitlab.com/ee/ci/jobs/job_control.html)
- [Caching in CI/CD](https://docs.gitlab.com/ee/ci/caching/)
- [Artifacts Configuration](https://docs.gitlab.com/ee/ci/jobs/job_artifacts.html)
- [Docker Integration](https://docs.gitlab.com/ee/ci/docker/using_docker_build.html)
- [Environments and Deployments](https://docs.gitlab.com/ee/ci/environments/)
- [Include and Extend](https://docs.gitlab.com/ee/ci/yaml/includes.html)
- [GitLab CI/CD Lint API](https://docs.gitlab.com/ee/api/lint.html)
