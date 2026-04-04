# Pyroscope (Continuous Profiling)

Open-source continuous profiling platform that collects, stores, and queries CPU, memory, and goroutine profiles with minimal overhead.

## Architecture

### Components

```
Application ──> Pyroscope Agent/SDK ──> Pyroscope Server ──> Object Storage
    (push or pull)                          │
                                            v
                              Grafana (pyroscope plugin)
```

### Profile Types

```bash
# CPU        — where is the program spending CPU time
# Memory     — heap allocations (alloc_objects, alloc_space, inuse_objects, inuse_space)
# Goroutine  — goroutine counts and stack traces (Go)
# Mutex      — contention on sync.Mutex / sync.RWMutex (Go)
# Block      — blocking operations (channel ops, select, etc.) (Go)
# Wall       — wall-clock time (includes I/O wait)
# Lock       — lock contention (Java)
# Exceptions — exception allocation profiles (Java/.NET)
```

## Installation

### Docker

```bash
# Run Pyroscope server
docker run -d --name pyroscope \
  -p 4040:4040 \
  grafana/pyroscope:latest

# With persistent storage
docker run -d --name pyroscope \
  -p 4040:4040 \
  -v pyroscope-data:/data \
  grafana/pyroscope:latest
```

### Helm Chart (Kubernetes)

```bash
# Add the Grafana Helm repo
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install Pyroscope
helm install pyroscope grafana/pyroscope -n pyroscope --create-namespace

# Install microservices mode
helm install pyroscope grafana/pyroscope -n pyroscope --create-namespace \
  --set pyroscope.mode=microservices
```

### Binary

```bash
# Download latest release
curl -Lo pyroscope https://github.com/grafana/pyroscope/releases/latest/download/pyroscope_$(uname -s | tr A-Z a-z)_amd64
chmod +x pyroscope

# Run server
./pyroscope -config.file=pyroscope.yaml
```

## Configuration

### Server Config (pyroscope.yaml)

```yaml
target: all

server:
  http_listen_port: 4040

storage:
  backend: s3
  s3:
    bucket_name: pyroscope-profiles
    endpoint: s3.us-east-1.amazonaws.com
    region: us-east-1

limits:
  max_query_length: 24h
  max_query_lookback: 720h             # 30 days

memberlist:
  join_members:
    - pyroscope-memberlist:7946
```

## Language SDKs — Push Mode

### Go (pprof-based)

```go
package main

import (
    "os"
    "runtime"

    "github.com/grafana/pyroscope-go"
)

func main() {
    runtime.SetMutexProfileFraction(5)
    runtime.SetBlockProfileRate(5)

    pyroscope.Start(pyroscope.Config{
        ApplicationName: "my-go-app",
        ServerAddress:   "http://pyroscope:4040",
        Logger:          pyroscope.StandardLogger,
        Tags:            map[string]string{"env": "production", "region": "us-east-1"},
        ProfileTypes: []pyroscope.ProfileType{
            pyroscope.ProfileCPU,
            pyroscope.ProfileAllocObjects,
            pyroscope.ProfileAllocSpace,
            pyroscope.ProfileInuseObjects,
            pyroscope.ProfileInuseSpace,
            pyroscope.ProfileGoroutines,
            pyroscope.ProfileMutexCount,
            pyroscope.ProfileMutexDuration,
            pyroscope.ProfileBlockCount,
            pyroscope.ProfileBlockDuration,
        },
    })
    defer pyroscope.Stop()

    // application code
}
```

### Java (async-profiler)

```java
// build.gradle
// implementation 'io.pyroscope:agent:0.13.0'

// Application startup
import io.pyroscope.javaagent.PyroscopeAgent;
import io.pyroscope.javaagent.config.Config;

PyroscopeAgent.start(
    new Config.Builder()
        .setApplicationName("my-java-app")
        .setServerAddress("http://pyroscope:4040")
        .setProfilingEvent(EventType.ITIMER)
        .setProfilingAlloc("512k")
        .setProfilingLock("10ms")
        .setLabels(Map.of("env", "production"))
        .build()
);
```

```bash
# Or via JVM agent flag
java -javaagent:pyroscope.jar \
  -Dpyroscope.application.name=my-java-app \
  -Dpyroscope.server.address=http://pyroscope:4040 \
  -jar myapp.jar
```

### Python (py-spy)

```python
import pyroscope

pyroscope.configure(
    application_name="my-python-app",
    server_address="http://pyroscope:4040",
    tags={"env": "production", "region": "us-east-1"},
)

# Tag specific code blocks
with pyroscope.tag_wrapper({"controller": "user_handler"}):
    handle_user_request()
```

### eBPF (system-wide, no instrumentation)

