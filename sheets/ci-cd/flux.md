# Flux (GitOps Toolkit for Kubernetes)

Flux v2 is a set of continuous delivery solutions for Kubernetes that keeps clusters in sync with sources of configuration (Git repositories, Helm repositories, S3-compatible buckets) using a reconciliation loop, supporting multi-tenancy, notifications, and image automation.

## Installation

### Install Flux CLI

```bash
# Install flux CLI (macOS)
brew install fluxcd/tap/flux

# Install flux CLI (Linux)
curl -s https://fluxcd.io/install.sh | sudo bash

# Verify CLI and cluster prerequisites
flux check --pre

# Enable shell completions (bash)
echo 'source <(flux completion bash)' >> ~/.bashrc

# Enable shell completions (zsh)
echo 'source <(flux completion zsh)' >> ~/.zshrc
```

### Bootstrap Flux on a Cluster

```bash
# Bootstrap with GitHub (creates repo if needed)
flux bootstrap github \
  --owner=my-org \
  --repository=fleet-infra \
  --branch=main \
  --path=clusters/production \
  --personal

# Bootstrap with GitLab
flux bootstrap gitlab \
  --owner=my-group \
  --repository=fleet-infra \
  --branch=main \
  --path=clusters/production

# Bootstrap with generic Git server
flux bootstrap git \
  --url=ssh://git@git.example.com/fleet-infra.git \
  --branch=main \
  --path=clusters/staging

# Uninstall Flux from a cluster
flux uninstall --silent
```

## Sources

### GitRepository

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  interval: 5m
  url: https://github.com/stefanprodan/podinfo
  ref:
    branch: master
  secretRef:
    name: git-credentials
  ignore: |
    # exclude all
    /*
    # include deploy dir
    !/deploy
```

### HelmRepository

```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: bitnami
  namespace: flux-system
spec:
  interval: 1h
  url: https://charts.bitnami.com/bitnami
  type: default
```

### Bucket (S3-Compatible)

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: Bucket
metadata:
  name: manifests
  namespace: flux-system
spec:
  interval: 10m
  provider: aws
  bucketName: my-manifests
  endpoint: s3.amazonaws.com
  region: us-east-1
  secretRef:
    name: s3-credentials
```

## Kustomization Controller

### Flux Kustomization

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: app
  namespace: flux-system
spec:
  interval: 10m
  targetNamespace: default
  sourceRef:
    kind: GitRepository
    name: podinfo
  path: ./deploy
  prune: true
  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: podinfo
      namespace: default
  timeout: 5m
  dependsOn:
    - name: infrastructure
```

### Variable Substitution

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: app
  namespace: flux-system
spec:
  interval: 10m
  sourceRef:
    kind: GitRepository
    name: fleet-infra
  path: ./apps/staging
  postBuild:
    substitute:
      cluster_env: staging
      domain: staging.example.com
    substituteFrom:
      - kind: ConfigMap
        name: cluster-settings
```

## Helm Controller

### HelmRelease

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: redis
  namespace: default
spec:
  interval: 30m
  chart:
    spec:
      chart: redis
      version: "18.x"
      sourceRef:
        kind: HelmRepository
        name: bitnami
        namespace: flux-system
  values:
    architecture: standalone
    auth:
      enabled: false
  upgrade:
    remediation:
      retries: 3
  rollback:
    cleanupOnFail: true
```

## Image Automation

### ImageRepository + ImagePolicy

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageRepository
metadata:
  name: podinfo
  namespace: flux-system
spec:
  image: ghcr.io/stefanprodan/podinfo
  interval: 5m
---
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: podinfo
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: podinfo
  policy:
    semver:
      range: ">=5.0.0"
```

### ImageUpdateAutomation

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImageUpdateAutomation
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 30m
  sourceRef:
    kind: GitRepository
    name: fleet-infra
  git:
    checkout:
      ref:
        branch: main
    commit:
      author:
        name: fluxbot
        email: flux@example.com
    push:
      branch: main
  update:
    path: ./clusters/production
    strategy: Setters
```

## Notifications

### Alert + Provider

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Provider
metadata:
  name: slack
  namespace: flux-system
spec:
  type: slack
  channel: flux-alerts
  secretRef:
    name: slack-webhook
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Alert
metadata:
  name: on-call
  namespace: flux-system
spec:
  providerRef:
    name: slack
  eventSeverity: error
  eventSources:
    - kind: Kustomization
      name: "*"
    - kind: HelmRelease
      name: "*"
```

## CLI Operations

### Common Commands

```bash
# Check Flux status
flux get all -A

# Reconcile a source immediately
flux reconcile source git flux-system

# Reconcile a kustomization
flux reconcile kustomization app --with-source

# Reconcile a helm release
flux reconcile helmrelease redis

# Suspend reconciliation
flux suspend kustomization app

# Resume reconciliation
flux resume kustomization app

# View Flux logs
flux logs --all-namespaces --level=error

# Export Flux resources to YAML
flux export source git --all > sources.yaml
flux export kustomization --all > kustomizations.yaml

# Create a source from CLI
flux create source git my-app \
  --url=https://github.com/org/repo \
  --branch=main \
  --interval=5m

# Diff local vs cluster
flux diff kustomization app --path ./deploy
```

## Tips

- Always set `prune: true` on Kustomizations so deleted manifests are garbage collected from the cluster
- Use `dependsOn` to order deployments: infrastructure first, then apps, then monitoring
- Pin Helm chart versions with semver ranges like `18.x` to avoid unexpected major upgrades
- Use `flux diff` before pushing to preview what changes will apply to the cluster
- Set up Slack or Teams alerts via the notification controller to catch reconciliation failures early
- Use `substituteFrom` with ConfigMaps to share environment-specific values across Kustomizations
- For image automation, add `# {"$imagepolicy": "flux-system:podinfo"}` markers in your YAML files
- Run `flux check` after upgrades to verify all controllers are healthy and compatible
- Use `--export` flag with `flux create` commands to generate YAML without applying to the cluster
- Separate infrastructure and app Kustomizations for independent reconciliation cycles

## See Also

- tekton, github-actions, helm, kustomize, argocd

## References

- [Flux Documentation](https://fluxcd.io/flux/)
- [Flux GitHub Repository](https://github.com/fluxcd/flux2)
- [GitOps Toolkit Components](https://fluxcd.io/flux/components/)
- [Flux Multi-Tenancy Guide](https://fluxcd.io/flux/installation/configuration/multitenancy/)
- [Image Automation Guide](https://fluxcd.io/flux/guides/image-update/)
