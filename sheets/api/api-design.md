# API Design (RESTful Principles, Versioning, and Best Practices)

A comprehensive reference for designing, versioning, and operating production HTTP APIs — covering resource design, pagination, rate limiting, and lifecycle management.

## RESTful Design Principles

### Resource Naming

```
Good:
  GET    /users                    List users
  POST   /users                    Create user
  GET    /users/123                Get user 123
  PUT    /users/123                Replace user 123
  PATCH  /users/123                Partial update user 123
  DELETE /users/123                Delete user 123

  GET    /users/123/orders         List orders for user 123
  POST   /users/123/orders         Create order for user 123
  GET    /users/123/orders/456     Get order 456 of user 123

Bad:
  GET    /getUser?id=123           Verb in URL (use HTTP method)
  POST   /createUser               Verb in URL
  GET    /user/123                 Singular (use plural)
  GET    /Users/123                Capital letters
  GET    /users/123/get-orders     Verb in nested resource

Rules:
  - Use plural nouns: /users not /user
  - Use lowercase with hyphens: /order-items not /orderItems
  - Max nesting depth: 2 levels (/users/123/orders)
  - No trailing slashes: /users not /users/
  - No file extensions: /users not /users.json
```

### HTTP Methods

| Method | Idempotent | Safe | Request Body | Response Body |
|---|---|---|---|---|
| GET | Yes | Yes | No | Yes |
| POST | No | No | Yes | Yes |
| PUT | Yes | No | Yes | Yes (or 204) |
| PATCH | No* | No | Yes | Yes (or 204) |
| DELETE | Yes | No | Rarely | Optional |
| HEAD | Yes | Yes | No | No |
| OPTIONS | Yes | Yes | No | Yes (CORS) |

*PATCH can be made idempotent with JSON Merge Patch but is not inherently so.

### HTTP Status Codes

```
2xx Success:
  200 OK              Successful GET, PUT, PATCH, DELETE
  201 Created          Successful POST (include Location header)
  202 Accepted         Async operation accepted (not yet completed)
  204 No Content       Successful DELETE, PUT with no response body

3xx Redirection:
  301 Moved Permanently  Resource URL changed permanently
  304 Not Modified       Conditional GET (ETag/If-None-Match)

4xx Client Error:
  400 Bad Request      Invalid request body or parameters
  401 Unauthorized     Missing or invalid authentication
  403 Forbidden        Authenticated but not authorized
  404 Not Found        Resource does not exist
  405 Method Not Allowed  HTTP method not supported for this resource
  409 Conflict         Resource conflict (e.g., duplicate, version mismatch)
  410 Gone             Resource permanently deleted (more specific than 404)
  415 Unsupported Media Type  Wrong Content-Type
  422 Unprocessable Entity    Validation error (syntactically correct, semantically wrong)
  429 Too Many Requests       Rate limit exceeded

5xx Server Error:
  500 Internal Server Error   Unexpected server failure
  502 Bad Gateway             Upstream service error
  503 Service Unavailable     Temporary overload or maintenance
  504 Gateway Timeout         Upstream service timeout
```

## Versioning Strategies

### URL Path Versioning (Recommended for Public APIs)

```
GET /api/v1/users/123
GET /api/v2/users/123

Pros: Explicit, cacheable, easy to route
Cons: Changes URL, harder to deprecate gradually
```

### Header Versioning

```bash
# Custom header
curl -H "API-Version: 2" https://api.example.com/users/123

# Accept header (content negotiation)
curl -H "Accept: application/vnd.example.v2+json" \
     https://api.example.com/users/123
```

### Query Parameter Versioning

```
GET /users/123?version=2

Pros: Easy to add
Cons: Pollutes query string, caching issues
```

### Versioning Decision Matrix

| Factor | URL Path | Header | Query Param |
|---|---|---|---|
| Visibility | High | Low | Medium |
| Cacheability | Excellent | Requires Vary | Moderate |
| API gateway routing | Easy | Moderate | Easy |
| Client simplicity | Simple | Requires config | Simple |
| Best for | Public APIs | Internal APIs | Transitional |

## Pagination

### Cursor-Based Pagination (Recommended)

```go
type PageResponse struct {
    Data       []Item `json:"data"`
    NextCursor string `json:"next_cursor,omitempty"`
    HasMore    bool   `json:"has_more"`
}

// Request
// GET /users?limit=20&cursor=eyJpZCI6MTIzfQ

// Response
// {
//   "data": [...],
//   "next_cursor": "eyJpZCI6MTQzfQ",
//   "has_more": true
// }

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    limit := parseLimit(r, 20, 100) // default 20, max 100
    cursor := r.URL.Query().Get("cursor")

    var afterID int
    if cursor != "" {
        afterID = decodeCursor(cursor) // base64-decoded opaque cursor
    }

    users, err := h.repo.ListAfter(r.Context(), afterID, limit+1)
    if err != nil {
        writeError(w, 500, err)
        return
    }

    hasMore := len(users) > limit
    if hasMore {
        users = users[:limit]
    }

    var nextCursor string
    if hasMore {
        nextCursor = encodeCursor(users[len(users)-1].ID)
    }

    writeJSON(w, 200, PageResponse{
        Data:       users,
        NextCursor: nextCursor,
        HasMore:    hasMore,
    })
}
```

