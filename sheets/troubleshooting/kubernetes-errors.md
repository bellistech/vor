# Kubernetes Errors

Pod states, kubectl errors, RBAC denials, scheduling failures, admission rejections — verbatim text, root cause, and the diagnostic ladder for each.

## Setup

The Kubernetes troubleshooting flow follows a fixed ladder. Skip a step and you'll waste hours chasing a symptom.

```text
1. kubectl get pods/nodes/svc -o wide      # status overview
2. kubectl describe pod/node/svc <name>    # events, conditions, last-state
3. kubectl logs <pod> [-c container]       # current container output
4. kubectl logs <pod> --previous           # last crashed container
5. kubectl get events --sort-by=.lastTimestamp -A
6. kubectl exec -it <pod> -- /bin/sh       # interactive debug
```

The cluster has three layers, and a failure can live at any of them:

```text
cluster-level   →  kube-apiserver, etcd, kube-scheduler, kube-controller-manager, CoreDNS
node-level      →  kubelet, container runtime (containerd/CRI-O), kube-proxy, CNI
pod-level       →  Pod spec, container image, env, mounts, probes, securityContext
```

A symptom like "pod stuck Pending" can come from cluster-level (scheduler down), node-level (no nodes have capacity), or pod-level (nodeAffinity matches nothing). The diagnostic walks down: `get` → `describe` → `events` → `logs` → `exec`.

```bash
# Pod-level overview
kubectl get pod <name> -o yaml
kubectl describe pod <name>

# Node-level overview
kubectl get nodes -o wide
kubectl describe node <name>
kubectl top nodes               # requires metrics-server

# Cluster-level overview
kubectl get componentstatuses   # deprecated but still works
kubectl get pods -n kube-system
kubectl cluster-info dump       # massive but exhaustive
```

The "context" trap: `kubectl` is silent about which cluster you're hitting. Always confirm.

```bash
kubectl config current-context
kubectl config get-contexts
kubectl config use-context <name>
kubectl config view --minify
```

Cross-link: see `kubectl` for command basics, `kubectl-debug` for ephemeral containers and `kubectl debug node/X`, `troubleshooting/docker-errors` for image/runtime issues that surface in Kubernetes, `troubleshooting/dns-errors` for CoreDNS failures.

## Pod States Catalog

`kubectl get pods` shows the high-level Phase plus a derived "Status" column. The Phase is one of `Pending`, `Running`, `Succeeded`, `Failed`, `Unknown` — but the displayed Status is often a Reason like `ImagePullBackOff` or `CrashLoopBackOff`.

```bash
kubectl get pods
# NAME           READY   STATUS              RESTARTS   AGE
# web-abc123     0/1     Pending             0          5s
# api-def456     0/1     ContainerCreating   0          10s
# db-ghi789      1/1     Running             0          2m
# job-jkl012     0/1     Completed           0          5m
# worker-mno345  0/1     CrashLoopBackOff    7          12m
# old-pqr678     0/1     Terminating         0          1m
# evicted-stu901 0/1     Evicted             0          2h
```

### Pending

Pod has been accepted by the cluster but at least one container is not yet running. Two sub-states matter:

```text
1. Not yet scheduled    — no node assigned (look at FailedScheduling events)
2. Scheduled, not yet running — node assigned but image pulling, volume mounting,
                                init container running, or hit a Reason like
                                ImagePullBackOff
```

```bash
kubectl get pod <name> -o jsonpath='{.spec.nodeName}'
# If empty → not yet scheduled
# If set   → check container statuses
```

### ContainerCreating

Substate of Pending. Image is being pulled, volumes attached/mounted, secrets/configmaps mounted, network configured. If stuck here >2 minutes, drop to events.

```bash
kubectl describe pod <name> | grep -A5 Events:
# Common Reasons in this state:
#   Pulling     - downloading image
#   Pulled      - image done
#   Created     - container object created
#   Started     - process started → moves to Running
#   FailedMount - volume issue
```

### Running

At least one container is running, or in the process of starting/restarting. The READY column shows the running/desired counts (`1/1`, `2/3`). Running ≠ Ready — a container can be running but failing readiness probes.

```bash
kubectl get pod <name> -o jsonpath='{.status.containerStatuses[*].ready}'
# true  → passing readinessProbe (or no probe)
# false → not ready, removed from Service endpoints
```

### Succeeded

All containers terminated with exit 0 and won't restart. Normal end state for `Job` pods. Pods with `restartPolicy: Always` (the default) never reach Succeeded; only `Never` and `OnFailure` can.

```bash
kubectl get pod <name> -o jsonpath='{.status.phase}'
# Succeeded
```

### Failed

All containers terminated and at least one exited non-zero (or was killed by the system). Common with `restartPolicy: Never` Jobs. The `kubectl get pods` Status column may show `Error`, `OOMKilled`, `Evicted`, `DeadlineExceeded` — these are Reasons, not Phases.

### Unknown

The kubelet has not reported back to the apiserver for >40s (default node monitor grace period). Pod state is genuinely unknowable. Usually means:

```text
- Node lost network to apiserver
- kubelet process died
- Node OS crashed/rebooted
```

After 5 minutes (default `pod-eviction-timeout`), the controller-manager will mark the pod for deletion and reschedule.

### Terminating

Pod is in graceful shutdown. SIGTERM has been sent to PID 1 in each container; the kubelet waits up to `terminationGracePeriodSeconds` (default 30s) before SIGKILL. If your container ignores SIGTERM, this drags on.

```bash
kubectl get pod <name> -o jsonpath='{.metadata.deletionTimestamp}'
# If set → in Terminating phase
```

Stuck Terminating? See Recovery Patterns below.

### Evicted

The kubelet on the node forcibly removed the pod due to resource pressure (memory, ephemeral-storage, or PID exhaustion). The pod is gone but the API object lingers as a tombstone with `Status: Evicted` until garbage-collected or manually deleted.

```bash
kubectl get pods --field-selector=status.phase=Failed
kubectl get pods -A | grep Evicted
# Cleanup:
kubectl delete pods --field-selector=status.phase=Failed -A
```

## Pod Reasons Catalog

The Reason field in `kubectl describe pod` (under `Status.Reason` and per-container `State.Waiting.Reason`) is where the truth lives. Mapping from displayed Status to Phase:

```text
ImagePullBackOff         → Phase: Pending
ErrImagePull             → Phase: Pending (transient, becomes ImagePullBackOff)
ErrImageNeverPull        → Phase: Pending (imagePullPolicy: Never + image not local)
CrashLoopBackOff         → Phase: Running (technically) or Pending after first crash
ContainerCreating        → Phase: Pending
Init:Error               → Phase: Pending (init container failed)
Init:CrashLoopBackOff    → Phase: Pending (init container crash-looping)
Init:0/2                 → Phase: Pending (still on init container 1 of 2)
PodInitializing          → Phase: Pending (init done, main starting)
CreateContainerConfigError → Phase: Pending (Secret/ConfigMap/key missing)
CreateContainerError     → Phase: Pending (OCI runtime, securityContext, etc.)
InvalidImageName         → Phase: Pending (image string syntactically invalid)
SignatureValidationFailed → Phase: Pending (cosign/sigstore verification failed)
OOMKilled                → Phase: Failed or Running (then back to CrashLoopBackOff)
Error                    → Phase: Failed (non-zero exit, no specific Reason)
Completed                → Phase: Succeeded (Jobs only)
Evicted                  → Phase: Failed (kubelet eviction)
DeadlineExceeded         → Phase: Failed (Job activeDeadlineSeconds passed)
ContainerCannotRun       → Phase: Failed (entrypoint missing, permission)
ContainerStatusUnknown   → Phase: Pending (kubelet can't query runtime)
NodeLost                 → Phase: Running but unreachable
Shutdown                 → Phase: Failed (graceful node shutdown evicted)
Terminated               → Phase: depends on exit code
```

Read the Reason from the right place:

```bash
# Top-level (rare; usually empty)
kubectl get pod <name> -o jsonpath='{.status.reason}'

# Per-container current state
kubectl get pod <name> -o jsonpath='{.status.containerStatuses[*].state}'

# Per-container last termination
kubectl get pod <name> -o jsonpath='{.status.containerStatuses[*].lastState.terminated.reason}'
kubectl get pod <name> -o jsonpath='{.status.containerStatuses[*].lastState.terminated.exitCode}'
```

## ImagePullBackOff Diagnostic

Verbatim event text and the cause for each.

```text
Failed to pull image "myorg/myapp:1.2.3": rpc error: code = NotFound desc = failed to pull and unpack image "docker.io/myorg/myapp:1.2.3": failed to resolve reference "docker.io/myorg/myapp:1.2.3": docker.io/myorg/myapp:1.2.3: not found
```

Cause: tag does not exist in the registry. Typo in the tag, or the CI never pushed it. Verify with `docker pull` from a workstation or check the registry UI.

```text
Failed to pull image "myorg/myapp:1.2.3": Error response from daemon: pull access denied for myorg/myapp, repository does not exist or may require 'docker login': denied: requested access to the resource is denied
```

Cause: private registry, no credentials presented. Either the repository is private and `imagePullSecrets` is missing/wrong, or the image path is wrong (typo treated as private repo on Docker Hub).

```text
Failed to pull image "registry.example.com/app:v1": Error response from daemon: manifest for registry.example.com/app:v1 not found: manifest unknown: manifest unknown
```

Cause: registry is reachable, auth is fine, but the manifest for that tag/digest doesn't exist. Frequently happens after a registry GC or when pulling a digest that was overwritten.

```text
Failed to pull image "registry.example.com/app:v1": Error response from daemon: Get "https://registry.example.com/v2/": dial tcp: lookup registry.example.com on 10.96.0.10:53: no such host
```

Cause: DNS lookup from the node failed. CoreDNS issue, or the node has no upstream DNS, or the registry hostname is wrong. Note the resolver IP `10.96.0.10` — that's the in-cluster CoreDNS service. Image pulls actually happen from the **node**, not the pod, so node-level DNS matters.

```text
Failed to pull image "registry.example.com/app:v1": Error response from daemon: Get "https://registry.example.com/v2/": net/http: TLS handshake timeout
```

Cause: network reachable but TLS hung. Firewall stripping/mangling, MITM proxy, or registry overloaded. Try from the node directly: `crictl pull registry.example.com/app:v1`.

```text
Failed to pull image "registry.example.com/app@sha256:...": Error response from daemon: unauthorized: authentication required
```

Cause: pulling by digest from a private registry without auth. Same as access denied but with a digest reference.

```text
Failed to pull image "myimage": Error response from daemon: invalid reference format
```

Cause: image string has illegal characters (uppercase in repo name, missing tag separator, double colon). Surfaces as Reason `InvalidImageName`.

```text
ErrImageNeverPull
```

Cause: `imagePullPolicy: Never` and the image isn't already on the node. Used in dev with `kind` or `minikube` after side-loading; production pods rarely have this.

