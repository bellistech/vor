# Caching Patterns (Strategies, Invalidation, and Multi-Tier)

A comprehensive reference for caching patterns in application, distributed, and CDN layers — covering read/write strategies, invalidation, stampede prevention, and HTTP caching.

## Cache-Aside (Lazy Loading)

```go
// Application manages cache explicitly
// Read: check cache → miss → read DB → populate cache
// Write: update DB → invalidate cache

func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    // 1. Check cache
    cached, err := s.cache.Get(ctx, "user:"+id)
    if err == nil && cached != nil {
        var user User
        json.Unmarshal(cached, &user)
        return &user, nil
    }

    // 2. Cache miss — read from database
    user, err := s.db.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }

    // 3. Populate cache
    data, _ := json.Marshal(user)
    s.cache.Set(ctx, "user:"+id, data, 5*time.Minute)

    return user, nil
}

func (s *Service) UpdateUser(ctx context.Context, id string, updates UserUpdate) error {
    // 1. Update database
    if err := s.db.UpdateUser(ctx, id, updates); err != nil {
        return err
    }

    // 2. Invalidate cache (don't update — avoids race conditions)
    s.cache.Delete(ctx, "user:"+id)

    return nil
}
```

**Pros**: Only caches data that is actually accessed. Simple to implement.
**Cons**: Cache miss penalty on first access. Potential for stale data if cache is not invalidated.

## Read-Through

```go
// Cache sits between app and DB — cache handles misses
type ReadThroughCache struct {
    cache  Cache
    loader func(ctx context.Context, key string) ([]byte, error)
    ttl    time.Duration
}

func (c *ReadThroughCache) Get(ctx context.Context, key string) ([]byte, error) {
    data, err := c.cache.Get(ctx, key)
    if err == nil && data != nil {
        return data, nil
    }

    // Cache handles the miss transparently
    data, err = c.loader(ctx, key)
    if err != nil {
        return nil, err
    }

    c.cache.Set(ctx, key, data, c.ttl)
    return data, nil
}
```

**Pros**: Application code is simpler — just calls cache. Logic centralized.
**Cons**: Cache library must understand data source.

## Write-Through

```go
// Writes go through cache to DB synchronously
func (c *WriteThroughCache) Set(ctx context.Context, key string, value []byte) error {
    // 1. Write to database first
    if err := c.db.Write(ctx, key, value); err != nil {
        return err
    }

    // 2. Write to cache
    return c.cache.Set(ctx, key, value, c.ttl)
}
```

**Pros**: Cache and DB always consistent. No stale reads.
**Cons**: Write latency increases (both DB + cache on write path).

## Write-Behind (Write-Back)

```go
// Writes go to cache immediately, async flush to DB
type WriteBehindCache struct {
    cache    Cache
    buffer   chan WriteOp
    db       Database
    interval time.Duration
}

func (c *WriteBehindCache) Set(ctx context.Context, key string, value []byte) error {
    // Write to cache immediately
    if err := c.cache.Set(ctx, key, value, c.ttl); err != nil {
        return err
    }

    // Queue async write to DB
    c.buffer <- WriteOp{Key: key, Value: value}
    return nil
}

func (c *WriteBehindCache) FlushLoop(ctx context.Context) {
    ticker := time.NewTicker(c.interval)
    var batch []WriteOp

    for {
        select {
        case op := <-c.buffer:
            batch = append(batch, op)
            if len(batch) >= 100 {
                c.flushBatch(ctx, batch)
                batch = batch[:0]
            }
        case <-ticker.C:
            if len(batch) > 0 {
                c.flushBatch(ctx, batch)
                batch = batch[:0]
            }
        case <-ctx.Done():
            if len(batch) > 0 {
                c.flushBatch(context.Background(), batch) // drain on shutdown
            }
            return
        }
    }
}
```

**Pros**: Lowest write latency. Batching reduces DB load.
**Cons**: Risk of data loss if cache fails before flush. Complex consistency model.

## Refresh-Ahead

