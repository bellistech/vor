# Junos Monitoring & Troubleshooting — Deep Dive

> Beyond the cheat sheet: systematic troubleshooting methodology, SNMP MIB trees,
> traceoptions internals, automation with event/op/commit scripts, and the Junos
> XML API (RPCs) for programmatic monitoring. JNCIA-Junos and beyond.

## Prerequisites

- Solid grasp of Junos CLI (operational and configuration modes).
- Familiarity with basic show/monitor commands (see `sheets/juniper/junos-monitoring.md`).
- Understanding of OSPF, BGP, and IP routing fundamentals.
- For automation sections: basic knowledge of XML, XSLT, or SLAX syntax.

---

## 1. Systematic Troubleshooting Methodology

Junos troubleshooting follows a layered, bottom-up approach. Work through each
layer before moving up; most problems are physical or configuration errors.

### 1.1 The Layer-by-Layer Approach

**Layer 1 — Physical:**

- Check `show interfaces ge-0/0/0 extensive` for link state and error counters.
- Look at carrier transitions (link flaps), CRC errors, framing errors.
- Verify SFP optics: `show interfaces diagnostics optics ge-0/0/0`.
- Check cable seating, patch panel connections, SFP compatibility.

**Layer 2 — Data Link:**

- Verify MAC learning: `show ethernet-switching table` (for switching).
- Check ARP resolution: `show arp` — stale or missing entries indicate L2 issues.
- Look for duplex mismatch: `show interfaces ge-0/0/0 media` — auto-negotiate vs forced.
- LLDP neighbor verification: `show lldp neighbors` — confirm expected topology.

**Layer 3 — Network:**

- Verify IP config: `show interfaces terse` — correct IP/mask on correct unit.
- Check routing table: `show route 10.0.0.0/24` — is the route present? Correct next-hop?
- Verify forwarding table: `show route forwarding-table destination 10.0.0.0/24`.
- Ping with source: `ping 10.0.0.1 source 192.168.1.1` — ensures correct egress path.
- Traceroute with AS lookup: `traceroute 10.0.0.1 as-number-lookup` — find where traffic drops.

**Layer 4+ — Transport / Application:**

- Test TCP connectivity: `telnet 10.0.0.1 port 443` — verify port reachability.
- Check firewall filters: `show firewall log` and `show firewall filter <name> counter`.
- Verify NAT/ALG if applicable: `show security flow session`.

### 1.2 The Troubleshooting Checklist

1. **Define the problem.** What works? What doesn't? When did it start?
2. **Gather data.** Run show commands, check logs (`show log messages | last 200`).
3. **Isolate the layer.** Use the bottom-up approach above.
4. **Form a hypothesis.** Based on data, what is the most likely cause?
5. **Test the hypothesis.** Make one change at a time. Use `commit confirmed 5` for safety.
6. **Document the fix.** Add commit comments: `commit comment "fix: corrected OSPF area mismatch"`.

### 1.3 Key Daemon Awareness

Every Junos subsystem has a daemon. Knowing which daemon owns what helps target logs
and traceoptions.

| Daemon    | Responsibility                         | Log File              |
|-----------|----------------------------------------|-----------------------|
| `rpd`     | Routing protocols (OSPF, BGP, IS-IS)  | `show log rpd`        |
| `chassisd`| Chassis management, fans, temps, PSUs  | `show log chassisd`   |
| `mgd`     | Management (CLI, NETCONF, commit)      | `show log mgd`        |
| `pfed`    | Packet Forwarding Engine               | `show log pfed`       |
| `dcd`     | Interface configuration                | `show log dcd`        |
| `snmpd`   | SNMP agent                             | `show log snmpd`      |
| `alarmd`  | Alarm management                       | `show log alarmd`     |
| `eventd`  | Event processing and automation        | `show log eventd`     |
| `cosd`    | Class of Service                       | `show log cosd`       |

---

## 2. SNMP MIB Tree for Juniper

### 2.1 MIB Structure Overview

