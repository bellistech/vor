# Cloud Security — Deep Dive

Theoretical foundations of cloud security architecture, threat modeling, IAM policy evaluation, encryption models, and multi-cloud security strategy.

## Shared Responsibility Boundary Analysis

### Where the Boundary Actually Lives

The shared responsibility model is often presented as a clean table, but real-world boundaries are more nuanced.

**IaaS boundary details (AWS EC2 example):**

```
Provider responsibility ends at:
  - Hypervisor layer (Nitro/Xen)
  - Physical network to the instance ENI
  - Hardware RNG entropy source
  - EBS volume block storage (encryption optional but key management is shared)

Customer responsibility starts at:
  - Guest OS installation and patching
  - OS-level firewall (iptables/nftables inside the instance)
  - Application deployment and configuration
  - Data encryption at application layer
  - IAM credentials management
  - Security group and NACL rules

Gray areas (truly shared):
  - Network encryption: provider gives you VPC, TLS endpoints;
    customer must configure and enforce HTTPS
  - Patching: provider patches hypervisor transparently;
    customer patches guest OS (or uses managed services that auto-patch)
  - Logging: provider provides CloudTrail infrastructure;
    customer must enable it, retain logs, and monitor them
```

**PaaS boundary shift (AWS RDS example):**
- Provider now handles: OS patching, database engine updates, replication, backups
- Customer still handles: database schema, query optimization, access control (IAM + DB users), data encryption key management
- The boundary has moved up, but the customer is not absolved of security — they must still configure encryption, access controls, and network isolation

**SaaS boundary shift (Microsoft 365 example):**
- Provider handles: application code, infrastructure, availability, patching
- Customer handles: user provisioning, access policies, data classification, DLP configuration, conditional access
- Common mistake: assuming SaaS = fully managed security. SaaS misconfigurations (oversharing, disabled MFA) are among the most common cloud breaches.

### Responsibility Gaps

The most dangerous security failures occur in responsibility gaps — areas where both parties assume the other is responsible.

Common gaps:
- **Backup verification:** Provider offers backup tools; customer assumes backups are tested. Neither verifies restore works.
- **Key rotation:** Provider offers KMS with rotation; customer never enables it and assumes default is sufficient.
- **Network segmentation:** Provider offers VPC/subnets; customer puts everything in one subnet with one security group.
- **Log monitoring:** Provider delivers logs to S3; customer never builds alerting on those logs.

## Cloud Threat Model (STRIDE for Cloud)

STRIDE threat modeling adapted for cloud environments:

### Spoofing

**Cloud-specific spoofing threats:**
- Compromised IAM credentials (access keys, service account keys leaked in code repositories)
- Metadata service exploitation: SSRF to `169.254.169.254` steals instance role credentials
- Cross-account role assumption with overly permissive trust policies
- Federated identity token forgery (SAML, OIDC)

**Mitigations:**
- IMDSv2 (token-based metadata service) blocks SSRF-based credential theft
- External ID requirement on cross-account role trust policies
- Short-lived credentials via STS AssumeRole instead of long-lived access keys
- Conditional access policies requiring device compliance and MFA

### Tampering

**Cloud-specific tampering threats:**
- Infrastructure-as-code modification (Terraform state file tampering)
- S3 object modification without versioning enabled
- CloudTrail log deletion or trail disabling
- Security group rule modification to open access
- DNS record hijacking (Route 53, Cloud DNS)

**Mitigations:**
- S3 Object Lock (WORM compliance) for immutable log storage
- CloudTrail log file validation (hash chain integrity)
- SCPs (Service Control Policies) preventing trail deletion
- Resource-level IAM policies with MFA requirements for destructive actions
- Infrastructure drift detection via CSPM or Terraform plan

### Repudiation

**Cloud-specific repudiation threats:**
- API calls from shared credentials (multiple users sharing one access key)
- Actions from assumed roles that obscure the original principal
- Console actions without CloudTrail data events enabled
- Cross-account access where the source account is not logged

