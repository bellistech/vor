# The Mathematics of ShellCheck -- Word Splitting and Shell Safety

> *Shell scripts are string-rewriting systems where unquoted variables expand into multiple tokens through word splitting and globbing. ShellCheck enforces quoting discipline because the cost of a single unquoted variable in the wrong context can be catastrophic.*

---

## 1. Word Splitting as Tokenization (Formal Languages)

### The Problem

When a shell expands an unquoted variable, it splits the result on characters in `$IFS` (default: space, tab, newline). A single variable can become zero, one, or many arguments. This is the root cause of SC2086.

### The Formula

Given a string $s$ with $n$ IFS characters, unquoted expansion produces at most:

$$|\text{tokens}(s)| = n + 1 - e$$

where $e$ is the number of consecutive IFS character sequences (empty tokens are collapsed). With quoting, the output is always exactly 1 token:

$$|\text{tokens}("s")| = 1 \quad \forall s$$

The danger factor of an unquoted variable:

$$D(s) = |\text{tokens}(s)| - 1$$

If $D > 0$, the command receives more arguments than intended.

### Worked Examples

Variable `file="my report.pdf"`:

**Unquoted:** `rm $file` expands to `rm my report.pdf` (3 tokens):
$$|\text{tokens}| = 1 + 1 = 2 \text{ (file tokens)} \Rightarrow \text{rm gets 2 args}$$

Result: deletes files named `my` and `report.pdf`, not `my report.pdf`.

**Quoted:** `rm "$file"` expands to `rm "my report.pdf"` (1 token).

For `files="a.txt b.txt c.txt"` (2 spaces):
$$|\text{tokens}| = 2 + 1 = 3$$

---

## 2. Globbing Expansion (Combinatorial Explosion)

### The Problem

Unquoted variables containing glob characters (`*`, `?`, `[`) are expanded against the filesystem. SC2046 catches this. The number of matches is unbounded and depends on directory contents.

### The Formula

Given a glob pattern $p$ and a directory with $N$ files, the number of matches:

$$|\text{glob}(p)| \in [0, N]$$

For pattern `*`, every file matches: $|\text{glob}(*)| = N$.

The probability that an arbitrary string triggers globbing:

$$P(\text{glob}) = 1 - \prod_{i=1}^{|s|} (1 - P(\text{glob char at } i))$$

where $P(\text{glob char})$ is the probability of a character being `*`, `?`, or `[`.

### Worked Examples

Variable `msg="Status: OK [success]"` in a directory with 500 files:

**Unquoted:** `echo $msg` triggers both word splitting (3 tokens) and glob expansion on `[success]`. If any file matches the character class `[success]` (any file starting with s, u, c, e):

$$|\text{matches}| = |\{f \in \text{dir} : f[0] \in \{s,u,c,e\}\}|$$

In a typical directory, this could be dozens of files substituted silently.

**Quoted:** `echo "$msg"` outputs the literal string. No splitting, no globbing.

---

## 3. Exit Code Propagation (Boolean Chains)

### The Problem

SC2164 flags `cd` without exit-on-failure. In shell scripts, every command returns an exit code, but the script continues by default. The probability of a silent failure causing damage depends on the command chain.

### The Formula

For a chain of $n$ commands without error checking, the probability that the script reaches command $k$ despite a failure at command $j < k$:

$$P(\text{reach } k | \text{fail at } j) = \begin{cases} 1 & \text{without set -e} \\ 0 & \text{with set -e} \end{cases}$$

The expected damage from an unguarded `cd` followed by `rm -rf`:

$$E[\text{damage}] = P(\text{cd fails}) \times S_{\text{cwd}}$$

where $S_{\text{cwd}}$ is the total size of files in the current working directory (which `rm` operates on when `cd` fails).

### Worked Examples

A deployment script:

```
cd /opt/myapp        # fails silently (dir doesn't exist)
rm -rf ./logs/*      # deletes CWD/logs/* instead!
```

If $P(\text{cd fails}) = 0.01$ (misconfigured deploy) and CWD is `/`:

$$E[\text{damage}] = 0.01 \times S_{\text{root filesystem}}$$

Even at 1% probability, the expected damage is catastrophic. This is why SC2164 exists.

With `set -euo pipefail`, the chain is:

$$P(\text{reach rm} | \text{cd fails}) = 0$$

---

## 4. Variable Masking in Subshells (SC2155)

### The Problem

