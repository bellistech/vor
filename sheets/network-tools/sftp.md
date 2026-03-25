# sftp (SSH File Transfer Protocol)

Interactive and batch file transfer over SSH — secure replacement for FTP.

## Connecting

### Start interactive session
```bash
sftp user@host
sftp -P 2222 user@host                 # custom SSH port
sftp -i ~/.ssh/deploy_key user@host    # specific key
sftp -J bastion user@internal-host     # jump host
sftp myserver                          # use ~/.ssh/config Host entry
```

## Interactive Commands

### Navigation (remote)
```bash
sftp> pwd                              # print remote working directory
sftp> ls                               # list remote files
sftp> ls -la                           # detailed listing
sftp> cd /var/log                      # change remote directory
```

### Navigation (local)
```bash
sftp> lpwd                             # print local working directory
sftp> lls                              # list local files
sftp> lcd /tmp                         # change local directory
```

### Download files
```bash
sftp> get remote-file.txt              # download to local cwd
sftp> get remote-file.txt local-name.txt   # download with different name
sftp> get -r remote-dir/              # download directory recursively
sftp> mget *.log                       # download multiple files (glob)
sftp> get -a remote-file.txt          # resume partial download (append)
```

### Upload files
```bash
sftp> put local-file.txt               # upload to remote cwd
sftp> put local-file.txt /remote/path/ # upload to specific path
sftp> put -r local-dir/               # upload directory recursively
sftp> mput *.csv                       # upload multiple files (glob)
sftp> put -a local-file.txt           # resume partial upload (append)
```

### File operations (remote)
```bash
sftp> mkdir new-dir                    # create directory
sftp> rmdir empty-dir                  # remove empty directory
sftp> rm unwanted-file.txt             # delete file
sftp> rename old-name.txt new-name.txt # rename/move
sftp> chmod 644 file.txt               # change permissions
sftp> chown user file.txt              # change owner (if permitted)
sftp> chgrp group file.txt             # change group
sftp> ln -s target link-name           # create symlink
sftp> df -h                            # disk usage on remote
```

### Other
```bash
sftp> !command                         # run local shell command
sftp> !ls -la                          # local ls
sftp> progress                         # toggle progress display
sftp> exit                             # disconnect
sftp> bye                              # same as exit
sftp> help                             # list all commands
```

## Batch Mode

### Run commands from file
```bash
sftp -b commands.txt user@host

# commands.txt:
# cd /var/log
# mget *.log
# bye
```

### One-liner batch
```bash
echo 'get /remote/file.txt' | sftp user@host
echo -e 'cd /data\nmget *.csv' | sftp user@host
```

### Suppress interactive prompts
```bash
sftp -b - user@host <<'EOF'
cd /backups
put backup.tar.gz
bye
EOF
```

## Transfer Options

### Buffer and request tuning
```bash
sftp -B 262144 user@host               # buffer size (default 32768)
sftp -R 256 user@host                  # number of outstanding requests
# Larger values improve throughput on high-latency links
```

### Compression
```bash
sftp -C user@host                      # enable compression
```

### Preserve attributes
```bash
sftp> put -p file.txt                  # preserve mtime and permissions
sftp> get -p file.txt                  # preserve on download too
```

## Subsystem and Alternate Server

### Specify subsystem path
```bash
sftp -s /usr/lib/openssh/sftp-server user@host
```

### Use sftp with ProxyCommand
```bash
sftp -o ProxyCommand="ssh -W %h:%p bastion" user@internal
```

## Common Patterns

### Download all logs from today
```bash
sftp -b - user@host <<'EOF'
cd /var/log/app
mget *2024-01-15*
bye
EOF
```

### Upload release artifact
```bash
sftp -b - deploy@prod <<'EOF'
cd /opt/releases
put app-v2.3.tar.gz
chmod 644 app-v2.3.tar.gz
bye
EOF
```

### Mirror a directory down
```bash
sftp -b - user@host <<'EOF'
get -r /remote/project/
bye
EOF
```

## Tips

- `sftp` runs over SSH — it is NOT related to FTP/FTPS despite the similar name
- Use batch mode (`-b`) for scripting; interactive mode is for ad-hoc work
- `mget` and `mput` support globs but not recursive — use `get -r` / `put -r` for directories
- `-B` (buffer size) and `-R` (requests) can significantly improve throughput over high-latency links
- `sftp` does not support resuming transfers by default; use `-a` (append mode) for a workaround
- For automated/recurring transfers, `rsync` is usually a better choice (delta sync, excludes, dry-run)
- Some SFTP servers have restricted shells (chroot) — `cd` above the chroot root will fail silently
- `!command` executes on the local machine — useful for quick checks without leaving the session
- Tab completion works in interactive mode for both local and remote paths (OpenSSH implementation)

## References

- [man sftp](https://man7.org/linux/man-pages/man1/sftp.1.html)
- [man sftp-server](https://man7.org/linux/man-pages/man8/sftp-server.8.html)
- [man ssh_config](https://man7.org/linux/man-pages/man5/ssh_config.5.html)
- [man sshd_config — SFTP Subsystem Configuration](https://man7.org/linux/man-pages/man5/sshd_config.5.html)
- [OpenSSH Official Manual Pages](https://www.openssh.com/manual.html)
- [RFC 4253 — The Secure Shell (SSH) Transport Layer Protocol](https://www.rfc-editor.org/rfc/rfc4253)
- [Arch Wiki — SFTP Chroot](https://wiki.archlinux.org/title/SFTP_chroot)
