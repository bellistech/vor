# Cargo

The Rust build system, package manager, test runner, doc generator, and project scaffolder. Reads `Cargo.toml`, writes `Cargo.lock`, drives `rustc`, `rustfmt`, `clippy`, downloads from crates.io, builds workspaces, runs benches, ships binaries via `cargo install`, publishes crates, and integrates with rustup-managed toolchains.

## Setup

### Install rustup (the toolchain installer)

```bash
# Linux / macOS / WSL — official one-liner from rustup.rs
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# non-interactive (CI)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable --profile minimal

# Windows: download rustup-init.exe from https://rustup.rs

# macOS Homebrew alternative (NOT recommended — rustup is preferred)
brew install rustup-init && rustup-init -y
```

### Activate the cargo env in this shell

```bash
. "$HOME/.cargo/env"          # bash/zsh
source "$HOME/.cargo/env.fish" # fish

# or add to your shell rc permanently
export PATH="$HOME/.cargo/bin:$PATH"
```

### Verify the install

```bash
rustup --version              # rustup 1.27.x
rustc --version               # rustc 1.79.0 (129f3b996 2024-06-10)
cargo --version               # cargo 1.79.0 (ffa9cf99a 2024-06-03)
rustup show                   # shows installed toolchains, default, components
rustup show active-toolchain  # which toolchain is in use right here
which cargo                   # ~/.cargo/bin/cargo (a rustup proxy)
```

### Channels: stable, beta, nightly

```bash
# install channels
rustup install stable
rustup install beta
rustup install nightly

# pin a specific dated nightly (reproducible)
rustup install nightly-2024-06-01

# pin a specific stable version
rustup install 1.79.0
rustup install 1.79.0-x86_64-unknown-linux-gnu

# set the default toolchain (used when rust-toolchain.toml is absent)
rustup default stable
rustup default nightly

# update toolchains
rustup update                 # update all installed channels
rustup update stable          # just stable
rustup self update            # update rustup itself

# remove a toolchain
rustup uninstall nightly-2024-01-01

# run a one-off cargo invocation on a different toolchain
cargo +nightly build
cargo +1.79.0 test
rustup run nightly cargo build  # same idea, longer
```

### Components (rustfmt, clippy, rust-analyzer, llvm-tools-preview)

```bash
# list every component installable for a toolchain
rustup component list
rustup component list --installed
rustup component list --toolchain nightly

# install components
rustup component add rustfmt
rustup component add clippy
rustup component add rust-analyzer            # LSP server
rustup component add rust-src                 # required for rust-analyzer expand-macro / no_std builds
rustup component add llvm-tools-preview       # llvm-cov, llvm-objdump, llvm-profdata
rustup component add miri --toolchain nightly # undefined behavior detector
rustup component add rustc-dev --toolchain nightly  # internal compiler crates
rustup component add cargo                    # rare; bundled by default

# remove a component
rustup component remove rustfmt
```

### Targets (cross-compilation)

```bash
# list every target tier
rustup target list
rustup target list --installed

# add common targets
rustup target add x86_64-unknown-linux-musl       # static linux
rustup target add aarch64-unknown-linux-gnu       # 64-bit ARM linux
rustup target add aarch64-apple-darwin            # Apple Silicon
rustup target add x86_64-apple-darwin             # Intel mac
rustup target add x86_64-pc-windows-gnu           # mingw windows
rustup target add x86_64-pc-windows-msvc          # native windows (msvc toolchain)
rustup target add wasm32-unknown-unknown          # browser wasm
rustup target add wasm32-wasi                     # WASI
rustup target add thumbv7em-none-eabihf           # ARM Cortex-M4F bare metal
rustup target add riscv64gc-unknown-linux-gnu

# remove
rustup target remove wasm32-unknown-unknown
```

### Toolchain pinning via rust-toolchain.toml

Drop this file at the repo root — every cargo/rustc invocation inside the repo will use the pinned toolchain (rustup downloads it on demand):

```bash
cat > rust-toolchain.toml <<'EOF'
[toolchain]
channel    = "1.79.0"                # or "stable" / "nightly-2024-06-01"
components = ["rustfmt", "clippy", "rust-analyzer", "rust-src"]
targets    = ["wasm32-unknown-unknown", "x86_64-unknown-linux-musl"]
profile    = "minimal"               # minimal | default | complete
EOF

# legacy single-line file (still supported, no components/targets)
echo "1.79.0" > rust-toolchain
```

`rustup show` from inside the repo confirms the override:

```bash
cd /path/to/repo && rustup show active-toolchain
# 1.79.0-x86_64-unknown-linux-gnu (overridden by '/path/to/repo/rust-toolchain.toml')
```

### Cargo home and registry cache

```bash
# default locations
echo $CARGO_HOME              # ~/.cargo  (override with CARGO_HOME)
echo $RUSTUP_HOME             # ~/.rustup (override with RUSTUP_HOME)

ls ~/.cargo
# bin/      registry/  git/    config.toml  credentials.toml  env
ls ~/.cargo/registry
# cache/  index/  src/

# clear the registry cache (forces re-download)
rm -rf ~/.cargo/registry/cache ~/.cargo/registry/src

# global cargo config
cat ~/.cargo/config.toml
```

## Project Layout

```bash
mycrate/
  Cargo.toml         # manifest — what this crate is, what it depends on
  Cargo.lock         # exact resolved versions (commit for bins, libs optional)
  src/
    main.rs          # binary crate root (default bin = package name)
    lib.rs           # library crate root
    bin/
      tool-a.rs      # extra binary, run via cargo run --bin tool-a
      tool-b/
        main.rs
  examples/
    demo.rs          # cargo run --example demo
  tests/
    integration.rs   # one separate binary per file; cargo test
  benches/
    bench.rs         # cargo bench (stable: criterion; nightly: test::Bencher)
  build.rs           # build script — runs before compile
  target/            # all build output (add to .gitignore)
  .cargo/
    config.toml      # repo-local cargo config (linker flags, aliases, etc.)
```

A package can contain at most one `[lib]` and any number of `[[bin]]`, `[[example]]`, `[[test]]`, `[[bench]]`. Auto-discovery turns every file in `src/bin/`, `examples/`, `tests/`, `benches/` into the matching target unless disabled with `autobins = false` etc.

```bash
# inspect what targets cargo sees
cargo metadata --no-deps --format-version=1 | jq '.packages[].targets[] | {name, kind, src_path}'
```

## cargo new / cargo init

`cargo new <path>` makes a directory; `cargo init` operates on the current directory.

```bash
# new binary crate (default)
cargo new hello
cargo new hello --bin              # explicit
cargo new hello --vcs git          # also: hg, pijul, fossil, none
cargo new hello --edition 2021     # 2015 | 2018 | 2021 | 2024
cargo new hello --name my-tool     # crate name differs from directory
cargo new --quiet hello            # no "Created binary..." line

# new library crate
cargo new mylib --lib

# initialize in existing directory
mkdir myproj && cd myproj && cargo init --lib --vcs none

# what the bin template gives you
cat hello/Cargo.toml
# [package]
# name = "hello"
# version = "0.1.0"
# edition = "2021"
#
# [dependencies]

cat hello/src/main.rs
# fn main() {
#     println!("Hello, world!");
# }
```

Edition matters. `--edition 2024` requires rustc 1.85+. Mixing editions across deps is fine — each crate uses its declared edition.

## cargo build

Compiles every default target (lib + bins) for the current package.

```bash
cargo build                           # debug build, output in target/debug/
cargo build --release                 # optimized, target/release/
cargo build -r                        # short for --release

# target selection
cargo build --bin myapp               # one binary
cargo build --bins                    # all binaries
cargo build --lib                     # just the library
cargo build --example demo            # one example
cargo build --examples                # all examples
cargo build --tests                   # all integration tests
cargo build --benches                 # all benches
cargo build --all-targets             # lib + bins + examples + tests + benches

# workspace
cargo build --workspace               # every member crate
cargo build -p myapp                  # one crate in the workspace
cargo build --exclude flaky-crate     # skip one (used with --workspace)

# cross-compile
cargo build --target wasm32-unknown-unknown
cargo build --target aarch64-apple-darwin --release

# profile selection (defined in Cargo.toml [profile.X])
cargo build --profile dev             # default for build (= debug)
cargo build --profile release         # default for --release
cargo build --profile bench           # used by cargo bench (release-like)
cargo build --profile test            # used by cargo test (debug-like)
cargo build --profile dist            # custom profile

# features
cargo build --features "tls postgres"
cargo build --features tls,postgres
cargo build -F tls -F postgres        # repeatable short form
cargo build --all-features
cargo build --no-default-features
cargo build --no-default-features --features tls

# concurrency
cargo build -j 4                      # 4 parallel rustc instances
cargo build -j 1                      # serial (debug a flaky build script)

# offline / locked
cargo build --offline                 # use only the local cache
cargo build --frozen                  # require Cargo.lock AND offline cache
cargo build --locked                  # require Cargo.lock to be up to date

# verbose / quiet
cargo build -v                        # show rustc command lines
cargo build -vv                       # also show build script output
cargo build --quiet                   # suppress progress
cargo build --message-format=json     # machine-readable diagnostics
cargo build --message-format=json-render-diagnostics
cargo build --message-format=short    # one line per error
cargo build --color=always            # force color (CI)
cargo build --timings                 # write target/cargo-timings/cargo-timing.html
cargo build --keep-going              # do not stop on first crate failure

# pass extra rustc flags one-shot
RUSTFLAGS="-C target-cpu=native" cargo build --release
cargo rustc -- -C link-arg=-fuse-ld=lld
```

After `cargo build` the artifact path is `target/<profile>/<name>` (or `<name>.exe` on Windows). For cross builds it's `target/<triple>/<profile>/<name>`.

## cargo run

