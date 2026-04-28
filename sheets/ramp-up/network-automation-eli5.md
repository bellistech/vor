# Network Automation (ELI5)

> From SSH-and-pray to YANG-and-pipeline — the thirty-year journey to making network configs into code.

## The Big Idea

Imagine you're a sysadmin in 1998. You've got a stack of Cisco routers humming away in a closet, and the boss says, "We need to add a new VLAN to all forty-seven branch offices." What do you do? You sit down at your terminal, you `telnet` into the first router, you type your username, you type the enable password (which is the same on every device because nobody wants to remember forty-seven passwords), and you tap-tap-tap your way through twelve commands. Then you do it forty-six more times. By router twenty, your fingers are tired and you're getting sloppy. By router thirty-five, you've fat-fingered a subnet mask. By router forty-seven, you're ready to throw your keyboard across the room.

Now imagine your buddy down the hall — the web developer. Her workflow in 1998 wasn't great either: edit a PHP file, FTP it up to the server, hope nothing broke, log onto the server to fix it when it did. Fast-forward twenty-five years. Today, your buddy types `git push`, a Continuous Integration (CI) pipeline kicks off, runs tests, builds a Docker image, deploys it to Kubernetes, and rolls back automatically if response times spike. Her job got better. Hers got way better.

What about you, the network engineer? Until shockingly recently — like, until 2018-ish for many shops — you were still telnetting (or at best SSH-ing) into routers and copy-pasting commands from a runbook into a terminal. Network engineering was the last bastion of point-and-click — except the "point" was a CLI prompt and the "click" was a `?` to see what command came next. Network engineers were the absolute last specialty in tech to get the DevOps treatment, and that's the story this sheet is here to tell.

### Why So Late to the Party?

Three reasons, mostly, and they all interlock:

**Reason one: vendors loved their CLIs.** Cisco had IOS. Juniper had Junos. Arista had EOS. Each one was a precious snowflake, with its own command syntax, its own quirks, its own way of saying "set the interface IP address." On Cisco it's `ip address 10.0.0.1 255.255.255.0`. On Juniper it's `set interfaces ge-0/0/0 unit 0 family inet address 10.0.0.1/24`. On Arista it looks Cisco-ish but isn't quite. The CLI was the product, and the vendors had no incentive to give you a programmatic alternative — because as long as you had to memorize their CLI, you were locked in. Imagine if Microsoft Word saved files in a format only Word could read, AND if you switched to Google Docs you had to relearn how to bold text. That was networking.

**Reason two: networks are scary.** Web apps fall over and you reload the page. Networks fall over and the entire company can't get to email, the office VOIP phones go dead, ATM machines stop working, factory robots freeze mid-weld. Network engineers became extremely conservative because the blast radius of a mistake is enormous. "If it ain't broke, don't touch it" became a deeply ingrained cultural norm. Automation felt risky — what if the script does the wrong thing on all forty-seven routers at once? Better to do it by hand, slowly, with a senior engineer watching.

**Reason three: the protocols weren't there.** Even when you wanted to automate, the tools were terrible. You could screen-scrape a CLI session with `expect` (a 1990s tool from Don Libes, named because it expects certain text and then sends a response), but parsing the output was a horror show — vendor CLIs print pretty tables for humans, not structured data for machines. You'd write a regex to grab the IP from `show ip interface brief`, the next IOS release would change the column widths by one space, and your regex would silently break. The world needed a structured protocol. NETCONF was published in 2006 as RFC 4741, but it took another decade for vendors to actually implement it well. gNMI didn't show up until 2017.

### The Snowflake Problem

In a typical pre-automation network, no two devices were exactly alike. Router-23 had been hand-tweaked by Bob in 2009 to fix a weird routing issue. Router-12 had a non-standard SNMP community because the monitoring team was migrating tools and never finished. Router-44 had three extra ACL lines that nobody remembered why. The "running config" — the live configuration in memory — drifted away from the "intended config" — what the documentation said it should be — because every troubleshooting session left fingerprints behind.

We call this **configuration drift**, and it's the silent killer of large networks. You think you have one network. You actually have N networks, where N is the number of devices, each subtly unique. When something goes wrong, you can't reason about "the network" anymore — you have to reason about each special case. Onboarding a new junior engineer becomes impossible because the tribal knowledge of "oh yeah, Router-23 is weird, don't touch the BGP timers there" lives only in the heads of the senior team. When senior Bob retires, his Router-23 quirks become time bombs.

The web-dev world had this problem too. They called it "works on my machine." They solved it with containers, immutable infrastructure, and "cattle not pets" philosophy — every server is interchangeable, no server is special, you can blow one away and rebuild it from declarative source-of-truth. Network automation is, in many ways, just network engineers finally catching up to that philosophy. Treat your routers like cattle. Configuration is code. Source of truth lives in git, not in the device.

### The Fat-Finger Outage Hall of Fame

Why does this matter? Because the alternative — humans typing commands into production routers — has historically been catastrophic. Let's tour a few cautionary tales.

**Class: BGP route leak from misconfigured filter.** This category of outage has happened so many times it has its own Wikipedia category. The pattern: an ISP or enterprise accidentally re-advertises BGP (Border Gateway Protocol — how the internet's networks discover each other) routes they should have kept inside their network. Suddenly the global internet thinks the best path to Google is through Pakistan Telecom or Indosat or some random company in Pennsylvania. Traffic blackholes. Half the internet goes dark for an hour. The fix is almost always "we forgot to apply our outbound BGP filter on this one peering session." A configuration template, applied automatically and validated automatically, would have prevented every single one of these.

**Class: typo'd ACL drops legitimate traffic.** Engineer is editing an Access Control List (ACL — a list of allow/deny rules for traffic). Means to type `permit tcp any any eq 443`. Types `deny tcp any any eq 443`. Hits enter. The router happily applies the new rule. Now nothing can reach HTTPS through this device. Every customer's website is dead. The engineer panics, can't remember the previous state, and frantically tries to undo. Twenty-three minutes of outage. Customers tweet angrily. Stock price dips. With automation: the change goes through code review, a linter catches `deny tcp any any` as suspicious, a pre-merge validator runs the change against a network simulator and notices that production traffic would be blocked, and the change never lands. With a `commit-confirmed` workflow (where the change auto-rolls back if you don't reconfirm within a timeout), even if it did land, it would auto-undo in two minutes.

**Class: spanning tree loop from a single misconfigured port.** Layer 2 (Ethernet) networks are plagued by loops — if a frame can travel in a circle forever, the network melts. Spanning Tree Protocol (STP) prevents this by automatically blocking redundant paths. But if an engineer mis-configures a port (say, by setting it to "trunk all VLANs" when they meant just one, or by disabling STP on a port that connects to another switch), they can punch a hole in STP's logic and create a forwarding loop. The result: broadcast storm, every switch's CPU pegs at 100%, the entire LAN dies. Recovery requires physically unplugging cables until you find the offender. With automation: a templated port profile ensures every access port has the same STP guard settings (BPDU Guard, Root Guard, Loop Guard) — no port is special, no port can punch through STP's defenses.

**Class: Comcast 2014 — DNS misconfiguration.** Without naming and shaming the engineer who pushed it, a major ISP had a DNS-related configuration push that took out a chunk of their service for hours. The class of failure: a change made manually to one cluster of devices was supposed to be propagated to all clusters; somebody forgot a cluster; the asymmetry caused cascading failures. With automation: the source of truth says "all clusters get this config," the automation engine pushes to all clusters, drift detection alerts if any cluster diverges. The class of "I forgot one" goes away.

**Class: AWS S3 outage 2017.** This isn't networking exactly, but it's the same pattern. An engineer was running a runbook to take a small number of servers offline for debugging. They mistyped a parameter and took down a much larger fleet. Half the internet went dark for four hours because so many websites depended on S3. The post-mortem said: we're going to add safeguards so that runbooks can't take down more than X% of capacity. That safeguard is exactly the kind of thing automation enables — humans can't refuse to type, but automation can refuse to execute commands that would exceed a blast-radius limit.

**Class: BGP hijack that wasn't really a hijack — Pakistan vs YouTube, 2008.** Pakistan's government wanted to block YouTube domestically. An engineer at Pakistan Telecom configured a BGP route to null-route YouTube's prefix internally. They forgot to apply the "internal only" filter. The route leaked to the global internet, and for about two hours, much of the world's YouTube traffic got sucked into Pakistan and dropped. With automation: the BGP filter is part of every peering session template, applied automatically, validated by a pre-merge check that says "this configuration is going to advertise routes externally that should stay internal."

These aren't rare. The internet has BGP route leaks of varying severity multiple times per week. Most are small and brief. Some take down major services. The common thread: a human typed a thing on a router, and there was no system between the human and the production network that could say "wait, this looks wrong."

**Class: Slack 2021 — provisioning-system overload.** Slack's network team was rolling out new capacity. The provisioning system pushed config to the new devices, but the existing devices in the same Layer 2 domain were caught flat-footed by a sudden wave of new MAC addresses, ARP requests, and route advertisements. Cascading control-plane meltdown. Hours of downtime for a service that millions of people use as their workplace nervous system. With automation: canary deployments would have lit up one device first, monitored its peers' control-plane CPU, and aborted before the wave hit. Blast-radius limits at the orchestration layer.

