# NETCONF (Network Configuration Protocol)

RFC 6241 protocol for network device configuration over SSH — XML-encoded RPCs with datastore model, candidate config, locking, validated commit, subtree/XPath filtering, and change notifications.

## Protocol Layers

### NETCONF stack

```bash
# Layer 4 — Content     : Configuration and state data (YANG-modeled)
# Layer 3 — Operations  : get, get-config, edit-config, copy-config, etc.
# Layer 2 — Messages    : <rpc>, <rpc-reply>, <notification>
# Layer 1 — Transport   : SSH (mandatory), TLS (optional), SOAP (deprecated)
```

## NETCONF Capabilities and Hello

### Hello exchange

```xml
<!-- Server hello (received on connect) -->
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:candidate:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:confirmed-commit:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:validate:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:rollback-on-error:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:xpath:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:writable-running:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:notification:1.0</capability>
  </capabilities>
  <session-id>12345</session-id>
</hello>
```

### Common capabilities

```bash
# :candidate        — device supports candidate datastore
# :confirmed-commit — commit with rollback timer
# :validate         — validate config before apply
# :rollback-on-error — revert on partial failure
# :xpath            — XPath filtering support
# :writable-running — direct edit of running config (no candidate needed)
# :startup          — separate startup config datastore
# :notification     — async event notifications
# :with-defaults    — control default value reporting
```

## NETCONF Operations

### get-config — retrieve configuration

```python
from ncclient import manager

with manager.connect(
    host='10.0.0.1',
    port=830,
    username='admin',
    password='cisco123',
    hostkey_verify=False,
    device_params={'name': 'default'}
) as m:
    # Get entire running config
    config = m.get_config(source='running')
    print(config)

    # Get candidate config
    config = m.get_config(source='candidate')

    # Get startup config
    config = m.get_config(source='startup')
```

### get-config with subtree filter

```python
# Subtree filter — select specific config branches
filter_xml = '''
<filter type="subtree">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
    <interface>
      <name>GigabitEthernet0/0/0</name>
    </interface>
  </interfaces>
</filter>
'''
config = m.get_config(source='running', filter=filter_xml)

# Filter for BGP config (IOS-XE native model)
bgp_filter = '''
<filter type="subtree">
  <native xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-native">
    <router>
      <bgp xmlns="http://cisco.com/ns/yang/Cisco-IOS-XE-bgp"/>
    </router>
  </native>
</filter>
'''
config = m.get_config(source='running', filter=bgp_filter)
```

### get-config with XPath filter

```python
# XPath filter — more powerful selection (requires :xpath capability)
xpath_filter = '''
<filter type="xpath"
  select="/interfaces/interface[name='GigabitEthernet0/0/0']"
  xmlns:if="urn:ietf:params:xml:ns:yang:ietf-interfaces"/>
'''
config = m.get_config(source='running', filter=xpath_filter)

# XPath with predicate
xpath_filter = '''
<filter type="xpath"
  select="/interfaces/interface[enabled='true']"/>
'''
config = m.get_config(source='running', filter=xpath_filter)
```

### get — retrieve config and state

```python
# get returns both config AND operational state
filter_xml = '''
<filter type="subtree">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces"/>
</filter>
'''
state = m.get(filter=filter_xml)
# Includes: admin-status, oper-status, counters, etc.
```

### edit-config — modify configuration

```python
# Edit running config directly (requires :writable-running)
config_xml = '''
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
    <interface>
      <name>Loopback99</name>
      <type xmlns:ianaift="urn:ietf:params:xml:ns:yang:iana-if-type">
        ianaift:softwareLoopback
      </type>
      <enabled>true</enabled>
      <description>Test loopback</description>
    </interface>
  </interfaces>
</config>
'''
m.edit_config(target='running', config=config_xml)

# Edit with default operation
m.edit_config(
    target='running',
    config=config_xml,
    default_operation='merge'          # merge (default), replace, none
)
```

