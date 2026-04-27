# systemd — ELI5

> systemd is the conductor of the orchestra that is your Linux machine. The kernel finishes booting, then it taps the conductor on the shoulder and walks off stage. From that point on, every service, every socket, every timer, every log line goes through the conductor's baton.

## Prerequisites

Read **ramp-up/linux-kernel-eli5** first. You need a basic feel for what the kernel is, what a process is, what user space is, what root is, what a syscall is, and what PID 1 means. This sheet picks up exactly where that one ends — at the moment the kernel has finished setting itself up and is about to launch the very first process.

If you have not read that sheet, the rest of this one will still mostly make sense, but a lot of "wait, what's a process?" or "wait, what's a daemon?" will land harder. Go read kernel-ELI5 for an hour, come back, and this sheet will feel obvious.

You do not need to know any C. You do not need to know any system administration. You do not need to have ever used a Linux server before. You do need to have a terminal open. If you see a `$` at the start of a line in a code block, that is a normal user prompt — type the rest of the line, do not type the `$`. If you see a `#` at the start of a line, that is a root prompt — same idea, do not type the `#`, but you will need to use `sudo` (or be the root user) for those commands. Lines without any prompt are output the computer prints back at you.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

## What Even Is systemd

### The conductor of the orchestra

Picture a giant orchestra. There are first violins. There are second violins. There are cellos. There are basses. There is a brass section. There are woodwinds. There is percussion. There is a piano. There is a harp. There is a singer. There are dozens of musicians in this orchestra, and every single one is incredibly skilled at their own instrument.

But put all those musicians in a room with sheet music and tell them "play Beethoven's Ninth" and you will not get music. You will get noise. The first violins will start. The second violins will start a beat later. The brass will come in too loud. The percussion will skip a measure. The singer will not know when to sing. There is no glue. There is no coordination. There is no shared sense of time.

So you put one person at the front of the stage with a baton. That person does not play any instrument. That person is not a virtuoso violinist or a great trumpet player. That person's only job is to make sure all the musicians start at the right time, end at the right time, hit their cues, follow the same tempo, and recover gracefully if one of them messes up.

**That conductor is systemd.**

The musicians are the services on your Linux box. Your web server is a musician. Your database is a musician. Your SSH daemon is a musician. Your printer queue is a musician. Your network manager is a musician. Your time-sync daemon is a musician. Your log collector is a musician. Without coordination, none of them know when to start, none of them know what depends on what, none of them know how to restart cleanly if they crash.

systemd holds the baton. systemd reads the score (the unit files). systemd cues each section. systemd watches every musician and notes when one stops playing. systemd restarts the cellos if they fall over. systemd schedules a timer that says "every Monday at 2am, the percussion plays this little piece." systemd writes down what every musician played and when (the journal) so you can review the concert later.

### Why we needed a conductor

Old Linux systems did not have a conductor. They had a script called `/etc/rc` that ran top to bottom and started everything one at a time. If a service crashed five seconds later, the script did not care — it had already moved on. If two services needed to start in a specific order, you had to manually number the scripts: `S05network`, `S10ssh`, `S15apache`, and you prayed nothing got out of order. If you wanted to restart a service, you ran a separate script. If you wanted logs, you used a totally separate program called syslog that did not know anything about which service produced which log line. If you wanted a job to run every hour, you used cron, which was its own world. If you wanted a service to start only when somebody connected to a port, you used something called `inetd`, which was yet another world. None of these things knew about each other.

systemd merged all those jobs into one tool with one consistent way of describing things. Every service, every timer, every socket, every mount point, every network unit is described in the same kind of file with the same kind of syntax. Every command to start, stop, restart, check, or inspect any of those things is the same command (`systemctl`). Every log line from any of those things lands in the same place (the journal) and you read it with the same command (`journalctl`).

That sounds boring. It is incredibly powerful. Once you learn the conductor's language, you can run the whole orchestra. Without the conductor, you would have to learn five different languages, each with its own quirks, just to start a single service.

### The recipe-book picture

Another way to picture systemd is as a giant recipe book and a chef who follows it. Each recipe in the book is a small file (a unit file). A typical recipe says things like:

- "This recipe is called nginx (a web server)."
- "Before you start this recipe, the network needs to be ready."
- "When you start this recipe, run this command: `/usr/sbin/nginx -g 'daemon off;'`."
- "If this recipe ever burns (the process crashes), wait five seconds and try again."
- "When you stop this recipe, run this command: `/usr/sbin/nginx -s stop`."
- "This recipe should run every time the kitchen opens (every boot)."

The chef is systemd. The chef reads the recipe. The chef follows it exactly. The chef writes down every step in a notebook (the journal). If the recipe fails, the chef notes the failure and either retries or gives up depending on what the recipe says. If two recipes share an oven (a port, a file, a piece of hardware), the chef makes sure they take turns. If a recipe says "this depends on flour being in the pantry first," the chef makes sure flour is delivered before starting that recipe.

You can write your own recipes and put them in the book. You can override existing recipes. You can ask the chef "what are you cooking right now?" and get an instant answer. You can ask the chef "what failed today?" and get a clean list. You can ask the chef "how long did each recipe take to start?" and get exact timings.

### The post office picture

Picture a post office. There is a giant wall of little numbered mailboxes. Each mailbox belongs to a different person or business. Mail for those mailboxes arrives all day from many directions. The post office has rules about how mail is sorted, how mailboxes are opened, who can access which mailbox, and what happens when a mailbox is full.

systemd is the postmaster. The mailboxes are the services. The mail is the work that comes in. Some mail is incoming connections on a port. Some mail is a timer firing. Some mail is a file appearing in a directory. Some mail is a hardware device being plugged in. systemd routes each piece of mail to the right mailbox, opens the mailbox if needed, and writes a delivery slip in the journal.

This picture is most useful when we get to socket activation, path activation, and device units later. The big idea: systemd does not just start services and walk away. systemd is constantly listening for events from the kernel, from timers, from filesystem watches, from D-Bus, and routing those events to the right service.

### Why so many pictures

You will see this trick again and again: the conductor, the recipe book, the post office. Different pictures help with different parts of systemd. The conductor is best for understanding **dependencies and start order**. The recipe book is best for understanding **what a unit file is and how systemd follows it**. The post office is best for understanding **how systemd reacts to events**. If one picture is not clicking for a particular feature, switch to another.

## The Boot Sequence

This is one of the most important pictures to put in your head. From the moment you press the power button to the moment a login prompt appears, here is the chain of events on a typical Linux box.

```
+---------------+        +-----------+        +----------+
|   firmware    |  -->   | bootloader|  -->   |  kernel  |
| (UEFI / BIOS) |        | (GRUB,    |        | (Linux)  |
|               |        |  systemd- |        |          |
|               |        |   boot)   |        |          |
+---------------+        +-----------+        +----------+
        |                                            |
        v                                            v
        power-on self-test,                  initializes hardware,
        find a boot device,                  mounts initramfs,
        load the bootloader.                 launches PID 1.
                                                     |
                                                     v
                                          +----------------------+
                                          |      systemd PID 1   |
                                          +----------------------+
                                                     |
                                                     v
                                          +----------------------+
                                          |   sysinit.target     |
                                          |   (mount real root,  |
                                          |    load modules,     |
                                          |    set hostname,     |
                                          |    create tmpfs etc.)|
                                          +----------------------+
                                                     |
                                                     v
                                          +----------------------+
                                          |    basic.target      |
                                          | (sockets, timers,    |
                                          |  paths, slices ready)|
                                          +----------------------+
                                                     |
                                                     v
                                          +----------------------+
                                          |  multi-user.target   |
                                          | (most services up:   |
                                          |  ssh, cron, syslog,  |
                                          |  network manager...) |
                                          +----------------------+
                                                     |
                                                     v
                                          +----------------------+
                                          |  graphical.target    |
                                          | (display manager,    |
                                          |  GNOME/KDE/etc.)     |
                                          +----------------------+
                                                     |
                                                     v
                                          +----------------------+
                                          |   getty / login      |
                                          | (you see a prompt)   |
                                          +----------------------+
```

Step by step, in plain English:

**1. Firmware (UEFI or BIOS).** When you press the power button, the very first code that runs is baked into a chip on the motherboard. On modern machines this is **UEFI**. On older machines it was **BIOS**. The firmware does a power-on self-test, looks at the boot order list, picks a disk, and reads the first chunk of code off that disk. The firmware does not know what Linux is. The firmware does not know what systemd is. The firmware just knows "load this little piece of code from this disk and run it."

**2. Bootloader.** That little piece of code is the bootloader. On most Linux boxes this is **GRUB** (the GRand Unified Bootloader). On systemd-y systems it can be **systemd-boot** instead. The bootloader's job is to find a kernel on disk, load it into memory, and jump into it. The bootloader can also show a menu of different kernels you might want to boot. The bootloader is not systemd.

**3. Kernel.** The kernel takes over. It probes hardware. It sets up memory protection. It mounts an early little temporary filesystem called the **initramfs** which contains just enough tools to find the real root filesystem. It does its kernel things (see the kernel-ELI5 sheet). When everything is ready, the kernel does one more crucial thing: it launches the very first user-space process, **PID 1**.

**4. PID 1 is systemd.** On a modern Linux distribution, PID 1 is `/usr/lib/systemd/systemd` (or `/sbin/init`, which is usually a symlink to it). This is the moment systemd starts conducting. PID 1 is the parent of every other process on the machine. If PID 1 dies, the kernel panics. Nothing on a running Linux box is more important than PID 1.

**5. systemd reads its unit files.** systemd looks in three directories for unit files:

```
/usr/lib/systemd/system/   <-- shipped by your distro and packages (don't edit)
/run/systemd/system/       <-- runtime, generated, ephemeral
/etc/systemd/system/       <-- your overrides and your custom units (edit here)
```

Files in `/etc/systemd/system/` win over files with the same name in `/usr/lib/systemd/system/`. That is the override mechanism. Files in `/run/systemd/system/` win over both — they are temporary and gone after reboot.

**6. systemd activates `default.target`.** A target is a named goal that systemd tries to reach. The default target is whatever the system is set to boot into. On servers, this is usually `multi-user.target`. On desktops, this is usually `graphical.target`. systemd looks at the dependency graph rooted at the default target and works out everything it needs to start.

**7. systemd marches up the target tree.** Each target depends on others. `graphical.target` depends on `multi-user.target`. `multi-user.target` depends on `basic.target`. `basic.target` depends on `sysinit.target`. systemd walks the tree, starting things in the right order, in parallel where it can, and waiting where it must. This is exactly what the conductor does: cue the cellos, then the violas, then bring in the brass, then the percussion. Each section comes in at the right moment, but lots of sections can play simultaneously once they have entered.

**8. Services come up.** As the targets activate, services that say `WantedBy=multi-user.target` (or whatever) get pulled in. SSH starts. The network manager starts. cron starts. `systemd-journald` (the log daemon) was already running really early. `systemd-logind` (the login session daemon) starts. The display manager starts. Each service writes its log lines to the journal as it comes up.

**9. getty (or a display manager) gives you a login.** The very last thing you usually see is a getty (a tiny program that runs on a virtual console and prompts you for a username) or a graphical display manager (gdm, sddm, lightdm). You log in. You are talking to PID 1's grandchildren. systemd is still conducting, will be conducting until you shut the machine down, and is the parent of every process you can see.

If you ever want to watch this happen in real time, reboot the machine and watch the boot messages, or run `systemd-analyze` after boot to get a timing report. We get to those commands later.

## Unit Types

systemd does not just manage services. systemd manages eleven different kinds of "thing," and they all use the same unit-file format. Each kind is called a **unit type**. The file extension tells systemd which type. Here are the eleven you will meet most often.

