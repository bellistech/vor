# The Engineering of Network CI/CD — Infrastructure as Code for the Network

> *Network CI/CD applies software engineering rigor to network infrastructure changes. Its foundations are in GitOps theory, verification-driven deployment, and risk-managed change propagation across shared physical infrastructure.*

---

## 1. CI/CD for Network Infrastructure

### The Problem

Network changes are historically manual, error-prone, and difficult to test. A single misconfigured ACL or routing policy can cause widespread outages. CI/CD brings the safety net of automated testing to this high-risk domain.

### Software CI/CD vs Network CI/CD

| Dimension | Software CI/CD | Network CI/CD |
|:---|:---|:---|
| Artifact | Container image, binary | Config file, template output |
| Test environment | Identical (containers) | Approximate (lab, simulation) |
| Rollback | Instant (redeploy previous) | Complex (stateful devices) |
| Blast radius | Per-service | Cross-service (shared infra) |
| State | Mostly stateless | Deeply stateful |
| Validation | Unit/integration tests | Config verification + traffic test |
| Deployment | Blue-green, canary | Rolling, maintenance window |

### Why Network CI/CD is Harder

1. **Shared infrastructure**: A router carries traffic for many services simultaneously
2. **Stateful devices**: Routing tables, ARP caches, MAC tables evolve over time
3. **Physical constraints**: Cannot spin up identical physical hardware for testing
4. **Vendor diversity**: Each vendor has different CLI syntax, API models, behaviors
5. **Convergence time**: Routing protocol convergence adds delay to verification
6. **Partial failure modes**: A config can be syntactically valid but operationally broken

---

## 2. GitOps Principles for Networking

### The Problem

GitOps provides a framework for using Git as the single source of truth for declarative infrastructure. Applying this to networking requires adaptation.

### Core GitOps Principles

1. **Declarative**: Desired network state is described, not imperative steps
2. **Versioned**: All changes tracked in Git with full history
3. **Automated**: Changes are applied automatically upon merge
4. **Observable**: Actual state can be compared to desired state

### The GitOps Equation

$$\text{Drift} = \text{Desired State}_{git} - \text{Actual State}_{device}$$

$$\text{Converged} \iff \text{Drift} = \emptyset$$

### GitOps for Network — Adaptations

Traditional GitOps (Flux, ArgoCD) assumes a Kubernetes-like reconciliation loop. Networks require modifications:

**Pull-based reconciliation** (Kubernetes-style) is dangerous for networks because:
- Continuous config pushes can disrupt convergence
- Rate limits on device management planes
- No atomic "apply" across multiple devices

**Event-driven reconciliation** (network-adapted):
```
Git merge → Webhook → Pipeline → Validate → Staged Deploy → Verify
```

The pipeline is the reconciliation loop, triggered by Git events rather than continuous polling.

### Repository Structure

```
network-automation/
├── inventory/
│   ├── hosts.yaml          # device inventory
│   ├── groups.yaml         # group definitions
│   └── defaults.yaml       # default values
├── templates/
│   ├── base.j2             # common config
│   ├── bgp.j2              # BGP config
│   └── acl.j2              # ACL config
├── host_vars/
│   ├── spine1.yaml         # per-device variables
│   └── leaf1.yaml
├── group_vars/
│   ├── spine.yaml          # per-group variables
│   └── leaf.yaml
├── configs/                # rendered configs (generated)
├── validations/
│   ├── bgp.yaml            # validation definitions
│   └── reachability.yaml
├── scripts/
│   ├── render.py           # template rendering
│   ├── deploy.py           # deployment
│   ├── verify.py           # verification
│   └── rollback.py         # rollback
├── tests/
│   ├── test_templates.py   # template unit tests
│   └── test_inventory.py   # inventory validation
└── .github/workflows/
    └── network-cicd.yml    # CI/CD pipeline
```

---

## 3. Testing Pyramid for Network Automation

### The Problem

Testing network changes requires a layered approach because each layer catches different classes of errors at different costs.

### The Network Testing Pyramid

```
          ╱ ╲
         ╱   ╲         Production traffic tests
        ╱ E2E ╲        (most expensive, highest fidelity)
       ╱───────╲
      ╱         ╲       Lab/staging deployment
     ╱ Integr.   ╲      (moderate cost, good fidelity)
    ╱─────────────╲
   ╱               ╲     Config verification (Batfish)
  ╱   Verification  ╲    (low cost, high coverage)
 ╱───────────────────╲
╱                     ╲   Syntax, schema, template tests
╱      Unit Tests      ╲  (cheapest, fastest)
╱───────────────────────╲
```

### Layer Details

| Layer | What It Catches | Tools | Cost | Speed |
|:---|:---|:---|:---:|:---:|
| Unit | Syntax errors, schema violations, template bugs | yamllint, pytest, Jinja2 | Low | Seconds |
| Verification | Routing loops, unreachable prefixes, ACL shadows | Batfish, custom validators | Low | Minutes |
| Integration | Connection failures, unexpected CLI behavior | Lab devices, containerlab | Medium | Minutes |
| E2E | Traffic loss, convergence issues, performance | Production canary | High | Minutes-hours |

