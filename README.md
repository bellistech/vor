# cs — Cheatsheet CLI

Single-binary CLI with 97 embedded cheatsheets. Better than man pages — real examples, clear explanations, instantly searchable.

## Install

```bash
git clone https://github.com/bellistech/cs.git
cd cs
make install
```

## Usage

```bash
cs                        # list all topics grouped by category
cs lvm                    # show LVM cheatsheet
cs storage                # list all storage-related sheets
cs lvm extend             # show only the "extend" section
cs -s lvextend            # search across all sheets
cs -l                     # list all topics with descriptions
cs --add mysheet.md       # add a custom cheatsheet
cs --edit lvm             # customize a sheet in $EDITOR
```

## Categories

| Category | Topics |
|----------|--------|
| shell | bash, zsh, shell-scripting, tmux, screen |
| editors | vim, neovim |
| vcs | git |
| package-managers | apt, dnf, brew, pip, npm, cargo |
| containers | docker, docker-compose, lxd, containerd, podman |
| orchestration | kubernetes, helm |
| networking | ss, netstat, ip, iptables, nftables, tcpdump, curl, wget, dig, nslookup, mtr, nc, nmap, tshark, ethtool |
| network-tools | ssh-tunneling, rsync, scp, sftp, socat |
| security | openssl, gpg, ssh, fail2ban, ufw, selinux, apparmor, wireguard, certbot |
| storage | lvm, zfs, btrfs, ceph, mdadm |
| disk | fdisk, parted, mount, fstab, df, du, ncdu |
| filesystems | ext4, xfs |
| system | systemd, journalctl, htop, iostat, vmstat, sar, strace, lsof, ps, dmesg, sysctl, find, grep, systemd-timers |
| process | cron, at, nice, kill |
| users | useradd, usermod, passwd, groups, sudo |
| archives | tar, gzip, xz, zip, 7z |
| config-mgmt | ansible, terraform, salt, puppet, chef |
| web-servers | nginx, haproxy, caddy |
| databases | postgresql, mysql, redis, sqlite |
| languages | python, go, rust, make |
| data-formats | yaml, json, xml, toml, jq, awk, sed |
| provisioning | nix |
| dns | bind, dnsmasq |
| monitoring | prometheus, grafana |
| logs | rsyslog, logrotate |
| performance | perf, bpftrace, ebpf |

## Custom Sheets

Custom sheets live in `~/.config/cs/sheets/<category>/<topic>.md` and override embedded ones.

```bash
# Add a custom sheet
cs --add ~/my-cheatsheet.md

# Edit an existing sheet (copies embedded → custom for modification)
cs --edit docker
```

## Sheet Format

```markdown
# Tool Name (Full Description)

One-liner explaining what this tool does.

## Section

` + "```" + `bash
# Comment explaining the command
command --flag value
` + "```" + `

## Tips

- Gotcha or important note
```

## Build

```bash
make build      # build ./cs binary
make test       # run tests with race detector
make install    # install to /usr/local/bin
```

## License

MIT
