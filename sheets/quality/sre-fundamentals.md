# SRE Fundamentals (Reliability, Observability, and Operations)

A practitioner's reference for Site Reliability Engineering — from SLI/SLO/SLA definitions to incident management and capacity planning.

## Service Level Indicators (SLIs)

### Defining Good SLIs

```
SLI = (good events / total events) * 100%

Latency SLI:
  good events  = requests completing in < 300ms
  total events = all requests
  SLI = (requests < 300ms) / (total requests) * 100%

Availability SLI:
  good events  = non-5xx responses
  total events = all responses
  SLI = (total - 5xx) / total * 100%

Throughput SLI:
  good events  = successfully processed jobs
  total events = all submitted jobs

Error Rate SLI:
  good events  = successful responses
  total events = all responses
  SLI = 1 - (errors / total)
```

### Common SLI Types by Service

| Service Type | Primary SLI | Measurement |
|---|---|---|
| Request-driven (API) | Availability, latency | Load balancer logs |
| Pipeline/batch | Freshness, correctness | Pipeline metrics |
| Storage | Durability, throughput | Storage system metrics |
| Streaming | Freshness, throughput | Consumer lag |

### Measuring SLIs

```bash
# Prometheus queries for SLIs

# Availability SLI (non-5xx rate over 30 days)
sum(rate(http_requests_total{status!~"5.."}[30d])) /
sum(rate(http_requests_total[30d])) * 100

# Latency SLI (p99 < 300ms over 30 days)
histogram_quantile(0.99,
  sum(rate(http_request_duration_seconds_bucket[30d])) by (le)
)

# Latency SLI as proportion of fast requests
sum(rate(http_request_duration_seconds_bucket{le="0.3"}[30d])) /
sum(rate(http_request_duration_seconds_count[30d])) * 100
```

## Service Level Objectives (SLOs)

### Setting SLOs

```
SLO = SLI target over a measurement window

Example:
  SLI: Availability (non-5xx responses / total responses)
  Target: 99.9%
  Window: 30-day rolling

  "99.9% of requests will return non-5xx responses
   over any rolling 30-day period"
```

### SLO Document Template

```yaml
service: payment-api
slos:
  - name: availability
    description: "Payment API returns successful responses"
    sli:
      type: availability
      good_event: "response status < 500"
      total_event: "all responses"
    target: 99.95%
    window: 30d
    consequences:
      - "Below target: freeze non-critical deploys"
      - "Burn rate > 2x: page on-call"

  - name: latency
    description: "Payment API responds quickly"
    sli:
      type: latency
      threshold: 500ms
      percentile: 99
    target: 99.0%
    window: 30d
    consequences:
      - "Below target: prioritize performance work"
```

## Service Level Agreements (SLAs)

```
SLA = contractual commitment with consequences for breach

Relationship:
  SLA ≤ SLO ≤ measured SLI (ideally)

  SLI measured: 99.97%  ← what we actually achieve
  SLO internal: 99.95%  ← our operational target
  SLA external: 99.9%   ← what we promise customers

Always set SLA below SLO to provide a safety buffer.
```

## Error Budgets

### Calculation

```
Error budget = 1 - SLO target

Example (30-day window):
  SLO = 99.9%
  Error budget = 0.1% = 0.001

  Total minutes in 30 days: 43,200
  Budget in minutes: 43,200 * 0.001 = 43.2 minutes of downtime

  Total requests (1000 req/s): 2,592,000,000
  Budget in errors: 2,592,000,000 * 0.001 = 2,592,000 errors allowed
```

### Burn Rate

```go
// Burn rate = actual consumption rate / ideal consumption rate
// Burn rate of 1.0 = consuming budget exactly as planned
// Burn rate of 2.0 = will exhaust budget in half the window

type ErrorBudget struct {
    SLOTarget    float64       // e.g., 0.999
    WindowSize   time.Duration // e.g., 30 * 24 * time.Hour
    WindowStart  time.Time
}

func (eb *ErrorBudget) BurnRate(goodEvents, totalEvents float64) float64 {
    if totalEvents == 0 {
        return 0
    }
    errorRate := 1 - (goodEvents / totalEvents)
    budgetRate := 1 - eb.SLOTarget // ideal error rate
    return errorRate / budgetRate
}

func (eb *ErrorBudget) RemainingBudget(goodEvents, totalEvents float64) float64 {
    consumed := (totalEvents - goodEvents) / totalEvents
    budget := 1 - eb.SLOTarget
    return budget - consumed
}

func (eb *ErrorBudget) TimeToExhaustion(burnRate float64) time.Duration {
    if burnRate <= 0 {
        return time.Duration(math.MaxInt64)
    }
    elapsed := time.Since(eb.WindowStart)
    remaining := eb.WindowSize - elapsed
    return time.Duration(float64(remaining) / burnRate)
}
```

