# iperf3 (Network Performance Measurement)

Measure TCP, UDP, and SCTP bandwidth, jitter, and packet loss between two endpoints.

## Server Mode

```bash
# Start iperf3 server on default port 5201
iperf3 -s

# Server on custom port
iperf3 -s -p 5001

# Server in daemon mode (background)
iperf3 -s -D
iperf3 -s -D --logfile /var/log/iperf3.log

# Server with one-off mode (exit after single test)
iperf3 -s -1

# Bind to specific interface
iperf3 -s -B 10.0.0.1

# Kill daemon
pkill iperf3
# or find PID
cat /var/run/iperf3.pid
```

## TCP Testing

```bash
# Basic TCP test (10 seconds, default)
iperf3 -c server.example.com

# Specify duration
iperf3 -c server -t 30          # 30 seconds
iperf3 -c server -t 60          # 60 seconds

# Specify bytes to transfer (instead of time)
iperf3 -c server -n 1G          # transfer 1 GB then stop

# Specify target bandwidth
iperf3 -c server -b 100M        # target 100 Mbps
iperf3 -c server -b 1G          # target 1 Gbps

# Parallel streams
iperf3 -c server -P 4           # 4 parallel TCP streams
iperf3 -c server -P 8           # 8 parallel streams

# Bidirectional test
iperf3 -c server --bidir        # simultaneous send + receive

# Reverse mode (server sends to client)
iperf3 -c server -R

# Specify MSS (Maximum Segment Size)
iperf3 -c server -M 1400        # set MSS to 1400 bytes

# Set TCP window size (socket buffer)
iperf3 -c server -w 256K        # 256 KB window
iperf3 -c server -w 1M          # 1 MB window
iperf3 -c server -w 4M          # 4 MB for high-latency links

# Set congestion control algorithm
iperf3 -c server -C bbr         # use BBR
iperf3 -c server -C cubic       # use CUBIC (default Linux)

# Zero-copy mode (sendfile syscall, reduces CPU)
iperf3 -c server -Z

# Set TOS/DSCP
iperf3 -c server -S 0x28        # AF11 (DSCP 10)
iperf3 -c server -S 0xB8        # EF (DSCP 46)

# Custom port
iperf3 -c server -p 5001

# Bind source address
iperf3 -c server -B 10.0.0.100
```

## UDP Testing

```bash
# UDP test (must specify bandwidth, default 1 Mbps)
iperf3 -c server -u

# UDP with target bandwidth
iperf3 -c server -u -b 100M     # 100 Mbps UDP
iperf3 -c server -u -b 1G       # 1 Gbps UDP
iperf3 -c server -u -b 0        # unlimited (blast mode)

# UDP with specific packet size
iperf3 -c server -u -l 1400     # 1400 byte datagrams
iperf3 -c server -u -l 64       # 64 byte datagrams (small packet stress)
iperf3 -c server -u -l 8192     # 8 KB datagrams (jumbo)

# UDP bidirectional
iperf3 -c server -u -b 50M --bidir

# UDP reverse
iperf3 -c server -u -b 100M -R

# Key UDP metrics in output:
# - Bandwidth (Mbits/sec)
# - Jitter (ms) - variation in packet arrival time
# - Lost/Total Datagrams (loss percentage)
```

## SCTP Testing

```bash
# SCTP test (requires SCTP support compiled in)
iperf3 -c server --sctp

# SCTP with bandwidth target
iperf3 -c server --sctp -b 500M

# SCTP with parallel streams
iperf3 -c server --sctp -P 4
```

## Output Formats

```bash
# JSON output (machine-parseable)
iperf3 -c server -J
iperf3 -c server -J > results.json

# JSON with pretty formatting
iperf3 -c server -J | python3 -m json.tool

# Extract key metrics from JSON
iperf3 -c server -J | python3 -c "
import json, sys
data = json.load(sys.stdin)
end = data['end']
sent = end['sum_sent']
recv = end['sum_received']
print(f'Sent: {sent[\"bits_per_second\"]/1e6:.2f} Mbps')
print(f'Recv: {recv[\"bits_per_second\"]/1e6:.2f} Mbps')
print(f'Retransmits: {sent.get(\"retransmits\", \"N/A\")}')
"

# Verbose output
iperf3 -c server -V

# Log to file
iperf3 -c server --logfile /tmp/iperf3.log

# Report interval (default 1 second)
iperf3 -c server -i 0.5         # every 0.5 seconds
iperf3 -c server -i 2           # every 2 seconds

# Omit first N seconds (skip TCP slow start)
iperf3 -c server -O 3           # omit first 3 seconds from stats

# Time-series output (interval reports)
iperf3 -c server -t 60 -i 1 -J | python3 -c "
import json, sys
data = json.load(sys.stdin)
for interval in data['intervals']:
    stream = interval['sum']
    t = stream['start']
    bps = stream['bits_per_second'] / 1e6
    print(f'{t:.1f}s  {bps:.2f} Mbps')
"
```

## Window Size Tuning

