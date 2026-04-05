# The Engineering of Cisco NSO — Transaction Theory, FASTMAP Algorithm, and Network Orchestration at Scale

> *NSO applies database transaction semantics to network configuration: ACID guarantees across hundreds of multi-vendor devices, a single source of truth in CDB, and the FASTMAP algorithm that transforms declarative service intent into per-device configuration diffs. It is, fundamentally, a distributed database engine for network state.*

---

## 1. NSO Transaction Model (ACID Over the Network)

### The Problem

Network configuration changes are inherently unreliable: devices may reject commands, connections drop mid-push, partial configurations create inconsistent states. Traditional automation (scripts, Ansible) operates without transactional guarantees — if device 3 of 10 fails, devices 1-2 have already changed.

### ACID Properties in NSO

| Property | Database Meaning | NSO Implementation |
|----------|-----------------|-------------------|
| Atomicity | All or nothing | All devices succeed or all roll back |
| Consistency | Valid state before and after | YANG model validation before commit |
| Isolation | Concurrent transactions do not interfere | Lock-based serialization in CDB |
| Durability | Committed data survives crashes | CDB persists to disk, rollback files |

### Transaction Lifecycle

```
1. BEGIN TRANSACTION
   |
   v
2. VALIDATION PHASE
   - YANG model validation (type, range, pattern, must, when)
   - Custom validation callbacks (Java/Python)
   - Service FASTMAP execution (compute per-device changes)
   - Reference integrity checks (leafref, if-feature)
   |
   v
3. PREPARE PHASE
   - Calculate per-device diffs
   - Send configurations to all devices (but don't commit yet)
   - CLI NEDs: Send commands, verify no errors
   - NETCONF NEDs: Send <edit-config> with <candidate> datastore
   |
   v
4. COMMIT PHASE (two-phase commit)
   Phase 1: All devices confirm "prepare OK"
     - If ANY device fails --> ABORT (rollback all prepared changes)
   Phase 2: All devices commit
     - CLI NEDs: Configuration already applied in prepare
     - NETCONF NEDs: Send <commit> to confirm candidate
   |
   v
5. PERSIST
   - Write changes to CDB
   - Create rollback file
   - Update service metadata (FASTMAP reference counts)
```

### Two-Phase Commit Across Devices

```
NSO Transaction Manager
  |
  |-- PREPARE --> Device A (CLI NED)  --> Config applied, waiting
  |-- PREPARE --> Device B (NETCONF)  --> Candidate edited, waiting
  |-- PREPARE --> Device C (CLI NED)  --> Config applied, waiting
  |
  | All devices report PREPARE OK?
  |
  |-- YES --> COMMIT ALL
  |     |-- Device A: (already applied in CLI)
  |     |-- Device B: <commit/> (confirm candidate)
  |     |-- Device C: (already applied in CLI)
  |
  |-- NO (Device B failed) --> ABORT ALL
        |-- Device A: Send rollback commands (reverse CLI)
        |-- Device B: <discard-changes/> (discard candidate)
        |-- Device C: Send rollback commands (reverse CLI)
```

### Transaction Isolation

NSO serializes transactions through CDB locking:

$$T_{throughput} = \frac{1}{T_{validation} + T_{prepare} + T_{commit}}$$

For concurrent service deployments, this creates a bottleneck. The commit queue solves this:

```
Without commit queue (serial):
  T1: [validate][prepare][commit] --> T2: [validate][prepare][commit]
  Total: T1 + T2

With commit queue (parallel device push):
  T1: [validate] --> queue --> [prepare+commit to devices]
  T2: [validate] --> queue --> [prepare+commit to devices]
  CDB lock held only during validation, not during device communication
  Total: max(T1_validate, T2_validate) + max(T1_push, T2_push)
```

### Commit Queue Architecture

```
Service Request --> CDB Transaction --> Commit Queue --> Device Push
                    (fast: ms)          (async)         (slow: seconds)

Queue Item States:
  - queued:    Waiting for device push
  - executing: Currently pushing to devices
  - completed: All devices confirmed
  - failed:    One or more devices failed
  - timeout:   Device push timed out

Failure handling options:
  - rollback-on-error: Undo changes on all devices
  - stop-on-error: Stop queue, leave partial state
  - continue-on-error: Skip failed device, continue others
  - reconnect-and-retry: Retry failed devices
```

