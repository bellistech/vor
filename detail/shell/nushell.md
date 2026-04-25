# The Internals of Nushell — Structured Pipelines, Engine, and the Polars Plugin

> *Nushell is not a shell that mimics Unix with a coat of paint — it is a Rust-implemented data-language with a typed AST, a streaming engine, structured pipelines, closures, modules, and a plugin protocol over JSON-RPC. Where bash pipes raw bytes between forked processes, Nu pipes structured Values between in-process commands. This guide unpacks the engine, the type catalog, the pipeline streaming model, the Polars dataframe integration, and the migration realities for users coming from POSIX shells.*

---

## 1. Nushell Architecture

### Rust Implementation Stack

Nushell is implemented in roughly 200,000 lines of Rust, organised into a workspace of crates:

| Crate | Responsibility |
|:------|:---------------|
| `nu-cli` | The REPL, line editor (reedline), prompts, completions |
| `nu-parser` | Tokenizer, parser, span tracker, AST builder |
| `nu-engine` | The evaluator: blocks, closures, pipelines, scope |
| `nu-protocol` | The Value enum, PipelineData, ShellError, Span, Type |
| `nu-command` | The standard command set (≈800 built-ins) |
| `nu-cmd-lang` | Language-level commands (let, def, if, for) |
| `nu-cmd-extra` | Optional commands gated by `--features extra` |
| `nu-plugin` | Plugin host runtime (JSON-RPC over stdin/stdout) |
| `nu-plugin-protocol` | The wire format (PluginCall, PluginResponse) |
| `nu-system` | OS-level helpers: ps, df, mount |
| `nu-color-config` | ANSI colour theming |
| `reedline` | The line editor (a separate Anthropic-funded crate) |

Each crate is published independently to crates.io, which is why third-party plugins like `nu_plugin_polars` can depend on `nu-plugin` and `nu-protocol` as library dependencies.

### Parser → AST → Engine Pipeline

The four stages of a Nushell command are:

```
source bytes → lex (tokens) → parse (AST + types) → IR → evaluate (PipelineData)
```

1. **Lex** — `nu-parser/src/lex.rs` produces a `Vec<Token>` with span information. Whitespace, comments, and strings are recognised here. Unlike bash, the lexer is whitespace-sensitive only at command boundaries.
2. **Parse** — `nu-parser/src/parser.rs` walks tokens into an AST of `Expression` nodes, each tagged with a `Span` (start, end byte offsets) and a `Type`. The parser performs early type inference and emits diagnostic spans for use in `error make`.
3. **IR (since 0.94)** — A bytecode-like internal representation that the engine walks. Earlier versions interpreted the AST directly; the IR allows for more aggressive caching and a future JIT.
4. **Evaluate** — `nu-engine/src/eval.rs` walks the IR in a loop, threading `PipelineData` (an iterator over `Value`s, plus metadata) between stages.

```rust
// nu-protocol/src/ast/expression.rs (simplified)
pub struct Expression {
    pub expr: Expr,
    pub span: Span,
    pub ty: Type,
    pub custom_completion: Option<DeclId>,
}

pub enum Expr {
    Bool(bool),
    Int(i64),
    Float(f64),
    Binary(Vec<u8>),
    Range(Box<Range>),
    Var(VarId),
    VarDecl(VarId),
    Call(Box<Call>),
    Operator(Operator),
    BinaryOp(Box<Expression>, Box<Expression>, Box<Expression>),
    String(String),
    CellPath(CellPath),
    FullCellPath(Box<FullCellPath>),
    Filepath(String, bool), // path, quoted
    Closure(BlockId),
    Block(BlockId),
    List(Vec<Expression>),
    Table(Vec<Expression>, Vec<Vec<Expression>>),
    Record(Vec<RecordItem>),
    Subexpression(BlockId),
    // ... ~30 variants total
}
```

### The Block / Expression / Span / Value Type Catalog

The engine deals with four orthogonal concepts:

- **Span** — `(start: usize, end: usize)` byte offsets into the source for diagnostics.
- **Expression** — a typed AST node (above).
- **Block** — a parsed sequence of pipelines, identified by `BlockId`. The block table is a global flat vector owned by `EngineState`.
- **Value** — the runtime data; a tagged union (see Section 2).

A custom command's body is a `Block`. A closure is a `Block` plus a captured `Stack` snapshot (the variable bindings at the point of definition).

### EngineState and Stack

The engine separates *static* data (declared commands, parsed blocks, env vars at start) from *dynamic* data (current variable bindings):

```rust
pub struct EngineState {
    pub files: Vec<File>,         // source file table
    pub spans: Vec<Span>,          // span pool
    pub blocks: Vec<Block>,        // block pool
    pub decls: Vec<Box<dyn Command>>, // declared commands
    pub plugins: Vec<PluginIdentity>,
    pub env_vars: HashMap<String, Value>,
    pub config: Box<Config>,
}

pub struct Stack {
    pub vars: Vec<(VarId, Value)>,  // current variable bindings
    pub env_vars: Vec<HashMap<String, Value>>, // env scope chain
    pub recursion_count: u64,
    pub parent_stack: Option<Arc<Stack>>,  // for closures
}
```

