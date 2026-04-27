# Ansible — ELI5 (Remote Control Recipes Over SSH)

> Ansible is a remote-control tool that lets you write a recipe once and have it cooked correctly on a hundred computers at the same time, just by SSH'ing in and running the steps.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` helps; you should also know basic SSH key auth)

This sheet does not assume you have ever written a playbook. It does assume you have at least once typed `ssh someuser@somebox` and gotten a shell prompt back. If you have ever logged into a server with SSH, you have everything you need to start understanding Ansible. If you have never done that, type `cs ramp-up linux-kernel-eli5` first to get a sense of what a Linux server even is, and then come back here.

If a word feels weird, look it up in the **Vocabulary** section near the bottom. Every weird word in this sheet is in that list with a one-line plain-English definition.

When you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is Ansible?

### The big picture in two sentences

You have a bunch of computers. Maybe two. Maybe two hundred. Maybe twenty thousand. They are all over the place. Some are in your house, some are in a data center, some are in the cloud. They are all running Linux (or sometimes Windows, but mostly Linux). You want to make all of them install nginx, then make sure nginx is running, then drop a config file on each one, then restart the service.

You could SSH into each one, by hand, one after the other, and type the same commands. If you have two boxes, that is annoying but doable. If you have twenty boxes, that is a long afternoon. If you have two hundred boxes, that is a week of pain and you will absolutely typo something on box 137 and not notice for three months.

Ansible is the answer to this. You write the recipe once, in a YAML file. You give Ansible a list of which computers to run the recipe on. Ansible logs into every single one of them at the same time, in parallel, runs the steps in order, and reports back what changed and what didn't. If you run the same recipe again tomorrow, Ansible will not redo the steps that are already done. It will only fix anything that has drifted out of place.

That is Ansible. Recipes you write once. Run on many computers. Idempotent. Done.

### The remote-control recipe analogy

Picture a TV remote. You point it at the TV, you press a button, the TV does something. Now imagine a magic remote that, when you press a button, makes a hundred TVs all do the same thing at the same time. You press "volume up" once, and a hundred TVs all turn their volume up at once. Press it again, all hundred go up another notch.

That is Ansible. The TVs are servers. The remote is your laptop running `ansible-playbook`. The buttons on the remote are tasks in your playbook. The "press a button" action is "run the playbook." Ansible is a remote control with a thousand-mile range and the ability to talk to a hundred TVs at once.

### The recipe analogy in detail

Pretend you are throwing a party and you are going to bake the same cake at fifty friends' houses. You write down the recipe on an index card:

```
1. Preheat oven to 350.
2. Mix flour, sugar, butter, eggs.
3. Pour into greased pan.
4. Bake 30 minutes.
5. Let cool.
```

Now imagine you have a magic friend named Ansible. You hand Ansible the index card and a list of fifty house addresses. Ansible runs to all fifty houses, lets themselves in (with the keys you gave them), reads your recipe, and bakes the cake at each house. If a house already has the oven preheated, Ansible doesn't preheat it again. If a house already has the cake done, Ansible doesn't bake another one. Ansible just makes sure that, in the end, every house has a finished cake exactly the way you described.

That is the whole idea. The recipe is the **playbook**. The list of houses is the **inventory**. The magic friend is the `ansible-playbook` command. The keys are SSH keys. The instructions on the card are **tasks**. Each task uses some standard kitchen action — preheating, mixing, pouring, baking — which we call a **module**.

### Push, not pull

Some other tools do things differently. With Puppet or Chef, every server has an agent. The agent on each server periodically wakes up, calls home to a central server, asks "is there anything new for me to do?", and if so, downloads instructions and runs them. That is the **pull** model. The server pulls work from the boss.

Ansible is the opposite. Ansible is **push**. You sit at your laptop, run a command, and Ansible reaches out to every target server and shoves the work onto it. Nothing has to be installed on the targets ahead of time. They just need SSH and a working Python. That's it.

This is why people say Ansible is **agentless**. There is no Ansible daemon running on the target. There is no service called `ansibled`. There is just Python, and there is just SSH. Both of those are already on every Linux box ever made. So Ansible can manage a server you have never touched before, the very first day, with zero pre-setup.

This is huge. With Puppet, before you can manage a new server, you have to install the Puppet agent on it, configure it to find your Puppet master, accept the cert, and so on. With Ansible, you just put the server in the inventory and SSH in. Done. That is most of the appeal.

### Why agentless matters for everyday work

Imagine you just got a new server from your hosting provider. Five minutes ago it didn't exist. Now you have its IP address and root SSH key. You want to install Docker, set up a user, drop a firewall config, and start a container.

With Ansible: add one line to your inventory file, run your playbook, done. The whole thing takes maybe two minutes. There was no agent to install. There was nothing to bootstrap. You just told Ansible "hey, also do this server now," and it did.

With other tools: install agent, configure agent, register with master, sign cert, wait for the master to find the new node, manually run a check-in, and then you can finally apply config. That is twenty minutes of plumbing before you can do five minutes of work.

This is also why Ansible plays well with cloud workflows. You spin up a new VM in AWS or GCP, get its IP, and immediately start running playbooks against it. There is no "pet" infrastructure that the agent has been baked into. It works on cattle just fine. Spin up, configure, tear down. Spin up another. Configure. Tear down. The same playbook works every time.

### Compare quickly to the other big config-management tools

Just so you know the lay of the land, here is how Ansible stacks up:

- **Puppet** — agent-based, pull-based, declarative DSL (its own language called Puppet DSL). Big in legacy enterprise. Strong typing. Steep learning curve. Master/agent setup is non-trivial.
- **Chef** — agent-based, pull-based, recipes written in Ruby. Big in legacy enterprise. Closer to writing real code. Has Solo and Zero modes that work without a server, similar to Ansible.
- **Salt** — agent-based by default but has agentless ssh mode (`salt-ssh`). Push or pull. Uses YAML like Ansible. Famous for being fast and good for thousands of nodes via ZeroMQ messaging.
- **Ansible** — agentless, push-based, YAML playbooks, Jinja2 templating. Easiest to start with. Slower than Salt at huge scale (thousands of nodes) but more than fast enough for almost everyone.

Most teams in 2026 use Ansible because the on-ramp is the easiest. There is nothing to install on the targets, the playbooks are just YAML so anyone can read them, and the community has thousands of pre-built modules and roles you can grab.

### The "I never wrote a playbook" sysadmin's mental model

If you are a sysadmin who has been running shell scripts over SSH for years, here is your mental model upgrade.

Today, you probably have a script called `setup-webserver.sh` that you scp to a new server, ssh in, run it, and watch it install nginx and drop config files. That works for one box. You probably have a wrapper for many boxes that loops over a list and runs the script via ssh.

Ansible is that, but with three big upgrades:

1. **Idempotent by default.** Your shell script probably has `if [ ! -f /etc/nginx/nginx.conf ]; then cp ./nginx.conf /etc/nginx/nginx.conf; fi` sprinkled all over. Ansible modules are idempotent out of the box. You write `copy: src=./nginx.conf dest=/etc/nginx/nginx.conf` and Ansible only copies it if the destination is different.
2. **Parallel.** Your wrapper script probably runs serially: one host at a time. Ansible runs in parallel by default (5 hosts at a time, configurable up to thousands). On 100 hosts, Ansible is 20x faster than your serial loop.
3. **Reports state.** Your script just prints whatever it prints. Ansible tells you "ok=12 changed=3 failed=0 unreachable=0" at the end. You instantly know whether anything actually changed across the fleet.

So think of Ansible as "shell scripting over SSH, but parallel, idempotent, and with a nice summary report." That's it.

### A picture of how Ansible flows

```
       YOUR LAPTOP                     TARGET SERVERS
                                       
+--------------------+                 +-----------+
|   ansible-playbook |  --(SSH)----->  | server-1  |
|     hello.yml      |  --(SSH)----->  | server-2  |
+--------------------+  --(SSH)----->  | server-3  |
        |               --(SSH)----->  | server-4  |
        |               --(SSH)----->  | server-5  |
        |
        v                              Ansible runs the
+--------------------+                 same tasks on all
|   inventory:       |                 of them in parallel,
|   hosts.ini        |                 reports back when
|                    |                 done.
|   playbook:        |
|   hello.yml        |
+--------------------+
```

The arrows are SSH connections. Ansible opens five (or fifty, or more) at the same time, runs each task on every host simultaneously, waits for them all to finish, and moves on to the next task. Then back to one host at a time? No — back to all of them at the same time, again. That is the whole loop. Tasks are run host-by-host but all hosts in parallel.

## The Three Core Concepts

If you remember nothing else from this sheet, remember these three words: **inventory**, **playbook**, **module**. Everything else in Ansible is a refinement of one of those three.

### Inventory

The inventory is the list of which computers Ansible is allowed to talk to. You give Ansible an inventory and it knows where to ssh.

The simplest inventory is a plain text file in INI format:

```
[web]
web1.example.com
web2.example.com
web3.example.com

[db]
db1.example.com
db2.example.com

[cache]
cache1.example.com
```

Each `[name]` is a **group**. Each line under a group is a **host**. A host can be in multiple groups — just list it under each group. There are also two implicit groups that always exist:

- `all` — every host in the inventory.
- `ungrouped` — any host not assigned to a custom group.

You can also put hosts in **groups of groups** using `[parent:children]`:

```
[web]
web1.example.com
web2.example.com

[db]
db1.example.com

[production:children]
web
db
```

Now `production` contains `web1`, `web2`, and `db1`.

Inventories can also be **dynamic.** Instead of a flat file, you point Ansible at a script or a plugin that queries some live system to figure out the hosts. The most common ones:

- `aws_ec2` — query AWS, list all your EC2 instances, group them by tag, region, VPC, etc.
- `gcp_compute` — same idea for Google Cloud.
- `azure_rm` — same idea for Azure.
- `docker` — list all running Docker containers.
- `kubernetes` — list all pods.

