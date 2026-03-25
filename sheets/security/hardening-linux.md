# Linux Hardening (Comprehensive CIS-Aligned System Hardening Checklist)

Full Linux hardening reference covering CIS Benchmark essentials, SSH lockdown,
kernel tuning, filesystem permissions, audit logging, PAM, and network defense.

---

## 1. CIS Benchmark Essentials

### Audit Current State

```bash
# Install CIS-CAT or use Lynis for a quick audit
# Lynis — open-source security auditing tool
apt install lynis        # Debian/Ubuntu
yum install lynis        # RHEL/CentOS

lynis audit system
# Review report at /var/log/lynis-report.dat

# Quick hardening score
lynis audit system --quick | grep "Hardening index"
```

### Filesystem Configuration

```bash
# Ensure /tmp is a separate partition with noexec,nosuid,nodev
# /etc/fstab entry:
# tmpfs  /tmp  tmpfs  defaults,rw,nosuid,nodev,noexec,relatime  0  0

# Verify mount options
mount | grep /tmp
# Expected: tmpfs on /tmp type tmpfs (rw,nosuid,nodev,noexec,relatime)

# Apply same restrictions to /var/tmp
# Bind mount /var/tmp to /tmp or add fstab entry:
# tmpfs  /var/tmp  tmpfs  defaults,rw,nosuid,nodev,noexec,relatime  0  0

# Secure /dev/shm
# /etc/fstab:
# tmpfs  /dev/shm  tmpfs  defaults,rw,nosuid,nodev,noexec  0  0

# Disable automounting
systemctl disable autofs 2>/dev/null
systemctl mask autofs 2>/dev/null
```

### Disable Unused Filesystems

```bash
# Create /etc/modprobe.d/hardening.conf
cat > /etc/modprobe.d/hardening.conf << 'EOF'
# Disable unused filesystems
install cramfs /bin/true
install freevxfs /bin/true
install jffs2 /bin/true
install hfs /bin/true
install hfsplus /bin/true
install squashfs /bin/true
install udf /bin/true

# Disable unused network protocols
install dccp /bin/true
install sctp /bin/true
install rds /bin/true
install tipc /bin/true
EOF
```

---

## 2. SSH Hardening

```bash
# /etc/ssh/sshd_config — hardened configuration

# --- Authentication ---
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
PermitEmptyPasswords no
ChallengeResponseAuthentication no
UsePAM yes
AuthenticationMethods publickey

# --- Session ---
MaxAuthTries 3
MaxSessions 3
LoginGraceTime 30
ClientAliveInterval 300
ClientAliveCountMax 2

# --- Crypto ---
# Strong key exchange, ciphers, and MACs only
KexAlgorithms sntrup761x25519-sha512@openssh.com,curve25519-sha256,curve25519-sha256@libssh.org
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com
MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com
HostKeyAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256

# --- Restrictions ---
AllowAgentForwarding no
AllowTcpForwarding no
X11Forwarding no
PermitTunnel no
GatewayPorts no
PermitUserEnvironment no

# --- Logging ---
LogLevel VERBOSE
SyslogFacility AUTH

# --- Access control ---
# AllowUsers deploy admin
# AllowGroups sshusers
# DenyUsers root guest

# --- Banner ---
Banner /etc/ssh/banner
PrintMotd no
PrintLastLog yes
```

```bash
# Validate and reload
sshd -t && systemctl reload sshd

# Generate strong host keys (if needed)
rm /etc/ssh/ssh_host_*
ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ""
ssh-keygen -t rsa -b 4096 -f /etc/ssh/ssh_host_rsa_key -N ""

# Set proper permissions
chmod 600 /etc/ssh/ssh_host_*_key
chmod 644 /etc/ssh/ssh_host_*_key.pub
```

---

## 3. Kernel Parameters (sysctl)