### Offset-Based Pagination

```
GET /users?offset=40&limit=20

Response:
{
  "data": [...],
  "total": 1234,
  "offset": 40,
  "limit": 20
}

Pros: Simple, supports "jump to page N"
Cons: Inconsistent with concurrent writes (phantom reads),
      slow for large offsets (DB must scan + skip)
```

### Keyset Pagination

```sql
-- More efficient than OFFSET for large datasets
-- First page
SELECT * FROM users ORDER BY created_at DESC, id DESC LIMIT 20;

-- Next page (after last item: created_at='2026-04-01', id=123)
SELECT * FROM users
WHERE (created_at, id) < ('2026-04-01', 123)
ORDER BY created_at DESC, id DESC
LIMIT 20;
```

## Idempotency

### Idempotency Keys

```go
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
    idempotencyKey := r.Header.Get("Idempotency-Key")
    if idempotencyKey == "" {
        writeError(w, 400, "Idempotency-Key header required")
        return
    }

    // Check for existing result
    existing, err := h.idempotencyStore.Get(r.Context(), idempotencyKey)
    if err == nil && existing != nil {
        // Return cached response
        w.WriteHeader(existing.StatusCode)
        w.Write(existing.Body)
        return
    }

    // Process the request
    result, statusCode, err := h.processPayment(r.Context(), r.Body)
    if err != nil {
        writeError(w, 500, err)
        return
    }

    // Cache the result (TTL: 24 hours)
    body, _ := json.Marshal(result)
    h.idempotencyStore.Set(r.Context(), idempotencyKey, &CachedResponse{
        StatusCode: statusCode,
        Body:       body,
    }, 24*time.Hour)

    w.WriteHeader(statusCode)
    w.Write(body)
}
```

### Natural Idempotency

```
Naturally idempotent (same result when repeated):
  PUT /users/123 {"name": "Alice"}     Always sets name to Alice
  DELETE /users/123                     Delete is idempotent (404 on repeat is OK)

NOT naturally idempotent (side effects accumulate):
  POST /payments {"amount": 100}        Creates new payment each time
  PATCH /users/123 {"balance": "+100"}  Increments each time

  Fix: Use idempotency keys for non-idempotent operations
```

## Rate Limiting

### Rate Limit Headers

```
Response headers:
  X-RateLimit-Limit: 1000        Max requests per window
  X-RateLimit-Remaining: 742     Requests remaining
  X-RateLimit-Reset: 1712234400  Unix timestamp when window resets

  Retry-After: 30                Seconds to wait (on 429 response)
```

### Implementation

```go
type RateLimiter struct {
    store RedisClient
}

func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, RateLimitInfo, error) {
    now := time.Now()
    windowKey := fmt.Sprintf("rl:%s:%d", key, now.Unix()/int64(window.Seconds()))

    count, err := rl.store.Incr(ctx, windowKey)
    if err != nil {
        return false, RateLimitInfo{}, err
    }

    if count == 1 {
        rl.store.Expire(ctx, windowKey, window)
    }

    info := RateLimitInfo{
        Limit:     limit,
        Remaining: max(0, limit-int(count)),
        Reset:     now.Truncate(window).Add(window),
    }

    return int(count) <= limit, info, nil
}

func rateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := extractAPIKey(r) // or IP address
            allowed, info, err := limiter.Allow(r.Context(), key, 1000, time.Hour)
            if err != nil {
                next.ServeHTTP(w, r) // fail open
                return
            }

            w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
            w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
            w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset.Unix(), 10))

            if !allowed {
                w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(info.Reset).Seconds())))
                writeError(w, 429, "rate limit exceeded")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Error Responses (RFC 7807)

### Problem Details Format

```json
{
    "type": "https://api.example.com/errors/insufficient-funds",
    "title": "Insufficient Funds",
    "status": 422,
    "detail": "Account abc-123 has balance $10.00, but $25.00 is required.",
    "instance": "/payments/pay-789",
    "balance": 10.00,
    "required": 25.00
}
```

```go
type ProblemDetail struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail,omitempty"`
    Instance string `json:"instance,omitempty"`
}

func writeError(w http.ResponseWriter, status int, detail string) {
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ProblemDetail{
        Type:   fmt.Sprintf("https://api.example.com/errors/%d", status),
        Title:  http.StatusText(status),
        Status: status,
        Detail: detail,
    })
}

// Validation errors with field-level detail
type ValidationProblem struct {
    ProblemDetail
    Errors []FieldError `json:"errors"`
}

type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}
```

## Filtering, Sorting, and Searching

```
# Filtering
GET /users?status=active&role=admin&created_after=2026-01-01

# Sorting (prefix - for descending)
GET /users?sort=created_at        Ascending
GET /users?sort=-created_at       Descending
GET /users?sort=-created_at,name  Multiple fields

# Searching
GET /users?q=alice                Full-text search
GET /users?name[contains]=ali     Field-specific operators

