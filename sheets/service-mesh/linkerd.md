# Linkerd (Lightweight Service Mesh)

Ultralight Kubernetes service mesh built on a Rust-based micro-proxy (linkerd2-proxy), providing automatic mTLS, traffic splitting, observability, and reliability without complex configuration.

## Installation

### Install linkerd CLI and control plane

```bash
# Install CLI
curl --proto '=https' --tlsv1.2 -sSfL https://run.linkerd.io/install | sh
export PATH=$HOME/.linkerd2/bin:$PATH

# Verify CLI version
linkerd version

# Pre-installation checks
linkerd check --pre

# Install CRDs
linkerd install --crds | kubectl apply -f -

# Install control plane
linkerd install | kubectl apply -f -

# Wait for control plane to be ready
linkerd check

# Install the viz extension (dashboard, metrics)
linkerd viz install | kubectl apply -f -
linkerd viz check
```

### Inject sidecars

```bash
# Inject proxy into a deployment
kubectl get deploy my-app -o yaml | linkerd inject - | kubectl apply -f -

# Inject entire namespace via annotation
kubectl annotate namespace default linkerd.io/inject=enabled

# Inject during kubectl apply
cat deployment.yaml | linkerd inject - | kubectl apply -f -

# Check injection status
linkerd check --proxy -n default
```

## Traffic Management

### Traffic splitting (canary deployments)

```yaml
# TrafficSplit resource (SMI spec)
apiVersion: split.smi-spec.io/v1alpha4
kind: TrafficSplit
metadata:
  name: web-split
  namespace: default
spec:
  service: web-svc
  backends:
    - service: web-stable
      weight: 900
    - service: web-canary
      weight: 100
```

```bash
# Apply traffic split
kubectl apply -f traffic-split.yaml

# Monitor split traffic
linkerd viz stat ts/web-split

# Gradually shift traffic
kubectl patch trafficsplit web-split --type=merge -p '{
  "spec": {
    "backends": [
      {"service": "web-stable", "weight": 500},
      {"service": "web-canary", "weight": 500}
    ]
  }
}'
```

### Service profiles (per-route metrics and retries)

```yaml
# Service profile for fine-grained control
apiVersion: linkerd.io/v1alpha2
kind: ServiceProfile
metadata:
  name: web-svc.default.svc.cluster.local
  namespace: default
spec:
  routes:
    - name: GET /api/users
      condition:
        method: GET
        pathRegex: /api/users
      isRetryable: true
      timeout: 5s
    - name: POST /api/orders
      condition:
        method: POST
        pathRegex: /api/orders
      isRetryable: false
      timeout: 30s
    - name: GET /health
      condition:
        method: GET
        pathRegex: /health
```

```bash
# Generate a service profile from OpenAPI spec
linkerd profile --open-api swagger.json web-svc | kubectl apply -f -

# Generate from live traffic (tap-based)
linkerd profile --tap deploy/web --tap-duration 30s web-svc
```

### Retries and timeouts

```yaml
# Configure retries in service profile
spec:
  retryBudget:
    retryRatio: 0.2          # Max 20% of requests can be retries
    minRetriesPerSecond: 10   # Floor of retries/sec
    ttl: 10s                  # Retry budget window
  routes:
    - name: GET /api/data
      condition:
        method: GET
        pathRegex: /api/data
      isRetryable: true
      timeout: 3s
```

## Security (Automatic mTLS)

### Verify mTLS status

```bash
# Check mTLS across the mesh
linkerd viz edges deploy -n default

# Verify specific connection
linkerd viz tap deploy/web --to deploy/api | grep tls

# Show identity (certificates) for a workload
linkerd identity -n default

# Check certificate expiry
linkerd check --proxy
```

### Authorization policies

```yaml
# Server resource (defines a port/protocol)
apiVersion: policy.linkerd.io/v1beta3
kind: Server
metadata:
  name: api-server
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: api
  port: 8080
  proxyProtocol: HTTP/2

---
# AuthorizationPolicy — allow only frontend
apiVersion: policy.linkerd.io/v1alpha1
kind: AuthorizationPolicy
metadata:
  name: api-authz
  namespace: default
spec:
  targetRef:
    group: policy.linkerd.io
    kind: Server
    name: api-server
  requiredAuthenticationRefs:
    - name: frontend-identity
      kind: MeshTLSAuthentication
      group: policy.linkerd.io

---
# MeshTLSAuthentication
apiVersion: policy.linkerd.io/v1alpha1
kind: MeshTLSAuthentication
metadata:
  name: frontend-identity
  namespace: default
spec:
  identities:
    - "*.default.serviceaccount.identity.linkerd.cluster.local"
```

