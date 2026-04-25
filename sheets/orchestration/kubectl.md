# kubectl (Kubernetes CLI)

The official command-line client for the Kubernetes API server — get, describe, apply, exec, logs, debug, RBAC, rollouts, port-forward, and the daily ops surface for every Kubernetes cluster.

## Setup

```bash
# Install on macOS
brew install kubectl
brew install kubernetes-cli   # alias

# Install on Linux (latest stable)
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install via apt (Debian/Ubuntu)
sudo apt-get update && sudo apt-get install -y apt-transport-https ca-certificates curl
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update && sudo apt-get install -y kubectl

# Version (1.28+ recommended; 1.30+ for sidecar containers stable)
kubectl version --client
kubectl version --client -o yaml
kubectl version --client -o json
kubectl version --output=yaml          # both client + server
kubectl version --short                # DEPRECATED 1.28+, use --output

# Skew policy: client may be +/- 1 minor from server
# kubectl 1.30 -> server 1.29-1.31 supported
```

### kubeconfig location

```bash
# Default location
~/.kube/config

# Override path
export KUBECONFIG=~/.kube/dev-cluster.yaml

# Merge multiple files (priority left-to-right)
export KUBECONFIG=~/.kube/dev:~/.kube/prod:~/.kube/staging
kubectl config view --merge=true --flatten         # produce merged file

# View effective config
kubectl config view
kubectl config view --minify                       # only current context
kubectl config view --minify --raw                 # include credentials (secret-bearing!)
kubectl config view -o jsonpath='{.users[*].name}'

# Where kubectl resolves config from (precedence)
# 1. --kubeconfig flag
# 2. KUBECONFIG env var (colon-separated list, merged)
# 3. ~/.kube/config
```

### krew plugin manager

```bash
# Install krew (the kubectl plugin manager)
(
  set -x; cd "$(mktemp -d)" &&
  OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
  KREW="krew-${OS}_${ARCH}" &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
  tar zxvf "${KREW}.tar.gz" &&
  ./"${KREW}" install krew
)

# Add to PATH
export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"

# Common productivity plugins
kubectl krew install ctx              # context switcher
kubectl krew install ns               # namespace switcher
kubectl krew install neat             # strip managed-fields/status from yaml
kubectl krew install tree             # owner-reference tree
kubectl krew install view-secret      # base64-decode secrets
kubectl krew install who-can          # RBAC reverse lookup
kubectl krew install slice            # split multi-doc yaml
kubectl krew install df-pv            # PV usage report
kubectl krew install get-all          # list all resources of all kinds
kubectl krew install access-matrix    # RBAC matrix
kubectl krew install debug-shell      # ephemeral debug pod
kubectl krew install trace            # bcc/bpftrace from kubectl
kubectl krew install resource-capacity
kubectl krew install rabbitmq         # if you run RMQ operator

# List installed
kubectl krew list

# Update plugin index + plugins
kubectl krew update
kubectl krew upgrade
```

## Cluster Connectivity

```bash
# Reach the API server
kubectl cluster-info
# Output:
# Kubernetes control plane is running at https://10.0.0.1:6443
# CoreDNS is running at https://10.0.0.1:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

# Diagnostic dump (verbose, do not paste — contains pod IPs, secrets metadata)
kubectl cluster-info dump
kubectl cluster-info dump --output-directory=/tmp/cluster-dump
kubectl cluster-info dump --namespaces=default,kube-system

# Version (server reachability check)
kubectl version --output=yaml
kubectl version --client                # offline; client only
kubectl version --output=json | jq '.serverVersion.gitVersion'
```

### Contexts

```bash
# List contexts
kubectl config get-contexts
# CURRENT  NAME           CLUSTER        AUTHINFO       NAMESPACE
# *        prod           prod-cluster   stevie         default
#          dev            dev-cluster    stevie         dev

# Show current
kubectl config current-context

# Switch
kubectl config use-context dev
kubectl config use-context prod

# Create / modify
kubectl config set-context my-ctx --cluster=prod-cluster --user=stevie --namespace=billing
kubectl config set-context --current --namespace=billing       # change ns of current ctx

# Rename / delete
kubectl config rename-context old-name new-name
kubectl config delete-context dev
kubectl config delete-cluster dev-cluster
kubectl config delete-user stevie

# Server URL / CA / token
kubectl config set-cluster prod --server=https://api.prod.example:6443 --certificate-authority=/path/ca.pem
kubectl config set-credentials stevie --token=eyJhbGc...
kubectl config set-credentials stevie --client-certificate=/path/cert.pem --client-key=/path/key.pem
```

## Contexts and Namespaces

```bash
# kubectx / kubens (krew) — fast switchers
kubectl ctx                       # list
kubectl ctx dev                   # switch
kubectl ctx -                     # previous
kubectl ctx -d old                # delete

kubectl ns                        # list namespaces
kubectl ns billing                # switch
kubectl ns -                      # previous

# Without plugins (built-in equivalents)
kubectl config use-context dev
kubectl config set-context --current --namespace=billing

# Multi-cluster ops pattern (do NOT alias k=kubectl prod permanently)
alias kdev='kubectl --context=dev'
alias kprod='kubectl --context=prod'

# Canonical alias the entire ecosystem expects
alias k=kubectl
source <(kubectl completion bash)
complete -F __start_kubectl k
# zsh:
source <(kubectl completion zsh)
```

### Namespace defaults

```bash
# Show current namespace of current context
kubectl config view --minify -o jsonpath='{..namespace}'

# Cluster-wide queries (any namespace)
kubectl get pods -A
kubectl get pods --all-namespaces
```

## Resource Discovery

```bash
# List every API resource the cluster supports
kubectl api-resources
# NAME            SHORTNAMES   APIVERSION   NAMESPACED   KIND
# pods            po           v1           true         Pod
# deployments     deploy       apps/v1      true         Deployment

# Filter
kubectl api-resources --namespaced=true        # only namespaced
kubectl api-resources --namespaced=false       # only cluster-scoped (Node, PV, ClusterRole)
kubectl api-resources --api-group=apps
kubectl api-resources -o wide                  # include verbs (get/list/watch/create/update/...)
kubectl api-resources --verbs=list,delete

# Versions
kubectl api-versions
# v1
# apps/v1
# networking.k8s.io/v1
# policy/v1

# Schema explain (the field reference without web search)
kubectl explain pod
kubectl explain pod.spec
kubectl explain pod.spec.containers
kubectl explain pod.spec.containers.resources
kubectl explain deployment.spec.strategy.rollingUpdate
kubectl explain --recursive pod.spec           # full tree
kubectl explain --api-version=apps/v1 deployment.spec
```

## Get / List Resources

```bash
# Basic gets
kubectl get pods
kubectl get pod my-pod
kubectl get po my-pod                          # short name
kubectl get pods,services                      # multiple kinds
kubectl get all                                # pods+services+deployments+rs+sts+ds+jobs+cronjobs (NOT cm/secret/ingress!)
kubectl get all -n my-ns

# Output formats
kubectl get pods -o wide                       # +Node, IP, NominatedNode
kubectl get pod my-pod -o yaml
kubectl get pod my-pod -o json
kubectl get pod my-pod -o name                 # pod/my-pod
kubectl get pods -o jsonpath='{.items[*].metadata.name}'
kubectl get pods -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}'
kubectl get pods -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,IP:.status.podIP
kubectl get pods -o custom-columns-file=cols.txt

# Label selectors
kubectl get pods -l app=nginx
kubectl get pods -l 'app in (nginx,apache)'
kubectl get pods -l 'app notin (nginx)'
kubectl get pods -l 'tier!=frontend'
kubectl get pods -l 'env=prod,tier=db'         # AND
kubectl get pods --selector=app=nginx          # long form
kubectl get pods --show-labels

# Field selectors (limited fields supported)
kubectl get pods --field-selector=status.phase=Running
kubectl get pods --field-selector=status.phase!=Running
kubectl get pods --field-selector=spec.nodeName=worker-1
kubectl get events --field-selector=type=Warning
kubectl get pods --field-selector=metadata.namespace!=kube-system

# Watch
kubectl get pods --watch                       # initial list + stream
kubectl get pods -w
kubectl get pods --watch-only                  # stream only, no initial

# Namespaces
kubectl get pods -n kube-system
kubectl get pods --all-namespaces
kubectl get pods -A

# Pagination on huge clusters (1.27+)
kubectl get pods --chunk-size=500 -A

# Sort
kubectl get pods --sort-by=.metadata.creationTimestamp
kubectl get pods --sort-by='.status.containerStatuses[0].restartCount'
kubectl get nodes --sort-by='.status.capacity.cpu'

# Show server resource for given client request
kubectl get pods -v=8                          # verbose; shows curl-equivalent
```

