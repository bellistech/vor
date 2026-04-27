# Helm — ELI5

> Helm is **npm for Kubernetes.** A "chart" is a folder of YAML templates. You install it, Helm fills in the blanks with your values, and ships the rendered YAML to your cluster.

## Prerequisites

- `cs ramp-up kubernetes-eli5` — strongly recommended. If you do not yet know what a Pod, a Deployment, a Service, a Namespace, or `kubectl apply -f file.yaml` is, read that sheet first. Helm is just a fancy way of producing those YAML files. If the YAML files themselves look like alphabet soup, this sheet will look like alphabet soup that has been put through a blender.
- `cs ramp-up docker-eli5` — handy. Charts always end up running container images. If you have never built a container, the chapter where we talk about `image: nginx:1.25` may feel hand-wavy.
- A terminal you can type into.
- A working Kubernetes cluster of any size. **Tiny is fine.** A laptop kind cluster, a minikube, a Docker Desktop with Kubernetes turned on, a k3d cluster, or a managed cluster in a cloud — they all work. If you do not have one yet, the kubernetes-eli5 sheet walks you through making one.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Helm

### A package manager, but for clusters

You probably already use a package manager. If you write JavaScript, you use **npm.** You type `npm install react` and a folder full of files appears in your project, and now your project knows how to use React. If you run a Mac, you use **Homebrew.** You type `brew install jq` and a tool you can use from anywhere on your computer just appears. If you run Ubuntu, you use **apt.** You type `apt install nginx` and an entire web server installs itself.

Package managers do four things in one.

1. **Find** a thing you want.
2. **Download** the thing.
3. **Install** the thing into your system the right way.
4. **Track** that you installed it, so you can update it or remove it later.

Now imagine you want to do that for a **Kubernetes cluster.** You don't want to install one tiny program — you want to install a whole running thing. A database. A web server. A monitoring system. Each of those is not just a file. Each of those is a bundle of:

- a Deployment (the recipe for running the program in pods)
- a Service (the network address that points at those pods)
- a ConfigMap (configuration file content)
- a Secret (the passwords)
- maybe an Ingress (a public web address)
- maybe a PersistentVolumeClaim (storage)
- maybe a ServiceAccount, a Role, a RoleBinding (permissions)
- maybe a CRD or two (custom Kubernetes object types)

Twelve YAML files. Possibly forty. Possibly a hundred. All written in the right shape, with values that match each other, with names that follow rules, with labels and annotations that have to agree across files.

That is hard. That is annoying. That is what **Helm** fixes.

**Helm is the package manager for Kubernetes clusters.** It is `apt-get` for things that run inside a cluster. The thing being installed is called a **chart** (because, well, helms steer ships and ships have charts, and the name stuck). When you `helm install` something, you are doing the cluster equivalent of `apt install`. Helm finds the chart, downloads it, fills in the blanks with your values, sends the resulting YAML to the cluster, and writes down "this thing is installed and here is its history" so you can upgrade or roll back later.

### One-line picture

```
+--------+   helm install    +-------+   kubectl apply   +---------+
| chart  | ----------------> | Helm  | ----------------> | cluster |
| (yaml  |   + your values   | (renders)                 | (Pods,  |
|  templ)|                   |                            |  Svcs,  |
+--------+                   +-------+                    |  ...)   |
                                                          +---------+
```

That diagram is the whole product. The chart is paper templates. Your values are the ink. Helm renders the templates into real Kubernetes YAML and sends them to the cluster. Then it remembers what it sent.

### Why is this better than just writing YAML?

Imagine you wrote your own Postgres deployment for your team. Forty-five YAML files. Now your friend at another company wants to run Postgres too. Without Helm they have to copy your files, edit your hardcoded names, edit your hardcoded passwords, edit your hardcoded version, hope they got everything, and pray. With Helm, you wrap those forty-five files into a chart with little blanks for "name" and "password" and "version", you publish it, and your friend types `helm install mypg bitnami/postgresql --set auth.password=hunter2` and it just works. The chart gets to be reusable across thousands of users with different needs.

That is also why **public chart repositories** are great. The Bitnami project, the prometheus-community project, jetstack, ingress-nginx, and many others maintain charts that you can install with one command. Tens of thousands of teams use these charts and have caught all the bugs already. You install Postgres in one command and your version is the same as everyone else's, which means somebody else has already fixed the weird startup error you were about to hit.

### Three big words

Three words show up over and over with Helm. Lock these in.

- **Chart** — a folder full of templated YAML, plus a `Chart.yaml` description file. The thing you install.
- **Release** — a chart that has been installed into a cluster. It has a name (you choose), a namespace, and a revision number that goes up by one every time you upgrade. Two installs of the same chart with different names are two different releases.
- **Repository** — a website that hosts a bunch of charts. You can browse it, search it, and install from it. Examples: Bitnami, prometheus-community, jetstack.

So the verbs of the trade are:

- **Add a repository:** `helm repo add bitnami https://charts.bitnami.com/bitnami`
- **Search a repository:** `helm search repo postgres`
- **Install a chart from a repo as a release:** `helm install mypg bitnami/postgresql`
- **Upgrade a release:** `helm upgrade mypg bitnami/postgresql`
- **Roll back a release:** `helm rollback mypg 2`
- **Uninstall a release:** `helm uninstall mypg`

Notice that the chart and the release are different. The chart is the recipe. The release is the meal you cooked. You can cook the same recipe twice with different names, and you can throw out the meal without throwing out the recipe.

### What problem does Helm not solve?

Helm does not run your cluster. Helm does not configure cloud accounts. Helm does not give you a database. Helm does not give you observability. Helm just packages and installs YAML for an existing Kubernetes cluster.

Helm is also **not** Argo CD or Flux. Argo CD and Flux can use Helm charts, but they are GitOps controllers — they continuously reconcile your cluster with what is in a git repository. Helm by itself is a one-shot CLI: you run `helm install`, it does the install, and then the CLI exits. Argo CD picks up where Helm leaves off if you want continuous reconciliation.

Helm also does not replace Kustomize. Kustomize is a different tool that does overlays on top of plain YAML. Some teams use both: Helm to install upstream charts, Kustomize to patch them. Helm 3.1+ supports a `--post-renderer` flag that pipes the rendered YAML through any program you want — most often that program is `kustomize`, which is the official "Helm + Kustomize" pattern.

## A Hello-World Chart (helm create hello)

The fastest way to feel a chart is to make one. Helm has a built-in command that scaffolds a starter chart with all the default files. You will not edit much. You will just look at it and run it.

```bash
$ helm version
version.BuildInfo{Version:"v3.14.4", GitCommit:"...", GitTreeState:"clean", GoVersion:"go1.21.9"}

$ helm create hello
Creating hello

$ ls hello
Chart.yaml  charts  templates  values.yaml

$ ls hello/templates
NOTES.txt           hpa.yaml            service.yaml
_helpers.tpl        ingress.yaml        serviceaccount.yaml
deployment.yaml     serviceaccount.yaml tests
```

Beautiful. Six files in `templates/`, plus a `_helpers.tpl`. Plus a top-level `Chart.yaml` and `values.yaml`. That is a working chart.

Let's see what it would do.

```bash
$ helm template hello ./hello
---
# Source: hello/templates/serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hello-hello
  labels:
    helm.sh/chart: hello-0.1.0
    app.kubernetes.io/name: hello
    app.kubernetes.io/instance: hello
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
---
# Source: hello/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: hello-hello
  labels:
    helm.sh/chart: hello-0.1.0
    app.kubernetes.io/name: hello
    app.kubernetes.io/instance: hello
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: hello
    app.kubernetes.io/instance: hello
---
# Source: hello/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-hello
  labels:
    helm.sh/chart: hello-0.1.0
    app.kubernetes.io/name: hello
    app.kubernetes.io/instance: hello
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: hello
      app.kubernetes.io/instance: hello
  template:
    metadata:
      labels:
        app.kubernetes.io/name: hello
        app.kubernetes.io/instance: hello
    spec:
      serviceAccountName: hello-hello
      containers:
        - name: hello
          image: "nginx:1.16.0"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
```

That is the chart, **rendered.** Three Kubernetes objects: a ServiceAccount, a Service, a Deployment. Notice that the labels say `app.kubernetes.io/managed-by: Helm` — Helm always stamps that on. That is how you tell, looking at random YAML in a cluster, "this came from Helm."

Now actually install it.

```bash
$ helm install hello ./hello
NAME: hello
LAST DEPLOYED: Mon Apr 27 10:12:04 2026
NAMESPACE: default
STATUS: deployed
REVISION: 1
NOTES:
1. Get the application URL by running these commands:
  export POD_NAME=$(kubectl get pods --namespace default -l "app.kubernetes.io/name=hello,app.kubernetes.io/instance=hello" -o jsonpath="{.items[0].metadata.name}")
  export CONTAINER_PORT=$(kubectl get pod --namespace default $POD_NAME -o jsonpath="{.spec.containers[0].ports[0].containerPort}")
  echo "Visit http://127.0.0.1:8080 to use your application"
  kubectl port-forward $POD_NAME 8080:$CONTAINER_PORT
```

