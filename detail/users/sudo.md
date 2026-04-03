# The Mathematics of sudo — Privilege Escalation, Credential Caching & Policy Evaluation

> *sudo is a policy engine with a caching layer. Every invocation is a decision: authenticate, check policy, cache credentials, and audit. The mathematics of sudo are the mathematics of access control, timeout windows, and risk quantification.*

---

## 1. Policy Evaluation — The sudoers Grammar

### Rule Structure

A sudoers rule is a tuple:

$$rule = (user\_spec, host\_spec, runas\_spec, command\_spec, options)$$

$$allowed(user, host, command) = \exists r \in rules : matches(r, user, host, command)$$

### Rule Matching Order

sudoers rules are evaluated **last match wins**:

$$effective\_rule = rules[max\{i : matches(rules[i], user, host, cmd)\}]$$

This means later rules override earlier ones — a common source of misconfiguration.

### Command Matching Complexity

For $R$ rules, each with a command spec potentially containing wildcards:

$$T_{policy\_check} = O(R \times L_{cmd})$$

Where $L_{cmd}$ is the command path length. Glob matching is $O(L)$ per pattern.

---

## 2. Credential Caching — The Timeout Window

### Timestamp Model

After successful authentication, sudo caches the credential:

$$valid\_until = T_{auth} + timestamp\_timeout$$

Default: `timestamp_timeout = 5` minutes.

$$cached = \begin{cases} true & \text{if } now < valid\_until \\ false & \text{otherwise (re-authenticate)} \end{cases}$$

### Cache Granularity

| Mode | Scope | Security |
|:---|:---|:---|
| Per-tty (default) | Each terminal independently | Higher |
| Per-user | Any terminal same user | Convenience |
| Global | Any user (not recommended) | Lowest |

### Exposure Window

$$T_{exposure} = timestamp\_timeout$$

During this window, any command can be run as root without re-authentication:

$$commands\_executable = \infty \text{ (within the timeout window)}$$

$$risk = P(unauthorized\_access) \times T_{exposure}$$

### Setting `timestamp_timeout = 0`

$$valid\_until = T_{auth} + 0 = T_{auth} \text{ (expired immediately)}$$

Every sudo command requires authentication. Maximum security, minimum convenience.

### Setting `timestamp_timeout = -1`

$$valid\_until = \infty \text{ (never expires)}$$

Authenticate once per session. Minimum security.

---

## 3. Authentication — Password Attempts and Lockout

### Attempt Model

$$max\_attempts = passwd\_tries = 3 \text{ (default)}$$

After $passwd\_tries$ failures:

$$locked\_out\_for = T_{cooldown} \text{ (typically: sudo exits, must re-invoke)}$$

### Brute Force Protection

$$P(guess) = \frac{1}{|password\_space|}$$

$$P(crack\_in\_n) = 1 - (1 - P(guess))^n$$

With 3 attempts and a reasonable password (entropy 40+ bits):

$$P(crack\_in\_3) = 1 - (1 - 2^{-40})^3 \approx 3 \times 10^{-12}$$

### Timing Side Channel

sudo introduces a fixed delay after failed authentication:

$$T_{delay} = 2 \text{ seconds (typical)}$$

$$max\_attempts\_per\_minute = \frac{60}{T_{delay} + T_{input}} \approx \frac{60}{2 + 3} = 12$$

---

## 4. NOPASSWD — Risk Quantification

### The Tradeoff

`user ALL=(ALL) NOPASSWD: ALL` eliminates authentication:

$$authentication\_factor = 0$$

$$risk = \begin{cases} P(account\_compromise) \times impact & \text{with NOPASSWD} \\ P(account\_compromise) \times P(password\_known) \times impact & \text{with password} \end{cases}$$

### Selective NOPASSWD

```
user ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
```

$$risk = P(compromise) \times impact(systemctl\_restart\_nginx)$$

Much lower than full NOPASSWD: the blast radius is bounded to one command.

