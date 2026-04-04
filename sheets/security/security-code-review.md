# Security Code Review (Adversarial Code Auditing Methodology)

Security code review is the systematic examination of source code to identify
vulnerabilities, logic flaws, and unsafe patterns that automated tools miss.
Adversarial review assumes every input is hostile, every boundary is crossable,
and every assumption is wrong.

---

## Parser and Input Validation

```bash
# Unbounded input — search for missing size limits
grep -rn "io.ReadAll" --include="*.go"             # reads entire body
grep -rn "request.get_json()" --include="*.py"     # unbounded JSON
grep -rn "JSON.parse" --include="*.js"             # no size limit

# Missing bounds checks on array/slice access
grep -rn "\[.*\]" --include="*.go" | grep -v "range"
grep -rn "argv\[" --include="*.c"

# Regex for ReDoS (catastrophic backtracking)
# Patterns like (a+)+ or (a|b|c)* on adversarial input
# Tool: npx recheck "^(a+)+$"

# Prefer allowlist over denylist validation
# DANGEROUS: if input not in blocklist  (attackers find bypasses)
# SAFE: if not re.match(r'^[a-zA-Z0-9_-]+$', input)
```

---

## Integer Overflow and Underflow

```bash
# Go: silent wrap on overflow (no panic)
grep -rn "int32(\|int16(\|int8(" --include="*.go"  # truncation risk
# Safe check: if a > 0 && b > math.MaxInt64/a { overflow }

# C/C++: arithmetic overflow in malloc
grep -rn "malloc\|calloc" --include="*.c"
# malloc(n * sizeof(T)) overflows → tiny allocation → buffer overflow

# Python: ctypes and numpy can overflow
grep -rn "ctypes\.\|numpy\.int32" --include="*.py"

# JavaScript: precision loss above 2^53
grep -rn "parseInt\|Number(" --include="*.js"
```

---

## Race Condition Identification

### TOCTOU

```bash
# Check-then-act without synchronization
grep -rn "os.Stat.*os.Open" --include="*.go"       # file TOCTOU
grep -rn "os.path.exists.*open(" --include="*.py"
grep -rn "access(.*open(" --include="*.c"
```

### Go-Specific Races

```bash
# Run race detector
go test -race ./...

# Shared map without mutex (maps are not goroutine-safe)
grep -rn "map\[" --include="*.go" | grep -v "sync"

# Goroutine leaks — no cancellation context
grep -rn "go func" --include="*.go" | grep -v "ctx\|cancel\|Done"

# Channel deadlocks
grep -rn "make(chan" --include="*.go"                # unbuffered channels
```

### Other Languages

```bash
# Python: threading + shared state
grep -rn "threading\.\|global " --include="*.py"

# JavaScript: async TOCTOU (check-then-await-then-act)
grep -rn "async\|await" --include="*.js"

# C/C++: unprotected shared state
grep -rn "pthread\|std::thread" --include="*.c" --include="*.cpp"
```

---

## Memory Safety

```bash
# C/C++: unsafe string/memory functions
grep -rn "strcpy\|strcat\|sprintf\|gets\b" --include="*.c"
# Safe alternatives: strncpy, strncat, snprintf, fgets

# Format string vulnerabilities
grep -rn "printf(.*)" --include="*.c" | grep -v 'printf("'
# Dangerous: printf(user_input)  Safe: printf("%s", user_input)

# Go: unsafe package bypasses memory safety
grep -rn "unsafe\." --include="*.go"
grep -rn "import \"C\"" --include="*.go"             # CGo boundary

# Rust: unsafe blocks
grep -rn "unsafe" --include="*.rs"
grep -rn "std::mem::transmute\|from_raw_parts" --include="*.rs"
```

---

## Injection Patterns

### SQL Injection

```bash
# String concatenation in SQL queries
grep -rn "fmt.Sprintf.*SELECT" --include="*.go"
grep -rn "execute.*f\"\|execute.*%" --include="*.py"
grep -rn "query.*\`.*\$\{" --include="*.js"

# Safe: parameterized queries
# Go:     db.Query("SELECT * FROM users WHERE id = ?", id)
# Python: cursor.execute("SELECT * WHERE id = %s", (id,))
# JS:     connection.query("SELECT * WHERE id = ?", [id])
```

### Command Injection

```bash
# Shell execution with user input
grep -rn "exec.Command" --include="*.go"
# Dangerous: exec.Command("sh", "-c", userInput)
# Safe: exec.Command("ls", "-la", safeArg)

grep -rn "os.system\|subprocess.*shell=True" --include="*.py"
grep -rn "child_process\|exec(\|eval(" --include="*.js"
```

---

## Cryptographic Misuse

### Weak RNG