```
.service     -- a long-running daemon or a one-shot script
.socket      -- a listening socket that activates a .service on first connection
.timer       -- a scheduled trigger that activates a .service at given times
.target      -- a named "goal" used to group other units together
.mount       -- a mount point (like /mnt/data)
.automount   -- a lazy mount that mounts on first access
.path        -- a watcher that activates a .service when a file or dir changes
.device      -- a kernel device, surfaced by udev (you rarely write these)
.swap        -- a swap area
.slice       -- a node in the cgroup tree, used for resource control
.scope       -- like a service but managed externally (logged in sessions, etc.)
```

A short tour of each one.

### .service

The bread-and-butter unit. Describes a process to run. Examples: `nginx.service`, `sshd.service`, `postgresql.service`. We will spend most of this sheet on services because they are the most common thing you will write or edit.

### .socket

A listening socket (TCP port, UDP port, Unix domain socket). When a connection arrives, systemd starts the matching `.service` unit on demand. Example: `cups.socket` listens on a Unix socket; the moment somebody sends a print job, `cups.service` launches. This is **socket activation**, an old idea borrowed from inetd and refined.

### .timer

A scheduled trigger. Replaces cron. Example: `logrotate.timer` fires daily and starts `logrotate.service`. Two flavors: **monotonic timers** (counted from boot, like "ten seconds after boot") and **realtime timers** (calendar-based, like "every Sunday at 3am").

### .target

A named goal. No process runs as a target; targets exist to group other units. Examples: `multi-user.target`, `network-online.target`, `default.target`. Targets are roughly the systemd equivalent of old SysV runlevels, but you can have as many as you want.

### .mount

Describes a mount point. systemd reads `/etc/fstab` at boot and turns each entry into a `.mount` unit automatically, but you can also write `.mount` files by hand. Naming is fiddly: a mount unit for `/mnt/data` is named `mnt-data.mount` (slashes become dashes).

### .automount

Pairs with a `.mount`. Tells the kernel "do not mount this filesystem yet, but the moment somebody touches the mount point, mount it for real." Useful for big remote shares that you do not want to mount until needed.

### .path

A watcher. Tells systemd "fire this `.service` when this file appears" or "when this directory changes" or "when this file becomes non-empty." Replaces some uses of inotify scripts.

### .device

systemd auto-creates a `.device` unit for every kernel device that has a sysfs entry, via udev. You almost never write one of these by hand. Other units can depend on `.device` units to wait for hardware to appear.

### .swap

Describes a swap partition or swap file. Like mount units, systemd usually generates these from `/etc/fstab`.

### .slice

A node in the cgroup tree. Used for resource control. `system.slice` holds all system services. `user.slice` holds all user sessions. You can create your own slices and group services inside them so they share a memory or CPU budget.

### .scope

Like a service, but the process was started by something other than systemd, and systemd is just keeping track of it for accounting and cgroups. Each logged-in user session is a `.scope`. So is each `systemd-run --scope ...` invocation.

## The Unit File Format

Every unit file is a plain text file in INI format. Section headers go in `[Brackets]`. Key/value pairs go below the header. Comments start with `#` or `;`. Whitespace around the `=` is not significant.

A skeleton service unit looks exactly like this:

```ini
[Unit]
Description=Hello world service
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/hello-world
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Three sections:

- `[Unit]` — generic stuff that applies to every unit type: description, dependencies, ordering. This section exists in every unit.
- `[Service]` — settings specific to the `.service` type: how to start it, how to stop it, how to restart, what user, what environment. Other unit types have their own type-specific section: `[Socket]`, `[Timer]`, `[Mount]`, `[Path]`, etc.
- `[Install]` — settings for `systemctl enable` and `systemctl disable`. The `[Install]` section is read **only** when you enable or disable the unit; it is ignored at boot. This is a common confusion — see Common Confusions below.

Multiple values can be set in two ways. Either repeat the key:

```ini
ExecStartPre=/usr/local/bin/prep-1
ExecStartPre=/usr/local/bin/prep-2
```

Or join with whitespace where the option allows it:

```ini
After=network.target sshd.service
```

To **clear** a list option that was set in a vendor unit, set it to empty first:

```ini
ExecStart=
ExecStart=/usr/local/bin/new-binary
```

That trick is essential when you write drop-in overrides — see the Drop-Ins section.

## A Hello-World Service

Let's build a real service from scratch. Open a terminal.

First, write a tiny program that just loops forever and prints the time. Save this as `/usr/local/bin/hello-world`:

```bash
#!/bin/bash
while true; do
  echo "hello from $$ at $(date)"
  sleep 5
done
```

Make it executable:

```bash
$ sudo chmod +x /usr/local/bin/hello-world
```

Now write the unit file. Put it in `/etc/systemd/system/hello-world.service`:

```ini
[Unit]
Description=A friendly hello-world service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/hello-world
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Tell systemd to re-read its unit files:

```bash
$ sudo systemctl daemon-reload
```

Start it:

```bash
$ sudo systemctl start hello-world.service
```

Check on it:

```bash
$ systemctl status hello-world.service
* hello-world.service - A friendly hello-world service
     Loaded: loaded (/etc/systemd/system/hello-world.service; disabled; vendor preset: enabled)
     Active: active (running) since Mon 2026-04-27 10:00:01 UTC; 5s ago
   Main PID: 12345 (hello-world)
      Tasks: 2 (limit: 9000)
     Memory: 1.4M
        CPU: 0.005s
     CGroup: /system.slice/hello-world.service
             |-12345 /bin/bash /usr/local/bin/hello-world
             `-12350 sleep 5

Apr 27 10:00:01 box hello-world[12345]: hello from 12345 at Mon Apr 27 10:00:01 UTC 2026
Apr 27 10:00:06 box hello-world[12345]: hello from 12345 at Mon Apr 27 10:00:06 UTC 2026
```

Look at all the things you got for free:

- A `Loaded:` line tells you which file systemd is using.
- An `Active:` line tells you the current state.
- A `Main PID:` line ties the unit to the actual kernel process.
- A `Tasks:` count, a `Memory:` measurement, and a `CPU:` time. systemd tracks these automatically because every service runs in its own cgroup.
- A `CGroup:` tree shows the parent process and any children.
- The last few log lines from this service inline at the bottom.

To stop it:

```bash
$ sudo systemctl stop hello-world.service
```

To make it start at every boot:

```bash
$ sudo systemctl enable hello-world.service
Created symlink /etc/systemd/system/multi-user.target.wants/hello-world.service -> /etc/systemd/system/hello-world.service.
```

To start it now and at every boot:

```bash
$ sudo systemctl enable --now hello-world.service
```

To stop it now and not start it next boot:

```bash
$ sudo systemctl disable --now hello-world.service
```

That's it. You wrote a service. systemd is now responsible for keeping it alive, restarting it on crash, capturing its output to the journal, and starting it at boot. The kernel makes processes possible. systemd makes services manageable.

### ExecStart, ExecStop, ExecReload, ExecStartPre, ExecStartPost

Beyond `ExecStart`, you have:

- `ExecStartPre=` — runs before `ExecStart`. Often used for pre-flight checks (config validation, directory creation).
- `ExecStartPost=` — runs after `ExecStart` succeeds. Often used to notify other systems.
- `ExecStop=` — runs to stop the service gracefully. If omitted, systemd sends `SIGTERM` to the main process.
- `ExecStopPost=` — runs after the process has stopped, regardless of success or failure. Cleanup code goes here.
- `ExecReload=` — runs when you do `systemctl reload`. Should reload config without restarting. Often `kill -HUP $MAINPID`.
- `ExecCondition=` — runs first; if it exits non-zero, systemd treats the service as not-needed and skips it. Use for "only run this on Tuesdays" guards.

Each `Exec*=` option can be repeated. Each call can have a leading `-` (ignore failure) or `@` (special argv0) or `+` (run as root regardless of `User=`) or `!` (run with elevated privileges before User= drop). These prefixes stack up in fiddly ways; the man page has the full list.

### Restart= and the restart policies

The `Restart=` option tells systemd what to do when the main process exits. Possible values:

- `no` — never restart (default).
- `on-success` — restart only if the process exited cleanly with code 0.
- `on-failure` — restart only if the process crashed, exited non-zero, was killed, or hit a watchdog timeout. **Most common.**
- `on-abnormal` — restart on signals or watchdog timeouts but not on clean non-zero exits.
- `on-abort` — restart only on uncaught signals.
- `on-watchdog` — restart only when the watchdog fires.
- `always` — always restart, no matter what. Good for "must always be running" jobs but watch out for restart loops.

Pair with `RestartSec=5` (wait five seconds before restarting), `StartLimitBurst=3` (allow at most three restarts), and `StartLimitIntervalSec=60` (within sixty seconds), or systemd will keep restarting forever if the service is broken. After hitting the burst limit, systemd marks the unit as `failed` and refuses to restart until you `systemctl reset-failed`.

## Service Types

The `Type=` setting tells systemd how to know when the service has actually started. This matters because other units may be waiting for this one to "come up" before they start. Pick the wrong type and dependencies fire too early, processes get killed unexpectedly, or systemd thinks the service crashed when it didn't.

### Type=simple

The default. systemd runs your `ExecStart=` command and considers the service "started" the moment the process is forked. systemd does not wait for anything. If your program needs to do setup work before it is "ready," and other services depend on it being ready, `simple` will fire those dependencies too early.

Use when: your service does not fork, does not double-fork, has nothing fancy to signal back, and "running" is good enough for "ready."

### Type=exec

Like `simple`, but systemd waits until the binary has been `execve()`'d (i.e., until the kernel has actually started running your program, not just forked the parent). Modestly safer than `simple`. Came in systemd 240. Use as a drop-in replacement for `simple`.

### Type=forking

For old-school daemons that fork into the background. systemd runs your `ExecStart=`, waits for the parent to exit, and considers the child the new main process. Often paired with `PIDFile=/run/foo.pid` so systemd knows which child is the real daemon. Many old services are still `forking` because they predate systemd. New services should not be `forking`.

### Type=oneshot

The service runs a command, the command exits, and the service is "done." Useful for setup tasks (clear a cache, build a config). Pair with `RemainAfterExit=yes` to make the service stay "active" in systemctl status even after the command exits — this is how you make a oneshot satisfy `Wants=`/`Requires=` from other units.

### Type=notify

The service tells systemd "I am ready" by calling `sd_notify(READY=1)` on a magic Unix socket whose path is in the environment variable `$NOTIFY_SOCKET`. systemd waits for that message before considering the service started. This is the **best** type for modern services because it is precise — no guessing.

To make it work, the program must be linked against libsystemd or call `sd_notify` directly. Many languages have libraries (`go-systemd`, `python-systemd`). Set `NotifyAccess=main` (only the main process can notify), `NotifyAccess=all` (any child can), or `NotifyAccess=none`.

### Type=dbus

The service registers a name on the D-Bus system bus, and systemd considers it ready once the bus name is taken. Good for D-Bus services. Set `BusName=org.example.Foo`.

### Type=idle

Like `simple`, but waits until all other startup jobs have finished before launching. Used for jobs that print a lot of output to the console (like getty) so they don't get tangled up in earlier boot messages. Rarely useful outside of getty.

## Targets

A target is a named goal. No process belongs to a target. A target exists so that other units can depend on it ("when this target is reached, also start me") and so that humans have a name they can isolate to ("isolate to this target means: shut down everything not needed for it, then activate it").

Common targets you will see:

- `default.target` — a symlink that points to the actual default. Usually `multi-user.target` on servers, `graphical.target` on desktops.
- `multi-user.target` — text-mode multi-user system. Most services up. No GUI. Like SysV runlevel 3.
- `graphical.target` — multi-user plus a display manager. Like SysV runlevel 5.
- `rescue.target` — single-user shell on the main console with the basic system mounted. For maintenance.
- `emergency.target` — minimal shell. Only the emergency console works. Filesystems are read-only. For severe rescue.
- `sysinit.target` — the very early stuff: mounts, kernel modules, hostname, tmpfiles, sysctl. Reached before basic.
- `basic.target` — sockets, timers, paths, slices ready. Nothing user-visible yet.
- `network.target` — the network stack is configured (interfaces are up, addresses assigned). Does not mean any host is reachable.
- `network-online.target` — the network is up **and** at least one route is usable. Stronger than `network.target`. Use this if your service needs to reach external hosts at startup. Activated by `systemd-networkd-wait-online` or `NetworkManager-wait-online`.
- `sockets.target` — all socket units active.
- `timers.target` — all timer units active.
- `shutdown.target` — engaged during shutdown.
- `poweroff.target`, `reboot.target`, `halt.target` — the three shutdown destinations.

To switch targets at runtime:

```bash
$ sudo systemctl isolate multi-user.target   # graphical -> text mode
$ sudo systemctl isolate rescue.target       # shut down everything but the basics
```

To change the default target permanently:

```bash
$ sudo systemctl set-default multi-user.target
$ sudo systemctl get-default
multi-user.target
```

Visual: the target dependency graph for a typical desktop boot looks like this.

```
                       graphical.target
                              |
                              v
                      multi-user.target
                              |
              +---------------+---------------+
              |               |               |
              v               v               v
          ssh.service   cron.service    NetworkManager.service ...
              |               |               |
              +---------------+---------------+
                              |
                              v
                        basic.target
                              |
                              v
                       sysinit.target
                              |
                              v
                        local-fs.target
                              |
                              v
                          (root mounted)
