# OPA (Open Policy Agent)

General-purpose policy engine that decouples policy decisions from application logic using the Rego language.

## Installation

```bash
# Install OPA binary
curl -L -o opa https://openpolicyagent.org/downloads/v0.64.1/opa_linux_amd64_static
chmod +x opa && sudo mv opa /usr/local/bin/

# Run OPA as a server
opa run --server --addr :8181

# Install OPA Gatekeeper on Kubernetes
kubectl apply -f https://raw.githubusercontent.com/open-policy-agent/gatekeeper/v3.16.0/deploy/gatekeeper.yaml

# Install conftest for IaC policy testing
brew install conftest
# or
go install github.com/open-policy-agent/conftest@latest
```

## Rego Language Basics

```bash
# Write a basic policy (policy.rego)
cat > policy.rego <<'EOF'
package authz

import rego.v1

default allow := false

allow if {
    input.method == "GET"
    input.path == ["public", "data"]
}

allow if {
    input.user.role == "admin"
}

denied_reasons contains msg if {
    not allow
    msg := sprintf("user %s is not authorized for %s %s",
        [input.user.name, input.method, concat("/", input.path)])
}
EOF

# Create test input (input.json)
cat > input.json <<'EOF'
{
  "method": "DELETE",
  "path": ["admin", "users"],
  "user": {"name": "alice", "role": "viewer"}
}
EOF

# Evaluate the policy
opa eval -i input.json -d policy.rego "data.authz.allow"
opa eval -i input.json -d policy.rego "data.authz.denied_reasons"

# Run policy in REPL
opa run policy.rego
# > data.authz.allow with input as {"method": "GET", "path": ["public", "data"]}
```

## OPA Server and Data API

```bash
# Start OPA server with a policy bundle
opa run --server --addr :8181 \
  --set bundles.authz.resource=/bundles/authz.tar.gz \
  --set bundles.authz.polling.min_delay_seconds=30

# Query the OPA API
curl -s localhost:8181/v1/data/authz/allow \
  -d '{"input": {"method": "GET", "path": ["public","data"]}}' | jq .

# Upload a policy via API
curl -X PUT localhost:8181/v1/policies/authz \
  --data-binary @policy.rego

# Upload data (external context)
curl -X PUT localhost:8181/v1/data/roles \
  -d '{"admin": ["alice","bob"], "viewer": ["charlie"]}'

# Query with partial evaluation (optimize for known inputs)
curl -s localhost:8181/v1/compile \
  -d '{
    "query": "data.authz.allow == true",
    "input": {"method": "GET"},
    "unknowns": ["input.user"]
  }' | jq .
```

## Policy Bundles

```bash
# Create a bundle directory structure
mkdir -p bundles/authz
cp policy.rego bundles/authz/
cat > bundles/authz/data.json <<'EOF'
{
  "roles": {
    "admin": ["alice", "bob"],
    "editor": ["charlie", "dave"]
  }
}
EOF

# Build a bundle
cd bundles && tar -czf authz.tar.gz -C authz . && cd ..

# Sign a bundle
opa sign --bundle bundles/authz/ \
  --signing-key /path/to/private.pem \
  --output-file bundles/authz/.signatures.json

# Serve bundles via HTTP
python3 -m http.server 8888 --directory bundles &

# Configure OPA to fetch bundles
opa run --server \
  --set bundles.authz.resource=authz.tar.gz \
  --set services.bundlehost.url=http://localhost:8888
```

## Gatekeeper (Kubernetes Admission Control)