### Commands Worth NOPASSWD

$$NOPASSWD\_safe \iff impact(command) \leq threshold$$

| Command | Impact | NOPASSWD Safe? |
|:---|:---|:---:|
| `systemctl status` | Read-only | Yes |
| `systemctl restart service` | Service disruption | Probably |
| `/bin/bash` | Full root shell | **No** |
| `cat /var/log/*` | Log access | Yes |
| `dd if=/dev/sda` | Read all disks | **No** |

---

## 5. Logging and Audit Trail

### Log Entry Structure

Each sudo invocation logs:

$$log = (timestamp, user, tty, cwd, command, runas\_user, success/fail)$$

### Log Volume

$$log\_rate = sudo\_invocations\_per\_day \times avg\_log\_line$$

Average log line: ~200 bytes.

| Invocations/Day | Daily Log Size |
|:---:|:---:|
| 100 | 20 KB |
| 1,000 | 200 KB |
| 10,000 | 2 MB |

### I/O Logging

With `log_input` and `log_output`:

$$storage = \sum_{session} (input\_bytes + output\_bytes)$$

A 1-hour interactive root session with verbose output:

$$storage \approx 1-50 \text{ MB}$$

---

## 6. sudoers Complexity — Alias Expansion

### Alias Types

$$aliases = User\_Alias \cup Host\_Alias \cup Runas\_Alias \cup Cmnd\_Alias$$

### Expansion Cost

An alias can reference other aliases (but not recursively):

$$expanded(alias) = \bigcup_{m \in members(alias)} \begin{cases} \{m\} & \text{if } m \text{ is literal} \\ expanded(m) & \text{if } m \text{ is alias name} \end{cases}$$

### Maximum Depth

Aliases expand to depth 1 (no recursive aliases). Total expansion:

$$|expanded| \leq \sum_{alias} |members(alias)|$$

### sudoers Syntax Validation

`visudo` parses the entire file:

$$T_{parse} = O(L) \text{ where } L = \text{total lines in sudoers}$$

A syntax error anywhere → the entire file is rejected (fail-safe).

---

## 7. Environment Security — Variable Filtering

### Default Environment Reset

sudo resets the environment except for whitelisted variables:

$$env_{sudo} = \{v : v \in env\_keep\} \cup defaults$$

### Dangerous Variables

| Variable | Risk | sudo Behavior |
|:---|:---|:---|
| PATH | Command hijacking | Reset to secure_path |
| LD_PRELOAD | Library injection | Stripped |
| LD_LIBRARY_PATH | Library hijacking | Stripped |
| PYTHONPATH | Module injection | Stripped |
| PERL5LIB | Module injection | Stripped |
| HOME | Config file hijacking | Set to target user |

### env_keep Size Impact

$$|env_{sudo}| = |env\_keep| + |defaults|$$

$$P(env\_attack) \propto |env\_keep|$$

Larger `env_keep` → larger attack surface. Keep it minimal.

### secure_path

$$PATH_{sudo} = \text{secure\_path} = /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin$$

This prevents `PATH=/tmp/evil:$PATH` attacks.

---

## 8. Summary of sudo Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Policy check | Last matching rule wins | Sequential evaluation |
| Credential cache | $valid = now < T_{auth} + timeout$ | Time window |
| Brute force | $P = 1 - (1 - 2^{-entropy})^n$ | Probability |
| Auth rate | $60 / (T_{delay} + T_{input})$ | Rate limiting |
| Log volume | $invocations \times 200$ bytes | Linear |
| Alias expansion | $\sum \|members\|$ | Set union |
| Risk (NOPASSWD) | $P(compromise) \times impact$ | Risk product |

---

*sudo is a policy-cached authentication gateway. Every invocation is a triple check — who are you, what do you want, and did we check your password recently enough? The timestamp cache trades security for convenience, and the timeout value is the quantified measure of that tradeoff.*