```

Each target sits on the layer above its dependencies. Going up activates more functionality. Going down (during shutdown) deactivates it.

## Dependencies

systemd has eight dependency keywords. They live in the `[Unit]` section. They split into two groups: **wiring** (who must run for me to run, and what happens if they fail) and **ordering** (who must run before me).

### Wiring (existence + status)

- `Requires=A` — if A fails to start, my service also fails. If A stops, I stop. **Hard dependency.**
- `Requisite=A` — like `Requires=` but A must already be running; do not start A on my behalf, fail immediately if it isn't up.
- `Wants=A` — start A when starting me, but if A fails, I am still fine. **Soft dependency.** Most common.
- `BindsTo=A` — like `Requires=` but stronger: if A stops for any reason (including success), I stop too. Used to glue a service tightly to a device.
- `PartOf=A` — if A is restarted or stopped, I am restarted/stopped. The reverse is not true. Used for grouped services.
- `Conflicts=A` — starting me stops A; starting A stops me. Used for mutually-exclusive services.

### Ordering (start order)

- `Before=A` — I must finish starting before A starts.
- `After=A` — A must finish starting before I start.

`Before=` and `After=` do **nothing** by themselves to wire units together. They only define ordering if both units are activated. You almost always combine an ordering keyword with a wiring keyword, e.g.:

```ini
Wants=network-online.target
After=network-online.target
```

That says: pull in `network-online.target` if not already there (`Wants=`), and wait for it to finish before starting me (`After=`). Without the `After=`, my service might start before the network is online, which defeats the point.

### Cheat sheet: what to use when

| You want                                                | Use                              |
|---------------------------------------------------------|----------------------------------|
| If A fails, do not run me                               | `Requires=A` + `After=A`         |
| Try to bring A up but I will limp on without it         | `Wants=A` + `After=A`            |
| Restart me whenever A is restarted                      | `PartOf=A`                       |
| If A vanishes, I should vanish too                      | `BindsTo=A` + `After=A`          |
| Replace an existing service with mine                   | `Conflicts=A`                    |
| I must finish before A even tries to start              | `Before=A`                       |

## Socket Activation

Socket activation is one of systemd's superpowers and one of the most confusing things to wrap your head around at first.

### The idea

Instead of running a daemon all the time on the off chance somebody connects, you let systemd hold a listening socket. The daemon is not running yet. The moment a connection arrives, systemd hands the open socket to a freshly-started instance of the daemon.

Why would you want this? Two reasons:

1. **Lazy startup.** The daemon does not consume any resources until somebody actually uses it. Boot is faster because you skipped starting it. Memory is freed because there is no idle daemon.
2. **No lost connections during restart.** systemd holds the socket across restarts of the daemon. The kernel does not lose pending connections in the socket queue. You can restart the daemon and clients see no disruption other than maybe a brief delay.

### The two-file pattern

Socket activation needs a pair of files: one `.socket` and one `.service`. They share a name.

`/etc/systemd/system/hello.socket`:

```ini
[Unit]
Description=Hello world socket

[Socket]
ListenStream=12345
Accept=no

[Install]
WantedBy=sockets.target
```

`/etc/systemd/system/hello.service`:

```ini
[Unit]
Description=Hello world service (socket-activated)
Requires=hello.socket
After=hello.socket

[Service]
ExecStart=/usr/local/bin/hello-server
StandardInput=socket
StandardOutput=socket
```

When you enable and start `hello.socket`, systemd binds to TCP port 12345 itself. The `hello.service` unit is not running. The first time somebody connects to port 12345, systemd accepts the connection and launches `hello.service`, passing the socket to it via stdin/stdout (because `StandardInput=socket` and `StandardOutput=socket`). The service does its work. When it exits, systemd remains bound to the port and is ready for the next connection.

`Accept=no` (the default) means the socket itself is passed to one long-running service. `Accept=yes` means one new instance of the service is started **per connection**, like inetd. With `Accept=yes` you use templated services (`hello@.service`) so each connection gets its own unit.

The `ListenStream=` directive accepts:

- A port number: `ListenStream=12345` (binds on all addresses).
- An IP and port: `ListenStream=127.0.0.1:12345`.
- A Unix socket path: `ListenStream=/run/hello.sock`.
- An abstract socket: `ListenStream=@hello`.
- IPv6: `ListenStream=[::]:12345`.

Cousins of `ListenStream=`:

- `ListenDatagram=` — UDP.
- `ListenSequentialPacket=` — SEQPACKET.
- `ListenFIFO=` — named pipes.
- `ListenSpecial=` — character devices.
- `ListenNetlink=` — kernel netlink sockets.
- `ListenMessageQueue=` — POSIX message queues.

### The flow

```
1. Boot:        systemd activates sockets.target, which pulls in hello.socket.
                hello.socket binds to port 12345. hello.service is NOT running.

2. Idle:        port 12345 is held by systemd (not by hello.service).
                The kernel's accept queue waits for clients.

3. Client:      curl http://localhost:12345/  -- a SYN arrives.

4. Activation:  systemd sees the pending connection. It launches hello.service
                and hands it the file descriptor for the listening socket
                (or the connected socket, with Accept=yes).

5. Serving:     hello.service responds. systemd is no longer in the data path;
                it just keeps an eye on the service.

6. Restart:     systemctl restart hello.service. systemd kills the daemon
                but keeps holding the socket. New connections queue.
                The new daemon picks up the socket. Clients never noticed.
```

This is also how `cups.socket`, `dbus.socket`, `docker.socket`, and many other system services work today. SSH can be socket-activated too.

## Timer Units

Timer units replace cron. A timer fires on a schedule and starts a matching service unit. The two-file pattern again: a `.timer` and a `.service`.

`/etc/systemd/system/cleanup.service`:

```ini
[Unit]
Description=Nightly tmp cleanup

[Service]
Type=oneshot
ExecStart=/usr/local/bin/cleanup-tmp
```

`/etc/systemd/system/cleanup.timer`:

```ini
[Unit]
Description=Run cleanup-tmp nightly

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=30min
AccuracySec=1min

[Install]
WantedBy=timers.target
```

Enable and start the timer, not the service:

```bash
$ sudo systemctl enable --now cleanup.timer
```

Now systemd will start `cleanup.service` once a day, near midnight, with up to 30 minutes of randomized jitter. If the machine was off at midnight, `Persistent=true` makes systemd run the missed job at next boot.

### Realtime triggers (calendar-based)

`OnCalendar=` accepts a flexible mini-language:

- `OnCalendar=daily` — every day at 00:00.
- `OnCalendar=hourly` — every hour at :00.
- `OnCalendar=Mon..Fri 09:00` — weekdays at 9am.
- `OnCalendar=*-*-* 03:30:00` — every day at 3:30am.
- `OnCalendar=*-*-1 04:00` — first day of every month at 4am.
- `OnCalendar=Sun *-*-* 02:00:00` — every Sunday at 2am.
- `OnCalendar=2026-04-27 12:00:00` — exactly once.

Test what a calendar expression resolves to:

```bash
$ systemd-analyze calendar 'Mon..Fri 09:00'
  Original form: Mon..Fri 09:00
Normalized form: Mon..Fri *-*-* 09:00:00
    Next elapse: Mon 2026-04-27 09:00:00 UTC
       From now: 22h left
```

### Monotonic triggers (counted from boot)

These count from a reference point in the system's life rather than the wall clock:

- `OnBootSec=10min` — 10 minutes after boot.
- `OnStartupSec=5min` — 5 minutes after systemd started (similar but slightly different on user buses).
- `OnActiveSec=1min` — 1 minute after the timer was last activated.
- `OnUnitActiveSec=1h` — 1 hour after the matching service was last activated.
- `OnUnitInactiveSec=15min` — 15 minutes after the matching service last became inactive.

Mix them. A common health-check timer uses `OnUnitActiveSec=5min` so the service runs every 5 minutes regardless of how long it took.

### Calendar-vs-monotonic differences

| Aspect             | OnCalendar                    | OnBootSec / OnUnitActiveSec   |
|--------------------|--------------------------------|--------------------------------|
| Reference          | Wall clock                    | Boot time / unit timing       |
| Survives reboot    | Yes                           | No (counts from new boot)     |
| Persistent= flag   | Catches missed runs after boot | Not applicable                 |
| Time-zone aware    | Yes (use `Timezone=` or system TZ) | No                       |

`Persistent=true` only works with `OnCalendar=`. systemd writes a small file under `/var/lib/systemd/timers/` recording the last fire time and uses it to fire missed runs after a reboot.

`AccuracySec=1min` lets systemd batch nearby timer fires within a window to save power. Default is 60s. Set lower for tighter scheduling.

`RandomizedDelaySec=30min` adds jitter so a thousand machines don't all hit the same backend at the exact same instant.

`WakeSystem=true` (rare) wakes the machine from suspend to fire the timer.

To list every timer:

```bash
$ systemctl list-timers --all
NEXT                         LEFT       LAST                         PASSED  UNIT             ACTIVATES
Tue 2026-04-28 00:00:00 UTC  13h left   Mon 2026-04-27 00:00:14 UTC  10h ago cleanup.timer    cleanup.service
Tue 2026-04-28 06:42:00 UTC  19h left   Mon 2026-04-27 06:42:00 UTC  4h ago  logrotate.timer  logrotate.service
```

## Path Units

A `.path` unit watches a path on disk and activates a `.service` when something changes.

`/etc/systemd/system/spool.path`:

```ini
[Unit]
Description=Watch /var/spool/incoming for new files

[Path]
PathChanged=/var/spool/incoming
Unit=spool-process.service

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/spool-process.service`:

```ini
[Unit]
Description=Process the spool directory

