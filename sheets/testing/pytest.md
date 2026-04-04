# pytest (Python Testing Framework)

Full-featured Python testing framework with powerful fixtures, parametrization, and plugin ecosystem.

## Running Tests

### Basic execution

```bash
pytest                              # discover and run all tests
pytest tests/                       # run tests in directory
pytest tests/test_api.py            # run single file
pytest tests/test_api.py::test_login  # run single test
pytest -k "login and not admin"     # keyword filter
pytest -x                           # stop on first failure
pytest -x --pdb                     # drop into debugger on failure
pytest --lf                         # re-run last failed tests only
pytest --ff                         # run failed first, then rest
pytest -v                           # verbose output with test names
pytest -q                           # quiet, minimal output
```

### Parallel execution (pytest-xdist)

```bash
pip install pytest-xdist
pytest -n auto                      # use all CPU cores
pytest -n 4                         # use 4 workers
pytest -n auto --dist loadscope     # group by module
pytest -n auto --dist loadfile      # group by file
```

## Writing Tests

### Basic assertions

```python
def test_addition():
    assert 1 + 1 == 2

def test_string_contains():
    assert "hello" in "hello world"

def test_exception():
    with pytest.raises(ValueError, match=r"invalid.*format"):
        parse_date("not-a-date")

def test_warning():
    with pytest.warns(DeprecationWarning):
        deprecated_function()

def test_approximate():
    assert 0.1 + 0.2 == pytest.approx(0.3)
    assert [0.1 + 0.2, 0.2 + 0.4] == pytest.approx([0.3, 0.6])
```

### Assert rewriting

```python
# pytest rewrites assert statements for detailed failure messages
# No need for assertEqual, assertTrue, etc.
def test_dict_comparison():
    result = {"name": "alice", "role": "admin"}
    expected = {"name": "alice", "role": "user"}
    assert result == expected
    # Failure shows:
    #   E  AssertionError: assert {'name': 'alice', 'role': 'admin'}
    #   E    == {'name': 'alice', 'role': 'user'}
    #   E  Differing items:
    #   E  {'role': 'admin'} != {'role': 'user'}
```

## Fixtures

### Basic fixtures

```python
import pytest

@pytest.fixture
def db_connection():
    conn = create_connection()
    yield conn               # yield = setup/teardown split
    conn.close()

def test_query(db_connection):
    result = db_connection.execute("SELECT 1")
    assert result == 1
```

### Fixture scopes

```python
@pytest.fixture(scope="function")   # default, per-test
def fresh_db(): ...

@pytest.fixture(scope="class")      # shared within test class
def class_db(): ...

@pytest.fixture(scope="module")     # shared within module
def module_db(): ...

@pytest.fixture(scope="package")    # shared within package
def package_db(): ...

@pytest.fixture(scope="session")    # shared across entire run
def session_db(): ...
```

### Autouse fixtures

```python
@pytest.fixture(autouse=True)
def reset_environment():
    """Runs before every test automatically."""
    os.environ.clear()
    yield
    os.environ.update(original_env)
```

### conftest.py

```python
# conftest.py — fixtures available to all tests in directory and below
# No import needed — pytest discovers automatically

# tests/conftest.py
@pytest.fixture
def api_client(db_connection):
    return TestClient(app, db=db_connection)

# tests/integration/conftest.py  — overrides or extends parent
@pytest.fixture
def api_client(db_connection):
    return TestClient(app, db=db_connection, auth=True)
```

## Parametrize

### Basic parametrize

```python
@pytest.mark.parametrize("input,expected", [
    ("hello", 5),
    ("", 0),
    ("world", 5),
])
def test_string_length(input, expected):
    assert len(input) == expected
```

### Multiple parametrize (cartesian product)

```python
@pytest.mark.parametrize("x", [1, 2])
@pytest.mark.parametrize("y", [10, 20])
def test_multiply(x, y):
    # runs 4 combinations: (1,10), (1,20), (2,10), (2,20)
    assert x * y > 0
```

### Parametrize with IDs

```python
@pytest.mark.parametrize("user,status", [
    pytest.param({"role": "admin"}, 200, id="admin-allowed"),
    pytest.param({"role": "guest"}, 403, id="guest-denied"),
    pytest.param(None, 401, id="anonymous"),
])
def test_access(user, status):
    assert check_access(user) == status
```

## Markers

### Built-in markers

```python
@pytest.mark.skip(reason="Not implemented yet")
def test_future_feature(): ...

@pytest.mark.skipif(sys.platform == "win32", reason="Unix only")
def test_unix_permissions(): ...

@pytest.mark.xfail(reason="Known bug #1234")
def test_known_issue(): ...

@pytest.mark.xfail(strict=True)  # must fail, or test fails
def test_must_fail(): ...
```

### Custom markers

