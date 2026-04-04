# Skopeo (Container Image Management)

Command-line tool for inspecting, copying, and managing container images across registries without requiring a daemon or pulling full images to local storage.

## Image Inspection

### Inspect Remote Images

```bash
# Inspect image manifest and config (no download)
skopeo inspect docker://docker.io/library/nginx:latest
skopeo inspect docker://registry.example.com/myapp:1.2.3

# Raw manifest (JSON)
skopeo inspect --raw docker://nginx:latest
skopeo inspect --raw docker://nginx:latest | jq .

# Show image configuration (entrypoint, env, labels)
skopeo inspect --config docker://nginx:latest | jq .

# Inspect specific platform in multi-arch image
skopeo inspect --override-arch arm64 docker://golang:1.24

# Inspect with authentication
skopeo inspect --creds user:password docker://private-registry.com/app:latest

# Format specific fields
skopeo inspect docker://nginx:latest | jq '.Digest, .RepoTags'

# Show all tags for a repository
skopeo list-tags docker://docker.io/library/nginx
skopeo list-tags docker://registry.example.com/myapp
```

### Inspect Local Images

```bash
# Inspect image in local containers storage (podman/buildah)
skopeo inspect containers-storage:localhost/myapp:latest

# Inspect OCI layout on disk
skopeo inspect oci:./myapp-oci:latest

# Inspect docker archive (tar)
skopeo inspect docker-archive:./myapp.tar

# Inspect OCI archive
skopeo inspect oci-archive:./myapp-oci.tar
```

## Copying Images

### Between Registries

```bash
# Copy between registries (no local storage needed)
skopeo copy docker://source-registry.com/myapp:1.0 \
            docker://dest-registry.com/myapp:1.0

# Copy with different tag
skopeo copy docker://source-registry.com/myapp:1.0 \
            docker://dest-registry.com/myapp:latest

# Copy all tags from a repository
skopeo copy --all docker://source-registry.com/myapp \
                  docker://dest-registry.com/myapp

# Copy with authentication (different creds per registry)
skopeo copy --src-creds user1:pass1 --dest-creds user2:pass2 \
  docker://source.com/app:v1 docker://dest.com/app:v1

# Copy with TLS options
skopeo copy --src-tls-verify=false --dest-tls-verify=true \
  docker://insecure-registry:5000/app:v1 docker://secure.com/app:v1
```

### Between Transports

```bash
# Registry to local docker-archive (tar file)
skopeo copy docker://nginx:latest docker-archive:/tmp/nginx.tar:nginx:latest

# Registry to OCI layout directory
skopeo copy docker://nginx:latest oci:./nginx-oci:latest

# Registry to OCI archive (tar)
skopeo copy docker://nginx:latest oci-archive:/tmp/nginx-oci.tar:latest

# Registry to directory (raw blobs)
skopeo copy docker://nginx:latest dir:/tmp/nginx-dir

# Docker archive to registry
skopeo copy docker-archive:/tmp/nginx.tar docker://registry.com/nginx:latest

# OCI layout to registry
skopeo copy oci:./nginx-oci:latest docker://registry.com/nginx:latest

# Local containers storage to registry
skopeo copy containers-storage:localhost/myapp:latest \
            docker://registry.com/myapp:latest

# Registry to local containers storage
skopeo copy docker://nginx:latest containers-storage:nginx:latest
```

### Multi-Architecture Copies

```bash
# Copy entire manifest list (all platforms)
skopeo copy --all docker://nginx:latest docker://mirror.com/nginx:latest

# Copy specific platform only
skopeo copy --override-arch amd64 --override-os linux \
  docker://nginx:latest docker://mirror.com/nginx:amd64

# Copy with manifest format conversion
skopeo copy --format v2s2 docker://nginx:latest docker://dest.com/nginx:latest
skopeo copy --format oci docker://nginx:latest docker://dest.com/nginx:latest
```

## Manifest Operations

### Manifest Inspection

```bash
# View raw manifest
skopeo inspect --raw docker://nginx:latest | jq .

# View manifest list (multi-arch)
skopeo inspect --raw docker://nginx:latest | jq '.manifests[]'

# Check manifest media type
skopeo inspect --raw docker://nginx:latest | jq '.mediaType'
# "application/vnd.docker.distribution.manifest.list.v2+json"  (manifest list)
# "application/vnd.docker.distribution.manifest.v2+json"       (single)
# "application/vnd.oci.image.index.v1+json"                    (OCI index)
# "application/vnd.oci.image.manifest.v1+json"                 (OCI manifest)
```

### Digest Operations

