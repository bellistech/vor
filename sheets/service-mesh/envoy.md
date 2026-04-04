# Envoy Proxy (L7 Proxy and Service Mesh Data Plane)

High-performance C++ edge and service proxy designed for cloud-native applications, featuring dynamic xDS configuration, advanced load balancing, circuit breaking, rate limiting, and observability.

## Architecture

### Core components

```yaml
# Envoy bootstrap config — minimal structure
static_resources:
  listeners:
    - name: listener_0
      address:
        socket_address:
          address: 0.0.0.0
          port_value: 8080
      filter_chains:
        - filters:
            - name: envoy.filters.network.http_connection_manager
              typed_config:
                "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
                stat_prefix: ingress_http
                route_config:
                  name: local_route
                  virtual_hosts:
                    - name: backend
                      domains: ["*"]
                      routes:
                        - match:
                            prefix: "/"
                          route:
                            cluster: backend_service
                http_filters:
                  - name: envoy.filters.http.router
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  clusters:
    - name: backend_service
      type: STRICT_DNS
      lb_policy: ROUND_ROBIN
      load_assignment:
        cluster_name: backend_service
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: backend
                      port_value: 8080
```

### Run envoy

```bash
# Run with static config
envoy -c /etc/envoy/envoy.yaml

# Run with debug logging
envoy -c /etc/envoy/envoy.yaml -l debug

# Validate configuration
envoy --mode validate -c /etc/envoy/envoy.yaml

# Run as Docker container
docker run -d --name envoy \
  -p 8080:8080 -p 9901:9901 \
  -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml \
  envoyproxy/envoy:v1.29-latest
```

## Listeners and Filters

### HTTP connection manager

```yaml
# Listener with access logging and tracing
listeners:
  - name: http_listener
    address:
      socket_address: { address: 0.0.0.0, port_value: 8080 }
    filter_chains:
      - filters:
          - name: envoy.filters.network.http_connection_manager
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
              stat_prefix: ingress
              codec_type: AUTO
              access_log:
                - name: envoy.access_loggers.stdout
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
              route_config:
                name: local_route
                virtual_hosts:
                  - name: app
                    domains: ["*"]
                    routes:
                      - match: { prefix: "/api" }
                        route: { cluster: api_cluster }
                      - match: { prefix: "/" }
                        route: { cluster: web_cluster }
              http_filters:
                - name: envoy.filters.http.router
                  typed_config:
                    "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
```

### TCP proxy filter

```yaml
# Layer 4 TCP proxy
filter_chains:
  - filters:
      - name: envoy.filters.network.tcp_proxy
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
          stat_prefix: tcp_stats
          cluster: tcp_backend
```

## Clusters and Load Balancing

### Cluster configuration

```yaml
clusters:
  # DNS-based service discovery
  - name: api_cluster
    type: STRICT_DNS
    connect_timeout: 5s
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: api_cluster
      endpoints:
        - lb_endpoints:
            - endpoint:
                address:
                  socket_address: { address: api-svc, port_value: 8080 }
              load_balancing_weight: 80
            - endpoint:
                address:
                  socket_address: { address: api-canary, port_value: 8080 }
              load_balancing_weight: 20

  # EDS — dynamic endpoint discovery via xDS
  - name: dynamic_cluster
    type: EDS
    eds_cluster_config:
      eds_config:
        api_config_source:
          api_type: GRPC
          grpc_services:
            - envoy_grpc:
                cluster_name: xds_cluster
```

### Health checking

```yaml
clusters:
  - name: backend
    health_checks:
      - timeout: 2s
        interval: 10s
        unhealthy_threshold: 3
        healthy_threshold: 2
        http_health_check:
          path: "/healthz"
          expected_statuses:
            - start: 200
              end: 299
```

## xDS API (Dynamic Configuration)

### xDS resource types

```bash
# LDS — Listener Discovery Service
# RDS — Route Discovery Service
# CDS — Cluster Discovery Service
# EDS — Endpoint Discovery Service
# SDS — Secret Discovery Service (TLS certs)

# Dynamic resources config (bootstrap)
```

