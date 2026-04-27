# nginx — ELI5

> nginx is a polite receptionist and a traffic cop, sitting at the front door of your web app, taking every visitor's request and quietly figuring out who should answer it.

## Prerequisites

(none — but it helps to know what HTTP is)

You do not need to know how to write code. You do not need to have ever run a website. You do not need to know what a "server" is. By the end of this sheet you will know all of those things in plain English, and you will have typed real `nginx` commands into a real terminal and watched real things happen.

It does help if you have heard of **HTTP** before. HTTP is the way web browsers talk to web servers. When you type `https://example.com` into your browser and press Enter, your browser sends an HTTP **request** ("hey, give me the home page please"), and the web server sends back an HTTP **response** ("here is the home page"). That is the whole conversation. If you do not know what HTTP is, you can still read this sheet, and we will explain HTTP again in tiny pieces as we go. There is also a sister sheet, `ramp-up/tcp-eli5`, that walks you through the layer underneath HTTP.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is nginx

### Imagine a really, really busy office building

Picture a giant office building with hundreds of doors and hundreds of phones. Inside the building there are dozens of teams: a billing team, a support team, a design team, a sales team, a kitchen team. Outside the building, all day long, people walk up to the front door and want to talk to one of those teams. Some people have appointments. Some are deliveries. Some are just lost. Some are trying to break in.

If every visitor just barged into the building and started wandering around looking for the right team, it would be chaos. The billing team would have strangers walking through their offices. The support team would have tourists asking about pizza. Some teams would be slammed with visitors while other teams sat empty. The whole building would fall apart in an hour.

So the building has a **receptionist** at the front desk. Every single visitor goes through the receptionist. The receptionist asks "who are you here to see?" The receptionist looks at a clipboard. The receptionist makes the visitor wait, or sends them down the right hallway, or hands them a brochure that already has the answer, or calls upstairs and gets the answer for them. The receptionist never makes pizza. The receptionist never writes invoices. The receptionist just sits at the front desk and routes visitors to the right place.

**That receptionist is nginx.**

The visitors are HTTP requests. The teams inside the building are your **backend** services — your real web app, your database, your image storage, your payment processor. The clipboard is the nginx **configuration file**. The brochures are **cached** answers nginx already knows. The hallways are **upstream** connections to your real services. nginx is the polite, fast, never-tired person at the front desk who takes every request and figures out where it should go.

### Imagine a polite traffic cop at a four-way intersection

Here is another picture. Pretend you have a really busy intersection in a city. Cars are coming from four directions. Some cars want to go straight. Some want to turn left. Some want to turn around and go home. Pedestrians want to cross. Bikes want to weave through. If everybody just went whenever they felt like it, there would be a crash every minute.

So there is a traffic cop in the middle. The cop holds up a hand and says "stop." The cop waves and says "go." The cop times each direction so cars from the north get a turn, then cars from the east, then pedestrians, then cars from the south, and so on. The cop never stops working. The cop is always there. The cop never lets two streams of traffic crash into each other.

**That cop is also nginx.**

The cars are HTTP requests. The four directions are different parts of your app. nginx waves them through to the right place, and if too many cars come from one direction it can hold them at a red light (this is called **rate limiting**) until things calm down.

### Why both pictures?

The receptionist picture is best for understanding that nginx **routes** every request to the right backend.

The traffic cop picture is best for understanding that nginx **manages flow** so backends do not get overwhelmed and visitors do not crash into each other.

You can use either picture. Whichever clicks for you is the right one. We will keep using both.

### The boring official definition (you can skip this)

nginx (pronounced "engine-X") is a free, open-source, high-performance HTTP server, reverse proxy, load balancer, mail proxy, and generic TCP/UDP proxy, written in C, originally created by Igor Sysoev in 2002 to solve the **C10K problem** (handling ten thousand concurrent connections on one machine).

You did not need to read that. The receptionist picture is the same idea.

### What people use nginx for

There are five jobs nginx does, and most websites use nginx for at least three of them at the same time.

**Job 1: Reverse proxy.** A reverse proxy is a fancy word for "the receptionist." Visitors hit nginx, nginx talks to the real app inside, nginx hands the answer back. The visitors never talk to the real app directly.

**Job 2: Load balancer.** If you have ten copies of your real app running (because one copy is not fast enough), nginx spreads visitors across the ten copies so no single copy gets crushed.

**Job 3: TLS terminator.** TLS (the "S" in HTTPS) is encryption. nginx handles all the encryption math at the front door, so your real app can speak plain unencrypted HTTP on the inside of the building where it is safe. This is called "terminating" TLS, like a bus terminus where the bus stops and everyone gets off.

**Job 4: Static file server.** If somebody asks for a picture or a CSS file or a JavaScript file, nginx can just hand it over from the disk without bothering the real app at all. nginx is unbelievably fast at this — way faster than most app frameworks.

**Job 5: Cache.** If the answer to a question is the same for a thousand visitors in a row, nginx can remember the answer for a few seconds and hand out the cached copy without asking the real app. This is the "brochures at the front desk" idea.

You can use nginx for one of these jobs, or all five at once. Most websites use all five.

## Apache vs nginx

You may have heard of **Apache HTTPD**, often just called "Apache." Apache is another web server. It came out in 1995, way before nginx (2002). For about ten years Apache was the most popular web server on the internet. Then nginx slowly took over. Today, nginx serves more of the internet than Apache. Why?

The short answer: **nginx is event-driven, Apache (by default) is process/thread-per-connection.** That is jargon. Let us explain it like you are five.

### Apache: one waiter per table

Imagine a restaurant with one rule: every table gets its own dedicated waiter, and that waiter does NOTHING else until that table leaves. The waiter takes the order, walks to the kitchen, waits, brings food, asks if you want dessert, brings the bill, walks you to the door, waves goodbye. Then the waiter goes back into the closet and waits for a new table.

If you have 50 tables, you need 50 waiters. If you have 5,000 tables, you need 5,000 waiters. Each waiter is a whole person — they take up space in the building, they need a paycheck, they need a chair to sit on. If 90% of those waiters are just standing there waiting for the kitchen to finish cooking, they are still costing you money and taking up space.

That is Apache's classic mode (**prefork** or **worker** MPM). Each connection gets a whole process or a whole thread, and that process or thread is locked to that connection until it ends. If the connection is slow (a visitor on a phone in a tunnel), the worker just sits there blocked, waiting. Apache solves the "blocked worker" problem by spinning up more workers — but each worker uses memory. Eventually you run out of memory.

### nginx: one super-fast waiter for the whole restaurant

Now imagine the same restaurant, but with one weird waiter. This waiter has a magic notebook. The waiter walks to table 1, takes the order, writes it down, immediately walks to table 2, takes their order, writes it down, immediately walks to table 3, takes their order, etc. Whenever any table needs anything — food arrives, water needs refilling, somebody waves — the waiter zooms over for half a second, does the thing, and zooms to the next table. The waiter is **never blocked.** The waiter is never just standing there waiting. The waiter is always doing something.

One waiter, this way, can handle hundreds of tables — because each individual interaction takes only milliseconds, and the waiter never sits and waits.

That is nginx. nginx has a small number of **worker processes** (often one per CPU core), and each worker uses an **event loop** to handle thousands of connections at the same time. When a connection is waiting for the kitchen (the backend), the worker does not sit there — it goes do something else, and comes back when the kitchen rings the bell.

### What is "event-driven"?

"Event-driven" means the program reacts to **events**: "a new visitor arrived," "this connection has data ready to read," "this backend just answered." Instead of having a thread per connection that sits and waits, you have one thread that watches a list of events and handles whichever one is ready next. The Linux kernel provides a system call called `epoll` that lets one process watch tens of thousands of network connections at once and only wake up when something interesting happens. nginx uses `epoll` (on Linux) or `kqueue` (on BSD/macOS) to do this.

Think of it like a sleeping watchdog with one ear up. The watchdog does not patrol the yard endlessly — it lies down and naps. But the second any one of a hundred sensors trips, the watchdog springs awake and deals with that sensor. Then it lies down again. That is `epoll`. That is nginx.

### So which is better?

Honestly? It depends. Apache has its own event-driven mode now (`event` MPM) that is much closer to nginx in performance. Apache also has a richer module ecosystem for some things, especially `.htaccess` per-directory config files, which nginx does not support (and on purpose — `.htaccess` files are slow because Apache re-reads them on every request). nginx is generally easier to configure, much faster for static files, much better at being a reverse proxy in front of other apps, and uses far less memory under heavy load.

Most modern setups use nginx as the front-door receptionist and let the real app (Python, Node.js, Ruby, Java, Go, whatever) live behind it. Apache is still common, but if you are starting fresh in 2026, nginx is almost always the right choice unless you have a very specific reason.

## The Process Model

When you start nginx, it does not run as one program. It runs as **several processes that talk to each other.** This sounds scary but is actually super simple once you see the picture.

```
                    ┌──────────────────┐
                    │  master process  │   (runs as root, doesn't serve traffic)
                    │   - reads config │
                    │   - opens ports  │
                    │   - spawns kids  │
                    └─────┬──┬──┬──┬───┘
                          │  │  │  │
              ┌───────────┘  │  │  └────────────┐
              │              │  │               │
              v              v  v               v
       ┌───────────┐ ┌───────────┐ ┌───────────────┐ ┌───────────────┐
       │  worker 1 │ │  worker 2 │ │ cache loader  │ │ cache manager │
       │ (epoll)   │ │ (epoll)   │ │ (one-shot,    │ │ (forever, low │
       │ serves    │ │ serves    │ │  on startup)  │ │  priority)    │
       │ requests  │ │ requests  │ │ scans cache   │ │ evicts old    │
       └───────────┘ └───────────┘ │ dir, builds   │ │ cache files   │
                                   │ in-mem index  │ │               │
                                   └───────────────┘ └───────────────┘
```

