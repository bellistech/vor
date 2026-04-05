# Code Coverage (Go, JS, Theory)

Complete reference for coverage measurement — go tool cover, lcov, istanbul/c8, coverage types (line/branch/condition/path/MC/DC), CI thresholds, profile merging, and coverage-guided fuzzing.

## Go Coverage

### Basic Usage

```bash
# Show coverage percentage
go test -cover ./...

# Generate coverage profile
go test -coverprofile=coverage.out ./...

# Coverage modes
go test -covermode=set ./...       # boolean: was line executed? (default)
go test -covermode=count ./...     # how many times each line ran
go test -covermode=atomic ./...    # thread-safe count (use with -race)

# Include all packages (not just tested ones)
go test -coverpkg=./... -coverprofile=coverage.out ./...
```

### Viewing Coverage

```bash
# Per-function summary
go tool cover -func=coverage.out

# Output:
# mypackage/handler.go:15:    GetUser         100.0%
# mypackage/handler.go:35:    CreateUser      75.0%
# mypackage/handler.go:60:    DeleteUser      0.0%
# total:                      (statements)    58.3%

# HTML report (opens browser)
go tool cover -html=coverage.out

# HTML report to file
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Profiles

A coverage profile is a text file:

```
mode: set
mypackage/handler.go:15.33,18.2 1 1
mypackage/handler.go:20.33,25.2 3 1
mypackage/handler.go:27.33,30.2 2 0
```

Format: `file:startline.startcol,endline.endcol numberOfStatements count`

### Merging Coverage Profiles

When you have multiple test runs (e.g., unit + integration), merge their profiles:

```bash
# Install gocovmerge
go install github.com/wadey/gocovmerge@latest

# Generate separate profiles
go test -coverprofile=unit.out ./...
go test -tags=integration -coverprofile=integration.out ./...

# Merge
gocovmerge unit.out integration.out > merged.out

# View merged coverage
go tool cover -html=merged.out
```

Alternative with `gotestsum`:

```bash
# Using -coverprofile with gotestsum
gotestsum -- -coverprofile=coverage.out ./...
```

### Excluding Files from Coverage

```go
//go:build !coverage

// This file is excluded when running with coverage build tag
package mypackage
```

Or exclude generated files by convention:

```bash
# Filter coverage profile
grep -v "_generated.go" coverage.out > coverage_filtered.out
grep -v "mock_" coverage.out > coverage_filtered.out
```

### Per-Package Coverage

```bash
# Coverage per package
go test -cover ./... 2>&1 | column -t

# Output:
# ok   mymodule/pkg/handler   0.5s   coverage: 85.2% of statements
# ok   mymodule/pkg/store      0.3s   coverage: 72.1% of statements
# ok   mymodule/pkg/auth       0.2s   coverage: 91.5% of statements
```

### coverpkg for Cross-Package Coverage

```bash
# Without -coverpkg: only counts coverage within each package's own tests
go test -coverprofile=c.out ./pkg/handler/...
# Shows: 85% of handler package

# With -coverpkg: counts coverage in ALL specified packages
go test -coverpkg=./... -coverprofile=c.out ./pkg/handler/...
# Shows: 85% of handler, plus coverage of store/auth exercised by handler tests
```

## JavaScript Coverage

### istanbul / c8 (Node.js)

```bash
# c8 uses V8's built-in coverage (faster, more accurate)
npx c8 node test.js
npx c8 npm test
npx c8 --reporter=html --reporter=text npm test

# istanbul (nyc) — instrumentation-based
npx nyc npm test
npx nyc --reporter=html --reporter=text-summary npm test
```

### Configuration

```json
// package.json
{
  "nyc": {
    "check-coverage": true,
    "lines": 80,
    "branches": 70,
    "functions": 80,
    "statements": 80,
    "include": ["src/**/*.js"],
    "exclude": ["**/*.test.js", "**/__mocks__/**"],
    "reporter": ["text", "html", "lcov"]
  }
}
```

### c8 Configuration

```json
// .c8rc.json
{
  "check-coverage": true,
  "lines": 80,
  "branches": 70,
  "functions": 80,
  "all": true,
  "include": ["src/**"],
  "exclude": ["**/*.test.*", "**/__tests__/**"],
  "reporter": ["text", "html", "lcov"]
}
```

### Jest Coverage

```bash
npx jest --coverage
npx jest --coverage --coverageReporters=text --coverageReporters=lcov
```

```json
// jest.config.js
{
  "collectCoverage": true,
  "coverageThreshold": {
    "global": {
      "branches": 70,
      "functions": 80,
      "lines": 80,
      "statements": 80
    },
    "./src/critical/": {
      "branches": 95,
      "functions": 95,
      "lines": 95
    }
  }
}
```

## lcov / genhtml

### Generate lcov Format

```bash
# From Go coverage
go test -coverprofile=coverage.out ./...
# Convert to lcov
go install github.com/jandelgado/gcov2lcov@latest
gcov2lcov -infile coverage.out -outfile coverage.lcov

# Generate HTML from lcov
genhtml coverage.lcov --output-directory coverage-html
```

### lcov Operations

```bash
# Merge coverage files
lcov -a coverage1.lcov -a coverage2.lcov -o merged.lcov

# Extract specific files
lcov --extract merged.lcov '*/src/*' -o src-only.lcov

# Remove files
lcov --remove merged.lcov '*/vendor/*' '*/test/*' -o filtered.lcov

