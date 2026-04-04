# Dagger (Programmable CI/CD Engine)

Programmable CI/CD engine that lets you write pipelines as code in Go, Python, or TypeScript, executing steps as containers with automatic caching, powered by BuildKit for reproducible builds anywhere.

## Installation

### Install dagger CLI

```bash
# macOS via Homebrew
brew install dagger/tap/dagger

# Linux / macOS via shell script
curl -fsSL https://dl.dagger.io/dagger/install.sh | sh

# Verify installation
dagger version

# Initialize a new Dagger module
dagger init --sdk=go --name=ci

# List available functions
dagger functions
```

### SDK setup

```bash
# Go SDK — initialize module
mkdir ci && cd ci
dagger init --sdk=go --name=ci
# Creates: dagger.json, main.go

# Python SDK
dagger init --sdk=python --name=ci
# Creates: dagger.json, src/main/__init__.py

# TypeScript SDK
dagger init --sdk=typescript --name=ci
# Creates: dagger.json, src/index.ts
```

## Go SDK

### Basic pipeline

```go
// main.go — Dagger module with build and test functions
package main

import (
	"context"
	"dagger/ci/internal/dagger"
)

type Ci struct{}

// Build compiles the Go application
func (m *Ci) Build(ctx context.Context, source *dagger.Directory) *dagger.File {
	return dag.Container().
		From("golang:1.24-alpine").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"go", "build", "-trimpath", "-ldflags", "-s -w", "-o", "/app", "./cmd/server"}).
		File("/app")
}

// Test runs the test suite with race detection
func (m *Ci) Test(ctx context.Context, source *dagger.Directory) (string, error) {
	return dag.Container().
		From("golang:1.24").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"go", "test", "-v", "-race", "-count=1", "./..."}).
		Stdout(ctx)
}

// Lint runs golangci-lint on the source code
func (m *Ci) Lint(ctx context.Context, source *dagger.Directory) (string, error) {
	return dag.Container().
		From("golangci/golangci-lint:latest").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "./..."}).
		Stdout(ctx)
}
```

### Container image build and push

```go
// BuildImage builds a container image and optionally pushes it
func (m *Ci) BuildImage(
	ctx context.Context,
	source *dagger.Directory,
	// +optional
	// +default="myapp:latest"
	tag string,
) *dagger.Container {
	// Build the binary
	binary := m.Build(ctx, source)

	// Create a minimal runtime image
	return dag.Container().
		From("alpine:3.19").
		WithExec([]string{"apk", "add", "--no-cache", "ca-certificates"}).
		WithFile("/usr/local/bin/app", binary).
		WithEntrypoint([]string{"/usr/local/bin/app"}).
		WithExposedPort(8080)
}

// Publish pushes the container image to a registry
func (m *Ci) Publish(
	ctx context.Context,
	source *dagger.Directory,
	registry string,
	username string,
	password *dagger.Secret,
) (string, error) {
	ctr := m.BuildImage(ctx, source, "")
	return ctr.
		WithRegistryAuth(registry, username, password).
		Publish(ctx, registry+"/myapp:latest")
}
```

### Caching dependencies

```go
// TestWithCache runs tests with Go module caching
func (m *Ci) TestWithCache(ctx context.Context, source *dagger.Directory) (string, error) {
	goModCache := dag.CacheVolume("go-mod-cache")
	goBuildCache := dag.CacheVolume("go-build-cache")

	return dag.Container().
		From("golang:1.24").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithMountedCache("/root/.cache/go-build", goBuildCache).
		WithExec([]string{"go", "test", "-v", "-race", "./..."}).
		Stdout(ctx)
}
```

### Multi-platform builds

```go
// BuildMultiPlatform builds for linux/amd64 and linux/arm64
func (m *Ci) BuildMultiPlatform(
	ctx context.Context,
	source *dagger.Directory,
) []*dagger.Container {
	platforms := []dagger.Platform{"linux/amd64", "linux/arm64"}
	var containers []*dagger.Container

	for _, platform := range platforms {
		binary := dag.Container().
			From("golang:1.24-alpine").
			WithDirectory("/src", source).
			WithWorkdir("/src").
			WithEnvVariable("CGO_ENABLED", "0").
			WithExec([]string{"go", "build", "-o", "/app", "./cmd/server"}).
			File("/app")

		ctr := dag.Container(dagger.ContainerOpts{Platform: platform}).
			From("alpine:3.19").
			WithFile("/usr/local/bin/app", binary).
			WithEntrypoint([]string{"/usr/local/bin/app"})

		containers = append(containers, ctr)
	}
	return containers
}
```

## Python SDK

### Basic pipeline

```python
# src/main/__init__.py
import dagger
from dagger import dag, function, object_type

@object_type
class Ci:
    @function
    async def test(self, source: dagger.Directory) -> str:
        """Run tests with pytest"""
        return await (
            dag.container()
            .from_("python:3.12-slim")
            .with_directory("/src", source)
            .with_workdir("/src")
            .with_exec(["pip", "install", "-r", "requirements.txt"])
            .with_exec(["pytest", "-v", "--tb=short"])
            .stdout()
        )

    @function
    async def lint(self, source: dagger.Directory) -> str:
        """Run ruff linter"""
        return await (
            dag.container()
            .from_("python:3.12-slim")
            .with_directory("/src", source)
            .with_workdir("/src")
            .with_exec(["pip", "install", "ruff"])
            .with_exec(["ruff", "check", "."])
            .stdout()
        )
```

