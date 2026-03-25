# containerd (Industry-Standard Container Runtime)

Low-level container runtime used by Docker and Kubernetes. Managed via `ctr` (low-level) or `nerdctl` (Docker-compatible CLI).

## ctr Basics

### Namespaces

```bash
ctr namespaces list                        # list all namespaces
ctr namespaces create testing
ctr namespaces remove testing
ctr -n k8s.io containers list             # use kubernetes namespace
```

Note: Docker uses the `moby` namespace; Kubernetes uses `k8s.io`.

## Images (ctr)

### Pull and manage images

```bash
ctr images pull docker.io/library/nginx:alpine
ctr images pull --platform linux/amd64 docker.io/library/golang:1.24
ctr images list
ctr images list -q                         # names only
ctr images check                           # verify image content
ctr images remove docker.io/library/nginx:alpine
ctr images tag docker.io/library/nginx:alpine myregistry.io/nginx:v1
```

### Import and export

```bash
ctr images export nginx.tar docker.io/library/nginx:alpine
ctr images import nginx.tar
ctr images import --base-name myapp app.tar   # set image name on import
```

## Containers (ctr)

### Run containers

```bash
ctr run -d docker.io/library/nginx:alpine web    # detached
ctr run --rm -t docker.io/library/alpine:latest shell /bin/sh   # interactive, auto-remove
ctr run -d --net-host docker.io/library/nginx:alpine web   # host networking
```

### Manage containers

```bash
ctr containers list
ctr containers info web
ctr containers delete web
```

### Tasks (running processes)

```bash
ctr task list                              # running tasks
ctr task attach web                        # attach to running task
ctr task exec --exec-id shell1 -t web /bin/sh   # exec into container
ctr task kill web                          # send SIGTERM
ctr task kill --signal SIGKILL web
ctr task delete web
ctr task pause web
ctr task resume web
```

## Content Store

```bash
ctr content list                           # list all content blobs
ctr content get sha256:abc123...           # fetch specific blob
ctr content fetch docker.io/library/nginx:alpine   # download without unpacking
```

## Snapshots

```bash
ctr snapshots list                         # list filesystem snapshots
ctr snapshots info sha256:abc123
ctr snapshots remove mysnap
ctr snapshots tree                         # show parent-child relationships
ctr snapshots usage sha256:abc123          # disk usage
```

## nerdctl (Docker-Compatible CLI)

### Run containers

```bash
nerdctl run -d --name web -p 8080:80 nginx:alpine
nerdctl run -it --rm alpine sh
nerdctl run -d -v mydata:/data --name db postgres:16
```

### Build images

```bash
nerdctl build -t myapp:latest .
nerdctl build -t myapp:v1 --platform linux/amd64,linux/arm64 .
```

### Compose support

```bash
nerdctl compose up -d
nerdctl compose down
nerdctl compose logs -f
```

### Container management

```bash
nerdctl ps
nerdctl ps -a
nerdctl logs -f web
nerdctl exec -it web sh
nerdctl stop web
nerdctl rm web
```

### Image management

```bash
nerdctl images
nerdctl pull nginx:alpine
nerdctl push myregistry.io/myapp:v1
nerdctl tag nginx:alpine myregistry.io/nginx:v1
nerdctl rmi nginx:alpine
nerdctl image prune -a
```

### Namespace selection

```bash
nerdctl -n k8s.io ps                       # see kubernetes containers
nerdctl -n moby images                     # see docker images
nerdctl namespace list
```

## Volume and Network (nerdctl)

```bash
nerdctl volume create mydata
nerdctl volume ls
nerdctl volume rm mydata
nerdctl network create mynet
nerdctl network ls
nerdctl network rm mynet
```

## System

```bash
nerdctl system prune -a                    # clean everything unused
nerdctl system info
nerdctl info                               # runtime info
```

## Tips

- `ctr` is for debugging and low-level operations; use `nerdctl` for a Docker-like experience.
- containerd namespaces isolate images, containers, and content. Docker (`moby`) and Kubernetes (`k8s.io`) use separate namespaces.
- `ctr` separates "containers" (config) from "tasks" (running processes). You create a container, then start a task in it.
- `nerdctl` supports most `docker` CLI flags and even docker-compose files.
- Images pulled with Docker are not visible to `ctr` unless you specify `-n moby`.
- `ctr run` is a convenience wrapper around `ctr container create` + `ctr task start`.
- For rootless containerd, use `containerd-rootless-setuptool.sh install` and prefix commands with `nerdctl` (rootless by default with rootless containerd).
- containerd uses OCI image spec; no proprietary image format.