**Mitigations:**
- Individual IAM users/roles per person (never share credentials)
- CloudTrail enabled in all regions with data events for sensitive resources
- AWS CloudTrail Lake or equivalent for long-term queryable audit logs
- Require session tags on role assumption to identify the original actor

### Information Disclosure

**Cloud-specific disclosure threats:**
- Public S3 buckets, Azure Blob containers, or GCS buckets
- Overly permissive IAM policies granting read access to sensitive data
- Unencrypted data at rest (EBS volumes, RDS snapshots, S3 objects)
- Snapshot sharing with unintended accounts
- Secrets in Lambda environment variables, container environment, or IaC templates

**Mitigations:**
- S3 Block Public Access at account level
- AWS Macie / GCP DLP for automated data classification and discovery
- CMK encryption with key policies restricting access
- Snapshot encryption and explicit share permissions
- Secrets Manager / Parameter Store with IAM-based access control

### Denial of Service

**Cloud-specific DoS threats:**
- Resource exhaustion (launching thousands of instances to consume quota and budget)
- API throttling attacks (CloudFormation, Lambda invoke rate)
- Denial of wallet (triggering unlimited auto-scaling)
- Cross-region replication storms (one write triggers cascading replications)

**Mitigations:**
- Service quotas and billing alerts
- Lambda reserved concurrency and account-level concurrency limits
- AWS Shield (Standard included, Advanced for DDoS protection)
- WAF rate limiting rules
- Budget actions that terminate resources when thresholds are exceeded

### Elevation of Privilege

**Cloud-specific privilege escalation:**
- IAM privilege escalation: user can attach policies to themselves or create new access keys
- Lambda function with overprivileged execution role can access resources beyond its scope
- Container escape from pod to node (Kubernetes)
- SSRF from web application to metadata service to obtain instance role credentials
- Cross-account role chains: A assumes B, B assumes C (transitive trust)

**Mitigations:**
- IAM Access Analyzer to identify overly permissive policies
- Permission boundaries to cap maximum privileges for delegated admin
- Pod Security Standards (restricted) in Kubernetes
- VPC endpoints for metadata service (IMDSv2 + hop limit = 1)
- Regular IAM credential reports and unused permission cleanup

## CSPM vs CWPP vs CASB Architecture

### Architectural Comparison

```
CSPM (Cloud Security Posture Management)
  Scope: Configuration plane
  Data source: Cloud provider APIs (describe-*, get-*, list-*)
  Analysis: Compare actual config against desired state / benchmarks
  Output: Misconfiguration findings with remediation guidance
  Deployment: Agentless (API-only)
  Example finding: "S3 bucket 'prod-data' has public read access enabled"

CWPP (Cloud Workload Protection Platform)
  Scope: Compute plane (what runs inside VMs/containers/serverless)
  Data source: Agent telemetry, API-based snapshot scanning, eBPF
  Analysis: Vulnerability scanning, runtime threat detection, file integrity
  Output: CVE findings, behavioral alerts, compliance posture
  Deployment: Agent-based or agentless (snapshot scanning)
  Example finding: "Container running as root with CVE-2024-XXXX in libssl"

CASB (Cloud Access Security Broker)
  Scope: SaaS application plane
  Data source: Inline traffic (proxy) or SaaS APIs (out-of-band)
  Analysis: Shadow IT discovery, DLP, access anomaly detection
  Output: Policy violations, data exposure, compromised account alerts
  Deployment: Forward proxy, reverse proxy, or API connector
  Example finding: "User uploaded 500 files to personal OneDrive in 10 minutes"
```

### Convergence into CNAPP

The market is converging CSPM, CWPP, and CIEM (Cloud Infrastructure Entitlement Management) into CNAPP (Cloud-Native Application Protection Platform):

