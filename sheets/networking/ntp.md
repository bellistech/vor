# NTP (Network Time Protocol)

Protocol for synchronizing clocks across networked systems to UTC using a hierarchical stratum architecture, operating over UDP port 123 with millisecond-level accuracy over the internet and sub-microsecond on LANs.

## NTP Stratum Architecture

```
Stratum 0 — Reference clocks (GPS, atomic, radio)
     |       Not network devices; directly attached hardware
     v
Stratum 1 — Primary servers (directly connected to stratum 0)
     |       time.nist.gov, time.google.com, ntp.ubuntu.com
     v
Stratum 2 — Synchronized to stratum 1 servers
     |       pool.ntp.org servers, enterprise NTP servers
     v
Stratum 3 — Synchronized to stratum 2
     |       Department/office servers
     v
   ...
Stratum 15 — Maximum valid stratum
Stratum 16 — Unsynchronized (special value)
```

## chrony (Modern NTP — Preferred)

```bash
# Install
sudo apt install chrony        # Debian/Ubuntu
sudo dnf install chrony        # Fedora/RHEL

# Main config: /etc/chrony/chrony.conf (Debian) or /etc/chrony.conf (RHEL)
server time.google.com iburst
server time.cloudflare.com iburst
pool pool.ntp.org iburst maxsources 4

# Allow LAN clients to sync
allow 10.0.0.0/8
allow 192.168.0.0/16

# Serve time even when not synchronized
local stratum 10

# Drift file
driftfile /var/lib/chrony/drift

# RTC synchronization
rtcsync

# Step clock if offset > 1 sec in first 3 updates
makestep 1.0 3

# Log measurements and statistics
log measurements statistics tracking

# Restart
sudo systemctl restart chronyd
```

## chronyc Commands

```bash
# Check synchronization status
chronyc tracking
# Reference ID    : A1B2C3D4 (time.google.com)
# Stratum         : 2
# System time     : 0.000012345 seconds fast of NTP time
# Last offset     : +0.000003421 seconds
# RMS offset      : 0.000015623 seconds
# Root delay      : 0.025431 seconds

# Show sources and their status
chronyc sources -v
# ^* time.google.com    2   6    17    25   +123us[ +156us] +/-  13ms
# ^+ time.cloudflare.com 2  6    17    28    -45us[  -12us] +/-  15ms
# Symbols: * = current best, + = acceptable, - = excluded, ? = unreachable

# Source statistics (jitter, offset, freq)
chronyc sourcestats -v

# Force immediate sync
chronyc makestep

# Add a server at runtime
chronyc add server ntp.example.com iburst

# Check if chrony is synchronized
chronyc waitsync 30 0.1
# Waits up to 30 sec for sync within 0.1 sec

# Show NTP clients (if acting as server)
chronyc clients

# Show activity summary
chronyc activity
```

## ntpd (Classic NTP Daemon)

```bash
# Install
sudo apt install ntp           # Debian/Ubuntu

# Config: /etc/ntp.conf
server 0.pool.ntp.org iburst
server 1.pool.ntp.org iburst
server 2.pool.ntp.org iburst
server 3.pool.ntp.org iburst

# Restrict access
restrict default kod nomodify notrap nopeer noquery
restrict 127.0.0.1
restrict ::1
restrict 10.0.0.0 mask 255.0.0.0 nomodify notrap

# Drift file
driftfile /var/lib/ntp/drift

# Log
statsdir /var/log/ntpstats/
statistics loopstats peerstats clockstats

# Restart
sudo systemctl restart ntpd
```

## ntpq Commands

```bash
# Interactive peer list
ntpq -p
#      remote           refid      st t when poll reach   delay   offset  jitter
# *time.google.com .GOOG.           1 u   34   64  377   25.123   -0.432  1.234
# +time.nist.gov   .NIST.           1 u   42   64  377   45.678    0.567  2.345
# Tally: * = sys.peer, + = candidate, - = outlier, x = falseticker

# Peer details
ntpq -c "rv 0"

# Kernel time variables
ntpq -c kerninfo

# Association list
ntpq -c associations

# Check if synchronized
ntpstat
```

## systemd-timesyncd (Lightweight SNTP)

```bash
# Status
timedatectl status
timedatectl timesync-status

# Config: /etc/systemd/timesyncd.conf
# [Time]
# NTP=time.google.com time.cloudflare.com
# FallbackNTP=0.pool.ntp.org 1.pool.ntp.org

# Enable and start
sudo systemctl enable --now systemd-timesyncd

# Set timezone
timedatectl set-timezone America/New_York
timedatectl list-timezones

# Enable NTP
timedatectl set-ntp true
```

## NTP Diagnostics