**Class: Cloudflare 2020 — backbone misconfiguration.** Cloudflare's own backbone took a hit from a router config push that interacted badly with their Atlanta hub. Traffic across multiple regions degraded for about half an hour. Cloudflare's own post-mortem (which they publish openly — read them, they're a master class) noted that the safeguards they had in place caught it within minutes. Without those safeguards, the same change would have lasted hours. This is a bright spot in the genre: companies with mature automation hit the same kinds of bugs, but the recovery is minutes instead of hours, and post-mortems trigger more safeguards rather than blame.

**Class: Microsoft Azure 2023 — DNS configuration cascade.** A planned operation affected DNS infrastructure in ways the operators hadn't anticipated. Azure's DNS chain has so many dependencies that a single weak link cascaded. The lesson, for the Nth time: DNS is networking, DNS is the hardest part of networking to get right, and the "blast radius" of a DNS misstep is the entire customer base. Automation around DNS — health checks on resolvers, auto-rollback if query success rate drops, regional canary — is the only sane way to operate at scale.

**Class: Akamai 2021 — DNS edge platform.** A configuration update on Akamai's edge DNS broke resolution for thousands of customer-facing properties, including major airlines and banks. The hour-long outage made the news. The technical class: a change to a configuration management system pushed an unintended state to the live fleet because a validation step was bypassed. Automation lesson: the validation step should not be skippable; the system itself should refuse to push unvalidated configs.

**Class: BGPlay-worthy hijack — Rostelecom, 2017.** A Russian state-affiliated ISP began advertising routes for some 80+ prefixes belonging to major financial institutions. Whether intentional or accidental, the result was that traffic destined for major banks transited through Rostelecom for several hours. With automation and RPKI (Resource Public Key Infrastructure — cryptographically signed BGP origin assertions), the global routing system can refuse to accept origin announcements that aren't authorized. RPKI rollout has been driven, in part, by automation tooling that makes it tractable to publish and validate ROAs (Route Origin Authorizations) at scale.

**The pattern across all of these:** a human action — usually well-intentioned, often by a senior engineer who'd done the same thing dozens of times — interacted with a hidden assumption in the network and produced an outsized failure. The automation pipeline isn't there to replace the human; it's there to add a thinking-time pause and a battery of mechanical checks that catch the assumptions humans miss when they're tired, in a hurry, or just having a bad day.

### The Renaissance: 2014–2018

Around 2014, a few things converged. Linux containers were finally usable (Docker shipped 2013). Configuration management tools (Puppet, Chef, Ansible) were mature in the server world. Public cloud was forcing networks to scale in ways that hand-configured snowflakes couldn't keep up with. SDN (Software Defined Networking) — the idea that the network's control plane should be programmable — had been bubbling for a few years (OpenFlow, 2008). And vendors were finally — finally — starting to expose modern APIs.

The breakthrough was a kind of two-pronged movement. From the top down, hyperscalers (Google, Facebook, Microsoft) had built their own internal automation because they had to — running a million-host data center by hand is impossible. They started open-sourcing pieces of their tooling, and the protocols they pushed (OpenConfig, gNMI) gained traction. From the bottom up, network engineers at smaller shops discovered Ansible (originally a server tool) had network modules, and you could template a Cisco config with Jinja2 (a templating language from the Python web world) just like you templated a web server config. Suddenly your routers were code.

The 2017–2018 era saw a flood of new tools: Nornir (a Python automation framework for networks), NAPALM (a vendor-agnostic library), NetBox (a "source of truth" database for network state), Batfish (offline network simulation for pre-merge validation), gnmic (a CLI for streaming telemetry), Containerlab (run virtual networks on your laptop). The momentum became unstoppable. Even traditionally CLI-only certifications started adding automation modules — Cisco's CCNP and CCIE now have entire automation tracks.

### The Web-Dev Parallel

A useful mental model: every step web developers took, network engineers took the same step ten to fifteen years later. FTP-the-file-up became `git push`. Hand-edited Apache configs became Puppet/Chef/Ansible. SSH-into-prod-to-tail-logs became centralized logging with Splunk and ELK and Loki. Cron jobs became Kubernetes CronJobs. SQL-injection-disasters became prepared-statement standards.

For networking: telnet-and-paste became SSH-and-Ansible. Hand-tracked spreadsheets of IPs became NetBox and IPAM (IP Address Management) databases. SNMP polling every five minutes became gNMI streaming sub-second telemetry. Debugging by `show running-config | grep` became querying a graph database that knows every neighbor and link. The destination is the same — declarative source of truth in git, automated reconciliation, observability, fast rollback, no humans typing into prod. We're just behind on the journey.

If you internalize one thing from this section, let it be: **network automation isn't about saving keystrokes.** It's about removing humans from the path between intent and configuration. Humans should describe what they want (intent). Software should figure out how to make it happen (declarative reconciliation). Software should monitor for drift, validate changes, roll back when things break, and free humans to do the engineering work that requires judgment.

That's the thirty-year arc, and we're roughly twenty years into it. The remaining ten years are going to be spectacular — and that's why now is exactly the right time to learn this stuff.

### What "Network as Code" Actually Means

Let's pin down a phrase you'll hear a lot: "network as code." It does NOT mean "we wrote a Python script that SSHes into routers." That's automation, but it's not "as code" in any deep sense. "Network as code" means **the desired state of the entire network is described in a set of source-controlled files, and the live network is continuously reconciled against those files.**

Concretely, the test of whether you're doing it is this: if your office burned down with all your network gear in it, could you, given a fresh delivery of identical hardware, rebuild the network exactly — same VLANs, same IP plan, same routing policy, same ACLs, same QoS — by running your automation against the source-controlled config? If yes, you have network as code. If you'd have to consult Bob, or read the running configs off salvaged backups, or reconstruct anything from memory, you don't yet.

That's a high bar, and it's deliberately uncompromising. Most shops don't fully meet it. Most shops have parts of the network captured in code and parts that live only on the devices. The journey from "some" to "all" is the multi-year project that "network as code" implies. It's worth the climb because the property you get at the top — total reproducibility — eliminates whole classes of disaster scenarios. Hardware fails? Replace and re-provision. Junior accidentally wipes a switch? Re-provision in three minutes. Vendor announces a critical CVE in a firmware version? Roll the entire fleet to the next version with confidence because the configs are reproducible.

Note that "as code" implies all the practices we know from software: version control (git), code review (pull requests), continuous integration (CI runs tests on PRs), automated tests (Batfish, unit tests on Jinja outputs), continuous deployment (merges trigger pushes), and observability (we know whether the deployment worked). None of these is optional. A "network as code" shop without code review is just a shop that types YAML instead of CLI — same risks, fancier syntax. The full discipline is what delivers the value.

### A Brief Word on Culture

Tools are necessary. Tools are not sufficient. The hardest part of network automation, in every shop that successfully adopts it, is the cultural shift. The senior network engineer who's been hand-configuring routers for twenty years has a deep and legitimate concern: "If a script can do my job, what's my job?" The answer — that the engineer's role evolves from typist to designer, from technician to architect — is true but it doesn't feel that way at first.

Successful adoption pairs the toolchain with explicit role evolution. The engineer who used to push configs becomes the engineer who designs the templates. The engineer who used to read syslog becomes the engineer who builds the alerting rules. The engineer who used to triage tickets becomes the engineer who codifies the triage logic into runbooks-as-code so the next ticket auto-resolves. The work doesn't go away; it moves up the value chain. Shops that fail at automation usually fail because they tried to impose new tooling on a team that wasn't given the time, training, or incentive to grow into the new roles.

The same pattern plays out for management. Network change windows have traditionally been "whatever it takes to get it done before 6 a.m." Automation makes change windows shorter, but they require more upfront investment in code review, simulation, and canary planning. Managers who measure "speed of change" without measuring "incident rate" or "rollback frequency" will incentivize their teams to skip the safeguards. Managers who measure outcomes (reliability, MTTR, change failure rate) get the benefits.

Get this right and your network team becomes a force multiplier — they enable the platform engineers, the SREs, the developers. Get it wrong and you have a YAML version of the same fragile snowflake-network you had before, with a fresh layer of complexity on top.

## Vocabulary

This is a long list. Don't try to memorize it all on first read — skim it, then come back when terms show up later in the sheet. The "Why it matters" column is the key part.

| Term | One-liner | Why it matters |
|------|-----------|----------------|
| NETCONF | XML-based network configuration protocol over SSH (RFC 6241) | The first vendor-agnostic structured config protocol; foundation of modern automation |
| RESTCONF | HTTP/JSON wrapper around NETCONF (RFC 8040) | Lets you do NETCONF-style ops with `curl` instead of XML over SSH |
| YANG | Data modeling language for network configs (RFC 7950) | Defines the *shape* of config — what fields exist, what values are valid, with type checking |
| XML | Tag-based markup; NETCONF's wire format | You'll wade through it any time you debug NETCONF; verbose but unambiguous |
| JSON | Brace-based markup; RESTCONF and gNMI's wire format | Lighter than XML; what you'll work with in modern toolchains |
| gNMI | gRPC-based network management interface (OpenConfig) | The streaming-telemetry protocol; sub-second push instead of SNMP polling |
| gNOI | gRPC Network Operations Interface | Sibling of gNMI for operational tasks (reboot, file transfer, certs) — config-adjacent |
| OpenConfig | Vendor-neutral YANG model collaboration | The "lingua franca" of multivendor networks; same model across Cisco/Juniper/Arista |
| IETF model | YANG models published by the IETF (e.g., `ietf-interfaces`) | Standards-track but slow-moving; OpenConfig moves faster |
| Vendor-native model | Vendor's own YANG (Cisco-IOS-XE-native, junos-config) | Exposes every knob the platform has, but locks you in |
| Source of Truth (SoT) | The authoritative database of intended network state | Single place that says "this is what the network *should* look like" |
| CMDB | Configuration Management Database | Generic IT inventory; pre-NetBox attempt at SoT |
| IPAM | IP Address Management | Tracks which IPs/subnets are allocated where; subset of SoT |
| Idempotent | Same input → same end state, no matter how many times applied | Run the playbook 10 times, get the same network — no double-apply damage |
| Declarative | "I want this end state" (vs. "do these steps") | Describes the *what*, not the *how* — the engine reconciles |
| Imperative | "Do these steps in this order" | The old-school CLI runbook style — fragile if state shifts mid-script |
| Drift | Live config diverges from intended config | The silent killer; automation must detect and reconcile |
| Reconciliation | Closing the gap between intent and actual | The "control loop" of declarative systems |
| Brownfield | Existing network you're retrofitting automation onto | The hard case — must onboard chaos and gradually impose order |
| Greenfield | Brand-new build from scratch | The easy case — bake automation in from day one |
| Intent | Business-level desired outcome | "VLAN 100 reaches every floor" — abstract, not "set interface vlan100 ip..." |
| DSL | Domain-specific language | A mini-language for one job — Jinja2 for templating, NETCONF XPath for queries |
| Controller | Centralized brain that programs the network | Cisco DNA Center, Juniper Apstra, Arista CloudVision; SDN incarnate |
| SDN | Software-Defined Networking | Separates control plane (decisions) from data plane (forwarding); enables central programmability |
| Ansible playbook | YAML file describing automation tasks | The most common entry point for network engineers; agentless via SSH/NETCONF |
| NAPALM | Network Automation and Programmability Abstraction Layer with Multivendor support | Python lib that hides vendor differences behind a common API |
| Nornir | Python automation framework for networks | Like Ansible but pure-Python; faster, more flexible, more code-y |
| Netmiko | SSH library for network devices | The low-level "just send me commands over SSH" Python lib most others build on |
| Jinja2 | Python templating language | How you turn "for each switch, render a config" into actual files |
| Candidate config | Pending config not yet active | NETCONF concept — stage changes in candidate, commit to running |
| Running config | Currently active config in device memory | What the device is actually using right now |
| Startup config | Config loaded at boot | If running ≠ startup, you'll lose changes on reboot |
| Dry-run | Show what *would* happen without doing it | The safety net; always run dry first on prod |
| Commit-confirmed | Apply change with auto-rollback if not reconfirmed | Junos invented this; saves you when you fat-finger a remote change |
| Rollback | Revert to previous known-good config | The emergency button; should be one command, not a 20-step process |
| Pre-merge validation | Check config in CI before it reaches the device | Catches errors when they're cheap to fix (PR comments, not outage) |
| Batfish | Offline network simulator for validation | "What would the routing table look like if I merged this PR?" — without touching prod |
| Containerlab | Tool for running virtual network topologies | Spin up 20-router topologies on your laptop in seconds |
| Suzieq | Multi-vendor network observability platform | "What's the state of every device's ARP table right now?" — queryable network state |
| NetBox | Open-source network SoT (DCIM + IPAM) | The de-facto open source "what should the network look like" database |
| Nautobot | NetBox fork with a richer plugin API | Same idea, different community; both very popular |
| gnmic | Command-line client for gNMI | The `curl` of gNMI — subscribe, set, get from the shell |
| pyang | Python tool to parse and validate YANG | When you need to know "what fields does this YANG model have?" |
| yanglint | C tool for YANG validation | Faster than pyang for big models; CI gate for YANG syntax |
| ncclient | Python NETCONF client library | The lower-level NETCONF lib most Python tooling builds on |
| paramiko | Python SSH client library | The bedrock SSH lib that Netmiko sits on |
| Agentless | No long-running daemon on the device | Ansible's selling point; just need SSH/NETCONF and creds |
| Push telemetry | Device pushes data without being asked | gNMI streaming; sub-second freshness, scales better than polling |
| Pull telemetry | Tool polls device on a schedule | SNMP, REST polling; simpler, but laggy and chatty at scale |
| Sample-interval | How often a streaming subscription emits | "Send me the interface counter every 10 seconds" |
| On-change | Streaming mode that fires only when value changes | Lower bandwidth than periodic; great for state flags (link up/down) |
| ONCE/POLL/STREAM | gNMI subscription modes | One-shot, request-response polling, or persistent streaming |
| JSON_IETF | One of gNMI's encoding formats | Standardized JSON serialization of YANG-modeled data |
| augment | YANG keyword to extend an existing model | How vendors add their proprietary leaves to a standard model |
| deviation | YANG keyword saying "we don't support this part of the model" | Where reality fails to match the standard — important to know |
| leafref | YANG type pointing to another leaf's value | "This interface name must match an existing interface" — referential integrity |
| identityref | YANG type referencing an enumerated identity | Type-safe enums (e.g., interface types: ethernet, loopback, tunnel) |
| gRPC | Google's RPC framework over HTTP/2 | The transport beneath gNMI/gNOI; binary, multiplexed, streaming |
| protobuf | Protocol Buffers — Google's serialization format | The wire format gRPC uses; smaller and faster than JSON |
| Capability exchange | Initial NETCONF negotiation of supported features | Server tells client "I speak these YANG models, these versions" |
| Hello message | First NETCONF message after SSH handshake | Carries capabilities; failing here means model/auth mismatch |
| Closed loop | Automation that observes, decides, acts, re-observes | The end goal — self-healing networks; intent → measure → reconcile → repeat |
| Self-healing | Automation auto-remediates when state drifts from intent | The promise of intent-based networking; rare in practice still |
| Intent-based networking (IBN) | Top-down declarative networking | Cisco/Juniper marketing buzz, but the real thing is closed-loop reconciliation |
| ZTP | Zero Touch Provisioning | New device boots, downloads its config from a server, joins automatically |
| TextFSM | Template language for parsing CLI output into structured data | Pre-NETCONF lifeline; turn `show interface` into a list of dicts |
| ntc-templates | Open library of TextFSM templates for many platforms | The starter kit for screen-scrape parsing |
| pyATS / Genie | Cisco's testing/parsing framework | Heavy but powerful; ships parsers for hundreds of show commands |
| Robot Framework | Keyword-driven test framework popular in network testing | The "test runner" many shops put on top of pyATS |
| Network CI/CD | Continuous integration for network configs | Treat configs like code — branch, PR, test, merge, deploy |
| GitOps | Git is the source of truth; reconciler watches and applies | The cloud-native flavor that's now invading networking |
| Webhook | HTTP callback fired when something changes | How NetBox tells your CI "config changed, go reconcile" |
| OPA | Open Policy Agent — generic policy engine | Used to enforce rules like "no /0 routes" or "all VLANs must have description" |
| Schema validation | Check data conforms to a model | Where YANG shines; reject bad config before it reaches the device |
| Vendor agnostic | Works across Cisco, Juniper, Arista, etc. | The dream; NAPALM and OpenConfig are the closest we have |
| Diff | Difference between two configs | The thing every commit ought to show before applying |
| Three-way diff | Diff among intended, candidate, and running configs | Critical when reconciling brownfield drift |
| Configlet | Small, reusable chunk of config | The "snippet" abstraction in many controller GUIs |
| Service profile | High-level template (e.g., "branch office") that expands to many configlets | Lets non-engineers provision common patterns |
| Day 0 | Initial provisioning | Brand-new box getting its first config (often via ZTP) |
| Day 1 | Initial service turn-up | Config is loaded but services are being lit up (BGP peering, VPN tunnels) |
| Day 2 | Ongoing operations | The 99% of network life — change management, troubleshooting, scaling |
| Day N | Decommissioning | The forgotten lifecycle stage; automation should handle this too |

That's seventy-something terms. You'll see most of them again. If a term confused you on first read, search this table when it shows up later — that's what it's here for.

## Why Automation Exists

To really understand modern network automation, you have to understand what came before. Every tool, every protocol, every weird convention exists in reaction to a specific pain. Let's walk the history.

### The CLI Era (1985–2005)

In the beginning, there was the Command Line Interface, and it was good. Cisco IOS launched in 1986 (sort of — the name came later). It gave network engineers a structured way to type commands and see output. Compared to dipswitches, jumper cables, and serial consoles, the CLI was a revelation. You could `telnet` into a router from anywhere on the network and configure it.

The workflow was: log into the device. Enter privileged mode (`enable`). Enter configuration mode (`configure terminal`). Type commands. Exit. Save (`copy running-config startup-config` or, in slang, `wr mem`). Log out. Repeat for the next device.

Tooling was minimal. The fanciest thing most engineers used was `expect` — a Tcl extension that let you script interactive sessions. You'd write something like:

```tcl
spawn telnet 10.0.0.1
expect "Username:"
send "admin\r"
expect "Password:"
send "secret\r"
expect "#"
send "configure terminal\r"
expect "(config)#"
send "interface Ethernet0/0\r"
send "ip address 10.0.0.1 255.255.255.0\r"
send "no shutdown\r"
send "end\r"
send "wr mem\r"
```

This worked. It was fragile. If the prompt changed, the script broke. If the device was slow to respond, the `expect` timed out. If the device asked an unexpected question (`Are you sure?`), the script would hang or send the wrong answer. Engineers wrote thousand-line `expect` scripts that nobody else could understand, and they'd "work" until the firmware was upgraded and a CLI prompt subtly changed.

There were also vendor-specific helpers: Cisco had **Cisconet** in the 1990s, **CiscoWorks** in the 2000s, eventually **DNA Center** and **Prime**. Each was a graphical wrapper around the CLI — they SSH'd in, sent commands, parsed output. They worked OK for monitoring (read-only, predictable output) and were terrible for change management (write operations on heterogeneous devices were too unpredictable).

The dominant pattern of this era was the **runbook**: a document, usually a Word file or a wiki page, that listed the exact commands you should type to perform a task. New VLAN provisioning had a runbook. Firewall rule additions had a runbook. Troubleshooting had a runbook for each failure class. Senior engineers wrote them; junior engineers executed them. This worked for a while — until your network grew past about 500 devices, at which point the cognitive load of which-runbook-which-device-which-vendor-which-version became unmanageable.

A typical runbook from this era looked like this (real example, lightly anonymized):

```
== Add a new VLAN ==
1. SSH to core-router-1
2. enable
3. configure terminal
4. vlan 100
5. name Marketing-Floor
6. exit
7. interface vlan 100
8. ip address 10.100.0.1 255.255.255.0
9. no shutdown
10. exit
11. end
12. write memory
13. Repeat steps 1-12 on core-router-2 (BUT use 10.100.0.2 in step 8)
14. SSH to access-switch-1 through access-switch-22
15. For each: enable, configure terminal, vlan 100, name Marketing-Floor, exit, end, write memory
16. Verify: ping 10.100.0.1 from a host on the new VLAN
```

The problem with runbooks is that they accumulate. After a year of operations, you have 200 runbooks. Some are out of date. Some reference removed devices. Some have been forked and improved by individual engineers but never merged. New hires don't know which runbook to use. Senior engineers' brains have been turned into runbook indexes — they're the only ones who know that "the OSPF runbook" is actually three different runbooks depending on which firmware family the device is on.

The runbook era's greatest invention was the **change ticket**. Before any production change, file a ticket. Get sign-off from the change advisory board. Schedule the change window. Execute the runbook. Document the outcome. This was a real improvement over cowboy operations — it created an audit trail and forced thinking-time before action — but the ticket process became its own bureaucracy. It's not unusual to find shops where the change ticket takes longer to write than the change itself takes to execute. Automation, done right, lets you keep the auditability of the ticket without the friction of manual execution.

The CLI era's other contribution: **TFTP-based config archival**. Engineers would set up a cron job (or use a tool like RANCID — Really Awesome New Cisco confIg Differ — released ~2002) that would `show running-config` on every device every day, save the output to a file, and commit it to a Subversion or CVS repo. This gave you a poor-man's version history. You could see what changed, who changed it (sort of, if you correlated with login logs), and roll back by hand if needed. RANCID was a key transitional tool — it taught a generation of network engineers that "configs in version control" was both possible and valuable, even if the workflow was retrieval-only and the rollback was manual.

The major weakness of RANCID-style archival: it captures state, not intent. You know what the config IS, but not what it SHOULD be. If your automation goal is "make the configs match what we want," RANCID can't help — there's no "want" file to compare against. That's the gap that NetBox/Nautobot fill: they describe what the network SHOULD look like, and the rendering pipeline turns that intent into a running config. RANCID-as-backup is still useful (every shop should be archiving live configs somewhere), but it's not a source of truth.

### The Snowflake Era (1995–2015, overlapping)

Even with runbooks, every network drifted into snowflake-hood. Why? Because troubleshooting leaves marks.

Picture this: it's 3am. Customer X is having intermittent issues. You SSH into Router-23 and discover that turning off `ip route-cache cef` on one interface "fixes" the problem. You don't know why. You leave it off. You go back to bed. Six months later, somebody else is debugging an unrelated issue on Router-23, sees the unusual config, removes it, and the original problem comes back — but now it manifests differently, and nobody connects the dots.

Multiply this by years. Every router has a graveyard of debug-driven config tweaks. Every senior engineer carries tribal knowledge — "oh, on Router-23, never touch X." When the senior engineer leaves the company, the tribal knowledge leaves too.

The "cattle vs pets" framing from the cloud world maps perfectly here. **Pets** are servers/devices you nurture individually — each has a name, a personality, special handling. **Cattle** are interchangeable units — if one dies you replace it from a template, and any one is identical to the others. Network devices were emphatically pets in the snowflake era. Modernizing means turning them into cattle, which requires that the config be defined in a single template-driven source of truth and any device can be wiped and re-provisioned in minutes.

Drift wasn't always malicious or careless. It often came from legitimate-but-undocumented engineering. Examples:

- **Vendor-recommended workarounds.** A TAC (Technical Assistance Center) case opens, the vendor engineer says "set this hidden command to work around bug CSCxx12345," the engineer applies it, the bug eventually gets fixed in firmware, but the workaround command stays in the config forever because nobody knows it's safe to remove.
- **Performance tuning.** A subset of devices in a high-traffic location got their TCP MSS tuned, their hold-queue raised, their CEF table sized larger. The tuning is correct for those devices and wrong for others. There's no template that captures "these specific devices in this specific role."
- **Customer-specific quirks.** "Customer X uses BFD timers of 50/3" — that's a one-off setting on three peering routers because it was negotiated in a contract years ago. The setting is defensible, but it lives in tribal memory.
- **Failed migrations.** A migration to a new monitoring system was started, half the devices were updated, the project was deprioritized, the work paused. Six months later, half the fleet has the old SNMP config and half has the new — and no documentation explains why.

These aren't bugs in process. They're the natural outputs of running a network for years with humans in the loop. Automation doesn't eliminate the impulse to tune; it captures the tuning in a way that's reviewable, version-controlled, and replayable. The vendor workaround becomes a comment in the template ("# CSC12345 workaround, retest 2026-Q3"). The performance tuning becomes a role-tagged variable in the SoT. The customer quirk becomes a customer-tagged override that's visible in PR review. The failed migration becomes a tracked transition state with an owner.

The deepest snowflake stories I've heard:

- A bank had three core routers, all "identical" in theory. After an automation rollout, the team did a config diff against the templates. The findings: 2,400 lines of one-off config across the three routers. Some were bug workarounds dating to 2008 — for a bug that had been fixed in 2010. Some were performance tuning for a circuit that was decommissioned in 2015. Some were monitoring directives for a tool that was replaced in 2018. The cleanup project to bring them all to a single template took six months and reduced the running config from 8,000 lines to 1,200. The smaller config was easier to reason about, faster to back up, and the device boot time dropped by 30 seconds.

- A telco had 40,000 customer-edge routers. They were "identical" in role but each had been touched by an average of 17 different engineers over the years. When they rolled out automation, the diff from "current state" to "template state" exposed configs from former employees who'd left a decade earlier. Some configs referenced VPN peers that no longer existed. Some had ACL rules whose business owners had retired. The forensic project of "is this still needed?" took 18 months across a team of six engineers and recovered an estimated 8% of router CPU and memory.

- A media company's Wi-Fi infrastructure had been "automated" by a previous regime — a Python script that hand-touched every controller. After three years and four engineering teams, the script had been forked and modified per-region. It took two engineers a full quarter just to consolidate the scripts back to a single canonical version, before they could even start migrating to a real orchestration framework.

These stories are not unusual. They're typical. If you take one operational lesson away from this section: **the best time to start capturing your network in code was ten years ago. The second best time is today.** Drift accumulates. Tribal knowledge fades. The longer you wait, the harder the cleanup gets.

### The Outage Stories

A small museum of cautionary tales — these aren't just "stuff happens," they're failure modes that automation directly addresses.

**The Comcast 2014 incident.** Internal config push went sideways. Class: change applied non-uniformly across regions. Automation lesson: source of truth should drive every region simultaneously; partial deployments are dangerous and should be policy-rejected.

**The 2008 YouTube/Pakistan BGP leak.** Pakistan Telecom internally null-routed YouTube; missing outbound filter leaked the route globally; YouTube went dark for ~2 hours worldwide. Class: missing safeguard on a peering session. Automation lesson: every BGP peering session should have a templated, never-skipped, outbound prefix-filter that's validated at merge-time.

**The 2017 AWS S3 outage.** Engineer running an internal runbook to remove a small number of servers mistyped a parameter, removed too many, and toppled S3 in us-east-1. Class: blast-radius miscalculation. Automation lesson: change tooling must enforce caps on how much capacity any one operation can affect.

**The 2019 CenturyLink 911 outage.** A misconfigured network management card went into a packet-storm cycle that took down voice services across multiple states for 37 hours. People couldn't reach 911. Class: cascading firmware/config interaction. Automation lesson: pre-merge simulation (Batfish) and canary rollouts catch many cascade-failure classes before they reach the second region.

**The 2021 Facebook outage.** A routine maintenance command was issued via internal tooling; a bug in the tool's audit/safeguard layer let the command go through; FB's BGP routes were withdrawn; their entire empire (Facebook, Instagram, WhatsApp) went dark for ~6 hours, AND their physical access systems were partly tied to the same network so engineers couldn't get into the data center to fix it. Class: tooling bug + insufficient safeguards + physical dependency on the same network being broken. Automation lesson: out-of-band management is non-negotiable; automation systems must have hardcoded refuse-to-execute logic for operations that would isolate the company from itself.

**Class: every BGP route leak, ever.** Search the BGPMon archive — there's a leak roughly every week. Most are small. Some are catastrophic. Almost all share the pattern: a peering session was missing an outbound filter or had an overly permissive one. Almost all would have been prevented by a template-driven, validated configuration management pipeline.

Each story is a lesson. Each lesson is a feature in a modern automation pipeline. Drift detection. Pre-merge validation. Canary deployment. Blast-radius limits. Out-of-band management. Templated peering sessions. Commit-confirmed. The automation toolkit didn't appear from thin air — it grew from a graveyard of outages.

### The Renaissance (2014–2018)

Several streams converged.

**Stream one: hyperscalers needed it.** Google, Facebook, Microsoft, Amazon were running networks of unimaginable scale (millions of hosts, hundreds of thousands of network devices). Hand-configuring was impossible. They wrote internal automation. Some of it leaked out. **OpenConfig** (started ~2014) is the most visible artifact — Google convened a working group to define vendor-neutral YANG models for the configs they actually cared about, and it gradually became a de-facto standard. **gNMI** came out of Google around 2017 as a more efficient successor to NETCONF for streaming.

**Stream two: Ansible reached the network world.** Ansible launched 2012 as a server config tool. Around 2015 the network modules landed (originally for Cisco IOS, then Junos, Arista EOS, Cumulus, etc.). Suddenly, network engineers who'd been afraid of Python could write YAML playbooks and template configs with Jinja2. Ansible's "agentless" model (just SSH) meant you didn't have to install anything on the devices, which removed a huge adoption barrier. By 2017, Ansible was the default first step for most teams entering network automation.

**Stream three: Python ecosystem.** **Netmiko** (Kirk Byers, ~2014) gave Python a clean SSH-to-network-device library. **NAPALM** (~2015) layered vendor-agnostic abstractions on top. **Nornir** (~2018) reimagined Ansible as Python code instead of YAML. The Python ecosystem matured to the point where you could string together inventory + connection + parser + diff + apply in 200 lines of code.

**Stream four: Source of Truth tools.** **NetBox** (Jeremy Stretch at DigitalOcean, ~2016) provided an open-source DCIM (Data Center Infrastructure Management) and IPAM tool that became the de-facto SoT for many shops. **Nautobot** forked from NetBox a few years later with a richer plugin model. Suddenly, "where does the truth about the network live?" had a credible open-source answer that wasn't "in spreadsheets" or "in some senior engineer's head."

A pre-NetBox shop typically tracked network state in: a Visio diagram (out of date the day after it was drawn), a few Excel spreadsheets (one for IPs, one for VLANs, one for VRFs), a wiki page or three (usually contradicting the spreadsheets), and the senior engineer's head (the only authoritative source, but un-queryable except by buying him beer). NetBox put all of that into a single relational database with a REST API, a clean GUI, and a webhook system. Suddenly you could ask programmatic questions: "give me every IP allocated to a device in the Marketing VRF" returns a JSON list, not a request to ping Bob.

**Stream five: validation tools.** **Batfish** (2014, then commercial as Intentionet) lets you simulate a network configuration offline and ask questions like "if I push this PR, can host A still reach host B?" **Suzieq** (~2020) does the observability piece — query the live network as a graph database. **Containerlab** (Nokia, ~2020) lets you spin up real virtual routers (Cisco IOL/IOS-XRd, Juniper vMX, Arista cEOS, Nokia SR Linux) on your laptop in containers, in seconds.

The validation tools are the hidden hero of the modern stack. Before Batfish, your "test" for a change was "push it and pray." There was no equivalent of `pytest` for networks. Batfish changed that. You can ingest your full network's configs (anonymized), run queries, and get definitive answers: "after this change, how many BGP sessions will be down?" "Is reachability preserved between every pair of hosts?" "Are any ACLs now unreachable?" These are the kinds of questions that, before Batfish, required a senior engineer to read every changed file and reason about it in their head. Now a CI job can answer them in seconds.

The Containerlab story is similar. Before it (and its predecessor, EVE-NG, and before that, Cisco's GNS3), spinning up a multi-router topology required dedicated lab gear or expensive virtualization. Today, you `docker pull` an Arista cEOS image, write a 30-line YAML topology file, and run `containerlab deploy`. Three minutes later you have a 12-router network running on your laptop, with full BGP, OSPF, IS-IS, whatever you need. You can develop and test automation against it, then point the same automation at production with confidence. The cost of "trying things" went from "schedule a lab next week" to "30 seconds and a coffee."

By 2018 the toolchain was complete enough that a small team could plausibly run a network like a software project: configs in git, peer review on PRs, CI runs Batfish to simulate the change, merge triggers Ansible/Nornir to push to canary devices, telemetry from gNMI confirms health, full rollout proceeds, drift detection runs nightly. This is the modern stack, more or less.

**Stream six: training and certification finally caught up.** The CCNP and CCIE certifications added Python, Ansible, NETCONF/YANG, and RESTful API content. Cisco DevNet — the developer-focused arm of Cisco — launched in 2014 and grew into a substantial community with sandboxes, training, and a dedicated DevNet Associate/Professional/Expert track. Juniper's JNCIE-DevOps and Arista's ACE-A program followed similar arcs. By 2020, "I'm a network engineer who codes" wasn't a unicorn anymore; it was a job description.

**Stream seven: industry conferences became automation-first.** NANOG (the North American Network Operators Group) talks shifted from BGP-only sessions to "here's how we automated our DDoS-mitigation playbook" sessions. AutoCon (Network Automation Forum) launched in 2023 as the first conference dedicated entirely to network automation, drawing 1,000+ attendees in its first year. The community exists; the patterns are documented; the war stories are shared. You're not on your own.

### The Parallel to Web-Dev

We've alluded to this throughout but let's nail it down. Web development experienced a sequence of revolutions, and network engineering is following the exact same path with about a decade lag.

| Web-dev (year) | Networking (year) | Lesson |
|----------------|-------------------|--------|
| FTP files to server (~1995) | Telnet+paste config (~1995) | Manual file transfer / command entry |
| `rsync` and `scp` (~2000) | SSH+paste (~2002) | Encrypted, scriptable |
| Hand-edited Apache configs (~2002) | Hand-edited router configs (~2002) | Per-server snowflakes |
| Subversion source control (~2004) | Configs in TFTP archives (~2004) | Some history, no code review |
| Capistrano for deploys (~2007) | RANCID for backups (~2005) | Automated retrieval, not push |
| Puppet/Chef (~2009) | Cisconet/CiscoWorks (~2002, but anemic) | Config as code (server world); networks lagged |
| Git and GitHub (~2008–2012) | Configs in git (~2015) | Distributed version control hits networking 5–7 yr later |
| Ansible (~2012) | Ansible-network (~2015) | Three-year lag; same tool, same playbook style |
| Docker (~2013) | Containerized network OSes (~2018) | Cumulus, SR Linux — five-year lag |
| Kubernetes (~2014) | Network controllers (DNA Center, Apstra, ~2018) | Centralized declarative orchestration; four-year lag |
| Prometheus pull metrics (~2015) | gNMI streaming telemetry (~2017) | Move from polling to push |
| GitOps (Flux/ArgoCD ~2018) | Network GitOps (~2022) | Not yet mainstream in networking |
| Service meshes (~2018) | Microsegmentation, ZTNA (~2020) | Identity-based networking |
| Platform engineering (~2022) | NetDevOps platforms (~2023) | Self-service for non-network teams |

The key insight: every transition has the same shape. There's a manual era. There's a "let's script it" era with brittle scripts. There's a structured-tool era. There's a declarative-config era. There's a control-loop era. And finally, a self-service-platform era. Networking is somewhere between "structured tool" and "declarative config" depending on the shop. The leading edge is getting into "control loop." Most enterprises haven't even started.

If you're reading this in 2026, you're catching the wave at exactly the right time. The tooling is mature enough to be useful, the patterns are settled enough to be teachable, and the community is large enough that you won't be alone. Learning network automation in 2026 is roughly like learning Kubernetes in 2018 — past the bleeding edge, before the mainstream.

### Lessons the Web World Learned (and Now You Get to Skip the Pain)

There's an enormous unfair advantage available to network automators in 2026: you get to skip a decade of expensive lessons that the web/devops world paid for. Here are some of the most painful ones, with what to do instead.

- **"Shell scripts in a cron job" is not a deployment system.** The web world learned this through the late 2000s; networking is learning it now. If your "automation" is a cron-driven bash script that SSHes into devices, you have all the failure modes of a deployment system with none of the safeguards. Use a real orchestrator (Ansible, Nornir, AWX/Tower, GitLab CI, GitHub Actions) from day one.
- **Secrets in source control are a vulnerability, always.** The web world learned this through years of accidentally-public GitHub repos exposing AWS keys. In networking the equivalent is putting enable secrets in your YAML inventory. Don't. Use a secret manager (Vault, AWS Secrets Manager, sealed-secrets) from day one. Treat git as untrusted; secrets must be referenced, not embedded.
- **Snowflake hosts beget snowflake outages.** Cattle-not-pets. Bake from a template. If you can't rebuild a device from your SoT, you have a snowflake. The cost of converting it is large but finite; the cost of leaving it is unbounded.
- **The wiki goes stale.** Documentation that lives in a separate system from the truth always rots. Generate documentation from the SoT — diagrams from your topology data, runbooks from your playbooks, runlogs from your CI history. If the docs and the truth diverge, fix the generator, not the docs.
- **Observability before automation.** You cannot safely close the loop on a system you cannot observe. The web world figured this out the hard way through years of mysterious 500 errors. In networking: build your telemetry pipeline before you build your automated remediation, not after.
- **Tests are the engineering culture, not a feature.** Adopting CI without adopting test-writing culture gets you nothing. The first team meeting where someone says "we don't have time to write a test for this PR, just merge it" is the meeting where your automation maturity stalls. Plant the flag early: every change has a test, no exceptions.

If you internalize these six lessons before you start, you'll save yourself two years of stumbling. The hard problems in network automation in 2026 are not technical — they're cultural and process problems that other engineering disciplines have already worked through. Steal their solutions.

### Why Now: The 2026 Snapshot

A handful of things came together in the last few years that make 2026 the inflection point:

1. **Vendor support is ubiquitous.** Every major vendor (Cisco, Juniper, Arista, Nokia, Huawei, Mikrotik, even Ubiquiti's enterprise lineup) has shipped credible NETCONF/RESTCONF/gNMI support. The "the vendor doesn't support APIs" excuse no longer holds.
2. **The talent pool exists.** A decade ago you had to hire one of maybe a hundred people in the world with both networking and Python depth. Today, a typical CCNP-level engineer is expected to know enough Python to be dangerous, and plenty of Python developers can read a network diagram. The two cultures are converging.
3. **Open source is mature.** NetBox, Nautobot, Nornir, NAPALM, Containerlab, Suzieq, gnmic, Batfish-as-OSS — every layer of the stack has a credible open-source option. You can build a competent automation stack with zero vendor licenses.
4. **The AI assist is real.** LLMs (yes, the technology that wrote this sheet) can scaffold playbooks, debug Jinja2, suggest YANG paths, and explain BGP-flap mysteries. They're not replacing engineers, but they reduce the friction of the tedious 60% of the work.
5. **The patterns are settled.** A decade ago, the answer to "what should the topology of an automation pipeline look like?" got fifteen different answers from fifteen different vendors. Today, the consensus stack — git → CI → SoT → renderer → push → telemetry → drift detection → reconciler — is well-understood.

So if you've been waiting for the right moment to invest in learning this — that moment is now. The next sections will get into the actual mechanics.

## The Layers of Automation

There's a useful mental ladder for thinking about where you (or your team, or your shop) are in the automation journey. Each level has a characteristic pain that drives you to the next. You don't have to climb the whole ladder — many shops live at level 2 or 3 forever, and that's fine. But knowing where you are clarifies what to invest in next.

```
+---------------------------------------------------+
|  Level 5: Intent-based + Closed-loop              |
|  (declare goals; system reconciles continuously)  |
+---------------------------------------------------+
|  Level 4: gNMI Streaming + Real-time              |
|  (push telemetry; sub-second observability)       |
+---------------------------------------------------+
|  Level 3: NETCONF / YANG / Structured APIs        |
|  (typed configs; transactional commits)           |
+---------------------------------------------------+
|  Level 2: Ansible + Jinja2 + SoT                  |
|  (templates from inventory; agentless; idempotent)|
+---------------------------------------------------+
|  Level 1: Expect / Screen-scrape Scripts          |
|  (SSH + parsing; brittle but a start)             |
+---------------------------------------------------+
|  Level 0: Manual SSH                              |
|  (humans typing; runbooks; no automation)         |
+---------------------------------------------------+
```

### Level 0 — Manual SSH

**What you do:** SSH into devices, type commands by hand, save config, log out. Maybe you have wiki runbooks. Maybe you have a Confluence page listing all the IPs.

**What it solves:** Nothing! It's the baseline.

**What it doesn't solve:** Scale, consistency, drift, audit trail, recovery time, knowledge transfer. Everything that hurts as you grow.

**When it's OK:** A network of fewer than ~10 devices, where one person owns it all and changes happen rarely. A homelab. A tiny startup.

**When to skip ahead:** As soon as you have 50+ devices, or 2+ engineers making changes, you're paying a hidden tax in inconsistency. Move at least to Level 2.

**How to migrate to Level 1:** Don't bother. Skip Level 1 entirely. Expect/screen-scrape was a bridge from Level 0 to Level 2; if you're starting fresh, just go to Level 2 (Ansible).

**The hidden cost of staying at Level 0:** It's easy to underestimate because the daily friction is normal-feeling — typing into a CLI is what you've always done. But the cost shows up in three places. First, change windows: every change requires a human, scheduled, with sign-off, often outside business hours. Second, talent: you can only hire engineers who like CLI work, which is a shrinking pool. Third, blast radius: every outage at Level 0 takes longer to recover from because there's no automated rollback path. Add up the engineer-hours per year you'd save by moving up one rung, and the math almost always favors investing in Level 2.

### Level 1 — Expect/Screen-Scrape

**What you do:** Write `expect` scripts (or Python with `paramiko` and regex parsing of `show` output) to send commands and parse responses.

**What it solves:** Bulk operations. You can update 100 devices at once. You can query state across the fleet (e.g., "give me the OSPF neighbors on every router").

**What it doesn't solve:** Reliability — the scripts break every time the CLI changes. Idempotency — running a script twice may double-apply commands. Diff — you can't easily see "what would change?". Audit — debugging a failed script means digging through stdout.

**When it's OK:** Legacy environments where the devices don't speak NETCONF, and you're stuck. Quick one-off "I need this data from 200 devices right now" tasks.

**When to skip ahead:** As soon as the device supports NETCONF or RESTCONF (most things made after 2015 do), use that instead. As soon as you have a halfway modern Python team, use Netmiko + TextFSM/ntc-templates rather than hand-rolled regex.

**How to migrate to Level 2:** Take your `expect` scripts, identify the patterns (e.g., "configure 12 OSPF neighbors on each device"), and rewrite as Jinja2 templates over an inventory file. The Jinja templates are easier to read, easier to diff, and Ansible's `network_cli` module gives you idempotency that raw `expect` doesn't.

**A common Level-1 anti-pattern:** the "screen-scrape monolith." This is a single 5,000-line Python script that connects to every device, runs every check, parses every output, and produces a giant report. It works. It's also unreviewable. Nobody can change it without breaking three other things. The fix isn't to clean up the monolith — it's to migrate to Level 2, where each capability becomes a discrete role/playbook that's individually testable. Don't refactor the monolith; replace it.

**Another Level-1 anti-pattern:** the "vendor-specific tools sprawl." Three engineers each picked their favorite tool for their favorite vendor. One uses Cisco's pyATS. One uses Juniper's PyEZ. One uses raw paramiko with custom regex. Each tool has its own inventory, its own config style, its own idea of what "current state" looks like. Multiply by six tools and you have an operational nightmare. The fix is consolidation under a single framework (Ansible or Nornir) with vendor-specific modules underneath. The framework provides the inventory, secrets, logging, and orchestration; the vendor modules provide the device-specific knowledge.

### Level 2 — Ansible + Jinja2 + Source of Truth

**What you do:** Define inventory in YAML or pull from NetBox. Write Jinja2 templates that render configs from variables. Use Ansible's `cisco.ios`, `junipernetworks.junos`, `arista.eos` collections (or `network_cli` for generic) to push the rendered config. Optionally diff against running config first.

**What it solves:** Consistency (templates ensure every device gets the same shape of config). Bulk changes (one playbook, all devices). Source of truth (variables live in NetBox/inventory, not in heads). Some idempotency (Ansible's network modules are mostly idempotent, though it's complicated). Audit (playbook runs are logged; YAML lives in git; PRs have history).

**What it doesn't solve:** Real-time observability (you're still polling). True transactional commits (Ansible can't do candidate-config staging on most platforms — it sends commands and hopes). Pre-merge validation that's deeper than syntax (you don't really know if the change will break routing until you push it). Drift detection at scale (you can write playbooks to detect drift, but it's chatty SSH-based polling).

**When it's OK:** Most enterprise networks. Most service providers. Most data centers below the hyperscale tier. You can run a 5,000-device network on Ansible if you're disciplined, and many do.

**When to skip ahead:** When you need transactional safety (commit-confirmed, atomic multi-device changes), or when SSH polling becomes your bottleneck for monitoring. Hyperscalers and highly transactional environments (financial trading, telco core) need Levels 3–4.

**How to migrate to Level 3:** Adopt NETCONF/RESTCONF for change operations on devices that support it. Replace your "send commands via SSH" with "send YANG-encoded XML via NETCONF." This gets you transactional commits, candidate config staging, and rollback. Ansible's `netconf_config` module is one entry point; `ncclient` directly is another.

**The Level 2 ceiling:** You'll know you've hit the limits of Level 2 when (a) your runs take so long that a daily reconcile is impractical, (b) partial-failure recovery is hurting you (some devices got the change, some didn't, and now your fleet is split-brain), or (c) your config templates are getting so complex that the Jinja2 itself is becoming a maintenance burden. Levels 3+ address all three, but only if you're prepared to invest in the structured-config skills (YANG comprehension, NETCONF debugging, model navigation).

### Level 3 — NETCONF/RESTCONF + YANG

**What you do:** Configurations are validated against YANG models before being sent. Changes go to a candidate config, are committed atomically (all-or-nothing), and can be rolled back to a previous datastore. Pre-merge tooling validates the YANG.

**What it solves:** Type safety (the device rejects malformed config rather than silently misbehaving). Atomicity (multi-line changes commit as one unit; partial application is a thing of the past). Rollback (devices natively support reverting to the previous commit). Standardization (NETCONF is RFC 6241 — every modern vendor speaks it).

**What it doesn't solve:** Vendor model differences (every vendor has its own YANG; OpenConfig helps but coverage is incomplete). Real-time telemetry (NETCONF can do `<get>` polling but isn't optimized for streaming). Closed-loop reconciliation (you can detect drift but the response logic is on you).

**When it's OK:** Service provider cores, large enterprise data centers, anywhere transactional safety matters more than ease-of-use.

**When to skip ahead:** When you need streaming telemetry at sub-second freshness for closed-loop or analytics work. When polling SNMP/NETCONF for state has become a bottleneck.

**How to migrate to Level 4:** Adopt gNMI for telemetry (subscribe to streams instead of polling). Most vendors that support NETCONF also support gNMI on modern firmware; the same YANG models often back both. You don't have to give up NETCONF — many shops use NETCONF for change ops and gNMI for telemetry simultaneously.

**Why NETCONF and gNMI coexist:** They optimize for different things. NETCONF is a transactional config protocol — its strength is "apply this whole change atomically and roll back if anything fails." gNMI is a streaming telemetry protocol — its strength is "send me every counter every second forever." gNMI does have config capabilities (Set RPC), but in practice most shops use NETCONF for "change the config" and gNMI for "tell me what's happening." That's not a contradiction; it's good tool selection. A skilled shop runs both, with each doing what it's best at.

### Level 4 — gNMI Streaming Telemetry + Real-Time State

**What you do:** Devices stream state changes (interface counters, BGP table updates, link state) over gRPC subscriptions. Telemetry pipelines (Telegraf, gnmic, custom) collect, encode, and ship to time-series databases (Prometheus, InfluxDB, ClickHouse). Dashboards and alerts run on sub-second freshness.

**What it solves:** Observability at scale (one device pushing 10,000 metrics/sec is no big deal for gNMI; the same load over SNMP would melt the device's CPU). Faster MTTD (mean time to detection) for outages — link goes down, alert fires within 100ms instead of 5 min after the next SNMP poll. Capacity planning (you have rich historical data instead of 5-min averages). Closed-loop input (you have the live data needed to drive automated reactions).

**What it doesn't solve:** The intent layer. You still have to know what you want; gNMI just gives you the data.

**When it's OK:** Hyperscalers, large data centers, modern telco. Anywhere observability is a first-class concern.

**When to skip ahead:** When you want the system to react automatically to telemetry (close the loop). That's Level 5.

**How to migrate to Level 5:** Define intent (what you want the network to do, in declarative terms). Build reconcilers that compare intent to live telemetry and take corrective action. This is where most orgs are still figuring it out — there's no single dominant tool yet (DNA Center, Apstra, NSX-T, custom Kubernetes operators all play in this space).

### Level 5 — Intent-Based + Closed Loop

**What you do:** Express network intent at a high level ("VLAN 100 reaches every floor of every branch", "the trading floor has sub-50µs latency to the exchange edge", "no path uses transit ISP X if path through Y is available"). The system translates intent into device config, applies it, observes the network via streaming telemetry, and corrects if reality diverges from intent.

**What it solves:** Self-healing (cable cut → traffic reroutes → re-optimizes when cable is repaired, all without humans). Onboarding (new device boots, gets ZTP'd, joins the fabric, gets its intent-driven config, ready in minutes). Compliance (intent enforces policies — "no /0 routes," "all VLANs must have descriptions," "BGP communities must follow the company schema"). Speed (changes are intent edits, not config edits — much higher leverage per keystroke).

**What it doesn't solve:** All the hard problems of distributed systems. Reconcilers can fight each other. Intent can be ambiguous. Edge cases are still edge cases. The promise of intent-based networking has been oversold by vendors; the reality is closer to "really good Level 4 with some closed-loop pieces."

**When it's OK:** Honestly, almost nobody is fully here in 2026. Pieces of it (ZTP, drift remediation for specific patterns, anycast/DNS-based traffic engineering) are common. Full closed-loop intent-driven networking is mostly aspirational.

**When to skip ahead:** Nowhere to go yet. Level 6 (some flavor of AI-driven self-design) is on the horizon but speculative.

**How to migrate from N to N+1, in general:**
1. Don't try to go from 0 to 5 in one step. Pick the next adjacent level.
2. Start with a small slice — one site, one service, one team. Prove the value. Expand.
3. Don't kill the old layer immediately. Run new and old in parallel until trust is built.
4. Invest in observability before invest in change automation. You can't safely automate what you can't observe.
5. Source of truth comes early. Without a SoT, every automation tool is making it up as it goes — guaranteed snowflake reproduction at higher speed.
6. Test in a sandbox (Containerlab, vendor virtual images, Batfish simulation) before production.
7. Cultivate the team's mental model. Tooling without skills is shelfware.
8. Pick a "first win" project that's high-visibility and low-blast-radius. New office turn-up, lab provisioning, dev-environment teardown. Bank an early success.
9. Document the wins. Quantify time saved, errors avoided, change windows reduced. Use the numbers to justify the next investment.
10. Invest in retention. The engineer who built your automation is now critical. Lose them and you'll be 18 months behind.

The ladder is not a forced march. Many shops live at Level 2 forever and have happy, reliable networks. The point of the ladder is to know which problem you're solving today, and which problem you'll need to solve next.

### A Day in the Life at Each Level

To make this concrete: imagine the same scenario — "we need to add a new VLAN to all branch routers" — at each level. The shape of the work changes dramatically.

**At Level 0:** Engineer opens the wiki runbook. Logs into a jumpbox. SSHes into branch-1. Types the commands. Saves. SSHes into branch-2. Types the commands. Saves. Repeats forty-seven times. Time: half a day at minimum, more with breaks. Risk: high, every typing instance is a fresh chance to err. Outcome: probably-correct configs with subtle drift.

**At Level 1:** Engineer opens the `add_vlan.expect` script. Edits the VLAN ID and description. Runs it against a list of forty-seven hosts. Watches the output scroll for errors. About 30% of the time, two or three devices have a subtle prompt difference and fail; engineer fixes those by hand. Time: an hour. Risk: medium-low, but error modes are obscure. Outcome: mostly-correct configs, with a handful of hand-touched exceptions.

**At Level 2:** Engineer edits `vlans.yaml` in the source-of-truth repo to add the new VLAN. Opens a pull request. Peer reviews it. CI runs Jinja-render and lint. PR merges. Pipeline triggers Ansible to roll out to canary (one branch). Telemetry confirms healthy. Pipeline rolls out to remaining 46. Time: 20 minutes of human work plus an hour of pipeline runtime. Risk: low, every step is observed. Outcome: identical configs across all 47 devices, captured in git history.

**At Level 3:** Same as Level 2, but the rollout uses NETCONF candidate-config commits with auto-rollback. If any device's reachability check fails after the commit, that device's commit auto-reverts within 90 seconds. Time: same. Risk: even lower because partial failures self-heal. Outcome: same.

**At Level 4:** Same as Level 3, plus telemetry confirms at sub-second granularity that traffic shaping on the new VLAN is working as expected — no missing flows, no unexpected drops, no policy mismatches. Engineer moves on to the next ticket within 25 minutes total.

**At Level 5:** Engineer doesn't even add a VLAN. Engineer adds a service ("Marketing-Floor needs Internet") to the intent layer. The system figures out which VLAN, which IPs, which ACLs, which QoS, and propagates to all relevant devices. Time: 5 minutes. Outcome: change ships, telemetry confirms, the engineer didn't even need to know which routers were involved.

That's the whole arc, condensed. Each level cuts the time and the risk by roughly half. Each level requires more upfront investment in tooling and discipline. The right level for you is the one where the upfront cost equals the avoided pain — and the threshold moves over time as your network grows and the tools improve.

### Skipping Levels (and When You Shouldn't)

A common temptation in 2026 is to skip from Level 0 directly to Level 4 or 5 — buy a controller, plug it in, declare victory. This usually fails. The reasons:

- Controllers depend on a clean source of truth. If your network is a brownfield mess, the controller will inherit the mess and amplify it.
- Streaming telemetry without operational maturity is just a firehose. You need the people, dashboards, and alerting discipline to use the data.
- Intent-based systems make assumptions about your operating model that often don't match reality. If your firewall change process requires three approvals from a security team, no amount of intent declaration will skip the approvals.

The shops that get the most value from Levels 4–5 are those that climbed to a solid Level 3 first. They have a SoT. They have observability discipline. They have CI culture. The controller, when added, sits on top of a working foundation. The shops that "skip" usually end up running a janky Level 1 underneath a fancy Level 5 dashboard, with the worst of both worlds.

So: climb the ladder. Pick the next adjacent rung. Don't try to skip three.

### What "Done" Looks Like at Each Level

If you're trying to assess your own shop, here's a quick litmus test for each level. You're "at" a level if you can answer "yes" to all the questions for that level (and, naturally, all lower levels).

**Level 0 done:** Can you reproduce a basic config change reliably? Do you have working credentials for every device? Do you know how to roll back? (Surprisingly often, the answer to the third question is no.)

**Level 1 done:** Can you run a script against the entire fleet and get structured output? Can you re-run the script and have it skip already-configured devices? Do you have a way to see the diff between current state and target state for any single change?

**Level 2 done:** Are configs in git? Are PRs reviewed? Does CI run on PRs? Do you have a defined merge → deploy pipeline that doesn't require manual intervention to push?

**Level 3 done:** Are changes transactional? Does a failed change auto-rollback? Is your config validated against a schema before being sent? Do you use candidate-config staging?

**Level 4 done:** Are you ingesting telemetry from devices via streaming (not polling)? Do you have a time-series store with multi-month retention? Can you correlate a config change to a telemetry shift in under a minute?

**Level 5 done:** Does the system reconcile drift without human intervention? Can you express intent in business terms ("Marketing has Internet") and have it translate to device config? Does the system refuse to do something that violates policy?

Most shops in 2026 are honestly at Level 2 with aspirations toward Level 3. That's a perfectly fine place to be. The point of the ladder isn't to make you feel inadequate; it's to give you a vocabulary for "where we are" and "where we're going."

### The Mixed-Level Reality

In practice, most networks operate at multiple levels simultaneously, depending on the layer of the network and the team operating it.

- **Data center fabric:** often Level 3 or 4. Hyperscalers run their leaf-spine fabrics at Level 4-5 with custom intent compilers; smaller shops run them at Level 2-3 with Ansible and NETCONF.
- **Campus access switches:** often Level 2. The blast radius of a single access switch is small enough that simple Ansible playbooks suffice. Vendor controllers (Cisco DNA Center, Aruba Central) creep this toward Level 3-4 in larger orgs.
- **WAN/SD-WAN:** Level 3 if you're using a controller (Viptela, Versa, Silver Peak, Cato). Otherwise often Level 1-2.
- **Firewalls:** notoriously stuck at Level 1-2. Vendor APIs are weaker; security teams are conservative; the blast radius of a misconfig is "the entire perimeter is down."
- **Load balancers and proxies:** often Level 3-4 in cloud-native shops; Level 2 elsewhere.
- **Wireless controllers:** Level 2-3. Most large enterprises run a vendor controller (Mist, Aruba, Cisco) that internally implements automation, with a customer-facing API for higher-level orchestration.
- **Telco core (RAN/transport):** Level 3-4 driven by ETSI/3GPP standards. Increasingly Level 5 in greenfield 5G deployments.

Recognizing the mixed-level reality is liberating: you don't have to bring your entire network up the ladder simultaneously. You can climb the data-center fabric to Level 4 while leaving the campus access at Level 2, and that's a perfectly defensible architecture. The ladder applies per-domain, not per-organization.

### Common Anti-Patterns at Every Level

A short field guide to mistakes you'll see (or make) at each level. Knowing the anti-pattern is half the battle.

- **Level 0:** "We have a wiki" (the wiki is out of date by the time you read it).
- **Level 1:** "We have a script" (the script is owned by one person and unreadable by anyone else).
- **Level 2:** "We have Ansible" (but no SoT, so the playbooks are templated by hand).
- **Level 3:** "We use NETCONF" (but you're sending vendor-native models, so you're locked in).
- **Level 4:** "We stream telemetry" (but the time-series store has 7-day retention, so you can't see seasonal patterns).
- **Level 5:** "We have intent" (but the controller's idea of intent doesn't match yours, so you fight it constantly).

The pattern across all of these: **adopting the tool is the easy part; using the tool well is the hard part.** Each level is a long-term commitment to operational discipline as much as it is a tooling choice. If your team can't sustain the discipline, you'll regress to a lower level whether the tools allow it or not.

That's the ladder. Climb at your own pace. Skip when you can. Don't pretend you're at a higher level than you actually are. Be honest about it; the work follows the honesty.

### Where to Start Today

If you've read this far and you're at Level 0 wondering what to do tomorrow morning, here's a concrete starter plan:

1. **Pick one repetitive task you do at least weekly.** New port turn-up. VLAN changes. Config backup. ACL updates. Whatever happens often enough to feel painful.
2. **Set up a git repo.** Just `git init`, no fancy hosting required for the prototype. Commit your current per-device configs as a baseline.
3. **Stand up Ansible (or Nornir if you're Python-comfortable).** Install on a Linux box, write your first inventory file with three to five test devices.
4. **Pick a vendor module that matches your fleet.** `cisco.ios`, `arista.eos`, `junipernetworks.junos`, `nokia.srlinux`, etc.
5. **Write one playbook that does the repetitive task end-to-end.** Test it against a non-prod device. Iterate until it's reliable.
6. **Get peer review.** Find one teammate willing to review your YAML. Make this a habit, not a one-off.
7. **Repeat for three more tasks.** By the third or fourth playbook, the patterns become muscle memory.
8. **Then start thinking about a SoT.** NetBox is the obvious first stop; a `vars.yaml` file is fine for a small fleet.

That's it. Three to six months of discipline and you're at a credible Level 2. You don't need a budget. You don't need executive sign-off. You don't need to convince anyone. Just start.

## NETCONF Deep ELI5

NETCONF is **git for routers** — only the router doesn't have a `git` binary, it has a tiny XML-speaking server bolted to its SSH daemon. You connect over SSH (port **830**, NOT port 22), exchange XML messages called RPCs, and the router answers with more XML.

If you've ever typed `git pull && edit config && git commit && git push`, you already understand NETCONF. The router has separate "branches" called **datastores**, you `lock` one, `edit-config` it, `validate` it, `commit`, and if you broke the world you `discard-changes`. The whole thing is ceremoniously XML because NETCONF was specified in 2006 (RFC 4741, then refined in **RFC 6241** in 2011) when XML was The Way and JSON hadn't won yet.

### Why NETCONF Exists

Before NETCONF, "config a router" meant SSH in, paste lines, pray. There was:

- **No transactional semantics** — half a config could apply, the other half could fail, you'd have a half-broken router with no rollback.
- **No schema** — every vendor's CLI was its own snowflake, parsing it was a regex nightmare.
- **No structured response** — `show interfaces` returned text humans read, not data machines parse.
- **No locks** — two engineers could push conflicting changes at the same time and silently corrupt each other.

NETCONF fixes all four:

- **Transactional**: edit `candidate`, `commit` atomically, optionally `commit confirmed` with auto-rollback.
- **Schema-driven**: every payload is validated against a YANG model.
- **Structured**: replies are XML you can XPath into.
- **Locks**: `<lock>` a datastore so nobody else can edit while you're working.

### The SSH Subsystem

Plain SSH gives you a shell. NETCONF runs as an **SSH subsystem** — same SSH transport, different application protocol. The client says "give me the netconf subsystem":

```bash
ssh -s -p 830 admin@router.example.com netconf
```

The `-s` flag tells SSH "I want a subsystem, not a shell," and `netconf` is the subsystem name. The router answers by sending an XML `<hello>` message; you respond with your own `<hello>`; from then on you swap RPCs framed by either the legacy `]]>]]>` end-of-message marker (NETCONF 1.0) or the chunked-framing format (NETCONF 1.1).

Port **830** is the IANA-assigned NETCONF port. Many vendors also let NETCONF run over port 22 if you specify `-s netconf` on a normal SSH session, but **830 is the standard and what your firewall rules should allow**.

### The Capabilities Exchange — `<hello>`

The first thing both sides do is announce what they can do. The router sends:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:writable-running:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:candidate:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:confirmed-commit:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:rollback-on-error:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:validate:1.1</capability>
    <capability>urn:ietf:params:netconf:capability:startup:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:xpath:1.0</capability>
    <capability>urn:ietf:params:netconf:capability:notification:1.0</capability>
    <capability>http://cisco.com/ns/yang/Cisco-IOS-XE-native?module=Cisco-IOS-XE-native&amp;revision=2024-03-01</capability>
    <capability>http://openconfig.net/yang/interfaces?module=openconfig-interfaces&amp;revision=2024-04-04</capability>
  </capabilities>
  <session-id>4711</session-id>
</hello>
]]>]]>
```

Each `<capability>` URI tells you something:

- `base:1.0` / `base:1.1` — protocol version (1.1 has chunked framing, 1.0 uses `]]>]]>`).
- `writable-running` — you can edit `running` directly (no `candidate` required).
- `candidate` — there is a `candidate` datastore.
- `confirmed-commit:1.1` — supports the "commit then rollback if not confirmed" pattern.
- `rollback-on-error` — if any operation in a transaction fails, roll the whole thing back.
- `validate:1.1` — supports the `<validate>` operation.
- `startup` — there is a separate `startup` datastore.
- `xpath:1.0` — you can use XPath in filters, not just subtree.
- `notification:1.0` — supports streaming notifications.
- The `http://...?module=...&revision=...` URIs are **YANG modules** the router supports (one per loaded model).

