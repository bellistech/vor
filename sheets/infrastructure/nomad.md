# Nomad (Workload Orchestrator)

Schedule and run containers, binaries, JVM apps, and batch jobs across a fleet of machines using HashiCorp Nomad's bin-packing scheduler, multi-region federation, and native Consul and Vault integrations.

## Job Specification

### Minimal Service Job

```hcl
job "web" {
  datacenters = ["dc1"]
  type        = "service"

  group "app" {
    count = 3

    network {
      port "http" {
        to = 8080
      }
    }

    service {
      name = "web"
      port = "http"

      check {
        type     = "http"
        path     = "/health"
        interval = "10s"
        timeout  = "3s"
      }
    }

    task "server" {
      driver = "docker"

      config {
        image = "myapp:v1.2.3"
        ports = ["http"]
      }

      resources {
        cpu    = 500   # MHz
        memory = 256   # MB
      }

      env {
        PORT = "${NOMAD_PORT_http}"
      }
    }
  }
}
```

### Batch Job

```hcl
job "etl-pipeline" {
  datacenters = ["dc1"]
  type        = "batch"

  periodic {
    cron             = "0 2 * * *"     # 2 AM daily
    prohibit_overlap = true
    time_zone        = "America/New_York"
  }

  group "transform" {
    count = 1

    task "run" {
      driver = "docker"

      config {
        image   = "etl:latest"
        command = "/bin/etl"
        args    = ["--date", "${NOMAD_META_run_date}"]
      }

      resources {
        cpu    = 2000
        memory = 1024
      }

      restart {
        attempts = 3
        interval = "30m"
        delay    = "15s"
        mode     = "fail"
      }
    }
  }
}
```

### System Job

```hcl
job "log-collector" {
  datacenters = ["dc1"]
  type        = "system"    # runs on every node

  group "collector" {
    task "fluentbit" {
      driver = "docker"

      config {
        image        = "fluent/fluent-bit:latest"
        network_mode = "host"
        volumes      = ["/var/log:/var/log:ro"]
      }

      resources {
        cpu    = 200
        memory = 128
      }
    }
  }
}
```

## Job Operations

### CLI Commands

```bash
# Plan a job (dry-run, shows diff)
nomad job plan web.nomad.hcl

# Run a job
nomad job run web.nomad.hcl

# Run with check-index (safe deployment)
nomad job run -check-index=42 web.nomad.hcl

# Check job status
nomad job status web
nomad job status -evals web     # show evaluations

# View allocations
nomad job allocs web
nomad alloc status <alloc-id>
nomad alloc logs <alloc-id>
nomad alloc logs -f <alloc-id>  # follow logs
nomad alloc logs -stderr <alloc-id>

# Exec into allocation
nomad alloc exec -task server <alloc-id> /bin/sh

# Stop a job
nomad job stop web
nomad job stop -purge web       # remove from state entirely

# Force periodic job to run now
nomad job periodic force etl-pipeline

# Inspect job spec from cluster
nomad job inspect web
```

### Deployment Management

```bash
# Check deployment status
nomad deployment status <deployment-id>

# Promote canary deployment
nomad deployment promote <deployment-id>

# Fail/rollback deployment
nomad deployment fail <deployment-id>

# List deployments
nomad job deployments web
```

## Task Drivers

### Docker Driver

```hcl
task "app" {
  driver = "docker"

  config {
    image      = "registry.example.com/myapp:v1.2.3"
    ports      = ["http"]
    force_pull = true

    auth {
      username = "deploy"
      password = "${DOCKER_PASSWORD}"
    }

    volumes = [
      "local/config.yaml:/app/config.yaml:ro"
    ]

    logging {
      type = "json-file"
      config {
        max-size = "10m"
        max-file = "3"
      }
    }

    ulimit {
      nofile = "65535:65535"
    }
  }
}
```

### Exec Driver

```hcl
task "worker" {
  driver = "exec"

  config {
    command = "/usr/local/bin/worker"
    args    = ["--config", "local/config.yaml"]
  }

  artifact {
    source      = "https://releases.example.com/worker-v1.2.3-linux-amd64"
    destination = "local/worker"
    mode        = "file"
  }
}
```

### Java Driver

```hcl
task "api" {
  driver = "java"

  config {
    jar_path    = "local/app.jar"
    jvm_options = ["-Xmx512m", "-Xms256m", "-XX:+UseG1GC"]
    args        = ["--spring.profiles.active=production"]
  }

  artifact {
    source = "https://releases.example.com/app-1.2.3.jar"
  }

  resources {
    cpu    = 1000
    memory = 768
  }
}
```

## Update Strategy and Canary Deployments

### Rolling Update