Juniper's enterprise MIB branch sits under:

```
iso.org.dod.internet.private.enterprises.juniperMIB
  1.3.6.1.4.1.2636
```

The tree subdivides into:

```
2636.1  — jnxProducts       (device model OIDs for sysObjectID)
2636.2  — jnxServices       (service-specific MIBs)
2636.3  — jnxMIBs           (main operational MIBs)
2636.4  — jnxTraps           (SNMP trap definitions)
2636.5  — jnxExperiment     (experimental/pre-standard)
```

### 2.2 Commonly Polled OIDs

| OID / MIB Object                          | Description                          |
|-------------------------------------------|--------------------------------------|
| `1.3.6.1.2.1.1.1.0` (sysDescr)           | System description string            |
| `1.3.6.1.2.1.1.3.0` (sysUpTime)          | Uptime in hundredths of seconds      |
| `1.3.6.1.2.1.1.5.0` (sysName)            | Hostname                             |
| `1.3.6.1.2.1.2.2.1.8` (ifOperStatus)     | Interface operational status (1=up)  |
| `1.3.6.1.2.1.2.2.1.10` (ifInOctets)      | Input bytes (32-bit counter)         |
| `1.3.6.1.2.1.2.2.1.16` (ifOutOctets)     | Output bytes (32-bit counter)        |
| `1.3.6.1.2.1.31.1.1.1.6` (ifHCInOctets)  | Input bytes (64-bit counter)         |
| `1.3.6.1.2.1.31.1.1.1.10` (ifHCOutOctets)| Output bytes (64-bit counter)        |
| `1.3.6.1.2.1.2.2.1.14` (ifInErrors)      | Input error count                    |
| `1.3.6.1.2.1.2.2.1.20` (ifOutErrors)     | Output error count                   |
| `1.3.6.1.4.1.2636.3.1.13.1.5` (jnxOperatingTemp) | Component temperature (C)    |
| `1.3.6.1.4.1.2636.3.1.13.1.6` (jnxOperatingCPU)  | Routing engine CPU util (%)  |
| `1.3.6.1.4.1.2636.3.1.13.1.11`(jnxOperatingMemory)| Memory utilization (%)       |
| `1.3.6.1.4.1.2636.3.1.13.1.7` (jnxOperatingState) | Component state (running=2)  |
| `1.3.6.1.4.1.2636.3.4.2.2.1`  (jnxYellowAlarmState)| Yellow alarm active         |
| `1.3.6.1.4.1.2636.3.4.2.3.1`  (jnxRedAlarmState)   | Red alarm active            |

### 2.3 SNMP Configuration on Junos

```bash
set snmp community public authorization read-only
set snmp community public clients 10.0.0.0/24       # restrict polling sources
set snmp trap-group TRAPS targets 10.0.0.100
set snmp trap-group TRAPS categories chassis link configuration
set snmp location "DC1-Row3-Rack7"
set snmp contact "noc@example.com"
```

### 2.4 Verifying SNMP

```bash
show snmp mib walk jnxOperatingCPU                   # walk CPU OID from CLI
show snmp statistics                                  # SNMP get/set/trap counts
show snmp mib get sysUpTime.0                        # single OID query
```

### 2.5 64-bit Counters

For 10G+ interfaces, always poll `ifHCInOctets` / `ifHCOutOctets` (64-bit from
IF-MIB). The 32-bit `ifInOctets` wraps every ~34 seconds at 10 Gbps, making it
useless for high-speed links. Configure your NMS to use HC (High Capacity) counters.

---

## 3. Traceoptions Deep Dive

Traceoptions is Junos's per-subsystem debug mechanism. Unlike Cisco's `debug`,
traceoptions writes to files (not the console), making it safer for production.

### 3.1 Anatomy of a Traceoptions Configuration

```bash
set protocols ospf traceoptions file ospf-trace size 10m files 5 world-readable
set protocols ospf traceoptions flag spf detail
set protocols ospf traceoptions flag hello detail
set protocols ospf traceoptions flag lsa-update detail
```

