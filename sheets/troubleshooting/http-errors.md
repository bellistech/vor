# HTTP Errors

Every HTTP status code, stream error, and protocol-level failure mode — with cause, when-to-return, diagnostic hint, and the fix. The quick reference for "what does this code actually mean?" and "why is my client/server doing this?"

## Setup

HTTP is a request/response protocol. A client (browser, curl, library) opens a connection to a server, sends a request consisting of a request-line, headers, and an optional body. The server replies with a status-line, headers, and an optional body.

```http
GET /api/users/42 HTTP/1.1
Host: example.com
Accept: application/json
Authorization: Bearer eyJhbGciOi...

HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 81
Cache-Control: max-age=60

{"id":42,"name":"Alice","email":"a@example.com"}
```

The status-line is `HTTP-version SP status-code SP reason-phrase`. The reason-phrase is informational only — clients MUST act on the numeric status code.

### HTTP/1.1 vs HTTP/2 vs HTTP/3

HTTP/1.1 is plain text framed by `\r\n`. One request per connection at a time (pipelining exists but is broken in practice). Browsers open 6 connections per origin to parallelize.

HTTP/2 (RFC 9113) is binary framing over a single TCP connection. Streams are multiplexed — many concurrent requests on one connection — and headers are HPACK-compressed. Head-of-line blocking still exists at the TCP level: one lost packet stalls all streams on that connection.

HTTP/3 (RFC 9114) runs on QUIC over UDP. QUIC implements its own reliability + congestion control + TLS 1.3, and provides independent streams so loss in one stream doesn't block others. 0-RTT resumption gives faster handshakes after the first connection.

```text
HTTP/1.1: text, one request at a time per connection, multiple connections per origin
HTTP/2:   binary, multiplexed over single TCP, HPACK headers, TCP head-of-line blocking
HTTP/3:   binary, multiplexed over QUIC/UDP, QPACK headers, no head-of-line blocking
```

### Idempotent vs non-idempotent methods

Idempotent methods can be safely retried: `GET`, `HEAD`, `PUT`, `DELETE`, `OPTIONS`, `TRACE`. Non-idempotent: `POST`, `PATCH`. Clients (browsers, libraries, intermediaries) MAY automatically retry idempotent requests on connection failure; they MUST NOT auto-retry POST without explicit signal.

Method semantics matter for redirects: 307/308 preserve the method, 301/302/303 historically downgrade POST → GET.

```text
Safe       — read-only, no side effects: GET, HEAD, OPTIONS
Idempotent — same effect if repeated:    GET, HEAD, PUT, DELETE, OPTIONS, TRACE
Cacheable  — response can be cached:     GET, HEAD, POST (rarely)
```

## Status Code Categories

Five categories, identified by the first digit:

```text
1xx  Informational  — provisional response, expect more
2xx  Success        — request was understood and accepted
3xx  Redirection    — further action required to complete
4xx  Client Error   — request was malformed or refused
5xx  Server Error   — server failed to fulfill a valid request
```

Mental model: **fail closed at 4xx, fail open at 5xx**. A 4xx is the client's fault; the client should not retry without changing the request. A 5xx is the server's fault; the client MAY retry (idempotently). Load balancers, retries, circuit breakers, and SLO dashboards all hinge on this distinction. Returning a 5xx for what is really a client mistake (missing field, bad format) makes every monitoring tool light up; returning a 4xx for a real server bug hides incidents.

```text
Rule of thumb:
  Don't return 500 for "user typed the wrong password" — that's 401.
  Don't return 400 for "database is down" — that's 503 or 500.
  Don't return 200 with {"error": "..."} — that's a lie to every monitor.
```

## 1xx Informational

### 100 Continue

Used when the client sends `Expect: 100-continue` and the server agrees to receive the body. Lets the client avoid uploading a large body if the headers alone would be rejected.

```http
PUT /upload HTTP/1.1
Host: example.com
Content-Length: 1073741824
Expect: 100-continue

HTTP/1.1 100 Continue

[client now sends body]

HTTP/1.1 201 Created
```

Diagnostic hint: if a client hangs after sending headers, check whether the server understands `Expect: 100-continue`. Some intermediaries strip the header; some servers ignore it and the client waits forever.

### 101 Switching Protocols

The server is changing protocols at the client's request. Most common with WebSocket and HTTP/2 cleartext upgrade.

```http
GET /chat HTTP/1.1
Host: example.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13

HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

Diagnostic hint: if a WebSocket fails to upgrade, look for missing `Connection: Upgrade` (some proxies strip hop-by-hop headers).

### 102 Processing (WebDAV)

Server has accepted the request but processing is not complete. Tells the client not to time out. Largely superseded by 103 and chunked responses.

### 103 Early Hints

Server sends preload hints (Link headers) before the final response, letting the browser begin loading critical resources while the server still computes the body.

```http
HTTP/1.1 103 Early Hints
Link: </styles.css>; rel=preload; as=style
Link: </app.js>; rel=preload; as=script

HTTP/1.1 200 OK
Content-Type: text/html
...
```

When to return: for slow-to-render pages where critical assets are known up front. Cloudflare and Fastly support it; not all middleware does.

## 2xx Success

### 200 OK

The standard success. The body contains the response.

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status": "ok"}
```

Common misuse: returning 200 with `{"error": "..."}` in the body — see "The 200-with-error-body Anti-Pattern."

### 201 Created

A new resource was created as a result of this request. SHOULD include a `Location` header pointing at the new resource.

```http
POST /api/users HTTP/1.1
Content-Type: application/json

{"name": "Alice"}

HTTP/1.1 201 Created
Location: /api/users/42
Content-Type: application/json

{"id": 42, "name": "Alice"}
```

Diagnostic hint: 201 without `Location` is technically allowed but a smell — clients can't easily fetch what they just made.

### 202 Accepted

Request has been accepted for processing but is not complete. Used for async APIs.

```http
POST /api/jobs HTTP/1.1

HTTP/1.1 202 Accepted
Location: /api/jobs/abc-123/status
```

The client polls the Location URL to check status. Some APIs include `Retry-After` to suggest poll interval.

### 203 Non-Authoritative Information

The response was modified by an intermediary (e.g., a transforming proxy). Rare in modern systems.

### 204 No Content

Success, but the response body is intentionally empty. Common after `DELETE` and `PUT`. Clients MUST NOT expect a body.

```http
DELETE /api/users/42 HTTP/1.1

HTTP/1.1 204 No Content
```

Gotcha: do not set `Content-Length: <nonzero>` on a 204. Don't include a body. Some HTTP/2 implementations will close the stream with a protocol error if you do.

### 205 Reset Content

Server tells the client to reset its view (e.g., clear the form after submission). Rarely used.

### 206 Partial Content

Response to a range request. Includes a `Content-Range` header.

```http
GET /video.mp4 HTTP/1.1
Range: bytes=1024-2047

HTTP/1.1 206 Partial Content
Content-Range: bytes 1024-2047/12345678
Content-Length: 1024
```

Multi-range responses use `multipart/byteranges`. Most clients ask for one range at a time.

### 207 Multi-Status (WebDAV)

XML body containing multiple status codes for sub-operations. WebDAV-specific.

### 208 Already Reported (WebDAV) / 226 IM Used

Niche. Don't return these from a regular API.

## 3xx Redirection

### 301 Moved Permanently

The resource has a new permanent URI. Clients SHOULD update bookmarks. Historically, many clients changed POST → GET on 301; modern clients may preserve the method, but for portability use 308 if you want POST preserved.

```http
HTTP/1.1 301 Moved Permanently
Location: https://example.com/new-path
```

SEO: search engines transfer ranking signals from the old URL to the new. The 301 is the "tell Google you moved" status.

