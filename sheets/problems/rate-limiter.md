# Rate Limiter (System Design / Concurrency)

Implement a sliding window rate limiter that allows at most N requests in any window of W seconds, with thread-safe concurrent access.

## Problem

Design a rate limiter with the following API:

- **Allow(timestamp)** -- Returns `true` if the request at the given timestamp is within
  the rate limit, `false` otherwise.
- At most `maxRequests` in any `windowSeconds` window.
- Thread-safe for concurrent access.

**Constraints:**

- Timestamps are in seconds (integers or floats).
- Timestamps are non-decreasing (calls come in order).
- Multiple threads/goroutines may call Allow concurrently.

**Examples:**

```
limiter = RateLimiter(maxRequests=3, windowSeconds=10)

limiter.allow(1)   => true   (1 request in window)
limiter.allow(2)   => true   (2 requests in window)
limiter.allow(3)   => true   (3 requests in window)
limiter.allow(4)   => false  (4th request, over limit)
limiter.allow(11)  => true   (timestamp 1 expired, back to 3 in window)
```

## Walkthrough

Consider `RateLimiter(maxRequests=3, windowSeconds=10)`:

```
t=1:  window=(−9, 1].  Timestamps: [].       Count=0 < 3 => ALLOW. Store [1].
t=2:  window=(−8, 2].  Timestamps: [1].      Count=1 < 3 => ALLOW. Store [1, 2].
t=3:  window=(−7, 3].  Timestamps: [1, 2].   Count=2 < 3 => ALLOW. Store [1, 2, 3].
t=4:  window=(−6, 4].  Timestamps: [1, 2, 3]. Count=3 >= 3 => REJECT.
t=11: window=(1, 11].  Evict t=1 (1 <= 1).   Timestamps: [2, 3]. Count=2 < 3 => ALLOW.
                                               Store [2, 3, 11].
t=12: window=(2, 12].  Evict t=2 (2 <= 2).   Timestamps: [3, 11]. Count=2 < 3 => ALLOW.
                                               Store [3, 11, 12].
```

Key insight: the cutoff is `timestamp - windowSeconds`, and we evict timestamps `<= cutoff`
(not `< cutoff`). This makes the window half-open: `(cutoff, timestamp]`.

## Hints

- **Sliding window log:** Store each request timestamp in a deque/slice. On each call,
  remove timestamps older than `timestamp - windowSeconds`. If the remaining count is
  below the limit, allow and append.
- **Cutoff is exclusive:** Remove timestamps `<= cutoff`, not `< cutoff`. A request at
  time `t` expires at exactly `t + windowSeconds`.
- **Thread safety:** Wrap the entire check-and-append in a mutex. The operation is not
  safe to split across lock/unlock boundaries. A check-then-act race allows two threads
  to both see count < limit and both append, exceeding the limit.
- **Alternative -- sliding window counter:** Uses fixed time buckets with interpolation for
  approximate counting with O(1) memory. Formula:
  `estimated = prevCount * (1 - elapsed/window) + currCount`.
- **Alternative -- token bucket:** Tokens refill at a steady rate. Each request consumes
  one token. If the bucket is empty, reject. Allows controlled bursts up to bucket capacity.

## Solution -- Go

```go
import (
	"sync"
)

// RateLimiter implements a sliding window rate limiter.
// Thread-safe via sync.Mutex.
type RateLimiter struct {
	maxRequests   int
	windowSeconds float64
	timestamps    []float64
	mu            sync.Mutex
}

// NewRateLimiter creates a rate limiter allowing maxRequests
// in any rolling window of windowSeconds.
func NewRateLimiter(maxRequests int, windowSeconds float64) *RateLimiter {
	return &RateLimiter{
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
	}
}

// Allow returns true if the request is within the rate limit.
// The entire check-evict-append sequence is atomic under the mutex.
func (r *RateLimiter) Allow(timestamp float64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove expired timestamps: evict all t where t <= cutoff
	cutoff := timestamp - r.windowSeconds
	i := 0
	for i < len(r.timestamps) && r.timestamps[i] <= cutoff {
		i++
	}
	r.timestamps = r.timestamps[i:]

	// Check if under limit, then record
	if len(r.timestamps) < r.maxRequests {
		r.timestamps = append(r.timestamps, timestamp)
		return true
	}
	return false
}
```

## Solution -- Python

```python
import threading
from collections import deque


class RateLimiter:
    """Sliding window rate limiter with thread safety.

    Uses a deque of timestamps for O(1) eviction from the front.
    The entire allow() operation is protected by a threading.Lock
    to prevent check-then-act races between concurrent threads.
    """

    def __init__(self, max_requests: int, window_seconds: float):
        self.max_requests = max_requests
        self.window_seconds = window_seconds
        self.timestamps: deque = deque()
        self.lock = threading.Lock()

    def allow(self, timestamp: float) -> bool:
        with self.lock:
            # Remove expired timestamps from the front.
            # Cutoff uses <=, making the window half-open: (cutoff, timestamp].
            cutoff = timestamp - self.window_seconds
            while self.timestamps and self.timestamps[0] <= cutoff:
                self.timestamps.popleft()

            # If under the limit, record this request and allow it.
            if len(self.timestamps) < self.max_requests:
                self.timestamps.append(timestamp)
                return True
            return False

    def count(self) -> int:
        """Return current number of requests in the window (for testing)."""
        with self.lock:
            return len(self.timestamps)
```

## Solution -- Rust