These are huge for cloud workflows. You don't have to keep a static list of IP addresses up to date when machines come and go. Ansible asks the cloud "what's running right now?" every time.

### Playbook

The playbook is the recipe. It is a YAML file. Inside the YAML file is a list of **plays**. Each play has a **target** (which group of hosts to run on) and a list of **tasks**. Each task is one thing to do, done by one **module**.

Here is the simplest playbook you can write:

```yaml
- hosts: web
  tasks:
    - name: install nginx
      apt:
        name: nginx
        state: present
```

That's it. One play. Targets the `web` group. One task. Uses the `apt` module to install nginx. Five lines including the dashes. That is a real playbook that really works.

A bigger playbook has multiple plays, multiple tasks, handlers, variables, and tags. We will see all of that.

### Module

A module is a Python program that does one specific thing on the target. There are over a thousand modules built into Ansible plus tens of thousands more in community collections.

Some everyday modules and what they do:

- `apt` — install/remove/update packages on Debian/Ubuntu.
- `yum` / `dnf` — same but for RHEL/CentOS/Fedora.
- `package` — generic, picks `apt` or `yum` automatically.
- `service` / `systemd` — start/stop/restart/enable services.
- `file` — create files and folders, set permissions, owners, modes, symlinks.
- `copy` — copy a local file to the remote host.
- `template` — copy a Jinja2 template to the remote host with variables filled in.
- `lineinfile` — make sure one line is in a file, replace if needed.
- `blockinfile` — same but for a multi-line block.
- `replace` — find-and-replace inside a file.
- `user` — create/delete/manage users.
- `group` — same for groups.
- `cron` — manage cron jobs.
- `git` — clone/update a git repo.
- `pip` — install Python packages.
- `mount` — mount/unmount filesystems.
- `firewalld` / `ufw` / `iptables` — manage firewalls.
- `shell` — run a shell command (last resort).
- `command` — run a command without shell features (also last resort).
- `debug` — print a message during the playbook run.
- `fail` — abort the playbook with an error.
- `assert` — check a condition or fail.
- `wait_for` — wait for a port to open, a file to exist, or a string to appear.
- `uri` — make HTTP requests.
- `get_url` — download a file.

Every module is **idempotent** if used correctly. That means: running the same task twice should leave the system in the same state. The first run might install nginx (changed). The second run sees nginx is already installed and does nothing (ok). That is the whole point of using modules instead of `shell:` commands. You can run your playbook over and over and over and nothing breaks.

## A Hello-World Playbook

Let's do this for real.

You have a server called `web1.example.com`. You can SSH into it as a user called `deploy` with a passwordless SSH key. The user has sudo access. You want to install nginx and start it.

Step 1: write your inventory file. Save this as `hosts.ini`:

```
[web]
web1.example.com
```

Step 2: write your playbook. Save this as `hello.yml`:

```yaml
- name: bring up the webserver
  hosts: web
  become: yes
  tasks:
    - name: install nginx
      apt:
        name: nginx
        state: present
        update_cache: yes

    - name: start nginx
      service:
        name: nginx
        state: started
        enabled: yes
```

Step 3: run it.

```
$ ansible-playbook -i hosts.ini hello.yml

PLAY [bring up the webserver] ***************************************

TASK [Gathering Facts] **********************************************
ok: [web1.example.com]

TASK [install nginx] ************************************************
changed: [web1.example.com]

TASK [start nginx] **************************************************
changed: [web1.example.com]

PLAY RECAP **********************************************************
web1.example.com : ok=3  changed=2  unreachable=0  failed=0  skipped=0
```

That's it. nginx is installed and running on `web1.example.com`. You did not SSH in. You did not run any commands manually. You wrote ten lines of YAML and one command.

Now run the same playbook again:

```
$ ansible-playbook -i hosts.ini hello.yml

PLAY [bring up the webserver] ***************************************

TASK [Gathering Facts] **********************************************
ok: [web1.example.com]

TASK [install nginx] ************************************************
ok: [web1.example.com]

TASK [start nginx] **************************************************
ok: [web1.example.com]

PLAY RECAP **********************************************************
web1.example.com : ok=3  changed=0  unreachable=0  failed=0  skipped=0
```

Notice `changed=0` this time. Ansible saw that nginx was already installed and already running, so it did nothing. **That is idempotency.** That is the whole point. You can run your playbook every day. You can run it after a reboot. You can run it during deployment. It only changes things that need changing.

### Walking through the YAML structure

Let's go line by line through that playbook:

```yaml
- name: bring up the webserver
```

The `- ` (dash space) starts a list item. This whole block is one play. The `name:` is human-readable label, not a hostname. It just shows up in the output to tell you which play is running.

```yaml
  hosts: web
```

`hosts:` is the target. It says which group from the inventory to run this play on. `web` matches the `[web]` group from `hosts.ini`. You can also write `hosts: all` to target everything, `hosts: web1.example.com` to target one specific host, or `hosts: web:!web3` for "all of `web` except `web3`."

```yaml
  become: yes
```

`become: yes` means "run all tasks in this play as root via sudo." Without this, every task would run as the `deploy` user and `apt install` would fail because it needs root.

```yaml
  tasks:
```

Now we are starting the list of tasks. Tasks come one after another and run in order.

```yaml
    - name: install nginx
      apt:
        name: nginx
        state: present
        update_cache: yes
```

One task. The `name:` is human-readable. `apt:` is the module. The four lines under `apt:` are arguments to that module. `name: nginx` says "the package called nginx." `state: present` says "make sure it's installed" (you could also say `state: absent` to uninstall, or `state: latest` to upgrade to the newest version). `update_cache: yes` is the equivalent of `apt-get update`.

```yaml
    - name: start nginx
      service:
        name: nginx
        state: started
        enabled: yes
```

Second task. Uses `service` module. `state: started` means "make sure the service is running right now." `enabled: yes` means "make sure it starts at boot."

That's the whole playbook. Two tasks. Twelve lines.

## Idempotency

This is the most important word in Ansible. Say it ten times. **Idempotent. Idempotent. Idempotent.**

Idempotent means: doing the same thing twice gives the same result as doing it once. The world ends up in the same state. Running your playbook three times should leave the server identical to running it once.

### The shell-script way (not idempotent)

Here is a shell script that installs nginx:

```bash
apt-get update
apt-get install -y nginx
systemctl start nginx
systemctl enable nginx
echo "server { listen 80; }" > /etc/nginx/sites-enabled/default
systemctl restart nginx
```

Run this once: it installs nginx, starts it, drops a config, and restarts. Fine.

Run this twice: it tries to apt update again (slow), reinstalls (might fail or just be a no-op), tries to start an already-running service, overwrites the config (which might be fine but might also clobber a manual edit you made), and restarts an already-running service for no reason.

Each of those is non-idempotent in some way. Some are harmless. Some are wasteful. Some are dangerous (overwriting the config could clobber real changes).

### The Ansible way (idempotent)

```yaml
- name: install nginx
  apt:
    name: nginx
    state: present
    update_cache: yes

- name: start nginx
  service:
    name: nginx
    state: started
    enabled: yes

- name: drop default config
  copy:
    src: ./nginx-default.conf
    dest: /etc/nginx/sites-enabled/default
  notify: restart nginx
```

Run this twice. The second run:
- `apt:` checks if nginx is installed. It is. Skip.
- `service:` checks if nginx is running. It is. Skip.
- `copy:` checks if the source matches the destination. It does (we already copied it). Skip.
- Because `copy` did nothing, the `notify: restart nginx` does not fire. No restart.

`changed=0` for the whole run. Nothing was redone. Nothing was wasted. Nothing was clobbered.

### The classic mistake: `command:` and `shell:`

If you ever write this:

```yaml
- name: install nginx
  command: apt-get install -y nginx
```

You have killed idempotency. Ansible cannot tell whether nginx is already installed. It will run `apt-get install -y nginx` every single time, which will succeed (apt is smart enough to say "already installed") but Ansible will report `changed` every time because it sees a new run of `command:`.

Even worse:

```yaml
- name: deploy app
  shell: cd /opt/myapp && git pull && systemctl restart myapp
```

That is going to redeploy and restart your app every single time the playbook runs. Even when nothing changed. That is bad.

### When you really do need shell or command

Sometimes there is no module for what you want. Maybe you need to run a custom CLI tool that nobody has written a module for. Fine. Use `command:` with `creates:` or `removes:` to make it conditionally idempotent:

```yaml
- name: run my custom tool
  command: /opt/bin/my-tool init
  args:
    creates: /var/lib/mytool/.initialized
```

`creates:` means "if this file exists, skip this task." So the first run runs `my-tool init` (and presumably my-tool creates `/var/lib/mytool/.initialized` as part of its work). Subsequent runs see the file already exists and skip. Idempotent.

`removes:` is the opposite. "Run this only if the file exists." Used for cleanup tasks.

Or you can use `changed_when:` to tell Ansible exactly when to consider something changed:

```yaml
- name: check if reboot needed
  command: needs-restarting -r
  register: reboot_check
  changed_when: false
  failed_when: reboot_check.rc not in [0, 1]
```

`changed_when: false` says "this task never reports as changed, no matter what." Useful for read-only checks.

### Idempotency in a picture

```
NON-IDEMPOTENT (shell command):
  Run 1: install nginx  -> nginx installed, "changed"
  Run 2: install nginx  -> still installed, "changed" (lies! nothing changed)
  Run 3: install nginx  -> still installed, "changed" (still lies)

IDEMPOTENT (apt module):
  Run 1: apt(present)   -> install nginx, "changed"
  Run 2: apt(present)   -> already installed, "ok"
  Run 3: apt(present)   -> already installed, "ok"
```

Top: every run reports change, but reality only changed once. Bottom: only the run that actually changed something reports change. The bottom one is what you want, all day every day.

## Variables and Templating

Hard-coding everything in your playbook is fine for "Hello, world." Real playbooks have variables — things you might want to change between environments.