```
CNAPP = CSPM + CWPP + CIEM + Pipeline Security

Build Time:
  - IaC scanning (Terraform, CloudFormation, ARM templates)
  - Container image scanning in CI/CD
  - Secret detection in code repositories
  - Software composition analysis (SCA)

Deploy Time:
  - Admission control (block misconfigured deployments)
  - Image signing and verification
  - Runtime configuration validation

Run Time:
  - CSPM continuous monitoring
  - CWPP runtime protection
  - CIEM least-privilege enforcement
  - Network security monitoring
```

## Cloud IAM Policy Evaluation

### AWS IAM Policy Evaluation Logic

AWS evaluates policies in a specific order. Understanding this logic is essential for debugging access issues and preventing privilege escalation.

```
Request arrives
  |
  v
Step 1: Gather all applicable policies
  - Identity-based policies (attached to user/role)
  - Resource-based policies (on the target resource)
  - Permission boundaries (cap on identity policies)
  - Session policies (from AssumeRole or federated session)
  - Service Control Policies (from AWS Organizations)
  - VPC Endpoint policies
  |
  v
Step 2: Evaluate SCPs (if in an Organization)
  - SCPs are evaluated first and act as a ceiling
  - If SCP denies, access is denied regardless of other policies
  - SCPs do not grant permissions, only limit them
  |
  v
Step 3: Evaluate resource-based policies
  - If resource-based policy allows AND there is no explicit deny elsewhere,
    access is granted (even without identity-based allow)
  - This is the one exception to "deny by default"
  |
  v
Step 4: Evaluate permission boundaries
  - If a permission boundary is set, it acts as a ceiling
  - Access requires BOTH identity policy allow AND boundary allow
  |
  v
Step 5: Evaluate session policies
  - Further restricts the session beyond the role's policies
  |
  v
Step 6: Evaluate identity-based policies
  - User/group/role policies evaluated
  - Explicit deny in ANY policy = denied (always wins)
  - Must have explicit allow (no allow = implicit deny)
  |
  v
Final decision matrix:
  Explicit Deny anywhere        --> DENY
  No applicable statement       --> DENY (implicit)
  Allow in resource-based only  --> ALLOW (cross-account exception)
  Allow in identity + boundary  --> ALLOW
  Allow in identity, no boundary--> ALLOW
  Allow in identity, boundary denies --> DENY
```

### Cross-Account Access Evaluation

Cross-account access adds complexity because policies in both the source and destination accounts must allow the action.

```
Account A (requester) --> Account B (resource owner)

For access to succeed:
  Account A: Identity policy must allow the action on Account B's resource
  Account B: Resource-based policy must allow Account A's principal

  Exception: If Account B's resource policy explicitly allows Account A's
  principal AND grants access, Account A does not need an identity policy
  (the resource policy alone is sufficient for same-partition access)

Role assumption (more common for cross-account):
  Account A: Identity policy allows sts:AssumeRole on Account B's role ARN
  Account B: Role trust policy allows Account A's principal to assume
  Account B: Role permission policy grants access to resources
```

### IAM Privilege Escalation Paths

Common IAM configurations that enable privilege escalation:

```
Path 1: iam:CreatePolicyVersion
  User can create a new version of an existing policy with Admin access
  and set it as the default version

Path 2: iam:AttachUserPolicy / iam:AttachRolePolicy
  User can attach AdministratorAccess to themselves

Path 3: iam:PutUserPolicy / iam:PutRolePolicy
  User can create inline policy with any permissions

Path 4: iam:CreateAccessKey
  If user can create access keys for other users, they can impersonate them

Path 5: iam:PassRole + lambda:CreateFunction + lambda:InvokeFunction
  User creates Lambda with privileged role, invokes it to perform actions

Path 6: iam:PassRole + ec2:RunInstances
  User launches EC2 with privileged instance profile, SSH in and use role

Path 7: sts:AssumeRole on wildcard resource (arn:aws:iam::*:role/*)
  User can assume any role in any account

Mitigation: Permission boundaries + SCPs + regular IAM Access Analyzer scans
```

## Encryption at Rest, in Transit, in Use

### Encryption at Rest

