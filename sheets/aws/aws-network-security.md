# AWS Network & Application Security (SAA-C03 Task 1.2)

> Design secure workloads and applications: security groups vs NACLs, subnet strategy, edge protection (WAF/Shield/Network Firewall), the threat-detection suite (GuardDuty, Macie, Inspector, Detective, Security Hub), Cognito, Secrets Manager, and secure external connectivity.

## VPC Security Layers

```
# Traffic reaches an instance through, in order:
#   Route table → Network ACL (subnet edge) → Security Group (ENI)
#
# Security Group (SG)                 Network ACL (NACL)
# ----------------------------------  ---------------------------------
# Instance/ENI level                  Subnet level
# STATEFUL (return traffic auto)      STATELESS (must allow both ways)
# Allow rules only                    Allow AND Deny rules
# All rules evaluated                 Rules evaluated by number, first
#                                       match wins (end: implicit deny *)
# Can reference other SGs             CIDR ranges only
# Default SG: all outbound, inbound   Default NACL: allow everything
#   only from same SG                 Custom NACL: deny everything
#
# "Block a specific IP address" → NACL Deny rule (SGs cannot deny)
# "App tier accepts only from web tier" → SG referencing web tier's SG
# Ephemeral ports: stateless NACLs need outbound (or inbound for
#   return) 1024–65535 allowed, or connections break — classic trap
```

```bash
# SG referencing another SG (tier isolation)
aws ec2 authorize-security-group-ingress --group-id sg-app \
  --protocol tcp --port 8080 --source-group sg-web
# NACL: deny one bad actor, allow the rest
aws ec2 create-network-acl-entry --network-acl-id acl-0abc \
  --rule-number 50 --protocol -1 --cidr-block 198.51.100.7/32 \
  --rule-action deny --ingress
```

## Subnet Segmentation Strategy

```
# Standard 3-tier VPC:
# Public subnets   — route 0.0.0.0/0 → Internet Gateway; only ALB/NLB,
#                    NAT gateways, bastion (if any) live here
# Private subnets  — route 0.0.0.0/0 → NAT gateway; app tier
# Isolated/data    — no internet route at all; databases; reach AWS
#                    APIs via VPC endpoints only
#
# "Public subnet" is defined ONLY by its route table pointing at an IGW
# Instances also need a public IP / EIP to be reachable inbound
# NAT gateway: outbound-only internet for private subnets; managed,
#   per-AZ (deploy one per AZ; cross-AZ NAT = cost + AZ coupling)
# NAT instance: legacy answer, only correct when "lowest cost" AND
#   tiny traffic, or "needs to be a bastion too"
#
# Admin access without opening port 22 (preferred exam answer):
#   SSM Session Manager — no bastion, no inbound rules, IAM-controlled,
#   fully logged to CloudWatch/S3; requires SSM agent + instance role
#   (+ VPC endpoints for ssm, ssmmessages, ec2messages if no internet)
# EC2 Instance Connect — short-lived SSH key push via API (still port 22)
# Bastion host — only when SSM is ruled out; harden in public subnet
```

## Edge Protection: WAF, Shield, Firewall Manager

```
# AWS WAF — L7 web ACLs on: CloudFront, ALB, API Gateway, AppSync,
#   Cognito. NOT on NLB (L4 — no WAF possible).
#   Rules: SQLi/XSS managed rule groups, IP sets, geo match, rate-based
#   rules (throttle per-IP, e.g. block > 500 req/5 min), regex, size,
#   header/body match, CAPTCHA/challenge actions
#   "Protect against SQL injection / XSS / scraping / L7 floods" → WAF
#
# AWS Shield Standard — free, automatic, every AWS customer:
#   L3/L4 DDoS protection (SYN/UDP floods, reflection)
# AWS Shield Advanced — paid (3k USD/mo, 1-yr commitment):
#   enhanced detection, 24/7 Shield Response Team (SRT), cost protection
#   (refunds scaling charges from attack), WAF included at no charge for
#   protected resources; protects EC2/EIP, ELB, CloudFront, Global
#   Accelerator, Route 53
#   "Business-critical app needs DDoS response team + cost insurance"
#     → Shield Advanced
#
# AWS Network Firewall — managed stateful L3–L7 firewall for the VPC
#   itself: Suricata-compatible rules, domain filtering, IDS/IPS.
#   Deployed via firewall endpoints + route table steering (often in a
#   central inspection VPC behind Transit Gateway).
#   "Inspect/filter ALL traffic entering or leaving VPCs, on-prem-style
#    firewall rules, egress domain allow-listing" → Network Firewall
#
# Firewall Manager — org-wide policy manager: rolls out WAF ACLs,
#   Shield Advanced, SG baselines, Network Firewall, Route 53 DNS
#   Firewall rules to every account/resource automatically.
#   "Ensure every new ALB in every account gets the corporate WAF rules"
#     → Firewall Manager (requires Organizations)
#
# Route 53 Resolver DNS Firewall — block/allow DNS domains at the VPC
#   resolver ("prevent DNS exfiltration", "block malware domains")
```

