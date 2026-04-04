# D-Bus (Desktop Bus IPC System)

D-Bus is an inter-process communication system providing two message buses — a system bus for OS services and a session bus for user applications — supporting method calls, signals, and properties over well-known interfaces with policy-based access control.

## Bus Types and Basics

### Connecting to Buses

```bash
# List all services on the system bus
busctl list

# List all services on the session bus
busctl list --user

# Show the bus address
echo $DBUS_SESSION_BUS_ADDRESS

# System bus socket
ls -la /run/dbus/system_bus_socket

# Check if a name is owned
busctl status org.freedesktop.NetworkManager
```

### Well-Known Names

```bash
# Common system bus services
# org.freedesktop.systemd1           - systemd
# org.freedesktop.NetworkManager     - NetworkManager
# org.freedesktop.login1             - logind (sessions/seats)
# org.freedesktop.UDisks2            - disk management
# org.freedesktop.PolicyKit1         - PolicyKit authorization
# org.freedesktop.hostname1          - hostnamed
# org.freedesktop.timedate1          - timedated
# org.freedesktop.locale1            - localed
# org.freedesktop.resolve1           - resolved

# Common session bus services
# org.freedesktop.Notifications      - desktop notifications
# org.freedesktop.portal.Desktop     - XDG desktop portal
# org.freedesktop.secrets            - Secret Service API
```

## busctl Commands

### Introspection

```bash
# Introspect a service (shows interfaces, methods, signals, properties)
busctl introspect org.freedesktop.systemd1 /org/freedesktop/systemd1

# Tree view of object paths
busctl tree org.freedesktop.systemd1

# Introspect with full interface details
busctl introspect org.freedesktop.login1 /org/freedesktop/login1

# Get a specific property
busctl get-property org.freedesktop.hostname1 \
  /org/freedesktop/hostname1 \
  org.freedesktop.hostname1 Hostname

# Set a property
busctl set-property org.freedesktop.hostname1 \
  /org/freedesktop/hostname1 \
  org.freedesktop.hostname1 StaticHostname s "myhost"
```

### Method Calls

```bash
# Call a method with busctl
busctl call org.freedesktop.systemd1 \
  /org/freedesktop/systemd1 \
  org.freedesktop.systemd1.Manager \
  ListUnits

# Restart a service via D-Bus
busctl call org.freedesktop.systemd1 \
  /org/freedesktop/systemd1 \
  org.freedesktop.systemd1.Manager \
  RestartUnit ss "sshd.service" "replace"

# Get hostname
busctl call org.freedesktop.hostname1 \
  /org/freedesktop/hostname1 \
  org.freedesktop.DBus.Properties \
  Get ss "org.freedesktop.hostname1" "Hostname"

# Send a desktop notification (session bus)
busctl --user call org.freedesktop.Notifications \
  /org/freedesktop/Notifications \
  org.freedesktop.Notifications \
  Notify susssasa\{sv\}i \
  "myapp" 0 "" "Title" "Body" 0 0 5000
```

### Monitoring

```bash
# Monitor all messages on the system bus
busctl monitor

# Monitor a specific service
busctl monitor org.freedesktop.systemd1

# Monitor with match rules
busctl monitor --match "type='signal',sender='org.freedesktop.login1'"

# Capture to a file (pcap-like)
busctl capture > dbus-trace.bin
```

## dbus-send and dbus-monitor

### dbus-send

```bash
# Call a method with dbus-send
dbus-send --system --print-reply \
  --dest=org.freedesktop.systemd1 \
  /org/freedesktop/systemd1 \
  org.freedesktop.systemd1.Manager.ListUnitFiles

# Get property via dbus-send
dbus-send --system --print-reply \
  --dest=org.freedesktop.hostname1 \
  /org/freedesktop/hostname1 \
  org.freedesktop.DBus.Properties.Get \
  string:"org.freedesktop.hostname1" \
  string:"Hostname"

# Send a signal
dbus-send --session --type=signal \
  /com/example/Signal \
  com.example.Signal.MySignal \
  string:"hello"
```

### dbus-monitor