### Error Detection Coverage

$$P(\text{catch}) = 1 - \prod_{i=1}^{4}(1 - P_i(\text{catch}))$$

Where $P_i$ is the probability of layer $i$ catching a given error class.

| Error Class | Unit | Verify | Integration | E2E |
|:---|:---:|:---:|:---:|:---:|
| YAML syntax | 99% | — | — | — |
| Missing variable | 90% | 50% | 95% | 99% |
| Routing loop | 0% | 95% | 80% | 99% |
| ACL shadow | 0% | 90% | 70% | 95% |
| Performance | 0% | 0% | 30% | 90% |
| Vendor bug | 0% | 0% | 50% | 80% |

---

## 4. Batfish Network Verification

### The Problem

Testing network config changes on live devices is risky. Batfish creates a mathematical model of the network from config files and answers questions about behavior without touching real devices.

### How Batfish Works

```
Config files → Parser → Network Model → Query Engine → Results
                              │
                        (routing tables,
                         ACLs, NAT rules,
                         interface state)
```

Batfish builds a complete forwarding model:

1. **Parse** vendor-specific configs into a normalized model
2. **Compute** routing tables (OSPF SPF, BGP best path, static routes)
3. **Evaluate** ACLs, NAT rules, firewall policies
4. **Answer** reachability, path, and compliance queries

### What Batfish Can Verify

| Query | Question It Answers |
|:---|:---|
| `undefinedReferences` | Are there references to objects that don't exist? |
| `unusedStructures` | Are there ACLs/route-maps never applied? |
| `traceroute` | What path does traffic take from A to B? |
| `reachability` | Can host A reach host B on port X? |
| `detectLoops` | Are there forwarding loops? |
| `searchFilters` | Which ACL rules match a given flow? |
| `compareConfigs` | What changed between two snapshots? |
| `bgpSessionStatus` | Are BGP sessions properly configured? |

### Verification as Mathematical Proof

Batfish performs **exhaustive reachability analysis**, not sampling. For a given set of configs, it can prove:

$$\forall p \in \text{Packets}: \text{path}(p) \neq \text{loop}$$

This is fundamentally different from testing (which samples) — it is formal verification of the forwarding plane.

### Batfish Limitations

1. **Config parsing coverage**: Not all vendor features are supported
2. **Data plane only**: Cannot model control plane convergence timing
3. **Static analysis**: Cannot model dynamic state (ARP, MAC learning)
4. **Performance**: Large networks (10,000+ devices) require significant memory

---

## 5. Deployment Risk Management

### The Problem

Network deployments carry inherent risk because network infrastructure is shared. A deployment strategy must minimize blast radius while maintaining velocity.

### Blast Radius Analysis

$$\text{Blast Radius} = \text{Affected Services} \times \text{Duration} \times \text{Severity}$$

### Deployment Strategies

| Strategy | Blast Radius | Speed | Complexity | Use Case |
|:---|:---:|:---:|:---:|:---|
| Big bang | Maximum | Fast | Low | Emergency/simple changes |
| Rolling | Gradual | Moderate | Medium | Routine changes |
| Canary | Minimal | Slow | High | High-risk changes |
| Blue-green | Binary | Fast | Very high | Full config replace |

### Canary Deployment for Networks

```
Phase 1: Deploy to 1 canary device (1% of traffic)
    → Verify: BGP established, traffic flowing, no errors
    → Wait: 15 minutes observation
Phase 2: Deploy to 10% of devices
    → Verify: Same checks + aggregate metrics
    → Wait: 30 minutes observation
Phase 3: Deploy to 50% of devices
    → Verify: Full validation suite
    → Wait: 1 hour observation
Phase 4: Deploy to 100%
    → Verify: Full validation
    → Monitor: 24 hours
```

### Rollback Decision Tree

```
Deploy → Verify
           │
     ┌─────┴─────┐
     │ Pass?      │
     │            │
    Yes          No
     │            │
   Done     ┌────┴────┐
             │ Auto    │
             │ rollback│
             │ safe?   │
             │         │
            Yes       No
             │         │
          Rollback   Alert
          auto       on-call
```

### Change Risk Score

$$\text{Risk} = \sum_{i} w_i \times f_i$$

| Factor ($f_i$) | Weight ($w_i$) | Low (1) | Medium (3) | High (5) |
|:---|:---:|:---|:---|:---|
| Scope | 3 | Single device | Device group | All devices |
| Reversibility | 2 | Config only | Routing change | Firmware |
| Time sensitivity | 1 | Anytime | Business hours | Peak traffic |
| Precedent | 2 | Done before | Similar | Novel |
| Validation | 1 | Batfish + lab | Batfish only | Manual review |

$$\text{Risk}_{max} = 3(5) + 2(5) + 1(5) + 2(5) + 1(5) = 45$$