`local var=$(command)` masks the exit code of `command` with the exit code of `local` (which is always 0). SC2155 flags this because error detection is silently broken.

### The Formula

The observed exit code:

$$\text{exit}_{observed} = \begin{cases} \text{exit}(\text{command}) & \text{if } \text{var}=\$(\text{command}) \\ \text{exit}(\text{local}) = 0 & \text{if local var}=\$(\text{command}) \end{cases}$$

The information loss:

$$I_{lost} = H(\text{exit}(\text{command})) - H(\text{exit}_{observed})$$

where $H$ is the entropy. When `local` masks the exit code, $H(\text{exit}_{observed}) = 0$ (always 0), so all error information is lost.

### Worked Examples

```bash
local config=$(parse_config "$file")    # exit code always 0
```

If `parse_config` fails with exit code 1:

$$\text{exit}_{observed} = 0 \quad (\text{local succeeded})$$

The script continues with an empty `$config`, potentially causing downstream errors far from the root cause.

Correct form:

```bash
local config
config=$(parse_config "$file")          # exit code preserved
```

Now: $\text{exit}_{observed} = \text{exit}(\text{parse\_config}) = 1$ and `set -e` catches it.

---

## 5. Redirect Efficiency (I/O Complexity)

### The Problem

SC2129 flags repeated redirections to the same file. Each redirection opens and closes the file descriptor, adding system call overhead. Grouped redirection opens the descriptor once.

### The Formula

System calls for $n$ individual redirections:

$$\text{syscalls}_{individual} = n \times (O + W + C) = 3n$$

where $O$ = open, $W$ = write, $C$ = close.

System calls for grouped redirection:

$$\text{syscalls}_{grouped} = O + n \times W + C = n + 2$$

### Worked Examples

Writing 100 lines individually vs. grouped:

$$\text{syscalls}_{individual} = 3 \times 100 = 300$$
$$\text{syscalls}_{grouped} = 100 + 2 = 102$$

$$\text{Reduction} = 1 - \frac{102}{300} = 66\%$$

For a log-heavy script writing 10,000 lines:

$$\text{syscalls}_{individual} = 30{,}000$$
$$\text{syscalls}_{grouped} = 10{,}002$$

Each `open` involves inode lookup, permission checks, and file descriptor allocation. At $\sim 5\mu s$ per syscall:

$$T_{individual} = 30{,}000 \times 5\mu s = 150 \text{ ms}$$
$$T_{grouped} = 10{,}002 \times 5\mu s = 50 \text{ ms}$$

---

## 6. Severity Classification (Risk Scoring)

### The Problem

ShellCheck assigns severity levels (error, warning, info, style) to each finding. These map to different levels of risk, from definite bugs to cosmetic preferences. A rational CI policy requires quantifying the risk.

### The Formula

Risk score per finding of severity $s$:

$$R(s) = P(\text{bug} | s) \times I(\text{impact} | s)$$

| Severity | $P(\text{bug})$ | $I(\text{impact})$ | $R$ |
|:---|:---:|:---:|:---:|
| error | 0.95 | 10 | 9.5 |
| warning | 0.60 | 6 | 3.6 |
| info | 0.20 | 3 | 0.6 |
| style | 0.05 | 1 | 0.05 |

Total risk for a script with $n_e$ errors, $n_w$ warnings, $n_i$ info, $n_s$ style findings:

$$R_{total} = 9.5 n_e + 3.6 n_w + 0.6 n_i + 0.05 n_s$$

### Worked Examples

Script with 2 errors (unquoted rm variables), 5 warnings, 10 info, 20 style:

$$R_{total} = 9.5(2) + 3.6(5) + 0.6(10) + 0.05(20) = 19 + 18 + 6 + 1 = 44$$

A CI threshold of $R_{max} = 20$ would block this script. Fixing just the 2 errors:

$$R_{after} = 0 + 18 + 6 + 1 = 25 \quad \text{(still above threshold)}$$

Fixing errors + warnings:

$$R_{after} = 0 + 0 + 6 + 1 = 7 \quad \text{(passes)}$$

---

## Prerequisites

- Formal language theory (tokenization, string rewriting)
- Shell expansion order (brace, tilde, parameter, command substitution, word splitting, globbing)
- Unix system calls (open, write, close, file descriptors)
- Information theory (entropy, information loss)
- Basic probability and expected value
- Boolean logic (short-circuit evaluation in shell)
