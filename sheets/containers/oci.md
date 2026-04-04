# OCI (Open Container Initiative)

Industry standards for container runtimes, images, and distribution, defining the config.json runtime specification, the image layer/manifest format, and the registry distribution API.

## Runtime Specification

### config.json Structure

```json
{
    "ociVersion": "1.1.0",
    "process": {
        "terminal": false,
        "user": { "uid": 1000, "gid": 1000 },
        "args": ["/usr/local/bin/app", "--config", "/etc/app.yaml"],
        "env": [
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin",
            "APP_ENV=production"
        ],
        "cwd": "/app",
        "capabilities": {
            "bounding": ["CAP_NET_BIND_SERVICE"],
            "effective": ["CAP_NET_BIND_SERVICE"],
            "permitted": ["CAP_NET_BIND_SERVICE"],
            "ambient": ["CAP_NET_BIND_SERVICE"]
        },
        "rlimits": [
            { "type": "RLIMIT_NOFILE", "hard": 65536, "soft": 65536 }
        ],
        "noNewPrivileges": true
    },
    "root": {
        "path": "rootfs",
        "readonly": true
    },
    "hostname": "app-container",
    "mounts": [
        {
            "destination": "/proc",
            "type": "proc",
            "source": "proc"
        },
        {
            "destination": "/dev",
            "type": "tmpfs",
            "source": "tmpfs",
            "options": ["nosuid", "strictatime", "mode=755", "size=65536k"]
        },
        {
            "destination": "/data",
            "type": "bind",
            "source": "/var/lib/app/data",
            "options": ["rbind", "rw"]
        }
    ]
}
```

### Linux Namespaces

```json
{
    "linux": {
        "namespaces": [
            { "type": "pid" },
            { "type": "network", "path": "/var/run/netns/app-ns" },
            { "type": "mount" },
            { "type": "ipc" },
            { "type": "uts" },
            { "type": "user" },
            { "type": "cgroup" }
        ],
        "uidMappings": [
            { "containerID": 0, "hostID": 1000, "size": 65536 }
        ],
        "gidMappings": [
            { "containerID": 0, "hostID": 1000, "size": 65536 }
        ]
    }
}
```

### Cgroups (Resource Limits)

```json
{
    "linux": {
        "resources": {
            "memory": {
                "limit": 536870912,
                "reservation": 268435456,
                "swap": 536870912
            },
            "cpu": {
                "shares": 1024,
                "quota": 100000,
                "period": 100000,
                "cpus": "0-3"
            },
            "pids": {
                "limit": 512
            },
            "blockIO": {
                "weight": 500
            }
        },
        "cgroupsPath": "/myapp"
    }
}
```

### Seccomp Profile

```json
{
    "linux": {
        "seccomp": {
            "defaultAction": "SCMP_ACT_ERRNO",
            "defaultErrnoRet": 1,
            "architectures": ["SCMP_ARCH_X86_64", "SCMP_ARCH_X86"],
            "syscalls": [
                {
                    "names": ["read", "write", "exit", "exit_group",
                              "openat", "close", "fstat", "mmap",
                              "brk", "futex", "nanosleep"],
                    "action": "SCMP_ACT_ALLOW"
                }
            ]
        }
    }
}
```

## Container Lifecycle

### Runtime Operations

```bash
# OCI runtime lifecycle (using runc as example)

# 1. Create container (sets up namespaces, cgroups, rootfs)
runc create --bundle /path/to/bundle mycontainer

# 2. Start container (executes the process)
runc start mycontainer

# 3. Query state
runc state mycontainer
# { "ociVersion": "1.1.0", "id": "mycontainer", "status": "running", ... }

# 4. Execute additional process
runc exec mycontainer /bin/sh

# 5. Pause/Resume
runc pause mycontainer
runc resume mycontainer

# 6. Send signal
runc kill mycontainer SIGTERM

# 7. Delete container
runc delete mycontainer

# State machine: creating -> created -> running -> stopped -> (deleted)
```

### Bundle Directory Structure

```
mycontainer/
├── config.json          # OCI runtime configuration
└── rootfs/              # Container root filesystem
    ├── bin/
    ├── etc/
    ├── usr/
    └── ...
```

## Image Specification

### Image Manifest

```json
{
    "schemaVersion": 2,
    "mediaType": "application/vnd.oci.image.manifest.v1+json",
    "config": {
        "mediaType": "application/vnd.oci.image.config.v1+json",
        "digest": "sha256:b5b2b2c...",
        "size": 7023
    },
    "layers": [
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
            "digest": "sha256:9834876d...",
            "size": 32654
        },
        {
            "mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
            "digest": "sha256:ec4b8955...",
            "size": 16724
        }
    ],
    "annotations": {
        "org.opencontainers.image.created": "2024-01-15T10:00:00Z",
        "org.opencontainers.image.source": "https://github.com/org/repo"
    }
}
```

### Image Index (Multi-Arch)