Look at the `NOTES.txt` getting printed. That is the chart talking back to you. The chart author wrote those instructions so users would know what to do next.

```bash
$ helm ls
NAME    NAMESPACE  REVISION  UPDATED                              STATUS    CHART        APP VERSION
hello   default    1         2026-04-27 10:12:04.123 +0000 UTC    deployed  hello-0.1.0  1.16.0

$ kubectl get pods
NAME                          READY   STATUS    RESTARTS   AGE
hello-hello-7c4d8c98b-q9k7f   1/1     Running   0          15s

$ kubectl get svc
NAME          TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE
hello-hello   ClusterIP   10.96.7.182   <none>        80/TCP    15s
```

The release exists. The pod is running. You typed three real-ish commands and got an entire mini app deployed. Now uninstall it.

```bash
$ helm uninstall hello
release "hello" uninstalled
```

That is the entire happy path. Install. Render. Inspect. Uninstall. Everything else in this sheet is depth on those four verbs.

## Chart Anatomy

A chart is a folder. The folder has a specific layout. Helm cares about specific filenames. If you put a file in the wrong place, Helm ignores it.

### The folder tree

```
mychart/
+-- Chart.yaml          # the chart's "package.json" — name, version, description
+-- values.yaml         # default values that fill the templates
+-- values.schema.json  # optional JSON Schema validating the values shape
+-- README.md           # human docs (Artifact Hub displays this)
+-- LICENSE             # optional license file
+-- .helmignore         # files to skip when packaging
+-- templates/
|   +-- _helpers.tpl    # named template snippets (no leading underscore = renders)
|   +-- deployment.yaml # actual k8s objects (templated)
|   +-- service.yaml
|   +-- ingress.yaml
|   +-- NOTES.txt       # printed after install/upgrade
|   +-- tests/
|       +-- test-connection.yaml  # helm test pods
+-- charts/             # subcharts (dependencies live here once fetched)
|   +-- redis/
|   +-- common/
+-- crds/               # CustomResourceDefinitions installed first, never templated
    +-- mything-crd.yaml
```

### Chart.yaml

This is the chart's "package.json." It looks like this:

```yaml
apiVersion: v2
name: mychart
description: An example chart
type: application
version: 0.1.0          # the chart version (this is what Helm tracks)
appVersion: "1.16.0"    # the version of the *thing* the chart packages
icon: https://example.com/icon.png
keywords: [example, demo]
home: https://example.com
sources:
  - https://github.com/example/mychart
maintainers:
  - name: You
    email: you@example.com
dependencies:
  - name: redis
    version: "^18.0.0"
    repository: https://charts.bitnami.com/bitnami
    condition: redis.enabled
```

The two version fields confuse everyone. **`version`** is the version of the chart itself: bump it every time you change templates. **`appVersion`** is the version of the application the chart packages: bump it when, say, the upstream project releases nginx 1.27 and you update the default image tag.

`apiVersion: v2` is mandatory for Helm 3+ charts. `v1` was the Helm 2 format and is dead. `type:` is `application` (the normal kind) or `library` (a chart whose only job is to provide helper templates to other charts; library charts cannot themselves be installed).

### values.yaml

The defaults. Everything in here is just YAML. The keys you choose are entirely up to you — Helm doesn't care what you call them. Convention: use camelCase, mirror the structure of the things you template.

```yaml
replicaCount: 1
image:
  repository: nginx
  pullPolicy: IfNotPresent
  tag: ""              # if empty, fall back to .Chart.AppVersion
service:
  type: ClusterIP
  port: 80
resources: {}
ingress:
  enabled: false
  className: ""
  hosts:
    - host: chart-example.local
      paths:
        - path: /
          pathType: ImplementationSpecific
```

When a user installs your chart, Helm loads this file as the starting point, then layers their `-f my-values.yaml` on top, then their `--set foo=bar` on top of that. The final merged object is what your templates see as `.Values`.

### templates/

Every file in here that does **not** start with an underscore (`_`) and is **not** `NOTES.txt` becomes a Kubernetes object. Helm renders the file with Go templates, then submits the result to the cluster. Files that start with `_` are partials — they don't render directly, they only define snippets that other templates can include.

```
templates/_helpers.tpl   # partial, renders nothing on its own
templates/deployment.yaml  # renders to a Deployment object
templates/NOTES.txt        # printed text, not applied to cluster
templates/tests/probe.yaml # rendered, applied during `helm test`
```

You can have nested folders inside `templates/`. They all flatten into one rendering context.

### _helpers.tpl

The conventional name for "this file holds reusable template snippets." Look inside the scaffold and you'll see:

```gotemplate
{{/*
Expand the name of the chart.
*/}}
{{- define "mychart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "mychart.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
```

Other templates use them with `include`:

```yaml
metadata:
  name: {{ include "mychart.fullname" . }}
```

### charts/

This folder holds **subcharts** — other charts that your chart depends on. You usually don't put files here yourself; you write `dependencies:` in `Chart.yaml` and run `helm dependency update`, which downloads the subcharts as `.tgz` files into `charts/` for you. Subcharts get rendered into the same release as the parent.

### crds/

A special directory for **Custom Resource Definitions.** Files in `crds/` are **plain YAML** — they are not templated. Helm installs them **before** anything in `templates/`. CRDs in this directory are also a one-time thing: Helm installs them on first install but does not upgrade them. (For day-to-day CRDs that you want to evolve, many charts choose to put CRDs in `templates/` instead and accept the trade-offs.)

### NOTES.txt

The chat message. Whatever you write here gets printed to the user after `helm install` and `helm upgrade`. It is templated, so you can include things like the actual ingress hostname or a kubectl port-forward command. This is the chart's "what now" doc.

### .helmignore

Like `.gitignore`. Files matching the patterns in this file are not packaged into the `.tgz` when you run `helm package`. Saves you from shipping `.git/`, `.DS_Store`, your unit test fixtures, etc.

### README.md

Optional but standard. **Artifact Hub** scrapes it and displays it on your chart's page. So it functions as your public-facing chart docs.

## Templating Engine (Go templates + Sprig)

Helm uses **Go templates** to fill in the blanks. Go templates have a syntax that, frankly, takes a few hours to grow on you. The good news is you can copy-paste 95% of what you ever need from the scaffold and from other people's charts. The bad news is the other 5% will trip you up at least once.

### The basics: `{{ ... }}`

Anywhere inside a template file, double-curly-braces are an expression that gets evaluated and inserted.

```yaml
metadata:
  name: {{ .Release.Name }}-{{ .Chart.Name }}
  labels:
    app: {{ .Values.name }}
```

If `.Release.Name` is `prod` and `.Chart.Name` is `web` and `.Values.name` is `frontend`, that renders to:

```yaml
metadata:
  name: prod-web
  labels:
    app: frontend
```

The `.` (just a dot) means "the root scope." Inside `{{ ... }}` you walk into nested fields with dotted paths. Anything in `values.yaml` is under `.Values`. Anything from `Chart.yaml` is under `.Chart`. Helm provides `.Release` (info about this install), `.Files` (other files in the chart), and `.Capabilities` (info about the cluster).

### Built-in objects

| Object | What it is |
|---|---|
| `.Release.Name` | the name you gave at install time (`helm install <name>`) |
| `.Release.Namespace` | the namespace |
| `.Release.IsInstall` | `true` on first install, `false` on upgrade |
| `.Release.IsUpgrade` | the opposite |
| `.Release.Revision` | starts at 1, increments per upgrade or rollback |
| `.Chart.Name` | from Chart.yaml |
| `.Chart.Version` | from Chart.yaml |
| `.Chart.AppVersion` | from Chart.yaml |
| `.Values` | the merged user/default values |
| `.Files` | helpers to read other files in the chart |
| `.Capabilities.KubeVersion.Version` | k8s version of the cluster |
| `.Capabilities.APIVersions.Has "..."` | does this k8s API exist on this cluster? |
| `.Template.Name` | name of the current template file |
| `.Subcharts` | values for subcharts (if you are the parent) |

### Pipes and Sprig

The `|` (pipe) chains values through functions, like in Bash.

```yaml
labels:
  app: {{ .Chart.Name | quote }}
  ts:  {{ now | date "2006-01-02" | quote }}
```

`quote` wraps the value in double quotes. `now` returns current time. `date` formats it. **All of these are Sprig functions** — Sprig is a giant library of helpers built into Helm. You did not have to add anything to use them.

A handful you will use constantly:

- `default "x" .Values.foo` — if `.Values.foo` is empty, use `"x"`
- `required "msg" .Values.foo` — if `.Values.foo` is empty, fail the render with `msg`
- `quote` — wrap in double quotes
- `lower`, `upper`, `title` — case
- `trim`, `trimSuffix "-"`, `trimPrefix "/"` — string trim
- `b64enc`, `b64dec` — base64 (use this in Secret data)
- `sha256sum` — hash a string
- `toYaml`, `fromYaml`, `toJson`, `fromJson` — convert
- `indent N`, `nindent N` — indent every line by N spaces (nindent adds a leading newline first; you almost always want nindent)
- `lookup "v1" "Pod" "default" "mypod"` — look up live state from the cluster
- `tpl "{{ .Values.x }}" .` — template-render a string at runtime