```python
# pytest.ini or pyproject.toml
# [tool.pytest.ini_options]
# markers = [
#     "slow: marks tests as slow",
#     "integration: integration tests",
# ]

@pytest.mark.slow
def test_big_dataset(): ...

@pytest.mark.integration
def test_api_roundtrip(): ...
```

```bash
pytest -m slow                      # run only slow tests
pytest -m "not slow"                # skip slow tests
pytest -m "integration and not slow"
pytest --strict-markers             # fail on unknown markers
```

## Built-in Fixtures

### tmp_path and tmp_path_factory

```python
def test_file_write(tmp_path):
    p = tmp_path / "output.txt"
    p.write_text("hello")
    assert p.read_text() == "hello"

@pytest.fixture(scope="session")
def shared_data_dir(tmp_path_factory):
    return tmp_path_factory.mktemp("data")
```

### capsys and capfd

```python
def test_stdout(capsys):
    print("hello")
    captured = capsys.readouterr()
    assert captured.out == "hello\n"
    assert captured.err == ""

def test_fd_capture(capfd):
    os.write(1, b"raw fd output")
    captured = capfd.readouterr()
    assert "raw fd" in captured.out
```

### monkeypatch

```python
def test_env_var(monkeypatch):
    monkeypatch.setenv("API_KEY", "test-key-123")
    assert os.environ["API_KEY"] == "test-key-123"

def test_mock_function(monkeypatch):
    monkeypatch.setattr("myapp.client.fetch", lambda url: {"ok": True})
    result = myapp.client.fetch("https://api.example.com")
    assert result == {"ok": True}

def test_delete_attr(monkeypatch):
    monkeypatch.delattr("os.remove")
    # os.remove no longer exists in this test
```

## Plugins

### pytest-cov (coverage)

```bash
pip install pytest-cov
pytest --cov=myapp                  # coverage for myapp package
pytest --cov=myapp --cov-report=html   # HTML report
pytest --cov=myapp --cov-report=term-missing  # show missed lines
pytest --cov=myapp --cov-fail-under=80  # fail if < 80%
```

### pytest-mock

```python
def test_with_mocker(mocker):
    mock_send = mocker.patch("myapp.email.send")
    process_order(order)
    mock_send.assert_called_once_with(
        to="user@example.com",
        subject="Order Confirmation"
    )

def test_spy(mocker):
    spy = mocker.spy(myapp.cache, "get")
    result = myapp.cache.get("key")
    spy.assert_called_once_with("key")
    # original function still runs
```

## Configuration

### pyproject.toml

```toml
[tool.pytest.ini_options]
testpaths = ["tests"]
python_files = ["test_*.py", "*_test.py"]
python_functions = ["test_*"]
python_classes = ["Test*"]
addopts = "-ra -q --strict-markers"
markers = [
    "slow: marks tests as slow",
    "integration: integration tests requiring external services",
]
filterwarnings = [
    "error",
    "ignore::DeprecationWarning:third_party.*",
]
```

### conftest.py hooks

```python
def pytest_collection_modifyitems(items):
    """Reorder tests: fast first, slow last."""
    slow = [i for i in items if i.get_closest_marker("slow")]
    fast = [i for i in items if not i.get_closest_marker("slow")]
    items[:] = fast + slow

def pytest_addoption(parser):
    parser.addoption("--runslow", action="store_true", default=False)

def pytest_configure(config):
    config.addinivalue_line("markers", "slow: slow test")
```

## Tips

- Use `assert` directly -- pytest rewrites assertions for rich diffs, no need for `self.assertEqual`
- Put shared fixtures in `conftest.py` at the appropriate directory level for automatic discovery
- Use `scope="session"` for expensive fixtures like database connections or Docker containers
- Combine `-x --pdb` to stop at first failure and inspect state interactively
- Use `--lf` (last failed) during development to re-run only broken tests
- Use `pytest.ini_options` in `pyproject.toml` to avoid extra config files
- Register custom markers with `--strict-markers` to catch typos in marker names
- Use `tmp_path` instead of the legacy `tmpdir` fixture for `pathlib.Path` objects
- Run `pytest --co` (collect only) to verify test discovery without executing
- Use `pytest-xdist` for parallel execution but be aware of shared state between workers
- Use `monkeypatch` over `unittest.mock.patch` for simpler environment and attribute mocking
- Use `pytest.approx()` for floating point comparisons instead of rounding hacks

## See Also

- unittest
- coverage
- tox
- nox
- hypothesis

## References

- [pytest Official Documentation](https://docs.pytest.org/en/stable/)
- [pytest Fixture Reference](https://docs.pytest.org/en/stable/reference/fixtures.html)
- [pytest-xdist Documentation](https://pytest-xdist.readthedocs.io/en/stable/)
- [pytest-cov Documentation](https://pytest-cov.readthedocs.io/en/latest/)
- [pytest-mock Documentation](https://pytest-mock.readthedocs.io/en/latest/)
- [Real Python: Effective Python Testing With pytest](https://realpython.com/pytest-python-testing/)
