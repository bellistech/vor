# Docker Compose (Multi-Container Orchestration)

Define and run multi-container applications with a single YAML file.

## Starting and Stopping

### Up and down

```bash
docker compose up -d                    # start all services detached
docker compose up -d api worker         # start specific services only
docker compose up --build               # rebuild images before starting
docker compose up --force-recreate      # recreate even if config unchanged
docker compose up --remove-orphans      # remove services no longer in file
docker compose down                     # stop and remove containers + networks
docker compose down -v                  # also remove named volumes
docker compose down --rmi all           # also remove images
```

### Restart and stop

```bash
docker compose restart api
docker compose stop                     # stop without removing
docker compose start                    # start previously stopped
docker compose pause api
docker compose unpause api
```

## Building

```bash
docker compose build                    # build all services with build: key
docker compose build --no-cache api     # rebuild without cache
docker compose build --parallel         # build services in parallel
docker compose build --pull             # pull base images before building
```

## Logs

```bash
docker compose logs                     # all services
docker compose logs -f                  # follow mode
docker compose logs -f api worker       # specific services
docker compose logs --tail 50 api       # last 50 lines
docker compose logs --since 10m         # last 10 minutes
docker compose logs --timestamps
```

## Exec and Run

```bash
docker compose exec api sh                     # shell into running container
docker compose exec -u root api bash            # as root
docker compose exec db psql -U postgres         # run command directly
docker compose run --rm api python manage.py migrate   # one-off command in new container
docker compose run --rm -e DEBUG=1 api pytest    # with extra env vars
```

## Status and Inspection

```bash
docker compose ps                       # running services
docker compose ps -a                    # all services including stopped
docker compose top                      # processes in each service
docker compose config                   # validate and print resolved config
docker compose config --services        # list service names
docker compose config --volumes         # list volume names
docker compose images                   # images used by services
docker compose port api 8080            # show host port mapping
```

## Compose File Features

### Volumes

```yaml
# compose.yaml
services:
  db:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data      # named volume
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql:ro  # bind mount

volumes:
  pgdata:
    driver: local
```

### Networks

```yaml
services:
  api:
    networks:
      - frontend
      - backend
  db:
    networks:
      - backend

networks:
  frontend:
  backend:
    internal: true    # no external access
```

### Healthchecks

```yaml
services:
  api:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

### Depends_on with conditions

```yaml
services:
  api:
    depends_on:
      db:
        condition: service_healthy    # wait for healthcheck to pass
      redis:
        condition: service_started    # just wait for start
```

### Profiles

```yaml
services:
  api:
    image: myapp
  debug:
    image: myapp-debug
    profiles:
      - debug               # only starts with --profile debug
```

```bash
docker compose up -d                        # starts api only
docker compose --profile debug up -d        # starts api + debug
```

### Environment variables

```yaml
services:
  api:
    environment:
      - DATABASE_URL=postgres://db:5432/app
      - DEBUG=false
    env_file:
      - .env
      - .env.local
```

### Resource limits

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 512M
        reservations:
          memory: 256M
```

## Multiple Compose Files

```bash
docker compose -f compose.yaml -f compose.prod.yaml up -d
docker compose -f compose.yaml -f compose.test.yaml run --rm tests
```

## Scaling

```bash
docker compose up -d --scale worker=3   # run 3 worker instances
```

## Tips

- `docker compose` (v2, plugin) replaces `docker-compose` (v1, standalone). Use the space form.
- `depends_on` without `condition` only controls startup order, not readiness. Use `service_healthy`.
- `.env` file in the same directory as `compose.yaml` is loaded automatically for variable substitution in the YAML.
- `docker compose config` is invaluable for debugging variable interpolation and merge issues.
- Profiles let you keep dev-only services (debuggers, seed scripts) in the same file without starting them by default.
- Named volumes persist across `down`; use `down -v` only when you want a clean slate.
- `docker compose run` creates a new container; `docker compose exec` attaches to an existing one.
- Use `restart: unless-stopped` for services that should survive host reboots.
