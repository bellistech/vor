# Docker (Container Runtime + Tooling)

Build, ship, and run OCI containers via the Docker CLI, BuildKit, Compose v2, and the containerd/runc runtime stack.

## Setup

### Install Docker Engine (Linux)

```bash
# Debian / Ubuntu — official convenience script (dev/test only)
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER"   # log out + back in for group to apply

# Debian / Ubuntu — apt repo (preferred for prod)
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release; echo "$VERSION_CODENAME") stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# RHEL / CentOS / Fedora
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker
```

### Docker Desktop (macOS, Windows)

```bash
# macOS — Homebrew cask (or download .dmg from docker.com)
brew install --cask docker

# Windows — winget
winget install Docker.DockerDesktop

# Docker Desktop runs a hidden Linux VM (HyperKit/Virtualization.framework on
# macOS, WSL2 or Hyper-V on Windows). All `docker` commands talk to dockerd
# inside that VM via a Unix socket forwarded to the host.
# CPU/RAM caps live in the GUI: Settings → Resources, NOT in `docker run`.
```

### Rootless Docker (Linux)

```bash
# Install the rootless toolkit (Ubuntu 22.04+)
sudo apt-get install -y dbus-user-session uidmap docker-ce-rootless-extras
dockerd-rootless-setuptool.sh install
# Adds systemd --user unit; daemon at $XDG_RUNTIME_DIR/docker.sock
export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock
systemctl --user enable --now docker
loginctl enable-linger "$USER"   # survive logout

# Limitations of rootless mode:
# - Cannot bind ports < 1024 without setcap (or use slirp4netns + setcap on rootlesskit)
# - overlay2 requires kernel 5.13+ for unprivileged user mounts
# - No AppArmor/SELinux enforcement on container processes
# - cgroup v2 required for resource limits
# - Some storage drivers (zfs, devicemapper) not supported
```

### Podman (drop-in alternative)

```bash
sudo apt-get install -y podman
alias docker=podman               # docker-compatible CLI
podman run hello-world
# Daemonless; uses fork/exec model. Native rootless. Pods (group of containers
# sharing a network namespace) modeled directly. Compatible with most docker
# commands and Dockerfile syntax.
```

### Verify Install

```bash
docker version       # client + server (engine) versions; will fail if daemon down
docker info          # storage driver, cgroup version, runtime, registry mirrors
docker run --rm hello-world   # full pull → create → start → exit cycle
```

## Architecture

```bash
# ┌─────────┐    REST/gRPC over /var/run/docker.sock    ┌──────────┐
# │ docker  │ ─────────────────────────────────────────▶ │ dockerd  │
# │  CLI    │                                            │ (engine) │
# └─────────┘                                            └────┬─────┘
#                                                             │
#                                       containerd gRPC API   ▼
#                                                       ┌────────────┐
#                                                       │ containerd │
#                                                       └─────┬──────┘
#                                                             │
#                                                  spawns runc per container
#                                                             ▼
#                                                       ┌────────────┐
#                                                       │   runc     │
#                                                       │ (OCI exec) │
#                                                       └─────┬──────┘
#                                                             │
#                                                       Linux namespaces +
#                                                       cgroups + capabilities
```

- **dockerd** — long-running daemon, owns image storage, networks, volumes, build coordination
- **containerd** — image-pull / snapshot / runtime supervisor; can be used directly with `nerdctl`
- **runc** — reference implementation of the OCI Runtime Spec (creates the actual container process)
- **BuildKit** — modern parallel/cached builder; default in Docker 23+; replaces the legacy builder
- **Compose v2** — Go rewrite shipped as the `docker compose` CLI plugin (NOT `docker-compose` v1 Python)

## Hello World

```bash
docker run hello-world
# Equivalent to:
docker pull hello-world         # fetch image from registry
docker create hello-world       # snapshot image into a writable layer; allocate ID
docker start <id>               # spawn process via runc
# Output streams via dockerd → CLI; exits → container in `Exited` state
docker rm <id>                  # release writable layer

# Interactive Ubuntu shell
docker run -it --rm ubuntu:22.04 bash
# -i  keep STDIN open
# -t  allocate a pseudo-TTY (so bash thinks it's a real terminal)
# --rm auto-delete container on exit
```

The full lifecycle: **created → running → paused → stopped → removed**.

## Image Operations

### Pull

```bash
docker pull nginx                          # latest tag from docker.io/library/nginx
docker pull nginx:1.25-alpine              # specific tag
docker pull nginx@sha256:abc123...         # immutable digest (recommended for prod)
docker pull --platform linux/arm64 nginx   # specific arch in multi-arch manifest
docker pull --all-tags myrepo/myapp        # every tag (rarely useful)
docker pull --quiet nginx:alpine           # suppress progress bars
```

### List / Filter

```bash
docker images                              # repo:tag table
docker image ls                            # same thing, modern verb form
docker images -a                           # include intermediate (untagged) layers
docker images -q                           # IDs only — feed to `docker rmi`
docker images --no-trunc                   # full sha256 IDs
docker images --digests                    # show pinned digests
docker images --filter "dangling=true"     # untagged orphans
docker images --filter "reference=nginx*"
docker images --filter "before=myapp:v2"
docker images --filter "since=alpine:3.18"
docker images --filter "label=stage=builder"
docker images --format '{{.Repository}}:{{.Tag}} {{.Size}}'
```

### Remove

```bash
docker rmi nginx:alpine                    # remove image
docker image rm nginx:alpine               # same thing
docker rmi -f nginx:alpine                 # force even if container references it
docker image prune                         # dangling (untagged) only
docker image prune -a                      # ALL unused images
docker image prune -a --filter "until=24h" # older than 24 hours
docker rmi $(docker images -q)             # nuke everything (dangerous)
```

### Tag / Push

```bash
docker tag myapp:latest registry.example.com/team/myapp:v1.2.3
docker push registry.example.com/team/myapp:v1.2.3
docker push --all-tags registry.example.com/team/myapp
```

### Save / Load (offline transfer)

```bash
docker save -o myapp.tar myapp:latest
docker save myapp:latest | gzip > myapp.tar.gz
scp myapp.tar.gz airgap-host:/tmp/
ssh airgap-host 'gunzip < /tmp/myapp.tar.gz | docker load'
docker load -i myapp.tar
# Note: `save` exports an image (layers + manifest); `export` flattens a
# CONTAINER's filesystem into a tar with no layers/history.
```

### History / Inspect

```bash
docker history myapp:latest                # layer-by-layer, sizes, command
docker history --no-trunc --human=false myapp:latest
docker inspect myapp:latest                # full JSON manifest + config
docker inspect -f '{{.Architecture}}' myapp:latest
docker inspect -f '{{range .RootFS.Layers}}{{.}}{{"\n"}}{{end}}' myapp:latest
docker manifest inspect nginx:alpine       # multi-arch manifest list
```

## Container Operations

### docker run — every common flag

```bash
docker run [OPTIONS] IMAGE [COMMAND] [ARGS...]
```

