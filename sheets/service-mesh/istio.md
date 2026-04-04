# Istio (Service Mesh Control Plane)

Kubernetes-native service mesh providing traffic management, mTLS security, and observability through sidecar proxies (Envoy) orchestrated by the istiod control plane.

## Installation

### Install istio

```bash
# Download istioctl
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# Install with default profile
istioctl install --set profile=demo -y

# Install with production profile (minimal, no extras)
istioctl install --set profile=minimal -y

# Verify installation
istioctl verify-install

# Enable sidecar injection for a namespace
kubectl label namespace default istio-injection=enabled

# Check proxy status
istioctl proxy-status
```

### Profiles

```bash
# List available profiles
istioctl profile list

# Dump profile config
istioctl profile dump demo

# Diff two profiles
istioctl profile diff demo default

# Custom install with IstioOperator
cat <<EOF | istioctl install -f -
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  profile: default
  meshConfig:
    accessLogFile: /dev/stdout
    enableTracing: true
  components:
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          resources:
            requests:
              cpu: 200m
              memory: 128Mi
EOF
```

## Traffic Management

### VirtualService

```yaml
# Route traffic with path-based routing and header matching
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews
spec:
  hosts:
    - reviews
  http:
    # Header-based routing (test users get v2)
    - match:
        - headers:
            end-user:
              exact: test-user
      route:
        - destination:
            host: reviews
            subset: v2
    # Weighted traffic split (canary)
    - route:
        - destination:
            host: reviews
            subset: v1
          weight: 90
        - destination:
            host: reviews
            subset: v2
          weight: 10
      timeout: 10s
      retries:
        attempts: 3
        perTryTimeout: 3s
        retryOn: gateway-error,connect-failure,refused-stream
```

### DestinationRule

```yaml
# Define subsets and connection pool settings
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: reviews
spec:
  host: reviews
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        h2UpgradePolicy: DEFAULT
        http1MaxPendingRequests: 100
        http2MaxRequests: 1000
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
    loadBalancer:
      simple: LEAST_REQUEST
  subsets:
    - name: v1
      labels:
        version: v1
    - name: v2
      labels:
        version: v2
```

### Fault injection

```yaml
# Inject delays and aborts for testing
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: ratings
spec:
  hosts:
    - ratings
  http:
    - fault:
        delay:
          percentage:
            value: 10
          fixedDelay: 5s
        abort:
          percentage:
            value: 5
          httpStatus: 503
      route:
        - destination:
            host: ratings
```

### Traffic mirroring

```yaml
# Mirror traffic to a shadow service
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: reviews
spec:
  hosts:
    - reviews
  http:
    - route:
        - destination:
            host: reviews
            subset: v1
      mirror:
        host: reviews
        subset: v2
      mirrorPercentage:
        value: 100
```

## Gateway

### Ingress gateway

```yaml
# Define gateway for external traffic
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: app-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 443
        name: https
        protocol: HTTPS
      tls:
        mode: SIMPLE
        credentialName: app-tls-cert
      hosts:
        - "app.example.com"
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - "app.example.com"
      tls:
        httpsRedirect: true
---
# Bind VirtualService to the gateway
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: app
spec:
  hosts:
    - "app.example.com"
  gateways:
    - app-gateway
  http:
    - route:
        - destination:
            host: app-service
            port:
              number: 8080
```

## Security (mTLS)

### PeerAuthentication

```yaml
# Enforce strict mTLS mesh-wide
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: istio-system
spec:
  mtls:
    mode: STRICT

---
# Permissive mode for a specific namespace (migration)
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: permissive
  namespace: legacy-app
spec:
  mtls:
    mode: PERMISSIVE
```

### AuthorizationPolicy

```yaml
# Allow only specific services to call the API
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: api-access
  namespace: default
spec:
  selector:
    matchLabels:
      app: api-server
  action: ALLOW
  rules:
    - from:
        - source:
            principals: ["cluster.local/ns/default/sa/frontend"]
      to:
        - operation:
            methods: ["GET", "POST"]
            paths: ["/api/*"]
    - from:
        - source:
            namespaces: ["monitoring"]
      to:
        - operation:
            methods: ["GET"]
            paths: ["/health", "/metrics"]
```

### RequestAuthentication (JWT)

```yaml
# Validate JWT tokens
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: jwt-auth
spec:
  selector:
    matchLabels:
      app: api-server
  jwtRules:
    - issuer: "https://auth.example.com"
      jwksUri: "https://auth.example.com/.well-known/jwks.json"
      forwardOriginalToken: true
```

## Observability

### istioctl diagnostics

```bash
# Check proxy sync status
istioctl proxy-status

# View proxy config for a pod
istioctl proxy-config routes deploy/reviews

# View clusters configured in sidecar
istioctl proxy-config clusters deploy/reviews

# Analyze mesh configuration for issues
istioctl analyze

# View Envoy access logs
kubectl logs deploy/reviews -c istio-proxy -f

# Dashboard access (Kiali, Grafana, Jaeger)
istioctl dashboard kiali
istioctl dashboard grafana
istioctl dashboard jaeger
```

### Telemetry configuration

```yaml
# Custom metrics and access logging
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: mesh-telemetry
  namespace: istio-system
spec:
  accessLogging:
    - providers:
        - name: envoy
  metrics:
    - providers:
        - name: prometheus
```

## Tips

- Start with `PERMISSIVE` mTLS mode during migration, then switch to `STRICT` after all services have sidecars.
- Use `istioctl analyze` before deploying any Istio configuration to catch misconfigurations early.
- Set explicit timeouts and retries in VirtualServices rather than relying on defaults.
- Use DestinationRule subsets to define service versions for canary and blue-green deployments.
- Enable access logging (`meshConfig.accessLogFile: /dev/stdout`) in production for debugging.
- AuthorizationPolicy defaults to ALLOW-all when no policies exist; adding the first policy changes behavior.
- Use `istioctl proxy-config` to inspect what Envoy actually received from istiod for debugging.
- Fault injection is invaluable for chaos testing — test with small percentages first.
- Traffic mirroring sends copies of live traffic to shadow services without impacting the primary path.
- Pin sidecar proxy versions with `istio.io/rev` labels to avoid unexpected upgrades during canary rollouts.
- Use Kiali for visual service graph and traffic flow analysis.
- Keep VirtualService and DestinationRule in the same namespace as the target service for clarity.

## See Also

envoy, linkerd, kubernetes, cloud-dns, vpc

## References

- [Istio Documentation](https://istio.io/latest/docs/)
- [Istio Traffic Management](https://istio.io/latest/docs/concepts/traffic-management/)
- [Istio Security](https://istio.io/latest/docs/concepts/security/)
- [Istio VirtualService Reference](https://istio.io/latest/docs/reference/config/networking/virtual-service/)
- [Istio DestinationRule Reference](https://istio.io/latest/docs/reference/config/networking/destination-rule/)
- [Istio AuthorizationPolicy Reference](https://istio.io/latest/docs/reference/config/security/authorization-policy/)
- [Istio GitHub Repository](https://github.com/istio/istio)
