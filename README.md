# cs

Single-binary Go CLI: **758 cheatsheets** + **722 deep-dive pages** across **63 categories**. Built-in calculator, subnet calculator, fuzzy search, interactive TUI, REST API, shell completions. Every sheet self-contained — paste-ready commands with expected output, every concept defined in-sheet, every cross-reference resolved.

Certification coverage: CCNP DC, CCNP Enterprise, CCIE EI/SP/Security/Automation, JNCIE-SP, JNCIE-SEC, CompTIA Linux+, CISSP, C|RAGE.

## Install

```bash
git clone git@github.com:bellistech/cs.git
cd cs
make install        # builds and installs to /usr/local/bin

# tab completion (pick one)
echo 'eval "$(cs --completions bash)"' >> ~/.bashrc
echo 'eval "$(cs --completions zsh)"'  >> ~/.zshrc
echo 'cs --completions fish | source'  >> ~/.config/fish/config.fish
```

## Usage

```bash
cs                        # list all topics by category
cs lvm                    # show LVM cheatsheet
cs storage                # list sheets in a category
cs lvm extend             # show only the "extend" section
cs -s lvextend            # search across all sheets
cs -l                     # list with descriptions
cs -i                     # interactive TUI
cs --add mysheet.md       # custom cheatsheet
cs --edit lvm             # override sheet in $EDITOR
cs --random               # random cheatsheet
cs --count                # per-category statistics + bar chart
```

Fuzzy match: `cs kube` → kubernetes. `cs lv` → lvm.

### Deep dive

```bash
cs -d bgp                 # peering formula, convergence, dampening decay
cs -d tcp                 # window math, congestion control, RTT estimation
cs -d postgresql          # query planner cost model, B-tree splits, MVCC
cs -d kubernetes          # scheduler scoring, Raft consensus, HPA formula
cs -d tls                 # handshake state machine, ECDHE math, cipher suites
cs -d bgp --prereqs       # prerequisites for a detail page
```

722 detail pages — formulas, worked examples, complexity analysis, engineering tradeoffs.

### Knowledge graph

```bash
cs --related bgp          # ospf, is-is, mpls, tcp, subnetting
cs --related docker       # podman, kubernetes, containerd
cs compare docker podman  # feature comparison table
cs compare ext4 xfs       # filesystem comparison
```

### Learning paths

```bash
cs learn networking       # prerequisite-ordered topic list
cs learn databases        # sql → postgresql → redis progression
cs --prereqs bgp          # show prerequisites for a deep-dive page
cs ramp-up                # one ELI5 sheet per topic (kernel today; more topics planned)
```

The `ramp-up` category is narrative-shaped — one comprehensive ELI5-voiced sheet per topic. Vocabulary tables defining every term, ASCII diagrams, paste-and-runnable shell with literal expected output, broken-then-fixed confusion pairs. Designed for absolute beginners; once a sheet feels easy, the dense reference (`cs fundamentals <topic>` / `cs -d <topic>`) is one command away.

### Built-in tools

```bash
cs calc "2**10"               # 1024
cs calc "0xff * 2"            # 510
cs calc "1<<16"               # 65536
cs calc "10GB / 1500bytes"    # unit-aware: 6,666,666 packets
cs calc "10Gbps / 8"          # 1.25 Gbps
cs calc help                  # full calculator manual

cs subnet 10.0.0.0/24         # network, broadcast, host range, mask
cs subnet 172.16.0.0/20       # usable hosts, wildcard, binary mask
cs subnet help                # full subnet calculator manual

cs verify bgp                 # check worked examples against the calculator
cs verify                     # verify all detail pages (CI-friendly, exit 1 on fail)
```

### Interactive TUI

```bash
cs -i                         # full-screen interactive browser
```

Keys: `j`/`k` navigate, `enter` open, `/` filter, `d` detail page, `esc` back, `q` quit.

### REST API

```bash
cs serve                      # 127.0.0.1:9876
cs serve 8080                 # custom port
```

| Method | Endpoint | Returns |
|--------|----------|---------|
| GET    | `/api/topics` | all topics |
| GET    | `/api/topics/:name` | sheet content (JSON) |
| GET    | `/api/topics/:name/detail` | detail page |
| GET    | `/api/topics/:name/related` | related topics |
| GET    | `/api/categories` | category list |
| GET    | `/api/search?q=<query>` | full-text search |
| GET    | `/api/compare?a=<X>&b=<Y>` | compare two topics |
| POST   | `/api/calc` | evaluate expression |
| POST   | `/api/subnet` | subnet calculator |
| GET    | `/api/verify/:name` | verify detail math |
| GET    | `/api/stats` | statistics |
| GET    | `/api/bookmarks` | list bookmarks |
| POST   | `/api/bookmarks/:name` | toggle bookmark |

### Export, bookmarks, self-update

```bash
cs lvm --format markdown      # raw markdown (pipe to pbcopy, wiki, etc.)
cs lvm --format json          # structured JSON
cs bgp --format json | jq .   # pipe to jq

cs --star lvm                 # toggle bookmark
cs --starred                  # list bookmarks

cs --update                   # check GitHub releases and update
```

### iOS bindings (gomobile)

`mobile/Cscore.xcframework/` is a generated artifact, gitignored. Rebuild:

```bash
make mobile-ios               # gomobile bind ./mobile/ → mobile/Cscore.xcframework
```

Used by the React Native `CsApp/` project for iOS distribution.

## Categories

63 categories. Run `cs --count` for the live breakdown with sheet counts and a per-category bar chart. Starting points:

| Goal | Entry point |
|------|-------------|
| Total beginner — kernel ELI5       | `cs ramp-up linux-kernel-eli5` |
| Network engineer (CCNP/CCIE)       | `cs networking` |
| Security / pentesting              | `cs security`, `cs offensive` |
| Platform / SRE                     | `cs orchestration kubernetes` |
| Linux internals                    | `cs fundamentals linux-kernel-internals` |
| Language reference                 | `cs languages` |
| Database internals                 | `cs databases` |

Every sheet includes `## See Also` cross-references and `## References` with official docs, RFCs, man pages, vendor guides.

## Custom sheets

Custom sheets live in `~/.config/cs/sheets/<category>/<topic>.md` and override embedded ones.

```bash
cs --add ~/my-cheatsheet.md   # prompts for category
cs --edit docker              # copies embedded → custom for editing
```

## Sheet format

```markdown
# Tool Name (Full Description)

One-liner.

## Functional Area

### Specific Operation

` ` `bash
# Comment
command --flag value
` ` `

## Tips

- Practical gotcha or note

## See Also

- related-topic1, related-topic2

## References

- [Official Docs](https://example.com)
- [RFC 1234](https://www.rfc-editor.org/rfc/rfc1234)
- [man page(1)](https://man7.org/linux/man-pages/man1/page.1.html)
```

## Build

```bash
make build      # ./cs binary
make test       # tests with race detector
make install    # install to /usr/local/bin
make lint       # go vet + staticcheck
make fmt        # gofmt -s -w
make mobile-ios # rebuild iOS xcframework
```

Requires Go 1.24+.

## License

GPL-3.0-or-later. See [LICENSE](LICENSE).