### 302 Found

Temporary redirect. Originally meant "preserve method"; in practice all major clients downgraded POST → GET, which became codified as 303. Modern guidance: do not use 302 for new APIs; use 303 (force GET) or 307 (preserve method) explicitly.

```http
HTTP/1.1 302 Found
Location: /login
```

This is "the original sin of HTTP" — the spec said one thing, browsers did another, and we got two new codes (303, 307) to disambiguate.

### 303 See Other

Force the client to GET the Location URL, regardless of the original method. The proper response to a successful POST-then-redirect (Post/Redirect/Get pattern).

```http
POST /comments HTTP/1.1
Content-Type: application/x-www-form-urlencoded

text=Hello

HTTP/1.1 303 See Other
Location: /comments
```

The browser GETs `/comments` — refresh-safe (no double POST).

### 304 Not Modified

Used with conditional requests. The cached copy is still valid; client should serve from cache. No body.

```http
GET /api/users/42 HTTP/1.1
If-None-Match: "abc123"

HTTP/1.1 304 Not Modified
ETag: "abc123"
Cache-Control: max-age=60
```

Diagnostic hint: if a CDN keeps fetching from origin, check that the ETag/Last-Modified is stable across requests (the "ETag mismatch causing infinite revalidation" pattern).

### 305 Use Proxy / 306 (unused)

Deprecated. Don't use.

### 307 Temporary Redirect

Like 302, but explicitly preserves the method. POST stays POST, PUT stays PUT.

```http
POST /api/foo HTTP/1.1

HTTP/1.1 307 Temporary Redirect
Location: /api/v2/foo
```

The client will replay the POST to `/api/v2/foo` with the same body.

### 308 Permanent Redirect

Like 301, but explicitly preserves the method. SEO-equivalent to 301; method-preserving.

```http
HTTP/1.1 308 Permanent Redirect
Location: https://example.com/new-path
```

### Method-preservation matrix

```text
Code  Method-change-on-redirect
301   MAY change (legacy: usually GET)
302   MAY change (legacy: usually GET)
303   MUST become GET
307   MUST be preserved
308   MUST be preserved
```

Modern recommendation:
- Permanent + force GET: **301** (SEO benefit, accept legacy behaviour)
- Permanent + preserve method: **308**
- Temporary + force GET: **303**
- Temporary + preserve method: **307**
- Avoid **302** in new code.

### SEO implications

- 301: search engines transfer PageRank/link equity to the new URL (after a delay).
- 302: officially does not transfer ranking; treated as "this is temporary, keep indexing the old."
- 308: same as 301 for SEO; method-preserving for crawlers that POST.
- 303 / 307: typically not used for SEO-critical redirects.

## 4xx Client Errors

### 400 Bad Request

Generic client error. The server can't parse or understand the request. Use it for malformed JSON, missing required headers, broken framing.

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error": "invalid JSON: unexpected character at offset 12"}
```

Diagnostic hint: many APIs over-use 400 for everything, including validation errors that are better expressed as 422. See "The 422 vs 400 Distinction."

### 401 Unauthorized

The request requires authentication and either none was provided or it failed. The name is misleading — this means **unauthenticated**, not "unauthorized." MUST include a `WWW-Authenticate` header to tell the client how to authenticate.

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="api", error="invalid_token", error_description="The access token expired"
```

Common misuse: returning 401 without `WWW-Authenticate`, leaving the client guessing the auth scheme.

### 402 Payment Required

Reserved for future use. Some APIs use it for quota / billing failures (Stripe, GitHub for billing-suspended accounts).

```http
HTTP/1.1 402 Payment Required
Content-Type: application/json

{"error": "billing_required", "message": "Subscription has lapsed"}
```

### 403 Forbidden

The client is authenticated, but does not have permission to perform this operation. The "I know who you are; you can't do this" code.

```http
HTTP/1.1 403 Forbidden
Content-Type: application/json

{"error": "permission_denied", "required_scope": "admin"}
```

Diagnostic hint: do not use 403 to hide existence — that's what 404 is for. Returning 403 leaks "this resource exists, you just can't see it."

### 404 Not Found

The resource doesn't exist, or the server is hiding it. Spec allows the server to use 404 instead of 403 to avoid revealing the existence of a resource.

```http
HTTP/1.1 404 Not Found
Content-Type: application/json

{"error": "not_found"}
```

Common misuse: returning HTML 404 pages from a JSON API. The client expects JSON; the proxy or framework injected an HTML error page. Set up your error handler to honour the request's `Accept` header.

### 405 Method Not Allowed

The resource exists, but doesn't support this method. MUST include an `Allow` header listing supported methods.

```http
DELETE /api/users HTTP/1.1

HTTP/1.1 405 Method Not Allowed
Allow: GET, POST
Content-Type: application/json

{"error": "method_not_allowed"}
```

### 406 Not Acceptable

The client's `Accept` header demands a content type the server can't produce.

```http
GET /api/users/42 HTTP/1.1
Accept: application/xml

HTTP/1.1 406 Not Acceptable
Content-Type: application/json

{"error": "only_json_supported"}
```

Many APIs ignore Accept and just return JSON anyway; that's pragmatic but not strictly correct.

### 407 Proxy Authentication Required

Like 401, but for the proxy (not the origin server). MUST include `Proxy-Authenticate` header.

```http
HTTP/1.1 407 Proxy Authentication Required
Proxy-Authenticate: Basic realm="proxy"
```

### 408 Request Timeout

The server timed out waiting for the client to finish sending the request. Often a connection that was opened but the client didn't send anything (or sent slowly).

```http
HTTP/1.1 408 Request Timeout
Connection: close
```

Diagnostic hint: high 408 rates suggest slow clients on bad networks (mobile), Slowloris-style attacks, or your read timeouts are too aggressive.

### 409 Conflict

The request conflicts with the current state of the resource. Common causes: optimistic concurrency violation (with `If-Match`), uniqueness constraint, race conditions.

```http
PUT /api/users/42 HTTP/1.1
If-Match: "v1"

HTTP/1.1 409 Conflict
Content-Type: application/json

{"error": "etag_mismatch", "current_etag": "v3"}
```

### 410 Gone

The resource was here, and is now permanently removed. Use this when you know something existed and is intentionally deleted; use 404 when the resource never existed or you don't track removals.

```http
HTTP/1.1 410 Gone
```

Search engines will deindex 410'd URLs faster than 404'd ones.

### 411 Length Required

`Content-Length` is missing on a request that needs one. Rare with modern clients; some old proxies still emit this for chunked-encoded uploads they don't understand.

### 412 Precondition Failed

A conditional request (`If-Match`, `If-Unmodified-Since`, `If-None-Match` on non-GET) failed.

```http
PUT /api/users/42 HTTP/1.1
If-Match: "v1"

HTTP/1.1 412 Precondition Failed
ETag: "v3"
```

### 413 Payload Too Large

The request body is bigger than the server is willing to process. Was previously called "Request Entity Too Large." Server SHOULD include `Retry-After` if the limit is temporary (rare).

```http
HTTP/1.1 413 Payload Too Large
Content-Type: application/json

{"error": "max_size_bytes": 10485760}
```

Often raised by nginx (`client_max_body_size`), Apache (`LimitRequestBody`), or load balancers before the request even reaches the app.

### 414 URI Too Long

The request URI is longer than the server will process. Default limits: nginx 8 KB; Apache 8 KB. Common when query strings get unbounded.

### 415 Unsupported Media Type

The `Content-Type` of the request body is not supported. Common cause: a client sending `Content-Type: application/x-www-form-urlencoded` to an API that only accepts JSON.

```http
POST /api/users HTTP/1.1
Content-Type: text/plain

HTTP/1.1 415 Unsupported Media Type
Accept-Post: application/json
```