```bash
# /etc/sysctl.d/99-hardening.conf

# --- Network hardening ---
# Disable IP forwarding (unless router/gateway)
net.ipv4.ip_forward = 0
net.ipv6.conf.all.forwarding = 0

# Disable source routing
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.conf.default.accept_source_route = 0
net.ipv6.conf.all.accept_source_route = 0

# Disable ICMP redirects
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv6.conf.all.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0

# Enable reverse path filtering (anti-spoofing)
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# Log martian packets
net.ipv4.conf.all.log_martians = 1
net.ipv4.conf.default.log_martians = 1

# Ignore ICMP broadcast requests
net.ipv4.icmp_echo_ignore_broadcasts = 1

# Ignore bogus ICMP error responses
net.ipv4.icmp_ignore_bogus_error_responses = 1

# SYN flood protection
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_syn_backlog = 2048
net.ipv4.tcp_synack_retries = 2

# Disable IPv6 if not needed
# net.ipv6.conf.all.disable_ipv6 = 1
# net.ipv6.conf.default.disable_ipv6 = 1

# --- Kernel hardening ---
# Restrict dmesg to root
kernel.dmesg_restrict = 1

# Restrict kernel pointers
kernel.kptr_restrict = 2

# Restrict ptrace (prevents process snooping)
kernel.yama.ptrace_scope = 2

# Disable SysRq (magic keys)
kernel.sysrq = 0

# ASLR — full randomization
kernel.randomize_va_space = 2

# Restrict core dumps
fs.suid_dumpable = 0

# Restrict unprivileged BPF
kernel.unprivileged_bpf_disabled = 1
net.core.bpf_jit_harden = 2

# Restrict unprivileged user namespaces (if not needed)
# kernel.unprivileged_userns_clone = 0

# Restrict loading kernel modules after boot
# WARNING: only enable if you won't need to load modules at runtime
# kernel.modules_disabled = 1
```

```bash
# Apply settings
sysctl --system

# Verify a specific setting
sysctl net.ipv4.ip_forward
# net.ipv4.ip_forward = 0
```

---

## 4. Filesystem Permissions

```bash
# Critical file permissions
chmod 644 /etc/passwd
chmod 600 /etc/shadow
chmod 644 /etc/group
chmod 600 /etc/gshadow
chmod 600 /boot/grub/grub.cfg
chmod 644 /etc/ssh/sshd_config
chmod 600 /etc/crontab
chmod 700 /etc/cron.d
chmod 700 /etc/cron.daily
chmod 700 /etc/cron.hourly
chmod 700 /etc/cron.weekly
chmod 700 /etc/cron.monthly

# Find world-writable files
find / -xdev -type f -perm -0002 -not -path "/proc/*" 2>/dev/null

# Find world-writable directories without sticky bit
find / -xdev -type d \( -perm -0002 -a ! -perm -1000 \) 2>/dev/null

# Find unowned files
find / -xdev -nouser -o -nogroup 2>/dev/null

# Find SUID/SGID binaries and audit against baseline
find / -perm /6000 -type f 2>/dev/null | sort > /tmp/suid_current.txt
# Compare with known-good list
# diff /tmp/suid_current.txt /baseline/suid_known_good.txt

# Remove unnecessary SUID bits
# chmod u-s /path/to/unnecessary/suid/binary

# Set sticky bit on world-writable directories
chmod +t /tmp
chmod +t /var/tmp
```

---

## 5. Service Minimization

```bash
# List all enabled services
systemctl list-unit-files --state=enabled --type=service --no-pager

# Disable unnecessary services
systemctl disable --now avahi-daemon.service   # mDNS/DNS-SD
systemctl disable --now cups.service           # Printing
systemctl disable --now bluetooth.service      # Bluetooth
systemctl disable --now ModemManager.service   # Modem
systemctl disable --now rpcbind.service        # RPC (NFS)
systemctl disable --now nfs-server.service     # NFS server
systemctl disable --now vsftpd.service         # FTP
systemctl disable --now telnet.socket          # Telnet
systemctl disable --now xinetd.service         # inetd services

# Mask services to prevent re-enabling
systemctl mask rpcbind.service
systemctl mask avahi-daemon.service

# Check for listening services (audit attack surface)
ss -tlnp | column -t

# Remove unnecessary packages
# Debian/Ubuntu
apt purge telnetd nis rsh-server rsh-client talk \
  xinetd tftp-hpa atftpd

# RHEL/CentOS
yum remove telnet-server rsh-server tftp-server \
  xinetd ypserv
```

---

## 6. Audit Logging (auditd)

```bash
# Install auditd
apt install auditd audispd-plugins   # Debian/Ubuntu
yum install audit audit-libs         # RHEL/CentOS

systemctl enable --now auditd

# /etc/audit/auditd.conf — key settings
# max_log_file = 50
# max_log_file_action = keep_logs
# space_left_action = email
# admin_space_left_action = halt
# disk_full_action = halt
# disk_error_action = halt
```

