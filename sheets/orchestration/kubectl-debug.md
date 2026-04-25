# kubectl Debug (Kubernetes Troubleshooting Recipes)

A recipe-book sheet for diagnosing broken Pods, Deployments, Services, and Nodes with kubectl — every command paste-runnable, every diagnostic with what-to-look-at and what-it-means.

## Setup

This is a recipe-book sheet. We assume:

- `kubectl` is installed and on your `PATH` (v1.18+ for `kubectl debug`; v1.25+ for ephemeral containers as stable).
- A working `~/.kube/config` (or `KUBECONFIG`) with credentials for the cluster.
- You can already run `kubectl get pods` against the right cluster and namespace.
- For node debugging recipes you need permission to create privileged pods (`debug.privileged: true` if PSA/PSS gates).
- The cluster has metrics-server installed if you want `kubectl top` to work.

Quick sanity check before any incident:

```bash
kubectl version --short
kubectl cluster-info
kubectl config current-context
kubectl config get-contexts
kubectl get nodes
kubectl get ns
```

The canonical "first thing in incident" workflow:

```bash
kubectl config current-context                      # am I on the right cluster?
kubectl get pods -A | grep -vE 'Running|Completed'  # what is unhealthy?
kubectl get events -A --sort-by=.lastTimestamp | tail -50
kubectl get nodes                                   # any node NotReady?
```

If `kubectl debug` is not found:

```bash
kubectl debug --help                                # confirm subcommand exists
kubectl version --short | grep -i client            # need >= v1.18
```

If you need ephemeral containers and the API rejects them, the feature gate `EphemeralContainers=true` was beta in v1.16–v1.22 and stable from v1.25. Most managed clouds (EKS, GKE, AKS) have it on by default in supported versions.

Useful environment knobs:

```bash
export KUBECONFIG=~/.kube/config:~/.kube/prod.yaml  # multi-cluster
export KUBECTL_EDITOR=vim                            # for kubectl edit
alias k=kubectl
source <(kubectl completion bash)                    # or zsh
complete -F __start_kubectl k
```

## The Diagnostic Ladder

The canonical "what to check first" order, in order. Every Kubernetes incident triage starts here.

```bash
kubectl get pod <pod> -n <ns>                  # 1. current high-level state
kubectl describe pod <pod> -n <ns>             # 2. events + conditions + container statuses
kubectl logs <pod> -n <ns> -c <container>      # 3. application output
kubectl logs <pod> -n <ns> --previous          # 3b. previous instance if it crashed
kubectl get events -n <ns> --sort-by=.lastTimestamp  # 4. cluster-side reasons
kubectl debug <pod> -n <ns> -it --image=busybox --target=<container>  # 5. interactive
```

The "narrow scope from cluster -> namespace -> workload -> pod -> container" funnel:

```bash
kubectl get nodes                                          # cluster-wide
kubectl get pods -A | grep -vE 'Running|Completed'         # cluster -> any unhealthy
kubectl get pods -n <ns>                                   # namespace
kubectl get deploy,sts,ds -n <ns>                          # workloads in namespace
kubectl get pods -n <ns> -l app=<name>                     # workload -> pods
kubectl logs -n <ns> -l app=<name> --all-containers --tail=200 --prefix
kubectl describe pod -n <ns> <pod>                          # pod
kubectl logs -n <ns> <pod> -c <container>                   # container
```

What each step actually answers:

- `get pod` -> "Is it Running, Pending, CrashLoop, etc.?" Plus `RESTARTS` and `AGE`.
- `describe pod` -> "Why is the cluster's view what it is?" Look at `Events:` at the bottom and `Conditions:` (PodScheduled, Initialized, ContainersReady, Ready).
- `logs` -> "What is the app saying?" If it's not output anything, the app probably never started.
- `logs --previous` -> "What did the app say last time before it crashed?" — the only way to see logs from a previously-terminated container.
- `get events` -> "What is the cluster-side reason?" Image pull errors, scheduler failures, OOM kills.
- `kubectl debug` -> "Let me poke at it" — interactive shell when the image has none, or processes are gone.

Rule of thumb: if you reach for `exec`/`debug` before reading events, you skipped the cheap step.

## Pod States — Diagnostic Recipes

### Pending

Pod accepted by the API but not yet scheduled.

```bash
kubectl describe pod <pod> -n <ns>
```

Look at the bottom `Events:` section. Common reasons:

- `0/3 nodes are available: 3 Insufficient cpu` -> requested CPU exceeds any node's free capacity.
- `0/3 nodes are available: 3 Insufficient memory` -> same for memory.
- `0/3 nodes are available: 1 node(s) didn't match Pod's node affinity/selector` -> labels/affinity wrong.
- `0/3 nodes are available: 3 node(s) had taint {dedicated: gpu}, that the pod didn't tolerate` -> taint mismatch.
- `0/3 nodes are available: 1 node(s) had volume node affinity conflict` -> PV's node affinity vs pod's nodeSelector.
- `FailedScheduling: pod has unbound immediate PersistentVolumeClaims` -> PVC not bound.

Then check capacity vs requests:

```bash
kubectl describe nodes | grep -E "Name:|Allocatable:|Allocated resources:" -A 5
kubectl top node
kubectl get nodes -o custom-columns='NAME:.metadata.name,CPU:.status.capacity.cpu,MEM:.status.capacity.memory'
```

PodDisruptionBudget conflicts (rare for Pending, common for stuck rollouts):

```bash
kubectl get pdb -A
kubectl describe pdb <name> -n <ns>
```

Recipe to fix scheduler "no nodes match":

```bash
kubectl get nodes --show-labels
kubectl get pod <pod> -n <ns> -o yaml | grep -A 20 -E 'nodeSelector|affinity|tolerations'
# Either add the label to a node:
kubectl label node <node-name> <key>=<value>
# Or remove the constraint:
kubectl edit deployment <deploy> -n <ns>
```

### ContainerCreating

Pod scheduled, but kubelet has not yet started the container.

```bash
kubectl describe pod <pod> -n <ns>
```

Common stuck reasons in `Events:`:

- `Failed to pull image "X"` -> see ImagePullBackOff section.
- `MountVolume.SetUp failed for volume "config" : configmap "missing-cm" not found` -> referenced ConfigMap missing.
- `MountVolume.SetUp failed for volume "tls" : secret "missing-secret" not found` -> referenced Secret missing.
- `Unable to attach or mount volumes: unmounted volumes=[data]` -> CSI plugin issue or PV not provisioned.
- `Init:0/2` static for a long time -> init container is running and not finishing; see Init below.
- `CreateContainerConfigError` -> ConfigMap/Secret keyref doesn't exist or is malformed.

Inspect the container config that kubelet refused:

```bash
kubectl get pod <pod> -n <ns> -o yaml | less
# look for env from configMapKeyRef / secretKeyRef
kubectl get configmap -n <ns>
kubectl get secret -n <ns>
```

Init container output:

```bash
kubectl logs <pod> -n <ns> -c <init-container-name>
kubectl logs <pod> -n <ns> -c <init-container-name> --previous
```

### ImagePullBackOff / ErrImagePull

Kubelet cannot fetch the image.

```bash
kubectl describe pod <pod> -n <ns> | tail -30
```

Likely lines you'll see:

- `Failed to pull image "registry/repo:tag": rpc error: code = NotFound desc = manifest unknown` -> typo or image truly does not exist.
- `Failed to pull image "registry/repo:tag": pull access denied, repository does not exist or may require 'docker login'` -> missing/wrong `imagePullSecrets`.
- `Failed to pull image "X": rpc error: code = Unknown desc = Error response from daemon: Get "https://registry/v2/": net/http: TLS handshake timeout` -> network from node to registry.
- `429 Too Many Requests` -> Docker Hub anonymous rate limit; auth or mirror.

Workflow:

```bash
# 1. Reproduce image name exactly
kubectl get pod <pod> -n <ns> -o jsonpath='{.spec.containers[*].image}'

# 2. Verify pull secret exists in the SAME namespace
kubectl get secret -n <ns>
kubectl get sa default -n <ns> -o yaml | grep -A 2 imagePullSecrets

# 3. Inspect pull secret
kubectl get secret <pull-secret> -n <ns> -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d

# 4. Test pull on a node (if you have shell access)
ssh <node>
crictl pull registry/repo:tag       # containerd
docker pull registry/repo:tag        # docker
```

Recreate a registry secret (the most common fix):

```bash
kubectl create secret docker-registry regcred \
  --docker-server=registry.example.com \
  --docker-username=USER \
  --docker-password=PASS \
  --docker-email=stevie@bellis.tech \
  -n <ns>

kubectl patch sa default -n <ns> -p '{"imagePullSecrets":[{"name":"regcred"}]}'
```

