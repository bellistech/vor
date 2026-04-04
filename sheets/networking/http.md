# HTTP (Hypertext Transfer Protocol)

Application-layer request-response protocol for distributed hypermedia systems, evolving from HTTP/1.1 text-based pipelining through HTTP/2 binary multiplexing to HTTP/3 QUIC-based zero-RTT transport.

## HTTP/1.1 Methods

```
Method    Safe  Idempotent  Body     Description
──────────────────────────────────────────────────────────────────
GET       Yes   Yes         No       Retrieve resource
HEAD      Yes   Yes         No       GET without body (headers only)
POST      No    No          Yes      Submit data / create resource
PUT       No    Yes         Yes      Replace resource entirely
PATCH     No    No          Yes      Partial update of resource
DELETE    No    Yes         Maybe    Remove resource
OPTIONS   Yes   Yes         No       Describe communication options
TRACE     Yes   Yes         No       Loop-back diagnostic
CONNECT   No    No          No       Establish tunnel (HTTPS proxy)
```

## Status Codes

### 1xx Informational

```
100 Continue           — Client should continue with request body
101 Switching Protocols — Upgrading to WebSocket/H2
103 Early Hints        — Preload resources before final response (RFC 8297)
```

### 2xx Success

```
200 OK                 — Standard success
201 Created            — Resource created (POST/PUT), Location header set
204 No Content         — Success with no body (DELETE, PUT)
206 Partial Content    — Range request fulfilled (downloads, video seeking)
```

### 3xx Redirection

```
301 Moved Permanently  — Permanent redirect (cacheable, may change method to GET)
302 Found              — Temporary redirect (commonly changes to GET)
303 See Other          — Redirect after POST, always use GET
304 Not Modified       — Conditional GET, use cached version
307 Temporary Redirect — Temporary, preserves method
308 Permanent Redirect — Permanent, preserves method (RFC 7538)
```

### 4xx Client Error

```
400 Bad Request        — Malformed syntax
401 Unauthorized       — Authentication required (send WWW-Authenticate)
403 Forbidden          — Authenticated but not authorized
404 Not Found          — Resource does not exist
405 Method Not Allowed — Method not supported for this resource
408 Request Timeout    — Client took too long
409 Conflict           — State conflict (e.g., edit collision)
413 Payload Too Large  — Request body exceeds server limit
415 Unsupported Media  — Content-Type not accepted
422 Unprocessable      — Valid syntax but semantic errors (WebDAV, widely adopted)
429 Too Many Requests  — Rate limited (check Retry-After header)
```

### 5xx Server Error

```
500 Internal Error     — Unhandled server exception
502 Bad Gateway        — Upstream server sent invalid response
503 Service Unavail    — Server overloaded or in maintenance
504 Gateway Timeout    — Upstream server did not respond in time
```

## Essential Headers

```bash
# Request headers
Host: example.com                           # Required in HTTP/1.1
Accept: application/json, text/html;q=0.9   # Content negotiation
Authorization: Bearer eyJhbG...             # Authentication token
Content-Type: application/json              # Request body format
User-Agent: curl/8.0                        # Client identification
If-None-Match: "abc123"                     # Conditional GET (ETag)
If-Modified-Since: Thu, 01 Jan 2025 00:00   # Conditional GET (date)
Range: bytes=0-1023                         # Partial content request

# Response headers
Content-Type: application/json; charset=utf-8
Content-Length: 1234
Cache-Control: public, max-age=3600         # Caching directive
ETag: "abc123"                              # Entity tag for conditionals
Last-Modified: Thu, 01 Jan 2025 00:00       # For If-Modified-Since
Location: /new-path                         # Redirect target
Retry-After: 60                             # Seconds to wait (429/503)
Set-Cookie: session=abc; HttpOnly; Secure   # Set browser cookie
Strict-Transport-Security: max-age=31536000 # HSTS
```

## HTTP/2