Builds (if needed) and executes a binary or example.

```bash
cargo run                             # default bin (package name)
cargo run --bin myapp                 # specific bin
cargo run --example demo              # an example
cargo run --release                   # optimized
cargo run --features "tls"
cargo run --target x86_64-unknown-linux-musl

# pass arguments to YOUR program with --
cargo run -- --help
cargo run --release --bin myapp -- --port 8080 --verbose
cargo run --example demo -- input.txt

# env vars are forwarded
RUST_LOG=debug RUST_BACKTRACE=1 cargo run

# combine with --quiet to hide cargos status lines
cargo run --quiet -- --version

# multi-crate workspace
cargo run -p server --bin server -- --addr 0.0.0.0:9000
```

The split between cargo flags and program args is exactly one `--`. Anything after it goes to the binary verbatim.

## cargo check

Type-checks and borrow-checks but does not generate machine code or link. ~3-5x faster than `cargo build`. The standard editor save loop.

```bash
cargo check                           # check default targets
cargo check --all-targets             # also examples/tests/benches
cargo check --workspace               # every member
cargo check --workspace --all-targets --all-features  # the maximal smoke test
cargo check --tests                   # check test code without running it
cargo check --message-format=json     # what rust-analyzer eats
cargo check -p mycrate                # one crate

# typical IDE setup (rust-analyzer runs this on save)
cargo check --workspace --all-targets --message-format=json-diagnostic-rendered-ansi
```

`cargo check` writes `.rmeta` metadata files but no `.rlib`/`.exe`. The artifacts cannot be `cargo run`. Use `cargo check` while iterating on type errors, then switch to `cargo build` once it compiles.

## cargo test

Compiles every test target and runs them.

```bash
cargo test                            # everything (lib unit + integration + doctests)
cargo test --release                  # tests in release profile (slower compile, faster run)
cargo test --workspace                # every member
cargo test -p mycrate                 # one crate
cargo test --all-features
cargo test --no-default-features

# target filters
cargo test --lib                      # only #[cfg(test)] modules in src/
cargo test --bins                     # tests inside bin crates
cargo test --tests                    # only files in tests/
cargo test --test integration         # only tests/integration.rs
cargo test --test '*'                 # all integration tests
cargo test --doc                      # only doctests inside /// blocks
cargo test --examples                 # examples are checked-compiled, not run
cargo test --benches

# filter by test name (substring match)
cargo test parse                      # any test whose path contains "parse"
cargo test parse::nested              # exact module path
cargo test -- --exact tests::parses_empty   # exact match (= -- before --exact)

# pass args to the test harness AFTER --
cargo test -- --nocapture             # show println! and dbg! output
cargo test -- --test-threads=1        # serialize tests (debug shared state)
cargo test -- --ignored               # only #[ignore]'d tests
cargo test -- --include-ignored       # both regular and ignored
cargo test -- --skip slow             # skip tests matching "slow"
cargo test -- --list                  # list test names without running
cargo test -- --show-output           # like --nocapture but only on success too
cargo test -- -Z unstable-options --report-time   # nightly: per-test timing
cargo test -- --format=json -Z unstable-options   # nightly: JSON output

# common combinations
cargo test --release -- --test-threads=1 --nocapture
cargo test --workspace --all-features -- --include-ignored

# run a specific doctest by name
cargo test --doc parse_csv

# environment variables
RUST_LOG=debug cargo test -- --nocapture
RUST_BACKTRACE=full cargo test
```

Integration tests live in `tests/`. Each `tests/foo.rs` is compiled as a separate binary that links against your library — so you can only `use mycrate::pub_item;`, not internal modules. Helper modules go in `tests/common/mod.rs` (using `mod.rs` form prevents auto-discovery).

```bash
# example tests/api.rs
cat > tests/api.rs <<'EOF'
use mylib::add;

#[test]
fn adds_positive() { assert_eq!(add(2, 3), 5); }

#[test]
#[ignore]
fn slow_db_test() { /* requires postgres */ }
EOF

cargo test --test api
cargo test --test api -- slow_db_test --include-ignored
```

Doctests run code in `///` and `//!` examples. They live in the rendered docs and act as compile checks:

```bash
# in src/lib.rs
# /// Adds two numbers.
# ///
# /// ```
# /// assert_eq!(mylib::add(1, 2), 3);
# /// ```
# pub fn add(a: i32, b: i32) -> i32 { a + b }

cargo test --doc                      # runs the assert above
```

Disable doctests for a crate by adding to Cargo.toml:

```bash
cat >> Cargo.toml <<'EOF'
[lib]
doctest = false
EOF
```

## cargo bench

Runs benchmark targets. The stable path is the criterion crate (statistical, no nightly). The nightly path is the built-in `test::Bencher` harness.

```bash
# stable: add criterion to dev-dependencies
cat >> Cargo.toml <<'EOF'
[dev-dependencies]
criterion = { version = "0.5", features = ["html_reports"] }

[[bench]]
name    = "fib"
harness = false                       # disables built-in test harness so criterion's main() runs
EOF

# benches/fib.rs
cat > benches/fib.rs <<'EOF'
use criterion::{criterion_group, criterion_main, Criterion, black_box};

fn fib(n: u64) -> u64 { if n < 2 { n } else { fib(n - 1) + fib(n - 2) } }

fn bench_fib(c: &mut Criterion) {
    c.bench_function("fib 20", |b| b.iter(|| fib(black_box(20))));
}

criterion_group!(benches, bench_fib);
criterion_main!(benches);
EOF

cargo bench                           # runs every bench
cargo bench --bench fib               # one bench file
cargo bench fib_20                    # filter by name
cargo bench -- --save-baseline main   # criterion: save baseline
cargo bench -- --baseline main        # compare against saved baseline

# nightly built-in
cargo +nightly bench
# benches/native.rs
# #![feature(test)]
# extern crate test;
# #[bench]
# fn bench_add(b: &mut test::Bencher) { b.iter(|| 1 + 1); }
```

`harness = false` is required for criterion (and any custom main in a bench) — without it, cargo wraps your main() in libtest and refuses to find `criterion_main!`.

Reports land in `target/criterion/`. Open `target/criterion/report/index.html` in a browser only if you choose; the terminal output already has mean/median/std-dev/regression numbers.

## cargo fmt

Wrapper around rustfmt. Formats every `.rs` in the package (or workspace).

```bash
rustup component add rustfmt          # required component

cargo fmt                             # format in place
cargo fmt --all                       # entire workspace
cargo fmt -- --check                  # check only; non-zero exit if reformatting needed (CI)
cargo fmt --all -- --check
cargo fmt -p mycrate                  # one crate

# format a specific file (rustfmt directly)
rustfmt src/lib.rs
rustfmt --edition 2021 src/lib.rs
rustfmt --emit=stdout src/lib.rs      # do not write the file

# nightly-only options need +nightly
cargo +nightly fmt -- --unstable-features --skip-children
cargo +nightly fmt -- --config 'group_imports=StdExternalCrate,imports_granularity=Crate'
```

### rustfmt.toml — common options

```bash
cat > rustfmt.toml <<'EOF'
edition            = "2021"
max_width          = 100
hard_tabs          = false
tab_spaces         = 4
newline_style      = "Unix"
use_small_heuristics = "Default"
fn_call_width      = 60
attr_fn_like_width = 70
struct_lit_width   = 18
chain_width        = 60
single_line_if_else_max_width = 50
wrap_comments      = false                # nightly
comment_width      = 80                   # nightly
format_code_in_doc_comments = false       # nightly
imports_granularity = "Crate"             # nightly
group_imports      = "StdExternalCrate"   # nightly
reorder_imports    = true
reorder_modules    = true
EOF
```

Stable rustfmt only honors stable options. Nightly options listed above are silently ignored unless `cargo +nightly fmt`.

## cargo clippy

Lint suite layered on top of rustc. Around 700 lints organized into groups.

```bash
rustup component add clippy

cargo clippy                          # default lints
cargo clippy --all-targets            # also tests/examples/benches
cargo clippy --workspace --all-targets --all-features
cargo clippy --release                # release-mode lints (catches dead_code differently)

# escalate warnings
cargo clippy -- -D warnings           # any clippy lint = compile error
cargo clippy -- -D clippy::all -W clippy::pedantic
cargo clippy -- -D clippy::unwrap_used -D clippy::expect_used

# lint groups
#   correctness  almost always a real bug         default = deny
#   suspicious   probably wrong                   default = warn
#   style        non-idiomatic                    default = warn
#   complexity   simpler way exists               default = warn
#   perf         performance issue                default = warn
#   pedantic     opinionated                      default = allow
#   nursery      under development                default = allow
#   restriction  opt-in (bans)                    default = allow
#   cargo        Cargo.toml lints                 default = allow

cargo clippy -- -W clippy::pedantic
cargo clippy -- -W clippy::nursery
cargo clippy -- -W clippy::cargo

# auto-fix (uses rustfix)
cargo clippy --fix                                        # requires clean tree
cargo clippy --fix --allow-dirty                          # uncommitted changes ok
cargo clippy --fix --allow-staged                         # staged changes ok
cargo clippy --fix --allow-dirty --allow-staged
cargo clippy --fix --workspace --all-targets

# JSON output
cargo clippy --message-format=json
```

### Per-crate or per-item lint config

```bash
# in src/lib.rs / src/main.rs
# #![deny(clippy::correctness)]
# #![warn(clippy::pedantic, clippy::nursery)]
# #![allow(clippy::too_many_arguments, clippy::module_name_repetitions)]

