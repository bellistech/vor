# Redis — ELI5

> Redis is a magic notebook held in RAM that everyone shares — write to a page, read from a page, keep a sorted leaderboard, run a queue, broadcast a memo, all in microseconds.

## Prerequisites

(very few — `ramp-up/linux-kernel-eli5.md` helps you understand what RAM and processes are; `ramp-up/postgres-eli5.md` helps you compare a database that lives on disk to a database that lives in memory; `ramp-up/tcp-eli5.md` helps you understand why a network round-trip is slow and why putting the data right next to the application matters)

You do not need to know SQL. You do not need to know how a hard drive is laid out. You do not need to know about indexes or query plans or B-trees. By the end of this sheet you will know what Redis is, why it is so fast, what data structures it gives you, how it survives a reboot, how it spreads across many servers, and you will have typed real `redis-cli` commands and watched them come back in microseconds.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back. We call that "output."

If you see a `>` at the start of a line in a code block, that is the `redis-cli` prompt — you are talking directly to a running Redis server. Lines without a `>` underneath are the server's reply.

## Plain English

### Imagine your computer has a magic notebook

Picture a plain school notebook. Pages numbered. Each page has a name written at the top — `user:42`, `cart:abc`, `score:alice` — and underneath the name, some stuff. A name. A number. A list. A bag. A leaderboard. Whatever you want.

The notebook is shared. Lots of people can write on it at the same time. Everyone sees the same pages. If Alice writes "hello" on the page named `greeting`, then Bob, sitting at a different desk, looks at the page named `greeting` and sees "hello." They are looking at the same notebook.

The notebook is also magic in two ways.

First way: it is fast. Stupidly fast. You can write a page in about ten microseconds. You can read a page in about the same time. Ten microseconds is one hundred-thousandth of a second. In one second, a single Redis server can answer hundreds of thousands of these reads and writes. That is because the whole notebook lives in RAM — the short-term memory of the computer, the same memory your programs already use to think — instead of on a hard drive.

Second way: the pages are not just paper. Each page can be a different *kind* of thing. One page can be a single word. Another page can be a long list, like a to-do list, where you push things on the front and pop things off the back. Another page can be a leaderboard, where every entry has a score and the page automatically keeps them sorted. Another page can be a bag of unique tags. Another page can be a tiny database table with named columns. Another page can be a never-ending log that everyone can read from the middle of.

That magic notebook is Redis.

### Imagine a giant whiteboard at the front of the office

Here is another way to think about it. Pretend your application has many servers. They are sitting in a row, each running a copy of your code. They all need to know things — who is logged in, what is in the cart, what the current top scores are, how many requests this user has sent in the last minute.

Each server could keep its own notes, but then the servers would disagree. One server would think Alice is logged in. Another would think she is not. One would say the leaderboard shows Bob in first place. Another would say it shows Carol.

So you put a giant whiteboard at the front of the office. Every server walks up and writes to the same whiteboard, reads from the same whiteboard. Now they all agree. Whatever is on the board is the truth.

