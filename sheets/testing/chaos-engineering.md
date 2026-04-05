# Chaos Engineering (Fault Injection + Resilience)

Complete reference for chaos engineering — principles, steady state hypothesis, tools (Litmus, chaos-mesh, toxiproxy, tc netem, pumba), blast radius control, gameday planning, and chaos in CI/CD.

## Principles of Chaos Engineering

### The Five Principles

```
1. Build a Hypothesis Around Steady State Behavior
   "Normal" looks like: p99 latency < 200ms, error rate < 0.1%, throughput > 1000 rps

2. Vary Real-World Events
   Inject failures that actually happen: server crashes, network partitions,
   disk full, clock skew, dependency slowdowns

3. Run Experiments in Production
   (or as close to production as possible)

4. Automate Experiments to Run Continuously
   Not a one-time event — embed in CI/CD

5. Minimize Blast Radius
   Start small, expand gradually, have a kill switch
```

### Steady State Hypothesis

```
Before Experiment:
  - p99 latency: 180ms
  - Error rate: 0.05%
  - Throughput: 1200 rps
  - Circuit breaker: closed

Experiment: Kill 1 of 3 API server instances

Expected Outcome:
  - p99 latency: < 300ms (degraded but functional)
  - Error rate: < 1% (brief spike, then recovery)
  - Throughput: > 800 rps (reduced but serving)
  - Circuit breaker: stays closed (or opens and recovers)

Actual Outcome: [fill in after experiment]
```

## Network Fault Injection

### toxiproxy

Programmable TCP proxy for simulating network conditions.

```bash
# Install
brew install toxiproxy       # macOS
go install github.com/Shopify/toxiproxy/v2/cmd/toxiproxy-server@latest
go install github.com/Shopify/toxiproxy/v2/cmd/toxiproxy-cli@latest

# Start server
toxiproxy-server &

# Create proxy
toxiproxy-cli create postgres_proxy \
    --listen 0.0.0.0:5433 \
    --upstream localhost:5432

# Add latency (100ms ± 50ms)
toxiproxy-cli toxic add postgres_proxy \
    -t latency \
    -a latency=100 \
    -a jitter=50

# Add bandwidth limit (1KB/s)
toxiproxy-cli toxic add postgres_proxy \
    -t bandwidth \
    -a rate=1

# Simulate connection timeout
toxiproxy-cli toxic add postgres_proxy \
    -t timeout \
    -a timeout=5000

# Simulate connection reset
toxiproxy-cli toxic add postgres_proxy \
    -t reset_peer \
    -a timeout=1000

# Sever connection entirely
toxiproxy-cli toggle postgres_proxy

# Remove all toxics
toxiproxy-cli toxic remove postgres_proxy -n latency_downstream

# List proxies and their toxics
toxiproxy-cli list
toxiproxy-cli inspect postgres_proxy
```

### toxiproxy in Go Tests

```go
import (
    "github.com/Shopify/toxiproxy/v2/toxiproxytest"
    toxiclient "github.com/Shopify/toxiproxy/v2/client"
)

func TestWithNetworkFault(t *testing.T) {
    // Create toxiproxy server for testing
    srv := toxiproxytest.NewServer()
    defer srv.Close()

    client := toxiclient.NewClient(srv.Addr)

    proxy, err := client.CreateProxy("redis", "localhost:26379", "localhost:6379")
    if err != nil {
        t.Fatal(err)
    }

    // Add 500ms latency
    _, err = proxy.AddToxic("latency", "latency", "downstream", 1.0, toxiclient.Attributes{
        "latency": 500,
        "jitter":  100,
    })
    if err != nil {
        t.Fatal(err)
    }

    // Test that your application handles the latency gracefully
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    result, err := myService.GetCachedValue(ctx, "key")
    // Assert that fallback behavior works
    assert.NoError(t, err) // should not error, should use fallback
}
```

### tc netem (Linux Network Emulation)

```bash
# Add 100ms latency with 10ms jitter (normal distribution)
sudo tc qdisc add dev eth0 root netem delay 100ms 10ms distribution normal

# Add packet loss (5%)
sudo tc qdisc add dev eth0 root netem loss 5%

# Add packet corruption (1%)
sudo tc qdisc add dev eth0 root netem corrupt 1%

# Add packet duplication (2%)
sudo tc qdisc add dev eth0 root netem duplicate 2%

# Add packet reordering (25% of packets delayed by 10ms)
sudo tc qdisc add dev eth0 root netem delay 10ms reorder 25% 50%

# Combine effects
sudo tc qdisc add dev eth0 root netem delay 50ms 20ms loss 2% corrupt 0.5%

# Rate limiting (1mbit)
sudo tc qdisc add dev eth0 root netem rate 1mbit

# Remove all netem rules
sudo tc qdisc del dev eth0 root

# Show current rules
tc qdisc show dev eth0
```