Pull-secret in the wrong namespace is the canonical mistake. Secrets are namespaced; you need one per namespace that pulls.

### CrashLoopBackOff

Container started, exited non-zero, and kubelet is backing off restarts.

```bash
kubectl get pod <pod> -n <ns>                  # see RESTARTS column
kubectl describe pod <pod> -n <ns> | grep -A 5 'Last State'
kubectl logs <pod> -n <ns> --previous          # the crashed instance's last logs
kubectl logs <pod> -n <ns>                     # if it has restarted at least once
```

Look for `Last State: Terminated` and the `Exit Code`:

- `137` -> SIGKILL (OOM, or cluster sent SIGKILL after grace period).
- `143` -> SIGTERM (clean shutdown).
- `139` -> SIGSEGV (segfault, native crash).
- `1`   -> generic application error.
- `2`   -> shell builtin misuse / argument error.
- `127` -> command not found in PATH.
- `255` -> often "the entrypoint returned an unhandled error".

Liveness-probe-killed crash loop:

```bash
kubectl describe pod <pod> -n <ns> | grep -E 'Liveness|Killing'
# "Liveness probe failed: HTTP probe failed with statuscode: 503"
# "Container app failed liveness probe, will be restarted"
```

Fix patterns:

- `137` with no probe events -> hit memory limit; bump or fix leak.
- Liveness 503 with healthy app -> probe too aggressive; raise `initialDelaySeconds` or `failureThreshold`, or use a `startupProbe`.
- `127 exec: 'X': not found` -> CMD/ENTRYPOINT wrong; fix Dockerfile or `command:` in the spec.

### OOMKilled

Container exceeded its memory limit and was killed.

```bash
kubectl describe pod <pod> -n <ns> | grep -A 5 'Last State'
# Reason: OOMKilled
# Exit Code: 137
kubectl get events -n <ns> --field-selector reason=OOMKilling
kubectl top pod <pod> -n <ns> --containers
```

Verify the limit:

```bash
kubectl get pod <pod> -n <ns> -o jsonpath='{range .spec.containers[*]}{.name}: {.resources.limits.memory}{"\n"}{end}'
```

Two paths:

- Bump the limit (legitimate need): `kubectl set resources deployment/<d> --limits=memory=1Gi -n <ns>`
- Find the leak: heap dump (`jmap`/`pprof`/`heaptrack`), check working set (`kubectl top`), profile.

OOMKill events at the node level (may not fire if cgroup-only OOM):

```bash
kubectl get events -A --field-selector reason=OOMKilling
```

### Init:Error / Init:CrashLoopBackOff

An init container failed.

```bash
kubectl describe pod <pod> -n <ns>
# Init Containers:
#   wait-db:
#     State: Terminated
#     Reason: Error
#     Exit Code: 1
kubectl logs <pod> -n <ns> -c <init-container-name>
kubectl logs <pod> -n <ns> -c <init-container-name> --previous
```

Common causes:

- Init container waits for a Service that never has endpoints.
- Init container runs a migration that fails.
- Init container has a typo in `command:`.

Pod doesn't get to `Running` until ALL init containers exit 0. They run sequentially, so the first failing one blocks everything.

### Error

Container exited but is not restarting (`restartPolicy: OnFailure`/`Never`, or a Job).

```bash
kubectl logs <pod> -n <ns>                     # output if available
kubectl logs <pod> -n <ns> --previous          # if it restarted before failing finally
kubectl describe pod <pod> -n <ns> | grep -A 5 'Last State'
```

Same exit-code interpretation as CrashLoopBackOff. For Jobs see the Job section below.

### Running but unhealthy

Pod is `Running` but the application doesn't work, or it's `0/1 Ready`.

```bash
kubectl get pod <pod> -n <ns>                  # READY column shows 0/1
kubectl describe pod <pod> -n <ns> | grep -E 'Readiness|Unhealthy'
# "Readiness probe failed: HTTP probe failed with statuscode: 500"
kubectl logs <pod> -n <ns>
```

Verify the probe path actually responds:

```bash
kubectl exec -it <pod> -n <ns> -- curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/healthz
```

If the image has no curl, use `kubectl debug` (see ephemeral containers below).

### Terminating (stuck)

Pod is stuck in `Terminating` for minutes/hours.

```bash
kubectl get pod <pod> -n <ns>
# STATUS: Terminating          AGE: 12m
kubectl describe pod <pod> -n <ns> | grep -A 3 'Conditions\|Finalizers'
```

Causes and fixes:

```bash
# 1. Grace period not elapsed and SIGTERM ignored: force.
kubectl delete pod <pod> -n <ns> --grace-period=0 --force

# 2. Finalizer holding the pod (look for finalizers in the spec).
kubectl get pod <pod> -n <ns> -o yaml | grep -A 5 finalizers
kubectl patch pod <pod> -n <ns> -p '{"metadata":{"finalizers":[]}}' --type=merge

# 3. Volumes stuck unmounting (storage backend issue).
kubectl describe pod <pod> -n <ns> | grep -i mount
# fix the underlying volume driver, then force-delete.
```

Force-delete is destructive: the API removes the object even if kubelet hasn't confirmed the container is gone. Use only when you've verified the workload won't come back somewhere bad.

### Evicted

Node was under pressure and kicked the pod off.

```bash
kubectl get pods -A | grep Evicted
kubectl describe pod <pod> -n <ns>
# Status: Failed
# Reason: Evicted
# Message: The node was low on resource: ephemeral-storage. Container app was using 5Gi, request is 1Gi.
```

Check node pressure:

```bash
kubectl describe node <node-name> | grep -E 'Conditions|MemoryPressure|DiskPressure|PIDPressure'
kubectl top node
```

Cleanup:

```bash
kubectl get pods -A --field-selector status.phase=Failed -o name | xargs -r kubectl delete -A
kubectl delete pods -A --field-selector status.phase=Failed
```

Fix is usually:

- Set ephemeral-storage requests/limits on noisy pods.
- Add bigger nodes or more replicas of the kubelet's image GC can keep up with.
- Move emptyDir bigness to a real volume.

### Unknown

Pod's status is `Unknown` — node lost contact with the control plane.

```bash
kubectl get nodes                              # is the node Ready?
kubectl describe node <node-name>
kubectl get pod <pod> -n <ns> -o wide          # which node?
```

If the node is `NotReady`, kubelet isn't checking in. SSH to the node and:

```bash
systemctl status kubelet
journalctl -u kubelet -n 200 --no-pager
crictl ps                                       # container runtime healthy?
```

