# Caddy (Web Server with Automatic HTTPS)

Modern web server that provisions and renews TLS certificates automatically.

## Caddyfile Basics

### Serve static files

```bash
# /etc/caddy/Caddyfile
# example.com {
#     root * /var/www/html
#     file_server
# }
```

### Multiple sites

```bash
# example.com {
#     root * /var/www/example
#     file_server
# }
# blog.example.com {
#     root * /var/www/blog
#     file_server
# }
```

### Localhost development (no TLS)

```bash
# :8080 {
#     root * /var/www/html
#     file_server browse    # enables directory listing
# }
```

## Reverse Proxy

### Basic reverse proxy

```bash
# example.com {
#     reverse_proxy localhost:8080
# }
```

### With path matching

```bash
# example.com {
#     reverse_proxy /api/* localhost:8080
#     reverse_proxy /ws/* localhost:8081
#     file_server
# }
```

### Load balancing

```bash
# example.com {
#     reverse_proxy localhost:8080 localhost:8081 localhost:8082 {
#         lb_policy round_robin      # or least_conn, ip_hash, first, random
#         health_uri /healthz
#         health_interval 10s
#     }
# }
```

### Headers to backend

```bash
# example.com {
#     reverse_proxy localhost:8080 {
#         header_up X-Real-IP {remote_host}
#         header_up X-Forwarded-For {remote_host}
#     }
# }
```

## Auto HTTPS & TLS

### Automatic (default behavior)

```bash
# Caddy auto-provisions certs from Let's Encrypt when you use a domain name.
# example.com {
#     reverse_proxy localhost:8080
# }
# That's it. TLS is on, HTTP->HTTPS redirect is automatic.
```

### Custom TLS settings

```bash
# example.com {
#     tls admin@example.com
#     reverse_proxy localhost:8080
# }
```

### Internal/self-signed TLS

```bash
# example.internal {
#     tls internal
#     reverse_proxy localhost:8080
# }
```

### Disable auto HTTPS

```bash
# http://example.com {
#     reverse_proxy localhost:8080
# }
```

### Use custom certificates

```bash
# example.com {
#     tls /etc/certs/cert.pem /etc/certs/key.pem
# }
```

## Headers

### Set response headers

```bash
# example.com {
#     header {
#         X-Frame-Options DENY
#         X-Content-Type-Options nosniff
#         Strict-Transport-Security "max-age=31536000; includeSubDomains"
#         -Server                    # remove Server header
#     }
#     reverse_proxy localhost:8080
# }
```

### Conditional headers

```bash
# example.com {
#     header /api/* {
#         Access-Control-Allow-Origin *
#         Access-Control-Allow-Methods "GET, POST, OPTIONS"
#     }
# }
```

## Rewrite & Redirect

### Rewrite (internal)

```bash
# example.com {
#     rewrite /old-path /new-path
#     rewrite /app/* /index.html    # SPA catch-all
# }
```

### Redirect (external)

```bash
# example.com {
#     redir /old-page /new-page 301
#     redir /docs https://docs.example.com{uri} 302
# }
```

## Handle & Route

### handle (first match wins, unordered)

```bash
# example.com {
#     handle /api/* {
#         reverse_proxy localhost:8080
#     }
#     handle {
#         root * /var/www/html
#         file_server
#     }
# }
```

### route (ordered, all directives evaluated)

```bash
# example.com {
#     route {
#         rewrite /app/* /index.html
#         file_server
#     }
# }
```

### handle_path (strips prefix)

```bash
# example.com {
#     handle_path /api/* {
#         reverse_proxy localhost:8080    # /api/users -> /users
#     }
# }
```

## Respond

### Static responses

```bash
# example.com {
#     respond /healthz 200 {
#         body "ok"
#     }
#     respond /robots.txt 200 {
#         body "User-agent: *\nDisallow: /admin"
#     }
# }
```

## Matchers

### Named matchers

```bash
# example.com {
#     @api path /api/*
#     @static path *.css *.js *.png *.jpg
#     reverse_proxy @api localhost:8080
#     header @static Cache-Control "max-age=2592000"
#     file_server
# }
```

### Method matcher

```bash
# @post method POST
# reverse_proxy @post localhost:8080
```

### Header matcher

```bash
# @websocket header Connection *Upgrade*
# @websocket header Upgrade websocket
# reverse_proxy @websocket localhost:8081
```

## Authentication

### Basic auth

```bash
# Hash a password first:
caddy hash-password --plaintext 'mypassword'
```

```bash
# example.com {
#     basicauth /admin/* {
#         admin $2a$14$...hashed...password...
#     }
#     reverse_proxy localhost:8080
# }
```

## Logging

### Access log

```bash
# example.com {
#     log {
#         output file /var/log/caddy/access.log
#         format json
#         level INFO
#     }
# }
```

## Operations

### Validate config

```bash
caddy validate --config /etc/caddy/Caddyfile
```

### Reload config

```bash
caddy reload --config /etc/caddy/Caddyfile
```

### Run in foreground

```bash
caddy run --config /etc/caddy/Caddyfile
```

### Format Caddyfile

```bash
caddy fmt --overwrite /etc/caddy/Caddyfile
```

### Reverse proxy one-liner (no config file)

```bash
caddy reverse-proxy --from :2015 --to localhost:8080
```

### File server one-liner

```bash
caddy file-server --root /var/www/html --listen :8080
```

## Tips

- Caddy gets TLS certificates automatically for any site with a public domain name. No configuration needed.
- Use `http://` prefix in the site address to disable automatic HTTPS for development.
- `handle` picks the first matching block (like nginx location). `route` processes directives in order.
- `handle_path` strips the matched prefix before forwarding, saving you from rewrite rules.
- `caddy fmt` normalizes your Caddyfile indentation and spacing.
- The JSON config API at `localhost:2019` allows live config changes without reload.
- Caddy stores certificates in `~/.local/share/caddy/` (or `$XDG_DATA_HOME/caddy/`).
- Use `tls internal` for internal services to get self-signed certs without Let's Encrypt.

## References

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Caddyfile Concepts](https://caddyserver.com/docs/caddyfile/concepts)
- [Caddyfile Directives](https://caddyserver.com/docs/caddyfile/directives)
- [Caddy reverse_proxy Directive](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy)
- [Caddy TLS / Automatic HTTPS](https://caddyserver.com/docs/automatic-https)
- [Caddy Request Matchers](https://caddyserver.com/docs/caddyfile/matchers)
- [Caddy JSON API](https://caddyserver.com/docs/api)
- [Caddy Modules and Plugins](https://caddyserver.com/docs/modules/)
- [Caddy Quick Starts](https://caddyserver.com/docs/quick-starts)
- [Caddy GitHub Repository](https://github.com/caddyserver/caddy)
- [Caddy Community Forum](https://caddy.community/)