## Observability

### Live traffic inspection

```bash
# Tap live requests (real-time stream)
linkerd viz tap deploy/web

# Filter tap by method and path
linkerd viz tap deploy/web \
  --method GET --path /api/users

# Tap requests between two services
linkerd viz tap deploy/web --to deploy/api

# View per-route stats
linkerd viz routes deploy/web

# Top destinations by request volume
linkerd viz top deploy/web

# View golden metrics per deployment
linkerd viz stat deploy -n default

# View stats with per-route breakdown
linkerd viz routes deploy/web --to svc/api-svc
```

### Dashboard

```bash
# Open Linkerd dashboard
linkerd viz dashboard

# Port-forward Grafana
linkerd viz dashboard --show grafana
```

### Golden metrics output

```bash
# Example output from linkerd viz stat
# NAME     MESHED   SUCCESS   RPS   LATENCY_P50   LATENCY_P95   LATENCY_P99
# web      1/1      99.50%    50    2ms           10ms          25ms
# api      2/2      98.80%    120   5ms           30ms          80ms
# db       1/1      100.00%   200   1ms           3ms           8ms
```

## Multi-cluster

### Link clusters

```bash
# Generate link credentials on target cluster
linkerd mc link --cluster-name east | kubectl apply -f -

# Verify multi-cluster setup
linkerd mc check

# Export a service to other clusters
kubectl label svc/api-svc mirror.linkerd.io/exported=true

# Check mirrored services
linkerd mc gateways
```

## Extensions

### Install extensions

```bash
# Jaeger extension (distributed tracing)
linkerd jaeger install | kubectl apply -f -

# Multicluster extension
linkerd mc install | kubectl apply -f -

# Policy controller (comes with control plane)
# Already included in linkerd install

# List installed extensions
linkerd check --extensions
```

## Debugging

### Diagnose issues

```bash
# Full health check
linkerd check

# Check specific namespace
linkerd check --proxy -n my-namespace

# Debug a pod's proxy
linkerd diagnostics proxy-metrics pod/web-abc123

# View proxy logs
kubectl logs pod/web-abc123 -c linkerd-proxy

# Check data plane version
linkerd version --proxy -n default

# Inspect proxy configuration
linkerd diagnostics endpoints svc/web-svc
```

## Tips

- Linkerd automatically enables mTLS for all meshed traffic with zero configuration.
- Use `linkerd viz stat` as the first diagnostic tool — it shows success rate, RPS, and latency percentiles.
- Service profiles enable per-route metrics; without them you only get per-service aggregates.
- The retry budget (default 20%) prevents retry storms; adjust `retryRatio` for sensitive services.
- Linkerd's Rust proxy uses ~10MB RAM and ~0.01 CPU cores per pod — significantly lighter than Envoy.
- Use `linkerd viz tap` for real-time request inspection without modifying application code.
- Annotate namespaces with `linkerd.io/inject=enabled` for automatic sidecar injection on new pods.
- TrafficSplit weights are relative (not percentages); 900:100 = 90:10 = 9:1.
- Always run `linkerd check` after installation and upgrades to verify mesh health.
- Use `linkerd viz edges` to visualize service-to-service connections and verify mTLS status.
- Linkerd follows the SMI spec for traffic splitting, making it portable across meshes.
- Upgrade Linkerd control plane before data plane proxies; proxies are backward compatible within a minor version.

## See Also

envoy, istio, kubernetes, tekton, prometheus

## References

- [Linkerd Documentation](https://linkerd.io/2/overview/)
- [Linkerd Getting Started](https://linkerd.io/2/getting-started/)
- [Linkerd Traffic Management](https://linkerd.io/2/features/traffic-split/)
- [Linkerd Service Profiles](https://linkerd.io/2/features/service-profiles/)
- [Linkerd Authorization Policy](https://linkerd.io/2/reference/authorization-policy/)
- [Linkerd Multi-cluster](https://linkerd.io/2/features/multicluster/)
- [SMI Traffic Split Spec](https://smi-spec.io/docs/traffic-split/)
- [Linkerd GitHub Repository](https://github.com/linkerd/linkerd2)