The client sends back its own `<hello>` declaring the highest base it supports. The lower of the two wins. After that, it's RPC time.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
  </capabilities>
</hello>
]]>]]>
```

### Datastores

A **datastore** is a named container of configuration. NETCONF defines three standard ones:

```
        +------------------+
        |     <startup>    |   persistent across reboot
        +--------+---------+
                 ^
                 | copy-config
                 |
        +--------+---------+
        |     <running>    |   what the device is doing right now
        +--------+---------+
                 ^
                 | commit
                 |
        +--------+---------+
        |    <candidate>   |   scratch pad — edit safely, commit when ready
        +------------------+
```

- **`running`** — the live config. Always exists. On some devices you can edit it directly (`writable-running` capability); on others you can only edit `candidate` and `commit` to push it into `running`.
- **`candidate`** — scratch pad. You stage edits here, validate, then `commit`. Multiple edits accumulate; `discard-changes` throws them away.
- **`startup`** — what gets loaded on boot. On Cisco IOS-XE this is "what `wr mem` writes." If absent, `running` is also `startup` (config persists automatically).

Some vendors add **`intended`** and **`operational`** datastores under NMDA (Network Management Datastore Architecture, RFC 8342) — those are more relevant for RESTCONF and gNMI than classic NETCONF.

### The Operations

NETCONF defines a small, fixed set of RPCs. Each is wrapped in `<rpc message-id="...">` with a unique message ID; the router replies with `<rpc-reply message-id="...">` matching that ID.

#### `<get>` — read everything

Returns running config **plus** operational state. Heavy. Use sparingly.

```xml
<rpc message-id="101" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get/>
</rpc>
```

#### `<get-config>` — read just config

Lighter. Specify which datastore.

```xml
<rpc message-id="102" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
  </get-config>
