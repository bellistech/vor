# Vegeta (HTTP Load Testing)

Constant-rate HTTP load testing tool and library built in Go with support for targets files, structured output (JSON/CSV/histogram/plot), and composable Unix pipeline design for automated performance testing.

## Installation

```bash
# Go install
go install github.com/tsenart/vegeta@latest

# macOS
brew install vegeta

# Linux (download binary)
curl -fsSL https://github.com/tsenart/vegeta/releases/download/v12.12.0/vegeta_12.12.0_linux_amd64.tar.gz | \
  tar xz && mv vegeta /usr/local/bin/

# Verify
vegeta --version
```

## Basic Attack

```bash
# Simple GET attack at 50 req/sec for 10 seconds
echo "GET http://localhost:8080/" | vegeta attack -rate=50 -duration=10s | vegeta report

# With explicit target
echo "GET http://localhost:8080/api/health" | \
  vegeta attack -rate=100 -duration=30s | \
  vegeta report
```

## Targets File

```bash
# targets.txt — one request per block, blank line between
# Simple GET requests
GET http://localhost:8080/api/users
GET http://localhost:8080/api/posts
GET http://localhost:8080/api/comments
```

```bash
# targets-advanced.txt — with headers and body
GET http://localhost:8080/api/users
Authorization: Bearer eyJhbGc...
X-Request-ID: test-001

POST http://localhost:8080/api/users
Content-Type: application/json
Authorization: Bearer eyJhbGc...
@user.json

PUT http://localhost:8080/api/users/1
Content-Type: application/json
@update.json

DELETE http://localhost:8080/api/users/999
Authorization: Bearer eyJhbGc...
```

```bash
# Run with targets file
vegeta attack -targets=targets.txt -rate=100 -duration=30s | vegeta report
```

## Attack Options

```bash
# Constant rate
vegeta attack -rate=500 -duration=60s

# Rate with time unit
vegeta attack -rate=100/1s                # 100 per second (default)
vegeta attack -rate=6000/1m               # 6000 per minute = 100/s
vegeta attack -rate=1/100ms               # 10 per second

# Infinite rate (max throughput, closed model)
vegeta attack -rate=0 -max-workers=100 -duration=30s

# Max connections
vegeta attack -rate=500 -connections=100 -duration=30s

# Workers (concurrent goroutines)
vegeta attack -rate=1000 -workers=50 -max-workers=200 -duration=60s

# Timeout
vegeta attack -rate=100 -timeout=5s -duration=30s

# HTTP/2
vegeta attack -rate=100 -http2 -duration=30s

# Keep-alive disabled
vegeta attack -rate=100 -keepalive=false -duration=30s

# Custom headers for all requests
vegeta attack -rate=100 -duration=30s \
  -header="Authorization: Bearer token123" \
  -header="Accept: application/json"

# Request body from file
echo "POST http://localhost:8080/api/data" | \
  vegeta attack -rate=50 -body=payload.json -duration=30s | \
  vegeta report

# TLS (skip verification)
vegeta attack -rate=100 -insecure -duration=30s
```

## Report Types

```bash
# Text report (default)
vegeta attack -rate=100 -duration=10s < targets.txt | vegeta report

# JSON report
vegeta attack -rate=100 -duration=10s < targets.txt | vegeta report -type=json

# Histogram (bucketed latency distribution)
vegeta attack -rate=100 -duration=10s < targets.txt | \
  vegeta report -type=hist[0,5ms,10ms,25ms,50ms,100ms,500ms,1s]

# Output:
# Bucket         #     %       Histogram
# [0,     5ms]   450   45.00%  #################
# [5ms,   10ms]  320   32.00%  ############
# [10ms,  25ms]  150   15.00%  ######
# [25ms,  50ms]   60    6.00%  ##
# [50ms,  100ms]  15    1.50%  #
# [100ms, 500ms]   4    0.40%
# [500ms, 1s]      1    0.10%
```

## Encoding & Plotting

```bash
# Save raw results to file (binary encoding)
vegeta attack -rate=100 -duration=60s < targets.txt > results.bin

# Decode to JSON (one result per line)
vegeta encode --to json < results.bin > results.json

# Decode to CSV
vegeta encode --to csv < results.bin > results.csv

# Generate HTML latency plot
vegeta plot < results.bin > plot.html

# Combine multiple result files
cat results1.bin results2.bin | vegeta report
cat results1.bin results2.bin | vegeta plot > combined.html

# Report from saved results
vegeta report < results.bin
vegeta report -type=json < results.bin
```

## Pipeline Patterns

```bash
# Rate sweep — find saturation point
for rate in 100 200 500 1000 2000 5000; do
  echo "=== Rate: $rate ==="
  echo "GET http://localhost:8080/" | \
    vegeta attack -rate=$rate -duration=15s | \
    vegeta report
  echo
done

# Endpoint comparison
for ep in /api/users /api/posts /api/comments; do
  echo "=== $ep ==="
  echo "GET http://localhost:8080$ep" | \
    vegeta attack -rate=200 -duration=15s | \
    vegeta report
done

# Ramp-up simulation (chained attacks)
for rate in 50 100 200 400 800; do
  echo "GET http://localhost:8080/" | \
    vegeta attack -rate=$rate -duration=30s
done | vegeta report

# A/B comparison with plots
echo "GET http://localhost:8080/v1/api" | \
  vegeta attack -rate=200 -duration=30s -name="v1" > v1.bin
echo "GET http://localhost:8080/v2/api" | \
  vegeta attack -rate=200 -duration=30s -name="v2" > v2.bin
vegeta plot v1.bin v2.bin > comparison.html
```

