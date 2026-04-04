# crane (Container Registry CLI)

crane is a tool for interacting with remote container images and registries, supporting pull, push, copy, mutate, flatten, digest computation, manifest inspection, multi-arch image handling, layer append, and registry authentication without requiring a Docker daemon.

## Installation

### Install crane

```bash
# Install via Go
go install github.com/google/go-containerregistry/cmd/crane@latest

# Install via Homebrew
brew install crane

# Install from release tarball (Linux)
VERSION=$(curl -s https://api.github.com/repos/google/go-containerregistry/releases/latest | jq -r .tag_name)
curl -sL "https://github.com/google/go-containerregistry/releases/download/${VERSION}/go-containerregistry_Linux_x86_64.tar.gz" | tar xz crane
sudo mv crane /usr/local/bin/

# Verify installation
crane version
```

## Authentication

### Registry Login

```bash
# Login to Docker Hub
crane auth login index.docker.io -u username -p password

# Login to GitHub Container Registry
crane auth login ghcr.io -u username -p $(cat github-token.txt)

# Login to AWS ECR
crane auth login $(aws sts get-caller-identity --query Account --output text).dkr.ecr.us-east-1.amazonaws.com \
  -u AWS -p $(aws ecr get-login-password --region us-east-1)

# Login to Google Artifact Registry
crane auth login us-docker.pkg.dev -u oauth2accesstoken -p $(gcloud auth print-access-token)

# Get current auth config
crane auth get docker.io

# Login using stdin (scripting)
echo "$REGISTRY_PASSWORD" | crane auth login registry.example.com -u user --password-stdin
```

## Image Operations

### Pull and Push

```bash
# Pull image to tarball
crane pull alpine:3.19 alpine.tar

# Pull specific platform
crane pull --platform linux/arm64 nginx:latest nginx-arm64.tar

# Push tarball to registry
crane push myimage.tar registry.example.com/myimage:latest

# Copy image between registries (no local download)
crane copy docker.io/library/nginx:latest registry.example.com/nginx:latest

# Copy with all tags
crane copy --all-tags docker.io/library/alpine registry.example.com/alpine

# Tag an image remotely
crane tag registry.example.com/myapp:abc123 latest
crane tag registry.example.com/myapp:abc123 v1.2.3
```

### Inspect Images

```bash
# Get image digest
crane digest alpine:3.19

# Get image manifest
crane manifest alpine:3.19

# Get image manifest (pretty-printed)
crane manifest alpine:3.19 | jq .

# Get image config
crane config alpine:3.19 | jq .

# List tags for a repository
crane ls alpine
crane ls registry.example.com/myapp

# List tags with digest
crane ls --full-ref alpine

# Get image size (compressed)
crane manifest alpine:3.19 | jq '[.layers[].size] | add'
```

### Mutate Images

```bash
# Add labels to an image
crane mutate alpine:3.19 \
  --label org.opencontainers.image.source=https://github.com/org/repo \
  --label org.opencontainers.image.version=3.19 \
  -t registry.example.com/alpine:labeled

# Change entrypoint
crane mutate alpine:3.19 \
  --entrypoint /bin/sh \
  -t registry.example.com/alpine:custom

# Change command
crane mutate alpine:3.19 \
  --cmd "/app/start.sh" \
  -t registry.example.com/alpine:with-cmd

# Set environment variables
crane mutate alpine:3.19 \
  --env KEY=value \
  --env ANOTHER=thing \
  -t registry.example.com/alpine:env

# Change user
crane mutate alpine:3.19 \
  --user nobody \
  -t registry.example.com/alpine:nonroot

# Set working directory
crane mutate alpine:3.19 \
  --workdir /app \
  -t registry.example.com/alpine:workdir

# Add annotations to manifest
crane mutate alpine:3.19 \
  --annotation key=value \
  -t registry.example.com/alpine:annotated
```

### Flatten and Append

