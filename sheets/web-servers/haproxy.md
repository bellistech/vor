# HAProxy (Load Balancer & Proxy)

High-performance TCP/HTTP load balancer with health checking, SSL termination, and ACLs.

## Frontend

### Basic HTTP frontend

```bash
# frontend http_front
#     bind *:80
#     default_backend app_servers
```

### HTTPS frontend with SSL termination

```bash
# frontend https_front
#     bind *:443 ssl crt /etc/haproxy/certs/example.com.pem
#     http-request redirect scheme https unless { ssl_fc }
#     default_backend app_servers
```

### HTTP to HTTPS redirect

```bash
# frontend http_redirect
#     bind *:80
#     http-request redirect scheme https code 301
```

## Backend

### Basic backend

```bash
# backend app_servers
#     balance roundrobin
#     server app1 10.0.0.1:8080 check
#     server app2 10.0.0.2:8080 check
#     server app3 10.0.0.3:8080 check weight 2
```

### Backend with options

```bash
# backend app_servers
#     balance leastconn
#     option httpchk GET /healthz
#     http-check expect status 200
#     server app1 10.0.0.1:8080 check inter 5s fall 3 rise 2
#     server app2 10.0.0.2:8080 check inter 5s fall 3 rise 2
#     server app3 10.0.0.3:8080 check backup    # only used when others are down
```

## Load Balancing Algorithms

### Available algorithms

```bash
# balance roundrobin      # weighted round-robin (default)
# balance leastconn       # fewest connections (good for long sessions)
# balance source          # hash client IP (sticky)
# balance uri             # hash request URI (good for caching)
# balance hdr(Host)       # hash by header value
# balance first           # fill first server before using next
```

## Server Options

### Server line parameters

```bash
# server app1 10.0.0.1:8080 check        # enable health checks
#     weight 3                             # 3x traffic share
#     maxconn 100                          # max concurrent connections
#     inter 5s                             # check interval
#     fall 3                               # failures before marking down
#     rise 2                               # successes before marking up
#     backup                               # standby server
#     disabled                             # admin disabled
#     ssl verify none                      # backend SSL (no verify)
```

## ACLs & Routing

### Route by path

```bash
# frontend http_front
#     bind *:80
#     acl is_api path_beg /api
#     acl is_static path_end .css .js .png .jpg
#     use_backend api_servers if is_api
#     use_backend static_servers if is_static
#     default_backend app_servers
```

### Route by host

```bash
# acl is_blog hdr(Host) -i blog.example.com
# acl is_app hdr(Host) -i app.example.com
# use_backend blog_servers if is_blog
# use_backend app_servers if is_app
```

### Route by header

```bash
# acl is_websocket hdr(Upgrade) -i websocket
# use_backend ws_servers if is_websocket
```

### Block by IP

```bash
# acl blocked_ip src 192.168.1.100
# http-request deny if blocked_ip
```

## Health Checks

### HTTP health check

```bash
# backend app_servers
#     option httpchk GET /healthz HTTP/1.1\r\nHost:\ localhost
#     http-check expect status 200
#     server app1 10.0.0.1:8080 check
```

### TCP health check

```bash
# backend db_servers
#     option tcp-check
#     server db1 10.0.0.1:5432 check
```

## SSL

### SSL termination

```bash
# frontend https
#     bind *:443 ssl crt /etc/haproxy/certs/example.com.pem alpn h2,http/1.1
#     # pem file = cert + key concatenated
```

### SSL passthrough (no termination)

```bash
# frontend tcp_ssl
#     bind *:443
#     mode tcp
#     tcp-request inspect-delay 5s
#     tcp-request content accept if { req_ssl_hello_type 1 }
#     use_backend ssl_passthrough
#
# backend ssl_passthrough
#     mode tcp
#     server app1 10.0.0.1:443 check
```

### Backend SSL

```bash
# backend secure_app
#     server app1 10.0.0.1:8443 ssl verify none
```

## Stats Page

