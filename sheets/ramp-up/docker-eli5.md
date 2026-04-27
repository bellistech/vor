# Docker — ELI5 (Shipping Containers for Software)

> Docker is shipping containers for software: pack your program once, run it anywhere, and stop saying "it works on my machine."

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` helps; containers are just namespaces + cgroups in a trench coat)

You do not need to know what a container is. You do not need to know what a kernel is. You do not need to be a "Linux person." If you have ever copied a file, opened a terminal, or even just looked at one over somebody's shoulder, you have enough to read this sheet.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Docker?

### Imagine the world before shipping containers

A long time ago, before the 1950s, ships did not carry shipping containers. Every ship was loaded by hand. Workers carried boxes, sacks of grain, barrels of oil, crates of fruit, bundles of clothes, and big awkward machines onto the ship one piece at a time. Some of those things were heavy. Some were fragile. Some were the wrong shape. Some leaked. Some smelled. Loading one ship took **days.** Sometimes weeks. The boxes on the bottom got crushed. Boxes on the top fell off. Stuff got lost. Stuff got stolen. Stuff got rained on. Half of it arrived broken.

When the ship finally got across the ocean, the workers on the other side had to do the same thing in reverse. Crane hooks. Sweat. Shouting. Splinters. The food on the docks rotted while it waited to be sorted. Sailors got injured. Insurance was a nightmare. Ports could only handle a few ships at a time, so ships sat in line for weeks waiting their turn to dock.

This is exactly what software was like for a long time too. You had a program that worked great on your laptop. You wanted to run it on a different computer — maybe a server, maybe your friend's laptop, maybe a customer's machine. So you tried to copy it over. But the other computer had a different version of the operating system. A different version of Python. A different version of OpenSSL. A weird old library. A missing font. The wrong time zone. Some little file in the wrong place.

So you spent the next three days figuring out why your program wouldn't start. You read error messages. You changed config files. You installed packages. You uninstalled packages. You re-installed packages. You rebooted. You re-formatted. You filed a support ticket. You cried into your keyboard. Eventually, somehow, you got it working. Then you had to do it all over again on the next computer. **And the next.** And the next.

This sucked. Everybody hated it. People literally said "it works on my machine" so often that it became a joke. There were stickers and t-shirts and memes. Companies wasted billions of dollars a year on this exact problem.

### Then somebody invented the shipping container

In 1956 a man named Malcolm McLean invented the modern shipping container. The idea was almost insultingly simple: **put everything in identical metal boxes.** Same dimensions. Same locking corners. Same crane hooks. Same truck-bed mounts. Same train-car mounts. Same stack rules.

Now nobody had to care what was *inside* the box. The crane just grabbed the box. The truck just held the box. The ship just stacked the box. The other crane just lifted the box off. Loading went from days to hours. Stuff stopped getting broken. Theft dropped. Insurance got cheap. Ports could process tens of times more cargo. Shipping prices fell off a cliff. International trade exploded.

The genius of shipping containers is **standardization.** Once everybody agrees on the shape and size of the box, the box can travel through any port, on any ship, on any truck, on any train, anywhere in the world, without anyone needing to repack it or even know what's inside it.

**Docker is shipping containers for software.**

That is the entire idea. You take your program, plus everything it needs to run — the right version of Python, the right libraries, the right config files, the right fonts, the right system tools, the right time zone — and you pack it all into a standard-shaped box called an **image.** Then you can ship that box anywhere Docker is installed and it just works. The box has the same shape on your laptop, on your colleague's laptop, on the test server, on the production server, on the customer's cloud account, on a Raspberry Pi sitting on a windowsill in Iceland. Same box. Same behavior. No drama.

The host computer doesn't have to know what's *inside* the box. It doesn't need to install Python. It doesn't need to install the libraries. It doesn't even need to know what programming language your program is written in. It just needs Docker. Docker grabs the box, plops it down, and lets the program inside run.

When somebody asks "does it work on your machine," with Docker the answer is "of course it works on my machine — and yours, and the boss's, and the customer's, and the cloud's." Because everybody is running the same box. The same standard-sized metal box. There is nowhere for differences to hide.

### Why this matters more than it sounds

It is hard to overstate how much this changed things. Before Docker, deploying a website might involve a 40-page document of "first install this, then install that, then run this script, then edit this file, then run this other script, then pray." After Docker, it is `docker run`. One command. Done.

Before Docker, having ten copies of your test environment was a nightmare. After Docker, you start ten containers. They are all identical. They are all isolated. None of them step on each other.

Before Docker, upgrading a server meant scheduling downtime, doing the upgrade in-place, and hoping nothing broke. After Docker, you build a new image, start a new container with it, point traffic at the new one, and shut down the old one. If something goes wrong, you point traffic back at the old container. Rollback in seconds.

Before Docker, "matching production" was an endless quest. After Docker, your laptop is bit-for-bit identical to production, because they're both running the same image. The image is the truth.

Whole categories of bug just disappear. Whole categories of meeting just disappear. Whole categories of stress just disappear. That is the gift of standardization, and it is why Docker took over the world in less than five years.

### A second picture: lunchboxes

If shipping containers feel too industrial, here is another picture.

Imagine you go to school every day. Every kid brings their lunch. Some kids bring lunchboxes. Some kids bring paper bags. Some kids bring plastic containers. Some kids bring loose stuff in their backpack and it spills everywhere. The cafeteria is a chaos of crumbs and squashed sandwiches and yogurt that leaked onto homework.

One day the school says: **everybody must bring lunch in this exact lunchbox.** Same size, same shape, same lid, stackable, sealed, washable. Now the cafeteria is clean. Lunchboxes stack neatly. They fit in everybody's locker. Cleanup takes two minutes. Nobody's lunch leaks onto somebody else's lunch.

Inside your lunchbox, you can have whatever you want. Sandwich. Salad. Pizza. Sushi. Whatever. The lunchbox doesn't care. It just keeps your stuff together and keeps it from touching everybody else's stuff.

A Docker image is the lunchbox. What is inside the lunchbox is your program plus everything it needs. The standardized lunchbox is what makes everything else clean and easy.

### A third picture: portable rooms

Here is one more picture, because some people prefer this one.

Imagine you could pack up your entire bedroom into a magic suitcase. Bed, lamp, books, posters, your favorite mug, the smell of your candle, the exact arrangement of pencils on your desk. You snap the suitcase shut. You take it on a plane. You arrive at a hotel. You open the suitcase. **Your whole room appears.** Not a hotel room with similar stuff — your actual room, exactly as you left it.

That is what a Docker image gives a program. The program packs up its whole "room" — the operating system files it expects, the libraries, the configs, the right version of everything. When the image runs as a container, the program looks around and sees its room exactly the way it expects. The hotel doesn't matter. The plane didn't matter. The room is the room.

That is why "it works on my machine" stops being a problem. Everybody is in the same room.

### Why "Docker" is a slightly weird name

The name is a pun, sort of. Docks are where shipping containers come and go. The thing that handles dock work is a docker — that's a real English word for a dock worker. So the program that handles the software shipping containers is "Docker." Once you know it's a shipping-container metaphor, the name makes sense.

The Docker logo is a whale carrying shipping containers on its back. That whale's name is **Moby Dock.** Yes, like Moby Dick. Yes, that is silly. The pieces of Docker that got open-sourced over the years live in a project called **Moby**, after the whale.

## Image vs Container

This is the single most confusing thing for people who are new to Docker, so we are going to spend a lot of time on it. There are two words that sound similar and people use them interchangeably even though they mean very different things.

- An **image** is a recipe.
- A **container** is a meal made from the recipe.

Or:

- An **image** is a class.
- A **container** is an instance of that class.

Or:

- An **image** is a shipping container's blueprint and packed contents sitting in a warehouse.
- A **container** (lowercase, the running thing) is when somebody actually opens that box and starts using what's inside.

Or:

- An **image** is a saved video game file on disk.
- A **container** is the game actually running on your console.

You can run **many containers from one image.** That is the whole point. The image is the template, and you can stamp out as many copies of it as you want. Each copy runs independently. Each copy has its own running state, its own writable filesystem, its own network connection, its own crashes and successes. But they all started from the same recipe.

### Walking through it slowly

Imagine you have a recipe for cookies. The recipe is a piece of paper. The recipe doesn't bake any cookies on its own. The recipe just sits there. You can put the recipe in a binder. You can email the recipe to a friend. You can print ten copies of the recipe. You can stick the recipe to your fridge. None of those things produce any cookies. The recipe is **frozen, read-only, just data.**

That is an image.

Now you actually go to the kitchen. You read the recipe. You measure the flour. You crack the eggs. You mix. You scoop. You bake. You take cookies out of the oven. **The cookies are real things.** They are warm. They smell good. They have weight. You can eat them. They will eventually go stale or get eaten or thrown away.

That is a container.

The recipe didn't change while you baked. You can use the recipe again tomorrow to bake another batch. Your friend can use the same recipe in their kitchen to bake their own batch. The two batches are different physical objects. They were made from the same recipe but they are not the same cookies.

This is exactly how Docker works. The image is read-only. The image stays the same forever (well, until you build a new version). You can run the image as a container as many times as you want. Each run produces its own container, with its own life, its own state, its own little world. When the container exits, the cookies are eaten — but the recipe is still on the fridge.

### Another example: a class and an object

If you have done any programming, this might click faster: image = class, container = object (instance). The class definition lives in a file. The class doesn't do anything by itself. When you call `new Foo()` you get an object, an instance. You can call `new Foo()` ten times to get ten objects. They all started from the same class definition but they are now ten separate things.

Image is the class. `docker run image` is `new`. The container is the object.

### Yet another: video games

A saved game file on your hard drive is just bytes. It is not playing. It is just a frozen blob. When you load that save into the console and start playing, the game is running. You are inside it. Time is passing. Things are happening. The save file didn't move. The save file is still on disk. But there is now a *play session* alive in memory based on it.

Image = save file. Container = play session. You can load the same save file ten times in a row and play ten different times. Same starting point, different runs.

### The naming gets confusing

Sadly the world uses the word "container" to mean two slightly different things. Sometimes "a container" means **a Docker container, a running process based on an image.** Sometimes "a container" means **a generic Linux thing, a process with namespaces and cgroups around it.** Docker containers are one specific kind of Linux container. There are also LXC containers, podman containers, containerd containers, and so on. They are all the same idea: a sandboxed running process with its own filesystem view. We will mostly say "container" and trust that you know we mean the running thing.

### A drawing

```
                IMAGE                              CONTAINER
            (read-only template)            (running, writable instance)

         +---------------------+              +-----------+
         |  hello-world:1.0    |              | abcd1234  |  <-- container
         |---------------------|   docker run | my program|      from this image
         |  Layer: alpine      |   --------> |  is going  |     (writable layer
         |  Layer: curl        |              |  brrr.    |      on top)
         |  Layer: hello.sh    |              +-----------+
         |  CMD: hello.sh      |              +-----------+
         +---------------------+              | ef567890  |  <-- another container
              one image                       | also brrr  |     same image,
              many containers                 |            |     different state
                                              +-----------+
                                              +-----------+
                                              | 99999999  |  <-- yet another
                                              | brrrrrrrr  |
                                              +-----------+