```bash
# Lifecycle / I/O
-d, --detach                       # run in background, return container ID
-i, --interactive                  # keep STDIN open
-t, --tty                          # allocate pseudo-TTY
--rm                               # auto-remove container on exit (no `docker rm` needed)
--name web                         # human name (else random "happy_curie")
--pull always|missing|never        # control re-pull behavior

# Networking
-p 8080:80                         # publish HOST:CONTAINER (TCP)
-p 8080:80/udp                     # UDP
-p 127.0.0.1:8080:80               # bind to specific host IP
-P                                 # auto-publish all EXPOSE'd ports to random host ports
--network bridge|host|none|mynet
--network-alias api                # extra DNS name within the user-defined network
--ip 172.20.0.10                   # static IP (custom networks only)
--dns 1.1.1.1
--add-host db:192.168.1.50         # /etc/hosts entry
--hostname web.local

# Filesystem
-v src:dst[:ro]                    # bind mount or named volume
--mount type=bind,src=/data,dst=/data,readonly
--mount type=volume,src=pgdata,dst=/var/lib/postgresql/data
--mount type=tmpfs,dst=/tmp,tmpfs-size=64m
-w, --workdir /app
--read-only                        # rootfs read-only (combine with --tmpfs for /tmp)
--tmpfs /tmp:size=64m,mode=1777

# Environment
-e KEY=value                       # single env var
-e KEY                             # pass through host's $KEY
--env-file .env                    # KEY=VAL lines (no quotes, no `export`)
-u 1000:1000                       # UID:GID; or `-u myuser`

# Resource limits
--memory 512m                      # OOMKill at this RSS
--memory-swap 1g                   # combined RAM+swap; -1 = unlimited swap
--memory-reservation 256m          # soft limit; reclaimed under pressure
--memory-swappiness 0              # 0 disables swap
--cpus 1.5                         # 150% of one core
--cpu-shares 512                   # relative weight (default 1024)
--cpuset-cpus 0,1                  # pin to specific cores
--pids-limit 100                   # max PIDs in container (fork-bomb defense)

# Security
--cap-drop ALL --cap-add NET_BIND_SERVICE
--security-opt no-new-privileges:true
--security-opt seccomp=/etc/docker/profile.json
--security-opt seccomp=unconfined  # disable seccomp (dangerous; debugging)
--security-opt apparmor=docker-default
--userns=host                      # opt out of userns-remap
--privileged                       # grant ALL caps + /dev access (avoid in prod)

# Restart policy
--restart no                       # default
--restart on-failure[:5]           # only restart on non-zero exit, max 5 times
--restart always                   # restart even on docker daemon restart
--restart unless-stopped           # like always but respects manual stop

# Health
--health-cmd "curl -f localhost/health || exit 1"
--health-interval 30s
--health-timeout 5s
--health-retries 3
--health-start-period 60s

# Misc
--label com.example.team=platform
--label-file labels.txt
--init                             # PID 1 = tini reaper (zombie cleanup)
--stop-signal SIGTERM
--stop-timeout 30                  # wait this many seconds before SIGKILL
--log-driver json-file --log-opt max-size=10m --log-opt max-file=3
```

### docker run vs docker create vs docker start

```bash
# docker run = pull + create + start + (optionally) attach
docker run -d --name web nginx

# docker create = pull + create writable layer; do NOT start
docker create --name web nginx          # state: Created
docker start web                        # transitions to Running

# Use create when you want to pre-stage configuration (mounts, env) but
# defer start, e.g. in a systemd unit or for ordered startup in a script.
```

## Container Inspection

### List / filter

```bash
docker ps                                  # running only
docker ps -a                               # all states
docker ps -aq                              # IDs only
docker ps -l                               # most recent
docker ps -n 5                             # last 5
docker ps --no-trunc

docker ps --filter "status=running"        # running|exited|created|paused|restarting|dead
docker ps --filter "name=web"
docker ps --filter "ancestor=nginx:alpine"
docker ps --filter "label=stage=prod"
docker ps --filter "exited=137"            # OOMKilled / SIGKILLed
docker ps --filter "health=unhealthy"
docker ps --filter "publish=8080"
docker ps --filter "network=mynet"
docker ps --filter "volume=pgdata"

docker ps --format '{{.Names}}\t{{.Status}}\t{{.Ports}}'
docker ps --format 'table {{.ID}}\t{{.Names}}\t{{.Status}}'
```

### Logs

```bash
docker logs web                            # all stdout/stderr
docker logs -f web                         # follow (Ctrl-C to exit)
docker logs --tail 100 web                 # last 100 lines
docker logs --tail 0 -f web                # only NEW lines (no backlog)
docker logs --since 5m web                 # last 5 minutes
docker logs --since 2026-04-25T10:00 web   # absolute time
docker logs --until 2026-04-25T11:00 web
docker logs -t web                         # prepend timestamps
docker logs --details web                  # include extra log driver fields
```

### Process / resources

```bash
docker top web                             # ps-style snapshot inside container
docker top web aux                         # any ps options
docker stats                               # live CPU/MEM/NET/IO for all
docker stats --no-stream                   # one-shot snapshot (useful in scripts)
docker stats --format '{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}'
```

### Networking / mounts

```bash
docker port web                            # show all published ports
docker port web 80                         # → 0.0.0.0:8080
docker inspect -f '{{.NetworkSettings.IPAddress}}' web
docker inspect -f '{{json .Mounts}}' web | jq
```

### Exec / attach

```bash
docker exec -it web bash                   # new process inside container
docker exec -it web sh                     # alpine images lack bash
docker exec -u root web cat /etc/shadow    # specific user
docker exec -e DEBUG=1 web printenv DEBUG  # extra env for the exec'd process
docker exec -w /var/log web ls             # cwd for the exec'd process
docker exec -d web touch /tmp/ran          # detached one-shot

docker attach web                          # reattach STDIN/STDOUT/STDERR to PID 1
# DANGER: Ctrl-C inside `attach` sends SIGINT to PID 1 → kills container.
# Use `Ctrl-P Ctrl-Q` to detach without killing.
```

## Lifecycle Management

```bash
docker start web                           # Created/Exited → Running
docker stop web                            # SIGTERM, then SIGKILL after --time (default 10s)
docker stop -t 30 web                      # 30 second grace
docker restart web                         # stop + start
docker kill web                            # SIGKILL immediately
docker kill -s SIGUSR1 web                 # custom signal (reopen logs, reload config)
docker pause web                           # SIGSTOP all processes (cgroup freezer)
docker unpause web                         # SIGCONT
docker rename web api                      # change container name
docker wait web                            # block until exit; print exit code

docker update --memory 1g --memory-swap 2g web
docker update --cpus 2 web
docker update --restart on-failure:5 web
# `update` works on running containers without restart for most flags.
```

## Docker Networks

### List / inspect

```bash
docker network ls
docker network ls --filter driver=bridge
docker network inspect bridge             # Default Docker bridge (the docker0 interface)
docker network inspect mynet              # JSON: subnet, gateway, containers, options
```

### Driver overview

| Driver | Use case | Notes |
|--------|----------|-------|
| `bridge` | default; isolated L2 segment per network | the default `bridge` net has NO automatic DNS; user-defined bridges DO |
| `host` | container shares host's network ns | no port mapping; `lsof -i :PORT` from container = host's view |
| `none` | no networking at all | for fully sandboxed batch jobs |
| `overlay` | multi-host (Swarm) | requires Swarm-mode active |
| `macvlan` | container gets its own MAC on host LAN | no host↔container traffic by default |
| `ipvlan` | like macvlan but L3 mode | no MAC explosion in big clusters |
| `container:ID` | reuse another container's net ns | sidecar pattern |

### Create user-defined bridge

```bash
docker network create mynet
docker network create \
  --driver bridge \
  --subnet 172.30.0.0/16 \
  --ip-range 172.30.5.0/24 \
  --gateway 172.30.0.1 \
  --opt com.docker.network.bridge.name=br-mynet \
  --opt com.docker.network.driver.mtu=1450 \
  mynet

docker run -d --network mynet --name api myapi
docker run -d --network mynet --name db --network-alias database postgres
# `api` resolves both `db` and `database` via Docker's embedded DNS at 127.0.0.11.
```

### Connect / disconnect

```bash
docker network connect mynet existing_container
docker network connect --alias backend mynet api
docker network disconnect mynet api
```

### Default bridge IP range

`/etc/docker/daemon.json`:

```bash
cat <<'JSON' | sudo tee /etc/docker/daemon.json
{
  "bip": "172.17.0.1/16",
  "default-address-pools": [
    {"base": "172.30.0.0/16", "size": 24}
  ],
  "fixed-cidr": "172.17.0.0/24",
  "mtu": 1450
}
JSON
sudo systemctl restart docker
```

### Cleanup

```bash
docker network prune                       # remove unused custom networks
docker network rm mynet                    # specific
```

## Docker Volumes

### Three storage types

| Type | Source | Persists? | macOS perf | Use case |
|------|--------|-----------|-----------|----------|
| named volume | `dockerd`-managed under `/var/lib/docker/volumes/` | yes | fast | DB data, app state |
| bind mount | host path | yes (host owns it) | SLOW on Docker Desktop | dev source-mount, configs |
| tmpfs | RAM | no — gone on stop | fast | secrets, scratch |

### Named volumes

```bash
docker volume create pgdata
docker volume create --driver local \
  --opt type=nfs \
  --opt o=addr=10.0.0.1,rw \
  --opt device=:/exports/data \
  nfsdata
docker volume ls
docker volume ls --filter dangling=true
docker volume inspect pgdata
docker run -d -v pgdata:/var/lib/postgresql/data postgres:16
docker volume rm pgdata
docker volume prune                        # remove all unused volumes
docker volume prune --filter "label!=keep"
```

### Bind mounts

```bash
docker run -d -v "$(pwd)/src:/app:ro" myapp        # short syntax
docker run -d --mount type=bind,src="$(pwd)/src",dst=/app,readonly myapp
docker run -d -v "$(pwd)/src:/app:cached" myapp    # macOS perf hint (host authoritative)
docker run -d -v "$(pwd)/src:/app:delegated" myapp # macOS perf hint (container authoritative)

# macOS gotcha: bind mounts go through gRPC-FUSE / VirtioFS — random-access I/O
# is 5-50x slower than Linux. For DB volumes use named volumes; for source code
# editing use bind mounts but expect slowness on `npm install`-style workloads.
```

### tmpfs

```bash
docker run --tmpfs /tmp:size=128m,mode=1777,noexec myapp
docker run --mount type=tmpfs,dst=/run,tmpfs-size=64m myapp
```

## Dockerfile — Reference

A Dockerfile is a sequence of directives that produce image layers. Each directive (with a few exceptions like `ARG`/`LABEL`/`EXPOSE`) creates a new layer.

### FROM

```bash
FROM ubuntu:22.04                          # base image
FROM ubuntu:22.04 AS base                  # named stage
FROM scratch                               # empty base (for static binaries)
FROM --platform=$TARGETPLATFORM golang:1.22 AS builder
# Multiple FROMs in one Dockerfile = multi-stage build.
```

### ARG

```bash
ARG GO_VERSION=1.22                        # build-time variable
ARG TARGETARCH                             # auto-set by buildx
FROM golang:${GO_VERSION}                  # ARGs BEFORE any FROM are global
ARG APP_VERSION                            # ARGs after FROM are stage-scoped
RUN echo "$APP_VERSION"
# Pass at build time: docker build --build-arg APP_VERSION=1.2.3 .
# WARNING: ARG values are visible in `docker history` — never use for secrets.
```

### ENV

```bash
ENV PATH="/opt/bin:$PATH"
ENV NODE_ENV=production \
    PYTHONUNBUFFERED=1 \
    DEBIAN_FRONTEND=noninteractive
# ENV persists in the final image; ARG does NOT.
```

### LABEL (OCI metadata)

```bash
LABEL org.opencontainers.image.source="https://github.com/me/myapp"
LABEL org.opencontainers.image.version="1.2.3"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.description="My web app"
LABEL com.example.team="platform"
```

### WORKDIR

```bash
WORKDIR /app                               # creates dir if missing
# Affects all subsequent RUN/CMD/ENTRYPOINT/COPY.
```

### COPY

```bash
COPY src/ /app/src/                        # default form
COPY --chown=1000:1000 src/ /app/src/      # set owner (avoids `RUN chown`)
COPY --chmod=755 entrypoint.sh /usr/local/bin/
COPY --from=builder /out/app /usr/local/bin/app   # multi-stage
COPY --from=nginx:alpine /etc/nginx/nginx.conf /etc/nginx/nginx.conf
COPY --link static/ /var/www/html/         # immutable layer (BuildKit, faster cache)
COPY ["file with spaces.txt", "/dest/"]    # JSON form for spaces
```

### ADD (avoid; prefer COPY)

```bash
ADD https://example.com/file.tar.gz /tmp/  # downloads remote URL (no checksum verify)
ADD app.tar.gz /opt/                       # auto-extracts local tar
# Prefer: RUN curl -fsSLO https://example.com/file.tar.gz && sha256sum -c ...
# ADD's auto-extract is surprising and skips checksum verification.
```

### RUN

```bash
RUN apt-get update && apt-get install -y --no-install-recommends \
      curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN ["executable", "param1", "param2"]     # exec form (no shell, no $VAR expansion)

# BuildKit cache mount — package manager cache survives across builds
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y curl

# Build secret — never persisted in image
RUN --mount=type=secret,id=npmrc,target=/root/.npmrc \
    npm ci

# SSH agent forwarding for private git
RUN --mount=type=ssh git clone git@github.com:me/private.git
```

### SHELL

```bash
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]
RUN cmd1 | cmd2 | cmd3                     # now `set -o pipefail` semantics
# Default SHELL is ["/bin/sh", "-c"].
```

### CMD vs ENTRYPOINT

The four interaction modes:

| ENTRYPOINT | CMD | `docker run img` runs | `docker run img foo` runs |
|------------|-----|----------------------|---------------------------|
| absent     | `CMD ["a","b"]` | `a b` | `foo` (overrides CMD) |
| `["e"]`    | absent          | `e`   | `e foo`                  |
| `["e"]`    | `["a"]`         | `e a` | `e foo` (CMD overridden) |
| `e` (shell)| `a` (shell)     | `/bin/sh -c "e"` | `foo` (CMD overridden, ENTRYPOINT shell-form swallows args) |

```bash
# Canonical pattern: ENTRYPOINT = the binary, CMD = default args (overridable)
ENTRYPOINT ["/usr/local/bin/myapp"]
CMD ["--config", "/etc/myapp.yaml"]
# `docker run img --debug` → /usr/local/bin/myapp --debug
```

### EXPOSE

```bash
EXPOSE 8080                                # documentation only — does NOT publish
EXPOSE 8080/tcp 53/udp
# Real publish requires `docker run -p` or compose `ports:`.
```

### VOLUME

```bash
VOLUME /var/lib/postgresql/data            # creates anonymous volume on first run
# Anonymous volumes are easy to lose track of. Prefer naming them in compose.
```

### USER

```bash
RUN groupadd -r app && useradd -r -g app -u 1000 app
USER 1000:1000                             # numeric for portability
# All subsequent RUN/CMD/ENTRYPOINT runs as this user.
```

### HEALTHCHECK

```bash
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD curl -fsS http://localhost:8080/health || exit 1
HEALTHCHECK NONE                           # disable inherited healthcheck from base
# `docker ps` shows (healthy)/(unhealthy)/(starting). Compose can wait on this.
```

### STOPSIGNAL

```bash
STOPSIGNAL SIGINT
# Default SIGTERM. Use SIGINT for nginx, SIGQUIT for some Go apps.
```

### ONBUILD

```bash
ONBUILD COPY . /app
ONBUILD RUN npm ci
# Triggers when this image is used as a FROM by a CHILD image.
# Rare; mostly used for "language base" images.
```

### .dockerignore

```bash
# .dockerignore (same syntax as .gitignore)
.git
node_modules
**/*.log
**/.env
**/.DS_Store
dist/
target/
*.md
!README.md          # negation: include README.md anyway
```

