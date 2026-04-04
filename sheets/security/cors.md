# CORS (Cross-Origin Resource Sharing)

Browser security mechanism that controls which web origins can access resources on a different origin, using HTTP headers to relax the same-origin policy for legitimate cross-domain API requests.

## Same-Origin Policy

### Origin Definition

```
# An origin = scheme + host + port
https://example.com:443    # Origin A
https://example.com:8080   # Different origin (port)
http://example.com:443     # Different origin (scheme)
https://api.example.com    # Different origin (host)

# Same-origin requests: no CORS needed
# Cross-origin requests: CORS headers required
```

### Request Classification

```
Simple Requests (no preflight):
  - Methods: GET, HEAD, POST
  - Headers: Accept, Accept-Language, Content-Language,
             Content-Type (only text/plain, multipart/form-data,
                           application/x-www-form-urlencoded)
  - No ReadableStream body
  - No event listeners on XMLHttpRequest.upload

Preflighted Requests (OPTIONS first):
  - Methods: PUT, DELETE, PATCH, etc.
  - Custom headers: Authorization, X-Custom-Header, etc.
  - Content-Type: application/json
  - Any request that doesn't qualify as "simple"
```

## Preflight Request

### OPTIONS Exchange

```
# Browser automatically sends preflight before actual request

# 1. Preflight request (browser sends automatically)
OPTIONS /api/data HTTP/1.1
Host: api.example.com
Origin: https://app.example.com
Access-Control-Request-Method: PUT
Access-Control-Request-Headers: Content-Type, Authorization

# 2. Preflight response (server must respond)
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://app.example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 86400

# 3. Actual request (browser sends if preflight succeeds)
PUT /api/data HTTP/1.1
Host: api.example.com
Origin: https://app.example.com
Content-Type: application/json
Authorization: Bearer eyJhbGci...

# 4. Actual response (server includes CORS headers)
HTTP/1.1 200 OK
Access-Control-Allow-Origin: https://app.example.com
```

## Response Headers

### Access-Control-Allow-Origin

```
# Single specific origin (most secure)
Access-Control-Allow-Origin: https://app.example.com

# Wildcard (no credentials allowed with wildcard)
Access-Control-Allow-Origin: *

# Dynamic — reflect the request Origin after validation
# Server checks Origin against allowlist, then echoes it back
# MUST include Vary: Origin when using dynamic origins
```

### All CORS Response Headers

```
# Required
Access-Control-Allow-Origin: https://app.example.com

# Preflight only
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID
Access-Control-Max-Age: 86400          # Cache preflight for 24h (seconds)

# Actual response
Access-Control-Expose-Headers: X-RateLimit-Remaining, X-Request-ID
Access-Control-Allow-Credentials: true  # Allow cookies/auth headers

# MUST accompany dynamic origins
Vary: Origin
```

## Credentials

### Credentialed Requests

```
# JavaScript must opt-in:
fetch('https://api.example.com/data', {
  credentials: 'include'      // Send cookies cross-origin
});

// XMLHttpRequest:
xhr.withCredentials = true;

# Server MUST respond with ALL of:
Access-Control-Allow-Origin: https://app.example.com    # NOT wildcard *
Access-Control-Allow-Credentials: true

# Wildcard restrictions with credentials:
# - Allow-Origin CANNOT be *
# - Allow-Headers CANNOT be *
# - Allow-Methods CANNOT be *
# - Expose-Headers CANNOT be *
# Each must list specific values
```

## Server Configuration

### Nginx

```nginx
server {
    location /api/ {
        # Handle preflight
        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' '$http_origin' always;
            add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'Content-Type, Authorization';
            add_header 'Access-Control-Allow-Credentials' 'true';
            add_header 'Access-Control-Max-Age' 86400;
            add_header 'Vary' 'Origin';
            return 204;
        }

        # Actual requests
        add_header 'Access-Control-Allow-Origin' '$http_origin' always;
        add_header 'Access-Control-Allow-Credentials' 'true';
        add_header 'Access-Control-Expose-Headers' 'X-Request-ID';
        add_header 'Vary' 'Origin';

        proxy_pass http://backend;
    }
}
```

### Go (net/http)

```go
func corsMiddleware(next http.Handler) http.Handler {
    allowed := map[string]bool{
        "https://app.example.com":  true,
        "https://admin.example.com": true,
    }
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if allowed[origin] {
            w.Header().Set("Access-Control-Allow-Origin", origin)
            w.Header().Set("Access-Control-Allow-Credentials", "true")
            w.Header().Set("Vary", "Origin")
        }
        if r.Method == http.MethodOptions {
            w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
            w.Header().Set("Access-Control-Max-Age", "86400")
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Express.js

```javascript
const cors = require('cors');
app.use(cors({
  origin: ['https://app.example.com', 'https://admin.example.com'],
  methods: ['GET', 'POST', 'PUT', 'DELETE'],
  allowedHeaders: ['Content-Type', 'Authorization'],
  exposedHeaders: ['X-Request-ID'],
  credentials: true,
  maxAge: 86400
}));
```

## Debugging

### Browser DevTools

```bash
# Check console for CORS errors:
# "Access to fetch at 'https://api...' from origin 'https://app...'
#  has been blocked by CORS policy"

# Network tab: look for OPTIONS request before actual request
# If OPTIONS returns non-2xx: server CORS config is wrong
# If OPTIONS missing required headers: specific header missing

# Common errors:
# - "No 'Access-Control-Allow-Origin' header"
# - "Wildcard '*' cannot be used with credentials"
# - "Method PUT is not allowed"
# - "Header 'Authorization' is not allowed"
```

### curl Testing

```bash
# Simulate preflight
curl -X OPTIONS https://api.example.com/data \
  -H "Origin: https://app.example.com" \
  -H "Access-Control-Request-Method: PUT" \
  -H "Access-Control-Request-Headers: Content-Type, Authorization" \
  -v 2>&1 | grep -i "access-control"

# Simulate simple request
curl https://api.example.com/data \
  -H "Origin: https://app.example.com" \
  -v 2>&1 | grep -i "access-control"
```

## Tips

- CORS is enforced by browsers only; curl and server-to-server calls ignore it entirely
- Always include `Vary: Origin` when dynamically reflecting the Origin header to prevent cache poisoning
- Set `Access-Control-Max-Age` to cache preflight results (max 7200s in Chrome, 86400s in Firefox)
- Wildcard `*` for Allow-Origin is incompatible with `credentials: include` on the client
- Preflight requests do not include cookies or Authorization headers; only the actual request does
- Use an allowlist of origins on the server rather than reflecting any Origin header blindly
- The `Access-Control-Expose-Headers` header is needed for JavaScript to read custom response headers
- CORS errors in the browser console do not show the server response body for security reasons
- Proxy the API through your own origin (same-origin) to avoid CORS entirely in development
- Private network access (localhost from public origins) requires additional CORS preflight headers
- Do not rely on CORS as a security boundary; it prevents browser-based abuse but not direct API access
- Test CORS with curl using `-H "Origin: ..."` to verify server headers without browser involvement

## See Also

oauth, jwt, tls, nginx, web-servers

## References

- [MDN — Cross-Origin Resource Sharing](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [Fetch Standard — CORS Protocol](https://fetch.spec.whatwg.org/#http-cors-protocol)
- [W3C CORS Specification](https://www.w3.org/TR/cors/)
- [OWASP CORS Misconfiguration](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/11-Client-side_Testing/07-Testing_Cross_Origin_Resource_Sharing)