### vars block in a play

```yaml
- hosts: web
  vars:
    nginx_port: 80
    nginx_user: www-data
    site_name: example.com
  tasks:
    - name: drop nginx config
      template:
        src: nginx.conf.j2
        dest: /etc/nginx/sites-enabled/default
```

Now anywhere in your tasks or templates, you can reference `{{ nginx_port }}`, `{{ nginx_user }}`, `{{ site_name }}`. Ansible substitutes them at run time.

### vars_files

If you have a lot of variables, put them in a separate YAML file and load it:

```yaml
- hosts: web
  vars_files:
    - vars/web.yml
    - vars/secrets.yml
  tasks:
    ...
```

`vars/web.yml` would just be plain YAML key-value pairs:

```yaml
nginx_port: 80
nginx_user: www-data
site_name: example.com
```

### group_vars

Even better: put variables in a folder named after the group. Ansible loads them automatically.

```
inventory/
  hosts.ini
  group_vars/
    web.yml
    db.yml
    all.yml
```

`group_vars/web.yml` is auto-loaded for every host in the `web` group. `group_vars/all.yml` is loaded for every host. `group_vars/db.yml` is loaded for hosts in the `db` group.

### host_vars

Same idea but for specific hosts:

```
inventory/
  hosts.ini
  host_vars/
    web1.example.com.yml
    web2.example.com.yml
```

`host_vars/web1.example.com.yml` is loaded only when running on `web1.example.com`. Useful for per-host overrides.

### Jinja2 templates

Variables come alive when you use them inside templates. A template is just a regular file with `{{ var_name }}` placeholders. Ansible runs the file through Jinja2 before copying it to the target.

`templates/nginx.conf.j2`:

```
server {
    listen {{ nginx_port }};
    server_name {{ site_name }};
    user {{ nginx_user }};
    root /var/www/{{ site_name }};
}
```

The `.j2` extension is convention. The `template:` module renders this and copies the result to the target:

```yaml
- name: drop nginx config
  template:
    src: nginx.conf.j2
    dest: /etc/nginx/sites-enabled/default
  notify: restart nginx
```

If `site_name=example.com`, `nginx_port=80`, `nginx_user=www-data`, the rendered file on the target will be:

```
server {
    listen 80;
    server_name example.com;
    user www-data;
    root /var/www/example.com;
}
```

### Filters

Jinja2 has filters that transform values. You apply them with `|`:

```
{{ my_list | join(',') }}     -- joins a list with commas
{{ my_string | upper }}       -- uppercase
{{ my_dict | to_json }}       -- convert to JSON
{{ my_value | default('x') }} -- use 'x' if my_value isn't set
{{ ip_addr | ipaddr('network') }} -- network address
```

There are dozens of filters. Some Ansible-specific ones include `b64encode`, `b64decode`, `regex_replace`, `regex_search`, `to_yaml`, `from_yaml`, `dict2items`, `items2dict`. Look up the docs when you need them.

### Variable precedence (the gotcha)

Variables can be defined in like 22 different places. When the same variable is set in multiple places, which one wins? Here is the rough order, from lowest priority to highest (later wins):

1. Role defaults (`defaults/main.yml`).
2. Inventory `group_vars/all.yml`.
3. Inventory `group_vars/<group>.yml`.
4. Inventory `host_vars/<host>.yml`.
5. Play `vars:` block.
6. Play `vars_files:`.
7. Block `vars:`.
8. Task `vars:`.
9. `set_fact`.
10. Extra vars (`-e` on the command line).

So `-e foo=bar` always wins. That is why you can override anything with `-e` on the command line for a one-off.

If your variable isn't doing what you expect, look at where it's defined. Use `-vvv` (very verbose) to see what value Ansible ended up with.

## Roles

Once your playbook gets long, split it into roles. A role is a packaged set of related tasks, files, templates, defaults, handlers, and metadata that you can drop into any playbook and reuse.

### The standard role layout

```
roles/
  nginx/
    tasks/
      main.yml         <- the tasks the role does
    handlers/
      main.yml         <- handlers (see next section)
    files/
      logo.png         <- static files to copy
    templates/
      nginx.conf.j2    <- Jinja2 templates
    defaults/
      main.yml         <- default variable values (lowest priority)
    vars/
      main.yml         <- role-specific variables (higher priority)
    meta/
      main.yml         <- role metadata: dependencies, author, etc.
    README.md
```

You don't need all of these directories. Most roles have at minimum `tasks/main.yml`. The others are optional.

### Using a role

Once you have a role at `roles/nginx/`, your top-level playbook becomes:

```yaml
- hosts: web
  become: yes
  roles:
    - nginx
```

Three lines. Ansible runs through `roles/nginx/tasks/main.yml`, picks up handlers from `roles/nginx/handlers/main.yml`, finds files in `roles/nginx/files/`, finds templates in `roles/nginx/templates/`, and so on. All the path lookups are scoped to the role.

### Multiple roles in one play

```yaml
- hosts: web
  become: yes
  roles:
    - common
    - nginx
    - my-app
```

Ansible runs them in order. `common` first (probably some baseline config), then `nginx`, then `my-app`.

### Role variables

Variables live in `defaults/main.yml` (low priority, easy to override) or `vars/main.yml` (high priority, hard to override). Best practice: put settings users might tweak in `defaults/`, and put hard-coded internal values in `vars/`.

### Role dependencies

If your `my-app` role needs nginx to be installed first, you can declare a dependency in `roles/my-app/meta/main.yml`:

```yaml
dependencies:
  - role: nginx
```

Now whenever you use `my-app`, Ansible automatically runs `nginx` first.

## Handlers

A handler is a task that only runs when something else notifies it. The classic example: "if I changed the config, restart the service."

### How handlers work

```yaml
- name: drop nginx config
  template:
    src: nginx.conf.j2
    dest: /etc/nginx/nginx.conf
  notify: restart nginx
```

```yaml
# in handlers/main.yml or under handlers: in the play
handlers:
  - name: restart nginx
    service:
      name: nginx
      state: restarted
```

Two key things about handlers:

1. They only fire if a task notifies them with `notify:`.
2. They only fire if the notifying task changed something. If the template was already in place and `template:` reported `ok` (not `changed`), the notify is ignored.
3. Handlers run at the **end** of the play, not immediately. So you can change five things and only get one restart, not five.

### The classic example

```yaml
- name: drop main config
  template:
    src: nginx.conf.j2
    dest: /etc/nginx/nginx.conf
  notify: restart nginx

- name: drop site config
  template:
    src: site.conf.j2
    dest: /etc/nginx/sites-enabled/site.conf
  notify: restart nginx

- name: drop another site
  template:
    src: site2.conf.j2
    dest: /etc/nginx/sites-enabled/site2.conf
  notify: restart nginx

# handlers run at end of play
handlers:
  - name: restart nginx
    service:
      name: nginx
      state: restarted
```

If only the first config file changed: nginx restarts once at the end. If all three changed: nginx still restarts only once at the end. If none changed: nginx does not restart at all. Beautiful.

### Forcing handlers immediately

Sometimes you need handlers to run right now, not at the end. Use `meta: flush_handlers`:

```yaml
- name: drop config
  template:
    src: foo.j2
    dest: /etc/foo
  notify: restart foo

- meta: flush_handlers

- name: do something that depends on foo being restarted
  ...
```

`flush_handlers` is "run any pending handlers right now."

## Become and Privilege Escalation

Most of the time, your tasks need to do things that require root: install packages, edit `/etc/`, restart services. You log in as a regular user (because logging in as root over SSH is a bad practice) and then sudo to root for the privileged work.

### become: yes

```yaml
- hosts: web
  become: yes
  tasks:
    - name: install package
      apt:
        name: htop
        state: present
```

`become: yes` at the play level means "all tasks in this play sudo to root." Ansible will use sudo (the default) to escalate.

You can also set become per-task:

```yaml
- hosts: web
  tasks:
    - name: read public file
      slurp:
        src: /etc/hostname

    - name: edit private file
      lineinfile:
        path: /etc/sudoers
        line: "deploy ALL=(ALL) NOPASSWD:ALL"
      become: yes
```

Only the second task uses sudo.

### become_user

Default become user is root. To sudo to a specific other user:

```yaml
- name: do thing as deployer
  command: /opt/bin/deploy
  become: yes
  become_user: deployer
```

This is useful for app-specific operations where you want to run as the app's user, not root.

### become_method

Default become method is `sudo`. Other options:

- `sudo` — the standard `sudo` command (default).
- `su` — `su -` (older, less common).
- `doas` — OpenBSD's sudo replacement.
- `pbrun` — PowerBroker (enterprise).
- `runas` — Windows.

```yaml
- hosts: web
  become: yes
  become_method: doas
  tasks:
    ...
```

### Sudo password

If sudo on your target asks for a password, you need `--ask-become-pass` (or `-K`):

```
$ ansible-playbook -i hosts.ini playbook.yml --ask-become-pass
BECOME password: <type sudo password>
```

This prompts once and uses the same password on all hosts.

Better: configure passwordless sudo on the targets. Drop a sudoers file like `/etc/sudoers.d/ansible`:

```
deploy ALL=(ALL) NOPASSWD:ALL
```

Now `become: yes` works with no password prompt.

### A picture of become

```
       ANSIBLE                       TARGET SERVER
                                     
+-----------------+   ssh deploy@   +-----------+
| ansible-playbook|  -------------> | deploy    |  (login user)
|   become: yes   |                 |    |      |
+-----------------+                 |    | sudo |
                                    |    v      |
                                    | root      |  (effective user for tasks)
                                    +-----------+
                                    | apt install nginx |
                                    | etc.              |
                                    +-------------------+
```

You ssh in as `deploy`. You become `root` via sudo. Tasks run as root. Done.

## Vault

Sometimes your variables are secrets. Database passwords. API keys. SSL private keys. You don't want to commit those in plain text to git. Ansible Vault encrypts them.

