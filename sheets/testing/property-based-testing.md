# Property-Based Testing (Go, Python, Rust)

Complete reference for property-based testing — rapid (Go), hypothesis (Python), proptest (Rust), generators, shrinking, common property patterns, and stateful testing.

## Core Concepts

### Example-Based vs Property-Based

```
Example-Based:
  "Add(2, 3) should equal 5"
  "Add(-1, 1) should equal 0"
  (you choose specific inputs)

Property-Based:
  "For all integers a, b: Add(a, b) == Add(b, a)"
  (the framework generates hundreds of random inputs)
```

### Key Components

| Component | Description |
|-----------|-------------|
| Property | A universal statement that must hold for all valid inputs |
| Generator | Produces random inputs of a given type |
| Shrinking | When a failure is found, reduces the input to the minimal failing case |
| Seed | Random seed for reproducibility |

## Common Property Patterns

### 1. Roundtrip / Encode-Decode

If you encode and then decode, you get the original back.

```go
// Go (rapid)
func TestJSONRoundtrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        original := &User{
            Name:  rapid.String().Draw(t, "name"),
            Age:   rapid.IntRange(0, 150).Draw(t, "age"),
            Email: rapid.StringMatching(`[a-z]+@[a-z]+\.[a-z]{2,3}`).Draw(t, "email"),
        }

        data, err := json.Marshal(original)
        if err != nil {
            t.Fatal(err)
        }

        var decoded User
        err = json.Unmarshal(data, &decoded)
        if err != nil {
            t.Fatal(err)
        }

        if !reflect.DeepEqual(original, &decoded) {
            t.Fatalf("roundtrip failed: %+v != %+v", original, &decoded)
        }
    })
}
```

```python
# Python (hypothesis)
from hypothesis import given
import hypothesis.strategies as st

@given(st.text(), st.integers(0, 150))
def test_json_roundtrip(name, age):
    original = {"name": name, "age": age}
    encoded = json.dumps(original)
    decoded = json.loads(encoded)
    assert original == decoded
```

### 2. Idempotency

Applying an operation twice gives the same result as once.

```go
func TestSortIdempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        data := rapid.SliceOf(rapid.Int()).Draw(t, "data")

        sorted1 := sortCopy(data)
        sorted2 := sortCopy(sorted1)

        if !reflect.DeepEqual(sorted1, sorted2) {
            t.Fatalf("sort not idempotent: %v != %v", sorted1, sorted2)
        }
    })
}
```

```python
@given(st.lists(st.integers()))
def test_sort_idempotent(xs):
    once = sorted(xs)
    twice = sorted(once)
    assert once == twice
```

### 3. Invariant Preservation

An operation preserves certain properties.

```go
func TestSortPreservesLength(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        data := rapid.SliceOf(rapid.Int()).Draw(t, "data")
        sorted := sortCopy(data)

        if len(sorted) != len(data) {
            t.Fatalf("length changed: %d -> %d", len(data), len(sorted))
        }
    })
}

func TestSortPreservesElements(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        data := rapid.SliceOf(rapid.Int()).Draw(t, "data")
        sorted := sortCopy(data)

        // Same elements (as multiset)
        origCounts := countElements(data)
        sortCounts := countElements(sorted)
        if !reflect.DeepEqual(origCounts, sortCounts) {
            t.Fatal("elements changed after sort")
        }
    })
}
```

### 4. Oracle / Model-Based

Compare your implementation against a known-correct (but possibly slower) reference.

```go
func TestCustomMapMatchesStdlib(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        key := rapid.String().Draw(t, "key")
        value := rapid.Int().Draw(t, "value")

        // System under test
        custom := NewCustomMap()
        custom.Set(key, value)
        got, ok := custom.Get(key)

        // Oracle (reference implementation)
        oracle := make(map[string]int)
        oracle[key] = value
        want, wantOk := oracle[key]

        if ok != wantOk || got != want {
            t.Fatalf("custom map diverges from stdlib: got=%d ok=%v, want=%d ok=%v",
                got, ok, want, wantOk)
        }
    })
}
```

### 5. Commutativity

Order of operations does not matter.

```go
func TestMergeCommutative(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        a := rapid.MapOf(rapid.String(), rapid.Int()).Draw(t, "a")
        b := rapid.MapOf(rapid.String(), rapid.Int()).Draw(t, "b")

        ab := Merge(a, b)
        ba := Merge(b, a)

        if !reflect.DeepEqual(ab, ba) {
            t.Fatalf("merge not commutative: Merge(a,b)=%v, Merge(b,a)=%v", ab, ba)
        }
    })
}
```

### 6. Algebraic Laws

```python
# Associativity
@given(st.integers(), st.integers(), st.integers())
def test_add_associative(a, b, c):
    assert custom_add(custom_add(a, b), c) == custom_add(a, custom_add(b, c))

# Identity element
@given(st.integers())
def test_add_identity(a):
    assert custom_add(a, 0) == a
    assert custom_add(0, a) == a

# Inverse
@given(st.integers())
def test_add_inverse(a):
    assert custom_add(a, -a) == 0
```

