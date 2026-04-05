# WAVE 7 BATTLE PLAN — 6 Phases, 85+ Steps

**Date**: 2026-04-04
**Sprint**: Wave 7 — Captain+Micromanager Gap Fill + Multi-Language Coding Problems
**Prerequisite**: Wave 6 committed (a1f8ffb), formatting fixes committed (a129433), build green
**Target**: 17 new sheet+detail pairs across 5 categories + 56 multi-language coding problems across 4 languages
**Estimated Duration**: 2-3 hours across 3 parallel agents
**Agent Strategy**: 3 parallel agents — Agent Alpha (sheets batch 1), Agent Bravo (sheets batch 2), Agent Charlie (coding problems)
**Commit Cadence**: Per-agent on completion, then coordinator verification + final commit
**Stuck Protocol**: Skip after 3x time estimate or 2 failed attempts

---

## LEGEND

[B] = Bash command (run directly)
[V] = Verification step (MUST pass before proceeding)
[D] = Debug step (only if prior step fails)
[W] = Write/create file
[R] = Read/inspect file
[P] = Parallelizable with other marked steps
[C] = Commit checkpoint

---

## PHASE 0: INTELLIGENCE GATHERING (Steps 1-8)

**Goal**: Verify current state, read format templates, confirm no conflicts
**Prerequisite**: None
**Time**: 5 minutes
**Agent**: Coordinator

- [ ] **Step 1** [B]: Verify build is green
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet && export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH" && go build ./...
  ```

- [ ] **Step 2** [V]: Build succeeds with exit 0. If fail → fix before proceeding.

- [ ] **Step 3** [B]: Confirm current inventory count
  ```bash
  echo "Sheets: $(find sheets -name '*.md' | wc -l)" && echo "Details: $(find detail -name '*.md' | wc -l)" && echo "Categories: $(ls -d sheets/*/ | wc -l)"
  ```

- [ ] **Step 4** [V]: Sheets=610, Details=610, Categories=56. Record actuals.

- [ ] **Step 5** [R]: Read format templates — one sheet and one detail for reference
  - Read `sheets/testing/pytest.md` (lines 1-30) for sheet format
  - Read `detail/testing/pytest.md` (lines 1-50) for detail format

- [ ] **Step 6** [R]: Read coding problem format
  - Read `~/tmp/learning/extra/coding-questions-python/001 Two Sum.py`
  - Read `~/tmp/learning/extra/coding-questions-python/020 Valid Parentheses.py`

- [ ] **Step 7** [V]: Format patterns confirmed. Sheet = `# Title (Subtitle)` + one-liner + `## Sections` + `## Tips` + `## See Also` + `## References`. Detail = `# The Mathematics of X — Y` + blockquote + numbered `## N. Section (Domain)` + LaTeX + `## Prerequisites` + `## Complexity` table.

- [ ] **Step 8** [V]: **PHASE 0 EXIT GATE** — Build green, formats confirmed, inventory baseline recorded.

---

## PHASE 1: AGENT ALPHA — SHEETS BATCH 1 (Steps 9-28)

**Goal**: Write 9 sheet+detail pairs: testing category (7) + auth/oidc (1) + quality/twelve-factor (1)
**Prerequisite**: Phase 0 complete
**Time**: 30-45 minutes
**Agent**: Agent Alpha [P]

### New directories (if needed)

- [ ] **Step 9** [B]: Ensure category directories exist
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  # testing/ and auth/ already exist, quality/ already exists
  ls -d sheets/testing/ sheets/auth/ sheets/quality/ detail/testing/ detail/auth/ detail/quality/
  ```

### Testing Category (7 topics)

- [ ] **Step 10** [W][P]: Write `sheets/testing/go-testing.md` (~300-400 lines)
  - Content: Go testing stdlib, table-driven tests, subtests, testify, httptest, benchmark funcs, test flags (-v, -run, -count, -race, -cover, -short), TestMain, golden files, testdata/, build tags
  - Format: `# go-testing (Go Testing Stdlib)` + one-liner + sections for: Running Tests, Table-Driven Tests, Subtests, HTTP Testing, TestMain, Test Helpers, Build Tags, Coverage, Tips, See Also, References