```bash
# Flatten image to single layer (reduces layers)
crane flatten alpine:3.19 -t registry.example.com/alpine:flat

# Append a layer from a tarball
crane append -b alpine:3.19 \
  -f app-layer.tar.gz \
  -t registry.example.com/myapp:latest

# Append with new entrypoint
crane append -b alpine:3.19 \
  -f myapp.tar.gz \
  -t registry.example.com/myapp:v1 \
  --set-entrypoint /app/myapp

# Create image from scratch with a layer
crane append -f rootfs.tar.gz \
  -t registry.example.com/scratch-app:latest
```

### Export and Rebase

```bash
# Export image filesystem as tarball
crane export alpine:3.19 - | tar tf -

# Export to a file
crane export alpine:3.19 alpine-rootfs.tar

# Rebase image onto new base
crane rebase \
  --original registry.example.com/myapp:old \
  --old_base ubuntu:22.04 \
  --new_base ubuntu:24.04 \
  --rebased registry.example.com/myapp:rebased
```

## Multi-Architecture Images

### Working with Manifests

```bash
# View manifest list (multi-arch)
crane manifest --platform all alpine:3.19 | jq .

# Get digest for specific platform
crane digest --platform linux/amd64 alpine:3.19
crane digest --platform linux/arm64 alpine:3.19

# Copy multi-arch image (preserves all platforms)
crane copy --all-tags alpine:3.19 registry.example.com/alpine:3.19

# Pull specific platform
crane pull --platform linux/arm/v7 alpine:3.19 alpine-armv7.tar
```

## Validation and Comparison

### Image Validation

```bash
# Validate image exists
crane validate --remote alpine:3.19

# Check if a tag exists
crane digest alpine:3.19 2>/dev/null && echo "exists" || echo "not found"

# Compare digests between registries
DIGEST1=$(crane digest docker.io/library/nginx:latest)
DIGEST2=$(crane digest registry.example.com/nginx:latest)
[ "$DIGEST1" = "$DIGEST2" ] && echo "identical" || echo "different"

# Diff two image configs
diff <(crane config alpine:3.18 | jq .) <(crane config alpine:3.19 | jq .)
```

## Scripting Examples

### Bulk Operations

```bash
# Mirror all tags of an image
for tag in $(crane ls docker.io/library/nginx); do
  crane copy "docker.io/library/nginx:${tag}" "registry.example.com/nginx:${tag}"
done

# Delete old tags (keep last 10)
REPO="registry.example.com/myapp"
crane ls "$REPO" | sort -V | head -n -10 | while read tag; do
  crane delete "${REPO}:${tag}"
done

# Find images over a size threshold (100MB)
crane manifest myimage:latest | jq '[.layers[].size] | add' | \
  awk '{ if ($1 > 100000000) print "Image exceeds 100MB: " $1/1000000 "MB" }'

# Get creation timestamp
crane config alpine:3.19 | jq -r '.created'
```

## Tips

- Use `crane copy` to mirror images between registries without pulling locally, saving bandwidth and disk
- Use `crane digest` in CI pipelines to verify image integrity after pushing
- Prefer `crane mutate` over rebuilding when only labels, env vars, or entrypoints need changing
- Use `crane flatten` to reduce layer count for images with many small layers, improving pull time
- Use `crane append` to add application binaries to a base image without a Dockerfile
- `crane export` pipes the entire filesystem to stdout, useful for inspecting image contents without Docker
- Use `--platform` flags consistently to avoid accidentally pulling the wrong architecture
- Combine `crane ls` with `crane delete` to implement tag retention policies in registries
- Use `crane rebase` to update base OS layers without rebuilding the entire image
- The `--insecure` flag works with HTTP registries for development, but never use it in production

## See Also

- skopeo, docker, buildah, podman, kaniko, oci

## References

- [crane GitHub Repository](https://github.com/google/go-containerregistry/tree/main/cmd/crane)
- [go-containerregistry Documentation](https://pkg.go.dev/github.com/google/go-containerregistry)
- [OCI Image Spec](https://github.com/opencontainers/image-spec)
- [crane Recipes](https://github.com/google/go-containerregistry/blob/main/cmd/crane/recipes.md)
