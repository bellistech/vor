# Kubernetes — ELI5 (The Robot Dock Manager for Containers)

> Kubernetes is a robot foreman that runs a shipping dock. You hand it a paper that says "I want three of these boxes running, plugged into a phone, and if any box falls over, replace it." The foreman makes the world match the paper. Forever.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` and a working knowledge of "what a container is" help; if you don't have that yet, run `docker ps` and we'll fill in the gaps)

You do not need to have ever written a YAML file. You do not need to have ever managed a server. You do not need to know what a "deployment" is. You do not need to have used the cloud. By the end of this sheet you will know what Kubernetes is, you will be able to read a Pod manifest, you will be able to find why a Pod is on fire, and you will be able to hold your own in any room where people use the word "k8s" without flinching.

If you have not yet read `cs ramp-up linux-kernel-eli5`, you can still read this sheet. Where it matters, we'll quickly remind you what a process or a namespace or a cgroup is. But the kernel sheet is the parent of this one — Kubernetes is just the kernel's idea of "isolated processes" stretched across many computers, dressed up with paperwork.

If you have not used containers before, here is the one-paragraph version. A **container** is a way of packaging a program along with its files and libraries into one bundle, so that the bundle runs the same way on any computer. The bundle is called an **image**. When you start the bundle, you get a **container**. A container looks like its own tiny computer to the program inside, but really it is just a regular process on a regular computer, walled off by some Linux kernel features (namespaces and cgroups, see `cs ramp-up linux-kernel-eli5`). One container = one running thing. That's it.

Now, with containers, you can run one program neatly. With Kubernetes, you can run **thousands** of programs across **hundreds of computers**, and the system will keep them all alive, healthy, and findable, even when computers catch fire and packages get dropped. That's the whole game.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Kubernetes?

### Imagine you run a shipping dock

Picture a giant shipping dock. Like, a real one. Big concrete pad next to a harbor. Forklifts driving back and forth. Stacks of metal shipping containers everywhere. Trucks pulling in and out. People in orange vests with clipboards.

You are the boss of this dock. Your job is to make sure that at any given moment, all the right shipping containers are sitting in the right places, plugged into the right power outlets, with their doors pointing the right way, ready to be loaded onto the right trucks.

You don't drive the forklifts. You don't pick up the containers. You don't pour the concrete. You just decide what the dock should look like.

Now, a normal shipping dock has humans doing all of this. A foreman walks around shouting at people. The forklifts are driven by humans. The clipboards are filled in by humans. If a container falls off a truck, a human notices, calls a forklift driver, and the driver picks it up.

A **Kubernetes shipping dock** is different. There are no humans on the dock. The forklifts drive themselves. The clipboards fill themselves in. Even the foreman is a robot. There is a single piece of paper pinned to the wall that says, "I want **3 copies** of the **red container** sitting in the dock at all times, each one plugged into a power outlet on **port 80**, and if any of them fall over, please replace them within thirty seconds."

The robot foreman reads that paper, walks around the dock, and makes the world match the paper. Forever. If a forklift breaks down, the robot foreman finds another forklift. If a container catches fire, the robot foreman wheels in a new one. If you change the paper to say "I want 5 copies," the robot foreman wheels in two more containers. If you change the paper to say "I want 0 copies," the robot foreman dismantles them all.

You never tell the robot foreman *how* to do anything. You only tell it *what* you want. The robot foreman figures out the *how* for itself, and keeps figuring it out, all day, every day, forever.

**That's Kubernetes.**

### The two key words: declarative and reconcile

Two words come up over and over in Kubernetes. Let's nail them down right now.

**Declarative** means you write down what you want, not how to do it. The opposite of declarative is **imperative**. If you walked up to a chef and said, "Stir the pot for two minutes, add salt, then turn off the heat," that's imperative. You're giving step-by-step instructions. If you instead said, "I want soup, ready by 7pm, salty," that's declarative. You're describing the goal, and the chef figures out the steps.

Kubernetes is declarative. You write a manifest (a YAML file) that says, "I want three copies of this container running, each with 200 megabytes of memory, all reachable on port 80." You don't say *how* to start the containers, *which computer* to put them on, *how* to wire up the network. You just say what you want. Kubernetes figures out the rest.

**Reconcile** means "make the world match the goal, even if the world keeps drifting." Kubernetes is constantly comparing what's on the paper (the **desired state**) with what's actually happening on the dock (the **current state**). Whenever they don't match, Kubernetes does something to fix the gap. This is called the **reconcile loop**, and it never stops.

Imagine you have a thermostat in your house. You set it to 70°F. That is the desired state. The actual temperature is the current state. If it's 65°F, the thermostat turns on the heat. If it's 75°F, it turns on the AC. The thermostat never stops checking. It is a tiny reconcile loop. Kubernetes is a giant reconcile loop, but for containers.

In code, the reconcile loop looks like:

```
while true:
    desired = read_the_paper()
    current = look_at_the_dock()
    if desired != current:
        do_something_to_close_the_gap()
    sleep_a_tiny_bit()
```

That is the entire heart of Kubernetes. Everything else is bells and whistles. Read the paper. Look at the dock. Fix the gap. Sleep. Repeat. Forever.

### The control plane and the data plane

The robot foreman in our analogy is actually a small team of robots. Together they're called the **control plane**. The control plane lives in a small office on one side of the dock. It does not pick up containers. It does not drive forklifts. It just thinks, decides, and writes things down.

The forklifts are robots too. They live out on the dock, doing the actual heavy lifting. Together, the forklifts are called the **data plane**, or sometimes "the worker nodes." Every forklift can pick up containers, plug them in, and report back to the control plane. The forklifts themselves are computers — usually virtual machines, sometimes bare-metal servers.

The control plane never touches the dock. The data plane never makes decisions. They talk to each other constantly, but their jobs are completely separate.

```
                         CONTROL PLANE (the office)
              +---------------------------------------+
              |  api-server   etcd   scheduler        |
              |  controller-manager  cloud-controller |
              +---------------------------------------+
                              ^
                              | "what should I be doing?"
                              | "here's what I'm doing"
                              v
        +-------------+-------------+-------------+
        |  NODE 1     |  NODE 2     |  NODE 3     |    DATA PLANE
        |  (forklift) |  (forklift) |  (forklift) |    (the dock)
        |             |             |             |
        |  kubelet    |  kubelet    |  kubelet    |
        |  kube-proxy |  kube-proxy |  kube-proxy |
        |             |             |             |
        |  [Pod] [Pod]|  [Pod] [Pod]|  [Pod] [Pod]|
        +-------------+-------------+-------------+
```

The boxes inside the control plane (api-server, etcd, scheduler, controller-manager) are the robot foreman's organs. We'll meet each one in turn.

### The tiniest possible Kubernetes example

Here is the simplest paper you can pin to the wall:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
      - name: web
        image: nginx:1.27
        ports:
        - containerPort: 80
```

Twenty lines. That paper says: "I want **3 copies** of an **nginx container**, each labeled `app=hello`, each listening on **port 80**." If you save that as `hello.yaml` and run `kubectl apply -f hello.yaml`, Kubernetes will:

1. Pick three forklifts (nodes) with enough room.
2. Pull the `nginx:1.27` image onto each of them.
3. Start three containers, one on each chosen forklift.
4. Watch them. If any die, replace them. Forever.

That's it. That's the entire workflow. Write paper. Apply paper. Walk away. Kubernetes does the rest.

### Why so many words for "thing"?

You're going to encounter a lot of jargon. **Pod**, **container**, **Deployment**, **ReplicaSet**, **Service**, **Ingress**, **Node**. They all sound like roughly the same thing — a "thing on the dock." But they each mean something specific, and they each exist for a specific reason.

The trick to keeping them straight is to remember **what each one is wrapping**.

- A **Container** is the cargo inside the box.
- A **Pod** is the box itself, holding one or more containers, plus a shared phone line.
- A **ReplicaSet** is the rule "I want N identical boxes."
- A **Deployment** is the higher-level rule "I want N boxes, and when I update the cargo, swap them out gracefully."
- A **Service** is the phone book entry that says "to reach any of those boxes, call this number."
- A **Ingress** is the front door of the warehouse that routes outside callers to the right phone book entries.
- A **Node** is the forklift / computer that physically holds the boxes.

We'll meet each one in detail soon. For now, just remember the wrapping order: container is wrapped by pod is wrapped by replicaset is wrapped by deployment, and pods are reachable through services through ingresses, all running on nodes.

## The Core Objects

We're going to walk through every object you'll meet in your first month of Kubernetes. Each one gets a one-paragraph plain-English explanation, then a tiny YAML example, then a line about when to use it.

### Pod

The smallest schedulable thing on the dock. A **Pod** is one or more containers that share a network address and some storage. Think of it as a single shipping container, except sometimes it's a shipping container with a smaller piggyback container welded to its side. The big container is your app. The piggyback (called a **sidecar**) might be a logging agent, an Istio proxy, or a credentials helper.

Most Pods have exactly one container. Some have two or three. Almost never more than that. If you find yourself wanting a Pod with eight containers, that's a sign you should split them into separate Pods.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hello-pod
  labels:
    app: hello
spec:
  containers:
  - name: web
    image: nginx:1.27
    ports:
    - containerPort: 80
```

When to use one directly: almost never. You usually let a Deployment or StatefulSet make Pods for you. We'll see why in a minute.

The big mental shift: in Docker land, you think about containers. In Kubernetes land, you think about Pods. A Pod is the **schedulable unit**. The scheduler doesn't pick a node for a container; it picks a node for a Pod. Every container in a Pod always runs on the same node. They share an IP address. They can talk to each other on `localhost`. They can share files via shared volumes.

This is a really important detail and easy to forget. Two containers in the same Pod = same IP, can talk on localhost. Two containers in different Pods = different IPs, must use the network. Even if they're on the same node.

### Deployment

A **Deployment** is a wrapper that says, "I want N copies of this Pod template, and I want a graceful way to update them." Deployments are the workhorse of Kubernetes. Almost everything stateless you run is a Deployment.

When you change the Pod template inside a Deployment (like updating the image from `nginx:1.27` to `nginx:1.28`), the Deployment performs a **rolling update**: it spins up new Pods one at a time, waits for each to become ready, and tears down old Pods in lockstep. Zero downtime. Beautiful.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
      - name: web
        image: nginx:1.27
```

When to use it: any time you have a stateless web service, batch worker, or background daemon that doesn't need a stable hostname. Which is most things.

The two-line summary: Deployments make ReplicaSets, ReplicaSets make Pods. You write the Deployment. Kubernetes does the rest.

### ReplicaSet

A **ReplicaSet** is the internal object the Deployment uses to keep N copies running. You almost never create a ReplicaSet directly. You create a Deployment, and the Deployment creates ReplicaSets behind the scenes.

Why have ReplicaSets at all? Because rolling updates need two ReplicaSets at once: the old one (winding down) and the new one (winding up). The Deployment object orchestrates the dance between them.

```
Deployment "hello"
   |
   +-- ReplicaSet "hello-abc123" (image: nginx:1.27, replicas: 3)
   |       |
   |       +-- Pod "hello-abc123-x1y2z"
   |       +-- Pod "hello-abc123-a1b2c"
   |       +-- Pod "hello-abc123-d4e5f"
   |
   +-- ReplicaSet "hello-def456" (image: nginx:1.28, replicas: 0)  (paused, for rollback)
```

When to use one directly: never, basically. If you find yourself touching ReplicaSets, you're probably debugging.

### Service

A **Service** is a stable phone number for a set of Pods. Pods come and go — they get rescheduled, they crash, they get replaced. Their IP addresses change all the time. A Service gives you a single virtual IP and DNS name that always points to "whichever Pods are currently up and ready."

