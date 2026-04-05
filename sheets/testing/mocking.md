# Mocking (Test Doubles for Go, Python, JS)

Complete reference for test doubles across languages — gomock, testify/mock, mockery, Python unittest.mock, Jest mocks, and the taxonomy of spy/stub/mock/fake.

## Test Double Taxonomy

### Definitions

```
+--------+-----------------------------------------------------------+
| Type   | Description                                               |
+--------+-----------------------------------------------------------+
| Dummy  | Passed around but never used. Satisfies a parameter.      |
| Stub   | Returns canned answers. No assertion on calls.             |
| Spy    | Records calls for later verification. May delegate.       |
| Mock   | Pre-programmed expectations. Verified automatically.       |
| Fake   | Working implementation, simplified. (e.g., in-memory DB)  |
+--------+-----------------------------------------------------------+
```

### When NOT to Mock

- **Thin wrappers**: Mocking `os.ReadFile` adds coupling to implementation, not behavior
- **Value objects**: Structs without side effects should be used directly
- **Your own code one layer down**: Prefer integration over excessive mocking
- **Highly stable dependencies**: The standard library rarely changes
- **When a fake exists**: Use an in-memory implementation over a mock

Rule of thumb: mock at architectural boundaries (network, disk, clock, external APIs), not within your own package.

## Go: Interface-Based Mocking

### The Pattern

```go
// Define a narrow interface at the consumer site
type UserStore interface {
    GetUser(ctx context.Context, id string) (*User, error)
    SaveUser(ctx context.Context, user *User) error
}

// Production implementation
type PostgresUserStore struct {
    db *sql.DB
}

func (s *PostgresUserStore) GetUser(ctx context.Context, id string) (*User, error) {
    // real database query
}

// Test: hand-rolled stub
type stubUserStore struct {
    users map[string]*User
    err   error
}

func (s *stubUserStore) GetUser(_ context.Context, id string) (*User, error) {
    if s.err != nil {
        return nil, s.err
    }
    u, ok := s.users[id]
    if !ok {
        return nil, ErrNotFound
    }
    return u, nil
}

func (s *stubUserStore) SaveUser(_ context.Context, user *User) error {
    if s.err != nil {
        return s.err
    }
    s.users[user.ID] = user
    return nil
}
```

### Dependency Injection

```go
type UserService struct {
    store  UserStore
    logger *slog.Logger
    clock  func() time.Time // injectable clock
}

func NewUserService(store UserStore, logger *slog.Logger) *UserService {
    return &UserService{
        store:  store,
        logger: logger,
        clock:  time.Now,
    }
}

// In tests:
func TestUserService(t *testing.T) {
    store := &stubUserStore{users: map[string]*User{
        "123": {ID: "123", Name: "Alice"},
    }}

    svc := NewUserService(store, slog.Default())
    svc.clock = func() time.Time {
        return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
    }

    user, err := svc.GetUser(context.Background(), "123")
    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

## Go: gomock

### Generate Mocks

```bash
# Install
go install go.uber.org/mock/mockgen@latest

# Source mode — from interface in a file
mockgen -source=store.go -destination=mock_store_test.go -package=mypackage

# Reflect mode — from a package + interface name
mockgen -destination=mock_store_test.go -package=mypackage \
    mymodule/internal/store UserStore

# go:generate directive
//go:generate mockgen -source=store.go -destination=mock_store_test.go -package=mypackage
```

### Using gomock in Tests

```go
import (
    "testing"
    "go.uber.org/mock/gomock"
)

func TestGetUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    // ctrl.Finish() is called automatically via t.Cleanup in modern gomock

    store := NewMockUserStore(ctrl)

    // Expect a call with specific args, return specific values
    store.EXPECT().
        GetUser(gomock.Any(), "123").
        Return(&User{ID: "123", Name: "Alice"}, nil).
        Times(1)

    svc := NewUserService(store, slog.Default())
    user, err := svc.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)
}
```

### gomock Matchers

```go
store.EXPECT().GetUser(gomock.Any(), gomock.Eq("123"))     // exact match
store.EXPECT().GetUser(gomock.Any(), gomock.Not(""))        // not empty
store.EXPECT().GetUser(gomock.Any(), gomock.Nil())          // nil
store.EXPECT().SaveUser(gomock.Any(), gomock.Any())         // any value