## TypeScript SDK

### Basic pipeline

```typescript
// src/index.ts
import { dag, Container, Directory, object, func } from "@dagger.io/dagger"

@object()
class Ci {
  @func()
  async test(source: Directory): Promise<string> {
    return dag
      .container()
      .from("node:20-slim")
      .withDirectory("/src", source)
      .withWorkdir("/src")
      .withExec(["npm", "ci"])
      .withExec(["npm", "test"])
      .stdout()
  }

  @func()
  async build(source: Directory): Promise<Container> {
    return dag
      .container()
      .from("node:20-slim")
      .withDirectory("/src", source)
      .withWorkdir("/src")
      .withExec(["npm", "ci"])
      .withExec(["npm", "run", "build"])
  }
}
```

## CLI Usage

### Running functions

```bash
# Call a function from the module
dagger call test --source=.

# Call with specific arguments
dagger call build --source=. export --path=./output/app

# Call build-image and publish
dagger call publish \
  --source=. \
  --registry=ghcr.io/myorg \
  --username=myuser \
  --password=env:REGISTRY_PASSWORD

# Chain function calls
dagger call build-image --source=. publish --address=ghcr.io/myorg/app:latest

# Export a file from a function
dagger call build --source=. export --path=./bin/app

# Run with debug logging
dagger call --debug test --source=.
```

### Using secrets

```bash
# Pass secrets from environment variables
dagger call deploy --token=env:DEPLOY_TOKEN

# Pass secrets from files
dagger call deploy --token=file:./secret.txt

# Pass secrets from command output
dagger call deploy --token=cmd:"vault kv get -field=token secret/deploy"
```

### Module management

```bash
# Initialize a new module
dagger init --sdk=go --name=ci

# Install a dependency module
dagger install github.com/purpleclay/daggerverse/golang@v0.5.0

# List installed dependencies
dagger functions

# Develop with live reload
dagger develop

# Generate SDK code
dagger develop --sdk=go
```

## Services and Networking

### Run services during tests

```go
// TestWithDB runs tests with a PostgreSQL sidecar
func (m *Ci) TestWithDB(ctx context.Context, source *dagger.Directory) (string, error) {
	// Start a PostgreSQL service
	postgres := dag.Container().
		From("postgres:16-alpine").
		WithEnvVariable("POSTGRES_PASSWORD", "testpass").
		WithEnvVariable("POSTGRES_DB", "testdb").
		WithExposedPort(5432).
		AsService()

	// Run tests with the database service
	return dag.Container().
		From("golang:1.24").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithServiceBinding("db", postgres).
		WithEnvVariable("DATABASE_URL", "postgres://postgres:testpass@db:5432/testdb?sslmode=disable").
		WithExec([]string{"go", "test", "-v", "./..."}).
		Stdout(ctx)
}
```

## Integration with CI Systems

### GitHub Actions

```yaml
# .github/workflows/ci.yaml
name: CI
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Dagger pipeline
        uses: dagger/dagger-for-github@v6
        with:
          verb: call
          args: test --source=.
```

### GitLab CI

```yaml
# .gitlab-ci.yml
test:
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - curl -fsSL https://dl.dagger.io/dagger/install.sh | sh
  script:
    - dagger call test --source=.
```

## Tips

- Every Dagger function call is cached by default; use `WithEnvVariable("CACHEBUSTER", time.Now())` to force re-execution.
- Use `dag.CacheVolume()` for dependency caches (Go modules, npm, pip) to dramatically speed up repeated builds.
- Secrets passed via `env:`, `file:`, or `cmd:` are never logged or stored in cache keys.
- Chain function calls in the CLI (`dagger call build publish`) to avoid round-trips and leverage caching.
- Dagger runs the same way locally and in CI — debug pipelines on your laptop before pushing.
- Use `WithExec` for each logical step rather than combining commands with `&&` for better cache granularity.
- Multi-platform builds require setting `CGO_ENABLED=0` for static Go binaries.
- Services (databases, APIs) started with `AsService()` are automatically cleaned up after the function returns.
- Use `--debug` flag to see BuildKit execution details when troubleshooting cache misses.
- Export files from containers with `.File("/path").Export("local/path")` for local development.
- Keep modules small and composable; install other Dagger modules as dependencies for reuse.
- Pin base images to specific tags (not `latest`) for reproducible builds.

## See Also

tekton, github-actions, gitlab-ci, docker, kubernetes

## References

- [Dagger Documentation](https://docs.dagger.io/)
- [Dagger Go SDK Reference](https://docs.dagger.io/sdk/go)
- [Dagger Python SDK Reference](https://docs.dagger.io/sdk/python)
- [Dagger TypeScript SDK Reference](https://docs.dagger.io/sdk/typescript)
- [Dagger CLI Reference](https://docs.dagger.io/reference/cli/)
- [Daggerverse (Module Registry)](https://daggerverse.dev/)
- [Dagger GitHub Repository](https://github.com/dagger/dagger)
- [BuildKit Documentation](https://github.com/moby/buildkit)