# scope to one item
# #[allow(clippy::cast_possible_truncation)]
# fn down(n: u64) -> u32 { n as u32 }
```

### clippy.toml

```bash
cat > clippy.toml <<'EOF'
msrv                = "1.65"          # do not suggest features past this
cognitive-complexity-threshold = 30
too-many-arguments-threshold   = 8
type-complexity-threshold      = 250
disallowed-methods = [
  { path = "std::env::var", reason = "use config_lib::var" },
]
disallowed-types = [
  { path = "std::collections::HashMap", reason = "use ahash::AHashMap" },
]
EOF
```

### Cargo.toml lint table (rust 1.74+)

```bash
cat >> Cargo.toml <<'EOF'
[lints.rust]
unsafe_code = "forbid"

[lints.clippy]
all       = "deny"
pedantic  = "warn"
unwrap_used = "deny"
expect_used = "deny"
EOF
```

## cargo doc

Builds rustdoc HTML for the package and its dependencies.

```bash
cargo doc                             # docs for this crate AND deps in target/doc/
cargo doc --no-deps                   # only this crate (much faster)
cargo doc --open                      # build then open in $BROWSER
cargo doc --no-deps --open
cargo doc --document-private-items    # include pub(crate) and private items
cargo doc --workspace --no-deps
cargo doc --all-features
cargo doc --release                   # docs for release-cfg'd code
cargo doc --target wasm32-unknown-unknown

# regenerate from scratch
rm -rf target/doc && cargo doc --no-deps

# serve locally without leaving the terminal
python3 -m http.server -d target/doc 8000
# then `lynx http://localhost:8000` or `links http://localhost:8000`
```

### Intra-doc links

```bash
# /// See [`Foo::bar`] and [the iterator chapter][iter].
# ///
# /// [iter]: std::iter
# pub struct Foo;

# any path that resolves at compile time works:
# [`std::vec::Vec`]
# [`Vec`]                    # if Vec is in scope
# [`crate::module::Item`]
# [`Item`]
# [`function`]
# [`Trait::method`]
```

Broken intra-doc links can be turned into errors:

```bash
cargo rustdoc -- -D rustdoc::broken_intra_doc_links
# or in lib.rs:  #![deny(rustdoc::broken_intra_doc_links)]
```

### Doctests vs doc generation

`cargo doc` does NOT run doctests; `cargo test --doc` does. A doctest with `ignore`/`no_run`/`should_panic`/`compile_fail` annotations is treated specially:

```bash
# /// ```ignore         — not compiled, not run
# /// ```no_run         — compiled, not run
# /// ```should_panic   — must panic to pass
# /// ```compile_fail   — must fail to compile to pass
# /// ```text           — not Rust; skipped entirely
```

## cargo tree

Print the dependency graph as a tree. Indispensable for diagnosing version conflicts and feature blow-ups.

```bash
cargo tree                            # the whole tree
cargo tree --workspace
cargo tree -p serde                   # only the subtree from one package
cargo tree --depth 1                  # direct deps only
cargo tree --depth 2

# duplicates: same crate at multiple versions
cargo tree -d                         # show duplicates
cargo tree -d -i tokio                # who pulls in each tokio?

# inverted: who depends on X?
cargo tree -i serde
cargo tree -i serde --workspace
cargo tree -i serde:1.0.0             # exact version

# features
cargo tree --features tls,postgres
cargo tree --all-features
cargo tree --no-default-features
cargo tree -e features                # show feature edges
cargo tree -e=no-dev                  # hide dev-deps
cargo tree -e=normal,build,dev        # show all edge kinds (default)

# target-specific deps
cargo tree --target x86_64-unknown-linux-gnu
cargo tree --target all               # union of every target

# formatting
cargo tree --prefix=indent            # default visual tree
cargo tree --prefix=depth             # numeric depth prefix
cargo tree --prefix=none              # flat list
cargo tree --format '{p} {l} {f}'     # placeholders: p=name+ver, l=license, f=features, r=repo
cargo tree --no-dedupe                # show every occurrence (default dedupes repeats)

# strip noise for grep
cargo tree --prefix=none --no-dedupe | sort -u
```

## cargo update

Updates `Cargo.lock` against the latest registry index.

```bash
cargo update                          # bump everything compatible with Cargo.toml
cargo update -p serde                 # bump just serde (and what it forces)
cargo update -p serde --precise 1.0.197   # pin to exact version
cargo update -p serde --aggressive    # also bump serde's own deps as far as allowed

# remove a crate from the lockfile (forces re-resolve)
# (cargo update -p X --precise <whatever> is the usual way; downgrades work too)

# show what is outdated WITHOUT modifying anything
cargo update --dry-run                # 1.78+
cargo update -p serde --dry-run

# refresh the registry index without changing versions
cargo update --workspace              # touches everything; effectively a re-resolve
```

`cargo update` is the only command that should ever modify `Cargo.lock` outside of `cargo build`/`cargo add`. Always commit the result to source control.

## cargo install

Builds and installs a binary crate to `$CARGO_HOME/bin` (default `~/.cargo/bin`, which should be on `PATH`).

```bash
cargo install ripgrep                 # latest published version
cargo install ripgrep --version 14.1.0
cargo install ripgrep --locked        # use the crate's own Cargo.lock (deterministic, recommended)
cargo install --git https://github.com/BurntSushi/ripgrep
cargo install --git https://github.com/BurntSushi/ripgrep --tag 14.1.0
cargo install --git https://github.com/BurntSushi/ripgrep --branch master
cargo install --git https://github.com/BurntSushi/ripgrep --rev a1b2c3d
cargo install --path .                # install the local package
cargo install --path . --bin myapp    # specific bin from a multi-bin crate

# features
cargo install ripgrep --features 'pcre2'
cargo install ripgrep --no-default-features --features 'simd-accel'
cargo install ripgrep --all-features

# installation root
cargo install ripgrep --root /opt/rg  # binary lands in /opt/rg/bin/rg
cargo install ripgrep --target-dir /tmp/build  # where to build (clean up after)

# force re-install (e.g., after toolchain upgrade)
cargo install ripgrep --force
cargo install ripgrep -f --locked

# install only specific binaries
cargo install foo --bin foo-cli       # skip foo other bins
cargo install foo --bins              # all bins (default)

# list / uninstall
cargo install --list                  # what is installed
cargo uninstall ripgrep
cargo uninstall foo --bin foo-cli     # remove just one bin

# cross-compile install (rare)
cargo install ripgrep --target x86_64-unknown-linux-musl
```

`--locked` is critical for reproducible binary installs — without it cargo re-resolves the dep graph using the latest compatible versions, which has caused real-world breakage.

## cargo publish

Uploads to crates.io (or any registry).

```bash
# one-time auth
cargo login                           # paste API token from https://crates.io/me
# token is saved to ~/.cargo/credentials.toml
cargo logout

# verify before publishing
cargo publish --dry-run               # build + package + verify, but do not upload
cargo package                         # build a .crate file in target/package/
cargo package --list                  # what files would be included?
cargo package --no-verify             # skip the from-scratch rebuild step

# publish
cargo publish                         # actually upload
cargo publish --token $CRATES_IO_TOKEN
cargo publish --no-verify             # skip rebuild from packaged tarball (dangerous)
cargo publish --allow-dirty           # skip git "uncommitted changes" check
cargo publish --registry my-registry  # publish to alternate registry (see Custom Registries)

# yank / unyank a published version
cargo yank --version 0.2.1
cargo yank --version 0.2.1 --undo

# owners
cargo owner --add github:org:team mycrate
cargo owner --remove some-user mycrate
cargo owner --list mycrate
```

Required Cargo.toml fields for crates.io: `name`, `version`, `description`, `license` (or `license-file`), `repository` or `homepage`, plus a non-empty `README` is conventional.

## cargo workspace

A workspace is a set of related packages that share `Cargo.lock` and `target/`.

```bash
# root Cargo.toml — virtual workspace (no [package] section, just [workspace])
cat > Cargo.toml <<'EOF'
[workspace]
resolver = "2"                        # use the v2 feature resolver (required for edition 2021+)
members  = [
    "crates/core",
    "crates/cli",
    "crates/server",
    "crates/macros",
]
exclude = ["crates/scratch"]
default-members = ["crates/cli"]      # cargo run with no -p uses these

# inherited package metadata — DRY
[workspace.package]
version       = "0.4.2"
edition       = "2021"
rust-version  = "1.75"
license       = "MIT OR Apache-2.0"
authors       = ["Jane <jane@example.com>"]
repository    = "https://github.com/example/myproj"

# inherited dependencies — defined once, opted into per-crate
[workspace.dependencies]
serde       = { version = "1", features = ["derive"] }
tokio       = { version = "1.38", features = ["full"] }
clap        = { version = "4", features = ["derive"] }
anyhow      = "1"
thiserror   = "1"

# inherited lints
[workspace.lints.rust]
unsafe_code = "forbid"
[workspace.lints.clippy]
all = "deny"
pedantic = "warn"
EOF

# member Cargo.toml — opt into inheritance
cat > crates/cli/Cargo.toml <<'EOF'
[package]
name         = "mycli"
version.workspace      = true
edition.workspace      = true
rust-version.workspace = true
license.workspace      = true
repository.workspace   = true

[dependencies]
serde     = { workspace = true }
clap      = { workspace = true }
mycore    = { path = "../core" }      # in-workspace path dep

[lints]
workspace = true
EOF
```

Run cargo across the workspace:

```bash
cargo build --workspace               # every member
cargo test --workspace
cargo clippy --workspace --all-targets

# a single crate
cargo build -p mycli
cargo test -p mycore --lib

# default-members are used when -p is absent
cargo run                              # runs the default member's bin
```

A virtual workspace has only `[workspace]` at the root (no `[package]`). A non-virtual workspace has both — the root crate is itself a member.

The `resolver = "2"` setting is mandatory for new workspaces: without it, target-specific and dev-only feature unification can pull surprising features into your release builds.

## cargo audit and cargo deny

Security and policy gates. Both are external subcommands you `cargo install`.

```bash
# cargo-audit — scans Cargo.lock against the RustSec advisory DB
cargo install cargo-audit --locked