```bash
# Non-cryptographic RNG for security purposes
grep -rn "math/rand" --include="*.go"          # use crypto/rand
grep -rn "import random\b" --include="*.py"    # use secrets module
grep -rn "Math\.random" --include="*.js"       # use crypto.randomBytes
grep -rn "rand()\|srand(" --include="*.c"      # use getrandom()
```

### Hardcoded Secrets

```bash
grep -rn "password\s*=\s*[\"']" --include="*.go" --include="*.py"
grep -rn "api_key\s*=\s*[\"']" --include="*.go" --include="*.py"
grep -rn "-----BEGIN.*PRIVATE KEY-----" --include="*.go" --include="*.py"
grep -rn "AKIA[0-9A-Z]{16}" --include="*.go"   # AWS access key

# Automated scanning
gitleaks detect --source . --report-path secrets.json
```

### Timing Leaks

```bash
# Non-constant-time comparison of secrets
grep -rn "==.*hmac\|hmac.*==" --include="*.go"      # use subtle.ConstantTimeCompare
grep -rn "==.*digest\|digest.*==" --include="*.py"   # use hmac.compare_digest
grep -rn "===.*token\|token.*===" --include="*.js"   # use crypto.timingSafeEqual
```

---

## Error Handling and Info Disclosure

```bash
# Detailed errors returned to users
grep -rn "http.Error.*err\|json.*err\.Error()" --include="*.go"
# Dangerous: http.Error(w, err.Error(), 500)
# Safe: http.Error(w, "internal error", 500); log.Error(err)

grep -rn "traceback\|DEBUG\s*=\s*True" --include="*.py"
grep -rn "res\.send.*err\|res\.json.*err" --include="*.js"

# Missing resource cleanup
grep -rn "\.Open(" --include="*.go" -A 3 | grep -v "defer.*Close"
grep -rn "open(" --include="*.py" | grep -v "with "
```

---

## Authentication and Authorization

```bash
# Missing auth middleware on endpoints
grep -rn "http.HandleFunc\|mux.Handle" --include="*.go"
grep -rn "@app.route" --include="*.py"         # check for @login_required
grep -rn "app\.get\|app\.post" --include="*.js"

# IDOR — direct object reference without ownership check
grep -rn "params\[.id.\]\|req\.params\.id" --include="*.go" --include="*.js"
# Check: WHERE user_id = currentUser.id in query

# Mass assignment
grep -rn "c\.Bind\|json\.Decode" --include="*.go"
# Check: only expected fields bound, admin/role fields protected
```

---

## Dependency Audit

```bash
# Go
govulncheck ./...                          # official vulnerability scanner
go list -m -u all                          # show available updates

# Python
pip install safety && safety check
pip-audit

# JavaScript
npm audit && npm audit --fix
npx snyk test

# Rust
cargo audit && cargo outdated
```

---

## Language Checklists

### Go

```bash
# [ ] No math/rand for security (use crypto/rand)
# [ ] No fmt.Sprintf in SQL (use parameterized queries)
# [ ] All shared state protected by mutex/channel
# [ ] go test -race passes
# [ ] No unsafe package (or audited)
# [ ] Request body size limited (http.MaxBytesReader)
# [ ] HTTP timeouts set on server and client
# [ ] TLS MinVersion >= tls.VersionTLS12
# [ ] govulncheck clean
```

### Python

```bash
# [ ] No pickle.loads on untrusted data
# [ ] No eval/exec on user input
# [ ] No shell=True with user input
# [ ] No yaml.load without Loader=SafeLoader
# [ ] Django DEBUG=False in production
# [ ] CSRF protection on all forms
# [ ] Template autoescape enabled
```

### JavaScript

```bash
# [ ] No eval() or new Function() with user input
# [ ] No innerHTML with unsanitized content
# [ ] Express helmet middleware enabled
# [ ] Content-Security-Policy set
# [ ] npm audit clean
# [ ] JWT algorithm verified server-side
```

---

## Tips

- Read code from the attacker's perspective — trace every external input to its final use
- Focus on trust boundary crossings — HTTP, files, databases, environment variables
- Integer overflow in Go is silent — always check arithmetic on user-supplied sizes
- Go maps are not goroutine-safe — concurrent access without sync.Mutex is a data race
- The most dangerous injections use string concatenation for query construction
- Hardcoded secrets are found by pattern search, not just variable name search
- Error messages with err.Error() in HTTP responses leak internal architecture
- IDOR is the most common authorization flaw — verify object ownership always
- Run govulncheck/npm audit/safety check in CI/CD, not just during review
- Every Open needs a Close — defer in Go, with statement in Python

---

## See Also

- sast-dast
- threat-modeling
- cryptography

## References

- [OWASP Code Review Guide](https://owasp.org/www-project-code-review-guide/)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- [Semgrep](https://semgrep.dev/)
- [CodeQL](https://codeql.github.com/)
- [Bandit - Python Security Linter](https://bandit.readthedocs.io/)