### Encrypting a file

```
$ ansible-vault encrypt secrets.yml
New Vault password: <type a password>
Confirm New Vault password: <type same password>
Encryption successful
```

The file `secrets.yml` is now encrypted. Cat it:

```
$ cat secrets.yml
$ANSIBLE_VAULT;1.1;AES256
65613664303932643539363733...
3137666463633862646435313...
6438393734386362363761396...
6664...
```

Gibberish. Safe to commit to git.

### Decrypting a file

```
$ ansible-vault decrypt secrets.yml
Vault password: <type the password>
Decryption successful
```

Now `secrets.yml` is back to plain YAML.

### Viewing without decrypting

```
$ ansible-vault view secrets.yml
Vault password: <type password>
db_password: hunter2
api_key: abc123
```

Just shows you the contents without changing the file.

### Editing in place

```
$ ansible-vault edit secrets.yml
```

Opens your `$EDITOR` with the decrypted contents. When you save and quit, it re-encrypts.

### Running a playbook that uses vaulted files

```
$ ansible-playbook -i hosts.ini playbook.yml --ask-vault-pass
Vault password: <type password>
```

Or store the password in a file (and chmod 0600):

```
$ ansible-playbook -i hosts.ini playbook.yml --vault-password-file ~/.vault_pass
```

Or use an environment variable:

```
$ export ANSIBLE_VAULT_PASSWORD_FILE=~/.vault_pass
$ ansible-playbook -i hosts.ini playbook.yml
```

### Encrypting just a string

You don't have to encrypt a whole file. You can encrypt a single string and embed it in plain YAML:

```
$ ansible-vault encrypt_string 'hunter2' --name 'db_password'
db_password: !vault |
    $ANSIBLE_VAULT;1.1;AES256
    65613664303932643539363733...
    ...
```

Drop that block into a regular YAML file. Most of the file is plain text; just the secret line is encrypted.

### Vault picture

```
PLAINTEXT FILE:                ENCRYPTED FILE:
                               
db_password: hunter2           $ANSIBLE_VAULT;1.1;AES256
api_key: abc123                65613664303932643539363733...
                               6438393734386362363761396...
                                                            
       |                              |
       | ansible-vault encrypt        | ansible-vault decrypt
       v                              v
                                      
$ANSIBLE_VAULT;1.1;AES256      db_password: hunter2
65613664303932643539...        api_key: abc123
6438393734386362...            
                               
       \________ commit to git ________/
                       (only the encrypted form goes in)
```

Plain text on the left. Encrypted on the right. Encrypt with the password to lock. Decrypt with the password to unlock. Only the right side ever goes into git.

## Galaxy and Collections

You don't have to write everything from scratch. The Ansible community has built thousands of reusable roles and collections. Ansible Galaxy is the package manager for them.

### Installing a role from Galaxy

```
$ ansible-galaxy role install geerlingguy.docker
- downloading role 'docker', owned by geerlingguy
- downloading role from https://github.com/geerlingguy/ansible-role-docker/archive/7.4.0.tar.gz
- extracting geerlingguy.docker to /home/me/.ansible/roles/geerlingguy.docker
- geerlingguy.docker (7.4.0) was installed successfully
```

Now you can use it in your playbook:

```yaml
- hosts: dockerhosts
  become: yes
  roles:
    - geerlingguy.docker
```

Done. Docker installed. Jeff Geerling has dozens of these roles for nginx, postgres, php, java, mysql, and more.

### Collections

Since Ansible 2.9, the new packaging unit is a **collection**. A collection is a bundle of roles, modules, plugins, and docs, all under a namespace. Format: `namespace.collection_name`. Examples:

- `community.general` — the big general-purpose community collection.
- `community.docker` — Docker-related modules.
- `community.postgresql` — Postgres modules.
- `ansible.posix` — POSIX utilities.
- `cisco.ios` — Cisco IOS networking modules.
- `kubernetes.core` — Kubernetes modules.
- `amazon.aws` — AWS modules.

Install one:

```
$ ansible-galaxy collection install community.general
Process install dependency map
Starting collection install process
Installing 'community.general:8.5.0' to '/home/me/.ansible/collections/ansible_collections/community/general'
```

Use a module from it in your playbook:

```yaml
- name: send slack notification
  community.general.slack:
    token: "{{ slack_token }}"
    msg: "Deploy complete"
```

The fully qualified name `community.general.slack` is the modern style. Old-style short names (`slack:`) still work for backwards compat, but FQCN is preferred.

### requirements.yml

Standard practice: list your dependencies in a `requirements.yml` file:

```yaml
collections:
  - name: community.general
    version: ">=8.0.0"
  - name: community.docker
    version: ">=3.0.0"

roles:
  - name: geerlingguy.docker
    version: "7.4.0"
```

Then anyone working on the project runs:

```
$ ansible-galaxy install -r requirements.yml
```

And gets all the same dependencies. Like `pip install -r requirements.txt` or `npm install`.

## Inventories — Static and Dynamic

We covered the basics earlier. Here is more.

### Static INI

```
[web]
web1.example.com
web2.example.com ansible_host=10.0.0.5 ansible_port=2222
web3.example.com ansible_user=root

[db]
db1.example.com
db2.example.com

[cache]
cache1.example.com

[loadbalancers]
lb1.example.com
lb2.example.com

[production:children]
web
db
cache

[staging]
staging1.example.com
staging2.example.com

[development]
localhost ansible_connection=local
```

Notes:
- `ansible_host` overrides the actual hostname/IP to ssh to (the inventory name is just a label).
- `ansible_port` overrides the SSH port.
- `ansible_user` overrides the SSH username.
- `ansible_connection=local` means "don't ssh, run on this machine directly."
- `[parent:children]` groups multiple groups together.

### Static YAML

Same thing in YAML:

```yaml
all:
  children:
    web:
      hosts:
        web1.example.com:
        web2.example.com:
          ansible_host: 10.0.0.5
          ansible_port: 2222
    db:
      hosts:
        db1.example.com:
        db2.example.com:
    production:
      children:
        web:
        db:
```

YAML is more verbose but supports nested structures better.

### Dynamic inventory: aws_ec2

Create `inventory/aws.yml`:

```yaml
plugin: amazon.aws.aws_ec2
regions:
  - us-east-1
filters:
  tag:Environment: production
keyed_groups:
  - prefix: tag
    key: tags
hostnames:
  - tag:Name
  - dns-name
  - private-ip-address
```

Run:

```
$ ansible-inventory -i inventory/aws.yml --list
```

Ansible queries AWS, finds all EC2 instances tagged `Environment=production` in `us-east-1`, and turns them into a live inventory grouped by tags. Now your playbook just runs against `aws_ec2` or specific groups, and Ansible figures out the IPs every time.

### Dynamic inventory: gcp_compute

```yaml
plugin: gcp_compute
projects:
  - my-project-id
auth_kind: serviceaccount
service_account_file: /path/to/sa.json
keyed_groups:
  - prefix: zone
    key: zone
```

Same idea, GCP version.

### Listing the inventory

```
$ ansible-inventory -i hosts.ini --list
```

Dumps the inventory as JSON. Useful for debugging.

```
$ ansible-inventory -i hosts.ini --graph
```

ASCII-art tree of the inventory.

```
@all:
  |--@web:
  |  |--web1.example.com
  |  |--web2.example.com
  |--@db:
  |  |--db1.example.com
  |--@ungrouped:
```

Easy to read.

## Connection Plugins

By default Ansible connects via SSH. But there are other connection plugins for special targets.

- `ssh` — the default. OpenSSH client.
- `paramiko` — pure-Python SSH (used in some edge cases, like ssh-pw on older systems).
- `local` — run on this machine directly. Set `ansible_connection: local` in inventory.
- `docker` — exec into a Docker container. `ansible_connection: docker` and `ansible_host: container_id`.
- `kubectl` — exec into a Kubernetes pod. `ansible_connection: kubectl`.
- `winrm` — Windows targets. `ansible_connection: winrm` and friends.
- `psrp` — alternative Windows PowerShell-based connection.
- `network_cli` — Cisco IOS, Juniper, Arista, etc., via expect-like interaction.
- `httpapi` — modern network device API connection.
- `netconf` — RFC 6241 NETCONF for network devices.

For day-to-day Linux work, you only ever need `ssh`. The rest exist for niche cases.

### Local connection example

Run an entire playbook on the machine you're sitting at:

```
$ ansible-playbook -c local playbook.yml
```

Or in inventory:

```
[local]
localhost ansible_connection=local
```

## Error Handling

Sometimes tasks fail. Sometimes you want them to fail. Sometimes you want to ignore failures. Ansible has tools for all of these.

### ignore_errors

```yaml
- name: try to do something
  command: /maybe-broken-tool
  ignore_errors: yes
```

If this task fails, Ansible prints the error but keeps going. The task will be marked failed in the recap, but the playbook continues.

### failed_when

Decide for yourself when a task is failed:

```yaml
- name: check disk usage
  command: df -h /
  register: disk_check
  failed_when: "disk_check.stdout.find('100%') != -1"
```

If `df`'s output contains "100%", consider the task failed (because the disk is full).

### changed_when

Decide when a task should be considered "changed" instead of "ok":

```yaml
- name: check status
  command: my-status-tool
  register: status
  changed_when: status.rc == 1
```

`my-status-tool` returns 1 when there's drift. Otherwise it returns 0. We say `changed_when: status.rc == 1` so a "drift detected" exit reports as changed.

### block / rescue / always

Ansible's try/catch/finally:

```yaml
- name: do stuff with rollback
  block:
    - name: drop new config
      template:
        src: app.conf.j2
        dest: /etc/myapp/app.conf

    - name: restart app
      service:
        name: myapp
        state: restarted

    - name: smoke test
      uri:
        url: http://localhost:8080/health
        status_code: 200

  rescue:
    - name: roll back config
      command: cp /etc/myapp/app.conf.bak /etc/myapp/app.conf

    - name: restart app
      service:
        name: myapp
        state: restarted

  always:
    - name: log the run
      lineinfile:
        path: /var/log/deploy.log
        line: "Deployed at {{ ansible_date_time.iso8601 }}"
```

