# Buildah (Daemonless Container Image Builder)

OCI-compliant container image builder that operates without a daemon, supporting both Dockerfile-based and shell-scripted builds with rootless operation and fine-grained layer control.

## Basic Operations

### Build from Dockerfile

```bash
# Build image from Dockerfile (docker-compatible)
buildah bud -t myapp:latest .
buildah bud -t myapp:latest -f Dockerfile.prod .

# Build with build args
buildah bud --build-arg VERSION=1.2.3 -t myapp:1.2.3 .

# Multi-stage build
buildah bud --target builder -t myapp-builder:latest .

# Build with specific format
buildah bud --format docker -t myapp:latest .   # Docker format
buildah bud --format oci -t myapp:latest .       # OCI format (default)

# Build with layers (cache)
buildah bud --layers -t myapp:latest .

# Squash all layers into one
buildah bud --squash -t myapp:latest .

# No cache
buildah bud --no-cache -t myapp:latest .
```

### Native Buildah Commands (Scriptable)

```bash
# Create a working container from base image
container=$(buildah from alpine:3.19)

# Run commands in the container
buildah run $container -- apk add --no-cache curl jq

# Copy files into the container
buildah copy $container ./app /usr/local/bin/app
buildah copy $container ./config/ /etc/myapp/

# Set working directory
buildah config --workingdir /app $container

# Set entrypoint and command
buildah config --entrypoint '["/usr/local/bin/app"]' $container
buildah config --cmd '["--config", "/etc/myapp/config.yaml"]' $container

# Set environment variables
buildah config --env APP_ENV=production $container
buildah config --env APP_PORT=8080 $container

# Set labels
buildah config --label maintainer="team@example.com" $container
buildah config --label version="1.2.3" $container

# Set exposed ports
buildah config --port 8080 $container

# Set user
buildah config --user 1000:1000 $container

# Set volumes
buildah config --volume /data $container

# Commit to image
buildah commit $container myapp:latest

# Remove working container
buildah rm $container
```

### Complete Build Script

```bash
#!/bin/bash
set -euo pipefail

# Multi-stage build using native commands
# Stage 1: Build
builder=$(buildah from golang:1.24-alpine)
buildah run $builder -- apk add --no-cache git ca-certificates
buildah copy $builder . /src
buildah config --workingdir /src $builder
buildah run $builder -- go build -trimpath -ldflags="-s -w" -o /app ./cmd/server
buildah run $builder -- go test ./... -count=1 -race

# Stage 2: Runtime
runtime=$(buildah from gcr.io/distroless/static-debian12:nonroot)
buildah copy --from=$builder $runtime /app /app
buildah copy --from=$builder $runtime /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
buildah config --entrypoint '["/app"]' $runtime
buildah config --port 8080 $runtime
buildah config --user 65534:65534 $runtime
buildah config --label org.opencontainers.image.source="https://github.com/org/repo" $runtime

# Commit final image
buildah commit --squash $runtime myapp:latest

# Cleanup
buildah rm $builder $runtime
```

## Container and Image Management

### Working Containers

```bash
# List working containers
buildah containers
buildah containers --json

# Inspect container
buildah inspect $container
buildah inspect --type container $container

# Mount container filesystem for direct manipulation
mountpoint=$(buildah mount $container)
echo "Filesystem at: $mountpoint"
# Direct file manipulation
cp myfile "$mountpoint/usr/local/bin/"
ls "$mountpoint/etc/"

# Unmount
buildah unmount $container

# Remove container
buildah rm $container
buildah rm --all   # Remove all working containers
```

### Image Management

```bash
# List images
buildah images

# Inspect image
buildah inspect --type image myapp:latest

# Tag image
buildah tag myapp:latest registry.example.com/myapp:latest

# Push image
buildah push myapp:latest docker://registry.example.com/myapp:latest
buildah push myapp:latest docker-archive:/tmp/myapp.tar
buildah push myapp:latest oci-archive:/tmp/myapp-oci.tar
buildah push myapp:latest dir:/tmp/myapp-dir

# Remove image
buildah rmi myapp:latest
buildah rmi --all          # Remove all images
buildah rmi --prune        # Remove dangling images

# Pull image
buildah pull alpine:3.19
buildah pull docker://docker.io/library/nginx:latest
```

