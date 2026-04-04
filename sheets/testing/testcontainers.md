# Testcontainers (Integration Testing with Docker)

Spin up real Docker containers for integration tests -- databases, message brokers, and services with throwaway instances.

## Getting Started

### Installation

```bash
# Go
go get github.com/testcontainers/testcontainers-go

# Java (Maven)
# <dependency>
#   <groupId>org.testcontainers</groupId>
#   <artifactId>testcontainers</artifactId>
#   <version>1.19.7</version>
#   <scope>test</scope>
# </dependency>

# Python
pip install testcontainers

# Node.js
npm install --save-dev testcontainers
```

## GenericContainer

### Go

```go
package mytest

import (
    "context"
    "testing"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestWithRedis(t *testing.T) {
    ctx := context.Background()
    req := testcontainers.ContainerRequest{
        Image:        "redis:7-alpine",
        ExposedPorts: []string{"6379/tcp"},
        WaitingFor:   wait.ForLog("Ready to accept connections"),
    }
    redis, err := testcontainers.GenericContainer(ctx,
        testcontainers.GenericContainerRequest{
            ContainerRequest: req,
            Started:          true,
        })
    if err != nil {
        t.Fatal(err)
    }
    defer redis.Terminate(ctx)

    host, _ := redis.Host(ctx)
    port, _ := redis.MappedPort(ctx, "6379")
    // connect to host:port.Port()
}
```

### Python

```python
from testcontainers.core.container import DockerContainer
from testcontainers.core.waiting_utils import wait_for_logs

def test_with_nginx():
    with DockerContainer("nginx:alpine").with_exposed_ports(80) as nginx:
        wait_for_logs(nginx, "start worker process")
        host = nginx.get_container_host_ip()
        port = nginx.get_exposed_port(80)
        # connect to host:port
```

### Java

```java
@Testcontainers
class MyTest {
    @Container
    static GenericContainer<?> redis =
        new GenericContainer<>("redis:7-alpine")
            .withExposedPorts(6379)
            .waitingFor(Wait.forLogMessage(".*Ready to accept connections.*", 1));

    @Test
    void testRedis() {
        String host = redis.getHost();
        int port = redis.getFirstMappedPort();
        // connect to host:port
    }
}
```

### Node.js

```js
import { GenericContainer } from "testcontainers";

test("redis container", async () => {
  const redis = await new GenericContainer("redis:7-alpine")
    .withExposedPorts(6379)
    .withWaitStrategy(Wait.forLogMessage("Ready to accept connections"))
    .start();

  const host = redis.getHost();
  const port = redis.getMappedPort(6379);
  // connect to host:port

  await redis.stop();
});
```

## Wait Strategies

### Common strategies

```go
// Wait for log message
wait.ForLog("database system is ready to accept connections")

// Wait for HTTP endpoint
wait.ForHTTP("/health").WithPort("8080/tcp").WithStatusCodeMatcher(
    func(status int) bool { return status == 200 },
)

// Wait for port to be listening
wait.ForListeningPort("5432/tcp")

// Wait for SQL query to succeed
wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
    return fmt.Sprintf("postgres://user:pass@%s:%s/db?sslmode=disable", host, port.Port())
})

// Combine strategies
wait.ForAll(
    wait.ForLog("started"),
    wait.ForHTTP("/ready").WithPort("8080/tcp"),
)

// Wait with timeout
wait.ForLog("ready").WithStartupTimeout(60 * time.Second)
```

## Module Containers

### PostgreSQL

```go
import "github.com/testcontainers/testcontainers-go/modules/postgres"

func TestPostgres(t *testing.T) {
    ctx := context.Background()
    pg, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        postgres.WithInitScripts("testdata/init.sql"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        t.Fatal(err)
    }
    defer pg.Terminate(ctx)

    connStr, _ := pg.ConnectionString(ctx, "sslmode=disable")
    // use connStr to connect
}
```

### Redis

```go
import "github.com/testcontainers/testcontainers-go/modules/redis"

func TestRedis(t *testing.T) {
    ctx := context.Background()
    r, err := redis.Run(ctx, "redis:7-alpine")
    if err != nil {
        t.Fatal(err)
    }
    defer r.Terminate(ctx)

    connStr, _ := r.ConnectionString(ctx)
    // use connStr: "redis://localhost:32768"
}
```

### Kafka

```go
import "github.com/testcontainers/testcontainers-go/modules/kafka"

func TestKafka(t *testing.T) {
    ctx := context.Background()
    k, err := kafka.Run(ctx,
        "confluentinc/confluent-local:7.6.0",
        kafka.WithClusterID("test-cluster"),
    )
    if err != nil {
        t.Fatal(err)
    }
    defer k.Terminate(ctx)

    brokers, _ := k.Brokers(ctx)
    // use brokers[0] to produce/consume
}
```