## Go: rapid

### Installation

```bash
go get pgregory.net/rapid
```

### Generators

```go
func TestWithGenerators(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Primitive types
        i := rapid.Int().Draw(t, "int")
        i32 := rapid.Int32Range(-100, 100).Draw(t, "int32")
        f := rapid.Float64().Draw(t, "float")
        s := rapid.String().Draw(t, "string")
        b := rapid.Bool().Draw(t, "bool")
        bs := rapid.SliceOfN(rapid.Byte(), 1, 100).Draw(t, "bytes")

        // Collections
        ints := rapid.SliceOf(rapid.Int()).Draw(t, "ints")
        m := rapid.MapOf(rapid.String(), rapid.Int()).Draw(t, "map")

        // Constrained
        positiveInt := rapid.IntRange(1, 1000).Draw(t, "positive")
        shortStr := rapid.StringN(1, 50, -1).Draw(t, "short")
        email := rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,8}\.(com|org|net)`).Draw(t, "email")

        // Choice from list
        status := rapid.SampledFrom([]string{"active", "inactive", "pending"}).Draw(t, "status")

        // One-of (union types)
        val := rapid.OneOf(
            rapid.Just("special"),
            rapid.StringN(1, 10, -1),
        ).Draw(t, "val")

        _, _, _, _, _, _, _, _, _, _, _, _, _ = i, i32, f, s, b, bs, ints, m, positiveInt, shortStr, email, status, val
    })
}
```

### Custom Generators

```go
func genUser() *rapid.Generator[*User] {
    return rapid.Custom(func(t *rapid.T) *User {
        return &User{
            Name:  rapid.StringN(1, 50, -1).Draw(t, "name"),
            Age:   rapid.IntRange(0, 150).Draw(t, "age"),
            Email: rapid.StringMatching(`[a-z]+@[a-z]+\.com`).Draw(t, "email"),
            Role:  rapid.SampledFrom([]string{"admin", "user", "guest"}).Draw(t, "role"),
        }
    })
}

func TestWithCustomGen(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        user := genUser().Draw(t, "user")
        // test with user
    })
}
```

### Configuration

```go
rapid.Check(t, func(t *rapid.T) {
    // properties
}, rapid.WithRuns(1000))  // default is 100
```

## Python: hypothesis

### Installation

```bash
pip install hypothesis
```

### Strategies

```python
from hypothesis import given, assume, settings, HealthCheck
import hypothesis.strategies as st

# Primitive types
@given(st.integers(), st.text(), st.floats(allow_nan=False), st.booleans())
def test_primitives(i, s, f, b):
    pass

# Constrained
@given(st.integers(min_value=1, max_value=100))
def test_positive(n):
    assert n > 0

# Collections
@given(st.lists(st.integers(), min_size=1, max_size=50))
def test_nonempty_list(xs):
    assert len(xs) >= 1

# Dictionaries
@given(st.dictionaries(st.text(), st.integers(), min_size=1))
def test_dict(d):
    assert len(d) >= 1

# Complex nested structures
@given(st.recursive(
    st.integers() | st.text(),
    lambda children: st.lists(children) | st.dictionaries(st.text(), children),
    max_leaves=50,
))
def test_nested(data):
    pass

# Filtering
@given(st.integers().filter(lambda x: x != 0))
def test_nonzero(n):
    assert n != 0

# assume() for preconditions
@given(st.integers(), st.integers())
def test_division(a, b):
    assume(b != 0)
    result = a / b
    assert result * b == pytest.approx(a)
```

### Composite Strategies

```python
@st.composite
def user_strategy(draw):
    name = draw(st.text(min_size=1, max_size=50))
    age = draw(st.integers(min_value=0, max_value=150))
    email = draw(st.emails())
    role = draw(st.sampled_from(["admin", "user", "guest"]))
    return User(name=name, age=age, email=email, role=role)

@given(user_strategy())
def test_user_serialization(user):
    data = user.to_dict()
    restored = User.from_dict(data)
    assert user == restored
```

### Settings

```python
from hypothesis import settings, Phase

@settings(
    max_examples=500,           # default 100
    deadline=timedelta(seconds=5),  # per-example timeout
    suppress_health_check=[HealthCheck.too_slow],
    database=None,              # disable example database
)
@given(st.lists(st.integers()))
def test_with_settings(xs):
    pass

# Profile for CI
settings.register_profile("ci", max_examples=1000)
settings.register_profile("dev", max_examples=50)
settings.load_profile(os.getenv("HYPOTHESIS_PROFILE", "dev"))
```

## Rust: proptest

### Setup

```toml
[dev-dependencies]
proptest = "1.4"
```

### Basic Usage

```rust
use proptest::prelude::*;