The Service uses **label selectors** to figure out which Pods belong. If your Pods have the label `app=hello`, your Service can target them with `selector: { app: hello }`.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hello
spec:
  selector:
    app: hello
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP
```

When to use it: any time you want one Pod to talk to another, or you want to expose a Pod to the cluster (or beyond). Almost every workload has a Service in front of it.

There are four flavors of Service:

- **ClusterIP** (default) — virtual IP only reachable inside the cluster. Use for service-to-service communication.
- **NodePort** — opens a port on every node (e.g., 30080), so traffic to `<any-node-IP>:30080` reaches your Pods.
- **LoadBalancer** — same as NodePort, but also asks the cloud provider for a real load balancer (an AWS ELB, a GCP Network LB, etc.).
- **ExternalName** — DNS alias to an external hostname. Doesn't actually route traffic; just creates a DNS CNAME.

The DNS name for a Service inside the cluster is `<service-name>.<namespace>.svc.cluster.local`, but you can usually just say `<service-name>` from another Pod in the same namespace.

### ConfigMap

A **ConfigMap** is a bag of key-value config strings. You can mount it as files inside a Pod, or inject the values as environment variables. ConfigMaps are how you keep your configuration **out of your image** and **in the cluster**.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hello-config
data:
  log_level: "debug"
  greeting: "Hello, world!"
  config.yaml: |
    server:
      port: 8080
      timeout: 30s
```

When to use it: anything that's not a secret. App config, feature flags, log levels, server addresses.

### Secret

A **Secret** is exactly like a ConfigMap, except its values are base64-encoded and (slightly) more access-controlled. They're meant for passwords, API keys, TLS certificates, and other things you don't want printed to logs.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
type: Opaque
data:
  username: ZGJ1c2Vy   # base64("dbuser")
  password: c2VjcmV0   # base64("secret")
```

**HUGE caveat:** by default, Secrets are **not encrypted at rest**. They're stored in etcd in the same way ConfigMaps are. Anyone with read access to etcd can read your secrets in plain text. To get real encryption at rest, you need to configure `EncryptionConfiguration` on the API server, or use an external secrets manager like HashiCorp Vault or AWS Secrets Manager.

When to use it: anything sensitive. Database passwords, API tokens, TLS keys.

### Namespace

A **Namespace** is a logical folder. It groups objects together so they don't collide. Two Deployments named `hello` can exist on the same cluster as long as they're in different namespaces.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: production
```

When to use it: separating teams, environments (dev/staging/prod), or applications.

Some special namespaces are pre-created:
- `default` — where stuff lands if you don't pick.
- `kube-system` — control plane components live here.
- `kube-public` — readable by everyone, used for cluster info.
- `kube-node-lease` — used internally for node heartbeats.

### Node

A **Node** is a machine — physical or virtual — that runs Pods. Every Node has a `kubelet` running on it, which is the tiny agent that talks to the control plane and runs Pods.

```
$ kubectl get nodes
NAME        STATUS   ROLES           AGE   VERSION
node-01     Ready    control-plane   45d   v1.31.2
node-02     Ready    <none>          45d   v1.31.2
node-03     Ready    <none>          45d   v1.31.2
```

You don't typically create Nodes through YAML. You add Nodes by spinning up a new machine, installing kubelet, and pointing it at the control plane. Or you let an autoscaler do it for you.

### StatefulSet

A **StatefulSet** is like a Deployment, but for stateful apps that need stable identity. Things like databases, message queues, or anything where each instance has its own role and its own data.

The differences:
- Pods get **stable, predictable hostnames**: `mydb-0`, `mydb-1`, `mydb-2` (instead of random hashes).
- Pods get **stable, persistent storage**: each Pod gets its own PersistentVolumeClaim that follows it across reschedules.
- Pods come up **in order**: `mydb-0` starts first, then `mydb-1`, then `mydb-2`. They tear down in reverse order.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mydb
spec:
  serviceName: mydb
  replicas: 3
  selector:
    matchLabels:
      app: mydb
  template:
    metadata:
      labels:
        app: mydb
    spec:
      containers:
      - name: postgres
        image: postgres:16
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 10Gi
```

When to use it: databases (Postgres, MySQL, MongoDB), distributed systems with consensus (Kafka, ZooKeeper, etcd), anything where Pod #0 is different from Pod #1.

### DaemonSet

A **DaemonSet** says "run exactly one Pod on every node, please." When you add a new node, the DaemonSet automatically schedules one of its Pods on it. When you remove a node, the Pod goes with it.

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: log-collector
spec:
  selector:
    matchLabels:
      app: log-collector
  template:
    metadata:
      labels:
        app: log-collector
    spec:
      containers:
      - name: collector
        image: fluentd:v1.16
        volumeMounts:
        - name: varlog
          mountPath: /var/log
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
```

When to use it: log shippers (Fluentd, Filebeat), monitoring agents (Prometheus node-exporter), CNI plugins, network policy enforcers, anything that needs to run on every node.

### Job and CronJob

A **Job** runs one or more Pods to completion. It's for batch tasks — "compute this report, then exit." Once the Pod finishes successfully, the Job is done.

A **CronJob** is a Job on a schedule. "Every night at 2 AM, run this batch task."

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly-cleanup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: my-cleanup-tool:1.0
          restartPolicy: OnFailure
```

When to use one: data exports, backup runs, scheduled cleanups, training pipelines, anything that's not a long-running service.

### Ingress

An **Ingress** routes external HTTP/HTTPS traffic to Services inside the cluster. It's the front door of your warehouse. Without an Ingress, the only way external clients reach your Pods is through `NodePort` or `LoadBalancer` Services, which are clumsy.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hello-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: hello.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hello
            port:
              number: 80
```

When to use it: any time you want external HTTP traffic routed to internal Services, especially with TLS termination, host-based routing, or path-based routing.

You need an **Ingress Controller** running in the cluster for Ingress objects to do anything. Common choices: nginx-ingress, Traefik, HAProxy, AWS ALB Ingress Controller, GKE Ingress.

The newer cousin of Ingress is the **Gateway API**, which is more expressive and is gradually replacing Ingress for sophisticated routing. See `cs orchestration gateway-api`.

### PersistentVolume / PersistentVolumeClaim

Storage in Kubernetes has two pieces:

- A **PersistentVolume (PV)** is a chunk of storage somewhere — an EBS volume, a GCE PD, a Ceph RBD, an NFS share. Cluster admins manage PVs. You usually don't create them by hand.
- A **PersistentVolumeClaim (PVC)** is a Pod's *request* for storage. "I want 10 gigs of fast storage." Kubernetes binds the PVC to a matching PV.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-claim
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd
```

You then mount the PVC into your Pod:

```yaml
spec:
  containers:
  - name: app
    volumeMounts:
    - name: data
      mountPath: /data
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: data-claim
```

When to use one: any time you need data to survive Pod reschedules — databases, file uploads, caches that should warm up between restarts.

A **StorageClass** is the recipe for dynamically provisioning new PVs. When a PVC asks for storage from `storageClassName: fast-ssd`, the cluster's CSI driver automatically creates a new EBS volume (or whatever) and binds it.

### Custom Resource Definition (CRD)

A **CRD** is how you teach Kubernetes about a brand-new kind of object. "I want to be able to write `kind: PostgresCluster` in my YAML and have Kubernetes understand it."

You write a CRD that defines the shape of the new object, then deploy a controller (called an **Operator**) that watches for instances of that object and does the right thing.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: postgresclusters.acid.zalan.do
spec:
  group: acid.zalan.do
  scope: Namespaced
  names:
    kind: PostgresCluster
    plural: postgresclusters
    singular: postgrescluster
  versions:
  - name: v1
    served: true
    storage: true
    schema: { ... }
```

When to use it: when you're building an Operator, or when you've installed an Operator that comes with its own CRDs (most production Kubernetes installs have hundreds).

The CRD is the schema. The instances of it (`kind: PostgresCluster`) are **Custom Resources (CRs)**. The pattern of "CRD + controller that watches CRs" is called the **Operator pattern**, and it's how Kubernetes gets extended without modifying the core. See `cs orchestration operator`.

## How a Pod Gets Scheduled

This is one of the most-asked questions, so let's walk it slowly. From the moment you say "I want a Pod" to the moment a Pod is actually running, here's what happens:

### Step-by-step journey

1. **You apply a Deployment manifest.** You run `kubectl apply -f hello.yaml`. The kubectl tool sends the YAML to the **API server**. The API server validates it (is the YAML well-formed? do you have permission?) and stores it in **etcd**, the cluster's database.

2. **The Deployment controller notices.** Inside the control plane, the **Deployment controller** is running its own little reconcile loop. It sees the new Deployment object in etcd and says, "Oh, a new Deployment that wants 3 replicas. Better make a ReplicaSet."

3. **A ReplicaSet is created.** The Deployment controller writes a new ReplicaSet object to etcd.

4. **The ReplicaSet controller notices.** Another controller, the **ReplicaSet controller**, sees the new ReplicaSet and says, "Oh, this ReplicaSet wants 3 Pods. Better create them." It writes 3 Pod objects to etcd, each with `nodeName` *unset*. The Pods are in **Pending** state.

5. **The scheduler notices.** The **scheduler** sees the 3 Pending Pods (Pods without an assigned node) and starts looking for a home for each one. For each Pod, it considers every node in the cluster, scores them based on how well they fit, and picks the best one. Then it writes the chosen node name into the Pod's spec.

6. **The kubelet on the chosen node notices.** The kubelet on the assigned node sees a Pod with its name on it and says, "Oh, that's my job." It pulls the container image (calling `containerd` or `cri-o`), creates the network namespace, mounts any volumes, and starts the containers. While this is happening, the Pod is in **ContainerCreating** state.

7. **The container starts.** Once the kubelet has the container running, the Pod transitions to **Running**. If readiness probes are configured, the Pod won't be marked **Ready** until the probes pass.

8. **The Service controller notices.** If the Pod has labels matching a Service's selector, and the Pod becomes Ready, an **Endpoints** entry is created for it. Now the Service load-balances to it.

That's the whole journey. Eight steps, but the cool thing is **none of these steps know about each other directly**. They each just watch etcd. The Deployment controller doesn't tell the ReplicaSet controller "hey, do this" — it just writes a new object to etcd, and the ReplicaSet controller happens to be watching that part of etcd. This pattern is called **the controller pattern** or **the watch-loop pattern**, and it's everywhere in Kubernetes.

### ASCII diagram of the journey

```
        you                  api-server          etcd            controllers           scheduler         kubelet (on node-2)
         |                       |                  |                  |                    |                   |
         |  apply Deployment ->  |                  |                  |                    |                   |
         |                       |  store -------->|                  |                    |                   |
         |                       |                  |                  |                    |                   |
         |                       |                  |  watch  -------->| Deployment ctrl    |                   |
         |                       |                  |                  | sees Deployment    |                   |
         |                       |  create RS  <--- |  <-------------- |                    |                   |
         |                       |  store -------->|                  |                    |                   |
         |                       |                  |  watch  -------->| ReplicaSet ctrl    |                   |
         |                       |  create 3 Pods <-|  <-------------- |                    |                   |
         |                       |  store -------->|                  |                    |                   |
         |                       |                  |  watch  --------------------------- > | sees Pending Pods |
         |                       |  patch Pod with  |  <----------------------------------- | picks node-2      |
         |                       |  nodeName=node-2 |                                       |                   |
         |                       |  store -------->|                                       |                   |
         |                       |                  |  watch  -------------------------------------------------> | sees Pod assigned to me
         |                       |                  |                                                            | pulls image
         |                       |                  |                                                            | creates netns
         |                       |                  |                                                            | starts containers
         |                       |                  |  <- update Pod status: Running -------------------------- |
```

This is what's happening in slow motion every time you apply a Deployment. In practice, the whole journey takes a few hundred milliseconds plus image pull time. Image pull dominates: a 500MB image on a slow network might take 30 seconds. A 50MB image on a warm cache takes 200 milliseconds.

### What the scheduler actually does

The scheduler is one of the most interesting components. It looks at every node and asks two questions for each Pod:

1. **Filter:** Can this Pod even fit on this node? (Does it have enough CPU? Memory? GPUs? Right architecture? No conflicting taints? NodeSelector matches? Topology constraints satisfied?)