### 416 Range Not Satisfiable

The range request asks for bytes outside the file. Server SHOULD include a `Content-Range` header indicating the file size.

```http
HTTP/1.1 416 Range Not Satisfiable
Content-Range: bytes */1024
```

### 417 Expectation Failed

The server can't fulfill the `Expect` header. Effectively only used to reject `Expect: 100-continue` (rare; usually the server just doesn't honour Expect).

### 418 I'm a teapot

April 1 1998 RFC 2324 joke. Some sites still return it (Google for `/teapot`). Don't use in production; some middleware treats it as a real error.

### 421 Misdirected Request

The request was directed to a server that can't produce a response. Used in HTTP/2 when a connection is reused for an authority the server isn't certified for. The client should retry on a fresh connection.

### 422 Unprocessable Entity (Unprocessable Content)

Originally WebDAV. In modern API design: "the syntax is fine, but the semantics are wrong." The canonical validation-error code in many REST styles.

```http
POST /api/users HTTP/1.1
Content-Type: application/json

{"email": "not-an-email"}

HTTP/1.1 422 Unprocessable Entity
Content-Type: application/json

{"errors": {"email": ["is not a valid email address"]}}
```

### 423 Locked (WebDAV)

The resource is locked. WebDAV-specific; some lock-style APIs reuse it.

### 424 Failed Dependency (WebDAV)

The request failed because a previous request depended on it failed.

### 425 Too Early

Server is unwilling to risk processing a 0-RTT request that might be replayed. Returned by TLS 1.3 / QUIC servers when the client retried with 0-RTT data on a non-idempotent request.

### 426 Upgrade Required

The server requires a different protocol — typically TLS or HTTP/2. MUST include `Upgrade` header.

```http
HTTP/1.1 426 Upgrade Required
Upgrade: TLS/1.2, HTTP/1.1
Connection: Upgrade
```

### 428 Precondition Required

The server requires the request to be conditional (e.g., `If-Match`). Used to prevent the lost-update problem.

### 429 Too Many Requests

The client has been rate limited. SHOULD include `Retry-After` (in seconds, or HTTP-date).

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 30
Content-Type: application/json

{"error": "rate_limited", "limit": 100, "window_seconds": 60}
```

GitHub and Stripe also include `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` (Unix timestamp).

### 431 Request Header Fields Too Large

Request headers are too big. Common cause: a Cookie header with too many cookies (browsers will keep them; server will refuse).

### 451 Unavailable For Legal Reasons

The resource is blocked due to legal demand (DMCA, court order, GDPR right-to-be-forgotten). RFC 7725. Number is a Bradbury reference.

```http
HTTP/1.1 451 Unavailable For Legal Reasons
Link: <https://example.com/legal/takedown>; rel="blocked-by"
```

## The 401 vs 403 Distinction

```text
401 Unauthorized — really means UNAUTHENTICATED
   "I don't know who you are. Send credentials."
   MUST include WWW-Authenticate.