## Rootless Builds

### Setup

```bash
# Check user namespace support
cat /proc/sys/user/max_user_namespaces   # Should be > 0

# Configure subordinate UID/GID mappings
grep $USER /etc/subuid /etc/subgid
# user:100000:65536

# Run buildah as non-root user
buildah bud -t myapp:latest .

# Storage location for rootless
# ~/.local/share/containers/storage/

# Rootless configuration
cat ~/.config/containers/storage.conf
# [storage]
# driver = "overlay"
# [storage.options.overlay]
# mount_program = "/usr/bin/fuse-overlayfs"
```

### Rootless Considerations

```bash
# Some operations require additional setup for rootless:
# - Binding to ports < 1024 needs net.ipv4.ip_unprivileged_port_start=0
# - Overlay filesystem needs fuse-overlayfs
# - Some package managers need --isolation chroot

# Force chroot isolation (no user namespaces)
buildah bud --isolation chroot -t myapp:latest .

# Check available isolation modes
buildah info | grep -i isolation
```

## Registry Authentication

```bash
# Login to registry
buildah login registry.example.com
buildah login -u user -p password registry.example.com
buildah login --get-login registry.example.com

# Auth file location
# ~/.local/share/containers/auth.json (rootless)
# /run/containers/0/auth.json (rootful)

# Push with explicit credentials
buildah push --creds user:password myapp:latest \
  docker://registry.example.com/myapp:latest

# Push with TLS options
buildah push --tls-verify=false myapp:latest \
  docker://insecure-registry.local:5000/myapp:latest

# Logout
buildah logout registry.example.com
buildah logout --all
```

## Advanced Features

### Build with Secrets

```bash
# Mount secret during build (not stored in image)
buildah bud --secret id=mysecret,src=./secret.txt -t myapp:latest .

# In Dockerfile:
# RUN --mount=type=secret,id=mysecret cat /run/secrets/mysecret
```

### Custom Build Context

```bash
# Add content from URL
buildah copy $container https://example.com/file.tar.gz /tmp/

# Add from another image
buildah copy --from=docker.io/library/nginx:latest $container \
  /usr/share/nginx/html /var/www/html

# Add with chown
buildah copy --chown 1000:1000 $container ./app /app
```

### Manifest Lists (Multi-Arch)

```bash
# Create manifest list
buildah manifest create myapp:latest

# Build and add platform-specific images
buildah bud --platform linux/amd64 --manifest myapp:latest .
buildah bud --platform linux/arm64 --manifest myapp:latest .

# Inspect manifest
buildah manifest inspect myapp:latest

# Push manifest list
buildah manifest push --all myapp:latest \
  docker://registry.example.com/myapp:latest
```

## Tips

- Buildah shares storage with Podman; images built with buildah are immediately available to podman
- Use native buildah commands instead of Dockerfiles for builds that need shell logic or conditional steps
- Mount the container filesystem (`buildah mount`) for direct file manipulation without running commands
- Rootless builds are production-ready and should be the default for CI/CD pipelines
- Use `--squash` to collapse all layers into one, reducing image size and hiding intermediate steps
- The `--layers` flag enables layer caching similar to Docker builds for faster rebuilds
- Buildah supports both Docker and OCI image formats; prefer OCI for new projects
- Use `buildah bud --isolation chroot` when user namespaces are unavailable
- Build scripts with native commands are more debuggable than Dockerfiles (standard bash error handling)
- Always clean up working containers with `buildah rm` after committing to avoid storage leaks
- Combine buildah with skopeo for a complete daemon-free container workflow

## See Also

podman, skopeo, docker, oci, containerd, container-security

## References

- [Buildah Documentation](https://buildah.io/)
- [Buildah GitHub Repository](https://github.com/containers/buildah)
- [Buildah Tutorials](https://github.com/containers/buildah/tree/main/docs/tutorials)
- [Red Hat — Building Container Images with Buildah](https://www.redhat.com/en/topics/containers/what-is-buildah)
- [OCI Image Specification](https://github.com/opencontainers/image-spec)