**Server-side encryption (SSE):**
- **SSE-S3 (AWS managed):** AWS manages both the data key and master key. Zero customer effort. Customer cannot audit key usage.
- **SSE-KMS (Customer master key in KMS):** AWS manages data keys, customer controls master key in KMS. Key usage logged in CloudTrail. Supports key rotation and key policies.
- **SSE-C (Customer-provided key):** Customer sends encryption key with every request. AWS never stores the key. Maximum control but operational complexity.

**Client-side encryption:**
- Application encrypts data before sending to cloud storage
- Only ciphertext exists in the cloud; provider cannot decrypt
- Customer manages encryption library, keys, and key rotation
- Required for zero-trust data protection or regulatory compliance

**Envelope encryption model:**
```
Master Key (in KMS, never leaves KMS)
  |
  | encrypt
  v
Data Key (generated per object/volume)
  |
  | encrypt
  v
Data (S3 object, EBS block, RDS tablespace)

Storage format:
  [ Encrypted Data Key | Encrypted Data ]

Decryption:
  1. Send encrypted data key to KMS
  2. KMS decrypts data key using master key (inside HSM)
  3. Use plaintext data key to decrypt data locally
  4. Discard plaintext data key from memory
```

### Encryption in Transit

- **TLS 1.2/1.3** for all API calls (enforced by cloud providers)
- **VPN (IPsec/WireGuard)** for site-to-cloud connections
- **PrivateLink / VPC Endpoints** to avoid public internet entirely
- **Certificate management:** ACM (AWS), App Service Certificates (Azure), Certificate Manager (GCP) for automated TLS certificate provisioning and rotation

**Enforcing encryption in transit:**
```json
// S3 bucket policy denying unencrypted transport
{
  "Statement": [{
    "Effect": "Deny",
    "Principal": "*",
    "Action": "s3:*",
    "Resource": "arn:aws:s3:::my-bucket/*",
    "Condition": {
      "Bool": { "aws:SecureTransport": "false" }
    }
  }]
}
```

### Encryption in Use (Confidential Computing)

- **Problem:** Data is vulnerable while being processed (in memory, in CPU registers)
- **Solution:** Hardware-based trusted execution environments (TEEs) that protect data during computation
- **Technologies:**
  - AWS Nitro Enclaves: isolated compute environment with attestation
  - Azure Confidential Computing: Intel SGX and AMD SEV-SNP based VMs
  - GCP Confidential VMs: AMD SEV encrypted memory
- **Threat model:** Protects against compromised hypervisor, malicious cloud operator, memory bus snooping
- **Limitations:** Performance overhead (5-30%), limited memory in enclaves, attestation complexity

## Cloud Security Architecture Frameworks

### AWS Well-Architected Security Pillar

Seven design principles:
1. **Implement a strong identity foundation:** Centralized identity, least privilege, separation of duties
2. **Enable traceability:** Logging, monitoring, alerting for all actions
3. **Apply security at all layers:** VPC, subnet, instance, application
4. **Automate security best practices:** IaC, automated remediation, policy-as-code
5. **Protect data in transit and at rest:** Encryption, tokenization, access control
6. **Keep people away from data:** Eliminate direct access; use automation and tools
7. **Prepare for security events:** Incident response playbooks, simulation exercises

### Zero Trust in Cloud

Zero trust eliminates the concept of a trusted network perimeter. Every request is verified regardless of source.

```
Traditional (perimeter-based):
  [Internet] --firewall--> [Trusted Network] --> [Resources]
  Once inside the perimeter, movement is unrestricted

Zero Trust:
  [Any Location] --> [Identity Verification] --> [Policy Engine] --> [Resource]
  Every access request is authenticated, authorized, and encrypted
  regardless of network location

Zero Trust pillars in cloud:
  1. Identity: strong authentication (MFA, passwordless, certificates)
  2. Device: device health assessment before granting access
  3. Network: micro-segmentation, no flat networks, east-west inspection
  4. Application: per-application access (not network-level VPN)
  5. Data: classification, encryption, DLP at every stage
  6. Visibility: continuous monitoring, analytics, threat detection
```

