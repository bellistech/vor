# Traefik (Cloud-Native Reverse Proxy)

Route HTTP, TCP, and UDP traffic to backend services with automatic TLS certificate management, Docker and Kubernetes service discovery, middleware chains, and weighted load balancing using Traefik Proxy.

## Static Configuration

### CLI / Environment / File

```yaml
# traefik.yaml (static config)
entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"
  metrics:
    address: ":8082"

api:
  dashboard: true
  insecure: false

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: traefik
  file:
    directory: /etc/traefik/dynamic/
    watch: true
  kubernetesIngress: {}

certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      storage: /acme/acme.json
      httpChallenge:
        entryPoint: web

log:
  level: INFO

accessLog:
  filePath: /var/log/traefik/access.log
  format: json

metrics:
  prometheus:
    entryPoint: metrics
```

### Docker Compose Deployment

```yaml
# docker-compose.yaml
services:
  traefik:
    image: traefik:v3.1
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik.yaml:/etc/traefik/traefik.yaml:ro
      - ./dynamic/:/etc/traefik/dynamic/:ro
      - acme-data:/acme
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.dashboard.rule=Host(`traefik.example.com`)"
      - "traefik.http.routers.dashboard.service=api@internal"
      - "traefik.http.routers.dashboard.tls.certResolver=letsencrypt"
      - "traefik.http.routers.dashboard.middlewares=auth"
      - "traefik.http.middlewares.auth.basicauth.users=admin:$$apr1$$..."

volumes:
  acme-data:

networks:
  traefik:
    external: true
```

## Routers (Dynamic Configuration)

### Docker Labels

```yaml
services:
  webapp:
    image: myapp:latest
    labels:
      - "traefik.enable=true"
      # HTTP router
      - "traefik.http.routers.webapp.rule=Host(`app.example.com`)"
      - "traefik.http.routers.webapp.entrypoints=websecure"
      - "traefik.http.routers.webapp.tls.certresolver=letsencrypt"
      # Service
      - "traefik.http.services.webapp.loadbalancer.server.port=8080"
      # Middleware chain
      - "traefik.http.routers.webapp.middlewares=ratelimit,compress,headers"
    networks:
      - traefik
```

### File Provider (Dynamic Config)

```yaml
# /etc/traefik/dynamic/routes.yaml
http:
  routers:
    api-router:
      rule: "Host(`api.example.com`) && PathPrefix(`/v1`)"
      service: api-service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt
      middlewares:
        - ratelimit
        - cors

    legacy-router:
      rule: "Host(`old.example.com`)"
      service: legacy-service
      entryPoints:
        - websecure
      middlewares:
        - redirect-to-new

  services:
    api-service:
      loadBalancer:
        servers:
          - url: "http://10.0.1.10:8080"
          - url: "http://10.0.1.11:8080"
          - url: "http://10.0.1.12:8080"
        healthCheck:
          path: /health
          interval: "10s"
          timeout: "3s"

    legacy-service:
      loadBalancer:
        servers:
          - url: "http://10.0.2.5:3000"
```

### Router Rules

```yaml
# Host matching
rule: "Host(`example.com`)"
rule: "Host(`example.com`) || Host(`www.example.com`)"

# Path matching
rule: "PathPrefix(`/api`)"
rule: "Path(`/api/v1/health`)"

# Combined rules
rule: "Host(`api.example.com`) && PathPrefix(`/v1`)"

# Header matching
rule: "Headers(`X-Custom-Header`, `value`)"
rule: "HeadersRegexp(`X-Forwarded-For`, `^10\\.`)"

# Method matching
rule: "Method(`GET`, `POST`)"

# Query parameter matching
rule: "Query(`debug=true`)"

# Priority (higher = matched first)
priority: 100
```

## Middlewares

### Common Middlewares

```yaml
http:
  middlewares:
    # Rate limiting
    ratelimit:
      rateLimit:
        average: 100          # requests per second
        burst: 200
        period: 1s
        sourceCriterion:
          ipStrategy:
            depth: 1

    # Basic auth
    auth:
      basicAuth:
        users:
          - "admin:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/"

    # Forward auth (external auth service)
    auth-forward:
      forwardAuth:
        address: "http://auth-service:9090/verify"
        trustForwardHeader: true
        authResponseHeaders:
          - "X-User-Id"
          - "X-User-Role"

    # Headers
    headers:
      headers:
        stsSeconds: 31536000
        stsIncludeSubdomains: true
        stsPreload: true
        forceSTSHeader: true
        contentTypeNosniff: true
        browserXssFilter: true
        frameDeny: true
        customResponseHeaders:
          X-Powered-By: ""

    # Compression
    compress:
      compress:
        excludedContentTypes:
          - "text/event-stream"

    # Strip prefix
    strip-api:
      stripPrefix:
        prefixes:
          - "/api"

    # Redirect
    redirect-to-new:
      redirectRegex:
        regex: "^https://old.example.com/(.*)"
        replacement: "https://new.example.com/${1}"
        permanent: true

    # Circuit breaker
    circuitbreaker:
      circuitBreaker:
        expression: "LatencyAtQuantileMS(50.0) > 200 || NetworkErrorRatio() > 0.30"

    # Retry
    retry:
      retry:
        attempts: 3
        initialInterval: "100ms"

    # IP whitelist
    ipallow:
      ipAllowList:
        sourceRange:
          - "10.0.0.0/8"
          - "172.16.0.0/12"
```

