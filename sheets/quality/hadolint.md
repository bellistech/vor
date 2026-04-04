# Hadolint (Dockerfile Linter)

Dockerfile linter that validates best practices, catches common mistakes, and enforces security policies using ShellCheck for RUN instructions.

## Installation

### Package managers

```bash
# macOS
brew install hadolint

# Download binary (Linux)
wget -O /usr/local/bin/hadolint \
  https://github.com/hadolint/hadolint/releases/latest/download/hadolint-Linux-x86_64
chmod +x /usr/local/bin/hadolint

# Docker
docker run --rm -i hadolint/hadolint < Dockerfile

# npm (wrapper)
npm install -g hadolint
```

## Basic Usage

### Running Hadolint

```bash
hadolint Dockerfile                     # lint default Dockerfile
hadolint Dockerfile.prod                # lint specific file
hadolint - < Dockerfile                 # read from stdin
hadolint --format json Dockerfile       # JSON output
hadolint --format codeclimate Dockerfile  # Code Climate format
hadolint --format checkstyle Dockerfile # Checkstyle XML
hadolint --format sarif Dockerfile      # SARIF (GitHub Security)
hadolint --format tty Dockerfile        # colored terminal (default)
hadolint --no-color Dockerfile          # plain text
cat Dockerfile | hadolint -             # pipe from stdin
```

### Filtering rules

```bash
hadolint --ignore DL3008 Dockerfile              # ignore single rule
hadolint --ignore DL3008 --ignore DL3003 Dockerfile  # ignore multiple
hadolint --failure-threshold warning Dockerfile   # fail only on warning+
hadolint --failure-threshold error Dockerfile     # fail only on errors
hadolint --strict-labels Dockerfile               # enforce label schema
```

## Common DL Codes

### DL3003 -- Use WORKDIR instead of cd

```dockerfile
# Bad
RUN cd /app && make install

# Good
WORKDIR /app
RUN make install
```

### DL3008 -- Pin versions in apt-get install

```dockerfile
# Bad -- unpinned, builds are not reproducible
RUN apt-get update && apt-get install -y curl wget

# Good -- pinned versions
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl=7.88.1-10+deb12u5 \
    wget=1.21.3-1+b2 \
    && rm -rf /var/lib/apt/lists/*
```

### DL3009 -- Delete apt-get lists after install

```dockerfile
# Bad -- bloated image
RUN apt-get update && apt-get install -y curl

# Good -- clean up
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*
```

### DL3013 -- Pin pip versions

```dockerfile
# Bad
RUN pip install flask requests

# Good
RUN pip install --no-cache-dir \
    flask==3.0.2 \
    requests==2.31.0
```

### DL3015 -- Use --no-install-recommends

```dockerfile
# Bad -- installs unnecessary recommended packages
RUN apt-get update && apt-get install -y python3

# Good
RUN apt-get update && apt-get install -y --no-install-recommends python3
```

### DL3018 -- Pin versions in apk add

```dockerfile
# Bad
RUN apk add curl jq

# Good
RUN apk add --no-cache \
    curl=8.5.0-r0 \
    jq=1.7.1-r0
```

### DL3020 -- Use COPY instead of ADD for files

```dockerfile
# Bad -- ADD has extra features (tar extraction, URL fetch) that are rarely needed
ADD config.yaml /etc/app/config.yaml

# Good -- COPY is explicit
COPY config.yaml /etc/app/config.yaml

# ADD is appropriate for:
ADD https://example.com/archive.tar.gz /tmp/     # URL fetch
ADD archive.tar.gz /opt/                          # auto-extraction
```

### DL3025 -- Use JSON notation for CMD/ENTRYPOINT

```dockerfile
# Bad -- shell form, runs via /bin/sh -c
CMD node server.js
ENTRYPOINT myapp start

# Good -- exec form, no shell wrapping
CMD ["node", "server.js"]
ENTRYPOINT ["myapp", "start"]
```

### DL4006 -- Set SHELL with pipefail

```dockerfile
# Bad -- pipe failures are silent
RUN curl https://example.com/install.sh | bash

# Good
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN curl https://example.com/install.sh | bash
```

### DL3006 -- Always tag images

```dockerfile
# Bad -- :latest is implicit and mutable
FROM ubuntu

# Good -- pinned tag
FROM ubuntu:22.04

# Best -- pinned digest
FROM ubuntu:22.04@sha256:abc123...
```