```hcl
group "app" {
  count = 6

  update {
    max_parallel     = 2         # update 2 at a time
    health_check     = "checks"  # use Consul checks
    min_healthy_time = "30s"
    healthy_deadline = "5m"
    auto_revert      = true      # rollback on failure
    stagger          = "10s"     # delay between batches
  }
}
```

### Canary Deployment

```hcl
group "app" {
  count = 6

  update {
    canary       = 2             # deploy 2 canaries first
    max_parallel = 2
    auto_promote = false         # manual promotion required
    auto_revert  = true
  }
}
```

### Blue-Green Deployment

```hcl
group "app" {
  count = 6

  update {
    canary           = 6         # canary count = total count = full blue-green
    max_parallel     = 6
    auto_promote     = false
    min_healthy_time = "30s"
  }
}
```

## Templates and Consul Integration

### Template Block (consul-template)

```hcl
task "app" {
  template {
    data = <<-EOF
      DATABASE_URL=postgresql://{{ key "config/db/host" }}:5432/mydb
      {{ range service "redis" }}
      REDIS_ADDR={{ .Address }}:{{ .Port }}
      {{ end }}
    EOF
    destination = "secrets/env.conf"
    env         = true
    change_mode = "restart"
  }

  template {
    source      = "local/nginx.conf.tpl"
    destination = "local/nginx.conf"
    change_mode = "signal"
    change_signal = "SIGHUP"
  }
}
```

### Vault Integration

```hcl
task "app" {
  vault {
    policies = ["web-app"]
  }

  template {
    data = <<-EOF
      {{ with secret "database/creds/web" }}
      DB_USER={{ .Data.username }}
      DB_PASS={{ .Data.password }}
      {{ end }}
    EOF
    destination = "secrets/db.env"
    env         = true
  }
}
```

## Volumes and Storage

### Host Volumes

```hcl
# Client config: /etc/nomad.d/client.hcl
client {
  host_volume "data" {
    path      = "/opt/data"
    read_only = false
  }
}

# Job spec
group "db" {
  volume "data" {
    type   = "host"
    source = "data"
  }

  task "postgres" {
    volume_mount {
      volume      = "data"
      destination = "/var/lib/postgresql/data"
    }
  }
}
```

### CSI Volumes

```hcl
# Register CSI volume
# nomad volume create ebs-vol.hcl

volume "ebs-data" {
  type      = "csi"
  source    = "ebs-vol-123"
  read_only = false

  mount_options {
    fs_type = "ext4"
  }

  access_mode     = "single-node-writer"
  attachment_mode = "file-system"
}
```

## Node Management

### Node Operations

```bash
# List nodes
nomad node status

# Detailed node info
nomad node status <node-id>

# Drain node (migrate allocations off)
nomad node drain -enable <node-id>
nomad node drain -enable -deadline 1h <node-id>

# Mark node ineligible for new placements
nomad node eligibility -disable <node-id>

# Re-enable node
nomad node drain -disable <node-id>
nomad node eligibility -enable <node-id>
```

## Tips

- Always run `nomad job plan` before `nomad job run` -- it shows the diff and returns a check-index for safe concurrent deployments
- Set `auto_revert = true` on update blocks so failed deployments automatically roll back to the last stable version
- Use `system` type jobs for infrastructure daemons (log collectors, monitoring agents) that must run on every node
- Set `max_parallel = 1` and `min_healthy_time = 30s` for critical services to catch failures before proceeding
- Use Consul service checks (not Nomad task checks) as the source of truth for deployment health -- they verify end-to-end connectivity
- Always set both `cpu` and `memory` resources -- without them, Nomad cannot bin-pack and you waste capacity
- Use `template` blocks with `env = true` instead of hardcoded `env` blocks so configs update from Consul/Vault automatically
- Drain nodes before maintenance with `nomad node drain -enable -deadline 1h` to gracefully migrate allocations
- Use `periodic` jobs with `prohibit_overlap = true` for cron-style workloads instead of external cron schedulers
- Set `kill_timeout` on tasks that need graceful shutdown time -- the default 5s is often too short for connection draining
- Use `artifact` blocks to download binaries at runtime so job specs are portable across versions

## See Also

- consul, vault, terraform, docker, kubernetes, traefik, prometheus

## References

- [Nomad Documentation](https://developer.hashicorp.com/nomad/docs)
- [Nomad Job Specification](https://developer.hashicorp.com/nomad/docs/job-specification)
- [Nomad Scheduling](https://developer.hashicorp.com/nomad/docs/concepts/scheduling/scheduling)
- [Nomad Deployment Strategies](https://developer.hashicorp.com/nomad/tutorials/job-updates)
- [Nomad Consul Integration](https://developer.hashicorp.com/nomad/docs/integrations/consul-integration)