### Target Specific Traffic

```bash
# Only affect traffic to port 5432 (PostgreSQL)
sudo tc qdisc add dev eth0 root handle 1: prio
sudo tc qdisc add dev eth0 parent 1:3 handle 30: netem delay 200ms
sudo tc filter add dev eth0 parent 1:0 protocol ip u32 \
    match ip dport 5432 0xffff flowid 1:3
```

## Container Chaos

### pumba

```bash
# Kill a container
pumba kill --signal SIGKILL myapp

# Kill with SIGTERM (graceful)
pumba kill --signal SIGTERM myapp

# Pause a container for 30 seconds
pumba pause --duration 30s myapp

# Remove a container
pumba rm myapp

# Network delay on container
pumba netem --duration 60s delay --time 200 --jitter 50 myapp

# Network loss on container
pumba netem --duration 30s loss --percent 10 myapp

# Stress container CPU
pumba stress --duration 30s --stressors "--cpu 4" myapp

# Recurring chaos (every 5 minutes, kill random container)
pumba --interval 5m kill --signal SIGKILL "re2:myapp-.*"
```

## Kubernetes Chaos

### Litmus

```yaml
# litmus-experiment.yaml
apiVersion: litmuschaos.io/v1alpha1
kind: ChaosEngine
metadata:
  name: api-chaos
  namespace: default
spec:
  appinfo:
    appns: default
    applabel: "app=api-server"
    appkind: deployment
  engineState: active
  chaosServiceAccount: litmus-admin
  experiments:
    - name: pod-delete
      spec:
        components:
          env:
            - name: TOTAL_CHAOS_DURATION
              value: "30"
            - name: CHAOS_INTERVAL
              value: "10"
            - name: FORCE
              value: "false"
```

```bash
# Install Litmus
kubectl apply -f https://litmuschaos.github.io/litmus/litmus-operator-v3.0.0.yaml

# Run experiment
kubectl apply -f litmus-experiment.yaml

# Check result
kubectl get chaosresult api-chaos-pod-delete -o jsonpath='{.status.experimentStatus.verdict}'
```

### chaos-mesh

```yaml
# network-delay.yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: NetworkChaos
metadata:
  name: api-network-delay
  namespace: default
spec:
  action: delay
  mode: all
  selector:
    namespaces:
      - default
    labelSelectors:
      app: api-server
  delay:
    latency: "200ms"
    correlation: "50"
    jitter: "50ms"
  duration: "5m"
  scheduler:
    cron: "@every 30m"
```

```yaml
# pod-kill.yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: api-pod-kill
spec:
  action: pod-kill
  mode: one
  selector:
    namespaces:
      - default
    labelSelectors:
      app: api-server
  scheduler:
    cron: "@every 1h"
```

```bash
# Install chaos-mesh
curl -sSL https://mirrors.chaos-mesh.org/latest/install.sh | bash

# Apply experiment
kubectl apply -f network-delay.yaml

# Check status
kubectl get networkchaos
kubectl describe networkchaos api-network-delay
```

## Process Chaos

### Kill Signals

```bash
# Graceful shutdown (SIGTERM) — process can clean up
kill -SIGTERM $PID

# Immediate kill (SIGKILL) — no cleanup
kill -9 $PID

# Test the difference
# SIGTERM: triggers graceful shutdown handlers, closes connections, flushes buffers
# SIGKILL: simulates OOM kill, power failure, kernel panic
```

### Go Graceful Shutdown Test

```go
func TestGracefulShutdown(t *testing.T) {
    srv := &http.Server{Addr: ":0", Handler: myHandler}

    ln, err := net.Listen("tcp", ":0")
    require.NoError(t, err)

    go srv.Serve(ln)

    // Start a long-running request
    done := make(chan struct{})
    go func() {
        resp, err := http.Get("http://" + ln.Addr().String() + "/slow")
        require.NoError(t, err)
        require.Equal(t, 200, resp.StatusCode)
        close(done)
    }()

    time.Sleep(100 * time.Millisecond) // let request start

    // Initiate graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    err = srv.Shutdown(ctx)
    require.NoError(t, err)

    // Verify in-flight request completed
    select {
    case <-done:
        // success — request finished before shutdown completed
    case <-time.After(10 * time.Second):
        t.Fatal("request did not complete during graceful shutdown")
    }
}
```

### Disk Full Testing