```rust
use std::collections::VecDeque;
use std::sync::Mutex;

struct RateLimiter {
    max_requests: usize,
    window_seconds: f64,
    inner: Mutex<VecDeque<f64>>,
}

impl RateLimiter {
    fn new(max_requests: usize, window_seconds: f64) -> Self {
        RateLimiter {
            max_requests,
            window_seconds,
            inner: Mutex::new(VecDeque::new()),
        }
    }

    fn allow(&self, timestamp: f64) -> bool {
        let mut timestamps = self.inner.lock().unwrap();

        // Remove expired timestamps
        let cutoff = timestamp - self.window_seconds;
        while let Some(&front) = timestamps.front() {
            if front <= cutoff {
                timestamps.pop_front();
            } else {
                break;
            }
        }

        if timestamps.len() < self.max_requests {
            timestamps.push_back(timestamp);
            true
        } else {
            false
        }
    }
}
```

## Solution -- TypeScript

```typescript
class RateLimiter {
    private maxRequests: number;
    private windowSeconds: number;
    private timestamps: number[] = [];

    constructor(maxRequests: number, windowSeconds: number) {
        this.maxRequests = maxRequests;
        this.windowSeconds = windowSeconds;
    }

    allow(timestamp: number): boolean {
        // Remove expired timestamps
        const cutoff = timestamp - this.windowSeconds;
        while (this.timestamps.length > 0 && this.timestamps[0] <= cutoff) {
            this.timestamps.shift();
        }

        if (this.timestamps.length < this.maxRequests) {
            this.timestamps.push(timestamp);
            return true;
        }
        return false;
    }
}
```

## Algorithm Comparison

```
Algorithm             Memory    Precision   Burst control   Use case
-------------------   --------  ----------  -------------   --------
Sliding window log    O(N)      Exact       Perfect         Low-volume APIs
Sliding window ctr    O(1)      Approx      Good            High-volume APIs
Fixed window          O(1)      Exact       Poor (2x burst) Simple counters
Token bucket          O(1)      Exact       Configurable    Bursty traffic
Leaky bucket          O(1)      Exact       None (smooth)   Steady output
```

Where N = maxRequests (the number of timestamps stored in the window).

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(k) per call where k = expired entries removed (amortized O(1)) |
| Space | O(maxRequests) -- at most maxRequests timestamps stored |

## Tips

- **Sliding window log vs. fixed window:** Fixed windows have boundary bursts -- a client
  can send `maxRequests` at the end of one window and `maxRequests` at the start of the next,
  doubling the rate momentarily. The sliding window log avoids this entirely because the
  window moves with each request.
- **Sliding window counter (approximate):** Interpolate between current and previous fixed
  windows for O(1) memory. Formula: `estimated = prevCount * (1 - elapsed/window) + currCount`.
  Less precise but uses constant memory regardless of request volume. The worst-case error
  is bounded: at most `maxRequests * (1 - elapsed/window)` extra requests in the transition
  zone.
- **Deque vs. array:** Python's `deque.popleft()` is O(1). Go slices and JS arrays use
  O(k) shifting, but k is bounded by the eviction rate. For high-throughput systems,
  use a circular buffer or ring buffer for guaranteed O(1) operations.
- **TypeScript has no threading:** The single-threaded event loop means no mutex is needed.
  However, if using Worker threads, you need `SharedArrayBuffer` + `Atomics` or a separate
  limiter per worker. The async nature of JS means concurrent requests still serialize
  through the event loop.
- **Production rate limiters** often use Redis with Lua scripts for distributed enforcement.
  The sliding window log maps naturally to a sorted set: `ZADD key timestamp timestamp`,
  `ZREMRANGEBYSCORE key 0 cutoff`, `ZCARD key`. Lua scripting ensures atomicity across
  the multi-command sequence.
- **Token bucket is the alternative:** Replenishes tokens at a fixed rate. Allows bursts up
  to bucket capacity. Simpler for APIs but less precise for strict window guarantees.
  The leaky bucket variant enforces a strict output rate with no bursts at all.
- **Memory considerations:** The sliding window log stores one timestamp per allowed request
  within the window. For a limiter allowing 1000 requests per second, that is 1000 float64
  values (8 KB). For 1M requests/sec, consider the counter approach instead.
- **Monotonic timestamps are critical.** If timestamps can go backward (e.g., clock skew
  in distributed systems), the eviction logic may not work correctly. In production, use
  monotonic clocks (e.g., `time.Monotonic` in Go, `time.monotonic()` in Python) rather
  than wall-clock time.
- **Per-key rate limiting:** In real APIs, you typically rate-limit per user/IP/API key.
  Use a map from key to RateLimiter instance. To bound memory, evict inactive limiters
  after an idle period using an LRU cache.
- **HTTP status codes:** When a request is rate-limited, return HTTP 429 (Too Many Requests)
  with a `Retry-After` header indicating how many seconds the client should wait before
  retrying. This is defined in RFC 6585.

## See Also

- sliding-window
- concurrency
- system-design
- token-bucket
- leaky-bucket
- distributed-systems

## References

- [Rate Limiting Algorithms (Cloudflare Blog)](https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)
- [Redis Rate Limiting](https://redis.io/commands/incr#pattern-rate-limiter)
- [Token Bucket (Wikipedia)](https://en.wikipedia.org/wiki/Token_bucket)
- [RFC 6585 -- Additional HTTP Status Codes (429 Too Many Requests)](https://www.rfc-editor.org/rfc/rfc6585)
- [Stripe Rate Limiting Design](https://stripe.com/blog/rate-limiters)