// Custom matcher
type hasNameMatcher struct{ name string }
func (m hasNameMatcher) Matches(x interface{}) bool {
    u, ok := x.(*User)
    return ok && u.Name == m.name
}
func (m hasNameMatcher) String() string { return "has name " + m.name }
func HasName(name string) gomock.Matcher { return hasNameMatcher{name} }

store.EXPECT().SaveUser(gomock.Any(), HasName("Alice"))
```

### Ordering

```go
first := store.EXPECT().GetUser(gomock.Any(), "1").Return(u1, nil)
store.EXPECT().GetUser(gomock.Any(), "2").Return(u2, nil).After(first)

// or use InOrder
gomock.InOrder(
    store.EXPECT().GetUser(gomock.Any(), "1").Return(u1, nil),
    store.EXPECT().SaveUser(gomock.Any(), gomock.Any()).Return(nil),
)
```

## Go: testify/mock

### Define Mock

```go
import "github.com/stretchr/testify/mock"

type MockUserStore struct {
    mock.Mock
}

func (m *MockUserStore) GetUser(ctx context.Context, id string) (*User, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserStore) SaveUser(ctx context.Context, user *User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}
```

### Using in Tests

```go
func TestService(t *testing.T) {
    store := new(MockUserStore)

    store.On("GetUser", mock.Anything, "123").
        Return(&User{ID: "123", Name: "Alice"}, nil).
        Once()

    store.On("SaveUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
        return u.Name == "Alice"
    })).Return(nil)

    svc := NewUserService(store, slog.Default())
    user, err := svc.GetUser(context.Background(), "123")

    require.NoError(t, err)
    assert.Equal(t, "Alice", user.Name)

    store.AssertExpectations(t)
    store.AssertCalled(t, "GetUser", mock.Anything, "123")
    store.AssertNumberOfCalls(t, "GetUser", 1)
}
```

## Go: mockery v2

### Generate from Interfaces

```bash
# Install
go install github.com/vektra/mockery/v2@latest

# Generate mocks for all exported interfaces
mockery --all --with-expecter

# Specific interface
mockery --name=UserStore --dir=./internal/store --output=./internal/store/mocks

# .mockery.yaml config
```

```yaml
# .mockery.yaml
with-expecter: true
packages:
  mymodule/internal/store:
    interfaces:
      UserStore:
        config:
          dir: "internal/store/mocks"
```

### Using mockery Output

```go
import "mymodule/internal/store/mocks"

func TestService(t *testing.T) {
    store := mocks.NewMockUserStore(t) // auto t.Cleanup

    store.EXPECT().
        GetUser(mock.Anything, "123").
        Return(&User{ID: "123", Name: "Alice"}, nil)

    // test logic...
}
```

## Python: unittest.mock

### Basic Mocking

```python
from unittest.mock import MagicMock, patch, PropertyMock

# MagicMock — auto-generates attributes and return values
mock_db = MagicMock()
mock_db.query.return_value = [{"id": 1, "name": "Alice"}]

result = mock_db.query("SELECT * FROM users")
mock_db.query.assert_called_once_with("SELECT * FROM users")
```

### patch Decorator

```python
from unittest.mock import patch

# Patch where it's USED, not where it's defined
@patch("myapp.services.user_service.requests.get")
def test_fetch_user(mock_get):
    mock_get.return_value.status_code = 200
    mock_get.return_value.json.return_value = {"name": "Alice"}

    user = fetch_user("123")

    assert user.name == "Alice"
    mock_get.assert_called_once_with("https://api.example.com/users/123")
