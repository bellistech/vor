# kaniko (Daemonless Container Image Builder)

kaniko builds container images from Dockerfiles inside a container or Kubernetes pod without requiring a Docker daemon or privileged access, supporting layer caching to remote registries, multiple build contexts (local, S3, GCS, Git), and reproducible builds for CI/CD pipelines.

## Basic Usage

### Kaniko Executor

```bash
# Build from local context and push to registry
docker run \
  -v $(pwd):/workspace \
  -v ~/.docker/config.json:/kaniko/.docker/config.json:ro \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --dockerfile /workspace/Dockerfile \
  --destination registry.example.com/myapp:latest

# Build with multiple tags
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --destination registry.example.com/myapp:latest \
  --destination registry.example.com/myapp:v1.2.3

# Build without pushing (tarball output)
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --no-push \
  --tar-path /workspace/image.tar
```

## Build Context Sources

### Local Context

```bash
# Local directory (most common in CI)
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context dir:///workspace \
  --destination registry.example.com/myapp:latest
```

### Git Context

```bash
# Build from Git repository
docker run \
  gcr.io/kaniko-project/executor:latest \
  --context git://github.com/org/repo.git#refs/heads/main \
  --destination registry.example.com/myapp:latest

# Build from specific branch/tag
docker run \
  gcr.io/kaniko-project/executor:latest \
  --context "git://github.com/org/repo.git#refs/tags/v1.0.0" \
  --git-subpath /path/in/repo \
  --destination registry.example.com/myapp:v1.0.0
```

### S3 Context

```bash
# Build from S3 bucket
docker run \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  gcr.io/kaniko-project/executor:latest \
  --context s3://my-bucket/build-context.tar.gz \
  --destination registry.example.com/myapp:latest
```

### GCS Context

```bash
# Build from Google Cloud Storage
docker run \
  -v /path/to/credentials.json:/secret/credentials.json \
  -e GOOGLE_APPLICATION_CREDENTIALS=/secret/credentials.json \
  gcr.io/kaniko-project/executor:latest \
  --context gs://my-bucket/build-context.tar.gz \
  --destination registry.example.com/myapp:latest
```

## Caching

### Layer Cache Configuration

```bash
# Enable remote layer caching
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --destination registry.example.com/myapp:latest \
  --cache=true \
  --cache-repo registry.example.com/myapp/cache

# Cache with TTL
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --destination registry.example.com/myapp:latest \
  --cache=true \
  --cache-repo registry.example.com/myapp/cache \
  --cache-ttl 168h

# Warm the cache (pre-populate)
docker run \
  gcr.io/kaniko-project/warmer:latest \
  --cache-dir=/cache \
  --image=golang:1.22 \
  --image=alpine:3.19
```

### Cache Copy Optimization

```bash
# Use --cache-copy-layers for more aggressive caching
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --destination registry.example.com/myapp:latest \
  --cache=true \
  --cache-repo registry.example.com/myapp/cache \
  --cache-copy-layers
```

## Multi-Stage Builds

### Multi-Stage Dockerfile

```dockerfile
# Example multi-stage Dockerfile for kaniko
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/server /usr/local/bin/server
ENTRYPOINT ["server"]
```

```bash
# Build multi-stage with target
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --destination registry.example.com/myapp:latest \
  --target builder \
  --cache=true \
  --cache-repo registry.example.com/myapp/cache
```

## Reproducible Builds

### Reproducibility Flags

```bash
# Build with reproducible timestamps
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --destination registry.example.com/myapp:latest \
  --reproducible \
  --snapshot-mode=redo

# Snapshot modes:
# --snapshot-mode=full   - full filesystem scan (slow, accurate)
# --snapshot-mode=redo   - only scan changed files (faster)
# --snapshot-mode=time   - use modification time (fastest, less accurate)
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Build with kaniko
  uses: aevea/action-kaniko@master
  with:
    image: myapp
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}
    registry: ghcr.io
    tag: ${{ github.sha }}
    cache: true
    cache_registry: ghcr.io/${{ github.repository }}/cache
```

