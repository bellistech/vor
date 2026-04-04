# JSON (Data Interchange Format)

Lightweight text-based data format used for APIs, config files, and data exchange.

## Syntax

### Data types

```bash
# String:  "hello world"           (double quotes only)
# Number:  42, 3.14, -1, 2.5e10   (no hex, no leading zeros)
# Boolean: true, false             (lowercase only)
# Null:    null
# Object:  {"key": "value"}       (keys must be quoted strings)
# Array:   [1, 2, 3]
```

### Object (key-value pairs)

```bash
# {
#   "name": "Alice",
#   "age": 30,
#   "active": true,
#   "address": {
#     "city": "Portland",
#     "zip": "97201"
#   },
#   "tags": ["admin", "user"]
# }
```

### Array

```bash
# [
#   {"id": 1, "name": "Alice"},
#   {"id": 2, "name": "Bob"}
# ]
```

### Nested structures

```bash
# {
#   "users": [
#     {
#       "name": "Alice",
#       "roles": ["admin"],
#       "settings": {
#         "theme": "dark",
#         "notifications": true
#       }
#     }
#   ]
# }
```

## Common Patterns

### API response envelope

```bash
# {
#   "data": [...],
#   "meta": {
#     "page": 1,
#     "per_page": 20,
#     "total": 150
#   }
# }
```

### Error response

```bash
# {
#   "error": {
#     "code": 404,
#     "message": "User not found",
#     "details": []
#   }
# }
```

### Config file (package.json style)

```bash
# {
#   "name": "myapp",
#   "version": "1.2.0",
#   "scripts": {
#     "build": "go build -o app .",
#     "test": "go test ./..."
#   },
#   "dependencies": {
#     "express": "^4.18.0"
#   }
# }
```

## JSON Schema

### Basic schema

```bash
# {
#   "$schema": "https://json-schema.org/draft/2020-12/schema",
#   "type": "object",
#   "required": ["name", "email"],
#   "properties": {
#     "name": {
#       "type": "string",
#       "minLength": 1
#     },
#     "email": {
#       "type": "string",
#       "format": "email"
#     },
#     "age": {
#       "type": "integer",
#       "minimum": 0,
#       "maximum": 150
#     },
#     "tags": {
#       "type": "array",
#       "items": {"type": "string"},
#       "uniqueItems": true
#     }
#   }
# }
```

### Common schema types and formats

```bash
# "type": "string"   | "number" | "integer" | "boolean" | "null" | "array" | "object"
# "format": "email"  | "uri" | "date" | "date-time" | "ipv4" | "ipv6" | "uuid"
# "enum": ["draft", "published", "archived"]
# "pattern": "^[A-Z]{2}[0-9]{4}$"
# "oneOf": [{...}, {...}]
# "anyOf": [{...}, {...}]
# "$ref": "#/$defs/Address"
```

## Validation

### Validate with python

```bash
python3 -c "import json, sys; json.load(sys.stdin)" < data.json && echo "Valid"
```

### Validate with jq

```bash
jq empty data.json    # exits 0 if valid, non-zero with error message
```

### Validate with node

```bash
node -e "JSON.parse(require('fs').readFileSync('data.json'))"
```

## Pretty Print

### With jq

```bash
jq . data.json
jq . data.json > formatted.json
```

### With python

```bash
python3 -m json.tool data.json
python3 -m json.tool --sort-keys data.json
cat data.json | python3 -m json.tool
```

### With node

```bash
node -e "console.log(JSON.stringify(JSON.parse(require('fs').readFileSync('data.json','utf8')),null,2))"
```

## Minify

### With jq

```bash
jq -c . data.json                        # compact output
jq -c . data.json > minified.json
```

### With python

```bash
python3 -c "import json,sys; json.dump(json.load(sys.stdin),sys.stdout,separators=(',',':'))" < data.json
```

## Command-Line Manipulation

### Extract a field

```bash
jq '.name' data.json
jq -r '.name' data.json                  # raw output (no quotes)
```

### Merge JSON files

```bash
jq -s '.[0] * .[1]' base.json override.json    # shallow merge
```

### Convert formats

```bash
# JSON to YAML
python3 -c "import json,yaml,sys; yaml.dump(json.load(sys.stdin),sys.stdout)" < data.json
# CSV to JSON
python3 -c "import csv,json,sys; print(json.dumps(list(csv.DictReader(sys.stdin))))" < data.csv
```

## Tips

- JSON keys must be double-quoted strings. Single quotes are invalid.
- No trailing commas allowed: `{"a": 1,}` is a syntax error.
- No comments in standard JSON. Use JSON5 or JSONC (VS Code) for commented config files.
- Numbers cannot have leading zeros: `007` is invalid. Use `7` or `"007"` (string).
- `null` is valid JSON. An empty response body is not.
- `jq empty file.json` is the fastest way to validate JSON from the command line.
- JSON does not distinguish between integers and floats. `1` and `1.0` may parse differently depending on the language.
- For large JSON files, streaming parsers (`jq --stream`, Python `ijson`) avoid loading everything into memory.

## See Also

- jq, yaml, toml, xml, javascript, python

## References

- [JSON Specification (json.org)](https://www.json.org/) -- grammar, syntax diagrams, and parser list
- [RFC 8259 -- The JSON Data Interchange Format](https://www.rfc-editor.org/rfc/rfc8259) -- current JSON standard
- [ECMA-404 -- The JSON Data Interchange Syntax](https://ecma-international.org/publications-and-standards/standards/ecma-404/) -- ISO/IEC 21778:2017
- [RFC 6901 -- JSON Pointer](https://www.rfc-editor.org/rfc/rfc6901) -- path syntax for addressing values
- [RFC 6902 -- JSON Patch](https://www.rfc-editor.org/rfc/rfc6902) -- operations for modifying JSON documents
- [RFC 7396 -- JSON Merge Patch](https://www.rfc-editor.org/rfc/rfc7396) -- simplified patching format
- [JSON Schema](https://json-schema.org/) -- vocabulary for validating JSON structure
- [jq Manual](https://jqlang.github.io/jq/manual/) -- command-line JSON processor
- [JSON Lines](https://jsonlines.org/) -- newline-delimited JSON for streaming and logs
- [JSONPath Specification (RFC 9535)](https://www.rfc-editor.org/rfc/rfc9535) -- query language for JSON