## Describe

```bash
# Describe is your first stop after get
kubectl describe pod my-pod
kubectl describe pod my-pod -n my-ns
kubectl describe deployment my-app
kubectl describe node worker-1

# What to look at on `describe pod`
# - Status: Running, Pending, Succeeded, Failed, Unknown
# - Containers[].State: Running, Waiting (Reason), Terminated (Reason+ExitCode)
# - Containers[].LastState: previous Terminated reason (= why it crashed)
# - Conditions: PodScheduled, Initialized, ContainersReady, Ready
# - Events: bottom of describe — what just happened (FailedScheduling, Pulled, Started, Killing, Unhealthy)

# What to look at on `describe node`
kubectl describe node worker-1
# - Conditions: Ready, MemoryPressure, DiskPressure, PIDPressure, NetworkUnavailable
# - Capacity:    raw machine
# - Allocatable: capacity minus kubelet reserves
# - Non-terminated Pods: who is consuming CPU/mem requests
# - Allocated resources: requests/limits totals + percentage of allocatable
# - Events: kubelet/node events

# By label / by name
kubectl describe pods -l app=nginx
kubectl describe deployment/my-app
```

## Logs

```bash
# Single-container pod
kubectl logs my-pod
kubectl logs my-pod -n my-ns

# Multi-container pod (must specify -c)
kubectl logs my-pod -c sidecar
kubectl logs my-pod --all-containers
kubectl logs my-pod --all-containers --prefix    # prepends [pod/container]

# Crashed container — see the LAST run's logs (the killer feature)
kubectl logs my-pod --previous
kubectl logs my-pod -p
kubectl logs my-pod -c app -p

# Follow (tail -f equivalent)
kubectl logs my-pod -f
kubectl logs my-pod --follow --tail=100
kubectl logs my-pod -f --since=10m
kubectl logs my-pod --since-time=2024-01-15T00:00:00Z

# Tail / since
kubectl logs my-pod --tail=50
kubectl logs my-pod --since=1h
kubectl logs my-pod --timestamps

# Aggregated logs across pods (great for deployments)
kubectl logs -l app=nginx --all-containers --prefix --tail=100
kubectl logs -l app=nginx -f --max-log-requests=10        # default is 5
kubectl logs deployment/my-app                            # logs from one pod of the deployment
kubectl logs deployment/my-app --all-pods=true            # 1.30+
kubectl logs statefulset/my-sts -c app
kubectl logs job/my-job

# Logs to file
kubectl logs my-pod > /tmp/my-pod.log
kubectl logs my-pod --previous > /tmp/my-pod-prev.log

# A pod whose logs are GC'd (kubelet rotated) → use --previous, or persist via Loki/EFK/CloudWatch
```

## Exec / Run / Attach

```bash
# Exec a command (single-container)
kubectl exec my-pod -- ls -la /app
kubectl exec my-pod -- env
kubectl exec my-pod -- cat /etc/hostname

# Multi-container pod — must -c
kubectl exec my-pod -c app -- ps aux

# Interactive shell (the daily debug move)
kubectl exec -it my-pod -- /bin/sh
kubectl exec -it my-pod -- /bin/bash
kubectl exec -it my-pod -c sidecar -- /bin/sh

# Exec into a Deployment-managed pod
kubectl exec -it deployment/my-app -- /bin/sh
kubectl exec -it sts/my-sts -- /bin/sh
kubectl exec -it daemonset/node-exporter -- /bin/sh        # picks one pod

# Pipe stdin
kubectl exec -i my-pod -- tar xvf - < bundle.tar
kubectl exec my-pod -- sh -c 'cat > /tmp/x' < /local/x

# Attach to a running container's stdin/stdout (vs exec which spawns a new process)
kubectl attach my-pod -i -t
kubectl attach my-pod -c app -i -t

# Run a one-off pod (debug / quick test)
kubectl run debug --rm -it --image=alpine -- sh
kubectl run nginx --image=nginx --port=80
kubectl run dnsutils --rm -it --image=registry.k8s.io/e2e-test-images/jessie-dnsutils:1.7 -- bash
kubectl run busybox --rm -it --image=busybox:1.36 --restart=Never --image-pull-policy=IfNotPresent -- sh
kubectl run curl --rm -it --image=curlimages/curl -- curl -v http://my-svc

# Run flags
# --rm                 delete pod when command exits
# -it                  interactive + tty (combined as one flag in shells)
# --image=IMG          required
# --restart=Never      pod (default Always = creates a Deployment, deprecated in newer kubectl)
# --image-pull-policy=Always|IfNotPresent|Never
# --command            treat extra args as the command (override image ENTRYPOINT)
# --env=KEY=VAL        env var
# -- ARG ARG           args after --

# NEVER use `kubectl run` to launch production workloads — it bypasses GitOps/manifests.
```

## Port-Forward

```bash
# Forward localhost:LOCAL → pod:REMOTE
kubectl port-forward pod/my-pod 8080:80
# Forwarding from 127.0.0.1:8080 -> 80
# Forwarding from [::1]:8080 -> 80

# Forward via service (load-balances to one of the matching pods)
kubectl port-forward service/my-svc 8080:80
kubectl port-forward svc/my-svc 8080:80

# Forward via deployment / sts (kubectl picks one matching pod)
kubectl port-forward deployment/my-app 8080:80
kubectl port-forward sts/my-sts 8080:80

# Multiple ports
kubectl port-forward pod/my-pod 8080:80 9090:9090

# Random local port
kubectl port-forward pod/my-pod :80
# Forwarding from 127.0.0.1:54321 -> 80

# Bind on all interfaces (DANGER — exposes to LAN)
kubectl port-forward --address=0.0.0.0 pod/my-pod 8080:80

# Background it
kubectl port-forward pod/my-pod 8080:80 &
disown

# Why your tunnel keeps dying:
# - The pod was rescheduled (new pod = need to re-target)
# - kube-apiserver dropped the streaming connection
# - Local network slept
# Fix: wrap in `while true; do kubectl port-forward ...; sleep 1; done`
```

## Copy

```bash
# Copy local → pod
kubectl cp ./file.txt my-pod:/tmp/file.txt
kubectl cp ./file.txt my-pod:/tmp/file.txt -c app
kubectl cp ./dir my-pod:/tmp/dir -c app

# Copy pod → local
kubectl cp my-pod:/tmp/file.txt ./file.txt
kubectl cp my-pod:/var/log/app.log ./app.log -c app

# Across namespaces
kubectl cp my-ns/my-pod:/tmp/file.txt ./file.txt

# Limitations (read these or be confused later)
# - Requires `tar` binary in the container (distroless has none → kubectl cp will fail)
# - Requires read access on the source path
# - Does not preserve ownership; root in pod → user on host
# - Symlinks: copied as files, not links
# - For distroless, exec a debug ephemeral container with shell + tar, then cp
```

## Top

```bash
# Top-style live view (requires metrics-server installed in cluster)
kubectl top pods
kubectl top pods -A
kubectl top pods --containers                  # per-container
kubectl top pods -l app=nginx
kubectl top pods --sort-by=cpu
kubectl top pods --sort-by=memory

# Nodes
kubectl top nodes
kubectl top nodes --sort-by=cpu

# If you see: "error: Metrics API not available"
# → metrics-server is not installed:
#   kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
#
# vs `kubectl describe node` — describe shows requests/limits sums (capacity planning),
# top shows live usage from cAdvisor (right now).
```

## Apply / Create / Replace / Edit / Delete / Patch