### GitLab CI

```yaml
build:
  stage: build
  image:
    name: gcr.io/kaniko-project/executor:debug
    entrypoint: [""]
  script:
    - mkdir -p /kaniko/.docker
    - echo "{\"auths\":{\"${CI_REGISTRY}\":{\"auth\":\"$(echo -n ${CI_REGISTRY_USER}:${CI_REGISTRY_PASSWORD} | base64)\"}}}" > /kaniko/.docker/config.json
    - /kaniko/executor
      --context "${CI_PROJECT_DIR}"
      --dockerfile "${CI_PROJECT_DIR}/Dockerfile"
      --destination "${CI_REGISTRY_IMAGE}:${CI_COMMIT_TAG}"
      --cache=true
      --cache-repo "${CI_REGISTRY_IMAGE}/cache"
```

### Tekton Task

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: kaniko-build
spec:
  params:
    - name: IMAGE
      description: Image reference to push
  workspaces:
    - name: source
    - name: docker-config
  steps:
    - name: build
      image: gcr.io/kaniko-project/executor:latest
      args:
        - --context=$(workspaces.source.path)
        - --destination=$(params.IMAGE)
        - --cache=true
      env:
        - name: DOCKER_CONFIG
          value: $(workspaces.docker-config.path)
```

### Kubernetes Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kaniko-build
spec:
  containers:
    - name: kaniko
      image: gcr.io/kaniko-project/executor:latest
      args:
        - --context=git://github.com/org/repo.git
        - --destination=registry.example.com/myapp:latest
        - --cache=true
        - --cache-repo=registry.example.com/myapp/cache
      volumeMounts:
        - name: docker-config
          mountPath: /kaniko/.docker/
  volumes:
    - name: docker-config
      secret:
        secretName: registry-credentials
        items:
          - key: .dockerconfigjson
            path: config.json
  restartPolicy: Never
```

## Advanced Options

### Build Arguments and Configuration

```bash
# Pass build arguments
docker run \
  -v $(pwd):/workspace \
  gcr.io/kaniko-project/executor:latest \
  --context /workspace \
  --destination registry.example.com/myapp:latest \
  --build-arg VERSION=1.2.3 \
  --build-arg COMMIT_SHA=abc123

# Use custom Dockerfile location
--dockerfile /workspace/docker/Dockerfile.prod

# Set label
--label org.opencontainers.image.source=https://github.com/org/repo

# Ignore paths from build context
--ignore-path /workspace/.git

# Skip TLS verification (development only)
--skip-tls-verify
--insecure

# Registry mirror
--registry-mirror mirror.gcr.io
```

## Tips

- Always use `--cache=true` with `--cache-repo` in CI pipelines to dramatically reduce build times
- Use the `debug` tag (`executor:debug`) in CI for shell access when troubleshooting build failures
- Use `--reproducible` for deterministic builds that produce the same digest for identical content
- Order Dockerfile instructions from least-changed to most-changed to maximize cache hit rates
- Use `--snapshot-mode=redo` for a good balance between speed and accuracy in layer detection
- Use Git context (`git://`) to avoid needing to mount source code in non-Docker CI environments
- Set `--cache-ttl` to match your base image update frequency to avoid stale cached layers
- For multi-stage builds, kaniko automatically caches intermediate stages when `--cache=true` is set
- Use `--single-snapshot` for Dockerfiles with many RUN commands to reduce snapshot overhead
- Mount registry credentials as Kubernetes secrets rather than baking them into pod specs

## See Also

- docker, buildah, crane, skopeo, tekton, oci

## References

- [kaniko GitHub Repository](https://github.com/GoogleContainerTools/kaniko)
- [kaniko Build Contexts](https://github.com/GoogleContainerTools/kaniko#kaniko-build-contexts)
- [kaniko Caching](https://github.com/GoogleContainerTools/kaniko#caching)
- [kaniko in CI/CD](https://github.com/GoogleContainerTools/kaniko#running-kaniko-in-a-kubernetes-cluster)