- **file**: Output filename under `/var/log/`. `size` caps each file. `files` sets rotation count.
- **world-readable**: Allows non-root users to read the trace file.
- **flag**: Which events to trace. `detail` increases verbosity.

### 3.2 Per-Protocol Traceoptions

**OSPF:**

| Flag           | What It Traces                                      |
|----------------|-----------------------------------------------------|
| `hello`        | Hello packet send/receive, neighbor discovery       |
| `spf`          | SPF calculations, path decisions                    |
| `lsa-update`   | LSA origination, flooding, aging                    |
| `lsa-ack`      | LSA acknowledgments                                 |
| `database`     | LSDB operations, database exchange                  |
| `general`      | Catch-all for non-specific events                   |
| `policy`        | Route policy evaluation for OSPF                    |
| `all`          | Everything (verbose — use cautiously)                |

**BGP:**

| Flag           | What It Traces                                      |
|----------------|-----------------------------------------------------|
| `open`         | OPEN message exchange, capability negotiation       |
| `update`       | UPDATE messages (routes received/sent)              |
| `keepalive`    | Keepalive exchange                                  |
| `notification` | NOTIFICATION messages (errors, session resets)      |
| `route`        | Route processing decisions                          |
| `policy`        | Policy evaluation on import/export                  |
| `all`          | Everything                                          |

**Routing general:**

```bash
set routing-options traceoptions file routing-trace size 5m files 3
set routing-options traceoptions flag route detail
set routing-options traceoptions flag task detail
set routing-options traceoptions flag timer detail
```

**Interface / DCD:**

```bash
set interfaces traceoptions file if-trace size 5m files 3
set interfaces traceoptions flag all
```

### 3.3 Reading Trace Output

```bash
show log ospf-trace                     # view the trace file
show log ospf-trace | last 100          # recent entries
show log ospf-trace | match "SPF"       # filter for SPF events
monitor start ospf-trace                # live tail (stream to terminal)
```

Trace output format:
```
Mar  5 14:23:01.234 OSPF HELLO sent to 224.0.0.5 via ge-0/0/0.0, area 0.0.0.0
Mar  5 14:23:01.456 OSPF HELLO rcvd from 10.0.0.2 via ge-0/0/0.0, area 0.0.0.0
Mar  5 14:23:05.789 OSPF SPF scheduled for area 0.0.0.0
```

### 3.4 Traceoptions Best Practices

1. **Always set file size and rotation.** Unbounded trace fills `/var/log` and crashes the box.
2. **Use specific flags, not `flag all`.** Especially on BGP with full tables — `flag update` on a full-table peer generates enormous output.
3. **Deactivate when done.** Use `deactivate protocols ospf traceoptions` rather than deleting — preserves config for next debug session.
4. **Use `match` on trace files.** Don't read raw output on busy routers; filter immediately.
5. **Monitor disk.** `show system storage` — watch `/var` usage while tracing.

---

## 4. Automation: Event-Scripts, Op-Scripts, Commit-Scripts

Junos supports on-box automation via three script types, all using SLAX (a
compact XSLT syntax) or XSLT. Python scripting is also supported on newer
releases.

### 4.1 Event Scripts

Event scripts trigger automatically when a specific syslog event occurs.

**Use case:** Auto-disable an interface when CRC errors exceed a threshold.

**Configuration:**

```bash
# Place script in /var/db/scripts/event/
set event-options policy CRC_SHUTDOWN {
    events SNMP_TRAP_LINK_DOWN;
    then {
        event-script crc-shutdown.slax;
    }
}

# Enable the script
set system scripts language slax
set event-options event-script file crc-shutdown.slax
```

**Example SLAX event script** (`crc-shutdown.slax`):