If any task in `block:` fails, `rescue:` runs (rollback). Whether or not `block:` and `rescue:` succeed, `always:` runs (logging). This is the cleanest way to do safe deployments with rollback.

### Why these matter for idempotency

Idempotency is about reaching the same end state. Sometimes a task fails halfway through and leaves things in a weird state. `block: / rescue:` lets you recover. `failed_when:` and `changed_when:` let you correctly report state for non-standard tools.

If a task runs a custom script that always exits 0 even when it does work, you need `changed_when:` to tell Ansible "look at stdout to figure out if anything actually happened." Otherwise Ansible thinks every run is `ok` even when stuff changed, or vice versa.

## Common Errors

These are the errors you will run into in your first month with Ansible. Fix is right there.

### "Failed to connect to the host via ssh"

```
fatal: [web1.example.com]: UNREACHABLE! => {"changed": false,
"msg": "Failed to connect to the host via ssh:
ssh: connect to host web1.example.com port 22: Connection refused",
"unreachable": true}
```

SSH cannot reach the host at all. Possible causes:
1. The host is down or sshd isn't running.
2. Firewall is blocking port 22.
3. Wrong hostname/IP in inventory.
4. Wrong user.
5. SSH key not on the remote host's `~/.ssh/authorized_keys`.

Fix: try `ssh user@host` manually first. If that doesn't work, Ansible can't either. Set up SSH first.

### "missing sudo password"

```
fatal: [web1.example.com]: FAILED! => {"msg": "Missing sudo password"}
```

Your `become: yes` task needs sudo, but sudo is asking for a password and Ansible doesn't have one.

Fix: either pass `--ask-become-pass` (or `-K`) to prompt for it, or configure passwordless sudo on the target via `/etc/sudoers.d/`.

### "module not found"

```
fatal: [web1.example.com]: FAILED! => {"msg": "couldn't resolve module/action 'community.docker.docker_container'.
This often indicates a misspelling, missing collection, or incorrect module path."}
```

You used a module that lives in a collection you don't have installed.

Fix:

```
$ ansible-galaxy collection install community.docker
```

Done.

### "Could not find or access '<file>'"

```
fatal: [web1.example.com]: FAILED! => {"msg": "Could not find or access 'templates/nginx.conf.j2'"}
```

Ansible can't find the file you referenced.

Fix:
1. Check the path is relative to where you run `ansible-playbook` from, not relative to the playbook file.
2. If you're inside a role, files are auto-located in the role's `files/`, `templates/`, etc.
3. Check spelling.

### "Got a 403 from EC2 metadata"

```
fatal: [localhost]: FAILED! => {"msg": "An error occurred (UnauthorizedOperation) when calling the DescribeInstances operation"}
```

Dynamic inventory tried to query AWS but doesn't have permission.

Fix: configure AWS credentials. Set `AWS_PROFILE`, `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, or attach an IAM role to the EC2 instance running Ansible.

### "AnsibleFileNotFound"

```
TASK [copy nginx.conf]
fatal: [web1.example.com]: FAILED! => {"msg": "Could not find or access 'nginx.conf'\nSearched in:\n\tfiles/nginx.conf"}
```

Same as the file-not-found error above, just nicer formatted. Same fix.

### "WARNING: empty inventory"

```
$ ansible web -m ping
[WARNING]:  * Failed to parse /etc/ansible/hosts with yaml plugin: Plugin Couldn't parse inventory source as YAML
[WARNING]: provided hosts list is empty, only localhost is available.
```

You didn't pass `-i hosts.ini`, so Ansible tried to use the default inventory at `/etc/ansible/hosts`, which is empty.

Fix: always pass `-i hosts.ini` (or set `ANSIBLE_INVENTORY=hosts.ini` in your environment).

### "ssh: Host key verification failed"

```
fatal: [web1.example.com]: UNREACHABLE! =>
{"msg": "Failed to connect to the host via ssh: Host key verification failed."}
```

SSH doesn't trust the host's fingerprint. Either it's a new host (not in `~/.ssh/known_hosts`), or the host's key changed (potentially scary — could be a man-in-the-middle).

Fix:
- For a known new host: `ssh-keyscan host >> ~/.ssh/known_hosts`.
- Or set `host_key_checking = False` in `ansible.cfg` (not great for production but fine for testing).

### "sudo: a password is required"

```
fatal: [web1.example.com]: FAILED! => {"msg": "Missing sudo password"}
```

Same as "missing sudo password" above. Same fix.

### "fatal: [host]: FAILED! => 'utf-8' codec can't decode"

```
{"msg": "Module returned non UTF-8 data"}
```

Some module returned weird binary on stdout. Usually a misuse of `command:` against a tool that prints binary.

Fix: use a real module instead of `command:`. Or pipe to `base64` and decode in Ansible.

### "Authentication failed"

```
{"msg": "Authentication failure."}
```

SSH key isn't accepted. Check:
- The right private key (`-i ~/.ssh/whatever` or `ansible_ssh_private_key_file` in inventory).
- The matching public key is in `~/.ssh/authorized_keys` on the target.
- Permissions on the private key are 0600.

## Hands-On

Below are real commands you should type at a real terminal. Output is what you should expect to see.

### Check your Ansible version

```
$ ansible --version
ansible [core 2.16.4]
  config file = /home/me/.ansible.cfg
  configured module search path = ['/home/me/.ansible/plugins/modules', '/usr/share/ansible/plugins/modules']
  ansible python module location = /usr/lib/python3.11/site-packages/ansible
  ansible collection location = /home/me/.ansible/collections:/usr/share/ansible/collections
  executable location = /usr/bin/ansible
  python version = 3.11.6 (main, Oct 30 2023, 00:00:00) [GCC 13.2.1]
  jinja version = 3.1.2
  libyaml = True
```

Tells you which Ansible you have, where it lives, and what Python it's using. Always run this first when troubleshooting.

### Ad-hoc ping (the "hello" check)

```
$ ansible all -i hosts.ini -m ping
web1.example.com | SUCCESS => {
    "ansible_facts": {
        "discovered_interpreter_python": "/usr/bin/python3"
    },
    "changed": false,
    "ping": "pong"
}
web2.example.com | SUCCESS => {
    "changed": false,
    "ping": "pong"
}
```

This is the absolute first thing you do with a new host. It uses the `ping` module (which is not ICMP — it's a real Python module that proves Ansible can SSH in and run Python on the target). If you get `pong` back, you're good. If you get `UNREACHABLE`, fix SSH first.

### Gather facts

```
$ ansible web -i hosts.ini -m setup
web1.example.com | SUCCESS => {
    "ansible_facts": {
        "ansible_all_ipv4_addresses": [
            "10.0.0.5"
        ],
        "ansible_architecture": "x86_64",
        "ansible_distribution": "Ubuntu",
        "ansible_distribution_release": "noble",
        "ansible_distribution_version": "24.04",
        "ansible_dns": {
            "nameservers": ["1.1.1.1", "8.8.8.8"]
        },
        ... (hundreds more lines) ...
    }
}
```

The `setup` module gathers **facts** about the target. Hundreds of variables get populated: OS version, IP addresses, memory size, mounted filesystems, kernel version, etc. You can use any of these in your playbook as `{{ ansible_distribution }}`, `{{ ansible_memtotal_mb }}`, and so on.

### Filter facts

```
$ ansible web -i hosts.ini -m setup -a 'filter=ansible_distribution*'
web1.example.com | SUCCESS => {
    "ansible_facts": {
        "ansible_distribution": "Ubuntu",
        "ansible_distribution_file_parsed": true,
        "ansible_distribution_file_path": "/etc/os-release",
        "ansible_distribution_file_variety": "Debian",
        "ansible_distribution_major_version": "24",
        "ansible_distribution_release": "noble",
        "ansible_distribution_version": "24.04"
    }
}
```

When the full setup output is too much, `filter=` keeps only matching facts. Useful for quick checks like "what OS is this thing?"

### Install a package ad-hoc

```
$ ansible web -i hosts.ini -m apt -a 'name=curl state=present' --become
web1.example.com | CHANGED => {
    "cache_update_time": 1714230000,
    "cache_updated": false,
    "changed": true,
    "stderr": "",
    "stdout": "Reading package lists...\nBuilding dependency tree...\ncurl is already the newest version (8.5.0-2ubuntu10.1).\n"
}
```

This is the same as a one-task playbook. `--become` is the CLI version of `become: yes`. Useful for one-off commands.

### Restart a service ad-hoc

```
$ ansible web -i hosts.ini -m service -a 'name=nginx state=restarted' --become
web1.example.com | CHANGED => {
    "changed": true,
    "name": "nginx",
    "state": "started",
    ...
}
```

Quick "restart nginx everywhere" command.

### Run a shell command ad-hoc

```
$ ansible web -i hosts.ini -a 'uptime'
web1.example.com | CHANGED | rc=0 >>
 14:32:01 up  3 days,  2:14,  2 users,  load average: 0.05, 0.03, 0.00
web2.example.com | CHANGED | rc=0 >>
 14:32:01 up  1 day,   8:42,  1 user,  load average: 0.10, 0.04, 0.01
```

When you don't pass `-m`, the default module is `command`. So `ansible web -a 'uptime'` runs `uptime` on every web host. Quick way to check things.

### Check mode (dry run)

```
$ ansible-playbook hello.yml -i hosts.ini --check

PLAY [bring up the webserver] *********************************

TASK [Gathering Facts] ****************************************
ok: [web1.example.com]

TASK [install nginx] ******************************************
changed: [web1.example.com]

TASK [start nginx] ********************************************
changed: [web1.example.com]

