# Web Crawler Concurrent (Concurrency / Graph Traversal)

Crawl all same-domain pages concurrently with bounded parallelism, visiting each URL at most once.

## Problem

Given a `startUrl` and a function `getUrls(url)` that returns all URLs linked from a page,
crawl all pages under the same domain concurrently. Each URL must be visited at most once.
Limit concurrency to at most N simultaneous requests.

**Constraints:**

- URLs are strings starting with `"http://"`
- Same domain = same hostname (e.g., `"http://example.com/a"` and `"http://example.com/b"`)
- `1 <= N` (max concurrency) `<= 10`
- The web graph may contain cycles.

**Examples:**

```
startUrl = "http://example.com/"
getUrls("http://example.com/") => [
    "http://example.com/page1",
    "http://example.com/page2",
    "http://other.com/external",
]
getUrls("http://example.com/page1") => [
    "http://example.com/",
    "http://example.com/page3",
]
getUrls("http://example.com/page2") => ["http://example.com/page1"]
getUrls("http://example.com/page3") => ["http://example.com/page1"]

Result: ["http://example.com/", "http://example.com/page1",
         "http://example.com/page2", "http://example.com/page3"]
```

## Hints

- **Visited set:** Use a concurrent-safe set (sync.Map, Set with lock, HashSet with Mutex)
  to track which URLs have been enqueued. Mark a URL visited *before* spawning its task
  to prevent duplicates.
- **Semaphore pattern:** Use a buffered channel (Go), asyncio.Semaphore (Python),
  tokio::sync::Semaphore (Rust), or a promise pool (TypeScript) to cap concurrent requests.
- **Domain filtering:** Parse each discovered URL and compare hostnames. Discard cross-domain links.
- **Coordination:** Use WaitGroup (Go), asyncio.gather (Python), thread join handles (Rust),
  or Promise.all (TypeScript) to know when all work is done.

## Solution -- Go

```go
import (
	"net/url"
	"sort"
	"sync"
)

func crawl(startUrl string, getUrls func(string) []string, maxConcurrency int) []string {
	startParsed, _ := url.Parse(startUrl)
	domain := startParsed.Hostname()

	var visited sync.Map
	var mu sync.Mutex
	var result []string
	var wg sync.WaitGroup

	// Buffered channel as semaphore
	sem := make(chan struct{}, maxConcurrency)

	var visit func(u string)
	visit = func(u string) {
		defer wg.Done()

		// Acquire semaphore slot
		sem <- struct{}{}

		urls := getUrls(u)
		mu.Lock()
		result = append(result, u)
		mu.Unlock()

		// Release semaphore slot
		<-sem

		// Process discovered URLs
		for _, nextUrl := range urls {
			parsed, err := url.Parse(nextUrl)
			if err != nil {
				continue
			}
			if parsed.Hostname() != domain {
				continue
			}
			if _, loaded := visited.LoadOrStore(nextUrl, true); !loaded {
				wg.Add(1)
				go visit(nextUrl)
			}
		}
	}

	visited.Store(startUrl, true)
	wg.Add(1)
	go visit(startUrl)
	wg.Wait()

	sort.Strings(result)
	return result
}
```

## Solution -- Python

```python
import asyncio
from typing import List, Callable, Set
from urllib.parse import urlparse


async def crawl(
    start_url: str,
    get_urls: Callable[[str], List[str]],
    max_concurrency: int = 5,
) -> List[str]:
    """Crawl all same-domain URLs starting from start_url."""
    domain = urlparse(start_url).hostname
    visited: Set[str] = set()
    visited.add(start_url)
    result: List[str] = []
    sem = asyncio.Semaphore(max_concurrency)

    async def visit(url: str) -> None:
        async with sem:
            urls = get_urls(url)
            result.append(url)

        # Spawn tasks for new same-domain URLs
        tasks = []
        for next_url in urls:
            next_domain = urlparse(next_url).hostname
            if next_domain == domain and next_url not in visited:
                visited.add(next_url)
                tasks.append(asyncio.create_task(visit(next_url)))

        if tasks:
            await asyncio.gather(*tasks)

    await visit(start_url)
    return sorted(result)
```

## Solution -- Rust