Risk thresholds:
- $\leq 15$: Auto-deploy with pipeline
- $16\text{-}30$: Requires peer review + manual approval
- $> 30$: Change advisory board + maintenance window

---

## 6. Network as Code Maturity Model

### The Problem

Organizations adopting network automation progress through stages. Understanding the maturity model helps plan the journey.

### Maturity Levels

| Level | Name | Characteristics |
|:---:|:---|:---|
| 0 | Manual | CLI copy-paste, no version control |
| 1 | Scripts | Ad-hoc scripts, some version control |
| 2 | Templates | Jinja2 templates, structured inventory |
| 3 | Pipeline | CI/CD pipeline, automated testing |
| 4 | GitOps | Git as source of truth, drift detection |
| 5 | Self-healing | Closed-loop automation, intent-based |

### Progression Path

$$\text{Level 0} \xrightarrow{\text{Git}} \text{Level 1} \xrightarrow{\text{Templates}} \text{Level 2} \xrightarrow{\text{CI/CD}} \text{Level 3} \xrightarrow{\text{SoT}} \text{Level 4} \xrightarrow{\text{Telemetry}} \text{Level 5}$$

### Key Enablers at Each Level

| Transition | Key Enabler | Typical Timeline |
|:---|:---|:---|
| 0 → 1 | Git adoption, first Python scripts | 1-3 months |
| 1 → 2 | Jinja2, structured data (YAML), Nornir/Ansible | 3-6 months |
| 2 → 3 | CI/CD platform, Batfish, lab environment | 6-12 months |
| 3 → 4 | Source of truth (NetBox), drift detection | 12-18 months |
| 4 → 5 | Streaming telemetry, closed-loop remediation | 18-36 months |

### Metrics per Level

| Metric | L0 | L1 | L2 | L3 | L4 | L5 |
|:---|:---:|:---:|:---:|:---:|:---:|:---:|
| Change lead time | Days | Hours | Hours | Minutes | Minutes | Seconds |
| Change failure rate | 30%+ | 20% | 10% | 5% | 2% | <1% |
| MTTR | Hours | Hours | 30min | 15min | 5min | Auto |
| Audit coverage | 0% | 20% | 50% | 90% | 100% | 100% |

---

## 7. Source of Truth Architecture

### The Problem

Network automation requires a single source of truth (SoT) that defines intended state. Without it, configuration drift is inevitable and undetectable.

### Source of Truth Components

```
┌─────────────────────────────────┐
│         Source of Truth         │
│                                 │
│  ┌───────┐  ┌───────┐  ┌─────┐│
│  │ IPAM  │  │ Device│  │VLAN ││
│  │       │  │ Roles │  │Mgmt ││
│  └───────┘  └───────┘  └─────┘│
│  ┌───────┐  ┌───────┐  ┌─────┐│
│  │Circuit│  │Tenant │  │Cable││
│  │Mgmt   │  │Mgmt   │  │Mgmt ││
│  └───────┘  └───────┘  └─────┘│
└────────────────┬────────────────┘
                 │ API
    ┌────────────┼────────────────┐
    │            │                │
    ▼            ▼                ▼
┌────────┐  ┌────────┐    ┌──────────┐
│Nornir  │  │Ansible │    │Terraform │
│Pipeline│  │Pipeline│    │Pipeline  │
└────────┘  └────────┘    └──────────┘
```

### SoT Platform Comparison

| Feature | NetBox | Nautobot | Infrahub |
|:---|:---:|:---:|:---:|
| IPAM | Yes | Yes | Yes |
| DCIM | Yes | Yes | Yes |
| Config contexts | Yes | Yes | Yes |
| Git integration | Plugin | Native | Native |
| GraphQL API | Plugin | Native | Native |
| Multi-tenancy | Yes | Yes | Yes |
| Extensibility | Plugins | Apps + Jobs | Schema-driven |
| Version control | Limited | Git data sources | Git-native |

### SoT Design Principles

1. **Single source**: One system owns each data domain (IPs, VLANs, devices)
2. **API-first**: All data accessible programmatically
3. **Validated**: Data integrity enforced at input time
4. **Versioned**: Changes tracked with who/when/why
5. **Integrated**: Pipeline consumes SoT data automatically

### Drift Detection

$$\text{Drift Score} = \frac{|\text{Desired} \triangle \text{Actual}|}{|\text{Desired}|}$$

Where $\triangle$ is the symmetric difference — lines present in one but not the other.

A drift score of 0 means perfect convergence. Automated drift detection runs on a schedule (hourly/daily), compares SoT-rendered configs against actual device configs, and alerts on divergence.

---

## See Also

- Nornir
- Ansible
- GitHub Actions
- GitLab CI
- Model-Driven Telemetry

## References

- Batfish: https://www.batfish.org/
- "Network Programmability and Automation" — O'Reilly
- GitOps Principles: https://opengitops.dev/
- NetBox: https://netbox.dev/
- Nautobot: https://www.networktocode.com/nautobot/
- pyATS: https://developer.cisco.com/pyats/
