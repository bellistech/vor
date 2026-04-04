# REST API (Representational State Transfer)

Design and consume HTTP APIs following REST architectural constraints including statelessness, uniform interface, layered system, and resource-oriented URLs with standard HTTP methods and status codes.

## HTTP Methods and Semantics

### Method Properties

```
Method    Safe    Idempotent    Request Body    Typical Use
------    ----    ----------    ------------    -----------
GET       yes     yes           no              Retrieve resource
HEAD      yes     yes           no              Headers only (check existence)
POST      no      no            yes             Create resource / trigger action
PUT       no      yes           yes             Replace entire resource
PATCH     no      no*           yes             Partial update
DELETE    no      yes           no              Remove resource
OPTIONS   yes     yes           no              CORS preflight / discover methods
```

### CRUD Mapping

```bash
# Create
curl -X POST https://api.example.com/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com"}'

# Read (single)
curl https://api.example.com/v1/users/42

# Read (collection)
curl "https://api.example.com/v1/users?role=admin&limit=20"

# Update (full replace)
curl -X PUT https://api.example.com/v1/users/42 \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Smith", "email": "alice@example.com", "role": "admin"}'

# Update (partial -- JSON Merge Patch)
curl -X PATCH https://api.example.com/v1/users/42 \
  -H "Content-Type: application/merge-patch+json" \
  -d '{"name": "Alice Smith"}'

# Update (partial -- JSON Patch)
curl -X PATCH https://api.example.com/v1/users/42 \
  -H "Content-Type: application/json-patch+json" \
  -d '[{"op": "replace", "path": "/name", "value": "Alice Smith"}]'

# Delete
curl -X DELETE https://api.example.com/v1/users/42
```

## Status Codes

### Success (2xx)

```
200 OK                    Standard success for GET, PUT, PATCH
201 Created               Resource created (POST), include Location header
202 Accepted              Async processing started, not complete yet
204 No Content            Success with no body (DELETE, PUT with no return)
```

### Redirection (3xx)

```
301 Moved Permanently     Resource URL changed forever
304 Not Modified          Conditional GET, client cache is fresh
307 Temporary Redirect    Retry same method at new URL
308 Permanent Redirect    Like 301, but preserves HTTP method
```

### Client Error (4xx)

```
400 Bad Request           Malformed request / validation failure
401 Unauthorized          Missing or invalid authentication
403 Forbidden             Authenticated but not authorized
404 Not Found             Resource does not exist
405 Method Not Allowed    HTTP method not supported on this resource
409 Conflict              State conflict (e.g., duplicate, version mismatch)
412 Precondition Failed   Conditional update failed (ETag mismatch)
415 Unsupported Media     Wrong Content-Type
422 Unprocessable Entity  Valid JSON but semantic errors
429 Too Many Requests     Rate limit exceeded, check Retry-After header
```

### Server Error (5xx)

```
500 Internal Server Error Unhandled server error
502 Bad Gateway           Upstream service returned invalid response
503 Service Unavailable   Server overloaded or in maintenance
504 Gateway Timeout       Upstream service timed out
```

## Pagination

### Cursor-Based Pagination

```bash
# First page
curl "https://api.example.com/v1/posts?limit=20"
# Response includes: { "data": [...], "next_cursor": "eyJpZCI6MTAwfQ==" }

# Next page
curl "https://api.example.com/v1/posts?limit=20&cursor=eyJpZCI6MTAwfQ=="

# Response headers approach
# Link: <https://api.example.com/v1/posts?cursor=abc123>; rel="next"
```

### Offset-Based Pagination

```bash
# Page 1
curl "https://api.example.com/v1/posts?offset=0&limit=20"

# Page 3
curl "https://api.example.com/v1/posts?offset=40&limit=20"

# With total count in response
# { "data": [...], "total": 342, "offset": 40, "limit": 20 }
```

## Content Negotiation

### Request and Response Formats

```bash
# Request JSON, get JSON (default)
curl https://api.example.com/v1/users/42 \
  -H "Accept: application/json"

# Request XML
curl https://api.example.com/v1/users/42 \
  -H "Accept: application/xml"

# Content negotiation with quality values
curl https://api.example.com/v1/users/42 \
  -H "Accept: application/json;q=1.0, application/xml;q=0.5"

# Specify request body format
curl -X POST https://api.example.com/v1/users \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"name": "Alice"}'
```

## Versioning Strategies

### URL Path Versioning

```bash
# Most common, explicit, easy to route
curl https://api.example.com/v1/users
curl https://api.example.com/v2/users
```

### Header Versioning

```bash
# Custom header
curl https://api.example.com/users \
  -H "API-Version: 2"

# Accept header with vendor media type
curl https://api.example.com/users \
  -H "Accept: application/vnd.example.v2+json"
```

