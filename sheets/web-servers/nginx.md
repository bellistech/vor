# Nginx (Web Server & Reverse Proxy)

High-performance HTTP server, reverse proxy, and load balancer.

## Server Blocks

### Basic virtual host

```bash
# /etc/nginx/sites-available/example.com
# server {
#     listen 80;
#     server_name example.com www.example.com;
#     root /var/www/example.com/html;
#     index index.html;
# }
```

### Enable a site

```bash
ln -s /etc/nginx/sites-available/example.com /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
```

## Location Blocks

### Exact, prefix, and regex matching

```bash
# location = /healthz { return 200 "ok"; }       # exact match (highest priority)
# location /api/ { proxy_pass http://backend; }   # prefix match
# location ~* \.(jpg|png|gif)$ { expires 30d; }   # case-insensitive regex
# location ^~ /static/ { root /var/www; }         # prefix, skip regex check
```

### Try files with fallback

```bash
# location / {
#     try_files $uri $uri/ /index.html;
# }
```

## Reverse Proxy

### Basic proxy_pass

```bash
# location /api/ {
#     proxy_pass http://127.0.0.1:8080/;   # trailing slash strips /api/ prefix
#     proxy_set_header Host $host;
#     proxy_set_header X-Real-IP $remote_addr;
#     proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
#     proxy_set_header X-Forwarded-Proto $scheme;
# }
```

### WebSocket proxy

```bash
# location /ws/ {
#     proxy_pass http://127.0.0.1:8080;
#     proxy_http_version 1.1;
#     proxy_set_header Upgrade $http_upgrade;
#     proxy_set_header Connection "upgrade";
# }
```

## SSL/TLS

### HTTPS with Let's Encrypt

```bash
# server {
#     listen 443 ssl http2;
#     server_name example.com;
#     ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
#     ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;
#     ssl_protocols TLSv1.2 TLSv1.3;
#     ssl_ciphers HIGH:!aNULL:!MD5;
#     ssl_prefer_server_ciphers on;
# }
```

### HTTP to HTTPS redirect

```bash
# server {
#     listen 80;
#     server_name example.com;
#     return 301 https://$host$request_uri;
# }
```

## Load Balancing

### Upstream block

```bash
# upstream backend {
#     server 10.0.0.1:8080 weight=3;
#     server 10.0.0.2:8080;
#     server 10.0.0.3:8080 backup;
# }
# server {
#     location / {
#         proxy_pass http://backend;
#     }
# }
```

### Load balancing methods

```bash
# upstream backend {
#     least_conn;                  # fewest active connections
#     # ip_hash;                   # sticky sessions by client IP
#     # hash $request_uri consistent; # consistent hashing
#     server 10.0.0.1:8080;
#     server 10.0.0.2:8080;
# }
```

## Rewrite & Redirect

### Redirect

```bash
# location /old-page { return 301 /new-page; }
# location /gone { return 410; }
```

### Rewrite

```bash
# rewrite ^/blog/(\d+)/(.*)$ /articles/$1-$2 permanent;
# rewrite ^/user/(.+)$ /profile?name=$1 last;
```

## Headers

### Add security headers

```bash
# add_header X-Frame-Options DENY always;
# add_header X-Content-Type-Options nosniff always;
# add_header X-XSS-Protection "1; mode=block" always;
# add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
# add_header Content-Security-Policy "default-src 'self'" always;
```

### CORS headers

```bash
# location /api/ {
#     add_header Access-Control-Allow-Origin * always;
#     add_header Access-Control-Allow-Methods "GET, POST, OPTIONS" always;
#     add_header Access-Control-Allow-Headers "Authorization, Content-Type" always;
#     if ($request_method = OPTIONS) { return 204; }
# }
```

## Rate Limiting

### Limit request rate

```bash
# http {
#     limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
# }
# server {
#     location /api/ {
#         limit_req zone=api burst=20 nodelay;
#     }
# }
```

### Limit connections

```bash
# limit_conn_zone $binary_remote_addr zone=addr:10m;
# location /downloads/ {
#     limit_conn addr 5;
# }
```

## Caching

### Proxy cache

```bash
# http {
#     proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=my_cache:10m
#                      max_size=1g inactive=60m;
# }
# server {
#     location / {
#         proxy_cache my_cache;
#         proxy_cache_valid 200 302 10m;
#         proxy_cache_valid 404 1m;
#         proxy_pass http://backend;
#         add_header X-Cache-Status $upstream_cache_status;
#     }
# }
```

### Static file caching

```bash
# location ~* \.(css|js|jpg|png|woff2)$ {
#     expires 30d;
#     add_header Cache-Control "public, immutable";
# }
```

## Operations

### Test config and reload

```bash
nginx -t
nginx -T                          # dump full resolved config
systemctl reload nginx
```

### Access and error logs

```bash
tail -f /var/log/nginx/access.log
tail -f /var/log/nginx/error.log
```

## Tips

- Always run `nginx -t` before `systemctl reload nginx` to catch syntax errors.
- `proxy_pass http://backend/` (with trailing slash) strips the matched location prefix. Without it, the full URI is forwarded.
- `try_files` is cheaper than `if` blocks for serving static files with SPA fallback.
- Rate limiting `burst` without `nodelay` queues excess requests. With `nodelay`, excess requests within the burst are served immediately.
- Use `$binary_remote_addr` (4 bytes) instead of `$remote_addr` (7-15 bytes) in shared memory zones.
- `nginx -T` prints the full merged config, invaluable for debugging includes.
- The `map` directive is more efficient than multiple `if` blocks for variable-based routing.

## See Also

- haproxy
- caddy
- html
- css
- prometheus
- grafana

## References

- [nginx Documentation](https://nginx.org/en/docs/)
- [nginx Beginner's Guide](https://nginx.org/en/docs/beginners_guide.html)
- [nginx Directive Reference](https://nginx.org/en/docs/dirindex.html)
- [nginx Variable Reference](https://nginx.org/en/docs/varindex.html)
- [nginx Reverse Proxy Guide](https://nginx.org/en/docs/http/ngx_http_proxy_module.html)
- [nginx Load Balancing](https://nginx.org/en/docs/http/load_balancing.html)
- [nginx SSL/TLS Configuration](https://nginx.org/en/docs/http/configuring_https_servers.html)
- [nginx Location Block Processing](https://nginx.org/en/docs/http/ngx_http_core_module.html#location)
- [nginx Rate Limiting](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html)
- [nginx Admin Guide (F5)](https://docs.nginx.com/nginx/admin-guide/)
- [nginx GitHub Mirror](https://github.com/nginx/nginx)