```go
// Proactively refresh cache before TTL expires
type RefreshAheadCache struct {
    cache       Cache
    loader      func(ctx context.Context, key string) ([]byte, error)
    ttl         time.Duration
    refreshAt   float64 // fraction of TTL (e.g., 0.8 = refresh at 80% TTL)
}

func (c *RefreshAheadCache) Get(ctx context.Context, key string) ([]byte, error) {
    data, meta, err := c.cache.GetWithMeta(ctx, key)
    if err != nil || data == nil {
        // Cache miss — synchronous load
        return c.loadAndCache(ctx, key)
    }

    // Check if nearing expiry
    elapsed := time.Since(meta.CreatedAt)
    if elapsed > time.Duration(float64(c.ttl)*c.refreshAt) {
        // Async refresh — return current value, update in background
        go func() {
            newData, err := c.loader(context.Background(), key)
            if err == nil {
                c.cache.Set(context.Background(), key, newData, c.ttl)
            }
        }()
    }

    return data, nil
}
```

**Pros**: Eliminates cache miss latency for popular keys. Users always get cached response.
**Cons**: Background refresh adds load. Less useful for infrequently accessed keys.

## Cache Invalidation Strategies

### TTL-Based

```go
// Simple: set expiry on cache entries
cache.Set(ctx, "user:123", data, 5*time.Minute)

// Staggered TTL to prevent simultaneous expiry
baseTTL := 5 * time.Minute
jitter := time.Duration(rand.Intn(60)) * time.Second
cache.Set(ctx, "user:123", data, baseTTL+jitter)
```

### Event-Based

```go
// Invalidate on data change events
func (c *CacheInvalidator) HandleEvent(event Event) {
    switch event.Type {
    case "UserUpdated":
        c.cache.Delete(context.Background(), "user:"+event.AggregateID)
        c.cache.Delete(context.Background(), "user-list:page:*") // pattern delete
    case "OrderCreated":
        c.cache.Delete(context.Background(), "user-orders:"+event.Data.CustomerID)
    }
}
```

### Versioned Keys

```go
// Version in key — bump version to invalidate all entries
func cacheKey(entity string, id string, version int) string {
    return fmt.Sprintf("%s:v%d:%s", entity, version, id)
}

// Config stores current version
var userCacheVersion = 1

// To invalidate all user cache entries:
userCacheVersion++ // all old keys become orphaned (eventually evicted)
```

## Cache Stampede Prevention

### singleflight (Go)

```go
import "golang.org/x/sync/singleflight"

var group singleflight.Group

func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
    // Check cache first
    if cached := s.cache.Get(ctx, "user:"+id); cached != nil {
        return cached.(*User), nil
    }

    // Deduplicate concurrent requests for the same key
    result, err, shared := group.Do("user:"+id, func() (any, error) {
        user, err := s.db.GetUser(ctx, id)
        if err != nil {
            return nil, err
        }
        data, _ := json.Marshal(user)
        s.cache.Set(ctx, "user:"+id, data, 5*time.Minute)
        return user, nil
    })

    if shared {
        log.Printf("singleflight: shared result for user:%s", id)
    }

    if err != nil {
        return nil, err
    }
    return result.(*User), nil
}
```

### Mutex/Lock

```go
// Distributed lock for cache rebuild
func (s *Service) GetWithLock(ctx context.Context, key string) ([]byte, error) {
    data, err := s.cache.Get(ctx, key)
    if err == nil && data != nil {
        return data, nil
    }

    // Acquire distributed lock
    lock, err := s.locker.Acquire(ctx, "lock:"+key, 10*time.Second)
    if err != nil {
        // Another process is rebuilding — wait and retry cache
        time.Sleep(100 * time.Millisecond)
        return s.cache.Get(ctx, key)
    }
    defer lock.Release(ctx)

    // Double-check cache (another goroutine may have filled it)
    data, err = s.cache.Get(ctx, key)
    if err == nil && data != nil {
        return data, nil
    }

    // Load from database
    data, err = s.loadFromDB(ctx, key)
    if err != nil {
        return nil, err
    }

    s.cache.Set(ctx, key, data, 5*time.Minute)
    return data, nil
}
```

### Probabilistic Early Expiration (XFetch)