```bash
# /etc/audit/rules.d/hardening.rules

# --- Identity and authentication ---
-w /etc/passwd -p wa -k identity
-w /etc/shadow -p wa -k identity
-w /etc/group -p wa -k identity
-w /etc/gshadow -p wa -k identity
-w /etc/security/opasswd -p wa -k identity

# --- Login/logout monitoring ---
-w /var/log/faillog -p wa -k logins
-w /var/log/lastlog -p wa -k logins
-w /var/log/tallylog -p wa -k logins
-w /var/run/utmp -p wa -k session
-w /var/log/wtmp -p wa -k session
-w /var/log/btmp -p wa -k session

# --- Sudo and privilege escalation ---
-w /etc/sudoers -p wa -k sudoers
-w /etc/sudoers.d/ -p wa -k sudoers

# --- SSH configuration ---
-w /etc/ssh/sshd_config -p wa -k sshd_config
-w /etc/ssh/ -p wa -k ssh_config

# --- Cron ---
-w /etc/crontab -p wa -k cron
-w /etc/cron.d/ -p wa -k cron
-w /etc/cron.daily/ -p wa -k cron
-w /etc/cron.hourly/ -p wa -k cron
-w /etc/cron.monthly/ -p wa -k cron
-w /etc/cron.weekly/ -p wa -k cron
-w /var/spool/cron/ -p wa -k cron

# --- System calls ---
# Unauthorized file access attempts
-a always,exit -F arch=b64 -S open,openat,creat -F exit=-EACCES -k access
-a always,exit -F arch=b64 -S open,openat,creat -F exit=-EPERM -k access

# Process execution
-a always,exit -F arch=b64 -S execve -k exec

# Module loading
-a always,exit -F arch=b64 -S init_module,finit_module -k modules
-a always,exit -F arch=b64 -S delete_module -k modules
-w /sbin/insmod -p x -k modules
-w /sbin/rmmod -p x -k modules
-w /sbin/modprobe -p x -k modules

# Network configuration changes
-a always,exit -F arch=b64 -S sethostname,setdomainname -k network
-w /etc/hosts -p wa -k network
-w /etc/network/ -p wa -k network
-w /etc/sysconfig/network -p wa -k network

# Time changes
-a always,exit -F arch=b64 -S adjtimex,settimeofday -k time
-a always,exit -F arch=b64 -S clock_settime -k time
-w /etc/localtime -p wa -k time

# Make rules immutable (must be last rule — requires reboot to change)
-e 2
```

```bash
# Load rules
augenrules --load
# or
auditctl -R /etc/audit/rules.d/hardening.rules

# Verify rules loaded
auditctl -l | wc -l

# Query audit events
ausearch -k identity --start today
aureport --auth --start today
aureport --login --start today --summary
```

---

## 7. PAM Configuration

```bash
# /etc/pam.d/common-auth (Debian) or /etc/pam.d/system-auth (RHEL)

# Account lockout after 5 failed attempts (pam_faillock)
# /etc/pam.d/common-auth:
auth    required    pam_faillock.so preauth silent audit deny=5 unlock_time=900
auth    [default=die] pam_faillock.so authfail audit deny=5 unlock_time=900
auth    sufficient  pam_faillock.so authsucc audit deny=5 unlock_time=900

# /etc/security/faillock.conf:
# deny = 5
# unlock_time = 900
# fail_interval = 900
# audit
# silent

# Check locked accounts
faillock --user <username>

# Unlock account
faillock --user <username> --reset
```

---

## 8. Password Policies

```bash
# /etc/login.defs
PASS_MAX_DAYS   90
PASS_MIN_DAYS   7
PASS_MIN_LEN    14
PASS_WARN_AGE   14

# Apply to existing users
chage -M 90 -m 7 -W 14 <username>

# View user password policy
chage -l <username>

# /etc/security/pwquality.conf
# minlen = 14
# dcredit = -1     # at least 1 digit
# ucredit = -1     # at least 1 uppercase
# lcredit = -1     # at least 1 lowercase
# ocredit = -1     # at least 1 special character
# minclass = 4     # require all 4 character classes
# maxrepeat = 3    # max 3 consecutive identical characters
# maxclassrepeat = 4
# reject_username
# enforce_for_root
# dictcheck = 1

# Restrict su to wheel group
# /etc/pam.d/su:
# auth required pam_wheel.so use_uid
usermod -aG wheel <admin_user>
```