```

One image, many containers. Each container gets a unique ID. Each container has its own running state. They all share the read-only layers underneath, but each gets its own scratch space on top.

### The lifecycle of an image and a container

An image is **built once** and **shipped many times.** You write a Dockerfile. You run `docker build`. The image is now sitting on your machine. You can `docker push` it to a registry, where everybody else can `docker pull` it. The image is essentially permanent. You can list it, inspect it, delete it. It does not "do" anything on its own.

A container is **started, lives, and exits.** You run `docker run image`. A container appears. It runs the program inside. The program does its thing. Eventually the program ends or you kill it. The container exits. By default, exited containers stick around in a "stopped" state until you `docker rm` them. If you used `--rm` they delete themselves on exit. While they are running, they have logs, processes, resource usage. When they exit, they leave behind logs and an exit code, but no live processes. You can restart a stopped container with `docker start`. You can re-create it from the image with `docker run` again.

### What "writable layer" means

We will talk about layers more in the next section, but here is the headline: a container has all the layers of its image stacked underneath, plus **one extra writable layer on top.** When the program inside the container writes a file, the file goes into the writable layer. The image itself is never modified. When the container exits, the writable layer is thrown away unless you explicitly committed it to a new image (which is rare and considered a bit of a smell).

This is why people say "containers are ephemeral." Anything they write to their own filesystem disappears with them. If you want data to survive, you put it in a **volume** or a **bind mount**, which we will get to.

## Layers and the Filesystem

Now we get to one of the prettiest ideas in Docker: layers.

Imagine you are stacking sheets of see-through plastic on a desk. The bottom sheet has some words written on it. You stack a second sheet on top, with some new words. You can still see the words from the bottom through the plastic. You stack a third sheet on top, with words that **cover up** some of the words from the bottom. You stack a fourth sheet on top with even more words.

If you look at the stack from above, you see a single image — the union of all the sheets, with later sheets covering earlier sheets. That is exactly what Docker does with the filesystem.

### Each instruction is a layer

When you write a Dockerfile, every instruction creates a layer. The base image (the `FROM` line) is the first layer. Each `RUN`, `COPY`, and `ADD` adds another layer on top. Layers stack from bottom to top, and the **top layer wins** when files conflict.

Here is a tiny Dockerfile:

```dockerfile
FROM alpine:3.19
RUN apk add --no-cache curl
COPY hello.sh /usr/local/bin/
CMD ["hello.sh"]
```

That builds an image with these layers, from bottom to top:

```
+-------------------------------+
| Layer 4: CMD metadata only    |  <-- not really filesystem; it's image config
+-------------------------------+
| Layer 3: COPY hello.sh        |  <-- adds /usr/local/bin/hello.sh
+-------------------------------+
| Layer 2: RUN apk add curl     |  <-- adds curl + its deps to /usr/bin etc.
+-------------------------------+
| Layer 1: FROM alpine:3.19     |  <-- adds /bin, /etc, /lib... a tiny linux
+-------------------------------+
```

When the container starts, Docker stitches all these layers into a single coherent filesystem using something called **OverlayFS** (the union filesystem). The container looks down and sees one filesystem with `/bin`, `/etc`, `/usr/bin/curl`, `/usr/local/bin/hello.sh`, all the right pieces, even though physically they live in different layers.

If layer 3 wrote to `/etc/hosts`, that copy would shadow whatever was in layer 1's `/etc/hosts`. The new file wins. This is called a **copy-on-write** stack: changes happen on top, never to the layers below.

### Why layers are amazing

Layers are not just a clever filing trick. They give Docker three superpowers.

**Superpower 1: Caching.**

If you build the same Dockerfile twice, Docker says "I already built this layer, here it is." It reuses the cached layer instead of rebuilding it. This means the second build is way faster than the first. If you only changed the last line of the Dockerfile, only the last layer rebuilds; everything underneath is already cached.

This is why people order Dockerfiles deliberately. You put the slow, rarely-changing stuff at the top (install OS packages, install language runtime, install dependencies) and the fast, frequently-changing stuff at the bottom (copy your source code). That way, when you change your source code, only the last layer rebuilds. Your daily edit cycle goes from "wait three minutes" to "wait two seconds."

**Superpower 2: Sharing across images.**

If you build ten different images all based on `FROM alpine:3.19`, the `alpine:3.19` layer is stored on disk **exactly once.** All ten images point to it. You don't pay for it ten times. The same is true for any layer that's identical across multiple images.

This is also true when pulling. If you `docker pull foo:latest` and then `docker pull bar:latest`, and they both share some base layers, those base layers only get downloaded once. Docker keeps a content-addressable store of layers, keyed by their hash. Same content = same hash = same layer = stored once.

When pushing, only the layers your registry doesn't already have get uploaded. That is why pushing a new version of an image after a tiny code change is fast: only the top layer (the changed source code) needs to go over the wire.

**Superpower 3: Reproducibility.**

Each layer is identified by a hash of its contents. If two people build "the same" Dockerfile and get different bytes, they will produce different hashes and they will be able to tell. Layers are immutable. Once a layer is built, its contents never change. If you want different contents, you build a new layer with a different hash.

### A picture of layer sharing

```
   IMAGE A (myapp:1.0)            IMAGE B (myapp:1.1)         IMAGE C (otherapp:7)

   +-----------------+            +-----------------+         +-----------------+
   | app code v1.0   |            | app code v1.1   |         | totally diff app|
   +-----------------+            +-----------------+         +-----------------+
   | npm install     |  <-shared- | npm install     |         | pip install     |
   +-----------------+            +-----------------+         +-----------------+
   | node:20-alpine  |  <-shared- | node:20-alpine  |         | python:3.11     |
   +-----------------+            +-----------------+         +-----------------+
                                                              | alpine:3.19     |
                                                              +-----------------+

   shared layers stored ONCE on disk.    only the top layer differs between A and B,
   only that one downloads on pull.      so the "diff" is tiny.
```

Three images, lots of overlap, total disk usage is way less than the sum of the parts.

### How OverlayFS actually merges layers

You don't strictly need to know this, but it's cool. OverlayFS (the default Docker storage driver on modern Linux) takes a list of "lower" directories and one "upper" directory and presents a merged view. The lower directories are read-only — they are your image's layers. The upper directory is writable — that is the container's writable layer. When the container reads a file, OverlayFS searches top-down: upper, then layer N, then layer N-1, and so on. The first hit wins. When the container writes a file, OverlayFS copies the file from wherever it found it down in the lowers up into the upper, then modifies the upper copy. The lowers never change. This is called **copy-up.**

Deletes are tricky. To delete a file that exists in a lower layer, OverlayFS creates a tiny "whiteout" marker in the upper layer that says "this path is gone." Reads see the whiteout and pretend the file doesn't exist.

The result is what we already said: containers see one coherent filesystem, image layers stay immutable, and the writable scratch space is just the upper directory.

### Cache invalidation: the fragile part

Layer caching is great, but it can also go wrong if you're not careful. Here is a common gotcha. Suppose your Dockerfile does:

```dockerfile
FROM alpine:3.19
RUN apk update && apk add curl
COPY . /app
RUN cd /app && make
```

The first time you build, all four layers run. Great. The second time, you only changed a source file, so the `COPY . /app` layer changes, and so does the `RUN make` layer below it. The first two layers are cached. Fine.

But what if you wanted curl to also be at the latest version every build? `apk update` doesn't actually re-run unless that line itself is invalidated, because the layer is cached. So your image will keep using whatever curl version was current the day you first built it, even months later. Weird "stale layer" bugs come from this.

The fixes are: use `--no-cache` periodically, pin versions explicitly (`apk add curl=8.5.0-r0`), or use multi-stage builds and `--build-arg` cache busters. These are all things you'll learn over time. Don't worry about them yet.

## A First Dockerfile

A Dockerfile is the recipe. It is a plain text file, usually just called `Dockerfile` (no extension), that tells Docker how to build an image. Each line is an instruction. The instructions run in order, top to bottom, and each one produces a layer (sort of — `CMD`, `ENV`, and `LABEL` only add metadata, not filesystem content).

Here is the simplest non-trivial Dockerfile:

```dockerfile
FROM alpine:3.19
RUN apk add --no-cache curl
COPY hello.sh /usr/local/bin/
CMD ["hello.sh"]
```

Let's walk through every line.

### `FROM alpine:3.19`

This says "start with the Alpine Linux 3.19 image as the base." Alpine is a tiny, minimal Linux distribution. The whole image is about 5 MB. That is incredibly small for a Linux system. Most other distros are 50-200 MB even at their leanest.

When Docker sees this line, it goes to the registry (Docker Hub by default), pulls down the `alpine:3.19` image, and uses it as the bottom layer of your image. Everything you do later stacks on top.

Almost every Dockerfile starts with `FROM`. Common base images:

- `alpine:3.19` — tiny Linux, uses musl instead of glibc.
- `debian:bookworm-slim` — small Debian.
- `ubuntu:22.04` — full Ubuntu.
- `python:3.12-slim` — Debian-based with Python pre-installed.
- `node:20-alpine` — Alpine-based with Node.js pre-installed.
- `golang:1.22-alpine` — Alpine-based with Go pre-installed.
- `nginx:alpine` — pre-built nginx web server.
- `scratch` — literally nothing. The empty image. Used for static binaries.

Picking the base is a tradeoff. Smaller bases boot faster, take less disk, take less to push and pull, and have a smaller attack surface. But they also have fewer tools — you might find that "ping" doesn't exist and you have to install it, or that musl behaves slightly differently from glibc and your binary breaks. For new users, start with whatever base your language community recommends (e.g., `python:3.12-slim` for Python) and only switch to alpine or scratch when you understand the tradeoffs.

### `RUN apk add --no-cache curl`

`RUN` executes a command **at build time** to modify the filesystem of the image. Whatever changes the command makes to the filesystem become a new layer.

`apk` is Alpine's package manager. `apk add curl` installs the `curl` command-line tool. `--no-cache` tells `apk` not to keep its package index lying around in the image after install — that saves a few MB. Without it, you'd ship the package metadata cache as dead weight.

After this line runs, the image has `/usr/bin/curl` and any libraries curl depends on. They are all baked into a layer.

Two things to know about `RUN`:

1. Each `RUN` is a layer. So if you do five `RUN` lines, you get five layers. People often combine commands with `&&` to reduce layers: `RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*`. This is also where the `&& \` line continuation pattern comes from.
2. `RUN` runs as **root** by default unless you've changed the user with `USER`. That has security implications, which we'll get to.

### `COPY hello.sh /usr/local/bin/`

`COPY` takes a file (or directory) from the **build context** (the folder you ran `docker build` from) and copies it into the image at the given path. After this line, `/usr/local/bin/hello.sh` is a real file inside the image.

There's a related instruction called `ADD`. `ADD` does what `COPY` does plus extra magic: it can extract tar archives and download URLs. Most of the time you don't want the extra magic, you just want `COPY`. The Docker community guideline is **prefer `COPY` over `ADD`** unless you specifically need the magic.

### `CMD ["hello.sh"]`

`CMD` says "when somebody runs a container from this image, this is the default command to run." It does not run anything at build time. It is metadata stored in the image config.

There are two forms:

- **Exec form (preferred):** `CMD ["hello.sh", "--flag"]`. A JSON array. The command runs directly, no shell wrapper.
- **Shell form:** `CMD hello.sh --flag`. The command runs as `/bin/sh -c "hello.sh --flag"`. This wraps your process in a shell, which can mess with signal handling and PID 1 behavior.

Use exec form unless you have a specific reason not to.

If you also have an `ENTRYPOINT`, things get fancier; we'll cover that. For now, `CMD` is "default command if user doesn't override."

### How to build it

Save the Dockerfile in a folder. Save `hello.sh` next to it. Make `hello.sh` executable. From that folder, run:

```bash
$ docker build -t hello:v1 .
```

Docker will:

1. Read the Dockerfile.
2. Pull `alpine:3.19` if you don't have it cached.
3. Run each instruction, producing a layer for each.
4. Tag the final result as `hello:v1`.

Then:

```bash
$ docker run --rm hello:v1
```

This creates a container from the image, runs `hello.sh` as the default command, and (because of `--rm`) removes the container as soon as it exits.

That is the entire core of Docker. Everything else is variations and conveniences on top of `FROM`, `RUN`, `COPY`, `CMD`, `docker build`, and `docker run`.

### The other Dockerfile instructions, briefly

- `ENV KEY=value` — sets an environment variable in the image. Available at runtime in the container.
- `ARG name=default` — sets a build-time variable (only exists during build, not at runtime).
- `WORKDIR /app` — sets the working directory for subsequent commands. Like `cd`.
- `USER appuser` — change to a non-root user for subsequent commands and at runtime.
- `EXPOSE 8080` — documentation: declares that the container listens on port 8080. Doesn't actually publish anything.
- `VOLUME /data` — declares that `/data` should be a volume mount point. (Often more trouble than it's worth — prefer to specify volumes at run time.)
- `LABEL key=value` — attach metadata to the image (versions, maintainers, etc.).
- `HEALTHCHECK CMD curl -f http://localhost/ || exit 1` — tells Docker how to check if the container is healthy.
- `ENTRYPOINT ["./entrypoint.sh"]` — fix the program; whatever you pass to `docker run` becomes its arguments.
- `STOPSIGNAL SIGTERM` — what signal to send when stopping the container.
- `SHELL ["/bin/bash", "-c"]` — change the shell used by shell-form `RUN`s.
- `ONBUILD INSTRUCTION` — defer an instruction until a child image uses this one as a base. Rarely seen.