```go
// Probabilistically refresh before TTL to prevent stampede
func (s *Service) GetWithXFetch(ctx context.Context, key string) ([]byte, error) {
    data, meta, err := s.cache.GetWithMeta(ctx, key)
    if err != nil || data == nil {
        return s.loadAndCache(ctx, key)
    }

    // XFetch: probabilistically refresh early
    ttl := meta.TTL
    age := time.Since(meta.CreatedAt)
    delta := meta.ComputeTime // time it took to generate the value

    // Probability of refresh increases as TTL approaches
    // P(refresh) = delta * beta * ln(rand()) + age > ttl
    beta := 1.0 // tuning parameter
    threshold := float64(age) - float64(delta)*beta*math.Log(rand.Float64())

    if threshold > float64(ttl) {
        // Refresh in background
        go s.loadAndCache(context.Background(), key)
    }

    return data, nil
}
```

## Multi-Tier Caching

```
Request → L1 (Process-local) → L2 (Distributed) → L3 (CDN) → Origin

L1: In-process map or LRU cache
    Latency: ~1μs
    Size: MB range (bounded by process memory)
    Shared: No (per-instance)

L2: Redis, Memcached
    Latency: ~1ms (network hop)
    Size: GB-TB range
    Shared: Yes (all instances)

L3: CDN (CloudFront, Fastly)
    Latency: ~10-50ms (edge pop)
    Size: Practically unlimited
    Shared: Yes (global)
```

```go
type MultiTierCache struct {
    l1     *lru.Cache       // process-local
    l2     *redis.Client    // distributed
    l1TTL  time.Duration
    l2TTL  time.Duration
}

func (c *MultiTierCache) Get(ctx context.Context, key string) ([]byte, error) {
    // L1: process-local
    if val, ok := c.l1.Get(key); ok {
        return val.([]byte), nil
    }

    // L2: distributed
    data, err := c.l2.Get(ctx, key).Bytes()
    if err == nil {
        c.l1.Add(key, data) // promote to L1
        return data, nil
    }

    return nil, ErrCacheMiss
}

func (c *MultiTierCache) Set(ctx context.Context, key string, value []byte) {
    c.l1.Add(key, value)
    c.l2.Set(ctx, key, value, c.l2TTL)
}

func (c *MultiTierCache) Invalidate(ctx context.Context, key string) {
    c.l1.Remove(key)
    c.l2.Del(ctx, key)
    // L3 (CDN): use cache tags or purge API
}
```

## HTTP Cache Headers

### Cache-Control

```
# Response headers
Cache-Control: public, max-age=3600          # CDN + browser cache for 1h
Cache-Control: private, max-age=600          # Browser only, 10 min
Cache-Control: no-cache                       # Must revalidate every time
Cache-Control: no-store                       # Never cache (sensitive data)
Cache-Control: public, max-age=31536000, immutable  # Static assets (1 year)
Cache-Control: s-maxage=3600, max-age=60     # CDN 1h, browser 1min
Cache-Control: stale-while-revalidate=60     # Serve stale for 60s while refreshing
Cache-Control: stale-if-error=300            # Serve stale for 5min on origin error
```

### ETag and Conditional Requests

```go
func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
    resource := h.loadResource(r.Context(), r.URL.Path)
    etag := fmt.Sprintf(`"%x"`, sha256.Sum256(resource.Data))

    // Check If-None-Match
    if match := r.Header.Get("If-None-Match"); match == etag {
        w.WriteHeader(http.StatusNotModified) // 304
        return
    }

    w.Header().Set("ETag", etag)
    w.Header().Set("Cache-Control", "public, max-age=60")
    w.Write(resource.Data)
}

// Client request flow:
// 1. GET /api/config → 200 OK, ETag: "abc123"
// 2. GET /api/config, If-None-Match: "abc123" → 304 Not Modified (no body)
// 3. GET /api/config, If-None-Match: "abc123" → 200 OK (if changed)
```

### Last-Modified

```go
func (h *Handler) ServeFile(w http.ResponseWriter, r *http.Request) {
    info, _ := os.Stat(filePath)
    modTime := info.ModTime()

    // Check If-Modified-Since
    if ims := r.Header.Get("If-Modified-Since"); ims != "" {
        if t, err := http.ParseTime(ims); err == nil && !modTime.After(t) {
            w.WriteHeader(http.StatusNotModified) // 304
            return
        }
    }

    w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))
    http.ServeFile(w, r, filePath)
}
```

### Vary Header