---

## 9. Cron Restrictions

```bash
# Restrict cron to authorized users only
# /etc/cron.allow — only users listed here can use cron
echo "root" > /etc/cron.allow
echo "deploy" >> /etc/cron.allow

# Remove cron.deny (cron.allow takes precedence)
rm -f /etc/cron.deny

# Same for at
echo "root" > /etc/at.allow
rm -f /etc/at.deny

# Set permissions on cron directories
chown root:root /etc/crontab
chmod 600 /etc/crontab
chmod 700 /etc/cron.d /etc/cron.daily /etc/cron.hourly \
  /etc/cron.weekly /etc/cron.monthly
```

---

## 10. GRUB Password Protection

```bash
# Generate password hash
grub-mkpasswd-pbkdf2
# Enter password, note the hash output

# /etc/grub.d/40_custom — add superuser
cat >> /etc/grub.d/40_custom << 'EOF'
set superusers="grubadmin"
password_pbkdf2 grubadmin grub.pbkdf2.sha512.10000.<HASH_HERE>
EOF

# Update GRUB
update-grub       # Debian/Ubuntu
grub2-mkconfig -o /boot/grub2/grub.cfg   # RHEL/CentOS

# Protect GRUB config
chmod 600 /boot/grub/grub.cfg
chown root:root /boot/grub/grub.cfg
```

---

## 11. Disable USB Storage

```bash
# Method 1: Blacklist kernel module
echo "install usb-storage /bin/true" >> /etc/modprobe.d/hardening.conf
echo "blacklist usb-storage" >> /etc/modprobe.d/hardening.conf

# Method 2: Remove the module if loaded
rmmod usb-storage 2>/dev/null

# Method 3: USBGuard (fine-grained USB device control)
apt install usbguard   # Debian/Ubuntu
yum install usbguard   # RHEL/CentOS

# Generate initial policy (allow currently connected devices)
usbguard generate-policy > /etc/usbguard/rules.conf

# Start the service
systemctl enable --now usbguard

# List connected USB devices
usbguard list-devices

# Block a specific device
usbguard block-device <device-id>
```

---

## 12. Automatic Security Updates

```bash
# --- Debian/Ubuntu ---
apt install unattended-upgrades
dpkg-reconfigure -plow unattended-upgrades

# /etc/apt/apt.conf.d/50unattended-upgrades:
# Unattended-Upgrade::Allowed-Origins {
#     "${distro_id}:${distro_codename}-security";
# };
# Unattended-Upgrade::Automatic-Reboot "false";
# Unattended-Upgrade::Mail "admin@example.com";
# Unattended-Upgrade::MailReport "on-change";

# Verify
unattended-upgrade --dry-run --debug

# --- RHEL/CentOS ---
yum install dnf-automatic

# /etc/dnf/automatic.conf:
# [commands]
# upgrade_type = security
# apply_updates = yes
# [emitters]
# emit_via = email
# [email]
# email_to = admin@example.com

systemctl enable --now dnf-automatic.timer
```

---

## 13. Network Hardening

```bash
# --- Firewall (nftables / iptables) ---

# Default deny policy
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT     # or DROP for strict environments

# Allow established connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow loopback
iptables -A INPUT -i lo -j ACCEPT

# Allow SSH from specific subnet
iptables -A INPUT -s 10.0.0.0/24 -p tcp --dport 22 -j ACCEPT

# Allow required services (example: HTTP/HTTPS)
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Log and drop everything else
iptables -A INPUT -j LOG --log-prefix "IPTABLES-DROP: " --log-level 4
iptables -A INPUT -j DROP

# Save rules
iptables-save > /etc/iptables/rules.v4
ip6tables-save > /etc/iptables/rules.v6

# --- TCP wrappers (if applicable) ---
# /etc/hosts.allow
# sshd: 10.0.0.0/24

# /etc/hosts.deny
# ALL: ALL

# --- Disable unused network interfaces ---
ip link set <unused_interface> down
```

---

## 14. File Integrity Monitoring (AIDE / OSSEC)

### AIDE