There is exactly one **master process.** The master is the boss. The master usually runs as the **root** user (because only root can bind to ports below 1024, like port 80 and 443). The master does NOT serve any traffic itself. The master's whole job is to read the config file, open the listening sockets, and spawn the worker processes. Then the master sits there and waits — for signals like "reload," "shut down," "rotate logs."

There are many **worker processes.** Workers are the ones that actually do the work. Each worker handles its own pile of connections. By default, nginx spawns one worker per CPU core. If you have 8 cores, you get 8 workers. Workers run as a non-root user (usually `www-data` or `nginx`) for safety — if a worker gets hacked, the attacker is not root.

There are usually two cache helpers, but only if caching is configured.

The **cache loader** runs once, when nginx starts. It walks through the on-disk cache directory and builds an in-memory index of what is cached. Then it exits. It is a one-shot process.

The **cache manager** runs forever, in the background, at low priority. Its job is to clean up old cache entries that have expired so the cache directory does not grow forever. It is the janitor.

### Why so many processes?

Because they each do one job and one job only. If a worker crashes, the master notices and spawns a new worker. The other workers keep serving traffic the entire time. The visitor never sees a hiccup. This is **fault isolation** — one bad request can blow up one worker, and nginx just restarts that worker and moves on.

It is also how nginx does **zero-downtime reloads.** When you tell nginx to reload its config, the master starts NEW workers with the new config, lets the OLD workers finish their current connections, then tells the old workers to exit. There is never a moment when no workers are running. We will talk about this more in the **Reload Without Downtime** section.

## Configuration File Anatomy

Every nginx setup is controlled by a configuration file. The main file is usually at `/etc/nginx/nginx.conf` on Linux. You can include other files from inside it, and most distributions ship with a layout like this:

```
/etc/nginx/
├── nginx.conf                  ← main file, read first
├── mime.types                  ← maps file extensions to Content-Type headers
├── conf.d/                     ← drop-in config snippets
│   └── default.conf
├── sites-available/            ← (Debian/Ubuntu) all your virtual hosts
│   └── example.com.conf
└── sites-enabled/              ← (Debian/Ubuntu) symlinks to sites-available
    └── example.com.conf -> ../sites-available/example.com.conf
```

The config file is a tree of nested **blocks** with **directives** inside them. Each directive ends with a semicolon `;`. Each block is wrapped in `{` and `}`. Indentation is for humans only — nginx does not care about whitespace.

Here is the shape of a complete config:

```
# top-level "main" context
user www-data;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    sendfile on;
    keepalive_timeout 65;

    server {
        listen 80;
        server_name example.com;

        location / {
            root /var/www/html;
            index index.html;
        }
    }
}
```

There are four important blocks here.

The **main** context is everything outside any block. It controls how nginx itself runs: which user to run as, how many workers, where the PID file lives.

The **events** block controls how each worker handles connections. You almost never need to change anything here except `worker_connections`.

The **http** block contains everything related to serving HTTP. Almost all of your interesting config goes inside `http { }`.

The **server** block (inside `http`) is one **virtual host** — one website. You can have many `server` blocks, one for each domain you serve.

The **location** block (inside `server`) handles one URL path or pattern. You can have many `location` blocks per server.

Other important contexts:

The **upstream** block (inside `http`) defines a pool of backend servers to load-balance across.

The **stream** block (top-level, like `http`) is for raw TCP/UDP proxying (not HTTP). This is for things like proxying SSH or Postgres or game servers.

### Directives are inherited

A directive set in an outer block is inherited by inner blocks unless overridden. If you set `gzip on;` at the `http` level, every `server` and `location` inside it has `gzip on` unless one of them turns it off. This makes config short and avoids repetition.

## A Hello-World nginx.conf

Here is the smallest useful nginx config. It serves one HTML page on port 8080.

```
worker_processes 1;
events { worker_connections 1024; }

http {
    server {
        listen 8080;
        server_name localhost;

        location / {
            return 200 "hello from nginx\n";
            add_header Content-Type text/plain;
        }
    }
}
```

That is it. Save it to `/tmp/hello.conf`, run nginx with that config, hit `http://localhost:8080/`, and you get back `hello from nginx`. Let us actually do that:

```
$ cat > /tmp/hello.conf <<'EOF'
worker_processes 1;
events { worker_connections 1024; }
http {
    server {
        listen 8080;
        location / {
            return 200 "hello from nginx\n";
            add_header Content-Type text/plain;
        }
    }
}
EOF

$ nginx -p /tmp -c /tmp/hello.conf -g 'daemon off; error_log /tmp/err.log;' &
[1] 12345

$ curl http://localhost:8080/
hello from nginx
```

You just ran a real web server. Eight lines of config, one HTTP response.

## Server Blocks (Virtual Hosts)

A **server block** is one website. You write one `server { }` block per domain. nginx picks which server block to use by looking at two things: the IP address and port the request came in on (`listen`), and the `Host:` header in the request (`server_name`).

```
http {
    # blog.example.com
    server {
        listen 80;
        server_name blog.example.com;
        root /var/www/blog;
    }

    # shop.example.com
    server {
        listen 80;
        server_name shop.example.com;
        root /var/www/shop;
    }

    # default fallback for everything else
    server {
        listen 80 default_server;
        server_name _;
        return 444;  # nginx-specific: drop the connection
    }
}
```

When a browser sends `GET / HTTP/1.1\r\nHost: blog.example.com\r\n`, nginx looks at the `Host` header, matches `server_name blog.example.com`, and serves files from `/var/www/blog`. When a browser sends `Host: shop.example.com`, it gets `/var/www/shop`. Anything else hits the `default_server` and gets dropped.

`server_name` can be a list of names, a wildcard (`*.example.com`), or a regex (`~^api\.(?<env>dev|stage|prod)\.example\.com$`).

The first server block matching a `listen` directive (or the one with `default_server`) is the default for that port. Requests with no `Host` header, or with an unknown host, hit the default.

## Location Matching

Inside a server block, `location` directives match URL paths. This is how nginx decides "the user asked for `/api/users/42`, so what do I do?"

There are several **prefixes** in front of a `location`, and they have a specific priority order. This is the part of nginx that confuses everyone, so we are going to draw it out.

```
                       Request URI: /api/users/42
                              │
                              v
           ┌─────────────────────────────────────────┐
           │ 1. EXACT MATCH:  location = /          │  highest priority
           │    Match found? → DONE, use this block  │
           └─────────────────────────────────────────┘
                              │ no exact match
                              v
           ┌─────────────────────────────────────────┐
           │ 2. PREFIX MATCH with ^~ :              │
           │    location ^~ /api/                    │
           │    Match found? → DONE, use this block  │
           │    (^~ means "stop after this prefix    │
           │     wins, do not try regex")            │
           └─────────────────────────────────────────┘
                              │ no ^~ match
                              v
           ┌─────────────────────────────────────────┐
           │ 3. REGEX MATCH (in order):             │
           │    location ~  /api/v[0-9]+/           │  ← case sensitive
           │    location ~* \.(jpg|png|gif)$        │  ← case insensitive
           │    First regex that matches wins.       │
           │    Match found? → DONE, use this block  │
           └─────────────────────────────────────────┘
                              │ no regex match
                              v
           ┌─────────────────────────────────────────┐
           │ 4. LONGEST PREFIX MATCH (no prefix):   │
           │    location /api/                       │
           │    location /                           │
           │    nginx remembered the longest         │
           │    prefix match earlier — use that.     │
           └─────────────────────────────────────────┘
```

The trick is: nginx does **prefix matching first**, remembers the **longest prefix that matched**, then checks **regexes**, and if any regex matches, the regex wins over the prefix. UNLESS the prefix used `^~`, which means "if I match, stop, do not check regex."

Let us write some.

```
location = / {
    # exact match for /
    return 200 "exact root\n";
}

location ^~ /static/ {
    # if URI starts with /static/, use this — do not check regex
    root /var/www;
}

location ~* \.(jpg|jpeg|png|gif|webp)$ {
    # any image file — case insensitive
    expires 30d;
    add_header Cache-Control public;
}

location / {
    # everything else
    proxy_pass http://backend;
}
```

A few rules of thumb:

`location =` is the fastest match. Use it for very hot paths like `/health`, `/favicon.ico`.

`location ^~` is for big static directories where you definitely do not want regex to take over.

`location ~` and `location ~*` are slower (regex evaluation per request) — use them only when prefix matching is not enough.

`location /` is the catch-all. Always have one.

## Reverse Proxy

This is the headline feature. nginx is a **reverse proxy**, which means it sits in front of a real backend application and forwards requests to it. The visitor talks to nginx. nginx talks to the backend. nginx hands the answer back to the visitor.

Why "reverse"? Because there is also a "forward proxy" (like Squid) which sits in front of clients (your browser) and forwards their outgoing requests to the public internet. nginx does the opposite — it sits in front of servers and forwards incoming requests to them.

```
                       +------------+
   visitor  <---->     |            |    <---->   your-app:3000
   visitor  <---->     |   nginx    |    <---->   your-app:3000
   visitor  <---->     |  (reverse  |    <---->   your-app:3000
   visitor  <---->     |   proxy)   |    <---->   your-app:3000
                       +------------+
                        port 80/443           backends on private network
```

Here is the simplest reverse proxy config:

```
server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}
```

Every request to `api.example.com` is forwarded to a Node/Python/Go/Java app listening on `127.0.0.1:3000`. nginx handles the public side. The app handles the actual work.

