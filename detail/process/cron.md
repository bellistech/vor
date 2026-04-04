# The Mathematics of cron — Expression Algebra, Scheduling Theory & Execution Guarantees

> *A cron expression is a set intersection problem: the next run is the smallest time point that satisfies all five constraints simultaneously. Understanding this algebra reveals why some schedules are impossible and how to compute the next fire time.*

---

## 1. Cron Expression as Set Intersection

### The Five Fields

A cron expression defines five sets:

$$schedule = minute \cap hour \cap dom \cap month \cap dow$$

Where each field specifies a set of valid values:

| Field | Range | Set Size |
|:---|:---:|:---:|
| Minute | 0-59 | $\leq 60$ |
| Hour | 0-23 | $\leq 24$ |
| Day of Month (dom) | 1-31 | $\leq 31$ |
| Month | 1-12 | $\leq 12$ |
| Day of Week (dow) | 0-7 (0,7=Sun) | $\leq 7$ |

### Cardinality of a Schedule

$$fires\_per\_year \leq |minute| \times |hour| \times |dom \cap dow\_days| \times |month|$$

**Example:** `30 9 * * 1-5` (9:30 AM weekdays)

$$minute = \{30\}, hour = \{9\}, month = \{1..12\}, dow = \{1,2,3,4,5\}$$

$$fires\_per\_year = 1 \times 1 \times \approx 261 \times 1 = 261 \text{ (weekdays/year)}$$

### The DOM/DOW Interaction

When **both** dom and dow are specified (not `*`), cron uses **union** (OR), not intersection:

$$effective\_days = dom\_set \cup dow\_set$$

This is a common pitfall. `0 9 15 * 5` means "9 AM on the 15th OR any Friday," not "9 AM on Fridays that are the 15th."

---

## 2. Next-Run Calculation Algorithm

### The Algorithm

```
function next_run(now, expression):
    t = now + 1_minute (rounded to minute boundary)
    while true:
        if t.month not in month_set:    advance to next valid month, reset day/hour/min
        if t.day not in valid_days:     advance to next valid day, reset hour/min
        if t.hour not in hour_set:      advance to next valid hour, reset min
        if t.minute not in minute_set:  advance to next valid minute
        if all valid: return t
```

### Complexity

Worst case iterations: bounded by the product of set sizes.

$$iterations \leq |month| \times |day| \times |hour| \times |minute|$$

Maximum: $12 \times 31 \times 24 \times 60 = 535,680$ iterations (but practically converges in $< 100$).

### Worked Example

Expression: `0 */6 1,15 * *` (midnight, 6 AM, noon, 6 PM on 1st and 15th)

Current time: 2026-04-03 10:00

$$minute = \{0\}, hour = \{0, 6, 12, 18\}, dom = \{1, 15\}, month = \{1..12\}$$

1. April 3 → not in dom → advance to April 15
2. April 15, 00:00 → all valid

$$T_{next} = 2026\text{-}04\text{-}15\ 00:00$$

$$fires\_per\_month = 2 \times 4 = 8$$

$$fires\_per\_year = 8 \times 12 = 96$$

---

## 3. Execution Timing and Drift

### The One-Minute Resolution

Cron checks schedules once per minute. Maximum delay:

$$T_{delay} = T_{check\_interval} = 60 \text{ seconds}$$

Actual execution time: $T_{scheduled} + U(0, T_{check\_interval})$ where $U$ is uniform.

### Execution Overlap

If a job takes longer than its period:

$$overlap \iff T_{execution} > T_{period}$$

**Example:** Job runs every 5 minutes, takes 7 minutes:

$$\text{At } t=0: \text{job A starts}$$
$$\text{At } t=5: \text{job B starts (A still running)}$$
$$\text{At } t=7: \text{job A finishes}$$
$$\text{At } t=10: \text{job C starts (B still running)}$$

Without lock files: $\lfloor T_{exec} / T_{period} \rfloor + 1$ concurrent instances.

### Missed Runs (cron)

cron does **not** run missed jobs. If the system was down:

$$missed\_runs = \lfloor \frac{T_{downtime}}{T_{period}} \rfloor$$

These runs are silently lost (unlike anacron or systemd timers with `Persistent=true`).

---

## 4. Resource Impact — Fork Rate

### System Call Cost per Job

Each cron job execution:

$$T_{overhead} = T_{fork} + T_{exec} + T_{setuid} + T_{env\_setup} + T_{shell\_startup}$$

