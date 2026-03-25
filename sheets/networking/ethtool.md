# ethtool (Ethernet Tool)

Query and control network device driver and hardware settings — speed, duplex, offloading, ring buffers, and statistics.

## Interface Information

### Basic info
```bash
ethtool eth0                           # link status, speed, duplex, autoneg
ethtool -i eth0                        # driver info (driver name, version, firmware)
ethtool -P eth0                        # permanent hardware MAC address
```

### Link status
```bash
ethtool eth0 | grep 'Link detected'   # quick link check
ethtool eth0 | grep 'Speed'           # current speed
```

## Speed and Duplex

### View supported modes
```bash
ethtool eth0 | grep -A 20 'Supported link modes'
ethtool eth0 | grep -A 5 'Advertised link modes'
```

### Set speed and duplex
```bash
ethtool -s eth0 speed 1000 duplex full autoneg off   # force 1Gbps full duplex
ethtool -s eth0 speed 10000 duplex full               # force 10Gbps
ethtool -s eth0 autoneg on                            # re-enable autonegotiation
```

## Ring Buffers

### View ring buffer sizes
```bash
ethtool -g eth0                        # current and max ring buffer sizes
```

### Set ring buffer sizes
```bash
ethtool -G eth0 rx 4096               # increase RX ring buffer
ethtool -G eth0 tx 4096               # increase TX ring buffer
ethtool -G eth0 rx 4096 tx 4096       # set both
```

## Offload Features

### View offload settings
```bash
ethtool -k eth0                        # all offload features
ethtool -k eth0 | grep -i offload     # filter for offload settings
ethtool -k eth0 | grep -E '(tcp-segmentation|generic-segmentation|generic-receive)'
```

### Toggle offload features
```bash
ethtool -K eth0 tso on                 # TCP segmentation offload
ethtool -K eth0 gso on                 # generic segmentation offload
ethtool -K eth0 gro on                 # generic receive offload
ethtool -K eth0 lro off                # large receive offload (often disabled)
ethtool -K eth0 tx-checksumming on     # TX checksum offload
ethtool -K eth0 rx-checksumming on     # RX checksum offload
ethtool -K eth0 sg on                  # scatter-gather
```

## Statistics

### NIC statistics
```bash
ethtool -S eth0                        # all driver statistics
ethtool -S eth0 | grep -i error       # error counters
ethtool -S eth0 | grep -i drop        # drop counters
ethtool -S eth0 | grep -i miss        # missed packets
ethtool -S eth0 | grep rx_queue       # per-queue RX stats
ethtool -S eth0 | grep tx_queue       # per-queue TX stats
```

### Key counters to watch
```bash
ethtool -S eth0 | grep -E '(rx_errors|tx_errors|rx_dropped|tx_dropped|rx_missed|rx_crc)'
```

## Interrupt Coalescing

### View coalescing settings
```bash
ethtool -c eth0                        # interrupt coalescing parameters
```

### Tune coalescing
```bash
ethtool -C eth0 rx-usecs 50           # RX interrupt delay (microseconds)
ethtool -C eth0 tx-usecs 50           # TX interrupt delay
ethtool -C eth0 adaptive-rx on        # let driver auto-tune RX
ethtool -C eth0 adaptive-tx on        # let driver auto-tune TX
```

## Flow Control (Pause Frames)

### View pause frame settings
```bash
ethtool -a eth0                        # pause/flow control status
```

### Set pause frames
```bash
ethtool -A eth0 rx on tx on            # enable pause frames
ethtool -A eth0 rx off tx off          # disable pause frames
ethtool -A eth0 autoneg on            # negotiate pause
```

## Wake-on-LAN

### View WoL status
```bash
ethtool eth0 | grep 'Wake-on'
```

### Enable/disable WoL
```bash
ethtool -s eth0 wol g                  # enable magic packet WoL
ethtool -s eth0 wol d                  # disable WoL
```

## Queue and Channel Configuration

### View queues
```bash
ethtool -l eth0                        # number of RX/TX queues (channels)
```

### Set queue count
```bash
ethtool -L eth0 combined 8            # set combined RX/TX queues
ethtool -L eth0 rx 4 tx 4             # separate RX and TX queues
```

## Testing

### Self-test
```bash
ethtool -t eth0 online                 # non-disruptive self-test
ethtool -t eth0 offline               # full test (link goes down!)
```

### Blink LED
```bash
ethtool -p eth0 5                      # blink port LED for 5 seconds (identify cable)
```

## EEE (Energy Efficient Ethernet)

### View and set EEE
```bash
ethtool --show-eee eth0               # EEE status
ethtool --set-eee eth0 eee off        # disable EEE (reduces latency jitter)
```

## Tips

- Increasing ring buffers (`-G`) is often the first step when seeing RX drops under load
- Disabling LRO (`-K eth0 lro off`) is usually necessary when the host is routing or bridging
- TSO/GSO/GRO should generally be on for performance; disable only when troubleshooting
- `ethtool -S` counters are driver-specific — field names vary between nic vendors
- `ethtool -p` (blink LED) is the fastest way to identify which physical port maps to which interface
- Interrupt coalescing trades latency for CPU efficiency — lower `rx-usecs` = lower latency but more CPU
- EEE can add microseconds of latency on idle-to-active transitions; disable for latency-sensitive workloads
- Changes made with `ethtool` are not persistent; use `udev` rules, `networkd`, or `/etc/network/interfaces` `pre-up` for persistence
- RSS queue count should generally match the number of CPU cores for optimal interrupt distribution