```bash
# Install
apt install aide     # Debian/Ubuntu
yum install aide     # RHEL/CentOS

# Initialize database
aide --init
# Move to active
mv /var/lib/aide/aide.db.new /var/lib/aide/aide.db

# Run integrity check
aide --check

# Update database after legitimate changes
aide --update
mv /var/lib/aide/aide.db.new /var/lib/aide/aide.db

# /etc/aide/aide.conf — key directories to monitor
# /boot     Full
# /bin      Full
# /sbin     Full
# /lib      Full
# /usr      Full
# /etc      Full
# /root     Full

# Schedule daily check
# /etc/cron.daily/aide-check:
#!/bin/bash
/usr/bin/aide --check | /usr/bin/mail -s "AIDE report $(hostname)" admin@example.com
```

### OSSEC / Wazuh

```bash
# Wazuh agent installation (connects to Wazuh manager)
curl -s https://packages.wazuh.com/key/GPG-KEY-WAZUH | gpg --dearmor \
  -o /usr/share/keyrings/wazuh.gpg
echo "deb [signed-by=/usr/share/keyrings/wazuh.gpg] https://packages.wazuh.com/4.x/apt/ stable main" \
  > /etc/apt/sources.list.d/wazuh.list
apt update && apt install wazuh-agent

# Configure manager address
sed -i 's/MANAGER_IP/<manager_ip>/' /var/ossec/etc/ossec.conf

# Enable syscheck (file integrity monitoring)
# In /var/ossec/etc/ossec.conf:
# <syscheck>
#   <frequency>7200</frequency>
#   <directories check_all="yes">/etc,/usr/bin,/usr/sbin</directories>
#   <directories check_all="yes">/bin,/sbin,/boot</directories>
# </syscheck>

systemctl enable --now wazuh-agent
```

---

## 15. Additional Hardening

### Disable Core Dumps

```bash
# /etc/security/limits.conf
# *    hard    core    0

# /etc/sysctl.d/99-hardening.conf (already covered above)
# fs.suid_dumpable = 0

# systemd: /etc/systemd/coredump.conf
# [Coredump]
# Storage=none
# ProcessSizeMax=0
```

### Restrict Compilers

```bash
# Remove compilers from production systems if not needed
apt purge gcc g++ make   # Debian/Ubuntu
yum remove gcc gcc-c++ make   # RHEL/CentOS

# Or restrict access to compilers
chmod 700 /usr/bin/gcc /usr/bin/g++ /usr/bin/make 2>/dev/null
```

### Umask

```bash
# Set restrictive default umask
# /etc/profile and /etc/bashrc:
umask 027

# For system services: /etc/login.defs
UMASK 027
```

### Banner Warning

```bash
# /etc/issue and /etc/issue.net — legal warning banner
cat > /etc/issue << 'EOF'
*************************************************************
WARNING: This system is for authorized use only.
All activity is monitored and logged. Unauthorized access
will be prosecuted to the fullest extent of the law.
*************************************************************
EOF
cp /etc/issue /etc/issue.net
cp /etc/issue /etc/ssh/banner
```

---

## Tips

- Apply hardening incrementally and test each change; never apply everything
  at once on production.
- Maintain a hardening baseline and use configuration management (Ansible,
  Puppet, Chef) to enforce it.
- Run CIS-CAT or Lynis regularly to audit compliance.
- Document every exception to the hardening baseline with a risk acceptance.
- Harden the boot process: set BIOS/UEFI password, disable boot from
  removable media, enable Secure Boot.
- Use SELinux or AppArmor in enforcing mode for mandatory access control.
- Separate log storage: send logs to a remote syslog server to prevent
  tampering.
- Review open ports weekly: `ss -tlnp` should match your expected services.
- Consider using systemd sandboxing features (ProtectSystem, PrivateTmp,
  NoNewPrivileges) for all custom services.
- For containers: apply the same kernel and network hardening on the host;
  containers share the kernel.

---

## References

- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks)
- [NIST SP 800-123 — Guide to General Server Security](https://csrc.nist.gov/publications/detail/sp/800-123/final)
- [NIST SP 800-53 — Security and Privacy Controls](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
- [SANS Linux Security Checklist](https://www.sans.org/score/checklists/linux)
- [OpenSSH Security Best Practices](https://www.openssh.com/security.html)
- [Lynis — Security Auditing Tool](https://cisofy.com/lynis/)
- [AIDE Documentation](https://aide.github.io/)
- [Wazuh Documentation](https://documentation.wazuh.com/)
- [Linux Audit Documentation](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/8/html/security_hardening/)
- [USBGuard Documentation](https://usbguard.github.io/)
