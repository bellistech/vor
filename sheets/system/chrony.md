# Chrony (NTP Client/Server)

> Modern NTP implementation for time synchronization; faster convergence and better accuracy than ntpd, especially for intermittent connections.

## chronyc Commands

### Checking Status

```bash
# Overall tracking info (offset, frequency, stratum)
chronyc tracking

# List NTP sources with status
chronyc sources
chronyc sources -v          # Verbose (explains column headers)

# Source statistics (offset, frequency, jitter)
chronyc sourcestats

# Activity summary (online/offline sources)
chronyc activity
```

### Time Correction

```bash
# Force an immediate step correction (if offset is large)
chronyc makestep

# Burst measurement (8 requests in quick succession)
chronyc burst 4/8           # 4 good / 8 max measurements

# Online/offline mode for sources
chronyc offline             # Mark all sources offline
chronyc online              # Mark all sources online

# Manual time input
chronyc settime 2024-01-15T12:00:00Z

# Reload sources from config
chronyc reload sources
```

### Monitoring

```bash
# Show connected clients (when running as server)
chronyc clients

# Show server statistics
chronyc serverstats

# NTP data for specific source
chronyc ntpdata 10.0.0.1

# Check if synchronized
chronyc waitsync 30 0.01    # Wait up to 30 checks for <10ms offset
```

### Maintenance

```bash
# Check chrony is running
systemctl status chronyd

# Restart
systemctl restart chronyd

# Verify NTP synchronization
timedatectl show --property=NTPSynchronized
```

## Configuration

### /etc/chrony.conf (or /etc/chrony/chrony.conf)

```bash
# NTP servers (prefer pool for redundancy)
pool pool.ntp.org iburst maxsources 4
server time.google.com iburst prefer
server time.cloudflare.com iburst

# iburst  — send 4 requests on start for fast initial sync
# prefer  — favor this source when multiple are available

# Allow step correction on startup if offset > 1s (first 3 updates)
makestep 1.0 3

# Sync hardware clock (RTC) to system time every 11 minutes
rtcsync

# Record measurement history for faster recovery after restart
driftfile /var/lib/chrony/drift

# Enable logging
logdir /var/log/chrony
log measurements statistics tracking

# Key file for authenticated NTP
keyfile /etc/chrony/chrony.keys

# Minimum sources before updating clock
minsources 2
```

### NTP Server Configuration

```bash
# Allow clients from local network
allow 192.168.1.0/24
allow 10.0.0.0/8

# Deny specific networks
deny 192.168.1.100

# Serve time even when not synchronized (stratum 10)
local stratum 10 orphan

# Rate limiting for clients
ratelimit interval 1 burst 16

# Bind to specific interface
bindaddress 0.0.0.0
port 123
```

### Advanced Options

```bash
# Minimum/maximum poll interval (2^N seconds)
server time.google.com minpoll 4 maxpoll 10   # 16s to 1024s

# Leap second handling
leapsectz right/UTC

# Hardware timestamping (for PTP-grade accuracy)
hwtimestamp eth0

# Temperature compensation
tempcomp /sys/class/hwmon/hwmon0/temp1_input 30 26000 0.0 0.000183 0.0
```

## Chrony vs ntpd

```
Feature              chrony                  ntpd
-------              ------                  ----
Initial sync         Faster (seconds)        Slower (minutes)
Intermittent conn    Handles well            Poor
Isolated networks    local stratum works     Needs orphan mode
Virtual machines     Better clock handling   Clock jitter issues
Memory footprint     Smaller                 Larger
HW timestamping      Supported               Supported
Configuration        Simpler                 More complex
Default on RHEL 8+   Yes                     No (removed)
```

## Tips

- Use `iburst` on all server/pool lines for faster initial synchronization after boot.
- `makestep 1.0 3` is the standard safety net; it allows stepping the clock in the first 3 updates if offset exceeds 1 second.
- Use `pool` instead of individual `server` lines for automatic DNS-based redundancy.
- Chrony handles VM clock drift and suspend/resume much better than ntpd.
- Use `chronyc tracking` to check the `System time` offset; it should be under 1ms on a good network.
- For PTP-level accuracy (microseconds), enable hardware timestamping with `hwtimestamp`.

## See Also

- systemd, systemd-timers, kernel, sysctl

## References

- [chronyc(1) Man Page](https://man7.org/linux/man-pages/man1/chronyc.1.html)
- [chrony.conf(5) Man Page](https://man7.org/linux/man-pages/man5/chrony.conf.5.html)
- [Chrony Project Documentation](https://chrony-project.org/documentation.html)
- [chrony.conf Reference](https://chrony-project.org/doc/4.4/chrony.conf.html)
- [chronyc Command Reference](https://chrony-project.org/doc/4.4/chronyc.html)
- [Chrony FAQ](https://chrony-project.org/faq.html)
- [Arch Wiki — Chrony](https://wiki.archlinux.org/title/Chrony)
- [Red Hat — Configuring NTP Using chrony](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/configuring-time-synchronization_configuring-basic-system-settings)
- [RFC 5905 — NTPv4 Specification](https://datatracker.ietf.org/doc/html/rfc5905)
- [Kernel Timekeeping Documentation](https://www.kernel.org/doc/html/latest/timers/timekeeping.html)
