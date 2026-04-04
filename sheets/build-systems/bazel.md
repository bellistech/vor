# Bazel

Google's polyglot build system designed for correctness, reproducibility, and scalability through hermetic builds, content-addressable caching, and remote execution.

## Getting Started

```bash
# Install via Bazelisk (recommended)
brew install bazelisk        # macOS
npm install -g @bazel/bazelisk  # or via npm

# Check version
bazel version

# Build a target
bazel build //src:myapp

# Run a target
bazel run //src:myapp

# Test everything
bazel test //...

# Clean build artifacts
bazel clean
bazel clean --expunge   # remove entire output base

# Query the build graph
bazel query //...
bazel query 'deps(//src:myapp)'
```

## WORKSPACE File

```python
# WORKSPACE (legacy) - defines external dependencies
workspace(name = "myproject")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "com_google_googletest",
    urls = ["https://github.com/google/googletest/archive/v1.14.0.tar.gz"],
    strip_prefix = "googletest-1.14.0",
    sha256 = "8ad598c73ad796e0d8280b082cebd82a630d73e73cd3c70057938a6501bba5d7",
)

# Go rules
http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/v0.46.0/rules_go-v0.46.0.zip"],
    sha256 = "...",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
go_rules_dependencies()
go_register_toolchains(version = "1.22.0")
```

## Bzlmod (MODULE.bazel)

```python
# MODULE.bazel (modern, replaces WORKSPACE)
module(
    name = "myproject",
    version = "1.0.0",
)

bazel_dep(name = "googletest", version = "1.14.0")
bazel_dep(name = "rules_go", version = "0.46.0")
bazel_dep(name = "rules_java", version = "7.4.0")
bazel_dep(name = "rules_python", version = "0.31.0")

# Go toolchain
go = use_extension("@rules_go//go:extensions.bzl", "go_sdk")
go.download(version = "1.22.0")
```

## BUILD Files

```python
# BUILD file - defines targets in a package
load("@rules_cc//cc:defs.bzl", "cc_binary", "cc_library", "cc_test")

cc_library(
    name = "utils",
    srcs = ["utils.cc"],
    hdrs = ["utils.h"],
    deps = ["//lib:core"],
    visibility = ["//visibility:public"],
)

cc_binary(
    name = "myapp",
    srcs = ["main.cc"],
    deps = [":utils"],
)

cc_test(
    name = "utils_test",
    srcs = ["utils_test.cc"],
    deps = [
        ":utils",
        "@com_google_googletest//:gtest_main",
    ],
)
```

## Multi-Language Rules

```python
# Java
load("@rules_java//java:defs.bzl", "java_binary", "java_library")

java_library(
    name = "lib",
    srcs = glob(["src/main/java/**/*.java"]),
    deps = ["@maven//:com_google_guava_guava"],
)

java_binary(
    name = "server",
    main_class = "com.example.Server",
    runtime_deps = [":lib"],
)

# Go
load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_library(
    name = "handler",
    srcs = ["handler.go"],
    importpath = "github.com/example/myapp/handler",
    deps = ["@com_github_gorilla_mux//:mux"],
)

go_binary(
    name = "server",
    embed = [":handler"],
    srcs = ["main.go"],
)

go_test(
    name = "handler_test",
    srcs = ["handler_test.go"],
    embed = [":handler"],
)

# Python
load("@rules_python//python:defs.bzl", "py_binary", "py_library")

py_binary(
    name = "train",
    srcs = ["train.py"],
    deps = [":model"],
)
```

## Starlark (Build Language)

```python
# .bzl files contain reusable macros and rules

# Macro (simple wrapper)
def my_cc_binary(name, srcs, **kwargs):
    cc_binary(
        name = name,
        srcs = srcs,
        copts = ["-Wall", "-Werror"],
        **kwargs
    )

# Custom rule
def _my_gen_impl(ctx):
    output = ctx.actions.declare_file(ctx.attr.name + ".h")
    ctx.actions.run(
        outputs = [output],
        inputs = ctx.files.srcs,
        executable = ctx.executable._tool,
        arguments = [output.path] + [f.path for f in ctx.files.srcs],
    )
    return [DefaultInfo(files = depset([output]))]

my_gen = rule(
    implementation = _my_gen_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "_tool": attr.label(
            default = "//tools:codegen",
            executable = True,
            cfg = "exec",
        ),
    },
)
```