### Control flow

```gotemplate
{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "mychart.fullname" . }}
spec:
  rules:
  {{- range .Values.ingress.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
        {{- range .paths }}
          - path: {{ .path }}
            pathType: {{ .pathType }}
        {{- end }}
  {{- end }}
{{- end }}
```

Three control structures:

- `if / else / end` — like every language
- `range` — like a `foreach` loop. Inside, `.` becomes the current item.
- `with` — sets the scope. Inside `{{- with .Values.image }}`, `.` becomes `.Values.image`.

### Trimming whitespace: the dashes

`{{- ... }}` trims whitespace **before** the expression. `{{ ... -}}` trims whitespace **after.** Without dashes, Go templates leave behind blank lines from your template file. With dashes, they don't.

A general rule: most lines that are pure logic (not output) should have dashes on both sides. Lines that produce output usually don't need them.

If your rendered YAML has weird blank lines or weird missing newlines, look at the dashes.

### The `tpl` function (template a string)

Sometimes a value in `values.yaml` itself contains a template expression and you want to render it. That is what `tpl` is for.

```yaml
# values.yaml
welcomeMsg: "Hello {{ .Release.Name }}!"
```

```gotemplate
# template
data:
  msg: {{ tpl .Values.welcomeMsg . | quote }}
```

Without `tpl`, the brace literal would just appear in the output as text. With `tpl`, it gets rendered.

## Releases (helm install, upgrade, rollback, uninstall)

A release is the "instance" you get when you install a chart into a cluster. It has:

- a **name** (you choose at install time)
- a **namespace** (where its objects live)
- a **revision** (a counter that goes up by one each time you change anything)
- a **status** (deployed, failed, superseded, uninstalled, pending-install, pending-upgrade, etc.)
- a **history** (every revision is kept around so you can roll back; default keeps last 10)

The data backing a release is stored as a **Kubernetes Secret** in the release's namespace, with name like `sh.helm.release.v1.<name>.v<revision>`. That is one of the cooler design moves of Helm 3 — there is no separate database, no Tiller server. The cluster itself stores the release.

```bash
$ kubectl get secret -l owner=helm -n default
NAME                          TYPE                 DATA   AGE
sh.helm.release.v1.hello.v1   helm.sh/release.v1   1      2m
sh.helm.release.v1.hello.v2   helm.sh/release.v1   1      30s
```

Two revisions, two Secrets. Roll back to v1:

```bash
$ helm rollback hello 1
Rollback was a success! Happy Helming!
```

