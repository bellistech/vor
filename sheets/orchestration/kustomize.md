# Kustomize (Kubernetes Configuration Management)

Template-free customization tool for Kubernetes manifests using overlays, patches, and transformers to compose and modify YAML resources without forking base configurations.

## Kustomization Structure

### Directory Layout

```
project/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
├── overlays/
│   ├── dev/
│   │   ├── kustomization.yaml
│   │   ├── replica-patch.yaml
│   │   └── env-configmap.yaml
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   └── ingress.yaml
│   └── prod/
│       ├── kustomization.yaml
│       ├── replica-patch.yaml
│       ├── hpa.yaml
│       └── pdb.yaml
```

### Base kustomization.yaml

```yaml
# base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
  - service.yaml
  - configmap.yaml

commonLabels:
  app: myapp
  team: platform

commonAnnotations:
  managed-by: kustomize

namespace: default

namePrefix: myapp-
nameSuffix: ""
```

### Overlay kustomization.yaml

```yaml
# overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base
  - hpa.yaml
  - pdb.yaml

namespace: production

namePrefix: prod-

patches:
  - path: replica-patch.yaml
  - target:
      kind: Deployment
      name: myapp-server
    patch: |-
      - op: replace
        path: /spec/replicas
        value: 5

images:
  - name: myapp
    newName: registry.example.com/myapp
    newTag: v1.2.3

configMapGenerator:
  - name: app-config
    literals:
      - APP_ENV=production
      - LOG_LEVEL=warn
```

## Patches

### Strategic Merge Patch

```yaml
# overlays/prod/replica-patch.yaml
# Merges with matching resource by GVK + name
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp-server
spec:
  replicas: 5
  template:
    spec:
      containers:
      - name: server
        resources:
          limits:
            cpu: "2"
            memory: "2Gi"
          requests:
            cpu: "500m"
            memory: "512Mi"
```

### JSON Patch (RFC 6902)

```yaml
# In kustomization.yaml
patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: myapp-server
    patch: |-
      - op: replace
        path: /spec/replicas
        value: 5
      - op: add
        path: /spec/template/spec/containers/0/env/-
        value:
          name: APP_ENV
          value: production
      - op: remove
        path: /spec/template/spec/containers/0/ports/1
      - op: copy
        from: /spec/template/metadata/labels/app
        path: /spec/template/metadata/annotations/app-name
```

### Patch with Target Selector

```yaml
# Apply patch to multiple resources by label/annotation/kind
patches:
  # All Deployments
  - target:
      kind: Deployment
    patch: |-
      - op: add
        path: /spec/template/spec/securityContext
        value:
          runAsNonRoot: true
          fsGroup: 1000

  # Specific name pattern (regex)
  - target:
      kind: Service
      name: ".*-internal"
    patch: |-
      - op: replace
        path: /spec/type
        value: ClusterIP

  # By label selector
  - target:
      kind: Deployment
      labelSelector: "tier=frontend"
    patch: |-
      - op: replace
        path: /spec/replicas
        value: 3
```

### Inline Patch (Strategic Merge)

```yaml
# Direct inline patch in kustomization.yaml
patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: myapp-server
      spec:
        template:
          spec:
            containers:
            - name: server
              env:
              - name: LOG_LEVEL
                value: debug
```

## Generators

### ConfigMap Generator

```yaml
configMapGenerator:
  # From literals
  - name: app-config
    literals:
      - APP_ENV=production
      - DB_HOST=db.example.com
      - LOG_LEVEL=info

  # From files
  - name: nginx-config
    files:
      - nginx.conf
      - mime.types=custom-mime.types    # Rename key

  # From env file
  - name: env-config
    envs:
      - .env.production

  # With options
  - name: static-config
    literals:
      - key=value
    options:
      disableNameSuffixHash: true       # No hash suffix
      labels:
        config-type: static
```

### Secret Generator

```yaml
secretGenerator:
  - name: db-credentials
    literals:
      - username=admin
      - password=secret123
    type: kubernetes.io/basic-auth

  - name: tls-secret
    files:
      - tls.crt=certs/server.crt
      - tls.key=certs/server.key
    type: kubernetes.io/tls

  - name: app-secrets
    envs:
      - .env.secrets
```