But the simple version above loses information. The backend app sees the request as if it came from `127.0.0.1` (because nginx is the one connecting to it). The app cannot tell which real visitor sent the request, what the original `Host` was, or whether the connection was HTTPS or plain HTTP. Fix that with these headers:

```
location / {
    proxy_pass http://127.0.0.1:3000;
    proxy_http_version 1.1;

    proxy_set_header Host              $host;
    proxy_set_header X-Real-IP         $remote_addr;
    proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host  $host;

    proxy_connect_timeout 5s;
    proxy_send_timeout    60s;
    proxy_read_timeout    60s;
}
```

`Host` tells the backend which domain the visitor asked for. `X-Real-IP` tells the backend the visitor's real IP. `X-Forwarded-For` is the chain of proxy IPs. `X-Forwarded-Proto` tells the backend whether the original visitor used HTTPS. Many app frameworks (Django, Rails, Express) read these headers to know they are behind a proxy.

`proxy_http_version 1.1` is required if you want WebSocket or keep-alive between nginx and backend.

For WebSockets, you need two more headers:

```
location /ws/ {
    proxy_pass http://127.0.0.1:3000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade    $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

## Upstream + Load Balancing

If you have more than one backend server, define an **upstream** pool and load-balance across it.

```
upstream app_servers {
    server 10.0.0.10:3000;
    server 10.0.0.11:3000;
    server 10.0.0.12:3000;
}

server {
    listen 80;
    location / {
        proxy_pass http://app_servers;
    }
}
```

By default nginx uses **round-robin**: request 1 goes to server 1, request 2 goes to server 2, request 3 goes to server 3, request 4 goes back to server 1. Even if one is overloaded.

You can change the strategy:

```
upstream app_servers {
    least_conn;                       # send to backend with fewest active connections
    server 10.0.0.10:3000;
    server 10.0.0.11:3000;
}

upstream sticky {
    ip_hash;                          # same client IP always goes to same backend
    server 10.0.0.10:3000;
    server 10.0.0.11:3000;
}

upstream weighted {
    server 10.0.0.10:3000 weight=3;   # gets 3x more traffic
    server 10.0.0.11:3000 weight=1;
}

upstream with_health {
    server 10.0.0.10:3000 max_fails=3 fail_timeout=30s;
    server 10.0.0.11:3000 max_fails=3 fail_timeout=30s;
    server 10.0.0.12:3000 backup;     # only used if all others are down
}

upstream keepalive_pool {
    server 10.0.0.10:3000;
    keepalive 32;                     # keep 32 idle connections to backend
}
```

`least_conn` is great when requests have wildly different durations (some take 10ms, some take 10s).

`ip_hash` is for "sticky sessions" — when each user must always hit the same backend (because session state lives on the backend, not in a shared cache). This is a workaround; the real fix is to put session state in Redis or a shared DB. But sometimes sticky is what you have.

`weight` is for when your backends have different capacity (one is a bigger box).

`max_fails=3 fail_timeout=30s` means "if a backend fails 3 times in 30 seconds, take it out of rotation for 30 seconds." This is **passive health checking** — nginx open source does not actively probe backends. (NGINX Plus does. There are also third-party modules like `nginx_upstream_check_module`.)

`backup` means "do not use this server unless all the others are down."

`keepalive 32` is critical for performance — without it, nginx opens a new TCP connection to the backend for every single request, which is incredibly wasteful. With it, nginx reuses connections.

## Static File Serving

If a request is for a file on disk, nginx is the fastest way to serve it. nginx uses `sendfile()` (a Linux system call) to copy a file directly from disk to the network socket without ever passing through nginx itself in user space. Almost zero CPU overhead.

```
server {
    listen 80;
    server_name static.example.com;

    root /var/www/site;

    location / {
        try_files $uri $uri/ =404;
    }

    location /downloads/ {
        alias /mnt/big-storage/files/;
        autoindex on;
    }
}
```

Three directives matter here.

`root` tells nginx where the document root is. A request for `/foo/bar.html` becomes a lookup for `/var/www/site/foo/bar.html`.

`alias` is like `root` but it **replaces** the prefix instead of appending. With `alias /mnt/big-storage/files/`, a request for `/downloads/movie.mp4` becomes `/mnt/big-storage/files/movie.mp4`. Notice that `/downloads/` was *replaced*, not appended. People mix up `root` and `alias` constantly. Rule: use `alias` if the URL prefix and the disk prefix do not match.

`try_files` tries multiple paths in order and returns the first one that exists. `try_files $uri $uri/ =404` means: try the file at `$uri`, then try a directory at `$uri/`, then return 404. This is the standard way to serve a single-page app: `try_files $uri /index.html;` falls back to `index.html` for any URL that does not match a real file (so the SPA's client-side router can handle it).

`autoindex on` shows a directory listing if the URL matches a directory. Useful for file servers.

## TLS Termination

TLS (Transport Layer Security) is the encryption that makes HTTP into HTTPS. nginx handles TLS for you. The browser does the TLS handshake with nginx, nginx decrypts the request, talks to your backend in plain HTTP on a private network, gets the answer, encrypts it, sends it back. This is **TLS termination** at nginx.

Get a certificate (most people use Let's Encrypt via `certbot`), then:

```
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;                         # since 1.25
    server_name example.com;

    ssl_certificate     /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE+AESGCM:ECDHE+CHACHA20:!aNULL:!MD5;
    ssl_prefer_server_ciphers off;

    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets off;

    ssl_stapling on;
    ssl_stapling_verify on;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location / {
        proxy_pass http://backend;
    }
}

# redirect all HTTP to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name example.com;
    return 301 https://$host$request_uri;
}
```

`listen 443 ssl` enables TLS on port 443.

`http2 on;` enables HTTP/2. (Old syntax was `listen 443 ssl http2;` — both work, but the new one is preferred since 1.25.)

For HTTP/3 (over QUIC, on UDP):

```
listen 443 ssl;
listen 443 quic reuseport;
http2 on;
http3 on;                             # since 1.25 (stable)
add_header Alt-Svc 'h3=":443"; ma=86400';
```

`ssl_certificate` is the full chain (your cert plus all intermediates). `ssl_certificate_key` is your private key. **Never** check the private key into git. Permissions on the key file should be `0600` and owned by root.

`ssl_protocols TLSv1.2 TLSv1.3` — disable everything older. TLS 1.0 and 1.1 are dead.

For a known-good modern cipher list, copy from `https://ssl-config.mozilla.org/`. They keep it updated. Do not hand-roll cipher strings.

`ssl_session_cache shared:SSL:10m` lets multiple workers share a TLS session cache, which speeds up repeat handshakes.

`ssl_stapling` lets nginx attach the OCSP response to the handshake, so browsers do not have to do their own OCSP lookup. Faster TLS for visitors.

`Strict-Transport-Security` (HSTS) tells browsers "always use HTTPS for this domain, even if the user typed http://."

## Caching

nginx can cache backend responses on disk and serve them to later visitors without bothering the backend. This is the "brochures at the front desk" idea.

```
http {
    proxy_cache_path /var/cache/nginx
                     levels=1:2
                     keys_zone=mycache:10m
                     max_size=1g
                     inactive=60m
                     use_temp_path=off;

    server {
        listen 80;
        location / {
            proxy_pass http://backend;
            proxy_cache mycache;
            proxy_cache_valid 200 302 10m;
            proxy_cache_valid 404      1m;
            proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
            proxy_cache_lock on;
            add_header X-Cache-Status $upstream_cache_status;
        }
    }
}
```

`proxy_cache_path` defines the cache: where on disk, how big, how the directory tree is laid out (`levels=1:2` means two levels of subdirectories), what the in-memory key index should be called and how big.

`proxy_cache mycache` enables caching for this location, using the named zone.

`proxy_cache_valid 200 302 10m` means "200 and 302 responses are cached for 10 minutes."

`proxy_cache_use_stale` is the magic. It says "if the backend is broken, serve a stale cached copy instead of an error." Your site stays up even when your backend is down.

`proxy_cache_lock on` ensures that if 100 visitors ask for the same uncached page at the exact same instant, only ONE request goes to the backend; the other 99 wait for the answer. Without this you get a "cache stampede" where 100 identical requests slam your backend.

`X-Cache-Status` is a debug header. Values: `MISS`, `HIT`, `EXPIRED`, `STALE`, `UPDATING`, `BYPASS`.

### Microcaching

A common trick: cache for **just one second.** Sounds useless, right? It is not. If your site gets 1000 requests per second for the home page, microcaching for 1 second means your backend gets ONE request per second — a 1000x reduction. The visitors do not notice the 1-second staleness.

```
proxy_cache_valid 200 1s;
```

That is microcaching. It is one of the most underrated nginx features.

## Compression

Compress text responses (HTML, JSON, CSS, JS) before sending them. Smaller bytes over the wire, faster page loads.

```
http {
    gzip on;
    gzip_comp_level 5;            # 1=fast, 9=tight; 5 is the sweet spot
    gzip_min_length 256;
    gzip_proxied any;
    gzip_vary on;
    gzip_types
        application/javascript
        application/json
        application/xml
        text/css
        text/plain
        text/xml;
}
```

`gzip on` enables gzip. `gzip_comp_level 5` is a good balance. `gzip_min_length 256` skips compression for tiny responses (compression overhead is not worth it). `gzip_vary on` adds the `Vary: Accept-Encoding` header so caches do the right thing.

`gzip_proxied any` tells nginx to compress responses even when they came from a proxied backend. Without this, nginx would only compress its own responses.

For better compression, install the `brotli` module and add:

```
brotli on;
brotli_comp_level 5;
brotli_types text/plain text/css application/json application/javascript;
```

