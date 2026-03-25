# Container Security (Docker and container hardening for production deployments)

> Security recommendations for hardening production container deployments.

## Image Security

### Image Scanning with Trivy

```bash
# Scan a local image for vulnerabilities
trivy image myapp:latest

# Scan with severity filter
trivy image --severity HIGH,CRITICAL myapp:latest

# Scan and fail CI if critical vulns found (exit code 1)
trivy image --exit-code 1 --severity CRITICAL myapp:latest

# Scan a tar archive
trivy image --input myapp.tar

# Scan with SBOM output
trivy image --format spdx-json --output sbom.json myapp:latest

# Ignore unfixed vulnerabilities
trivy image --ignore-unfixed myapp:latest

# Scan filesystem (useful in CI before building)
trivy fs --security-checks vuln,secret,config .
```

### Image Scanning with Grype

```bash
# Scan an image
grype myapp:latest

# Scan with severity threshold
grype myapp:latest --fail-on critical

# Scan a directory
grype dir:/path/to/project

# Scan an SBOM
grype sbom:./sbom.json

# Output in JSON for automation
grype myapp:latest -o json > results.json

# Generate SBOM with syft, then scan
syft myapp:latest -o spdx-json > sbom.json
grype sbom:sbom.json
```

### Dockerfile Best Practices

```dockerfile
# Pin base image with digest, not just tag
FROM golang:1.24-alpine@sha256:abc123... AS builder

# Run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Use multi-stage build to minimize attack surface
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app ./cmd/server

# Final stage: minimal image
FROM scratch
# Import CA certs for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app /app

USER appuser
ENTRYPOINT ["/app"]
```

```bash
# Lint Dockerfiles with hadolint
hadolint Dockerfile

# Check for secrets accidentally baked into images
trivy image --security-checks secret myapp:latest

# Use .dockerignore to exclude sensitive files
cat <<'EOF' > .dockerignore
.git
.env
*.key
*.pem
credentials/
EOF
```

## Runtime Hardening

### Rootless Containers

```bash
# Run container as non-root (override if Dockerfile doesn't set USER)
docker run --user 1000:1000 myapp:latest

# Rootless Docker daemon (run dockerd without root)
# Install rootless mode
dockerd-rootless-setuptool.sh install

# Verify rootless mode
docker info | grep -i rootless
# Expected: rootless: true
```

### Read-Only Filesystems

```bash
# Run with read-only root filesystem
docker run --read-only myapp:latest

# Allow specific writable paths via tmpfs
docker run --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=64m \
  --tmpfs /run:rw,noexec,nosuid,size=16m \
  myapp:latest

# Mount a volume for data that must persist
docker run --read-only \
  -v /data/app:/app/data:rw \
  --tmpfs /tmp \
  myapp:latest
```

### Capability Dropping

```bash
# Drop all capabilities, add only what's needed
docker run --cap-drop ALL \
  --cap-add NET_BIND_SERVICE \
  myapp:latest

# Common minimal capability sets by use case:
# Web server binding to port 80:  --cap-add NET_BIND_SERVICE
# Ping/ICMP:                      --cap-add NET_RAW
# Changing file ownership:        --cap-add CHOWN
# Setting file permissions:       --cap-add FOWNER

# List capabilities of a running container
docker exec <container> capsh --print

# View current capabilities (from inside container)
cat /proc/1/status | grep Cap
```

### Seccomp Profiles

```bash
# Run with default Docker seccomp profile (blocks ~44 syscalls)
docker run --security-opt seccomp=default myapp:latest

# Run with custom seccomp profile
docker run --security-opt seccomp=custom-seccomp.json myapp:latest

# Generate a seccomp profile from observed syscalls (using OCI tooling)
# Record syscalls with strace, then build profile
strace -f -o /tmp/syscalls.log ./app
# Parse output to build allowlist

# Disable seccomp (NOT recommended in production)
docker run --security-opt seccomp=unconfined myapp:latest
```

```json
// Example custom seccomp profile (custom-seccomp.json)
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "architectures": ["SCMP_ARCH_X86_64"],
  "syscalls": [
    {
      "names": ["read", "write", "open", "close", "stat", "fstat",
                "mmap", "mprotect", "munmap", "brk", "access",
                "getpid", "socket", "connect", "accept", "bind",
                "listen", "exit_group", "futex", "epoll_wait"],
      "action": "SCMP_ACT_ALLOW"
    }
  ]
}
```

### User Namespaces

```bash
# Enable user namespace remapping in Docker daemon
# /etc/docker/daemon.json
cat <<'EOF' > /etc/docker/daemon.json
{
  "userns-remap": "default"
}
EOF

# Restart Docker daemon
sudo systemctl restart docker

# Verify remapping — root in container maps to unprivileged host UID
docker run --rm alpine id
# uid=0(root) inside, but mapped to high UID on host

# Check subordinate UID/GID mappings
cat /etc/subuid
cat /etc/subgid

# Run with explicit user namespace
docker run --userns=host myapp:latest  # disable (use host namespace)
```

### No New Privileges

```bash
# Prevent privilege escalation via setuid/setgid binaries
docker run --security-opt no-new-privileges:true myapp:latest

# Combine with capability dropping for defense in depth
docker run \
  --cap-drop ALL \
  --security-opt no-new-privileges:true \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid \
  --user 1000:1000 \
  myapp:latest
```

## Docker Bench for Security

