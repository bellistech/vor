# The Architecture of Nornir — Python Network Automation at Scale

> *Nornir replaces YAML-based DSLs with native Python, giving network engineers direct control over inventory, tasks, and execution. Its mathematical foundations are in thread pool scheduling, inventory set algebra, and plugin composition.*

---

## 1. Nornir vs Ansible — Architectural Divergence

### The Problem

Both Nornir and Ansible automate network devices at scale, but their architectures diverge fundamentally in how they express automation logic.

### Ansible: DSL-First Architecture

Ansible uses YAML as a domain-specific language:

```
User Intent → YAML Playbook → Ansible Engine → Module → Device
```

The YAML layer is interpreted at runtime. Control flow (loops, conditionals, error handling) is expressed as YAML constructs (`when`, `loop`, `block/rescue`), not native language features.

**Limitation**: Complex logic in YAML becomes unwieldy. Variable scoping, exception handling, and data transformation require workarounds or custom filters.

### Nornir: Python-Native Architecture

Nornir uses Python directly:

```
User Intent → Python Code → Nornir Core → Plugin → Device
```

Control flow uses native Python (`if`, `for`, `try/except`). Data structures are Python objects. Testing uses pytest. Debugging uses pdb.

### Comparison Matrix

| Dimension | Ansible | Nornir |
|:---|:---|:---|
| Language | YAML DSL | Python |
| Agent | Agentless (SSH) | Agentless (SSH/API) |
| Parallelism | Fork-based (processes) | Thread-based |
| Inventory | INI/YAML + plugins | Python objects + plugins |
| Error handling | `block/rescue/always` | `try/except` |
| Testing | Molecule | pytest |
| Learning curve | Lower (YAML) | Higher (Python) |
| Extensibility | Modules (any language) | Plugins (Python) |
| IDE support | Limited (YAML) | Full (Python) |
| Debugging | `-vvvv` flags | pdb, breakpoints |
| State management | Facts + registered vars | Python objects |
| Community size | Massive | Growing |

### When Nornir Wins

Nornir excels when automation requires:

1. **Complex data transformation** — parsing CLI output, correlating across devices
2. **Custom connection logic** — retry strategies, connection pooling
3. **Integration with Python ecosystem** — pandas, requests, databases
4. **Unit testing** — pytest fixtures, mocking, coverage
5. **Dynamic workflows** — branching logic based on device state

### When Ansible Wins

Ansible excels when:

1. **Team skill level varies** — YAML is accessible to non-developers
2. **Existing playbook library** — large ecosystem of roles and collections
3. **Multi-domain automation** — cloud + network + server in one tool
4. **Audit requirements** — YAML playbooks as human-readable runbooks

---

## 2. Inventory Model Design

### The Problem

Network automation requires a structured representation of devices, their attributes, and their relationships. The inventory model determines how efficiently you can target subsets of your infrastructure.

### Inventory Hierarchy

Nornir's SimpleInventory uses a three-tier model:

```
defaults.yaml → groups.yaml → hosts.yaml
```

Attribute resolution follows child-overrides-parent:

$$\text{effective}(attr) = \text{host}[attr] \oplus \text{group}[attr] \oplus \text{defaults}[attr]$$

Where $\oplus$ represents "use left if defined, otherwise fall through to right."

### Resolution Algorithm

For a host with groups `[g1, g2]`:

1. Check `host.attr` — if defined, return it
2. Check `g1.attr` — if defined, return it (group order matters)
3. Check `g2.attr` — if defined, return it
4. Check `defaults.attr` — if defined, return it
5. Return `None`

### Group Inheritance

Groups can contain parent groups, creating a DAG (Directed Acyclic Graph):

```
defaults
├── dc1
│   ├── spine
│   │   └── spine1, spine2
│   └── leaf
│       └── leaf1, leaf2
└── dc2
    ├── spine
    └── leaf
```

**Key constraint**: The inheritance graph must be acyclic. Circular group references cause infinite recursion.

### Inventory as Set Algebra

The inventory is a universal set $U$ of all hosts. Filtering produces subsets:

$$F_{platform}(\text{ios}) = \{h \in U : h.platform = \text{ios}\}$$

F objects compose with standard set operations:

$$F_1 \cap F_2 = F(platform=\text{ios}) \wedge F(site=\text{dc1})$$
$$F_1 \cup F_2 = F(platform=\text{ios}) \vee F(platform=\text{nxos})$$
$$\overline{F_1} = \neg F(platform=\text{junos})$$

### Inventory Scaling

| Inventory Size | SimpleInventory | Database-backed | NetBox Plugin |
|:---|:---:|:---:|:---:|
| 10 devices | Instant | Overhead | Overhead |
| 100 devices | < 100ms | < 200ms | < 500ms |
| 1,000 devices | < 1s | < 500ms | < 2s |
| 10,000 devices | 5-10s | < 1s | < 5s |

SimpleInventory loads all hosts into memory at init. For very large inventories, database-backed plugins (NetBox, Nautobot) with lazy loading are more efficient.

---

## 3. Plugin System Architecture

### The Problem

Nornir must support diverse network platforms, transport mechanisms, and output formats without coupling the core framework to any specific implementation.

### Plugin Categories

| Category | Purpose | Examples |
|:---|:---|:---|
| Inventory | Device data source | SimpleInventory, NetBox, Nautobot, Ansible |
| Connection | Transport to device | Netmiko, NAPALM, Scrapli, Paramiko |
| Task | Operations on devices | netmiko_send_command, napalm_get, template_file |
| Runner | Execution strategy | Threaded, Serial, RetryRunner |
| Processor | Result observation | Custom logging, metrics, persistence |
| Transform | Inventory mutation | Filter transforms, data enrichment |

### Plugin Registration

Nornir uses Python entry points for plugin discovery:

```
[nornir.plugins.connections]
netmiko = nornir_netmiko.connections:Netmiko
napalm = nornir_napalm.connections:Napalm
scrapli = nornir_scrapli.connection:ScrapliCore
```

At runtime, Nornir resolves plugins by name string to Python class.

### Connection Plugin Lifecycle

```
open() → [task1, task2, ..., taskN] → close()
```

Connections are opened lazily (on first use) and reused across tasks within the same `nr.run()` call. This amortizes connection setup cost across multiple operations.

### Connection Plugin Comparison

| Plugin | Protocol | Platforms | Structured Data | Speed |
|:---|:---|:---|:---:|:---|
| Netmiko | SSH (screen scraping) | 50+ | Via TextFSM/Genie | Moderate |
| NAPALM | SSH/API (abstracted) | 10+ | Native getters | Moderate |
| Scrapli | SSH (async-capable) | Cisco/Arista/Juniper | Via TextFSM/Genie | Fast |
| Paramiko | SSH (raw) | Any SSH device | None | Fast (raw) |

---

## 4. Task Execution Model

### The Problem

Network automation must execute tasks across hundreds or thousands of devices efficiently while handling per-device failures gracefully.

### Execution Flow

```
nr.run(task=T)
    │
    ├── Runner distributes to thread pool
    │   ├── Thread 1: T(host1) → MultiResult
    │   ├── Thread 2: T(host2) → MultiResult
    │   ├── Thread 3: T(host3) → MultiResult
    │   └── ...
    │
    └── AggregatedResult = {host1: MR1, host2: MR2, host3: MR3}
```

### Threading Model

The `ThreadedRunner` uses `concurrent.futures.ThreadPoolExecutor`:

$$\text{batch\_count} = \lceil \frac{N}{W} \rceil$$

Where $N$ = number of hosts, $W$ = `num_workers` (thread count).

**Execution time**:

$$T_{total} = \sum_{b=1}^{\text{batch\_count}} \max_{h \in \text{batch}_b} T_{task}(h)$$

### Optimal Worker Count

The optimal thread count depends on task characteristics:

- **I/O-bound tasks** (SSH commands, API calls): $W = 2N$ to $5N$ is safe since threads spend most time waiting
- **CPU-bound tasks** (parsing, template rendering): $W = \text{CPU cores}$ to avoid GIL contention
- **Mixed tasks**: Start with $W = N$ and tune