```bash
# Apply (declarative — the canonical way)
kubectl apply -f deploy.yaml
kubectl apply -f manifests/                    # apply all .yaml/.json in dir
kubectl apply -R -f manifests/                 # recurse subdirs
kubectl apply -f https://raw.githubusercontent.com/.../app.yaml
kubectl apply -k overlays/prod                 # kustomize
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-cm
data:
  key: value
EOF

# Server-side apply (the modern path; field managers + conflict detection)
kubectl apply -f deploy.yaml --server-side
kubectl apply -f deploy.yaml --server-side --force-conflicts

# Apply with declarative pruning
kubectl apply -f manifests/ --prune --selector=app=my-app
# Removes resources NOT in the file but matching selector

# Dry runs
kubectl apply -f deploy.yaml --dry-run=client          # local only
kubectl apply -f deploy.yaml --dry-run=server          # send to API, validate (admission webhooks fire)
kubectl apply -f deploy.yaml --dry-run=server -o yaml  # see what would be applied

# Create (imperative)
kubectl create -f deploy.yaml
kubectl create deployment nginx --image=nginx:1.27 --replicas=3 --port=80
kubectl create deployment nginx --image=nginx --dry-run=client -o yaml > deploy.yaml
kubectl create namespace billing
kubectl create configmap my-cm --from-literal=key=value
kubectl create configmap my-cm --from-file=app.conf=./app.conf
kubectl create configmap my-cm --from-env-file=app.env
kubectl create secret generic my-secret --from-literal=password=hunter2
kubectl create secret generic my-secret --from-file=ca.pem
kubectl create secret docker-registry regcred --docker-server=ghcr.io --docker-username=stevie --docker-password=ghp_xxx --docker-email=stevie@bellis.tech
kubectl create secret tls my-tls --cert=cert.pem --key=key.pem
kubectl create role pod-reader --verb=get,list,watch --resource=pods
kubectl create rolebinding read-pods --role=pod-reader --user=alice
kubectl create clusterrole all-pods --verb='*' --resource=pods
kubectl create clusterrolebinding alice-admin --clusterrole=cluster-admin --user=alice
kubectl create serviceaccount my-sa
kubectl create job my-job --image=busybox -- echo hello
kubectl create job manual-run --from=cronjob/nightly-backup       # one-shot trigger of a cronjob

# Replace (delete + recreate; fails if the object is missing)
kubectl replace -f deploy.yaml
kubectl replace --force -f deploy.yaml         # force recreate (delete then create)

# Edit (opens $EDITOR — saves diff via apply)
kubectl edit deployment my-app
kubectl edit configmap my-cm -n my-ns
KUBE_EDITOR=vim kubectl edit deployment my-app

# Delete
kubectl delete -f deploy.yaml
kubectl delete pod my-pod
kubectl delete pods -l app=nginx
kubectl delete pods --all -n test
kubectl delete pod my-pod --grace-period=0 --force         # last resort, leaks if kubelet down
kubectl delete pod my-pod --wait=false                     # return before fully gone
kubectl delete deployment my-app --cascade=foreground      # block until owned ReplicaSet/Pods deleted
kubectl delete deployment my-app --cascade=background      # default
kubectl delete deployment my-app --cascade=orphan          # leave orphaned ReplicaSet/Pods

# Patch
kubectl patch pod my-pod -p '{"spec":{"containers":[{"name":"app","image":"nginx:1.28"}]}}'
kubectl patch deployment my-app -p '{"spec":{"replicas":5}}'
kubectl patch deployment my-app --type=json -p '[{"op":"replace","path":"/spec/replicas","value":5}]'
kubectl patch deployment my-app --type=merge -p '{"spec":{"template":{"metadata":{"labels":{"version":"v2"}}}}}'
kubectl patch deployment my-app --type=strategic -p '{"spec":{"template":{"spec":{"containers":[{"name":"app","image":"nginx:1.28"}]}}}}'
# Patch types:
#   strategic (default; understands list-merge keys; pod containers merge by name)
#   merge     (RFC 7396; replaces lists wholesale)
#   json      (RFC 6902; ops: add/remove/replace/copy/move/test)
```

## Apply Patterns

```bash
# Last-applied-configuration annotation
# - Client-side apply stores last-applied JSON in metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"]
# - Three-way diff: live -> last-applied -> desired
# - Server-side apply REPLACES this with field managers (no annotation needed)

# Server-side apply: declarative ownership of fields
kubectl apply -f deploy.yaml --server-side --field-manager=stevie
kubectl apply -f deploy.yaml --server-side --force-conflicts
# Conflict example:
#   error: Apply failed with 1 conflict: conflict with "kubectl-edit" using apps/v1: .spec.replicas
# Means: another field manager owns that field. --force-conflicts to take ownership.

# Inspect ownership
kubectl get deployment my-app -o yaml | grep -A1 managedFields

# Pruning (dangerous; always test first)
kubectl apply -f manifests/ --prune --selector=app=my-app --dry-run=server
kubectl apply -f manifests/ --prune --selector=app=my-app

# Validate before apply
kubectl apply -f deploy.yaml --dry-run=server -o yaml
kubectl diff -f deploy.yaml
```

## Output Formats

```bash
# -o options
# wide              extra columns (Node, IP, etc)
# yaml              full yaml
# json              full json
# name              "kind/name" only
# jsonpath=EXPR     custom jsonpath
# jsonpath-file=F
# go-template=EXPR  text/template
# go-template-file=F
# custom-columns=SPEC
# custom-columns-file=F

# JSONPath cookbook
kubectl get pods -o jsonpath='{.items[*].metadata.name}'
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\n"}{end}'
kubectl get pods -o jsonpath='{.items[*].spec.containers[*].image}' | tr -s '[:space:]' '\n' | sort -u
kubectl get pods -o jsonpath='{.items[?(@.status.phase=="Running")].metadata.name}'
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[*].restartCount}{"\n"}{end}'
kubectl get nodes -o jsonpath='{.items[*].status.addresses[?(@.type=="InternalIP")].address}'
kubectl get svc my-svc -o jsonpath='{.spec.clusterIP}'
kubectl get secret my-secret -o jsonpath='{.data.password}' | base64 -d
kubectl get cm my-cm -o jsonpath='{.data.config\.yaml}'              # escape dots in keys
kubectl get pvc -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.resources.requests.storage}{"\n"}{end}'

# go-template (more powerful for conditionals)
kubectl get pods -o go-template='{{range .items}}{{if eq .status.phase "Running"}}{{.metadata.name}}{{"\n"}}{{end}}{{end}}'

# custom-columns
kubectl get pods -o custom-columns='NAME:.metadata.name,STATUS:.status.phase,NODE:.spec.nodeName,IP:.status.podIP'
kubectl get pods -o custom-columns='NAME:.metadata.name,IMAGES:.spec.containers[*].image'
kubectl get pods -o custom-columns='NAME:.metadata.name,RESTARTS:.status.containerStatuses[0].restartCount'
kubectl get nodes -o custom-columns='NAME:.metadata.name,CPU:.status.allocatable.cpu,MEM:.status.allocatable.memory'

# Replace 90% of jq usage
kubectl get pods -o json | jq -r '.items[] | "\(.metadata.name)\t\(.status.phase)"'   # equivalent
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\n"}{end}'
```

## Resource Manifests Catalog

### Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
  namespace: default
  labels:
    app: web
  annotations:
    kubectl.kubernetes.io/default-container: app
spec:
  serviceAccountName: web-sa
  imagePullSecrets:
    - name: regcred
  initContainers:                                # run-to-completion before app containers
    - name: init-db
      image: busybox:1.36
      command: ['sh', '-c', 'until nslookup db; do sleep 2; done']
  containers:
    - name: app
      image: ghcr.io/me/web:1.2.3@sha256:abc...
      imagePullPolicy: IfNotPresent              # Always | Never | IfNotPresent
      command: ['/app/server']                   # overrides ENTRYPOINT
      args: ['--port=8080']                      # overrides CMD
      env:
        - name: DB_HOST
          value: postgres.svc.cluster.local
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: DB_PASS
          valueFrom:
            secretKeyRef:
              name: db
              key: password
      envFrom:
        - configMapRef: { name: app-config }
        - secretRef:    { name: app-secrets }
      ports:
        - name: http
          containerPort: 8080
          protocol: TCP
      resources:
        requests: { cpu: 100m, memory: 128Mi }
        limits:   { cpu: 500m, memory: 512Mi }
      volumeMounts:
        - name: data
          mountPath: /var/lib/app
        - name: config
          mountPath: /etc/app
          readOnly: true
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        readOnlyRootFilesystem: true
        allowPrivilegeEscalation: false
        capabilities: { drop: ['ALL'] }
        seccompProfile: { type: RuntimeDefault }
      livenessProbe:                             # restart container on fail
        httpGet: { path: /healthz, port: 8080 }
        initialDelaySeconds: 10
        periodSeconds: 10
        timeoutSeconds: 1
        failureThreshold: 3
      readinessProbe:                            # remove from Service endpoints on fail
        httpGet: { path: /ready, port: 8080 }
        periodSeconds: 5
      startupProbe:                              # disable liveness until startup probe passes
        httpGet: { path: /healthz, port: 8080 }
        failureThreshold: 30
        periodSeconds: 10
      lifecycle:
        preStop:
          exec: { command: ['/bin/sh', '-c', 'sleep 10; /app/drain'] }
  restartPolicy: Always                          # Always | OnFailure | Never (Job uses OnFailure/Never)
  terminationGracePeriodSeconds: 30
  volumes:
    - name: data
      persistentVolumeClaim: { claimName: web-data }
    - name: config
      configMap: { name: app-config }
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchLabels: { app: web }
            topologyKey: kubernetes.io/hostname
  tolerations:
    - key: dedicated
      operator: Equal
      value: web
      effect: NoSchedule
  nodeSelector:
    disktype: ssd
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: DoNotSchedule
      labelSelector:
        matchLabels: { app: web }
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 3
  revisionHistoryLimit: 10
  progressDeadlineSeconds: 600
  selector:
    matchLabels: { app: web }
  strategy:
    type: RollingUpdate                          # RollingUpdate | Recreate
    rollingUpdate:
      maxSurge: 25%                              # extra pods above replicas during rollout
      maxUnavailable: 25%                        # how many can be down during rollout
  template:
    metadata:
      labels: { app: web }
    spec:
      containers:
        - name: app
          image: ghcr.io/me/web:1.2.3
