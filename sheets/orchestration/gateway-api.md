# Gateway API

Kubernetes-native API for modeling service networking with expressive, role-oriented routing resources that supersede Ingress.

## Core Concepts

```bash
# Gateway API is a set of CRDs, not a controller.
# Install the standard channel CRDs:
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml

# Install experimental channel (includes TCPRoute, TLSRoute, GRPCRoute):
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/experimental-install.yaml

# Verify CRDs are installed
kubectl get crd | grep gateway.networking.k8s.io
```

## GatewayClass

```bash
# GatewayClass defines which controller handles Gateway resources
# Most implementations install their own GatewayClass automatically.

# List available GatewayClasses
kubectl get gatewayclass

# Example GatewayClass (installed by controller, rarely created manually)
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: my-gateway-class
spec:
  controllerName: example.com/gateway-controller
  parametersRef:
    group: example.com
    kind: GatewayConfig
    name: my-config
    namespace: default
EOF
```

## Gateway Resource

```bash
# Create a Gateway with HTTP and HTTPS listeners
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: main-gateway
  namespace: gateway-system
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  gatewayClassName: nginx
  listeners:
  - name: http
    protocol: HTTP
    port: 80
    allowedRoutes:
      namespaces:
        from: All
  - name: https
    protocol: HTTPS
    port: 443
    tls:
      mode: Terminate
      certificateRefs:
      - name: wildcard-tls
        namespace: gateway-system
    allowedRoutes:
      namespaces:
        from: Selector
        selector:
          matchLabels:
            gateway-access: "true"
  - name: https-specific
    protocol: HTTPS
    port: 443
    hostname: api.example.com
    tls:
      mode: Terminate
      certificateRefs:
      - name: api-tls
    allowedRoutes:
      namespaces:
        from: Same
EOF

# Check Gateway status
kubectl get gateway main-gateway -n gateway-system -o yaml
kubectl describe gateway main-gateway -n gateway-system
```

## HTTPRoute

```bash
# Basic HTTPRoute with path matching
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: app-routes
  namespace: default
spec:
  parentRefs:
  - name: main-gateway
    namespace: gateway-system
  hostnames:
  - "app.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /api
    backendRefs:
    - name: api-service
      port: 8080
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: frontend-service
      port: 3000
EOF

# HTTPRoute with header matching
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: header-routing
  namespace: default
spec:
  parentRefs:
  - name: main-gateway
    namespace: gateway-system
  hostnames:
  - "api.example.com"
  rules:
  - matches:
    - headers:
      - name: X-API-Version
        value: "v2"
    backendRefs:
    - name: api-v2
      port: 8080
  - matches:
    - headers:
      - name: X-API-Version
        value: "v1"
    backendRefs:
    - name: api-v1
      port: 8080
  - backendRefs:
    - name: api-v1
      port: 8080
EOF

# Traffic splitting (canary/weighted routing)
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: canary-route
  namespace: default
spec:
  parentRefs:
  - name: main-gateway
    namespace: gateway-system
  hostnames:
  - "app.example.com"
  rules:
  - backendRefs:
    - name: app-stable
      port: 8080
      weight: 90
    - name: app-canary
      port: 8080
      weight: 10
EOF

# Request redirect and URL rewrite
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: redirect-and-rewrite
  namespace: default
spec:
  parentRefs:
  - name: main-gateway
    namespace: gateway-system
  hostnames:
  - "old.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /old-path
    filters:
    - type: RequestRedirect
      requestRedirect:
        hostname: new.example.com
        statusCode: 301
  - matches:
    - path:
        type: PathPrefix
        value: /v1
    filters:
    - type: URLRewrite
      urlRewrite:
        path:
          type: ReplacePrefixMatch
          replacePrefixMatch: /v2
    backendRefs:
    - name: api-service
      port: 8080
EOF

```

## TCPRoute and TLSRoute

```bash
# TCPRoute for non-HTTP TCP traffic
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TCPRoute
metadata:
  name: postgres-route
  namespace: default
spec:
  parentRefs:
  - name: main-gateway
    namespace: gateway-system
    sectionName: postgres
  rules:
  - backendRefs:
    - name: postgres-service
      port: 5432
EOF

# TLSRoute with SNI-based routing (TLS passthrough)
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: TLSRoute
metadata:
  name: tls-passthrough
  namespace: default
spec:
  parentRefs:
  - name: main-gateway
    namespace: gateway-system
    sectionName: tls-passthrough
  hostnames:
  - "secure.example.com"
  rules:
  - backendRefs:
    - name: secure-backend
      port: 443
EOF
```

## Implementation-Specific Setup

```bash
# NGINX Gateway Fabric
helm install ngf oci://ghcr.io/nginx/charts/nginx-gateway-fabric \
  --namespace nginx-gateway --create-namespace

# Envoy Gateway
helm install eg oci://docker.io/envoyproxy/gateway-helm \
  --namespace envoy-gateway-system --create-namespace

# Istio with Gateway API
istioctl install --set profile=minimal
kubectl get gatewayclass istio

# Traefik with Gateway API
helm install traefik traefik/traefik \
  --namespace traefik --create-namespace \
  --set providers.kubernetesGateway.enabled=true

# Check which implementations are available
kubectl get gatewayclass -o wide
```

## Cross-Namespace References

```bash
# Grant access for routes in other namespaces
cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1beta1
kind: ReferenceGrant
metadata:
  name: allow-routes-from-apps
  namespace: gateway-system
spec:
  from:
  - group: gateway.networking.k8s.io
    kind: HTTPRoute
    namespace: apps
  to:
  - group: ""
    kind: Service
  - group: ""
    kind: Secret
EOF

# Check Gateway status and route attachment
kubectl get gateway -A -o wide
kubectl describe httproute app-routes
kubectl get httproute app-routes -o jsonpath='{.status.parents[*].conditions}'
```

## Tips

- Gateway API separates concerns: infra admins manage Gateways, app teams manage Routes
- Use `allowedRoutes.namespaces.from: Selector` to control which namespaces can attach routes
- ReferenceGrant is required for cross-namespace references to Services and Secrets
- Weight-based traffic splitting in HTTPRoute replaces Ingress canary annotations natively
- Header matching enables version-based routing without separate hostnames or paths
- TLS termination at the Gateway centralizes certificate management via `certificateRefs`
- TCPRoute and TLSRoute are experimental channel; install experimental CRDs to use them
- Multiple listeners on the same port differentiate by hostname (SNI-based multiplexing)
- Gateway API is implementation-agnostic; switching from nginx to envoy only changes the GatewayClass
- Use `sectionName` in parentRefs to bind a route to a specific Gateway listener
- Filter chaining (redirect, rewrite, header modification) applies in a defined order per the spec
- Check `status.conditions` on both Gateway and Route resources for debugging attachment failures

## See Also

- cert-manager
- argo-rollouts
- kyverno

## References

- [Gateway API Documentation](https://gateway-api.sigs.k8s.io/)
- [Gateway API GitHub Repository](https://github.com/kubernetes-sigs/gateway-api)
- [Gateway API Implementations](https://gateway-api.sigs.k8s.io/implementations/)
- [HTTPRoute Specification](https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1.HTTPRoute)
- [GEP-1742: Gateway API Mesh](https://gateway-api.sigs.k8s.io/geps/gep-1742/)