Brotli compresses 15-25% better than gzip for text. Modern browsers support it.

## Rate Limiting

Stop one bad client (or one annoying scraper) from overloading your backend.

```
http {
    limit_req_zone $binary_remote_addr zone=mylimit:10m rate=10r/s;
    limit_conn_zone $binary_remote_addr zone=conns:10m;

    server {
        listen 80;

        location /api/ {
            limit_req zone=mylimit burst=20 nodelay;
            limit_conn conns 10;
            proxy_pass http://backend;
        }
    }
}
```

`limit_req_zone` defines a rate-limit zone. `$binary_remote_addr` is the client IP in compact binary form. `zone=mylimit:10m` reserves 10MB of shared memory for the zone (enough for ~160k unique IPs). `rate=10r/s` is 10 requests per second per IP.

`limit_req zone=mylimit burst=20 nodelay` applies the limit. `burst=20` lets up to 20 requests queue up if traffic exceeds the rate (smooths over short bursts). `nodelay` means burst requests are served immediately, not delayed; subsequent requests beyond burst get a 503.

`limit_conn` limits **simultaneous connections** per IP, not request rate. Useful for download links where one IP could otherwise open 500 parallel connections.

When a limit is hit, nginx returns `503 Service Unavailable` (or `429 Too Many Requests` if you set `limit_req_status 429;`).

## Logging

nginx writes two main log files: **access** and **error**.

```
http {
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    log_format json_combined escape=json
        '{'
            '"time":"$time_iso8601",'
            '"remote_ip":"$remote_addr",'
            '"method":"$request_method",'
            '"path":"$request_uri",'
            '"status":$status,'
            '"bytes":$body_bytes_sent,'
            '"req_time":$request_time,'
            '"upstream_time":"$upstream_response_time",'
            '"ua":"$http_user_agent",'
            '"referer":"$http_referer",'
            '"xff":"$http_x_forwarded_for"'
        '}';

    access_log /var/log/nginx/access.log json_combined;
    error_log  /var/log/nginx/error.log warn;
}
```

`log_format` defines a named format. `escape=json` (added in 1.11.8) tells nginx to JSON-escape strings in variables — critical if you log user-controlled data, otherwise quotes and newlines in user agents will corrupt your JSON logs.

`access_log /var/log/nginx/access.log json_combined;` writes one line per request in the JSON format.

`error_log` levels: `debug | info | notice | warn | error | crit | alert | emerg`. Use `warn` in production. `debug` is extremely chatty (and requires nginx built with `--with-debug`).

You can disable logging for noisy paths:

```
location /health {
    access_log off;
    return 200;
}
```

You can write to stderr/stdout (useful in containers):

```
access_log /dev/stdout;
error_log  /dev/stderr;
```

## Reload Without Downtime

The killer feature. You change `nginx.conf`, you run `nginx -s reload`, and nginx applies the new config **without dropping a single connection.**

How does it work?

```
1. You edit /etc/nginx/nginx.conf.
2. You run:  sudo nginx -s reload
3. The master process re-reads the config file.
4. The master sanity-checks the config. If broken: abort, keep old workers.
5. The master starts NEW workers with the new config.
6. The master sends SIGQUIT (graceful shutdown) to the OLD workers.
7. Old workers stop accepting new connections, but keep handling
   their existing in-flight requests until those finish.
8. When an old worker finishes its last request, it exits.
9. Eventually only new workers are running. Reload complete.
```

There is **never** a moment without workers serving traffic. Visitors do not see anything.

You can also send signals directly:

```
kill -HUP   $(cat /run/nginx.pid)   # same as: nginx -s reload
kill -USR1  $(cat /run/nginx.pid)   # rotate logs (close+reopen log files)
kill -USR2  $(cat /run/nginx.pid)   # binary upgrade (replace nginx binary live)
kill -QUIT  $(cat /run/nginx.pid)   # graceful shutdown
kill -TERM  $(cat /run/nginx.pid)   # fast shutdown
```

The binary upgrade is wild: you can replace the nginx binary on disk with a new version, send `USR2`, and the master spawns a new master from the new binary that takes over without dropping traffic. Then `WINCH` to the old master to stop its workers, and finally `QUIT` to kill the old master.

**Always test before reloading:** `nginx -t` parses the config file and reports errors but does not apply anything. Make this a habit.

## Modules

nginx is built from modules. Some are **statically compiled** in (you cannot turn them off without rebuilding). Some are **dynamic** (compiled as `.so` files and loaded with `load_module`).

```
$ nginx -V 2>&1 | tr ' ' '\n' | grep '^--with'
--with-http_ssl_module
--with-http_v2_module
--with-http_realip_module
--with-http_gzip_static_module
--with-http_gunzip_module
--with-http_stub_status_module
--with-http_sub_module
--with-stream
--with-stream_ssl_module
...
```

That is the list of modules baked into your nginx binary.

To load a dynamic module at startup:

```
load_module modules/ngx_http_brotli_filter_module.so;
load_module modules/ngx_http_brotli_static_module.so;

events { }
http { ... }
```

Common third-party modules: `brotli`, `geoip2`, `headers-more`, `lua-nginx-module` (OpenResty), `nginx-rtmp-module` (streaming), `mod_security` (WAF).

Use `nginx -V` to see what was compiled in. If something you need is missing, you have to either install a different package or compile nginx yourself.

## Security Hardening

A short checklist of things every production nginx should do.

```
http {
    # Don't leak the version number
    server_tokens off;

    # Cap request body size to prevent giant-upload DoS
    client_max_body_size 10m;

    # Cap header sizes
    large_client_header_buffers 4 8k;

    # Reasonable timeouts
    client_body_timeout 12s;
    client_header_timeout 12s;
    keepalive_timeout 15s;
    send_timeout 10s;

    # Modern TLS only
    ssl_protocols TLSv1.2 TLSv1.3;

    server {
        # Force HTTPS in browser
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
        # Don't let your site be framed (clickjacking)
        add_header X-Frame-Options "SAMEORIGIN" always;
        # Stop browsers from MIME-sniffing wrong content types
        add_header X-Content-Type-Options "nosniff" always;
        # Limit referer leak
        add_header Referrer-Policy "strict-origin-when-cross-origin" always;
        # Restrict what features the page can use
        add_header Permissions-Policy "geolocation=(), camera=(), microphone=()" always;
        # Optional: Content Security Policy (powerful but easy to break)
        # add_header Content-Security-Policy "default-src 'self'" always;
    }
}
```

`server_tokens off` removes the nginx version from the `Server` header and error pages. Less info for attackers.

`client_max_body_size 10m` is critical — the default is 1MB and you will get `413 Request Entity Too Large` for anything bigger. Set this on file-upload endpoints. But do not set it absurdly high everywhere — that is a DoS waiting to happen.

`X-Frame-Options: SAMEORIGIN` stops clickjacking. The newer way is `Content-Security-Policy: frame-ancestors 'self'`, but X-Frame-Options is widely supported and easy.

`always` on `add_header` is important: without it, nginx only adds the header for 200/201/204/206/301/302/303/304/307/308 responses. You usually want the security headers on error responses too.

Hide your version even more by patching out the "Server: nginx" altogether using the `headers-more-nginx-module`:

```
more_clear_headers Server;
```

## Common Errors

Real nginx error messages and what they mean.

### 1. `bind() to 0.0.0.0:80 failed (98: Address already in use)`

Something else is already using port 80. Probably another nginx, or Apache, or a stuck previous instance.

```
$ sudo ss -tlnp 'sport = :80'
LISTEN 0 511 0.0.0.0:80 0.0.0.0:* users:(("apache2",pid=1234,fd=4))
```

Stop the other thing: `sudo systemctl stop apache2`.

### 2. `bind() to 0.0.0.0:80 failed (13: Permission denied)`

You are running nginx as a non-root user, and ports below 1024 are privileged. Either run nginx as root (so the master can bind, then drop to `www-data` for workers — this is normal), or use a high port (`listen 8080`), or grant the binary the capability: `sudo setcap 'cap_net_bind_service=+ep' $(which nginx)`.

### 3. `nginx: [emerg] unknown directive "proxy_ass" in /etc/nginx/nginx.conf:42`

Typo. You wrote `proxy_ass` instead of `proxy_pass`. Or you used a directive from a module that is not loaded.

### 4. `nginx: [emerg] host not found in upstream "backend.local" in /etc/nginx/nginx.conf:10`

DNS lookup failed at startup. nginx resolves upstream hostnames at config-load time by default. If the DNS lookup fails, nginx refuses to start. Two fixes: put backend IPs (not hostnames) in the upstream, or use a `resolver` directive plus the `set $var "backend.local"; proxy_pass http://$var;` trick to defer DNS to per-request.

### 5. `nginx: [emerg] SSL_CTX_use_PrivateKey_file("/etc/nginx/cert.key") failed (SSL: ... )`

The private key file is broken, the wrong file, or unreadable. Check that `ssl_certificate` and `ssl_certificate_key` exist, are readable by the nginx user, and that the key matches the cert: `openssl x509 -noout -modulus -in cert.crt | openssl md5` and `openssl rsa -noout -modulus -in cert.key | openssl md5` should produce the same hash.

### 6. `connect() failed (111: Connection refused) while connecting to upstream`

Browser sees `502 Bad Gateway`. nginx tried to connect to the backend and got refused. The backend is not running, or it is listening on a different port. Test from the nginx box: `curl -v http://127.0.0.1:3000/`. If that fails, your app is the problem, not nginx.

### 7. `upstream timed out (110: Connection timed out) while reading response header from upstream`