```yaml
dynamic_resources:
  lds_config:
    resource_api_version: V3
    api_config_source:
      api_type: GRPC
      transport_api_version: V3
      grpc_services:
        - envoy_grpc:
            cluster_name: xds_cluster
  cds_config:
    resource_api_version: V3
    api_config_source:
      api_type: GRPC
      transport_api_version: V3
      grpc_services:
        - envoy_grpc:
            cluster_name: xds_cluster

# xDS control plane cluster
static_resources:
  clusters:
    - name: xds_cluster
      type: STRICT_DNS
      connect_timeout: 5s
      load_assignment:
        cluster_name: xds_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address: { address: control-plane, port_value: 18000 }
      typed_extension_protocol_options:
        envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
          "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
          explicit_http_config:
            http2_protocol_options: {}
```

## Circuit Breaking

### Configuration

```yaml
clusters:
  - name: backend
    circuit_breakers:
      thresholds:
        - priority: DEFAULT
          max_connections: 1024
          max_pending_requests: 1024
          max_requests: 1024
          max_retries: 3
          retry_budget:
            budget_percent:
              value: 20.0
            min_retry_concurrency: 3
```

## Rate Limiting

### Local rate limiting

```yaml
http_filters:
  - name: envoy.filters.http.local_ratelimit
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
      stat_prefix: http_local_rate_limiter
      token_bucket:
        max_tokens: 100
        tokens_per_fill: 100
        fill_interval: 60s
      filter_enabled:
        runtime_key: local_rate_limit_enabled
        default_value: { numerator: 100, denominator: HUNDRED }
      filter_enforced:
        runtime_key: local_rate_limit_enforced
        default_value: { numerator: 100, denominator: HUNDRED }
```

### Route-level rate limiting

```yaml
routes:
  - match: { prefix: "/api" }
    route:
      cluster: api_cluster
      rate_limits:
        - actions:
            - request_headers:
                header_name: "x-api-key"
                descriptor_key: "api_key"
            - remote_address: {}
```

## Admin Interface

### Querying stats and config

```bash
# Enable admin interface in bootstrap
# admin:
#   address:
#     socket_address: { address: 0.0.0.0, port_value: 9901 }

# View server info
curl http://localhost:9901/server_info

# View all stats
curl http://localhost:9901/stats

# Filter stats
curl "http://localhost:9901/stats?filter=cluster.backend"

# View clusters and their health
curl http://localhost:9901/clusters

# View current config dump
curl http://localhost:9901/config_dump

# Drain listeners (graceful shutdown)
curl -X POST http://localhost:9901/drain_listeners

# View active connections per listener
curl http://localhost:9901/listeners
```

## Tips

- Use `envoy --mode validate` before deploying config changes to catch syntax errors early.
- Start with static config, then migrate to xDS incrementally (CDS/EDS first, then LDS/RDS).
- Set `connect_timeout` on clusters to avoid hanging connections to dead backends.
- Use `STRICT_DNS` for Kubernetes services and `LOGICAL_DNS` for external hostnames with multiple IPs.
- Enable access logs with structured JSON format for machine-parseable observability.
- Circuit breakers protect the proxy itself — set `max_connections` based on upstream capacity, not downstream demand.
- The admin interface should never be exposed publicly; bind it to localhost or use network policies.
- Use retry budgets instead of fixed retry counts to prevent retry storms under high load.
- Watch the `upstream_cx_overflow` stat to detect when circuit breakers are tripping.
- Prefer GRPC xDS over REST for lower latency config updates and bidirectional streaming.
- Use `outlier_detection` on clusters to automatically eject unhealthy endpoints.
- Configure `idle_timeout` on connections to prevent resource leaks from stale connections.

## See Also

istio, linkerd, kubernetes, nginx, haproxy

## References

- [Envoy Proxy Documentation](https://www.envoyproxy.io/docs/envoy/latest/)
- [Envoy xDS Protocol](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol)
- [Envoy Architecture Overview](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/arch_overview)
- [Envoy Circuit Breaking](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/circuit_breaking)
- [Envoy Rate Limiting](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/rate_limit_filter)
- [Envoy GitHub Repository](https://github.com/envoyproxy/envoy)
- [xDS API Reference](https://www.envoyproxy.io/docs/envoy/latest/api-v3/api)