The build context is uploaded to the daemon BEFORE the build starts. A bloated context (e.g. forgetting `.git/` or `node_modules/`) means slow builds AND larger images if you do `COPY . .`.

## Multi-Stage Builds

```bash
# Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM scratch
COPY --from=builder /out/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
USER 1000:1000
ENTRYPOINT ["/app"]
# Final image: ~10MB. Build image: ~800MB but discarded.
```

```bash
# Test stage pattern — fail the build if tests fail
FROM golang:1.22 AS test
WORKDIR /src
COPY . .
RUN go test ./...

FROM golang:1.22 AS builder
WORKDIR /src
COPY --from=test /src .
RUN go build -o /out/app ./cmd/app

FROM scratch
COPY --from=builder /out/app /app
ENTRYPOINT ["/app"]
```

```bash
# Build only a specific stage (e.g. CI runs tests but not deploy):
docker build --target test -t myapp:test .
docker build --target builder -t myapp:build .
```

## BuildKit + buildx

```bash
# BuildKit is default in Docker 23+. Force on/off:
DOCKER_BUILDKIT=1 docker build .
DOCKER_BUILDKIT=0 docker build .           # legacy builder

# buildx is a CLI plugin that drives BuildKit with multi-platform / matrix support
docker buildx version
docker buildx ls
docker buildx create --name multi --driver docker-container --use
docker buildx inspect --bootstrap

# Multi-arch build + push (requires registry)
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  -t registry.example.com/me/myapp:v1 \
  --push .

# Cache to / from a registry
docker buildx build \
  --cache-from type=registry,ref=registry.example.com/me/myapp:cache \
  --cache-to   type=registry,ref=registry.example.com/me/myapp:cache,mode=max \
  -t registry.example.com/me/myapp:v1 --push .

# Cache to / from local dir (CI artifact)
docker buildx build \
  --cache-from type=local,src=/tmp/buildcache \
  --cache-to   type=local,dest=/tmp/buildcache,mode=max .

# Inline cache (cache embedded in pushed image)
docker buildx build --cache-to type=inline -t myapp:v1 --push .

# Build secrets (never in `docker history`)
echo "$NPM_TOKEN" | docker buildx build --secret id=npm,src=- .

# SSH forwarding for private git deps
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
docker buildx build --ssh default .

# Bake — matrix builds defined in HCL/JSON
cat <<'HCL' > docker-bake.hcl
group "default" { targets = ["api", "worker"] }
target "api"    { context = "./api"    tags = ["myapp/api:v1"] }
target "worker" { context = "./worker" tags = ["myapp/worker:v1"] }
HCL
docker buildx bake
```

## Build Context

```bash
docker build -t myapp .                    # current dir = context
docker build -t myapp -                    # context from STDIN (Dockerfile only)
cat Dockerfile | docker build -t myapp -

docker build -t myapp https://github.com/me/repo.git
docker build -t myapp https://github.com/me/repo.git#main
docker build -t myapp https://github.com/me/repo.git#main:subdir
docker build -t myapp https://example.com/myapp.tar.gz

# Remote contexts (no local checkout)
docker build -t myapp git@github.com:me/repo.git#v1.2:subdir

# Specify Dockerfile explicitly
docker build -f Dockerfile.prod -t myapp:prod .
docker build -f - . <<'DOCKER'
FROM alpine
CMD ["echo","hi"]
DOCKER
```

## Docker Compose v2

`compose.yaml` (or `docker-compose.yaml`) defines a multi-container app.

```bash
# compose.yaml
services:
  api:
    build:
      context: ./api
      dockerfile: Dockerfile
      args:
        GO_VERSION: "1.22"
      target: builder        # multi-stage target
    image: myapp/api:dev      # tag for local builds
    container_name: api       # avoid in compose unless single-instance
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "127.0.0.1:9090:9090" # bind to localhost only
    environment:
      DATABASE_URL: postgres://app:pass@db:5432/app
      LOG_LEVEL: ${LOG_LEVEL:-info}        # default if unset
    env_file:
      - .env
      - .env.local            # later files override earlier
    volumes:
      - ./api:/app:cached     # bind mount for dev
      - api_node_modules:/app/node_modules   # named, so host doesn't clobber
    networks:
      - backend
    depends_on:
      db:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 30s
    command: ["./api", "--config", "/etc/api.yaml"]
    entrypoint: ["/sbin/tini", "--"]
    working_dir: /app
    user: "1000:1000"
    labels:
      com.example.team: platform
    profiles: ["dev"]         # only active with --profile dev
    expose:
      - "9090"                # internal-only port (no host publish)
    extra_hosts:
      - "host.docker.internal:host-gateway"
    dns:
      - 1.1.1.1
    sysctls:
      net.core.somaxconn: 1024
    cap_add: ["NET_BIND_SERVICE"]
    cap_drop: ["ALL"]
    security_opt:
      - no-new-privileges:true
      - seccomp:unconfined
    read_only: true
    tmpfs:
      - /tmp:size=64m
    ulimits:
      nofile:
        soft: 65535
        hard: 65535
    init: true
    stop_grace_period: 30s
    stop_signal: SIGTERM
    deploy:
      resources:
        limits: { cpus: "1.5", memory: 512M }
        reservations: { cpus: "0.5", memory: 128M }

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: app
      POSTGRES_USER: app
      POSTGRES_PASSWORD: pass
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks:
      - backend
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d app"]
      interval: 10s
      timeout: 5s
      retries: 5

  migrate:
    image: myapp/migrate:dev
    networks: [backend]
    depends_on:
      db:
        condition: service_healthy
    command: ["./migrate", "up"]

networks:
  backend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.30.0.0/24

volumes:
  pgdata:
  api_node_modules:
```

```bash
# version: '3.8' is DEPRECATED in Compose v2 — omit it. The schema is
# inferred from the `services` shape.
```

### Override files

```bash
# docker-compose.override.yaml is auto-merged on top of compose.yaml.
# Useful for dev-only tweaks (bind mounts, debug ports).
docker compose up -d   # reads compose.yaml + override.yaml automatically

# Explicit file selection for prod
docker compose -f compose.yaml -f compose.prod.yaml up -d
```

## Compose Commands

```bash
docker compose up                          # foreground; Ctrl-C stops all
docker compose up -d                       # detached
docker compose up -d --build               # force rebuild
docker compose up -d --force-recreate      # re-create even if config unchanged
docker compose up -d --no-deps api         # only `api`, skip its depends_on
docker compose up -d --scale worker=3      # 3 replicas of worker

docker compose down                        # stop + remove containers + default network
docker compose down -v                     # also remove named volumes
docker compose down --remove-orphans       # remove containers from removed services
docker compose down --rmi local            # also delete locally-built images

docker compose ps                          # services in this project
docker compose ps -a                       # include stopped
docker compose ps --services               # service names only

docker compose logs                        # all services
docker compose logs -f api                 # follow
docker compose logs --tail 100 --since 5m

docker compose exec api bash               # exec in running container
docker compose run --rm api ./bin/migrate  # one-off ephemeral container
docker compose run --rm --service-ports api  # publish ports for one-offs

docker compose build api                   # rebuild specific service
docker compose build --no-cache --pull
docker compose pull                        # pull all images (no build)

docker compose config                      # validated, fully-rendered compose.yaml
docker compose config --services
docker compose config --quiet              # exit 0 if valid, non-zero otherwise

docker compose top                         # ps inside each container
docker compose pause api
docker compose unpause api
docker compose restart api
docker compose stop                        # all
docker compose start
docker compose kill                        # SIGKILL all
docker compose rm -f                       # remove stopped service containers
docker compose port api 8080               # show 0.0.0.0:HOSTPORT
docker compose images                      # service → image table
docker compose events                      # stream container events
docker compose cp api:/var/log/app.log ./
```

## Compose Profiles

