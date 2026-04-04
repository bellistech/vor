# FTP (File Transfer Protocol)

Client-server protocol for transferring files over TCP, using a control connection (port 21) for commands and a separate data connection for file transfers in either active or passive mode.

## Active vs Passive Mode

```
Active Mode (PORT):
  Client opens random port P, sends PORT command
  Server connects FROM port 20 TO client port P

  Client:P ----control (21)----> Server:21
  Client:P <----data (20)------- Server:20

  Problem: Client firewall blocks inbound from server

Passive Mode (PASV):
  Client sends PASV command
  Server opens random port Q, tells client
  Client connects TO server port Q

  Client:R ----control (21)----> Server:21
  Client:R ----data-----------> Server:Q

  Works through client-side NAT/firewall
  Default for most modern clients
```

## FTP vs FTPS vs SFTP

```
FTP   — Plain text, ports 20/21, no encryption
        Protocol: FTP (RFC 959)

FTPS  — FTP + TLS encryption, ports 990 (implicit) or 21 (explicit/STARTTLS)
        Protocol: FTP over TLS (RFC 4217)
        Two modes: Implicit (TLS from start) and Explicit (AUTH TLS upgrade)

SFTP  — SSH File Transfer Protocol, port 22
        Protocol: Completely different — runs over SSH, NOT FTP
        Single connection, no active/passive complexity
        Preferred in most modern environments
```

## Core FTP Commands

```bash
# FTP command-line client
ftp ftp.example.com

# Common session commands
ftp> open ftp.example.com        # Connect
ftp> user alice                  # Login
ftp> pass ****                   # Password
ftp> pwd                         # Print working directory
ftp> ls                          # List files (LIST)
ftp> cd /pub/data                # Change directory (CWD)
ftp> lcd /tmp                    # Change local directory
ftp> binary                      # Set binary transfer mode (TYPE I)
ftp> ascii                       # Set ASCII transfer mode (TYPE A)
ftp> get file.tar.gz             # Download (RETR)
ftp> put upload.txt              # Upload (STOR)
ftp> mget *.csv                  # Download multiple files
ftp> mput *.log                  # Upload multiple files
ftp> mkdir backups               # Create directory (MKD)
ftp> rmdir old                   # Remove directory (RMD)
ftp> delete file.txt             # Delete file (DELE)
ftp> rename old.txt new.txt      # Rename (RNFR/RNTO)
ftp> passive                     # Toggle passive mode
ftp> bye                         # Disconnect (QUIT)
```

## Raw FTP Protocol Commands

```
USER alice                    — Send username
PASS secret                   — Send password
SYST                          — System type
PWD                           — Print working directory
CWD /pub                      — Change directory
CDUP                          — Change to parent directory
LIST                          — List directory (over data connection)
NLST                          — Name list only
RETR file.txt                 — Retrieve (download) file
STOR file.txt                 — Store (upload) file
APPE file.txt                 — Append to file
DELE file.txt                 — Delete file
MKD dirname                   — Make directory
RMD dirname                   — Remove directory
RNFR old.txt                  — Rename from
RNTO new.txt                  — Rename to
TYPE I                        — Binary transfer mode
TYPE A                        — ASCII transfer mode
PASV                          — Enter passive mode
PORT h1,h2,h3,h4,p1,p2       — Active mode (IP,port)
SIZE file.txt                 — Get file size
MDTM file.txt                 — Get modification time
FEAT                          — List supported features
QUIT                          — Close session
```

## FTP Response Codes

```
1xx — Positive Preliminary (action started)
  125  Data connection already open, transfer starting
  150  File status okay, about to open data connection

2xx — Positive Completion
  200  Command okay
  211  System status
  213  File status (SIZE response)
  220  Service ready
  221  Closing control connection
  226  Transfer complete, closing data connection
  227  Entering Passive Mode (h1,h2,h3,h4,p1,p2)
  230  User logged in
  250  Requested file action okay
  257  "PATHNAME" created

3xx — Positive Intermediate
  331  Username okay, need password
  332  Need account for login
  350  Pending further action (RNFR accepted)

4xx — Transient Negative
  421  Service not available
  425  Cannot open data connection
  426  Connection closed, transfer aborted
  450  File unavailable (busy)
  451  Local error

5xx — Permanent Negative
  500  Syntax error
  501  Syntax error in parameters
  502  Command not implemented
  530  Not logged in
  550  File unavailable (not found / permission denied)
  553  File name not allowed
```

## lftp (Advanced FTP/SFTP Client)