cargo audit                           # check for known vulnerable crates
cargo audit --json                    # JSON output for CI
cargo audit fix                       # apply trivial version bumps
cargo audit fix --dry-run
cargo audit --deny warnings           # any advisory = non-zero exit
cargo audit --ignore RUSTSEC-2023-0071  # ignore a specific advisory
cargo audit --db ~/.cargo/advisory-db  # custom DB path
cargo audit --stale                   # warn if local DB > 90 days old

# cargo-deny — license/source/advisory/ban policies
cargo install cargo-deny --locked

cargo deny init                       # generate deny.toml
cargo deny check                      # all checks
cargo deny check advisories
cargo deny check licenses
cargo deny check bans
cargo deny check sources

cat > deny.toml <<'EOF'
[graph]
all-features = true

[advisories]
db-urls          = ["https://github.com/RustSec/advisory-db"]
ignore           = []                 # advisory IDs to skip
yanked           = "deny"

[licenses]
allow            = ["MIT", "Apache-2.0", "BSD-3-Clause", "ISC", "Unicode-DFS-2016"]
confidence-threshold = 0.8

[bans]
multiple-versions = "warn"
deny             = [{ name = "openssl", version = "*" }]   # forbid openssl entirely
skip             = [{ name = "windows-sys" }]              # tolerate duplicates of these
skip-tree        = [{ name = "windows-targets" }]

[sources]
unknown-registry = "deny"
unknown-git      = "deny"
allow-git        = ["https://github.com/myorg/internal-crate"]
EOF
```

CI flow:

```bash
cargo audit --deny warnings && cargo deny check
```

## cargo expand

Expands declarative and procedural macros. Indispensable for debugging derive / `#[tokio::main]` / etc.

```bash
cargo install cargo-expand --locked
rustup component add rustfmt          # cargo-expand uses it to pretty-print

cargo expand                          # expand the default target (lib or main bin)
cargo expand --bin myapp
cargo expand --lib
cargo expand --test integration
cargo expand --example demo
cargo expand mymodule                 # only this module path
cargo expand --release                # release-cfg expansion
cargo expand --features tls
cargo expand --color always | less -R

# pipe to a file for diffing
cargo expand > /tmp/before.rs
# edit code...
cargo expand > /tmp/after.rs && diff -u /tmp/before.rs /tmp/after.rs
```

`cargo expand` requires the nightly toolchain to do its work (it uses `--pretty=expanded`), so the first run will offer to install nightly. You can be explicit:

```bash
cargo +nightly expand
```

## cargo udeps

Finds dependencies declared in `Cargo.toml` that are not actually used. Nightly only.

```bash
cargo install cargo-udeps --locked
rustup install nightly                # required

cargo +nightly udeps                  # check default
cargo +nightly udeps --all-targets
cargo +nightly udeps --workspace
cargo +nightly udeps --all-features
cargo +nightly udeps --backend depinfo  # alternative analysis backend (faster)
cargo +nightly udeps -p mycrate

# example output
# unused dependencies:
# `mycrate v0.1.0 (/repo/crates/mycrate)`
#     dependencies: "regex"
#     dev-dependencies: "tempfile"
```

Caveat: `cargo udeps` reports false positives for crates whose use is gated by a feature you did not enable. Always run with `--all-features` once before deleting a dep.

## cargo vendor

Downloads every dep into a `vendor/` directory and rewrites cargo to consume them — perfect for offline / air-gapped / reproducible builds.

```bash
cargo vendor                          # writes vendor/ + prints config snippet
cargo vendor third_party              # use a custom directory
cargo vendor --locked
cargo vendor --frozen
cargo vendor --no-delete              # do not wipe vendor/ first
cargo vendor --respect-source-config  # honor existing source replacements
cargo vendor --versioned-dirs         # crateA-1.2.3/ instead of crateA/

# the printed snippet — paste into .cargo/config.toml
cat >> .cargo/config.toml <<'EOF'
[source.crates-io]
replace-with = "vendored-sources"

[source.vendored-sources]
directory = "vendor"
EOF

# now this builds entirely offline
CARGO_NET_OFFLINE=true cargo build --offline
```

Vendoring is one-way: once `[source.crates-io]` is replaced, cargo will not hit the network even with `cargo update`. Re-run `cargo vendor` after any dep change.

## Cargo.toml — [package]

```bash
cat > Cargo.toml <<'EOF'
[package]
name         = "mycrate"              # required, [a-zA-Z0-9_-]
version      = "0.4.2"                # required, semver
edition      = "2021"                 # 2015 | 2018 | 2021 | 2024
rust-version = "1.75"                 # MSRV — cargo errors if rustc < this
authors      = ["Jane <jane@example.com>", "Joe <joe@example.com>"]
description  = "A short, plain-English description (required for crates.io)."
license      = "MIT OR Apache-2.0"    # SPDX expression
# license-file = "LICENSE.txt"        # alternative when SPDX is wrong
repository   = "https://github.com/example/mycrate"
homepage     = "https://example.com"
documentation = "https://docs.rs/mycrate"
readme       = "README.md"            # included in the published .crate
keywords     = ["cli", "tool", "fast", "no-deps", "embed"]   # max 5
categories   = ["command-line-utilities", "filesystem"]      # max 5; from crates.io list
include      = ["src/**", "Cargo.toml", "README.md", "LICENSE-*"]
exclude      = ["tests/big-fixtures/", "*.gif"]
build        = "build.rs"             # default; set to false to disable
links        = "ssl"                  # signals a native lib; enables links build-script metadata
publish      = false                  # set false to forbid cargo publish
publish      = ["my-registry"]        # whitelist registries
default-run  = "myapp"                # which bin cargo run picks
metadata     = { docs.rs.all-features = true }   # custom; ignored by cargo

# auto-discovery toggles
autobins     = true                   # default; false stops scanning src/bin/
autoexamples = true                   # default; false stops scanning examples/
autotests    = true
autobenches  = true

# explicit lib customization
[lib]
name        = "mycrate"               # default = sanitized package name
path        = "src/lib.rs"
crate-type  = ["lib"]                 # ["rlib"], ["cdylib"], ["staticlib"], ["dylib"], ["proc-macro"]
test        = true
doctest     = true
bench       = true
doc         = true
proc-macro  = false
harness     = true

# multiple binaries
[[bin]]
name = "myapp"
path = "src/main.rs"

[[bin]]
name = "mycli"
path = "src/bin/cli.rs"
required-features = ["cli"]           # only built when these features are on

[[example]]
name = "demo"
path = "examples/demo.rs"
required-features = ["serde"]

[[test]]
name = "smoke"
path = "tests/smoke.rs"
harness = true                        # set false for criterion-style custom main

[[bench]]
name    = "fib"
path    = "benches/fib.rs"
harness = false
EOF
```

## Cargo.toml — [dependencies]

```bash
cat >> Cargo.toml <<'EOF'
[dependencies]
# version requirements
once_cell  = "1.19"                   # caret: ^1.19 → >=1.19.0, <2.0.0
serde      = "^1"                     # explicit caret
regex      = "~1.10"                  # tilde: >=1.10.0, <1.11.0
exact_dep  = "=2.3.4"                 # exactly this version
wildcard   = "1.*"                    # NOT allowed for crates.io; works for path/git
range      = ">=1.2, <1.5"
multi_constraint = { version = ">=1.0, <2.0" }

# detailed table form
serde = { version = "1", features = ["derive", "rc"], default-features = false }

# optional dep — flips on with a feature
postgres = { version = "0.19", optional = true }

# rename so two versions can coexist OR a name conflict resolves
rand_old = { package = "rand", version = "0.7" }
rand_new = { package = "rand", version = "0.8" }

# git source
mylib = { git = "https://github.com/example/mylib.git" }
mylib = { git = "https://github.com/example/mylib.git", branch = "next" }
mylib = { git = "https://github.com/example/mylib.git", tag    = "v0.3.0" }
mylib = { git = "https://github.com/example/mylib.git", rev    = "9c8b7a6" }

# path source (in-workspace or sibling)
mycore = { path = "../core" }

# git + version (the version is for crates.io fallback when git is missing)
mylib = { git = "https://...", version = "0.3" }

# alternate registry
mycrate = { version = "1", registry = "my-registry" }

# workspace inheritance — pull from [workspace.dependencies]
serde = { workspace = true }
serde = { workspace = true, features = ["derive"] }   # add features locally
EOF
```

Version requirement quick reference:

```bash
"1.2.3"   = ^1.2.3   = >=1.2.3, <2.0.0   (caret)
"1.2"     = ^1.2     = >=1.2.0, <2.0.0
"1"       = ^1       = >=1.0.0, <2.0.0
"0.2.3"   = ^0.2.3   = >=0.2.3, <0.3.0   (0.x → minor breaks)
"0.0.3"   = ^0.0.3   = >=0.0.3, <0.0.4   (0.0.x → patch breaks)
"~1.2.3"             = >=1.2.3, <1.3.0
"~1.2"               = >=1.2.0, <1.3.0
"~1"                 = >=1.0.0, <2.0.0
"=1.2.3"             = exactly 1.2.3
">=1.2, <1.5"        = range
"*"                  = any (BANNED on crates.io)
```

## Cargo.toml — [dev-dependencies], [build-dependencies], [target...]

