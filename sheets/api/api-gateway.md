# API Gateway

Centralized entry point that manages, secures, and routes API traffic between clients and backend services.

## Core Functions

```
+--------+      +-------------+      +-----------+
| Client | ---> | API Gateway | ---> | Backend   |
+--------+      +-------------+      | Services  |
                  |  Auth           +-----------+
                  |  Rate limiting
                  |  Routing
                  |  Transformation
                  |  Caching
                  |  Load balancing
                  |  Monitoring
```

## Rate Limiting

### Token Bucket Algorithm

```bash
# Bucket starts full (capacity C), refills at rate R tokens/sec
# Each request consumes 1 token; rejected if bucket is empty

# Nginx rate limiting (token bucket)
# /etc/nginx/nginx.conf
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

server {
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://backend;
    }
}
```

### Sliding Window

```bash
# Redis-based sliding window rate limiter
redis-cli EVAL "
    local key = KEYS[1]
    local window = tonumber(ARGV[1])
    local limit = tonumber(ARGV[2])
    local now = tonumber(ARGV[3])
    redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
    local count = redis.call('ZCARD', key)
    if count < limit then
        redis.call('ZADD', key, now, now .. math.random())
        redis.call('EXPIRE', key, window)
        return 1
    end
    return 0
" 1 "ratelimit:user:123" 60 100 $(date +%s)
# Window: 60s, Limit: 100 requests
```

### Rate Limit Headers

```
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1704067200
Retry-After: 30
```

## Authentication

### JWT Validation

```bash
# Nginx JWT validation (njs module)
# Or use Kong JWT plugin:
curl -X POST http://kong:8001/routes/api/plugins \
    --data "name=jwt"

# Verify JWT manually
echo "$TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

### API Key Authentication

```bash
# Kong API key plugin
curl -X POST http://kong:8001/routes/api/plugins \
    --data "name=key-auth" \
    --data "config.key_names=X-API-Key"

# Create a consumer with key
curl -X POST http://kong:8001/consumers \
    --data "username=app1"
curl -X POST http://kong:8001/consumers/app1/key-auth \
    --data "key=my-secret-api-key"

# AWS API Gateway API key
aws apigateway create-api-key \
    --name "partner-key" \
    --enabled
```

### OAuth 2.0 / OIDC

```bash
# Kong OIDC plugin
curl -X POST http://kong:8001/routes/api/plugins \
    --data "name=openid-connect" \
    --data "config.issuer=https://auth.example.com/.well-known/openid-configuration" \
    --data "config.client_id=my-client" \
    --data "config.client_secret=secret"
```

## Request Routing

```bash
# Path-based routing
# /api/v1/users   -> user-service:8080
# /api/v1/orders  -> order-service:8080
# /api/v2/users   -> user-service-v2:8080

# Nginx upstream routing
upstream users_v1 { server user-service:8080; }
upstream orders_v1 { server order-service:8080; }