2. **Score:** Among the nodes that *can* fit it, which is best? (Spread across zones? Match anti-affinity? Lowest already-allocated CPU? Image already cached?)

The scheduler picks the highest-scored node and binds the Pod to it. The whole thing usually takes single-digit milliseconds per Pod.

The scoring algorithm is pluggable. Some clusters use the default scheduler. Some use **Karpenter** (an open-source scheduler that's more aggressive about bin-packing). Some have multiple schedulers, and Pods can pick which one they want via `spec.schedulerName`.

### NodeSelector, affinity, taints, tolerations

You can influence where Pods land:

- **NodeSelector** — "only schedule on nodes with these labels." Cheapest and simplest.
  ```yaml
  spec:
    nodeSelector:
      disktype: ssd
  ```

- **Affinity / anti-affinity** — more flexible, can express "prefer" vs. "require", and can match Pod labels (not just node labels). Lets you say "schedule near these other Pods" or "spread away from those Pods."

- **Taints and tolerations** — taints repel Pods from nodes. Tolerations let specific Pods opt in. "This node is for GPU jobs only — taint it `gpu=true:NoSchedule`. GPU Pods have a toleration for that taint, so only they can land there."

- **TopologySpreadConstraints** — spread Pods across zones, regions, or arbitrary topology. "Don't put all 6 replicas in the same availability zone."

## Labels and Selectors

This is the matchmaking system that holds the whole cluster together. Every object has **labels** (key-value tags). **Selectors** match labels. Almost every relationship in Kubernetes is built on labels and selectors.

### Why labels and selectors?

The naive way to wire up a cluster would be to give every Pod a unique name and have other things refer to those names. But Pod names change all the time — they get appended with hashes, they get recreated, they're random. If you tried to wire things up by name, you'd be in pain.

Instead, every Pod gets a sticky note (or several sticky notes) on the side. Say, `app=hello`, `tier=frontend`, `version=v2`. Now anything that wants to talk to "the hello frontend" doesn't ask for a specific Pod by name. It says, "give me Pods where `app=hello` and `tier=frontend`." If the Pods change, the labels stay the same, and everything keeps working.

### Where labels show up

- **A Service finds its Pods** by label selector.
- **A Deployment manages its Pods** by label selector.
- **A ReplicaSet manages its Pods** by label selector.
- **A NetworkPolicy chooses which Pods it applies to** by label selector.
- **A PodDisruptionBudget protects Pods** by label selector.
- **An HPA scales Pods** by label selector.
- **NodeSelector picks nodes** by label selector.
- **Pod affinity colocates Pods** by label selector.

You see the pattern. Labels are the universal glue.

### Setting labels

In a manifest:

```yaml
metadata:
  labels:
    app: hello
    tier: frontend
    version: v2
    managed-by: argocd
```

From the command line:

```
$ kubectl label pod hello-abc123 owner=alice
$ kubectl label pod hello-abc123 owner-                  # remove
$ kubectl label nodes node-01 disktype=ssd
```

### Selecting on labels

```
$ kubectl get pods -l app=hello                          # exact match
$ kubectl get pods -l 'app in (hello, world)'             # set match
$ kubectl get pods -l 'tier!=backend'                     # negation
$ kubectl get pods -l 'app=hello,tier=frontend'           # AND
$ kubectl get pods -l 'env'                               # has the label
$ kubectl get pods -l '!env'                              # missing the label
```

### Annotations vs. labels

A close cousin of the label is the **annotation**. Both are key-value tags, but:

- **Labels** are for selection. They're queryable. They're indexed in etcd.
- **Annotations** are for unstructured metadata. They're not queryable. They're for tools to scribble things on objects.

```yaml
metadata:
  annotations:
    deployment.kubernetes.io/revision: "5"
    kubectl.kubernetes.io/last-applied-configuration: |
      { ... }
    nginx.ingress.kubernetes.io/rewrite-target: /
```

Annotations often contain machine-readable things like timestamps, JSON config, or structured directives that Kubernetes itself or operators read.

## Why Pods Die (And What Happens)

This is the section every Kubernetes user comes back to. Pods die for many reasons, and learning to read why a Pod died is half the job.

### The Pod lifecycle

A Pod is in one of these phases:

- **Pending** — accepted by the cluster, but at least one container hasn't started yet. Possibly waiting for the scheduler, possibly waiting for image pull, possibly waiting for a volume.
- **Running** — bound to a node, all containers created, at least one container is running or starting.
- **Succeeded** — all containers exited successfully and won't be restarted.
- **Failed** — all containers exited, at least one with a non-zero status.
- **Unknown** — the kubelet on the assigned node stopped reporting. The control plane has lost contact.

`kubectl get pods` shows the phase plus a more useful "status" column that combines the phase with what's happening: `ContainerCreating`, `CrashLoopBackOff`, `ImagePullBackOff`, `Terminating`, etc.

### The big death reasons

#### ImagePullBackOff (or ErrImagePull)

The image couldn't be pulled. Reasons:

- The image doesn't exist (`nginx:1.27` is fine, but `nginx:1.99999` doesn't exist yet).
- Image typo (`ngnix:1.27` instead of `nginx:1.27`).
- Private registry, no auth (forgot to add an `imagePullSecret`).
- Network blocked (the node can't reach the registry).
- Rate-limited by Docker Hub (yes, this is real, and yes, it'll bite you).

To debug:

```
$ kubectl describe pod <name>
```

Look at the Events section at the bottom. It'll say something like `Failed to pull image "ngnix:1.27": rpc error: code = NotFound`.

#### CrashLoopBackOff

The container started, but immediately exited. Kubernetes restarted it. It exited again. Kubernetes restarted it. After a few restarts, Kubernetes starts backing off — waiting longer and longer between restart attempts. That state is `CrashLoopBackOff`.

Reasons:

- The app is misconfigured (missing env var, wrong DB password, can't reach a service).
- The app crashed on startup (bug, unhandled exception, panic).
- The container's command is wrong (typo in the entrypoint, missing binary).
- The image doesn't have what you think it does.

To debug:

```
$ kubectl logs <pod>                    # current container logs
$ kubectl logs <pod> --previous         # logs from the last (crashed) container
$ kubectl describe pod <name>           # events + container statuses
```

The logs from the crashed container are gold. Almost always they say exactly why it died. `panic: cannot connect to db at postgres:5432: connection refused`.

#### OOMKilled

The container exceeded its memory limit and was killed by the kernel's OOM killer. The container's exit code will be 137 (which is 128 + signal 9, SIGKILL).

To check:

```
$ kubectl describe pod <name> | grep -i -A2 "Last State"
    Last State:     Terminated
      Reason:       OOMKilled
      Exit Code:    137
```

The fix is one of:

1. Raise the memory limit. Maybe the app legitimately needs more.
2. Find the leak. Maybe the app is bloated or has a memory leak.
3. Lower the workload. Maybe each Pod is doing too much.

#### Evicted

The node ran out of resources (memory, disk, inodes) and had to kill some Pods to recover. Pods get evicted in priority order: BestEffort first, then Burstable, then Guaranteed.

```
$ kubectl describe pod <name>
Status:        Failed
Reason:        Evicted
Message:       The node was low on resource: ephemeral-storage. ...
```

Fix: free up resources on the node, give the Pod higher priority, set proper resource requests.

#### Preempted

A higher-priority Pod kicked your Pod off the node to make room.

#### Liveness probe failure

You configured a liveness probe (a periodic check) and it's failing. Kubernetes assumes the container is broken and restarts it.

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  failureThreshold: 3
```

The container above will be restarted if `/healthz` returns non-200 three times in a row.

If you see liveness probe failures, either the app is genuinely broken or the probe is too aggressive. A common mistake: setting `initialDelaySeconds` too low for a slow-starting app. The app is still warming up, the liveness probe fails, the container gets killed, repeat.

The fix: increase `initialDelaySeconds`, or use a **startup probe** (which is a separate probe just for slow starts).

#### Readiness probe failure (not a death, but related)

A failing readiness probe doesn't kill the Pod, but it removes the Pod from Service Endpoints. Traffic stops flowing to it. Usually you want this — "Pod isn't ready, don't send it traffic." But sometimes a misconfigured readiness probe makes a healthy Pod look unhealthy and traffic mysteriously stops flowing.

```
$ kubectl get endpoints svc/myservice
NAME           ENDPOINTS                                              AGE
myservice      <none>                                                 5m
```

`<none>` for endpoints = either no Pods match the selector, or none of the matching Pods are Ready. Usually the latter when you've deployed correctly but probes are failing.

#### Node went down

The node hosting your Pod died (kernel panic, network partition, hardware failure, somebody pulled the plug). The control plane notices after about 40 seconds (configurable via `--node-monitor-grace-period`). Pods are marked for eviction. After a tolerated period (default 5 minutes), Pods on a NotReady node are deleted and rescheduled.

This is one of the rare cases where Pod loss can take minutes to recover from. Most other cases are seconds.

## Networking — The Hardest Part

Every Kubernetes user agrees: networking is the hardest part. Here is the simple story, then the complicated story.

### The simple story

Every Pod gets its own IP address. From inside the cluster, any Pod can reach any other Pod by IP, with no NAT. That's it. That's the whole network model from the Pod's point of view.

```
Pod A (10.244.1.5) ─────► Pod B (10.244.2.7)
              just talk to 10.244.2.7. that's it.
```

### Wait, how?

This is the part that's complicated. Pods are processes on nodes. Nodes have IPs (say, 192.168.1.10, 192.168.1.11). Pod IPs are different from node IPs. When Pod A on node-1 sends a packet to Pod B on node-2, somebody has to:

1. Get the packet out of Pod A's network namespace and onto node-1's network stack.
2. Get the packet from node-1 to node-2 (across the underlying network).
3. Get the packet onto node-2's network stack.
4. Get the packet into Pod B's network namespace.

The thing that makes all this happen is the **CNI plugin** (Container Network Interface). Common CNIs:

- **Flannel** — simple overlay network, easy to set up.
- **Calico** — BGP-based, no overlay, fast, plus NetworkPolicy enforcement.
- **Cilium** — eBPF-based, fast, fancy NetworkPolicy with L7 awareness.
- **Weave Net** — encrypted overlay.
- **AWS VPC CNI / GKE CNI / Azure CNI** — cloud-native, use cloud routing.

The CNI runs as a DaemonSet (one Pod per node), and it sets up `iptables` rules, `ipvs` rules, BPF programs, or VXLAN tunnels — whatever it takes to make Pod-to-Pod packets flow.

### Services and how they actually work

A Service has a virtual IP (the **ClusterIP**). When a Pod sends a packet to the ClusterIP, it doesn't actually go anywhere — there's no Pod with that IP. Instead, on every node, **kube-proxy** (or a CNI like Cilium) intercepts the packet, picks a real Pod from the Service's Endpoints list, and rewrites the destination IP to that Pod's real IP.

Three modes:

- **iptables mode** (default for kube-proxy) — installs a pile of `iptables` rules. Simple, robust, but slow for large clusters.
- **ipvs mode** — uses the kernel's IPVS load balancer. Faster for many Services.
- **eBPF mode** (Cilium) — uses BPF programs for the same effect. Fastest, most flexible.

```
       (inside Pod A)
       │ packet to ClusterIP 10.96.5.5:80
       ▼
       (kube-proxy/eBPF on node-1 intercepts)
       │ rewrites to one of:
       │   - Pod B (10.244.2.7:80)
       │   - Pod C (10.244.3.9:80)
       │   - Pod D (10.244.1.12:80)
       │ picks one (round-robin or hash)
       ▼
       packet leaves node-1 toward chosen Pod
```

This is **client-side load balancing**. There's no central load balancer. Every node load-balances on its own.

### NodePort

A **NodePort Service** is the same as a ClusterIP Service, but with a port also opened on every node. So you can reach the Service from outside the cluster by hitting `<any-node-ip>:30080` or whatever port was assigned.

```
external ─► node-1:30080 ─► (kube-proxy) ─► one of the Pods (anywhere in cluster)
```

NodePorts are blunt. They use a high port number (30000-32767 by default), they don't do TLS, and you have to know which node IPs to hit. Mostly you graduate to LoadBalancer or Ingress.

### LoadBalancer

A **LoadBalancer Service** is a NodePort that also asks the cloud provider for a real load balancer. AWS gives you an ELB, GCP gives you a Network LB, Azure gives you an Azure LB. Traffic flows through the cloud LB, into a NodePort on the cluster, into a Pod.

```
external ─► AWS ELB ─► node-X:30080 ─► (kube-proxy) ─► Pod
```

Each LoadBalancer Service costs money (it provisions a real cloud resource). Production clusters often consolidate by putting an Ingress controller behind a single LoadBalancer.

### Ingress

An **Ingress Controller** is a Pod (or DaemonSet) running an HTTP reverse proxy (nginx, Traefik, HAProxy, Envoy). External traffic hits the Ingress Controller, which then looks at the host header and path, picks the right Service, and forwards.

```
external ─► AWS ELB ─► nginx-ingress-controller Pod ─► Service ─► Pod
            (one LB)              (looks at /api/* routing)
```

You can have one LoadBalancer in front of an Ingress Controller, and that Ingress Controller can route hundreds of hostnames and paths to different Services. Way cheaper than one LB per Service.

### NetworkPolicy

Out of the box, all Pods can talk to all Pods. That's terrifying for production. **NetworkPolicy** lets you restrict it.

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-only-from-frontend
  namespace: prod
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: frontend
    ports:
    - port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: db
    ports:
    - port: 5432
```

This says: "Pods labeled `app=api` only accept traffic from Pods labeled `app=frontend` on port 8080, and only initiate traffic to Pods labeled `app=db` on port 5432."

NetworkPolicy enforcement requires a CNI that supports it. Calico and Cilium do. Some cloud CNIs do too. Flannel does not (you have to layer Calico on top).

### DNS in the cluster

Every cluster has a built-in DNS service (usually CoreDNS). Pods can look up Services by name:

```
$ kubectl exec -it some-pod -- nslookup hello.default.svc.cluster.local
Server:         10.96.0.10
Address:        10.96.0.10#53

Name:   hello.default.svc.cluster.local
Address: 10.96.5.5
```

From inside a Pod, `hello` (in the same namespace) or `hello.default` works without the long suffix.

## Storage

### emptyDir

Scratch space that lives as long as the Pod lives. Dies when the Pod dies. Stored on the node's local disk (or memory, if you set `medium: Memory`).

```yaml
volumes:
- name: scratch
  emptyDir: {}
```

When to use: temporary files, caches, work-in-progress data that doesn't need to survive.

### hostPath

Mounts a directory from the node into the Pod. Tight coupling — Pod is tied to a specific node's filesystem. Generally avoid.

```yaml
volumes:
- name: docker-socket
  hostPath:
    path: /var/run/docker.sock
    type: Socket
```

When to use: rarely. Mostly for system-level DaemonSets that need to read node files (log collectors reading `/var/log`, monitoring agents reading `/proc`).

### PVC + StorageClass

The grown-up way to do persistent storage. The Pod asks for a PVC, the PVC binds to a PV, and the PV is backed by some real storage (cloud disk, NFS, Ceph, etc.).

```yaml
volumes:
- name: data
  persistentVolumeClaim:
    claimName: my-data
```

If you want **dynamic provisioning** (no admin involvement when a new PVC asks for storage), set up a StorageClass:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
allowVolumeExpansion: true
```

Now any PVC that says `storageClassName: fast-ssd` triggers automatic creation of an EBS gp3 volume of the requested size.

### ConfigMap and Secret as files

Both can be mounted as files inside the Pod:

```yaml
volumes:
- name: config
  configMap:
    name: hello-config
```

```yaml
volumeMounts:
- name: config
  mountPath: /etc/hello
```

Each key in the ConfigMap becomes a file under `/etc/hello/`. Same idea for Secrets.

### Projected volumes

Combine multiple sources into one mount:

```yaml
volumes:
- name: combined
  projected:
    sources:
    - configMap:
        name: hello-config
    - secret:
        name: hello-secret
    - serviceAccountToken:
        audience: vault
        expirationSeconds: 3600
        path: vault-token
```

Useful for sidecars (like Vault Agent or Istio) that need a mix of config, secrets, and short-lived tokens.

## RBAC (Role-Based Access Control)

This is how Kubernetes decides who can do what. The model has four key objects:

### ServiceAccount

A **ServiceAccount** is an identity for Pods. Every Pod runs as a ServiceAccount. By default, every namespace has a `default` ServiceAccount, and Pods use that unless told otherwise.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hello-sa
  namespace: default
```

Pods reference it:

```yaml
spec:
  serviceAccountName: hello-sa
  containers:
  - name: app
    image: my-app:1.0
```

A token for the ServiceAccount is automatically mounted at `/var/run/secrets/kubernetes.io/serviceaccount/token`. The Pod can use it to call the Kubernetes API.

### Role and ClusterRole

A **Role** says what a ServiceAccount (or user) can do, scoped to one namespace. A **ClusterRole** is the same thing but cluster-wide.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: default
  name: pod-reader
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
```

This says "the holder of this role can list, get, and watch Pods in `default`."

### RoleBinding and ClusterRoleBinding

Roles by themselves don't do anything. You bind them to a subject (a ServiceAccount, user, or group):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: hello-can-read-pods
  namespace: default
subjects:
- kind: ServiceAccount
  name: hello-sa
  namespace: default
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

Now `hello-sa` can list, get, and watch Pods in the `default` namespace.

### ASCII diagram of RBAC

```
   ┌─────────────────┐         ┌──────────────────┐         ┌────────────────┐
   │  ServiceAccount │         │   RoleBinding    │         │      Role      │
   │   hello-sa      │ ──bind─►│  hello-can-read  │ ──refs─►│   pod-reader   │
   │  (in default)   │         │   (in default)   │         │   (in default) │
   └─────────────────┘         └──────────────────┘         └────────────────┘
                                                                     │
                                                                     │ rules
                                                                     ▼
                                                        ┌────────────────────────┐
                                                        │ apiGroups:  [""]       │
                                                        │ resources:  [pods]     │
                                                        │ verbs:      [get,list] │
                                                        └────────────────────────┘
```

You bind a Subject to a Role using a Binding. The Role contains the rules (the verbs and resources). The Subject inherits those rules.

### The default ServiceAccount and why you should care

Every Pod that doesn't specify `serviceAccountName` runs as the namespace's `default` ServiceAccount. By default, that account has minimal permissions — it can read its own namespace, but not much else.

If you grant permissions to `default`, every Pod in that namespace silently gets them. This is a common security mistake. Instead, create dedicated ServiceAccounts per workload and bind only the minimum needed.

## Helm and Kustomize

These are the two big tools for managing your YAML at scale. Once you have more than five manifests, you'll want one of these.

### Helm

**Helm** is a templating engine plus a release manager for Kubernetes. You write **charts** (templated YAML) and install them with `helm install`. Helm tracks the install as a "release" so you can upgrade, rollback, or delete it as a unit.

A chart is a directory:

```
mychart/
├── Chart.yaml          # metadata: name, version
├── values.yaml         # default values
├── templates/
│   ├── deployment.yaml # template using Go template syntax
│   ├── service.yaml
│   └── _helpers.tpl    # shared template snippets
└── charts/             # chart dependencies
```

A template:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}-web
spec:
  replicas: {{ .Values.replicas }}
  template:
    spec:
      containers:
      - name: web
        image: "{{ .Values.image.repo }}:{{ .Values.image.tag }}"
```

Install:

```
$ helm install hello ./mychart
$ helm install hello ./mychart --set replicas=5
$ helm install hello ./mychart -f my-values.yaml
$ helm upgrade hello ./mychart --set image.tag=v2
$ helm rollback hello 1
$ helm uninstall hello
```

Helm has a public chart hub at `artifacthub.io` with thousands of charts (Postgres, Redis, Prometheus, Grafana, you name it).

When to use it: anything that's "an app." Application packages, third-party services, anything that's installed-as-a-unit and upgraded-as-a-unit.

### Kustomize

**Kustomize** is the opposite philosophy: no templates, just overlay-merge.

You start with a **base** (plain YAML) and apply **overlays** that patch the base for different environments.

```
my-app/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── replicas-patch.yaml
    └── prod/
        ├── kustomization.yaml
        └── replicas-patch.yaml
```

`base/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- deployment.yaml
- service.yaml
```

`overlays/prod/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../base
patches:
- path: replicas-patch.yaml
images:
- name: my-app
  newTag: v1.2.3
```

Apply:

```
$ kubectl apply -k overlays/prod
```

Kustomize is built into kubectl. No external tool needed.

When to use it: when you want plain YAML with environment overlays, no template engine in the way.

### Helm vs. Kustomize — which?

Religious war. Some teams use one, some use the other, some use both (Helm chart in base, Kustomize overlays for environment-specific patches).

Helm has more ecosystem (artifacthub charts). Kustomize is simpler and more transparent. You don't have to pick now. You'll learn both.

## Common Errors and Their Causes

### "ImagePullBackOff" / "ErrImagePull"

The kubelet can't pull the image.

```
$ kubectl describe pod my-pod
...
Events:
  Type     Reason          Age   From               Message
  ----     ------          ----  ----               -------
  Warning  Failed          10s   kubelet            Failed to pull image "ngnx:1.27": Error response from daemon: pull access denied for ngnx
  Warning  Failed          10s   kubelet            Error: ErrImagePull
  Normal   BackOff         9s    kubelet            Back-off pulling image "ngnx:1.27"
```

Causes and fixes:

1. **Typo in image name.** `ngnx` instead of `nginx`. Fix the typo.
2. **Image doesn't exist.** Tag missing. Verify with `docker pull <image>` from your laptop.
3. **Private registry, no auth.** Add an `imagePullSecret`:
   ```
   $ kubectl create secret docker-registry regcred \
       --docker-server=registry.example.com \
       --docker-username=alice \
       --docker-password=hunter2
   ```
   Reference it in the Pod:
   ```yaml
   spec:
     imagePullSecrets:
     - name: regcred
   ```
4. **Network blocked.** The node can't reach the registry. Check egress firewalls.
5. **Rate-limited.** Docker Hub rate-limits anonymous pulls. Use a paid account or a registry mirror.

### "CrashLoopBackOff"

The container started but exited. Kubernetes is restarting it with exponential backoff.

```
$ kubectl get pod my-pod
NAME      READY   STATUS             RESTARTS   AGE
my-pod    0/1     CrashLoopBackOff   5          3m
```

Always: `kubectl logs <pod>`. Then `kubectl logs <pod> --previous` for the last incarnation. The logs almost always say why.

Common causes:

- App misconfigured (missing env var).
- Bad command (typo in the entrypoint).
- App can't reach a dependency (DB unreachable on startup).
- Crash bug in the app.
- Container's filesystem is read-only and app needs to write somewhere.

### "OOMKilled"

The container exceeded its memory limit. The kernel OOM killer terminated it. Exit code 137.

```
$ kubectl describe pod my-pod | grep -A2 "Last State"
    Last State:     Terminated
      Reason:       OOMKilled
      Exit Code:    137
```

Fixes:

1. Raise the limit: `resources.limits.memory: 512Mi` → `1Gi`.
2. Find the leak. `kubectl top pod` can show steady-state usage.
3. Tune the app (JVM `-Xmx`, Node.js `--max-old-space-size`, Python GC, etc.).

### "FailedScheduling: 0/N nodes are available"

The scheduler couldn't find a home.

```
$ kubectl describe pod my-pod
...
Events:
  Type     Reason            Age   From               Message
  ----     ------            ----  ----               -------
  Warning  FailedScheduling  30s   default-scheduler  0/3 nodes are available: 3 Insufficient cpu.
```

The message tells you why. Common reasons:

- **Insufficient cpu / memory** — your `resources.requests` is too high to fit anywhere.
- **Untolerated taints** — your Pod doesn't tolerate the node's taint (often `node.kubernetes.io/not-ready`).
- **Anti-affinity blocked** — your anti-affinity rule excluded all nodes.
- **NodeSelector matched nothing** — your `nodeSelector` doesn't match any node's labels.
- **No PVC available** — the Pod's PVC isn't bound, and nothing can bind it.

### "Init:0/1" stuck

The Pod has an init container that's not finishing.

```
$ kubectl get pod my-pod
NAME      READY   STATUS     RESTARTS   AGE
my-pod    0/1     Init:0/1   3          2m
```

`Init:0/1` means "0 of 1 init containers are done." Look at the init container logs:

```
$ kubectl logs my-pod -c init-db
```

### "ContainerCreating" stuck

The kubelet picked up the Pod but is having trouble starting the container.

```
$ kubectl get pod my-pod
NAME      READY   STATUS              RESTARTS   AGE
my-pod    0/1     ContainerCreating   0          5m
```

Common reasons:

- Image pull is slow (large image, slow network).
- Volume mount is failing (PVC not bound, NFS server unreachable).
- ConfigMap or Secret referenced but doesn't exist.
- CNI plugin can't allocate a Pod IP (cluster out of IPs, CNI broken).

`kubectl describe pod` will usually have an event line.

### "Terminating" stuck

A Pod is being deleted but won't go away.

```
$ kubectl get pod my-pod
NAME      READY   STATUS        RESTARTS   AGE
my-pod    1/1     Terminating   0          1h
```

Causes:

- The container is ignoring SIGTERM. After `terminationGracePeriodSeconds` (default 30), it'll get SIGKILL.
- A finalizer is preventing deletion. Check `kubectl get pod my-pod -o yaml | grep finalizers`.
- The kubelet on the node is unreachable (node down).

To force delete:

```
$ kubectl delete pod my-pod --grace-period=0 --force
```

(Avoid this on stateful workloads — it can leave dangling state.)

### "Service has no endpoints"

The Service exists, but no Pods match its selector or no matching Pods are Ready.

```
$ kubectl get endpoints svc/my-service
NAME          ENDPOINTS                                              AGE
my-service    <none>                                                 5m
```

Check:

1. `kubectl get pods -l <selector>` — do any Pods match?
2. `kubectl get pods -l <selector> -o wide` — are they Ready?
3. If they're not Ready, `kubectl describe pod` and look at probes.

### "Forbidden" / "User cannot list X"

You hit an RBAC wall.

```
$ kubectl get pods
Error from server (Forbidden): pods is forbidden: User "alice" cannot list resource "pods" in API group "" in the namespace "default"
```

You need a Role and RoleBinding granting `list` on `pods`.

### "no matches for kind" / "no matches for apiVersion"

You're applying a resource that the cluster doesn't know about.

```
$ kubectl apply -f my-prometheus.yaml
error: unable to recognize "my-prometheus.yaml": no matches for kind "Prometheus" in version "monitoring.coreos.com/v1"
```

The CRD isn't installed. Install the operator first.

## Hands-On

≥ 30 paste-and-runnable kubectl commands. Each one shows expected output.

### Cluster info

```
$ kubectl version
Client Version: v1.31.2
Kustomize Version: v5.4.2
Server Version: v1.31.2

$ kubectl cluster-info
Kubernetes control plane is running at https://10.0.0.10:6443
CoreDNS is running at https://10.0.0.10:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

$ kubectl get nodes
NAME        STATUS   ROLES           AGE   VERSION
node-01     Ready    control-plane   45d   v1.31.2
node-02     Ready    <none>          45d   v1.31.2
node-03     Ready    <none>          45d   v1.31.2

$ kubectl get nodes -o wide
NAME      STATUS   ROLES           AGE   VERSION   INTERNAL-IP    EXTERNAL-IP   OS-IMAGE
node-01   Ready    control-plane   45d   v1.31.2   10.0.0.10      <none>        Ubuntu 22.04
node-02   Ready    <none>          45d   v1.31.2   10.0.0.11      <none>        Ubuntu 22.04
node-03   Ready    <none>          45d   v1.31.2   10.0.0.12      <none>        Ubuntu 22.04

$ kubectl api-resources | head -10
NAME                              SHORTNAMES   APIVERSION                  NAMESPACED   KIND
bindings                                       v1                          true         Binding
componentstatuses                 cs           v1                          false        ComponentStatus
configmaps                        cm           v1                          true         ConfigMap
endpoints                         ep           v1                          true         Endpoints
events                            ev           v1                          true         Event
limitranges                       limits       v1                          true         LimitRange
namespaces                        ns           v1                          false        Namespace
nodes                             no           v1                          false        Node
persistentvolumeclaims            pvc          v1                          true         PersistentVolumeClaim
```

### Listing things

```
$ kubectl get pods
NAME              READY   STATUS    RESTARTS   AGE
hello-7d4b-x1y2z  1/1     Running   0          5m
hello-7d4b-a1b2c  1/1     Running   0          5m
hello-7d4b-d4e5f  1/1     Running   0          5m

$ kubectl get pods -A
NAMESPACE     NAME                       READY   STATUS    RESTARTS   AGE
default       hello-7d4b-x1y2z           1/1     Running   0          5m
kube-system   coredns-565d847f94-abc12   1/1     Running   0          45d
kube-system   etcd-node-01               1/1     Running   0          45d
kube-system   kube-apiserver-node-01     1/1     Running   0          45d
kube-system   kube-proxy-xyz             1/1     Running   0          45d

$ kubectl get deploy,svc,ing -A
NAMESPACE   NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
default     deployment.apps/hello   3/3     3            3           5m

NAMESPACE   NAME                 TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE
default     service/hello        ClusterIP   10.96.5.5     <none>        80/TCP    5m
default     service/kubernetes   ClusterIP   10.96.0.1     <none>        443/TCP   45d

NAMESPACE   NAME                              CLASS   HOSTS                ADDRESS         PORTS   AGE
default     ingress.networking/hello-ingress  nginx   hello.example.com    10.0.0.100      80      5m

$ kubectl get pods -l app=hello
NAME              READY   STATUS    RESTARTS   AGE
hello-7d4b-x1y2z  1/1     Running   0          5m
hello-7d4b-a1b2c  1/1     Running   0          5m
hello-7d4b-d4e5f  1/1     Running   0          5m
```

### Describing

```
$ kubectl describe pod hello-7d4b-x1y2z
Name:         hello-7d4b-x1y2z
Namespace:    default
Priority:     0
Node:         node-02/10.0.0.11
Start Time:   Mon, 13 Apr 2026 09:00:00 +0000
Labels:       app=hello
              pod-template-hash=7d4b
Status:       Running
IP:           10.244.2.7
IPs:
  IP:           10.244.2.7
Controlled By:  ReplicaSet/hello-7d4b
Containers:
  web:
    Container ID:   containerd://abc123def456
    Image:          nginx:1.27
    Image ID:       docker.io/library/nginx@sha256:fee...
    Port:           80/TCP
    State:          Running
      Started:      Mon, 13 Apr 2026 09:00:05 +0000
    Ready:          True
    Restart Count:  0
Events:
  Type    Reason     Age   From               Message
  ----    ------     ----  ----               -------
  Normal  Scheduled  5m    default-scheduler  Successfully assigned default/hello-7d4b-x1y2z to node-02
  Normal  Pulled     5m    kubelet            Container image "nginx:1.27" already present on machine
  Normal  Created    5m    kubelet            Created container web
  Normal  Started    5m    kubelet            Started container web
```

### Logs

```
$ kubectl logs hello-7d4b-x1y2z
/docker-entrypoint.sh: /docker-entrypoint.d/ is not empty, will attempt to perform configuration
/docker-entrypoint.sh: Looking for shell scripts in /docker-entrypoint.d/
2026/04/13 09:00:05 [notice] 1#1: nginx/1.27.0
2026/04/13 09:00:05 [notice] 1#1: start worker processes

$ kubectl logs hello-7d4b-x1y2z --previous
Error from server (BadRequest): previous terminated container "web" in pod "hello-7d4b-x1y2z" not found

$ kubectl logs -f hello-7d4b-x1y2z
(streams logs continuously, like tail -f)

$ kubectl logs hello-7d4b-x1y2z -c web
(specific container in a multi-container Pod)

$ kubectl logs --since=10m -l app=hello
(all logs from last 10m, all Pods matching label)

$ kubectl logs --tail=50 hello-7d4b-x1y2z
(last 50 lines)
```

### Exec and port-forward

```
$ kubectl exec -it hello-7d4b-x1y2z -- /bin/sh
/ # ls /usr/share/nginx/html
50x.html  index.html
/ # cat /etc/hostname
hello-7d4b-x1y2z

$ kubectl exec hello-7d4b-x1y2z -- nginx -v
nginx version: nginx/1.27.0

$ kubectl port-forward svc/hello 8080:80
Forwarding from 127.0.0.1:8080 -> 80
Forwarding from [::1]:8080 -> 80
# now from another terminal:
$ curl localhost:8080
<!DOCTYPE html>
<html>
<head><title>Welcome to nginx!</title></head>
...

$ kubectl port-forward pod/hello-7d4b-x1y2z 8080:80
(forward to a specific Pod instead of through Service)
```

### Events

```
$ kubectl get events --sort-by='.lastTimestamp' | head -10
LAST SEEN   TYPE      REASON              OBJECT                  MESSAGE
5m          Normal    Scheduled           pod/hello-7d4b-x1y2z    Successfully assigned default/hello-7d4b-x1y2z to node-02
5m          Normal    Pulled              pod/hello-7d4b-x1y2z    Container image "nginx:1.27" already present
5m          Normal    Created             pod/hello-7d4b-x1y2z    Created container web
5m          Normal    Started             pod/hello-7d4b-x1y2z    Started container web

$ kubectl get events -n kube-system --field-selector type=Warning
LAST SEEN   TYPE      REASON              OBJECT             MESSAGE
2m          Warning   FailedScheduling    pod/test-pod       0/3 nodes are available: 3 Insufficient memory
```

### Top (resource usage)

```
$ kubectl top nodes
NAME      CPU(cores)   CPU%   MEMORY(bytes)   MEMORY%
node-01   125m         3%     1245Mi          15%
node-02   850m         21%    3450Mi          43%
node-03   320m         8%     2100Mi          26%

$ kubectl top pods -A
NAMESPACE     NAME                       CPU(cores)   MEMORY(bytes)
default       hello-7d4b-x1y2z           5m           20Mi
default       hello-7d4b-a1b2c           4m           19Mi
default       hello-7d4b-d4e5f           3m           18Mi
kube-system   coredns-565d847f94-abc12   2m           15Mi

$ kubectl top pod hello-7d4b-x1y2z --containers
POD              NAME    CPU(cores)   MEMORY(bytes)
hello-7d4b-x1y2z web     5m           20Mi
```

(`kubectl top` requires the metrics-server addon to be installed.)

### Rollouts

```
$ kubectl rollout status deploy/hello
Waiting for deployment "hello" rollout to finish: 1 of 3 updated replicas are available...
Waiting for deployment "hello" rollout to finish: 2 of 3 updated replicas are available...
deployment "hello" successfully rolled out

$ kubectl rollout history deploy/hello
deployment.apps/hello
REVISION  CHANGE-CAUSE
1         <none>
2         <none>
3         kubectl set image deployment/hello web=nginx:1.28

$ kubectl rollout undo deploy/hello
deployment.apps/hello rolled back

$ kubectl rollout undo deploy/hello --to-revision=1
deployment.apps/hello rolled back

$ kubectl rollout pause deploy/hello
deployment.apps/hello paused

$ kubectl rollout resume deploy/hello
deployment.apps/hello resumed
```

### Scale

```
$ kubectl scale deploy/hello --replicas=5
deployment.apps/hello scaled

$ kubectl scale deploy/hello --replicas=0
deployment.apps/hello scaled

$ kubectl get hpa
NAME    REFERENCE          TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
hello   Deployment/hello   23%/70%   2         10        3          1h
```

### Quick debug Pod

```
$ kubectl run debug --image=busybox -it --rm -- sh
If you don't see a command prompt, try pressing enter.
/ # nslookup hello
Server:    10.96.0.10
Address:   10.96.0.10:53

Name:      hello.default.svc.cluster.local
Address: 10.96.5.5

/ # wget -qO- hello
<!DOCTYPE html>...
/ # exit
pod "debug" deleted

$ kubectl run -it --rm --image=nicolaka/netshoot debug -- bash
(more powerful net tools: dig, nmap, tcpdump, traceroute, jq, curl)
```

### Ephemeral debug containers

```
$ kubectl debug -it hello-7d4b-x1y2z --image=busybox --target=web
Defaulting debug container name to debugger-abc12.
If you don't see a command prompt, try pressing enter.
/ # ps aux
PID   USER     TIME  COMMAND
    1 root      0:00 nginx: master process nginx -g daemon off;
   29 nginx     0:00 nginx: worker process
   30 root      0:00 sh
```

This injects a sidecar into the running Pod for live debugging — without restarting it. Available since Kubernetes 1.25 (stable in 1.27+).

### YAML inspection

```
$ kubectl get pod hello-7d4b-x1y2z -o yaml | head -40
apiVersion: v1
kind: Pod
metadata:
  name: hello-7d4b-x1y2z
  namespace: default
  labels:
    app: hello
    pod-template-hash: 7d4b
spec:
  containers:
  - image: nginx:1.27
    name: web
    ports:
    - containerPort: 80
      protocol: TCP
status:
  phase: Running
  podIP: 10.244.2.7
...

$ kubectl get pod hello-7d4b-x1y2z -o jsonpath='{.status.podIP}'
10.244.2.7

$ kubectl get pods -o jsonpath='{.items[*].spec.nodeName}'
node-02 node-03 node-02

$ kubectl get pod hello-7d4b-x1y2z -o json | jq '.status.containerStatuses[].image'
"nginx:1.27"
```

### Built-in docs

```
$ kubectl explain pod.spec.containers.resources
KIND:       Pod
VERSION:    v1

FIELD: resources <ResourceRequirements>

DESCRIPTION:
    Compute Resources required by this container. Cannot be updated.
    More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

FIELDS:
  claims <[]ResourceClaim>
  limits <map[string]Quantity>
  requests <map[string]Quantity>

$ kubectl explain deployment.spec.strategy
$ kubectl explain hpa.spec --recursive
```

`kubectl explain` is the offline manual for every Kubernetes resource. Use it constantly.

### Auth checks

```
$ kubectl auth can-i list pods
yes

$ kubectl auth can-i delete deployments --namespace prod
no

$ kubectl auth can-i '*' '*' --all-namespaces
yes
(only if you're cluster-admin)

$ kubectl auth can-i create pods --as=system:serviceaccount:default:hello-sa
yes
```

### Listing common kube-system stuff

```
$ kubectl get pods -n kube-system | head -10
NAME                              READY   STATUS    RESTARTS   AGE
coredns-565d847f94-abc12          1/1     Running   0          45d
coredns-565d847f94-def34          1/1     Running   0          45d
etcd-node-01                      1/1     Running   0          45d
kube-apiserver-node-01            1/1     Running   0          45d
kube-controller-manager-node-01   1/1     Running   0          45d
kube-proxy-9xyz                   1/1     Running   0          45d
kube-proxy-abc12                  1/1     Running   0          45d
kube-proxy-def34                  1/1     Running   0          45d
kube-scheduler-node-01            1/1     Running   0          45d

$ kubectl get secret -n kube-system | head
NAME                          TYPE                                  DATA   AGE
bootstrap-token-abcdef        bootstrap.kubernetes.io/token         5      45d
default-token-xxxxx           kubernetes.io/service-account-token   3      45d

$ kubectl get configmap -n kube-system | head
NAME                                 DATA   AGE
coredns                              1      45d
kube-apiserver                       1      45d
kube-proxy                           2      45d
kube-root-ca.crt                     1      45d

$ kubectl get sa -A | head
NAMESPACE     NAME                                 SECRETS   AGE
default       default                              0         45d
kube-system   coredns                              0         45d
kube-system   default                              0         45d
```

### Dry run + yaml generation

```
$ kubectl create deployment hello --image=nginx --dry-run=client -o yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: hello
  name: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: hello
    spec:
      containers:
      - image: nginx
        name: nginx
        resources: {}
status: {}

$ kubectl create deployment hello --image=nginx --replicas=3 --dry-run=server -o yaml
(server-side dry run includes admission webhook results)
```

### Apply from heredoc

```
$ kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: quick-cm
data:
  greeting: hello
EOF
configmap/quick-cm created
```

### Node management

```
$ kubectl drain node-01 --ignore-daemonsets --delete-emptydir-data
node/node-01 cordoned
evicting pod default/hello-7d4b-x1y2z
pod/hello-7d4b-x1y2z evicted
node/node-01 drained

$ kubectl cordon node-01
node/node-01 cordoned

$ kubectl uncordon node-01
node/node-01 uncordoned

$ kubectl taint nodes node-01 dedicated=gpu:NoSchedule
node/node-01 tainted

$ kubectl taint nodes node-01 dedicated:NoSchedule-
node/node-01 untainted
```

### Disruption budget

```
$ kubectl get pdb -A
NAMESPACE   NAME           MIN AVAILABLE   MAX UNAVAILABLE   ALLOWED DISRUPTIONS   AGE
default     hello-pdb      2               N/A               1                     2h
```

### In-place edit

```
$ kubectl edit deploy/hello
# opens $EDITOR with the live YAML; save+quit applies changes immediately
deployment.apps/hello edited
```

### Patch

```
$ kubectl patch deploy/hello --patch '{"spec":{"replicas":5}}'
deployment.apps/hello patched

$ kubectl patch deploy/hello --type='json' -p='[{"op":"replace","path":"/spec/replicas","value":5}]'
deployment.apps/hello patched

$ kubectl set image deploy/hello web=nginx:1.28
deployment.apps/hello image updated
```

### Wait

```
$ kubectl wait --for=condition=Ready pod/hello-7d4b-x1y2z --timeout=60s
pod/hello-7d4b-x1y2z condition met

$ kubectl wait --for=condition=available --timeout=300s deploy/hello
deployment.apps/hello condition met

$ kubectl wait --for=delete pod/old-pod --timeout=60s
```

### Diff before apply

```
$ kubectl diff -f new-deploy.yaml
diff -u -N /tmp/LIVE-12345/apps.v1.Deployment.default.hello /tmp/MERGED-12345/apps.v1.Deployment.default.hello
--- /tmp/LIVE-12345/apps.v1.Deployment.default.hello
+++ /tmp/MERGED-12345/apps.v1.Deployment.default.hello
@@ -10,7 +10,7 @@
   replicas: 3
   template:
     spec:
-      containers: [{image: nginx:1.27}]
+      containers: [{image: nginx:1.28}]
```

### Context and namespace

```
$ kubectl config get-contexts
CURRENT   NAME              CLUSTER         AUTHINFO     NAMESPACE
*         dev-cluster       dev-cluster     dev-user     default
          prod-cluster      prod-cluster    prod-user    production

$ kubectl config use-context prod-cluster
Switched to context "prod-cluster".

$ kubectl config set-context --current --namespace=prod
Context "prod-cluster" modified.

$ kubectl get pods --context=dev-cluster --namespace=default
```

(Tools like `kubectx` and `kubens` make this painless.)

## Common Confusions

12+ broken-then-fixed pairs.

### "Why is my Pod 'Pending' forever?"

Almost always one of:

1. No node has enough resources for the Pod's `requests`. Lower the requests, or add a bigger node.
2. The Pod's `nodeSelector` or `affinity` doesn't match any node. `kubectl get nodes --show-labels` to see node labels.
3. The Pod's tolerations don't cover a node taint. `kubectl describe nodes | grep Taint`.
4. The Pod requires a PVC that's stuck in `Pending`. `kubectl get pvc` to check.

`kubectl describe pod <name>` and look at the Events. The scheduler explains itself: `0/3 nodes are available: 3 Insufficient cpu`.

### "Why does kubectl exec hang?"

Usually nothing's wrong. The container started a TTY shell and is waiting for input. Try typing.

If `kubectl exec` actually hangs forever, suspect:
- Network issue between you and the API server.
- API server is overloaded.
- Kubelet on the node is unhealthy.

### "Why doesn't my Service work? It's there but nothing answers."

Three checks, in order:

1. **Endpoints empty?** `kubectl get endpoints svc/<name>`. If `<none>`, your selector doesn't match any Pods (or no matching Pods are Ready).
2. **Pod not ready?** `kubectl get pods -l <selector>` — are they `1/1 Ready`? If not, fix readiness probes.
3. **Network policy blocking?** `kubectl get networkpolicy -A`. Are there any rules that would block your traffic?

The `Endpoints` object is the single source of truth for what a Service routes to. If it's empty, the Service has nowhere to send traffic.

### "Why does my container restart constantly?"

Check exit code. `kubectl describe pod` and look at `Last State`. If exit code is non-zero, the app crashed. Check logs (`kubectl logs --previous`).

If exit code is 0, your liveness probe is failing or the app intentionally exits. Long-running services should never exit on their own.

### "Should I use a Deployment or a StatefulSet?"

Default to Deployment. Use StatefulSet only when you need:

- **Stable hostnames** (each Pod gets `name-0`, `name-1`, ...).
- **Stable persistent storage** (each Pod gets its own dedicated PVC that follows it).
- **Ordered startup/teardown** (Pod 0 comes up before Pod 1; teardown is reverse).

Databases, message brokers, leader-elected systems, distributed consensus — StatefulSet. Web servers, APIs, batch workers, anything stateless — Deployment.

### "What's the difference between ConfigMap and Secret?"

Both are key-value blobs. Both can be mounted as files or env vars.

The differences are mostly cultural:
- Secret values are base64-encoded in YAML (so you can put binary data in YAML).
- Secret has different RBAC defaults (typically more restricted).
- Secret can be encrypted at rest if `EncryptionConfiguration` is set on the API server.

Out of the box, **a Secret is NOT encrypted at rest** — it's just base64-encoded, which is trivially decodable. Anyone with read access to etcd can read your Secrets. For real protection, configure encryption at rest, or use an external secrets manager (Vault, AWS Secrets Manager, GCP Secret Manager) with the External Secrets Operator.

### "Why is my Ingress returning 404?"

Possibilities:

1. **No host header match.** Curl with `-H "Host: hello.example.com"` to test.
2. **No path match.** Check the Ingress rule paths.
3. **Ingress controller not running.** `kubectl get pods -n ingress-nginx`.
4. **Service has no endpoints.** Same root cause as the Service issues above.
5. **Wrong `ingressClassName`.** If your cluster has multiple Ingress controllers, the class matters.
6. **TLS issue.** If using HTTPS, check the cert is mounted correctly and matches the host.

### "Why does helm template differ from helm install?"

`helm template` only renders the templates locally. `helm install` also:

1. Validates the rendered YAML against the cluster's API.
2. Records the release in the cluster.
3. Runs admission webhooks (which can mutate or reject the manifest).
4. Catches conflicts with existing resources.

`helm template | kubectl apply -f -` skips Helm's release tracking. Avoid it; use `helm install` or `helm upgrade --install`.

### "What's a sidecar?"

A second (or third) container in the same Pod, running alongside the main app. Common sidecars:

- **Service mesh proxy** (Istio's `envoy`, Linkerd's `linkerd-proxy`).
- **Log collector** (Filebeat shipping app logs to a central system).
- **Cloud SQL proxy** (Google's database proxy as a sidecar).
- **OAuth2 proxy** (terminating auth before the app sees the request).
- **Vault Agent** (renewing secrets).

Sidecars share network and storage with the main container. They start with the Pod and die with the Pod.

In Kubernetes 1.28+ there's official `restartPolicy: Always` on init containers, which makes "real" sidecars (started before main, restarted independently) a first-class feature. Before that, sidecars were just regular containers in the Pod.

### "Why does kubectl logs work for one Pod but not another?"

The second Pod has multiple containers, and you didn't specify which:

```
$ kubectl logs my-pod
error: a container name must be specified for pod my-pod, choose one of: [app sidecar]

$ kubectl logs my-pod -c app
(now it works)
```

Use `-c <container>` for multi-container Pods.

### "Why isn't my env var getting through?"

If you wrote:

```yaml
env:
- name: GREETING
  value: "Hello, $(USER)!"
```

You probably expected the shell to expand `$(USER)`. It didn't. `value:` is a literal string. Use `valueFrom` for dynamic values, or do the substitution in your container's entrypoint.

Also: changes to a ConfigMap or Secret only show up in already-running Pods if you mount them as files (the kubelet refreshes mounted ConfigMaps every minute or so). If you reference them as `valueFrom: configMapKeyRef`, the values are baked into the Pod at startup and won't update until you restart the Pod.

### "Why doesn't my CronJob run?"

Check:

1. **`schedule:` syntax.** It's standard cron format. Test with an online cron parser.
2. **Concurrency policy.** `concurrencyPolicy: Forbid` skips runs if the previous is still going. `Replace` kills the old to start the new. `Allow` allows overlap.
3. **Cluster time.** The control plane uses UTC by default. If you wrote `0 9 * * *` expecting 9am local, you got 9am UTC.
4. **`startingDeadlineSeconds`.** If the controller missed the start window (because of a control plane outage, say), the run is skipped.
5. **`suspend: true`.** A CronJob can be paused.

`kubectl get jobs --watch` to see if Jobs are being created. `kubectl get events` for errors.

### "Why is my readiness probe failing on a healthy app?"

Common causes:

- Probe URL wrong (returns 404).
- Probe expects HTTP 200, app returns 301.
- `initialDelaySeconds` too low — app hasn't finished starting.
- Probe runs in a TCP-only mode but app uses TLS, or vice versa.
- Probe runs from inside the Pod (different network namespace than you'd expect).

Test by hand: `kubectl exec -it <pod> -- wget -qO- http://localhost:8080/healthz`.

### "Why do I see 'Forbidden' calling the Kubernetes API from inside a Pod?"

The Pod's ServiceAccount doesn't have permission. Check:

1. `kubectl get pod <name> -o yaml | grep serviceAccountName` — is it the right SA?
2. `kubectl get rolebinding,clusterrolebinding -A | grep <sa-name>` — is it bound to a role?
3. The Role has the required verbs/resources.

Quick check: `kubectl auth can-i <verb> <resource> --as=system:serviceaccount:<namespace>:<sa>`.

### "Why does my new image not get pulled?"

If you used a tag that already exists in the kubelet's cache (like `:latest`), the kubelet skips the pull. Solutions:

1. Use `imagePullPolicy: Always`.
2. Use a unique tag per build (`:v1.2.3`, `:abc123` from a git SHA). Never reuse tags.

```yaml
containers:
- name: app
  image: my-app:latest
  imagePullPolicy: Always   # always pull
```

### "Why is `kubectl apply` complaining about field immutability?"

Some fields are immutable after creation. The Service's `clusterIP` is one. The Job's `selector` is another. The error looks like:

```
The Service "myservice" is invalid: spec.clusterIP: Invalid value: "": field is immutable
```

Solution: delete the resource and recreate it, or only change mutable fields.

## Vocabulary

100+ entries. Look here first when a word is weird.

| Term | Plain English |
|------|---------------|
| **Pod** | The smallest schedulable thing. One or more containers sharing a network and storage. |
| **container** | A packaged program with its files. Smallest building block. |
| **Deployment** | "I want N copies of this Pod, and a graceful update path." Most-used object. |
| **ReplicaSet** | The internal piece a Deployment uses to keep N Pods running. |
| **StatefulSet** | Like Deployment but with stable hostnames + storage. For databases. |
| **DaemonSet** | "Run one Pod on every Node." For per-node services. |
| **Job** | One-shot batch task. Runs Pods to completion. |
| **CronJob** | Job on a schedule. |
| **Service** | A stable virtual IP + DNS name in front of a set of Pods. |
| **ClusterIP** | Service type: only reachable inside the cluster. Default. |
| **NodePort** | Service type: opens a port on every node for outside access. |
| **LoadBalancer** | Service type: provisions a cloud load balancer. |
| **ExternalName** | Service type: a DNS alias to an external host. No real routing. |
| **headless service** | A Service with `clusterIP: None`. DNS returns Pod IPs directly. Used by StatefulSets. |
| **Endpoints** | The list of Pod IPs a Service routes to. Auto-managed. |
| **EndpointSlice** | The newer, more scalable replacement for Endpoints. |
| **Ingress** | HTTP/HTTPS router from outside to internal Services. |
| **IngressClass** | Picks which Ingress Controller handles which Ingress. |
| **Gateway API** | Newer routing API replacing Ingress. More expressive. |
| **HTTPRoute** | A routing rule under Gateway API. |
| **Namespace** | A logical folder for objects. |
| **ConfigMap** | Bag of key-value config strings. Mountable as env vars or files. |
| **Secret** | Like ConfigMap but for sensitive data. Base64-encoded, NOT encrypted by default. |
| **EncryptionConfiguration** | Cluster-level config that encrypts Secrets at rest in etcd. |
| **ServiceAccount** | Identity for Pods. Used for RBAC. |
| **Role** | RBAC: list of allowed verbs/resources, scoped to one namespace. |
| **ClusterRole** | RBAC: same but cluster-wide. |
| **RoleBinding** | Binds a Role to a ServiceAccount/user/group. |
| **ClusterRoleBinding** | Binds a ClusterRole to a subject across the cluster. |
| **RBAC** | Role-Based Access Control. The "who can do what" system. |
| **control plane** | The brains of Kubernetes — api-server, scheduler, controller-manager, etcd. |
| **kube-apiserver** | The front door. Every read/write to the cluster goes through it. |
| **etcd** | The cluster's database. Stores all object state. Raft-based. |
| **kube-controller-manager** | The bag of controllers (Deployment, ReplicaSet, Job, etc). |
| **kube-scheduler** | Decides which node each Pod runs on. |
| **kubelet** | The agent on each node. Runs Pods, reports back. |
| **kube-proxy** | The component that programs iptables/ipvs to make Services work. |
| **cri-o** | A container runtime. Pulls images, starts containers. |
| **containerd** | The most common container runtime today. |
| **runc** | The lowest-level runtime, actually creates the container processes. |
| **gVisor** | A sandboxed container runtime that intercepts syscalls. |
| **Kata Containers** | A VM-based container runtime for stronger isolation. |
| **CNI** | Container Network Interface. The pluggable Pod networking layer. |
| **Cilium** | A CNI based on eBPF. Very fast, fancy NetworkPolicy. |
| **Calico** | A CNI based on BGP. No overlay required. |
| **Flannel** | A simple overlay-network CNI. |
| **Weave Net** | An encrypted overlay CNI. |
| **CSI** | Container Storage Interface. The pluggable storage layer. |
| **PV (PersistentVolume)** | A chunk of storage available to the cluster. |
| **PVC (PersistentVolumeClaim)** | A Pod's request for storage. Gets bound to a PV. |
| **StorageClass** | The "recipe" for dynamic PV provisioning. |
| **VolumeAttachment** | The CSI object tracking a volume attached to a node. |
| **CRD (CustomResourceDefinition)** | Teaches the API a new object type. |
| **CR (CustomResource)** | An instance of a CRD. |
| **Operator** | A controller that manages a CRD. Custom logic for custom objects. |
| **OperatorHub** | A registry of pre-built Operators. |
| **Helm** | Templating + release manager for Kubernetes. |
| **chart** | A Helm package. Directory of templates and values. |
| **values.yaml** | Helm chart's default values. Override per install. |
| **release** | A specific install of a Helm chart. |
| **repo** | A collection of Helm charts. Like a registry. |
| **Kustomize** | Overlay-merge alternative to Helm. Plain YAML, no templating. |
| **base** | Kustomize's set of "default" YAML manifests. |
| **overlay** | Kustomize's environment-specific patches on top of a base. |
| **patch** | A diff applied by Kustomize (or kubectl). Strategic merge or JSON patch. |
| **kustomization.yaml** | The Kustomize config file in a base or overlay. |
| **label** | Key-value tag, queryable, used for selection. |
| **annotation** | Key-value tag, NOT queryable, used for unstructured metadata. |
| **selector** | Query that matches labels. |
| **matchLabels** | Selector: exact key-value match. |
| **matchExpressions** | Selector: more flexible (In, NotIn, Exists, DoesNotExist). |
| **affinity** | Scheduler hint: "schedule near these labeled things." |
| **anti-affinity** | Scheduler hint: "schedule away from these labeled things." |
| **taint** | Marker on a node that repels Pods. |
| **toleration** | Pod's opt-in to a taint. |
| **NodeSelector** | Simple version of node affinity. |
| **PodAntiAffinity** | Spread Pods away from each other. |
| **topologySpreadConstraints** | Spread Pods across zones/regions/etc. |
| **PriorityClass** | Per-Pod priority. Higher priority can preempt lower. |
| **PodDisruptionBudget (PDB)** | "Don't let more than N of my Pods be unavailable at once." |
| **HPA (HorizontalPodAutoscaler)** | Auto-scales replicas based on CPU/memory/custom metrics. |
| **VPA (VerticalPodAutoscaler)** | Auto-tunes a Pod's resource requests. |
| **ClusterAutoscaler** | Auto-adds/removes nodes. |
| **Karpenter** | An aggressive AWS-native cluster autoscaler / scheduler. |
| **kubectl** | The main CLI tool. |
| **kubectx** | CLI tool for switching kubectl contexts. |
| **kubens** | CLI tool for switching default namespace. |
| **k9s** | A terminal UI for Kubernetes. |
| **lens** | A graphical desktop UI for Kubernetes. |
| **liveness probe** | "Is the container still alive?" Failure → restart. |
| **readiness probe** | "Is the container ready for traffic?" Failure → remove from Endpoints. |
| **startup probe** | "Has the container finished starting?" Suppresses liveness during startup. |
| **exec probe** | Probe that runs a command inside the container. |
| **http probe** | Probe that does an HTTP GET. |
| **tcp probe** | Probe that opens a TCP connection. |
| **gRPC probe** | Probe that calls the gRPC health protocol. |
| **init container** | A container that runs to completion before the main container starts. |
| **sidecar** | A second container in the Pod. Long-running. |
| **ephemeralContainer** | An on-demand debug container injected into a running Pod. |
| **preStop hook** | Action run before a container is terminated. |
| **postStart hook** | Action run right after a container starts. |
| **terminationGracePeriodSeconds** | How long Kubernetes waits before SIGKILL after SIGTERM. |
| **finalizer** | A "this object can't be deleted until X happens" marker. |
| **ownerReference** | A "this object is owned by that one" link. Used for garbage collection. |
| **garbage collection** | Auto-removing children when their parent is deleted. |
| **resource request** | "I need at least this much CPU/memory." Used for scheduling. |
| **resource limit** | "Don't let me use more than this." Enforced by the kernel. |
| **QoS class** | Pod's quality-of-service: Guaranteed (req=lim), Burstable (req<lim), BestEffort (none). |
| **OOMKilled** | Killed by the kernel for exceeding memory limit. |
| **Evicted** | Removed by the kubelet because the node is under pressure. |
| **ImagePullBackOff** | Failed to pull image. |
| **CrashLoopBackOff** | Container keeps exiting. Backing off restarts. |
| **ContainerCreating** | Pod is being prepared (image pull, volume mount). |
| **Pending** | Pod is accepted but not yet scheduled or starting. |
| **Running** | Pod is bound and at least one container is up. |
| **Succeeded** | All containers exited with success. |
| **Failed** | At least one container exited non-zero, no restart pending. |
| **Unknown** | Lost contact with kubelet. |
| **kubeconfig** | Config file for kubectl. Has clusters, users, contexts. |
| **context** | Combination of cluster + user + namespace in kubeconfig. |
| **cluster** | A kubeconfig entry pointing at a Kubernetes API server. |
| **server** | The URL of the Kubernetes API server. |
| **user** | A kubeconfig entry with credentials. |
| **apiVersion** | YAML field: which API version this object uses. |
| **kind** | YAML field: which type of object this is. |
| **metadata** | YAML field: name, namespace, labels, annotations. |
| **spec** | YAML field: the desired state. |
| **status** | YAML field: the current state (set by Kubernetes). |
| **dryRun** | "Pretend to apply but don't actually." |
| **server-side apply (SSA)** | Modern way to apply YAML where the server tracks ownership of fields. |
| **strategic merge patch** | A patch format that knows about Kubernetes semantics. |
| **JSON patch** | RFC 6902 patch format. Operations like add/remove/replace. |
| **merge patch** | Simple JSON merge. |
| **node** | A worker machine running kubelet. |
| **master** | Old term for control plane node. Deprecated. |
| **control plane node** | A node running control plane components (api-server, etc). |
| **worker node** | A node that just runs Pods. |
| **kind (the tool)** | Kubernetes IN Docker. Spins up a local cluster in containers. |
| **minikube** | Single-node local Kubernetes for development. |
| **k3s** | A lightweight Kubernetes distribution. |
| **k0s** | Another lightweight Kubernetes distribution. |
| **kubeadm** | The official tool for installing a real Kubernetes cluster. |
| **EKS / GKE / AKS** | AWS / Google / Azure managed Kubernetes services. |
| **multi-tenancy** | Multiple teams/users sharing one cluster. |
| **tenant** | One team/user in a multi-tenant cluster. |

## Try This

Ten experiments that teach Kubernetes by doing.

### 1. Spin up a local cluster

```
$ brew install kind                     # macOS
$ kind create cluster --name learn
Creating cluster "learn" ...
 ✓ Ensuring node image
 ✓ Preparing nodes
 ✓ Writing configuration
 ✓ Starting control-plane
 ✓ Installing CNI
 ✓ Installing StorageClass
Set kubectl context to "kind-learn"

$ kubectl get nodes
NAME                  STATUS   ROLES           AGE   VERSION
learn-control-plane   Ready    control-plane   30s   v1.31.0
```

You now have a one-node Kubernetes cluster running locally in Docker. Free, no cloud bill.

### 2. Run hello world

```
$ kubectl create deployment hello --image=nginx
deployment.apps/hello created

$ kubectl get pods
NAME                     READY   STATUS    RESTARTS   AGE
hello-7d4b8d6c55-abc12   1/1     Running   0          10s

$ kubectl expose deployment hello --port=80
service/hello exposed

$ kubectl port-forward svc/hello 8080:80
Forwarding from 127.0.0.1:8080 -> 80
```

Open http://localhost:8080 in your browser. You should see "Welcome to nginx!"

### 3. Force a CrashLoopBackOff

Save as `crash.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: crashy
spec:
  containers:
  - name: failer
    image: busybox
    command: ["sh", "-c", "echo starting; sleep 1; exit 1"]
```

```
$ kubectl apply -f crash.yaml
$ watch kubectl get pod crashy
crashy   0/1     CrashLoopBackOff   3 (10s ago)   30s
$ kubectl logs crashy
starting
$ kubectl describe pod crashy | grep -A5 "Last State"
    Last State:     Terminated
      Reason:       Error
      Exit Code:    1
$ kubectl delete pod crashy
```

You just saw exactly what `CrashLoopBackOff` means. The container ran, exited 1, ran again, exited 1, and Kubernetes is starting to wait longer between attempts.

### 4. Force an OOMKilled

Save as `oom.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hungry
spec:
  containers:
  - name: hog
    image: polinux/stress
    args: ["--vm", "1", "--vm-bytes", "200M", "--vm-hang", "1"]
    resources:
      limits:
        memory: 100Mi
```

```
$ kubectl apply -f oom.yaml
$ kubectl get pod hungry
hungry   0/1     OOMKilled   2 (5s ago)   30s
$ kubectl describe pod hungry | grep -A2 "Last State"
    Last State:     Terminated
      Reason:       OOMKilled
      Exit Code:    137
$ kubectl delete pod hungry
```

The container asked for 200MB but the limit was 100MB. The kernel killed it. Exit code 137 is the OOM killer's signature.

### 5. Watch a rolling update

```
$ kubectl create deployment rolling --image=nginx:1.25 --replicas=4
$ kubectl rollout status deploy/rolling
$ kubectl set image deploy/rolling nginx=nginx:1.27
$ kubectl rollout status deploy/rolling
Waiting for deployment "rolling" rollout to finish: 1 of 4 updated replicas are available...
Waiting for deployment "rolling" rollout to finish: 2 of 4 updated replicas are available...
deployment "rolling" successfully rolled out
$ kubectl rollout history deploy/rolling
```

Watch in another terminal: `watch kubectl get pods`. You'll see Pods with the old `pod-template-hash` getting replaced one at a time by Pods with a new hash.

### 6. Roll back

```
$ kubectl rollout undo deploy/rolling
deployment.apps/rolling rolled back
$ kubectl rollout history deploy/rolling
$ kubectl describe deploy rolling | grep Image
    Image:        nginx:1.25
```

You went forward, then went back. No downtime either way.

### 7. Use kubectl debug on a running Pod

```
$ kubectl run quiet --image=nginx
pod/quiet created
$ kubectl debug -it quiet --image=nicolaka/netshoot --target=quiet
Defaulting debug container name to debugger-xyz
quiet:~$ ps aux
quiet:~$ curl localhost:80
quiet:~$ ss -tlnp
quiet:~$ exit
```

You added a debug container with extra tools, sharing the network namespace with the real container. No image rebuild required.

### 8. Inspect events when things go wrong

```
$ kubectl run badpull --image=ngnix:1.27   # typo
$ kubectl get pod badpull
badpull   0/1     ErrImagePull   0   30s
$ kubectl describe pod badpull
$ kubectl get events --sort-by='.lastTimestamp' | tail -10
$ kubectl delete pod badpull
```

Read the events. They explain everything Kubernetes is doing about your Pod.

### 9. Try kubectl explain

```
$ kubectl explain pod
$ kubectl explain pod.spec
$ kubectl explain pod.spec.containers
$ kubectl explain pod.spec.containers.resources
$ kubectl explain hpa.spec.metrics
```

The whole API documented offline. Use this whenever you're not sure what fields exist.

### 10. Apply a NetworkPolicy

```yaml
# np-default-deny.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny
  namespace: default
spec:
  podSelector: {}
  policyTypes:
  - Ingress
```

```
$ kubectl apply -f np-default-deny.yaml
$ kubectl run client --image=busybox -it --rm -- wget -qO- --timeout=2 hello
wget: download timed out
$ kubectl delete networkpolicy default-deny
$ kubectl run client --image=busybox -it --rm -- wget -qO- --timeout=2 hello
<!DOCTYPE html>...
```

(Requires a CNI that enforces NetworkPolicy. kind uses kindnet which doesn't, so swap to Calico for this one: `kind delete cluster --name learn && kind create cluster --config kind-calico.yaml`. Or just imagine it works.)

You created a deny-all rule and watched traffic stop. Removed it and watched traffic flow. That's NetworkPolicy in one paragraph.

### 11. Explore RBAC

```
$ kubectl create namespace test
$ kubectl create serviceaccount lookup -n test
$ kubectl create role pod-reader --verb=get,list,watch --resource=pods -n test
$ kubectl create rolebinding lookup-pods --role=pod-reader --serviceaccount=test:lookup -n test
$ kubectl auth can-i list pods --as=system:serviceaccount:test:lookup -n test
yes
$ kubectl auth can-i list pods --as=system:serviceaccount:test:lookup -n default
no
$ kubectl auth can-i delete pods --as=system:serviceaccount:test:lookup -n test
no
```

You created an identity, gave it minimal permissions, and verified what it can and can't do.

### 12. Tear down the cluster

```
$ kind delete cluster --name learn
Deleting cluster "learn" ...
```

All gone. No bill. Repeat as often as you like.

## Where to Go Next

You now know enough Kubernetes to do real work. From here:

- `cs orchestration kubernetes` — dense, no-fluff reference for everything in this sheet.
- `cs detail orchestration/kubernetes` — the deep theory: scheduler scoring algorithm, Raft consensus inside etcd, HPA math, controller-runtime internals.
- `cs orchestration kubectl` — every kubectl flag explained.
- `cs orchestration kubectl-debug` — debugging tooling beyond kubectl logs.
- `cs orchestration helm` — Helm in detail.
- `cs orchestration kustomize` — Kustomize in detail.
- `cs orchestration argocd` — GitOps continuous delivery for Kubernetes.
- `cs orchestration argo-rollouts` — canary and blue-green deployment strategies.
- `cs orchestration gateway-api` — the next-generation Ingress.
- `cs orchestration cert-manager` — automatic TLS certificates from Let's Encrypt and others.
- `cs orchestration external-secrets` — pulling secrets from Vault, AWS Secrets Manager, etc.
- `cs orchestration kyverno` — policy-as-code for admission control.
- `cs orchestration opa` — OpenPolicyAgent for cluster-wide policy.
- `cs orchestration operator` — building your own Operator with kubebuilder or Operator SDK.
- `cs troubleshooting kubernetes-errors` — error → fix lookup table.
- `cs containers docker` — the container basics that this sheet skipped over.
- `cs containers podman` — Docker's daemonless cousin.
- `cs containers containerd` — the runtime kubelet actually uses.
- `cs service-mesh istio` — service mesh built on Envoy sidecars.
- `cs service-mesh cilium` — eBPF-based service mesh.
- `cs ci-cd argocd` — same tool, different lens (CI/CD perspective).
- `cs ramp-up linux-kernel-eli5` — the kernel features that make containers work.
- `cs ramp-up tcp-eli5` — TCP, the protocol underneath everything.
- `cs ramp-up tls-eli5` — TLS, encryption between Services.

## See Also

- `orchestration/kubernetes` — dense reference
- `orchestration/kubectl` — CLI reference
- `orchestration/kubectl-debug` — debugging commands
- `orchestration/helm` — Helm packaging
- `orchestration/kustomize` — Kustomize overlays
- `orchestration/argocd` — GitOps CD
- `orchestration/argo-rollouts` — progressive delivery
- `orchestration/gateway-api` — next-gen ingress
- `orchestration/cert-manager` — TLS certificates
- `orchestration/external-secrets` — secrets from external stores
- `orchestration/kyverno` — policy engine
- `orchestration/opa` — Open Policy Agent
- `orchestration/operator` — building Operators
- `troubleshooting/kubernetes-errors` — error → fix
- `containers/docker` — Docker basics
- `containers/podman` — Podman alternative
- `containers/containerd` — runtime details
- `service-mesh/istio` — Istio mesh
- `service-mesh/cilium` — Cilium mesh
- `orchestration/argocd` — ArgoCD pipelines
- `ramp-up/linux-kernel-eli5` — kernel basics
- `ramp-up/tcp-eli5` — TCP basics
- `ramp-up/tls-eli5` — TLS basics

## References

- kubernetes.io/docs — the official documentation
- "Kubernetes in Action" by Marko Lukša — the gentle introduction book
- "Kubernetes Up & Running" by Brendan Burns, Joe Beda, Kelsey Hightower, Lachlan Evenson — the practical book
- "Programming Kubernetes" by Michael Hausenblas, Stefan Schimanski — for writing controllers and Operators
- `man kubectl`, `kubectl explain <resource>` — built-in documentation
- KEPs (Kubernetes Enhancement Proposals): github.com/kubernetes/enhancements — proposals for every change
- The Helm chart spec: helm.sh/docs/topics/charts
- The Kustomize spec: kubectl.docs.kubernetes.io/references/kustomize
- CNCF landscape: landscape.cncf.io — the entire cloud-native ecosystem mapped out
- The Kubernetes API reference: kubernetes.io/docs/reference/generated/kubernetes-api
- The official Kubernetes blog: kubernetes.io/blog
- KubeCon talks on YouTube — every major topic has a deep dive