### Multi-Window Burn Rate Alerts

```yaml
# Google SRE recommended alerting strategy
alerts:
  - name: "High burn rate - page"
    # 2% budget consumed in 1 hour = 14.4x burn rate
    # Short window catches fast burns
    condition: burn_rate_1h > 14.4 AND burn_rate_5m > 14.4
    severity: page
    action: "Wake on-call immediately"

  - name: "Moderate burn rate - page"
    # 5% budget consumed in 6 hours = 6x burn rate
    condition: burn_rate_6h > 6 AND burn_rate_30m > 6
    severity: page
    action: "Page during business hours"

  - name: "Slow burn rate - ticket"
    # 10% budget consumed in 3 days = 1x burn rate
    condition: burn_rate_3d > 1 AND burn_rate_6h > 1
    severity: ticket
    action: "Create ticket, investigate this week"
```

## Toil Measurement

### Identifying Toil

```
Toil characteristics:
  - Manual (human does it, not automation)
  - Repetitive (done more than once)
  - Automatable (could be scripted)
  - Tactical (interrupt-driven, reactive)
  - No enduring value (doesn't improve the system)
  - Scales with service growth (O(n) with load)

NOT toil:
  - Incident response (learning value)
  - Architecture design (enduring value)
  - Writing automation (eliminates future toil)
```

### Automation ROI

```
Time saved per occurrence: T_save
Frequency per month: F
Automation development cost: T_dev
Maintenance cost per month: T_maint

Monthly savings: T_save * F - T_maint
Break-even months: T_dev / (T_save * F - T_maint)

Example:
  Manual task: 30 min, 20 times/month = 600 min/month
  Automation: 3 days to build, 30 min/month maintenance
  Monthly savings: 600 - 30 = 570 min = 9.5 hours
  Break-even: 1440 / 570 = 2.5 months
```

## Incident Management

### Lifecycle

```
Detect ──→ Triage ──→ Mitigate ──→ Resolve ──→ Postmortem
  │           │            │           │            │
  │           │            │           │            │
Alert      Severity     Stop the    Fix root     Learn &
fires      assessed     bleeding    cause        prevent
```

### Severity Levels

| Level | Impact | Response Time | Examples |
|---|---|---|---|
| SEV1 | Complete outage, data loss risk | < 5 min | Site down, data corruption |
| SEV2 | Major feature broken, degraded | < 15 min | Payments failing, high errors |
| SEV3 | Minor feature broken | < 1 hour | Search slow, UI glitch |
| SEV4 | Cosmetic, non-urgent | Next business day | Typo, minor UI issue |

### Blameless Postmortem Template

```markdown
## Incident Postmortem: [Title]

**Date**: YYYY-MM-DD
**Duration**: X hours Y minutes
**Severity**: SEV-N
**Incident Commander**: [Name]
**Author**: [Name]

### Summary
One-paragraph description of what happened.

### Impact
- Users affected: N
- Revenue impact: $X
- Error budget consumed: Y%

### Timeline (all times UTC)
- HH:MM - Alert fired: [description]
- HH:MM - IC assigned, investigation began
- HH:MM - Root cause identified
- HH:MM - Mitigation applied
- HH:MM - Full resolution confirmed

### Root Cause
Technical explanation of the underlying cause.

### Contributing Factors
- Factor 1: [description]
- Factor 2: [description]

### What Went Well
- Alerting caught the issue within N minutes
- Runbook was accurate and up to date

### What Went Poorly
- Escalation took too long
- Missing monitoring for X

### Action Items
| Item | Owner | Priority | Due Date |
|---|---|---|---|
| Add monitoring for X | @engineer | P1 | YYYY-MM-DD |
| Update runbook for Y | @oncall | P2 | YYYY-MM-DD |
| Fix root cause Z | @team | P1 | YYYY-MM-DD |

### Lessons Learned
What systemic changes prevent recurrence?
```

## On-Call Best Practices

### Escalation Policy

```yaml
escalation:
  level_1:
    who: primary on-call
    timeout: 5_minutes
    contact: pager + sms

  level_2:
    who: secondary on-call
    timeout: 10_minutes
    contact: pager + sms + phone

  level_3:
    who: engineering manager
    timeout: 15_minutes
    contact: phone

  level_4:
    who: VP engineering
    timeout: 30_minutes
    contact: phone

rotation:
  schedule: weekly
  handoff: Monday 10:00 UTC
  overlap: 1_hour  # both primary and incoming on-call available
  max_consecutive: 1_week
  cooldown: 2_weeks  # minimum gap between on-call shifts
```