403 Forbidden — AUTHENTICATED but unauthorized
   "I know who you are. You don't have permission."
   No WWW-Authenticate (you're already authenticated).
```

Common misuse:
- Spring Security defaults to 403 even when the user isn't authenticated. Configure it to return 401 + `WWW-Authenticate` for unauthenticated requests.
- Django's `LoginRequired` defaults to a 302 redirect to `/accounts/login/`; for an API you want a 401 + JSON.
- Returning 403 to hide existence: leaks "this exists, you can't see it." Prefer 404.

```text
Decision flow:
  No credentials supplied?      → 401 + WWW-Authenticate
  Bad credentials?              → 401 + WWW-Authenticate
  Token expired?                → 401 + WWW-Authenticate (error="invalid_token")
  Authenticated, missing scope? → 403
  Authenticated, wrong tenant?  → 404 (don't leak existence)
```

## The 422 vs 400 Distinction

```text
400 Bad Request
   "Your bytes don't form a valid HTTP request."
   "Your JSON doesn't parse."
   "Required header is missing or malformed."
   Generic syntactic failure. Pragmatic catch-all.

422 Unprocessable Entity
   "Your bytes parse fine. The semantics are wrong."
   "Email field has invalid format."
   "Quantity must be > 0."
   "Foreign key references a non-existent record."
   Validation failure where the request was structurally correct.
```

Two camps in practice:
- **Strict REST:** parse errors → 400; validation errors → 422.
- **Pragmatic:** everything client-side → 400 with structured error body. Easier to teach, fewer arguments at code review.

Both are acceptable. Pick one and apply consistently across the API. The worst is "sometimes 400, sometimes 422, depending on which middleware caught the error first."

## The 200-with-error-body Anti-Pattern

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"success": false, "error": "user_not_found"}
```

This pattern is ubiquitous in legacy SOAP-flavored APIs and some modern frameworks (especially when the framework's default error path returns 200 + body).

Why it's broken:
- **Monitoring:** every load balancer, APM, log aggregator, and dashboard counts 200s as success. Your error rate looks like 0%, the user experience is "everything is broken."
- **Retries:** clients won't retry a 200. A transient backend failure dressed up as 200 won't recover.
- **Caching:** intermediaries may cache the 200, including its error body, for the duration of `Cache-Control`.
- **Browsers:** `fetch().then(r => ...)` doesn't reject on 200; you have to manually inspect the body, every call.

Modern API design: use the right status code. A 4xx for client errors; 5xx for server errors. Put structured error info in the body, but signal "this is an error" via the status line.

```http
Bad:
HTTP/1.1 200 OK
{"ok": false, "error": "validation"}

Good:
HTTP/1.1 422 Unprocessable Entity
{"errors": {"email": ["required"]}}
```

## 5xx Server Errors

### 500 Internal Server Error

Generic server error. Bug, panic, unhandled exception, NPE, division by zero. The server was supposed to handle this case and didn't.

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json

{"error": "internal_error", "request_id": "req_abc123"}
```

Best practice: include a request ID the client can quote in support tickets; log a stack trace server-side keyed by the same ID; never leak internals to the client.

### 501 Not Implemented

The server doesn't support the request method (e.g., the server doesn't implement PATCH at all). Distinct from 405 (this specific resource doesn't support this method).

### 502 Bad Gateway

A proxy or gateway received an invalid response from upstream — the upstream returned malformed HTTP, closed the connection mid-response, or sent garbage.

```http
HTTP/1.1 502 Bad Gateway
```

Diagnostic hint: 502 = upstream **misbehaved**. Check upstream logs for crashes, panics, restarts. Common causes:
- Upstream process crashed and restarted mid-request.
- Upstream listening on the wrong port.
- TLS misconfiguration between proxy and upstream.
- Upstream sending HTTP/1.0 to a proxy expecting HTTP/1.1.

### 503 Service Unavailable

The server is currently unable to handle the request — temporarily overloaded, in maintenance, or scaling. SHOULD include `Retry-After`.

```http
HTTP/1.1 503 Service Unavailable
Retry-After: 120
Content-Type: application/json

{"error": "maintenance", "estimated_recovery_seconds": 120}
```

This is the right code for graceful degradation and circuit-breaker open states. Don't use 500 when you mean "we're full, try later."

### 504 Gateway Timeout

Proxy/gateway timed out waiting for upstream. The upstream never replied (or the response wasn't complete within timeout).

```http
HTTP/1.1 504 Gateway Timeout
```

Diagnostic hint: 504 = upstream is **silent**. Check upstream resource saturation: CPU, threads, DB connections, lock contention, slow queries.

### 505 HTTP Version Not Supported

The server doesn't speak the requested HTTP version. Rare in modern web; can appear when an HTTP/2-only server gets HTTP/0.9 traffic.

### 506 Variant Also Negotiates / 507 Insufficient Storage / 508 Loop Detected (WebDAV)

Niche. 507 is sometimes seen on object stores when quota is exhausted. 508 fires when the server detects an infinite loop in WebDAV traversal.

### 510 Not Extended

Used with HTTP `Extensions` framework (effectively dead).

### 511 Network Authentication Required

The client must authenticate with the network (not the origin) to gain access. Used by captive portals — coffee shop WiFi, hotel networks.

```http
HTTP/1.1 511 Network Authentication Required
Content-Type: text/html

<html><body>Click <a href="https://wifi.hotel.com/login">here</a> to log in.</body></html>
```

Not all captive portals use this; many just intercept HTTP and rewrite responses (which is itself a problem for HTTPS).

## The 502 vs 504 Distinction

```text
502 Bad Gateway
   Upstream replied — with garbage, an empty response, or closed mid-stream.
   Diagnose: upstream logs (panics, OOM, restarts), TLS misconfig, version mismatch.

504 Gateway Timeout
   Upstream never replied within the proxy's timeout window.
   Diagnose: upstream saturation — CPU, threads, DB pool, slow queries.
```

```text
nginx -> app:
  app crashed mid-request    → 502
  app slow-but-eventually    → 504 (if proxy_read_timeout exceeded)
  app accepted then RST'd    → 502
  app returned HTTP/0.9      → 502 (proxy can't parse)
  proxy can't even connect   → 502 (upstream prematurely closed)
```

## The 503 Done Right

A 503 should always be paired with `Retry-After`. The whole point is to tell the client when to come back.

```http
HTTP/1.1 503 Service Unavailable
Retry-After: 30
Content-Type: application/json

{"error": "overloaded", "queue_length": 1024}
```

```http
HTTP/1.1 503 Service Unavailable
Retry-After: Wed, 21 Oct 2026 07:28:00 GMT
```

Status-page integration patterns:
- Static page on a separate origin, served when the app is hard-down.
- Health check endpoint that load balancers poll; remove unhealthy nodes from rotation.
- Graceful drain: when shutting down, return 503 + `Connection: close` for in-flight requests.
- Maintenance window: serve 503 with HTML pointing to status page; API clients get JSON.

```bash
# nginx maintenance mode
location / {
  return 503;
}
error_page 503 /maintenance.html;
location = /maintenance.html {
  root /var/www;
  internal;
}
```

## Conditional Requests

Conditional headers let clients say "give me this resource only if X." The server returns 304/412 if the condition isn't met.

### If-Match (with ETag) — optimistic concurrency

```http
PUT /api/users/42 HTTP/1.1
If-Match: "v1"
Content-Type: application/json

{"name": "Alice"}

HTTP/1.1 412 Precondition Failed
ETag: "v3"
```

Used to prevent the lost-update problem: if the resource has changed since the client read it, the PUT fails. The client refetches, merges, retries.

### If-None-Match — cache validation

```http
GET /api/users/42 HTTP/1.1
If-None-Match: "v3"

HTTP/1.1 304 Not Modified
ETag: "v3"
Cache-Control: max-age=60
```

Server returns 304 (no body) if the ETag matches; 200 with body otherwise.

### If-Modified-Since / If-Unmodified-Since

```http
GET /static/file.css HTTP/1.1
If-Modified-Since: Wed, 21 Oct 2026 07:28:00 GMT

HTTP/1.1 304 Not Modified
Last-Modified: Wed, 21 Oct 2026 07:28:00 GMT
```

ETag is preferred over Last-Modified — ETag has 1-byte resolution; Last-Modified has 1-second resolution and breaks for sub-second updates.

### The 304 Not Modified flow

```text
Client                          Server
  |                                |
  |  GET /foo                      |
  |  If-None-Match: "abc"          |
  |------------------------------->|
  |                                | Compute current ETag
  |                                | "abc" matches?
  |                                |
  |  304 Not Modified              |
  |  ETag: "abc"                   |
  |<-------------------------------|
  |                                |
  |  Use cached body               |
```

### The Vary header

`Vary` tells caches that the response depends on the listed request headers. Without `Vary`, a CDN may return the wrong response (e.g., English text to a French-speaking client).

```http
GET /home HTTP/1.1
Accept-Language: en

HTTP/1.1 200 OK
Vary: Accept-Language
Content-Type: text/html
```

Without `Vary`, the CDN keys on the URL alone and returns whatever it cached first — possibly the wrong language. With `Vary: Accept-Language`, the cache keys on URL + Accept-Language.

```text
Common Vary headers:
  Vary: Accept            (content negotiation)
  Vary: Accept-Encoding   (gzip vs identity)
  Vary: Accept-Language   (i18n)
  Vary: Cookie            (per-user; effectively defeats shared caching)
  Vary: Origin            (CORS)
  Vary: Authorization     (per-user; defeats shared caching)
```

## CORS Errors

Cross-Origin Resource Sharing. The browser is the one enforcing CORS — the server has no way to "fix" it server-side except by sending the right headers.

### The preflight (OPTIONS) flow

For non-simple cross-origin requests, the browser sends an OPTIONS preflight first:

```http
OPTIONS /api/users HTTP/1.1
Origin: https://app.example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Content-Type, Authorization

HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 86400
```

If the preflight passes, the browser sends the real request. If not, it never even tries.

### Simple vs preflighted requests

A "simple" request:
- Method: GET, HEAD, POST.
- Content-Type: `application/x-www-form-urlencoded`, `multipart/form-data`, or `text/plain` only.
- No custom headers (Authorization, X-*, etc.).

Anything else is preflighted.

### Common CORS errors (browser console)

```text
Access to fetch at 'https://api.example.com/foo' from origin 'https://app.example.com'
has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present
on the requested resource.
```
Fix: server must respond with `Access-Control-Allow-Origin: https://app.example.com` (or `*` if no credentials).

```text
Access to fetch at 'https://api.example.com/foo' from origin 'https://app.example.com'
has been blocked by CORS policy: Response to preflight request doesn't pass access
control check: It does not have HTTP ok status.
```
Fix: the OPTIONS handler returned 4xx or 5xx. The OPTIONS endpoint must return 200/204.

```text
Access to fetch at 'https://api.example.com/foo' from origin 'https://app.example.com'
has been blocked by CORS policy: The 'Access-Control-Allow-Origin' header has a
value 'https://OTHER.example.com' that is not equal to the supplied origin.
```
Fix: server is echoing a hardcoded origin. Echo the request's Origin header (from an allowlist).

```text
Access to fetch at 'https://api.example.com/foo' from origin 'https://app.example.com'
has been blocked by CORS policy: Request header field x-trace-id is not allowed by
Access-Control-Allow-Headers in preflight response.
```
Fix: add `x-trace-id` to `Access-Control-Allow-Headers`.

```text
Access to fetch at 'https://api.example.com/foo' from origin 'https://app.example.com'
has been blocked by CORS policy: The value of the 'Access-Control-Allow-Credentials'
header in the response is '' which must be 'true' when the request's credentials
mode is 'include'.
```
Fix: send `Access-Control-Allow-Credentials: true` AND make sure `Access-Control-Allow-Origin` is a specific origin (not `*`).

```text
Access to fetch at 'https://api.example.com/foo' from origin 'https://app.example.com'
has been blocked by CORS policy: The 'Access-Control-Allow-Origin' header contains
the invalid value 'null'.
```
Fix: usually means the request came from a `file://` page or a sandboxed iframe. Don't treat null as an allowlisted origin.

### Cookies + CORS + SameSite

For cross-origin requests to send cookies:
1. Client must use `credentials: 'include'` (fetch) or `withCredentials = true` (XHR).
2. Server must send `Access-Control-Allow-Credentials: true`.
3. Server must send `Access-Control-Allow-Origin: <specific origin>` (not `*`).
4. Cookies must have `SameSite=None` and `Secure` (HTTPS).

If any of these is missing, cookies don't go.

## Cookie Errors / Gotchas

```http
Set-Cookie: session=abc; Domain=example.com; Path=/; Secure; HttpOnly; SameSite=Lax
```

### SameSite=None requires Secure

```http
Set-Cookie: session=abc; SameSite=None
```
Browser silently rejects: "SameSite=None must be paired with Secure" since Chrome 80 (Feb 2020).

Fix:
```http
Set-Cookie: session=abc; SameSite=None; Secure
```

### HttpOnly + JavaScript

```js
// document.cookie won't see this:
// Set-Cookie: session=abc; HttpOnly
console.log(document.cookie); // empty or only non-HttpOnly cookies
```

That's the point — HttpOnly defends against XSS theft. If you need JS to read it, you've got an architectural problem (use a separate non-sensitive cookie for read access).

### Domain attribute semantics

```http
Set-Cookie: foo=1; Domain=example.com   # sent to example.com AND sub.example.com
Set-Cookie: foo=1; Domain=.example.com  # same as above (legacy leading dot)
Set-Cookie: foo=1                       # sent only to the host that set it
```

The leading dot is legacy; modern browsers treat `example.com` and `.example.com` identically.

Cookies with `Domain` attribute are sent to all subdomains. To pin a cookie to one host only, omit `Domain`.

### Path-prefixed cookies

```http
Set-Cookie: a=1; Path=/admin
```
Sent to `/admin`, `/admin/foo`, `/admin/bar` — but **not** to `/`. If you load `/login` and look at `document.cookie`, `a` is missing. Path-prefixed cookies are a frequent source of "set on /a, not visible on /b" bugs.

### Third-party cookie phase-out

Browsers (Safari, Firefox, Chrome) are phasing out third-party cookies. Cookies on requests to a domain different from the page's URL bar domain are blocked or partitioned (CHIPS). Migrate to first-party patterns: same-origin auth, OAuth with PKCE, partitioned cookies for legitimate cross-site state.

### __Host- and __Secure- prefixes

```http
Set-Cookie: __Host-id=abc; Secure; Path=/; SameSite=Lax
Set-Cookie: __Secure-id=abc; Secure; SameSite=Lax
```

`__Host-` cookies must be: Secure, Path=/, no Domain attribute. Pinned to one host, can't be overwritten by subdomains.
`__Secure-` cookies must be: Secure. Less strict than `__Host-`.

These prefixes are enforced by the browser — a non-conforming Set-Cookie is rejected.

## HTTP/2 Stream Errors (RFC 9113)

When an HTTP/2 stream fails, the server (or client) sends a RST_STREAM frame with an error code. These codes are also used in GOAWAY frames at the connection level.

```text
0x0  NO_ERROR              — graceful close (used in GOAWAY)
0x1  PROTOCOL_ERROR        — peer violated the HTTP/2 spec
0x2  INTERNAL_ERROR        — implementation error in the peer
0x3  FLOW_CONTROL_ERROR    — peer violated flow-control protocol
0x4  SETTINGS_TIMEOUT      — peer didn't ACK SETTINGS frame in time
0x5  STREAM_CLOSED         — frame received on already-closed stream
0x6  FRAME_SIZE_ERROR      — frame size invalid (too large or wrong size for type)
0x7  REFUSED_STREAM        — server refused to process the stream (safe to retry)
0x8  CANCEL                — peer canceled the stream
0x9  COMPRESSION_ERROR     — HPACK decoding error (state corrupted)
0xa  CONNECT_ERROR         — TCP connection for CONNECT method failed
0xb  ENHANCE_YOUR_CALM     — peer is sending traffic too aggressively
0xc  INADEQUATE_SECURITY   — TLS does not meet minimum requirements
0xd  HTTP_1_1_REQUIRED     — endpoint requires HTTP/1.1 (e.g., for upgrade flows)
```

Diagnostic hints:
- `REFUSED_STREAM (0x7)`: the server didn't process anything; client MAY safely retry on a new stream/connection. Common when a server hits MAX_CONCURRENT_STREAMS limit.
- `ENHANCE_YOUR_CALM (0xb)`: rate limiting at the protocol level. Slow down.
- `COMPRESSION_ERROR (0x9)`: HPACK is stateful — once corrupted, the connection is unrecoverable. Open a new connection.
- `INADEQUATE_SECURITY (0xc)`: cipher suite is too weak (e.g., RC4, TLS < 1.2).
- `HTTP_1_1_REQUIRED (0xd)`: rare but seen with WebSocket-over-HTTP/2 not yet supported. Fall back.

## HTTP/3 / QUIC Errors

QUIC defines connection-level errors; HTTP/3 (RFC 9114) defines an application-error space.

### QUIC connection errors (transport)

```text
0x0  NO_ERROR
0x1  INTERNAL_ERROR
0x2  CONNECTION_REFUSED
0x3  FLOW_CONTROL_ERROR
0x4  STREAM_LIMIT_ERROR
0x5  STREAM_STATE_ERROR
0x6  FINAL_SIZE_ERROR
0x7  FRAME_ENCODING_ERROR
0x8  TRANSPORT_PARAMETER_ERROR
0x9  CONNECTION_ID_LIMIT_ERROR
0xa  PROTOCOL_VIOLATION
0xb  INVALID_TOKEN
0xc  APPLICATION_ERROR
0xd  CRYPTO_BUFFER_EXCEEDED
0xe  KEY_UPDATE_ERROR
0xf  AEAD_LIMIT_REACHED
0x10 NO_VIABLE_PATH
0x100-0x1ff  CRYPTO_ERROR (TLS alert + 0x100)
```

### HTTP/3 application errors

```text
0x100 H3_NO_ERROR              — graceful close
0x101 H3_GENERAL_PROTOCOL_ERROR
0x102 H3_INTERNAL_ERROR
0x103 H3_STREAM_CREATION_ERROR
0x104 H3_CLOSED_CRITICAL_STREAM
0x105 H3_FRAME_UNEXPECTED       — frame on wrong stream type
0x106 H3_FRAME_ERROR
0x107 H3_EXCESSIVE_LOAD         — peer too aggressive
0x108 H3_ID_ERROR               — invalid stream/push ID
0x109 H3_SETTINGS_ERROR
0x10a H3_MISSING_SETTINGS
0x10b H3_REQUEST_REJECTED       — server rejected before processing
0x10c H3_REQUEST_CANCELLED
0x10d H3_REQUEST_INCOMPLETE
0x10e H3_MESSAGE_ERROR
0x10f H3_CONNECT_ERROR
0x110 H3_VERSION_FALLBACK       — fall back to HTTP/2
```

Diagnostic hints:
- `H3_REQUEST_REJECTED (0x10b)`: server didn't begin processing — safe to retry.
- `H3_VERSION_FALLBACK (0x110)`: server tells client to retry over HTTP/2. Fall back via Alt-Svc.
- `CRYPTO_ERROR` is QUIC's way of carrying TLS alerts. Subtract 0x100 to get the TLS alert code.

## WebSocket Close Codes (RFC 6455)

When a WebSocket closes, the closing party sends a Close frame containing a 16-bit code.

```text
1000  Normal closure
1001  Going away (page navigated, server shutting down)
1002  Protocol error
1003  Unsupported data type (e.g., binary on text-only endpoint)
1004  Reserved
1005  No status received (reserved; not sent on the wire)
1006  Abnormal closure (reserved; not sent — connection dropped)
1007  Invalid frame payload data (e.g., non-UTF-8 in text frame)
1008  Policy violation (generic reject)
1009  Message too big
1010  Mandatory extension (client expected, server didn't agree)
1011  Internal server error
1012  Service restart (non-standard but seen)
1013  Try again later (non-standard but seen)
1014  Bad gateway (non-standard but seen)
1015  TLS handshake failure (reserved; not sent)
3000-3999  Registered (libraries / frameworks)
4000-4999  Application-defined
```

Diagnostic hints:
- `1006`: TCP connection dropped without a Close frame. Check network, proxy timeouts, idle disconnects.
- `1011`: server bug. Check server logs.
- `1009`: message exceeded server's max frame size. Check server config (`max_frame_size`).
- `1008`: server-defined policy reject. Body of the Close frame may have details.
- `1015`: TLS error during the upgrade — not a WebSocket-level error per se.

```js
ws.addEventListener('close', (e) => {
  console.log('code:', e.code, 'reason:', e.reason, 'wasClean:', e.wasClean);
});
```

## Caching Headers — Common Errors

### max-age vs s-maxage

```http
Cache-Control: max-age=60, s-maxage=300
```
- `max-age` — applies to all caches (browser, CDN, proxy).
- `s-maxage` — applies only to **shared** caches (CDN, reverse proxy). Overrides max-age for those.

Use case: cache for 5 minutes at the CDN (`s-maxage=300`) but revalidate every 60 seconds at the browser (`max-age=60`).

### public vs private vs no-cache vs no-store

```text
public         — any cache may store this response (default for cacheable methods)
private        — only the user's own cache (browser) may store; CDN must not
no-cache       — caches MAY store, but MUST revalidate before serving
no-store       — caches MUST NOT store the response at all
must-revalidate — once stale, MUST revalidate (don't serve stale-on-error)
immutable      — content will never change for the URL's lifetime; don't even revalidate
```

Common confusion: `no-cache` is **not** "don't cache" — it's "always revalidate before reuse." The "don't cache" directive is `no-store`.

```http
# I really don't want this stored:
Cache-Control: no-store

# I want it cached but always revalidated:
Cache-Control: no-cache

# Old-school "don't cache" (less reliable):
Cache-Control: no-store, no-cache, must-revalidate
Pragma: no-cache
Expires: 0
```

### must-revalidate vs immutable

```http
# CSS/JS with hash in filename, will never change:
Cache-Control: max-age=31536000, immutable

# API response, cached but always check:
Cache-Control: max-age=0, must-revalidate
```

`immutable` tells the browser "don't even revalidate on F5" — saves the conditional request.

### Vary consequences for CDN caching

```http
Cache-Control: public, max-age=300
Vary: Cookie
```

`Vary: Cookie` means every distinct Cookie header creates a separate cache entry. Effectively kills CDN caching for any user-cookied page. Use `Vary` only on headers that genuinely affect the response.

Common bug: setting `Vary: User-Agent` (every UA gets its own cache entry — your hit rate plummets).

### The "ETag mismatch causing infinite revalidation" pattern

Symptom: the client sends `If-None-Match: "v1"`, the server returns 200 with `ETag: "v1"` instead of 304.

Causes:
- ETag is being recomputed each request and includes a timestamp or random salt.
- ETag is computed differently after compression (`gzip` etag != identity etag); some servers strip etag suffixes incorrectly.
- Multiple servers behind a load balancer compute etags from instance-local state — the etag the client cached was from server A, the next request hits server B.

Fix: ETag must be a deterministic function of response content. Use a hash of the body (after canonicalization).

## Auth Header Patterns

### Bearer (OAuth, JWT)

```http
GET /api/me HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiI0MiJ9.signature
```

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="api", error="invalid_token", error_description="Token expired"
```

The `error` codes are RFC 6750:
- `invalid_request` — malformed Authorization header.
- `invalid_token` — bad signature, expired, etc.
- `insufficient_scope` — token is fine but lacks required scope.

### Basic

```http
GET /api/me HTTP/1.1
Authorization: Basic dXNlcjpwYXNz
```

`dXNlcjpwYXNz` is base64 of `user:pass`. **Not** encryption — only encoding. Use only over TLS.

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="api"
```

Browsers will pop a native auth dialog when they see `WWW-Authenticate: Basic`. To suppress that on AJAX endpoints, omit the header (less correct) or use a non-Basic scheme.

### The 401 + WWW-Authenticate retry-after-auth pattern

```text
Client                          Server
  |                                |
  |  GET /private                  |
  |------------------------------->|
  |                                |
  |  401 Unauthorized              |
  |  WWW-Authenticate: Bearer …    |
  |<-------------------------------|
  |                                |
  |  [obtain token]                |
  |                                |
  |  GET /private                  |
  |  Authorization: Bearer abc     |
  |------------------------------->|
  |                                |
  |  200 OK                        |
  |<-------------------------------|
```

OAuth 2.0 dance: the 401 includes hints about which authorization server / scopes are needed via the `WWW-Authenticate` header parameters.

## Curl Exit Codes

When troubleshooting, `curl` exits with a non-zero code on failure. Relevant codes:

```text
0   OK
1   Unsupported protocol
2   Failed to initialize
3   URL malformed
5   Couldn't resolve proxy
6   Couldn't resolve host (DNS failure)
7   Failed to connect to host (TCP/TLS connect failure)
22  HTTP page not retrieved (with --fail/-f and 4xx/5xx)
23  Write error
26  Read error
27  Out of memory
28  Operation timeout
35  SSL connect error
51  Server certificate has issues (cert verification failed)
52  The server didn't reply anything (empty response)
55  Failed sending network data
56  Failure receiving network data
58  Problem with the local certificate
60  Peer certificate cannot be authenticated with given CA cert
61  Unrecognized transfer encoding
67  Login denied
77  Problem with the SSL CA cert
```

Diagnostic flags:

```bash
curl -v https://example.com         # verbose: show TLS handshake + headers
curl -i https://example.com         # include response headers in output
curl -I https://example.com         # HEAD only — just headers
curl --fail https://example.com     # exit 22 on 4xx/5xx
curl --fail-with-body https://...   # exit 22 on 4xx/5xx but still print body
curl --resolve host:443:1.2.3.4 \   # bypass DNS (useful for testing)
     https://host/
curl --http2-prior-knowledge ...    # force HTTP/2 over TLS
curl --http3 ...                    # force HTTP/3
curl -k https://...                 # skip TLS verification (DEBUG ONLY)
curl --trace-ascii - https://...    # full trace including bodies
curl -w "%{http_code} %{time_total}\n" -o /dev/null -s https://...
```

## Common Server-Side Logs Patterns

### nginx error.log

```text
upstream timed out (110: Connection timed out) while connecting to upstream
   → 504. Upstream not accepting connections within proxy_connect_timeout.
   Fix: increase timeout, or scale upstream, or check firewall.

upstream prematurely closed connection while reading response header from upstream
   → 502. Upstream closed before sending complete response.
   Fix: upstream is crashing; check app logs.

no live upstreams while connecting to upstream
   → 502. All upstreams are marked down by health checks.
   Fix: upstream health check failing; restart upstream.

client intended to send too large body
   → 413. Increase client_max_body_size.

upstream sent too big header while reading response header
   → 502. proxy_buffer_size too small. Increase.

SSL_do_handshake() failed
   → TLS error to upstream.
   Fix: cert / SNI / version mismatch upstream.
```

### nginx access.log

```text
$status field in log_format
192.0.2.1 - - [01/Jan/2026:00:00:00 +0000] "GET / HTTP/1.1" 200 5321 ...
                                                          ^^^ status
```

Useful patterns:
```bash
# Top error codes:
awk '{print $9}' access.log | sort | uniq -c | sort -rn

# Slowest requests:
awk '$NF > 1.0 {print}' access.log

# 5xx rate:
awk '$9 ~ /^5/' access.log | wc -l
```

### Apache error.log

```text
[error] [client X] End of script output before headers
   → 500. CGI exited without producing headers.

[error] [client X] (104)Connection reset by peer
   → 502. Upstream/peer closed connection.

[error] [client X] client denied by server configuration
   → 403. Require directive failed.

[error] AH01102: error reading status line from remote server
   → 502. Upstream sent invalid response.
```

### Caddy

Structured JSON; query with `jq`:
```bash
journalctl -u caddy -o cat | jq 'select(.status >= 500)'
journalctl -u caddy -o cat | jq 'select(.duration > 1)'
```

### HAProxy

```text
[01/Jan/2026:00:00:00.000] frontend backend/server 0/0/0/100/100 200 ...
                                                   ^Tq/Tw/Tc/Tr/Tt
```
- `Tq` — time to receive request.
- `Tw` — time spent in queue.
- `Tc` — time to connect to server.
- `Tr` — time to receive response from server.
- `Tt` — total session time.

Termination flags appear at end of log line (e.g., `cD--` = client-side timeout, `sD--` = server-side timeout).

## Common Errors When Self-Implementing HTTP

### Missing Content-Length when no Transfer-Encoding

Per HTTP/1.1, a response with a body MUST have either `Content-Length` or `Transfer-Encoding: chunked`. Some clients tolerate "neither" (they read until close); most middleware treats it as a protocol error.

```text
Bad:  HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{"x":1}
                                          ^ no Content-Length, not chunked

Good: HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 8\r\n\r\n{"x":1}\n
Or:   HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n7\r\n{"x":1}\r\n0\r\n\r\n
```

### Mixing Content-Length with Transfer-Encoding: chunked

```http
HTTP/1.1 200 OK
Content-Length: 100
Transfer-Encoding: chunked

7
{"x":1}
0
```

This is **request smuggling territory**. Some intermediaries trust Content-Length, some trust Transfer-Encoding. If a frontend and backend disagree, an attacker can sneak a second request inside the first. RFC 9112 says: if both are present, ignore Content-Length **and** treat the message as suspicious. Many implementations now reject outright.

### Not consuming request body before sending response

If the server sends the response and closes without reading the request body, the client may interpret the close as an error. For HTTP/1.1 with keep-alive, the server must consume (or `Connection: close`).

```text
Bad: server reads headers, decides 400, sends response, closes
     -> client may have been mid-upload; sees "broken pipe"

Good: server reads body fully (or sends Connection: close), then 400
```

### Connection: keep-alive vs Connection: close

HTTP/1.1 default is keep-alive (persistent connections). HTTP/1.0 default is close. To opt-out of keep-alive, send `Connection: close` and the peer must close after this exchange.

```http
HTTP/1.1 200 OK
Content-Length: 81
Connection: close
```

### HTTP/1.0 vs HTTP/1.1

- HTTP/1.0 has no `Host` header — one host per IP.
- HTTP/1.0 has no chunked encoding.
- HTTP/1.0 default is `Connection: close`; HTTP/1.1 default is keep-alive.
- HTTP/1.0 has no 100-continue.

If you see `HTTP/1.0` on the wire today, it's usually a broken proxy or a hand-rolled client. Most modern clients are 1.1+ even when they don't advertise it.

## HTTP Smuggling (CL.TE / TE.CL / TE.TE)

When a frontend (proxy/CDN) and backend (app server) disagree on how to parse a request, an attacker can sneak a second request inside the first.

### CL.TE — Frontend uses Content-Length; Backend uses Transfer-Encoding

```http
POST / HTTP/1.1
Host: vulnerable.com
Content-Length: 13
Transfer-Encoding: chunked

0

SMUGGLED
```

Frontend reads 13 bytes (the whole thing including `SMUGGLED`). Backend sees `Transfer-Encoding: chunked`, reads `0\r\n\r\n` as end-of-body, and treats `SMUGGLED` as the start of a new request.

### TE.CL — Frontend uses Transfer-Encoding; Backend uses Content-Length

```http
POST / HTTP/1.1
Host: vulnerable.com
Content-Length: 4
Transfer-Encoding: chunked

5c
GPOST / HTTP/1.1
Host: vulnerable.com
…
0

```

Frontend processes the chunked body (proper request); backend reads 4 bytes per Content-Length and treats the rest as a new request.

### TE.TE — Both use Transfer-Encoding, but parse differently

```http
Transfer-Encoding: chunked
Transfer-Encoding: x
```
Or with whitespace tricks:
```http
Transfer-Encoding : chunked
```
Or hex tricks. One side honours both, the other strips one. Disagreement → smuggling.

### Defenses

- **Reject ambiguous requests.** RFC 9112: if both CL and TE are present, treat as invalid.
- **Normalize at the frontend.** The frontend should rewrite the request to a single canonical form before forwarding.
- **HTTP/2 to backend.** HTTP/2 has explicit framing — no Content-Length/Transfer-Encoding ambiguity.
- **Use modern, well-tested proxies.** Smuggling vulnerabilities have been found in nginx, Apache, Squid, IIS, Akamai, etc.; keep them patched.

## Common Gotchas — Broken → Fixed

### 1. 200 with error body when 4xx/5xx fits

```http
Bad:
HTTP/1.1 200 OK
Content-Type: application/json
{"error": "User not found"}

Good:
HTTP/1.1 404 Not Found
Content-Type: application/json
{"error": "user_not_found", "user_id": 42}
```

### 2. 401 when 403 was correct (and vice versa)

```text
Bad: returning 403 when no credentials were supplied
     (client doesn't know it should authenticate)

Good: 401 + WWW-Authenticate when no/bad auth
      403 only when authenticated but lacking permission
```

### 3. 422 vs 400 confusion

```text
Bad: sometimes 400, sometimes 422, depending on which middleware caught the error

Good: pick a convention and document it:
      - 400 for parse errors and malformed framing
      - 422 for validation errors on syntactically valid input
      OR
      - everything client-side is 400 with a structured error body
```

### 4. Forgetting Retry-After on 429/503

```http
Bad:
HTTP/1.1 429 Too Many Requests
{"error": "rate_limited"}

Good:
HTTP/1.1 429 Too Many Requests
Retry-After: 30
{"error": "rate_limited", "retry_after_seconds": 30}
```

### 5. Not setting WWW-Authenticate on 401

```http
Bad:
HTTP/1.1 401 Unauthorized

Good:
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer realm="api", error="invalid_token"
```

### 6. Cache-Control: no-cache when no-store was meant

```http
Bad: # I want this never cached
Cache-Control: no-cache

Good:
Cache-Control: no-store
```

### 7. Cookie without SameSite

```http
Bad:
Set-Cookie: session=abc

Good:
Set-Cookie: session=abc; Secure; HttpOnly; SameSite=Lax
```

(Modern browsers default to `SameSite=Lax` since Chrome 80, but be explicit.)

### 8. Returning HTML 404 from JSON API

```http
Bad: # API endpoint returns the framework's default HTML 404
HTTP/1.1 404 Not Found
Content-Type: text/html
<html>...</html>

Good:
HTTP/1.1 404 Not Found
Content-Type: application/json
{"error": "not_found"}
```

### 9. Mixing 301 and 302

```text
Bad: 301 for "site is in maintenance, redirect to status page"
     (browser caches the 301 forever; site comes back, users still hit status page)

Good: 302 (or 307) for temporary redirects
      301/308 only for genuinely permanent moves
```

### 10. Missing Vary on content-negotiated endpoints

```http
Bad:
GET /home (with Accept: en)
HTTP/1.1 200 OK
Cache-Control: public, max-age=300
Content-Type: text/html
[English content]

# Then GET /home (with Accept: fr) hits the CDN cache, gets English.

Good:
HTTP/1.1 200 OK
Vary: Accept-Language
Cache-Control: public, max-age=300
Content-Type: text/html
```

### 11. CORS Allow-Origin: * with Allow-Credentials: true

```http
Bad: # browser will reject this combination
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true

Good:
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Credentials: true
Vary: Origin
```

### 12. 204 with non-empty body or Content-Length > 0

```http
Bad:
HTTP/1.1 204 No Content
Content-Length: 8
Content-Type: application/json
{"ok":1}

Good:
HTTP/1.1 204 No Content
```

(Or use 200 if you actually have a body.)

### 13. Content-Type missing on JSON responses

```http
Bad:
HTTP/1.1 200 OK
{"foo":1}
# Browser sees text/plain, may render as text or trigger MIME sniffing.

Good:
HTTP/1.1 200 OK
Content-Type: application/json
{"foo":1}
```

### 14. Setting cookies without Secure on HTTPS

```http
Bad:
Set-Cookie: session=abc; HttpOnly

Good (over HTTPS):
Set-Cookie: session=abc; HttpOnly; Secure; SameSite=Lax
```

### 15. ETag without quotes

```http
Bad:
ETag: abc123

Good:
ETag: "abc123"           # strong validator
ETag: W/"abc123"         # weak validator
```

The quotes are part of the syntax.

## Diagnostic Tools

### curl

```bash
# Verbose: show TLS, headers, body
curl -v https://example.com

# Headers only
curl -I https://example.com

# Show response headers + body
curl -i https://example.com

# Timing breakdown
curl -w "@-" -o /dev/null -s https://example.com <<'EOF'
   namelookup:  %{time_namelookup}s
      connect:  %{time_connect}s
   appconnect:  %{time_appconnect}s
  pretransfer:  %{time_pretransfer}s
     redirect:  %{time_redirect}s
starttransfer:  %{time_starttransfer}s
        total:  %{time_total}s
EOF

# Force HTTP version
curl --http1.1 ...
curl --http2 ...
curl --http3 ...

# Bypass DNS for testing
curl --resolve api.example.com:443:1.2.3.4 https://api.example.com/

# Save and examine cookies
curl -c jar.txt -b jar.txt https://example.com

# POST JSON
curl -X POST -H "Content-Type: application/json" -d '{"x":1}' https://example.com/api

# Follow redirects
curl -L https://example.com

# Show only response code
curl -o /dev/null -s -w "%{http_code}\n" https://example.com
```

### httpie (more readable)

```bash
http GET https://example.com
http POST https://example.com/api Content-Type:application/json x:=1
http --verify=no https://localhost:8443
```

### mitmproxy

```bash
# Interactive proxy
mitmproxy --listen-port 8080
# Configure browser/app to use 127.0.0.1:8080 as HTTP proxy
# View, modify, replay all requests

# Headless capture to file
mitmdump -w capture.dump

# Replay a saved request
mitmproxy -r capture.dump
```

### Charles / Burp Suite / Fiddler

GUI-driven HTTP proxies with TLS interception. Burp is the industry standard for security testing — its Repeater and Intruder modules are great for exploring API edge cases.

### Wireshark + TLS keylog

```bash
# In your app or browser:
SSLKEYLOGFILE=/tmp/keys.log curl https://example.com
# Then in Wireshark: Edit → Preferences → TLS → (Pre)-Master-Secret log filename → /tmp/keys.log
# Wireshark can now decrypt your TLS traffic.
```

### ngrep / tcpdump

```bash
# Show HTTP traffic to/from port 80
sudo ngrep -d any 'HTTP' port 80

# Capture to a file for later analysis
sudo tcpdump -i any -w capture.pcap port 443

# Read with Wireshark
wireshark capture.pcap
```

### Browser devtools

- Network tab: shows request/response, status, timing waterfall, headers, body.
- Console: shows JS errors including CORS rejections (the actual "blocked by CORS policy" message comes from the browser's console).
- `Cmd-Opt-J` / `Ctrl-Shift-J`: open console.
- Right-click a request → "Copy as cURL": reproduce request from the command line.
- "Preserve log" checkbox: keep entries across navigation.
- "Disable cache" checkbox: useful when debugging caching behaviour.

## Idioms

> Use the right status code, even if it stings.

If a user submitted bad input, return 4xx — even if it makes the dashboard look bad. Hiding errors with 200 doesn't fix them; it just masks them until production.

> Always set Content-Type.

Even `text/plain; charset=utf-8` is better than nothing. Content-sniffing browsers are a security hazard (`X-Content-Type-Options: nosniff` + correct Content-Type is the right answer).

> Vary on every header that affects the response.

If `Accept`, `Accept-Language`, `Accept-Encoding`, `Origin`, `Cookie`, or `Authorization` change the response body, list them in `Vary`. Otherwise CDNs will serve the wrong response.

> Fail closed at 4xx, fail open at 5xx.

Clients should not auto-retry 4xx. Clients MAY retry 5xx (with backoff, on idempotent methods). Make sure your status codes mean what they say.

> Include Retry-After when you mean 429 or 503.

Otherwise the client guesses, hammers you with retries, and a transient overload becomes a sustained outage.

> Idempotency keys for non-idempotent operations.

Stripe-style: client sends `Idempotency-Key: abc123` on POST. Server stores the result for that key and returns the same response on retries. Decouples retry safety from method semantics.

> Surface a request ID on every response.

```http
HTTP/1.1 500 Internal Server Error
X-Request-Id: req_01HMXJ3K8Z2VW9P0QF3T7N5R6Y
```
Users quote this in support tickets; you grep your logs.

> Don't return stack traces to clients.

Log them server-side; respond with a generic error and request ID.

> Prefer HTTPS-only redirects to upgrade-insecure-requests.

```http
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```

> Document your error format.

If your API returns JSON errors, define a schema (`{error, message, code, details}`) and stick to it.

> Test your 5xx paths.

Most teams test 200s extensively and 5xx paths almost never. Inject failures: kill a database, break a downstream, simulate timeouts. Your error responses are a feature; treat them like one.

## See Also

- tls
- dns
- troubleshooting/tls-errors
- troubleshooting/dns-errors
- troubleshooting/javascript-errors

## References

- RFC 9110 — HTTP Semantics
- RFC 9111 — HTTP Caching
- RFC 9112 — HTTP/1.1
- RFC 9113 — HTTP/2
- RFC 9114 — HTTP/3
- RFC 6265 — HTTP State Management Mechanism (Cookies)
- RFC 6265bis — Cookies (working draft)
- RFC 6455 — The WebSocket Protocol
- RFC 6750 — OAuth 2.0 Bearer Token Usage
- RFC 7235 — HTTP Authentication (obsoleted by 9110)
- RFC 7234 — HTTP Caching (obsoleted by 9111)
- RFC 7725 — 451 Unavailable For Legal Reasons
- RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport
- RFC 9001 — Using TLS to Secure QUIC
- RFC 9002 — QUIC Loss Detection and Congestion Control
- RFC 7540 — HTTP/2 (obsoleted by 9113)
- RFC 2324 — Hyper Text Coffee Pot Control Protocol (April 1, 1998)
- MDN Web Docs — HTTP/Status (https://developer.mozilla.org/docs/Web/HTTP/Status)
- MDN Web Docs — HTTP/Headers (https://developer.mozilla.org/docs/Web/HTTP/Headers)
- IANA HTTP Status Code Registry (https://www.iana.org/assignments/http-status-codes/)
- IANA HTTP/2 and HTTP/3 Error Code Registries
- PortSwigger Web Security Academy — HTTP request smuggling
- OWASP — HTTP Security Headers cheat sheet