PLAY RECAP ****************************************************
web1.example.com : ok=3 changed=2 unreachable=0 failed=0
```

`--check` runs the playbook in pretend mode. Ansible says "this is what I would do" without actually doing it. Sometimes called dry run. Hugely useful before you blast a change out to 200 production servers. Note: not all modules support check mode perfectly, so you might still see some real action for `command:` and `shell:` tasks.

### Show file diffs

```
$ ansible-playbook hello.yml -i hosts.ini --diff

TASK [drop nginx config] **************************************
--- before: /etc/nginx/sites-enabled/default
+++ after: /home/me/.ansible/tmp/...
@@ -1,4 +1,5 @@
 server {
     listen 80;
+    server_name example.com;
 }

changed: [web1.example.com]
```

`--diff` shows you exactly what changed in any file Ansible modified. Combine with `--check` to see what would change without doing it: `--check --diff`. The combination is your safest possible "preview" before a real run.

### Verbose output

```
$ ansible-playbook hello.yml -i hosts.ini -vvv
```

`-v` shows extra info. `-vv` more. `-vvv` even more. `-vvvv` so much you wish you hadn't.

`-vvv` is great for debugging. It prints the full SSH command, the temp directory on the target, the Python interpreter being used, the JSON sent to and received from the module. If your task is misbehaving, run with `-vvv` and you can usually see why.

### Start at a specific task

```
$ ansible-playbook hello.yml -i hosts.ini --start-at-task='install nginx'
```

Skips everything before the task named "install nginx" and starts from there. Useful when a long playbook fails halfway and you want to resume without redoing the early tasks.

### Run only tasks with a tag

```
$ ansible-playbook hello.yml -i hosts.ini --tags=config
```

Only runs tasks tagged `config`. To use this, your playbook needs tags:

```yaml
- name: drop nginx config
  template:
    src: nginx.conf.j2
    dest: /etc/nginx/nginx.conf
  tags: config
```

### Skip tasks with a tag

```
$ ansible-playbook hello.yml -i hosts.ini --skip-tags=db
```

Runs everything except tasks tagged `db`.

### Limit to specific hosts

```
$ ansible-playbook hello.yml -i hosts.ini --limit=web1.example.com
```

Even though the playbook targets `web` (which has multiple hosts), only `web1` runs. You can use globs (`--limit='web1*'`) and group names (`--limit=web,db`).

### Pass extra variables

```
$ ansible-playbook hello.yml -i hosts.ini -e "version=1.21"
```

`-e` (extra vars) override anything else. Now `{{ version }}` evaluates to `1.21` everywhere in the playbook.

### List the inventory as JSON

```
$ ansible-inventory -i hosts.ini --list
{
    "_meta": {
        "hostvars": {
            "web1.example.com": {},
            "web2.example.com": {}
        }
    },
    "all": {
        "children": ["ungrouped", "web", "db"]
    },
    "db": {
        "hosts": ["db1.example.com"]
    },
    "web": {
        "hosts": ["web1.example.com", "web2.example.com"]
    }
}
```

Dumps the whole inventory as JSON. Helpful for debugging or for scripts that want to know what's in the inventory.

### Show inventory as a tree

```
$ ansible-inventory -i hosts.ini --graph
@all:
  |--@db:
  |  |--db1.example.com
  |--@ungrouped:
  |--@web:
  |  |--web1.example.com
  |  |--web2.example.com
```

ASCII-art tree of the inventory. Easier to read than the JSON.

### Dump only changed config

```
$ ansible-config dump --only-changed
DEFAULT_HOST_LIST(/home/me/.ansible.cfg) = ['hosts.ini']
DEFAULT_REMOTE_USER(/home/me/.ansible.cfg) = deploy
HOST_KEY_CHECKING(/home/me/.ansible.cfg) = False
```

Shows you which Ansible config settings differ from the defaults. Useful for figuring out where settings come from.

### Encrypt a vault file

```
$ ansible-vault encrypt secrets.yml
New Vault password: <type>
Confirm New Vault password: <type>
Encryption successful
```

### Decrypt a vault file

```
$ ansible-vault decrypt secrets.yml
Vault password: <type>
Decryption successful
```

### View a vault file

```
$ ansible-vault view secrets.yml
Vault password: <type>
db_password: hunter2
api_key: abc123
```

### Re-key a vault file (change the password)

```
$ ansible-vault rekey secrets.yml
Vault password: <type old>
New Vault password: <type new>
Confirm New Vault password: <type new>
Rekey successful
```

### List installed collections

```
$ ansible-galaxy collection list | head -20

# /home/me/.ansible/collections/ansible_collections
Collection                    Version
----------------------------- -------
amazon.aws                    7.3.0
ansible.netcommon             6.1.0
ansible.posix                 1.5.4
ansible.utils                 4.1.0
community.crypto              2.18.0
community.docker              3.10.1
community.general             8.5.0
community.kubernetes          2.0.1
community.postgresql          3.2.0
kubernetes.core               3.0.1
```

Shows every collection installed and its version.

### Install a collection

```
$ ansible-galaxy collection install community.general
Process install dependency map
Starting collection install process
Installing 'community.general:8.5.0' to '/home/me/.ansible/collections/ansible_collections/community/general'
Installing 'ansible.posix:1.5.4' to '/home/me/.ansible/collections/ansible_collections/ansible/posix'
```

### Install a community role

```
$ ansible-galaxy role install geerlingguy.docker
- downloading role 'docker', owned by geerlingguy
- downloading role from https://github.com/geerlingguy/ansible-role-docker/archive/7.4.0.tar.gz
- extracting geerlingguy.docker to /home/me/.ansible/roles/geerlingguy.docker
- geerlingguy.docker (7.4.0) was installed successfully
```

### List all modules

```
$ ansible-doc -l | head -20
add_host                                              Add a host (and alternatively a group) to the ansible-playbook in-memory inventory
amazon.aws.aws_az_facts                               Gather information about availability zones in AWS
amazon.aws.aws_az_info                                Gather information about availability zones in AWS
amazon.aws.aws_caller_facts                           Get information about the user and account being used to make AWS calls
ansible.builtin.add_host                              Add a host (and alternatively a group) to the ansible-playbook in-memory inventory
ansible.builtin.apt                                   Manages apt-packages
ansible.builtin.apt_key                               Add or remove an apt key
ansible.builtin.apt_repository                        Add and remove APT repositories
ansible.builtin.assemble                              Assembles a configuration file from fragments
ansible.builtin.assert                                Asserts given expressions are true
ansible.builtin.async_status                          Check the way back to async tasks
ansible.builtin.blockinfile                           Insert/update/remove a block of multi-line text in a managed host
ansible.builtin.command                               Execute commands on targets
ansible.builtin.copy                                  Copies files to remote locations
ansible.builtin.cron                                  Manage cron.d and crontab entries
ansible.builtin.debug                                 Print a message during execution
ansible.builtin.dnf                                   Manages packages with the dnf package manager
ansible.builtin.dpkg_selections                       Manage dpkg package selection selections
ansible.builtin.expect                                Executes the "expect" module to run a command and respond to prompts
ansible.builtin.fetch                                 Manages apt-packages
```

Lists every available module with a one-line description. Pipe to `grep` to find one quickly.

### Read module docs

```
$ ansible-doc apt | head -40
> ANSIBLE.BUILTIN.APT    (/usr/lib/python3/dist-packages/ansible/modules/apt.py)

  Manages `apt' packages (such as for Debian/Ubuntu).

OPTIONS (= is mandatory):

- allow_change_held_packages
        Allows changing the version of a package which is on
        the apt hold list.
        [Default: False]
        type: bool
        version_added: 2.13
        version_added_collection: ansible.builtin

- allow_downgrade
        Corresponds to the `--allow-downgrades' option for `apt'.
        Allows downgrading of packages.
        [Default: False]
        type: bool

- name (aliases: package, pkg)
        A list of package names, like `foo', or package
        specifier with version, like `foo=1.0'.
        Mutually exclusive with `upgrade'.
        ...
```

`ansible-doc <module>` shows you the full docs for any module. No need to leave the terminal. No need to open docs.ansible.com. Everything is right here.

### Lint your playbook

```
$ ansible-lint hello.yml
WARNING: PATH altered to include /home/me/.local/lib/python3.11/site-packages/ansiblelint/_data
hello.yml:1: yaml[indentation]: Wrong indentation: expected 2 but found 0
hello.yml:5: name[casing]: All names should start with an uppercase letter.

Failed: 2 failures, 0 warnings
```

`ansible-lint` checks your playbook for common mistakes and bad style. Run it before committing. It will catch dozens of issues you might not notice.

### Test a role with Molecule

```
$ molecule test
INFO     default scenario test matrix: dependency, lint, cleanup, destroy, syntax, create, prepare, converge, idempotence, side_effect, verify, cleanup, destroy
INFO     Performing prerun with role_name_check=0...
INFO     Running default > dependency
INFO     Running default > syntax
...
INFO     Running default > converge
TASK [Gathering Facts] ***
TASK [my-role : install package] ***
changed: [instance]
...
INFO     Running default > idempotence
TASK [my-role : install package] ***
ok: [instance]
INFO     Idempotence completed successfully.
```

Molecule is the testing framework for Ansible roles. It spins up a Docker container or a VM, applies your role, and checks idempotency by running it twice. Standard practice for any role you ship publicly.

## Common Confusions

These are the questions every beginner asks. The answers are below.

### "Should I use shell or command module?"

Neither, if you can avoid it. Use a real module: `apt`, `service`, `file`, `copy`, `template`, `lineinfile`, `git`, `user`, `cron`. The real modules are idempotent. `shell:` and `command:` are not.

When you really must use them, add `creates:`, `removes:`, `changed_when:`, or `failed_when:` so they behave correctly. They are last resorts, not first choices.

### "What's the difference between ad-hoc and playbook?"

Ad-hoc is a single command: `ansible web -m apt -a 'name=curl state=present'`. One target group, one module, one set of args, run once. No YAML.