server {
    location /api/v1/users { proxy_pass http://users_v1; }
    location /api/v1/orders { proxy_pass http://orders_v1; }
}

# Header-based routing (canary)
# Route 10% of traffic to canary
map $request_id $backend {
    ~^[0-9a-f]$   canary_upstream;    # ~6% of traffic
    default       stable_upstream;
}
```

## Request/Response Transformation

```bash
# Kong request-transformer plugin
curl -X POST http://kong:8001/routes/api/plugins \
    --data "name=request-transformer" \
    --data "config.add.headers=X-Request-Source:gateway" \
    --data "config.remove.querystring=debug" \
    --data "config.rename.headers=X-Old:X-New"

# AWS API Gateway mapping template (VTL)
# Transform request body
#set($inputRoot = $input.path('$'))
{
    "userId": "$inputRoot.user_id",
    "timestamp": "$context.requestTimeEpoch"
}
```

## Caching

```bash
# Nginx proxy caching
proxy_cache_path /var/cache/nginx levels=1:2
    keys_zone=api_cache:10m max_size=1g inactive=60m;

server {
    location /api/ {
        proxy_cache api_cache;
        proxy_cache_valid 200 10m;
        proxy_cache_valid 404 1m;
        proxy_cache_key "$request_uri";
        proxy_cache_bypass $http_cache_control;
        add_header X-Cache-Status $upstream_cache_status;
        proxy_pass http://backend;
    }
}

# AWS API Gateway caching
aws apigateway update-stage \
    --rest-api-id abc123 \
    --stage-name prod \
    --patch-operations \
        op=replace,path=/cacheClusterEnabled,value=true \
        op=replace,path=/cacheClusterSize,value=0.5
```

## Circuit Breaking

```bash
# Envoy circuit breaker configuration
# envoy.yaml
clusters:
  - name: backend
    circuit_breakers:
      thresholds:
        - max_connections: 100
          max_pending_requests: 50
          max_requests: 200
          max_retries: 3

# Kong circuit breaker (via proxy-cache or custom plugin)
# Upstream health checks
curl -X PATCH http://kong:8001/upstreams/backend \
    --data "healthchecks.active.healthy.interval=5" \
    --data "healthchecks.active.unhealthy.interval=5" \
    --data "healthchecks.active.unhealthy.tcp_failures=3"
```

## Load Balancing

```bash
# Nginx load balancing strategies
upstream backend {
    # Round robin (default)
    server backend1:8080;
    server backend2:8080;

    # Weighted
    server backend1:8080 weight=3;
    server backend2:8080 weight=1;

    # Least connections
    least_conn;
    server backend1:8080;
    server backend2:8080;

    # IP hash (sticky sessions)
    ip_hash;
    server backend1:8080;
    server backend2:8080;
}
```

## Canary Routing

```bash
# Kong canary release plugin
curl -X POST http://kong:8001/routes/api/plugins \
    --data "name=canary" \
    --data "config.percentage=10" \
    --data "config.upstream_host=canary-backend" \
    --data "config.upstream_port=8080"

# AWS API Gateway canary deployment
aws apigateway create-deployment \
    --rest-api-id abc123 \
    --stage-name prod \
    --canary-settings '{
        "percentTraffic": 10,
        "useStageCache": false
    }'
```

## OpenAPI Integration

```bash
# Import OpenAPI spec into AWS API Gateway
aws apigateway import-rest-api \
    --body fileb://openapi.yaml \
    --fail-on-warnings

# Kong with decK (declarative config from OpenAPI)
deck file openapi2kong --spec openapi.yaml --output kong.yaml
deck sync --state kong.yaml

# APISIX import OpenAPI
apisix-cli import openapi -f openapi.yaml
```

## Gateway Comparison

```
Feature           | Kong        | APISIX     | Tyk        | AWS API GW
------------------|-------------|------------|------------|------------
Core              | Nginx/Lua   | Nginx/Lua  | Go         | Managed
License           | Apache 2.0  | Apache 2.0 | MPL 2.0    | Proprietary
Plugins           | 100+        | 80+        | 50+        | Limited
Config            | DB/dbless   | etcd       | Redis      | Console/CLI
K8s Ingress       | Yes         | Yes        | Yes        | N/A
gRPC              | Yes         | Yes        | Yes        | Yes (HTTP)
WebSocket         | Yes         | Yes        | Yes        | Yes
Rate Limiting     | Plugin      | Plugin     | Built-in   | Usage plans
Auth              | Plugin      | Plugin     | Built-in   | Authorizers
Latency           | ~1-2ms      | ~0.5-1ms   | ~1-3ms     | ~10-30ms
```

## Kong Quickstart

```bash
# Start Kong with Docker
docker run -d --name kong-database \
    -e POSTGRES_USER=kong -e POSTGRES_DB=kong \
    postgres:15

docker run --rm --link kong-database \
    -e KONG_DATABASE=postgres \
    -e KONG_PG_HOST=kong-database \
    kong:latest kong migrations bootstrap

docker run -d --name kong --link kong-database \
    -e KONG_DATABASE=postgres \
    -e KONG_PG_HOST=kong-database \
    -e KONG_PROXY_LISTEN=0.0.0.0:8000 \
    -e KONG_ADMIN_LISTEN=0.0.0.0:8001 \
    -p 8000:8000 -p 8001:8001 \
    kong:latest

# Add a service and route
curl -X POST http://localhost:8001/services \
    --data "name=my-api" --data "url=http://backend:8080"
curl -X POST http://localhost:8001/services/my-api/routes \
    --data "paths[]=/api"
```

## Tips

- Always terminate TLS at the gateway; backends should receive plaintext over internal networks
- Use correlation IDs (`X-Request-ID`) injected at the gateway for distributed tracing
- Rate limit by multiple dimensions: IP, API key, user ID, endpoint, HTTP method
- Cache only GET requests with stable responses; invalidate caches on writes
- Set request size limits to prevent abuse (`client_max_body_size` in Nginx)
- Use health checks (active and passive) to remove unhealthy backends automatically
- Log all requests at the gateway level for centralized audit and debugging
- Prefer API Gateway in DB-less/declarative mode for GitOps workflows
- Strip internal headers (`X-Internal-*`) at the gateway before forwarding to clients
- Use request transformation to normalize client inputs before reaching backend services
- Deploy gateways in multiple availability zones for high availability
- Monitor p99 latency at the gateway; it reveals both gateway overhead and backend slowness

## See Also

- Load Balancer (L4/L7 traffic distribution)
- Service Mesh (sidecar-based inter-service communication)
- OAuth 2.0 / OIDC (authentication protocols)
- OpenAPI / Swagger (API specification format)
- Webhook (event-driven API pattern)

## References

- [Kong Gateway Documentation](https://docs.konghq.com/gateway/latest/)
- [Apache APISIX Documentation](https://apisix.apache.org/docs/)
- [Tyk Gateway Documentation](https://tyk.io/docs/)
- [AWS API Gateway Developer Guide](https://docs.aws.amazon.com/apigateway/latest/developerguide/)
- [Rate Limiting Algorithms (Stripe blog)](https://stripe.com/blog/rate-limiters)
- [NGINX Reverse Proxy Guide](https://docs.nginx.com/nginx/admin-guide/web-server/reverse-proxy/)