### edit-config operations (per-element)

```python
# Delete a specific element
delete_xml = '''
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
    <interface operation="delete">
      <name>Loopback99</name>
    </interface>
  </interfaces>
</config>
'''
m.edit_config(target='running', config=delete_xml)

# Replace entire subtree
replace_xml = '''
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces"
              operation="replace">
    <interface>
      <name>Loopback0</name>
      <enabled>true</enabled>
    </interface>
  </interfaces>
</config>
'''
m.edit_config(target='running', config=replace_xml)

# Create (fail if exists)
create_xml = '''
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
    <interface operation="create">
      <name>Loopback100</name>
      <enabled>true</enabled>
    </interface>
  </interfaces>
</config>
'''
m.edit_config(target='running', config=create_xml)

# Remove (silent if not exists)
remove_xml = '''
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
    <interface operation="remove">
      <name>Loopback100</name>
    </interface>
  </interfaces>
</config>
'''
m.edit_config(target='running', config=remove_xml)
```

### Candidate datastore workflow

```python
# 1. Lock candidate
m.lock(target='candidate')

# 2. Edit candidate
m.edit_config(target='candidate', config=config_xml)

# 3. Validate (optional)
m.validate(source='candidate')

# 4. Commit candidate → running
m.commit()

# 5. Unlock candidate
m.unlock(target='candidate')

# Discard changes (rollback candidate to running)
m.discard_changes()
```

### Confirmed commit

```python
# Commit with rollback timer (seconds)
m.commit(confirmed=True, timeout=300)         # auto-rollback in 5 min
# ... verify changes work ...
m.commit()                                    # confirm (cancel rollback timer)

# If confirm never comes → device auto-reverts after timeout
```

### copy-config

```python
# Copy running to startup
m.copy_config(source='running', target='startup')

# Copy from URL
m.copy_config(
    source='https://tftp.lab.local/config.xml',
    target='running'
)
```

### delete-config

```python
# Delete startup config
m.delete_config(target='startup')
# Cannot delete running — only candidate and startup
```

### lock / unlock

```python
# Lock running datastore (prevent other sessions from editing)
m.lock(target='running')
# ... make changes ...
m.unlock(target='running')

# Lock candidate
m.lock(target='candidate')
m.unlock(target='candidate')
```

## NETCONF Notifications

### Subscribe to notifications

```python
from ncclient import manager

with manager.connect(
    host='10.0.0.1', port=830,
    username='admin', password='cisco123',
    hostkey_verify=False
) as m:
    # Subscribe to all notifications
    m.create_subscription()

    # Subscribe to specific stream
    m.create_subscription(stream_name='NETCONF')

    # Subscribe with filter
    m.create_subscription(
        filter_type='subtree',
        filter_xml='<interface-event/>'
    )

    # Receive notifications
    while True:
        notification = m.take_notification(timeout=60)
        if notification:
            print(notification.notification_xml)
```

## NETCONF Call-Home

### Server-initiated connection (RFC 8071)

```bash
# IOS-XE — configure NETCONF call-home
netconf-yang ssh
call-home
 profile callhome-profile
  active
  destination transport-method ssh
  destination address ipv4 10.0.1.100 port 4334
!
```

```python
# Python — accept call-home connections
from ncclient import manager

# Listen for incoming NETCONF call-home
m = manager.connect(
    host='0.0.0.0',
    port=4334,
    username='admin',
    password='cisco123',
    hostkey_verify=False,
    call_home=True,
    timeout=120
)
config = m.get_config(source='running')
```

## Device Configuration

### IOS-XE NETCONF

```bash
netconf-yang                                  # enable NETCONF
netconf-yang ssh port 830                     # set SSH subsystem port
ip ssh version 2                              # require SSHv2

# Verify
show netconf-yang sessions                    # active sessions
show netconf-yang statistics                  # operation counters
show platform software yang-management process
```

### IOS-XR NETCONF