After the node comes back, the pod usually self-heals. If the node is gone, you may need to delete the pod (use `--force` if it's stuck) so the controller schedules a replacement.

## The Three Sources of Truth

1. **`kubectl get pod`** — current state. Cheap, machine-readable.
2. **`kubectl describe pod`** — human-readable spec, status, conditions, AND the rolling event log for that object.
3. **`kubectl logs <pod> [-c <container>]`** — what the app printed to stdout/stderr.

Each answers a different question:

```bash
kubectl get pod <pod> -n <ns> -o wide           # phase, restarts, node, IP
kubectl describe pod <pod> -n <ns>               # why phase is what it is
kubectl logs <pod> -n <ns>                       # app's voice
kubectl logs <pod> -n <ns> --previous            # app's last words before crash
```

`--previous` (`-p`) is the only way to see logs from a container that has crashed and been restarted. After the second crash, `--previous` shows the second-to-last logs. After the kubelet GCs old containers (default after death), `--previous` returns `Error from server (BadRequest): previous terminated container ... not found`.

`kubectl edit pod` vs the controller:

```bash
kubectl edit pod <pod> -n <ns>      # changes are applied to live pod
# but the Deployment/StatefulSet controller will overwrite on the next reconcile
```

Edit the controller, not the pod, for persistent fixes:

```bash
kubectl edit deployment <deploy> -n <ns>
kubectl edit statefulset <sts> -n <ns>
kubectl edit daemonset <ds> -n <ns>
```

## kubectl debug — Ephemeral Containers

The `kubectl debug` subcommand attaches a new container to a running pod. Stable since Kubernetes 1.25.

Attach a debug container that shares the target's process namespace:

```bash
kubectl debug -it pod/web -n <ns> --image=busybox:1.36 --target=web -- sh
```

What `--target=web` does: the new container shares PID namespace with container `web`, so `ps -ef` inside the debug container sees the target's processes. You can `kill -TERM <pid>`, `cat /proc/<pid>/status`, etc.

Common debug images:

```bash
kubectl debug -it pod/web -n <ns> --image=busybox:1.36 --target=web -- sh
kubectl debug -it pod/web -n <ns> --image=alpine --target=web -- sh
kubectl debug -it pod/web -n <ns> --image=nicolaka/netshoot --target=web -- bash
kubectl debug -it pod/web -n <ns> --image=ubuntu:22.04 --target=web -- bash
kubectl debug -it pod/web -n <ns> --image=mcr.microsoft.com/azure-cli --target=web -- bash
```

`--share-processes` is the legacy flag for clusters that don't honour `--target` (or for pods where you want shared PID namespace with all containers in the pod, not one specific container):

```bash
kubectl debug -it pod/web -n <ns> --image=busybox --share-processes --copy-to=web-debug
```

`--copy-to=<name>` creates a debug copy of the pod with the modifications. The original pod is untouched. Combine with `--share-processes` and command overrides to inspect a frozen snapshot:

```bash
kubectl debug pod/web -n <ns> \
  --copy-to=web-debug \
  --container=web \
  --image=ubuntu:22.04 \
  --set-image=web=ubuntu:22.04 \
  -- sleep infinity

kubectl exec -it web-debug -n <ns> -c web -- bash
kubectl delete pod web-debug -n <ns>     # cleanup
```

List ephemeral containers on a pod:

```bash
kubectl get pod <pod> -n <ns> -o jsonpath='{.spec.ephemeralContainers[*].name}'
```

Limitations:

- You cannot remove an ephemeral container from a pod. They live until the pod dies.
- Resources on ephemeral containers can't be set the way they can on normal containers.
- Some PSPs/PSS profiles forbid `securityContext.privileged` on ephemeral containers.

## kubectl debug — Node Debugging

When you don't have SSH to a node, run a privileged debug pod on it that chroots to the host:

```bash
kubectl debug node/<node-name> -it --image=ubuntu:22.04 -- bash
```

Inside the pod, `/host` is the node's root filesystem. The canonical recipe:

```bash
chroot /host
# now you are effectively on the node
journalctl -u kubelet -n 200 --no-pager
journalctl -u containerd -n 200 --no-pager
dmesg | tail -100
df -h
ls /var/log/
crictl ps
```

Useful images:

```bash
kubectl debug node/<n> -it --image=ubuntu:22.04 -- bash
kubectl debug node/<n> -it --image=alpine -- sh
kubectl debug node/<n> -it --image=nicolaka/netshoot -- bash
kubectl debug node/<n> -it --image=quay.io/kubernetes/troubleshoot:v0.10 -- bash
```

Cleanup:

```bash
kubectl get pods -A | grep node-debugger
kubectl delete pod -n default node-debugger-<n>-xxxxx
```

The node debug pod is named `node-debugger-<node>-<random>`. It is created in the `default` namespace by default unless you pass `-n`.

## kubectl debug — Pod Copy Pattern

Useful when you want to "swap entrypoint to bash to inspect" without disturbing the original pod (which may be serving traffic).

```bash
# Make a copy of the pod with a new image and command:
kubectl debug pod/web -n <ns> \
  --copy-to=web-shell \
  --set-image=web=ubuntu:22.04 \
  --container=web \
  -- sleep 1h

kubectl exec -it web-shell -n <ns> -c web -- bash
# poke around as user, with the same volume mounts, env vars, network, service account.

kubectl delete pod web-shell -n <ns>
```

What "copy-to" copies: spec.volumes, env, volumeMounts, serviceAccount, securityContext, network policies, all the original pod's plumbing — except the broken entrypoint, which you've replaced.

When to use copy vs ephemeral:

- Ephemeral container: the original is healthy enough to share namespaces; you want to read its memory/proc.
- Copy-to: the original is broken (CrashLoop, distroless), you want to reproduce its environment with a working shell.

## kubectl exec — When and How

Run a command inside a running container:

```bash
kubectl exec <pod> -n <ns> -- ls /etc
kubectl exec -it <pod> -n <ns> -- /bin/sh         # interactive
kubectl exec -it <pod> -n <ns> -c <container> -- bash    # multi-container pod
```

`exec` requires the binary to exist in the image. The "no shell in distroless image" problem:

```bash
kubectl exec -it pod/web -n <ns> -- sh
# error: exec: "sh": executable file not found in $PATH
```

Use `kubectl debug` with an ephemeral container instead:

```bash
kubectl debug -it pod/web -n <ns> --image=busybox --target=web -- sh
```

`exec` vs `debug`:

- `exec` -> attaches to an existing process namespace and runs a binary IN that container.
- `debug` -> creates a new container that may share namespaces, with its own image (so its own binaries).

Common exec recipes:

```bash
# Get the env the app sees
kubectl exec <pod> -- env | sort

# Verify file exists at mount path
kubectl exec <pod> -- ls -la /etc/config

# Test connectivity from inside
kubectl exec <pod> -- curl -sS http://service.ns.svc.cluster.local

# Tail a file inside the container
kubectl exec <pod> -- tail -f /var/log/app.log

# Run a one-shot diagnostic
kubectl exec <pod> -- nslookup kubernetes.default.svc.cluster.local
```

Caveats:

- Output piping needs `--`: `kubectl exec pod -- cmd | grep`, not `kubectl exec pod cmd | grep`.
- For multi-container pods without `-c`, kubectl picks the first container. Errors with `Defaulted container "X" out of: X, Y, Z`.
- Pods with `restartPolicy: Never` that have terminated cannot be exec'd into.

## kubectl logs — The Full Toolkit

Stream container output to your terminal:

```bash
kubectl logs <pod> -n <ns>                          # full log
kubectl logs <pod> -n <ns> -c <container>           # multi-container
kubectl logs -f <pod> -n <ns>                       # follow (tail -f)
kubectl logs <pod> -n <ns> --previous               # previous instance
kubectl logs <pod> -n <ns> -p -c <container>        # both, short form
kubectl logs <pod> -n <ns> --tail=100               # last 100 lines
kubectl logs <pod> -n <ns> --tail=-1                # all lines (default)
kubectl logs <pod> -n <ns> --since=10m              # last 10 minutes
kubectl logs <pod> -n <ns> --since=1h
kubectl logs <pod> -n <ns> --since-time=2026-04-25T08:00:00Z
kubectl logs <pod> -n <ns> --timestamps             # prepend RFC3339Nano timestamps
kubectl logs <pod> -n <ns> --limit-bytes=1048576    # cap output size
kubectl logs <pod> -n <ns> --all-containers         # all containers in this pod
kubectl logs <pod> -n <ns> --all-containers --prefix  # prepend [pod/container]
```

Aggregate logs from a label selector — multiple pods, one stream:

```bash
kubectl logs -l app=web -n <ns> --tail=200 --prefix
kubectl logs -l app=web -n <ns> --all-containers --prefix --max-log-requests=20 -f
```

Why `--max-log-requests`: the default is 5. With more than 5 matching pods you'll get `error: you are attempting to follow N log streams, but maximum allowed concurrency is 5`. Bump it.

Logs from a specific deployment / job / statefulset (kubectl will pick a representative pod):

```bash
kubectl logs deployment/web -n <ns> --tail=200
kubectl logs job/migrate -n <ns>
kubectl logs statefulset/db -n <ns> -c sidecar
```

Logs from a node-level perspective when an app is crashy and pods rotate fast:

```bash
# Watch events while logs roll
kubectl get events -n <ns> --watch &
kubectl logs -l app=web -n <ns> -f --prefix
```

If logs are empty (the app never wrote to stdout/stderr): the container died before emitting, OR the app writes only to a file. Common gotcha:

```bash
# bad: app logs to /var/log/app.log -> nothing shows in kubectl logs
# fix: log to stdout/stderr, or sidecar tail.
```

## Image Pull Failures — Diagnostic Workflow

```bash
kubectl describe pod <pod> -n <ns> | tail -30
```

Look for one of:

- `Failed to pull image "X": rpc error: code = NotFound desc = manifest unknown` — image truly doesn't exist or tag is wrong.
- `Failed to pull image "X": rpc error: code = Unknown desc = Error response from daemon: pull access denied` — auth.
- `ErrImagePull` — first failure; will become `ImagePullBackOff` after retries.
- `ImagePullBackOff` — kubelet has given up retrying for now.

Verify the secret content:

```bash
kubectl get secret <pull-secret> -n <ns> -o yaml
kubectl get secret <pull-secret> -n <ns> -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d | jq
# Should look like:
# {
#   "auths": {
#     "registry.example.com": {
#       "auth": "BASE64_OF_user:pass"
#     }
#   }
# }
```

Verify the secret is referenced:

```bash
# By the pod spec
kubectl get pod <pod> -n <ns> -o yaml | grep -A 2 imagePullSecrets

# Or by the service account
kubectl get sa default -n <ns> -o yaml | grep -A 2 imagePullSecrets
```

Reproduce on a node to isolate registry vs cluster issue:

```bash
kubectl debug node/<node> -it --image=ubuntu -- bash
# inside:
chroot /host
crictl pull registry.example.com/repo:tag
# or
docker pull registry.example.com/repo:tag
```

Gotchas:

- imagePullSecrets must be in the SAME namespace as the pod. Secrets are namespaced.
- `auth` field is `base64(user:pass)`, not `base64(user):base64(pass)`.
- For ECR, the pull secret expires every 12h unless you use the AWS ECR helper or IRSA.
- Docker Hub anonymous rate limit is 100 pulls / 6h per IP. Authenticate or mirror.

## Resource-Exhaustion Workflow

Is my pod hitting its limits, or is it being squeezed by neighbours?

```bash
# Right now
kubectl top pod <pod> -n <ns> --containers
kubectl top node

# What did I ask for and what's my ceiling
kubectl get pod <pod> -n <ns> -o jsonpath='{range .spec.containers[*]}{.name} req:{.resources.requests} lim:{.resources.limits}{"\n"}{end}'

# What's the node's headroom
kubectl describe node <node> | grep -A 5 'Allocated resources'
# Allocated resources:
#   cpu                3500m (87%)
#   memory             7Gi   (90%)
```

Reading the node's allocatable vs requested vs limited:

```bash
kubectl describe node <n> | grep -E 'Capacity:|Allocatable:|Allocated resources:' -A 7
```

- `Capacity` -> total node hardware.
- `Allocatable` -> what kubelet will let pods use (capacity minus system reservations).
- `Allocated resources: requests` -> sum of all pod requests on this node (the scheduler's view).
- Limits can exceed Allocatable; that's how you over-commit and where evictions come from.

OOMKilled events:

```bash
kubectl get events -A --field-selector reason=OOMKilling --sort-by=.lastTimestamp
kubectl get events -n <ns> --field-selector reason=Evicted
```

Find the noisiest pods on a node:

```bash
kubectl top pod -A --sort-by=memory | head -20
kubectl top pod -A --sort-by=cpu | head -20
```

If `kubectl top` returns `Metrics API not available`, install metrics-server:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

## Networking Debugging

Service has no endpoints (the most common networking ticket):

```bash
kubectl get svc <svc> -n <ns>
kubectl get endpoints <svc> -n <ns>
# NAME   ENDPOINTS   AGE
# web    <none>      5m
```

What it means: no Pod matches the Service's selector AND is `Ready`. Diagnose:

```bash
kubectl describe svc <svc> -n <ns> | grep -E 'Selector|Endpoints'
# Selector: app=web,tier=front
kubectl get pods -n <ns> -l app=web,tier=front
# 0 pods? -> labels wrong. some pods, all NotReady? -> readiness probe failing.
kubectl get pods -n <ns> -l app=web -o wide
```

In-cluster connectivity check from a debug pod:

```bash
kubectl run netshoot --rm -it --image=nicolaka/netshoot -n <ns> -- bash
# inside:
curl -sS http://web.<ns>.svc.cluster.local:8080
nslookup web.<ns>.svc.cluster.local
dig +short web.<ns>.svc.cluster.local
nc -zv web 8080
```

The Cluster DNS naming convention:

```
<service>.<namespace>.svc.cluster.local
<pod-ip-with-dashes>.<namespace>.pod.cluster.local
```

Local testing — `port-forward`:

```bash
kubectl port-forward pod/<pod> 8080:80 -n <ns>
kubectl port-forward svc/<svc> 8080:80 -n <ns>
kubectl port-forward deployment/<d> 8080:80 -n <ns>
# then on your laptop:
curl http://localhost:8080
```

Local API access — `kubectl proxy`:

```bash
kubectl proxy --port=8001 &
curl http://localhost:8001/api/v1/namespaces/<ns>/services/<svc>/proxy/
curl http://localhost:8001/api/v1/namespaces/<ns>/pods/<pod>/proxy/
```

CoreDNS check:

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns
kubectl logs -n kube-system -l k8s-app=kube-dns --tail=50
kubectl get cm coredns -n kube-system -o yaml
# from a pod:
kubectl exec -it <pod> -n <ns> -- nslookup kubernetes.default.svc.cluster.local
kubectl exec -it <pod> -n <ns> -- cat /etc/resolv.conf
```

`/etc/resolv.conf` inside a pod should look like:

```
nameserver 10.96.0.10
search <ns>.svc.cluster.local svc.cluster.local cluster.local
options ndots:5
```

Netshoot — the canonical networking debug image:

```bash
kubectl run netshoot --rm -it --image=nicolaka/netshoot -- bash
# Comes with: curl, dig, nslookup, host, tcpdump, mtr, traceroute, iperf3,
# ip, ss, nmap, nc, telnet, jq, drill, ngrep, nethogs, conntrack, nft.
```

Capture packets inside a pod's network namespace:

```bash
kubectl debug -it pod/web --image=nicolaka/netshoot --target=web -- bash
# inside:
tcpdump -i eth0 -nn 'host 10.244.1.42 and port 8080' -w /tmp/cap.pcap
```

Then copy out:

```bash
kubectl cp <ns>/web:/tmp/cap.pcap ./cap.pcap -c <debug-container>
```

## NetworkPolicy Debugging

```bash
kubectl get networkpolicy -A
kubectl describe networkpolicy <name> -n <ns>
```

Test connectivity between two pods:

```bash
# from pod A in ns-a
kubectl exec -it -n ns-a pod-a -- nc -zv pod-b.ns-b.svc.cluster.local 8080
# OR by IP
kubectl get pod pod-b -n ns-b -o jsonpath='{.status.podIP}'
kubectl exec -it -n ns-a pod-a -- nc -zv 10.244.2.5 8080
```

The "block-by-default + allow specific" review:

```bash
kubectl get networkpolicy -n <ns> -o yaml | grep -E 'name:|podSelector:|policyTypes:|ingress:|egress:'
```

A NetworkPolicy with empty `podSelector: {}` and `policyTypes: [Ingress, Egress]` denies everything. You then add allow policies on top.

CNI-specific tooling:

- Calico: `calicoctl get networkpolicy -A`, `calicoctl get globalnetworkpolicy`, `calicoctl node diags`.
- Cilium: `cilium status`, `cilium connectivity test`, `cilium monitor`, `cilium policy get`, `cilium endpoint list`.
- Weave: `weave status`, `weave report`.

Symptom: traffic dropped silently with no logs in the application. Cause: a NetworkPolicy is dropping it. Fix: temporarily allow-all to confirm, then add a precise rule.

```bash
# Temporary allow-all (DANGEROUS — for diagnosis only):
cat <<EOF | kubectl apply -n <ns> -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-all-debug
spec:
  podSelector: {}
  ingress:
  - {}
  egress:
  - {}
  policyTypes: [Ingress, Egress]
EOF
# verify, diagnose, then DELETE:
kubectl delete networkpolicy allow-all-debug -n <ns>
```

## RBAC Debugging

"Forbidden: cannot get resource X" tracing:

```bash
# error message format:
# Error from server (Forbidden): pods is forbidden: User "system:serviceaccount:ns:sa"
# cannot list resource "pods" in API group "" in the namespace "ns"
```

`auth can-i` is your friend:

```bash
kubectl auth can-i list pods                                    # me, default ns
kubectl auth can-i list pods -n <ns>                            # me, specific ns
kubectl auth can-i list pods --all-namespaces
kubectl auth can-i create deployments --as=alice
kubectl auth can-i create pods --as=system:serviceaccount:<ns>:<sa>
kubectl auth can-i --list                                        # what can I do?
kubectl auth can-i --list --as=system:serviceaccount:<ns>:<sa>
kubectl auth can-i --list --namespace=<ns>
```

Find what bindings exist:

```bash
kubectl get rolebindings -A
kubectl get clusterrolebindings
kubectl describe rolebinding <name> -n <ns>
kubectl describe clusterrolebinding <name>

# What roles does this SA have?
kubectl get rolebindings,clusterrolebindings --all-namespaces -o json | \
  jq -r '.items[] | select(.subjects[]?.name=="<sa>") | .metadata.name'
```

Inspect a role's permissions:

```bash
kubectl describe role <role> -n <ns>
kubectl describe clusterrole <role>
kubectl get clusterrole <role> -o yaml
```

Trace impersonation chain when the calling identity isn't obvious — enable audit logs on the apiserver and search:

```bash
# in /var/log/kubernetes/audit.log on control plane (or via a SIEM):
grep '"verb":"list","resource":"pods"' audit.log | jq 'select(.responseStatus.code == 403)'
```

Quick fix patterns:

```bash
# Grant a SA the read role in a namespace:
kubectl create rolebinding sa-read --serviceaccount=<ns>:<sa> --clusterrole=view -n <ns>

# Grant cluster-wide admin to a user (use with care):
kubectl create clusterrolebinding alice-admin --user=alice --clusterrole=cluster-admin
```

## PVC / PV Debugging

```bash
kubectl get pvc -n <ns>
# NAME      STATUS    VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS   AGE
# data      Pending   <none>                              standard       3m
kubectl describe pvc data -n <ns>
```

`Pending` PVC reasons in events:

- `ProvisioningFailed`: the StorageClass provisioner errored. Look at the message.
- `WaitForFirstConsumer`: VolumeBindingMode is `WaitForFirstConsumer` — PVC binds when a Pod that uses it is scheduled. This is normal until a Pod references it.
- `no persistent volumes available for this claim and no storage class is set`: no default StorageClass.

Check StorageClasses:

```bash
kubectl get storageclass
# NAME                 PROVISIONER          RECLAIMPOLICY   VOLUMEBINDINGMODE      AGE
# standard (default)   kubernetes.io/gce-pd Delete          WaitForFirstConsumer   30d
kubectl describe storageclass <name>
```

Find the default StorageClass:

```bash
kubectl get sc -o jsonpath='{range .items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")]}{.metadata.name}{"\n"}{end}'
```

Set a default StorageClass:

```bash
kubectl patch storageclass standard -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

Bound PV:

```bash
kubectl get pv
kubectl describe pv <pv-name>
# look at: Source (csi/hostPath/nfs), Claim (which PVC), AccessModes
```

AccessMode mismatch:

- PVC asks for `ReadWriteMany`, the StorageClass only supports `ReadWriteOnce` -> stays Pending.
- Multiple pods on different nodes with `ReadWriteOnce` PVC -> `Multi-Attach error`.

Recipes:

```bash
# Force-unbind a stuck PVC:
kubectl patch pvc data -n <ns> -p '{"metadata":{"finalizers":null}}'

# Same for PV:
kubectl patch pv <pv> -p '{"metadata":{"finalizers":null}}'

# Reclaim policy: change Retain -> Delete (or vice versa) on a PV:
kubectl patch pv <pv> -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

CSI driver health:

```bash
kubectl get pods -n kube-system -l app=ebs-csi-controller
kubectl logs -n kube-system <csi-controller-pod> -c csi-provisioner
```

## ConfigMap / Secret Debugging

Pod is stuck `CreateContainerConfigError` or `CreateContainerError`:

```bash
kubectl describe pod <pod> -n <ns> | tail -20
# Error: configmap "app-config" not found
# Error: couldn't find key "DB_HOST" in ConfigMap <ns>/app-config
# Error: secret "tls" not found
```

Diagnose:

```bash
kubectl get configmap -n <ns>
kubectl get secret -n <ns>
kubectl describe configmap <name> -n <ns>
kubectl get configmap <name> -n <ns> -o yaml
kubectl get secret <name> -n <ns> -o yaml
```

Secret data is base64-encoded — decode with `base64 -d`:

```bash
kubectl get secret <name> -n <ns> -o jsonpath='{.data.password}' | base64 -d
echo                                          # newline, base64 -d does not add one

# All keys:
kubectl get secret <name> -n <ns> -o json | jq '.data | map_values(@base64d)'
```

Create from literal vs file:

```bash
kubectl create configmap app-config --from-literal=DB_HOST=db --from-literal=DB_PORT=5432 -n <ns>
kubectl create configmap app-config --from-file=app.properties -n <ns>
kubectl create configmap app-config --from-file=key=path/to/file -n <ns>

kubectl create secret generic db-creds --from-literal=password=hunter2 -n <ns>
kubectl create secret generic db-creds --from-file=./db.password -n <ns>
kubectl create secret tls tls-cert --cert=tls.crt --key=tls.key -n <ns>
kubectl create secret docker-registry regcred ... -n <ns>
```

Update a ConfigMap and restart consumers:

```bash
kubectl create configmap app-config --from-file=app.properties --dry-run=client -o yaml | \
  kubectl apply -f - -n <ns>
kubectl rollout restart deployment/<d> -n <ns>
```

Why `rollout restart`: ConfigMaps mounted as volumes get updated in the pod, but apps usually don't re-read them. Mounted as env vars, they don't update at all without a pod restart.

## Job / CronJob Debugging

A Job that won't complete:

```bash
kubectl get jobs -n <ns>
# NAME      COMPLETIONS   DURATION   AGE
# migrate   0/1           5m         5m
kubectl describe job migrate -n <ns>
# Events:
# BackoffLimitExceeded   Job has reached the specified backoff limit
kubectl get pods -n <ns> -l job-name=migrate
kubectl logs -n <ns> -l job-name=migrate
kubectl logs job/migrate -n <ns>
```

Backoff limit reached -> the pods kept failing past `.spec.backoffLimit` (default 6). Look at the failed pods' logs.

CronJob:

```bash
kubectl get cronjob -n <ns>
# NAME      SCHEDULE     SUSPEND   ACTIVE   LAST SCHEDULE   AGE
# nightly   0 2 * * *    False     0        12h             7d
kubectl describe cronjob nightly -n <ns>
# Last Schedule Time, Active Jobs, Events
kubectl get jobs -n <ns> -l cronjob-name=nightly                  # newer K8s
kubectl get jobs -n <ns> | grep nightly                            # all jobs from this cron
```

Common CronJob issues:

- `ConcurrencyPolicy: Forbid` and previous job still running -> `Cannot determine if job needs to be started` event, run is skipped.
- Clock skew between control plane and worker nodes -> jobs run at the wrong times.
- `startingDeadlineSeconds` too small + missed schedule (cluster down) -> job is skipped permanently.

Clean up old failed jobs:

```bash
kubectl delete job -n <ns> -l cronjob-name=nightly --field-selector status.successful=0
```

Manually trigger a CronJob (smoke test):

```bash
kubectl create job manual-run --from=cronjob/nightly -n <ns>
kubectl logs job/manual-run -n <ns>
kubectl delete job manual-run -n <ns>
```

## Ingress / Service Debugging

```bash
kubectl get ingress -A
kubectl describe ingress <name> -n <ns>
kubectl get ingressclass
```

`kubectl describe ingress` shows the class, the rules, and the resolved backends:

```
Rules:
  Host          Path  Backends
  ----          ----  --------
  app.example.com
                /     web:80 (10.244.1.5:8080)
```

If `Backends` shows `<error: endpoints "web" not found>`, the Service is missing or its name is wrong. If it shows `(<none>)`, the Service has no Ready endpoints.

Service with no endpoints:

```bash
kubectl get svc <svc> -n <ns> -o yaml | grep -A 5 selector
kubectl get pods -n <ns> -l <selector-from-svc> -o wide
kubectl get endpoints <svc> -n <ns>
kubectl get endpointslice -n <ns> -l kubernetes.io/service-name=<svc>
```

`502 Bad Gateway` from an Ingress controller:

- Upstream pod is down or has crashed -> check `kubectl get pods -n <ns> -l <app>`.
- Wrong port -> Service's `targetPort` doesn't match container's listening port.
- TLS termination mismatch -> Ingress assumes HTTP backend, app speaks HTTPS, or vice versa.

Test the path bypassing the Ingress:

```bash
kubectl port-forward svc/<svc> 8080:80 -n <ns>
curl -v http://localhost:8080/
```

If that works, your problem is in the Ingress controller. If it doesn't, your problem is in the workload.

Ingress controller logs:

```bash
kubectl get pods -A | grep -E 'ingress|nginx|traefik|haproxy|envoy'
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx --tail=100
```

`IngressClass` selection:

```bash
kubectl get ingressclass
# NAME    CONTROLLER                 PARAMETERS   AGE
# nginx   k8s.io/ingress-nginx       <none>       7d
kubectl get ingress <ing> -n <ns> -o jsonpath='{.spec.ingressClassName}'
```

If `ingressClassName` is unset and you have multiple classes, none might claim the Ingress. Either set it explicitly or annotate one class as default.

## Liveness / Readiness Probe Debugging

```bash
kubectl describe pod <pod> -n <ns> | grep -A 10 -E 'Liveness|Readiness|Startup'
# Liveness: http-get http://:8080/healthz delay=30s timeout=1s period=10s #success=1 #failure=3
# Readiness: http-get http://:8080/ready  delay=5s  timeout=1s period=5s  #success=1 #failure=3
kubectl describe pod <pod> -n <ns> | grep -E 'Unhealthy|Killing|Liveness probe failed'
```

Reproduce the probe from inside the pod:

```bash
kubectl exec -it <pod> -n <ns> -- curl -sv http://localhost:8080/healthz
kubectl exec -it <pod> -n <ns> -- wget -qO- http://localhost:8080/healthz
# tcp probe:
kubectl exec -it <pod> -n <ns> -- nc -zv localhost 8080
# exec probe:
kubectl exec -it <pod> -n <ns> -- /bin/healthcheck
```

For distroless images, use `kubectl debug` with `--target` and a busybox image to run curl.

Tuning patterns:

```yaml
# Slow-start app (JVM, .NET, big Python imports): use startupProbe so liveness doesn't kill it.
startupProbe:
  httpGet: { path: /healthz, port: 8080 }
  failureThreshold: 30        # 30 * 10s = 5 minutes max startup
  periodSeconds: 10
livenessProbe:
  httpGet: { path: /healthz, port: 8080 }
  periodSeconds: 10
  failureThreshold: 3         # 30s of unhealthy = restart
readinessProbe:
  httpGet: { path: /ready, port: 8080 }
  periodSeconds: 5
  failureThreshold: 3
```

Common probe mistakes:

- Liveness too aggressive on a slow app -> CrashLoop.
- Readiness on a path that depends on a not-yet-available DB -> never Ready.
- Probe goes through a proxy/sidecar that isn't up yet -> false failure on startup.
- `httpGet` with `host:` set -> probe doesn't go to the pod, goes elsewhere; usually leave `host:` unset.

## Cluster-Level Debugging

```bash
kubectl get nodes
kubectl describe node <node>
kubectl top node
kubectl get nodes -o wide
```

`describe node` highlights:

- `Conditions:` — Ready, MemoryPressure, DiskPressure, PIDPressure, NetworkUnavailable.
- `Capacity:` / `Allocatable:` — hardware vs schedulable.
- `System Info:` — kernel, container runtime, kubelet version.
- `Allocated resources:` — sum of pod requests and limits.
- `Events:` — recent kubelet/scheduler messages.

Cluster-wide events sorted newest-first:

```bash
kubectl get events -A --sort-by=.lastTimestamp | tail -50
kubectl get events -A --field-selector type!=Normal --sort-by=.lastTimestamp
kubectl get events -A -w                                  # watch
```

`componentstatuses` is deprecated since 1.19, but if you're on an older cluster:

```bash
kubectl get componentstatuses
# NAME                 STATUS    MESSAGE             ERROR
# scheduler            Healthy   ok
# controller-manager   Healthy   ok
# etcd-0               Healthy   {"health":"true"}
```

On modern clusters use the kube-system namespace pods directly:

```bash
kubectl get pods -n kube-system
kubectl logs -n kube-system kube-apiserver-<node> --tail=100
kubectl logs -n kube-system kube-controller-manager-<node> --tail=100
kubectl logs -n kube-system kube-scheduler-<node> --tail=100
kubectl logs -n kube-system etcd-<node> --tail=100
kubectl logs -n kube-system -l k8s-app=kube-dns --tail=100
```

Kubelet logs from a node (via `kubectl debug node`):

```bash
kubectl debug node/<n> -it --image=ubuntu -- bash
chroot /host
journalctl -u kubelet -n 200 --no-pager
journalctl -u containerd -n 200 --no-pager
```

API server health:

```bash
kubectl get --raw='/livez?verbose'
kubectl get --raw='/readyz?verbose'
kubectl get --raw='/healthz/etcd'
```

## Capturing Pod State for Investigation

The "incident snapshot bundle" — capture everything before the pod restarts, gets evicted, or someone redeploys:

```bash
NS=production
POD=web-7ff8f8d4d6-abcde
DIR=incident-$(date +%Y%m%dT%H%M%S)
mkdir -p $DIR && cd $DIR

kubectl describe pod $POD -n $NS > pod-describe.txt
kubectl get pod $POD -n $NS -o yaml > pod.yaml
kubectl logs $POD -n $NS --all-containers --prefix > logs.txt
kubectl logs $POD -n $NS --all-containers --prefix --previous > logs-previous.txt 2>/dev/null
kubectl get events -n $NS --sort-by=.lastTimestamp > events.txt
kubectl get pods -n $NS -o wide > pods-list.txt
kubectl get deploy,sts,ds -n $NS -o yaml > workloads.yaml
kubectl get svc,ingress -n $NS -o yaml > services.yaml
kubectl describe node $(kubectl get pod $POD -n $NS -o jsonpath='{.spec.nodeName}') > node.txt

cd ..
tar czf incident-$NS-$POD-$(date +%Y%m%dT%H%M%S).tgz $DIR
```

Heap/thread dumps before kill:

```bash
# JVM:
kubectl exec <pod> -n <ns> -c <container> -- jcmd 1 GC.heap_dump /tmp/heap.hprof
kubectl cp <ns>/<pod>:/tmp/heap.hprof ./heap.hprof -c <container>

# Go pprof (if app exposes /debug/pprof):
kubectl port-forward <pod> 6060:6060 -n <ns> &
go tool pprof -png http://localhost:6060/debug/pprof/heap > heap.png
```

## "Why is my pod restarting?" Diagnostic

```bash
kubectl get pod <pod> -n <ns> -o wide
# READY   STATUS    RESTARTS         AGE
# 1/1     Running   12 (5m ago)      3h
kubectl describe pod <pod> -n <ns> | grep -A 8 'Last State'
# Last State: Terminated
#   Reason:   OOMKilled
#   Exit Code: 137
#   Started:  ...
#   Finished: ...
kubectl logs <pod> -n <ns> --previous --tail=200
```

Common causes:

- **OOMKilled (137)** — exceeded memory limit. Bump or fix leak.
- **Liveness probe failed** — `kubectl describe pod` shows `Liveness probe failed: ...`. Tune probe or fix endpoint.
- **Signal from app (143)** — process exited cleanly, possibly from a deployment-side update or app-internal kill.
- **Image rolled out** — controller restarted pod with a new image; that's not a "crash". Check `kubectl get rs -n <ns>`.
- **Node lost / drained** — pod recreated elsewhere; restart count resets per pod, but the controller-level history shows it.

Restart count by container:

```bash
kubectl get pod <pod> -n <ns> -o jsonpath='{range .status.containerStatuses[*]}{.name}: {.restartCount}{"\n"}{end}'
```

## "Why is my deployment stuck rolling out?" Diagnostic

```bash
kubectl rollout status deployment/<d> -n <ns>
# Waiting for deployment "web" rollout to finish: 1 of 3 updated replicas are available...
kubectl get deployment <d> -n <ns>
kubectl describe deployment <d> -n <ns> | tail -30
kubectl get rs -n <ns> -l app=<name>                # ReplicaSets
kubectl get pods -n <ns> -l app=<name>
```

`describe deployment` events to look for:

- `ReplicaSetUpdated` -> normal progress.
- `ReplicaSetCreateError` -> selector or quota issue.
- `ProgressDeadlineExceeded` -> the rollout did not progress within `progressDeadlineSeconds` (default 600s).

Why rollouts get stuck:

- New pods crash on startup -> see CrashLoopBackOff section.
- New pods never become Ready -> see Readiness Probe section.
- PDB blocks eviction of old pods:

```bash
kubectl get pdb -n <ns>
kubectl describe pdb <name> -n <ns>
# DISRUPTIONS ALLOWED: 0
```

Roll back to the previous revision:

```bash
kubectl rollout undo deployment/<d> -n <ns>
kubectl rollout undo deployment/<d> -n <ns> --to-revision=3
kubectl rollout history deployment/<d> -n <ns>
kubectl rollout history deployment/<d> -n <ns> --revision=3
kubectl rollout pause deployment/<d> -n <ns>
kubectl rollout resume deployment/<d> -n <ns>
```

Force a re-roll without changing the spec (picks up new ConfigMap, etc.):

```bash
kubectl rollout restart deployment/<d> -n <ns>
```

## "What changed?" Diagnostic

```bash
kubectl rollout history deployment/<d> -n <ns>
# REVISION  CHANGE-CAUSE
# 1         <none>
# 2         kubectl set image ...
# 3         kubectl apply --filename=deploy.yaml ...
kubectl rollout history deployment/<d> -n <ns> --revision=3
```

What's actually deployed (the digest, not the tag):

```bash
kubectl get pod <pod> -n <ns> -o jsonpath='{range .status.containerStatuses[*]}{.name}: {.imageID}{"\n"}{end}'
# web: docker-pullable://registry/web@sha256:abc123...
kubectl describe pod <pod> -n <ns> | grep -E 'Image|Image ID'
```

Recent cluster events as a "what changed in the last hour":

```bash
kubectl get events -A --sort-by=.lastTimestamp | tail -100
kubectl get events -n <ns> --field-selector type=Warning
```

Diff a live object against your manifests (kubectl >= 1.18):

```bash
kubectl diff -f deploy.yaml
```

If you use ArgoCD/Flux, check there too — they may have synced something behind your back.

## Useful Debug Pods

Common one-liners:

```bash
# Shell pod for a quick poke
kubectl run shell --rm -it --image=alpine -- sh
kubectl run shell --rm -it --image=busybox:1.36 -- sh
kubectl run shell --rm -it --image=ubuntu:22.04 -- bash

# Networking toolbox
kubectl run nettool --rm -it --image=nicolaka/netshoot -- bash

# OS-level toolbox (lsof, strace, gdb)
kubectl run os --rm -it --image=quay.io/kubernetes/troubleshoot:v0.10 -- bash

# Same node as a target pod (use nodeSelector)
NODE=$(kubectl get pod <pod> -n <ns> -o jsonpath='{.spec.nodeName}')
kubectl run -it --rm shell --image=alpine \
  --overrides='{"spec":{"nodeSelector":{"kubernetes.io/hostname":"'$NODE'"}}}' -- sh

# Debug pod with host networking
kubectl run -it --rm hostnet --image=nicolaka/netshoot \
  --overrides='{"spec":{"hostNetwork":true}}' -- bash

# Debug pod with privileged + host PID + chroot
kubectl run -it --rm hostpriv --image=alpine \
  --overrides='{"spec":{"hostPID":true,"containers":[{"name":"a","image":"alpine","stdin":true,"tty":true,"securityContext":{"privileged":true},"command":["nsenter","-t","1","-m","-u","-i","-n","-p","--","sh"]}]}}' -- sh
```

`netshoot` includes: `curl`, `wget`, `dig`, `nslookup`, `host`, `tcpdump`, `mtr`, `traceroute`, `iperf3`, `ip`, `ss`, `nmap`, `nc`, `telnet`, `jq`, `drill`, `ngrep`, `nethogs`, `conntrack`, `nft`, `ethtool`.

`troubleshoot`/`toolbox` includes: `lsof`, `strace`, `gdb`, `tcpdump`, `perf`, `htop`, `iotop`.

## Common Misconfigurations and Fixes

```bash
# bad: imagePullSecrets in wrong namespace
# Pod in ns=app references secret regcred that lives in ns=default.
kubectl get secret regcred -n app                # NotFound
# fix: secret must be in the same namespace as the pod
kubectl create secret docker-registry regcred ... -n app
```

```bash
# bad: ServiceAccount lacks permission
kubectl logs <controller> -n <ns>
# error: pods "X" is forbidden: User "system:serviceaccount:ns:default" cannot create
# fix: bind the SA to a Role/ClusterRole
kubectl create rolebinding controller-edit \
  --clusterrole=edit --serviceaccount=<ns>:<sa> -n <ns>
```

```bash
# bad: NetworkPolicy denies traffic by default with no allow rules
# Symptom: timeouts between pods that used to talk fine.
kubectl get networkpolicy -A
# fix: add an allow rule for the specific selector
cat <<EOF | kubectl apply -n <ns> -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: { name: allow-web-to-db }
spec:
  podSelector: { matchLabels: { app: db } }
  ingress:
  - from:
    - podSelector: { matchLabels: { app: web } }
    ports:
    - port: 5432
EOF
```

```bash
# bad: PVC accessMode mismatch with PV
# PVC asks ReadWriteMany, StorageClass only supports ReadWriteOnce.
kubectl describe pvc data -n <ns> | grep AccessModes
kubectl describe storageclass standard | grep -i provisioner
# fix: use ReadWriteOnce, or pick a SC that supports RWX (e.g. NFS, EFS, CephFS).
```

```bash
# bad: liveness probe too aggressive
# JVM app takes 90s to warm up; liveness fires at 30s and kills it.
# fix: add a startupProbe; keep liveness conservative.
startupProbe:
  httpGet: { path: /healthz, port: 8080 }
  failureThreshold: 30
  periodSeconds: 10
```

```bash
# bad: resource requests = 0 (or omitted)
# Scheduler over-commits; under load, pod is the first to be evicted.
kubectl get pod <pod> -n <ns> -o jsonpath='{.spec.containers[*].resources}'
# fix: set realistic requests (use kubectl top to baseline).
resources:
  requests: { cpu: 100m, memory: 128Mi }
  limits:   { cpu: 500m, memory: 256Mi }
```

```bash
# bad: hostPath on a multi-node cluster
# Pod scheduled to node-2 but data is on node-1's hostPath.
volumes:
- name: data
  hostPath: { path: /var/data }
# fix: use a PVC backed by a real PV.
volumes:
- name: data
  persistentVolumeClaim: { claimName: data }
```

```bash
# bad: latest tag in deployment
# rollback is impossible because :latest's previous content is lost.
image: registry/web:latest
# fix: pin to digest.
image: registry/web@sha256:abc123def456...
# or use immutable tags (semver, git-sha).
image: registry/web:v1.4.2
image: registry/web:sha-3a1b2c3
```

```bash
# bad: emptyDir for persistent data
# Pod restart -> data gone (emptyDir lives on the node, dies with the pod).
volumes:
- name: data
  emptyDir: {}
# fix: use a PVC.
volumes:
- name: data
  persistentVolumeClaim: { claimName: data }
```

```bash
# bad: missing terminationGracePeriodSeconds for stateful workloads
# Default is 30s; databases may need longer to flush.
spec:
  terminationGracePeriodSeconds: 30
# fix: tune to the app's shutdown time.
spec:
  terminationGracePeriodSeconds: 300
```

```bash
# bad: same selector across two Services -> traffic split unpredictably
kubectl get svc -n <ns> -o wide
# fix: use unique label sets per Service / workload.
```

```bash
# bad: matchLabels on Deployment != labels on PodTemplate
spec:
  selector:
    matchLabels: { app: web }
  template:
    metadata:
      labels: { app: webpage }   # mismatch
# Deployment won't manage its pods (or will fail to create).
# fix: keep them identical.
```

## Common Errors and Fixes (EXACT text)

```
0/3 nodes are available: 3 Insufficient cpu, 2 Insufficient memory.
```

You requested more than any single node has free. Either reduce `requests` or scale the cluster.

```bash
kubectl describe nodes | grep -E 'Allocatable|Allocated' -A 5
kubectl set resources deployment/<d> --requests=cpu=100m,memory=128Mi -n <ns>
```

```
0/3 nodes are available: 3 node(s) didn't match Pod's node affinity/selector.
```

Your `nodeSelector` / `affinity` doesn't match any node's labels.

```bash
kubectl get nodes --show-labels
kubectl get pod <pod> -n <ns> -o yaml | grep -A 10 -E 'nodeSelector|affinity'
kubectl label node <node> <key>=<value>          # or fix the spec
```

```
0/3 nodes are available: 3 node(s) had taint {dedicated: gpu}, that the pod didn't tolerate.
```

Add a toleration for the taint, or remove the taint from at least one node.

```yaml
tolerations:
- key: dedicated
  operator: Equal
  value: gpu
  effect: NoSchedule
```

```bash
kubectl taint nodes <node> dedicated:NoSchedule-       # remove taint
```

```
Failed to pull image "registry/repo:tag": rpc error: code = NotFound desc = manifest unknown.
```

Image typo or it doesn't exist. Verify the registry, repo, and tag.

```bash
kubectl get pod <pod> -n <ns> -o jsonpath='{.spec.containers[*].image}'
# verify by pulling locally:
docker pull registry/repo:tag
```

```
container has runAsNonRoot and image will run as root.
```

Pod's `securityContext.runAsNonRoot: true` but the image's USER is root (or unset).

```bash
# fix in image: USER 1000 in Dockerfile
# fix in pod: set runAsUser explicitly
spec:
  securityContext:
    runAsUser: 1000
    runAsNonRoot: true
```

```
exec: "X": executable file not found in $PATH.
```

Your `command:` / `args:` references a binary that's not in the image, or PATH is wrong.

```bash
kubectl debug -it pod/<pod> --image=alpine --target=<container> -- sh
# inside, find the binary:
which X || find / -name X 2>/dev/null
```

```
Error: configmap "X" not found.
```

ConfigMap referenced by the pod doesn't exist in this namespace.

```bash
kubectl get configmap -n <ns>
kubectl create configmap X --from-literal=KEY=VALUE -n <ns>
```

```
Error: couldn't find key X in ConfigMap <ns>/Y.
```

ConfigMap exists but the key in `configMapKeyRef.key` doesn't.

```bash
kubectl get configmap Y -n <ns> -o yaml
# add the key:
kubectl edit configmap Y -n <ns>
```

```
MountVolume.SetUp failed for volume "X" : secret "Y" not found.
```

Same as ConfigMap, but for Secrets.

```bash
kubectl get secret -n <ns>
kubectl create secret generic Y --from-literal=KEY=VALUE -n <ns>
```

```
back-off restarting failed container
```

CrashLoopBackOff in human form. Read `kubectl logs --previous`.

```
The Deployment "X" is invalid: spec.template.metadata.labels: Invalid value: ...
matchLabels selector does not match template labels
```

Your `spec.selector.matchLabels` doesn't equal your `spec.template.metadata.labels`. Keep them identical.

```
forbidden: error looking up service account default/default: serviceaccount "default" not found
```

Namespace just created (or someone deleted the default SA). Recreate:

```bash
kubectl create sa default -n <ns>
```

```
Unable to mount volumes for pod "X": timeout expired waiting for volumes to attach or mount
```

CSI driver / storage backend issue. Check the driver pods and the cloud volume's status (e.g., AWS EBS still attached to a previous node).

```bash
kubectl describe pod <pod> -n <ns> | grep -A 10 Volume
kubectl get pods -n kube-system | grep csi
```

```
Error from server (NotFound): pods "X" not found
```

You're in the wrong namespace, or the pod is already gone.

```bash
kubectl get pods -A | grep X
kubectl config view --minify | grep namespace:
```

```
The connection to the server <api> was refused - did you specify the right host or port?
```

API server unreachable. Check network, kubeconfig, VPN.

```bash
kubectl cluster-info
kubectl config view --minify
```

## Idioms

The canonical aliases:

```bash
alias k=kubectl
alias kn='kubectl config set-context --current --namespace'
alias kgp='kubectl get pods'
alias kgs='kubectl get svc'
alias kgd='kubectl get deploy'
alias kdp='kubectl describe pod'
alias kl='kubectl logs'
alias klf='kubectl logs -f'
alias kex='kubectl exec -it'

source <(kubectl completion bash)        # zsh: kubectl completion zsh
complete -F __start_kubectl k
```

Krew — kubectl plugin manager:

```bash
# install krew once, then:
kubectl krew install ctx ns tree neat view-secret tail stern who-can
kubectl ctx                       # switch context (kubectx)
kubectl ns <ns>                   # switch namespace (kubens)
kubectl tree deployment/<d>       # ownership tree
kubectl neat -- ...               # strip noisy fields from output
kubectl view-secret <secret>      # decoded secret view
kubectl tail <pod>                # better tail
kubectl who-can list pods         # RBAC reverse lookup
```

`stern` for multi-pod log tailing (the better `kubectl logs -l ... -f`):

```bash
stern <pattern> -n <ns>
stern -n <ns> 'web-.*'
stern --since 10m -l app=web -n <ns>
```

The polling pattern when you want to watch state change:

```bash
watch -n 1 'kubectl get pods -n <ns>'
watch -n 2 'kubectl top pod -n <ns>'
kubectl get pods -n <ns> -w
kubectl get events -n <ns> -w
```

Always start with `describe pod`. Half of all kubectl tickets are answered by reading the bottom of `describe pod`.

The "two terminals" rule for incidents:

```
Terminal 1: kubectl get events -A --watch
Terminal 2: kubectl logs -l app=web -n <ns> -f --prefix
```

You'll see in real time which side speaks first when something breaks.

The "minify your kubeconfig" sanity check:

```bash
kubectl config view --minify
kubectl config get-contexts
kubectl config use-context <ctx>
kubectl config set-context --current --namespace=<ns>
```

The dry-run / server-side dry-run:

```bash
kubectl apply -f deploy.yaml --dry-run=client
kubectl apply -f deploy.yaml --dry-run=server
kubectl create deployment web --image=nginx --dry-run=client -o yaml > deploy.yaml
```

## Tips

- `--v=6` to `--v=9` increases kubectl verbosity (HTTP traces). `--v=8` is great for debugging API calls.

```bash
kubectl get pods -v=8
```

- `kubectl get -o wide` is almost always more useful than the default columns.
- `kubectl get -o yaml | less` over `describe` when you need exact field values.
- `kubectl get -o custom-columns=...` for ad-hoc tables:

```bash
kubectl get pods -o custom-columns='NAME:.metadata.name,NODE:.spec.nodeName,IP:.status.podIP,PHASE:.status.phase'
kubectl get pods -A -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,IMAGE:.spec.containers[*].image'
```

- `kubectl get -o jsonpath` for scripting:

```bash
kubectl get pods -n <ns> -l app=web -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.phase}{"\n"}{end}'
kubectl get nodes -o jsonpath='{.items[*].status.addresses[?(@.type=="InternalIP")].address}'
```

- `kubectl explain` is your in-terminal API reference:

```bash
kubectl explain pod.spec
kubectl explain pod.spec.containers.resources --recursive
```

- `--field-selector` for server-side filtering (not all fields are indexed):

```bash
kubectl get pods --field-selector status.phase=Failed -A
kubectl get pods --field-selector spec.nodeName=<node>
kubectl get events --field-selector reason=FailedScheduling
```

- Force-delete is destructive. Use only when:
  - The node is gone for good.
  - The workload won't get re-scheduled to a place it shouldn't be.

- `kubectl wait` for scripts:

```bash
kubectl wait --for=condition=Ready pod/<pod> -n <ns> --timeout=2m
kubectl wait --for=condition=Available deployment/<d> -n <ns> --timeout=5m
kubectl wait --for=delete pod/<pod> -n <ns> --timeout=1m
```

- Annotate the deployment with a change cause for `rollout history`:

```bash
kubectl annotate deployment <d> kubernetes.io/change-cause="bumped to v1.4.2 for CVE-X" -n <ns>
```

- Test a new manifest in a throwaway namespace:

```bash
kubectl create ns scratch
kubectl apply -f deploy.yaml -n scratch
kubectl delete ns scratch              # nukes everything
```

- The "big hammer" rollout reset (rollout restart) re-rolls every pod with the same spec. Useful after rotating a Secret/ConfigMap.

- Annotate Pods with debugging metadata so old logs in your aggregator are searchable:

```bash
kubectl annotate pod <pod> debug.acme.com/incident=INC-12345 -n <ns>
```

- Save commonly-used commands as kubectl plugins. Any executable on `PATH` named `kubectl-<x>` becomes `kubectl <x>`:

```bash
# /usr/local/bin/kubectl-events:
#!/bin/sh
kubectl get events --sort-by=.lastTimestamp "$@"
chmod +x /usr/local/bin/kubectl-events
kubectl events -n <ns>
```

## See Also

- kubectl
- kubernetes
- helm
- docker
- polyglot
- bash

## References

- Kubernetes Docs — Debug Running Pods: <https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/>
- Kubernetes Docs — Debug Cluster: <https://kubernetes.io/docs/tasks/debug/debug-cluster/>
- Kubernetes Docs — Determine the Reason for Pod Failure: <https://kubernetes.io/docs/tasks/debug/debug-application/determine-reason-pod-failure/>
- Kubernetes Docs — Application Introspection and Debugging: <https://kubernetes.io/docs/tasks/debug/debug-application/>
- Kubernetes Docs — kubectl Reference: <https://kubernetes.io/docs/reference/kubectl/>
- Kubernetes Docs — kubectl Cheat Sheet: <https://kubernetes.io/docs/reference/kubectl/cheatsheet/>
- Kubernetes Docs — kubectl debug: <https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/#ephemeral-container>
- Kubernetes Docs — Ephemeral Containers: <https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/>
- Kubernetes Docs — Pod Lifecycle: <https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/>
- Kubernetes Docs — Configure Liveness, Readiness, Startup Probes: <https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/>
- Kubernetes Docs — Network Policies: <https://kubernetes.io/docs/concepts/services-networking/network-policies/>
- Kubernetes Docs — Service: <https://kubernetes.io/docs/concepts/services-networking/service/>
- Kubernetes Docs — Ingress: <https://kubernetes.io/docs/concepts/services-networking/ingress/>
- Kubernetes Docs — RBAC Authorization: <https://kubernetes.io/docs/reference/access-authn-authz/rbac/>
- Kubernetes Docs — Persistent Volumes: <https://kubernetes.io/docs/concepts/storage/persistent-volumes/>
- Kubernetes Docs — Resource Management: <https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/>
- Kubernetes Docs — Events: <https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/>
- nicolaka/netshoot: <https://github.com/nicolaka/netshoot>
- kubectl-debug (krew): <https://github.com/aylei/kubectl-debug>
- stern: <https://github.com/stern/stern>
- krew: <https://krew.sigs.k8s.io/>
- kubectx + kubens: <https://github.com/ahmetb/kubectx>
- "Kubernetes Patterns" by Bilgin Ibryam and Roland Huss (O'Reilly, 2nd ed.) — chapters 11–12 on operational debugging.
- "Kubernetes in Action" by Marko Lukša (Manning, 2nd ed.) — chapter on troubleshooting.
