# Cargo (Rust Package Manager)

> Rust's build system and package manager — compile, test, bench, and publish crates.

## Project Setup

```bash
cargo new myapp                        # new binary project
cargo new mylib --lib                  # new library project
cargo init                             # initialize in current directory
cargo init --lib                       # initialize as library
```

### Cargo.toml Basics

```toml
[package]
name = "myapp"
version = "0.1.0"
edition = "2021"
rust-version = "1.75"
description = "My application"
license = "MIT"
repository = "https://github.com/user/myapp"

[dependencies]
serde = { version = "1.0", features = ["derive"] }
tokio = { version = "1", features = ["full"] }
clap = { version = "4", features = ["derive"] }

[dev-dependencies]
tempfile = "3"
assert_cmd = "2"

[build-dependencies]
cc = "1"

[profile.release]
lto = true
strip = true
codegen-units = 1
```

## Build and Run

```bash
cargo build                            # debug build
cargo build --release                  # optimized release build
cargo run                              # build and run
cargo run -- arg1 arg2                 # pass arguments to binary
cargo run --release                    # run release build
cargo run --bin other-binary           # run specific binary
cargo run --example demo              # run example
cargo check                            # check without compiling (fast)
```

### Build Options

```bash
cargo build --target x86_64-unknown-linux-musl  # cross-compile
cargo build --jobs 4                   # limit parallel jobs
cargo build --verbose                  # show compilation commands
cargo build --timings                  # generate build timing report
RUSTFLAGS="-C target-cpu=native" cargo build --release  # native CPU optimizations
```

## Dependencies

### Adding

```bash
cargo add serde                        # add latest version
cargo add serde@1.0.193               # add specific version
cargo add serde --features derive      # add with features
cargo add tokio -F full                # shorthand for --features
cargo add --dev tempfile               # add dev dependency
cargo add --build cc                   # add build dependency
cargo add --path ../mylib              # add local crate
cargo add --git https://github.com/user/repo.git  # add from git
```

### Removing and Updating

```bash
cargo remove serde                     # remove dependency
cargo update                           # update all deps (within semver)
cargo update serde                     # update specific crate
cargo update --dry-run                 # show what would update
```

### Inspecting

```bash
cargo tree                             # dependency tree
cargo tree -d                          # show duplicated dependencies
cargo tree -i serde                    # inverted: show what depends on serde
cargo tree --depth 1                   # top-level only
cargo tree -e features                 # show feature flags
```

## Testing

```bash
cargo test                             # run all tests
cargo test auth                        # run tests matching "auth"
cargo test --lib                       # library tests only
cargo test --doc                       # doc tests only
cargo test --test integration          # run tests/integration.rs
cargo test -- --nocapture              # show println! output
cargo test -- --test-threads=1        # single-threaded
cargo test -- --ignored                # run #[ignore] tests
cargo test -- --show-output            # show output of passing tests
cargo test --release                   # test with release profile
cargo test --no-fail-fast              # run all tests even if some fail
```

### Test in Cargo.toml

```toml
[[test]]
name = "integration"
path = "tests/integration.rs"
harness = false                        # custom test harness
```

## Benchmarking

```bash
cargo bench                            # run all benchmarks
cargo bench auth                       # benchmarks matching "auth"
cargo bench -- --save-baseline before  # save baseline (criterion)

# requires nightly for built-in benches, or use criterion crate
# Cargo.toml:
# [dev-dependencies]
# criterion = { version = "0.5", features = ["html_reports"] }
#
# [[bench]]
# name = "my_benchmark"
# harness = false
```

## Documentation

```bash
cargo doc                              # build documentation
cargo doc --open                       # build and open in browser
cargo doc --no-deps                    # skip dependency docs
cargo doc --document-private-items     # include private items
```

## Publishing

```bash
cargo login                            # authenticate with crates.io
cargo package                          # create .crate file
cargo package --list                   # show files that would be packaged
cargo publish                          # publish to crates.io
cargo publish --dry-run                # validate without publishing
cargo yank --version 1.0.0             # yank version (prevent new deps)
cargo yank --version 1.0.0 --undo      # un-yank
cargo owner --add username             # add crate owner
```