```bash
# Monitor all system bus messages
dbus-monitor --system

# Monitor session bus signals only
dbus-monitor --session "type='signal'"

# Monitor specific interface
dbus-monitor --system "interface='org.freedesktop.login1.Manager'"

# Monitor method calls to a destination
dbus-monitor --system "type='method_call',destination='org.freedesktop.systemd1'"

# Profile format (timestamps)
dbus-monitor --system --profile
```

## D-Bus Types and Signatures

### Type Signatures

```bash
# Type codes used in D-Bus signatures:
# s  - string          a  - array
# b  - boolean         v  - variant
# y  - byte (uint8)    (  - struct start
# n  - int16           )  - struct end
# q  - uint16          {  - dict entry start
# i  - int32           }  - dict entry end
# u  - uint32          o  - object path
# x  - int64           g  - signature
# t  - uint64          h  - unix fd
# d  - double

# Examples:
# "s"       - one string
# "ss"      - two strings
# "as"      - array of strings
# "a{sv}"   - dictionary (string -> variant)
# "(si)"    - struct of string and int32
```

## Policy Configuration

### System Bus Policy

```xml
<!-- /etc/dbus-1/system.d/com.example.Service.conf -->
<!DOCTYPE busconfig PUBLIC
  "-//freedesktop//DTD D-BUS Bus Configuration 1.0//EN"
  "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <!-- Allow the service user to own the name -->
  <policy user="myservice">
    <allow own="com.example.Service"/>
  </policy>

  <!-- Allow anyone to call methods -->
  <policy context="default">
    <allow send_destination="com.example.Service"
           send_interface="com.example.Service.Manager"/>
    <allow send_destination="com.example.Service"
           send_interface="org.freedesktop.DBus.Introspectable"/>
  </policy>

  <!-- Deny access to admin interface except root -->
  <policy context="default">
    <deny send_destination="com.example.Service"
          send_interface="com.example.Service.Admin"/>
  </policy>
  <policy user="root">
    <allow send_destination="com.example.Service"
           send_interface="com.example.Service.Admin"/>
  </policy>
</busconfig>
```

## sd-bus API (C)

### Basic Client Example

```c
#include <systemd/sd-bus.h>
#include <stdio.h>

int main(void) {
    sd_bus *bus = NULL;
    sd_bus_error error = SD_BUS_ERROR_NULL;
    char *hostname = NULL;

    /* Connect to system bus */
    sd_bus_open_system(&bus);

    /* Get hostname property */
    sd_bus_get_property_string(bus,
        "org.freedesktop.hostname1",
        "/org/freedesktop/hostname1",
        "org.freedesktop.hostname1",
        "Hostname",
        &error,
        &hostname);

    printf("Hostname: %s\n", hostname);

    free(hostname);
    sd_bus_error_free(&error);
    sd_bus_unref(bus);
    return 0;
}
```

```bash
# Compile with sd-bus
gcc -o hostname hostname.c $(pkg-config --cflags --libs libsystemd)
```

## Tips

- Use `busctl` over `dbus-send` on systemd systems for better output formatting and tab completion
- Run `busctl tree <service>` first to discover available object paths before introspecting
- Use `busctl monitor` with `--match` filters to reduce noise when debugging specific interactions
- D-Bus type signature `a{sv}` (dict of string to variant) is the universal extensible parameter pattern
- Session bus services often require a running desktop session; use `systemctl --user` to manage them
- Policy files in `/etc/dbus-1/system.d/` are loaded at bus startup; restart dbus after changes
- Use `busctl capture` for detailed tracing; the output can be analyzed with `dbus-monitor --pcap`
- The `org.freedesktop.DBus.Properties` interface is universal for getting and setting any property
- Always check the `org.freedesktop.DBus.Introspectable` interface to discover available methods
- Use `gdbus` on GNOME systems as an alternative to `busctl` for GLib-based introspection

## See Also

- systemd, systemctl, journalctl, polkit, udev

## References

- [D-Bus Specification](https://dbus.freedesktop.org/doc/dbus-specification.html)
- [D-Bus Tutorial](https://dbus.freedesktop.org/doc/dbus-tutorial.html)
- [sd-bus API Reference](https://www.freedesktop.org/software/systemd/man/sd-bus.html)
- [busctl Manual](https://www.freedesktop.org/software/systemd/man/busctl.html)