```bash
# Connect to FTP
lftp ftp://alice@ftp.example.com

# Connect to FTPS (explicit)
lftp ftps://alice@ftp.example.com

# Connect to SFTP
lftp sftp://alice@server.example.com

# Mirror a remote directory (download)
lftp -e "mirror --parallel=4 /remote/dir /local/dir; quit" \
  ftp://alice:pass@ftp.example.com

# Mirror upload (reverse mirror)
lftp -e "mirror -R --parallel=4 /local/dir /remote/dir; quit" \
  ftp://alice:pass@ftp.example.com

# Download with resume support
lftp -e "pget -n 5 /path/to/largefile.iso; quit" \
  ftp://alice:pass@ftp.example.com

# Scripted batch operations
lftp -f /path/to/commands.lftp

# Bookmark management
lftp -e "bookmark add myserver ftp://alice@ftp.example.com"
lftp -e "open myserver"
```

## vsftpd Configuration

```bash
# /etc/vsftpd.conf — secure FTP server

# Access control
anonymous_enable=NO
local_enable=YES
write_enable=YES
local_umask=022
chroot_local_user=YES
allow_writeable_chroot=YES

# Passive mode (configure for firewall)
pasv_enable=YES
pasv_min_port=40000
pasv_max_port=40100
pasv_address=203.0.113.10

# Logging
xferlog_enable=YES
xferlog_std_format=YES
log_ftp_protocol=YES

# TLS / FTPS (explicit)
ssl_enable=YES
force_local_data_ssl=YES
force_local_logins_ssl=YES
ssl_tlsv1_2=YES
ssl_sslv2=NO
ssl_sslv3=NO
rsa_cert_file=/etc/ssl/certs/vsftpd.pem
rsa_private_key_file=/etc/ssl/private/vsftpd.key

# User restrictions
userlist_enable=YES
userlist_deny=NO
userlist_file=/etc/vsftpd.userlist

# Reload
sudo systemctl restart vsftpd
```

## Firewall Rules for FTP

```bash
# iptables — allow active + passive FTP
iptables -A INPUT -p tcp --dport 21 -j ACCEPT
iptables -A INPUT -p tcp --dport 40000:40100 -j ACCEPT

# Load connection tracking module for active FTP
modprobe nf_conntrack_ftp
iptables -A INPUT -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT

# nftables equivalent
nft add rule inet filter input tcp dport 21 accept
nft add rule inet filter input tcp dport 40000-40100 accept

# UFW
ufw allow 21/tcp
ufw allow 40000:40100/tcp
```

## curl for FTP

```bash
# List directory
curl ftp://ftp.example.com/pub/

# Download file
curl -o localfile.txt ftp://ftp.example.com/pub/file.txt

# Upload file
curl -T upload.txt ftp://alice:pass@ftp.example.com/uploads/

# Download with resume
curl -C - -o largefile.iso ftp://ftp.example.com/pub/largefile.iso

# FTPS (explicit)
curl --ftp-ssl ftp://alice:pass@ftp.example.com/

# Create directory
curl --ftp-create-dirs ftp://alice:pass@ftp.example.com/new/path/

# Verbose (see protocol exchange)
curl -v ftp://ftp.example.com/pub/
```

## Tips

- Always prefer SFTP over FTP/FTPS; SFTP uses one connection, no firewall issues, and SSH-level encryption
- Use passive mode by default; active mode breaks through NAT and most firewalls
- Set `pasv_min_port`/`pasv_max_port` in vsftpd and open those ports in the firewall
- Enable `chroot_local_user` in vsftpd to jail users to their home directories
- Use `lftp` for scripted transfers; its mirror command with `--parallel` is far faster than `mget`
- For anonymous FTP, set `anon_upload_enable=NO` and use a dedicated read-only directory
- FTP transfers have no integrity checking; verify checksums (MD5/SHA256) after large transfers
- Monitor vsftpd logs at `/var/log/vsftpd.log` for unauthorized access attempts
- Set `idle_session_timeout` and `data_connection_timeout` to prevent stale connections
- Use `TYPE I` (binary mode) for all non-text files to prevent line-ending corruption
- FTPS implicit mode (port 990) is deprecated; use explicit mode (AUTH TLS on port 21)
- Load `nf_conntrack_ftp` kernel module when using iptables with active mode FTP

## See Also

- sftp, scp, rsync, curl, ssh, vsftpd, tls

## References

- [RFC 959 — File Transfer Protocol](https://datatracker.ietf.org/doc/html/rfc959)
- [RFC 4217 — FTP Security (TLS)](https://datatracker.ietf.org/doc/html/rfc4217)
- [RFC 2428 — FTP Extensions for IPv6 and NATs (EPSV/EPRT)](https://datatracker.ietf.org/doc/html/rfc2428)
- [vsftpd Documentation](https://security.appspot.com/vsftpd.html)
- [lftp Manual](https://lftp.yar.ru/lftp-man.html)