### imagePullSecrets configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: regcred
  namespace: myapp
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-of-~/.docker/config.json>
---
apiVersion: v1
kind: Pod
metadata:
  name: web
  namespace: myapp
spec:
  imagePullSecrets:
    - name: regcred
  containers:
    - name: web
      image: registry.example.com/private/web:1.0
```

Create the secret:

```bash
kubectl create secret docker-registry regcred \
  --docker-server=registry.example.com \
  --docker-username=<user> \
  --docker-password=<pass> \
  --docker-email=<email> \
  -n myapp
```

The secret **must live in the same namespace as the pod**. Copying across namespaces is one of the top-five gotchas:

```bash
kubectl get secret regcred -n source -o yaml \
  | sed 's/namespace: source/namespace: target/' \
  | kubectl apply -n target -f -
```

### Service-account-scoped imagePullSecrets

Attach to the ServiceAccount so every pod using it inherits without explicit `imagePullSecrets`:

```bash
kubectl patch serviceaccount default \
  -p '{"imagePullSecrets":[{"name":"regcred"}]}' \
  -n myapp
```

Diagnostic ladder:

```bash
# 1. Confirm exact error
kubectl describe pod <name> | grep -A3 'Failed to pull'

# 2. Check the secret exists in the pod's namespace
kubectl get secret -n <ns>

# 3. Decode and verify the secret
kubectl get secret regcred -n <ns> -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d

# 4. Try the pull from a node
ssh <node>
sudo crictl pull --creds <user>:<pass> registry.example.com/app:v1

# 5. DNS check from a debug pod (note: pod-level DNS, not node-level)
kubectl run debug --rm -it --image=busybox -- nslookup registry.example.com

# 6. Force a re-pull (delete kills the cached failure backoff)
kubectl delete pod <name>
```

## CrashLoopBackOff Diagnostic

```text
Back-off restarting failed container <name> in pod <pod>_<ns>(<uid>)
```

Cause: a container has terminated, the kubelet restarted it per `restartPolicy: Always`, and it terminated again. After repeated crashes, the kubelet imposes exponential back-off:

```text
Crash 1 → restart after 10s
Crash 2 → restart after 20s
Crash 3 → restart after 40s
Crash 4 → restart after 80s
Crash 5 → restart after 160s
Crash 6+ → restart after 300s (5m, capped)
```

The back-off resets after the container has run successfully for 10 minutes.

The single most useful command for crashloops:

```bash
kubectl logs <pod> --previous
# Or with a container name:
kubectl logs <pod> -c <container> --previous
```

`--previous` shows the **last container's** stdout/stderr — the one that just crashed. Without `--previous` you see the current (still-starting) container, which has nothing yet.

Common causes:

```text
- App crashes on startup            → app stack trace in --previous logs
- Missing required env var          → "config: NEW_RELIC_KEY is required"
- Database/dependency unavailable   → "could not connect to postgres:5432"
- Bad config file                   → "yaml: line 12: did not find expected key"
- File-not-found on mount           → "open /etc/cfg/app.yaml: no such file"
- Permission denied                 → "Permission denied: '/var/log/app.log'"
- Image entrypoint wrong            → "exec: \"./app\": stat ./app: no such file"
- Liveness probe killing on cold start → SIGTERM in logs, no app error
```

The liveness-probe-killing-healthy-pod case is sneaky. Container starts fine, app boots in 30s, liveness probe with `initialDelaySeconds: 5` and `failureThreshold: 3` kills it at second 35. Logs show a graceful shutdown sequence rather than a crash:

```bash
kubectl describe pod <name> | grep -A2 Liveness:
# Liveness:  http-get http://:8080/healthz delay=5s timeout=1s period=10s #success=1 #failure=3

kubectl logs <pod> --previous
# 2026-04-25T12:00:00 starting up...
# 2026-04-25T12:00:30 listening on :8080
# 2026-04-25T12:00:35 received SIGTERM, shutting down...
# (no panic, no error)

kubectl get events --field-selector involvedObject.name=<pod>
# Liveness probe failed: HTTP probe failed with statuscode: 503
```

Fix: add a `startupProbe` that gives the app time to come up, then a more aggressive `livenessProbe` after.

Diagnostic ladder:

```bash
# 1. Confirm it's actually crashing (vs. probe failure)
kubectl get pod <name> -o jsonpath='{.status.containerStatuses[*].lastState.terminated}'
# Look for: exitCode, reason ("Error", "OOMKilled", "Completed"), startedAt, finishedAt

# 2. The crashed container's last words
kubectl logs <pod> --previous --tail=200

# 3. Events for context
kubectl describe pod <name> | tail -30

# 4. If logs are silent, run interactively
kubectl run shell --rm -it --image=<same-image> --command -- sh
# Try the entrypoint manually

# 5. If env-var-related, dump env
kubectl exec <pod> -- env  # only works if it's running long enough
# Or:
kubectl get pod <name> -o jsonpath='{.spec.containers[*].env}'

# 6. Disable liveness temporarily to see if app stabilizes
kubectl edit deployment <name>
# Comment out livenessProbe → save → wait
```

## OOMKilled

```text
Last State:     Terminated
  Reason:       OOMKilled
  Exit Code:    137
```

Exit code 137 = `128 + 9` = SIGKILL. The kernel's cgroup OOM killer fired because the container exceeded `resources.limits.memory`.

```bash
kubectl describe pod <name> | grep -A5 'Last State'
# Last State:     Terminated
#   Reason:       OOMKilled
#   Exit Code:    137
#   Started:      Sat, 25 Apr 2026 12:00:00 +0000
#   Finished:     Sat, 25 Apr 2026 12:05:23 +0000
```

The "request matters for scheduling, limit matters for OOM-killer" rule:

```yaml
resources:
  requests:
    memory: 256Mi   # scheduler reserves this on a node
    cpu: 100m
  limits:
    memory: 512Mi   # cgroup hard cap; exceed → OOMKill
    cpu: 500m
```

Per-container OOM kills only that container; the pod keeps running and the kubelet restarts the container per restartPolicy. This shows as the Reason on the *previous* state while the pod stays Running. Whole-pod kills happen on **node-level eviction** (memory pressure on the node), which is different — see Node-Level Eviction below.

### JVM gotcha

OpenJDK <8u191 ignored cgroup limits and read host memory. Even modern JVMs need explicit flags to behave:

```bash
# OpenJDK 11+
java -XX:+UseContainerSupport -XX:MaxRAMPercentage=75.0 -jar app.jar

# Pre-11 (avoid)
java -Xmx384m -jar app.jar
```

Without `UseContainerSupport`, JVM heap defaults to 1/4 of host RAM (could be 16GB on a 64GB node) → OOMKilled when limit is 512Mi.

### Node.js gotcha

V8 default heap is ~1.5GB. Set explicitly:

```bash
node --max-old-space-size=384 server.js   # MB
```

In a 512Mi container, leave headroom for non-heap memory (stack, off-heap buffers, native modules).

Diagnostic ladder:

```bash
# 1. Confirm OOM
kubectl get pod <name> -o jsonpath='{.status.containerStatuses[*].lastState.terminated.reason}'

# 2. Look at memory usage just before death (if metrics-server)
kubectl top pod <name>           # current
kubectl top pod <name> --containers

# 3. Compare requested vs limit
kubectl get pod <name> -o jsonpath='{.spec.containers[*].resources}'

# 4. Long-term: enable cAdvisor metrics in Prometheus
#    container_memory_working_set_bytes vs limit
```

## Node-Level Eviction

```text
Status: Failed
Reason: Evicted
Message: The node was low on resource: memory. Container <name> was using 512Mi, request is 256Mi, limit is 512Mi.
```

Variants of the message:

```text
The node was low on resource: ephemeral-storage. Container X was using 5Gi, request is unset, has larger consumption of ephemeral-storage than other pods.
The node was low on resource: pid. Container X was using 4096, request is unset.
```

The kubelet evicts pods when it crosses a hard threshold:

```text
--eviction-hard=memory.available<100Mi,nodefs.available<10%,imagefs.available<15%,nodefs.inodesFree<5%,pid.available<10%
--eviction-soft=memory.available<300Mi    # with --eviction-soft-grace-period
```

### Eviction priority (QoS class)

Pods get a QoS class derived from their requests/limits:

```text
Guaranteed → requests == limits for ALL resources on ALL containers
Burstable  → some requests set, but not Guaranteed
BestEffort → no requests OR limits set anywhere
```

Eviction order under memory pressure:

```text
1. BestEffort pods using more memory than expected (no request to compare)
2. Burstable pods using more than their request
3. Burstable pods using less than their request (rare, only if needed)
4. Guaranteed pods (last resort, only if system pods need memory)
```

Within a tier, `Priority` (PodPriorityClass) breaks ties — lower priority dies first.

### The "Evicted ghost" problem

Evicted pods stay as `Failed` API objects forever (or until namespace deletion / GC). They clutter `kubectl get pods` output but consume no resources.

```bash
# Cleanup all evicted in all namespaces
kubectl get pods -A --field-selector=status.phase=Failed -o json \
  | jq -r '.items[] | select(.status.reason=="Evicted") | "\(.metadata.namespace) \(.metadata.name)"' \
  | while read ns name; do kubectl delete pod -n "$ns" "$name"; done

# Or simpler:
kubectl delete pods -A --field-selector=status.phase=Failed
```

Diagnostic ladder:

```bash
# 1. Find which node had pressure
kubectl get pod <name> -o jsonpath='{.spec.nodeName}'

# 2. Check node conditions
kubectl describe node <node> | grep -A5 Conditions
# MemoryPressure   True/False
# DiskPressure     True/False
# PIDPressure      True/False

# 3. Per-pod resource usage
kubectl top pod -A --sort-by=memory

# 4. Per-node usage
kubectl top nodes
kubectl describe node <node> | grep -A10 'Allocated resources'

# 5. kubelet eviction events on that node
kubectl get events --field-selector involvedObject.kind=Node,involvedObject.name=<node>
```

## CreateContainerConfigError

The kubelet pulled the image but cannot construct the container's environment because a referenced Secret/ConfigMap is missing or has a missing key.

```text
Error: couldn't find key DB_PASSWORD in Secret myapp/db-creds
```

Cause: pod refers to `secretKeyRef.key: DB_PASSWORD` but the Secret has only `DB_URL` and `DB_USER`.

```text
Error: couldn't find key application.yaml in ConfigMap myapp/app-config
```

Cause: pod mounts `configMap.items[].key: application.yaml` but ConfigMap has only `config.yaml`.

```text
Error: secret "db-creds" not found
```

Cause: the Secret doesn't exist in the pod's namespace (most often, deployed to wrong namespace, or never applied).

```text
Error: configmap "app-config" not found
```

Cause: same as above for ConfigMaps.

```text
User "system:serviceaccount:myapp:default" cannot get resource "secrets" in API group "" in the namespace "myapp"
```

Cause: the pod's ServiceAccount lacks RBAC to read the Secret. Either the SA has no Role/RoleBinding for secrets, or the Secret is in a different namespace and the SA has no ClusterRole.

```bash
# Pod referencing the secret
spec:
  containers:
    - env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-creds
              key: DB_PASSWORD