# Partial responses (field selection)
GET /users/123?fields=id,name,email
GET /users?fields=id,name&sort=-created_at&limit=10

# Combined
GET /users?status=active&sort=-created_at&fields=id,name&limit=20&cursor=abc
```

```go
type ListParams struct {
    Filters map[string]string
    Sort    []SortField
    Fields  []string
    Limit   int
    Cursor  string
}

func parseListParams(r *http.Request) ListParams {
    q := r.URL.Query()
    params := ListParams{
        Filters: make(map[string]string),
        Limit:   min(parseInt(q.Get("limit"), 20), 100),
        Cursor:  q.Get("cursor"),
    }

    // Parse filters
    for _, key := range []string{"status", "role", "type"} {
        if v := q.Get(key); v != "" {
            params.Filters[key] = v
        }
    }

    // Parse sort fields
    if sortStr := q.Get("sort"); sortStr != "" {
        for _, field := range strings.Split(sortStr, ",") {
            if strings.HasPrefix(field, "-") {
                params.Sort = append(params.Sort, SortField{Field: field[1:], Desc: true})
            } else {
                params.Sort = append(params.Sort, SortField{Field: field, Desc: false})
            }
        }
    }

    // Parse field selection
    if fields := q.Get("fields"); fields != "" {
        params.Fields = strings.Split(fields, ",")
    }

    return params
}
```

## Bulk Operations

```go
// Batch create
// POST /users/batch
// Request: {"items": [{...}, {...}, {...}]}
// Response: {"results": [{"status": 201, ...}, {"status": 400, ...}, ...]}

type BatchRequest struct {
    Items []json.RawMessage `json:"items"`
}

type BatchResult struct {
    Index  int             `json:"index"`
    Status int             `json:"status"`
    Data   json.RawMessage `json:"data,omitempty"`
    Error  *ProblemDetail  `json:"error,omitempty"`
}

func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
    var req BatchRequest
    json.NewDecoder(r.Body).Decode(&req)

    if len(req.Items) > 100 {
        writeError(w, 400, "batch size exceeds maximum of 100")
        return
    }

    results := make([]BatchResult, len(req.Items))
    for i, item := range req.Items {
        result, status, err := h.createOne(r.Context(), item)
        if err != nil {
            results[i] = BatchResult{Index: i, Status: status, Error: toProblem(err)}
        } else {
            data, _ := json.Marshal(result)
            results[i] = BatchResult{Index: i, Status: status, Data: data}
        }
    }

    w.WriteHeader(207) // Multi-Status
    json.NewEncoder(w).Encode(map[string]any{"results": results})
}
```

## API Lifecycle

```
Alpha → Beta → GA → Deprecated → Sunset

Alpha:
  - Breaking changes expected
  - No stability guarantee
  - Header: X-API-Stability: alpha

Beta:
  - Mostly stable, breaking changes with notice
  - 30-day deprecation notice for changes
  - Header: X-API-Stability: beta

GA (General Availability):
  - Stable, breaking changes require new version
  - 6-12 month deprecation period

Deprecated:
  - Still functional but scheduled for removal
  - Header: Deprecation: true
  - Header: Sunset: Sat, 01 Nov 2026 00:00:00 GMT
  - Response header: Link: </api/v2/users>; rel="successor-version"

Sunset:
  - API version removed
  - Returns 410 Gone
```

## Backward Compatibility Rules

```
Safe changes (non-breaking):
  + Adding new endpoints
  + Adding optional request fields
  + Adding response fields
  + Adding new enum values (if client handles unknown)
  + Adding new HTTP methods to existing resources
  + Relaxing validation (accepting wider input range)

Breaking changes (require new version):
  - Removing or renaming endpoints
  - Removing or renaming fields
  - Changing field types
  - Adding required request fields
  - Tightening validation
  - Changing error codes/formats
  - Changing authentication scheme
  - Removing enum values
```

## Tips

- Use nouns for resources, HTTP methods for actions — the URL is the "what", the method is the "how"
- Always paginate list endpoints — unbounded responses are a reliability risk
- Cursor-based pagination is more robust than offset-based for data that changes
- Idempotency keys should be client-generated UUIDs, not server-generated
- Rate limiting should fail open (allow request) if the rate limiter itself is down
- Use RFC 7807 Problem Details for all error responses — clients can parse consistently
- Version the API from day one, even if you only have v1
- Return 404 for resources that do not exist, 403 for resources the user cannot access (prevents enumeration)

## See Also

- `detail/api/api-design.md` — token bucket math, pagination consistency
- `sheets/performance/caching-patterns.md` — HTTP caching headers
- `sheets/quality/sre-fundamentals.md` — rate limiting and error budgets

## References

- RFC 7807 Problem Details: https://tools.ietf.org/html/rfc7807
- REST API Design Rulebook (O'Reilly, Mark Masse, 2011)
- Google API Design Guide: https://cloud.google.com/apis/design
- Microsoft REST API Guidelines: https://github.com/microsoft/api-guidelines
- Stripe API Design: https://stripe.com/docs/api
