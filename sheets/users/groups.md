# groups (group management)

Manage groups and group membership.

## View Groups

### Check Membership

```bash
# Groups for current user
groups

# Groups for a specific user
groups deploy

# Detailed: UID, GID, all groups
id deploy

# Just group IDs
id -G deploy

# Just group names
id -Gn deploy

# Just primary group
id -gn deploy
```

## Create Groups

### groupadd

```bash
# Create a group
groupadd developers

# Create with specific GID
groupadd -g 2000 developers

# Create system group (low GID)
groupadd -r myservice
```

## Modify Groups

### groupmod

```bash
# Rename a group
groupmod -n engineering developers

# Change GID
groupmod -g 2001 developers
```

## Delete Groups

### groupdel

```bash
# Delete a group
groupdel developers

# Cannot delete a group that is a user's primary group
# Change the user's primary group first:
usermod -g users deploy
groupdel oldgroup
```

## Manage Members

### gpasswd

```bash
# Add user to group
gpasswd -a deploy docker

# Remove user from group
gpasswd -d deploy docker

# Set group administrators
gpasswd -A deploy developers

# Set group members (replaces all)
gpasswd -M alice,bob,carol developers

# Set a group password (rarely used)
gpasswd developers

# Remove group password
gpasswd -r developers
```

### usermod (alternative)

```bash
# Append to supplementary groups
usermod -aG docker,sudo deploy
```

## Switch Group

### newgrp

```bash
# Switch primary group for the current session
newgrp docker

# Files created after this will have docker as group owner
touch testfile
ls -la testfile   # ... deploy docker ... testfile

# Exit back to original primary group
exit
```

## Primary vs Supplementary

### Understanding the Difference

```bash
# Primary group: set at user creation, used for new file ownership
# Supplementary groups: additional access (e.g. sudo, docker)

# Check primary group
id -gn deploy     # shows primary group name

# Check all supplementary groups
id -Gn deploy     # shows all groups including primary

# Change primary group
usermod -g developers deploy

# /etc/passwd stores primary group (GID field)
# /etc/group stores supplementary memberships
```

## Relevant Files

### Configuration

```bash
# Group definitions
/etc/group

# Group passwords (shadow)
/etc/gshadow

# Example /etc/group line:
# docker:x:999:deploy,jane,bob
# name:password:GID:members
```

## Tips

- `usermod -aG` (with `-a`) appends groups. Without `-a`, it replaces all supplementary groups -- a very common destructive mistake.
- `gpasswd -a` is the cleanest way to add a user to a single group without worrying about replacing other memberships.
- Group changes do not take effect for currently running sessions. The user must log out and back in (or use `newgrp`).
- `newgrp` opens a new shell with the changed primary group. It does not modify the existing session.
- A user's primary group cannot be deleted. Change it first with `usermod -g`.
- Files in `/etc/group` and `/etc/gshadow` should be edited with `vigr` and `vigr -s` respectively, not directly.

## References

- [man groupadd(8)](https://man7.org/linux/man-pages/man8/groupadd.8.html)
- [man groupmod(8)](https://man7.org/linux/man-pages/man8/groupmod.8.html)
- [man groupdel(8)](https://man7.org/linux/man-pages/man8/groupdel.8.html)
- [man groups(1)](https://man7.org/linux/man-pages/man1/groups.1.html)
- [man group(5) — /etc/group](https://man7.org/linux/man-pages/man5/group.5.html)
- [man gshadow(5) — /etc/gshadow](https://man7.org/linux/man-pages/man5/gshadow.5.html)
- [man gpasswd(1)](https://man7.org/linux/man-pages/man1/gpasswd.1.html)
- [Arch Wiki — Users and Groups](https://wiki.archlinux.org/title/Users_and_groups)
- [Red Hat — Managing User Groups](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-users-and-groups_configuring-basic-system-settings)
- [Ubuntu — User Management](https://help.ubuntu.com/community/AddUsersHowto)
