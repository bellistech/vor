# scp (Secure Copy)

Copy files between hosts over SSH — simple syntax for one-off transfers.

## Copy To Remote

### Single file
```bash
scp file.txt user@host:/remote/path/
scp file.txt user@host:~                       # copy to home directory
scp file.txt user@host:/tmp/newname.txt        # copy with different name
```

### Multiple files
```bash
scp file1.txt file2.txt user@host:/remote/path/
scp *.log user@host:/remote/logs/
```

### Directory (recursive)
```bash
scp -r /local/dir/ user@host:/remote/path/
```

## Copy From Remote

### Single file
```bash
scp user@host:/remote/file.txt /local/path/
scp user@host:~/file.txt .                     # to current directory
```

### Directory
```bash
scp -r user@host:/remote/dir/ /local/path/
```

## Copy Between Remote Hosts

### Remote to remote
```bash
scp user1@host1:/path/file.txt user2@host2:/path/
# Traffic goes through your local machine by default

scp -3 user1@host1:/path/file.txt user2@host2:/path/
# -3 routes through local machine explicitly (needed when hosts can't reach each other)
```

## Connection Options

### Custom SSH port
```bash
scp -P 2222 file.txt user@host:/path/          # uppercase -P (not -p)
```

### Identity file (SSH key)
```bash
scp -i ~/.ssh/deploy_key file.txt user@host:/path/
```

### SSH config host
```bash
scp file.txt myserver:/path/                   # uses Host myserver from ~/.ssh/config
```

### Jump host
```bash
scp -o ProxyJump=bastion file.txt user@internal:/path/
scp -J bastion file.txt user@internal:/path/   # shorthand (OpenSSH 7.3+)
```

## Transfer Options

### Compression
```bash
scp -C file.txt user@host:/path/               # enable compression
```

### Preserve attributes
```bash
scp -p file.txt user@host:/path/               # preserve mtime, atime, mode
```

### Bandwidth limit
```bash
scp -l 8000 large-file.iso user@host:/path/    # limit to 8000 Kbit/s (= 1 MB/s)
```

### Quiet mode
```bash
scp -q file.txt user@host:/path/               # suppress progress meter
```

### Verbose / debug
```bash
scp -v file.txt user@host:/path/               # verbose (debug SSH connection)
```

## Cipher Selection

### Use a faster cipher
```bash
scp -c aes128-gcm@openssh.com large-file.iso user@host:/path/
```

## IPv4 / IPv6

### Force protocol
```bash
scp -4 file.txt user@host:/path/               # IPv4 only
scp -6 file.txt user@host:/path/               # IPv6 only
```

## Common Patterns

### Deploy a build artifact
```bash
scp -C build/app.tar.gz deploy@prod:/opt/releases/
```

### Grab remote logs
```bash
scp user@host:/var/log/app/*.log /tmp/remote-logs/
```

### Copy SSH key to remote
```bash
scp ~/.ssh/id_ed25519.pub user@host:~/.ssh/authorized_keys
# Better: use ssh-copy-id instead
```

## Tips

- `scp` is deprecated in favor of `sftp` or `rsync` as of OpenSSH 9.0; it still works but new features go elsewhere
- `-P` (uppercase) sets the port — different from `ssh -p` (lowercase)
- `-p` (lowercase) preserves file times and modes — easy to confuse with `-P`
- `scp` does not support resuming interrupted transfers; use `rsync -avP` for large files
- `scp -r` does not follow symlinks in the copied directory tree by default
- For recurring syncs, `rsync` is better (delta transfers, exclude patterns, dry-run)
- Wildcards in remote paths are expanded by the remote shell: `scp user@host:~/logs/*.log .`
- `scp` copies files through your local machine even for remote-to-remote; use `-3` to be explicit about this
- Use `~/.ssh/config` to avoid repeating `-P`, `-i`, and usernames on every command
