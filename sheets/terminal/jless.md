# jless (interactive JSON/YAML viewer)

An interactive command-line JSON and YAML viewer that provides vim-style navigation, search, collapse and expand controls, and a data mode for exploring deeply nested structures without losing context in the terminal.

## Basic Usage

### Opening Files

```bash
# Open a JSON file
jless data.json

# Open a YAML file
jless config.yaml

# Read from stdin
curl -s https://api.example.com/data | jless

# Pipe command output
kubectl get pods -o json | jless
docker inspect container_id | jless
aws ec2 describe-instances | jless
terraform show -json | jless
```

### Modes

```bash
# Default mode: line mode (shows formatted JSON with indentation)
jless data.json

# Data mode: shows key-value pairs in a tree view
jless --mode data data.json
# Or press 'd' to toggle while viewing

# Line mode shows raw JSON formatting
# Data mode shows structured key: value pairs
```

## Navigation

### Basic Movement (Vim-Style)

```
j / Down       Move down one line
k / Up         Move up one line
h / Left       Collapse node / move to parent
l / Right      Expand node / move into child
```

### Fast Movement

```
J              Move to next sibling
K              Move to previous sibling
H              Move to parent node
g              Jump to top of document
G              Jump to bottom of document
Ctrl-d         Page down (half screen)
Ctrl-u         Page up (half screen)
Ctrl-f         Page down (full screen)
Ctrl-b         Page up (full screen)
```

### Jumping to Elements

```
w              Move to next key in object
b              Move to previous key in object
0              Move to first element in array/object
$              Move to last element in array/object
```

## Collapse and Expand

### Folding Controls

```
h              Collapse current node (when expanded)
l              Expand current node (when collapsed)
Space          Toggle collapse/expand of current node
c              Collapse all children of current node
e              Expand all children of current node
```

### Depth-Based Folding

```
1              Collapse to depth 1 (top-level keys only)
2              Collapse to depth 2
3              Collapse to depth 3
...
9              Collapse to depth 9
```

## Search

### Finding Values and Keys

```
/ pattern      Forward search for pattern in keys and values
? pattern      Backward search for pattern
n              Next match
N              Previous match
```

### Search Tips

```
# Search for a key name
/ "username"

# Search for a value
/ "error"

# Search for a number
/ 42

# Search for nested key paths
# Navigate to result, use 'p' to see full path
```

## Copying and Output

### Yanking (Copying)

```
yy             Copy current line to clipboard
yp             Copy path to current node
yv             Copy value of current node
yb             Copy current node and all children (pretty-printed)
```

### Path Display

```
p              Print/display path to current node (e.g., .data[0].name)
```

## Data Types and Display

### Type Indicators

```
# jless displays type information:
# Strings:   "hello"        (green, quoted)
# Numbers:   42, 3.14       (cyan)
# Booleans:  true, false    (yellow)
# Null:      null           (gray/dim)
# Arrays:    [...] (N)      (shows element count when collapsed)
# Objects:   {...} (N)      (shows key count when collapsed)
```

### Array and Object Summaries

```
# When collapsed, containers show their size:
# "users": [...] (150)     — array with 150 elements
# "config": {...} (12)     — object with 12 keys

# This helps you understand structure without expanding everything
```

## Command Line Options

### Startup Options

```bash
# Specify initial mode
jless --mode line data.json        # default line mode
jless --mode data data.json        # data/tree mode

# YAML input
jless config.yaml                  # auto-detected
jless --yaml config.txt            # force YAML parsing

# Scrolloff (keep N lines visible above/below cursor)
jless --scrolloff 5 data.json
```

## Practical Workflows

### API Response Exploration

```bash
# Explore REST API response
curl -s https://api.github.com/repos/sharkdp/bat | jless

# Explore GraphQL response
curl -s -X POST https://api.example.com/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ users { name } }"}' | jless

# Explore Kubernetes resources
kubectl get deployment -o json | jless
kubectl get configmap myconfig -o json | jless
```

### Configuration File Navigation

```bash
# Navigate complex configs
jless tsconfig.json
jless package.json
jless docker-compose.yaml

# Explore Terraform state
terraform show -json | jless

# Explore Ansible inventory
ansible-inventory --list | jless
```

### Log and Data Analysis

```bash
# View structured logs
cat app.log | jq -s '.' | jless

# Parse and view newline-delimited JSON
cat events.jsonl | jq -s '.' | jless

# Explore large data files
jless database-dump.json
# Use 1/2/3 to collapse to manageable depth
# Use / to search for specific records
```

## Keyboard Reference

### Quick Reference Table

```
Navigation:  j/k (up/down)  h/l (collapse/expand)  J/K (siblings)
Jumping:     g (top)  G (bottom)  w/b (next/prev key)
Folding:     Space (toggle)  c/e (collapse/expand all)  1-9 (depth)
Search:      / (forward)  ? (backward)  n/N (next/prev match)
Copy:        yy (line)  yp (path)  yv (value)  yb (subtree)
Mode:        d (toggle data/line mode)
Display:     p (show path)
Exit:        q (quit)
```

## Tips

- Press `d` to switch to data mode when you want a clean tree view without JSON syntax noise (braces, commas, quotes).
- Use numeric keys `1`-`9` to instantly collapse to a specific depth -- `1` gives you top-level overview of any structure.
- `yp` (yank path) copies the dotted path like `.data[0].name` -- paste it directly into jq or programming code.
- Pipe any JSON-producing command into jless for exploration: `curl -s url | jless` beats scrolling through raw output.
- Use `J` and `K` to skip between sibling elements -- much faster than `j`/`k` when arrays have many items.
- `Space` to toggle expand/collapse is the fastest way to drill into and back out of nested structures.
- jless auto-detects YAML files by extension; for stdin YAML, use `--yaml` to force the parser.
- For large arrays, collapse with `c`, then expand individual items with `l` to avoid rendering thousands of elements.
- The `p` command shows the full path to the cursor, helping you understand exactly where you are in deep nesting.
- Combine with jq for preprocessing: `cat data.json | jq '.results' | jless` to start viewing at a specific subtree.
- jless handles files up to hundreds of megabytes -- it streams and indexes on load rather than holding everything in memory.

## See Also

- jq, bat, fzf, json, yaml, curl

## References

- [jless GitHub Repository](https://github.com/PaulJuliworthy/jless)
- [jless Website](https://jless.io)
- [jless Documentation — Keybindings](https://jless.io/user-guide.html)
- [JSON Specification (RFC 8259)](https://datatracker.ietf.org/doc/html/rfc8259)
- [YAML Specification](https://yaml.org/spec/1.2.2/)
- [jq Manual](https://jqlang.github.io/jq/manual/)
