# cs — Cheatsheet CLI

Single-binary Go CLI with **685 embedded cheatsheets** and **685 deep-dive theory pages** across **59 categories**. Built-in calculator, subnet calculator, fuzzy search, interactive TUI, REST API, shell completions. Better than man pages — real examples, clear explanations, official references, and deep mathematical/theoretical analysis instantly searchable.

Covers **11 certification domains**: CCNP DC, CCNP Enterprise, CCIE Enterprise Infrastructure, CCIE Service Provider, CCIE Security, CCIE Automation, JNCIE-SP, JNCIE-SEC, CompTIA Linux+, CISSP, and C|RAGE.

## Install

```bash
git clone git@github.com:bellistech/cs.git
cd cs
make install        # builds and installs to /usr/local/bin

# enable tab completion (pick one)
echo 'eval "$(cs --completions bash)"' >> ~/.bashrc
echo 'eval "$(cs --completions zsh)"'  >> ~/.zshrc
echo 'cs --completions fish | source'  >> ~/.config/fish/config.fish
```

## Usage

```bash
cs                        # list all topics grouped by category
cs lvm                    # show LVM cheatsheet
cs storage                # list all storage-related sheets
cs lvm extend             # show only the "extend" section
cs -s lvextend            # search across all sheets
cs -l                     # list all topics with descriptions
cs -i                     # interactive TUI browser
cs --add mysheet.md       # add a custom cheatsheet
cs --edit lvm             # customize a sheet in $EDITOR
cs --random               # show a random cheatsheet
cs --count                # show statistics with per-category bar chart
```

Fuzzy matching is built in — `cs kube` finds kubernetes, `cs lv` finds lvm.

### Deep Dive Mode

```bash
# Show the theory, math, and internals behind any topic
cs -d bgp                     # BGP peering formula, convergence, dampening decay
cs -d tcp                     # window math, congestion control, RTT estimation
cs -d postgresql              # query planner cost model, B-tree splits, MVCC
cs -d kubernetes              # scheduler scoring, Raft consensus, HPA formula
cs -d tls                     # handshake state machine, ECDHE math, cipher suites
cs -d bgp --prereqs           # show prerequisites for a detail page
```

Every topic has a companion deep dive with formulas, worked examples, complexity analysis, and engineering tradeoffs.

### Knowledge Graph

```bash
# Cross-references — discover related topics
cs --related bgp              # shows ospf, is-is, mpls, tcp, subnetting
cs --related docker           # shows podman, kubernetes, containerd

# Compare two tools side by side
cs compare docker podman      # feature comparison table
cs compare ext4 xfs           # filesystem comparison

# Learning paths — ordered by prerequisites
cs learn networking           # foundational → advanced topic order
cs learn databases            # sql → postgresql → redis progression
```

### Built-in Tools

```bash
# Calculator — arithmetic, hex/oct/bin, units
cs calc "2**10"               # 1024
cs calc "0xff * 2"            # 510
cs calc "1<<16"               # 65536
cs calc "10GB / 1500bytes"    # unit-aware: 6,666,666 packets
cs calc "10Gbps / 8"          # 1.25 Gbps
cs calc help                  # show full calculator manual

# Subnet Calculator — CIDR breakdown
cs subnet 10.0.0.0/24         # network, broadcast, host range, mask
cs subnet 172.16.0.0/20       # usable hosts, wildcard, binary mask
cs subnet help                # show full subnet calculator manual

# Math Verification — validate formulas in detail pages
cs verify bgp                 # check worked examples against calculator
cs verify                     # verify all detail pages (CI-friendly, exit 1 on fail)
```

### Interactive TUI

```bash
cs -i                         # launch full-screen interactive browser
```

Browse categories, fuzzy-filter topics, read sheets and detail pages — all with keyboard navigation (j/k, enter, /, d for detail, esc to go back, q to quit).

### REST API / Daemon Mode

```bash
cs serve                      # start API on 127.0.0.1:9876
cs serve 8080                 # custom port
```

Endpoints:
- `GET /api/topics` — list all topics
- `GET /api/topics/:name` — get sheet content (JSON)
- `GET /api/topics/:name/detail` — get detail page
- `GET /api/topics/:name/related` — get related topics
- `GET /api/categories` — list categories
- `GET /api/search?q=<query>` — search across sheets
- `GET /api/compare?a=<X>&b=<Y>` — compare two topics
- `POST /api/calc` — evaluate expression
- `POST /api/subnet` — subnet calculator
- `GET /api/verify/:name` — verify detail math
- `GET /api/stats` — statistics
- `GET /api/bookmarks` — list bookmarks
- `POST /api/bookmarks/:name` — toggle bookmark

### Export & Share

```bash
cs lvm --format markdown      # raw markdown (pipe to pbcopy, wiki, etc.)
cs lvm --format json          # structured JSON with sections, see_also, etc.
cs bgp --format json | jq .   # pipe to jq for processing
```

### Bookmarks

```bash
cs --star lvm                 # bookmark a topic
cs --star lvm                 # run again to remove
cs --starred                  # list all bookmarked topics
```

### Self-Update

```bash
cs --update                   # check GitHub for new releases and update
```

## Categories (59)

