# CIS Benchmarks (Center for Internet Security Hardening Standards)

Consensus-based security configuration guidelines providing prescriptive hardening recommendations for operating systems, cloud platforms, containers, and applications with scoring and audit automation.

## Benchmark Structure

```bash
# CIS Benchmark hierarchy:
# Profile Level 1 - Essential security, minimal performance impact
# Profile Level 2 - Defense in depth, may reduce functionality
#
# Recommendation types:
# Scored    - Compliance failure/pass affects benchmark score
# Unscored  - Best practice advisory, no score impact
#
# Each recommendation contains:
# - Description and rationale
# - Audit procedure (manual check commands)
# - Remediation steps
# - Impact assessment
# - Default value
# - CIS Controls v8 mapping
# - References (NIST, PCI-DSS, etc.)

# Download benchmarks (requires free CIS account)
# https://www.cisecurity.org/cis-benchmarks
# Available for: Linux, Windows, macOS, Docker, Kubernetes,
# AWS, Azure, GCP, Oracle DB, PostgreSQL, Apache, Nginx, etc.
```

## CIS-CAT Assessment Tool

```bash
# CIS-CAT Pro (commercial) or CIS-CAT Lite (free, limited)
# Download from https://www.cisecurity.org/cis-cat-pro

# Run assessment (Linux)
cd /opt/cis-cat
./Assessor-CLI.sh \
  -b benchmarks/CIS_Ubuntu_Linux_22.04_LTS_Benchmark_v1.0.0-xccdf.xml \
  -p "Level 1 - Server" \
  -r /var/reports/cis/ \
  -html -csv

# Automated scan with specific profile
./Assessor-CLI.sh \
  -b benchmarks/CIS_Docker_Benchmark_v1.6.0-xccdf.xml \
  -p "Level 1 - Docker" \
  -r /var/reports/cis/ \
  -nts  # no timestamp in filename

# Remote assessment via SSH
./Assessor-CLI.sh \
  -b benchmarks/CIS_Ubuntu_Linux_22.04_LTS_Benchmark_v1.0.0-xccdf.xml \
  -p "Level 1 - Server" \
  --sessions session-config.xml \
  -r /var/reports/cis/

# session-config.xml for remote scanning
cat << 'EOF' > session-config.xml
<sessions>
  <session id="ubuntu-web1">
    <type>ssh</type>
    <host>web1.internal</host>
    <port>22</port>
    <user>cisauditor</user>
    <identity>/opt/cis-cat/keys/audit_key</identity>
    <tmp>/tmp/cis-cat</tmp>
  </session>
</sessions>
EOF
```

## Linux Hardening (CIS Ubuntu/RHEL)

```bash
# 1.1.1 - Disable unused filesystems
cat << 'EOF' > /etc/modprobe.d/cis-filesystems.conf
install cramfs /bin/true
install freevxfs /bin/true
install jffs2 /bin/true
install hfs /bin/true
install hfsplus /bin/true
install squashfs /bin/true
install udf /bin/true
EOF

# 1.3.1 - Ensure AIDE is installed (file integrity)
apt install aide aide-common
aideinit
cp /var/lib/aide/aide.db.new /var/lib/aide/aide.db
# Schedule daily checks
echo "0 5 * * * root /usr/bin/aide.wrapper --check" >> /etc/crontab

# 1.4.1 - Ensure bootloader password is set
grub-mkpasswd-pbkdf2  # generate hash
cat << 'EOF' >> /etc/grub.d/40_custom
set superusers="grubadmin"
password_pbkdf2 grubadmin <hash>
EOF
update-grub

# 3.4.1 - Ensure firewall is active
ufw enable
ufw default deny incoming
ufw default allow outgoing
ufw allow from 10.0.0.0/8 to any port 22 proto tcp

# 5.2 - SSH hardening
cat << 'EOF' > /etc/ssh/sshd_config.d/cis.conf
Protocol 2
LogLevel INFO
MaxAuthTries 4
PermitRootLogin no
PermitEmptyPasswords no
HostbasedAuthentication no
X11Forwarding no
MaxStartups 10:30:60
LoginGraceTime 60
ClientAliveInterval 300
ClientAliveCountMax 3
AllowUsers sshusers
Banner /etc/issue.net
EOF
systemctl restart sshd

# 5.4 - Password policy
cat << 'EOF' > /etc/security/pwquality.conf
minlen = 14
dcredit = -1
ucredit = -1
ocredit = -1
lcredit = -1
EOF

# 5.4.1.1 - Password expiration
sed -i 's/^PASS_MAX_DAYS.*/PASS_MAX_DAYS 365/' /etc/login.defs
sed -i 's/^PASS_MIN_DAYS.*/PASS_MIN_DAYS 1/' /etc/login.defs
sed -i 's/^PASS_WARN_AGE.*/PASS_WARN_AGE 7/' /etc/login.defs

# 6.1.2 - Audit /etc/passwd permissions
stat -c '%a %U %G' /etc/passwd  # Should be 644 root root
chmod 644 /etc/passwd
chown root:root /etc/passwd

# 6.2.1 - Ensure no duplicate UIDs
awk -F: '{print $3}' /etc/passwd | sort -n | uniq -d
```

