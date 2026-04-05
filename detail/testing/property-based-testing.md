# The Mathematics of Property-Based Testing — Sampling, Shrinking, and Bug-Finding Probability

> *Property-based testing is random sampling applied to program verification. This explores the sampling theory that governs how likely we are to find bugs, the binary-search mathematics of shrinking, the birthday paradox in input generation, and the tradeoffs between random and coverage-guided generation.*

---

## 1. Random Sampling Theory (Probability)

### The Problem

A property-based test generates $N$ random inputs and checks a property against each. What is the probability of finding a bug that exists in the input space?

### The Formula

Let the input space be $\Omega$ with $|\Omega|$ possible inputs. Let $B \subseteq \Omega$ be the set of bug-triggering inputs, with $|B| / |\Omega| = p$ (the bug density).

For uniform random sampling, the probability of finding at least one bug-triggering input in $N$ draws:

$$P(\text{find bug}) = 1 - (1 - p)^N$$

To achieve detection probability $\geq \alpha$:

$$N \geq \frac{\ln(1 - \alpha)}{\ln(1 - p)}$$

For small $p$ (rare bugs), the approximation holds:

$$N \approx \frac{-\ln(1 - \alpha)}{p}$$

### Worked Examples

**Example 1**: A function has an off-by-one error that triggers for exactly 1 out of every 1000 inputs ($p = 0.001$).

To achieve 95% detection probability:

$$N \geq \frac{\ln(0.05)}{\ln(0.999)} = \frac{-2.996}{-0.001} \approx 2996$$

Default hypothesis runs 100 examples: $P = 1 - 0.999^{100} = 9.5\%$ — poor detection. Increasing to 3000: $P = 95.0\%$.

**Example 2**: A function fails for negative inputs ($p = 0.5$ assuming uniform distribution over integers).

$$N \geq \frac{\ln(0.05)}{\ln(0.5)} = \frac{-2.996}{-0.693} \approx 4.3$$

Just 5 random inputs give 96.9% detection probability.

**Key insight**: Property-based testing excels at finding bugs with moderate density ($p > 0.01$). For very rare bugs ($p < 0.0001$), you need coverage-guided techniques or domain-specific generators.

## 2. Shrinking as Binary Search (Search Algorithms)

### The Problem

When a failing input is found, shrinking reduces it to the minimal failing case. How does this work, and what is its complexity?

### The Formula

Shrinking is a search over a **shrink tree**: a directed acyclic graph where each node is an input value and edges point to "simpler" values.

For an integer $n$, the shrink tree has depth:

$$d = \lceil \log_2 |n| \rceil + 1$$

because integer shrinking uses binary search toward 0.

For a list of length $l$ with elements from a domain of size $v$:

$$d_{list} \leq l \cdot \lceil \log_2 l \rceil + l \cdot \lceil \log_2 v \rceil$$

The first term accounts for removing elements (binary search for minimal length), and the second for shrinking each remaining element.

**Total shrink attempts** (worst case):

$$S = O(d \cdot b)$$

where $b$ is the branching factor (number of simpler values tried per step). For hypothesis, $b$ is typically 2-8.

**Effective shrink time**:

$$T_{shrink} = S \cdot T_{property}$$

### Worked Examples

**Example**: A failing list `[42, -17, 8, 0, -3, 99, -1]` with a bug triggered by any negative element.

Shrinking steps:
1. Remove elements: `[42, -17, 8, 0]` -- still fails (has -17)
2. Remove more: `[-17, 8]` -- still fails
3. Remove more: `[-17]` -- still fails
4. Shrink -17: try `[-9]` -- fails, try `[-5]` -- fails, try `[-3]` -- fails, try `[-2]` -- fails, try `[-1]` -- fails, try `[0]` -- passes
5. Minimal failing input: `[-1]`

Total steps: $\lceil \log_2 7 \rceil + \lceil \log_2 17 \rceil = 3 + 5 = 8$ shrink rounds, each with 2-3 attempts = ~20 property evaluations.

## 3. Birthday Paradox in Input Generation (Combinatorics)

### The Problem

When generating random inputs, collisions (duplicate inputs) waste test budget. The birthday paradox determines how quickly collisions occur.

### The Formula

For a domain of size $D$, the expected number of unique values after $N$ draws:

$$E[\text{unique}] = D \left(1 - \left(1 - \frac{1}{D}\right)^N\right)$$

The probability of at least one collision in $N$ draws:

$$P(\text{collision}) \approx 1 - e^{-N(N-1)/(2D)}$$

Collisions become likely (~50%) when:

$$N \approx \sqrt{2D \ln 2} \approx 1.177\sqrt{D}$$

### Worked Examples

**Example 1**: Generating random `uint8` values (D = 256).

Collisions become likely at $N \approx 1.177\sqrt{256} = 18.8$ draws. After 100 draws, expected unique values:

$$E[\text{unique}] = 256\left(1 - \left(\frac{255}{256}\right)^{100}\right) = 256 \times 0.324 = 82.9$$

Only 83 unique values out of 100 draws — 17% waste.

**Example 2**: Generating random strings of length 5 from `[a-z]` ($D = 26^5 = 11,881,376$).

Collisions become likely at $N \approx 1.177\sqrt{11,881,376} \approx 4055$. With the default 100 runs, collision probability is negligible.