| Category | Sheets | Highlights |
|----------|--------|------------|
| networking | 135 | bgp, ospf, is-is, mpls, vxlan, eigrp, lisp, dmvpn, flexvpn, sd-access, multicast, qos, carrier-ethernet, fibre-channel, fcoe, roce, vpc, aci, pbr, gre-tunnels, ipv6-advanced, vrf, tcp, udp, quic, dns, ipsec, tcpdump, nmap, tshark |
| security | 84 | tls, pki, dot1x, macsec, copp, nxos-security, network-security-infra, cryptography, ids-ips, incident-response, threat-hunting, forensics, container-security, hardening-linux, cissp domains |
| offensive | 37 | recon, web-attacks, privilege-escalation, lateral-movement, password-attacks, metasploit, burpsuite, CEH v13 topics |
| system | 34 | kernel, systemd, journalctl, htop, iostat, vmstat, strace, lsof, dmesg, sysctl, gdb, valgrind |
| juniper | 30 | junos, junos-routing, junos-firewall, junos-mpls, junos-multicast, jncie-sp, jncie-sec topics |
| cs-theory | 25 | algorithms, data-structures, complexity, graph-theory, automata, computability |
| coding-problems | 20 | lru-cache, sliding-window, merge-k-sorted, edit-distance, rate-limiter |
| databases | 16 | postgresql, mysql, redis, sqlite, sql, time-series, graph-databases |
| ai-ml | 15 | llm-fundamentals, prompt-engineering, rag, transformers, fine-tuning, c\|rage topics |
| monitoring | 14 | prometheus, grafana, netflow-ipfix, ip-sla, model-driven-telemetry, snmp, sflow |
| data-formats | 13 | json, yaml, xml, toml, jq, awk, sed, regex, ascii, unicode |
| orchestration | 11 | kubernetes, helm, service-mesh, istio |
| containers | 11 | docker, docker-compose, lxd, containerd, podman |
| testing | 11 | unit-testing, integration-testing, load-testing, chaos-engineering, fuzz-testing |
| languages | 10 | c, go, python, ruby, rust, javascript, typescript, lua, make |
| config-mgmt | 10 | ansible, terraform, salt, puppet, chef, dc-automation, eem |
| compliance | 10 | nist, fedramp, soc2, pci-dss, hipaa, gdpr, iso27001 |
| storage | 9 | lvm, zfs, btrfs, ceph, mdadm, san-storage |
| fundamentals | 9 | networking-basics, linux-basics, security-fundamentals |
| terminal | 8 | tmux, screen, terminal-emulators |
| shell | 8 | bash, zsh, shell-scripting |
| ci-cd | 8 | github-actions, gitlab-ci, jenkins, argocd |
| cloud | 7 | aws-cli, gcloud, azure-cli |
| big-data | 7 | spark, hadoop, data-pipelines |
| network-tools | 7 | ssh-tunneling, rsync, scp, sftp, socat |
| disk | 7 | fdisk, parted, mount, fstab, df, du, ncdu |
| package-managers | 6 | apt, dnf, brew, pip, npm, cargo |
| kernel-tuning | 6 | sysctl, cgroups, namespaces, ebpf |
| filesystems | 6 | ext4, xfs, btrfs, zfs |
| api | 6 | rest, graphql, grpc, openapi |
| users | 5 | useradd, usermod, passwd, groups, sudo |
| quality | 5 | code-review, linting, static-analysis |
| process | 5 | cron, at, nice, kill |
| performance | 5 | perf, bpftrace, ebpf |
| patterns | 5 | distributed-systems, microservices, event-driven |
| data-science | 5 | pandas, numpy, matplotlib |
| auth | 5 | oauth, oidc, saml, ldap |
| archives | 5 | tar, gzip, xz, zip, 7z |
| virtualization | 4 | kvm, qemu, libvirt, vagrant |
| vcs | 4 | git |
| service-mesh | 4 | istio, envoy, linkerd |
| provisioning | 4 | cloud-init, nix, packer, vagrant |
| network-os | 4 | cisco-ios, cisco-nexus, cisco-ios-xr, junos |
| infrastructure | 4 | cisco-ucs, data-center-design |
| editors | 4 | vim, neovim, emacs, nano |
| web-servers | 3 | nginx, haproxy, caddy |
| queuing | 3 | kafka, rabbitmq, nats |
| messaging | 3 | kafka, rabbitmq, nats |
| logs | 3 | rsyslog, logrotate, elk |
| load-testing | 3 | k6, locust, wrk |
| iac | 3 | terraform, pulumi, crossplane |
| email | 3 | postfix, dovecot, spf-dkim |
| data-engineering | 3 | airflow, dbt, kafka-streams |
| backup | 3 | restic, borgbackup, velero |
| web | 2 | css, html |
| serverless | 2 | lambda, cloud-functions |
| secrets | 2 | vault, sops |
| dns | 2 | bind, dnsmasq |
| build-systems | 2 | make, bazel |

Every sheet includes `## References` with official documentation, RFCs, man pages, vendor guides, and project wikis. Every sheet includes `## See Also` cross-references to related topics.

## Custom Sheets

Custom sheets live in `~/.config/cs/sheets/<category>/<topic>.md` and override embedded ones.

```bash
# Add a custom sheet (prompts for category)
cs --add ~/my-cheatsheet.md

# Edit an existing sheet (copies embedded -> custom for modification)
cs --edit docker

# Custom sheets take priority over embedded ones
```

## Sheet Format

Sheets are markdown with a consistent structure:

```markdown
# Tool Name (Full Description)

One-liner explaining what this tool does.

## Functional Area

### Specific Operation

` ` `bash
# Comment explaining the command
command --flag value

# Another example with real values
command --option actual-value
` ` `

## Tips

- Practical gotcha or important note
- Performance consideration

## See Also

- related-topic1, related-topic2, related-topic3

## References

- [Official Documentation](https://example.com/docs)
- [RFC 1234 — Protocol Title](https://www.rfc-editor.org/rfc/rfc1234)
- [man page(1)](https://man7.org/linux/man-pages/man1/page.1.html)
```

## Build

```bash
make build      # build ./cs binary
make test       # run tests with race detector
make install    # install to /usr/local/bin
make lint       # go vet + staticcheck
make fmt        # gofmt -s -w
```

Requires Go 1.24+.

## License

MIT
