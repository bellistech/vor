# CRI (Container Runtime Interface)

Kubernetes plugin API that abstracts container runtime operations behind a gRPC interface, allowing kubelets to manage pod sandboxes and containers through any compliant runtime like containerd or CRI-O.

## Architecture

### CRI Components

```
kubelet
  └── CRI gRPC client
        ├── RuntimeService    # Pod sandbox + container lifecycle
        └── ImageService      # Image pull, list, remove

CRI Runtime (server)
  ├── containerd (+ containerd-shim-runc-v2)
  ├── CRI-O (+ conmon)
  └── Other compliant runtimes

OCI Runtime (low-level)
  ├── runc          # Default, reference implementation
  ├── crun          # C implementation, faster startup
  ├── youki         # Rust implementation
  ├── gVisor (runsc)  # Sandboxed, user-space kernel
  └── Kata Containers # VM-based isolation
```

### CRI gRPC Services

```protobuf
// RuntimeService — Pod and container lifecycle
service RuntimeService {
    rpc RunPodSandbox(RunPodSandboxRequest) returns (RunPodSandboxResponse);
    rpc StopPodSandbox(StopPodSandboxRequest) returns (StopPodSandboxResponse);
    rpc RemovePodSandbox(RemovePodSandboxRequest) returns (RemovePodSandboxResponse);
    rpc PodSandboxStatus(PodSandboxStatusRequest) returns (PodSandboxStatusResponse);
    rpc ListPodSandbox(ListPodSandboxRequest) returns (ListPodSandboxResponse);

    rpc CreateContainer(CreateContainerRequest) returns (CreateContainerResponse);
    rpc StartContainer(StartContainerRequest) returns (StartContainerResponse);
    rpc StopContainer(StopContainerRequest) returns (StopContainerResponse);
    rpc RemoveContainer(RemoveContainerRequest) returns (RemoveContainerResponse);
    rpc ListContainers(ListContainersRequest) returns (ListContainersResponse);
    rpc ContainerStatus(ContainerStatusRequest) returns (ContainerStatusResponse);
    rpc ExecSync(ExecSyncRequest) returns (ExecSyncResponse);
    rpc Exec(ExecRequest) returns (ExecResponse);
    rpc Attach(AttachRequest) returns (AttachResponse);
    rpc PortForward(PortForwardRequest) returns (PortForwardResponse);
}

// ImageService — Image management
service ImageService {
    rpc ListImages(ListImagesRequest) returns (ListImagesResponse);
    rpc ImageStatus(ImageStatusRequest) returns (ImageStatusResponse);
    rpc PullImage(PullImageRequest) returns (PullImageResponse);
    rpc RemoveImage(RemoveImageRequest) returns (RemoveImageResponse);
    rpc ImageFsInfo(ImageFsInfoRequest) returns (ImageFsInfoResponse);
}
```

## Pod Sandbox Lifecycle

### State Machine

```
                    RunPodSandbox
         ┌──────────────────────────┐
         │                          ▼
    NOT_READY ◄──── StopPodSandbox ─── READY
         │                                │
         └──── RemovePodSandbox ──────────┘
                      │
                      ▼
                   REMOVED

# Pod sandbox = network namespace + shared resources
# Containers run inside the pod sandbox
# Pause container (or equivalent) holds namespaces
```

### Container Lifecycle

```
                 CreateContainer
         ┌────────────────────────┐
         │                        ▼
    (none) ──────────────────── CREATED
                                  │
                           StartContainer
                                  │
                                  ▼
                               RUNNING
                                  │
                           StopContainer
                                  │
                                  ▼
                               EXITED
                                  │
                          RemoveContainer
                                  │
                                  ▼
                               REMOVED
```

## crictl — CRI CLI Tool

### Configuration

```bash
# Configure crictl endpoint
cat > /etc/crictl.yaml <<EOF
runtime-endpoint: unix:///run/containerd/containerd.sock
image-endpoint: unix:///run/containerd/containerd.sock
timeout: 10
debug: false
EOF

# Or for CRI-O:
# runtime-endpoint: unix:///var/run/crio/crio.sock

# Or via flags:
crictl --runtime-endpoint unix:///run/containerd/containerd.sock ps
```

### Pod Operations

```bash
# List pods
crictl pods
crictl pods --name nginx --state ready
crictl pods --namespace kube-system

# Inspect pod
crictl inspectp POD_ID
crictl inspectp --output table POD_ID

# Run a pod sandbox (from config)
cat > pod-config.json <<EOF
{
    "metadata": { "name": "test-pod", "namespace": "default", "uid": "test-uid" },
    "log_directory": "/tmp/test-pod",
    "linux": {}
}
EOF
crictl runp pod-config.json

# Stop and remove pod
crictl stopp POD_ID
crictl rmp POD_ID
```

### Container Operations