```bash
# One-shot query (don't set clock)
ntpdate -q time.google.com
chronyd -Q 'server time.google.com iburst'

# Measure offset to a specific server
ntpdate -d time.google.com 2>&1 | grep "offset"

# Check NTP port reachability
nc -zuv time.google.com 123

# tcpdump NTP traffic
sudo tcpdump -i eth0 port 123 -vv

# Check system clock vs hardware clock
hwclock --show
date -u

# Measure clock drift manually
adjtimex --print | grep "tick"
```

## pool.ntp.org Configuration

```bash
# Use continental pool zones for better accuracy
# /etc/chrony/chrony.conf or /etc/ntp.conf
server 0.us.pool.ntp.org iburst      # North America
server 0.europe.pool.ntp.org iburst  # Europe
server 0.asia.pool.ntp.org iburst    # Asia

# Vendor pools
server time.google.com iburst        # Google (smeared leap seconds)
server time.cloudflare.com iburst    # Cloudflare
server time.apple.com iburst         # Apple
server time.windows.com iburst       # Microsoft
server time.facebook.com iburst      # Meta

# Google vs standard leap second handling
# Google: "leap smear" — spreads leap second over 24 hours
# Standard NTP: inserts/deletes 1 second at midnight UTC
# NEVER mix Google time with non-Google in the same pool
```

## NTP vs PTP Comparison

```
Feature          NTP                    PTP (IEEE 1588)
Accuracy         1-50 ms (internet)     <1 us (LAN)
                 <1 ms (LAN)            <100 ns (HW timestamping)
Transport        UDP/123                UDP/319,320 or L2
Architecture     Client-server          Master-slave (grandmaster)
Hardware         Software only          HW timestamping NICs
Use case         General purpose        Financial, telecom, 5G
Linux daemon     chrony/ntpd            ptp4l/phc2sys (linuxptp)
```

## Leap Second Handling

```bash
# Check upcoming leap seconds
ntpq -c "rv 0 leap"

# Chrony leap second status
chronyc tracking | grep "Leap status"

# Kernel leap second state
adjtimex --print | grep status

# Leap second file (for ntpd)
# Download from IERS
wget https://hpiers.obspm.fr/iers/bul/bulc/ntp/leap-seconds.list
# Add to ntp.conf: leapfile /etc/ntp/leap-seconds.list

# Google leap smear vs step
# Step: 23:59:59 -> 23:59:60 -> 00:00:00
# Smear: gradually adjusts over 24 hours (noon to noon UTC)
```

## Securing NTP

```bash
# NTP authentication with symmetric keys
# /etc/chrony/chrony.keys
1 SHA256 HEX:a1b2c3d4e5f6...

# chrony.conf
keyfile /etc/chrony/chrony.keys
server time.example.com key 1

# NTS (Network Time Security) — TLS-based authentication
# chrony >= 4.0
server time.cloudflare.com iburst nts

# Verify NTS
chronyc -N authdata
# Name/IP address       Mode KeyID Type KLen Last Atmp  NAK Cook CLen
# time.cloudflare.com    NTS     1    15  256    3    0    0    8  100

# Firewall: allow only outbound NTP
iptables -A OUTPUT -p udp --dport 123 -j ACCEPT
iptables -A INPUT -p udp --sport 123 -m state --state ESTABLISHED -j ACCEPT
```

## Tips

- Use chrony over ntpd for modern systems; it syncs faster, handles intermittent connections, and supports NTS
- Never mix Google time servers with non-Google servers; Google uses leap smearing while others step
- Set `iburst` on all server lines; it sends 4 quick packets at startup for faster initial sync
- Use `makestep 1.0 3` in chrony to allow stepping the clock if offset exceeds 1 second (first 3 updates only)
- Run `chronyc sources -v` to verify synchronization; look for `*` next to the selected source
- For VMs, disable hypervisor time sync (e.g., VMware Tools) and use NTP exclusively to avoid conflicts
- Configure at least 4 NTP sources; the intersection algorithm needs 3+ to detect a falseticker
- Enable NTS (Network Time Security) where available; `time.cloudflare.com` supports it since chrony 4.0
- Monitor clock offset with `chronyc tracking`; sustained offset > 100ms usually indicates network issues
- Use PTP (linuxptp) instead of NTP when you need sub-microsecond accuracy on a LAN
- Set `rtcsync` in chrony to periodically sync the hardware clock, important for correct time at boot
- Log NTP statistics for trend analysis; sudden jitter increases can indicate network path changes

## See Also

- ptp, timedatectl, hwclock, chrony, systemd-timesyncd

## References

- [RFC 5905 — NTPv4](https://datatracker.ietf.org/doc/html/rfc5905)
- [RFC 8915 — Network Time Security (NTS)](https://datatracker.ietf.org/doc/html/rfc8915)
- [chrony Documentation](https://chrony-project.org/documentation.html)
- [pool.ntp.org — NTP Pool Project](https://www.ntppool.org/en/)
- [Google Public NTP — Leap Smear](https://developers.google.com/time/smear)
- [linuxptp Project](https://linuxptp.sourceforge.net/)