# Summary
lcov --summary merged.lcov
```

## Coverage Types

### Line Coverage (Statement Coverage)

Was each line of code executed?

```go
func Abs(x int) int {
    if x < 0 {        // line 1 ✓
        return -x      // line 2 — need negative input
    }
    return x           // line 3 — need non-negative input
}
// 100% line coverage: test with x=-1 and x=1
```

### Branch Coverage (Decision Coverage)

Was each branch of each conditional taken?

```go
func Classify(a, b bool) string {
    if a && b {          // branch: true, false
        return "both"    // requires a=true, b=true
    }
    return "not both"    // requires a=false OR b=false
}
// 100% line coverage with: (true, true) and (false, false)
// but NOT 100% branch coverage — missing (true, false) and (false, true)
```

### Condition Coverage

Was each boolean sub-expression evaluated to both true and false?

```go
if a && b {  // conditions: a=true, a=false, b=true, b=false
```

Requires testing: a=true, a=false, b=true, b=false (4 conditions, minimum 2 test cases).

### Path Coverage

Was every possible path through the function executed?

```go
func Example(a, b bool) {
    if a {
        doA()       // path 1: a=true
    }
    if b {
        doB()       // path 2: b=true
    }
}
// Paths: (!a,!b), (!a,b), (a,!b), (a,b) — 4 paths
// With n independent conditionals: 2^n paths
```

### MC/DC (Modified Condition/Decision Coverage)

Each condition independently affects the decision outcome. Required by DO-178C (avionics).

```go
if a && b {
```

MC/DC test cases:
| a | b | result | demonstrates |
|---|---|--------|-------------|
| T | T | T | baseline |
| F | T | F | a independently affects result |
| T | F | F | b independently affects result |

Only 3 tests needed (not 4 for full condition coverage).

### Coverage Hierarchy

```
Path Coverage ⊇ MC/DC ⊇ Branch Coverage ⊇ Condition Coverage ⊇ Line Coverage
(strongest)                                                      (weakest)
```

## CI Coverage Gates

### GitHub Actions

```yaml
- name: Run tests with coverage
  run: go test -coverprofile=coverage.out -covermode=atomic ./...

- name: Check coverage threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    echo "Total coverage: ${COVERAGE}%"
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage ${COVERAGE}% is below threshold 80%"
      exit 1
    fi
```

### Makefile Gate

```makefile
COVERAGE_THRESHOLD := 80

.PHONY: coverage-check
coverage-check:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%'); \
	echo "Coverage: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "FAIL: Coverage $${COVERAGE}% below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi
```

### Ratcheting (Never-Decrease)

```bash
# Save current coverage
go tool cover -func=coverage.out | grep total | awk '{print $3}' > .coverage-baseline

# In CI, compare against baseline
CURRENT=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
BASELINE=$(cat .coverage-baseline | tr -d '%')
if (( $(echo "$CURRENT < $BASELINE" | bc -l) )); then
    echo "Coverage decreased from ${BASELINE}% to ${CURRENT}%"
    exit 1
fi
echo "$CURRENT%" > .coverage-baseline
```

## Coverage-Guided Fuzzing Connection

### Go Native Fuzzing

Go's fuzzer uses coverage instrumentation to guide input generation:

```go
func FuzzJSON(f *testing.F) {
    // Seed corpus
    f.Add([]byte(`{"key": "value"}`))
    f.Add([]byte(`[]`))
    f.Add([]byte(`null`))

    f.Fuzz(func(t *testing.T, data []byte) {
        var v interface{}
        if err := json.Unmarshal(data, &v); err != nil {
            return // invalid input, skip
        }
        // Re-encode and check roundtrip
        encoded, err := json.Marshal(v)
        if err != nil {
            t.Fatalf("Marshal failed: %v", err)
        }
        var v2 interface{}
        if err := json.Unmarshal(encoded, &v2); err != nil {
            t.Fatalf("roundtrip Unmarshal failed: %v", err)
        }
    })
}
```

```bash
go test -fuzz=FuzzJSON -fuzztime=30s
```

The fuzzer maintains a corpus in `testdata/fuzz/` and discovers inputs that exercise new code paths.

## Tips

- `-covermode=atomic` is required when using `-race`; `set` is fastest for simple coverage
- Use `-coverpkg=./...` to get accurate cross-package coverage (tests in package A covering code in package B)
- Coverage thresholds should ratchet (never decrease) rather than be set to an arbitrary target
- 100% line coverage is achievable but does not guarantee correctness — branch and MC/DC coverage are stronger
- Merge unit and integration coverage profiles for a complete picture
- Exclude generated code, mocks, and test helpers from coverage metrics
- Focus coverage efforts on critical paths (error handling, security, data validation)
- Coverage-guided fuzzing combines coverage measurement with test generation
- lcov format is the universal exchange format — most tools can produce and consume it
- Branch coverage is typically 15-25% lower than line coverage for the same test suite

## See Also

- `sheets/testing/go-testing.md` — Go test flags including coverage flags
- `sheets/testing/property-based-testing.md` — generative testing for higher coverage
- `detail/testing/coverage.md` — mathematical analysis of coverage metrics

## References

- https://go.dev/blog/cover — Go coverage design
- https://pkg.go.dev/cmd/cover — go tool cover documentation
- https://github.com/istanbuljs/nyc — istanbul/nyc
- https://github.com/nicholasgasior/gocovmerge — coverage profile merging
- https://en.wikipedia.org/wiki/Modified_condition/decision_coverage — MC/DC specification
