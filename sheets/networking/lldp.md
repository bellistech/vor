# LLDP (Link Layer Discovery Protocol)

IEEE 802.1AB Layer 2 protocol that allows network devices to advertise their identity, capabilities, and neighbors using multicast Ethernet frames (EtherType 0x88CC) sent to the well-known destination 01:80:C2:00:00:0E.

## LLDPDU Frame Format

```
Ethernet Frame
+-----------------+-------------------+------------+---------+-----+
| Dst MAC         | Src MAC           | EtherType  | LLDPDU  | FCS |
| 01:80:C2:00:00:0E | (sender's MAC) | 0x88CC     | TLVs... |     |
+-----------------+-------------------+------------+---------+-----+

LLDPDU = sequence of TLVs, terminated by End of LLDPDU TLV (type 0, length 0)
```

## TLV Structure (Type-Length-Value)

```
 0                   1                   2
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 ...
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-----+
|  Type (7 bits)  | Length (9 bits)  |     Value ...    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-----+

Type:   0-127 (7 bits) — identifies the TLV kind
Length: 0-511 (9 bits) — length of the Value field in octets
Value:  variable — depends on Type
```

## Mandatory TLVs

```
Type  Name                  Description
────────────────────────────────────────────────────────────
0     End of LLDPDU         Marks the end of the LLDPDU (length 0)
1     Chassis ID            Identifies the sending device (MAC, IP, hostname, etc.)
2     Port ID               Identifies the sending port (ifName, MAC, local ID)
3     Time to Live (TTL)    Seconds the receiving device should hold this info
                            (0 = request to delete, max 65535)

# Order matters: Chassis ID must be first, Port ID second, TTL third
# End of LLDPDU must always be last
```

## Optional TLVs

```
Type  Name                    Description
────────────────────────────────────────────────────────────
4     Port Description        ifDescr of the sending port
5     System Name             sysName (hostname/FQDN)
6     System Description      sysDescr (OS, hardware, version)
7     System Capabilities     Bridge, router, telephone, repeater, etc.
                              Two-byte bitmap: supported + enabled
8     Management Address      IP or other address for SNMP management
```

### System Capabilities Bitmap

```
Bit   Capability
──────────────────────────
0     Other
1     Repeater
2     MAC Bridge
3     WLAN Access Point
4     Router
5     Telephone
6     DOCSIS Cable Device
7     Station Only
```

## Organizationally-Specific TLVs (Type 127)

```
Type 127 TLV:
+--------+--------+-----------+
| OUI    | Subtype | Info      |
| 3 bytes| 1 byte  | variable  |
+--------+--------+-----------+

Common OUIs:
  00:80:C2  — IEEE 802.1 (VLAN name, port VLAN ID, link aggregation)
  00:12:0F  — IEEE 802.3 (MAC/PHY config, power via MDI, link agg,
              max frame size)
  TIA       — LLDP-MED (media endpoint discovery)
```

## LLDP-MED (Media Endpoint Discovery)

```
# Extension for VoIP phones, video endpoints, PoE
# Defined by TIA-1057

LLDP-MED TLV types (OUI: 00:12:BB):
  1  LLDP-MED Capabilities        — what the endpoint supports
  2  Network Policy                — VLAN ID, L2/L3 priority, DSCP for VoIP
  3  Location Identification       — civic address, ELIN, GPS coordinates
  4  Extended Power-via-MDI        — PoE power type, source, priority, value

# VoIP VLAN assignment flow:
# 1. IP phone boots, sends untagged LLDP
# 2. Switch responds with LLDP-MED Network Policy TLV
#    containing voice VLAN ID and QoS markings
# 3. Phone reconfigures to use the voice VLAN
# 4. DHCP and SIP registration proceed on the voice VLAN
```

## CDP (Cisco Discovery Protocol) Comparison

```
Feature              LLDP (802.1AB)       CDP (Cisco)
──────────────────────────────────────────────────────────
Standard             IEEE open standard   Cisco proprietary
Layer                Layer 2              Layer 2
EtherType            0x88CC               SNAP (0x2000)
Multicast Dst        01:80:C2:00:00:0E    01:00:0C:CC:CC:CC
Default Timer        30s                  60s
Default TTL          120s                 180s
VoIP VLAN            LLDP-MED             CDP native
PoE negotiation      LLDP-MED + 802.3at   CDP
Extensibility        Org-specific TLVs    Limited
Multi-vendor         Yes                  No (Cisco only)
```

## lldpd / lldpctl Commands

```bash
# lldpd is the open-source LLDP daemon (also speaks CDP, EDP, SONMP)
# lldpctl is the CLI client for lldpd

# Install
apt install lldpd                         # Debian/Ubuntu
yum install lldpd                         # RHEL/CentOS

# Start the daemon
systemctl enable --now lldpd

# Show discovered neighbors
lldpctl                                   # all interfaces
lldpctl eth0                              # specific interface

# Show neighbors in different formats
lldpctl -f json                           # JSON output
lldpctl -f keyvalue                       # key=value pairs
lldpctl -f xml                            # XML output

# Show local chassis information
lldpctl show chassis

# Configure system name
lldpcli configure system hostname "switch-01"

# Configure management address
lldpcli configure system ip management pattern 192.168.1.*

# Enable/disable CDP interoperability
lldpcli configure lldp agent-type nearest-bridge
lldpcli configure cdp enable

# Set transmit interval (seconds)
lldpcli configure lldp tx-interval 30

# Set transmit hold multiplier (TTL = tx-interval * tx-hold)
lldpcli configure lldp tx-hold 4

# Disable LLDP on an interface
lldpcli configure ports eth1 lldp status disabled

# Show running configuration
lldpcli show running-configuration

# Show statistics
lldpcli show statistics
```