</rpc>
```

With a subtree filter to narrow it down:

```xml
<rpc message-id="103" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get-config>
    <source><running/></source>
    <filter type="subtree">
      <interfaces xmlns="http://openconfig.net/yang/interfaces">
        <interface>
          <name>GigabitEthernet0/0/0/1</name>
        </interface>
      </interfaces>
    </filter>
  </get-config>
</rpc>
```

#### `<edit-config>` — change config

The workhorse. Default operation is `merge` (merge new tree with existing). Other ops via the `nc:operation` attribute: `merge`, `replace`, `create`, `delete`, `remove`.

```xml
<rpc message-id="104" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <edit-config>
    <target><candidate/></target>
    <config>
      <interfaces xmlns="http://openconfig.net/yang/interfaces">
        <interface>
          <name>GigabitEthernet0/0/0/1</name>
          <config>
            <name>GigabitEthernet0/0/0/1</name>
            <description>uplink to spine-1</description>
            <enabled>true</enabled>
            <mtu>9000</mtu>
          </config>
        </interface>
      </interfaces>
    </config>
  </edit-config>
</rpc>
```

#### `<copy-config>` — overwrite one datastore from another

```xml
<rpc message-id="105" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <copy-config>
    <target><startup/></target>
    <source><running/></source>
  </copy-config>
</rpc>
```

This is the equivalent of `wr mem` — copies running into startup so it survives reboot.

#### `<delete-config>` — wipe a datastore

```xml
<rpc message-id="106" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <delete-config>
    <target><startup/></target>
  </delete-config>
</rpc>
```

You usually cannot delete `running` (rejected with an error).

#### `<lock>` / `<unlock>` — exclusive access

```xml
<rpc message-id="107" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <lock>
    <target><candidate/></target>
  </lock>
</rpc>
```

If somebody else holds the lock you get an `<rpc-error>` back. Always lock before edit, unlock after commit.

```xml
<rpc message-id="108" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <unlock>
    <target><candidate/></target>
  </unlock>
</rpc>
```

#### `<commit>` — push candidate to running

```xml
<rpc message-id="109" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <commit/>
</rpc>
```

Atomic. Either the whole candidate becomes running or none of it does.

#### `<validate>` — check syntactically before commit

```xml
<rpc message-id="110" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <validate>
    <source><candidate/></source>
  </validate>
</rpc>
```

#### `<discard-changes>` — throw away candidate

```xml
<rpc message-id="111" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <discard-changes/>
</rpc>
```

Resets candidate to look exactly like running.

#### `<kill-session>` — boot another session

```xml
<rpc message-id="112" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <kill-session>
    <session-id>4710</session-id>
  </kill-session>
</rpc>
```

Useful when somebody else's stale session is holding a lock.

#### `<close-session>` — graceful logout

```xml
<rpc message-id="113" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <close-session/>
</rpc>
```

### Subtree Filtering vs XPath Filtering

When you `<get>` or `<get-config>` you can filter to just what you need. Two flavors:

**Subtree filter** — you provide a skeleton XML tree; the router returns the matching subtree. Selection is by structure: empty elements are "match anything here," populated elements are "exact match this value."

```xml
<filter type="subtree">
  <interfaces xmlns="http://openconfig.net/yang/interfaces">
    <interface>
      <name>GigabitEthernet0/0/0/1</name>
      <state/>
    </interface>
  </interfaces>
</filter>
```

This says "show me the `<state>` of the interface named `GigabitEthernet0/0/0/1`."

**XPath filter** — full XPath 1.0. Powerful but only available if the device advertises `:xpath:1.0`.

```xml
<filter type="xpath" select="/oc-if:interfaces/oc-if:interface[oc-if:name='GigabitEthernet0/0/0/1']/oc-if:state"
        xmlns:oc-if="http://openconfig.net/yang/interfaces"/>
```

XPath wins for "all interfaces where admin-status is up AND mtu > 1500" — predicates aren't expressible as subtrees. Subtree wins for "give me this exact branch" — simpler, easier to read, supported everywhere.

### Commit-Confirmed and Rollback (RFC 6241 §8.4, refined by RFC 4741bis)

The "I'm about to push a config change that might cut my own SSH session" problem. Solution: **confirmed commit**.

```xml
<rpc message-id="201" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <commit>
    <confirmed/>
    <confirm-timeout>120</confirm-timeout>
  </commit>
</rpc>
```

The router commits `candidate` to `running` — but starts a **120-second timer**. If you don't send a follow-up `<commit/>` (without `<confirmed/>`) before the timer expires, the router automatically rolls running back to what it was before. This means:

1. You commit-confirmed your change.
2. Your SSH still works → you send a plain `<commit/>` to confirm. Done.
3. Your SSH was killed by your own change → timer expires → router auto-rolls back → you can SSH again and figure out what went wrong.

**This is the killer feature**. It's the network engineer's safety net for "I'm not 100% sure this MTU change won't break my management tunnel." Use it any time the change touches reachability.

### `ncclient` — the Python NETCONF Client

`ncclient` is the de facto Python NETCONF library. Install:

```bash
pip install ncclient
```

A complete worked example — connect, get-config, edit, validate, commit-confirmed, confirm:

```python
from ncclient import manager
from ncclient.xml_ import to_xml

device = {
    "host": "router.example.com",
    "port": 830,
    "username": "admin",
    "password": "redacted",
    "hostkey_verify": False,           # set True in production
    "device_params": {"name": "iosxr"}, # or "csr", "junos", "default"
    "allow_agent": False,
    "look_for_keys": False,
}

with manager.connect(**device) as m:
    print("Server capabilities:")
    for cap in m.server_capabilities:
        print("  ", cap)

    # 1. read current interface config
    filter_xml = """
    <filter type="subtree">
      <interfaces xmlns="http://openconfig.net/yang/interfaces">
        <interface>
          <name>GigabitEthernet0/0/0/1</name>
        </interface>
      </interfaces>
    </filter>
    """
    reply = m.get_config(source="running", filter=filter_xml)
    print("Current config:")
    print(reply.data_xml)

    # 2. lock candidate
    with m.locked("candidate"):
        # 3. edit candidate
        edit_xml = """
        <config>
          <interfaces xmlns="http://openconfig.net/yang/interfaces">
            <interface>
              <name>GigabitEthernet0/0/0/1</name>
              <config>
                <name>GigabitEthernet0/0/0/1</name>
                <description>uplink to spine-1 (jumbo)</description>
                <mtu>9000</mtu>
                <enabled>true</enabled>
              </config>
            </interface>
          </interfaces>
        </config>
        """
        m.edit_config(target="candidate", config=edit_xml)

        # 4. validate
        m.validate(source="candidate")

        # 5. commit-confirmed with 120s rollback window
        m.commit(confirmed=True, timeout="120")

        # 6. ... verify reachability here ...
        # if all good:
        m.commit()  # confirm, cancels the auto-rollback
```

Key things to notice:

- `manager.connect(**device)` does the SSH + hello handshake.
- `device_params={"name": "iosxr"}` tells `ncclient` which vendor quirks to apply. Options: `default`, `iosxr`, `iosxe` (use `csr`), `junos`, `nexus`, `huaweiyang`, `alu`.
- `m.locked("candidate")` is a context manager — locks on enter, unlocks on exit, even if your code raises.
- `m.edit_config(target=..., config=...)` takes XML as a string — `ncclient` wraps it in `<rpc><edit-config>...`.
- `m.commit(confirmed=True, timeout="120")` issues the confirmed-commit RPC.
- A second `m.commit()` (no args) confirms; the rollback timer is cancelled.
- If you crash or your script exits without confirming, the timer expires and the router rolls back. **Built-in safety net**.

### Common Gotchas

**Vendor capability differences.** Cisco IOS-XR supports `:candidate` natively, IOS-XE has it for NETCONF but not always for the same paths as RESTCONF, NX-OS supports `:writable-running` only on some platforms, Junos is candidate-based and uses its own commit-confirm syntax. Always check `m.server_capabilities` first.

**Candidate not supported.** If the device only advertises `:writable-running:1.0` (no `:candidate:1.0`), you have to edit `running` directly. No `commit`, no `discard-changes`. Edits are immediate. Be careful.

**Port confusion.** NETCONF SSH subsystem is **port 830**. NETCONF over SSH is also possible on port 22 with `-s netconf`, and many vendors do **NETCONF over TLS** on port 6513 (RFC 7589). Older Junos used port 22 + `netconf` subsystem by default. Always check `show netconf-yang sessions` (or vendor equivalent) and your firewall rules.

**Hello timeout.** If your client doesn't respond to the device's `<hello>` within the device's hello timeout (often 30s, sometimes 60s), the device drops the session with `Hello: timed out`. Using a slow Python REPL or breaking on a debugger can trigger this. `ncclient` does the hello automatically and quickly, but if you're hand-rolling NETCONF, send your hello immediately.

**Namespace prefixes.** XML namespaces are real. `<interfaces xmlns="http://openconfig.net/yang/interfaces">` and `<oc-if:interfaces xmlns:oc-if="http://openconfig.net/yang/interfaces">` are the same thing logically — but if you mismatch the namespace URI you get back `<rpc-error>unknown-element</rpc-error>` and a confused face.

**Chunked framing vs `]]>]]>`.** NETCONF 1.0 ends each message with `]]>]]>`. NETCONF 1.1 uses chunked framing (`#<size>\n<bytes>` then `##\n` to end). `ncclient` handles this for you, but raw `socket` + `paramiko` code has to know which one to use based on the agreed base capability.

**`get` is huge.** A full `<get/>` on a busy data-center switch returns megabytes of operational state (every neighbor, every counter). Always filter. Use `<get-config>` if you only need config.

**Locks are per-datastore, per-session.** If you lock candidate and your session dies without unlocking, the lock is auto-released (NETCONF tracks session liveness). But if it doesn't get released, use `<kill-session>` from a second session to clear it.

**Why XML?** History. NETCONF predates JSON's network-management adoption. It's verbose, it's annoying to type, but it's expressive (mixed content, attributes, namespaces) and every device speaks it. **YANG**'s data model maps cleanly to either XML or JSON; NETCONF chose XML, RESTCONF and gNMI later added JSON. The data model is the same — only the wire encoding differs.

## YANG Deep ELI5

YANG is **TypeScript for network configuration**. You write a `.yang` file declaring "an interface has a name (string), an MTU (int between 64 and 9216), an admin-status (one of UP/DOWN/TESTING)" and that becomes the source of truth for what valid config looks like. Tools like NETCONF/RESTCONF/gNMI use it to validate every message you send, every reply you get.

YANG was specified in **RFC 6020** (YANG 1.0, 2010) and updated in **RFC 7950** (YANG 1.1, 2016). The 1.1 version added several features — `anydata`, action statements, notifications inside data nodes, improved type system — but the basic shape is identical.

### What YANG Looks Like

A tiny module that defines a router with a list of interfaces:

```yang
module example-router {
    yang-version 1.1;
    namespace "urn:example:router";
    prefix "exrtr";

    organization "Example Corp";
    contact "noc@example.com";
    description "Toy router model for the YANG ELI5 section.";
    revision 2026-04-27 {
        description "Initial version.";
    }

    container router {
        description "Top-level router config.";
        leaf hostname {
            type string {
                length "1..63";
                pattern "[a-zA-Z][a-zA-Z0-9-]*";
            }
            mandatory true;
        }
        list interface {
            key "name";
            leaf name {
                type string;
            }
            leaf mtu {
                type uint16 {
                    range "64..9216";
                }
                default 1500;
            }
            leaf admin-status {
                type enumeration {
                    enum UP;
                    enum DOWN;
                    enum TESTING;
                }
                default UP;
            }
        }
    }
}
```

This module declares a single top-level **container** `router` containing a `hostname` **leaf** and a **list** of interfaces. Each interface has a `name` (the key, like a primary key in a database), an `mtu` and an `admin-status`. A device that "implements" this module agrees: "yes, you can configure these fields, and I'll reject anything outside the constraints."

### Modules and Submodules

A **module** is the top-level unit. Each module:

- Has a unique **namespace** (URI) — e.g. `http://openconfig.net/yang/interfaces`.
- Has a **prefix** — short alias used inside the module (`exrtr` above).
- May **import** other modules to reference their types.
- May **include** submodules, which are pieces of the same module split across files for organization.
- Has zero or more **revisions**, each dated, newest first.

```yang
module openconfig-interfaces {
    yang-version 1.1;
    namespace "http://openconfig.net/yang/interfaces";
    prefix "oc-if";

    import openconfig-extensions { prefix "oc-ext"; }
    import ietf-yang-types { prefix "yang"; }

    include openconfig-interfaces-base;   // submodule
    include openconfig-interfaces-state;

    revision 2024-04-04 { ... }
    revision 2023-07-01 { ... }
    revision 2022-12-22 { ... }
}
```

A **submodule** uses `submodule X { belongs-to Y; ... }` and contributes to its parent module's namespace. Submodules are an organizational tool; consumers see one logical module.

### Containers, Leaves, Leaf-lists, Lists

- **`container`** — like a struct or a JSON object. Groups related nodes. Always exactly one (or zero, if `presence` is set).
- **`leaf`** — a single named scalar value with a type.
- **`leaf-list`** — an ordered list of values of a single type.
- **`list`** — an ordered collection of structured entries, each identified by a **`key`** (one or more leafs).

```yang
container interfaces {
    list interface {
        key "name";
        leaf name {
            type string;
        }
        leaf description {
            type string;
        }
        leaf mtu {
            type uint16;
        }
        leaf-list secondary-ipv4-addresses {
            type string;          // toy example; real one would use ipv4-address
            ordered-by user;
        }
    }
}
```

In NETCONF XML this becomes:

```xml
<interfaces>
  <interface>
    <name>eth0</name>
    <description>uplink</description>
    <mtu>1500</mtu>
    <secondary-ipv4-addresses>10.0.0.1/24</secondary-ipv4-addresses>
    <secondary-ipv4-addresses>10.0.0.2/24</secondary-ipv4-addresses>
  </interface>
</interfaces>
```

In JSON-IETF (RESTCONF/gNMI):

```json
{
  "interfaces": {
    "interface": [
      {
        "name": "eth0",
        "description": "uplink",
        "mtu": 1500,
        "secondary-ipv4-addresses": ["10.0.0.1/24", "10.0.0.2/24"]
      }
    ]
  }
}
```

### Built-in Types

YANG ships a complete primitive type system:

| Type | Range | Notes |
|------|-------|-------|
| `int8` | -128 to 127 | |
| `int16` | -32768 to 32767 | |
| `int32` | -2³¹ to 2³¹-1 | |
| `int64` | -2⁶³ to 2⁶³-1 | |
| `uint8` | 0 to 255 | |
| `uint16` | 0 to 65535 | typical for MTU, VLAN ID |
| `uint32` | 0 to 4294967295 | typical for ASN, OSPF area, ifIndex |
| `uint64` | 0 to 2⁶⁴-1 | typical for byte counters |
| `decimal64` | depends on `fraction-digits` | for "9.5", "12.34" |
| `string` | UTF-8 | |
| `boolean` | `true` / `false` | |
| `enumeration` | named symbolic values | `enum UP; enum DOWN;` |
| `bits` | named bit positions | `bit unicast { position 0; }` |
| `binary` | base64-encoded bytes | for certs, raw blobs |
| `leafref` | reference to another leaf via XPath | foreign key |
| `identityref` | reference to an identity hierarchy | extensible enums |
| `instance-identifier` | XPath to a specific instance | e.g. an interface entry |
| `empty` | leaf either present or absent | flag-style |
| `union` | one of several types | `union { type uint32; type string; }` |

Plus the standard import `ietf-yang-types` (RFC 6991) which gives you reusable types:

- `yang:date-and-time` — `2026-04-27T13:24:11Z`
- `yang:counter32`, `yang:counter64` — monotonically-increasing counters
- `yang:gauge32`, `yang:gauge64` — values that go up and down
- `yang:mac-address` — `aa:bb:cc:dd:ee:ff`
- `yang:phys-address` — generic L2 address
- `yang:dotted-quad` — `192.0.2.1`
- `yang:uuid` — `f81d4fae-7dec-11d0-a765-00a0c91e6bf6`
- `yang:hex-string` — `1a:2b:3c`

And `ietf-inet-types` (also RFC 6991):

- `inet:ipv4-address` — `192.0.2.1`
- `inet:ipv6-address` — `2001:db8::1`
- `inet:ip-address` — either
- `inet:ipv4-prefix` — `192.0.2.0/24`
- `inet:port-number` — `0..65535`
- `inet:as-number` — `uint32`
- `inet:domain-name` — DNS-friendly string
- `inet:uri` — RFC 3986 URI

You'd import them:

```yang
import ietf-inet-types { prefix inet; }
import ietf-yang-types { prefix yang; }

leaf address { type inet:ipv4-address; }
leaf last-up { type yang:date-and-time; }
```

### Constraints — `range`, `length`, `pattern`, `must`, `when`

YANG validates every value against type-level and node-level constraints.

**`range`** (numeric only):

```yang
leaf mtu {
    type uint16 {
        range "64..9216";
    }
}
```

**`length`** (string-like only):

```yang
leaf description {
    type string {
        length "0..255";
    }
}
```

**`pattern`** (regex on strings):

```yang
leaf hostname {
    type string {
        pattern "[a-zA-Z][a-zA-Z0-9-]{0,62}";
    }
}
```

**`must`** (XPath assertion at this node — schema-wide constraint):

```yang
leaf bandwidth-percentage {
    type uint8 { range "0..100"; }
    must "current() <= 100" {
        error-message "Bandwidth percent cannot exceed 100.";
    }
}
```

**`when`** (this node only exists if XPath is true):

```yang
leaf vlan-id {
    type uint16 { range "1..4094"; }
    when "../interface-mode = 'TRUNK' or ../interface-mode = 'ACCESS'";
}
```

### Groupings, `uses`

**`grouping`** is a reusable bag of nodes — like a struct definition or a mixin. `uses` instantiates it inside another container.

```yang
grouping address-family-config {
    leaf enabled {
        type boolean;
        default true;
    }
    leaf mtu {
        type uint16;
    }
}

container ipv4 {
    uses address-family-config;
    leaf default-gateway {
        type inet:ipv4-address;
    }
}

container ipv6 {
    uses address-family-config;
    leaf default-gateway {
        type inet:ipv6-address;
    }
}
```

### `augment` — extending without forking

This is the **superpower** of YANG. You can add nodes to someone else's model without modifying their `.yang` file.

```yang
module cisco-ios-xe-interfaces-augment {
    namespace "http://cisco.com/ns/yang/iosxe-if-augment";
    prefix "cisco-if-aug";

    import openconfig-interfaces { prefix "oc-if"; }

    augment "/oc-if:interfaces/oc-if:interface/oc-if:config" {
        leaf load-interval {
            type uint16 {
                range "30..600";
            }
            description "Cisco-specific load averaging interval (seconds).";
        }
    }
}
```

After loading both modules, the device understands a new `load-interval` leaf hanging under every interface — but the OpenConfig core module is **untouched**. This is how vendors add their proprietary knobs to OpenConfig without breaking the standard model.

### `deviation` — declaring "I don't support this"

The opposite of `augment`. Lets a vendor say "the standard model says X is mandatory, but on my hardware it's not supported."

```yang
module cisco-ios-xe-deviations {
    namespace "http://cisco.com/ns/yang/iosxe-deviations";
    prefix "cisco-dev";

    import openconfig-interfaces { prefix "oc-if"; }

    deviation /oc-if:interfaces/oc-if:interface/oc-if:config/oc-if:loopback-mode {
        deviate not-supported;
    }
}
```

A NETCONF client that knows about deviations can avoid sending `loopback-mode` to this device.

### Worked Example — OpenConfig Interfaces

A real (simplified) snippet from `openconfig-interfaces`:

```yang
container interfaces {
    description "Top-level container for interface config and state.";
    list interface {
        key "name";
        leaf name {
            type leafref {
                path "../config/name";
            }
        }
        container config {
            description "Configurable parameters for the interface.";
            leaf name {
                type string;
            }
            leaf type {
                type identityref {
                    base ift:interface-type;
                }
                mandatory true;
            }
            leaf mtu {
                type uint16;
            }
            leaf description {
                type string;
            }
            leaf enabled {
                type boolean;
                default true;
            }
        }
        container state {
            config false;
            description "Operational state, read-only.";
            leaf name { type string; }
            leaf type { type identityref { base ift:interface-type; } }
            leaf mtu { type uint16; }
            leaf admin-status {
                type enumeration {
                    enum UP;
                    enum DOWN;
                    enum TESTING;
                }
            }
            leaf oper-status {
                type enumeration {
                    enum UP;
                    enum DOWN;
                    enum TESTING;
                    enum UNKNOWN;
                    enum DORMANT;
                    enum NOT_PRESENT;
                    enum LOWER_LAYER_DOWN;
                }
            }
            leaf ifindex {
                type uint32;
            }
            container counters {
                leaf in-octets { type yang:counter64; }
                leaf out-octets { type yang:counter64; }
                leaf in-errors { type yang:counter64; }
            }
        }
    }
}
```

Notice the **OpenConfig pattern**: a `config` container (read-write) and a `state` container (read-only — `config false;`). The state container often duplicates the config-mirror leaves (so you can read back what was applied) and adds operational telemetry (`oper-status`, `counters`).

### `pyang` — the YANG Compiler

`pyang` parses and validates YANG modules and emits useful views.

Install:

```bash
pip install pyang
```

Tree view of a module:

```bash
pyang --format tree openconfig-interfaces.yang
```

Output (truncated):

```
module: openconfig-interfaces
  +--rw interfaces
     +--rw interface* [name]
        +--rw name             -> ../config/name
        +--rw config
        |  +--rw name?           string
        |  +--rw type            identityref
        |  +--rw mtu?            uint16
        |  +--rw description?    string
        |  +--rw enabled?        boolean
        +--ro state
           +--ro name?           string
           +--ro type            identityref
           +--ro mtu?            uint16
           +--ro admin-status?   enumeration
           +--ro oper-status?    enumeration
           +--ro ifindex?        uint32
           +--ro counters
              +--ro in-octets?   yang:counter64
              +--ro out-octets?  yang:counter64
              +--ro in-errors?   yang:counter64
```