Now there is a v3 secret (yes, **rolling back creates a new revision** — so if you roll back twice you don't end up at v1, you end up at v3 which is a copy of v1).

### `helm install`

```bash
helm install <release-name> <chart-ref> [flags]
```

Chart-ref can be:

- a path to a local folder: `./mychart`
- a path to a packaged chart: `./mychart-0.1.0.tgz`
- `<repo>/<chart>` after `helm repo add`: `bitnami/redis`
- a full URL to a `.tgz`
- an OCI registry: `oci://ghcr.io/me/mychart`

Common flags:

- `-n, --namespace ns` — install into namespace `ns` (default: `default`)
- `--create-namespace` — create the namespace if missing
- `-f values.yaml` — a values file (can be repeated; later wins on conflict)
- `--set foo=bar` — set a value inline
- `--set-string foo=bar` — same but treat as string (don't try to parse as bool/int)
- `--set-file foo=path` — load value from file
- `--version 1.2.3` — pin chart version
- `--dry-run` — render but don't apply
- `--debug` — print rendered YAML and verbose info
- `--wait` — block until all resources are ready
- `--wait-for-jobs` — also wait for Jobs to complete
- `--timeout 5m` — how long to wait
- `--atomic` — if install fails, automatically uninstall
- `--cleanup-on-fail` — like `--atomic` but only cleans up the last batch
- `--skip-crds` — do not install resources from `crds/`
- `--include-crds` (template only) — include CRDs in the rendered output

### `helm upgrade --install`

This is the **idempotent** form. If the release does not exist, install. If it does, upgrade. Use this in CI/CD almost always.

```bash
helm upgrade --install hello ./mychart -n myns --create-namespace --wait --timeout 5m
```

### `helm rollback`

```bash
helm rollback <release> [revision]
```

If you omit the revision, it rolls back to the previous revision.

```bash
$ helm history hello
REVISION  UPDATED                   STATUS      CHART        APP VERSION  DESCRIPTION
1         2026-04-27 10:12:04 UTC   superseded  hello-0.1.0  1.16.0       Install complete
2         2026-04-27 10:15:00 UTC   superseded  hello-0.1.0  1.16.0       Upgrade complete
3         2026-04-27 10:17:33 UTC   deployed    hello-0.1.0  1.16.0       Rollback to 1
```

### `helm uninstall`

```bash
helm uninstall <release> [-n namespace]
```

By default, this **deletes the release history** as well. If you want to keep history (so you can see what was once installed):

```bash
helm uninstall hello --keep-history
```

That leaves the release marked `uninstalled` rather than gone. You can `helm rollback` to a previous revision later if you want to bring it back.

### Status states

A release is always in one of these states:

| State | Meaning |
|---|---|
| `pending-install` | install is in progress |
| `deployed` | installed and currently active |
| `pending-upgrade` | upgrade is in progress |
| `pending-rollback` | rollback is in progress |
| `superseded` | this revision used to be deployed but isn't anymore |
| `failed` | the install/upgrade failed |
| `uninstalling` | uninstall is in progress |
| `uninstalled` | uninstalled (only seen if `--keep-history`) |

Helm 3.13+ added **status drift detection**: if the resources in the cluster have been changed outside of Helm, `helm status` flags it.

## Repositories

A Helm repository is a website that serves an `index.yaml` plus a bunch of `.tgz` chart packages. That's it. Any HTTPS server can host one.

```bash
$ helm repo add bitnami https://charts.bitnami.com/bitnami
"bitnami" has been added to your repositories

$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
"prometheus-community" has been added to your repositories

$ helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
"ingress-nginx" has been added to your repositories

$ helm repo add jetstack https://charts.jetstack.io
"jetstack" has been added to your repositories

$ helm repo add hashicorp https://helm.releases.hashicorp.com
"hashicorp" has been added to your repositories

$ helm repo add grafana https://grafana.github.io/helm-charts
"grafana" has been added to your repositories

$ helm repo list
NAME                  URL
bitnami               https://charts.bitnami.com/bitnami
prometheus-community  https://prometheus-community.github.io/helm-charts
ingress-nginx         https://kubernetes.github.io/ingress-nginx
jetstack              https://charts.jetstack.io
hashicorp             https://helm.releases.hashicorp.com
grafana               https://grafana.github.io/helm-charts

$ helm repo update
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "bitnami" chart repository
...Successfully got an update from the "prometheus-community" chart repository
...
Update Complete. Happy Helming!

$ helm search repo nginx
NAME                            CHART VERSION  APP VERSION  DESCRIPTION
bitnami/nginx                   15.4.4         1.25.3       NGINX Open Source is a web server
ingress-nginx/ingress-nginx     4.10.1         1.10.1       Ingress controller for Kubernetes
```

`helm repo add` only writes to your local `~/.config/helm/repositories.yaml`. `helm repo update` actually fetches the latest `index.yaml` from each repo so search returns fresh results.

### The major public repositories

| Repo | Run by | Charts you'll use |
|---|---|---|
| Bitnami | Bitnami / VMware | postgresql, mysql, redis, mongodb, kafka, nginx, wordpress, harbor |
| prometheus-community | Prometheus Operator team | prometheus, alertmanager, grafana, kube-state-metrics, node-exporter |
| ingress-nginx | Kubernetes SIG Network | ingress-nginx (the controller) |
| jetstack | Jetstack (cert-manager folks) | cert-manager |
| hashicorp | HashiCorp | vault, consul, terraform |
| grafana | Grafana Labs | grafana, loki, tempo, mimir, promtail |
| argo | Argo project | argo-cd, argo-workflows, argo-rollouts, argo-events |

### `helm search hub`

`helm search repo` only searches repos you have added. To search every public chart on the planet, you query **Artifact Hub** (the chart-of-charts catalog).

```bash
$ helm search hub redis
URL                                                  CHART VERSION  APP VERSION  DESCRIPTION
https://artifacthub.io/packages/helm/bitnami/redis   18.13.1        7.2.4        Redis(R) is an open source...
https://artifacthub.io/packages/helm/dandydev/redis  3.2.0          7.0.5        ...
```

The URL is what you click on a browser. To `helm install`, you still need to add the repo first.

### OCI registries (since v3.8)

Since Helm 3.8 (GA), charts can also live in any OCI-compliant container registry — the same servers that hold Docker images. GitHub Container Registry, ECR, GAR, Harbor, AWS Public ECR. You don't `helm repo add` an OCI registry; you reference charts directly with `oci://`.

```bash
$ helm registry login ghcr.io
Username: bellistech
Password:
Login Succeeded

$ helm package ./mychart
Successfully packaged chart and saved it to: mychart-1.0.0.tgz

$ helm push mychart-1.0.0.tgz oci://ghcr.io/bellistech
Pushed: ghcr.io/bellistech/mychart:1.0.0
Digest: sha256:abc123...

$ helm pull oci://ghcr.io/bellistech/mychart --version 1.0.0

$ helm install myrelease oci://ghcr.io/bellistech/mychart --version 1.0.0
```

OCI is the future. New private chart distributions almost always pick OCI over the old `index.yaml` style, because you get to use the same registry you already pay for.

## helm template (render without applying)

Sometimes you want to **see** the YAML the chart would generate, without sending it to a cluster. That is what `helm template` is for. It runs the templates locally, prints the result to stdout, and exits. Nothing gets installed.

```bash
$ helm template myrelease ./mychart > rendered.yaml

$ helm template myrelease ./mychart -f my-values.yaml --set image.tag=1.27 > rendered.yaml

$ helm template myrelease bitnami/postgresql --version 13.2.7 \
    --set auth.postgresPassword=hunter2 > rendered.yaml
```

You can do `kubectl apply -f rendered.yaml` to apply it without involving Helm at all (some teams do this for stricter GitOps flows: render with Helm, commit to git, apply with Argo CD).

The `--validate` flag asks the cluster (yes, the live cluster) whether the rendered YAML is shape-correct. It is one step short of a real install.

```bash
$ helm template hello ./mychart --validate
```

If the rendered YAML has a typo like `aPiVersion: v1`, `--validate` will catch it.

`--dry-run` is the same idea but for `helm install` / `helm upgrade`: it goes through the install motions, but the cluster never actually applies the YAML.

```bash
$ helm install hello ./mychart --dry-run --debug
```

`--debug` adds verbose info about what Helm is doing, which is invaluable when you're trying to figure out why a value got the wrong shape.

## helm lint (validate)

`helm lint` runs a bunch of checks against a chart folder. Use it before you publish.

```bash
$ helm lint ./mychart
==> Linting ./mychart
[INFO] Chart.yaml: icon is recommended

1 chart(s) linted, 0 chart(s) failed

$ helm lint ./mychart --strict
```

`--strict` turns INFO/WARN into errors. Good for CI.

What it checks:

- `Chart.yaml` is well-formed (apiVersion v2, name, version, etc.)
- Templates parse as Go templates
- Templates render as valid YAML
- All template files have the required Kubernetes metadata
- `values.schema.json` (if present) matches `values.yaml`

It does not check for runtime correctness — your Deployment can still be wrong in ways the linter doesn't catch. But the obvious typos, the missing fields, the malformed YAML — all caught.

## helm dependency

A chart can depend on other charts. You declare dependencies in `Chart.yaml`:

```yaml
dependencies:
  - name: redis
    version: "^18.0.0"
    repository: https://charts.bitnami.com/bitnami
  - name: postgresql
    version: "13.x"
    repository: https://charts.bitnami.com/bitnami
    condition: postgresql.enabled
    alias: pg
```

`condition` lets users disable the dependency from values. `alias` renames the dependency inside the parent chart (useful if you need two copies of the same dep with different config).

To actually fetch them:

```bash
$ helm dependency update ./mychart
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "bitnami" chart repository
Update Complete. Happy Helming!
Saving 2 charts
Downloading redis from repo https://charts.bitnami.com/bitnami
Downloading postgresql from repo https://charts.bitnami.com/bitnami
Deleting outdated charts
```

That puts `redis-18.0.0.tgz` and `postgresql-13.4.4.tgz` in `mychart/charts/`, and writes a `Chart.lock` file pinning the exact resolved versions. Commit `Chart.lock` to git so other developers and CI get the same versions.

`helm dependency build` is similar but uses the existing `Chart.lock` instead of resolving fresh — like `npm ci` vs `npm install`.

`helm dependency list` shows what dependencies the chart declares and whether they are present in `charts/`.

When the parent chart is rendered, the subcharts are rendered too, with the parent's `.Values.<subchartName>` becoming the subchart's `.Values`. So `myredis.enabled = false` at the parent level would be `enabled = false` to the redis subchart.

## Library Charts (type: library)

Most charts are `type: application`. A library chart has `type: library` in `Chart.yaml`. Library charts:

- cannot be installed on their own
- only export named templates (in `_helpers.tpl` and friends)
- exist to share helpers across multiple application charts in your org

If you have ten internal microservice charts that all need the same labels, ingress block, image-pulling logic — extract them into a library chart and `dependencies:` it from each microservice. The microservice's templates do `{{ include "common.deployment" . }}` and the library produces the YAML.

```yaml
# Chart.yaml of the library chart
apiVersion: v2
name: common
type: library
version: 1.0.0
```

```yaml
# Chart.yaml of a microservice that uses it
apiVersion: v2
name: my-svc
type: application
version: 0.1.0
dependencies:
  - name: common
    version: "^1.0.0"
    repository: oci://ghcr.io/myorg
```

Bitnami publishes a popular `common` library chart. Many in-house platform teams build their own.

## Common Helpers (define + include)

Inside `_helpers.tpl` (or any underscored file) you `define` named templates:

```gotemplate
{{- define "mychart.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- end -}}
```

And use them in real templates:

```yaml
metadata:
  labels:
    {{- include "mychart.labels" . | nindent 4 }}
```

`include` is `template`'s younger, more capable sibling. The difference: `include` returns a string you can pipe through filters (like `nindent`). `template` is a statement, not an expression. **Always use `include` for things you indent.**

### Sprig: required and default

These two functions are the single most useful pair Sprig gives you.

```gotemplate
password: {{ required "auth.password is required" .Values.auth.password | b64enc }}
size: {{ default "10Gi" .Values.persistence.size }}
```

`required "msg" v` says "if `v` is missing, fail the render with `msg`." Use this for things the user **must** set.

`default "x" v` says "if `v` is empty, use `x`." Use this for sensible fallbacks.

### Sprig: lookup

`lookup` reads live state from the cluster while rendering.

```gotemplate
{{- $existing := lookup "v1" "Secret" .Release.Namespace "myapp-secret" }}
{{- if $existing }}
data:
  password: {{ index $existing.data "password" }}
{{- else }}
data:
  password: {{ randAlphaNum 32 | b64enc }}
{{- end }}
```

The first install generates a random password and stores it in a Secret. The second install **reads** the Secret and reuses it, so the password is stable across upgrades. `lookup` returns nil when the resource doesn't exist (or during `helm template` without `--validate`, since there's no cluster).

### Sprig: fromYaml / toJson

These move between formats.

```gotemplate
{{- $cfg := fromYaml (.Files.Get "config.yaml") }}
data:
  config.json: {{ $cfg | toJson | quote }}
```

Or the safer "must" variants which fail loudly on parse errors instead of returning empty:

```gotemplate
{{- $cfg := mustFromYaml (.Files.Get "config.yaml") }}
data:
  config.json: {{ $cfg | mustToJson | quote }}
```

## Hooks

A hook is a Kubernetes resource that runs at a specific point in the release lifecycle. You declare a resource as a hook by adding the annotation `helm.sh/hook`:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migrate
  annotations:
    "helm.sh/hook": pre-upgrade,pre-install
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": before-hook-creation
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: myapp:{{ .Chart.AppVersion }}
          command: ["./migrate.sh"]
      restartPolicy: Never
```

When this chart is installed or upgraded, Helm runs the migration Job before any other resources are applied. Once the Job succeeds, Helm proceeds with the rest of the install.

### Hook events

| Hook | When |
|---|---|
| `pre-install` | before any resources are created on first install |
| `post-install` | after all resources are created on first install |
| `pre-upgrade` | before any resource changes during upgrade |
| `post-upgrade` | after all resource changes during upgrade |
| `pre-delete` | before resources are deleted by uninstall |
| `post-delete` | after resources are deleted by uninstall |
| `pre-rollback` | before rollback resource changes |
| `post-rollback` | after rollback resource changes |
| `test` | runs only when you call `helm test` |

### Hook weights

Multiple hooks for the same event are sorted by `helm.sh/hook-weight` (a string-encoded integer, lower runs first). Within the same weight, sort by name.

### Hook delete policies

`helm.sh/hook-delete-policy` controls when Helm cleans up a hook resource:

| Policy | Behavior |
|---|---|
| `before-hook-creation` (default) | delete the previous hook of this name before creating a new one |
| `hook-succeeded` | delete after the hook succeeds |
| `hook-failed` | delete after the hook fails |

You can list multiple comma-separated. A common combo: `hook-succeeded,before-hook-creation`.

### Hook lifecycle diagram

```
helm install
  |
  v
+---------+    +-------------+    +-----------+    +--------------+
|  CRDs   | -> | pre-install | -> | manifests | -> | post-install |
+---------+    +-------------+    +-----------+    +--------------+
                  (weight                              (weight
                   sorted)                              sorted)

helm upgrade
  |
  v
+-------------+    +-----------+    +--------------+
| pre-upgrade | -> | manifests | -> | post-upgrade |
+-------------+    +-----------+    +--------------+

helm test
  |
  v
+------+
| test |
+------+
```

Hooks run as ordinary Kubernetes resources (most often Jobs, sometimes Pods). Helm watches them and waits for success before moving on.

## Test Hooks (helm test)

A test hook is just a hook with `helm.sh/hook: test`. The chart scaffold puts one in `templates/tests/test-connection.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: "{{ include "mychart.fullname" . }}-test-connection"
  annotations:
    "helm.sh/hook": test
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args: ['{{ include "mychart.fullname" . }}:{{ .Values.service.port }}']
  restartPolicy: Never
```

Run it with `helm test`:

```bash
$ helm test hello
NAME: hello
LAST DEPLOYED: Mon Apr 27 10:30:00 2026
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE:     hello-test-connection
Last Started:   Mon Apr 27 10:31:12 2026
Last Completed: Mon Apr 27 10:31:18 2026
Phase:          Succeeded
```

If the test pod's container exits 0, the test passed. Non-zero, it failed. Helm prints the result.

`--logs` includes the test pod's stdout/stderr in the output:

```bash
$ helm test hello --logs
```

Tests are great in CI. Many production charts include real connection tests, schema-version checks, and smoke probes as test hooks.

## Schema Validation (values.schema.json)

You can ship a JSON Schema next to `values.yaml` named `values.schema.json`. Helm validates the user's merged values against it before rendering.

```json
{
  "$schema": "https://json-schema.org/draft-07/schema",
  "type": "object",
  "required": ["replicaCount", "image"],
  "properties": {
    "replicaCount": {
      "type": "integer",
      "minimum": 1
    },
    "image": {
      "type": "object",
      "required": ["repository"],
      "properties": {
        "repository": { "type": "string" },
        "tag": { "type": "string" },
        "pullPolicy": {
          "type": "string",
          "enum": ["Always", "IfNotPresent", "Never"]
        }
      }
    }
  }
}
```

If a user passes `replicaCount: -3` or `image.pullPolicy: Yes`, Helm refuses to install with a message like `Error: values don't meet the specifications of the schema(s)`.

This is the cleanest way to give users early, readable feedback about misconfigured values.

## Helm 3 vs Helm 2 (no Tiller)

You may stumble across blog posts and tutorials from years ago that talk about a thing called **Tiller.** Forget Tiller. Tiller is dead.

Helm 2 had a server-side component running in your cluster called Tiller. Tiller did the actual work of installing charts. Tiller had a giant, unscoped service account because it had to install anything anywhere. Tiller was also a security nightmare. Anybody who could talk to Tiller could install anything cluster-wide.

**Helm 3 (November 2019)** removed Tiller entirely. Now Helm is just a CLI. The CLI talks to the cluster's API server with **your** credentials and **your** RBAC. Releases are stored as Secrets directly in the cluster. There is no server.

| Thing | Helm 2 | Helm 3 |
|---|---|---|
| Server side | Tiller (in cluster) | none |
| RBAC | Tiller's | yours |
| Release storage | ConfigMap in `kube-system` | Secret in release namespace |
| Chart format | apiVersion v1 | apiVersion v2 |
| Three-way strategic merge | no | yes |
| OCI registry support | no | yes (since 3.8) |

### Version timeline

| Helm version | Year | Notable |
|---|---|---|
| 3.0.0 GA | Nov 2019 | No Tiller, release Secrets, apiVersion v2 |
| 3.1 | Feb 2020 | `--post-renderer` |
| 3.5 | Jan 2021 | `helm pull`, removal of `helm push` (came back via OCI) |
| 3.7 | Sep 2021 | OCI registries (experimental) |
| 3.8 | Jan 2022 | OCI GA |
| 3.10 | Sep 2022 | KubeVersion checks |
| 3.13 | Oct 2023 | Status drift detection |
| 3.14 | Feb 2024 | `--post-renderer-args` |

## Common Patterns

### Umbrella chart

The umbrella pattern: one chart that does almost no work itself, but lists many subcharts as dependencies. You install **the umbrella**, and you get the whole stack at once.

```yaml
# umbrella/Chart.yaml
apiVersion: v2
name: my-platform
version: 0.1.0
dependencies:
  - name: postgresql
    version: "^13"
    repository: https://charts.bitnami.com/bitnami
  - name: redis
    version: "^18"
    repository: https://charts.bitnami.com/bitnami
  - name: nginx
    version: "^15"
    repository: https://charts.bitnami.com/bitnami
  - name: my-app
    version: "0.1.0"
    repository: file://../my-app
```

Then `helm dependency update ./umbrella` and `helm install platform ./umbrella` brings up the whole stack as **one release** named `platform`.

```
+--------- platform (release) ---------+
|                                       |
|  +-----------+  +-------+  +------+   |
|  | postgres  |  | redis |  | nginx|   |
|  +-----------+  +-------+  +------+   |
|              +-----------+            |
|              |  my-app   |            |
|              +-----------+            |
+---------------------------------------+
```

### Multi-environment values files

The pattern almost everyone uses: keep one base `values.yaml` plus per-environment overrides.

```
deploy/
+-- values.yaml         # baseline / defaults
+-- values-dev.yaml
+-- values-staging.yaml
+-- values-prod.yaml
+-- secrets-prod.enc.yaml   # encrypted with helm-secrets/sops
```

```bash
$ helm upgrade --install myapp ./chart \
    -f deploy/values.yaml \
    -f deploy/values-prod.yaml \
    -f deploy/secrets-prod.enc.yaml \
    --set image.tag=$GIT_SHA
```

Order matters. **Later wins.** Conceptual precedence chain:

```
chart's values.yaml
    |
    v  (overlaid by)
-f file 1
    |
    v
-f file 2
    |
    v
--set / --set-string / --set-file
    |
    v
final .Values seen by templates
```

### --set syntax

`--set` is one-line value-setting. Nested keys use dots:

```bash
helm install x ./chart --set image.repository=nginx --set image.tag=1.27
```

Lists use indexed brackets:

```bash
helm install x ./chart --set ports[0].name=http --set ports[0].port=80
```

To set a key that **literally contains a dot**, escape it with a backslash:

```bash
helm install x ./chart --set 'annotations.app\.kubernetes\.io/name=foo'
```

`--set-string` forces string type (because `--set version=3.14` would parse 3.14 as a float, but `--set-string version=3.14` keeps it as the string `"3.14"`).

`--set-file` reads the value from a file (great for cert blobs):

```bash
helm install x ./chart --set-file 'tls.crt=./cert.pem'
```

### --reuse-values, --reset-values, --reset-then-reuse-values

When you `helm upgrade`, the question of "should the previous release's values be remembered, or do we start fresh?" matters.

| Flag | Behavior |
|---|---|
| (default) | use chart defaults + your `-f`/`--set` (does **not** auto-reuse) |
| `--reuse-values` | start from previous release's merged values, then apply `-f`/`--set` |
| `--reset-values` | explicit version of the default |
| `--reset-then-reuse-values` (3.14+) | reset to chart defaults, then layer on previous user values, then your new flags |

If you upgrade a release without `-f` or `--set`, on Helm 3 the default behavior is `--reset-values` — you go back to chart defaults. If you only meant to bump the chart version, this can wipe your overrides. The fix: pass `--reuse-values` or always pass your full values file.

### Post-renderer (Kustomize integration)

Sometimes a chart almost does what you want except for one tiny thing — say it's missing a sidecar container, or it sets a label you don't want. Forking the chart is overkill. **Post-renderers** let you pipe Helm's rendered output through any program before it gets applied:

```bash
helm install x ./chart --post-renderer ./kustomize.sh
```

`kustomize.sh` is a tiny shell script that takes YAML on stdin, applies a Kustomize overlay, and writes the result on stdout. The pattern:

```bash
#!/usr/bin/env bash
cat > base.yaml
cd overlay && kustomize build .
```

3.14+ added `--post-renderer-args` so you don't have to hand-craft a script.

## Common Errors

These are real strings you will see. Memorize the fixes.

### `Error: INSTALLATION FAILED: Kubernetes cluster unreachable`

```
$ helm install hello ./mychart
Error: INSTALLATION FAILED: Kubernetes cluster unreachable: Get "https://10.0.0.1:6443/version": dial tcp 10.0.0.1:6443: i/o timeout
```

Helm cannot reach the cluster. Check:

- `kubectl cluster-info` — does kubectl work? If not, fix that first.
- `kubectl config current-context` — are you on the right cluster?
- `KUBECONFIG` env var — pointing at the right file?
- Network/VPN — sometimes the cluster only exists inside a VPN.

### `Error: chart "X" version "Y" not found`

```
$ helm install x bitnami/redis --version 99.0.0
Error: chart "redis" version "99.0.0" not found in https://charts.bitnami.com/bitnami repository
```

Usually means:

- the version doesn't exist (typo, or you're imagining it)
- you forgot `helm repo update` (the local `index.yaml` is stale)
- the chart was yanked