---

## 2. CDB (Configuration Database) Architecture

### CDB Design

CDB is NSO's central data store. It is NOT a general-purpose database — it is purpose-built for YANG-modeled network configuration.

```
CDB Internal Structure:

+--------------------------------------------------+
|                    CDB                            |
|                                                   |
|  +-------------------+  +---------------------+  |
|  | Running Datastore |  | Operational Store   |  |
|  | (configuration)   |  | (status, counters)  |  |
|  |                   |  |                     |  |
|  | - Device configs  |  | - Device status     |  |
|  | - Service models  |  | - Transaction log   |  |
|  | - System settings |  | - Alarm state       |  |
|  +-------------------+  +---------------------+  |
|                                                   |
|  +-------------------+  +---------------------+  |
|  | Candidate (opt.)  |  | Startup (optional)  |  |
|  | (staged changes)  |  | (boot config)       |  |
|  +-------------------+  +---------------------+  |
|                                                   |
|  Storage: Memory-mapped files (LMDB-based)        |
|  Indexing: XPath-optimized tree                    |
|  Locking: Write lock + read snapshots              |
+--------------------------------------------------+
```

### CDB vs Traditional Databases

| Feature | CDB | PostgreSQL / MySQL |
|---------|-----|-------------------|
| Schema | YANG models (compiled) | SQL DDL |
| Query language | XPath | SQL |
| Transactions | ACID | ACID |
| Replication | Built-in HA (active/standby) | Various (streaming, logical) |
| Indexing | XPath tree paths | B-tree, hash, GiST |
| Subscriptions | CDB subscribers (change notification) | Triggers, LISTEN/NOTIFY |
| Typical size | Hundreds of MB (device configs) | GB-TB |
| Performance | Optimized for config read/write | General-purpose |

### CDB Subscriptions

CDB supports reactive programming through subscriptions — code that runs when specific data changes:

```python
# Python CDB subscription example
import ncs

class ConfigChangeSubscriber(ncs.cdb.Subscriber):
    def init(self):
        # Subscribe to changes under /devices/device
        self.register('/devices/device', priority=100)

    def pre_iterate(self):
        return []  # State passed to iterate()

    def iterate(self, kp, op, old_val, new_val, state):
        # kp = keypath of changed element
        # op = MOP_CREATED, MOP_DELETED, MOP_MODIFIED, MOP_VALUE_SET
        if op == ncs.MOP_MODIFIED:
            self.log.info(f'Device config changed: {kp}')
            state.append(str(kp))
        return ncs.ITER_CONTINUE

    def post_iterate(self, state):
        # Called after all changes processed
        for path in state:
            self.log.info(f'Processing change: {path}')
        # Trigger compliance check, notification, etc.
```

### CDB Storage Mechanics

CDB uses memory-mapped files for performance:

$$T_{read} = O(d)$$

Where $d$ = depth of the YANG tree path. Reads are tree traversals, not table scans.

Write performance depends on journal sync:

$$T_{write} = T_{tree\_update} + T_{journal\_fsync}$$

For large configurations (100K+ nodes), CDB memory usage:

$$M_{CDB} \approx N_{nodes} \times (S_{key} + S_{value} + S_{metadata})$$

Typical overhead is 200-500 bytes per YANG node, so a device with 10,000 config lines uses approximately 2-5 MB in CDB.

---

## 3. The FASTMAP Algorithm

### The Core Problem

Service orchestration must handle three operations:
1. **Create**: Deploy new service instance
2. **Modify**: Change service parameters (add/remove endpoints)
3. **Delete**: Remove all device configuration created by the service

Traditional approach: Write three separate code paths. This is error-prone and fragile.

### FASTMAP Solution

FASTMAP requires the developer to write only the **create** operation. NSO automatically derives modify and delete:

```
FASTMAP Principle:

  create(service_params) --> device_config

  This is a PURE FUNCTION: same inputs always produce same outputs.

  Modify = create(new_params) - create(old_params)
  Delete = -create(old_params)

NSO tracks the mapping:
  Service instance X --> created these CDB nodes: [A, B, C, D]

  When X is modified:
    1. Record current mapping: [A, B, C, D]
    2. Run create(new_params) --> produces: [A, B', C, E]
    3. Compute diff:
       - Keep: A, C (unchanged)
       - Modify: B --> B' (changed)
       - Delete: D (no longer produced)
       - Create: E (new)
    4. Apply diff to devices

  When X is deleted:
    1. Remove everything FASTMAP recorded: [A, B, C, D]
    2. But ONLY if no other service also created those nodes
```

### Reference Counting

Multiple services may configure the same device node (e.g., two VPNs using the same BGP neighbor):

```
Service VPN-A creates:
  /devices/device[pe1]/config/bgp/neighbor[10.0.0.1]
  /devices/device[pe1]/config/bgp/neighbor[10.0.0.1]/remote-as = 65001

Service VPN-B creates:
  /devices/device[pe1]/config/bgp/neighbor[10.0.0.1]
  /devices/device[pe1]/config/bgp/neighbor[10.0.0.1]/remote-as = 65001

Reference count for /bgp/neighbor[10.0.0.1]:
  VPN-A: 1
  VPN-B: 1
  Total: 2

When VPN-A is deleted:
  Decrement reference count: 2 --> 1
  Node is NOT removed (VPN-B still needs it)

When VPN-B is deleted:
  Decrement reference count: 1 --> 0
  Node IS removed (no service references it)
```

### FASTMAP Conflict Resolution

What if two services set the same leaf to different values?

```
Service VPN-A sets:  /device[pe1]/config/qos/bandwidth = 100
Service VPN-B sets:  /device[pe1]/config/qos/bandwidth = 200

Conflict! FASTMAP rules:
  - Last writer wins (order of service deployment matters)
  - NSO logs a warning
  - Re-deploy of VPN-A may overwrite VPN-B's value

Best practice: Design services to own distinct subtrees.
Use resource pools (IP addresses, VLANs) to avoid conflicts.
```

---

## 4. NED Abstraction Layer

### How CLI NEDs Work

CLI NEDs translate between YANG models and device CLI syntax:

```
YANG Model (what NSO sees):
  /devices/device[ce0]/config/interface/Loopback[100]/ip/address = 10.0.0.1

CLI NED Translation Layer:
  YANG --> CLI (south-bound, config push):
    interface Loopback100
     ip address 10.0.0.1 255.255.255.255
     no shutdown

  CLI --> YANG (north-bound, sync-from):
    Parse "show running-config"
    Match regex patterns to YANG paths
    Populate CDB nodes

The NED contains:
  - YANG models (device-specific data model)
  - CLI parser rules (regex patterns for show output)
  - CLI template rules (how to generate CLI commands from YANG)
  - Diff calculator (how to compute minimal CLI changes)
```

### NED Diff Calculation

When NSO needs to modify a device configuration, the NED must compute the minimal CLI diff:

```
Current config on device:
  interface GigabitEthernet0/0
   ip address 10.0.0.1 255.255.255.0
   speed 1000
   no shutdown

Desired config in CDB:
  interface GigabitEthernet0/0
   ip address 10.0.0.2 255.255.255.0
   speed auto
   no shutdown

NED computes diff:
  interface GigabitEthernet0/0
   ip address 10.0.0.2 255.255.255.0    (changed)
   speed auto                             (changed)
                                          (shutdown unchanged, omitted)

Only changed lines are sent. The NED understands:
  - Which commands require "no" prefix to remove
  - Which commands are "replace" vs "merge"
  - Ordered vs unordered lists
  - Interface naming conventions per platform
```

### NETCONF NED vs CLI NED