```rust
use std::collections::{HashMap, HashSet};
use std::sync::{Arc, Mutex};
use std::thread;

struct Solution;

impl Solution {
    fn crawl(
        start_url: &str,
        get_urls: &(dyn Fn(&str) -> Vec<String> + Sync),
        max_concurrency: usize,
    ) -> Vec<String> {
        let domain = Self::get_domain(start_url);

        let visited = Arc::new(Mutex::new(HashSet::new()));
        let result = Arc::new(Mutex::new(Vec::new()));
        let sem = Arc::new(Semaphore::new(max_concurrency));

        visited.lock().unwrap().insert(start_url.to_string());

        let queue = Arc::new(Mutex::new(vec![start_url.to_string()]));
        let active = Arc::new(Mutex::new(0usize));

        loop {
            let url = {
                let mut q = queue.lock().unwrap();
                q.pop()
            };

            match url {
                Some(url) => {
                    *active.lock().unwrap() += 1;
                    sem.acquire();

                    let urls = get_urls(&url);
                    result.lock().unwrap().push(url);

                    sem.release();

                    for next_url in urls {
                        let next_domain = Self::get_domain(&next_url);
                        if next_domain == domain {
                            let mut vis = visited.lock().unwrap();
                            if !vis.contains(&next_url) {
                                vis.insert(next_url.clone());
                                queue.lock().unwrap().push(next_url);
                            }
                        }
                    }

                    *active.lock().unwrap() -= 1;
                }
                None => {
                    if *active.lock().unwrap() == 0 {
                        break;
                    }
                    thread::yield_now();
                }
            }
        }

        let mut res = result.lock().unwrap().clone();
        res.sort();
        res
    }

    fn get_domain(url: &str) -> String {
        let without_scheme = url
            .strip_prefix("http://")
            .or_else(|| url.strip_prefix("https://"))
            .unwrap_or(url);
        without_scheme
            .split('/')
            .next()
            .unwrap_or("")
            .to_string()
    }
}

struct Semaphore {
    count: Mutex<usize>,
    max: usize,
}

impl Semaphore {
    fn new(max: usize) -> Self {
        Semaphore {
            count: Mutex::new(0),
            max,
        }
    }

    fn acquire(&self) {
        loop {
            let mut count = self.count.lock().unwrap();
            if *count < self.max {
                *count += 1;
                return;
            }
            drop(count);
            thread::yield_now();
        }
    }

    fn release(&self) {
        let mut count = self.count.lock().unwrap();
        *count -= 1;
    }
}
```

## Solution -- TypeScript

```typescript
function getDomain(url: string): string {
    const withoutScheme = url.replace(/^https?:\/\//, "");
    return withoutScheme.split("/")[0];
}

class PromisePool {
    private active = 0;
    private queue: Array<() => void> = [];

    constructor(private maxConcurrency: number) {}

    async run<T>(fn: () => Promise<T>): Promise<T> {
        if (this.active >= this.maxConcurrency) {
            await new Promise<void>((resolve) => {
                this.queue.push(resolve);
            });
        }
        this.active++;
        try {
            return await fn();
        } finally {
            this.active--;
            if (this.queue.length > 0) {
                const next = this.queue.shift()!;
                next();
            }
        }
    }
}

async function crawl(
    startUrl: string,
    getUrls: (url: string) => string[],
    maxConcurrency: number
): Promise<string[]> {
    const domain = getDomain(startUrl);
    const visited = new Set<string>();
    const result: string[] = [];
    const pool = new PromisePool(maxConcurrency);

    visited.add(startUrl);

    async function visit(url: string): Promise<void> {
        const urls = await pool.run(async () => {
            result.push(url);
            return getUrls(url);
        });

        const tasks: Promise<void>[] = [];
        for (const nextUrl of urls) {
            if (getDomain(nextUrl) === domain && !visited.has(nextUrl)) {
                visited.add(nextUrl);
                tasks.push(visit(nextUrl));
            }
        }
        await Promise.all(tasks);
    }

    await visit(startUrl);
    return result.sort();
}
```

## Complexity

| Metric | Value |
|--------|-------|
| Time | O(V + E) where V = pages crawled, E = links followed |
| Space | O(V) for visited set and result storage |
| Concurrency | Bounded by N (semaphore permits) |

## Tips

- **Mark visited before spawning**, not after. If you check-then-spawn without atomically
  marking, two goroutines/tasks may both see a URL as unvisited and spawn duplicate work.
  Use `sync.Map.LoadOrStore` (Go), `set.add` before `create_task` (Python), or
  `visited.has` + `visited.add` in the same synchronous block (TypeScript).
- **Semaphore vs. worker pool:** A semaphore limits how many goroutines/tasks are *inside*
  the fetch section simultaneously. A worker pool limits the total number of goroutines/tasks
  alive. Both work; the semaphore pattern is simpler when tasks are short-lived.
- **Cycle handling is automatic** if you use a visited set. BFS and DFS both naturally
  handle cycles -- the key invariant is "never enqueue a URL twice."
- **Domain extraction:** Parse the URL properly rather than using string heuristics.
  `url.Parse` (Go), `urlparse` (Python), or `new URL()` (TypeScript) handle edge cases
  like ports, userinfo, and encoded characters.
- **Error handling in production:** Real crawlers must handle DNS failures, timeouts,
  HTTP errors, redirects, and robots.txt. The semaphore pattern extends naturally to
  include retry logic and backoff.

## See Also

- bounded-blocking-queue
- graph-traversal
- concurrency
- breadth-first-search

## References

- [LeetCode 1242 -- Web Crawler Multithreaded](https://leetcode.com/problems/web-crawler-multithreaded/)
- [Go sync.Map Documentation](https://pkg.go.dev/sync#Map)
- [Python asyncio.Semaphore](https://docs.python.org/3/library/asyncio-sync.html#asyncio.Semaphore)