proptest! {
    #[test]
    fn test_sort_preserves_length(ref v in prop::collection::vec(any::<i32>(), 0..100)) {
        let mut sorted = v.clone();
        sorted.sort();
        prop_assert_eq!(v.len(), sorted.len());
    }

    #[test]
    fn test_roundtrip(s in "\\PC*") {
        let encoded = encode(&s);
        let decoded = decode(&encoded).unwrap();
        prop_assert_eq!(&s, &decoded);
    }

    #[test]
    fn test_add_commutative(a in -1000i64..1000, b in -1000i64..1000) {
        prop_assert_eq!(a + b, b + a);
    }
}
```

### Custom Strategies

```rust
#[derive(Debug, Clone)]
struct User {
    name: String,
    age: u8,
}

fn arb_user() -> impl Strategy<Value = User> {
    ("[a-z]{1,20}", 0u8..150).prop_map(|(name, age)| User { name, age })
}

proptest! {
    #[test]
    fn test_user_roundtrip(user in arb_user()) {
        let json = serde_json::to_string(&user).unwrap();
        let decoded: User = serde_json::from_str(&json).unwrap();
        prop_assert_eq!(user.name, decoded.name);
        prop_assert_eq!(user.age, decoded.age);
    }
}
```

## Stateful Testing

### State Machine Testing with hypothesis

```python
from hypothesis.stateful import RuleBasedStateMachine, rule, invariant, initialize

class SetMachine(RuleBasedStateMachine):
    """Test a custom set implementation against Python's built-in set."""

    def __init__(self):
        super().__init__()
        self.model = set()        # oracle
        self.real = CustomSet()   # system under test

    @rule(value=st.integers())
    def add(self, value):
        self.model.add(value)
        self.real.add(value)

    @rule(value=st.integers())
    def discard(self, value):
        self.model.discard(value)
        self.real.discard(value)

    @rule(value=st.integers())
    def contains(self, value):
        assert (value in self.model) == self.real.contains(value)

    @invariant()
    def lengths_match(self):
        assert len(self.model) == self.real.size()

TestSet = SetMachine.TestCase
```

### Stateful Testing with rapid

```go
func TestMapStateful(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        m := NewCustomMap()           // system under test
        oracle := make(map[string]int) // reference

        // Generate a sequence of operations
        nOps := rapid.IntRange(1, 100).Draw(t, "nOps")
        for i := 0; i < nOps; i++ {
            key := rapid.StringN(1, 10, -1).Draw(t, "key")
            op := rapid.SampledFrom([]string{"set", "get", "delete"}).Draw(t, "op")

            switch op {
            case "set":
                val := rapid.Int().Draw(t, "val")
                m.Set(key, val)
                oracle[key] = val
            case "get":
                got, ok := m.Get(key)
                want, wantOk := oracle[key]
                if ok != wantOk || got != want {
                    t.Fatalf("Get(%q): got=(%d,%v), want=(%d,%v)", key, got, ok, want, wantOk)
                }
            case "delete":
                m.Delete(key)
                delete(oracle, key)
            }
        }

        // Final invariant: sizes match
        if m.Len() != len(oracle) {
            t.Fatalf("size mismatch: %d != %d", m.Len(), len(oracle))
        }
    })
}
```

## Shrinking

### How Shrinking Works

When a property fails, the framework finds the *minimal* failing input:

```
Found failing input: [5, 3, -2, 8, 0, 1, -7, 4]
Shrinking...
  Try: [5, 3, -2, 8, 0, 1, -7]  -> PASS
  Try: [5, 3, -2, 8]            -> FAIL
  Try: [5, 3]                    -> PASS
  Try: [5, -2, 8]               -> FAIL
  Try: [-2, 8]                  -> FAIL
  Try: [-2]                     -> FAIL
  Try: [0]                      -> PASS
  Try: [-1]                     -> FAIL
Minimal failing input: [-1]
```

### Shrinking Strategies

| Type | Strategy |
|------|----------|
| Integer | Binary search toward 0 |
| String | Remove characters, simplify to ASCII |
| List | Remove elements, shrink remaining |
| Custom | User-defined shrink function |

## Tips

- Start with roundtrip and invariant properties — they are easiest to identify
- Use `assume()` sparingly — heavy filtering slows generation; prefer constrained generators
- Stateful testing is the most powerful technique but also the most complex
- hypothesis saves failing examples in a database and replays them in subsequent runs
- Always check the shrunk output — it often reveals the exact boundary condition
- rapid's `-rapid.seed=N` flag reproduces failures deterministically
- Property-based tests complement example-based tests — use both
- Common bug categories caught: off-by-one, empty collection, negative numbers, Unicode edge cases
- Run property tests with more iterations in CI than locally (`HYPOTHESIS_PROFILE=ci`)
- For concurrent code, combine property testing with `-race` flag

## See Also

- `sheets/testing/go-testing.md` — Go testing fundamentals
- `sheets/testing/coverage.md` — measuring what properties exercise
- `detail/testing/property-based-testing.md` — sampling theory and bug-finding probability

## References

- https://pgregory.net/rapid/ — rapid (Go)
- https://hypothesis.readthedocs.io/ — hypothesis (Python)
- https://proptest-rs.github.io/proptest/ — proptest (Rust)
- https://fsharpforfunandprofit.com/posts/property-based-testing/ — property patterns guide
- https://www.hillelwayne.com/post/contract-testing/ — formal property identification