```bash
services:
  api: { ... }                    # always active
  db:  { ... }                    # always active
  pgadmin:
    image: dpage/pgadmin4
    profiles: ["debug"]           # only when --profile debug
  fakedata:
    image: mydata-seeder
    profiles: ["dev", "ci"]       # active when EITHER profile selected
```

```bash
docker compose up -d                       # only api + db
docker compose --profile debug up -d       # api + db + pgadmin
docker compose --profile dev --profile ci up -d
COMPOSE_PROFILES=dev,ci docker compose up -d
```

## Resource Limits

```bash
# Memory
--memory 512m                              # hard limit — OOMKill at 100%
--memory-swap 1g                           # combined RAM+swap; -1 disables swap cap
--memory-reservation 256m                  # soft floor; reclaimed under pressure
--memory-swappiness 0                      # 0 = avoid swap (use anon mem only)
--oom-score-adj -100                       # bias OOM killer to skip this container
--oom-kill-disable                         # NEVER do this in prod (machine wedge)

# CPU
--cpus 1.5                                 # = 150% of one core (period * quota)
--cpu-shares 1024                          # relative weight (default 1024)
--cpu-period 100000                        # microseconds (default 100ms)
--cpu-quota 50000                          # 50ms / 100ms = 0.5 cpus
--cpuset-cpus 0,2-3                        # pin to physical cores
--cpu-rt-runtime 950000                    # realtime quota
--cpus 1.5 --cpu-shares 512                # combined: capped at 1.5 cores, 0.5 weight

# Process limits
--pids-limit 100                           # cap forks (defends against fork-bomb)
--ulimit nofile=1024:2048                  # soft:hard
--ulimit nproc=4096
```

When `--memory` is hit, the kernel OOM-kills the process; container exits 137 (= 128 + 9 SIGKILL). Logs typically say:

```bash
# in /var/log/syslog or `dmesg`
Out of memory: Killed process 12345 (myapp) total-vm:1234567kB ...
# Container in `docker ps -a`: STATUS = Exited (137)
```

## Security — Containers

```bash
# Drop all caps, add only what you need
docker run --cap-drop ALL --cap-add NET_BIND_SERVICE myapp

# Common caps to know:
# - NET_BIND_SERVICE: bind to ports < 1024 as non-root
# - SYS_ADMIN: dangerous; many subsystems (mount, namespaces)
# - SYS_PTRACE: needed for `strace` inside container
# - NET_ADMIN: configure interfaces, iptables (needed for VPN containers)
# - CHOWN/FOWNER/DAC_OVERRIDE: file ops as non-owner

docker run --security-opt no-new-privileges:true myapp
# Process can never gain privs via setuid binaries — defense against privesc.

docker run --security-opt seccomp=/etc/docker/seccomp.json myapp
docker run --security-opt seccomp=unconfined myapp        # disable (debugging)
docker run --security-opt apparmor=docker-default myapp
docker run --security-opt label=type:my_container_t myapp # SELinux

docker run --read-only --tmpfs /tmp --tmpfs /var/run myapp
# Root filesystem is read-only; only writeable areas are tmpfs mounts.

docker run -u 1000:1000 myapp              # explicit non-root UID:GID

docker run --pids-limit 100 myapp          # fork-bomb defense

docker run --userns=host myapp             # opt OUT of userns-remap
# In daemon.json, "userns-remap": "default" maps container UID 0 → host UID 100000
# (or whatever subuid range), so container root ≠ host root.
```

## Security — Image Build

```bash
FROM alpine:3.19@sha256:c5b1261...        # PIN by digest, not by tag

# Create a non-root user
RUN addgroup -S app && adduser -S -G app -u 1000 app

# Multi-stage to drop build tools
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM alpine:3.19@sha256:c5b1261...
RUN addgroup -S app && adduser -S -G app -u 1000 app \
    && apk add --no-cache ca-certificates tini
COPY --from=builder /out/app /usr/local/bin/app
USER 1000:1000
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/app"]
```

```bash
# NEVER do this:
ARG SECRET=hunter2                         # visible in docker history
ENV API_KEY=hunter2                        # bakes into image, anyone with image can read

# Do this instead (BuildKit secret mount):
RUN --mount=type=secret,id=api_key,target=/run/secrets/key \
    curl -H "Authorization: Bearer $(cat /run/secrets/key)" ...
# Build with:
echo "$API_KEY" | docker buildx build --secret id=api_key,src=- .
```

`.dockerignore` MUST exclude:

```bash
.git
**/.env*
**/*.pem
**/id_rsa*
**/*.key
**/secrets/**
```

## Image Inspection and Reverse-Engineering

```bash
docker inspect myapp:latest                # full manifest + config
docker history --no-trunc myapp:latest     # layer commands
docker save myapp:latest -o myapp.tar
tar -tf myapp.tar | head                   # OCI image layout:
#  manifest.json
#  <sha256>.json   (image config)
#  <sha256>/layer.tar  per layer

# `dive` — interactive layer browser (third-party but invaluable)
brew install dive
dive myapp:latest

# Run interactively to poke around
docker run -it --rm --entrypoint sh myapp:latest

# Extract a single file from an image without running it
container=$(docker create myapp:latest)
docker cp "$container":/etc/myapp.yaml ./
docker rm "$container"
```

OCI image format (what `save` produces, what registries store):

```bash
# Image manifest (per-arch)
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": { "digest": "sha256:abc...", "size": 1234 },
  "layers": [
    { "digest": "sha256:def...", "size": 5678, "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip" }
  ]
}

# Image index (multi-arch list)
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    { "digest": "sha256:...", "platform": { "os": "linux", "architecture": "amd64" } },
    { "digest": "sha256:...", "platform": { "os": "linux", "architecture": "arm64" } }
  ]
}
```

## Registries

```bash
# Login (writes to ~/.docker/config.json under "auths")
docker login                                          # default: docker.io
docker login registry.example.com
docker login -u myuser registry.example.com           # password via stdin
echo "$TOKEN" | docker login -u myuser --password-stdin registry.example.com
docker logout registry.example.com

# Default registry: registry-1.docker.io.
# `nginx` → `docker.io/library/nginx`
# `myuser/myapp` → `docker.io/myuser/myapp`
# `ghcr.io/me/myapp` is fully qualified.

# Run a local registry for testing
docker run -d --name reg -p 5000:5000 registry:2
docker tag myapp:v1 localhost:5000/myapp:v1
docker push localhost:5000/myapp:v1

# Pull-through cache (mirror to docker.io)
# /etc/docker/daemon.json:
#   { "registry-mirrors": ["https://mirror.example.com"] }
sudo systemctl restart docker
```

## Tags vs Digests

```bash
# Tags are MUTABLE — the same `nginx:1.25` can point to different bytes tomorrow
docker pull nginx:1.25
docker pull nginx:1.25@sha256:c5b1261d6d3e43071626f5ff1e91e22c... 

# Production: pull by digest
FROM nginx:1.25@sha256:c5b1261d6d3e43071626f5ff1e91e22c...
# This is reproducible; supply-chain attack-resistant; survives tag retag.

# Find current digest
docker buildx imagetools inspect nginx:1.25
docker manifest inspect nginx:1.25 | jq -r '.manifests[].digest'
docker inspect --format='{{index .RepoDigests 0}}' nginx:1.25
```

## Multi-Architecture Builds

```bash
# Set up QEMU emulation (one-time per host)
docker run --privileged --rm tonistiigi/binfmt --install all

# Verify supported platforms
docker buildx ls
# Look for: linux/amd64*, linux/arm64*, linux/arm/v7, linux/arm/v6, linux/386, linux/ppc64le, linux/s390x

# Build for multiple platforms (registry required for multi-arch push)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t registry.example.com/me/myapp:v1 \
  --push .

# Use TARGETPLATFORM in Dockerfile
FROM --platform=$BUILDPLATFORM golang:1.22 AS builder
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o /out/app ./cmd/app
FROM scratch
COPY --from=builder /out/app /app
ENTRYPOINT ["/app"]

# Test specific platform
docker run --rm --platform linux/arm64 myapp:v1 uname -m   # → aarch64
```