```
# Binary framing — all communication via frames on a single TCP connection
# Multiplexing — multiple streams (requests) on one connection
# Header compression — HPACK reduces header overhead
# Server push — server can proactively send resources
# Stream prioritization — weight + dependency tree

# Frame types
DATA          — Carries request/response body
HEADERS       — Opens a stream, carries compressed headers
PRIORITY      — Stream priority (deprecated in RFC 9113)
RST_STREAM    — Immediately terminate a single stream
SETTINGS      — Connection-level configuration
PUSH_PROMISE  — Server push notification
PING          — Keepalive / RTT measurement
GOAWAY        — Graceful shutdown of connection
WINDOW_UPDATE — Flow control (per-stream and per-connection)
CONTINUATION  — Continuation of HEADERS if too large for one frame
```

```bash
# Test HTTP/2 with curl
curl -I --http2 https://example.com
curl -v --http2-prior-knowledge http://localhost:8080  # h2c (cleartext)

# Check HTTP/2 support
curl -sI https://example.com | grep -i "http/2"

# Negotiate via ALPN (TLS)
openssl s_client -connect example.com:443 -alpn h2
```

## HTTP/3 + QUIC

```
# HTTP/3 uses QUIC (UDP-based transport) instead of TCP
# Benefits:
#   - No head-of-line blocking (stream-level loss isolation)
#   - 0-RTT connection establishment (resumed connections)
#   - Connection migration (survives IP changes, e.g., WiFi → cellular)
#   - Built-in TLS 1.3 (no separate TLS handshake)

# QUIC uses UDP port 443
# Discovered via Alt-Svc header or HTTPS DNS record
```

```bash
# Test HTTP/3 with curl (requires --http3 flag, curl 7.88+)
curl --http3 https://example.com

# Check Alt-Svc header for HTTP/3 advertisement
curl -sI https://example.com | grep -i alt-svc
# Alt-Svc: h3=":443"; ma=86400

# HTTPS DNS record (SVCB/HTTPS RR type 65)
dig HTTPS example.com
```

## Caching

```bash
# Cache-Control directives (request or response)
Cache-Control: public                # Any cache can store
Cache-Control: private               # Only browser cache, not CDN/proxy
Cache-Control: no-cache              # Must revalidate before using
Cache-Control: no-store              # Never cache (sensitive data)
Cache-Control: max-age=3600          # Fresh for 1 hour
Cache-Control: s-maxage=86400        # CDN/proxy max age (overrides max-age)
Cache-Control: must-revalidate       # After stale, must check origin
Cache-Control: immutable             # Never changes (versioned assets)
Cache-Control: stale-while-revalidate=60  # Serve stale while refreshing

# Conditional requests
# ETag-based:
#   Server: ETag: "abc123"
#   Client: If-None-Match: "abc123"
#   Server: 304 Not Modified (or 200 with new body)

# Date-based:
#   Server: Last-Modified: Thu, 01 Jan 2025 00:00:00 GMT
#   Client: If-Modified-Since: Thu, 01 Jan 2025 00:00:00 GMT
#   Server: 304 Not Modified (or 200 with new body)
```

## curl Examples

```bash
# GET request
curl https://api.example.com/users

# POST JSON
curl -X POST https://api.example.com/users \
    -H "Content-Type: application/json" \
    -d '{"name": "Alice", "email": "alice@example.com"}'

# PUT with file
curl -X PUT https://api.example.com/users/1 \
    -H "Content-Type: application/json" \
    -d @user.json

# DELETE
curl -X DELETE https://api.example.com/users/1

# Show response headers
curl -I https://example.com              # HEAD request
curl -i https://example.com              # Include headers in output
curl -v https://example.com              # Verbose (all headers + TLS)

# Authentication
curl -u user:pass https://api.example.com          # Basic auth
curl -H "Authorization: Bearer TOKEN" https://api.example.com

# Follow redirects
curl -L https://example.com

# Download file with progress
curl -O https://example.com/file.tar.gz
curl -o output.tar.gz https://example.com/file.tar.gz

# Resume interrupted download
curl -C - -O https://example.com/large-file.iso

# Rate limit
curl --limit-rate 1M https://example.com/file.tar.gz -O

# Timing info
curl -o /dev/null -s -w "DNS: %{time_namelookup}s\nConnect: %{time_connect}s\nTLS: %{time_appconnect}s\nTTFB: %{time_starttransfer}s\nTotal: %{time_total}s\n" https://example.com
```