## Threat Detection Suite (Pick-the-Service Questions)

```
# GuardDuty  — THREAT DETECTION. ML on CloudTrail, VPC Flow Logs, DNS
#   logs (+ optional S3 data events, EKS audit, RDS login, Lambda,
#   Malware Protection for EBS). Findings: crypto-mining, C2 beacons,
#   credential exfil, unusual API calls, port scans.
#   Trigger words: "intelligent threat detection", "compromised
#   instance", "anomalous behavior", "no agents/infrastructure"
#
# Macie      — DATA CLASSIFICATION. ML discovery of PII/PHI/secrets
#   in S3 ONLY. "Find sensitive data / PII in S3 buckets" → Macie
#
# Inspector  — VULNERABILITY SCANNING. CVEs + network reachability on
#   EC2 (via SSM agent), ECR container images, Lambda functions.
#   "Scan for software vulnerabilities / unintended network exposure"
#
# Detective  — INVESTIGATION. Graph analysis linking GuardDuty findings,
#   CloudTrail, VPC Flow Logs for root-cause. "Investigate/visualize
#   the scope of a security finding" → Detective
#
# Security Hub — AGGREGATION + POSTURE. Collects findings from all of
#   the above + Config; runs CIS/AWS Foundational/PCI standards checks;
#   single pane across accounts. "Central dashboard of security
#   findings org-wide" → Security Hub
#
# Trusted Advisor (security checks) — account-level best-practice
#   checks (open SGs, MFA on root, public S3) — cs aws-monitoring-governance
#
# Response automation: GuardDuty/Security Hub finding → EventBridge
#   rule → Lambda/Step Functions (isolate instance: swap SG, snapshot,
#   terminate) — the standard "automatically respond" pattern
```

## Cognito (Customer Identity)

```
# User Pools — the user DIRECTORY + authentication:
#   sign-up/sign-in, MFA, password policies, hosted UI, social logins
#   (Google/Facebook/Apple) and SAML/OIDC IdPs; issues JWTs
#   (id/access/refresh). ALB and API Gateway can authorize directly
#   against a user pool.
# Identity Pools (federated identities) — exchange a token (from user
#   pool or external IdP) for TEMPORARY AWS CREDENTIALS (via STS) to
#   call AWS services directly; supports unauthenticated guest roles.
# Memory hook: User pool = who are you (authentication, JWT);
#              Identity pool = what can you touch in AWS (credentials)
# Exam triggers: "millions of app users sign in", "social login",
#   "mobile app uploads directly to S3" (identity pool creds),
#   "add authentication to ALB/API Gateway with least code"
# Workforce SSO instead? → IAM Identity Center (cs aws-iam)
```

## Secrets Manager vs SSM Parameter Store

```
# Secrets Manager
# - Automatic ROTATION built-in (managed rotation for RDS/Aurora,
#   Redshift, DocumentDB; custom rotation via Lambda)
# - Cross-account access via resource policy; cross-Region replication
# - Always encrypted with KMS; ~0.40 USD/secret/month + API calls
# - Trigger: "rotate database credentials automatically" → ALWAYS this
#
# SSM Parameter Store
# - Free standard tier (10 KB advanced tier paid); SecureString uses KMS
# - No native rotation; hierarchical paths (/app/prod/db-url);
#   integrates with almost every AWS service
# - Trigger: "store configuration values / license keys at no
#   additional cost" → Parameter Store
#
# Never the answer: hardcoding, env vars in plaintext, S3 objects
# Retrieval in code: SDK call at startup + cache (or Lambda extension)
```

```bash
aws secretsmanager create-secret --name prod/db --secret-string '{"u":"app","p":"..."}'
aws secretsmanager rotate-secret --secret-id prod/db \
  --rotation-lambda-arn arn:aws:lambda:...:function:rotate \
  --rotation-rules AutomaticallyAfterDays=30
aws ssm put-parameter --name /app/prod/api-url --type SecureString \
  --value https://api.example.com --key-id alias/app
aws ssm get-parameter --name /app/prod/api-url --with-decryption
```