```bash
# Create a full filesystem (Linux)
dd if=/dev/zero of=/tmp/full.img bs=1M count=50
mkfs.ext4 /tmp/full.img
mkdir /tmp/full
mount -o loop /tmp/full.img /tmp/full
dd if=/dev/zero of=/tmp/full/fill bs=1M count=45  # fill most of it

# macOS: create small disk image
hdiutil create -size 50m -fs HFS+ -volname TestDisk /tmp/testdisk.dmg
hdiutil attach /tmp/testdisk.dmg

# Clean up
umount /tmp/full  # Linux
hdiutil detach /Volumes/TestDisk  # macOS
```

## Blast Radius Control

### Progressive Expansion

```
Phase 1: Single non-critical pod in staging        (blast radius: 1 pod)
Phase 2: Single pod in production canary            (blast radius: 1 pod, real traffic)
Phase 3: Random pod in production (10% traffic)     (blast radius: 1 of N pods)
Phase 4: Multiple pods in production (25% traffic)  (blast radius: N/4 pods)
Phase 5: Full zone failure in production            (blast radius: 1 AZ)
```

### Kill Switch Pattern

```go
type ChaosExperiment struct {
    Name     string
    Active   bool
    KillChan chan struct{}
}

func (e *ChaosExperiment) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            log.Printf("experiment %s cancelled by context", e.Name)
            return ctx.Err()
        case <-e.KillChan:
            log.Printf("experiment %s killed by operator", e.Name)
            return nil
        default:
            if !e.Active {
                return nil
            }
            e.injectFault()
            time.Sleep(e.Interval)
        }
    }
}
```

## Gameday Planning

### Template

```
## Gameday: [Title]
Date: YYYY-MM-DD
Participants: [names and roles]

### Hypothesis
[What we expect to happen]

### Steady State
- Metric A: [current value]
- Metric B: [current value]

### Experiment
1. [Step-by-step fault injection]
2. [Observation points]
3. [Rollback procedure]

### Abort Criteria
- Error rate > 5%
- p99 latency > 2s
- Any 5xx from critical path

### Results
- [What actually happened]
- [Surprises]
- [Action items]

### Follow-Up
- [ ] Fix: [identified issue]
- [ ] Monitor: [new alert]
- [ ] Test: [new chaos test]
```

## Chaos in CI/CD

### Integration Test with Fault Injection

```yaml
# .github/workflows/chaos.yml
name: Chaos Tests
on:
  schedule:
    - cron: '0 6 * * 1-5'  # weekdays at 6am
  workflow_dispatch:

jobs:
  chaos:
    runs-on: ubuntu-latest
    services:
      toxiproxy:
        image: ghcr.io/shopify/toxiproxy:latest
        ports:
          - 8474:8474
          - 5433:5433

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Run chaos tests
        run: go test -tags=chaos -v -timeout=5m ./tests/chaos/...
```

## Netflix Simian Army Legacy

```
Tool               | Fault Type                    | Modern Equivalent
Chaos Monkey       | Random instance termination    | Litmus pod-delete
Latency Monkey     | Artificial network delay       | chaos-mesh NetworkChaos
Conformity Monkey  | Non-conformant instance checks | OPA/Gatekeeper
Security Monkey    | Security vulnerability scans   | Falco, Trivy
Chaos Gorilla      | Full AZ outage                 | chaos-mesh AZChaos
Chaos Kong         | Full region outage             | AWS FIS
Janitor Monkey     | Resource cleanup               | Cloud Custodian
```

## Tips

- Always define abort criteria before starting a chaos experiment
- Start with the smallest blast radius possible and expand gradually
- Chaos engineering is NOT just breaking things — it is hypothesis-driven experimentation
- Run chaos experiments during business hours when engineers are available to respond
- toxiproxy is the best choice for network fault injection in integration tests
- Use SIGTERM before SIGKILL — test graceful shutdown before simulating crashes
- Monitor your steady state metrics before, during, and after experiments
- Document every gameday — the findings are valuable institutional knowledge
- Automate proven chaos experiments in CI/CD for continuous validation
- Never run chaos experiments without a kill switch and rollback plan

## See Also

- `sheets/testing/integration-testing.md` — test infrastructure setup
- `detail/testing/chaos-engineering.md` — MTTR modeling and reliability mathematics
- `sheets/quality/twelve-factor.md` — Factor IX (Disposability) and graceful shutdown

## References

- https://principlesofchaos.org/ — Principles of Chaos Engineering
- https://github.com/Shopify/toxiproxy — toxiproxy
- https://litmuschaos.io/ — LitmusChaos
- https://chaos-mesh.org/ — Chaos Mesh
- https://github.com/alexei-led/pumba — Pumba
- https://netflixtechblog.com/the-netflix-simian-army-16e57fbab116 — Netflix Simian Army
