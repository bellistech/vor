# YAML (Data Serialization Format)

Human-readable data serialization format used in config files, CI/CD, Kubernetes, and Ansible.

## Scalars

### Strings

```bash
# plain: hello world
# single_quoted: 'no escapes, literal \n'
# double_quoted: "supports escapes \n \t"
# colon_in_value: "key: must quote if colon follows space"
# boolean_safe: "yes"              # quote to prevent YAML treating as boolean
# number_safe: "1.0"               # quote to keep as string
```

### Numbers

```bash
# integer: 42
# negative: -17
# float: 3.14
# scientific: 6.022e23
# hex: 0xFF
# octal: 0o77
# infinity: .inf
# not_a_number: .nan
```

### Booleans and null

```bash
# enabled: true                    # also: True, TRUE, yes, Yes, YES
# disabled: false                  # also: False, FALSE, no, No, NO
# empty: null                      # also: ~, or omit the value
```

## Sequences (Lists)

### Block style

```bash
# fruits:
#   - apple
#   - banana
#   - cherry
```

### Flow style (inline)

```bash
# fruits: [apple, banana, cherry]
```

### List of objects

```bash
# users:
#   - name: Alice
#     email: alice@example.com
#   - name: Bob
#     email: bob@example.com
```

## Mappings (Dictionaries)

### Block style

```bash
# database:
#   host: localhost
#   port: 5432
#   name: mydb
```

### Flow style (inline)

```bash
# database: {host: localhost, port: 5432, name: mydb}
```

### Nested

```bash
# server:
#   production:
#     host: prod.example.com
#     port: 443
#   staging:
#     host: staging.example.com
#     port: 8443
```

## Multi-Line Strings

### Literal block (|) preserves newlines

```bash
# description: |
#   This is line 1.
#   This is line 2.
#
#   This is line 4 (blank line preserved).
# # Result: "This is line 1.\nThis is line 2.\n\nThis is line 4 (blank line preserved).\n"
```

### Folded block (>) joins lines

```bash
# description: >
#   This is a long
#   paragraph that will
#   be joined into one line.
# # Result: "This is a long paragraph that will be joined into one line.\n"
```

### Chomping indicators

```bash
# strip: |-               # no trailing newline
#   hello
# keep: |+                # keep all trailing newlines
#   hello
#
#
# clip: |                 # single trailing newline (default)
#   hello
```

### Indentation indicator

```bash
# code: |2                # content starts at 2-space indent
#   if true:
#     print("hello")
```

## Anchors & Aliases

### Define and reuse

```bash
# defaults: &defaults
#   adapter: postgres
#   host: localhost
#   port: 5432
#
# development:
#   <<: *defaults          # merge all keys from defaults
#   database: dev_db
#
# production:
#   <<: *defaults
#   host: prod.example.com
#   database: prod_db
```

### Simple anchor/alias

```bash
# admin_email: &admin alice@example.com
# notifications:
#   error_to: *admin       # resolves to alice@example.com
```

## Merge Keys

### Merge multiple anchors

```bash
# base: &base
#   timeout: 30
#   retries: 3
#
# logging: &logging
#   level: info
#   format: json
#
# service:
#   <<: [*base, *logging]  # merge both
#   name: api
```

Later keys override merged keys:

```bash
# service:
#   <<: *base
#   timeout: 60            # overrides base timeout of 30
```

## Tags

### Explicit typing

```bash
# not_a_bool: !!str yes           # force string "yes"
# not_a_float: !!str 1.0          # force string "1.0"
# binary: !!binary |
#   R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7
```

## Multiple Documents

```bash
# ---
# name: doc1
# ---
# name: doc2
# ...                              # optional end-of-document marker
```

## Common Patterns

### Environment variables in Docker Compose

```bash
# services:
#   app:
#     image: myapp:latest
#     environment:
#       - DATABASE_URL=postgres://localhost/mydb
#       - REDIS_URL=redis://localhost:6379
#     ports:
#       - "8080:8080"
#     volumes:
#       - ./data:/app/data
```

### Kubernetes resource

```bash
# apiVersion: apps/v1
# kind: Deployment
# metadata:
#   name: myapp
#   labels:
#     app: myapp
# spec:
#   replicas: 3
#   selector:
#     matchLabels:
#       app: myapp
```

## Tips

- YAML uses spaces for indentation, never tabs. Two spaces is the convention.
- Unquoted `yes`, `no`, `true`, `false`, `on`, `off` are booleans. Quote them if you mean strings.
- Unquoted `1.0` is a float, `1` is an integer. Quote for strings: `"1.0"`.
- The Norway problem: `NO` (country code) parses as boolean false. Always quote country codes.
- Anchors (`&`) and aliases (`*`) reduce repetition but hurt readability. Use sparingly.
- `|` (literal) keeps newlines. `>` (folded) joins lines. Add `-` to strip trailing newline.
- Multi-document files use `---` as separator. Tools like `kubectl apply -f` handle them natively.
- Use a YAML linter (`yamllint`) to catch indentation errors before they reach production.

## References

- [YAML 1.2.2 Specification](https://yaml.org/spec/1.2.2/) -- full language specification
- [YAML 1.1 Specification](https://yaml.org/spec/1.1/) -- older spec still used by many parsers
- [YAML Schema Index](https://yaml.org/type/) -- core, JSON, and failsafe schema type definitions
- [YAML Ain't Markup Language (yaml.org)](https://yaml.org/) -- official site, links, and resources
- [yamllint](https://yamllint.readthedocs.io/) -- linter for YAML files (configurable rules)
- [yq (Go)](https://github.com/mikefarah/yq) -- jq-like YAML/JSON/XML processor
- [PyYAML Documentation](https://pyyaml.org/wiki/PyYAMLDocumentation) -- Python YAML library
- [YAML Multiline Strings](https://yaml-multiline.info/) -- interactive guide to block scalars and flow scalars
- [StrictYAML](https://hitchdev.com/strictyaml/) -- type-safe YAML subset that disables dangerous features