```json
{
    "schemaVersion": 2,
    "mediaType": "application/vnd.oci.image.index.v1+json",
    "manifests": [
        {
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": "sha256:e692418...",
            "size": 7143,
            "platform": {
                "architecture": "amd64",
                "os": "linux"
            }
        },
        {
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": "sha256:5b0bcaa...",
            "size": 7682,
            "platform": {
                "architecture": "arm64",
                "os": "linux",
                "variant": "v8"
            }
        }
    ]
}
```

### Image Configuration

```json
{
    "created": "2024-01-15T10:00:00Z",
    "architecture": "amd64",
    "os": "linux",
    "config": {
        "User": "1000:1000",
        "ExposedPorts": { "8080/tcp": {} },
        "Env": ["PATH=/usr/local/bin:/usr/bin", "APP_ENV=production"],
        "Entrypoint": ["/app"],
        "Cmd": ["--config", "/etc/app.yaml"],
        "WorkingDir": "/app",
        "Labels": {
            "version": "1.2.3",
            "org.opencontainers.image.source": "https://github.com/org/repo"
        }
    },
    "rootfs": {
        "type": "layers",
        "diff_ids": [
            "sha256:abc123...",
            "sha256:def456..."
        ]
    },
    "history": [
        {
            "created": "2024-01-15T10:00:00Z",
            "created_by": "/bin/sh -c #(nop) ADD file:abc123 in / "
        }
    ]
}
```

## Distribution Specification

### Registry API

```bash
# API v2 base URL
GET /v2/                                   # Check API support

# List repositories
GET /v2/_catalog                           # Repository list

# List tags
GET /v2/<repo>/tags/list                   # Tags for repository

# Pull manifest
GET /v2/<repo>/manifests/<reference>       # By tag or digest
# Accept: application/vnd.oci.image.manifest.v1+json
# Accept: application/vnd.oci.image.index.v1+json

# Pull blob (layer or config)
GET /v2/<repo>/blobs/<digest>              # By content digest

# Check blob existence
HEAD /v2/<repo>/blobs/<digest>             # Returns 200 or 404

# Push blob (two-step)
POST /v2/<repo>/blobs/uploads/             # Initiate upload
PUT /v2/<repo>/blobs/uploads/<uuid>?digest=sha256:...  # Complete

# Push manifest
PUT /v2/<repo>/manifests/<reference>       # By tag

# Delete manifest
DELETE /v2/<repo>/manifests/<digest>       # By digest only

# Delete blob
DELETE /v2/<repo>/blobs/<digest>
```

### Content Addressing

```bash
# All content is addressed by digest
# digest = algorithm:hex_hash
# sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

# Verify blob integrity
curl -sL "https://registry.com/v2/repo/blobs/sha256:abc123..." | sha256sum
# Must match: abc123...

# Digest calculation
sha256sum < layer.tar.gz
# sha256:9834876dcfb05cb167a5c24953eba58c4ac89b1adf57f28f2f9d09af107ee8f0
```

### Standard Annotations

```
org.opencontainers.image.created       # RFC 3339 timestamp
org.opencontainers.image.authors       # Contact details
org.opencontainers.image.url           # URL for more info
org.opencontainers.image.documentation # Documentation URL
org.opencontainers.image.source        # Source code URL
org.opencontainers.image.version       # Semantic version
org.opencontainers.image.revision      # VCS revision (git SHA)
org.opencontainers.image.vendor        # Distributing entity
org.opencontainers.image.licenses      # SPDX license expression
org.opencontainers.image.title         # Human-readable title
org.opencontainers.image.description   # Human-readable description
org.opencontainers.image.base.name     # Base image reference
org.opencontainers.image.base.digest   # Base image digest
```

## Tips

- The OCI Runtime Spec defines how to run containers; the Image Spec defines how to package them
- Content-addressable storage (digest-based) ensures integrity: any modification changes the digest
- Image layers are ordered; each layer is a tar archive of filesystem changes (additions, modifications, deletions)
- The image index (manifest list) enables multi-architecture images under a single tag
- Use OCI annotations for image metadata instead of custom labels for interoperability
- The distribution spec is backwards-compatible with Docker Registry v2 API
- Whiteout files (`.wh.filename`) in layers mark deleted files from previous layers
- The `noNewPrivileges` flag in runtime config prevents setuid escalation inside containers
- runc is the reference OCI runtime; alternatives like crun (C) and youki (Rust) offer better performance
- Always set `readonly: true` for the root filesystem and use explicit volume mounts for writable paths
- Use `diff_ids` (uncompressed digests) to identify layer content independently of compression

## See Also

docker, podman, buildah, skopeo, containerd, cri

## References

- [OCI Runtime Specification](https://github.com/opencontainers/runtime-spec)
- [OCI Image Specification](https://github.com/opencontainers/image-spec)
- [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec)
- [runc — OCI Runtime Reference Implementation](https://github.com/opencontainers/runc)
- [Open Container Initiative](https://opencontainers.org/)
