# jq (JSON Processor)

Command-line tool for parsing, filtering, and transforming JSON data.

## Basic Usage

### Pretty print

```bash
jq . data.json
cat data.json | jq .
curl -s https://api.example.com/users | jq .
```

### Raw output (no quotes)

```bash
jq -r '.name' data.json
```

### Compact output

```bash
jq -c . data.json
```

## Select Fields

### Object field access

```bash
jq '.name' data.json                     # "Alice"
jq '.address.city' data.json             # nested field
jq '.users[0].name' data.json            # first array element
jq '.users[-1]' data.json                # last element
jq '.users[2:5]' data.json               # slice (index 2,3,4)
```

### Multiple fields

```bash
jq '{name: .name, email: .email}' data.json
jq '{name, email}' data.json              # shorthand (same keys)
```

### Optional field (no error if missing)

```bash
jq '.missing // "default"' data.json      # alternative operator
jq '.foo?' data.json                      # suppress errors
```

## Filters

### Select with condition

```bash
jq '.users[] | select(.age > 30)' data.json
jq '.users[] | select(.name == "Alice")' data.json
jq '.users[] | select(.tags | contains(["admin"]))' data.json
jq '.users[] | select(.email | test("@example\\.com$"))' data.json
jq '.users[] | select(.active and .age >= 18)' data.json
jq '.users[] | select(.name | startswith("A"))' data.json
```

## Map & Transform

### Map over arrays

```bash
jq '[.users[] | .name]' data.json                # extract names
jq '.users | map(.name)' data.json                # same result
jq '.users | map({name, email})' data.json        # project fields
jq '.users | map(. + {status: "active"})' data.json  # add field
jq '.users | map(del(.password))' data.json       # remove field
jq '[.[] | .price * .quantity]' data.json         # compute values
```

### Map keys and values

```bash
jq '.config | to_entries | map(.key)' data.json
jq '.config | keys' data.json                     # sorted keys
jq '.config | values' data.json
jq '.config | to_entries | map({(.key): (.value | tostring)}) | add' data.json
```

## Keys, Values, Entries

```bash
jq 'keys' data.json                               # top-level keys
jq '.users[0] | keys' data.json                   # object keys
jq '.config | values' data.json
jq '.config | has("timeout")' data.json
jq '.config | length' data.json                   # number of keys
jq '.users | length' data.json                    # array length
jq '.name | length' data.json                     # string length
```

## Type Checking

```bash
jq '.[] | type' data.json                          # "string", "number", etc.
jq '.[] | select(type == "string")' data.json
jq 'map(select(. != null))' data.json              # remove nulls
```

## Grouping & Sorting

### Sort

```bash
jq '.users | sort_by(.name)' data.json
jq '.users | sort_by(.age) | reverse' data.json
jq '[.users[] | .name] | sort' data.json           # sort strings
jq '.users | sort_by(.created_at) | last' data.json
```

### Group

```bash
jq '.users | group_by(.department)' data.json
jq '.users | group_by(.department) | map({dept: .[0].department, count: length})' data.json
```

### Unique

```bash
jq '[.users[].department] | unique' data.json
jq '.users | unique_by(.email)' data.json
```

## Aggregation

```bash
jq '.prices | add' data.json                       # sum
jq '.prices | add / length' data.json              # average
jq '.prices | min' data.json
jq '.prices | max' data.json
jq '[.users[] | .age] | add / length' data.json    # average age
```

## String Interpolation

```bash
jq -r '.users[] | "\(.name) <\(.email)>"' data.json
jq -r '.users[] | "User: \(.name), Age: \(.age)"' data.json
```

## Conditionals

```bash
jq '.users[] | if .age >= 18 then "adult" else "minor" end' data.json
jq '.users | map(if .active then . else . + {status: "inactive"} end)' data.json
```

## Slurp & Raw

### Slurp (combine multiple JSON inputs into array)

```bash
jq -s '.' file1.json file2.json            # array of two objects
jq -s 'add' file1.json file2.json          # merge objects
jq -s 'map(.name)' file1.json file2.json
```

### Raw input (non-JSON lines)

```bash
jq -R . <<< "plain text"                   # wrap text as JSON string
jq -R -s 'split("\n") | map(select(. != ""))' logfile.txt  # lines to array
```

### Raw output

```bash
jq -r '.name' data.json                    # Alice (no quotes)
jq -r '.users[] | .name' data.json         # one name per line
```

## Output Formats

### CSV

```bash
jq -r '.users[] | [.name, .email, .age] | @csv' data.json
```

### TSV

```bash
jq -r '.users[] | [.name, .email] | @tsv' data.json
```

### URI encoding

```bash
jq -r '.query | @uri' data.json
```

### HTML encoding

```bash
jq -r '.content | @html' data.json
```

### Base64

```bash
jq -r '.data | @base64' data.json
jq -r '.encoded | @base64d' data.json      # decode
```

## Advanced Patterns

### Flatten nested arrays

```bash
jq '[.[][] ]' data.json                     # one level
jq '[.. | numbers]' data.json               # all numbers recursively
```

### Recursive descent

```bash
jq '.. | .name? // empty' data.json        # all "name" fields at any depth
jq '[.. | strings]' data.json               # all string values
```

### Update in place

```bash
jq '.users[0].name = "Alice Smith"' data.json
jq '.config.timeout = 60' data.json
jq 'del(.users[] | select(.active == false))' data.json
```

### Reduce

```bash
jq '.users | reduce .[] as $u ({}; . + {($u.name): $u.email})' data.json
```

### Define variables

```bash
jq --arg name "Alice" '.users[] | select(.name == $name)' data.json
jq --argjson age 30 '.users[] | select(.age > $age)' data.json
```

## Tips

- `jq -r` (raw output) strips quotes from strings. Essential for shell scripting.
- `jq -e` sets exit code 1 if the output is `false` or `null`. Useful in conditionals.
- `//` is the alternative operator: `.missing // "default"` returns the default if the field is null or missing.
- `?` suppresses errors: `.foo?` returns nothing instead of erroring if `.foo` does not exist on the input.
- `-s` (slurp) reads all inputs into a single array. Use it when processing multiple JSON files.
- `@csv` and `@tsv` require arrays of values, not objects. Project fields first: `[.name, .age] | @csv`.
- `--arg` passes a string variable. `--argjson` passes a JSON value (number, bool, object).
- Use `env.VAR_NAME` to access environment variables inside jq expressions.