```bash
cat >> Cargo.toml <<'EOF'
# only compiled for tests, examples, benches, doctests — NOT linked into the lib/bin
[dev-dependencies]
criterion = "0.5"
proptest  = "1"
tempfile  = "3"
mockito   = "1"
tokio     = { version = "1", features = ["full", "test-util"] }

# only compiled for build.rs
[build-dependencies]
cc       = "1"                        # compile C/C++ from build.rs
bindgen  = "0.69"                     # auto-gen Rust FFI from C headers
prost-build = "0.12"

# target-specific deps — same syntax as cfg() attributes
[target.'cfg(unix)'.dependencies]
nix = "0.27"

[target.'cfg(windows)'.dependencies]
winapi = { version = "0.3", features = ["winuser"] }

[target.'cfg(target_os = "linux")'.dependencies]
inotify = "0.10"

[target.'cfg(any(target_os = "linux", target_os = "macos"))'.dependencies]
libc = "0.2"

[target.'cfg(target_arch = "wasm32")'.dependencies]
wasm-bindgen = "0.2"

# pinned-triple form
[target.x86_64-unknown-linux-musl.dependencies]
ring = "0.17"
EOF
```

`[dev-dependencies]` are NOT part of your crate's public dep tree — downstream users never see them. Use this for test fixtures and example deps that should not bloat consumers.

## Cargo.toml — [features]

```bash
cat >> Cargo.toml <<'EOF'
[dependencies]
serde     = { version = "1", optional = true }
postgres  = { version = "0.19", optional = true }
mysql     = { version = "24", optional = true }
tracing   = { version = "0.1", default-features = false, optional = true }

[features]
default = ["std"]                     # auto-enabled unless --no-default-features
std     = []                          # named feature, no deps it needs to enable
serde   = ["dep:serde"]               # ENABLE the optional dep `serde` (modern syntax)
postgres = ["dep:postgres", "std"]    # postgres pulls in std too
mysql    = ["dep:mysql", "std"]
all-dbs  = ["postgres", "mysql"]      # umbrella feature
log      = ["dep:tracing"]            # rename a feature relative to its dep

# weak feature — only forwards if dep is already on
serde-extra = ["postgres?/serde"]     # if postgres is enabled, turn on its serde feature

# enable a feature on a non-optional dep
nightly = ["serde/derive"]            # NOT a dep: prefix because serde is not optional here
EOF
```

Activate features on the command line:

```bash
cargo build --features serde
cargo build --features "serde postgres"
cargo build --features serde,postgres
cargo build -F serde -F postgres
cargo build --no-default-features
cargo build --no-default-features --features std,serde
cargo build --all-features
```

### Feature rules and gotchas

- Features are additive. Two consumers of your crate may turn on different features; cargo unifies them. Never make features mutually exclusive — use separate crates if you need exclusivity.
- `dep:foo` syntax (rust 1.60+) lets you have a feature named `foo` without auto-creating a feature `foo` for the optional dep. Without `dep:`, declaring `foo = { optional = true }` implicitly creates a `foo` feature.
- `?/` (weak dep): `myfeat = ["serde?/derive"]` adds the `derive` feature to `serde` only if `serde` is already enabled by something else.
- `/` (strong): `myfeat = ["serde/derive"]` turns ON `serde` AND its `derive` feature.
- No `default = []` is the same as omitting the line. The line is for clarity.
- The mutually-exclusive feature anti-pattern — `runtime-tokio` and `runtime-async-std` that conflict at compile time. Cargo cannot enforce this. The fix is to gate at runtime or split the crate. If you must, panic at compile time:

```bash
# in src/lib.rs
# #[cfg(all(feature = "runtime-tokio", feature = "runtime-async-std"))]
# compile_error!("only one of runtime-tokio / runtime-async-std may be enabled");
```

Inspect what features resolved to:

```bash
cargo metadata --format-version=1 | jq '.resolve.nodes[] | select(.id|test("mycrate")) | .features'
cargo tree -e features --features "tls postgres"
```

## Cargo.toml — [profile.*]

Profiles control compile flags. Built-ins: `dev` (`cargo build`), `release` (`cargo build --release`), `test`, `bench`. You can define custom profiles too.

```bash
cat >> Cargo.toml <<'EOF'
[profile.dev]
opt-level         = 0                 # 0=none, 1, 2, 3, "s"=size, "z"=more size
debug             = true              # full DWARF (or 0|1|2|"line-tables-only"|"line-directives-only")
debug-assertions  = true              # debug_assert! enabled
overflow-checks   = true              # arithmetic overflow → panic
lto               = false             # no link-time optimization
codegen-units     = 256               # parallelism, but worse opts
panic             = "unwind"          # "unwind" | "abort"
incremental       = true              # speed up rebuilds
strip             = "none"            # "none" | "symbols" | "debuginfo"
split-debuginfo   = "off"             # "off" | "packed" | "unpacked" (mac/linux: unpacked is default)

[profile.release]
opt-level         = 3
debug             = false
debug-assertions  = false
overflow-checks   = false
lto               = false             # set to "thin" or true ("fat") for max perf
codegen-units     = 16                # 1 = best opt, slowest compile
panic             = "unwind"
incremental       = false
strip             = "none"            # set "symbols" for smaller binaries

# the typical "ship a small fast binary" preset
[profile.dist]
inherits          = "release"
lto               = "fat"             # full LTO; very slow link, smallest+fastest output
codegen-units     = 1
strip             = "symbols"
panic             = "abort"           # smaller, no unwinding tables — tradeoff: no panic catching

[profile.test]
opt-level         = 0                 # mirrors dev
debug             = true
debug-assertions  = true
overflow-checks   = true

[profile.bench]
opt-level         = 3                 # mirrors release
debug             = false
debug-assertions  = false
overflow-checks   = false
lto               = false
codegen-units     = 16

# tune ONE dep's profile (rarely needed; e.g., ring is slow to compile in debug)
[profile.dev.package.ring]
opt-level = 3

[profile.dev.package."*"]             # all transitive deps
opt-level = 1
EOF

# build a custom profile
cargo build --profile dist
```

Profile flag cheat sheet:

```bash
opt-level
   0  no optimization (default for dev)
   1  basic
   2  good
   3  full (default for release)
   s  optimize for size
   z  more aggressively for size

lto
   false  off (default)
   "thin" thin LTO; fast, ~20% slower than fat
   true   == "fat"; full cross-crate LTO; very slow link

codegen-units
   N >= 1; lower = better optimization, longer compile

panic
   "unwind" allows std::panic::catch_unwind, slightly larger
   "abort"  stops the process on panic, smaller binary, faster
```

## Cargo.toml — [patch] / [replace]

Replace a dep with a different source — without forking every consumer.

```bash
cat >> Cargo.toml <<'EOF'
# patch a single registry version with a fork or local copy
[patch.crates-io]
serde = { git = "https://github.com/example/serde", branch = "fix-bug-1234" }
mylib = { path = "../mylib-fork" }

# patch a git dep
[patch."https://github.com/some/repo"]
some-crate = { path = "../some-crate" }

# patch a custom registry
[patch.my-registry]
internal = { path = "../internal" }

# [replace] is the older form — semi-deprecated in favor of [patch]
[replace]
"serde:1.0.197" = { git = "https://github.com/example/serde" }
EOF
```

`[patch]` only works at the top level of the crate or workspace you control — patches in dependencies are ignored. The patched version must be semver-compatible with the version it replaces, otherwise cargo errors:

```bash
# error: failed to resolve patches for `https://github.com/rust-lang/crates.io-index`
# Caused by:
#   patch for `serde` in ... did not resolve to any crates.
```

Fix: bump the version in the patch source, or change the version requirement in `[dependencies]`.

## Cargo.lock semantics

`Cargo.lock` records the exact resolved version + source + checksum of every dep in the graph. Cargo only writes it; never read it by hand to determine availability.

```bash
# when to commit Cargo.lock
#   binary crates / applications / workspaces with bins:  YES, always commit
#   pure libraries published to crates.io:                Conventionally NO
#                                                          (consumers re-resolve anyway,
#                                                          but committing it gives reproducible CI;
#                                                          the modern recommendation is YES, commit it.)
echo Cargo.lock >> .gitignore                # only if you really mean to (libs only)

# generate or refresh Cargo.lock without compiling
cargo generate-lockfile
cargo update --workspace                     # re-resolve

# CI flags
cargo build --locked                         # error if Cargo.lock is missing OR would change
cargo build --frozen                         # --locked + --offline (no network)

# inspect
head -20 Cargo.lock
# # This file is automatically @generated by Cargo.
# # It is not intended for manual editing.
# version = 4
# [[package]]
# name = "anyhow"
# version = "1.0.86"
# source = "registry+https://github.com/rust-lang/crates.io-index"
# checksum = "b3d1d046238990b9cf5bcde22a3fb3584ee5cf65fb2765f454ed428c7a0063da"

# lockfile-version
#   1 — old default
#   3 — rust 1.53+
#   4 — rust 1.78+ (current)
# you do not normally touch this; cargo upgrades on its own

# resolve a merge conflict in Cargo.lock — never hand-edit
git checkout --ours Cargo.lock && cargo build       # accept ours, then re-resolve
git checkout --theirs Cargo.lock && cargo build
# either way, finish with `cargo build` so the lockfile is valid
```

## Conditional compilation — #[cfg(...)]

```bash
# in source files, gate items by predicate
# #[cfg(unix)]                                 fn unixonly() {}
# #[cfg(windows)]                              fn winonly() {}
# #[cfg(target_os = "linux")]                  fn lin() {}
# #[cfg(target_os = "macos")]                  fn mac() {}
# #[cfg(target_arch = "x86_64")]               fn x64() {}
# #[cfg(target_arch = "aarch64")]              fn arm64() {}
# #[cfg(target_pointer_width = "64")]
# #[cfg(target_endian = "little")]
# #[cfg(target_env = "musl")]                  // gnu | musl | msvc | sgx
# #[cfg(target_vendor = "apple")]
# #[cfg(target_feature = "avx2")]              // CPU features
# #[cfg(feature = "tls")]                      fn tls_path() {}
# #[cfg(test)]                                 mod tests {}
# #[cfg(debug_assertions)]                     fn slow_check() {}
# #[cfg(not(target_os = "windows"))]
# #[cfg(any(unix, target_os = "redox"))]
# #[cfg(all(unix, target_arch = "x86_64"))]
# #[cfg(all(feature = "tls", any(unix, windows)))]