Fix: `helm repo update`, then `helm search repo redis --versions` to see what's actually available.

### `Error: failed to download "X"`

```
$ helm install x oci://ghcr.io/me/mychart --version 1.0.0
Error: failed to download "oci://ghcr.io/me/mychart" (hint: running `helm repo update` may help)
```

Network problem or auth problem. For OCI, `helm registry login ghcr.io` first.

### `Error: rendered manifests contain a resource that already exists`

```
Error: rendered manifests contain a resource that already exists. Unable to continue with install: existing resource conflict: namespace: default, name: hello-svc, existing_kind: /v1, Kind=Service, new_kind: /v1, Kind=Service
```

You're trying to install a release whose YAML would create a resource that's already in the cluster but **not owned by Helm.** Maybe you ran `kubectl apply -f` earlier, or there's a leftover from a deleted release.

Fix:

- `kubectl delete <kind> <name>` to remove the squatter, then retry, or
- adopt the resource by adding the right `app.kubernetes.io/managed-by=Helm` and `meta.helm.sh/release-name`/`meta.helm.sh/release-namespace` annotations and labels.

### `Error: UPGRADE FAILED: another operation (install/upgrade/rollback) is in progress`

```
Error: UPGRADE FAILED: another operation (install/upgrade/rollback) is in progress
```

