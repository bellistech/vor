# TOML (Tom's Obvious Minimal Language)

Minimal configuration format that maps clearly to a hash table, used by Cargo, pyproject, and Hugo.

## Tables

### Basic table (section)

```bash
# [database]
# host = "localhost"
# port = 5432
# name = "mydb"
```

### Nested tables

```bash
# [server]
# host = "0.0.0.0"
#
# [server.tls]
# cert = "/etc/certs/cert.pem"
# key = "/etc/certs/key.pem"
```

### Super-table shorthand (dotted)

```bash
# [server.tls]
# cert = "/etc/certs/cert.pem"
# # is equivalent to:
# [server]
# [server.tls]
# cert = "/etc/certs/cert.pem"
```

## Dotted Keys

```bash
# name = "Alice"
# physical.color = "blue"
# physical.shape = "round"
# # equivalent to:
# [physical]
# color = "blue"
# shape = "round"
```

## Strings

### Basic string (double quotes, escapes)

```bash
# str = "hello\nworld"
# path = "C:\\Users\\alice"
```

### Literal string (single quotes, no escapes)

```bash
# regex = '\d+\.\d+'
# winpath = 'C:\Users\alice'
```

### Multi-line basic string

```bash
# bio = """
# This is a
# multi-line string.
# Escapes work: \n \t"""
```

### Multi-line literal string

```bash
# regex = '''
# \d{1,3}
# \.\d{1,3}
# '''
```

## Numbers

```bash
# integer = 42
# negative = -17
# hex = 0xDEADBEEF
# octal = 0o755
# binary = 0b11010110
# float = 3.14
# scientific = 5e+22
# infinity = inf
# not_a_number = nan
# with_separator = 1_000_000      # underscores for readability
```

## Booleans

```bash
# enabled = true
# debug = false
```

## Datetime

```bash
# odt = 1979-05-27T07:32:00Z                  # offset datetime (UTC)
# odt2 = 1979-05-27T07:32:00-08:00            # with timezone
# ldt = 1979-05-27T07:32:00                   # local datetime (no tz)
# ld = 1979-05-27                              # local date
# lt = 07:32:00                                # local time
```

## Arrays

### Basic arrays

```bash
# ports = [8080, 8081, 8082]
# names = ["Alice", "Bob", "Carol"]
# mixed = ["hello", 42, true]                  # mixed types allowed in TOML v1.0
```

### Multi-line array

```bash
# hosts = [
#     "web1.example.com",
#     "web2.example.com",
#     "web3.example.com",     # trailing comma OK
# ]
```

## Array of Tables

### Define repeated sections

```bash
# [[servers]]
# name = "web1"
# ip = "10.0.0.1"
# role = "frontend"
#
# [[servers]]
# name = "db1"
# ip = "10.0.0.2"
# role = "database"
#
# # Parses to: servers = [{name="web1",...}, {name="db1",...}]
```

### Nested array of tables

```bash
# [[fruits]]
# name = "apple"
#
# [[fruits.varieties]]
# name = "granny smith"
#
# [[fruits.varieties]]
# name = "fuji"
#
# [[fruits]]
# name = "banana"
#
# [[fruits.varieties]]
# name = "cavendish"
```

## Inline Tables

```bash
# point = {x = 1, y = 2}
# user = {name = "Alice", email = "alice@example.com"}
#
# # Inline tables must be on one line and cannot be extended later.
# # Use regular tables for anything complex.
```

## Common Patterns

### Cargo.toml (Rust)

```bash
# [package]
# name = "myapp"
# version = "0.1.0"
# edition = "2021"
#
# [dependencies]
# serde = { version = "1.0", features = ["derive"] }
# tokio = { version = "1", features = ["full"] }
#
# [dev-dependencies]
# criterion = "0.5"
#
# [[bin]]
# name = "myapp"
# path = "src/main.rs"
```

### pyproject.toml (Python)

```bash
# [project]
# name = "mypackage"
# version = "1.0.0"
# requires-python = ">=3.10"
# dependencies = [
#     "requests>=2.28",
#     "click>=8.0",
# ]
#
# [tool.ruff]
# line-length = 100
# target-version = "py310"
#
# [tool.pytest.ini_options]
# testpaths = ["tests"]
```

### Hugo config

```bash
# baseURL = "https://example.com"
# languageCode = "en-us"
# title = "My Site"
#
# [params]
# description = "A personal blog"
# author = "Alice"
#
# [[menus.main]]
# name = "Home"
# url = "/"
# weight = 1
#
# [[menus.main]]
# name = "About"
# url = "/about/"
# weight = 2
```

## Tips

- TOML maps directly to a hash table. Every key-value pair has an unambiguous type.
- `[[section]]` (double brackets) creates an array of tables. `[section]` creates a single table.
- Inline tables (`{key = "val"}`) cannot span multiple lines and cannot be extended after definition.
- Keys are case-sensitive: `Name` and `name` are different keys.
- Trailing commas in arrays are allowed. Trailing commas in inline tables are not (TOML v1.0).
- Dotted keys (`physical.color = "blue"`) are syntactic sugar for nested tables.
- TOML has native datetime support, unlike JSON and YAML.
- Use `taplo` or `toml-sort` for formatting and validating TOML files.