## SBOM, Signing, Scanning

```bash
# Generate SBOM (Software Bill of Materials)
docker sbom myapp:v1                       # built into Docker Desktop
docker buildx build --sbom=true -t myapp:v1 --push .
syft myapp:v1 -o spdx-json > sbom.json     # standalone tool

# Scan for vulnerabilities
docker scout cves myapp:v1                 # Docker's built-in
docker scout recommendations myapp:v1
trivy image myapp:v1                       # standalone, popular alternative
grype myapp:v1                             # another popular scanner

# Sign images with cosign
cosign generate-key-pair                   # produces cosign.key + cosign.pub
cosign sign --key cosign.key registry.example.com/me/myapp:v1
cosign verify --key cosign.pub registry.example.com/me/myapp:v1

# Supply-chain hardening checklist:
# 1. Pin base image by digest
# 2. Multi-stage to drop build tools
# 3. Run as non-root
# 4. .dockerignore for secrets
# 5. Scan in CI (trivy/grype/scout)
# 6. Generate SBOM
# 7. Sign on push (cosign)
# 8. Verify signature on pull (cosign verify, sigstore policy-controller)
```

## Rootless Docker

```bash
# Install (Ubuntu 22.04+ already has docker-ce-rootless-extras)
dockerd-rootless-setuptool.sh install

export DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock
systemctl --user enable --now docker

# Limitations:
# - No bind to ports < 1024 (use setcap, or use 8080 instead of 80)
# - cgroup v2 required for resource limits
# - Some kernels reject overlay2 in user namespace; falls back to vfs (slow)
# - No AppArmor / SELinux confinement
# - --net=host doesn't share host's net ns (uses slirp4netns instead)
# - Devices in /dev limited

# Alternative: podman is built rootless-first
podman run -d --name web -p 8080:80 nginx
# Same CLI, no daemon, native rootless, works on macOS via QEMU VM (`podman machine`).
```

## Docker Desktop Specifics

```bash
# Docker Desktop runs `dockerd` inside a hidden Linux VM:
#   macOS: HyperKit (older) or Apple Virtualization.framework (newer)
#   Windows: WSL2 (preferred) or Hyper-V
#   Linux: KVM via QEMU (Docker Desktop for Linux)

# Resource limits live in the GUI (Settings → Resources), NOT in `docker run`:
#   - CPU cores allocated to the VM
#   - RAM allocated to the VM
#   - Swap, disk image size

# File sharing is gRPC-FUSE / VirtioFS:
#   - macOS bind mounts are SLOW for random IO (5-50x vs Linux native)
#   - Settings → Resources → File sharing must list the directory
#   - Symlinks across the boundary are hairy

# Networking:
#   - host.docker.internal resolves to the host (special hostname Docker injects)
#   - The VM bridges through the host's interface; published ports work as on Linux
#   - You CANNOT use --network=host meaningfully on macOS/Windows; the "host" is the VM

# Useful Desktop-only env:
DOCKER_HOST=unix:///Users/$USER/.docker/run/docker.sock     # macOS
docker context ls                          # contexts: default, desktop-linux, ...
docker context use desktop-linux
```

## Cleanup and Maintenance

```bash
docker system df                           # disk usage by category
docker system df -v                        # per-image / per-container detail

docker system prune                        # stopped containers + dangling images + unused networks + build cache
docker system prune -a                     # ALSO unused tagged images
docker system prune -a --volumes           # ALSO unused volumes — NUCLEAR
docker system prune -a --volumes --filter "until=24h"
docker system prune --force                # skip confirmation prompt

docker builder prune                       # build cache only
docker builder prune --all                 # all build cache
docker builder prune --keep-storage 10GB   # keep last 10GB of cache
docker buildx prune --filter "until=168h"  # older than 7 days

docker image prune                         # dangling
docker image prune -a                      # all unused
docker image prune -a --filter "until=72h"
docker image prune -a --filter "label=stage=builder"

docker container prune                     # stopped containers
docker container prune --filter "until=24h"

docker network prune
docker volume prune
docker volume prune --filter "label!=keep"

docker system events                       # live event stream
docker system info | grep -E 'Storage|Cgroup|Runtime|Logging'
```

## Logging Drivers

```bash
# Per-container
docker run --log-driver json-file \
           --log-opt max-size=10m \
           --log-opt max-file=3 \
           --log-opt labels=service \
           --log-opt env=ENV \
           myapp

# Daemon-wide default in /etc/docker/daemon.json:
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3",
    "compress": "true"
  }
}
```

| Driver | Use case |
|--------|----------|
| `json-file` | default; readable by `docker logs`; ROTATE THIS or disk fills |
| `local` | json-file++; smaller, faster, compressed, also `docker logs`-readable |
| `journald` | systemd integration; `journalctl CONTAINER_ID=...` |
| `syslog` | classic syslog forward (UDP/TCP/TLS) |
| `fluentd` | structured to Fluentd/Fluent Bit (great for k8s parity) |
| `gelf` | Graylog / Logstash GELF format (UDP) |
| `awslogs` | direct to CloudWatch Logs |
| `gcplogs` | direct to Stackdriver |
| `splunk` | direct to HEC |
| `none` | discard (`docker logs` returns empty) |

## Storage Drivers

```bash
docker info | grep -E 'Storage Driver|Backing Filesystem|Cgroup'
# Storage Driver: overlay2          ← default, best
# Backing Filesystem: extfs|xfs     (xfs ftype=1 required for overlay2 on RHEL)
```

| Driver | Notes |
|--------|-------|
| `overlay2` | default, recommended; needs kernel ≥ 4.0 (recommended 5.x) |
| `btrfs` | only if `/var/lib/docker` is on btrfs |
| `zfs` | only if `/var/lib/docker` is on a zpool |
| `vfs` | NO copy-on-write; slow but always works (rootless fallback) |
| `devicemapper` | DEPRECATED; old RHEL/CentOS 7 default |
| `aufs` | DEPRECATED; old Ubuntu default before 18.04 |

## Common Errors and Fixes

```bash
# 1) Daemon not running
$ docker ps
Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
# Fix:
sudo systemctl start docker            # Linux
# macOS/Windows: start Docker Desktop
ls -l /var/run/docker.sock              # confirm socket exists
```

```bash
# 2) Permission denied on socket
$ docker ps
permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock: Get "http://%2Fvar%2Frun%2Fdocker.sock/v1.45/containers/json": dial unix /var/run/docker.sock: connect: permission denied
# Fix:
sudo usermod -aG docker "$USER"
newgrp docker                            # or log out + back in
```

```bash
# 3) Pull access denied
$ docker pull mycorp/private:v1
Error response from daemon: pull access denied for mycorp/private, repository does not exist or may require 'docker login': denied: requested access to the resource is denied
# Fix:
docker login registry.example.com
# Or check the spelling — wrong org/name returns the same error.
```

```bash
# 4) Manifest not found
$ docker pull nginx:99.99
Error response from daemon: manifest for nginx:99.99 not found: manifest unknown: manifest unknown
# Fix: tag/digest doesn't exist. List actual tags:
curl -s https://hub.docker.com/v2/repositories/library/nginx/tags | jq -r '.results[].name'
```

```bash
# 5) Binary not found in container
$ docker run myapp
OCI runtime create failed: container_linux.go:380: starting container process caused: exec: "myapp": executable file not found in $PATH: unknown
# Fix: ENTRYPOINT/CMD references missing binary. Check the path:
docker run --rm --entrypoint sh myapp -c 'ls -la /usr/local/bin'
docker history myapp | grep ENTRYPOINT
```