- [ ] **Step 11** [W][P]: Write `detail/testing/go-testing.md` (~200-250 lines)
  - Content: Mathematics of table-driven test coverage, combinatorial explosion in parameter spaces, coverage metrics (statement vs branch vs path), statistical confidence from `-count=N`, race detector probability
  - Format: `# The Mathematics of Go Testing — Coverage and Confidence`

- [ ] **Step 12** [W][P]: Write `sheets/testing/mocking.md` (~300-350 lines)
  - Content: gomock, testify/mock, mockery, interface-based mocking in Go, Python unittest.mock, Jest mocking, spy vs stub vs mock vs fake, dependency injection patterns
  - See Also: go-testing, pytest, jest, integration-testing

- [ ] **Step 13** [W][P]: Write `detail/testing/mocking.md` (~200 lines)
  - Content: Test double taxonomy (Meszaros), coupling metrics, mock vs real — when mocks lie, contract testing mathematics

- [ ] **Step 14** [W][P]: Write `sheets/testing/benchmarking.md` (~300 lines)
  - Content: Go benchmarks (testing.B), b.N, b.ResetTimer, b.ReportAllocs, benchstat, criterion (Rust), hyperfine, pprof integration, benchmark patterns
  - See Also: go-testing, perf, flamegraph

- [ ] **Step 15** [W][P]: Write `detail/testing/benchmarking.md` (~200 lines)
  - Content: Statistics of benchmarking — variance, confidence intervals, t-tests for comparison, Amdahl's law, Little's law, benchstat statistical methodology

- [ ] **Step 16** [W][P]: Write `sheets/testing/integration-testing.md` (~300 lines)
  - Content: testcontainers patterns, database integration tests, API integration tests, docker-compose for tests, test isolation, cleanup patterns, Go TestMain with setup/teardown
  - See Also: testcontainers, go-testing, docker-compose, mocking

- [ ] **Step 17** [W][P]: Write `detail/testing/integration-testing.md` (~200 lines)
  - Content: Test pyramid mathematics, cost/confidence tradeoffs, blast radius, test isolation invariants, state space reduction

- [ ] **Step 18** [W][P]: Write `sheets/testing/property-based-testing.md` (~280 lines)
  - Content: rapid (Go), hypothesis (Python), QuickCheck concepts, generators, shrinking, stateful testing, example-based vs property-based, common properties (roundtrip, idempotent, invariant, oracle)
  - See Also: go-testing, pytest, fuzzing

- [ ] **Step 19** [W][P]: Write `detail/testing/property-based-testing.md` (~200 lines)
  - Content: Search space sampling theory, shrinking algorithms, probability of finding bugs vs number of cases, birthday paradox in input generation

- [ ] **Step 20** [W][P]: Write `sheets/testing/coverage.md` (~280 lines)
  - Content: go tool cover, lcov, istanbul/c8, gcov, coverage types (line/branch/condition/path/MC/DC), coverage thresholds, coverage-guided fuzzing, merging coverage, CI gates
  - See Also: go-testing, pytest, fuzzing

- [ ] **Step 21** [W][P]: Write `detail/testing/coverage.md` (~200 lines)
  - Content: Coverage metrics mathematics, cyclomatic complexity vs paths, MC/DC (modified condition/decision coverage), theoretical limits of coverage

- [ ] **Step 22** [W][P]: Write `sheets/testing/chaos-engineering.md` (~320 lines)
  - Content: Principles of chaos, Litmus, chaos-mesh, toxiproxy, tc netem, steady state hypothesis, blast radius control, gamedays, Netflix Simian Army history, chaos in Kubernetes
  - See Also: integration-testing, kubernetes, sre-fundamentals

- [ ] **Step 23** [W][P]: Write `detail/testing/chaos-engineering.md` (~220 lines)
  - Content: Steady state deviation metrics, MTTR modeling, failure injection probability, cascading failure graphs, reliability mathematics (nines), queuing theory under fault

### Auth Category (1 topic)

- [ ] **Step 24** [W][P]: Write `sheets/auth/oidc.md` (~300 lines)
  - Content: OpenID Connect flows (authorization code, implicit, hybrid, client credentials), ID tokens, UserInfo endpoint, claims, scopes (openid profile email), discovery (.well-known), PKCE, token validation, provider setup (Keycloak, Auth0, Okta)
  - See Also: oauth, saml, jwt, tls