```bash
# Run the Pyroscope eBPF agent
docker run -d --name pyroscope-ebpf \
  --privileged \
  --pid=host \
  -e PYROSCOPE_SERVER_ADDRESS=http://pyroscope:4040 \
  -e PYROSCOPE_APPLICATION_NAME=my-system \
  grafana/agent:latest \
  -config.file=/etc/agent/agent.yaml

# Grafana Agent config for eBPF profiling
# agent.yaml
pyroscope:
  profiles_path: /data/pyroscope
  configs:
    - name: ebpf
      scrape_configs:
        - job_name: ebpf
          targets: [{__address__: "localhost"}]
      pyroscope_write:
        endpoint: http://pyroscope:4040
```

## Pull Mode (Scraping)

### Scrape Go pprof Endpoints

```yaml
# pyroscope.yaml — scrape config
scrape_configs:
  - job_name: my-go-service
    scrape_interval: 15s
    targets:
      - localhost:6060
    profiling_config:
      pprof_config:
        cpu:
          enabled: true
          delta: true
          path: /debug/pprof/profile
        memory:
          enabled: true
          delta: true
          path: /debug/pprof/heap
        goroutine:
          enabled: true
          path: /debug/pprof/goroutine
        mutex:
          enabled: true
          delta: true
          path: /debug/pprof/mutex
        block:
          enabled: true
          delta: true
          path: /debug/pprof/block
```

### Kubernetes Service Discovery

```yaml
scrape_configs:
  - job_name: kubernetes-pods
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_pyroscope_io_scrape]
        action: keep
        regex: "true"
      - source_labels: [__meta_kubernetes_pod_annotation_pyroscope_io_port]
        action: replace
        target_label: __address__
        regex: (.+)
        replacement: ${1}
```

## Querying

### API

```bash
# Query profiles
curl -G http://pyroscope:4040/pyroscope/render \
  --data-urlencode 'query=my-go-app.cpu{}' \
  --data-urlencode 'from=now-1h' \
  --data-urlencode 'until=now' \
  --data-urlencode 'format=json'

# Query with label selectors
curl -G http://pyroscope:4040/pyroscope/render \
  --data-urlencode 'query=my-go-app.cpu{env="production",region="us-east-1"}' \
  --data-urlencode 'from=now-30m' \
  --data-urlencode 'until=now'

# List label names
curl http://pyroscope:4040/pyroscope/label-names \
  --data-urlencode 'query=my-go-app.cpu{}'

# List label values
curl -G http://pyroscope:4040/pyroscope/label-values \
  --data-urlencode 'query=my-go-app.cpu{}' \
  --data-urlencode 'label=env'

# Health check
curl http://pyroscope:4040/ready
```

## Grafana Integration

### Datasource Provisioning

```yaml
# grafana/provisioning/datasources/pyroscope.yaml
apiVersion: 1
datasources:
  - name: Pyroscope
    type: grafana-pyroscope-datasource
    access: proxy
    url: http://pyroscope:4040
    jsonData:
      minStep: 15s
```

## Multi-Tenancy

```bash
# Send profiles with tenant ID
curl -X POST http://pyroscope:4040/ingest \
  -H "X-Scope-OrgID: tenant-1" \
  --data-binary @profile.pprof

# Query with tenant ID
curl -G http://pyroscope:4040/pyroscope/render \
  -H "X-Scope-OrgID: tenant-1" \
  --data-urlencode 'query=my-app.cpu{}'
```

## Tips

- Start with CPU and memory (alloc_space) profiles; add mutex/block profiles only when investigating contention
- Set `runtime.SetMutexProfileFraction(5)` and `runtime.SetBlockProfileRate(5)` in Go for low-overhead contention profiling
- Use tag wrappers (Go/Python) to segment profiles by request type, endpoint, or user for targeted analysis
- Pull mode (scraping pprof endpoints) is simpler for Go services; push mode is required for Java and Python
- The eBPF agent profiles all processes system-wide without code changes but requires privileged access
- Keep the scrape interval at 15s for CPU profiles; shorter intervals increase overhead without meaningful detail
- Use diff flame graphs in Grafana to compare profiles between two time ranges (before/after deploy)
- Connect profiles to traces by propagating span IDs — Grafana links Tempo traces to Pyroscope profiles
- In Kubernetes, annotate pods with `pyroscope.io/scrape: "true"` for automatic discovery
- Memory profiles distinguish alloc_space (total allocated) from inuse_space (currently held) — use inuse for leaks
- Set `max_query_lookback` to limit storage scans and prevent expensive queries on old data
- Profile data compresses well (10-20x); object storage costs are modest even at scale

## See Also

- flamegraph
- perf
- ebpf
- grafana
- prometheus
- grafana-tempo

## References

- [Grafana Pyroscope Documentation](https://grafana.com/docs/pyroscope/latest/)
- [Pyroscope GitHub Repository](https://github.com/grafana/pyroscope)
- [Pyroscope Go SDK](https://github.com/grafana/pyroscope-go)
- [Pyroscope Java Agent](https://github.com/grafana/pyroscope-java)
- [async-profiler (Java)](https://github.com/async-profiler/async-profiler)
- [py-spy (Python)](https://github.com/benfred/py-spy)
- [Go pprof Documentation](https://pkg.go.dev/net/http/pprof)
