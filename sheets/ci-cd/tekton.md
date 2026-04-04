# Tekton (Cloud-Native CI/CD Pipelines)

Kubernetes-native CI/CD framework using Tasks, Pipelines, and Triggers as custom resources, with workspaces for data sharing, built-in artifact management, and event-driven pipeline execution.

## Installation

### Install Tekton Pipelines

```bash
# Install Tekton Pipelines
kubectl apply --filename \
  https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml

# Install Tekton Triggers
kubectl apply --filename \
  https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
kubectl apply --filename \
  https://storage.googleapis.com/tekton-releases/triggers/latest/interceptors.yaml

# Install Tekton Dashboard (optional)
kubectl apply --filename \
  https://storage.googleapis.com/tekton-releases/dashboard/latest/release.yaml

# Verify installation
kubectl get pods -n tekton-pipelines

# Install tkn CLI (macOS)
brew install tektoncd-cli

# Install tkn CLI (Linux)
curl -LO https://github.com/tektoncd/cli/releases/download/v0.37.0/tkn_0.37.0_Linux_x86_64.tar.gz
tar xvzf tkn_0.37.0_Linux_x86_64.tar.gz tkn
sudo mv tkn /usr/local/bin/

# Verify CLI
tkn version
```

## Tasks

### Simple task

```yaml
# A Task defines a series of steps that run sequentially
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: build-and-test
spec:
  params:
    - name: go-version
      type: string
      default: "1.24"
    - name: package
      type: string
      description: Go package to build
  workspaces:
    - name: source
      description: Source code workspace
  steps:
    - name: build
      image: golang:$(params.go-version)
      workingDir: $(workspaces.source.path)
      script: |
        #!/bin/bash
        set -ex
        go build -v ./$(params.package)/...

    - name: test
      image: golang:$(params.go-version)
      workingDir: $(workspaces.source.path)
      script: |
        #!/bin/bash
        set -ex
        go test -v -race -count=1 ./$(params.package)/...

    - name: lint
      image: golangci/golangci-lint:latest
      workingDir: $(workspaces.source.path)
      script: |
        #!/bin/bash
        golangci-lint run ./$(params.package)/...
```

### Task with results

```yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: git-version
spec:
  workspaces:
    - name: source
  results:
    - name: commit-sha
      description: Git commit SHA
    - name: version
      description: Semantic version
  steps:
    - name: get-version
      image: alpine/git:latest
      workingDir: $(workspaces.source.path)
      script: |
        #!/bin/sh
        git rev-parse HEAD | tr -d '\n' | tee $(results.commit-sha.path)
        git describe --tags --always | tr -d '\n' | tee $(results.version.path)
```

### Run a task

```bash
# Start a TaskRun from CLI
tkn task start build-and-test \
  --param go-version=1.24 \
  --param package=./cmd/server \
  --workspace name=source,claimName=source-pvc

# Start with last parameters
tkn task start build-and-test --last

# List task runs
tkn taskrun list

# View task run logs
tkn taskrun logs build-and-test-run-abc123 -f

# Describe a task run
tkn taskrun describe build-and-test-run-abc123
```

```yaml
# TaskRun YAML
apiVersion: tekton.dev/v1
kind: TaskRun
metadata:
  generateName: build-and-test-
spec:
  taskRef:
    name: build-and-test
  params:
    - name: go-version
      value: "1.24"
    - name: package
      value: "./cmd/server"
  workspaces:
    - name: source
      persistentVolumeClaim:
        claimName: source-pvc
```

## Pipelines

### Multi-task pipeline

```yaml
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: ci-pipeline
spec:
  params:
    - name: repo-url
      type: string
    - name: revision
      type: string
      default: main
    - name: image-name
      type: string
  workspaces:
    - name: shared-workspace
    - name: docker-credentials
  tasks:
    # Step 1: Clone repository
    - name: clone
      taskRef:
        name: git-clone
      params:
        - name: url
          value: $(params.repo-url)
        - name: revision
          value: $(params.revision)
      workspaces:
        - name: output
          workspace: shared-workspace

    # Step 2: Get version (parallel with tests)
    - name: version
      taskRef:
        name: git-version
      runAfter: ["clone"]
      workspaces:
        - name: source
          workspace: shared-workspace

    # Step 3: Run tests (parallel with version)
    - name: test
      taskRef:
        name: build-and-test
      runAfter: ["clone"]
      params:
        - name: package
          value: "./..."
      workspaces:
        - name: source
          workspace: shared-workspace

    # Step 4: Build and push image (after tests pass)
    - name: build-image
      taskRef:
        name: kaniko
      runAfter: ["test", "version"]
      params:
        - name: IMAGE
          value: $(params.image-name):$(tasks.version.results.version)
      workspaces:
        - name: source
          workspace: shared-workspace
        - name: dockerconfig
          workspace: docker-credentials

    # Step 5: Deploy
    - name: deploy
      taskRef:
        name: kubectl-deploy
      runAfter: ["build-image"]
      params:
        - name: image
          value: $(params.image-name):$(tasks.version.results.version)
```

