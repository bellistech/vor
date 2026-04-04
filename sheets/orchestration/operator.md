# Operator (Kubernetes Operator Pattern)

Software extension pattern that uses Custom Resource Definitions and controller loops to encode operational knowledge for managing complex stateful applications on Kubernetes, automating lifecycle operations through the reconciliation paradigm.

## Operator Pattern

### Core Concepts

```
Custom Resource Definition (CRD)  — Extends the Kubernetes API
Custom Resource (CR)              — Instance of a CRD (desired state)
Controller                        — Reconciliation loop (drives actual → desired)
Operator                          — Controller + CRD + domain knowledge

# The reconciliation loop:
# 1. Observe: Watch for CR changes (create/update/delete)
# 2. Diff:    Compare desired state (CR spec) vs actual state (cluster)
# 3. Act:     Take actions to converge actual → desired
# 4. Update:  Write status back to CR

# Level-triggered (not edge-triggered):
# - Reacts to current state, not state transitions
# - Idempotent: running reconcile N times = running it once
# - Robust to missed events (always converges)
```

### Operator Capability Levels

```
Level 1: Basic Install
  - Automated provisioning (create/delete)
  - Declarative configuration via CR

Level 2: Seamless Upgrades
  - Version-aware upgrades/rollbacks
  - Schema migration

Level 3: Full Lifecycle
  - Backup/restore
  - Failure recovery
  - Scaling operations

Level 4: Deep Insights
  - Metrics, alerts, log processing
  - Dashboards and analysis

Level 5: Auto Pilot
  - Automated tuning
  - Anomaly detection
  - Self-healing without human intervention
```

## Custom Resource Definition

### CRD Manifest

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: databases.example.com
spec:
  group: example.com
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              required: ["engine", "version", "storage"]
              properties:
                engine:
                  type: string
                  enum: ["postgres", "mysql", "redis"]
                version:
                  type: string
                replicas:
                  type: integer
                  minimum: 1
                  maximum: 7
                  default: 3
                storage:
                  type: object
                  properties:
                    size:
                      type: string
                      pattern: "^[0-9]+(Gi|Ti)$"
                    storageClass:
                      type: string
                backup:
                  type: object
                  properties:
                    enabled:
                      type: boolean
                      default: true
                    schedule:
                      type: string
                    retention:
                      type: integer
                      default: 7
            status:
              type: object
              properties:
                phase:
                  type: string
                  enum: ["Pending", "Creating", "Running", "Failed", "Deleting"]
                replicas:
                  type: integer
                readyReplicas:
                  type: integer
                conditions:
                  type: array
                  items:
                    type: object
                    properties:
                      type:
                        type: string
                      status:
                        type: string
                      lastTransitionTime:
                        type: string
                        format: date-time
                      reason:
                        type: string
                      message:
                        type: string
      subresources:
        status: {}
        scale:
          specReplicasPath: .spec.replicas
          statusReplicasPath: .status.replicas
      additionalPrinterColumns:
        - name: Engine
          type: string
          jsonPath: .spec.engine
        - name: Version
          type: string
          jsonPath: .spec.version
        - name: Replicas
          type: integer
          jsonPath: .status.readyReplicas
        - name: Phase
          type: string
          jsonPath: .status.phase
        - name: Age
          type: date
          jsonPath: .metadata.creationTimestamp
  scope: Namespaced
  names:
    plural: databases
    singular: database
    kind: Database
    shortNames:
      - db
    categories:
      - all
```

### Custom Resource Instance

```yaml
apiVersion: example.com/v1alpha1
kind: Database
metadata:
  name: mydb
  namespace: production
spec:
  engine: postgres
  version: "16.2"
  replicas: 3
  storage:
    size: 100Gi
    storageClass: fast-ssd
  backup:
    enabled: true
    schedule: "0 2 * * *"
    retention: 30
```

## Controller Implementation (Go)

### Using controller-runtime

```go
package controllers

import (
    "context"
    "fmt"

    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    examplev1 "example.com/operator/api/v1alpha1"
)

type DatabaseReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=example.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.com,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // 1. Fetch the Database CR
    var db examplev1.Database
    if err := r.Get(ctx, req.NamespacedName, &db); err != nil {
        if errors.IsNotFound(err) {
            return ctrl.Result{}, nil   // CR deleted, nothing to do
        }
        return ctrl.Result{}, err
    }

    // 2. Handle deletion (finalizer pattern)
    if !db.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, &db)
    }

    // 3. Ensure finalizer is set
    if err := r.ensureFinalizer(ctx, &db); err != nil {
        return ctrl.Result{}, err
    }

    // 4. Reconcile owned resources
    if err := r.reconcileStatefulSet(ctx, &db); err != nil {
        return ctrl.Result{}, err
    }
    if err := r.reconcileService(ctx, &db); err != nil {
        return ctrl.Result{}, err
    }

    // 5. Update status
    db.Status.Phase = "Running"
    if err := r.Status().Update(ctx, &db); err != nil {
        return ctrl.Result{}, err
    }

    log.Info("Reconciliation complete", "database", db.Name)
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&examplev1.Database{}).              // Watch Database CR
        Owns(&appsv1.StatefulSet{}).              // Watch owned StatefulSets
        Owns(&corev1.Service{}).                  // Watch owned Services
        Complete(r)
}
```

### Reconciliation Patterns

```go
// Owner References — garbage collection
func setOwnerRef(owner *examplev1.Database, obj metav1.Object, scheme *runtime.Scheme) error {
    return ctrl.SetControllerReference(owner, obj, scheme)
}