## Configuration

### .hadolint.yaml

```yaml
# .hadolint.yaml (project root)
ignored:
  - DL3008    # pin versions in apt-get
  - DL3013    # pin versions in pip

override:
  error:
    - DL3001  # pipe curl to shell
    - DL3002  # last user should not be root
  warning:
    - DL3042  # cache dir in pip install
  info:
    - DL3032  # yum clean all

failure-threshold: warning

trustedRegistries:
  - docker.io
  - gcr.io
  - ghcr.io
  - registry.example.com

label-schema:
  maintainer: text
  org.opencontainers.image.source: url
  org.opencontainers.image.version: semver

strict-labels: true
```

### Inline ignores

```dockerfile
# hadolint ignore=DL3008
RUN apt-get update && apt-get install -y curl

# hadolint ignore=DL3008,DL3015
RUN apt-get update && apt-get install -y python3 build-essential

# Ignore for specific line
RUN apt-get update \
    # hadolint ignore=DL3008
    && apt-get install -y curl
```

### Global config location

```bash
# Default locations (searched in order):
# .hadolint.yaml (current directory)
# $XDG_CONFIG_HOME/hadolint.yaml
# ~/.config/hadolint.yaml

# Explicit config
hadolint --config /path/to/hadolint.yaml Dockerfile
```

## Severity Levels

### Level definitions

```bash
# error   -- definite problems (security, correctness)
# warning -- best practice violations
# info    -- improvement suggestions
# style   -- cosmetic issues

# DL codes: Dockerfile rules (DL1000-DL9999)
# SC codes: ShellCheck rules applied to RUN (SC1000-SC9999)
```

### Common severity mapping

```
error:    DL3001 (pipe curl|bash), DL3002 (USER root last)
warning:  DL3008 (unpin apt), DL3009 (no cleanup), DL3025 (CMD form)
info:     DL3015 (no-install-recommends), DL3020 (ADD vs COPY)
style:    DL3003 (use WORKDIR), DL4000 (MAINTAINER deprecated)
```

## CI Integration

### GitHub Actions

```yaml
jobs:
  hadolint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: hadolint/hadolint-action@v3.1.0
        with:
          dockerfile: Dockerfile
          failure-threshold: warning
          ignore: DL3008,DL3018
```

### GitLab CI

```yaml
hadolint:
  image: hadolint/hadolint:latest-debian
  script:
    - hadolint Dockerfile
    - hadolint --failure-threshold warning services/*/Dockerfile
```

### Pre-commit hook

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/hadolint/hadolint
    rev: v2.12.0
    hooks:
      - id: hadolint
        args: ['--failure-threshold', 'warning']
```

### Makefile target

```makefile
.PHONY: lint-docker
lint-docker:
	@find . -name 'Dockerfile*' -exec hadolint --failure-threshold warning {} +
```

## Tips

- Pin package versions (`DL3008`/`DL3013`/`DL3018`) for reproducible builds even if it increases maintenance burden
- Always clean up package manager caches in the same `RUN` layer to avoid bloating the image
- Use `COPY` over `ADD` unless you specifically need tar auto-extraction or URL fetching
- Use exec form (`CMD ["app"]`) over shell form (`CMD app`) to receive signals properly
- Set `SHELL ["/bin/bash", "-o", "pipefail", "-c"]` before any `RUN` with pipes
- Use `--no-install-recommends` with `apt-get` to keep images minimal
- Configure `trustedRegistries` to enforce that images come only from approved sources
- Use `hadolint --format sarif` with GitHub code scanning for inline PR annotations
- Combine hadolint with multi-stage builds -- lint each stage's Dockerfile separately
- Run hadolint in CI with `--failure-threshold warning` to catch issues without blocking on style
- Use `.hadolint.yaml` in the project root for team-wide consistent configuration
- Hadolint applies ShellCheck rules to `RUN` instructions automatically -- you get shell linting for free

## See Also

- docker
- shellcheck
- dockerfile
- buildah
- pre-commit

## References

- [Hadolint GitHub Repository](https://github.com/hadolint/hadolint)
- [Hadolint Rule Reference](https://github.com/hadolint/hadolint/wiki)
- [Dockerfile Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Hadolint GitHub Action](https://github.com/hadolint/hadolint-action)
- [OCI Image Spec Labels](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