Playbook is a YAML file with multiple plays and many tasks: `ansible-playbook playbook.yml`. Run repeatedly, organized, version-controlled, idempotent.

Use ad-hoc for one-off "do this everywhere right now." Use playbooks for everything you ever expect to do twice.

### "Why does my task always show 'changed'?"

You're probably using `command:` or `shell:` instead of a real module. Or you're using a real module but in a way that doesn't compare to the current state. Check `-vvv` output to see what's happening. Switch to a proper module if you can; add `changed_when: false` if it's a read-only check; or use `creates:`/`removes:` for conditional execution.

### "Why is my variable not interpolating?"

Some possibilities:
1. Wrong precedence: another variable with the same name is overriding yours. Check `-vvv`.
2. The variable doesn't exist where you think. Run with `debug:` to print it.
3. You used `{{var}}` in a string starting with the variable, which YAML mistakes for a dictionary. Wrap in quotes: `"{{var}}"`.
4. Typo. Always look for typos first.

### "What's the difference between Ansible and Salt/Puppet/Chef?"

- **Ansible** — push, agentless, YAML.
- **Salt** — push or pull, agent-based by default (also has `salt-ssh` for agentless), YAML.
- **Puppet** — pull, agent-based, Puppet DSL.
- **Chef** — pull, agent-based, Ruby.

Ansible is the easiest to start with because there's nothing to install on the targets. Salt is faster at very large scale (5000+ nodes). Puppet and Chef are entrenched in older enterprises but have steeper learning curves.

### "Should I use roles or collections?"

Both. Roles are the unit of reusable task bundles. Collections are the unit of distribution and packaging — a collection contains roles, modules, plugins, and docs.

Modern advice: write roles, package them inside a collection, distribute via Galaxy. For your private projects, just use roles directly. For anything you publish, ship as a collection.

### "Why does my become fail?"