```bash
# 6) Mount not shared (Docker Desktop)
$ docker run -v /Users/me/code:/app myapp
docker: Error response from daemon: Mounts denied: 
The path /Users/me/code is not shared from the host and is not known to Docker.
# Fix: Docker Desktop → Settings → Resources → File sharing → Add /Users/me/code
```

```bash
# 7) Disk full
$ docker pull nginx
no space left on device
# Fix:
docker system df                        # see usage
docker system prune -a --volumes        # nuclear cleanup
df -h /var/lib/docker                   # confirm reclaim
```

```bash
# 8) Port collision
$ docker run -p 80:80 nginx
docker: Error response from daemon: driver failed programming external connectivity on endpoint web (...): Bind for 0.0.0.0:80 failed: port is already allocated.
# Fix: find the process holding the port
sudo lsof -i :80
sudo ss -tlnp '( sport = :80 )'
# Stop it, or use a different host port: -p 8080:80
```

```bash
# 9) Container marked for removal but alive
$ docker rm web
Error response from daemon: removal of container web is already in progress
# Fix: daemon-state inconsistency
sudo systemctl restart docker
docker rm -f web
```

```bash
# 10) Swap limit warning
$ docker info
WARNING: Your kernel does not support swap limit capabilities. Limitation discarded.
# Fix (Ubuntu): edit /etc/default/grub:
GRUB_CMDLINE_LINUX="cgroup_enable=memory swapaccount=1"
sudo update-grub && sudo reboot
# Often safe to ignore on modern kernels with cgroup v2.
```

```bash
# 11) iptables / NAT broken
$ docker run -p 8080:80 nginx
docker: Error response from daemon: driver failed programming external connectivity on endpoint web: Error starting userland proxy: listen tcp4 0.0.0.0:8080: bind: address already in use.
# Or:
docker: Error response from daemon: failed to create endpoint web on network bridge: iptables failed: iptables --wait -t nat -A DOCKER -p tcp -d 0.0.0.0/0 ...
# Fix:
sudo systemctl restart docker
sudo iptables -t nat -F DOCKER && sudo systemctl restart docker
```

```bash
# 12) containerd nil pointer panic
$ docker ps
panic: runtime error: invalid memory address or nil pointer dereference
# Fix: daemon bug; restart
sudo systemctl restart docker containerd
```

```bash
# 13) Pull timeout
$ docker pull bigimage:v1
Error response from daemon: Get "https://registry-1.docker.io/v2/": context deadline exceeded
# Fix: slow registry / network. Increase timeout, or use a mirror:
# /etc/docker/daemon.json:
#   { "registry-mirrors": ["https://mirror.gcr.io"] }
sudo systemctl restart docker
```

```bash
# 14) Kernel too old for overlay2
$ docker info
WARNING: the overlay storage-driver is deprecated, and will be removed in a future release.
# or:
ERROR: failed to register layer: error creating overlay mount: invalid argument
# Fix: kernel ≥ 4.0 (5.x recommended); on RHEL 7 use overlay2 with xfs ftype=1.
```

```bash
# 15) Compose v1 vs v2 confusion
$ docker-compose up
docker-compose: command not found
# Fix: use the v2 plugin
docker compose up                       # space, not hyphen
# Install: apt-get install docker-compose-plugin
```

## Container Exit Codes

```bash
docker inspect --format='{{.State.ExitCode}}' web
docker ps -a --filter "exited=137"

# 0    success
# 1    generic error (your app's `os.Exit(1)`)
# 2    misuse of shell builtin (bash convention)
# 125  `docker run` itself failed (bad flag, daemon error)
# 126  command found but not executable (chmod issue)
# 127  command not found (typo or missing binary)
# 128+N  killed by signal N
#   130 = 128 + 2  SIGINT  (Ctrl-C)
#   137 = 128 + 9  SIGKILL (OOMKilled, kubelet eviction, --memory hit)
#   139 = 128 + 11 SIGSEGV (segfault)
#   143 = 128 + 15 SIGTERM (graceful shutdown — what `docker stop` sends first)
```

## Common Gotchas

### Shell-form ENTRYPOINT swallows signals

```bash
# Broken — SIGTERM goes to /bin/sh, not your app; your app never gets a chance to clean up
ENTRYPOINT ./app

# Fixed — exec form, your binary is PID 1, receives SIGTERM directly
ENTRYPOINT ["./app"]
# Or use tini for proper PID 1 (signal forwarding + zombie reaping)
ENTRYPOINT ["/sbin/tini", "--", "./app"]
```

### Wrong-arch silent build on Apple Silicon

```bash
# Broken — on M1/M2 macs this builds linux/arm64, then fails on amd64 server
docker build -t myapp .

# Fixed
docker buildx build --platform linux/amd64 -t myapp --load .
```

### Cache-busting from timestamp churn

```bash
# Broken — every git checkout updates timestamps, busting layer cache
COPY . .
RUN go build ./...

# Fixed — copy go.mod/go.sum first (rarely changes), then download deps,
# then copy source. Source changes don't bust the dep-download layer.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build ./...

# Also: --link makes the COPY layer immutable, cache survives unrelated changes
COPY --link . .
```

### Secret in ARG / ENV

```bash
# Broken — visible in `docker history`, leaks to anyone with image access
ARG NPM_TOKEN
RUN npm config set //registry.npmjs.org/:_authToken="$NPM_TOKEN" && npm ci

# Fixed — BuildKit secret mount, never persisted
RUN --mount=type=secret,id=npm,target=/root/.npmrc npm ci
# Build with:
echo "//registry.npmjs.org/:_authToken=$NPM_TOKEN" \
  | docker buildx build --secret id=npm,src=- .
```

### Layer-bloat from many small RUNs

```bash
# Broken — three layers, three times the apt overhead
RUN apt-get update
RUN apt-get install -y curl
RUN apt-get install -y vim
RUN rm -rf /var/lib/apt/lists/*

# Fixed — one layer, one apt cache update, cleanup in same layer
RUN apt-get update && apt-get install -y --no-install-recommends \
      curl vim \
    && rm -rf /var/lib/apt/lists/*
```

### apt without --no-install-recommends

```bash
# Broken — installs recommended packages too (bloats image 100s of MB)
RUN apt-get install -y python3

# Fixed
RUN apt-get install -y --no-install-recommends python3 \
    && rm -rf /var/lib/apt/lists/*
```

### Forgetting apt cleanup

```bash
# Broken — /var/lib/apt/lists/ stays in the image (~50MB)
RUN apt-get update && apt-get install -y curl

# Fixed
RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*
```

### latest tag in production

```bash
# Broken — image silently changes between deploys; supply-chain attack vector
FROM nginx:latest

# Fixed — pin to digest
FROM nginx:1.25.3@sha256:c5b1261d6d3e43071626f5ff1e91e22c...
```

### VOLUME directive in Dockerfile

```bash
# Broken — anonymous volume created on every run; orphaned named volumes accumulate
VOLUME /data

# Fixed — declare in compose where you control the name
# (Dockerfile)
WORKDIR /data
# (compose.yaml)
services:
  app:
    volumes:
      - appdata:/data
volumes:
  appdata:
```

### Running as root

```bash
# Broken — container compromise → host UID 0
FROM alpine
COPY app /app
ENTRYPOINT ["/app"]

# Fixed
FROM alpine
RUN addgroup -S app && adduser -S -G app -u 1000 app
COPY --chown=1000:1000 app /app
USER 1000:1000
ENTRYPOINT ["/app"]
```

### Build context leak

```bash
# Broken — uploads .git/, node_modules/, .env to daemon every build (slow + secret leak)
$ ls -la
.git/  .env  node_modules/  src/  Dockerfile

# Fixed — .dockerignore
cat <<'EOF' > .dockerignore
.git
.env*
node_modules
**/*.log
EOF
```