```

### StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: db
spec:
  serviceName: db                               # MUST exist as a headless Service (clusterIP: None)
  replicas: 3
  podManagementPolicy: OrderedReady             # OrderedReady (default) | Parallel
  updateStrategy:
    type: RollingUpdate
    rollingUpdate: { partition: 0 }             # canary: bump partition to limit which ordinals update
  selector:
    matchLabels: { app: db }
  template:
    metadata:
      labels: { app: db }
    spec:
      containers:
        - name: db
          image: postgres:16
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:                         # one PVC per pod (db-0, db-1, db-2)
    - metadata: { name: data }
      spec:
        accessModes: [ReadWriteOnce]
        storageClassName: standard
        resources: { requests: { storage: 20Gi } }
```

### DaemonSet

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
spec:
  selector:
    matchLabels: { app: node-exporter }
  template:
    metadata:
      labels: { app: node-exporter }
    spec:
      tolerations:
        - operator: Exists                      # tolerate ALL taints (run on every node)
      hostNetwork: true
      hostPID: true
      containers:
        - name: node-exporter
          image: prom/node-exporter:latest
          ports: [{ containerPort: 9100 }]
```

### Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
spec:
  parallelism: 1                                # how many pods run concurrently
  completions: 1                                # how many successful pods needed
  backoffLimit: 4                               # retries before marking Failed
  activeDeadlineSeconds: 600                    # kill after N seconds total
  ttlSecondsAfterFinished: 3600                 # auto-delete completed/failed after N seconds
  template:
    spec:
      restartPolicy: OnFailure                  # Job pods cannot be Always
      containers:
        - name: migrate
          image: my/migrator:1.0
          command: ['/migrate', '--up']
```

### CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly-backup
spec:
  schedule: '0 2 * * *'                         # 02:00 every night (cluster TZ; UTC by default unless timeZone set)
  timeZone: 'America/Los_Angeles'               # 1.27+ stable
  concurrencyPolicy: Forbid                     # Allow | Forbid | Replace
  startingDeadlineSeconds: 60
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1
  suspend: false
  jobTemplate:
    spec:
      backoffLimit: 2
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: backup
              image: my/backup:1.0
              args: ['--target=s3://bucket/']
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  type: ClusterIP                               # ClusterIP (default) | NodePort | LoadBalancer | ExternalName
  selector: { app: web }
  ports:
    - name: http
      port: 80                                  # Service port
      targetPort: 8080                          # Container port
      protocol: TCP
  sessionAffinity: None                         # None | ClientIP
  sessionAffinityConfig:
    clientIP: { timeoutSeconds: 10800 }
  externalTrafficPolicy: Cluster                # Cluster (default) | Local (preserves source IP, may drop if no local pod)
  internalTrafficPolicy: Cluster                # Cluster | Local

# Headless (no clusterIP; DNS round-robin to pod IPs; needed for StatefulSet)
---
apiVersion: v1
kind: Service
metadata: { name: db }
spec:
  clusterIP: None
  selector: { app: db }
  ports: [{ port: 5432 }]

# NodePort
---
apiVersion: v1
kind: Service
metadata: { name: web-np }
spec:
  type: NodePort
  selector: { app: web }
  ports:
    - port: 80
      targetPort: 8080
      nodePort: 30080                           # 30000-32767

# ExternalName (DNS CNAME, no proxy)
---
apiVersion: v1
kind: Service
metadata: { name: external-db }
spec:
  type: ExternalName
  externalName: db.prod.example.com
```

### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: web
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  tls:
    - hosts: [web.example.com]
      secretName: web-tls
  rules:
    - host: web.example.com
      http:
        paths:
          - path: /api
            pathType: Prefix                    # Prefix | Exact | ImplementationSpecific
            backend:
              service:
                name: web
                port: { number: 80 }
          - path: /static
            pathType: Prefix
            backend:
              service:
                name: static
                port: { number: 80 }
```

### Gateway API (newer, replacing Ingress for advanced routing)

```yaml
# Hint: Gateway API CRDs (gateway.networking.k8s.io)
# - Gateway       (a load balancer / listener config)
# - GatewayClass  (defines the implementation; like IngressClass)
# - HTTPRoute     (routing rules; multiple per Gateway)
# - TCPRoute / UDPRoute / TLSRoute / GRPCRoute
# - ReferenceGrant (cross-namespace permission for backendRefs)
# Better than Ingress for: header/method matching, traffic splitting, mirroring, cross-namespace.
# See sheets/orchestration/gateway-api.md
```

### ConfigMap and Secret

```yaml
apiVersion: v1
kind: ConfigMap
metadata: { name: app-config }
immutable: true                                 # 1.21+ stable; locks data, perf win
data:
  app.conf: |
    log_level=info
    workers=4
  feature_flags: '{"new_ui": true}'
binaryData:
  cert.bin: <base64>

---
apiVersion: v1
kind: Secret
metadata: { name: app-secrets }
type: Opaque                                    # Opaque | kubernetes.io/dockerconfigjson | kubernetes.io/tls | kubernetes.io/service-account-token | bootstrap.kubernetes.io/token
data:
  password: aHVudGVyMg==                        # base64 of 'hunter2'
stringData:                                     # plaintext at write time, kubectl base64-encodes for you
  api_token: 'plain text token'
immutable: true
```

### PersistentVolume / PersistentVolumeClaim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata: { name: web-data }
spec:
  storageClassName: standard
  accessModes: [ReadWriteOnce]                  # RWO | ROX | RWX | RWOP (Single-Pod, 1.27+ stable)
  volumeMode: Filesystem                        # Filesystem | Block
  resources:
    requests: { storage: 10Gi }

---
apiVersion: v1
kind: PersistentVolume
metadata: { name: pv-001 }
spec:
  capacity: { storage: 10Gi }
  accessModes: [ReadWriteOnce]
  persistentVolumeReclaimPolicy: Retain         # Delete | Retain | Recycle (deprecated)
  storageClassName: manual
  hostPath: { path: /mnt/data }                 # demo only — never in prod
```

### StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata: { name: fast-ssd }
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: '3000'
  throughput: '125'
reclaimPolicy: Delete                           # Delete | Retain
volumeBindingMode: WaitForFirstConsumer         # Immediate | WaitForFirstConsumer
allowVolumeExpansion: true
```

### NetworkPolicy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: { name: deny-all-ingress }
spec:
  podSelector: {}                               # all pods in namespace
  policyTypes: [Ingress]
  # no ingress rules => deny all

---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata: { name: web-allow }
spec:
  podSelector: { matchLabels: { app: web } }
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - podSelector: { matchLabels: { role: frontend } }
        - namespaceSelector: { matchLabels: { team: frontend } }
        - ipBlock:
            cidr: 10.0.0.0/8
            except: [10.0.5.0/24]
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - podSelector: { matchLabels: { app: db } }
      ports: [{ protocol: TCP, port: 5432 }]
    - to:
        - namespaceSelector: { matchLabels: { kubernetes.io/metadata.name: kube-system } }
          podSelector: { matchLabels: { k8s-app: kube-dns } }
      ports:
        - { protocol: UDP, port: 53 }
        - { protocol: TCP, port: 53 }
```

### RBAC: ServiceAccount, Role, ClusterRole, Bindings

```yaml
apiVersion: v1
kind: ServiceAccount
metadata: { name: web-sa }
automountServiceAccountToken: true

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: { name: pod-reader, namespace: default }
rules:
  - apiGroups: ['']
    resources: [pods, pods/log]
    verbs: [get, list, watch]
  - apiGroups: [apps]
    resources: [deployments]
    resourceNames: [my-app]
    verbs: [get]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata: { name: read-pods }