**Practical ceiling**: SSH connections consume file descriptors. Most OSes default to 1024 open files per process.

$$W_{max} = \min(\text{num\_hosts}, \text{ulimit} - \text{reserved\_fds})$$

### Task Composition

Tasks can invoke sub-tasks, creating a tree:

```
configure_device (parent task)
├── template_file (sub-task 1 — render config)
├── netmiko_send_config (sub-task 2 — push config)
└── netmiko_send_command (sub-task 3 — verify)
```

Each sub-task appends its Result to the parent's MultiResult. The parent task's Result is the first item; sub-task Results follow in execution order.

---

## 5. Result Aggregation Patterns

### The Problem

When running tasks across many devices, results must be collected, filtered, and acted upon systematically.

### Result Hierarchy

```
AggregatedResult (dict-like)
└── host_name → MultiResult (list-like)
    ├── Result[0] — parent task result
    ├── Result[1] — sub-task 1 result
    └── Result[2] — sub-task 2 result
```

### Result Object Fields

| Field | Type | Description |
|:---|:---|:---|
| `result` | Any | Task output (string, dict, list) |
| `failed` | bool | Whether the task failed |
| `changed` | bool | Whether the task changed device state |
| `diff` | str | Configuration diff (if applicable) |
| `exception` | Exception | Exception object (if failed) |
| `severity_level` | int | Logging severity |
| `name` | str | Task name |
| `host` | Host | Host object |

### Aggregation Strategies

**Collect all outputs**:

$$\text{all\_outputs} = \{h.name: r[0].result \mid (h, r) \in \text{AggregatedResult}\}$$

**Filter failures**:

$$\text{failed} = \{h.name: r.exception \mid (h, r) \in \text{AggregatedResult}, r.failed\}$$

**Compliance ratio**:

$$\text{compliance} = \frac{|\{r \mid \neg r.failed\}|}{|\text{AggregatedResult}|} \times 100\%$$

### Failed Host Tracking

Nornir maintains a global `failed_hosts` set. When a host fails a task, it is added to this set. Subsequent `nr.run()` calls skip failed hosts by default.

This prevents cascading failures — if a device is unreachable, all subsequent tasks skip it automatically.

To retry: `nr.data.reset_failed_hosts()` clears the set.

---

## 6. Nornir in Production

### The Problem

Moving from lab scripts to production automation requires patterns for reliability, observability, and change management.

### Production Architecture

```
┌─────────────┐     ┌───────────────┐     ┌──────────────┐
│ Source of    │────>│ Nornir        │────>│ Network      │
│ Truth       │     │ Application   │     │ Devices      │
│ (NetBox)    │     │               │     │              │
└─────────────┘     └───────┬───────┘     └──────────────┘
                            │
                    ┌───────┴───────┐
                    │ Observability │
                    │ (Logs/Metrics)│
                    └───────────────┘
```

### Key Production Patterns

1. **Inventory from source of truth**: Use NetBox/Nautobot inventory plugin instead of static YAML files
2. **Secrets management**: Pull credentials from HashiCorp Vault, not plaintext YAML
3. **Change windows**: Wrap `nr.run()` in time-window checks before making changes
4. **Dry run first**: Always run with `dry_run=True`, diff, then apply
5. **Rollback on failure**: If any host fails, roll back successfully changed hosts
6. **Structured logging**: Use processors to emit JSON logs for Splunk/ELK
7. **Rate limiting**: Custom runner to limit concurrent connections per platform

### Rollback Pattern

```
1. Backup current config → backups/
2. Generate desired config from templates
3. Diff backup vs desired
4. Apply with dry_run=True → review diffs
5. Apply for real → collect results
6. Verify → run validation tasks
7. If verification fails → rollback from backup
```

### Secrets Integration

Never store credentials in inventory YAML for production:

| Method | Mechanism | Rotation |
|:---|:---|:---|
| Environment variables | `os.environ["DEVICE_PASS"]` | Manual |
| HashiCorp Vault | `hvac` Python client | Automatic |
| AWS Secrets Manager | `boto3` | Automatic |
| CyberArk | REST API | Automatic |

---

## 7. Nornir with CI/CD

### The Problem

Network changes must follow the same rigor as software deployments — version control, review, testing, staged rollout.

### Pipeline Stages

```
┌────────┐   ┌──────────┐   ┌────────┐   ┌────────┐   ┌────────┐
│ Commit │──>│ Validate │──>│ Test   │──>│ Stage  │──>│ Deploy │
│        │   │          │   │ (Lab)  │   │ (Canary│   │ (Prod) │
└────────┘   └──────────┘   └────────┘   └────────┘   └────────┘
```

### Stage Details

| Stage | Tool | What It Does |
|:---|:---|:---|
| Lint | yamllint, pylint | Validate YAML syntax, Python quality |
| Validate | Batfish, custom | Verify config correctness pre-deploy |
| Test | pytest + Nornir | Run against lab devices or mocks |
| Stage | Nornir dry_run | Diff against canary devices |
| Deploy | Nornir | Push to production with rollback |
| Verify | Nornir + NAPALM | Post-deploy validation |

### Testing Nornir Code

```python
# Use pytest fixtures with mock inventory
# nornir_tests/conftest.py:
#
# @pytest.fixture
# def nornir_instance():
#     nr = InitNornir(
#         inventory={
#             "plugin": "SimpleInventory",
#             "options": {
#                 "host_file": "tests/inventory/hosts.yaml",
#             },
#         },
#         runner={"plugin": "serial"},
#     )
#     yield nr
#     nr.close_connections()
```

### Git Workflow for Network Changes

```
feature/add-vlan-500
    │
    ├── inventory changes (if new hosts)
    ├── template changes (Jinja2)
    ├── host_vars changes (per-device data)
    └── tests (pytest)
         │
         └── Pull Request
              ├── CI: lint + validate + lab test
              ├── Review: network engineer approval
              └── Merge → deploy pipeline
```

---

## 8. Threading Deep Dive — GIL and Network Automation

### The Problem

Python's Global Interpreter Lock (GIL) limits true parallelism for CPU-bound work, but network automation is overwhelmingly I/O-bound.

### Why Threads Work for Network Automation

Network tasks spend 95%+ of their time waiting:

- SSH handshake: 200-500ms (waiting for remote)
- Command execution: 100ms-30s (waiting for device)
- API calls: 50-500ms (waiting for response)

During these waits, the GIL is released, allowing other threads to run. The effective parallelism approaches the thread count.

### Thread Scheduling Analysis

For $N$ hosts with $W$ workers, each task taking average time $T_{avg}$:

$$T_{serial} = N \times T_{avg}$$
$$T_{threaded} = \lceil \frac{N}{W} \rceil \times T_{avg}$$
$$\text{Speedup} = \frac{N}{\lceil N/W \rceil} \approx \min(W, N)$$

### Worked Example

100 devices, each command takes 2 seconds:

| Workers | Batches | Total Time | Speedup |
|:---:|:---:|:---:|:---:|
| 1 (serial) | 100 | 200s | 1x |
| 10 | 10 | 20s | 10x |
| 20 | 5 | 10s | 20x |
| 50 | 2 | 4s | 50x |
| 100 | 1 | 2s | 100x |

### Thread Safety Considerations

Nornir's inventory objects are **not thread-safe** for writes. Each task receives its own copy of the host object (`task.host`), but modifications to shared data structures require explicit locking.

Safe: Reading `task.host` attributes within a task.
Unsafe: Modifying `nr.inventory` from within a task.

---

## See Also

- NAPALM
- Ansible
- Network CI/CD
- Python

## References

- Nornir Docs: https://nornir.readthedocs.io/
- "Network Automation with Nornir" — Patrick Ogenstad
- Python threading: https://docs.python.org/3/library/concurrent.futures.html
- Netmiko: https://github.com/ktbyers/netmiko
- NAPALM: https://napalm.readthedocs.io/