- [ ] **Step 25** [W][P]: Write `detail/auth/oidc.md` (~200 lines)
  - Content: JWT mathematics (RS256/ES256 signature verification), token lifetime optimization, nonce entropy, PKCE code_verifier entropy requirements

### Quality Category (1 topic)

- [ ] **Step 26** [W][P]: Write `sheets/quality/twelve-factor.md` (~300 lines)
  - Content: All 12 factors with examples (codebase, dependencies, config, backing services, build/release/run, processes, port binding, concurrency, disposability, dev/prod parity, logs, admin processes), modern interpretations, cloud-native mapping
  - See Also: docker, kubernetes, sre-fundamentals, ci-cd

- [ ] **Step 27** [W][P]: Write `detail/quality/twelve-factor.md` (~200 lines)
  - Content: Process algebra for factor VI (stateless processes), horizontal scaling mathematics, config entropy, deployment pipeline DAG modeling

- [ ] **Step 28** [V]: **PHASE 1 EXIT GATE** — 18 files created (9 sheets + 9 details). All have correct H1 format, ## See Also, ## References (sheets), ## Prerequisites, ## Complexity (details). No unclosed code fences.
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  for f in sheets/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md sheets/auth/oidc.md sheets/quality/twelve-factor.md; do
    [ -f "$f" ] && echo "OK: $f" || echo "MISSING: $f"
  done
  for f in detail/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md detail/auth/oidc.md detail/quality/twelve-factor.md; do
    [ -f "$f" ] && echo "OK: $f" || echo "MISSING: $f"
  done
  # Verify no unclosed code fences
  for f in sheets/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md sheets/auth/oidc.md sheets/quality/twelve-factor.md detail/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md detail/auth/oidc.md detail/quality/twelve-factor.md; do
    count=$(grep -c '```' "$f" 2>/dev/null || echo 0)
    remainder=$((count % 2))
    [ "$remainder" -ne 0 ] && echo "ODD FENCES: $f ($count)" || true
  done
  ```

---

## PHASE 2: AGENT BRAVO — SHEETS BATCH 2 (Steps 29-48)

**Goal**: Write 8 sheet+detail pairs: patterns (5) + quality/sre-fundamentals (1) + api/api-design (1) + performance/caching-patterns (1)
**Prerequisite**: Phase 0 complete
**Time**: 30-45 minutes
**Agent**: Agent Bravo [P]

### New directories

- [ ] **Step 29** [B]: Create patterns category directories
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  mkdir -p sheets/patterns detail/patterns
  ```

### Patterns Category (5 topics)

- [ ] **Step 30** [W][P]: Write `sheets/patterns/concurrency-patterns.md` (~350 lines)
  - Content: Go concurrency (goroutines, channels, select, sync.Mutex, sync.WaitGroup, errgroup), fan-out/fan-in, pipeline, worker pool, semaphore, context cancellation, Rust (tokio, async/await, Arc<Mutex>), Python (asyncio, threading, multiprocessing)
  - See Also: go-testing, design-patterns, channels

- [ ] **Step 31** [W][P]: Write `detail/patterns/concurrency-patterns.md` (~220 lines)
  - Content: CSP (Communicating Sequential Processes) formal model, happens-before relations, Lamport clocks, dining philosophers, ABA problem, lock-free data structures mathematics

- [ ] **Step 32** [W][P]: Write `sheets/patterns/design-patterns.md` (~380 lines)
  - Content: GoF patterns organized by type (creational: factory, builder, singleton; structural: adapter, decorator, proxy, facade; behavioral: strategy, observer, command, iterator, state), SOLID principles, Go-idiomatic patterns (functional options, interface embedding)
  - See Also: concurrency-patterns, microservices-patterns

- [ ] **Step 33** [W][P]: Write `detail/patterns/design-patterns.md` (~220 lines)
  - Content: Pattern composition algebra, coupling/cohesion metrics, dependency inversion graph theory, cyclomatic complexity reduction through patterns

- [ ] **Step 34** [W][P]: Write `sheets/patterns/microservices-patterns.md` (~350 lines)
  - Content: Circuit breaker, bulkhead, sidecar, ambassador, saga (orchestration vs choreography), API gateway, service discovery, health checks, retry with exponential backoff, deadline propagation, distributed tracing context
  - See Also: design-patterns, event-driven-architecture, service-mesh, grpc

