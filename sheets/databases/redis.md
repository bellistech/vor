# Redis (In-Memory Data Store)

In-memory key-value store supporting strings, hashes, lists, sets, sorted sets, pub/sub, and more.

## Connection

```bash
redis-cli
redis-cli -h redis.example.com -p 6379 -a mypassword
redis-cli -n 2                        # select database 2
redis-cli --tls -h redis.example.com   # TLS connection
```

## Strings

```bash
# SET user:1:name "Alice"
# SET session:abc "data" EX 3600          # expires in 1 hour
# SET lock:resource "owner" NX EX 30      # set only if not exists (distributed lock)
# GET user:1:name
# MSET key1 "val1" key2 "val2"
# MGET key1 key2
# INCR page:views                          # atomic increment
# INCRBY counter 5
# DECR counter
# APPEND key " more data"
# STRLEN key
# SETNX key "value"                        # set if not exists
# GETSET key "new"                         # set and return old value
```

## Key Management

```bash
# KEYS user:*                              # find keys (slow, avoid in production)
# SCAN 0 MATCH user:* COUNT 100           # iterate keys safely
# EXISTS key
# DEL key1 key2
# UNLINK key1 key2                         # async delete
# TYPE key
# RENAME old_key new_key
# EXPIRE key 300                           # set TTL in seconds
# PEXPIRE key 5000                         # TTL in milliseconds
# TTL key                                  # check remaining TTL (-1 = no expiry, -2 = gone)
# PERSIST key                              # remove expiry
# DBSIZE                                   # total key count
# FLUSHDB                                  # clear current database
```

## Hashes

```bash
# HSET user:1 name "Alice" email "alice@example.com" age 30
# HGET user:1 name
# HMGET user:1 name email
# HGETALL user:1
# HDEL user:1 age
# HEXISTS user:1 email
# HINCRBY user:1 age 1
# HKEYS user:1
# HVALS user:1
# HLEN user:1
```

## Lists

```bash
# LPUSH queue "task1"                      # push to head
# RPUSH queue "task2"                      # push to tail
# LPOP queue                               # pop from head
# RPOP queue                               # pop from tail
# BLPOP queue 30                           # blocking pop (30s timeout)
# BRPOP queue 0                            # blocking pop (wait forever)
# LRANGE queue 0 -1                        # get all elements
# LRANGE queue 0 9                         # first 10 elements
# LLEN queue
# LINDEX queue 0                           # element at index
# LREM queue 0 "task1"                     # remove all occurrences
# LTRIM queue 0 99                         # keep only first 100
```

## Sets

```bash
# SADD tags "go" "linux" "docker"
# SMEMBERS tags
# SISMEMBER tags "go"                      # check membership
# SCARD tags                               # count members
# SREM tags "docker"
# SPOP tags                                # remove random element
# SRANDMEMBER tags 2                       # 2 random members (no remove)
# SUNION tags1 tags2                       # union
# SINTER tags1 tags2                       # intersection
# SDIFF tags1 tags2                        # difference
```

## Sorted Sets

```bash
# ZADD leaderboard 100 "alice" 85 "bob" 92 "carol"
# ZSCORE leaderboard "alice"
# ZRANK leaderboard "alice"                # rank (0-based, low to high)
# ZREVRANK leaderboard "alice"             # rank (high to low)
# ZRANGE leaderboard 0 -1 WITHSCORES      # all, ascending
# ZREVRANGE leaderboard 0 9 WITHSCORES    # top 10, descending
# ZRANGEBYSCORE leaderboard 80 100        # scores between 80-100
# ZINCRBY leaderboard 10 "bob"            # add 10 to bob's score
# ZREM leaderboard "carol"
# ZCARD leaderboard                        # count members
# ZCOUNT leaderboard 80 100               # count in score range
```

## Pub/Sub

```bash
# SUBSCRIBE notifications                  # subscribe to channel
# PSUBSCRIBE user:*                        # pattern subscribe
# PUBLISH notifications "new message"      # publish to channel
```

```bash
# In terminal 1:
redis-cli SUBSCRIBE events
# In terminal 2:
redis-cli PUBLISH events "hello"
```

## Persistence

### RDB (point-in-time snapshots)

```bash
# BGSAVE                                   # trigger background save
# LASTSAVE                                 # timestamp of last save
# CONFIG SET save "900 1 300 10 60 10000"  # save after 900s if 1 key changed, etc.
```

### AOF (append-only file)

```bash
# CONFIG SET appendonly yes
# CONFIG SET appendfsync everysec          # fsync every second (recommended)
# BGREWRITEAOF                             # compact the AOF file
```

## TTL & Expiration

```bash
# SET cache:page "html" EX 600            # 10 minutes
# EXPIRE key 3600                          # set TTL on existing key
# EXPIREAT key 1735689600                  # expire at Unix timestamp
# TTL key                                  # remaining seconds
# PTTL key                                 # remaining milliseconds
# PERSIST key                              # remove TTL
```

## Cluster & Sentinel

### Cluster info

```bash
redis-cli --cluster info redis.example.com:6379
redis-cli --cluster check redis.example.com:6379
redis-cli CLUSTER NODES
redis-cli CLUSTER INFO
```

### Sentinel

```bash
redis-cli -h sentinel.example.com -p 26379
# SENTINEL masters
# SENTINEL master mymaster
# SENTINEL get-master-addr-by-name mymaster
# SENTINEL failover mymaster
```

## Common Patterns

### Rate limiting (sliding window)

```bash
# MULTI
# ZADD rate:user:1 <timestamp> <request_id>
# ZREMRANGEBYSCORE rate:user:1 0 <timestamp - window>
# ZCARD rate:user:1
# EXPIRE rate:user:1 <window>
# EXEC
```

### Distributed lock

```bash
# SET lock:resource <unique_id> NX EX 30   # acquire
# -- verify value before release --
# DEL lock:resource                         # release (use Lua for safety)
```

### Cache-aside pattern

```bash
# GET cache:user:1
# -- if miss: query DB, then SET cache:user:1 <data> EX 300
```

## Diagnostics

```bash
redis-cli INFO                             # full server info
redis-cli INFO memory                      # memory usage
redis-cli INFO replication                 # replication status
redis-cli SLOWLOG GET 10                   # last 10 slow queries
redis-cli MONITOR                          # live command stream (debug only)
redis-cli CLIENT LIST                      # connected clients
redis-cli MEMORY USAGE key                 # memory for one key
redis-cli DEBUG SLEEP 0                    # test connectivity
```

## Tips

- Never use `KEYS *` in production. Use `SCAN` with a cursor for safe iteration.
- `UNLINK` is non-blocking `DEL`. Use it for large keys to avoid stalling.
- `SET key val NX EX 30` is the correct pattern for distributed locks (atomic set-if-not-exists with TTL).
- Redis is single-threaded for commands. One slow command blocks everything. Watch `SLOWLOG`.
- Use hashes instead of many individual keys for related data. They are more memory-efficient.
- `BLPOP`/`BRPOP` implement a reliable queue pattern without polling.
- Set `maxmemory` and `maxmemory-policy` (e.g., `allkeys-lru`) to prevent OOM.
- Pub/Sub messages are fire-and-forget. If a subscriber is disconnected, messages are lost. Use Streams for durable messaging.