subjects:
  - kind: User
    name: alice
    apiGroup: rbac.authorization.k8s.io
  - kind: ServiceAccount
    name: web-sa
    namespace: default
  - kind: Group
    name: system:serviceaccounts:dev
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata: { name: aggregated-monitoring }
aggregationRule:
  clusterRoleSelectors:
    - matchLabels: { rbac.example.com/aggregate-to-monitoring: 'true' }
rules: []                                       # populated by aggregation
```

### PodDisruptionBudget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata: { name: web-pdb }
spec:
  minAvailable: 2                               # OR maxUnavailable: 1 (NOT both)
  selector:
    matchLabels: { app: web }
  unhealthyPodEvictionPolicy: AlwaysAllow       # 1.27+, lets evictions free stuck pods
```

### HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata: { name: web }
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: web
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target: { type: Utilization, averageUtilization: 80 }
    - type: Resource
      resource:
        name: memory
        target: { type: AverageValue, averageValue: 500Mi }
    - type: Pods
      pods:
        metric: { name: requests_per_second }
        target: { type: AverageValue, averageValue: 1000 }
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - { type: Percent, value: 50, periodSeconds: 60 }
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - { type: Percent, value: 100, periodSeconds: 15 }
        - { type: Pods,    value: 4,   periodSeconds: 15 }
      selectPolicy: Max
```

### VerticalPodAutoscaler hint

```yaml
# VPA is a separate project (autoscaling.k8s.io) — install the operator + CRDs:
# https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler
# updateMode: Off | Initial | Recreate | Auto
# DO NOT use VPA + HPA on CPU/mem at the same time (oscillates) unless HPA uses custom metrics.
```

### LimitRange and ResourceQuota

```yaml
apiVersion: v1
kind: LimitRange
metadata: { name: defaults }
spec:
  limits:
    - type: Container
      default: { cpu: 500m, memory: 512Mi }     # implicit limit if not set
      defaultRequest: { cpu: 100m, memory: 128Mi }
      max: { cpu: '2', memory: 4Gi }
      min: { cpu: 10m, memory: 16Mi }
    - type: Pod
      max: { cpu: '4', memory: 8Gi }
    - type: PersistentVolumeClaim
      max: { storage: 100Gi }
      min: { storage: 1Gi }

---
apiVersion: v1
kind: ResourceQuota
metadata: { name: ns-quota }
spec:
  hard:
    requests.cpu: '20'
    requests.memory: 40Gi
    limits.cpu: '40'
    limits.memory: 80Gi
    persistentvolumeclaims: '10'
    requests.storage: 500Gi
    pods: '100'
    services: '20'
    services.loadbalancers: '2'
    count/deployments.apps: '50'
    count/jobs.batch: '20'
```

## Selectors and Labels

```bash
# Recommended labels (from kubernetes.io/docs/concepts/overview/working-with-objects/common-labels)
# app.kubernetes.io/name        web
# app.kubernetes.io/instance    web-prod-7
# app.kubernetes.io/version     1.2.3
# app.kubernetes.io/component   frontend
# app.kubernetes.io/part-of     storefront
# app.kubernetes.io/managed-by  helm | kustomize | argocd

# Read labels
kubectl get pods --show-labels
kubectl get pods -L app -L tier
kubectl get pods -l app=nginx
kubectl get pods -l '!canary'
kubectl get pods -l 'env in (prod,staging)'
kubectl get pods -l 'release notin (canary)'

# Add / change labels
kubectl label pod my-pod env=prod
kubectl label pod my-pod env=staging --overwrite
kubectl label pods --all canary=false
kubectl label pod my-pod env-                                # remove label

# Annotations (free-form metadata, NOT for selection)
kubectl annotate pod my-pod owner=stevie@bellis.tech
kubectl annotate pod my-pod owner-                           # remove
kubectl annotate pod my-pod kubernetes.io/change-cause='deploy 1.2.3'
```

## JSONPath and go-template Cookbook

```bash
# Pod -> images
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.containers[*]}{.image}{","}{end}{"\n"}{end}'

# Pod -> phase only
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.phase}{"\n"}{end}'

# Restarts per container
kubectl get pods -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[*].restartCount}{"\n"}{end}'

# Nodes -> internal IPs
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.addresses[?(@.type=="InternalIP")].address}{"\n"}{end}'

# Services -> ClusterIP
kubectl get svc -A -o jsonpath='{range .items[*]}{.metadata.namespace}/{.metadata.name}{"\t"}{.spec.clusterIP}{"\n"}{end}'

# Decode a Secret value
kubectl get secret db -o jsonpath='{.data.password}' | base64 -d ; echo

# go-template with conditional
kubectl get pods -o go-template='{{range .items}}{{if ne .status.phase "Running"}}{{.metadata.name}} {{.status.phase}}{{"\n"}}{{end}}{{end}}'

# Find pods on a specific node
kubectl get pods -A --field-selector=spec.nodeName=worker-3
```

## Rollout

```bash
kubectl rollout status deployment/web                        # blocks until rolled out
kubectl rollout status deployment/web --timeout=2m
kubectl rollout status statefulset/db
kubectl rollout status daemonset/node-exporter

kubectl rollout history deployment/web
kubectl rollout history deployment/web --revision=4

kubectl rollout undo deployment/web                          # to previous revision
kubectl rollout undo deployment/web --to-revision=3

kubectl rollout restart deployment/web                       # 1.15+; bumps an annotation, rolls pods in-place
kubectl rollout restart statefulset/db
kubectl rollout restart daemonset/node-exporter

kubectl rollout pause deployment/web                         # stop rolling out further changes
kubectl rollout resume deployment/web
```

### Recording change-cause

```bash
# --record is DEPRECATED 1.19+. Annotate instead:
kubectl annotate deployment/web kubernetes.io/change-cause="bump to 1.2.4 (#1234)"
# Now `kubectl rollout history deployment/web` shows your message.
```

## Scale and Autoscale

```bash
# Manual scale
kubectl scale deployment/web --replicas=5
kubectl scale deployment/web --replicas=0                    # full shutdown
kubectl scale --replicas=3 -f deploy.yaml
kubectl scale deployment/web --current-replicas=3 --replicas=5    # CAS-style

# Autoscale (creates HPA imperatively)
kubectl autoscale deployment/web --min=2 --max=10 --cpu-percent=80

# Inspect HPA
kubectl get hpa
kubectl describe hpa web
# Look at: "current/target utilization", "ScalingActive", events with "FailedGetResourceMetric"
```

## Debug Recipes

```bash
# 1) Ephemeral debug container (1.25+ stable) — attach a sidecar shell to a running pod
kubectl debug -it my-pod --image=busybox:1.36 --target=app
kubectl debug -it my-pod --image=nicolaka/netshoot --share-processes --copy-to=my-pod-debug
# --target=app          shares process namespace with container 'app'
# --share-processes     used with --copy-to; lets you `ps` the original
# --copy-to=NEW         creates a copy of the pod with debug container injected

# 2) Debug a node (creates a privileged pod on the node, mounts /host)
kubectl debug node/worker-1 -it --image=ubuntu:22.04
# inside: chroot /host    # full host access (DANGEROUS, prod-banned by many orgs)

# 3) Debug a CrashLoopBackOff pod (you can't exec — it's not running)
kubectl logs my-pod --previous
kubectl describe pod my-pod                                   # State.Waiting.Reason / Last State.Terminated.ExitCode
kubectl debug -it my-pod --image=busybox --copy-to=debug --share-processes

# 4) ImagePullBackOff
kubectl describe pod my-pod | grep -A5 Events                 # see the registry error
kubectl get secrets                                            # is the imagePullSecret present?
kubectl get sa default -o yaml                                 # is it referenced in serviceaccount?

# 5) OOMKilled
kubectl get pod my-pod -o jsonpath='{.status.containerStatuses[*].lastState.terminated.exitCode}'
# 137 means SIGKILL (OOM or kubelet eviction)
kubectl describe pod my-pod | grep -A2 'Last State'
# Reason: OOMKilled
# Fix: bump resources.limits.memory OR fix the leak

# Flow:
# OOMKilled       -> bump memory limit, profile heap
# CrashLoopBackOff-> logs --previous; describe; check command/args; livenessProbe too aggressive
# ImagePullBackOff-> imagePullSecret, registry auth, image typo
# CreateContainerConfigError -> missing/invalid configMap or secret reference
# RunContainerError -> command not found in image, permissions, missing volume mount
```

## Pod States and Their Meanings

```text
Phase = Pending | Running | Succeeded | Failed | Unknown   (high-level)

