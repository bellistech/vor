# Podman (Daemonless Container Engine)

Docker-compatible container engine that runs rootless by default with no daemon process.

## Running Containers

### Basic run

```bash
podman run -d --name web -p 8080:80 nginx:alpine
podman run -it --rm alpine sh
podman run -d -e POSTGRES_PASSWORD=secret -v pgdata:/var/lib/postgresql/data postgres:16
podman run -d --restart always --name api myapp:latest
podman run --userns keep-id -v $(pwd):/app myapp   # map host UID into container
```

### Container lifecycle

```bash
podman ps
podman ps -a
podman stop web
podman start web
podman restart web
podman rm web
podman rm -f web
podman logs -f web
podman logs --tail 50 web
podman exec -it web sh
podman inspect web
podman top web                             # processes in container
podman stats                               # live resource usage
```

## Building Images

```bash
podman build -t myapp:latest .
podman build -t myapp:v1 -f Dockerfile.prod .
podman build --no-cache -t myapp .
podman build --layers=false -t myapp .     # squash all layers
podman build --platform linux/amd64 -t myapp .
```

### Image management

```bash
podman images
podman pull docker.io/library/nginx:alpine
podman push myregistry.io/myapp:v1
podman tag myapp:latest myregistry.io/myapp:v1
podman rmi nginx:alpine
podman image prune -a
podman save myapp:latest -o myapp.tar
podman load -i myapp.tar
```

## Pods

### Create and manage pods

```bash
podman pod create --name mypod -p 8080:80 -p 5432:5432
podman pod list
podman pod inspect mypod
podman pod start mypod
podman pod stop mypod
podman pod rm mypod
podman pod rm -f mypod
```

### Run containers in a pod

```bash
podman run -d --pod mypod --name web nginx:alpine
podman run -d --pod mypod --name db postgres:16   # shares network with web
podman pod ps
podman ps --pod                            # show pod column
```

## Generate Kubernetes YAML

### From containers and pods

```bash
podman generate kube mypod > pod.yaml
podman generate kube mypod -s > pod-with-service.yaml   # include Service
podman generate kube web > deployment.yaml
```

### Play Kubernetes YAML

```bash
podman play kube pod.yaml
podman play kube pod.yaml --network mynet
podman play kube deployment.yaml --replace    # update existing
podman play kube pod.yaml --down              # tear down
```

## Rootless Containers

```bash
podman info --format '{{.Host.Security.Rootless}}'   # check if rootless
podman unshare cat /proc/self/uid_map                 # view user namespace mapping
podman system migrate                                  # after /etc/subuid changes
```

### Storage and config for rootless

```bash
# Config lives at ~/.config/containers/
# Storage at ~/.local/share/containers/
podman info --format '{{.Store.GraphRoot}}'
```

## Systemd Integration

### Generate systemd units

```bash
podman generate systemd --new --name web > ~/.config/systemd/user/container-web.service
podman generate systemd --new --name mypod --files   # generates files for pod + containers
systemctl --user daemon-reload
systemctl --user enable --now container-web.service
systemctl --user status container-web.service
```

### Quadlet (podman 4.4+)

```bash
# Place .container files in ~/.config/containers/systemd/
# Example: ~/.config/containers/systemd/web.container
# [Container]
# Image=nginx:alpine
# PublishPort=8080:80
# Volume=webdata:/usr/share/nginx/html

systemctl --user daemon-reload              # generates unit from quadlet
systemctl --user start web.service
```

### Enable lingering for rootless services

```bash
loginctl enable-linger $USER               # services run without active login
```

## Volumes and Networks

```bash
podman volume create mydata
podman volume ls
podman volume inspect mydata
podman volume rm mydata
podman network create mynet
podman network ls
podman network inspect mynet
podman network connect mynet web
podman network rm mynet
```

## Cleanup

```bash
podman system prune -a                     # remove all unused data
podman system prune -a --volumes
podman system df                           # show disk usage
podman container prune
podman image prune -a
```

## Tips

- Podman is daemonless: each container runs as a direct child process. No socket, no daemon to crash.
- Rootless is the default. Use `sudo podman` only when you need real root (bind to port < 1024, access host devices).
- `podman` CLI is command-compatible with `docker`. Alias `alias docker=podman` works for most workflows.
- Pods group containers sharing the same network namespace, similar to Kubernetes pods.
- `podman generate kube` + `podman play kube` is a bridge between local dev and Kubernetes deployment.
- Quadlet files (`.container`, `.volume`, `.network`) are the modern way to manage podman services with systemd. They replace `podman generate systemd`.
- `--userns keep-id` maps your host UID into the container, solving file permission issues with bind mounts in rootless mode.
- Podman uses Buildah under the hood for builds; all Buildah features are available.

## References

- [Podman Documentation](https://docs.podman.io/)
- [Podman GitHub Repository](https://github.com/containers/podman)
- [Podman CLI Reference](https://docs.podman.io/en/latest/Commands.html)
- [Podman Compose](https://github.com/containers/podman-compose)
- [Podman Pod Management](https://docs.podman.io/en/latest/markdown/podman-pod.1.html)
- [Podman Rootless Configuration](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)
- [Buildah — OCI Image Builder](https://github.com/containers/buildah)
- [Skopeo — Container Image Utility](https://github.com/containers/skopeo)
- [containers-registries.conf(5)](https://github.com/containers/image/blob/main/docs/containers-registries.conf.5.md)
- [Red Hat — Podman Documentation](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/building_running_and_managing_containers/)
- [podman(1) man page](https://docs.podman.io/en/latest/markdown/podman.1.html)