```

Diagnostic ladder:

```bash
# 1. Get the exact error
kubectl describe pod <name> | grep -i 'couldn'\''t\|not found'

# 2. Confirm the Secret/ConfigMap exists in the pod's namespace
kubectl get secret db-creds -n <ns>
kubectl get configmap app-config -n <ns>

# 3. List the keys in the Secret
kubectl get secret db-creds -n <ns> -o jsonpath='{.data}' | jq 'keys'

# 4. Decode a specific key
kubectl get secret db-creds -n <ns> -o jsonpath='{.data.DB_PASSWORD}' | base64 -d

# 5. List ConfigMap keys
kubectl get cm app-config -n <ns> -o jsonpath='{.data}' | jq 'keys'
```

## CreateContainerError

The OCI runtime (containerd/CRI-O via runc/crun) refused to create the container. Image is pulled, env is built, but the runtime says no.

```text
Error: failed to create containerd task: failed to create shim task: OCI runtime create failed: runc create failed: unable to start container process: exec: "./bin/app": stat ./bin/app: no such file or directory: unknown
```

Cause: ENTRYPOINT/CMD points to a binary that doesn't exist in the image at that path.

```text
Error: container has runAsNonRoot and image will run as root (pod: "X(uid)", container: app)
```

Cause: pod sets `securityContext.runAsNonRoot: true` and the image declares no `USER` or has `USER root`. The runtime refuses to start.

```text
Error: container has runAsNonRoot and image has non-numeric user (myuser), cannot verify user is non-root
```

Cause: the image has `USER myuser` (a name, not a UID). Kubernetes can't introspect /etc/passwd to resolve "myuser" → 1000, so it errs on the side of caution. Fix: set `USER 1000` in the Dockerfile, or set `securityContext.runAsUser: 1000` in the pod (which overrides).

```text
Error: failed to create containerd task: failed to create shim: OCI runtime create failed: runc create failed: unable to start container process: error during container init: write /proc/self/oom_score_adj: permission denied: unknown
```

Cause: usually a security profile (AppArmor, SELinux, seccomp) blocking. Check the node's runtime logs.

```text
Error: failed to create containerd task: ... unable to apply mount: ... read-only file system
```

Cause: `securityContext.readOnlyRootFilesystem: true` and the image's entrypoint tries to write to a path not mounted as a writable volume.

### PodSecurity admission interaction

`kubectl describe` may show the underlying violation:

```text
Error creating: pods "web-" is forbidden: violates PodSecurity "restricted:latest":
  allowPrivilegeEscalation != false (container "web" must set securityContext.allowPrivilegeEscalation=false),
  unrestricted capabilities (container "web" must set securityContext.capabilities.drop=["ALL"]),
  runAsNonRoot != true (pod or container "web" must set securityContext.runAsNonRoot=true),
  seccompProfile (pod or container "web" must set securityContext.seccompProfile.type to "RuntimeDefault" or "Localhost")
```

This rejection happens at the **admission** layer (before pod is even created), but a similar policy enforced lower in the stack (PSP, OPA Gatekeeper) can surface as CreateContainerError.

Fix:

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: web
      image: myorg/web:1.0
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities:
          drop: [ALL]
```

## Scheduling Failures (FailedScheduling Events)

A pod in `Pending` with no `nodeName` and a `FailedScheduling` event tells you the scheduler tried but couldn't place it. The message format is:

```text
0/<N> nodes are available: <reasons separated by commas>
```

### Affinity / selector mismatch

```text
Warning  FailedScheduling  default-scheduler  0/5 nodes are available: 5 node(s) didn't match Pod's node affinity/selector. preemption: 0/5 nodes are available: 5 Preemption is not helpful for scheduling..
```

Cause: `nodeSelector` or `nodeAffinity` matches no nodes. Verify labels:

```bash
kubectl get nodes --show-labels
kubectl get nodes -l <label-key>=<label-value>
```

### Insufficient resources

```text
Warning  FailedScheduling  default-scheduler  0/5 nodes are available: 3 Insufficient cpu, 2 Insufficient memory. preemption: 0/5 nodes are available: 5 No preemption victims found for incoming pod..
```

Cause: every node either lacks free CPU or free memory **based on requests** (not actual usage — scheduler is request-based). The scheduler considers `requests`, not `limits`, against `node.allocatable`.

```bash
# Per-node allocatable
kubectl describe node <node> | grep -A5 Allocatable
kubectl describe node <node> | grep -A5 'Allocated resources'

# Find the heaviest pods on a node
kubectl get pods -A --field-selector spec.nodeName=<node> \
  -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,CPU:.spec.containers[*].resources.requests.cpu,MEM:.spec.containers[*].resources.requests.memory'
```

### Untolerated taints

```text
Warning  FailedScheduling  default-scheduler  0/5 nodes are available: 5 node(s) had untolerated taint {dedicated=gpu:NoSchedule}. preemption: 0/5 nodes are available: 5 Preemption is not helpful for scheduling..
```

Cause: nodes have a taint and the pod has no matching toleration. Add:

```yaml
spec:
  tolerations:
    - key: dedicated
      operator: Equal
      value: gpu
      effect: NoSchedule
```

The taint/toleration matrix:

```text
Taint effect       Behavior
NoSchedule         Don't schedule new pods without tolerations
PreferNoSchedule   Avoid scheduling but allow if no choice
NoExecute          Evict existing pods without tolerations + don't schedule new
```

A toleration matches if `key`, `effect`, and either (`operator: Exists`) or (`operator: Equal` + `value`) align.

```yaml
# Tolerate any taint with this key (any effect)
tolerations:
  - key: node-role.kubernetes.io/master
    operator: Exists

# Tolerate everything (rare, control-plane-only)
tolerations:
  - operator: Exists
```

### Free port conflict

```text
Warning  FailedScheduling  default-scheduler  0/5 nodes are available: 5 node(s) didn't have free ports for the requested pod ports. preemption: 0/5 nodes are available: 5 No preemption victims found for incoming pod..
```

Cause: pod uses `hostPort` and another pod on every candidate node already binds that port.

```yaml
# This is the offender
ports:
  - containerPort: 8080
    hostPort: 8080   # conflicts on shared nodes
```

Generally avoid `hostPort` unless you really need it (DaemonSets binding node-local services). Use a Service instead.

### Volume node affinity conflict

```text
0/5 nodes are available: 5 node(s) had volume node affinity conflict.
```

Cause: a PV (typically `local` or zone-pinned EBS/PD) has `nodeAffinity` requiring a specific node/zone, and the pod can't go to a node satisfying both its own constraints and the PV's. Often a zone mismatch:

```bash
kubectl get pv <pv> -o yaml | grep -A10 nodeAffinity
kubectl get nodes -L topology.kubernetes.io/zone
```

### Volume affinity conflict

```text
0/5 nodes are available: 5 node(s) had volume affinity conflict.
```

Cause: similar but for inter-volume affinity (different volumes pinned to different zones cannot all be mounted on one node).

### Unbound PVC

```text
0/5 nodes are available: 5 pod has unbound immediate PersistentVolumeClaims.
```

Cause: PVC has no PV bound and the StorageClass isn't dynamically provisioning (or provisioning failed). For `volumeBindingMode: WaitForFirstConsumer`, this message is **expected and benign** — the PV will be created when scheduling actually picks a node.

```bash
kubectl get pvc <pvc>
# Look at STATUS column: Pending vs Bound
kubectl describe pvc <pvc>
# Events at the bottom show provisioning attempts
```

### Pod anti-affinity

```text
0/5 nodes are available: 5 node(s) didn't satisfy existing pods anti-affinity rules.
```

Cause: another pod with `podAntiAffinity` is already on every candidate node, blocking this pod from co-locating. Common with "spread one replica per node" patterns:

```yaml
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app: web
        topologyKey: kubernetes.io/hostname
```

If `replicas > nodes`, replicas N+1 stay Pending forever. Either reduce replicas, scale nodes, or switch to `preferredDuringSchedulingIgnoredDuringExecution`.

### Node unschedulable

```text
0/5 nodes are available: 5 node(s) were unschedulable.
```

Cause: every node is cordoned (`kubectl cordon`). Common during cluster maintenance.

```bash
kubectl get nodes
# STATUS shows: Ready,SchedulingDisabled
kubectl uncordon <node>
```

### Multi-reason composite

```text
0/10 nodes are available: 1 node(s) had untolerated taint {dedicated=gpu:NoSchedule}, 3 Insufficient cpu, 2 Insufficient memory, 4 node(s) didn't match Pod's node affinity/selector.
```

The numbers always add up to N (10 here) — every node has at least one reason. To pass, **one** node must have **zero** reasons.

### topologySpreadConstraints

```yaml
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app: web
```

If `whenUnsatisfiable: DoNotSchedule` and zones can't satisfy the skew, you'll see:

```text
0/5 nodes are available: 5 node(s) didn't match pod topology spread constraints.
```

Set `whenUnsatisfiable: ScheduleAnyway` for soft spreading.

## FailedMount Events

Volume problems surface as `FailedMount` events on the pod after scheduling but during `ContainerCreating`.

```text
Warning  FailedMount  kubelet  Unable to attach or mount volumes: unmounted volumes=[data], unattached volumes=[data]: timed out waiting for the condition
```

Cause: generic timeout. Drill into the specific reason via the previous events.

```text
Warning  FailedMount  kubelet  MountVolume.SetUp failed for volume "data" : mount failed: exit status 32. Mounting command: mount. Mounting arguments: -t nfs nfs.example.com:/exports/data /var/lib/kubelet/.../data Output: mount.nfs: Connection timed out
```

Cause: NFS server unreachable from the node. Check firewalls, NFS daemon, exports config.

```text
Warning  FailedMount  kubelet  MountVolume.SetUp failed for volume "config-vol" : configmap "app-config" not found
```

Cause: pod mounts a ConfigMap that doesn't exist in the namespace. Same fix as CreateContainerConfigError but surfaces here when the volume is the issue rather than env.

```text
Warning  FailedAttachVolume  attachdetach-controller  AttachVolume.Attach failed for volume "pvc-..." : rpc error: code = Internal desc = Could not attach volume "..." to node "...": ...
```

Cause: CSI driver issue. The driver-specific suffix tells you which driver and why (zone mismatch, quota exceeded, attachment limit).

```text
Warning  FailedAttachVolume  attachdetach-controller  Multi-Attach error for volume "pvc-..." Volume is already used by pod(s) <other-pod>
```

Cause: pod is trying to mount a `ReadWriteOnce` (RWO) PV that's still attached to another node. Common during rolling updates if the new pod schedules on a different node before the old one detaches. Also happens when a node goes NotReady — the volume is "stuck" attached to the dead node.

Fix:

```bash
# 1. Force-detach (rare; usually let the controller handle it)
# Look at VolumeAttachment objects:
kubectl get volumeattachment | grep <pv-name>
kubectl delete volumeattachment <name>

# 2. Or move pod to same node as the existing attachment
# 3. Or use ReadWriteMany (RWX) volumes (NFS, CephFS, Azure Files)
```

### Access mode mismatches

```text
Warning  FailedMount  kubelet  failed to provision volume with StorageClass "fast": invalid AccessModes [ReadWriteMany]: only AccessModes [ReadWriteOnce] are supported
```

Cause: PVC requests RWX but the StorageClass only supports RWO (block storage like AWS EBS, GCE PD).

```yaml
# Mismatch
kind: PersistentVolumeClaim
spec:
  accessModes: [ReadWriteMany]
  storageClassName: gp3   # AWS EBS, RWO only
```

### StorageClass missing or wrong volumeBindingMode

```text
no persistent volumes available for this claim and no storage class is set
```

Cause: PVC has neither `storageClassName` nor an explicit volumeName, and there's no default StorageClass. Set a default:

```bash
kubectl patch storageclass standard \
  -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

`volumeBindingMode: Immediate` provisions the PV at PVC creation; `WaitForFirstConsumer` waits until a pod is scheduled. The latter prevents zone mismatches with multi-AZ clusters.

```bash
# Diagnostic ladder for volume issues
kubectl describe pod <pod> | grep -A5 'Volumes:'
kubectl get pvc <pvc> -n <ns>
kubectl describe pvc <pvc> -n <ns>      # Events at bottom
kubectl get pv | grep <pvc>
kubectl get sc                           # is one default?
kubectl get volumeattachment | grep <pv>
```

## Liveness / Readiness Probe Failures

```text
Warning  Unhealthy  kubelet  Liveness probe failed: HTTP probe failed with statuscode: 503
Warning  Unhealthy  kubelet  Liveness probe failed: Get "http://10.244.1.5:8080/healthz": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
Warning  Unhealthy  kubelet  Liveness probe failed: Get "http://10.244.1.5:8080/healthz": dial tcp 10.244.1.5:8080: connect: connection refused
Warning  Unhealthy  kubelet  Readiness probe failed: HTTP probe failed with statuscode: 500
Warning  Unhealthy  kubelet  Startup probe failed: ...
```

Probe semantics:

```text
Startup    — disables liveness/readiness until it succeeds; for slow-starting apps
Readiness  — failing → pod removed from Service endpoints (but not killed)
Liveness   — failing → kubelet restarts the container
```

The startup-probe-prevents-aggressive-liveness pattern:

```yaml
containers:
  - name: app
    startupProbe:
      httpGet:
        path: /healthz
        port: 8080
      failureThreshold: 30        # 30 attempts × 10s = 5 min max
      periodSeconds: 10
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
      periodSeconds: 10
      failureThreshold: 3         # only kicks in after startup succeeds
    readinessProbe:
      httpGet:
        path: /ready
        port: 8080
      periodSeconds: 5
      failureThreshold: 2
```

### Probe types

```yaml
# HTTP
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
    httpHeaders:
      - name: X-Custom-Header
        value: probe

# TCP
livenessProbe:
  tcpSocket:
    port: 5432

# Exec
livenessProbe:
  exec:
    command: [sh, -c, "pg_isready -U postgres"]

# gRPC (Kubernetes 1.24+, GA in 1.27)
livenessProbe:
  grpc:
    port: 9090
    service: health   # optional, defaults to all services
```

Verify a probe manually from inside the cluster:

```bash
# HTTP probe equivalent
kubectl exec <pod> -- wget -O- -T 1 http://localhost:8080/healthz
# Or curl
kubectl exec <pod> -- curl -fsS http://localhost:8080/healthz

# TCP
kubectl exec <pod> -- nc -z localhost 5432

# Exec
kubectl exec <pod> -- pg_isready -U postgres
```

## Service / Networking Failures

### Service has no endpoints

```text
no endpoints available for service "myservice"
```

Cause: Service selector matches no Pods (all unready, label typo, no replicas).

```bash
kubectl get svc myservice -o jsonpath='{.spec.selector}'
# {"app":"web","tier":"frontend"}

kubectl get pods -l app=web,tier=frontend
# (none)  →  selector typo or no pods labeled

kubectl get endpoints myservice
# NAME        ENDPOINTS    AGE
# myservice   <none>       5m

# When endpoints are empty, kubectl describe svc shows:
kubectl describe svc myservice | grep Endpoints
# Endpoints:    <none>
```

If pods exist with the right labels but endpoints are empty, the pods are **not Ready** (failing readinessProbe). The Service only adds Ready pods to its Endpoints object.

```bash
kubectl get pods -l app=web -o wide
# READY column shows 0/1 → readinessProbe failing

kubectl describe pod <pod> | grep -A2 Readiness:
kubectl describe pod <pod> | grep -i unhealthy
```

### Endpoints already exists

```text
Error from server (AlreadyExists): error when creating "...": endpoints "myservice" already exists
```

Cause: stale Endpoints object from a previous Service. Usually after a manual create-then-recreate. Delete the Endpoints (controller will recreate from selector):

```bash
kubectl delete endpoints myservice
```

### IPv6 family error

```text
Service "myservice" is invalid: spec.ipFamilies[0]: Invalid value: "IPv6": Cluster does not have IPv6 family
```

Cause: cluster wasn't configured with `--service-cluster-ip-range` containing an IPv6 CIDR. Either dual-stack the cluster or use IPv4 in the Service.

### DNS lookup failing inside pods

```text
dial tcp: lookup myservice.myns on 10.96.0.10:53: no such host
```

Cause: in-cluster DNS resolution failing. The resolver IP `10.96.0.10` is the CoreDNS Service ClusterIP (default for kubeadm).

Diagnostic:

```bash
# 1. CoreDNS pods running?
kubectl get pods -n kube-system -l k8s-app=kube-dns
# Should be 2+ Running

# 2. CoreDNS service has endpoints?
kubectl get endpoints -n kube-system kube-dns

# 3. From a debug pod, query directly
kubectl run debug --rm -it --image=busybox -- sh
nslookup kubernetes.default.svc.cluster.local
nslookup myservice.myns.svc.cluster.local

# 4. Check resolv.conf in pods
kubectl run debug --rm -it --image=busybox -- cat /etc/resolv.conf
# search myns.svc.cluster.local svc.cluster.local cluster.local
# nameserver 10.96.0.10
# options ndots:5
```

The `ndots:5` option means any name with fewer than 5 dots is tried with the search domains first. `myservice` becomes:

```text
myservice.myns.svc.cluster.local      (tried first)
myservice.svc.cluster.local
myservice.cluster.local
myservice                              (last)
```

This is why FQDNs (with trailing dot) skip the search list:

```bash
nslookup api.example.com.    # absolute, no search
```

### Service typo or wrong namespace

```text
dial tcp: lookup my-svc on 10.96.0.10:53: no such host
```

Cause: wrong service name, or service exists in a different namespace. Within the same namespace, use the bare service name. Cross-namespace requires `<svc>.<ns>` or `<svc>.<ns>.svc.cluster.local`.

```bash
# This pod is in 'frontend' namespace
# Service 'api' is in 'backend' namespace

# This fails:
curl http://api/

# This works:
curl http://api.backend/
curl http://api.backend.svc.cluster.local/
```

### hostNetwork interaction with kube-proxy

A pod with `hostNetwork: true` skips the cluster network (no pod IP) and binds directly on the node. It bypasses kube-proxy for inbound, so Service resolution works *to* it but `iptables` rules don't redirect *from* it via ClusterIP unless kube-proxy is on the node.

```yaml
spec:
  hostNetwork: true
  dnsPolicy: ClusterFirstWithHostNet  # required to use cluster DNS
```

Without `dnsPolicy: ClusterFirstWithHostNet`, DNS queries hit the node's resolv.conf instead of CoreDNS — and your service names won't resolve.

## kubectl Connection Errors

```text
The connection to the server localhost:8080 was refused - did you specify the right host or port?
```

Cause: no kubeconfig (or empty). kubectl falls back to `localhost:8080` (Kubernetes' historical default). Check `KUBECONFIG`, `~/.kube/config`, and current context.

```bash
echo $KUBECONFIG
ls -la ~/.kube/config
kubectl config view
kubectl config current-context
```

```text
Unable to connect to the server: x509: certificate signed by unknown authority
```

Cause: kubeconfig CA bundle doesn't match the apiserver cert. Three sub-cases:

```text
1. apiserver cert was rotated (esp. on managed clusters)  → re-fetch kubeconfig
2. Kubeconfig is for a different cluster                  → check current-context
3. MITM (proxy) intercepting TLS                          → check corp proxy
```

```bash
# Inspect what cert the server is presenting
openssl s_client -connect <apiserver>:6443 -showcerts < /dev/null

# Compare with kubeconfig CA
kubectl config view --raw -o jsonpath='{.clusters[*].cluster.certificate-authority-data}' | base64 -d
```

```text
Unable to connect to the server: dial tcp 10.0.0.1:6443: i/o timeout
```

Cause: network can't reach apiserver. VPN down, firewall, NAT, wrong IP after node reboot.

```text
Unable to connect to the server: net/http: TLS handshake timeout
```

Cause: TCP connects but TLS hangs. Apiserver overloaded, intermediary stripping/buffering TLS, MTU issues with VPN.

```text
error: You must be logged in to the server (Unauthorized)
```

Cause: token expired (most common with OIDC, AWS IAM Authenticator, exec credentials). For OIDC, re-login. For AWS:

```bash
aws eks update-kubeconfig --name <cluster> --region <region>
```

For service-account tokens in pods, the token rotates automatically (Bound Service Account Tokens) — only an issue if the workload caches the token across rotation.

```text
error: kubeconfig: unknown server "myserver"
```

Cause: kubeconfig file references a server name that doesn't exist in `clusters:`. Likely manual edit broke the file.

```text
error: current-context is not set in config
```

Cause: kubeconfig has no `current-context:` field. Set one:

```bash
kubectl config use-context <name>
# or edit ~/.kube/config
```

```text
error: couldn't get current server API group list: ...: dial tcp ...: i/o timeout
```

Cause: kubectl tried to discover the API but couldn't reach apiserver. Almost always network/DNS. Same as the i/o timeout above but during discovery rather than the main request.

Diagnostic ladder:

```bash
# 1. Confirm context and cluster
kubectl config current-context
kubectl config view --minify

# 2. Raw apiserver reachability
APISERVER=$(kubectl config view -o jsonpath='{.clusters[?(@.name=="<cluster-name>")].cluster.server}')
curl -k $APISERVER/healthz

# 3. With auth (uses kubectl's current credentials)
kubectl get --raw /healthz
kubectl get --raw /readyz
kubectl get --raw /livez

# 4. Server version (lightest discovery call)
kubectl version