Browser sees `504 Gateway Timeout`. The backend accepted the connection but took longer than `proxy_read_timeout` (default 60s) to respond. Either the backend is slow, or the backend hung. Bump the timeout (`proxy_read_timeout 300s;`) for legitimate slow endpoints, or fix the backend.

### 8. `client intended to send too large body: 1531234 bytes`

Browser sees `413 Request Entity Too Large`. The request body exceeded `client_max_body_size`. Bump it: `client_max_body_size 50m;` (default is 1m).

### 9. `upstream sent invalid header while reading response header from upstream`

Browser sees `502 Bad Gateway`. The backend sent garbage that does not look like HTTP. Either the backend crashed mid-response, the backend is speaking the wrong protocol (you are proxying HTTPS to an HTTP-only backend with `proxy_pass http://...` instead of `https://...`, or vice versa), or you are pointing nginx at a non-HTTP service like a database.

### 10. `conflicting server name "example.com" on 0.0.0.0:80, ignored`

You have two `server { server_name example.com; listen 80; }` blocks. nginx will only use the first one. Delete one or change the `server_name`.

### 11. `nginx: [warn] could not build optimal types_hash, you should increase either types_hash_max_size: 1024 or types_hash_bucket_size: 64`

Just bump it: `types_hash_max_size 4096;`. Annoying but harmless.

### 12. `worker_connections are not enough`

Single worker is full. Either bump `worker_connections` (default 1024 is tiny) or `worker_processes`. Each worker can hold `worker_connections` total connections across all activities (clients + upstream + cache, etc.).

## Hands-On

Real commands you can copy-paste right now. Output is what you should see (your mileage varies on hostnames and timestamps).

### 1. What version of nginx and what was it built with?

```
$ nginx -v
nginx version: nginx/1.26.0
```

```
$ nginx -V
nginx version: nginx/1.26.0
built by gcc 11.4.0 (Ubuntu 11.4.0-1ubuntu1~22.04)
built with OpenSSL 3.0.2 15 Mar 2022
TLS SNI support enabled
configure arguments: --prefix=/etc/nginx --sbin-path=/usr/sbin/nginx
  --modules-path=/usr/lib/nginx/modules --conf-path=/etc/nginx/nginx.conf
  --error-log-path=/var/log/nginx/error.log --http-log-path=/var/log/nginx/access.log
  --pid-path=/var/run/nginx.pid --lock-path=/var/run/nginx.lock
  --with-http_ssl_module --with-http_v2_module --with-http_v3_module
  --with-http_realip_module --with-http_addition_module --with-http_sub_module
  --with-stream --with-stream_ssl_module ...
```

### 2. Test your config without applying

```
$ sudo nginx -t
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
```

If broken:

```
$ sudo nginx -t
nginx: [emerg] unknown directive "lisetn" in /etc/nginx/sites-enabled/foo:5
nginx: configuration file /etc/nginx/nginx.conf test failed
```

### 3. Reload nginx without downtime

```
$ sudo nginx -s reload
$ # no output = success
```

### 4. Show the full config nginx is currently using (with includes resolved)

```
$ sudo nginx -T 2>/dev/null | head -30
# configuration file /etc/nginx/nginx.conf:
user www-data;
worker_processes auto;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;
events { worker_connections 768; }
http {
    sendfile on;
    tcp_nopush on;
    types_hash_max_size 2048;
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;
    gzip on;
    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites-enabled/*;
}
...
```

### 5. See the running processes

```
$ ps -ef | grep nginx | grep -v grep
root      1234     1  0 09:00 ?  00:00:00 nginx: master process /usr/sbin/nginx
www-data  1235  1234  0 09:00 ?  00:00:01 nginx: worker process
www-data  1236  1234  0 09:00 ?  00:00:01 nginx: worker process
www-data  1237  1234  0 09:00 ?  00:00:01 nginx: worker process
www-data  1238  1234  0 09:00 ?  00:00:01 nginx: worker process
```

One master, four workers. The workers' parent PID (PPID, second column) is the master's PID.

### 6. See what nginx is listening on

```
$ sudo ss -tlnp | grep nginx
LISTEN 0 511   0.0.0.0:80    0.0.0.0:*  users:(("nginx",pid=1235,fd=6),("nginx",pid=1234,fd=6))
LISTEN 0 511   0.0.0.0:443   0.0.0.0:*  users:(("nginx",pid=1235,fd=7),("nginx",pid=1234,fd=7))
LISTEN 0 511      [::]:80       [::]:*  users:(("nginx",pid=1235,fd=8),("nginx",pid=1234,fd=8))
LISTEN 0 511      [::]:443      [::]:*  users:(("nginx",pid=1235,fd=9),("nginx",pid=1234,fd=9))
```

Both IPv4 and IPv6, both 80 and 443.

### 7. Logs on systemd

```
$ sudo journalctl -u nginx --since "10 min ago" --no-pager | tail
Apr 27 09:14:01 host systemd[1]: Reloading A high performance web server...
Apr 27 09:14:01 host systemd[1]: Reloaded A high performance web server.
```

### 8. Tail the access log live

```
$ sudo tail -f /var/log/nginx/access.log
192.0.2.10 - - [27/Apr/2026:09:14:32 +0000] "GET / HTTP/1.1" 200 612 "-" "curl/8.5.0"
192.0.2.10 - - [27/Apr/2026:09:14:33 +0000] "GET /favicon.ico HTTP/1.1" 404 153 "-" "curl/8.5.0"
```

### 9. Send a request locally

```
$ curl -I http://localhost/
HTTP/1.1 200 OK
Server: nginx/1.26.0
Date: Mon, 27 Apr 2026 09:15:11 GMT
Content-Type: text/html
Content-Length: 612
Last-Modified: Thu, 04 Apr 2024 14:20:36 GMT
Connection: keep-alive
ETag: "660ec5e4-264"
Accept-Ranges: bytes
```

### 10. Look at TLS handshake details

```
$ openssl s_client -connect example.com:443 -servername example.com -showcerts < /dev/null 2>/dev/null | openssl x509 -noout -dates -subject -issuer
notBefore=Jan 30 00:00:00 2026 GMT
notAfter=Apr 30 23:59:59 2026 GMT
subject=CN = example.com
issuer=C = US, O = Let's Encrypt, CN = R3
```

### 11. Show negotiated protocol and cipher

```
$ openssl s_client -connect example.com:443 -tls1_3 < /dev/null 2>/dev/null | grep -E 'Protocol|Cipher'
Protocol  : TLSv1.3
Cipher    : TLS_AES_256_GCM_SHA384
```

### 12. Confirm HTTP/2 is on

```
$ curl -sI --http2 https://example.com/ | head -1
HTTP/2 200
```

### 13. Confirm HTTP/3 is on

```
$ curl -sI --http3 https://example.com/ | head -1
HTTP/3 200
```

(Requires curl built with HTTP/3 support: `curl -V | grep -i http3`.)

### 14. Use nghttp to inspect HTTP/2

```
$ nghttp -nv https://example.com/ 2>&1 | grep -E 'recv|send'
[  0.073] send SETTINGS frame <length=12, flags=0x00, stream_id=0>
[  0.073] send HEADERS frame <length=42, flags=0x05, stream_id=1>
[  0.143] recv SETTINGS frame <length=24, flags=0x00, stream_id=0>
[  0.143] recv HEADERS frame <length=87, flags=0x04, stream_id=1>
[  0.143] recv DATA frame <length=612, flags=0x01, stream_id=1>
```

### 15. Load test with `ab` (ApacheBench, comes with apache-utils)

```
$ ab -n 10000 -c 100 http://localhost/
This is ApacheBench, Version 2.3 <$Revision: 1903618 $>
Benchmarking localhost (be patient)
Completed 1000 requests
Completed 2000 requests
...
Server Software:        nginx/1.26.0
Server Hostname:        localhost
Server Port:            80
Document Path:          /
Concurrency Level:      100
Time taken for tests:   0.713 seconds
Complete requests:      10000
Failed requests:        0
Requests per second:    14026.00 [#/sec] (mean)
Time per request:       7.130 [ms] (mean)
```

### 16. Load test with `wrk` (more modern)

```
$ wrk -t4 -c100 -d10s http://localhost/
Running 10s test @ http://localhost/
  4 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     2.41ms    1.07ms  20.34ms   88.94%
    Req/Sec     8.41k     1.04k   12.80k    72.50%
  335,442 requests in 10.00s, 269.31MB read
Requests/sec:  33543.27
Transfer/sec:     26.93MB
```

### 17. Load test HTTP/2 with `h2load`

```
$ h2load -n 10000 -c 100 -m 10 https://localhost/
finished in 1.07s, 9345.79 req/s, 1.91MB/s
requests: 10000 total, 10000 started, 10000 done, 10000 succeeded, 0 failed, 0 errored
status codes: 10000 2xx, 0 3xx, 0 4xx, 0 5xx
traffic: 2.05MB (2152400) total, 1.83MB (1922400) headers (space savings 60.74%)
```

### 18. Watch live request stats with `ngxtop`

```
$ sudo ngxtop -l /var/log/nginx/access.log
running for 5 seconds, 1234 records processed: 246.80 req/sec

Summary:
|   count |   avg_bytes_sent |   2xx |   3xx |   4xx |   5xx |
|---------|------------------|-------|-------|-------|-------|
|    1234 |          1842.18 |  1100 |    98 |    32 |     4 |

Detailed:
| request_path     |   count |   avg_bytes_sent |   2xx |   3xx |   4xx |   5xx |
|------------------|---------|------------------|-------|-------|-------|-------|
| /api/users       |     412 |          2103.40 |   400 |    10 |     2 |     0 |
| /                |     203 |          1500.00 |   200 |     0 |     3 |     0 |
| /static/app.js   |     180 |          8400.10 |   180 |     0 |     0 |     0 |
| /favicon.ico     |     112 |           153.00 |     0 |     0 |   112 |     0 |
```