Container State (the useful one):
  Waiting   { Reason: ContainerCreating | ImagePullBackOff | ErrImagePull
                     | CreateContainerConfigError | CrashLoopBackOff | InvalidImageName }
  Running   { StartedAt: ts }
  Terminated{ ExitCode: int, Reason: Completed | Error | OOMKilled, Signal: int }
```

Diagnostic recipes:

| Reason / status         | What it means                                        | First check                                                                  |
| ----------------------- | ---------------------------------------------------- | ---------------------------------------------------------------------------- |
| Pending                 | Not yet scheduled                                    | `kubectl describe pod` events: FailedScheduling, taints/tolerations, nodeSelector, requests vs allocatable |
| ContainerCreating       | Scheduled; pulling image / mounting volume / running init | events; `describe`; `kubectl logs --container=<init>`                       |
| Init:Error              | An initContainer exited non-zero                     | `kubectl logs my-pod -c init-name --previous`                                 |
| Init:CrashLoopBackOff   | initContainer keeps crashing                         | logs --previous; fix init logic                                              |
| Running                 | Containers started; not always ready                 | `kubectl get pod -o wide` READY column; readiness probe                       |
| CrashLoopBackOff        | Main container exits, restarts, exits...             | `kubectl logs --previous`; exit code; livenessProbe too aggressive            |
| ImagePullBackOff        | Cannot pull (auth/typo/network) — backoff between attempts | events; imagePullSecrets; registry connectivity                          |
| ErrImagePull            | Pull failed; will retry                              | as above                                                                     |
| CreateContainerConfigError | ConfigMap or Secret reference missing             | events show which name; create or fix reference                              |
| OOMKilled               | Cgroup OOM-killed the container (exit 137)           | `describe` last state; bump memory limit; fix leak                           |
| Error                   | Container exited non-zero                            | `logs --previous`; exit code                                                 |
| Completed               | Job/CronJob exited 0                                 | nothing to do; check ttlSecondsAfterFinished                                 |
| Terminating             | Pod marked for deletion; preStop + grace period      | stuck? `kubectl delete --grace-period=0 --force`                              |
| Unknown                 | kubelet unreachable; node lost                       | `kubectl get node`; node Ready=False                                          |
| Evicted                 | Node memory/disk pressure killed pod                 | `kubectl get events --field-selector=reason=Evicted`; reduce requests or scale |

```bash
# Why was my pod evicted?
kubectl get events -A --field-selector=reason=Evicted
# Pods evicted by kubelet have status.reason="Evicted" + status.message
kubectl get pods -A -o jsonpath='{range .items[?(@.status.reason=="Evicted")]}{.metadata.namespace}/{.metadata.name}{"\n"}{end}'

# The "5 minute toleration" — by default, kubelet adds these tolerations to every pod:
#   node.kubernetes.io/not-ready:NoExecute for 300s
#   node.kubernetes.io/unreachable:NoExecute for 300s
# So a node going NotReady waits 5 min before pods are evicted.
```

## Common Container Exit Codes

| Code | Signal  | Meaning                                                                |
| ---- | ------- | ---------------------------------------------------------------------- |
| 0    | -       | Success                                                                |
| 1    | -       | Generic application error                                              |
| 2    | -       | Misuse of shell builtins (often: bad command flags)                    |
| 125  | -       | Container failed to start (docker run never spawned the process)       |
| 126  | -       | Command found but not executable (chmod +x)                            |
| 127  | -       | Command not found (typo or missing binary in image)                    |
| 128  | -       | Invalid argument to exit                                               |
| 130  | SIGINT  | Container received Ctrl-C                                              |
| 137  | SIGKILL | OOMKilled, or kubelet kill (SIGTERM ignored, then SIGKILL after grace) |
| 139  | SIGSEGV | Segfault                                                               |
| 143  | SIGTERM | Graceful shutdown (the polite goodbye)                                 |

```bash
kubectl get pod my-pod -o jsonpath='{.status.containerStatuses[*].lastState.terminated.exitCode}'
kubectl get pod my-pod -o jsonpath='{.status.containerStatuses[*].lastState.terminated.reason}'
```

## RBAC

```bash
# Can I do this?
kubectl auth can-i create deployments
kubectl auth can-i create deployments --namespace=billing
kubectl auth can-i '*' '*' --all-namespaces                  # cluster-admin?
kubectl auth can-i create deployments --as=alice
kubectl auth can-i create deployments --as=system:serviceaccount:default:web-sa
kubectl auth can-i create deployments --as=alice --as-group=ops

# Full list of allowed verbs
kubectl auth can-i --list
kubectl auth can-i --list -n billing
kubectl auth can-i --list --as=system:serviceaccount:default:web-sa

# Reverse lookup: who can do X (krew plugin)
kubectl who-can list pods -n default
kubectl who-can delete deployments

# Inspect bindings
kubectl get rolebindings,clusterrolebindings -A
kubectl get clusterrolebinding cluster-admin -o yaml
kubectl get clusterrole admin -o yaml | yq .rules

# RBAC structure
# Role/ClusterRole: rules of (apiGroups, resources, verbs, resourceNames)
# RoleBinding/ClusterRoleBinding: subjects -> roleRef
# subjects: { kind: User|Group|ServiceAccount, name, namespace?, apiGroup? }

# ClusterRole aggregation (auto-merge rules from labeled ClusterRoles)
kubectl label clusterrole my-extra rbac.example.com/aggregate-to-admin=true
# Bound ClusterRole "admin" with aggregationRule will pick this up.
```

## Service Account Tokens

```bash
# 1.24+ behavior: ServiceAccount no longer auto-creates a Secret with a token by default.
# Use the TokenRequest API:
kubectl create token web-sa --duration=1h
kubectl create token web-sa --duration=24h --bound-object-kind=Pod --bound-object-name=my-pod

# Projected token (in-pod, auto-rotated)
# spec.containers[].volumeMounts -> ServiceAccountToken volume; kubelet auto-projects.

# Manually create a long-lived secret-bearer token (legacy, 1.24+ requires explicit Secret)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: web-sa-token
  annotations:
    kubernetes.io/service-account.name: web-sa
type: kubernetes.io/service-account-token
EOF
kubectl get secret web-sa-token -o jsonpath='{.data.token}' | base64 -d ; echo

# OIDC / external auth via exec credential plugin (e.g., kubectl-oidc-login, aws eks get-token, gke-gcloud-auth-plugin)
# Configured under users[].exec in kubeconfig:
#   exec:
#     apiVersion: client.authentication.k8s.io/v1
#     command: aws
#     args: [eks, get-token, --cluster-name, prod]
```

## Namespaces

```bash
kubectl get ns
kubectl create namespace billing
kubectl delete namespace billing                             # async; can hang on finalizers!
kubectl get all -n billing                                   # warning: NOT all kinds
kubectl get ns billing -o yaml                                # check status.phase

# Stuck "Terminating" namespace — almost always a finalizer:
kubectl get ns billing -o json | jq '.spec.finalizers, .status'
# Find resources with finalizers preventing deletion:
kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n1 kubectl get -n billing --show-kind --ignore-not-found
# Once empty, remove the namespace finalizer (LAST RESORT):
kubectl get ns billing -o json | jq '.spec.finalizers = []' \
  | kubectl replace --raw "/api/v1/namespaces/billing/finalize" -f -

# Default namespace per context
kubectl config set-context --current --namespace=billing
```

## Resource Quotas + Limits

```bash
kubectl describe resourcequota -n billing
kubectl describe limitrange -n billing

# The canonical pattern: every namespace gets a LimitRange + ResourceQuota
# Without LimitRange, devs ship requests=0/limits=0 and the scheduler thinks the pod is "free"
# (= overcommit, eviction, surprise OOM). LimitRange prevents this by defaulting.
```

## Events

```bash
kubectl get events
kubectl get events -n my-ns
kubectl get events -A
kubectl get events --sort-by=.metadata.creationTimestamp
kubectl get events --sort-by=.lastTimestamp
kubectl get events --field-selector=type=Warning
kubectl get events --field-selector=reason=FailedScheduling
kubectl get events --field-selector=involvedObject.name=my-pod
kubectl get events --watch
kubectl events --for=pod/my-pod                              # 1.27+
kubectl events --watch --for=pod/my-pod
```

## Diff and Dry-Run

```bash
# See changes before apply
kubectl diff -f deploy.yaml
kubectl diff -k overlays/prod
KUBECTL_EXTERNAL_DIFF='diff -u' kubectl diff -f deploy.yaml   # custom diff command