- [ ] **Step 35** [W][P]: Write `detail/patterns/microservices-patterns.md` (~220 lines)
  - Content: Circuit breaker state machine, exponential backoff mathematics (E[retries]), saga rollback DAG, CAP theorem proof sketch, Brewer's conjecture

- [ ] **Step 36** [W][P]: Write `sheets/patterns/distributed-systems.md` (~380 lines)
  - Content: CAP theorem, PACELC, consistency models (strong, eventual, causal), consensus (Raft, Paxos overview), vector clocks, CRDTs, consistent hashing, leader election, split-brain, quorum (R+W>N), replication strategies, partition tolerance
  - See Also: microservices-patterns, etcd, cassandra, cockroachdb

- [ ] **Step 37** [W][P]: Write `detail/patterns/distributed-systems.md` (~250 lines)
  - Content: FLP impossibility theorem, Raft log replication mathematics, consistent hashing ring analysis, CRDT merge lattice theory, Byzantine fault tolerance (3f+1)

- [ ] **Step 38** [W][P]: Write `sheets/patterns/event-driven-architecture.md` (~320 lines)
  - Content: Event sourcing, CQRS, pub/sub patterns, event schemas, exactly-once semantics, idempotency keys, dead letter queues, event replay, outbox pattern, change data capture (CDC), event versioning
  - See Also: kafka, nats, rabbitmq, microservices-patterns, distributed-systems

- [ ] **Step 39** [W][P]: Write `detail/patterns/event-driven-architecture.md` (~220 lines)
  - Content: Event ordering guarantees (total vs partial order), Lamport timestamps, idempotency mathematics, stream processing windowing functions, exactly-once vs at-least-once probability analysis

### Quality Category (1 topic)

- [ ] **Step 40** [W][P]: Write `sheets/quality/sre-fundamentals.md` (~350 lines)
  - Content: SLIs/SLOs/SLAs, error budgets, toil measurement and reduction, incident management lifecycle, blameless postmortems, on-call best practices, runbook structure, capacity planning, change management, reliability vs velocity tradeoff
  - See Also: twelve-factor, chaos-engineering, prometheus, alertmanager