```
CLI NED path:
  CDB YANG --> NED CLI generator --> SSH --> Device CLI parser --> Running config
  Weaknesses:
    - Screen scraping is fragile (format changes break parsing)
    - Some commands have side effects not visible in show output
    - Ordered lists are hard to diff

NETCONF NED path:
  CDB YANG --> NETCONF <edit-config> --> SSH --> Device YANG datastore
  Strengths:
    - Model-to-model (no screen scraping)
    - Native transaction support (<candidate> + <commit>)
    - Structured error reporting
    - Standard protocol (RFC 6241)

Performance comparison:
  CLI NED:     100-500 ms per device (SSH + parse)
  NETCONF NED: 50-200 ms per device (structured XML)
  Generic NED: Variable (depends on API)
```

---

## 5. Service Lifecycle Management

### Service States

```
Service Instance Lifecycle:

  [Not Deployed] --> deploy --> [Deployed]
                                   |
                            +------+------+
                            |             |
                       re-deploy     un-deploy
                            |             |
                            v             v
                       [Modified]   [Not Deployed]

  Internal states tracked by NSO:
  - Service meta-data (creation time, last modified, owner)
  - FASTMAP backpointers (which CDB nodes this service created)
  - Plan state (for nano services: which steps completed)
  - Device list (which devices are touched by this service)
```

### Re-Deploy Strategies

```bash
# Re-deploy a single service (re-run FASTMAP)
admin@nso> request l3vpn CUSTOMER-A re-deploy

# Re-deploy with dry-run (see what would change)
admin@nso> request l3vpn CUSTOMER-A re-deploy dry-run

# Re-deploy all instances of a service type
admin@nso> request l3vpn * re-deploy

# Deep re-deploy (recalculate from scratch, ignore cached state)
admin@nso> request l3vpn CUSTOMER-A re-deploy reconcile

# Re-deploy after NED upgrade (device model changed)
admin@nso> request devices device ce0 sync-from
admin@nso> request l3vpn CUSTOMER-A re-deploy
```

### Service Health Monitoring

```
Service health check flow:

1. Periodic check-sync on all managed devices
   --> Detect out-of-band changes (someone CLI'd the device directly)

2. Service re-deploy dry-run
   --> Detect drift between service intent and device config

3. If drift detected:
   a. re-deploy (override out-of-band changes)
   b. sync-from (accept out-of-band changes into CDB)
   c. Alert (notify operator, do not auto-remediate)

Automation pattern:
  Scheduled job: Every 15 minutes
    for each device:
      check-sync
      if out-of-sync:
        log alarm
        for each service touching device:
          re-deploy dry-run
          if changes needed:
            create incident ticket
```

---

## 6. LSA Architecture for Scaling

### Why LSA?

Single-node NSO has practical limits:

| Dimension | Single NSO Limit | LSA Solution |
|-----------|------------------|-------------|
| Device count | ~5,000-10,000 | Distribute across lower nodes |
| NED diversity | All NEDs loaded in one JVM | Different NEDs per lower node |
| Team ownership | Shared config, conflict risk | Domain isolation |
| Upgrade risk | One upgrade affects everything | Rolling upgrades per node |
| Transaction size | Large transactions are slow | Smaller per-domain transactions |

### LSA Communication

```
Upper NSO (CFS)                    Lower NSO (RFS)
  |                                   |
  | [CFS service creates RFS          |
  |  service instance via NETCONF]    |
  |                                   |
  |--- <edit-config> --------------->|
  |    (RFS service params)          |
  |                                   |
  |                                   | [RFS FASTMAP runs]
  |                                   | [Pushes to actual devices]
  |                                   |
  |<-- <rpc-reply> ok ---------------|
  |                                   |
  | [Upper NSO treats lower NSO      |
  |  as just another "device"]       |

Key insight: The lower NSO appears to the upper NSO as a
"device" with a NETCONF NED. The upper NSO does not know
or care about the actual network devices underneath.
```

### CFS-to-RFS Mapping

$$\text{CFS}(\text{customer intent}) \xrightarrow{\text{CFS service code}} \text{RFS}_1(\text{DC params}) + \text{RFS}_2(\text{WAN params})$$