# Dry-run levels
kubectl apply -f deploy.yaml --dry-run=client                 # local only, no API call
kubectl apply -f deploy.yaml --dry-run=server                 # API call, runs admission webhooks, no persist
kubectl create deployment x --image=nginx --dry-run=client -o yaml
kubectl run x --image=nginx --dry-run=client -o yaml
```

## Wait

```bash
# Wait for condition
kubectl wait --for=condition=Ready pod/my-pod --timeout=60s
kubectl wait --for=condition=Available deployment/web --timeout=2m
kubectl wait --for=condition=Complete job/migrate --timeout=10m
kubectl wait --for=condition=Ready pod -l app=web --timeout=120s
kubectl wait --for=jsonpath='{.status.phase}'=Running pod/my-pod --timeout=60s

# Wait for delete
kubectl wait --for=delete pod/my-pod --timeout=30s
kubectl wait --for=delete pod -l app=web --timeout=120s

# Canonical CI pattern
kubectl apply -f manifests/
kubectl rollout status deployment/web --timeout=2m
kubectl wait --for=condition=Available deployment/web --timeout=2m
kubectl wait --for=condition=Ready pod -l app=web --timeout=120s
```

## Common kubectl Errors and Fixes

### Connection refused

```text
The connection to the server localhost:8080 was refused — did you specify the right host or port?
```

- KUBECONFIG empty / not set; kubectl is hitting the default localhost:8080 dev server.
- Fix:
  ```bash
  echo $KUBECONFIG
  kubectl config view --minify
  export KUBECONFIG=~/.kube/config
  ```

### x509 unknown authority

```text
Unable to connect to the server: x509: certificate signed by unknown authority
```

- Cluster API cert is signed by a CA your kubeconfig doesn't have.
- Fix: re-fetch kubeconfig from cluster admin / `aws eks update-kubeconfig` / `gcloud container clusters get-credentials`.

### x509 SANs

```text
x509: cannot validate certificate for 10.0.0.1 because it doesn't contain any IP SANs
```

- API server cert lacks the SAN you're connecting to.
- Fix: connect via DNS name, or regen API certs with proper SANs (`--apiserver-cert-extra-sans`).

### Forbidden (RBAC)

```text
Error from server (Forbidden): pods is forbidden: User "alice" cannot list resource "pods" in API group "" in the namespace "billing"
```

- Fix:
  ```bash
  kubectl auth can-i list pods -n billing --as=alice
  kubectl auth can-i --list -n billing --as=alice
  # Then create a Role + RoleBinding granting list pods.
  ```

### FailedScheduling

```text
Warning  FailedScheduling  ...  0/5 nodes are available: 3 Insufficient cpu, 2 node(s) had untolerated taint {dedicated: gpu}.
```

- Causes: requests too large; taints; nodeSelector; affinity/anti-affinity; PVC waiting on volume.
- Fix:
  ```bash
  kubectl describe pod my-pod | grep -A3 Events
  kubectl describe nodes | grep -E 'Taints|Allocated resources' -A5
  # Reduce requests; add tolerations; adjust nodeSelector; expand the cluster.
  ```

### ImagePullBackOff / ErrImagePull

```text
Failed to pull image "ghcr.io/me/web:1.2.3": rpc error: code = Unknown desc = failed to pull and unpack image "ghcr.io/me/web:1.2.3": failed to resolve reference "ghcr.io/me/web:1.2.3": failed to authorize: ...
```

- Causes: image typo, image doesn't exist, registry auth missing, network egress blocked.
- Fix:
  ```bash
  kubectl describe pod my-pod | tail -30
  kubectl get sa default -o yaml                # imagePullSecrets?
  kubectl get secret regcred -o yaml            # exists? right ns?
  ```

### OOMKilled

```text
Last State:     Terminated
  Reason:       OOMKilled
  Exit Code:    137
```

- Fix: bump `resources.limits.memory`; profile the app; check for goroutine leaks / large allocations.

### no matches for kind

```text
error: resource mapping not found for name: "my-resource" namespace: "" from "manifest.yaml": no matches for kind "MyKind" in version "example.com/v1"
ensure CRDs are installed first
```

- CRD missing or wrong apiVersion.
- Fix:
  ```bash
  kubectl get crds | grep example.com
  kubectl apply -f crd.yaml                     # install CRD before resources
  kubectl apply -f manifest.yaml --server-side  # SSA tolerates ordering better
  ```

### Admission webhook denied

```text
Error from server: error when creating "deploy.yaml": admission webhook "validate.kyverno.svc" denied the request: validation error: [...]
```

- Fix:
  ```bash
  kubectl describe deployment my-app
  kubectl get events --field-selector=type=Warning
  kubectl get validatingwebhookconfigurations
  # Adjust manifest to satisfy policy; or fix/disable the webhook.
  ```

### Container terminated OOMKilled

```text
Container has been terminated due to "OOMKilled". Reason: OOMKilled
```

- See OOMKilled above.

### Namespace stuck terminating

```text
namespace "billing" is being terminated, attempts to insert resources rejected
```

- A finalizer is blocking deletion.
- Fix: see Namespaces section; remove offending finalizer after migrating resources.

### TLS handshake / kubelet API

```text
Error from server: Get "https://10.0.1.5:10250/containerLogs/...": tls: failed to verify certificate: x509: cannot validate certificate for 10.0.1.5 because it doesn't contain any IP SANs
```

- kubelet serving cert SANs.
- Fix: enable kubelet serving cert rotation; or set `--tls-cert-file` / `--tls-private-key-file` with proper SANs.

### Throttled by API server

```text
client rate limiter Wait returned an error: rate: Wait(n=1) would exceed context deadline
```

- Fix:
  ```bash
  kubectl get pods --chunk-size=500 -A
  # Or bump --kube-api-burst and --kube-api-qps in your client config.
  ```

## Common Patterns (broken+fixed)

```bash
# BAD: exec into a crashed pod
kubectl exec -it my-pod -- sh                                # error: container is not running
# FIXED:
kubectl logs my-pod --previous
kubectl debug -it my-pod --image=busybox --copy-to=debug --share-processes

# BAD: delete a Deployment-managed pod hoping it goes away
kubectl delete pod web-7d8b9c8d4-x2j5k                       # ReplicaSet recreates it instantly
# FIXED:
kubectl scale deployment/web --replicas=0
# or
kubectl delete deployment web

# BAD: edit a pod that's part of a Deployment
kubectl edit pod web-7d8b9c8d4-x2j5k                         # overwritten on next reconcile
# FIXED:
kubectl edit deployment web

# BAD: rely on --record (deprecated 1.19+)
kubectl apply -f deploy.yaml --record
# FIXED: annotate
kubectl annotate deployment/web kubernetes.io/change-cause="bump 1.2.4 (#123)" --overwrite

# BAD: bind ServiceAccount to cluster-admin "to make it work"
kubectl create clusterrolebinding web-admin --clusterrole=cluster-admin --serviceaccount=default:web-sa
# FIXED: least-privilege Role
kubectl create role web-reader --verb=get,list,watch --resource=pods,configmaps -n default
kubectl create rolebinding web-reader --role=web-reader --serviceaccount=default:web-sa -n default

# BAD: latest tag in production
image: my/web:latest                                          # no rollback safety
# FIXED:
image: my/web@sha256:abc123...                                # immutable digest
# or pinned
image: my/web:1.2.3

# BAD: kubectl get all hoping you saw everything
kubectl get all                                               # NO ConfigMap, NO Secret, NO Ingress, NO PVC, NO CRDs
# FIXED:
kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n1 kubectl get --show-kind --ignore-not-found -n my-ns

# BAD: kubectl delete -f manifest.yaml --grace-period=0 --force as a habit
# FIXED: only when pod is genuinely stuck (kubelet unreachable, finalizer)

# BAD: read a Secret with `kubectl describe secret`
kubectl describe secret db                                    # only shows lengths
# FIXED:
kubectl get secret db -o jsonpath='{.data.password}' | base64 -d ; echo
# Or:
kubectl view-secret db                                        # krew plugin
```

## Performance and Scale Tips

```bash
# Pagination on huge lists
kubectl get pods -A --chunk-size=500

# Field-selector to filter server-side (cheaper than -o json | jq)
kubectl get pods -A --field-selector=status.phase=Running

# Avoid -o yaml on enormous resources
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.allocatable.cpu}{"\n"}{end}'