```bash
# Create a ConstraintTemplate
cat <<EOF | kubectl apply -f -
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: k8srequiredlabels
spec:
  crd:
    spec:
      names:
        kind: K8sRequiredLabels
      validation:
        openAPIV3Schema:
          type: object
          properties:
            labels:
              type: array
              items:
                type: string
  targets:
  - target: admission.k8s.gatekeeper.sh
    rego: |
      package k8srequiredlabels

      import rego.v1

      violation contains {"msg": msg, "details": {"missing_labels": missing}} if {
        provided := {label | input.review.object.metadata.labels[label]}
        required := {label | label := input.parameters.labels[_]}
        missing := required - provided
        count(missing) > 0
        msg := sprintf("Missing required labels: %v", [missing])
      }
EOF

# Create a Constraint using the template
cat <<EOF | kubectl apply -f -
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredLabels
metadata:
  name: require-team-label
spec:
  enforcementAction: deny
  match:
    kinds:
    - apiGroups: [""]
      kinds: ["Namespace"]
    - apiGroups: ["apps"]
      kinds: ["Deployment"]
  parameters:
    labels:
    - "team"
    - "environment"
EOF

# Check constraint violations
kubectl get k8srequiredlabels -o yaml
kubectl get constrainttemplate -o wide

# Dry-run enforcement
cat <<EOF | kubectl apply -f -
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: K8sRequiredLabels
metadata:
  name: warn-cost-center
spec:
  enforcementAction: warn
  match:
    kinds:
    - apiGroups: ["apps"]
      kinds: ["Deployment"]
  parameters:
    labels:
    - "cost-center"
EOF
```

## conftest (IaC Policy Testing)

```bash
# Write a policy for Dockerfiles
mkdir -p policy
cat > policy/dockerfile.rego <<'EOF'
package main

import rego.v1

deny contains msg if {
    input[i].Cmd == "from"
    val := input[i].Value
    contains(val[0], ":latest")
    msg := "Do not use :latest tag in FROM"
}

deny contains msg if {
    input[i].Cmd == "user"
    val := input[i].Value
    val[0] == "root"
    msg := "Do not run as root user"
}
EOF

# Test a Dockerfile
conftest test Dockerfile

# Test Kubernetes manifests
cat > policy/k8s.rego <<'EOF'
package main

import rego.v1

deny contains msg if {
    input.kind == "Deployment"
    container := input.spec.template.spec.containers[_]
    not container.resources.limits
    msg := sprintf("Container %s has no resource limits", [container.name])
}
EOF

conftest test deployment.yaml

# Test Terraform plans
terraform show -json plan.out > plan.json
conftest test plan.json --policy policy/terraform/

# Test with multiple namespaces
conftest test --all-namespaces deployment.yaml
```

## Decision Logs and Monitoring

```bash
# Enable decision logging
opa run --server \
  --set decision_logs.console=true \
  --set decision_logs.reporting.min_delay_seconds=5

# Configure remote decision log endpoint
cat > config.yaml <<'EOF'
services:
  logservice:
    url: https://logs.example.com
decision_logs:
  service: logservice
  reporting:
    min_delay_seconds: 10
    max_delay_seconds: 60
EOF

opa run --server --config-file config.yaml

# Check OPA health
curl localhost:8181/health
curl localhost:8181/health?bundles=true
curl localhost:8181/health?plugins=true

# Prometheus metrics
curl localhost:8181/metrics
```

## Testing Rego Policies

```bash
# Run tests
opa test . -v

# Run tests with coverage
opa test . -v --coverage | jq '.coverage'

# Benchmark policies
opa bench -d policy.rego -i input.json "data.authz.allow"
```

## Tips

- Always write Rego tests alongside policies; use `opa test . -v --coverage` to enforce coverage thresholds
- Use `import rego.v1` in all new policies for forward-compatible syntax with future OPA versions
- Prefer `contains` and `if` keywords over implicit set membership for readability
- Bundle signing prevents tampered policies from being loaded into production OPA instances
- Use partial evaluation (`/v1/compile`) to pre-compute policies when some inputs are known at deploy time
- Gatekeeper's `enforcementAction: warn` lets you audit violations before enforcing them in production
- conftest integrates into CI pipelines to catch misconfigurations before they reach the cluster
- Keep Rego policies small and composable; import shared rules from library packages
- Decision logs are critical for audit trails; ship them to a SIEM for compliance reporting
- Use `data.json` files alongside policies to externalize lookup tables and role mappings
- Set `--addr` to a unix socket when OPA runs as a sidecar to avoid network exposure

## See Also

- kyverno
- gateway-api
- external-secrets

## References

- [OPA Documentation](https://www.openpolicyagent.org/docs/latest/)
- [Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA Gatekeeper](https://open-policy-agent.github.io/gatekeeper/website/docs/)
- [conftest Documentation](https://www.conftest.dev/)
- [OPA GitHub Repository](https://github.com/open-policy-agent/opa)
