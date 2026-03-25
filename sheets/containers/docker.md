# Docker (Container Runtime & Build Tool)

Build, ship, and run containers from images using layered filesystems and process isolation.

## Building Images

### Build from Dockerfile

```bash
docker build -t myapp:latest .
docker build -t myapp:v1.2.3 -f Dockerfile.prod .
docker build --no-cache -t myapp:latest .       # ignore layer cache
docker build --build-arg GO_VERSION=1.24 -t myapp .
docker build --target builder -t myapp-build .   # stop at named stage
```

### Multi-stage builds

```bash
# In Dockerfile:
# FROM golang:1.24 AS builder
# RUN go build -o /app .
# FROM alpine:3.19
# COPY --from=builder /app /app

docker build -t myapp:slim .    # final stage only, tiny image
```

### Buildx (multi-platform)

```bash
docker buildx create --use --name multiarch
docker buildx build --platform linux/amd64,linux/arm64 -t myapp:latest --push .
docker buildx ls                          # list builders
docker buildx inspect multiarch
docker buildx prune                       # clean build cache
```

## Running Containers

### Basic run

```bash
docker run -d --name web -p 8080:80 nginx
docker run -it --rm ubuntu bash                    # interactive, auto-remove
docker run -d --restart unless-stopped myapp       # survive reboots
docker run -d -e DATABASE_URL=postgres://db myapp  # env var
docker run -d --env-file .env myapp                # env from file
docker run -d --memory 512m --cpus 1.5 myapp       # resource limits
```

### Exec into running container

```bash
docker exec -it web bash
docker exec -it web sh              # alpine images lack bash
docker exec web cat /etc/hosts      # one-off command
docker exec -u root web whoami      # run as specific user
```

## Container Management

### List and inspect

```bash
docker ps                           # running containers
docker ps -a                        # all containers including stopped
docker ps -q                        # IDs only (useful for scripting)
docker ps --format '{{.Names}}\t{{.Status}}'
docker inspect web                  # full JSON details
docker inspect -f '{{.NetworkSettings.IPAddress}}' web
docker stats                        # live resource usage
docker top web                      # processes inside container
```

### Lifecycle

```bash
docker stop web
docker start web
docker restart web
docker kill web                     # SIGKILL immediately
docker rm web
docker rm -f web                    # force remove running container
docker stop $(docker ps -q)         # stop all running containers
```

### Logs

```bash
docker logs web
docker logs -f web                  # follow/tail
docker logs --tail 100 web          # last 100 lines
docker logs --since 5m web          # last 5 minutes
docker logs --timestamps web
```

## Images

### Manage images

```bash
docker images                       # list local images
docker images -q                    # IDs only
docker pull nginx:alpine
docker push myregistry/myapp:v1
docker tag myapp:latest myregistry/myapp:v1
docker rmi nginx:alpine
docker image history myapp:latest   # show layers
docker save myapp:latest -o myapp.tar
docker load -i myapp.tar
```

## Volumes

### Named volumes

```bash
docker volume create pgdata
docker volume ls
docker volume inspect pgdata
docker run -d -v pgdata:/var/lib/postgresql/data postgres:16
docker volume rm pgdata
```

### Bind mounts

```bash
docker run -d -v $(pwd)/config:/etc/myapp/config:ro myapp   # read-only
docker run -d -v $(pwd)/data:/data myapp
```

## Networks

```bash
docker network create mynet
docker network ls
docker network inspect mynet
docker run -d --network mynet --name api myapp
docker run -d --network mynet --name db postgres   # api can reach db by hostname
docker network connect mynet existing_container
docker network rm mynet
```

## Compose (single-file)

```bash
docker compose up -d
docker compose down
docker compose down -v              # also remove volumes
docker compose ps
docker compose logs -f api
docker compose exec api sh
docker compose build --no-cache
```

## Cleanup

### Prune unused resources

```bash
docker system prune                 # stopped containers, dangling images, unused networks
docker system prune -a --volumes    # everything unused including tagged images and volumes
docker system df                    # show disk usage
docker image prune -a               # all unused images
docker container prune              # stopped containers
docker volume prune                 # unused volumes
docker builder prune                # build cache
```

## Copy Files

```bash
docker cp web:/var/log/nginx/access.log ./access.log
docker cp ./config.yaml web:/etc/myapp/config.yaml
```

## Tips

- Use `.dockerignore` to exclude `.git/`, `node_modules/`, build artifacts from context -- speeds up builds dramatically.
- Multi-stage builds reduce final image size by 10-100x; never ship compiler toolchains.
- `docker run --rm` prevents accumulating stopped containers.
- `docker system df` before and after prune to see how much space you reclaimed.
- Pin image tags in production (`nginx:1.25-alpine` not `nginx:latest`).
- Containers on the same user-defined network can resolve each other by container name.
- `docker inspect` output is JSON; pipe through `jq` for complex queries.
- `HEALTHCHECK` in Dockerfile lets Docker track container health beyond "process running".

## References

- [Docker Documentation](https://docs.docker.com/)
- [Docker CLI Reference](https://docs.docker.com/reference/cli/docker/)
- [Dockerfile Reference](https://docs.docker.com/reference/dockerfile/)
- [Docker Build — Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [Docker Networking Overview](https://docs.docker.com/engine/network/)
- [Docker Storage — Volumes](https://docs.docker.com/engine/storage/volumes/)
- [Docker Hub](https://hub.docker.com/)
- [Docker Engine API Reference](https://docs.docker.com/engine/api/)
- [Docker Security Best Practices](https://docs.docker.com/build/building/best-practices/)
- [Moby Project (Docker Engine Source)](https://github.com/moby/moby)
- [docker(1) man page](https://man7.org/linux/man-pages/man1/docker.1.html)