## Clippy (Linting)

```bash
cargo clippy                           # run linter
cargo clippy --fix                     # auto-fix suggestions
cargo clippy --all-targets             # lint tests and examples too
cargo clippy -- -W clippy::pedantic    # stricter lints
cargo clippy -- -D warnings           # treat warnings as errors
cargo clippy -- -A clippy::needless_return  # allow specific lint
```

### In Code

```rust
#![warn(clippy::pedantic)]             // crate-level
#[allow(clippy::too_many_arguments)]   // function-level
```

## Formatting

```bash
cargo fmt                              # format all code
cargo fmt --check                      # check without modifying (CI)
cargo fmt -- --config max_width=100    # custom config
```

### rustfmt.toml

```toml
max_width = 100
tab_spaces = 4
edition = "2021"
imports_granularity = "Crate"
group_imports = "StdExternalCrate"
```

## Features

### Cargo.toml

```toml
[features]
default = ["json"]
json = ["dep:serde_json"]
yaml = ["dep:serde_yaml"]
full = ["json", "yaml"]

[dependencies]
serde_json = { version = "1", optional = true }
serde_yaml = { version = "0.9", optional = true }
```

### Using Features

```bash
cargo build --features json            # enable feature
cargo build --features "json,yaml"     # enable multiple
cargo build --all-features             # enable all features
cargo build --no-default-features      # disable default features
cargo build --no-default-features --features yaml  # only yaml
```

## Workspaces

### Cargo.toml (Root)

```toml
[workspace]
members = [
    "crates/*",
    "apps/*",
]
resolver = "2"

[workspace.dependencies]
serde = { version = "1", features = ["derive"] }
tokio = { version = "1", features = ["full"] }
```

### Commands

```bash
cargo build --workspace                # build all members
cargo test --workspace                 # test all members
cargo build -p mylib                   # build specific package
cargo test -p mylib                    # test specific package
```

## Useful Commands

```bash
cargo clean                            # remove target/ directory
cargo clean --release                  # remove release artifacts only
cargo locate-project                   # show Cargo.toml path
cargo metadata --format-version 1     # machine-readable project info
cargo vendor                           # vendor dependencies locally
cargo install ripgrep                  # install binary crate
cargo install --list                   # list installed binaries
cargo uninstall ripgrep                # remove installed binary
```

## Tips

- `cargo check` is much faster than `cargo build` -- use it during development for quick feedback.
- `cargo clippy` catches far more issues than the compiler alone. Run it before every commit.
- `cargo tree -d` finds duplicate crate versions that bloat your binary.
- `cargo build --timings` generates an HTML report showing which crates are slow to compile.
- `[profile.release] lto = true` enables link-time optimization -- smaller and faster binaries at the cost of compile time.
- `strip = true` in release profile removes debug symbols, significantly reducing binary size.
- `cargo add` (built-in since Rust 1.62) is the easiest way to add dependencies. No manual Cargo.toml editing.
- `cargo test -- --nocapture` is needed to see `println!` output from passing tests.
- `cargo update` only updates within semver-compatible ranges. Edit Cargo.toml to bump major versions.
- Use `workspace.dependencies` to keep dependency versions consistent across workspace members.

## References

- [The Cargo Book](https://doc.rust-lang.org/cargo/)
- [Cargo Commands Reference](https://doc.rust-lang.org/cargo/commands/)
- [Cargo.toml Manifest Format](https://doc.rust-lang.org/cargo/reference/manifest.html)
- [Cargo Profiles (dev, release)](https://doc.rust-lang.org/cargo/reference/profiles.html)
- [Cargo Workspaces](https://doc.rust-lang.org/cargo/reference/workspaces.html)
- [Cargo Build Scripts](https://doc.rust-lang.org/cargo/reference/build-scripts.html)
- [Cargo Environment Variables](https://doc.rust-lang.org/cargo/reference/environment-variables.html)
- [crates.io (Rust Package Registry)](https://crates.io/)
- [Rust Edition Guide](https://doc.rust-lang.org/edition-guide/)
- [Cargo GitHub Repository](https://github.com/rust-lang/cargo)
