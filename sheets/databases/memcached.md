# Memcached (Distributed Memory Cache)

High-performance distributed memory object caching system for speeding up dynamic web applications by alleviating database load.

## Connection

```bash
memcached -d -m 64 -p 11211 -u memcache        # start daemon with 64MB
memcached -d -m 1024 -c 4096 -t 4 -p 11211     # 1GB, 4096 connections, 4 threads
memcached -l 0.0.0.0 -p 11211 -m 256           # listen on all interfaces
memcached -vv                                    # very verbose (debug)
```

## Telnet Interface

```bash
telnet localhost 11211                           # connect to memcached
echo "stats" | nc localhost 11211               # quick stats via netcat
```

## Storage Commands

```bash
# set <key> <flags> <exptime> <bytes>\r\n<data>\r\n
set mykey 0 3600 5
hello

# add — store only if key does NOT exist
add mykey 0 3600 5
hello

# replace — store only if key DOES exist
replace mykey 0 3600 7
goodbye

# append — append data after existing value
append mykey 0 0 6
_world

# prepend — prepend data before existing value
prepend mykey 0 0 7
hello_
```

## Retrieval Commands

```bash
get mykey                                        # get single key
get key1 key2 key3                               # multi-get
gets mykey                                       # get with CAS token

# Response format:
# VALUE <key> <flags> <bytes> [<cas unique>]\r\n
# <data>\r\n
# END\r\n
```

## Delete and Arithmetic

```bash
delete mykey                                     # remove key
delete mykey noreply                             # fire-and-forget delete

incr counter 1                                   # increment by 1
incr counter 10                                  # increment by 10
decr counter 5                                   # decrement by 5
# Note: incr/decr operate on 64-bit unsigned integers stored as ASCII
```

## Check-and-Set (CAS)

```bash
# Optimistic locking — update only if CAS token matches
gets mykey
# VALUE mykey 0 5 12345
# hello
# END

cas mykey 0 3600 7 12345
goodbye
# STORED — success, token matched
# EXISTS — another client modified the key
# NOT_FOUND — key does not exist
```

## Expiration

```bash
# exptime = 0 — never expires (until evicted by LRU)
set permanent 0 0 4
data

# exptime < 2592000 (30 days) — relative seconds from now
set short_lived 0 300 4
data

# exptime >= 2592000 — absolute Unix timestamp
set scheduled 0 1735689600 4
data

# touch — update expiration without fetching
touch mykey 3600

# gat — get and touch (atomic get + update expiry)
gat 3600 mykey
```

## Flush and Version

```bash
flush_all                                        # invalidate all items immediately
flush_all 60                                     # invalidate all items in 60 seconds
version                                          # server version
verbosity 2                                      # set log verbosity
quit                                             # close connection
```

## Stats Commands

```bash
stats                                            # general statistics
stats items                                      # per-slab item counts
stats slabs                                      # slab allocator stats
stats sizes                                      # item size distribution
stats cachedump <slab_id> <limit>               # dump keys from slab
stats conns                                      # connection details
stats settings                                   # runtime settings
```

## Key Stats Fields

```bash
# stats output — important fields:
# curr_items        — current number of items stored
# total_items       — total items stored since start
# bytes             — current bytes used
# limit_maxbytes    — max bytes allowed (-m flag)
# curr_connections  — open connections
# get_hits          — cache hits
# get_misses        — cache misses
# evictions         — items evicted (LRU) to free memory
# cmd_get           — total GET commands
# cmd_set           — total SET commands

# Hit ratio calculation:
# hit_ratio = get_hits / (get_hits + get_misses) * 100
```

## memcached-tool

```bash
memcached-tool localhost:11211 display           # slab allocation summary
memcached-tool localhost:11211 stats             # formatted stats
memcached-tool localhost:11211 dump              # dump all keys (small caches)
memcached-tool localhost:11211 settings          # show settings
memcached-tool localhost:11211 sizes             # item size histogram
```

## Slab Allocator Tuning

```bash
# Default slab growth factor is 1.25
memcached -f 1.25 -m 1024                       # default growth factor
memcached -f 2 -m 1024                          # double slab sizes each class
memcached -n 48                                  # minimum item size (bytes)
memcached -I 10m                                 # max item size (default 1MB)

# Slab reassignment (runtime)
slabs reassign <src_class> <dst_class>          # move slab page between classes
slabs automove 1                                 # enable automatic slab rebalancing
slabs automove 2                                 # aggressive mode
```