You don't need to know all of these to start. `FROM`, `RUN`, `COPY`, `WORKDIR`, `ENV`, `EXPOSE`, `CMD` will take you 90% of the way.

### A more realistic Dockerfile

Here is a Dockerfile for a Python web app, with comments:

```dockerfile
# Base image with Python pre-installed.
FROM python:3.12-slim

# Where the app lives inside the container.
WORKDIR /app

# Install OS packages we need (build tools to compile native extensions).
RUN apt-get update \
    && apt-get install -y --no-install-recommends gcc libpq-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy the dependency list first (rarely changes), so it gets cached.
COPY requirements.txt .

# Install Python deps. This layer caches well — only invalidates when
# requirements.txt changes.
RUN pip install --no-cache-dir -r requirements.txt

# Now copy the actual source code (changes a lot).
COPY . .

# Drop privileges.
RUN useradd --create-home --shell /bin/bash appuser
USER appuser

# Document the port.
EXPOSE 8000

# Default command to run.
CMD ["python", "-m", "uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8000"]
```

Notice the layer ordering trick: `COPY requirements.txt .` and `RUN pip install` happen before `COPY . .`. That way, when you edit your code, only the `COPY . .` layer (and anything below it) rebuilds — pip install stays cached. If you do `COPY . .` first, every code change re-runs pip install, and your build slows from 3 seconds to 90 seconds.

## Building, Tagging, Pushing

Once you have a Dockerfile, the workflow is build → tag → push.

### Building

```bash
$ docker build -t myapp:1.0 .
```

`-t myapp:1.0` tags the image as `myapp` with tag `1.0`. The `.` at the end is the **build context**: the folder Docker uploads to the build process. Everything inside that folder is fair game for `COPY` and `ADD`. Everything outside is invisible.

Build context matters. If you run `docker build .` in a folder that contains a 10 GB `node_modules` directory, Docker uploads 10 GB before it even starts. This is what `.dockerignore` is for, which we'll cover.

Other useful build flags:

- `-f path/to/Dockerfile` — use a Dockerfile not named `Dockerfile`.
- `--no-cache` — don't reuse cached layers.
- `--build-arg KEY=value` — pass build-time variables.
- `--target build` — stop at a specific stage in a multi-stage build.
- `--platform linux/arm64` — build for a different architecture (cross-build).
- `--progress plain` — verbose build output.

Modern Docker uses **BuildKit** by default, which is way smarter than the legacy builder: parallel build steps, better caching, smaller intermediate images, content-addressable everything, support for secrets and SSH agents during build. You shouldn't have to think about it most of the time, but if somebody mentions BuildKit, that's the engine behind `docker build` now.

### Tagging

A tag is a human-friendly label for an image. Image identity is actually a SHA-256 hash (the **digest**), but nobody types those. You give the image a tag instead.

```bash
$ docker tag myapp:1.0 myapp:latest
$ docker tag myapp:1.0 myreg.example.com/team/myapp:1.0
```

You can have many tags pointing at the same image. Common conventions:

- `:latest` — the most recent stable build (but using `:latest` in production is considered a smell because it's a moving target).
- `:1.0`, `:1.1` — semantic version tags.
- `:dev`, `:staging`, `:prod` — environment tags.
- `:abc1234` — git short-SHA, for full reproducibility.
- `:1.2.3-alpine` — version + base flavor.

The full image reference looks like:

```
[registry-host[:port]]/[user-or-org]/repo[:tag][@digest]
```

Examples:

- `nginx` — short for `docker.io/library/nginx:latest`. The default registry is `docker.io` (Docker Hub), the default user is `library`, the default tag is `latest`.
- `bitnami/nginx:1.25` — Bitnami's nginx tag 1.25 on Docker Hub.
- `ghcr.io/myorg/myapp:1.0` — myapp version 1.0 on GitHub Container Registry.
- `123456.dkr.ecr.us-east-1.amazonaws.com/myteam/myapp:1.0` — on Amazon ECR.
- `myreg.example.com:5000/internal/secret-tool:dev` — on a self-hosted registry running on port 5000.
- `nginx@sha256:abcd1234...` — by exact digest, no tag. Maximum reproducibility.

### Registries

A **registry** is a server that stores images. You push images to a registry, others pull from it. Common public registries:

- **Docker Hub** (`docker.io`): the default. Owns the namespace `library/` for official images.
- **GitHub Container Registry** (`ghcr.io`): tied to your GitHub account. Free for public images.
- **Amazon ECR** (`*.dkr.ecr.*.amazonaws.com`): AWS's registry, IAM-authenticated.
- **Google Artifact Registry** (`*-docker.pkg.dev`): GCP's. (The older Google Container Registry `gcr.io` is being retired.)
- **Azure Container Registry** (`*.azurecr.io`): Azure's.
- **Quay** (`quay.io`): Red Hat's. Strong on signed images.
- **Harbor**: open-source, often self-hosted.
- **JFrog Artifactory**: enterprise.

You authenticate to a registry once with `docker login registry.example.com` and then `docker push` and `docker pull` just work.

### Pushing

```bash
$ docker tag myapp:1.0 ghcr.io/myorg/myapp:1.0
$ docker push ghcr.io/myorg/myapp:1.0
```

Docker uploads only the layers the registry doesn't already have. If you only changed the top layer, only the top layer's bytes go over the wire. The rest is referenced by digest.

### Pulling

```bash
$ docker pull ghcr.io/myorg/myapp:1.0
```

Docker downloads layers it doesn't have, in parallel. Layers it already has on disk are referenced.

### Inspecting an image

```bash
$ docker inspect myapp:1.0           # full JSON dump of image config and layers
$ docker history myapp:1.0           # the layer-by-layer history
$ docker images                      # list all local images
$ docker images --digests            # show the SHA digests too
```

`docker history` is great for understanding why an image is so big: it shows you the size of each layer.

## Running Containers

`docker run` is the workhorse. You will type it more than any other Docker command in your life. It has dozens of flags. Here are the ones you will actually use.

```bash
$ docker run [flags] image[:tag] [command] [args...]
```

### `-it` — interactive terminal

`-i` keeps stdin open. `-t` allocates a pseudo-TTY. Together they let you actually use the terminal inside the container.

```bash
$ docker run --rm -it alpine sh
/ # ls
bin   etc   home  lib   ...
```

You are now "inside" the container's shell. Type `exit` to leave.

### `--rm` — auto-remove

By default, when a container exits, it stays around in stopped state until you `docker rm` it. `--rm` says "delete the container automatically when it exits." Use this for one-off commands. Don't use this for long-running services where you might want to inspect the corpse after a crash.

### `-d` — detached

Run in the background. The terminal returns immediately. The container keeps running.

```bash
$ docker run -d --name web nginx:alpine
e9a8f7b6c5d4...
```

You can then look at logs (`docker logs web`), exec into it (`docker exec -it web sh`), stop it (`docker stop web`), etc.

### `-p` — publish port

`-p HOST:CONTAINER` maps a port on the host to a port inside the container.

```bash
$ docker run -d -p 8080:80 nginx:alpine
```

Now `http://localhost:8080` on the host hits port 80 inside the container.

You can also do `-p 127.0.0.1:8080:80` to bind only to localhost (more secure), or `-p 8080:80/udp` for UDP.

`-P` (capital) auto-assigns random host ports for every `EXPOSE`d container port. Useful for testing.

### `-v` — volume / bind mount

`-v` mounts something into the container.

- `-v myvol:/data` — named volume.
- `-v /home/me/code:/code` — bind mount a host path.
- `-v /tmp` — anonymous volume at that path.

There's a newer, more readable form using `--mount`:

```bash
$ docker run --mount type=volume,src=myvol,dst=/data ...
$ docker run --mount type=bind,src=/home/me/code,dst=/code ...
$ docker run --mount type=tmpfs,dst=/scratch ...
```

We'll cover volumes vs bind mounts vs tmpfs in their own section.

### `-e` — environment variable

```bash
$ docker run -e DATABASE_URL=postgres://... -e LOG_LEVEL=debug myapp
$ docker run --env-file .env myapp
```

Environment variables are how containers most commonly receive configuration. The Twelve-Factor App principle says: config in env vars, not config files baked into the image.

### `--name` — give it a name

```bash
$ docker run -d --name web nginx:alpine
```

Without `--name`, Docker assigns a fun random name like `clever_einstein` or `angry_volhard`. With `--name`, you can refer to the container by your chosen name. Names must be unique across containers (running or stopped), so if you `docker run --rm --name web ...` and the container exits without `--rm`, you can't start a new one called `web` until you remove the old one.

### `--network` — pick a network

```bash
$ docker run --network mynet myapp
$ docker run --network host myapp        # share the host's network namespace
$ docker run --network none myapp        # no network at all
```

### `--user` — run as a specific user

```bash
$ docker run --user 1000:1000 alpine id
uid=1000 gid=1000
```

Containers run as root by default, which is a security smell. `--user 1000:1000` runs as UID 1000 / GID 1000 (a typical non-privileged user). Some images set `USER` in their Dockerfile already.

### `--cap-add` / `--cap-drop` — Linux capabilities

Linux capabilities are little switches for individual root powers. By default, Docker gives containers a limited set of capabilities. You can add or drop them.

```bash
$ docker run --cap-add NET_ADMIN ...     # let it manage interfaces, iptables, etc.
$ docker run --cap-drop ALL ...          # zero capabilities — most secure
$ docker run --cap-drop ALL --cap-add NET_BIND_SERVICE ...   # only bind low ports
```

Best practice: drop ALL, add back only what you need.

### `--read-only` — read-only root filesystem

```bash
$ docker run --read-only --tmpfs /tmp myapp
```

The container can't write anywhere, except where you explicitly mount writable volumes or tmpfs. Great hardening: even if an attacker gets RCE, they can't drop a payload to disk.

### `--memory` and `--cpus` — resource limits

```bash
$ docker run --memory 256m --cpus 0.5 myapp
```

`--memory 256m` caps RAM at 256 megabytes. If the process tries to use more, it gets killed by the OOM killer. `--cpus 0.5` gives it half a CPU's worth of CPU time. These flags wire into the Linux **cgroups** subsystem under the hood.

### `--restart` — restart policy

```bash
$ docker run -d --restart unless-stopped myapp
```

- `no` (default) — never restart.
- `on-failure[:N]` — restart if exit code is non-zero, up to N times.
- `always` — always restart, even if you stopped it manually (until daemon restart, kind of).
- `unless-stopped` — like `always`, but if you stopped it, leave it stopped.

For long-running services, `unless-stopped` is usually the right choice.

### Everything together

Realistic prod-ish run command:

```bash
$ docker run -d \
    --name api \
    --restart unless-stopped \
    -p 127.0.0.1:8080:8080 \
    -e DATABASE_URL=$DATABASE_URL \
    -e LOG_LEVEL=info \
    --user 1000:1000 \
    --read-only --tmpfs /tmp \
    --cap-drop ALL \
    --memory 512m --cpus 1 \
    -v api-data:/data \
    --network app-net \
    myorg/api:1.4.2
```

This is the kind of thing Compose or Kubernetes will write for you so you don't have to type all that, but every flag in there should make sense after this section.

## Volumes and Bind Mounts

Containers are ephemeral. Anything they write to their own filesystem evaporates when the container goes away. So how do you store data? Three answers.

### Named volumes — Docker manages it

```bash
$ docker volume create mydata
$ docker run -v mydata:/data myapp
```

Or all in one:

```bash
$ docker run -v mydata:/data myapp     # auto-creates mydata if not present
```

The volume `mydata` is **managed by Docker**. It lives somewhere under `/var/lib/docker/volumes/mydata/_data` on the host, but you don't really care where. You refer to it by name. Volumes survive container restarts and removals. They are the right choice for **databases, caches, anything where the data is owned by the container ecosystem.**

```bash
$ docker volume ls
$ docker volume inspect mydata
$ docker volume rm mydata
$ docker volume prune     # clean up unused
```

### Bind mounts — you point at a host directory

```bash
$ docker run -v /home/me/code:/code myapp
```

This mounts the host's `/home/me/code` directly into the container at `/code`. **Changes flow both ways in real time.** Edit a file on the host, the container sees it. Container writes a file, the host sees it. Permissions are real host permissions.

Bind mounts are perfect for development: edit your code on your laptop, the running container sees the changes instantly.

Bind mounts are dangerous in production: if you fat-finger the host path, you can mount over `/etc` or `/`. Container can also escape into host data by misbehaving. So use them carefully.

### tmpfs — RAM only

```bash
$ docker run --tmpfs /scratch myapp
```

`/scratch` inside the container is backed by RAM. Writes never hit disk. When the container exits, all the data evaporates. Use for fast scratch space, sensitive temporary data (so it never touches disk), or to make a `--read-only` container have one writable hot spot.

### When to use which

- **Database data, persistent app state:** named volume.
- **Local development of code:** bind mount your source dir.
- **Secrets that shouldn't hit disk:** tmpfs.
- **Reading a host config file:** bind mount, read-only (`-v /etc/app.conf:/etc/app.conf:ro`).
- **Sharing data between containers:** named volume mounted in both.

### A picture

```
  +-----------------+   +-----------------+   +-----------------+
  | NAMED VOLUME    |   | BIND MOUNT      |   | TMPFS           |
  +-----------------+   +-----------------+   +-----------------+
  | /var/lib/docker |   | /home/me/code   |   | (in RAM)        |
  | /volumes/data   |   | (your real fs)  |   |                 |
  +-----------------+   +-----------------+   +-----------------+
        |                       |                      |
        v                       v                      v
       /data                   /code                /scratch
   in container A            in container B       in container C
```

## Networking

Docker has several network drivers. You pick the one that matches what you want.

### bridge — the default

Each Docker host has a virtual switch called `docker0`. Bridge networks attach containers to that switch (or to a custom bridge). Containers on the same bridge can talk to each other by container name. Outbound traffic gets NAT'd through the host's IP. Inbound only happens if you explicitly publish ports with `-p`.

The default bridge is a bit dumb (no DNS-based service discovery between containers). Always create your own user-defined bridge:

```bash
$ docker network create mynet
$ docker run -d --name db --network mynet postgres:16
$ docker run -d --name api --network mynet myapi
```

Now `api` can reach `db` at `db:5432` thanks to Docker's built-in DNS.

### host — share the host's network

```bash
$ docker run --network host nginx:alpine
```

The container uses the **host's** network namespace directly. There is no isolation. The container's port 80 is the host's port 80. No NAT, no overhead. Use when you really need raw network performance or want to listen on host ports without `-p`. Loses much of the isolation benefit. Doesn't work on Docker Desktop the way it works on Linux.

### none — no networking

```bash
$ docker run --network none alpine ip a
```

Container has only `lo`. No internet, no anything. Use for fully isolated batch jobs.

### overlay — multi-host

Used by Docker Swarm and other orchestrators. Lets containers running on different physical hosts share a single virtual network. VXLAN under the hood. You probably won't touch this unless you're using Swarm. Kubernetes does its own equivalent.

### macvlan — give each container a real MAC

```bash
$ docker network create -d macvlan \
    --subnet=192.168.1.0/24 --gateway=192.168.1.1 \
    -o parent=eth0 mymacvlan
```

Each container gets its own MAC address on the physical network. To other devices on the LAN, the container looks like a separate machine. Useful for things like home-lab routing or running legacy apps that expect a real LAN presence. Has tricky gotchas with host-to-container traffic.

### A picture

```
  HOST
  +--------------------------------------------------------+
  |  +--------+    +--------+    +--------+                |
  |  | api    |    | web    |    | db     |  containers    |
  |  | :8000  |    | :80    |    | :5432  |                |
  |  +---+----+    +---+----+    +---+----+                |
  |      \             |             /                      |
  |       \____________|____________/                       |
  |                    |                                    |
  |             +------+------+                             |
  |             | mynet bridge|         user-defined bridge |
  |             +------+------+         (DNS by name)       |
  |                    |                                    |
  |             +------+------+                             |
  |             | docker0     |        default bridge       |
  |             +------+------+                             |
  |                    |                                    |
  |             [iptables NAT]                              |
  |                    |                                    |
  +--------------------|------------------------------------+
                       |
                  eth0 / wifi
                       |
                  the real world
```

### Network commands

```bash
$ docker network ls
$ docker network create mynet
$ docker network create --driver bridge --subnet 10.10.0.0/24 mynet
$ docker network inspect mynet
$ docker network connect mynet some-container
$ docker network disconnect mynet some-container
$ docker network rm mynet
$ docker network prune
```

## Compose (`docker-compose.yml`)

Real apps usually have many containers: one for the database, one for the cache, one for the API, one for the frontend, one for the background worker. Typing all those `docker run` commands is painful and error-prone. Enter **Compose.**

Compose is a YAML file that describes your whole multi-container stack. One command brings the whole thing up. One command tears it all down.

### A first compose file

```yaml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - dbdata:/var/lib/postgresql/data
    restart: unless-stopped

  cache:
    image: redis:7-alpine
    restart: unless-stopped

  api:
    build: ./api
    environment:
      DATABASE_URL: postgres://postgres:secret@db:5432/postgres
      REDIS_URL: redis://cache:6379
    depends_on:
      - db
      - cache
    ports:
      - "127.0.0.1:8080:8080"
    restart: unless-stopped

  web:
    build: ./web
    depends_on:
      - api
    ports:
      - "8443:443"
    restart: unless-stopped

volumes:
  dbdata:

networks:
  default:
    name: myapp_net
```

Save this as `docker-compose.yml`. Then:

```bash
$ docker compose up -d        # build (if needed) and start everything in background
$ docker compose ps           # what's running
$ docker compose logs -f api  # follow the api's logs
$ docker compose exec api sh  # shell into the running api container
$ docker compose down         # stop and remove everything
$ docker compose down -v      # also delete volumes
```

The first run takes a while (build images, pull base images). Subsequent runs are quick.

### What Compose did for you

- Created a network called `myapp_net`.
- Pulled `postgres:16` and `redis:7-alpine`.
- Built the `api` and `web` images from their respective Dockerfiles.
- Started all four containers attached to the same network.
- Connected them so `api` can reach `db` at `db:5432` and `cache` at `cache:6379` by name.
- Created the `dbdata` volume and attached it to `db`.
- Set up restart policies.
- Mapped ports to the host.

You wrote 30 lines of YAML. To do that with raw `docker run` commands you'd have written, oh, 50 lines of bash with manual ordering, error handling, and a teardown script. Compose is enormously productive.

### Compose v1 vs v2

There are two Composes in the world.

- **docker-compose** (v1, Python, `docker-compose up`, with a hyphen) — old, deprecated.
- **docker compose** (v2, Go, built into the Docker CLI as a plugin, no hyphen) — modern.

Use v2. The YAML format is mostly compatible.

### A graph picture

```
                     COMPOSE STACK
                  +----------------+
                  |  myapp_net     |  <-- network created by Compose
                  +-------+--------+
                          |
        +-----------------+-----------------+
        |        |        |                 |
     +--v--+  +--v--+  +--v--+           +--v--+
     | db  |  |cache|  | api |---bind--->| web |
     +-----+  +-----+  +-----+           +-----+
        |                 |                 |
     [dbdata             [127.0.0.1:8080  [8443:443]
      volume]              :8080]
```

### Useful Compose flags

- `docker compose up --build` — force a rebuild.
- `docker compose up --scale worker=3` — run three copies of the `worker` service.
- `docker compose pull` — pull all images.
- `docker compose run --rm api alembic upgrade head` — one-off command using the api image and config.
- `docker compose config` — print the resolved config (after env interpolation).

## Multi-Stage Builds

Your final image should not contain compilers, build tools, source code, package caches, or any other "stuff you only needed at build time." Multi-stage builds are how you keep the final image lean.

### The pattern

```dockerfile
# Stage 1: build the binary.
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/app ./cmd/app

# Stage 2: tiny final image.
FROM alpine:3.19
COPY --from=build /out/app /usr/local/bin/app
ENTRYPOINT ["/usr/local/bin/app"]
```

Two `FROM` lines = two stages. The first stage has Go installed (a few hundred MB). The second stage has only Alpine (5 MB) plus your compiled binary. The final image is tiny — you never paid for the Go toolchain in the shipped artifact.

`COPY --from=build` is the magic. It copies a path from the build stage into the current stage. Anything not copied from the build stage is **discarded** when the build finishes.

### Even more extreme: scratch

If your binary is statically linked (Go is by default), you can go even further:

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app ./cmd/app

FROM scratch
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
```

`scratch` is the empty image. **Zero bytes** of base. Your final image is just your binary. A typical Go app this way is 5-15 MB. The smaller the image, the faster it pulls, the less surface area for vulnerabilities.

### A picture

```
                       BUILD STAGE                 FINAL STAGE
                    +------------------+        +-----------------+
                    | golang:1.22      |        | alpine:3.19     |
                    | + your sources   |        | + just the      |
                    | + go modules     |        |   compiled      |
                    | + go build       |   +--->|   binary        |
                    | -> /out/app      |   |    +-----------------+
                    +------------------+   |              |
                            |              |       this is what you ship
                            +-- COPY ------+       (5-30 MB instead of
                                --from=build       500-1500 MB)
                            
                    discarded after build
```

### Tips

- Name your stages with `AS name` so you can reference them by name.
- You can have more than two stages (`AS deps`, `AS build`, `AS test`, `AS final`).
- Use `--target` to stop at a specific stage: `docker build --target build -t myapp:builder .`.
- Use a stage purely as a place to vendor or cache dependencies, then copy from it later.

## .dockerignore

`.dockerignore` is to `docker build` what `.gitignore` is to git: a list of patterns that exclude files from the **build context**. Docker uploads the build context to the daemon (or to BuildKit) at the start of a build. If your repo has a 5 GB `node_modules` folder and you don't ignore it, every build uploads 5 GB before it does anything. Builds become miserably slow.

Even if you don't `COPY` those files into the image, they still get uploaded as part of the context. So the rule is: if you don't need it inside the image, ignore it.

A typical `.dockerignore`:

```
.git
.gitignore
node_modules
__pycache__
*.pyc
.venv
venv
target
dist
build
.env
.env.local
.idea
.vscode
*.md
README*
LICENSE
docker-compose*.yml
Dockerfile*
.dockerignore
.DS_Store
*.log
coverage
.cache
.next
.svelte-kit
secrets/
```

A few notes:

- `.git` is huge in established repos. Always ignore it unless you specifically need git metadata in the image.
- `.env` files often contain secrets. Always ignore them. Hardcoding secrets in images is a classic mistake.
- Ignoring `Dockerfile*` keeps the image tidy: nobody needs the build instructions inside the running container.
- Patterns are relative to the build context root and use shell globs.
- `**/foo` matches `foo` at any depth (BuildKit only).

A good `.dockerignore` makes builds 5-50x faster on big repos. It is not optional.

## Container Internals

Behind the magic, a Docker container is just a Linux process. There is no special "container" thing in the kernel. The container is just a regular process that has been wrapped in a bunch of Linux features that, together, make it feel like a separate computer. The features stack like layers (different layers from filesystem layers — these are *isolation* layers).

### Namespaces

Namespaces partition kernel resources so a process sees only its own slice. There are several:

- **PID namespace** — the container has its own process tree. Inside the container, your app is PID 1 and the world looks empty. The host sees all processes; the container sees only its own.
- **MNT namespace** — the container has its own filesystem mount table. That's how a container can have its own `/`.
- **NET namespace** — the container has its own network interfaces, routing table, iptables rules. It can have eth0 with no relation to the host's eth0.
- **UTS namespace** — the container can have its own hostname.
- **IPC namespace** — isolation of System V IPC and POSIX message queues.
- **USER namespace** — UID 0 inside the container can map to UID 1000 on the host. Critical for rootless containers.
- **CGROUP namespace** — isolation of the cgroup view.
- **TIME namespace** — isolation of system time (rare, newish).

When you start a container, Docker creates a new set of namespaces and starts the process inside them. That is the bulk of the "isolation" you feel.

### cgroups

Cgroups (control groups) are the kernel's accounting and limiting subsystem. They answer "how much CPU is this group of processes using? how much RAM? how much I/O? Now cap that group at X."

When you say `docker run --memory 256m --cpus 0.5`, Docker creates a cgroup for the container and sets memory and CPU limits on it. The kernel enforces them.

Cgroups also provide accounting: `docker stats` reads from cgroups to show you per-container CPU%, mem%, network I/O, etc.

### Capabilities

In old Linux, root could do everything. Capabilities split "do everything" into ~40 separate switches. `CAP_NET_ADMIN` lets you manage network interfaces. `CAP_SYS_ADMIN` is the kitchen sink. Containers run with a default subset (about 14 capabilities). You can drop them all and add back only what you need.

### seccomp

Seccomp is a syscall filter. Docker ships a default seccomp profile that blocks ~50 dangerous or rarely-used syscalls. If a process inside the container tries one, it gets EPERM. This shrinks the kernel's attack surface from "everything" to "everything minus the obviously bad stuff."

### LSM (AppArmor / SELinux)

Linux Security Modules are kernel modules that enforce mandatory access control: rules about what processes can do, beyond Unix permissions. AppArmor uses path-based rules; SELinux uses label-based rules. Docker ships a default AppArmor profile (on Ubuntu/Debian) and works with SELinux (on Fedora/RHEL) for additional confinement.

### Putting it together

A "container" is a process that:

- Lives in its own set of namespaces (so it sees its own PID 1, its own /, its own eth0).
- Is constrained by a cgroup (so it can't hog all the CPU or RAM).
- Has reduced capabilities.
- Is running under a seccomp filter.
- Has an LSM profile applied.

The image's filesystem is mounted as the new root. The image's command runs as PID 1. The default network namespace gets a veth pair into the host bridge. Done. That is a container.

For deeper coverage, see `cs ramp-up linux-kernel-eli5`.

## Container Runtime Architecture

When you type `docker run`, a tower of programs cooperates to make a container actually exist. From the top:

```
   YOU
    |
    |  $ docker run alpine echo hi
    v
  +-----------+
  | docker    |  CLI client.
  | (CLI)     |  Sends a REST request to the docker daemon.
  +-----+-----+
        |
        |  HTTP over /var/run/docker.sock
        v
  +-----------+
  | dockerd   |  The daemon. Long-running root process.
  | (daemon)  |  Image management, build, network, volumes.
  +-----+-----+
        |
        |  gRPC
        v
  +-----------+
  | containerd|  Higher-level container runtime.
  |           |  Image transfer, lifecycle, snapshots.
  +-----+-----+
        |
        |  fork+exec, OCI runtime spec
        v
  +-----------+
  | runc      |  Low-level OCI runtime. Tiny binary.
  |           |  Does the actual namespace/cgroup/seccomp/exec dance.
  +-----+-----+
        |
        v
   container process running
```

### What each piece does

- `docker` (CLI) — what you type. Sends commands to the daemon.
- `dockerd` (daemon) — the long-running boss. Handles images, builds, networks, volumes, the API.
- `containerd` — the container runtime. Pulls images, manages snapshots, supervises container lifecycles. Speaks gRPC.
- `runc` — the OCI runtime. The tiny C/Go program that calls `clone()` with the right namespace flags, sets cgroup limits, applies the seccomp profile, drops capabilities, and `execve()`s your command. After runc has done its job, it's done — the container is just a process now.

### Why so many layers?

Originally Docker did all of this in one big monolithic daemon. As the ecosystem matured, people wanted to swap parts out. So the project got cut up:

- **OCI specs** — Open Container Initiative. Standardizes the image format and runtime spec so different tools agree on bytes.
- **runc** — extracted as the reference OCI runtime.
- **containerd** — extracted as a higher-level runtime, donated to the CNCF.
- **dockerd** — became a thin layer on top of containerd.

Now there are alternatives at every layer:

- Instead of `runc`: **crun** (faster), **kata** (lightweight VMs), **gVisor** (user-space kernel for extra isolation).
- Instead of `containerd`: **CRI-O** (Kubernetes-focused), **podman** (daemonless).
- Instead of `dockerd`: just talk to containerd directly with `nerdctl` or `ctr`.

### podman

Podman is "Docker without the daemon." It runs containers as the calling user (rootless), using fork-and-exec instead of a long-lived service. The CLI is intentionally near-identical to Docker's: `podman run`, `podman build`, `podman ps`. You can `alias docker=podman` and most things just work. Big in the Red Hat / Fedora world.

### buildah

Buildah is a build-only tool. It can build OCI images without a daemon and without a Dockerfile (you script builds in shell). Often paired with podman.

### skopeo

Skopeo moves images around without unpacking them: copy from one registry to another, inspect remote images, sign images. Read-only utility.

## Image Internals

A Docker image is not a "single thing." It is **a manifest plus a config plus a list of layer tarballs**, all addressed by content hashes.

### The OCI image format

The Open Container Initiative (OCI) standardizes the format. An OCI image is:

- **A manifest** — JSON listing the config and layers by digest.
- **A config** — JSON describing the image: entrypoint, env, exposed ports, OS, architecture, build history.
- **Layers** — each layer is a gzipped tarball of filesystem changes (additions, modifications, deletions via whiteout).

When you `docker pull image:tag`:

1. Fetch the manifest by tag.
2. Read the manifest, get the digests of the config and each layer.
3. Fetch the config and each layer (in parallel) — but skip ones we already have, by digest.
4. Verify each piece's hash matches its digest.
5. Untar the layers into the storage driver's directory.

When you `docker run`:

1. Storage driver stitches the layer tar trees together with OverlayFS.
2. New writable upper layer is created.
3. Container starts with the union as its root.
4. Image's config (entrypoint, env, etc.) is applied to the new process.

### Multi-arch images

A single image **tag** can point to an "image index" (also called a "manifest list") that has multiple manifests, one per platform. So `nginx:alpine` is actually a manifest list. When `linux/amd64` pulls, it gets the amd64 manifest. When `linux/arm64` pulls, it gets the arm64 manifest. The right binaries arrive automatically.

You build multi-arch images with `docker buildx`:

```bash
$ docker buildx build --platform linux/amd64,linux/arm64 -t myorg/myimg:1.0 --push .
```

This is critical now that ARM (Apple Silicon, AWS Graviton) is everywhere. If your image is amd64-only, ARM users get `exec format error`.

### Digests vs tags

A **tag** is mutable. Today `nginx:latest` is one image; tomorrow it might be a totally different one.

A **digest** is immutable. `nginx@sha256:abc1234...` will always, forever, be the exact same bits.

For reproducibility, especially in production, refer to images by digest. CI/CD pipelines often resolve a tag to a digest at deploy time and pin to the digest from then on.

## Common Errors

Here are the errors you will see, what they mean, and what to do.

### "Cannot connect to the Docker daemon at unix:///var/run/docker.sock"

The daemon isn't running, or you can't reach it.

```
Cannot connect to the Docker daemon at unix:///var/run/docker.sock.
Is the docker daemon running?
```

Fixes:

- `sudo systemctl start docker` (Linux).
- Open Docker Desktop (macOS / Windows).
- Check `DOCKER_HOST` env var — it might be pointing somewhere weird.
- Check `docker context ls` — wrong context can hide the daemon.

### "permission denied while trying to connect to the Docker daemon socket"

Your user isn't in the `docker` group.

```
permission denied while trying to connect to the Docker daemon socket
at unix:///var/run/docker.sock: ...
```

Fix:

```bash
$ sudo usermod -aG docker $USER
# log out, log back in
$ docker ps     # should now work
```

Caveat: the docker group is effectively root-equivalent. Treat membership accordingly.

### "exec format error"

Wrong architecture for the image.

```
standard_init_linux.go:228: exec user process caused: exec format error
```

You're trying to run an `amd64` image on `arm64` (or vice versa). Common when an Apple Silicon Mac tries to run an old `amd64` image. Fix: rebuild the image multi-arch, or `docker run --platform linux/amd64` to force emulation (slow).

### "no space left on device"

Your Docker storage is full of dangling images, containers, and volumes.

Fix:

```bash
$ docker system df               # see what's eating space
$ docker system prune            # remove stopped containers, dangling images, networks
$ docker system prune -a         # also remove unused images (not just dangling)
$ docker system prune -a --volumes   # also remove unused volumes (DANGER: data loss)
```

### "manifest unknown"

The image:tag combo doesn't exist.

```
Error response from daemon: manifest for myorg/myimg:badtag not found:
manifest unknown: manifest unknown
```

You probably typo'd the tag. Or you're on the wrong registry. Or it really doesn't exist. Run `docker search` or browse the registry's UI.

### "OCI runtime create failed"

Something in the runtime layer (runc) refused to start the container.

```
docker: Error response from daemon: OCI runtime create failed:
container_linux.go: ... permission denied: unknown.
```

Usually caused by:

- A seccomp profile blocking a syscall the entrypoint needs.
- An LSM (AppArmor/SELinux) denial.
- Bad capabilities (you `--cap-drop ALL` and the container needs at least `NET_BIND_SERVICE`).
- Bad mount: trying to mount over a path that doesn't exist.

Try `docker run --security-opt seccomp=unconfined ...` to test if seccomp is the culprit. Read `/var/log/audit/audit.log` for SELinux denials. Read `dmesg` for AppArmor denials.

### "address already in use"

You tried to publish to a host port something else is using.

```
docker: Error response from daemon: driver failed programming external connectivity
on endpoint web ... bind: address already in use.
```

Fix:

```bash
$ sudo lsof -i :8080            # who's on it?
$ docker ps                     # is another container holding it?
```

Then kill that thing, or pick a different host port.

### "container is not running"

You tried to `exec` into a container that already exited.

Fix: `docker logs <name>` to see why it died. Common causes are: missing required env var, wrong command, immediate crash on startup.

### "image operating system 'linux' cannot be used on this platform"

You're on Docker Desktop with Windows containers selected, and you're trying to run a Linux image. Or vice versa. Switch container OS in Docker Desktop's settings.

### "Error response from daemon: pull access denied for myreg/img"

You aren't authenticated to the registry, or the image isn't visible to your user.

Fix:

```bash
$ docker login myreg.example.com
```

Or check that the repo exists and your account has access.

### "ERROR [internal] load metadata for docker.io/library/X"

Build failure pulling base image. Usually a DNS / network issue. Check that `docker info` shows healthy networking, that you can `curl https://registry-1.docker.io/v2/`, etc. Behind a corporate proxy? Configure Docker's proxy.

### "ERROR: failed to solve: ... no match for platform in manifest"

Image doesn't have a manifest for your CPU architecture. Fix: rebuild the source image multi-arch, or pull with `--platform linux/amd64` (and accept emulation overhead).

### "killed" (no specific message)

Process was killed by the OOM killer. You hit `--memory` limit. Either the app legitimately needs more, or it has a memory leak. Inspect with `docker stats` to see steady-state usage.

## Hands-On

Open a terminal. We are going to go through ~40 commands. Type each one and look at the output.

```bash
$ docker version
Client: Docker Engine - Community
 Version:           24.0.7
 API version:       1.43
 ...
Server: Docker Engine - Community
 Engine:
  Version:          24.0.7
  ...
 containerd:
  Version:          1.6.25
 runc:
  Version:          1.1.10
 ...
```

You see the client version, the server (daemon) version, and which containerd/runc the daemon is using. If you see "Cannot connect to the Docker daemon" here, the rest of the sheet won't work — fix that first.

```bash
$ docker info | head -30
Client: Docker Engine - Community
 Version:    24.0.7
 ...

Server:
 Containers: 0
  Running: 0
  Paused: 0
  Stopped: 0
 Images: 0
 Server Version: 24.0.7
 Storage Driver: overlay2
 ...
```

`docker info` is the kitchen-sink summary: storage driver, network drivers, runtimes, security options, root dir, registry, total containers, total images. Skim it once to know what's there.

```bash
$ docker ps
CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES
```

Empty. No running containers right now.

```bash
$ docker ps -a
CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES
```

Also empty. `-a` shows stopped containers too.

```bash
$ docker images
REPOSITORY   TAG       IMAGE ID   CREATED   SIZE
```

No images on disk yet.

```bash
$ docker pull alpine:3.19
3.19: Pulling from library/alpine
e7b300aee9f9: Pull complete
Digest: sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48
Status: Downloaded newer image for alpine:3.19
docker.io/library/alpine:3.19
```

You pulled the Alpine 3.19 base image. About 5 MB. The big hex string is the manifest digest.

```bash
$ docker run --rm -it alpine sh
/ # ls
bin    etc    lib    media  opt    root   sbin   sys    usr
dev    home   lib64  mnt    proc   run    srv    tmp    var
/ # cat /etc/os-release | head -2
NAME="Alpine Linux"
ID=alpine
/ # exit
```

You jumped into a container, ran some commands, exited. Because of `--rm`, the container is now gone.

```bash
$ docker run -d --name web -p 8080:80 nginx:alpine
abc123def456...
```

You started nginx in the background. The big hex string is the container ID. Now hit `http://localhost:8080` in your browser. You should see the nginx welcome page.

```bash
$ docker logs web
/docker-entrypoint.sh: /docker-entrypoint.d/ is not empty, will attempt to perform configuration
...
2024/05/01 12:34:56 [notice] 1#1: start worker processes
```

The container's stdout/stderr.

```bash
$ docker logs -f web
... (live tail; press Ctrl+C to exit)
```

`-f` follows new log output as it appears.

```bash
$ docker exec -it web sh
/ # ps aux
PID   USER     TIME  COMMAND
    1 root      0:00 nginx: master process nginx -g daemon off;
   29 nginx     0:00 nginx: worker process
   30 nginx     0:00 nginx: worker process
   ...
/ # exit
```

`docker exec` runs a new command **in an already-running container.** Notice nginx is PID 1.

```bash
$ docker stop web && docker rm web
web
web
```

Stops the container (sends SIGTERM, then SIGKILL after 10s) and removes it.

Now let's build our own image. Make a folder somewhere:

```bash
$ mkdir hello-docker && cd hello-docker
$ cat > hello.sh <<'EOF'
#!/bin/sh
echo "hello from container, brrr"
EOF
$ chmod +x hello.sh
$ cat > Dockerfile <<'EOF'
FROM alpine:3.19
COPY hello.sh /usr/local/bin/
CMD ["hello.sh"]
EOF
$ docker build -t hello:v1 .
[+] Building 0.5s (7/7) FINISHED
 => [internal] load build definition from Dockerfile
 => [internal] load .dockerignore
 => [internal] load metadata for docker.io/library/alpine:3.19
 => [1/2] FROM docker.io/library/alpine:3.19
 => [internal] load build context
 => [2/2] COPY hello.sh /usr/local/bin/
 => exporting to image
 => => writing image sha256:abc...
 => => naming to docker.io/library/hello:v1
```

```bash
$ docker run --rm hello:v1
hello from container, brrr
```

Your first own image, working.

```bash
$ docker tag hello:v1 myreg.example.com/hello:v1
$ docker push myreg.example.com/hello:v1
The push refers to repository [myreg.example.com/hello]
... (would push, but myreg doesn't exist; this would fail in real life)
```

In real life, you'd `docker login myreg.example.com` first.

```bash
$ docker login myreg.example.com
Username: stevie
Password: ********
Login Succeeded
```

```bash
$ docker history hello:v1
IMAGE          CREATED         CREATED BY                                      SIZE
abc...         2 minutes ago   CMD ["hello.sh"]                                0B
def...         2 minutes ago   COPY hello.sh /usr/local/bin/                   45B
e7b...         2 weeks ago     /bin/sh -c #(nop) CMD ["/bin/sh"]               0B
<missing>      2 weeks ago     /bin/sh -c #(nop) ADD file:abcdef in /          7.73MB
```

Each layer with its size. Notice the `0B` for the CMD layer — pure metadata.

```bash
$ docker inspect hello:v1
[
    {
        "Id": "sha256:abc...",
        "RepoTags": ["hello:v1", "myreg.example.com/hello:v1"],
        "Created": "2024-05-01T12:34:56Z",
        "Architecture": "arm64",
        "Os": "linux",
        "Config": {
            "Cmd": ["hello.sh"],
            ...
        },
        ...
    }
]
```

Full JSON dump of image metadata.

```bash
$ docker rmi hello:v1
Untagged: hello:v1
Deleted: sha256:abc...
Deleted: sha256:def...
```

Remove the image.

```bash
$ docker system df
TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          5         1         150MB     100MB (66%)
Containers      1         1         0B        0B
Local Volumes   2         1         100MB     50MB (50%)
Build Cache     20        0         50MB      50MB
```

How much disk Docker is using, and how much you could free up.

```bash
$ docker system prune -a --volumes
WARNING! This will remove:
  - all stopped containers
  - all networks not used by at least one container
  - all anonymous volumes not used by at least one container
  - all images without at least one container associated to them
  - all build cache
Are you sure you want to continue? [y/N] y
Deleted Containers: ...
Deleted Networks: ...
Deleted Volumes: ...
Deleted Images: ...
Total reclaimed space: 1.5GB
```

Big cleanup. **Can lose data** — be sure before you say y.

```bash
$ docker stats
CONTAINER ID   NAME    CPU %     MEM USAGE / LIMIT     MEM %     NET I/O     BLOCK I/O   PIDS
e9a8f7b6c5d4   web     0.01%     8.5MiB / 7.7GiB       0.11%     5kB / 0B    0B / 0B     5
```

Live resource stats. Press Ctrl+C to exit.

```bash
$ docker top web
UID   PID    PPID  C  STIME  TTY  TIME      CMD
root  12345  ...   0  12:34  ?    00:00:00  nginx: master process nginx -g daemon off;
```

The processes inside a container, as seen by the host kernel.

```bash
$ docker volume create data
data
$ docker volume ls
DRIVER    VOLUME NAME
local     data
$ docker network ls
NETWORK ID    NAME      DRIVER    SCOPE
abc...        bridge    bridge    local
def...        host      host      local
xyz...        none      null      local
$ docker network create mynet
0123456789ab...
```

Volume and network plumbing.

Now Compose. In a folder, save:

```yaml
# docker-compose.yml
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
```

```bash
$ docker compose up -d
[+] Running 2/2
 ✔ Network myproj_default  Created
 ✔ Container myproj-web-1  Started
$ docker compose ps
NAME             IMAGE          COMMAND                  SERVICE   CREATED   STATUS    PORTS
myproj-web-1     nginx:alpine   "/docker-entrypoint..."  web       3s ago    Up 2s     0.0.0.0:8080->80/tcp
$ docker compose logs -f web
... (live logs; Ctrl+C to exit)
$ docker compose exec web sh
/ # exit
$ docker compose down -v
[+] Running 2/2
 ✔ Container myproj-web-1  Removed
 ✔ Network myproj_default  Removed
```

Compose lifecycle in five commands.

```bash
$ docker buildx build --platform linux/amd64,linux/arm64 -t myorg/img:multi --push .
... (builds for both arches and pushes the manifest list)
```

Multi-arch build.

```bash
$ docker save hello:v1 | gzip > hello.tar.gz
$ docker load < hello.tar.gz
Loaded image: hello:v1
```

Save an image to a tarball, load it elsewhere — useful for air-gapped environments.

```bash
$ docker context ls
NAME          DESCRIPTION                                DOCKER ENDPOINT
default *     Current DOCKER_HOST based config           unix:///var/run/docker.sock
my-remote     Remote production server                   ssh://prod@build-host
$ docker context use my-remote
my-remote
```

Contexts let you switch between local and remote daemons with one command.

```bash
$ docker scout cves alpine:3.19
... (lists CVEs in this image and how to fix them)
```

Modern Docker CLI ships with `scout` for vulnerability scanning. Other popular scanners: Trivy, Grype, Snyk.

You now have hands-on experience with the core of Docker. There is a lot more (Swarm, secrets, configs, plugins, BuildKit advanced features), but you can pick those up as needed.

## Common Confusions

There are a lot of things in Docker that sound the same but aren't. Here are the pairs that trip up new users.

### Image vs container

Already covered. Image = recipe. Container = meal. The single most common confusion. If you find yourself saying "the container has Python 3.12 installed in it," you probably mean the image.

### `docker run` vs `docker start`

- `docker run image` creates a new container from the image and starts it. Always.
- `docker start NAME` starts an existing stopped container that already exists.

`docker run` is for "I want to spin up a fresh container." `docker start` is for "the container already exists, fire it back up."

### `docker exec` vs `docker run`

- `docker run image cmd` makes a **new** container from the image and runs cmd as its main process.
- `docker exec container cmd` runs cmd **inside an already-running** container as a side process.

`docker run` is "fork a new world." `docker exec` is "open another shell in the existing world."

### `CMD` vs `ENTRYPOINT`

Both set what runs when a container starts. They behave differently in combination.

- `CMD ["foo"]` alone: container runs `foo`. If user passes args to `docker run`, those replace `foo`.
- `ENTRYPOINT ["foo"]` alone: container runs `foo`. If user passes args, they get appended to `foo` as its arguments.
- Both: container runs `ENTRYPOINT` followed by `CMD`. User args replace `CMD` but not `ENTRYPOINT`.

Mental model: `ENTRYPOINT` is the program; `CMD` is the default arguments. Use `ENTRYPOINT` for a fixed binary plus a default command.

### `EXPOSE` vs `-p`

- `EXPOSE 8080` in a Dockerfile: documentation. Says "this container listens on 8080." **Doesn't actually publish.**
- `-p 8080:80` at run time: actually publishes. Maps host:8080 → container:80.

You can publish a port without `EXPOSE`, and `EXPOSE` without `-p` doesn't make the port reachable from outside. `EXPOSE` only matters as metadata and as input to `-P` (capital).

### Volume vs bind mount

- Volume: managed by Docker, lives in Docker's directory, named.
- Bind mount: you point at a host path directly.

Use volumes for "data owned by Docker." Use bind mounts for "data owned by the host" (like your dev source tree).

### `docker-compose` (v1) vs `docker compose` (v2)

- `docker-compose` (with hyphen): old Python CLI. Deprecated.
- `docker compose` (no hyphen): new Go plugin. Modern.

Use v2.

### Image tag vs image digest

- Tag: human label. Mutable. Today's `:latest` ≠ tomorrow's `:latest`.
- Digest: hash. Immutable. Always exactly the same image bits.

Tags for humans, digests for production reproducibility.

### `docker stop` vs `docker kill`

- `docker stop`: SIGTERM, wait 10s for graceful shutdown, then SIGKILL.
- `docker kill`: SIGKILL immediately. No grace.

Use `stop` by default. `kill` is for "I don't trust the process to shut down."

### Container exits vs container is removed

- A container that exited is **stopped**. It still exists; you can `docker start` it.
- A container that was `docker rm`'d is **gone**.

Stopped containers consume disk space (their writable layer). `docker ps` doesn't show them; `docker ps -a` does.

### `RUN` vs `CMD`

- `RUN`: executes at **build time**, modifies the image. Each is a layer.
- `CMD`: sets the **runtime default command**. No build-time effect.

If you say `RUN nginx`, you're (uselessly) starting nginx during the build. If you say `CMD ["nginx", "-g", "daemon off;"]`, you're setting the default command for when somebody runs the image.

### `COPY` vs `ADD`

- `COPY`: literal file copy.
- `ADD`: copy + auto-extract tarballs + auto-download URLs.

Prefer `COPY`. Use `ADD` only when you actually want the magic.

### Image not found vs image pull access denied

- "manifest unknown" / "image not found": the image:tag doesn't exist anywhere.
- "pull access denied": the image exists but you don't have permission.

Different fixes (correct the tag vs `docker login`).

### `--platform` build vs run

- `docker build --platform linux/arm64`: build for arm64.
- `docker run --platform linux/amd64`: run an amd64 image (possibly under emulation).

These are different operations. People sometimes try to "run a build for another platform" and confuse themselves.

### Containerd vs Docker

- Containerd: the runtime layer. Pulls images, supervises containers.
- Docker: a higher-level platform on top of containerd, with build, network plugins, volumes, etc.

Kubernetes uses containerd directly (no Docker daemon). Docker uses containerd internally.

## Vocabulary

| Word | Plain English |
|---|---|
| **Docker** | The platform that runs software in standardized containers. |
| **dockerd** | The Docker daemon. Long-running root process that does the work. |
| **daemon** | A long-running background program that handles requests. |
| **CLI** | Command-line interface. The `docker` command you type. |
| **image** | The recipe / template / class. Read-only. Built once. |
| **container** | A running instance of an image. Mortal. Has writable scratch space. |
| **layer** | One slice of an image's filesystem. Each Dockerfile instruction = one layer. |
| **base image** | The image your image starts `FROM`. Bottom layer. |
| **scratch** | The empty image. Zero bytes. For static binaries. |
| **distroless** | A category of base images with only the runtime, no shell or package manager. |
| **alpine** | A tiny Linux distro (~5 MB), uses musl libc. Common base. |
| **debian** | A bigger Linux distro. Common base, GNU libc. |
| **ubuntu** | A Linux distro based on Debian. Sometimes used as a base. |
| **FROM** | Dockerfile instruction: "start from this base image." |
| **RUN** | Dockerfile instruction: "run this command at build time, in a new layer." |
| **COPY** | Dockerfile instruction: "copy these files from build context into the image." |
| **ADD** | Like COPY but also extracts tar archives and fetches URLs. Prefer COPY. |
| **ENV** | Dockerfile instruction: set an environment variable. |
| **ARG** | Dockerfile instruction: declare a build-time variable. |
| **CMD** | Dockerfile instruction: default command at container start. |
| **ENTRYPOINT** | Dockerfile instruction: fixed program at container start; user args become its args. |
| **EXPOSE** | Dockerfile instruction: documentation that a port is used. Doesn't publish. |
| **VOLUME** | Dockerfile instruction: declare a path as a volume mount point. |
| **WORKDIR** | Dockerfile instruction: cd to this directory for subsequent instructions. |
| **USER** | Dockerfile instruction: switch to this user for subsequent instructions and at runtime. |
| **LABEL** | Dockerfile instruction: attach metadata key-values to the image. |
| **HEALTHCHECK** | Dockerfile instruction: define how Docker checks if the container is healthy. |
| **STOPSIGNAL** | Dockerfile instruction: which signal to send when stopping the container. |
| **ONBUILD** | Dockerfile instruction: deferred instruction for downstream images. |
| **SHELL** | Dockerfile instruction: change the shell used by shell-form RUN. |
| **Dockerfile** | The plain-text recipe used to build an image. |
| **.dockerignore** | List of files to exclude from the build context. Like .gitignore but for builds. |
| **build context** | The folder uploaded to the build process. Source for COPY/ADD. |
| **build cache** | Cached layers reused on subsequent builds. |
| **BuildKit** | The modern Docker build engine. Parallel, smarter caching, secret support. |
| **buildx** | The Docker CLI plugin for advanced builds, including multi-arch. |
| **multi-stage build** | A Dockerfile with multiple FROM stages; final image is leaner. |
| **multi-arch / multi-platform** | An image that supports multiple CPU architectures (amd64, arm64). |
| **manifest** | JSON descriptor listing an image's config and layers by digest. |
| **manifest list** | A "fat manifest" pointing at multiple per-platform manifests. |
| **OCI** | Open Container Initiative. Standards body for image and runtime specs. |
| **OCI image** | An image in the OCI standard format (the ecosystem standard since ~2017). |
| **OCI runtime** | A program that conforms to the OCI runtime spec. Can launch containers. |
| **runc** | The reference OCI runtime. Tiny Go binary that does the actual container exec. |
| **crun** | A faster OCI runtime written in C. Drop-in for runc. |
| **kata** | An OCI runtime that runs each container in a lightweight VM. |
| **gVisor** | An OCI runtime with a user-space kernel for stronger isolation. |
| **containerd** | A higher-level container runtime. Manages images, containers, snapshots. |
| **ctr** | containerd's bare-bones CLI. |
| **nerdctl** | A Docker-CLI-compatible CLI for containerd. |
| **podman** | Daemonless Docker alternative. Same CLI, runs as the calling user. |
| **buildah** | Build-only tool, often paired with podman. |
| **skopeo** | Image transport tool. Copy/inspect images without unpacking. |
| **registry** | A server that stores images. You push and pull from it. |
| **Docker Hub** | The default public registry, at docker.io. |
| **GHCR** | GitHub Container Registry, ghcr.io. |
| **ECR** | AWS Elastic Container Registry. |
| **GCR** | Google Container Registry (older). Replaced by Artifact Registry. |
| **ACR** | Azure Container Registry. |
| **Quay** | Red Hat's container registry. |
| **Harbor** | Open-source self-hosted registry. |
| **JFrog Artifactory** | Enterprise artifact (incl. container) repository. |
| **namespace** | Kernel feature: partitions a resource so a process sees only its slice. |
| **mount namespace (MNT)** | Each container has its own filesystem mount table. |
| **network namespace (NET)** | Each container has its own interfaces, routing, iptables. |
| **PID namespace** | Each container has its own process tree, with its app as PID 1. |
| **IPC namespace** | Isolation of System V IPC and POSIX message queues. |
| **UTS namespace** | Isolation of hostname / NIS domain. |
| **USER namespace** | Maps UIDs/GIDs between host and container. Powers rootless. |
| **CGROUP namespace** | Isolation of the cgroup view. |
| **cgroup** | Kernel control group. Accounting and limits for CPU, memory, I/O, etc. |
| **capabilities** | Per-power root permissions (e.g. CAP_NET_ADMIN). Containers get a subset. |
| **seccomp** | Kernel syscall filter. Docker ships a default profile blocking dangerous syscalls. |
| **AppArmor** | Linux LSM using path-based MAC rules. Common on Ubuntu/Debian. |
| **SELinux** | Linux LSM using label-based MAC rules. Common on RHEL/Fedora. |
| **rootless** | Running containers as a non-root host user, using user namespaces. |
| **rootful** | Running containers via a root daemon (the classic Docker setup). |
| **Compose** | The tool that brings up a multi-container stack from a YAML file. |
| **docker-compose v1 (legacy)** | Old Python-based Compose. Hyphenated command. Deprecated. |
| **docker compose v2** | Modern Go-based Compose plugin. Space, no hyphen. |
| **service** | A logical unit in Compose: one or more containers from the same image. |
| **volume** | Persistent storage managed by Docker. |
| **named volume** | A volume with a chosen name. |
| **bind mount** | A direct mount of a host path into the container. |
| **tmpfs** | A mount backed by RAM. Disappears with the container. |
| **network (Docker)** | A virtual network containers can attach to. |
| **bridge** | Default Docker network driver. Layer 2 switch + NAT. |
| **host** | Network mode that shares the host's network namespace. |
| **none** | Network mode with no networking. |
| **overlay** | Multi-host network, used by Swarm and similar. |
| **macvlan** | Each container gets its own MAC address on the LAN. |
| **ipvlan** | Like macvlan but at L3. Containers share host MAC. |
| **port mapping / port binding** | Mapping a host port to a container port (-p). |
| **expose** | Declaring a port in the Dockerfile (metadata only). |
| **internal port** | The port inside the container. |
| **host port** | The port on the host the internal port maps to. |
| **restart policy** | Rule for whether/when to restart a stopped container. |
| **no (restart)** | Default. Never restart. |
| **on-failure** | Restart on non-zero exit, optionally up to N times. |
| **always** | Always restart. Stays restarted across daemon restart. |
| **unless-stopped** | Like always, but if the user stopped it, leave it stopped. |
| **entrypoint** | Fixed program for container start. Combined with CMD args. |
| **exec form** | Dockerfile/run instruction in JSON array form. No shell wrapping. |
| **shell form** | Dockerfile/run instruction in plain string form. Wrapped in `/bin/sh -c`. |
| **init process** | The PID 1 process. Reaps zombies, handles signals. |
| **tini** | A tiny init for containers. Use it if your app doesn't reap children. |
| **zombie process** | A child process whose exit hasn't been reaped. PID 1's job to reap. |
| **signal handling** | How a process reacts to SIGTERM, SIGINT, SIGKILL, etc. |
| **healthcheck** | A command Docker runs to check if a container is healthy. |
| **attach** | Connect your terminal to a running container's stdin/stdout/stderr. |
| **detach** | Background a container (or detach from one with Ctrl-P Ctrl-Q). |
| **log driver** | How Docker collects container logs (json-file, journald, syslog, etc.). |
| **json-file** | Default log driver. Writes per-container JSON files on the host. |
| **journald** | Log driver that ships logs to systemd's journal. |
| **syslog** | Log driver that ships logs to a syslog server. |
| **fluentd** | Log driver that ships logs to a Fluentd collector. |
| **awslogs** | Log driver for AWS CloudWatch Logs. |
| **gcplogs** | Log driver for Google Cloud Logging. |
| **loki driver** | Log driver for Grafana Loki. |
| **image scanning** | Checking an image for known vulnerabilities. |
| **Trivy** | A popular open-source vulnerability scanner. |
| **Grype** | Another popular open-source vulnerability scanner. |
| **Snyk** | Commercial vulnerability scanner with a free tier. |
| **Dockle** | Linter for image security best practices. |
| **hadolint** | Linter for Dockerfile best practices. |
| **image signing** | Cryptographically signing an image so consumers can verify it. |
| **Cosign** | Tool for signing/verifying images using OCI signatures. |
| **Notary** | An older image-signing system, used by Docker Content Trust. |
| **image promotion** | Moving an image from a dev registry to a prod registry. |
| **immutable tags** | A registry policy that forbids re-pushing the same tag. |
| **digest** | A SHA-256 hash that uniquely identifies an image's content. |
| **sha256** | The hash algorithm used for digests. |

That's 100+ words. If a word in this sheet isn't in the table, please flag it and we'll add it.

## Try This

Some experiments. Each one teaches a thing.

### Experiment 1: kill the daemon and see what happens

Stop the Docker daemon (`sudo systemctl stop docker` on Linux). Now try `docker ps`. See the "Cannot connect" error. Start it again. See the difference. This builds intuition for "the CLI is just a client; the daemon does everything."

### Experiment 2: identical layer reuse

Build two images that share their bottom layers. `docker images` won't show you the savings, but `docker history` will: identical layer IDs across both. Run `docker system df` before and after building the second image — you'll see the second image adds way less than its full size to disk.

### Experiment 3: the writable layer is ephemeral

```bash
$ docker run --name temp -it alpine sh
/ # echo hello > /important.txt
/ # exit
$ docker start temp
$ docker exec temp cat /important.txt
hello
$ docker rm temp
$ docker run --rm -it alpine sh
/ # cat /important.txt
cat: can't open '/important.txt': No such file or directory
```

The file lived in the writable layer of the first container. When that container was removed, so was the file. A new container starts fresh.

### Experiment 4: bind mount a file you edit live

Bind-mount a folder of source code into a running container. Edit the file on the host. See the change appear instantly inside. This is your dev loop with Docker.

### Experiment 5: blow up your memory limit

```bash
$ docker run --rm -it --memory 64m alpine sh
/ # apk add stress-ng
/ # stress-ng --vm 1 --vm-bytes 200M
... (gets killed quickly)
```

The OOM killer takes you out. Now you've felt cgroups firsthand.

### Experiment 6: see PID 1

Inside any container, `ps aux` shows your app as PID 1 — even though on the host it's PID 12345 or whatever. That's the PID namespace. From the host:

```bash
$ docker top mycontainer
```

Shows the host PID. Different number, same process.

### Experiment 7: write a multi-stage build

Take a small Go or Rust program. Build a single-stage image. Note the size. Now write a multi-stage Dockerfile that copies the binary into `scratch` (Go) or `alpine` (Rust). Compare sizes. Often you go from 800 MB to 15 MB.

### Experiment 8: break the network

```bash
$ docker run --rm --network none alpine ping 8.8.8.8
ping: sendto: Network unreachable
```

No network, no ping. Confirms the netns isolation.

### Experiment 9: read-only root

```bash
$ docker run --rm --read-only alpine sh -c 'echo hi > /tmp/x.txt'
sh: can't create /tmp/x.txt: Read-only file system
$ docker run --rm --read-only --tmpfs /tmp alpine sh -c 'echo hi > /tmp/x.txt && cat /tmp/x.txt'
hi
```

Lock the root filesystem; allow exactly one writable spot.

### Experiment 10: build cache invalidation

Edit your Dockerfile to add `RUN apt-get update` somewhere. Build. Build again, no changes. Note the cache hit. Now change a comment in the Dockerfile **above** the apt line. Build again. Cache invalidated, apt reruns. Now move the apt line **above** the comment. Build. Build again. Comment edits below don't invalidate the apt line. **Cache works on what's above each instruction**, so put your stable layers high.

## Where to Go Next

- `cs containers docker` — dense reference for daily use.
- `cs detail containers/docker` — internals deep dive.
- `cs containers podman` — daemonless alternative.
- `cs containers containerd` — the layer beneath Docker.
- `cs containers docker-compose` — Compose-specific reference.
- `cs ramp-up kubernetes-eli5` — orchestrating many containers across many machines.
- `cs ramp-up linux-kernel-eli5` — what containers actually ARE under the hood.
- `cs security/container-security` — how to harden the whole stack.
- `cs security/container-hardening` — concrete hardening checklists.

## See Also

- `containers/docker`
- `containers/docker-compose`
- `containers/podman`
- `containers/containerd`
- `containers/lxd`
- `orchestration/kubernetes`
- `orchestration/helm`
- `orchestration/kubectl`
- `security/container-security`
- `security/container-hardening`
- `ramp-up/kubernetes-eli5`
- `ramp-up/linux-kernel-eli5`
- `ramp-up/ip-eli5`

## References

- docs.docker.com — official Docker docs.
- "Docker Deep Dive" by Nigel Poulton — friendly, thorough book.
- OCI Image Spec — github.com/opencontainers/image-spec
- OCI Runtime Spec — github.com/opencontainers/runtime-spec
- containerd.io — containerd project home.
- BuildKit — github.com/moby/buildkit
- "Container Security" by Liz Rice — short, pragmatic, essential.
- moby/moby — github.com/moby/moby — the upstream that becomes Docker Engine.
- man dockerd
- man docker-run
- man docker-build
- man docker-compose