## Docker Hardening (CIS Docker Benchmark)

```bash
# 2.1 - Restrict network traffic between containers
dockerd --icc=false

# 2.2 - Set logging level
dockerd --log-level=info

# 2.5 - Enable Content Trust
export DOCKER_CONTENT_TRUST=1

# 4.1 - Ensure images are scanned
docker scan myimage:latest
# Or use Trivy
trivy image myimage:latest

# 4.5 - Ensure Content Trust is enabled for builds
DOCKER_CONTENT_TRUST=1 docker build -t myapp:v1 .

# 5.2 - Do not use host networking
# Bad: docker run --net=host myapp
# Good: docker run --net=bridge myapp

# 5.4 - Restrict Linux capabilities
docker run --cap-drop ALL --cap-add NET_BIND_SERVICE myapp

# 5.10 - Limit memory
docker run --memory=512m --memory-swap=512m myapp

# 5.12 - Mount root filesystem as read-only
docker run --read-only --tmpfs /tmp myapp

# 5.25 - Restrict container from gaining new privileges
docker run --security-opt=no-new-privileges myapp

# 5.28 - Use PIDs cgroup limit
docker run --pids-limit=100 myapp

# Audit script (docker-bench-security)
git clone https://github.com/docker/docker-bench-security.git
cd docker-bench-security
sudo sh docker-bench-security.sh
```

## Kubernetes Hardening (CIS K8s Benchmark)

```bash
# 1.1.1 - API server anonymous auth disabled
# /etc/kubernetes/manifests/kube-apiserver.yaml
spec:
  containers:
  - command:
    - kube-apiserver
    - --anonymous-auth=false
    - --audit-log-path=/var/log/kubernetes/audit.log
    - --audit-log-maxage=30
    - --authorization-mode=Node,RBAC
    - --enable-admission-plugins=NodeRestriction,PodSecurityAdmission

# 4.2.1 - Kubelet authentication (/var/lib/kubelet/config.yaml)
# authentication.anonymous.enabled: false
# authentication.webhook.enabled: true
# authorization.mode: Webhook
# readOnlyPort: 0

# 5.2.1 - Pod Security Standards
apiVersion: v1
kind: Namespace
metadata:
  name: production
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted

# 5.7.1 - Network policies (deny all by default)
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: production
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress

# kube-bench (automated CIS K8s scanning)
docker run --pid=host --net=host --userns=host \
  -v /etc:/etc:ro -v /var:/var:ro \
  -v /usr/bin/kubectl:/usr/bin/kubectl:ro \
  aquasec/kube-bench:latest run --targets=master
```

## Cloud Hardening (CIS AWS/Azure/GCP)

```bash
# AWS - CIS AWS Foundations Benchmark

# 1.4 - Ensure no root account access key
aws iam get-account-summary | grep AccountAccessKeysPresent

# 1.5 - Ensure MFA on root
aws iam get-account-summary | grep AccountMFAEnabled

# 3.1 - CloudTrail enabled in all regions
aws cloudtrail describe-trails --query 'trailList[*].IsMultiRegionTrail'

# Prowler (open-source AWS/Azure/GCP CIS scanner)
pip install prowler
prowler aws --compliance cis_2.0_aws
prowler aws --compliance cis_2.0_aws -M csv json html

# ScoutSuite (multi-cloud auditing)
pip install scoutsuite
scout aws --report-dir /var/reports/scout
```

