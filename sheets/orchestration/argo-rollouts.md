# Argo Rollouts

Progressive delivery controller for Kubernetes providing canary deployments, blue-green deployments, and automated analysis-driven promotions.

## Installation

```bash
# Install Argo Rollouts controller
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/download/v1.7.0/install.yaml

# Install kubectl plugin
brew install argoproj/tap/kubectl-argo-rollouts
# or
curl -LO https://github.com/argoproj/argo-rollouts/releases/download/v1.7.0/kubectl-argo-rollouts-darwin-amd64
chmod +x kubectl-argo-rollouts-darwin-amd64
sudo mv kubectl-argo-rollouts-darwin-amd64 /usr/local/bin/kubectl-argo-rollouts

# Verify installation
kubectl argo rollouts version
kubectl get pods -n argo-rollouts
```

## Canary Strategy

```bash
# Basic canary rollout with step-based progression
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  replicas: 10
  revisionHistoryLimit: 3
  selector:
    matchLabels:
      app: my-app
  strategy:
    canary:
      canaryService: my-app-canary
      stableService: my-app-stable
      steps:
      - setWeight: 10
      - pause: {duration: 5m}
      - setWeight: 25
      - pause: {duration: 5m}
      - setWeight: 50
      - pause: {duration: 10m}
      - setWeight: 75
      - pause: {duration: 5m}
      maxSurge: "25%"
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-app
        image: my-app:v2.0.0
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 256Mi
EOF

```

## Blue-Green Strategy

```bash
# Blue-green rollout with preview service
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app-bg
spec:
  replicas: 5
  revisionHistoryLimit: 2
  selector:
    matchLabels:
      app: my-app-bg
  strategy:
    blueGreen:
      activeService: my-app-active
      previewService: my-app-preview
      autoPromotionEnabled: false
      previewReplicaCount: 2
      scaleDownDelaySeconds: 300
      scaleDownDelayRevisionLimit: 1
  template:
    metadata:
      labels:
        app: my-app-bg
    spec:
      containers:
      - name: app
        image: my-app:v2.0.0
        ports:
        - containerPort: 8080
EOF

# Create the active and preview services
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: my-app-active
spec:
  selector:
    app: my-app-bg
  ports:
  - port: 80
    targetPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: my-app-preview
spec:
  selector:
    app: my-app-bg
  ports:
  - port: 80
    targetPort: 8080
EOF
```

## Analysis Templates

```bash
# AnalysisTemplate using Prometheus metrics
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  args:
  - name: service-name
  metrics:
  - name: success-rate
    interval: 60s
    count: 5
    failureLimit: 1
    successCondition: result[0] >= 0.95
    provider:
      prometheus:
        address: http://prometheus.monitoring:9090
        query: |
          sum(rate(http_requests_total{service="{{args.service-name}}",status=~"2.."}[5m]))
          /
          sum(rate(http_requests_total{service="{{args.service-name}}"}[5m]))
EOF

# AnalysisTemplate using a web hook
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: load-test
spec:
  metrics:
  - name: load-test
    count: 1
    provider:
      job:
        spec:
          template:
            spec:
              containers:
              - name: load-test
                image: grafana/k6:latest
                command: ["k6", "run", "/scripts/test.js"]
              restartPolicy: Never
          backoffLimit: 0
EOF

# Rollout referencing the analysis template
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app-analyzed
spec:
  replicas: 10
  selector:
    matchLabels:
      app: my-app-analyzed
  strategy:
    canary:
      steps:
      - setWeight: 20
      - pause: {duration: 2m}
      - analysis:
          templates:
          - templateName: success-rate
          args:
          - name: service-name
            value: my-app-analyzed
      - setWeight: 50
      - pause: {duration: 5m}
      - setWeight: 80
      - pause: {duration: 5m}
  template:
    metadata:
      labels:
        app: my-app-analyzed
    spec:
      containers:
      - name: app
        image: my-app:v2.0.0
EOF
```

## Traffic Management with Istio

```bash
# Canary with Istio traffic routing
cat <<EOF | kubectl apply -f -
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app-istio
spec:
  replicas: 10
  selector:
    matchLabels:
      app: my-app-istio
  strategy:
    canary:
      canaryService: my-app-canary
      stableService: my-app-stable
      trafficRouting:
        istio:
          virtualServices:
          - name: my-app-vsvc
            routes:
            - primary
          destinationRule:
            name: my-app-destrule
            canarySubsetName: canary
            stableSubsetName: stable
      steps:
      - setWeight: 5
      - pause: {duration: 2m}
      - setWeight: 20
      - pause: {duration: 5m}
      - setWeight: 50
      - pause: {duration: 10m}
  template:
    metadata:
      labels:
        app: my-app-istio
    spec:
      containers:
      - name: app
        image: my-app:v2.0.0
EOF
```

## kubectl Plugin Commands

```bash
# Watch rollout status live
kubectl argo rollouts get rollout my-app --watch

# Promote a paused rollout
kubectl argo rollouts promote my-app

# Full promote (skip remaining steps)
kubectl argo rollouts promote my-app --full

# Abort a rollout (revert to stable)
kubectl argo rollouts abort my-app

# Retry a failed rollout
kubectl argo rollouts retry rollout my-app

# Restart pods (trigger rollout with same image)
kubectl argo rollouts restart my-app

# Set image to trigger a new rollout
kubectl argo rollouts set image my-app my-app=my-app:v3.0.0

# Undo a rollout (revert to previous revision)
kubectl argo rollouts undo my-app

# List all rollouts
kubectl argo rollouts list rollouts

# Open the dashboard
kubectl argo rollouts dashboard
```

## Tips

- Start canary at 5-10% traffic weight to catch severe regressions before broader exposure
- Use `pause: {}` (no duration) for critical steps that require explicit human approval
- Always define `maxUnavailable: 0` for zero-downtime canary progressions
- Analysis templates should query metrics over at least 2-3 minutes to avoid noise-driven decisions
- Set `scaleDownDelaySeconds` in blue-green to allow in-flight requests to drain before termination
- Use `failureLimit` in analysis to tolerate transient metric dips without aborting the rollout
- Combine header-based routing (`canary-by-header`) with weight-based routing for internal testing
- The kubectl plugin dashboard provides real-time rollout visualization at `localhost:3100`
- Set `revisionHistoryLimit` to limit stored ReplicaSets and prevent resource sprawl
- Experiments are useful for A/B testing where both versions need dedicated traffic splits
- Always pair Argo Rollouts with proper monitoring; automated promotion without observability is dangerous
- Use `ClusterAnalysisTemplate` for organization-wide quality gates shared across teams

## See Also

- gateway-api
- kyverno
- opa

## References

- [Argo Rollouts Documentation](https://argo-rollouts.readthedocs.io/en/stable/)
- [Argo Rollouts GitHub Repository](https://github.com/argoproj/argo-rollouts)
- [Canary Strategy Specification](https://argo-rollouts.readthedocs.io/en/stable/features/canary/)
- [Analysis and Progressive Delivery](https://argo-rollouts.readthedocs.io/en/stable/features/analysis/)
- [Traffic Management Integrations](https://argo-rollouts.readthedocs.io/en/stable/features/traffic-management/)
