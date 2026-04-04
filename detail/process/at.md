# The Mathematics of at — One-Shot Scheduling, Queue Priority & Resource Modeling

> *at is the simplest scheduler: one job, one time, one execution. But behind that simplicity are queue priority systems, permission models, and timing calculations that make at the right tool when cron is overkill.*

---

## 1. Time Specification — Parsing to Epoch

### Time Expression Model

at converts human time expressions to Unix epoch timestamps:

$$T_{target} = parse(expression) \to epoch\_seconds$$

### Expression Types and Resolution

| Expression | Resolution | Example |
|:---|:---:|:---|
| `now + N minutes` | 1 minute | `at now + 30 minutes` |
| `now + N hours` | 1 minute | `at now + 2 hours` |
| `now + N days` | 1 minute | `at now + 7 days` |
| `HH:MM` | 1 minute | `at 14:30` |
| `HH:MM MMDDYY` | 1 minute | `at 14:30 040326` |
| `midnight` | Fixed | `at midnight` (00:00 next) |
| `noon` | Fixed | `at noon` (12:00 next) |
| `teatime` | Fixed | `at teatime` (16:00 next) |

### Next-Occurrence Logic

If the specified time is in the past today:

$$T_{target} = \begin{cases} today + time & \text{if } time > now \\ tomorrow + time & \text{if } time \leq now \end{cases}$$

**Example:** Current time 15:00, `at 14:00`:

$$T_{target} = tomorrow\ 14:00 = now + 23h$$

---

## 2. Queue System — Priority Levels

### Queue Letters

at supports queues `a` through `z` (plus `A-Z` for batch):

$$queue \in \{a, b, c, ..., z\}$$

Default: queue `a`. Batch (`batch`): queue `b`.

### Nice Value by Queue

Each queue has a different nice value:

$$nice(queue) = 2 \times (queue\_letter - 'a')$$

| Queue | Letter Value | Nice | CPU Priority |
|:---:|:---:|:---:|:---|
| a | 0 | 0 | Default |
| b | 1 | 2 | Slightly lower (batch) |
| c | 2 | 4 | Lower |
| d | 3 | 6 | Lower still |
| z | 25 | 50 | Clamped to 19 |

$$nice_{effective} = \min(nice(queue), 19)$$

### Queue Ordering

Jobs in the same queue execute by scheduled time:

$$order = sort\_by(T_{scheduled})$$

Jobs across queues: all fire at their scheduled time, but with different nice values.

---

## 3. batch — Load-Aware Execution

### The Load Threshold Model

`batch` queues a job that runs when system load drops below a threshold:

$$execute \iff load_{1min} < threshold$$

Default threshold: 1.5 (configurable via atd `-l` flag).

### Waiting Time Distribution

$$E[T_{wait}] = E[\text{time until } load < threshold]$$

This depends on load patterns:

| System State | Typical Wait |
|:---|:---:|
| Idle (load < 0.5) | Immediate |
| Moderate (load 1.0-1.5) | Minutes |
| Heavy (load 2.0-4.0) | 30 min - hours |
| Overloaded (load > 4.0) | Hours to never |

### Check Interval

atd checks load every 60 seconds (default, configurable with `-b`):

$$T_{granularity} = check\_interval = 60s$$

$$T_{wait} = T_{until\_load\_drops} + U(0, check\_interval)$$

---

## 4. Job Storage and Execution

### Job File Format

Each at job is stored in `/var/spool/at/` as a shell script:

$$file\_name = queue\_letter + job\_id + scheduled\_time\_hex$$

### Job File Contents

```
#!/bin/sh
# atrun uid=1000 gid=1000
# mail user 0
cd /original/working/directory
# exported environment variables
VARIABLE=value; export VARIABLE
...
# user commands
the_actual_command
```

### Storage Cost

$$storage_{per\_job} = env\_size + command\_size + header\_size$$

Typical: 2-10 KB per job (environment can be large).

$$total\_storage = N_{pending} \times avg\_job\_size$$

### Environment Capture

at captures the **entire environment** at submission time:

$$env_{job} = env_{current} \text{ at time of } at \text{ command}$$

This includes PATH, HOME, SHELL, and all exported variables. Environment size:

$$|env| = \sum_{var} (|name| + |value| + 2) \text{ bytes}$$

Typical: 1-5 KB.

---

## 5. Execution Cost Model

### Per-Job Overhead

$$T_{execution} = T_{atd\_poll} + T_{fork} + T_{shell\_startup} + T_{env\_restore} + T_{command}$$

| Component | Cost |
|:---|:---:|
| atd poll check | 0 ms (already scheduled) |
| fork/exec | 1-5 ms |
| Shell startup (/bin/sh) | 5-20 ms |
| Environment restoration | 1-5 ms |
| **Overhead** | **7-30 ms** |

### Mail Delivery

By default, at mails any output to the user:

$$mail\_generated \iff |stdout| + |stderr| > 0$$

To suppress: redirect output in the job (`>/dev/null 2>&1`).

Mail cost: $\approx 50-200ms$ per sendmail invocation.

---

## 6. Permission Model

### Access Control

$$allowed(user) = \begin{cases} true & \text{if } user \in /etc/at.allow \\ false & \text{if } user \in /etc/at.deny \\ true & \text{if neither file exists (Debian)} \\ false & \text{if neither file exists (RHEL)} \end{cases}$$

### Priority:

1. If `at.allow` exists: only listed users allowed
2. If only `at.deny` exists: listed users denied, all others allowed
3. If neither exists: distribution-dependent

### Privilege Escalation Risk

at jobs run as the submitting user:

$$uid_{execution} = uid_{submission}$$

A root-submitted at job runs as root. Risk:

$$risk = P(job\_file\_tampered) \times impact(root\_execution)$$

Job files in `/var/spool/at/` are owned by the submitting user with restricted permissions (mode 0700).

---

## 7. at vs Alternatives — Decision Matrix

### When to Use at

| Scenario | Best Tool | Why |
|:---|:---|:---|
| One-time future task | **at** | Designed for this |
| Recurring task | cron | at is one-shot |
| Load-dependent | **batch** | Built-in load check |
| After downtime | systemd timer | at doesn't survive reboot |
| Complex dependencies | systemd timer | at has no dependency model |
| Sub-minute precision | sleep + command | at is minute-precision |

### Reliability Model

$$P(job\_runs) = P(system\_up) \times P(atd\_running) \times P(no\_permission\_change)$$

at jobs do **not** survive reboots unless atd has a persistent queue mechanism (most implementations re-read the spool on startup, so pending jobs do survive).

---

## 8. Summary of at Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Time parsing | Human expression → epoch | Conversion |
| Next occurrence | $time > now ? today : tomorrow$ | Conditional |
| Queue nice | $2 \times (letter - 'a')$ | Linear mapping |
| Batch threshold | $load_{1min} < 1.5$ | Inequality |
| Wait time | $T_{load\_drop} + U(0, 60s)$ | Random + conditional |
| Storage | $N_{jobs} \times avg\_size$ | Linear |
| Execution overhead | $T_{fork} + T_{shell} + T_{env}$ | Constant per job |

## Prerequisites

- epoch time conversion, process scheduling, queue priority, load averages, permission models

---

*at is Unix minimalism applied to scheduling: one job, one time, fire and forget. When cron is a machine gun, at is a sniper rifle — and batch adds a trigger that waits for the right moment.*