## Keep-Alive & Connection Management

```bash
# HTTP/1.1 persistent connections (default: Connection: keep-alive)
# Close after request: Connection: close

# HTTP/1.1 pipelining (rarely used — head-of-line blocking)
# Multiple requests sent before responses — but responses must be in order

# HTTP/2 multiplexing — no pipelining needed, all streams concurrent
# Single TCP connection, typically 100+ concurrent streams

# Connection limits per domain (browser defaults)
# HTTP/1.1: 6-8 connections per domain
# HTTP/2: 1 connection per domain (multiplexed)
```

## Tips

- HTTP/2 multiplexes all requests over a single TCP connection. This means one lost packet stalls ALL streams (TCP head-of-line blocking). HTTP/3 solves this by using QUIC (UDP), where packet loss only stalls the affected stream.
- `Cache-Control: no-cache` does NOT mean "don't cache." It means "cache it but revalidate every time." Use `no-store` to truly prevent caching of sensitive responses.
- A `301` redirect can be cached indefinitely by browsers. Use `308` for permanent redirects that must preserve the HTTP method (POST stays POST). Use `307` for temporary method-preserving redirects.
- HTTP/2 Server Push was widely deployed but rarely useful in practice. Chrome removed support in 2022. Use `103 Early Hints` instead to hint the browser about resources to preload.
- The `429 Too Many Requests` response should always include a `Retry-After` header (seconds or HTTP-date). Without it, clients have no guidance on when to retry and will hammer the server.
- curl's `-w` format string is invaluable for debugging latency. The difference between `time_connect` and `time_appconnect` is the TLS handshake duration. TTFB (`time_starttransfer`) minus `time_appconnect` is server processing time.
- HTTP/2 HPACK header compression maintains a dynamic table per connection. For many short-lived connections, the compression benefit is minimal. Long-lived connections compress much better.
- QUIC's 0-RTT resumption sends data in the first packet, but it is vulnerable to replay attacks. Servers must ensure 0-RTT data is idempotent (GETs, not POSTs with side effects).
- When debugging HTTP/2 issues, use `curl -v --http2` and look for `SETTINGS`, `WINDOW_UPDATE`, and `GOAWAY` frames. A `GOAWAY` with error code 0 is graceful shutdown; nonzero indicates a protocol error.
- Content negotiation via `Accept` headers with quality values (`q=0.9`) lets clients express format preferences. Servers should return `406 Not Acceptable` if they cannot satisfy any offered type, but most just return their default format.

## See Also

- curl, tcp, quic, dns, iptables, wget

## References

- [RFC 9110 — HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110)
- [RFC 9112 — HTTP/1.1](https://www.rfc-editor.org/rfc/rfc9112)
- [RFC 9113 — HTTP/2](https://www.rfc-editor.org/rfc/rfc9113)
- [RFC 9114 — HTTP/3](https://www.rfc-editor.org/rfc/rfc9114)
- [RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport](https://www.rfc-editor.org/rfc/rfc9000)
- [RFC 7541 — HPACK: Header Compression for HTTP/2](https://www.rfc-editor.org/rfc/rfc7541)
- [RFC 8297 — Early Hints (103)](https://www.rfc-editor.org/rfc/rfc8297)
- [MDN — HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP)
- [curl man page](https://curl.se/docs/manpage.html)