**Implication**: For small domains, use exhaustive enumeration instead of random sampling. For medium domains, increase the test count. For large domains, random sampling is efficient.

## 4. Coverage-Guided vs Random Generation (Information Theory)

### The Problem

Random generation treats all inputs equally, but bugs often cluster near specific code paths. Coverage-guided generation (as in fuzzing) directs generation toward unexplored paths. When is each approach better?

### The Formula

**Random generation** explores the input space uniformly. Expected code coverage after $N$ inputs, assuming $P$ total paths:

$$C_{random}(N) = P\left(1 - \left(1 - \frac{1}{P}\right)^N\right)$$

This converges logarithmically: covering the last few paths requires exponentially more inputs.

**Coverage-guided generation** maintains a corpus and mutates inputs that discover new paths. Expected coverage:

$$C_{guided}(N) \approx P\left(1 - e^{-\beta N / P}\right)$$

where $\beta > 1$ is the guidance efficiency factor (typically 2-10x for well-instrumented programs).

**Crossover point**: coverage-guided becomes more efficient when:

$$N > \frac{P}{\beta - 1}$$

### Worked Examples

**Example**: A parser with 50 distinct code paths ($P = 50$), coverage-guided with $\beta = 5$.

After 100 random inputs: $C_{random} = 50(1 - 0.98^{100}) = 50 \times 0.867 = 43.4$ paths (87%).

After 100 guided inputs: $C_{guided} = 50(1 - e^{-5 \times 100/50}) = 50(1 - e^{-10}) = 50 \times 0.99995 = 49.997$ paths (~100%).

Coverage-guided reaches near-100% in 100 inputs; random needs ~230 for the same coverage.

**Go's built-in fuzzing** (go test -fuzz) uses coverage-guided generation:

```go
func FuzzParse(f *testing.F) {
    f.Add([]byte(`{"key": "value"}`))
    f.Fuzz(func(t *testing.T, data []byte) {
        result, err := Parse(data)
        if err != nil {
            return // invalid input, skip
        }
        // Check invariants on valid parses
        encoded := Encode(result)
        result2, err := Parse(encoded)
        if err != nil {
            t.Fatalf("roundtrip failed: %v", err)
        }
        if !reflect.DeepEqual(result, result2) {
            t.Fatal("roundtrip mismatch")
        }
    })
}
```

## 5. Bug-Finding Probability vs Test Count (Decision Theory)

### The Problem

Given a fixed testing budget, how should we allocate between property-based tests (many random inputs per property) and example-based tests (few curated inputs)?

### The Formula

For $k$ properties with $N$ random inputs each, and bug density $p_i$ for property $i$:

$$P(\text{find any bug}) = 1 - \prod_{i=1}^{k}(1 - p_i)^N$$

For $m$ example-based tests, each with detection probability $q_j$:

$$P(\text{find any bug}) = 1 - \prod_{j=1}^{m}(1 - q_j)$$

The comparative advantage of property-based testing depends on:

$$\frac{\partial P}{\partial N} = \sum_{i=1}^{k} (1-p_i)^{N-1} p_i \cdot \prod_{j \neq i}(1-p_j)^N$$

This derivative decreases with $N$, showing diminishing returns. The optimal strategy is often a mix: example-based tests for known edge cases ($q_j \approx 1$), property-based for unknown bugs ($p_i$ unknown but uniform sampling provides broad coverage).

### Worked Examples

**Example**: 5 properties, uniform $p = 0.01$, budget for 500 total test evaluations.

Option A: 5 properties x 100 inputs = 500 evaluations
$$P_A = 1 - (1 - 0.01)^{100 \times 5} = 1 - 0.99^{500} = 1 - 0.00657 = 99.3\%$$

But this is misleading — the bugs are in *different* properties. Correct:
$$P_A = 1 - (0.99^{100})^5 = 1 - 0.366^5 = 1 - 0.00656 = 99.3\%$$

Option B: 50 properties x 10 inputs = 500 evaluations (broader, shallower)
$$P_B = 1 - (0.99^{10})^{50} = 1 - 0.904^{50} = 1 - 0.00627 = 99.4\%$$

Nearly identical. But if bugs cluster in a few properties, Option A (deeper) wins. If bugs are spread across many properties, Option B (broader) wins.

## Prerequisites

- Probability theory (Bernoulli trials, geometric distribution)
- Combinatorics (birthday problem, counting arguments)
- Search algorithms (binary search, tree traversal)
- Information theory (entropy, coverage metrics)

## Complexity

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|-----------------|
| Random generation ($N$ inputs) | $O(N \cdot G)$ | $O(N)$ or $O(1)$ streaming |
| Shrinking (integer) | $O(\log n \cdot T_{prop})$ | $O(\log n)$ |
| Shrinking (list of length $l$) | $O(l \log l \cdot T_{prop})$ | $O(l)$ |
| Coverage-guided mutation | $O(N \cdot (G + I))$ | $O(C \cdot S)$ corpus |
| Stateful testing ($k$ steps) | $O(k \cdot T_{prop})$ | $O(k)$ state history |

Where: $G$ = generation cost, $T_{prop}$ = property evaluation time, $I$ = instrumentation overhead, $C$ = corpus size, $S$ = average input size.