## Container Networking

### Shared network

```go
net, err := testcontainers.GenericNetwork(ctx,
    testcontainers.GenericNetworkRequest{
        NetworkRequest: testcontainers.NetworkRequest{Name: "test-net"},
    })
defer net.Remove(ctx)

// Containers on the same network can reach each other by name
appReq := testcontainers.ContainerRequest{
    Image:    "myapp:latest",
    Networks: []string{"test-net"},
    NetworkAliases: map[string][]string{
        "test-net": {"app"},
    },
}

dbReq := testcontainers.ContainerRequest{
    Image:    "postgres:16",
    Networks: []string{"test-net"},
    NetworkAliases: map[string][]string{
        "test-net": {"db"},
    },
}
// app container can connect to "db:5432"
```

## Docker Compose

### Using compose in tests

```go
import "github.com/testcontainers/testcontainers-go/modules/compose"

func TestCompose(t *testing.T) {
    ctx := context.Background()
    c, err := compose.NewDockerCompose("docker-compose.yml")
    if err != nil {
        t.Fatal(err)
    }
    defer c.Down(ctx, compose.RemoveOrphans(true), compose.RemoveVolumes(true))

    err = c.Up(ctx, compose.Wait(true))
    if err != nil {
        t.Fatal(err)
    }

    // Access individual services
    web, err := c.ServiceContainer(ctx, "web")
    host, _ := web.Host(ctx)
    port, _ := web.MappedPort(ctx, "8080")
}
```

## Container Configuration

### Environment, commands, volumes

```go
req := testcontainers.ContainerRequest{
    Image:        "myapp:latest",
    ExposedPorts: []string{"8080/tcp"},
    Env: map[string]string{
        "DATABASE_URL": "postgres://db:5432/app",
        "LOG_LEVEL":    "debug",
    },
    Cmd:  []string{"./server", "--config", "/etc/app.yaml"},
    Files: []testcontainers.ContainerFile{
        {
            HostFilePath:      "./testdata/config.yaml",
            ContainerFilePath: "/etc/app.yaml",
            FileMode:          0o644,
        },
    },
    Mounts: testcontainers.ContainerMounts{
        testcontainers.BindMount("./testdata", "/data"),
    },
}
```

### Container lifecycle hooks

```go
req := testcontainers.ContainerRequest{
    Image: "postgres:16",
    LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
        {
            PostStarts: []testcontainers.ContainerHook{
                func(ctx context.Context, c testcontainers.Container) error {
                    // run migrations after container starts
                    code, _, err := c.Exec(ctx, []string{"psql", "-c", "CREATE TABLE test(id INT)"})
                    return err
                },
            },
        },
    },
}
```

## CI Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./... -count=1 -race -timeout 300s
        env:
          TESTCONTAINERS_RYUK_DISABLED: "false"
```

### GitLab CI (Docker-in-Docker)

```yaml
test:
  image: golang:1.24
  services:
    - docker:24-dind
  variables:
    DOCKER_HOST: tcp://docker:2375
    TESTCONTAINERS_HOST_OVERRIDE: docker
  script:
    - go test ./... -count=1 -race
```

## Tips

- Always call `Terminate()` or `Stop()` in a defer to clean up containers after tests
- Use module containers (PostgreSQL, Redis, Kafka) instead of GenericContainer when available -- they handle configuration automatically
- Set `WithStartupTimeout` generously in CI -- container pulls and starts are slower than local
- Use `wait.ForAll` to combine multiple readiness checks for complex containers
- Use `NetworkAliases` for service-to-service communication in multi-container tests
- Set `TESTCONTAINERS_RYUK_DISABLED=true` only if your CI cleans up containers itself
- Use `WithInitScripts` to load test schemas instead of running migrations in every test
- Pin container image tags (e.g., `postgres:16.2-alpine`) to avoid flaky tests from upstream changes
- Use `t.Parallel()` with separate containers to speed up Go integration tests
- Run testcontainer tests with `-timeout 300s` to avoid premature timeouts on slow pulls
- Use `Exec` to run commands inside containers for seed data or verification
- Keep testcontainer tests in separate `_integration_test.go` files with build tags for fast unit test runs

## See Also

- docker
- docker-compose
- pytest
- jest
- github-actions

## References

- [Testcontainers Official Documentation](https://testcontainers.com/)
- [Testcontainers for Go](https://golang.testcontainers.org/)
- [Testcontainers for Java](https://java.testcontainers.org/)
- [Testcontainers for Python](https://testcontainers-python.readthedocs.io/)
- [Testcontainers for Node.js](https://node.testcontainers.org/)
- [Testcontainers GitHub Repository](https://github.com/testcontainers)