// Finalizer pattern — cleanup external resources
const finalizerName = "databases.example.com/finalizer"

func (r *DatabaseReconciler) handleDeletion(ctx context.Context, db *examplev1.Database) (ctrl.Result, error) {
    if containsString(db.Finalizers, finalizerName) {
        // Clean up external resources (e.g., cloud storage, DNS)
        if err := r.deleteExternalResources(ctx, db); err != nil {
            return ctrl.Result{}, err
        }
        // Remove finalizer
        db.Finalizers = removeString(db.Finalizers, finalizerName)
        if err := r.Update(ctx, db); err != nil {
            return ctrl.Result{}, err
        }
    }
    return ctrl.Result{}, nil
}

// Status conditions pattern
func setCondition(db *examplev1.Database, condType, status, reason, message string) {
    now := metav1.Now()
    for i, c := range db.Status.Conditions {
        if c.Type == condType {
            if c.Status != status {
                db.Status.Conditions[i].LastTransitionTime = now
            }
            db.Status.Conditions[i].Status = status
            db.Status.Conditions[i].Reason = reason
            db.Status.Conditions[i].Message = message
            return
        }
    }
    db.Status.Conditions = append(db.Status.Conditions, examplev1.Condition{
        Type: condType, Status: status, Reason: reason,
        Message: message, LastTransitionTime: now,
    })
}
```

## Scaffolding with Kubebuilder

```bash
# Initialize project
kubebuilder init --domain example.com --repo example.com/operator

# Create API (CRD + controller)
kubebuilder create api --group example --version v1alpha1 --kind Database

# Generate manifests (CRD YAML, RBAC, webhook configs)
make manifests

# Generate deep copy methods
make generate

# Run locally (against current kubeconfig)
make run

# Build and push controller image
make docker-build docker-push IMG=registry.com/operator:v1.0

# Deploy to cluster
make deploy IMG=registry.com/operator:v1.0

# Run tests
make test

# Project structure:
# api/v1alpha1/       — Types, deepcopy, defaulting
# controllers/        — Reconciler implementation
# config/crd/         — Generated CRD manifests
# config/rbac/        — Generated RBAC manifests
# config/manager/     — Controller manager deployment
```

## Testing

### envtest (Integration Testing)

```go
var _ = Describe("Database Controller", func() {
    ctx := context.Background()

    It("should create StatefulSet for Database CR", func() {
        db := &examplev1.Database{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "test-db",
                Namespace: "default",
            },
            Spec: examplev1.DatabaseSpec{
                Engine:   "postgres",
                Version:  "16.2",
                Replicas: 3,
                Storage:  examplev1.StorageSpec{Size: "10Gi"},
            },
        }
        Expect(k8sClient.Create(ctx, db)).Should(Succeed())

        // Verify StatefulSet is created
        sts := &appsv1.StatefulSet{}
        Eventually(func() error {
            return k8sClient.Get(ctx, client.ObjectKeyFromObject(db), sts)
        }, timeout, interval).Should(Succeed())

        Expect(*sts.Spec.Replicas).To(Equal(int32(3)))
    })
})
```

## Tips

- Level-triggered reconciliation is the key insight: always compare desired vs actual state, never track events
- Make reconcile functions idempotent; the controller may run reconcile multiple times for the same state
- Use owner references on all child resources so Kubernetes garbage collects them when the CR is deleted
- Implement finalizers for cleanup of external resources (cloud APIs, DNS, storage) that Kubernetes cannot GC
- Use `RequeueAfter` for periodic reconciliation (health checks, cert renewal) instead of polling
- Status subresource updates do not trigger reconciliation, preventing infinite loops
- Watch owned resources (`Owns()`) to detect drift: if someone deletes a managed StatefulSet, reconcile recreates it
- Use kubebuilder markers (`+kubebuilder:rbac`, `+kubebuilder:validation`) for codegen instead of manual YAML
- Test with envtest (real API server, no kubelet) for fast integration tests without a full cluster
- Implement conversion webhooks when evolving CRD versions (v1alpha1 to v1beta1 to v1)
- Use leader election (`--leader-elect`) for HA operator deployments with multiple replicas
- Rate-limit reconciliation with `MaxConcurrentReconciles` and exponential backoff on errors

## See Also

kubernetes, kustomize, helm, argocd, cri

## References

- [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Kubebuilder Documentation](https://book.kubebuilder.io/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Operator SDK](https://sdk.operatorframework.io/)
- [OperatorHub.io](https://operatorhub.io/)
- [Custom Resource Definitions](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)
