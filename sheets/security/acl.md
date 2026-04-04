# ACL (POSIX Access Control Lists)

> Fine-grained file permissions beyond traditional owner/group/other model.

## Viewing ACLs

### getfacl

```bash
# Show ACL for a file
getfacl file.txt

# Show ACL for a directory
getfacl /var/www/html

# Omit header (owner, group, flags)
getfacl --omit-header file.txt

# Recursive
getfacl -R /var/www/html

# Numeric UIDs/GIDs
getfacl -n file.txt
```

### Output Format

```
# file: file.txt
# owner: alice
# group: dev
user::rwx              # Owner permissions
user:bob:rw-           # Named user
group::r-x             # Owning group
group:ops:r--          # Named group
mask::rwx              # Effective rights mask
other::r--             # Others
```

## Setting ACLs

### setfacl

```bash
# Grant user read/write
setfacl -m u:bob:rw file.txt

# Grant group read
setfacl -m g:ops:r file.txt

# Set permissions for others
setfacl -m o::rx file.txt

# Multiple entries at once
setfacl -m u:bob:rw,g:ops:r,g:dev:rwx file.txt

# Remove a specific entry
setfacl -x u:bob file.txt

# Remove all ACL entries (reset to base permissions)
setfacl -b file.txt

# Remove all default ACL entries
setfacl -k directory/
```

## Default ACLs (Inheritance)

### Set Defaults on Directories

```bash
# New files/dirs in this directory inherit these ACLs
setfacl -d -m u:bob:rwx /var/www/html
setfacl -d -m g:dev:rx /var/www/html

# View default ACLs
getfacl /var/www/html
# default:user::rwx
# default:user:bob:rwx
# default:group::r-x
# default:group:dev:r-x
# default:mask::rwx
# default:other::r-x

# Copy ACL from one file as default ACL on a directory
getfacl file.txt | setfacl -d -M- /var/www/html
```

## Mask

### Effective Permissions

```bash
# The mask limits the maximum permissions for named users and groups
# Effective = entry AND mask
setfacl -m m::r /var/www/html    # Restrict all named entries to read-only

# Example: user:bob:rwx with mask::r-- results in effective r--
```

## Recursive Operations

```bash
# Apply ACLs recursively
setfacl -R -m u:bob:rx /var/www/html

# Set defaults recursively (directories only)
setfacl -R -d -m g:dev:rwx /var/www/html

# Remove all ACLs recursively
setfacl -R -b /var/www/html

# Apply different ACLs for files vs directories
# Files: read-only, Directories: read+execute
find /var/www -type f -exec setfacl -m u:bob:r {} +
find /var/www -type d -exec setfacl -m u:bob:rx {} +
```

## Backup and Restore

```bash
# Backup ACLs to a file
getfacl -R /var/www/html > acl_backup.txt

# Restore ACLs from backup
setfacl --restore=acl_backup.txt

# Copy ACLs from one file to another
getfacl source.txt | setfacl --set-file=- target.txt
```

## Tips

- A `+` in `ls -l` output (e.g., `-rw-r--r--+`) indicates extended ACLs are set.
- The mask is automatically recalculated when you add ACL entries; set it explicitly if needed.
- Default ACLs only apply to new files created inside the directory, not existing ones.
- ACLs require the filesystem to be mounted with the `acl` option (default on ext4, XFS).
- `cp -p` and `rsync -A` preserve ACLs; `mv` within the same filesystem preserves them automatically.
- `chmod` modifies the mask when ACLs are present, which can restrict named user/group permissions unexpectedly.

## See Also

- pam, selinux, apparmor, hardening-linux, auditd

## References

- [getfacl(1) Man Page](https://man7.org/linux/man-pages/man1/getfacl.1.html)
- [setfacl(1) Man Page](https://man7.org/linux/man-pages/man1/setfacl.1.html)
- [acl(5) Man Page](https://man7.org/linux/man-pages/man5/acl.5.html)
- [POSIX.1e ACL Specification](https://www.usenix.org/legacy/events/usenix01/freenix01/full_papers/gruenbacher/gruenbacher.pdf)
- [Arch Wiki — Access Control Lists](https://wiki.archlinux.org/title/Access_control_lists)
- [Red Hat RHEL 9 — Managing ACLs](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_file_systems/managing-acl_managing-file-systems)
- [Ubuntu Manpage — setfacl](https://manpages.ubuntu.com/manpages/noble/man1/setfacl.1.html)
- [Kernel Filesystem ACL Documentation](https://www.kernel.org/doc/html/latest/filesystems/acl.html)