```

### Context Manager

```python
def test_file_read():
    with patch("builtins.open", mock_open(read_data="file content")) as mock_file:
        result = read_config("/etc/app.conf")
        mock_file.assert_called_once_with("/etc/app.conf", "r")
        assert result == "file content"
```

### spec and autospec

```python
# spec restricts the mock to the real class's interface
mock_db = MagicMock(spec=DatabaseConnection)
mock_db.query("SELECT 1")       # OK — query exists
mock_db.nonexistent()            # raises AttributeError

# autospec does this recursively
@patch("myapp.db.Connection", autospec=True)
def test_with_autospec(MockConn):
    conn = MockConn.return_value
    conn.execute.return_value = 42
```

### side_effect

```python
# Raise an exception
mock_db.query.side_effect = ConnectionError("lost connection")

# Return different values on successive calls
mock_db.query.side_effect = [result1, result2, ConnectionError("fail")]

# Custom function
def fake_query(sql):
    if "users" in sql:
        return [{"id": 1}]
    return []
mock_db.query.side_effect = fake_query
```

## JavaScript: Jest Mocks

### jest.fn

```javascript
const mockCallback = jest.fn(x => x + 42);

[0, 1].forEach(mockCallback);

expect(mockCallback).toHaveBeenCalledTimes(2);
expect(mockCallback).toHaveBeenCalledWith(0);
expect(mockCallback).toHaveBeenCalledWith(1);
expect(mockCallback.mock.results[0].value).toBe(42);
```

### jest.mock (Module Mock)

```javascript
// Automatically mocks the entire module
jest.mock('./database');

import { getUser } from './database';

getUser.mockResolvedValue({ id: '123', name: 'Alice' });

test('fetches user', async () => {
    const user = await getUser('123');
    expect(user.name).toBe('Alice');
    expect(getUser).toHaveBeenCalledWith('123');
});
```

### jest.spyOn

```javascript
const video = {
    play() { return true; },
    stop() { return false; }
};

const spy = jest.spyOn(video, 'play');
video.play();

expect(spy).toHaveBeenCalled();
spy.mockRestore(); // restore original implementation
```

### Manual Mocks (__mocks__)

```
src/
  __mocks__/
    axios.js         # manual mock for axios
  services/
    __mocks__/
      userService.js  # manual mock for userService
    userService.js
```

```javascript
// __mocks__/axios.js
export default {
    get: jest.fn(() => Promise.resolve({ data: {} })),
    post: jest.fn(() => Promise.resolve({ data: {} })),
};
```

## Tips

- Mock at boundaries, not within — mock the database adapter, not the SQL query builder
- Prefer stubs (canned answers) over mocks (behavior verification) when possible
- Use `spec=True` (Python) or typed mocks (Go) to catch interface drift
- In Go, define interfaces at the consumer site, not the provider
- Keep interfaces small — a 1-2 method interface is easier to mock than a 20-method one
- gomock's `gomock.Any()` is tempting but weakens assertions — be specific where it matters
- In Python, always patch where it's *imported*, not where it's *defined*
- Test the contract, not the implementation — verify results, not call sequences
- If you need more than 3 mocks in one test, your unit is too big
- Fakes (in-memory implementations) are often better than mocks for complex interfaces

## See Also

- `sheets/testing/go-testing.md` — Go testing fundamentals
- `sheets/testing/integration-testing.md` — when to stop mocking and test for real
- `sheets/testing/property-based-testing.md` — generative testing as alternative to manual stubs

## References

- https://go.uber.org/mock — gomock documentation
- https://github.com/vektra/mockery — mockery v2
- https://github.com/stretchr/testify — testify/mock
- https://docs.python.org/3/library/unittest.mock.html — Python mock
- https://jestjs.io/docs/mock-functions — Jest mock functions
- https://martinfowler.com/articles/mocksArentStubs.html — Fowler's test double taxonomy