# 5. Verbose mode shows the full HTTP exchange
kubectl get pods -v=8 2>&1 | head -50
```

## RBAC Denials

```text
Error from server (Forbidden): pods is forbidden: User "alice@example.com" cannot list resource "pods" in API group "" in the namespace "default"
```

```text
Error from server (Forbidden): clusterroles.rbac.authorization.k8s.io "edit" is forbidden: User "alice@example.com" cannot create resource "clusterroles" in API group "rbac.authorization.k8s.io" at the cluster scope
```

The error tells you everything:

```text
User      → "alice@example.com"   (the principal)
Verb      → "list" / "create"
Resource  → "pods" / "clusterroles"
API Group → "" (core) / "rbac.authorization.k8s.io"
Scope     → namespace "default" / cluster scope
```

Two common diagnostic commands:

```bash
# Can the current user do X?
kubectl auth can-i list pods
kubectl auth can-i create deployments -n production
kubectl auth can-i '*' '*' --all-namespaces    # admin check

# As a different user / serviceaccount
kubectl auth can-i list pods --as alice@example.com
kubectl auth can-i list pods --as system:serviceaccount:myns:default

# Enumerate everything a user can do
kubectl auth can-i --list
kubectl auth can-i --list --as system:serviceaccount:myns:default
kubectl auth can-i --list --as system:serviceaccount:myns:default -n othernamespace
```

`--list` output:

```text
Resources                   Non-Resource URLs   Resource Names    Verbs
pods                        []                  []                [get list watch]
pods/log                    []                  []                [get]
configmaps                  []                  []                [get list]
secrets                     []                  []                [get]
                            [/api/*]            []                [get]
```

### ServiceAccount RBAC pattern

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: log-reader
  namespace: myapp
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: myapp
  name: pod-log-reader
rules:
  - apiGroups: [""]
    resources: [pods, pods/log]
    verbs: [get, list, watch]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: log-reader-binding
  namespace: myapp
subjects:
  - kind: ServiceAccount
    name: log-reader
    namespace: myapp
roleRef:
  kind: Role
  name: pod-log-reader
  apiGroup: rbac.authorization.k8s.io
```

Then in the pod:

```yaml
spec:
  serviceAccountName: log-reader
  containers:
    - name: app
      image: ...
```

### Cross-namespace permissions

A `Role` is namespaced. To grant a SA in `ns-a` access to resources in `ns-b`, create a `Role` in `ns-b` and a `RoleBinding` in `ns-b` referencing the SA in `ns-a`:

```yaml
# Role in target namespace (ns-b)
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: ns-b
  name: configmap-reader
rules:
  - apiGroups: [""]
    resources: [configmaps]
    verbs: [get, list]
---
# RoleBinding in target namespace, referencing SA in different namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: ns-b
  name: ns-a-sa-can-read-cm
subjects:
  - kind: ServiceAccount
    name: my-sa
    namespace: ns-a
roleRef:
  kind: Role
  name: configmap-reader
  apiGroup: rbac.authorization.k8s.io
```

For cross-cluster-scope access, use a `ClusterRole` + `ClusterRoleBinding`.

### Aggregated ClusterRoles

Default ClusterRoles like `admin`, `edit`, `view` are *aggregated* — they include any ClusterRole with the right label:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: aggregate-edit-myresource
  labels:
    rbac.authorization.k8s.io/aggregate-to-edit: "true"
rules:
  - apiGroups: [my.example.com]
    resources: [myresources]
    verbs: [get, list, watch, create, update, patch, delete]
```

Now anyone with `edit` automatically gets these permissions. Use this when you ship a CRD and want to extend default roles without editing them.

Diagnostic ladder:

```bash
# 1. Confirm the principal
kubectl auth whoami    # 1.27+
# Or:
kubectl config view --minify -o jsonpath='{.users[*].name}'

# 2. Test the specific verb
kubectl auth can-i <verb> <resource> -n <ns>

# 3. Find which Role binds you
kubectl get rolebindings,clusterrolebindings -A -o json \
  | jq '.items[] | select(.subjects[]?.name=="<principal>") | {kind:.kind, name:.metadata.name, ns:.metadata.namespace, role:.roleRef.name}'

# 4. Inspect the Role's rules
kubectl describe role <name> -n <ns>
kubectl describe clusterrole <name>

# 5. Test as the SA from inside a pod
kubectl run test --rm -it \
  --image=bitnami/kubectl \
  --serviceaccount=<sa> \
  -- get pods
```

## Admission Controller Rejections

Admission webhooks intercept create/update/delete and can mutate or validate.

### Webhook denial

```text
Error from server (Forbidden): admission webhook "policy.kyverno.svc" denied the request: 
  validation error: Privileged containers are not allowed. Rule autogen-default-validate failed at path /spec/template/spec/containers/0/securityContext/privileged/
```

The exact text after `denied the request:` is from the webhook itself.

### Webhook unreachable

```text
Internal error occurred: failed calling webhook "validation.gatekeeper.sh": failed to call webhook: Post "https://gatekeeper-webhook-service.gatekeeper-system.svc:443/v1/admit?timeout=3s": context deadline exceeded
```

Cause: the webhook's service has no Ready endpoints, or the apiserver can't reach it. If `failurePolicy: Fail` (default for security policies), this **blocks** all matching operations cluster-wide.

Recovery: temporarily set `failurePolicy: Ignore` or delete the webhook config:

```bash
kubectl get validatingwebhookconfigurations
kubectl edit validatingwebhookconfiguration <name>
# Change failurePolicy: Fail → Ignore (or scope timeoutSeconds down)

# Or delete entirely (use carefully):
kubectl delete validatingwebhookconfiguration <name>
```

### Server-side schema validation (1.25+)

```text
error: error validating "deploy.yaml": error validating data: ValidationError(Deployment.spec.template.spec.containers[0]): unknown field "imagePullPolicies" in io.k8s.api.core.v1.Container
```

Cause: typo in a field name. With `kubectl apply --validate=true` (default since 1.25 server-side), the apiserver validates against the OpenAPI schema.

### PodSecurity admission

The built-in `PodSecurity` admission plugin enforces the Pod Security Standards (Privileged / Baseline / Restricted) at the namespace level via labels:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: myapp
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/enforce-version: latest
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

Violation message:

```text
Error from server (Forbidden): pods "web-abc-123" is forbidden: violates PodSecurity "restricted:latest":
  allowPrivilegeEscalation != false (containers "web", "sidecar" must set securityContext.allowPrivilegeEscalation=false),
  unrestricted capabilities (containers "web", "sidecar" must set securityContext.capabilities.drop=["ALL"]),
  runAsNonRoot != true (pod or containers "web", "sidecar" must set securityContext.runAsNonRoot=true),
  seccompProfile (pod or containers "web", "sidecar" must set securityContext.seccompProfile.type to "RuntimeDefault" or "Localhost")
```

### Gatekeeper / Kyverno specifics

Gatekeeper:

```text
admission webhook "validation.gatekeeper.sh" denied the request: 
[k8sallowedrepos] container <name> has an invalid image repo <foo>, allowed repos are ["registry.example.com"]
```

Kyverno:

```text
admission webhook "validate.kyverno.svc-fail" denied the request: 
resource Pod/myns/web-abc-123 was blocked due to the following policies:
  require-labels:
    require-team-label: 'validation error: The label `team` is required. Rule require-team-label failed at path /metadata/labels/team/'
```

### LimitRange / ResourceQuota

```text
Error from server (Forbidden): error when creating "deploy.yaml": pods "web-abc-123" is forbidden: exceeded quota: compute-resources, requested: requests.memory=512Mi, used: requests.memory=4Gi, limited: requests.memory=4Gi
```

Cause: namespace ResourceQuota maxes out. Either reduce the request or raise the quota.

```text
Error from server (Forbidden): pods "web-abc-123" is forbidden: minimum cpu usage per Container is 100m, but request is 50m
```

Cause: LimitRange in the namespace enforces a minimum that this container undershoots.

```text
Error from server (Forbidden): pods "web-abc-123" is forbidden: maximum memory usage per Container is 1Gi, but limit is 2Gi
```

Cause: LimitRange max exceeded.

```text
Error from server (Forbidden): the request is invalid: : LimitRange "default-limits" has no default request, must explicitly specify
```

Cause: namespace requires explicit requests/limits and pod doesn't set them.

## ResourceQuota / LimitRange

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: compute-resources
  namespace: myapp
spec:
  hard:
    requests.cpu: "4"
    requests.memory: 4Gi
    limits.cpu: "8"
    limits.memory: 8Gi
    persistentvolumeclaims: "10"
    pods: "20"
```

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
  namespace: myapp
spec:
  limits:
    - type: Container
      default:
        cpu: 500m
        memory: 512Mi
      defaultRequest:
        cpu: 100m
        memory: 128Mi
      max:
        cpu: "2"
        memory: 4Gi
      min:
        cpu: 50m
        memory: 64Mi
```

The "set both requests AND limits or get unexpected eviction" rule:

```text
- requests only → BestEffort or Burstable depending on if any container has limits
- limits only   → request defaults to limit (Guaranteed)
- both equal    → Guaranteed (best protection from eviction)
- neither       → BestEffort (first to be evicted, no resource reservation)
```

Inspect:

```bash
kubectl describe quota -n myapp
# Shows used vs hard for each constraint

kubectl describe limitrange -n myapp
```

## Ingress / IngressController Errors

### No endpoints via Ingress

```text
HTTP/1.1 503 Service Unavailable
no endpoints available for service "web-svc"
```

Cause: same as Service-has-no-endpoints — the backend Service selector matches no Ready pods. Surfaces through the Ingress controller (nginx, traefik, etc.).

### Host header mismatch

```text
HTTP/1.1 404 Not Found
default backend - 404
```

Cause: request `Host` header doesn't match any Ingress rule's `host:`. NGINX Ingress falls back to the default backend (which 404s by default). Check what host the client sent:

```bash
curl -v -H 'Host: app.example.com' http://<ingress-lb-ip>/
```

### TLS secret problems

```text
Error: could not load TLS certificate from secret myns/tls-cert: secret tls-cert in namespace myns is not in tls format
```

Cause: Secret type isn't `kubernetes.io/tls`, or `tls.crt` / `tls.key` keys are missing/wrong format.

```bash
kubectl create secret tls tls-cert \
  --cert=fullchain.pem \
  --key=privkey.pem \
  -n myns

kubectl get secret tls-cert -n myns -o yaml
# type: kubernetes.io/tls
# data:
#   tls.crt: <base64>
#   tls.key: <base64>
```

### IngressClass mismatch

```yaml
# Modern (1.18+): use ingressClassName
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web
spec:
  ingressClassName: nginx       # must match an IngressClass resource
  rules: ...
```

```bash
kubectl get ingressclasses
# NAME    CONTROLLER             PARAMETERS   AGE
# nginx   k8s.io/ingress-nginx   <none>       1h
```

If `ingressClassName` doesn't match, **no controller picks up the Ingress** — looks like nothing is happening, no events, just nothing routed.

The annotation form is legacy:

```yaml
metadata:
  annotations:
    kubernetes.io/ingress.class: nginx     # deprecated but still honored
```

### Ingress vs Gateway API

Gateway API (gateway.networking.k8s.io) is the successor. Different objects:

```text
Ingress       → IngressClass + Ingress
Gateway API   → GatewayClass + Gateway + HTTPRoute / TCPRoute / etc.
```

Different errors, but conceptually similar (no listener for host, TLS secret missing, etc.). Check `kubectl describe gateway` and `kubectl describe httproute` for status conditions.

## CRD / Operator Errors

```text
error: the server doesn't have a resource type "myresource"
```

Cause: CRD not installed (or CRD installed but in a different API group/version than expected).

```text
error: no matches for kind "MyResource" in version "my.example.com/v1alpha1"
```

Cause: same. Check installed CRDs:

```bash
kubectl get crd
kubectl get crd | grep example.com
kubectl api-resources | grep myresource
kubectl explain myresource     # if CRD installed and has openAPISchema
```

Version skew:

```text
error: no matches for kind "MyResource" in version "my.example.com/v1"
# but you have v1alpha1 installed
```

Fix the manifest's `apiVersion`, or upgrade the CRD.

```text
error: couldn't get current API resources: GroupVersion "my.example.com/v1" not found in unstructured object
```

Cause: usually a transient discovery issue, sometimes a CRD without proper conversion webhook for the requested version.

### Conversion webhook errors

```text
Internal error occurred: failed to call webhook: Post "https://my-operator-webhook.my-operator-ns.svc:443/convert?timeout=30s": context deadline exceeded
```

Cause: CRD conversion webhook (used to convert between v1alpha1 and v1beta1, etc.) is unreachable. Check the operator pod is running and its Service has endpoints.

## Storage Errors

```text
no persistent volumes available for this claim and no storage class is set
```

Cause: PVC has no `storageClassName` and the cluster has no default. Either set a default StorageClass, or set `storageClassName` explicitly.

```text
PersistentVolumeClaim "data" not found
```

Cause: pod references a PVC that doesn't exist in the namespace. Verify:

```bash
kubectl get pvc -n <ns>
kubectl get pod <name> -o yaml | grep -A2 persistentVolumeClaim
```

```text
PVC data has no storage class
```

Cause: PVC was created without `storageClassName` and there's no cluster default.

```text
exceeded MaxVolumeCount (default of 39 for AWS EBS)
```

Cause: hit the per-node attachment limit for a CSI driver. AWS EBS limits volumes per instance based on instance type (e.g., 39 for t3.small, more for larger). The pod schedules on a node that already has the max attached.

```bash
# How many PVs attached on each node
kubectl get volumeattachment | awk '{print $4}' | sort | uniq -c
```

Workarounds:

```text
- Spread pods across nodes (anti-affinity, topologySpreadConstraints)
- Use larger instance types (more attachable volumes)
- Use a CSI driver with higher limits (e.g., NVMe instance store)
- Consolidate volumes (one big PV instead of many small)
```

PV reclaim policies:

```text
Retain   — PV remains after PVC deletion; manual cleanup needed (data preserved)
Delete   — PV (and underlying storage) deleted with PVC (default for dynamic provisioning)
Recycle  — DEPRECATED; basic scrub then re-bind
```

A PV stuck in `Released` state (PVC deleted, but Retain policy) needs manual reset:

```bash
kubectl edit pv <name>
# Remove the entire spec.claimRef section
# PV transitions to Available
```

Cross-link: see FailedMount Events above for `FailedAttachVolume`, Multi-Attach, mount failures.

## Job / CronJob Errors

```text
Job has reached the specified backoff limit
```

```text
Warning  BackoffLimitExceeded  Job has reached the specified backoff limit
```

Cause: `spec.backoffLimit` retries exhausted (default 6). The Job is marked Failed and won't retry.

```yaml
apiVersion: batch/v1
kind: Job
spec:
  backoffLimit: 4
  activeDeadlineSeconds: 600   # hard wall-clock deadline
  template:
    spec:
      restartPolicy: OnFailure   # or Never (Never always counts crashes against backoffLimit)
      containers: ...
```

`restartPolicy: OnFailure` retries the container in-place, only counting full restarts toward `backoffLimit`. `Never` recreates pods, each crash counts.

```text
Job has been suspended
```

Cause: `spec.suspend: true` was set (1.21+). Useful for pausing batch work.

```text
Warning  TooManyMissedRunsTimePassed  CronJob X missed too many recent schedules and is now ineligible to run
```

Cause: CronJob's `startingDeadlineSeconds` window has lapsed for missed runs (controller couldn't run them, e.g., apiserver was down). Once >100 misses elapsed, it gives up and marks the CronJob non-runnable until you restart it.

```yaml
apiVersion: batch/v1
kind: CronJob
spec:
  schedule: "*/5 * * * *"
  startingDeadlineSeconds: 200      # don't start runs more than 200s late
  concurrencyPolicy: Forbid          # never overlap; alternatives: Allow, Replace
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
```

ImagePullBackOff inside Jobs:

```text
Status: Pending
Containers:
  worker:
    State: Waiting   Reason: ImagePullBackOff
```

Treated like any pod-level pull failure, but consumes a backoff retry per attempt. If the image will never become pullable (deleted tag), the Job will exhaust `backoffLimit` and fail rather than wait forever.

## Networking Diagnostics — kubectl tools

### exec into a pod and curl

```bash
# Service via DNS (in-cluster)
kubectl exec -it <pod> -- curl -fsS http://myservice.myns.svc.cluster.local:8080/healthz

# Direct pod IP
kubectl get pod <other-pod> -o jsonpath='{.status.podIP}'
kubectl exec -it <pod> -- curl -fsS http://10.244.1.5:8080/healthz

# ClusterIP
kubectl get svc myservice -o jsonpath='{.spec.clusterIP}'
kubectl exec -it <pod> -- curl http://10.96.10.20:8080/
```

If the target pod has no shell, exec into a sidecar or use ephemeral debug.

### netshoot debug pod

```bash
kubectl run debug --rm -it --image=nicolaka/netshoot:latest -- bash

# Inside netshoot:
nslookup myservice                                  # DNS
dig +short myservice.myns.svc.cluster.local         # full resolution
curl -v http://myservice:8080/                      # HTTP
nc -zv myservice 8080                               # TCP probe
ping <pod-ip>                                       # ICMP (if NetworkPolicy allows)
mtr myservice                                       # path
ss -tnp                                             # socket state
tcpdump -i eth0 -n port 80                         # packet capture
iptables-save | head -100                          # NAT rules (root)
```

Pin debug pod to a specific node:

```bash
kubectl run debug --rm -it --image=nicolaka/netshoot:latest \
  --overrides='{"spec":{"nodeName":"node-1"}}' -- bash
```

### kubectl debug node

```bash
kubectl debug node/node-1 -it --image=ubuntu
# Drops you into a privileged pod with hostPID, hostNetwork, root volume mounted at /host
chroot /host
# Now you have a shell on the node
```

For node-level diagnostics: kubelet logs, /var/log, runtime sockets, kernel parameters.

```bash
chroot /host journalctl -u kubelet --since "5 minutes ago"
chroot /host crictl ps
chroot /host crictl logs <container-id>
```

### kube-system pods

```bash
kubectl get pods -n kube-system
# NAME                                READY  STATUS    AGE
# coredns-...                         1/1    Running   5d
# coredns-...                         1/1    Running   5d
# etcd-control-1                      1/1    Running   30d
# kube-apiserver-control-1            1/1    Running   30d
# kube-controller-manager-control-1   1/1    Running   30d
# kube-scheduler-control-1            1/1    Running   30d
# kube-proxy-...                      1/1    Running   30d   (DaemonSet, one per node)
# <cni-plugin>-...                    1/1    Running   30d   (calico-node, cilium, weave)
```

Where they live:

```text
control plane components → static pods on control nodes (not visible as Deployments)
                           manifests in /etc/kubernetes/manifests/ on control nodes
kube-proxy               → DaemonSet on every node
CoreDNS                  → Deployment in kube-system, 2+ replicas
CNI                      → DaemonSet (varies: calico-node, cilium, etc.)
```

For managed clusters (EKS, GKE, AKS), the control plane is hidden — you'll only see kube-proxy, CoreDNS, and CNI.

### Cluster-wide events

```bash
kubectl get events -A --sort-by='.lastTimestamp' | tail -50
kubectl get events -A --field-selector type=Warning --sort-by='.lastTimestamp'
```

Watch for repeating reasons; they fingerprint cluster-wide problems (e.g., FailedScheduling appearing across many namespaces → scheduler down or cluster full).

## describe Pod Walkthrough

```bash
kubectl describe pod <name>
```

The output sections, top to bottom:

### Metadata header

```text
Name:         web-abc-123
Namespace:    myapp
Priority:     0
Node:         node-1/10.0.0.10
Start Time:   Sat, 25 Apr 2026 12:00:00 +0000
Labels:       app=web,pod-template-hash=abc
Annotations:  ...
Status:       Running
IP:           10.244.1.5
IPs:
  IP:           10.244.1.5
Controlled By:  ReplicaSet/web-abc
```

`Controlled By` tells you who manages this pod. Don't delete a pod expecting it to stay gone — its parent (Deployment / StatefulSet / DaemonSet / Job) will recreate it.

### Containers section

```text
Containers:
  web:
    Container ID:   containerd://abc123def456...
    Image:          myorg/web:1.2.3
    Image ID:       docker.io/myorg/web@sha256:def456...
    Port:           8080/TCP
    Host Port:      0/TCP
    State:          Running
      Started:      Sat, 25 Apr 2026 12:00:05 +0000
    Last State:     Terminated
      Reason:       OOMKilled
      Exit Code:    137
      Started:      Sat, 25 Apr 2026 11:55:00 +0000
      Finished:     Sat, 25 Apr 2026 11:59:58 +0000
    Ready:          True
    Restart Count:  3
    Limits:
      cpu:     500m
      memory:  512Mi
    Requests:
      cpu:        100m
      memory:     128Mi
    Liveness:     http-get http://:8080/healthz delay=10s timeout=1s period=10s #success=1 #failure=3
    Readiness:    http-get http://:8080/ready delay=5s timeout=1s period=5s #success=1 #failure=2
    Environment:
      DB_HOST:      postgres.myapp.svc.cluster.local
      DB_PASSWORD:  <set to the key 'DB_PASSWORD' in secret 'db-creds'>  Optional: false
    Mounts:
      /etc/config from app-config (ro)
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-xyz (ro)
```

Critical fields:

```text
State / Last State + Reason + ExitCode → what happened
Restart Count                          → how often
Image ID                               → the digest actually pulled (immutable proof)
Limits / Requests                      → why eviction or OOM happened
Liveness / Readiness                   → probe config
Mounts                                 → volume sources
```

### Conditions

```text
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
```

```text
Initialized      → all init containers completed
PodScheduled     → assigned to a node
ContainersReady  → all containers Ready
Ready            → pod is Ready (passes readinessGates if any)
```

If `Ready: False`, look at the underlying conditions to find what's missing.

### Volumes section

```text
Volumes:
  app-config:
    Type:      ConfigMap (a volume populated by a ConfigMap)
    Name:      app-config
    Optional:  false
  data:
    Type:        PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)
    ClaimName:   web-data
    ReadOnly:    false
  kube-api-access-xyz:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    DownwardAPI:             true