```python
# CFS service code example
class CfsVpnService(Service):
    @Service.create
    def cb_create(self, tctx, root, service, proplist):
        # Decompose customer intent into domain-specific RFS instances

        # DC domain (lower NSO #1)
        dc_rfs = root.devices.device['lower-nso-dc'].config
        dc_vpn = dc_rfs.rfs_dc_vpn.create(service.name)
        dc_vpn.vni = service.vni
        dc_vpn.endpoints = service.dc_endpoints

        # WAN domain (lower NSO #2)
        wan_rfs = root.devices.device['lower-nso-wan'].config
        wan_vpn = wan_rfs.rfs_wan_vpn.create(service.name)
        wan_vpn.rd = service.route_distinguisher
        wan_vpn.endpoints = service.wan_endpoints
```

---

## 7. NSO in OSS/BSS Integration

### NSO's Place in the Telecom Stack

```
BSS (Business Support Systems)
  |  CRM, Billing, Order Management
  |
  v
OSS (Operations Support Systems)
  |  Service Catalog, Inventory, Assurance
  |
  v
+-------------------+
|  Orchestrator     |  NSO sits here
|  (Cisco NSO)      |  Translates service orders into
|                   |  network configuration
+-------------------+
  |
  v
Network Infrastructure
  Routers, Switches, Firewalls, Load Balancers
```

### Integration Patterns

| Pattern | Protocol | Use Case |
|---------|----------|----------|
| Northbound REST | RESTCONF | OSS/BSS order fulfillment |
| Northbound NETCONF | NETCONF | Controller-to-controller |
| Kafka/Message Bus | Generic NED | Event-driven orchestration |
| Webhook callbacks | HTTP POST | Async status notification |
| YANG-based inventory | CDB subscription | Real-time inventory sync |

---

## 8. NSO vs Ansible vs Terraform for Network Automation

### Fundamental Differences

| Dimension | NSO | Ansible | Terraform |
|-----------|-----|---------|-----------|
| Paradigm | Model-driven, transactional | Task-driven, procedural | Resource-driven, declarative |
| State management | CDB (always in sync) | Stateless (check each run) | State file (terraform.tfstate) |
| Transaction model | ACID across all devices | None (best-effort per device) | None (per-resource) |
| Rollback | Native (rollback files) | Manual (write reverse playbook) | terraform destroy + re-apply |
| Service abstraction | YANG service models + FASTMAP | Roles (convention, not enforced) | Modules (no service lifecycle) |
| Multi-vendor | NED per vendor (deep integration) | Module per vendor (varies) | Provider per vendor (varies) |
| Scale | 5,000-10,000+ devices | ~500 devices (SSH scaling) | Cloud APIs (scales well) |
| Network focus | Purpose-built for network | General-purpose + network modules | Cloud-first, network secondary |
| Learning curve | Steep (YANG, NED, CDB) | Low (YAML, SSH) | Medium (HCL, state) |
| Cost | Commercial license (expensive) | Free (open source) + Tower/AAP | Free (open source) + Cloud/Enterprise |

### When to Use Each

```
Use NSO when:
  - Multi-vendor network with 500+ devices
  - Need transactional guarantees (rollback on failure)
  - Service lifecycle management (create/modify/delete tracking)
  - Telecom/SP environment with OSS/BSS integration
  - Compliance: must know exactly what changed and why

Use Ansible when:
  - Smaller network (< 500 devices)
  - Mixed infrastructure (servers + network)
  - Simple push-config automation
  - Team already knows Ansible
  - Budget constraint (no NSO license)

Use Terraform when:
  - Cloud-native infrastructure (AWS, Azure, GCP)
  - Infrastructure provisioning (VMs, VPCs, LBs)
  - GitOps workflow desired
  - Network is primarily cloud-based (VPC, SD-WAN API)
```

### Hybrid Architecture

Many organizations use NSO + Ansible/Terraform together:

```
Terraform: Provision cloud infrastructure (VPCs, VMs, cloud firewalls)
     |
     v
NSO: Configure network devices (routers, switches, on-prem firewalls)
     |
     v
Ansible: Configure servers and applications (OS, packages, services)

Integration:
  Terraform outputs (IP addresses, VPC IDs)
    --> NSO service inputs (device IPs, VRF parameters)
      --> Ansible inventory (server IPs, roles)
```

