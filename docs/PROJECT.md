# cs — The Complete Cheatsheet CLI

> A single-binary Go CLI embedding 549 markdown cheatsheets and 549 deep-dive theory pages across 59 categories. Built-in calculator, subnet calculator, fuzzy search, interactive TUI, REST API, shell completions, bookmarks, cross-references, export, learning paths, and math verification — all in a 25 MB binary with zero runtime dependencies.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Content Inventory](#content-inventory)
4. [Features](#features)
5. [Build & Install](#build--install)
6. [Content Format](#content-format)
7. [Certification Coverage](#certification-coverage)
8. [Coding Problems](#coding-problems)
9. [Category Reference](#category-reference)
10. [Design Decisions](#design-decisions)
11. [Development History](#development-history)

---

## Overview

`cs` is a terminal-native knowledge base for systems engineers, network architects, security professionals, and software developers. Every cheatsheet is a markdown file embedded at compile time via Go's `embed.FS` — no internet connection, no package manager, no config files needed. Just the binary.

**Key numbers:**
- **549 cheatsheets** — practical quick-reference with code blocks, tables, tips
- **549 detail pages** — deep-dive theory, math, proofs, architectural analysis
- **59 categories** — networking, security, offensive, system, cloud, databases, CS theory, fundamentals, kernel tuning, Juniper, and more
- **20 coding problems** — multi-language solutions (Go, Rust, Python, TypeScript)
- **25 MB binary** — single static binary, zero runtime deps
- **4,563 lines of Go** — across 14 source files and 8 internal packages

**What it replaces:** Scattered bookmarks, `tldr` pages, man pages, personal wikis, multiple browser tabs of documentation. Everything in one `cs <topic>` command.

---

## Architecture

```
cs (root module)
├── sheets.go                    # go:embed sheets/*/*.md + detail/*/*.md
├── cmd/cs/main.go               # CLI entry point, flag parsing, REST API
├── internal/
│   ├── registry/                # Sheet struct, parsing, search, fuzzy match
│   │   └── Sheet{Name, Category, Content, SeeAlso, Prerequisites, Complexity}
│   ├── render/                  # glamour terminal rendering, TTY detection, pager
│   ├── tui/                     # Interactive TUI (bubbletea + bubbles + lipgloss)
│   ├── calc/                    # Expression calculator (arithmetic, hex/oct/bin, units)
│   ├── subnet/                  # CIDR subnet calculator (IPv4 + IPv6)
│   ├── custom/                  # User overlay sheets from ~/.config/cs/sheets/
│   ├── bookmarks/               # Bookmark management (~/.config/cs/bookmarks.json)
│   └── verify/                  # Math verification for detail pages
├── sheets/                      # 549 embedded cheatsheets
│   ├── networking/ (85)
│   ├── security/ (48)
│   ├── offensive/ (37)
│   ├── system/ (32)
│   ├── cs-theory/ (25)
│   ├── coding-problems/ (20)
│   ├── ... (53 more categories)
│   └── build-systems/ (2)
└── detail/                      # 549 embedded deep-dive pages
    └── (mirrors sheets/ structure)
```

### How Content Discovery Works

The entire content system is zero-config, driven by Go's embed directive:

```go
//go:embed sheets/*/*.md
var EmbeddedSheets embed.FS

//go:embed detail/*/*.md
var EmbeddedDetails embed.FS
```

**Adding a new category or topic requires zero code changes.** Drop a `.md` file into `sheets/<category>/` and rebuild. The glob auto-discovers it. The registry parses the H1 title, one-liner description, `## See Also` cross-references, `## References`, and `## Tips` sections from each file at init time.

### Dependencies

| Dependency | Purpose |
|-----------|---------|
| `glamour` | Terminal markdown rendering (syntax highlighting, tables, links) |
| `x/term` | Terminal width detection, raw mode |
| `bubbletea` | TUI framework (Elm architecture for terminals) |
| `bubbles` | TUI components (text input, viewport, list) |
| `lipgloss` | TUI styling (colors, borders, padding) |

**No external routers** (stdlib `net/http` for REST API), **no logging frameworks** (simple `stderr`), **no CLI frameworks** (stdlib `flag`).

---

## Content Inventory

### By Category (59 categories, sorted by sheet count)

| Category | Sheets | Focus |
|----------|--------|-------|
| `networking` | 85 | Protocols (TCP/IP/BGP/OSPF), tools, wireless, tunneling |
| `security` | 48 | Hardening, forensics, compliance, IDS/IPS, cryptography |
| `offensive` | 37 | Ethical hacking, pentesting, exploit tools, CTF methodology |
| `system` | 32 | Linux internals, process management, kernel, debugging |
| `cs-theory` | 25 | Turing machines, complexity, category theory, crypto theory |
| `coding-problems` | 20 | LeetCode-style problems with Go/Rust/Python/TypeScript solutions |
| `databases` | 16 | PostgreSQL, MySQL, Redis, MongoDB, SQLite, Elasticsearch |
| `data-formats` | 13 | JSON, YAML, TOML, Protocol Buffers, Avro, Parquet |
| `orchestration` | 11 | Kubernetes, Helm, operators, CRDs, scheduling |
| `monitoring` | 11 | Prometheus, Grafana, alerting, distributed tracing |
| `containers` | 11 | Docker, OCI, CRI, Podman, buildah, container security |
| `testing` | 10 | Unit/integration/E2E, TDD, property-based, mutation testing |
| `languages` | 10 | Go, Rust, Python, TypeScript, Bash, Lua, C, Java, Zig, Ruby |
| `fundamentals` | 9 | Tiered ELI5-to-college: computers, networking, binary, ISAs, kernel |
| `terminal` | 8 | tmux, screen, readline, terminal emulators |
| `juniper` | 8 | JNCIA-Junos certification: architecture, routing, filters, interfaces |
| `compliance` | 8 | SOC2, PCI-DSS, HIPAA, GDPR, NIST, CIS benchmarks |
| `ai-ml` | 8 | Transformers, LoRA, vector databases, prompt engineering |
| `storage` | 7 | LVM, Ceph, btrfs, ZFS, Rook, Longhorn |
| `shell` | 7 | Bash, Zsh, fish, POSIX, readline, dotfiles |
| `network-tools` | 7 | Wireshark, iperf, mtr, dig, netcat, sftp |
| `disk` | 7 | fdisk, parted, mount, fstab, SMART, mdadm |
| `cloud` | 7 | AWS, GCP, Azure CLIs, IAM, VPC, S3 |
| `ci-cd` | 7 | GitHub Actions, GitLab CI, Jenkins, ArgoCD |
| `big-data` | 7 | Spark, Kafka, Flink, Airflow, Hadoop |
| `kernel-tuning` | 6 | CPU scheduler, memory, network stack, I/O, IRQ, hardening |
| `package-managers` | 6 | apt, dnf, brew, snap, nix, pip |
| `filesystems` | 6 | ext4, XFS, NFS, FUSE, OverlayFS, tmpfs |
| `api` | 6 | REST, GraphQL, gRPC, WebSocket, OpenAPI |
| `patterns` | 5 | Distributed systems, microservices, event-driven, design patterns |
| `performance` | 5 | eBPF, bpftrace, perf, flamegraphs, caching |
| `config-mgmt` | 5 | Ansible, Chef, Puppet, Salt, Terraform |
| `auth` | 4 | OAuth2, OIDC, LDAP, Kerberos |
| `vcs` | 4 | Git, GitHub, GitLab, Mercurial |
| `virtualization` | 4 | KVM, QEMU, libvirt, Vagrant |
| `service-mesh` | 4 | Istio, Envoy, Linkerd, Consul |
| `web-servers` | 3 | Nginx, HAProxy, Caddy |
| `messaging` | 3 | RabbitMQ, NATS, ZeroMQ |
| `dns` | 2 | BIND, CoreDNS |
| `network-os` | 2 | JunOS, Cisco IOS |
| *...and 19 more* | 2-3 each | logs, backup, email, serverless, IaC, etc. |

### Content Tiers

Every topic has **two tiers**:

1. **Sheet** (`sheets/<category>/<topic>.md`) — Practical quick-reference. Code blocks you can copy-paste. Tables of options. Tips. Cross-references. Designed to answer "how do I do X?" in 30 seconds.

2. **Detail** (`detail/<category>/<topic>.md`) — Deep theory. Mathematical analysis. Algorithm internals. Proofs. Architecture diagrams. Designed to answer "why does X work that way?" for study or interview prep.

### Tiered Educational Content (Fundamentals)

The `fundamentals` category uses a special format with progressive depth levels:

```
## ELI5 (Explain Like I'm 5)
## Middle School
## High School
## College
```

Topics: How Computers Work, Binary & Number Systems, How Networking Works, How the Internet Works, Linux Kernel Internals, x86-64 Assembly, ARM64 Architecture, RISC-V, eBPF Bytecode.

---

## Features

### Core Usage

```bash
cs tcp                        # show TCP cheatsheet
cs -d tcp                     # show TCP deep-dive theory
cs tcp "congestion"           # show only the congestion section
cs networking                 # list all networking topics
cs -s "raft consensus"        # full-text search across all sheets
cs -l                         # list all 549 topics with descriptions
cs -i                         # launch interactive TUI
cs --random                   # random cheatsheet (learn something new)
cs --count                    # show statistics
```

### Built-in Tools

```bash
cs calc "2**32 - 1"           # = 4294967295
cs calc "100 GB / 1 Gbps"     # = 800 seconds (unit-aware)
cs calc "0xFF & 0x0F"         # = 15 (bitwise operations)
cs calc "0b11001010"          # = 202 (binary conversion)
cs subnet 10.0.0.0/24         # full subnet breakdown (IPv4)
cs subnet 2001:db8::/48       # IPv6 subnet info
cs compare tcp udp            # side-by-side comparison
cs verify                     # verify math expressions in detail pages
cs learn networking           # ordered learning path by prerequisites
```

### Knowledge Graph

```bash
cs --related bgp              # show related topics from See Also links
cs -d bgp --prereqs           # show prerequisites for the BGP detail page
```

### Bookmarks & Export

```bash
cs --star tcp                 # bookmark a topic
cs --starred                  # list bookmarked topics
cs --format json tcp          # export as JSON
cs --format markdown tcp      # export raw markdown
```

### REST API

```bash
cs serve                      # start API on :9876
curl localhost:9876/api/sheets              # list all sheets
curl localhost:9876/api/sheets/tcp          # get TCP sheet
curl localhost:9876/api/search?q=consensus  # search
```

### Interactive TUI

The TUI features an "Amber Throne" color palette with category browsing, fuzzy filtering, content viewport with vim-like keybindings, and bar charts showing topic distribution per category.

```bash
cs -i                         # launch TUI
# j/k or arrows to navigate, / to filter, Enter to view, q to quit
```

### Shell Completions

```bash
cs --completions bash >> ~/.bashrc
cs --completions zsh >> ~/.zshrc
cs --completions fish > ~/.config/fish/completions/cs.fish
```

### Custom Sheets

```bash
cs --add my-notes.md          # add custom sheet to ~/.config/cs/sheets/
cs --edit my-topic            # create/edit in $EDITOR
# Custom sheets overlay embedded ones (user overrides take priority)
```

---

## Build & Install

```bash
make build          # build ./cs binary (25 MB)
make install        # install to /usr/local/bin
make test           # go test ./... -count=1 -race
make lint           # go vet + staticcheck
make fmt            # gofmt -s -w .
```

### Build Flags

```bash
go build -trimpath -ldflags="-s -w -X main.version=$(git describe --tags --always)" -o cs ./cmd/cs/
```

- `-trimpath` — reproducible builds (strip local paths)
- `-s -w` — strip debug info and DWARF (reduces binary ~30%)
- `-X main.version=...` — inject version at build time

### Requirements

- Go 1.24+
- No CGO dependencies
- Cross-compiles to any Go-supported platform

---

## Content Format

### Sheet Format (`sheets/<category>/<topic>.md`)

```markdown
# Topic Name (Subtitle / Context)

One-liner description of what this topic covers.

## Section Name

### Subsection

\`\`\`bash
# practical command examples
command --flag value
\`\`\`

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| data     | data     | data     |

## Tips

- Practical tip #1
- Practical tip #2

## See Also

- `related-topic-1` — brief description
- `related-topic-2` — brief description

## References

- Official documentation link
- RFC or standard
- Book or paper reference
```

### Detail Format (`detail/<category>/<topic>.md`)

```markdown
# Topic Name — Deep Dive Subtitle

> *Blockquote summarizing what this deep-dive covers and why it matters.*

## Prerequisites

| Concept | Sheet |
|---------|-------|
| Required concept | `prerequisite-sheet` |

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| operation | O(n) | O(1) |

## 1. First Major Section

Theory, math, proofs. LaTeX notation: $O(n \log n)$

## 2. Second Major Section

More theory with worked examples.

## References

- Academic papers, RFCs, textbooks
```

### Adding New Content

1. Create `sheets/<category>/<topic>.md` following the sheet format
2. Optionally create `detail/<category>/<topic>.md` for the deep dive
3. Run `make build` — the embed glob auto-discovers new files
4. No code changes required for new categories or topics

---

## Certification Coverage

### JNCIA-Junos (Juniper Networks Certified Associate)

Full coverage of all 7 exam objective areas:

| Exam Area | Sheet(s) |
|-----------|----------|
| Networking Fundamentals | `collision-broadcast-domains`, `cos-qos`, `ethernet`, `arp`, `ipv4`, `ipv6`, `subnetting`, `tcp`, `udp` |
| Junos OS Fundamentals | `junos-architecture` — RE/PFE, control/forwarding planes, transit/exception traffic |
| User Interfaces | `junos` — CLI modes, navigation, help, filtering |
| Configuration Basics | `junos-user-management` — factory default, users, login classes, NTP/SNMP/syslog |
| Operational Monitoring | `junos-monitoring` — show/monitor commands, interface errors, logging/tracing |
| Routing Fundamentals | `junos-routing-fundamentals` — tables, route preference, instances, static routing |
| Routing Policy & Filters | `junos-routing-policy` + `junos-firewall-filters` — policy flow, match criteria, uRPF |

Additional supporting sheets: `junos-interfaces`, `junos-software`, `ospf`, `bgp`, `vlan`, `ntp`, `snmp`

### CEH v13 (Certified Ethical Hacker)

Full coverage of all 20 CEH modules:

| Module | Sheet(s) |
|--------|----------|
| M01 — Ethical Hacking Intro | `pentest-methodology`, `mitre-attack`, `zero-trust` |
| M02 — Footprinting & Recon | `recon`, `osint` |
| M03 — Scanning Networks | `network-scanning`, `nmap` |
| M04 — Enumeration | `enumeration-techniques`, `ldap`, `snmp` |
| M05 — Vulnerability Analysis | `vulnerability-scanning`, `grype` |
| M06 — System Hacking | `system-hacking`, `privilege-escalation` |
| M07 — Malware Threats | `malware-analysis` |
| M08 — Sniffing | `sniffing-attacks`, `wireshark`, `tshark`, `tcpdump` |
| M09 — Social Engineering | `social-engineering` |
| M10 — DoS/DDoS | `dos-ddos-attacks` |
| M11 — Session Hijacking | `session-hijacking` |
| M12 — Evading IDS/FW | `evasion-techniques`, `ids-ips` |
| M13 — Web Servers | `nginx`, `haproxy`, `caddy` |
| M14 — Web Applications | `web-app-hacking`, `burpsuite` |
| M15 — SQL Injection | `sql-injection` |
| M16 — Wireless Hacking | `wireless-hacking` |
| M17 — Mobile Hacking | `mobile-hacking` |
| M18 — IoT/OT Hacking | `iot-ot-hacking` |
| M19 — Cloud Computing | `iam`, `vpc`, `s3`, `aws-cli`, `gcloud`, `azure-cli` |
| M20 — Cryptography | `cryptography`, `cryptography-attacks`, `tls`, `pki` |

---

## Coding Problems

20 problems spanning Easy to Hard, each with solutions in Go, Rust, Python, and TypeScript:

| Problem | Difficulty | Category |
|---------|-----------|----------|
| Two Sum | Easy | Arrays, Hash Maps |
| Valid Parentheses | Easy | Stacks |
| Group Anagrams | Medium | Hashing, Sorting |
| Longest Consecutive Sequence | Medium | HashSet |
| Binary Tree Level Order | Medium | Trees, BFS |
| Word Break | Medium | DP, Tries |
| Merge K Sorted Lists | Hard | Heap, Linked Lists |
| LRU Cache | Medium | Design, Linked Lists |
| Sliding Window Maximum | Hard | Arrays, Monotonic Deque |
| Longest Increasing Subsequence | Medium | DP, Binary Search |
| Edit Distance | Medium | DP, Strings |
| Course Schedule | Medium | Graphs, Topological Sort |
| Serialize/Deserialize Tree | Hard | Trees, BFS |
| Single Number III | Medium | Bit Manipulation |
| Bounded Blocking Queue | Hard | Concurrency |
| Web Crawler Concurrent | Medium | Concurrency, Graphs |
| Rate Limiter | Medium | System Design |
| Consistent Hashing | Medium | System Design |
| Trapping Rain Water | Hard | Arrays, Two Pointers |
| Gaussian Elimination | Hard | Linear Algebra |

Standalone solution files also live in `~/tmp/learning/extra/coding-questions-{go,rust,python-new,typescript}/`.

---

## Design Decisions

### Why Embedded Content (not a Database)

Go's `embed.FS` compiles all markdown into the binary at build time. This means:
- **Zero runtime dependencies** — no config files, no data directory, no internet
- **Atomic updates** — new binary = new content, guaranteed consistent
- **Cross-platform** — same binary works on any OS, any arch
- **Fast** — all content is in-memory, no file I/O at runtime

Trade-off: binary grows with content (~25 MB for 1,098 markdown files). Acceptable for a CLI tool.

### Why stdlib `flag` (not Cobra)

The CLI has ~30 flags. stdlib `flag` handles this cleanly without 200+ transitive dependencies. The binary stays small and builds fast.

### Why No External Router (stdlib net/http)

The REST API has 5 endpoints. `http.HandleFunc` is sufficient. No need for gorilla/mux, chi, or gin.

### Why glamour for Rendering

glamour provides syntax-highlighted, word-wrapped, terminal-aware markdown rendering. It handles tables, code blocks, headers, and links correctly across terminal widths. The pager integration (less/more) handles long sheets.

### Why Two-Tier Content

Separating practical reference (sheets) from theory (details) serves two use cases:
1. **Working engineer**: needs the command NOW → `cs iptables`
2. **Studying engineer**: needs to understand WHY → `cs -d iptables`

Both access the same topic but at different depths.

### Why "Amber Throne" TUI Palette

The TUI uses a warm amber/gold palette inspired by classic terminal aesthetics. It's distinctive, readable on dark and light terminals, and doesn't clash with common terminal color schemes.

---

## Development History

| Commit | Description |
|--------|-------------|
| Initial | Core CLI with ~200 sheets across ~30 categories |
| `529152d` | TUI beautification: Amber Throne palette, bordered panels |
| `05a4378` | 15 coding problems with multi-language solutions |
| `91dfb35` | Wave 7: testing, patterns, auth, quality, API, performance |
| `5e690af` | 25 CS theory topics, 5 more coding problems, TUI alignment fix |
| `c430af5` | 13 networking topics (HTTP/2, HTTP/3, DPDK, AF_XDP, io_uring, etc.) |
| `285b04a` | Kernel tuning (6) + fundamentals with tiered ELI5-to-college (9) |
| `37243db` | JNCIA-Junos certification prep (10 topics) |
| `b6d354e` | CEH v13 certification prep (12 offensive security topics) |

---

## Conventions

- **Go 1.24**, deps: glamour, x/term, bubbletea, bubbles
- **No zerolog** — simple stderr for errors
- **No cobra** — stdlib flag
- **Build flags**: `-trimpath -s -w`
- **Version injection**: `-X main.version=$(VERSION)`
- **REST API**: stdlib `net/http` (no external router)
- **Git email**: `stevie@bellis.tech`
- **Testing**: `go test ./... -count=1 -race`

---

*Generated by the Unheaded Librarian. 549 sheets. 549 details. 59 categories. One binary to rule them all.*