Helm thinks an install/upgrade is already running for this release. Maybe a previous run got killed. The release is stuck in a `pending-*` state.

Fix:

```bash
$ helm history myrelease -n myns
# look for a revision in pending-upgrade / pending-install
$ helm rollback myrelease <last-good-revision> -n myns
# or, if it's a fresh install that died:
$ kubectl delete secret -l owner=helm,name=myrelease -n myns
$ helm install myrelease ... # try again
```

### `Error: cannot re-use a name that is still in use`

```
$ helm install hello ./mychart
Error: INSTALLATION FAILED: cannot re-use a name that is still in use
```

A release with that name already exists. Either:

- pick a different name, or
- `helm uninstall hello` first, or
- use `helm upgrade --install hello ./mychart` (idempotent — install if missing, upgrade if present).

### `Error: render error in "X.tpl": template: helpers.tpl:N:M: executing "Y" at <.Values.foo>: nil pointer evaluating`

```
Error: render error in "mychart/templates/deployment.yaml": template: mychart/templates/_helpers.tpl:23:14: executing "mychart.fullname" at <.Values.fullnameOverride>: nil pointer evaluating interface {}.fullnameOverride
```

The template referenced `.Values.foo`, but `foo` is a sub-key of an object that doesn't exist. Sprig is unforgiving about that.

Fix: use `default` or guard with `if`:

```gotemplate
{{ default "" .Values.image.tag }}
{{- if .Values.someOptional }}{{ .Values.someOptional.field }}{{- end }}
```

Or assert presence with `required`:

```gotemplate
{{ required ".Values.foo is required" .Values.foo }}
```

### `Error: values don't meet the specifications of the schema(s)`

```
Error: values don't meet the specifications of the schema(s) in the following chart(s):
mychart:
- replicaCount: Must be greater than or equal to 1
```

You have a `values.schema.json`, and the user's values violated it. Read the message; it usually points right at the field.

### `Error: invalid argument "X" for "--set" flag`

```
Error: invalid argument "image:nginx" for "--set" flag: parse error at (string:1:6): unexpected ":" in operand
```

`--set key=value` uses `=`, not `:` (which is YAML, not Helm `--set` syntax). The fix is `--set image=nginx` or `--set image.repository=nginx`.

### `Error: unable to build kubernetes objects from release manifest`

```
Error: UPGRADE FAILED: unable to build kubernetes objects from release manifest: error validating "": error validating data: ValidationError(Deployment.spec.template.spec.containers[0]): unknown field "imag" in io.k8s.api.core.v1.Container
```

The rendered YAML is shape-correct YAML but invalid Kubernetes (typo: `imag` instead of `image`). Run `helm template` and check the rendered output — your eyes will spot the typo faster than the parser.

`--validate` on `helm template` catches these before install.

## Hands-On

Type each command. Read the output. Try not to skip; the muscle memory is the point.

### Bootstrap

```bash
# version check
$ helm version
version.BuildInfo{Version:"v3.14.4", GitCommit:"...", GitTreeState:"clean", GoVersion:"go1.21.9"}

# add the popular repos
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
$ helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
$ helm repo add jetstack https://charts.jetstack.io
$ helm repo add grafana https://grafana.github.io/helm-charts

# pull latest indexes
$ helm repo update
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "bitnami" chart repository
...
Update Complete. Happy Helming!

# search local + hub
$ helm search repo nginx
$ helm search repo redis --versions
$ helm search hub redis
```

### Make and inspect a chart

```bash
$ helm create skeleton
Creating skeleton

$ ls skeleton
Chart.yaml  charts  templates  values.yaml

$ helm lint ./skeleton
==> Linting ./skeleton
[INFO] Chart.yaml: icon is recommended
1 chart(s) linted, 0 chart(s) failed

$ helm template release1 ./skeleton > rendered.yaml
$ wc -l rendered.yaml
85 rendered.yaml

$ helm template release1 ./skeleton --validate
# (validates against the live cluster; needs KUBECONFIG)

$ helm template release1 ./skeleton --set service.type=NodePort | grep -A 2 'kind: Service'
kind: Service
metadata:
  name: release1-skeleton
```

### Install, inspect, upgrade, roll back, uninstall

```bash
# idempotent install/upgrade
$ helm upgrade --install hello ./skeleton -n demo --create-namespace --wait --timeout 5m
Release "hello" does not exist. Installing it now.
NAME: hello
LAST DEPLOYED: Mon Apr 27 10:40:18 2026
NAMESPACE: demo
STATUS: deployed
REVISION: 1

$ helm ls -A
NAME    NAMESPACE  REVISION  UPDATED                              STATUS    CHART              APP VERSION
hello   demo       1         2026-04-27 10:40:18.456 +0000 UTC    deployed  skeleton-0.1.0     1.16.0

$ helm ls -A --output json | jq '.[].name'
"hello"

$ helm status hello -n demo
NAME: hello
LAST DEPLOYED: Mon Apr 27 10:40:18 2026
NAMESPACE: demo
STATUS: deployed
REVISION: 1

$ helm history hello -n demo
REVISION  UPDATED                   STATUS    CHART           APP VERSION  DESCRIPTION
1         2026-04-27 10:40:18 UTC   deployed  skeleton-0.1.0  1.16.0       Install complete

$ helm get values hello -n demo
USER-SUPPLIED VALUES:
null

$ helm get values hello -n demo --all
COMPUTED VALUES:
affinity: {}
autoscaling:
  enabled: false
  ...

$ helm get manifest hello -n demo | head -20

$ helm get hooks hello -n demo

$ helm get all hello -n demo > /tmp/everything.yaml

# upgrade
$ helm upgrade --install hello ./skeleton -n demo --set replicaCount=3 --wait

$ helm history hello -n demo
REVISION  UPDATED                   STATUS      CHART           APP VERSION  DESCRIPTION
1         2026-04-27 10:40:18 UTC   superseded  skeleton-0.1.0  1.16.0       Install complete
2         2026-04-27 10:42:01 UTC   deployed    skeleton-0.1.0  1.16.0       Upgrade complete

# roll back
$ helm rollback hello 1 -n demo
Rollback was a success! Happy Helming!

# run tests
$ helm test hello -n demo

# uninstall
$ helm uninstall hello -n demo
release "hello" uninstalled

# uninstall but keep history
$ helm uninstall hello -n demo --keep-history
```

### Real chart from Bitnami

```bash
# install Redis with a custom password and a values file
$ cat > /tmp/myredis.yaml <<EOF
auth:
  password: hunter2
master:
  persistence:
    size: 2Gi
EOF

$ helm install myredis bitnami/redis \
    --version 18.x \
    --values /tmp/myredis.yaml \
    --set auth.password=hunter2 \
    --create-namespace -n redis-test

$ helm ls -n redis-test
NAME     NAMESPACE   REVISION  UPDATED                            STATUS    CHART         APP VERSION
myredis  redis-test  1         2026-04-27 10:50:00 UTC            deployed  redis-18.X.Y  7.x.y

$ kubectl get pods -n redis-test
```

### Dry-run and debug

```bash
$ helm install hello ./skeleton --dry-run --debug 2>&1 | head -40
install.go:200: [debug] Original chart version: ""
install.go:217: [debug] CHART PATH: /home/me/skeleton
NAME: hello
LAST DEPLOYED: Mon Apr 27 10:55:00 2026
...
HOOKS:
---
# Source: skeleton/templates/tests/test-connection.yaml
apiVersion: v1
kind: Pod
metadata:
  name: "hello-skeleton-test-connection"
  ...
MANIFEST:
---
# Source: skeleton/templates/serviceaccount.yaml
...
```

### Dependencies

```bash
$ cat >> ./skeleton/Chart.yaml <<EOF
dependencies:
  - name: redis
    version: "^18.0.0"
    repository: https://charts.bitnami.com/bitnami
EOF

$ helm dependency update ./skeleton
Saving 1 charts
Downloading redis from repo https://charts.bitnami.com/bitnami

$ ls ./skeleton/charts
redis-18.X.Y.tgz

$ helm dependency build ./skeleton
$ helm dependency list ./skeleton
NAME    VERSION   REPOSITORY                            STATUS
redis   ^18.0.0   https://charts.bitnami.com/bitnami    ok
```