```

Maps logical mount points to concrete sources.

### Events section

```text
Events:
  Type     Reason     Age    From               Message
  ----     ------     ----   ----               -------
  Normal   Scheduled  5m     default-scheduler  Successfully assigned myapp/web-abc-123 to node-1
  Normal   Pulling    5m     kubelet            Pulling image "myorg/web:1.2.3"
  Normal   Pulled     4m     kubelet            Successfully pulled image "myorg/web:1.2.3" in 1.2s
  Normal   Created    4m     kubelet            Created container web
  Normal   Started    4m     kubelet            Started container web
  Warning  Unhealthy  2m     kubelet            Liveness probe failed: HTTP probe failed with statuscode: 503
  Normal   Killing    2m     kubelet            Container web failed liveness probe, will be restarted
```

Events are kept ~1 hour by default. If `Events: <none>` and you suspect history, check `kubectl get events -A --field-selector involvedObject.name=<pod>` (might still be there with broader filter).

## describe Node Walkthrough

```bash
kubectl describe node <name>
```

### Header / Labels / Taints

```text
Name:               node-1
Roles:              <none>
Labels:             beta.kubernetes.io/arch=amd64
                    beta.kubernetes.io/os=linux
                    kubernetes.io/hostname=node-1
                    topology.kubernetes.io/region=us-east-1
                    topology.kubernetes.io/zone=us-east-1a