Symbols: `+--rw` = read-write, `+--ro` = read-only, `*` = list, `?` = optional, `->` = leafref.

HTML/JS browser tree:

```bash
pyang --format jstree -o interfaces.html openconfig-interfaces.yang
```

Open `interfaces.html` in a browser → expandable tree with hover descriptions.

JSON schema (for tools that don't speak YANG):

```bash
pyang --format jsonxsl openconfig-interfaces.yang
```

### `yanglint` — Validation, Including Data

`pyang` checks the model. **`yanglint`** (from the libyang project) goes further: validates **data files** against models.

```bash
yanglint -f tree openconfig-interfaces.yang
```

```bash
# validate a config payload against the loaded modules
yanglint openconfig-interfaces.yang my-config.json
```

If `my-config.json` violates a constraint (mtu out of range, missing mandatory leaf, unknown enum), `yanglint` prints the exact path and error.

### YANG → JSON Encoding (RFC 7951)

YANG was originally XML-only. **RFC 7951** defines the IETF JSON encoding rules — used by RESTCONF and gNMI.

Key rules:

1. Container → JSON object.
2. List → JSON array of objects.
3. Leaf → JSON scalar (string/number/boolean).
4. Top-level data nodes are **namespace-qualified** with `module:name` keys. Same module → bare name; foreign module → `module:name`.
5. `int64` and `uint64` are **strings** in JSON (because JS numbers can't hold 64-bit ints reliably).
6. Empty leaf → `[null]` (a one-element array containing null).
7. `decimal64` → JSON number.

Example:

```json
{
  "openconfig-interfaces:interfaces": {
    "interface": [
      {
        "name": "eth0",
        "config": {
          "name": "eth0",
          "type": "iana-if-type:ethernetCsmacd",
          "mtu": 1500,
          "enabled": true
        },
        "state": {
          "name": "eth0",
          "oper-status": "UP",
          "counters": {
            "in-octets": "12345678901234",
            "out-octets": "98765432109876"
          }
        }
      }
    ]
  }
}
```

Notice `"in-octets": "12345678901234"` — a **string** because it's a `counter64`. Notice `"openconfig-interfaces:interfaces"` — namespace-qualified at the top level. Notice `"iana-if-type:ethernetCsmacd"` — an `identityref` value uses `module:identity` form.

### YANG Identities — the Extensible Enum

A regular `enumeration` is closed: only the listed enums are valid. **Identities** are an open hierarchy.

```yang
identity interface-type {
    description "Base for interface types.";
}
identity ethernetCsmacd {
    base interface-type;
}
identity softwareLoopback {
    base interface-type;
}

leaf type {
    type identityref {
        base interface-type;
    }
}
```

Another module can **add** more identities (`base interface-type`) without changing the original. The leaf accepts any identity descended from `interface-type`. This is how IANA's interface types (`iana-if-type`) extends OpenConfig's base.

## RESTCONF Deep ELI5

RESTCONF is **the REST/JSON skin on the same YANG data NETCONF speaks**. Same models, same datastores, same validation — but instead of XML over SSH, it's HTTP and (usually) JSON. Specified in **RFC 8040**.

If your team is already comfortable with `curl`, OpenAPI, REST, OAuth — RESTCONF is your easiest on-ramp to model-driven networking.

### URI Structure

The URI is **derived directly from the YANG path**.

```
https://<host>:<port>/restconf/data/<module>:<container>/<list>=<key>/<leaf>
```

Concrete examples:

```
GET /restconf/data/openconfig-interfaces:interfaces
GET /restconf/data/openconfig-interfaces:interfaces/interface=eth0
GET /restconf/data/openconfig-interfaces:interfaces/interface=eth0/config/mtu
GET /restconf/data/openconfig-interfaces:interfaces/interface=eth0/state/counters/in-octets
```

Path components map cleanly:

| YANG concept | URI form |
|--------------|----------|
| Top container `interfaces` in module `openconfig-interfaces` | `/openconfig-interfaces:interfaces` |
| List `interface` inside that container | `/interface` |
| List entry where `name=eth0` | `/interface=eth0` |
| Multi-key list with key1=A, key2=B | `/list=A,B` (comma-separated, percent-escaped) |
| Leaf inside | `/mtu` |

### HTTP Methods

| Method | Effect |
|--------|--------|
| `GET` | Read config and/or state |
| `HEAD` | Like GET but no body |
| `POST` | Create a new resource (list entry or leaf-list value) — fails if it exists |
| `PUT` | Replace a resource fully (create-or-replace semantics) |
| `PATCH` | Merge into existing — partial update (RFC 8040 uses Plain Patch) |
| `DELETE` | Remove a resource |
| `OPTIONS` | Discover allowed methods |

### Content-Types

Two main encodings:

- `application/yang-data+json` — JSON-IETF (RFC 7951)
- `application/yang-data+xml` — same XML as NETCONF

Always send `Accept:` and `Content-Type:` headers.

```bash
curl -H "Accept: application/yang-data+json" \
     -H "Content-Type: application/yang-data+json" \
     -u admin:redacted \
     https://router.example.com/restconf/data/openconfig-interfaces:interfaces
```

### Datastores Under NMDA (RFC 8342)

RESTCONF (per RFC 8527) supports the NMDA datastores:

- `running` — what's been configured
- `intended` — what was requested (after templates expanded, defaults applied)
- `operational` — what the device is actually doing (config + state + counters)
- `candidate` — scratchpad (like NETCONF)
- `startup` — persists across reboot

To target a specific datastore, prefix with `/ds/<name>`:

```
GET /restconf/data/openconfig-interfaces:interfaces                # default = running
GET /restconf/ds/ietf-datastores:operational/openconfig-interfaces:interfaces
GET /restconf/ds/ietf-datastores:running/openconfig-interfaces:interfaces
GET /restconf/ds/ietf-datastores:candidate/openconfig-interfaces:interfaces
```

### Query Parameters

- `?depth=N` — limit tree depth in response (e.g. `?depth=2` returns only 2 levels)
- `?fields=foo;bar` — return only these fields (semicolon-separated)
- `?content=config` — only configurable nodes
- `?content=nonconfig` — only operational state
- `?content=all` — both (default)
- `?with-defaults=report-all` — include leafs at their default values
- `?with-defaults=trim` — omit leafs that are at default
- `?with-defaults=explicit` — only leafs explicitly set
- `?filter=<xpath>` — XPath filter (on event streams)

Examples:

```bash
# only the names of all interfaces
curl ".../restconf/data/openconfig-interfaces:interfaces?fields=interface/name"

# operational state only, depth 3
curl ".../restconf/data/openconfig-interfaces:interfaces?content=nonconfig&depth=3"
```

### Concrete `curl` Examples

**GET all interfaces:**

```bash
curl -s -u admin:redacted \
  -H "Accept: application/yang-data+json" \
  https://router.example.com/restconf/data/openconfig-interfaces:interfaces
```

**GET a single interface:**

```bash
curl -s -u admin:redacted \
  -H "Accept: application/yang-data+json" \
  https://router.example.com/restconf/data/openconfig-interfaces:interfaces/interface=GigabitEthernet0%2F0%2F0%2F1
```

Note the `%2F` — the slashes in the interface name `GigabitEthernet0/0/0/1` must be percent-encoded.

**PATCH the MTU of an existing interface:**

```bash
curl -s -u admin:redacted \
  -X PATCH \
  -H "Content-Type: application/yang-data+json" \
  -d '{"openconfig-interfaces:config":{"mtu":9000}}' \
  https://router.example.com/restconf/data/openconfig-interfaces:interfaces/interface=GigabitEthernet0%2F0%2F0%2F1/config
```

PATCH merges — anything not specified stays as-is.

**PUT replaces the entire config block:**

```bash
curl -s -u admin:redacted \
  -X PUT \
  -H "Content-Type: application/yang-data+json" \
  -d '{
    "openconfig-interfaces:config": {
      "name": "GigabitEthernet0/0/0/1",
      "type": "iana-if-type:ethernetCsmacd",
      "mtu": 9000,
      "description": "uplink to spine-1",
      "enabled": true
    }
  }' \
  https://router.example.com/restconf/data/openconfig-interfaces:interfaces/interface=GigabitEthernet0%2F0%2F0%2F1/config
```

PUT is "replace the whole config block with this." Any leaf you omit goes back to default (or disappears if not mandatory).

**POST creates a new interface (fails if `interface=lo10` already exists):**

```bash
curl -s -u admin:redacted \
  -X POST \
  -H "Content-Type: application/yang-data+json" \
  -d '{
    "openconfig-interfaces:interface": [{
      "name": "Loopback10",
      "config": {
        "name": "Loopback10",
        "type": "iana-if-type:softwareLoopback",
        "description": "BGP source",
        "enabled": true
      }
    }]
  }' \
  https://router.example.com/restconf/data/openconfig-interfaces:interfaces
```

If `Loopback10` already exists, the server returns `409 Conflict` with a YANG error payload.

**DELETE an interface:**

```bash
curl -s -u admin:redacted \
  -X DELETE \
  https://router.example.com/restconf/data/openconfig-interfaces:interfaces/interface=Loopback10
```

**POST a new ACL (calling an RPC under `/operations`):**

RPC operations (YANG `rpc` statements) live under `/restconf/operations/<module>:<rpc-name>`:

```bash
curl -s -u admin:redacted \
  -X POST \
  -H "Content-Type: application/yang-data+json" \
  -d '{"input": {"target": "candidate"}}' \
  https://router.example.com/restconf/operations/ietf-netconf:lock
```

### Real JSON-IETF Payload Format

JSON-IETF is **not** standard JSON. Two big rules:

1. **Top-level keys are namespace-qualified**: `"openconfig-interfaces:interfaces"`, not `"interfaces"`.
2. **64-bit integers are strings**: `"in-octets": "12345678901234"`, not `12345678901234`.

Full real-shaped reply to `GET /restconf/data/openconfig-interfaces:interfaces/interface=eth0`:

```json
{
  "openconfig-interfaces:interface": [
    {
      "name": "eth0",
      "config": {
        "name": "eth0",
        "type": "iana-if-type:ethernetCsmacd",
        "mtu": 1500,
        "description": "uplink",
        "enabled": true
      },
      "state": {
        "name": "eth0",
        "type": "iana-if-type:ethernetCsmacd",
        "mtu": 1500,
        "admin-status": "UP",
        "oper-status": "UP",
        "ifindex": 12,
        "counters": {
          "in-octets": "12345678901234",
          "out-octets": "98765432109876",
          "in-errors": "0",
          "out-errors": "0"
        }
      },
      "subinterfaces": {
        "subinterface": [
          {
            "index": 0,
            "openconfig-if-ip:ipv4": {
              "addresses": {
                "address": [
                  {
                    "ip": "192.0.2.1",
                    "config": { "ip": "192.0.2.1", "prefix-length": 24 }
                  }
                ]
              }
            }
          }
        ]
      }
    }
  ]
}
```

Notice `openconfig-if-ip:ipv4` mid-tree — that's an **augmentation** from a different module, namespace-qualified because it's not the same module as `openconfig-interfaces`.

### Differences from NETCONF

| | NETCONF | RESTCONF |
|---|---|---|
| Transport | SSH (port 830) | HTTPS (port 443 typically) |
| Encoding | XML | JSON or XML |
| Auth | SSH keys / passwords | HTTP Basic / OAuth / cert |
| Datastore lock | `<lock>` RPC | None (rely on transactional PATCH/PUT) |
| Multi-RPC transaction | `<commit>` after multiple `<edit-config>` | Each HTTP request is its own transaction (yang-patch helps) |
| Streaming notifications | NETCONF notifications (RFC 5277) | HTTP event streams (SSE) |
| Capability discovery | `<hello>` | `GET /restconf/data/ietf-yang-library:modules-state` |
| Tooling | ncclient, netconf-console | curl, requests, Postman |

### When to Use Which

- **RESTCONF** when your team thinks in REST/JSON, you want simple `curl`-able operations, and you don't need NETCONF's transactional multi-edit semantics or candidate datastore. Best for "set the MTU on this interface" one-shots, OpenAPI integration, web-tier orchestration.

- **NETCONF** when you need explicit datastore locking, candidate-then-commit workflows, multi-edit transactions that succeed or fail atomically, or you're talking to older devices that only speak NETCONF. Best for big-batch changes, rollback workflows, and any environment where two operators might step on each other.

- **gNMI** (covered in Part 3) when you need streaming telemetry (Subscribe RPC), high-rate state collection, or Google-flavored model-driven config. Best for telemetry pipelines, large-scale data-center networks, modern day-2 ops.

A practical rule of thumb: **read state with gNMI Subscribe, push config with NETCONF or RESTCONF, model everything in YANG.** Hybrid stacks are normal.

## gNMI / gNOI Deep ELI5

OK, so here's where we leave the 2002-era XML-over-SSH world (NETCONF) and step into the 2018-and-beyond world. **gNMI** stands for **gRPC Network Management Interface**. Let's unpack every word of that.

- **gRPC** is Google's Remote Procedure Call framework. It rides on top of HTTP/2, uses Protocol Buffers (protobuf) as its data format, and is built for low-latency, high-throughput, streaming-friendly communication. Think "the call you'd make if you were designing for streaming telemetry from 2018 onward, not for stateful XML-over-SSH from 2006."
- **Network** because, well, it talks to network devices.
- **Management Interface** because it's the protocol you use to GET state, SET config, and SUBSCRIBE to streaming updates.

It was defined by **OpenConfig** (the working group, more on that later) and the canonical reference implementation lives at `github.com/openconfig/gnmi`. The protobuf definitions live in that repo as `proto/gnmi/gnmi.proto`. Everything ultimately compiles down to those proto definitions, so a client in Go, Python, Rust, or Java all speak the same wire format.

### Why gNMI when we already had NETCONF

NETCONF is fine for what it is, but:

- **XML is heavy.** A single interface counter snapshot in XML can be 10x the size of the same data in protobuf.
- **SSH per session adds round-trip latency.** Every NETCONF session does an SSH handshake first. That's ~3 RTTs before you even speak NETCONF.
- **No real streaming.** NETCONF has notifications (RFC 5277) but they're not optimized for high-volume telemetry. Vendors bolted them on, and they show.
- **gRPC is the modern stack.** Your Kubernetes control plane, your service mesh, your distributed databases — all gRPC. Network management not being gRPC was the odd one out.

So gNMI came along and said "let's do management with the same RPC framework as everything else, encode in protobuf for size, use HTTP/2 for streaming and multiplexing, and reuse the YANG models we already have."

### Path syntax — the gNMI XPath

Every operation in gNMI specifies a **path**, which is gNMI's address scheme into the YANG data tree. It looks like a stripped-down XPath:

```
/interfaces/interface[name=Ethernet1]/state/counters/in-octets
```

Reading it left to right:

- `/interfaces` — the top-level YANG container called `interfaces`
- `/interface[name=Ethernet1]` — the list entry whose `name` key equals `Ethernet1`
- `/state` — the read-only state container (vs `/config` which is read-write)
- `/counters` — the counters subcontainer
- `/in-octets` — the leaf you want

Wildcards are allowed:

- `/interfaces/interface[name=*]/state/counters/in-octets` — counters for every interface
- `/interfaces/interface[name=Ethernet*]/state` — state for every interface whose name starts with `Ethernet`

A path can have a **prefix** so you don't repeat yourself across many subscriptions. The prefix and path concatenate.

### The four operations

gNMI defines exactly four RPCs in its proto:

1. **Capabilities** — "what models do you support, what versions, what encodings?"
2. **Get** — "give me a one-shot snapshot of this path right now."
3. **Set** — "change config at these paths." Set takes three list arguments: `delete`, `replace`, and `update`. Replace is a clobber-and-replace at the path; update is a merge.
4. **Subscribe** — "stream updates to me."

Subscribe is the killer feature. It's why gNMI exists at all.

### Subscription modes

When you Subscribe, you pick a **mode**:

- **ONCE** — server sends a one-shot snapshot of every path in the subscription, then closes. Useful when you want a consistent moment-in-time view of multiple paths together (Get is per-path).
- **POLL** — server holds the connection open; you ask the client lib to send a poll request, server replies with current values. This is rare in practice — if you're polling, just use Get.
- **STREAM** — server pushes updates indefinitely. This is the production mode.

STREAM has three **sub-modes** per subscription path:

- **SAMPLE** — server sends the value at a fixed cadence (`sample_interval`, in nanoseconds). For counters this is the workhorse: `5s`, `10s`, `30s` are typical.
- **ON_CHANGE** — server sends only when the value changes. Perfect for state leafs (link up/down, BGP session state, OSPF neighbor state). Don't use this for counters that change every microsecond — you'll DDoS yourself.
- **TARGET_DEFINED** — "you decide, server." The server picks SAMPLE or ON_CHANGE per path based on the leaf type. Convenient but less predictable for capacity planning.

You can also set `suppress_redundant: true` (don't resend a SAMPLE value if it's identical to last time) and `heartbeat_interval` (resend at least every N seconds, even if unchanged or suppressed, so the client knows you're alive).

### Encodings

gNMI lets you negotiate the encoding of leaf values:

- **JSON_IETF** — RFC 7951 JSON encoding of YANG. The most common choice, the most interop-friendly. This is what you'll see 90% of the time.
- **JSON** — a less-strict JSON encoding (uses YANG type defaults, doesn't always namespace-qualify).
- **BYTES** — raw bytes. For leaves of type `binary`. Niche.
- **PROTO** — protobuf-encoded values. The smallest on the wire and zero-copy on both ends, but you need the .proto definitions for the model. Cisco IOS-XR and some others support this for max throughput.
- **ASCII** — vendor CLI text dropped into a value. Some vendors expose this for legacy reasons (e.g., `show running-config` text). Avoid for automation; it's a regex trap.

Pick **JSON_IETF** unless you have a specific reason. PROTO is great if you're scaling to thousands of subscriptions per device — you'll spend 50% less CPU encoding/decoding.

### Authentication

gNMI runs over HTTP/2 over TLS by default. You authenticate with:

- **TLS server cert** (always, in production) — the device proves its identity to you.
- **TLS mutual auth (mTLS)** — you also present a client cert. Common in production fleets with a real PKI.
- **Per-RPC username/password** — sent as gRPC metadata on each call. Vendors usually integrate this with their AAA / TACACS+ / RADIUS stack.
- **Insecure mode** — `--insecure` or `--skip-verify`. Lab only. Never production.

A typical production setup is mTLS plus AAA usernames, so you have both transport-level and identity-level auth.

### gnmic — the canonical CLI

`gnmic` (https://gnmic.openconfig.net) is the OpenConfig project's official command-line gNMI client. It's written in Go, ships as a single binary, and is the fastest way to learn gNMI hands-on. Install with `brew install gnmic` or grab a release binary.

A typical config file (`~/.gnmic.yml`) looks like:

```yaml
username: admin
password: admin
insecure: true
encoding: json_ietf
targets:
  192.0.2.10:6030:
    name: ar1
  192.0.2.11:6030:
    name: ar2
```

Some bread-and-butter commands. Capabilities first — always. It tells you what models the device knows:

```bash
gnmic -a 192.0.2.10:6030 capabilities
```

Output looks like:

```
gNMI version: 0.7.0
supported models:
  - openconfig-interfaces, OpenConfig working group, 2.4.3
  - openconfig-bgp, OpenConfig working group, 6.1.0
  - arista-intf-augments, Arista Networks, Inc., 2.7.0
  ...
supported encodings:
  - JSON
  - JSON_IETF
  - ASCII
```

Get a snapshot:

```bash
gnmic -a 192.0.2.10:6030 get \
  --path /interfaces/interface[name=Ethernet1]/state
```

Output is JSON like:

```json
[
  {
    "source": "192.0.2.10:6030",
    "timestamp": 1714200234123456789,
    "time": "2024-04-27T10:23:54.123456789Z",
    "updates": [
      {
        "Path": "interfaces/interface[name=Ethernet1]/state",
        "values": {
          "interfaces/interface/state": {
            "admin-status": "UP",
            "oper-status": "UP",
            "counters": {
              "in-octets": "12345678901",
              "out-octets": "98765432101",
              "in-errors": "0"
            }
          }
        }
      }
    ]
  }
]
```

Subscribe in STREAM/SAMPLE mode for live counters:

```bash
gnmic -a 192.0.2.10:6030 subscribe \
  --path /interfaces/interface[name=Ethernet1]/state/counters \
  --stream-mode sample \
  --sample-interval 5s
```

Subscribe in STREAM/ON_CHANGE for BGP session state:

```bash
gnmic -a 192.0.2.10:6030 subscribe \
  --path "/network-instances/network-instance[name=default]/protocols/protocol/bgp/neighbors/neighbor[neighbor-address=*]/state/session-state" \
  --stream-mode on-change
```

Set a description (single update):

```bash
gnmic -a 192.0.2.10:6030 set \
  --update /interfaces/interface[name=Ethernet1]/config/description:::string:::"updated by gnmic"
```

The `:::string:::` is gnmic's syntax for `<path>:::<type>:::<value>` so it can encode the value correctly.

Set with a JSON file (more typical for batch):

```bash
gnmic -a 192.0.2.10:6030 set \
  --update-path /interfaces/interface[name=Ethernet1]/config \
  --update-file ./eth1-config.json
```

Where `eth1-config.json` contains:

```json
{
  "description": "uplink to spine1",
  "enabled": true,
  "mtu": 9214
}
```

Replace (full clobber) of the BGP neighbors list — useful for declarative management:

```bash
gnmic -a 192.0.2.10:6030 set \
  --replace-path /network-instances/network-instance[name=default]/protocols/protocol/bgp/neighbors \
  --replace-file ./neighbors.json
```

Delete:

```bash
gnmic -a 192.0.2.10:6030 set \
  --delete /interfaces/interface[name=Ethernet1]/config/description
```

### gNOI — the operational sibling

gNMI handles config and state. **gNOI** (gRPC Network Operations Interface) handles everything else — the actions you'd otherwise telnet in to do. Same gRPC framework, same TLS, separate proto definitions in `github.com/openconfig/gnoi`.

The big gNOI services:

- **gnoi.system** — `Reboot`, `RebootStatus`, `CancelReboot`, `Ping`, `Traceroute`, `Time`, `SetPackage`, `SwitchControlProcessor`. Yes, you can `Reboot` a router via RPC.
- **gnoi.os** — `Install` (push a new image), `Activate` (boot to new image), `Verify` (confirm what's running). The image push is a streaming RPC — chunks of the image come over gRPC, the device reassembles and verifies.
- **gnoi.file** — `Get`, `Put`, `Stat`, `Remove`. File transfer over gRPC. Replaces TFTP/SCP/FTP.
- **gnoi.cert** — `Install`, `Rotate`, `RevokeCertificates`, `GetCertificates`, `CanGenerateCSR`, `GenerateCSR`. PKI lifecycle ops.
- **gnoi.factory_reset** — `Start`. Wipes and resets to factory.
- **gnoi.healthz** — `Get`, `List`, `Acknowledge`, `Artifact`, `Check`. Modern health check / "what's wrong with this device" interface.

A `gnoic` CLI exists as the gNOI counterpart to gnmic:

```bash
gnoic -a 192.0.2.10:6030 system reboot --message "scheduled maintenance"
gnoic -a 192.0.2.10:6030 system ping --destination 8.8.8.8 --count 5
gnoic -a 192.0.2.10:6030 file get --remote-file /mnt/flash/startup-config --local-file ./startup-config
gnoic -a 192.0.2.10:6030 os install --version 4.30.2F --pkg /tmp/EOS-4.30.2F.swi
```

### The gNxI family

You'll see these abbreviations together. The "x" varies, but they all live under github.com/openconfig:

- **gNMI** — config, state, telemetry (what we've been describing).
- **gNOI** — operations (reboot, ping, file, cert, OS install).
- **gNSI** — gRPC Network Security Interface — auth, authz, accounting, credential rotation, RBAC. The newest member.
- **gRIBI** — gRPC Routing Information Base Interface — programmable RIB. Push routes into the device's RIB from a controller, with strict semantics around AFT (Abstract Forwarding Tables). Used by hyperscalers for SDN-style traffic engineering.

You can think of them as four protocols sharing one transport (gRPC), one auth model (TLS+mTLS), one workflow (declarative + streaming), and four different scopes (config/state, ops, security, routing).

## Model-Driven Telemetry

OK, so the old way to get a counter off a router was **SNMP polling**. Your monitoring server sends `GET-REQUEST` UDP packets every 60 seconds asking "what's the in-octets on Ethernet1?" The router replies. You graph the delta. Repeat.

This worked in 1995. It does not work in 2024. Three reasons:

- **Too slow.** 60s poll interval is the sweet spot SNMP can sustain at scale; you can't see the 5-second microbursts that crash a TCP session.
- **Doesn't scale.** Every poll is a separate round trip. With 10,000 devices and 100 OIDs per device, your poller is doing a million round trips per cycle. Network and CPU both melt.
- **It's pull.** The device only knows you wanted data when you ask. It can't tell you proactively when something interesting just happened.

**Model-Driven Telemetry (MDT)** flips it: the device **pushes** structured, model-defined data to a collector. The "model-driven" part means the data is shaped by a YANG model, not a vendor-specific MIB schema. You get a typed object stream, not OID strings.

### Push vs pull, in one image

Pull (SNMP):
```
collector --GET in-octets--> router
router   <--12345----------- collector
[wait 60s]
collector --GET in-octets--> router
router   <--12350----------- collector
```

Push (MDT/gNMI streaming):
```
collector <--SUBSCRIBE----- router
router    --in-octets=12345-->
router    --in-octets=12348--> [5s later]
router    --in-octets=12353--> [5s later]
... forever ...
```

One subscription, one TCP/HTTP/2 stream, deltas pushed at whatever interval you want. You set up the subscription once and the device does the rest.

### gNMI streaming as the modern push model

Almost every modern MDT implementation rides gNMI Subscribe (STREAM mode). The device acts as the gRPC server; your collector is the gRPC client that opens a long-lived subscription. Some legacy MDT (Cisco's pre-gNMI "Telemetry over gRPC" using cisco-grpc-dialin/dialout) is its own thing, but the industry is converging on gNMI.

There are two shapes:

- **dial-in (gNMI Subscribe)** — collector connects to device. Standard. Usually what you want.
- **dial-out** — device connects to collector. Same data on the wire, just reversed TCP direction. Useful through restrictive NAT/firewalls where the device is behind something the collector can't reach.

### Cisco IOS-XR sensor groups + subscriptions

IOS-XR's MDT config maps onto the gNMI concept like this. You define a **sensor-group** (paths to stream), a **destination-group** (where to send), and a **subscription** (link the two with a cadence).

```
telemetry model-driven
 sensor-group SG-INTERFACES
  sensor-path openconfig-interfaces:interfaces/interface/state/counters
 !
 sensor-group SG-BGP
  sensor-path openconfig-network-instance:network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state
 !
 destination-group DG-COLLECTOR
  address-family ipv4 10.0.0.50 port 57400
   encoding self-describing-gpb
   protocol grpc no-tls
  !
 !
 subscription SUB-INTERFACES
  sensor-group-id SG-INTERFACES sample-interval 5000
  destination-id DG-COLLECTOR
 !
 subscription SUB-BGP
  sensor-group-id SG-BGP sample-interval 0
  destination-id DG-COLLECTOR
 !
!
```

Notes:

- `sample-interval 5000` is milliseconds — 5 seconds.
- `sample-interval 0` means **on-change** in IOS-XR's dialect.
- `self-describing-gpb` is GPB (Google Protobuf) with the schema embedded in each message; lighter alternatives are `compact-gpb` (smaller, but you need the .proto on the collector side) and `json`.
- `grpc no-tls` is for lab; production should be `protocol grpc tls` plus a trustpoint.

### Junos OpenConfig telemetry

Junos uses a slightly different config style. Streaming telemetry config goes under `services` and `protocols`:

```
set system services extension-service request-response grpc clear-text port 32767
set system services extension-service notification allow-clients address 10.0.0.50/32
set services analytics streaming-server collector1 remote-address 10.0.0.50
set services analytics streaming-server collector1 remote-port 50051
set services analytics export-profile prof1 reporting-rate 5
set services analytics export-profile prof1 transport grpc
set services analytics export-profile prof1 format gpb
set services analytics sensor s1 server-name collector1
set services analytics sensor s1 export-name prof1
set services analytics sensor s1 resource /interfaces/interface/state/counters/
```

Junos also supports gNMI Subscribe directly when you enable the JTI (Junos Telemetry Interface) gRPC service — that's the dial-in flavor.

### Arista EOS streaming telemetry

EOS has the cleanest config — gNMI is just a config block under management:

```
management api gnmi
   transport grpc default
      port 6030
      ssl profile gnmi-ssl
   provider eos-native
```

Then your collector opens a gNMI Subscribe to port 6030 and you're done. Subscriptions are entirely client-side — no sensor-groups to define on the device. EOS will stream whatever path you ask for.

For high-cardinality state, EOS also exposes the OpenConfig models plus `eos-native` paths (a richer, EOS-specific schema). Use OpenConfig for portable dashboards, eos-native for deep visibility.

### Consuming on the collector side

A few common stacks for "where does the streaming data go":

**TIG — Telegraf + InfluxDB + Grafana.** The community standard for network telemetry.

- Telegraf has a `cisco_telemetry_mdt` input plugin and a `gnmi` input plugin. It receives the stream, transforms it into time-series, and writes to InfluxDB.
- InfluxDB stores the time-series, indexed by tag.
- Grafana queries InfluxDB and renders dashboards.

A minimal Telegraf gNMI config:

```toml
[[inputs.gnmi]]
  addresses = ["192.0.2.10:6030", "192.0.2.11:6030"]
  username = "admin"
  password = "admin"
  encoding = "json_ietf"
  redial = "10s"

  [[inputs.gnmi.subscription]]
    name = "ifcounters"
    origin = "openconfig-interfaces"
    path = "/interfaces/interface/state/counters"
    subscription_mode = "sample"
    sample_interval = "5s"

[[outputs.influxdb_v2]]
  urls = ["http://influxdb:8086"]
  token = "$INFLUX_TOKEN"
  organization = "netops"
  bucket = "telemetry"
```

**Prometheus + gnmi-prometheus-exporter (or gnmic's prometheus output).** If your shop is Prometheus-native:

- gnmic itself can act as a Prometheus exporter (`gnmic --config gnmic.yaml subscribe` with the `prometheus` output plugin).
- Or run a dedicated `gnmi-prometheus-exporter`.

A snippet of gnmic acting as a Prometheus output:

```yaml
outputs:
  prom-output:
    type: prometheus
    listen: ":9804"
    path: /metrics
    metric-prefix: gnmic
    append-subscription-name: true
    expiration: 60s
```

**Kafka pipeline pattern.** When you have hundreds of collectors and a real data platform downstream:

```
device --gNMI--> gnmic (collector) --kafka--> Kafka topic --consumer 1--> InfluxDB
                                                          --consumer 2--> ClickHouse
                                                          --consumer 3--> alerting service
                                                          --consumer 4--> SIEM
```

gnmic has a Kafka output. This is the pattern hyperscalers use because it decouples collection from consumption.

### Suggested cadences

Not every metric needs the same sample rate. Some rough rules:

- **Counters that aggregate cleanly (in-octets, in-pkts):** 30s. You're going to graph 5-minute means anyway.
- **QoS counters, latency-sensitive paths, microburst detection:** 1s or even 100ms.
- **State leafs (admin-status, oper-status, BGP session-state, OSPF neighbor-state):** ON_CHANGE. There's no reason to sample a leaf that changes once a week.
- **Environmental (temp, fan, PSU):** 60s. They don't change fast.
- **Memory, CPU:** 10s. Fast enough to catch leaks, slow enough not to drown you.

### Storage cost tradeoffs

Telemetry can crush your TSDB if you're not careful:

- **Cardinality explosion.** Every unique combination of tag values creates a new time series. A device with 96 interfaces, each with 8 queues, each with 4 counter types, sampled at 1s... do the math. Your TSDB will OOM by lunch.
- **Mitigation: keep tag values bounded.** Don't tag with timestamps. Don't tag with traffic source IPs.
- **Downsampling.** Keep 1s data for 1 day, downsample to 30s for 30 days, downsample to 5min for 1 year. InfluxDB has continuous queries, Prometheus has recording rules, ClickHouse has materialized views.
- **Retention policies.** Set them or your disk fills.

## OpenConfig vs Native Models

### History

In 2014, Google, AT&T, Microsoft, and a few other operators looked at their multi-vendor networks and went "this is impossible." Every vendor had its own YANG models (or worse, only had MIBs). Writing a single tool to configure both a Cisco and a Juniper required two completely separate codepaths. They formed the **OpenConfig working group** to write **vendor-neutral YANG models** for everything: interfaces, BGP, OSPF, LACP, system, AAA, you name it.

The output lives at github.com/openconfig/public — hundreds of YANG modules with names like `openconfig-interfaces`, `openconfig-network-instance`, `openconfig-bgp`, `openconfig-system`. Vendors agreed (eventually, mostly) to implement them.

### Common-base + vendor-augments

OpenConfig models are designed as a **common base**. A leaf like `/interfaces/interface[name=*]/state/counters/in-octets` looks identical on Arista, Cisco, Juniper, Nokia. That's the dream.

Reality is messier. Vendors have features OpenConfig doesn't model (yet, or ever — proprietary stuff that won't standardize). So OpenConfig allows **augmentation**:

```
augment "/oc-if:interfaces/oc-if:interface" {
  leaf vendor-special-knob {
    type string;
  }
}
```

Now `/interfaces/interface[name=*]/vendor-special-knob` exists on devices supporting that augment. Your portable dashboards ignore it; your vendor-specific tooling can use it.

### The ugly truth

Vendors implemented OpenConfig with varying levels of fidelity. Common pain points:

- **Missing leaves.** A given leaf in the OpenConfig model just isn't implemented on vendor X yet (or maybe ever). You query it and get nothing.
- **Different units.** OpenConfig says counters are uint64 octets; one vendor returned them as a stringified decimal because of a JSON encoding quirk; another returned 32-bit and wrapped at 4GB.
- **Subtly different semantics.** "BGP session-state ESTABLISHED" should mean the same thing everywhere. It mostly does. Sometimes it doesn't, especially around timing of transitions.
- **Stale model versions.** Vendor X is on `openconfig-interfaces@2018-02-20`; vendor Y is on `@2024-01-15`. Some leaves only exist in newer versions.

So OpenConfig gives you 80-90% portability, not 100%. Plan for the gap.

### When to use OpenConfig vs native

**Use OpenConfig when:**

- You have a multi-vendor network (any non-trivial enterprise does).
- You're building portable tooling (a single Python script that works against any vendor).
- You want to future-proof against vendor swaps.
- You're doing telemetry — OpenConfig telemetry paths are where the industry is converging.

**Use native when:**

- You need a feature that's only in the vendor model (proprietary QoS knobs, vendor-specific MPLS-TE features, EVPN extensions).
- You're working in a single-vendor shop and don't expect to change.
- You need exact bit-for-bit fidelity with the CLI for compliance or audit reasons.

In practice, real shops use **OpenConfig for the 90% common path and native for the 10% specialty**. Your tooling has a thin shim that picks the right model per leaf.

### Example: BGP container

OpenConfig BGP (simplified):

```
openconfig-network-instance:
  network-instances:
    network-instance:
      - name: default
        protocols:
          protocol:
            - identifier: BGP
              name: bgp
              bgp:
                global:
                  config:
                    as: 65000
                    router-id: 10.0.0.1
                neighbors:
                  neighbor:
                    - neighbor-address: 10.0.0.2
                      config:
                        peer-as: 65001
```

Cisco IOS-XE native BGP (Cisco-IOS-XE-bgp-oper / Cisco-IOS-XE-bgp):

```
Cisco-IOS-XE-bgp:
  bgp:
    bgp-no-instance:
      id: 65000
      router-id: 10.0.0.1
      neighbor:
        - id: 10.0.0.2
          remote-as: 65001
```

Same intent, different paths, different field names. OpenConfig has `bgp/global/config/as`, Cisco native has `bgp/bgp-no-instance/id`. A multi-vendor shim would have to know that mapping per vendor.

### pyang for inspecting models

`pyang` is a YANG parser/validator/translator from the same era as NETCONF. It's invaluable for inspecting model differences:

```bash
# show the tree of a YANG file (ASCII tree of containers and leaves)
pyang -f tree openconfig-interfaces.yang

# limit depth
pyang -f tree --tree-depth=3 openconfig-interfaces.yang

# show the namespace and module dependencies
pyang -f tree --tree-print-yang-data openconfig-interfaces.yang

# show namespace and prefix info specifically
pyang -f info openconfig-interfaces.yang
```

The `info` format prints things like:

```
module:           openconfig-interfaces
namespace:        http://openconfig.net/yang/interfaces
prefix:           oc-if
yang-version:     1.1
organization:     OpenConfig working group
```

### Translation strategies

When you do need multi-vendor portability and OpenConfig isn't 100% there:

- **Model-driven shim.** A library (often per-tool: NSO has it, Ansible has its OpenConfig roles, custom Python often grows one) that maps "tool's intended OpenConfig leaf" to "vendor's actual model+leaf." Maintains a per-vendor translation table.
- **Multi-pass commit.** First pass: write everything that's in OpenConfig. Second pass: write vendor-native config for the missing leaves. Some shops codify this as two playbook layers.
- **Intent-based abstraction.** Model your intent at a higher level (e.g., "configure VLAN 100 on these ports as access") and let a per-vendor renderer turn that into the right model+leaves. NSO and Nornir patterns do this.
- **NETCONF subtree filter + XSLT** for the XML-native crowd. Old-school but still works.

## Ansible for Network ELI5

OK, Ansible. You probably know Ansible from server-land — SSH into a Linux box, run a Python module, capture results. Networks were a bolt-on for years, and that bolt-on has matured a lot.

The core problem: most network devices don't run Python. You can't ship a Python module to an IOS box, run it locally, and capture stdout. Instead, Ansible runs the module **on the controller** and uses a connection plugin to push commands or RPCs to the device.

### Connection plugins

The big three:

- **`ansible_connection=network_cli`** — for vendors with a traditional CLI you'd otherwise SSH to. Ansible opens an SSH session, runs commands, screen-scrapes (or uses structured output if available). This is the workhorse for IOS, IOS-XE, NX-OS-CLI, EOS-CLI, IOS-XR-CLI, ASA, JUNOS-CLI.
- **`ansible_connection=netconf`** — for NETCONF/YANG. Uses ncclient under the hood. Junos, modern IOS-XE, Cisco IOS-XR.
- **`ansible_connection=httpapi`** — for REST APIs. NX-API on Nexus, eAPI on Arista, vManage, Panorama, Meraki, F5 iControl. The HTTP plugin handles auth and session reuse.

You set this in your inventory or host_vars, alongside `network_os`:

```yaml
# group_vars/cisco_ios.yml
ansible_connection: network_cli
ansible_network_os: ios
ansible_user: admin
ansible_password: admin
```

`network_os` valid values include `ios`, `iosxr`, `nxos`, `junos`, `eos`, `asa`, `panos`, and many more. Ansible uses this to pick the right module dispatcher.

### Collections

Modules ship in **collections** (Ansible's namespaced packaging from 2.9+). The big network ones:

- `cisco.ios` — IOS / IOS-XE
- `cisco.iosxr` — IOS-XR
- `cisco.nxos` — NX-OS
- `cisco.asa` — ASA
- `arista.eos` — Arista EOS
- `juniper.junos` — Junos OS
- `paloaltonetworks.panos` — PAN-OS firewalls
- `community.network` — long tail of vendors (Cumulus, ExtremeOS, ICX, etc.)
- `openconfig.openconfig` — OpenConfig-modeled roles for cross-vendor configuration

Install with `ansible-galaxy collection install cisco.ios` or list dependencies in `requirements.yml`:

```yaml
collections:
  - name: cisco.ios
    version: ">=4.0.0"
  - name: arista.eos
    version: ">=6.0.0"
  - name: juniper.junos
```

### Modules — the per-vendor toolbox

Within each collection there are modules for facts, raw commands, and resource-specific config. A non-exhaustive tour:

**`cisco.ios`:**
- `ios_facts` — gathers structured device facts (version, interfaces, neighbors, ARP).
- `ios_command` — runs arbitrary `show` commands and returns output.
- `ios_config` — pushes raw config lines (with idempotency via `parents` and `before`/`after`).
- `ios_l2_interfaces` — declarative L2 interface config (access/trunk, native VLAN).
- `ios_l3_interfaces` — declarative L3 interface config (IPv4/IPv6 addresses).
- `ios_vlans` — declarative VLAN database.
- `ios_acls` — declarative ACLs.
- `ios_bgp_global` / `ios_bgp_address_family` — declarative BGP.
- `ios_ospf_interfaces`, `ios_ospfv2`, `ios_ospfv3` — declarative OSPF.

**`juniper.junos`:**
- `junos_command` — run `show` commands.
- `junos_config` — load/commit Junos config (text, set, or XML format).
- `junos_facts` — facts.
- `junos_interfaces` — declarative interface config.
- `junos_l3_interfaces`, `junos_vlans`, `junos_lldp_interfaces`, etc.

**`arista.eos`:**
- `eos_command`, `eos_config`, `eos_facts` — same shape.
- `eos_interfaces`, `eos_l2_interfaces`, `eos_l3_interfaces`, `eos_vlans`.
- `eos_bgp_global`, `eos_bgp_address_family`.

**`cisco.nxos`:**
- `nxos_command`, `nxos_config`, `nxos_facts`.
- `nxos_acls`, `nxos_vlans`, `nxos_l3_interfaces`.
- `nxos_bgp_global`, `nxos_bgp_neighbor_address_family`.

### Idempotency: state values

Resource modules support `state` to control declarative behavior:

- **`state: merged`** — merge the supplied config into existing config. Default for most resource modules.
- **`state: replaced`** — replace the matching resource(s) with what's supplied. Anything not in your input is removed *for the resources you specified*.
- **`state: overridden`** — replace the entire feature. Like `replaced`, but for everything in that namespace, not just specified items.
- **`state: deleted`** — remove the specified config.
- **`state: gathered`** — read-only; populate facts about current state.
- **`state: rendered`** — build the CLI/RPC payload but don't push (useful for review or offline rendering).
- **`state: parsed`** — parse a provided config blob into the model (useful for migration tooling).

Example difference:

```yaml
# merged: only ensures the supplied keys; existing other config stays
- cisco.ios.ios_l2_interfaces:
    config:
      - name: GigabitEthernet0/1
        access:
          vlan: 100
    state: merged

# replaced: replaces the per-interface block with exactly what's supplied
- cisco.ios.ios_l2_interfaces:
    config:
      - name: GigabitEthernet0/1
        access:
          vlan: 100
    state: replaced

# overridden: replaces ALL interfaces' L2 config with only what's supplied (others get reset)
- cisco.ios.ios_l2_interfaces:
    config:
      - name: GigabitEthernet0/1
        access:
          vlan: 100
    state: overridden
```

The semantics matter — `overridden` is dangerous if you forget to list every interface.

### OpenConfig Ansible roles

`openconfig.openconfig` (collection) provides cross-vendor roles like `openconfig.openconfig.system`, `openconfig.openconfig.bgp`, `openconfig.openconfig.interfaces`. You feed them OpenConfig-shaped data, they pick the right vendor backend based on `ansible_network_os`. Useful if you have a real multi-vendor estate and want one playbook.

### Playbook example: deploy VLANs to a fleet

```yaml
---
- name: Deploy VLANs to access switches
  hosts: cisco_access_switches
  gather_facts: false
  connection: ansible.netcommon.network_cli

  vars:
    site_vlans:
      - vlan_id: 10
        name: corp-data
      - vlan_id: 20
        name: voice
      - vlan_id: 30
        name: guest
      - vlan_id: 99
        name: mgmt

  tasks:
    - name: Ensure VLANs are present
      cisco.ios.ios_vlans:
        config: "{{ site_vlans }}"
        state: merged
      check_mode: true
      diff: true
      register: vlan_diff

    - name: Show what would change
      ansible.builtin.debug:
        var: vlan_diff.diff

    - name: Apply VLAN changes (if approved)
      cisco.ios.ios_vlans:
        config: "{{ site_vlans }}"
        state: merged
      when: not ansible_check_mode and apply_changes | default(false)
```

Run it:

```bash
# dry run with diff
ansible-playbook -i inventory.yml deploy-vlans.yml --check --diff

# real run
ansible-playbook -i inventory.yml deploy-vlans.yml -e apply_changes=true
```

### Inventory patterns

YAML inventory with groups:

```yaml
# inventory.yml
all:
  children:
    cisco_ios:
      hosts:
        sw1.dc1.example.com:
        sw2.dc1.example.com:
        sw3.dc1.example.com:
      vars:
        ansible_network_os: ios
        ansible_connection: ansible.netcommon.network_cli
    arista_eos:
      hosts:
        leaf1.dc1.example.com:
        leaf2.dc1.example.com:
      vars:
        ansible_network_os: eos
        ansible_connection: ansible.netcommon.network_cli
    juniper_junos:
      hosts:
        mx1.dc1.example.com:
      vars:
        ansible_network_os: junos
        ansible_connection: ansible.netcommon.netconf
    dc1:
      children:
        cisco_ios:
        arista_eos:
        juniper_junos:
```

`group_vars/cisco_ios.yml` would hold credentials, common config, etc. `host_vars/sw1.dc1.example.com.yml` holds host-specific data.

### Vault for credentials

Never check plaintext passwords into git. Ansible Vault encrypts files (or strings within files) with a passphrase:

```bash
# encrypt a whole file
ansible-vault encrypt group_vars/cisco_ios/credentials.yml

# edit it later (will prompt for vault pass)
ansible-vault edit group_vars/cisco_ios/credentials.yml

# encrypt a single string for inline use
ansible-vault encrypt_string 'SuperSecret123' --name 'ansible_password'
```

Output looks like:

```yaml
ansible_password: !vault |
  $ANSIBLE_VAULT;1.1;AES256
  64323464643839613561343532316464333664633765326565623462393238...
  ...
  3962653637353636333566363162646237393865323033306536386162616364
```

Then run with `--ask-vault-pass`, or point at a vault password file: `ansible-playbook --vault-password-file ~/.vault_pass ...`. In CI, store the vault pass as a secret env var.

### ansible-pull vs ansible-playbook

- **`ansible-playbook` (push)** — controller pushes to many targets. Standard. Best when controller has connectivity and credentials to all devices.
- **`ansible-pull` (pull)** — target itself runs Ansible against a local checkout of a playbook repo. The target must run Ansible (so this rarely works for network devices — they don't run Python). On Linux servers it's great for "node configures itself from git." For network gear, the analog is a controller that polls git and runs ansible-playbook on commit.

For network automation, you'll almost always use `ansible-playbook` from a controller, often wrapped by AWX/Tower/AAP or invoked from a CI pipeline.

### AWX / Tower / AAP

- **AWX** is the open-source upstream of Ansible Tower. Web UI, RBAC, scheduled jobs, credentials, surveys, audit log.
- **Ansible Tower** was the commercial product on top of AWX.
- **Ansible Automation Platform (AAP)** is the rebranded successor — Tower plus other components (event-driven Ansible, automation hub, etc.).

For network teams, AWX/AAP gives you:

- A single place where playbooks live, with version-pinned execution environments.
- Credential vault that injects creds at run time (not stored in inventory).
- RBAC so the NOC can run "show interface" jobs but not config jobs.
- Audit log: who ran what playbook against what hosts, when, with what result.
- Scheduling: nightly compliance scans, weekly health checks.

### Common gotchas

- **Connection caching.** `network_cli` keeps the SSH connection alive between tasks for performance. If the device drops the session (idle timeout), you'll see weird "device returned no output" errors. Tune `ansible_command_timeout` (default 30s) and `ansible_connect_timeout` (default 30s).
- **`become` is not applicable to network_cli by default.** Network devices don't have `sudo`. If you need to enter privileged exec mode, use `become: true` with `become_method: enable` and set `ansible_become_password`. Some platforms auto-enable; some don't.
- **Persistent connection timeout.** If a play stalls between tasks longer than the persistent connection timeout (default 30s), the next task triggers a reconnect. Set `ANSIBLE_PERSISTENT_CONNECT_TIMEOUT` and `ANSIBLE_PERSISTENT_COMMAND_TIMEOUT` env vars (or `[persistent_connection]` in ansible.cfg).
- **Idempotency lies.** `ios_config` with `lines:` is line-based, not model-based. It will sometimes re-push lines that the box has reformatted (e.g., your input has `ip address 10.0.0.1 255.0.0.0` but the running-config shows `ip address 10.0.0.1 /8`). Use resource modules (`ios_l3_interfaces`) for true idempotency.
- **`gather_facts: false`.** The default fact gathering tries to run the Linux setup module. On a network device, that hangs and times out. Always explicitly set `gather_facts: false` and use `*_facts` modules for vendor-specific facts.
- **Privilege levels.** Some IOS deployments restrict `privilege` level. If your account is privilege 1, half the modules will silently fail (or return weird "command not found" output).
- **Command output parsing.** When a vendor changes the `show` output format in a new release, screen-scraping breaks. Prefer structured outputs: `| json`, `| display xml`, gNMI Get, or netconf RPC over CLI screen-scraping wherever you can.
- **Diff output.** `--diff` works for resource modules but is hit-or-miss for raw `*_config`. The diff for raw config is "the lines I'm about to push," which isn't a real diff — it's an intent dump.
- **Check mode (`--check`).** Resource modules respect check mode well. Raw `*_command` and `*_config` modules that just push commands may not — they might still execute if the module wasn't written to honor check mode for that path.

## NAPALM, Nornir, Netmiko

Three Python libraries, three different jobs. Beginners conflate them. Don't. Each solves a different layer of the automation stack.

### Netmiko — the SSH plumber

**Netmiko** = "Network Multi-vendor SSH for Python". It's a thin wrapper around **Paramiko** (a pure-Python SSH client). Netmiko's job: log in to a device over SSH, send a command, capture the output. That's it. No magic, no abstraction, no "make it idempotent". You're writing a glorified expect script in Python.

**When you'd reach for Netmiko**: you have 50 routers, you want to run `show version` on every one and grep for the firmware string. Or you want to push a one-off config that doesn't fit Ansible's modules. Netmiko is the *low-level* tool — the duct tape.

It supports about 50 device types via "device_type" strings: `cisco_ios`, `cisco_nxos`, `cisco_xr`, `juniper_junos`, `arista_eos`, `paloalto_panos`, `f5_ltm`, `huawei`, `vyos`, `linux`, etc. The full list lives in the `netmiko/ssh_dispatcher.py` source.

Install:
```bash
pip install netmiko
```

A real session — connect to a switch and grab `show version`:

```python
from netmiko import ConnectHandler

device = {
    "device_type": "cisco_ios",
    "host":        "192.0.2.10",
    "username":    "admin",
    "password":    "secret",
    "secret":      "enable_secret",   # for `enable` mode
    "port":        22,
    "timeout":     30,
}

with ConnectHandler(**device) as conn:
    conn.enable()                         # drop into privileged-EXEC
    output = conn.send_command("show version")
    print(output)
```

`send_command` waits for the prompt to return before reading the output. That's how it knows the command finished — pattern-match the prompt regex. If the prompt regex is wrong (custom hostnames with weird chars), it'll hang until `read_timeout` fires. Override the prompt with `expect_string=r"#\s*$"` if needed.

Push config:

```python
cfg = [
    "vlan 100",
    " name USERS",
    "interface GigabitEthernet0/1",
    " switchport mode access",
    " switchport access vlan 100",
]

with ConnectHandler(**device) as conn:
    conn.enable()
    conn.config_mode()
    output = conn.send_config_set(cfg)    # enters config mode, sends each line
    conn.save_config()                    # write memory / commit
    print(output)
```

Netmiko gives you exactly what the device CLI gave you. You parse it. You decide if it succeeded. There's no "facts" abstraction, no "diff", no "candidate datastore". You're driving the CLI by hand, in Python.

Useful Netmiko helpers:

- `send_command_timing()` — for prompts mid-command (e.g. `reload` asking "are you sure?"). It uses `delay_factor` instead of prompt-regex.
- `send_command(use_textfsm=True)` — auto-parse with NTC-templates if available. Returns a list of dicts.
- `find_prompt()` — sanity-check what the device thinks its prompt is.
- `read_channel()` / `write_channel()` — raw read/write if you really want to drive things by hand.
- `disconnect()` — tear down the SSH session (or use the `with` context manager and don't worry).

### NAPALM — the cross-vendor abstraction

**NAPALM** = **N**etwork **A**utomation and **P**rogrammability **A**bstraction **L**ayer with **M**ultivendor support. Netmiko is the SSH plumber; NAPALM is the **API** that hides which vendor you're talking to.

You ask NAPALM `get_facts()` and you get back a dict — same shape on Cisco IOS, on Junos, on Arista EOS, on NX-OS. NAPALM's drivers handle the per-vendor quirks: maybe Junos uses NETCONF, Cisco uses Netmiko, Arista uses pyeapi. You don't care. You just call `get_facts()`.

Install:
```bash
pip install napalm
```

A real NAPALM session:

```python
from napalm import get_network_driver

driver = get_network_driver("ios")          # or "junos", "eos", "nxos_ssh", ...
device = driver(
    hostname="192.0.2.10",
    username="admin",
    password="secret",
    optional_args={"secret": "enable"},
)

device.open()

facts = device.get_facts()
# {'uptime': 12345, 'vendor': 'Cisco', 'os_version': '15.2(7)E', 'model': 'WS-C3850-48T', 'hostname': 'sw1', 'fqdn': 'sw1.example.com', 'serial_number': 'FOC1234ABCD', 'interface_list': ['Gi1/0/1', 'Gi1/0/2', ...]}

interfaces = device.get_interfaces()
# {'Gi1/0/1': {'is_up': True, 'is_enabled': True, 'description': 'uplink', 'last_flapped': 12.3, 'speed': 1000, 'mtu': 1500, 'mac_address': 'ab:cd:ef:01:02:03'}}

bgp = device.get_bgp_neighbors()
# {'global': {'router_id': '10.0.0.1', 'peers': {'10.0.0.2': {'is_up': True, 'is_enabled': True, 'description': '', 'uptime': 9999, ...}}}}

arp = device.get_arp_table()
# [{'interface': 'Vlan100', 'mac': 'ab:cd:ef:01:02:03', 'ip': '192.0.2.50', 'age': 12.0}, ...]

lldp = device.get_lldp_neighbors()
# {'Gi1/0/1': [{'hostname': 'sw2.example.com', 'port': 'Gi1/0/24'}]}

device.close()
```

Same code works on Junos, Arista, Nokia, Palo Alto. That's the point. NAPALM is the **lingua franca** of multi-vendor automation.

The configuration methods are where NAPALM really shines — they implement the **NETCONF candidate-datastore model in software**, even on devices that don't natively support it. A Cisco IOS box with no candidate datastore? NAPALM fakes it: it stages the config, lets you compare, lets you discard, then commits.

```python
device.open()

# Stage a config (merge into running, like NETCONF merge)
device.load_merge_candidate(filename="new-vlan.conf")

# OR replace the entire running config (NETCONF "replace" semantics)
device.load_replace_candidate(filename="full-config.conf")

# See the diff before committing
diff = device.compare_config()
print(diff)
# +vlan 100
# + name USERS

if diff:
    if input("Commit? [y/N] ").lower() == "y":
        device.commit_config()        # apply for real
    else:
        device.discard_config()       # roll back

device.close()
```

Other NAPALM goodies:

- `get_environment()` — fans, PSU, temps, CPU, memory.
- `get_users()` — local user accounts.
- `ping(destination=...)` — wrap the device's ping CLI in JSON.
- `traceroute(destination=...)` — same for traceroute.
- `cli(commands=[...])` — fall-through to raw CLI. Returns `{cmd: output}`.

NAPALM is *opinionated*. If a getter exists for the thing you want, use it. If you need raw CLI, you can `cli()`-fallback, but you've left the abstraction.

### Nornir — the parallel orchestrator

**Nornir** is a **task runner** for network automation. It takes inventory (hosts) + tasks (functions) + workers (threads) and runs them all in parallel. Think "Ansible, but Python all the way down". No YAML, no Jinja runtime — just Python.

When you'd reach for Nornir over Ansible:

- You want types, debuggers, IDE autocomplete.
- You want to run a custom task that doesn't fit an Ansible module.
- You want fine-grained parallelism control.
- You want to inherit from existing Python codebase.

Install:
```bash
pip install nornir nornir-napalm nornir-netmiko nornir-utils
```

`config.yaml`:
```yaml
inventory:
  plugin: SimpleInventory
  options:
    host_file: "hosts.yaml"
    group_file: "groups.yaml"
runner:
  plugin: threaded
  options:
    num_workers: 50
```

`hosts.yaml`:
```yaml
sw1:
  hostname: 192.0.2.10
  groups: [cisco]
sw2:
  hostname: 192.0.2.11
  groups: [cisco]
```

`groups.yaml`:
```yaml
cisco:
  platform: ios
  username: admin
  password: secret
```

A simple task — get version on every device:

```python
from nornir import InitNornir
from nornir_napalm.plugins.tasks import napalm_get
from nornir_utils.plugins.functions import print_result

nr = InitNornir(config_file="config.yaml")

result = nr.run(task=napalm_get, getters=["facts"])
print_result(result)
```

That just ran `get_facts` on every host in parallel (up to `num_workers` at a time). Output is a result-object you can iterate over, filter, or pretty-print.

Custom task:

```python
from nornir.core.task import Task, Result

def upgrade_check(task: Task) -> Result:
    facts = task.run(task=napalm_get, getters=["facts"]).result
    version = facts["facts"]["os_version"]
    needs_upgrade = "15.2" not in version
    return Result(host=task.host, result={"version": version, "upgrade": needs_upgrade})

nr.run(task=upgrade_check)
```

Filtering:

```python
cisco = nr.filter(platform="ios")            # by attribute
distrib = nr.filter(F(groups__contains="distribution"))  # filter helper
```

Nornir's tasks-plugin ecosystem: `nornir_napalm`, `nornir_netmiko`, `nornir_paramiko`, `nornir_jinja2`, `nornir_pyez` (Junos PyEZ), `nornir_pyntc`, `nornir_routeros`. Each plugin exports task functions you import and pass to `nr.run(task=...)`.

### When each fits — the cheat-sheet

| Need                                         | Pick     |
|----------------------------------------------|----------|
| One-off SSH script, single device            | Netmiko  |
| "I just need raw CLI output, no abstraction" | Netmiko  |
| Cross-vendor, get a fact in a known shape    | NAPALM   |
| Compare-config / commit / rollback flow      | NAPALM   |
| Run something on 200 devices in parallel     | Nornir   |
| Pipeline with filtering + custom Python      | Nornir   |
| YAML config language, declarative            | Ansible  |

You can compose them: Nornir-with-NAPALM is the standard "I want Ansible-style fanout but pure Python" stack. Nornir-with-Netmiko is "I want fanout but raw CLI". Plain Netmiko is "I just need to ssh and grep".

## Intent-Based Networking

**Intent-based networking** (IBN) is the marketing term for "describe what you want, not how to get it". It's the network-engineering version of "I want a Lyft to the airport" instead of "turn left here, then right at the light, then merge onto the freeway".

### What "intent" really means

The classic config sentence is *imperative*: `interface Gi1/0/1; switchport access vlan 100`. You told the device exactly what to do.

The intent sentence is *declarative*: "VLAN 100 must reach every site for application X". You described an outcome. Some piece of software has to figure out which interfaces, which VLAN trunks, which routes, which firewall rules satisfy that outcome — and apply them everywhere.

That software is the **intent engine**. It speaks YANG/OpenConfig out the back to push real config, but the front-door API takes high-level intents.

### The closed loop

The big-deal idea in IBN is the **closed loop**:

```
intent ── translate ──> policy ── render ──> config ──> push to devices
   ^                                                            │
   │                                                            v
drift detection <──── analyze ──── telemetry ──── stream from devices
```

Intent flows down, telemetry flows up, drift-detection compares "what we said we wanted" with "what's actually happening", and the engine reconciles automatically (or alerts a human).

Without the closed loop, it's just declarative config — write-once, hope-it-stays-right. With the closed loop, the system *re-converges* when reality drifts from intent. Someone unplugs a cable, telemetry sees a link-down, the engine recomputes paths, traffic re-routes, an alert fires saying "expected resilience reduced from N+2 to N+1".

### Real products

- **Cisco DNA Center / Catalyst Center** — Cisco's IBN platform for campus/branch. Programs Catalyst switches via NETCONF/RESTCONF/YANG. Has its own UI for "intents" like "users in Engineering get policy X". Drives SD-Access fabrics.
- **Apstra** (now part of Juniper) — DC-focused IBN. Pioneered the **blueprint** concept. You describe the fabric topology (e.g., "3-stage Clos, 4 spines, 16 leaves, EVPN-VXLAN") and Apstra renders configs for whichever vendor's gear is racked. Multi-vendor (Cumulus, Junos, EOS, NX-OS, SONiC).
- **Nokia NSP** — Service-provider IBN. Drives SR-OS gear (and others) for IP/MPLS, EVPN, segment-routing.
- **Cradlepoint NetCloud** — IBN for SD-WAN and 5G/LTE branches.
- **Arista CloudVision** — leans IBN-ish via "studios" — declarative fabric config templates on top of EOS.

### Apstra's "blueprint"

Apstra invented the term **blueprint** for "a complete, deployable description of a fabric". A blueprint includes:

- Logical topology (spine count, leaf count, link types).
- Resource pools (ASN ranges, IP-prefix pools, VNI pools).
- Connectivity templates (EVPN VLANs, VXLAN VNIs, virtual networks).
- Device profiles (which physical SKU plays each role).
- Policies (BGP timers, MTU, MLAG settings).

You "stage" the blueprint, "commit" it (Apstra renders configs and pushes), and the closed-loop "Intent-Based Analytics" probes run continuously to catch drift.

### The promise vs. the reality

The pitch: "describe intent, the network self-builds, self-heals, self-explains". The reality is messier:

- **Greenfield** works great. Build a brand-new fabric, blueprint it, lights-on. Rollouts in days, not months.
- **Brownfield** is hard. Existing config has decades of muscle-memory. Inserting an IBN layer over running gear means either reverse-engineering all that config back into intent (manual, slow) or accepting that some devices are "out of band" of the IBN.
- **Edge cases** still need the CLI. There's always one weird thing — a vendor bug, a one-off feature — that the IBN abstraction doesn't model.
- **Intent collisions** — two intents that both want to use VLAN 100 for different things. The engine has to detect and refuse, or it'll happily clobber.
- **Vendor coverage gaps** — IBN platforms support some platforms beautifully (the ones their dev team focuses on) and grudgingly support others. The "multi-vendor" claim is always a leaky one.

### Intent vs. config — concretely

Config-level: a thousand lines of `vlan`, `interface`, `route-map`, `ip access-list`.

Intent-level: half a dozen high-level statements:

```
intent: virtual-network APP-X
  vlan-id: 100
  scope: all-sites
  routing: anycast-gw

intent: security-policy FINANCE-ONLY
  source: group:finance
  destination: app:GL
  action: permit
```

The IBN engine renders that down into the thousand lines. If you change the intent ("scope: all-sites" → "scope: sites in EU only"), the engine re-renders and pushes the diff. The intent doc is now your source of truth; the device configs are byproducts.

### Why pure intent-based is hard

- **Legacy hooks** — gear from 2008 with no NETCONF; the engine has to fall back to scrape-and-template, defeating the abstraction.
- **Manual fixes** — humans `clear bgp neighbor` on a cranky peer; that's not in the intent doc; engine sees drift and may re-revert.
- **Intent collisions** — two intents disagree about the same resource; engine needs precedence rules.
- **Observability gaps** — telemetry isn't perfect; drift goes undetected.
- **Trust** — "click commit and the engine reconfigures 200 devices" requires battle-tested confidence, which takes years.

The compromise most shops land on: IBN for greenfield builds and well-defined slices (a campus, a fabric, a SD-WAN deployment), CLI/Ansible for the rest, and a long migration plan to swallow the rest into IBN over time.

## Source of Truth

Once your network is automated, the next question is: **where is the truth?** Not the running config — running configs drift, get hand-edited, fail to keep pace. The Source of Truth (**SoT**) is the canonical "what should be" — the model from which all configs derive.

### NetBox vs. Nautobot

**NetBox** is the open-source IPAM/DCIM/topology tool from DigitalOcean's Jeremy Stretch. **Nautobot** is a fork by Network To Code, with a more plugin-friendly architecture and richer extensibility. Both store the same kinds of data; pick one.

Both model:

- **Devices** — switches, routers, firewalls, servers, with vendor/model/serial.
- **Sites** — datacenters, campuses, branches, with addresses and time zones.
- **Racks** — physical racks, with rack-units, U-position of each device.
- **IPs and Prefixes** — IPAM. Every prefix tracked, every IP assigned to an interface.
- **VLANs** — VLAN-IDs, names, scopes (site/group/global).
- **Circuits** — WAN circuits, providers, A-side and Z-side terminations.
- **Cables** — physical cables between interfaces, with type and color.
- **Power** — PSUs, PDUs, power feeds.
- **VRFs, route-targets, BGP-AS** — the control-plane resources.

Web UI for humans, REST API for machines.

### The "single pane of source" doctrine

The doctrine: every fact lives in exactly one place. IP allocations live in NetBox; configs reference NetBox; spreadsheets and wikis are *forbidden* as truth (they're acceptable as views). When two systems disagree, NetBox wins.

If you let truth be in two places — say, NetBox *and* a spreadsheet — the two will drift, you'll fix one, forget the other, and create incidents. Pick one. Make it the law.

### API access

NetBox/Nautobot speak REST. You GET, POST, PUT, DELETE on `/api/dcim/devices/`, `/api/ipam/ip-addresses/`, etc. Auth is token-based.

Python clients:

- `pynetbox` for NetBox
- `pynautobot` for Nautobot (basically the same client with renamed entry-points)

```python
import pynetbox

nb = pynetbox.api(url="https://netbox.example.com", token="abc123def456")

# All devices in site SFO1
devices = nb.dcim.devices.filter(site="sfo1")
for d in devices:
    print(d.name, d.primary_ip4, d.device_type.model)

# Allocate a new prefix
new_prefix = nb.ipam.prefixes.create(prefix="10.50.0.0/24", site=12, role=3, status="active")

# Mark an IP used
ip = nb.ipam.ip_addresses.get(address="10.50.0.5/24")
ip.status = "active"
ip.dns_name = "sw-new.example.com"
ip.save()
```

### How SoT plugs into automation

Two patterns:

1. **Render configs from SoT data via Jinja2.** A pipeline reads NetBox, hydrates a template, produces config text, pushes to the device.

   ```jinja2
   hostname {{ device.name }}
   {% for vlan in device.site.vlans %}
   vlan {{ vlan.vid }}
    name {{ vlan.name }}
   {% endfor %}
   {% for iface in device.interfaces %}
   interface {{ iface.name }}
    description {{ iface.description }}
    switchport access vlan {{ iface.untagged_vlan.vid }}
   {% endfor %}
   ```

2. **Drive Ansible inventory dynamically from SoT.** Ansible has a `netbox.netbox.nb_inventory` plugin that pulls live device lists, IPs, and groups from NetBox. Your `hosts.yml` is now just `plugin: netbox.netbox.nb_inventory; api_endpoint: https://netbox.example.com`. Add a device to NetBox; Ansible sees it the next run.

### Custom fields, tags, journals

NetBox/Nautobot let you extend the schema:

- **Custom fields** — add `circuit_owner_email`, `change_window`, `compliance_required` to any object type.
- **Tags** — color-coded labels. `production`, `pci-zone`, `legacy`, `to-decommission`.
- **Journal entries** — per-object change log. "Replaced PSU on 2025-03-14" stays attached to the device forever.

These let you encode operational metadata in the SoT, not in tribal-knowledge spreadsheets.

### Webhooks for change events

NetBox can fire HTTP webhooks on object create/update/delete. Aim them at:

- **Slack** — "Bob just allocated 10.50.0.0/24"
- **Jenkins/GitHub Actions** — trigger config-render-and-push pipeline
- **Ticketing** — auto-create change record

This is how the SoT becomes *active* — changes in the model trigger downstream actions.

### The "one-way truth" rule

Configs reconcile *from* the SoT. The SoT does not learn from configs. If you allow reverse sync ("import from device" jobs that update NetBox from running configs), you've created a loop where bugs in templating leak into the SoT and become canonical. Resist.

The exception is **initial seed** — when you onboard a brownfield network, you bulk-import current state into NetBox to bootstrap. After that, NetBox leads, devices follow.

### Legacy alternatives

- **LibreNMS / Observium** — primarily monitoring, but they discover topology and host data, sometimes used as an SoT-of-last-resort.
- **Spreadsheets / Confluence pages / .txt files** — common, and a perpetual source of pain. Migrate.
- **HOMEGROWN** — every shop with 10+ engineers eventually has a NIH-built SQL+Flask "asset DB". Consider replacing it with NetBox unless you've earned the right to maintain it.

## CI/CD for Network

If your config is in Git, every change can go through the same gates as software: pull request, code review, automated tests, automated deploy, automated rollback. This is **CI/CD for network**, sometimes called **NetDevOps**.

### Git-flow for configs

The dance:

1. Network engineer wants to change a VLAN.
2. They branch off `main`, edit the YAML/Jinja/SoT entry that drives the VLAN config.
3. They open a Pull Request.
4. Peers review the diff.
5. CI runs validation: lint, dry-run, simulation.
6. If green, the PR merges to `main`.
7. CD pipeline picks up the merge and deploys.
8. Telemetry confirms the change took effect.
9. If broken, revert PR rolls back.

Same workflow as application code. Same review culture, same audit trail, same rollback velocity.

### Pre-merge validation

Before merging, run **automated checks** that catch errors humans miss:

- **Batfish** — symbolic analysis of network configs. Parses Cisco/Juniper/Arista/etc. config text, builds an in-memory model of the network, then lets you ask:
  - "Can host A reach service B?" → flow simulation
  - "Are there any BGP peers in active/idle?" → control-plane checks
  - "Does any ACL block what shouldn't be blocked?" → ACL holes
  - "Are any prefixes leaked between VRFs?" → routing-policy bugs

  Batfish runs in a Docker container; you talk to it with `pybatfish`:

  ```python
  from pybatfish.client.session import Session

  bf = Session()
  bf.set_network("prod")
  bf.init_snapshot("snapshots/2025-04-27", name="ss1", overwrite=True)

  # Reachability question
  result = bf.q.reachability(
      pathConstraints=PathConstraints(startLocation="@enter(sw1[Gi1/0/1])"),
      headers=HeaderConstraints(dstIps="10.0.0.5", applications="HTTP"),
      actions="SUCCESS",
  ).answer()
  print(result.frame())
  ```

- **Containerlab** (`clab`) — spin up a topology of containerized routers in seconds. Supports cEOS (Arista), vSRX (Juniper), Junos cRPD, Nokia SR-Linux, FRR, BGP-tools, SONiC. Define topology in YAML:

  ```yaml
  name: lab1
  topology:
    nodes:
      r1:
        kind: ceos
        image: ceos:4.31.0F
      r2:
        kind: ceos
        image: ceos:4.31.0F
    links:
      - endpoints: ["r1:eth1", "r2:eth1"]
  ```

  ```bash
  clab deploy -t lab.yml
  # spins up r1, r2 as containers
  # ssh admin@r1, real EOS CLI, real BGP, real ACLs
  clab destroy -t lab.yml
  ```

  CI can deploy the topology, push the candidate config, run synthetic-traffic tests, tear down. Real router behavior, no hardware.

- **Suzieq** — operational state collection. Polls devices via SSH/NETCONF, normalizes facts (interfaces, routes, BGP neighbors, MAC tables, LLDP) into a SQL-queryable store. CI uses Suzieq for **posture checks**: "after deploy, every BGP session is established" / "no interface flapped".

  ```bash
  suzieq-cli
  > bgp summary state="!Established"
  # if any rows, the deploy left a peer down — fail the pipeline
  ```

### Deploy strategies

You don't push to every device at once. Bad pushes happen. Limit the blast radius:

- **Canary** — deploy to one device first. Watch metrics for 5 min. If green, do the next 5%. Then 25%. Then the rest.
- **Blast-radius limits** — never touch >N devices in one window. Scope by site, by role, by fabric.
- **Commit-confirmed timer** — most NETCONF devices support `commit-confirmed 600`. The candidate becomes running, but if you don't issue a final `commit` within 10 minutes, the device auto-rolls back. Magic for fat-fingered config that locks you out of management.
- **Maintenance windows** — schedule risky changes for low-traffic windows.
- **Deploy gates** — pipeline pauses for human approval before push to production.

### Rollback strategies

When a deploy breaks something:

- **Revision history** — Junos `rollback 0` (last commit), `rollback 5` (5 commits ago), Cisco IOS-XE `archive config` + `rollback`. Easy because the device tracks revisions.
- **Atomic commits** — NETCONF/gNMI commits are all-or-nothing. Either every change applied, or none did.
- **Snapshot-and-restore** — pre-deploy, capture full running config to Git/S3; post-deploy, if rollback needed, push the snapshot back.
- **Forward-fix** — sometimes the right answer is a new commit that fixes the bug, not rolling back. Especially if the broken config has been on production for a while and rollback would re-break unrelated things.
- **Auto-rollback** — pipeline watches for symptom-set after deploy (e.g., BGP-down count > 0), auto-triggers rollback if seen.

### Example GitHub Actions workflow

```yaml
# .github/workflows/network-ci.yml
name: Network CI/CD

on:
  pull_request:
    paths: ["configs/**", "templates/**", "intents/**"]
  push:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install pyang
        run: pip install pyang
      - name: Validate YANG
        run: |
          for f in models/*.yang; do
            pyang --ietf "$f" || exit 1
          done
      - name: yamllint
        run: pip install yamllint && yamllint configs/

  batfish:
    runs-on: ubuntu-latest
    services:
      batfish:
        image: batfish/batfish:latest
        ports: [9996:9996, 9997:9997]
    steps:
      - uses: actions/checkout@v4
      - name: Install pybatfish
        run: pip install pybatfish
      - name: Reachability tests
        run: python tests/batfish_reachability.py

  containerlab:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install containerlab
        run: bash -c "$(curl -sL https://get.containerlab.dev)"
      - name: Deploy lab
        run: clab deploy -t lab.yml
      - name: Smoke test
        run: ansible-playbook -i lab-inventory smoke.yml
      - name: Tear down
        if: always()
        run: clab destroy -t lab.yml

  deploy:
    needs: [lint, batfish, containerlab]
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    environment: production       # forces approval gate
    steps:
      - uses: actions/checkout@v4
      - name: Ansible deploy
        env:
          ANSIBLE_VAULT_PASSWORD: ${{ secrets.VAULT }}
        run: |
          ansible-playbook -i inventory/prod deploy.yml \
            --vault-password-file <(echo "$ANSIBLE_VAULT_PASSWORD")
      - name: Post-deploy posture
        run: python tests/suzieq_posture.py
```

The pipeline lints, runs Batfish symbolic checks, spins up a containerlab dry-run, then on merge-to-main pushes via Ansible — gated by GitHub's `environment: production` approval rule. Every step has an exit-code; failure stops the pipeline.

## Paste-and-Runnable Shell

Real commands. Real expected output. Type these. Watch them work.

### Install the toolchain

```bash
$ pip install ansible ncclient pyang gnmic-py napalm netmiko nornir nornir-napalm pybatfish pynetbox suzieq
Collecting ansible
  Downloading ansible-9.4.0-py3-none-any.whl (50 MB)
Collecting ncclient
  Downloading ncclient-0.6.15-py2.py3-none-any.whl (181 kB)
Collecting pyang
  Downloading pyang-2.6.0-py2.py3-none-any.whl (596 kB)
...
Successfully installed ansible-9.4.0 ncclient-0.6.15 pyang-2.6.0 napalm-4.1.0 ...
```

### pyang — render a YANG module as a tree

```bash
$ pyang --tree /tmp/openconfig-interfaces.yang | head -10
module: openconfig-interfaces
  +--rw interfaces
     +--rw interface* [name]
        +--rw name      -> ../config/name
        +--rw config
        |  +--rw name?            string
        |  +--rw type             identityref
        |  +--rw mtu?             uint16
        |  +--rw loopback-mode?   boolean
        |  +--rw description?    string
```

`+--rw` = config (read-write). `+--ro` = state (read-only). `*` = list with key in brackets. Trailing `?` = optional leaf.

### Ansible — dry-run a VLAN deploy

```bash
$ ansible-playbook deploy-vlan.yml --check --diff
PLAY [distribution-switches] *********************************************

TASK [Gathering Facts] ***************************************************
ok: [dist01]
ok: [dist02]

TASK [config vlan 100] ***************************************************
--- before
+++ after
@@ -45,3 +45,5 @@
 vlan 50
  name MGMT
+vlan 100
+ name USERS
changed: [dist01]
changed: [dist02]

PLAY RECAP ***************************************************************
dist01: ok=2 changed=1 unreachable=0 failed=0 skipped=0 rescued=0 ignored=0
dist02: ok=2 changed=1 unreachable=0 failed=0 skipped=0 rescued=0 ignored=0
```

`--check` = dry run (no changes). `--diff` = show what would change. Real deploy = drop `--check`.

### gNMIc — get a single state value

```bash
$ gnmic -a router1:6030 --insecure --username admin --password admin \
       get --path /interfaces/interface[name=Ethernet1]/state
[
  {
    "source": "router1:6030",
    "timestamp": 1714220420000000000,
    "time": "2025-04-27T10:20:20.000000000-04:00",
    "updates": [
      {
        "Path": "interfaces/interface[name=Ethernet1]/state",
        "values": {
          "interfaces/interface/state": {
            "name": "Ethernet1",
            "admin-status": "UP",
            "oper-status": "UP",
            "mtu": 1500,
            "counters": {
              "in-octets": "98765432",
              "out-octets": "12345678",
              "in-pkts": "98765",
              "out-pkts": "12345"
            }
          }
        }
      }
    ]
  }
]
```

### gNMIc — subscribe to streaming counters

```bash
$ gnmic -a router1:6030 --insecure --username admin --password admin \
       subscribe \
       --path /interfaces/interface[name=Ethernet1]/state/counters \
       --stream-mode sample --sample-interval 5s
{"timestamp":1714220425000000000,"time":"2025-04-27T10:20:25-04:00","updates":[{"Path":"interfaces/interface[name=Ethernet1]/state/counters/in-octets","values":{"in-octets":"98765432"}}]}
{"timestamp":1714220430000000000,"time":"2025-04-27T10:20:30-04:00","updates":[{"Path":"interfaces/interface[name=Ethernet1]/state/counters/in-octets","values":{"in-octets":"98770432"}}]}
{"timestamp":1714220435000000000,"time":"2025-04-27T10:20:35-04:00","updates":[{"Path":"interfaces/interface[name=Ethernet1]/state/counters/in-octets","values":{"in-octets":"98775000"}}]}
^C
```

Stream stays open until you Ctrl-C. Each line is a 5-second sample.

### ncclient — interactive NETCONF session

```python
$ python
>>> from ncclient import manager
>>> m = manager.connect(
...     host="router1.example.com",
...     port=830,
...     username="admin",
...     password="secret",
...     hostkey_verify=False,
...     device_params={"name": "iosxr"},
... )
>>> m.connected
True
>>> caps = list(m.server_capabilities)
>>> len(caps)
67
>>> caps[0]
'urn:ietf:params:netconf:base:1.0'
>>> result = m.get_config(source="running")
>>> print(result.data_xml[:500])
<?xml version="1.0"?>
<data xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <interface-configurations xmlns="http://cisco.com/ns/yang/Cisco-IOS-XR-ifmgr-cfg">
    <interface-configuration>
      <active>act</active>
      <interface-name>GigabitEthernet0/0/0/0</interface-name>
      <description>uplink-to-core</description>
...
>>> m.close_session()
```

### pynetbox — quick GET

```python
$ python
>>> import pynetbox
>>> nb = pynetbox.api(url="https://netbox.example.com", token="abc123def456")
>>> sw = nb.dcim.devices.get(name="sw-core-01")
>>> sw.name
'sw-core-01'
>>> sw.device_type.model
'C9500-48Y4C'
>>> sw.primary_ip4.address
'10.255.1.10/24'
>>> for iface in sw.interfaces.all()[:3]:
...     print(iface.name, iface.untagged_vlan)
TwentyFiveGigE1/0/1 VLAN10 (10)
TwentyFiveGigE1/0/2 VLAN20 (20)
TwentyFiveGigE1/0/3 VLAN30 (30)
```

### Batfish — symbolic reachability test

```python
$ python
>>> from pybatfish.client.session import Session
>>> from pybatfish.datamodel.flow import HeaderConstraints, PathConstraints
>>> bf = Session(host="localhost")
>>> bf.set_network("prod")
'prod'
>>> bf.init_snapshot("snapshots/baseline", name="baseline", overwrite=True)
'baseline'
>>> ans = bf.q.reachability(
...     pathConstraints=PathConstraints(startLocation="sw1"),
...     headers=HeaderConstraints(dstIps="10.0.0.5", applications="HTTP"),
...     actions="SUCCESS",
... ).answer().frame()
>>> print(ans)
   Flow  Traces
0  start=sw1 [10.255.1.10:49152->10.0.0.5:80 TCP]   1
>>> # one trace, success — reachable
```

### Containerlab — deploy a 2-node lab

```bash
$ cat lab.yml
name: ramp
topology:
  nodes:
    r1: {kind: ceos, image: ceos:4.31.0F}
    r2: {kind: ceos, image: ceos:4.31.0F}
  links:
    - endpoints: ["r1:eth1", "r2:eth1"]

$ sudo clab deploy -t lab.yml
INFO[0000] Containerlab v0.55.0
INFO[0000] Parsing & checking topology file: lab.yml
INFO[0000] Creating lab directory: /home/op/clab-ramp
INFO[0000] Creating container: "r1"
INFO[0001] Creating container: "r2"
INFO[0005] Creating virtual wire: r1:eth1 <--> r2:eth1
INFO[0006] Adding host entries
+---+-------------+--------------+----------+------+---------+
| # |    Name     |  Kind/Image  |  State   |  IPv4 Address  |
+---+-------------+--------------+----------+------+---------+
| 1 | clab-ramp-r1| ceos:4.31.0F | running  | 172.20.20.2/24 |
| 2 | clab-ramp-r2| ceos:4.31.0F | running  | 172.20.20.3/24 |
+---+-------------+--------------+----------+------+---------+

$ ssh admin@172.20.20.2
admin@172.20.20.2's password:
r1#show version
Arista cEOS-Lab
Software image version: 4.31.0F
...
```

### Suzieq-cli — posture check after deploy

```bash
$ suzieq-cli
suzieq-cli> namespace prod

suzieq-cli> bgp summary state="!Established"
                  hostname  vrf      peer       state    afi
0                      sw1   default  10.0.0.2   Idle     ipv4

# one peer Idle — investigate before declaring deploy successful
```

## Common Errors and Fixes

The exact text you'll see, and the canonical fix.

```
candidate datastore not supported
```
Vendor lacks NETCONF candidate. Either fall back to running with `target=running` (and accept partial-apply risk), upgrade to a release with candidate, or use NAPALM's emulated candidate (`load_merge_candidate` works even where the device doesn't natively).

```
gNMI Subscribe stream closed: rpc error: code = Unavailable desc = transport: Error while dialing
```
TLS or auth misconfig. Check (1) TLS cert paths and CN match, (2) gRPC port (default 6030 for cEOS, 50051 for many others), (3) `--insecure` flag for testing, (4) firewall rule for gRPC port.

```
YANG module load failed: undefined identityref base
```
Missing `import` of the YANG module that defines the identity. Check the `import` statements at the top of your module; pyang needs the imported file findable via `--path`.

```
ansible.errors.AnsibleConnectionFailure: ssh: connection refused
```
Two common roots: (1) `host_key_checking = True` and the device's host key isn't in `known_hosts`; set `host_key_checking = False` in `ansible.cfg` or pre-populate. (2) The device's SSH service isn't listening — check `show ip ssh` and `crypto key generate rsa modulus 2048`.

```
RESTCONF 401 Unauthorized
```
HTTP basic auth not enabled, wrong user/pass, or `aaa authentication login default local` misconfigured. Confirm `restconf` is enabled and AAA chain accepts the user. Curl with `-v` to see the auth headers.

```
HTTP/1.1 404 Not Found on /restconf/data/ietf-interfaces:interfaces
```
The YANG module isn't loaded or you typo'd the path. List loaded modules: `GET /restconf/data/ietf-yang-library:modules-state/module`. Check spelling; identityrefs are case-sensitive.

```
pyang: error: unsupported substatement 'action' for 'rpc'
```
YANG 1.0 vs 1.1 mismatch. `action` is YANG 1.1 only. Either upgrade your module to `yang-version 1.1;` or avoid 1.1-only constructs. Validate with `pyang -V` or `yanglint`.

```
napalm.base.exceptions.ConnectionException: cannot connect to device
```
Driver mismatch (e.g., `nxos_ssh` driver against an IOS box), bad credentials, SSH port wrong, or `optional_args={'secret': ...}` missing for IOS enable. Run Netmiko directly first — if Netmiko works and NAPALM doesn't, it's a driver issue.

```
ansible-playbook: persistent connection idle timeout
```
The persistent SSH connection sat idle too long. Bump in `ansible.cfg`:
```ini
[persistent_connection]
connect_timeout = 30
command_timeout = 30
```

```
Batfish: invalid file format
```
Batfish needs configs prefixed with vendor headers like `! Cisco IOS` or in subfolder `configs/` with hostname matching filename. Re-organize the snapshot per Batfish docs, ensure `hosts/` and `configs/` subdirs exist.

```
containerlab: failed to pull image registry.com/ceos:4.31.0F: 401 Unauthorized
```
Login first: `docker login registry.com`. For Arista: download cEOS tar from arista.com, then `docker import cEOS-lab.tar ceos:4.31.0F`.

```
yanglint: invalid leafref target "/if:interfaces/if:interface/if:name"
```
The referenced leaf doesn't exist (typo, missing prefix, wrong namespace). Check the leafref `path` against the actual data tree.

```
ncclient.transport.errors.SSHError: Negotiation failed.
```
Old IOS uses `diffie-hellman-group1-sha1` and weak ciphers, modern Paramiko refuses. Force kex/cipher in ncclient connect: `device_params={'name': 'default'}, hostkey_verify=False, allow_agent=False, look_for_keys=False, ssh_config=None, key_filename=None`. If that fails, try ncclient with `~/.ssh/config` setting `KexAlgorithms +diffie-hellman-group1-sha1` and `Ciphers +aes256-cbc`.

```
ansible-galaxy collection install cisco.ios — ERROR! - the collection cannot be installed
```
Often a corporate proxy or PyPI mirror; set `https_proxy` or use `--server galaxy.ansible.com`.

## Migration Stories

### Brownfield → NETCONF: shadow-mode parallel run

You have 200 routers, hand-configured for 15 years. You want to flip them to NETCONF-driven. You don't flip overnight.

The pattern: **shadow mode**.

1. Stand up the SoT (NetBox) and pipeline (Ansible/Nornir + NETCONF templates).
2. For each device, render the candidate config from SoT.
3. Compare-but-don't-deploy: diff candidate vs running. Hand the diff to a human.
4. Human investigates discrepancies: are they bugs in the template, gaps in the SoT, or legitimate-but-undocumented config?
5. Patch the SoT or template until candidate == running for that device.
6. Mark the device "managed". From now on, all changes go via the pipeline.
7. Repeat for the next device.

This is **trust-then-verify**: build trust in the pipeline by making it produce the existing config exactly, *then* let it deploy changes.

### "Why we couldn't fully automate"

Real reasons real teams stay partially manual:

- **Vendor bugs**: NETCONF on this NX-OS release leaks file descriptors and crashes after 200 commits. Until the bug is fixed, this fabric is CLI-managed.
- **Legacy gear**: the 6500 in DC-3 doesn't speak NETCONF. We're decommissioning it next quarter; until then, hand-configure.
- **Political**: Operations team owns datacenter A, refuses to give up CLI. Engineering owns datacenter B, fully automated. Two parallel worlds until org chart resolves.
- **Skill gap**: the team has two engineers comfortable with NETCONF, eighteen comfortable with CLI. Until training catches up, can't risk full-automation rollouts to non-experts.
- **Trust deficit**: last year's automation pushed a typo'd ACL to 200 devices in 90 seconds. Every new automated change now needs human approval gate, defeating much of the speed benefit.
- **The human-in-loop checkpoint pattern**: pipeline does 95% — render, diff, validate. Final `git push` to production-branch is a human action, not auto. Slows you down, but a wrong human commit is cheaper than a wrong robot deploy.

The honest truth: full automation is a journey, not a destination. Most large networks are forever 70-90% automated, with the long tail being one-offs, brownfield, exceptions.

## Russ White's Take (Ch 26, "Computer Networking Problems and Solutions")

Russ White (BGP-veteran, RFC author, co-author of *Computer Networking Problems and Solutions* with Ethan Banks) devotes Chapter 26 to network automation. The big themes:

### Imperative vs. declarative

Russ frames automation paradigms cleanly. **Imperative** = "do X then Y then Z" (a script, an Ansible playbook with raw `cli_command`). **Declarative** = "the end-state should be S" (Terraform, NETCONF replace, intent-based). Imperative is procedural; declarative is descriptive. Declarative scales because you don't have to spec every step — the system figures out the diff.

His warning: "declarative" is a spectrum. Ansible playbooks claim declarative but use imperative modules; NETCONF replace is more declarative than NETCONF merge; pure intent-based engines push furthest into declarative-land. Know where on the spectrum you are.

### Control plane vs. data plane

Russ is famous for hammering the **control plane / data plane / management plane** separation. Automation lives mostly in the **management plane** — the orchestration layer that programs the control plane. Good automation respects the boundary: it configures the routing protocols (BGP, OSPF, IS-IS) but doesn't try to *replace* them. The protocols still own convergence, failure detection, repath. Automation just sets the stage.

When automation tries to do control-plane work (e.g., a centralized SDN controller computing every path), you get the SDN promises-and-pitfalls of the 2010s. Russ's view: protocols evolved over decades to handle distributed convergence; don't reinvent that wheel from a centralized controller unless your scale and topology really demand it.

### What's coming

Russ sees a few directions:

- **ML-driven anomaly detection**: tons of telemetry, ML over time-series catches "this BGP session is about to flap" 30 seconds early.
- **Intent purer**: blueprint engines mature to where greenfield IBN is reliable; brownfield IBN remains hard.
- **Fewer humans-in-loop, but never zero**: automation handles 99% of changes; humans handle the 1% that matter (architecture, exceptions, post-incident review).
- **Disaggregation**: open NOSes (SONiC, FRR, DENT) decouple hardware from software. Automation looks the same regardless of vendor.
- **Protocol simplification**: tomorrow's protocols are designed with automation as a first-class consumer (gNMI/gRPC native, OpenConfig models built-in), not bolted on.

His skepticism: "automate every change" is a marketing slogan, not a target. Automate what's repeatable and high-volume; leave the architectural and post-incident work to humans.

## When NOT to Automate

Counterintuitively, the mature automation-shop knows when *not* to automate:

- **One-off changes**. If you're going to do this exactly once, don't write 200 lines of Ansible. PR the manual change to your config repo, get review, apply by hand, document in the journal. Automation is amortization; one-off doesn't amortize.
- **Emergency hotfix**. Production is on fire, BGP is down, customers are paging. SSH in and fix it. *Then* go back, write the automation that would have prevented it, add the test that catches it. Don't try to author robust idempotent code at 3am while the SLO is bleeding.
- **Trust-not-yet-built**. Brand new automation has bugs. Run it in shadow mode (compare-but-don't-deploy) for weeks before letting it touch real config. Pipeline that auto-deploys on day one will lose its trust budget on day two.
- **Complex troubleshooting**. Forensic debugging is a creative human activity. You can automate data-collection (Suzieq, gNMI dumps), but the "what's actually wrong" reasoning needs a human.
- **Regulatory compliance gates**. SOX, HIPAA, PCI-DSS often require explicit human approval for production changes. Automate up to the gate; let the human pull the trigger.
- **High-blast-radius**. Rare changes that touch every device — re-keying a domain TLS cert, rolling all PSK creds — sometimes safer in a structured manual playbook with checkpoints than a fully-automated push.
- **When the abstraction lies**. If the automation tool's abstraction is fundamentally wrong for the change you're making (e.g., Ansible can't express the dependency graph), don't fight it. Drop to lower level (NETCONF, raw CLI) for that one change.

The mature stance: **automation is a lever**, not a religion. Use it where it pays off. Skip it where it doesn't.

## See Also

- [networking/restconf](../../sheets/networking/restconf.md) — the dense RESTCONF reference once this feels easy
- [networking/yang-models](../../sheets/networking/yang-models.md) — YANG dense reference
- [networking/network-programmability](../../sheets/networking/network-programmability.md) — programmability dense reference
- [config-mgmt/ansible](../../sheets/config-mgmt/ansible.md) — Ansible dense reference
- [config-mgmt/dc-automation](../../sheets/config-mgmt/dc-automation.md) — DC-automation dense reference
- [config-mgmt/napalm](../../sheets/config-mgmt/napalm.md) — NAPALM dense reference
- [monitoring/gnmi-gnoi](../../sheets/monitoring/gnmi-gnoi.md) — gNMI/gNOI deep dive
- [monitoring/model-driven-telemetry](../../sheets/monitoring/model-driven-telemetry.md) — MDT deep dive
- [ramp-up/ansible-eli5](../../sheets/ramp-up/ansible-eli5.md) — Ansible ELI5
- [ramp-up/terraform-eli5](../../sheets/ramp-up/terraform-eli5.md) — IaC sibling
- [ramp-up/github-actions-eli5](../../sheets/ramp-up/github-actions-eli5.md) — CI sibling

## References

- RFC 6020 — YANG 1.0 (Data Modeling Language for NETCONF)
- RFC 7950 — YANG 1.1 (revised data modeling language)
- RFC 6241 — NETCONF Protocol
- RFC 6242 — Using the NETCONF Protocol over Secure Shell (SSH)
- RFC 6243 — With-defaults Capability for NETCONF
- RFC 8040 — RESTCONF Protocol
- RFC 8071 — NETCONF Call Home and RESTCONF Call Home
- RFC 8072 — YANG Patch Media Type
- RFC 8345 — Network Topology and Topology State YANG Data Models
- RFC 8525 — YANG Library
- RFC 8526 — NETCONF Extensions to Support the Network Management Datastore Architecture
- RFC 8639 — Subscription to YANG Notifications
- RFC 8641 — Subscription to YANG Notifications for Datastore Updates
- gNMI specification — github.com/openconfig/gnmi
- gNOI specification — github.com/openconfig/gnoi
- OpenConfig models — github.com/openconfig/public
- IETF YANG Catalog — yangcatalog.org
- Ansible network documentation — docs.ansible.com/ansible/latest/network/
- NAPALM documentation — napalm.readthedocs.io
- Nornir documentation — nornir.readthedocs.io
- Netmiko documentation — github.com/ktbyers/netmiko
- NetBox — github.com/netbox-community/netbox
- Nautobot — github.com/nautobot/nautobot
- pynetbox — github.com/netbox-community/pynetbox
- Batfish — batfish.org and github.com/batfish/batfish
- Containerlab — containerlab.dev
- Suzieq — suzieq.io and github.com/netenglabs/suzieq
- gNMIc CLI — gnmic.openconfig.net
- "Network Programmability and Automation: Skills for the Next-Generation Network Engineer" — Edelman, Lowe, Oswalt (O'Reilly, 2018, 2nd ed. 2022)
- "Computer Networking Problems and Solutions" — Russ White & Ethan Banks (Pearson, 2018), Chapter 26 ("Network Automation")
- "Automate Your Network" — John W. Capobianco (self-published, 2019)
- "Network Automation Cookbook" — Karim Okasha (Packt, 2020)
- Cisco DevNet learning labs — developer.cisco.com/learning
- Juniper Day One Books on automation — juniper.net/dayone
- Apstra technical documentation — juniper.net/documentation/product/us/en/apstra