### Query Parameter Versioning

```bash
curl "https://api.example.com/users?version=2"
```

## Idempotency and Safety

### Idempotency Keys

```bash
# POST with idempotency key for safe retries
curl -X POST https://api.example.com/v1/payments \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1000, "currency": "USD"}'

# Retry same request safely (server returns cached response)
curl -X POST https://api.example.com/v1/payments \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1000, "currency": "USD"}'
```

### Conditional Requests (ETags)

```bash
# Get resource with ETag
curl -i https://api.example.com/v1/users/42
# ETag: "a1b2c3d4"

# Conditional GET (304 if unchanged)
curl https://api.example.com/v1/users/42 \
  -H "If-None-Match: \"a1b2c3d4\""

# Conditional PUT (409 if changed since)
curl -X PUT https://api.example.com/v1/users/42 \
  -H "If-Match: \"a1b2c3d4\"" \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name"}'
```

## Rate Limiting

### Standard Headers

```bash
# Response headers for rate limit info
# X-RateLimit-Limit: 1000
# X-RateLimit-Remaining: 997
# X-RateLimit-Reset: 1625097600
# Retry-After: 30

# IETF draft standard headers
# RateLimit-Limit: 1000
# RateLimit-Remaining: 997
# RateLimit-Reset: 30
```

### Handling Rate Limits

```bash
# Exponential backoff on 429
for i in 1 2 4 8 16; do
  response=$(curl -s -o /dev/null -w "%{http_code}" \
    https://api.example.com/v1/users)
  if [ "$response" = "429" ]; then
    echo "Rate limited, waiting ${i}s..."
    sleep $i
  else
    break
  fi
done
```

## HATEOAS (Hypermedia)

### Link Relations

```json
{
  "id": 42,
  "name": "Alice",
  "email": "alice@example.com",
  "_links": {
    "self": { "href": "/v1/users/42" },
    "posts": { "href": "/v1/users/42/posts" },
    "avatar": { "href": "/v1/users/42/avatar" },
    "deactivate": { "href": "/v1/users/42", "method": "DELETE" }
  }
}
```

### Collection with Links

```json
{
  "data": [
    { "id": 1, "title": "Post 1" },
    { "id": 2, "title": "Post 2" }
  ],
  "_links": {
    "self": { "href": "/v1/posts?page=2" },
    "next": { "href": "/v1/posts?page=3" },
    "prev": { "href": "/v1/posts?page=1" },
    "first": { "href": "/v1/posts?page=1" },
    "last": { "href": "/v1/posts?page=17" }
  },
  "_meta": {
    "total": 342,
    "page": 2,
    "per_page": 20
  }
}
```

## Error Response Format

### RFC 7807 Problem Details

```json
{
  "type": "https://api.example.com/errors/validation",
  "title": "Validation Error",
  "status": 422,
  "detail": "The 'email' field is not a valid email address.",
  "instance": "/v1/users",
  "errors": [
    {
      "field": "email",
      "code": "invalid_format",
      "message": "Must be a valid email address"
    }
  ]
}
```

## Tips

- Use nouns for resource URLs (`/users/42`) and avoid verbs (`/getUser`) -- the HTTP method is the verb
- Always return `Location` header with `201 Created` pointing to the new resource URL
- Use `429 Too Many Requests` with `Retry-After` header instead of silently dropping requests
- Make PUT truly idempotent by replacing the entire resource -- use PATCH for partial updates
- Implement idempotency keys on POST endpoints that create resources or trigger side effects (payments, emails)
- Return `204 No Content` for successful DELETE rather than `200` with an empty body
- Prefer cursor-based pagination for large or frequently-changing datasets -- offset pagination skips or duplicates rows on concurrent writes
- Use ETags and conditional requests (`If-Match`, `If-None-Match`) to prevent lost updates and reduce bandwidth
- Version your API in the URL path (`/v1/`) for simplicity -- header versioning adds complexity with little benefit
- Include a `request_id` in every response for debugging and support correlation
- Design for eventual consistency -- return `202 Accepted` for long-running operations with a status polling endpoint
- Use ISO 8601 for all timestamps and include timezone information

## See Also

- openapi, graphql, http, curl, json, api-gateway, nginx

## References

- [Fielding's REST Dissertation (Chapter 5)](https://www.ics.uci.edu/~fielding/pubs/dissertation/rest_arch_style.htm)
- [RFC 9110 - HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110)
- [RFC 7807 - Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc7807)
- [RFC 6585 - Additional HTTP Status Codes (429)](https://www.rfc-editor.org/rfc/rfc6585)
- [JSON Merge Patch - RFC 7396](https://www.rfc-editor.org/rfc/rfc7396)
- [JSON Patch - RFC 6902](https://www.rfc-editor.org/rfc/rfc6902)