| Component | Cost |
|:---|:---:|
| fork() | 0.1-0.5 ms |
| exec(/bin/sh) | 1-3 ms |
| setuid/setgid | 0.01 ms |
| Environment setup | 0.1 ms |
| Shell startup | 5-20 ms |
| **Total overhead** | **7-25 ms** |

### Aggregate Load from Many Jobs

$$load = \sum_{j \in jobs} \frac{T_{exec}(j) + T_{overhead}}{T_{period}(j)}$$

**Example:** 100 jobs running every minute, each taking 2 seconds:

$$load = 100 \times \frac{2 + 0.025}{60} = 100 \times 0.0337 = 3.37 \text{ CPU-seconds/second}$$

This consumes ~3.4 CPU cores continuously.

### Mail Overhead

By default, cron emails any output. For jobs with output:

$$mail\_cost = N_{jobs\_with\_output} \times (T_{sendmail} + message\_size / bandwidth)$$

Redirect to `/dev/null` or a log file to eliminate this.

---

## 5. Special Strings — Predefined Schedules

### Translations

| String | Equivalent | Fires/Day | Fires/Year |
|:---|:---|:---:|:---:|
| `@yearly` | `0 0 1 1 *` | 1/365 | 1 |
| `@monthly` | `0 0 1 * *` | 1/30 | 12 |
| `@weekly` | `0 0 * * 0` | 1/7 | 52 |
| `@daily` | `0 0 * * *` | 1 | 365 |
| `@hourly` | `0 * * * *` | 24 | 8,760 |
| `@reboot` | (on startup) | N/A | N/A |

### Step Values

`*/N` means "every N-th value from the start of the range":

$$set = \{start, start + N, start + 2N, ...\}$$

$$|set| = \lceil \frac{range}{N} \rceil$$

**Example:** `*/7` in minute field: $\{0, 7, 14, 21, 28, 35, 42, 49, 56\}$ — 9 values, **not** every 7 minutes around the clock (resets each hour).

### The /7 Minute Trap

`*/7 * * * *` fires at minutes 0, 7, 14, 21, 28, 35, 42, 49, 56 **each hour**.

$$gap_{hour\_boundary} = 60 - 56 + 0 = 4 \text{ minutes (not 7!)}$$

For true every-7-minutes: use a systemd timer or a loop.

---

## 6. Crontab Security — User Isolation

### Permission Model

$$allowed(user) = (user \in /etc/cron.allow) \lor (user \notin /etc/cron.deny \land \nexists cron.allow)$$

### Privilege Separation

| Crontab | Runs As | Can setuid? |
|:---|:---|:---:|
| User crontab (`crontab -e`) | Owning user | No |
| System (`/etc/crontab`) | Specified user | Yes (root-owned) |
| `/etc/cron.d/*` | Specified user | Yes (root-owned) |

### Environment Isolation

Cron runs with minimal environment:

$$PATH_{cron} = /usr/bin:/bin \text{ (not user's PATH)}$$

$$HOME_{cron} = /home/user \text{ (from passwd)}$$

Missing environment variables are the #1 cause of "works in terminal but not in cron."

---

## 7. anacron — Delayed Catch-Up Scheduling

### anacron Model

anacron runs jobs that were missed during downtime:

$$should\_run = (now - last\_run) > period$$

$$delay = anacron\_delay + random\_delay$$

### Comparison

| Feature | cron | anacron |
|:---|:---:|:---:|
| Missed runs | Lost | Caught up |
| Precision | 1 minute | 1 day minimum |
| Runs during downtime | No | N/A (runs on boot) |
| Repeat interval | Minutes to months | Days to months |
| System requirement | Always on | Can be intermittent |

---

## 8. Summary of cron Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Schedule | $min \cap hour \cap dom \cap month \cap dow$ | Set intersection |
| DOM+DOW | $dom \cup dow$ (when both specified) | Set union |
| Fires per year | $\|min\| \times \|hour\| \times \|days\| \times \|month\|$ | Cardinality |
| Step values | $\{start, start+N, ...\}$ | Arithmetic sequence |
| Overlap condition | $T_{exec} > T_{period}$ | Inequality |
| Aggregate load | $\sum T_{exec}/T_{period}$ | Utilization |
| Missed runs | $\lfloor downtime / period \rfloor$ | Floor division |

## Prerequisites

- set theory, set intersection/union, scheduling theory, modular arithmetic, process lifecycle

---

*A cron expression is discrete mathematics in five fields. The scheduler computes set intersections 60 times per hour, matching the current time against every installed job. When it matches, fork-exec-run. When it doesn't, sleep and try again in a minute.*