# Use --raw for special endpoints
kubectl get --raw /api/v1/namespaces/default/pods/my-pod/status
kubectl get --raw /readyz
kubectl get --raw /metrics                                    # prometheus-style

# Discovery + OpenAPI cache
ls ~/.kube/cache/discovery/
ls ~/.kube/cache/http/
# Stale cache symptom: `kubectl explain` for a CRD shows nothing → delete the cache:
rm -rf ~/.kube/cache/discovery/

# Krew get-all walks API discovery and lists EVERY object
kubectl get-all -n my-ns

# Throttle settings (set on client side)
# --kube-api-burst, --kube-api-qps (bake into kubectl flags or kubeconfig users[].client-key:)
```

## Plugins via krew (curated set)

```bash
kubectl krew install ctx              # context switcher
kubectl krew install ns               # namespace switcher
kubectl krew install neat             # `kubectl get -o yaml | kubectl neat` strips junk
kubectl krew install tree             # owner-reference tree (Deployment -> RS -> Pods)
kubectl krew install view-secret      # decode a Secret in one shot
kubectl krew install who-can          # reverse RBAC lookup
kubectl krew install slice            # split multi-doc yaml into files
kubectl krew install df-pv            # PV / PVC usage report
kubectl krew install get-all          # list every object in every kind
kubectl krew install access-matrix    # full subjects x verbs RBAC matrix
kubectl krew install resource-capacity
kubectl krew install trace            # bpftrace targeted at pod/container
kubectl krew install krew-blame       # who installed which plugin
kubectl krew install debug-shell

# Daily power-flow
k ctx prod
k ns billing
k get pods
k tree deployment web
k view-secret db password
```

## Idioms

```bash
# Daily alias + completion
alias k=kubectl
source <(kubectl completion bash)
complete -F __start_kubectl k
# zsh:
source <(kubectl completion zsh)

# Watch your deployment go
watch -n 1 kubectl get pods -l app=web -o wide

# Clean YAML before you store/diff
kubectl get deployment web -o yaml | kubectl neat > web.yaml

# JSONPath one-liner: pods with restarts > 0
kubectl get pods -A -o jsonpath='{range .items[?(@.status.containerStatuses[*].restartCount>0)]}{.metadata.namespace}/{.metadata.name}{"\n"}{end}'

# Safe restart pattern for a Deployment
kubectl rollout restart deployment/web
kubectl rollout status deployment/web --timeout=2m

# Tail logs across a Deployment
kubectl logs -f -l app=web --all-containers --prefix --max-log-requests=20

# Reload a ConfigMap (Deployments don't auto-reload mounted configmaps)
kubectl rollout restart deployment/web
# Or use https://github.com/stakater/Reloader to auto-bump on cm/secret change.

# Capture a failing pod's state for postmortem before deleting
kubectl get pod my-pod -o yaml > /tmp/my-pod-state.yaml
kubectl logs my-pod --all-containers --previous > /tmp/my-pod-prev.log
kubectl describe pod my-pod > /tmp/my-pod-describe.txt
kubectl get events --field-selector=involvedObject.name=my-pod -o yaml > /tmp/my-pod-events.yaml

# Lock a CronJob without deleting it
kubectl patch cronjob nightly-backup -p '{"spec":{"suspend":true}}'

# Trigger a CronJob run NOW (one-off Job from its template)
kubectl create job manual-$(date +%s) --from=cronjob/nightly-backup

# Drain a node (before cordon/maintenance)
kubectl cordon worker-3                                       # mark unschedulable
kubectl drain worker-3 --ignore-daemonsets --delete-emptydir-data --grace-period=120 --timeout=10m
# After maintenance:
kubectl uncordon worker-3

# Taint a node (only pods with matching toleration can land)
kubectl taint nodes worker-1 dedicated=gpu:NoSchedule
kubectl taint nodes worker-1 dedicated:NoSchedule-                # remove

# Scale a StatefulSet down ordered (N-1, N-2, ...) to retain volumes
kubectl scale statefulset/db --replicas=0
# PVCs remain (volumeClaimTemplates create them, do NOT auto-delete).

# Force-delete a stuck pod (LAST RESORT)
kubectl delete pod my-pod --grace-period=0 --force

# JSON Patch (RFC 6902) — surgical edits
kubectl patch deployment/web --type=json \
  -p='[{"op":"replace","path":"/spec/replicas","value":4}]'
kubectl patch deployment/web --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/env/-","value":{"name":"DEBUG","value":"1"}}]'

# Strategic Merge Patch (default; understands list keys)
kubectl patch deployment/web -p '{"spec":{"template":{"spec":{"containers":[{"name":"app","image":"nginx:1.28"}]}}}}'
```

## Sidecar Containers (1.29 stable)

```yaml
# A sidecar is now a "restartable initContainer".
# It starts before main containers, stays running, gets a graceful shutdown after main containers exit.
spec:
  initContainers:
    - name: log-shipper
      image: fluent/fluent-bit:3.0
      restartPolicy: Always                 # this is what makes it a sidecar
      volumeMounts: [{ name: logs, mountPath: /var/log }]
  containers:
    - name: app
      image: my/app:1.0
```

## Tips

- `kubectl get` does NOT include CRDs in `all` — list them with `kubectl get crds` and `kubectl get <crd-kind>`.
- `kubectl describe` order matters: read TOP-DOWN: phase → conditions → containers (state, lastState) → events.
- `kubectl logs` with `-l` caps at 5 pods by default; pass `--max-log-requests=N` for more.
- `kubectl edit` opens YAML; if the cluster rejects the diff, kubectl saves your edit to `/tmp/kubectl-edit-XXXX.yaml` so you can fix and `kubectl apply` it.
- `kubectl run` is for ad-hoc/debug — never put it in a pipeline; use `apply -f` and a real manifest.
- `kubectl debug` (1.25+) is the modern replacement for "kubectl exec into a debug image and hope".
- The `default` ServiceAccount has NO permissions in 1.6+ unless bound. Don't bind cluster-admin to it as a shortcut.
- `--server-side` apply is the future; client-side apply's last-applied annotation is fragile under multiple controllers.
- `kubectl wait --for=condition=Ready` only works once the resource has the condition — for fresh objects, the condition may not exist for a moment; tolerate this with retries in CI.
- Use `kubectl diff -f` in PR reviews — it surfaces unexpected drift before apply.
- `kubectl proxy` opens a local proxy to the API on 127.0.0.1:8001 — useful for browsing the dashboard or hitting the API with curl without auth tokens (`curl localhost:8001/api/v1/namespaces`).
- The `/healthz`, `/readyz`, `/livez` endpoints are reachable via `kubectl get --raw /readyz?verbose`.
- Node memory pressure → `Evicted` is silent. Always set `requests.memory` so the scheduler accounts for it.
- For multi-cluster ops, prefer separate kubeconfigs (`KUBECONFIG=:`) over giant merged configs — cleaner, easier to audit.

## See Also

- [kubernetes](../orchestration/kubernetes.md) — Kubernetes architecture, controllers, scheduler, API objects
- [helm](../orchestration/helm.md) — Helm chart packaging on top of kubectl-apply
- [istio](../orchestration/istio.md) — service mesh sitting on top of Services/Pods
- [docker](../orchestration/docker.md) — container build/run; the image kubectl ultimately deploys
- [yaml](../orchestration/yaml.md) — manifest syntax, anchors, multi-doc
- [polyglot](../orchestration/polyglot.md) — language quick reference for app code that ends up in pods
- [bash](../orchestration/bash.md) — shell scripting around kubectl in CI/operations

## References

- Kubernetes kubectl reference: https://kubernetes.io/docs/reference/kubectl/
- Official kubectl cheatsheet: https://kubernetes.io/docs/reference/kubectl/quick-reference/
- kubectl conventions: https://kubernetes.io/docs/reference/kubectl/conventions/
- JSONPath support: https://kubernetes.io/docs/reference/kubectl/jsonpath/
- Concepts (Pod, Service, Volume, ...): https://kubernetes.io/docs/concepts/
- API reference (object schemas): https://kubernetes.io/docs/reference/kubernetes-api/
- krew plugin manager: https://krew.sigs.k8s.io/
- krew plugin index: https://krew.sigs.k8s.io/plugins/
- Common labels: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
- RBAC reference: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
- Pod lifecycle (states/phases): https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/
- Debug pods (ephemeral containers): https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod/
- Sidecar containers: https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/
- Server-side apply: https://kubernetes.io/docs/reference/using-api/server-side-apply/
- Gateway API: https://gateway-api.sigs.k8s.io/
- Version skew policy: https://kubernetes.io/releases/version-skew-policy/