### Enable stats dashboard

```bash
# listen stats
#     bind *:8404
#     stats enable
#     stats uri /stats
#     stats refresh 10s
#     stats admin if LOCALHOST         # allow admin actions from localhost
#     stats auth admin:secretpassword
```

## Logging

### Configure syslog output

```bash
# global
#     log /dev/log local0
#     log /dev/log local1 notice
#
# defaults
#     log global
#     option httplog          # detailed HTTP logging
#     option dontlognull      # skip health check log noise
```

### Custom log format

```bash
# log-format "%ci:%cp [%tr] %ft %b/%s %TR/%Tw/%Tc/%Tr/%Ta %ST %B %CC %CS %tsc %ac/%fc/%bc/%sc/%rc %sq/%bq %hr %hs %{+Q}r"
```

## Stick Tables

### Session persistence by cookie

```bash
# backend app_servers
#     cookie SERVERID insert indirect nocache
#     server app1 10.0.0.1:8080 check cookie s1
#     server app2 10.0.0.2:8080 check cookie s2
```

### Rate limiting with stick tables

```bash
# frontend http_front
#     stick-table type ip size 100k expire 30s store http_req_rate(10s)
#     http-request track-sc0 src
#     http-request deny deny_status 429 if { sc_http_req_rate(0) gt 100 }
```

## Headers

### Add/modify headers

```bash
# http-request set-header X-Forwarded-Proto https if { ssl_fc }
# http-request set-header X-Real-IP %[src]
# http-response set-header X-Frame-Options DENY
# http-response del-header Server
```

## Operations

### Validate config

```bash
haproxy -c -f /etc/haproxy/haproxy.cfg
```

### Reload without dropping connections

```bash
systemctl reload haproxy
```

### Runtime socket commands

```bash
echo "show stat" | socat stdio /var/run/haproxy/admin.sock
echo "show info" | socat stdio /var/run/haproxy/admin.sock
echo "disable server app_servers/app1" | socat stdio /var/run/haproxy/admin.sock
echo "enable server app_servers/app1" | socat stdio /var/run/haproxy/admin.sock
```

## Tips

- Always validate config with `haproxy -c -f` before reloading.
- Concatenate cert + key into one `.pem` file for the `ssl crt` directive.
- `option httpchk` defaults to `OPTIONS /` if you do not specify a method and path.
- `inter 5s fall 3 rise 2` means: check every 5s, mark down after 3 failures, mark up after 2 successes.
- Use `mode tcp` for non-HTTP protocols (databases, SMTP, custom TCP).
- Stick tables with `http_req_rate` are a lightweight way to rate-limit without external tools.
- `haproxy -d -f haproxy.cfg` runs in debug mode (foreground, verbose) for troubleshooting.
- The stats page at `/stats` is the fastest way to see backend health and traffic distribution.

## See Also

- nginx
- caddy
- prometheus
- grafana
- socat

## References

- [HAProxy Documentation](https://docs.haproxy.org/)
- [HAProxy Configuration Manual](https://www.haproxy.org/download/2.8/doc/configuration.txt)
- [HAProxy Starter Guide](https://www.haproxy.org/download/2.8/doc/intro.txt)
- [HAProxy Management Guide](https://www.haproxy.org/download/2.8/doc/management.txt)
- [HAProxy Logging](https://www.haproxy.com/blog/introduction-to-haproxy-logging)
- [HAProxy ACL Documentation](https://docs.haproxy.org/2.8/configuration.html#7)
- [HAProxy Health Checks](https://www.haproxy.com/blog/how-to-enable-health-checks-in-haproxy)
- [HAProxy SSL/TLS Configuration](https://www.haproxy.com/blog/haproxy-ssl-termination)
- [HAProxy Enterprise Documentation](https://www.haproxy.com/documentation/)
- [HAProxy GitHub Repository](https://github.com/haproxy/haproxy)
- [HAProxy Runtime API](https://www.haproxy.org/download/2.8/doc/management.txt)