## Secure External Connectivity

```
# Site-to-Site VPN — IPsec over internet; two tunnels (two AWS
#   endpoints) per connection; ~1.25 Gbps ceiling per tunnel; minutes
#   to set up; encrypted by definition
# Client VPN — managed OpenVPN endpoint for roaming users into VPC
# Direct Connect (DX) — private dedicated link (1/10/100/400 Gbps);
#   NOT encrypted by itself. Options for encryption over DX:
#     - MACsec (10G+ dedicated links, L2 encryption)
#     - Site-to-Site VPN over a DX public VIF (IPsec on top) —
#       "consistent bandwidth AND encryption" exam answer
# DX + VPN failover: DX primary, VPN backup = standard hybrid HA answer
# Details, VIF types, resiliency models → cs aws-networking
#
# VPC endpoints kill internet exposure for AWS API traffic:
#   Gateway endpoints (S3, DynamoDB — free, route-table entry)
#   Interface endpoints / PrivateLink (ENI + private DNS, most services)
#   Endpoint policies restrict what the endpoint can reach;
#   bucket policies with aws:SourceVpce pin buckets to your endpoint
```

## Threat Vectors (Know the Countermeasure)

```
# DDoS L3/4 (SYN/UDP flood)  → Shield (Standard auto; Advanced for SRT),
#                               CloudFront/Route 53 absorb at edge,
#                               autoscaling as shock absorber
# DDoS L7 (HTTP flood)       → WAF rate-based rules + CloudFront
# SQL injection / XSS        → WAF managed rules + parameterized queries
# Credential theft via SSRF  → IMDSv2 required (cs aws-iam)
# DNS exfiltration           → Route 53 Resolver DNS Firewall
# Malware on EBS             → GuardDuty Malware Protection
# Secrets in code            → Secrets Manager + rotation
# Port scan / recon          → GuardDuty finding; SGs default-deny
# Data exfil from S3         → Block Public Access, VPC endpoints +
#                               aws:SourceVpce conditions, Macie to find
#                               what's exposed, versioning+MFA-Delete
```

## Exam Traps

```
# SG cannot DENY; "block one IP" → NACL (or WAF IP set at L7)
# NACL stateless: forgot ephemeral-port return rule = broken app
# WAF does not attach to NLB or EC2 directly — CloudFront/ALB/API GW
# GuardDuty vs Inspector vs Macie vs Detective — match by object:
#   threats=GuardDuty, CVEs=Inspector, PII-in-S3=Macie,
#   investigation=Detective, aggregate=Security Hub
# Shield Standard is already on — questions asking "enable basic DDoS
#   protection" often need NO action / it's free
# "Encrypt Direct Connect" — DX is not encrypted; add VPN over it or
#   MACsec
# Cognito user pool vs identity pool — JWT for your app vs AWS creds
# Secrets Manager vs Parameter Store — rotation vs free
# Session Manager beats bastion hosts in every modern answer: no
#   inbound ports, IAM auth, full audit logging
# Central egress filtering by DOMAIN NAME → Network Firewall (or a
#   forward proxy), not SGs/NACLs (they don't understand domains)
# "Apply WAF rules to all accounts automatically" → Firewall Manager
```

## See Also

aws-saa-c03, aws-iam, aws-data-protection, aws-networking, aws-high-availability, aws-monitoring-governance, vpc, cloud-security, waf, ids-ips, zero-trust, dos-ddos-attacks, sql-injection

## References

- [Security groups vs network ACLs](https://docs.aws.amazon.com/vpc/latest/userguide/infrastructure-security.html)
- [AWS WAF Developer Guide](https://docs.aws.amazon.com/waf/latest/developerguide/waf-chapter.html)
- [AWS Shield Advanced](https://docs.aws.amazon.com/waf/latest/developerguide/ddos-advanced-summary.html)
- [AWS Network Firewall](https://docs.aws.amazon.com/network-firewall/latest/developerguide/what-is-aws-network-firewall.html)
- [Amazon GuardDuty User Guide](https://docs.aws.amazon.com/guardduty/latest/ug/what-is-guardduty.html)
- [Amazon Macie User Guide](https://docs.aws.amazon.com/macie/latest/user/what-is-macie.html)
- [Amazon Cognito Developer Guide](https://docs.aws.amazon.com/cognito/latest/developerguide/what-is-amazon-cognito.html)
- [AWS Secrets Manager User Guide](https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html)
- [Session Manager](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager.html)