## Transformers

### Image Transformer

```yaml
images:
  - name: myapp                          # Original image name
    newName: registry.example.com/myapp  # New registry/name
    newTag: v1.2.3                        # New tag

  - name: nginx
    newTag: "1.25"                        # Change tag only

  - name: sidecar
    newName: new-sidecar
    digest: sha256:abc123...              # Pin to digest
```

### Label and Annotation Transformers

```yaml
# Apply to all resources
commonLabels:
  app.kubernetes.io/name: myapp
  app.kubernetes.io/part-of: platform
  environment: production

commonAnnotations:
  config.kubernetes.io/managed-by: kustomize
  team: platform-engineering

# Labels also added to selectors (spec.selector.matchLabels)
# Annotations are metadata only
```

### Namespace Transformer

```yaml
# Set namespace for all resources
namespace: production

# Except cluster-scoped resources (automatically excluded):
# - ClusterRole
# - ClusterRoleBinding
# - Namespace
# - PersistentVolume
# - etc.
```

### Replacements (v4.5.0+)

```yaml
# Replace values from one resource to another
replacements:
  - source:
      kind: ConfigMap
      name: app-config
      fieldPath: data.DB_HOST
    targets:
      - select:
          kind: Deployment
          name: myapp-server
        fieldPaths:
          - spec.template.spec.containers.[name=server].env.[name=DB_HOST].value
```

## CLI Usage

```bash
# Preview rendered output
kubectl kustomize overlays/prod/
kustomize build overlays/prod/

# Apply to cluster
kubectl apply -k overlays/prod/
kustomize build overlays/prod/ | kubectl apply -f -

# Diff against live cluster
kubectl diff -k overlays/prod/

# Delete resources managed by kustomization
kubectl delete -k overlays/prod/

# Build with specific load restrictions
kustomize build --load-restrictor LoadRestrictionsNone overlays/prod/

# Edit kustomization
cd overlays/prod/
kustomize edit set image myapp=registry.com/myapp:v2.0
kustomize edit set namespace staging
kustomize edit add resource new-resource.yaml
kustomize edit add label env:staging
kustomize edit set nameprefix staging-
```

## Components (Reusable Mixins)

```yaml
# components/monitoring/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

resources:
  - service-monitor.yaml

patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: myapp-server
      spec:
        template:
          metadata:
            annotations:
              prometheus.io/scrape: "true"
              prometheus.io/port: "9090"

# Use in overlays:
# overlays/prod/kustomization.yaml
components:
  - ../../components/monitoring
  - ../../components/logging
```

## Tips

- Always use `kubectl diff -k` before `kubectl apply -k` to preview changes in production
- ConfigMap and Secret generators append a content hash suffix to force pod restarts on config changes
- Use `disableNameSuffixHash: true` for ConfigMaps referenced by name outside Kubernetes (e.g., scripts)
- Strategic merge patches are simpler for adding/changing fields; JSON patches are needed for arrays and removals
- Keep bases generic and environment-agnostic; push all environment specifics into overlays
- Use `images` transformers instead of patching container image references for cleaner version management
- Components (`kind: Component`) enable reusable feature mixins shared across multiple overlays
- The `replacements` field (replacing deprecated `vars`) supports cross-resource value injection
- Pin image tags or use digests in production overlays for reproducible deployments
- Use `kustomize build` with `--enable-helm` to integrate Helm charts as bases
- Namespace transformer automatically skips cluster-scoped resources
- Test kustomize output in CI by running `kustomize build` and validating with `kubeval` or `kubeconform`

## See Also

kubernetes, helm, argocd, operator

## References

- [Kustomize Documentation](https://kubectl.docs.kubernetes.io/references/kustomize/)
- [Kustomize GitHub](https://github.com/kubernetes-sigs/kustomize)
- [Kubernetes — Managing Resources with Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
- [Kustomize API Reference](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/)
- [JSON Patch RFC 6902](https://datatracker.ietf.org/doc/html/rfc6902)