# cfg_attr — apply another attribute conditionally
# #[cfg_attr(feature = "serde", derive(serde::Serialize))]
# #[cfg_attr(target_os = "linux", path = "linux.rs")]
# mod platform;

# cfg!() — runtime boolean (the cfg is still resolved at compile time;
# both branches type-check, only one survives DCE)
# if cfg!(target_os = "linux") { /* ... */ }

# the cfg() macro returns nothing — only used for #[cfg]
```

Custom cfg keys (e.g., from build.rs `cargo:rustc-cfg=...`) must be declared in Cargo.toml on rust 1.80+:

```bash
cat >> Cargo.toml <<'EOF'
[lints.rust]
unexpected_cfgs = { level = "warn", check-cfg = ['cfg(has_foo)', 'cfg(myproduct, values("a","b"))'] }
EOF
```

Otherwise rustc emits `unexpected cfg condition name` warnings.

## Cross-compilation

```bash
# 1. install the target
rustup target add x86_64-unknown-linux-musl
rustup target add aarch64-apple-darwin
rustup target add wasm32-unknown-unknown
rustup target add x86_64-pc-windows-gnu

# 2. install a linker for the target (sometimes needed)
# linux musl on linux:        sudo apt install musl-tools
# arm64 linux on linux:       sudo apt install gcc-aarch64-linux-gnu
# windows on linux/macos:     mingw-w64 (apt: gcc-mingw-w64) or use --target *-windows-msvc + xwin
# anything to mac:             xcrun is required; cross-mac is officially unsupported by Apple

# 3. configure the linker per-target in .cargo/config.toml
mkdir -p .cargo && cat > .cargo/config.toml <<'EOF'
[target.aarch64-unknown-linux-gnu]
linker = "aarch64-linux-gnu-gcc"

[target.x86_64-unknown-linux-musl]
linker = "musl-gcc"
rustflags = ["-C", "target-feature=+crt-static"]

[target.x86_64-pc-windows-gnu]
linker = "x86_64-w64-mingw32-gcc"
ar     = "x86_64-w64-mingw32-ar"

# build for Apple Silicon from an Intel mac (requires Xcode)
[target.aarch64-apple-darwin]
linker = "clang"
rustflags = ["-C", "link-arg=-target", "-C", "link-arg=arm64-apple-macosx11.0"]

# default target if cargo is invoked with no --target
[build]
target = "x86_64-unknown-linux-musl"
EOF

# 4. build
cargo build --release --target x86_64-unknown-linux-musl
cargo build --release --target wasm32-unknown-unknown
ls target/x86_64-unknown-linux-musl/release/

# the cross crate — opinionated cross-compile via Docker
cargo install cross --git https://github.com/cross-rs/cross --locked
cross build --target aarch64-unknown-linux-musl --release
cross test  --target x86_64-pc-windows-gnu
# requires docker or podman; pulls a pre-built sysroot image per target
```

Cross-compiling C deps via FFI (e.g., `openssl-sys`) usually fails without extra env vars. Either use `cross`, or vendor pure-Rust alternatives (`rustls` instead of `openssl`, `ring`, etc.).

## Build scripts

`build.rs` runs BEFORE the crate compiles. Use it to generate code, link native libs, query env, or detect features.

```bash
cat > build.rs <<'EOF'
use std::{env, path::PathBuf, process::Command};

fn main() {
    // Tell cargo when to re-run this script
    println!("cargo:rerun-if-changed=build.rs");
    println!("cargo:rerun-if-changed=src/proto/");
    println!("cargo:rerun-if-env-changed=MY_FLAG");

    // Link a system library
    println!("cargo:rustc-link-lib=dylib=ssl");
    println!("cargo:rustc-link-lib=static=mylib");
    println!("cargo:rustc-link-lib=framework=CoreFoundation"); // mac

    // Add a search path for the linker
    println!("cargo:rustc-link-search=native=/opt/mylib/lib");
    println!("cargo:rustc-link-search=framework=/Library/Frameworks");

    // Pass extra flags to rustc for THIS crate
    println!("cargo:rustc-flags=-l dylib=z");
    println!("cargo:rustc-link-arg=-Wl,-rpath,/opt/mylib/lib");
    println!("cargo:rustc-cdylib-link-arg=-Wl,--export-dynamic");

    // Set a custom cfg flag (must be declared in [lints.rust].check-cfg on 1.80+)
    println!("cargo:rustc-cfg=has_avx2");

    // Set an env var visible to the crate via env!()
    println!("cargo:rustc-env=GIT_HASH={}", git_sha());

    // Emit a warning during the build
    println!("cargo:warning=using fallback C lib");

    // Generate code into OUT_DIR
    let out_dir = env::var("OUT_DIR").unwrap();
    let out_path = PathBuf::from(out_dir).join("generated.rs");
    std::fs::write(out_path, "pub const VERSION: u32 = 1;").unwrap();
}

fn git_sha() -> String {
    Command::new("git")
        .args(["rev-parse", "--short=12", "HEAD"])
        .output()
        .ok()
        .and_then(|o| String::from_utf8(o.stdout).ok())
        .map(|s| s.trim().to_string())
        .unwrap_or_else(|| "unknown".to_string())
}
EOF

# in src/lib.rs, include the generated file
# include!(concat!(env!("OUT_DIR"), "/generated.rs"));

# in any source file, read the env var
# const SHA: &str = env!("GIT_HASH");
```

### Build script env vars (set by cargo for build.rs)

```bash
OUT_DIR                 # writable scratch dir for generated code
CARGO                   # path to the cargo binary
CARGO_MANIFEST_DIR      # the package's source root
CARGO_PKG_NAME
CARGO_PKG_VERSION
CARGO_PKG_VERSION_MAJOR
CARGO_PKG_VERSION_MINOR
CARGO_PKG_VERSION_PATCH
CARGO_PKG_AUTHORS
CARGO_PKG_DESCRIPTION
CARGO_PKG_REPOSITORY
CARGO_FEATURE_<NAME>    # for each enabled feature, in SCREAMING_SNAKE_CASE
CARGO_CFG_TARGET_OS
CARGO_CFG_TARGET_ARCH
CARGO_CFG_TARGET_FAMILY
CARGO_CFG_TARGET_ENV
CARGO_CFG_TARGET_VENDOR
CARGO_CFG_UNIX
CARGO_CFG_WINDOWS
HOST                    # the host triple
TARGET                  # the build target triple
NUM_JOBS                # value of -j
OPT_LEVEL               # numeric or "s"/"z"
PROFILE                 # "debug" or "release"
DEBUG                   # "true" or "false"
RUSTC                   # path to rustc
RUSTDOC
RUSTC_WRAPPER           # if set
LINKER
```

### Build script cargo: directives quick reference

```bash
cargo:rerun-if-changed=PATH        # only re-run if PATH changed
cargo:rerun-if-env-changed=NAME    # only re-run if env var NAME changed
cargo:rustc-link-lib=KIND=NAME     # KIND in {dylib, static, framework}
cargo:rustc-link-search=KIND=PATH  # KIND in {dependency, crate, native, framework, all}
cargo:rustc-link-arg=ARG           # raw -C link-arg=ARG for this crate
cargo:rustc-link-arg-bin=NAME=ARG  # only for one bin
cargo:rustc-link-arg-bins=ARG      # only for bin targets
cargo:rustc-flags=FLAGS            # -L PATH or -l NAME, restricted set
cargo:rustc-cfg=KEY                # set #[cfg(KEY)]
cargo:rustc-env=NAME=VALUE         # set env var visible at compile time
cargo:rustc-cdylib-link-arg=ARG    # link arg for cdylib only
cargo:warning=MESSAGE              # print a warning
cargo:metadata=KEY=VALUE           # passed to dependents that link this links lib
```

## Custom registries

Cargo can publish to and consume from private/internal registries.

```bash
# .cargo/config.toml — declare the registry
mkdir -p .cargo && cat > .cargo/config.toml <<'EOF'
[registries.my-registry]
index = "sparse+https://my-registry.example.com/api/v1/crates/"   # sparse protocol (recommended)
# index = "https://github.com/myorg/crate-index"                 # legacy git index
token = "..."                                                     # usually in credentials.toml instead

[source.crates-io]
# replace-with = "my-registry"   # to make my-registry the default
EOF

# auth — separate file so config.toml stays public
cat > ~/.cargo/credentials.toml <<'EOF'
[registries.my-registry]
token = "Bearer xxx"
EOF
chmod 600 ~/.cargo/credentials.toml

# publish to it
cargo publish --registry my-registry

# depend on a crate from it
# in Cargo.toml:
# [dependencies]
# internal = { version = "1.2", registry = "my-registry" }

# allow publishing this crate ONLY to specific registries
cat >> Cargo.toml <<'EOF'
[package]
publish = ["my-registry"]
EOF
```

### Sparse vs git index

- Sparse (cargo 1.68+, default for crates.io since 1.70): plain HTTPS, fetches only the index files for crates you need. Massively faster than git for huge registries.
- Git index (legacy): cargo clones the entire registry-index git repo. Slower but works on every cargo version.

Force sparse for crates.io if you are on an older cargo:

```bash
# .cargo/config.toml
[registries.crates-io]
protocol = "sparse"

# or env var
export CARGO_REGISTRIES_CRATES_IO_PROTOCOL=sparse
```

## Common errors (verbatim text → fix)

```bash
# error: could not find `Cargo.toml` in `/some/path` or any parent directory
# Cause:  not in a cargo package
# Fix:    cd into a directory that has Cargo.toml or run `cargo init`
cd /path/to/repo