```bash
ssh server netconf port 830                   # enable NETCONF over SSH
ssh server netconf vrf default                # in default VRF
netconf agent tty                             # enable TTY agent
netconf-yang agent ssh                        # enable YANG agent

# Verify
show netconf-yang clients
show netconf-yang statistics
```

### JunOS NETCONF

```bash
set system services netconf ssh port 830      # enable NETCONF
set system services netconf rfc-compliant     # RFC 6241 mode
set system services netconf yang-compliant    # YANG support

# Verify
show system connections | match 830
show system netconf statistics
```

### NX-OS NETCONF

```bash
feature netconf                               # enable NETCONF
# NX-OS uses port 830 by default

# Verify
show netconf-yang sessions
```

## Raw SSH NETCONF

### Manual NETCONF session

```bash
# Connect via SSH subsystem
ssh -p 830 admin@10.0.0.1 -s netconf

# After hello exchange, send RPC:
<?xml version="1.0" encoding="UTF-8"?>
<rpc message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
  </get-config>
</rpc>
]]>]]>

# Close session
<rpc message-id="2" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <close-session/>
</rpc>
]]>]]>
```

### Message framing

```bash
# NETCONF 1.0 — end-of-message marker
]]>]]>

# NETCONF 1.1 — chunked framing (RFC 6242)
#N\n            (N = chunk size in bytes)
...data...
##\n            (end of message)

# Example:
#100
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="1">
  <get-config><source><running/></source></get-config>
</rpc>
##
```

## ncclient Advanced Usage

### Connect with device-specific handlers

```python
from ncclient import manager

# IOS-XE
m = manager.connect(
    host='10.0.0.1', port=830,
    username='admin', password='cisco123',
    hostkey_verify=False,
    device_params={'name': 'csr'}              # CSR1000v handler
)

# IOS-XR
m = manager.connect(
    host='10.0.0.2', port=830,
    username='admin', password='cisco123',
    hostkey_verify=False,
    device_params={'name': 'iosxr'}
)

# Junos
m = manager.connect(
    host='10.0.0.3', port=830,
    username='admin', password='cisco123',
    hostkey_verify=False,
    device_params={'name': 'junos'}
)

# NX-OS
m = manager.connect(
    host='10.0.0.4', port=830,
    username='admin', password='cisco123',
    hostkey_verify=False,
    device_params={'name': 'nexus'}
)
```

### Dispatch raw RPC

```python
# Send arbitrary XML RPC
rpc_xml = '''
<get-schema xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring">
  <identifier>ietf-interfaces</identifier>
  <version>2018-02-20</version>
  <format>yang</format>
</get-schema>
'''
reply = m.dispatch(to_ele(rpc_xml))

# List available YANG schemas
schemas_filter = '''
<filter type="subtree">
  <netconf-state xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring">
    <schemas/>
  </netconf-state>
</filter>
'''
schemas = m.get(filter=schemas_filter)
```

### Error handling

```python
from ncclient.operations import RPCError

try:
    m.edit_config(target='running', config=config_xml)
except RPCError as e:
    print(f"Error type: {e.type}")            # protocol, application, etc.
    print(f"Error tag: {e.tag}")              # data-exists, invalid-value, etc.
    print(f"Error severity: {e.severity}")    # error, warning
    print(f"Error message: {e.message}")
    print(f"Error path: {e.path}")            # XPath to error location
```

## See Also

- restconf
- yang-models
- gnmi-gnoi
- pyats

## References

- RFC 6241 — NETCONF Protocol: https://datatracker.ietf.org/doc/html/rfc6241
- RFC 6242 — NETCONF over SSH: https://datatracker.ietf.org/doc/html/rfc6242
- RFC 5277 — NETCONF Event Notifications: https://datatracker.ietf.org/doc/html/rfc5277
- RFC 8071 — NETCONF Call Home: https://datatracker.ietf.org/doc/html/rfc8071
- ncclient documentation: https://ncclient.readthedocs.io/
