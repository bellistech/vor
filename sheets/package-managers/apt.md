# apt (Advanced Package Tool)

> Package manager for Debian, Ubuntu, and derivatives — install, update, and manage .deb packages.

## Update and Upgrade

```bash
sudo apt update                        # refresh package index
sudo apt upgrade                       # upgrade installed packages
sudo apt full-upgrade                  # upgrade with dependency changes (add/remove)
sudo apt dist-upgrade                  # same as full-upgrade (older name)
sudo apt update && sudo apt upgrade -y # non-interactive update + upgrade
```

## Install and Remove

### Installing

```bash
sudo apt install nginx                 # install package
sudo apt install nginx=1.24.0-1       # install specific version
sudo apt install ./local-package.deb   # install local .deb file
sudo apt install -y nginx curl htop    # install multiple, skip confirmation
sudo apt install --no-install-recommends nginx  # minimal install
sudo apt install -f                    # fix broken dependencies
sudo apt reinstall nginx               # reinstall without removing first
```

### Removing

```bash
sudo apt remove nginx                  # remove package (keep config)
sudo apt purge nginx                   # remove package + config files
sudo apt autoremove                    # remove orphaned dependencies
sudo apt autoremove --purge            # remove orphaned + their configs
sudo apt clean                         # delete all cached .deb files
sudo apt autoclean                     # delete only obsolete cached .debs
```

## Search and Info

```bash
apt search nginx                       # search package names + descriptions
apt list --installed                   # list installed packages
apt list --upgradable                  # list packages with available upgrades
apt list --all-versions nginx          # show all available versions
apt show nginx                         # detailed package info
apt depends nginx                      # show dependencies
apt rdepends nginx                     # show reverse dependencies (what depends on it)
apt policy nginx                       # show installed and candidate versions + sources
```

## apt-cache (Query Cache)

```bash
apt-cache search "web server"          # search descriptions
apt-cache show nginx                   # detailed info from cache
apt-cache showpkg nginx                # package relationships
apt-cache madison nginx                # show versions from all sources
apt-cache depends nginx                # dependency tree
apt-cache rdepends nginx               # reverse dependency tree
apt-cache stats                        # cache statistics
apt-cache pkgnames ngi                 # package names starting with prefix
```

## dpkg (Low-Level)

```bash
dpkg -i package.deb                    # install local .deb
dpkg -r nginx                          # remove package
dpkg -P nginx                          # purge (remove + config)
dpkg -l                                # list all installed packages
dpkg -l nginx                          # check if specific package is installed
dpkg -L nginx                          # list files installed by package
dpkg -S /usr/sbin/nginx                # find which package owns a file
dpkg --configure -a                    # fix interrupted installs
dpkg-reconfigure tzdata                # reconfigure package
```

## Sources and Repositories

### Managing Sources

```bash
# list sources
cat /etc/apt/sources.list
ls /etc/apt/sources.list.d/

# add repository (Ubuntu PPA)
sudo add-apt-repository ppa:deadsnakes/ppa
sudo apt update

# add custom repository
echo "deb https://apt.example.com/repo stable main" | \
    sudo tee /etc/apt/sources.list.d/example.list

# add GPG key for repository
curl -fsSL https://apt.example.com/key.gpg | \
    sudo gpg --dearmor -o /usr/share/keyrings/example.gpg

# modern signed-by format (preferred)
echo "deb [signed-by=/usr/share/keyrings/example.gpg] https://apt.example.com/repo stable main" | \
    sudo tee /etc/apt/sources.list.d/example.list

# remove repository
sudo add-apt-repository --remove ppa:deadsnakes/ppa
```

## Pinning (Version Control)

```bash
# /etc/apt/preferences.d/nginx
# pin nginx to a specific version
Package: nginx
Pin: version 1.24.0*
Pin-Priority: 1001

# hold package (prevent upgrades via dpkg)
sudo apt-mark hold nginx
sudo apt-mark unhold nginx
sudo apt-mark showhold                 # list held packages

# pin priorities:
# 1001+ = force install even if downgrade
# 990   = default for target release
# 500   = default for other releases
# 100   = installed packages
# -1    = never install
```

## History and Logs

```bash
# apt history
cat /var/log/apt/history.log
zcat /var/log/apt/history.log.1.gz     # older history

# dpkg log (more detail)
cat /var/log/dpkg.log
```

## Offline and Downloading

```bash
apt download nginx                     # download .deb without installing
apt source nginx                       # download source package
sudo apt install --download-only nginx # download to cache only
apt-get changelog nginx                # view package changelog
```

## Tips

- `apt` is the user-friendly frontend; `apt-get` is the scriptable backend. In scripts, prefer `apt-get` for stable output.
- `sudo apt update` only refreshes the package index -- it does not install anything.
- `apt autoremove` can remove packages you still need if they were auto-installed. Check the list before confirming.
- `--no-install-recommends` significantly reduces install size for server deployments.
- `dpkg -S $(which command)` quickly finds which package provides a binary.
- `apt policy package` shows exactly which repo a version comes from -- essential for debugging.
- `apt-mark hold` prevents a package from being upgraded. Use this for critical packages where you need version stability.
- After adding a new source, always run `sudo apt update` before trying to install from it.
- `apt purge` removes config files; `apt remove` does not. Use `purge` for a clean uninstall.
- For unattended upgrades, install `unattended-upgrades` and configure `/etc/apt/apt.conf.d/50unattended-upgrades`.

## References

- [apt(8) Man Page](https://manpages.debian.org/bookworm/apt/apt.8.en.html)
- [apt-get(8) Man Page](https://manpages.debian.org/bookworm/apt/apt-get.8.en.html)
- [apt-cache(8) Man Page](https://manpages.debian.org/bookworm/apt/apt-cache.8.en.html)
- [dpkg(1) Man Page](https://manpages.debian.org/bookworm/dpkg/dpkg.1.en.html)
- [Debian Package Management](https://www.debian.org/doc/manuals/debian-reference/ch02.en.html)
- [Ubuntu Package Management Guide](https://ubuntu.com/server/docs/package-management)
- [sources.list(5) Man Page](https://manpages.debian.org/bookworm/apt/sources.list.5.en.html)
- [APT Preferences (Pinning)](https://manpages.debian.org/bookworm/apt/apt_preferences.5.en.html)
- [Debian APT HOWTO](https://www.debian.org/doc/manuals/apt-howto/)
- [Unattended Upgrades](https://wiki.debian.org/UnattendedUpgrades)