## Let's Encrypt (ACME)

### Certificate Resolvers

```yaml
# HTTP challenge (port 80 must be accessible)
certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      storage: /acme/acme.json
      httpChallenge:
        entryPoint: web

# TLS challenge (port 443, no separate HTTP needed)
certificatesResolvers:
  letsencrypt-tls:
    acme:
      email: admin@example.com
      storage: /acme/acme.json
      tlsChallenge: {}

# DNS challenge (wildcard certs, no public port needed)
certificatesResolvers:
  letsencrypt-dns:
    acme:
      email: admin@example.com
      storage: /acme/acme.json
      dnsChallenge:
        provider: cloudflare
        resolvers:
          - "1.1.1.1:53"
        delayBeforeCheck: 10s
```

### Wildcard Certificates

```yaml
# Router with wildcard cert
http:
  routers:
    wildcard:
      rule: "HostRegexp(`{subdomain:.+}.example.com`)"
      tls:
        certResolver: letsencrypt-dns
        domains:
          - main: "example.com"
            sans:
              - "*.example.com"
```

## TCP and UDP Routing

### TCP Router

```yaml
tcp:
  routers:
    postgres:
      rule: "HostSNI(`db.example.com`)"
      service: postgres-service
      tls:
        passthrough: true
      entryPoints:
        - postgres

  services:
    postgres-service:
      loadBalancer:
        servers:
          - address: "10.0.1.20:5432"
          - address: "10.0.1.21:5432"

# Entry point for TCP
# entryPoints:
#   postgres:
#     address: ":5432"
```

### UDP Router

```yaml
udp:
  routers:
    dns:
      service: dns-service
      entryPoints:
        - dns

  services:
    dns-service:
      loadBalancer:
        servers:
          - address: "10.0.1.30:53"

# entryPoints:
#   dns:
#     address: ":53/udp"
```

## Load Balancing

### Weighted Round Robin

```yaml
http:
  services:
    canary-service:
      weighted:
        services:
          - name: stable
            weight: 90
          - name: canary
            weight: 10

    stable:
      loadBalancer:
        servers:
          - url: "http://10.0.1.10:8080"
          - url: "http://10.0.1.11:8080"

    canary:
      loadBalancer:
        servers:
          - url: "http://10.0.2.10:8080"
```

### Sticky Sessions

```yaml
http:
  services:
    sticky-service:
      loadBalancer:
        sticky:
          cookie:
            name: srv_id
            secure: true
            httpOnly: true
        servers:
          - url: "http://10.0.1.10:8080"
          - url: "http://10.0.1.11:8080"
```

## Tips

- Set `exposedByDefault: false` in the Docker provider and explicitly enable routing per service with `traefik.enable=true` -- otherwise every container gets a public route
- Use the DNS ACME challenge for wildcard certificates -- HTTP and TLS challenges cannot issue wildcards
- Chain middlewares in order: authentication first, then rate limiting, then compression to avoid wasting CPU on unauthorized or rate-limited requests
- Set `stsSeconds: 31536000` with `stsPreload: true` for HSTS -- anything less than 1 year fails browser preload list requirements
- Use circuit breakers (`NetworkErrorRatio() > 0.30`) to stop sending traffic to backends that are failing, giving them time to recover
- Store `acme.json` on a persistent volume -- losing it forces re-issuance of all certificates and can hit Let's Encrypt rate limits
- Use `forwardAuth` middleware to delegate authentication to an external service rather than embedding auth logic in Traefik config
- Enable access logs in JSON format for parsing with log aggregation tools -- the common log format is harder to query
- Use `HostSNI(`*`)` for catch-all TCP routing only when TLS passthrough is needed -- prefer explicit hostnames
- Set health check intervals on load balancers to detect backend failures before users do
- Use weighted services (90/10 split) for canary deployments rather than deploying directly to all backends

## See Also

- nginx, haproxy, envoy, consul, nomad, letsencrypt, docker

## References

- [Traefik Documentation](https://doc.traefik.io/traefik/)
- [Traefik Routers](https://doc.traefik.io/traefik/routing/routers/)
- [Traefik Middlewares](https://doc.traefik.io/traefik/middlewares/overview/)
- [Let's Encrypt ACME Protocol](https://letsencrypt.org/docs/)
- [Traefik Docker Provider](https://doc.traefik.io/traefik/providers/docker/)
