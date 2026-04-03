# The Mathematics of systemd Timers — Calendar Expressions, Monotonic Scheduling & Accuracy

> *systemd timers are a superset of cron with two fundamental modes: wall-clock (calendar) triggers and monotonic (elapsed-time) triggers. Each mode has its own scheduling algebra, accuracy tradeoffs, and next-fire calculation.*

---

## 1. Calendar Expressions — Set Intersection Model

### The Expression Grammar

An `OnCalendar=` expression defines a **set of time points**:

$$trigger\_set = DOW \cap Year \cap Month \cap Day \cap Hour \cap Minute \cap Second$$

Each component is a set of valid values:

$$Hour = \{h : h \in expression\}$$

**Example:** `Mon,Fri *-*-1..7 09:00:00`

$$DOW = \{Mon, Fri\}$$
$$Day = \{1, 2, 3, 4, 5, 6, 7\}$$
$$Hour = \{9\}, Minute = \{0\}, Second = \{0\}$$

### Next-Fire Calculation

Given current time $t_{now}$, find the smallest $t > t_{now}$ in the trigger set.

Algorithm:
1. Start from $t_{now}$, round up to next valid second
2. If second valid → check minute → check hour → check day → check month → check DOW
3. If any component invalid → increment that component, reset lower components
4. Repeat until all components valid

$$T_{next\_fire} = \min\{t \in trigger\_set : t > t_{now}\}$$

### Worked Example

Expression: `*-*-* 06,18:30:00` (6:30 AM and 6:30 PM daily)

$$trigger\_set = \{..., 2026\text{-}04\text{-}03\ 06:30, 2026\text{-}04\text{-}03\ 18:30, 2026\text{-}04\text{-}04\ 06:30, ...\}$$

Current time: 2026-04-03 10:00:00

$$T_{next} = 2026\text{-}04\text{-}03\ 18:30:00 \quad (\Delta = 8h\ 30m)$$

### Repetition Syntax

`OnCalendar=*-*-* *:00/15:00` means every 15 minutes:

$$Minute = \{0, 15, 30, 45\}$$

$$fires\_per\_day = 24 \times 4 = 96$$

$$interval = 15 \text{ minutes} = 900 \text{ seconds}$$

---

## 2. Monotonic Timers — Elapsed Time Scheduling

### OnBootSec and OnUnitActiveSec

$$T_{fire} = T_{reference} + delay$$

| Directive | Reference Point | Fires After |
|:---|:---|:---|
| `OnBootSec=5min` | System boot | 5 min after boot |
| `OnStartupSec=5min` | systemd start (user) | 5 min after login |
| `OnActiveSec=1h` | Timer activation | 1 hour after timer starts |
| `OnUnitActiveSec=30min` | Last unit activation | 30 min after service last ran |
| `OnUnitInactiveSec=1h` | Last unit deactivation | 1 hour after service stopped |

### Repeating Monotonic Timer

With `OnUnitActiveSec=30min`:

$$T_{fire}(n) = T_{activation}(n-1) + 30 \text{ min}$$

This creates a **fixed-delay** schedule (not fixed-rate):

$$actual\_period = 30min + T_{service\_duration}$$

If the service takes 5 minutes:

$$effective\_interval = 35 \text{ minutes}$$

$$fires\_per\_day = \frac{1440}{35} = 41.1 \approx 41$$

### OnUnitInactiveSec — Fixed Spacing

With `OnUnitInactiveSec=30min`:

$$T_{fire}(n) = T_{deactivation}(n-1) + 30 \text{ min}$$

$$actual\_period = 30min + T_{service\_duration}$$

Same behavior as `OnUnitActiveSec` but measured from service stop instead of start.

---

## 3. Accuracy and Randomization

### AccuracySec — Coalescing Window

`AccuracySec=` defines a window within which the timer may fire to coalesce with other timers:

$$T_{actual} \in [T_{scheduled}, T_{scheduled} + AccuracySec]$$

Default: `AccuracySec=1min`.

### Power Savings from Coalescing

If $n$ timers would each wake the system separately:

$$wakeups_{uncoalesced} = n$$

$$wakeups_{coalesced} = \lceil \frac{n \times avg\_spread}{AccuracySec} \rceil \leq n$$

**Example:** 10 timers within a 5-minute window, `AccuracySec=1min`:

$$wakeups \leq \lceil \frac{5}{1} \rceil = 5 \text{ (vs 10 without coalescing)}$$

