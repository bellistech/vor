# Kyverno

Kubernetes-native policy engine that uses YAML to validate, mutate, generate, and verify images without learning a new language.

## Installation

```bash
# Install via Helm
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update
helm install kyverno kyverno/kyverno \
  --namespace kyverno \
  --create-namespace \
  --set replicaCount=3

# Install with policy reporter for visibility
helm install kyverno-policies kyverno/kyverno-policies \
  --namespace kyverno

# Verify installation
kubectl get pods -n kyverno
kubectl get crd | grep kyverno
```

## Validate Rules

```bash
# Require resource limits on all containers
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-resource-limits
spec:
  validationFailureAction: Enforce
  background: true
  rules:
  - name: check-resource-limits
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "All containers must have CPU and memory limits defined."
      pattern:
        spec:
          containers:
          - resources:
              limits:
                memory: "?*"
                cpu: "?*"
EOF

# Disallow latest tag
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: disallow-latest-tag
spec:
  validationFailureAction: Enforce
  rules:
  - name: validate-image-tag
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "Using ':latest' tag is not allowed. Specify an explicit tag."
      pattern:
        spec:
          containers:
          - image: "!*:latest"
          initContainers:
          - image: "!*:latest"
EOF

# Restrict host namespaces
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: disallow-host-namespaces
spec:
  validationFailureAction: Enforce
  rules:
  - name: host-namespaces
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "Host namespaces (hostPID, hostIPC, hostNetwork) are not allowed."
      pattern:
        spec:
          =(hostPID): false
          =(hostIPC): false
          =(hostNetwork): false
EOF

# Namespace-scoped policy
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: Policy
metadata:
  name: require-app-label
  namespace: production
spec:
  validationFailureAction: Enforce
  rules:
  - name: check-app-label
    match:
      any:
      - resources:
          kinds:
          - Deployment
          - StatefulSet
    validate:
      message: "The label 'app.kubernetes.io/name' is required."
      pattern:
        metadata:
          labels:
            app.kubernetes.io/name: "?*"
EOF
```

## Mutate Rules

```bash
# Add default resource requests
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: add-default-resources
spec:
  rules:
  - name: add-default-requests
    match:
      any:
      - resources:
          kinds:
          - Pod
    mutate:
      patchStrategicMerge:
        spec:
          containers:
          - (name): "*"
            resources:
              requests:
                =(memory): "128Mi"
                =(cpu): "100m"
EOF

# Add labels using variables
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: add-namespace-label
spec:
  rules:
  - name: add-ns-label
    match:
      any:
      - resources:
          kinds:
          - Pod
    mutate:
      patchStrategicMerge:
        metadata:
          labels:
            namespace: "{{request.namespace}}"
            created-by: "{{serviceAccountName}}"
EOF
```

## Generate Rules

```bash
# Auto-generate NetworkPolicy for new namespaces
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: generate-default-netpol
spec:
  rules:
  - name: default-deny-ingress
    match:
      any:
      - resources:
          kinds:
          - Namespace
    exclude:
      any:
      - resources:
          namespaces:
          - kube-system
          - kyverno
    generate:
      synchronize: true
      apiVersion: networking.k8s.io/v1
      kind: NetworkPolicy
      name: default-deny-ingress
      namespace: "{{request.object.metadata.name}}"
      data:
        spec:
          podSelector: {}
          policyTypes:
          - Ingress
EOF

```

## Verify Images

```bash
# Verify container image signatures with cosign
cat <<EOF | kubectl apply -f -
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: verify-image-signature
spec:
  validationFailureAction: Enforce
  webhookTimeoutSeconds: 30
  rules:
  - name: verify-cosign-signature
    match:
      any:
      - resources:
          kinds:
          - Pod
    verifyImages:
    - imageReferences:
      - "ghcr.io/my-org/*"
      attestors:
      - count: 1
        entries:
        - keys:
            publicKeys: |-
              -----BEGIN PUBLIC KEY-----
              MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
              -----END PUBLIC KEY-----
    - imageReferences:
      - "docker.io/my-org/*"
      attestors:
      - entries:
        - keyless:
            url: https://fulcio.sigstore.dev
            roots: |-
              -----BEGIN CERTIFICATE-----
              ...
              -----END CERTIFICATE-----
            subject: "build@my-org.com"
            issuer: "https://accounts.google.com"
EOF
```

## Background Scanning and Policy Reports

```bash
# Check policy report results
kubectl get policyreport -A
kubectl get clusterpolicyreport

# Detailed report for a namespace
kubectl get policyreport -n default -o yaml

# Count violations by policy
kubectl get policyreport -A -o json | \
  jq '[.items[].results[]? | select(.result=="fail")] | group_by(.policy) | map({policy: .[0].policy, count: length})'

# View cluster-wide report
kubectl describe clusterpolicyreport

# Test a policy against a resource without applying
kyverno apply policy.yaml --resource resource.yaml

# Test policies in CI with kyverno CLI
kyverno test .
```

## Tips

- Use `validationFailureAction: Audit` initially to observe violations before switching to `Enforce`
- Background scanning catches existing non-compliant resources, not just new admissions
- The `=(field)` anchor in patterns means "if the field exists, it must match" -- vital for optional fields
- Use `exclude` to exempt system namespaces (kube-system, kyverno) from policies
- Generate rules with `synchronize: true` keep generated resources in sync when the trigger changes
- Variables like `{{request.namespace}}` and `{{serviceAccountName}}` enable dynamic mutations
- Image verification with cosign integrates supply chain security directly into admission control
- Policy reports provide compliance dashboards without blocking workloads when in Audit mode
- Use `preconditions` with JMESPath for complex conditional logic before rule evaluation
- The kyverno CLI (`kyverno apply`, `kyverno test`) enables policy testing in CI pipelines before deployment
- Namespace-scoped `Policy` resources let teams manage their own policies without cluster-admin
- Set `webhookTimeoutSeconds` appropriately for image verification rules that make external calls

## See Also

- opa
- cert-manager
- external-secrets

## References

- [Kyverno Documentation](https://kyverno.io/docs/)
- [Kyverno GitHub Repository](https://github.com/kyverno/kyverno)
- [Kyverno Policy Library](https://kyverno.io/policies/)
- [Policy Reports](https://kyverno.io/docs/policy-reports/)
- [Image Verification](https://kyverno.io/docs/writing-policies/verify-images/)