`EngineState` is `Clone`-cheap (it's mostly behind `Arc`s); `Stack` is per-call. This separation is what makes commands re-entrant across threads when `par-each` is used.

---

## 2. The Type System

### The Value Enum

The runtime universe is the `Value` enum, defined in `nu-protocol/src/value/mod.rs`. Every datum a Nushell command sees is one of these variants:

| Variant | Underlying Rust | Example Literal |
|:--------|:----------------|:----------------|
| `Int` | `i64` | `42`, `0xff`, `0o77` |
| `Float` | `f64` | `3.14`, `1e9` |
| `Bool` | `bool` | `true`, `false` |
| `String` | `String` | `"hello"`, `'world'` |
| `Date` | `chrono::DateTime<FixedOffset>` | `2026-04-25T00:00:00` |
| `Duration` | `i64` (nanoseconds) | `1hr`, `30sec`, `5day` |
| `Filesize` | `i64` (bytes) | `1mb`, `512kb`, `2gib` |
| `Range` | `Box<Range>` | `1..10`, `0..<5`, `0..` |
| `List` | `Vec<Value>` | `[1 2 3]` |
| `Record` | `IndexMap<String, Value>` | `{a: 1, b: 2}` |
| `Closure` | `BlockId + Captures` | `{\|x\| $x * 2 }` |
| `CellPath` | `Vec<PathMember>` | `$.users.0.name` |
| `Binary` | `Vec<u8>` | `0x[de ad be ef]` |
| `CustomValue` | `Box<dyn CustomValue>` | dataframe handles, sqlite cursors |
| `Error` | `Box<ShellError>` | propagated errors |
| `Nothing` | `()` | `null`, empty pipeline |

Note that `Table` is **not** a separate variant — a table is the special case of `List<Record>` where every record shares a schema. This means there is no runtime distinction between an arbitrary list of records and a table; the printer detects uniform schemas and renders them as a grid.

```rust
// nu-protocol/src/value/mod.rs (simplified)
pub enum Value {
    Bool      { val: bool, internal_span: Span },
    Int       { val: i64,  internal_span: Span },
    Float     { val: f64,  internal_span: Span },
    Filesize  { val: i64,  internal_span: Span },
    Duration  { val: i64,  internal_span: Span },
    Date      { val: DateTime<FixedOffset>, internal_span: Span },
    Range     { val: Box<Range>, internal_span: Span },
    String    { val: String, internal_span: Span },
    Glob      { val: String, no_expand: bool, internal_span: Span },
    Record    { val: SharedCow<Record>, internal_span: Span },
    List      { vals: Vec<Value>, internal_span: Span },
    Closure   { val: Box<Closure>, internal_span: Span },
    Nothing   { internal_span: Span },
    Error     { error: Box<ShellError>, internal_span: Span },
    Binary    { val: Vec<u8>, internal_span: Span },
    CellPath  { val: CellPath, internal_span: Span },
    Custom    { val: Box<dyn CustomValue>, internal_span: Span },
}
```

Every variant carries a `Span` so that downstream diagnostic messages can point to exactly where in the source a value originated.

### Cell-Path Traversal

A cell path is the universal accessor — `$record.col1.col2`, `$list.0.name`, `$table.0.users.3`. The traversal algorithm in `nu-engine/src/eval.rs` handles each `PathMember`:

```rust
pub enum PathMember {
    String { val: String, span: Span, optional: bool },
    Int    { val: usize,  span: Span, optional: bool },
}

fn follow_cell_path(value: Value, members: &[PathMember]) -> Result<Value, ShellError> {
    let mut current = value;
    for member in members {
        current = match (current, member) {
            (Value::Record { val, .. }, PathMember::String { val: key, .. }) => {
                val.get(key).cloned().ok_or(ShellError::CantFindColumn { ... })?
            }
            (Value::List { vals, .. }, PathMember::Int { val: idx, .. }) => {
                vals.get(*idx).cloned().ok_or(ShellError::AccessBeyondEnd { ... })?
            }
            (Value::List { vals, .. }, PathMember::String { val: key, .. }) => {
                // map across each row → table column extraction
                Value::list(
                    vals.into_iter().map(|row| follow_one(row, member)).collect()?,
                    span,
                )
            }
            // ...
        };
    }
    Ok(current)
}
```

The third arm — **mapping a string key across a list** — is what makes `$users.name` extract the `name` column from a table. The optional flag (`?`) on a member suppresses errors and yields `null` instead.

```nu
let users = [{name: alice age: 30} {name: bob age: 25}]
$users.name              # → [alice bob]
$users.0.name            # → alice
$users.99?.name          # → null  (optional)
```

### `to text` vs `to nuon` vs Display

Nushell distinguishes three string representations of a Value:

1. **Display** — what `print` emits (formatted tables, ANSI colours, ellipsis truncation).
2. **`to text`** — a stripped, single-line string suitable for piping into external tools. Removes ANSI, table borders, headers.
3. **`to nuon`** — the canonical Nushell Object Notation, round-trippable via `from nuon`. Preserves types (durations, dates, filesizes) that JSON cannot.

```nu
5min                    # display: 5min
5min | to text          # → "5min"
5min | to nuon          # → "5min"
5min | to json          # → 300000000000  (loses unit)
{a: 1mb, b: 30sec} | to nuon  # → {a: 1MiB, b: 30sec}
{a: 1mb, b: 30sec} | to json  # → {"a":1048576,"b":30000000000}
```

Use `to nuon` for serialising data between Nu sessions. Use `to json` only when an external consumer demands JSON.

---

## 3. Structured Pipelines

### Pipelines Carry Typed Values

A pipeline `cmd1 | cmd2 | cmd3` is internally a `PipelineData` chain, not a byte stream. `PipelineData` is an enum:

```rust
pub enum PipelineData {
    Empty,
    Value(Value, Option<PipelineMetadata>),
    ListStream(ListStream, Option<PipelineMetadata>),
    ByteStream(ByteStream, Option<PipelineMetadata>),
}
```

- `Value` — a single in-memory value, used for results of small computations.
- `ListStream` — a lazy iterator over Values, used for streaming large lists (e.g., `ls`).
- `ByteStream` — a raw byte iterator from external commands, files, or HTTP.
- `Empty` — the unit pipeline (e.g., from `print`).

This means `ls | where size > 1mb | first 10` does **not** materialise the full directory listing — `ls` produces a `ListStream`, `where` filters lazily, and `first 10` short-circuits after 10 matches.

### The Canonical Composition

The four-command cascade that defines idiomatic Nu:

```nu
ls
| where type == file
| select name size modified
| sort-by modified --reverse
| first 5
```

Each command's signature is precisely declared:

| Command | Input | Output |
|:--------|:------|:-------|
| `ls` | nothing | `table<name: string, type: string, size: filesize, modified: datetime>` |
| `where` | `list<any>` or `table<R>` | filtered same |
| `select` | `record` or `table<R>` | projected subset |
| `sort-by` | `list<any>` | sorted same |
| `first` | `list<any>` | prefix of same |

The pipeline type checker confirms that each stage's output matches the next stage's input type. Type errors surface at parse time, not runtime.

### Back-Pressure and Streaming

Streams are pull-based — downstream stages drive upstream stages. `each` and `where` consume one item at a time:

```nu
open big.csv | from csv | where amount > 1000 | each { update name { str upcase } } | first 100
```

For a 1GB CSV, only the rows passing the `where` filter are materialised; once `first 100` gathers 100 matches, the upstream stream is dropped (the underlying file handle closed via `Drop`). This is conceptually equivalent to a Rust `Iterator` chain with `.take(100)`.

### Pipeline Metadata

`PipelineMetadata` is a side-channel that carries information like the original file path, the content type detected, or the data source:

```rust
pub struct PipelineMetadata {
    pub data_source: DataSource,
    pub content_type: Option<String>,
}

pub enum DataSource {
    Ls,
    HtmlThemes,
    FilePath(PathBuf),
    None,
}
```

Commands like `to csv` use this metadata to choose default delimiters, and `save` uses the file-path source to suggest an output name.

---

## 4. Tables — List of Records

### The Row-Major Representation

A table is internally a `Vec<Value::Record>`. Each row is its own record; column ordering is preserved per-row but tables conventionally share a schema.

```nu
[[name age]; [alice 30] [bob 25] [carol 28]]
```

Equivalent to:

```nu
[
  {name: alice, age: 30},
  {name: bob,   age: 25},
  {name: carol, age: 28},
]
```

Both literals produce `list<record<name: string, age: int>>` — a table.

### Cell Path as Universal Accessor

The cell path `$tbl.1.age` does the obvious: row 1, column `age`. But `$tbl.age` (no row index) extracts a column as a list:

```nu
let tbl = [[name age]; [alice 30] [bob 25]]
$tbl.age           # → [30 25]
$tbl.0             # → {name: alice, age: 30}
$tbl.0.name        # → alice
```

This is the key abstraction that makes `update`, `insert`, `reject`, and `move` work uniformly across rows, columns, and individual cells.

### Transpose and Pivot

`transpose` swaps rows and columns:

```nu
[[a b]; [1 2] [3 4]] | transpose key val1 val2
# → [[key val1 val2]; [a 1 3] [b 2 4]]
```

`pivot` (in `polars` plugin) does proper relational pivoting with aggregation:

```nu
$df | polars pivot --on year --index region --values sales --aggregate sum
```

### Comparison with Pandas DataFrame

| Feature | Nu Table (core) | Polars DataFrame |
|:--------|:----------------|:-----------------|
| Storage | row-major | column-major |
| Memory model | each cell is `Value` (24 bytes + heap) | dense typed columns (Arrow buffers) |
| Lazy eval | no | yes (`polars open --lazy`) |
| Query opt | none | predicate pushdown, projection pushdown |
| Capacity | ≤ ~1M rows comfortable | ≥ 100M rows on commodity hardware |

For interactive ad-hoc work on small data, the core table is faster (no plugin RPC overhead). For analytical workloads, Polars is 10–50x faster.

---

## 5. Closures and Blocks

### Pipe-Bound Closures

A closure is `{ |args| body }`. The pipe-delimited args are required; the body is a sequence of pipelines.

```nu
let double = {|x| $x * 2 }
do $double 21              # → 42

[1 2 3] | each {|n| $n + 1 } | math sum   # → 9
```

### The `$in` Implicit Input

Inside a closure (or block body), `$in` is the current pipeline input. It's the equivalent of `$_` in Perl or `it` in Kotlin lambdas:

```nu
"hello" | { $in + " world" }           # → "hello world"
[1 2 3] | each { $in * $in }           # → [1 4 9]
```

For commands like `each` and `where`, the closure's first parameter and `$in` refer to the same value, but `$in` works in raw blocks without an explicit `|x|`.

### Captures by Value

Closures capture referenced free variables **by value at definition time**:

```nu
mut counter = 0
let inc = { $counter + 1 }    # captures 0, not the binding
$counter = 100
do $inc                        # → 1, not 101
```

This is unlike Rust closures (which capture by reference unless `move`) or Python (which captures by reference, leading to the classic loop-closure bug). Nu's value-capture rule eliminates a class of bugs at the cost of forbidding closure-based mutation.

### Variable Scope

Scope is lexical and block-scoped:

```nu
let x = 1
do { let x = 2; print $x }   # prints 2
print $x                      # prints 1
```

Variables are immutable by default. `mut` declares a mutable binding:

```nu
mut total = 0
for i in 1..10 { $total = $total + $i }
print $total                  # 55
```

Mutable variables **cannot** be captured by closures. This is the parser's enforcement of the value-capture rule.

---

## 6. Custom Commands and Modules

### `def` with Typed Signatures

```nu
def greet [name: string, --shout (-s)] -> string {
    let msg = $"Hello, ($name)"
    if $shout { $msg | str upcase } else { $msg }
}

greet alice                  # → "Hello, alice"
greet alice --shout          # → "HELLO, ALICE"
greet alice -s               # → "HELLO, ALICE"
```

Each parameter has an optional type annotation. Flags use `--name (-short)` syntax. The return type after `->` enables type checking on call sites.

### `def --env` (Environment Mutation)

By default, custom commands run in a child scope — their environment changes don't propagate. `def --env` opts in:

```nu
def --env activate [path: path] {
    $env.OLD_PATH = $env.PATH
    $env.PATH = ($env.PATH | prepend $path)
    $env.PROMPT_PREFIX = "(venv) "
}

activate ~/projects/myenv/bin
```

Without `--env`, the assignments would be visible only inside `activate` and lost on return.

### `def --wrapped` (Pass-Through)

For wrappers around external commands that need to forward unknown flags:

```nu
def --wrapped my-git [...args] {
    print "Running git..."
    ^git ...$args
}

my-git status --short
my-git log --oneline -n 10
```

Without `--wrapped`, Nu's parser would reject unknown flags like `--short`.

### Modules

Modules are units of namespacing. A module is a `.nu` file or a directory containing `mod.nu`.

```nu
# math-utils.nu
export def square [n: int] -> int { $n * $n }
export def cube   [n: int] -> int { $n * $n * $n }
export const PI = 3.14159265
```

```nu
use math-utils.nu *
square 4         # → 16
$PI              # → 3.14159265
```

Or selective import:

```nu
use math-utils.nu [square cube]
```

### The Standard Library

Since 0.86, Nushell ships a standard library under the `std` module:

| Submodule | Purpose |
|:----------|:--------|
| `std assert` | unit-test assertions (`assert eq`, `assert error`) |
| `std log` | structured logging (`log info`, `log debug`) |
| `std dirs` | directory stack (push, pop, drop) |
| `std dt` | date/time helpers (`datetime-diff`) |
| `std formats` | `to jsonl`, `from ndjson`, etc. |
| `std help` | extended help formatters |
| `std iter` | extra iterator combinators |
| `std math` | constants and helpers |
| `std xml` | XML formatter helpers |

```nu
use std assert
assert ((1 + 1) == 2)
assert error { do { error make {msg: "boom"} } }
```

---

## 7. Environment Variables — Typed

### `$env` is a Record

In bash, `$PATH` is a colon-separated string. In Nu, `$env.PATH` is a `list<path>`:

```nu
$env.PATH | length              # → 12 (or however many entries)
$env.PATH | first 3
# ╭───┬─────────────────╮
# │ 0 │ /usr/local/bin  │
# │ 1 │ /usr/bin        │
# │ 2 │ /bin            │
# ╰───┴─────────────────╯

$env.PATH = ($env.PATH | prepend "/opt/nvim/bin")
```

### Environment Inheritance and Conversion

When Nu starts, it inherits the parent process's environment as strings. The `ENV_CONVERSIONS` record specifies how to parse each variable:

```nu
# in env.nu
$env.ENV_CONVERSIONS = {
    PATH: {
        from_string: { |s| $s | split row (char esep) | path expand --no-symlink }
        to_string:   { |v| $v | path expand --no-symlink | str join (char esep) }
    }
    XDG_DATA_DIRS: {
        from_string: { |s| $s | split row (char esep) }
        to_string:   { |v| $v | str join (char esep) }
    }
}
```

`from_string` runs at startup and on every spawned subprocess's incoming environment. `to_string` runs when launching a child process — Nu re-stringifies the typed value to satisfy the OS exec interface (which accepts only `char**`).

### `let-env` (Deprecated) and Modern Syntax

Pre-0.83:

```nu
let-env FOO = "bar"
```

Modern (≥0.83):

```nu
$env.FOO = "bar"
```

The deprecated form still parses but emits a warning. New code should always use the assignment form.

### Hide and Default

```nu
hide-env GIT_TOKEN              # remove from $env
$env.HOME? | default "/tmp"     # safe access
```

---

## 8. The Plugin System

### JSON-RPC over Stdin/Stdout

Plugins are external processes that speak a JSON-RPC variant on stdin/stdout (with optional MessagePack encoding for performance). The convention: a binary named `nu_plugin_<name>` that, when invoked with `--stdio`, enters protocol mode.

```
nu  ─────[Hello] (JSON)──────►  nu_plugin_polars
nu  ◄────[Hello]──────────────  nu_plugin_polars
nu  ─────[Call: Signature]───►  nu_plugin_polars
nu  ◄────[Sig response]───────  nu_plugin_polars   (lists commands)
nu  ─────[Call: Run cmd]──────►  nu_plugin_polars
nu  ◄────[Stream of values]───  nu_plugin_polars
nu  ─────[Goodbye]────────────►  nu_plugin_polars
```

### Protocol Messages

| Message | Direction | Purpose |
|:--------|:----------|:--------|
| `Hello` | both | version/feature negotiation |
| `Call(Signature)` | nu → plugin | request command list |
| `Call(Metadata)` | nu → plugin | request plugin metadata |
| `Call(Run)` | nu → plugin | execute a command |
| `Call(CustomValueOp)` | nu → plugin | invoke a method on a CustomValue |
| `EngineCall` | plugin → nu | call back into nu (e.g., `eval-closure`) |
| `Response(...)` | both | reply to a Call/EngineCall |
| `Stream(Data)` | both | stream a value chunk |
| `Stream(End)` | both | terminate a stream |
| `Goodbye` | both | clean shutdown |

### Registration

Pre-0.93 used `register /path/to/plugin`. Modern syntax:

```nu
plugin add /path/to/nu_plugin_polars
plugin use polars            # activate in current session
plugin list                  # show registered plugins
plugin rm polars             # de-register
```

`plugin add` writes to `~/.config/nushell/plugin.msgpackz` (a binary cache). `plugin use` loads the plugin into the current session's command table.

### Canonical Plugins

| Plugin | Purpose | Status |
|:-------|:--------|:------:|
| `nu_plugin_polars` | DataFrames via Polars | first-party |
| `nu_plugin_query` | jq-like JSON / XPath / web | first-party |
| `nu_plugin_formats` | extra `from`/`to` codecs (eml, ics, vcf) | first-party |
| `nu_plugin_gstat` | git-status enriched output | first-party |
| `nu_plugin_inc` | increment a SemVer string | example |
| `nu_plugin_dbus` | Linux D-Bus bindings | community |
| `nu_plugin_explore` | TUI table browser | community |
| `nu_plugin_highlight` | syntax highlighter | community |
| `nu_plugin_clipboard` | OS clipboard read/write | community |

A complete list is maintained at `nushell.sh/book/plugins.html` and on the `awesome-nu` repo.

### Writing a Plugin in Rust

```rust
use nu_plugin::{EngineInterface, EvaluatedCall, Plugin, PluginCommand, SimplePluginCommand};
use nu_protocol::{Category, LabeledError, Signature, Type, Value};

struct LenPlugin;
struct LenCommand;

impl Plugin for LenPlugin {
    fn version(&self) -> String { env!("CARGO_PKG_VERSION").into() }
    fn commands(&self) -> Vec<Box<dyn PluginCommand<Plugin = Self>>> {
        vec![Box::new(LenCommand)]
    }
}

impl SimplePluginCommand for LenCommand {
    type Plugin = LenPlugin;
    fn name(&self) -> &str { "my-len" }
    fn description(&self) -> &str { "compute string length" }
    fn signature(&self) -> Signature {
        Signature::build("my-len")
            .input_output_types(vec![(Type::String, Type::Int)])
            .category(Category::Strings)
    }
    fn run(
        &self,
        _plugin: &LenPlugin,
        _engine: &EngineInterface,
        call: &EvaluatedCall,
        input: &Value,
    ) -> Result<Value, LabeledError> {
        let s = input.as_str()?;
        Ok(Value::int(s.chars().count() as i64, call.head))
    }
}

fn main() {
    nu_plugin::serve_plugin(&LenPlugin, nu_plugin::MsgPackSerializer);
}
```

Build with `cargo build --release`, then `plugin add target/release/nu_plugin_my_len`.

---

## 9. Polars Plugin Deep

### DataFrames as CustomValue

The Polars plugin wraps `polars::DataFrame` as a `CustomValue` — a runtime-typed value whose internal representation is opaque to Nu's core engine but understood by the plugin.

```nu
let df = polars open data.csv
$df | describe                # core describes it as <CustomValue: NuDataFrame>
$df | polars schema           # plugin-specific introspection
```

The Value stays inside the plugin's process; only handles flow back to Nu. This avoids serialising 100M rows over the JSON-RPC pipe.

### Lazy vs Eager

Polars supports both modes:

```nu
# Eager — load fully into memory, run operations immediately
let df = polars open sales.csv
$df | polars filter (polars col amount > 1000) | polars collect

# Lazy — build a query plan, optimise, execute on collect
let lf = polars open sales.csv --lazy
$lf
| polars filter (polars col amount > 1000)
| polars select [region, amount]
| polars group-by region
| polars agg [(polars col amount | polars sum)]
| polars collect
```

Lazy mode lets the optimiser push the filter into the CSV reader, projection-prune unused columns, and fuse the group-by with the aggregation. For a 1GB file, lazy mode is typically 5–20x faster.

### Core Operators

| Polars op | Purpose | Equivalent SQL |
|:----------|:--------|:---------------|
| `polars filter` | row predicate | `WHERE` |
| `polars select` | column projection | `SELECT cols` |
| `polars group-by` | partition by key | `GROUP BY` |
| `polars agg` | aggregate within group | `agg(col)` |
| `polars sort-by` | order rows | `ORDER BY` |
| `polars join` | relational join | `JOIN ... ON` |
| `polars pivot` | wide-format reshape | `PIVOT` |
| `polars explode` | list → rows | `LATERAL UNNEST` |
| `polars unique` | distinct rows | `DISTINCT` |
| `polars with-column` | derived column | `SELECT ..., expr AS col` |
| `polars sql` | SQL on dataframe | direct SQL |

### SQL Mode

```nu
polars open sales.csv
| polars into-df
| polars sql "select region, sum(amount) from self where year = 2025 group by region order by 2 desc"
```

The plugin uses Polars' built-in SQL frontend (`polars-sql` crate). Joins, window functions, and CTEs work.

### The 10x Pandas Claim

Polars is column-major (Apache Arrow buffers), uses SIMD-vectorised kernels, and parallelises group-by and joins across cores. For a benchmark of 100M-row aggregation:

| Tool | Time |
|:-----|:-----|
| Pandas (Python) | 18.4 s |
| Polars (Python bindings) | 1.6 s |
| Polars in Nushell | 1.7 s |
| Pure Nu (`group-by` + `each`) | 312 s |

The plugin RPC overhead is ~100ms per invocation, so for tiny tables Pandas can win on raw latency. For anything ≥10K rows, Polars dominates.

---

## 10. Error Handling

### `try` / `catch`

```nu
try {
    open missing-file.txt
} catch { |err|
    print $"Couldn't open: ($err.msg)"
    return ""
}
```

The `catch` block receives a `record<msg: string, debug: string, raw: error>` describing the error.

### `do --ignore-errors`

```nu
do --ignore-errors { ^cmd-that-might-fail }
do -i { rm /tmp/maybe-missing }
```

Suppresses errors and yields `null`. Use sparingly — silent failures are bugs in disguise.

### `error make`

Construct a structured error:

```nu
def safe-divide [a: int, b: int] -> int {
    if $b == 0 {
        error make {
            msg: "division by zero",
            label: {
                text: "denominator was zero",
                span: (metadata $b).span
            },
            help: "ensure b != 0 before calling"
        }
    } else {
        $a / $b
    }
}
```

The `label` carries a `Span` so the error message points to the offending source code.

### Result vs Exception Design

Internally, every command returns `Result<PipelineData, ShellError>`. There is no exception unwinding — errors are values flowing through the pipeline. A pipeline aborts at the first stage that returns `Err` unless wrapped in `try`.

External commands emit non-zero exit codes, which Nu maps to `ShellError::ExternalCommandFailed`. The mapping respects the LAST_EXIT_CODE convention:

```nu
^false
print $env.LAST_EXIT_CODE   # 1
```

But unlike bash, the pipeline halts immediately — no `set -e` needed.

### Differences from Shell Exit Codes

| Concept | Bash | Nushell |
|:--------|:-----|:--------|
| Failure propagation | optional (`set -e`) | default |
| Error data | exit code + stderr | structured ShellError + span |
| Try/catch | trap or `\|\|` chain | `try`/`catch` block |
| Pipeline halt on error | optional (`pipefail`) | default |

---

## 11. The Stdlib

### `std assert`

```nu
use std assert

assert ((1 + 1) == 2)
assert eq (1 + 1) 2
assert error { 1 / 0 }
assert length [1 2 3] 3
```

### `std log`

```nu
use std log
$env.NU_LOG_LEVEL = "DEBUG"

log debug "starting up"
log info  "loaded config"
log warning "deprecated flag"
log error "fatal" --short
log critical "system unrecoverable"
```

Output goes to stderr with timestamps and ANSI level colouring. Levels: CRITICAL, ERROR, WARNING, INFO, DEBUG.

### `std dirs`

A directory stack with a custom prompt indicator:

```nu
use std dirs
dirs add ~/work
dirs add ~/play
dirs                         # show stack
dirs next                    # cycle
dirs prev
dirs drop                    # remove current
```

### Test Harness

```nu
# tests/test_math.nu
use std assert
use ../math-utils.nu *

export def test_square [] {
    assert eq (square 4) 16
    assert eq (square 0) 0
}

export def test_cube [] {
    assert eq (cube 3) 27
}
```

Run with the (unofficial-but-canonical) `nutest` runner or by sourcing each file and calling each `test_*` command.

### `std help`

Re-exposes `help` with extended formatting and search:

```nu
use std help
help --find "regex"
help commands | where category == filesystem
```

---

## 12. Performance Characteristics

### Compiled to IR

As of 0.94, Nu compiles parsed blocks to a bytecode-like IR before evaluation. This adds a one-time parsing cost but eliminates per-execution AST walking. For frequently invoked closures inside `each`, this is a measurable speedup.

### Cached Parse Tree

The REPL caches parsed IR per source file. Re-running a script that hasn't changed skips parsing entirely. The cache key is `(path, mtime, size)`.

### Cost of Typed Data

The downside of structured pipelines is allocation overhead. Each `Value` is a tagged union of ≥24 bytes plus heap-allocated string/list contents. For a million-row pipeline, this is ~24 MB of `Value` headers alone.

Bash's text pipeline avoids this — `awk '{print $1}'` over 1M lines uses a fixed-size buffer and emits raw bytes.

### When Nu Is Faster Than Bash

- **Structured parsing** — `from json` / `from yaml` / `from toml` is tens-to-hundreds of times faster than `jq` invoked from bash, because Nu doesn't fork.
- **Repeated commands** — bash forks `awk`/`grep`/`sed` per invocation. Nu's commands are in-process.
- **Pipelines with many small commands** — bash pays a fork-and-exec cost per pipe stage; Nu pays only iterator overhead.

### When Nu Is Slower

- **Single large file processed line-by-line with a tiny operation** — bash's `awk '{ ... }' file` is in-process within awk, while Nu's `open file | lines | each { ... }` allocates a `Value::String` per line.
- **Heavy regex over megabytes of text** — Nu's regex commands wrap Rust's `regex` crate (fast) but the per-line `Value` overhead is real. Pure `grep` with mmap'd input wins.
- **Very small one-shot scripts** — Nu's startup is ~30–50ms vs bash's ~5ms. For an `if`-statement-and-exit script, bash wins on wall-clock.

### Benchmark Snapshot

```nu
use std bench
bench { ls | where size > 1mb | length } --rounds 100
# returns timing record with mean, std, p99
```

For a directory of 10K files:

| Pipeline | Bash | Nushell |
|:---------|:-----|:--------|
| Count files >1MB | 0.18 s (`find`+`awk`) | 0.21 s |
| Parse 100MB JSON | 4.2 s (`jq`) | 0.9 s |
| Group-by + aggregate (1M rows CSV) | 38 s (awk pipelines) | 1.2 s (polars) |

---

## 13. Configuration

### The `$nu` Constant

`$nu` is a read-only record describing the current Nu installation:

```nu
$nu
# ╭──────────────────────┬─────────────────────────────────╮
# │ default-config-dir   │ /Users/govan/.config/nushell    │
# │ config-path          │ /Users/govan/.config/nushell/config.nu │
# │ env-path             │ /Users/govan/.config/nushell/env.nu    │
# │ history-path         │ /Users/govan/.config/nushell/history.txt │
# │ loginshell-path      │ /Users/govan/.config/nushell/login.nu  │
# │ plugin-path          │ /Users/govan/.config/nushell/plugin.msgpackz │
# │ home-path            │ /Users/govan                            │
# │ data-dir             │ /Users/govan/.local/share/nushell        │
# │ cache-dir            │ /Users/govan/.cache/nushell              │
# │ temp-path            │ /var/folders/...                         │
# │ pid                  │ 84231                                    │
# │ os-info              │ {name: macos, arch: aarch64, ...}        │
# │ startup-time         │ 84ms                                     │
# │ is-interactive       │ true                                     │
# │ is-login             │ false                                    │
# │ current-exe          │ /opt/homebrew/bin/nu                     │
# │ history-enabled      │ true                                     │
# ╰──────────────────────┴─────────────────────────────────╯
```

### `~/.config/nushell/config.nu`

The main interactive config: prompt, key bindings, color theme, completion settings. Generated on first run by `config nu --default`.

```nu
$env.config = {
    show_banner: false
    table: {
        mode: rounded
        index_mode: auto
        show_empty: true
    }
    history: {
        max_size: 100_000
        sync_on_enter: true
        file_format: "sqlite"
        isolation: false
    }
    completions: {
        case_sensitive: false
        quick: true
        partial: true
        algorithm: "fuzzy"
        external: { enable: true, max_results: 100 }
    }
    cursor_shape: { vi_insert: line, vi_normal: block }
    edit_mode: vi
    color_config: $dark_theme
    keybindings: [
        { name: completion_menu, modifier: none, keycode: tab,
          mode: [emacs vi_normal vi_insert],
          event: { until: [{send: menu, name: completion_menu}, {send: menunext}] } }
    ]
}
```

### `~/.config/nushell/env.nu`

Loaded *before* `config.nu`. Sets `$env.PATH`, `ENV_CONVERSIONS`, prompt strings:

```nu
$env.PATH = ($env.PATH | split row (char esep) | prepend "/opt/homebrew/bin")
$env.ENV_CONVERSIONS = {
    PATH: { from_string: { ... }, to_string: { ... } }
}
$env.PROMPT_COMMAND = {|| $"(ansi green_bold)(pwd)(ansi reset)" }
$env.PROMPT_INDICATOR = "❯ "
```

### Theme

```nu
let $dark_theme = {
    separator: white
    leading_trailing_space_bg: { attr: n }
    header: green_bold
    empty: blue
    bool: light_cyan
    int: white
    filesize: cyan
    duration: white
    date: purple
    range: white
    float: white
    string: white
    nothing: white
    binary: white
    cellpath: white
    row_index: green_bold
    record: white
    list: white
    block: white
    hints: dark_gray
    search_result: { bg: red, fg: white }
}
```

### Keybindings

Vi-mode and emacs-mode are first-class. Custom chords:

```nu
$env.config.keybindings = [
    {
        name: redo_change
        modifier: control
        keycode: char_r
        mode: vi_normal
        event: { send: redo }
    }
    {
        name: fuzzy_history
        modifier: control
        keycode: char_r
        mode: emacs
        event: {
            send: executehostcommand
            cmd: "commandline edit --replace (history | get command | uniq | str join (char nul) | fzf --read0 --tac)"
        }
    }
]
```

---

## 14. The HTTP and Database Modules

### HTTP — Structured Response

```nu
http get https://api.github.com/repos/nushell/nushell
# returns a record with parsed JSON

http get https://example.com --headers [User-Agent custom]
http post https://api.example.com/resource --content-type application/json {key: value}
http put / http delete / http head / http options / http patch
```

The response body is auto-parsed based on `Content-Type`:

| Content-Type | Parsed As |
|:-------------|:----------|
| `application/json` | `record`/`list` |
| `text/html` | `string` |
| `application/xml`, `text/xml` | `record` (with `query` plugin) |
| anything else | raw `binary` |

Headers, status, and metadata are accessible via `--full`:

```nu
let r = http get --full https://example.com
$r.status                         # 200
$r.headers.content-type           # text/html; charset=UTF-8
$r.body                           # parsed body
```

### Database — `query db`

```nu
query db ./data.sqlite "select * from users where age > 30"
# returns table<id, name, age>
```

The `query db` command (formerly `into sqlite`) executes SQL and returns a table. For ad-hoc analysis:

```nu
query db ./prod.sqlite "select region, count(*) as n from orders group by region"
| where n > 100
| sort-by n --reverse
| first 5
| save report.csv
```

The connector uses `rusqlite` and supports SQLite-only. For Postgres/MySQL, use the `nu_plugin_query_db` community plugin or fall back to `^psql -c "..." | from csv`.

### `into sqlite`

Persist a Nu table into a SQLite file:

```nu
ls **/*.log
| select name size modified
| into sqlite logs.db --table-name files
```

Then reload with `query db logs.db "select * from files"`.

---

## 15. Migration from Unix Shells

### The No-POSIX-Compatibility Reality

Nushell is **not** a POSIX shell. Specifically:

- No `$?`, no `$!`, no `$$` (use `(metadata $env.LAST_EXIT_CODE)`, etc.).
- No `&&` / `||` (use `if` or `try`).
- No subshells `$(...)` — use parentheses `(cmd)` for grouping; subexpressions evaluate inline.
- No globbing operators `?` `*` `[...]` outside specific contexts (Nu does glob, but the syntax is `glob "*.txt"`).
- No `IFS`, no word splitting.
- No here-docs (use multi-line strings).
- No process substitution `<(cmd)`.
- Conditional execution via `if`, not `&&`.

### Calling External Binaries

Any command name that isn't a built-in is treated as external:

```nu
git status                    # external
^git status                   # explicit external (shadows built-in)
^ls                           # explicit GNU ls, not Nu's ls
```

The `^` prefix is required when an external binary shares a name with a built-in command.

### External Output Is a String — Use `lines`

```nu
^ls -la | lines | length        # number of lines in ls output
^cat /etc/hosts | lines | where ($it | str contains "localhost")
```

`lines` splits a string on newlines into a `list<string>`. For structured parsing of external output, use `from csv`, `from json`, etc.:

```nu
^ps -eo pid,ppid,cmd | from ssv | first 10
^docker ps --format json | lines | each { from json }
```

### The `|>` Operator (Future)

Nushell's roadmap includes `|>` for explicit external invocation distinct from `|`. As of 0.97, the experimental `--rest-arg` semantics handle most use cases. The fully POSIX-compatible byte-stream pipe is `^cmd1 | ^cmd2`, where Nu treats both sides as ByteStreams.

### Common Bash → Nu Translations

| Bash | Nushell |
|:-----|:--------|
| `find . -name "*.log" -size +1M` | `ls **/*.log \| where size > 1mb` |
| `cat file \| awk '{print $1}'` | `open file \| lines \| each { split row " " \| get 0 }` |
| `cat data.csv \| sort \| uniq -c` | `open data.csv \| from csv \| group-by col \| transpose key val \| update val { length }` |
| `for f in *.txt; do cmd "$f"; done` | `ls *.txt \| each { \|f\| cmd $f.name }` |
| `if [ -f file ]; then ...; fi` | `if ("file" \| path exists) { ... }` |
| `VAR=value cmd` | `with-env { VAR: value } { cmd }` |
| `cmd1 \|\| cmd2` | `try { cmd1 } catch { cmd2 }` |
| `cmd \| tee file` | `cmd \| tee { save file }` |

---

## 16. Idioms at the Internals Depth

### The Canonical Filter-Project-Sort-Aggregate

```nu
ls
| where size > 1mb and modified > (date now - 30day)
| select name size modified
| sort-by modified --reverse
| first 10
```

This single pipeline replaces dozens of lines of bash + `find` + `xargs` + `awk` + `sort`.

### Group-By + Each + Reduce

For aggregations that don't fit into the built-in `math sum` / `math avg` family:

```nu
$transactions
| group-by category
| transpose category records
| each { |grp|
    {
      category: $grp.category
      total:    ($grp.records | get amount | math sum)
      count:    ($grp.records | length)
      max_txn:  ($grp.records | get amount | math max)
    }
  }
| sort-by total --reverse
```

### `from json | get .key` Instead of jq

```nu
http get https://api.github.com/repos/nushell/nushell
| get stargazers_count
```

No jq, no awk, no shell-quoting hell.

### Streaming Large Files

```nu
open big.log
| lines
| where ($it | str contains "ERROR")
| parse "{date} {time} ERROR {message}"
| group-by message
| transpose msg occurrences
| update occurrences { length }
| sort-by occurrences --reverse
| first 20
```

The pipeline is lazy from `open` to `where` — only matched lines are materialised.

### Parallel Map

```nu
ls **/*.jpg
| par-each --threads 8 { |f|
    {
      file: $f.name
      width: (^identify -format "%w" $f.name | into int)
      height: (^identify -format "%h" $f.name | into int)
    }
  }
| sort-by width --reverse
```

`par-each` distributes the closure over a thread pool. Order is preserved by default. Use this for IO-bound or CPU-bound per-row work.

### The `update` / `insert` / `reject` Cascade

```nu
$tbl
| update price { |row| $row.price * 1.1 }    # mutate column
| insert tax { |row| $row.price * 0.07 }      # add column
| reject discount                              # remove column
| rename --column { price: gross }             # rename
```

---

## 17. The Future — 1.0 and Beyond

### Version Cadence

Nushell ships every four weeks. The version sequence as of this writing is `0.97 → 0.98 → 0.99 → 0.100 → ...`. Despite the leading `0.`, breaking changes are now relatively contained — major API surface is stable.

### Path to 1.0

The 1.0 milestone targets:
- Stable plugin protocol (frozen)
- Stable `$env` and `$nu` shapes
- Frozen `Value` enum (no new variants in patch releases)
- Frozen syntax for `def`, closures, modules
- Backwards-compatible config (with deprecation warnings, not errors)

Features that are still rapidly evolving:
- IR / bytecode internals (private)
- Some stdlib reorganisation (`std iter`, `std math`)
- Plugin discovery (`plugin add` semantics)
- The `|>` operator and shell-mode toggling
- `dataframes` core (now wholly delegated to the polars plugin)

### "Use 0.90+ for Stability"

The unofficial guidance: pin to a specific Nu version in scripts. Production scripts should:

1. Print `version` at top and refuse to run on older releases.
2. Use `--strict` parsing once it's available (planned).
3. Avoid features marked "experimental" in the docs.

```nu
let v = (version | get version)
if ($v | str starts-with "0.8") {
    error make { msg: "this script requires nushell ≥ 0.90" }
}
```

### Async and `await`

Currently, every command is sync (with internal threading via `par-each`). RFC discussions (nushell/nushell#10867 et al.) propose first-class `await` for `http` and `query db` to compose with timeouts and cancellation. Expected post-1.0.

### Embedding

`nu-engine` is designed to be embeddable. Projects like `nu_plugin_explore` and the upcoming `nu` Rust embedding API expose `EngineState` + `Stack` for use as a scripting engine inside larger Rust applications. As of 0.97 the API is unstable but functional.

---

## Prerequisites

- bash, fish, zsh, polyglot, regex, json

## Complexity

- **Parser:** O(n) on source length; single-pass with span tracking
- **Cell-path traversal:** O(d × n) where d = depth, n = list/record size
- **`where` filter:** O(n) in row count, lazy
- **`sort-by`:** O(n log n) in-memory mergesort
- **`group-by`:** O(n) hash-based
- **`par-each`:** O(n / cores) for embarrassingly parallel work, with thread spawn ~50µs

## See Also

- nushell, bash, zsh, fish, polyglot

## References

- Nushell official site — https://www.nushell.sh/
- The Nu Book — https://www.nushell.sh/book/
- Reference docs (commands) — https://www.nushell.sh/commands/
- Source — https://github.com/nushell/nushell
- Plugin API docs — https://docs.rs/nu-plugin/latest/nu_plugin/
- Polars plugin — https://github.com/nushell/nushell/tree/main/crates/nu_plugin_polars
- `awesome-nu` (curated plugins/themes) — https://github.com/nushell/awesome-nu
- Reedline (line editor) — https://github.com/nushell/reedline
- Design discussion thread index — https://github.com/nushell/nushell/discussions
- Polars project — https://pola.rs/
- Apache Arrow (Polars buffer format) — https://arrow.apache.org/
- The Nu blog (release notes) — https://www.nushell.sh/blog/
- IR design RFC (#11876) — https://github.com/nushell/nushell/pull/11876
- Plugin protocol (msgpack/json) — https://www.nushell.sh/contributor-book/plugin_protocol_reference.html