### Package and OCI push

```bash
$ helm package ./skeleton
Successfully packaged chart and saved it to: skeleton-0.1.0.tgz

$ helm registry login ghcr.io
Username: bellistech
Password:
Login Succeeded

$ helm push skeleton-0.1.0.tgz oci://ghcr.io/bellistech
Pushed: ghcr.io/bellistech/skeleton:0.1.0
Digest: sha256:abc...

$ helm pull oci://ghcr.io/bellistech/skeleton --version 0.1.0
$ ls
skeleton-0.1.0.tgz

$ helm install fromoci oci://ghcr.io/bellistech/skeleton --version 0.1.0
```

### Plugins

```bash
$ helm plugin install https://github.com/databus23/helm-diff
Installed plugin: diff

$ helm diff upgrade hello ./skeleton --set replicaCount=4 -n demo
demo, hello-skeleton, Deployment (apps) has changed:
  ...
-     replicas: 3
+     replicas: 4
  ...

$ helm plugin list
NAME  VERSION  DESCRIPTION
diff  3.x.y    Preview helm upgrade changes as a diff
```

### Look at the release Secrets

```bash
$ kubectl get secret -l owner=helm -A
NAMESPACE  NAME                          TYPE                 DATA   AGE
demo       sh.helm.release.v1.hello.v1   helm.sh/release.v1   1      30m
demo       sh.helm.release.v1.hello.v2   helm.sh/release.v1   1      28m
demo       sh.helm.release.v1.hello.v3   helm.sh/release.v1   1      27m
```

That's Helm's database, right there in the cluster.

### Alternatives worth knowing about

```bash
# helmfile — declarative wrapper, "kubectl-style" file describing many releases
$ helmfile sync
# Reads helmfile.yaml that lists multiple releases and brings the cluster to that state.

# helmsman — similar declarative wrapper, single binary
$ helmsman --apply -f desired-state.yaml
```

You don't need helmfile or helmsman for everyday use; just know they exist if you have many releases to manage.

## Common Confusions

A bunch of paired pitfalls people hit in the first month of Helm.

### `helm install` vs `helm upgrade --install`

`helm install` fails if the release exists. `helm upgrade --install` (sometimes pronounced "upsert") installs if absent, upgrades if present. **In CI, always use `helm upgrade --install`.**

### `--set` and dotted keys

`--set foo.bar=baz` sets `.Values.foo.bar`. To set a **literal** key with a dot in it (like an annotation key), escape: `--set 'annotations.app\.kubernetes\.io/name=foo'`. Forgetting to escape is the cause of about 30% of mysterious "value not found" errors.

### `--values` precedence

Multiple `-f` files: **later wins** on conflict. `--set` wins over all `-f`. `--set-string` wins over `--set` (last-set).

### `Chart.yaml dependencies` vs `helm dependency update`

`Chart.yaml dependencies:` is the **declaration.** `helm dependency update` is the **command** that reads the declaration and downloads the dependency tarballs into `charts/`. You need both.

### Library charts vs application charts

Application chart = installable. Library chart = template-only, not installable. If you `helm install` a library chart, you get `Error: library charts cannot be used in this context`.

### `{{ .Values.foo | quote }}` vs `"{{ .Values.foo }}"`

`{{ .Values.foo | quote }}` produces a quoted string and **escapes any internal quotes correctly.** `"{{ .Values.foo }}"` looks similar but if `.Values.foo` contains a quote or a newline, it produces broken YAML. Always prefer `| quote`.

### `required` vs `default`

`required` errors out if the value is missing. `default` substitutes a fallback. Use `required` for things the user must set; use `default` for sensible automatic values.

### `lookup` returns nil when there's no cluster

During `helm template` (no cluster), `lookup` returns nil. Don't write code that crashes on nil — write `{{- if $obj }}...{{- end }}`.

### Namespace pinning in templates

If a template hardcodes `namespace: prod`, your release can be installed into namespace `prod` but the resources will all live in `prod` regardless of the user's `-n` flag. **Don't hardcode namespaces.** Use `{{ .Release.Namespace }}` and let users target where they want.

### `helm test` pod lifecycle

By default, the test pods stick around after running. To clean up, set `helm.sh/hook-delete-policy: hook-succeeded`. Also: rerunning `helm test` reruns **all** test hooks; there's no per-test selection.

### Release storage as Secret

In Helm 3 every release revision is a Kubernetes Secret in the release's namespace. **If you `kubectl delete ns demo`, you delete all of `demo`'s release history with it.** That's why `helm uninstall` exists: it cleans up neatly.

### Multi-version release history

By default Helm keeps the last 10 revisions. You can change with `--history-max` on install/upgrade. Helm prunes oldest first. A release that has been `superseded` (older revisions) is fine to leave around — it just provides a rollback target.

### Subchart values

To set a value on a subchart, prefix with the subchart's name in your values:

```yaml
# parent values.yaml
redis:
  auth:
    password: hunter2
```

That sets `.Values.auth.password` from the redis subchart's perspective.

### `--create-namespace`

`helm install -n foo --create-namespace` creates namespace `foo` if missing. Without this flag, `helm install -n foo` against a missing namespace fails with `Error: ... namespaces "foo" not found`.

### `--no-hooks` and `--skip-tests`

`--no-hooks` skips **all** hooks. Useful when a hook is broken and you need to deploy anyway. `--skip-tests` is specifically for `helm test`.

### `--skip-crds` vs `--include-crds`

`--skip-crds` (install/upgrade) skips installing files in `crds/`. `--include-crds` (template only) **includes** them in rendered output (since `crds/` resources aren't part of normal templates, they're not rendered by default).

## Vocabulary

A long table. If a word in this sheet didn't quite click, look it up here.

