# Docker Errors

Verbatim error text, root cause, and fix for every Docker daemon, OCI runtime, build, Compose, network, volume, registry, and exit-code situation you will hit on Linux, macOS, and Windows.

## Setup

Docker is a layered system. Errors surface from whichever layer fails, so the first job is to identify the layer.

```text
docker (CLI, Go)            <- you type here, parses flags, talks HTTP to daemon
        |
        | unix:///var/run/docker.sock  (or tcp://, npipe:// on Windows)
        v
dockerd (the Engine, Go)    <- API server, image store, network manager, volume manager
        |
        | gRPC over unix socket
        v
containerd (Go)             <- container supervisor, image pull, snapshotter
        |
        | shim per container
        v
runc (Go, calls into libcontainer + Linux kernel) <- the actual OCI runtime
        |
        v
Linux kernel: namespaces, cgroups, capabilities, seccomp, AppArmor, overlayfs, netfilter
```

The OCI specs that govern this are in `github.com/opencontainers`:

- `image-spec` — the image manifest, config JSON, layer tar format, the `application/vnd.oci.image.manifest.v1+json` media type.
- `runtime-spec` — `config.json` plus the bundle directory layout and the lifecycle hooks (`prestart`, `createRuntime`, `createContainer`, `startContainer`, `poststart`, `poststop`).
- `distribution-spec` — the registry HTTP API (`/v2/`) used by `docker push`/`docker pull`.

Docker Engine versus Docker Desktop:

- **Docker Engine** is `dockerd` running directly on Linux. Native, fast, no VM.
- **Docker Desktop** on macOS or Windows runs a hidden Linux VM (LinuxKit on macOS, WSL2 or Hyper-V on Windows). The CLI on the host talks over a socket to the dockerd inside the VM. Almost every "weird" error on macOS or Windows traces back to "the daemon lives in a VM and the file system is bridged".

Rootless docker:

- The daemon runs as a regular user, not root. Uses user namespaces (`subuid`, `subgid`), `slirp4netns` or `vpnkit` for networking, and `fuse-overlayfs` for storage.
- Limitations: no privileged ports without `CAP_NET_BIND_SERVICE`, no host-network mode, restricted cgroup features, slower I/O.
- Set up via `dockerd-rootless-setuptool.sh install` then `systemctl --user start docker`.

Podman as drop-in:

- `alias docker=podman` works for ~90% of commands.
- Differences: daemonless (each `podman` invocation forks `conmon` + `runc`), pods are a first-class concept, default registry list comes from `/etc/containers/registries.conf`, default storage is in `~/.local/share/containers/storage`.

```bash
# Check daemon
docker info
docker version
systemctl status docker        # systemd
sudo journalctl -u docker -n 200 --no-pager
sudo journalctl -fu docker     # follow

# macOS / Windows
launchctl list | grep -i docker             # macOS
~/Library/Containers/com.docker.docker/Data/log/host/com.docker.driver.amd64-linux/docker.log  # Desktop log
```

## How to Read a Docker Error

Every Docker error is one of:

1. **CLI-side parse error** — bad flags, malformed `-v` or `-p` syntax. The CLI never even talked to the daemon.
2. **Cannot connect to the Docker daemon** — socket is missing, daemon down, perms wrong. The CLI tried to dial.
3. **Error response from daemon: ...** — daemon replied with an HTTP 4xx/5xx. The body of that error is the gold.
4. **OCI runtime create/exec failed: ...** — daemon asked containerd, containerd asked runc, and runc returned an error. The text after `OCI runtime create failed:` is straight from `libcontainer`.
5. **Container exited N** — the process inside the container exited. The number tells you whether the kernel killed it, the shell rejected it, or the program returned its own code.
6. **Build error from BuildKit** — `ERROR: failed to solve: ...`. BuildKit reports the failing step and command.
7. **Compose error** — Compose layers its own validation on top of the daemon, so error format is `ERROR: ...` and may be a YAML parse, a service-graph error, or a daemon error wrapped.

The three error streams to keep open:

```bash
docker logs -f <container>            # stdout/stderr from PID 1 inside the container
docker events                          # live event stream from the daemon
sudo journalctl -fu docker             # daemon's own logs (Linux)

# macOS Docker Desktop
log stream --predicate 'process CONTAINS "com.docker"' --info
~/Library/Containers/com.docker.docker/Data/log/host/com.docker.driver.amd64-linux/docker.log
```

When in doubt, run `docker events &` in another terminal and reproduce the failure — every state transition is reported.

## Daemon Connection Errors

### "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?"

```text
Cannot connect to the Docker daemon at unix:///var/run/docker.sock.
Is the docker daemon running?
```

**Cause:** The daemon is not running, or the socket file is missing/stale.

**Fix:**

```bash
# Linux
sudo systemctl status docker
sudo systemctl start docker
sudo systemctl enable docker
ls -l /var/run/docker.sock          # should be srw-rw---- root:docker

# If socket missing but daemon "running"
sudo journalctl -u docker -n 200 --no-pager
# Look for: "failed to start daemon: Error initializing network controller"
# Look for: "no space left on device"
# Look for: "containerd: not found"

# macOS / Windows
# Restart Docker Desktop. If stuck, quit then:
killall Docker && open -a Docker      # macOS
# Settings > Troubleshoot > Reset to factory defaults (last resort)
```

### "Cannot connect to the Docker daemon at tcp://X. Is the docker daemon running?"

```text
Cannot connect to the Docker daemon at tcp://10.0.0.5:2375.
Is the docker daemon running?
```

**Cause:** `DOCKER_HOST` is set to a remote daemon that is unreachable, refusing the connection, or listening on a different port.

**Fix:**

```bash
echo $DOCKER_HOST
unset DOCKER_HOST                       # use local socket again

# Test reachability
nc -vz 10.0.0.5 2375
curl http://10.0.0.5:2375/_ping         # should return OK

# If using TLS (always do this for tcp://)
docker --tlsverify --tlscacert=ca.pem --tlscert=cert.pem --tlskey=key.pem \
       -H tcp://10.0.0.5:2376 info
```

Never expose `tcp://` without TLS. An open `2375` is a remote-root vulnerability.

### "Got permission denied while trying to connect to the Docker daemon socket"

```text
Got permission denied while trying to connect to the Docker daemon socket
at unix:///var/run/docker.sock: Get "http://%2Fvar%2Frun%2Fdocker.sock/v1.43/containers/json":
dial unix /var/run/docker.sock: connect: permission denied
```

**Cause:** Your user is not in the `docker` group. The socket is `srw-rw---- root:docker` so only members of `docker` can write.

**Fix:**

```bash
sudo usermod -aG docker $USER
newgrp docker                # apply in current shell, or log out/in
id                           # confirm "docker" appears in groups

# Verify socket perms
ls -l /var/run/docker.sock
# srw-rw---- 1 root docker 0 ... /var/run/docker.sock
```

Security note: `docker` group membership is equivalent to root. Anyone in the group can `docker run -v /:/host -it ubuntu chroot /host` and own the box.

### "Error response from daemon: ..."

This is the generic preamble for any HTTP error from `dockerd`. Read the suffix.

```text
Error response from daemon: <THE INTERESTING PART>
```

**Fix:** Read the suffix. It tells you the actual error.

### "request returned 503 Service Unavailable for API route"

```text
Error response from daemon: request returned 503 Service Unavailable for API route
and version http://%2Fvar%2Frun%2Fdocker.sock/v1.43/containers/json, check if the
server supports the requested API version
```