**Cloud-native zero trust implementation:**
- AWS: IAM + VPC + PrivateLink + Verified Access + GuardDuty
- Azure: Entra ID Conditional Access + Private Link + Microsoft Defender
- GCP: BeyondCorp Enterprise + VPC Service Controls + IAP (Identity-Aware Proxy)

## Multi-Cloud Security Strategy

### Challenges

- **Identity fragmentation:** Three separate IAM systems (AWS, Azure, GCP) with different policy languages, evaluation logic, and role models
- **Network boundary differences:** Security groups vs NSGs vs firewall rules have different semantics (stateful vs stateless, priority models)
- **Logging format differences:** CloudTrail vs Activity Log vs Audit Logs have different schemas, retention policies, and query interfaces
- **Encryption key management:** Each provider has its own KMS with different key hierarchy, rotation, and access control models
- **Compliance mapping:** Same compliance requirement maps to different controls in each provider

### Architecture Patterns

**Abstraction layer approach:**
- Use Terraform or Pulumi to abstract provider-specific resources behind a common IaC interface
- Use a CSPM tool that normalizes findings across providers (Prisma Cloud, Wiz, Orca)
- Use a centralized SIEM that ingests and normalizes logs from all providers
- Use a cloud-agnostic identity provider (Okta, Azure AD) as the single source of truth for identity

**Federated identity pattern:**
```
Central IdP (Okta, Azure AD)
  |
  +--SAML/OIDC--> AWS (IAM Identity Center)
  |                 |
  |                 +--> AWS Roles (mapped from IdP groups)
  |
  +--SAML/OIDC--> Azure (Entra ID)
  |                 |
  |                 +--> Azure Roles (mapped from IdP groups)
  |
  +--SAML/OIDC--> GCP (Workforce Identity Federation)
                    |
                    +--> GCP Roles (mapped from IdP groups)

Benefit: One place to manage users, groups, MFA, access reviews
Risk: Central IdP is a single point of compromise
Mitigation: Harden IdP with hardware MFA, privileged access workstations,
            break-glass accounts with offline MFA
```

### Cloud-Native vs Third-Party Security Tools

**When to use cloud-native:**
- Deep integration with provider's API and services
- No additional licensing cost (included or usage-based)
- Faster feature updates for new services
- Lower operational overhead

**When to use third-party:**
- Multi-cloud environments requiring unified visibility
- Capabilities exceeding native tools (advanced analytics, ML)
- Compliance requirements mandating independence from provider
- Organizational preference for best-of-breed over integrated

**Recommendation:** Use cloud-native tools as the foundation in each provider, add third-party tools for cross-cloud correlation, advanced analytics, and unified dashboards. Do not try to standardize on one provider's native tools across all clouds.

## See Also

- endpoint-security, security-operations

## References

- [AWS IAM Policy Evaluation Logic](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_evaluation-logic.html)
- [AWS Well-Architected Security Pillar](https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/)
- [NIST SP 800-144 — Guidelines on Security and Privacy in Public Cloud Computing](https://csrc.nist.gov/publications/detail/sp/800-144/final)
- [NIST SP 800-210 — General Access Control Guidance for Cloud Systems](https://csrc.nist.gov/publications/detail/sp/800-210/final)
- [CSA Cloud Controls Matrix v4](https://cloudsecurityalliance.org/research/cloud-controls-matrix)
- [MITRE ATT&CK Cloud Matrix](https://attack.mitre.org/matrices/enterprise/cloud/)
- [BeyondCorp: A New Approach to Enterprise Security (Google)](https://research.google/pubs/pub43231/)
- [Azure Security Architecture](https://learn.microsoft.com/en-us/azure/architecture/framework/security/)
- [GCP Security Foundations Blueprint](https://cloud.google.com/architecture/security-foundations)
- [Rhino Security Labs — AWS IAM Privilege Escalation](https://rhinosecuritylabs.com/aws/aws-privilege-escalation-methods-mitigation/)