## Hardening Automation

```bash
# Ansible - CIS hardening role
ansible-galaxy install dev-sec.os-hardening
ansible-galaxy install dev-sec.ssh-hardening

# playbook.yml
- hosts: all
  roles:
    - dev-sec.os-hardening
    - dev-sec.ssh-hardening
  vars:
    os_auth_pw_max_age: 365
    os_auth_pw_min_age: 1
    ssh_permit_root_login: "no"
    ssh_max_auth_retries: 4
    ssh_client_alive_interval: 300
    sysctl_overwrite:
      net.ipv4.ip_forward: 0
      net.ipv4.conf.all.send_redirects: 0
      net.ipv4.conf.all.accept_redirects: 0

ansible-playbook -i inventory playbook.yml --check  # dry run
ansible-playbook -i inventory playbook.yml

# InSpec compliance profile (audit)
inspec exec https://github.com/dev-sec/linux-baseline \
  -t ssh://user@target --reporter cli json:/var/reports/inspec.json
```

## CIS Controls v8 Mapping

```bash
# CIS Controls v8 - 18 control families
# Implementation Groups (IG):
# IG1 - Essential Cyber Hygiene (56 safeguards)
# IG2 - IG1 + 74 additional safeguards (130 total)
# IG3 - IG2 + 23 additional safeguards (153 total)

# Controls mapped to compliance frameworks:
# CIS Control 1 (Inventory)   -> NIST CM-8, PCI-DSS 2.4, SOC2 CC6.1
# CIS Control 4 (Secure Config) -> NIST CM-6, PCI-DSS 2.2
# CIS Control 5 (Accounts)    -> NIST AC-2/IA-5, PCI-DSS 8.1-8.3
# CIS Control 8 (Audit Logs)  -> NIST AU-2, PCI-DSS 10.1-10.7
# CIS Control 13 (Network Mon) -> NIST SI-4, PCI-DSS 11.4
```

## Tips

- Start with Level 1 profile; it provides strong baseline security with minimal operational disruption
- Use CIS-CAT or equivalent automated scanning; manual audits are error-prone on hundreds of settings
- Treat benchmark exceptions as risk acceptance decisions requiring documented justification and sign-off
- Run hardening in check/dry-run mode first; some settings (like disabling IPv6) can break applications
- Automate hardening via Ansible/Chef/Puppet and enforce via CI/CD pipeline gates
- Map CIS Controls to your compliance requirements (PCI-DSS, SOC 2, HIPAA) to satisfy multiple audits with one hardening effort
- Re-scan after every system update or configuration change; patches can reset hardened settings
- Use the Implementation Group (IG) model from CIS Controls v8 to prioritize by organizational maturity
- Establish benchmark exceptions for development and staging environments separately from production
- Version control your hardening playbooks and scan results for audit trail
- Monitor for configuration drift between scans using FIM or osquery scheduled queries

## See Also

- osquery for continuous CIS compliance monitoring
- SIEM for centralized compliance event logging
- MITRE ATT&CK for mapping hardening controls to attack techniques
- Docker Bench Security for container-specific auditing
- Prowler/ScoutSuite for cloud CIS scanning

## References

- [CIS Benchmarks Download](https://www.cisecurity.org/cis-benchmarks)
- [CIS Controls v8](https://www.cisecurity.org/controls/v8)
- [CIS-CAT Pro Assessor](https://www.cisecurity.org/cybersecurity-tools/cis-cat-pro)
- [dev-sec Hardening Framework](https://dev-sec.io/)
- [Docker Bench Security](https://github.com/docker/docker-bench-security)
- [kube-bench](https://github.com/aquasecurity/kube-bench)
- [Prowler](https://github.com/prowler-cloud/prowler)
- [NIST SP 800-53 Control Mappings](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