### Runbook Structure

```markdown
## Runbook: [Alert Name]

### Symptoms
What does this alert mean? What are users experiencing?

### Severity Assessment
- Check dashboard: [link]
- If error rate > 5%: SEV2
- If error rate > 50%: SEV1

### Quick Mitigation
1. Check recent deploys: `kubectl rollout history deployment/app`
2. If recent deploy: rollback: `kubectl rollout undo deployment/app`
3. Check downstream dependencies: [dashboard link]

### Investigation Steps
1. Check logs: `kubectl logs -l app=myapp --since=15m`
2. Check metrics: [Grafana dashboard link]
3. Check database: [query to run]

### Escalation
If not resolved in 30 minutes, escalate to [team/person].
```

## Capacity Planning

```
Process:
  1. Measure current usage
  2. Model growth rate
  3. Identify bottleneck resources
  4. Calculate time to exhaustion
  5. Plan and provision ahead

Key metrics to forecast:
  - CPU utilization (target: < 70% sustained)
  - Memory usage (target: < 80%)
  - Disk I/O and capacity
  - Network bandwidth
  - Request rate and connection count
```

```bash
# Load testing with k6
k6 run --vus 100 --duration 5m load-test.js

# Forecasting with linear regression (simple)
# If growing 10% monthly and at 50% capacity:
# months_remaining = log(max/current) / log(1 + growth_rate)
# = log(0.7/0.5) / log(1.1) = 3.5 months until 70% threshold
```

## Monitoring Methods

### USE Method (Resources)

```
For every resource (CPU, memory, disk, network):
  U - Utilization: % time resource is busy
  S - Saturation: work queued / waiting
  E - Errors: error events count
```

### RED Method (Services)

```
For every service:
  R - Rate: requests per second
  E - Errors: failed requests per second
  D - Duration: latency distribution (histograms)
```

### Four Golden Signals

```
1. Latency    - Time to serve a request (success vs error latency)
2. Traffic    - Demand on the system (req/s, sessions, transactions)
3. Errors     - Rate of failed requests (explicit and implicit)
4. Saturation - How "full" the service is (queue depth, memory %)
```

```go
// Implementing the Four Golden Signals in Go with Prometheus
import "github.com/prometheus/client_golang/prometheus"

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "Latency of HTTP requests",
            Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
        },
        []string{"method", "path", "status"},
    )
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests (traffic + errors)",
        },
        []string{"method", "path", "status"},
    )
    inFlightRequests = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "http_requests_in_flight",
            Help: "Current number of in-flight requests (saturation)",
        },
    )
)

func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        inFlightRequests.Inc()
        defer inFlightRequests.Dec()

        start := time.Now()
        wrapped := &statusRecorder{ResponseWriter: w, status: 200}
        next.ServeHTTP(wrapped, r)

        duration := time.Since(start).Seconds()
        status := strconv.Itoa(wrapped.status)

        requestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
        requestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
    })
}
```

## Change Management

```
Deployment checklist:
  [ ] Change reviewed and approved
  [ ] Canary deployment (1-5% traffic)
  [ ] Monitor error rate for 15 minutes
  [ ] Gradual rollout (10% → 25% → 50% → 100%)
  [ ] Rollback plan documented and tested
  [ ] Feature flags for quick disable
  [ ] Communication to stakeholders

Rollback criteria:
  - Error rate increases > 1% above baseline
  - Latency p99 increases > 50% above baseline
  - Any SEV1/SEV2 incident
```

## Tips

- SLOs should be based on user experience, not internal metrics
- Start with fewer, meaningful SLOs rather than many vague ones
- Error budgets create alignment: spend budget on velocity, save it for reliability
- Toil should be < 50% of an SRE's time; track it quarterly
- Blameless postmortems focus on systemic fixes, not individual blame
- Monitor symptom-based (errors users see), not cause-based (CPU usage)
- Capacity plan for 3-6 months ahead; sudden growth requires pre-provisioned headroom
- On-call rotations need at least 8 people for sustainable coverage

## See Also

- `detail/quality/sre-fundamentals.md` — nines math, queuing theory, MTBF/MTTR
- `sheets/patterns/microservices-patterns.md` — circuit breakers, health checks
- `sheets/performance/caching-patterns.md` — caching for reliability

## References

- "Site Reliability Engineering" by Betsy Beyer et al. (Google, 2016)
- "The Site Reliability Workbook" (Google, 2018)
- "Implementing Service Level Objectives" by Alex Hidalgo (O'Reilly, 2020)
- Google SRE Books: https://sre.google/books/
