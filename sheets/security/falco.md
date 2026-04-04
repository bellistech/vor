# Falco (Runtime Security Monitoring)

Falco is a cloud-native runtime security tool that detects anomalous activity in containers and hosts by monitoring Linux syscalls via an eBPF probe or kernel module, processing Kubernetes audit logs, evaluating events against a rules engine with macros and lists, and routing alerts through Falcosidekick to dozens of output destinations.

## Installation

### Install Falco

```bash
# Install via Helm (Kubernetes)
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update

helm install falco falcosecurity/falco \
  --namespace falco \
  --create-namespace \
  --set falcosidekick.enabled=true \
  --set falcosidekick.webui.enabled=true

# Install on Linux (package)
curl -fsSL https://falco.org/repo/falcosecurity-packages.asc | \
  sudo gpg --dearmor -o /usr/share/keyrings/falco-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/falco-archive-keyring.gpg] https://download.falco.org/packages/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/falcosecurity.list

sudo apt-get update && sudo apt-get install falco

# Install with eBPF probe (preferred)
sudo FALCO_BPF_PROBE="" falco

# Start Falco as a service
sudo systemctl enable --now falco

# Verify Falco is running
sudo falco --version
sudo systemctl status falco
```

### Driver Selection

```bash
# Driver options: modern-bpf (preferred), ebpf, module (legacy)
helm install falco falcosecurity/falco --set driver.kind=modern-bpf
```

## Rules Engine

### Rule Structure

```yaml
# /etc/falco/falco_rules.yaml

# Lists define reusable sets of values
- list: trusted_images
  items:
    - docker.io/library/nginx
    - docker.io/library/alpine
    - gcr.io/my-project

- list: sensitive_mount_paths
  items:
    - /etc/shadow
    - /etc/passwd
    - /root/.ssh

# Macros define reusable condition snippets
- macro: container
  condition: container.id != host

- macro: spawned_process
  condition: evt.type in (execve, execveat) and evt.dir = <

- macro: is_shell
  condition: proc.name in (bash, sh, zsh, dash, ksh)

# Rules combine conditions with output formatting
- rule: Shell in Container
  desc: Detect shell execution inside a container
  condition: >
    spawned_process and
    container and
    is_shell and
    not proc.pname in (cron, crond, sshd)
  output: >
    Shell spawned in container
    (user=%user.name container=%container.name
     shell=%proc.name parent=%proc.pname
     cmdline=%proc.cmdline image=%container.image.repository)
  priority: WARNING
  tags: [container, shell, mitre_execution]
```

### Common Rule Patterns

```yaml
# Detect write to sensitive files
- rule: Write to /etc
  desc: Detect writes to /etc directory
  condition: >
    evt.type in (open, openat, openat2) and
    evt.dir = < and
    fd.typechar = f and
    fd.name startswith /etc/ and
    (evt.arg.flags contains O_WRONLY or evt.arg.flags contains O_RDWR) and
    container
  output: >
    File opened for writing in /etc (user=%user.name
    file=%fd.name container=%container.name
    image=%container.image.repository)
  priority: ERROR
  tags: [filesystem, mitre_persistence]

# Detect outbound connections from unexpected containers
- rule: Unexpected Outbound Connection
  desc: Container making outbound network connections
  condition: >
    evt.type = connect and
    evt.dir = < and
    fd.typechar = 4 and
    fd.ip != "0.0.0.0" and
    container and
    not container.image.repository in (trusted_images)
  output: >
    Unexpected outbound connection
    (container=%container.name image=%container.image.repository
     connection=%fd.name user=%user.name)
  priority: NOTICE
  tags: [network, mitre_command_and_control]

# Detect privilege escalation
- rule: Container Privilege Escalation
  desc: Detect setuid/setgid calls in container
  condition: >
    evt.type in (setuid, setgid, setreuid, setregid) and
    evt.dir = < and
    container and
    not user.name = root
  output: >
    Privilege escalation in container
    (user=%user.name container=%container.name
     proc=%proc.name evt=%evt.type)
  priority: CRITICAL
  tags: [container, mitre_privilege_escalation]

# Detect crypto mining
- rule: Detect Crypto Mining
  desc: Detect potential cryptocurrency mining
  condition: >
    spawned_process and
    (proc.name in (xmrig, minerd, cpuminer, cgminer, ethminer) or
     proc.cmdline contains stratum+tcp or
     proc.cmdline contains mining.pool)
  output: >
    Crypto mining detected (proc=%proc.name
    cmdline=%proc.cmdline container=%container.name
    user=%user.name)
  priority: CRITICAL
  tags: [cryptomining, mitre_resource_hijacking]
```

## Kubernetes Audit Log

### Enable K8s Audit

```yaml
# Falco Helm values for K8s audit
falco:
  json_output: true
  plugins:
    - name: k8saudit
      library_path: libk8saudit.so
      init_config:
        sslCertificate: /etc/falco/falco.pem
      open_params: "http://:9765/k8s-audit"
  load_plugins:
    - k8saudit
```

### Audit Rules