## Remote Caching and Execution

```bash
# Remote cache (read/write)
bazel build //... --remote_cache=grpc://cache.example.com:443

# Remote cache (read-only, for CI)
bazel build //... --remote_cache=grpc://cache.example.com:443 \
    --remote_upload_local_results=false

# HTTP cache (simpler setup)
bazel build //... --remote_cache=https://cache.example.com

# Remote execution
bazel build //... \
    --remote_executor=grpc://rbe.example.com:443 \
    --remote_instance_name=projects/myproject/instances/default

# Disk cache (local)
bazel build //... --disk_cache=~/.cache/bazel
```

## Querying the Build Graph

```bash
# All targets in the repo
bazel query //...

# Dependencies of a target
bazel query 'deps(//src:myapp)'

# Reverse dependencies (what depends on this?)
bazel query 'rdeps(//..., //lib:utils)'

# All tests
bazel query 'kind(".*_test", //...)'

# Dependency path between two targets
bazel query 'allpaths(//src:myapp, //lib:utils)'

# Targets matching a pattern
bazel query 'attr(tags, "integration", //...)'

# cquery (configured query, respects select())
bazel cquery 'deps(//src:myapp)' --output=graph

# aquery (action graph)
bazel aquery '//src:myapp'
```

## Platforms and Configurations

```python
# BUILD file: conditional dependencies
cc_binary(
    name = "myapp",
    srcs = ["main.cc"],
    deps = select({
        "@platforms//os:linux": ["//lib:linux_impl"],
        "@platforms//os:macos": ["//lib:macos_impl"],
        "//conditions:default": ["//lib:generic_impl"],
    }),
)

# Platform definition
platform(
    name = "linux_x86_64",
    constraint_values = [
        "@platforms//os:linux",
        "@platforms//cpu:x86_64",
    ],
)
```

```bash
# Build for a specific platform
bazel build //src:myapp --platforms=//platforms:linux_x86_64

# Cross-compilation
bazel build //src:myapp --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
```

## Aspects

```python
# Aspects traverse the dependency graph and add actions
def _print_deps_impl(target, ctx):
    for dep in ctx.rule.attr.deps:
        print("{} depends on {}".format(target.label, dep.label))
    return []

print_deps = aspect(
    implementation = _print_deps_impl,
    attr_aspects = ["deps"],
)
```

```bash
bazel build //src:myapp --aspects=//tools:aspects.bzl%print_deps
```

## Tips

- Use Bazelisk instead of Bazel directly to automatically manage Bazel versions via `.bazelversion`
- Migrate from WORKSPACE to MODULE.bazel (bzlmod) for better dependency resolution and diamond dependency handling
- Use `bazel query` and `bazel cquery` to understand your build graph before optimizing
- Set up remote caching early -- even a simple HTTP cache dramatically reduces CI build times
- Use `select()` for platform-specific code instead of preprocessor macros or runtime checks
- Prefer fine-grained targets (one library per logical unit) to maximize cache hit rates and parallelism
- Use `visibility` to enforce module boundaries and prevent unauthorized dependencies
- Pin all external dependency versions with SHA-256 hashes for reproducibility
- Use `--sandbox_debug` to diagnose hermeticity issues when builds fail in sandboxed mode
- Run `bazel test //... --test_output=errors` to only see output from failing tests
- Use `tags = ["exclusive"]` for tests that cannot run in parallel with others
- Set up `.bazelrc` for common flags rather than requiring long command lines

## See Also

- Buck2 (Meta's build system)
- Pants build system
- CMake
- Gradle (for Java/Kotlin)
- Turborepo (for JavaScript monorepos)

## References

- [Bazel Documentation](https://bazel.build/docs)
- [Bazel BUILD Encyclopedia](https://bazel.build/reference/be/overview)
- [Starlark Language Spec](https://github.com/bazelbuild/starlark/blob/master/spec.md)
- [Bzlmod Migration Guide](https://bazel.build/external/migration)
- [Remote Execution API](https://github.com/bazelbuild/remote-apis)
- [Bazelisk](https://github.com/bazelbuild/bazelisk)
- [rules_go Documentation](https://github.com/bazelbuild/rules_go)