```slax
version 1.0;
ns junos = "http://xml.juniper.net/junos/*/junos";
ns xnm = "http://xml.juniper.net/xnm/1.1/xnm";
ns jcs = "http://xml.juniper.net/junos/commit-scripts/1.0";

match / {
    var $message = event-script-input/trigger-event/message;

    if (contains($message, "link down")) {
        var $interface = event-script-input/trigger-event/attribute-list/attribute[name == "interface-name"]/value;
        expr jcs:syslog("external.warning", "EVENT-SCRIPT: Link down on ", $interface);
    }
}
```

### 4.2 Op Scripts

Op (operational) scripts extend the CLI with custom commands. Invoked manually
with `op <script-name>`.

**Use case:** Custom health-check command that aggregates key show outputs.

**Configuration:**

```bash
# Place script in /var/db/scripts/op/
set system scripts op file health-check.slax
```

**Example SLAX op script** (`health-check.slax`):

```slax
version 1.0;
ns junos = "http://xml.juniper.net/junos/*/junos";
ns jcs = "http://xml.juniper.net/junos/commit-scripts/1.0";

match / {
    <op-script-results> {
        /* Get chassis alarms */
        var $alarms = jcs:invoke("get-alarm-information");
        <output> "=== Chassis Alarms ===";
        for-each ($alarms/alarm-detail) {
            <output> alarm-class _ ": " _ alarm-description;
        }

        /* Get RE status */
        var $re = jcs:invoke("get-route-engine-information");
        <output> "=== Routing Engine ===";
        <output> "CPU: " _ $re/route-engine/cpu-user _ "% user";
        <output> "Memory: " _ $re/route-engine/memory-buffer-utilization _ "% used";

        /* Get interface errors */
        var $ifs = jcs:invoke("get-interface-information");
        <output> "=== Interfaces with Errors ===";
        for-each ($ifs/physical-interface[input-error-count > 0]) {
            <output> name _ ": " _ input-error-count _ " input errors";
        }
    }
}
```

**Run it:**

```bash
op health-check
```

### 4.3 Commit Scripts

Commit scripts run automatically at commit time. They can enforce policies,
validate configuration, or auto-generate config.

**Use case:** Require that every interface has a description configured.

**Configuration:**

```bash
# Place script in /var/db/scripts/commit/
set system scripts commit file require-description.slax
```

**Example SLAX commit script** (`require-description.slax`):

```slax
version 1.0;
ns junos = "http://xml.juniper.net/junos/*/junos";
ns jcs = "http://xml.juniper.net/junos/commit-scripts/1.0";

match configuration {
    for-each (interfaces/interface[not(description)]) {
        if (not(starts-with(name, "lo")) && not(starts-with(name, "fxp"))) {
            <xnm:warning> {
                <message> "Interface " _ name _ " has no description configured.";
            }
        }
    }
}
```

This emits a warning (not error) at commit time for any interface missing a
description, excluding loopback and management interfaces.

### 4.4 Script Locations Summary

| Script Type   | Directory                      | Trigger              |
|---------------|--------------------------------|----------------------|
| Event scripts | `/var/db/scripts/event/`       | Syslog event match   |
| Op scripts    | `/var/db/scripts/op/`          | Manual `op` command  |
| Commit scripts| `/var/db/scripts/commit/`      | Every `commit`       |

---

## 5. Junos RPCs (XML API) for Programmatic Monitoring

Every Junos `show` command has an equivalent XML RPC. This is the foundation
of NETCONF-based automation and programmatic monitoring.

### 5.1 Discovering the RPC for Any Show Command

```bash
show interfaces terse | display xml rpc
```

Output:

```xml
<rpc>
  <get-interface-information>
    <terse/>
  </get-interface-information>
</rpc>
```

This tells you the exact RPC element name (`get-interface-information`) and
parameters (`<terse/>`).

### 5.2 Common Monitoring RPCs

