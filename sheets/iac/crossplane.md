# crossplane (Kubernetes-native IaC)

Crossplane extends Kubernetes with custom resource definitions to provision and manage cloud infrastructure from any provider, using Compositions to define reusable platform abstractions that let application teams self-service infrastructure through standard kubectl workflows.

## Installation

### Setup Crossplane

```bash
# Install Crossplane via Helm
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace --wait

# Verify installation
kubectl get pods -n crossplane-system
kubectl api-resources | grep crossplane

# Install Crossplane CLI
curl -sL https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh | sh
sudo mv crossplane /usr/local/bin/crank
```

## Providers

### Install and Configure

```bash
# Install AWS provider
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-aws
spec:
  package: xpkg.upbound.io/upbound/provider-aws-s3:v1.1.0
EOF

# Check provider status
kubectl get providers
kubectl wait --for=condition=Healthy provider/provider-aws --timeout=300s
```

### ProviderConfig

```yaml
# AWS credentials secret
apiVersion: v1
kind: Secret
metadata:
  name: aws-creds
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    [default]
    aws_access_key_id = AKIAIOSFODNN7EXAMPLE
    aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
---
apiVersion: aws.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: aws-creds
      key: credentials
```

## Managed Resources

### Direct Cloud Resource Management

```yaml
# S3 Bucket
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
metadata:
  name: my-app-bucket
spec:
  forProvider:
    region: us-east-1
    tags:
      Environment: production
  providerConfigRef:
    name: default
```

```bash
# Apply and check managed resources
kubectl apply -f bucket.yaml
kubectl get bucket my-app-bucket -o yaml
kubectl wait --for=condition=Ready bucket/my-app-bucket --timeout=300s
kubectl delete bucket my-app-bucket
```

## Composite Resources (XRs)

### CompositeResourceDefinition (XRD)

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xdatabases.platform.example.com
spec:
  group: platform.example.com
  names:
    kind: XDatabase
    plural: xdatabases
  claimNames:
    kind: Database
    plural: databases
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                parameters:
                  type: object
                  properties:
                    size:
                      type: string
                      enum: [small, medium, large]
                    engine:
                      type: string
                      enum: [postgres, mysql]
                  required: [size]
```

### Composition

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: xdatabases.aws.platform.example.com
  labels:
    provider: aws
spec:
  compositeTypeRef:
    apiVersion: platform.example.com/v1alpha1
    kind: XDatabase
  resources:
    - name: rds-instance
      base:
        apiVersion: rds.aws.upbound.io/v1beta1
        kind: Instance
        spec:
          forProvider:
            region: us-east-1
            engine: postgres
            publiclyAccessible: false
            skipFinalSnapshot: true
          writeConnectionSecretToRef:
            namespace: crossplane-system
      patches:
        - type: FromCompositeFieldPath
          fromFieldPath: spec.parameters.size
          toFieldPath: spec.forProvider.instanceClass
          transforms:
            - type: map
              map:
                small: db.t3.micro
                medium: db.t3.medium
                large: db.t3.large
      connectionDetails:
        - type: FromConnectionSecretKey
          fromConnectionSecretKey: endpoint
          name: host
```

### Claims

```yaml
# Application team creates a claim (no cloud knowledge needed)
apiVersion: platform.example.com/v1alpha1
kind: Database
metadata:
  name: myapp-db
  namespace: default
spec:
  parameters:
    size: medium
    engine: postgres
  compositionSelector:
    matchLabels:
      provider: aws
  writeConnectionSecretToRef:
    name: myapp-db-connection
```

```bash
# Check claim, XR, and managed resources
kubectl get database myapp-db
kubectl get xdatabase
kubectl get managed
kubectl get secret myapp-db-connection -o yaml
```

## Composition Functions

### Pipeline Mode

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: xdatabases.fn.platform.example.com
spec:
  compositeTypeRef:
    apiVersion: platform.example.com/v1alpha1
    kind: XDatabase
  mode: Pipeline
  pipeline:
    - step: patch-and-transform
      functionRef:
        name: function-patch-and-transform
      input:
        apiVersion: pt.fn.crossplane.io/v1beta1
        kind: Resources
        resources:
          - name: rds-instance
            base:
              apiVersion: rds.aws.upbound.io/v1beta1
              kind: Instance
              spec:
                forProvider:
                  engine: postgres
            patches:
              - type: FromCompositeFieldPath
                fromFieldPath: spec.parameters.region
                toFieldPath: spec.forProvider.region
    - step: auto-ready
      functionRef:
        name: function-auto-detect-ready
```

```bash
# Install composition function
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-patch-and-transform
spec:
  package: xpkg.upbound.io/crossplane-contrib/function-patch-and-transform:v0.5.0
EOF

kubectl get functions
```

## Package Management

### Build and Push Configurations

```bash
# crossplane.yaml (package metadata)
cat > crossplane.yaml << 'EOF'
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: my-platform
spec:
  crossplane:
    version: ">=v1.14.0"
  dependsOn:
    - provider: xpkg.upbound.io/upbound/provider-aws-s3
      version: ">=v1.0.0"
EOF

# Build and push package
crank xpkg build --package-root . --output my-platform.xpkg
crank xpkg push -f my-platform.xpkg xpkg.upbound.io/myorg/my-platform:v1.0.0
```

## Troubleshooting

### Debug Commands

```bash
# Check all Crossplane resources
kubectl get crossplane

# Check provider logs
kubectl logs -n crossplane-system -l pkg.crossplane.io/revision \
  -c package-runtime --tail=100

# Crossplane controller logs
kubectl logs -n crossplane-system deployment/crossplane --tail=100

# Resource status conditions
kubectl get bucket my-bucket -o jsonpath='{.status.conditions}' | jq .

# Check sync status of all managed resources
kubectl get managed -o custom-columns=\
NAME:.metadata.name,SYNCED:.status.conditions[0].status,READY:.status.conditions[1].status
```

## Tips

- Use Claims (not XRs directly) to give application teams a namespace-scoped, simplified interface
- Map enum values (small/medium/large) to provider-specific instance types in Composition patches
- Write connection secrets to the consuming namespace so apps use them directly as Kubernetes Secrets
- Use composition selectors with labels to support multi-cloud: same claim, different provider
- Install provider families (`provider-family-aws`) for broad resource coverage with a single package
- Composition Functions (Pipeline mode) replace the legacy Resources mode and are more flexible
- Use `kubectl wait --for=condition=Ready` in CI/CD to gate deployments on infrastructure readiness
- Package Configurations as OCI images for versioned, distributable platform definitions
- Debug provider issues by checking both the managed resource events and provider pod logs
- Set `skipFinalSnapshot: true` on RDS for dev/test but never for production databases

## See Also

- terraform, pulumi, helm, kubectl, argocd, kustomize

## References

- [Crossplane Documentation](https://docs.crossplane.io/)
- [Crossplane Concepts](https://docs.crossplane.io/latest/concepts/)
- [Upbound Marketplace](https://marketplace.upbound.io/)
- [Crossplane Composition Functions](https://docs.crossplane.io/latest/concepts/composition-functions/)
- [Crossplane GitHub](https://github.com/crossplane/crossplane)