### 19. Process the access log with `goaccess`

```
$ sudo goaccess /var/log/nginx/access.log -o report.html --log-format=COMBINED
Parsing... [|] [12,345 - 14,201/s]
Output: report.html
```

Open `report.html` in a browser to see graphs.

### 20. Live-tail status codes with awk

```
$ sudo tail -F /var/log/nginx/access.log | awk '{print $9}'
200
200
404
200
301
500
```

### 21. Get the most-hit URLs

```
$ sudo awk '{print $7}' /var/log/nginx/access.log | sort | uniq -c | sort -rn | head
   12041 /api/health
    8112 /
    6420 /static/app.js
    5811 /static/app.css
    4112 /api/users
    3890 /favicon.ico
```

### 22. Get certbot to install a Let's Encrypt cert and edit nginx.conf for you

```
$ sudo certbot --nginx -d example.com -d www.example.com
Saving debug log to /var/log/letsencrypt/letsencrypt.log
Plugins selected: Authenticator nginx, Installer nginx
Obtaining a new certificate
Performing the following challenges: http-01 challenge for example.com
Waiting for verification... Cleaning up challenges
Deploying Certificate to VirtualHost /etc/nginx/sites-enabled/example.com
Redirecting all traffic on port 80 to ssl in /etc/nginx/sites-enabled/example.com
Congratulations! You have successfully enabled https://example.com
```

### 23. Manual cert renewal (certbot does this automatically via cron/timer)

```
$ sudo certbot renew --dry-run
Cert not yet due for renewal
Cert not yet due for renewal
Congratulations, all renewals succeeded.
```

### 24. Show the systemd unit status

```
$ sudo systemctl status nginx --no-pager
* nginx.service - A high performance web server and a reverse proxy server
     Loaded: loaded (/lib/systemd/system/nginx.service; enabled; preset: enabled)
     Active: active (running) since Mon 2026-04-27 09:00:00 UTC; 1h ago
       Docs: man:nginx(8)
   Main PID: 1234 (nginx)
      Tasks: 5 (limit: 9425)
     Memory: 12.4M
        CPU: 14.382s
     CGroup: /system.slice/nginx.service
             |- 1234 "nginx: master process /usr/sbin/nginx ..."
             |- 1235 "nginx: worker process"
             |- 1236 "nginx: worker process"
             |- 1237 "nginx: worker process"
             `- 1238 "nginx: worker process"
```

### 25. Stop, start, restart, reload via systemd

```
$ sudo systemctl stop nginx
$ sudo systemctl start nginx
$ sudo systemctl restart nginx     # full restart, brief downtime
$ sudo systemctl reload nginx      # graceful, zero downtime
```

### 26. Stub-status for monitoring

Add to your config:
```
location = /nginx_status {
    stub_status;
    allow 127.0.0.1;
    deny all;
}
```

Then:
```
$ curl -s http://localhost/nginx_status
Active connections: 12
server accepts handled requests
 1421342 1421342 8123451
Reading: 0 Writing: 3 Waiting: 9
```

`Active connections` = total open connections. `accepts/handled/requests` = total counters since start. `Reading` = workers reading request headers. `Writing` = writing response. `Waiting` = idle keep-alive.

### 27. Open files limit (frequent gotcha at scale)

```
$ ulimit -n
1024
```

If you are doing 10k+ connections, bump this. nginx config:
```
worker_rlimit_nofile 65535;
```

### 28. Check what config file the running nginx loaded

```
$ ps -ef | grep "nginx: master" | grep -v grep
root  1234  1 0 09:00 ? 00:00:00 nginx: master process /usr/sbin/nginx -c /etc/nginx/nginx.conf
```

The `-c` argument shows the config file.

### 29. Reopen log files (after `logrotate` moved them)

```
$ sudo nginx -s reopen
```

This is what `logrotate` calls in its `postrotate` directive.

### 30. Build a docker container with nginx in five seconds

```
$ docker run --rm -d -p 8080:80 --name mynginx nginx:1.26
$ curl -s http://localhost:8080 | head -5
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
$ docker stop mynginx
```

### 31. Get into the running container and look around

```
$ docker exec -it mynginx /bin/sh
# nginx -t
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
# cat /etc/nginx/nginx.conf | head
user  nginx;
worker_processes  auto;
error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;
events {
    worker_connections  1024;
}
# exit
```

### 32. Trace what nginx is doing right now (Linux only)

```
$ sudo strace -f -p $(pgrep -f 'nginx: master') -e trace=network 2>&1 | head -20
strace: Process 1234 attached
[pid  1235] accept4(6, {sa_family=AF_INET, sin_port=htons(54321), sin_addr=inet_addr("192.0.2.10")}, [16], SOCK_NONBLOCK) = 9
[pid  1235] recvfrom(9, "GET / HTTP/1.1\r\nHost: localhost\r\n", 1024, 0, NULL, NULL) = 84
[pid  1235] writev(9, [...], 4) = 234
[pid  1235] close(9) = 0
```

Powerful for debugging. Stop with Ctrl-C.

## Common Confusions

Real things people get wrong, paired with the fix.

### 1. `root` vs `alias` — appended vs replaced

Broken:
```
location /static/ {
    alias /var/www/site/static;
}
```
Request `/static/app.js` becomes `/var/www/site/staticapp.js` (no slash). 404.

Fixed:
```
location /static/ {
    alias /var/www/site/static/;
}
```
Note the trailing slash. Or just use `root` if the disk path mirrors the URL: `root /var/www/site;` and let nginx append `/static/app.js`.

### 2. Forgot `proxy_set_header Host`

Broken: backend sees `Host: backend-name-from-upstream` because `proxy_pass` uses the upstream name as the Host. Some apps (like ones that key on Host) break.

Fixed:
```
proxy_set_header Host $host;
```

### 3. Forgot `proxy_http_version 1.1` for keep-alive or websockets

Broken: nginx talks HTTP/1.0 to the backend, opens a new TCP connection per request, and websockets do not work at all.

Fixed:
```
proxy_http_version 1.1;
proxy_set_header Connection "";
```

### 4. `proxy_pass` with vs without trailing slash

```
location /api/ {
    proxy_pass http://backend;        # request /api/foo  →  backend /api/foo
}