## Client Libraries

```bash
# Python — pymemcache (recommended)
pip install pymemcache
# from pymemcache.client.hash import HashClient
# client = HashClient([('host1', 11211), ('host2', 11211)])

# Python — python-memcached (legacy)
pip install python-memcached

# Ruby
gem install dalli
# client = Dalli::Client.new('host1:11211,host2:11211')

# PHP — built-in extension
# $m = new Memcached();
# $m->addServer('localhost', 11211);

# Node.js
npm install memcached
# var Memcached = require('memcached');
# var mc = new Memcached('localhost:11211');

# Java — spymemcached
# MemcachedClient c = new MemcachedClient(
#     new InetSocketAddress("localhost", 11211));

# Go — bradfitz/gomemcache
# go get github.com/bradfitz/gomemcache/memcache
```

## Configuration Flags

```bash
memcached -d                                     # daemonize
memcached -p 11211                               # TCP port
memcached -U 11211                               # UDP port (0 to disable)
memcached -l 127.0.0.1                          # listen address
memcached -m 1024                                # max memory in MB
memcached -c 1024                                # max connections
memcached -t 4                                   # number of threads
memcached -M                                     # return error on OOM (no eviction)
memcached -S                                     # enable SASL authentication
memcached -o modern                              # modern defaults (slab_reassign, etc.)
memcached -B binary                              # binary protocol only
memcached -B ascii                               # ASCII protocol only
```

## Docker

```bash
docker run -d --name memcached -p 11211:11211 memcached:latest
docker run -d --name memcached -p 11211:11211 memcached:latest \
  memcached -m 256 -c 2048 -t 4
```

## Consistent Hashing (Client-Side)

```bash
# Most clients support consistent hashing for multi-server pools
# Ensures minimal key redistribution when servers are added/removed

# Python pymemcache with consistent hashing:
# from pymemcache.client.hash import HashClient
# client = HashClient(
#     [('host1', 11211), ('host2', 11211), ('host3', 11211)],
#     use_pooling=True,
#     max_pool_size=4,
#     hasher=RendezvousHash   # or KetamaHash
# )

# libmemcached (C) — ketama consistent hashing:
# memcached_behavior_set(mc, MEMCACHED_BEHAVIOR_DISTRIBUTION,
#     MEMCACHED_DISTRIBUTION_CONSISTENT_KETAMA);
```

## Monitoring

```bash
# Calculate hit ratio
echo "stats" | nc localhost 11211 | grep -E 'get_hits|get_misses'

# Watch stats in real-time (every 2 seconds)
watch -n 2 "echo stats | nc localhost 11211 | grep -E 'curr_items|bytes|evictions|get_hits'"

# Prometheus exporter
docker run -d -p 9150:9150 prom/memcached-exporter \
  --memcached.address=memcached:11211
```

## Tips

- Keep keys under 250 bytes and values under 1 MB (adjustable with -I flag)
- Use consistent hashing to minimize cache invalidation when scaling horizontally
- Monitor the evictions stat closely; rising evictions mean you need more memory
- Set expiration times on all keys to prevent stale data accumulation
- Use CAS (check-and-set) for optimistic locking instead of external locks
- Prefer multi-get over individual gets to reduce round trips and latency
- Disable UDP (-U 0) in production to prevent DDoS amplification attacks
- Use binary protocol (-B binary) for better performance and reduced parsing overhead
- Tune the slab growth factor (-f) if your value sizes cluster at specific ranges
- Enable slab automove to let memcached rebalance memory between slab classes
- Never treat memcached as persistent storage; always have a backing data source
- Use namespacing in keys (e.g., user:1234:profile) for logical organization

## See Also

- redis
- etcd
- elasticsearch

## References

- [Memcached Official Wiki](https://github.com/memcached/memcached/wiki)
- [Memcached Protocol Specification](https://github.com/memcached/memcached/blob/master/doc/protocol.txt)
- [pymemcache Documentation](https://pymemcache.readthedocs.io/)
- [Consistent Hashing — Wikipedia](https://en.wikipedia.org/wiki/Consistent_hashing)
- [Scaling Memcached at Facebook](https://research.facebook.com/publications/scaling-memcache-at-facebook/)