## Dynamic Targets with jq

```bash
# Generate targets from API response
curl -s http://localhost:8080/api/users | \
  jq -r '.[] | "GET http://localhost:8080/api/users/\(.id)"' | \
  vegeta attack -rate=100 -duration=30s | \
  vegeta report

# Generate POST targets
for i in $(seq 1 100); do
  echo "POST http://localhost:8080/api/data"
  echo "Content-Type: application/json"
  echo ""
  echo "{\"id\": $i, \"value\": \"test-$i\"}"
  echo ""
done > dynamic_targets.txt

vegeta attack -targets=dynamic_targets.txt -rate=50 -duration=30s | vegeta report
```

## CI/CD Integration

```bash
#!/bin/bash
# perf-gate.sh — fail build if latency exceeds threshold

RESULT=$(echo "GET http://localhost:8080/api/health" | \
  vegeta attack -rate=200 -duration=30s | \
  vegeta report -type=json)

P99=$(echo "$RESULT" | jq '.latencies."99th"')
P99_MS=$(echo "scale=2; $P99 / 1000000" | bc)

echo "P99 latency: ${P99_MS}ms"

# Fail if p99 > 100ms
if (( $(echo "$P99_MS > 100" | bc -l) )); then
  echo "FAIL: P99 latency ${P99_MS}ms exceeds 100ms threshold"
  exit 1
fi

echo "PASS: Performance within acceptable range"
```

## Library Usage (Go)

```go
package main

import (
    "fmt"
    "time"
    vegeta "github.com/tsenart/vegeta/v12/lib"
)

func main() {
    rate := vegeta.Rate{Freq: 100, Per: time.Second}
    duration := 30 * time.Second
    targeter := vegeta.NewStaticTargeter(vegeta.Target{
        Method: "GET",
        URL:    "http://localhost:8080/api/health",
    })
    attacker := vegeta.NewAttacker()

    var metrics vegeta.Metrics
    for res := range attacker.Attack(targeter, rate, duration, "test") {
        metrics.Add(res)
    }
    metrics.Close()

    fmt.Printf("Requests: %d\n", metrics.Requests)
    fmt.Printf("Rate:     %.2f/s\n", metrics.Rate)
    fmt.Printf("P99:      %s\n", metrics.Latencies.P99)
    fmt.Printf("Success:  %.2f%%\n", metrics.Success*100)
}
```

## Report Fields Explained

```
Requests      [total, rate, throughput]  1500, 50.03, 49.98
Duration      [total, attack, wait]     30.012s, 29.980s, 32.012ms
Latencies     [min, mean, 50, 90, 95, 99, max]  2.1ms, 15.3ms, 12.1ms, 28.4ms, 35.2ms, 52.1ms, 150.3ms
Bytes In      [total, mean]             1500000, 1000.00
Bytes Out     [total, mean]             0, 0.00
Success       [ratio]                   99.80%
Status Codes  [code:count]              200:1497  500:3
Error Set:    500 Internal Server Error
```

## Tips

- Vegeta uses an open-loop (constant-rate) model by default, which avoids the coordinated omission problem that affects wrk and ab.
- Save raw results to a binary file first, then report/plot from the file. This lets you analyze the same data multiple ways without re-running the test.
- Use `vegeta plot` to generate HTML latency plots comparing multiple runs side-by-side for regression detection.
- Pipe targets through `jq` or shell loops to dynamically generate request patterns from API responses.
- Set `-max-workers` to limit concurrency when testing at high rates to control the load generator's resource usage.
- Use the histogram report type (`-type=hist[buckets]`) for SLA validation against specific latency buckets.
- Chain multiple `vegeta attack` commands with different rates to simulate ramp-up patterns through simple Unix piping.
- Use `-name` flag to label attack runs for distinguishing them in combined plots.
- Test with `-keepalive=false` to measure connection setup overhead separately from request handling.
- The JSON report format integrates easily with `jq` for automated threshold checking in CI/CD pipelines.
- Use `-connections` to limit the connection pool size when testing with connection pooling behavior similar to production.
- For very high rates (>10k/s), increase `ulimit -n` and tune `net.ipv4.ip_local_port_range` on the load generator.

## See Also

k6, wrk, ab, curl, hey

## References

- [Vegeta GitHub Repository](https://github.com/tsenart/vegeta)
- [Vegeta Go Library Documentation](https://pkg.go.dev/github.com/tsenart/vegeta/v12/lib)
- [Vegeta README & Usage](https://github.com/tsenart/vegeta/blob/master/README.md)
- [How NOT to Measure Latency (Gil Tene)](https://www.youtube.com/watch?v=lJ8ydIuPFeU)
- [Load Testing with Vegeta (Blog)](https://serialized.net/2017/06/load-testing-with-vegeta-and-bash/)