location /api/ {
    proxy_pass http://backend/;       # request /api/foo  →  backend /foo
}
```

The trailing slash on the proxy_pass URL is *load-bearing*. Without it, nginx forwards the entire URI. With it, nginx strips the matched prefix and forwards the rest.

### 5. `try_files` order matters

Broken (404 on every page in a SPA):
```
try_files /index.html $uri =404;
```
This returns `index.html` for every request, including JS/CSS — your app loads as HTML.

Fixed:
```
try_files $uri $uri/ /index.html;
```
Tries the file, then the directory, then falls back to `index.html`.

### 6. Forgot `always` on security headers

Broken:
```
add_header X-Frame-Options "DENY";
```
Header is missing on 404 and 500 pages — error pages can be framed.

Fixed:
```
add_header X-Frame-Options "DENY" always;
```

### 7. `server_name _` is not a wildcard

People think `server_name _;` matches everything. It does not. The underscore is a placeholder name that will never match a real Host header. You use it on a `default_server` block to mean "this server has no real name; just be the default."

### 8. Multiple `add_header` directives — only some apply

Broken:
```
server {
    add_header X-A "a";
    location / {
        add_header X-B "b";    # X-A is GONE here
    }
}
```
`add_header` does not stack across levels — the inner level *replaces* the outer level.

Fixed: repeat both at the inner level, or define them once at the outermost level that applies.

### 9. `if` is evil

`if` blocks in nginx are tricky and can cause unexpected behavior. The official nginx wiki literally has a page called "If is Evil." Most uses of `if` can be replaced with `try_files`, `map`, or different `location` blocks.

Broken (works but is fragile):
```
location / {
    if ($request_method = POST) { return 405; }
}
```

Fixed:
```
location / {
    limit_except GET HEAD { deny all; }
}
```

### 10. `rewrite` vs `return` for redirects

Broken (slower, more resources):
```
rewrite ^ https://$host$request_uri? permanent;
```

Fixed (zero-cost):
```
return 301 https://$host$request_uri;
```

### 11. Upstream resolved at startup (DNS pinning)

Broken:
```
upstream backend {
    server backend.local:3000;     # IP cached at startup; if DNS changes, nginx misses it
}
```

Fixed (re-resolve at runtime):
```
resolver 1.1.1.1 valid=30s;
set $backend "backend.local";
proxy_pass http://$backend:3000;
```

Or use NGINX Plus for `resolve` keyword on upstream.

### 12. `worker_connections` is per worker, not total

Broken (thinking 1024 means 1024 total):
```
worker_processes 4;
events { worker_connections 1024; }
```
This gives you 4 × 1024 = 4096 simultaneous connections. Plenty. People sometimes panic and set `worker_connections 100000` thinking they need to.

### 13. `gzip` does not compress already-compressed content

JPEG, PNG, MP4, ZIP, gzip files — already compressed. Trying to gzip them wastes CPU and produces no smaller output (sometimes even larger). Only put text MIME types in `gzip_types`. nginx defaults are conservative; the snippet in this sheet is correct.

### 14. SSL certificate file order

Broken (cert first, then intermediate):
```
ssl_certificate /etc/nginx/cert.pem;     # contains: cert + intermediate
```
But you put intermediate first. Browsers reject the chain.

Fixed: leaf cert first, then intermediates in chain order, ending with the root (which is optional). Just use `fullchain.pem` from Let's Encrypt — it does this correctly.

### 15. Using `proxy_pass` on a `location` with regex

Broken:
```
location ~ ^/api/(.*) {
    proxy_pass http://backend/$1;     # works, but...
}
```

Subtle bug: when `proxy_pass` is in a regex location, the URI is **not modified** — you must build the new URI yourself. Mostly fine, but it surprises people coming from prefix-location habits.

### 16. Trying to set `Cache-Control` from inside the cached response

The browser-facing `Cache-Control` and the upstream `Cache-Control` are different things. Use `expires` and `add_header Cache-Control` for the browser. Use `proxy_cache_valid` or `proxy_cache_use_stale` for nginx's own cache.

## Vocabulary

| Term | Plain English |
| ---- | ------------- |
| nginx | Polite receptionist sitting in front of your web app, taking and routing every HTTP request. |
| HTTP | The protocol web browsers use to talk to web servers. Request → response. |
| HTTPS | HTTP encrypted using TLS. Same protocol, scrambled in transit. |
| TLS | The encryption layer underneath HTTPS. Was called SSL in older versions. |
| SSL | Old name for TLS. The "S" in HTTPS still stands for it, but everyone uses TLS now. |
| Web server | A program that listens for HTTP requests and answers them. |
| Reverse proxy | Something that sits in front of one or more backends and forwards requests to them. |
| Forward proxy | The opposite of a reverse proxy. Sits in front of clients (your browser). Squid is one. |
| Backend | The "real" application behind nginx — your Python/Node/Go/Java app. |
| Upstream | nginx's word for "the backend." Configured in `upstream { }` blocks. |
| Frontend | What the user sees in their browser; or the server-side thing serving HTML, depending on context. |
| Load balancer | Spreads incoming requests across multiple identical backends. |
| Static file | A file on disk (HTML, JS, CSS, image) that nginx can serve directly without involving any app. |
| Dynamic content | A response that requires running code (database queries, computation) — handled by the backend. |
| Cache | A copy of an answer kept around so we do not have to recompute it. |
| Cache hit | The cache had the answer, no backend call needed. |
| Cache miss | The cache did not have the answer, the backend was called and answered. |
| Cache stampede | A bunch of identical uncached requests slamming the backend at once. |
| Microcaching | Caching for very short times (1-10 seconds) to absorb traffic spikes. |
| Worker process | A child process spawned by the master that actually serves traffic. |
| Master process | The boss process that reads config, opens ports, manages workers. |
| Event loop | A loop that watches a list of events (sockets) and reacts to whichever is ready. |
| epoll | Linux system call that lets one process watch tens of thousands of sockets efficiently. |
| kqueue | BSD/macOS equivalent of epoll. |
| C10K | The "ten thousand concurrent connections" problem nginx was created to solve. |
| Process model | How a server organizes processes/threads to handle requests. |
| Thread per connection | Apache's classic mode: one whole thread blocked per connection. |
| Event-driven | nginx's mode: one thread, many connections, react to events. |
| Configuration file | The text file that tells nginx what to do. Usually `/etc/nginx/nginx.conf`. |
| Directive | One line of nginx config, ending in a semicolon. |
| Block | A `{ }` section of nginx config containing directives. |
| Context | The kind of block (`http`, `server`, `location`, etc.). |
| Main context | Top-level config, outside any block. |
| `events` block | Controls connection-handling settings, mostly `worker_connections`. |
| `http` block | Contains everything related to HTTP. Most config goes inside this. |
| `server` block | One virtual host (one website). |
| `location` block | A URL path or pattern handler. |
| `upstream` block | A pool of backend servers for load balancing. |
| `stream` block | Like `http` but for raw TCP/UDP, not HTTP. |
| `mail` block | For nginx's mail-proxy mode (rarely used). |
| Virtual host | One website served by nginx. Same machine, different domains. |
| Server name | The domain a `server` block answers to (`server_name example.com;`). |
| `listen` | Which port (and IP) a `server` block accepts connections on. |
| Default server | The `server` block used when no other matches; marked `default_server`. |
| Location matching | The order nginx uses to pick which `location` block handles a request. |
| `=` (location prefix) | Exact match, fastest. |
| `^~` (location prefix) | Prefix match that wins over regex if it matches. |
| `~` (location prefix) | Case-sensitive regex match. |
| `~*` (location prefix) | Case-insensitive regex match. |
| Prefix match | nginx looks for the longest matching path prefix. |
| Regex match | nginx evaluates regular expressions against the URI. |
| `root` | nginx directive that prepends to the URI to find a file. |
| `alias` | nginx directive that replaces the matched prefix to find a file. |
| `try_files` | Tries multiple files in order; first existing one wins. |
| `index` | What file to serve when a directory is requested (e.g., `index.html`). |
| `autoindex` | Show a file listing for a directory. |
| `proxy_pass` | Forward the request to a backend. |
| `proxy_set_header` | Add or override a header before forwarding. |
| `proxy_http_version` | Which HTTP version to use to talk to the backend. |
| Round robin | Default load-balance strategy: rotate through backends in order. |
| Least connections | Send to backend with fewest active connections. |
| IP hash | Hash the client IP and always route the same client to the same backend. |
| Sticky session | Keeping the same client on the same backend (often via `ip_hash`). |
| Weight | Relative capacity of a backend in the upstream pool. |
| Health check | Probe to decide whether a backend is healthy. |
| Passive health check | Mark unhealthy after N failed real requests (`max_fails`/`fail_timeout`). |
| Active health check | Periodically probe a backend (NGINX Plus or third-party module). |
| Backup server | A backend used only when all others are down. |
| Keep-alive | Reuse a TCP connection for multiple requests. |
| `keepalive` (upstream) | Number of idle backend connections to keep around per worker. |
| TLS handshake | The negotiation between client and server that sets up encryption. |
| Certificate | A signed document proving the server is who it claims to be. |
| Private key | The secret half of the TLS keypair. Never share. |
| Full chain | Cert + intermediates concatenated, in chain order. |
| Let's Encrypt | Free, automated certificate authority. |
| certbot | Tool that gets and renews Let's Encrypt certs. |
| OCSP | Online Certificate Status Protocol — way to check if a cert was revoked. |
| OCSP stapling | Server attaches the OCSP response to its handshake so the client does not have to ask. |
| HSTS | Strict-Transport-Security header; tells browsers to use HTTPS for this domain. |
| HTTP/1.1 | Original modern HTTP. One request at a time per connection (with pipelining quirks). |
| HTTP/2 | Multiplexed binary HTTP. Many requests over one TCP connection. |
| HTTP/3 | HTTP over QUIC, which runs over UDP. Avoids TCP head-of-line blocking. |
| QUIC | UDP-based transport protocol underlying HTTP/3. |
| ALPN | TLS extension that negotiates the application protocol (HTTP/2 vs HTTP/1.1) during the handshake. |
| SNI | Server Name Indication — TLS extension that lets a client say which hostname it wants. |
| `proxy_cache` | nginx directive that enables caching of upstream responses for a location. |
| `proxy_cache_path` | Where on disk to keep the cache; size limits and shared zones. |
| `proxy_cache_valid` | How long to cache by status code. |
| `proxy_cache_use_stale` | Serve a stale cached copy if backend is broken. |
| `proxy_cache_lock` | Only one request to backend at a time for the same cache key. |
| Cache key | The string used to look up an entry in the cache (default: `$scheme$proxy_host$request_uri`). |
| Stale-while-revalidate | Serve a stale copy while updating in the background. |
| gzip | Compression algorithm; nginx applies it to responses. |
| brotli | Newer compression; ~20% better than gzip for text. |
| `gzip_types` | Which MIME types to compress. |
| `gzip_min_length` | Skip compression for tiny responses. |
| `gzip_proxied` | Whether to compress responses from a proxied backend. |
| Rate limit | Cap on how many requests a client can make per unit time. |
| `limit_req_zone` | Defines shared memory zone for rate limiting. |
| `limit_req` | Applies a rate limit to a location. |
| `burst` | How many extra requests can queue beyond the rate. |
| `nodelay` | Burst requests served immediately, not paced. |
| `limit_conn_zone` | Defines a connection-count limit zone. |
| `limit_conn` | Limit on simultaneous connections per key (usually IP). |
| 429 Too Many Requests | The "you're rate-limited" status code. |
| Access log | One line per request, what was served. |
| Error log | nginx's own diagnostic messages. |
| `log_format` | Defines how access log lines are formatted. |
| `escape=json` | JSON-escape variable values in log lines. Critical for safe JSON logs. |
| Log rotation | Move old logs aside so the active log file does not grow forever. |
| `nginx -s reload` | Tell master to reload config without dropping connections. |
| `nginx -s reopen` | Close and reopen log files (after rotation). |
| `nginx -s stop` | Fast shutdown. |
| `nginx -s quit` | Graceful shutdown. |
| `nginx -t` | Test config syntax without applying. |
| `nginx -T` | Test and dump the full effective config. |
| `nginx -V` | Show version and compiled-in modules. |
| Module | A piece of nginx functionality, statically or dynamically compiled. |
| Static module | Compiled into the nginx binary. |
| Dynamic module | A `.so` file loaded with `load_module`. |
| `load_module` | Top-level directive to load a dynamic module. |
| OpenResty | nginx + Lua + many modules — a "batteries included" nginx distro. |
| WAF | Web Application Firewall — module like ModSecurity that blocks attack patterns. |
| `server_tokens off` | Hides the nginx version from headers and error pages. |
| `client_max_body_size` | Maximum request body size; default 1MB. |
| `client_body_timeout` | How long to wait for the request body. |
| `keepalive_timeout` | How long to keep an idle connection open. |
| `proxy_read_timeout` | Max time to wait for a backend response (default 60s). |
| `proxy_connect_timeout` | Max time to wait for backend TCP handshake. |
| 502 Bad Gateway | nginx tried to talk to backend, got nothing usable. |
| 504 Gateway Timeout | Backend was too slow. |
| 413 Request Entity Too Large | Body bigger than `client_max_body_size`. |
| 444 | nginx-specific code: drop the connection without responding. |
| `stub_status` | Built-in module exposing `/nginx_status` counters for monitoring. |
| Prometheus exporter | Sidecar that scrapes stub-status (or VTS module) and exposes Prometheus metrics. |
| `ngxtop` | Command-line tool that summarizes access log live. |
| `goaccess` | Command-line tool that generates HTML reports from access logs. |
| `wrk` | Modern HTTP load-testing tool. |
| `ab` | ApacheBench — older HTTP load-testing tool. |
| `h2load` | HTTP/2 load tester from the nghttp2 project. |
| `nghttp` | HTTP/2 client and protocol debugger. |
| `curl` | Universal HTTP client; the "swiss army knife" of web debugging. |
| `openssl s_client` | OpenSSL's TLS client for inspecting handshakes. |
| `ss` | Linux socket-status tool, replaces `netstat`. |
| `journalctl` | systemd's log viewer. |
| systemd unit | The systemd file that defines how nginx starts/stops. |
| PID file | File holding the master process's PID; used to send signals. |
| Signal | A short message sent to a process (HUP, USR1, USR2, QUIT, TERM). |
| SIGHUP | Reload config (in nginx). |
| SIGUSR1 | Reopen log files. |
| SIGUSR2 | Binary upgrade. |
| SIGQUIT | Graceful shutdown. |
| SIGTERM | Fast shutdown. |
| Binary upgrade | Live-replace the nginx binary on disk without dropping connections. |
| `worker_processes` | How many worker processes to run; usually `auto` (one per core). |
| `worker_connections` | Max connections per worker. |
| `worker_rlimit_nofile` | Per-worker max open files (raise for high concurrency). |
| `sendfile` | Fast file-to-socket copy via the `sendfile()` syscall. |
| `tcp_nopush` | Avoid sending small TCP packets; only effective with `sendfile on`. |
| `tcp_nodelay` | Disable Nagle's algorithm; send small packets immediately. |
| `resolver` | DNS server nginx uses for runtime DNS lookups (e.g., dynamic upstreams). |
| `map` | Define a variable based on the value of another variable. |
| `set` | Set a custom variable. |
| `$host` | The Host header (or `server_name` if no Host). |
| `$remote_addr` | The client's IP. |
| `$remote_user` | Authenticated user (basic auth). |
| `$request_uri` | The full original request URI (path + query). |
| `$uri` | The current URI (after rewrites). |
| `$args` | The query string. |
| `$scheme` | `http` or `https`. |
| `$http_<header>` | Any incoming request header (e.g., `$http_user_agent`). |
| `$upstream_addr` | Which backend served this request. |
| `$upstream_response_time` | How long the backend took. |
| `$request_time` | Total time nginx spent on this request. |
| `$upstream_cache_status` | `HIT`, `MISS`, etc. |
| OOM | Out of memory; the kernel killed something. |
| `$binary_remote_addr` | Client IP in 4-byte (IPv4) or 16-byte (IPv6) form, used for rate-limit keys. |
| Connection upgrade | Switching protocols mid-connection (e.g., HTTP → WebSocket). |
| WebSocket | Bidirectional persistent connection over HTTP. |
| Protocol | A set of rules for two computers to talk. |
| Layer 4 | Transport layer — TCP/UDP. |
| Layer 7 | Application layer — HTTP, etc. |
| L4 load balancer | Routes by IP/port only (HAProxy in TCP mode, nginx `stream`). |
| L7 load balancer | Routes by HTTP-level info (URL, headers, method) — nginx default. |
| HAProxy | Another popular load balancer/reverse proxy. Stronger at pure TCP, weaker than nginx for static content. |
| Caddy | Another web server. Simpler config, automatic Let's Encrypt by default. |
| Apache HTTPD | The old-school web server. Rich `.htaccess` support. |
| `.htaccess` | Apache's per-directory config file. nginx does not have these (intentionally). |
| MIME type | A label like `text/html` or `image/png` that says what kind of file something is. |
| `Content-Type` | The HTTP header that carries the MIME type. |
| Header | Key/value metadata in an HTTP request or response. |
| Body | The actual payload of an HTTP request or response. |
| Status code | Three-digit code that summarizes the response (200 OK, 404 Not Found, 500 Server Error). |
| Idempotent | An operation safe to repeat (GET is idempotent; POST is not). |
| FastCGI | Old protocol for talking to PHP/Python apps. nginx supports it via `fastcgi_pass`. |
| uwsgi protocol | Like FastCGI but used by Python's uwsgi server. nginx supports via `uwsgi_pass`. |
| gRPC | Google's RPC over HTTP/2. nginx supports via `grpc_pass`. |
| `realip` module | Reads `X-Forwarded-For`/`X-Real-IP` and uses it as `$remote_addr` (when nginx is itself behind another proxy). |
| `set_real_ip_from` | Defines which upstream proxies to trust for the realip module. |

## Try This

Five small experiments. None of them take more than a few minutes.

### 1. Hello world

Write the eight-line config from this sheet and serve it on port 8080. Hit it with `curl`. Verify the body says `hello from nginx`.

### 2. Add a second `server` block

Make a new `server` block with `listen 8081;` that returns `hello from server B`. Reload nginx. Confirm both ports work.

### 3. Reverse proxy a Python one-liner

In one terminal: `python3 -m http.server 9000`. In nginx, add:
```
server {
    listen 8082;
    location / { proxy_pass http://127.0.0.1:9000; }
}
```
Reload, then `curl http://localhost:8082/`. You are reverse-proxying.

### 4. Watch the workers reload

```
ps -ef | grep nginx
sudo nginx -s reload
ps -ef | grep nginx
```
Note the PIDs change for workers. The master PID stays the same.

### 5. Add gzip and watch the response shrink

```
curl -sI http://localhost:8082/ -H 'Accept-Encoding: gzip' | grep -i content-encoding
```
First time, no `Content-Encoding`. Add `gzip on;` and `gzip_types text/html text/plain application/json;` to your config. Reload. Repeat the curl. Now you get `Content-Encoding: gzip`.

### 6. Cache something for 5 seconds

Set up a `proxy_cache_path` and a `proxy_cache_valid 200 5s;`. Make 10 requests in 1 second. Check the response time and the `X-Cache-Status` header — first one is `MISS`, next nine are `HIT`. Wait 6 seconds, the next is `EXPIRED → MISS`.

### 7. Break it on purpose

Edit the config, type `lisetn 80;` (typo). Run `nginx -t`. Read the error message. Fix it. Run `nginx -t` again. Reload.

## Where to Go Next

You now know what nginx is and how to make it do basic things. From here:

- Walk through the production cheat sheet at `web-servers/nginx` for compact reference (no analogies, just answers).
- Read `ramp-up/tls-eli5` to deeply understand the encryption layer that nginx terminates.
- Read `ramp-up/http3-quic-eli5` for the next-gen transport.
- Read `ramp-up/tcp-eli5` for the layer underneath everything.
- Read `web-servers/haproxy` if you ever need pure TCP load balancing or the most-flexible L7 routing.
- Read `web-servers/caddy` if you want simpler config and automatic certs.
- When containerizing, read `ramp-up/docker-eli5` and combine: `docker run nginx`.

## Version Notes

- **1.9.5** (2015): HTTP/2 support added.
- **1.13.10** (2018): TLS 1.3 support.
- **1.18** (2020): Long-Term Support (LTS) branch.
- **1.25** (2023): HTTP/3 (QUIC) marked stable. New `http2 on;`/`http3 on;` syntax.
- **1.26** (2024): Current LTS. Default for production.
- **1.27+**: mainline; new features land here first, then back-port to 1.28 LTS.

If your distro ships 1.18 or older, consider the official nginx repos (`nginx.org/en/linux_packages.html`) for newer versions.

## See Also

- [web-servers/nginx](../web-servers/nginx.md)
- [web-servers/haproxy](../web-servers/haproxy.md)
- [web-servers/caddy](../web-servers/caddy.md)
- [networking/http](../networking/http.md)
- [networking/http2](../networking/http2.md)
- [networking/http3](../networking/http3.md)
- [security/tls](../security/tls.md)
- [ramp-up/tls-eli5](tls-eli5.md)
- [ramp-up/http3-quic-eli5](http3-quic-eli5.md)
- [ramp-up/tcp-eli5](tcp-eli5.md)
- [ramp-up/docker-eli5](docker-eli5.md)
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md)

## References

- nginx official documentation: https://nginx.org/en/docs/
- nginx beginner's guide: https://nginx.org/en/docs/beginners_guide.html
- nginx admin guide: https://docs.nginx.com/nginx/admin-guide/
- "NGINX Cookbook" by Derek DeJonghe (O'Reilly, 3rd edition 2022)
- "NGINX HTTP Server" by Clement Nedelcu (Packt, 4th edition)
- Mozilla SSL Configuration Generator: https://ssl-config.mozilla.org/
- "If is Evil" wiki page: https://www.nginx.com/resources/wiki/start/topics/depth/ifisevil/
- Pitfalls and Common Mistakes: https://www.nginx.com/resources/wiki/start/topics/tutorials/config_pitfalls/
- nginx source code: https://github.com/nginx/nginx
- OpenResty (nginx + Lua): https://openresty.org/
- The C10K problem (Dan Kegel, 1999): http://www.kegel.com/c10k.html