```go
// Tell caches which request headers affect the response
w.Header().Set("Vary", "Accept-Encoding, Accept-Language")

// Without Vary: CDN caches gzip response, serves to client that wants brotli
// With Vary: CDN caches separate entries per Accept-Encoding value
```

## Cache Warming

```go
// Preload cache on startup or deployment
func (s *Service) WarmCache(ctx context.Context) error {
    // Load most popular items
    popular, err := s.db.GetMostAccessed(ctx, 1000)
    if err != nil {
        return err
    }

    sem := make(chan struct{}, 20) // limit concurrency
    var wg sync.WaitGroup

    for _, item := range popular {
        wg.Add(1)
        sem <- struct{}{}
        go func(id string) {
            defer wg.Done()
            defer func() { <-sem }()

            data, err := s.db.Get(ctx, id)
            if err != nil {
                return
            }
            s.cache.Set(ctx, "item:"+id, data, 10*time.Minute)
        }(item.ID)
    }

    wg.Wait()
    log.Printf("cache warmed with %d items", len(popular))
    return nil
}
```

## Cache Sizing

```
Rule of thumb:
  Working set = data accessed within one TTL period
  Cache size >= working set for high hit rate

Estimate working set:
  Unique keys accessed in 5 min: 50,000
  Average value size: 2 KB
  Working set: 50,000 * 2 KB = 100 MB

  With overhead (metadata, fragmentation): 150-200 MB

Redis memory per key (approximate):
  Key overhead: ~70 bytes
  String value: value_size + ~40 bytes
  Total per entry: ~110 bytes + value_size

  50,000 entries * (110 + 2048) bytes = ~103 MB
```

## Redis vs Memcached Selection

| Feature | Redis | Memcached |
|---|---|---|
| Data structures | Strings, hashes, lists, sets, sorted sets | Strings only |
| Persistence | RDB + AOF | None |
| Replication | Built-in (master-replica) | None |
| Clustering | Redis Cluster (automatic sharding) | Client-side sharding |
| Memory efficiency | Higher overhead per key | More memory-efficient |
| Pub/sub | Built-in | None |
| Lua scripting | Yes | None |
| Max value size | 512 MB | 1 MB (default) |
| Multi-threaded | Single-threaded (io-threads in 6.0+) | Multi-threaded |

**Use Redis when**: You need data structures, persistence, pub/sub, or Lua scripting.
**Use Memcached when**: You need simple key-value caching with maximum memory efficiency and multi-threaded performance.

## CDN Caching

```
CDN cache strategy:
  Static assets (JS, CSS, images):
    Cache-Control: public, max-age=31536000, immutable
    Fingerprinted filenames: app.a1b2c3d4.js

  API responses:
    Cache-Control: public, s-maxage=60, stale-while-revalidate=30
    Vary: Authorization, Accept

  HTML pages:
    Cache-Control: public, max-age=0, must-revalidate
    ETag for conditional requests

  Purge strategies:
    - URL purge: invalidate specific URL
    - Tag purge: invalidate all URLs with a cache tag
    - Surrogate keys: group related resources for bulk purge
```

## Tips

- Cache-aside is the safest default — explicit control over what is cached and when
- Never update cache on write — delete and let the next read repopulate (avoids race conditions)
- Use singleflight in Go to prevent cache stampedes — it is simple and effective
- Stagger TTLs with jitter to prevent thundering herd on mass expiry
- Set max-age shorter than s-maxage: browser cache expires before CDN, triggering 304 validation
- Monitor cache hit ratio — below 80% means your cache is too small or TTL too short
- Use the `immutable` directive for fingerprinted static assets — prevents unnecessary revalidation
- Redis persistence (RDB/AOF) is for crash recovery, not a replacement for your database

## See Also

- `detail/performance/caching-patterns.md` — hit ratio math, LRU analysis, Bloom filters
- `sheets/api/api-design.md` — API rate limiting and pagination
- `sheets/quality/sre-fundamentals.md` — caching for reliability

## References

- "Designing Data-Intensive Applications" by Martin Kleppmann (Chapter 5)
- Redis Documentation: https://redis.io/docs/
- MDN HTTP Caching: https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching
- "XFetch: A Probabilistic Early Expiration Algorithm" (Vattani, 2015)