---

## 9. NSO Performance Optimization

### Transaction Performance

$$T_{total} = T_{validation} + T_{prepare} + T_{commit} + T_{CDB\_write}$$

Optimization strategies:

| Bottleneck | Cause | Solution |
|------------|-------|----------|
| Validation slow | Complex YANG must/when expressions | Simplify constraints, cache evaluations |
| Prepare slow | Many devices, slow SSH | Commit queue (async), parallel push |
| Commit slow | Large transaction, many CDB nodes | Batch service deployments |
| CDB write slow | Large datastore, disk I/O | SSD storage, CDB compaction |

### Commit Queue Tuning

```
# Optimal commit queue settings for large deployments

admin@nso(config)# devices global-settings commit-queue enabled-by-default true
admin@nso(config)# devices global-settings commit-queue connection-timeout 120
admin@nso(config)# devices global-settings commit-queue retry-attempts 3
admin@nso(config)# devices global-settings commit-queue retry-timeout 30
admin@nso(config)# devices global-settings commit-queue error-option rollback-on-error
```

### CDB Optimization

```
CDB performance tuning:

1. Compaction: CDB files grow over time. Schedule periodic compaction:
   admin@nso> request system cdb compact

2. Operational data: Do NOT store high-churn data in CDB.
   Use external database (Elasticsearch, InfluxDB) for:
   - Interface counters
   - Alarm history
   - Transaction logs

3. Subscription priority: Set correct priorities to avoid
   cascade re-evaluations:
   - Priority 1-10: Critical infrastructure changes
   - Priority 50-100: Service mapping callbacks
   - Priority 200+: Logging, notification, non-critical

4. Read performance: Use CDB sessions (not transactions) for
   read-only operations. Transactions acquire locks.
```

### NED Performance

```
CLI NED optimization:
  - Use "ned-settings" to tune SSH timeouts
  - Enable connection pooling (keep SSH sessions alive)
  - Use "write-timeout" to handle slow devices
  - Minimize "show running-config" scope with filters

  admin@nso(config)# devices device ce0 ned-settings
  admin@nso(config-ned-settings)# connection-settings number-of-retries 3
  admin@nso(config-ned-settings)# connection-settings time-between-retries 5
  admin@nso(config-ned-settings)# connection-settings pool-idle-timeout 300

NETCONF NED optimization:
  - Use <candidate> datastore (supports native two-phase commit)
  - Use subtree filtering in <get-config> (reduce data transfer)
  - Enable confirmed-commit for extra safety

Device sync optimization:
  - Partial sync-from (specific subtrees only)
  - Schedule sync-from during maintenance windows
  - Use check-sync (fast) before full sync-from (slow)
```

---

## See Also

- ansible, terraform, salt, puppet, chef

## References

- [Cisco NSO Architecture White Paper](https://www.cisco.com/c/en/us/solutions/service-provider/network-services-orchestrator/white-paper.html)
- [NSO FASTMAP and Reactive FASTMAP Documentation](https://developer.cisco.com/docs/nso/guides/)
- [NSO Layered Service Architecture Guide](https://developer.cisco.com/docs/nso/guides/)
- [RFC 6241 — NETCONF Configuration Protocol](https://www.rfc-editor.org/rfc/rfc6241)
- [RFC 6020 — YANG 1.0](https://www.rfc-editor.org/rfc/rfc6020)
- [RFC 7950 — YANG 1.1](https://www.rfc-editor.org/rfc/rfc7950)
- [RFC 8040 — RESTCONF Protocol](https://www.rfc-editor.org/rfc/rfc8040)
- [RFC 8342 — Network Management Datastore Architecture (NMDA)](https://www.rfc-editor.org/rfc/rfc8342)
- [Cisco NSO Scaling Guide](https://www.cisco.com/c/en/us/td/docs/net_mgmt/network_services_orchestrator/admin_guide.html)