### Run a pipeline

```bash
# Start a pipeline run
tkn pipeline start ci-pipeline \
  --param repo-url=https://github.com/org/repo.git \
  --param revision=main \
  --param image-name=registry.example.com/app \
  --workspace name=shared-workspace,claimName=ci-pvc \
  --workspace name=docker-credentials,secret=docker-creds

# List pipeline runs
tkn pipelinerun list

# View pipeline run logs
tkn pipelinerun logs ci-pipeline-run-abc123 -f

# Cancel a running pipeline
tkn pipelinerun cancel ci-pipeline-run-abc123

# Delete old pipeline runs
tkn pipelinerun delete --keep 5
```

## Workspaces

### Workspace types

```yaml
# PersistentVolumeClaim — shared storage across tasks
workspaces:
  - name: source
    persistentVolumeClaim:
      claimName: source-pvc

# EmptyDir — ephemeral, per-TaskRun
workspaces:
  - name: temp
    emptyDir: {}

# ConfigMap — read-only config
workspaces:
  - name: config
    configMap:
      name: app-config

# Secret — credentials
workspaces:
  - name: credentials
    secret:
      secretName: docker-creds

# VolumeClaimTemplate — auto-provisioned PVC per run
workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 1Gi
```

## Triggers

### Event-driven pipeline execution

```yaml
# TriggerTemplate — creates PipelineRun from event data
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerTemplate
metadata:
  name: ci-template
spec:
  params:
    - name: git-revision
    - name: git-repo-url
  resourcetemplates:
    - apiVersion: tekton.dev/v1
      kind: PipelineRun
      metadata:
        generateName: ci-run-
      spec:
        pipelineRef:
          name: ci-pipeline
        params:
          - name: repo-url
            value: $(tt.params.git-repo-url)
          - name: revision
            value: $(tt.params.git-revision)
        workspaces:
          - name: shared-workspace
            volumeClaimTemplate:
              spec:
                accessModes: ["ReadWriteOnce"]
                resources:
                  requests:
                    storage: 1Gi

---
# TriggerBinding — extracts data from webhook payload
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerBinding
metadata:
  name: github-push
spec:
  params:
    - name: git-revision
      value: $(body.head_commit.id)
    - name: git-repo-url
      value: $(body.repository.clone_url)

---
# EventListener — receives webhooks
apiVersion: triggers.tekton.dev/v1beta1
kind: EventListener
metadata:
  name: github-listener
spec:
  triggers:
    - name: github-push
      bindings:
        - ref: github-push
      template:
        ref: ci-template
      interceptors:
        - ref:
            name: "github"
          params:
            - name: secretRef
              value:
                secretName: github-webhook-secret
                secretKey: token
            - name: eventTypes
              value: ["push"]
```

```bash
# Expose the EventListener
kubectl get svc el-github-listener

# Test trigger locally
curl -X POST http://el-github-listener:8080 \
  -H "Content-Type: application/json" \
  -d '{"head_commit":{"id":"abc123"},"repository":{"clone_url":"https://github.com/org/repo"}}'
```

## Common Tasks from Tekton Hub

```bash
# Install git-clone task from Tekton Hub
tkn hub install task git-clone

# Install kaniko (container image build)
tkn hub install task kaniko

# Install kubectl-deploy
tkn hub install task kubernetes-actions

# Search Tekton Hub
tkn hub search --tags build
tkn hub search --tags git

# List installed tasks
tkn task list
```

## Tips

- Use `volumeClaimTemplate` instead of named PVCs for isolated, auto-cleaned workspace per run.
- Tasks run steps sequentially; use Pipeline `runAfter` to control inter-task ordering and parallelism.
- Results pass data between tasks — use them for commit SHAs, version strings, and image digests.
- Use `finally` tasks in Pipelines for cleanup and notification regardless of success or failure.
- Install commonly used tasks from Tekton Hub rather than writing them from scratch.
- Set resource requests and limits on steps to prevent noisy neighbor problems in shared clusters.
- Use `tkn pipelinerun delete --keep N` to clean up old runs and prevent etcd bloat.
- Sidecar containers in Tasks are useful for running services needed during steps (databases, mock servers).
- Use `when` expressions in Pipeline tasks for conditional execution based on parameters or results.
- Enable Tekton Chains for supply chain security (automated signing and attestation of artifacts).
- Debug failing steps with `tkn taskrun logs` and check pod events with `kubectl describe pod`.
- Use `securityContext.runAsNonRoot: true` in steps for security-hardened pipelines.

## See Also

dagger, github-actions, gitlab-ci, jenkins, kubernetes

## References

- [Tekton Documentation](https://tekton.dev/docs/)
- [Tekton Pipelines](https://tekton.dev/docs/pipelines/)
- [Tekton Triggers](https://tekton.dev/docs/triggers/)
- [Tekton Hub](https://hub.tekton.dev/)
- [Tekton CLI Reference](https://tekton.dev/docs/cli/)
- [Tekton GitHub Repository](https://github.com/tektoncd/pipeline)
- [Tekton Chains](https://tekton.dev/docs/chains/)