```bash
# List containers
crictl ps                          # Running only
crictl ps -a                       # All (including stopped)
crictl ps --pod POD_ID             # Containers in specific pod

# Create and start container
cat > container-config.json <<EOF
{
    "metadata": { "name": "test-container" },
    "image": { "image": "busybox:latest" },
    "command": ["sleep", "3600"],
    "log_path": "test-container.log",
    "linux": {}
}
EOF
CONTAINER_ID=$(crictl create POD_ID container-config.json pod-config.json)
crictl start $CONTAINER_ID

# Inspect container
crictl inspect CONTAINER_ID

# Execute command in container
crictl exec -it CONTAINER_ID sh

# View logs
crictl logs CONTAINER_ID
crictl logs --tail 100 -f CONTAINER_ID

# Stop and remove
crictl stop CONTAINER_ID
crictl rm CONTAINER_ID
```

### Image Operations

```bash
# List images
crictl images
crictl images --digests

# Pull image
crictl pull nginx:latest
crictl pull --creds user:password registry.example.com/image:tag

# Inspect image
crictl inspecti nginx:latest

# Remove image
crictl rmi nginx:latest

# Image filesystem info
crictl imagefsinfo
```

## Containerd + Shim Architecture

### Shim Process Model

```
kubelet
  └── containerd (CRI plugin)
        └── containerd-shim-runc-v2 (per-pod process)
              ├── Container 1 (runc)
              ├── Container 2 (runc)
              └── Container N (runc)

# Shim responsibilities:
# - Holds container stdio
# - Reports exit status to containerd
# - Survives containerd restarts (shim is reparented to init)
# - One shim per pod (v2) vs one per container (v1)
```

### Containerd Configuration for K8s

```toml
# /etc/containerd/config.toml
version = 2

[plugins."io.containerd.grpc.v1.cri"]
  sandbox_image = "registry.k8s.io/pause:3.9"

  [plugins."io.containerd.grpc.v1.cri".containerd]
    default_runtime_name = "runc"

    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
      runtime_type = "io.containerd.runc.v2"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
        SystemdCgroup = true

  [plugins."io.containerd.grpc.v1.cri".registry]
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
        endpoint = ["https://registry-1.docker.io"]
```

## CRI-O

### Configuration

```toml
# /etc/crio/crio.conf
[crio.runtime]
default_runtime = "runc"
conmon = "/usr/bin/conmon"
log_level = "info"

[crio.runtime.runtimes.runc]
runtime_path = "/usr/bin/runc"
runtime_type = "oci"

[crio.runtime.runtimes.kata]
runtime_path = "/usr/bin/kata-runtime"
runtime_type = "oci"

[crio.image]
pause_image = "registry.k8s.io/pause:3.9"

[crio.network]
network_dir = "/etc/cni/net.d"
plugin_dirs = ["/opt/cni/bin"]
```

### CRI-O vs Containerd

```
Feature             containerd          CRI-O
Scope               General-purpose     K8s-only
Docker compatible   Yes (via Docker)    No
Image building      Yes (BuildKit)      No
CRI version         v1                  v1
OCI runtime         runc, gVisor, Kata  runc, gVisor, Kata
Cgroup driver       cgroupfs, systemd   cgroupfs, systemd
Pod overhead        ~10MB per pod       ~8MB per pod
```

## Tips

- The CRI replaced the earlier dockershim interface; Docker Engine is no longer a supported CRI runtime
- Use `crictl` instead of `docker` commands when debugging Kubernetes node-level container issues
- Containerd's v2 shim model uses one shim process per pod, reducing overhead compared to per-container shims
- CRI-O is purpose-built for Kubernetes and has a smaller attack surface than general-purpose containerd
- The pause container in each pod holds the network namespace alive so containers can be restarted independently
- Set `SystemdCgroup = true` in containerd config to match kubelet's cgroup driver (required for systemd-based distros)
- `crictl stats` provides resource usage per container, useful for capacity debugging on nodes
- RuntimeClass in Kubernetes maps workloads to different OCI runtimes (e.g., runc for default, gVisor for untrusted)
- CRI streaming (exec, attach, port-forward) uses a separate HTTP endpoint, not the main gRPC connection
- Always configure crictl.yaml to avoid passing `--runtime-endpoint` on every command
- Use `crictl pods --state notready` to quickly find pods stuck in bad states on a node
- The CRI ImageService is separate from RuntimeService, allowing independent image management

## See Also

containerd, docker, podman, oci, kubernetes

## References

- [Kubernetes CRI Documentation](https://kubernetes.io/docs/concepts/architecture/cri/)
- [CRI API Protobuf Definition](https://github.com/kubernetes/cri-api)
- [containerd CRI Plugin](https://github.com/containerd/containerd/tree/main/pkg/cri)
- [CRI-O Documentation](https://cri-o.io/)
- [crictl User Guide](https://github.com/kubernetes-sigs/cri-tools/blob/master/docs/crictl.md)
- [RuntimeClass Documentation](https://kubernetes.io/docs/concepts/containers/runtime-class/)
