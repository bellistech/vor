# Argo CD (GitOps Continuous Delivery)

Declarative GitOps continuous delivery tool for Kubernetes that synchronizes application state from Git repositories to clusters, with automated drift detection, sync waves, health checks, and rollback.

## Architecture

### Components

```
argocd-server         # API server + Web UI (gRPC + REST)
argocd-repo-server    # Git repo operations, manifest generation
argocd-application-controller   # Reconciliation loop, sync, health
argocd-dex-server     # OIDC/SSO authentication (optional)
argocd-redis          # Caching layer
argocd-notifications  # Event-driven notifications (optional)
argocd-applicationset-controller  # Dynamic Application generation
```

### GitOps Flow

```
Developer → Git Push → Repository
                          │
              argocd-repo-server (clone, render)
                          │
              argocd-application-controller
                    ├── Compare (desired vs live)
                    ├── Sync (apply if OutOfSync)
                    └── Health Check (monitor)
                          │
              Kubernetes Cluster(s)
```

## Application CRD

### Basic Application

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: myapp
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io    # Cascade delete
spec:
  project: default

  source:
    repoURL: https://github.com/org/repo.git
    targetRevision: main
    path: k8s/overlays/prod

  destination:
    server: https://kubernetes.default.svc
    namespace: production

  syncPolicy:
    automated:
      prune: true              # Delete resources removed from Git
      selfHeal: true           # Revert manual cluster changes
      allowEmpty: false        # Prevent sync with zero resources
    syncOptions:
      - CreateNamespace=true
      - PrunePropagationPolicy=foreground
      - PruneLast=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

### Kustomize Source

```yaml
spec:
  source:
    repoURL: https://github.com/org/repo.git
    targetRevision: main
    path: k8s/overlays/prod
    kustomize:
      namePrefix: prod-
      nameSuffix: ""
      images:
        - myapp=registry.com/myapp:v1.2.3
      commonLabels:
        env: production
      commonAnnotations:
        team: platform
```

### Helm Source

```yaml
spec:
  source:
    repoURL: https://charts.example.com
    chart: myapp
    targetRevision: 1.2.3
    helm:
      releaseName: myapp-prod
      values: |
        replicaCount: 3
        image:
          repository: registry.com/myapp
          tag: v1.2.3
        resources:
          limits:
            cpu: "2"
            memory: 2Gi
      valueFiles:
        - values-prod.yaml
      parameters:
        - name: service.type
          value: LoadBalancer
```

### Multi-Source Application

```yaml
spec:
  sources:
    - repoURL: https://charts.example.com
      chart: myapp
      targetRevision: 1.2.3
      helm:
        valueFiles:
          - $values/overlays/prod/values.yaml
    - repoURL: https://github.com/org/config.git
      targetRevision: main
      ref: values    # Referenced as $values above
```

## Sync Waves and Hooks

### Sync Waves (Ordered Deployment)

```yaml
# Wave -1: Namespace and RBAC (applied first)
apiVersion: v1
kind: Namespace
metadata:
  name: myapp
  annotations:
    argocd.argoproj.io/sync-wave: "-1"

---
# Wave 0: ConfigMaps and Secrets (default wave)
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  annotations:
    argocd.argoproj.io/sync-wave: "0"

---
# Wave 1: Database migration job
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
  annotations:
    argocd.argoproj.io/sync-wave: "1"
    argocd.argoproj.io/hook: Sync
    argocd.argoproj.io/hook-delete-policy: BeforeHookCreation

---
# Wave 2: Application deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  annotations:
    argocd.argoproj.io/sync-wave: "2"
```

### Hook Types

```yaml
# Resource hooks — run at specific sync phases
annotations:
  argocd.argoproj.io/hook: PreSync      # Before sync
  argocd.argoproj.io/hook: Sync         # During sync
  argocd.argoproj.io/hook: PostSync     # After sync
  argocd.argoproj.io/hook: SyncFail     # On sync failure
  argocd.argoproj.io/hook: Skip         # Skip this resource

# Hook deletion policies
  argocd.argoproj.io/hook-delete-policy: HookSucceeded
  argocd.argoproj.io/hook-delete-policy: HookFailed
  argocd.argoproj.io/hook-delete-policy: BeforeHookCreation
```

## Health Checks

### Built-in Health Assessment

```
Healthy      # Resource is operating normally
Progressing  # Resource is not yet healthy but making progress
Degraded     # Resource has failed or is in an error state
Suspended    # Resource is paused/suspended
Missing      # Resource does not exist in cluster
Unknown      # Health status cannot be determined
```

### Custom Health Check (Lua)