Taints:             dedicated=gpu:NoSchedule
Unschedulable:      false
```

### Conditions

```text
Conditions:
  Type             Status  LastHeartbeatTime   Reason                Message
  ----             ------  -----------------   ------                -------
  MemoryPressure   False   ...                 KubeletHasSufficientMemory   kubelet has sufficient memory available
  DiskPressure     False   ...                 KubeletHasNoDiskPressure     kubelet has no disk pressure
  PIDPressure      False   ...                 KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready            True    ...                 KubeletReady                 kubelet is posting ready status
```

If any pressure condition is `True`, the kubelet is evicting pods. If `Ready: False`, the node is offline / kubelet not reporting.

```text
Ready: Unknown   → kubelet hasn't reported in node-monitor-grace-period (40s)
Ready: False     → kubelet reports it's not ready (something specific is wrong)
```

### Capacity / Allocatable

```text
Capacity:
  cpu:                4
  ephemeral-storage:  100Gi
  memory:             16Gi
  pods:               110
Allocatable:
  cpu:                3800m       # capacity - kube-reserved - system-reserved
  ephemeral-storage:  90Gi
  memory:             15Gi
  pods:               110
```

`Allocatable` is what the scheduler sees. Difference = system + kubelet overhead.

### Allocated resources

```text
Allocated resources:
  Resource          Requests     Limits
  cpu               2200m (57%)  4500m (118%)
  memory            6Gi (40%)    9Gi (60%)
  ephemeral-storage 0 (0%)       0 (0%)
```

The percentages are vs Allocatable. Sums of requests > 100% is **impossible** (scheduler enforces). Sums of limits > 100% is fine and common (overcommit).

### Non-terminated Pods

```text
Non-terminated Pods:           (15 in total)
  Namespace          Name                 CPU Requests  CPU Limits  Memory Requests  Memory Limits  Age
  ---------          ----                 ------------  ----------  ---------------  -------------  ---
  default            web-abc-123          100m (2%)     500m (13%)  128Mi (0%)       512Mi (3%)     5m
  ...
```

Scan for outliers consuming most of the node.

### Events

```text
Events:
  Type     Reason                   Age    From          Message
  Normal   Starting                 30d    kube-proxy    Starting kube-proxy
  Warning  Rebooted                 5m     kubelet       Node node-1 has been rebooted, boot id: ...
  Normal   NodeReady                3m     kubelet       Node node-1 status is now: NodeReady
```

Node-level events (reboots, kubelet restarts, evictions) live here.

## logs Patterns

```bash
# Current container output (pod with one container)
kubectl logs <pod>

# Multi-container pod, must specify container
kubectl logs <pod> -c <container>
kubectl logs <pod> --all-containers           # interleaved, useful for sidecar correlation

# Previous container instance (after restart/crash)
kubectl logs <pod> --previous
kubectl logs <pod> -c <container> --previous

# Stream live
kubectl logs -f <pod>

# Last N lines
kubectl logs --tail=100 <pod>

# Since timestamp / duration
kubectl logs --since=10m <pod>
kubectl logs --since-time=2026-04-25T12:00:00Z <pod>

# Across all pods of a label selector
kubectl logs -l app=web --all-containers --tail=50

# All pods in a Deployment
kubectl logs deployment/web --all-containers --tail=100

# Specific replica
kubectl logs -l app=web --max-log-requests=10 --all-containers
```

### kubelet log rotation

The kubelet rotates container logs at `/var/log/pods/<ns>_<pod>_<uid>/<container>/`. Default rotation size is 10MB per container, 5 rotations kept (configurable via kubelet flags `--container-log-max-size`, `--container-log-max-files`).

`kubectl logs` only sees the **current** rotated file. To get older logs, you need:

```bash
# On the node directly
ls /var/log/pods/<ns>_<pod>_<uid>/<container>/
# 0.log  0.log.20260425-100000.gz  ...