## lldpad (Intel LLDP Agent)

```bash
# lldpad is an alternative LLDP daemon focused on DCB (Data Center Bridging)
# Used primarily for FCoE and DCB negotiation

# Install
apt install lldpad

# Start
systemctl enable --now lldpad

# Query LLDP neighbors using lldptool
lldptool -t -i eth0 -V sysName           # get system name TLV
lldptool -t -i eth0 -V portDesc          # get port description TLV
lldptool -t -i eth0 -V mngAddr           # get management address

# Get all neighbor TLVs
lldptool get-tlv -n -i eth0

# Set local TLVs
lldptool set-lldp -i eth0 adminStatus=rxtx
lldptool set-tlv -i eth0 -V sysName enableTx=yes
```

## Capturing LLDP with tcpdump

```bash
# Capture LLDP frames on an interface
tcpdump -i eth0 -nn ether proto 0x88cc

# Capture and decode LLDP with full detail
tcpdump -i eth0 -nn -vv ether proto 0x88cc

# Capture LLDP to a file for analysis
tcpdump -i eth0 -nn -w /tmp/lldp.pcap ether proto 0x88cc

# Filter by LLDP multicast destination
tcpdump -i eth0 -nn ether dst 01:80:c2:00:00:0e

# Capture both LLDP and CDP
tcpdump -i eth0 -nn '(ether proto 0x88cc) or (ether dst 01:00:0c:cc:cc:cc)'

# Read LLDP from pcap file
tcpdump -nn -vv -r /tmp/lldp.pcap

# tshark alternative with field extraction
tshark -i eth0 -f "ether proto 0x88cc" -T fields \
  -e lldp.chassis.id -e lldp.port.id -e lldp.tlv.system.name
```

## Use Cases

```
Topology Discovery
  Switches and routers exchange neighbor info to build topology maps.
  NMS tools (LibreNMS, Zabbix, NetBox) poll LLDP data via SNMP
  (LLDP-MIB: IEEE8021-LLDP-MIB) to auto-discover network graphs.

VoIP VLAN Assignment
  LLDP-MED Network Policy TLV pushes voice VLAN and QoS settings
  to IP phones at link-up, eliminating manual phone configuration.

PoE Negotiation
  LLDP-MED Extended Power TLV and IEEE 802.3at/bt TLVs negotiate
  power requirements between powered devices and switches.

Cable Diagnostics
  Port Description and System Name TLVs document what is connected
  to each switch port, acting as a live cable label database.

Data Center Bridging (DCB)
  lldpad exchanges DCBx TLVs for priority flow control (PFC),
  enhanced transmission selection (ETS), and FCoE setup.
```

## Tips

- LLDP frames are never forwarded by bridges. They are consumed by the directly connected neighbor only. This is by design -- each link has its own independent LLDP exchange.
- The default TTL is tx-interval (30s) multiplied by tx-hold (4), giving 120 seconds. If a neighbor stops sending, its entry is purged after 120s. Reduce tx-interval for faster failure detection.
- LLDP is unidirectional. Each device transmits its own information independently. There is no request/response handshake and no acknowledgment mechanism.
- On servers, enable LLDP to help network admins identify which switch port a server is patched to. The `lldpd` daemon is lightweight and requires no configuration for basic operation.
- When migrating from CDP to LLDP, run both protocols simultaneously during the transition. Most modern Cisco switches support both. Configure `lldp run` globally on Cisco IOS.
- LLDP-MED Network Policy TLVs carry DSCP and 802.1p values. Ensure the switch port is configured as a trunk or voice VLAN-capable port for the phone to tag traffic correctly.
- LLDP has no authentication. Any device can send spoofed LLDP frames claiming to be a different switch or system. Use 802.1X port authentication as a complementary control.
- Maximum LLDPDU size is limited by the Ethernet MTU. With a 1500-byte MTU and 14-byte header + 4-byte FCS, the LLDPDU payload maximum is approximately 1482 bytes.
- On Linux, verify LLDP is reaching the host with `tcpdump -i eth0 -nn ether proto 0x88cc -c 1`. If nothing appears, the NIC or hypervisor may be filtering the LLDP multicast address.
- LLDP-MED location data (civic address, GPS) is critical for E911 in enterprise VoIP deployments. The switch pushes location to the phone, which passes it to the call control system.

## See Also

- ethernet, vlan, stp

## References

- [IEEE 802.1AB-2016 -- Station and Media Access Control Connectivity Discovery](https://standards.ieee.org/standard/802_1AB-2016.html)
- [TIA-1057 -- LLDP for Media Endpoint Devices (LLDP-MED)](https://tiaonline.org/what-we-do/standards/)
- [RFC 2922 -- Physical Topology MIB](https://www.rfc-editor.org/rfc/rfc2922)
- [lldpd -- open-source LLDP implementation](https://lldpd.github.io/)
- [IEEE 802.3at -- PoE+ (Power over Ethernet)](https://standards.ieee.org/standard/802_3at-2009.html)
- [man lldpctl](https://lldpd.github.io/usage.html)
- [man lldptool](https://linux.die.net/man/8/lldptool)