### Docker Compose v1 syntax

```bash
# Broken — `version: '3.8'` triggers warnings and is ignored
version: '3.8'
services:
  api:
    image: myapp

# Fixed — drop the version key entirely
services:
  api:
    image: myapp
```

### Bind mount permissions on Linux

```bash
# Broken — host UID 1000 doesn't match container UID; files written by container
# are owned by root on the host (or vice-versa)
docker run -v "$(pwd)/data:/data" myapp

# Fixed — match UIDs explicitly
docker run -u "$(id -u):$(id -g)" -v "$(pwd)/data:/data" myapp
```

### depends_on without healthcheck

```bash
# Broken — `api` starts the moment `db` *container* starts, before postgres is ready
services:
  api:
    depends_on: [db]
  db:
    image: postgres

# Fixed — wait for actual readiness
services:
  api:
    depends_on:
      db:
        condition: service_healthy
  db:
    image: postgres
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 5
```

## Performance Tips

- **Multi-stage builds**: final image carries only runtime artifacts. Go binary in `scratch` = ~10MB.
- **`--link` on COPY**: layer is immutable, cache invalidation independent of preceding layers.
- **`--mount=type=cache`**: persist package-manager state across builds. Massive speedup for `apt`, `npm`, `pip`, `go mod`, `cargo`.
- **Order layers by churn**: rarely-changing first (deps), often-changing last (source). Cache hits stay high.
- **Pin base image by digest**: prevents surprise rebuilds when upstream tag is repointed.
- **Distroless / scratch / alpine**: smaller surface area, smaller registry transfers, faster pulls.
- **BuildKit parallelism**: independent stages build concurrently. Group unrelated work into separate stages.
- **`--cache-from registry`**: CI builds reuse layers across runners.
- **Docker Desktop on macOS**: prefer named volumes for hot data; bind mounts for source code only. Use `:cached` or `:delegated` consistency hints if available.
- **`docker system df` regularly**: catch the disk fill before it bites.

## Idioms

### Static Go binary in scratch

```bash
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/app /app
USER 1000:1000
ENTRYPOINT ["/app"]
```

### Python slim with pip wheel cache

```bash
FROM python:3.12-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --user -r requirements.txt

FROM python:3.12-slim
RUN useradd -m -u 1000 app
COPY --from=builder --chown=1000:1000 /root/.local /home/app/.local
COPY --chown=1000:1000 . /app
USER 1000
WORKDIR /app
ENV PATH="/home/app/.local/bin:$PATH"
ENTRYPOINT ["python", "-m", "myapp"]
```

### Node.js with `npm ci`

```bash
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --omit=dev

FROM node:20-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN --mount=type=cache,target=/root/.npm npm run build

FROM node:20-alpine
RUN addgroup -S app && adduser -S -G app -u 1000 app
WORKDIR /app
COPY --from=deps  --chown=1000:1000 /app/node_modules ./node_modules
COPY --from=build --chown=1000:1000 /app/dist ./dist
COPY --chown=1000:1000 package.json .
USER 1000
ENV NODE_ENV=production
ENTRYPOINT ["node", "dist/index.js"]
```

### Compose dev-vs-prod with override + profiles

```bash
# compose.yaml — minimal shared base
services:
  api:
    build: ./api
    image: myapp/api
    environment:
      LOG_LEVEL: ${LOG_LEVEL:-info}
    networks: [backend]
  db:
    image: postgres:16
    networks: [backend]
networks: { backend: {} }

# docker-compose.override.yaml — dev (auto-merged)
services:
  api:
    volumes:
      - ./api:/app
    ports: ["8080:8080"]
    environment:
      LOG_LEVEL: debug
  db:
    ports: ["5432:5432"]
    environment:
      POSTGRES_PASSWORD: dev

# compose.prod.yaml — explicit
services:
  api:
    image: registry.example.com/me/api:${VERSION}
    deploy:
      resources:
        limits: { cpus: "1.5", memory: 512M }
    restart: unless-stopped
  db:
    image: registry.example.com/me/db:${DB_VERSION}
    restart: unless-stopped

# Run:
docker compose up -d                                  # dev
docker compose -f compose.yaml -f compose.prod.yaml up -d  # prod
```

### Healthcheck-based dependency ordering

```bash
services:
  app:
    depends_on:
      db:        { condition: service_healthy }
      cache:     { condition: service_started }
      migrate:   { condition: service_completed_successfully }
  migrate:
    image: myapp/migrate
    command: ["./migrate", "up"]
    depends_on:
      db: { condition: service_healthy }
  db:
    image: postgres:16
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      retries: 10
  cache:
    image: redis:7
```

## Tips

- `docker run --rm` for one-offs to avoid stopped-container clutter.
- `docker logs -f --tail 0` to watch only NEW lines (no scrollback).
- `docker compose config` validates and prints the fully-rendered compose file — great in CI.
- `docker exec` always opens a NEW process; it does NOT share env with PID 1.
- `Ctrl-P Ctrl-Q` detaches from `docker attach` without killing PID 1.
- `docker inspect | jq` for any complex query — avoid `--format` for one-offs.
- `docker system df -v` reveals which images/volumes/builders are eating disk.
- Pin base images by digest in production. `nginx:latest` will betray you eventually.
- `HEALTHCHECK` lets `docker ps` show real readiness, not just "process running".
- Always set `--restart unless-stopped` for long-lived services on a dev host.
- For local registries, `--insecure-registry localhost:5000` in `daemon.json`.
- `docker buildx imagetools inspect IMG` shows the manifest list without pulling.
- `docker context` lets you target remote dockerd over SSH: `docker context create remote --docker host=ssh://user@host`.
- On macOS, prefer named volumes for hot data — bind mounts are FUSE-slow.
- `tini` (`--init` flag) for proper PID 1 signal handling and zombie reaping.
- `docker events` streams real-time activity — invaluable for debugging "what just happened?"
- `docker compose run --rm --service-ports api ./script.sh` runs a one-off with the same network/ports as the service.

## See Also

- kubernetes
- kubectl
- helm
- podman
- bash
- make

## References

- [Docker Documentation](https://docs.docker.com/)
- [Docker CLI Reference](https://docs.docker.com/reference/cli/docker/)
- [Dockerfile Reference](https://docs.docker.com/reference/dockerfile/)
- [Docker Compose Specification](https://github.com/compose-spec/compose-spec)
- [Compose CLI Reference](https://docs.docker.com/reference/cli/docker/compose/)
- [BuildKit Documentation](https://docs.docker.com/build/buildkit/)
- [docker buildx Reference](https://docs.docker.com/reference/cli/docker/buildx/)
- [Docker Engine API Reference](https://docs.docker.com/engine/api/)
- [Docker Networking Overview](https://docs.docker.com/engine/network/)
- [Docker Storage — Volumes](https://docs.docker.com/engine/storage/volumes/)
- [Docker Hub](https://hub.docker.com/)
- [Docker Build — Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [Docker Security Best Practices](https://docs.docker.com/build/building/best-practices/)
- [Rootless Docker](https://docs.docker.com/engine/security/rootless/)
- [Docker Desktop](https://docs.docker.com/desktop/)
- [OCI Image Specification](https://github.com/opencontainers/image-spec)
- [OCI Runtime Specification](https://github.com/opencontainers/runtime-spec)
- [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec)
- [containerd](https://containerd.io/)
- [runc](https://github.com/opencontainers/runc)
- [Moby Project (Docker Engine Source)](https://github.com/moby/moby)
- [Podman](https://podman.io/)
- [Sigstore / cosign](https://www.sigstore.dev/)
- [Trivy](https://trivy.dev/)
- [docker(1) man page](https://man7.org/linux/man-pages/man1/docker.1.html)