```bash
# Get image digest
skopeo inspect docker://nginx:latest | jq -r '.Digest'
# sha256:abc123...

# Copy by digest (immutable reference)
skopeo copy docker://nginx@sha256:abc123... \
            docker://mirror.com/nginx@sha256:abc123...

# Compare digests across registries
DIGEST_SRC=$(skopeo inspect docker://source.com/app:v1 | jq -r '.Digest')
DIGEST_DST=$(skopeo inspect docker://dest.com/app:v1 | jq -r '.Digest')
[ "$DIGEST_SRC" = "$DIGEST_DST" ] && echo "Images match"
```

## Authentication

### Login and Credentials

```bash
# Login to registry
skopeo login registry.example.com
skopeo login -u user -p password registry.example.com

# Login with stdin password
echo "$REGISTRY_PASSWORD" | skopeo login -u user --password-stdin registry.com

# Specify auth file location
skopeo login --authfile /tmp/auth.json registry.example.com

# Use existing Docker config
skopeo inspect --authfile ~/.docker/config.json docker://private.com/app:v1

# Logout
skopeo logout registry.example.com
skopeo logout --all

# Auth file format (~/.local/share/containers/auth.json)
# {
#   "auths": {
#     "registry.example.com": {
#       "auth": "base64(user:password)"
#     }
#   }
# }
```

## Synchronization

### Mirror Registries

```bash
# Sync entire repository
skopeo sync --src docker --dest docker \
  source-registry.com/myapp dest-registry.com/

# Sync from YAML config
cat > sync.yaml <<EOF
source-registry.com:
  images:
    nginx:
      - "1.25"
      - "1.26"
      - "latest"
    alpine:
      - "3.19"
      - "3.20"
EOF
skopeo sync --src yaml --dest docker sync.yaml dest-registry.com/

# Sync to local directory
skopeo sync --src docker --dest dir source-registry.com/myapp /tmp/mirror/

# Sync from local directory to registry
skopeo sync --src dir --dest docker /tmp/mirror/ dest-registry.com/

# Dry run (show what would be copied)
skopeo sync --src docker --dest docker --dry-run \
  source-registry.com/myapp dest-registry.com/
```

## Image Deletion

### Delete from Registry

```bash
# Delete image tag from registry (requires registry support)
skopeo delete docker://registry.example.com/myapp:old-tag

# Delete by digest
skopeo delete docker://registry.example.com/myapp@sha256:abc123...

# Delete with credentials
skopeo delete --creds user:password docker://registry.com/myapp:v1

# Note: Docker Hub does not support deletion via API
# Most private registries (Harbor, GitLab, Quay) do support it
```

## Transport Reference

### Supported Transports

```
docker://          # Docker registry API v2
docker-archive:    # Docker tar archive (docker save format)
oci:               # OCI image layout directory
oci-archive:       # OCI tar archive
dir:               # Raw blobs in a directory
containers-storage: # Local containers/storage (podman, buildah)

# Format: transport:reference
# docker://registry.com/repo:tag
# docker-archive:/path/to/file.tar[:docker-reference]
# oci:/path/to/layout:tag
# oci-archive:/path/to/file.tar:tag
# dir:/path/to/directory
# containers-storage:[storage-specifier]image-reference
```

## Tips

- Skopeo never requires pulling a full image to local storage; it operates on manifests and blobs directly
- Use `skopeo inspect` to check image details before pulling, saving bandwidth and time
- Combine skopeo with buildah for a fully daemon-free container workflow (build + push + inspect)
- The `--all` flag on copy operations preserves manifest lists for multi-architecture support
- Use `skopeo sync` with a YAML config for maintaining air-gapped registry mirrors
- Registry deletion support varies; test with `skopeo delete` before relying on it in automation
- The `containers-storage` transport interoperates with podman and buildah local storage
- Use digest-based references (`@sha256:...`) for immutable, reproducible image references
- Skopeo shares authentication files with podman and buildah (`auth.json`)
- The `dir:` transport is useful for inspecting individual layers and blobs on disk
- Use `--override-arch` and `--override-os` to handle specific platform images from multi-arch tags

## See Also

buildah, podman, docker, oci, container-security

## References

- [Skopeo GitHub Repository](https://github.com/containers/skopeo)
- [Skopeo Documentation](https://github.com/containers/skopeo/tree/main/docs)
- [containers-transports(5)](https://github.com/containers/image/blob/main/docs/containers-transports.5.md)
- [Red Hat — Skopeo Guide](https://www.redhat.com/en/topics/containers/what-is-skopeo)
- [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec)