# Or send logs to an aggregator (Loki, ELK, CloudWatch)
```

### Multi-pod log streaming

`stern`:

```bash
stern --selector app=web --tail 50 --since 10m
stern --selector app=web --container web --tail 50
stern <pod-prefix>                              # auto-globs
stern --output raw --selector app=web           # plain output, no headers
```

`kail`:

```bash
kail --label app=web
kail --ns myapp --label app=web
```

Both colorize output per pod and stream concurrently. Indispensable when debugging a multi-replica problem.

## Cluster-Level Issues

### kube-apiserver unhealthy

Symptoms:

```text
- kubectl: "the server doesn't have a resource type"  (random failures)
- kubectl: "Unable to connect to the server: dial tcp ...: connection refused"
- Webhook calls timing out  (apiserver overloaded)
- New pods Pending forever (controller-manager can't read state)
```

Diagnostic on a control node:

```bash
sudo crictl ps | grep apiserver
sudo crictl logs <apiserver-id> | tail -100
# Or static pod logs:
ls /var/log/pods/kube-system_kube-apiserver-*/

# Health endpoint
curl -k https://localhost:6443/healthz
curl -k https://localhost:6443/readyz?verbose
```

### etcd unhealthy

Symptoms: writes fail, leader election storms.

```bash
# On a control node
sudo etcdctl --endpoints=https://127.0.0.1:2379 \
  --cacert=/etc/kubernetes/pki/etcd/ca.crt \
  --cert=/etc/kubernetes/pki/etcd/peer.crt \
  --key=/etc/kubernetes/pki/etcd/peer.key \
  endpoint health

sudo etcdctl ... endpoint status -w table
sudo etcdctl ... member list -w table
```

Out-of-space:

```text
mvcc: database space exceeded
```

Defrag and adjust quota:

```bash
sudo etcdctl ... defrag --cluster
sudo etcdctl ... compact $(etcdctl ... endpoint status --write-out json | jq -r '.[0].Status.header.revision')
```

### kube-scheduler down

Symptom: every new pod is Pending forever, no `FailedScheduling` events (because scheduler isn't even trying).

```bash
kubectl get pods -n kube-system -l component=kube-scheduler
kubectl logs -n kube-system kube-scheduler-<node>
```

### kube-controller-manager down

Symptoms:

```text
- ReplicaSet count not honored (delete a pod, it's not recreated)
- Garbage collection paused (orphaned pods, stuck Terminating)
- ServiceAccount tokens not auto-created
- Endpoints not updated when pods change
```

```bash
kubectl get pods -n kube-system -l component=kube-controller-manager
kubectl logs -n kube-system kube-controller-manager-<node>
```

### kubelet not running

Symptom: node `NotReady`, pods on that node go into the 5-minute `pod-eviction-timeout` countdown.

```bash
ssh <node>
sudo systemctl status kubelet
sudo journalctl -u kubelet -n 200 --no-pager
sudo systemctl restart kubelet
```

### CoreDNS in CrashLoopBackOff

Cluster-wide DNS failure. **Every pod's DNS lookup fails.** Symptoms:

```text
- Service-to-service calls fail with "no such host"
- ImagePullBackOff (registry hostname doesn't resolve from pods)  -- actually node-side, but...
- kubectl logs / exec etc. still works (uses kubeconfig server, not in-cluster DNS)
```

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns
kubectl logs -n kube-system <coredns-pod>

# Common reason: bad upstream forwarder in the Corefile
kubectl get cm coredns -n kube-system -o yaml
# Look for `forward . <upstream>`
```

A famously broken Corefile:

```text
forward . /etc/resolv.conf
```

If `/etc/resolv.conf` on the CoreDNS pod's node points to `127.0.0.53` (systemd-resolved stub) and CoreDNS can't reach it from inside the pod, you get a loop. Fix: use explicit upstream like `forward . 1.1.1.1 8.8.8.8`.

## Common Gotchas

### 1. Liveness probe too aggressive (broken)

```yaml
livenessProbe:
  httpGet: { path: /healthz, port: 8080 }
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 2
```

App takes 30s to start. At T=15s the probe fails twice → pod killed. Restart loop never lets the app actually start. CrashLoopBackOff with no app errors in logs.

### Fixed: add a startup probe

```yaml
startupProbe:
  httpGet: { path: /healthz, port: 8080 }
  failureThreshold: 30
  periodSeconds: 10
livenessProbe:
  httpGet: { path: /healthz, port: 8080 }
  periodSeconds: 10
  failureThreshold: 3
```

### 2. Resource limits without requests (broken)

```yaml
resources:
  limits:
    cpu: "1"
    memory: 1Gi
```

When `requests` is unset, it defaults to `limits` → pod is `Guaranteed`, but the scheduler reserves 1 CPU + 1 GB on every node it considers. Often unschedulable on small nodes; on larger nodes you've over-reserved.

### Fixed: set both, with requests lower than limits

```yaml
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 1Gi
```

### 3. `:latest` tag with default ImagePullPolicy (broken)

```yaml
image: myorg/app:latest
# imagePullPolicy default = Always for :latest, IfNotPresent for explicit tags
```

Default policy for `:latest` is `Always`, but every other tag is `IfNotPresent`. Push a new image with the same explicit tag (e.g., overwriting `:dev`) → kubelet keeps using the cached old image.

### Fixed: use immutable tags + IfNotPresent, or imagePullPolicy: Always

```yaml
image: myorg/app:1.2.3       # version-bump on every change
imagePullPolicy: IfNotPresent
```

Or for mutable tags:

```yaml
image: myorg/app:dev
imagePullPolicy: Always
```

### 4. kubectl apply -f without -k (broken)

```bash
kubectl apply -f .
# Reads YAML files but ignores kustomization.yaml
```

Kustomize layering, patches, and replacements are all skipped — you applied raw bases.

### Fixed: use -k

```bash
kubectl apply -k .
# or:
kubectl kustomize . | kubectl apply -f -
```

### 5. Service selector typo (broken)

```yaml
# Pod
labels: { app: web, tier: frontend }
# Service
spec:
  selector: { app: web, teir: frontend }     # 'teir' typo
```

Service has `<none>` endpoints. No error, just no traffic flowing.

### Fixed: verify selector matches actual pod labels

```bash
kubectl get pods -l app=web,tier=frontend
kubectl describe svc web | grep Endpoints
```

### 6. readinessProbe identical to livenessProbe (broken)

```yaml
readinessProbe: { httpGet: { path: /, port: 8080 } }
livenessProbe:  { httpGet: { path: /, port: 8080 } }
```

When the app has a transient issue (DB slow), readiness fails (removed from Service — fine), but liveness also fails (pod restarted — bad). Restart kills in-flight work, doesn't fix anything.

### Fixed: keep them distinct, with liveness much looser

```yaml
readinessProbe:
  httpGet: { path: /ready, port: 8080 }       # checks dependencies
  failureThreshold: 2
livenessProbe:
  httpGet: { path: /live, port: 8080 }        # checks process responsiveness only
  failureThreshold: 6                          # very tolerant
  periodSeconds: 30
```

### 7. hostPath without securityContext (broken)

```yaml
volumes:
  - name: data
    hostPath: { path: /var/data }
```

Pod runs as root, mounts host path, can stomp on host files. PodSecurity `restricted` blocks this; without admission policy, you have a privilege escalation risk.

### Fixed: use a PV with proper access controls + non-root pod

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: data-pvc
```

### 8. SecurityContext UID mismatch with image USER (broken)

```dockerfile
# Dockerfile
USER appuser           # name, not number
```

```yaml
spec:
  securityContext:
    runAsNonRoot: true
```

CreateContainerError: "container has runAsNonRoot and image has non-numeric user (appuser), cannot verify user is non-root".

### Fixed: use numeric UID

```dockerfile
RUN useradd -u 1000 appuser
USER 1000
```

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
```

### 9. imagePullSecret in different namespace (broken)

```bash
kubectl create secret docker-registry regcred ... -n default
# Pod is in 'myapp' namespace, references regcred → not found
```

### Fixed: secret must live in pod's namespace

```bash
kubectl create secret docker-registry regcred ... -n myapp
# Or attach to default SA in myapp namespace
```

### 10. Forgetting --record on apply (broken)

```bash
kubectl apply -f deploy.yaml
# Rollout history shows: 'change-cause: <none>'
kubectl rollout history deployment/web
# REVISION  CHANGE-CAUSE
# 1         <none>
# 2         <none>
```

### Fixed: record (deprecated in newer versions but still works) or annotate manually

```bash
kubectl apply -f deploy.yaml --record=true
# Or, modern approach:
kubectl annotate deployment web kubernetes.io/change-cause="Update to v1.2.3" --overwrite
```

### 11. NodePort vs LoadBalancer confusion (broken)

```yaml
spec:
  type: NodePort
  ports: [{ port: 80, targetPort: 8080, nodePort: 30080 }]
```

User browses `<lb-dns>:80` expecting connection. NodePort exposes only on `<any-node-ip>:30080`. No managed LB created.

### Fixed: type LoadBalancer for managed cloud LB

```yaml
spec:
  type: LoadBalancer
  ports: [{ port: 80, targetPort: 8080 }]
```

### 12. PVC accessMode RWX on RWO storage (broken)

```yaml
spec:
  accessModes: [ReadWriteMany]
  storageClassName: gp3            # AWS EBS, RWO only
```

PVC stuck Pending forever. No clear event saying "access mode wrong" — just no provisioning.

### Fixed: match accessMode to storage class capability

```yaml
spec:
  accessModes: [ReadWriteOnce]
  storageClassName: gp3
# Or, for RWX:
spec:
  accessModes: [ReadWriteMany]
  storageClassName: efs-sc         # AWS EFS supports RWX
```

### 13. Implicit default ServiceAccount RBAC (broken)

```yaml
spec:
  # No serviceAccountName specified → uses 'default' SA in the namespace
  containers:
    - name: app
      image: myorg/operator:1.0
      # operator code calls kubectl/client-go to list pods → 403 Forbidden
```

The `default` SA has no RBAC by default.

### Fixed: explicit SA + Role + RoleBinding

```yaml
# Or use the cluster-admin SA explicitly (not advised in prod)
spec:
  serviceAccountName: my-app-sa
```

## Recovery Patterns

### Stuck Terminating pod

```bash
# Try graceful first; sometimes the kubelet just needs a kick
kubectl delete pod <name>

# Force-delete (skips graceful shutdown, removes API object immediately)
kubectl delete pod <name> --grace-period=0 --force
```

If the pod has finalizers (rare), edit them out:

```bash
kubectl patch pod <name> -p '{"metadata":{"finalizers":null}}' --type=merge
```

### Stuck Terminating namespace

```bash
kubectl get namespace <name> -o json > ns.json
# Edit ns.json, remove "kubernetes" from spec.finalizers (and any others)
# Then PUT it back via the /finalize subresource:

kubectl proxy &
curl -k -H "Content-Type: application/json" -X PUT \
  --data-binary @ns.json \
  http://127.0.0.1:8001/api/v1/namespaces/<name>/finalize
```

The namespace deletion is blocked by stuck child resources (CRD finalizers, missing webhooks). Forcing is a last resort — fix the underlying resource if possible.

### Stuck PV in Released state

```bash
kubectl get pv | grep Released
kubectl edit pv <name>
# Delete the entire spec.claimRef block
# Save → PV → Available
```

### Pod stuck Pending on missing PV

```bash
kubectl get pvc -n <ns>
# STATUS: Pending  → PV not bound yet

kubectl describe pvc <pvc> -n <ns>
# Events at bottom show provisioning attempts
```

If `volumeBindingMode: WaitForFirstConsumer` and pod is also Pending due to resource issues → chicken-and-egg, fix scheduling first.

### Force pod recreation on a Deployment

```bash
# Modern approach (Kubernetes 1.15+)
kubectl rollout restart deployment/<name>
# Triggers rolling restart by patching the pod template's annotations

# Old-school: bump a label/annotation
kubectl patch deployment <name> -p \
  '{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"}}}}}'

# Or scale down+up (causes brief outage)
kubectl scale deployment <name> --replicas=0
kubectl scale deployment <name> --replicas=3
```

### Force-recreate via pod delete (ReplicaSet pattern)

```bash
# A pod managed by a ReplicaSet (Deployment) will be recreated automatically:
kubectl delete pod <name>
# RS controller sees missing pod, creates a replacement
```

### Stuck CRD deletion

If you `kubectl delete crd <name>` and it hangs because instances exist, list and force-delete the instances:

```bash
kubectl get <kind> -A
# Delete each, removing finalizers if needed:
kubectl patch <kind> <name> -n <ns> --type merge -p '{"metadata":{"finalizers":null}}'
kubectl delete <kind> <name> -n <ns>
```

### Recover a node from NotReady

```bash
ssh <node>
sudo systemctl status kubelet
sudo journalctl -u kubelet -n 100 --no-pager

# Common: container runtime issue
sudo systemctl status containerd
sudo crictl version

# Restart everything in order
sudo systemctl restart containerd
sudo systemctl restart kubelet
```

### Drain and replace a node

```bash
kubectl cordon <node>
kubectl drain <node> --ignore-daemonsets --delete-emptydir-data
# Pods reschedule to other nodes
# Now safe to terminate / replace

kubectl uncordon <node>     # if returning to service
# Or:
kubectl delete node <node>  # remove from cluster (after termination)
```

## Idioms

```text
- "describe pod first, then events, then logs"
   → describe surfaces the right keywords (Reason, ExitCode, FailedScheduling)
   → then targeted log/event commands

- "always check the previous container's logs after CrashLoopBackOff"
   → kubectl logs <pod> --previous
   → otherwise you see a still-starting (empty) container

- "kubectl get events --sort-by=.lastTimestamp -A"
   → cluster-wide recent activity, ordered chronologically

- "use stern for multi-pod log streaming"
   → kubectl logs is single-pod; stern handles selector + all replicas at once

- "use k9s for interactive cluster browsing"
   → terminal UI, fast switching between contexts/namespaces/resources

- "scope kubectl commands with --namespace or set a default"
   → kubectl config set-context --current --namespace=myapp
   → avoids 'pods not found' due to default namespace assumption

- "trust the Reason field over the Status column"
   → 'Status' is heuristic; 'Reason' (in describe / -o yaml) is authoritative

- "the scheduler is request-based; the OOM killer is limit-based"
   → request governs whether a pod fits; limit governs whether it gets killed

- "start every pull failure check with: does the secret exist in this namespace"
   → 80% of ImagePullBackOff in private registries

- "check Endpoints, not Service, when traffic doesn't flow"
   → Service is the spec; Endpoints is the runtime truth

- "Pending without an event = scheduler hasn't picked it up; with FailedScheduling = scheduler tried"
   → if no event in 30s, suspect scheduler down or admission rejection silenced

- "for a multi-container pod, never omit -c"
   → kubectl logs / exec without -c may pick the wrong container silently

- "kubectl debug is your scalpel"
   → debug pod (--copy-to + ephemeral container) preserves the original
   → debug node/<n> for host-level access
```

## See Also

- kubectl
- kubectl-debug
- container-hardening
- helm
- troubleshooting/docker-errors
- troubleshooting/linux-errors
- troubleshooting/dns-errors

## References

- kubernetes.io/docs/tasks/debug
- kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle
- kubernetes.io/docs/reference/kubectl/cheatsheet
- kubernetes.io/docs/concepts/security
- kubernetes.io/docs/concepts/configuration/manage-resources-containers
- kubernetes.io/docs/concepts/scheduling-eviction
- kubernetes.io/docs/reference/access-authn-authz/rbac
- kubernetes.io/docs/concepts/storage/persistent-volumes
- kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes
- kubernetes.io/docs/concepts/cluster-administration/networking
- kubernetes.io/docs/concepts/security/pod-security-standards
- kubernetes.io/docs/reference/access-authn-authz/admission-controllers
- kubernetes.io/docs/tasks/debug/debug-cluster
- github.com/kubernetes/kubernetes/blob/master/CHANGELOG
- github.com/wercker/stern
- github.com/derailed/k9s
- github.com/nicolaka/netshoot