**Cause:** Daemon is overloaded, mid-restart, or its containerd backend is unresponsive. Common when starting hundreds of containers concurrently or when the host is OOM.

**Fix:**

```bash
# Check daemon resource state
sudo journalctl -u docker -n 200 --no-pager | grep -E '(error|fatal|503)'
sudo systemctl status containerd
top -p $(pgrep dockerd)

# Throttle concurrent calls
docker info --format '{{.ServerErrors}}'

# If containerd is wedged
sudo systemctl restart containerd
sudo systemctl restart docker
```

### "client version X is too new. Maximum supported API version is Y"

```text
Error response from daemon: client version 1.45 is too new.
Maximum supported API version is 1.43
```

**Cause:** The CLI is newer than the daemon and is sending an API version the daemon does not support.

**Fix:**

```bash
# Pin the API version client-side
export DOCKER_API_VERSION=1.43
docker version            # confirm "API version" matches daemon

# Or upgrade the daemon
sudo apt-get update && sudo apt-get install --only-upgrade docker-ce docker-ce-cli
sudo systemctl restart docker
```

### "DOCKER_HOST" pointing somewhere invalid

```text
unable to resolve docker endpoint: open /home/me/.docker/desktop/certs/ca.pem: no such file
```

**Cause:** A `docker context` or `DOCKER_HOST` references a stale Docker Desktop install or a remote that no longer exists.

**Fix:**

```bash
docker context ls
docker context use default
docker context rm broken-context
unset DOCKER_HOST DOCKER_CERT_PATH DOCKER_TLS_VERIFY
```

## Image Pull / Push Errors

### "pull access denied for X, repository does not exist or may require 'docker login'"

```text
Error response from daemon: pull access denied for myorg/private-image,
repository does not exist or may require 'docker login':
denied: requested access to the resource is denied
```

**Cause:** Image is private and you have no token, or the repo name is wrong, or the registry namespace is wrong (e.g. `myorg/foo` on Docker Hub vs `ghcr.io/myorg/foo`).

**Fix:**

```bash
docker login                              # Docker Hub
docker login ghcr.io -u USER -p $TOKEN     # GitHub CR
docker login registry.gitlab.com           # GitLab CR
docker login 1234.dkr.ecr.us-east-1.amazonaws.com   # ECR (use aws ecr get-login-password)

aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 1234.dkr.ecr.us-east-1.amazonaws.com

cat ~/.docker/config.json                  # confirm auth saved
```

### "Error response from daemon: manifest unknown"

```text
Error response from daemon: manifest unknown:
manifest unknown
```

**Cause:** The tag/digest does not exist in the registry. Often a typo in the tag, or the build job that pushes that tag failed.

**Fix:**

```bash
# List tags via Hub API
curl -s "https://hub.docker.com/v2/repositories/library/nginx/tags/?page_size=100" | jq -r '.results[].name'

# List tags via /v2/ API (token may be required)
curl -s https://registry-1.docker.io/v2/library/nginx/tags/list

# Pull by digest (immutable)
docker pull nginx@sha256:abcd... 
```

### "manifest for X:tag not found"

```text
Error response from daemon: manifest for myimage:v1.2.3 not found:
manifest unknown: manifest unknown
```

**Cause:** Same as above but explicit about the tag. Usually a CI tag drift.

**Fix:**

```bash
docker manifest inspect myimage:v1.2.3
docker buildx imagetools inspect myimage:v1.2.3   # works for OCI manifests too
```

### "TLS handshake timeout"

```text
Error response from daemon: Get https://registry-1.docker.io/v2/:
net/http: TLS handshake timeout
```

**Cause:** Network blackhole or proxy sitting in the middle. Common in corporate networks or when egress firewalling is wrong.

**Fix:**

```bash
# Test egress
curl -v https://registry-1.docker.io/v2/

# Configure proxy for daemon (NOT the shell)
sudo mkdir -p /etc/systemd/system/docker.service.d
sudo tee /etc/systemd/system/docker.service.d/http-proxy.conf <<'EOF'
[Service]
Environment="HTTP_PROXY=http://proxy.corp:3128"
Environment="HTTPS_PROXY=http://proxy.corp:3128"
Environment="NO_PROXY=localhost,127.0.0.1,*.corp"
EOF
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### "x509: certificate signed by unknown authority"

```text
Error response from daemon: Get https://registry.corp.local/v2/:
x509: certificate signed by unknown authority
```

**Cause:** Private registry has a self-signed or internal-CA cert that the Docker daemon does not trust.

**Fix (option 1, trust the cert — preferred):**

```bash
sudo mkdir -p /etc/docker/certs.d/registry.corp.local
sudo cp ca.crt /etc/docker/certs.d/registry.corp.local/ca.crt
# No daemon restart needed
docker pull registry.corp.local/myimage:v1
```

**Fix (option 2, mark insecure — labs only):**

```json
// /etc/docker/daemon.json
{
  "insecure-registries": ["registry.corp.local"]
}
```

```bash
sudo systemctl restart docker
```

### "Head X: unauthorized"

```text
Error response from daemon: Head https://ghcr.io/v2/myorg/img/manifests/v1: unauthorized
```

**Cause:** Token expired, missing scope, or you logged in to the wrong registry.

**Fix:**

```bash
docker logout ghcr.io
docker login ghcr.io -u USER -p $GITHUB_TOKEN
# Token must have read:packages (pull) and write:packages (push)
```

### "denied: requested access to the resource is denied"

```text
denied: requested access to the resource is denied
```

**Cause:** Authenticated but lacking permission on this repo (push to a repo you do not own, or wrong tag namespace).

**Fix:** Verify the namespace matches your username/org and that the token scope includes write.

### "toomanyrequests: You have reached your pull rate limit"

```text
Error response from daemon: toomanyrequests: You have reached your pull rate limit.
You may increase the limit by authenticating and upgrading: https://www.docker.com/increase-rate-limits
```

**Cause:** Docker Hub anonymous pull rate limit (100 pulls per 6 hours per IP). Authenticated free is 200, paid is unlimited.

**Fix:**

```bash
docker login
# Or use a pull-through cache (registry:2 in mirror mode)
# Or use an alternative registry: ghcr.io, public.ecr.aws, quay.io, mcr.microsoft.com
```

```json
// /etc/docker/daemon.json
{
  "registry-mirrors": ["https://mirror.gcr.io"]
}
```

## Build Errors

### "Error: building at STEP X: failed to fetch metadata"

```text
Error: building at STEP "FROM alpine:3.19": failed to fetch metadata:
Get https://registry-1.docker.io/v2/: net/http: TLS handshake timeout
```

**Cause:** Cannot reach the registry to resolve the base image. Network or proxy issue.

**Fix:** See "TLS handshake timeout" above. Configure proxy on the daemon, not the shell.

### "Error: failed to compute cache key"

```text
ERROR: failed to solve: failed to compute cache key:
"/app/package.json" not found: not found
```

**Cause:** A `COPY src dst` instruction references a path that does not exist in the build context.

**Fix:**

```bash
# Confirm the path exists relative to the build context (the directory you pass to docker build)
ls -la app/package.json
docker build -f Dockerfile .   # context is "."

# .dockerignore excluding the file?
grep -E "(package|node_modules)" .dockerignore

# COPY syntax: <src> is relative to context, <dst> is in image
```

```dockerfile
# Wrong — context-relative paths must exist
COPY app/package.json /app/

# Correct — verified path exists in context
COPY package.json /app/
```

### "ERROR: failed to solve: process did not complete successfully: exit code: N"

```text
------
 > [3/8] RUN apt-get update && apt-get install -y curl:
#7 0.234 Reading package lists...
#7 0.567 E: Unable to locate package curlx
------
ERROR: failed to solve: process "/bin/sh -c apt-get update && apt-get install -y curlx" did not complete successfully: exit code: 100
```

**Cause:** A `RUN` step exited non-zero. Read the lines above the `ERROR:` for the actual command output.

**Fix:**

```bash
# Re-run with full output and no cache
docker build --no-cache --progress=plain -t myimg .

# Drop into the failing stage
docker run --rm -it $(docker build -q --target=stage-name .) sh
```

```dockerfile
# Common apt-get error fix: need apt-get update in the same RUN as install,
# else cache may keep stale package lists.
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl ca-certificates \
 && rm -rf /var/lib/apt/lists/*
```

### "Step N/M : COPY X Y => ERROR"

```text
Step 4/12 : COPY ./build /app
 ---> ERROR
COPY failed: stat /var/lib/docker/tmp/docker-builder123/build: no such file or directory
```

**Cause:** Source path missing from context (often because `.dockerignore` excluded it, or the `docker build .` was run from the wrong directory).

**Fix:**

```bash
cat .dockerignore
ls build/                         # confirm it exists relative to context
docker build -t myimg .            # context is current directory
```

### "Dockerfile parse error line N: unknown instruction: X"

```text
Error response from daemon: Dockerfile parse error line 5:
unknown instruction: COPYY
```

**Cause:** Typo in instruction name, or a stray non-instruction line at top level.

**Fix:** Instructions are case-sensitive uppercase by convention but accepted lowercase. Valid: `FROM ARG ENV RUN COPY ADD CMD ENTRYPOINT EXPOSE WORKDIR USER VOLUME LABEL HEALTHCHECK ONBUILD STOPSIGNAL SHELL MAINTAINER` (deprecated).

```dockerfile
FROM alpine:3.19
COPY app /app          # not COPYY
WORKDIR /app
CMD ["./run"]
```

### "invalid argument X for -t, --tag flag"

```text
invalid argument "MyImage:V1" for "-t, --tag" flag:
invalid reference format: repository name must be lowercase
```

**Cause:** Tag has uppercase letters, or contains forbidden characters. Repository names must be lowercase; tags can have `[A-Za-z0-9_.-]`, max 128 chars, no leading `.` or `-`.

**Fix:**

```bash
docker build -t myimage:v1 .       # all lowercase repo
docker build -t myimage:V1 .       # uppercase tag is OK, repo must be lowercase
```

### "BuildKit failed: ..."

```text
ERROR: failed to solve: rpc error: code = Unknown desc = ...
```

**Cause:** BuildKit-specific failure (the modern builder, default since Docker 23.0). Often a sub-error wrapped in gRPC.

**Fix:**

```bash
# Get verbose output
docker build --progress=plain --no-cache .

# Restart the BuildKit container if using docker-container driver
docker buildx ls
docker buildx inspect --bootstrap
docker buildx rm mybuilder
docker buildx create --name mybuilder --use --bootstrap
```

### "Cannot connect to the Docker daemon" during build (containerd image store gotcha)

```text
ERROR: failed to solve: Cannot connect to the Docker daemon at unix:///var/run/docker.sock
```

**Cause:** Buildx with `docker-container` driver cannot reach the daemon, or you have enabled the containerd image store but a tool expects the legacy graph driver.

**Fix:**

```bash
docker buildx use default
docker info | grep -i 'storage driver\|server'
# If using containerd image store and an old tool fails, switch back:
# Settings > General > Use containerd for pulling and storing images (toggle off, restart)
```

## OCI Runtime Errors

### "exec: X: executable file not found in $PATH"

```text
Error response from daemon: OCI runtime create failed: container_linux.go:380:
starting container process caused: exec: "myapp": executable file not found in $PATH:
unknown
```

**Cause:** The CMD/ENTRYPOINT binary is not in `PATH` inside the image, or the binary file is missing, or the architecture is wrong (built for amd64 image running on arm64 host).

**Fix:**

```bash
docker run --rm -it myimage sh             # use sh/bash to inspect
docker run --rm myimage which myapp
docker run --rm myimage ls -l /usr/local/bin/myapp
docker run --rm myimage file /usr/local/bin/myapp   # check arch
```

```dockerfile
# Use absolute path or set PATH
ENV PATH="/opt/myapp/bin:${PATH}"
ENTRYPOINT ["/opt/myapp/bin/myapp"]
```

### "chdir to cwd set in config.json failed: no such file or directory"

```text
OCI runtime create failed: container_linux.go:380:
starting container process caused: chdir to cwd ("/app") set in config.json failed:
no such file or directory: unknown
```

**Cause:** `WORKDIR /app` was set but the directory does not exist in the image (rare — `WORKDIR` normally creates it) or a `-w /missing` flag at run time.

**Fix:**

```bash
docker run --rm -w /existing-dir myimage ls
docker run --rm myimage ls /app          # confirm dir exists in image
```

### "write sysctl key X: open /proc/sys/X: permission denied"

```text
OCI runtime create failed: write sysctl key net.core.somaxconn:
open /proc/sys/net/core/somaxconn: permission denied
```

**Cause:** Tried to set a kernel sysctl that the runtime cannot write — either it is not namespaced, or the daemon does not have capability, or rootless mode forbids it.

**Fix:**

```bash
# Allow the namespaced sysctls only
docker run --sysctl net.core.somaxconn=1024 myimage     # OK in user net ns
docker run --sysctl kernel.shm_rmid_forced=1 myimage    # OK in IPC ns

# For non-namespaced sysctls, set on the host instead
sudo sysctl -w vm.max_map_count=262144
```

### "OCI runtime exec failed: exec failed: ..."

```text
Error response from daemon: OCI runtime exec failed:
exec failed: unable to start container process: exec: "bash": executable file not found in $PATH:
unknown
```

**Cause:** `docker exec` ran a binary not present in the container. Many minimal images (`alpine`, `distroless`, `scratch`) lack bash or even sh.

**Fix:**

```bash
docker exec -it mycontainer sh             # alpine has sh, not bash
docker exec -it mycontainer /busybox sh    # busybox-based images
docker exec -it mycontainer /bin/busybox sh
# distroless: no shell at all — copy in busybox via debug image
docker run --rm -it --pid=container:mycontainer --net=container:mycontainer \
  --cap-add=SYS_PTRACE nicolaka/netshoot
```

### "OCI runtime create failed: invalid CDI device name"

```text
OCI runtime create failed: invalid CDI device name "nvidia.com/gpu=all": ...
```

**Cause:** CDI (Container Device Interface) device requested but the CDI spec is missing or malformed. Common with GPU containers.

**Fix:**

```bash
ls /etc/cdi /var/run/cdi               # CDI specs
nvidia-ctk cdi list                     # if using nvidia container toolkit
nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml
docker run --rm --device nvidia.com/gpu=all nvidia/cuda:12.4.0-base-ubuntu22.04 nvidia-smi
```

### "ulimit X is not valid"

```text
OCI runtime create failed: ulimit "nofiles=65536:65536" is not valid: invalid ulimit name nofiles
```

**Cause:** Misspelled ulimit. Valid names: `core cpu data fsize locks memlock msgqueue nice nofile nproc rss rtprio rttime sigpending stack`.

**Fix:**

```bash
docker run --ulimit nofile=65536:65536 myimage     # not nofiles
docker run --ulimit nproc=8192:8192 myimage
```

## Container Lifecycle Errors

### "Conflict. The container name is already in use"

```text
docker: Error response from daemon: Conflict. The container name "/web" is already in use
by container "abcd1234567890". You have to remove (or rename) that container to be
able to reuse that name.
```

**Cause:** A previous container with that `--name` still exists (running or stopped).

**Fix:**

```bash
docker rm -f web                                # remove the conflict
docker run --name web --rm -d nginx              # --rm auto-removes on exit

# In Compose, this is usually a stale container from a previous run
docker compose down
docker compose up -d
```

### "No such container: X"

```text
Error: No such container: web
```

**Cause:** The container name or ID is wrong, or it was already removed.

**Fix:**

```bash
docker ps -a                            # all containers, including stopped
docker ps -a --filter "name=web"
docker ps --format 'table {{.ID}}\t{{.Names}}\t{{.Status}}'
```

### "Container X is not running"

```text
Error response from daemon: Container web is not running
```

**Cause:** Tried to `docker exec` or `docker logs --follow` on a stopped container.

**Fix:**

```bash
docker ps -a --filter "name=web"
docker logs web                          # logs work even after stop
docker start web
docker exec -it web sh
```

### "cannot stop a paused container, unpause and try again"

```text
Error response from daemon: cannot stop a paused container, unpause and try again
```

**Cause:** Container was paused via `docker pause` and is frozen at the cgroup freezer.

**Fix:**

```bash
docker unpause web
docker stop web
```

### "cannot remove a running container"

```text
Error response from daemon: You cannot remove a running container abcd1234.
Stop the container before attempting removal or force remove
```

**Fix:**

```bash
docker stop web && docker rm web
docker rm -f web                         # force (sends SIGKILL)
```

### "Cannot kill container: X: No such container"

```text
Error response from daemon: Cannot kill container: web: No such container: web
```

**Cause:** Race — container exited between your `kill` and the daemon's lookup.

**Fix:** Treat it as success; the container is gone.

### "container X is dead, refusing to start"

```text
Error response from daemon: container abcd1234 is dead, refusing to start
```

**Cause:** Container is in `Dead` state — runc could not finish setup or teardown, leaving the bundle inconsistent.

**Fix:**

```bash
docker rm -f abcd1234
sudo journalctl -u docker -n 200 | grep -i abcd1234
# Common root causes:
# - Storage driver corruption (overlay2 inode exhaustion)
# - cgroup mount failure
# - Daemon crash mid-create
# Last resort: docker system prune, then re-create.
```

## Network Errors

### "port is already allocated"

```text
docker: Error response from daemon: driver failed programming external connectivity
on endpoint web (abcd1234): Bind for 0.0.0.0:8080 failed: port is already allocated
```

**Cause:** Another container or another process on the host has bound that port.

**Fix:**

```bash
docker ps --format 'table {{.Names}}\t{{.Ports}}' | grep 8080
sudo ss -ltnp | grep ':8080 '
sudo lsof -i :8080
sudo fuser 8080/tcp

docker rm -f conflicting-container
# Or run on a different host port
docker run -p 8081:80 nginx
```

### "address X already in use"

```text
Error response from daemon: address 172.20.0.0/16 already in use
```

**Cause:** Tried to create a network with a subnet that overlaps an existing Docker network or a host route.

**Fix:**

```bash
docker network ls
docker network inspect $(docker network ls -q) | jq -r '.[] | "\(.Name)\t\(.IPAM.Config)"'
ip route | grep 172.20

# Pick a non-overlapping subnet
docker network create --subnet 172.30.0.0/24 mynet
```

### "operation not supported" creating veth pair

```text
Error response from daemon: failed to create endpoint web on network bridge:
failed to add the host (veth1234) <=> sandbox (veth5678) pair interfaces:
operation not supported
```

**Cause:** Kernel missing `CONFIG_VETH`, or AppArmor/SELinux blocking, or running in a nested container without `--privileged`.

**Fix:**

```bash
zcat /proc/config.gz | grep VETH       # or /boot/config-$(uname -r)
sudo modprobe veth
sudo dmesg | tail -50
# Nested docker (docker-in-docker) needs --privileged
docker run --privileged docker:dind
```

### "network with name X already exists"

```text
Error response from daemon: network with name mynet already exists
```

**Fix:**

```bash
docker network ls --filter name=mynet
docker network rm mynet
docker network create mynet
```

### "No such network: X" / "network X not found"

```text
Error response from daemon: network frontend not found
```

**Fix:**

```bash
docker network ls
docker network create frontend
# Compose recreates networks: docker compose up always works.
```

### "cannot remove network X: has active endpoints"

```text
Error response from daemon: error while removing network: network mynet id abcd
has active endpoints
```

**Fix:**

```bash
docker network inspect mynet --format '{{range .Containers}}{{.Name}} {{end}}'
docker network disconnect -f mynet web
docker network rm mynet
# Or stop/remove the connected containers first.
```

### "iptables failed: ... CAP_NET_ADMIN"

```text
Error response from daemon: driver failed programming external connectivity on endpoint web:
iptables failed: iptables --wait -t nat -A DOCKER -p tcp -d 0/0 --dport 8080 -j DNAT
--to-destination 172.17.0.2:80 ! -i docker0: iptables: No chain/target/match by that name.
```

**Cause:** A competing firewall manager wiped out Docker's iptables chains, or `iptables` is using `nf_tables` backend without compat layer, or `firewalld` reloaded.

**Fix:**

```bash
# Recreate Docker chains by restarting
sudo systemctl restart docker

# Persistent fix on RHEL/Fedora with firewalld
sudo firewall-cmd --permanent --zone=trusted --add-interface=docker0
sudo firewall-cmd --reload

# Switch iptables backend if needed
sudo update-alternatives --config iptables    # choose iptables-legacy if mixed
```

### "HostPort is not supported on rootless mode"

```text
Error response from daemon: failed to create endpoint web: HostPort is not supported on rootless mode
```

**Cause:** Rootless docker cannot bind privileged or host ports the same way. By default it uses slirp4netns.

**Fix:**

```bash
# Use a high port
docker run -p 8080:80 nginx          # OK
docker run -p 80:80 nginx            # needs CAP_NET_BIND_SERVICE
sudo setcap cap_net_bind_service=ep $(which rootlesskit)
systemctl --user restart docker
```

## Volume Errors

### "Error processing tar file"

```text
Error response from daemon: Error processing tar file(exit status 1):
unexpected EOF
```

**Cause:** Image layer tar is corrupt or truncated — common after a partial pull or disk-full mid-pull.

**Fix:**

```bash
docker rmi <image-id>
docker pull <image>:<tag>
docker system df
docker system prune
df -h /var/lib/docker
```

### "bind source path does not exist"

```text
docker: Error response from daemon: invalid mount config for type "bind":
bind source path does not exist: /home/me/data
```

**Cause:** `-v /host:/container` source does not exist on the host.

**Fix:**

```bash
mkdir -p /home/me/data
docker run -v /home/me/data:/data myimage
```

### "Mounts denied: not shared from the host" (Docker Desktop)

```text
docker: Error response from daemon: Mounts denied:
The path /Volumes/extdrive is not shared from the host and is not known to Docker.
You can configure shared paths from Docker -> Preferences... -> Resources -> File Sharing.
```

**Cause:** Docker Desktop on macOS only mounts whitelisted paths into the Linux VM. By default `/Users` is shared but external volumes and other paths are not.

**Fix:** Docker Desktop > Settings > Resources > File Sharing > add the path > Apply & Restart.

### "invalid volume specification"

```text
Error response from daemon: invalid volume specification:
'C:\Users\me\app:/app': invalid mode: /app
```

**Cause:** Windows path with colons confuses the `-v` parser.

**Fix:**

```bash
# Use forward slashes or quote
docker run -v "C:/Users/me/app:/app" myimage
docker run -v "/$(pwd):/app" myimage          # Git Bash
# Or use --mount syntax (long form, unambiguous)
docker run --mount type=bind,source=C:\Users\me\app,target=/app myimage
```

### "cannot remove volume X: volume is in use"

```text
Error response from daemon: remove myvol: volume is in use - [abcd1234]
```

**Fix:**

```bash
docker ps -a --filter volume=myvol
docker rm -f abcd1234
docker volume rm myvol
docker volume prune          # remove all unused
```

## Storage Driver Errors

### "failed to register layer"

```text
Error response from daemon: failed to register layer: re-exec error: exit status 1:
output: write /var/lib/docker/overlay2/.../foo: no space left on device
```

**Cause:** `/var/lib/docker` is full (or its inodes are exhausted).

**Fix:**

```bash
df -h /var/lib/docker
df -i /var/lib/docker         # inodes
docker system df
docker system prune -af --volumes      # nuke unused images/containers/volumes
sudo journalctl --vacuum-time=2d        # also free journald

# Move /var/lib/docker to bigger disk
sudo systemctl stop docker
sudo rsync -aP /var/lib/docker/ /mnt/big/docker/
# /etc/docker/daemon.json: { "data-root": "/mnt/big/docker" }
sudo systemctl start docker
```

### "no space left on device" (overlay2 disk pressure)

Same as above plus inode exhaustion is common with many small files. `df -i` is your friend.

### "device mapper:" (devicemapper, mostly historical)

```text
Error response from daemon: devmapper: Thin Pool has 0 free data blocks
which is less than minimum required 163840 free data blocks. Create more free space
in thin pool or use dm.min_free_space option to change behavior
```

**Cause:** The legacy devicemapper storage driver. Unless on RHEL 7.x with the loop-lvm setup, you should not see this in 2026.

**Fix:** Migrate to `overlay2`. Edit `/etc/docker/daemon.json`:

```json
{ "storage-driver": "overlay2" }
```

```bash
sudo systemctl stop docker
sudo mv /var/lib/docker /var/lib/docker.old
sudo systemctl start docker
docker info | grep 'Storage Driver'
```

### "btrfs:" issues

```text
Error response from daemon: failed to register layer: rename ... : invalid argument
```

**Cause:** btrfs subvolume issues — usually nesting limits or quota corruption.

**Fix:**

```bash
sudo btrfs subvolume list /var/lib/docker
sudo btrfs quota disable /var/lib/docker
# Or migrate to overlay2 on top of ext4/xfs.
```

### "failed to update overlay2 fs"

```text
Error: failed to update overlay2 fs: ...
```

**Cause:** Underlying filesystem does not support overlayfs (e.g. NFS), or `lowerdir`/`upperdir` perms wrong.

**Fix:**

```bash
mount | grep /var/lib/docker
# overlay2 requires xfs (with d_type=1) or ext4. Avoid NFS for /var/lib/docker.
xfs_info /var/lib/docker | grep ftype
```

## Permission / Rootless Errors

### "permission denied"

Usually:

1. User not in `docker` group — see Daemon Connection Errors section.
2. Rootless: subuid/subgid not configured.
3. SELinux enforcing — bind-mount labelling.

```bash
# SELinux: relabel a bind-mount
docker run -v /data:/data:Z myimage           # private label
docker run -v /data:/data:z myimage           # shared label (lowercase z)
# or
sudo chcon -Rt svirt_sandbox_file_t /data
```

### "the input device is not a TTY"

```text
the input device is not a TTY
```

**Cause:** `-t` requested a TTY but stdin is a pipe (CI, `<(cat)`, etc.).

**Fix:**

```bash
docker run -i myimage          # piped input, no -t
docker exec -i myimage sh -c 'echo hi'
# Use -t only when stdin is a real terminal
[ -t 0 ] && tty=-t              # detect
docker run $tty -i myimage
```

### "could not find a sub-uid range for current user"

```text
[ERROR] Could not find sub-uid range for user me in /etc/subuid
```

**Cause:** Rootless docker needs `/etc/subuid` and `/etc/subgid` mappings.

**Fix:**

```bash
sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 $USER
grep $USER /etc/subuid /etc/subgid
dockerd-rootless-setuptool.sh install
```

### "newuidmap: bind subuid range"

```text
newuidmap: write to uid_map failed: Operation not permitted
```

**Fix:**

```bash
sudo apt-get install uidmap                  # Debian/Ubuntu
sudo dnf install shadow-utils-subid          # Fedora
sudo setcap cap_setuid=ep /usr/bin/newuidmap
sudo setcap cap_setgid=ep /usr/bin/newgidmap
```

### "rootlesskit: error"

```text
[rootlesskit:parent] error: failed to start the child: ...
```

**Fix:**

```bash
# Tear down and reinstall
systemctl --user stop docker
dockerd-rootless-setuptool.sh uninstall
rm -rf ~/.local/share/docker
dockerd-rootless-setuptool.sh install
journalctl --user -u docker -n 200
```

## Compose Errors

### "Couldn't connect to Docker daemon at http+docker://localhost"

```text
ERROR: Couldn't connect to Docker daemon at http+docker://localhost - is it running?
```

**Cause:** Same root cause as the `docker` CLI version.

**Fix:** See Daemon Connection Errors. Ensure `DOCKER_HOST` is correct and the daemon is up.

### "Compose file format X is not supported"

```text
ERROR: Version in "./docker-compose.yml" is unsupported. You might be seeing this
error because you're using the wrong Compose file version.
```

**Cause:** Old `docker-compose` v1 (Python) cannot read newer `version: "3.9"`, or you are using the new compose v2 which warns the `version:` key is obsolete.

**Fix:**

```yaml
# docker-compose.yml — Compose v2 (2026): no version key needed
services:
  web:
    image: nginx
```

```bash
docker compose version       # built-in plugin (v2)
docker-compose version       # legacy v1, deprecated
```

### "Service 'X' failed to build"

```text
ERROR: Service 'web' failed to build: failed to compute cache key:
"/app/package.json" not found
```

**Cause:** A Dockerfile build error wrapped by Compose. Read the underlying message.

**Fix:** Reproduce with `docker build` and fix as in the Build Errors section.

```bash
docker compose build --no-cache --progress=plain web
```

### "Service 'X' depends on undefined service: Y"

```yaml
services:
  web:
    depends_on:
      - dbX           # typo
  db:
    image: postgres
```

**Fix:** Match the service name exactly:

```yaml
services:
  web:
    depends_on:
      - db
  db:
    image: postgres:16
```

### "Network with name X already exists with different config"

```text
ERROR: Network with name myproject_default already exists with different config:
{driver=bridge, ipam={Subnet:172.21.0.0/16}} vs requested {Subnet:172.22.0.0/16}
```

**Cause:** A previous run created a network with different settings.

**Fix:**

```bash
docker compose down --volumes --remove-orphans
docker network rm myproject_default
docker compose up -d
```

### "Container Y is unhealthy"

```text
ERROR: for web  Container db is unhealthy
```

**Cause:** A `depends_on` with `condition: service_healthy` is waiting on a service whose `HEALTHCHECK` returned non-zero.

**Fix:**

```bash
docker inspect --format='{{json .State.Health}}' db | jq
docker logs db
# Adjust HEALTHCHECK or fix the underlying service
```

```yaml
services:
  db:
    image: postgres:16
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
  web:
    depends_on:
      db:
        condition: service_healthy
```

### "Cannot start service Y: driver failed programming external connectivity"

Same as the bridge network port-already-allocated error above. Resolve with `docker ps`/`ss`.

### "yaml.parser.ParserError" / "yaml: line N: did not find expected key"

```text
ERROR: yaml.parser.ParserError: while parsing a block mapping
  in "./docker-compose.yml", line 7, column 3
expected <block end>, but found '<scalar>'
```

**Cause:** YAML indentation is inconsistent (tabs vs spaces, two spaces vs four).

**Fix:**

```bash
yamllint docker-compose.yml
docker compose config            # parse + dump canonical form
# Use 2-space indents, never tabs
```

### "The Compose file X is invalid"

```text
ERROR: The Compose file './docker-compose.yml' is invalid because:
services.web.ports contains an invalid type, it should be a list
```

**Fix:** `ports`, `environment`, `volumes`, `depends_on` accept either list or map — read the diagnostic.

```yaml
services:
  web:
    image: nginx
    ports:
      - "8080:80"          # list of "host:container"
    environment:
      - DEBUG=1            # list of KEY=VAL
    # OR map form:
    # environment:
    #   DEBUG: "1"
```

### "Conflict. The container name is already in use" (Compose)

Compose uses `<project>_<service>_<index>` names. If a previous run crashed mid-down it can leave stragglers.

```bash
docker compose down --remove-orphans
docker ps -a --filter "name=myproject_"
docker rm -f $(docker ps -aq --filter "name=myproject_")
```

### "WARNING: The X variable is not set. Defaulting to a blank string"

```text
WARNING: The IMAGE_TAG variable is not set. Defaulting to a blank string.
```

**Fix:** Provide via `.env` (auto-loaded), `--env-file`, or shell.

```bash
# .env
IMAGE_TAG=v1.2.3
```

```yaml
services:
  web:
    image: myorg/web:${IMAGE_TAG:-latest}     # default if unset
```

## Container Exit Code Interpretation

```text
0    success
1    generic error (often app exception inside container)
2    misuse of shell builtin
125  docker daemon error (could not start container)
126  container command found but not executable
127  container command not found
128  invalid argument to exit (rare)
130  SIGINT  (128 + 2)   — Ctrl+C from `docker run -it`
137  SIGKILL (128 + 9)   — usually OOM-killer or `docker kill`
139  SIGSEGV (128 + 11)  — segfault
143  SIGTERM (128 + 15)  — graceful shutdown via `docker stop`
255  exit code wrap-around (signed/unsigned mismatch)
```

```bash
docker run myimage; echo "exit=$?"
docker inspect --format='{{.State.ExitCode}} {{.State.Error}} {{.State.OOMKilled}}' web
docker inspect --format='{{.State.Status}} {{.State.StartedAt}} {{.State.FinishedAt}}' web
```

Practical hints:

- 125 → daemon-side; check `journalctl -u docker`.
- 126 → file is there but not `+x` (`COPY --chmod=755`).
- 127 → typo in CMD or wrong PATH.
- 137 → `dmesg | grep -i kill`; check `--memory` and host RAM.
- 139 → app crashed. `docker run --cap-add=SYS_PTRACE` and `gcore`/`gdb`.
- 143 → graceful stop. If the app received SIGTERM but ignored it, `docker stop` will then send SIGKILL after `--stop-timeout` (default 10s).

## OOM in Container

Symptoms:

- Container exits with 137.
- `docker inspect ... .State.OOMKilled == true`.
- `dmesg` shows: `Memory cgroup out of memory: Killed process 12345 (myapp) total-vm:...`.

```bash
docker inspect --format='{{.State.OOMKilled}}' web
sudo dmesg -T | tail -100 | grep -E 'Memory cgroup|Killed process'
journalctl -k | grep -i 'memory cgroup'
```

cgroup-v2 vs v1: v2 sets `memory.max` on the container's cgroup; OOM-killer is invoked when usage hits that limit. Same mechanism, different paths.

```bash
# v2 path (Linux 5.4+ on most modern distros)
ls /sys/fs/cgroup/system.slice/docker-<id>.scope/memory.max

# Inspect from inside the container
cat /sys/fs/cgroup/memory.max          # cgroup v2
cat /sys/fs/cgroup/memory/memory.limit_in_bytes  # cgroup v1
```

Tuning:

```bash
# Hard limit
docker run --memory=512m --memory-swap=1g myimage

# Soft reservation (best-effort, kicks in under host pressure)
docker run --memory-reservation=256m --memory=512m myimage

# Disable OOM-killer for this container (DON'T — causes whole host to wedge)
docker run --oom-kill-disable --memory=512m myimage

# Bias the OOM-killer
docker run --oom-score-adj=500 myimage    # higher = killed sooner
```

If a JVM is the OOM victim, also pass:

```bash
docker run -e JAVA_TOOL_OPTIONS="-XX:MaxRAMPercentage=75.0" --memory=2g myjava
```

## SIGTERM Not Honored

`docker stop` sends SIGTERM, waits `--stop-timeout` (default 10s), then SIGKILL. If your container always takes 10s to stop, PID 1 is not getting the signal.

The classic broken pattern:

```dockerfile
# Broken — sh -c forks; sh becomes PID 1 and swallows SIGTERM
CMD sh -c "node server.js"
CMD ["sh", "-c", "node server.js"]
```

```dockerfile
# Fixed — exec form, your app is PID 1
CMD ["node", "server.js"]

# Or with shell wrapping, exec into the real binary
CMD ["sh", "-c", "exec node server.js"]
```

When you genuinely need a shell wrapper, use a tiny init that forwards signals:

```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends tini && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["node", "server.js"]
```

```bash
docker run --init myimage     # uses docker-init (tini-equivalent) automatically
```

Verify your app sees the signal:

```bash
docker run -d --name web myimage
docker exec web ps aux        # PID 1 should be your app, not sh
docker stop web
docker logs web | tail        # confirm graceful shutdown logs
```

## Healthcheck Errors

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1
```

States: `starting → healthy | unhealthy`. During `start_period` failures do not count toward `retries`.

Exit codes returned by the check command:

- `0` healthy
- `1` unhealthy
- `2` reserved — do not use

```bash
docker inspect --format='{{json .State.Health}}' web | jq
docker events --filter event=health_status
```

```yaml
services:
  web:
    image: myorg/web
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      start_period: 10s
      retries: 3
```

If you have no `curl` in the image, use `wget --spider`, `nc -z`, or a built-in `/healthz` from the binary itself:

```dockerfile
HEALTHCHECK CMD ["/myapp", "healthcheck"]
```

## Docker Desktop Specific

### "Cannot connect to the Docker daemon" after Desktop crash (macOS/Windows)

```bash
# macOS
killall Docker
open -a Docker
# Wait for whale icon to stop pulsing.

# View logs
~/Library/Containers/com.docker.docker/Data/log/host/com.docker.driver.amd64-linux/docker.log
~/Library/Containers/com.docker.docker/Data/log/vm/dockerd.log
```

```powershell
# Windows
Restart-Service com.docker.service
Get-Process -Name "Docker Desktop" | Stop-Process -Force
Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe"
```

### "An invalid file system has been detected" (Hyper-V on Windows)

```text
An invalid file system has been detected. Please reset Docker Desktop to factory defaults.
```

**Cause:** Hyper-V VHDX corruption.

**Fix:**

```powershell
# Quit Docker Desktop, then
Remove-Item "$env:LOCALAPPDATA\Docker\wsl\data\ext4.vhdx" -Force
# Or "Reset to factory defaults" from Docker Desktop > Troubleshoot
```

### "WSL2 backend not started"

```text
WSL2 backend not started. Please make sure WSL2 is enabled and running.
```

**Fix:**

```powershell
wsl --status
wsl --update
wsl --set-default-version 2
wsl --shutdown
```

### File-sharing performance on macOS

Bind mounts on macOS go through the bridged file system. Pre-2023 it was `osxfs` (slow). 2023+ Docker Desktop has VirtioFS on macOS 12.5+, which is much faster.

```text
Settings > General > Choose file-sharing implementation > VirtioFS
```

For node_modules-heavy projects:

- Put `node_modules` in a named volume (not bind-mounted from host).
- Use `:cached` or `:delegated` mount flags (legacy, ignored by VirtioFS but harmless).

```yaml
services:
  app:
    volumes:
      - .:/app
      - /app/node_modules     # anonymous volume hides host path
```

### "Docker Desktop is starting..." stuck loop

```bash
# macOS — wipe state
killall Docker
rm -rf ~/Library/Group\ Containers/group.com.docker
rm -rf ~/Library/Containers/com.docker.docker
rm -rf ~/.docker/
open -a Docker
```

```powershell
# Windows
wsl --shutdown
Remove-Item "$env:LOCALAPPDATA\Docker" -Recurse -Force
Remove-Item "$env:APPDATA\Docker Desktop" -Recurse -Force
Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe"
```

## Buildx / Multi-arch Errors

### "multiple platforms feature is currently not supported for docker driver"

```text
ERROR: multiple platforms feature is currently not supported for docker driver.
Please switch to a different driver (eg. "docker buildx create --use")
```

**Cause:** The default `docker` driver only supports single-platform builds. Multi-platform builds need the `docker-container` driver.

**Fix:**

```bash
docker buildx create --name multi --driver docker-container --use --bootstrap
docker buildx ls
docker buildx build --platform=linux/amd64,linux/arm64 -t myorg/web:v1 --push .
```

### "failed to push: failed commit on ref"

```text
ERROR: failed to solve: failed to push myorg/web:v1: failed commit on ref ...:
unexpected status: 413 Request Entity Too Large
```

**Cause:** Registry rejecting a layer larger than its limit, or auth token expired mid-push.

**Fix:**

```bash
docker logout && docker login
# Reduce layer size — multi-stage build, slim base, fewer artifacts.
# For self-hosted registry:2 raise client_max_body_size in nginx fronting it.
```

### "qemu: uncaught target signal"

```text
qemu: uncaught target signal 11 (Segmentation fault) - core dumped
```

**Cause:** Cross-architecture emulation under QEMU hit a glibc/kernel mismatch or a syscall not yet emulated.

**Fix:**

```bash
docker run --rm --privileged tonistiigi/binfmt --install all   # register binfmt
docker buildx ls
# If a specific arch keeps crashing, build natively on that arch
# (GitHub Actions has linux/arm64 runners now).
```

```yaml
# GitHub Actions multi-arch
- uses: docker/setup-qemu-action@v3
- uses: docker/setup-buildx-action@v3
- uses: docker/build-push-action@v5
  with:
    platforms: linux/amd64,linux/arm64
    push: true
    tags: myorg/web:v1
```

## Registry Errors

### "denied: insufficient_scope: authorization failed"

```text
denied: insufficient_scope: authorization failed
```

**Cause:** Token has read but not write scope, or vice versa.

**Fix:**

```bash
# GitHub: token must include write:packages
# Docker Hub: PAT scope must include "Read & Write" if pushing
docker logout && docker login
```

### "blob upload unknown to registry"

```text
blob upload unknown to registry
```

**Cause:** Registry storage has been pruned mid-upload, or two clients are pushing the same blob and racing.

**Fix:**

```bash
docker buildx imagetools inspect myorg/web:v1
# Retry. If recurrent, registry GC may be too aggressive.
```

### "manifest unknown" / "name unknown" / "name invalid"

```text
manifest unknown                       # manifest with that ref does not exist
name unknown: repository name not known to registry  # whole repo missing
name invalid: invalid repository name  # syntactically wrong (uppercase, etc.)
```

**Fix:** Confirm the full ref `host[:port]/namespace/repo:tag` and that the repo has been created/seeded.

### Bearer-token-vs-Basic-auth

Modern registries use OAuth Bearer tokens. The flow:

1. Client `GET /v2/` -> `401` with `Www-Authenticate: Bearer realm="...",service="...",scope="..."`.
2. Client requests token from realm with credentials -> JWT.
3. Client retries with `Authorization: Bearer <jwt>`.

Some private registries still use HTTP Basic. `docker login` handles both. If you write your own client and see endless 401s, you probably did not follow the redirect to the token endpoint.

```bash
TOKEN=$(curl -s -u user:pass "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/nginx:pull" | jq -r .token)
curl -s -H "Authorization: Bearer $TOKEN" https://registry-1.docker.io/v2/library/nginx/manifests/latest
```

## Logging Driver Errors

```text
Error: Failed to initialize logging driver: ...
```

Common drivers:

- `json-file` (default) — writes to `/var/lib/docker/containers/<id>/<id>-json.log`. **Default unlimited size — set `max-size`.**
- `journald` — pipes to systemd-journald.
- `syslog` — pipes to syslog.
- `fluentd` — sends to a Fluentd daemon over TCP.
- `awslogs` — CloudWatch Logs.
- `splunk` — Splunk HEC.
- `gcplogs` — Google Cloud Logging.
- `gelf` — Graylog.
- `local` — binary format, faster than json-file.
- `none` — discard.

```json
// /etc/docker/daemon.json — sane defaults
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "5",
    "compress": "true"
  }
}
```

```bash
# Per container override
docker run --log-driver=journald myimage
docker run --log-driver=json-file --log-opt max-size=10m --log-opt max-file=3 myimage

# Inspect
docker inspect --format='{{.HostConfig.LogConfig}}' web
```

The log-rotation gotcha: with no `max-size`, a chatty container can fill the disk. Set it cluster-wide via daemon.json and reboot.

## Common Gotchas

### COPY before WORKDIR

```dockerfile
# Broken — files end up in /
FROM alpine
COPY app /
WORKDIR /app
RUN ls           # /app is empty
```

```dockerfile
# Fixed
FROM alpine
WORKDIR /app
COPY app /app/
RUN ls           # /app contains your files
```

### ENV vs ARG

```dockerfile
# ARG is build-time only; not in final image env
ARG VERSION=1.0
RUN echo "$VERSION"        # works during build
# CMD ["sh", "-c", "echo $VERSION"]  # empty at runtime

# ENV persists into the image runtime
ENV VERSION=1.0
CMD ["sh", "-c", "echo $VERSION"]    # prints 1.0
```

### Shell form vs exec form

```dockerfile
# Shell form: runs through /bin/sh -c, sh becomes PID 1, signals lost
CMD node server.js

# Exec form: argv[0] is your binary, signals work
CMD ["node", "server.js"]
```

### Layering apt-get

```dockerfile
# Broken — apt-get update is in a separate layer; cache may keep stale lists
RUN apt-get update
RUN apt-get install -y curl
RUN apt-get clean
```

```dockerfile
# Fixed — single layer, cache flushed
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl ca-certificates \
 && rm -rf /var/lib/apt/lists/*
```

### Caching — package.json before npm install

```dockerfile
# Broken — every code change invalidates npm install
FROM node:20-alpine
WORKDIR /app
COPY . .
RUN npm ci
```

```dockerfile
# Fixed — package files first, then code
FROM node:20-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
```

### Missing .dockerignore

```text
.dockerignore
---
node_modules
.git
*.log
.env*
build/
dist/
.DS_Store
```

Without it `docker build .` ships the whole tree to the daemon. A 2 GiB context will silently waste minutes.

### --rm forgotten

```bash
# Without --rm, every `docker run` leaves a stopped container
docker run --rm -it alpine sh
docker container prune                # remove all stopped
```

### root user inside container

```dockerfile
# Default is root — capability + escape risk
FROM alpine
RUN adduser -D -u 1000 app
USER 1000
```

```bash
docker run --user 1000:1000 myimage           # at runtime
docker run --read-only --tmpfs /tmp myimage   # belt-and-braces
```

### Bind-mounting /var/run/docker.sock

```yaml
# Anti-pattern — gives the container root on the host
services:
  agent:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
```

If you must, use a socket proxy (`tecnativa/docker-socket-proxy`) that exposes only specific endpoints, and bind it on a private network.

### HEALTHCHECK in multi-stage final image

```dockerfile
# Broken — curl was in the build stage, missing in distroless final
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN go build -o /out/myapp

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/myapp /myapp
HEALTHCHECK CMD curl -f http://localhost:8080/healthz || exit 1   # no curl
ENTRYPOINT ["/myapp"]
```

```dockerfile
# Fixed — bake health into the binary
ENTRYPOINT ["/myapp"]
HEALTHCHECK CMD ["/myapp", "healthcheck"]
```

### Tag mutation — `:latest` shifting

`:latest` is a moving pointer. CI builds on Tuesday and Friday will not be reproducible. Pin by digest:

```bash
docker pull nginx:1.27.1
docker inspect --format='{{index .RepoDigests 0}}' nginx:1.27.1
# nginx@sha256:abcd...
docker run nginx@sha256:abcd...
```

```dockerfile
FROM nginx@sha256:abcd1234567890...   # immutable
```

### --net=host inheriting host network

```bash
# Container shares host's network ns. No port mapping needed; loud security risk.
docker run --network=host nginx
```

Host-mode skips Docker's userland proxy and iptables NAT — useful for performance — but the container can bind any host port and see all host traffic. Never combine with untrusted images.

## Diagnostic Commands

```bash
docker info                          # daemon overview: storage driver, runtimes, plugins, security
docker version                       # client + server version mismatch
docker system df                     # image/container/volume/build-cache disk use
docker system df -v                  # per-image breakdown
docker system prune                  # remove dangling
docker system prune -af --volumes    # NUKE: stopped containers, unused networks, all dangling images, build cache, anonymous volumes
docker events                        # live JSON event stream from daemon
docker events --since 1h --until now
docker logs -f --tail 100 web        # tail container logs
docker logs --since 10m web
docker exec -it web sh               # shell into running container
docker exec -u 0 -it web bash        # as root for debug
docker inspect web | jq              # full JSON state
docker inspect --format='{{.State.Pid}}' web
docker stats                          # live CPU/mem/net/io across containers
docker stats --no-stream
docker network inspect bridge | jq
docker top web                        # process list inside container
docker port web                       # port mappings
docker history nginx:1.27             # layer-by-layer of an image
docker buildx debug                   # buildkit-level debug (Buildx 0.13+)
trivy image myorg/web:v1              # vuln scan
docker scout cves myorg/web:v1        # Docker's built-in scanner
grype myorg/web:v1                    # Anchore Grype scanner
```

For deeper diagnosis use `nicolaka/netshoot` to share a network namespace:

```bash
docker run -it --rm --net=container:web nicolaka/netshoot
# Inside: tcpdump, iperf, dig, curl, mtr, ss, tshark, nmap.
```

## Performance Issues

### macOS file-share performance

- Use VirtioFS (Settings > General).
- Avoid bind-mounting `node_modules`, `vendor`, `target`, `.git`.
- Prefer named volumes for hot directories.

```yaml
services:
  app:
    volumes:
      - .:/app                         # source code, bind
      - /app/node_modules              # anonymous volume, fast
      - /app/.git                       # exclude .git
```

### overlay2 disk pressure

```bash
docker system df
docker image prune -af --filter "until=168h"   # images older than 7d
docker container prune -f
docker volume prune -af
docker builder prune -af --filter "until=168h"
```

### user-defined network throughput

User-defined bridges add NAT and a userland proxy. For workloads that move gigabytes between containers:

- Use the same user-defined network (no NAT between members).
- Or `--network=host` if security model permits.
- Or macvlan/ipvlan for bare-metal NIC passthrough.

```bash
docker network create -d macvlan \
  --subnet=192.168.1.0/24 --gateway=192.168.1.1 \
  -o parent=eth0 macvlan-lan
```

### --network=host escape hatch

Best raw network performance, but the container shares the host network ns. Acceptable for single-tenant boxes; never for shared/untrusted hosts.

## Idioms

- **Multi-stage builds** are the default. Builder stage installs toolchains, final stage copies only artifacts. Final image stays small.
- **Run as non-root** with `USER 1000` (or a random high UID). Pair with `--read-only`, `--cap-drop=ALL`, and only the caps you need.
- **Pin by digest in CI** so a re-run produces byte-identical artifacts.
- **Scan before push** (`trivy image`, `docker scout cves`, `grype`); fail the pipeline on high-severity CVEs.
- **Never bind-mount /var/run/docker.sock** to a public-facing container. If you need to talk to Docker from a container, use a vetted socket proxy and a private network.
- **Set log-opts** so chatty containers do not exhaust the disk.
- **Set a HEALTHCHECK** on every long-running service; rely on it in `depends_on`.
- **Prefer COPY over ADD** unless you need ADD's tar-extract or remote-URL behavior.
- **Use --init or tini** for any container that uses shell wrapping; you want PID 1 to be a real init.
- **Tag with both a moving and an immutable tag**: `myorg/web:v1.2.3` and `myorg/web@sha256:abcd...`; CI pulls the digest, humans look at the tag.
- **`.dockerignore` first** before any large project.
- **One process per container** — log to stdout/stderr, let the orchestrator restart you.
- **Treat the image as ephemeral**; persist data only in named volumes.

## See Also

- docker
- kubectl
- container-hardening
- troubleshooting/kubernetes-errors
- troubleshooting/linux-errors

## References

- Docker CLI reference — https://docs.docker.com/reference/cli/docker/
- Dockerfile reference — https://docs.docker.com/reference/dockerfile/
- Compose file reference — https://docs.docker.com/reference/compose-file/
- OCI runtime spec — https://github.com/opencontainers/runtime-spec
- OCI image spec — https://github.com/opencontainers/image-spec
- OCI distribution spec — https://github.com/opencontainers/distribution-spec
- BuildKit — https://github.com/moby/buildkit
- containerd — https://github.com/containerd/containerd
- runc — https://github.com/opencontainers/runc
- Docker Engine release notes — https://docs.docker.com/engine/release-notes/
- Docker Desktop release notes — https://docs.docker.com/desktop/release-notes/
- Rootless mode — https://docs.docker.com/engine/security/rootless/
- Docker Hub rate limits — https://docs.docker.com/docker-hub/download-rate-limit/
- Trivy — https://github.com/aquasecurity/trivy
- Grype — https://github.com/anchore/grype
- Docker Scout — https://docs.docker.com/scout/
- netshoot — https://github.com/nicolaka/netshoot
- tini — https://github.com/krallin/tini
- dumb-init — https://github.com/Yelp/dumb-init