[Service]
Type=oneshot
ExecStart=/usr/local/bin/process-spool
```

The `[Path]` section accepts:

- `PathExists=/path` — fire when the path exists.
- `PathExistsGlob=/dir/*.txt` — fire when any path matching the glob exists.
- `PathChanged=/path` — fire when the file or directory is modified and the modifying process closes it.
- `PathModified=/path` — like `PathChanged` but fires while modification is happening, not just on close.
- `DirectoryNotEmpty=/path` — fire when the directory has at least one file in it.

Path units use the kernel's inotify mechanism under the hood. They are good for "drop a file in this directory and trigger a job" patterns, replacing crude cron-driven polling.

## Mount and Automount Units

systemd reads `/etc/fstab` at boot and synthesizes `.mount` units for each line. You usually just edit `fstab` like normal. But you can write a `.mount` unit by hand for finer control.

The naming rule: a mount unit for `/srv/data` is named `srv-data.mount`. Slashes become dashes. Special characters get escaped (use `systemd-escape -p /weird/path` to compute the right name).

`/etc/systemd/system/srv-data.mount`:

```ini
[Unit]
Description=Mount /srv/data

[Mount]
What=/dev/disk/by-uuid/12345678-90ab-cdef-1234-567890abcdef
Where=/srv/data
Type=ext4
Options=defaults,noatime

[Install]
WantedBy=multi-user.target
```

Pair with a `.automount` unit if you want lazy mounting:

`/etc/systemd/system/srv-data.automount`:

```ini
[Unit]
Description=Automount /srv/data on demand

[Automount]
Where=/srv/data
TimeoutIdleSec=10min

[Install]
WantedBy=multi-user.target
```

Now `srv-data.mount` will not be mounted at boot. The first time anything reads `/srv/data`, the kernel triggers systemd, systemd activates the `.mount`, the filesystem comes up, and the read succeeds. After 10 minutes of nobody touching it, systemd unmounts it again. Great for big NFS shares.

## Slice Units and Resource Control

Every service runs in its own cgroup. Cgroups are a kernel feature that lets you measure and limit CPU, memory, IO, and other resources per group of processes. systemd builds a cgroup tree:

```
-.slice
 |-- system.slice
 |    |-- nginx.service
 |    |-- postgresql.service
 |    `-- ssh.service
 |-- user.slice
 |    |-- user-1000.slice
 |    |    |-- user@1000.service
 |    |    |    |-- gnome-shell.service
 |    |    |    `-- pulseaudio.service
 |    |    `-- session-3.scope
 `-- machine.slice
      `-- machine-mycontainer.scope
```

Each slice can carry resource limits that apply to everything inside it. Inside a slice, individual services can have their own tighter limits.

### Resource controls you will use

In `[Service]` (or in `[Slice]`):

- `MemoryMax=2G` — hard cap. Going over triggers the OOM killer for the cgroup.
- `MemoryHigh=1.5G` — soft cap. Over this, processes are throttled, not killed.
- `MemorySwapMax=0` — no swap usage at all.
- `CPUQuota=50%` — at most half a core's worth of CPU.
- `CPUWeight=100` — relative weight when there is contention. Default 100.
- `IOWeight=200` — block-IO bandwidth weight.
- `IOReadBandwidthMax=/dev/sda 50M` — cap read bandwidth.
- `IOWriteBandwidthMax=/dev/sda 50M` — cap write bandwidth.
- `TasksMax=512` — maximum number of tasks (processes + threads) in the cgroup.
- `IPAddressDeny=any` and `IPAddressAllow=10.0.0.0/8` — built-in IP firewall, applied per-cgroup using BPF.

systemd exposes all of this through the unified cgroup v2 hierarchy. As of systemd 230 (2016), cgroup v2 unified hierarchy is the default on most modern distros.

To watch live cgroup usage:

```bash
$ systemd-cgtop
Path                                       Tasks   %CPU   Memory  Input/s Output/s
/                                            312    8.4    1.9G        -        -
/system.slice                                152    3.5    920M        -        -
/system.slice/postgresql.service              31    1.8    412M        -        -
/system.slice/nginx.service                    9    0.4    18.0M       -        -
/user.slice                                  158    4.8    980M        -        -
```

Press `q` to quit.

## journald

systemd ships with its own log daemon called **systemd-journald**. It runs as `systemd-journald.service` and is started extremely early, even before sysinit.target, so it can capture messages from the kernel and the very first user-space programs.

What journald collects:

- **stdout/stderr** of every service whose unit has `StandardOutput=journal` (the default).
- **syslog messages** from any program that calls the syslog API.
- **kernel ring buffer** messages (the things you see with `dmesg`).
- **structured records** from any program that calls `sd_journal_send()` directly.
- **process metadata** for every line: PID, UID, GID, executable name, command line, cgroup, capabilities, hostname, boot ID, transport.

That metadata is the killer feature. Every line in the journal is a structured record, not just a string. You can filter by any field. You can ask "show me everything from this PID across reboots." You can ask "show me everything that ran as UID 1000 in the last hour."

### Storage modes

`/etc/systemd/journald.conf` controls where logs go. The `Storage=` setting can be:

- `volatile` — only `/run/log/journal/`, gone on reboot.
- `persistent` — `/var/log/journal/`, kept across reboots.
- `auto` — persistent if `/var/log/journal/` exists, else volatile. **Default.**
- `none` — discard everything. Bad idea.

To switch to persistent storage:

```bash
$ sudo mkdir -p /var/log/journal
$ sudo systemd-tmpfiles --create --prefix /var/log/journal
$ sudo systemctl restart systemd-journald
```

Other useful settings:

- `SystemMaxUse=2G` — never exceed 2 GB on disk.
- `SystemKeepFree=1G` — leave at least 1 GB free.
- `MaxRetentionSec=2week` — drop entries older than two weeks.
- `MaxFileSec=1month` — rotate files monthly.
- `ForwardToSyslog=yes` — also send to a classic syslog daemon if you have one.

### journald flow diagram

```
+-----------------+      +-----------------+      +------------------+
| service stdout  |--+   |  syslog API     |--+   |  kernel ring     |
| service stderr  |  |   |  (libc syslog)  |  |   |  buffer (dmesg)  |
+-----------------+  |   +-----------------+  |   +------------------+
                     |                        |              |
                     v                        v              v
                +----+------+--------+--------+-----+-------------+
                |        systemd-journald (PID N)                |
                |  - tags every record with PID/UID/cgroup/etc.  |
                |  - de-duplicates                                |
                |  - rate-limits abusive sources                  |
                +-----------------+-------------------------------+
                                  |
                            stores to:
                                  |
                  +---------------+----------------+
                  |                                |
                  v                                v
       /run/log/journal/<id>/         /var/log/journal/<id>/
       (volatile, RAM)                 (persistent, disk)
                  |                                |
                  +---------------+----------------+
                                  |
                                  v
                         journalctl reads here
```

## journalctl Power User Cookbook

`journalctl` is your one tool for reading the journal. Some recipes that come up daily:

```bash
# Tail every log line, like tail -f
$ journalctl -f

# Logs for one service
$ journalctl -u nginx.service

# Since today (00:00 today)
$ journalctl --since=today

# Since two hours ago
$ journalctl --since="2 hours ago"

# Between two times
$ journalctl --since="2026-04-27 09:00" --until="2026-04-27 10:00"

# Only errors and worse (priority 0..3)
$ journalctl -p err

# Just this boot
$ journalctl --boot

# Previous boot
$ journalctl --boot=-1

# By PID
$ journalctl _PID=1234

# By UID
$ journalctl _UID=1000

# By executable name
$ journalctl _COMM=sshd

# Combined (AND)
$ journalctl _COMM=sshd --since=today -p warning

# Combined (OR with +)
$ journalctl _COMM=sshd + _COMM=nginx

# Show structured fields (every field, not just MESSAGE)
$ journalctl -o verbose -u nginx --since=today

# Output as JSON
$ journalctl -o json --no-pager -u nginx | head -1

# Disk usage of the journal
$ journalctl --disk-usage
Archived and active journals take up 1.2G in the file system.

# Trim to keep only last 500 MB
$ sudo journalctl --vacuum-size=500M

# Trim to keep only last two weeks
$ sudo journalctl --vacuum-time=2weeks

# Trim to keep only the latest 10 files
$ sudo journalctl --vacuum-files=10

# Verify the journal is not corrupted
$ journalctl --verify

# Read /var/log/journal from a recovered disk
$ journalctl -D /mnt/recovered/var/log/journal

# Log a one-shot message (great for cron-style scripts)
$ logger "manual message"
$ systemd-cat -t myscript -p info echo "hello"
```

`-f`, `--since=`, `--until=`, `-u`, `-p`, `-b`, `_COMM=`, and `--vacuum-*` are the seven you should memorize.

## systemd-analyze (Boot Timing)

`systemd-analyze` is a profiler for boot.

```bash
$ systemd-analyze
Startup finished in 1.823s (firmware) + 4.917s (loader) + 3.012s (kernel) + 12.481s (userspace) = 22.234s
graphical.target reached after 12.481s in userspace.

$ systemd-analyze blame
   3.812s NetworkManager-wait-online.service
   2.105s systemd-cryptsetup@cryptroot.service
   1.998s systemd-journal-flush.service
   ...

$ systemd-analyze critical-chain
The time when unit became active or started is printed after the "@" character.
The time the unit took to start is printed after the "+" character.

graphical.target @12.481s
`-multi-user.target @12.481s
  `-NetworkManager-wait-online.service @4.667s +3.812s
    `-NetworkManager.service @4.642s +21ms
      `-dbus.service @4.628s
        `-basic.target @4.620s
          `-sockets.target @4.620s
            ...

$ systemd-analyze plot > /tmp/boot.svg
$ xdg-open /tmp/boot.svg     # opens an SVG with a horizontal timeline of every unit
```

Other tricks:

```bash
# Dump the dependency graph to dot format
$ systemd-analyze dot --to-pattern='multi-user.target' | dot -Tsvg > graph.svg

# Lint a unit file for common mistakes
$ systemd-analyze verify /etc/systemd/system/hello.service

# Unfold security-related directives for a unit
$ systemd-analyze security nginx.service
  → Overall exposure level for nginx.service: 6.6 MEDIUM 
  ... (shows every hardening option, whether it is set, and an exposure score)

# Dump the unit cgroup tree
$ systemd-cgls

# Show systemd's own unit-load timing
$ systemd-analyze time
```

Use `blame` to see which units are slow. Use `critical-chain` to see the longest path through the dependency graph (the actual bottleneck). Use `plot` for a beautiful boot waterfall. Use `security` to audit your services against systemd's hardening menu.

## systemd-cgtop (Live cgroup view)

`systemd-cgtop` is `top` for cgroups. It lets you see CPU, memory, and IO grouped by service.

```bash
$ systemd-cgtop -d 1 -n 5
```

`-d 1` means update every second. `-n 5` means run five iterations and exit. Without `-n`, it runs forever like `top`.

You can sort by `%CPU` (default), memory, IO, or task count using `-c`/`-m`/`-i`/`-t`.

This is the fastest way to figure out "which service is eating my CPU?" or "which service is leaking memory?" because the unit name shows up directly.

## loginctl, machinectl, hostnamectl, timedatectl

systemd ships a family of `*ctl` tools for managing different parts of the system.

### loginctl

Manages login sessions and seats. `systemd-logind` is the daemon behind logins on modern desktops.

```bash
$ loginctl                              # list active sessions
   SESSION  UID USER     SEAT  TTY
         3 1000 alice    seat0 tty2
         5 1000 alice          pts/0

$ loginctl session-status 3
3 - alice (1000)
   Since: Mon 2026-04-27 09:00:01 UTC; 1h ago
   ...

$ loginctl list-users
$ loginctl terminate-session 5         # kick session 5
$ loginctl lock-session 3              # lock the desktop
$ loginctl enable-linger alice         # let alice's user services run when she's logged out
```

### machinectl

Manages containers and virtual machines registered with systemd-machined.

```bash
$ machinectl list
MACHINE      CLASS     SERVICE        OS     VERSION
mycontainer  container systemd-nspawn debian 12

$ machinectl shell mycontainer        # open a shell inside
$ machinectl start mycontainer
$ machinectl poweroff mycontainer
```

### hostnamectl

Sets the system hostname.

```bash
$ hostnamectl
   Static hostname: box
        Icon name: computer-desktop
       Machine ID: 1234abcd...
          Boot ID: 5678efgh...
   Operating System: Debian GNU/Linux 12 (bookworm)
            Kernel: Linux 6.1.0
      Architecture: x86-64

$ sudo hostnamectl set-hostname new-name
$ sudo hostnamectl set-icon-name computer-laptop
```

### timedatectl

Sets the timezone, syncs to NTP, and shows time info.

```bash
$ timedatectl
               Local time: Mon 2026-04-27 11:05:32 UTC
           Universal time: Mon 2026-04-27 11:05:32 UTC
                 RTC time: Mon 2026-04-27 11:05:32
                Time zone: UTC (UTC, +0000)
System clock synchronized: yes
              NTP service: active

$ sudo timedatectl set-timezone America/Denver
$ sudo timedatectl set-ntp true
$ sudo timedatectl set-time '2026-04-27 12:00:00'
$ timedatectl list-timezones | grep -i denver
$ timedatectl show-timesync --all
```

`localectl` (locale and keymap) and `resolvectl` (DNS resolution via systemd-resolved) round out the family.

## systemd-tmpfiles, systemd-sysusers

Two small, important tools you will eventually need.

### systemd-tmpfiles

Creates, cleans, and removes files and directories on a schedule. The rules live in `/etc/tmpfiles.d/` and `/usr/lib/tmpfiles.d/`. They look like:

```
# type path                mode user group age argument
d /run/myapp               0755 myapp myapp -
d /var/lib/myapp           0755 myapp myapp -
f /var/log/myapp/log       0644 myapp myapp -
L /var/lib/myapp/current   -    -     -     -    /var/lib/myapp/v3
e /tmp/myapp-cache         -    -     -     1d
```

Type letters: `d` = directory, `f` = file, `L` = symlink, `e` = age-out, `r` = remove on boot, `R` = recursive remove, `Z` = recursive perms. There are about thirty types.

Run rules manually:

```bash
$ sudo systemd-tmpfiles --create
$ sudo systemd-tmpfiles --clean
$ sudo systemd-tmpfiles --remove
```

systemd runs `--create` early at boot, then `--clean` periodically via `systemd-tmpfiles-clean.timer`.

### systemd-sysusers

Creates system users and groups declaratively. Rules in `/etc/sysusers.d/` look like:

```
# type name id gecos home shell
u myapp -   "My App service" /var/lib/myapp /usr/sbin/nologin
g mygrp -
m myapp mygrp
```

Run:

```bash
$ sudo systemd-sysusers
```

Replaces the dance of `useradd` / `groupadd` / `usermod` for system accounts and makes user creation idempotent and packageable.

## User Services

Almost everything we have done so far is for the **system** instance of systemd, which runs as PID 1. But there is a second flavor: a **user** instance of systemd, one per logged-in user. It manages services that belong to that user, independent of root.

User units live in:

- `~/.config/systemd/user/` — your own.
- `/etc/systemd/user/` — admin overrides.
- `/usr/lib/systemd/user/` — distro-shipped.

You manage them with the `--user` flag:

```bash
$ systemctl --user status
$ systemctl --user list-units
$ systemctl --user start myapp.service
$ systemctl --user enable myapp.service
$ journalctl --user -u myapp.service
```

A simple user service: put this in `~/.config/systemd/user/myapp.service`:

```ini
[Unit]
Description=My personal app

[Service]
Type=simple
ExecStart=%h/bin/myapp

[Install]
WantedBy=default.target
```

Note `WantedBy=default.target` (not `multi-user.target`) — for user units, `default.target` is the user's own default target.

`%h` is a specifier that expands to your home directory at unit-load time. There are many specifiers: `%u` (username), `%U` (uid), `%h` (home), `%i` (instance name for templated units), `%j` (final component of the unit name), `%t` (runtime directory, like `/run/user/1000`), `%T` (system tmp), `%H` (hostname), `%p` (prefix). The full list is in `man systemd.unit`.

User services normally die when you log out. To keep them running when you are logged out, enable **lingering** for your user:

```bash
$ sudo loginctl enable-linger $USER
```

After that, your user systemd instance starts at boot and keeps your enabled user services alive across logins/logouts.

## Drop-Ins

You bought a server. The distro shipped `nginx.service` in `/usr/lib/systemd/system/nginx.service`. You want to change one line. You do **not** edit the distro file directly — package upgrades will overwrite your changes.

Instead, you create a drop-in. Drop-ins live in a directory named after the unit, with `.d` appended:

```
/etc/systemd/system/nginx.service.d/override.conf
```

You can have as many `.conf` files in there as you want. systemd merges them all on top of the original unit.

The fastest way to create a drop-in is `systemctl edit`:

```bash
$ sudo systemctl edit nginx.service
```

systemd opens an editor on a blank file at `/etc/systemd/system/nginx.service.d/override.conf`. You write the override:

```ini
[Service]
MemoryMax=2G
Environment=NGINX_WORKER_CONNECTIONS=10000
```

Save and exit. systemd auto-runs `daemon-reload` for you. Now your override is in place.

To replace a list option (like `ExecStart=`), set it to empty first to clear the inherited value:

```ini
[Service]
ExecStart=
ExecStart=/usr/local/bin/my-nginx -g 'daemon off;'
```

Without the empty `ExecStart=`, your line would just be **added** to the inherited one and systemd would refuse to start the unit because two ExecStart= entries are not allowed for `Type=simple`.

To see the merged effective unit:

```bash
$ systemctl cat nginx.service
```

To see only the drop-in:

```bash
$ ls /etc/systemd/system/nginx.service.d/
$ cat /etc/systemd/system/nginx.service.d/override.conf
```

To wipe the override:

```bash
$ sudo systemctl revert nginx.service
```

### Drop-in merge order

```
+--------------------------------------------+
|  /usr/lib/systemd/system/nginx.service     |  <- base (vendor)
+--------------------------------------------+
                  ^
                  | overlay
                  |
+--------------------------------------------+
|  /run/systemd/system/nginx.service.d/*.conf |  <- runtime drop-ins
+--------------------------------------------+
                  ^
                  | overlay
                  |
+--------------------------------------------+
|  /etc/systemd/system/nginx.service.d/*.conf |  <- admin drop-ins (highest)
+--------------------------------------------+
```

Each layer can also be a full replacement file (with the same name) instead of a drop-in directory; in that case the lower file is shadowed entirely.

## DynamicUser=

A modern systemd service does not need to come with a fixed user account. If you set `DynamicUser=true`, systemd allocates a user and group on the fly when the service starts and frees them when it stops. The names look like `myservice` and the UID is in a high range that does not collide with anything.

```ini
[Service]
DynamicUser=true
StateDirectory=myservice
LogsDirectory=myservice
RuntimeDirectory=myservice
```

`DynamicUser=true` was added in systemd 232 (2016). It works hand-in-hand with `StateDirectory=`, `LogsDirectory=`, `RuntimeDirectory=`, `CacheDirectory=`, and `ConfigurationDirectory=`, which create directories under `/var/lib/`, `/var/log/`, `/run/`, `/var/cache/`, `/etc/` respectively, owned by the dynamic user.

The dynamic user is also `PrivateTmp=true`, `RemoveIPC=true`, `ProtectSystem=strict`, `ProtectHome=true` by default. So you get a tightly sandboxed account for free.

## Security Hardening Menu

systemd has a long list of `Protect*=`, `Restrict*=`, `Private*=` knobs. Each of them flips on a kernel feature (mount namespaces, seccomp filters, capabilities) to limit what your service can do. Setting them is cheap — it just goes into the unit file. Auditing them is cheap — `systemd-analyze security` scores you.

The greatest hits:

- `NoNewPrivileges=true` — service cannot acquire new privileges (no setuid binaries can re-elevate). **Always set this.**
- `PrivateTmp=true` — gives the service its own `/tmp` and `/var/tmp`, mount-namespaced.
- `PrivateDevices=true` — only `/dev/null`, `/dev/zero`, `/dev/random` etc.; no real device nodes.
- `PrivateNetwork=true` — service runs in its own network namespace with only loopback.
- `PrivateUsers=true` — runs in a user namespace where everyone is mapped to nobody outside.
- `PrivateMounts=true` — independent mount namespace.
- `PrivateIPC=true` — independent SysV IPC namespace.
- `ProtectSystem=strict` — `/usr`, `/boot`, `/efi`, `/etc` are read-only. `strict` is the toughest setting; `full` and `yes` and `false` are the lighter ones.
- `ProtectHome=true` — `/home`, `/root`, `/run/user` are inaccessible. Or `read-only`. Or `tmpfs`.
- `ProtectKernelTunables=true` — `/proc/sys`, `/sys/fs`, `/sys/kernel` are read-only.
- `ProtectKernelModules=true` — service cannot load kernel modules.
- `ProtectKernelLogs=true` — service cannot read the kernel log.
- `ProtectClock=true` — service cannot change the system clock.
- `ProtectControlGroups=true` — `/sys/fs/cgroup` is read-only.
- `ProtectHostname=true` — service cannot change the hostname.
- `ProtectProc=invisible` — `/proc` only shows the service's own processes.
- `ProcSubset=pid` — `/proc` only exposes process directories, not kernel files.
- `RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6` — only allow these socket families.
- `RestrictNamespaces=true` — service cannot create new namespaces.
- `RestrictRealtime=true` — service cannot set realtime priority.
- `RestrictSUIDSGID=true` — service cannot create setuid/setgid binaries.
- `LockPersonality=true` — service cannot change `personality(2)`.
- `MemoryDenyWriteExecute=true` — service cannot map memory writable+executable. Defeats classic shellcode injection.
- `SystemCallFilter=@system-service` — seccomp filter; only allow this group of syscalls. There are pre-defined groups: `@basic-io`, `@file-system`, `@network-io`, `@process`, etc.
- `SystemCallArchitectures=native` — disallow non-native syscall ABIs.
- `CapabilityBoundingSet=` — list of capabilities allowed; everything else is dropped.
- `AmbientCapabilities=` — capabilities granted to the executed binary even if not setuid.

A modern hardened service might look like:

```ini
[Service]
Type=simple
DynamicUser=true
StateDirectory=myapp
ExecStart=/usr/local/bin/myapp

NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
ProtectHostname=true
ProtectClock=true
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6
RestrictNamespaces=true
RestrictRealtime=true
RestrictSUIDSGID=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallArchitectures=native
CapabilityBoundingSet=
AmbientCapabilities=
```

Run `systemd-analyze security myapp.service` and watch the score drop from "9.5 EXPOSED" to "1.0 PROTECTED."

## Common Errors

The exact strings you will see, plus what they mean and how to fix them.

### `Failed to start X.service: Unit X.service not found.`

systemd cannot find a unit file with that name. Either the file does not exist, you misspelled it, or you forgot `daemon-reload` after creating it.

```bash
$ sudo systemctl daemon-reload
$ ls /etc/systemd/system/X.service /usr/lib/systemd/system/X.service 2>/dev/null
```

### `Failed to start X.service: A dependency job for X.service failed. See 'journalctl -xe' for details.`

A unit X requires (or is wanted-by) failed. Look at the chain:

```bash
$ systemctl list-dependencies X.service --before
$ journalctl -xe -u X.service
```

### `Failed to enable unit: Unit file /etc/systemd/system/X.service does not exist.`

`systemctl enable` was called but the file is missing or `[Install]` is missing.

### `Failed to start X.service: Operation refused, unit X.service may be requested by dependency only.`

The unit has `RefuseManualStart=yes` (often on `.target` units that are only meant to be reached as a side effect). Use `systemctl isolate` instead, or activate one of the units that depends on it.

### `Job for X.service failed because the control process exited with error code.`

`ExecStart=` ran, the process exited with a non-zero exit code, and `Restart=` did not catch it. Look in the journal:

```bash
$ journalctl -u X.service -n 50 --no-pager
```

You will usually find the actual error from the binary itself a few lines up.

### `X.service: Failed with result 'timeout'.`

The service did not signal ready within `TimeoutStartSec=` (default 90 seconds). Either the service is slow, or the `Type=` is wrong (e.g. `Type=notify` but the program never calls `sd_notify`). Increase the timeout, or fix the type.

### `/usr/bin/X: Permission denied`

The binary is not executable, or `User=`/`Group=` don't have execute permission, or it's on a filesystem mounted `noexec`.

```bash
$ ls -l /usr/bin/X
$ stat -c '%a %U:%G' /usr/bin/X
```

### `X.service: main process exited, code=exited, status=1/FAILURE`

The program crashed with exit code 1. systemd is just reporting it. The actual reason is in the journal lines just before this one. `status=1` is generic "the program said something went wrong"; check its own logs.

### `activation timed out`

A `Type=notify` service did not call `sd_notify(READY=1)` within `TimeoutStartSec=`. Either the program is broken, the program is not linked against libsystemd, or `NotifyAccess=` is set wrong.

### `Killing process N (X) with signal SIGKILL.`

systemd waited `TimeoutStopSec=` for graceful shutdown after sending SIGTERM, did not get it, and sent SIGKILL. Either the program ignores SIGTERM, or the timeout is too short.

### `PID file /run/X.pid does not exist after start`

`Type=forking` plus `PIDFile=` was set, but the daemon never wrote the PID file (or wrote it to a different path). Check `RuntimeDirectory=`, the daemon's own pidfile config, and try `Type=simple` if the daemon supports running in foreground.

### `Active: failed (Result: signal)`

The main process was killed by a signal (e.g. SIGSEGV from a real crash, or SIGKILL because the OOM killer fired, or SIGTERM from a user). Check the journal and `dmesg` to see which signal and why.

### `Active: activating (auto-restart)`

systemd is between restarts. Either the start hasn't taken yet, or `RestartSec=` is counting down. If it sticks here forever, your service is in a restart loop and you are about to hit `StartLimitBurst`.

### `Failed to load environment files: /etc/X/env: No such file or directory`

`EnvironmentFile=/etc/X/env` was set but the file does not exist. Either remove the directive, prefix it with `-` (`EnvironmentFile=-/etc/X/env`) to make it optional, or actually create the file.

### `status=203/EXEC`

The kernel could not run the binary. Causes: file does not exist, wrong arch, wrong interpreter (`#!/bin/wrong`), missing executable bit, `noexec` mount.

### `status=200/CHDIR`

The kernel could not change to `WorkingDirectory=`. Either the directory does not exist or the user does not have access.

## Hands-On

Time to type. Open a terminal. Each block is a real command you can run on a Linux system with systemd. Some need `sudo`.

```bash
# 1. Show the status of a service
$ systemctl status sshd

# 2. Start a service
$ sudo systemctl start sshd

# 3. Stop a service
$ sudo systemctl stop sshd

# 4. Restart a service
$ sudo systemctl restart sshd

# 5. Reload a service's config (if it supports it)
$ sudo systemctl reload sshd

# 6. Enable at boot
$ sudo systemctl enable sshd

# 7. Disable at boot
$ sudo systemctl disable sshd

# 8. Enable and start in one step
$ sudo systemctl enable --now sshd

# 9. Disable and stop in one step
$ sudo systemctl disable --now sshd

# 10. List all currently loaded units
$ systemctl list-units

# 11. List all unit files installed (loaded or not)
$ systemctl list-unit-files

# 12. List active jobs (start/stop in progress)
$ systemctl list-jobs

# 13. Check whether a unit is active right now
$ systemctl is-active sshd

# 14. Check whether a unit will start at boot
$ systemctl is-enabled sshd

# 15. Check whether a unit is in the failed state
$ systemctl is-failed sshd

# 16. Clear a failed state so systemd will retry
$ sudo systemctl reset-failed sshd

# 17. Re-read every unit file from disk
$ sudo systemctl daemon-reload

# 18. Edit a vendor unit safely (creates an override drop-in)
$ sudo systemctl edit sshd

# 19. Show the merged effective unit file
$ systemctl cat sshd

# 20. Show every property of a unit
$ systemctl show sshd | head -40

# 21. List only failed units
$ systemctl --failed

# 22. List dependencies of a unit (what pulls in what)
$ systemctl list-dependencies multi-user.target

# 23. List reverse dependencies (who pulls in this unit)
$ systemctl list-dependencies sshd --reverse

# 24. Mask a unit so nothing can start it
$ sudo systemctl mask telnet.service

# 25. Unmask
$ sudo systemctl unmask telnet.service

# 26. Tail the journal live
$ journalctl -f

# 27. Logs for one unit since today
$ journalctl -u sshd --since=today

# 28. Logs from this boot only
$ journalctl --boot

# 29. Errors and worse
$ journalctl -p err --since=today

# 30. Logs by PID
$ journalctl _PID=1

# 31. Logs by UID
$ journalctl _UID=0 --since=today | head -20

# 32. Logs by binary name
$ journalctl _COMM=sshd --since=today

# 33. Trim the journal to the last 500 MB
$ sudo journalctl --vacuum-size=500M

# 34. Trim the journal to the last two weeks
$ sudo journalctl --vacuum-time=2weeks

# 35. Boot timing summary
$ systemd-analyze

# 36. Slowest services to start
$ systemd-analyze blame | head -10

# 37. Critical chain (the actual bottleneck path)
$ systemd-analyze critical-chain

# 38. SVG of the boot timeline
$ systemd-analyze plot > /tmp/boot.svg && xdg-open /tmp/boot.svg

# 39. Live cgroup view
$ systemd-cgtop

# 40. Login session listing
$ loginctl

# 41. Detailed status of one session
$ loginctl session-status 1

# 42. Containers / VMs registered with systemd
$ machinectl list

# 43. System hostname info
$ hostnamectl

# 44. Time and timezone
$ timedatectl

# 45. Set the timezone
$ sudo timedatectl set-timezone America/Denver

# 46. List system D-Bus services
$ busctl list | head -20

# 47. Tree of all D-Bus objects
$ busctl tree org.freedesktop.systemd1 | head -20

# 48. Send a one-shot message to D-Bus
$ dbus-send --system --print-reply --dest=org.freedesktop.systemd1 \
    /org/freedesktop/systemd1 org.freedesktop.systemd1.Manager.GetUnit string:sshd.service

# 49. Legacy runlevel command (still works)
$ runlevel

# 50. Legacy telinit (still works)
$ sudo telinit 3

# 51. Switch to rescue mode (single-user)
$ sudo systemctl rescue

# 52. Switch to emergency mode (even more minimal)
$ sudo systemctl emergency

# 53. Reach a target without changing the default
$ sudo systemctl isolate multi-user.target

# 54. Set the default target permanently
$ sudo systemctl set-default multi-user.target

# 55. Power off
$ sudo systemctl poweroff      # or:  shutdown -h now / poweroff

# 56. Reboot
$ sudo systemctl reboot        # or:  shutdown -r now / reboot

# 57. Halt without powering off
$ sudo systemctl halt

# 58. Schedule a shutdown
$ sudo shutdown -h +5 "Going down for maintenance"

# 59. Cancel a scheduled shutdown
$ sudo shutdown -c

# 60. Run a one-shot command tagged in the journal
$ systemd-cat -t myscript -p info echo "this lands in the journal"

# 61. Run a transient service from the command line
$ systemd-run --unit=demo.service --on-active=10s /usr/bin/touch /tmp/hi
$ systemd-run --user --unit=demo.service /usr/local/bin/myapp

# 62. Run a transient timer
$ systemd-run --on-calendar='*-*-* 03:00' /usr/local/bin/nightly

# 63. Run a command in its own scope (for ad-hoc cgroup limits)
$ systemd-run --scope -p MemoryMax=500M /usr/bin/heavy-job

# 64. Lint a unit
$ systemd-analyze verify /etc/systemd/system/hello.service

# 65. Audit hardening of a unit
$ systemd-analyze security sshd

# 66. Watch boot in real time on next reboot
$ sudo systemctl reboot
# (then look at the console; or after boot, run `systemd-analyze`)

# 67. Show all timer units and when they fire next
$ systemctl list-timers --all

# 68. Show all socket units and what they listen on
$ systemctl list-sockets --all

# 69. Show the cgroup tree
$ systemd-cgls

# 70. User-mode systemd: status of your services
$ systemctl --user status
```

Seventy commands. The first thirty are the daily ones. Memorize those. The rest fill in over time.

## Common Confusions

These are the ones that bite people on day three.

### 1. `enable` vs `start`

`enable` adds the unit to a target's wants list (creates a symlink in `*.target.wants/`) so it runs at next boot. `start` activates it right now. They are independent. You can enable without starting (it'll run next boot), start without enabling (it runs now but not after reboot), do both with `enable --now`, or neither with `disable --now`.

### 2. Forgetting `daemon-reload` after editing

systemd does not watch unit files for changes. After you create or edit `/etc/systemd/system/foo.service`, you must run `sudo systemctl daemon-reload` before `systemctl start foo.service` will see your changes. `systemctl edit` and `systemctl revert` do this for you. Hand-edits do not.

### 3. What is a target *really*

A target is a sentinel unit. It does not run any process. It just exists so other units can say `WantedBy=target` and humans can say "isolate to that." When you "reach" a target, systemd has finished starting everything that wants it.

### 4. Why do I need `WantedBy=multi-user.target`?

The `[Install]` section is read **only** when you run `systemctl enable`. `WantedBy=multi-user.target` tells systemctl: "when I am enabled, create a symlink in `multi-user.target.wants/` pointing to me, so that booting `multi-user.target` pulls me in." Without `[Install]`, `enable` has nothing to do and prints "no installation config."

### 5. `Requires=` vs `Wants=`

`Requires=A` is a hard dependency: if A fails to start (or stops), I also fail. `Wants=A` is a soft dependency: pull A in but I'm fine if A is broken. Almost everybody overuses `Requires=` and ends up with cascading failures. Default to `Wants=` and reach for `Requires=` only when you really mean it.

### 6. `Type=simple` vs `Type=forking`

`simple` (the default) means your binary runs in the foreground; systemd considers it started the instant it forks. `forking` means your binary daemonizes itself by double-forking; systemd waits for the parent to exit and treats the surviving child as the main process. Modern programs should run in the foreground and be `simple` (or `notify`). Old programs that double-fork were what `forking` was added for. Mismatch = restart loops or "service started but nothing happened."

### 7. How do restart loops happen?

`Restart=always` plus a binary that immediately crashes plus `RestartSec=0` = systemd starts it, it dies, systemd restarts it, it dies, forever. systemd defends with `StartLimitBurst=` (default 5) and `StartLimitIntervalSec=` (default 10s): if a unit fails too fast too many times, it goes to `failed` and stops restarting. You then need to `reset-failed` after fixing the bug. Tune `RestartSec=` and `StartLimitBurst=` to tolerate flaky-but-recovering services without infinite churn.

### 8. What does `Type=notify` need?

`Type=notify` requires the program to call `sd_notify(0, "READY=1")` (or write `READY=1` to the socket whose path is in `$NOTIFY_SOCKET`). Without that call, systemd waits forever and eventually times out. If your code does not have an `sd_notify` call, do not pick `Type=notify`.

### 9. Why is my service stuck in `auto-restart`?

It is between restarts. Either `RestartSec=` is counting down, or it crashed and systemd is about to retry. If it stays here, the binary is failing very fast. Check logs.

### 10. User services vs system services

System services run as root by default and live in `/etc/systemd/system/` etc. User services run as your user, live in `~/.config/systemd/user/`, and are managed with `systemctl --user`. They are in completely separate worlds. `systemctl enable foo.service` does not enable user services. You must use `systemctl --user enable foo.service`.

### 11. How does socket activation actually start the daemon?

The `.socket` unit is enabled and started at boot. systemd binds the listening port. The matching `.service` is **not** running. The first time a client connects, the kernel notifies systemd. systemd starts the `.service` and passes the socket file descriptor via standard file descriptor inheritance (FDs 3 and up, with environment variables `LISTEN_FDS` and `LISTEN_PID` set). The service uses `sd_listen_fds()` to grab the inherited socket and starts serving. systemd holds onto the original socket too, so it can re-pass it to a fresh service after a restart.

### 12. `Restart=on-failure` vs `Restart=always`

`on-failure` restarts only on crashes, non-zero exits, kills, watchdog timeouts. `always` restarts no matter what, including clean exits. If your service is a daemon that should never voluntarily exit, `always` is fine. If your service might cleanly exit when it has done its job (and you want it to stay exited), use `on-failure`.

### 13. What does `StandardOutput=journal` mean?

It tells systemd to capture the service's stdout and write each line to the journal as a record tagged with the unit name. `journal` is the default. Other values: `inherit`, `null`, `tty`, `kmsg`, `file:/path`, `append:/path`, `truncate:/path`, `socket`, `fd:name`. `StandardError=` works the same way.

### 14. How do I override only one line of a vendor unit?

Use a drop-in. `sudo systemctl edit foo.service` creates `/etc/systemd/system/foo.service.d/override.conf`. Put the section header and only the line you want to change. For list options like `ExecStart=`, write `ExecStart=` (empty) first to clear the inherited value, then your replacement. systemd merges the drop-in on top of the original; the original is untouched and will not be overwritten by package upgrades.

### 15. `systemctl reload` vs `systemctl restart`

`reload` runs the unit's `ExecReload=` (often `kill -HUP $MAINPID`), which tells the daemon to re-read its config without dropping connections. `restart` stops and starts the service, dropping connections. Always prefer `reload` when the daemon supports it.

### 16. `mask` vs `disable`

`disable` removes the symlink in `*.target.wants/` so the unit won't start at boot but you can still start it manually. `mask` symlinks the unit to `/dev/null` so **nothing** can start it (manual, dependency, anything). Mask is the strongest "do not run." Unmask with `systemctl unmask`.

### 17. Where do logs go for `Type=oneshot` after the process exits?

To the journal, exactly like every other service. Read with `journalctl -u foo.service`. Even short-lived oneshots leave a complete record. systemd captures everything.

## Vocabulary

| Word | Plain English |
|------|---------------|
| systemd | The conductor/init system that runs as PID 1 on most modern Linux distros. |
| PID 1 | The very first user-space process. Parent of everything else. If it dies, kernel panics. |
| init | Generic term for "the program that runs as PID 1." Used to be sysvinit; now usually systemd. |
| sysvinit (legacy) | Old SysV-style init that ran `/etc/rc` scripts. Largely replaced by systemd. |
| upstart (legacy) | An older Ubuntu init system. Replaced by systemd around 2014. |
| runit | An alternative init used by Void Linux. Tiny, scriptless. Not systemd. |
| openrc | Gentoo's init system. Compatible with sysvinit scripts. Not systemd. |
| s6 | Another tiny init/supervisor used in some distros. Not systemd. |
| unit | The basic object systemd manages. A service, socket, timer, etc. |
| .service | A unit type for processes (daemons, scripts). |
| .socket | A unit type for listening sockets that activate services on demand. |
| .timer | A unit type for scheduled triggers. Replaces cron. |
| .target | A unit type that names a goal; groups other units. Replaces SysV runlevels. |
| .mount | A unit type for filesystem mount points. |
| .automount | A unit type for lazy mounting on first access. |
| .path | A unit type that watches a path and fires a service on change. |
| .device | A unit type representing a kernel device (auto-generated by udev). |
| .swap | A unit type for swap partitions or files. |
| .slice | A unit type representing a node in the cgroup tree for resource control. |
| .scope | A unit type for processes started outside systemd but tracked by it. |
| .nspawn | Config for systemd-nspawn containers. |
| .link | systemd-udev config for naming and configuring network links. |
| .netdev | systemd-networkd config for creating virtual network devices. |
| .network | systemd-networkd config for configuring network interfaces. |
| `[Unit]` | The first section of a unit file: description, dependencies, ordering. |
| `[Service]` | The service-specific section: how to run the process. |
| `[Install]` | The section read by `systemctl enable`/`disable`. |
| Type=simple | Default service type. Considered started as soon as the process is forked. |
| Type=exec | Like simple but waits for execve(). |
| Type=forking | For programs that double-fork; systemd waits for the parent to exit. |
| Type=oneshot | Short-lived script. Considered done when the process exits. |
| Type=notify | Program calls sd_notify(READY=1) when ready; systemd waits for that signal. |
| Type=dbus | Program registers a D-Bus name; systemd waits for the name. |
| Type=idle | Waits for other startup jobs to finish. Used for getty. |
| ExecStart= | The command that starts the service. |
| ExecStartPre= | Runs before ExecStart. |
| ExecStartPost= | Runs after ExecStart succeeds. |
| ExecStop= | Runs to stop the service gracefully. |
| ExecStopPost= | Runs after the process has stopped. |
| ExecReload= | Runs on `systemctl reload`. |
| ExecCondition= | Runs first; if non-zero, service is treated as not-needed. |
| Restart= | When to restart: no, on-success, on-failure, on-abnormal, on-abort, on-watchdog, always. |
| RestartSec= | How long to wait before restarting after exit. |
| RestartSteps= | Stepped backoff between restarts. |
| StartLimitBurst= | Max restarts within the interval before unit is marked failed. |
| StartLimitIntervalSec= | Window over which StartLimitBurst counts. |
| RemainAfterExit= | For oneshot units; treat the unit as still active after the process exits. |
| TimeoutStartSec= | How long to wait for start to succeed (default 90s). |
| TimeoutStopSec= | How long to wait for stop before SIGKILL (default 90s). |
| TimeoutAbortSec= | Like TimeoutStopSec but for abort path. |
| RuntimeMaxSec= | Hard ceiling on how long the service can run. |
| KillSignal= | Signal sent on stop (default SIGTERM). |
| KillMode= | control-group (kill cgroup), process (kill main), mixed (TERM main, KILL cgroup), none. |
| SendSIGKILL= | Whether to escalate to SIGKILL on timeout (default yes). |
| FinalKillSignal= | Last-ditch signal if SIGKILL doesn't reap. |
| WorkingDirectory= | The cwd of the process. |
| RootDirectory= | chroot directory. |
| RootImage= | Mount this disk image as root for the service. |
| BindPaths= | Mount-bind paths into the service's namespace. |
| BindReadOnlyPaths= | Read-only bind mounts. |
| ReadOnlyPaths= | Mark these paths read-only inside the service. |
| ReadWritePaths= | Override ProtectSystem= for these paths. |
| InaccessiblePaths= | Hide these paths from the service. |
| TemporaryFileSystem= | Mount a tmpfs at this path inside the service's namespace. |
| PrivateTmp= | Give the service its own /tmp. |
| PrivateDevices= | Restrict /dev to a safe minimum. |
| PrivateNetwork= | Run in own network namespace, only loopback. |
| PrivateUsers= | Run in user namespace; outside, all UIDs map to nobody. |
| PrivateMounts= | Independent mount namespace. |
| PrivateIPC= | Independent SysV IPC namespace. |
| ProtectSystem= | yes/full/strict; mark /usr, /boot, etc. read-only. |
| ProtectHome= | yes/read-only/tmpfs; hide /home. |
| ProtectKernelTunables= | Read-only /proc/sys etc. |
| ProtectKernelModules= | Block module loading. |
| ProtectKernelLogs= | Hide kernel log. |
| ProtectClock= | Block clock changes. |
| ProtectControlGroups= | Read-only /sys/fs/cgroup. |
| ProtectHostname= | Block hostname changes. |
| ProtectProc= | invisible/ptraceable/default; restrict /proc visibility. |
| ProcSubset= | pid/all; restrict /proc to process dirs. |
| NoNewPrivileges= | Set NO_NEW_PRIVS bit; disables setuid escalation. |
| RestrictAddressFamilies= | Limit which socket families the service can use. |
| RestrictNamespaces= | Block namespace creation. |
| RestrictRealtime= | Block realtime scheduling. |
| RestrictSUIDSGID= | Block creation of setuid/setgid binaries. |
| LockPersonality= | Pin the process personality. |
| MemoryDenyWriteExecute= | Block writable+executable memory mappings. |
| SystemCallFilter= | Seccomp allow/deny list of syscalls. |
| SystemCallArchitectures= | Allowed syscall ABIs. |
| SystemCallErrorNumber= | Errno returned to blocked syscalls. |
| CapabilityBoundingSet= | Capabilities the service is allowed. |
| AmbientCapabilities= | Capabilities granted to the executed binary. |
| User= | UID under which the service runs. |
| Group= | Primary GID under which the service runs. |
| SupplementaryGroups= | Extra GIDs. |
| DynamicUser= | Allocate a transient user/UID for the service. |
| UMask= | File-creation umask. |
| Environment= | Set environment variables for the service. |
| EnvironmentFile= | Read environment variables from a file. |
| PassEnvironment= | Forward listed env vars from manager to service. |
| UnsetEnvironment= | Clear listed env vars from service. |
| StandardInput= | Where stdin comes from: null, tty, file:, socket, fd:. |
| StandardOutput= | Where stdout goes: journal (default), inherit, null, tty, kmsg, file:, append:, truncate:, socket, fd:. |
| StandardError= | Same as StandardOutput= but for stderr. |
| journal | The main log destination managed by systemd-journald. |
| syslog | A legacy log protocol; journald can forward to it. |
| kmsg | The kernel ring buffer (read with dmesg). |
| file: | Send output to a file (overwrites or creates). |
| append: | Send output to a file in append mode. |
| truncate: | Send output to a file, truncating each start. |
| socket | Connect stdio to the activation socket. |
| fd: | Connect to a named file descriptor passed via SocketActivation. |
| NotifyAccess= | main/all/exec/none; who is allowed to call sd_notify. |
| WatchdogSec= | Service must call sd_notify(WATCHDOG=1) within this interval or it's killed. |
| IOWeight= | Block-IO bandwidth weight (cgroup IO controller). |
| IOReadBandwidthMax= | Cap read bandwidth for a device. |
| IOWriteBandwidthMax= | Cap write bandwidth for a device. |
| MemoryHigh= | Soft memory throttling threshold. |
| MemoryMax= | Hard memory limit. |
| MemorySwapMax= | Cap on swap usage by this cgroup. |
| CPUWeight= | Relative CPU share. |
| CPUQuota= | Absolute CPU cap as percentage. |
| TasksMax= | Max processes/threads in the cgroup. |
| IPAddressAllow= | BPF-based allow-list of IP ranges. |
| IPAddressDeny= | BPF-based deny-list of IP ranges. |
| IPIngressFilterPath= | Custom BPF filter for ingress. |
| IPEgressFilterPath= | Custom BPF filter for egress. |
| NetworkNamespacePath= | Bind the service to a specific network namespace. |
| ListenStream= | TCP / Unix-stream listening socket. |
| ListenDatagram= | UDP / Unix-dgram socket. |
| ListenSequentialPacket= | SEQPACKET socket. |
| ListenFIFO= | Named pipe. |
| ListenSpecial= | Character device. |
| ListenNetlink= | Kernel netlink socket. |
| ListenMessageQueue= | POSIX message queue. |
| BindIPv6Only= | Whether the socket is v6-only or dual-stack. |
| Backlog= | Listen backlog size. |
| BindToDevice= | Bind socket to a specific NIC. |
| SocketUser= | Owner of the socket file. |
| SocketGroup= | Group of the socket file. |
| SocketMode= | Permissions of the socket file. |
| OnCalendar= | Realtime timer expression. |
| OnBootSec= | Monotonic timer offset from boot. |
| OnUnitActiveSec= | Timer offset from when the matched unit was last active. |
| OnUnitInactiveSec= | Timer offset from when the matched unit went inactive. |
| OnActiveSec= | Timer offset from when the timer was last activated. |
| OnStartupSec= | Offset from when systemd started. |
| AccuracySec= | Window over which systemd may batch timer fires. |
| RandomizedDelaySec= | Random extra delay added to the timer fire. |
| Persistent=true | For OnCalendar timers; catch up missed runs after reboot. |
| WakeSystem=true | Wake from suspend to fire the timer. |
| journald | The systemd log daemon (systemd-journald.service). |
| journalctl | CLI for reading the journal. |
| drop-in | A small override file in /etc/systemd/system/UNIT.d/*.conf. |
| override.conf | Conventional name for a drop-in file. |
| /etc/systemd/system | Admin-edited unit files (highest priority among on-disk locations). |
| /usr/lib/systemd/system | Distro/package-shipped unit files. |
| /run/systemd/system | Runtime-generated unit files (cleared on reboot). |
| ~/.config/systemd/user | A user's own user-mode unit files. |
| target unit | A unit of type .target — a named goal. |
| default.target | The unit systemd brings up at boot. Symlink to multi-user or graphical. |
| multi-user.target | Text-mode multi-user. Like SysV runlevel 3. |
| graphical.target | multi-user plus a display manager. Like SysV runlevel 5. |
| rescue.target | Single-user maintenance shell. |
| emergency.target | Minimal shell, read-only root. Most extreme rescue mode. |
| network.target | Network is configured. Does not mean reachable. |
| network-online.target | Network is configured and at least one route is usable. |
| sysinit.target | Very early init: mounts, modules, hostname. |
| basic.target | Sockets, timers, paths, slices ready. |
| sockets.target | All socket units active. |
| timers.target | All timer units active. |
| default.target.wants | Directory of symlinks pulled in when default.target is reached. |
| instance unit | A unit derived from a template, e.g. getty@tty1.service from getty@.service. |
| X@.service | A template unit; instances substitute %i with the part after the @. |
| %i | Specifier: instance string in a template unit. |
| %f | Specifier: unescaped filename (often path). |
| %u | Specifier: username (User= or current user). |
| %h | Specifier: home directory of User= or current user. |
| %H | Specifier: hostname. |
| %t | Specifier: runtime directory (/run for system, /run/user/UID for user). |
| %T | Specifier: $TMPDIR or /tmp. |
| %j | Specifier: final component of unit name. |
| %J | Specifier: unescaped final component. |
| %p | Specifier: prefix part of the unit name (before @). |
| %P | Specifier: unescaped prefix. |
| specifier expansion | systemd's macro substitution at unit-load time using % codes. |
| getty | A program that runs on a virtual console and presents a login prompt. |
| agetty | The GNU implementation of getty used on most distros. |
| console-getty | getty bound to /dev/console. |
| serial-getty | getty bound to a serial device, used for headless servers. |
| machinectl | CLI for managing systemd-nspawn containers and registered VMs. |
| nspawn | systemd's lightweight container runner (systemd-nspawn). |
| journalctl --boot | Filter journal to current (or selected) boot. |
| --identifier | journalctl flag: filter by SYSLOG_IDENTIFIER. |
| --pager-end | journalctl flag: jump straight to the end of the pager. |
| sd_notify | The libsystemd C function used by Type=notify services to signal readiness. |
| watchdog | Periodic "I am alive" signal from a service to systemd. |
| cgroup | Kernel control group. systemd assigns each service to its own cgroup. |
| cgroup v2 | Unified hierarchy of cgroups. Default in modern systemd (since v230). |
| systemctl | The main CLI for talking to systemd. |
| systemd-analyze | CLI for boot timing, dependency graph, security audit, calendar parsing. |
| systemd-cgtop | top-like live view of cgroup CPU/memory/IO. |
| systemd-cgls | Tree of all cgroups managed by systemd. |
| systemd-tmpfiles | Tool that creates/cleans/removes files according to declarative rules. |
| systemd-sysusers | Tool that creates system users/groups declaratively. |
| systemd-run | Run a command as a transient unit (service, scope, or timer). |
| systemd-cat | Pipe a command's output into the journal under a chosen tag. |
| systemd-escape | Convert a path or string into the systemd unit-name escaping form. |
| systemd-machined | Daemon that registers containers and VMs. |
| systemd-logind | Daemon that tracks user logins, seats, sessions. |
| systemd-resolved | A DNS resolver from the systemd project. |
| systemd-networkd | A network configurator from the systemd project. |
| systemd-homed | Per-user home directory daemon (systemd 245+). |
| systemd-portable | Portable services packaged as images (systemd 245+). |
| systemd-creds | Encrypted credential delivery (systemd 250+). |
| varlink | An IPC protocol systemd is moving toward (systemd 250+). |
| sysupdate | systemd-sysupdate, image-based system updates (systemd 256+). |
| daemon-reload | systemctl command to re-read all unit files from disk. |
| reset-failed | Clears the "failed" state of one or all units. |
| isolate | Activate a target and stop everything not pulled in by it. |
| mask | Symlink a unit to /dev/null so nothing can start it. |
| unmask | Reverse of mask. |
| linger | Setting (via loginctl enable-linger) that lets a user's services run when they're logged out. |
| seat | A physical workplace (one display, one keyboard) tracked by logind. |
| session | One user's login on one seat or remote pty. |
| scope | A cgroup of processes systemd tracks but did not start itself (e.g. user sessions). |
| transient unit | A unit created at runtime (e.g. via systemd-run) and gone after it exits. |

About 175 entries. More than 120.

## Try This

A small project to cement everything. Open a terminal.

1. Write a Python script `/usr/local/bin/heartbeat.py`:

   ```python
   #!/usr/bin/env python3
   import time, sys
   while True:
       print(f"heartbeat at {time.time()}", flush=True)
       time.sleep(2)
   ```

   `chmod +x /usr/local/bin/heartbeat.py`.

2. Write `/etc/systemd/system/heartbeat.service`:

   ```ini
   [Unit]
   Description=A heartbeat
   After=network.target

   [Service]
   Type=simple
   ExecStart=/usr/local/bin/heartbeat.py
   Restart=on-failure
   RestartSec=3
   DynamicUser=true
   NoNewPrivileges=true
   ProtectSystem=strict
   ProtectHome=true
   PrivateTmp=true

   [Install]
   WantedBy=multi-user.target
   ```

3. `sudo systemctl daemon-reload`
4. `sudo systemctl enable --now heartbeat.service`
5. `systemctl status heartbeat.service`
6. `journalctl -u heartbeat.service -f` (Ctrl+C to stop tailing)
7. `systemd-analyze security heartbeat.service`
8. Add a timer: `/etc/systemd/system/heartbeat-summary.service` (Type=oneshot, ExecStart=/bin/sh -c 'journalctl -u heartbeat.service --since="5 min ago" | wc -l') and a matching `.timer` (OnUnitActiveSec=5min, OnBootSec=1min). Enable it. Watch the journal for the periodic count.
9. Try breaking the heartbeat (`exit 1` at the top of the script). Watch `Restart=on-failure` in action. Then watch `StartLimitBurst` kick in if you make it fail in a tight loop.
10. Use `systemctl edit heartbeat.service` to add `MemoryMax=50M`. Watch `systemd-cgtop` to see the cgroup limit applied.

You have just used services, drop-ins, timers, hardening, the journal, cgroups, and analyze. That is most of systemd day-to-day.

## Where to Go Next

After this sheet, the natural next steps:

1. Read the **system/systemd** sheet for the dense reference of every option.
2. Read the **system/systemd-timers** sheet for advanced timer patterns and cron migration.
3. Read the **system/journalctl** sheet for advanced log queries and structured journal usage.
4. Look at one real distro service (e.g. `systemctl cat sshd`) and notice how many of these options you now recognize.
5. Run `systemd-analyze security --no-pager | sort -k2 -g` on a real machine and pick the worst-scored service to harden.
6. Read the **ramp-up/bash-eli5** sheet because most of your `ExecStart=` commands will be small bash scripts.
7. Read the **ramp-up/docker-eli5** sheet to compare systemd's approach to containerized services with how Docker / Podman handle the same problems.

## See Also

- **system/systemd** — full reference for options, units, and patterns.
- **system/systemd-timers** — deep dive on timer units and cron replacement.
- **system/journalctl** — power-user cookbook for the journal.
- **ramp-up/linux-kernel-eli5** — the kernel that systemd runs on top of.
- **ramp-up/bash-eli5** — the shell most ExecStart= scripts are written in.
- **ramp-up/docker-eli5** — a different way of packaging long-running services.

## References

- systemd home page — <https://systemd.io/>
- "systemd System and Service Manager" — <https://www.freedesktop.org/wiki/Software/systemd/>
- `man systemd.unit` — generic options for every unit type.
- `man systemd.service` — service-type-specific options.
- `man systemd.socket` — socket activation reference.
- `man systemd.timer` — timer reference.
- `man systemd.exec` — execution environment options (User=, Environment=, Protect*=, etc.).
- `man systemd.resource-control` — cgroup-based resource limits.
- `man systemd.target` — target unit reference.
- `man systemd.path` — path-watching units.
- `man systemd.mount` — mount unit reference.
- `man systemd-analyze` — boot timing and audit tooling.
- `man journalctl` — journal CLI.
- `man journald.conf` — journald configuration file.
- `man systemctl` — top-level CLI reference.
- `man systemd.special` — list of well-known target units.
- "Systemd Essentials" by Jürgen Kappler — concise practical book.
- "The systemd handbook" — community-maintained reference.
- Lennart Poettering's blog series "systemd for Administrators" — the original 21-part introduction.

Version notes you should know:

- **systemd 230** (2016) — cgroup v2 unified hierarchy support landed.
- **systemd 232** (2016) — `DynamicUser=` introduced; massive sandboxing improvements.
- **systemd 245** (2020) — systemd-homed and portable services shipped.
- **systemd 250** (2021) — `systemd-creds` and varlink IPC arrived.
- **systemd 254/255** (2023) — significant ARM64 nspawn improvements and tooling polish.
- **systemd 256** (2024) — `systemd-sysupdate` enhancements; various journal performance wins.

If your distro ships an older systemd, some of the options in this sheet will not exist. Check `systemctl --version` and consult the man pages for your version.