| Word | Meaning |
|---|---|
| helm | the CLI and the project; a package manager for Kubernetes |
| Helm 3 | current major version (since November 2019); no Tiller; release stored as Secret |
| Helm 2 | the legacy version; deprecated; required Tiller in cluster; do not use |
| Tiller | Helm 2's server-side component, removed in Helm 3 |
| chart | a folder of templates + Chart.yaml that Helm renders and installs |
| Chart.yaml | the chart's metadata file (name, version, deps, etc.) |
| apiVersion v2 | the chart format version used by Helm 3 |
| type: application | normal installable chart |
| type: library | chart that exports helpers only; cannot be installed |
| name | the chart name in Chart.yaml |
| version | the chart version (semver); bumps when templates change |
| appVersion | the version of the upstream application the chart packages |
| dependencies | other charts your chart depends on (subcharts) |
| templates/ | directory holding the templated YAML resources |
| values.yaml | default values fed into templates |
| values.schema.json | optional JSON Schema validating user values |
| _helpers.tpl | conventional file holding named template snippets |
| _helpers.yaml | rare alternate name for the same idea |
| .helmignore | gitignore-like file controlling what's packaged |
| NOTES.txt | template printed to user after install/upgrade |
| charts/ | subchart tarballs live here after `helm dependency update` |
| crds/ | CustomResourceDefinitions installed once, never templated |
| README.md | rendered on Artifact Hub as the chart's docs |
| requirements.yaml | deprecated Helm 2 dependency file; replaced by `dependencies:` in Chart.yaml |
| repository | a website that hosts charts (HTTPS or OCI) |
| helm repo | the subcommand for managing repositories |
| helm hub | legacy chart catalog, replaced by Artifact Hub |
| Artifact Hub | the global directory of public charts (artifacthub.io) |
| OCI registry | container-image-style chart hosting (Helm 3.8+) |
| OCI helm chart | a chart pushed to an OCI registry |
| oci:// | URL scheme used to reference OCI charts |
| helm registry login | `docker login`-equivalent for OCI registries |
| helm push | upload a chart .tgz to an OCI registry |
| helm pull | download a chart .tgz from a repo or OCI registry |
| helm package | turn a chart folder into a `.tgz` archive |
| .tgz chart archive | the file format charts are distributed as |
| release | an instance of a chart installed in a cluster |
| release name | user-chosen identifier for a release |
| release namespace | namespace the release lives in |
| release revision | counter that increments on each upgrade or rollback |
| release status | current lifecycle state (deployed, failed, superseded, ...) |
| deployed | release status: currently active |
| failed | release status: install/upgrade failed |
| superseded | release status: prior revision, no longer active |
| uninstalled | release status: removed but kept in history |
| uninstalling | release status: in the middle of being removed |
| pending-install | release status: install in progress |
| pending-upgrade | release status: upgrade in progress |
| pending-rollback | release status: rollback in progress |
| HISTORY_MAX | env var or alias for the per-release history cap |
| --history-max | flag setting how many revisions to keep |
| --keep-history | uninstall flag: keep the release record around |
| --no-hooks | skip running any hooks |
| --skip-crds | skip resources in `crds/` |
| --include-crds | include CRDs in rendered output (template-only) |
| --skip-tests | skip resources annotated as `helm.sh/hook: test` |
| --reuse-values | upgrade: start from prior release's values |
| --reset-values | upgrade: start from chart defaults (default behavior) |
| --reset-then-reuse-values | upgrade: defaults then prior values; layered with new flags (3.14+) |
| --atomic | install/upgrade: undo on failure |
| --cleanup-on-fail | clean partial resources on failure (less aggressive than --atomic) |
| --wait | block until resources are ready |
| --wait-for-jobs | also wait for Job completion |
| --timeout | how long to wait before giving up |
| --post-renderer | pipe rendered output through a program |
| --post-renderer-args | extra args to pass the post-renderer (3.14+) |
| post-renderer | the integration pattern (often Kustomize) |
| helm-diff plugin | shows what an upgrade *would* change |
| helmfile | external tool for declarative multi-release management |
| helmsman | external tool similar to helmfile |
| helm-secrets plugin | sops-based encryption of values files |
| capabilities | runtime info about the cluster Helm is talking to |
| .Capabilities.APIVersions | available API groups/kinds in the cluster |
| .Capabilities.KubeVersion | the cluster's Kubernetes version |
| .Capabilities.HelmVersion | the running Helm version |
| .Release.Name | the release name |
| .Release.Namespace | the release namespace |
| .Release.IsInstall | true on first install |
| .Release.IsUpgrade | true on upgrade |
| .Release.Revision | the release revision number |
| .Chart.Name | from Chart.yaml |
| .Chart.Version | from Chart.yaml |
| .Chart.AppVersion | from Chart.yaml |
| .Files.Get | read a chart-bundled file as string |
| .Files.GetBytes | read a chart-bundled file as bytes |
| .Files.AsConfig | turn a glob of files into ConfigMap data |
| .Files.AsSecrets | turn a glob of files into Secret data |
| .Files.Glob | match a glob pattern of chart files |
| .Values | the merged values root |
| .Subcharts | subchart-specific values from the parent's perspective |
| .Template.Name | full path of the template currently rendering |
| .Template.BasePath | base path of the templates directory |
| Sprig | the bundled function library Helm uses for template helpers |
| default | Sprig: fallback when value is empty |
| required | Sprig: error if value is missing |
| hasKey | Sprig: does a map have a given key |
| lookup | Sprig: read live cluster state during render |
| regexMatch | Sprig: regex test |
| regexReplaceAll | Sprig: regex global replace |
| fromYaml | Sprig: parse YAML into a structure |
| toYaml | Sprig: serialize structure to YAML |
| toJson | Sprig: serialize to JSON |
| fromJson | Sprig: parse JSON |
| b64enc | Sprig: base64 encode |
| b64dec | Sprig: base64 decode |
| sha256sum | Sprig: SHA-256 hex digest of input |
| randAlphaNum | Sprig: random N-character alphanumeric string |
| mustToJson | Sprig: like toJson but errors on failure |
| mustFromYaml | Sprig: like fromYaml but errors on parse fail |
| define | Go template: declare a named template snippet |
| include | Go template helper: invoke a named template, returning string |
| template | Go template: invoke a named template (statement form) |
| range | Go template: loop over a list or map |
| if | Go template: conditional |
| else if | Go template: chained conditional |
| with | Go template: set scope to expression's value |
| tpl | Helm: render a string at runtime as a template |
| hash function | any Sprig hashing helper (sha256sum, md5sum, etc.) |
| indent | Sprig: indent every line by N spaces |
| nindent | Sprig: like indent, but prepends a newline |
| trimSuffix | Sprig: trim trailing string |
| trimPrefix | Sprig: trim leading string |
| semverCompare | Sprig: compare semver strings |
| hooks | resources annotated to run at lifecycle points |
| pre-install | hook: before resources on install |
| post-install | hook: after resources on install |
| pre-delete | hook: before resources on uninstall |
| post-delete | hook: after resources on uninstall |
| pre-upgrade | hook: before resources on upgrade |
| post-upgrade | hook: after resources on upgrade |
| pre-rollback | hook: before resources on rollback |
| post-rollback | hook: after resources on rollback |
| test | hook: runs only on `helm test` |
| helm.sh/hook | annotation that marks a resource as a hook |
| helm.sh/hook-weight | annotation that orders multiple hooks for the same event |
| helm.sh/hook-delete-policy | annotation that controls hook resource cleanup |
| before-hook-creation | hook delete policy: delete prior copy before recreating |
| hook-succeeded | hook delete policy: delete after success |
| hook-failed | hook delete policy: delete after failure |
| values precedence chain | chart defaults < -f files in order < --set/-set-string/--set-file |
| dry-run | install/upgrade flag: don't actually apply |
| debug | install/upgrade flag: verbose output |
| plugins | external commands installed under `helm plugin` |
| helm secrets | popular plugin for sops-encrypted values files |
| helm diff | popular plugin showing upgrade preview |
| helm git | plugin for git-based repos |
| helm-monitor | plugin for monitoring deploy success |
| helm-x509 | plugin for managing x509 certs in charts |
| --set | inline value on CLI |
| --set-string | inline value, force string type |
| --set-file | inline value sourced from a file |
| list (helm ls) | list releases |
| namespace scoping | the `-n` flag confines operations to a namespace |
| --all-namespaces | the `-A` flag, list across all namespaces |
| JSON Schema for values | validation grammar for values.yaml |
| schema validation | the act of checking values against the schema |
| library chart usage | depend on a library chart and `include` its helpers |
| application chart usage | install an application chart as a release |
| app.kubernetes.io/managed-by=Helm | label Helm stamps on every resource it owns |
| ownership labels | the `app.kubernetes.io/*` family Helm sets by convention |
| ChartCenter | deprecated chart catalog (gone) |
| distributing private charts via OCI | the modern way to host private charts |
| IRSA | AWS pattern: pod IAM roles via service accounts |
| Workload Identity | GCP equivalent of IRSA |
| GitOps integration | Argo CD or Flux reading Helm charts from git and applying |
| Argo CD | GitOps controller, can render Helm charts |
| Flux helm controllers | GitOps controller, has a HelmRelease CRD |
| umbrella chart | a chart that's mostly subcharts |
| subchart | a chart used as a dependency of another chart |
| subchart exposed values | the values a parent passes through to a subchart by namespacing under the subchart's name |

## Try This

A short ladder of exercises. Do them in order. Each one builds confidence.

1. `helm version`. Make sure you're on 3.10+ minimum, ideally 3.14+.
2. `helm repo add bitnami https://charts.bitnami.com/bitnami` and `helm repo update`.
3. `helm search repo redis --versions | head -5` — confirm you see versions.
4. `helm create lab` — make a starter chart.
5. `helm template foo ./lab > /tmp/lab.yaml` — render without applying. Open `/tmp/lab.yaml` and read it. Find the labels block. Find the deployment. Find the service.
6. `helm lint ./lab` — confirm clean.
7. `helm install lab ./lab -n labspace --create-namespace --wait` — real install.
8. `helm ls -n labspace` — confirm release exists.
9. `kubectl get all -n labspace` — confirm pod, service, deployment exist.
10. Edit `lab/values.yaml`, change `replicaCount: 1` to `replicaCount: 2`. `helm upgrade --install lab ./lab -n labspace --wait`.
11. `helm history lab -n labspace` — see two revisions.
12. `helm rollback lab 1 -n labspace` — back to one replica.
13. `helm test lab -n labspace` — run the test pod.
14. `helm uninstall lab -n labspace` — cleanup.
15. `kubectl delete ns labspace`.
16. Now do the same with a real chart: `helm install demo bitnami/nginx --create-namespace -n demo --wait`. Then port-forward and curl it. `helm uninstall demo -n demo`.

That sequence — install, look at YAML, upgrade, roll back, test, uninstall — is the daily reality of Helm. Once you can do it without thinking about it, you're fluent.

## Where to Go Next

- `cs orchestration helm` — the precision reference for daily Helm work.
- `cs orchestration kubernetes` — the underlying k8s reference if you need depth on a Pod, Deployment, Service, etc.
- `cs orchestration kubectl` — the kubectl reference (you'll be using it constantly alongside helm).
- `cs orchestration kustomize` — the alternative-but-also-complementary tool.
- `cs orchestration argocd` — when you outgrow CLI helm and want continuous reconciliation.
- `cs orchestration external-secrets` — for keeping secrets out of values files.
- `cs ramp-up kubernetes-eli5` — review if any k8s concept above felt foggy.

## See Also

- orchestration/helm
- orchestration/kubernetes
- orchestration/kubectl
- orchestration/kustomize
- orchestration/argocd
- orchestration/external-secrets
- ramp-up/kubernetes-eli5
- ramp-up/docker-eli5
- ramp-up/linux-kernel-eli5

## References

- helm.sh/docs — the official documentation.
- *Learning Helm* by Matt Butcher, Matt Farina, and Josh Dolitsky — the canonical book by the original Helm authors.
- charts.bitnami.com/bitnami — the Bitnami chart catalog, the most-used public source of production-quality charts.
- artifacthub.io — Artifact Hub, the global catalog of public charts (and operators, plugins, and more).
- KubeCon Helm talks (kubernetes.io/community/) — every KubeCon has at least one good Helm talk; the recordings live on YouTube.
- github.com/helm/helm — the source code and release notes.