- [ ] **Step 41** [W][P]: Write `detail/quality/sre-fundamentals.md` (~220 lines)
  - Content: Nines mathematics (99.9% = 8.76h/year), error budget burn rate, SLO algebra (composite SLO from dependent services), queuing theory (Little's law, M/M/1), MTTR/MTTF/MTBF relationships

### API Category (1 topic)

- [ ] **Step 42** [W][P]: Write `sheets/api/api-design.md` (~320 lines)
  - Content: RESTful design principles, versioning strategies (URL, header, query), pagination (cursor vs offset), idempotency, rate limiting (token bucket, sliding window), error response format (RFC 7807), HATEOAS, API lifecycle, deprecation, OpenAPI-first design
  - See Also: rest-api, graphql, openapi, grpc, webhook

- [ ] **Step 43** [W][P]: Write `detail/api/api-design.md` (~200 lines)
  - Content: Token bucket mathematics, rate limiting fairness analysis, pagination consistency under concurrent writes, API versioning graph (compatibility matrices)

### Performance Category (1 topic)

- [ ] **Step 44** [W][P]: Write `sheets/performance/caching-patterns.md` (~320 lines)
  - Content: Cache-aside (lazy loading), write-through, write-behind, read-through, cache invalidation strategies (TTL, event-based, versioned keys), cache stampede prevention (singleflight, probabilistic early expiration), multi-tier caching (L1 local, L2 distributed), CDN caching, HTTP cache headers
  - See Also: redis, memcached, cdn, http

- [ ] **Step 45** [W][P]: Write `detail/performance/caching-patterns.md` (~200 lines)
  - Content: Cache hit ratio mathematics, LRU analysis, working set theory, Zipf distribution in cache access patterns, probabilistic early expiration (XFetch), Bloom filter for negative caching

- [ ] **Step 46** [V]: All 16 files created (8 sheets + 8 details).
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  for f in sheets/patterns/{concurrency-patterns,design-patterns,microservices-patterns,distributed-systems,event-driven-architecture}.md sheets/quality/sre-fundamentals.md sheets/api/api-design.md sheets/performance/caching-patterns.md; do
    [ -f "$f" ] && echo "OK: $f" || echo "MISSING: $f"
  done
  for f in detail/patterns/{concurrency-patterns,design-patterns,microservices-patterns,distributed-systems,event-driven-architecture}.md detail/quality/sre-fundamentals.md detail/api/api-design.md detail/performance/caching-patterns.md; do
    [ -f "$f" ] && echo "OK: $f" || echo "MISSING: $f"
  done
  ```

- [ ] **Step 47** [V]: No unclosed code fences in any Bravo file.
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  for f in sheets/patterns/*.md detail/patterns/*.md sheets/quality/sre-fundamentals.md detail/quality/sre-fundamentals.md sheets/api/api-design.md detail/api/api-design.md sheets/performance/caching-patterns.md detail/performance/caching-patterns.md; do
    count=$(grep -c '```' "$f" 2>/dev/null || echo 0)
    remainder=$((count % 2))
    [ "$remainder" -ne 0 ] && echo "ODD FENCES: $f ($count)" || true
  done
  ```

- [ ] **Step 48** [V]: **PHASE 2 EXIT GATE** — 16 files present, 0 unclosed fences.

---

## PHASE 3: AGENT CHARLIE — MULTI-LANGUAGE CODING PROBLEMS (Steps 49-72)

**Goal**: Create 56 LeetCode-style coding problems (14 problems x 4 languages: Go, Rust, Python, TypeScript)
**Prerequisite**: Phase 0 complete
**Time**: 40-50 minutes
**Agent**: Agent Charlie [P]

### Directory Structure

- [ ] **Step 49** [B]: Create language directories
  ```bash
  mkdir -p ~/tmp/learning/extra/coding-questions-go
  mkdir -p ~/tmp/learning/extra/coding-questions-rust
  mkdir -p ~/tmp/learning/extra/coding-questions-python-new
  mkdir -p ~/tmp/learning/extra/coding-questions-typescript
  ```

### Problem Set (14 problems, each in 4 languages = 56 files)

Each problem file follows the existing format:
- Docstring/comment block with problem description, constraints, examples, hints
- Solution class/struct with idiomatic implementation
- Main/test block demonstrating usage with assertions

**Category: Arrays/Strings (3 problems)**

- [ ] **Step 50** [W][P]: Problem 001 — **Sliding Window Maximum**
  - Description: Given array and window size k, return max of each sliding window
  - Files: `001_sliding_window_maximum.{go,rs,py,ts}` in respective dirs
  - Go: use container/heap or deque
  - Rust: use VecDeque
  - Python: collections.deque
  - TypeScript: array-based deque
  - Complexity: O(n) time, O(k) space

- [ ] **Step 51** [W][P]: Problem 002 — **Group Anagrams**
  - Description: Group strings that are anagrams of each other
  - Files: `002_group_anagrams.{go,rs,py,ts}`
  - Key: sorted string as hash key
  - Complexity: O(n * k log k) time

- [ ] **Step 52** [W][P]: Problem 003 — **Longest Consecutive Sequence**
  - Description: Find length of longest consecutive elements sequence in unsorted array
  - Files: `003_longest_consecutive_sequence.{go,rs,py,ts}`
  - Key: HashSet, only start from sequence beginnings
  - Complexity: O(n) time

**Category: Linked Lists (2 problems)**

- [ ] **Step 53** [W][P]: Problem 004 — **Merge K Sorted Lists**
  - Description: Merge k sorted linked lists into one sorted list
  - Files: `004_merge_k_sorted_lists.{go,rs,py,ts}`
  - Key: min-heap / priority queue
  - Complexity: O(N log k) time

- [ ] **Step 54** [W][P]: Problem 005 — **LRU Cache**
  - Description: Design an LRU cache with get and put in O(1)
  - Files: `005_lru_cache.{go,rs,py,ts}`
  - Key: HashMap + doubly linked list
  - Complexity: O(1) get/put

**Category: Trees/Graphs (2 problems)**

- [ ] **Step 55** [W][P]: Problem 006 — **Serialize and Deserialize Binary Tree**
  - Description: Design encode/decode for binary tree to/from string
  - Files: `006_serialize_deserialize_tree.{go,rs,py,ts}`
  - Key: BFS or preorder with nil markers
  - Complexity: O(n) time and space

- [ ] **Step 56** [W][P]: Problem 007 — **Course Schedule (Topological Sort)**
  - Description: Given prerequisites, determine if all courses can be finished (cycle detection)
  - Files: `007_course_schedule.{go,rs,py,ts}`
  - Key: Kahn's algorithm or DFS with coloring
  - Complexity: O(V+E)

**Category: Dynamic Programming (2 problems)**

- [ ] **Step 57** [W][P]: Problem 008 — **Longest Increasing Subsequence**
  - Description: Find length of longest strictly increasing subsequence
  - Files: `008_longest_increasing_subsequence.{go,rs,py,ts}`
  - Key: O(n^2) DP or O(n log n) patience sorting with binary search
  - Both solutions shown

- [ ] **Step 58** [W][P]: Problem 009 — **Edit Distance**
  - Description: Minimum operations (insert, delete, replace) to convert word1 to word2
  - Files: `009_edit_distance.{go,rs,py,ts}`
  - Key: 2D DP table, Wagner-Fischer algorithm
  - Complexity: O(mn) time, O(min(m,n)) space optimized

**Category: Concurrency (2 problems)**

- [ ] **Step 59** [W][P]: Problem 010 — **Bounded Blocking Queue**
  - Description: Implement thread-safe bounded queue with enqueue/dequeue that blocks
  - Files: `010_bounded_blocking_queue.{go,rs,py,ts}`
  - Go: channels or sync.Cond
  - Rust: Mutex + Condvar
  - Python: threading.Condition
  - TypeScript: Promise-based with async/await

- [ ] **Step 60** [W][P]: Problem 011 — **Web Crawler (Concurrent)**
  - Description: Crawl URLs concurrently, each URL visited once, respect max concurrency
  - Files: `011_web_crawler_concurrent.{go,rs,py,ts}`
  - Go: goroutines + sync.Map + semaphore
  - Rust: tokio + DashMap
  - Python: asyncio + aiohttp
  - TypeScript: Promise.all with pool

**Category: Bit Manipulation (1 problem)**

- [ ] **Step 61** [W][P]: Problem 012 — **Single Number III**
  - Description: Array where every element appears twice except two. Find those two.
  - Files: `012_single_number_iii.{go,rs,py,ts}`
  - Key: XOR all, find differing bit, partition
  - Complexity: O(n) time, O(1) space

**Category: System Design (2 problems)**

- [ ] **Step 62** [W][P]: Problem 013 — **Rate Limiter**
  - Description: Implement sliding window rate limiter (allow N requests per window)
  - Files: `013_rate_limiter.{go,rs,py,ts}`
  - Key: Sorted set or sliding window counter
  - Include thread-safe version

- [ ] **Step 63** [W][P]: Problem 014 — **Consistent Hashing**
  - Description: Implement consistent hash ring with virtual nodes
  - Files: `014_consistent_hashing.{go,rs,py,ts}`
  - Key: Sorted ring, binary search for node lookup
  - Include add/remove node, key lookup

### Verification

- [ ] **Step 64** [V]: Count files per directory
  ```bash
  echo "Go: $(ls ~/tmp/learning/extra/coding-questions-go/*.go 2>/dev/null | wc -l)"
  echo "Rust: $(ls ~/tmp/learning/extra/coding-questions-rust/*.rs 2>/dev/null | wc -l)"
  echo "Python: $(ls ~/tmp/learning/extra/coding-questions-python-new/*.py 2>/dev/null | wc -l)"
  echo "TypeScript: $(ls ~/tmp/learning/extra/coding-questions-typescript/*.ts 2>/dev/null | wc -l)"
  ```

- [ ] **Step 65** [V]: Expected: 14 files per language, 56 total.

- [ ] **Step 66** [B]: Syntax-check Go files
  ```bash
  cd ~/tmp/learning/extra/coding-questions-go && export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
  for f in *.go; do
    go vet "$f" 2>&1 | head -3 && echo "OK: $f" || echo "FAIL: $f"
  done
  ```

- [ ] **Step 67** [B]: Syntax-check Python files
  ```bash
  cd ~/tmp/learning/extra/coding-questions-python-new
  for f in *.py; do
    python3 -c "import py_compile; py_compile.compile('$f', doraise=True)" 2>&1 && echo "OK: $f" || echo "FAIL: $f"
  done
  ```

- [ ] **Step 68** [D]: Fix any syntax errors found in Steps 66-67.

- [ ] **Step 69** [V]: **PHASE 3 EXIT GATE** — 56 problem files created, Go and Python syntax-checked.

---

## PHASE 4: COORDINATOR — BUILD VERIFICATION (Steps 70-77)

**Goal**: Verify all new cheat sheet files integrate with the cs binary
**Prerequisite**: Phases 1 and 2 complete
**Time**: 10 minutes
**Agent**: Coordinator

- [ ] **Step 70** [B]: Full build with new embedded files
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet && export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH" && go build ./...
  ```

- [ ] **Step 71** [V]: Build succeeds. If fail → check for embed issues, fix.

- [ ] **Step 72** [B]: Run tests with race detection
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet && export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH" && go test ./... -count=1 -race 2>&1 | tail -20
  ```

- [ ] **Step 73** [V]: All tests pass. If fail → run twice (known transient issue).

- [ ] **Step 74** [B]: Verify new topics are discoverable
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet && export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
  ./cs list 2>/dev/null | grep -E "(go-testing|mocking|benchmarking|chaos-engineering|oidc|twelve-factor|concurrency-patterns|design-patterns|microservices-patterns|distributed-systems|event-driven|sre-fundamentals|api-design|caching-patterns)" | head -20
  ```

- [ ] **Step 75** [V]: All 17 new topics appear in list output.

- [ ] **Step 76** [B]: Verify new inventory counts
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  echo "Sheets: $(find sheets -name '*.md' | wc -l)"
  echo "Details: $(find detail -name '*.md' | wc -l)"
  echo "Categories: $(ls -d sheets/*/ | wc -l)"
  ```

- [ ] **Step 77** [V]: **PHASE 4 EXIT GATE** — Sheets=627, Details=627, Categories=57 (new: patterns/). Build green, tests pass, all 17 topics discoverable.

---

## PHASE 5: COORDINATOR — FENCE + FORMAT VERIFICATION (Steps 78-82)

**Goal**: Verify no unclosed code fences or formatting issues across ALL new files
**Prerequisite**: Phases 1, 2, 3 complete
**Time**: 5 minutes
**Agent**: Coordinator

- [ ] **Step 78** [B]: Check ALL new sheet+detail files for unclosed fences
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  bad=0
  for f in \
    sheets/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md \
    sheets/auth/oidc.md sheets/quality/{twelve-factor,sre-fundamentals}.md \
    sheets/api/api-design.md sheets/performance/caching-patterns.md \
    sheets/patterns/{concurrency-patterns,design-patterns,microservices-patterns,distributed-systems,event-driven-architecture}.md \
    detail/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md \
    detail/auth/oidc.md detail/quality/{twelve-factor,sre-fundamentals}.md \
    detail/api/api-design.md detail/performance/caching-patterns.md \
    detail/patterns/{concurrency-patterns,design-patterns,microservices-patterns,distributed-systems,event-driven-architecture}.md; do
    count=$(grep -c '```' "$f" 2>/dev/null || echo 0)
    remainder=$((count % 2))
    if [ "$remainder" -ne 0 ]; then
      echo "ODD FENCES ($count): $f"
      bad=$((bad+1))
    fi
  done
  echo "Total bad: $bad"
  ```

- [ ] **Step 79** [V]: 0 files with odd fence counts. If any → fix immediately.

- [ ] **Step 80** [B]: Verify no bare EOF heredoc patterns in new files
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet
  grep -rn "^EOF$" sheets/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md sheets/auth/oidc.md sheets/quality/*.md sheets/api/api-design.md sheets/performance/caching-patterns.md sheets/patterns/*.md 2>/dev/null | head -20
  echo "---"
  grep -rn "cat <<'EOF'" sheets/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md sheets/auth/oidc.md sheets/quality/*.md sheets/api/api-design.md sheets/performance/caching-patterns.md sheets/patterns/*.md 2>/dev/null | head -20
  ```

- [ ] **Step 81** [V]: 0 bare EOF patterns. Known issue from prior waves — prevent recurrence.

- [ ] **Step 82** [V]: **PHASE 5 EXIT GATE** — All formatting clean. Ready for commit.

---

## PHASE 6: COMMIT (Steps 83-85)

**Goal**: Commit all Wave 7 work in organized commits
**Prerequisite**: Phases 4 and 5 exit gates passed
**Time**: 5 minutes
**Agent**: Coordinator

- [ ] **Step 83** [C]: Commit cheat sheet topics
  ```bash
  cd /Users/govan/tmp/projects/cheat_sheet && git add \
    sheets/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md \
    detail/testing/{go-testing,mocking,benchmarking,integration-testing,property-based-testing,coverage,chaos-engineering}.md \
    sheets/auth/oidc.md detail/auth/oidc.md \
    sheets/quality/{twelve-factor,sre-fundamentals}.md detail/quality/{twelve-factor,sre-fundamentals}.md \
    sheets/api/api-design.md detail/api/api-design.md \
    sheets/performance/caching-patterns.md detail/performance/caching-patterns.md \
    sheets/patterns/ detail/patterns/ \
  && git commit -m "$(cat <<'COMMITEOF'
  Wave 7: 17 topics — testing (7), patterns (5), auth, quality (2), api, performance

  Captain+Micromanager gap analysis: testing category was critically thin (3→10),
  added new patterns/ category (5 topics), filled auth/quality/api/performance gaps.

  Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
  COMMITEOF
  )"
  ```

- [ ] **Step 84** [C]: Commit coding problems
  ```bash
  cd ~/tmp/learning/extra && git add \
    coding-questions-go/ \
    coding-questions-rust/ \
    coding-questions-python-new/ \
    coding-questions-typescript/ \
  && git commit -m "$(cat <<'COMMITEOF'
  Add multi-language coding problems: 14 problems x 4 languages (Go, Rust, Python, TS)

  LeetCode-style problems covering arrays, linked lists, trees, DP, concurrency,
  bit manipulation, and system design. Each with description, solution, and tests.

  Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
  COMMITEOF
  )" 2>/dev/null || echo "NOTE: learning/extra may not be a git repo — files written but not committed"
  ```

- [ ] **Step 85** [V]: **PHASE 6 EXIT GATE** — Commits successful. `git log --oneline -3` shows new commits.

---

## AGENT ASSIGNMENT MATRIX

| Phase | Agent | Parallelizable | Dependencies | Est. Time |
|-------|-------|----------------|-------------|-----------|
| Phase 0 | Coordinator | No | None | 5 min |
| Phase 1 | Agent Alpha | YES [P] | Phase 0 | 30-45 min |
| Phase 2 | Agent Bravo | YES [P] | Phase 0 | 30-45 min |
| Phase 3 | Agent Charlie | YES [P] | Phase 0 | 40-50 min |
| Phase 4 | Coordinator | No | Phases 1+2 | 10 min |
| Phase 5 | Coordinator | No | Phases 1+2+3 | 5 min |
| Phase 6 | Coordinator | No | Phases 4+5 | 5 min |

**Critical Path**: Phase 0 → Phase 3 (longest) → Phase 5 → Phase 6 = ~60 min
**Parallel Execution**: Phases 1+2+3 run simultaneously after Phase 0

## DELIVERABLES SUMMARY

| Objective | Items | Files | Location |
|-----------|-------|-------|----------|
| A: Cheat sheets | 17 topics | 34 (17 sheets + 17 details) | `sheets/` + `detail/` |
| B: Coding problems | 14 problems x 4 langs | 56 files | `~/tmp/learning/extra/coding-questions-{go,rust,python-new,typescript}/` |
| **Total** | | **90 files** | |

**Post-Wave 7 Inventory**: 627 sheets + 627 details = 1,254 cheat sheet files across 57 categories, plus 56 coding problems across 4 languages.

---

*Wave 7 Battle Plan — Forged 2026-04-04*
*6 Phases. 85 Steps. Testing finds its voice. Patterns earn their category. Four languages sharpen their swords.*
*The Captain saw the gap. The Micromanager counted the missing. The Warmonger wrote the orders.*