# error: the lock file Cargo.lock needs to be updated but --locked was passed to prevent this
#   If you want to try to generate the lock file without accessing the network, remove the --locked flag and use --offline instead.
# Cause:  Cargo.toml changed since Cargo.lock was generated; --locked forbids changes
# Fix:    drop --locked while updating, then commit the new Cargo.lock
cargo update -p the-changed-dep
cargo build --locked

# error: failed to select a version for the requirement `foo = "^2.0"`
#   candidate versions found which did not match: 1.5.3, 1.5.2, 1.5.1
#   location searched: crates.io index
#   required by package `mycrate v0.1.0 ...`
# Cause:  no published version satisfies the requirement
# Fix:    relax the version: foo = "1"   OR   ensure the version exists on the registry
cargo search foo
cargo update -p foo --precise 1.5.3

# error[E0658]: use of unstable library feature 'X'
#  --> src/lib.rs:5:5
#   |
# 5 |     std::sync::Once::new();
#   |     ^^^^^^^^^^^^^^^
#   = note: see issue #N for more information
#   = help: add `#![feature(X)]` to the crate attributes to enable
# Cause:  using a nightly-only feature on stable
# Fix:    switch to nightly OR find a stable replacement OR pin to nightly
cargo +nightly build
echo "[toolchain]" > rust-toolchain.toml; echo 'channel = "nightly"' >> rust-toolchain.toml

# error: linker `cc` not found
#   |
#   = note: No such file or directory (os error 2)
# Cause:  no system C linker installed
# Fix (debian/ubuntu):  sudo apt install build-essential
# Fix (fedora):         sudo dnf groupinstall "Development Tools"
# Fix (alpine):         apk add build-base
# Fix (mac):            xcode-select --install
# Fix (windows-msvc):   install Visual Studio C++ build tools

# error[E0463]: can't find crate for `core`
#   = note: the `wasm32-unknown-unknown` target may not be installed
# Cause:  rustc has no std for the requested target
# Fix:    rustup target add wasm32-unknown-unknown

# error: failed to run custom build command for `openssl-sys v0.9.x`
#   Caused by: process did not exit successfully: `... build-script-build` (exit status: 101)
#   --- stderr
#   Could not find directory of OpenSSL installation
# Cause:  openssl-sys can not find headers/libs at build time
# Fix (apt):    sudo apt install pkg-config libssl-dev
# Fix (mac):    brew install openssl pkg-config
#               export OPENSSL_DIR=$(brew --prefix openssl)
# Fix (avoid):  swap to a pure-Rust TLS impl (rustls) and remove openssl-sys

# error[E0432]: unresolved import `serde::Serialize`
#  --> src/lib.rs:1:5
#   |
# 1 | use serde::Serialize;
#   |     ^^^^^ no `Serialize` in the root
# Cause:  the `derive` feature is not enabled on serde
# Fix:    serde = { version = "1", features = ["derive"] }
cargo add serde --features derive

# warning: unused import: `std::io::Read`
#  --> src/main.rs:1:5
# Cause:  dead import
# Fix:    delete the use OR allow it: #![allow(unused_imports)]
# Escalation:  -D warnings turns this into a compile error
# RUSTFLAGS="-D warnings" cargo build
RUSTFLAGS="-D warnings" cargo build

# error: no `Cargo.toml` file found in `/repo`, but found virtual workspace at `/repo`
# Cause:  workspace root has [workspace] but no [package], so cargo run/build needs -p
# Fix:    cargo build -p some-member   OR   add `default-members` in [workspace]
cargo build -p mycli

# error: failed to compile `mycrate v0.1.0`, intermediate artifacts can be found at `target/`
# (generic catch-all — scroll up for the real error)

# error: package `mycrate v0.1.0` cannot be built because it requires rustc 1.80 or newer
# Cause:  Cargo.toml has rust-version = "1.80" but the active toolchain is older
# Fix:    rustup update    OR    pin via rust-toolchain.toml

# error: feature `Z` is required
#   The package requires the Cargo feature called `Z`, but that feature is not stabilized in this version of Cargo.
# Cause:  Cargo.toml uses cargo-features = ["Z"] which is nightly-only
# Fix:    cargo +nightly build

# error: 2 jobs failed:
#   `cargo:rustc-link-lib=foo` cannot be used with non-`links` package
# Cause:  build.rs requested a link directive but Cargo.toml lacks `links = "foo"`
# Fix:    [package] links = "foo"
```

## Common Gotchas

### 1. cargo build does not update Cargo.lock; cargo update does

```bash
# BROKEN: bumping a version in Cargo.toml without rebuilding silently uses old lockfile pin
# Cargo.toml changed:  serde = "1.0.200"
# Cargo.lock still pinned to 1.0.150
cargo build           # cargo will re-resolve when Cargo.toml's requirement is no longer
                      # satisfied by the lockfile, but partial bumps confuse people.

# FIXED: explicit refresh
cargo update -p serde
cargo build
```

### 2. Forgetting --release makes profiling lies

```bash
# BROKEN: benchmarking debug code
cargo run -- --benchmark        # 100x slower than reality, wastes hours

# FIXED: always profile release
cargo run --release -- --benchmark
cargo build --release && time ./target/release/myapp
```

### 3. --features quoting eats the second feature

```bash
# BROKEN: bash splits on space, cargo sees two args
cargo build --features tls postgres
# error: Found argument 'postgres' which was not expected

# FIXED: quote OR comma-separate OR repeat -F
cargo build --features "tls postgres"
cargo build --features tls,postgres
cargo build -F tls -F postgres
```

### 4. Mutually-exclusive features are an anti-pattern

```bash
# BROKEN: Cargo.toml
# [features]
# tls-rustls   = ["dep:rustls"]
# tls-openssl  = ["dep:openssl"]
# Both turn on simultaneously when one consumer enables each → double-includes,
# duplicate symbols, bizarre runtime behavior. Cargo will NOT error.

# FIXED: one feature for "tls", a runtime config or build.rs that picks an impl;
# OR split into two crates: mylib-rustls, mylib-openssl
```

### 5. Dev-dependencies leak into production via cyclic features

```bash
# BROKEN: declaring a normal dep AND a dev-dep on the same crate with different features
# [dependencies]      tokio = { version = "1", features = ["rt"] }
# [dev-dependencies]  tokio = { version = "1", features = ["full", "test-util"] }
# With resolver 1, the unification leaks "full" + "test-util" into prod. Resolver 2 fixes this.

# FIXED: use resolver = "2" in [workspace] or [package]
cat >> Cargo.toml <<'EOF'
[workspace]
resolver = "2"
EOF
```

### 6. cargo install without --locked re-resolves and may break

```bash
# BROKEN: a recently-yanked transitive dep, or a new minor with a bug, slips in
cargo install ripgrep

# FIXED: always pass --locked for tools
cargo install ripgrep --locked
cargo install --git https://github.com/example/tool --locked
```

### 7. Path deps with version confuse cargo publish

```bash
# BROKEN: in a workspace, you depend on a sibling by path only
# Cargo.toml:  mylib = { path = "../mylib" }
# `cargo publish` for the parent will fail because path-only deps cannot be on crates.io.

# FIXED: include both a version AND path. Path is used in-workspace; version is used after publish.
# mylib = { path = "../mylib", version = "0.3.0" }
```

### 8. cfg(test) is per-CRATE, not per-workspace

```bash
# BROKEN: a sibling crate `mylib` puts test helpers behind #[cfg(test)],
# then crate `myapp` tries to use them in its OWN tests. They are invisible —
# `cfg(test)` is only true while compiling the crate that defined it.

# FIXED: gate behind a feature instead
# in mylib/Cargo.toml:
# [features]
# test-utils = []
# in mylib/src/lib.rs:
# #[cfg(any(test, feature = "test-utils"))]
# pub mod testing { /* helpers */ }
# in myapp/Cargo.toml:
# [dev-dependencies]
# mylib = { path = "../mylib", features = ["test-utils"] }
```

### 9. Cargo.lock not committed for an application

```bash
# BROKEN: deploying an app whose CI uses `cargo build` without a committed Cargo.lock
#         every build resolves fresh, "works on my machine", random regressions
# .gitignore had:  Cargo.lock

# FIXED: commit Cargo.lock for binaries; use `cargo build --locked` in CI
git rm --cached Cargo.lock 2>/dev/null; true   # if tracked
sed -i.bak '/^Cargo.lock$/d' .gitignore
git add Cargo.lock && git commit -m "Commit Cargo.lock for reproducible builds"
```

### 10. cargo test runs nothing because the filter eats everything

```bash
# BROKEN: typoed module path
cargo test parser::parses_csv     # no test matches → "running 0 tests" (no error)

# FIXED: list first
cargo test -- --list | grep parser
cargo test -- --exact parser::parses_csv
```

### 11. --all-features is dangerous in workspaces

```bash
# BROKEN: cargo build --workspace --all-features
# turns on every feature in every member, including mutually-incompatible ones
# (e.g., one member has feature `runtime-tokio`, another has `runtime-async-std`)

# FIXED: enable specific feature sets per CI matrix entry
cargo test -p mycore --features "tls"
cargo test -p mycore --no-default-features --features "blocking"
```

### 12. cargo clippy --fix refuses to run on a dirty tree

```bash
# BROKEN
cargo clippy --fix
# error: the working directory of this package has uncommitted changes,
#        and `cargo clippy --fix` can potentially perform destructive changes;
#        if you would like to suppress this error pass `--allow-dirty`

# FIXED: commit first, OR pass --allow-dirty
git stash && cargo clippy --fix && git stash pop
cargo clippy --fix --allow-dirty --allow-staged
```

## Idioms

### The check-loop / build-once flow

```bash
# tight inner loop while writing code
cargo check --workspace --all-targets       # 1-3s once warm
# editor on save runs the same; rust-analyzer pipes diagnostics in