```bash
# Check current kernel TCP buffer settings
sysctl net.core.rmem_max
sysctl net.core.wmem_max
sysctl net.ipv4.tcp_rmem
sysctl net.ipv4.tcp_wmem

# Calculate optimal window size for a high-latency link
# BDP (Bandwidth-Delay Product) = Bandwidth * RTT
# Example: 1 Gbps link with 50ms RTT
# BDP = 1,000,000,000 * 0.050 / 8 = 6,250,000 bytes = ~6 MB

# Set kernel maximums (requires root)
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 65536 16777216"

# Test with explicit window size
iperf3 -c server -w 6M -t 30

# Compare auto-tuning vs manual
iperf3 -c server -t 30                  # auto-tuned
iperf3 -c server -t 30 -w 6M            # manual BDP-matched

# Test multiple window sizes
for w in 64K 256K 1M 4M 8M 16M; do
  echo "=== Window: $w ==="
  iperf3 -c server -w $w -t 10 -O 2
  sleep 1
done
```

## Advanced Scenarios

```bash
# Multi-stream with per-stream reporting
iperf3 -c server -P 4 -t 30 -i 1

# Simulate constrained link (combine with tc)
# Server side: add latency and loss
tc qdisc add dev eth0 root netem delay 50ms 10ms loss 1%
iperf3 -s
# Client side:
iperf3 -c server -t 30 -J

# Remove traffic shaping
tc qdisc del dev eth0 root

# Repeated tests (scripted)
for i in $(seq 1 10); do
  echo "=== Run $i ==="
  iperf3 -c server -t 10 -J >> results_all.json
  sleep 2
done

# Test path MTU discovery
iperf3 -c server -M 1500          # standard MTU
iperf3 -c server -M 9000          # jumbo frames
iperf3 -c server -M 1400          # common VPN/tunnel MTU

# Test with specific number of blocks
iperf3 -c server -k 100           # send exactly 100 blocks

# CPU affinity (pin to specific core)
iperf3 -c server -A 0             # pin to CPU 0
iperf3 -c server -A 2,3           # pin client to 2, server to 3

# IPv6 testing
iperf3 -c server -6

# Multicast UDP test
iperf3 -c 239.1.1.1 -u -b 10M -T 32
```

## Interpreting Results

```bash
# Key TCP metrics:
# - Bitrate (sender/receiver): effective throughput
# - Retr: TCP retransmissions (indicates loss or congestion)
# - Cwnd: congestion window size (should grow if no loss)

# Key UDP metrics:
# - Bitrate: sending rate
# - Jitter: inter-packet delay variation (< 1ms good, > 5ms poor)
# - Lost/Total: packet loss ratio (< 0.1% good, > 1% problematic)

# Example output interpretation:
# [ ID] Interval       Transfer    Bitrate         Retr  Cwnd
# [  5]  0.00-10.00 sec 1.10 GBytes   943 Mbits/sec   12  3.01 MBytes
#
# 943 Mbps on 1G link = 94.3% efficiency
# 12 retransmissions = minor congestion
# 3.01 MB Cwnd = healthy window

# Red flags:
# - Bitrate << expected link speed: bottleneck somewhere
# - High retransmissions: packet loss, congestion, or buffer overflow
# - Cwnd oscillating wildly: congestion control instability
# - Large jitter (UDP): queue depth variation, possible QoS issue
# - Asymmetric --bidir results: one direction is constrained
```

## Tips

- Always run the server first; the client initiates the test and needs a listening server.
- Use `-O 3` to omit the first 3 seconds and exclude TCP slow start from your measurements.
- Use `-P 4` (or more) parallel streams to saturate high-bandwidth links -- a single stream may be CPU-bound.
- Calculate BDP (bandwidth x delay) to set the correct window size for high-latency links.
- Use `-Z` (zero-copy) on Linux to reduce CPU overhead when testing 10G+ links.
- JSON output (`-J`) is essential for automated testing; parse with jq or Python for trend analysis.
- Use `--bidir` instead of running two separate tests to measure true simultaneous bidirectional throughput.
- For UDP tests, always specify `-b` (bandwidth) -- the default 1 Mbps is almost never what you want.
- Combine iperf3 with `tc netem` to simulate real-world conditions (latency, jitter, loss).
- Use `-R` (reverse mode) to test server-to-client throughput without opening firewall ports on the client.
- Run tests at different times of day to detect peak-hour congestion patterns.
- Use daemon mode (`-s -D`) with `--logfile` for persistent monitoring endpoints.

## See Also

- wireshark (capture and analyze iperf3 traffic at packet level)
- postfix (network performance affects mail delivery latency)

## References

- [iperf3 Official Documentation](https://software.es.net/iperf/)
- [iperf3 GitHub Repository](https://github.com/esnet/iperf)
- [iperf3 Man Page](https://software.es.net/iperf/invoking.html)
- [ESnet Network Tuning Guide](https://fasterdata.es.net/host-tuning/)
- [TCP Tuning Guide](https://fasterdata.es.net/host-tuning/linux/)
- [Understanding iperf3 Results](https://fasterdata.es.net/performance-testing/network-troubleshooting-tools/iperf/)