Three causes:
1. Sudo wants a password and you didn't pass `--ask-become-pass` or set up passwordless sudo.
2. Your user isn't in the sudoers file at all.
3. You set `become_method` to something not installed (e.g., `doas` on a Linux box that doesn't have it).

Test by SSH'ing in manually and trying `sudo whoami`. If that doesn't work without prompting, fix sudoers first.

### "What's the difference between vars and defaults in a role?"

`defaults/main.yml` has very low priority. Anyone using your role can easily override these.

`vars/main.yml` has higher priority. Hard to override. Use for internal constants you don't want users messing with.

Rule of thumb: anything a user might want to tweak (hostnames, ports, file paths) goes in `defaults/`. Anything that's truly internal (e.g., the path to a config file the role itself created) goes in `vars/`.

### "What's a fact?"

A fact is a variable that Ansible discovers about the target by running the `setup` module. Things like `ansible_distribution` (the OS name), `ansible_memtotal_mb` (RAM in MB), `ansible_default_ipv4.address` (the IP). They're all available in your tasks as `{{ ansible_X }}`. By default `gather_facts: yes` runs setup at the start of every play.

### "Why is fact gathering slow?"

Setup queries hundreds of system properties. On a slow box or many hosts, it can add seconds. If you don't need facts, set `gather_facts: no` to skip it.

### "What's a handler vs a task?"

A handler is a task that only runs when notified. A regular task runs every time the playbook runs. Handlers run at the end of the play, deduplicated. Use handlers for "restart this service if its config changed."

### "Should I use `with_items` or `loop`?"

`loop:` is the modern style (Ansible 2.5+). `with_items:` and the rest of the `with_*` family are legacy but still work. Always prefer `loop:`. They're functionally identical for simple lists.

### "Why does my playbook hang forever?"

Probably one of:
1. SSH is prompting for a password or key passphrase. Run `-vvv` to see.
2. A task is genuinely taking a long time (large package install, big git clone).
3. The target is hung. Check it directly via SSH.
4. Forks limit too low — Ansible is queueing hosts. Increase `forks=50` in `ansible.cfg`.

### "What's the inventory_hostname?"

It's a magic variable that holds the name of the current host as it appears in the inventory. So `{{ inventory_hostname }}` would be `web1.example.com` when running on web1.

## Vocabulary

70+ terms you will hear constantly. Quick definitions for each.

- **ansible** — the umbrella name and also the CLI for ad-hoc commands.
- **ansible-playbook** — the CLI to run YAML playbooks.
- **ansible-galaxy** — the CLI to install roles and collections from Galaxy.
- **ansible-vault** — the CLI to encrypt/decrypt secrets.
- **ansible-doc** — the CLI to read module docs.
- **ansible-inventory** — the CLI to inspect your inventory.
- **ansible-config** — the CLI to inspect Ansible config.
- **ansible-lint** — third-party linter for playbooks.
- **molecule** — third-party testing framework for roles.
- **inventory** — the list of target hosts. INI, YAML, or dynamic plugin.
- **hosts.ini** — common filename for an INI-format inventory.
- **group** — a named bundle of hosts in the inventory.
- **host** — a single target machine.
- **group_vars** — a directory of YAML files; each one defines variables for a specific group.
- **host_vars** — same but for a specific host.
- **all** — built-in group containing every host in the inventory.
- **ungrouped** — built-in group containing hosts not in any custom group.
- **playbook** — a YAML file containing one or more plays.
- **play** — one section of a playbook: a target plus a list of tasks.
- **task** — one step in a play: a module + args.
- **handler** — a task that runs only when notified, at the end of the play.
- **role** — a packaged set of tasks/handlers/templates/defaults for reuse.
- **collection** — modern packaging unit. Bundles roles, modules, plugins, docs.
- **namespace** — the part of a collection name before the dot. `community.general` has namespace `community`.
- **module** — the atomic action; a Python program that does one thing on the target.
- **action plugin** — runs on the controller, sometimes in front of a module. Most modules don't need a custom action plugin.
- **callback plugin** — receives events during playbook execution. Used for custom output formats and notifications.
- **connection plugin** — handles how Ansible connects to the target (ssh, local, docker, kubectl).
- **lookup plugin** — fetches data from external sources at parse time. `{{ lookup('file', '/etc/hostname') }}`.
- **filter plugin** — Jinja2 filter, transforms a value. `{{ list | join(',') }}`.
- **test plugin** — Jinja2 test, returns true/false. `{{ var is defined }}`.
- **strategy** — how plays are executed across hosts. `linear` (default), `free`, `debug`.
- **forks** — how many hosts Ansible runs tasks on in parallel. Default 5.
- **serial** — limit a play to N hosts at a time, in batches. Useful for rolling deploys.
- **async** — run a task in the background on the target.
- **poll** — how often to check on an async task.
- **fact** — a piece of data about the target, gathered by `setup`.
- **gather_facts** — whether to run `setup` at start of play. Default yes.
- **ansible_facts** — the namespace where facts live in variables.
- **register** — capture a task's result into a variable.
- **when** — conditional execution. Skip task if expression is false.
- **loop** — iterate a task over a list.
- **with_items** — legacy version of `loop:`. Still works.
- **tags** — labels you put on tasks. Allows `--tags` and `--skip-tags`.
- **--tags** — CLI flag, run only tasks with these tags.
- **--skip-tags** — CLI flag, skip tasks with these tags.
- **--check** — CLI flag, dry run mode.
- **--diff** — CLI flag, show file diffs.
- **--limit** — CLI flag, run only on subset of inventory.
- **become** — escalate privileges (sudo). Default off.
- **become_user** — which user to become. Default root.
- **become_method** — how to escalate. Default sudo.
- **sudo** — the standard Linux privilege escalation tool.
- **doas** — OpenBSD's sudo replacement.
- **su** — switch user, lower-level.
- **runas** — Windows privilege escalation.
- **vars** — block in a play that defines variables.
- **vars_files** — list of YAML files to load as variables.
- **var precedence** — the rules for which variable wins when multiple are defined.
- **defaults** — role's default variables (lowest priority).
- **set_fact** — task that sets a variable mid-playbook.
- **hostvars** — magic variable, dict of all variables for all hosts.
- **groupvars** — older spelling, less common.
- **magic variables** — variables Ansible always provides: `inventory_hostname`, `play_hosts`, `groups`, `hostvars`.
- **inventory_hostname** — magic variable, the name of the current host.
- **ansible_host** — variable per-host, the actual hostname/IP to ssh to.
- **ansible_user** — variable per-host, the SSH username.
- **ansible_port** — variable per-host, the SSH port.
- **ansible_ssh_private_key_file** — variable per-host, the SSH key path.
- **ansible_python_interpreter** — variable per-host, the Python on the target.
- **Jinja2** — the templating language used everywhere in Ansible.
- **template module** — copies a Jinja2 template to the remote, rendered.
- **copy module** — copies a static file to the remote.
- **file module** — manage file/directory state, perms, owner, mode.
- **user module** — create/delete users.
- **group module** — create/delete groups.
- **lineinfile** — make sure one specific line is in a file.
- **blockinfile** — make sure a block of lines is in a file.
- **replace** — search-and-replace inside a file.
- **debug module** — print a message during a run.
- **fail module** — abort the playbook with an error.
- **assert module** — fail if a condition is false.
- **pause** — pause for N seconds or until enter is pressed.
- **wait_for** — wait for a port to open, file to exist, etc.
- **uri** — make HTTP requests.
- **get_url** — download a file from a URL.
- **git module** — clone/update a git repo.
- **package module** — generic, picks `apt`/`yum`/etc. based on OS.
- **service module** — manage System V / systemd services.
- **systemd module** — explicitly systemd, with more features.
- **cron module** — manage cron jobs.
- **mount module** — mount/unmount filesystems and edit `/etc/fstab`.
- **sysctl module** — manage kernel parameters.
- **hostname module** — set the system hostname.
- **dnf module** — Fedora/RHEL 8+ package manager.
- **apt module** — Debian/Ubuntu package manager.
- **yum module** — RHEL/CentOS 7 package manager.
- **brew module** — macOS Homebrew.
- **pip module** — Python packages.
- **npm module** — Node packages.
- **gem module** — Ruby gems.
- **win_*** modules — Windows-specific modules. Many.
- **vault** — Ansible's encryption-at-rest for secrets.
- **vault password file** — file containing the vault password, used in CI.
- **vault id** — labels for multiple vaults with different passwords.
- **encrypt** — `ansible-vault encrypt FILE` — turn a plain file into a vault.
- **decrypt** — turn a vault back into plain.
- **view** — see contents of a vault without modifying.
- **rekey** — change the password on a vault.
- **encrypt_string** — encrypt a single string for embedding in plain YAML.
- **vault label** — name attached to a vault for `--vault-id`.

## Try This

Real experiments to do today. Each one teaches you something.

### Experiment 1: Spin up two local VMs

Use `multipass` (Ubuntu's lightweight VM tool) or `vagrant` to make two VMs. With multipass:

```
$ multipass launch --name vm1 24.04
$ multipass launch --name vm2 24.04
$ multipass list
Name    State    IPv4         Image
vm1     Running  10.18.5.20   Ubuntu 24.04 LTS
vm2     Running  10.18.5.21   Ubuntu 24.04 LTS
```

Now you have two real Linux machines on your laptop with IP addresses. Add them to a `hosts.ini`:

```
[web]
vm1 ansible_host=10.18.5.20 ansible_user=ubuntu
vm2 ansible_host=10.18.5.21 ansible_user=ubuntu
```

Set up SSH keys: `multipass transfer ~/.ssh/id_ed25519.pub vm1:/home/ubuntu/.ssh/authorized_keys`. Same for vm2.

Test:

```
$ ansible all -i hosts.ini -m ping
vm1 | SUCCESS => {"ping": "pong"}
vm2 | SUCCESS => {"ping": "pong"}
```

You now have a two-node Ansible lab. Keep these around. They are your sandbox.

### Experiment 2: Write the hello-world nginx playbook

Save as `nginx.yml`:

```yaml
- hosts: web
  become: yes
  tasks:
    - name: install nginx
      apt:
        name: nginx
        state: present
        update_cache: yes

    - name: start nginx
      service:
        name: nginx
        state: started
        enabled: yes
```

Run:

```
$ ansible-playbook -i hosts.ini nginx.yml
```

Verify:

```
$ curl http://10.18.5.20
<!DOCTYPE html>
<html>
<head><title>Welcome to nginx!</title></head>
...
```

You just installed nginx on two machines from one command. Pinch yourself.

### Experiment 3: Run it again and see idempotency

```
$ ansible-playbook -i hosts.ini nginx.yml
```

The PLAY RECAP at the bottom should say `changed=0`. Nothing changed because nothing needed to. Welcome to idempotency.

### Experiment 4: Try check mode

Add a third task to your playbook:

```yaml
    - name: drop a note
      copy:
        content: "Hello from Ansible\n"
        dest: /tmp/ansible-note.txt
```

Run with `--check`:

```
$ ansible-playbook -i hosts.ini nginx.yml --check
```

Notice it says `changed: [vm1]` for the new task, but if you SSH into vm1 the file doesn't exist. That's check mode. Now run for real:

```
$ ansible-playbook -i hosts.ini nginx.yml
$ ssh ubuntu@10.18.5.20 cat /tmp/ansible-note.txt
Hello from Ansible
```

### Experiment 5: Use `register:` and `debug:`

```yaml
- hosts: web
  tasks:
    - name: get hostname
      command: hostname
      register: hostname_result

    - name: print hostname
      debug:
        msg: "Hostname is {{ hostname_result.stdout }}"
```

Run:

```
$ ansible-playbook -i hosts.ini debug.yml
TASK [print hostname]
ok: [vm1] => {
    "msg": "Hostname is vm1"
}
ok: [vm2] => {
    "msg": "Hostname is vm2"
}
```

You captured the output of a command and printed it. This is how you debug variables and chain tasks.

### Experiment 6: Use a template

`templates/welcome.txt.j2`:

```
Hello, {{ inventory_hostname }}!
You are running {{ ansible_distribution }} {{ ansible_distribution_version }}
You have {{ ansible_memtotal_mb }} MB of RAM.
```

Task:

```yaml
- name: drop welcome file
  template:
    src: templates/welcome.txt.j2
    dest: /tmp/welcome.txt
```

Run, then check both VMs:

```
$ ssh ubuntu@10.18.5.20 cat /tmp/welcome.txt
Hello, vm1!
You are running Ubuntu 24.04
You have 1024 MB of RAM.
```

Different file on each VM, all from one template. That's the magic.

### Experiment 7: Add a handler

```yaml
- hosts: web
  become: yes
  tasks:
    - name: drop nginx config
      copy:
        content: |
          server {
              listen 80;
              server_name _;
              root /var/www/html;
              index index.html;
          }
        dest: /etc/nginx/sites-enabled/default
      notify: restart nginx

  handlers:
    - name: restart nginx
      service:
        name: nginx
        state: restarted
```

First run: changes config, restarts nginx. Second run: no change, no restart.

### Experiment 8: Convert to a role

Make a role:

```
$ mkdir -p roles/nginx/tasks roles/nginx/handlers
$ mv (your tasks) into roles/nginx/tasks/main.yml
$ mv (your handlers) into roles/nginx/handlers/main.yml
```

Top-level playbook:

```yaml
- hosts: web
  become: yes
  roles:
    - nginx
```

Same behavior, cleaner structure. Now you can drop this role into any other project.

### Experiment 9: Encrypt a secret

```
$ ansible-vault encrypt vars/secrets.yml
$ cat vars/secrets.yml
$ANSIBLE_VAULT;1.1;AES256
65613664303932...
```

Reference it in a playbook:

```yaml
- hosts: web
  vars_files:
    - vars/secrets.yml
  tasks:
    - debug:
        msg: "DB password is {{ db_password }}"
```

Run:

```
$ ansible-playbook -i hosts.ini secret.yml --ask-vault-pass
```

You just used an encrypted variable.

### Experiment 10: Watch what happens with -vvv

```
$ ansible-playbook -i hosts.ini nginx.yml -vvv
```

Look at the avalanche of output. You can see the SSH command Ansible runs, the temp directory, the Python interpreter, the JSON sent to and received from each module. When something goes wrong, this is your first debugging tool.

## Where to Go Next

Once this sheet feels easy, dive into the real reference material. Stay in the terminal:

- **`cs config-mgmt ansible`** — dense reference. Every module, every flag, every pattern.
- **`cs detail config-mgmt/ansible`** — execution model, fact gathering math, performance theory.
- **`cs config-mgmt puppet`**, **`cs config-mgmt chef`**, **`cs config-mgmt salt`** — the alternatives.
- **`cs config-mgmt nornir`** — Python-native network automation.
- **`cs config-mgmt napalm`** — multi-vendor network module library.
- **`cs config-mgmt cisco-nso`** — service orchestration.
- **`cs config-mgmt dc-automation`** — datacenter-scale config management patterns.
- **`cs config-mgmt eem`** — Cisco IOS Embedded Event Manager.
- **`cs ramp-up linux-kernel-eli5`** — what the targets are running underneath.
- **`cs ramp-up tcp-eli5`** — how SSH gets to the targets in the first place.

## See Also

- `config-mgmt/ansible` — engineer-grade Ansible reference.
- `config-mgmt/puppet` — Puppet, agent-based pull config management.
- `config-mgmt/chef` — Chef, Ruby-based pull config management.
- `config-mgmt/salt` — SaltStack, fast push/pull config management.
- `config-mgmt/nornir` — Python-first network automation framework.
- `config-mgmt/napalm` — multi-vendor network library.
- `config-mgmt/cisco-nso` — Cisco service orchestration.
- `config-mgmt/dc-automation` — datacenter-scale config management.
- `config-mgmt/eem` — IOS Embedded Event Manager.
- `iac/terragrunt` — Terraform wrapper for DRY infrastructure code.
- `iac/pulumi` — IaC in real programming languages.
- `iac/crossplane` — Kubernetes-native control plane for cloud resources.
- `ramp-up/linux-kernel-eli5` — what the target servers are actually running.
- `ramp-up/tcp-eli5` — how SSH connections get from your laptop to the targets.
- `ramp-up/terraform-eli5` — provisioning the infrastructure you'll then configure.

## References

- **docs.ansible.com** — official documentation. Modules, plugins, plugins, plugins.
- **"Ansible: Up and Running"** by Lorin Hochstein and René Moser — the classic intro book. Read after this sheet feels easy.
- **"Ansible for DevOps"** by Jeff Geerling — practical recipes from a power user. Excellent companion volume. Jeff also runs the geerlingguy roles you'll keep installing.
- **github.com/ansible-collections** — the official collections, all open source.
- **`man ansible`** — `man ansible` in your terminal. Never leave the terminal.
- **`man ansible-playbook`** — same idea for playbook runs.
- **galaxy.ansible.com** — community roles and collections.
- **Jinja2 docs** — `jinja.palletsprojects.com` — the templating language behind everything.
- **`ansible-doc <module>`** — module docs in your terminal.
- **`ansible-doc -l`** — list every module on your system.
- **`ansible-doc -t lookup -l`** — list every lookup plugin (and other plugin types).

— End of ELI5 —

When this sheet feels easy, type `cs config-mgmt ansible` for the dense reference. After that, `cs detail config-mgmt/ansible` for the execution-model math. By the time you've worked through both, you'll be writing playbooks for real production fleets without flinching.

### One last thing before you go

Pick one experiment from the Try This section that you have not done yet. Do it right now. Spin up the VMs. Write the YAML. Run the playbook. Watch nginx come up on two machines from one command. The first time you do that, something clicks. Reading is good. Doing is better.

Welcome to Ansible. The fleet is yours to command. Have fun.

— End of ELI5 — (really this time!)