| Show Command                        | RPC Element                              |
|-------------------------------------|------------------------------------------|
| `show interfaces terse`             | `<get-interface-information><terse/>`    |
| `show interfaces extensive`         | `<get-interface-information><extensive/>`|
| `show route`                        | `<get-route-information>`                |
| `show route table inet.0`           | `<get-route-information><table>inet.0</table>` |
| `show bgp summary`                  | `<get-bgp-summary-information>`          |
| `show bgp neighbor`                 | `<get-bgp-neighbor-information>`         |
| `show ospf neighbor`                | `<get-ospf-neighbor-information>`        |
| `show ospf database`                | `<get-ospf-database-information>`        |
| `show chassis hardware`             | `<get-chassis-inventory>`                |
| `show chassis environment`          | `<get-environment-information>`          |
| `show chassis alarms`               | `<get-alarm-information>`                |
| `show chassis routing-engine`       | `<get-route-engine-information>`         |
| `show system uptime`                | `<get-system-uptime-information>`        |
| `show system storage`               | `<get-system-storage>`                   |
| `show system processes extensive`   | `<get-system-processes-information><extensive/>` |
| `show arp`                          | `<get-arp-table-information>`            |
| `show lldp neighbors`               | `<get-lldp-neighbors-information>`       |
| `show log messages`                 | `<get-log><filename>messages</filename>` |

### 5.3 Using RPCs via NETCONF

NETCONF is the standard transport for Junos RPCs. Session flow:

```
Client                           Junos Device
  |--- TCP/SSH (port 830) -------->|
  |<-- <hello> (capabilities) -----|
  |--- <hello> (capabilities) ---->|
  |                                |
  |--- <rpc>                       |
  |     <get-interface-information>|
  |       <terse/>                 |
  |     </get-interface-information>
  |   </rpc> --------------------->|
  |                                |
  |<-- <rpc-reply>                 |
  |     <interface-information>    |
  |       ...XML data...           |
  |     </interface-information>   |
  |   </rpc-reply> ----------------|
  |                                |
  |--- <close-session/> ---------->|
```

**Enable NETCONF on the device:**

```bash
set system services netconf ssh port 830
```

### 5.4 Python Example: Polling Interface Counters via RPCs

Using the `junos-eznc` (PyEZ) library:

```python
from jnpr.junos import Device
from jnpr.junos.op.ethport import EthPortTable

dev = Device(host="10.0.0.1", user="admin", password="secret123")
dev.open()

# Method 1: Using structured tables
ports = EthPortTable(dev)
ports.get()
for port in ports:
    print(f"{port.name}: oper={port.oper}, in_bytes={port.rx_bytes}, out_bytes={port.tx_bytes}")

# Method 2: Raw RPC call
rpc_reply = dev.rpc.get_interface_information(interface_name="ge-0/0/0", extensive=True)
in_errors = rpc_reply.findtext(".//input-errors")
out_errors = rpc_reply.findtext(".//output-errors")
crc_errors = rpc_reply.findtext(".//input-crc-errors")
print(f"ge-0/0/0: in_errors={in_errors}, out_errors={out_errors}, crc={crc_errors}")

# Method 3: Get chassis alarms
alarms = dev.rpc.get_alarm_information()
for alarm in alarms.findall("alarm-detail"):
    print(f"ALARM: {alarm.findtext('alarm-class')} - {alarm.findtext('alarm-description')}")

dev.close()
```

### 5.5 RPC Output Formats

From the CLI, you can view any command's output in multiple formats:

```bash
show interfaces terse | display xml       # raw XML
show interfaces terse | display json       # JSON (Junos 14.2+)
show interfaces terse | display xml rpc    # the RPC request itself
```

This makes it straightforward to build monitoring scripts: discover the RPC
from the CLI, then call it programmatically via NETCONF or PyEZ.

### 5.6 REST API (Junos 21.1+)

Modern Junos versions also expose a REST API:

```bash
# Enable REST API
set system services rest http port 8080
set system services rest enable-explorer   # Swagger-like API browser

# Query from curl
curl -u admin:password http://10.0.0.1:8080/rpc/get-interface-information \
  -H "Content-Type: application/xml" \
  -d "<get-interface-information><terse/></get-interface-information>"
```

This provides an HTTP-native alternative to NETCONF for monitoring integrations.