```bash
# Run Docker Bench (CIS Docker Benchmark)
docker run --rm --net host --pid host \
  --userns host --cap-add audit_control \
  -e DOCKER_CONTENT_TRUST=$DOCKER_CONTENT_TRUST \
  -v /etc:/etc:ro \
  -v /usr/bin/containerd:/usr/bin/containerd:ro \
  -v /usr/bin/runc:/usr/bin/runc:ro \
  -v /usr/lib/systemd:/usr/lib/systemd:ro \
  -v /var/lib:/var/lib:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  docker/docker-bench-security

# Review specific sections
# 1 - Host Configuration
# 2 - Docker daemon configuration
# 3 - Docker daemon configuration files
# 4 - Container Images and Build File
# 5 - Container Runtime
# 6 - Docker Security Operations
```

## Runtime Security with Falco

```bash
# Install Falco
curl -fsSL https://falco.org/repo/falcosecurity-packages.asc | \
  sudo gpg --dearmor -o /usr/share/keyrings/falco-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/falco-archive-keyring.gpg] \
  https://download.falco.org/packages/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/falcosecurity.list
sudo apt-get update && sudo apt-get install -y falco

# Start Falco
sudo systemctl start falco

# View Falco alerts
sudo journalctl -u falco -f

# Custom Falco rule: detect shell spawned in container
cat <<'EOF' >> /etc/falco/falco_rules.local.yaml
- rule: Shell in Container
  desc: Detect shell execution in a container
  condition: >
    spawned_process and container and
    proc.name in (bash, sh, zsh, dash, ksh)
  output: >
    Shell spawned in container
    (user=%user.name container=%container.name shell=%proc.name
     parent=%proc.pname cmdline=%proc.cmdline)
  priority: WARNING
  tags: [container, shell]
EOF

# Reload Falco rules
sudo kill -SIGHUP $(pidof falco)
```

## Registry Security

```bash
# Enable Docker Content Trust (image signing)
export DOCKER_CONTENT_TRUST=1

# Sign and push an image
docker trust sign myregistry.com/myapp:latest

# Inspect image signatures
docker trust inspect myregistry.com/myapp:latest

# Run a private registry with TLS
docker run -d -p 5000:5000 \
  --name registry \
  -v /certs:/certs \
  -v /registry-data:/var/lib/registry \
  -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt \
  -e REGISTRY_HTTP_TLS_KEY=/certs/domain.key \
  registry:2

# Configure registry authentication (htpasswd)
docker run --rm --entrypoint htpasswd \
  httpd:2 -Bbn admin secretpassword > /auth/htpasswd

docker run -d -p 5000:5000 \
  --name registry \
  -v /auth:/auth \
  -e REGISTRY_AUTH=htpasswd \
  -e REGISTRY_AUTH_HTPASSWD_REALM="Registry Realm" \
  -e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
  registry:2
```

## Secrets Management

```bash
# Use Docker secrets (Swarm mode)
echo "db_password_here" | docker secret create db_password -
docker service create --secret db_password myapp:latest
# Secret available at /run/secrets/db_password inside container

# Use BuildKit secrets for build-time secrets (never baked into layers)
docker buildx build --secret id=npmrc,src=$HOME/.npmrc .

# In Dockerfile:
# RUN --mount=type=secret,id=npmrc,target=/root/.npmrc npm install

# Never pass secrets via ENV in production
# BAD:  docker run -e DB_PASSWORD=secret myapp
# GOOD: docker run --mount type=tmpfs,destination=/secrets myapp
#        (inject secrets via orchestrator or vault)

# Use environment files with restricted permissions
chmod 600 .env
docker run --env-file .env myapp:latest
```

## Network Policies

```bash
# Create an isolated network
docker network create --driver bridge --internal isolated_net

# Run container with no external network access
docker run --network isolated_net myapp:latest

# Restrict inter-container communication (daemon level)
# /etc/docker/daemon.json
# { "icc": false }

# Use network aliases for service discovery
docker network create app_net
docker run --network app_net --network-alias db postgres:16
docker run --network app_net --network-alias app myapp:latest

# Limit container network bandwidth (tc-based)
docker run --cap-add NET_ADMIN myapp:latest \
  sh -c "tc qdisc add dev eth0 root tbf rate 10mbit burst 32kbit latency 400ms"
```

## Resource Limits

```bash
# Limit memory and CPU
docker run \
  --memory 512m \
  --memory-swap 512m \
  --cpus 1.0 \
  --pids-limit 100 \
  myapp:latest

# Prevent fork bombs
docker run --pids-limit 50 myapp:latest

# Set ulimits
docker run --ulimit nofile=1024:1024 --ulimit nproc=64:64 myapp:latest
```

## Tips

- Always scan images in CI before pushing to registries; fail the pipeline on CRITICAL findings.
- Use `scratch` or `distroless` base images to minimize attack surface; fewer binaries means fewer exploits.
- Never run containers with `--privileged` in production; it grants nearly full host access.
- Combine `--cap-drop ALL`, `--read-only`, `--no-new-privileges`, and a non-root user for defense in depth.
- Pin image versions with digests (`@sha256:...`), not mutable tags like `latest`.
- Rotate secrets regularly and never bake them into image layers.
- Enable Docker Content Trust to verify image provenance before pulling.
- Use Falco or a similar runtime security tool to detect unexpected behavior in running containers.
- Audit your Docker daemon configuration regularly with Docker Bench for Security.

## References

- [Docker Security Documentation](https://docs.docker.com/engine/security/)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [Trivy - Aqua Security](https://github.com/aquasecurity/trivy)
- [Grype - Anchore](https://github.com/anchore/grype)
- [Falco - Runtime Security](https://github.com/falcosecurity/falco)
- [Docker Bench for Security](https://github.com/docker/docker-bench-security)
- [NIST SP 800-190 - Application Container Security Guide](https://csrc.nist.gov/publications/detail/sp/800-190/final)
- [Dockerfile Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Seccomp Security Profiles for Docker](https://docs.docker.com/engine/security/seccomp/)
- [Hadolint - Dockerfile Linter](https://github.com/hadolint/hadolint)