```yaml
# Detect K8s secret access
- rule: K8s Secret Accessed
  desc: Detect access to Kubernetes secrets
  condition: >
    ka.verb in (get, list) and
    ka.target.resource = secrets and
    not ka.user.name in (system:serviceaccount:kube-system:*)
  output: >
    K8s secret accessed (user=%ka.user.name
    verb=%ka.verb namespace=%ka.target.namespace
    secret=%ka.target.name)
  source: k8s_audit
  priority: WARNING
  tags: [k8s, secrets, mitre_credential_access]

# Detect pod creation with host network
- rule: Pod with Host Network
  desc: Pod created with hostNetwork=true
  condition: >
    ka.verb = create and
    ka.target.resource = pods and
    jevt.value[/requestObject/spec/hostNetwork] = true
  output: >
    Pod with host network created (user=%ka.user.name
    pod=%ka.target.name namespace=%ka.target.namespace)
  source: k8s_audit
  priority: ERROR
  tags: [k8s, network, mitre_privilege_escalation]
```

## Configuration

### falco.yaml

```yaml
# /etc/falco/falco.yaml

# Rule files to load (in order)
rules_file:
  - /etc/falco/falco_rules.yaml
  - /etc/falco/falco_rules.local.yaml
  - /etc/falco/rules.d

# Output settings
json_output: true
json_include_output_property: true
json_include_tags_property: true

# Log level
log_level: info
log_stderr: true
log_syslog: true

# Output channels
stdout_output:
  enabled: true

syslog_output:
  enabled: true

file_output:
  enabled: true
  filename: /var/log/falco/events.log
  keep_alive: false

http_output:
  enabled: true
  url: http://falcosidekick:2801
  keep_alive: true

# Priority filter (minimum level to output)
priority: DEBUG

# Buffer sizes
syscall_buf_size_preset: 4
```

## Falcosidekick

### Deploy Falcosidekick

```bash
# Deploy with Falco
helm install falco falcosecurity/falco \
  --set falcosidekick.enabled=true \
  --set falcosidekick.config.slack.webhookurl="https://hooks.slack.com/services/XXX" \
  --set falcosidekick.config.slack.minimumpriority=warning

# Standalone Falcosidekick
helm install falcosidekick falcosecurity/falcosidekick \
  --set config.slack.webhookurl="https://hooks.slack.com/services/XXX" \
  --set config.pagerduty.routingkey="YOUR-KEY" \
  --set config.elasticsearch.hostport="http://elasticsearch:9200"
```

### Supported Outputs

```bash
# Falcosidekick forwards alerts to 60+ destinations:
# Messaging: Slack, Teams, Discord, Mattermost, Telegram
# Alerting: PagerDuty, OpsGenie, AlertManager
# Storage: Elasticsearch, Loki, InfluxDB, TimescaleDB
# Cloud: AWS (SNS/SQS/Lambda/S3/CloudWatch), GCP (PubSub/Cloud Run)
# SIEM: Splunk, Datadog, Sumo Logic
# Workflow: JIRA, GitHub, GitLab
# Kubernetes: Talon (response engine), policy-report
```

## CLI Operations

### Falco CLI

```bash
# Run Falco in foreground (debug)
sudo falco -r /etc/falco/falco_rules.yaml

# Validate rules syntax
sudo falco -V /etc/falco/falco_rules.local.yaml

# List available fields
sudo falco --list

# List fields for specific source
sudo falco --list syscall
sudo falco --list k8s_audit

# List supported syscalls
sudo falco --list-syscall-events

# Print compiled rules
sudo falco --print-base64

# Test rule against specific event
sudo falco -r rules.yaml -M 30  # run for 30 seconds
```

## Tips

- Use modern-bpf driver for best performance and no kernel header dependency (requires kernel 5.8+)
- Start with the default rules and create overrides in `falco_rules.local.yaml` rather than editing defaults
- Use macros to keep rules DRY; extract common conditions like `container` and `spawned_process`
- Tag rules with MITRE ATT&CK technique IDs for mapping detections to the threat framework
- Use Falcosidekick to fan out alerts to multiple destinations simultaneously
- Set `priority: WARNING` as the minimum output level in production to reduce noise
- Use lists for trusted images and allowed processes that are specific to your environment
- Monitor `falco --list-syscall-events` to understand which syscalls are available for rules
- Use `json_output: true` in production for structured log parsing and SIEM ingestion
- Test custom rules with `-V` flag before deploying to catch syntax errors
- Use the Talon response engine with Falcosidekick to automate incident response actions
- Run Falco as a DaemonSet in Kubernetes to cover all nodes in the cluster

## See Also

- trivy, auditd, apparmor, seccomp, container-security, ebpf

## References

- [Falco Documentation](https://falco.org/docs/)
- [Falco Rules Reference](https://falco.org/docs/reference/rules/)
- [Falco Supported Fields](https://falco.org/docs/reference/rules/supported-fields/)
- [Falcosidekick](https://github.com/falcosecurity/falcosidekick)
- [MITRE ATT&CK for Containers](https://attack.mitre.org/matrices/enterprise/containers/)