# only `cargo build` when you need to RUN something
cargo run -- --help

# only `cargo build --release` once before you ship / benchmark
cargo build --release && time ./target/release/myapp benchmark
```

### Always --locked for installed tools

```bash
cargo install ripgrep --locked
cargo install cargo-watch --locked
cargo install cargo-audit --locked
cargo install just --locked

# script form
for t in ripgrep fd-find tokei hyperfine; do
    cargo install "$t" --locked
done
```

### cargo watch for autobuild on save

```bash
cargo install cargo-watch --locked
cargo watch -x check                     # cargo check on every file save
cargo watch -x 'check --all-targets'
cargo watch -x 'test --lib'
cargo watch -x 'clippy --all-targets'
cargo watch -x 'run -- --port 8080'
cargo watch -c -x check                  # -c clears the screen
cargo watch -s 'cargo check && cargo test'
```

### cargo nextest — faster test runner

```bash
cargo install cargo-nextest --locked
cargo nextest run                         # parallel, isolated, machine-readable
cargo nextest run --workspace --all-features
cargo nextest run --no-fail-fast
cargo nextest list
# nextest does not yet run doctests — pair with: cargo test --doc
```

### Adding/removing/upgrading deps with cargo add / cargo remove / cargo upgrade

```bash
# cargo add (built-in since 1.62)
cargo add serde
cargo add serde --features derive,rc
cargo add serde@1                          # specific req
cargo add anyhow thiserror                 # multiple
cargo add --dev tempfile
cargo add --build cc
cargo add --target 'cfg(unix)' nix
cargo add --git https://github.com/example/lib --branch main
cargo add --path ../mylib
cargo add --rename oldserde --package serde --features derive
cargo add tokio --no-default-features --features rt,macros

cargo remove serde                         # built-in since 1.66
cargo remove --dev tempfile

# cargo-edit external crate adds `cargo upgrade`
cargo install cargo-edit --locked
cargo upgrade                              # bump Cargo.toml requirements to latest
cargo upgrade --workspace
cargo upgrade --incompatible              # also do major-version bumps
cargo upgrade -p serde --to 1.0.200
cargo upgrade --dry-run
```

### Quoting features in scripts

```bash
# in a Makefile (use single quotes; tabs not spaces inside recipes)
build:
	cargo build --release --features 'tls postgres redis'

# in a justfile
build features='tls':
    cargo build --release --features '{{features}}'

# in shell with array (preferred for complex flags)
features=(--features tls --features postgres --no-default-features)
cargo build "${features[@]}"
```

### RUST_LOG, RUST_BACKTRACE, CARGO_LOG

```bash
# trace logs from `tracing` / `env_logger`
RUST_LOG=debug cargo run
RUST_LOG=mycrate=trace,hyper=info cargo run
RUST_LOG=trace cargo test -- --nocapture

# stack traces on panic
RUST_BACKTRACE=1 cargo run
RUST_BACKTRACE=full cargo test

# debug cargo itself
CARGO_LOG=trace cargo build
CARGO_HTTP_DEBUG=true cargo build           # dump HTTP requests to the registry
```

### Build acceleration

```bash
# 1. sccache — caches rustc output across projects
cargo install sccache --locked
export RUSTC_WRAPPER=sccache
sccache --show-stats

# 2. mold linker (linux) or lld (everywhere) — much faster than ld
sudo apt install mold        # or build from source
mkdir -p .cargo && cat > .cargo/config.toml <<'EOF'
[target.x86_64-unknown-linux-gnu]
linker = "clang"
rustflags = ["-C", "link-arg=-fuse-ld=mold"]
EOF

# 3. cranelift backend (faster debug builds, nightly)
rustup component add rustc-codegen-cranelift-preview --toolchain nightly
cargo +nightly build -Z codegen-backend=cranelift

# 4. share target/ across projects
cat >> ~/.cargo/config.toml <<'EOF'
[build]
target-dir = "/tmp/cargo-target"
EOF
```

### Reproducible release builds

```bash
# Cargo.toml — the small/fast preset
# [profile.release]
# strip          = "symbols"
# lto            = "fat"
# codegen-units  = 1
# panic          = "abort"
# opt-level      = "z"      # or 3 if speed > size

# command line — pin everything
cargo build --release --locked --frozen --target x86_64-unknown-linux-musl

# verify reproducibility
sha256sum target/x86_64-unknown-linux-musl/release/myapp
```

### CI snippet (GitHub Actions)

```bash
# .github/workflows/ci.yml fragments
# - uses: dtolnay/rust-toolchain@stable
#   with:
#     components: rustfmt, clippy
# - run: cargo fmt --all -- --check
# - run: cargo clippy --workspace --all-targets --all-features -- -D warnings
# - run: cargo test --workspace --all-features --locked
# - run: cargo build --workspace --release --locked
# - run: cargo audit --deny warnings
# - run: cargo deny check
```

### Aliases via .cargo/config.toml

```bash
cat >> ~/.cargo/config.toml <<'EOF'
[alias]
b   = "build"
br  = "build --release"
c   = "check --all-targets"
cw  = "check --workspace --all-targets"
t   = "test"
tw  = "test --workspace"
r   = "run"
rr  = "run --release"
xt  = "test --workspace --all-features -- --nocapture"
fix = "clippy --fix --allow-dirty --allow-staged --workspace --all-targets"
ci  = "test --workspace --all-features --locked"
EOF

cargo b              # cargo build
cargo cw             # cargo check --workspace --all-targets
cargo fix            # cargo clippy --fix --allow-dirty --allow-staged ...
```

### Inspecting the resolved dep graph

```bash
cargo metadata --format-version=1 | jq '.packages | length'           # number of crates
cargo metadata --format-version=1 | jq '.packages[].name' | sort -u
cargo metadata --no-deps --format-version=1 | jq '.workspace_members'
cargo metadata --format-version=1 | jq '.resolve.nodes[].id' | wc -l

# what features did each crate end up with?
cargo metadata --format-version=1 | jq '.resolve.nodes[] | {id, features}'

# binary size of every dep object (release)
cargo build --release
find target/release/deps -name '*.rlib' -exec du -h {} + | sort -h | tail -20

# detailed timing report
cargo build --release --timings
$BROWSER target/cargo-timings/cargo-timing.html        # or just `xdg-open` / `open`
```

### Avoiding unnecessary rebuilds

```bash
# BAD: this nukes the cache
cargo clean && cargo build

# GOOD: incremental rebuild, only changed crates rebuild
cargo build

# BAD: passing different RUSTFLAGS each invocation invalidates the cache
RUSTFLAGS="-C target-cpu=native" cargo build && RUSTFLAGS="-C target-cpu=x86-64" cargo build
# both always rebuild from scratch

# GOOD: pin RUSTFLAGS in .cargo/config.toml
# [build]
# rustflags = ["-C", "target-cpu=native"]

# nuke just one crate's artifacts
cargo clean -p mycrate
cargo clean --release
cargo clean --target wasm32-unknown-unknown
cargo clean --doc
```

## See Also

- rust — the Rust language sheet (syntax, types, traits, lifetimes, async)
- gomod — Go modules, the closest neighbor in package-manager design
- pnpm — JS package manager with workspaces (similar workspace semantics)
- uv — Python package/project manager (modern Cargo-style ergonomics)
- poetry — Python alternative with a pyproject.toml analogue to Cargo.toml
- gpg — for signing released artifacts and verifying maintainer keys
- polyglot — cross-language quick reference

## References

- rustup — https://rustup.rs and https://rust-lang.github.io/rustup/
- The Cargo Book — https://doc.rust-lang.org/cargo/
- Cargo reference (manifest, profiles, features) — https://doc.rust-lang.org/cargo/reference/
- crates.io registry — https://crates.io/
- docs.rs (rendered docs for every published crate) — https://docs.rs/
- The Rust Reference — https://doc.rust-lang.org/reference/
- The Rust API Guidelines — https://rust-lang.github.io/api-guidelines/
- RustSec advisory database — https://rustsec.org/ and https://github.com/RustSec/advisory-db
- cargo-deny — https://github.com/EmbarkStudios/cargo-deny
- cargo-audit — https://github.com/RustSec/rustsec/tree/main/cargo-audit
- cargo-expand — https://github.com/dtolnay/cargo-expand
- cargo-udeps — https://github.com/est31/cargo-udeps
- cargo-nextest — https://nexte.st/
- cargo-watch — https://github.com/watchexec/cargo-watch
- criterion (benchmarking) — https://bheisler.github.io/criterion.rs/book/
- The cross crate (cross-compile via Docker) — https://github.com/cross-rs/cross
- mold linker — https://github.com/rui314/mold
- sccache — https://github.com/mozilla/sccache
- rust-toolchain.toml spec — https://rust-lang.github.io/rustup/overrides.html
- Cargo features explained — https://doc.rust-lang.org/cargo/reference/features.html
- Cargo workspaces — https://doc.rust-lang.org/cargo/reference/workspaces.html
- Cargo profiles — https://doc.rust-lang.org/cargo/reference/profiles.html
- Build scripts — https://doc.rust-lang.org/cargo/reference/build-scripts.html
- Custom registries — https://doc.rust-lang.org/cargo/reference/registries.html
- Conditional compilation — https://doc.rust-lang.org/reference/conditional-compilation.html
- Rust target tier policy — https://doc.rust-lang.org/rustc/platform-support.html
- Semver compatibility — https://doc.rust-lang.org/cargo/reference/semver.html
- Rust release channels — https://forge.rust-lang.org/infra/channel-layouts.html
- Edition guide — https://doc.rust-lang.org/edition-guide/