$$savings = 1 - \frac{5}{10} = 50\%$$

### RandomizedDelaySec — Jitter

`RandomizedDelaySec=` adds uniform random delay:

$$T_{actual} = T_{scheduled} + U(0, RandomizedDelaySec)$$

Where $U(a, b)$ is a uniform random variable.

**Purpose:** Prevent thundering herd when many machines have the same timer:

$$P(\text{all } n \text{ machines fire within } \Delta t) = \left(\frac{\Delta t}{RandomizedDelaySec}\right)^n$$

With `RandomizedDelaySec=1h` and 100 machines:

$$P(\text{all within 1 min}) = \left(\frac{1}{60}\right)^{100} \approx 10^{-178}$$

---

## 4. Persistent Timers — Catching Up After Downtime

### The Persistent= Directive

When `Persistent=true`, if the system was off during a scheduled fire time:

$$T_{catchup} = T_{boot} \text{ (fires immediately on boot)}$$

### Missed Run Calculation

$$missed\_runs = \lfloor \frac{T_{downtime}}{period} \rfloor$$

With `Persistent=true`, only **one** catch-up run happens (not all missed runs).

**Example:** Hourly timer, system down for 8 hours:

$$missed = \lfloor 8/1 \rfloor = 8$$

On boot: 1 catch-up run executes, not 8.

---

## 5. Timer vs Cron — Comparison

### Feature Parity

| Feature | cron | systemd timer |
|:---|:---:|:---:|
| Calendar scheduling | Yes | Yes |
| Monotonic scheduling | No | Yes |
| Randomized delay | No | Yes |
| Dependencies | No | Yes |
| Resource limits | No | Yes (cgroup) |
| Logging | Custom | journald |
| Persistent/catch-up | anacron (limited) | Yes |
| Per-second precision | No (minute min) | Yes |
| Coalescing | No | Yes (AccuracySec) |

### Calendar Expression Translation

| cron | systemd OnCalendar | Period |
|:---|:---|:---:|
| `*/5 * * * *` | `*-*-* *:00/5:00` | 5 minutes |
| `0 */2 * * *` | `*-*-* 00/2:00:00` | 2 hours |
| `0 9 * * 1-5` | `Mon..Fri *-*-* 09:00:00` | Weekday 9 AM |
| `0 0 1 * *` | `*-*-01 00:00:00` | Monthly |
| `0 0 * * 0` | `Sun *-*-* 00:00:00` | Weekly Sunday |

---

## 6. WakeSystem — Power State Interaction

### WakeSystem=true

The timer can wake the system from suspend/hibernate:

$$T_{wake} = T_{next\_fire} - T_{suspend}$$

Power cost of waking:

$$E_{wake} = P_{resume} \times T_{resume} + P_{idle} \times T_{service} + E_{suspend\_again}$$

| Component | Duration | Power |
|:---|:---:|:---:|
| Resume from suspend | 2-10 s | 50-100 W |
| Service execution | Variable | Idle + CPU |
| Return to suspend | 1-5 s | 50-100 W |

**Break-even:** Only worth waking if the task value exceeds $E_{wake}$.

---

## 7. Testing and Verification

### systemd-analyze calendar

Parses expressions and shows next N fire times:

$$iterations = \frac{time\_range}{avg\_period}$$

### Timer Execution Order

When multiple timers fire simultaneously:

$$order = \text{arbitrary (parallel by default)}$$

Add `After=other.timer` to serialize.

### OnClockChange and OnTimezoneChange

Fires when the system clock changes (NTP step, DST, timezone):

$$\Delta clock = |clock_{new} - clock_{old}|$$

Useful for recalculating schedules after DST transitions that shift wall-clock times.

---

## 8. Summary of systemd Timer Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Calendar schedule | $DOW \cap Month \cap Day \cap Time$ | Set intersection |
| Monotonic delay | $T_{ref} + delay$ | Fixed offset |
| Effective interval | $delay + T_{service}$ | Fixed-delay scheduling |
| Coalescing window | $[T, T + AccuracySec]$ | Interval |
| Random jitter | $T + U(0, delay)$ | Uniform distribution |
| Thundering herd | $(dt / jitter)^n$ | Probability |
| Missed runs | $\lfloor downtime / period \rfloor$ | Floor division |

---

*systemd timers are calendar algebra plus monotonic delays, with randomization for load spreading and persistence for reliability. They replace cron not by being simpler, but by being more mathematically complete.*