```yaml
# argocd-cm ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
  namespace: argocd
data:
  resource.customizations.health.mycrd.example.com_MyResource: |
    hs = {}
    if obj.status ~= nil then
      if obj.status.phase == "Running" then
        hs.status = "Healthy"
        hs.message = "Running normally"
      elseif obj.status.phase == "Pending" then
        hs.status = "Progressing"
        hs.message = "Waiting for resources"
      else
        hs.status = "Degraded"
        hs.message = obj.status.message or "Unknown error"
      end
    end
    return hs
```

## CLI Operations

### Application Management

```bash
# Login
argocd login argocd.example.com --grpc-web

# List applications
argocd app list

# Get application details
argocd app get myapp
argocd app get myapp --output json

# Create application
argocd app create myapp \
  --repo https://github.com/org/repo.git \
  --path k8s/overlays/prod \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace production \
  --sync-policy automated \
  --auto-prune \
  --self-heal

# Sync application
argocd app sync myapp
argocd app sync myapp --prune
argocd app sync myapp --resource apps:Deployment:myapp-server
argocd app sync myapp --revision v1.2.3

# Diff
argocd app diff myapp

# Rollback
argocd app rollback myapp REVISION_ID
argocd app history myapp

# Delete
argocd app delete myapp
argocd app delete myapp --cascade    # Delete app + cluster resources
```

### Cluster and Project Management

```bash
# Add cluster
argocd cluster add my-cluster-context

# List clusters
argocd cluster list

# Create project
argocd proj create myproject \
  --src https://github.com/org/* \
  --dest https://kubernetes.default.svc,production \
  --dest https://kubernetes.default.svc,staging
```

## RBAC

### Project-Based Access Control

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: team-frontend
  namespace: argocd
spec:
  description: Frontend team project

  sourceRepos:
    - 'https://github.com/org/frontend-*'

  destinations:
    - namespace: 'frontend-*'
      server: https://kubernetes.default.svc

  clusterResourceWhitelist:
    - group: ''
      kind: Namespace

  namespaceResourceBlacklist:
    - group: ''
      kind: ResourceQuota
    - group: ''
      kind: LimitRange

  roles:
    - name: developer
      description: Frontend developer
      policies:
        - p, proj:team-frontend:developer, applications, get, team-frontend/*, allow
        - p, proj:team-frontend:developer, applications, sync, team-frontend/*, allow
      groups:
        - frontend-devs    # SSO group
```

### RBAC Policy (argocd-rbac-cm)

```csv
# Format: p, subject, resource, action, object, effect
p, role:admin, applications, *, */*, allow
p, role:readonly, applications, get, */*, allow
p, role:team-lead, applications, sync, myproject/*, allow
p, role:team-lead, applications, override, myproject/*, allow

# Group bindings
g, admin-group, role:admin
g, dev-group, role:readonly
```

## ApplicationSet

### Git Generator

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: cluster-apps
  namespace: argocd
spec:
  generators:
    - git:
        repoURL: https://github.com/org/config.git
        revision: main
        directories:
          - path: apps/*
  template:
    metadata:
      name: '{{path.basename}}'
    spec:
      project: default
      source:
        repoURL: https://github.com/org/config.git
        targetRevision: main
        path: '{{path}}'
      destination:
        server: https://kubernetes.default.svc
        namespace: '{{path.basename}}'
```

## Tips

- Enable automated sync with `selfHeal: true` only after validating sync behavior in manual mode
- Use sync waves to enforce deployment ordering: CRDs before operators, migrations before apps
- The `PreSync` hook is ideal for database migrations; use `hook-delete-policy: BeforeHookCreation` to clean up
- Set `prune: true` cautiously; it deletes cluster resources removed from Git (use `PruneLast` for safety)
- Use AppProjects to enforce source repo, destination namespace, and cluster access boundaries per team
- The multi-source feature allows separating Helm charts from values files across repositories
- ApplicationSets auto-generate Applications from generators (Git directories, cluster lists, pull requests)
- Monitor sync status with `argocd app wait myapp --health` in CI/CD pipelines
- Use `argocd app diff` before syncing to preview what will change in the cluster
- Resource hooks (PreSync/PostSync) are Job or Pod resources that run during sync phases
- Configure notifications (Slack, email) for sync failures and degraded health via argocd-notifications
- Use `--grpc-web` flag when connecting through ingress controllers that don't support gRPC

## See Also

kubernetes, kustomize, helm, operator

## References

- [Argo CD Documentation](https://argo-cd.readthedocs.io/)
- [Argo CD GitHub](https://github.com/argoproj/argo-cd)
- [ApplicationSet Documentation](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/)
- [Argo CD Best Practices](https://argo-cd.readthedocs.io/en/stable/operator-manual/best_practices/)
- [GitOps Principles (OpenGitOps)](https://opengitops.dev/)
