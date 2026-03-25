# dnf (Dandified YUM)

> Package manager for Fedora, RHEL 8+, CentOS Stream, and derivatives — manages RPM packages.

## Install and Remove

```bash
sudo dnf install nginx                 # install package
sudo dnf install nginx-1.24.0         # install specific version
sudo dnf install ./local-package.rpm   # install local RPM
sudo dnf install -y httpd php php-fpm  # install multiple, skip confirmation
sudo dnf reinstall nginx               # reinstall package
sudo dnf localinstall package.rpm      # install local (older alias)

sudo dnf remove nginx                  # remove package
sudo dnf autoremove                    # remove orphaned dependencies
sudo dnf remove --noautoremove nginx   # remove without cleaning deps
```

## Update

```bash
sudo dnf check-update                 # check for available updates
sudo dnf upgrade                      # upgrade all packages
sudo dnf upgrade nginx                # upgrade specific package
sudo dnf upgrade --security           # security updates only
sudo dnf upgrade-minimal              # minimal updates (security + bugfix)
sudo dnf distro-sync                  # sync to latest in repo (can downgrade)
sudo dnf downgrade nginx              # downgrade to previous version
```

## Search and Info

```bash
dnf search nginx                       # search names + summaries
dnf search all "web server"            # search all metadata
dnf info nginx                         # detailed package info
dnf list installed                     # list installed packages
dnf list available                     # list available packages
dnf list updates                       # list packages with updates
dnf list --installed 'php*'            # glob match installed
dnf provides /usr/sbin/nginx           # find package owning a file
dnf provides "*/bin/htop"              # find by path pattern
dnf repoquery --requires nginx         # show dependencies
dnf repoquery --whatrequires nginx     # show reverse dependencies
dnf repoquery -l nginx                 # list files in package (not installed)
```

## Groups

```bash
dnf group list                         # list available groups
dnf group list --hidden                # include hidden groups
dnf group info "Development Tools"     # show group contents
sudo dnf group install "Development Tools"
sudo dnf group remove "Development Tools"
sudo dnf group upgrade "Development Tools"

# shorter aliases
sudo dnf groupinstall "Development Tools"
```

## History

```bash
dnf history                            # show transaction history
dnf history info 15                    # details of transaction 15
dnf history list --reverse             # oldest first
sudo dnf history undo 15               # undo transaction 15
sudo dnf history redo 15               # redo transaction 15
sudo dnf history rollback 15           # rollback to state before transaction 15
```

## Modules (RHEL/CentOS Stream)

```bash
dnf module list                        # list available modules
dnf module list nodejs                 # list nodejs module streams
dnf module info nodejs:18              # info on stream 18
sudo dnf module enable nodejs:18       # enable stream
sudo dnf module install nodejs:18      # install module stream
sudo dnf module reset nodejs           # reset module to default
sudo dnf module disable nodejs         # disable module entirely
sudo dnf module switch-to nodejs:20    # switch streams
```

## Repository Management

```bash
dnf repolist                           # list enabled repos
dnf repolist all                       # list all repos (enabled + disabled)
dnf repoinfo                           # detailed repo info

# enable/disable repos
sudo dnf config-manager --set-enabled powertools
sudo dnf config-manager --set-disabled testing

# add a repo
sudo dnf config-manager --add-repo https://rpm.example.com/repo.repo

# install from specific repo
sudo dnf install --repo=epel htop

# temporary enable/disable
sudo dnf install --enablerepo=testing package-name
sudo dnf upgrade --disablerepo=unstable
```

### Adding EPEL (Extra Packages)

```bash
# Fedora
sudo dnf install epel-release

# RHEL 9 / CentOS Stream 9
sudo dnf install https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm
```

## Configuration

```bash
# main config
cat /etc/dnf/dnf.conf

# useful options in /etc/dnf/dnf.conf
[main]
gpgcheck=1                             # verify package signatures
installonly_limit=3                    # keep N kernel versions
clean_requirements_on_remove=True      # autoremove deps on remove
fastestmirror=True                     # pick fastest mirror
max_parallel_downloads=10              # parallel downloads
defaultyes=True                        # default to yes for prompts
keepcache=True                         # keep downloaded RPMs
```

## Cache Management

```bash
sudo dnf clean all                     # clear all caches
sudo dnf clean packages                # clear cached RPMs
sudo dnf clean metadata                # clear repo metadata
sudo dnf makecache                     # rebuild cache
```

## RPM Queries (Low-Level)

```bash
rpm -qa                                # list all installed packages
rpm -qi nginx                          # info on installed package
rpm -ql nginx                          # list files from installed package
rpm -qf /usr/sbin/nginx                # which package owns this file
rpm -qp package.rpm -l                 # list files in an RPM file
rpm -V nginx                           # verify installed package
rpm --import https://example.com/RPM-GPG-KEY  # import GPG key
```

## Tips

- `dnf` replaced `yum` starting with Fedora 22 and RHEL 8. The `yum` command still works as an alias.
- `dnf check-update` exits with code 100 if updates are available, 0 if not. Useful in scripts.
- `dnf provides` is the fastest way to find which package contains a specific file or command.
- `--setopt=install_weak_deps=False` skips weak (recommended) dependencies for minimal installs.
- Module streams lock you to a major version. `dnf module reset` then re-enable to switch streams.
- `dnf history undo` is invaluable for reverting a bad install or upgrade.
- `keepcache=True` in dnf.conf caches RPMs in `/var/cache/dnf/` -- useful for offline or repeated installs.
- `max_parallel_downloads=10` dramatically speeds up large installs and upgrades.
- Unlike apt, `dnf upgrade` handles both package updates and dependency changes in one command.
- `dnf needs-restarting -r` checks if a reboot is needed (after kernel/glibc updates).
