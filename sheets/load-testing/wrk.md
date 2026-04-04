# wrk (HTTP Benchmarking)

High-performance HTTP benchmarking tool using epoll/kqueue for efficient connection handling, with Lua scripting support for custom request generation, response processing, and latency distribution reporting.

## Installation

```bash
# Build from source (Linux)
git clone https://github.com/wg/wrk.git
cd wrk && make && cp wrk /usr/local/bin/

# macOS
brew install wrk

# wrk2 (constant throughput variant)
git clone https://github.com/giltene/wrk2.git
cd wrk2 && make && cp wrk /usr/local/bin/wrk2

# Verify
wrk --version
```

## Basic Usage

```bash
# Simple benchmark (default: 2 threads, 10 connections, 10s)
wrk http://localhost:8080/

# Specify threads, connections, duration
wrk -t4 -c100 -d30s http://localhost:8080/api/health

# With timeout
wrk -t4 -c200 -d60s --timeout 5s http://localhost:8080/

# Print latency distribution
wrk -t4 -c100 -d30s --latency http://localhost:8080/

# Custom headers
wrk -t2 -c50 -d10s \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  http://localhost:8080/api/users
```

## Understanding Output

```
Running 30s test @ http://localhost:8080/
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     5.23ms    2.11ms  45.67ms   78.32%
    Req/Sec    4.78k   312.45     5.89k    72.50%
  Latency Distribution
     50%    4.89ms
     75%    6.12ms
     90%    7.83ms
     99%   12.45ms
  572,134 requests in 30.01s, 89.45MB read
Requests/sec:  19,064.23
Transfer/sec:      2.98MB
```

```
# Key metrics:
# Avg Latency    — mean response time across all requests
# Stdev          — standard deviation (lower = more consistent)
# Max            — worst-case response time
# +/- Stdev      — percentage within one standard deviation
# Req/Sec        — per-thread requests per second
# Latency Dist.  — percentile breakdown (p50, p75, p90, p99)
```

## Lua Scripting

### Script Phases

```lua
-- wrk Lua API has three phases:
-- 1. setup(thread)  — called once per thread during init
-- 2. init(args)     — called once per thread after setup
-- 3. request()      — called for every request
-- 4. response(status, headers, body) — called for every response
-- 5. done(summary, latency, requests) — called once at end
```

### Custom Request (POST with JSON)

```lua
-- post.lua
wrk.method = "POST"
wrk.body   = '{"username": "test", "password": "secret"}'
wrk.headers["Content-Type"] = "application/json"
```

```bash
wrk -t4 -c50 -d30s -s post.lua http://localhost:8080/api/login
```

### Dynamic Request Generation

```lua
-- dynamic.lua
local counter = 0

request = function()
  counter = counter + 1
  local path = "/api/users/" .. (counter % 1000)
  return wrk.format("GET", path)
end
```

### Request Pipeline

```lua
-- pipeline.lua — send multiple requests per connection
init = function(args)
  local r = {}
  r[1] = wrk.format("GET", "/api/users")
  r[2] = wrk.format("GET", "/api/posts")
  r[3] = wrk.format("GET", "/api/comments")
  req = table.concat(r)
end

request = function()
  return req
end
```

### Custom Reporting

```lua
-- report.lua
done = function(summary, latency, requests)
  io.write("------------------------------\n")
  io.write(string.format("Total Requests: %d\n", summary.requests))
  io.write(string.format("Total Errors:   %d\n", summary.errors.status))
  io.write(string.format("Avg Latency:    %.2f ms\n", latency.mean / 1000))
  io.write(string.format("Max Latency:    %.2f ms\n", latency.max / 1000))
  io.write(string.format("P50 Latency:    %.2f ms\n", latency:percentile(50) / 1000))
  io.write(string.format("P99 Latency:    %.2f ms\n", latency:percentile(99) / 1000))
  io.write(string.format("Req/Sec:        %.2f\n", summary.requests / (summary.duration / 1e6)))
  io.write(string.format("Transfer:       %.2f MB\n", summary.bytes / 1e6))
  io.write("------------------------------\n")
end
```

### Authentication Flow