The whiteboard has to be very fast (otherwise every server is waiting in line). The whiteboard has to be very organized (because you can't have everybody scribbling in the middle of each other). The whiteboard has to remember things even if someone bumps into it.

That whiteboard is Redis.

### Imagine a notebook that forgets on purpose

Here is one more picture. Pretend you have a notebook where every page can be marked with a self-destruct timer. You write "Alice is logged in" on a page named `session:alice` and you tell the notebook, "tear this page out in 30 minutes." You walk away. You don't have to do anything. Thirty minutes later, the page is gone. The notebook tore it out by itself. Now if anyone reads `session:alice` they get nothing — meaning Alice's session expired.

This sounds small but it is enormous. It means you can use Redis to remember things *for a while* without ever cleaning up. The notebook cleans up for you. This one trick — TTL, time-to-live — is half the reason people use Redis.

That self-destructing notebook is Redis.

### Why so many pictures?

Because Redis is many things at once. It is a key-value store, like a notebook. It is a shared cache, like a whiteboard. It is a session store, like the self-destructing pages. It is a queue, a leaderboard, a counter, a lock, a broadcast bus. People build whole product features out of single Redis commands. The pictures in your head help you decide *which* Redis you are using right now.

If you are caching the result of a slow database query, the *whiteboard* picture fits — every server reads the same cached answer.

If you are storing user sessions, the *self-destructing notebook* picture fits — every session has a timer.

If you are running a leaderboard or a queue, the *magic notebook with weird pages* picture fits — the page itself is a sorted set or a list.

Whichever picture clicks, keep it in your head.

## What Even Is Redis

Redis is a single program called `redis-server`. It listens on a TCP port (default `6379`). Clients connect with TCP. They send commands. The server holds all the data in RAM. The server replies. That is it.

The name "Redis" stands for **RE**mote **DI**ctionary **S**erver. A dictionary in programming is a thing that maps a *key* (a name) to a *value* (some data). Redis is a remote one — across the network — and a very fancy one — the values can be much more than just strings.

Redis was created by Salvatore Sanfilippo (handle: antirez) in 2009 as a side project to make a website's analytics page faster. It became wildly popular. Today it is the most-used in-memory data store in the world. It is written in C. The whole core of Redis is around 100,000 lines of C. You can read it in a long weekend.

In 2024 Redis Inc. changed the license of the project from BSD to RSALv2/SSPL — a "source-available" license that restricts commercial use. The community immediately forked the last open-source version under the name **Valkey**, which is now developed under the Linux Foundation with a BSD license. When this sheet says "Redis" it means the protocol, data model, and commands — which Valkey, KeyDB, Dragonfly, and others all implement. If you are starting a new project today, Valkey is the same thing without the licensing risk.

### Single binary, no dependencies

You can install Redis with one command:

```bash
$ brew install redis        # macOS
$ apt install redis-server  # Debian/Ubuntu
$ dnf install redis         # Fedora/RHEL
```

You start it with one command:

```bash
$ redis-server
```

You talk to it with one command:

```bash
$ redis-cli
127.0.0.1:6379> PING
PONG
```

That is the whole world. One server, one client, one TCP port. No schema. No setup. No migrations. You connect and you start writing pages.

## In-Memory vs On-Disk

This is the single most important idea about Redis, so we are going to dwell on it.

A traditional database (Postgres, MySQL, SQLite) keeps its data on the **disk** — the long-term storage. The disk is enormous (terabytes are cheap) but slow. Reading one record from a spinning hard drive can take ten thousand microseconds. Reading from an SSD is faster, maybe a hundred microseconds. Both are forever compared to RAM.

RAM (random access memory) is small (sixty-four gigabytes is a lot for a server) but fast. Reading one record from RAM takes about a tenth of a microsecond. RAM is roughly a thousand times faster than SSD and a hundred thousand times faster than spinning disk.

Redis keeps the *whole* dataset in RAM, all of it, all the time. Every key. Every value. When you ask for a key, Redis does a hash table lookup in RAM and replies. There is no disk read in the hot path. There is no query plan. There is no "warm up the cache." The whole thing is the cache.

That is why Redis is so fast. It is not a clever algorithm. It is not a special chip. It is just RAM.

### What about losing the data when the power goes out?

This is the real question. RAM is volatile. The instant the server reboots, RAM is wiped. If Redis only ever lived in RAM, every reboot would lose everything.

Redis solves this with **persistence**: it writes the data to disk too, but only as a *backup*, never on the read path.

There are two persistence modes (you can run both at the same time):

- **RDB snapshots.** Every so often, Redis takes a photograph of the whole notebook and writes it to a file called `dump.rdb`. If the server reboots, it loads the photograph back into RAM and keeps going. You lose anything written between the last snapshot and the crash.

- **AOF (Append-Only File).** Every write command Redis receives is also appended to a log file (`appendonly.aof`). If the server reboots, Redis replays the log and reconstructs the dataset. You lose at most one second of writes (or zero, depending on `fsync` policy).

Reads never touch disk. Writes go to RAM first and then *also* go to disk in the background. The application sees the speed of RAM, but the data survives reboots.

We will go deep on persistence later. The point now: Redis is fast because reads come from RAM; Redis is durable because writes also drip out to disk.

## The Single-Threaded Model (until Redis 6 I/O threads)

Here is a thing that surprises people: Redis processes commands one at a time, on a single thread. Yes — even though your server has 16 cores, Redis uses one. That sounds insane. Let's see why it works and why it changed in Redis 6.

### One pizza chef in a tiny kitchen

Imagine a pizza shop with one chef. Orders come in. The chef makes one pizza, then the next, then the next. There is no second chef in the kitchen. There are no two chefs reaching for the same dough at the same time. There is no "wait, who has the cheese?" There is no fight. The chef is fast — wildly fast — and orders are simple — slap dough, sauce, cheese, into oven — so the line moves.

This is Redis. Each command is so cheap (a hash lookup, a list push, a counter increment) that doing them one after another is faster than coordinating multiple threads. Threads need locks. Locks need waiting. Waiting is slow. By having one thread do everything, Redis avoids all the synchronization cost.

A single Redis instance handles roughly a million simple commands per second on a modern CPU. That is more than most applications need from a database. If you need more, you shard (split the keys across many Redis instances) — but you don't usually need to.

### What about slow commands?

The catch: if one command is slow, *every other command is blocked behind it*. There is one chef. If the chef has to roll out a giant pizza for thirty seconds, every other customer waits thirty seconds.

That is why some commands are dangerous in production:

- `KEYS *` — scans the entire keyspace. Forever. Blocks the server.
- `SMEMBERS huge_set` — reads a million-element set in one go.
- `LRANGE big_list 0 -1` — reads a million-element list.
- `DEBUG SLEEP 5` — explicitly sleeps for 5 seconds, blocking everything.
- A bad Lua script that loops without yielding.

These are called "big-key" or "slow-command" problems and they are the #1 cause of latency spikes in Redis. We will come back to this.

### Redis 6 added I/O threads

In Redis 6 (2020), the team added optional I/O threads. The trick: command *execution* is still single-threaded (still one chef), but the work of *reading bytes off the network and writing bytes back* is split across multiple threads. The chef still cooks alone, but a team of waiters takes orders and serves food.

You enable it with two config settings:

```conf
io-threads 4
io-threads-do-reads yes
```

This helps when network I/O is the bottleneck (lots of clients, lots of small commands). It does not change the fundamental model. The data is still touched by exactly one thread.

```
        Redis Single-Threaded Event Loop
        ================================

   clients ──────►  ┌─────────────────────────┐
                    │  epoll / kqueue / event  │
                    │       multiplexer       │
                    └────────────┬────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │   command dispatch      │
                    │  (one command at a time)│
                    └────────────┬────────────┘
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
                  GET          ZADD          XADD
                    │            │            │
                    └────────────┼────────────┘
                                 ▼
                       reply written to socket
                                 │
                                 ▼
                          back to event loop
```

That picture is the *whole* server. One loop. One command at a time. Microseconds per pass.

## Data Structures

This is where Redis gets fun. The "value" attached to a key is not just a string. It can be one of nine different kinds of structure, and each one has its own commands.

### Strings

The simplest. The page is just a chunk of bytes — text, a number, a serialized blob, an image, anything up to 512 MB.

```
> SET greeting "hello"
OK
> GET greeting
"hello"
> APPEND greeting ", world"
(integer) 12
> GET greeting
"hello, world"
> STRLEN greeting
(integer) 12
```

If the string looks like an integer, you can use atomic counter commands:

```
> SET counter 0
OK
> INCR counter
(integer) 1
> INCR counter
(integer) 2
> INCRBY counter 10
(integer) 12
> DECR counter
(integer) 11
> INCRBYFLOAT counter 0.5
"11.5"
```

`INCR` is atomic — if a hundred clients all call `INCR counter`, the result is the count, not a race condition. This single fact is why Redis is so often used for rate-limiting and counting.

### Lists

A linked list of strings, with fast push/pop on both ends. Think: a to-do list, a queue, a feed.

```
> RPUSH tasks "wash dishes"
(integer) 1
> RPUSH tasks "fold laundry"
(integer) 2
> LPUSH tasks "wake up"
(integer) 3
> LRANGE tasks 0 -1
1) "wake up"
2) "wash dishes"
3) "fold laundry"
> LPOP tasks
"wake up"
> LLEN tasks
(integer) 2
```

`LPUSH` and `RPUSH` add to the left or right end. `LPOP` and `RPOP` remove. Combined, you have a queue (`LPUSH` + `RPOP`) or a stack (`LPUSH` + `LPOP`).

The killer feature: **blocking pops**. `BRPOP` waits for an element to appear, blocking the client (with a timeout). This makes Redis lists a real, distributed, low-latency work queue:

```
> BRPOP tasks 5
1) "tasks"
2) "fold laundry"
```

If `tasks` is empty, the client waits up to 5 seconds. If something appears, the client returns immediately with the value.

In Redis 6.2, `LMPOP` and `BLMPOP` were added — pop from the first non-empty list among many.

### Sets

An unordered bag of unique strings. Think: tags, group membership, deduplication.

```
> SADD tags:post:42 "redis" "database" "tutorial"
(integer) 3
> SADD tags:post:42 "redis"
(integer) 0           # already there, no-op
> SMEMBERS tags:post:42
1) "redis"
2) "tutorial"
3) "database"
> SISMEMBER tags:post:42 "redis"
(integer) 1
> SCARD tags:post:42
(integer) 3
```

The interesting commands are set algebra:

```
> SADD set:a 1 2 3 4
(integer) 4
> SADD set:b 3 4 5 6
(integer) 4
> SINTER set:a set:b      # intersection
1) "3"
2) "4"
> SUNION set:a set:b      # union
1) "1" 2) "2" 3) "3" 4) "4" 5) "5" 6) "6"
> SDIFF set:a set:b       # in a but not b
1) "1"
2) "2"
```

Useful for "users who follow both A and B," "tags on either post," "items in cart but not yet purchased."

### Sorted Sets (ZSET)

This is the marquee data structure. A set, but every element has a floating-point **score**, and the set is automatically kept sorted by score. Think: leaderboards, priority queues, time-ordered feeds.

```
> ZADD leaderboard 100 alice
(integer) 1
> ZADD leaderboard 250 bob
(integer) 1
> ZADD leaderboard 175 carol
(integer) 1
> ZRANGE leaderboard 0 -1 WITHSCORES
1) "alice"
2) "100"
3) "carol"
4) "175"
5) "bob"
6) "250"
> ZREVRANGE leaderboard 0 2 WITHSCORES   # top 3 highest
1) "bob"   2) "250"
3) "carol" 4) "175"
5) "alice" 6) "100"
> ZINCRBY leaderboard 50 alice
"150"
> ZRANK leaderboard alice
(integer) 1                   # 0-indexed from low to high
> ZREVRANK leaderboard alice
(integer) 1                   # rank from high
> ZRANGEBYSCORE leaderboard 100 200
1) "alice"
2) "carol"
```

Internally a sorted set is a hash table (for lookups by member) plus a **skiplist** (for ordered traversal). Both insert and lookup are O(log N). For small sorted sets Redis uses a more compact `listpack` encoding.

This structure carries an enormous fraction of Redis use cases: any time you need "give me the top N" or "give me everything between time X and time Y," it is a sorted set with the score being the rank or the timestamp.

### Hashes

A page that contains a small dictionary — named fields with values. Think: a row in a table.

```
> HSET user:1 name alice age 30 email alice@example.com
(integer) 3
> HGET user:1 name
"alice"
> HGETALL user:1
1) "name"
2) "alice"
3) "age"
4) "30"
5) "email"
6) "alice@example.com"
> HINCRBY user:1 age 1
(integer) 31
> HDEL user:1 email
(integer) 1
```

Useful when you have a record-shaped value and you want to update one field without rewriting the whole thing. More memory-efficient than storing a JSON blob in a string.

### Streams

A log. Append-only, time-ordered, with consumer groups. Think: Kafka but simpler, in-process.

```
> XADD events * type login user alice
"1714233600000-0"
> XADD events * type purchase user alice item book
"1714233601000-0"
> XLEN events
(integer) 2
> XRANGE events - +
1) 1) "1714233600000-0"
   2) 1) "type"
      2) "login"
      3) "user"
      4) "alice"
2) 1) "1714233601000-0"
   2) ...
```

The `*` means "Redis, pick the ID for me using the current millisecond timestamp." You get back an ID like `1714233600000-0` — the millisecond plus a sequence number for entries within the same millisecond.

Streams add **consumer groups** — multiple workers reading from the same stream, each getting a different slice of the load:

```
> XGROUP CREATE events workers $ MKSTREAM
OK
> XREADGROUP GROUP workers worker-1 COUNT 10 BLOCK 5000 STREAMS events >
```

Streams replaced "Redis as a Kafka" — they are durable, time-ordered, support replay, and integrate with Redis's other data structures.

### Bitmaps

Strings used as a bit array. Think: presence bitmaps, feature flags per user.

```
> SETBIT users:online 42 1
(integer) 0
> GETBIT users:online 42
(integer) 1
> SETBIT users:online 99 1
(integer) 0
> BITCOUNT users:online
(integer) 2                   # number of users currently online
```

A bitmap of 100 million users is 12.5 MB. Counting them with `BITCOUNT` is microseconds. People build daily-active-user analytics this way: one bitmap per day, set the bit on every login, count bits at end of day.

### HyperLogLog

A probabilistic counter for cardinality (how many *distinct* things). Uses 12 KB regardless of the cardinality, with about 0.81% error.

```
> PFADD visitors alice bob carol
(integer) 1
> PFADD visitors alice
(integer) 0
> PFCOUNT visitors
(integer) 3
> PFADD visitors:other dave eve
(integer) 1
> PFMERGE all visitors visitors:other
OK
> PFCOUNT all
(integer) 5
```

For analytics: "how many unique visitors today" without storing the visitor list.

### Geospatial

Coordinates on a globe. Stored internally as sorted sets of geohashes.

```
> GEOADD places -122.41 37.77 "san_francisco"
(integer) 1
> GEOADD places -73.99 40.75 "new_york"
(integer) 1
> GEODIST places san_francisco new_york km
"4128.7977"
> GEOSEARCH places FROMMEMBER san_francisco BYRADIUS 100 km ASC
1) "san_francisco"
```

Useful for "find nearby." Not a full geo-database, but enough for cache-layer location queries.

### Encoding visualization

Different encodings depending on the size:

```
        Sorted Set Internal Encoding
        ============================

    if size < 128 entries AND each value < 64 bytes:
       ┌──────────────────────────────────┐
       │            listpack              │   compact, scan-based, no skiplist
       └──────────────────────────────────┘
    else:
       ┌────────────┐    ┌─────────────────┐
       │  hashtable │ ──►│    skiplist     │   O(1) lookup + O(log N) order
       └────────────┘    └─────────────────┘

        Hash Internal Encoding
        ======================

    if size < 128 fields AND each field/value < 64 bytes:
       listpack
    else:
       hashtable
```

The thresholds are tunable in `redis.conf` (`hash-max-listpack-entries`, `zset-max-listpack-entries`). You don't usually care, but knowing they exist explains why your `MEMORY USAGE` numbers jump when a hash crosses the threshold.

## Common Use Cases

### Cache

The most common. Sit Redis in front of a slow database. Application reads check Redis first; on miss, fall through to the database, then `SET` the result with a TTL.

```python
def get_user(user_id):
    cached = redis.get(f"user:{user_id}")
    if cached:
        return json.loads(cached)
    user = db.query("SELECT * FROM users WHERE id = ?", user_id)
    redis.setex(f"user:{user_id}", 3600, json.dumps(user))
    return user
```

The TTL means stale entries clean themselves up. The cache shrinks naturally.

### Session store

Sessions are perfect Redis fodder: small, hot, time-bounded.

```
> SET session:abc123 "{user_id: 42, csrf: 'xyz'}" EX 1800
OK
```

`EX 1800` means "expire in 1800 seconds (30 minutes)." Every request bumps the TTL by re-setting it. Sign out: `DEL session:abc123`.

### Leaderboard

A sorted set with player scores.

```
> ZINCRBY leaderboard 100 alice
"100"
> ZREVRANGE leaderboard 0 9 WITHSCORES   # top 10
```

Scales to billions of entries, microseconds per query.

### Rate limiter

Per-user counter with a TTL.

```
> INCR ratelimit:alice
(integer) 1
> EXPIRE ratelimit:alice 60      # only on first INCR
(integer) 1
```

If the counter exceeds the limit, reject the request. The classic "fixed window" algorithm. Sliding-window variants use sorted sets where the score is the timestamp.

### Queue

A list as a worker queue.

```python
# producer
redis.lpush("jobs", json.dumps(job))

# worker
while True:
    _, job = redis.brpop("jobs", timeout=0)
    process(json.loads(job))
```

`BRPOP` blocks the worker until work arrives. No polling. No CPU waste.

### Distributed lock

The infamous Redlock pattern. Use `SET` with `NX` and `PX` to acquire a lock atomically:

```
> SET lock:resource123 "$random_token" NX PX 30000
OK
```

`NX` = "only set if it doesn't exist." `PX 30000` = "expire in 30 seconds." If you got `OK`, you hold the lock. To release, run a Lua script that checks the token before deleting (so you don't release someone else's lock).

This is a useful primitive but read antirez's "Redlock" article and Martin Kleppmann's response before trusting it for safety-critical work — Redis locks are *advisory*, not bulletproof.

### Pub/Sub

Broadcast messages to anyone listening on a channel.

```
# subscriber
> SUBSCRIBE news
Reading messages... (press Ctrl-C to quit)

# publisher (different terminal)
> PUBLISH news "hello everyone"
(integer) 1
```

The `1` is "one subscriber received the message." If no one is listening, the message vanishes — pub/sub is fire-and-forget. For durable broadcast use Streams.

### Time-series via Streams

Streams plus the `XADD MAXLEN` cap give you a time-bounded log of events.

```
> XADD metrics:cpu MAXLEN 10000 * value 0.42
"1714233600000-0"
```

Cap the stream to the last 10,000 entries. Old entries are evicted automatically. For real time-series at scale use the `RedisTimeSeries` module.

### Full-text search via RediSearch

A module (not in core) that adds inverted indexes and query language. Index documents stored in hashes or JSON; query with `FT.SEARCH`. We will mention modules below.

## Persistence

This is the most important config knob in Redis. How much can you afford to lose?

### RDB snapshots

Periodically, Redis forks the process and dumps the entire dataset to a binary file (`dump.rdb`). The fork uses **copy-on-write** so the parent (which keeps serving requests) and the child (which writes to disk) share memory pages until something is written, at which point the OS makes a copy.

Configured in `redis.conf`:

```conf
save 3600 1       # snapshot if at least 1 key changed in 3600 sec
save 300 100      # snapshot if at least 100 keys changed in 300 sec
save 60 10000     # snapshot if at least 10000 keys changed in 60 sec
```

The snapshot is a single file you can rsync to S3, archive nightly, or use as a backup.

Pros: tiny on disk, fast to load on restart, perfect for backups.

Cons: you lose everything between snapshots. If the server crashes 10 minutes after the last snapshot, those 10 minutes are gone.

```
        RDB Snapshot via fork()
        =======================

   parent (still serving requests)
        │
        │  fork()
        ▼
   ┌──────────────┐         ┌──────────────┐
   │   parent     │         │    child     │
   │              │ COW──►  │              │
   │ shared pages │ ◄──COW  │ shared pages │
   └──────┬───────┘         └──────┬───────┘
          │                        │
          │                        ▼
          │                 write dump.rdb
          │                        │
          │                        ▼
          │                  exit cleanly
          ▼
    keep serving
```

Copy-on-write means a 100 GB dataset doesn't need 200 GB of RAM during the dump — only the pages that the parent modifies during the dump are copied.

### AOF (Append-Only File)

Every write command is appended to `appendonly.aof` as a RESP-formatted log. On restart, Redis replays the log to rebuild the dataset.

Configured with `appendonly yes` and `appendfsync`:

```conf
appendonly yes
appendfsync everysec    # fsync the log to disk once per second
# alternatives:
# appendfsync always   — fsync after every command (slow but durable)
# appendfsync no       — let the OS decide (fast but lossy)
```

`everysec` is the default and usually right — at most one second of writes lost on a crash, with minimal performance impact.

Pros: durable down to one second; replays the actual command history.

Cons: the file grows forever as you write. So Redis periodically rewrites it — `BGREWRITEAOF` — by reading the current dataset and emitting the *minimum* set of commands that would produce it. The new file replaces the old. This is also forked.

### Hybrid AOF + RDB

Modern Redis (>= 4.0) supports `aof-use-rdb-preamble yes` (the default since 5.0). The AOF file starts with an RDB binary preamble (a snapshot of the data at the time of the rewrite) and then appends commands after that. This gives you both: fast startup (from the RDB preamble) and durability down to a second (from the AOF tail).

Most production deployments run AOF with `appendfsync everysec` and `aof-use-rdb-preamble yes`. That is the usual recommended setup.

```
        AOF Rewrite Triggered
        =====================

      old AOF (huge)              new AOF (compact)
   ┌────────────────────┐      ┌────────────────────┐
   │ SET k1 v1          │      │ RDB preamble       │
   │ INCR k2            │      │  (snapshot of      │
   │ SET k1 v2          │      │   current state)   │
   │ DEL k3             │      │                    │
   │ SET k1 v3          │      │  ── then ──        │
   │ HSET h f1 a        │      │ (commands after    │
   │ HSET h f1 b        │      │   the snapshot)    │
   │ ...                │      └────────────────────┘
   └────────────────────┘
        │                                ▲
        │  BGREWRITEAOF (forks)          │
        └────────────────────────────────┘
```

## Replication

Redis supports primary–replica replication. One server is the primary; one or more replicas connect and receive a stream of writes.

You enable it on the replica side:

```
> REPLICAOF 192.168.1.10 6379
OK
```

(In older Redis versions this was `SLAVEOF`. Both names still work; `REPLICAOF` is the modern alias.)

Replication is **asynchronous by default**. The primary doesn't wait for replicas to acknowledge writes. This means a primary can lose recently written data if it crashes before replicas catch up.

When a replica connects:

1. It tries a **partial sync** — "I'm at offset 12345, send me everything from there." This works if the primary still has the relevant chunk in its `repl-backlog-size` (an in-memory ring buffer of recent writes; default 1 MB, tune up for busy servers).
2. If the offset is gone, it falls back to a **full sync** — the primary forks, dumps an RDB, ships it to the replica, and then streams new writes after.

Replicas serve reads (good for scaling read traffic). Writes go only to the primary (otherwise replication breaks). If you `SET` against a replica:

```
(error) READONLY You can't write against a read only replica.
```

## Sentinel

Replication alone doesn't give you automatic failover. If the primary dies, no replica becomes the new primary on its own. **Sentinel** is the failover system.

Sentinel is a separate process (`redis-sentinel`). You run three or more sentinels (odd number for quorum). They watch the primary, gossip among themselves, and if a quorum agrees the primary is down, they elect one of the replicas to be the new primary and tell clients about it.

```conf
# sentinel.conf
sentinel monitor mymaster 192.168.1.10 6379 2
sentinel down-after-milliseconds mymaster 5000
sentinel failover-timeout mymaster 60000
```

Quorum of 2 means 2 out of 3 sentinels must agree. `down-after 5000` means a primary unreachable for 5 seconds is considered down.

Sentinel works for non-clustered Redis (single primary, several replicas). For sharded multi-primary setups, use Cluster.

```
        Sentinel Failover Decision Tree
        ===============================

       primary unreachable?
              │
       ┌──────┴───────┐
       no             yes
       │              │
   keep going    quorum sentinels agree?
                      │
                ┌─────┴─────┐
                no          yes
                │           │
            wait/retry   elect new primary from replicas
                              │
                              ▼
                     promote replica → REPLICAOF NO ONE
                              │
                              ▼
                  reconfigure other replicas to follow new primary
                              │
                              ▼
                    notify clients via pub/sub
```

## Cluster

For datasets larger than one server's RAM, Redis Cluster shards keys across multiple primaries.

The keyspace is divided into **16,384 hash slots**. Each key hashes (CRC16 mod 16384) to one slot. Each primary owns a range of slots. Add a node, slots get redistributed.

Cluster topology is gossiped between nodes — every node knows which slots every other node owns. When a client sends a command for a key whose slot lives on a different node, the node replies:

```
(error) MOVED 12182 192.168.1.10:6379
```

A "smart" client library handles this transparently — re-issues the command to the correct node and updates its slot cache. A "dumb" client requires the application to follow the redirect manually.

Special case: during slot migration, a node can reply `(error) ASK ip:port`, meaning "I don't have this slot anymore but you should ask the new owner *just for this one command*." Slightly different from `MOVED` — `ASK` is one-time, `MOVED` is permanent.

```
        Cluster Hash Slot Distribution
        ==============================

       16,384 slots total
   ┌─────────────────────────────────────────┐
   │ slot 0 ──── slot 5460 ──── slot 10922 ──── slot 16383 │
   └─────┬──────────────┬─────────────────┬────────┘
         │              │                 │
         ▼              ▼                 ▼
      node A          node B            node C
   slots 0..5460   5461..10922     10923..16383
       │ replica       │ replica         │ replica
       ▼ A'            ▼ B'              ▼ C'

    client → CRC16("user:42") % 16384 = 12182
    12182 lives on node C → send command to C
    if client guessed wrong: "MOVED 12182 nodeC-ip:port"
```

Cluster has no central coordinator. Nodes failover their own replicas via gossip — similar logic to Sentinel but built in. If half or more of the primaries are unreachable and have no available replica, the cluster goes into `CLUSTERDOWN` state.

You also pay a price: not all commands work across slots. `MGET k1 k2 k3` only works if all three keys are in the same slot. Use **hash tags** (`{tag}`) to force keys into the same slot:

```
> SET {user:42}:name alice
OK
> SET {user:42}:cart "{...}"
OK
> MGET {user:42}:name {user:42}:cart   # both hash on "user:42", same slot
```

## Eviction Policies

What happens when Redis hits `maxmemory`? You configure that.

```conf
maxmemory 4gb
maxmemory-policy allkeys-lru
```

Policies:

- `noeviction` — refuse writes with OOM error (good for primary databases where data loss is unacceptable).
- `allkeys-lru` — evict any key, least recently used first (good for cache).
- `volatile-lru` — only evict keys with a TTL set, LRU among them.
- `allkeys-lfu` — least frequently used; better than LRU for skewed access patterns (Redis 4+).
- `volatile-lfu` — LFU among keys with TTL.
- `allkeys-random` — pick a random key.
- `volatile-random` — random pick from keys with TTL.
- `volatile-ttl` — evict the key with the shortest remaining TTL.

For a pure cache: `allkeys-lru` or `allkeys-lfu`. For a primary store with critical data: `noeviction` (and monitor memory like a hawk).

## Transactions

Redis transactions are not what database transactions are. They are a way to *queue commands* and execute them atomically, in order, with no other client's commands interleaved.

```
> MULTI
OK
> SET balance:alice 90
QUEUED
> SET balance:bob 110
QUEUED
> EXEC
1) OK
2) OK
```

Between `MULTI` and `EXEC`, commands are queued — *not executed*. `EXEC` runs them all, atomically, in order. `DISCARD` throws away the queue.

The catch: there is no rollback. If the second command in the transaction errors at runtime, the first still happened. Redis transactions guarantee *atomic execution*, not *atomic semantics*.

`WATCH` adds optimistic locking. Watch a key; if the key changes between `WATCH` and `EXEC`, `EXEC` fails (returns nil) and the application retries.

```
> WATCH balance:alice
OK
> GET balance:alice
"100"
> MULTI
OK
> SET balance:alice 90
QUEUED
> EXEC                    # if someone else SET balance:alice in between, returns nil
```

This gives you a check-and-set primitive for safe updates without locking.

## Lua Scripting

`EVAL` runs a Lua script atomically on the server side. The whole script blocks the event loop — no other commands run until the script finishes.

```
> EVAL "return redis.call('GET', KEYS[1])" 1 mykey
"hello"
> EVAL "redis.call('SET', KEYS[1], ARGV[1]); return redis.call('GET', KEYS[1])" 1 mykey "world"
"world"
```

`KEYS` and `ARGV` are the conventional way to pass keys and arguments. The first number after the script is `numkeys` — how many of the following arguments are keys.

Compile and cache scripts with `SCRIPT LOAD`:

```
> SCRIPT LOAD "return redis.call('GET', KEYS[1])"
"a5260dd66ce02462c5b5231c727b3f7772c0bcc5"
> EVALSHA a5260dd66ce02462c5b5231c727b3f7772c0bcc5 1 mykey
"hello"
```

Use Lua for atomic multi-step ops: rate-limiting algorithms, conditional updates, custom data structures. Watch out for slow scripts — they block everyone.

Redis 7 introduced **Functions** (`FUNCTION LOAD`) — a more permanent, modular replacement for `SCRIPT LOAD`. Functions persist across restarts and can be organized into libraries.

## RESP Protocol

Redis speaks a simple text-and-binary protocol called **RESP** (REdis Serialization Protocol). It is so simple you can drive Redis with `nc` (netcat).

A `SET foo bar` command on the wire is:

```
*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
```

That is:

- `*3\r\n` — array of 3 elements
- `$3\r\nSET\r\n` — bulk string of length 3, contents `SET`
- `$3\r\nfoo\r\n` — bulk string `foo`
- `$3\r\nbar\r\n` — bulk string `bar`

The reply is `+OK\r\n` (a simple string, prefix `+`).

The whole protocol has five types: simple strings (`+`), errors (`-`), integers (`:`), bulk strings (`$`), and arrays (`*`). RESP2 has been the standard for years.

**RESP3** (Redis 6+) adds richer types: maps, sets, doubles, booleans, big numbers, push messages. Clients negotiate with `HELLO 3` after connecting. Most modern client libraries opt in.

For client library authors this matters a lot. For application developers, you mostly never see RESP — your library hides it.

## Memory Management

Redis uses **jemalloc** as its allocator (linked in by default). Jemalloc reduces fragmentation under highly variable allocation patterns, which is exactly what Redis has.

The `INFO memory` section is the cockpit:

```
> INFO memory
used_memory:104857600
used_memory_human:100.00M
used_memory_rss:120586240
used_memory_peak:135266304
maxmemory:4294967296
maxmemory_human:4.00G
mem_fragmentation_ratio:1.15
```

Key numbers:

- `used_memory` — bytes Redis thinks it is using (allocated to data structures).
- `used_memory_rss` — bytes the OS thinks the process is using (resident set size).
- `mem_fragmentation_ratio` — `rss / used_memory`. Greater than 1 = fragmentation. Around 1.0–1.5 is normal. Greater than 1.5 = real fragmentation; consider `MEMORY PURGE` or restart.

Less than 1.0 = some memory is paged out to disk. That is bad. Increase RAM or fix your config.

`MEMORY USAGE key` tells you how big a single key is. `MEMORY STATS` is a kitchen-sink report. `MEMORY DOCTOR` gives a human-readable summary of any problems.

## Slow Log

Redis remembers any command that took longer than `slowlog-log-slower-than` microseconds (default 10000 = 10ms).

```
> SLOWLOG GET 10
1) 1) (integer) 14         # entry id
   2) (integer) 1714233600 # unix timestamp
   3) (integer) 12345      # microseconds
   4) 1) "KEYS"            # the command
      2) "*"
   5) "127.0.0.1:51234"    # client
   6) ""
> SLOWLOG LEN
(integer) 27
> SLOWLOG RESET
OK
```

The slow log is your first stop when "Redis got slow." Open it. Look for `KEYS *`. Look for `LRANGE huge 0 -1`. Look for big-key commands.

## Latency Diagnostics

```
> LATENCY DOCTOR
Dave, I have observed the system for some time and I am ready to
report what I noticed:

(...) redis-server can offer fast access only when the system has good
real-time performance.
```

Redis monitors its own latency. Set a threshold:

```conf
latency-monitor-threshold 100   # log events that exceed 100 ms
```

Then:

```
> LATENCY HISTORY event-name
> LATENCY GRAPH event-name
> LATENCY RESET
```

Common events: `aof-fsync-always`, `expire-cycle`, `eviction-cycle`, `fork`. The fork during RDB or AOF rewrite is a frequent culprit on big datasets.

## Modern Modules

Redis core is small. Modules extend it with new commands and types.

- **RediSearch** — secondary indexes, full-text search, vector search. `FT.CREATE`, `FT.SEARCH`, `FT.AGGREGATE`.
- **RedisJSON** — JSON values as a native type. `JSON.SET`, `JSON.GET`, `JSON.ARRAPPEND`.
- **RedisGraph** — property graph database (deprecated as of Redis 7.2; use a real graph DB).
- **RedisTimeSeries** — time-series with downsampling, compaction, range queries.
- **RedisBloom** — probabilistic data structures: Bloom filters, Cuckoo filters, Top-K, T-Digest.
- **RedisAI** — model serving (deprecated; superseded by RedisGears and external infrastructure).

The bundle is shipped as **RedisStack** — Redis core plus the modules, one container, batteries included. Great for development; for production you can run modules a la carte.

Forks and alternatives:

- **KeyDB** — Redis fork with multithreaded execution.
- **Dragonfly** — clean-room reimplementation in C++ with a modern multi-shared-nothing design.
- **Valkey** — the BSD-licensed Linux Foundation fork after the 2024 license change.

## Common Errors

The verbatim text of errors you will see and what they mean.

```
(error) NOAUTH Authentication required.
```

You connected without authenticating. Run `AUTH password` (or `AUTH user password` with ACLs).

```
(error) WRONGTYPE Operation against a key holding the wrong kind of value
```

You ran a list command on a string, or a hash command on a list, etc. Check the key's `TYPE`.

```
(error) READONLY You can't write against a read only replica.
```

You connected to a replica and tried to write. Connect to the primary instead, or use Sentinel/Cluster client routing.

```
(error) MISCONF Redis is configured to save RDB snapshots, but it is currently not able to persist on disk. Commands that may modify the data set are disabled.
```

A snapshot failed (disk full, permissions). Either fix the disk or set `stop-writes-on-bgsave-error no`.

```
(error) OOM command not allowed when used memory > 'maxmemory'.
```

You hit the memory limit and the policy is `noeviction`. Either evict, increase memory, or change the policy.

```
(error) MOVED 12182 192.168.1.10:6379
```

Cluster — this slot is on another node. Use a cluster-aware client.

```
(error) BUSY Redis is busy running a script. You can only call SCRIPT KILL or SHUTDOWN NOSAVE.
```

A Lua script is hogging the event loop. `SCRIPT KILL` if it's read-only; `SHUTDOWN NOSAVE` if it has written.

```
(error) LOADING Redis is loading the dataset in memory
```

Server just restarted, replaying RDB/AOF. Wait. Reads will start working as soon as it's done.

```
(error) CLUSTERDOWN Hash slot not served
```

Cluster lost coverage of some slots — a primary is down with no replica available.

```
(error) ERR max number of clients reached
```

Hit `maxclients` (default 10000). Raise it or fix the connection leak.

```
LATENCY DOCTOR
"redis-server can offer fast access only when the system has good real-time performance."
```

The catch-all "your kernel/disk/network is making me slow" message.

## Hands-On

Open one terminal and run `redis-server`. Open a second terminal for `redis-cli`.

```bash
# basic connectivity
$ redis-cli PING
PONG

# strings
$ redis-cli SET greeting "hello"
OK
$ redis-cli GET greeting
"hello"

# counters
$ redis-cli INCR counter
(integer) 1
$ redis-cli INCR counter
(integer) 2
$ redis-cli DECR counter
(integer) 1

# expiration
$ redis-cli EXPIRE counter 60
(integer) 1
$ redis-cli TTL counter
(integer) 58
$ redis-cli PERSIST counter
(integer) 1
$ redis-cli TTL counter
(integer) -1

# keyspace info
$ redis-cli TYPE counter
string
$ redis-cli EXISTS counter
(integer) 1
$ redis-cli DEL counter
(integer) 1

# DON'T use KEYS in production — blocks the server
$ redis-cli KEYS '*'
(empty array)

# DO use SCAN — incremental, non-blocking
$ redis-cli SCAN 0 MATCH 'user:*' COUNT 100
1) "0"
2) (empty array)

# lists
$ redis-cli RPUSH tasks "wash" "fold"
(integer) 2
$ redis-cli LPUSH tasks "wake"
(integer) 3
$ redis-cli LPOP tasks
"wake"
$ redis-cli RPOP tasks
"fold"

# blocking pop (waits up to 5 seconds for an element)
$ redis-cli BRPOP tasks 5
1) "tasks"
2) "wash"

# atomic move from one list to another (Redis 6.2+)
$ redis-cli BLMOVE src dst LEFT RIGHT 0

# sets
$ redis-cli SADD tags redis db cache
(integer) 3
$ redis-cli SMEMBERS tags
1) "redis"
2) "db"
3) "cache"
$ redis-cli SISMEMBER tags redis
(integer) 1
$ redis-cli SDIFF tags other-tags
$ redis-cli SINTER tags other-tags

# sorted sets
$ redis-cli ZADD leaderboard 100 alice 250 bob 175 carol
(integer) 3
$ redis-cli ZRANGE leaderboard 0 9 WITHSCORES
1) "alice"
2) "100"
3) "carol"
4) "175"
5) "bob"
6) "250"
$ redis-cli ZREVRANGEBYSCORE leaderboard +inf -inf LIMIT 0 3 WITHSCORES

# hashes
$ redis-cli HSET user:1 name alice age 30
(integer) 2
$ redis-cli HGET user:1 name
"alice"
$ redis-cli HGETALL user:1
1) "name"
2) "alice"
3) "age"
4) "30"

# streams
$ redis-cli XADD events '*' type login user alice
"1714233600000-0"
$ redis-cli XREAD COUNT 100 BLOCK 0 STREAMS events 0

# geospatial
$ redis-cli GEOADD places -122.41 37.77 "san_francisco"
(integer) 1
$ redis-cli GEORADIUS places -122.4 37.78 100 km

# transactions
$ redis-cli
> MULTI
OK
> SET balance:a 90
QUEUED
> SET balance:b 110
QUEUED
> EXEC
1) OK
2) OK

# Lua
$ redis-cli EVAL "return redis.call('GET', KEYS[1])" 1 greeting
"hello"

# replication info
$ redis-cli INFO replication

# clients
$ redis-cli CLIENT LIST

# config
$ redis-cli CONFIG GET maxmemory
$ redis-cli CONFIG SET maxmemory-policy allkeys-lru

# memory tools
$ redis-cli MEMORY USAGE greeting
(integer) 56
$ redis-cli MEMORY DOCTOR
$ redis-cli MEMORY STATS

# latency tools
$ redis-cli LATENCY DOCTOR
$ redis-cli LATENCY HISTORY event-loop

# slow log
$ redis-cli SLOWLOG GET 10
$ redis-cli SLOWLOG RESET

# debug
$ redis-cli DEBUG SLEEP 1.0
OK
$ redis-cli DEBUG OBJECT greeting

# MONITOR — DEBUG ONLY, prints every command, kills production
$ redis-cli MONITOR

# pub/sub (two terminals)
# terminal A:
$ redis-cli SUBSCRIBE news
# terminal B:
$ redis-cli PUBLISH news "hello"

# inspection helpers
$ redis-cli --bigkeys           # find the largest keys
$ redis-cli --hotkeys           # find the most-accessed keys (requires LFU)
$ redis-cli --latency           # continuous latency sampler
$ redis-cli --memstats          # memory snapshot

# benchmark
$ redis-benchmark -n 100000 -c 50
```

Type `MONITOR` once in dev to see what your application is actually doing — it is the most enlightening sixty seconds you will spend with Redis. Then never run `MONITOR` in production. It echoes every command on every connection back to your terminal, which can be tens of millions per second.

## Common Confusions

**KEYS vs SCAN.** `KEYS *` is O(N) over the entire keyspace and runs in the main thread, blocking everything else. With a million keys, it blocks for seconds. `SCAN 0 COUNT 100` walks the keyspace in chunks across many round-trips and never blocks for more than a few microseconds at a time. Use `SCAN`. Always. The fact that `KEYS` exists is a historical accident.

**RDB save points and what fork does.** `save 60 1000` does NOT mean "save every 60 seconds." It means "if at least 1000 keys changed in the last 60 seconds, save." The save itself is a `fork()` — a near-zero-cost OS operation that creates a child process sharing memory pages with the parent (copy-on-write). The child writes the snapshot. The parent keeps serving. If the parent modifies a page, the OS copies it. On a 100 GB instance with low write rate, the fork uses almost no extra RAM. On a high-write instance, the fork can balloon RAM by tens of GB during the snapshot. This is the #1 cause of mysterious memory spikes during persistence.

**AOF rewrite triggers.** Two settings: `auto-aof-rewrite-percentage` (default 100) and `auto-aof-rewrite-min-size` (default 64mb). Translation: rewrite when the AOF has at least doubled in size since the last rewrite, *and* is at least 64 MB. You can also trigger manually with `BGREWRITEAOF`.

**Why MONITOR is dangerous.** Every `MONITOR` connection adds the cost of formatting and sending *every command* over the wire. On a 100k-ops-per-second server, that is 100k extra writes per `MONITOR` client per second. Two `MONITOR` clients can double the workload. Keep `MONITOR` to development only, or run it for a few seconds at a time.

**What is a "big key" and why it matters.** A "big key" is a single key with a huge value — a list with a million elements, a hash with a hundred thousand fields, a sorted set with millions of entries. Operations on big keys (`SMEMBERS`, `HGETALL`, `LRANGE 0 -1`) take milliseconds to seconds and block the single-threaded server. Every other client waits. `redis-cli --bigkeys` finds them. The fix is usually to split the structure (`user:42:cart:0`, `user:42:cart:1`, ...) or use cursor-based commands (`HSCAN`, `SSCAN`, `ZSCAN`).

**When to pick Sentinel vs Cluster.** Sentinel: single-primary topology, high availability via automatic failover, dataset fits in one node's RAM. Cluster: dataset bigger than one node, want to shard, can tolerate the operational complexity of multi-primary. Most teams should start with Sentinel and only move to Cluster when they actually need to shard.

**Why a client gets MOVED in Cluster mode and how the client should handle it.** The cluster topology is gossiped; clients get an initial map but can be out of date if slots have moved. When a client sends a command for a slot the server doesn't own, the server returns `MOVED slot ip:port`. A smart client immediately re-issues the command to the new node and updates its slot cache. A dumb client surfaces the error to the application. Use a smart client.

**How WATCH-MULTI-EXEC differs from a real transaction.** Database transactions guarantee atomicity *and* allow rollback on error. Redis transactions guarantee atomicity (no other client interleaves) but **do not roll back on error**. If the third command in a `MULTI` errors at runtime, the first two have already executed. `WATCH` adds optimistic concurrency control: the transaction aborts (returns nil) if a watched key changed before `EXEC`. So Redis transactions are "ordered batch with optional check-and-set," not "ACID transaction."

**Expired keys vs evicted keys.** Expired keys are deleted because their TTL ran out. Eviction happens when memory hits `maxmemory` and the policy chooses victims to make room. Both result in keys disappearing, but for different reasons. `INFO stats` shows `expired_keys` and `evicted_keys` separately.

**Idle vs last-access in OBJECT IDLETIME.** `OBJECT IDLETIME key` returns seconds since the key was last *touched* by any read or write. With LFU enabled, you instead get `OBJECT FREQ` — a logarithmic frequency counter rather than a recency timestamp.

**Why your INFO memory shows fragmentation > 1.0.** Allocators give back pages in chunks; small free holes don't always get reused. A ratio of 1.0–1.5 is normal. Above 1.5, run `MEMORY PURGE` (forces jemalloc to release free pages back to the OS) or restart the instance during a maintenance window.

**The difference between TYPE STRING and a serialized object stored as a string.** Redis returns `TYPE "string"` for any key that holds bytes — whether you set it as `SET name alice` or `SET blob "{json: 'lots of stuff'}"`. From Redis's perspective they are the same. The serialization format is your application's problem.

**Why pipelining is faster than transactions for batch ops.** Pipelining sends many commands without waiting for replies, then reads all replies at the end — saving network round-trips. Transactions add atomicity guarantees but the wire cost is similar to pipelining. If you only need to bulk-write a thousand `SET`s, pipeline; you don't need `MULTI`/`EXEC`. Use transactions only when you need atomicity.

**RESP2 vs RESP3 difference for client libraries.** RESP3 adds typed maps, sets, doubles, big numbers, and push messages (out-of-band notifications used by client-side caching and keyspace notifications). RESP2 has only strings, integers, errors, and arrays — clients have to know that "array of name/value alternating" means "this is a hash" and reassemble the structure. RESP3 makes this explicit. As a client library author, RESP3 is much easier to parse cleanly.

**`DEL` vs `UNLINK`.** Both delete a key. `DEL` is synchronous — the memory is freed in the main thread, blocking on big keys. `UNLINK` removes the key from the namespace immediately and queues memory reclamation in a background thread. For deleting a 10-million-element set, always use `UNLINK`.

**Why `SUBSCRIBE` is not a queue.** Pub/sub is fire-and-forget. If no subscriber is connected when a message is published, the message is lost. Subscribers don't get history. For durable broadcast use Streams with consumer groups. For simple "tell everyone connected right now," pub/sub is fine.

## Vocabulary

| Term | Plain English |
|------|---------------|
| Redis | The remote dictionary server we are learning about. |
| RESP | The wire protocol Redis speaks — short for REdis Serialization Protocol. |
| RESP2 | The original RESP, used since Redis 1.0. |
| RESP3 | A richer protocol with typed maps, sets, push messages — opt-in via `HELLO 3`. |
| `redis-cli` | The command-line client. Talks RESP. |
| `redis-server` | The actual server binary. |
| `redis-sentinel` | The high-availability watcher process. Same binary, different mode. |
| redis-cluster | A multi-node Redis with sharded keyspace and automatic failover. |
| RDB | A binary point-in-time snapshot of the dataset on disk. |
| AOF | Append-Only File — a log of every write command for replay on restart. |
| append-only file | The same thing as AOF. |
| fsync policy | How often the AOF log gets flushed to disk: `always`, `everysec`, or `no`. |
| `bgsave` | Background save — fork and write an RDB without blocking. |
| `lastsave` | Unix timestamp of the last successful RDB save. |
| copy-on-write | OS trick where a fork shares pages until one process writes; only then is a copy made. |
| jemalloc | The memory allocator Redis uses; reduces fragmentation. |
| `mem_fragmentation_ratio` | RSS divided by used memory; >1 means fragmentation, <1 means swap. |
| `maxmemory` | The hard cap on how much RAM Redis will use. |
| `maxmemory-policy` | What to do when `maxmemory` is hit: evict or refuse. |
| LRU | Least Recently Used — pick the key not touched for the longest. |
| LFU | Least Frequently Used — pick the key accessed the least often. |
| `allkeys-lru` | Eviction policy: any key, LRU. |
| `volatile-lru` | Eviction policy: only keys with TTL, LRU among them. |
| `allkeys-lfu` | Eviction policy: any key, LFU. |
| `noeviction` | Don't evict; refuse writes when full. |
| key | The name of a value. The page header in our magic notebook. |
| value | The data attached to a key. |
| string | The simplest value type — a chunk of bytes. |
| integer | A string that looks like a number; usable with `INCR`/`DECR`. |
| bitmap | A string used as a bit array; `SETBIT`, `GETBIT`. |
| list | A linked list of strings with O(1) push/pop on both ends. |
| set | An unordered bag of unique strings. |
| sorted set | A set where each element has a score; ordered automatically. |
| ZSET | Short for sorted set. |
| score | The floating-point number attached to each element of a sorted set. |
| hash | A small dictionary value — named fields with values. |
| field | A named slot inside a hash. |
| stream | An append-only log of entries with auto-generated time-based IDs. |
| consumer group | A coordinated set of stream consumers that share the load. |
| consumer | One client inside a consumer group reading from a stream. |
| last-delivered-id | The point in a stream up to which a consumer group has delivered. |
| geospatial set | A sorted set encoded with geohashes for `GEOADD`/`GEORADIUS`. |
| HyperLogLog | A probabilistic distinct-counter using ~12 KB regardless of size. |
| `PFADD` | Add an element to a HyperLogLog. |
| `PFCOUNT` | Estimate the cardinality of a HyperLogLog. |
| `PFMERGE` | Combine multiple HyperLogLogs into one. |
| `BITCOUNT` | Count the number of set bits in a string. |
| `BITPOS` | Find the position of the first set or unset bit. |
| `BITOP` | Bitwise AND/OR/XOR/NOT on multiple keys. |
| `SETBIT` | Set a single bit in a bitmap. |
| `GETBIT` | Read a single bit from a bitmap. |
| `BITFIELD` | Read/write integer fields packed into a bitmap at arbitrary positions. |
| `INCR` | Atomic increment of an integer value. |
| `DECR` | Atomic decrement. |
| `INCRBY` | Increment by a given integer. |
| `DECRBY` | Decrement by a given integer. |
| `INCRBYFLOAT` | Increment by a float. |
| `APPEND` | Append a string to an existing string value. |
| `STRLEN` | Length of a string in bytes. |
| `GETRANGE` | Read a byte range from a string. |
| `SETRANGE` | Overwrite part of a string at a byte offset. |
| `GETSET` | Atomically set a new value and return the old one. |
| `MSET` | Set many key/value pairs in one round-trip. |
| `MGET` | Read many keys in one round-trip. |
| `EXPIRE` | Attach a TTL to a key in seconds. |
| `EXPIREAT` | Expire at an absolute Unix timestamp (seconds). |
| `PEXPIRE` | Expire in milliseconds. |
| `PEXPIREAT` | Expire at an absolute Unix timestamp (milliseconds). |
| `TTL` | Read remaining seconds. -1 means no expiry; -2 means no such key. |
| `PTTL` | Same in milliseconds. |
| `PERSIST` | Remove a TTL — make the key permanent. |
| `EXISTS` | Does this key exist? |
| `DEL` | Synchronous delete. |
| `UNLINK` | Async delete — namespace removal in main thread, memory freed in background. |
| `RENAME` | Rename a key, overwriting the destination if it exists. |
| `RENAMENX` | Rename only if the destination doesn't exist. |
| `TYPE` | Returns the data structure kind of a key. |
| `OBJECT IDLETIME` | Seconds since the key was last touched (LRU). |
| `OBJECT FREQ` | Logarithmic frequency counter (LFU). |
| `KEYS` | List all keys matching a pattern. Avoid in production. |
| `SCAN` | Cursor-based incremental keyspace iteration. The right way. |
| `HSCAN` | Same idea but inside a hash. |
| `SSCAN` | Inside a set. |
| `ZSCAN` | Inside a sorted set. |
| `RANDOMKEY` | Return a random existing key. |
| `DBSIZE` | Total key count in the current database. |
| `FLUSHDB` | Wipe the current database. |
| `FLUSHALL` | Wipe every database. |
| `SELECT` | Switch to a numbered database (0–15 by default). |
| `SWAPDB` | Atomically swap two databases. |
| `MULTI` | Start a transaction — queue commands. |
| `EXEC` | Run the queued transaction atomically. |
| `DISCARD` | Throw away the queued transaction. |
| `WATCH` | Optimistic-lock a key for the next `MULTI`/`EXEC`. |
| `UNWATCH` | Drop watches without executing. |
| `EVAL` | Run a Lua script atomically server-side. |
| `EVALSHA` | Run a previously-loaded script by SHA1 hash. |
| `SCRIPT LOAD` | Load a Lua script into the cache without running it. |
| `SCRIPT EXISTS` | Check whether scripts with given SHA1s are loaded. |
| `SCRIPT FLUSH` | Empty the script cache. |
| `FUNCTION LOAD` | Load a Function library (Redis 7+). |
| `FUNCTION DUMP` | Export loaded functions as a binary blob. |
| `FUNCTION RESTORE` | Re-import functions from a dump. |
| `FUNCTION FLUSH` | Drop all loaded functions. |
| `CLUSTER NODES` | Report on every node in the cluster. |
| `CLUSTER SLOTS` | Map of slot ranges to nodes. |
| `CLUSTER SHARDS` | Modern shard topology view (Redis 7+). |
| `CLUSTER INFO` | Cluster health summary. |
| `CLUSTER MEET` | Tell a node to join another node. |
| `CLUSTER ADDSLOTS` | Assign slots to a node. |
| `CLUSTER FAILOVER` | Manually trigger replica promotion. |
| `REPLICAOF` | Make this server a replica of another. |
| `SLAVEOF` | Deprecated alias for `REPLICAOF`. |
| `MIGRATE` | Atomically move a key from one server to another. |
| `DUMP` | Serialize a key's value into a Redis-internal binary blob. |
| `RESTORE` | Deserialize a `DUMP` blob into a key. |
| `OBJECT ENCODING` | Show the internal encoding (listpack, hashtable, skiplist, intset, embstr...). |
| `GETDEL` | Atomically read and delete a key (Redis 6.2+). |
| `GETEX` | Read a key and optionally set/clear its TTL atomically. |
| `COPY` | Duplicate a key into another name. |
| `LMPOP` | Pop from the first non-empty list among many (Redis 6.2+). |
| `BLMPOP` | Blocking version of `LMPOP`. |
| `ZMPOP` | Same idea for sorted sets. |
| `SMISMEMBER` | Test multiple `SISMEMBER`s in one round-trip. |
| `LMOVE` | Atomically pop from one list and push to another. |
| `SUBSCRIBE` | Listen on a pub/sub channel. |
| `PSUBSCRIBE` | Pattern-subscribe to channels matching a glob. |
| `UNSUBSCRIBE` | Stop listening on channels. |
| `PUNSUBSCRIBE` | Stop pattern subscriptions. |
| `PUBSUB CHANNELS` | List active channels. |
| `PUBSUB NUMSUB` | Subscriber counts per channel. |
| `PUBSUB NUMPAT` | Number of pattern subscriptions. |
| `ACL LIST` | Show all configured users (Redis 6+). |
| `ACL SETUSER` | Create or modify a user. |
| `ACL WHOAMI` | Which user am I authenticated as? |
| `ACL CAT` | List ACL categories or commands in a category. |
| `AUTH` | Authenticate with password (or user + password). |
| `CLIENT LIST` | Show every connected client. |
| `CLIENT KILL` | Disconnect a specific client. |
| `CLIENT PAUSE` | Stop processing client commands for N milliseconds. |
| `CLIENT UNPAUSE` | Resume processing. |
| `CLIENT GETNAME` | Read the connection's name. |
| `CLIENT SETNAME` | Tag the connection with a name (for logs). |
| `CLIENT NO-EVICT` | Mark the client immune to client-eviction (Redis 7+). |
| `MEMORY USAGE` | How many bytes a single key is using. |
| `MEMORY STATS` | Detailed memory breakdown. |
| `MEMORY DOCTOR` | Plain-English memory health summary. |
| `MEMORY PURGE` | Tell the allocator to release free pages. |
| `INFO` | The big status report. Sections: server, clients, memory, persistence, etc. |
| `INFO server` | Build, version, uptime. |
| `INFO clients` | Connected client count. |
| `INFO memory` | Memory and fragmentation. |
| `INFO persistence` | RDB/AOF status. |
| `INFO replication` | Primary/replica state. |
| `INFO commandstats` | Per-command call counts and latency. |
| `INFO keyspace` | Per-database key counts and TTL distribution. |
| `LATENCY HISTORY` | Past latency events for a named source. |
| `LATENCY RESET` | Clear the latency log. |
| `LATENCY GRAPH` | ASCII art latency histogram. |
| `LATENCY DOCTOR` | Plain-English latency summary. |
| `SLOWLOG GET` | Last N commands that exceeded the slow threshold. |
| `SLOWLOG RESET` | Clear the slow log. |
| `SLOWLOG LEN` | How many slow log entries exist. |
| `MONITOR` | Echo every command on every connection. Debug only. |
| `DEBUG OBJECT` | Internal info about a single key. |
| `DEBUG SLEEP` | Block the server for N seconds — for testing. |
| `DEBUG SDSLEN` | Internal string-buffer info. |
| `redis-benchmark` | Built-in workload generator. |
| `redis-check-aof` | Repair a damaged AOF. |
| `redis-check-rdb` | Validate an RDB file. |
| RediSearch | Module: secondary indexes, full-text, vector search. |
| RedisJSON | Module: JSON values as a native type. |
| RedisTimeSeries | Module: time-series with downsampling. |
| RedisBloom | Module: Bloom, Cuckoo, Top-K, T-Digest. |
| RedisAI | Module (deprecated): in-process model serving. |
| RedisStack | The all-in-one bundle of Redis core + popular modules. |
| KeyDB | A multithreaded Redis fork. |
| Dragonfly | A clean-room reimplementation, modern multi-shared-nothing C++. |
| Valkey | The BSD-licensed Linux Foundation fork after Redis went RSALv2/SSPL in 2024. |
| primary | The writable Redis instance in a replication setup. |
| replica | A read-only copy that streams writes from the primary. |
| Sentinel | Process that monitors primary/replicas and triggers failover. |
| Cluster | Sharded Redis with 16,384 hash slots gossiped across nodes. |
| hash slot | One of the 16,384 buckets a key can hash into. |
| MOVED | Redirect from a cluster node telling the client a slot is owned elsewhere. |
| ASK | Temporary redirect during slot migration — ask the new owner just for this command. |
| hash tag | The `{...}` portion of a key that determines its slot, allowing multi-key ops. |
| big key | A single key with a very large value; causes latency spikes. |
| listpack | Compact, scan-based encoding for small hashes/lists/zsets. |
| skiplist | Probabilistic ordered structure used in big sorted sets. |
| keyspace notification | Server-side pub/sub event when keys change. |
| `client side caching` | Redis 6+ feature that lets clients cache values and get push-invalidation. |
| OOM | Out Of Memory — Redis hit `maxmemory` and the policy refuses writes. |
| eviction | Removal of a key by Redis to free memory. |
| expiration | Automatic deletion of a key whose TTL elapsed. |
| TTL | Time To Live in seconds. |
| pipelining | Sending many commands without waiting between replies. |
| MULTI/EXEC | Redis transaction begin/end. |
| optimistic locking | The pattern of `WATCH` + `MULTI`/`EXEC` to detect conflicting writes. |
| RDB preamble | The initial snapshot bytes inside a hybrid AOF file. |
| `aof-use-rdb-preamble` | Config that turns on hybrid AOF. |
| `repl-backlog-size` | Ring buffer of recent writes used for partial replication sync. |

## Try This

Quick exercises to do right now while everything is fresh.

### 1. Build a tiny rate limiter

```bash
$ redis-cli SET ratelimit:alice 0 EX 60 NX     # first request, expire in 60s
OK
$ redis-cli INCR ratelimit:alice
(integer) 1
$ redis-cli INCR ratelimit:alice
(integer) 2
$ redis-cli TTL ratelimit:alice
(integer) 58
```

When `INCR` returns a value greater than your limit, reject the request. The TTL cleans up automatically.

### 2. Build a leaderboard

```bash
$ redis-cli ZADD scores 100 alice 250 bob 175 carol 320 dave
(integer) 4
$ redis-cli ZREVRANGE scores 0 2 WITHSCORES
1) "dave"   2) "320"
3) "bob"    4) "250"
5) "carol"  6) "175"
$ redis-cli ZINCRBY scores 50 alice
"150"
$ redis-cli ZRANK scores alice          # 0 = lowest
$ redis-cli ZREVRANK scores alice       # 0 = highest
```

Try `ZRANGEBYSCORE scores 100 200` to get everyone in a score range.

### 3. Build a queue and a worker

Two terminals.

Producer:

```bash
$ redis-cli LPUSH jobs '{"id":1,"task":"resize image"}'
$ redis-cli LPUSH jobs '{"id":2,"task":"send email"}'
```

Worker:

```bash
$ redis-cli BRPOP jobs 0
1) "jobs"
2) "{\"id\":1,\"task\":\"resize image\"}"
```

`BRPOP` blocks indefinitely (`0`) until a job arrives.

### 4. Snapshot, kill, restart

```bash
$ redis-cli SET pet alice
$ redis-cli BGSAVE
$ pkill redis-server
$ redis-server &
$ redis-cli GET pet
"alice"
```

Confirms RDB persistence works. The dataset survived a hard kill.

### 5. Watch a transaction abort under WATCH

Two terminals.

Terminal A:

```bash
$ redis-cli
> WATCH counter
OK
> GET counter
"5"
> MULTI
OK
> SET counter 6
QUEUED
```

Terminal B (before A runs `EXEC`):

```bash
$ redis-cli SET counter 100
OK
```

Terminal A:

```
> EXEC
(nil)
```

The transaction aborted because the watched key changed. Retry from `WATCH`.

### 6. Find your big keys

```bash
$ redis-cli --bigkeys
```

Run on any non-trivial Redis. The output is a per-type summary plus the largest keys it found.

## Where to Go Next

After this sheet, you have a real working mental model. Concrete next moves:

- Read `databases/redis` for the dense reference of every command and option.
- Read `ramp-up/postgres-eli5` to compare on-disk relational vs in-memory key-value.
- Read `databases/sql` if you have not used SQL much — it makes the comparison sharper.
- Practice with `redis-cli` against a local server — try every command in the Hands-On section.
- Read antirez's blog posts on data structure design (`The end of the line for slaves`, `Why Redis is so fast`, `Lua scripting and atomicity`) — they are short and excellent.
- Read "The Little Redis Book" by Karl Seguin (free PDF, two-hour read) for a second voice.

## See Also

- `databases/redis`
- `databases/postgresql`
- `databases/sql`
- `databases/sqlite`
- `ramp-up/postgres-eli5`
- `ramp-up/linux-kernel-eli5`
- `ramp-up/tcp-eli5`

## References

- redis.io/docs — the canonical command reference and tutorials
- "Redis in Action" by Josiah Carlson — the long-form book, freely available
- "Redis Essentials" by Maxwell Dayvson Da Silva — patterns and recipes
- "The Little Redis Book" by Karl Seguin — short, free, an excellent first read
- antirez (Salvatore Sanfilippo) blog — long technical posts on data structures, persistence, and the philosophy of Redis design
- `redis-cli` man page — `man redis-cli` (or `redis-cli --help`)
- `redis.conf` reference — annotated example shipped with every release; the most underread document in the project
- RESP3 spec — `https://github.com/redis/redis-specifications/blob/master/protocol/RESP3.md`
- Valkey project — `https://valkey.io` — the BSD-licensed continuation