```lua
-- auth.lua — login first, then use token
local token = nil

setup = function(thread)
  thread:set("id", 1)
end

init = function(args)
  -- Login request
  local body = '{"username":"test","password":"secret"}'
  local headers = {}
  headers["Content-Type"] = "application/json"

  -- Store for later use
  wrk.headers["Content-Type"] = "application/json"
end

request = function()
  if token then
    wrk.headers["Authorization"] = "Bearer " .. token
    return wrk.format("GET", "/api/protected")
  else
    wrk.method = "POST"
    wrk.body = '{"username":"test","password":"secret"}'
    return wrk.format("POST", "/api/login")
  end
end

response = function(status, headers, body)
  if not token and status == 200 then
    -- Extract token from JSON response
    token = body:match('"token":"([^"]+)"')
  end
end
```

### Random Data from File

```lua
-- random_paths.lua
local paths = {}

init = function(args)
  local f = io.open(args[1] or "paths.txt", "r")
  if f then
    for line in f:lines() do
      paths[#paths + 1] = line
    end
    f:close()
  end
end

request = function()
  local path = paths[math.random(#paths)]
  return wrk.format("GET", path)
end
```

```bash
wrk -t4 -c100 -d30s -s random_paths.lua -- paths.txt http://localhost:8080
```

## wrk2 (Constant Rate)

```bash
# wrk2 adds -R flag for constant request rate
wrk2 -t4 -c100 -d60s -R 5000 http://localhost:8080/
#                      ^^^^^ 5000 requests/sec target

# With latency correction
wrk2 -t4 -c100 -d60s -R 10000 --latency http://localhost:8080/

# Rate sweep (shell loop)
for rate in 1000 2000 5000 10000 20000; do
  echo "=== Rate: $rate ==="
  wrk2 -t4 -c100 -d30s -R $rate --latency http://localhost:8080/
done
```

## Connection Tuning

```bash
# Increase file descriptor limit (before testing)
ulimit -n 65535

# System tuning (Linux)
sysctl -w net.ipv4.ip_local_port_range="1024 65535"
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.core.somaxconn=65535

# High connection count test
wrk -t8 -c1000 -d60s http://localhost:8080/

# Keep-alive test (default, wrk uses persistent connections)
wrk -t4 -c100 -d30s http://localhost:8080/
```

## Common Patterns

```bash
# Quick smoke test
wrk -t1 -c1 -d5s http://localhost:8080/health

# Find max throughput (increase connections until plateaus)
for c in 10 50 100 200 500 1000; do
  echo "=== Connections: $c ==="
  wrk -t4 -c$c -d15s --latency http://localhost:8080/
done

# Compare endpoints
for ep in /api/users /api/posts /api/comments; do
  echo "=== Endpoint: $ep ==="
  wrk -t4 -c100 -d15s --latency http://localhost:8080$ep
done

# POST benchmark
wrk -t4 -c100 -d30s -s post.lua http://localhost:8080/api/data

# Output to file for analysis
wrk -t4 -c100 -d30s --latency http://localhost:8080/ 2>&1 | tee results.txt
```

## Tips

- Always use `--latency` to see p50/p75/p90/p99 percentiles. Average latency alone hides tail latency problems.
- Use wrk2 instead of wrk when you need constant-rate load testing. Standard wrk uses a closed model that reduces throughput as latency increases.
- Set threads (`-t`) equal to the number of CPU cores on the load generator machine for optimal performance.
- Connections (`-c`) must be greater than or equal to threads. Each thread gets `c/t` connections.
- Increase `ulimit -n` on the load generator before tests with high connection counts to avoid file descriptor exhaustion.
- Run wrk from a separate machine than the server under test to avoid competing for CPU and network resources.
- Use Lua scripts for anything beyond simple GET requests. The `request()` function is called per-request for dynamic payloads.
- Watch for socket errors and timeouts in the output. High error counts mean the test results are measuring failure, not performance.
- wrk keeps connections alive by default (HTTP/1.1 keep-alive). This is realistic for most API testing.
- Run multiple test durations (15s, 60s, 300s) to distinguish warmup effects from steady-state performance.
- Pipe output through `tee` to capture results while still seeing them in real time.
- Use the `done()` Lua callback to generate CSV or JSON reports for automated regression tracking.

## See Also

k6, vegeta, ab, curl, hey

## References

- [wrk GitHub Repository](https://github.com/wg/wrk)
- [wrk2 GitHub Repository (Constant Throughput)](https://github.com/giltene/wrk2)
- [wrk Lua Scripting](https://github.com/wg/wrk/blob/master/SCRIPTING)
- [How NOT to Measure Latency (Gil Tene)](https://www.youtube.com/watch?v=lJ8ydIuPFeU)
- [Coordinated Omission Problem](https://groups.google.com/g/mechanical-sympathy/c/icNZJejUHfE)
